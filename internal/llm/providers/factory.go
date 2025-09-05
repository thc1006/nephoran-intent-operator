package providers

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

// DefaultFactory is the standard implementation of the Factory interface.
type DefaultFactory struct {
	// Registry of provider constructors
	constructors map[ProviderType]func(*Config) (Provider, error)
}

// NewFactory creates a new provider factory with all supported providers registered.
func NewFactory() Factory {
	f := &DefaultFactory{
		constructors: make(map[ProviderType]func(*Config) (Provider, error)),
	}

	// Register provider constructors for all supported provider types
	f.RegisterProvider(ProviderTypeOffline, func(config *Config) (Provider, error) {
		return NewOfflineProvider(config)
	})
	f.RegisterProvider(ProviderTypeOpenAI, func(config *Config) (Provider, error) {
		return NewOpenAIProvider(config)
	})
	f.RegisterProvider(ProviderTypeAnthropic, func(config *Config) (Provider, error) {
		return NewAnthropicProvider(config)
	})

	return f
}

// RegisterProvider registers a constructor function for a provider type.
// This allows for extensibility - new providers can be added by registering constructors.
func (f *DefaultFactory) RegisterProvider(providerType ProviderType, constructor func(*Config) (Provider, error)) {
	f.constructors[providerType] = constructor
}

// CreateProvider creates a new provider instance based on the config.
func (f *DefaultFactory) CreateProvider(config *Config) (Provider, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Get the constructor for the requested provider type
	constructor, exists := f.constructors[config.Type]
	if !exists {
		return nil, fmt.Errorf("provider type %s: %w", config.Type, ErrProviderNotSupported)
	}

	// Create the provider instance
	provider, err := constructor(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider %s: %w", config.Type, err)
	}

	// Validate the provider configuration
	if err := provider.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("provider configuration validation failed: %w", err)
	}

	return provider, nil
}

// GetSupportedProviders returns a list of supported provider types.
func (f *DefaultFactory) GetSupportedProviders() []ProviderType {
	providers := make([]ProviderType, 0, len(f.constructors))
	for providerType := range f.constructors {
		providers = append(providers, providerType)
	}
	return providers
}

// ValidateProviderConfig validates configuration for a specific provider type.
func (f *DefaultFactory) ValidateProviderConfig(providerType ProviderType, config *Config) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if config.Type != providerType {
		return fmt.Errorf("config type %s does not match requested provider type %s", config.Type, providerType)
	}

	// Use the provider's validation logic
	constructor, exists := f.constructors[providerType]
	if !exists {
		return fmt.Errorf("provider type %s: %w", providerType, ErrProviderNotSupported)
	}

	// Create a temporary provider to validate the config
	provider, err := constructor(config)
	if err != nil {
		return fmt.Errorf("failed to create provider for validation: %w", err)
	}
	defer provider.Close() // Clean up the temporary provider

	return provider.ValidateConfig()
}

// ConfigFromEnvironment creates a provider config from environment variables.
// This is a convenience function for common deployment scenarios.
func ConfigFromEnvironment() (*Config, error) {
	// Get provider type from environment
	providerTypeStr := os.Getenv("LLM_PROVIDER")
	if providerTypeStr == "" {
		providerTypeStr = "OFFLINE" // Default to offline provider
	}

	providerType := ProviderType(strings.ToUpper(providerTypeStr))
	if !providerType.IsValid() {
		return nil, fmt.Errorf("invalid LLM_PROVIDER environment variable: %s", providerTypeStr)
	}

	config := &Config{
		Type:   providerType,
		APIKey: os.Getenv("LLM_API_KEY"),
		BaseURL: os.Getenv("LLM_BASE_URL"),
		Model:  os.Getenv("LLM_MODEL"),
		Extra:  make(map[string]interface{}),
	}

	// Parse timeout
	if timeoutStr := os.Getenv("LLM_TIMEOUT"); timeoutStr != "" {
		timeoutSeconds, err := strconv.Atoi(timeoutStr)
		if err != nil || timeoutSeconds < 0 || timeoutSeconds > math.MaxInt32 {
			return nil, fmt.Errorf("invalid LLM_TIMEOUT: %s", timeoutStr)
		}
		config.Timeout = time.Duration(timeoutSeconds) * time.Second
	} else {
		config.Timeout = 30 * time.Second
	}

	// Parse max retries
	if retriesStr := os.Getenv("LLM_MAX_RETRIES"); retriesStr != "" {
		maxRetries, err := strconv.Atoi(retriesStr)
		if err != nil || maxRetries < 0 || maxRetries > math.MaxInt32 {
			return nil, fmt.Errorf("invalid LLM_MAX_RETRIES: %s", retriesStr)
		}
		config.MaxRetries = maxRetries
	} else {
		config.MaxRetries = 3
	}

	return config, nil
}

// MustCreateFromEnvironment creates a provider from environment variables.
// It panics if the configuration is invalid or provider creation fails.
// This is useful for startup scenarios where invalid config should terminate the application.
func MustCreateFromEnvironment() Provider {
	config, err := ConfigFromEnvironment()
	if err != nil {
		panic(fmt.Sprintf("failed to create config from environment: %v", err))
	}

	factory := NewFactory()
	provider, err := factory.CreateProvider(config)
	if err != nil {
		panic(fmt.Sprintf("failed to create provider: %v", err))
	}

	return provider
}

// CreateFromEnvironment creates a provider from environment variables.
// Returns an error if the configuration is invalid or provider creation fails.
func CreateFromEnvironment() (Provider, error) {
	config, err := ConfigFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to create config from environment: %w", err)
	}

	factory := NewFactory()
	return factory.CreateProvider(config)
}

// GetProviderTypeFromEnv returns the provider type from the LLM_PROVIDER environment variable.
// Returns ProviderTypeOffline as the default if not set or invalid.
func GetProviderTypeFromEnv() ProviderType {
	providerTypeStr := os.Getenv("LLM_PROVIDER")
	if providerTypeStr == "" {
		return ProviderTypeOffline
	}

	providerType := ProviderType(strings.ToUpper(providerTypeStr))
	if !providerType.IsValid() {
		return ProviderTypeOffline
	}

	return providerType
}