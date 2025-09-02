/*

Copyright 2025.



Licensed under the Apache License, Version 2.0 (the "License");

you may not use this file except in compliance with the License.

You may obtain a copy of the License at



    http://www.apache.org/licenses/LICENSE-2.0



Unless required by applicable law or agreed to in writing, software

distributed under the License is distributed on an "AS IS" BASIS,

WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.

See the License for the specific language governing permissions and

limitations under the License.

*/

package shared

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// IntegrationManager orchestrates all shared components and controllers.

type IntegrationManager struct {
	logger logr.Logger

	manager manager.Manager

	client client.Client

	scheme *runtime.Scheme

	recorder record.EventRecorder

	// Core shared components.

	stateManager *StateManager

	eventBus *EnhancedEventBus

	coordinationManager *CoordinationManager

	performanceOptimizer *PerformanceOptimizer

	recoveryManager *RecoveryManager

	// Configuration.

	config *IntegrationConfig

	// Component registry.

	controllers map[ComponentType]ControllerInterface

	webhooks map[string]WebhookInterface

	healthCheckers map[string]HealthChecker

	// Lifecycle management.

	mutex sync.RWMutex

	started bool

	components []StartableComponent

	// Monitoring and metrics.

	systemHealth *SystemHealth

	metricsCollector *MetricsCollector
}

// IntegrationConfig provides comprehensive configuration for the integration.

type IntegrationConfig struct {
	// Manager configuration.

	MetricsAddr string `json:"metricsAddr"`

	ProbeAddr string `json:"probeAddr"`

	PprofAddr string `json:"pprofAddr"`

	EnableLeaderElection bool `json:"enableLeaderElection"`

	LeaderElectionID string `json:"leaderElectionID"`

	// State management configuration.

	StateManager *StateManagerConfig `json:"stateManager,omitempty"`

	// Event bus configuration.

	EventBus *EventBusConfig `json:"eventBus,omitempty"`

	// Coordination configuration.

	Coordination *CoordinationConfig `json:"coordination,omitempty"`

	// Performance configuration.

	Performance *PerformanceConfig `json:"performance,omitempty"`

	// Recovery configuration.

	Recovery *RecoveryConfig `json:"recovery,omitempty"`

	// Health and monitoring.

	HealthCheckInterval time.Duration `json:"healthCheckInterval"`

	MetricsInterval time.Duration `json:"metricsInterval"`

	EnableProfiling bool `json:"enableProfiling"`

	// Webhook configuration.

	WebhookPort int `json:"webhookPort"`

	WebhookCertDir string `json:"webhookCertDir"`

	// Security configuration.

	EnableRBAC bool `json:"enableRBAC"`

	SecureMetrics bool `json:"secureMetrics"`
}

// DefaultIntegrationConfig returns default configuration.

func DefaultIntegrationConfig() *IntegrationConfig {
	return &IntegrationConfig{
		MetricsAddr: ":8080",

		ProbeAddr: ":8081",

		PprofAddr: ":6060",

		EnableLeaderElection: true,

		LeaderElectionID: "nephoran-intent-operator-leader",

		StateManager: DefaultStateManagerConfig(),

		EventBus: DefaultEventBusConfig(),

		Coordination: DefaultCoordinationConfig(),

		Performance: DefaultPerformanceConfig(),

		Recovery: DefaultRecoveryConfig(),

		HealthCheckInterval: 30 * time.Second,

		MetricsInterval: 1 * time.Minute,

		EnableProfiling: false,

		WebhookPort: 9443,

		WebhookCertDir: "/tmp/k8s-webhook-server/serving-certs",

		EnableRBAC: true,

		SecureMetrics: true,
	}
}

// StartableComponent defines interface for components that can be started/stopped.

type StartableComponent interface {
	Start(ctx context.Context) error

	Stop(ctx context.Context) error

	IsHealthy() bool

	GetName() string
}

// WebhookInterface defines interface for webhooks.

type WebhookInterface interface {
	SetupWithManager(mgr manager.Manager) error

	GetPath() string
}

// HealthChecker defines interface for health checkers.

type HealthChecker interface {
	Check(ctx context.Context) error

	GetName() string
}

// MetricsCollector collects and exposes metrics.

type MetricsCollector struct {
	mutex sync.RWMutex

	metrics map[string]interface{}

	lastCollection time.Time
}

// NewIntegrationManager creates a new integration manager.

func NewIntegrationManager(mgr manager.Manager, config *IntegrationConfig) (*IntegrationManager, error) {
	if config == nil {
		config = DefaultIntegrationConfig()
	}

	im := &IntegrationManager{
		logger: ctrl.Log.WithName("integration-manager"),

		manager: mgr,

		client: mgr.GetClient(),

		scheme: mgr.GetScheme(),

		recorder: mgr.GetEventRecorderFor("integration-manager"),

		config: config,

		controllers: make(map[ComponentType]ControllerInterface),

		webhooks: make(map[string]WebhookInterface),

		healthCheckers: make(map[string]HealthChecker),

		components: make([]StartableComponent, 0),

		systemHealth: &SystemHealth{
			Components: make(map[string]*ComponentStatus),

			ResourceUsage: ResourceUsage{},
		},

		metricsCollector: &MetricsCollector{
			metrics: make(map[string]interface{}),
		},
	}

	// Initialize core components.

	if err := im.initializeComponents(); err != nil {
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	return im, nil
}

// initializeComponents initializes all core shared components.

func (im *IntegrationManager) initializeComponents() error {
	// Create event bus.

	im.eventBus = NewEnhancedEventBus(im.config.EventBus)

	im.addComponent(im.wrapEventBus(im.eventBus))

	// Create state manager.

	im.stateManager = NewStateManager(im.client, im.recorder, im.logger, im.scheme, im.eventBus)

	im.addComponent(im.wrapStateManager(im.stateManager))

	// Create coordination manager.

	im.coordinationManager = NewCoordinationManager(im.stateManager, im.eventBus, im.config.Coordination)

	im.addComponent(im.wrapCoordinationManager(im.coordinationManager))

	// Create performance optimizer.

	im.performanceOptimizer = NewPerformanceOptimizer(im.config.Performance)

	im.addComponent(im.wrapPerformanceOptimizer(im.performanceOptimizer))

	// Create recovery manager.

	im.recoveryManager = NewRecoveryManager(im.stateManager, im.eventBus)

	im.addComponent(im.wrapRecoveryManager(im.recoveryManager))

	im.logger.Info("Core components initialized", "componentCount", len(im.components))

	return nil
}

// addComponent adds a component to the integration manager.

func (im *IntegrationManager) addComponent(component StartableComponent) {
	im.components = append(im.components, component)

	im.logger.V(1).Info("Added component", "component", component.GetName())
}

// RegisterController registers a specialized controller.

func (im *IntegrationManager) RegisterController(controller ControllerInterface) error {
	componentType := controller.GetComponentType()

	im.mutex.Lock()

	defer im.mutex.Unlock()

	// Register with coordination manager.

	if err := im.coordinationManager.RegisterController(controller); err != nil {
		return fmt.Errorf("failed to register controller with coordination manager: %w", err)
	}

	// Store in registry.

	im.controllers[componentType] = controller

	im.logger.Info("Registered controller", "componentType", componentType)

	return nil
}

// RegisterWebhook registers a webhook.

func (im *IntegrationManager) RegisterWebhook(name string, webhook WebhookInterface) error {
	im.mutex.Lock()

	defer im.mutex.Unlock()

	// Setup webhook with manager.

	if err := webhook.SetupWithManager(im.manager); err != nil {
		return fmt.Errorf("failed to setup webhook with manager: %w", err)
	}

	// Store in registry.

	im.webhooks[name] = webhook

	im.logger.Info("Registered webhook", "name", name, "path", webhook.GetPath())

	return nil
}

// RegisterHealthChecker registers a health checker.

func (im *IntegrationManager) RegisterHealthChecker(checker HealthChecker) {
	im.mutex.Lock()

	defer im.mutex.Unlock()

	im.healthCheckers[checker.GetName()] = checker

	im.logger.Info("Registered health checker", "name", checker.GetName())
}

// SetupWithManager sets up the integration manager with the controller manager.

func (im *IntegrationManager) SetupWithManager() error {
	// Add health and readiness checks.

	if err := im.manager.AddHealthzCheck("healthz", im.healthzCheck); err != nil {
		return fmt.Errorf("failed to add healthz check: %w", err)
	}

	if err := im.manager.AddReadyzCheck("readyz", im.readyzCheck); err != nil {
		return fmt.Errorf("failed to add readyz check: %w", err)
	}

	// Setup webhook server if webhooks are registered.

	if len(im.webhooks) > 0 {

		webhookServer := im.manager.GetWebhookServer()

		// Note: In controller-runtime v0.21.0, webhook server configuration.

		// is typically done during manager creation with webhook.Options.

		// The direct field access is not available, so we skip this configuration.

		// and rely on the manager's default webhook configuration.

		im.logger.Info("Webhook server configured with default settings",

			"expectedPort", im.config.WebhookPort,

			"expectedCertDir", im.config.WebhookCertDir)

		_ = webhookServer // Use the webhook server reference to avoid unused variable

	}

	// Add integration manager as a runnable.

	if err := im.manager.Add(im); err != nil {
		return fmt.Errorf("failed to add integration manager to controller manager: %w", err)
	}

	im.logger.Info("Integration manager setup completed")

	return nil
}

// Start implements the manager.Runnable interface.

func (im *IntegrationManager) Start(ctx context.Context) error {
	im.mutex.Lock()

	defer im.mutex.Unlock()

	if im.started {
		return fmt.Errorf("integration manager already started")
	}

	im.logger.Info("Starting integration manager", "componentCount", len(im.components))

	// Start all components in order.

	for i, component := range im.components {

		im.logger.Info("Starting component", "component", component.GetName(), "index", i)

		if err := component.Start(ctx); err != nil {

			im.logger.Error(err, "Failed to start component", "component", component.GetName())

			// Stop previously started components.

			for j := i - 1; j >= 0; j-- {
				if stopErr := im.components[j].Stop(ctx); stopErr != nil {
					im.logger.Error(stopErr, "Failed to stop component during rollback",

						"component", im.components[j].GetName())
				}
			}

			return fmt.Errorf("failed to start component %s: %w", component.GetName(), err)

		}

		im.logger.Info("Component started successfully", "component", component.GetName())

	}

	// Start background monitoring.

	go im.monitoringLoop(ctx)

	go im.healthCheckLoop(ctx)

	go im.metricsCollectionLoop(ctx)

	im.started = true

	im.logger.Info("Integration manager started successfully")

	return nil
}

// Stop stops the integration manager and all components.

func (im *IntegrationManager) Stop(ctx context.Context) error {
	im.mutex.Lock()

	defer im.mutex.Unlock()

	if !im.started {
		return nil
	}

	im.logger.Info("Stopping integration manager")

	// Stop components in reverse order.

	for i := len(im.components) - 1; i >= 0; i-- {

		component := im.components[i]

		im.logger.Info("Stopping component", "component", component.GetName())

		if err := component.Stop(ctx); err != nil {
			im.logger.Error(err, "Failed to stop component", "component", component.GetName())
		} else {
			im.logger.Info("Component stopped successfully", "component", component.GetName())
		}

	}

	im.started = false

	im.logger.Info("Integration manager stopped")

	return nil
}

// GetStateManager returns the state manager.

func (im *IntegrationManager) GetStateManager() *StateManager {
	return im.stateManager
}

// GetEventBus returns the event bus.

func (im *IntegrationManager) GetEventBus() EventBus {
	return im.eventBus
}

// GetCoordinationManager returns the coordination manager.

func (im *IntegrationManager) GetCoordinationManager() *CoordinationManager {
	return im.coordinationManager
}

// GetPerformanceOptimizer returns the performance optimizer.

func (im *IntegrationManager) GetPerformanceOptimizer() *PerformanceOptimizer {
	return im.performanceOptimizer
}

// GetRecoveryManager returns the recovery manager.

func (im *IntegrationManager) GetRecoveryManager() *RecoveryManager {
	return im.recoveryManager
}

// GetSystemHealth returns the current system health status.

func (im *IntegrationManager) GetSystemHealth() *SystemHealth {
	im.mutex.RLock()

	defer im.mutex.RUnlock()

	// Create a copy to avoid concurrent access issues.

	healthCopy := *im.systemHealth

	healthCopy.Components = make(map[string]*ComponentStatus)

	for name, status := range im.systemHealth.Components {

		statusCopy := *status

		healthCopy.Components[name] = &statusCopy

	}

	return &healthCopy
}

// GetMetrics returns current system metrics.

func (im *IntegrationManager) GetMetrics() map[string]interface{} {
	im.metricsCollector.mutex.RLock()

	defer im.metricsCollector.mutex.RUnlock()

	// Create a copy of metrics.

	metricsCopy := make(map[string]interface{})

	for key, value := range im.metricsCollector.metrics {
		metricsCopy[key] = value
	}

	return metricsCopy
}

// Health check implementations.

func (im *IntegrationManager) healthzCheck(_ *http.Request) error {
	// Check if all critical components are healthy.

	for _, component := range im.components {
		if !component.IsHealthy() {
			return fmt.Errorf("component %s is unhealthy", component.GetName())
		}
	}

	// Check registered health checkers.

	for name, checker := range im.healthCheckers {
		if err := checker.Check(context.Background()); err != nil {
			return fmt.Errorf("health check %s failed: %w", name, err)
		}
	}

	return nil
}

func (im *IntegrationManager) readyzCheck(_ *http.Request) error {
	if !im.started {
		return fmt.Errorf("integration manager not started")
	}

	return im.healthzCheck(nil)
}

// Background monitoring loops.

func (im *IntegrationManager) monitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(im.config.HealthCheckInterval)

	defer ticker.Stop()

	for {
		select {

		case <-ticker.C:

			im.updateSystemHealth(ctx)

		case <-ctx.Done():

			return

		}
	}
}

func (im *IntegrationManager) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(im.config.HealthCheckInterval)

	defer ticker.Stop()

	for {
		select {

		case <-ticker.C:

			im.performHealthChecks(ctx)

		case <-ctx.Done():

			return

		}
	}
}

func (im *IntegrationManager) metricsCollectionLoop(ctx context.Context) {
	ticker := time.NewTicker(im.config.MetricsInterval)

	defer ticker.Stop()

	for {
		select {

		case <-ticker.C:

			im.collectMetrics(ctx)

		case <-ctx.Done():

			return

		}
	}
}

func (im *IntegrationManager) updateSystemHealth(ctx context.Context) {
	im.mutex.Lock()

	defer im.mutex.Unlock()

	now := time.Now()

	overallHealthy := true

	// Update component statuses.

	for _, component := range im.components {

		name := component.GetName()

		healthy := component.IsHealthy()

		if !healthy {
			overallHealthy = false
		}

		status := &ComponentStatus{
			Type: ComponentTypeLLMProcessor, // This would be properly determined

			Name: name,

			Status: "running",

			Healthy: healthy,

			LastUpdate: now,

			Metadata: make(map[string]interface{}),

			Metrics: make(map[string]float64),

			Errors: make([]string, 0),
		}

		if !healthy {

			status.Status = "unhealthy"

			status.Errors = append(status.Errors, "component health check failed")

		}

		im.systemHealth.Components[name] = status

	}

	// Update overall system health.

	im.systemHealth.Healthy = overallHealthy

	im.systemHealth.LastUpdate = now

	if overallHealthy {
		im.systemHealth.OverallStatus = "healthy"
	} else {
		im.systemHealth.OverallStatus = "degraded"
	}
}

func (im *IntegrationManager) performHealthChecks(ctx context.Context) {
	im.mutex.RLock()

	checkers := make(map[string]HealthChecker)

	for name, checker := range im.healthCheckers {
		checkers[name] = checker
	}

	im.mutex.RUnlock()

	for name, checker := range checkers {
		if err := checker.Check(ctx); err != nil {
			im.logger.Error(err, "Health check failed", "checker", name)
		}
	}
}

func (im *IntegrationManager) collectMetrics(ctx context.Context) {
	im.metricsCollector.mutex.Lock()

	defer im.metricsCollector.mutex.Unlock()

	now := time.Now()

	// Collect metrics from all components.

	if im.stateManager != nil {

		stats := im.stateManager.GetStatistics()

		im.metricsCollector.metrics["state_manager"] = stats

	}

	if im.eventBus != nil {
		// Collect event bus metrics.

		im.metricsCollector.metrics["event_bus"] = json.RawMessage("{}")
	}

	if im.performanceOptimizer != nil {
		// Collect performance metrics.

		im.metricsCollector.metrics["performance"] = json.RawMessage("{}")
	}

	im.metricsCollector.lastCollection = now
}

// Component adapter implementations.

// StateManagerAdapter adapts StateManager to StartableComponent interface.

type StateManagerAdapter struct {
	*StateManager
}

// GetName performs getname operation.

func (sma *StateManagerAdapter) GetName() string {
	return "state-manager"
}

// IsHealthy performs ishealthy operation.

func (sma *StateManagerAdapter) IsHealthy() bool {
	// Check if the state manager is started and healthy.

	return sma.StateManager != nil && sma.started
}

// EnhancedEventBusAdapter adapts EnhancedEventBus to StartableComponent interface.

type EnhancedEventBusAdapter struct {
	*EnhancedEventBus
}

// GetName performs getname operation.

func (eeba *EnhancedEventBusAdapter) GetName() string {
	return "enhanced-event-bus"
}

// IsHealthy performs ishealthy operation.

func (eeba *EnhancedEventBusAdapter) IsHealthy() bool {
	return eeba.started
}

// CoordinationManagerAdapter adapts CoordinationManager to StartableComponent interface.

type CoordinationManagerAdapter struct {
	*CoordinationManager
}

// GetName performs getname operation.

func (cma *CoordinationManagerAdapter) GetName() string {
	return "coordination-manager"
}

// IsHealthy performs ishealthy operation.

func (cma *CoordinationManagerAdapter) IsHealthy() bool {
	return cma.started
}

// PerformanceOptimizerAdapter adapts PerformanceOptimizer to StartableComponent interface.

type PerformanceOptimizerAdapter struct {
	*PerformanceOptimizer
}

// GetName performs getname operation.

func (poa *PerformanceOptimizerAdapter) GetName() string {
	return "performance-optimizer"
}

// IsHealthy performs ishealthy operation.

func (poa *PerformanceOptimizerAdapter) IsHealthy() bool {
	return poa.started
}

// RecoveryManagerAdapter adapts RecoveryManager to StartableComponent interface.

type RecoveryManagerAdapter struct {
	*RecoveryManager
}

// GetName performs getname operation.

func (rma *RecoveryManagerAdapter) GetName() string {
	return "recovery-manager"
}

// IsHealthy performs ishealthy operation.

func (rma *RecoveryManagerAdapter) IsHealthy() bool {
	return rma.started
}

// Utility functions for creating component adapters.

func (im *IntegrationManager) wrapStateManager(sm *StateManager) StartableComponent {
	return &StateManagerAdapter{StateManager: sm}
}

func (im *IntegrationManager) wrapEventBus(eb *EnhancedEventBus) StartableComponent {
	return &EnhancedEventBusAdapter{EnhancedEventBus: eb}
}

func (im *IntegrationManager) wrapCoordinationManager(cm *CoordinationManager) StartableComponent {
	return &CoordinationManagerAdapter{CoordinationManager: cm}
}

func (im *IntegrationManager) wrapPerformanceOptimizer(po *PerformanceOptimizer) StartableComponent {
	return &PerformanceOptimizerAdapter{PerformanceOptimizer: po}
}

func (im *IntegrationManager) wrapRecoveryManager(rm *RecoveryManager) StartableComponent {
	return &RecoveryManagerAdapter{RecoveryManager: rm}
}
