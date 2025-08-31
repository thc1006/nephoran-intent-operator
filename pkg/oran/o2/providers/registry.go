package providers

import (
	"context"
	"fmt"
	"sync"
	"time"
	
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// DefaultProviderRegistry implements the ProviderRegistry interface
type DefaultProviderRegistry struct {
	providers      map[string]CloudProvider
	configs        map[string]*ProviderConfiguration
	healthCheckers map[string]*HealthChecker
	mu             sync.RWMutex
	healthMu       sync.RWMutex
	healthStopCh   chan struct{}
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() ProviderRegistry {
	return &DefaultProviderRegistry{
		providers:      make(map[string]CloudProvider),
		configs:        make(map[string]*ProviderConfiguration),
		healthCheckers: make(map[string]*HealthChecker),
		healthStopCh:   make(chan struct{}),
	}
}

// RegisterProvider registers a provider with configuration
func (r *DefaultProviderRegistry) RegisterProvider(name string, provider CloudProvider, config *ProviderConfiguration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider %s already registered", name)
	}
	
	r.providers[name] = provider
	r.configs[name] = config
	
	// Create health checker
	r.healthMu.Lock()
	r.healthCheckers[name] = NewHealthChecker(provider)
	r.healthMu.Unlock()
	
	return nil
}

// UnregisterProvider removes a provider
func (r *DefaultProviderRegistry) UnregisterProvider(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	provider, exists := r.providers[name]
	if !exists {
		return fmt.Errorf("provider %s not found", name)
	}
	
	// Clean up provider resources
	if err := provider.Close(); err != nil {
		// Log error but don't fail unregistration
		logger := log.Log
		logger.Error(err, "error closing provider during unregistration", "provider", name)
	}
	
	delete(r.providers, name)
	delete(r.configs, name)
	
	// Remove health checker
	r.healthMu.Lock()
	delete(r.healthCheckers, name)
	r.healthMu.Unlock()
	
	return nil
}

// GetProvider retrieves a provider by name
func (r *DefaultProviderRegistry) GetProvider(name string) (CloudProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	provider, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}
	
	return provider, nil
}

// ListProviders returns all registered provider names
func (r *DefaultProviderRegistry) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// ConnectAll connects to all registered providers
func (r *DefaultProviderRegistry) ConnectAll(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var firstError error
	for name, provider := range r.providers {
		if err := provider.Connect(ctx); err != nil {
			if firstError == nil {
				firstError = fmt.Errorf("failed to connect to provider %s: %w", name, err)
			}
			logger := log.FromContext(ctx)
			logger.Error(err, "failed to connect to provider", "provider", name)
		}
	}
	
	return firstError
}

// ConnectProvider connects to a specific provider
func (r *DefaultProviderRegistry) ConnectProvider(ctx context.Context, name string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	provider, exists := r.providers[name]
	if !exists {
		return fmt.Errorf("provider %s not found", name)
	}
	
	return provider.Connect(ctx)
}

// DisconnectAll disconnects from all providers
func (r *DefaultProviderRegistry) DisconnectAll(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var firstError error
	for name, provider := range r.providers {
		if err := provider.Disconnect(ctx); err != nil {
			if firstError == nil {
				firstError = fmt.Errorf("failed to disconnect from provider %s: %w", name, err)
			}
			logger := log.FromContext(ctx)
			logger.Error(err, "failed to disconnect from provider", "provider", name)
		}
	}
	
	return firstError
}

// StartHealthChecks starts health checking for all providers
func (r *DefaultProviderRegistry) StartHealthChecks(ctx context.Context) {
	r.healthMu.RLock()
	defer r.healthMu.RUnlock()
	
	for _, checker := range r.healthCheckers {
		go checker.Start(ctx)
	}
}

// StopHealthChecks stops health checking
func (r *DefaultProviderRegistry) StopHealthChecks() {
	r.healthMu.RLock()
	defer r.healthMu.RUnlock()
	
	close(r.healthStopCh)
	
	for _, checker := range r.healthCheckers {
		checker.Stop()
	}
}

// GetAllProviderHealth returns health status for all providers
func (r *DefaultProviderRegistry) GetAllProviderHealth() map[string]*HealthStatus {
	r.healthMu.RLock()
	defer r.healthMu.RUnlock()
	
	health := make(map[string]*HealthStatus)
	for name, checker := range r.healthCheckers {
		health[name] = checker.GetStatus()
	}
	
	return health
}

// SelectProvider selects the best provider based on criteria
func (r *DefaultProviderRegistry) SelectProvider(ctx context.Context, criteria *ProviderSelectionCriteria) (CloudProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var candidates []string
	
	// Filter providers based on criteria
	for name, provider := range r.providers {
		config := r.configs[name]
		
		// Check type
		if criteria.Type != "" && config.Type != criteria.Type {
			continue
		}
		
		// Check region
		if criteria.Region != "" && config.Region != criteria.Region {
			continue
		}
		
		// Check health if required
		if criteria.RequireHealthy {
			r.healthMu.RLock()
			checker, exists := r.healthCheckers[name]
			r.healthMu.RUnlock()
			
			if !exists || checker.GetStatus().Status != HealthStatusHealthy {
				continue
			}
		}
		
		// Check capabilities
		if len(criteria.RequiredCapabilities) > 0 {
			caps := provider.GetCapabilities()
			if !hasRequiredCapabilities(caps, criteria.RequiredCapabilities) {
				continue
			}
		}
		
		candidates = append(candidates, name)
	}
	
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no providers match the selection criteria")
	}
	
	// Apply selection strategy
	selectedName := candidates[0] // Default to first candidate
	switch criteria.SelectionStrategy {
	case "random":
		// For simplicity, just use first candidate
		// In production, implement proper random selection
	case "least-loaded":
		// For simplicity, just use first candidate
		// In production, implement load-based selection
	case "round-robin":
		// For simplicity, just use first candidate
		// In production, implement round-robin selection
	}
	
	return r.providers[selectedName], nil
}

// hasRequiredCapabilities checks if provider has required capabilities
func hasRequiredCapabilities(caps *ProviderCapabilities, required []string) bool {
	if caps == nil {
		return false
	}
	
	// Simple implementation - in production, this would be more sophisticated
	for _, req := range required {
		switch req {
		case "autoscaling":
			if !caps.AutoScaling {
				return false
			}
		case "monitoring":
			if !caps.Monitoring {
				return false
			}
		case "networking":
			if !caps.Networking {
				return false
			}
		// Add more capability checks as needed
		}
	}
	
	return true
}

// Global provider registry instance
var globalProviderRegistry ProviderRegistry = NewProviderRegistry()

// GetGlobalProviderRegistry returns the global provider registry instance
func GetGlobalProviderRegistry() ProviderRegistry {
	return globalProviderRegistry
}

// HealthChecker manages health checking for a provider
type HealthChecker struct {
	provider CloudProvider
	status   *HealthStatus
	mu       sync.RWMutex
	stopCh   chan struct{}
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(provider CloudProvider) *HealthChecker {
	return &HealthChecker{
		provider: provider,
		status: &HealthStatus{
			Status:    HealthStatusUnknown,
			Timestamp: time.Now(),
		},
		stopCh: make(chan struct{}),
	}
}

// Start starts the health checker
func (h *HealthChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			h.checkHealth(ctx)
		case <-h.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// Stop stops the health checker
func (h *HealthChecker) Stop() {
	close(h.stopCh)
}

// GetStatus returns the current health status
func (h *HealthChecker) GetStatus() *HealthStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	return &HealthStatus{
		Status:    h.status.Status,
		Message:   h.status.Message,
		Timestamp: h.status.Timestamp,
		Details:   h.status.Details,
	}
}

func (h *HealthChecker) checkHealth(ctx context.Context) {
	err := h.provider.HealthCheck(ctx)
	
	h.mu.Lock()
	defer h.mu.Unlock()
	
	if err != nil {
		h.status = &HealthStatus{
			Status:    HealthStatusUnhealthy,
			Message:   err.Error(),
			Timestamp: time.Now(),
		}
	} else {
		h.status = &HealthStatus{
			Status:    HealthStatusHealthy,
			Message:   "Provider is healthy",
			Timestamp: time.Now(),
		}
	}
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