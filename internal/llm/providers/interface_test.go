package providers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		pt       ProviderType
		expected bool
	}{
		{
			name:     "Valid OFFLINE provider",
			pt:       ProviderTypeOffline,
			expected: true,
		},
		{
			name:     "Valid OPENAI provider",
			pt:       ProviderTypeOpenAI,
			expected: true,
		},
		{
			name:     "Valid ANTHROPIC provider",
			pt:       ProviderTypeAnthropic,
			expected: true,
		},
		{
			name:     "Invalid provider type",
			pt:       ProviderType("INVALID"),
			expected: false,
		},
		{
			name:     "Empty provider type",
			pt:       ProviderType(""),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.pt.IsValid()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProviderType_String(t *testing.T) {
	tests := []struct {
		name     string
		pt       ProviderType
		expected string
	}{
		{
			name:     "OFFLINE provider string",
			pt:       ProviderTypeOffline,
			expected: "OFFLINE",
		},
		{
			name:     "OPENAI provider string",
			pt:       ProviderTypeOpenAI,
			expected: "OPENAI",
		},
		{
			name:     "ANTHROPIC provider string",
			pt:       ProviderTypeAnthropic,
			expected: "ANTHROPIC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.pt.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid OFFLINE config",
			config: &Config{
				Type:       ProviderTypeOffline,
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			expectError: false,
		},
		{
			name: "Valid OPENAI config",
			config: &Config{
				Type:       ProviderTypeOpenAI,
				APIKey:     "sk-test-key",
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			expectError: false,
		},
		{
			name: "Valid ANTHROPIC config",
			config: &Config{
				Type:       ProviderTypeAnthropic,
				APIKey:     "claude-key",
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			expectError: false,
		},
		{
			name: "Invalid provider type",
			config: &Config{
				Type:       ProviderType("INVALID"),
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			expectError: true,
			errorMsg:    "provider type INVALID",
		},
		{
			name: "Missing API key for OPENAI",
			config: &Config{
				Type:       ProviderTypeOpenAI,
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			expectError: true,
			errorMsg:    "API key is required",
		},
		{
			name: "Missing API key for ANTHROPIC",
			config: &Config{
				Type:       ProviderTypeAnthropic,
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			expectError: true,
			errorMsg:    "API key is required",
		},
		{
			name: "Zero timeout gets default",
			config: &Config{
				Type:       ProviderTypeOffline,
				Timeout:    0,
				MaxRetries: 3,
			},
			expectError: false,
		},
		{
			name: "Negative retries gets default",
			config: &Config{
				Type:       ProviderTypeOffline,
				Timeout:    30 * time.Second,
				MaxRetries: -1,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				
				// Check that defaults were applied
				if tt.config.Timeout == 0 {
					assert.Equal(t, 30*time.Second, tt.config.Timeout)
				}
				if tt.config.MaxRetries < 0 {
					assert.Equal(t, 3, tt.config.MaxRetries)
				}
			}
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Rate limited error",
			err:      ErrRateLimited,
			expected: true,
		},
		{
			name:     "Provider unavailable error",
			err:      ErrProviderUnavailable,
			expected: true,
		},
		{
			name:     "Authentication error",
			err:      ErrAuthenticationFailed,
			expected: false,
		},
		{
			name:     "Invalid input error",
			err:      ErrInvalidInput,
			expected: false,
		},
		{
			name:     "Generic error",
			err:      assert.AnError,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Authentication failed error",
			err:      ErrAuthenticationFailed,
			expected: true,
		},
		{
			name:     "Rate limited error",
			err:      ErrRateLimited,
			expected: false,
		},
		{
			name:     "Generic error",
			err:      assert.AnError,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAuthError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsConfigError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Invalid configuration error",
			err:      ErrInvalidConfiguration,
			expected: true,
		},
		{
			name:     "Provider not supported error",
			err:      ErrProviderNotSupported,
			expected: true,
		},
		{
			name:     "Authentication failed error",
			err:      ErrAuthenticationFailed,
			expected: false,
		},
		{
			name:     "Generic error",
			err:      assert.AnError,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsConfigError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMockProvider_Integration(t *testing.T) {
	ctx := context.Background()
	config := &Config{
		Type:       ProviderType("MOCK"),
		Timeout:    30 * time.Second,
		MaxRetries: 3,
	}
	
	mock := NewMockProvider(config)
	defer mock.Close()

	t.Run("Default response", func(t *testing.T) {
		response, err := mock.ProcessIntent(ctx, "scale to 3 replicas")
		require.NoError(t, err)
		require.NotNil(t, response)
		
		// Verify JSON structure
		var intent map[string]interface{}
		err = json.Unmarshal(response.JSON, &intent)
		require.NoError(t, err)
		
		// Check required fields from intent.schema.json
		assert.Equal(t, "scaling", intent["intent_type"])
		assert.Equal(t, "mock-target", intent["target"])
		assert.Equal(t, "default", intent["namespace"])
		assert.Equal(t, float64(3), intent["replicas"]) // JSON numbers are float64
		
		// Check metadata
		assert.Equal(t, "MOCK", response.Metadata.Provider)
		assert.Equal(t, "mock-model", response.Metadata.Model)
	})

	t.Run("Custom response", func(t *testing.T) {
		customResponse := CreateMockIntentResponse("scaling", "custom-target", "custom-ns", 5)
		mock.SetMockResponse("custom input", customResponse)
		
		response, err := mock.ProcessIntent(ctx, "custom input")
		require.NoError(t, err)
		require.NotNil(t, response)
		
		// Verify custom response was returned
		var intent map[string]interface{}
		err = json.Unmarshal(response.JSON, &intent)
		require.NoError(t, err)
		
		assert.Equal(t, "custom-target", intent["target"])
		assert.Equal(t, "custom-ns", intent["namespace"])
		assert.Equal(t, float64(5), intent["replicas"])
	})

	t.Run("Error simulation", func(t *testing.T) {
		mock.SetMockError(ErrInvalidInput)
		
		_, err := mock.ProcessIntent(ctx, "test input")
		require.Error(t, err)
		assert.Equal(t, ErrInvalidInput, err)
	})

	t.Run("Call tracking", func(t *testing.T) {
		mock.Reset() // Clear previous calls
		
		_, _ = mock.ProcessIntent(ctx, "first call")
		_, _ = mock.ProcessIntent(ctx, "second call")
		
		assert.Equal(t, 2, mock.GetCallCount())
		assert.Equal(t, "second call", mock.GetLastInput())
		assert.Equal(t, []string{"first call", "second call"}, mock.ProcessIntentCalls)
	})

	t.Run("Provider info", func(t *testing.T) {
		info := mock.GetProviderInfo()
		assert.Equal(t, "MOCK", info.Name)
		assert.Equal(t, "1.0.0-test", info.Version)
		assert.False(t, info.RequiresAuth)
		assert.Contains(t, info.SupportedFeatures, "intent_processing")
	})
}