package o2providers

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MockProvider provides a mock implementation of the Provider interface
// for testing and development purposes.
type MockProvider struct {
	mu          sync.RWMutex
	info        ProviderInfo
	config      ProviderConfig
	resources   map[string]*Resource
	initialized bool
	closed      bool
}

// NewMockProvider creates a new mock provider instance
func NewMockProvider(name string) *MockProvider {
	return &MockProvider{
		info: ProviderInfo{
			Name:        name,
			Type:        "mock",
			Version:     "1.0.0",
			Description: "Mock provider for testing",
			Vendor:      "Nephoran",
			Tags: map[string]string{
				"type": "mock",
			},
			LastUpdated: time.Now(),
		},
		resources: make(map[string]*Resource),
	}
}

// GetInfo returns metadata about this provider implementation
func (m *MockProvider) GetInfo() ProviderInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.info
}

// Initialize configures the provider with given configuration
func (m *MockProvider) Initialize(ctx context.Context, config ProviderConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.initialized {
		return fmt.Errorf("provider already initialized")
	}

	if m.closed {
		return fmt.Errorf("provider is closed")
	}

	m.config = config
	m.initialized = true

	return nil
}

// CreateResource creates a new resource
func (m *MockProvider) CreateResource(ctx context.Context, req ResourceRequest) (*Resource, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.initialized {
		return nil, fmt.Errorf("provider not initialized")
	}

	if m.closed {
		return nil, fmt.Errorf("provider is closed")
	}

	// Generate a simple ID
	id := fmt.Sprintf("%s-%d", req.Name, time.Now().Unix())

	resource := &Resource{
		ID:        id,
		Name:      req.Name,
		Type:      req.Type,
		Status:    StatusCreating,
		Spec:      req.Spec,
		Labels:    req.Labels,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	m.resources[id] = resource

	// Simulate async creation
	go func() {
		time.Sleep(100 * time.Millisecond)
		m.mu.Lock()
		defer m.mu.Unlock()
		if res, exists := m.resources[id]; exists {
			res.Status = StatusReady
			res.UpdatedAt = time.Now()
		}
	}()

	return resource, nil
}

// GetResource retrieves a resource by ID
func (m *MockProvider) GetResource(ctx context.Context, id string) (*Resource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.initialized {
		return nil, fmt.Errorf("provider not initialized")
	}

	resource, exists := m.resources[id]
	if !exists {
		return nil, fmt.Errorf("resource %s not found", id)
	}

	// Return a copy to avoid external modifications
	resourceCopy := *resource
	return &resourceCopy, nil
}

// ListResources returns resources matching the filter criteria
func (m *MockProvider) ListResources(ctx context.Context, filter ResourceFilter) ([]*Resource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.initialized {
		return nil, fmt.Errorf("provider not initialized")
	}

	var results []*Resource

	for _, resource := range m.resources {
		if m.matchesFilter(resource, filter) {
			// Return a copy to avoid external modifications
			resourceCopy := *resource
			results = append(results, &resourceCopy)
		}
	}

	return results, nil
}

// UpdateResource updates an existing resource
func (m *MockProvider) UpdateResource(ctx context.Context, id string, req ResourceRequest) (*Resource, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.initialized {
		return nil, fmt.Errorf("provider not initialized")
	}

	resource, exists := m.resources[id]
	if !exists {
		return nil, fmt.Errorf("resource %s not found", id)
	}

	// Update resource fields
	resource.Name = req.Name
	resource.Spec = req.Spec
	resource.Labels = req.Labels
	resource.Status = StatusUpdating
	resource.UpdatedAt = time.Now()

	// Simulate async update
	go func() {
		time.Sleep(50 * time.Millisecond)
		m.mu.Lock()
		defer m.mu.Unlock()
		if res, exists := m.resources[id]; exists {
			res.Status = StatusReady
			res.UpdatedAt = time.Now()
		}
	}()

	// Return a copy
	resourceCopy := *resource
	return &resourceCopy, nil
}

// DeleteResource removes a resource
func (m *MockProvider) DeleteResource(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.initialized {
		return fmt.Errorf("provider not initialized")
	}

	resource, exists := m.resources[id]
	if !exists {
		return fmt.Errorf("resource %s not found", id)
	}

	resource.Status = StatusDeleting
	resource.UpdatedAt = time.Now()

	// Simulate async deletion
	go func() {
		time.Sleep(50 * time.Millisecond)
		m.mu.Lock()
		defer m.mu.Unlock()
		delete(m.resources, id)
	}()

	return nil
}

// GetResourceStatus returns the current status of a resource
func (m *MockProvider) GetResourceStatus(ctx context.Context, id string) (ResourceStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.initialized {
		return StatusUnknown, fmt.Errorf("provider not initialized")
	}

	resource, exists := m.resources[id]
	if !exists {
		return StatusUnknown, fmt.Errorf("resource %s not found", id)
	}

	return resource.Status, nil
}

// Close cleans up provider resources
func (m *MockProvider) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil // Already closed
	}

	// Clear resources
	m.resources = make(map[string]*Resource)
	m.initialized = false
	m.closed = true

	return nil
}

// matchesFilter checks if a resource matches the given filter
func (m *MockProvider) matchesFilter(resource *Resource, filter ResourceFilter) bool {
	// Name filter
	if len(filter.Names) > 0 {
		found := false
		for _, name := range filter.Names {
			if resource.Name == name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Type filter
	if len(filter.Types) > 0 {
		found := false
		for _, filterType := range filter.Types {
			if string(resource.Type) == filterType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Status filter
	if len(filter.Statuses) > 0 {
		found := false
		for _, filterStatus := range filter.Statuses {
			if string(resource.Status) == filterStatus {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Labels filter
	if filter.Labels != nil {
		for key, value := range filter.Labels {
			if resource.Labels == nil {
				return false
			}
			if labelValue, exists := resource.Labels[key]; !exists || labelValue != value {
				return false
			}
		}
	}

	return true
}

// Helper function to check string prefix
func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}

// MockProviderConstructor is a constructor function for mock providers
func MockProviderConstructor(config ProviderConfig) (Provider, error) {
	name := "mock-provider"
	if nameVal, exists := config.Config["name"]; exists {
		if nameStr, ok := nameVal.(string); ok {
			name = nameStr
		}
	}

	provider := NewMockProvider(name)

	// Initialize with the provided config
	ctx := context.Background()
	if err := provider.Initialize(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to initialize mock provider: %w", err)
	}

	return provider, nil
}

// GetMockProviderSchema returns the configuration schema for mock provider
func GetMockProviderSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Name of the mock provider instance",
				"default":     "mock-provider",
			},
			"simulateLatency": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether to simulate network latency",
				"default":     false,
			},
			"errorRate": map[string]interface{}{
				"type":        "number",
				"description": "Simulated error rate (0.0 to 1.0)",
				"default":     0.0,
				"minimum":     0.0,
				"maximum":     1.0,
			},
		},
	}
}

// init function to register mock provider with global factory
func init() {
	RegisterGlobalProvider("mock", MockProviderConstructor, GetMockProviderSchema())
}
