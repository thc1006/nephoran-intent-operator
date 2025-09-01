package o2

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// O2IMSConfig holds the configuration for O2 IMS.

type O2IMSConfig struct {
	ServerAddress string `json:"serverAddress"`

	Port int `json:"port"`

	ServerPort int `json:"serverPort,omitempty"`

	TLSEnabled bool `json:"tlsEnabled"`

	CertPath string `json:"certPath,omitempty"`

	KeyPath string `json:"keyPath,omitempty"`

	ReadTimeout time.Duration `json:"readTimeout"`

	WriteTimeout time.Duration `json:"writeTimeout"`

	IdleTimeout time.Duration `json:"idleTimeout"`

	MaxHeaderBytes int `json:"maxHeaderBytes"`

	// O-Cloud Configuration.

	CloudID string `json:"cloudId"`

	CloudDescription string `json:"cloudDescription,omitempty"`

	// Authentication.

	AuthEnabled bool `json:"authEnabled"`

	TokenExpiry time.Duration `json:"tokenExpiry"`

	AuthenticationConfig *AuthenticationConfig `json:"authenticationConfig,omitempty"`

	// Database.

	DatabaseURL string `json:"databaseUrl"`

	DatabaseType string `json:"databaseType"`

	DatabaseConfig map[string]interface{} `json:"databaseConfig,omitempty"`

	ConnectionPool int `json:"connectionPool"`

	// Monitoring.

	MetricsEnabled bool `json:"metricsEnabled"`

	TracingEnabled bool `json:"tracingEnabled"`

	// Security.

	SecurityConfig *SecurityConfig `json:"securityConfig,omitempty"`

	// Additional Configuration Fields.

	Logger *logging.StructuredLogger `json:"-"`

	Host string `json:"host,omitempty"`

	// Service configuration from DefaultO2IMSConfig
	ServiceName    string `json:"serviceName"`
	ServiceVersion string `json:"serviceVersion"`
	ListenAddress  string `json:"listenAddress"`
	ListenPort     int    `json:"listenPort"`
	MetricsPort    int    `json:"metricsPort"`
	HealthPort     int    `json:"healthPort"`
	LogLevel       string `json:"logLevel"`
	RedisURL       string `json:"redisUrl"`
	CloudProviders []string `json:"cloudProviders"`
	Features       map[string]bool `json:"features"`
	Timeouts       map[string]string `json:"timeouts"`
	Limits         map[string]int `json:"limits"`

	CloudProviderConfigs map[string]*CloudProviderConfig `json:"cloudProviderConfigs,omitempty"`

	HealthCheckConfig *HealthCheckConfigForAPI `json:"healthCheckConfig,omitempty"`

	MetricsConfig *MetricsConfig `json:"metricsConfig,omitempty"`

	CertFile string `json:"certFile,omitempty"`

	KeyFile string `json:"keyFile,omitempty"`

	NotificationConfig *NotificationConfig `json:"notificationConfig,omitempty"`

	ResourceConfig *ResourceConfig `json:"resourceConfig,omitempty"`

	// Test and compliance settings.

	ComplianceMode bool `json:"complianceMode,omitempty"`

	SpecificationVersion string `json:"specificationVersion,omitempty"`

	ProviderConfigs map[string]interface{} `json:"providerConfigs,omitempty"`
}

// O2IMSService defines the interface for O2 IMS core service.

type O2IMSService interface {

	// Resource Management.

	ListResources(ctx context.Context, filter ResourceFilter) ([]Resource, error)

	GetResource(ctx context.Context, resourceID string) (*Resource, error)

	CreateResource(ctx context.Context, request interface{}) (*Resource, error)

	UpdateResource(ctx context.Context, resourceID string, request interface{}) (*Resource, error)

	DeleteResource(ctx context.Context, resourceID string) error

	// Inventory Management.

	ListResourcePools(ctx context.Context) ([]ResourcePool, error)

	GetResourcePools(ctx context.Context, filter interface{}) ([]ResourcePool, error)

	GetResourcePool(ctx context.Context, poolID string) (*ResourcePool, error)

	CreateResourcePool(ctx context.Context, request interface{}) (*ResourcePool, error)

	UpdateResourcePool(ctx context.Context, poolID string, request interface{}) (*ResourcePool, error)

	DeleteResourcePool(ctx context.Context, poolID string) error

	// Resource Type Management.

	GetResourceTypes(ctx context.Context, filter ...interface{}) ([]ResourceType, error)

	GetResourceType(ctx context.Context, typeID string) (*ResourceType, error)

	CreateResourceType(ctx context.Context, request interface{}) (*ResourceType, error)

	UpdateResourceType(ctx context.Context, typeID string, request interface{}) (*ResourceType, error)

	DeleteResourceType(ctx context.Context, typeID string) error

	// Extended Resource Management.

	GetResources(ctx context.Context, filter interface{}) ([]Resource, error)

	GetResourceHealth(ctx context.Context, resourceID string) (*HealthStatus, error)

	GetResourceAlarms(ctx context.Context, resourceID string, filter ...interface{}) (interface{}, error)

	GetResourceMetrics(ctx context.Context, resourceID string, filter ...interface{}) (interface{}, error)

	// Deployment Template Management.

	GetDeploymentTemplates(ctx context.Context, filter ...interface{}) (interface{}, error)

	GetDeploymentTemplate(ctx context.Context, templateID string) (interface{}, error)

	CreateDeploymentTemplate(ctx context.Context, request interface{}) (interface{}, error)

	UpdateDeploymentTemplate(ctx context.Context, templateID string, request interface{}) (interface{}, error)

	DeleteDeploymentTemplate(ctx context.Context, templateID string) error

	// Deployment Management.

	GetDeployments(ctx context.Context, filter ...interface{}) (interface{}, error)

	GetDeployment(ctx context.Context, deploymentID string) (interface{}, error)

	CreateDeployment(ctx context.Context, request interface{}) (interface{}, error)

	UpdateDeployment(ctx context.Context, deploymentID string, request interface{}) (interface{}, error)

	DeleteDeployment(ctx context.Context, deploymentID string) error

	// Subscription Management.

	CreateSubscription(ctx context.Context, request interface{}) (interface{}, error)

	GetSubscriptions(ctx context.Context, filter ...interface{}) (interface{}, error)

	GetSubscription(ctx context.Context, subscriptionID string) (interface{}, error)

	UpdateSubscription(ctx context.Context, subscriptionID string, request interface{}) (interface{}, error)

	DeleteSubscription(ctx context.Context, subscriptionID string) error

	// Cloud Provider Management.

	RegisterCloudProvider(ctx context.Context, request interface{}) (interface{}, error)

	GetCloudProviders(ctx context.Context, filter ...interface{}) (interface{}, error)

	GetCloudProvider(ctx context.Context, providerID string) (interface{}, error)

	UpdateCloudProvider(ctx context.Context, providerID string, request interface{}) (interface{}, error)

	DeleteCloudProvider(ctx context.Context, providerID string) error

	RemoveCloudProvider(ctx context.Context, providerID string) error

	// Lifecycle Management.

	GetDeploymentManager(ctx context.Context, managerID string) (*DeploymentManager, error)

	ListDeploymentManagers(ctx context.Context) ([]DeploymentManager, error)

	// Health and Status.

	GetHealth(ctx context.Context) (*HealthStatus, error)

	GetVersion(ctx context.Context) (*VersionInfo, error)
}

// ResourceManager handles resource lifecycle operations.

type ResourceManager interface {

	// Resource Operations.

	AllocateResource(ctx context.Context, request *AllocationRequest) (*AllocationResponse, error)

	DeallocateResource(ctx context.Context, resourceID string) error

	UpdateResourceStatus(ctx context.Context, resourceID string, status ResourceStatus) error

	// Resource Discovery.

	DiscoverResources(ctx context.Context) ([]Resource, error)

	ValidateResource(ctx context.Context, resource *Resource) error

	// Resource Monitoring.

	GetResourceMetrics(ctx context.Context, resourceID string) (*ResourceMetrics, error)

	GetResourceEvents(ctx context.Context, resourceID string, filter EventFilter) ([]ResourceEvent, error)

	// Extended Resource Operations.

	ProvisionResource(ctx context.Context, request interface{}) (interface{}, error)

	ConfigureResource(ctx context.Context, resourceID string, request interface{}) (interface{}, error)

	ScaleResource(ctx context.Context, resourceID string, request interface{}) (interface{}, error)

	MigrateResource(ctx context.Context, resourceID string, request interface{}) (interface{}, error)

	BackupResource(ctx context.Context, resourceID string, request interface{}) (interface{}, error)

	RestoreResource(ctx context.Context, resourceID string, request interface{}) (interface{}, error)

	TerminateResource(ctx context.Context, resourceID string) error
}

// InventoryService manages the inventory of resources.

type InventoryService interface {

	// Inventory Management.

	SyncInventory(ctx context.Context) error

	GetInventoryStatus(ctx context.Context) (*InventoryStatus, error)

	// Resource Types.

	ListResourceTypes(ctx context.Context) ([]ResourceType, error)

	GetResourceType(ctx context.Context, typeID string) (*ResourceType, error)

	// Capacity Management.

	GetCapacityInfo(ctx context.Context, resourceType string) (*CapacityInfo, error)

	ReserveCapacity(ctx context.Context, request *CapacityReservation) error

	ReleaseCapacity(ctx context.Context, reservationID string) error

	// Infrastructure Discovery.

	DiscoverInfrastructure(ctx context.Context, provider string) (interface{}, error)

	UpdateInventory(ctx context.Context, request interface{}) (interface{}, error)

	GetAsset(ctx context.Context, assetID string) (interface{}, error)
}

// MonitoringService provides monitoring and alerting capabilities.

type MonitoringService interface {

	// Metrics.

	CollectMetrics(ctx context.Context, resourceID string) (*ResourceMetrics, error)

	GetMetricsHistory(ctx context.Context, resourceID string, timeRange TimeRange) ([]MetricPoint, error)

	// Events and Alarms.

	GetAlarms(ctx context.Context, filter AlarmFilter) ([]Alarm, error)

	AcknowledgeAlarm(ctx context.Context, alarmID string) error

	ClearAlarm(ctx context.Context, alarmID string) error

	// Health Monitoring.

	StartHealthCheck(ctx context.Context, resourceID string) error

	StopHealthCheck(ctx context.Context, resourceID string) error
}

// HealthCheck provides health check functionality.

type HealthCheck struct {
	Status string `json:"status"`

	Timestamp time.Time `json:"timestamp"`

	Checks []ComponentCheck `json:"checks"`

	Version string `json:"version"`

	Environment string `json:"environment"`

	Uptime time.Duration `json:"uptime"`

	Components map[string]interface{} `json:"components"`

	Services []HealthServiceStatus `json:"services"`

	Resources *ResourceHealthSummary `json:"resources"`
}

// CheckResult represents the result of a health check.

type CheckResult struct {
	Status string `json:"status"`

	Message string `json:"message,omitempty"`

	Duration time.Duration `json:"duration"`

	Error string `json:"error,omitempty"`
}

// CNFLifecycleManager manages CNF lifecycle operations.

type CNFLifecycleManager interface {

	// CNF Operations.

	InstantiateCNF(ctx context.Context, request *CNFInstantiationRequest) (*CNFInstance, error)

	TerminateCNF(ctx context.Context, instanceID string) error

	ScaleCNF(ctx context.Context, instanceID string, scaleRequest *CNFScaleRequest) error

	UpdateCNF(ctx context.Context, instanceID string, updateRequest *CNFUpdateRequest) error

	// CNF Status.

	GetCNFStatus(ctx context.Context, instanceID string) (*CNFStatus, error)

	ListCNFInstances(ctx context.Context, filter CNFFilter) ([]CNFInstance, error)
}

// CNFLifecycleManagerImpl implements CNFLifecycleManager.

type CNFLifecycleManagerImpl struct {
	config *CNFLifecycleConfig

	k8sClient client.Client

	logger *logging.StructuredLogger
}

// NewCNFLifecycleManager creates a new CNF lifecycle manager.

func NewCNFLifecycleManager(config *CNFLifecycleConfig, k8sClient client.Client, logger *logging.StructuredLogger) *CNFLifecycleManagerImpl {

	return &CNFLifecycleManagerImpl{

		config: config,

		k8sClient: k8sClient,

		logger: logger,
	}

}

// Implement CNFLifecycleManager methods (placeholders).

func (clm *CNFLifecycleManagerImpl) InstantiateCNF(ctx context.Context, request *CNFInstantiationRequest) (*CNFInstance, error) {

	return nil, nil

}

// TerminateCNF performs terminatecnf operation.

func (clm *CNFLifecycleManagerImpl) TerminateCNF(ctx context.Context, instanceID string) error {

	return nil

}

// ScaleCNF performs scalecnf operation.

func (clm *CNFLifecycleManagerImpl) ScaleCNF(ctx context.Context, instanceID string, scaleRequest *CNFScaleRequest) error {

	return nil

}

// UpdateCNF performs updatecnf operation.

func (clm *CNFLifecycleManagerImpl) UpdateCNF(ctx context.Context, instanceID string, updateRequest *CNFUpdateRequest) error {

	return nil

}

// GetCNFStatus performs getcnfstatus operation.

func (clm *CNFLifecycleManagerImpl) GetCNFStatus(ctx context.Context, instanceID string) (*CNFStatus, error) {

	return nil, nil

}

// ListCNFInstances performs listcnfinstances operation.

func (clm *CNFLifecycleManagerImpl) ListCNFInstances(ctx context.Context, filter CNFFilter) ([]CNFInstance, error) {

	return nil, nil

}

// HelmManager manages Helm chart operations.

type HelmManager interface {

	// Lifecycle.

	Initialize() error

	// Chart Operations.

	InstallChart(ctx context.Context, request *HelmInstallRequest) (*HelmRelease, error)

	UpgradeChart(ctx context.Context, releaseName string, request *HelmUpgradeRequest) (*HelmRelease, error)

	UninstallChart(ctx context.Context, releaseName string) error

	// Deployment operations.

	Deploy(ctx context.Context, request *HelmDeployRequest) (*HelmRelease, error)

	Upgrade(ctx context.Context, request *HelmUpgradeRequest) (*HelmRelease, error)

	Uninstall(ctx context.Context, releaseName, namespace string) error

	// Chart Status.

	GetReleaseStatus(ctx context.Context, releaseName string) (*HelmReleaseStatus, error)

	ListReleases(ctx context.Context, namespace string) ([]HelmRelease, error)

	// Chart Repository.

	AddRepository(ctx context.Context, repo *HelmRepository) error

	UpdateRepository(ctx context.Context, repoName string) error
}

// HelmManagerImpl implements HelmManager.

type HelmManagerImpl struct {
	config *HelmConfig

	logger *logging.StructuredLogger
}

// NewHelmManager creates a new Helm manager.

func NewHelmManager(config *HelmConfig, logger *logging.StructuredLogger) *HelmManagerImpl {

	return &HelmManagerImpl{

		config: config,

		logger: logger,
	}

}

// Implement HelmManager methods (placeholders).

func (hm *HelmManagerImpl) Initialize() error {

	return nil

}

// InstallChart performs installchart operation.

func (hm *HelmManagerImpl) InstallChart(ctx context.Context, request *HelmInstallRequest) (*HelmRelease, error) {

	return nil, nil

}

// UpgradeChart performs upgradechart operation.

func (hm *HelmManagerImpl) UpgradeChart(ctx context.Context, releaseName string, request *HelmUpgradeRequest) (*HelmRelease, error) {

	return nil, nil

}

// UninstallChart performs uninstallchart operation.

func (hm *HelmManagerImpl) UninstallChart(ctx context.Context, releaseName string) error {

	return nil

}

// Deploy performs deploy operation.

func (hm *HelmManagerImpl) Deploy(ctx context.Context, request *HelmDeployRequest) (*HelmRelease, error) {

	return nil, nil

}

// Upgrade performs upgrade operation.

func (hm *HelmManagerImpl) Upgrade(ctx context.Context, request *HelmUpgradeRequest) (*HelmRelease, error) {

	return nil, nil

}

// Uninstall performs uninstall operation.

func (hm *HelmManagerImpl) Uninstall(ctx context.Context, releaseName, namespace string) error {

	return nil

}

// GetReleaseStatus performs getreleasestatus operation.

func (hm *HelmManagerImpl) GetReleaseStatus(ctx context.Context, releaseName string) (*HelmReleaseStatus, error) {

	return nil, nil

}

// ListReleases performs listreleases operation.

func (hm *HelmManagerImpl) ListReleases(ctx context.Context, namespace string) ([]HelmRelease, error) {

	return nil, nil

}

// AddRepository performs addrepository operation.

func (hm *HelmManagerImpl) AddRepository(ctx context.Context, repo *HelmRepository) error {

	return nil

}

// UpdateRepository performs updaterepository operation.

func (hm *HelmManagerImpl) UpdateRepository(ctx context.Context, repoName string) error {

	return nil

}

// NetworkPolicyRule defines network policy rules.

type NetworkPolicyRule struct {
	Name string `json:"name"`

	Direction string `json:"direction"`

	Action string `json:"action"`

	Protocol string `json:"protocol,omitempty"`

	SourcePorts []int `json:"sourcePorts,omitempty"`

	DestPorts []int `json:"destPorts,omitempty"`

	SourceCIDR string `json:"sourceCidr,omitempty"`

	DestCIDR string `json:"destCidr,omitempty"`

	Priority int `json:"priority"`

	Metadata map[string]string `json:"metadata,omitempty"`
}

// Additional supporting types.


// Resource represents a resource.

type Resource struct {
	ID string `json:"id"`

	ResourceID string `json:"resourceId"` // Added for compatibility

	Type string `json:"type"`

	Name string `json:"name"`

	Description string `json:"description,omitempty"`

	Status ResourceStatus `json:"status"`

	Properties map[string]interface{} `json:"properties,omitempty"`

	Metadata map[string]string `json:"metadata,omitempty"`

	CreatedAt time.Time `json:"createdAt"`

	UpdatedAt time.Time `json:"updatedAt"`
}

// ResourceStatus is defined in helper_types.go to avoid duplication.

// ResourcePool represents a resourcepool.

type ResourcePool struct {
	ID string `json:"id"`

	ResourcePoolID string `json:"resourcePoolId"` // Added for compatibility

	Name string `json:"name"`

	Description string `json:"description,omitempty"`

	Capacity CapacityInfo `json:"capacity"`

	Resources []string `json:"resources"`

	Metadata map[string]string `json:"metadata,omitempty"`

	Status *ResourcePoolStatus `json:"status,omitempty"` // Added Status field

	CreatedAt time.Time `json:"createdAt"`

	UpdatedAt time.Time `json:"updatedAt"`
}

// ResourcePoolStatus represents the current status of a resource pool.

type ResourcePoolStatus struct {
	State string `json:"state"` // AVAILABLE, UNAVAILABLE, MAINTENANCE

	Health string `json:"health"` // HEALTHY, DEGRADED, UNHEALTHY

	Utilization float64 `json:"utilization"`

	ErrorMessage string `json:"errorMessage,omitempty"`
}

// DeploymentManager represents a deploymentmanager.

type DeploymentManager struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Type string `json:"type"`

	Endpoint string `json:"endpoint"`

	Status string `json:"status"`

	Capabilities []string `json:"capabilities"`

	Properties map[string]interface{} `json:"properties,omitempty"`

	CreatedAt time.Time `json:"createdAt"`

	UpdatedAt time.Time `json:"updatedAt"`
}

// HealthStatus represents a healthstatus.

type HealthStatus struct {
	Status string `json:"status"`

	Services map[string]string `json:"services"`

	Timestamp time.Time `json:"timestamp"`
}

// VersionInfo represents a versioninfo.

type VersionInfo struct {
	Version string `json:"version"`

	BuildTime string `json:"buildTime"`

	GitCommit string `json:"gitCommit"`

	APIVersion string `json:"apiVersion"`
}

// AllocationRequest represents a allocationrequest.

type AllocationRequest struct {
	ResourceType string `json:"resourceType"`

	Requirements map[string]interface{} `json:"requirements"`

	Duration *time.Duration `json:"duration,omitempty"`

	Metadata map[string]string `json:"metadata,omitempty"`
}

// AllocationResponse represents a allocationresponse.

type AllocationResponse struct {
	ResourceID string `json:"resourceId"`

	Resource *Resource `json:"resource"`

	AllocationID string `json:"allocationId"`

	ExpiresAt *time.Time `json:"expiresAt,omitempty"`

	Properties map[string]interface{} `json:"properties,omitempty"`
}

// Manager implementations are defined in cnf_management.go.

// Request/Response types for manager operations.

type OperatorDeployRequest struct {
	Name string

	Namespace string

	OperatorName string

	Version string

	Channel string

	Source string

	CustomResources []interface{}
}

// OperatorUpdateRequest represents a operatorupdaterequest.

type OperatorUpdateRequest struct {
	Name string

	Namespace string

	CustomResources []interface{}
}

// HelmDeployRequest represents a helmdeployrequest.

type HelmDeployRequest struct {
	ReleaseName string

	Namespace string

	Repository string

	Chart string

	Version string

	Values map[string]interface{}

	ValuesFiles []string
}

// HelmUpgradeRequest represents a helmupgraderequest.

type HelmUpgradeRequest struct {
	ReleaseName string

	Namespace string

	Repository string

	Chart string

	Version string

	Values map[string]interface{}

	ValuesFiles []string
}

// HelmRelease represents a helmrelease.

type HelmRelease struct {
	Name string

	Namespace string

	Version string

	Status string

	Resources []interface{}
}

// Additional stub types to avoid undefined errors.

type (
	ResourceMetrics struct{}

	// EventFilter represents a eventfilter.

	EventFilter struct {
		ResourceIDs []string  `json:"resourceIds,omitempty"`
		EventTypes  []string  `json:"eventTypes,omitempty"`
		States      []string  `json:"states,omitempty"`
		Sources     []string  `json:"sources,omitempty"`
		After       time.Time `json:"after,omitempty"`
		Before      time.Time `json:"before,omitempty"`
	}

	// ResourceEvent represents a resourceevent.

	ResourceEvent struct {
		EventID   string                 `json:"eventId"`
		Type      string                 `json:"type"`
		Reason    string                 `json:"reason"`
		Message   string                 `json:"message"`
		Source    string                 `json:"source"`
		Timestamp time.Time              `json:"timestamp"`
		Metadata  map[string]interface{} `json:"metadata,omitempty"`
	}

	// InventoryStatus represents a inventorystatus.

	InventoryStatus struct{}

	// ResourceType represents a resourcetype.

	ResourceType struct {
		ID string `json:"id"`

		ResourceTypeID string `json:"resourceTypeId"` // Added for compatibility

		Name string `json:"name"`

		Description string `json:"description,omitempty"`

		Vendor string `json:"vendor,omitempty"`

		Version string `json:"version,omitempty"`

		Model string `json:"model,omitempty"`

		Properties map[string]interface{} `json:"properties,omitempty"`

		Metadata map[string]string `json:"metadata,omitempty"`

		CreatedAt time.Time `json:"createdAt"`

		UpdatedAt time.Time `json:"updatedAt"`
	}
)

// CapacityInfo represents a capacityinfo.

type (
	CapacityInfo struct{}

	// CapacityReservation represents a capacityreservation.

	CapacityReservation struct{}

	// TimeRange represents a timerange.

	TimeRange struct{}

	// MetricPoint represents a metricpoint.

	MetricPoint struct{}

	// CNFInstantiationRequest represents a cnfinstantiationrequest.

	CNFInstantiationRequest struct{}

	// CNFScaleRequest represents a cnfscalerequest.

	CNFScaleRequest struct{}

	// CNFUpdateRequest represents a cnfupdaterequest.

	CNFUpdateRequest struct{}

	// CNFFilter represents a cnffilter.

	CNFFilter struct{}

	// HelmInstallRequest represents a helminstallrequest.

	HelmInstallRequest struct{}

	// HelmReleaseStatus represents a helmreleasestatus.

	HelmReleaseStatus struct{}
)

// Config types are defined in cnf_management.go to avoid duplication.

// Additional stub types for missing dependencies.

type InfrastructureHealthChecker interface {
	// Health checking
	CheckHealth(ctx context.Context, resourceID string) (*HealthStatus, error)
	CheckAllResources(ctx context.Context) (map[string]*HealthStatus, error)
	
	// Health monitoring
	StartHealthMonitoring(ctx context.Context, resourceID string, interval time.Duration) error
	StopHealthMonitoring(ctx context.Context, resourceID string) error
	
	// Health policies
	SetHealthPolicy(ctx context.Context, resourceID string, policy interface{}) error
	GetHealthPolicy(ctx context.Context, resourceID string) (interface{}, error)
	
	// Health history and trends
	GetHealthHistory(ctx context.Context, resourceID string, duration time.Duration) ([]interface{}, error)
	GetHealthTrends(ctx context.Context, resourceID string, duration time.Duration) (interface{}, error)
	
	// Health events and callbacks
	RegisterHealthEventCallback(ctx context.Context, resourceID string, callback func(interface{}) error) error
	UnregisterHealthEventCallback(ctx context.Context, resourceID string) error
	EmitHealthEvent(ctx context.Context, event interface{}) error
}

// Note: ResourceStatus, AlarmFilter, Alarm, ComponentCheck are defined in other files.

type (
	SLAMonitor struct{}

	// EventProcessor represents a eventprocessor.

	EventProcessor struct{}
)

// O2IMSStorage interface defines storage operations for O2 IMS.


// Removed - defined below.

type (
	DiscoveryEngine struct{}

	// RelationshipEngine represents a relationshipengine.

	RelationshipEngine struct{}

	// AuditEngine represents a auditengine.

	AuditEngine struct{}

	// AssetIndex represents a assetindex.

	AssetIndex struct{}

	// RelationshipIndex represents a relationshipindex.

	RelationshipIndex struct{}

	// GrafanaClient represents a grafanaclient.

	GrafanaClient struct{}

	// AlertmanagerClient represents a alertmanagerclient.

	AlertmanagerClient struct{}

	// JaegerClient represents a jaegerclient.

	JaegerClient struct{}

	// AlertProcessor represents a alertprocessor.

	AlertProcessor struct{}

	// DashboardManager represents a dashboardmanager.

	DashboardManager struct{}
)

// CloudProviderConfig defines cloud provider configuration.


// CNF and Helm related stub types are defined in cnf_management.go.

type RequestContext struct {
	RequestID string `json:"requestId"`

	Method string `json:"method"`

	Path string `json:"path"`

	RemoteAddr string `json:"remoteAddr"`

	UserAgent string `json:"userAgent"`

	Headers http.Header `json:"headers"`

	QueryParams url.Values `json:"queryParams"`

	StartTime time.Time `json:"startTime"`
}

// ResourceHealthSummary represents a resourcehealthsummary.

type ResourceHealthSummary struct {
	TotalResources int `json:"totalResources"`

	HealthyResources int `json:"healthyResources"`

	DegradedResources int `json:"degradedResources"`

	UnhealthyResources int `json:"unhealthyResources"`

	UnknownResources int `json:"unknownResources"`
}

// ResourceState represents a resourcestate.

type ResourceState struct {
	ResourceID string `json:"resourceId"`
}


// Operation type constants.

const (

	// OperationTypeProvision holds operationtypeprovision value.

	OperationTypeProvision = "provision"

	// OperationTypeConfigure holds operationtypeconfigure value.

	OperationTypeConfigure = "configure"

	// OperationTypeScale holds operationtypescale value.

	OperationTypeScale = "scale"

	// OperationTypeTerminate holds operationtypeterminate value.

	OperationTypeTerminate = "terminate"

	// OperationTypeMigrate holds operationtypemigrate value.

	OperationTypeMigrate = "migrate"

	// OperationTypeBackup holds operationtypebackup value.

	OperationTypeBackup = "backup"

	// OperationTypeRestore holds operationtyperestore value.

	OperationTypeRestore = "restore"
)

// HealthServiceStatus represents the status of an external service dependency.

type HealthServiceStatus struct {
	ServiceName string `json:"serviceName"`

	Status string `json:"status"`

	Endpoint string `json:"endpoint"`

	LastCheck time.Time `json:"lastCheck"`

	Latency time.Duration `json:"latency"`
}

// Missing configuration types.

type NotificationConfig struct {
	Enabled bool `json:"enabled"`
}

// ResourceConfig represents a resourceconfig.

type ResourceConfig struct {
	Enabled bool `json:"enabled"`

	MaxConcurrentOperations int `json:"maxConcurrentOperations"`

	StateReconcileInterval time.Duration `json:"stateReconcileInterval"`

	AutoDiscoveryEnabled bool `json:"autoDiscoveryEnabled"`
}




// Security Configuration Types.

// APISecurityConfig defines security configuration for the API server.

type APISecurityConfig struct {

	// CORS Configuration.

	CORSEnabled bool `json:"corsEnabled"`

	CORSAllowedOrigins []string `json:"corsAllowedOrigins,omitempty"`

	CORSAllowedMethods []string `json:"corsAllowedMethods,omitempty"`

	CORSAllowedHeaders []string `json:"corsAllowedHeaders,omitempty"`

	// Rate Limiting.

	RateLimitConfig *RateLimitConfig `json:"rateLimitConfig,omitempty"`

	// Input Validation.

	InputValidation *InputValidationConfig `json:"inputValidation,omitempty"`

	// Additional Security Settings.

	EnableCSRF bool `json:"enableCSRF"`

	AuditLogging bool `json:"auditLogging"`
}

// RateLimitConfig defines rate limiting configuration.


// InputValidationConfig defines input validation configuration.


// BasicAuthConfig defines basic authentication configuration.

type BasicAuthConfig struct {
	Enabled bool `json:"enabled"`

	JWTSecret string `json:"jwtSecret,omitempty"`

	TokenValidation bool `json:"tokenValidation,omitempty"`
}

// AuthenticationConfig alias for compatibility.


// APIHealthCheckConfig defines health check configuration for API server.

type APIHealthCheckConfig struct {
	Enabled bool `json:"enabled"`

	Interval time.Duration `json:"interval"`

	DeepHealthCheck bool `json:"deepHealthCheck,omitempty"`
}

// MetricsConfig defines metrics configuration.


// APIHealthCheckerConfig defines health check configuration for the API server health checker.

type APIHealthCheckerConfig struct {
	Enabled bool `json:"enabled"`

	CheckInterval time.Duration `json:"checkInterval"`

	Timeout time.Duration `json:"timeout"`

	FailureThreshold int `json:"failureThreshold"`

	SuccessThreshold int `json:"successThreshold"`

	DeepHealthCheck bool `json:"deepHealthCheck"`
}

// DefaultO2IMSConfig returns a default configuration for O2 IMS.

