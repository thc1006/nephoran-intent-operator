package config

import (
	"strings"
	"time"
	
	"github.com/thc1006/nephoran-intent-operator/pkg/interfaces"
)

// NephoranAuthConfig holds authentication configuration for Nephoran
type NephoranAuthConfig struct {
	Enabled     bool
	RequireAuth bool
	OAuth2      OAuth2Config
	LDAP        LDAPConfig
	JWT         JWTConfig
	RBAC        RBACConfig
	Sessions    SessionConfig
	Security    *SecurityConfig
	Audit       AuditConfig
	Controller  ControllerAuthConfig
	Endpoints   EndpointConfig
}

// OAuth2Config holds OAuth2 provider configuration
type OAuth2Config struct {
	Enabled         bool
	Providers       []OAuth2Provider
	DefaultProvider string
	RedirectURL     string
}

// OAuth2Provider represents an OAuth2 provider
type OAuth2Provider struct {
	Name         string
	Type         string
	ClientID     string
	ClientSecret string
	TenantID     string
	Enabled      bool
}

// LDAPConfig holds LDAP configuration
type LDAPConfig struct {
	Enabled bool
	Host    string
	Port    int
	BaseDN  string
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	SecretKey  string
	TokenTTL   time.Duration
	RefreshTTL time.Duration
}

// RBACConfig holds RBAC configuration
type RBACConfig struct {
	Enabled     bool
	DefaultRole string
}

// SessionConfig holds session configuration
type SessionConfig struct {
	Enabled       bool
	Timeout       time.Duration
	MaxConcurrent int
}

// SecurityConfig holds security settings - use common config type
type SecurityConfig = interfaces.CommonSecurityConfig

// AuditConfig holds audit configuration
type AuditConfig struct {
	Enabled       bool
	LogLevel      string
	RetentionDays int
}

// ControllerAuthConfig configures controller authentication
type ControllerAuthConfig struct {
	Enabled                bool
	RequireUserContext     bool
	ValidatePermissions    bool
	AuditControllerActions bool
}

// EndpointConfig configures endpoint authentication
type EndpointConfig struct {
	PublicEndpoints []string
}

// LoadNephoranAuthConfig loads auth configuration from environment
func LoadNephoranAuthConfig() (*NephoranAuthConfig, error) {
	return &NephoranAuthConfig{
		Enabled:     GetBoolEnv("NEPHORAN_AUTH_ENABLED", true),
		RequireAuth: GetBoolEnv("NEPHORAN_REQUIRE_AUTH", true),
		OAuth2: OAuth2Config{
			Enabled:         GetBoolEnv("NEPHORAN_OAUTH2_ENABLED", true),
			DefaultProvider: GetEnvOrDefault("NEPHORAN_OAUTH2_DEFAULT_PROVIDER", "github"),
			RedirectURL:     GetEnvOrDefault("NEPHORAN_OAUTH2_REDIRECT_URL", "http://localhost:8080/auth/callback"),
		},
		LDAP: LDAPConfig{
			Enabled: GetBoolEnv("NEPHORAN_LDAP_ENABLED", false),
			Host:    GetEnvOrDefault("NEPHORAN_LDAP_HOST", ""),
			Port:    GetIntEnv("NEPHORAN_LDAP_PORT", 389),
			BaseDN:  GetEnvOrDefault("NEPHORAN_LDAP_BASE_DN", ""),
		},
		JWT: JWTConfig{
			SecretKey:  GetEnvOrDefault("NEPHORAN_JWT_SECRET_KEY", ""),
			TokenTTL:   GetDurationEnv("NEPHORAN_JWT_TOKEN_TTL", 1*time.Hour),
			RefreshTTL: GetDurationEnv("NEPHORAN_JWT_REFRESH_TTL", 24*time.Hour),
		},
		RBAC: RBACConfig{
			Enabled:     GetBoolEnv("NEPHORAN_RBAC_ENABLED", true),
			DefaultRole: GetEnvOrDefault("NEPHORAN_RBAC_DEFAULT_ROLE", "nephoran-viewer"),
		},
		Sessions: SessionConfig{
			Enabled:       GetBoolEnv("NEPHORAN_SESSIONS_ENABLED", true),
			Timeout:       GetDurationEnv("NEPHORAN_SESSIONS_TIMEOUT", 30*time.Minute),
			MaxConcurrent: GetIntEnv("NEPHORAN_SESSIONS_MAX_CONCURRENT", 100),
		},
		Security: &SecurityConfig{
			SecurityHeaders: &interfaces.SecurityHeadersConfig{
				Enabled: GetBoolEnv("NEPHORAN_SECURITY_CSRF_ENABLED", true),
			},
			TLS: &interfaces.TLSConfig{
				Enabled: GetBoolEnv("NEPHORAN_SECURITY_REQUIRE_HTTPS", false),
			},
		},
		Audit: AuditConfig{
			Enabled:       GetBoolEnv("NEPHORAN_AUDIT_ENABLED", true),
			LogLevel:      GetEnvOrDefault("NEPHORAN_AUDIT_LOG_LEVEL", "info"),
			RetentionDays: GetIntEnv("NEPHORAN_AUDIT_RETENTION_DAYS", 90),
		},
		Controller: ControllerAuthConfig{
			Enabled:                GetBoolEnv("NEPHORAN_CONTROLLER_AUTH_ENABLED", true),
			RequireUserContext:     GetBoolEnv("NEPHORAN_CONTROLLER_REQUIRE_USER_CONTEXT", false),
			ValidatePermissions:    GetBoolEnv("NEPHORAN_CONTROLLER_VALIDATE_PERMISSIONS", true),
			AuditControllerActions: GetBoolEnv("NEPHORAN_CONTROLLER_AUDIT_ACTIONS", true),
		},
		Endpoints: EndpointConfig{
			PublicEndpoints: strings.Split(GetEnvOrDefault("NEPHORAN_ENDPOINTS_PUBLIC", "/healthz,/readyz,/metrics"), ","),
		},
	}, nil
}
