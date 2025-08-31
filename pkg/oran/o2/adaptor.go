// Package o2 provides a comprehensive implementation of the O-RAN O2 Interface.

// for Infrastructure Management Services (IMS), supporting multi-cloud.

// resource orchestration, deployment management, and cloud-native operations.

package o2

import (
	"context"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/ims"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/providers"
)

// O2AdaptorInterface defines the complete O2 IMS interface following O-RAN.WG6.O2ims-Interface-v01.01.

type O2AdaptorInterface interface {

	// Infrastructure Management Services (O-RAN.WG6.O2ims-Interface-v01.01).

	// Resource Pool Management.

	GetResourcePools(ctx context.Context, filter *models.ResourcePoolFilter) ([]*models.ResourcePool, error)

	GetResourcePool(ctx context.Context, poolID string) (*models.ResourcePool, error)

	CreateResourcePool(ctx context.Context, req *models.CreateResourcePoolRequest) (*models.ResourcePool, error)

	UpdateResourcePool(ctx context.Context, poolID string, req *models.UpdateResourcePoolRequest) error

	DeleteResourcePool(ctx context.Context, poolID string) error

	// Resource Type Management.

	GetResourceTypes(ctx context.Context, filter *models.ResourceTypeFilter) ([]*models.ResourceType, error)

	GetResourceType(ctx context.Context, typeID string) (*models.ResourceType, error)

	// Resource Management.

	GetResources(ctx context.Context, filter *models.ResourceFilter) ([]*models.Resource, error)

	GetResource(ctx context.Context, resourceID string) (*models.Resource, error)

	CreateResource(ctx context.Context, req *models.CreateResourceRequest) (*models.Resource, error)

	UpdateResource(ctx context.Context, resourceID string, req *models.UpdateResourceRequest) error

	DeleteResource(ctx context.Context, resourceID string) error

	// Deployment Template Management.

	GetDeploymentTemplates(ctx context.Context, filter *models.DeploymentTemplateFilter) ([]*models.DeploymentTemplate, error)

	GetDeploymentTemplate(ctx context.Context, templateID string) (*models.DeploymentTemplate, error)

	CreateDeploymentTemplate(ctx context.Context, req *models.CreateDeploymentTemplateRequest) (*models.DeploymentTemplate, error)

	UpdateDeploymentTemplate(ctx context.Context, templateID string, req *models.UpdateDeploymentTemplateRequest) error

	DeleteDeploymentTemplate(ctx context.Context, templateID string) error

	// Deployment Manager Interface.

	CreateDeployment(ctx context.Context, req *models.CreateDeploymentRequest) (*models.Deployment, error)

	GetDeployments(ctx context.Context, filter *models.DeploymentFilter) ([]*models.Deployment, error)

	GetDeployment(ctx context.Context, deploymentID string) (*models.Deployment, error)

	UpdateDeployment(ctx context.Context, deploymentID string, req *models.UpdateDeploymentRequest) error

	DeleteDeployment(ctx context.Context, deploymentID string) error

	// Subscription Management.

	CreateSubscription(ctx context.Context, req *models.CreateSubscriptionRequest) (*models.Subscription, error)

	GetSubscriptions(ctx context.Context, filter *models.SubscriptionFilter) ([]*models.Subscription, error)

	GetSubscription(ctx context.Context, subscriptionID string) (*models.Subscription, error)

	UpdateSubscription(ctx context.Context, subscriptionID string, req *models.UpdateSubscriptionRequest) error

	DeleteSubscription(ctx context.Context, subscriptionID string) error

	// Notification Management.

	GetNotificationEventTypes(ctx context.Context) ([]*models.NotificationEventType, error)

	// Infrastructure Inventory Management.

	GetInventoryNodes(ctx context.Context, filter *models.NodeFilter) ([]*models.Node, error)

	GetInventoryNode(ctx context.Context, nodeID string) (*models.Node, error)

	// Alarm and Fault Management.

	GetAlarms(ctx context.Context, filter *models.AlarmFilter) ([]*models.Alarm, error)

	GetAlarm(ctx context.Context, alarmID string) (*models.Alarm, error)

	AcknowledgeAlarm(ctx context.Context, alarmID string, req *models.AlarmAcknowledgementRequest) error

	ClearAlarm(ctx context.Context, alarmID string, req *models.AlarmClearRequest) error
}

// O2Adaptor implements the complete O2 IMS interface.

type O2Adaptor struct {

	// Core clients and configuration.

	kubeClient client.Client

	clientset kubernetes.Interface

	config *O2Config

	// Service components.

	imsService *ims.IMSService

	catalogService *ims.CatalogService

	inventoryService *ims.InventoryService

	lifecycleService *ims.LifecycleService

	subscriptionService *ims.SubscriptionService

	// Multi-cloud providers.

	providers map[string]providers.CloudProvider

	// Circuit breaker and resilience.

	circuitBreaker *llm.CircuitBreaker

	retryConfig *RetryConfig

	// State management.

	resourcePools map[string]*models.ResourcePool

	deployments map[string]*models.Deployment

	subscriptions map[string]*models.Subscription

	mutex sync.RWMutex

	// Background services.

	monitoringCtx context.Context

	monitoringCancel context.CancelFunc
}

// O2Config holds comprehensive O2 interface configuration.

type O2Config struct {

	// Basic configuration.

	Namespace string `yaml:"namespace"`

	ServiceAccount string `yaml:"serviceAccount"`

	DefaultResources *corev1.ResourceRequirements `yaml:"defaultResources,omitempty"`

	// Multi-cloud configuration.

	Providers map[string]*ProviderConfig `yaml:"providers"`

	DefaultProvider string `yaml:"defaultProvider"`

	// O2 IMS specific configuration.

	IMSConfiguration *IMSConfig `yaml:"imsConfiguration"`

	// API configuration.

	APIServerConfig *APIServerConfig `yaml:"apiServerConfig"`

	// Monitoring and observability.

	MonitoringConfig *MonitoringConfig `yaml:"monitoringConfig"`

	// Security and authentication.

	TLSConfig *oran.TLSConfig `yaml:"tlsConfig"`

	AuthConfig *AuthConfig `yaml:"authConfig"`

	// Performance and resilience.

	CircuitBreakerConfig *llm.CircuitBreakerConfig `yaml:"circuitBreakerConfig"`

	RetryConfig *RetryConfig `yaml:"retryConfig"`
}

// ProviderConfig holds configuration for cloud providers.

type ProviderConfig struct {
	Type string `yaml:"type"` // kubernetes, aws, azure, gcp

	Enabled bool `yaml:"enabled"`

	Region string `yaml:"region"`

	Credentials map[string]string `yaml:"credentials"`

	Config map[string]string `yaml:"config"`
}

// IMSConfig holds O2 IMS specific configuration.

type IMSConfig struct {
	SystemName string `yaml:"systemName"`

	SystemDescription string `yaml:"systemDescription"`

	SystemVersion string `yaml:"systemVersion"`

	SupportedAPIVersions []string `yaml:"supportedAPIVersions"`

	MaxResourcePools int `yaml:"maxResourcePools"`

	MaxResources int `yaml:"maxResources"`

	MaxDeployments int `yaml:"maxDeployments"`
}

// APIServerConfig holds REST API server configuration.

type APIServerConfig struct {
	Enabled bool `yaml:"enabled"`

	Port int `yaml:"port"`

	MetricsPort int `yaml:"metricsPort"`

	EnableOpenAPI bool `yaml:"enableOpenAPI"`

	EnableCORS bool `yaml:"enableCORS"`

	RequestTimeout time.Duration `yaml:"requestTimeout"`

	MaxRequestSize int64 `yaml:"maxRequestSize"`
}

// MonitoringConfig holds monitoring and observability configuration.

type MonitoringConfig struct {
	EnableMetrics bool `yaml:"enableMetrics"`

	EnableTracing bool `yaml:"enableTracing"`

	MetricsInterval time.Duration `yaml:"metricsInterval"`

	RetentionPeriod time.Duration `yaml:"retentionPeriod"`

	AlertingEnabled bool `yaml:"alertingEnabled"`
}

// AuthConfig holds authentication and authorization configuration.

type AuthConfig struct {
	AuthMode string `yaml:"authMode"` // none, token, certificate, oauth2

	TokenValidation *TokenConfig `yaml:"tokenValidation"`

	CertificateConfig *CertificateConfig `yaml:"certificateConfig"`

	OAuth2Config *OAuth2Config `yaml:"oauth2Config"`

	RBACEnabled bool `yaml:"rbacEnabled"`
}

// TokenConfig holds token-based authentication configuration.

type TokenConfig struct {
	Enabled bool `yaml:"enabled"`

	TokenExpiry time.Duration `yaml:"tokenExpiry"`

	SecretKey string `yaml:"secretKey"`

	ValidateURL string `yaml:"validateURL"`
}

// CertificateConfig holds certificate-based authentication configuration.

type CertificateConfig struct {
	Enabled bool `yaml:"enabled"`

	CAFile string `yaml:"caFile"`

	CertFile string `yaml:"certFile"`

	KeyFile string `yaml:"keyFile"`

	ClientCAFile string `yaml:"clientCAFile"`

	VerifyClientCert bool `yaml:"verifyClientCert"`
}

// OAuth2Config holds OAuth2 authentication configuration.

type OAuth2Config struct {
	Enabled bool `yaml:"enabled"`

	ProviderURL string `yaml:"providerURL"`

	ClientID string `yaml:"clientID"`

	ClientSecret string `yaml:"clientSecret"`

	RedirectURL string `yaml:"redirectURL"`

	Scopes []string `yaml:"scopes"`
}

// RetryConfig holds retry configuration for O2 operations.

type RetryConfig struct {
	MaxRetries int `json:"max_retries"`

	InitialDelay time.Duration `json:"initial_delay"`

	MaxDelay time.Duration `json:"max_delay"`

	BackoffFactor float64 `json:"backoff_factor"`

	Jitter bool `json:"jitter"`

	RetryableErrors []string `json:"retryable_errors"`
}

// NewO2Adaptor creates a new O2 adaptor with comprehensive IMS capabilities.

func NewO2Adaptor(kubeClient client.Client, clientset kubernetes.Interface, config *O2Config) (*O2Adaptor, error) {

	if config == nil {

		config = DefaultO2Config()

	}

	// Initialize core services.

	catalogService := ims.NewCatalogService()

	inventoryService := ims.NewInventoryService(kubeClient, clientset)

	lifecycleService := ims.NewLifecycleService()

	subscriptionService := ims.NewSubscriptionService()

	imsService := ims.NewIMSService(catalogService, inventoryService, lifecycleService, subscriptionService)

	// Initialize cloud providers.

	providerManager, err := initializeProviders(config.Providers, kubeClient, clientset)

	if err != nil {

		return nil, fmt.Errorf("failed to initialize cloud providers: %w", err)

	}

	// Set up circuit breaker configuration.

	if config.CircuitBreakerConfig == nil {

		config.CircuitBreakerConfig = &llm.CircuitBreakerConfig{

			FailureThreshold: 5,

			FailureRate: 0.5,

			MinimumRequestCount: 10,

			Timeout: 30 * time.Second,

			HalfOpenTimeout: 60 * time.Second,

			SuccessThreshold: 3,

			HalfOpenMaxRequests: 5,

			ResetTimeout: 60 * time.Second,

			SlidingWindowSize: 100,

			EnableHealthCheck: true,

			HealthCheckInterval: 30 * time.Second,

			HealthCheckTimeout: 10 * time.Second,
		}

	}

	// Set up retry configuration.

	if config.RetryConfig == nil {

		config.RetryConfig = &RetryConfig{

			MaxRetries: 3,

			InitialDelay: 1 * time.Second,

			MaxDelay: 30 * time.Second,

			BackoffFactor: 2.0,

			Jitter: true,

			RetryableErrors: []string{

				"connection refused",

				"timeout",

				"temporary failure",

				"service unavailable",
			},
		}

	}

	// Create circuit breaker.

	circuitBreaker := llm.NewCircuitBreaker("o2-adaptor", config.CircuitBreakerConfig)

	// Initialize monitoring context.

	monitoringCtx, monitoringCancel := context.WithCancel(context.Background())

	adaptor := &O2Adaptor{

		kubeClient: kubeClient,

		clientset: clientset,

		config: config,

		imsService: imsService,

		catalogService: catalogService,

		inventoryService: inventoryService,

		lifecycleService: lifecycleService,

		subscriptionService: subscriptionService,

		providers: providerManager,

		circuitBreaker: circuitBreaker,

		retryConfig: config.RetryConfig,

		resourcePools: make(map[string]*models.ResourcePool),

		deployments: make(map[string]*models.Deployment),

		subscriptions: make(map[string]*models.Subscription),

		monitoringCtx: monitoringCtx,

		monitoringCancel: monitoringCancel,
	}

	// Start background monitoring services.

	if config.MonitoringConfig.EnableMetrics {

		go adaptor.startResourceMonitoring()

		go adaptor.startHealthMonitoring()

	}

	return adaptor, nil

}

// DefaultO2Config returns default O2 configuration.

func DefaultO2Config() *O2Config {

	return &O2Config{

		Namespace: "o-ran-o2",

		ServiceAccount: "o2-adaptor",

		DefaultProvider: "kubernetes",

		Providers: map[string]*ProviderConfig{

			"kubernetes": {

				Type: "kubernetes",

				Enabled: true,

				Config: map[string]string{

					"in_cluster": "true",
				},
			},
		},

		IMSConfiguration: &IMSConfig{

			SystemName: "Nephoran O2 IMS",

			SystemDescription: "O-RAN O2 Infrastructure Management Services",

			SystemVersion: "1.0.0",

			SupportedAPIVersions: []string{"1.0.0"},

			MaxResourcePools: 100,

			MaxResources: 10000,

			MaxDeployments: 1000,
		},

		APIServerConfig: &APIServerConfig{

			Enabled: true,

			Port: 8082,

			MetricsPort: 8083,

			EnableOpenAPI: true,

			EnableCORS: true,

			RequestTimeout: 30 * time.Second,

			MaxRequestSize: 10 * 1024 * 1024, // 10MB

		},

		MonitoringConfig: &MonitoringConfig{

			EnableMetrics: true,

			EnableTracing: true,

			MetricsInterval: 30 * time.Second,

			RetentionPeriod: 24 * time.Hour,

			AlertingEnabled: true,
		},

		AuthConfig: &AuthConfig{

			AuthMode: "token",

			RBACEnabled: true,

			TokenValidation: &TokenConfig{

				Enabled: true,

				TokenExpiry: 24 * time.Hour,
			},
		},
	}

}

// initializeProviders initializes and validates cloud providers.

func initializeProviders(providerConfigs map[string]*ProviderConfig, kubeClient client.Client, clientset kubernetes.Interface) (map[string]providers.CloudProvider, error) {

	providerManager := make(map[string]providers.CloudProvider)

	for name, config := range providerConfigs {

		if !config.Enabled {

			continue

		}

		// Convert ProviderConfig to ProviderConfiguration.

		providerConfig := &providers.ProviderConfiguration{

			Name: name,

			Type: config.Type,

			Region: config.Region,

			Credentials: config.Credentials,

			Enabled: config.Enabled,
		}

		var provider providers.CloudProvider

		var err error

		switch config.Type {

		case "kubernetes":

			provider, err = providers.NewKubernetesProvider(kubeClient, clientset, config.Config)

		case "aws":

			provider, err = providers.NewAWSProvider(providerConfig)

		case "azure":

			provider, err = providers.NewAzureProvider(providerConfig)

		case "gcp":

			provider, err = providers.NewGCPProvider(providerConfig)

		default:

			return nil, fmt.Errorf("unsupported provider type: %s", config.Type)

		}

		if err != nil {

			return nil, fmt.Errorf("failed to initialize provider %s: %w", name, err)

		}

		providerManager[name] = provider

	}

	if len(providerManager) == 0 {

		return nil, fmt.Errorf("no enabled cloud providers configured")

	}

	return providerManager, nil

}

// Health and lifecycle management.

// Shutdown gracefully shuts down the O2 adaptor.

func (a *O2Adaptor) Shutdown(ctx context.Context) error {

	logger := log.FromContext(ctx)

	logger.Info("shutting down O2 adaptor")

	// Cancel monitoring services.

	a.monitoringCancel()

	// Close circuit breaker.

	if a.circuitBreaker != nil {

		// Circuit breaker cleanup would be implemented here.

	}

	// Close provider connections.

	for name, provider := range a.providers {

		if err := provider.Close(); err != nil {

			logger.Error(err, "failed to close provider", "provider", name)

		}

	}

	logger.Info("O2 adaptor shutdown completed")

	return nil

}

// GetSystemInfo returns O2 IMS system information.

func (a *O2Adaptor) GetSystemInfo(ctx context.Context) (*models.SystemInfo, error) {

	return &models.SystemInfo{

		Name: a.config.IMSConfiguration.SystemName,

		Description: a.config.IMSConfiguration.SystemDescription,

		Version: a.config.IMSConfiguration.SystemVersion,

		APIVersions: a.config.IMSConfiguration.SupportedAPIVersions,

		SupportedResourceTypes: a.getSupportedResourceTypes(),

		Extensions: a.getSystemExtensions(),

		Timestamp: time.Now(),
	}, nil

}

// Private helper methods.

func (a *O2Adaptor) getSupportedResourceTypes() []string {

	return []string{

		"compute",

		"storage",

		"network",

		"accelerator",

		"deployment",

		"service",

		"configmap",

		"secret",

		"persistent_volume",

		"persistent_volume_claim",
	}

}

func (a *O2Adaptor) getSystemExtensions() map[string]interface{} {

	return map[string]interface{}{

		"multi_cloud": true,

		"kubernetes_native": true,

		"auto_scaling": true,

		"fault_tolerance": true,

		"monitoring": a.config.MonitoringConfig.EnableMetrics,

		"tracing": a.config.MonitoringConfig.EnableTracing,

		"rbac": a.config.AuthConfig.RBACEnabled,

		"provider_count": len(a.providers),
	}

}

// Background monitoring services.

func (a *O2Adaptor) startResourceMonitoring() {

	logger := log.FromContext(a.monitoringCtx)

	logger.Info("starting resource monitoring service")

	ticker := time.NewTicker(a.config.MonitoringConfig.MetricsInterval)

	defer ticker.Stop()

	for {

		select {

		case <-a.monitoringCtx.Done():

			logger.Info("stopping resource monitoring service")

			return

		case <-ticker.C:

			a.collectResourceMetrics()

		}

	}

}

func (a *O2Adaptor) startHealthMonitoring() {

	logger := log.FromContext(a.monitoringCtx)

	logger.Info("starting health monitoring service")

	ticker := time.NewTicker(30 * time.Second) // Fixed health check interval

	defer ticker.Stop()

	for {

		select {

		case <-a.monitoringCtx.Done():

			logger.Info("stopping health monitoring service")

			return

		case <-ticker.C:

			a.performHealthChecks()

		}

	}

}

func (a *O2Adaptor) collectResourceMetrics() {

	logger := log.FromContext(a.monitoringCtx)

	// Collect metrics from all providers.

	for name, provider := range a.providers {

		metrics, err := provider.GetMetrics(a.monitoringCtx)

		if err != nil {

			logger.Error(err, "failed to collect metrics from provider", "provider", name)

			continue

		}

		// Process and store metrics.

		// This would integrate with Prometheus or other monitoring systems.

		logger.V(1).Info("collected provider metrics", "provider", name, "metrics", len(metrics))

	}

	// Collect O2-specific metrics.

	a.mutex.RLock()

	resourcePoolCount := len(a.resourcePools)

	deploymentCount := len(a.deployments)

	subscriptionCount := len(a.subscriptions)

	a.mutex.RUnlock()

	logger.V(1).Info("O2 system metrics",

		"resource_pools", resourcePoolCount,

		"deployments", deploymentCount,

		"subscriptions", subscriptionCount)

}

func (a *O2Adaptor) performHealthChecks() {

	logger := log.FromContext(a.monitoringCtx)

	// Check provider health.

	for name, provider := range a.providers {

		if err := provider.HealthCheck(a.monitoringCtx); err != nil {

			logger.Error(err, "provider health check failed", "provider", name)

			// Could trigger alerts or automatic remediation here.

		}

	}

	// Check circuit breaker status.

	if a.circuitBreaker != nil {

		stats := a.circuitBreaker.GetStats()

		if failureRate, ok := stats["failure_rate"].(float64); ok && failureRate > 0.5 {

			logger.Info("high failure rate detected in circuit breaker", "failure_rate", failureRate)

		}

	}

}
