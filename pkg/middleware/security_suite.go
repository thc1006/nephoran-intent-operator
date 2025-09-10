// Package middleware provides HTTP middleware components for the Nephoran Intent Operator
package middleware

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// DefaultCORSConfig returns a secure default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{}, // Empty by default - must be configured explicitly
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Requested-With", "Accept", "Origin"},
		ExposedHeaders:   []string{},
		AllowCredentials: false,
		MaxAge:           24 * time.Hour,
	}
}

// NewCORS creates a new CORS middleware instance
func NewCORS(config *CORSConfig, logger *slog.Logger) *CORSMiddleware {
	if config == nil {
		defaultConfig := DefaultCORSConfig()
		config = &defaultConfig
	}
	return NewCORSMiddleware(*config, logger)
}

// SecuritySuiteConfig provides comprehensive security configuration
type SecuritySuiteConfig struct {
	// Core security components
	SecurityHeaders *SecurityHeadersConfig
	InputValidation *InputValidationConfig
	RateLimit       *RateLimiterConfig
	RequestSize     *RequestSizeLimiter
	CORS            *CORSConfig

	// Authentication & Authorization
	RequireAuth   bool
	AuthValidator func(r *http.Request) (bool, error)

	// Audit & Logging
	EnableAudit bool
	AuditLogger *slog.Logger

	// Metrics
	EnableMetrics bool
	MetricsPrefix string

	// Security Policy
	EnableCSRF      bool
	CSRFTokenHeader string
	CSRFCookieName  string

	// Request fingerprinting for anomaly detection
	EnableFingerprinting bool

	// IP-based security
	IPWhitelist []string
	IPBlacklist []string

	// Timeout configurations
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DefaultSecuritySuiteConfig returns a comprehensive secure configuration
func DefaultSecuritySuiteConfig() *SecuritySuiteConfig {
	return &SecuritySuiteConfig{
		SecurityHeaders:      DefaultSecurityHeadersConfig(),
		InputValidation:      DefaultInputValidationConfig(),
		RateLimit:            func() *RateLimiterConfig { c := DefaultRateLimiterConfig(); return &c }(),
		RequestSize:          NewRequestSizeLimiter(32*1024*1024, nil), // 32MB default
		CORS:                 func() *CORSConfig { c := DefaultCORSConfig(); return &c }(),
		RequireAuth:          false,
		EnableAudit:          true,
		EnableMetrics:        true,
		MetricsPrefix:        "nephoran_security",
		EnableCSRF:           false,
		CSRFTokenHeader:      "X-CSRF-Token",
		CSRFCookieName:       "csrf_token",
		EnableFingerprinting: true,
		ReadTimeout:          30 * time.Second,
		WriteTimeout:         30 * time.Second,
		IdleTimeout:          120 * time.Second,
	}
}

// SecuritySuite provides comprehensive security middleware
type SecuritySuite struct {
	config *SecuritySuiteConfig
	logger *slog.Logger

	// Component middlewares
	headers     *SecurityHeaders
	validator   *InputValidator
	rateLimiter *RateLimiter
	sizeLimit   *RequestSizeLimiter
	cors        *CORSMiddleware

	// Security state
	fingerprintCache sync.Map
	csrfTokens       sync.Map

	// Metrics
	metrics *securityMetrics
}

// securityMetrics holds Prometheus metrics for security monitoring
type securityMetrics struct {
	requestsTotal      *prometheus.CounterVec
	requestsDuration   *prometheus.HistogramVec
	securityViolations *prometheus.CounterVec
	authFailures       *prometheus.CounterVec
	rateLimitHits      prometheus.Counter
	maliciousRequests  *prometheus.CounterVec
}

// NewSecuritySuite creates a comprehensive security middleware suite
func NewSecuritySuite(config *SecuritySuiteConfig, logger *slog.Logger) (*SecuritySuite, error) {
	if config == nil {
		config = DefaultSecuritySuiteConfig()
	}

	ss := &SecuritySuite{
		config: config,
		logger: logger.With(slog.String("component", "security-suite")),
	}

	// Initialize component middlewares
	if config.SecurityHeaders != nil {
		ss.headers = NewSecurityHeaders(config.SecurityHeaders, logger)
	}

	if config.InputValidation != nil {
		var err error
		ss.validator, err = NewInputValidator(config.InputValidation, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create input validator: %w", err)
		}
	}

	if config.RateLimit != nil {
		ss.rateLimiter = NewRateLimiter(*config.RateLimit, logger)
	}

	if config.RequestSize != nil {
		ss.sizeLimit = config.RequestSize // RequestSize is already a *RequestSizeLimiter
	}

	if config.CORS != nil {
		ss.cors = NewCORS(config.CORS, logger)
	}

	// Initialize metrics if enabled
	if config.EnableMetrics {
		ss.initMetrics()
	}

	return ss, nil
}

// initMetrics initializes Prometheus metrics
func (ss *SecuritySuite) initMetrics() {
	prefix := ss.config.MetricsPrefix
	if prefix == "" {
		prefix = "security"
	}

	ss.metrics = &securityMetrics{
		requestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: fmt.Sprintf("%s_requests_total", prefix),
				Help: "Total number of requests processed",
			},
			[]string{"method", "path", "status"},
		),
		requestsDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    fmt.Sprintf("%s_request_duration_seconds", prefix),
				Help:    "Request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		securityViolations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: fmt.Sprintf("%s_violations_total", prefix),
				Help: "Total number of security violations",
			},
			[]string{"type", "severity"},
		),
		authFailures: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: fmt.Sprintf("%s_auth_failures_total", prefix),
				Help: "Total number of authentication failures",
			},
			[]string{"reason"},
		),
		rateLimitHits: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: fmt.Sprintf("%s_rate_limit_hits_total", prefix),
				Help: "Total number of rate limit hits",
			},
		),
		maliciousRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: fmt.Sprintf("%s_malicious_requests_total", prefix),
				Help: "Total number of detected malicious requests",
			},
			[]string{"attack_type"},
		),
	}
}

// Middleware returns the complete security middleware chain
func (ss *SecuritySuite) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create wrapped response writer for capturing status
		wrapped := &securityResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Create request context with security info
		ctx := r.Context()
		ctx = context.WithValue(ctx, "security_suite", ss)
		ctx = context.WithValue(ctx, "request_start", start)
		r = r.WithContext(ctx)

		// IP-based filtering
		if err := ss.checkIPFilters(r); err != nil {
			ss.handleSecurityViolation(wrapped, r, "ip_filter", err)
			return
		}

		// Request fingerprinting for anomaly detection
		if ss.config.EnableFingerprinting {
			fingerprint := ss.generateFingerprint(r)
			ctx = context.WithValue(ctx, "request_fingerprint", fingerprint)
			r = r.WithContext(ctx)
		}

		// Apply security headers first
		if ss.headers != nil {
			ss.headers.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Continue with the chain
				ss.processRequest(wrapped, r, next)
			})).ServeHTTP(wrapped, r)
		} else {
			ss.processRequest(wrapped, r, next)
		}

		// Record metrics
		if ss.config.EnableMetrics && ss.metrics != nil {
			duration := time.Since(start).Seconds()
			ss.metrics.requestsDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
			ss.metrics.requestsTotal.WithLabelValues(r.Method, r.URL.Path, fmt.Sprintf("%d", wrapped.statusCode)).Inc()
		}

		// Audit logging
		if ss.config.EnableAudit {
			ss.auditLog(r, wrapped.statusCode, time.Since(start))
		}
	})
}

// processRequest handles the main request processing chain
func (ss *SecuritySuite) processRequest(w http.ResponseWriter, r *http.Request, next http.Handler) {
	// CORS handling
	if ss.cors != nil {
		ss.cors.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ss.continueProcessing(w, r, next)
		})).ServeHTTP(w, r)
	} else {
		ss.continueProcessing(w, r, next)
	}
}

// continueProcessing continues the middleware chain
func (ss *SecuritySuite) continueProcessing(w http.ResponseWriter, r *http.Request, next http.Handler) {
	// Request size limiting
	if ss.sizeLimit != nil {
		ss.sizeLimit.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ss.validateAndProcess(w, r, next)
		})).ServeHTTP(w, r)
	} else {
		ss.validateAndProcess(w, r, next)
	}
}

// validateAndProcess handles validation and further processing
func (ss *SecuritySuite) validateAndProcess(w http.ResponseWriter, r *http.Request, next http.Handler) {
	// Input validation
	if ss.validator != nil {
		ss.validator.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ss.authenticateAndProcess(w, r, next)
		})).ServeHTTP(w, r)
	} else {
		ss.authenticateAndProcess(w, r, next)
	}
}

// authenticateAndProcess handles authentication and rate limiting
func (ss *SecuritySuite) authenticateAndProcess(w http.ResponseWriter, r *http.Request, next http.Handler) {
	// Authentication check
	if ss.config.RequireAuth && ss.config.AuthValidator != nil {
		authenticated, err := ss.config.AuthValidator(r)
		if err != nil || !authenticated {
			if ss.config.EnableMetrics && ss.metrics != nil {
				reason := "invalid_credentials"
				if err != nil {
					reason = "auth_error"
				}
				ss.metrics.authFailures.WithLabelValues(reason).Inc()
			}
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// CSRF protection
	if ss.config.EnableCSRF && !ss.validateCSRF(r) {
		ss.handleSecurityViolation(w, r, "csrf", fmt.Errorf("invalid CSRF token"))
		return
	}

	// Rate limiting
	if ss.rateLimiter != nil {
		ss.rateLimiter.Middleware(next).ServeHTTP(w, r)
	} else {
		next.ServeHTTP(w, r)
	}
}

// checkIPFilters checks IP whitelist/blacklist
func (ss *SecuritySuite) checkIPFilters(r *http.Request) error {
	clientIP := ss.getClientIP(r)

	// Check blacklist first
	for _, ip := range ss.config.IPBlacklist {
		if ip == clientIP {
			return fmt.Errorf("IP %s is blacklisted", clientIP)
		}
	}

	// Check whitelist if configured
	if len(ss.config.IPWhitelist) > 0 {
		whitelisted := false
		for _, ip := range ss.config.IPWhitelist {
			if ip == clientIP {
				whitelisted = true
				break
			}
		}
		if !whitelisted {
			return fmt.Errorf("IP %s is not whitelisted", clientIP)
		}
	}

	return nil
}

// generateFingerprint creates a unique fingerprint for the request
func (ss *SecuritySuite) generateFingerprint(r *http.Request) string {
	h := sha256.New()

	// Include various request attributes
	h.Write([]byte(r.Method))
	h.Write([]byte(r.URL.Path))
	h.Write([]byte(r.Header.Get("User-Agent")))
	h.Write([]byte(r.Header.Get("Accept")))
	h.Write([]byte(r.Header.Get("Accept-Language")))
	h.Write([]byte(r.Header.Get("Accept-Encoding")))
	h.Write([]byte(ss.getClientIP(r)))

	return hex.EncodeToString(h.Sum(nil))
}

// validateCSRF validates CSRF token
func (ss *SecuritySuite) validateCSRF(r *http.Request) bool {
	// Skip CSRF for safe methods
	if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
		return true
	}

	// Get token from header
	headerToken := r.Header.Get(ss.config.CSRFTokenHeader)
	if headerToken == "" {
		return false
	}

	// Get token from cookie
	cookie, err := r.Cookie(ss.config.CSRFCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}

	// Compare tokens
	return headerToken == cookie.Value
}

// getClientIP extracts the client IP address from the request
func (ss *SecuritySuite) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
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

// handleSecurityViolation handles security violations
func (ss *SecuritySuite) handleSecurityViolation(w http.ResponseWriter, r *http.Request, violationType string, err error) {
	// Log the violation
	ss.logger.Warn("Security violation detected",
		slog.String("type", violationType),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("client_ip", ss.getClientIP(r)),
		slog.String("error", err.Error()))

	// Record metrics
	if ss.config.EnableMetrics && ss.metrics != nil {
		ss.metrics.securityViolations.WithLabelValues(violationType, "high").Inc()
		ss.metrics.maliciousRequests.WithLabelValues(violationType).Inc()
	}

	// Return error response
	http.Error(w, "Security violation detected", http.StatusForbidden)
}

// auditLog creates an audit log entry for the request
func (ss *SecuritySuite) auditLog(r *http.Request, statusCode int, duration time.Duration) {
	if ss.config.AuditLogger == nil {
		ss.config.AuditLogger = ss.logger
	}

	ss.config.AuditLogger.Info("Security audit",
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("client_ip", ss.getClientIP(r)),
		slog.String("user_agent", r.Header.Get("User-Agent")),
		slog.Int("status_code", statusCode),
		slog.Duration("duration", duration),
		slog.String("fingerprint", r.Context().Value("request_fingerprint").(string)))
}

// securityResponseWriter wraps http.ResponseWriter to capture status code
type securityResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *securityResponseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.ResponseWriter.WriteHeader(code)
		rw.written = true
	}
}

func (rw *securityResponseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// GetSecuritySuite retrieves the security suite from context
func GetSecuritySuite(ctx context.Context) *SecuritySuite {
	if suite, ok := ctx.Value("security_suite").(*SecuritySuite); ok {
		return suite
	}
	return nil
}

// GenerateCSRFToken generates a new CSRF token
func (ss *SecuritySuite) GenerateCSRFToken() string {
	token := make([]byte, 32)
	_, err := rand.Read(token)
	if err != nil {
		// Fallback to time-based generation if crypto/rand fails
		for i := range token {
			token[i] = byte((time.Now().UnixNano() + int64(i)) & 0xFF)
		}
	}
	h := sha256.Sum256(token)
	tokenStr := hex.EncodeToString(h[:])

	// Store token with expiry
	ss.csrfTokens.Store(tokenStr, time.Now().Add(24*time.Hour))

	return tokenStr
}

// ValidateCSRFToken validates a CSRF token
func (ss *SecuritySuite) ValidateCSRFToken(token string) bool {
	if val, ok := ss.csrfTokens.Load(token); ok {
		expiry := val.(time.Time)
		if time.Now().Before(expiry) {
			return true
		}
		// Clean up expired token
		ss.csrfTokens.Delete(token)
	}
	return false
}

// CleanupExpiredTokens removes expired CSRF tokens
func (ss *SecuritySuite) CleanupExpiredTokens() {
	ss.csrfTokens.Range(func(key, value interface{}) bool {
		expiry := value.(time.Time)
		if time.Now().After(expiry) {
			ss.csrfTokens.Delete(key)
		}
		return true
	})
}
