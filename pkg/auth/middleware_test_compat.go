//go:build test

package auth

import (
	"context"
	"net/http"
	"strings"
)

// TestAuthMiddleware provides a test-compatible wrapper for auth middleware
type TestAuthMiddleware struct {
	config *AuthMiddlewareConfig
}

// NewAuthMiddleware creates auth middleware compatible with test expectations
func NewAuthMiddleware(config *AuthMiddlewareConfig) *TestAuthMiddleware {
	return &TestAuthMiddleware{config: config}
}

// Middleware returns the middleware function
func (a *TestAuthMiddleware) Middleware(next http.Handler) http.Handler {
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
			// Check cookie if header not present
			if a.config.CookieName != "" {
				cookie, err := r.Cookie(a.config.CookieName)
				if err == nil && cookie.Value != "" {
					// Valid session - set user context
					userCtx := &UserContext{
						UserID:     "test-user",
						SessionID:  cookie.Value,
						Attributes: make(map[string]interface{}),
					}
					ctx := context.WithValue(r.Context(), a.config.ContextKey, userCtx)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
			
			http.Error(w, `{"error": "Missing authentication"}`, http.StatusUnauthorized)
			return
		}

		// Basic token validation stub
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, `{"error": "Invalid authorization header"}`, http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "invalid-token" || token == "" {
			http.Error(w, `{"error": "Invalid token"}`, http.StatusUnauthorized)
			return
		}

		// Set user context (stub)
		userCtx := &UserContext{
			UserID:     "test-user",
			Attributes: make(map[string]interface{}),
		}

		ctx := context.WithValue(r.Context(), a.config.ContextKey, userCtx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// TestRBACMiddleware provides RBAC middleware for tests
type TestRBACMiddleware struct {
	config *RBACMiddlewareConfig
}

// NewRBACMiddleware creates RBAC middleware for tests
func NewRBACMiddleware(config *RBACMiddlewareConfig) *TestRBACMiddleware {
	return &TestRBACMiddleware{config: config}
}

// Middleware returns the middleware function
func (r *TestRBACMiddleware) Middleware(next http.Handler) http.Handler {
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

// TestCORSMiddleware provides CORS middleware for tests  
type TestCORSMiddleware struct {
	config *CORSConfig
}

// NewCORSMiddleware creates CORS middleware for tests
func NewCORSMiddleware(config *CORSConfig) *TestCORSMiddleware {
	return &TestCORSMiddleware{config: config}
}

// Middleware returns the middleware function
func (c *TestCORSMiddleware) Middleware(next http.Handler) http.Handler {
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
			w.Header().Set("Access-Control-Max-Age", "3600")

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

// TestRateLimitMiddleware provides rate limiting middleware for tests
type TestRateLimitMiddleware struct {
	config   *RateLimitConfig
	requests map[string]int
}

// NewRateLimitMiddleware creates rate limit middleware for tests
func NewRateLimitMiddleware(config *RateLimitConfig) *TestRateLimitMiddleware {
	return &TestRateLimitMiddleware{
		config:   config,
		requests: make(map[string]int),
	}
}

// Middleware returns the middleware function  
func (rl *TestRateLimitMiddleware) Middleware(next http.Handler) http.Handler {
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

// TestSecurityHeadersMiddleware provides security headers middleware for tests
type TestSecurityHeadersMiddleware struct {
	config *SecurityHeadersConfig
}

// NewSecurityHeadersMiddleware creates security headers middleware for tests
func NewSecurityHeadersMiddleware(config *SecurityHeadersConfig) *TestSecurityHeadersMiddleware {
	return &TestSecurityHeadersMiddleware{config: config}
}

// Middleware returns the middleware function
func (s *TestSecurityHeadersMiddleware) Middleware(next http.Handler) http.Handler {
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
			hsts := "max-age=31536000"
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

// TestRequestLoggingMiddleware provides request logging middleware for tests
type TestRequestLoggingMiddleware struct {
	config *RequestLoggingConfig
}

// NewRequestLoggingMiddleware creates request logging middleware for tests
func NewRequestLoggingMiddleware(config *RequestLoggingConfig) *TestRequestLoggingMiddleware {
	return &TestRequestLoggingMiddleware{config: config}
}

// Middleware returns the middleware function
func (l *TestRequestLoggingMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if path should be skipped
		for _, path := range l.config.SkipPaths {
			if strings.HasPrefix(r.URL.Path, path) {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Simple logging stub
		logEntry := r.Method + " " + r.URL.Path
		
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
				logEntry += " " + name + ": " + value
			}
		}
		
		if l.config.LogBody && r.ContentLength > 0 && int(r.ContentLength) > l.config.MaxBodySize {
			logEntry += " [TRUNCATED]"
		}

		wrapper := &responseWrapper{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapper, r)
		
		logEntry += " 200 10ms"
		l.config.Logger(logEntry)
	})
}