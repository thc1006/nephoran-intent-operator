package o2providers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ProviderRegistry manages cloud provider instances
type ProviderRegistry struct {
	providers           map[string]CloudProvider
	configurations      map[string]*ProviderConfiguration
	healthStatus        map[string]*HealthStatus
	metrics             map[string]map[string]interface{}
	mu                  sync.RWMutex
	stopCh              chan struct{}
	healthCheckInterval time.Duration
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers:           make(map[string]CloudProvider),
		configurations:      make(map[string]*ProviderConfiguration),
		healthStatus:        make(map[string]*HealthStatus),
		metrics:             make(map[string]map[string]interface{}),
		stopCh:              make(chan struct{}),
		healthCheckInterval: 30 * time.Second,
	}
}

// RegisterProvider registers a new cloud provider
func (r *ProviderRegistry) RegisterProvider(name string, provider CloudProvider, config *ProviderConfiguration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider %s already registered", name)
	}

	// Validate configuration if provided
	if config != nil {
		if err := provider.ValidateConfiguration(context.Background(), config); err != nil {
			return fmt.Errorf("invalid configuration for provider %s: %w", name, err)
		}
		r.configurations[name] = config
	}

	r.providers[name] = provider
	r.healthStatus[name] = &HealthStatus{
		Status:      HealthStatusUnknown,
		Message:     "Provider registered but not yet connected",
		LastUpdated: time.Now(),
	}

	return nil
}

// UnregisterProvider removes a provider from the registry
func (r *ProviderRegistry) UnregisterProvider(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	provider, exists := r.providers[name]
	if !exists {
		return fmt.Errorf("provider %s not found", name)
	}

	// Disconnect and close the provider
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := provider.Disconnect(ctx); err != nil {
		// Log error but continue with unregistration
		log.FromContext(ctx).Error(err, "failed to disconnect provider", "provider", name)
	}

	if err := provider.Close(); err != nil {
		log.FromContext(ctx).Error(err, "failed to close provider", "provider", name)
	}

	delete(r.providers, name)
	delete(r.configurations, name)
	delete(r.healthStatus, name)
	delete(r.metrics, name)

	return nil
}

// GetProvider retrieves a registered provider
func (r *ProviderRegistry) GetProvider(name string) (CloudProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}

	return provider, nil
}

// GetProviderByType retrieves the first provider of a specific type
func (r *ProviderRegistry) GetProviderByType(providerType string) (CloudProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, provider := range r.providers {
		info := provider.GetProviderInfo()
		if info.Type == providerType {
			return provider, nil
		}
	}

	return nil, fmt.Errorf("no provider of type %s found", providerType)
}

// ListProviders returns all registered provider names
func (r *ProviderRegistry) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}

	return names
}

// GetSupportedProviders returns all supported provider types
func (r *ProviderRegistry) GetSupportedProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var providerTypes []string
	for _, provider := range r.providers {
		info := provider.GetProviderInfo()
		providerTypes = append(providerTypes, info.Type)
	}

	return providerTypes
}

// ListProvidersByType returns providers of a specific type
func (r *ProviderRegistry) ListProvidersByType(providerType string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var names []string
	for name, provider := range r.providers {
		info := provider.GetProviderInfo()
		if info.Type == providerType {
			names = append(names, name)
		}
	}

	return names
}

// GetProviderInfo returns information about a provider
func (r *ProviderRegistry) GetProviderInfo(name string) (*ProviderInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}

	return provider.GetProviderInfo(), nil
}

// GetProviderHealth returns the health status of a provider
func (r *ProviderRegistry) GetProviderHealth(name string) (*HealthStatus, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	health, exists := r.healthStatus[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}

	return health, nil
}

// GetAllProviderHealth returns health status for all providers
func (r *ProviderRegistry) GetAllProviderHealth() map[string]*HealthStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*HealthStatus)
	for name, health := range r.healthStatus {
		result[name] = health
	}

	return result
}

// GetProviderMetrics returns metrics for a provider
func (r *ProviderRegistry) GetProviderMetrics(name string) (map[string]interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metrics, exists := r.metrics[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}

	return metrics, nil
}

// ConnectAll connects all registered providers
func (r *ProviderRegistry) ConnectAll(ctx context.Context) error {
	r.mu.RLock()
	providers := make(map[string]CloudProvider)
	for name, provider := range r.providers {
		providers[name] = provider
	}
	r.mu.RUnlock()

	var firstError error
	for name, provider := range providers {
		if err := r.connectProvider(ctx, name, provider); err != nil {
			if firstError == nil {
				firstError = err
			}
			log.FromContext(ctx).Error(err, "failed to connect provider", "provider", name)
		}
	}

	return firstError
}

// ConnectProvider connects a specific provider
func (r *ProviderRegistry) ConnectProvider(ctx context.Context, name string) error {
	provider, err := r.GetProvider(name)
	if err != nil {
		return err
	}

	return r.connectProvider(ctx, name, provider)
}

// connectProvider handles the actual connection logic
func (r *ProviderRegistry) connectProvider(ctx context.Context, name string, provider CloudProvider) error {
	// Apply configuration if available
	r.mu.RLock()
	config := r.configurations[name]
	r.mu.RUnlock()

	if config != nil {
		if err := provider.ApplyConfiguration(ctx, config); err != nil {
			r.updateHealthStatus(name, &HealthStatus{
				Status:      HealthStatusUnhealthy,
				Message:     fmt.Sprintf("Failed to apply configuration: %v", err),
				LastUpdated: time.Now(),
			})
			return fmt.Errorf("failed to apply configuration for provider %s: %w", name, err)
		}
	}

	// Connect to the provider
	if err := provider.Connect(ctx); err != nil {
		r.updateHealthStatus(name, &HealthStatus{
			Status:      HealthStatusUnhealthy,
			Message:     fmt.Sprintf("Failed to connect: %v", err),
			LastUpdated: time.Now(),
		})
		return fmt.Errorf("failed to connect provider %s: %w", name, err)
	}

	// Perform initial health check
	if err := provider.HealthCheck(ctx); err != nil {
		r.updateHealthStatus(name, &HealthStatus{
			Status:      HealthStatusUnhealthy,
			Message:     fmt.Sprintf("Health check failed: %v", err),
			LastUpdated: time.Now(),
		})
		// Don't return error here as provider is connected
		log.FromContext(ctx).Error(err, "provider health check failed", "provider", name)
	} else {
		r.updateHealthStatus(name, &HealthStatus{
			Status:      HealthStatusHealthy,
			Message:     "Provider connected and healthy",
			LastUpdated: time.Now(),
		})
	}

	return nil
}

// DisconnectAll disconnects all providers
func (r *ProviderRegistry) DisconnectAll(ctx context.Context) error {
	r.mu.RLock()
	providers := make(map[string]CloudProvider)
	for name, provider := range r.providers {
		providers[name] = provider
	}
	r.mu.RUnlock()

	var firstError error
	for name, provider := range providers {
		if err := provider.Disconnect(ctx); err != nil {
			if firstError == nil {
				firstError = err
			}
			log.FromContext(ctx).Error(err, "failed to disconnect provider", "provider", name)
		}

		r.updateHealthStatus(name, &HealthStatus{
			Status:      HealthStatusUnknown,
			Message:     "Provider disconnected",
			LastUpdated: time.Now(),
		})
	}

	return firstError
}

// StartHealthChecks starts periodic health checks for all providers
func (r *ProviderRegistry) StartHealthChecks(ctx context.Context) {
	go wait.Until(func() {
		r.performHealthChecks(ctx)
	}, r.healthCheckInterval, r.stopCh)
}

// StopHealthChecks stops periodic health checks
func (r *ProviderRegistry) StopHealthChecks() {
	close(r.stopCh)
}

// performHealthChecks performs health checks on all providers
func (r *ProviderRegistry) performHealthChecks(ctx context.Context) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("performing health checks on all providers")

	r.mu.RLock()
	providers := make(map[string]CloudProvider)
	for name, provider := range r.providers {
		providers[name] = provider
	}
	r.mu.RUnlock()

	for name, provider := range providers {
		healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err := provider.HealthCheck(healthCtx)
		cancel()

		if err != nil {
			logger.Error(err, "provider health check failed", "provider", name)
			r.updateHealthStatus(name, &HealthStatus{
				Status:      HealthStatusUnhealthy,
				Message:     fmt.Sprintf("Health check failed: %v", err),
				LastUpdated: time.Now(),
			})
		} else {
			r.updateHealthStatus(name, &HealthStatus{
				Status:      HealthStatusHealthy,
				Message:     "Provider healthy",
				LastUpdated: time.Now(),
			})
		}

		// Collect metrics
		metricsCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		metrics, err := provider.GetMetrics(metricsCtx)
		cancel()

		if err != nil {
			logger.Error(err, "failed to collect provider metrics", "provider", name)
		} else {
			r.updateMetrics(name, metrics)
		}
	}
}

// updateHealthStatus updates the health status for a provider
func (r *ProviderRegistry) updateHealthStatus(name string, status *HealthStatus) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.healthStatus[name] = status
}

// updateMetrics updates metrics for a provider
func (r *ProviderRegistry) updateMetrics(name string, metrics map[string]interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.metrics[name] = metrics
}

// SelectProvider selects the best provider based on selection criteria
func (r *ProviderRegistry) SelectProvider(ctx context.Context, criteria *ProviderSelectionCriteria) (CloudProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var candidates []string

	// Filter by type if specified
	if criteria.Type != "" {
		for name, provider := range r.providers {
			info := provider.GetProviderInfo()
			if info.Type == criteria.Type {
				candidates = append(candidates, name)
			}
		}
	} else {
		for name := range r.providers {
			candidates = append(candidates, name)
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no providers match selection criteria")
	}

	// Filter by health if required
	if criteria.RequireHealthy {
		var healthyCandidates []string
		for _, name := range candidates {
			if health, exists := r.healthStatus[name]; exists && health.Status == HealthStatusHealthy {
				healthyCandidates = append(healthyCandidates, name)
			}
		}
		candidates = healthyCandidates
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no healthy providers available")
	}

	// Filter by capabilities if specified
	if len(criteria.RequiredCapabilities) > 0 {
		var capableCandidates []string
		for _, name := range candidates {
			provider := r.providers[name]
			capabilities := provider.GetCapabilities()
			if r.hasRequiredCapabilities(capabilities, criteria.RequiredCapabilities) {
				capableCandidates = append(capableCandidates, name)
			}
		}
		candidates = capableCandidates
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no providers with required capabilities")
	}

	// Filter by region if specified
	if criteria.Region != "" {
		var regionalCandidates []string
		for _, name := range candidates {
			provider := r.providers[name]
			info := provider.GetProviderInfo()
			if info.Region == criteria.Region {
				regionalCandidates = append(regionalCandidates, name)
			}
		}
		if len(regionalCandidates) > 0 {
			candidates = regionalCandidates
		}
	}

	// Select based on strategy
	switch criteria.SelectionStrategy {
	case "random":
		// Return random provider from candidates
		return r.providers[candidates[0]], nil // Simplified for now
	case "least-loaded":
		return r.selectLeastLoadedProvider(candidates)
	case "round-robin":
		return r.selectRoundRobinProvider(candidates)
	default:
		// Return first available
		return r.providers[candidates[0]], nil
	}
}

// hasRequiredCapabilities checks if provider has required capabilities
func (r *ProviderRegistry) hasRequiredCapabilities(capabilities *ProviderCapabilities, required []string) bool {
	for _, req := range required {
		switch req {
		case "AutoScaling":
			if !capabilities.AutoScaling {
				return false
			}
		case "LoadBalancing":
			if !capabilities.LoadBalancing {
				return false
			}
		case "Monitoring":
			if !capabilities.Monitoring {
				return false
			}
		case "MultiZone":
			if !capabilities.MultiZone {
				return false
			}
		case "MultiRegion":
			if !capabilities.MultiRegion {
				return false
			}
		case "BackupRestore":
			if !capabilities.BackupRestore {
				return false
			}
		case "Encryption":
			if !capabilities.Encryption {
				return false
			}
		case "SecretManagement":
			if !capabilities.SecretManagement {
				return false
			}
		}
	}
	return true
}

// selectLeastLoadedProvider selects the provider with the lowest load
func (r *ProviderRegistry) selectLeastLoadedProvider(candidates []string) (CloudProvider, error) {
	var selectedName string
	var lowestLoad float64 = 100.0

	for _, name := range candidates {
		if metrics, exists := r.metrics[name]; exists {
			if load, ok := metrics["load_percentage"].(float64); ok && load < lowestLoad {
				lowestLoad = load
				selectedName = name
			}
		}
	}

	if selectedName == "" && len(candidates) > 0 {
		selectedName = candidates[0]
	}

	if selectedName == "" {
		return nil, fmt.Errorf("no provider selected")
	}

	return r.providers[selectedName], nil
}

// selectRoundRobinProvider selects providers in round-robin fashion
func (r *ProviderRegistry) selectRoundRobinProvider(candidates []string) (CloudProvider, error) {
	// Simplified implementation - would need state tracking for true round-robin
	if len(candidates) > 0 {
		return r.providers[candidates[0]], nil
	}
	return nil, fmt.Errorf("no provider selected")
}

// CreateAndRegisterProvider creates a provider using the global factory and registers it
func (r *ProviderRegistry) CreateAndRegisterProvider(name, providerType string, config ProviderConfig) error {
	// Create provider using global factory
	provider, err := CreateGlobalProvider(providerType, config)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	// Initialize the provider if it's not already initialized
	ctx := context.Background()
	// Note: We assume the provider from factory may already be initialized
	// Try to initialize, but don't fail if already initialized
	if err := provider.Initialize(ctx, config); err != nil {
		// Check if the error is because it's already initialized
		if err.Error() != "provider already initialized" {
			return fmt.Errorf("failed to initialize provider: %w", err)
		}
	}

	// Create an adapter to bridge Provider and CloudProvider interfaces
	adapter := &ProviderAdapter{provider: provider}
	return r.RegisterProvider(name, adapter, nil)
}

// Close shuts down the registry and disconnects all providers
func (r *ProviderRegistry) Close() error {
	// Stop health checks
	r.StopHealthChecks()

	// Disconnect all providers
	ctx := context.Background()
	return r.DisconnectAll(ctx)
}

// ProviderAdapter adapts Provider interface to CloudProvider interface
type ProviderAdapter struct {
	provider Provider
}

// GetProviderInfo returns metadata about the provider
func (pa *ProviderAdapter) GetProviderInfo() *ProviderInfo {
	info := pa.provider.GetInfo()
	return &ProviderInfo{
		Name:        info.Name,
		Type:        info.Type,
		Version:     info.Version,
		Description: info.Description,
		Vendor:      info.Vendor,
		Tags:        info.Tags,
		LastUpdated: info.LastUpdated,
	}
}

// GetSupportedResourceTypes returns supported resource types (simplified)
func (pa *ProviderAdapter) GetSupportedResourceTypes() []string {
	return []string{"deployment", "service", "configmap", "secret"}
}

// GetCapabilities returns provider capabilities (simplified)
func (pa *ProviderAdapter) GetCapabilities() *ProviderCapabilities {
	return &ProviderCapabilities{
		AutoScaling: false,
		Monitoring:  false,
		Networking:  false,
	}
}

// Connect establishes connection (no-op for mock)
func (pa *ProviderAdapter) Connect(ctx context.Context) error {
	return nil
}

// Disconnect closes connection (no-op for mock)
func (pa *ProviderAdapter) Disconnect(ctx context.Context) error {
	return nil
}

// HealthCheck performs health check (simplified)
func (pa *ProviderAdapter) HealthCheck(ctx context.Context) error {
	return nil
}

// Close cleans up resources
func (pa *ProviderAdapter) Close() error {
	return pa.provider.Close()
}

// CreateResource creates a resource (adapter method)
func (pa *ProviderAdapter) CreateResource(ctx context.Context, req *CreateResourceRequest) (*ResourceResponse, error) {
	// Convert CreateResourceRequest to ResourceRequest
	resourceReq := ResourceRequest{
		Name:   req.Name,
		Type:   ResourceType(req.Type),
		Spec:   req.Specification,
		Labels: req.Labels,
	}
	
	resource, err := pa.provider.CreateResource(ctx, resourceReq)
	if err != nil {
		return nil, err
	}

	// Convert Resource to ResourceResponse
	return &ResourceResponse{
		ID:            resource.ID,
		Name:          resource.Name,
		Type:          string(resource.Type),
		Status:        string(resource.Status),
		Specification: resource.Spec,
		Labels:        resource.Labels,
		CreatedAt:     resource.CreatedAt,
		UpdatedAt:     resource.UpdatedAt,
	}, nil
}

// GetResource gets a resource (adapter method)
func (pa *ProviderAdapter) GetResource(ctx context.Context, resourceID string) (*ResourceResponse, error) {
	resource, err := pa.provider.GetResource(ctx, resourceID)
	if err != nil {
		return nil, err
	}

	return &ResourceResponse{
		ID:            resource.ID,
		Name:          resource.Name,
		Type:          string(resource.Type),
		Status:        string(resource.Status),
		Specification: resource.Spec,
		Labels:        resource.Labels,
		CreatedAt:     resource.CreatedAt,
		UpdatedAt:     resource.UpdatedAt,
	}, nil
}

// UpdateResource updates a resource (adapter method)
func (pa *ProviderAdapter) UpdateResource(ctx context.Context, resourceID string, req *UpdateResourceRequest) (*ResourceResponse, error) {
	// Convert UpdateResourceRequest to ResourceRequest
	resourceReq := ResourceRequest{
		Spec:   req.Specification,
		Labels: req.Labels,
	}
	
	resource, err := pa.provider.UpdateResource(ctx, resourceID, resourceReq)
	if err != nil {
		return nil, err
	}

	return &ResourceResponse{
		ID:            resource.ID,
		Name:          resource.Name,
		Type:          string(resource.Type),
		Status:        string(resource.Status),
		Specification: resource.Spec,
		Labels:        resource.Labels,
		CreatedAt:     resource.CreatedAt,
		UpdatedAt:     resource.UpdatedAt,
	}, nil
}

// DeleteResource deletes a resource
func (pa *ProviderAdapter) DeleteResource(ctx context.Context, resourceID string) error {
	return pa.provider.DeleteResource(ctx, resourceID)
}

// ListResources lists resources (adapter method)
func (pa *ProviderAdapter) ListResources(ctx context.Context, filter *ResourceFilter) ([]*ResourceResponse, error) {
	resources, err := pa.provider.ListResources(ctx, *filter)
	if err != nil {
		return nil, err
	}

	var responses []*ResourceResponse
	for _, resource := range resources {
		responses = append(responses, &ResourceResponse{
			ID:            resource.ID,
			Name:          resource.Name,
			Type:          string(resource.Type),
			Status:        string(resource.Status),
			Specification: resource.Spec,
			Labels:        resource.Labels,
			CreatedAt:     resource.CreatedAt,
			UpdatedAt:     resource.UpdatedAt,
		})
	}

	return responses, nil
}

// Implement remaining CloudProvider methods with stubs for now
func (pa *ProviderAdapter) Deploy(ctx context.Context, req *DeploymentRequest) (*DeploymentResponse, error) {
	return nil, fmt.Errorf("deploy not implemented in adapter")
}

func (pa *ProviderAdapter) GetDeployment(ctx context.Context, deploymentID string) (*DeploymentResponse, error) {
	return nil, fmt.Errorf("get deployment not implemented in adapter")
}

func (pa *ProviderAdapter) UpdateDeployment(ctx context.Context, deploymentID string, req *UpdateDeploymentRequest) (*DeploymentResponse, error) {
	return nil, fmt.Errorf("update deployment not implemented in adapter")
}

func (pa *ProviderAdapter) DeleteDeployment(ctx context.Context, deploymentID string) error {
	return fmt.Errorf("delete deployment not implemented in adapter")
}

func (pa *ProviderAdapter) ListDeployments(ctx context.Context, filter *DeploymentFilter) ([]*DeploymentResponse, error) {
	return nil, fmt.Errorf("list deployments not implemented in adapter")
}

func (pa *ProviderAdapter) ScaleResource(ctx context.Context, resourceID string, req *ScaleRequest) error {
	return fmt.Errorf("scale resource not implemented in adapter")
}

func (pa *ProviderAdapter) GetScalingCapabilities(ctx context.Context, resourceID string) (*ScalingCapabilities, error) {
	return nil, fmt.Errorf("get scaling capabilities not implemented in adapter")
}

func (pa *ProviderAdapter) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{"load_percentage": 10.0}, nil
}

func (pa *ProviderAdapter) GetResourceMetrics(ctx context.Context, resourceID string) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (pa *ProviderAdapter) GetResourceHealth(ctx context.Context, resourceID string) (*HealthStatus, error) {
	status, err := pa.provider.GetResourceStatus(ctx, resourceID)
	if err != nil {
		return nil, err
	}
	
	return &HealthStatus{
		Status:      string(status),
		Message:     "Resource is healthy",
		LastUpdated: time.Now(),
	}, nil
}

func (pa *ProviderAdapter) CreateNetworkService(ctx context.Context, req *NetworkServiceRequest) (*NetworkServiceResponse, error) {
	return nil, fmt.Errorf("create network service not implemented in adapter")
}

func (pa *ProviderAdapter) GetNetworkService(ctx context.Context, serviceID string) (*NetworkServiceResponse, error) {
	return nil, fmt.Errorf("get network service not implemented in adapter")
}

func (pa *ProviderAdapter) DeleteNetworkService(ctx context.Context, serviceID string) error {
	return fmt.Errorf("delete network service not implemented in adapter")
}

func (pa *ProviderAdapter) ListNetworkServices(ctx context.Context, filter *NetworkServiceFilter) ([]*NetworkServiceResponse, error) {
	return nil, fmt.Errorf("list network services not implemented in adapter")
}

func (pa *ProviderAdapter) CreateStorageResource(ctx context.Context, req *StorageResourceRequest) (*StorageResourceResponse, error) {
	return nil, fmt.Errorf("create storage resource not implemented in adapter")
}

func (pa *ProviderAdapter) GetStorageResource(ctx context.Context, resourceID string) (*StorageResourceResponse, error) {
	return nil, fmt.Errorf("get storage resource not implemented in adapter")
}

func (pa *ProviderAdapter) DeleteStorageResource(ctx context.Context, resourceID string) error {
	return fmt.Errorf("delete storage resource not implemented in adapter")
}

func (pa *ProviderAdapter) ListStorageResources(ctx context.Context, filter *StorageResourceFilter) ([]*StorageResourceResponse, error) {
	return nil, fmt.Errorf("list storage resources not implemented in adapter")
}

func (pa *ProviderAdapter) SubscribeToEvents(ctx context.Context, callback EventCallback) error {
	return fmt.Errorf("subscribe to events not implemented in adapter")
}

func (pa *ProviderAdapter) UnsubscribeFromEvents(ctx context.Context) error {
	return fmt.Errorf("unsubscribe from events not implemented in adapter")
}

func (pa *ProviderAdapter) ApplyConfiguration(ctx context.Context, config *ProviderConfiguration) error {
	return fmt.Errorf("apply configuration not implemented in adapter")
}

func (pa *ProviderAdapter) GetConfiguration(ctx context.Context) (*ProviderConfiguration, error) {
	return nil, fmt.Errorf("get configuration not implemented in adapter")
}

func (pa *ProviderAdapter) ValidateConfiguration(ctx context.Context, config *ProviderConfiguration) error {
	return nil // Simplified validation
}

// ProviderSelectionCriteria defines criteria for selecting a provider
type ProviderSelectionCriteria struct {
	Type                 string   // Provider type (kubernetes, openstack, aws, etc.)
	Region               string   // Preferred region
	Zone                 string   // Preferred zone
	RequireHealthy       bool     // Only select healthy providers
	RequiredCapabilities []string // Required capabilities
	SelectionStrategy    string   // Selection strategy (random, least-loaded, round-robin)
}