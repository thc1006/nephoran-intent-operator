package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thc1006/nephoran-intent-operator/pkg/security"
)

func TestLoadAuthConfig_JWTSecretValidation(t *testing.T) {
	// Save original environment to restore later
	originalAuthEnabled := os.Getenv("AUTH_ENABLED")
	originalJWTSecret := os.Getenv("JWT_SECRET_KEY")
	originalGoogleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	originalGoogleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	
	defer func() {
		os.Setenv("AUTH_ENABLED", originalAuthEnabled)
		os.Setenv("JWT_SECRET_KEY", originalJWTSecret)
		os.Setenv("GOOGLE_CLIENT_ID", originalGoogleClientID)
		os.Setenv("GOOGLE_CLIENT_SECRET", originalGoogleClientSecret)
	}()

	tests := []struct {
		name        string
		authEnabled string
		jwtSecret   string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "auth disabled, empty JWT secret - should pass",
			authEnabled: "false",
			jwtSecret:   "",
			expectError: false,
		},
		{
			name:        "auth disabled, valid JWT secret - should pass",
			authEnabled: "false", 
			jwtSecret:   "test-secret-key-that-is-long-enough-for-jwt",
			expectError: false,
		},
		{
			name:        "auth enabled, empty JWT secret - should fail",
			authEnabled: "true",
			jwtSecret:   "",
			expectError: true,
			errorMsg:    "auth enabled but JWTSecretKey is empty",
		},
		{
			name:        "auth enabled, whitespace-only JWT secret - should fail",
			authEnabled: "true",
			jwtSecret:   "   \t\n   ",
			expectError: true,
			errorMsg:    "auth enabled but JWTSecretKey is empty",
		},
		{
			name:        "auth enabled, valid JWT secret - should pass",
			authEnabled: "true",
			jwtSecret:   "test-secret-key-that-is-long-enough-for-jwt-validation",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			os.Setenv("AUTH_ENABLED", tt.authEnabled)
			os.Setenv("JWT_SECRET_KEY", tt.jwtSecret)
			
			// Set up a dummy OAuth2 provider if auth is enabled to pass provider validation
			if tt.authEnabled == "true" {
				os.Setenv("GOOGLE_CLIENT_ID", "test-client-id")
				os.Setenv("GOOGLE_CLIENT_SECRET", "test-client-secret")
			} else {
				os.Setenv("GOOGLE_CLIENT_ID", "")
				os.Setenv("GOOGLE_CLIENT_SECRET", "")
			}

			// Call LoadAuthConfig
			config, err := LoadAuthConfig("")

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("expected error message %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if config == nil {
					t.Errorf("expected config but got nil")
				}
			}
		})
	}
}

func TestLoadAuthConfig_AuthDisabled(t *testing.T) {
	// Save original environment
	originalAuthEnabled := os.Getenv("AUTH_ENABLED")
	originalJWTSecret := os.Getenv("JWT_SECRET_KEY")
	
	defer func() {
		os.Setenv("AUTH_ENABLED", originalAuthEnabled)
		os.Setenv("JWT_SECRET_KEY", originalJWTSecret)
	}()

	// Test with auth disabled and no JWT secret
	os.Setenv("AUTH_ENABLED", "false")
	os.Setenv("JWT_SECRET_KEY", "")

	config, err := LoadAuthConfig("")
	if err != nil {
		t.Errorf("unexpected error when auth is disabled: %v", err)
		return
	}

	if config == nil {
		t.Errorf("expected config but got nil")
		return
	}

	if config.Enabled {
		t.Errorf("expected auth to be disabled but it was enabled")
	}
}

func TestLoadAuthConfig_AuthEnabledWithValidSecret(t *testing.T) {
	// Save original environment
	originalAuthEnabled := os.Getenv("AUTH_ENABLED")
	originalJWTSecret := os.Getenv("JWT_SECRET_KEY")
	originalGoogleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	
	defer func() {
		os.Setenv("AUTH_ENABLED", originalAuthEnabled)
		os.Setenv("JWT_SECRET_KEY", originalJWTSecret)
		os.Setenv("GOOGLE_CLIENT_ID", originalGoogleClientID)
	}()

	// Test with auth enabled and valid JWT secret
	os.Setenv("AUTH_ENABLED", "true")
	os.Setenv("JWT_SECRET_KEY", "this-is-a-very-secure-jwt-secret-key-for-testing-purposes")
	os.Setenv("GOOGLE_CLIENT_ID", "test-client-id")
	os.Setenv("GOOGLE_CLIENT_SECRET", "test-client-secret")

	config, err := LoadAuthConfig("")
	if err != nil {
		t.Errorf("unexpected error with valid JWT secret: %v", err)
		return
	}

	if config == nil {
		t.Errorf("expected config but got nil")
		return
	}

	if !config.Enabled {
		t.Errorf("expected auth to be enabled but it was disabled")
	}

	if config.JWTSecretKey != "this-is-a-very-secure-jwt-secret-key-for-testing-purposes" {
		t.Errorf("expected JWT secret to be set correctly")
	}
}

// TestGetOAuth2ClientSecret_ErrorScenarios tests various error scenarios for OAuth2 client secret loading
func TestGetOAuth2ClientSecret_ErrorScenarios(t *testing.T) {
	// Save original environment and global audit logger
	originalAuditLogger := security.GlobalAuditLogger
	defer func() {
		security.GlobalAuditLogger = originalAuditLogger
	}()

	// Create a real audit logger for testing but with null output
	security.GlobalAuditLogger, _ = security.NewAuditLogger("", security.AuditLevelInfo)

	tests := []struct {
		name              string
		provider          string
		setupSecrets      func(t *testing.T) (cleanup func())
		setupEnvironment  func(t *testing.T) (cleanup func())
		expectError       bool
		expectedErrorMsg  string
		expectAuditLogs   bool
	}{
		{
			name:             "empty provider name",
			provider:         "",
			setupSecrets:     func(t *testing.T) func() { return func() {} },
			setupEnvironment: func(t *testing.T) func() { return func() {} },
			expectError:      true,
			expectedErrorMsg: "provider name cannot be empty",
		},
		{
			name:     "SecretLoader creation failure with successful env fallback",
			provider: "azure",
			setupSecrets: func(t *testing.T) func() {
				// Create invalid base path to force SecretLoader failure
				return func() {}
			},
			setupEnvironment: func(t *testing.T) func() {
				originalEnv := os.Getenv("AZURE_CLIENT_SECRET")
				os.Setenv("AZURE_CLIENT_SECRET", "test-azure-oauth2-from-env-1234567890abcdef")
				return func() {
					os.Setenv("AZURE_CLIENT_SECRET", originalEnv)
				}
			},
			expectError:       false,
			expectAuditLogs: true,
		},
		{
			name:     "SecretLoader creation failure with env fallback also empty",
			provider: "okta",
			setupSecrets: func(t *testing.T) func() {
				return func() {}
			},
			setupEnvironment: func(t *testing.T) func() {
				originalEnv := os.Getenv("OKTA_CLIENT_SECRET")
				os.Setenv("OKTA_CLIENT_SECRET", "")
				return func() {
					os.Setenv("OKTA_CLIENT_SECRET", originalEnv)
				}
			},
			expectError:      true,
			expectedErrorMsg: "OAuth2 client secret not configured for provider: okta",
		},
		{
			name:     "file loading failure but successful env fallback",
			provider: "keycloak",
			setupSecrets: func(t *testing.T) func() {
				// Create temporary directory structure for testing
				tempDir := t.TempDir()
				secretsDir := filepath.Join(tempDir, "secrets", "oauth2")
				if err := os.MkdirAll(secretsDir, 0755); err != nil {
					t.Fatalf("Failed to create test directory: %v", err)
				}
				// Don't create the file to simulate loading failure
				return func() {}
			},
			setupEnvironment: func(t *testing.T) func() {
				originalEnv := os.Getenv("KEYCLOAK_CLIENT_SECRET")
				os.Setenv("KEYCLOAK_CLIENT_SECRET", "keycloak-env-fallback-secret-12345678901234567890")
				return func() {
					os.Setenv("KEYCLOAK_CLIENT_SECRET", originalEnv)
				}
			},
			expectError:       false,
			expectAuditLogs: true,
		},
		{
			name:     "both file and env failure",
			provider: "google",
			setupSecrets: func(t *testing.T) func() {
				return func() {}
			},
			setupEnvironment: func(t *testing.T) func() {
				originalEnv := os.Getenv("GOOGLE_CLIENT_SECRET")
				os.Setenv("GOOGLE_CLIENT_SECRET", "")
				return func() {
					os.Setenv("GOOGLE_CLIENT_SECRET", originalEnv)
				}
			},
			expectError:      true,
			expectedErrorMsg: "OAuth2 client secret not configured for provider: google",
		},
		{
			name:     "successful loading from environment with validation success",
			provider: "azure",
			setupSecrets: func(t *testing.T) func() {
				return func() {}
			},
			setupEnvironment: func(t *testing.T) func() {
				originalEnv := os.Getenv("AZURE_CLIENT_SECRET")
				// Use a valid Azure-style secret (16+ characters)
				os.Setenv("AZURE_CLIENT_SECRET", "valid-azure-secret-1234567890abcdef")
				return func() {
					os.Setenv("AZURE_CLIENT_SECRET", originalEnv)
				}
			},
			expectError:       false,
			expectAuditLogs: true,
		},
		{
			name:     "invalid secret format",
			provider: "azure",
			setupSecrets: func(t *testing.T) func() {
				return func() {}
			},
			setupEnvironment: func(t *testing.T) func() {
				originalEnv := os.Getenv("AZURE_CLIENT_SECRET")
				// Use an invalid secret (too short)
				os.Setenv("AZURE_CLIENT_SECRET", "short")
				return func() {
					os.Setenv("AZURE_CLIENT_SECRET", originalEnv)
				}
			},
			expectError:      true,
			expectedErrorMsg: "invalid OAuth2 client secret format for provider azure",
		},
		{
			name:     "placeholder secret detected",
			provider: "okta",
			setupSecrets: func(t *testing.T) func() {
				return func() {}
			},
			setupEnvironment: func(t *testing.T) func() {
				originalEnv := os.Getenv("OKTA_CLIENT_SECRET")
				os.Setenv("OKTA_CLIENT_SECRET", "your-secret-here-changeme-placeholder")
				return func() {
					os.Setenv("OKTA_CLIENT_SECRET", originalEnv)
				}
			},
			expectError:      true,
			expectedErrorMsg: "invalid OAuth2 client secret format for provider okta",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			cleanupSecrets := tt.setupSecrets(t)
			cleanupEnv := tt.setupEnvironment(t)
			
			defer cleanupSecrets()
			defer cleanupEnv()

			// Test the function
			secret, err := getOAuth2ClientSecret(tt.provider)

			// Validate results
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.expectedErrorMsg != "" && !strings.Contains(err.Error(), tt.expectedErrorMsg) {
					t.Errorf("expected error message to contain %q, got %q", tt.expectedErrorMsg, err.Error())
				}
				if secret != "" {
					t.Errorf("expected empty secret on error, got %q", secret)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if secret == "" {
					t.Errorf("expected non-empty secret but got empty")
				}
			}
		})
	}
}

// TestLoadProviders_ErrorPropagation tests error propagation in loadProviders
func TestLoadProviders_ErrorPropagation(t *testing.T) {
	originalAuditLogger := security.GlobalAuditLogger
	defer func() {
		security.GlobalAuditLogger = originalAuditLogger
	}()

	security.GlobalAuditLogger, _ = security.NewAuditLogger("", security.AuditLevelInfo)

	tests := []struct {
		name                string
		setupEnvironment    func(t *testing.T) (cleanup func())
		expectError         bool
		expectedErrorCount  int
		expectedProviders   []string
		expectedErrorSubstr string
	}{
		{
			name: "all providers disabled - should succeed",
			setupEnvironment: func(t *testing.T) func() {
				env := map[string]string{
					"AZURE_CLIENT_ID":     "",
					"OKTA_CLIENT_ID":      "",
					"KEYCLOAK_CLIENT_ID":  "",
					"GOOGLE_CLIENT_ID":    "",
					"CUSTOM_CLIENT_ID":    "",
				}
				cleanup := setMultipleEnvVars(env)
				return cleanup
			},
			expectError:       false,
			expectedProviders: []string{},
		},
		{
			name: "enabled provider missing secret",
			setupEnvironment: func(t *testing.T) func() {
				env := map[string]string{
					"AZURE_CLIENT_ID":     "test-client-id",
					"AZURE_ENABLED":       "true",
					"AZURE_CLIENT_SECRET": "",
					"OKTA_CLIENT_ID":      "",
					"KEYCLOAK_CLIENT_ID":  "",
					"GOOGLE_CLIENT_ID":    "",
					"CUSTOM_CLIENT_ID":    "",
				}
				return setMultipleEnvVars(env)
			},
			expectError:         true,
			expectedErrorSubstr: "azure-ad",
		},
		{
			name: "multiple providers with mixed success/failure",
			setupEnvironment: func(t *testing.T) func() {
				env := map[string]string{
					"AZURE_CLIENT_ID":       "test-azure-client-id",
					"AZURE_ENABLED":         "true", 
					"AZURE_CLIENT_SECRET":   "valid-azure-secret-1234567890abcdef",
					"OKTA_CLIENT_ID":        "test-okta-client-id",
					"OKTA_ENABLED":          "true",
					"OKTA_CLIENT_SECRET":    "", // Missing secret - should cause error
					"KEYCLOAK_CLIENT_ID":    "",
					"GOOGLE_CLIENT_ID":      "",
					"CUSTOM_CLIENT_ID":      "",
				}
				return setMultipleEnvVars(env)
			},
			expectError:         true,
			expectedProviders:   []string{"azure-ad"},
			expectedErrorSubstr: "okta",
		},
		{
			name: "multiple providers all failing",
			setupEnvironment: func(t *testing.T) func() {
				env := map[string]string{
					"AZURE_CLIENT_ID":     "test-azure-client-id",
					"AZURE_ENABLED":       "true",
					"AZURE_CLIENT_SECRET": "",
					"OKTA_CLIENT_ID":      "test-okta-client-id", 
					"OKTA_ENABLED":        "true",
					"OKTA_CLIENT_SECRET":  "",
					"KEYCLOAK_CLIENT_ID":  "",
					"GOOGLE_CLIENT_ID":    "",
					"CUSTOM_CLIENT_ID":    "",
				}
				return setMultipleEnvVars(env)
			},
			expectError:         true,
			expectedErrorSubstr: "multiple provider configuration errors",
		},
		{
			name: "disabled provider doesn't cause errors even with missing secret",
			setupEnvironment: func(t *testing.T) func() {
				env := map[string]string{
					"AZURE_CLIENT_ID":     "test-azure-client-id",
					"AZURE_ENABLED":       "false", // Disabled
					"AZURE_CLIENT_SECRET": "", // Missing but disabled
					"GOOGLE_CLIENT_ID":    "test-google-client-id",
					"GOOGLE_ENABLED":      "true",
					"GOOGLE_CLIENT_SECRET": "valid-google-secret-123456789012345678901234",
					"OKTA_CLIENT_ID":      "",
					"KEYCLOAK_CLIENT_ID":  "",
					"CUSTOM_CLIENT_ID":    "",
				}
				return setMultipleEnvVars(env)
			},
			expectError:       false,
			expectedProviders: []string{"google"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			cleanup := tt.setupEnvironment(t)
			defer cleanup()

			// Create config and test loadProviders
			config := &AuthConfig{
				Providers: make(map[string]ProviderConfig),
			}

			err := config.loadProviders()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.expectedErrorSubstr != "" && !strings.Contains(err.Error(), tt.expectedErrorSubstr) {
					t.Errorf("expected error to contain %q, got %q", tt.expectedErrorSubstr, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
			}

			// Check that expected providers exist
			for _, expectedProvider := range tt.expectedProviders {
				if _, exists := config.Providers[expectedProvider]; !exists {
					t.Errorf("expected provider %s to exist", expectedProvider)
				}
			}

			// Ensure provider count matches expectations (only if we expect specific providers)
			if len(tt.expectedProviders) > 0 {
				actualCount := 0
				for _, expected := range tt.expectedProviders {
					if _, exists := config.Providers[expected]; exists {
						actualCount++
					}
				}
				if actualCount != len(tt.expectedProviders) {
					t.Errorf("expected %d specific providers (%v), got %d matching", len(tt.expectedProviders), tt.expectedProviders, actualCount)
				}
			}
		})
	}
}

// TestValidate_EnhancedValidation tests the enhanced validation function
func TestValidate_EnhancedValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *AuthConfig
		expectError bool
		errorSubstr string
	}{
		{
			name: "auth disabled - skip validation",
			config: &AuthConfig{
				Enabled: false,
				// Invalid settings that should be ignored when auth is disabled
				JWTSecretKey: "",
				Providers:    make(map[string]ProviderConfig),
			},
			expectError: false,
		},
		{
			name: "empty JWT secret when auth enabled",
			config: &AuthConfig{
				Enabled:      true,
				JWTSecretKey: "",
				Providers:    make(map[string]ProviderConfig),
			},
			expectError: true,
			errorSubstr: "JWT_SECRET_KEY is required",
		},
		{
			name: "weak JWT secret",
			config: &AuthConfig{
				Enabled:      true,
				JWTSecretKey: "secret-but-long-enough-to-pass-length-check", // Weak secret but long enough
				Providers:    make(map[string]ProviderConfig),
			},
			expectError: true,
			errorSubstr: "weak or default secret detected",
		},
		{
			name: "JWT secret too short",
			config: &AuthConfig{
				Enabled:      true,
				JWTSecretKey: "short", // Too short
				Providers:    make(map[string]ProviderConfig),
			},
			expectError: true,
			errorSubstr: "must be at least 32 characters long",
		},
		{
			name: "repetitive JWT secret pattern",
			config: &AuthConfig{
				Enabled:      true,
				JWTSecretKey: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", // All same character
				Providers:    make(map[string]ProviderConfig),
			},
			expectError: true,
			errorSubstr: "repetitive pattern detected",
		},
		{
			name: "enabled provider with empty client secret",
			config: &AuthConfig{
				Enabled:      true,
				JWTSecretKey: "this-is-a-very-strong-jwt-unique-key-for-validation",
				Providers: map[string]ProviderConfig{
					"azure-ad": {
						Enabled:      true,
						Type:         "azure-ad",
						ClientID:     "test-client-id",
						ClientSecret: "", // Empty secret
						TenantID:     "test-tenant",
					},
				},
			},
			expectError: true,
			errorSubstr: "client_secret is required but not available",
		},
		{
			name: "provider-specific validation - Azure missing tenant",
			config: &AuthConfig{
				Enabled:      true,
				JWTSecretKey: "this-is-a-very-strong-jwt-unique-key-for-validation",
				Providers: map[string]ProviderConfig{
					"azure-ad": {
						Enabled:      true,
						Type:         "azure-ad",
						ClientID:     "test-client-id",
						ClientSecret: "valid-azure-secret-1234567890abcdef",
						TenantID:     "", // Missing required field
					},
				},
			},
			expectError: true,
			errorSubstr: "tenant_id is required for Azure AD",
		},
		{
			name: "provider-specific validation - Okta missing domain",
			config: &AuthConfig{
				Enabled:      true,
				JWTSecretKey: "this-is-a-very-strong-jwt-unique-key-for-validation",
				Providers: map[string]ProviderConfig{
					"okta": {
						Enabled:      true,
						Type:         "okta",
						ClientID:     "test-client-id",
						ClientSecret: "valid-okta-secret-1234567890abcdef1234567890abcdef1234567890abcdef",
						Domain:       "", // Missing required field
					},
				},
			},
			expectError: true,
			errorSubstr: "domain is required for Okta",
		},
		{
			name: "provider-specific validation - Keycloak missing base_url",
			config: &AuthConfig{
				Enabled:      true,
				JWTSecretKey: "this-is-a-very-strong-jwt-unique-key-for-validation",
				Providers: map[string]ProviderConfig{
					"keycloak": {
						Enabled:      true,
						Type:         "keycloak",
						ClientID:     "test-client-id",
						ClientSecret: "valid-keycloak-secret-12345678901234567890123456789012",
						BaseURL:      "", // Missing required field
						Realm:        "master",
					},
				},
			},
			expectError: true,
			errorSubstr: "base_url is required for Keycloak",
		},
		{
			name: "provider-specific validation - Custom missing URLs",
			config: &AuthConfig{
				Enabled:      true,
				JWTSecretKey: "this-is-a-very-strong-jwt-unique-key-for-validation",
				Providers: map[string]ProviderConfig{
					"custom": {
						Enabled:      true,
						Type:         "custom",
						ClientID:     "test-client-id",
						ClientSecret: "valid-custom-secret-1234567890abcdef",
						AuthURL:      "", // Missing required field
						TokenURL:     "https://example.com/token",
						UserInfoURL:  "https://example.com/userinfo",
					},
				},
			},
			expectError: true,
			errorSubstr: "auth_url is required for custom provider",
		},
		{
			name: "no enabled providers",
			config: &AuthConfig{
				Enabled:      true,
				JWTSecretKey: "this-is-a-very-strong-jwt-unique-key-for-validation",
				Providers: map[string]ProviderConfig{
					"azure-ad": {
						Enabled:      false, // All providers disabled
						Type:         "azure-ad",
						ClientID:     "test-client-id",
						ClientSecret: "valid-azure-secret-1234567890abcdef",
						TenantID:     "test-tenant",
					},
				},
			},
			expectError: true,
			errorSubstr: "at least one OAuth2 provider must be enabled",
		},
		{
			name: "valid configuration",
			config: &AuthConfig{
				Enabled:      true,
				JWTSecretKey: "this-is-a-very-strong-jwt-unique-key-for-validation",
				Providers: map[string]ProviderConfig{
					"azure-ad": {
						Enabled:      true,
						Type:         "azure-ad",
						ClientID:     "test-client-id",
						ClientSecret: "valid-azure-secret-1234567890abcdef",
						TenantID:     "test-tenant-id",
					},
					"google": {
						Enabled:      true,
						Type:         "google",
						ClientID:     "test-google-client-id",
						ClientSecret: "valid-google-secret-123456789012345678901234",
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorSubstr != "" && !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("expected error to contain %q, got %q", tt.errorSubstr, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestCreateOAuth2Providers_ErrorHandling tests error handling in CreateOAuth2Providers
func TestCreateOAuth2Providers_ErrorHandling(t *testing.T) {
	tests := []struct {
		name                string
		config              *AuthConfig
		expectError         bool
		expectedProviders   []string
		expectedErrorSubstr string
	}{
		{
			name: "invalid provider configuration - empty client_id",
			config: &AuthConfig{
				Providers: map[string]ProviderConfig{
					"azure-ad": {
						Enabled:      true,
						Type:         "azure-ad",
						ClientID:     "", // Empty client ID
						ClientSecret: "valid-azure-secret-1234567890abcdef",
						TenantID:     "test-tenant",
					},
				},
			},
			expectError:         true,
			expectedErrorSubstr: "client_id is empty",
		},
		{
			name: "invalid provider configuration - empty client_secret",
			config: &AuthConfig{
				Providers: map[string]ProviderConfig{
					"okta": {
						Enabled:      true,
						Type:         "okta",
						ClientID:     "test-client-id",
						ClientSecret: "", // Empty client secret
						Domain:       "test.okta.com",
					},
				},
			},
			expectError:         true,
			expectedErrorSubstr: "client_secret is empty",
		},
		{
			name: "partial success scenario - some providers succeed, others fail",
			config: &AuthConfig{
				Providers: map[string]ProviderConfig{
					"azure-ad": {
						Enabled:      true,
						Type:         "azure-ad",
						ClientID:     "test-azure-client-id",
						ClientSecret: "valid-azure-secret-1234567890abcdef",
						TenantID:     "test-tenant",
					},
					"okta": {
						Enabled:      true,
						Type:         "okta",
						ClientID:     "", // Empty - should fail
						ClientSecret: "valid-okta-secret-1234567890abcdef1234567890abcdef1234567890abcdef",
						Domain:       "test.okta.com",
					},
					"google": {
						Enabled:      true,
						Type:         "google",
						ClientID:     "test-google-client-id",
						ClientSecret: "valid-google-secret-123456789012345678901234",
					},
				},
			},
			expectError:       false, // Should succeed with partial results
			expectedProviders: []string{"azure-ad", "google"},
		},
		{
			name: "unknown provider type",
			config: &AuthConfig{
				Providers: map[string]ProviderConfig{
					"unknown": {
						Enabled:      true,
						Type:         "unknown-type",
						ClientID:     "test-client-id",
						ClientSecret: "valid-secret-1234567890abcdef",
					},
				},
			},
			expectError:         true,
			expectedErrorSubstr: "unknown provider type",
		},
		{
			name: "all providers fail",
			config: &AuthConfig{
				Providers: map[string]ProviderConfig{
					"azure-ad": {
						Enabled:      true,
						Type:         "azure-ad",
						ClientID:     "", // Invalid
						ClientSecret: "valid-azure-secret-1234567890abcdef",
						TenantID:     "test-tenant",
					},
					"okta": {
						Enabled:      true,
						Type:         "okta",
						ClientID:     "test-client-id",
						ClientSecret: "", // Invalid
						Domain:       "test.okta.com",
					},
				},
			},
			expectError:         true,
			expectedErrorSubstr: "failed to create any OAuth2 providers",
		},
		{
			name: "valid providers only",
			config: &AuthConfig{
				Providers: map[string]ProviderConfig{
					"azure-ad": {
						Enabled:      true,
						Type:         "azure-ad",
						ClientID:     "test-azure-client-id",
						ClientSecret: "valid-azure-secret-1234567890abcdef",
						TenantID:     "test-tenant",
					},
					"google": {
						Enabled:      true,
						Type:         "google",
						ClientID:     "test-google-client-id",
						ClientSecret: "valid-google-secret-123456789012345678901234",
					},
					"disabled-provider": {
						Enabled:      false, // Should be ignored
						Type:         "okta",
						ClientID:     "invalid",
						ClientSecret: "invalid",
					},
				},
			},
			expectError:       false,
			expectedProviders: []string{"azure-ad", "google"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providers, err := tt.config.CreateOAuth2Providers()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.expectedErrorSubstr != "" && !strings.Contains(err.Error(), tt.expectedErrorSubstr) {
					t.Errorf("expected error to contain %q, got %q", tt.expectedErrorSubstr, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}

				if providers == nil {
					t.Errorf("expected providers map but got nil")
					return
				}

				// Check expected providers exist
				for _, expectedProvider := range tt.expectedProviders {
					if _, exists := providers[expectedProvider]; !exists {
						t.Errorf("expected provider %s to exist", expectedProvider)
					}
				}

				// Check provider count
				if len(tt.expectedProviders) != len(providers) {
					t.Errorf("expected %d providers, got %d", len(tt.expectedProviders), len(providers))
				}
			}
		})
	}
}

// TestLoadAuthConfig_IntegrationTests tests the full LoadAuthConfig flow with various error scenarios
func TestLoadAuthConfig_IntegrationTests(t *testing.T) {
	originalAuditLogger := security.GlobalAuditLogger
	defer func() {
		security.GlobalAuditLogger = originalAuditLogger
	}()

	security.GlobalAuditLogger, _ = security.NewAuditLogger("", security.AuditLevelInfo)

	tests := []struct {
		name              string
		setupEnvironment  func(t *testing.T) (cleanup func())
		expectError       bool
		expectedErrorSubstr string
		validateResult    func(t *testing.T, config *AuthConfig)
	}{
		{
			name: "complete valid configuration",
			setupEnvironment: func(t *testing.T) func() {
				env := map[string]string{
					"AUTH_ENABLED":          "true",
					"JWT_SECRET_KEY":        "this-is-a-very-strong-jwt-unique-key-for-validation-purposes",
					"AZURE_CLIENT_ID":       "test-azure-client-id",
					"AZURE_CLIENT_SECRET":   "valid-azure-secret-1234567890abcdef",
					"AZURE_TENANT_ID":       "test-tenant-id",
					"GOOGLE_CLIENT_ID":      "test-google-client-id",
					"GOOGLE_CLIENT_SECRET":  "valid-google-secret-123456789012345678901234",
				}
				return setMultipleEnvVars(env)
			},
			expectError: false,
			validateResult: func(t *testing.T, config *AuthConfig) {
				if !config.Enabled {
					t.Errorf("expected auth to be enabled")
				}
				if len(config.Providers) < 2 {
					t.Errorf("expected at least 2 providers, got %d", len(config.Providers))
				}
				if _, exists := config.Providers["azure-ad"]; !exists {
					t.Errorf("expected azure-ad provider")
				}
				if _, exists := config.Providers["google"]; !exists {
					t.Errorf("expected google provider")
				}
			},
		},
		{
			name: "provider loading failure causes overall failure",
			setupEnvironment: func(t *testing.T) func() {
				env := map[string]string{
					"AUTH_ENABLED":         "true",
					"JWT_SECRET_KEY":       "this-is-a-very-strong-jwt-unique-key-for-validation-purposes",
					"AZURE_CLIENT_ID":      "test-azure-client-id",
					"AZURE_CLIENT_SECRET":  "", // Missing secret
					"AZURE_ENABLED":        "true",
				}
				return setMultipleEnvVars(env)
			},
			expectError:         true,
			expectedErrorSubstr: "failed to load providers",
		},
		{
			name: "validation failure after successful provider loading",
			setupEnvironment: func(t *testing.T) func() {
				env := map[string]string{
					"AUTH_ENABLED":         "true",
					"JWT_SECRET_KEY":       "weak", // Weak JWT secret
					"GOOGLE_CLIENT_ID":     "test-google-client-id",
					"GOOGLE_CLIENT_SECRET": "valid-google-secret-123456789012345678901234",
				}
				return setMultipleEnvVars(env)
			},
			expectError:         true,
			expectedErrorSubstr: "invalid configuration",
		},
		{
			name: "auth disabled with invalid providers - should succeed",
			setupEnvironment: func(t *testing.T) func() {
				env := map[string]string{
					"AUTH_ENABLED":         "false",
					"JWT_SECRET_KEY":       "", // Empty but auth disabled
					// Don't set any provider config when auth is disabled
				}
				return setMultipleEnvVars(env)
			},
			expectError: false,
			validateResult: func(t *testing.T, config *AuthConfig) {
				if config.Enabled {
					t.Errorf("expected auth to be disabled")
				}
			},
		},
		{
			name: "partial provider success with auth enabled",
			setupEnvironment: func(t *testing.T) func() {
				env := map[string]string{
					"AUTH_ENABLED":         "true",
					"JWT_SECRET_KEY":       "this-is-a-very-strong-jwt-unique-key-for-validation-purposes",
					"AZURE_CLIENT_ID":      "test-azure-client-id",
					"AZURE_CLIENT_SECRET":  "", // Will fail
					"AZURE_ENABLED":        "true",
					"GOOGLE_CLIENT_ID":     "test-google-client-id",
					"GOOGLE_CLIENT_SECRET": "valid-google-secret-123456789012345678901234",
					"GOOGLE_ENABLED":       "true",
				}
				return setMultipleEnvVars(env)
			},
			expectError:         true,
			expectedErrorSubstr: "failed to load providers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			cleanup := tt.setupEnvironment(t)
			defer cleanup()

			config, err := LoadAuthConfig("")

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.expectedErrorSubstr != "" && !strings.Contains(err.Error(), tt.expectedErrorSubstr) {
					t.Errorf("expected error to contain %q, got %q", tt.expectedErrorSubstr, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if config == nil {
					t.Errorf("expected config but got nil")
					return
				}
				if tt.validateResult != nil {
					tt.validateResult(t, config)
				}
			}
		})
	}
}

// TestLoadAuthConfigWithCustomPath tests comprehensive custom auth config file path functionality
func TestLoadAuthConfigWithCustomPath(t *testing.T) {
	// Save original audit logger
	originalAuditLogger := security.GlobalAuditLogger
	defer func() {
		security.GlobalAuditLogger = originalAuditLogger
	}()
	security.GlobalAuditLogger, _ = security.NewAuditLogger("", security.AuditLevelInfo)

	tests := []struct {
		name                  string
		configPath            string
		setupEnvironment      func(t *testing.T) (customFile, envFile string, cleanup func())
		expectError           bool
		expectedErrorSubstring string
		validateConfig        func(*testing.T, *AuthConfig)
	}{
		{
			name:       "custom path takes precedence over environment variable",
			configPath: "custom-config.json", // Will be replaced with temp file path
			setupEnvironment: func(t *testing.T) (string, string, func()) {
				// Create temporary files
				customFile := filepath.Join(t.TempDir(), "custom-config.json")
				envFile := filepath.Join(t.TempDir(), "env-config.json")
				
				customContent := `{
					"enabled": true,
					"jwt_secret_key": "custom-jwt-secret-key-from-file-12345678901234567890",
					"providers": {
						"test-custom": {
							"enabled": true,
							"type": "custom",
							"client_id": "custom-file-client-id",
							"client_secret": "custom-file-secret-1234567890abcdef",
							"auth_url": "https://custom.example.com/auth",
							"token_url": "https://custom.example.com/token",
							"user_info_url": "https://custom.example.com/userinfo"
						}
					},
					"rbac": {
						"enabled": true,
						"default_role": "viewer"
					}
				}`
				
				envContent := `{
					"enabled": false,
					"providers": {
						"env-provider": {
							"enabled": true,
							"type": "google"
						}
					}
				}`
				
				// Write custom config file
				if err := os.WriteFile(customFile, []byte(customContent), 0644); err != nil {
					t.Fatalf("Failed to create custom config file: %v", err)
				}
				
				// Write env config file
				if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
					t.Fatalf("Failed to create env config file: %v", err)
				}
				
				// Set environment variables
				os.Setenv("AUTH_CONFIG_FILE", envFile)
				os.Setenv("AUTH_ENABLED", "false") // Should be overridden by custom file
				
				cleanup := func() {
					os.Unsetenv("AUTH_CONFIG_FILE")
					os.Unsetenv("AUTH_ENABLED")
				}
				
				return customFile, envFile, cleanup
			},
			expectError: false,
			validateConfig: func(t *testing.T, config *AuthConfig) {
				if config == nil {
					t.Fatal("config should not be nil")
				}
				// Should use values from custom file, not env file
				if !config.Enabled {
					t.Error("expected auth to be enabled from custom config file")
				}
				if config.JWTSecretKey != "custom-jwt-secret-key-from-file-12345678901234567890" {
					t.Errorf("expected JWT secret from custom file, got: %s", config.JWTSecretKey)
				}
				if _, exists := config.Providers["test-custom"]; !exists {
					t.Error("expected custom provider from custom config file")
				}
			},
		},
		{
			name:          "empty config path falls back to AUTH_CONFIG_FILE environment variable",
			configPath:    "", // Empty - should use env var
			setupEnvironment: func(t *testing.T) (string, string, func()) {
				envFile := filepath.Join(t.TempDir(), "env-fallback-config.json")
				
				envContent := `{
					"enabled": true,
					"jwt_secret_key": "env-jwt-secret-key-from-file-12345678901234567890",
					"providers": {
						"env-provider": {
							"enabled": true,
							"type": "google",
							"client_id": "env-google-client-id",
							"client_secret": "env-google-secret-123456789012345678901234"
						}
					}
				}`
				
				if err := os.WriteFile(envFile, []byte(envContent), 0644); err != nil {
					t.Fatalf("Failed to create env config file: %v", err)
				}
				
				os.Setenv("AUTH_CONFIG_FILE", envFile)
				os.Setenv("AUTH_ENABLED", "false") // Should be overridden by file
				
				cleanup := func() {
					os.Unsetenv("AUTH_CONFIG_FILE")
					os.Unsetenv("AUTH_ENABLED")
				}
				
				return "", envFile, cleanup
			},
			expectError: false,
			validateConfig: func(t *testing.T, config *AuthConfig) {
				if config == nil {
					t.Fatal("config should not be nil")
				}
				if !config.Enabled {
					t.Error("expected auth to be enabled from env config file")
				}
				if config.JWTSecretKey != "env-jwt-secret-key-from-file-12345678901234567890" {
					t.Errorf("expected JWT secret from env file, got: %s", config.JWTSecretKey)
				}
				if _, exists := config.Providers["env-provider"]; !exists {
					t.Error("expected env provider from env config file")
				}
			},
		},
		{
			name:                   "custom path with non-existent file returns error",
			configPath:             "/absolutely/non/existent/path/config.json",
			setupEnvironment: func(t *testing.T) (string, string, func()) {
				os.Setenv("JWT_SECRET_KEY", "fallback-jwt-secret-12345678901234567890")
				cleanup := func() {
					os.Unsetenv("JWT_SECRET_KEY")
				}
				return "", "", cleanup
			},
			expectError:            true,
			expectedErrorSubstring: "failed to load config file",
		},
		{
			name:       "custom path with invalid JSON returns error",
			configPath: "invalid-json-config.json",
			setupEnvironment: func(t *testing.T) (string, string, func()) {
				customFile := filepath.Join(t.TempDir(), "invalid-json-config.json")
				
				invalidContent := `{
					"enabled": true,
					"providers": {
						"invalid-json": {
							"enabled": true,
							// Invalid JSON comment
							"client_id": "test"
						}
					// Missing closing brace
				}`
				
				if err := os.WriteFile(customFile, []byte(invalidContent), 0644); err != nil {
					t.Fatalf("Failed to create custom config file: %v", err)
				}
				
				os.Setenv("JWT_SECRET_KEY", "fallback-jwt-secret-12345678901234567890")
				
				cleanup := func() {
					os.Unsetenv("JWT_SECRET_KEY")
				}
				
				return customFile, "", cleanup
			},
			expectError:            true,
			expectedErrorSubstring: "failed to load config file",
		},
		{
			name:       "both custom path and env var empty - uses environment variables only",
			configPath: "",
			setupEnvironment: func(t *testing.T) (string, string, func()) {
				os.Setenv("AUTH_CONFIG_FILE", "") // Explicit empty
				os.Setenv("AUTH_ENABLED", "true")
				os.Setenv("JWT_SECRET_KEY", "env-only-jwt-secret-12345678901234567890")
				os.Setenv("GOOGLE_CLIENT_ID", "env-google-client-id")
				os.Setenv("GOOGLE_CLIENT_SECRET", "env-google-secret-123456789012345678901234")
				
				cleanup := func() {
					os.Unsetenv("AUTH_CONFIG_FILE")
					os.Unsetenv("AUTH_ENABLED")
					os.Unsetenv("JWT_SECRET_KEY")
					os.Unsetenv("GOOGLE_CLIENT_ID")
					os.Unsetenv("GOOGLE_CLIENT_SECRET")
				}
				
				return "", "", cleanup
			},
			expectError: false,
			validateConfig: func(t *testing.T, config *AuthConfig) {
				if config == nil {
					t.Fatal("config should not be nil")
				}
				if !config.Enabled {
					t.Error("expected auth to be enabled from environment variables")
				}
				if config.JWTSecretKey != "env-only-jwt-secret-12345678901234567890" {
					t.Errorf("expected JWT secret from environment, got: %s", config.JWTSecretKey)
				}
				if _, exists := config.Providers["google"]; !exists {
					t.Error("expected google provider from environment variables")
				}
			},
		},
		{
			name:       "custom file overrides environment variables and providers",
			configPath: "override-config.json",
			setupEnvironment: func(t *testing.T) (string, string, func()) {
				customFile := filepath.Join(t.TempDir(), "override-config.json")
				
				overrideContent := `{
					"enabled": false,
					"jwt_secret_key": "file-overrides-env-jwt-secret-12345678901234567890",
					"token_ttl": "2h",
					"refresh_ttl": "48h",
					"providers": {
						"file-azure": {
							"enabled": true,
							"type": "azure-ad",
							"client_id": "file-azure-client-id",
							"client_secret": "file-azure-secret-1234567890abcdef",
							"tenant_id": "file-azure-tenant-id",
							"scopes": ["openid", "profile", "email", "User.Read"]
						}
					},
					"rbac": {
						"enabled": true,
						"default_role": "operator",
						"admin_roles": ["admin", "super-admin"],
						"operator_roles": ["operator", "network-admin"],
						"readonly_roles": ["viewer", "guest"]
					},
					"admin_users": ["admin@example.com"],
					"operator_users": ["operator@example.com"]
				}`
				
				if err := os.WriteFile(customFile, []byte(overrideContent), 0644); err != nil {
					t.Fatalf("Failed to create custom config file: %v", err)
				}
				
				// Set conflicting environment variables that should be overridden
				os.Setenv("AUTH_ENABLED", "true")
				os.Setenv("JWT_SECRET_KEY", "env-jwt-secret-that-should-be-overridden")
				os.Setenv("GOOGLE_CLIENT_ID", "env-google-client-id")
				os.Setenv("GOOGLE_CLIENT_SECRET", "env-google-secret-123456789012345678901234")
				
				cleanup := func() {
					os.Unsetenv("AUTH_ENABLED")
					os.Unsetenv("JWT_SECRET_KEY")
					os.Unsetenv("GOOGLE_CLIENT_ID")
					os.Unsetenv("GOOGLE_CLIENT_SECRET")
				}
				
				return customFile, "", cleanup
			},
			expectError: false,
			validateConfig: func(t *testing.T, config *AuthConfig) {
				if config == nil {
					t.Fatal("config should not be nil")
				}
				// File should override environment
				if config.Enabled {
					t.Error("expected auth to be disabled per config file override")
				}
				if config.JWTSecretKey != "file-overrides-env-jwt-secret-12345678901234567890" {
					t.Errorf("expected JWT secret from file override, got: %s", config.JWTSecretKey)
				}
				if config.TokenTTL.String() != "2h0m0s" {
					t.Errorf("expected token TTL from file, got: %s", config.TokenTTL)
				}
				if config.RefreshTTL.String() != "48h0m0s" {
					t.Errorf("expected refresh TTL from file, got: %s", config.RefreshTTL)
				}
				if _, exists := config.Providers["file-azure"]; !exists {
					t.Error("expected azure provider from file")
				}
				if _, exists := config.Providers["google"]; exists {
					t.Error("should not have google provider from env when file is provided")
				}
				if config.RBAC.DefaultRole != "operator" {
					t.Errorf("expected RBAC default role from file, got: %s", config.RBAC.DefaultRole)
				}
				if len(config.AdminUsers) != 1 || config.AdminUsers[0] != "admin@example.com" {
					t.Errorf("expected admin users from file, got: %v", config.AdminUsers)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			customFile, envFile, cleanup := tt.setupEnvironment(t)
			defer cleanup()
			
			// Use the actual file paths created by the setup
			configPath := tt.configPath
			if customFile != "" {
				configPath = customFile
			}

			// Call LoadAuthConfig with the determined config path
			config, err := LoadAuthConfig(configPath)

			// Validate error expectations
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
					return
				}
				if tt.expectedErrorSubstring != "" && !strings.Contains(err.Error(), tt.expectedErrorSubstring) {
					t.Errorf("expected error to contain %q, got %q", tt.expectedErrorSubstring, err.Error())
				}
				return
			}

			// Validate success cases
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if config == nil {
				t.Fatal("expected valid config but got nil")
			}

			if tt.validateConfig != nil {
				tt.validateConfig(t, config)
			}
		})
	}
}

// Helper functions are defined in config_security_test.go

// TestValidateOAuth2ClientSecret_ProviderSpecific tests provider-specific secret validation
func TestValidateOAuth2ClientSecret_ProviderSpecific(t *testing.T) {
	tests := []struct {
		name      string
		provider  string
		secret    string
		expectError bool
		errorSubstr string
	}{
		{
			name:      "empty secret",
			provider:  "azure",
			secret:    "",
			expectError: true,
			errorSubstr: "empty secret",
		},
		{
			name:      "whitespace-only secret",
			provider:  "azure",
			secret:    "   \t\n   ",
			expectError: true,
			errorSubstr: "empty secret",
		},
		{
			name:      "azure secret too short",
			provider:  "azure",
			secret:    "short",
			expectError: true,
			errorSubstr: "secret too short",
		},
		{
			name:      "azure valid secret",
			provider:  "azure",
			secret:    "valid-azure-secret-1234567890",
			expectError: false,
		},
		{
			name:      "okta secret too short",
			provider:  "okta",
			secret:    "too-short-secret",
			expectError: true,
			errorSubstr: "secret too short",
		},
		{
			name:      "okta valid secret",
			provider:  "okta",
			secret:    "valid-okta-secret-1234567890abcdef1234567890abcdef1234567890",
			expectError: false,
		},
		{
			name:      "keycloak secret too short",
			provider:  "keycloak",
			secret:    "short-keycloak-secret",
			expectError: true,
			errorSubstr: "secret too short",
		},
		{
			name:      "keycloak valid secret",
			provider:  "keycloak",
			secret:    "valid-keycloak-secret-12345678901234567890123456789012",
			expectError: false,
		},
		{
			name:      "google secret too short",
			provider:  "google",
			secret:    "google-short",
			expectError: true,
			errorSubstr: "secret too short",
		},
		{
			name:      "google valid secret",
			provider:  "google",
			secret:    "valid-google-secret-123456789012345678901234",
			expectError: false,
		},
		{
			name:      "custom provider secret too short",
			provider:  "custom",
			secret:    "custom-short",
			expectError: true,
			errorSubstr: "secret too short",
		},
		{
			name:      "custom provider valid secret",
			provider:  "custom",
			secret:    "valid-custom-secret-1234567890",
			expectError: false,
		},
		{
			name:      "placeholder secret detected - your-secret",
			provider:  "azure",
			secret:    "your-secret-here-changeme-1234567890",
			expectError: true,
			errorSubstr: "placeholder secret detected",
		},
		{
			name:      "placeholder secret detected - client-secret",
			provider:  "okta",
			secret:    "client-secret-placeholder-1234567890abcdef1234567890abcdef1234567890",
			expectError: true,
			errorSubstr: "placeholder secret detected",
		},
		{
			name:      "placeholder secret detected - changeme",
			provider:  "keycloak",
			secret:    "changeme-keycloak-secret-1234567890123456789012",
			expectError: true,
			errorSubstr: "placeholder secret detected",
		},
		{
			name:      "placeholder secret detected - example",
			provider:  "google",
			secret:    "example-google-secret-123456789012345678901234",
			expectError: true,
			errorSubstr: "placeholder secret detected",
		},
		{
			name:      "simple secret placeholder",
			provider:  "custom",
			secret:    "secret",
			expectError: true,
			errorSubstr: "secret too short", // "secret" is only 6 chars, less than 16 minimum for custom
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOAuth2ClientSecret(tt.provider, tt.secret)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorSubstr != "" && !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("expected error to contain %q, got %q", tt.errorSubstr, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestValidateJWTSecret_WeakSecrets tests JWT secret validation for weak secrets
func TestValidateJWTSecret_WeakSecrets(t *testing.T) {
	tests := []struct {
		name      string
		secret    string
		expectError bool
		errorSubstr string
	}{
		{
			name:      "weak secret - secret",
			secret:    "secret",
			expectError: true,
			errorSubstr: "weak or default secret detected",
		},
		{
			name:      "weak secret - changeme",
			secret:    "changeme-but-still-weak-1234567890",
			expectError: true,
			errorSubstr: "weak or default secret detected",
		},
		{
			name:      "weak secret - password",
			secret:    "password-is-weak-1234567890123456",
			expectError: true,
			errorSubstr: "weak or default secret detected",
		},
		{
			name:      "weak secret - 12345678",
			secret:    "12345678-weak-pattern-1234567890123456",
			expectError: true,
			errorSubstr: "weak or default secret detected",
		},
		{
			name:      "weak secret - default",
			secret:    "default-jwt-secret-1234567890123456",
			expectError: true,
			errorSubstr: "weak or default secret detected",
		},
		{
			name:      "weak secret - admin",
			secret:    "admin-secret-key-1234567890123456",
			expectError: true,
			errorSubstr: "weak or default secret detected",
		},
		{
			name:      "weak secret - test",
			secret:    "test-jwt-secret-key-1234567890123456",
			expectError: true,
			errorSubstr: "weak or default secret detected",
		},
		{
			name:      "weak secret - demo",
			secret:    "demo-application-secret-1234567890123456",
			expectError: true,
			errorSubstr: "weak or default secret detected",
		},
		{
			name:      "repetitive pattern - all same character",
			secret:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			expectError: true,
			errorSubstr: "repetitive pattern detected",
		},
		{
			name:      "repetitive pattern - all zeros",
			secret:    "00000000000000000000000000000000000000",
			expectError: true,
			errorSubstr: "repetitive pattern detected",
		},
		{
			name:      "strong secret",
			secret:    "R@nd0m!Str0ng#JWT$S3cr3t%K3y&F0r*T3st1ng+2024",
			expectError: false,
		},
		{
			name:      "UUID-like strong secret",
			secret:    "a7b8c9d0-e1f2-4a5b-8c9d-0e1f2a3b4c5d-jwt-strong-key",
			expectError: false,
		},
		{
			name:      "base64-like strong secret",
			secret:    "YWJjZGVmZ2hpams987654321+/ABCDEFGHIJKLmnopqrstuvwxyz",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJWTSecret(tt.secret)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorSubstr != "" && !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("expected error to contain %q, got %q", tt.errorSubstr, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}