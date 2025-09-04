package security

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/golang-jwt/jwt/v5"
)

// SecurityMiddleware provides A1 interface security middleware.

type SecurityMiddleware struct {
	config *SecurityConfig

	tokenValidator *TokenValidator

	rbacManager *RBACManagerService

	auditLogger *AuditLogger

	rateLimiter interface{}  // Use interface{} to avoid type conversion issues

	metrics *MiddlewareMetrics

	mu sync.RWMutex
}

// TokenValidator validates JWT tokens.

type TokenValidator struct {
	signingKey []byte

	signingMethod jwt.SigningMethod

	issuer string

	audience string
}

// RBACManagerService wraps RBAC functionality for the middleware
type RBACManagerService struct {
	policies map[string]*RBACPolicy

	roles map[string]*Role

	users map[string]*User

	mu sync.RWMutex
}

// RBACPolicy represents a simple RBAC policy for middleware
type RBACPolicy struct {
	Name string `json:"name"`

	Rules []RBACRule `json:"rules"`
}

// RBACRule represents a simple RBAC rule for middleware
type RBACRule struct {
	Resource string `json:"resource"`

	Action string `json:"action"`

	Effect string `json:"effect"`

	Conditions map[string]interface{} `json:"conditions,omitempty"`
}

// MiddlewareMetrics tracks security metrics.

type MiddlewareMetrics struct {
	totalRequests int64

	authorizedRequests int64

	unauthorizedRequests int64

	rateLimitedRequests int64

	tokenValidationFailures int64

	mu sync.RWMutex
}

// ErrorResponse represents an API error response.
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// HeadersConfig defines security headers configuration.
type HeadersConfig struct {
	EnableHSTS       bool `json:"enable_hsts"`
	EnableCSP        bool `json:"enable_csp"`
	EnableXSSProtect bool `json:"enable_xss_protect"`
	EnableNoSniff    bool `json:"enable_nosniff"`
	EnableFrameDeny  bool `json:"enable_frame_deny"`
}

// CORSConfigLocal defines CORS configuration (local version to avoid conflicts).
type CORSConfigLocal struct {
	Enabled          bool     `json:"enabled"`
	AllowedOrigins   []string `json:"allowed_origins"`
	AllowedMethods   []string `json:"allowed_methods"`
	AllowedHeaders   []string `json:"allowed_headers"`
	ExposedHeaders   []string `json:"exposed_headers"`
	AllowCredentials bool     `json:"allow_credentials"`
	MaxAge           int      `json:"max_age"`
}

// CSRFConfig defines CSRF protection configuration.
type CSRFConfig struct {
	Enabled    bool   `json:"enabled"`
	TokenName  string `json:"token_name"`
	CookieName string `json:"cookie_name"`
	SecretKey  string `json:"secret_key"`
}

// SimpleRateLimiter is a minimal rate limiter interface for middleware
type SimpleRateLimiter struct {
	enabled bool
}

// Allow returns true if request is allowed
func (s *SimpleRateLimiter) Allow(ctx context.Context, identifier string) bool {
	if !s.enabled {
		return true
	}
	// Simple implementation - always allow for now
	return true
}

// NewSecurityMiddleware creates a new security middleware using existing types.

func NewSecurityMiddleware(config *SecurityConfig) *SecurityMiddleware {
	// Initialize token validator if JWT config exists
	var validator *TokenValidator
	if config.Authentication != nil && config.Authentication.JWTConfig != nil {
		jwtConfig := config.Authentication.JWTConfig
		
		// Use the correct field names from JWTConfig struct
		var signingKey string
		var issuer string
		var audience string
		
		// Extract signing key from private key path or use direct key
		if jwtConfig.PrivateKeyPath != "" {
			// In a real implementation, read the key from file
			signingKey = "default-signing-key" // placeholder
		}
		
		// Use first issuer and audience if available
		if len(jwtConfig.Issuers) > 0 {
			issuer = jwtConfig.Issuers[0]
		}
		if len(jwtConfig.Audiences) > 0 {
			audience = jwtConfig.Audiences[0]
		}
		
		validator = &TokenValidator{
			signingKey: []byte(signingKey),
			issuer:     issuer,
			audience:   audience,
		}

		// Set signing method based on SigningAlgorithm
		switch jwtConfig.SigningAlgorithm {
		case "HS256":
			validator.signingMethod = jwt.SigningMethodHS256
		case "HS384":
			validator.signingMethod = jwt.SigningMethodHS384
		case "HS512":
			validator.signingMethod = jwt.SigningMethodHS512
		case "RS256":
			validator.signingMethod = jwt.SigningMethodRS256
		case "RS384":
			validator.signingMethod = jwt.SigningMethodRS384
		case "RS512":
			validator.signingMethod = jwt.SigningMethodRS512
		default:
			validator.signingMethod = jwt.SigningMethodHS256
		}
	}

	// Initialize RBAC manager
	rbacManager := &RBACManagerService{
		policies: make(map[string]*RBACPolicy),
		roles:    make(map[string]*Role),
		users:    make(map[string]*User),
	}

	// Initialize audit logger if config exists
	var auditLogger *AuditLogger
	if config.Audit != nil {
		if logger, err := NewAuditLogger(config.Audit); err == nil {
			auditLogger = logger
		}
	}

	// Initialize simple rate limiter if config exists
	var rateLimiter interface{}
	if config.RateLimit != nil {
		// For now, create a simple rate limiter without full dependency
		rateLimiter = &SimpleRateLimiter{
			enabled: config.RateLimit.Enabled,
		}
	}

	return &SecurityMiddleware{
		config:         config,
		tokenValidator: validator,
		rbacManager:    rbacManager,
		auditLogger:    auditLogger,
		rateLimiter:    rateLimiter,
		metrics:        &MiddlewareMetrics{},
	}
}

// Middleware returns the HTTP middleware handler.

func (sm *SecurityMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Increment total requests
		sm.metrics.mu.Lock()
		sm.metrics.totalRequests++
		sm.metrics.mu.Unlock()

		// Apply security headers
		if sm.config.SecurityHeaders != nil {
			sm.applySecurityHeaders(w, sm.config.SecurityHeaders)
		}

		// Apply CORS if enabled
		if sm.config.CORS != nil && sm.config.CORS.Enabled {
			if sm.handleCORS(w, r) {
				return
			}
		}

		// Apply CSRF protection if enabled
		if sm.config.CSRF != nil && sm.config.CSRF.Enabled {
			if !sm.validateCSRF(r) {
				sm.sendErrorResponse(w, http.StatusForbidden, "CSRF token validation failed")
				return
			}
		}

		// Apply rate limiting if enabled (simplified check)
		if sm.rateLimiter != nil && sm.config.RateLimit != nil && sm.config.RateLimit.Enabled {
			// Simple rate limiting check
			if !sm.checkRateLimit(ctx, getClientIdentifier(r)) {
				sm.metrics.mu.Lock()
				sm.metrics.rateLimitedRequests++
				sm.metrics.mu.Unlock()

				sm.sendErrorResponse(w, http.StatusTooManyRequests, "Rate limit exceeded")
				return
			}
		}

		// Validate authentication if enabled
		if sm.config.Authentication != nil && sm.config.Authentication.Enabled {
			userID, err := sm.validateAuthentication(ctx, r)
			if err != nil {
				sm.metrics.mu.Lock()
				sm.metrics.unauthorizedRequests++
				sm.metrics.mu.Unlock()

				// Log authentication failure
				if sm.auditLogger != nil {
					sm.auditLogger.LogAuthentication(ctx, "", ResultFailure, "jwt")
				}

				sm.sendErrorResponse(w, http.StatusUnauthorized, "Authentication failed")
				return
			}

			// Add user ID to context
			ctx = context.WithValue(ctx, "user_id", userID)
		}

		// Validate authorization if enabled
		if sm.rbacManager != nil {
			if !sm.validateAuthorization(ctx, r) {
				sm.metrics.mu.Lock()
				sm.metrics.unauthorizedRequests++
				sm.metrics.mu.Unlock()

				// Log authorization failure
				if sm.auditLogger != nil {
					userID := ctx.Value("user_id")
					if userID != nil {
						sm.auditLogger.LogAuthorization(ctx, userID.(string), &Resource{
							Type: "api_endpoint",
							Path: r.URL.Path,
						}, r.Method, ResultFailure)
					}
				}

				sm.sendErrorResponse(w, http.StatusForbidden, "Authorization failed")
				return
			}
		}

		// Increment authorized requests
		sm.metrics.mu.Lock()
		sm.metrics.authorizedRequests++
		sm.metrics.mu.Unlock()

		// Log successful access
		if sm.auditLogger != nil {
			userID := ctx.Value("user_id")
			if userID != nil {
				sm.auditLogger.LogDataAccess(ctx, userID.(string), &Resource{
					Type: "api_endpoint",
					Path: r.URL.Path,
				}, r.Method)
			}
		}

		// Continue to next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// checkRateLimit is a simple rate limit check
func (sm *SecurityMiddleware) checkRateLimit(ctx context.Context, identifier string) bool {
	// Simple implementation - always allow for now
	// In a real implementation, this would check rate limits
	return true
}

// validateAuthentication validates the request authentication.

func (sm *SecurityMiddleware) validateAuthentication(ctx context.Context, r *http.Request) (string, error) {
	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("missing authorization header")
	}

	// Check for Bearer token
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", fmt.Errorf("invalid authorization header format")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	// Validate JWT token if token validator is available
	if sm.tokenValidator != nil {
		return sm.validateJWTToken(token)
	}

	return "", fmt.Errorf("no token validator configured")
}

// validateJWTToken validates a JWT token.

func (sm *SecurityMiddleware) validateJWTToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Check the signing method
		if token.Method != sm.tokenValidator.signingMethod {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return sm.tokenValidator.signingKey, nil
	})

	if err != nil {
		return "", fmt.Errorf("token validation failed: %w", err)
	}

	if !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid token claims")
	}

	// Validate issuer
	if sm.tokenValidator.issuer != "" {
		if iss, ok := claims["iss"].(string); !ok || iss != sm.tokenValidator.issuer {
			return "", fmt.Errorf("invalid issuer")
		}
	}

	// Validate audience
	if sm.tokenValidator.audience != "" {
		if aud, ok := claims["aud"].(string); !ok || aud != sm.tokenValidator.audience {
			return "", fmt.Errorf("invalid audience")
		}
	}

	// Extract user ID
	if sub, ok := claims["sub"].(string); ok {
		return sub, nil
	}

	return "", fmt.Errorf("missing subject claim")
}

// validateAuthorization validates the request authorization.

func (sm *SecurityMiddleware) validateAuthorization(ctx context.Context, r *http.Request) bool {
	// This is a simplified RBAC implementation
	// In a real implementation, this would check user roles and permissions

	userID := ctx.Value("user_id")
	if userID == nil {
		return false
	}

	// For now, allow all authenticated users
	// TODO: Implement proper RBAC logic
	return true
}

// applySecurityHeaders applies security headers to the response.

func (sm *SecurityMiddleware) applySecurityHeaders(w http.ResponseWriter, config *HeadersConfig) {
	if config.EnableHSTS {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	}

	if config.EnableCSP {
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
	}

	if config.EnableXSSProtect {
		w.Header().Set("X-XSS-Protection", "1; mode=block")
	}

	if config.EnableNoSniff {
		w.Header().Set("X-Content-Type-Options", "nosniff")
	}

	if config.EnableFrameDeny {
		w.Header().Set("X-Frame-Options", "DENY")
	}
}

// handleCORS handles CORS preflight and actual requests.

func (sm *SecurityMiddleware) handleCORS(w http.ResponseWriter, r *http.Request) bool {
	corsConfig := sm.config.CORS

	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}

	// Check if origin is allowed
	allowed := false
	for _, allowedOrigin := range corsConfig.AllowedOrigins {
		if allowedOrigin == "*" || allowedOrigin == origin {
			allowed = true
			break
		}
	}

	if !allowed {
		return false
	}

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Credentials", strconv.FormatBool(corsConfig.AllowCredentials))

	if len(corsConfig.ExposedHeaders) > 0 {
		w.Header().Set("Access-Control-Expose-Headers", strings.Join(corsConfig.ExposedHeaders, ", "))
	}

	// Handle preflight request
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(corsConfig.AllowedMethods, ", "))
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(corsConfig.AllowedHeaders, ", "))
		w.Header().Set("Access-Control-Max-Age", strconv.Itoa(corsConfig.MaxAge))
		w.WriteHeader(http.StatusOK)
		return true
	}

	return false
}

// validateCSRF validates CSRF token.

func (sm *SecurityMiddleware) validateCSRF(r *http.Request) bool {
	// This is a simplified CSRF validation
	// In a real implementation, this would validate the CSRF token

	// For now, just check if the token is present
	token := r.Header.Get("X-CSRF-Token")
	if token == "" {
		token = r.FormValue("csrf_token")
	}

	return token != ""
}

// sendErrorResponse sends an error response.

func (sm *SecurityMiddleware) sendErrorResponse(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	response := ErrorResponse{
		Code:    code,
		Message: message,
	}

	json.NewEncoder(w).Encode(response)
}

// getClientIdentifier gets a client identifier for rate limiting.

func getClientIdentifier(r *http.Request) string {
	// Use client IP address as identifier
	clientIP := r.Header.Get("X-Real-IP")
	if clientIP == "" {
		clientIP = r.Header.Get("X-Forwarded-For")
	}
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}

	return clientIP
}

// GetMetrics returns security middleware metrics.

func (sm *SecurityMiddleware) GetMetrics() map[string]interface{} {
	sm.metrics.mu.RLock()
	defer sm.metrics.mu.RUnlock()

	return map[string]interface{}{
		"total_requests":              sm.metrics.totalRequests,
		"authorized_requests":         sm.metrics.authorizedRequests,
		"unauthorized_requests":       sm.metrics.unauthorizedRequests,
		"rate_limited_requests":       sm.metrics.rateLimitedRequests,
		"token_validation_failures":   sm.metrics.tokenValidationFailures,
	}
}

// Close shuts down the security middleware gracefully.

func (sm *SecurityMiddleware) Close() error {
	if sm.auditLogger != nil {
		return sm.auditLogger.Close()
	}
	return nil
}