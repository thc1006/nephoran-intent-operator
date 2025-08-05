package auth

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/security"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled       bool                     `json:"enabled"`
	JWTSecretKey  string                   `json:"jwt_secret_key"`
	TokenTTL      time.Duration            `json:"token_ttl"`
	RefreshTTL    time.Duration            `json:"refresh_ttl"`
	Providers     map[string]ProviderConfig `json:"providers"`
	RBAC          RBACConfig               `json:"rbac"`
	AdminUsers    []string                 `json:"admin_users"`
	OperatorUsers []string                 `json:"operator_users"`
}

// ProviderConfig holds OAuth2 provider configuration
type ProviderConfig struct {
	Enabled      bool     `json:"enabled"`
	Type         string   `json:"type"` // azure-ad, okta, keycloak, google
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	TenantID     string   `json:"tenant_id,omitempty"`     // Azure AD
	Domain       string   `json:"domain,omitempty"`        // Okta
	BaseURL      string   `json:"base_url,omitempty"`      // Keycloak
	Realm        string   `json:"realm,omitempty"`         // Keycloak
	Scopes       []string `json:"scopes,omitempty"`
	AuthURL      string   `json:"auth_url,omitempty"`      // Custom
	TokenURL     string   `json:"token_url,omitempty"`     // Custom
	UserInfoURL  string   `json:"user_info_url,omitempty"` // Custom
}

// RBACConfig holds role-based access control configuration
type RBACConfig struct {
	Enabled       bool                       `json:"enabled"`
	DefaultRole   string                     `json:"default_role"`
	RoleMapping   map[string][]string        `json:"role_mapping"`   // provider_role -> internal_roles
	GroupMapping  map[string][]string        `json:"group_mapping"`  // provider_group -> internal_roles
	AdminRoles    []string                   `json:"admin_roles"`
	OperatorRoles []string                   `json:"operator_roles"`
	ReadOnlyRoles []string                   `json:"readonly_roles"`
	Permissions   map[string][]string        `json:"permissions"`    // role -> permissions
}

// Permission constants
const (
	PermissionCreateIntent    = "intent:create"
	PermissionReadIntent      = "intent:read"
	PermissionUpdateIntent    = "intent:update"
	PermissionDeleteIntent    = "intent:delete"
	PermissionManageE2Nodes   = "e2nodes:manage"
	PermissionViewMetrics     = "metrics:view"
	PermissionManageSystem    = "system:manage"
	PermissionManageUsers     = "users:manage"
	PermissionViewLogs        = "logs:view"
	PermissionManageSecrets   = "secrets:manage"
)

// LoadAuthConfig loads authentication configuration from environment and file
func LoadAuthConfig() (*AuthConfig, error) {
	// Load JWT secret key from file or env
	jwtSecretKey, _ := config.LoadJWTSecretKeyFromFile(security.GlobalAuditLogger)

	authConfig := &AuthConfig{
		Enabled:      getBoolEnv("AUTH_ENABLED", false),
		JWTSecretKey: jwtSecretKey,
		TokenTTL:     getDurationEnv("TOKEN_TTL", 24*time.Hour),
		RefreshTTL:   getDurationEnv("REFRESH_TTL", 7*24*time.Hour),
		Providers:    make(map[string]ProviderConfig),
		RBAC: RBACConfig{
			Enabled:       getBoolEnv("RBAC_ENABLED", true),
			DefaultRole:   getEnv("DEFAULT_ROLE", "viewer"),
			RoleMapping:   make(map[string][]string),
			GroupMapping:  make(map[string][]string),
			AdminRoles:    []string{"admin", "system-admin", "nephoran-admin"},
			OperatorRoles: []string{"operator", "network-operator", "telecom-operator"},
			ReadOnlyRoles: []string{"viewer", "readonly", "guest"},
			Permissions:   getDefaultPermissions(),
		},
		AdminUsers:    getStringSliceEnv("ADMIN_USERS", []string{}),
		OperatorUsers: getStringSliceEnv("OPERATOR_USERS", []string{}),
	}

	// Load provider configurations
	if err := authConfig.loadProviders(); err != nil {
		return nil, fmt.Errorf("failed to load providers: %w", err)
	}

	// Load from config file if specified
	configFile := getEnv("AUTH_CONFIG_FILE", "")
	if configFile != "" {
		if err := authConfig.loadFromFile(configFile); err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
	}

	// Validate JWT secret if auth is enabled
	if authConfig.Enabled && strings.TrimSpace(authConfig.JWTSecretKey) == "" {
		return nil, fmt.Errorf("auth enabled but JWTSecretKey is empty")
	}

	// Validate configuration
	if err := authConfig.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return authConfig, nil
}

// loadProviders loads OAuth2 provider configurations from environment
func (c *AuthConfig) loadProviders() error {
	// Azure AD provider
	if azureClientID := getEnv("AZURE_CLIENT_ID", ""); azureClientID != "" {
		c.Providers["azure-ad"] = ProviderConfig{
			Enabled:      getBoolEnv("AZURE_ENABLED", true),
			Type:         "azure-ad",
			ClientID:     azureClientID,
			ClientSecret: getOAuth2ClientSecret("azure"),
			TenantID:     getEnv("AZURE_TENANT_ID", ""),
			Scopes:       getStringSliceEnv("AZURE_SCOPES", []string{"openid", "profile", "email", "User.Read"}),
		}
	}

	// Okta provider
	if oktaClientID := getEnv("OKTA_CLIENT_ID", ""); oktaClientID != "" {
		c.Providers["okta"] = ProviderConfig{
			Enabled:      getBoolEnv("OKTA_ENABLED", true),
			Type:         "okta",
			ClientID:     oktaClientID,
			ClientSecret: getOAuth2ClientSecret("okta"),
			Domain:       getEnv("OKTA_DOMAIN", ""),
			Scopes:       getStringSliceEnv("OKTA_SCOPES", []string{"openid", "profile", "email", "groups"}),
		}
	}

	// Keycloak provider
	if keycloakClientID := getEnv("KEYCLOAK_CLIENT_ID", ""); keycloakClientID != "" {
		c.Providers["keycloak"] = ProviderConfig{
			Enabled:      getBoolEnv("KEYCLOAK_ENABLED", true),
			Type:         "keycloak",
			ClientID:     keycloakClientID,
			ClientSecret: getOAuth2ClientSecret("keycloak"),
			BaseURL:      getEnv("KEYCLOAK_BASE_URL", ""),
			Realm:        getEnv("KEYCLOAK_REALM", "master"),
			Scopes:       getStringSliceEnv("KEYCLOAK_SCOPES", []string{"openid", "profile", "email", "roles"}),
		}
	}

	// Google provider
	if googleClientID := getEnv("GOOGLE_CLIENT_ID", ""); googleClientID != "" {
		c.Providers["google"] = ProviderConfig{
			Enabled:      getBoolEnv("GOOGLE_ENABLED", true),
			Type:         "google",
			ClientID:     googleClientID,
			ClientSecret: getOAuth2ClientSecret("google"),
			Scopes:       getStringSliceEnv("GOOGLE_SCOPES", []string{"openid", "profile", "email"}),
		}
	}

	// Custom provider
	if customClientID := getEnv("CUSTOM_CLIENT_ID", ""); customClientID != "" {
		c.Providers["custom"] = ProviderConfig{
			Enabled:      getBoolEnv("CUSTOM_ENABLED", true),
			Type:         "custom",
			ClientID:     customClientID,
			ClientSecret: getOAuth2ClientSecret("custom"),
			AuthURL:      getEnv("CUSTOM_AUTH_URL", ""),
			TokenURL:     getEnv("CUSTOM_TOKEN_URL", ""),
			UserInfoURL:  getEnv("CUSTOM_USERINFO_URL", ""),
			Scopes:       getStringSliceEnv("CUSTOM_SCOPES", []string{"openid", "profile", "email"}),
		}
	}

	return nil
}

// loadFromFile loads configuration from JSON file
func (c *AuthConfig) loadFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, c)
}

// validate validates the configuration
func (c *AuthConfig) validate() error {
	if !c.Enabled {
		return nil // Skip validation if auth is disabled
	}

	if c.JWTSecretKey == "" {
		return fmt.Errorf("JWT_SECRET_KEY is required when authentication is enabled")
	}

	if len(c.JWTSecretKey) < 32 {
		return fmt.Errorf("JWT_SECRET_KEY must be at least 32 characters long")
	}

	// Validate at least one provider is configured
	hasEnabledProvider := false
	for name, provider := range c.Providers {
		if !provider.Enabled {
			continue
		}

		hasEnabledProvider = true

		if provider.ClientID == "" {
			return fmt.Errorf("client_id is required for provider %s", name)
		}

		if provider.ClientSecret == "" {
			return fmt.Errorf("client_secret is required for provider %s", name)
		}

		// Type-specific validation
		switch provider.Type {
		case "azure-ad":
			if provider.TenantID == "" {
				return fmt.Errorf("tenant_id is required for Azure AD provider")
			}
		case "okta":
			if provider.Domain == "" {
				return fmt.Errorf("domain is required for Okta provider")
			}
		case "keycloak":
			if provider.BaseURL == "" {
				return fmt.Errorf("base_url is required for Keycloak provider")
			}
		case "custom":
			if provider.AuthURL == "" || provider.TokenURL == "" || provider.UserInfoURL == "" {
				return fmt.Errorf("auth_url, token_url, and user_info_url are required for custom provider")
			}
		}
	}

	if !hasEnabledProvider {
		return fmt.Errorf("at least one OAuth2 provider must be enabled")
	}

	return nil
}

// CreateOAuth2Providers creates OAuth2 provider instances from configuration
func (c *AuthConfig) CreateOAuth2Providers() (map[string]*OAuth2Provider, error) {
	providers := make(map[string]*OAuth2Provider)

	for name, config := range c.Providers {
		if !config.Enabled {
			continue
		}

		var provider *OAuth2Provider
		switch config.Type {
		case "azure-ad":
			provider = NewAzureADProvider(config.TenantID, config.ClientID, config.ClientSecret)
		case "okta":
			provider = NewOktaProvider(config.Domain, config.ClientID, config.ClientSecret)
		case "keycloak":
			provider = NewKeycloakProvider(config.BaseURL, config.Realm, config.ClientID, config.ClientSecret)
		case "google":
			provider = NewGoogleProvider(config.ClientID, config.ClientSecret)
		case "custom":
			provider = NewOAuth2Provider(
				name,
				config.ClientID,
				config.ClientSecret,
				config.AuthURL,
				config.TokenURL,
				config.UserInfoURL,
				config.Scopes,
			)
		default:
			return nil, fmt.Errorf("unknown provider type: %s", config.Type)
		}

		if len(config.Scopes) > 0 {
			provider.Scopes = config.Scopes
		}

		providers[name] = provider
	}

	return providers, nil
}

// getDefaultPermissions returns default role-permission mapping
func getDefaultPermissions() map[string][]string {
	return map[string][]string{
		"admin": {
			PermissionCreateIntent,
			PermissionReadIntent,
			PermissionUpdateIntent,
			PermissionDeleteIntent,
			PermissionManageE2Nodes,
			PermissionViewMetrics,
			PermissionManageSystem,
			PermissionManageUsers,
			PermissionViewLogs,
			PermissionManageSecrets,
		},
		"operator": {
			PermissionCreateIntent,
			PermissionReadIntent,
			PermissionUpdateIntent,
			PermissionDeleteIntent,
			PermissionManageE2Nodes,
			PermissionViewMetrics,
			PermissionViewLogs,
		},
		"network-operator": {
			PermissionCreateIntent,
			PermissionReadIntent,
			PermissionUpdateIntent,
			PermissionManageE2Nodes,
			PermissionViewMetrics,
		},
		"viewer": {
			PermissionReadIntent,
			PermissionViewMetrics,
		},
		"readonly": {
			PermissionReadIntent,
			PermissionViewMetrics,
		},
	}
}

// Helper functions for environment variable parsing

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getStringSliceEnv(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Simple comma-separated parsing
		// In production, you might want more sophisticated parsing
		result := []string{}
		for _, item := range strings.Split(value, ",") {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultValue
}

// getOAuth2ClientSecret loads OAuth2 client secret from file or environment
func getOAuth2ClientSecret(provider string) string {
	// Create secure secret loader for OAuth2 secrets
	loader, err := config.NewSecretLoader("/secrets/oauth2", security.GlobalAuditLogger)
	if err != nil {
		// Log the error but continue with fallback
		slog.Error("Failed to create OAuth2 secret loader, falling back to environment variable", "error", err)
		envVar := strings.ToUpper(provider) + "_CLIENT_SECRET"
		return os.Getenv(envVar)
	}

	// Try to load from file first
	filename := fmt.Sprintf("%s-client-secret", strings.ToLower(provider))
	envVar := strings.ToUpper(provider) + "_CLIENT_SECRET"
	
	secret, err := loader.LoadSecretWithFallback(filename, envVar)
	if err != nil {
		slog.Error("Failed to load OAuth2 client secret", "error", err, "provider", provider)
		return ""
	}

	// Audit log for secret access (without revealing the secret)
	slog.Info("OAuth2 client secret loaded successfully", 
		"provider", provider, 
		"source", determineSecretSource(filename, envVar),
		"secret_preview", config.SanitizeForLog(secret))
	
	return secret
}

// determineSecretSource determines whether secret was loaded from file or environment
func determineSecretSource(filename, envVar string) string {
	// Check if file exists
	if _, err := os.Stat(filepath.Join("/secrets/oauth2", filename)); err == nil {
		return "file"
	}
	
	// Check if environment variable is set
	if os.Getenv(envVar) != "" {
		return "environment"
	}
	
	return "unknown"
}

// ToOAuth2Config converts AuthConfig to OAuth2Config
func (c *AuthConfig) ToOAuth2Config() (*OAuth2Config, error) {
	providers, err := c.CreateOAuth2Providers()
	if err != nil {
		return nil, err
	}

	return &OAuth2Config{
		Providers:     providers,
		DefaultScopes: []string{"openid", "profile", "email"},
		TokenTTL:      c.TokenTTL,
		RefreshTTL:    c.RefreshTTL,
	}, nil
}