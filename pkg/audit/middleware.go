
package audit



import (

	"bufio"

	"bytes"

	"context"

	"encoding/json"

	"errors"

	"fmt"

	"io"

	"net"

	"net/http"

	"strconv"

	"strings"

	"sync"

	"time"



	"github.com/go-logr/logr"

	"github.com/google/uuid"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus/client_golang/prometheus/promauto"

	"go.opentelemetry.io/otel/trace"



	"sigs.k8s.io/controller-runtime/pkg/log"

)



const (

	// HTTP header names for audit context.

	HeaderRequestID = "X-Request-ID"

	// HeaderCorrelationID holds headercorrelationid value.

	HeaderCorrelationID = "X-Correlation-ID"

	// HeaderUserID holds headeruserid value.

	HeaderUserID = "X-User-ID"

	// HeaderUserAgent holds headeruseragent value.

	HeaderUserAgent = "User-Agent"

	// HeaderForwardedFor holds headerforwardedfor value.

	HeaderForwardedFor = "X-Forwarded-For"

	// HeaderRealIP holds headerrealip value.

	HeaderRealIP = "X-Real-IP"

	// HeaderAuthorization holds headerauthorization value.

	HeaderAuthorization = "Authorization"

	// HeaderContentType holds headercontenttype value.

	HeaderContentType = "Content-Type"

	// HeaderContentLength holds headercontentlength value.

	HeaderContentLength = "Content-Length"

	// HeaderAccept holds headeraccept value.

	HeaderAccept = "Accept"



	// Context keys.

	ContextKeyRequestID = "audit.request_id"

	// ContextKeyUserID holds contextkeyuserid value.

	ContextKeyUserID = "audit.user_id"

	// ContextKeyStartTime holds contextkeystarttime value.

	ContextKeyStartTime = "audit.start_time"

	// ContextKeyAuditEvent holds contextkeyauditevent value.

	ContextKeyAuditEvent = "audit.event"



	// Maximum request/response body size to capture (10MB).

	MaxBodySize = 10 * 1024 * 1024

)



var (

	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{

		Name: "audit_http_requests_total",

		Help: "Total number of HTTP requests processed by audit middleware",

	}, []string{"method", "path", "status", "user_id"})



	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{

		Name:    "audit_http_request_duration_seconds",

		Help:    "Duration of HTTP requests in seconds",

		Buckets: []float64{0.001, 0.01, 0.1, 0.5, 1.0, 2.5, 5.0, 10.0},

	}, []string{"method", "path"})



	httpRequestSize = promauto.NewHistogramVec(prometheus.HistogramOpts{

		Name:    "audit_http_request_size_bytes",

		Help:    "Size of HTTP requests in bytes",

		Buckets: prometheus.ExponentialBuckets(64, 2, 16),

	}, []string{"method", "path"})



	httpResponseSize = promauto.NewHistogramVec(prometheus.HistogramOpts{

		Name:    "audit_http_response_size_bytes",

		Help:    "Size of HTTP responses in bytes",

		Buckets: prometheus.ExponentialBuckets(64, 2, 16),

	}, []string{"method", "path", "status"})

)



// HTTPAuditMiddleware provides HTTP request/response auditing.

type HTTPAuditMiddleware struct {

	auditSystem *AuditSystem

	config      *HTTPAuditConfig

	logger      logr.Logger



	// Request correlation.

	requestPool  sync.Pool

	responsePool sync.Pool

}



// HTTPAuditConfig configures the HTTP audit middleware.

type HTTPAuditConfig struct {

	// Enabled controls whether HTTP auditing is active.

	Enabled bool `json:"enabled" yaml:"enabled"`



	// CaptureRequestBody controls whether to capture request bodies.

	CaptureRequestBody bool `json:"capture_request_body" yaml:"capture_request_body"`



	// CaptureResponseBody controls whether to capture response bodies.

	CaptureResponseBody bool `json:"capture_response_body" yaml:"capture_response_body"`



	// CaptureHeaders controls whether to capture HTTP headers.

	CaptureHeaders bool `json:"capture_headers" yaml:"capture_headers"`



	// MaxBodySize limits the size of request/response bodies to capture.

	MaxBodySize int64 `json:"max_body_size" yaml:"max_body_size"`



	// SensitiveHeaders defines headers that should be redacted.

	SensitiveHeaders []string `json:"sensitive_headers" yaml:"sensitive_headers"`



	// SensitivePaths defines URL paths that should have bodies redacted.

	SensitivePaths []string `json:"sensitive_paths" yaml:"sensitive_paths"`



	// ExcludePaths defines URL paths to exclude from auditing.

	ExcludePaths []string `json:"exclude_paths" yaml:"exclude_paths"`



	// HealthCheckPaths defines health check paths to exclude from auditing.

	HealthCheckPaths []string `json:"health_check_paths" yaml:"health_check_paths"`



	// UserExtractor defines how to extract user information from requests.

	UserExtractor UserExtractorFunc `json:"-" yaml:"-"`



	// PathSanitizer defines how to sanitize URL paths for metrics/logging.

	PathSanitizer PathSanitizerFunc `json:"-" yaml:"-"`

}



// UserExtractorFunc extracts user information from HTTP requests.

type UserExtractorFunc func(*http.Request) *UserContext



// PathSanitizerFunc sanitizes URL paths to remove sensitive information.

type PathSanitizerFunc func(string) string



// HTTPRequestInfo captures information about an HTTP request.

type HTTPRequestInfo struct {

	RequestID     string              `json:"request_id"`

	Method        string              `json:"method"`

	Path          string              `json:"path"`

	Query         map[string][]string `json:"query,omitempty"`

	Headers       map[string]string   `json:"headers,omitempty"`

	Body          string              `json:"body,omitempty"`

	BodySize      int64               `json:"body_size"`

	RemoteAddr    string              `json:"remote_addr"`

	UserAgent     string              `json:"user_agent"`

	Referer       string              `json:"referer,omitempty"`

	ContentType   string              `json:"content_type,omitempty"`

	AcceptedTypes []string            `json:"accepted_types,omitempty"`

	StartTime     time.Time           `json:"start_time"`

}



// HTTPResponseInfo captures information about an HTTP response.

type HTTPResponseInfo struct {

	StatusCode  int               `json:"status_code"`

	StatusText  string            `json:"status_text"`

	Headers     map[string]string `json:"headers,omitempty"`

	Body        string            `json:"body,omitempty"`

	BodySize    int64             `json:"body_size"`

	ContentType string            `json:"content_type,omitempty"`

	Duration    time.Duration     `json:"duration"`

	EndTime     time.Time         `json:"end_time"`

}



// auditResponseWriter wraps http.ResponseWriter to capture response information.

type auditResponseWriter struct {

	http.ResponseWriter

	statusCode  int

	size        int64

	body        *bytes.Buffer

	config      *HTTPAuditConfig

	captureBody bool

}



// NewHTTPAuditMiddleware creates a new HTTP audit middleware.

func NewHTTPAuditMiddleware(auditSystem *AuditSystem, config *HTTPAuditConfig) *HTTPAuditMiddleware {

	if config == nil {

		config = DefaultHTTPAuditConfig()

	}



	middleware := &HTTPAuditMiddleware{

		auditSystem: auditSystem,

		config:      config,

		logger:      log.Log.WithName("http-audit-middleware"),

	}



	// Initialize pools for request/response capture.

	middleware.requestPool = sync.Pool{

		New: func() interface{} {

			return &bytes.Buffer{}

		},

	}



	middleware.responsePool = sync.Pool{

		New: func() interface{} {

			return &bytes.Buffer{}

		},

	}



	return middleware

}



// Handler returns an HTTP middleware handler.

func (m *HTTPAuditMiddleware) Handler(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if !m.config.Enabled {

			next.ServeHTTP(w, r)

			return

		}



		// Check if this path should be excluded.

		if m.shouldExcludePath(r.URL.Path) {

			next.ServeHTTP(w, r)

			return

		}



		// Generate request ID if not present.

		requestID := r.Header.Get(HeaderRequestID)

		if requestID == "" {

			requestID = uuid.New().String()

		}



		// Add request ID to response header.

		w.Header().Set(HeaderRequestID, requestID)



		// Capture request information.

		requestInfo, err := m.captureRequestInfo(r, requestID)

		if err != nil {

			m.logger.Error(err, "Failed to capture request info", "request_id", requestID)

		}



		// Add context information to request.

		ctx := context.WithValue(r.Context(), ContextKeyRequestID, requestID)

		ctx = context.WithValue(ctx, ContextKeyStartTime, time.Now())

		r = r.WithContext(ctx)



		// Wrap response writer.

		shouldCaptureResponseBody := m.config.CaptureResponseBody && !m.isSensitivePath(r.URL.Path)

		wrappedWriter := &auditResponseWriter{

			ResponseWriter: w,

			statusCode:     200, // Default status code

			body:           bytes.NewBuffer(nil),

			config:         m.config,

			captureBody:    shouldCaptureResponseBody,

		}



		// Execute the request.

		startTime := time.Now()

		next.ServeHTTP(wrappedWriter, r)

		duration := time.Since(startTime)



		// Capture response information.

		responseInfo := m.captureResponseInfo(wrappedWriter, duration)



		// Create audit event.

		m.createHTTPAuditEvent(r, requestInfo, responseInfo, duration)



		// Update metrics.

		m.updateMetrics(r, requestInfo, responseInfo, duration)

	})

}



// captureRequestInfo captures information about the HTTP request.

func (m *HTTPAuditMiddleware) captureRequestInfo(r *http.Request, requestID string) (*HTTPRequestInfo, error) {

	info := &HTTPRequestInfo{

		RequestID:   requestID,

		Method:      r.Method,

		Path:        r.URL.Path,

		Query:       r.URL.Query(),

		BodySize:    r.ContentLength,

		RemoteAddr:  m.getClientIP(r),

		UserAgent:   r.Header.Get(HeaderUserAgent),

		Referer:     r.Header.Get("Referer"),

		ContentType: r.Header.Get(HeaderContentType),

		StartTime:   time.Now().UTC(),

	}



	// Capture headers if enabled.

	if m.config.CaptureHeaders {

		info.Headers = m.sanitizeHeaders(r.Header)

	}



	// Parse accepted content types.

	if acceptHeader := r.Header.Get(HeaderAccept); acceptHeader != "" {

		info.AcceptedTypes = strings.Split(acceptHeader, ",")

		for i, t := range info.AcceptedTypes {

			info.AcceptedTypes[i] = strings.TrimSpace(t)

		}

	}



	// Capture request body if enabled and not sensitive.

	if m.config.CaptureRequestBody && !m.isSensitivePath(r.URL.Path) && r.Body != nil {

		body, err := m.captureRequestBody(r)

		if err != nil {

			return info, fmt.Errorf("failed to capture request body: %w", err)

		}

		info.Body = body

	}



	return info, nil

}



// captureRequestBody reads and captures the request body.

func (m *HTTPAuditMiddleware) captureRequestBody(r *http.Request) (string, error) {

	if r.ContentLength > m.config.MaxBodySize {

		return fmt.Sprintf("[TRUNCATED: Body size %d exceeds limit %d]", r.ContentLength, m.config.MaxBodySize), nil

	}



	// Get buffer from pool.

	buf := m.requestPool.Get().(*bytes.Buffer)

	defer func() {

		buf.Reset()

		m.requestPool.Put(buf)

	}()



	// Read body.

	_, err := io.CopyN(buf, r.Body, m.config.MaxBodySize+1)

	if err != nil && !errors.Is(err, io.EOF) {

		return "", err

	}



	bodyBytes := buf.Bytes()

	if int64(len(bodyBytes)) > m.config.MaxBodySize {

		bodyBytes = bodyBytes[:m.config.MaxBodySize]

		bodyString := string(bodyBytes) + "[TRUNCATED]"



		// Replace body so subsequent handlers can read it.

		r.Body = io.NopCloser(strings.NewReader(string(bodyBytes[:m.config.MaxBodySize-int64(len("[TRUNCATED]"))])))

		return bodyString, nil

	}



	// Replace body so subsequent handlers can read it.

	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))



	// Try to format JSON for better readability.

	if strings.Contains(r.Header.Get(HeaderContentType), "application/json") {

		var jsonData interface{}

		if err := json.Unmarshal(bodyBytes, &jsonData); err == nil {

			if prettyJSON, err := json.MarshalIndent(jsonData, "", "  "); err == nil {

				return string(prettyJSON), nil

			}

		}

	}



	return string(bodyBytes), nil

}



// captureResponseInfo captures information about the HTTP response.

func (m *HTTPAuditMiddleware) captureResponseInfo(w *auditResponseWriter, duration time.Duration) *HTTPResponseInfo {

	info := &HTTPResponseInfo{

		StatusCode:  w.statusCode,

		StatusText:  http.StatusText(w.statusCode),

		BodySize:    w.size,

		Duration:    duration,

		EndTime:     time.Now().UTC(),

		ContentType: w.Header().Get(HeaderContentType),

	}



	// Capture headers if enabled.

	if m.config.CaptureHeaders {

		info.Headers = m.sanitizeHeaders(http.Header(w.Header()))

	}



	// Capture response body if enabled and captured.

	if w.captureBody && w.body.Len() > 0 {

		bodyString := w.body.String()



		// Try to format JSON for better readability.

		if strings.Contains(info.ContentType, "application/json") {

			var jsonData interface{}

			if err := json.Unmarshal([]byte(bodyString), &jsonData); err == nil {

				if prettyJSON, err := json.MarshalIndent(jsonData, "", "  "); err == nil {

					bodyString = string(prettyJSON)

				}

			}

		}



		info.Body = bodyString

	}



	return info

}



// createHTTPAuditEvent creates an audit event for the HTTP request/response.

func (m *HTTPAuditMiddleware) createHTTPAuditEvent(r *http.Request, requestInfo *HTTPRequestInfo, responseInfo *HTTPResponseInfo, duration time.Duration) {

	// Extract user context.

	var userContext *UserContext

	if m.config.UserExtractor != nil {

		userContext = m.config.UserExtractor(r)

	} else {

		userContext = m.defaultUserExtractor(r)

	}



	// Determine event result.

	var result EventResult

	if responseInfo.StatusCode >= 200 && responseInfo.StatusCode < 400 {

		result = ResultSuccess

	} else if responseInfo.StatusCode >= 400 && responseInfo.StatusCode < 500 {

		result = ResultDenied

	} else {

		result = ResultError

	}



	// Determine severity based on status code and method.

	severity := SeverityInfo

	if responseInfo.StatusCode >= 400 {

		severity = SeverityWarning

	}

	if responseInfo.StatusCode >= 500 {

		severity = SeverityError

	}

	if requestInfo.Method == "DELETE" && result == ResultSuccess {

		severity = SeverityNotice

	}



	// Get sanitized path for consistent logging.

	sanitizedPath := requestInfo.Path

	if m.config.PathSanitizer != nil {

		sanitizedPath = m.config.PathSanitizer(requestInfo.Path)

	}



	// Create audit event.

	event := NewEventBuilder().

		WithEventType(EventTypeAPICall).

		WithSeverity(severity).

		WithComponent("http-api").

		WithAction(strings.ToLower(requestInfo.Method)).

		WithDescription(fmt.Sprintf("HTTP %s %s", requestInfo.Method, sanitizedPath)).

		WithResult(result).

		WithNetwork(requestInfo.RemoteAddr, requestInfo.UserAgent).

		WithResource("http-endpoint", sanitizedPath, requestInfo.Method).

		WithData("request_info", requestInfo).

		WithData("response_info", responseInfo).

		WithData("duration_ms", duration.Milliseconds()).

		Build()



	// Add user context if available.

	if userContext != nil {

		event.UserContext = userContext

	}



	// Add tracing information if available.

	if span := trace.SpanFromContext(r.Context()); span != nil && span.SpanContext().HasTraceID() {

		spanContext := span.SpanContext()

		event.TraceID = spanContext.TraceID().String()

		event.SpanID = spanContext.SpanID().String()

	}



	// Log the event.

	if err := m.auditSystem.LogEvent(event); err != nil {

		m.logger.Error(err, "Failed to log HTTP audit event",

			"request_id", requestInfo.RequestID,

			"method", requestInfo.Method,

			"path", sanitizedPath)

	}

}



// updateMetrics updates Prometheus metrics.

func (m *HTTPAuditMiddleware) updateMetrics(r *http.Request, requestInfo *HTTPRequestInfo, responseInfo *HTTPResponseInfo, duration time.Duration) {

	// Sanitize path for metrics.

	path := requestInfo.Path

	if m.config.PathSanitizer != nil {

		path = m.config.PathSanitizer(path)

	}



	userID := "anonymous"

	if userContext := m.defaultUserExtractor(r); userContext != nil && userContext.UserID != "" {

		userID = userContext.UserID

	}



	// Update metrics.

	httpRequestsTotal.WithLabelValues(

		requestInfo.Method,

		path,

		strconv.Itoa(responseInfo.StatusCode),

		userID,

	).Inc()



	httpRequestDuration.WithLabelValues(

		requestInfo.Method,

		path,

	).Observe(duration.Seconds())



	if requestInfo.BodySize > 0 {

		httpRequestSize.WithLabelValues(

			requestInfo.Method,

			path,

		).Observe(float64(requestInfo.BodySize))

	}



	if responseInfo.BodySize > 0 {

		httpResponseSize.WithLabelValues(

			requestInfo.Method,

			path,

			strconv.Itoa(responseInfo.StatusCode),

		).Observe(float64(responseInfo.BodySize))

	}

}



// Helper methods.



func (m *HTTPAuditMiddleware) getClientIP(r *http.Request) string {

	// Check X-Forwarded-For header.

	if xff := r.Header.Get(HeaderForwardedFor); xff != "" {

		// Take the first IP if multiple are present.

		if ips := strings.Split(xff, ","); len(ips) > 0 {

			return strings.TrimSpace(ips[0])

		}

	}



	// Check X-Real-IP header.

	if realIP := r.Header.Get(HeaderRealIP); realIP != "" {

		return realIP

	}



	// Fall back to RemoteAddr.

	ip, _, err := net.SplitHostPort(r.RemoteAddr)

	if err != nil {

		return r.RemoteAddr

	}

	return ip

}



func (m *HTTPAuditMiddleware) sanitizeHeaders(headers http.Header) map[string]string {

	sanitized := make(map[string]string)

	for name, values := range headers {

		if m.isSensitiveHeader(name) {

			sanitized[name] = "[REDACTED]"

		} else {

			sanitized[name] = strings.Join(values, ", ")

		}

	}

	return sanitized

}



func (m *HTTPAuditMiddleware) isSensitiveHeader(header string) bool {

	for _, sensitive := range m.config.SensitiveHeaders {

		if strings.EqualFold(sensitive, header) {

			return true

		}

	}



	// Default sensitive headers.

	defaultSensitive := []string{

		"authorization", "x-api-key", "x-auth-token",

		"cookie", "set-cookie", "x-csrf-token",

	}

	for _, sensitive := range defaultSensitive {

		if strings.EqualFold(sensitive, header) {

			return true

		}

	}



	return false

}



func (m *HTTPAuditMiddleware) isSensitivePath(path string) bool {

	for _, sensitivePath := range m.config.SensitivePaths {

		if strings.HasPrefix(path, sensitivePath) {

			return true

		}

	}

	return false

}



func (m *HTTPAuditMiddleware) shouldExcludePath(path string) bool {

	// Check health check paths.

	for _, healthPath := range m.config.HealthCheckPaths {

		if path == healthPath {

			return true

		}

	}



	// Check excluded paths.

	for _, excludePath := range m.config.ExcludePaths {

		if strings.HasPrefix(path, excludePath) {

			return true

		}

	}



	return false

}



func (m *HTTPAuditMiddleware) defaultUserExtractor(r *http.Request) *UserContext {

	userID := r.Header.Get(HeaderUserID)

	if userID == "" {

		// Try to extract from Authorization header.

		if auth := r.Header.Get(HeaderAuthorization); auth != "" {

			// This is a simplified extraction - in practice, you'd decode JWT tokens.

			if strings.HasPrefix(auth, "Bearer ") {

				userID = "bearer-token-user" // Placeholder

			} else if strings.HasPrefix(auth, "Basic ") {

				userID = "basic-auth-user" // Placeholder

			}

		}

	}



	if userID == "" {

		return nil

	}



	return &UserContext{

		UserID: userID,

	}

}



// auditResponseWriter methods.



// Write performs write operation.

func (w *auditResponseWriter) Write(data []byte) (int, error) {

	// Write to original response writer.

	n, err := w.ResponseWriter.Write(data)

	w.size += int64(n)



	// Capture body if enabled and within limits.

	if w.captureBody && w.body.Len()+n <= int(w.config.MaxBodySize) {

		w.body.Write(data[:n])

	}



	return n, err

}



// WriteHeader performs writeheader operation.

func (w *auditResponseWriter) WriteHeader(statusCode int) {

	w.statusCode = statusCode

	w.ResponseWriter.WriteHeader(statusCode)

}



// Header performs header operation.

func (w *auditResponseWriter) Header() http.Header {

	return w.ResponseWriter.Header()

}



// Implement http.Hijacker if the underlying ResponseWriter supports it.

func (w *auditResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {

	hijacker, ok := w.ResponseWriter.(http.Hijacker)

	if !ok {

		return nil, nil, fmt.Errorf("ResponseWriter does not support hijacking")

	}

	return hijacker.Hijack()

}



// Implement http.Flusher if the underlying ResponseWriter supports it.

func (w *auditResponseWriter) Flush() {

	flusher, ok := w.ResponseWriter.(http.Flusher)

	if ok {

		flusher.Flush()

	}

}



// DefaultHTTPAuditConfig returns a default HTTP audit configuration.

func DefaultHTTPAuditConfig() *HTTPAuditConfig {

	return &HTTPAuditConfig{

		Enabled:             true,

		CaptureRequestBody:  false, // Default to false for performance

		CaptureResponseBody: false, // Default to false for performance

		CaptureHeaders:      true,

		MaxBodySize:         MaxBodySize,

		SensitiveHeaders: []string{

			"Authorization", "X-API-Key", "X-Auth-Token",

			"Cookie", "Set-Cookie", "X-CSRF-Token",

		},

		SensitivePaths: []string{

			"/api/v1/auth", "/api/v1/login", "/api/v1/password",

			"/api/v1/secrets", "/api/v1/keys",

		},

		ExcludePaths: []string{

			"/metrics", "/debug", "/favicon.ico",

		},

		HealthCheckPaths: []string{

			"/health", "/healthz", "/readiness", "/liveness",

			"/status", "/ping",

		},

	}

}



// DefaultPathSanitizer provides a default path sanitization function.

func DefaultPathSanitizer(path string) string {

	// Replace UUIDs, numbers, and other variable path components.

	// This is a simplified example - in practice, you'd use regex.

	parts := strings.Split(path, "/")

	for i, part := range parts {

		if len(part) == 36 && strings.Count(part, "-") == 4 {

			// Likely a UUID.

			parts[i] = "{uuid}"

		} else if _, err := strconv.Atoi(part); err == nil && part != "" {

			// Numeric ID.

			parts[i] = "{id}"

		}

	}

	return strings.Join(parts, "/")

}

