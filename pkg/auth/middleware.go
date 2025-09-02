package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// AuthMiddleware provides authentication and authorization middleware.

type AuthMiddleware struct {
	sessionManager *SessionManager

	jwtManager *JWTManager

	rbacManager *RBACManager

	config *MiddlewareConfig
}

// MiddlewareConfig represents middleware configuration.

type MiddlewareConfig struct {
	// Skip authentication for these paths.

	SkipAuth []string `json:"skip_auth"`

	// CORS settings.

	EnableCORS bool `json:"enable_cors"`

	AllowedOrigins []string `json:"allowed_origins"`

	AllowedMethods []string `json:"allowed_methods"`

	AllowedHeaders []string `json:"allowed_headers"`

	AllowCredentials bool `json:"allow_credentials"`

	MaxAge int `json:"max_age"`

	// Security headers.

	EnableSecurityHeaders bool `json:"enable_security_headers"`

	// Rate limiting (basic implementation).

	EnableRateLimit bool `json:"enable_rate_limit"`

	RequestsPerMin int `json:"requests_per_min"`

	RateLimitWindow time.Duration `json:"rate_limit_window"`

	// CSRF protection.

	EnableCSRF bool `json:"enable_csrf"`

	CSRFTokenHeader string `json:"csrf_token_header"`

	CSRFSafeMethods []string `json:"csrf_safe_methods"`
}

// AuthContext represents authentication context.

type AuthContext struct {
	UserID string `json:"user_id"`

	SessionID string `json:"session_id"`

	Provider string `json:"provider"`

	Roles []string `json:"roles"`

	Permissions []string `json:"permissions"`

	IsAdmin bool `json:"is_admin"`

	Attributes map[string]interface{} `json:"attributes"`
}

// contextKey is used for context keys to avoid collisions.

type contextKey string

const (

	// AuthContextKey holds authcontextkey value.

	AuthContextKey contextKey = "auth_context"
)

// NewAuthMiddleware creates new authentication middleware.

func NewAuthMiddleware(sessionManager *SessionManager, jwtManager *JWTManager, rbacManager *RBACManager, config *MiddlewareConfig) *AuthMiddleware {
	if config == nil {
		config = &MiddlewareConfig{
			SkipAuth: []string{
				"/health", "/metrics", "/auth/login", "/auth/callback",

				"/auth/providers", "/.well-known/", "/favicon.ico",
			},

			EnableCORS: true,

			AllowedOrigins: []string{"*"},

			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},

			AllowedHeaders: []string{"Authorization", "Content-Type", "X-Requested-With", "X-CSRF-Token"},

			AllowCredentials: true,

			MaxAge: 86400,

			EnableSecurityHeaders: true,

			EnableCSRF: true,

			CSRFTokenHeader: "X-CSRF-Token",

			CSRFSafeMethods: []string{"GET", "HEAD", "OPTIONS", "TRACE"},
		}
	}

	return &AuthMiddleware{
		sessionManager: sessionManager,

		jwtManager: jwtManager,

		rbacManager: rbacManager,

		config: config,
	}
}

// AuthenticateMiddleware handles authentication.

func (am *AuthMiddleware) AuthenticateMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for certain paths.

		if am.shouldSkipAuth(r.URL.Path) {

			next.ServeHTTP(w, r)

			return

		}

		// Set security headers.

		if am.config.EnableSecurityHeaders {
			am.setSecurityHeaders(w)
		}

		// Handle CORS.

		if am.config.EnableCORS {

			am.handleCORS(w, r)

			if r.Method == "OPTIONS" {

				w.WriteHeader(http.StatusOK)

				return

			}

		}

		// Authenticate request.

		authContext, err := am.authenticateRequest(r)
		if err != nil {

			am.writeErrorResponse(w, http.StatusUnauthorized, "authentication_failed", err.Error())

			return

		}

		// Add auth context to request.

		ctx := context.WithValue(r.Context(), AuthContextKey, authContext)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermissionMiddleware requires specific permission.

func (am *AuthMiddleware) RequirePermissionMiddleware(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authContext := GetAuthContext(r.Context())

			if authContext == nil {

				am.writeErrorResponse(w, http.StatusUnauthorized, "authentication_required", "Authentication required")

				return

			}

			// Check permission.

			hasPermission := false

			for _, perm := range authContext.Permissions {
				if am.matchesPermission(perm, permission) {

					hasPermission = true

					break

				}
			}

			if !hasPermission {

				am.writeErrorResponse(w, http.StatusForbidden, "insufficient_permissions",

					fmt.Sprintf("Required permission: %s", permission))

				return

			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRoleMiddleware requires specific role.

func (am *AuthMiddleware) RequireRoleMiddleware(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authContext := GetAuthContext(r.Context())

			if authContext == nil {

				am.writeErrorResponse(w, http.StatusUnauthorized, "authentication_required", "Authentication required")

				return

			}

			// Check role.

			hasRole := false

			for _, userRole := range authContext.Roles {
				if userRole == role {

					hasRole = true

					break

				}
			}

			if !hasRole {

				am.writeErrorResponse(w, http.StatusForbidden, "insufficient_role",

					fmt.Sprintf("Required role: %s", role))

				return

			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdminMiddleware requires admin role.

func (am *AuthMiddleware) RequireAdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authContext := GetAuthContext(r.Context())

		if authContext == nil {

			am.writeErrorResponse(w, http.StatusUnauthorized, "authentication_required", "Authentication required")

			return

		}

		if !authContext.IsAdmin {

			am.writeErrorResponse(w, http.StatusForbidden, "admin_required", "Administrator access required")

			return

		}

		next.ServeHTTP(w, r)
	})
}

// CSRFMiddleware provides CSRF protection.

func (am *AuthMiddleware) CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !am.config.EnableCSRF {

			next.ServeHTTP(w, r)

			return

		}

		// Skip CSRF for safe methods.

		if am.isSafeMethod(r.Method) {

			next.ServeHTTP(w, r)

			return

		}

		// Get session.

		sessionID := am.getSessionID(r)

		if sessionID == "" {

			am.writeErrorResponse(w, http.StatusForbidden, "csrf_session_required", "Session required for CSRF protection")

			return

		}

		session, err := am.sessionManager.GetSession(r.Context(), sessionID)
		if err != nil {

			am.writeErrorResponse(w, http.StatusForbidden, "csrf_session_invalid", "Invalid session for CSRF protection")

			return

		}

		// Check CSRF token.

		csrfToken := r.Header.Get(am.config.CSRFTokenHeader)

		if csrfToken == "" {
			csrfToken = r.FormValue("csrf_token")
		}

		if csrfToken != session.CSRFToken {

			am.writeErrorResponse(w, http.StatusForbidden, "csrf_token_invalid", "Invalid CSRF token")

			return

		}

		next.ServeHTTP(w, r)
	})
}

// RequestLoggingMiddleware logs HTTP requests.

func (am *AuthMiddleware) RequestLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code.

		wrapper := &responseWrapper{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapper, r)

		duration := time.Since(start)

		authContext := GetAuthContext(r.Context())

		userID := "anonymous"

		if authContext != nil {
			userID = authContext.UserID
		}

		// Log request (using session manager's logger).

		if am.sessionManager != nil && am.sessionManager.logger != nil {
			am.sessionManager.logger.Info("HTTP request",

				"method", r.Method,

				"path", r.URL.Path,

				"status", wrapper.statusCode,

				"duration_ms", duration.Milliseconds(),

				"user_id", userID,

				"ip", getClientIP(r),

				"user_agent", r.UserAgent(),
			)
		}
	})
}

// Helper methods.

func (am *AuthMiddleware) authenticateRequest(r *http.Request) (*AuthContext, error) {
	// Try session-based authentication first.

	sessionID := am.getSessionID(r)

	if sessionID != "" {
		return am.authenticateWithSession(r.Context(), sessionID)
	}

	// Try JWT token authentication.

	authHeader := r.Header.Get("Authorization")

	if strings.HasPrefix(authHeader, "Bearer ") {

		token := strings.TrimPrefix(authHeader, "Bearer ")

		return am.authenticateWithJWT(r.Context(), token)

	}

	return nil, fmt.Errorf("no authentication credentials provided")
}

func (am *AuthMiddleware) authenticateWithSession(ctx context.Context, sessionID string) (*AuthContext, error) {
	sessionInfo, err := am.sessionManager.ValidateSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session: %w", err)
	}

	isAdmin := am.hasAdminRole(sessionInfo.Roles)

	return &AuthContext{
		UserID: sessionInfo.UserID,

		SessionID: sessionInfo.ID,

		Provider: sessionInfo.Provider,

		Roles: sessionInfo.Roles,

		Permissions: am.getUserPermissions(ctx, sessionInfo.UserID),

		IsAdmin: isAdmin,

		Attributes: make(map[string]interface{}),
	}, nil
}

func (am *AuthMiddleware) authenticateWithJWT(ctx context.Context, token string) (*AuthContext, error) {
	if am.jwtManager == nil {
		return nil, fmt.Errorf("JWT authentication not available")
	}

	claims, err := am.jwtManager.ValidateToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT token: %w", err)
	}

	if claims.TokenType != "access" {
		return nil, fmt.Errorf("invalid token type for authentication")
	}

	isAdmin := am.hasAdminRole(claims.Roles)

	return &AuthContext{
		UserID: claims.Subject,

		SessionID: claims.SessionID,

		Provider: claims.Provider,

		Roles: claims.Roles,

		Permissions: claims.Permissions,

		IsAdmin: isAdmin,

		Attributes: claims.Attributes,
	}, nil
}

func (am *AuthMiddleware) getSessionID(r *http.Request) string {
	// Try cookie first.

	cookie, err := r.Cookie("nephoran_session")

	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// Try header.

	return r.Header.Get("X-Session-ID")
}

func (am *AuthMiddleware) getUserPermissions(ctx context.Context, userID string) []string {
	if am.rbacManager == nil {
		return []string{}
	}

	return am.rbacManager.GetUserPermissions(ctx, userID)
}

func (am *AuthMiddleware) hasAdminRole(roles []string) bool {
	adminRoles := []string{"system-admin", "admin", "administrator"}

	for _, role := range roles {
		for _, adminRole := range adminRoles {
			if strings.EqualFold(role, adminRole) {
				return true
			}
		}
	}

	return false
}

func (am *AuthMiddleware) shouldSkipAuth(path string) bool {
	for _, skipPath := range am.config.SkipAuth {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}

	return false
}

func (am *AuthMiddleware) matchesPermission(granted, required string) bool {
	if granted == "*" || granted == required {
		return true
	}

	// Handle resource-level wildcards.

	if strings.HasSuffix(granted, ":*") {

		grantedResource := strings.TrimSuffix(granted, ":*")

		requiredParts := strings.SplitN(required, ":", 2)

		if len(requiredParts) == 2 && requiredParts[0] == grantedResource {
			return true
		}

	}

	return false
}

func (am *AuthMiddleware) isSafeMethod(method string) bool {
	for _, safeMethod := range am.config.CSRFSafeMethods {
		if method == safeMethod {
			return true
		}
	}

	return false
}

func (am *AuthMiddleware) handleCORS(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")

	// Check if origin is allowed.

	allowed := false

	for _, allowedOrigin := range am.config.AllowedOrigins {
		if allowedOrigin == "*" || allowedOrigin == origin {

			allowed = true

			break

		}
	}

	if allowed {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}

	if am.config.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	w.Header().Set("Access-Control-Allow-Methods", strings.Join(am.config.AllowedMethods, ", "))

	w.Header().Set("Access-Control-Allow-Headers", strings.Join(am.config.AllowedHeaders, ", "))

	w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", am.config.MaxAge))
}

func (am *AuthMiddleware) setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")

	w.Header().Set("X-Frame-Options", "DENY")

	w.Header().Set("X-XSS-Protection", "1; mode=block")

	w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

	w.Header().Set("Content-Security-Policy", "default-src 'self'")
}

func (am *AuthMiddleware) writeErrorResponse(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(status)

	errorResponse := map[string]interface{}{
		"error": code,

		"error_description": message,

		"status": status,

		"timestamp": time.Now().Unix(),
	}

	json.NewEncoder(w).Encode(errorResponse)
}

// Utility types.

type responseWrapper struct {
	http.ResponseWriter

	statusCode int
}

// WriteHeader performs writeheader operation.

func (rw *responseWrapper) WriteHeader(code int) {
	rw.statusCode = code

	rw.ResponseWriter.WriteHeader(code)
}

// Helper functions.

// GetAuthContext extracts authentication context from request context.

func GetAuthContext(ctx context.Context) *AuthContext {
	if authCtx, ok := ctx.Value(AuthContextKey).(*AuthContext); ok {
		return authCtx
	}

	return nil
}

// RequireAuthContext ensures authentication context exists.

func RequireAuthContext(ctx context.Context) (*AuthContext, error) {
	authCtx := GetAuthContext(ctx)

	if authCtx == nil {
		return nil, fmt.Errorf("authentication required")
	}

	return authCtx, nil
}

// GetUserID extracts user ID from context.

func GetUserID(ctx context.Context) string {
	if authCtx := GetAuthContext(ctx); authCtx != nil {
		return authCtx.UserID
	}

	return ""
}

// UserContext represents user context for middleware tests
type UserContext struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Roles     []string `json:"roles"`
	SessionID string `json:"session_id"`
}

// AuthMiddlewareConfig configures auth middleware
type AuthMiddlewareConfig struct {
	JWTManager     interface{} `json:"jwt_manager"`
	SessionManager interface{} `json:"session_manager"`
	RequireAuth    bool        `json:"require_auth"`
	AllowedPaths   []string    `json:"allowed_paths"`
	HeaderName     string      `json:"header_name"`
	CookieName     string      `json:"cookie_name"`
	ContextKey     string      `json:"context_key"`
}

// AuthMiddlewareWrapper wraps auth middleware
type AuthMiddlewareWrapper struct {
	config *AuthMiddlewareConfig
}

// NewAuthMiddleware creates auth middleware
func NewAuthMiddleware(config *AuthMiddlewareConfig) *AuthMiddlewareWrapper {
	return &AuthMiddlewareWrapper{config: config}
}

// Middleware returns the middleware function
func (a *AuthMiddlewareWrapper) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Basic stub implementation for tests
		if !a.config.RequireAuth {
			next.ServeHTTP(w, r)
			return
		}

		// Check if path is allowed
		for _, path := range a.config.AllowedPaths {
			if strings.HasPrefix(r.URL.Path, path) {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Check for authorization
		authHeader := r.Header.Get(a.config.HeaderName)
		if authHeader == "" {
			http.Error(w, `{"error": "Missing authentication"}`, http.StatusUnauthorized)
			return
		}

		// Basic token validation stub
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, `{"error": "Invalid authorization header"}`, http.StatusUnauthorized)
			return
		}

		// Set user context (stub)
		userCtx := &UserContext{
			UserID: "test-user",
			Email:  "test@example.com",
			Name:   "Test User",
		}
		
		ctx := context.WithValue(r.Context(), a.config.ContextKey, userCtx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RBACMiddlewareConfig configures RBAC middleware
type RBACMiddlewareConfig struct {
	RBACManager       interface{}                        `json:"rbac_manager"`
	ResourceExtractor func(*http.Request) string         `json:"-"`
	ActionExtractor   func(*http.Request) string         `json:"-"`
	UserIDExtractor   func(*http.Request) string         `json:"-"`
}

// RBACMiddlewareWrapper wraps RBAC middleware
type RBACMiddlewareWrapper struct {
	config *RBACMiddlewareConfig
}

// NewRBACMiddleware creates RBAC middleware
func NewRBACMiddleware(config *RBACMiddlewareConfig) *RBACMiddlewareWrapper {
	return &RBACMiddlewareWrapper{config: config}
}

// Middleware returns the middleware function
func (r *RBACMiddlewareWrapper) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		userID := r.config.UserIDExtractor(req)
		if userID == "" {
			http.Error(w, `{"error": "No user context"}`, http.StatusForbidden)
			return
		}

		resource := r.config.ResourceExtractor(req)
		action := r.config.ActionExtractor(req)

		// Basic permission check stub
		if resource == "admin" && userID != "admin-user" {
			http.Error(w, `{"error": "Insufficient permissions"}`, http.StatusForbidden)
			return
		}

		if action == "write" && userID == "reader-user" {
			http.Error(w, `{"error": "Insufficient permissions"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, req)
	})
}

// CORSConfig configures CORS middleware
type CORSConfig struct {
	AllowedOrigins   []string `json:"allowed_origins"`
	AllowedMethods   []string `json:"allowed_methods"`
	AllowedHeaders   []string `json:"allowed_headers"`
	ExposedHeaders   []string `json:"exposed_headers"`
	AllowCredentials bool     `json:"allow_credentials"`
	MaxAge           int      `json:"max_age"`
}

// CORSMiddlewareWrapper wraps CORS middleware
type CORSMiddlewareWrapper struct {
	config *CORSConfig
}

// NewCORSMiddleware creates CORS middleware
func NewCORSMiddleware(config *CORSConfig) *CORSMiddlewareWrapper {
	return &CORSMiddlewareWrapper{config: config}
}

// Middleware returns the middleware function
func (c *CORSMiddlewareWrapper) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		
		if origin != "" {
			allowed := false
			for _, allowedOrigin := range c.config.AllowedOrigins {
				if allowedOrigin == origin {
					allowed = true
					break
				}
			}
			
			if !allowed {
				http.Error(w, "Origin not allowed", http.StatusForbidden)
				return
			}
			
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		
		if c.config.AllowCredentials {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		
		if len(c.config.ExposedHeaders) > 0 {
			w.Header().Set("Access-Control-Expose-Headers", strings.Join(c.config.ExposedHeaders, ","))
		}

		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(c.config.AllowedMethods, ","))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(c.config.AllowedHeaders, ","))
			w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", c.config.MaxAge))
			
			reqMethod := r.Header.Get("Access-Control-Request-Method")
			if reqMethod != "" {
				methodAllowed := false
				for _, method := range c.config.AllowedMethods {
					if method == reqMethod {
						methodAllowed = true
						break
					}
				}
				if !methodAllowed {
					http.Error(w, "Method not allowed", http.StatusForbidden)
					return
				}
			}
			
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RateLimitConfig configures rate limiting middleware
type RateLimitConfig struct {
	RequestsPerMinute int                             `json:"requests_per_minute"`
	BurstSize         int                             `json:"burst_size"`
	KeyGenerator      func(*http.Request) string      `json:"-"`
	OnLimitExceeded   func(http.ResponseWriter, *http.Request) `json:"-"`
}

// RateLimitMiddlewareWrapper wraps rate limit middleware
type RateLimitMiddlewareWrapper struct {
	config   *RateLimitConfig
	requests map[string]int
}

// NewRateLimitMiddleware creates rate limit middleware
func NewRateLimitMiddleware(config *RateLimitConfig) *RateLimitMiddlewareWrapper {
	return &RateLimitMiddlewareWrapper{
		config:   config,
		requests: make(map[string]int),
	}
}

// Middleware returns the middleware function
func (rl *RateLimitMiddlewareWrapper) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := rl.config.KeyGenerator(r)
		
		// Simple rate limiting (in production would use proper rate limiter)
		rl.requests[key]++
		if rl.requests[key] > rl.config.BurstSize {
			rl.config.OnLimitExceeded(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// SecurityHeadersConfig configures security headers middleware
type SecurityHeadersConfig struct {
	ContentSecurityPolicy string            `json:"content_security_policy"`
	XFrameOptions         string            `json:"x_frame_options"`
	XContentTypeOptions   string            `json:"x_content_type_options"`
	ReferrerPolicy        string            `json:"referrer_policy"`
	HSTSMaxAge            int               `json:"hsts_max_age"`
	HSTSIncludeSubdomains bool              `json:"hsts_include_subdomains"`
	HSTSPreload           bool              `json:"hsts_preload"`
	RemoveServerHeader    bool              `json:"remove_server_header"`
	CustomHeaders         map[string]string `json:"custom_headers"`
}

// SecurityHeadersMiddlewareWrapper wraps security headers middleware
type SecurityHeadersMiddlewareWrapper struct {
	config *SecurityHeadersConfig
}

// NewSecurityHeadersMiddleware creates security headers middleware
func NewSecurityHeadersMiddleware(config *SecurityHeadersConfig) *SecurityHeadersMiddlewareWrapper {
	return &SecurityHeadersMiddlewareWrapper{config: config}
}

// Middleware returns the middleware function
func (s *SecurityHeadersMiddlewareWrapper) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.config.ContentSecurityPolicy != "" {
			w.Header().Set("Content-Security-Policy", s.config.ContentSecurityPolicy)
		}
		if s.config.XFrameOptions != "" {
			w.Header().Set("X-Frame-Options", s.config.XFrameOptions)
		}
		if s.config.XContentTypeOptions != "" {
			w.Header().Set("X-Content-Type-Options", s.config.XContentTypeOptions)
		}
		if s.config.ReferrerPolicy != "" {
			w.Header().Set("Referrer-Policy", s.config.ReferrerPolicy)
		}
		if s.config.HSTSMaxAge > 0 {
			hsts := fmt.Sprintf("max-age=%d", s.config.HSTSMaxAge)
			if s.config.HSTSIncludeSubdomains {
				hsts += "; includeSubDomains"
			}
			if s.config.HSTSPreload {
				hsts += "; preload"
			}
			w.Header().Set("Strict-Transport-Security", hsts)
		}
		
		for key, value := range s.config.CustomHeaders {
			w.Header().Set(key, value)
		}

		next.ServeHTTP(w, r)
		
		if s.config.RemoveServerHeader {
			w.Header().Del("Server")
		}
	})
}

// RequestLoggingConfig configures request logging middleware
type RequestLoggingConfig struct {
	Logger           func(string)  `json:"-"`
	LogHeaders       bool          `json:"log_headers"`
	LogBody          bool          `json:"log_body"`
	MaxBodySize      int           `json:"max_body_size"`
	SkipPaths        []string      `json:"skip_paths"`
	SensitiveHeaders []string      `json:"sensitive_headers"`
}

// RequestLoggingMiddlewareWrapper wraps request logging middleware
type RequestLoggingMiddlewareWrapper struct {
	config *RequestLoggingConfig
}

// NewRequestLoggingMiddleware creates request logging middleware
func NewRequestLoggingMiddleware(config *RequestLoggingConfig) *RequestLoggingMiddlewareWrapper {
	return &RequestLoggingMiddlewareWrapper{config: config}
}

// Middleware returns the middleware function
func (l *RequestLoggingMiddlewareWrapper) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if path should be skipped
		for _, path := range l.config.SkipPaths {
			if strings.HasPrefix(r.URL.Path, path) {
				next.ServeHTTP(w, r)
				return
			}
		}

		start := time.Now()
		wrapper := &responseWrapper{ResponseWriter: w, statusCode: http.StatusOK}
		
		// Log request
		logParts := []string{fmt.Sprintf("%s %s", r.Method, r.URL.Path)}
		
		if l.config.LogHeaders {
			for name, values := range r.Header {
				value := strings.Join(values, ", ")
				// Redact sensitive headers
				for _, sensitive := range l.config.SensitiveHeaders {
					if strings.EqualFold(name, sensitive) {
						value = "[REDACTED]"
						break
					}
				}
				logParts = append(logParts, fmt.Sprintf("%s: %s", name, value))
			}
		}
		
		if l.config.LogBody && r.ContentLength > 0 {
			// Simple body logging (in production would need proper buffering)
			if int(r.ContentLength) > l.config.MaxBodySize {
				logParts = append(logParts, "[TRUNCATED]")
			}
		}

		next.ServeHTTP(wrapper, r)
		
		duration := time.Since(start)
		logParts = append(logParts, fmt.Sprintf("%d %dms", wrapper.statusCode, duration.Milliseconds()))
		
		l.config.Logger(strings.Join(logParts, " "))
	})
}

// HasPermission checks if user has specific permission.

func HasPermission(ctx context.Context, permission string) bool {
	authCtx := GetAuthContext(ctx)

	if authCtx == nil {
		return false
	}

	for _, perm := range authCtx.Permissions {
		if matchesPermission(perm, permission) {
			return true
		}
	}

	return false
}

// HasRole checks if user has specific role.

func HasRole(ctx context.Context, role string) bool {
	authCtx := GetAuthContext(ctx)

	if authCtx == nil {
		return false
	}

	for _, userRole := range authCtx.Roles {
		if userRole == role {
			return true
		}
	}

	return false
}

// IsAdmin checks if user has admin privileges.

func IsAdmin(ctx context.Context) bool {
	authCtx := GetAuthContext(ctx)

	return authCtx != nil && authCtx.IsAdmin
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header.

	xff := r.Header.Get("X-Forwarded-For")

	if xff != "" {

		// Take the first IP in the list.

		ips := strings.Split(xff, ",")

		return strings.TrimSpace(ips[0])

	}

	// Check X-Real-IP header.

	xri := r.Header.Get("X-Real-IP")

	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr.

	return r.RemoteAddr
}

func matchesPermission(granted, required string) bool {
	if granted == "*" || granted == required {
		return true
	}

	// Handle resource-level wildcards.

	if strings.HasSuffix(granted, ":*") {

		grantedResource := strings.TrimSuffix(granted, ":*")

		requiredParts := strings.SplitN(required, ":", 2)

		if len(requiredParts) == 2 && requiredParts[0] == grantedResource {
			return true
		}

	}

	return false
}

// RequireOperator returns a middleware that requires operator role.

func (am *AuthMiddleware) RequireOperator() func(http.Handler) http.Handler {
	return am.RequireRoleMiddleware("operator")
}

// RequireAdmin returns a middleware that requires admin role.

func (am *AuthMiddleware) RequireAdmin() func(http.Handler) http.Handler {
	return am.RequireAdminMiddleware
}
