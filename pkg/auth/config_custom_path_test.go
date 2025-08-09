package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thc1006/nephoran-intent-operator/pkg/security"
)

// TestLoadAuthConfig_CustomFilePath tests the custom file path functionality independently
func TestLoadAuthConfig_CustomFilePath(t *testing.T) {
	// Save original audit logger
	originalAuditLogger := security.GlobalAuditLogger
	defer func() {
		security.GlobalAuditLogger = originalAuditLogger
	}()
	security.GlobalAuditLogger, _ = security.NewAuditLogger("", security.AuditLevelInfo)

	tests := []struct {
		name           string
		configPath     string
		fileContent    string
		envConfigFile  string
		expectError    bool
		expectedError  string
		validateConfig func(t *testing.T, config *AuthConfig)
	}{
		{
			name:       "custom path takes precedence over env var",
			configPath: "custom.json", // Will be replaced with temp file
			fileContent: `{
				"enabled": true,
				"jwt_secret_key": "custom-secret-12345678901234567890123456789012",
				"providers": {}
			}`,
			envConfigFile: "env.json",
			expectError:   false,
			validateConfig: func(t *testing.T, config *AuthConfig) {
				if config.JWTSecretKey != "custom-secret-12345678901234567890123456789012" {
					t.Errorf("Expected custom JWT secret, got: %s", config.JWTSecretKey)
				}
			},
		},
		{
			name:        "empty path uses env var",
			configPath:  "", // Empty - should fall back
			fileContent: `{"enabled": true, "jwt_secret_key": "env-secret-12345678901234567890123456789012", "providers": {}}`,
			expectError: false,
			validateConfig: func(t *testing.T, config *AuthConfig) {
				if config.JWTSecretKey != "env-secret-12345678901234567890123456789012" {
					t.Errorf("Expected env JWT secret, got: %s", config.JWTSecretKey)
				}
			},
		},
		{
			name:          "non-existent custom path returns error",
			configPath:    "/absolutely/non/existent/path.json",
			expectError:   true,
			expectedError: "failed to load config file",
		},
		{
			name:          "invalid JSON returns error",
			configPath:    "invalid.json",
			fileContent:   `{"invalid": json}`, // Invalid JSON
			expectError:   true,
			expectedError: "failed to load config file",
		},
		{
			name:       "empty path and env var works with defaults",
			configPath: "",
			expectError: false,
			validateConfig: func(t *testing.T, config *AuthConfig) {
				// Should work with environment variables only
				if config == nil {
					t.Fatal("config should not be nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env vars
			originalAuthFile := os.Getenv("AUTH_CONFIG_FILE")
			originalAuthEnabled := os.Getenv("AUTH_ENABLED")
			originalJWTSecret := os.Getenv("JWT_SECRET_KEY")
			
			defer func() {
				os.Setenv("AUTH_CONFIG_FILE", originalAuthFile)
				os.Setenv("AUTH_ENABLED", originalAuthEnabled)
				os.Setenv("JWT_SECRET_KEY", originalJWTSecret)
			}()

			var configPath string
			if tt.fileContent != "" {
				// Create temporary file
				tmpFile := filepath.Join(t.TempDir(), "config.json")
				if err := os.WriteFile(tmpFile, []byte(tt.fileContent), 0644); err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				
				if tt.configPath == "" {
					// Test env var fallback
					os.Setenv("AUTH_CONFIG_FILE", tmpFile)
					configPath = ""
				} else {
					configPath = tmpFile
				}
			} else {
				configPath = tt.configPath
			}

			// Set default environment for cases that don't use files
			if tt.fileContent == "" {
				os.Setenv("AUTH_ENABLED", "false")
				os.Setenv("JWT_SECRET_KEY", "fallback-jwt-secret-12345678901234567890123456")
			}

			// Call LoadAuthConfig
			config, err := LoadAuthConfig(configPath)

			// Check error expectations
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				if tt.expectedError != "" && !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing %q, got: %v", tt.expectedError, err)
				}
				return
			}

			// Check success cases
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if config == nil {
				t.Fatal("Expected config but got nil")
			}

			if tt.validateConfig != nil {
				tt.validateConfig(t, config)
			}
		})
	}
}

// TestLoadAuthConfig_PathPrecedence tests specific precedence behavior
func TestLoadAuthConfig_PathPrecedence(t *testing.T) {
	originalAuditLogger := security.GlobalAuditLogger
	defer func() {
		security.GlobalAuditLogger = originalAuditLogger
	}()
	security.GlobalAuditLogger, _ = security.NewAuditLogger("", security.AuditLevelInfo)

	// Create temp directory for test files
	tempDir := t.TempDir()
	
	// Create custom config file
	customFile := filepath.Join(tempDir, "custom.json")
	customConfig := map[string]interface{}{
		"enabled":        true,
		"jwt_secret_key": "custom-jwt-secret-from-custom-file-12345678901234567890",
		"providers":      map[string]interface{}{},
	}
	customData, _ := json.Marshal(customConfig)
	if err := os.WriteFile(customFile, customData, 0644); err != nil {
		t.Fatalf("Failed to create custom config file: %v", err)
	}

	// Create env config file  
	envFile := filepath.Join(tempDir, "env.json")
	envConfig := map[string]interface{}{
		"enabled":        false,
		"jwt_secret_key": "env-jwt-secret-from-env-file-12345678901234567890",
		"providers":      map[string]interface{}{},
	}
	envData, _ := json.Marshal(envConfig)
	if err := os.WriteFile(envFile, envData, 0644); err != nil {
		t.Fatalf("Failed to create env config file: %v", err)
	}

	// Set env var to the env file
	originalAuthFile := os.Getenv("AUTH_CONFIG_FILE")
	os.Setenv("AUTH_CONFIG_FILE", envFile)
	defer os.Setenv("AUTH_CONFIG_FILE", originalAuthFile)

	// Test that custom path takes precedence
	config, err := LoadAuthConfig(customFile)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should use custom file values, not env file values
	if config.JWTSecretKey != "custom-jwt-secret-from-custom-file-12345678901234567890" {
		t.Errorf("Expected JWT secret from custom file, got: %s", config.JWTSecretKey)
	}
	
	if !config.Enabled {
		t.Error("Expected auth enabled from custom file (true), but got disabled")
	}

	// Test fallback to env var when custom path is empty
	config2, err := LoadAuthConfig("")
	if err != nil {
		t.Fatalf("Unexpected error on fallback: %v", err)
	}

	// Should use env file values
	if config2.JWTSecretKey != "env-jwt-secret-from-env-file-12345678901234567890" {
		t.Errorf("Expected JWT secret from env file, got: %s", config2.JWTSecretKey)
	}
	
	if config2.Enabled {
		t.Error("Expected auth disabled from env file (false), but got enabled")
	}
}