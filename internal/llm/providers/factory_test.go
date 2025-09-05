package providers

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory()
	require.NotNil(t, factory)
	
	// Should be a DefaultFactory
	defaultFactory, ok := factory.(*DefaultFactory)
	require.True(t, ok)
	require.NotNil(t, defaultFactory.constructors)
}

func TestDefaultFactory_RegisterProvider(t *testing.T) {
	factory := NewFactory().(*DefaultFactory)
	
	// Register a mock constructor
	mockConstructor := func(config *Config) (Provider, error) {
		return NewMockProvider(config), nil
	}
	
	factory.RegisterProvider(ProviderType("TEST"), mockConstructor)
	
	// Verify it was registered
	assert.Contains(t, factory.constructors, ProviderType("TEST"))
}

func TestDefaultFactory_CreateProvider(t *testing.T) {
	factory := NewFactory().(*DefaultFactory)
	
	// Register a mock constructor
	mockConstructor := func(config *Config) (Provider, error) {
		return NewMockProvider(config), nil
	}
	factory.RegisterProvider(ProviderType("MOCK"), mockConstructor)
	
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Nil config",
			config:      nil,
			expectError: true,
			errorMsg:    "config cannot be nil",
		},
		{
			name: "Invalid config",
			config: &Config{
				Type: ProviderType("INVALID"),
			},
			expectError: true,
			errorMsg:    "invalid config",
		},
		{
			name: "Unsupported provider",
			config: &Config{
				Type:       ProviderType("UNSUPPORTED"),
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			expectError: true,
			errorMsg:    "provider type UNSUPPORTED",
		},
		{
			name: "Valid offline provider config",
			config: &Config{
				Type:       ProviderTypeOffline,
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			expectError: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := factory.CreateProvider(tt.config)
			
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, provider)
			} else {
				require.NoError(t, err)
				require.NotNil(t, provider)
				
				// Clean up
				provider.Close()
			}
		})
	}
}

func TestDefaultFactory_CreateProvider_WithConstructor(t *testing.T) {
	factory := NewFactory().(*DefaultFactory)
	
	// Register a mock constructor for testing
	mockConstructor := func(config *Config) (Provider, error) {
		return NewMockProvider(config), nil
	}
	factory.RegisterProvider(ProviderTypeOffline, mockConstructor)
	
	config := &Config{
		Type:       ProviderTypeOffline,
		Timeout:    30 * time.Second,
		MaxRetries: 3,
	}
	
	provider, err := factory.CreateProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)
	
	// Verify it's the correct type
	mockProvider, ok := provider.(*MockProvider)
	require.True(t, ok)
	
	info := mockProvider.GetProviderInfo()
	assert.Equal(t, "MOCK", info.Name)
	
	// Clean up
	provider.Close()
}

func TestDefaultFactory_GetSupportedProviders(t *testing.T) {
	factory := NewFactory().(*DefaultFactory)
	
	// Should have all providers registered by default
	providers := factory.GetSupportedProviders()
	assert.Len(t, providers, 3)
	assert.Contains(t, providers, ProviderTypeOffline)
	assert.Contains(t, providers, ProviderTypeOpenAI)
	assert.Contains(t, providers, ProviderTypeAnthropic)
	
	// Test with a fresh factory with no registrations
	emptyFactory := &DefaultFactory{
		constructors: make(map[ProviderType]func(*Config) (Provider, error)),
	}
	
	emptyProviders := emptyFactory.GetSupportedProviders()
	assert.Empty(t, emptyProviders)
	
	// Register a provider manually
	mockConstructor := func(config *Config) (Provider, error) {
		return NewMockProvider(config), nil
	}
	
	emptyFactory.RegisterProvider(ProviderType("MOCK"), mockConstructor)
	
	mockProviders := emptyFactory.GetSupportedProviders()
	assert.Len(t, mockProviders, 1)
	assert.Contains(t, mockProviders, ProviderType("MOCK"))
}

func TestDefaultFactory_ValidateProviderConfig(t *testing.T) {
	factory := NewFactory().(*DefaultFactory)
	
	// Register a mock constructor
	mockConstructor := func(config *Config) (Provider, error) {
		return NewMockProvider(config), nil
	}
	factory.RegisterProvider(ProviderTypeOffline, mockConstructor)
	
	tests := []struct {
		name         string
		providerType ProviderType
		config       *Config
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "Nil config",
			providerType: ProviderTypeOffline,
			config:       nil,
			expectError:  true,
			errorMsg:     "config cannot be nil",
		},
		{
			name:         "Mismatched provider type",
			providerType: ProviderTypeOffline,
			config: &Config{
				Type: ProviderTypeOpenAI,
			},
			expectError: true,
			errorMsg:    "does not match requested provider type",
		},
		{
			name:         "Unsupported provider",
			providerType: ProviderType("UNSUPPORTED"),
			config: &Config{
				Type: ProviderType("UNSUPPORTED"),
			},
			expectError: true,
			errorMsg:    "not supported",
		},
		{
			name:         "Valid config",
			providerType: ProviderTypeOffline,
			config: &Config{
				Type:       ProviderTypeOffline,
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			expectError: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := factory.ValidateProviderConfig(tt.providerType, tt.config)
			
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfigFromEnvironment(t *testing.T) {
	// Save original environment
	originalProvider := os.Getenv("LLM_PROVIDER")
	originalAPIKey := os.Getenv("LLM_API_KEY")
	originalBaseURL := os.Getenv("LLM_BASE_URL")
	originalModel := os.Getenv("LLM_MODEL")
	originalTimeout := os.Getenv("LLM_TIMEOUT")
	originalRetries := os.Getenv("LLM_MAX_RETRIES")
	
	// Clean up after test
	defer func() {
		os.Setenv("LLM_PROVIDER", originalProvider)
		os.Setenv("LLM_API_KEY", originalAPIKey)
		os.Setenv("LLM_BASE_URL", originalBaseURL)
		os.Setenv("LLM_MODEL", originalModel)
		os.Setenv("LLM_TIMEOUT", originalTimeout)
		os.Setenv("LLM_MAX_RETRIES", originalRetries)
	}()
	
	tests := []struct {
		name        string
		envVars     map[string]string
		expectError bool
		errorMsg    string
		validate    func(t *testing.T, config *Config)
	}{
		{
			name:    "Default configuration",
			envVars: map[string]string{},
			validate: func(t *testing.T, config *Config) {
				assert.Equal(t, ProviderTypeOffline, config.Type)
				assert.Equal(t, 30*time.Second, config.Timeout)
				assert.Equal(t, 3, config.MaxRetries)
			},
		},
		{
			name: "Full configuration",
			envVars: map[string]string{
				"LLM_PROVIDER":    "OPENAI",
				"LLM_API_KEY":     "sk-test",
				"LLM_BASE_URL":    "https://api.example.com",
				"LLM_MODEL":       "gpt-4",
				"LLM_TIMEOUT":     "60",
				"LLM_MAX_RETRIES": "5",
			},
			validate: func(t *testing.T, config *Config) {
				assert.Equal(t, ProviderTypeOpenAI, config.Type)
				assert.Equal(t, "sk-test", config.APIKey)
				assert.Equal(t, "https://api.example.com", config.BaseURL)
				assert.Equal(t, "gpt-4", config.Model)
				assert.Equal(t, 60*time.Second, config.Timeout)
				assert.Equal(t, 5, config.MaxRetries)
			},
		},
		{
			name: "Invalid provider defaults to offline",
			envVars: map[string]string{
				"LLM_PROVIDER": "INVALID",
			},
			expectError: false,
			validate: func(t *testing.T, config *Config) {
				assert.Equal(t, ProviderTypeOffline, config.Type)
				assert.Equal(t, 30*time.Second, config.Timeout)
				assert.Equal(t, 3, config.MaxRetries)
			},
		},
		{
			name: "Invalid timeout",
			envVars: map[string]string{
				"LLM_TIMEOUT": "invalid",
			},
			expectError: true,
			errorMsg:    "invalid LLM_TIMEOUT",
		},
		{
			name: "Invalid retries",
			envVars: map[string]string{
				"LLM_MAX_RETRIES": "invalid",
			},
			expectError: true,
			errorMsg:    "invalid LLM_MAX_RETRIES",
		},
		{
			name: "Case insensitive provider",
			envVars: map[string]string{
				"LLM_PROVIDER": "openai",
			},
			validate: func(t *testing.T, config *Config) {
				assert.Equal(t, ProviderTypeOpenAI, config.Type)
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all environment variables
			os.Unsetenv("LLM_PROVIDER")
			os.Unsetenv("LLM_API_KEY")
			os.Unsetenv("LLM_BASE_URL")
			os.Unsetenv("LLM_MODEL")
			os.Unsetenv("LLM_TIMEOUT")
			os.Unsetenv("LLM_MAX_RETRIES")
			
			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			
			config, err := ConfigFromEnvironment()
			
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, config)
			} else {
				require.NoError(t, err)
				require.NotNil(t, config)
				
				if tt.validate != nil {
					tt.validate(t, config)
				}
			}
		})
	}
}

func TestGetProviderTypeFromEnv(t *testing.T) {
	originalProvider := os.Getenv("LLM_PROVIDER")
	defer os.Setenv("LLM_PROVIDER", originalProvider)
	
	tests := []struct {
		name     string
		envValue string
		expected ProviderType
	}{
		{
			name:     "Not set",
			envValue: "",
			expected: ProviderTypeOffline,
		},
		{
			name:     "Valid OPENAI",
			envValue: "OPENAI",
			expected: ProviderTypeOpenAI,
		},
		{
			name:     "Valid lowercase",
			envValue: "anthropic",
			expected: ProviderTypeAnthropic,
		},
		{
			name:     "Invalid value",
			envValue: "INVALID",
			expected: ProviderTypeOffline,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue == "" {
				os.Unsetenv("LLM_PROVIDER")
			} else {
				os.Setenv("LLM_PROVIDER", tt.envValue)
			}
			
			result := GetProviderTypeFromEnv()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateFromEnvironment_Integration(t *testing.T) {
	// This test demonstrates how CreateFromEnvironment would work
	// when providers are registered with the factory
	
	// Save original environment
	originalProvider := os.Getenv("LLM_PROVIDER")
	defer os.Setenv("LLM_PROVIDER", originalProvider)
	
	// Set test environment
	os.Setenv("LLM_PROVIDER", "OFFLINE")
	
	// Test config creation from environment
	config, err := ConfigFromEnvironment()
	require.NoError(t, err)
	require.NotNil(t, config)
	
	assert.Equal(t, ProviderTypeOffline, config.Type)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 3, config.MaxRetries)
	
	// Test factory creation with all providers registered
	factory := NewFactory()
	require.NotNil(t, factory)
	
	supportedProviders := factory.GetSupportedProviders()
	// Should have all three providers registered
	assert.Len(t, supportedProviders, 3)
	assert.Contains(t, supportedProviders, ProviderTypeOffline)
	assert.Contains(t, supportedProviders, ProviderTypeOpenAI)
	assert.Contains(t, supportedProviders, ProviderTypeAnthropic)
}