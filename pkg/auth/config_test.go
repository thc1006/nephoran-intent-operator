package auth

import (
	"os"
	"testing"
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
			config, err := LoadAuthConfig()

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

	config, err := LoadAuthConfig()
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

	config, err := LoadAuthConfig()
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