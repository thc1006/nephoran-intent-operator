package rag

import (
	"context"
	"fmt"
	"sync"
)

// SharedResource represents a shared resource in the RAG system
type SharedResource struct {
	ID       string
	Name     string
	Type     string
	Data     interface{}
	metadata map[string]interface{}
	mu       sync.RWMutex
}

// NewSharedResource creates a new shared resource
func NewSharedResource(id, name, resourceType string) *SharedResource {
	return &SharedResource{
		ID:       id,
		Name:     name,
		Type:     resourceType,
		metadata: make(map[string]interface{}),
	}
}

// SetData sets the data for the shared resource
func (sr *SharedResource) SetData(data interface{}) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.Data = data
}

// GetData gets the data from the shared resource
func (sr *SharedResource) GetData() interface{} {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	return sr.Data
}

// SetMetadata sets metadata for the shared resource
func (sr *SharedResource) SetMetadata(key string, value interface{}) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.metadata[key] = value
}

// GetMetadata gets metadata from the shared resource
func (sr *SharedResource) GetMetadata(key string) (interface{}, bool) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	value, exists := sr.metadata[key]
	return value, exists
}

// SharedResourceManager manages shared resources
type SharedResourceManager struct {
	resources map[string]*SharedResource
	mu        sync.RWMutex
}

// NewSharedResourceManager creates a new shared resource manager
func NewSharedResourceManager() *SharedResourceManager {
	return &SharedResourceManager{
		resources: make(map[string]*SharedResource),
	}
}

// AddResource adds a shared resource
func (srm *SharedResourceManager) AddResource(resource *SharedResource) error {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	if _, exists := srm.resources[resource.ID]; exists {
		return fmt.Errorf("resource with ID %s already exists", resource.ID)
	}

	srm.resources[resource.ID] = resource
	return nil
}

// GetResource gets a shared resource by ID
func (srm *SharedResourceManager) GetResource(id string) (*SharedResource, error) {
	srm.mu.RLock()
	defer srm.mu.RUnlock()

	resource, exists := srm.resources[id]
	if !exists {
		return nil, fmt.Errorf("resource with ID %s not found", id)
	}

	return resource, nil
}

// RemoveResource removes a shared resource
func (srm *SharedResourceManager) RemoveResource(id string) error {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	if _, exists := srm.resources[id]; !exists {
		return fmt.Errorf("resource with ID %s not found", id)
	}

	delete(srm.resources, id)
	return nil
}

// ListResources returns all shared resources
func (srm *SharedResourceManager) ListResources() []*SharedResource {
	srm.mu.RLock()
	defer srm.mu.RUnlock()

	resources := make([]*SharedResource, 0, len(srm.resources))
	for _, resource := range srm.resources {
		resources = append(resources, resource)
	}

	return resources
}

// SharedContext provides shared context for RAG operations
type SharedContext struct {
	ctx     context.Context
	manager *SharedResourceManager
}

// NewSharedContext creates a new shared context
func NewSharedContext(ctx context.Context) *SharedContext {
	return &SharedContext{
		ctx:     ctx,
		manager: NewSharedResourceManager(),
	}
}

// Context returns the underlying context
func (sc *SharedContext) Context() context.Context {
	return sc.ctx
}

// ResourceManager returns the shared resource manager
func (sc *SharedContext) ResourceManager() *SharedResourceManager {
	return sc.manager
}
