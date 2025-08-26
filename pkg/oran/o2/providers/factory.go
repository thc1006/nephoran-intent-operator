package providers

import (
	"fmt"
	"sync"
)

// DefaultProviderFactory implements ProviderFactory interface
type DefaultProviderFactory struct {
	mu          sync.RWMutex
	constructors map[string]ProviderConstructor
	schemas     map[string]map[string]interface{}
}

// ProviderConstructor is a function that creates a new provider instance
type ProviderConstructor func(config ProviderConfig) (Provider, error)

// NewDefaultProviderFactory creates a new provider factory
func NewDefaultProviderFactory() *DefaultProviderFactory {
	return &DefaultProviderFactory{
		constructors: make(map[string]ProviderConstructor),
		schemas:     make(map[string]map[string]interface{}),
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

// ProviderRegistry manages multiple provider instances
type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]Provider
	factory   ProviderFactory
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry(factory ProviderFactory) *ProviderRegistry {
	if factory == nil {
		factory = globalFactory
	}
	
	return &ProviderRegistry{
		providers: make(map[string]Provider),
		factory:   factory,
	}
}

// RegisterProvider registers a provider instance with a unique name
func (r *ProviderRegistry) RegisterProvider(name string, provider Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider with name %s already exists", name)
	}
	
	r.providers[name] = provider
	return nil
}

// GetProvider returns a provider by name
func (r *ProviderRegistry) GetProvider(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	provider, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}
	
	return provider, nil
}

// CreateAndRegisterProvider creates a new provider and registers it
func (r *ProviderRegistry) CreateAndRegisterProvider(name, providerType string, config ProviderConfig) error {
	provider, err := r.factory.CreateProvider(providerType, config)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}
	
	return r.RegisterProvider(name, provider)
}

// ListProviders returns names of all registered providers
func (r *ProviderRegistry) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	
	return names
}

// UnregisterProvider removes a provider from the registry
func (r *ProviderRegistry) UnregisterProvider(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	provider, exists := r.providers[name]
	if !exists {
		return fmt.Errorf("provider %s not found", name)
	}
	
	// Close the provider if it supports cleanup
	if err := provider.Close(); err != nil {
		return fmt.Errorf("failed to close provider %s: %w", name, err)
	}
	
	delete(r.providers, name)
	return nil
}

// Close closes all registered providers
func (r *ProviderRegistry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	var lastErr error
	for name, provider := range r.providers {
		if err := provider.Close(); err != nil {
			lastErr = fmt.Errorf("failed to close provider %s: %w", name, err)
		}
	}
	
	r.providers = make(map[string]Provider)
	return lastErr
}