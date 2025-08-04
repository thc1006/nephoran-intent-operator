package oran

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestTLSIntegration demonstrates the complete TLS configuration flow
func TestTLSIntegration(t *testing.T) {
	// Create a test HTTPS server
	testServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello from secure server"))
	}))
	defer testServer.Close()

	// Test with InsecureSkipVerify = true (for testing purposes)
	tlsConfig := &TLSConfig{
		SkipVerify: true,
	}

	config, err := BuildTLSConfig(tlsConfig)
	if err != nil {
		t.Fatalf("Failed to build TLS config: %v", err)
	}

	// Create HTTP client with TLS configuration
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: config,
		},
		Timeout: 10 * time.Second,
	}

	// Make a request to the test server
	resp, err := client.Get(testServer.URL)
	if err != nil {
		t.Fatalf("Failed to make HTTPS request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

// TestE2AdaptorTLSConfiguration demonstrates TLS configuration for E2 adaptor
func TestE2AdaptorTLSConfiguration(t *testing.T) {
	// Mock E2 adaptor configuration with TLS
	mockConfig := struct {
		RICURL                  string
		APIVersion              string
		Timeout                 time.Duration
		TLSConfig               *TLSConfig
		HeartbeatInterval       time.Duration
		MaxRetries              int
	}{
		RICURL:            "https://near-rt-ric:38080",
		APIVersion:        "v1",
		Timeout:           30 * time.Second,
		HeartbeatInterval: 30 * time.Second,
		MaxRetries:        3,
		TLSConfig: &TLSConfig{
			SkipVerify: true, // For testing only
		},
	}

	// Validate TLS configuration
	err := ValidateTLSConfig(mockConfig.TLSConfig)
	if err != nil {
		t.Errorf("TLS configuration validation failed: %v", err)
	}

	// Build TLS configuration
	tlsConfig, err := BuildTLSConfig(mockConfig.TLSConfig)
	if err != nil {
		t.Errorf("Failed to build TLS configuration: %v", err)
	}

	// Verify TLS configuration properties
	if tlsConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("Expected TLS 1.2 minimum version, got %v", tlsConfig.MinVersion)
	}

	if !tlsConfig.PreferServerCipherSuites {
		t.Error("Expected server cipher suite preference to be enabled")
	}

	if len(tlsConfig.CipherSuites) == 0 {
		t.Error("Expected cipher suites to be configured")
	}

	// Create HTTP client with TLS configuration
	httpClient := &http.Client{
		Timeout: mockConfig.Timeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	if httpClient == nil {
		t.Error("Expected HTTP client to be created")
	}
}

// TestA1AdaptorTLSConfiguration demonstrates TLS configuration for A1 adaptor
func TestA1AdaptorTLSConfiguration(t *testing.T) {
	// Mock A1 adaptor configuration with TLS
	mockConfig := struct {
		RICURL     string
		APIVersion string
		Timeout    time.Duration
		TLSConfig  *TLSConfig
	}{
		RICURL:     "https://near-rt-ric:8080",
		APIVersion: "v1",
		Timeout:    30 * time.Second,
		TLSConfig: &TLSConfig{
			SkipVerify: true, // For testing only
		},
	}

	// Validate TLS configuration
	err := ValidateTLSConfig(mockConfig.TLSConfig)
	if err != nil {
		t.Errorf("TLS configuration validation failed: %v", err)
	}

	// Build TLS configuration
	tlsConfig, err := BuildTLSConfig(mockConfig.TLSConfig)
	if err != nil {
		t.Errorf("Failed to build TLS configuration: %v", err)
	}

	// Verify TLS configuration properties
	if tlsConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("Expected TLS 1.2 minimum version, got %v", tlsConfig.MinVersion)
	}

	// Create HTTP client with TLS configuration
	httpClient := &http.Client{
		Timeout: mockConfig.Timeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	if httpClient == nil {
		t.Error("Expected HTTP client to be created")
	}
}

// TestMutualTLSScenario demonstrates a mutual TLS scenario
func TestMutualTLSScenario(t *testing.T) {
	// This test demonstrates how mutual TLS would work in practice
	// In a real scenario, you would have actual certificate files

	tlsConfig := &TLSConfig{
		CertFile:   "/path/to/client.crt",
		KeyFile:    "/path/to/client.key",
		CAFile:     "/path/to/ca.crt",
		SkipVerify: false,
	}

	// Validate configuration (will fail due to non-existent files, but demonstrates the flow)
	err := ValidateTLSConfig(tlsConfig)
	if err == nil {
		t.Error("Expected validation to fail with non-existent files")
	}

	// This demonstrates the validation catches missing files
	if err != nil && len(err.Error()) > 0 {
		// The error should indicate missing certificate file
		t.Logf("Validation correctly caught missing files: %v", err)
	}
}

// TestSecurityCompliance verifies that the TLS configuration meets security standards
func TestSecurityCompliance(t *testing.T) {
	config := &TLSConfig{
		SkipVerify: false,
	}

	tlsConfig, err := BuildTLSConfig(config)
	if err != nil {
		t.Fatalf("Failed to build TLS config: %v", err)
	}

	// Verify minimum TLS version compliance
	if tlsConfig.MinVersion < tls.VersionTLS12 {
		t.Errorf("TLS version below minimum required (TLS 1.2), got %v", tlsConfig.MinVersion)
	}

	// Verify cipher suite security
	secureCiphers := map[uint16]string{
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384:   "ECDHE-RSA-AES256-GCM-SHA384",
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305:    "ECDHE-RSA-CHACHA20-POLY1305",
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384: "ECDHE-ECDSA-AES256-GCM-SHA384",
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305:  "ECDHE-ECDSA-CHACHA20-POLY1305",
	}

	secureFound := 0
	for _, cipher := range tlsConfig.CipherSuites {
		if _, isSecure := secureCiphers[cipher]; isSecure {
			secureFound++
		}
	}

	if secureFound == 0 {
		t.Error("No secure cipher suites found in configuration")
	}

	t.Logf("Found %d secure cipher suites configured", secureFound)
}

// BenchmarkTLSConfigBuild benchmarks the TLS configuration building process
func BenchmarkTLSConfigBuild(b *testing.B) {
	config := &TLSConfig{
		SkipVerify: false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := BuildTLSConfig(config)
		if err != nil {
			b.Fatalf("Failed to build TLS config: %v", err)
		}
	}
}

// ExampleBuildTLSConfig demonstrates how to use the TLS helper functions
func ExampleBuildTLSConfig() {
	// Create a TLS configuration for production use
	config := GetRecommendedTLSConfig(
		"/etc/ssl/certs/client.crt",
		"/etc/ssl/private/client.key",
		"/etc/ssl/certs/ca.crt",
	)

	// Validate the configuration
	if err := ValidateTLSConfig(config); err != nil {
		fmt.Printf("TLS configuration validation failed: %v\n", err)
		return
	}

	// Build the TLS configuration
	tlsConfig, err := BuildTLSConfig(config)
	if err != nil {
		fmt.Printf("Failed to build TLS configuration: %v\n", err)
		return
	}

	// Use the TLS configuration with an HTTP client
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
		Timeout: 30 * time.Second,
	}

	// The client is now configured for mutual TLS
	fmt.Printf("HTTP client configured with mutual TLS, minimum version: %v\n", tlsConfig.MinVersion)
	_ = client // Use the client variable to avoid unused variable error
}