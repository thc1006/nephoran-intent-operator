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

// Configuration constants for security limits and validation
const (
	// Secret file size limits
	MaxSecretFileSize       = 64 * 1024 // 64KB limit for secret files
	MinSecretLength         = 16        // Minimum secret length for security
	MaxSecretLength         = 512       // Maximum secret length to prevent DoS
	MaxProviderNameLength   = 50        // Maximum provider name length
	MinJWTSecretLength      = 32        // Minimum JWT secret length
	
	// Default configuration values
	DefaultKeycloakRealm    = "master"  // Default Keycloak realm
	DefaultTokenTTL         = 24 * 60 * 60 // 24 hours in seconds
	DefaultRefreshTTL       = 7 * 24 * 60 * 60 // 7 days in seconds
)

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
// configPath: Path to the auth config file. If empty, falls back to AUTH_CONFIG_FILE env var
func LoadAuthConfig(configPath string) (*AuthConfig, error) {
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

	// Determine config file path: use provided path or fall back to environment variable
	configFile := configPath
	if configFile == "" {
		configFile = getEnv("AUTH_CONFIG_FILE", "")
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
	if azureClientID := getEnv("AZURE_CLIENT_ID", ""); azureClientID != "" {
		secret, err := getOAuth2ClientSecret("azure")
		if err != nil && getBoolEnv("AZURE_ENABLED", true) {
			errors = append(errors, fmt.Errorf("azure-ad: %w", err))
		}
		
		c.Providers["azure-ad"] = ProviderConfig{
			Enabled:      getBoolEnv("AZURE_ENABLED", true),
			Type:         "azure-ad",
			ClientID:     azureClientID,
			ClientSecret: secret,
			TenantID:     getEnv("AZURE_TENANT_ID", ""),
			Scopes:       getStringSliceEnv("AZURE_SCOPES", []string{"openid", "profile", "email", "User.Read"}),
		}
	}

	// Okta provider
	if oktaClientID := getEnv("OKTA_CLIENT_ID", ""); oktaClientID != "" {
		secret, err := getOAuth2ClientSecret("okta")
		if err != nil && getBoolEnv("OKTA_ENABLED", true) {
			errors = append(errors, fmt.Errorf("okta: %w", err))
		}
		
		c.Providers["okta"] = ProviderConfig{
			Enabled:      getBoolEnv("OKTA_ENABLED", true),
			Type:         "okta",
			ClientID:     oktaClientID,
			ClientSecret: secret,
			Domain:       getEnv("OKTA_DOMAIN", ""),
			Scopes:       getStringSliceEnv("OKTA_SCOPES", []string{"openid", "profile", "email", "groups"}),
		}
	}

	// Keycloak provider
	if keycloakClientID := getEnv("KEYCLOAK_CLIENT_ID", ""); keycloakClientID != "" {
		secret, err := getOAuth2ClientSecret("keycloak")
		if err != nil && getBoolEnv("KEYCLOAK_ENABLED", true) {
			errors = append(errors, fmt.Errorf("keycloak: %w", err))
		}
		
		c.Providers["keycloak"] = ProviderConfig{
			Enabled:      getBoolEnv("KEYCLOAK_ENABLED", true),
			Type:         "keycloak",
			ClientID:     keycloakClientID,
			ClientSecret: secret,
			BaseURL:      getEnv("KEYCLOAK_BASE_URL", ""),
			Realm:        getEnv("KEYCLOAK_REALM", "master"),
			Scopes:       getStringSliceEnv("KEYCLOAK_SCOPES", []string{"openid", "profile", "email", "roles"}),
		}
	}

	// Google provider
	if googleClientID := getEnv("GOOGLE_CLIENT_ID", ""); googleClientID != "" {
		secret, err := getOAuth2ClientSecret("google")
		if err != nil && getBoolEnv("GOOGLE_ENABLED", true) {
			errors = append(errors, fmt.Errorf("google: %w", err))
		}
		
		c.Providers["google"] = ProviderConfig{
			Enabled:      getBoolEnv("GOOGLE_ENABLED", true),
			Type:         "google",
			ClientID:     googleClientID,
			ClientSecret: secret,
			Scopes:       getStringSliceEnv("GOOGLE_SCOPES", []string{"openid", "profile", "email"}),
		}
	}

	// Custom provider
	if customClientID := getEnv("CUSTOM_CLIENT_ID", ""); customClientID != "" {
		secret, err := getOAuth2ClientSecret("custom")
		if err != nil && getBoolEnv("CUSTOM_ENABLED", true) {
			errors = append(errors, fmt.Errorf("custom: %w", err))
		}
		
		c.Providers["custom"] = ProviderConfig{
			Enabled:      getBoolEnv("CUSTOM_ENABLED", true),
			Type:         "custom",
			ClientID:     customClientID,
			ClientSecret: secret,
			AuthURL:      getEnv("CUSTOM_AUTH_URL", ""),
			TokenURL:     getEnv("CUSTOM_TOKEN_URL", ""),
			UserInfoURL:  getEnv("CUSTOM_USERINFO_URL", ""),
			Scopes:       getStringSliceEnv("CUSTOM_SCOPES", []string{"openid", "profile", "email"}),
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

// loadFromFile loads configuration from JSON file
func (c *AuthConfig) loadFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, c)
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
			providerErrors[name] = errs
		}
	}

	// Report validation errors
	if len(providerErrors) > 0 {
		var errorMsgs []string
		for provider, errs := range providerErrors {
			errorMsgs = append(errorMsgs, fmt.Sprintf("%s: %s", provider, strings.Join(errs, ", ")))
		}
		return fmt.Errorf("provider validation failed: %s", strings.Join(errorMsgs, "; "))
	}

	if !hasEnabledProvider {
		return fmt.Errorf("at least one OAuth2 provider must be enabled")
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
	return os.Getenv(key)
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

// validateFilePath prevents path traversal attacks and validates file path security
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
	currentDir, _ := os.Getwd()
	if currentDir != "" {
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
	if mode&0044 != 0 { // Check if world or group readable
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
	if strings.Contains(strings.ToLower(secret), "password") ||
		strings.Contains(strings.ToLower(secret), "secret") ||
		strings.Contains(strings.ToLower(secret), "test") ||
		strings.Contains(strings.ToLower(secret), "example") {
		return fmt.Errorf("secret contains common weak patterns")
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