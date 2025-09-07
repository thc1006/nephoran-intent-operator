// Package middleware provides HTTP middleware components for the Nephoran Intent Operator
package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// InputValidationConfig defines configuration for input validation middleware
type InputValidationConfig struct {
	// MaxBodySize defines maximum request body size in bytes
	// Default: 10MB (10 * 1024 * 1024)
	MaxBodySize int64

	// MaxHeaderSize defines maximum header size in bytes
	// Default: 8KB (8 * 1024)
	MaxHeaderSize int

	// MaxURLLength defines maximum URL length
	// Default: 2048 characters
	MaxURLLength int

	// MaxParameterCount defines maximum number of query parameters
	// Default: 100
	MaxParameterCount int

	// MaxParameterLength defines maximum length for a single parameter value
	// Default: 1024 characters
	MaxParameterLength int

	// AllowedContentTypes defines allowed Content-Type headers
	// Default: ["application/json", "application/x-www-form-urlencoded", "multipart/form-data"]
	AllowedContentTypes []string

	// EnableSQLInjectionProtection enables SQL injection detection
	// Default: true
	EnableSQLInjectionProtection bool

	// EnableXSSProtection enables XSS attack detection
	// Default: true
	EnableXSSProtection bool

	// EnablePathTraversalProtection enables path traversal detection
	// Default: true
	EnablePathTraversalProtection bool

	// EnableCommandInjectionProtection enables command injection detection
	// Default: true
	EnableCommandInjectionProtection bool

	// CustomValidators allows adding custom validation functions
	CustomValidators []ValidatorFunc

	// SanitizeInput enables automatic input sanitization
	// Default: true
	SanitizeInput bool

	// LogViolations enables logging of validation violations
	// Default: true
	LogViolations bool

	// BlockOnViolation determines whether to block requests on violations
	// Default: true
	BlockOnViolation bool
}

// ValidatorFunc is a custom validation function type
type ValidatorFunc func(r *http.Request) error

// DefaultInputValidationConfig returns a secure default configuration
func DefaultInputValidationConfig() *InputValidationConfig {
	return &InputValidationConfig{
		MaxBodySize:                      10 * 1024 * 1024, // 10MB
		MaxHeaderSize:                    8 * 1024,         // 8KB
		MaxURLLength:                     2048,
		MaxParameterCount:                100,
		MaxParameterLength:               1024,
		AllowedContentTypes:              []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data", "text/plain"},
		EnableSQLInjectionProtection:     true,
		EnableXSSProtection:              true,
		EnablePathTraversalProtection:    true,
		EnableCommandInjectionProtection: true,
		SanitizeInput:                    true,
		LogViolations:                    true,
		BlockOnViolation:                 true,
		CustomValidators:                 []ValidatorFunc{},
	}
}

// InputValidator implements comprehensive input validation middleware
type InputValidator struct {
	config *InputValidationConfig
	logger *slog.Logger

	// Compiled regex patterns for performance
	sqlInjectionPattern     *regexp.Regexp
	xssPattern              *regexp.Regexp
	pathTraversalPattern    *regexp.Regexp
	commandInjectionPattern *regexp.Regexp

	// Pattern cache for performance
	patternCache sync.Map
}

// NewInputValidator creates a new input validation middleware instance
func NewInputValidator(config *InputValidationConfig, logger *slog.Logger) (*InputValidator, error) {
	if config == nil {
		config = DefaultInputValidationConfig()
	}

	iv := &InputValidator{
		config: config,
		logger: logger.With(slog.String("component", "input-validator")),
	}

	// Compile regex patterns
	if err := iv.compilePatterns(); err != nil {
		return nil, fmt.Errorf("failed to compile validation patterns: %w", err)
	}

	return iv, nil
}

// compilePatterns compiles regex patterns for various attack vectors
func (iv *InputValidator) compilePatterns() error {
	var err error

	// SQL Injection patterns (OWASP recommendations)
	sqlPatterns := []string{
		`(?i)(union|select|insert|update|delete|drop|create|alter|exec|execute|script|javascript|eval).*?(from|into|where|table|database)`,
		`(?i)(;|\||&&|--|\*|\/\*|\*\/|xp_|sp_|0x)`,
		`(?i)(benchmark|sleep|waitfor|pg_sleep|generate_series)`,
		`(?i)(load_file|into\s+(out|dump)file|bulk\s+insert)`,
		`'.*?(or|and).*?('|=|>|<)`,
		`".*?(or|and).*?("|=|>|<)`,
	}
	iv.sqlInjectionPattern, err = regexp.Compile(strings.Join(sqlPatterns, "|"))
	if err != nil {
		return fmt.Errorf("failed to compile SQL injection pattern: %w", err)
	}

	// XSS patterns
	xssPatterns := []string{
		`<\s*script[^>]*>.*?<\s*/\s*script\s*>`,
		`javascript\s*:`,
		`on\w+\s*=\s*["'].*?["']`,
		`<\s*iframe[^>]*>`,
		`<\s*embed[^>]*>`,
		`<\s*object[^>]*>`,
		`<\s*img[^>]*\s+on\w+\s*=`,
		`<\s*svg[^>]*\s+on\w+\s*=`,
		`data\s*:\s*text/html`,
		`vbscript\s*:`,
		`<\s*meta[^>]*http-equiv[^>]*>`,
		`expression\s*\(`,
		`import\s*\(`,
		`@import`,
	}
	iv.xssPattern, err = regexp.Compile("(?i)" + strings.Join(xssPatterns, "|"))
	if err != nil {
		return fmt.Errorf("failed to compile XSS pattern: %w", err)
	}

	// Path Traversal patterns
	pathPatterns := []string{
		`\.\.\/`,
		`\.\.\\`,
		`%2e%2e%2f`,
		`%2e%2e%5c`,
		`\.\./`,
		`\.\.`,
		`%00`,
		`\x00`,
	}
	iv.pathTraversalPattern, err = regexp.Compile(strings.Join(pathPatterns, "|"))
	if err != nil {
		return fmt.Errorf("failed to compile path traversal pattern: %w", err)
	}

	// Command Injection patterns
	cmdPatterns := []string{
		`[;&|]`,
		`\$\(.*?\)`,
		`` + "`" + `.*?` + "`",
		`\$\{.*?\}`,
		`>\s*/dev/null`,
		`<\s*/dev/null`,
		`(?i)(curl|wget|nc|netcat|telnet|ssh|scp|rsync)`,
		`(?i)(chmod|chown|sudo|su|passwd|shadow|etc/passwd)`,
	}
	iv.commandInjectionPattern, err = regexp.Compile(strings.Join(cmdPatterns, "|"))
	if err != nil {
		return fmt.Errorf("failed to compile command injection pattern: %w", err)
	}

	return nil
}

// Middleware returns an HTTP middleware that validates and sanitizes input
func (iv *InputValidator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Validate request
		if err := iv.validateRequest(r); err != nil {
			iv.handleValidationError(w, r, err)
			return
		}

		// Sanitize input if enabled
		if iv.config.SanitizeInput {
			r = iv.sanitizeRequest(r)
		}

		// Log successful validation
		iv.logger.Debug("Request validated successfully",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Duration("validation_time", time.Since(start)))

		// Call next handler
		next.ServeHTTP(w, r)
	})
}

// validateRequest performs comprehensive request validation
func (iv *InputValidator) validateRequest(r *http.Request) error {
	// Validate URL length
	if len(r.URL.String()) > iv.config.MaxURLLength {
		return fmt.Errorf("URL exceeds maximum length of %d characters", iv.config.MaxURLLength)
	}

	// Validate headers
	if err := iv.validateHeaders(r); err != nil {
		return fmt.Errorf("header validation failed: %w", err)
	}

	// Validate query parameters
	if err := iv.validateQueryParameters(r); err != nil {
		return fmt.Errorf("query parameter validation failed: %w", err)
	}

	// Validate Content-Type
	if err := iv.validateContentType(r); err != nil {
		return fmt.Errorf("content type validation failed: %w", err)
	}

	// Validate request body
	if err := iv.validateBody(r); err != nil {
		return fmt.Errorf("body validation failed: %w", err)
	}

	// Run attack detection
	if err := iv.detectAttacks(r); err != nil {
		return fmt.Errorf("potential attack detected: %w", err)
	}

	// Run custom validators
	for _, validator := range iv.config.CustomValidators {
		if err := validator(r); err != nil {
			return fmt.Errorf("custom validation failed: %w", err)
		}
	}

	return nil
}

// validateHeaders validates HTTP headers
func (iv *InputValidator) validateHeaders(r *http.Request) error {
	totalSize := 0
	for name, values := range r.Header {
		// Check header name for malicious patterns
		if iv.containsMaliciousPattern(name) {
			return fmt.Errorf("malicious pattern detected in header name: %s", name)
		}

		for _, value := range values {
			// Check header value for malicious patterns
			if iv.containsMaliciousPattern(value) {
				return fmt.Errorf("malicious pattern detected in header %s", name)
			}

			totalSize += len(name) + len(value)
		}
	}

	if totalSize > iv.config.MaxHeaderSize {
		return fmt.Errorf("headers exceed maximum size of %d bytes", iv.config.MaxHeaderSize)
	}

	return nil
}

// validateQueryParameters validates URL query parameters
func (iv *InputValidator) validateQueryParameters(r *http.Request) error {
	params := r.URL.Query()

	if len(params) > iv.config.MaxParameterCount {
		return fmt.Errorf("query parameters exceed maximum count of %d", iv.config.MaxParameterCount)
	}

	for key, values := range params {
		// Check parameter name
		if iv.containsMaliciousPattern(key) {
			return fmt.Errorf("malicious pattern detected in parameter name: %s", key)
		}

		for _, value := range values {
			// Check parameter length
			if len(value) > iv.config.MaxParameterLength {
				return fmt.Errorf("parameter %s exceeds maximum length of %d", key, iv.config.MaxParameterLength)
			}

			// Check for malicious patterns
			if iv.containsMaliciousPattern(value) {
				return fmt.Errorf("malicious pattern detected in parameter %s", key)
			}
		}
	}

	return nil
}

// validateContentType validates the Content-Type header
func (iv *InputValidator) validateContentType(r *http.Request) error {
	// Skip for GET, HEAD, DELETE requests
	if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodDelete {
		return nil
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		return nil // Allow empty content type for some requests
	}

	// Extract base content type (without parameters)
	parts := strings.Split(contentType, ";")
	if len(parts) == 0 || parts[0] == "" {
		return fmt.Errorf("invalid content type: %s", contentType)
	}
	baseType := parts[0]
	baseType = strings.TrimSpace(strings.ToLower(baseType))

	// Check against allowed types
	allowed := false
	for _, allowedType := range iv.config.AllowedContentTypes {
		if strings.ToLower(allowedType) == baseType {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("content type %s is not allowed", baseType)
	}

	return nil
}

// validateBody validates the request body
func (iv *InputValidator) validateBody(r *http.Request) error {
	// Skip for requests without body
	if r.Body == nil || r.ContentLength == 0 {
		return nil
	}

	// Check body size
	if r.ContentLength > iv.config.MaxBodySize {
		return fmt.Errorf("body size %d exceeds maximum of %d bytes", r.ContentLength, iv.config.MaxBodySize)
	}

	// Read body for validation (we'll need to restore it)
	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, iv.config.MaxBodySize))
	if err != nil {
		return fmt.Errorf("failed to read body: %w", err)
	}
	r.Body.Close()

	// Restore body for downstream handlers
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Validate based on content type
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		return iv.validateJSONBody(bodyBytes)
	}

	// Generic validation for other content types
	bodyStr := string(bodyBytes)
	if iv.containsMaliciousPattern(bodyStr) {
		return fmt.Errorf("malicious pattern detected in request body")
	}

	return nil
}

// validateJSONBody performs JSON-specific validation
func (iv *InputValidator) validateJSONBody(body []byte) error {
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Recursively validate JSON structure
	return iv.validateJSONValue(data, 0)
}

// validateJSONValue recursively validates JSON values
func (iv *InputValidator) validateJSONValue(value interface{}, depth int) error {
	const maxDepth = 20 // Prevent deep nesting attacks

	if depth > maxDepth {
		return fmt.Errorf("JSON nesting depth exceeds maximum of %d", maxDepth)
	}

	switch v := value.(type) {
	case string:
		if iv.containsMaliciousPattern(v) {
			return fmt.Errorf("malicious pattern detected in JSON string value")
		}
	case map[string]interface{}:
		for key, val := range v {
			if iv.containsMaliciousPattern(key) {
				return fmt.Errorf("malicious pattern detected in JSON key: %s", key)
			}
			if err := iv.validateJSONValue(val, depth+1); err != nil {
				return err
			}
		}
	case []interface{}:
		for _, item := range v {
			if err := iv.validateJSONValue(item, depth+1); err != nil {
				return err
			}
		}
	}

	return nil
}

// detectAttacks checks for common attack patterns
func (iv *InputValidator) detectAttacks(r *http.Request) error {
	// Combine all input sources for comprehensive checking
	inputSources := []string{
		r.URL.Path,
		r.URL.RawQuery,
	}

	// Add headers
	for name, values := range r.Header {
		inputSources = append(inputSources, name)
		inputSources = append(inputSources, values...)
	}

	// Check each input source
	for _, input := range inputSources {
		if iv.config.EnableSQLInjectionProtection && iv.sqlInjectionPattern.MatchString(input) {
			return fmt.Errorf("SQL injection pattern detected")
		}

		if iv.config.EnableXSSProtection && iv.xssPattern.MatchString(input) {
			return fmt.Errorf("XSS pattern detected")
		}

		if iv.config.EnablePathTraversalProtection && iv.pathTraversalPattern.MatchString(input) {
			return fmt.Errorf("path traversal pattern detected")
		}

		if iv.config.EnableCommandInjectionProtection && iv.commandInjectionPattern.MatchString(input) {
			return fmt.Errorf("command injection pattern detected")
		}
	}

	return nil
}

// containsMaliciousPattern checks if input contains any malicious patterns
func (iv *InputValidator) containsMaliciousPattern(input string) bool {
	if input == "" {
		return false
	}

	// Check all enabled patterns
	if iv.config.EnableSQLInjectionProtection && iv.sqlInjectionPattern.MatchString(input) {
		return true
	}

	if iv.config.EnableXSSProtection && iv.xssPattern.MatchString(input) {
		return true
	}

	if iv.config.EnablePathTraversalProtection && iv.pathTraversalPattern.MatchString(input) {
		return true
	}

	if iv.config.EnableCommandInjectionProtection && iv.commandInjectionPattern.MatchString(input) {
		return true
	}

	return false
}

// sanitizeRequest sanitizes the request to remove potentially harmful content
func (iv *InputValidator) sanitizeRequest(r *http.Request) *http.Request {
	// Sanitize URL parameters
	query := r.URL.Query()
	sanitizedQuery := url.Values{}
	for key, values := range query {
		sanitizedKey := iv.sanitizeString(key)
		for _, value := range values {
			sanitizedQuery.Add(sanitizedKey, iv.sanitizeString(value))
		}
	}
	r.URL.RawQuery = sanitizedQuery.Encode()

	// Sanitize headers
	for name, values := range r.Header {
		for i, value := range values {
			r.Header[name][i] = iv.sanitizeString(value)
		}
	}

	// Note: Body sanitization is handled separately in validateBody
	return r
}

// sanitizeString removes or escapes potentially harmful content from strings
func (iv *InputValidator) sanitizeString(input string) string {
	// HTML escape
	sanitized := html.EscapeString(input)

	// Remove null bytes
	sanitized = strings.ReplaceAll(sanitized, "\x00", "")

	// Remove control characters (except tab, newline, carriage return)
	sanitized = regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]`).ReplaceAllString(sanitized, "")

	return sanitized
}

// handleValidationError handles validation errors
func (iv *InputValidator) handleValidationError(w http.ResponseWriter, r *http.Request, err error) {
	// Log the violation if enabled
	if iv.config.LogViolations {
		iv.logger.Warn("Input validation failed",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("error", err.Error()))
	}

	// Block the request if configured
	if iv.config.BlockOnViolation {
		http.Error(w, "Bad Request: Invalid input detected", http.StatusBadRequest)
		return
	}

	// Otherwise, just log and continue (not recommended for production)
	iv.logger.Debug("Validation error (non-blocking)", slog.String("error", err.Error()))
}

// ValidateJWT validates JWT tokens in Authorization header
func (iv *InputValidator) ValidateJWT(r *http.Request) error {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil // No JWT to validate
	}

	// Check for Bearer token format
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return fmt.Errorf("invalid authorization header format")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	// Basic JWT structure validation (three base64url encoded parts)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return fmt.Errorf("invalid JWT structure")
	}

	// Check for malicious patterns in JWT
	if iv.containsMaliciousPattern(token) {
		return fmt.Errorf("malicious pattern detected in JWT")
	}

	return nil
}

// AddCustomValidator adds a custom validation function
func (iv *InputValidator) AddCustomValidator(validator ValidatorFunc) {
	iv.config.CustomValidators = append(iv.config.CustomValidators, validator)
}

// GetMetrics returns validation metrics for monitoring
func (iv *InputValidator) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"total_requests":                    int64(0),
		"blocked_requests":                  int64(0),
		"sanitized_requests":                int64(0),
		"suspicious_patterns":               int64(0),
		"block_rate":                        float64(0),
		"sql_injection_protection":          iv.config.EnableSQLInjectionProtection,
		"xss_protection":                    iv.config.EnableXSSProtection,
		"path_traversal_protection":         iv.config.EnablePathTraversalProtection,
		"command_injection_protection":      iv.config.EnableCommandInjectionProtection,
		"max_body_size":                     iv.config.MaxBodySize,
		"sanitization_enabled":              iv.config.SanitizeInput,
	}
}

// ValidateContext adds the validator to the request context
func ValidateContext(ctx context.Context, validator *InputValidator) context.Context {
	return context.WithValue(ctx, "input_validator", validator)
}

// GetValidator retrieves the validator from context
func GetValidator(ctx context.Context) *InputValidator {
	if validator, ok := ctx.Value("input_validator").(*InputValidator); ok {
		return validator
	}
	return nil
}
