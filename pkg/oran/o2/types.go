// Package o2 implements O-RAN Infrastructure Management Service (IMS) compliant with O-RAN.WG6.O2ims-Interface-v01.01 specification
// This package provides complete O2 IMS APIs for cloud infrastructure and VNF lifecycle management
package o2

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
)

// O2IMSInterface represents the three main O-RAN O2 IMS interface capabilities
type O2IMSInterface string

const (
	O2InfrastructureInventoryInterface     O2IMSInterface = "O2IMS-InfrastructureInventory"     // Infrastructure inventory management
	O2InfrastructureMonitoringInterface   O2IMSInterface = "O2IMS-InfrastructureMonitoring"   // Infrastructure monitoring and alarming
	O2InfrastructureProvisioningInterface O2IMSInterface = "O2IMS-InfrastructureProvisioning" // Infrastructure provisioning and lifecycle
)

// O2IMSVersion represents supported O2 IMS API versions
type O2IMSVersion string

const (
	O2IMSVersionV1_0 O2IMSVersion = "v1.0" // O2 IMS API version 1.0
)

// CloudProvider represents different cloud infrastructure providers
type CloudProvider string

const (
	CloudProviderKubernetes CloudProvider = "kubernetes"
	CloudProviderOpenStack CloudProvider = "openstack"
	CloudProviderAWS        CloudProvider = "aws"
	CloudProviderAzure      CloudProvider = "azure"
	CloudProviderGCP        CloudProvider = "gcp"
	CloudProviderVMware     CloudProvider = "vmware"
	CloudProviderEdge       CloudProvider = "edge"
)

// O2IMSService defines the core O2 IMS service interface following O-RAN specification
type O2IMSService interface {
	// Infrastructure Inventory Management
	GetResourcePools(ctx context.Context, filter *models.ResourcePoolFilter) ([]*models.ResourcePool, error)
	GetResourcePool(ctx context.Context, resourcePoolID string) (*models.ResourcePool, error)
	CreateResourcePool(ctx context.Context, req *models.CreateResourcePoolRequest) (*models.ResourcePool, error)
	UpdateResourcePool(ctx context.Context, resourcePoolID string, req *models.UpdateResourcePoolRequest) (*models.ResourcePool, error)
	DeleteResourcePool(ctx context.Context, resourcePoolID string) error

	GetResourceTypes(ctx context.Context, filter *models.ResourceTypeFilter) ([]*models.ResourceType, error)
	GetResourceType(ctx context.Context, resourceTypeID string) (*models.ResourceType, error)
	CreateResourceType(ctx context.Context, resourceType *models.ResourceType) (*models.ResourceType, error)
	UpdateResourceType(ctx context.Context, resourceTypeID string, resourceType *models.ResourceType) (*models.ResourceType, error)
	DeleteResourceType(ctx context.Context, resourceTypeID string) error

	GetResources(ctx context.Context, filter *models.ResourceFilter) ([]*models.Resource, error)
	GetResource(ctx context.Context, resourceID string) (*models.Resource, error)
	CreateResource(ctx context.Context, req *models.CreateResourceRequest) (*models.Resource, error)
	UpdateResource(ctx context.Context, resourceID string, req *models.UpdateResourceRequest) (*models.Resource, error)
	DeleteResource(ctx context.Context, resourceID string) error

	// Deployment Template Management
	GetDeploymentTemplates(ctx context.Context, filter *DeploymentTemplateFilter) ([]*DeploymentTemplate, error)
	GetDeploymentTemplate(ctx context.Context, templateID string) (*DeploymentTemplate, error)
	CreateDeploymentTemplate(ctx context.Context, template *DeploymentTemplate) (*DeploymentTemplate, error)
	UpdateDeploymentTemplate(ctx context.Context, templateID string, template *DeploymentTemplate) (*DeploymentTemplate, error)
	DeleteDeploymentTemplate(ctx context.Context, templateID string) error

	// Deployment Management
	GetDeployments(ctx context.Context, filter *DeploymentFilter) ([]*Deployment, error)
	GetDeployment(ctx context.Context, deploymentID string) (*Deployment, error)
	CreateDeployment(ctx context.Context, req *CreateDeploymentRequest) (*Deployment, error)
	UpdateDeployment(ctx context.Context, deploymentID string, req *UpdateDeploymentRequest) (*Deployment, error)
	DeleteDeployment(ctx context.Context, deploymentID string) error

	// Subscription and Event Management
	CreateSubscription(ctx context.Context, sub *Subscription) (*Subscription, error)
	GetSubscription(ctx context.Context, subscriptionID string) (*Subscription, error)
	GetSubscriptions(ctx context.Context, filter *SubscriptionFilter) ([]*Subscription, error)
	UpdateSubscription(ctx context.Context, subscriptionID string, sub *Subscription) (*Subscription, error)
	DeleteSubscription(ctx context.Context, subscriptionID string) error

	// Infrastructure Monitoring and Health
	GetResourceHealth(ctx context.Context, resourceID string) (*ResourceHealth, error)
	GetResourceAlarms(ctx context.Context, resourceID string, filter *AlarmFilter) ([]*Alarm, error)
	GetResourceMetrics(ctx context.Context, resourceID string, filter *MetricsFilter) (*MetricsData, error)

	// Multi-Cloud Provider Management
	RegisterCloudProvider(ctx context.Context, provider *CloudProviderConfig) error
	GetCloudProviders(ctx context.Context) ([]*CloudProviderConfig, error)
	GetCloudProvider(ctx context.Context, providerID string) (*CloudProviderConfig, error)
	UpdateCloudProvider(ctx context.Context, providerID string, provider *CloudProviderConfig) error
	RemoveCloudProvider(ctx context.Context, providerID string) error
}

// ResourceManager defines the interface for infrastructure resource lifecycle operations
type ResourceManager interface {
	// Resource lifecycle operations
	ProvisionResource(ctx context.Context, req *ProvisionResourceRequest) (*models.Resource, error)
	ConfigureResource(ctx context.Context, resourceID string, config *runtime.RawExtension) error
	ScaleResource(ctx context.Context, resourceID string, req *ScaleResourceRequest) error
	MigrateResource(ctx context.Context, resourceID string, req *MigrateResourceRequest) error
	BackupResource(ctx context.Context, resourceID string, req *BackupResourceRequest) (*BackupInfo, error)
	RestoreResource(ctx context.Context, resourceID string, backupID string) error
	TerminateResource(ctx context.Context, resourceID string) error

	// Resource discovery and synchronization
	DiscoverResources(ctx context.Context, providerID string) ([]*models.Resource, error)
	SyncResourceState(ctx context.Context, resourceID string) (*models.Resource, error)
	ValidateResourceConfiguration(ctx context.Context, resourceID string, config *runtime.RawExtension) error
}

// InventoryService defines the interface for infrastructure inventory tracking
type InventoryService interface {
	// Infrastructure discovery
	DiscoverInfrastructure(ctx context.Context, provider CloudProvider) (*InfrastructureDiscovery, error)
	UpdateInventory(ctx context.Context, updates []*InventoryUpdate) error
	
	// Asset management
	TrackAsset(ctx context.Context, asset *Asset) error
	GetAsset(ctx context.Context, assetID string) (*Asset, error)
	UpdateAsset(ctx context.Context, assetID string, asset *Asset) error
	
	// Capacity management
	CalculateCapacity(ctx context.Context, poolID string) (*models.ResourceCapacity, error)
	PredictCapacity(ctx context.Context, poolID string, horizon time.Duration) (*CapacityPrediction, error)
}

// MonitoringService defines the interface for infrastructure monitoring
type MonitoringService interface {
	// Health monitoring
	CheckResourceHealth(ctx context.Context, resourceID string) (*ResourceHealth, error)
	MonitorResourceHealth(ctx context.Context, resourceID string, callback HealthCallback) error
	
	// Metrics collection
	CollectMetrics(ctx context.Context, resourceID string, metricNames []string) (*MetricsData, error)
	StartMetricsCollection(ctx context.Context, resourceID string, config *MetricsCollectionConfig) (string, error)
	StopMetricsCollection(ctx context.Context, collectionID string) error
	
	// Alarm management
	RaiseAlarm(ctx context.Context, alarm *Alarm) error
	ClearAlarm(ctx context.Context, alarmID string) error
	GetAlarms(ctx context.Context, filter *AlarmFilter) ([]*Alarm, error)
	
	// Event handling
	EmitEvent(ctx context.Context, event *InfrastructureEvent) error
	ProcessEvent(ctx context.Context, event *InfrastructureEvent) error
}

// O2IMSConfig holds configuration for the O2 IMS service
type O2IMSConfig struct {
	// Service configuration
	Port                    int                        `json:"port" validate:"required,min=1,max=65535"`
	Host                    string                     `json:"host"`
	TLSEnabled              bool                       `json:"tls_enabled"`
	CertFile                string                     `json:"cert_file"`
	KeyFile                 string                     `json:"key_file"`
	ReadTimeout             time.Duration              `json:"read_timeout"`
	WriteTimeout            time.Duration              `json:"write_timeout"`
	IdleTimeout             time.Duration              `json:"idle_timeout"`
	MaxHeaderBytes          int                        `json:"max_header_bytes"`

	// Cloud provider configurations
	CloudProviders          map[string]*CloudProviderConfig `json:"cloud_providers"`
	DefaultProvider         string                     `json:"default_provider"`

	// Database configuration
	DatabaseURL             string                     `json:"database_url"`
	DatabaseType            string                     `json:"database_type"` // postgres, mysql, sqlite
	
	// Authentication configuration
	AuthenticationConfig    *AuthenticationConfig      `json:"authentication_config"`
	
	// Monitoring configuration
	MetricsConfig           *MetricsConfig             `json:"metrics_config"`
	HealthCheckConfig       *HealthCheckConfig         `json:"health_check_config"`
	
	// Event and notification configuration
	NotificationConfig      *NotificationConfig        `json:"notification_config"`
	
	// Security configuration
	SecurityConfig          *SecurityConfig            `json:"security_config"`
	
	// Resource management configuration
	ResourceConfig          *ResourceManagementConfig  `json:"resource_config"`
	
	// Logging configuration
	Logger                  *logging.StructuredLogger  `json:"-"`
}

// CloudProviderConfig represents configuration for a cloud infrastructure provider
type CloudProviderConfig struct {
	ProviderID      string                 `json:"provider_id"`
	Name            string                 `json:"name"`
	Type            CloudProvider          `json:"type"`
	Description     string                 `json:"description,omitempty"`
	Endpoint        string                 `json:"endpoint"`
	Region          string                 `json:"region,omitempty"`
	
	// Authentication
	AuthMethod      string                 `json:"auth_method"` // oauth2, apikey, certificate, basic
	AuthConfig      map[string]interface{} `json:"auth_config"`
	
	// Connection settings
	Timeout         time.Duration          `json:"timeout"`
	MaxRetries      int                    `json:"max_retries"`
	RetryInterval   time.Duration          `json:"retry_interval"`
	
	// TLS configuration
	TLSConfig       *oran.TLSConfig        `json:"tls_config"`
	
	// Provider-specific configuration
	Properties      map[string]interface{} `json:"properties"`
	
	// Status and monitoring
	Enabled         bool                   `json:"enabled"`
	Status          string                 `json:"status"` // ACTIVE, INACTIVE, ERROR, MAINTENANCE
	LastHealthCheck time.Time              `json:"last_health_check"`
	
	// Metadata
	Tags            map[string]string      `json:"tags"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// AuthenticationConfig configures authentication for O2 IMS
type AuthenticationConfig struct {
	Enabled          bool              `json:"enabled"`
	Method           string            `json:"method"` // oauth2, jwt, basic, apikey
	JWTSecret        string            `json:"jwt_secret,omitempty"`
	TokenValidation  bool              `json:"token_validation"`
	AllowedIssuers   []string          `json:"allowed_issuers,omitempty"`
	RequiredClaims   map[string]string `json:"required_claims,omitempty"`
	TokenExpiry      time.Duration     `json:"token_expiry"`
	RefreshEnabled   bool              `json:"refresh_enabled"`
}

// MetricsConfig configures metrics collection for O2 IMS
type MetricsConfig struct {
	Enabled         bool          `json:"enabled"`
	Endpoint        string        `json:"endpoint"`
	Namespace       string        `json:"namespace"`
	Subsystem       string        `json:"subsystem"`
	CollectionInterval time.Duration `json:"collection_interval"`
	RetentionPeriod time.Duration `json:"retention_period"`
	PrometheusConfig *PrometheusConfig `json:"prometheus_config,omitempty"`
}

// PrometheusConfig configures Prometheus integration
type PrometheusConfig struct {
	Enabled         bool   `json:"enabled"`
	PushGateway     string `json:"push_gateway,omitempty"`
	JobName         string `json:"job_name"`
	Instance        string `json:"instance"`
}

// HealthCheckConfig configures health checking behavior
type HealthCheckConfig struct {
	Enabled          bool          `json:"enabled"`
	CheckInterval    time.Duration `json:"check_interval"`
	Timeout          time.Duration `json:"timeout"`
	FailureThreshold int           `json:"failure_threshold"`
	SuccessThreshold int           `json:"success_threshold"`
	DeepHealthCheck  bool          `json:"deep_health_check"`
}

// NotificationConfig configures event notifications
type NotificationConfig struct {
	Enabled         bool                    `json:"enabled"`
	WebhookURL      string                  `json:"webhook_url,omitempty"`
	EventTypes      []string                `json:"event_types"`
	RetryPolicy     *RetryPolicy            `json:"retry_policy"`
	WebhookConfig   *WebhookConfig          `json:"webhook_config,omitempty"`
	EmailConfig     *EmailConfig            `json:"email_config,omitempty"`
	SlackConfig     *SlackConfig            `json:"slack_config,omitempty"`
}

// RetryPolicy defines retry behavior for notifications
type RetryPolicy struct {
	MaxRetries    int           `json:"max_retries"`
	RetryInterval time.Duration `json:"retry_interval"`
	BackoffFactor float64       `json:"backoff_factor"`
	MaxInterval   time.Duration `json:"max_interval"`
}

// WebhookConfig configures webhook notifications
type WebhookConfig struct {
	URL             string            `json:"url"`
	Method          string            `json:"method"`
	Headers         map[string]string `json:"headers,omitempty"`
	AuthToken       string            `json:"auth_token,omitempty"`
	TLSConfig       *oran.TLSConfig   `json:"tls_config,omitempty"`
}

// EmailConfig configures email notifications
type EmailConfig struct {
	SMTPServer   string   `json:"smtp_server"`
	SMTPPort     int      `json:"smtp_port"`
	Username     string   `json:"username"`
	Password     string   `json:"password"`
	FromAddress  string   `json:"from_address"`
	ToAddresses  []string `json:"to_addresses"`
	Subject      string   `json:"subject"`
	TLSEnabled   bool     `json:"tls_enabled"`
}

// SlackConfig configures Slack notifications
type SlackConfig struct {
	WebhookURL string `json:"webhook_url"`
	Channel    string `json:"channel"`
	Username   string `json:"username"`
	IconEmoji  string `json:"icon_emoji,omitempty"`
}

// SecurityConfig configures security settings
type SecurityConfig struct {
	EnableCSRF          bool              `json:"enable_csrf"`
	CSRFTokenLength     int               `json:"csrf_token_length"`
	CORSEnabled         bool              `json:"cors_enabled"`
	CORSAllowedOrigins  []string          `json:"cors_allowed_origins"`
	CORSAllowedMethods  []string          `json:"cors_allowed_methods"`
	CORSAllowedHeaders  []string          `json:"cors_allowed_headers"`
	RateLimitConfig     *RateLimitConfig  `json:"rate_limit_config"`
	InputValidation     *ValidationConfig `json:"input_validation"`
	AuditLogging        bool              `json:"audit_logging"`
}

// RateLimitConfig configures request rate limiting
type RateLimitConfig struct {
	Enabled        bool          `json:"enabled"`
	RequestsPerMin int           `json:"requests_per_min"`
	BurstSize      int           `json:"burst_size"`
	WindowSize     time.Duration `json:"window_size"`
	KeyFunc        string        `json:"key_func"` // ip, user, token
}

// ValidationConfig configures input validation
type ValidationConfig struct {
	EnableSchemaValidation bool `json:"enable_schema_validation"`
	StrictValidation      bool `json:"strict_validation"`
	MaxRequestSize        int  `json:"max_request_size"`
	SanitizeInput         bool `json:"sanitize_input"`
}

// ResourceManagementConfig configures resource management behavior
type ResourceManagementConfig struct {
	DefaultTimeout          time.Duration `json:"default_timeout"`
	MaxConcurrentOperations int           `json:"max_concurrent_operations"`
	ResourceCacheSize       int           `json:"resource_cache_size"`
	ResourceCacheTTL        time.Duration `json:"resource_cache_ttl"`
	AutoDiscoveryEnabled    bool          `json:"auto_discovery_enabled"`
	AutoDiscoveryInterval   time.Duration `json:"auto_discovery_interval"`
	StateReconcileInterval  time.Duration `json:"state_reconcile_interval"`
}

// O2IMSHandler defines the HTTP handler interface for O2 IMS endpoints
type O2IMSHandler interface {
	// Infrastructure Inventory Handlers
	HandleGetResourcePools(w http.ResponseWriter, r *http.Request)
	HandleGetResourcePool(w http.ResponseWriter, r *http.Request)
	HandleCreateResourcePool(w http.ResponseWriter, r *http.Request)
	HandleUpdateResourcePool(w http.ResponseWriter, r *http.Request)
	HandleDeleteResourcePool(w http.ResponseWriter, r *http.Request)

	HandleGetResourceTypes(w http.ResponseWriter, r *http.Request)
	HandleGetResourceType(w http.ResponseWriter, r *http.Request)
	HandleCreateResourceType(w http.ResponseWriter, r *http.Request)
	HandleUpdateResourceType(w http.ResponseWriter, r *http.Request)
	HandleDeleteResourceType(w http.ResponseWriter, r *http.Request)

	HandleGetResources(w http.ResponseWriter, r *http.Request)
	HandleGetResource(w http.ResponseWriter, r *http.Request)
	HandleCreateResource(w http.ResponseWriter, r *http.Request)
	HandleUpdateResource(w http.ResponseWriter, r *http.Request)
	HandleDeleteResource(w http.ResponseWriter, r *http.Request)

	// Deployment Template Handlers
	HandleGetDeploymentTemplates(w http.ResponseWriter, r *http.Request)
	HandleGetDeploymentTemplate(w http.ResponseWriter, r *http.Request)
	HandleCreateDeploymentTemplate(w http.ResponseWriter, r *http.Request)
	HandleUpdateDeploymentTemplate(w http.ResponseWriter, r *http.Request)
	HandleDeleteDeploymentTemplate(w http.ResponseWriter, r *http.Request)

	// Deployment Management Handlers
	HandleGetDeployments(w http.ResponseWriter, r *http.Request)
	HandleGetDeployment(w http.ResponseWriter, r *http.Request)
	HandleCreateDeployment(w http.ResponseWriter, r *http.Request)
	HandleUpdateDeployment(w http.ResponseWriter, r *http.Request)
	HandleDeleteDeployment(w http.ResponseWriter, r *http.Request)

	// Subscription Handlers
	HandleCreateSubscription(w http.ResponseWriter, r *http.Request)
	HandleGetSubscription(w http.ResponseWriter, r *http.Request)
	HandleGetSubscriptions(w http.ResponseWriter, r *http.Request)
	HandleUpdateSubscription(w http.ResponseWriter, r *http.Request)
	HandleDeleteSubscription(w http.ResponseWriter, r *http.Request)

	// Monitoring and Health Handlers
	HandleGetResourceHealth(w http.ResponseWriter, r *http.Request)
	HandleGetResourceAlarms(w http.ResponseWriter, r *http.Request)
	HandleGetResourceMetrics(w http.ResponseWriter, r *http.Request)

	// Cloud Provider Management Handlers
	HandleRegisterCloudProvider(w http.ResponseWriter, r *http.Request)
	HandleGetCloudProviders(w http.ResponseWriter, r *http.Request)
	HandleGetCloudProvider(w http.ResponseWriter, r *http.Request)
	HandleUpdateCloudProvider(w http.ResponseWriter, r *http.Request)
	HandleRemoveCloudProvider(w http.ResponseWriter, r *http.Request)

	// Service information handlers
	HandleGetServiceInfo(w http.ResponseWriter, r *http.Request)
	HandleHealthCheck(w http.ResponseWriter, r *http.Request)
}

// O2IMSStorage defines the storage interface for O2 IMS data persistence
type O2IMSStorage interface {
	// Resource Pool storage
	StoreResourcePool(ctx context.Context, pool *models.ResourcePool) error
	GetResourcePool(ctx context.Context, poolID string) (*models.ResourcePool, error)
	ListResourcePools(ctx context.Context, filter *models.ResourcePoolFilter) ([]*models.ResourcePool, error)
	UpdateResourcePool(ctx context.Context, poolID string, updates map[string]interface{}) error
	DeleteResourcePool(ctx context.Context, poolID string) error

	// Resource Type storage
	StoreResourceType(ctx context.Context, resourceType *models.ResourceType) error
	GetResourceType(ctx context.Context, typeID string) (*models.ResourceType, error)
	ListResourceTypes(ctx context.Context, filter *models.ResourceTypeFilter) ([]*models.ResourceType, error)
	UpdateResourceType(ctx context.Context, typeID string, resourceType *models.ResourceType) error
	DeleteResourceType(ctx context.Context, typeID string) error

	// Resource storage
	StoreResource(ctx context.Context, resource *models.Resource) error
	GetResource(ctx context.Context, resourceID string) (*models.Resource, error)
	ListResources(ctx context.Context, filter *models.ResourceFilter) ([]*models.Resource, error)
	UpdateResource(ctx context.Context, resourceID string, updates map[string]interface{}) error
	DeleteResource(ctx context.Context, resourceID string) error

	// Deployment Template storage
	StoreDeploymentTemplate(ctx context.Context, template *DeploymentTemplate) error
	GetDeploymentTemplate(ctx context.Context, templateID string) (*DeploymentTemplate, error)
	ListDeploymentTemplates(ctx context.Context, filter *DeploymentTemplateFilter) ([]*DeploymentTemplate, error)
	UpdateDeploymentTemplate(ctx context.Context, templateID string, template *DeploymentTemplate) error
	DeleteDeploymentTemplate(ctx context.Context, templateID string) error

	// Deployment storage
	StoreDeployment(ctx context.Context, deployment *Deployment) error
	GetDeployment(ctx context.Context, deploymentID string) (*Deployment, error)
	ListDeployments(ctx context.Context, filter *DeploymentFilter) ([]*Deployment, error)
	UpdateDeployment(ctx context.Context, deploymentID string, updates map[string]interface{}) error
	DeleteDeployment(ctx context.Context, deploymentID string) error

	// Subscription storage
	StoreSubscription(ctx context.Context, subscription *Subscription) error
	GetSubscription(ctx context.Context, subscriptionID string) (*Subscription, error)
	ListSubscriptions(ctx context.Context, filter *SubscriptionFilter) ([]*Subscription, error)
	UpdateSubscription(ctx context.Context, subscriptionID string, subscription *Subscription) error
	DeleteSubscription(ctx context.Context, subscriptionID string) error

	// Cloud Provider storage
	StoreCloudProvider(ctx context.Context, provider *CloudProviderConfig) error
	GetCloudProvider(ctx context.Context, providerID string) (*CloudProviderConfig, error)
	ListCloudProviders(ctx context.Context) ([]*CloudProviderConfig, error)
	UpdateCloudProvider(ctx context.Context, providerID string, provider *CloudProviderConfig) error
	DeleteCloudProvider(ctx context.Context, providerID string) error

	// Health and status tracking
	UpdateResourceStatus(ctx context.Context, resourceID string, status *models.ResourceStatus) error
	GetResourceHistory(ctx context.Context, resourceID string, limit int) ([]*ResourceHistoryEntry, error)

	// Transaction support
	BeginTransaction(ctx context.Context) (interface{}, error)
	CommitTransaction(ctx context.Context, tx interface{}) error
	RollbackTransaction(ctx context.Context, tx interface{}) error
}

// RequestContext holds request-specific context information for O2 IMS
type RequestContext struct {
	RequestID       string                 `json:"request_id"`
	UserID          string                 `json:"user_id,omitempty"`
	TenantID        string                 `json:"tenant_id,omitempty"`
	UserAgent       string                 `json:"user_agent,omitempty"`
	RemoteAddr      string                 `json:"remote_addr,omitempty"`
	Method          string                 `json:"method"`
	Path            string                 `json:"path"`
	Headers         map[string][]string    `json:"headers,omitempty"`
	QueryParams     map[string][]string    `json:"query_params,omitempty"`
	Authentication  map[string]interface{} `json:"authentication,omitempty"`
	StartTime       time.Time              `json:"start_time"`
	Trace           *TraceContext          `json:"trace,omitempty"`
}

// TraceContext holds distributed tracing information
type TraceContext struct {
	TraceID  string `json:"trace_id"`
	SpanID   string `json:"span_id"`
	ParentID string `json:"parent_id,omitempty"`
}

// ResponseInfo holds response information for logging and monitoring
type ResponseInfo struct {
	StatusCode    int                    `json:"status_code"`
	ContentLength int64                  `json:"content_length"`
	Duration      time.Duration          `json:"duration"`
	Headers       map[string][]string    `json:"headers,omitempty"`
	Errors        []string               `json:"errors,omitempty"`
	Warnings      []string               `json:"warnings,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// HealthCheck represents health status information for O2 IMS
type HealthCheck struct {
	Status      string                 `json:"status" validate:"required,oneof=UP DOWN DEGRADED"`
	Timestamp   time.Time              `json:"timestamp"`
	Version     string                 `json:"version,omitempty"`
	Uptime      time.Duration          `json:"uptime,omitempty"`
	Components  map[string]interface{} `json:"components,omitempty"`
	Checks      []ComponentCheck       `json:"checks,omitempty"`
	Services    []ServiceStatus        `json:"services,omitempty"`
	Resources   *ResourceHealthSummary `json:"resources,omitempty"`
}

// ComponentCheck represents individual component health status
type ComponentCheck struct {
	Name        string                 `json:"name" validate:"required"`
	Status      string                 `json:"status" validate:"required,oneof=UP DOWN DEGRADED"`
	Message     string                 `json:"message,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Duration    time.Duration          `json:"duration,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	CheckType   string                 `json:"check_type,omitempty"` // connectivity, resource, dependency
}

// ServiceStatus represents the status of external service dependencies
type ServiceStatus struct {
	ServiceName string        `json:"service_name"`
	Status      string        `json:"status"`
	Endpoint    string        `json:"endpoint,omitempty"`
	LastCheck   time.Time     `json:"last_check"`
	Latency     time.Duration `json:"latency,omitempty"`
	Error       string        `json:"error,omitempty"`
}

// ResourceHealthSummary provides a summary of resource health across the infrastructure
type ResourceHealthSummary struct {
	TotalResources    int `json:"total_resources"`
	HealthyResources  int `json:"healthy_resources"`
	DegradedResources int `json:"degraded_resources"`
	UnhealthyResources int `json:"unhealthy_resources"`
	UnknownResources  int `json:"unknown_resources"`
}

// Common HTTP status codes used in O2 IMS interface
const (
	StatusOK                  = http.StatusOK                  // 200
	StatusCreated             = http.StatusCreated             // 201
	StatusAccepted            = http.StatusAccepted            // 202
	StatusNoContent           = http.StatusNoContent           // 204
	StatusBadRequest          = http.StatusBadRequest          // 400
	StatusUnauthorized        = http.StatusUnauthorized        // 401
	StatusForbidden           = http.StatusForbidden           // 403
	StatusNotFound            = http.StatusNotFound            // 404
	StatusMethodNotAllowed    = http.StatusMethodNotAllowed    // 405
	StatusNotAcceptable       = http.StatusNotAcceptable       // 406
	StatusConflict            = http.StatusConflict            // 409
	StatusUnprocessableEntity = http.StatusUnprocessableEntity // 422
	StatusInternalServerError = http.StatusInternalServerError // 500
	StatusNotImplemented      = http.StatusNotImplemented      // 501
	StatusBadGateway          = http.StatusBadGateway          // 502
	StatusServiceUnavailable  = http.StatusServiceUnavailable  // 503
	StatusGatewayTimeout      = http.StatusGatewayTimeout      // 504
)

// Common MIME types for O2 IMS interfaces
const (
	ContentTypeJSON        = "application/json"
	ContentTypeProblemJSON = "application/problem+json"
	ContentTypeTextPlain   = "text/plain"
	ContentTypeXML         = "application/xml"
	ContentTypeYAML        = "application/yaml"
)

// DefaultO2IMSConfig returns a default configuration for the O2 IMS service
func DefaultO2IMSConfig() *O2IMSConfig {
	return &O2IMSConfig{
		Port:           8090,
		Host:           "0.0.0.0",
		TLSEnabled:     false,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
		
		CloudProviders:  make(map[string]*CloudProviderConfig),
		DefaultProvider: "kubernetes",
		
		DatabaseType:    "sqlite",
		
		AuthenticationConfig: &AuthenticationConfig{
			Enabled: false,
			Method:  "jwt",
			TokenExpiry: 24 * time.Hour,
			RefreshEnabled: true,
		},
		
		MetricsConfig: &MetricsConfig{
			Enabled:            true,
			Endpoint:           "/metrics",
			Namespace:          "nephoran",
			Subsystem:          "o2ims",
			CollectionInterval: 30 * time.Second,
			RetentionPeriod:    24 * time.Hour,
		},
		
		HealthCheckConfig: &HealthCheckConfig{
			Enabled:          true,
			CheckInterval:    30 * time.Second,
			Timeout:          10 * time.Second,
			FailureThreshold: 3,
			SuccessThreshold: 1,
			DeepHealthCheck:  true,
		},
		
		NotificationConfig: &NotificationConfig{
			Enabled: true,
			EventTypes: []string{
				"ResourceCreated", "ResourceUpdated", "ResourceDeleted",
				"DeploymentCreated", "DeploymentCompleted", "DeploymentFailed",
				"AlarmRaised", "AlarmCleared",
			},
			RetryPolicy: &RetryPolicy{
				MaxRetries:    3,
				RetryInterval: 5 * time.Second,
				BackoffFactor: 2.0,
				MaxInterval:   30 * time.Second,
			},
		},
		
		SecurityConfig: &SecurityConfig{
			EnableCSRF:         true,
			CSRFTokenLength:    32,
			CORSEnabled:        true,
			CORSAllowedOrigins: []string{"*"},
			CORSAllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
			CORSAllowedHeaders: []string{"*"},
			RateLimitConfig: &RateLimitConfig{
				Enabled:        true,
				RequestsPerMin: 1000,
				BurstSize:      100,
				WindowSize:     time.Minute,
				KeyFunc:        "ip",
			},
			InputValidation: &ValidationConfig{
				EnableSchemaValidation: true,
				StrictValidation:      false,
				MaxRequestSize:        10 * 1024 * 1024, // 10MB
				SanitizeInput:         true,
			},
			AuditLogging: true,
		},
		
		ResourceConfig: &ResourceManagementConfig{
			DefaultTimeout:          5 * time.Minute,
			MaxConcurrentOperations: 100,
			ResourceCacheSize:       1000,
			ResourceCacheTTL:        15 * time.Minute,
			AutoDiscoveryEnabled:    true,
			AutoDiscoveryInterval:   5 * time.Minute,
			StateReconcileInterval:  1 * time.Minute,
		},
	}
}

// MarshalJSON provides custom JSON marshaling for CloudProviderConfig
func (cpc *CloudProviderConfig) MarshalJSON() ([]byte, error) {
	type Alias CloudProviderConfig
	return json.Marshal(&struct {
		*Alias
		CreatedAt       string `json:"created_at"`
		UpdatedAt       string `json:"updated_at"`
		LastHealthCheck string `json:"last_health_check"`
	}{
		Alias:           (*Alias)(cpc),
		CreatedAt:       cpc.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       cpc.UpdatedAt.Format(time.RFC3339),
		LastHealthCheck: cpc.LastHealthCheck.Format(time.RFC3339),
	})
}

// UnmarshalJSON provides custom JSON unmarshaling for CloudProviderConfig
func (cpc *CloudProviderConfig) UnmarshalJSON(data []byte) error {
	type Alias CloudProviderConfig
	aux := &struct {
		*Alias
		CreatedAt       string `json:"created_at"`
		UpdatedAt       string `json:"updated_at"`
		LastHealthCheck string `json:"last_health_check"`
	}{
		Alias: (*Alias)(cpc),
	}
	
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	
	if aux.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, aux.CreatedAt); err == nil {
			cpc.CreatedAt = t
		}
	}
	
	if aux.UpdatedAt != "" {
		if t, err := time.Parse(time.RFC3339, aux.UpdatedAt); err == nil {
			cpc.UpdatedAt = t
		}
	}
	
	if aux.LastHealthCheck != "" {
		if t, err := time.Parse(time.RFC3339, aux.LastHealthCheck); err == nil {
			cpc.LastHealthCheck = t
		}
	}
	
	return nil
}

// String returns a string representation of O2IMSInterface
func (oi O2IMSInterface) String() string {
	return string(oi)
}

// String returns a string representation of O2IMSVersion
func (ov O2IMSVersion) String() string {
	return string(ov)
}

// String returns a string representation of CloudProvider
func (cp CloudProvider) String() string {
	return string(cp)
}

// IsHealthy returns true if the health check status is UP
func (hc *HealthCheck) IsHealthy() bool {
	return hc.Status == "UP"
}

// IsDegraded returns true if the health check status is DEGRADED
func (hc *HealthCheck) IsDegraded() bool {
	return hc.Status == "DEGRADED"
}

// Validate validates the CloudProviderConfig
func (cpc *CloudProviderConfig) Validate() error {
	if cpc.ProviderID == "" {
		return fmt.Errorf("provider ID is required")
	}
	if cpc.Name == "" {
		return fmt.Errorf("provider name is required")
	}
	if cpc.Type == "" {
		return fmt.Errorf("provider type is required")
	}
	if cpc.Endpoint == "" {
		return fmt.Errorf("provider endpoint is required")
	}
	return nil
}

// GetEffectiveTimeout returns the effective timeout for the provider
func (cpc *CloudProviderConfig) GetEffectiveTimeout() time.Duration {
	if cpc.Timeout > 0 {
		return cpc.Timeout
	}
	return 30 * time.Second // Default timeout
}

// IsActive returns true if the provider is active and healthy
func (cpc *CloudProviderConfig) IsActive() bool {
	return cpc.Enabled && cpc.Status == "ACTIVE"
}