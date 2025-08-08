// Package middleware provides HTTP middleware components for the Nephoran Intent Operator
package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// RedactLoggerConfig defines configuration for the redacting logger middleware
type RedactLoggerConfig struct {
	// Enabled controls whether logging is active
	Enabled bool

	// LogLevel sets the minimum log level (debug, info, warn, error)
	LogLevel slog.Level

	// SkipPaths lists paths to skip logging (e.g., health checks)
	SkipPaths []string

	// PathLogLevels maps specific paths to their log levels
	PathLogLevels map[string]slog.Level

	// SensitiveHeaders lists headers to redact (case-insensitive)
	SensitiveHeaders []string

	// SensitiveQueryParams lists query parameters to redact
	SensitiveQueryParams []string

	// SensitiveJSONFields lists JSON field names to redact in bodies
	SensitiveJSONFields []string

	// LogRequestBody controls whether to log request bodies
	LogRequestBody bool

	// LogResponseBody controls whether to log response bodies
	LogResponseBody bool

	// MaxBodySize limits the size of logged bodies (bytes)
	MaxBodySize int64

	// SlowRequestThreshold defines what constitutes a slow request
	SlowRequestThreshold time.Duration

	// IncludeClientIP controls whether to log client IP addresses
	IncludeClientIP bool

	// IncludeUserAgent controls whether to log user agent
	IncludeUserAgent bool

	// RedactedValue is the replacement for sensitive data
	RedactedValue string

	// CorrelationIDHeader specifies the header name for correlation ID
	CorrelationIDHeader string

	// GenerateCorrelationID controls whether to generate IDs if missing
	GenerateCorrelationID bool
}

// DefaultRedactLoggerConfig returns secure default configuration
func DefaultRedactLoggerConfig() *RedactLoggerConfig {
	return &RedactLoggerConfig{
		Enabled:      true,
		LogLevel:     slog.LevelInfo,
		SkipPaths:    []string{"/health", "/healthz", "/ready", "/readyz", "/metrics"},
		PathLogLevels: map[string]slog.Level{
			"/debug": slog.LevelDebug,
		},
		SensitiveHeaders: []string{
			"Authorization",
			"Cookie",
			"Set-Cookie",
			"X-API-Key",
			"X-Auth-Token",
			"X-Session-Token",
			"X-CSRF-Token",
			"Proxy-Authorization",
			"X-Amz-Security-Token",
		},
		SensitiveQueryParams: []string{
			"token",
			"api_key",
			"apikey",
			"secret",
			"password",
			"auth",
			"key",
			"access_token",
			"refresh_token",
			"client_secret",
		},
		SensitiveJSONFields: []string{
			"password",
			"secret",
			"token",
			"api_key",
			"apiKey",
			"private_key",
			"privateKey",
			"client_secret",
			"clientSecret",
			"access_token",
			"accessToken",
			"refresh_token",
			"refreshToken",
		},
		LogRequestBody:        false,
		LogResponseBody:       false,
		MaxBodySize:           4096, // 4KB default
		SlowRequestThreshold:  5 * time.Second,
		IncludeClientIP:       true,
		IncludeUserAgent:      true,
		RedactedValue:         "[REDACTED]",
		CorrelationIDHeader:   "X-Request-ID",
		GenerateCorrelationID: true,
	}
}

// RedactLogger implements redacting logger middleware
type RedactLogger struct {
	config *RedactLoggerConfig
	logger *slog.Logger

	// Compiled patterns for efficiency
	skipPathsMutex sync.RWMutex
	skipPatterns   []*regexp.Regexp

	// Case-insensitive header lookup map
	sensitiveHeadersMap map[string]bool

	// Case-insensitive query param lookup map  
	sensitiveParamsMap map[string]bool

	// JSON field patterns for redaction
	jsonFieldPatterns []*regexp.Regexp
}

// NewRedactLogger creates a new redacting logger middleware instance
func NewRedactLogger(config *RedactLoggerConfig, logger *slog.Logger) (*RedactLogger, error) {
	if config == nil {
		config = DefaultRedactLoggerConfig()
	}

	rl := &RedactLogger{
		config:              config,
		logger:              logger.With(slog.String("component", "redact-logger")),
		sensitiveHeadersMap: make(map[string]bool),
		sensitiveParamsMap:  make(map[string]bool),
	}

	// Compile skip path patterns
	for _, path := range config.SkipPaths {
		// Support wildcards
		pattern := strings.ReplaceAll(path, "*", ".*")
		pattern = "^" + pattern + "$"
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid skip path pattern %s: %w", path, err)
		}
		rl.skipPatterns = append(rl.skipPatterns, re)
	}

	// Build case-insensitive header map
	for _, header := range config.SensitiveHeaders {
		rl.sensitiveHeadersMap[strings.ToLower(header)] = true
	}

	// Build case-insensitive query param map
	for _, param := range config.SensitiveQueryParams {
		rl.sensitiveParamsMap[strings.ToLower(param)] = true
	}

	// Compile JSON field patterns for body redaction
	for _, field := range config.SensitiveJSONFields {
		// Match JSON field in various formats
		patterns := []string{
			fmt.Sprintf(`"%s"\s*:\s*"[^"]*"`, field),              // "field": "value"
			fmt.Sprintf(`"%s"\s*:\s*'[^']*'`, field),              // "field": 'value'
			fmt.Sprintf(`"%s"\s*:\s*[0-9]+`, field),               // "field": 123
			fmt.Sprintf(`"%s"\s*:\s*\[[^\]]*\]`, field),           // "field": [...]
			fmt.Sprintf(`"%s"\s*:\s*\{[^\}]*\}`, field),           // "field": {...}
		}
		for _, pattern := range patterns {
			re, err := regexp.Compile(pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid JSON field pattern for %s: %w", field, err)
			}
			rl.jsonFieldPatterns = append(rl.jsonFieldPatterns, re)
		}
	}

	return rl, nil
}

// responseWriter wraps http.ResponseWriter to capture status code and body
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
	written    bool
	logBody    bool
	maxSize    int64
}

func newResponseWriter(w http.ResponseWriter, logBody bool, maxSize int64) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           &bytes.Buffer{},
		logBody:        logBody,
		maxSize:        maxSize,
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}

	// Capture body for logging if enabled
	if rw.logBody && int64(rw.body.Len()) < rw.maxSize {
		remaining := rw.maxSize - int64(rw.body.Len())
		if int64(len(b)) <= remaining {
			rw.body.Write(b)
		} else {
			rw.body.Write(b[:remaining])
		}
	}

	return rw.ResponseWriter.Write(b)
}

// Middleware returns the logging middleware handler
func (rl *RedactLogger) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Check if path should be skipped
		if rl.shouldSkipPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Get or generate correlation ID
		correlationID := rl.getOrGenerateCorrelationID(r)
		
		// Add correlation ID to context
		ctx := context.WithValue(r.Context(), "correlation_id", correlationID)
		r = r.WithContext(ctx)

		// Set correlation ID in response header
		w.Header().Set(rl.config.CorrelationIDHeader, correlationID)

		// Determine log level for this path
		logLevel := rl.getLogLevelForPath(r.URL.Path)

		// Start timing
		start := time.Now()

		// Capture request body if needed
		var requestBody []byte
		if rl.config.LogRequestBody && r.Body != nil {
			bodyReader := io.LimitReader(r.Body, rl.config.MaxBodySize)
			requestBody, _ = io.ReadAll(bodyReader)
			r.Body = io.NopCloser(bytes.NewReader(requestBody))
		}

		// Wrap response writer
		wrapped := newResponseWriter(w, rl.config.LogResponseBody, rl.config.MaxBodySize)

		// Log request start
		if logLevel <= slog.LevelDebug {
			rl.logRequestStart(r, correlationID, requestBody)
		}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Calculate duration
		duration := time.Since(start)

		// Log request completion
		rl.logRequestComplete(r, wrapped, correlationID, duration, requestBody, logLevel)
	})
}

// shouldSkipPath checks if the path should be skipped from logging
func (rl *RedactLogger) shouldSkipPath(path string) bool {
	rl.skipPathsMutex.RLock()
	defer rl.skipPathsMutex.RUnlock()

	for _, pattern := range rl.skipPatterns {
		if pattern.MatchString(path) {
			return true
		}
	}
	return false
}

// getOrGenerateCorrelationID gets the correlation ID from request or generates one
func (rl *RedactLogger) getOrGenerateCorrelationID(r *http.Request) string {
	// Check for existing correlation ID
	correlationID := r.Header.Get(rl.config.CorrelationIDHeader)
	if correlationID != "" {
		return correlationID
	}

	// Check alternate headers
	alternateHeaders := []string{"X-Correlation-ID", "X-Trace-ID", "X-Request-ID"}
	for _, header := range alternateHeaders {
		if id := r.Header.Get(header); id != "" {
			return id
		}
	}

	// Generate new ID if configured
	if rl.config.GenerateCorrelationID {
		return uuid.New().String()
	}

	return ""
}

// getLogLevelForPath returns the appropriate log level for a path
func (rl *RedactLogger) getLogLevelForPath(path string) slog.Level {
	if level, ok := rl.config.PathLogLevels[path]; ok {
		return level
	}
	return rl.config.LogLevel
}

// logRequestStart logs the beginning of a request
func (rl *RedactLogger) logRequestStart(r *http.Request, correlationID string, body []byte) {
	attrs := []slog.Attr{
		slog.String("correlation_id", correlationID),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("query", rl.redactQueryParams(r.URL.Query())),
		slog.String("proto", r.Proto),
	}

	if rl.config.IncludeClientIP {
		attrs = append(attrs, slog.String("client_ip", rl.getClientIP(r)))
	}

	if rl.config.IncludeUserAgent {
		attrs = append(attrs, slog.String("user_agent", r.Header.Get("User-Agent")))
	}

	// Add redacted headers
	headers := rl.redactHeaders(r.Header)
	attrs = append(attrs, slog.Any("headers", headers))

	// Add redacted body if available
	if len(body) > 0 {
		redactedBody := rl.redactBody(body)
		attrs = append(attrs, slog.String("request_body", redactedBody))
	}

	rl.logger.LogAttrs(r.Context(), slog.LevelDebug, "Request started", attrs...)
}

// logRequestComplete logs the completion of a request
func (rl *RedactLogger) logRequestComplete(r *http.Request, w *responseWriter, correlationID string, duration time.Duration, requestBody []byte, logLevel slog.Level) {
	// Determine if this was a slow request
	isSlow := duration > rl.config.SlowRequestThreshold

	// Base attributes
	attrs := []slog.Attr{
		slog.String("correlation_id", correlationID),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.Int("status", w.statusCode),
		slog.Duration("duration", duration),
		slog.Float64("duration_ms", float64(duration.Nanoseconds())/1e6),
	}

	// Add client info if configured
	if rl.config.IncludeClientIP {
		attrs = append(attrs, slog.String("client_ip", rl.getClientIP(r)))
	}

	if rl.config.IncludeUserAgent {
		attrs = append(attrs, slog.String("user_agent", r.Header.Get("User-Agent")))
	}

	// Add query parameters (redacted)
	if r.URL.RawQuery != "" {
		attrs = append(attrs, slog.String("query", rl.redactQueryParams(r.URL.Query())))
	}

	// Add request body if configured and at debug level
	if rl.config.LogRequestBody && len(requestBody) > 0 && logLevel <= slog.LevelDebug {
		redactedBody := rl.redactBody(requestBody)
		attrs = append(attrs, slog.String("request_body", redactedBody))
	}

	// Add response body if configured
	if rl.config.LogResponseBody && w.body.Len() > 0 && logLevel <= slog.LevelDebug {
		redactedBody := rl.redactBody(w.body.Bytes())
		attrs = append(attrs, slog.String("response_body", redactedBody))
	}

	// Determine log level based on status code and speed
	msgLogLevel := logLevel
	message := "Request completed"

	if w.statusCode >= 500 {
		msgLogLevel = slog.LevelError
		message = "Request failed with server error"
	} else if w.statusCode >= 400 {
		msgLogLevel = slog.LevelWarn
		message = "Request failed with client error"
	} else if isSlow {
		msgLogLevel = slog.LevelWarn
		message = "Slow request completed"
		attrs = append(attrs, slog.Bool("slow_request", true))
	}

	rl.logger.LogAttrs(r.Context(), msgLogLevel, message, attrs...)
}

// redactHeaders redacts sensitive headers
func (rl *RedactLogger) redactHeaders(headers http.Header) map[string][]string {
	redacted := make(map[string][]string)
	
	for name, values := range headers {
		lowerName := strings.ToLower(name)
		if rl.sensitiveHeadersMap[lowerName] {
			// Redact the values
			redactedValues := make([]string, len(values))
			for i := range values {
				redactedValues[i] = rl.config.RedactedValue
			}
			redacted[name] = redactedValues
		} else {
			redacted[name] = values
		}
	}
	
	return redacted
}

// redactQueryParams redacts sensitive query parameters
func (rl *RedactLogger) redactQueryParams(params url.Values) string {
	redacted := make(url.Values)
	
	for key, values := range params {
		lowerKey := strings.ToLower(key)
		if rl.sensitiveParamsMap[lowerKey] {
			// Redact the values
			redactedValues := make([]string, len(values))
			for i := range values {
				redactedValues[i] = rl.config.RedactedValue
			}
			redacted[key] = redactedValues
		} else {
			redacted[key] = values
		}
	}
	
	return redacted.Encode()
}

// redactBody redacts sensitive fields in request/response bodies
func (rl *RedactLogger) redactBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}

	// Try to parse as JSON for structured redaction
	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err == nil {
		// It's valid JSON, perform structured redaction
		redacted := rl.redactJSONFields(jsonData)
		if redactedBytes, err := json.Marshal(redacted); err == nil {
			return string(redactedBytes)
		}
	}

	// Fall back to regex-based redaction for non-JSON or failed JSON
	bodyStr := string(body)
	for _, pattern := range rl.jsonFieldPatterns {
		bodyStr = pattern.ReplaceAllStringFunc(bodyStr, func(match string) string {
			// Preserve the field name but redact the value
			parts := strings.SplitN(match, ":", 2)
			if len(parts) == 2 {
				return parts[0] + `: "` + rl.config.RedactedValue + `"`
			}
			return rl.config.RedactedValue
		})
	}

	return bodyStr
}

// redactJSONFields recursively redacts sensitive fields in JSON data
func (rl *RedactLogger) redactJSONFields(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			lowerKey := strings.ToLower(key)
			shouldRedact := false
			
			// Check if this key should be redacted
			for _, sensitiveField := range rl.config.SensitiveJSONFields {
				if strings.ToLower(sensitiveField) == lowerKey {
					shouldRedact = true
					break
				}
			}
			
			if shouldRedact {
				result[key] = rl.config.RedactedValue
			} else {
				// Recursively redact nested structures
				result[key] = rl.redactJSONFields(value)
			}
		}
		return result
		
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = rl.redactJSONFields(item)
		}
		return result
		
	default:
		return data
	}
}

// getClientIP extracts the client IP address from the request
func (rl *RedactLogger) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
		return r.RemoteAddr[:idx]
	}
	
	return r.RemoteAddr
}

// UpdateConfig updates the configuration dynamically
func (rl *RedactLogger) UpdateConfig(config *RedactLoggerConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Rebuild sensitive maps
	sensitiveHeadersMap := make(map[string]bool)
	for _, header := range config.SensitiveHeaders {
		sensitiveHeadersMap[strings.ToLower(header)] = true
	}

	sensitiveParamsMap := make(map[string]bool)
	for _, param := range config.SensitiveQueryParams {
		sensitiveParamsMap[strings.ToLower(param)] = true
	}

	// Compile new skip patterns
	var skipPatterns []*regexp.Regexp
	for _, path := range config.SkipPaths {
		pattern := strings.ReplaceAll(path, "*", ".*")
		pattern = "^" + pattern + "$"
		re, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid skip path pattern %s: %w", path, err)
		}
		skipPatterns = append(skipPatterns, re)
	}

	// Update with lock
	rl.skipPathsMutex.Lock()
	rl.config = config
	rl.sensitiveHeadersMap = sensitiveHeadersMap
	rl.sensitiveParamsMap = sensitiveParamsMap
	rl.skipPatterns = skipPatterns
	rl.skipPathsMutex.Unlock()

	return nil
}

// GetStats returns statistics about the logger (can be extended)
func (rl *RedactLogger) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"enabled":               rl.config.Enabled,
		"log_level":            rl.config.LogLevel.String(),
		"skip_paths_count":     len(rl.config.SkipPaths),
		"sensitive_headers":    len(rl.config.SensitiveHeaders),
		"sensitive_params":     len(rl.config.SensitiveQueryParams),
		"sensitive_json_fields": len(rl.config.SensitiveJSONFields),
		"log_request_body":     rl.config.LogRequestBody,
		"log_response_body":    rl.config.LogResponseBody,
	}
}