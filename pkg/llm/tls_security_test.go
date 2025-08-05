package llm

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// TestTLSSecurityCore tests core TLS security functionality without dependencies
func TestTLSSecurityCore(t *testing.T) {
	// Test the allowInsecureClient function directly
	t.Run("AllowInsecureClient_Function", func(t *testing.T) {
		// Save original environment
		originalEnv := os.Getenv("ALLOW_INSECURE_CLIENT")
		defer func() {
			if originalEnv == "" {
				os.Unsetenv("ALLOW_INSECURE_CLIENT")
			} else {
				os.Setenv("ALLOW_INSECURE_CLIENT", originalEnv)
			}
		}()

		// Test cases
		testCases := []struct {
			envValue string
			expected bool
		}{
			{"", false},
			{"true", true},
			{"false", false},
			{"TRUE", false},
			{"True", false},
			{" true", false},
			{"true ", false},
			{"1", false},
			{"yes", false},
		}

		for _, tc := range testCases {
			if tc.envValue == "" {
				os.Unsetenv("ALLOW_INSECURE_CLIENT")
			} else {
				os.Setenv("ALLOW_INSECURE_CLIENT", tc.envValue)
			}

			result := allowInsecureClient()
			if result != tc.expected {
				t.Errorf("allowInsecureClient() with env='%s': expected %v, got %v", tc.envValue, tc.expected, result)
			}
		}
	})

	// Test NewClient creation with TLS security
	t.Run("NewClient_TLS_Security", func(t *testing.T) {
		originalEnv := os.Getenv("ALLOW_INSECURE_CLIENT")
		defer func() {
			if originalEnv == "" {
				os.Unsetenv("ALLOW_INSECURE_CLIENT")
			} else {
				os.Setenv("ALLOW_INSECURE_CLIENT", originalEnv)
			}
		}()

		// Test secure client creation
		os.Unsetenv("ALLOW_INSECURE_CLIENT")
		client := NewClient("https://test.example.com")
		if client == nil {
			t.Fatal("Failed to create secure client")
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

		if tlsConfig.InsecureSkipVerify {
			t.Error("Expected secure TLS configuration, but InsecureSkipVerify is true")
		}

		if tlsConfig.MinVersion != tls.VersionTLS12 {
			t.Errorf("Expected MinVersion=TLS1.2, got %d", tlsConfig.MinVersion)
		}

		client.Shutdown()
	})

	// Test security violation panic
	t.Run("Security_Violation_Panic", func(t *testing.T) {
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
			SkipTLSVerification: true, // This should trigger panic
		}

		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for security violation, but none occurred")
			} else if !strings.Contains(fmt.Sprintf("%v", r), "Security violation") {
				t.Errorf("Expected security violation panic, got: %v", r)
			}
		}()

		NewClientWithConfig("https://test.example.com", config)
		t.Error("Should not reach this point - expected panic")
	})

	// Test allowed insecure mode
	t.Run("Allowed_Insecure_Mode", func(t *testing.T) {
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
			Timeout:             30 * time.Second,
			SkipTLSVerification: true,
		}

		client := NewClientWithConfig("https://test.example.com", config)
		if client == nil {
			t.Fatal("Failed to create insecure client when allowed")
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

		if !tlsConfig.InsecureSkipVerify {
			t.Error("Expected InsecureSkipVerify=true in insecure mode")
		}

		client.Shutdown()
	})
}

// TestTLSWithMockServer tests TLS behavior with mock HTTP server
func TestTLSWithMockServer(t *testing.T) {
	// Create a test server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Return a valid LLM response
		response := `{"choices":[{"message":{"content":"{\"type\":\"NetworkFunctionDeployment\",\"name\":\"test-nf\",\"namespace\":\"default\",\"spec\":{\"replicas\":1,\"image\":\"test:latest\"}}"}}]}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	originalEnv := os.Getenv("ALLOW_INSECURE_CLIENT")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("ALLOW_INSECURE_CLIENT")
		} else {
			os.Setenv("ALLOW_INSECURE_CLIENT", originalEnv)
		}
	}()

	// Test with insecure client (should work with self-signed cert)
	t.Run("Insecure_Client_Self_Signed_Cert", func(t *testing.T) {
		os.Setenv("ALLOW_INSECURE_CLIENT", "true")

		config := ClientConfig{
			APIKey:              "test-key",
			ModelName:           "test-model",
			MaxTokens:           1000,
			BackendType:         "openai",
			Timeout:             5 * time.Second,
			SkipTLSVerification: true,
		}

		client := NewClientWithConfig(server.URL, config)
		defer client.Shutdown()

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		_, err := client.ProcessIntent(ctx, "deploy test network function")
		if err != nil {
			t.Errorf("Expected insecure client to work with self-signed cert, got error: %v", err)
		}
	})

	// Test with secure client (should fail with self-signed cert)
	t.Run("Secure_Client_Self_Signed_Cert", func(t *testing.T) {
		os.Unsetenv("ALLOW_INSECURE_CLIENT")

		config := ClientConfig{
			APIKey:              "test-key",
			ModelName:           "test-model",
			MaxTokens:           1000,
			BackendType:         "openai",
			Timeout:             5 * time.Second,
			SkipTLSVerification: false,
		}

		client := NewClientWithConfig(server.URL, config)
		defer client.Shutdown()

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		_, err := client.ProcessIntent(ctx, "deploy test network function")
		if err == nil {
			t.Error("Expected secure client to fail with self-signed cert, but it succeeded")
		}

		// Check if it's a certificate-related error
		errMsg := strings.ToLower(err.Error())
		if !strings.Contains(errMsg, "certificate") && !strings.Contains(errMsg, "tls") && !strings.Contains(errMsg, "x509") {
			t.Errorf("Expected certificate-related error, got: %v", err)
		}
	})
}

// TestTLSConfiguration tests TLS configuration details
func TestTLSConfiguration(t *testing.T) {
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

	// Test cipher suites
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
		if i >= len(tlsConfig.CipherSuites) {
			t.Errorf("Missing cipher suite at index %d", i)
			break
		}
		if tlsConfig.CipherSuites[i] != expected {
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

// BenchmarkTLSSecurity benchmarks TLS security operations
func BenchmarkTLSSecurity(b *testing.B) {
	originalEnv := os.Getenv("ALLOW_INSECURE_CLIENT")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("ALLOW_INSECURE_CLIENT")
		} else {
			os.Setenv("ALLOW_INSECURE_CLIENT", originalEnv)
		}
	}()

	b.Run("AllowInsecureClient_Check", func(b *testing.B) {
		os.Setenv("ALLOW_INSECURE_CLIENT", "true")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = allowInsecureClient()
		}
	})

	b.Run("Secure_Client_Creation", func(b *testing.B) {
		os.Unsetenv("ALLOW_INSECURE_CLIENT")
		config := ClientConfig{
			APIKey:              "test-key",
			ModelName:           "test-model",
			MaxTokens:           1000,
			BackendType:         "openai",
			Timeout:             30 * time.Second,
			SkipTLSVerification: false,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			client := NewClientWithConfig("https://test.example.com", config)
			if client != nil {
				client.Shutdown()
			}
		}
	})

	b.Run("Insecure_Client_Creation", func(b *testing.B) {
		os.Setenv("ALLOW_INSECURE_CLIENT", "true")
		config := ClientConfig{
			APIKey:              "test-key",
			ModelName:           "test-model",
			MaxTokens:           1000,
			BackendType:         "openai",
			Timeout:             30 * time.Second,
			SkipTLSVerification: true,
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

// TestEdgeCases tests edge cases in TLS security
func TestEdgeCases(t *testing.T) {
	originalEnv := os.Getenv("ALLOW_INSECURE_CLIENT")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("ALLOW_INSECURE_CLIENT")
		} else {
			os.Setenv("ALLOW_INSECURE_CLIENT", originalEnv)
		}
	}()

	// Test concurrent client creation
	t.Run("Concurrent_Client_Creation", func(t *testing.T) {
		os.Unsetenv("ALLOW_INSECURE_CLIENT")

		config := ClientConfig{
			APIKey:              "test-key",
			ModelName:           "test-model",
			MaxTokens:           1000,
			BackendType:         "openai",
			Timeout:             30 * time.Second,
			SkipTLSVerification: false,
		}

		numGoroutines := 10
		clients := make([]*Client, numGoroutines)
		done := make(chan int, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				clients[index] = NewClientWithConfig("https://test.example.com", config)
				done <- index
			}(i)
		}

		// Wait for all clients to be created
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Verify all clients were created successfully
		for i, client := range clients {
			if client == nil {
				t.Errorf("Client %d was not created", i)
				continue
			}

			// Verify TLS configuration
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

			client.Shutdown()
		}
	})

	// Test environment variable changes during execution
	t.Run("Environment_Variable_Changes", func(t *testing.T) {
		// Start with secure
		os.Unsetenv("ALLOW_INSECURE_CLIENT")
		if allowInsecureClient() {
			t.Error("Expected false when env is unset")
		}

		// Change to insecure
		os.Setenv("ALLOW_INSECURE_CLIENT", "true")
		if !allowInsecureClient() {
			t.Error("Expected true when env is set to 'true'")
		}

		// Change back to secure
		os.Unsetenv("ALLOW_INSECURE_CLIENT")
		if allowInsecureClient() {
			t.Error("Expected false when env is unset again")
		}
	})
}