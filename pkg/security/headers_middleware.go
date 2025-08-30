package security

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// SecurityHeadersMiddleware adds comprehensive security headers to HTTP responses
type SecurityHeadersMiddleware struct {
	isDevelopment bool
	cspNonce      func() string
	hstsMaxAge    int
}

// NewSecurityHeadersMiddleware creates a new security headers middleware
func NewSecurityHeadersMiddleware(isDevelopment bool) *SecurityHeadersMiddleware {
	return &SecurityHeadersMiddleware{
		isDevelopment: isDevelopment,
		cspNonce:      generateCSPNonce,
		hstsMaxAge:    31536000, // 1 year in seconds
	}
}

// Middleware returns the HTTP middleware function
func (s *SecurityHeadersMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate CSP nonce for this request
		nonce := s.cspNonce()
		
		// Add security headers before processing the request
		s.setSecurityHeaders(w, r, nonce)
		
		// Process the request
		next.ServeHTTP(w, r)
	})
}

// setSecurityHeaders sets all security headers according to OWASP recommendations
func (s *SecurityHeadersMiddleware) setSecurityHeaders(w http.ResponseWriter, r *http.Request, nonce string) {
	// 1. Content Security Policy (CSP) - Prevents XSS attacks
	csp := s.buildCSP(nonce)
	w.Header().Set("Content-Security-Policy", csp)
	
	// Report-Only CSP for monitoring violations
	if !s.isDevelopment {
		w.Header().Set("Content-Security-Policy-Report-Only", csp+" report-uri /api/v1/security/csp-report;")
	}
	
	// 2. X-Content-Type-Options - Prevents MIME sniffing
	w.Header().Set("X-Content-Type-Options", "nosniff")
	
	// 3. X-Frame-Options - Prevents clickjacking
	w.Header().Set("X-Frame-Options", "DENY")
	
	// 4. X-XSS-Protection - Legacy XSS protection for older browsers
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	
	// 5. Referrer-Policy - Controls referrer information
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	
	// 6. Permissions-Policy (formerly Feature-Policy) - Controls browser features
	w.Header().Set("Permissions-Policy", s.buildPermissionsPolicy())
	
	// 7. Strict-Transport-Security (HSTS) - Forces HTTPS
	if !s.isDevelopment && r.TLS != nil {
		hstsValue := fmt.Sprintf("max-age=%d; includeSubDomains; preload", s.hstsMaxAge)
		w.Header().Set("Strict-Transport-Security", hstsValue)
	}
	
	// 8. Cache-Control - Prevents caching of sensitive data
	if strings.Contains(r.URL.Path, "/api/") || strings.Contains(r.URL.Path, "/auth/") {
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
	}
	
	// 9. X-Permitted-Cross-Domain-Policies - Controls Adobe products' access
	w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")
	
	// 10. X-Download-Options - Prevents IE from executing downloads
	w.Header().Set("X-Download-Options", "noopen")
	
	// 11. Cross-Origin Headers for enhanced security
	w.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")
	w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
	w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
	
	// 12. Security headers for API responses
	if strings.HasPrefix(r.URL.Path, "/api/") {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
	}
	
	// 13. Remove sensitive headers
	w.Header().Del("X-Powered-By")
	w.Header().Del("Server")
	w.Header().Del("X-AspNet-Version")
	w.Header().Del("X-AspNetMvc-Version")
	
	// 14. Add custom security headers
	w.Header().Set("X-Security-Policy", "enabled")
	w.Header().Set("X-Request-ID", generateSimpleRequestID())
}

// buildCSP builds a Content Security Policy based on OWASP Top 10 2025
func (s *SecurityHeadersMiddleware) buildCSP(nonce string) string {
	directives := []string{
		"default-src 'self'",
		fmt.Sprintf("script-src 'self' 'nonce-%s'", nonce),
		"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
		"img-src 'self' data: https:",
		"font-src 'self' https://fonts.gstatic.com",
		"connect-src 'self' https://api.nephoran.io wss://ws.nephoran.io",
		"media-src 'none'",
		"object-src 'none'",
		"frame-src 'none'",
		"frame-ancestors 'none'",
		"base-uri 'self'",
		"form-action 'self'",
		"manifest-src 'self'",
		"worker-src 'self'",
		"child-src 'none'",
		"prefetch-src 'none'",
		"navigate-to 'self'",
		"upgrade-insecure-requests",
		"block-all-mixed-content",
		"require-trusted-types-for 'script'",
	}
	
	// Relax CSP in development
	if s.isDevelopment {
		directives = []string{
			"default-src 'self'",
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'",
			"style-src 'self' 'unsafe-inline'",
			"img-src 'self' data: https:",
			"connect-src 'self' http://localhost:* ws://localhost:*",
		}
	}
	
	return strings.Join(directives, "; ")
}

// buildPermissionsPolicy builds a Permissions Policy header
func (s *SecurityHeadersMiddleware) buildPermissionsPolicy() string {
	policies := []string{
		"accelerometer=()",
		"ambient-light-sensor=()",
		"autoplay=()",
		"battery=()",
		"camera=()",
		"cross-origin-isolated=(self)",
		"display-capture=()",
		"document-domain=()",
		"encrypted-media=()",
		"execution-while-not-rendered=()",
		"execution-while-out-of-viewport=()",
		"fullscreen=(self)",
		"geolocation=()",
		"gyroscope=()",
		"keyboard-map=()",
		"magnetometer=()",
		"microphone=()",
		"midi=()",
		"navigation-override=()",
		"payment=()",
		"picture-in-picture=()",
		"publickey-credentials-get=()",
		"screen-wake-lock=()",
		"sync-xhr=()",
		"usb=()",
		"web-share=()",
		"xr-spatial-tracking=()",
		"interest-cohort=()",
	}
	
	return strings.Join(policies, ", ")
}

// Note: Using CORSConfig from config.go to avoid duplication

// NewSecureCORSMiddleware creates a secure CORS middleware
func NewSecureCORSMiddleware(config CORSConfig) func(http.Handler) http.Handler {
	// Set secure defaults if not provided
	if len(config.AllowedOrigins) == 0 {
		config.AllowedOrigins = []string{"https://nephoran.io"}
	}
	if len(config.AllowedMethods) == 0 {
		config.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	if len(config.AllowedHeaders) == 0 {
		config.AllowedHeaders = []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-CSRF-Token",
			"X-Request-ID",
		}
	}
	if len(config.ExposedHeaders) == 0 {
		config.ExposedHeaders = []string{
			"X-Request-ID",
			"X-RateLimit-Limit",
			"X-RateLimit-Remaining",
			"X-RateLimit-Reset",
		}
	}
	if config.MaxAge == 0 {
		config.MaxAge = 86400 // 24 hours
	}
	
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			
			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range config.AllowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}
			
			if allowed && origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				
				if config.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
				
				// Handle preflight requests
				if r.Method == "OPTIONS" {
					w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
					w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
					w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", config.MaxAge))
					w.WriteHeader(http.StatusNoContent)
					return
				}
				
				// Set exposed headers for actual requests
				if len(config.ExposedHeaders) > 0 {
					w.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
				}
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// CSRFMiddleware provides CSRF protection
type CSRFMiddleware struct {
	tokenLength int
	cookieName  string
	headerName  string
	secure      bool
}

// NewCSRFMiddleware creates a new CSRF middleware
func NewCSRFMiddleware(secure bool) *CSRFMiddleware {
	return &CSRFMiddleware{
		tokenLength: 32,
		cookieName:  "csrf_token",
		headerName:  "X-CSRF-Token",
		secure:      secure,
	}
}

// Middleware returns the CSRF middleware function
func (c *CSRFMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip CSRF for safe methods
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}
		
		// Get token from cookie
		cookie, err := r.Cookie(c.cookieName)
		if err != nil {
			http.Error(w, "CSRF token missing", http.StatusForbidden)
			return
		}
		
		// Get token from header or form
		headerToken := r.Header.Get(c.headerName)
		if headerToken == "" {
			headerToken = r.FormValue("csrf_token")
		}
		
		// Validate tokens match
		if !secureCompare(cookie.Value, headerToken) {
			http.Error(w, "CSRF token invalid", http.StatusForbidden)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// GenerateCSRFToken generates a new CSRF token
func (c *CSRFMiddleware) GenerateCSRFToken() (string, error) {
	b := make([]byte, c.tokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// Helper functions

func generateCSPNonce() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

func generateSimpleRequestID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%d", b, time.Now().UnixNano())
}

func secureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	
	result := byte(0)
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	
	return result == 0
}