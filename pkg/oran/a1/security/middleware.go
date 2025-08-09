package security

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/a1"
)

// SecurityMiddleware provides comprehensive security middleware for A1 service
type SecurityMiddleware struct {
	config            *SecurityConfig
	logger            *logging.StructuredLogger
	authManager       *AuthManager
	mtlsManager       *MTLSManager
	encryptionManager *EncryptionManager
	sanitizer         *SanitizationManager
	auditLogger       *AuditLogger
	rateLimiter       *RateLimiter
}

// SecurityConfig holds comprehensive security configuration
type SecurityConfig struct {
	Authentication  *AuthConfig         `json:"authentication"`
	MTLS            *MTLSConfig         `json:"mtls"`
	Encryption      *EncryptionConfig   `json:"encryption"`
	Sanitization    *SanitizationConfig `json:"sanitization"`
	Audit           *AuditConfig        `json:"audit"`
	RateLimit       *RateLimitConfig    `json:"rate_limit"`
	SecurityHeaders *HeadersConfig      `json:"security_headers"`
	CORS            *CORSConfig         `json:"cors"`
	CSRF            *CSRFConfig         `json:"csrf"`
}

// HeadersConfig holds security headers configuration
type HeadersConfig struct {
	Enabled                   bool              `json:"enabled"`
	StrictTransportSecurity   string            `json:"strict_transport_security"`
	ContentSecurityPolicy     string            `json:"content_security_policy"`
	XContentTypeOptions       string            `json:"x_content_type_options"`
	XFrameOptions             string            `json:"x_frame_options"`
	XXSSProtection            string            `json:"x_xss_protection"`
	ReferrerPolicy            string            `json:"referrer_policy"`
	PermissionsPolicy         string            `json:"permissions_policy"`
	CrossOriginEmbedderPolicy string            `json:"cross_origin_embedder_policy"`
	CrossOriginOpenerPolicy   string            `json:"cross_origin_opener_policy"`
	CrossOriginResourcePolicy string            `json:"cross_origin_resource_policy"`
	CustomHeaders             map[string]string `json:"custom_headers"`
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	Enabled          bool     `json:"enabled"`
	AllowedOrigins   []string `json:"allowed_origins"`
	AllowedMethods   []string `json:"allowed_methods"`
	AllowedHeaders   []string `json:"allowed_headers"`
	ExposedHeaders   []string `json:"exposed_headers"`
	AllowCredentials bool     `json:"allow_credentials"`
	MaxAge           int      `json:"max_age"`
}

// CSRFConfig holds CSRF protection configuration
type CSRFConfig struct {
	Enabled        bool     `json:"enabled"`
	TokenLength    int      `json:"token_length"`
	TokenHeader    string   `json:"token_header"`
	TokenCookie    string   `json:"token_cookie"`
	SafeMethods    []string `json:"safe_methods"`
	TrustedOrigins []string `json:"trusted_origins"`
}

// RequestContext holds request-specific security context
type RequestContext struct {
	RequestID     string                 `json:"request_id"`
	SessionID     string                 `json:"session_id,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	User          *User                  `json:"user,omitempty"`
	ClientIP      string                 `json:"client_ip"`
	UserAgent     string                 `json:"user_agent"`
	Method        string                 `json:"method"`
	Path          string                 `json:"path"`
	StartTime     time.Time              `json:"start_time"`
	SecurityFlags map[string]bool        `json:"security_flags,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// NewSecurityMiddleware creates a new security middleware
func NewSecurityMiddleware(config *SecurityConfig, logger *logging.StructuredLogger) (*SecurityMiddleware, error) {
	sm := &SecurityMiddleware{
		config: config,
		logger: logger,
	}

	// Initialize authentication manager
	if config.Authentication != nil && config.Authentication.Enabled {
		authManager, err := NewAuthManager(config.Authentication, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize auth manager: %w", err)
		}
		sm.authManager = authManager
	}

	// Initialize mTLS manager
	if config.MTLS != nil && config.MTLS.Enabled {
		mtlsManager, err := NewMTLSManager(config.MTLS, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize mTLS manager: %w", err)
		}
		sm.mtlsManager = mtlsManager
	}

	// Initialize encryption manager
	if config.Encryption != nil && config.Encryption.Enabled {
		encryptionManager, err := NewEncryptionManager(config.Encryption, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize encryption manager: %w", err)
		}
		sm.encryptionManager = encryptionManager
	}

	// Initialize sanitization manager
	if config.Sanitization != nil && config.Sanitization.Enabled {
		sanitizer, err := NewSanitizationManager(config.Sanitization, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize sanitization manager: %w", err)
		}
		sm.sanitizer = sanitizer
	}

	// Initialize audit logger
	if config.Audit != nil && config.Audit.Enabled {
		auditLogger, err := NewAuditLogger(config.Audit, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize audit logger: %w", err)
		}
		sm.auditLogger = auditLogger
	}

	// Initialize rate limiter
	if config.RateLimit != nil && config.RateLimit.Enabled {
		rateLimiter, err := NewRateLimiter(config.RateLimit, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize rate limiter: %w", err)
		}
		sm.rateLimiter = rateLimiter
	}

	return sm, nil
}

// CreateMiddlewareChain creates the complete security middleware chain
func (sm *SecurityMiddleware) CreateMiddlewareChain() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Build middleware chain in reverse order (innermost first)
		handler := next

		// Apply response security headers
		handler = sm.securityHeadersMiddleware(handler)

		// Apply CORS if enabled
		if sm.config.CORS != nil && sm.config.CORS.Enabled {
			handler = sm.corsMiddleware(handler)
		}

		// Apply CSRF protection if enabled
		if sm.config.CSRF != nil && sm.config.CSRF.Enabled {
			handler = sm.csrfMiddleware(handler)
		}

		// Apply audit logging
		if sm.auditLogger != nil {
			handler = sm.auditMiddleware(handler)
		}

		// Apply authorization
		if sm.authManager != nil {
			handler = sm.authorizationMiddleware(handler)
		}

		// Apply authentication
		if sm.authManager != nil {
			handler = sm.authenticationMiddleware(handler)
		}

		// Apply rate limiting
		if sm.rateLimiter != nil {
			handler = sm.rateLimitMiddleware(handler)
		}

		// Apply input sanitization
		if sm.sanitizer != nil {
			handler = sm.sanitizationMiddleware(handler)
		}

		// Apply request context initialization (outermost)
		handler = sm.requestContextMiddleware(handler)

		return handler
	}
}

// requestContextMiddleware initializes request context
func (sm *SecurityMiddleware) requestContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create request context
		reqCtx := &RequestContext{
			RequestID:     uuid.New().String(),
			ClientIP:      sm.getClientIP(r),
			UserAgent:     r.UserAgent(),
			Method:        r.Method,
			Path:          r.URL.Path,
			StartTime:     time.Now(),
			SecurityFlags: make(map[string]bool),
			Metadata:      make(map[string]interface{}),
		}

		// Extract correlation ID if present
		if corrID := r.Header.Get("X-Correlation-ID"); corrID != "" {
			reqCtx.CorrelationID = corrID
		}

		// Add request context to context
		ctx := context.WithValue(r.Context(), "request_context", reqCtx)
		ctx = context.WithValue(ctx, "request_id", reqCtx.RequestID)

		// Add request ID to response header
		w.Header().Set("X-Request-ID", reqCtx.RequestID)

		// Continue with request
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// authenticationMiddleware handles authentication
func (sm *SecurityMiddleware) authenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx := sm.getRequestContext(ctx)

		// Perform authentication
		authResult, err := sm.authManager.Authenticate(ctx, r)
		if err != nil {
			sm.logger.Error("authentication failed",
				slog.String("request_id", reqCtx.RequestID),
				slog.String("path", r.URL.Path),
				slog.Error(err))

			// Log authentication failure
			if sm.auditLogger != nil {
				sm.auditLogger.LogAuthenticationEvent(ctx, "", false, err.Error())
			}

			sm.sendErrorResponse(w, http.StatusUnauthorized, "Authentication failed")
			return
		}

		// Add user to context
		if authResult.User != nil {
			ctx = context.WithValue(ctx, "user", authResult.User)
			ctx = context.WithValue(ctx, "user_id", authResult.User.ID)
			reqCtx.User = authResult.User
		}

		// Add token claims to context if present
		if authResult.Claims != nil {
			ctx = context.WithValue(ctx, "claims", authResult.Claims)
		}

		// Log successful authentication
		if sm.auditLogger != nil && authResult.User != nil {
			sm.auditLogger.LogAuthenticationEvent(ctx, authResult.User.Username, true, "")
		}

		// Continue with authenticated request
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// authorizationMiddleware handles authorization
func (sm *SecurityMiddleware) authorizationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx := sm.getRequestContext(ctx)

		// Get user from context
		user, ok := ctx.Value("user").(*User)
		if !ok || user == nil {
			sm.sendErrorResponse(w, http.StatusUnauthorized, "User not authenticated")
			return
		}

		// Determine resource and action
		resource := r.URL.Path
		action := r.Method

		// Check authorization
		allowed, err := sm.authManager.Authorize(ctx, user, resource, action)
		if err != nil {
			sm.logger.Error("authorization check failed",
				slog.String("request_id", reqCtx.RequestID),
				slog.String("user_id", user.ID),
				slog.String("resource", resource),
				slog.String("action", action),
				slog.Error(err))

			sm.sendErrorResponse(w, http.StatusInternalServerError, "Authorization check failed")
			return
		}

		if !allowed {
			// Log authorization failure
			if sm.auditLogger != nil {
				sm.auditLogger.LogAuthorizationEvent(ctx, user, resource, action, false)
			}

			sm.sendErrorResponse(w, http.StatusForbidden, "Access denied")
			return
		}

		// Log successful authorization
		if sm.auditLogger != nil {
			sm.auditLogger.LogAuthorizationEvent(ctx, user, resource, action, true)
		}

		// Continue with authorized request
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// rateLimitMiddleware handles rate limiting
func (sm *SecurityMiddleware) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx := sm.getRequestContext(ctx)

		// Check rate limit
		result, err := sm.rateLimiter.CheckLimit(ctx, r)
		if err != nil {
			sm.logger.Error("rate limit check failed",
				slog.String("request_id", reqCtx.RequestID),
				slog.Error(err))

			sm.sendErrorResponse(w, http.StatusInternalServerError, "Rate limit check failed")
			return
		}

		// Set rate limit headers
		sm.rateLimiter.SetResponseHeaders(w, result)

		if !result.Allowed {
			// Log rate limit violation
			if sm.auditLogger != nil {
				event := &AuditEvent{
					EventType:    EventTypeRateLimitExceeded,
					Severity:     SeverityWarning,
					Result:       ResultDenied,
					ErrorMessage: result.Reason,
				}
				sm.auditLogger.LogEvent(ctx, event)
			}

			sm.sendErrorResponse(w, http.StatusTooManyRequests, "Rate limit exceeded")
			return
		}

		// Continue with request
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// sanitizationMiddleware handles input sanitization
func (sm *SecurityMiddleware) sanitizationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx := sm.getRequestContext(ctx)

		// Only sanitize for methods with body
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			// Parse request body
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				sm.logger.Error("failed to parse request body",
					slog.String("request_id", reqCtx.RequestID),
					slog.Error(err))

				sm.sendErrorResponse(w, http.StatusBadRequest, "Invalid request body")
				return
			}

			// Sanitize input
			result, err := sm.sanitizer.SanitizePolicyInput(ctx, body)
			if err != nil {
				sm.logger.Error("input sanitization failed",
					slog.String("request_id", reqCtx.RequestID),
					slog.Error(err))

				// Log security violation if malicious patterns detected
				if sm.auditLogger != nil && len(result.Violations) > 0 {
					violations := make(map[string]interface{})
					for _, v := range result.Violations {
						violations[v.Field] = v.Message
					}
					sm.auditLogger.LogSecurityViolation(ctx, "input_sanitization", violations)
				}

				sm.sendErrorResponse(w, http.StatusBadRequest, "Input validation failed")
				return
			}

			// Replace body with sanitized version
			if result.Modified {
				sanitizedBody, _ := json.Marshal(result.SanitizedValue)
				r.Body = io.NopCloser(strings.NewReader(string(sanitizedBody)))
				r.ContentLength = int64(len(sanitizedBody))
			}
		}

		// Continue with sanitized request
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// auditMiddleware handles audit logging
func (sm *SecurityMiddleware) auditMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqCtx := sm.getRequestContext(ctx)

		// Create response wrapper to capture response details
		rw := &responseWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Create audit event
		event := &AuditEvent{
			EventType:     sm.getEventType(r),
			Severity:      SeverityInfo,
			RequestID:     reqCtx.RequestID,
			SessionID:     reqCtx.SessionID,
			CorrelationID: reqCtx.CorrelationID,
			ClientIP:      reqCtx.ClientIP,
			UserAgent:     reqCtx.UserAgent,
			RequestMethod: r.Method,
			RequestPath:   r.URL.Path,
		}

		// Add actor information if available
		if reqCtx.User != nil {
			event.Actor = &Actor{
				Type:     ActorTypeUser,
				ID:       reqCtx.User.ID,
				Username: reqCtx.User.Username,
				Email:    reqCtx.User.Email,
				Roles:    reqCtx.User.Roles,
			}
		}

		// Process request
		next.ServeHTTP(rw, r)

		// Update event with response details
		event.ResponseStatus = rw.statusCode
		event.Duration = time.Since(reqCtx.StartTime)

		// Set result based on status code
		if rw.statusCode >= 200 && rw.statusCode < 300 {
			event.Result = ResultSuccess
		} else if rw.statusCode >= 400 && rw.statusCode < 500 {
			event.Result = ResultDenied
		} else {
			event.Result = ResultError
		}

		// Log audit event
		sm.auditLogger.LogEvent(ctx, event)
	})
}

// securityHeadersMiddleware adds security headers
func (sm *SecurityMiddleware) securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if sm.config.SecurityHeaders != nil && sm.config.SecurityHeaders.Enabled {
			headers := sm.config.SecurityHeaders

			// Add standard security headers
			if headers.StrictTransportSecurity != "" {
				w.Header().Set("Strict-Transport-Security", headers.StrictTransportSecurity)
			}
			if headers.ContentSecurityPolicy != "" {
				w.Header().Set("Content-Security-Policy", headers.ContentSecurityPolicy)
			}
			if headers.XContentTypeOptions != "" {
				w.Header().Set("X-Content-Type-Options", headers.XContentTypeOptions)
			}
			if headers.XFrameOptions != "" {
				w.Header().Set("X-Frame-Options", headers.XFrameOptions)
			}
			if headers.XXSSProtection != "" {
				w.Header().Set("X-XSS-Protection", headers.XXSSProtection)
			}
			if headers.ReferrerPolicy != "" {
				w.Header().Set("Referrer-Policy", headers.ReferrerPolicy)
			}
			if headers.PermissionsPolicy != "" {
				w.Header().Set("Permissions-Policy", headers.PermissionsPolicy)
			}
			if headers.CrossOriginEmbedderPolicy != "" {
				w.Header().Set("Cross-Origin-Embedder-Policy", headers.CrossOriginEmbedderPolicy)
			}
			if headers.CrossOriginOpenerPolicy != "" {
				w.Header().Set("Cross-Origin-Opener-Policy", headers.CrossOriginOpenerPolicy)
			}
			if headers.CrossOriginResourcePolicy != "" {
				w.Header().Set("Cross-Origin-Resource-Policy", headers.CrossOriginResourcePolicy)
			}

			// Add custom headers
			for name, value := range headers.CustomHeaders {
				w.Header().Set(name, value)
			}
		}

		next.ServeHTTP(w, r)
	})
}

// corsMiddleware handles CORS
func (sm *SecurityMiddleware) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		if sm.isOriginAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)

			if sm.config.CORS.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(sm.config.CORS.AllowedMethods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(sm.config.CORS.AllowedHeaders, ", "))
				w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", sm.config.CORS.MaxAge))
				w.WriteHeader(http.StatusNoContent)
				return
			}

			// Set exposed headers
			if len(sm.config.CORS.ExposedHeaders) > 0 {
				w.Header().Set("Access-Control-Expose-Headers", strings.Join(sm.config.CORS.ExposedHeaders, ", "))
			}
		}

		next.ServeHTTP(w, r)
	})
}

// csrfMiddleware handles CSRF protection
func (sm *SecurityMiddleware) csrfMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip CSRF check for safe methods
		for _, method := range sm.config.CSRF.SafeMethods {
			if r.Method == method {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Check CSRF token
		token := r.Header.Get(sm.config.CSRF.TokenHeader)
		if token == "" {
			// Try to get from cookie
			if cookie, err := r.Cookie(sm.config.CSRF.TokenCookie); err == nil {
				token = cookie.Value
			}
		}

		if token == "" {
			sm.sendErrorResponse(w, http.StatusForbidden, "CSRF token missing")
			return
		}

		// Validate CSRF token
		// This is a simplified implementation
		// In production, you would validate against stored tokens

		next.ServeHTTP(w, r)
	})
}

// Helper methods

func (sm *SecurityMiddleware) getRequestContext(ctx context.Context) *RequestContext {
	if reqCtx, ok := ctx.Value("request_context").(*RequestContext); ok {
		return reqCtx
	}
	return &RequestContext{}
}

func (sm *SecurityMiddleware) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

func (sm *SecurityMiddleware) getEventType(r *http.Request) EventType {
	// Determine event type based on path and method
	path := r.URL.Path
	method := r.Method

	if strings.Contains(path, "/policies") {
		switch method {
		case http.MethodPost:
			return EventTypePolicyCreate
		case http.MethodPut, http.MethodPatch:
			return EventTypePolicyUpdate
		case http.MethodDelete:
			return EventTypePolicyDelete
		default:
			return EventTypePolicyAccess
		}
	}

	return EventTypeDataAccess
}

func (sm *SecurityMiddleware) isOriginAllowed(origin string) bool {
	for _, allowed := range sm.config.CORS.AllowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

func (sm *SecurityMiddleware) sendErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    statusCode,
			"message": message,
		},
	}

	json.NewEncoder(w).Encode(response)
}

// responseWrapper wraps http.ResponseWriter to capture response details
type responseWrapper struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWrapper) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.ResponseWriter.WriteHeader(code)
		rw.written = true
	}
}

func (rw *responseWrapper) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// CreateA1SecurityMiddleware creates security middleware specifically for A1 service
func CreateA1SecurityMiddleware(config *a1.A1ServerConfig) (func(http.Handler) http.Handler, error) {
	// Convert A1 config to security config
	secConfig := &SecurityConfig{
		Authentication: &AuthConfig{
			Enabled: config.AuthenticationConfig != nil && config.AuthenticationConfig.Enabled,
			Type:    AuthType(config.AuthenticationConfig.Method),
			JWTConfig: &JWTConfig{
				Issuers:        config.AuthenticationConfig.AllowedIssuers,
				RequiredClaims: config.AuthenticationConfig.RequiredClaims,
			},
		},
		RateLimit: &RateLimitConfig{
			Enabled: config.RateLimitConfig != nil && config.RateLimitConfig.Enabled,
			DefaultLimits: &RateLimitPolicy{
				RequestsPerMin: config.RateLimitConfig.RequestsPerMin,
				BurstSize:      config.RateLimitConfig.BurstSize,
				WindowSize:     config.RateLimitConfig.WindowSize,
			},
		},
		SecurityHeaders: &HeadersConfig{
			Enabled:                 true,
			StrictTransportSecurity: "max-age=31536000; includeSubDomains",
			ContentSecurityPolicy:   "default-src 'self'",
			XContentTypeOptions:     "nosniff",
			XFrameOptions:           "DENY",
			XXSSProtection:          "1; mode=block",
			ReferrerPolicy:          "strict-origin-when-cross-origin",
		},
		CORS: &CORSConfig{
			Enabled:          true,
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Content-Type", "Authorization"},
			AllowCredentials: true,
			MaxAge:           3600,
		},
	}

	// Create security middleware
	middleware, err := NewSecurityMiddleware(secConfig, config.Logger)
	if err != nil {
		return nil, err
	}

	return middleware.CreateMiddlewareChain(), nil
}
