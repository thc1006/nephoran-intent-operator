package o2

import (
	"context"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
)

// ===== HTTP STATUS CONSTANTS =====

const (
	StatusOK                  = 200
	StatusCreated            = 201
	StatusAccepted           = 202
	StatusNoContent          = 204
	StatusBadRequest         = 400
	StatusUnauthorized       = 401
	StatusForbidden          = 403
	StatusNotFound           = 404
	StatusConflict           = 409
	StatusInternalServerError = 500
	StatusServiceUnavailable = 503
)

// ===== MISSING INTERFACE DEFINITIONS =====

// ResourceManager provides resource management capabilities for O-RAN O2 IMS
type ResourceManager interface {
	// Resource lifecycle operations
	ProvisionResource(ctx context.Context, req *ProvisionResourceRequest) (*models.Resource, error)
	ConfigureResource(ctx context.Context, resourceID string, config interface{}) error
	ScaleResource(ctx context.Context, resourceID string, req *ScaleResourceRequest) error
	TerminateResource(ctx context.Context, resourceID string) error
	MigrateResource(ctx context.Context, resourceID string, req *MigrateResourceRequest) error

	// Resource discovery and inventory
	DiscoverResources(ctx context.Context, providerID string) ([]*models.Resource, error)
	SyncInventory(ctx context.Context) error
	
	// Resource monitoring
	GetResourceStatus(ctx context.Context, resourceID string) (*models.ResourceStatus, error)
	GetResourceMetrics(ctx context.Context, resourceID string) (map[string]interface{}, error)
	GetResourceHealth(ctx context.Context, resourceID string) (*models.HealthStatus, error)

	// Resource operations
	StartResource(ctx context.Context, resourceID string) error
	StopResource(ctx context.Context, resourceID string) error
	RestartResource(ctx context.Context, resourceID string) error
}

// MonitoringService provides comprehensive monitoring capabilities following O-RAN specifications
type MonitoringService interface {
	// Infrastructure monitoring
	StartMonitoring(ctx context.Context, resourceID string, config *MonitoringConfig) error
	StopMonitoring(ctx context.Context, resourceID string) error
	
	// Metrics collection
	CollectMetrics(ctx context.Context, resourceIDs []string) (map[string]interface{}, error)
	GetMetrics(ctx context.Context, resourceID string, metricNames []string) (map[string]interface{}, error)
	
	// Health monitoring
	CheckHealth(ctx context.Context, resourceID string) (*models.HealthStatus, error)
	GetHealthHistory(ctx context.Context, resourceID string, duration time.Duration) ([]*models.HealthHistoryEntry, error)
	
	// Alerting and notifications
	CreateAlert(ctx context.Context, alert *AlertO2) error
	GetActiveAlerts(ctx context.Context) ([]*AlertO2, error)
	ResolveAlert(ctx context.Context, alertID string) error
	
	// Performance monitoring
	MonitorPerformance(ctx context.Context, resourceID string) (*PerformanceMetrics, error)
	GetPerformanceTrends(ctx context.Context, resourceID string, duration time.Duration) (*PerformanceTrends, error)
}

// CNFLifecycleManager manages CNF lifecycle operations
type CNFLifecycleManager interface {
	// CNF deployment operations
	DeployCNF(ctx context.Context, spec *CNFSpec) (*CNFInstance, error)
	UpdateCNF(ctx context.Context, cnfID string, spec *CNFSpec) error
	ScaleCNF(ctx context.Context, cnfID string, replicas int32) error
	TerminateCNF(ctx context.Context, cnfID string) error
	
	// CNF state management
	GetCNFState(ctx context.Context, cnfID string) (*CNFState, error)
	ListCNFs(ctx context.Context, filter *CNFFilter) ([]*CNFInstance, error)
	
	// CNF monitoring
	MonitorCNF(ctx context.Context, cnfID string) error
	GetCNFMetrics(ctx context.Context, cnfID string) (map[string]interface{}, error)
	GetCNFHealth(ctx context.Context, cnfID string) (*models.HealthStatus, error)
}

// HelmManager manages Helm chart deployments
type HelmManager interface {
	// Chart management
	InstallChart(ctx context.Context, req *HelmInstallRequest) (*HelmRelease, error)
	UpgradeChart(ctx context.Context, req *HelmUpgradeRequest) (*HelmRelease, error)
	UninstallChart(ctx context.Context, releaseName, namespace string) error
	
	// Release operations
	GetRelease(ctx context.Context, releaseName, namespace string) (*HelmRelease, error)
	ListReleases(ctx context.Context, namespace string) ([]*HelmRelease, error)
	RollbackRelease(ctx context.Context, releaseName, namespace string, revision int) error
	
	// Chart repository management
	AddRepository(ctx context.Context, repo *HelmRepositoryO2) error
	UpdateRepository(ctx context.Context, repoName string) error
	
	// Values and configuration
	GetValues(ctx context.Context, releaseName, namespace string) (map[string]interface{}, error)
	ValidateValues(ctx context.Context, chartPath string, values map[string]interface{}) error
}

// OperatorManager manages Kubernetes operators
type OperatorManager interface {
	// Operator installation
	InstallOperator(ctx context.Context, req *OperatorInstallRequest) (*OperatorInstance, error)
	UninstallOperator(ctx context.Context, operatorName, namespace string) error
	
	// Operator lifecycle
	GetOperator(ctx context.Context, operatorName, namespace string) (*OperatorInstance, error)
	ListOperators(ctx context.Context, namespace string) ([]*OperatorInstance, error)
	UpdateOperator(ctx context.Context, operatorName, namespace string, config map[string]interface{}) error
	
	// Custom resources
	CreateCustomResource(ctx context.Context, resource *CustomResourceSpec) error
	UpdateCustomResource(ctx context.Context, resource *CustomResourceSpec) error
	DeleteCustomResource(ctx context.Context, resourceName, namespace, apiVersion, kind string) error
	
	// Operator catalogs
	RefreshCatalog(ctx context.Context) error
	SearchOperators(ctx context.Context, query string) ([]*OperatorInfo, error)
}

// ServiceMeshManager manages service mesh integrations
type ServiceMeshManager interface {
	// Service mesh management
	EnableServiceMesh(ctx context.Context, namespace string, config *ServiceMeshConfig) error
	DisableServiceMesh(ctx context.Context, namespace string) error
	
	// Traffic management
	ConfigureTrafficPolicy(ctx context.Context, policy *TrafficPolicy) error
	GetTrafficMetrics(ctx context.Context, serviceName, namespace string) (map[string]interface{}, error)
	
	// Security policies
	ConfigureSecurity(ctx context.Context, policy *SecurityPolicy) error
	GetSecurityStatus(ctx context.Context, serviceName, namespace string) (*SecurityStatus, error)
	
	// Observability
	EnableTracing(ctx context.Context, serviceName, namespace string) error
	GetTraceData(ctx context.Context, serviceName, namespace string, duration time.Duration) ([]*TraceData, error)
}

// ContainerRegistryManager manages container registry operations
type ContainerRegistryManager interface {
	// Registry management
	AddRegistry(ctx context.Context, registry *ContainerRegistry) error
	RemoveRegistry(ctx context.Context, registryName string) error
	ListRegistries(ctx context.Context) ([]*ContainerRegistry, error)
	
	// Image operations
	PushImage(ctx context.Context, image *ContainerImage, registryName string) error
	PullImage(ctx context.Context, imageName, registryName string) (*ContainerImage, error)
	DeleteImage(ctx context.Context, imageName, registryName string) error
	
	// Image scanning
	ScanImage(ctx context.Context, imageName, registryName string) (*ScanResult, error)
	GetScanResults(ctx context.Context, imageName, registryName string) ([]*ScanResult, error)
	
	// Registry health
	CheckRegistryHealth(ctx context.Context, registryName string) (*models.HealthStatus, error)
	GetRegistryMetrics(ctx context.Context, registryName string) (map[string]interface{}, error)
}

// InfrastructureHealthChecker provides infrastructure health checking capabilities
type InfrastructureHealthChecker interface {
	// Health checking
	CheckHealth(ctx context.Context, resourceID string) (*models.HealthStatus, error)
	CheckAllResources(ctx context.Context) (map[string]*models.HealthStatus, error)
	
	// Health monitoring
	StartHealthMonitoring(ctx context.Context, resourceID string, interval time.Duration) error
	StopHealthMonitoring(ctx context.Context, resourceID string) error
	
	// Health policies
	SetHealthPolicy(ctx context.Context, resourceID string, policy *HealthPolicy) error
	GetHealthPolicy(ctx context.Context, resourceID string) (*HealthPolicy, error)
	
	// Health history and trends
	GetHealthHistory(ctx context.Context, resourceID string, duration time.Duration) ([]*models.HealthHistoryEntry, error)
	GetHealthTrends(ctx context.Context, resourceID string, duration time.Duration) (*HealthTrendsO2, error)
	
	// Health events and callbacks
	RegisterHealthEventCallback(ctx context.Context, resourceID string, callback func(*HealthEventO2) error) error
	UnregisterHealthEventCallback(ctx context.Context, resourceID string) error
	EmitHealthEvent(ctx context.Context, event *HealthEventO2) error
}

// O2IMSStorage provides storage capabilities for O2 IMS data
type O2IMSStorage interface {
	// Resource storage
	StoreResource(ctx context.Context, resource *models.Resource) error
	RetrieveResource(ctx context.Context, resourceID string) (*models.Resource, error)
	UpdateResource(ctx context.Context, resource *models.Resource) error
	DeleteResource(ctx context.Context, resourceID string) error
	ListResources(ctx context.Context, filters map[string]interface{}) ([]*models.Resource, error)
	
	// Metadata and inventory
	StoreInventory(ctx context.Context, inventory *InfrastructureAsset) error
	RetrieveInventory(ctx context.Context, assetID string) (*InfrastructureAsset, error)
	
	// Lifecycle operations
	StoreLifecycleOperation(ctx context.Context, operation *LifecycleOperation) error
	RetrieveLifecycleOperation(ctx context.Context, operationID string) (*LifecycleOperation, error)
	
	// Backup and restore
	BackupData(ctx context.Context, backupID string) error
	RestoreData(ctx context.Context, backupID string) error
	
	// Storage health
	CheckStorageHealth(ctx context.Context) (*models.HealthStatus, error)
	GetStorageMetrics(ctx context.Context) (map[string]interface{}, error)
}

// O2IMSService provides the main service interface for O2 IMS operations with cloud provider support
type O2IMSService interface {
	// Core IMS operations
	GetResourceAlarms(ctx context.Context, resourceID string, filter *AlarmFilter) ([]*models.Alarm, error)
	GetResourceMetrics(ctx context.Context, resourceID string, filter *MetricsFilter) (map[string]interface{}, error)
	GetResourceStatus(ctx context.Context, resourceID string) (*models.ResourceStatus, error)
	
	// Cloud provider management
	RegisterCloudProvider(ctx context.Context, provider *CloudProviderConfigO2) error
	UnregisterCloudProvider(ctx context.Context, providerID string) error
	GetCloudProviders(ctx context.Context) ([]*CloudProviderConfigO2, error)
	GetCloudProvider(ctx context.Context, providerID string) (*CloudProviderConfigO2, error)
	
	// Resource pool management
	CreateResourcePool(ctx context.Context, pool *models.ResourcePool) (*models.ResourcePool, error)
	GetResourcePool(ctx context.Context, poolID string) (*models.ResourcePool, error)
	ListResourcePools(ctx context.Context) ([]*models.ResourcePool, error)
	UpdateResourcePool(ctx context.Context, pool *models.ResourcePool) (*models.ResourcePool, error)
	DeleteResourcePool(ctx context.Context, poolID string) error
	
	// Subscription management
	CreateSubscription(ctx context.Context, req *CreateSubscriptionRequest) (*models.Subscription, error)
	GetSubscription(ctx context.Context, subscriptionID string) (*models.Subscription, error)
	UpdateSubscription(ctx context.Context, subscriptionID string, req *UpdateSubscriptionRequest) (*models.Subscription, error)
	DeleteSubscription(ctx context.Context, subscriptionID string) error
	
	// Service health
	GetServiceHealth(ctx context.Context) (*models.HealthStatus, error)
	GetServiceMetrics(ctx context.Context) (map[string]interface{}, error)
}

// ===== SUPPORTING TYPE DEFINITIONS =====

// Configuration and service types
type O2IMSConfig struct {
	ServiceName     string            `json:"serviceName"`
	ServiceVersion  string            `json:"serviceVersion"`
	ListenAddress   string            `json:"listenAddress"`
	ListenPort      int               `json:"listenPort"`
	MetricsPort     int               `json:"metricsPort"`
	HealthPort      int               `json:"healthPort"`
	LogLevel        string            `json:"logLevel"`
	DatabaseURL     string            `json:"databaseUrl"`
	RedisURL        string            `json:"redisUrl"`
	CloudProviders  []string          `json:"cloudProviders"`
	Features        map[string]bool   `json:"features"`
	Timeouts        map[string]string `json:"timeouts"`
	Limits          map[string]int    `json:"limits"`
	Authentication  map[string]string `json:"authentication"`
	TLS             map[string]string `json:"tls"`
	Monitoring      map[string]string `json:"monitoring"`
	Storage         map[string]string `json:"storage"`
	Cache           map[string]string `json:"cache"`
	Notifications   map[string]string `json:"notifications"`
	CustomSettings  map[string]interface{} `json:"customSettings"`
}

// Cloud provider configuration with O2 suffix to avoid conflicts
type CloudProviderConfigO2 struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Type           string                 `json:"type"` // AWS, Azure, GCP, OpenStack, VMware
	Region         string                 `json:"region,omitempty"`
	Endpoint       string                 `json:"endpoint,omitempty"`
	Credentials    map[string]string      `json:"credentials"`
	Configuration  map[string]interface{} `json:"configuration,omitempty"`
	Capabilities   []string               `json:"capabilities"`
	Status         string                 `json:"status"`
	LastSync       time.Time              `json:"lastSync,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// Type alias for backward compatibility
type CloudProviderConfig = CloudProviderConfigO2

// Service implementations with stub methods - using interface{} for logger to avoid import issues
type IMSService struct {
	Config    *O2IMSConfig
	Logger    interface{}  // Placeholder for logger
	K8sClient client.Client
	ClientSet *kubernetes.Clientset
	IMSClient interface{}  // Placeholder for IMS client
	mu        sync.RWMutex
}

type InventoryService struct {
	Config *O2IMSConfig
	Logger interface{}  // Placeholder for logger
}

type LifecycleService struct {
	Config *O2IMSConfig
	Logger interface{}  // Placeholder for logger
}

type SubscriptionService struct {
	Config *O2IMSConfig
	Logger interface{}  // Placeholder for logger
}

type LifecycleOperation struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	ResourceID    string                 `json:"resourceId"`
	Status        string                 `json:"status"`
	StartTime     time.Time              `json:"startTime"`
	EndTime       *time.Time             `json:"endTime,omitempty"`
	Parameters    map[string]interface{} `json:"parameters,omitempty"`
	ErrorMessage  string                 `json:"errorMessage,omitempty"`
}

type InfrastructureAsset struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Type           string                 `json:"type"`
	Location       string                 `json:"location,omitempty"`
	Status         string                 `json:"status"`
	Properties     map[string]interface{} `json:"properties,omitempty"`
	LastDiscovered time.Time              `json:"lastDiscovered"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// Alert type with O2 suffix to avoid conflicts  
type AlertO2 struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	ResourceID  string                 `json:"resourceId,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Status      string                 `json:"status"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Annotations map[string]string      `json:"annotations,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Performance monitoring types
type PerformanceMetrics struct {
	ResourceID  string                    `json:"resourceId"`
	Timestamp   time.Time                 `json:"timestamp"`
	CPUMetrics  *CPUMetrics               `json:"cpuMetrics,omitempty"`
	Memory      *MemoryMetrics            `json:"memoryMetrics,omitempty"`
	Network     *NetworkMetrics           `json:"networkMetrics,omitempty"`
	Storage     *StorageMetrics           `json:"storageMetrics,omitempty"`
	Custom      map[string]interface{}    `json:"customMetrics,omitempty"`
}

type CPUMetrics struct {
	Usage       float64 `json:"usage"`
	LoadAverage float64 `json:"loadAverage"`
	Cores       int     `json:"cores"`
	Frequency   float64 `json:"frequency,omitempty"`
}

type MemoryMetrics struct {
	TotalBytes     int64   `json:"totalBytes"`
	UsedBytes      int64   `json:"usedBytes"`
	AvailableBytes int64   `json:"availableBytes"`
	UsagePercent   float64 `json:"usagePercent"`
}

type NetworkMetrics struct {
	BytesIn    int64   `json:"bytesIn"`
	BytesOut   int64   `json:"bytesOut"`
	PacketsIn  int64   `json:"packetsIn"`
	PacketsOut int64   `json:"packetsOut"`
	ErrorsIn   int64   `json:"errorsIn"`
	ErrorsOut  int64   `json:"errorsOut"`
	Latency    float64 `json:"latency,omitempty"`
}

type StorageMetrics struct {
	TotalBytes     int64   `json:"totalBytes"`
	UsedBytes      int64   `json:"usedBytes"`
	AvailableBytes int64   `json:"availableBytes"`
	UsagePercent   float64 `json:"usagePercent"`
	IOPS           float64 `json:"iops,omitempty"`
	Throughput     float64 `json:"throughput,omitempty"`
}

type PerformanceTrends struct {
	ResourceID   string            `json:"resourceId"`
	Period       time.Duration     `json:"period"`
	Trends       []MetricTrendData `json:"trends"`
	Summary      PerformanceSummary `json:"summary"`
	Anomalies    []PerformanceAnomaly `json:"anomalies,omitempty"`
}

type MetricTrendData struct {
	MetricName string    `json:"metricName"`
	Values     []float64 `json:"values"`
	Timestamps []time.Time `json:"timestamps"`
	Trend      string    `json:"trend"` // INCREASING, DECREASING, STABLE
}

type PerformanceSummary struct {
	OverallHealth string  `json:"overallHealth"`
	AvgCPUUsage   float64 `json:"avgCpuUsage"`
	AvgMemUsage   float64 `json:"avgMemoryUsage"`
	PeakCPUUsage  float64 `json:"peakCpuUsage"`
	PeakMemUsage  float64 `json:"peakMemoryUsage"`
}

type PerformanceAnomaly struct {
	Timestamp   time.Time `json:"timestamp"`
	MetricName  string    `json:"metricName"`
	Value       float64   `json:"value"`
	ExpectedValue float64 `json:"expectedValue"`
	Deviation   float64   `json:"deviation"`
	Severity    string    `json:"severity"`
}

// CNF management types - using existing definitions from cnf_management.go where they exist
type CNFState struct {
	Status    string    `json:"status"`
	Replicas  int32     `json:"replicas"`
	Ready     int32     `json:"ready"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CNFFilter struct {
	Names       []string `json:"names,omitempty"`
	Namespaces  []string `json:"namespaces,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Status      []string `json:"status,omitempty"`
	Types       []string `json:"types,omitempty"`
}

// Helm management types
type HelmInstallRequest struct {
	ChartName   string                 `json:"chartName"`
	ReleaseName string                 `json:"releaseName"`
	Namespace   string                 `json:"namespace"`
	Values      map[string]interface{} `json:"values,omitempty"`
	Version     string                 `json:"version,omitempty"`
	RepoURL     string                 `json:"repoUrl,omitempty"`
}

type HelmUpgradeRequest struct {
	ReleaseName string                 `json:"releaseName"`
	Namespace   string                 `json:"namespace"`
	ChartName   string                 `json:"chartName,omitempty"`
	Values      map[string]interface{} `json:"values,omitempty"`
	Version     string                 `json:"version,omitempty"`
	ResetValues bool                   `json:"resetValues"`
}

type HelmRelease struct {
	Name        string                 `json:"name"`
	Namespace   string                 `json:"namespace"`
	Chart       string                 `json:"chart"`
	Version     string                 `json:"version"`
	Status      string                 `json:"status"`
	Revision    int                    `json:"revision"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	Values      map[string]interface{} `json:"values,omitempty"`
}

type HelmRepositoryO2 struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// Operator management types
type OperatorInstallRequest struct {
	Name        string                 `json:"name"`
	Namespace   string                 `json:"namespace"`
	Channel     string                 `json:"channel,omitempty"`
	Source      string                 `json:"source,omitempty"`
	Version     string                 `json:"version,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

type OperatorInstance struct {
	Name           string                 `json:"name"`
	Namespace      string                 `json:"namespace"`
	Version        string                 `json:"version"`
	Status         string                 `json:"status"`
	Channel        string                 `json:"channel"`
	InstallPlan    string                 `json:"installPlan,omitempty"`
	InstalledAt    time.Time              `json:"installedAt"`
	Config         map[string]interface{} `json:"config,omitempty"`
}

type CustomResourceSpec struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Name       string                 `json:"name"`
	Namespace  string                 `json:"namespace"`
	Spec       map[string]interface{} `json:"spec"`
}

type OperatorInfo struct {
	Name         string   `json:"name"`
	DisplayName  string   `json:"displayName"`
	Description  string   `json:"description"`
	Version      string   `json:"version"`
	Provider     string   `json:"provider"`
	Categories   []string `json:"categories"`
	Capabilities []string `json:"capabilities"`
	Repository   string   `json:"repository,omitempty"`
}

// Container registry types
type ContainerRegistry struct {
	Name        string            `json:"name"`
	URL         string            `json:"url"`
	Type        string            `json:"type"` // Docker, Harbor, ECR, GCR, ACR
	Credentials map[string]string `json:"credentials"`
	Secure      bool              `json:"secure"`
	Default     bool              `json:"default"`
}

type ContainerImage struct {
	Name         string            `json:"name"`
	Tag          string            `json:"tag"`
	Registry     string            `json:"registry"`
	Size         int64             `json:"size"`
	CreatedAt    time.Time         `json:"createdAt"`
	Labels       map[string]string `json:"labels,omitempty"`
	Architecture string            `json:"architecture,omitempty"`
	OS           string            `json:"os,omitempty"`
	Digest       string            `json:"digest,omitempty"`
}

type ScanResult struct {
	ImageName        string                `json:"imageName"`
	Registry         string                `json:"registry"`
	ScanID           string                `json:"scanId"`
	Timestamp        time.Time             `json:"timestamp"`
	Status           string                `json:"status"`
	Vulnerabilities  []Vulnerability       `json:"vulnerabilities"`
	Summary          VulnerabilitySummary  `json:"summary"`
}

type Vulnerability struct {
	ID          string  `json:"id"`
	Severity    string  `json:"severity"`
	Score       float64 `json:"score,omitempty"`
	Package     string  `json:"package"`
	Version     string  `json:"version"`
	FixVersion  string  `json:"fixVersion,omitempty"`
	Description string  `json:"description"`
	References  []string `json:"references,omitempty"`
}

type VulnerabilitySummary struct {
	Total    int `json:"total"`
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
}

// Security and traffic policy types - referenced from existing files
type SecurityStatus struct {
	Status       string                 `json:"status"`
	Policies     []string               `json:"policies"`
	Violations   []SecurityViolation    `json:"violations,omitempty"`
	LastChecked  time.Time              `json:"lastChecked"`
	Details      map[string]interface{} `json:"details,omitempty"`
}

type SecurityViolation struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Resource    string    `json:"resource"`
	DetectedAt  time.Time `json:"detectedAt"`
}

type SecurityPolicy struct {
	Name        string          `json:"name"`
	Namespace   string          `json:"namespace"`
	Rules       []SecurityRule  `json:"rules"`
	Enabled     bool            `json:"enabled"`
	CreatedAt   time.Time       `json:"createdAt"`
}

type SecurityRule struct {
	Action      string                 `json:"action"`
	Conditions  map[string]interface{} `json:"conditions"`
	Targets     []string               `json:"targets"`
	Priority    int                    `json:"priority"`
}

type TrafficRule struct {
	Source      map[string]interface{} `json:"source"`
	Destination map[string]interface{} `json:"destination"`
	Match       map[string]interface{} `json:"match,omitempty"`
	Route       []map[string]interface{} `json:"route,omitempty"`
	Fault       map[string]interface{} `json:"fault,omitempty"`
	Timeout     string                 `json:"timeout,omitempty"`
	Retries     map[string]interface{} `json:"retries,omitempty"`
}

type TraceData struct {
	TraceID       string                 `json:"traceId"`
	SpanID        string                 `json:"spanId"`
	ParentSpanID  string                 `json:"parentSpanId,omitempty"`
	OperationName string                 `json:"operationName"`
	StartTime     time.Time              `json:"startTime"`
	Duration      time.Duration          `json:"duration"`
	Tags          map[string]interface{} `json:"tags,omitempty"`
	Logs          []map[string]interface{} `json:"logs,omitempty"`
}

// Health events with O2 suffix to avoid conflicts
type HealthEventO2 struct {
	EventID        string                 `json:"eventId"`
	ResourceID     string                 `json:"resourceId"`
	EventType      string                 `json:"eventType"`
	Severity       string                 `json:"severity"`
	Message        string                 `json:"message"`
	HealthStatus   string                 `json:"healthStatus"`
	PreviousStatus string                 `json:"previousStatus,omitempty"`
	Timestamp      time.Time              `json:"timestamp"`
	Details        map[string]interface{} `json:"details,omitempty"`
}

type HealthTrendsO2 struct {
	ResourceID       string              `json:"resourceId"`
	Period           time.Duration       `json:"period"`
	OverallTrend     string              `json:"overallTrend"`
	AvailabilityRate float64             `json:"availabilityRate"`
	IncidentCount    int                 `json:"incidentCount"`
	MTBF             time.Duration       `json:"mtbf"` // Mean Time Between Failures
	MTTR             time.Duration       `json:"mttr"` // Mean Time To Recovery
}

// ===== STUB IMPLEMENTATIONS =====

// IMSService stub implementations for O2IMSService interface
func (s *IMSService) GetResourceAlarms(ctx context.Context, resourceID string, filter *AlarmFilter) ([]*models.Alarm, error) {
	// Stub implementation
	return []*models.Alarm{}, nil
}

func (s *IMSService) GetResourceMetrics(ctx context.Context, resourceID string, filter *MetricsFilter) (map[string]interface{}, error) {
	// Stub implementation
	return map[string]interface{}{}, nil
}

func (s *IMSService) GetResourceStatus(ctx context.Context, resourceID string) (*models.ResourceStatus, error) {
	// Stub implementation
	return &models.ResourceStatus{}, nil
}

func (s *IMSService) RegisterCloudProvider(ctx context.Context, provider *CloudProviderConfigO2) error {
	// Stub implementation
	return nil
}

func (s *IMSService) UnregisterCloudProvider(ctx context.Context, providerID string) error {
	// Stub implementation
	return nil
}

func (s *IMSService) GetCloudProviders(ctx context.Context) ([]*CloudProviderConfigO2, error) {
	// Stub implementation
	return []*CloudProviderConfigO2{}, nil
}

func (s *IMSService) GetCloudProvider(ctx context.Context, providerID string) (*CloudProviderConfigO2, error) {
	// Stub implementation
	return &CloudProviderConfigO2{}, nil
}

func (s *IMSService) CreateResourcePool(ctx context.Context, pool *models.ResourcePool) (*models.ResourcePool, error) {
	// Stub implementation
	return pool, nil
}

func (s *IMSService) GetResourcePool(ctx context.Context, poolID string) (*models.ResourcePool, error) {
	// Stub implementation
	return &models.ResourcePool{}, nil
}

func (s *IMSService) ListResourcePools(ctx context.Context) ([]*models.ResourcePool, error) {
	// Stub implementation
	return []*models.ResourcePool{}, nil
}

func (s *IMSService) UpdateResourcePool(ctx context.Context, pool *models.ResourcePool) (*models.ResourcePool, error) {
	// Stub implementation
	return pool, nil
}

func (s *IMSService) DeleteResourcePool(ctx context.Context, poolID string) error {
	// Stub implementation
	return nil
}

func (s *IMSService) CreateSubscription(ctx context.Context, req *CreateSubscriptionRequest) (*models.Subscription, error) {
	// Stub implementation
	return &models.Subscription{}, nil
}

func (s *IMSService) GetSubscription(ctx context.Context, subscriptionID string) (*models.Subscription, error) {
	// Stub implementation
	return &models.Subscription{}, nil
}

func (s *IMSService) UpdateSubscription(ctx context.Context, subscriptionID string, req *UpdateSubscriptionRequest) (*models.Subscription, error) {
	// Stub implementation
	return &models.Subscription{}, nil
}

func (s *IMSService) DeleteSubscription(ctx context.Context, subscriptionID string) error {
	// Stub implementation
	return nil
}

func (s *IMSService) GetServiceHealth(ctx context.Context) (*models.HealthStatus, error) {
	// Stub implementation
	return &models.HealthStatus{}, nil
}

func (s *IMSService) GetServiceMetrics(ctx context.Context) (map[string]interface{}, error) {
	// Stub implementation
	return map[string]interface{}{}, nil
}