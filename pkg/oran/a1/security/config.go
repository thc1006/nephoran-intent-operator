package security

// No imports needed for this config file

// SecurityConfig represents security configuration for A1 middleware
type SecurityConfig struct {
	// Authentication configuration (uses AuthConfig from auth.go)
	Authentication *AuthConfig `json:"authentication,omitempty"`

	// mTLS configuration (uses MTLSConfig from mtls.go)
	MTLS *MTLSConfig `json:"mtls,omitempty"`

	// Encryption configuration (uses EncryptionConfig from encryption.go)
	Encryption *EncryptionConfig `json:"encryption,omitempty"`

	// Input sanitization configuration (uses SanitizationConfig from sanitization.go)
	Sanitization *SanitizationConfig `json:"sanitization,omitempty"`

	// Audit logging configuration (uses AuditConfig from audit.go)
	Audit *AuditConfig `json:"audit,omitempty"`

	// Rate limiting configuration (uses RateLimitConfig from ratelimit.go)
	RateLimit *RateLimitConfig `json:"rate_limit,omitempty"`

	// Security headers configuration (uses HeadersConfig from middleware.go)
	SecurityHeaders *HeadersConfig `json:"security_headers,omitempty"`

	// CORS configuration (uses CORSConfig from middleware.go)
	CORS *CORSConfig `json:"cors,omitempty"`

	// CSRF configuration (uses CSRFConfig from middleware.go)
	CSRF *CSRFConfig `json:"csrf,omitempty"`
}

// AuthenticationConfig is now replaced by AuthConfig from auth.go

// MTLSConfig defines mutual TLS settings (simplified version, main definition in mtls.go)

// EncryptionConfig defines encryption settings (simplified version, main definition in encryption.go)

// SanitizationConfig defines input sanitization settings (simplified version, main definition in sanitization.go)

// AuditConfig defines audit logging settings (simplified version, main definition in audit.go)

// RateLimitConfig defines rate limiting settings (simplified version, main definition in ratelimit.go)
