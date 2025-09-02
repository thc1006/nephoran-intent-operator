package providers

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

// GetProviderInfo returns metadata about this provider implementation (CloudProvider interface)
func (m *MockProvider) GetProviderInfo() *ProviderInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return &m.info
}

// GetSupportedResourceTypes returns supported resource types
func (m *MockProvider) GetSupportedResourceTypes() []string {
	return []string{"deployment", "service", "configmap", "secret", "pvc"}
}

// GetCapabilities returns provider capabilities
func (m *MockProvider) GetCapabilities() *ProviderCapabilities {
	return &ProviderCapabilities{
		ComputeTypes:             []string{"deployment", "pod"},
		StorageTypes:             []string{"pvc", "pv"},
		NetworkTypes:             []string{"service", "ingress"},
		AcceleratorTypes:         []string{},
		AutoScaling:              false,
		LoadBalancing:            false,
		Monitoring:               true,
		Logging:                  true,
		Networking:               true,
		StorageClasses:           false,
		HorizontalPodAutoscaling: false,
		VerticalPodAutoscaling:   false,
	}
}

// Connect establishes connection to the provider
func (m *MockProvider) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return fmt.Errorf("provider is closed")
	}

	// Mock connection - always succeeds
	return nil
}

// Disconnect disconnects from the provider
func (m *MockProvider) Disconnect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return fmt.Errorf("provider is closed")
	}

	// Mock disconnection - always succeeds
	return nil
}

// HealthCheck performs a health check
func (m *MockProvider) HealthCheck(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return fmt.Errorf("provider is closed")
	}

	if !m.initialized {
		return fmt.Errorf("provider not initialized")
	}

	// Mock health check - always healthy
	return nil
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

// CreateNetworkService creates a network service
func (m *MockProvider) CreateNetworkService(ctx context.Context, req *NetworkServiceRequest) (*NetworkServiceResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.initialized {
		return nil, fmt.Errorf("provider not initialized")
	}

	// Mock implementation
	response := &NetworkServiceResponse{
		ID:     fmt.Sprintf("net-%d", time.Now().UnixNano()),
		Name:   req.Name,
		Type:   req.Type, // Use Type instead of ServiceType
		Status: "active",
	}

	return response, nil
}

// IsConnected checks if provider is connected
func (m *MockProvider) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.initialized && !m.closed
}

// CreateResource creates a new resource (CloudProvider interface)
func (m *MockProvider) CreateResource(ctx context.Context, req *CreateResourceRequest) (*ResourceResponse, error) {
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

	response := &ResourceResponse{
		ID:            id,
		Name:          req.Name,
		Type:          req.Type,
		Namespace:     req.Namespace,
		Status:        "creating",
		Health:        "healthy",
		Specification: req.Specification,
		Labels:        req.Labels,
		Annotations:   req.Annotations,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Simulate async creation
	go func() {
		time.Sleep(100 * time.Millisecond)
		m.mu.Lock()
		defer m.mu.Unlock()
		// In a real implementation, you would update the resource status in storage
		response.Status = "active"
		response.UpdatedAt = time.Now()
	}()

	return response, nil
}

// GetResource retrieves a resource by ID (CloudProvider interface)
func (m *MockProvider) GetResource(ctx context.Context, resourceID string) (*ResourceResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.initialized {
		return nil, fmt.Errorf("provider not initialized")
	}

	// Mock implementation - simulate resource lookup
	response := &ResourceResponse{
		ID:        resourceID,
		Name:      "mock-" + resourceID,
		Type:      "mock-resource",
		Status:    "active",
		Health:    "healthy",
		CreatedAt: time.Now().Add(-time.Hour),
		UpdatedAt: time.Now(),
	}

	return response, nil
}

// ListResources returns resources matching the filter criteria (CloudProvider interface)
func (m *MockProvider) ListResources(ctx context.Context, filter *ResourceFilter) ([]*ResourceResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.initialized {
		return nil, fmt.Errorf("provider not initialized")
	}

	// Mock implementation - return some sample resources
	var results []*ResourceResponse
	for i := 0; i < 3; i++ {
		resource := &ResourceResponse{
			ID:        fmt.Sprintf("mock-resource-%d", i),
			Name:      fmt.Sprintf("mock-resource-%d", i),
			Type:      "mock-resource",
			Status:    "active",
			Health:    "healthy",
			CreatedAt: time.Now().Add(-time.Hour),
			UpdatedAt: time.Now(),
		}
		results = append(results, resource)
	}

	return results, nil
}

// UpdateResource updates an existing resource (CloudProvider interface)
func (m *MockProvider) UpdateResource(ctx context.Context, resourceID string, req *UpdateResourceRequest) (*ResourceResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.initialized {
		return nil, fmt.Errorf("provider not initialized")
	}

	// Mock implementation
	response := &ResourceResponse{
		ID:            resourceID,
		Name:          "updated-" + resourceID,
		Status:        "updating",
		Health:        "healthy",
		Specification: req.Specification,
		Labels:        req.Labels,
		Annotations:   req.Annotations,
		UpdatedAt:     time.Now(),
	}

	// Simulate async update
	go func() {
		time.Sleep(50 * time.Millisecond)
		m.mu.Lock()
		defer m.mu.Unlock()
		response.Status = "active"
		response.UpdatedAt = time.Now()
	}()

	return response, nil
}

// Deploy creates a deployment (CloudProvider interface)
func (m *MockProvider) Deploy(ctx context.Context, req *DeploymentRequest) (*DeploymentResponse, error) {
	// Mock implementation
	return &DeploymentResponse{
		ID:     fmt.Sprintf("deploy-%d", time.Now().Unix()),
		Name:   req.Name,
		Status: "deployed",
	}, nil
}

// GetDeployment retrieves a deployment (CloudProvider interface)
func (m *MockProvider) GetDeployment(ctx context.Context, deploymentID string) (*DeploymentResponse, error) {
	// Mock implementation
	return &DeploymentResponse{
		ID:     deploymentID,
		Name:   "mock-deployment",
		Status: "deployed",
	}, nil
}

// UpdateDeployment updates a deployment (CloudProvider interface)
func (m *MockProvider) UpdateDeployment(ctx context.Context, deploymentID string, req *UpdateDeploymentRequest) (*DeploymentResponse, error) {
	// Mock implementation
	return &DeploymentResponse{
		ID:     deploymentID,
		Name:   "updated-deployment",
		Status: "deployed",
	}, nil
}

// DeleteDeployment removes a deployment (CloudProvider interface)
func (m *MockProvider) DeleteDeployment(ctx context.Context, deploymentID string) error {
	// Mock implementation
	return nil
}

// ListDeployments lists deployments (CloudProvider interface)
func (m *MockProvider) ListDeployments(ctx context.Context, filter *DeploymentFilter) ([]*DeploymentResponse, error) {
	// Mock implementation
	return []*DeploymentResponse{
		{ID: "deploy-1", Name: "mock-deployment-1", Status: "deployed"},
		{ID: "deploy-2", Name: "mock-deployment-2", Status: "deployed"},
	}, nil
}

// ScaleResource scales a resource (CloudProvider interface)
func (m *MockProvider) ScaleResource(ctx context.Context, resourceID string, req *ScaleRequest) error {
	// Mock implementation
	return nil
}

// GetScalingCapabilities returns scaling capabilities (CloudProvider interface)
func (m *MockProvider) GetScalingCapabilities(ctx context.Context, resourceID string) (*ScalingCapabilities, error) {
	// Mock implementation
	return &ScalingCapabilities{
		MinReplicas: 1,
		MaxReplicas: 100,
	}, nil
}

// GetMetrics returns provider metrics (CloudProvider interface)
func (m *MockProvider) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	// Mock implementation
	return json.RawMessage(`{}`), nil
}

// GetResourceMetrics returns resource-specific metrics (CloudProvider interface)
func (m *MockProvider) GetResourceMetrics(ctx context.Context, resourceID string) (map[string]interface{}, error) {
	// Mock implementation
	return json.RawMessage(`{}`), nil
}

// GetResourceHealth returns resource health (CloudProvider interface)
func (m *MockProvider) GetResourceHealth(ctx context.Context, resourceID string) (*HealthStatus, error) {
	// Mock implementation
	return &HealthStatus{
		Status:    "healthy",
		Message:   "Resource is running normally",
		Timestamp: time.Now(),
	}, nil
}

// GetNetworkService gets a network service (CloudProvider interface)
func (m *MockProvider) GetNetworkService(ctx context.Context, serviceID string) (*NetworkServiceResponse, error) {
	// Mock implementation
	return &NetworkServiceResponse{
		ID:     serviceID,
		Name:   "mock-service",
		Type:   "service",
		Status: "active",
	}, nil
}

// DeleteNetworkService deletes a network service (CloudProvider interface)
func (m *MockProvider) DeleteNetworkService(ctx context.Context, serviceID string) error {
	// Mock implementation
	return nil
}

// ListNetworkServices lists network services (CloudProvider interface)
func (m *MockProvider) ListNetworkServices(ctx context.Context, filter *NetworkServiceFilter) ([]*NetworkServiceResponse, error) {
	// Mock implementation
	return []*NetworkServiceResponse{
		{ID: "svc-1", Name: "mock-service-1", Type: "service", Status: "active"},
	}, nil
}

// CreateStorageResource creates a storage resource (CloudProvider interface)
func (m *MockProvider) CreateStorageResource(ctx context.Context, req *StorageResourceRequest) (*StorageResourceResponse, error) {
	// Mock implementation
	return &StorageResourceResponse{
		ID:     fmt.Sprintf("storage-%d", time.Now().Unix()),
		Name:   req.Name,
		Type:   req.Type,
		Status: "ready",
	}, nil
}

// GetStorageResource gets a storage resource (CloudProvider interface)
func (m *MockProvider) GetStorageResource(ctx context.Context, resourceID string) (*StorageResourceResponse, error) {
	// Mock implementation
	return &StorageResourceResponse{
		ID:     resourceID,
		Name:   "mock-storage",
		Type:   "pvc",
		Status: "ready",
	}, nil
}

// DeleteStorageResource deletes a storage resource (CloudProvider interface)
func (m *MockProvider) DeleteStorageResource(ctx context.Context, resourceID string) error {
	// Mock implementation
	return nil
}

// ListStorageResources lists storage resources (CloudProvider interface)
func (m *MockProvider) ListStorageResources(ctx context.Context, filter *StorageResourceFilter) ([]*StorageResourceResponse, error) {
	// Mock implementation
	return []*StorageResourceResponse{
		{ID: "storage-1", Name: "mock-storage-1", Type: "pvc", Status: "ready"},
	}, nil
}

// SubscribeToEvents subscribes to events (CloudProvider interface)
func (m *MockProvider) SubscribeToEvents(ctx context.Context, callback EventCallback) error {
	// Mock implementation
	return nil
}

// UnsubscribeFromEvents unsubscribes from events (CloudProvider interface)
func (m *MockProvider) UnsubscribeFromEvents(ctx context.Context) error {
	// Mock implementation
	return nil
}

// GetConfiguration gets provider configuration (CloudProvider interface)
func (m *MockProvider) GetConfiguration(ctx context.Context) (*ProviderConfiguration, error) {
	// Mock implementation
	return &ProviderConfiguration{
		Name:    "mock-provider",
		Type:    "mock",
		Enabled: true,
	}, nil
}

// ValidateConfiguration validates provider configuration (CloudProvider interface)
func (m *MockProvider) ValidateConfiguration(ctx context.Context, config *ProviderConfiguration) error {
	// Mock implementation
	return nil
}

// DeleteResource removes a resource (CloudProvider interface)
func (m *MockProvider) DeleteResource(ctx context.Context, resourceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.initialized {
		return fmt.Errorf("provider not initialized")
	}

	// Mock implementation - simulate deletion
	// In a real implementation, this would actually delete the resource
	return nil
}

// ApplyConfiguration applies provider configuration
func (m *MockProvider) ApplyConfiguration(ctx context.Context, config *ProviderConfiguration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return fmt.Errorf("provider is closed")
	}

	// Mock implementation - just store the config if needed
	return nil
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

// Helper function to check string prefix
func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}

// ProviderAdapter adapts CloudProvider to implement the old Provider interface
type ProviderAdapter struct {
	cloudProvider CloudProvider
}

// NewProviderAdapter creates a new adapter
func NewProviderAdapter(cloudProvider CloudProvider) Provider {
	return &ProviderAdapter{cloudProvider: cloudProvider}
}

// GetInfo returns provider info (Provider interface)
func (a *ProviderAdapter) GetInfo() ProviderInfo {
	info := a.cloudProvider.GetProviderInfo()
	return ProviderInfo{
		Name:        info.Name,
		Type:        info.Type,
		Version:     info.Version,
		Description: info.Description,
		Vendor:      info.Vendor,
		Tags:        map[string]string{"type": info.Type},
		LastUpdated: time.Now(),
	}
}

// Initialize configures the provider (Provider interface)
func (a *ProviderAdapter) Initialize(ctx context.Context, config ProviderConfig) error {
	// Delegate to CloudProvider's ApplyConfiguration
	providerConfig := &ProviderConfiguration{
		Name:    "mock-adapter",
		Type:    "mock",
		Enabled: true,
	}
	return a.cloudProvider.ApplyConfiguration(ctx, providerConfig)
}

// CreateResource creates a new resource (Provider interface)
func (a *ProviderAdapter) CreateResource(ctx context.Context, req ResourceRequest) (*Resource, error) {
	// Convert to CloudProvider request format
	cloudReq := &CreateResourceRequest{
		Name:          req.Name,
		Type:          string(req.Type),
		Specification: json.RawMessage(`{}`),
		Labels:        req.Labels,
	}

	response, err := a.cloudProvider.CreateResource(ctx, cloudReq)
	if err != nil {
		return nil, err
	}

	// Convert back to Provider response format
	return &Resource{
		ID:        response.ID,
		Name:      response.Name,
		Type:      ResourceType(response.Type),
		Status:    ResourceStatus(response.Status),
		Spec:      response.Specification,
		Labels:    response.Labels,
		CreatedAt: response.CreatedAt,
		UpdatedAt: response.UpdatedAt,
	}, nil
}

// GetResource retrieves a resource by ID (Provider interface)
func (a *ProviderAdapter) GetResource(ctx context.Context, id string) (*Resource, error) {
	response, err := a.cloudProvider.GetResource(ctx, id)
	if err != nil {
		return nil, err
	}

	return &Resource{
		ID:        response.ID,
		Name:      response.Name,
		Type:      ResourceType(response.Type),
		Status:    ResourceStatus(response.Status),
		Spec:      response.Specification,
		Labels:    response.Labels,
		CreatedAt: response.CreatedAt,
		UpdatedAt: response.UpdatedAt,
	}, nil
}

// ListResources returns resources matching the filter criteria (Provider interface)
func (a *ProviderAdapter) ListResources(ctx context.Context, filter ResourceFilter) ([]*Resource, error) {
	// Convert to CloudProvider filter format
	cloudFilter := &ResourceFilter{
		Names:  filter.Names,
		Types:  filter.Types,
		Labels: filter.Labels,
	}

	responses, err := a.cloudProvider.ListResources(ctx, cloudFilter)
	if err != nil {
		return nil, err
	}

	// Convert to Provider format
	var resources []*Resource
	for _, response := range responses {
		resource := &Resource{
			ID:        response.ID,
			Name:      response.Name,
			Type:      ResourceType(response.Type),
			Status:    ResourceStatus(response.Status),
			Spec:      response.Specification,
			Labels:    response.Labels,
			CreatedAt: response.CreatedAt,
			UpdatedAt: response.UpdatedAt,
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// UpdateResource updates an existing resource (Provider interface)
func (a *ProviderAdapter) UpdateResource(ctx context.Context, id string, req ResourceRequest) (*Resource, error) {
	// Convert to CloudProvider request format
	cloudReq := &UpdateResourceRequest{
		Specification: json.RawMessage(`{}`),
		Labels:        req.Labels,
	}

	response, err := a.cloudProvider.UpdateResource(ctx, id, cloudReq)
	if err != nil {
		return nil, err
	}

	// Convert back to Provider response format
	return &Resource{
		ID:        response.ID,
		Name:      response.Name,
		Type:      ResourceType(response.Type),
		Status:    ResourceStatus(response.Status),
		Spec:      response.Specification,
		Labels:    response.Labels,
		CreatedAt: response.CreatedAt,
		UpdatedAt: response.UpdatedAt,
	}, nil
}

// DeleteResource removes a resource (Provider interface)
func (a *ProviderAdapter) DeleteResource(ctx context.Context, id string) error {
	return a.cloudProvider.DeleteResource(ctx, id)
}

// GetResourceStatus returns the current status of a resource (Provider interface)
func (a *ProviderAdapter) GetResourceStatus(ctx context.Context, id string) (ResourceStatus, error) {
	health, err := a.cloudProvider.GetResourceHealth(ctx, id)
	if err != nil {
		return StatusUnknown, err
	}

	// Convert health status to resource status
	switch health.Status {
	case "healthy":
		return StatusReady, nil
	case "unhealthy":
		return StatusError, nil
	default:
		return StatusUnknown, nil
	}
}

// Close cleans up provider resources (Provider interface)
func (a *ProviderAdapter) Close() error {
	return a.cloudProvider.Close()
}

// MockProviderConstructor is a constructor function for mock providers
func MockProviderConstructor(config ProviderConfig) (Provider, error) {
	name := "mock-provider"
	if nameVal, exists := config.Config["name"]; exists {
		if nameStr, ok := nameVal.(string); ok {
			name = nameStr
		}
	}

	cloudProvider := NewMockProvider(name)

	// Initialize with the provided config
	ctx := context.Background()
	if err := cloudProvider.Initialize(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to initialize mock provider: %w", err)
	}

	// Return the adapter to implement Provider interface
	return NewProviderAdapter(cloudProvider), nil
}

// GetMockProviderSchema returns the configuration schema for mock provider
func GetMockProviderSchema() map[string]interface{} {
	return map[string]interface{}{
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type": "string",
				"description": "Mock provider name",
			},
			"simulateLatency": map[string]interface{}{
				"type": "boolean",
				"description": "Whether to simulate latency",
			},
			"errorRate": map[string]interface{}{
				"type": "number",
				"description": "Error rate percentage",
			},
		},
	}
}

// init function to register mock provider with global factory
func init() {
	RegisterGlobalProvider("mock", MockProviderConstructor, GetMockProviderSchema())
}

