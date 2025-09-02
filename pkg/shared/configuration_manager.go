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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-logr/logr"
	yaml "gopkg.in/yaml.v2"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ConfigurationManager manages system configuration.

type ConfigurationManager struct {
	logger logr.Logger

	mutex sync.RWMutex

	// Configuration storage.

	configs map[string]interface{}

	configFiles map[string]string

	configWatchers map[string]*ConfigWatcher

	// Configuration validation.

	validators map[string]ConfigValidator

	schemas map[string]interface{}

	// Change notification.

	changeNotifiers []ConfigChangeNotifier

	// Configuration source.

	configDir string

	defaultConfigs map[string]interface{}

	// Runtime state.

	started bool

	stopChan chan bool

	workerWG sync.WaitGroup
}

// ConfigValidator validates configuration.

type ConfigValidator interface {
	Validate(config interface{}) error

	GetConfigType() string
}

// ConfigChangeNotifier notifies about configuration changes.

type ConfigChangeNotifier interface {
	OnConfigChange(configType string, oldConfig, newConfig interface{}) error
}

// ConfigWatcher watches configuration files for changes.

type ConfigWatcher struct {
	filePath string

	lastModTime time.Time

	configType string

	manager *ConfigurationManager
}

// NewConfigurationManager creates a new configuration manager.

func NewConfigurationManager(configDir string) *ConfigurationManager {
	cm := &ConfigurationManager{
		logger: ctrl.Log.WithName("configuration-manager"),

		configs: make(map[string]interface{}),

		configFiles: make(map[string]string),

		configWatchers: make(map[string]*ConfigWatcher),

		validators: make(map[string]ConfigValidator),

		schemas: make(map[string]interface{}),

		changeNotifiers: make([]ConfigChangeNotifier, 0),

		configDir: configDir,

		defaultConfigs: make(map[string]interface{}),

		stopChan: make(chan bool),
	}

	// Initialize default configurations.

	cm.initializeDefaults()

	return cm
}

// initializeDefaults sets up default configurations.

func (cm *ConfigurationManager) initializeDefaults() {
	cm.defaultConfigs["integration"] = DefaultIntegrationConfig()

	cm.defaultConfigs["state-manager"] = DefaultStateManagerConfig()

	cm.defaultConfigs["event-bus"] = DefaultEventBusConfig()

	cm.defaultConfigs["coordination"] = DefaultCoordinationConfig()

	cm.defaultConfigs["performance"] = DefaultPerformanceConfig()

	cm.defaultConfigs["recovery"] = DefaultRecoveryConfig()
}

// Start starts the configuration manager.

func (cm *ConfigurationManager) Start(ctx context.Context) error {
	cm.mutex.Lock()

	defer cm.mutex.Unlock()

	if cm.started {
		return nil
	}

	// Ensure config directory exists.

	if err := os.MkdirAll(cm.configDir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Load all configurations.

	if err := cm.loadAllConfigurations(); err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}

	// Start configuration watchers.

	cm.workerWG.Add(1)

	go cm.configWatcherLoop(ctx)

	cm.started = true

	cm.logger.Info("Configuration manager started", "configDir", cm.configDir)

	return nil
}

// Stop stops the configuration manager.

func (cm *ConfigurationManager) Stop(ctx context.Context) error {
	cm.mutex.Lock()

	defer cm.mutex.Unlock()

	if !cm.started {
		return nil
	}

	// Signal stop.

	close(cm.stopChan)

	// Wait for workers.

	cm.workerWG.Wait()

	cm.started = false

	cm.logger.Info("Configuration manager stopped")

	return nil
}

// IsHealthy returns the health status.

func (cm *ConfigurationManager) IsHealthy() bool {
	return cm.started
}

// GetName returns the component name.

func (cm *ConfigurationManager) GetName() string {
	return "configuration-manager"
}

// LoadConfiguration loads a specific configuration.

func (cm *ConfigurationManager) LoadConfiguration(configType string, target interface{}) error {
	cm.mutex.RLock()

	defer cm.mutex.RUnlock()

	// Check if config exists.

	config, exists := cm.configs[configType]

	if !exists {
		// Use default configuration.

		if defaultConfig, hasDefault := cm.defaultConfigs[configType]; hasDefault {
			config = defaultConfig
		} else {
			return fmt.Errorf("configuration not found: %s", configType)
		}
	}

	// Marshal and unmarshal to convert to target type.

	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

// SaveConfiguration saves a configuration.

func (cm *ConfigurationManager) SaveConfiguration(configType string, config interface{}) error {
	cm.mutex.Lock()

	defer cm.mutex.Unlock()

	// Validate configuration.

	if validator, exists := cm.validators[configType]; exists {
		if err := validator.Validate(config); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}
	}

	// Store old config for change notification.

	oldConfig := cm.configs[configType]

	// Save new configuration.

	cm.configs[configType] = config

	// Save to file if configured.

	if filePath, exists := cm.configFiles[configType]; exists {
		if err := cm.saveConfigToFile(filePath, config); err != nil {
			return fmt.Errorf("failed to save config to file: %w", err)
		}
	}

	// Notify change listeners.

	for _, notifier := range cm.changeNotifiers {
		if err := notifier.OnConfigChange(configType, oldConfig, config); err != nil {
			cm.logger.Error(err, "Config change notification failed", "configType", configType)
		}
	}

	cm.logger.Info("Configuration saved", "configType", configType)

	return nil
}

// RegisterConfigFile registers a configuration file to watch.

func (cm *ConfigurationManager) RegisterConfigFile(configType, filePath string) error {
	cm.mutex.Lock()

	defer cm.mutex.Unlock()

	// Store file path mapping.

	cm.configFiles[configType] = filePath

	// Create watcher.

	watcher := &ConfigWatcher{
		filePath: filePath,

		configType: configType,

		manager: cm,
	}

	cm.configWatchers[configType] = watcher

	// Load configuration from file if it exists.

	if _, err := os.Stat(filePath); err == nil {
		if err := cm.loadConfigFromFile(configType, filePath); err != nil {
			return fmt.Errorf("failed to load config from file: %w", err)
		}
	}

	cm.logger.Info("Registered config file", "configType", configType, "filePath", filePath)

	return nil
}

// RegisterValidator registers a configuration validator.

func (cm *ConfigurationManager) RegisterValidator(validator ConfigValidator) {
	cm.mutex.Lock()

	defer cm.mutex.Unlock()

	configType := validator.GetConfigType()

	cm.validators[configType] = validator

	cm.logger.Info("Registered config validator", "configType", configType)
}

// RegisterChangeNotifier registers a configuration change notifier.

func (cm *ConfigurationManager) RegisterChangeNotifier(notifier ConfigChangeNotifier) {
	cm.mutex.Lock()

	defer cm.mutex.Unlock()

	cm.changeNotifiers = append(cm.changeNotifiers, notifier)

	cm.logger.Info("Registered config change notifier")
}

// GetAllConfigTypes returns all available configuration types.

func (cm *ConfigurationManager) GetAllConfigTypes() []string {
	cm.mutex.RLock()

	defer cm.mutex.RUnlock()

	types := make([]string, 0, len(cm.configs))

	for configType := range cm.configs {
		types = append(types, configType)
	}

	// Add default configs not yet loaded.

	for configType := range cm.defaultConfigs {

		found := false

		for _, existing := range types {
			if existing == configType {

				found = true

				break

			}
		}

		if !found {
			types = append(types, configType)
		}

	}

	return types
}

// GetConfigurationStatus returns the status of all configurations.

func (cm *ConfigurationManager) GetConfigurationStatus() map[string]ConfigStatus {
	cm.mutex.RLock()

	defer cm.mutex.RUnlock()

	status := make(map[string]ConfigStatus)

	for configType := range cm.configs {

		configStatus := ConfigStatus{
			Type: configType,

			Loaded: true,

			HasFile: false,

			LastLoaded: time.Now(), // This would be tracked properly

			Valid: true,
		}

		if filePath, exists := cm.configFiles[configType]; exists {

			configStatus.HasFile = true

			configStatus.FilePath = filePath

			if stat, err := os.Stat(filePath); err == nil {
				configStatus.LastModified = stat.ModTime()
			}

		}

		// Check validation.

		if validator, exists := cm.validators[configType]; exists {
			if config := cm.configs[configType]; config != nil {
				if err := validator.Validate(config); err != nil {

					configStatus.Valid = false

					configStatus.ValidationError = err.Error()

				}
			}
		}

		status[configType] = configStatus

	}

	return status
}

// ReloadConfiguration reloads a configuration from file.

func (cm *ConfigurationManager) ReloadConfiguration(configType string) error {
	cm.mutex.Lock()

	defer cm.mutex.Unlock()

	filePath, exists := cm.configFiles[configType]

	if !exists {
		return fmt.Errorf("no file registered for config type: %s", configType)
	}

	return cm.loadConfigFromFile(configType, filePath)
}

// ReloadAllConfigurations reloads all configurations from files.

func (cm *ConfigurationManager) ReloadAllConfigurations() error {
	return cm.loadAllConfigurations()
}

// Internal methods.

func (cm *ConfigurationManager) loadAllConfigurations() error {
	for configType, filePath := range cm.configFiles {
		if err := cm.loadConfigFromFile(configType, filePath); err != nil {
			cm.logger.Error(err, "Failed to load config", "configType", configType, "filePath", filePath)
		}
	}

	return nil
}

func (cm *ConfigurationManager) loadConfigFromFile(configType, filePath string) error {
	// Check if file exists.

	if _, err := os.Stat(filePath); os.IsNotExist(err) {

		cm.logger.V(1).Info("Config file does not exist, using defaults", "configType", configType, "filePath", filePath)

		return nil

	}

	// Read file.

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Determine file format.

	var config interface{}

	ext := filepath.Ext(filePath)

	switch ext {

	case ".json":

		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to unmarshal JSON config: %w", err)
		}

	case ".yaml", ".yml":

		if err := yaml.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to unmarshal YAML config: %w", err)
		}

	default:

		// Try JSON first, then YAML.

		if err := json.Unmarshal(data, &config); err != nil {
			if err := yaml.Unmarshal(data, &config); err != nil {
				return fmt.Errorf("failed to unmarshal config (tried JSON and YAML): %w", err)
			}
		}

	}

	// Validate configuration.

	if validator, exists := cm.validators[configType]; exists {
		if err := validator.Validate(config); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}
	}

	// Store old config for change notification.

	oldConfig := cm.configs[configType]

	// Store configuration.

	cm.configs[configType] = config

	// Update watcher modification time.

	if watcher, exists := cm.configWatchers[configType]; exists {
		if stat, err := os.Stat(filePath); err == nil {
			watcher.lastModTime = stat.ModTime()
		}
	}

	// Notify change listeners.

	for _, notifier := range cm.changeNotifiers {
		if err := notifier.OnConfigChange(configType, oldConfig, config); err != nil {
			cm.logger.Error(err, "Config change notification failed", "configType", configType)
		}
	}

	cm.logger.Info("Configuration loaded from file", "configType", configType, "filePath", filePath)

	return nil
}

func (cm *ConfigurationManager) saveConfigToFile(filePath string, config interface{}) error {
	// Ensure directory exists.

	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Determine format from file extension.

	ext := filepath.Ext(filePath)

	var data []byte

	var err error

	switch ext {

	case ".json":

		data, err = json.MarshalIndent(config, "", "  ")

	case ".yaml", ".yml":

		data, err = yaml.Marshal(config)

	default:

		// Default to JSON.

		data, err = json.MarshalIndent(config, "", "  ")

	}

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write file atomically.

	tmpPath := filePath + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0o640); err != nil {
		return fmt.Errorf("failed to write temp config file: %w", err)
	}

	if err := os.Rename(tmpPath, filePath); err != nil {

		os.Remove(tmpPath) // Cleanup temp file

		return fmt.Errorf("failed to rename config file: %w", err)

	}

	return nil
}

func (cm *ConfigurationManager) configWatcherLoop(ctx context.Context) {
	defer cm.workerWG.Done()

	ticker := time.NewTicker(5 * time.Second) // Check every 5 seconds

	defer ticker.Stop()

	for {
		select {

		case <-ticker.C:

			cm.checkConfigChanges()

		case <-cm.stopChan:

			return

		case <-ctx.Done():

			return

		}
	}
}

func (cm *ConfigurationManager) checkConfigChanges() {
	cm.mutex.RLock()

	watchers := make(map[string]*ConfigWatcher)

	for k, v := range cm.configWatchers {
		watchers[k] = v
	}

	cm.mutex.RUnlock()

	for _, watcher := range watchers {
		if stat, err := os.Stat(watcher.filePath); err == nil {
			if stat.ModTime().After(watcher.lastModTime) {

				cm.logger.Info("Config file changed, reloading", "configType", watcher.configType, "filePath", watcher.filePath)

				if err := cm.ReloadConfiguration(watcher.configType); err != nil {
					cm.logger.Error(err, "Failed to reload changed config", "configType", watcher.configType)
				}

			}
		}
	}
}

// ConfigStatus represents the status of a configuration.

type ConfigStatus struct {
	Type string `json:"type"`

	Loaded bool `json:"loaded"`

	HasFile bool `json:"hasFile"`

	FilePath string `json:"filePath,omitempty"`

	LastLoaded time.Time `json:"lastLoaded"`

	LastModified time.Time `json:"lastModified,omitempty"`

	Valid bool `json:"valid"`

	ValidationError string `json:"validationError,omitempty"`
}

// Built-in configuration validators.

// IntegrationConfigValidator validates integration configuration.

type IntegrationConfigValidator struct{}

// Validate performs validate operation.

func (v *IntegrationConfigValidator) Validate(config interface{}) error {
	// Convert to IntegrationConfig and validate.

	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	var integrationConfig IntegrationConfig

	if err := json.Unmarshal(data, &integrationConfig); err != nil {
		return err
	}

	// Validate required fields and ranges.

	if integrationConfig.MetricsAddr == "" {
		return fmt.Errorf("metricsAddr is required")
	}

	if integrationConfig.ProbeAddr == "" {
		return fmt.Errorf("probeAddr is required")
	}

	if integrationConfig.WebhookPort <= 0 || integrationConfig.WebhookPort > 65535 {
		return fmt.Errorf("webhookPort must be between 1 and 65535")
	}

	return nil
}

// GetConfigType performs getconfigtype operation.

func (v *IntegrationConfigValidator) GetConfigType() string {
	return "integration"
}

// StateManagerConfigValidator validates state manager configuration.

type StateManagerConfigValidator struct{}

// Validate performs validate operation.

func (v *StateManagerConfigValidator) Validate(config interface{}) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	var stateConfig StateManagerConfig

	if err := json.Unmarshal(data, &stateConfig); err != nil {
		return err
	}

	// Validate ranges and required fields.

	if stateConfig.CacheSize <= 0 {
		return fmt.Errorf("cacheSize must be positive")
	}

	if stateConfig.CacheTTL <= 0 {
		return fmt.Errorf("cacheTTL must be positive")
	}

	return nil
}

// GetConfigType performs getconfigtype operation.

func (v *StateManagerConfigValidator) GetConfigType() string {
	return "state-manager"
}

// RegisterBuiltinValidators registers all built-in validators.

func (cm *ConfigurationManager) RegisterBuiltinValidators() {
	cm.RegisterValidator(&IntegrationConfigValidator{})

	cm.RegisterValidator(&StateManagerConfigValidator{})
}
