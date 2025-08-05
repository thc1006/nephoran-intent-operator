package middleware

import (
	"fmt"
	"strings"
	"testing"
)

// Mock LLMProcessorConfig for testing our validation logic
type MockLLMProcessorConfig struct {
	MaxRequestSize int64
}

// Mock validate function that replicates our validation logic
func (c *MockLLMProcessorConfig) Validate() error {
	var errors []string

	// Replicate our validation logic
	if c.MaxRequestSize <= 0 {
		errors = append(errors, "MAX_REQUEST_SIZE must be positive")
	}

	if c.MaxRequestSize < 1024 {
		errors = append(errors, "MAX_REQUEST_SIZE should be at least 1KB for basic functionality")
	}

	if c.MaxRequestSize > 100*1024*1024 { // 100MB
		errors = append(errors, "MAX_REQUEST_SIZE should not exceed 100MB to prevent DoS attacks")
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}

	return nil
}

func TestMaxRequestSizeValidation(t *testing.T) {
	tests := []struct {
		name        string
		maxSize     int64
		expectError bool
		errorSubstr string
	}{
		{
			name:        "Valid size - 1MB",
			maxSize:     1024 * 1024,
			expectError: false,
		},
		{
			name:        "Valid size - 10MB",
			maxSize:     10 * 1024 * 1024,
			expectError: false,
		},
		{
			name:        "Valid size - exactly 1KB",
			maxSize:     1024,
			expectError: false,
		},
		{
			name:        "Valid size - exactly 100MB",
			maxSize:     100 * 1024 * 1024,
			expectError: false,
		},
		{
			name:        "Zero size - invalid",
			maxSize:     0,
			expectError: true,
			errorSubstr: "must be positive",
		},
		{
			name:        "Negative size - invalid",
			maxSize:     -1,
			expectError: true,
			errorSubstr: "must be positive",
		},
		{
			name:        "Too small - invalid",
			maxSize:     512, // Less than 1KB
			expectError: true,
			errorSubstr: "at least 1KB",
		},
		{
			name:        "Too large - invalid",
			maxSize:     200 * 1024 * 1024, // 200MB, exceeds 100MB limit
			expectError: true,
			errorSubstr: "should not exceed 100MB",
		},
		{
			name:        "Edge case - just under 1KB",
			maxSize:     1023,
			expectError: true,
			errorSubstr: "at least 1KB",
		},
		{
			name:        "Edge case - just over 100MB",
			maxSize:     100*1024*1024 + 1,
			expectError: true,
			errorSubstr: "should not exceed 100MB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &MockLLMProcessorConfig{
				MaxRequestSize: tt.maxSize,
			}

			err := cfg.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected validation error for size %d, but got none", tt.maxSize)
				} else if !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("Expected error containing '%s', got: %s", tt.errorSubstr, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no validation error for size %d, got: %s", tt.maxSize, err.Error())
				}
			}
		})
	}
}

func TestHumanReadableSizes(t *testing.T) {
	// Test that our size limits make sense in human terms
	tests := []struct {
		name        string
		sizeBytes   int64
		description string
	}{
		{
			name:        "1KB minimum",
			sizeBytes:   1024,
			description: "Basic JSON request",
		},
		{
			name:        "1MB default",
			sizeBytes:   1024 * 1024,
			description: "Typical AI request with context",
		},
		{
			name:        "10MB reasonable",
			sizeBytes:   10 * 1024 * 1024,
			description: "Large context or document upload",
		},
		{
			name:        "100MB maximum",
			sizeBytes:   100 * 1024 * 1024,
			description: "Maximum to prevent DoS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &MockLLMProcessorConfig{
				MaxRequestSize: tt.sizeBytes,
			}

			err := cfg.Validate()
			if err != nil {
				t.Errorf("Size %s (%d bytes) should be valid but got error: %s", 
					tt.name, tt.sizeBytes, err.Error())
			}

			t.Logf("%s: %d bytes (%.2f MB) - %s", 
				tt.name, tt.sizeBytes, float64(tt.sizeBytes)/(1024*1024), tt.description)
		})
	}
}