package llm

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// TestTLSVerificationBehavior tests the comprehensive TLS verification behavior
func TestTLSVerificationBehavior(t *testing.T) {
	tests := []struct {
		name                   string
		skipTLSVerification    bool
		envValue               string
		expectedPanic          bool
		expectedInsecure       bool
		expectedWarningLog     bool
		expectedSecurityLog    bool
		description            string
	}{
		{
			name:                "Default_Secure_NoEnv",
			skipTLSVerification: false,
			envValue:            "",
			expectedPanic:       false,
			expectedInsecure:    false,
			expectedWarningLog:  false,
			expectedSecurityLog: false,
			description:         "Default behavior: secure TLS verification enforced",
		},
		{
			name:                "Default_Secure_EnvSet",
			skipTLSVerification: false,
			envValue:            "true",
			expectedPanic:       false,
			expectedInsecure:    false,
			expectedWarningLog:  false,
			expectedSecurityLog: false,
			description:         "Secure by default even when env is set but SkipTLS is false",
		},
		{
			name:                "Security_Violation_NoEnv",
			skipTLSVerification: true,
			envValue:            "",
			expectedPanic:       true,
			expectedInsecure:    false,
			expectedWarningLog:  false,
			expectedSecurityLog: true,
			description:         "Security violation: SkipTLS=true but no env permission",
		},
		{
			name:                "Security_Violation_FalseEnv",
			skipTLSVerification: true,
			envValue:            "false",
			expectedPanic:       true,
			expectedInsecure:    false,
			expectedWarningLog:  false,
			expectedSecurityLog: true,
			description:         "Security violation: SkipTLS=true but env=false",
		},
		{
			name:                "Insecure_Mode_Allowed",
			skipTLSVerification: true,
			envValue:            "true",
			expectedPanic:       false,
			expectedInsecure:    true,
			expectedWarningLog:  true,
			expectedSecurityLog: false,
			description:         "Insecure mode: both conditions met, TLS verification skipped with warning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			originalEnv := os.Getenv("ALLOW_INSECURE_CLIENT")
			defer func() {
				if originalEnv == "" {
					os.Unsetenv("ALLOW_INSECURE_CLIENT")
				} else {
					os.Setenv("ALLOW_INSECURE_CLIENT", originalEnv)
				}
			}()

			if tt.envValue == "" {
				os.Unsetenv("ALLOW_INSECURE_CLIENT")
			} else {
				os.Setenv("ALLOW_INSECURE_CLIENT", tt.envValue)
			}

			config := ClientConfig{
				APIKey:              "test-key",
				ModelName:           "test-model",
				MaxTokens:           1000,
				BackendType:         "openai",
				Timeout:             30 * time.Second,
				SkipTLSVerification: tt.skipTLSVerification,
			}

			if tt.expectedPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected panic but none occurred")
					} else if !strings.Contains(fmt.Sprintf("%v", r), "Security violation") {
						t.Errorf("Expected security violation panic, got: %v", r)
					}
				}()
				NewClientWithConfig("https://test.example.com", config)
				return // Should not reach here if panic occurs
			}

			// Create client and verify TLS configuration
			client := NewClientWithConfig("https://test.example.com", config)
			if client == nil {
				t.Fatal("Client creation failed")
			}

			// Verify TLS configuration
			transport, ok := client.httpClient.Transport.(*http.Transport)
			if !ok {
				t.Fatal("Transport is not *http.Transport")
			}

			tlsConfig := transport.TLSClientConfig
			if tlsConfig == nil {
				t.Fatal("TLS config is nil")
			}

			if tt.expectedInsecure {
				if !tlsConfig.InsecureSkipVerify {
					t.Error("Expected InsecureSkipVerify=true but got false")
				}
			} else {
				if tlsConfig.InsecureSkipVerify {
					t.Error("Expected InsecureSkipVerify=false but got true")
				}
			}

			// Verify minimum TLS version
			if tlsConfig.MinVersion != tls.VersionTLS12 {
				t.Errorf("Expected MinVersion=TLS1.2, got %d", tlsConfig.MinVersion)
			}

			// Verify cipher suites are configured
			if len(tlsConfig.CipherSuites) == 0 {
				t.Error("Expected cipher suites to be configured")
			}

			// Cleanup
			client.Shutdown()
		})
	}
}

// TestEnvironmentVariableValidation tests environment variable validation
func TestEnvironmentVariableValidation(t *testing.T) {
	tests := []struct {
		name       string
		envValue   string
		expected   bool
		description string
	}{
		{
			name:        "Empty_Value",
			envValue:    "",
			expected:    false,
			description: "Empty environment variable should return false",
		},
		{
			name:        "True_Value",
			envValue:    "true",
			expected:    true,
			description: "Exact 'true' value should return true",
		},
		{
			name:        "False_Value",
			envValue:    "false",
			expected:    false,
			description: "False value should return false",
		},
		{
			name:        "Case_Sensitive_TRUE",
			envValue:    "TRUE",
			expected:    false,
			description: "Case sensitivity: TRUE should return false",
		},
		{
			name:        "Case_Sensitive_True",
			envValue:    "True",
			expected:    false,
			description: "Case sensitivity: True should return false",
		},
		{
			name:        "Whitespace_Leading",
			envValue:    " true",
			expected:    false,
			description: "Leading whitespace should not be allowed",
		},
		{
			name:        "Whitespace_Trailing",
			envValue:    "true ",
			expected:    false,
			description: "Trailing whitespace should not be allowed",
		},
		{
			name:        "Whitespace_Both",
			envValue:    " true ",
			expected:    false,
			description: "Surrounding whitespace should not be allowed",
		},
		{
			name:        "Numeric_1",
			envValue:    "1",
			expected:    false,
			description: "Numeric 1 should not be accepted",
		},
		{
			name:        "Yes_Value",
			envValue:    "yes",
			expected:    false,
			description: "Yes value should not be accepted",
		},
		{
			name:        "Random_String",
			envValue:    "allow_insecure",
			expected:    false,
			description: "Random string should not be accepted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			originalEnv := os.Getenv("ALLOW_INSECURE_CLIENT")
			defer func() {
				if originalEnv == "" {
					os.Unsetenv("ALLOW_INSECURE_CLIENT")
				} else {
					os.Setenv("ALLOW_INSECURE_CLIENT", originalEnv)
				}
			}()

			if tt.envValue == "" {
				os.Unsetenv("ALLOW_INSECURE_CLIENT")
			} else {
				os.Setenv("ALLOW_INSECURE_CLIENT", tt.envValue)
			}

			result := allowInsecureClient()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for env value '%s'", tt.expected, result, tt.envValue)
			}
		})
	}
}

// TestTLSIntegrationWithMockServer tests TLS behavior with actual connections
func TestTLSIntegrationWithMockServer(t *testing.T) {
	tests := []struct {
		name                string
		serverTLS           bool
		clientSkipTLS       bool
		envValue            string
		expectConnectionOK  bool
		expectTLSError      bool
		description         string
	}{
		{
			name:               "HTTPS_Server_Secure_Client",
			serverTLS:          true,
			clientSkipTLS:      false,
			envValue:           "",
			expectConnectionOK: true,
			expectTLSError:     false,
			description:        "HTTPS server with secure client should work",
		},
		{
			name:               "HTTPS_Server_Insecure_Client",
			serverTLS:          true,
			clientSkipTLS:      true,
			envValue:           "true",
			expectConnectionOK: true,
			expectTLSError:     false,
			description:        "HTTPS server with insecure client should work",
		},
		{
			name:               "HTTP_Server_Secure_Client",
			serverTLS:          false,
			clientSkipTLS:      false,
			envValue:           "",
			expectConnectionOK: false,
			expectTLSError:     true,
			description:        "HTTP server with secure client should fail with TLS error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			originalEnv := os.Getenv("ALLOW_INSECURE_CLIENT")
			defer func() {
				if originalEnv == "" {
					os.Unsetenv("ALLOW_INSECURE_CLIENT")
				} else {
					os.Setenv("ALLOW_INSECURE_CLIENT", originalEnv)
				}
			}()

			os.Setenv("ALLOW_INSECURE_CLIENT", tt.envValue)

			// Create test server
			var server *httptest.Server
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"choices":[{"message":{"content":"{\"type\":\"NetworkFunctionDeployment\",\"name\":\"test-nf\",\"namespace\":\"default\",\"spec\":{\"replicas\":1,\"image\":\"test:latest\"}}"}}]}`))
			})

			if tt.serverTLS {
				server = httptest.NewTLSServer(handler)
			} else {
				server = httptest.NewServer(handler)
			}
			defer server.Close()

			// Create client
			config := ClientConfig{
				APIKey:              "test-key",
				ModelName:           "test-model",
				MaxTokens:           1000,
				BackendType:         "openai",
				Timeout:             5 * time.Second,
				SkipTLSVerification: tt.clientSkipTLS,
			}

			var client *Client
			if tt.clientSkipTLS && tt.envValue != "true" {
				// Should panic - test panic recovery
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic for security violation")
					}
				}()
				client = NewClientWithConfig(server.URL, config)
				return
			}

			client = NewClientWithConfig(server.URL, config)
			defer client.Shutdown()

			// Test connection
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			_, err := client.ProcessIntent(ctx, "deploy test network function")

			if tt.expectConnectionOK {
				if err != nil {
					t.Errorf("Expected successful connection, got error: %v", err)
				}
			} else {
				if err == nil {
					t.Error("Expected connection to fail, but it succeeded")
				}
				if tt.expectTLSError && !strings.Contains(err.Error(), "tls") && !strings.Contains(err.Error(), "certificate") {
					t.Errorf("Expected TLS-related error, got: %v", err)
				}
			}
		})
	}
}

// TestTLSCertificateValidation tests certificate validation behavior
func TestTLSCertificateValidation(t *testing.T) {
	// Create a server with self-signed certificate
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices":[{"message":{"content":"{\"type\":\"NetworkFunctionDeployment\",\"name\":\"test-nf\",\"namespace\":\"default\",\"spec\":{\"replicas\":1,\"image\":\"test:latest\"}}"}}]}`))
	}))
	defer server.Close()

	tests := []struct {
		name              string
		skipTLS           bool
		envValue          string
		expectSuccess     bool
		expectCertError   bool
		description       string
	}{
		{
			name:            "Self_Signed_Cert_Secure_Client",
			skipTLS:         false,
			envValue:        "",
			expectSuccess:   false,
			expectCertError: true,
			description:     "Self-signed cert should fail with secure client",
		},
		{
			name:            "Self_Signed_Cert_Insecure_Client",
			skipTLS:         true,
			envValue:        "true",
			expectSuccess:   true,
			expectCertError: false,
			description:     "Self-signed cert should work with insecure client",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			originalEnv := os.Getenv("ALLOW_INSECURE_CLIENT")
			defer func() {
				if originalEnv == "" {
					os.Unsetenv("ALLOW_INSECURE_CLIENT")
				} else {
					os.Setenv("ALLOW_INSECURE_CLIENT", originalEnv)
				}
			}()

			os.Setenv("ALLOW_INSECURE_CLIENT", tt.envValue)

			config := ClientConfig{
				APIKey:              "test-key",
				ModelName:           "test-model",
				MaxTokens:           1000,
				BackendType:         "openai",
				Timeout:             5 * time.Second,
				SkipTLSVerification: tt.skipTLS,
			}

			client := NewClientWithConfig(server.URL, config)
			defer client.Shutdown()

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			_, err := client.ProcessIntent(ctx, "deploy test network function")

			if tt.expectSuccess {
				if err != nil {
					t.Errorf("Expected success, got error: %v", err)
				}
			} else {
				if err == nil {
					t.Error("Expected failure but got success")
				}
				if tt.expectCertError {
					errMsg := strings.ToLower(err.Error())
					if !strings.Contains(errMsg, "certificate") && !strings.Contains(errMsg, "tls") && !strings.Contains(errMsg, "x509") {
						t.Errorf("Expected certificate-related error, got: %v", err)
					}
				}
			}
		})
	}
}

// TestTLSSecurityConfiguration tests TLS security hardening
func TestTLSSecurityConfiguration(t *testing.T) {
	originalEnv := os.Getenv("ALLOW_INSECURE_CLIENT")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("ALLOW_INSECURE_CLIENT")
		} else {
			os.Setenv("ALLOW_INSECURE_CLIENT", originalEnv)
		}
	}()

	os.Unsetenv("ALLOW_INSECURE_CLIENT")

	config := ClientConfig{
		APIKey:              "test-key",
		ModelName:           "test-model",
		MaxTokens:           1000,
		BackendType:         "openai",
		Timeout:             30 * time.Second,
		SkipTLSVerification: false,
	}

	client := NewClientWithConfig("https://test.example.com", config)
	defer client.Shutdown()

	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Transport is not *http.Transport")
	}

	tlsConfig := transport.TLSClientConfig
	if tlsConfig == nil {
		t.Fatal("TLS config is nil")
	}

	// Test minimum TLS version
	if tlsConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("Expected MinVersion=TLS1.2 (%d), got %d", tls.VersionTLS12, tlsConfig.MinVersion)
	}

	// Test cipher suites are configured
	expectedCiphers := []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	}

	if len(tlsConfig.CipherSuites) != len(expectedCiphers) {
		t.Errorf("Expected %d cipher suites, got %d", len(expectedCiphers), len(tlsConfig.CipherSuites))
	}

	for i, expected := range expectedCiphers {
		if i >= len(tlsConfig.CipherSuites) || tlsConfig.CipherSuites[i] != expected {
			t.Errorf("Cipher suite mismatch at index %d: expected %d, got %d", i, expected, tlsConfig.CipherSuites[i])
		}
	}

	// Test server cipher preference
	if !tlsConfig.PreferServerCipherSuites {
		t.Error("Expected PreferServerCipherSuites=true")
	}

	// Test InsecureSkipVerify is false by default
	if tlsConfig.InsecureSkipVerify {
		t.Error("Expected InsecureSkipVerify=false by default")
	}
}

// TestTLSErrorHandling tests error handling in TLS scenarios
func TestTLSErrorHandling(t *testing.T) {
	tests := []struct {
		name            string
		setupFunc       func() (*httptest.Server, func())
		clientSkipTLS   bool
		envValue        string
		expectedErrors  []string
		description     string
	}{
		{
			name: "Connection_Refused",
			setupFunc: func() (*httptest.Server, func()) {
				// Return a server that's immediately closed
				server := httptest.NewTLSServer(nil)
				server.Close()
				return server, func() {}
			},
			clientSkipTLS:  true,
			envValue:       "true",
			expectedErrors: []string{"connection refused", "connect"},
			description:    "Connection refused error should be handled gracefully",
		},
		{
			name: "Invalid_Certificate",
			setupFunc: func() (*httptest.Server, func()) {
				// Create server with invalid certificate
				server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
				return server, func() { server.Close() }
			},
			clientSkipTLS:  false,
			envValue:       "",
			expectedErrors: []string{"certificate", "x509", "tls"},
			description:    "Invalid certificate should produce appropriate error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			originalEnv := os.Getenv("ALLOW_INSECURE_CLIENT")
			defer func() {
				if originalEnv == "" {
					os.Unsetenv("ALLOW_INSECURE_CLIENT")
				} else {
					os.Setenv("ALLOW_INSECURE_CLIENT", originalEnv)
				}
			}()

			os.Setenv("ALLOW_INSECURE_CLIENT", tt.envValue)

			server, cleanup := tt.setupFunc()
			defer cleanup()

			config := ClientConfig{
				APIKey:              "test-key",
				ModelName:           "test-model",
				MaxTokens:           1000,
				BackendType:         "openai",
				Timeout:             2 * time.Second,
				SkipTLSVerification: tt.clientSkipTLS,
			}

			client := NewClientWithConfig(server.URL, config)
			defer client.Shutdown()

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			_, err := client.ProcessIntent(ctx, "test intent")

			if err == nil {
				t.Error("Expected error but got none")
				return
			}

			errorMsg := strings.ToLower(err.Error())
			foundExpectedError := false
			for _, expectedError := range tt.expectedErrors {
				if strings.Contains(errorMsg, strings.ToLower(expectedError)) {
					foundExpectedError = true
					break
				}
			}

			if !foundExpectedError {
				t.Errorf("Expected error containing one of %v, got: %v", tt.expectedErrors, err)
			}
		})
	}
}

// TestTLSPerformanceBenchmark benchmarks TLS security checks
func TestTLSPerformanceBenchmark(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance benchmark in short mode")
	}

	// Setup
	originalEnv := os.Getenv("ALLOW_INSECURE_CLIENT")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("ALLOW_INSECURE_CLIENT")
		} else {
			os.Setenv("ALLOW_INSECURE_CLIENT", originalEnv)
		}
	}()

	os.Unsetenv("ALLOW_INSECURE_CLIENT")

	config := ClientConfig{
		APIKey:              "test-key",
		ModelName:           "test-model",
		MaxTokens:           1000,
		BackendType:         "openai",
		Timeout:             30 * time.Second,
		SkipTLSVerification: false,
	}

	// Benchmark client creation (includes TLS setup)
	iterations := 1000
	start := time.Now()

	for i := 0; i < iterations; i++ {
		client := NewClientWithConfig("https://test.example.com", config)
		if client == nil {
			t.Fatal("Client creation failed")
		}
		client.Shutdown()
	}

	duration := time.Since(start)
	avgDuration := duration / time.Duration(iterations)

	t.Logf("TLS client creation benchmark:")
	t.Logf("  Total time: %v", duration)
	t.Logf("  Average per creation: %v", avgDuration)
	t.Logf("  Operations per second: %.2f", float64(iterations)/duration.Seconds())

	// Performance threshold (adjust based on requirements)
	maxAvgDuration := 5 * time.Millisecond
	if avgDuration > maxAvgDuration {
		t.Errorf("TLS client creation too slow: %v > %v", avgDuration, maxAvgDuration)
	}
}

// BenchmarkAllowInsecureClient benchmarks the environment variable check
func BenchmarkAllowInsecureClient(b *testing.B) {
	// Setup different environment scenarios
	scenarios := []struct {
		name     string
		envValue string
	}{
		{"NoEnv", ""},
		{"EnvTrue", "true"},
		{"EnvFalse", "false"},
		{"EnvInvalid", "invalid"},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			// Setup environment
			originalEnv := os.Getenv("ALLOW_INSECURE_CLIENT")
			defer func() {
				if originalEnv == "" {
					os.Unsetenv("ALLOW_INSECURE_CLIENT")
				} else {
					os.Setenv("ALLOW_INSECURE_CLIENT", originalEnv)
				}
			}()

			if scenario.envValue == "" {
				os.Unsetenv("ALLOW_INSECURE_CLIENT")
			} else {
				os.Setenv("ALLOW_INSECURE_CLIENT", scenario.envValue)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = allowInsecureClient()
			}
		})
	}
}

// BenchmarkTLSClientCreation benchmarks client creation with different TLS configurations
func BenchmarkTLSClientCreation(b *testing.B) {
	scenarios := []struct {
		name      string
		skipTLS   bool
		envValue  string
	}{
		{"Secure", false, ""},
		{"Insecure", true, "true"},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			// Setup environment
			originalEnv := os.Getenv("ALLOW_INSECURE_CLIENT")
			defer func() {
				if originalEnv == "" {
					os.Unsetenv("ALLOW_INSECURE_CLIENT")
				} else {
					os.Setenv("ALLOW_INSECURE_CLIENT", originalEnv)
				}
			}()

			if scenario.envValue == "" {
				os.Unsetenv("ALLOW_INSECURE_CLIENT")
			} else {
				os.Setenv("ALLOW_INSECURE_CLIENT", scenario.envValue)
			}

			config := ClientConfig{
				APIKey:              "test-key",
				ModelName:           "test-model",
				MaxTokens:           1000,
				BackendType:         "openai",
				Timeout:             30 * time.Second,
				SkipTLSVerification: scenario.skipTLS,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				client := NewClientWithConfig("https://test.example.com", config)
				if client != nil {
					client.Shutdown()
				}
			}
		})
	}
}

// TestTLSEdgeCases tests edge cases in TLS configuration
func TestTLSEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func()
		cleanupFunc func()
		testFunc    func(t *testing.T)
		description string
	}{
		{
			name: "Environment_Variable_Race_Condition",
			setupFunc: func() {
				os.Setenv("ALLOW_INSECURE_CLIENT", "true")
			},
			cleanupFunc: func() {
				os.Unsetenv("ALLOW_INSECURE_CLIENT")
			},
			testFunc: func(t *testing.T) {
				// Test concurrent access to environment variable
				numGoroutines := 10
				results := make(chan bool, numGoroutines)

				for i := 0; i < numGoroutines; i++ {
					go func() {
						result := allowInsecureClient()
						results <- result
					}()
				}

				// Collect results
				for i := 0; i < numGoroutines; i++ {
					result := <-results
					if !result {
						t.Error("Expected true from allowInsecureClient() in concurrent access")
					}
				}
			},
			description: "Test race condition safety in environment variable access",
		},
		{
			name: "Multiple_Client_Creation",
			setupFunc: func() {
				os.Unsetenv("ALLOW_INSECURE_CLIENT")
			},
			cleanupFunc: func() {
				os.Unsetenv("ALLOW_INSECURE_CLIENT")
			},
			testFunc: func(t *testing.T) {
				config := ClientConfig{
					APIKey:              "test-key",
					ModelName:           "test-model",
					MaxTokens:           1000,
					BackendType:         "openai",
					Timeout:             30 * time.Second,
					SkipTLSVerification: false,
				}

				// Create multiple clients to ensure no shared state issues
				clients := make([]*Client, 5)
				for i := 0; i < 5; i++ {
					clients[i] = NewClientWithConfig("https://test.example.com", config)
					if clients[i] == nil {
						t.Fatalf("Client %d creation failed", i)
					}
				}

				// Verify each client has independent TLS configuration
				for i, client := range clients {
					transport, ok := client.httpClient.Transport.(*http.Transport)
					if !ok {
						t.Errorf("Client %d transport is not *http.Transport", i)
						continue
					}

					tlsConfig := transport.TLSClientConfig
					if tlsConfig == nil {
						t.Errorf("Client %d TLS config is nil", i)
						continue
					}

					if tlsConfig.InsecureSkipVerify {
						t.Errorf("Client %d has InsecureSkipVerify=true when it should be false", i)
					}
				}

				// Cleanup
				for _, client := range clients {
					if client != nil {
						client.Shutdown()
					}
				}
			},
			description: "Test multiple client creation doesn't share TLS state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			originalEnv := os.Getenv("ALLOW_INSECURE_CLIENT")
			defer func() {
				if originalEnv == "" {
					os.Unsetenv("ALLOW_INSECURE_CLIENT")
				} else {
					os.Setenv("ALLOW_INSECURE_CLIENT", originalEnv)
				}
			}()

			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			if tt.cleanupFunc != nil {
				defer tt.cleanupFunc()
			}

			tt.testFunc(t)
		})
	}
}

// TestTLSConnectionTimeout tests timeout behavior with TLS connections
func TestTLSConnectionTimeout(t *testing.T) {
	// Create a server that accepts connections but doesn't respond
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverURL := fmt.Sprintf("https://%s", listener.Addr().String())

	// Setup environment for insecure client
	originalEnv := os.Getenv("ALLOW_INSECURE_CLIENT")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("ALLOW_INSECURE_CLIENT")
		} else {
			os.Setenv("ALLOW_INSECURE_CLIENT", originalEnv)
		}
	}()

	os.Setenv("ALLOW_INSECURE_CLIENT", "true")

	config := ClientConfig{
		APIKey:              "test-key",
		ModelName:           "test-model",
		MaxTokens:           1000,
		BackendType:         "openai",
		Timeout:             1 * time.Second, // Short timeout
		SkipTLSVerification: true,
	}

	client := NewClientWithConfig(serverURL, config)
	defer client.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	_, err = client.ProcessIntent(ctx, "test intent")
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Expected timeout error but got none")
	}

	// Verify timeout occurred within reasonable bounds
	if elapsed > 3*time.Second {
		t.Errorf("Timeout took too long: %v", elapsed)
	}

	if !strings.Contains(strings.ToLower(err.Error()), "timeout") &&
		!strings.Contains(strings.ToLower(err.Error()), "context deadline exceeded") {
		t.Errorf("Expected timeout-related error, got: %v", err)
	}
}

// TestTLSWithDifferentCertificateTypes tests behavior with different certificate scenarios
func TestTLSWithDifferentCertificateTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping certificate tests in short mode")
	}

	tests := []struct {
		name                string
		certSetupFunc       func() (*httptest.Server, func())
		clientSkipTLS       bool
		envValue            string
		expectSuccess       bool
		expectedErrorTypes  []string
		description         string
	}{
		{
			name: "Valid_Certificate_Secure_Client",
			certSetupFunc: func() (*httptest.Server, func()) {
				server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"choices":[{"message":{"content":"{\"type\":\"NetworkFunctionDeployment\",\"name\":\"test-nf\",\"namespace\":\"default\",\"spec\":{\"replicas\":1,\"image\":\"test:latest\"}}"}}]}`))
				}))

				// Add the server's certificate to a custom cert pool
				transport := &http.Transport{
					TLSClientConfig: &tls.Config{
						RootCAs: x509.NewCertPool(),
					},
				}
				transport.TLSClientConfig.RootCAs.AddCert(server.Certificate())

				return server, func() { server.Close() }
			},
			clientSkipTLS:      false,
			envValue:           "",
			expectSuccess:      false, // Will still fail due to hostname mismatch
			expectedErrorTypes: []string{"certificate", "x509"},
			description:        "Even with valid cert, hostname mismatch should cause failure",
		},
		{
			name: "Valid_Certificate_Insecure_Client",
			certSetupFunc: func() (*httptest.Server, func()) {
				server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"choices":[{"message":{"content":"{\"type\":\"NetworkFunctionDeployment\",\"name\":\"test-nf\",\"namespace\":\"default\",\"spec\":{\"replicas\":1,\"image\":\"test:latest\"}}"}}]}`))
				}))
				return server, func() { server.Close() }
			},
			clientSkipTLS:      true,
			envValue:           "true",
			expectSuccess:      true,
			expectedErrorTypes: []string{},
			description:        "Insecure client should accept any certificate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			originalEnv := os.Getenv("ALLOW_INSECURE_CLIENT")
			defer func() {
				if originalEnv == "" {
					os.Unsetenv("ALLOW_INSECURE_CLIENT")
				} else {
					os.Setenv("ALLOW_INSECURE_CLIENT", originalEnv)
				}
			}()

			os.Setenv("ALLOW_INSECURE_CLIENT", tt.envValue)

			server, cleanup := tt.certSetupFunc()
			defer cleanup()

			config := ClientConfig{
				APIKey:              "test-key",
				ModelName:           "test-model",
				MaxTokens:           1000,
				BackendType:         "openai",
				Timeout:             5 * time.Second,
				SkipTLSVerification: tt.clientSkipTLS,
			}

			client := NewClientWithConfig(server.URL, config)
			defer client.Shutdown()

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			_, err := client.ProcessIntent(ctx, "test intent")

			if tt.expectSuccess {
				if err != nil {
					t.Errorf("Expected success, got error: %v", err)
				}
			} else {
				if err == nil {
					t.Error("Expected failure but got success")
				}

				if len(tt.expectedErrorTypes) > 0 {
					errorMsg := strings.ToLower(err.Error())
					foundExpectedError := false
					for _, expectedType := range tt.expectedErrorTypes {
						if strings.Contains(errorMsg, strings.ToLower(expectedType)) {
							foundExpectedError = true
							break
						}
					}
					if !foundExpectedError {
						t.Errorf("Expected error containing one of %v, got: %v", tt.expectedErrorTypes, err)
					}
				}
			}
		})
	}
}