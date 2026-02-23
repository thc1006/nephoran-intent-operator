package main

import (
	"os"
	"strings"
	"testing"
)

func TestValidateEndpoints_AllEmpty(t *testing.T) {
	// Clear all environment variables
	clearEndpointEnvVars(t)

	// With no endpoints configured, validation should pass (all are optional)
	err := validateEndpoints("", "", "", false)
	if err != nil {
		t.Errorf("Expected no error when all endpoints are empty, got: %v", err)
	}
}

func TestValidateEndpoints_ValidHTTP(t *testing.T) {
	clearEndpointEnvVars(t)

	err := validateEndpoints(
		"http://a1-mediator:8080",
		"http://llm-service:11434",
		"http://porch-server:7007",
		false,
	)
	if err != nil {
		t.Errorf("Expected no error for valid HTTP URLs, got: %v", err)
	}
}

func TestValidateEndpoints_InvalidScheme(t *testing.T) {
	clearEndpointEnvVars(t)

	err := validateEndpoints(
		"ftp://a1-mediator:8080",
		"",
		"",
		false,
	)
	if err == nil {
		t.Error("Expected error for FTP scheme, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported URL scheme") {
		t.Errorf("Expected 'unsupported URL scheme' error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "A1_MEDIATOR_URL") {
		t.Errorf("Expected error to suggest A1_MEDIATOR_URL configuration, got: %v", err)
	}
}

func TestValidateEndpoints_MissingHost(t *testing.T) {
	clearEndpointEnvVars(t)

	err := validateEndpoints(
		"http://",
		"",
		"",
		false,
	)
	if err == nil {
		t.Error("Expected error for missing hostname, got nil")
	}
	if !strings.Contains(err.Error(), "missing hostname") {
		t.Errorf("Expected 'missing hostname' error, got: %v", err)
	}
}

func TestValidateEndpoints_FromEnvironment(t *testing.T) {
	clearEndpointEnvVars(t)

	// Set environment variables
	os.Setenv("A1_MEDIATOR_URL", "http://a1-from-env:8080")
	os.Setenv("LLM_PROCESSOR_URL", "http://llm-from-env:11434")
	defer func() {
		os.Unsetenv("A1_MEDIATOR_URL")
		os.Unsetenv("LLM_PROCESSOR_URL")
	}()

	// Flags are empty, should read from environment
	err := validateEndpoints("", "", "", false)
	if err != nil {
		t.Errorf("Expected no error when reading from environment, got: %v", err)
	}
}

func TestValidateEndpoints_FlagOverridesEnvironment(t *testing.T) {
	clearEndpointEnvVars(t)

	// Set environment variables with invalid URLs
	os.Setenv("A1_MEDIATOR_URL", "ftp://bad-from-env:8080")
	defer os.Unsetenv("A1_MEDIATOR_URL")

	// Flag with valid URL should override environment
	err := validateEndpoints("http://a1-from-flag:8080", "", "", false)
	if err != nil {
		t.Errorf("Expected no error when flag overrides environment, got: %v", err)
	}
}

func TestValidateEndpoints_MultipleErrors(t *testing.T) {
	clearEndpointEnvVars(t)

	err := validateEndpoints(
		"ftp://bad-a1:8080",     // Invalid scheme
		"http://",               // Missing host
		"ssh://bad-porch:7007",  // Invalid scheme
		false,
	)
	if err == nil {
		t.Error("Expected error for multiple invalid endpoints, got nil")
	}

	errMsg := err.Error()
	// Should contain multiple errors
	if !strings.Contains(errMsg, "A1 Mediator") {
		t.Errorf("Expected error to mention A1 Mediator, got: %v", err)
	}
	if !strings.Contains(errMsg, "LLM Service") {
		t.Errorf("Expected error to mention LLM Service, got: %v", err)
	}
	if !strings.Contains(errMsg, "Porch Server") {
		t.Errorf("Expected error to mention Porch Server, got: %v", err)
	}
}

func TestValidateEndpoints_DNSValidation(t *testing.T) {
	clearEndpointEnvVars(t)

	// localhost should always resolve
	err := validateEndpoints("http://localhost:8080", "", "", true)
	if err != nil {
		t.Errorf("Expected no error for localhost with DNS validation, got: %v", err)
	}

	// Non-existent hostname should fail with DNS validation enabled
	err = validateEndpoints("http://this-definitely-does-not-exist-12345.invalid:8080", "", "", true)
	if err == nil {
		t.Error("Expected DNS error for non-existent hostname, got nil")
	}
	if !strings.Contains(err.Error(), "resolve") && !strings.Contains(err.Error(), "DNS") {
		t.Errorf("Expected DNS-related error, got: %v", err)
	}
}

func TestGetFirstNonEmpty(t *testing.T) {
	tests := []struct {
		name     string
		values   []string
		expected string
	}{
		{
			name:     "First is non-empty",
			values:   []string{"first", "second", "third"},
			expected: "first",
		},
		{
			name:     "Second is first non-empty",
			values:   []string{"", "second", "third"},
			expected: "second",
		},
		{
			name:     "All empty",
			values:   []string{"", "", ""},
			expected: "",
		},
		{
			name:     "Empty slice",
			values:   []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFirstNonEmpty(tt.values...)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExtractServiceName(t *testing.T) {
	tests := []struct {
		name     string
		errorMsg string
		expected string
	}{
		{
			name:     "A1 Mediator error",
			errorMsg: "A1 Mediator endpoint validation failed: invalid URL",
			expected: "A1 Mediator",
		},
		{
			name:     "LLM Service error",
			errorMsg: "LLM Service endpoint validation failed: missing hostname",
			expected: "LLM Service",
		},
		{
			name:     "Unknown format",
			errorMsg: "Some other error",
			expected: "Unknown Service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractServiceName(tt.errorMsg)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestEndpointValidationFailure_Error(t *testing.T) {
	err := &EndpointValidationFailure{
		Message: "Test validation failure",
	}

	if err.Error() != "Test validation failure" {
		t.Errorf("Expected error message to match, got: %s", err.Error())
	}
}

// Helper function to clear all endpoint-related environment variables
func clearEndpointEnvVars(t *testing.T) {
	t.Helper()
	envVars := []string{
		"A1_MEDIATOR_URL",
		"A1_ENDPOINT",
		"LLM_PROCESSOR_URL",
		"LLM_ENDPOINT",
		"PORCH_SERVER_URL",
		"PORCH_ENDPOINT",
		"RAG_API_URL",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}
}
