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

	"path/filepath"

	"time"



	"github.com/go-logr/logr"



	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"



	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/controller-runtime/pkg/manager"

)



// SharedInfrastructure provides the complete shared infrastructure.

type SharedInfrastructure struct {

	// Core managers.

	integrationManager   *IntegrationManager

	configurationManager *ConfigurationManager



	// Core components (accessible through integration manager).

	stateManager         *StateManager

	eventBus             EventBus

	coordinationManager  *CoordinationManager

	performanceOptimizer *PerformanceOptimizer

	recoveryManager      *RecoveryManager



	// Configuration.

	config    *IntegrationConfig

	configDir string



	// Runtime state.

	logger  logr.Logger

	started bool

}



// NewSharedInfrastructure creates a new shared infrastructure instance.

func NewSharedInfrastructure(mgr manager.Manager, configDir string) (*SharedInfrastructure, error) {

	logger := ctrl.Log.WithName("shared-infrastructure")



	// Create configuration manager.

	configMgr := NewConfigurationManager(configDir)



	// Register built-in validators.

	configMgr.RegisterBuiltinValidators()



	// Register configuration files.

	if err := configMgr.RegisterConfigFile("integration", filepath.Join(configDir, "integration.yaml")); err != nil {

		return nil, fmt.Errorf("failed to register integration config: %w", err)

	}



	if err := configMgr.RegisterConfigFile("state-manager", filepath.Join(configDir, "state-manager.yaml")); err != nil {

		return nil, fmt.Errorf("failed to register state manager config: %w", err)

	}



	if err := configMgr.RegisterConfigFile("event-bus", filepath.Join(configDir, "event-bus.yaml")); err != nil {

		return nil, fmt.Errorf("failed to register event bus config: %w", err)

	}



	if err := configMgr.RegisterConfigFile("coordination", filepath.Join(configDir, "coordination.yaml")); err != nil {

		return nil, fmt.Errorf("failed to register coordination config: %w", err)

	}



	if err := configMgr.RegisterConfigFile("performance", filepath.Join(configDir, "performance.yaml")); err != nil {

		return nil, fmt.Errorf("failed to register performance config: %w", err)

	}



	if err := configMgr.RegisterConfigFile("recovery", filepath.Join(configDir, "recovery.yaml")); err != nil {

		return nil, fmt.Errorf("failed to register recovery config: %w", err)

	}



	// Start configuration manager.

	if err := configMgr.Start(context.Background()); err != nil {

		return nil, fmt.Errorf("failed to start configuration manager: %w", err)

	}



	// Load integration configuration.

	var integrationConfig IntegrationConfig

	if err := configMgr.LoadConfiguration("integration", &integrationConfig); err != nil {

		logger.Info("Using default integration configuration", "reason", err.Error())

		integrationConfig = *DefaultIntegrationConfig()

	}



	// Load component-specific configurations.

	var stateConfig StateManagerConfig

	if err := configMgr.LoadConfiguration("state-manager", &stateConfig); err != nil {

		logger.Info("Using default state manager configuration", "reason", err.Error())

		stateConfig = *DefaultStateManagerConfig()

	}

	integrationConfig.StateManager = &stateConfig



	var eventConfig EventBusConfig

	if err := configMgr.LoadConfiguration("event-bus", &eventConfig); err != nil {

		logger.Info("Using default event bus configuration", "reason", err.Error())

		eventConfig = *DefaultEventBusConfig()

	}

	integrationConfig.EventBus = &eventConfig



	var coordConfig CoordinationConfig

	if err := configMgr.LoadConfiguration("coordination", &coordConfig); err != nil {

		logger.Info("Using default coordination configuration", "reason", err.Error())

		coordConfig = *DefaultCoordinationConfig()

	}

	integrationConfig.Coordination = &coordConfig



	var perfConfig PerformanceConfig

	if err := configMgr.LoadConfiguration("performance", &perfConfig); err != nil {

		logger.Info("Using default performance configuration", "reason", err.Error())

		perfConfig = *DefaultPerformanceConfig()

	}

	integrationConfig.Performance = &perfConfig



	var recoveryConfig RecoveryConfig

	if err := configMgr.LoadConfiguration("recovery", &recoveryConfig); err != nil {

		logger.Info("Using default recovery configuration", "reason", err.Error())

		recoveryConfig = *DefaultRecoveryConfig()

	}

	integrationConfig.Recovery = &recoveryConfig



	// Create integration manager.

	integrationMgr, err := NewIntegrationManager(mgr, &integrationConfig)

	if err != nil {

		return nil, fmt.Errorf("failed to create integration manager: %w", err)

	}



	// Create shared infrastructure.

	si := &SharedInfrastructure{

		integrationManager:   integrationMgr,

		configurationManager: configMgr,

		config:               &integrationConfig,

		configDir:            configDir,

		logger:               logger,

	}



	// Extract component references.

	si.stateManager = integrationMgr.GetStateManager()

	si.eventBus = integrationMgr.GetEventBus()

	si.coordinationManager = integrationMgr.GetCoordinationManager()

	si.performanceOptimizer = integrationMgr.GetPerformanceOptimizer()

	si.recoveryManager = integrationMgr.GetRecoveryManager()



	// Register configuration manager as a health checker.

	integrationMgr.RegisterHealthChecker(&ConfigManagerHealthChecker{configMgr})



	logger.Info("Shared infrastructure created successfully")



	return si, nil

}



// SetupWithManager sets up the shared infrastructure with the controller manager.

func (si *SharedInfrastructure) SetupWithManager() error {

	// Setup integration manager.

	if err := si.integrationManager.SetupWithManager(); err != nil {

		return fmt.Errorf("failed to setup integration manager: %w", err)

	}



	si.logger.Info("Shared infrastructure setup completed")



	return nil

}



// RegisterController registers a controller with the coordination system.

func (si *SharedInfrastructure) RegisterController(controller ControllerInterface) error {

	return si.integrationManager.RegisterController(controller)

}



// RegisterWebhook registers a webhook with the integration system.

func (si *SharedInfrastructure) RegisterWebhook(name string, webhook WebhookInterface) error {

	return si.integrationManager.RegisterWebhook(name, webhook)

}



// GetStateManager returns the state manager.

func (si *SharedInfrastructure) GetStateManager() *StateManager {

	return si.stateManager

}



// GetEventBus returns the event bus.

func (si *SharedInfrastructure) GetEventBus() EventBus {

	return si.eventBus

}



// GetCoordinationManager returns the coordination manager.

func (si *SharedInfrastructure) GetCoordinationManager() *CoordinationManager {

	return si.coordinationManager

}



// GetPerformanceOptimizer returns the performance optimizer.

func (si *SharedInfrastructure) GetPerformanceOptimizer() *PerformanceOptimizer {

	return si.performanceOptimizer

}



// GetRecoveryManager returns the recovery manager.

func (si *SharedInfrastructure) GetRecoveryManager() *RecoveryManager {

	return si.recoveryManager

}



// GetConfigurationManager returns the configuration manager.

func (si *SharedInfrastructure) GetConfigurationManager() *ConfigurationManager {

	return si.configurationManager

}



// GetSystemHealth returns the current system health.

func (si *SharedInfrastructure) GetSystemHealth() *SystemHealth {

	return si.integrationManager.GetSystemHealth()

}



// GetMetrics returns system metrics.

func (si *SharedInfrastructure) GetMetrics() map[string]interface{} {

	return si.integrationManager.GetMetrics()

}



// ProcessIntent processes a network intent through the coordination system.

func (si *SharedInfrastructure) ProcessIntent(ctx context.Context, intent *nephoranv1.NetworkIntent) error {

	namespacedName := client.ObjectKeyFromObject(intent)

	return si.coordinationManager.ProcessIntent(ctx, namespacedName)

}



// IsHealthy returns the overall health status.

func (si *SharedInfrastructure) IsHealthy() bool {

	health := si.GetSystemHealth()

	return health.Healthy

}



// Start starts the shared infrastructure (automatically handled by controller manager).

func (si *SharedInfrastructure) Start(ctx context.Context) error {

	if si.started {

		return nil

	}



	si.started = true

	si.logger.Info("Shared infrastructure started")



	return nil

}



// Stop stops the shared infrastructure.

func (si *SharedInfrastructure) Stop(ctx context.Context) error {

	if !si.started {

		return nil

	}



	// Stop configuration manager.

	if err := si.configurationManager.Stop(ctx); err != nil {

		si.logger.Error(err, "Failed to stop configuration manager")

	}



	si.started = false

	si.logger.Info("Shared infrastructure stopped")



	return nil

}



// Utility functions.



// SetupSharedInfrastructureWithManager is a convenience function to set up the complete shared infrastructure.

func SetupSharedInfrastructureWithManager(mgr manager.Manager, configDir string) (*SharedInfrastructure, error) {

	// Add our scheme to the manager.

	if err := nephoranv1.AddToScheme(mgr.GetScheme()); err != nil {

		return nil, fmt.Errorf("failed to add scheme: %w", err)

	}



	// Create shared infrastructure.

	sharedInfra, err := NewSharedInfrastructure(mgr, configDir)

	if err != nil {

		return nil, fmt.Errorf("failed to create shared infrastructure: %w", err)

	}



	// Setup with manager.

	if err := sharedInfra.SetupWithManager(); err != nil {

		return nil, fmt.Errorf("failed to setup shared infrastructure: %w", err)

	}



	return sharedInfra, nil

}



// ConfigManagerHealthChecker implements health checking for configuration manager.

type ConfigManagerHealthChecker struct {

	configMgr *ConfigurationManager

}



// Check performs check operation.

func (c *ConfigManagerHealthChecker) Check(ctx context.Context) error {

	if !c.configMgr.IsHealthy() {

		return fmt.Errorf("configuration manager is unhealthy")

	}

	return nil

}



// GetName performs getname operation.

func (c *ConfigManagerHealthChecker) GetName() string {

	return "configuration-manager"

}



// ConfigurationChangeHandler handles configuration changes for the shared infrastructure.

type ConfigurationChangeHandler struct {

	sharedInfra *SharedInfrastructure

	logger      logr.Logger

}



// NewConfigurationChangeHandler performs newconfigurationchangehandler operation.

func NewConfigurationChangeHandler(si *SharedInfrastructure) *ConfigurationChangeHandler {

	return &ConfigurationChangeHandler{

		sharedInfra: si,

		logger:      ctrl.Log.WithName("config-change-handler"),

	}

}



// OnConfigChange performs onconfigchange operation.

func (cch *ConfigurationChangeHandler) OnConfigChange(configType string, oldConfig, newConfig interface{}) error {

	cch.logger.Info("Configuration changed", "configType", configType)



	// Handle different configuration types.

	switch configType {

	case "integration":

		return cch.handleIntegrationConfigChange(oldConfig, newConfig)

	case "state-manager":

		return cch.handleStateManagerConfigChange(oldConfig, newConfig)

	case "event-bus":

		return cch.handleEventBusConfigChange(oldConfig, newConfig)

	case "coordination":

		return cch.handleCoordinationConfigChange(oldConfig, newConfig)

	case "performance":

		return cch.handlePerformanceConfigChange(oldConfig, newConfig)

	case "recovery":

		return cch.handleRecoveryConfigChange(oldConfig, newConfig)

	}



	return nil

}



func (cch *ConfigurationChangeHandler) handleIntegrationConfigChange(oldConfig, newConfig interface{}) error {

	// Handle integration configuration changes.

	cch.logger.Info("Integration configuration changed - may require restart")

	return nil

}



func (cch *ConfigurationChangeHandler) handleStateManagerConfigChange(oldConfig, newConfig interface{}) error {

	// Handle state manager configuration changes.

	cch.logger.Info("State manager configuration changed")

	return nil

}



func (cch *ConfigurationChangeHandler) handleEventBusConfigChange(oldConfig, newConfig interface{}) error {

	// Handle event bus configuration changes.

	cch.logger.Info("Event bus configuration changed")

	return nil

}



func (cch *ConfigurationChangeHandler) handleCoordinationConfigChange(oldConfig, newConfig interface{}) error {

	// Handle coordination configuration changes.

	cch.logger.Info("Coordination configuration changed")

	return nil

}



func (cch *ConfigurationChangeHandler) handlePerformanceConfigChange(oldConfig, newConfig interface{}) error {

	// Handle performance configuration changes.

	cch.logger.Info("Performance configuration changed")

	return nil

}



func (cch *ConfigurationChangeHandler) handleRecoveryConfigChange(oldConfig, newConfig interface{}) error {

	// Handle recovery configuration changes.

	cch.logger.Info("Recovery configuration changed")

	return nil

}



// ValidationConfig provides validation configuration.

type ValidationConfig struct {

	EnableStrict          bool `json:"enableStrict"`

	ValidateOnStartup     bool `json:"validateOnStartup"`

	ValidateOnChange      bool `json:"validateOnChange"`

	FailOnValidationError bool `json:"failOnValidationError"`

}



// MonitoringConfig provides monitoring configuration.

type MonitoringConfig struct {

	EnableMetrics       bool          `json:"enableMetrics"`

	EnableTracing       bool          `json:"enableTracing"`

	EnableProfiling     bool          `json:"enableProfiling"`

	MetricsInterval     time.Duration `json:"metricsInterval"`

	HealthCheckInterval time.Duration `json:"healthCheckInterval"`

}

