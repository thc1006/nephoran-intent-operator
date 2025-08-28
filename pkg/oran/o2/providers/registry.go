package providers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ProviderRegistry manages cloud provider instances.
type ProviderRegistry struct {
	providers           map[string]CloudProvider
	configurations      map[string]*ProviderConfiguration
	healthStatus        map[string]*HealthStatus
	metrics             map[string]map[string]interface{}
	mu                  sync.RWMutex
	stopCh              chan struct{}
	healthCheckInterval time.Duration
}

// NewProviderRegistry creates a new provider registry.
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

// RegisterProvider registers a new cloud provider.
func (r *ProviderRegistry) RegisterProvider(name string, provider CloudProvider, config *ProviderConfiguration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider %s already registered", name)
	}

	// Validate configuration if provided.
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

// UnregisterProvider removes a provider from the registry.
func (r *ProviderRegistry) UnregisterProvider(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	provider, exists := r.providers[name]
	if !exists {
		return fmt.Errorf("provider %s not found", name)
	}

	// Disconnect and close the provider.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := provider.Disconnect(ctx); err != nil {
		// Log error but continue with unregistration.
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

// GetProvider retrieves a registered provider.
func (r *ProviderRegistry) GetProvider(name string) (CloudProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}

	return provider, nil
}

// GetProviderByType retrieves the first provider of a specific type.
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

// ListProviders returns all registered provider names.
func (r *ProviderRegistry) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}

	return names
}

// ListProvidersByType returns providers of a specific type.
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

// GetSupportedProviders returns a list of supported provider types.
func (r *ProviderRegistry) GetSupportedProviders() []string {
	return []string{
		ProviderTypeKubernetes,
		ProviderTypeOpenStack,
		ProviderTypeVMware,
		ProviderTypeAWS,
		ProviderTypeAzure,
		ProviderTypeGCP,
	}
}

// GetProviderInfo returns information about a provider.
func (r *ProviderRegistry) GetProviderInfo(name string) (*ProviderInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}

	return provider.GetProviderInfo(), nil
}

// GetProviderHealth returns the health status of a provider.
func (r *ProviderRegistry) GetProviderHealth(name string) (*HealthStatus, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	health, exists := r.healthStatus[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}

	return health, nil
}

// GetAllProviderHealth returns health status for all providers.
func (r *ProviderRegistry) GetAllProviderHealth() map[string]*HealthStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*HealthStatus)
	for name, health := range r.healthStatus {
		result[name] = health
	}

	return result
}

// GetProviderMetrics returns metrics for a provider.
func (r *ProviderRegistry) GetProviderMetrics(name string) (map[string]interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metrics, exists := r.metrics[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}

	return metrics, nil
}

// ConnectAll connects all registered providers.
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

// ConnectProvider connects a specific provider.
func (r *ProviderRegistry) ConnectProvider(ctx context.Context, name string) error {
	provider, err := r.GetProvider(name)
	if err != nil {
		return err
	}

	return r.connectProvider(ctx, name, provider)
}

// connectProvider handles the actual connection logic.
func (r *ProviderRegistry) connectProvider(ctx context.Context, name string, provider CloudProvider) error {
	// Apply configuration if available.
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

	// Connect to the provider.
	if err := provider.Connect(ctx); err != nil {
		r.updateHealthStatus(name, &HealthStatus{
			Status:      HealthStatusUnhealthy,
			Message:     fmt.Sprintf("Failed to connect: %v", err),
			LastUpdated: time.Now(),
		})
		return fmt.Errorf("failed to connect provider %s: %w", name, err)
	}

	// Perform initial health check.
	if err := provider.HealthCheck(ctx); err != nil {
		r.updateHealthStatus(name, &HealthStatus{
			Status:      HealthStatusUnhealthy,
			Message:     fmt.Sprintf("Health check failed: %v", err),
			LastUpdated: time.Now(),
		})
		// Don't return error here as provider is connected.
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

// DisconnectAll disconnects all providers.
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

// StartHealthChecks starts periodic health checks for all providers.
func (r *ProviderRegistry) StartHealthChecks(ctx context.Context) {
	go wait.Until(func() {
		r.performHealthChecks(ctx)
	}, r.healthCheckInterval, r.stopCh)
}

// StopHealthChecks stops periodic health checks.
func (r *ProviderRegistry) StopHealthChecks() {
	close(r.stopCh)
}

// performHealthChecks performs health checks on all providers.
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

		// Collect metrics.
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

// updateHealthStatus updates the health status for a provider.
func (r *ProviderRegistry) updateHealthStatus(name string, status *HealthStatus) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.healthStatus[name] = status
}

// updateMetrics updates metrics for a provider.
func (r *ProviderRegistry) updateMetrics(name string, metrics map[string]interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.metrics[name] = metrics
}

// SelectProvider selects the best provider based on selection criteria.
func (r *ProviderRegistry) SelectProvider(ctx context.Context, criteria *ProviderSelectionCriteria) (CloudProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var candidates []string

	// Filter by type if specified.
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

	// Filter by health if required.
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

	// Filter by capabilities if specified.
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

	// Filter by region if specified.
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

	// Select based on strategy.
	switch criteria.SelectionStrategy {
	case "random":
		// Return random provider from candidates.
		return r.providers[candidates[0]], nil // Simplified for now
	case "least-loaded":
		return r.selectLeastLoadedProvider(candidates)
	case "round-robin":
		return r.selectRoundRobinProvider(candidates)
	default:
		// Return first available.
		return r.providers[candidates[0]], nil
	}
}

// hasRequiredCapabilities checks if provider has required capabilities.
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

// selectLeastLoadedProvider selects the provider with the lowest load.
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

// selectRoundRobinProvider selects providers in round-robin fashion.
func (r *ProviderRegistry) selectRoundRobinProvider(candidates []string) (CloudProvider, error) {
	// Simplified implementation - would need state tracking for true round-robin.
	if len(candidates) > 0 {
		return r.providers[candidates[0]], nil
	}
	return nil, fmt.Errorf("no provider selected")
}

// ProviderSelectionCriteria defines criteria for selecting a provider.
type ProviderSelectionCriteria struct {
	Type                 string   // Provider type (kubernetes, openstack, aws, etc.)
	Region               string   // Preferred region
	Zone                 string   // Preferred zone
	RequireHealthy       bool     // Only select healthy providers
	RequiredCapabilities []string // Required capabilities
	SelectionStrategy    string   // Selection strategy (random, least-loaded, round-robin)
}

// ProviderFactory creates provider instances.
type ProviderFactory struct {
	registry *ProviderRegistry
}

// NewProviderFactory creates a new provider factory.
func NewProviderFactory(registry *ProviderRegistry) *ProviderFactory {
	return &ProviderFactory{
		registry: registry,
	}
}

// CreateProvider creates a new provider instance based on configuration.
func (f *ProviderFactory) CreateProvider(config *ProviderConfiguration) (CloudProvider, error) {
	switch config.Type {
	case ProviderTypeKubernetes:
		// Would need to pass appropriate clients.
		return nil, fmt.Errorf("kubernetes provider requires k8s clients")
	case ProviderTypeOpenStack:
		return NewOpenStackProvider(config)
	case ProviderTypeAWS:
		return NewAWSProvider(config)
	case ProviderTypeAzure:
		return NewAzureProvider(config)
	case ProviderTypeGCP:
		return NewGCPProvider(config)
	case ProviderTypeVMware:
		return NewVMwareProvider(config)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", config.Type)
	}
}

// CreateAndRegisterProvider creates and registers a provider.
func (f *ProviderFactory) CreateAndRegisterProvider(name string, config *ProviderConfiguration) error {
	provider, err := f.CreateProvider(config)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	if err := f.registry.RegisterProvider(name, provider, config); err != nil {
		return fmt.Errorf("failed to register provider: %w", err)
	}

	return nil
}
