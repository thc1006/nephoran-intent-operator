// Package security provides comprehensive security configuration and validation
package security

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// SecurityConfig holds security-related configuration settings
type SecurityConfig struct {
	// TLS Configuration
	TLSMinVersion   string   `json:"tls_min_version" yaml:"tls_min_version"`
	TLSCipherSuites []string `json:"tls_cipher_suites" yaml:"tls_cipher_suites"`
	RequireMTLS     bool     `json:"require_mtls" yaml:"require_mtls"`

	// Authentication Settings
	RequireStrongPasswords bool   `json:"require_strong_passwords" yaml:"require_strong_passwords"`
	MinPasswordLength      int    `json:"min_password_length" yaml:"min_password_length"`
	PasswordComplexity     string `json:"password_complexity" yaml:"password_complexity"`
	SessionTimeout         int    `json:"session_timeout" yaml:"session_timeout"` // in minutes
	MaxFailedAttempts      int    `json:"max_failed_attempts" yaml:"max_failed_attempts"`
	LockoutDuration        int    `json:"lockout_duration" yaml:"lockout_duration"` // in minutes

	// API Security
	RateLimitPerMinute int      `json:"rate_limit_per_minute" yaml:"rate_limit_per_minute"`
	AllowedOrigins     []string `json:"allowed_origins" yaml:"allowed_origins"`
	RequireAPIKey      bool     `json:"require_api_key" yaml:"require_api_key"`
	APIKeyRotationDays int      `json:"api_key_rotation_days" yaml:"api_key_rotation_days"`

	// Security Headers
	EnableHSTS bool   `json:"enable_hsts" yaml:"enable_hsts"`
	HSTSMaxAge int    `json:"hsts_max_age" yaml:"hsts_max_age"`
	EnableCSP  bool   `json:"enable_csp" yaml:"enable_csp"`
	CSPPolicy  string `json:"csp_policy" yaml:"csp_policy"`

	// Input Validation
	MaxRequestSize    int64    `json:"max_request_size" yaml:"max_request_size"` // in bytes
	AllowedFileTypes  []string `json:"allowed_file_types" yaml:"allowed_file_types"`
	SanitizeUserInput bool     `json:"sanitize_user_input" yaml:"sanitize_user_input"`

	// Audit and Logging
	EnableAuditLog        bool `json:"enable_audit_log" yaml:"enable_audit_log"`
	LogSensitiveData      bool `json:"log_sensitive_data" yaml:"log_sensitive_data"`
	AuditLogRetentionDays int  `json:"audit_log_retention_days" yaml:"audit_log_retention_days"`
}

// DefaultSecurityConfig returns a secure default configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		// TLS Configuration - strict settings
		TLSMinVersion: "1.3",
		TLSCipherSuites: []string{
			"TLS_AES_256_GCM_SHA384",
			"TLS_CHACHA20_POLY1305_SHA256",
			"TLS_AES_128_GCM_SHA256",
		},
		RequireMTLS: false, // Can be enabled based on requirements

		// Authentication - OWASP compliant
		RequireStrongPasswords: true,
		MinPasswordLength:      12,
		PasswordComplexity:     "high", // requires uppercase, lowercase, numbers, special chars
		SessionTimeout:         30,     // 30 minutes
		MaxFailedAttempts:      5,
		LockoutDuration:        15, // 15 minutes

		// API Security
		RateLimitPerMinute: 100,
		AllowedOrigins:     []string{}, // Must be explicitly configured
		RequireAPIKey:      true,
		APIKeyRotationDays: 90,

		// Security Headers
		EnableHSTS: true,
		HSTSMaxAge: 31536000, // 1 year
		EnableCSP:  true,
		CSPPolicy:  "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:;",

		// Input Validation
		MaxRequestSize:    10 * 1024 * 1024, // 10MB
		AllowedFileTypes:  []string{".json", ".yaml", ".yml", ".txt"},
		SanitizeUserInput: true,

		// Audit and Logging
		EnableAuditLog:        true,
		LogSensitiveData:      false, // Never log sensitive data
		AuditLogRetentionDays: 90,
	}
}

// Validate checks if the security configuration is valid and secure
func (sc *SecurityConfig) Validate() error {
	// Validate TLS settings
	if sc.TLSMinVersion != "" {
		validVersions := map[string]bool{
			"1.2": true,
			"1.3": true,
		}
		if !validVersions[sc.TLSMinVersion] {
			return fmt.Errorf("invalid TLS min version: %s (must be 1.2 or 1.3)", sc.TLSMinVersion)
		}
	}

	// Validate password settings
	if sc.RequireStrongPasswords {
		if sc.MinPasswordLength < 8 {
			return fmt.Errorf("minimum password length must be at least 8 when strong passwords are required")
		}
	}

	// Validate session settings
	if sc.SessionTimeout < 1 {
		return fmt.Errorf("session timeout must be at least 1 minute")
	}
	if sc.SessionTimeout > 1440 { // 24 hours
		return fmt.Errorf("session timeout cannot exceed 24 hours (1440 minutes)")
	}

	// Validate rate limiting
	if sc.RateLimitPerMinute < 1 {
		return fmt.Errorf("rate limit must be at least 1 request per minute")
	}

	// Validate HSTS settings
	if sc.EnableHSTS && sc.HSTSMaxAge < 86400 { // 1 day minimum
		return fmt.Errorf("HSTS max age must be at least 86400 seconds (1 day)")
	}

	// Validate request size
	if sc.MaxRequestSize < 1024 { // 1KB minimum
		return fmt.Errorf("max request size must be at least 1KB")
	}
	if sc.MaxRequestSize > 100*1024*1024 { // 100MB maximum
		return fmt.Errorf("max request size cannot exceed 100MB")
	}

	return nil
}

// SecretValidator validates that no secrets are hardcoded in the codebase
type SecretValidator struct {
	patterns []*regexp.Regexp
}

// NewSecretValidator creates a new secret validator
func NewSecretValidator() *SecretValidator {
	return &SecretValidator{
		patterns: []*regexp.Regexp{
			// API Keys and tokens
			regexp.MustCompile(`(?i)(api[_-]?key|apikey|api_secret|api[_-]?token)[\s]*[:=][\s]*["'][a-zA-Z0-9\-_]{20,}["']`),
			// Passwords
			regexp.MustCompile(`(?i)(password|passwd|pwd|pass)[\s]*[:=][\s]*["'][^"']{8,}["']`),
			// Secret keys
			regexp.MustCompile(`(?i)(secret[_-]?key|secret|private[_-]?key)[\s]*[:=][\s]*["'][a-zA-Z0-9\-_]{16,}["']`),
			// Database credentials
			regexp.MustCompile(`(?i)(db[_-]?password|database[_-]?password|mysql[_-]?password|postgres[_-]?password)[\s]*[:=][\s]*["'][^"']+["']`),
			// AWS credentials
			regexp.MustCompile(`(?i)(aws[_-]?access[_-]?key[_-]?id|aws[_-]?secret[_-]?access[_-]?key)[\s]*[:=][\s]*["'][A-Z0-9]{16,}["']`),
			// JWT secrets
			regexp.MustCompile(`(?i)(jwt[_-]?secret|jwt[_-]?key)[\s]*[:=][\s]*["'][^"']{16,}["']`),
			// OAuth secrets
			regexp.MustCompile(`(?i)(client[_-]?secret|oauth[_-]?secret)[\s]*[:=][\s]*["'][a-zA-Z0-9\-_]{16,}["']`),
		},
	}
}

// ValidateFile checks a file for hardcoded secrets
func (sv *SecretValidator) ValidateFile(filepath string) ([]string, error) {
	// Skip test files and examples
	if strings.Contains(filepath, "_test.go") ||
		strings.Contains(filepath, "/test/") ||
		strings.Contains(filepath, "/tests/") ||
		strings.Contains(filepath, "/examples/") ||
		strings.Contains(filepath, "/example/") ||
		strings.Contains(filepath, "/mock") ||
		strings.Contains(filepath, "/fixture") {
		return nil, nil // Skip test and example files
	}

	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filepath, err)
	}

	var violations []string
	lines := strings.Split(string(content), "\n")

	for lineNum, line := range lines {
		// Skip comments
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "//") || strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		// Check for hardcoded secrets
		for _, pattern := range sv.patterns {
			if pattern.MatchString(line) {
				// Check if it's loading from environment
				if !strings.Contains(line, "os.Getenv") &&
					!strings.Contains(line, "viper.Get") &&
					!strings.Contains(line, "config.Get") {
					violations = append(violations, fmt.Sprintf("Line %d: Potential hardcoded secret detected", lineNum+1))
				}
			}
		}
	}

	return violations, nil
}

// InputSanitizer provides input sanitization utilities
type InputSanitizer struct {
	// HTML/XSS patterns
	htmlPatterns []*regexp.Regexp
	// SQL injection patterns
	sqlPatterns []*regexp.Regexp
	// Command injection patterns
	cmdPatterns []*regexp.Regexp
}

// NewInputSanitizer creates a new input sanitizer
func NewInputSanitizer() *InputSanitizer {
	return &InputSanitizer{
		htmlPatterns: []*regexp.Regexp{
			regexp.MustCompile(`<script[^>]*>.*?</script>`),
			regexp.MustCompile(`<iframe[^>]*>.*?</iframe>`),
			regexp.MustCompile(`javascript:`),
			regexp.MustCompile(`on\w+\s*=`), // Event handlers
		},
		sqlPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop|create|alter|exec|execute|script|javascript|eval)`),
			regexp.MustCompile(`['";].*?(or|and)\s+.*?['";]?\s*=\s*['";]?`),
			regexp.MustCompile(`\b(or|and)\s+\d+\s*=\s*\d+`),
		},
		cmdPatterns: []*regexp.Regexp{
			regexp.MustCompile(`[;&|]`),
			regexp.MustCompile(`\$\(`),
			regexp.MustCompile("`"),
			regexp.MustCompile(`\b(exec|system|eval|subprocess|popen)\b`),
		},
	}
}

// SanitizeInput removes potentially dangerous content from user input
func (is *InputSanitizer) SanitizeInput(input string, ctx context.Context) string {
	sanitized := input

	// Remove HTML/XSS patterns
	for _, pattern := range is.htmlPatterns {
		sanitized = pattern.ReplaceAllString(sanitized, "")
	}

	// Escape special characters
	replacements := map[string]string{
		"<":  "&lt;",
		">":  "&gt;",
		"&":  "&amp;",
		"\"": "&quot;",
		"'":  "&#x27;",
		"/":  "&#x2F;",
	}

	for old, new := range replacements {
		sanitized = strings.ReplaceAll(sanitized, old, new)
	}

	return sanitized
}

// ValidateSQL checks for SQL injection attempts
func (is *InputSanitizer) ValidateSQL(input string) bool {
	for _, pattern := range is.sqlPatterns {
		if pattern.MatchString(input) {
			return false
		}
	}
	return true
}

// ValidateCommand checks for command injection attempts
func (is *InputSanitizer) ValidateCommand(input string) bool {
	for _, pattern := range is.cmdPatterns {
		if pattern.MatchString(input) {
			return false
		}
	}
	return true
}

// ErrorSanitizer provides error message sanitization to prevent information leakage
type ErrorSanitizer struct {
	// Patterns that may leak sensitive information
	sensitivePatterns []*regexp.Regexp
}

// NewErrorSanitizer creates a new error sanitizer
func NewErrorSanitizer() *ErrorSanitizer {
	return &ErrorSanitizer{
		sensitivePatterns: []*regexp.Regexp{
			// File paths
			regexp.MustCompile(`(?i)(/[a-z0-9_\-./]+)+`),
			// Stack traces
			regexp.MustCompile(`at\s+.*?\(.*?:\d+:\d+\)`),
			// Database errors
			regexp.MustCompile(`(?i)(mysql|postgres|mongodb|redis|elasticsearch).*error`),
			// Network addresses
			regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b`),
			// Ports
			regexp.MustCompile(`:\d{2,5}\b`),
		},
	}
}

// SanitizeError returns a safe error message for external consumption
func (es *ErrorSanitizer) SanitizeError(err error) string {
	if err == nil {
		return ""
	}

	errMsg := err.Error()

	// Map specific error types to generic messages
	genericErrors := map[string]string{
		"connection refused": "Service temporarily unavailable",
		"timeout":            "Request timeout",
		"permission denied":  "Access denied",
		"not found":          "Resource not found",
		"invalid":            "Invalid request",
		"unauthorized":       "Authentication required",
		"forbidden":          "Insufficient permissions",
		"too many":           "Rate limit exceeded",
		"database":           "Data operation failed",
		"parse":              "Invalid input format",
		"certificate":        "Security validation failed",
	}

	lowerErr := strings.ToLower(errMsg)
	for key, generic := range genericErrors {
		if strings.Contains(lowerErr, key) {
			return generic
		}
	}

	// Check for sensitive patterns
	for _, pattern := range es.sensitivePatterns {
		if pattern.MatchString(errMsg) {
			return "An internal error occurred"
		}
	}

	// Default generic error
	return "An error occurred processing your request"
}

// SecurityAuditor provides security audit functionality
type SecurityAuditor struct {
	config *SecurityConfig
}

// NewSecurityAuditor creates a new security auditor
func NewSecurityAuditor(config *SecurityConfig) *SecurityAuditor {
	if config == nil {
		config = DefaultSecurityConfig()
	}
	return &SecurityAuditor{
		config: config,
	}
}

// AuditRequest logs security-relevant request information
func (sa *SecurityAuditor) AuditRequest(ctx context.Context, method, path, userID string, statusCode int) {
	// Implementation would log to audit system
	// This is a placeholder for the actual implementation
	if sa.config.EnableAuditLog {
		// Log audit event (implementation depends on logging system)
		_ = fmt.Sprintf("AUDIT: %s %s by user %s - status %d", method, path, userID, statusCode)
	}
}

// AuditSecurityEvent logs security events
func (sa *SecurityAuditor) AuditSecurityEvent(ctx context.Context, eventType, description string, metadata map[string]interface{}) {
	if sa.config.EnableAuditLog {
		// Log security event (implementation depends on logging system)
		_ = fmt.Sprintf("SECURITY_EVENT: %s - %s", eventType, description)
	}
}
