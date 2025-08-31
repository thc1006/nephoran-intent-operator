package providers

import (
	"context"
	"fmt"
	"sync"
)

// ProviderRegistry manages cloud provider registrations
type ProviderRegistry struct {
	providers map[string]CloudProvider
	mu        sync.RWMutex
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]CloudProvider),
	}
}

// Register registers a cloud provider
func (r *ProviderRegistry) Register(name string, provider CloudProvider) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[name] = provider
	return nil
}

// Get retrieves a registered provider
func (r *ProviderRegistry) Get(name string) (CloudProvider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	provider, exists := r.providers[name]
	return provider, exists
}

// List returns all registered providers
func (r *ProviderRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// CloudProvider defines the interface for cloud providers
type CloudProvider interface {
	Deploy(ctx context.Context, spec interface{}) error
	Scale(ctx context.Context, target string, replicas int) error
	Delete(ctx context.Context, target string) error
	GetStatus(ctx context.Context, target string) (interface{}, error)
}

// ProviderFactory creates cloud provider instances
type ProviderFactory struct {
	registry *ProviderRegistry
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory(registry *ProviderRegistry) *ProviderFactory {
	return &ProviderFactory{
		registry: registry,
	}
}

// CreateProvider creates a provider instance
func (f *ProviderFactory) CreateProvider(name string) (CloudProvider, error) {
	provider, exists := f.registry.Get(name)
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}
	return provider, nil
}

// Rest of the existing ProviderRegistry implementation 
// (all methods remain exactly the same as in the previous version)

// ProviderSelectionCriteria defines criteria for selecting a provider.
type ProviderSelectionCriteria struct {
	Type string // Provider type (kubernetes, openstack, aws, etc.)

	Region string // Preferred region

	Zone string // Preferred zone

	RequireHealthy bool // Only select healthy providers

	RequiredCapabilities []string // Required capabilities

	SelectionStrategy string // Selection strategy (random, least-loaded, round-robin)
}

// Note: Remove the DefaultProviderFactory and its methods 
// These are now defined in factory.go