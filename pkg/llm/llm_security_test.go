package llm

import (
	"crypto/tls"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestTLSVerificationSecurity tests the TLS verification security controls
func TestTLSVerificationSecurity(t *testing.T) {
	tests := []struct {
		name                string
		config              ClientConfig
		envValue            string
		expectPanic         bool
		expectInsecureSkip  bool
		description         string
	}{
		{
			name: "secure_by_default",
			config: ClientConfig{
				APIKey:              "test-key",
				ModelName:           "test-model",
				MaxTokens:           100,
				BackendType:         "openai",
				Timeout:             30 * time.Second,
				SkipTLSVerification: false,
			},
			envValue:           "",
			expectPanic:        false,
			expectInsecureSkip: false,
			description:        "Default configuration should use secure TLS",
		},
		{
			name: "attempt_skip_without_env",
			config: ClientConfig{
				APIKey:              "test-key",
				ModelName:           "test-model",
				MaxTokens:           100,
				BackendType:         "openai",
				Timeout:             30 * time.Second,
				SkipTLSVerification: true,
			},
			envValue:           "",
			expectPanic:        true,
			expectInsecureSkip: false,
			description:        "Should panic when trying to skip TLS without environment permission",
		},
		{
			name: "attempt_skip_with_wrong_env",
			config: ClientConfig{
				APIKey:              "test-key",
				ModelName:           "test-model",
				MaxTokens:           100,
				BackendType:         "openai",
				Timeout:             30 * time.Second,
				SkipTLSVerification: true,
			},
			envValue:           "false",
			expectPanic:        true,
			expectInsecureSkip: false,
			description:        "Should panic when environment variable is not exactly 'true'",
		},
		{
			name: "allow_skip_with_both_conditions",
			config: ClientConfig{
				APIKey:              "test-key",
				ModelName:           "test-model",
				MaxTokens:           100,
				BackendType:         "openai",
				Timeout:             30 * time.Second,
				SkipTLSVerification: true,
			},
			envValue:           "true",
			expectPanic:        false,
			expectInsecureSkip: true,
			description:        "Should allow insecure TLS only when both conditions are met",
		},
		{
			name: "env_set_but_config_false",
			config: ClientConfig{
				APIKey:              "test-key",
				ModelName:           "test-model",
				MaxTokens:           100,
				BackendType:         "openai",
				Timeout:             30 * time.Second,
				SkipTLSVerification: false,
			},
			envValue:           "true",
			expectPanic:        false,
			expectInsecureSkip: false,
			description:        "Should remain secure when config doesn't request insecure mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			originalEnv := os.Getenv("ALLOW_INSECURE_CLIENT")
			if tt.envValue != "" {
				os.Setenv("ALLOW_INSECURE_CLIENT", tt.envValue)
			} else {
				os.Unsetenv("ALLOW_INSECURE_CLIENT")
			}
			defer func() {
				if originalEnv != "" {
					os.Setenv("ALLOW_INSECURE_CLIENT", originalEnv)
				} else {
					os.Unsetenv("ALLOW_INSECURE_CLIENT")
				}
			}()

			// Test with panic recovery
			if tt.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Test %s: expected panic but didn't get one", tt.name)
					}
				}()
			}

			// Create client
			client := NewClientWithConfig("https://test.example.com", tt.config)

			// If we expect panic, we shouldn't reach here
			if tt.expectPanic {
				t.Errorf("Test %s: expected panic but client was created", tt.name)
				return
			}

			// Verify TLS configuration
			transport, ok := client.httpClient.Transport.(*http.Transport)
			if !ok {
				t.Fatalf("Test %s: unexpected transport type", tt.name)
			}

			if transport.TLSClientConfig == nil {
				t.Fatalf("Test %s: TLS config is nil", tt.name)
			}

			// Check InsecureSkipVerify setting
			if transport.TLSClientConfig.InsecureSkipVerify != tt.expectInsecureSkip {
				t.Errorf("Test %s: InsecureSkipVerify = %v, want %v",
					tt.name,
					transport.TLSClientConfig.InsecureSkipVerify,
					tt.expectInsecureSkip)
			}

			// Verify minimum TLS version
			if transport.TLSClientConfig.MinVersion != tls.VersionTLS12 {
				t.Errorf("Test %s: MinVersion = %v, want %v",
					tt.name,
					transport.TLSClientConfig.MinVersion,
					tls.VersionTLS12)
			}

			// Verify cipher suites are set
			if len(transport.TLSClientConfig.CipherSuites) == 0 {
				t.Errorf("Test %s: no cipher suites configured", tt.name)
			}
		})
	}
}

// TestAllowInsecureClient tests the environment variable check
func TestAllowInsecureClient(t *testing.T) {
	tests := []struct {
		envValue string
		expected bool
	}{
		{"", false},
		{"false", false},
		{"False", false},
		{"TRUE", false},
		{"True", false},
		{"1", false},
		{"yes", false},
		{"true", true},
	}

	for _, tt := range tests {
		t.Run(tt.envValue, func(t *testing.T) {
			// Save and restore original env
			original := os.Getenv("ALLOW_INSECURE_CLIENT")
			defer func() {
				if original != "" {
					os.Setenv("ALLOW_INSECURE_CLIENT", original)
				} else {
					os.Unsetenv("ALLOW_INSECURE_CLIENT")
				}
			}()

			// Set test value
			if tt.envValue != "" {
				os.Setenv("ALLOW_INSECURE_CLIENT", tt.envValue)
			} else {
				os.Unsetenv("ALLOW_INSECURE_CLIENT")
			}

			// Test
			result := allowInsecureClient()
			if result != tt.expected {
				t.Errorf("allowInsecureClient() with env=%q = %v, want %v",
					tt.envValue, result, tt.expected)
			}
		})
	}
}

// TestTLSConfigurationHardening tests the security hardening of TLS configuration
func TestTLSConfigurationHardening(t *testing.T) {
	// Test with secure configuration
	config := ClientConfig{
		APIKey:              "test-key",
		ModelName:           "test-model",
		MaxTokens:           100,
		BackendType:         "openai",
		Timeout:             30 * time.Second,
		SkipTLSVerification: false,
	}

	client := NewClientWithConfig("https://test.example.com", config)
	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("unexpected transport type")
	}

	tlsConfig := transport.TLSClientConfig

	// Verify secure cipher suites
	expectedCiphers := []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	}

	if len(tlsConfig.CipherSuites) != len(expectedCiphers) {
		t.Errorf("Expected %d cipher suites, got %d", len(expectedCiphers), len(tlsConfig.CipherSuites))
	}

	// Verify PreferServerCipherSuites is enabled
	if !tlsConfig.PreferServerCipherSuites {
		t.Error("PreferServerCipherSuites should be true for security")
	}

	// Verify minimum TLS version
	if tlsConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("MinVersion = %v, want %v", tlsConfig.MinVersion, tls.VersionTLS12)
	}
}

// TestBackwardCompatibility ensures the default NewClient function still works
func TestBackwardCompatibility(t *testing.T) {
	// This should work without any issues
	client := NewClient("https://test.example.com")
	
	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	// Verify it uses secure defaults
	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("unexpected transport type")
	}

	if transport.TLSClientConfig.InsecureSkipVerify {
		t.Error("Default client should not skip TLS verification")
	}
}