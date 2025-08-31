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

package packagerevision

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
	"github.com/thc1006/nephoran-intent-operator/pkg/templates"
	"github.com/thc1006/nephoran-intent-operator/pkg/validation/yang"
)

// SystemFactory orchestrates the creation and initialization of all PackageRevision lifecycle components.

// Provides a centralized factory for creating and configuring the complete system.

type SystemFactory interface {

	// Core component creation.

	CreatePorchClient(ctx context.Context, config *porch.ClientConfig) (porch.PorchClient, error)

	CreateLifecycleManager(ctx context.Context, client porch.PorchClient, config *porch.LifecycleManagerConfig) (porch.LifecycleManager, error)

	CreateTemplateEngine(ctx context.Context, yangValidator yang.YANGValidator, config *templates.EngineConfig) (templates.TemplateEngine, error)

	CreateYANGValidator(ctx context.Context, config *yang.ValidatorConfig) (yang.YANGValidator, error)

	CreatePackageRevisionManager(ctx context.Context, components *SystemComponents, config *ManagerConfig) (PackageRevisionManager, error)

	// Integrated system creation.

	CreateCompleteSystem(ctx context.Context, systemConfig *SystemConfig) (*PackageRevisionSystem, error)

	CreateNetworkIntentIntegration(ctx context.Context, k8sClient client.Client, scheme *runtime.Scheme, system *PackageRevisionSystem, config *IntegrationConfig) (*NetworkIntentPackageReconciler, error)

	// Health and lifecycle management.

	GetSystemHealth(ctx context.Context) (*SystemHealth, error)

	ShutdownSystem(ctx context.Context, gracePeriod time.Duration) error

	Close() error
}

// systemFactory implements comprehensive system factory.

type systemFactory struct {
	logger logr.Logger

	components map[string]interface{}

	systems []*PackageRevisionSystem

	shutdown chan struct{}
}

// SystemComponents holds all the core system components.

type SystemComponents struct {
	PorchClient porch.PorchClient

	LifecycleManager porch.LifecycleManager

	TemplateEngine templates.TemplateEngine

	YANGValidator yang.YANGValidator

	PackageManager PackageRevisionManager
}

// SystemConfig contains configuration for the complete system.

type SystemConfig struct {

	// Component configurations.

	PorchConfig *porch.ClientConfig `yaml:"porch"`

	LifecycleConfig *porch.LifecycleManagerConfig `yaml:"lifecycle"`

	TemplateConfig *templates.EngineConfig `yaml:"templates"`

	YANGConfig *yang.ValidatorConfig `yaml:"yang"`

	PackageManagerConfig *ManagerConfig `yaml:"packageManager"`

	IntegrationConfig *IntegrationConfig `yaml:"integration"`

	// System-level configuration.

	SystemName string `yaml:"systemName"`

	Environment string `yaml:"environment"`

	LogLevel string `yaml:"logLevel"`

	EnableMetrics bool `yaml:"enableMetrics"`

	MetricsPort int `yaml:"metricsPort"`

	HealthCheckPort int `yaml:"healthCheckPort"`

	GracefulShutdownTimeout time.Duration `yaml:"gracefulShutdownTimeout"`

	// Feature flags.

	Features *FeatureFlags `yaml:"features"`

	// External integrations.

	Integrations *ExternalIntegrations `yaml:"integrations"`
}

// FeatureFlags controls optional system features.

type FeatureFlags struct {
	EnableORANCompliance bool `yaml:"enableOranCompliance"`

	Enable3GPPValidation bool `yaml:"enable3gppValidation"`

	EnableDriftDetection bool `yaml:"enableDriftDetection"`

	EnableAutoCorrection bool `yaml:"enableAutoCorrection"`

	EnableApprovalWorkflows bool `yaml:"enableApprovalWorkflows"`

	EnableAdvancedTemplating bool `yaml:"enableAdvancedTemplating"`

	EnableSecurityScanning bool `yaml:"enableSecurityScanning"`

	EnableChaosEngineering bool `yaml:"enableChaosEngineering"`

	EnableMLOptimization bool `yaml:"enableMlOptimization"`
}

// ExternalIntegrations defines external system integrations.

type ExternalIntegrations struct {
	GitOps *GitOpsIntegration `yaml:"gitops"`

	CICD *CICDIntegration `yaml:"cicd"`

	Monitoring *MonitoringIntegration `yaml:"monitoring"`

	SecretManagement *SecretIntegration `yaml:"secretManagement"`

	CertificateManagement *CertIntegration `yaml:"certificateManagement"`

	NetworkTelemetry *TelemetryIntegration `yaml:"networkTelemetry"`
}

// Integration configuration types.

// GitOpsIntegration represents a gitopsintegration.

type GitOpsIntegration struct {
	Provider string `yaml:"provider"` // argocd, flux, tekton

	Endpoint string `yaml:"endpoint"`

	Repository string `yaml:"repository"`

	Branch string `yaml:"branch"`

	Credentials map[string]string `yaml:"credentials"`

	AutoSync bool `yaml:"autoSync"`

	SyncWave int `yaml:"syncWave"`
}

// CICDIntegration represents a cicdintegration.

type CICDIntegration struct {
	Provider string `yaml:"provider"` // jenkins, gitlab, github-actions, tekton

	Endpoint string `yaml:"endpoint"`

	Credentials map[string]string `yaml:"credentials"`

	PipelineTemplates []string `yaml:"pipelineTemplates"`

	TriggerOnEvents []string `yaml:"triggerOnEvents"`
}

// MonitoringIntegration represents a monitoringintegration.

type MonitoringIntegration struct {
	Prometheus *PrometheusConfig `yaml:"prometheus"`

	Grafana *GrafanaConfig `yaml:"grafana"`

	Jaeger *JaegerConfig `yaml:"jaeger"`

	Fluentd *FluentdConfig `yaml:"fluentd"`
}

// PrometheusConfig represents a prometheusconfig.

type PrometheusConfig struct {
	Enabled bool `yaml:"enabled"`

	Endpoint string `yaml:"endpoint"`

	Namespace string `yaml:"namespace"`

	ScrapeInterval string `yaml:"scrapeInterval"`
}

// GrafanaConfig represents a grafanaconfig.

type GrafanaConfig struct {
	Enabled bool `yaml:"enabled"`

	Endpoint string `yaml:"endpoint"`

	DashboardRepo string `yaml:"dashboardRepo"`

	AutoProvisioning bool `yaml:"autoProvisioning"`
}

// JaegerConfig represents a jaegerconfig.

type JaegerConfig struct {
	Enabled bool `yaml:"enabled"`

	Endpoint string `yaml:"endpoint"`

	SamplingRate float64 `yaml:"samplingRate"`
}

// FluentdConfig represents a fluentdconfig.

type FluentdConfig struct {
	Enabled bool `yaml:"enabled"`

	Endpoint string `yaml:"endpoint"`

	IndexPattern string `yaml:"indexPattern"`
}

// SecretIntegration represents a secretintegration.

type SecretIntegration struct {
	Provider string `yaml:"provider"` // vault, k8s-secrets, aws-secrets-manager

	Endpoint string `yaml:"endpoint"`

	Credentials map[string]string `yaml:"credentials"`

	VaultPath string `yaml:"vaultPath,omitempty"`

	KeyRotation *KeyRotationConfig `yaml:"keyRotation"`
}

// KeyRotationConfig represents a keyrotationconfig.

type KeyRotationConfig struct {
	Enabled bool `yaml:"enabled"`

	Interval time.Duration `yaml:"interval"`

	PreRotationHook string `yaml:"preRotationHook"`

	PostRotationHook string `yaml:"postRotationHook"`
}

// CertIntegration represents a certintegration.

type CertIntegration struct {
	Provider string `yaml:"provider"` // cert-manager, vault, external-ca

	Issuer string `yaml:"issuer"`

	CABundle string `yaml:"caBundle"`

	AutoRenewal bool `yaml:"autoRenewal"`

	RenewalBefore time.Duration `yaml:"renewalBefore"`
}

// TelemetryIntegration represents a telemetryintegration.

type TelemetryIntegration struct {
	ONAPIntegration *ONAPConfig `yaml:"onap"`

	OSMIntegration *OSMConfig `yaml:"osm"`

	CustomEndpoints []*TelemetryEndpoint `yaml:"customEndpoints"`
}

// ONAPConfig represents a onapconfig.

type ONAPConfig struct {
	Enabled bool `yaml:"enabled"`

	DCCAEEndpoint string `yaml:"dccaeEndpoint"`

	PolicyEndpoint string `yaml:"policyEndpoint"`

	SDCEndpoint string `yaml:"sdcEndpoint"`
}

// OSMConfig represents a osmconfig.

type OSMConfig struct {
	Enabled bool `yaml:"enabled"`

	NBI_Endpoint string `yaml:"nbiEndpoint"`

	KeystoneURL string `yaml:"keystoneUrl"`
}

// TelemetryEndpoint represents a telemetryendpoint.

type TelemetryEndpoint struct {
	Name string `yaml:"name"`

	URL string `yaml:"url"`

	Type string `yaml:"type"` // metrics, logs, traces, events

	Credentials map[string]string `yaml:"credentials"`

	Format string `yaml:"format"` // json, xml, protobuf

}

// PackageRevisionSystem represents a complete configured system.

type PackageRevisionSystem struct {
	Config *SystemConfig

	Components *SystemComponents

	Health *SystemHealth

	CreatedAt time.Time

	LastHealthCheck time.Time

	SystemID string
}

// SystemHealth represents the overall system health.

type SystemHealth struct {
	Status string `json:"status"` // healthy, degraded, unhealthy

	Components map[string]ComponentHealth `json:"components"`

	LastCheck time.Time `json:"lastCheck"`

	Uptime time.Duration `json:"uptime"`

	Version string `json:"version"`

	BuildInfo *BuildInfo `json:"buildInfo,omitempty"`

	FeatureStatus map[string]bool `json:"featureStatus"`

	IntegrationStatus map[string]IntegrationHealth `json:"integrationStatus"`

	Metrics *SystemMetrics `json:"metrics,omitempty"`
}

// ComponentHealth represents a componenthealth.

type ComponentHealth struct {
	Status string `json:"status"`

	LastCheck time.Time `json:"lastCheck"`

	Error string `json:"error,omitempty"`

	Latency time.Duration `json:"latency,omitempty"`

	Uptime time.Duration `json:"uptime,omitempty"`

	Version string `json:"version,omitempty"`
}

// IntegrationHealth represents a integrationhealth.

type IntegrationHealth struct {
	Status string `json:"status"`

	LastCheck time.Time `json:"lastCheck"`

	Error string `json:"error,omitempty"`

	ResponseTime time.Duration `json:"responseTime,omitempty"`

	Available bool `json:"available"`
}

// BuildInfo represents a buildinfo.

type BuildInfo struct {
	Version string `json:"version"`

	GitCommit string `json:"gitCommit"`

	BuildDate time.Time `json:"buildDate"`

	GoVersion string `json:"goVersion"`

	Platform string `json:"platform"`
}

// SystemMetrics represents a systemmetrics.

type SystemMetrics struct {
	TotalRequests int64 `json:"totalRequests"`

	SuccessfulRequests int64 `json:"successfulRequests"`

	FailedRequests int64 `json:"failedRequests"`

	AverageResponseTime time.Duration `json:"averageResponseTime"`

	ActiveConnections int `json:"activeConnections"`

	MemoryUsage int64 `json:"memoryUsage"`

	CPUUsage float64 `json:"cpuUsage"`

	DiskUsage int64 `json:"diskUsage"`
}

// NewSystemFactory creates a new system factory.

func NewSystemFactory() SystemFactory {

	return &systemFactory{

		logger: log.Log.WithName("packagerevision-factory"),

		components: make(map[string]interface{}),

		systems: []*PackageRevisionSystem{},

		shutdown: make(chan struct{}),
	}

}

// CreateCompleteSystem creates a complete PackageRevision system with all components.

func (f *systemFactory) CreateCompleteSystem(ctx context.Context, systemConfig *SystemConfig) (*PackageRevisionSystem, error) {

	f.logger.Info("Creating complete PackageRevision system",

		"systemName", systemConfig.SystemName,

		"environment", systemConfig.Environment)

	startTime := time.Now()

	// Validate system configuration.

	if err := f.validateSystemConfig(systemConfig); err != nil {

		return nil, fmt.Errorf("invalid system configuration: %w", err)

	}

	// Apply default configurations for nil components.

	systemConfig = f.applyDefaults(systemConfig)

	// Create YANG validator first (required by template engine).

	yangValidator, err := f.CreateYANGValidator(ctx, systemConfig.YANGConfig)

	if err != nil {

		return nil, fmt.Errorf("failed to create YANG validator: %w", err)

	}

	// Create template engine.

	templateEngine, err := f.CreateTemplateEngine(ctx, yangValidator, systemConfig.TemplateConfig)

	if err != nil {

		return nil, fmt.Errorf("failed to create template engine: %w", err)

	}

	// Create Porch client.

	porchClient, err := f.CreatePorchClient(ctx, systemConfig.PorchConfig)

	if err != nil {

		return nil, fmt.Errorf("failed to create Porch client: %w", err)

	}

	// Create lifecycle manager.

	lifecycleManager, err := f.CreateLifecycleManager(ctx, porchClient, systemConfig.LifecycleConfig)

	if err != nil {

		return nil, fmt.Errorf("failed to create lifecycle manager: %w", err)

	}

	// Create system components.

	components := &SystemComponents{

		PorchClient: porchClient,

		LifecycleManager: lifecycleManager,

		TemplateEngine: templateEngine,

		YANGValidator: yangValidator,
	}

	// Create package revision manager.

	packageManager, err := f.CreatePackageRevisionManager(ctx, components, systemConfig.PackageManagerConfig)

	if err != nil {

		return nil, fmt.Errorf("failed to create package revision manager: %w", err)

	}

	components.PackageManager = packageManager

	// Create system with initial health check.

	system := &PackageRevisionSystem{

		Config: systemConfig,

		Components: components,

		CreatedAt: startTime,

		SystemID: fmt.Sprintf("%s-%d", systemConfig.SystemName, time.Now().UnixNano()),
	}

	// Perform initial health check.

	health, err := f.performSystemHealthCheck(ctx, system)

	if err != nil {

		f.logger.Error(err, "Initial health check failed, but system created")

		health = &SystemHealth{

			Status: "unhealthy",

			LastCheck: time.Now(),

			Components: map[string]ComponentHealth{},
		}

	}

	system.Health = health

	system.LastHealthCheck = time.Now()

	// Initialize external integrations if configured.

	if err := f.initializeIntegrations(ctx, system); err != nil {

		f.logger.Error(err, "Failed to initialize some external integrations")

	}

	// Register system for management.

	f.systems = append(f.systems, system)

	f.logger.Info("PackageRevision system created successfully",

		"systemName", systemConfig.SystemName,

		"systemID", system.SystemID,

		"health", health.Status,

		"duration", time.Since(startTime))

	return system, nil

}

// CreatePorchClient creates and configures a Porch client.

func (f *systemFactory) CreatePorchClient(ctx context.Context, config *porch.ClientConfig) (porch.PorchClient, error) {

	f.logger.Info("Creating Porch client", "endpoint", config.Endpoint)

	opts := porch.ClientOptions{

		Config: config,

		Logger: f.logger,
	}

	client, err := porch.NewClient(opts)

	if err != nil {

		return nil, fmt.Errorf("failed to create Porch client: %w", err)

	}

	// Test connectivity.

	if _, err := client.Health(ctx); err != nil {

		f.logger.Error(err, "Porch client health check failed")

		// Continue even if health check fails for now.

	}

	f.components["porch-client"] = client

	return client, nil

}

// CreateLifecycleManager creates and configures a lifecycle manager.

func (f *systemFactory) CreateLifecycleManager(ctx context.Context, client porch.PorchClient, config *porch.LifecycleManagerConfig) (porch.LifecycleManager, error) {

	f.logger.Info("Creating lifecycle manager")

	// Type assert PorchClient to *Client for NewLifecycleManager.

	clientImpl, ok := client.(*porch.Client)

	if !ok {

		return nil, fmt.Errorf("client must be *porch.Client, got %T", client)

	}

	lifecycleManager, err := porch.NewLifecycleManager(clientImpl, config)

	if err != nil {

		return nil, fmt.Errorf("failed to create lifecycle manager: %w", err)

	}

	f.components["lifecycle-manager"] = lifecycleManager

	return lifecycleManager, nil

}

// CreateTemplateEngine creates and configures a template engine.

func (f *systemFactory) CreateTemplateEngine(ctx context.Context, yangValidator yang.YANGValidator, config *templates.EngineConfig) (templates.TemplateEngine, error) {

	f.logger.Info("Creating template engine")

	templateEngine, err := templates.NewTemplateEngine(yangValidator, config)

	if err != nil {

		return nil, fmt.Errorf("failed to create template engine: %w", err)

	}

	f.components["template-engine"] = templateEngine

	return templateEngine, nil

}

// CreateYANGValidator creates and configures a YANG validator.

func (f *systemFactory) CreateYANGValidator(ctx context.Context, config *yang.ValidatorConfig) (yang.YANGValidator, error) {

	f.logger.Info("Creating YANG validator")

	yangValidator, err := yang.NewYANGValidator(config)

	if err != nil {

		return nil, fmt.Errorf("failed to create YANG validator: %w", err)

	}

	f.components["yang-validator"] = yangValidator

	return yangValidator, nil

}

// CreatePackageRevisionManager creates and configures a package revision manager.

func (f *systemFactory) CreatePackageRevisionManager(ctx context.Context, components *SystemComponents, config *ManagerConfig) (PackageRevisionManager, error) {

	f.logger.Info("Creating package revision manager")

	packageManager, err := NewPackageRevisionManager(

		components.PorchClient,

		components.LifecycleManager,

		components.TemplateEngine,

		components.YANGValidator,

		config,
	)

	if err != nil {

		return nil, fmt.Errorf("failed to create package revision manager: %w", err)

	}

	f.components["package-manager"] = packageManager

	return packageManager, nil

}

// CreateNetworkIntentIntegration creates the NetworkIntent integration reconciler.

func (f *systemFactory) CreateNetworkIntentIntegration(ctx context.Context, k8sClient client.Client, scheme *runtime.Scheme, system *PackageRevisionSystem, config *IntegrationConfig) (*NetworkIntentPackageReconciler, error) {

	f.logger.Info("Creating NetworkIntent integration")

	reconciler := NewNetworkIntentPackageReconciler(

		k8sClient,

		scheme,

		system.Components.PackageManager,

		system.Components.TemplateEngine,

		system.Components.YANGValidator,

		system.Components.PorchClient,

		system.Components.LifecycleManager,

		config,
	)

	return reconciler, nil

}

// SetupWithManager sets up the complete system with a controller manager.

func SetupWithManager(mgr manager.Manager, systemConfig *SystemConfig) error {

	factory := NewSystemFactory()

	// Create the complete system.

	system, err := factory.CreateCompleteSystem(context.Background(), systemConfig)

	if err != nil {

		return fmt.Errorf("failed to create PackageRevision system: %w", err)

	}

	// Create and setup the NetworkIntent integration.

	reconciler, err := factory.CreateNetworkIntentIntegration(

		context.Background(),

		mgr.GetClient(),

		mgr.GetScheme(),

		system,

		systemConfig.IntegrationConfig,
	)

	if err != nil {

		return fmt.Errorf("failed to create NetworkIntent integration: %w", err)

	}

	// Setup the reconciler with the manager.

	if err := reconciler.SetupWithManager(mgr); err != nil {

		return fmt.Errorf("failed to setup reconciler with manager: %w", err)

	}

	return nil

}

// Helper methods.

func (f *systemFactory) validateSystemConfig(config *SystemConfig) error {

	if config.SystemName == "" {

		return fmt.Errorf("system name is required")

	}

	if config.PorchConfig == nil {

		return fmt.Errorf("Porch configuration is required")

	}

	if config.PorchConfig.Endpoint == "" {

		return fmt.Errorf("Porch endpoint is required")

	}

	return nil

}

func (f *systemFactory) applyDefaults(config *SystemConfig) *SystemConfig {

	if config.Environment == "" {

		config.Environment = "development"

	}

	if config.LogLevel == "" {

		config.LogLevel = "info"

	}

	if config.GracefulShutdownTimeout == 0 {

		config.GracefulShutdownTimeout = 30 * time.Second

	}

	if config.LifecycleConfig == nil {

		config.LifecycleConfig = &porch.LifecycleManagerConfig{

			EventQueueSize: 1000,

			EventWorkers: 5,

			LockCleanupInterval: 5 * time.Minute,

			DefaultLockTimeout: 30 * time.Minute,

			EnableMetrics: true,
		}

	}

	if config.YANGConfig == nil {

		config.YANGConfig = &yang.ValidatorConfig{

			EnableConstraintValidation: true,

			EnableDataTypeValidation: true,

			EnableMandatoryCheck: true,

			ValidationTimeout: 30 * time.Second,

			MaxValidationDepth: 100,

			EnableO_RANModels: true,

			Enable3GPPModels: true,

			EnableMetrics: true,
		}

	}

	if config.TemplateConfig == nil {

		config.TemplateConfig = &templates.EngineConfig{

			TemplateRepository: "https://github.com/nephoran/templates.git",

			RepositoryBranch: "main",

			RefreshInterval: 60 * time.Minute,

			MaxConcurrentRenders: 10,

			RenderTimeout: 5 * time.Minute,

			EnableCaching: true,

			EnableParameterValidation: true,

			EnableYANGValidation: true,

			EnableMetrics: true,
		}

	}

	if config.PackageManagerConfig == nil {

		config.PackageManagerConfig = getDefaultManagerConfig()

	}

	if config.IntegrationConfig == nil {

		config.IntegrationConfig = GetDefaultIntegrationConfig()

	}

	if config.Features == nil {

		config.Features = &FeatureFlags{

			EnableORANCompliance: true,

			Enable3GPPValidation: true,

			EnableDriftDetection: true,

			EnableAutoCorrection: false,

			EnableApprovalWorkflows: true,

			EnableAdvancedTemplating: true,

			EnableSecurityScanning: true,

			EnableChaosEngineering: false,

			EnableMLOptimization: false,
		}

	}

	return config

}

func (f *systemFactory) performSystemHealthCheck(ctx context.Context, system *PackageRevisionSystem) (*SystemHealth, error) {

	health := &SystemHealth{

		Status: "healthy",

		Components: make(map[string]ComponentHealth),

		LastCheck: time.Now(),

		Uptime: time.Since(system.CreatedAt),

		Version: "1.0.0", // Would be from build info

		FeatureStatus: make(map[string]bool),

		IntegrationStatus: make(map[string]IntegrationHealth),
	}

	// Check Porch client health.

	if _, err := system.Components.PorchClient.Health(ctx); err != nil {

		health.Status = "degraded"

		health.Components["porch-client"] = ComponentHealth{

			Status: "unhealthy",

			LastCheck: time.Now(),

			Error: err.Error(),
		}

	} else {

		health.Components["porch-client"] = ComponentHealth{

			Status: "healthy",

			LastCheck: time.Now(),

			// Note: HealthStatus doesn't have Version field.

		}

	}

	// Check lifecycle manager health.

	if lifecycleHealth, err := system.Components.LifecycleManager.GetManagerHealth(ctx); err != nil {

		health.Status = "degraded"

		health.Components["lifecycle-manager"] = ComponentHealth{

			Status: "unhealthy",

			LastCheck: time.Now(),

			Error: err.Error(),
		}

	} else {

		health.Components["lifecycle-manager"] = ComponentHealth{

			Status: lifecycleHealth.Status,

			LastCheck: time.Now(),
		}

	}

	// Check template engine health.

	if templateHealth, err := system.Components.TemplateEngine.GetEngineHealth(ctx); err != nil {

		health.Status = "degraded"

		health.Components["template-engine"] = ComponentHealth{

			Status: "unhealthy",

			LastCheck: time.Now(),

			Error: err.Error(),
		}

	} else {

		health.Components["template-engine"] = ComponentHealth{

			Status: templateHealth.Status,

			LastCheck: time.Now(),
		}

	}

	// Check YANG validator health.

	if yangHealth, err := system.Components.YANGValidator.GetValidatorHealth(ctx); err != nil {

		health.Status = "degraded"

		health.Components["yang-validator"] = ComponentHealth{

			Status: "unhealthy",

			LastCheck: time.Now(),

			Error: err.Error(),
		}

	} else {

		health.Components["yang-validator"] = ComponentHealth{

			Status: yangHealth.Status,

			LastCheck: time.Now(),
		}

	}

	// Check package manager health.

	if managerHealth, err := system.Components.PackageManager.GetManagerHealth(ctx); err != nil {

		health.Status = "degraded"

		health.Components["package-manager"] = ComponentHealth{

			Status: "unhealthy",

			LastCheck: time.Now(),

			Error: err.Error(),
		}

	} else {

		health.Components["package-manager"] = ComponentHealth{

			Status: managerHealth.Status,

			LastCheck: time.Now(),
		}

	}

	return health, nil

}

func (f *systemFactory) initializeIntegrations(ctx context.Context, system *PackageRevisionSystem) error {

	if system.Config.Integrations == nil {

		return nil

	}

	// Initialize GitOps integration.

	if system.Config.Integrations.GitOps != nil {

		if err := f.initializeGitOpsIntegration(ctx, system.Config.Integrations.GitOps); err != nil {

			f.logger.Error(err, "Failed to initialize GitOps integration")

		}

	}

	// Initialize monitoring integration.

	if system.Config.Integrations.Monitoring != nil {

		if err := f.initializeMonitoringIntegration(ctx, system.Config.Integrations.Monitoring); err != nil {

			f.logger.Error(err, "Failed to initialize monitoring integration")

		}

	}

	// Initialize other integrations...

	return nil

}

func (f *systemFactory) initializeGitOpsIntegration(ctx context.Context, config *GitOpsIntegration) error {

	f.logger.Info("Initializing GitOps integration", "provider", config.Provider)

	// Implementation would set up GitOps integration.

	return nil

}

func (f *systemFactory) initializeMonitoringIntegration(ctx context.Context, config *MonitoringIntegration) error {

	f.logger.Info("Initializing monitoring integration")

	// Implementation would set up monitoring integration.

	return nil

}

// GetSystemHealth returns the current system health.

func (f *systemFactory) GetSystemHealth(ctx context.Context) (*SystemHealth, error) {

	if len(f.systems) == 0 {

		return &SystemHealth{Status: "no-systems"}, nil

	}

	// For simplicity, return the first system's health.

	// In a real implementation, this might aggregate health across all systems.

	return f.performSystemHealthCheck(ctx, f.systems[0])

}

// ShutdownSystem gracefully shuts down the system.

func (f *systemFactory) ShutdownSystem(ctx context.Context, gracePeriod time.Duration) error {

	f.logger.Info("Shutting down PackageRevision systems", "gracePeriod", gracePeriod)

	// Create a timeout context.

	shutdownCtx, cancel := context.WithTimeout(ctx, gracePeriod)

	defer cancel()

	// Shutdown all systems.

	for _, system := range f.systems {

		if err := f.shutdownSingleSystem(shutdownCtx, system); err != nil {

			f.logger.Error(err, "Failed to shutdown system", "systemID", system.SystemID)

		}

	}

	f.logger.Info("All systems shutdown completed")

	return nil

}

func (f *systemFactory) shutdownSingleSystem(ctx context.Context, system *PackageRevisionSystem) error {

	f.logger.Info("Shutting down system", "systemID", system.SystemID)

	// Shutdown components in reverse order of creation.

	if system.Components.PackageManager != nil {

		if err := system.Components.PackageManager.Close(); err != nil {

			f.logger.Error(err, "Failed to close package manager")

		}

	}

	if system.Components.LifecycleManager != nil {

		if err := system.Components.LifecycleManager.Close(); err != nil {

			f.logger.Error(err, "Failed to close lifecycle manager")

		}

	}

	if system.Components.TemplateEngine != nil {

		if err := system.Components.TemplateEngine.Close(); err != nil {

			f.logger.Error(err, "Failed to close template engine")

		}

	}

	if system.Components.YANGValidator != nil {

		if err := system.Components.YANGValidator.Close(); err != nil {

			f.logger.Error(err, "Failed to close YANG validator")

		}

	}

	return nil

}

// Close gracefully shuts down the factory.

func (f *systemFactory) Close() error {

	f.logger.Info("Shutting down system factory")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer cancel()

	return f.ShutdownSystem(ctx, 25*time.Second)

}

// GetDefaultSystemConfig returns a default system configuration.

func GetDefaultSystemConfig() *SystemConfig {

	return &SystemConfig{

		SystemName: "nephoran-packagerevision-system",

		Environment: "development",

		LogLevel: "info",

		EnableMetrics: true,

		MetricsPort: 8080,

		HealthCheckPort: 8081,

		GracefulShutdownTimeout: 30 * time.Second,

		PorchConfig: &porch.ClientConfig{

			Endpoint: "http://porch-server:8080",

			// Note: ClientConfig doesn't have Timeout field.

		},

		Features: &FeatureFlags{

			EnableORANCompliance: true,

			Enable3GPPValidation: true,

			EnableDriftDetection: true,

			EnableAutoCorrection: false,

			EnableApprovalWorkflows: true,

			EnableAdvancedTemplating: true,

			EnableSecurityScanning: true,

			EnableChaosEngineering: false,

			EnableMLOptimization: false,
		},
	}

}
