package ca

import (
	"context"
	"fmt"
	"time"

	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// AutomationIntegration provides the main integration point for certificate automation.

type AutomationIntegration struct {
	manager manager.Manager

	logger *logging.StructuredLogger

	caManager *CAManager

	automationEngine *AutomationEngine

	config *AutomationIntegrationConfig
}

// AutomationIntegrationConfig configures the automation integration.

type AutomationIntegrationConfig struct {
	// CA Manager configuration.

	CAManagerConfig *Config `yaml:"ca_manager"`

	// Automation Engine configuration.

	AutomationConfig *AutomationConfig `yaml:"automation_engine"`

	// Integration settings.

	EnableKubernetesIntegration bool `yaml:"enable_kubernetes_integration"`

	EnableMonitoringIntegration bool `yaml:"enable_monitoring_integration"`

	EnableAlertingIntegration bool `yaml:"enable_alerting_integration"`

	// Performance settings.

	MaxConcurrentOperations int `yaml:"max_concurrent_operations"`

	OperationTimeout time.Duration `yaml:"operation_timeout"`

	HealthCheckInterval time.Duration `yaml:"health_check_interval"`

	// Security settings.

	EnableSecurityScanning bool `yaml:"enable_security_scanning"`

	EnableComplianceChecks bool `yaml:"enable_compliance_checks"`

	EnableAuditLogging bool `yaml:"enable_audit_logging"`
}

// NewAutomationIntegration creates a new automation integration.

func NewAutomationIntegration(
	mgr manager.Manager,

	logger *logging.StructuredLogger,

	config *AutomationIntegrationConfig,
) (*AutomationIntegration, error) {
	if config == nil {
		return nil, fmt.Errorf("integration config is required")
	}

	// Set defaults.

	if config.MaxConcurrentOperations == 0 {
		config.MaxConcurrentOperations = 50
	}

	if config.OperationTimeout == 0 {
		config.OperationTimeout = 5 * time.Minute
	}

	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 30 * time.Second
	}

	integration := &AutomationIntegration{
		manager: mgr,

		logger: logger,

		config: config,
	}

	return integration, nil
}

// Initialize initializes all components of the automation integration.

func (ai *AutomationIntegration) Initialize(ctx context.Context) error {
	ai.logger.Info("initializing certificate automation integration")

	// Initialize CA Manager.

	if err := ai.initializeCAManager(); err != nil {
		return fmt.Errorf("failed to initialize CA manager: %w", err)
	}

	// Initialize Automation Engine.

	if err := ai.initializeAutomationEngine(); err != nil {
		return fmt.Errorf("failed to initialize automation engine: %w", err)
	}

	// Setup Kubernetes controllers if enabled.

	if ai.config.EnableKubernetesIntegration {
		if err := ai.setupKubernetesControllers(); err != nil {
			return fmt.Errorf("failed to setup Kubernetes controllers: %w", err)
		}
	}

	// Setup monitoring integration if enabled.

	if ai.config.EnableMonitoringIntegration {
		if err := ai.setupMonitoringIntegration(); err != nil {
			return fmt.Errorf("failed to setup monitoring integration: %w", err)
		}
	}

	// Setup alerting integration if enabled.

	if ai.config.EnableAlertingIntegration {
		if err := ai.setupAlertingIntegration(); err != nil {
			return fmt.Errorf("failed to setup alerting integration: %w", err)
		}
	}

	ai.logger.Info("certificate automation integration initialized successfully")

	return nil
}

// Start starts the automation integration.

func (ai *AutomationIntegration) Start(ctx context.Context) error {
	ai.logger.Info("starting certificate automation integration")

	// Start CA Manager background processes.

	go func() {
		ai.caManager.runCertificateLifecycleManager()
	}()

	// Start Automation Engine.

	go func() {
		if err := ai.automationEngine.Start(ctx); err != nil {
			ai.logger.Error("automation engine error", "error", err)
		}
	}()

	// Start health checking.

	go ai.runHealthChecks(ctx)

	// Wait for context cancellation.

	<-ctx.Done()

	ai.logger.Info("certificate automation integration stopped")

	return nil
}

// Stop stops the automation integration.

func (ai *AutomationIntegration) Stop() error {
	ai.logger.Info("stopping certificate automation integration")

	// Stop automation engine.

	if ai.automationEngine != nil {
		ai.automationEngine.Stop()
	}

	// Stop CA manager.

	if ai.caManager != nil {
		ai.caManager.Close()
	}

	return nil
}

// GetCAManager returns the CA manager instance.

func (ai *AutomationIntegration) GetCAManager() *CAManager {
	return ai.caManager
}

// GetAutomationEngine returns the automation engine instance.

func (ai *AutomationIntegration) GetAutomationEngine() *AutomationEngine {
	return ai.automationEngine
}

// GetHealthStatus returns the overall health status.

func (ai *AutomationIntegration) GetHealthStatus() map[string]interface{} {
	status := make(map[string]interface{})

	// Check CA Manager health.

	if ai.caManager != nil {

		caHealth := ai.caManager.HealthCheck(context.Background())

		caStatus := map[string]interface{}{
			"healthy": len(caHealth) == 0,
			"errors": caHealth,
		}

		if len(caHealth) > 0 {
			status["healthy"] = false
		}

		status["components"].(map[string]interface{})["ca_manager"] = caStatus

	}

	// Check Automation Engine health.

	if ai.automationEngine != nil {

		engineStatus := map[string]interface{}{
			"running": ai.automationEngine.IsRunning(),
		}

		status["components"].(map[string]interface{})["automation_engine"] = engineStatus

	}

	return status
}

// GetMetrics returns integration metrics.

func (ai *AutomationIntegration) GetMetrics() map[string]interface{} {
	metrics := map[string]interface{}{
		"automation_engine": make(map[string]interface{}),
	}

	// Add CA Manager metrics if available.

	if ai.caManager != nil && ai.caManager.monitor != nil {
		// This would collect CA manager metrics.

		metrics["ca_manager"] = make(map[string]interface{})
	}

	// Add Automation Engine metrics.

	if ai.automationEngine != nil {
		metrics["automation_engine"] = make(map[string]interface{})
	}

	return metrics
}

// Emergency procedures for incident response.

func (ai *AutomationIntegration) EmergencyRevokeCertificate(ctx context.Context, serialNumber string, reason int) error {
	ai.logger.Warn("emergency certificate revocation initiated",

		"serial_number", serialNumber,

		"reason", reason)

	if ai.caManager == nil {
		return fmt.Errorf("CA manager not available for emergency revocation")
	}

	if err := ai.caManager.RevokeCertificate(ctx, serialNumber, reason, "emergency"); err != nil {

		ai.logger.Error("emergency certificate revocation failed",

			"serial_number", serialNumber,

			"error", err)

		return fmt.Errorf("emergency revocation failed: %w", err)

	}

	ai.logger.Info("emergency certificate revocation completed",

		"serial_number", serialNumber)

	return nil
}

// BulkRevokeCertificates revokes multiple certificates (emergency procedure).

func (ai *AutomationIntegration) BulkRevokeCertificates(ctx context.Context, serialNumbers []string, reason int) error {
	ai.logger.Warn("bulk certificate revocation initiated",

		"count", len(serialNumbers),

		"reason", reason)

	if ai.caManager == nil {
		return fmt.Errorf("CA manager not available for bulk revocation")
	}

	errors := []string{}

	successCount := 0

	for _, serialNumber := range serialNumbers {
		if err := ai.caManager.RevokeCertificate(ctx, serialNumber, reason, "bulk_emergency"); err != nil {

			ai.logger.Error("bulk revocation failed for certificate",

				"serial_number", serialNumber,

				"error", err)

			errors = append(errors, fmt.Sprintf("%s: %v", serialNumber, err))

		} else {
			successCount++
		}
	}

	ai.logger.Info("bulk certificate revocation completed",

		"total", len(serialNumbers),

		"successful", successCount,

		"failed", len(errors))

	if len(errors) > 0 {
		return fmt.Errorf("bulk revocation partially failed: %v", errors)
	}

	return nil
}

// Internal initialization methods.

func (ai *AutomationIntegration) initializeCAManager() error {
	ai.logger.Info("initializing CA manager")

	caManager, err := NewCAManager(ai.config.CAManagerConfig, ai.logger, ai.manager.GetClient())
	if err != nil {
		return fmt.Errorf("failed to create CA manager: %w", err)
	}

	ai.caManager = caManager

	return nil
}

func (ai *AutomationIntegration) initializeAutomationEngine() error {
	ai.logger.Info("initializing automation engine")

	if ai.caManager == nil {
		return fmt.Errorf("CA manager must be initialized before automation engine")
	}

	kubeClient, err := kubernetes.NewForConfig(ai.manager.GetConfig())
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	automationEngine, err := NewAutomationEngine(

		ai.config.AutomationConfig,

		ai.logger,

		ai.caManager,

		kubeClient,

		ai.manager.GetClient(),
	)
	if err != nil {
		return fmt.Errorf("failed to create automation engine: %w", err)
	}

	ai.automationEngine = automationEngine

	return nil
}

func (ai *AutomationIntegration) setupKubernetesControllers() error {
	ai.logger.Info("setting up Kubernetes controllers")

	// This would setup the CertificateAutomationReconciler and other controllers.

	// Implementation would register controllers with the manager.

	return nil
}

func (ai *AutomationIntegration) setupMonitoringIntegration() error {
	ai.logger.Info("setting up monitoring integration")

	// This would integrate with existing Prometheus/Grafana monitoring.

	// Implementation would expose metrics endpoints and dashboards.

	return nil
}

func (ai *AutomationIntegration) setupAlertingIntegration() error {
	ai.logger.Info("setting up alerting integration")

	// This would integrate with existing alerting systems.

	// Implementation would configure alert rules and notification channels.

	return nil
}

func (ai *AutomationIntegration) runHealthChecks(ctx context.Context) {
	ticker := time.NewTicker(ai.config.HealthCheckInterval)

	defer ticker.Stop()

	for {
		select {

		case <-ctx.Done():

			return

		case <-ticker.C:

			ai.performHealthChecks()

		}
	}
}

func (ai *AutomationIntegration) performHealthChecks() {
	status := ai.GetHealthStatus()

	if !status["healthy"].(bool) {

		ai.logger.Warn("certificate automation integration health check failed",

			"status", status)

		// Trigger alerts if configured.

		if ai.config.EnableAlertingIntegration {
			ai.triggerHealthAlert(status)
		}

	} else {
		ai.logger.Debug("certificate automation integration health check passed")
	}
}

func (ai *AutomationIntegration) triggerHealthAlert(status map[string]interface{}) {
	// This would trigger health-related alerts.

	ai.logger.Warn("health alert triggered", "status", status)
}

// Default configuration factory.

func NewDefaultAutomationIntegrationConfig() *AutomationIntegrationConfig {
	return &AutomationIntegrationConfig{
		CAManagerConfig: &Config{
			DefaultBackend: BackendSelfSigned,

			BackendConfigs: make(map[CABackendType]interface{}),

			DefaultValidityDuration: 365 * 24 * time.Hour, // 1 year

			RenewalThreshold: 30 * 24 * time.Hour, // 30 days

			MaxRetryAttempts: 3,

			RetryBackoff: 5 * time.Second,

			KeySize: 2048,

			AllowedKeyTypes: []string{"rsa", "ecdsa"},

			MinValidityDuration: 24 * time.Hour, // 1 day

			MaxValidityDuration: 2 * 365 * 24 * time.Hour, // 2 years

			AutoRotationEnabled: true,

			CertificateStore: &CertificateStoreConfig{
				Type: "kubernetes",

				Namespace: "nephoran-system",

				SecretPrefix: "nephoran-cert",

				BackupEnabled: true,

				BackupInterval: 24 * time.Hour,

				RetentionPeriod: 30 * 24 * time.Hour,
			},

			MonitoringConfig: &MonitoringConfig{
				Enabled: true,

				HealthCheckInterval: 30 * time.Second,

				MetricsEnabled: true,

				AlertingEnabled: true,

				ExpirationWarningDays: 30,
			},
		},

		AutomationConfig: &AutomationConfig{
			ServiceDiscoveryEnabled: true,

			DiscoveryInterval: 5 * time.Minute,

			DiscoveryNamespaces: []string{"default", "nephoran-system"},

			DiscoverySelectors: []string{"app.kubernetes.io/managed-by=nephoran"},

			AutoRenewalEnabled: true,

			RenewalThreshold: 30 * 24 * time.Hour, // 30 days

			RenewalCheckInterval: 1 * time.Hour,

			CertificateBackup: true,

			BackupRetention: 30,

			HealthCheckEnabled: true,

			HealthCheckTimeout: 30 * time.Second,

			HealthCheckRetries: 3,

			HealthCheckInterval: 1 * time.Minute,

			NotificationEnabled: true,

			NotificationEndpoints: []string{},

			AlertThresholds: make(map[string]int),

			MaxConcurrentOps: 50,

			OperationTimeout: 5 * time.Minute,

			BatchSize: 10,

			ProcessingInterval: 30 * time.Second,
		},

		EnableKubernetesIntegration: true,

		EnableMonitoringIntegration: true,

		EnableAlertingIntegration: true,

		MaxConcurrentOperations: 50,

		OperationTimeout: 5 * time.Minute,

		HealthCheckInterval: 30 * time.Second,

		EnableSecurityScanning: true,

		EnableComplianceChecks: true,

		EnableAuditLogging: true,
	}
}

