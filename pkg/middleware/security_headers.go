// Package middleware provides HTTP middleware components for the Nephoran Intent Operator.

package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

// SecurityHeadersConfig defines configuration for security headers middleware.

// Following OWASP security headers best practices.

type SecurityHeadersConfig struct {
	// EnableHSTS controls whether to set Strict-Transport-Security header.

	// Should only be enabled when TLS is properly configured.

	EnableHSTS bool

	// HSTSMaxAge defines max-age directive in seconds for HSTS.

	// Default: 31536000 (1 year) as recommended by OWASP.

	HSTSMaxAge int

	// HSTSIncludeSubDomains adds includeSubDomains directive to HSTS.

	// Applies HSTS policy to all subdomains.

	HSTSIncludeSubDomains bool

	// HSTSPreload adds preload directive to HSTS.

	// Allows domain to be included in browser HSTS preload lists.

	// WARNING: This is permanent and requires careful consideration.

	HSTSPreload bool

	// FrameOptions controls X-Frame-Options header value.

	// Options: "DENY" (default), "SAMEORIGIN", or specific origin.

	// Prevents clickjacking attacks.

	FrameOptions string

	// ContentTypeOptions controls X-Content-Type-Options header.

	// When true, sets "nosniff" to prevent MIME type sniffing.

	ContentTypeOptions bool

	// ReferrerPolicy controls Referrer-Policy header.

	// Options: "no-referrer" (default), "strict-origin-when-cross-origin", etc.

	// Controls how much referrer information is shared.

	ReferrerPolicy string

	// ContentSecurityPolicy defines CSP directives.

	// Default: "default-src 'none'; frame-ancestors 'none'".

	// Can be customized based on application requirements.

	ContentSecurityPolicy string

	// PermissionsPolicy defines Permissions-Policy header.

	// Controls which browser features can be used.

	// Default: "geolocation=(), microphone=(), camera=()".

	PermissionsPolicy string

	// XSSProtection controls X-XSS-Protection header.

	// Legacy header, but still useful for older browsers.

	// Default: "1; mode=block".

	XSSProtection string

	// CustomHeaders allows adding additional security headers.

	// Key-value pairs for headers not covered by standard options.

	CustomHeaders map[string]string
}

// DefaultSecurityHeadersConfig returns a secure default configuration.

// Following OWASP recommendations for security headers.

func DefaultSecurityHeadersConfig() *SecurityHeadersConfig {
	return &SecurityHeadersConfig{
		EnableHSTS: false, // Disabled by default, enable only with proper TLS

		HSTSMaxAge: 31536000, // 1 year

		HSTSIncludeSubDomains: false,

		HSTSPreload: false,

		FrameOptions: "DENY",

		ContentTypeOptions: true,

		ReferrerPolicy: "strict-origin-when-cross-origin",

		ContentSecurityPolicy: "default-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'",

		PermissionsPolicy: "geolocation=(), microphone=(), camera=(), payment=(), usb=(), magnetometer=(), gyroscope=(), accelerometer=()",

		XSSProtection: "1; mode=block",

		CustomHeaders: make(map[string]string),
	}
}

// SecurityHeaders implements security headers middleware.

// Adds comprehensive security headers to HTTP responses.

type SecurityHeaders struct {
	config *SecurityHeadersConfig

	logger *slog.Logger
}

// NewSecurityHeaders creates a new security headers middleware instance.

// config: Security headers configuration (nil uses defaults).

// logger: Structured logger for middleware operations.

func NewSecurityHeaders(config *SecurityHeadersConfig, logger *slog.Logger) *SecurityHeaders {
	if config == nil {
		config = DefaultSecurityHeadersConfig()
	}

	// Validate configuration.

	if config.HSTSMaxAge < 0 {
		config.HSTSMaxAge = 31536000
	}

	if config.FrameOptions == "" {
		config.FrameOptions = "DENY"
	}

	if config.ReferrerPolicy == "" {
		config.ReferrerPolicy = "strict-origin-when-cross-origin"
	}

	return &SecurityHeaders{
		config: config,

		logger: logger.With(slog.String("component", "security-headers")),
	}
}

// Middleware returns an HTTP middleware that adds security headers to responses.

// The middleware should be applied early in the chain to ensure headers are set.

func (sh *SecurityHeaders) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set Strict-Transport-Security header if HSTS is enabled.

		// Only set on HTTPS connections to avoid issues.

		if sh.config.EnableHSTS && sh.isSecureConnection(r) {

			hstsValue := sh.buildHSTSHeader()

			w.Header().Set("Strict-Transport-Security", hstsValue)

			sh.logger.Debug("HSTS header set",

				slog.String("value", hstsValue),

				slog.String("path", r.URL.Path))

		}

		// X-Frame-Options prevents clickjacking attacks.

		// DENY: Page cannot be displayed in a frame.

		// SAMEORIGIN: Page can only be displayed in a frame on the same origin.

		if sh.config.FrameOptions != "" {
			w.Header().Set("X-Frame-Options", sh.config.FrameOptions)
		}

		// X-Content-Type-Options prevents MIME type sniffing.

		// nosniff: Blocks requests if the MIME type doesn't match.

		if sh.config.ContentTypeOptions {
			w.Header().Set("X-Content-Type-Options", "nosniff")
		}

		// Referrer-Policy controls referrer information sent with requests.

		// Helps prevent information leakage.

		if sh.config.ReferrerPolicy != "" {
			w.Header().Set("Referrer-Policy", sh.config.ReferrerPolicy)
		}

		// Content-Security-Policy provides fine-grained control over resource loading.

		// Helps prevent XSS, data injection, and other attacks.

		if sh.config.ContentSecurityPolicy != "" {
			w.Header().Set("Content-Security-Policy", sh.config.ContentSecurityPolicy)
		}

		// Permissions-Policy (formerly Feature-Policy) controls browser features.

		// Restricts access to sensitive APIs.

		if sh.config.PermissionsPolicy != "" {
			w.Header().Set("Permissions-Policy", sh.config.PermissionsPolicy)
		}

		// X-XSS-Protection for older browsers that don't support CSP.

		// Modern browsers have this built-in, but keeping for compatibility.

		if sh.config.XSSProtection != "" {
			w.Header().Set("X-XSS-Protection", sh.config.XSSProtection)
		}

		// Apply any custom headers.

		for header, value := range sh.config.CustomHeaders {
			w.Header().Set(header, value)
		}

		// Log security headers application for audit trail.

		sh.logger.Debug("Security headers applied",

			slog.String("method", r.Method),

			slog.String("path", r.URL.Path),

			slog.String("remote_addr", r.RemoteAddr),

			slog.Bool("secure", sh.isSecureConnection(r)))

		// Call the next handler.

		next.ServeHTTP(w, r)
	})
}

// buildHSTSHeader constructs the HSTS header value based on configuration.

func (sh *SecurityHeaders) buildHSTSHeader() string {
	var directives []string

	// max-age is required.

	directives = append(directives, fmt.Sprintf("max-age=%d", sh.config.HSTSMaxAge))

	// includeSubDomains is optional but recommended.

	if sh.config.HSTSIncludeSubDomains {
		directives = append(directives, "includeSubDomains")
	}

	// preload requires includeSubDomains and careful consideration.

	// WARNING: Once preloaded, it's very difficult to remove.

	if sh.config.HSTSPreload && sh.config.HSTSIncludeSubDomains {
		directives = append(directives, "preload")
	}

	return strings.Join(directives, "; ")
}

// isSecureConnection determines if the request is over HTTPS.

// Checks multiple indicators including TLS state and forwarded headers.

func (sh *SecurityHeaders) isSecureConnection(r *http.Request) bool {
	// Check if TLS is directly configured.

	if r.TLS != nil {
		return true
	}

	// Check X-Forwarded-Proto header (common with reverse proxies).

	if forwarded := r.Header.Get("X-Forwarded-Proto"); forwarded == "https" {
		return true
	}

	// Check Forwarded header (RFC 7239).

	if forwarded := r.Header.Get("Forwarded"); strings.Contains(forwarded, "proto=https") {
		return true
	}

	// Check URL scheme.

	if r.URL.Scheme == "https" {
		return true
	}

	return false
}

// ValidateConfig checks if the security headers configuration is valid.

// Returns an error if any configuration is invalid.

func (sh *SecurityHeaders) ValidateConfig() error {
	// Validate HSTS configuration.

	if sh.config.EnableHSTS {

		if sh.config.HSTSMaxAge < 10886400 { // 18 weeks minimum for preload

			if sh.config.HSTSPreload {
				return fmt.Errorf("HSTS max-age must be at least 10886400 seconds (18 weeks) for preload")
			}
		}

		if sh.config.HSTSPreload && !sh.config.HSTSIncludeSubDomains {
			return fmt.Errorf("HSTS preload requires includeSubDomains to be enabled")
		}

	}

	// Validate FrameOptions.

	validFrameOptions := map[string]bool{
		"DENY": true,

		"SAMEORIGIN": true,
	}

	if sh.config.FrameOptions != "" && !validFrameOptions[sh.config.FrameOptions] {
		// Check if it's an ALLOW-FROM directive (deprecated but still used).

		if !strings.HasPrefix(sh.config.FrameOptions, "ALLOW-FROM ") {
			return fmt.Errorf("invalid X-Frame-Options value: %s", sh.config.FrameOptions)
		}
	}

	// Validate CSP if provided.

	if sh.config.ContentSecurityPolicy != "" {
		// Basic validation - ensure it contains directives.

		if !strings.Contains(sh.config.ContentSecurityPolicy, "-src") &&

			!strings.Contains(sh.config.ContentSecurityPolicy, "frame-ancestors") &&

			!strings.Contains(sh.config.ContentSecurityPolicy, "base-uri") {

			sh.logger.Warn("CSP appears to have no directives",

				slog.String("csp", sh.config.ContentSecurityPolicy))
		}
	}

	return nil
}

// GetActiveHeaders returns a map of headers that will be set based on current configuration.

// Useful for debugging and testing.

func (sh *SecurityHeaders) GetActiveHeaders(isSecure bool) map[string]string {
	headers := make(map[string]string)

	if sh.config.EnableHSTS && isSecure {
		headers["Strict-Transport-Security"] = sh.buildHSTSHeader()
	}

	if sh.config.FrameOptions != "" {
		headers["X-Frame-Options"] = sh.config.FrameOptions
	}

	if sh.config.ContentTypeOptions {
		headers["X-Content-Type-Options"] = "nosniff"
	}

	if sh.config.ReferrerPolicy != "" {
		headers["Referrer-Policy"] = sh.config.ReferrerPolicy
	}

	if sh.config.ContentSecurityPolicy != "" {
		headers["Content-Security-Policy"] = sh.config.ContentSecurityPolicy
	}

	if sh.config.PermissionsPolicy != "" {
		headers["Permissions-Policy"] = sh.config.PermissionsPolicy
	}

	if sh.config.XSSProtection != "" {
		headers["X-XSS-Protection"] = sh.config.XSSProtection
	}

	// Add custom headers.

	for k, v := range sh.config.CustomHeaders {
		headers[k] = v
	}

	return headers
}

// ReportOnlyCSP sets a Content-Security-Policy-Report-Only header.

// Useful for testing CSP policies without enforcing them.

func (sh *SecurityHeaders) ReportOnlyCSP(policy, reportURI string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cspValue := policy

		if reportURI != "" {
			cspValue = fmt.Sprintf("%s; report-uri %s", policy, reportURI)
		}

		w.Header().Set("Content-Security-Policy-Report-Only", cspValue)
	})
}
