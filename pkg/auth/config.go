package auth

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/auth/providers"
	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/security"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled       bool                          `json:"enabled"`
	JWTSecretKey  string                        `json:"jwt_secret_key"`
	TokenTTL      time.Duration                 `json:"token_ttl"`
	RefreshTTL    time.Duration                 `json:"refresh_ttl"`
	Providers     map[string]ProviderConfig     `json:"providers"`
	LDAPProviders map[string]LDAPProviderConfig `json:"ldap_providers"`
	RBAC          RBACConfig                    `json:"rbac"`
	AdminUsers    []string                      `json:"admin_users"`
	OperatorUsers []string                      `json:"operator_users"`
}

// ProviderConfig holds OAuth2 provider configuration
type ProviderConfig struct {
	Enabled      bool     `json:"enabled"`
	Type         string   `json:"type"` // azure-ad, okta, keycloak, google, ldap
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	TenantID     string   `json:"tenant_id,omitempty"` // Azure AD
	Domain       string   `json:"domain,omitempty"`    // Okta
	BaseURL      string   `json:"base_url,omitempty"`  // Keycloak
	Realm        string   `json:"realm,omitempty"`     // Keycloak
	Scopes       []string `json:"scopes,omitempty"`
	AuthURL      string   `json:"auth_url,omitempty"`      // Custom
	TokenURL     string   `json:"token_url,omitempty"`     // Custom
	UserInfoURL  string   `json:"user_info_url,omitempty"` // Custom
}

// LDAPProviderConfig holds LDAP provider configuration
type LDAPProviderConfig struct {
	Enabled              bool                 `json:"enabled"`
	Host                 string               `json:"host"`
	Port                 int                  `json:"port"`
	UseSSL               bool                 `json:"use_ssl"`
	UseTLS               bool                 `json:"use_tls"`
	SkipVerify           bool                 `json:"skip_verify"`
	BindDN               string               `json:"bind_dn"`
	BindPassword         string               `json:"bind_password"`
	BaseDN               string               `json:"base_dn"`
	UserFilter           string               `json:"user_filter"`
	GroupFilter          string               `json:"group_filter"`
	UserSearchBase       string               `json:"user_search_base"`
	GroupSearchBase      string               `json:"group_search_base"`
	UserAttributes       LDAPAttributeMapping `json:"user_attributes"`
	GroupAttributes      LDAPAttributeMapping `json:"group_attributes"`
	AuthMethod           string               `json:"auth_method"`
	Timeout              time.Duration        `json:"timeout"`
	ConnectionTimeout    time.Duration        `json:"connection_timeout"`
	MaxConnections       int                  `json:"max_connections"`
	IsActiveDirectory    bool                 `json:"is_active_directory"`
	Domain               string               `json:"domain"`
	GroupMemberAttribute string               `json:"group_member_attribute"`
	UserGroupAttribute   string               `json:"user_group_attribute"`
	RoleMappings         map[string][]string  `json:"role_mappings"`
	DefaultRoles         []string             `json:"default_roles"`
}

// LDAPAttributeMapping maps LDAP attributes to standard fields
type LDAPAttributeMapping struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	DisplayName string `json:"display_name"`
	Groups      string `json:"groups"`
	Title       string `json:"title"`
	Department  string `json:"department"`
	Phone       string `json:"phone"`
	Manager     string `json:"manager"`
}

// RBACConfig holds role-based access control configuration
type RBACConfig struct {
	Enabled       bool                `json:"enabled"`
	DefaultRole   string              `json:"default_role"`
	RoleMapping   map[string][]string `json:"role_mapping"`  // provider_role -> internal_roles
	GroupMapping  map[string][]string `json:"group_mapping"` // provider_group -> internal_roles
	AdminRoles    []string            `json:"admin_roles"`
	OperatorRoles []string            `json:"operator_roles"`
	ReadOnlyRoles []string            `json:"readonly_roles"`
	Permissions   map[string][]string `json:"permissions"` // role -> permissions
}

// Configuration constants for security limits and validation
const (
	// Secret file size limits
	MaxSecretFileSize     = 64 * 1024 // 64KB limit for secret files
	MinSecretLength       = 16        // Minimum secret length for security
	MaxSecretLength       = 512       // Maximum secret length to prevent DoS
	MaxProviderNameLength = 50        // Maximum provider name length
	MinJWTSecretLength    = 32        // Minimum JWT secret length

	// Default configuration values
	DefaultKeycloakRealm = "master"         // Default Keycloak realm
	DefaultTokenTTL      = 24 * 60 * 60     // 24 hours in seconds
	DefaultRefreshTTL    = 7 * 24 * 60 * 60 // 7 days in seconds
)

// Permission constants
const (
	PermissionCreateIntent  = "intent:create"
	PermissionReadIntent    = "intent:read"
	PermissionUpdateIntent  = "intent:update"
	PermissionDeleteIntent  = "intent:delete"
	PermissionManageE2Nodes = "e2nodes:manage"
	PermissionViewMetrics   = "metrics:view"
	PermissionManageSystem  = "system:manage"
	PermissionManageUsers   = "users:manage"
	PermissionViewLogs      = "logs:view"
	PermissionManageSecrets = "secrets:manage"
)

// LoadAuthConfig loads authentication configuration from environment and file
// configPath: Path to the auth config file. If empty, falls back to AUTH_CONFIG_FILE env var
func LoadAuthConfig(configPath string) (*AuthConfig, error) {
	// Load JWT secret key from file or env
	jwtSecretKey, _ := config.LoadJWTSecretKeyFromFile(security.GlobalAuditLogger)

	authConfig := &AuthConfig{
		Enabled:       config.GetBoolEnv("AUTH_ENABLED", false),
		JWTSecretKey:  jwtSecretKey,
		TokenTTL:      config.GetDurationEnv("TOKEN_TTL", 24*time.Hour),
		RefreshTTL:    config.GetDurationEnv("REFRESH_TTL", 7*24*time.Hour),
		Providers:     make(map[string]ProviderConfig),
		LDAPProviders: make(map[string]LDAPProviderConfig),
		RBAC: RBACConfig{
			Enabled:       config.GetBoolEnv("RBAC_ENABLED", true),
			DefaultRole:   config.GetEnvOrDefault("DEFAULT_ROLE", "viewer"),
			RoleMapping:   make(map[string][]string),
			GroupMapping:  make(map[string][]string),
			AdminRoles:    []string{"admin", "system-admin", "nephoran-admin"},
			OperatorRoles: []string{"operator", "network-operator", "telecom-operator"},
			ReadOnlyRoles: []string{"viewer", "readonly", "guest"},
			Permissions:   getDefaultPermissions(),
		},
		AdminUsers:    config.GetStringSliceEnv("ADMIN_USERS", []string{}),
		OperatorUsers: config.GetStringSliceEnv("OPERATOR_USERS", []string{}),
	}

	// Load provider configurations
	if err := authConfig.loadProviders(); err != nil {
		return nil, fmt.Errorf("failed to load providers: %w", err)
	}

	// Load LDAP provider configurations
	if err := authConfig.loadLDAPProviders(); err != nil {
		return nil, fmt.Errorf("failed to load LDAP providers: %w", err)
	}

	// Determine config file path: use provided path or fall back to environment variable
	configFile := configPath
	if configFile == "" {
		configFile = config.GetEnvOrDefault("AUTH_CONFIG_FILE", "")
	}

	// Load from config file if specified
	if configFile != "" {
		if err := authConfig.loadFromFile(configFile); err != nil {
			return nil, fmt.Errorf("failed to load config file %q: %w", configFile, err)
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

// loadProviders loads OAuth2 provider configurations from environment with secure error handling
func (c *AuthConfig) loadProviders() error {
	var errors []error

	// Azure AD provider
	if azureClientID := config.GetEnvOrDefault("AZURE_CLIENT_ID", ""); azureClientID != "" {
		secret, err := getOAuth2ClientSecret("azure")
		if err != nil && config.GetBoolEnv("AZURE_ENABLED", true) {
			errors = append(errors, fmt.Errorf("azure-ad: %w", err))
		}

		c.Providers["azure-ad"] = ProviderConfig{
			Enabled:      config.GetBoolEnv("AZURE_ENABLED", true),
			Type:         "azure-ad",
			ClientID:     azureClientID,
			ClientSecret: secret,
			TenantID:     config.GetEnvOrDefault("AZURE_TENANT_ID", ""),
			Scopes:       config.GetStringSliceEnv("AZURE_SCOPES", []string{"openid", "profile", "email", "User.Read"}),
		}
	}

	// Okta provider
	if oktaClientID := config.GetEnvOrDefault("OKTA_CLIENT_ID", ""); oktaClientID != "" {
		secret, err := getOAuth2ClientSecret("okta")
		if err != nil && config.GetBoolEnv("OKTA_ENABLED", true) {
			errors = append(errors, fmt.Errorf("okta: %w", err))
		}

		c.Providers["okta"] = ProviderConfig{
			Enabled:      config.GetBoolEnv("OKTA_ENABLED", true),
			Type:         "okta",
			ClientID:     oktaClientID,
			ClientSecret: secret,
			Domain:       config.GetEnvOrDefault("OKTA_DOMAIN", ""),
			Scopes:       config.GetStringSliceEnv("OKTA_SCOPES", []string{"openid", "profile", "email", "groups"}),
		}
	}

	// Keycloak provider
	if keycloakClientID := config.GetEnvOrDefault("KEYCLOAK_CLIENT_ID", ""); keycloakClientID != "" {
		secret, err := getOAuth2ClientSecret("keycloak")
		if err != nil && config.GetBoolEnv("KEYCLOAK_ENABLED", true) {
			errors = append(errors, fmt.Errorf("keycloak: %w", err))
		}

		c.Providers["keycloak"] = ProviderConfig{
			Enabled:      config.GetBoolEnv("KEYCLOAK_ENABLED", true),
			Type:         "keycloak",
			ClientID:     keycloakClientID,
			ClientSecret: secret,
			BaseURL:      config.GetEnvOrDefault("KEYCLOAK_BASE_URL", ""),
			Realm:        config.GetEnvOrDefault("KEYCLOAK_REALM", "master"),
			Scopes:       config.GetStringSliceEnv("KEYCLOAK_SCOPES", []string{"openid", "profile", "email", "roles"}),
		}
	}

	// Google provider
	if googleClientID := config.GetEnvOrDefault("GOOGLE_CLIENT_ID", ""); googleClientID != "" {
		secret, err := getOAuth2ClientSecret("google")
		if err != nil && config.GetBoolEnv("GOOGLE_ENABLED", true) {
			errors = append(errors, fmt.Errorf("google: %w", err))
		}

		c.Providers["google"] = ProviderConfig{
			Enabled:      config.GetBoolEnv("GOOGLE_ENABLED", true),
			Type:         "google",
			ClientID:     googleClientID,
			ClientSecret: secret,
			Scopes:       config.GetStringSliceEnv("GOOGLE_SCOPES", []string{"openid", "profile", "email"}),
		}
	}

	// Custom provider
	if customClientID := config.GetEnvOrDefault("CUSTOM_CLIENT_ID", ""); customClientID != "" {
		secret, err := getOAuth2ClientSecret("custom")
		if err != nil && config.GetBoolEnv("CUSTOM_ENABLED", true) {
			errors = append(errors, fmt.Errorf("custom: %w", err))
		}

		c.Providers["custom"] = ProviderConfig{
			Enabled:      config.GetBoolEnv("CUSTOM_ENABLED", true),
			Type:         "custom",
			ClientID:     customClientID,
			ClientSecret: secret,
			AuthURL:      config.GetEnvOrDefault("CUSTOM_AUTH_URL", ""),
			TokenURL:     config.GetEnvOrDefault("CUSTOM_TOKEN_URL", ""),
			UserInfoURL:  config.GetEnvOrDefault("CUSTOM_USERINFO_URL", ""),
			Scopes:       config.GetStringSliceEnv("CUSTOM_SCOPES", []string{"openid", "profile", "email"}),
		}
	}

	// If there are errors for enabled providers, aggregate them
	if len(errors) > 0 {
		// Log individual errors for debugging
		for _, err := range errors {
			slog.Error("Provider configuration error", "error", err)
		}

		// Return aggregated error
		if len(errors) == 1 {
			return errors[0]
		}
		return fmt.Errorf("multiple provider configuration errors: %d providers failed", len(errors))
	}

	return nil
}

// loadLDAPProviders loads LDAP provider configurations from environment with secure error handling
func (c *AuthConfig) loadLDAPProviders() error {
	var errors []error

	// LDAP provider (primary)
	if ldapHost := config.GetEnvOrDefault("LDAP_HOST", ""); ldapHost != "" {
		bindPassword := config.GetEnvOrDefault("LDAP_BIND_PASSWORD", "")
		if bindPassword == "" {
			// Try to load from file
			if passwordFile := config.GetEnvOrDefault("LDAP_BIND_PASSWORD_FILE", ""); passwordFile != "" {
				if password, err := readSecretFile(passwordFile); err == nil {
					bindPassword = password
				} else {
					errors = append(errors, fmt.Errorf("ldap: failed to read bind password file: %w", err))
				}
			}
		}

		c.LDAPProviders["ldap"] = LDAPProviderConfig{
			Enabled:         config.GetBoolEnv("LDAP_ENABLED", true),
			Host:            ldapHost,
			Port:            config.GetIntEnv("LDAP_PORT", 389),
			UseSSL:          config.GetBoolEnv("LDAP_USE_SSL", false),
			UseTLS:          config.GetBoolEnv("LDAP_USE_TLS", true),
			SkipVerify:      config.GetBoolEnv("LDAP_SKIP_VERIFY", false),
			BindDN:          config.GetEnvOrDefault("LDAP_BIND_DN", ""),
			BindPassword:    bindPassword,
			BaseDN:          config.GetEnvOrDefault("LDAP_BASE_DN", ""),
			UserFilter:      config.GetEnvOrDefault("LDAP_USER_FILTER", ""),
			GroupFilter:     config.GetEnvOrDefault("LDAP_GROUP_FILTER", ""),
			UserSearchBase:  config.GetEnvOrDefault("LDAP_USER_SEARCH_BASE", ""),
			GroupSearchBase: config.GetEnvOrDefault("LDAP_GROUP_SEARCH_BASE", ""),
			UserAttributes: LDAPAttributeMapping{
				Username:    config.GetEnvOrDefault("LDAP_USER_ATTR_USERNAME", ""),
				Email:       config.GetEnvOrDefault("LDAP_USER_ATTR_EMAIL", ""),
				FirstName:   config.GetEnvOrDefault("LDAP_USER_ATTR_FIRST_NAME", ""),
				LastName:    config.GetEnvOrDefault("LDAP_USER_ATTR_LAST_NAME", ""),
				DisplayName: config.GetEnvOrDefault("LDAP_USER_ATTR_DISPLAY_NAME", ""),
				Title:       config.GetEnvOrDefault("LDAP_USER_ATTR_TITLE", ""),
				Department:  config.GetEnvOrDefault("LDAP_USER_ATTR_DEPARTMENT", ""),
				Phone:       config.GetEnvOrDefault("LDAP_USER_ATTR_PHONE", ""),
				Manager:     config.GetEnvOrDefault("LDAP_USER_ATTR_MANAGER", ""),
			},
			AuthMethod:           config.GetEnvOrDefault("LDAP_AUTH_METHOD", "simple"),
			Timeout:              config.GetDurationEnv("LDAP_TIMEOUT", 30*time.Second),
			ConnectionTimeout:    config.GetDurationEnv("LDAP_CONNECTION_TIMEOUT", 10*time.Second),
			MaxConnections:       config.GetIntEnv("LDAP_MAX_CONNECTIONS", 10),
			IsActiveDirectory:    config.GetBoolEnv("LDAP_IS_ACTIVE_DIRECTORY", false),
			Domain:               config.GetEnvOrDefault("LDAP_DOMAIN", ""),
			GroupMemberAttribute: config.GetEnvOrDefault("LDAP_GROUP_MEMBER_ATTR", ""),
			UserGroupAttribute:   config.GetEnvOrDefault("LDAP_USER_GROUP_ATTR", ""),
			RoleMappings:         parseRoleMappings(config.GetEnvOrDefault("LDAP_ROLE_MAPPINGS", "")),
			DefaultRoles:         config.GetStringSliceEnv("LDAP_DEFAULT_ROLES", []string{"user"}),
		}
	}

	// Active Directory provider (alternative configuration)
	if adHost := config.GetEnvOrDefault("AD_HOST", ""); adHost != "" {
		bindPassword := config.GetEnvOrDefault("AD_BIND_PASSWORD", "")
		if bindPassword == "" {
			// Try to load from file
			if passwordFile := config.GetEnvOrDefault("AD_BIND_PASSWORD_FILE", ""); passwordFile != "" {
				if password, err := readSecretFile(passwordFile); err == nil {
					bindPassword = password
				} else {
					errors = append(errors, fmt.Errorf("active-directory: failed to read bind password file: %w", err))
				}
			}
		}

		c.LDAPProviders["active-directory"] = LDAPProviderConfig{
			Enabled:         config.GetBoolEnv("AD_ENABLED", true),
			Host:            adHost,
			Port:            config.GetIntEnv("AD_PORT", 389),
			UseSSL:          config.GetBoolEnv("AD_USE_SSL", false),
			UseTLS:          config.GetBoolEnv("AD_USE_TLS", true),
			SkipVerify:      config.GetBoolEnv("AD_SKIP_VERIFY", false),
			BindDN:          config.GetEnvOrDefault("AD_BIND_DN", ""),
			BindPassword:    bindPassword,
			BaseDN:          config.GetEnvOrDefault("AD_BASE_DN", ""),
			UserFilter:      config.GetEnvOrDefault("AD_USER_FILTER", ""),
			GroupFilter:     config.GetEnvOrDefault("AD_GROUP_FILTER", ""),
			UserSearchBase:  config.GetEnvOrDefault("AD_USER_SEARCH_BASE", ""),
			GroupSearchBase: config.GetEnvOrDefault("AD_GROUP_SEARCH_BASE", ""),
			UserAttributes: LDAPAttributeMapping{
				Username:    config.GetEnvOrDefault("AD_USER_ATTR_USERNAME", ""),
				Email:       config.GetEnvOrDefault("AD_USER_ATTR_EMAIL", ""),
				FirstName:   config.GetEnvOrDefault("AD_USER_ATTR_FIRST_NAME", ""),
				LastName:    config.GetEnvOrDefault("AD_USER_ATTR_LAST_NAME", ""),
				DisplayName: config.GetEnvOrDefault("AD_USER_ATTR_DISPLAY_NAME", ""),
				Title:       config.GetEnvOrDefault("AD_USER_ATTR_TITLE", ""),
				Department:  config.GetEnvOrDefault("AD_USER_ATTR_DEPARTMENT", ""),
				Phone:       config.GetEnvOrDefault("AD_USER_ATTR_PHONE", ""),
				Manager:     config.GetEnvOrDefault("AD_USER_ATTR_MANAGER", ""),
			},
			AuthMethod:           config.GetEnvOrDefault("AD_AUTH_METHOD", "simple"),
			Timeout:              config.GetDurationEnv("AD_TIMEOUT", 30*time.Second),
			ConnectionTimeout:    config.GetDurationEnv("AD_CONNECTION_TIMEOUT", 10*time.Second),
			MaxConnections:       config.GetIntEnv("AD_MAX_CONNECTIONS", 10),
			IsActiveDirectory:    true, // Always true for AD provider
			Domain:               config.GetEnvOrDefault("AD_DOMAIN", ""),
			GroupMemberAttribute: config.GetEnvOrDefault("AD_GROUP_MEMBER_ATTR", ""),
			UserGroupAttribute:   config.GetEnvOrDefault("AD_USER_GROUP_ATTR", ""),
			RoleMappings:         parseRoleMappings(config.GetEnvOrDefault("AD_ROLE_MAPPINGS", "")),
			DefaultRoles:         config.GetStringSliceEnv("AD_DEFAULT_ROLES", []string{"user"}),
		}
	}

	// Log warnings for any configuration errors but don't fail
	if len(errors) > 0 {
		for _, err := range errors {
			slog.Warn("LDAP provider configuration warning", "error", err)
		}
	}

	return nil
}

// parseRoleMappings parses role mappings from environment string format
// Expected format: "group1:role1,role2;group2:role3,role4"
func parseRoleMappings(mappingStr string) map[string][]string {
	mappings := make(map[string][]string)
	if mappingStr == "" {
		return mappings
	}

	// Split by semicolon for different groups
	groupMappings := strings.Split(mappingStr, ";")
	for _, groupMapping := range groupMappings {
		groupMapping = strings.TrimSpace(groupMapping)
		if groupMapping == "" {
			continue
		}

		// Split by colon to separate group and roles
		parts := strings.SplitN(groupMapping, ":", 2)
		if len(parts) != 2 {
			continue
		}

		group := strings.TrimSpace(parts[0])
		rolesStr := strings.TrimSpace(parts[1])

		if group != "" && rolesStr != "" {
			// Split roles by comma
			roles := strings.Split(rolesStr, ",")
			for i, role := range roles {
				roles[i] = strings.TrimSpace(role)
			}
			mappings[group] = roles
		}
	}

	return mappings
}

// loadFromFile loads configuration from JSON file with comprehensive security validation
func (c *AuthConfig) loadFromFile(filename string) error {
	// SECURITY: Validate config file path to prevent path traversal attacks
	if err := validateConfigFilePath(filename); err != nil {
		// Log security event for audit trail
		slog.Error("Config file path validation failed",
			"file", filename,
			"error", err,
			"event", "config_path_traversal_attempt")
		if security.GlobalAuditLogger != nil {
			security.GlobalAuditLogger.LogSecretAccess(
				"auth_config",
				"file",
				filename,
				"",
				false,
				fmt.Errorf("path validation failed: %w", err))
		}
		return fmt.Errorf("invalid config file path: %w", err)
	}

	// SECURITY: Check file metadata before reading
	info, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config file does not exist: %s", filename)
		}
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied accessing config file: %s", filename)
		}
		return fmt.Errorf("cannot access config file: %w", err)
	}

	// SECURITY: Prevent reading extremely large files (potential DoS)
	const maxConfigFileSize = 10 * 1024 * 1024 // 10MB limit for config files
	if info.Size() > maxConfigFileSize {
		return fmt.Errorf("config file too large (max %d bytes): %d bytes", maxConfigFileSize, info.Size())
	}

	if info.Size() == 0 {
		return fmt.Errorf("config file is empty")
	}

	// SECURITY: Check if file is a symlink and resolve it safely
	if info.Mode()&os.ModeSymlink != 0 {
		// Resolve symlink and validate the target
		realPath, err := filepath.EvalSymlinks(filename)
		if err != nil {
			return fmt.Errorf("cannot resolve symlink: %w", err)
		}
		// Re-validate the resolved path
		if err := validateConfigFilePath(realPath); err != nil {
			slog.Error("Symlink target validation failed",
				"symlink", filename,
				"target", realPath,
				"error", err,
				"event", "config_symlink_traversal_attempt")
			return fmt.Errorf("symlink target validation failed: %w", err)
		}
		filename = realPath
	}

	// SECURITY: Warn about overly permissive file permissions
	mode := info.Mode()
	if mode&0o044 != 0 { // Check if world or group readable
		slog.Warn("Config file has overly permissive permissions",
			"file", filename,
			"mode", mode.String(),
			"recommended", "0600 or 0640")
	}

	// Read file contents with security checks passed
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// SECURITY: Validate JSON structure before unmarshaling to prevent malformed input attacks
	if !json.Valid(data) {
		return fmt.Errorf("config file contains invalid JSON")
	}

	// Parse JSON with size-limited decoder to prevent memory exhaustion
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields() // Strict parsing - reject unknown fields

	if err := decoder.Decode(c); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Log successful config load for audit trail
	slog.Info("Auth config loaded from file",
		"file", filename,
		"size", info.Size(),
		"providers_count", len(c.Providers),
		"ldap_providers_count", len(c.LDAPProviders))

	return nil
}

// validate validates the configuration with enhanced security checks
func (c *AuthConfig) validate() error {
	if !c.Enabled {
		return nil // Skip validation if auth is disabled
	}

	// Validate JWT secret key
	if c.JWTSecretKey == "" {
		return fmt.Errorf("JWT_SECRET_KEY is required when authentication is enabled")
	}

	if len(c.JWTSecretKey) < MinJWTSecretLength {
		return fmt.Errorf("JWT_SECRET_KEY must be at least %d characters long for security", MinJWTSecretLength)
	}

	// Check for common weak JWT secrets
	if err := validateJWTSecret(c.JWTSecretKey); err != nil {
		return fmt.Errorf("JWT_SECRET_KEY validation failed: %w", err)
	}

	// Validate at least one provider is configured
	hasEnabledProvider := false
	providerErrors := make(map[string][]string)

	// Check OAuth2 providers

	for name, provider := range c.Providers {
		if !provider.Enabled {
			continue
		}

		hasEnabledProvider = true
		var errs []string

		// Basic validation
		if provider.ClientID == "" {
			errs = append(errs, "client_id is required")
		}

		// Enhanced secret validation
		if provider.ClientSecret == "" {
			// Check if this is a loading error or truly empty
			// The error would have been logged during loadProviders
			errs = append(errs, "client_secret is required but not available")
		} else {
			// Validate the secret format if it exists
			if err := validateOAuth2ClientSecret(name, provider.ClientSecret); err != nil {
				errs = append(errs, fmt.Sprintf("client_secret validation failed: %v", err))
			}
		}

		// Type-specific validation
		switch provider.Type {
		case "azure-ad":
			if provider.TenantID == "" {
				errs = append(errs, "tenant_id is required for Azure AD")
			}
		case "okta":
			if provider.Domain == "" {
				errs = append(errs, "domain is required for Okta")
			}
		case "keycloak":
			if provider.BaseURL == "" {
				errs = append(errs, "base_url is required for Keycloak")
			}
			if provider.Realm == "" {
				// Update the provider in the map with default realm
				updatedProvider := provider
				updatedProvider.Realm = DefaultKeycloakRealm
				c.Providers[name] = updatedProvider
			}
		case "custom":
			if provider.AuthURL == "" {
				errs = append(errs, "auth_url is required for custom provider")
			}
			if provider.TokenURL == "" {
				errs = append(errs, "token_url is required for custom provider")
			}
			if provider.UserInfoURL == "" {
				errs = append(errs, "user_info_url is required for custom provider")
			}
		}

		if len(errs) > 0 {
			providerErrors["oauth2:"+name] = errs
		}
	}

	// Check LDAP providers
	for name, provider := range c.LDAPProviders {
		if !provider.Enabled {
			continue
		}

		hasEnabledProvider = true
		var errs []string

		// Basic LDAP validation
		if provider.Host == "" {
			errs = append(errs, "host is required")
		}
		if provider.BaseDN == "" {
			errs = append(errs, "base_dn is required")
		}
		if provider.IsActiveDirectory && provider.Domain == "" {
			errs = append(errs, "domain is required for Active Directory")
		}

		if len(errs) > 0 {
			providerErrors["ldap:"+name] = errs
		}
	}

	// Report validation errors for all providers
	if len(providerErrors) > 0 {
		var errorMsgs []string
		for provider, errs := range providerErrors {
			errorMsgs = append(errorMsgs, fmt.Sprintf("%s: %s", provider, strings.Join(errs, ", ")))
		}
		return fmt.Errorf("provider validation failed: %s", strings.Join(errorMsgs, "; "))
	}

	if !hasEnabledProvider {
		return fmt.Errorf("at least one provider (OAuth2 or LDAP) must be enabled")
	}

	return nil
}

// validateJWTSecret checks JWT secret for common weak values
func validateJWTSecret(secret string) error {
	// Check for common weak secrets
	weakSecrets := []string{
		"secret", "changeme", "password", "12345678",
		"default", "admin", "test", "demo",
	}

	lowerSecret := strings.ToLower(secret)
	for _, weak := range weakSecrets {
		if lowerSecret == weak || strings.Contains(lowerSecret, weak) {
			return fmt.Errorf("weak or default secret detected")
		}
	}

	// Check for repetitive patterns
	if len(secret) > 4 {
		firstChar := secret[0]
		allSame := true
		for i := 1; i < len(secret); i++ {
			if secret[i] != firstChar {
				allSame = false
				break
			}
		}
		if allSame {
			return fmt.Errorf("repetitive pattern detected")
		}
	}

	return nil
}

// CreateOAuth2Providers creates OAuth2 provider instances from configuration with validation
func (c *AuthConfig) CreateOAuth2Providers() (map[string]*OAuth2Provider, error) {
	providers := make(map[string]*OAuth2Provider)
	var errors []error

	for name, config := range c.Providers {
		if !config.Enabled {
			continue
		}

		// Validate configuration before creating provider
		if config.ClientID == "" {
			errors = append(errors, fmt.Errorf("%s: client_id is empty", name))
			continue
		}

		if config.ClientSecret == "" {
			errors = append(errors, fmt.Errorf("%s: client_secret is empty", name))
			continue
		}

		// Additional validation for secret format
		if err := validateOAuth2ClientSecret(strings.TrimSuffix(name, "-ad"), config.ClientSecret); err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", name, err))
			continue
		}

		var provider *OAuth2Provider
		switch config.Type {
		case "azure-ad":
			if config.TenantID == "" {
				errors = append(errors, fmt.Errorf("%s: tenant_id is required", name))
				continue
			}
			provider = NewAzureADProvider(config.TenantID, config.ClientID, config.ClientSecret)

		case "okta":
			if config.Domain == "" {
				errors = append(errors, fmt.Errorf("%s: domain is required", name))
				continue
			}
			provider = NewOktaProvider(config.Domain, config.ClientID, config.ClientSecret)

		case "keycloak":
			if config.BaseURL == "" {
				errors = append(errors, fmt.Errorf("%s: base_url is required", name))
				continue
			}
			provider = NewKeycloakProvider(config.BaseURL, config.Realm, config.ClientID, config.ClientSecret)

		case "google":
			provider = NewGoogleProvider(config.ClientID, config.ClientSecret)

		case "custom":
			if config.AuthURL == "" || config.TokenURL == "" || config.UserInfoURL == "" {
				errors = append(errors, fmt.Errorf("%s: auth_url, token_url, and user_info_url are required", name))
				continue
			}
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
			errors = append(errors, fmt.Errorf("%s: unknown provider type: %s", name, config.Type))
			continue
		}

		// Apply custom scopes if specified
		if len(config.Scopes) > 0 {
			provider.Scopes = config.Scopes
		}

		providers[name] = provider

		// Log successful provider creation
		slog.Info("OAuth2 provider created successfully",
			"provider", name,
			"type", config.Type,
			"scopes", config.Scopes)
	}

	// If no providers were successfully created and there were errors, return error
	if len(providers) == 0 && len(errors) > 0 {
		return nil, fmt.Errorf("failed to create any OAuth2 providers: %v", errors)
	}

	// Log warnings for failed providers but continue if at least one succeeded
	if len(errors) > 0 {
		for _, err := range errors {
			slog.Warn("Failed to create OAuth2 provider", "error", err)
		}
	}

	return providers, nil
}

// CreateLDAPProviders creates LDAP provider instances from configuration with validation
func (c *AuthConfig) CreateLDAPProviders() (map[string]providers.LDAPProvider, error) {
	ldapProviders := make(map[string]providers.LDAPProvider)
	var errors []error

	// Create logger for LDAP providers
	logger := slog.Default()

	for name, ldapConfig := range c.LDAPProviders {
		if !ldapConfig.Enabled {
			continue
		}

		// Convert config format
		providerConfig := &providers.LDAPConfig{
			Host:                 ldapConfig.Host,
			Port:                 ldapConfig.Port,
			UseSSL:               ldapConfig.UseSSL,
			UseTLS:               ldapConfig.UseTLS,
			SkipVerify:           ldapConfig.SkipVerify,
			BindDN:               ldapConfig.BindDN,
			BindPassword:         ldapConfig.BindPassword,
			BaseDN:               ldapConfig.BaseDN,
			UserFilter:           ldapConfig.UserFilter,
			GroupFilter:          ldapConfig.GroupFilter,
			UserSearchBase:       ldapConfig.UserSearchBase,
			GroupSearchBase:      ldapConfig.GroupSearchBase,
			AuthMethod:           ldapConfig.AuthMethod,
			Timeout:              ldapConfig.Timeout,
			ConnectionTimeout:    ldapConfig.ConnectionTimeout,
			MaxConnections:       ldapConfig.MaxConnections,
			IsActiveDirectory:    ldapConfig.IsActiveDirectory,
			Domain:               ldapConfig.Domain,
			GroupMemberAttribute: ldapConfig.GroupMemberAttribute,
			UserGroupAttribute:   ldapConfig.UserGroupAttribute,
			RoleMappings:         ldapConfig.RoleMappings,
			DefaultRoles:         ldapConfig.DefaultRoles,
			UserAttributes: providers.LDAPAttributeMap{
				Username:    ldapConfig.UserAttributes.Username,
				Email:       ldapConfig.UserAttributes.Email,
				FirstName:   ldapConfig.UserAttributes.FirstName,
				LastName:    ldapConfig.UserAttributes.LastName,
				DisplayName: ldapConfig.UserAttributes.DisplayName,
				Title:       ldapConfig.UserAttributes.Title,
				Department:  ldapConfig.UserAttributes.Department,
				Phone:       ldapConfig.UserAttributes.Phone,
				Manager:     ldapConfig.UserAttributes.Manager,
			},
		}

		// Create LDAP provider instance
		provider := providers.NewLDAPClient(providerConfig, logger.With("ldap_provider", name))
		ldapProviders[name] = provider

		// Log successful provider creation
		slog.Info("LDAP provider created successfully",
			"provider", name,
			"host", ldapConfig.Host,
			"is_ad", ldapConfig.IsActiveDirectory,
			"use_ssl", ldapConfig.UseSSL,
			"use_tls", ldapConfig.UseTLS)
	}

	// If no providers were successfully created and there were errors, return error
	if len(ldapProviders) == 0 && len(errors) > 0 {
		return nil, fmt.Errorf("failed to create any LDAP providers: %v", errors)
	}

	// Log warnings for failed providers but continue if at least one succeeded
	if len(errors) > 0 {
		for _, err := range errors {
			slog.Warn("Failed to create LDAP provider", "error", err)
		}
	}

	return ldapProviders, nil
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

// Note: Environment helper functions have been moved to pkg/config/env_helpers.go
// All getEnv, getBoolEnv, getDurationEnv, getStringSliceEnv calls now use config.* variants

// getOAuth2ClientSecret loads OAuth2 client secret from environment variables or files with secure error handling
func getOAuth2ClientSecret(provider string) (string, error) {
	// Validate provider name to prevent injection attacks
	if provider == "" {
		auditSecretAccess("", "validation", "", false, "empty provider name")
		return "", fmt.Errorf("provider name cannot be empty")
	}

	// Sanitize provider name to prevent injection - only allow alphanumeric and hyphens
	sanitizedProvider := strings.ToUpper(strings.ReplaceAll(provider, "-", "_"))
	if !isValidProviderName(sanitizedProvider) {
		auditSecretAccess(provider, "validation", "", false, "invalid provider name format")
		return "", fmt.Errorf("invalid provider name format")
	}

	// Step 1: Check environment variable first - OAUTH2_<PROVIDER>_CLIENT_SECRET
	envVar := fmt.Sprintf("OAUTH2_%s_CLIENT_SECRET", sanitizedProvider)

	// Use atomic read of environment variable to prevent race conditions
	secret := getEnvAtomic(envVar)
	if secret != "" {
		// Validate the secret content
		if err := validateSecretContent(secret); err != nil {
			auditSecretAccess(provider, "environment", envVar, false, "secret validation failed")
			return "", fmt.Errorf("invalid OAuth2 client secret format for provider %s", provider)
		}

		auditSecretAccess(provider, "environment", envVar, true, "")
		slog.Info("OAuth2 client secret loaded from environment",
			"provider", provider,
			"source", "environment",
			"env_var", envVar)
		return secret, nil
	}

	// Step 1.5: Check fallback environment variable for backward compatibility - <PROVIDER>_CLIENT_SECRET
	fallbackEnvVar := fmt.Sprintf("%s_CLIENT_SECRET", sanitizedProvider)
	secret = getEnvAtomic(fallbackEnvVar)
	if secret != "" {
		// Validate the secret content
		if err := validateSecretContent(secret); err != nil {
			auditSecretAccess(provider, "environment", fallbackEnvVar, false, "secret validation failed")
			return "", fmt.Errorf("invalid OAuth2 client secret format for provider %s", provider)
		}

		auditSecretAccess(provider, "environment", fallbackEnvVar, true, "")
		slog.Info("OAuth2 client secret loaded from environment (backward compatibility)",
			"provider", provider,
			"source", "environment",
			"env_var", fallbackEnvVar,
			"note", "Consider migrating to OAUTH2_<PROVIDER>_CLIENT_SECRET format")
		return secret, nil
	}

	// Step 2: Fall back to file path from environment variable - OAUTH2_<PROVIDER>_SECRET_FILE
	fileEnvVar := fmt.Sprintf("OAUTH2_%s_SECRET_FILE", sanitizedProvider)
	filePath := getEnvAtomic(fileEnvVar)

	if filePath != "" {
		// Validate file path to prevent path traversal attacks
		if err := validateFilePath(filePath); err != nil {
			auditSecretAccess(provider, "file_path_validation", fileEnvVar, false, err.Error())
			slog.Error("Invalid file path for OAuth2 secret",
				"provider", provider,
				"error", "path validation failed")
			return "", fmt.Errorf("OAuth2 client secret configuration error for provider: %s", provider)
		}

		// Read file contents securely
		secret, err := readSecretFile(filePath)
		if err != nil {
			auditSecretAccess(provider, "file", filePath, false, "file read failed")
			slog.Error("Failed to read OAuth2 client secret file",
				"provider", provider,
				"error", "file access error")
			return "", fmt.Errorf("OAuth2 client secret file not accessible for provider: %s", provider)
		}

		// Validate the secret content
		if err := validateSecretContent(secret); err != nil {
			auditSecretAccess(provider, "file", filePath, false, "secret validation failed")
			return "", fmt.Errorf("invalid OAuth2 client secret format for provider %s", provider)
		}

		auditSecretAccess(provider, "file", filePath, true, "")
		slog.Info("OAuth2 client secret loaded from file",
			"provider", provider,
			"source", "file")
		return secret, nil
	}

	// Step 3: Neither environment variable nor file path provided
	auditSecretAccess(provider, "not_configured", "", false, "no secret source configured")
	slog.Error("OAuth2 client secret not configured",
		"provider", provider,
		"expected_env_var", envVar,
		"expected_file_env_var", fileEnvVar)

	return "", fmt.Errorf("OAuth2 client secret not configured for provider: %s. Set either %s or %s environment variable",
		provider, envVar, fileEnvVar)
}

// getEnvAtomic performs atomic read of environment variable to prevent race conditions
func getEnvAtomic(key string) string {
	return config.GetEnvOrDefault(key, "")
}

// isValidProviderName validates provider name format to prevent injection
func isValidProviderName(provider string) bool {
	if len(provider) == 0 || len(provider) > MaxProviderNameLength {
		return false
	}

	for _, char := range provider {
		if !((char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_') {
			return false
		}
	}
	return true
}

// validateConfigFilePath validates configuration file paths with stricter security rules
// This is used specifically for main config files, not OAuth2 secrets
func validateConfigFilePath(filePath string) error {
	// Check for empty path
	if strings.TrimSpace(filePath) == "" {
		return fmt.Errorf("empty file path")
	}

	// SECURITY: Clean the path first to normalize it
	cleanedPath := filepath.Clean(filePath)

	// SECURITY: Detect obvious path traversal attempts before resolution
	if strings.Contains(filePath, "..") || strings.Contains(cleanedPath, "..") {
		return fmt.Errorf("path traversal attempt detected: contains '..'")
	}

	// SECURITY: Reject paths with null bytes (potential injection)
	if strings.Contains(filePath, "\x00") {
		return fmt.Errorf("path contains null byte")
	}

	// Convert to absolute path for validation
	absPath, err := filepath.Abs(cleanedPath)
	if err != nil {
		return fmt.Errorf("invalid file path format: %w", err)
	}

	// SECURITY: Additional check after absolute path conversion
	if strings.Contains(absPath, "..") {
		return fmt.Errorf("path traversal detected in absolute path")
	}

	// SECURITY: Check for Windows UNC paths or special devices
	if strings.HasPrefix(absPath, "\\\\") || strings.HasPrefix(absPath, "//") {
		return fmt.Errorf("UNC paths not allowed")
	}

	// SECURITY: Restrict to specific directories for config files
	// Config files should only be in designated configuration directories
	allowedPrefixes := []string{
		"/etc/nephoran",          // Production config directory
		"/etc/nephoran-operator", // Alternative production config
		"/var/lib/nephoran",      // State directory
		"/opt/nephoran/config",   // Alternative config location
		"/config",                // Kubernetes ConfigMap mount
		"/etc/config",            // Alternative Kubernetes mount
	}

	// For development/testing, allow current directory subdirectories
	currentDir, err := os.Getwd()
	if err == nil && currentDir != "" {
		// Only allow config subdirectory in development
		allowedPrefixes = append(allowedPrefixes, filepath.Join(currentDir, "config"))
		allowedPrefixes = append(allowedPrefixes, filepath.Join(currentDir, "test", "config"))

		// For CI/CD environments
		if strings.Contains(currentDir, "github") || strings.Contains(currentDir, "gitlab") {
			allowedPrefixes = append(allowedPrefixes, currentDir)
		}
	}

	// SECURITY: Check if path starts with any allowed prefix
	pathAllowed := false
	for _, prefix := range allowedPrefixes {
		// Use proper path comparison to avoid prefix bypass
		prefixAbs, _ := filepath.Abs(prefix)
		if prefixAbs != "" && (absPath == prefixAbs || strings.HasPrefix(absPath, prefixAbs+string(filepath.Separator))) {
			pathAllowed = true
			break
		}
	}

	if !pathAllowed {
		return fmt.Errorf("config file path not in allowed directory (must be in /etc/nephoran, /config, or ./config)")
	}

	// SECURITY: Validate file extension
	ext := strings.ToLower(filepath.Ext(absPath))
	allowedExtensions := []string{".json", ".yaml", ".yml", ".conf", ""}
	extensionAllowed := false
	for _, allowedExt := range allowedExtensions {
		if ext == allowedExt {
			extensionAllowed = true
			break
		}
	}

	if !extensionAllowed {
		return fmt.Errorf("invalid config file extension: %s (allowed: .json, .yaml, .yml, .conf)", ext)
	}

	// SECURITY: Check filename for suspicious patterns
	filename := filepath.Base(absPath)
	if strings.HasPrefix(filename, ".") && filename != ".config" {
		return fmt.Errorf("hidden files not allowed as config")
	}

	// SECURITY: Reject special file names
	dangerousNames := []string{"/dev/", "/proc/", "/sys/", "passwd", "shadow", "sudoers"}
	for _, dangerous := range dangerousNames {
		if strings.Contains(strings.ToLower(absPath), dangerous) {
			return fmt.Errorf("suspicious file path detected")
		}
	}

	return nil
}

// validateFilePath prevents path traversal attacks and validates file path security
// This is used for OAuth2 secret files with different allowed directories
func validateFilePath(filePath string) error {
	// Check for empty path
	if strings.TrimSpace(filePath) == "" {
		return fmt.Errorf("empty file path")
	}

	// Convert to absolute path and clean it
	absPath, err := filepath.Abs(filepath.Clean(filePath))
	if err != nil {
		return fmt.Errorf("invalid file path format")
	}

	// Prevent path traversal attacks - check for suspicious patterns
	if strings.Contains(absPath, "..") {
		return fmt.Errorf("path traversal attempt detected")
	}

	// Restrict to reasonable base directories for security
	allowedPrefixes := []string{
		"/etc/secrets",
		"/var/secrets",
		"/tmp/secrets",
		"/secrets",
		"/run/secrets", // Common Kubernetes secret mount path
	}

	// Allow relative paths under current directory for development
	currentDir, err := os.Getwd()
	if err == nil && currentDir != "" {
		allowedPrefixes = append(allowedPrefixes, filepath.Join(currentDir, "secrets"))
		allowedPrefixes = append(allowedPrefixes, filepath.Join(currentDir, "config"))
	}

	// Check if path starts with any allowed prefix
	pathAllowed := false
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(absPath, prefix) {
			pathAllowed = true
			break
		}
	}

	if !pathAllowed {
		return fmt.Errorf("file path not in allowed directory")
	}

	return nil
}

// readSecretFile securely reads secret from file with proper error handling
func readSecretFile(filePath string) (string, error) {
	// Check if file exists and is readable
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("secret file does not exist")
		}
		if os.IsPermission(err) {
			return "", fmt.Errorf("permission denied accessing secret file")
		}
		return "", fmt.Errorf("cannot access secret file")
	}

	// Check file size limits (prevent reading huge files)
	if info.Size() > MaxSecretFileSize {
		return "", fmt.Errorf("secret file too large")
	}

	if info.Size() == 0 {
		return "", fmt.Errorf("secret file is empty")
	}

	// Check file permissions - should not be world-readable
	mode := info.Mode()
	if mode&0o044 != 0 { // Check if world or group readable
		slog.Warn("Secret file has overly permissive permissions",
			"file", filePath,
			"mode", mode.String())
	}

	// Read file contents
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read secret file contents")
	}

	// Trim whitespace and newlines
	secret := strings.TrimSpace(string(content))
	if secret == "" {
		return "", fmt.Errorf("secret file contains only whitespace")
	}

	return secret, nil
}

// validateSecretContent validates OAuth2 client secret format and security requirements
func validateSecretContent(secret string) error {
	// Check for empty or whitespace-only secrets
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return fmt.Errorf("secret is empty or contains only whitespace")
	}

	// Check minimum length for security
	if len(secret) < MinSecretLength {
		return fmt.Errorf("secret too short, minimum %d characters required", MinSecretLength)
	}

	// Check maximum length to prevent potential DoS
	if len(secret) > MaxSecretLength {
		return fmt.Errorf("secret too long, maximum %d characters allowed", MaxSecretLength)
	}

	// Check for common weak patterns
	lowerSecret := strings.ToLower(secret)

	// Check if secret starts with obvious weak patterns
	weakPrefixes := []string{"password", "test", "example", "demo", "sample"}
	for _, prefix := range weakPrefixes {
		if strings.HasPrefix(lowerSecret, prefix+"-") || strings.HasPrefix(lowerSecret, prefix+"_") {
			return fmt.Errorf("secret starts with weak pattern: %s", prefix)
		}
	}

	// Only reject if the secret is JUST a weak pattern or very simple
	weakPatterns := []string{"password", "secret", "test", "example", "admin", "demo"}
	for _, pattern := range weakPatterns {
		// Reject if the secret is exactly the pattern or pattern with simple suffix
		if lowerSecret == pattern ||
			lowerSecret == pattern+"123" ||
			lowerSecret == pattern+"1234" ||
			lowerSecret == "my"+pattern ||
			lowerSecret == "your"+pattern {
			return fmt.Errorf("secret contains common weak patterns")
		}
	}

	// Basic entropy check - ensure it's not all the same character
	if len(strings.TrimLeft(secret, string(secret[0]))) == 0 {
		return fmt.Errorf("secret has insufficient entropy")
	}

	return nil
}

// auditSecretAccess provides comprehensive audit logging for security compliance
func auditSecretAccess(provider, source, location string, success bool, errorMsg string) {
	logLevel := slog.LevelInfo
	if !success {
		logLevel = slog.LevelWarn
	}

	attrs := []slog.Attr{
		slog.String("event", "oauth2_secret_access"),
		slog.String("provider", provider),
		slog.String("source", source),
		slog.Bool("success", success),
		slog.Time("timestamp", time.Now()),
	}

	if location != "" && success {
		// For successful operations, log safe location info
		if strings.Contains(location, "OAUTH2_") {
			attrs = append(attrs, slog.String("env_var", "OAUTH2_***_CLIENT_SECRET"))
		} else {
			attrs = append(attrs, slog.String("file_type", "secret_file"))
		}
	}

	if !success && errorMsg != "" {
		attrs = append(attrs, slog.String("error_type", errorMsg))
	}

	slog.LogAttrs(nil, logLevel, "OAuth2 client secret access attempt", attrs...)

	// Additional audit logging through security package if available
	if security.GlobalAuditLogger != nil {
		security.GlobalAuditLogger.LogSecretAccess(
			fmt.Sprintf("oauth2_%s", provider),
			source,
			"", "", success,
			func() error {
				if errorMsg != "" {
					return fmt.Errorf("%s", errorMsg)
				}
				return nil
			}())
	}
}

// validateOAuth2ClientSecret validates OAuth2 client secret format and strength
func validateOAuth2ClientSecret(provider, secret string) error {
	// Check for empty secret
	if strings.TrimSpace(secret) == "" {
		return fmt.Errorf("empty secret")
	}

	// Provider-specific validation
	switch provider {
	case "azure":
		// Azure client secrets should be at least minimum length
		if len(secret) < MinSecretLength {
			return fmt.Errorf("secret too short")
		}
		// Azure secrets often contain special characters
		// No specific format validation beyond length

	case "okta":
		// Okta client secrets are typically 64 characters
		if len(secret) < 40 {
			return fmt.Errorf("secret too short")
		}

	case "keycloak":
		// Keycloak secrets are UUID-like or random strings - require longer minimum
		if len(secret) < MinJWTSecretLength {
			return fmt.Errorf("secret too short")
		}

	case "google":
		// Google OAuth2 secrets have specific format
		if len(secret) < 24 {
			return fmt.Errorf("secret too short")
		}

	default:
		// Generic validation for custom providers
		if len(secret) < MinSecretLength {
			return fmt.Errorf("secret too short")
		}
	}

	// Check for common placeholder values
	lowerSecret := strings.ToLower(secret)
	if strings.Contains(lowerSecret, "your-secret") ||
		strings.Contains(lowerSecret, "client-secret") ||
		strings.Contains(lowerSecret, "changeme") ||
		strings.Contains(lowerSecret, "placeholder") ||
		strings.Contains(lowerSecret, "example") ||
		secret == "secret" {
		return fmt.Errorf("placeholder secret detected")
	}

	return nil
}

// determineSecretSource determines whether secret was loaded from file or environment
func determineSecretSource(filename, envVar string) string {
	// Check if file exists
	if _, err := os.Stat(filepath.Join("/secrets/oauth2", filename)); err == nil {
		return "file"
	}

	// Check if environment variable is set
	if config.GetEnvOrDefault(envVar, "") != "" {
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
