package validation

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestValidateEndpoint_EmptyEndpoint(t *testing.T) {
	// Empty endpoint should be allowed (optional endpoints)
	config := DefaultValidationConfig()
	err := ValidateEndpoint("", "Test Service", config)
	if err != nil {
		t.Errorf("Expected no error for empty endpoint, got: %v", err)
	}
}

func TestValidateEndpoint_ValidHTTP(t *testing.T) {
	config := DefaultValidationConfig()
	err := ValidateEndpoint("http://localhost:8080", "Test Service", config)
	if err != nil {
		t.Errorf("Expected no error for valid HTTP URL, got: %v", err)
	}
}

func TestValidateEndpoint_ValidHTTPS(t *testing.T) {
	config := DefaultValidationConfig()
	err := ValidateEndpoint("https://example.com:443/api/v1", "Test Service", config)
	if err != nil {
		t.Errorf("Expected no error for valid HTTPS URL, got: %v", err)
	}
}

func TestValidateEndpoint_InvalidScheme(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		scheme   string
	}{
		{
			name:     "FTP scheme",
			endpoint: "ftp://example.com",
			scheme:   "ftp",
		},
		{
			name:     "File scheme",
			endpoint: "file:///etc/passwd",
			scheme:   "file",
		},
		{
			name:     "Data scheme",
			endpoint: "data:text/plain;base64,SGVsbG8=",
			scheme:   "data",
		},
		{
			name:     "SSH scheme",
			endpoint: "ssh://user@host",
			scheme:   "ssh",
		},
	}

	config := DefaultValidationConfig()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEndpoint(tt.endpoint, "Test Service", config)
			if err == nil {
				t.Errorf("Expected error for %s scheme, got nil", tt.scheme)
			}
			if !strings.Contains(err.Error(), "unsupported URL scheme") {
				t.Errorf("Expected 'unsupported URL scheme' error, got: %v", err)
			}
			if !strings.Contains(err.Error(), tt.scheme) {
				t.Errorf("Expected error to mention scheme '%s', got: %v", tt.scheme, err)
			}
		})
	}
}

func TestValidateEndpoint_InvalidURL(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
	}{
		{
			name:     "Invalid characters",
			endpoint: "http://host with spaces:8080",
		},
		{
			name:     "Malformed URL",
			endpoint: "://missing-scheme",
		},
		{
			name:     "Just hostname",
			endpoint: "example.com",
		},
	}

	config := DefaultValidationConfig()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEndpoint(tt.endpoint, "Test Service", config)
			if err == nil {
				t.Errorf("Expected error for invalid URL, got nil")
			}
		})
	}
}

func TestValidateEndpoint_MissingHostname(t *testing.T) {
	config := DefaultValidationConfig()
	err := ValidateEndpoint("http://", "Test Service", config)
	if err == nil {
		t.Errorf("Expected error for missing hostname, got nil")
	}
	if !strings.Contains(err.Error(), "missing hostname") {
		t.Errorf("Expected 'missing hostname' error, got: %v", err)
	}
}

func TestValidateEndpoint_DNSResolution(t *testing.T) {
	tests := []struct {
		name      string
		endpoint  string
		shouldErr bool
	}{
		{
			name:      "Valid localhost",
			endpoint:  "http://localhost:8080",
			shouldErr: false,
		},
		{
			name:      "Valid IP address",
			endpoint:  "http://127.0.0.1:8080",
			shouldErr: false,
		},
		{
			name:      "Valid IP v6 address",
			endpoint:  "http://[::1]:8080",
			shouldErr: false,
		},
		{
			name:      "Invalid hostname",
			endpoint:  "http://this-hostname-definitely-does-not-exist-12345.invalid:8080",
			shouldErr: true,
		},
	}

	config := &ValidationConfig{
		ValidateDNS:          true,
		ValidateReachability: false,
		Timeout:              3 * time.Second,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEndpoint(tt.endpoint, "Test Service", config)
			if tt.shouldErr && err == nil {
				t.Errorf("Expected DNS error for %s, got nil", tt.endpoint)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error for %s, got: %v", tt.endpoint, err)
			}
			if tt.shouldErr && err != nil {
				// Verify helpful error message
				errMsg := err.Error()
				if !strings.Contains(errMsg, "DNS") && !strings.Contains(errMsg, "resolve") {
					t.Errorf("Expected DNS-related error message, got: %v", err)
				}
			}
		})
	}
}

func TestValidateEndpoint_Reachability(t *testing.T) {
	// Start a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tests := []struct {
		name      string
		endpoint  string
		shouldErr bool
	}{
		{
			name:      "Reachable endpoint",
			endpoint:  server.URL,
			shouldErr: false,
		},
		{
			name:      "Unreachable endpoint (wrong port)",
			endpoint:  "http://localhost:65534",
			shouldErr: true,
		},
	}

	config := &ValidationConfig{
		ValidateDNS:          false,
		ValidateReachability: true,
		Timeout:              2 * time.Second,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEndpoint(tt.endpoint, "Test Service", config)
			if tt.shouldErr && err == nil {
				t.Errorf("Expected reachability error for %s, got nil", tt.endpoint)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error for %s, got: %v", tt.endpoint, err)
			}
			if tt.shouldErr && err != nil {
				// Verify helpful error message
				errMsg := err.Error()
				if !strings.Contains(errMsg, "unreachable") {
					t.Errorf("Expected 'unreachable' in error message, got: %v", err)
				}
			}
		})
	}
}

func TestValidateEndpoints_Multiple(t *testing.T) {
	endpoints := map[string]string{
		"Valid Service":   "http://localhost:8080",
		"Invalid Scheme":  "ftp://example.com",
		"Empty Service":   "",
		"Another Valid":   "https://example.com",
		"Missing Host":    "http://",
	}

	config := DefaultValidationConfig()
	errors := ValidateEndpoints(endpoints, config)

	// Should have 2 errors (Invalid Scheme and Missing Host)
	// Empty Service should not error
	if len(errors) != 2 {
		t.Errorf("Expected 2 errors, got %d: %v", len(errors), errors)
	}

	// Verify error details
	foundSchemeError := false
	foundHostError := false
	for _, err := range errors {
		if strings.Contains(err.Error(), "unsupported URL scheme") {
			foundSchemeError = true
		}
		if strings.Contains(err.Error(), "missing hostname") {
			foundHostError = true
		}
	}

	if !foundSchemeError {
		t.Errorf("Expected to find scheme error in results")
	}
	if !foundHostError {
		t.Errorf("Expected to find missing host error in results")
	}
}

func TestGetCommonErrorSuggestions(t *testing.T) {
	tests := []struct {
		service     string
		expectMatch string
	}{
		{
			service:     "A1 Mediator",
			expectMatch: "A1_MEDIATOR_URL",
		},
		{
			service:     "Porch Server",
			expectMatch: "PORCH_SERVER_URL",
		},
		{
			service:     "LLM Service",
			expectMatch: "LLM_PROCESSOR_URL",
		},
		{
			service:     "RAG Service",
			expectMatch: "RAG_API_URL",
		},
		{
			service:     "Unknown Service",
			expectMatch: "environment variable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.service, func(t *testing.T) {
			suggestion := GetCommonErrorSuggestions(tt.service)
			if !strings.Contains(suggestion, tt.expectMatch) {
				t.Errorf("Expected suggestion to contain '%s', got: %s", tt.expectMatch, suggestion)
			}
		})
	}
}

func TestEndpointValidationError_Message(t *testing.T) {
	err := &EndpointValidationError{
		Endpoint: "http://bad-endpoint:8080",
		Service:  "Test Service",
		Err:      fmt.Errorf("connection refused"),
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "Test Service") {
		t.Errorf("Expected error message to contain service name, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "validation failed") {
		t.Errorf("Expected error message to contain 'validation failed', got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "connection refused") {
		t.Errorf("Expected error message to contain underlying error, got: %s", errMsg)
	}
}

func TestValidateEndpoint_KubernetesServiceNames(t *testing.T) {
	// NOTE: These are test examples for URL validation only.
	// In production, use environment variables to configure endpoints:
	//   - PORCH_SERVER_URL or PORCH_ENDPOINT for Porch
	//   - Recommended default: http://porch-server.porch-system.svc.cluster.local:7007
	tests := []struct {
		name     string
		endpoint string
	}{
		{
			name:     "Simple service (legacy port example)",
			endpoint: "http://porch-server:8080",
		},
		{
			name:     "Service with namespace",
			endpoint: "http://service.namespace.svc.cluster.local:8080",
		},
		{
			name:     "A1 Mediator typical URL",
			endpoint: "http://service-ricplt-a1mediator-http.ricplt:8080",
		},
		{
			name:     "Ollama service",
			endpoint: "http://ollama-service.ollama.svc.cluster.local:11434",
		},
	}

	config := DefaultValidationConfig()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEndpoint(tt.endpoint, "Test Service", config)
			if err != nil {
				t.Errorf("Expected no error for valid Kubernetes service name, got: %v", err)
			}
		})
	}
}

func TestDefaultValidationConfig(t *testing.T) {
	config := DefaultValidationConfig()

	if config == nil {
		t.Fatal("DefaultValidationConfig returned nil")
	}

	// DNS and reachability should be disabled by default for fast startup
	if config.ValidateDNS {
		t.Error("Expected ValidateDNS to be false by default")
	}
	if config.ValidateReachability {
		t.Error("Expected ValidateReachability to be false by default")
	}
	if config.Timeout != 5*time.Second {
		t.Errorf("Expected default timeout of 5s, got: %v", config.Timeout)
	}
}
