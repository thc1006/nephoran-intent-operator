package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thc1006/nephoran-intent-operator/pkg/security"
)

// TestGetOAuth2ClientSecret_EnvironmentVariables tests environment variable loading for various providers
func TestGetOAuth2ClientSecret_EnvironmentVariables(t *testing.T) {
	// Save original environment and global audit logger
	originalAuditLogger := security.GlobalAuditLogger
	defer func() {
		security.GlobalAuditLogger = originalAuditLogger
	}()

	// Create a real audit logger for testing but with null output
	security.GlobalAuditLogger, _ = security.NewAuditLogger("", security.AuditLevelInfo)

	tests := []struct {
		name         string
		provider     string
		secret       string
		expectError  bool
		errorSubstr  string
	}{
		{
			name:        "AzureAD valid secret from environment",
			provider:    "azure",
			secret:      "valid-azure-secret-1234567890abcdef",
			expectError: false,
		},
		{
			name:        "AzureAD with hyphens in provider name",
			provider:    "azure-ad",
			secret:      "valid-azure-secret-1234567890abcdef",
			expectError: false,
		},
		{
			name:        "Keycloak valid secret from environment",
			provider:    "keycloak",
			secret:      "valid-keycloak-secret-12345678901234567890123456789012",
			expectError: false,
		},
		{
			name:        "Okta valid secret from environment",
			provider:    "okta",
			secret:      "valid-okta-secret-1234567890abcdef1234567890abcdef1234567890abcdef",
			expectError: false,
		},
		{
			name:        "Google valid secret from environment",
			provider:    "google",
			secret:      "valid-google-secret-123456789012345678901234",
			expectError: false,
		},
		{
			name:        "Custom provider valid secret",
			provider:    "custom",
			secret:      "valid-custom-secret-1234567890abcdef",
			expectError: false,
		},
		{
			name:        "weak secret with password pattern",
			provider:    "azure",
			secret:      "password-azure-secret-1234567890abcdef",
			expectError: true,
			errorSubstr: "invalid OAuth2 client secret format",
		},
		{
			name:        "weak secret with test pattern", 
			provider:    "keycloak",
			secret:      "test-keycloak-secret-12345678901234567890123456789012",
			expectError: true,
			errorSubstr: "invalid OAuth2 client secret format",
		},
		{
			name:        "secret too short",
			provider:    "azure",
			secret:      "short",
			expectError: true,
			errorSubstr: "invalid OAuth2 client secret format",
		},
		{
			name:        "whitespace-only secret",
			provider:    "okta",
			secret:      "   \t\n   ",
			expectError: true,
			errorSubstr: "invalid OAuth2 client secret format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean environment first
			clearOAuth2Environment(t, tt.provider)
			
			if tt.secret != "" {
				// Set environment variable with the expected naming convention
				envVar := strings.ToUpper(strings.ReplaceAll(tt.provider, "-", "_"))
				envVarName := "OAUTH2_" + envVar + "_CLIENT_SECRET"
				t.Setenv(envVarName, tt.secret)
			}

			// Test the function
			secret, err := getOAuth2ClientSecret(tt.provider)

			// Validate results
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorSubstr != "" && !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("expected error message to contain %q, got %q", tt.errorSubstr, err.Error())
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
				if secret != strings.TrimSpace(tt.secret) {
					t.Errorf("expected secret %q, got %q", strings.TrimSpace(tt.secret), secret)
				}
			}
		})
	}
}

// TestGetOAuth2ClientSecret_FileBasedSecrets tests file-based secret loading
func TestGetOAuth2ClientSecret_FileBasedSecrets(t *testing.T) {
	// Save original environment and global audit logger
	originalAuditLogger := security.GlobalAuditLogger
	defer func() {
		security.GlobalAuditLogger = originalAuditLogger
	}()

	// Create a real audit logger for testing but with null output
	security.GlobalAuditLogger, _ = security.NewAuditLogger("", security.AuditLevelInfo)

	tests := []struct {
		name         string
		provider     string
		secretContent string
		expectError  bool
		errorSubstr  string
	}{
		{
			name:         "AzureAD valid secret from file",
			provider:     "azure",
			secretContent: "valid-azure-secret-1234567890abcdef",
			expectError:  false,
		},
		{
			name:         "Keycloak valid secret from file",
			provider:     "keycloak", 
			secretContent: "valid-keycloak-secret-12345678901234567890123456789012",
			expectError:  false,
		},
		{
			name:         "Okta valid secret from file",
			provider:     "okta",
			secretContent: "valid-okta-secret-1234567890abcdef1234567890abcdef1234567890abcdef",
			expectError:  false,
		},
		{
			name:         "Google valid secret from file",
			provider:     "google",
			secretContent: "valid-google-secret-123456789012345678901234",
			expectError:  false,
		},
		{
			name:         "Custom provider valid secret from file",
			provider:     "custom",
			secretContent: "valid-custom-secret-1234567890abcdef", 
			expectError:  false,
		},
		{
			name:         "file with weak secret content",
			provider:     "azure",
			secretContent: "example-weak-secret-1234567890abcdef",
			expectError:  true,
			errorSubstr:  "invalid OAuth2 client secret format",
		},
		{
			name:         "empty file",
			provider:     "okta", 
			secretContent: "",
			expectError:  true,
			errorSubstr:  "file not accessible",
		},
		{
			name:         "whitespace-only file content",
			provider:     "google",
			secretContent: "   \t\n   ",
			expectError:  true,
			errorSubstr:  "invalid OAuth2 client secret format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean environment first
			clearOAuth2Environment(t, tt.provider)

			// Create temporary directory for testing
			tempDir := t.TempDir()
			secretFile := filepath.Join(tempDir, "secret")

			if tt.secretContent != "" {
				// Create secret file with appropriate permissions
				if err := os.WriteFile(secretFile, []byte(tt.secretContent), 0600); err != nil {
					t.Fatalf("Failed to create test secret file: %v", err)
				}
			}
			// For empty content test, don't create the file at all

			// Set file path environment variable
			envVar := strings.ToUpper(strings.ReplaceAll(tt.provider, "-", "_"))
			fileEnvVar := "OAUTH2_" + envVar + "_SECRET_FILE"
			t.Setenv(fileEnvVar, secretFile)

			// Test the function
			secret, err := getOAuth2ClientSecret(tt.provider)

			// Validate results
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorSubstr != "" && !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("expected error message to contain %q, got %q", tt.errorSubstr, err.Error())
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
				if secret != strings.TrimSpace(tt.secretContent) {
					t.Errorf("expected secret %q, got %q", strings.TrimSpace(tt.secretContent), secret)
				}
			}
		})
	}
}

// TestGetOAuth2ClientSecret_EnvironmentPrecedence tests that environment variables take precedence over files
func TestGetOAuth2ClientSecret_EnvironmentPrecedence(t *testing.T) {
	originalAuditLogger := security.GlobalAuditLogger
	defer func() {
		security.GlobalAuditLogger = originalAuditLogger
	}()
	security.GlobalAuditLogger, _ = security.NewAuditLogger("", security.AuditLevelInfo)

	provider := "azure"
	envSecret := "env-secret-1234567890abcdef"
	fileSecret := "file-secret-1234567890abcdef"

	// Clean environment first
	clearOAuth2Environment(t, provider)

	// Create temporary file
	tempDir := t.TempDir()
	secretFile := filepath.Join(tempDir, "secret")
	if err := os.WriteFile(secretFile, []byte(fileSecret), 0600); err != nil {
		t.Fatalf("Failed to create test secret file: %v", err)
	}

	// Set both environment variable and file path
	t.Setenv("OAUTH2_AZURE_CLIENT_SECRET", envSecret)
	t.Setenv("OAUTH2_AZURE_SECRET_FILE", secretFile)

	// Test the function
	secret, err := getOAuth2ClientSecret(provider)

	// Should get environment variable, not file content
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	if secret != envSecret {
		t.Errorf("expected environment secret %q, got %q (should prioritize env over file)", envSecret, secret)
	}
}

// TestGetOAuth2ClientSecret_ErrorCases tests various error scenarios
func TestGetOAuth2ClientSecret_ErrorCases(t *testing.T) {
	originalAuditLogger := security.GlobalAuditLogger
	defer func() {
		security.GlobalAuditLogger = originalAuditLogger
	}()
	security.GlobalAuditLogger, _ = security.NewAuditLogger("", security.AuditLevelInfo)

	tests := []struct {
		name              string
		provider          string
		setupEnvironment  func(t *testing.T) (cleanup func())
		expectError       bool
		expectedErrorSubstr string
	}{
		{
			name:              "empty provider name",
			provider:          "",
			setupEnvironment:  func(t *testing.T) func() { return func() {} },
			expectError:       true,
			expectedErrorSubstr: "provider name cannot be empty",
		},
		{
			name:              "invalid provider name format - special characters",
			provider:          "invalid@provider#name",
			setupEnvironment:  func(t *testing.T) func() { return func() {} },
			expectError:       true,
			expectedErrorSubstr: "invalid provider name format",
		},
		{
			name:              "invalid provider name format - too long",
			provider:          "verylongprovidernamethatexceedsthelimitssetforsecurityreasons",
			setupEnvironment:  func(t *testing.T) func() { return func() {} },
			expectError:       true,
			expectedErrorSubstr: "invalid provider name format",
		},
		{
			name:     "neither environment nor file configured",
			provider: "azure",
			setupEnvironment: func(t *testing.T) func() {
				clearOAuth2Environment(t, "azure")
				return func() {}
			},
			expectError:       true,
			expectedErrorSubstr: "not configured for provider",
		},
		{
			name:     "nonexistent file path",
			provider: "okta",
			setupEnvironment: func(t *testing.T) func() {
				clearOAuth2Environment(t, "okta")
				t.Setenv("OAUTH2_OKTA_SECRET_FILE", "/nonexistent/path/secret")
				return func() {}
			},
			expectError:       true,
			expectedErrorSubstr: "file not accessible",
		},
		{
			name:     "path traversal attempt",
			provider: "keycloak",
			setupEnvironment: func(t *testing.T) func() {
				clearOAuth2Environment(t, "keycloak")
				t.Setenv("OAUTH2_KEYCLOAK_SECRET_FILE", "../../../etc/passwd")
				return func() {}
			},
			expectError:       true,
			expectedErrorSubstr: "configuration error",
		},
		{
			name:     "secret too long",
			provider: "google",
			setupEnvironment: func(t *testing.T) func() {
				clearOAuth2Environment(t, "google")
				// Create a secret that's too long (over 512 characters)
				longSecret := strings.Repeat("a", 600)
				t.Setenv("OAUTH2_GOOGLE_CLIENT_SECRET", longSecret)
				return func() {}
			},
			expectError:       true,
			expectedErrorSubstr: "invalid OAuth2 client secret format",
		},
		{
			name:     "low entropy secret - all same character",
			provider: "custom",
			setupEnvironment: func(t *testing.T) func() {
				clearOAuth2Environment(t, "custom")
				t.Setenv("OAUTH2_CUSTOM_CLIENT_SECRET", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
				return func() {}
			},
			expectError:       true,
			expectedErrorSubstr: "invalid OAuth2 client secret format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			cleanup := tt.setupEnvironment(t)
			defer cleanup()

			// Test the function
			secret, err := getOAuth2ClientSecret(tt.provider)

			// Validate results
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.expectedErrorSubstr != "" && !strings.Contains(err.Error(), tt.expectedErrorSubstr) {
					t.Errorf("expected error message to contain %q, got %q", tt.expectedErrorSubstr, err.Error())
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

// TestGetOAuth2ClientSecret_SecurityValidation tests security validations
func TestGetOAuth2ClientSecret_SecurityValidation(t *testing.T) {
	originalAuditLogger := security.GlobalAuditLogger
	defer func() {
		security.GlobalAuditLogger = originalAuditLogger
	}()
	security.GlobalAuditLogger, _ = security.NewAuditLogger("", security.AuditLevelInfo)

	tests := []struct {
		name      string
		provider  string
		secret    string
		expectError bool
		errorSubstr string
	}{
		{
			name:      "weak pattern - password",
			provider:  "azure",
			secret:    "password-azure-secret-1234567890abcdef",
			expectError: true,
			errorSubstr: "invalid OAuth2 client secret format",
		},
		{
			name:      "weak pattern - secret",
			provider:  "okta",
			secret:    "secret-okta-value-1234567890abcdef1234567890abcdef1234567890abcdef",
			expectError: true,
			errorSubstr: "invalid OAuth2 client secret format",
		},
		{
			name:      "weak pattern - test",
			provider:  "keycloak",
			secret:    "test-keycloak-secret-12345678901234567890123456789012",
			expectError: true,
			errorSubstr: "invalid OAuth2 client secret format",
		},
		{
			name:      "weak pattern - example",
			provider:  "google",
			secret:    "example-google-secret-123456789012345678901234",
			expectError: true,
			errorSubstr: "invalid OAuth2 client secret format",
		},
		{
			name:      "insufficient entropy - all same character",
			provider:  "custom",
			secret:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			expectError: true,
			errorSubstr: "invalid OAuth2 client secret format",
		},
		{
			name:      "too short secret",
			provider:  "azure",
			secret:    "short",
			expectError: true,
			errorSubstr: "invalid OAuth2 client secret format",
		},
		{
			name:      "too long secret",
			provider:  "okta", 
			secret:    strings.Repeat("a", 600),
			expectError: true,
			errorSubstr: "invalid OAuth2 client secret format",
		},
		{
			name:      "valid strong secret - random",
			provider:  "azure",
			secret:    "R@nd0m!Str0ng#Az$ur3%S3cr3t&K3y*F0r+T3st1ng",
			expectError: false,
		},
		{
			name:      "valid strong secret - UUID-like",
			provider:  "keycloak",
			secret:    "a7b8c9d0-e1f2-4a5b-8c9d-0e1f2a3b4c5d-jwt-strong-key-12345678901234567890123456789012",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean environment first
			clearOAuth2Environment(t, tt.provider)

			// Set environment variable
			envVar := strings.ToUpper(strings.ReplaceAll(tt.provider, "-", "_"))
			envVarName := "OAUTH2_" + envVar + "_CLIENT_SECRET"
			t.Setenv(envVarName, tt.secret)

			// Test the function
			secret, err := getOAuth2ClientSecret(tt.provider)

			// Validate results
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorSubstr != "" && !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("expected error message to contain %q, got %q", tt.errorSubstr, err.Error())
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

// TestLoadProviders_WithNewSecretLoading tests integration with loadProviders function
func TestLoadProviders_WithNewSecretLoading(t *testing.T) {
	originalAuditLogger := security.GlobalAuditLogger
	defer func() {
		security.GlobalAuditLogger = originalAuditLogger
	}()
	security.GlobalAuditLogger, _ = security.NewAuditLogger("", security.AuditLevelInfo)

	tests := []struct {
		name                string
		setupEnvironment    func(t *testing.T) (cleanup func())
		expectError         bool
		expectedErrorSubstr string
		expectedProviders   []string
	}{
		{
			name: "AzureAD with valid environment secret",
			setupEnvironment: func(t *testing.T) func() {
				env := map[string]string{
					"AZURE_CLIENT_ID":       "test-azure-client-id",
					"AZURE_ENABLED":         "true",
					"OAUTH2_AZURE_CLIENT_SECRET": "valid-azure-secret-1234567890abcdef",
					"AZURE_TENANT_ID":       "test-tenant-id",
				}
				return setMultipleEnvVars(env)
			},
			expectError:       false,
			expectedProviders: []string{"azure-ad"},
		},
		{
			name: "Keycloak with valid file-based secret",
			setupEnvironment: func(t *testing.T) func() {
				// Create temporary file with secret
				tempDir := t.TempDir()
				secretFile := filepath.Join(tempDir, "keycloak-secret")
				if err := os.WriteFile(secretFile, []byte("valid-keycloak-secret-12345678901234567890123456789012"), 0600); err != nil {
					t.Fatalf("Failed to create test secret file: %v", err)
				}

				env := map[string]string{
					"KEYCLOAK_CLIENT_ID":          "test-keycloak-client-id",
					"KEYCLOAK_ENABLED":            "true",
					"OAUTH2_KEYCLOAK_SECRET_FILE": secretFile,
					"KEYCLOAK_BASE_URL":           "https://keycloak.example.com",
				}
				return setMultipleEnvVars(env)
			},
			expectError:       false,
			expectedProviders: []string{"keycloak"},
		},
		{
			name: "AzureAD with weak secret should fail",
			setupEnvironment: func(t *testing.T) func() {
				env := map[string]string{
					"AZURE_CLIENT_ID":       "test-azure-client-id", 
					"AZURE_ENABLED":         "true",
					"OAUTH2_AZURE_CLIENT_SECRET": "password-weak-secret-1234567890abcdef", // Weak pattern
					"AZURE_TENANT_ID":       "test-tenant-id",
				}
				return setMultipleEnvVars(env)
			},
			expectError:         true,
			expectedErrorSubstr: "azure-ad",
		},
		{
			name: "Keycloak with file read failure",
			setupEnvironment: func(t *testing.T) func() {
				env := map[string]string{
					"KEYCLOAK_CLIENT_ID":          "test-keycloak-client-id",
					"KEYCLOAK_ENABLED":            "true",
					"OAUTH2_KEYCLOAK_SECRET_FILE": "/nonexistent/path/secret", // Nonexistent file
					"KEYCLOAK_BASE_URL":           "https://keycloak.example.com",
				}
				return setMultipleEnvVars(env)
			},
			expectError:         true,
			expectedErrorSubstr: "keycloak",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			cleanup := tt.setupEnvironment(t)
			defer cleanup()

			// Create config and test loadProviders
			config := &AuthConfig{
				Providers: make(map[string]ProviderConfig),
			}

			err := config.loadProviders()

			// Validate results
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

				// Check that expected providers exist
				for _, expectedProvider := range tt.expectedProviders {
					if _, exists := config.Providers[expectedProvider]; !exists {
						t.Errorf("expected provider %s to exist", expectedProvider)
					}
				}

				// Ensure provider count matches expectations
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

// Helper functions for test compatibility

// clearOAuth2Environment clears OAuth2-related environment variables for a provider
func clearOAuth2Environment(t *testing.T, provider string) {
	envVar := strings.ToUpper(strings.ReplaceAll(provider, "-", "_"))
	t.Setenv("OAUTH2_"+envVar+"_CLIENT_SECRET", "")
	t.Setenv("OAUTH2_"+envVar+"_SECRET_FILE", "")
}

// setMultipleEnvVars sets multiple environment variables and returns cleanup function  
// This is compatible with the existing test helper in config_test.go
func setMultipleEnvVars(envVars map[string]string) func() {
	originalVars := make(map[string]string)
	
	// Save original values and set new ones
	for key, value := range envVars {
		originalVars[key] = os.Getenv(key)
		os.Setenv(key, value)
	}
	
	// Return cleanup function
	return func() {
		for key, originalValue := range originalVars {
			os.Setenv(key, originalValue)
		}
	}
}