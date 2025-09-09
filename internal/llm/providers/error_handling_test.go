package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProviderErrorHandling tests comprehensive error scenarios across all providers
func TestProviderErrorHandling(t *testing.T) {
	tests := []struct {
		name            string
		providerType    ProviderType
		config          *Config
		input           string
		expectedError   error
		errorContains   string
		setupEnv        map[string]string
		cleanupEnv      []string
		contextTimeout  time.Duration
	}{
		{
			name:         "OFFLINE Provider - Nil Config",
			providerType: ProviderTypeOffline,
			config:       nil,
			input:        "test input",
			expectedError: ErrInvalidConfiguration,
			errorContains: "config cannot be nil",
		},
		{
			name:         "OFFLINE Provider - Wrong Provider Type",
			providerType: ProviderTypeOffline,
			config: &Config{
				Type:       ProviderTypeOpenAI, // Wrong type
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			input:         "test input",
			expectedError: ErrInvalidConfiguration,
			errorContains: "does not match requested provider type",
		},
		{
			name:         "OFFLINE Provider - Negative Timeout",
			providerType: ProviderTypeOffline,
			config: &Config{
				Type:       ProviderTypeOffline,
				Timeout:    -1 * time.Second, // Invalid negative timeout
				MaxRetries: 3,
			},
			input:         "test input",
			expectedError: ErrInvalidConfiguration,
			errorContains: "timeout must be positive",
		},
		{
			name:         "OpenAI Provider - Missing API Key",
			providerType: ProviderTypeOpenAI,
			config: &Config{
				Type:       ProviderTypeOpenAI,
				APIKey:     "", // Missing API key
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			input:         "test input",
			expectedError: ErrInvalidConfiguration,
			errorContains: "API key is required",
		},
		{
			name:         "Anthropic Provider - Missing API Key",
			providerType: ProviderTypeAnthropic,
			config: &Config{
				Type:       ProviderTypeAnthropic,
				APIKey:     "", // Missing API key
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			input:         "test input",
			expectedError: ErrInvalidConfiguration,
			errorContains: "API key is required",
		},
		{
			name:         "OFFLINE Provider - Empty Input",
			providerType: ProviderTypeOffline,
			config: &Config{
				Type:       ProviderTypeOffline,
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			input:         "",
			expectedError: ErrInvalidInput,
			errorContains: "input cannot be empty",
		},
		{
			name:         "OFFLINE Provider - Context Timeout",
			providerType: ProviderTypeOffline,
			config: &Config{
				Type:       ProviderTypeOffline,
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			input:          "test input",
			contextTimeout: 1 * time.Nanosecond, // Very short timeout
			errorContains:  "context deadline exceeded",
		},
		{
			name:         "Factory - Unsupported Provider Type",
			providerType: ProviderType("UNSUPPORTED"),
			config: &Config{
				Type:       ProviderType("UNSUPPORTED"),
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			input:         "test input",
			expectedError: ErrProviderNotSupported,
			errorContains: "provider type UNSUPPORTED",
		},
		{
			name:         "Factory - Invalid Config Validation",
			providerType: ProviderTypeOffline,
			config: &Config{
				Type:       ProviderTypeOffline,
				Timeout:    -1, // Invalid timeout
				MaxRetries: -5, // Invalid retries
			},
			input:         "test input",
			expectedError: ErrInvalidConfiguration,
			errorContains: "timeout must be positive",
		},
		{
			name:         "Environment Config - Invalid Provider String Defaults to Offline",
			providerType: ProviderTypeOffline,
			setupEnv:     map[string]string{"LLM_PROVIDER": "INVALID_PROVIDER"},
			cleanupEnv:   []string{"LLM_PROVIDER"},
			input:        "test input",
			// Should not error - should default to offline provider
		},
		{
			name:         "Environment Config - Invalid Timeout String",
			providerType: ProviderTypeOffline,
			setupEnv:     map[string]string{"LLM_PROVIDER": "OFFLINE", "LLM_TIMEOUT": "invalid"},
			cleanupEnv:   []string{"LLM_PROVIDER", "LLM_TIMEOUT"},
			input:        "test input",
			errorContains: "invalid LLM_TIMEOUT",
		},
		{
			name:         "Environment Config - Invalid Max Retries String",
			providerType: ProviderTypeOffline,
			setupEnv:     map[string]string{"LLM_PROVIDER": "OFFLINE", "LLM_MAX_RETRIES": "not_a_number"},
			cleanupEnv:   []string{"LLM_PROVIDER", "LLM_MAX_RETRIES"},
			input:        "test input",
			errorContains: "invalid LLM_MAX_RETRIES",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment variables
			originalEnv := make(map[string]string)
			for key, value := range tt.setupEnv {
				originalEnv[key] = os.Getenv(key)
				os.Setenv(key, value)
			}
			defer func() {
				for key, originalValue := range originalEnv {
					if originalValue == "" {
						os.Unsetenv(key)
					} else {
						os.Setenv(key, originalValue)
					}
				}
			}()

			var provider Provider
			var err error

			// Test environment-based configuration if setupEnv is provided
			if len(tt.setupEnv) > 0 {
				config, err := ConfigFromEnvironment()
				if tt.errorContains != "" && err != nil {
					assert.Contains(t, err.Error(), tt.errorContains, "Environment config error should contain expected text")
					return // Expected error from environment config
				}
				// For successful environment config cases, create provider
				if err == nil && config != nil {
					factory := NewFactory()
					provider, err = factory.CreateProvider(config)
				}
			}

			// Create provider through factory (even if config is nil to test error handling)
			if len(tt.setupEnv) == 0 {
				factory := NewFactory()
				
				// For tests that check provider type validation, use ValidateProviderConfig
				if tt.name == "OFFLINE Provider - Wrong Provider Type" {
					err = factory.ValidateProviderConfig(tt.providerType, tt.config)
				} else {
					provider, err = factory.CreateProvider(tt.config)
				}
			}

			// Check for expected errors during provider creation
			// Some tests expect errors during creation, others during processing
			creationErrorCases := []string{
				"OFFLINE Provider - Nil Config",
				"OFFLINE Provider - Wrong Provider Type", 
				"OFFLINE Provider - Negative Timeout",
				"OpenAI Provider - Missing API Key",
				"Anthropic Provider - Missing API Key",
				"Factory - Unsupported Provider Type",
				"Factory - Invalid Config Validation",
			}
			
			shouldFailDuringCreation := false
			for _, name := range creationErrorCases {
				if tt.name == name {
					shouldFailDuringCreation = true
					break
				}
			}
			
			if shouldFailDuringCreation && tt.expectedError != nil {
				assert.Error(t, err, "Should return an error during creation")
				assert.ErrorIs(t, err, tt.expectedError, "Should return expected error type")
				if tt.errorContains != "" && err != nil {
					assert.Contains(t, err.Error(), tt.errorContains, "Error should contain expected text")
				}
				return
			}

			if err != nil && tt.errorContains != "" {
				assert.Contains(t, err.Error(), tt.errorContains, "Error should contain expected text")
				return
			}

			require.NoError(t, err, "Provider creation should not fail")
			require.NotNil(t, provider, "Provider should not be nil")
			defer provider.Close()

			// Test ProcessIntent with context timeout if specified
			var ctx context.Context
			var cancel context.CancelFunc

			if tt.contextTimeout > 0 {
				ctx, cancel = context.WithTimeout(context.Background(), tt.contextTimeout)
				// For very short timeouts, ensure the context is already expired
				if tt.contextTimeout <= time.Millisecond {
					time.Sleep(tt.contextTimeout + time.Millisecond)
				}
			} else {
				ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
			}
			defer cancel()

			// Process intent
			response, err := provider.ProcessIntent(ctx, tt.input)

			// Check for expected errors during processing
			if tt.expectedError != nil && !shouldFailDuringCreation {
				assert.Error(t, err, "Should return expected error during processing")
				assert.ErrorIs(t, err, tt.expectedError, "Should return expected error type")
				if tt.errorContains != "" && err != nil {
					assert.Contains(t, err.Error(), tt.errorContains, "Processing error should contain expected text")
				}
				return
			}

			if tt.errorContains != "" && tt.expectedError == nil {
				assert.Error(t, err, "Should return an error during processing")
				if err != nil {
					assert.Contains(t, err.Error(), tt.errorContains, "Processing error should contain expected text")
				}
				return
			}

			// For successful cases, verify response
			assert.NoError(t, err, "Processing should not fail")
			assert.NotNil(t, response, "Response should not be nil")
		})
	}
}

// TestProviderEdgeCases tests edge cases and boundary conditions
func TestProviderEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		config       *Config
		input        string
		expectError  bool
		validateFunc func(t *testing.T, response *IntentResponse, err error)
	}{
		{
			name: "Extremely Long Input",
			config: &Config{
				Type:       ProviderTypeOffline,
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			input:       strings.Repeat("scale workload ", 10000), // Very long input
			expectError: false,
			validateFunc: func(t *testing.T, response *IntentResponse, err error) {
				assert.NoError(t, err, "Should handle long input")
				assert.NotNil(t, response, "Response should not be nil")
				assert.NotNil(t, response.JSON, "Response JSON should not be nil")
			},
		},
		{
			name: "Input with Special Characters",
			config: &Config{
				Type:       ProviderTypeOffline,
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			input:       "Scale 网络功能 to 3 replicas with UTF-8 chars: αβγ, 你好世界, emoji: 🚀",
			expectError: false,
			validateFunc: func(t *testing.T, response *IntentResponse, err error) {
				assert.NoError(t, err, "Should handle special characters")
				assert.NotNil(t, response, "Response should not be nil")
			},
		},
		{
			name: "Input with JSON-like Content",
			config: &Config{
				Type:       ProviderTypeOffline,
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			input:       `Scale workload with config {"replicas": 5, "namespace": "test"}`,
			expectError: false,
			validateFunc: func(t *testing.T, response *IntentResponse, err error) {
				assert.NoError(t, err, "Should handle JSON-like input")
				assert.NotNil(t, response, "Response should not be nil")
				
				// Verify the JSON is valid
				var jsonData map[string]interface{}
				err = json.Unmarshal(response.JSON, &jsonData)
				assert.NoError(t, err, "Response should contain valid JSON")
			},
		},
		{
			name: "Input with Control Characters",
			config: &Config{
				Type:       ProviderTypeOffline,
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			input:       "Scale workload\x00\x01\x02 with control chars",
			expectError: false,
			validateFunc: func(t *testing.T, response *IntentResponse, err error) {
				assert.NoError(t, err, "Should handle control characters")
				assert.NotNil(t, response, "Response should not be nil")
			},
		},
		{
			name: "Maximum Timeout Configuration",
			config: &Config{
				Type:       ProviderTypeOffline,
				Timeout:    24 * time.Hour, // Very long timeout
				MaxRetries: 100,            // Many retries
			},
			input:       "Scale workload to 3 replicas",
			expectError: false,
			validateFunc: func(t *testing.T, response *IntentResponse, err error) {
				assert.NoError(t, err, "Should handle maximum timeout")
				assert.NotNil(t, response, "Response should not be nil")
			},
		},
		{
			name: "Minimum Valid Configuration",
			config: &Config{
				Type:       ProviderTypeOffline,
				Timeout:    1 * time.Millisecond, // Very short but positive
				MaxRetries: 0,                    // No retries
			},
			input:       "Scale",
			expectError: false,
			validateFunc: func(t *testing.T, response *IntentResponse, err error) {
				assert.NoError(t, err, "Should handle minimum configuration")
				assert.NotNil(t, response, "Response should not be nil")
			},
		},
		{
			name: "Single Character Input",
			config: &Config{
				Type:       ProviderTypeOffline,
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			input:       "s",
			expectError: false,
			validateFunc: func(t *testing.T, response *IntentResponse, err error) {
				assert.NoError(t, err, "Should handle single character")
				assert.NotNil(t, response, "Response should not be nil")
			},
		},
		{
			name: "Whitespace Only Input",
			config: &Config{
				Type:       ProviderTypeOffline,
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			input:       "   \t\n\r   ",
			expectError: false,
			validateFunc: func(t *testing.T, response *IntentResponse, err error) {
				assert.NoError(t, err, "Should handle whitespace input")
				assert.NotNil(t, response, "Response should not be nil")
			},
		},
		{
			name: "Mixed Language Input",
			config: &Config{
				Type:       ProviderTypeOffline,
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			input:       "Scale POD до 5 réplicas в namespace テスト",
			expectError: false,
			validateFunc: func(t *testing.T, response *IntentResponse, err error) {
				assert.NoError(t, err, "Should handle mixed language input")
				assert.NotNil(t, response, "Response should not be nil")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewFactory()
			provider, err := factory.CreateProvider(tt.config)
			if tt.expectError {
				assert.Error(t, err, "Should error for invalid configuration")
				return
			}
			require.NoError(t, err, "Failed to create provider")
			require.NotNil(t, provider, "Provider should not be nil")
			defer provider.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			response, err := provider.ProcessIntent(ctx, tt.input)
			
			if tt.validateFunc != nil {
				tt.validateFunc(t, response, err)
			}
		})
	}
}

// TestConcurrentProviderUsage tests provider thread safety and concurrent access
func TestConcurrentProviderUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	config := &Config{
		Type:       ProviderTypeOffline,
		Timeout:    30 * time.Second,
		MaxRetries: 3,
	}

	factory := NewFactory()
	provider, err := factory.CreateProvider(config)
	require.NoError(t, err, "Failed to create provider")
	defer provider.Close()

	const numGoroutines = 20
	const numRequestsPerGoroutine = 10

	results := make(chan error, numGoroutines*numRequestsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			for j := 0; j < numRequestsPerGoroutine; j++ {
				input := fmt.Sprintf("Scale workload-%d-%d to %d replicas", goroutineID, j, j+1)
				
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				response, err := provider.ProcessIntent(ctx, input)
				cancel()

				if err != nil {
					results <- fmt.Errorf("goroutine %d, request %d: %w", goroutineID, j, err)
					continue
				}

				if response == nil {
					results <- fmt.Errorf("goroutine %d, request %d: nil response", goroutineID, j)
					continue
				}

				if response.JSON == nil {
					results <- fmt.Errorf("goroutine %d, request %d: nil JSON in response", goroutineID, j)
					continue
				}

				// Verify JSON is valid
				var jsonData map[string]interface{}
				if err := json.Unmarshal(response.JSON, &jsonData); err != nil {
					results <- fmt.Errorf("goroutine %d, request %d: invalid JSON: %w", goroutineID, j, err)
					continue
				}

				results <- nil
			}
		}(i)
	}

	// Collect results
	errorCount := 0
	for i := 0; i < numGoroutines*numRequestsPerGoroutine; i++ {
		err := <-results
		if err != nil {
			t.Errorf("Concurrent request failed: %v", err)
			errorCount++
		}
	}

	assert.Equal(t, 0, errorCount, "All concurrent requests should succeed")
}

// TestProviderMemoryLeaks tests for memory leaks in provider usage
func TestProviderMemoryLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	config := &Config{
		Type:       ProviderTypeOffline,
		Timeout:    30 * time.Second,
		MaxRetries: 3,
	}

	// Test creating and closing many providers
	for i := 0; i < 1000; i++ {
		factory := NewFactory()
		provider, err := factory.CreateProvider(config)
		require.NoError(t, err, "Failed to create provider")

		// Process a request
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err = provider.ProcessIntent(ctx, "test input")
		cancel()
		assert.NoError(t, err, "Processing should not fail")

		// Close provider
		err = provider.Close()
		assert.NoError(t, err, "Close should not fail")
	}
}

// TestProviderStateConsistency tests provider state consistency
func TestProviderStateConsistency(t *testing.T) {
	config := &Config{
		Type:       ProviderTypeOffline,
		Timeout:    30 * time.Second,
		MaxRetries: 3,
	}

	factory := NewFactory()
	provider, err := factory.CreateProvider(config)
	require.NoError(t, err, "Failed to create provider")

	// Test provider info consistency
	info1 := provider.GetProviderInfo()
	info2 := provider.GetProviderInfo()
	assert.Equal(t, info1, info2, "Provider info should be consistent")

	// Test config validation consistency
	err1 := provider.ValidateConfig()
	err2 := provider.ValidateConfig()
	assert.Equal(t, err1, err2, "Config validation should be consistent")

	// Test processing consistency with same input
	input := "Scale workload to 3 replicas"
	
	ctx1, cancel1 := context.WithTimeout(context.Background(), 30*time.Second)
	response1, err1 := provider.ProcessIntent(ctx1, input)
	cancel1()

	ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	response2, err2 := provider.ProcessIntent(ctx2, input)
	cancel2()

	assert.Equal(t, err1, err2, "Errors should be consistent")
	if err1 == nil {
		assert.NotNil(t, response1, "First response should not be nil")
		assert.NotNil(t, response2, "Second response should not be nil")
		assert.Equal(t, response1.Metadata.Provider, response2.Metadata.Provider, "Provider name should be consistent")
	}

	// Test cleanup
	err = provider.Close()
	assert.NoError(t, err, "Provider close should not fail")
}

// TestFactoryEdgeCases tests factory edge cases and error conditions
func TestFactoryEdgeCases(t *testing.T) {
	factory := NewFactory()

	tests := []struct {
		name          string
		config        *Config
		expectError   bool
		expectedError error
	}{
		{
			name:          "Nil Config",
			config:        nil,
			expectError:   true,
			expectedError: ErrInvalidConfiguration,
		},
		{
			name: "Config with nil Extra field",
			config: &Config{
				Type:       ProviderTypeOffline,
				Timeout:    30 * time.Second,
				MaxRetries: 3,
				Extra:      nil, // Should not cause issues
			},
			expectError: false,
		},
		{
			name: "Config with empty Extra map",
			config: &Config{
				Type:       ProviderTypeOffline,
				Timeout:    30 * time.Second,
				MaxRetries: 3,
				Extra:      make(map[string]interface{}),
			},
			expectError: false,
		},
		{
			name: "Config with complex Extra data",
			config: &Config{
				Type:       ProviderTypeOffline,
				Timeout:    30 * time.Second,
				MaxRetries: 3,
				Extra: map[string]interface{}{
					"nested": map[string]interface{}{
						"deep": []interface{}{1, 2, 3, "test"},
					},
					"array": []interface{}{},
					"null":  nil,
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := factory.CreateProvider(tt.config)
			
			if tt.expectError {
				assert.Error(t, err, "Should return an error")
				if tt.expectedError != nil {
					assert.ErrorIs(t, err, tt.expectedError, "Should return expected error type")
				}
				assert.Nil(t, provider, "Provider should be nil on error")
			} else {
				assert.NoError(t, err, "Should not return an error")
				assert.NotNil(t, provider, "Provider should not be nil")
				provider.Close()
			}
		})
	}

	// Test supported providers
	supportedProviders := factory.GetSupportedProviders()
	assert.Contains(t, supportedProviders, ProviderTypeOffline, "Should support OFFLINE provider")
	assert.Contains(t, supportedProviders, ProviderTypeOpenAI, "Should support OPENAI provider")
	assert.Contains(t, supportedProviders, ProviderTypeAnthropic, "Should support ANTHROPIC provider")

	// Test provider config validation
	validConfig := &Config{
		Type:       ProviderTypeOffline,
		Timeout:    30 * time.Second,
		MaxRetries: 3,
	}
	err := factory.ValidateProviderConfig(ProviderTypeOffline, validConfig)
	assert.NoError(t, err, "Valid config should pass validation")

	invalidConfig := &Config{
		Type:       ProviderTypeOpenAI, // Mismatch
		Timeout:    30 * time.Second,
		MaxRetries: 3,
	}
	err = factory.ValidateProviderConfig(ProviderTypeOffline, invalidConfig)
	assert.Error(t, err, "Mismatched config should fail validation")
}

