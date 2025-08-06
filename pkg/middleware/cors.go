package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// CORSMiddleware handles Cross-Origin Resource Sharing (CORS) configuration
type CORSMiddleware struct {
	allowedOrigins   []string
	allowedMethods   []string
	allowedHeaders   []string
	exposedHeaders   []string
	allowCredentials bool
	maxAge          int
	logger          *slog.Logger
}

// CORSConfig holds CORS configuration options
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           time.Duration
}

// NewCORSMiddleware creates a new CORS middleware with security-focused defaults
func NewCORSMiddleware(config CORSConfig, logger *slog.Logger) *CORSMiddleware {
	// Security-focused defaults
	if len(config.AllowedMethods) == 0 {
		config.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	
	if len(config.AllowedHeaders) == 0 {
		// Restrict to essential headers only - do not use "*"
		config.AllowedHeaders = []string{
			"Content-Type", 
			"Authorization", 
			"X-Requested-With",
			"Accept",
			"Origin",
		}
	}
	
	// Default max age of 24 hours (recommended by security guidelines)
	maxAge := int(config.MaxAge.Seconds())
	if maxAge == 0 {
		maxAge = 86400 // 24 hours
	}

	return &CORSMiddleware{
		allowedOrigins:   config.AllowedOrigins,
		allowedMethods:   config.AllowedMethods,
		allowedHeaders:   config.AllowedHeaders,
		exposedHeaders:   config.ExposedHeaders,
		allowCredentials: config.AllowCredentials,
		maxAge:          maxAge,
		logger:          logger,
	}
}

// Middleware returns the CORS middleware handler
func (c *CORSMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		
		// Log CORS requests for security monitoring
		c.logger.Debug("CORS request", 
			slog.String("origin", origin),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path))

		// Check if origin is allowed
		if !c.isOriginAllowed(origin) {
			c.logger.Warn("CORS request blocked - origin not allowed", 
				slog.String("origin", origin),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("user_agent", r.Header.Get("User-Agent")))
			
			// Don't set any CORS headers for disallowed origins
			// This is more secure than sending CORS headers with empty values
			next.ServeHTTP(w, r)
			return
		}

		// Set CORS headers for allowed origins
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		
		// Set other CORS headers
		if len(c.allowedMethods) > 0 {
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(c.allowedMethods, ", "))
		}
		
		if len(c.allowedHeaders) > 0 {
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(c.allowedHeaders, ", "))
		}
		
		if len(c.exposedHeaders) > 0 {
			w.Header().Set("Access-Control-Expose-Headers", strings.Join(c.exposedHeaders, ", "))
		}
		
		if c.allowCredentials {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		
		if c.maxAge > 0 {
			w.Header().Set("Access-Control-Max-Age", strconv.Itoa(c.maxAge))
		}

		// Handle preflight OPTIONS requests
		if r.Method == "OPTIONS" {
			c.logger.Debug("Handling CORS preflight request", 
				slog.String("origin", origin),
				slog.String("requested_method", r.Header.Get("Access-Control-Request-Method")),
				slog.String("requested_headers", r.Header.Get("Access-Control-Request-Headers")))
			
			// Additional security validation for preflight requests
			requestedMethod := r.Header.Get("Access-Control-Request-Method")
			if requestedMethod != "" && !c.isMethodAllowed(requestedMethod) {
				c.logger.Warn("CORS preflight blocked - method not allowed", 
					slog.String("origin", origin),
					slog.String("requested_method", requestedMethod))
				w.WriteHeader(http.StatusForbidden)
				return
			}
			
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// isOriginAllowed checks if the given origin is in the allowed list
func (c *CORSMiddleware) isOriginAllowed(origin string) bool {
	if origin == "" {
		return true // Allow same-origin requests
	}
	
	for _, allowedOrigin := range c.allowedOrigins {
		if allowedOrigin == "*" {
			// Wildcard allowed (should only be in development)
			c.logger.Warn("Wildcard CORS origin allowed", 
				slog.String("origin", origin),
				slog.String("security_warning", "wildcard should only be used in development"))
			return true
		}
		
		if allowedOrigin == origin {
			return true
		}
		
		// Additional pattern matching could be added here for subdomain wildcards
		// But for security, we prefer exact matches
	}
	
	return false
}

// isMethodAllowed checks if the HTTP method is allowed
func (c *CORSMiddleware) isMethodAllowed(method string) bool {
	for _, allowedMethod := range c.allowedMethods {
		if allowedMethod == method {
			return true
		}
	}
	return false
}

// ValidateConfig performs security validation on CORS configuration
func ValidateConfig(config CORSConfig) error {
	// Check for insecure wildcard configurations
	for _, origin := range config.AllowedOrigins {
		if origin == "*" && config.AllowCredentials {
			return fmt.Errorf("wildcard origin '*' cannot be used with credentials enabled - this is a security vulnerability")
		}
		
		if strings.Contains(origin, "*") && origin != "*" {
			return fmt.Errorf("partial wildcard origins like '%s' are not supported for security reasons", origin)
		}
		
		// Validate origin format
		if origin != "*" && !strings.HasPrefix(origin, "http://") && !strings.HasPrefix(origin, "https://") {
			return fmt.Errorf("origin '%s' must start with http:// or https://", origin)
		}
	}
	
	return nil
}