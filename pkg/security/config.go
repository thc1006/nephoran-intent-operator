package security

import (
	"time"
)

// CommonSecurityConfig defines common security configuration
type CommonSecurityConfig struct {
	// TLS Configuration
	TLS *TLSConfig `json:"tls,omitempty"`
	
	// Security Headers Configuration
	SecurityHeaders *SecurityHeadersConfig `json:"security_headers,omitempty"`
	
	// Enable security features
	Enabled bool `json:"enabled"`
	
	// Security mode (strict, moderate, permissive)
	Mode string `json:"mode"`
}

// TLSConfig defines TLS configuration settings
type TLSConfig struct {
	Enabled          bool              `json:"enabled"`
	CertFile         string            `json:"cert_file"`
	KeyFile          string            `json:"key_file"`
	CAFile           string            `json:"ca_file,omitempty"`
	MinVersion       string            `json:"min_version"`
	MaxVersion       string            `json:"max_version,omitempty"`
	CipherSuites     []string          `json:"cipher_suites,omitempty"`
	ClientAuth       string            `json:"client_auth,omitempty"`
	InsecureSkipVerify bool           `json:"insecure_skip_verify"`
	CertificatePaths *CertificatePaths `json:"certificate_paths,omitempty"`
}

// SecurityHeadersConfig defines security headers configuration
type SecurityHeadersConfig struct {
	ContentSecurityPolicy   string `json:"content_security_policy,omitempty"`
	XFrameOptions          string `json:"x_frame_options,omitempty"`
	XContentTypeOptions    string `json:"x_content_type_options,omitempty"`
	StrictTransportSecurity string `json:"strict_transport_security,omitempty"`
	XSSProtection          string `json:"xss_protection,omitempty"`
	ReferrerPolicy         string `json:"referrer_policy,omitempty"`
	PermissionsPolicy      string `json:"permissions_policy,omitempty"`
}

// CertificatePaths defines paths for certificate files
type CertificatePaths struct {
	CertFile string `json:"cert_file"`
	KeyFile  string `json:"key_file"`
	CAFile   string `json:"ca_file,omitempty"`
}

// Extended security config with additional fields not in the core interfaces
type ExtendedSecurityConfig struct {
	// Embed the common config from interfaces
	*CommonSecurityConfig

	// Authentication Configuration
	Auth *AuthConfig `json:"auth,omitempty"`

	// RBAC Configuration
	RBAC *RBACConfig `json:"rbac,omitempty"`

	// Rate Limiting Configuration
	RateLimit *RateLimitConfig `json:"rate_limit,omitempty"`

	// CORS Configuration
	CORS *CORSConfig `json:"cors,omitempty"`

	// Input Validation Configuration
	InputValidation *InputValidationConfig `json:"input_validation,omitempty"`

	// Audit Configuration
	Audit *AuditConfig `json:"audit,omitempty"`

	// Encryption Configuration
	Encryption *EncryptionConfig `json:"encryption,omitempty"`
}


// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled        bool                      `json:"enabled"`
	Providers      []string                  `json:"providers,omitempty"`
	OAuthProviders map[string]*OAuthProvider `json:"oauth_providers,omitempty"`
	JWT            *JWTConfig                `json:"jwt,omitempty"`
	LDAP           *LDAPConfig               `json:"ldap,omitempty"`
	DefaultScopes  []string                  `json:"default_scopes,omitempty"`
	TokenTTL       string                    `json:"token_ttl,omitempty"`
	RefreshEnabled bool                      `json:"refresh_enabled"`
	CacheEnabled   bool                      `json:"cache_enabled"`
	CacheTTL       string                    `json:"cache_ttl,omitempty"`
}

// OAuthProvider represents an OAuth provider configuration
type OAuthProvider struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	IssuerURL    string            `json:"issuer_url"`
	ClientID     string            `json:"client_id"`
	ClientSecret string            `json:"client_secret"`
	RedirectURL  string            `json:"redirect_url,omitempty"`
	TenantID     string            `json:"tenant_id,omitempty"`
	Scopes       []string          `json:"scopes,omitempty"`
	ExtraParams  map[string]string `json:"extra_params,omitempty"`
	Enabled      bool              `json:"enabled"`
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	SecretKey      string        `json:"secret_key,omitempty"`
	TokenDuration  string        `json:"token_duration,omitempty"`
	RefreshEnabled bool          `json:"refresh_enabled"`
	TokenTTL       time.Duration `json:"token_ttl,omitempty"`
	RefreshTTL     time.Duration `json:"refresh_ttl,omitempty"`
}

// LDAPConfig holds LDAP configuration
type LDAPConfig struct {
	Enabled bool   `json:"enabled"`
	Host    string `json:"host,omitempty"`
	Port    int    `json:"port,omitempty"`
	BaseDN  string `json:"base_dn,omitempty"`
}

// RBACConfig holds RBAC configuration
type RBACConfig struct {
	Enabled       bool     `json:"enabled"`
	PolicyPath    string   `json:"policy_path,omitempty"`
	DefaultPolicy string   `json:"default_policy,omitempty"` // ALLOW, DENY
	DefaultRole   string   `json:"default_role,omitempty"`
	AdminUsers    []string `json:"admin_users,omitempty"`
	AdminRoles    []string `json:"admin_roles,omitempty"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled           bool          `json:"enabled"`
	RequestsPerMin    int           `json:"requests_per_min,omitempty"`
	RequestsPerMinute int           `json:"requests_per_minute,omitempty"` // Alternative naming
	BurstSize         int           `json:"burst_size,omitempty"`
	BurstLimit        int           `json:"burst_limit,omitempty"` // Alternative naming
	KeyFunc           string        `json:"key_func,omitempty"`    // ip, user, token
	RateLimitWindow   time.Duration `json:"rate_limit_window,omitempty"`
	RateLimitByIP     bool          `json:"rate_limit_by_ip"`
	RateLimitByAPIKey bool          `json:"rate_limit_by_api_key"`
	CleanupInterval   time.Duration `json:"cleanup_interval,omitempty"`
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	Enabled          bool     `json:"enabled"`
	AllowedOrigins   []string `json:"allowed_origins,omitempty"`
	AllowedMethods   []string `json:"allowed_methods,omitempty"`
	AllowedHeaders   []string `json:"allowed_headers,omitempty"`
	ExposedHeaders   []string `json:"exposed_headers,omitempty"`
	AllowCredentials bool     `json:"allow_credentials"`
	MaxAge           int      `json:"max_age,omitempty"`
}

// InputValidationConfig defines input validation configuration
type InputValidationConfig struct {
	Enabled                bool     `json:"enabled"`
	MaxRequestSize         int      `json:"max_request_size,omitempty"`
	MaxInputLength         int      `json:"max_input_length,omitempty"`
	MaxOutputLength        int      `json:"max_output_length,omitempty"`
	SanitizeHTML           bool     `json:"sanitize_html"`
	SanitizeInput          bool     `json:"sanitize_input"`
	ValidateJSONSchema     bool     `json:"validate_json_schema"`
	EnableSchemaValidation bool     `json:"enable_schema_validation"`
	StrictValidation       bool     `json:"strict_validation"`
	ForbiddenPatterns      []string `json:"forbidden_patterns,omitempty"`
	RequiredHeaders        []string `json:"required_headers,omitempty"`
}


// AuditConfig holds audit configuration
type AuditConfig struct {
	Enabled               bool   `json:"enabled"`
	LogLevel              string `json:"log_level,omitempty"`
	LogSuccessfulRequests bool   `json:"log_successful_requests"`
	LogFailedRequests     bool   `json:"log_failed_requests"`
	RetentionDays         int    `json:"retention_days,omitempty"`
}

// EncryptionConfig holds encryption configuration
type EncryptionConfig struct {
	Enabled           bool     `json:"enabled"`
	Algorithm         string   `json:"algorithm,omitempty"`
	KeySize           int      `json:"key_size,omitempty"`
	EncryptionKey     string   `json:"encryption_key,omitempty"`
	EncryptionKeyPath string   `json:"encryption_key_path,omitempty"`
	SupportedCiphers  []string `json:"supported_ciphers,omitempty"`
}

// ToCommonConfig converts any SecurityConfig variant to CommonSecurityConfig
func ToCommonConfig(config interface{}) *CommonSecurityConfig {
	// This function can be used to convert package-specific configs
	// to the common format. Implementation would depend on the source type.

	if common, ok := config.(*CommonSecurityConfig); ok {
		return common
	}

	// Default empty config if conversion fails
	return &CommonSecurityConfig{}
}

// DefaultSecurityConfig returns a default security configuration
func DefaultSecurityConfig() *ExtendedSecurityConfig {
	return &ExtendedSecurityConfig{
		CommonSecurityConfig: &CommonSecurityConfig{
			TLS: &TLSConfig{
				Enabled:    true,
				MinVersion: "1.2",
			},
			SecurityHeaders: &SecurityHeadersConfig{
				StrictTransportSecurity: "max-age=31536000; includeSubDomains",
				ContentSecurityPolicy:   "default-src 'self'",
				XContentTypeOptions:     "nosniff",
				XFrameOptions:           "DENY",
				ReferrerPolicy:          "strict-origin-when-cross-origin",
			},
		},
		Auth: &AuthConfig{
			Enabled:        true,
			RefreshEnabled: true,
			CacheEnabled:   true,
			TokenTTL:       "1h",
		},
		RBAC: &RBACConfig{
			Enabled:       true,
			DefaultPolicy: "DENY",
			DefaultRole:   "viewer",
		},
		RateLimit: &RateLimitConfig{
			Enabled:           true,
			RequestsPerMinute: 60,
			BurstLimit:        10,
			RateLimitByIP:     true,
		},
		CORS: &CORSConfig{
			Enabled:          false,
			AllowCredentials: false,
		},
		InputValidation: &InputValidationConfig{
			Enabled:                true,
			MaxInputLength:         10000,
			MaxOutputLength:        50000,
			StrictValidation:       true,
			EnableSchemaValidation: true,
			SanitizeInput:          true,
		},
		Audit: &AuditConfig{
			Enabled:               true,
			LogLevel:              "info",
			LogSuccessfulRequests: false,
			LogFailedRequests:     true,
			RetentionDays:         90,
		},
		Encryption: &EncryptionConfig{
			Enabled:   true,
			Algorithm: "AES-256-GCM",
			KeySize:   256,
		},
	}
}
