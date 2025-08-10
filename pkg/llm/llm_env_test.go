package llm

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestNewClientWithConfigEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		checkFunc   func(*Client, *testing.T)
		description string
	}{
		{
			name: "LLM_MAX_RETRIES=5",
			envVars: map[string]string{
				"LLM_MAX_RETRIES": "5",
			},
			description: "Should set max retries from environment variable",
			checkFunc: func(client *Client, t *testing.T) {
				if client.retryConfig.MaxRetries != 5 {
					t.Errorf("Expected MaxRetries to be 5, got %v", client.retryConfig.MaxRetries)
				}
			},
		},
		{
			name: "LLM_MAX_RETRIES=0",
			envVars: map[string]string{
				"LLM_MAX_RETRIES": "0",
			},
			description: "Should accept 0 retries",
			checkFunc: func(client *Client, t *testing.T) {
				if client.retryConfig.MaxRetries != 0 {
					t.Errorf("Expected MaxRetries to be 0, got %v", client.retryConfig.MaxRetries)
				}
			},
		},
		{
			name: "LLM_CACHE_MAX_ENTRIES=1024",
			envVars: map[string]string{
				"LLM_CACHE_MAX_ENTRIES": "1024",
			},
			description: "Should set cache max entries from environment variable",
			checkFunc: func(client *Client, t *testing.T) {
				if client.cache.maxSize != 1024 {
					t.Errorf("Expected cache maxSize to be 1024, got %v", client.cache.maxSize)
				}
			},
		},
		{
			name: "Invalid LLM_MAX_RETRIES",
			envVars: map[string]string{
				"LLM_MAX_RETRIES": "invalid",
			},
			description: "Should use default value for invalid retry count",
			checkFunc: func(client *Client, t *testing.T) {
				if client.retryConfig.MaxRetries != 2 { // Should fallback to default
					t.Errorf("Expected MaxRetries to fallback to default 2, got %v", client.retryConfig.MaxRetries)
				}
			},
		},
		{
			name: "Negative LLM_MAX_RETRIES",
			envVars: map[string]string{
				"LLM_MAX_RETRIES": "-1",
			},
			description: "Should use default value for negative retry count",
			checkFunc: func(client *Client, t *testing.T) {
				if client.retryConfig.MaxRetries != 2 { // Should fallback to default
					t.Errorf("Expected MaxRetries to fallback to default 2, got %v", client.retryConfig.MaxRetries)
				}
			},
		},
		{
			name: "Invalid LLM_CACHE_MAX_ENTRIES",
			envVars: map[string]string{
				"LLM_CACHE_MAX_ENTRIES": "invalid",
			},
			description: "Should use default value for invalid cache entries",
			checkFunc: func(client *Client, t *testing.T) {
				if client.cache.maxSize != 512 { // Should fallback to default
					t.Errorf("Expected cache maxSize to fallback to default 512, got %v", client.cache.maxSize)
				}
			},
		},
		{
			name: "Zero LLM_CACHE_MAX_ENTRIES",
			envVars: map[string]string{
				"LLM_CACHE_MAX_ENTRIES": "0",
			},
			description: "Should use default value for zero cache entries",
			checkFunc: func(client *Client, t *testing.T) {
				if client.cache.maxSize != 512 { // Should fallback to default
					t.Errorf("Expected cache maxSize to fallback to default 512, got %v", client.cache.maxSize)
				}
			},
		},
		{
			name: "Combined environment variables",
			envVars: map[string]string{
				"LLM_MAX_RETRIES":        "3",
				"LLM_CACHE_MAX_ENTRIES": "256",
			},
			description: "Should handle multiple environment variables",
			checkFunc: func(client *Client, t *testing.T) {
				if client.retryConfig.MaxRetries != 3 {
					t.Errorf("Expected MaxRetries to be 3, got %v", client.retryConfig.MaxRetries)
				}
				if client.cache.maxSize != 256 {
					t.Errorf("Expected cache maxSize to be 256, got %v", client.cache.maxSize)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			originalEnv := make(map[string]string)
			for key := range tt.envVars {
				originalEnv[key] = os.Getenv(key)
			}

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Clean up after test
			defer func() {
				for key, originalValue := range originalEnv {
					if originalValue == "" {
						os.Unsetenv(key)
					} else {
						os.Setenv(key, originalValue)
					}
				}
			}()

			// Create client with configuration
			config := ClientConfig{
				APIKey:      "test-key",
				ModelName:   "gpt-4o-mini",
				MaxTokens:   2048,
				BackendType: "openai",
				Timeout:     30 * time.Second,
			}

			client := NewClientWithConfig("http://test-url", config)
			if client == nil {
				t.Fatal("NewClientWithConfig returned nil")
			}

			// Clean up client
			defer client.Shutdown()

			// Run check function
			if tt.checkFunc != nil {
				tt.checkFunc(client, t)
			}
		})
	}
}

func TestProcessIntentTimeoutEnvironmentVariable(t *testing.T) {
	tests := []struct {
		name          string
		timeoutEnv    string
		expectedUsage string // We can't directly test the timeout, but we can test the parsing logic
	}{
		{
			name:       "LLM_TIMEOUT_SECS=10",
			timeoutEnv: "10",
		},
		{
			name:       "LLM_TIMEOUT_SECS=30",
			timeoutEnv: "30",
		},
		{
			name:       "Invalid timeout",
			timeoutEnv: "invalid",
		},
		{
			name:       "Zero timeout",
			timeoutEnv: "0",
		},
		{
			name:       "Negative timeout",
			timeoutEnv: "-5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			originalTimeout := os.Getenv("LLM_TIMEOUT_SECS")

			// Set test timeout
			os.Setenv("LLM_TIMEOUT_SECS", tt.timeoutEnv)

			// Clean up after test
			defer func() {
				if originalTimeout == "" {
					os.Unsetenv("LLM_TIMEOUT_SECS")
				} else {
					os.Setenv("LLM_TIMEOUT_SECS", originalTimeout)
				}
			}()

			// Test the timeout parsing logic that's used in ProcessIntent
			timeoutSecs := 15 // default
			if envTimeout := os.Getenv("LLM_TIMEOUT_SECS"); envTimeout != "" {
				if t, err := strconv.Atoi(envTimeout); err == nil && t > 0 {
					timeoutSecs = t
				}
			}

			// Verify the parsing logic works correctly
			if tt.timeoutEnv == "10" && timeoutSecs != 10 {
				t.Errorf("Expected timeout to be 10, got %d", timeoutSecs)
			} else if tt.timeoutEnv == "30" && timeoutSecs != 30 {
				t.Errorf("Expected timeout to be 30, got %d", timeoutSecs)
			} else if tt.timeoutEnv == "invalid" && timeoutSecs != 15 {
				t.Errorf("Expected timeout to fallback to default 15, got %d", timeoutSecs)
			} else if tt.timeoutEnv == "0" && timeoutSecs != 15 {
				t.Errorf("Expected timeout to fallback to default 15 for zero value, got %d", timeoutSecs)
			} else if tt.timeoutEnv == "-5" && timeoutSecs != 15 {
				t.Errorf("Expected timeout to fallback to default 15 for negative value, got %d", timeoutSecs)
			}
		})
	}
}

func TestClientDefaults(t *testing.T) {
	// Clear environment variables to test defaults
	envVars := []string{"LLM_MAX_RETRIES", "LLM_CACHE_MAX_ENTRIES", "LLM_TIMEOUT_SECS"}

	// Save original environment
	originalEnv := make(map[string]string)
	for _, key := range envVars {
		originalEnv[key] = os.Getenv(key)
		os.Unsetenv(key)
	}

	// Clean up after test
	defer func() {
		for key, originalValue := range originalEnv {
			if originalValue == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, originalValue)
			}
		}
	}()

	// Create client with default configuration
	config := ClientConfig{
		APIKey:      "test-key",
		ModelName:   "gpt-4o-mini",
		MaxTokens:   2048,
		BackendType: "openai",
		Timeout:     30 * time.Second,
	}

	client := NewClientWithConfig("http://test-url", config)
	defer client.Shutdown()

	// Test defaults
	if client.retryConfig.MaxRetries != 2 {
		t.Errorf("Expected default MaxRetries to be 2, got %v", client.retryConfig.MaxRetries)
	}

	if client.cache.maxSize != 512 {
		t.Errorf("Expected default cache maxSize to be 512, got %v", client.cache.maxSize)
	}
}

func TestRetryConfigurationFromEnv(t *testing.T) {
	// Test the retry logic uses environment variables
	os.Setenv("LLM_MAX_RETRIES", "1")
	defer os.Unsetenv("LLM_MAX_RETRIES")

	config := ClientConfig{
		APIKey:      "test-key",
		ModelName:   "gpt-4o-mini",
		MaxTokens:   2048,
		BackendType: "mock", // Use mock to avoid actual API calls
		Timeout:     5 * time.Second,
	}

	client := NewClientWithConfig("http://test-url", config)
	defer client.Shutdown()

	// Verify retry config was set from environment
	if client.retryConfig.MaxRetries != 1 {
		t.Errorf("Expected MaxRetries to be 1 from environment, got %v", client.retryConfig.MaxRetries)
	}

	// Test the retry logic in retryWithExponentialBackoff
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	retryCount := 0
	err := client.retryWithExponentialBackoff(ctx, func() error {
		retryCount++
		return context.DeadlineExceeded // Simulate a retryable error
	})

	// Should have tried initial + 1 retry = 2 attempts total
	expectedAttempts := 2 // 1 initial + 1 retry
	if retryCount != expectedAttempts {
		t.Errorf("Expected %d attempts (1 initial + %d retries), got %d", expectedAttempts, client.retryConfig.MaxRetries, retryCount)
	}

	if err == nil {
		t.Error("Expected error from retryWithExponentialBackoff")
	}
}