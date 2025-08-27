package providers

import (
	"fmt"
	"sync"
)

// DefaultProviderFactory implements ProviderFactory interface
type DefaultProviderFactory struct {
	mu           sync.RWMutex
	constructors map[string]ProviderConstructor
	schemas      map[string]map[string]interface{}
}

// ProviderConstructor is a function that creates a new provider instance
type ProviderConstructor func(config ProviderConfig) (Provider, error)

// NewDefaultProviderFactory creates a new provider factory
func NewDefaultProviderFactory() *DefaultProviderFactory {
	return &DefaultProviderFactory{
		constructors: make(map[string]ProviderConstructor),
		schemas:      make(map[string]map[string]interface{}),
	}
}

// RegisterProvider registers a provider constructor with the factory
func (f *DefaultProviderFactory) RegisterProvider(
	providerType string,
	constructor ProviderConstructor,
	schema map[string]interface{},
) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.constructors[providerType]; exists {
		return fmt.Errorf("provider type %s is already registered", providerType)
	}

	f.constructors[providerType] = constructor
	if schema != nil {
		f.schemas[providerType] = schema
	}

	return nil
}

// CreateProvider creates a new provider instance
func (f *DefaultProviderFactory) CreateProvider(providerType string, config ProviderConfig) (Provider, error) {
	f.mu.RLock()
	constructor, exists := f.constructors[providerType]
	f.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}

	provider, err := constructor(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider %s: %w", providerType, err)
	}

	return provider, nil
}

// ListSupportedTypes returns supported provider types
func (f *DefaultProviderFactory) ListSupportedTypes() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	types := make([]string, 0, len(f.constructors))
	for providerType := range f.constructors {
		types = append(types, providerType)
	}

	return types
}

// GetProviderSchema returns configuration schema for a provider type
func (f *DefaultProviderFactory) GetProviderSchema(providerType string) (map[string]interface{}, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	schema, exists := f.schemas[providerType]
	if !exists {
		return nil, fmt.Errorf("no schema found for provider type: %s", providerType)
	}

	return schema, nil
}

// UnregisterProvider removes a provider type from the factory
func (f *DefaultProviderFactory) UnregisterProvider(providerType string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.constructors[providerType]; !exists {
		return fmt.Errorf("provider type %s is not registered", providerType)
	}

	delete(f.constructors, providerType)
	delete(f.schemas, providerType)

	return nil
}

// Global factory instance
var globalFactory = NewDefaultProviderFactory()

// RegisterGlobalProvider registers a provider with the global factory
func RegisterGlobalProvider(
	providerType string,
	constructor ProviderConstructor,
	schema map[string]interface{},
) error {
	return globalFactory.RegisterProvider(providerType, constructor, schema)
}

// CreateGlobalProvider creates a provider using the global factory
func CreateGlobalProvider(providerType string, config ProviderConfig) (Provider, error) {
	return globalFactory.CreateProvider(providerType, config)
}

// GetGlobalFactory returns the global provider factory instance
func GetGlobalFactory() ProviderFactory {
	return globalFactory
}
