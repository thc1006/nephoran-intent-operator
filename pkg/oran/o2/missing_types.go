package o2

import (
	"context"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/oran/common"
	o2models "github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
	securityconfig "github.com/thc1006/nephoran-intent-operator/pkg/security"
)

// ===== HTTP STATUS CONSTANTS =====

const (
	StatusOK                  = 200
	StatusCreated             = 201
	StatusAccepted            = 202
	StatusNoContent           = 204
	StatusBadRequest          = 400
	StatusUnauthorized        = 401
	StatusForbidden           = 403
	StatusNotFound            = 404
	StatusConflict            = 409
	StatusInternalServerError = 500
	StatusServiceUnavailable  = 503
)

// Content type constants
const (
	ContentTypeProblemJSON = "application/problem+json"
	ContentTypeJSON        = "application/json"
	ContentTypeXML         = "application/xml"
)

// ===== MISSING INTERFACE DEFINITIONS =====

// OperatorManagerInterface manages Kubernetes operators
type OperatorManagerInterface interface {
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

// ServiceMeshManagerInterface manages service mesh integrations
type ServiceMeshManagerInterface interface {
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

// ContainerRegistryManagerInterface manages container registry operations
type ContainerRegistryManagerInterface interface {
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
	CheckRegistryHealth(ctx context.Context, registryName string) (*o2models.HealthStatus, error)
	GetRegistryMetrics(ctx context.Context, registryName string) (map[string]interface{}, error)
}

// InfrastructureInventoryManager provides infrastructure inventory management capabilities
type InfrastructureInventoryManager interface {
	// Inventory operations
	GetInventory(ctx context.Context) (*InfrastructureAsset, error)
	UpdateInventory(ctx context.Context, asset *InfrastructureAsset) error
	SynchronizeInventory(ctx context.Context) error

	// Asset discovery
	DiscoverAssets(ctx context.Context) ([]*InfrastructureAsset, error)
	RefreshAsset(ctx context.Context, assetID string) (*InfrastructureAsset, error)

	// Asset lifecycle
	RegisterAsset(ctx context.Context, asset *InfrastructureAsset) error
	UnregisterAsset(ctx context.Context, assetID string) error

	// Asset queries
	ListAssets(ctx context.Context, filters map[string]interface{}) ([]*InfrastructureAsset, error)
	GetAsset(ctx context.Context, assetID string) (*InfrastructureAsset, error)
}

// O2IMSStorage provides storage capabilities for O2 IMS data
type O2IMSStorage interface {
	// Resource storage
	StoreResource(ctx context.Context, resource *o2models.Resource) error
	RetrieveResource(ctx context.Context, resourceID string) (*o2models.Resource, error)
	UpdateResource(ctx context.Context, resource *o2models.Resource) error
	DeleteResource(ctx context.Context, resourceID string) error
	ListResources(ctx context.Context, filters map[string]interface{}) ([]*o2models.Resource, error)

	// Resource pool storage
	ListResourcePools(ctx context.Context, filters map[string]interface{}) ([]*o2models.ResourcePool, error)
	GetResourcePool(ctx context.Context, poolID string) (*o2models.ResourcePool, error)
	StoreResourcePool(ctx context.Context, pool *o2models.ResourcePool) error
	UpdateResourcePool(ctx context.Context, poolID string, pool *o2models.ResourcePool) error
	DeleteResourcePool(ctx context.Context, poolID string) error

	// Resource type storage
	ListResourceTypes(ctx context.Context, filter map[string]interface{}) ([]*o2models.ResourceType, error)
	GetResourceType(ctx context.Context, typeID string) (*o2models.ResourceType, error)
	StoreResourceType(ctx context.Context, resourceType *o2models.ResourceType) error
	UpdateResourceType(ctx context.Context, typeID string, resourceType *o2models.ResourceType) error
	DeleteResourceType(ctx context.Context, typeID string) error

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
	CheckStorageHealth(ctx context.Context) (*o2models.HealthStatus, error)
	GetStorageMetrics(ctx context.Context) (map[string]interface{}, error)
}

// Note: O2IMSService type definition moved to adaptor.go to avoid duplicates

// ===== SUPPORTING TYPE DEFINITIONS =====

// SecurityConfig defines security configuration for the O2 IMS service - extends common config
type SecurityConfig struct {
	securityconfig.CommonSecurityConfig

	// CORS Configuration
	CORSEnabled        bool     `json:"corsEnabled"`
	CORSAllowedOrigins []string `json:"corsAllowedOrigins,omitempty"`
	CORSAllowedMethods []string `json:"corsAllowedMethods,omitempty"`
	CORSAllowedHeaders []string `json:"corsAllowedHeaders,omitempty"`

	// Rate Limiting
	RateLimitConfig *RateLimitConfig `json:"rateLimitConfig,omitempty"`

	// Input Validation
	InputValidation *InputValidationConfig `json:"inputValidation,omitempty"`

	// Additional Security Settings
	EnableCSRF   bool `json:"enableCSRF"`
	AuditLogging bool `json:"auditLogging"`
}

// RateLimitConfig defines rate limiting configuration
type RateLimitConfig struct {
	Enabled        bool   `json:"enabled"`
	RequestsPerMin int    `json:"requestsPerMin"`
	BurstSize      int    `json:"burstSize"`
	KeyFunc        string `json:"keyFunc"` // ip, user, token
}

// TLSSecurityConfig defines TLS security configuration
type TLSSecurityConfig struct {
	Enabled    bool   `json:"enabled"`
	CertFile   string `json:"certFile"`
	KeyFile    string `json:"keyFile"`
	CACertFile string `json:"caCertFile"`
}

// AuthConfigSecurity defines authentication configuration for security
type AuthConfigSecurity struct {
	Enabled   bool      `json:"enabled"`
	Providers []string  `json:"providers"`
	JWTConfig JWTConfig `json:"jwtConfig"`
}

// JWTConfig defines JWT authentication configuration
type JWTConfig struct {
	SecretKey      string `json:"secretKey"`
	TokenDuration  string `json:"tokenDuration"`
	RefreshEnabled bool   `json:"refreshEnabled"`
}

// AuthenticationConfig defines authentication configuration for the API
type AuthenticationConfig struct {
	Enabled        bool     `json:"enabled"`
	JWTSecret      string   `json:"jwtSecret"`
	TokenExpiry    string   `json:"tokenExpiry"`
	AllowedIssuers []string `json:"allowedIssuers"`
	RequiredClaims []string `json:"requiredClaims"`
}

// MetricsConfig defines metrics configuration
type MetricsConfig struct {
	Enabled            bool          `json:"enabled"`
	Path               string        `json:"path"`
	Port               int           `json:"port"`
	CollectionInterval time.Duration `json:"collectionInterval"`
}

// InputValidationConfig defines input validation configuration
type InputValidationConfig struct {
	Enabled                bool `json:"enabled"`
	MaxRequestSize         int  `json:"maxRequestSize"`
	SanitizeHTML           bool `json:"sanitizeHTML"`
	ValidateJSONSchema     bool `json:"validateJSONSchema"`
	StrictValidation       bool `json:"strictValidation"`
	EnableSchemaValidation bool `json:"enableSchemaValidation"`
	SanitizeInput          bool `json:"sanitizeInput"`
}

// RequestContext is already defined in types.go, removing duplicate

// HealthCheckConfigForAPI defines health check configuration for API server (different from infrastructure one)
type HealthCheckConfigForAPI struct {
	Enabled          bool          `json:"enabled"`
	CheckInterval    time.Duration `json:"checkInterval"`
	Timeout          time.Duration `json:"timeout"`
	FailureThreshold int           `json:"failureThreshold"`
	SuccessThreshold int           `json:"successThreshold"`
	DeepHealthCheck  bool          `json:"deepHealthCheck"`
}

// DefaultO2IMSConfig returns the default O2 IMS configuration
func DefaultO2IMSConfig() *O2IMSConfig {
	return &O2IMSConfig{
		ServiceName:    "nephoran-o2-ims",
		ServiceVersion: "1.0.0",
		Host:           "localhost",
		Port:           8080,
		ListenAddress:  "0.0.0.0",
		ListenPort:     8080,
		MetricsPort:    8081,
		HealthPort:     8082,
		TLSEnabled:     false,
		LogLevel:       "info",
		DatabaseURL:    "postgres://localhost/o2ims",
		RedisURL:       "redis://localhost:6379",
		CloudProviders: []string{"kubernetes"},
		Features: map[string]bool{
			"monitoring": true,
			"alerting":   true,
			"metrics":    true,
		},
		Timeouts: map[string]string{
			"request":  "30s",
			"shutdown": "30s",
		},
		Limits: map[string]int{
			"max_connections": 1000,
			"max_requests":    10000,
		},
		SecurityConfig: &SecurityConfig{
			RateLimitConfig: &RateLimitConfig{
				Enabled:        true,
				RequestsPerMin: 1000,
				BurstSize:      10,
				KeyFunc:        "ip",
			},
			CORSEnabled:        true,
			CORSAllowedOrigins: []string{"*"},
			CORSAllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			CORSAllowedHeaders: []string{"Content-Type", "Authorization", "X-Request-ID"},
			InputValidation: &InputValidationConfig{
				Enabled:                true,
				MaxRequestSize:         10 * 1024 * 1024, // 10MB
				SanitizeHTML:           true,
				ValidateJSONSchema:     true,
				StrictValidation:       true,
				EnableSchemaValidation: true,
				SanitizeInput:          true,
			},
		},
		// Server timeout defaults
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB

		// Authentication config
		AuthenticationConfig: &AuthenticationConfig{
			Enabled:        false,
			JWTSecret:      "default-secret",
			TokenExpiry:    "24h",
			AllowedIssuers: []string{"nephoran-o2-ims"},
			RequiredClaims: []string{"iss", "aud", "exp"},
		},

		// Metrics config
		MetricsConfig: &MetricsConfig{
			Enabled:            true,
			Path:               "/metrics",
			Port:               8081,
			CollectionInterval: 30 * time.Second,
		},

		// Certificate files
		CertFile: "",
		KeyFile:  "",
	}
}

// Component health check types are defined in health_checker.go

// Health check wrapper functions to convert between common.ComponentCheck and local ComponentCheck

// WrapCommonComponentCheck converts a function returning common.ComponentCheck to ComponentHealthCheck
func WrapCommonComponentCheck(checkFunc func(ctx context.Context) common.ComponentCheck) ComponentHealthCheck {
	return func(ctx context.Context) ComponentCheck {
		commonCheck := checkFunc(ctx)
		return ComponentCheck{
			Name:      commonCheck.Name,
			Status:    commonCheck.Status,
			Message:   commonCheck.Message,
			Timestamp: commonCheck.Timestamp,
			Duration:  commonCheck.Duration,
			Details:   commonCheck.Details,
			CheckType: commonCheck.CheckType,
		}
	}
}

// Cloud provider configuration with O2 suffix to avoid conflicts
type CloudProviderConfigO2 struct {
	ID            string                 `json:"id"`
	ProviderID    string                 `json:"providerId"`
	Name          string                 `json:"name"`
	Type          string                 `json:"type"` // AWS, Azure, GCP, OpenStack, VMware
	Version       string                 `json:"version,omitempty"`
	Description   string                 `json:"description,omitempty"`
	Enabled       bool                   `json:"enabled"`
	Region        string                 `json:"region,omitempty"`
	Zone          string                 `json:"zone,omitempty"`
	Endpoint      string                 `json:"endpoint,omitempty"`
	Credentials   map[string]string      `json:"credentials"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
	Capabilities  []string               `json:"capabilities"`
	Status        string                 `json:"status"`
	CreatedAt     time.Time              `json:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt"`
	LastSync      time.Time              `json:"lastSync,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Type alias for backward compatibility
type CloudProviderConfig = CloudProviderConfigO2

// Note: Service struct definitions moved to adaptor.go to avoid duplicates

type LifecycleOperation struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	ResourceID   string                 `json:"resourceId"`
	Status       string                 `json:"status"`
	StartTime    time.Time              `json:"startTime"`
	EndTime      *time.Time             `json:"endTime,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	ErrorMessage string                 `json:"errorMessage,omitempty"`
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
	ResourceID string                 `json:"resourceId"`
	Timestamp  time.Time              `json:"timestamp"`
	CPUMetrics *CPUMetrics            `json:"cpuMetrics,omitempty"`
	Memory     *MemoryMetrics         `json:"memoryMetrics,omitempty"`
	Network    *NetworkMetrics        `json:"networkMetrics,omitempty"`
	Storage    *StorageMetrics        `json:"storageMetrics,omitempty"`
	Custom     map[string]interface{} `json:"customMetrics,omitempty"`
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
	ResourceID string               `json:"resourceId"`
	Period     time.Duration        `json:"period"`
	Trends     []MetricTrendData    `json:"trends"`
	Summary    PerformanceSummary   `json:"summary"`
	Anomalies  []PerformanceAnomaly `json:"anomalies,omitempty"`
}

type MetricTrendData struct {
	MetricName string      `json:"metricName"`
	Values     []float64   `json:"values"`
	Timestamps []time.Time `json:"timestamps"`
	Trend      string      `json:"trend"` // INCREASING, DECREASING, STABLE
}

type PerformanceSummary struct {
	OverallHealth string  `json:"overallHealth"`
	AvgCPUUsage   float64 `json:"avgCpuUsage"`
	AvgMemUsage   float64 `json:"avgMemoryUsage"`
	PeakCPUUsage  float64 `json:"peakCpuUsage"`
	PeakMemUsage  float64 `json:"peakMemoryUsage"`
}

type PerformanceAnomaly struct {
	Timestamp     time.Time `json:"timestamp"`
	MetricName    string    `json:"metricName"`
	Value         float64   `json:"value"`
	ExpectedValue float64   `json:"expectedValue"`
	Deviation     float64   `json:"deviation"`
	Severity      string    `json:"severity"`
}

// CNF management types - using existing definitions from cnf_management.go where they exist
type CNFState struct {
	Status    string    `json:"status"`
	Replicas  int32     `json:"replicas"`
	Ready     int32     `json:"ready"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Helm management types

type HelmRepositoryO2 struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// Operator management types
type OperatorInstallRequest struct {
	Name      string                 `json:"name"`
	Namespace string                 `json:"namespace"`
	Channel   string                 `json:"channel,omitempty"`
	Source    string                 `json:"source,omitempty"`
	Version   string                 `json:"version,omitempty"`
	Config    map[string]interface{} `json:"config,omitempty"`
}

type OperatorInstance struct {
	Name        string                 `json:"name"`
	Namespace   string                 `json:"namespace"`
	Version     string                 `json:"version"`
	Status      string                 `json:"status"`
	Channel     string                 `json:"channel"`
	InstallPlan string                 `json:"installPlan,omitempty"`
	InstalledAt time.Time              `json:"installedAt"`
	Config      map[string]interface{} `json:"config,omitempty"`
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
	ImageName       string               `json:"imageName"`
	Registry        string               `json:"registry"`
	ScanID          string               `json:"scanId"`
	Timestamp       time.Time            `json:"timestamp"`
	Status          string               `json:"status"`
	Vulnerabilities []Vulnerability      `json:"vulnerabilities"`
	Summary         VulnerabilitySummary `json:"summary"`
}

type Vulnerability struct {
	ID          string   `json:"id"`
	Severity    string   `json:"severity"`
	Score       float64  `json:"score,omitempty"`
	Package     string   `json:"package"`
	Version     string   `json:"version"`
	FixVersion  string   `json:"fixVersion,omitempty"`
	Description string   `json:"description"`
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
	Status      string                 `json:"status"`
	Policies    []string               `json:"policies"`
	Violations  []SecurityViolation    `json:"violations,omitempty"`
	LastChecked time.Time              `json:"lastChecked"`
	Details     map[string]interface{} `json:"details,omitempty"`
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
	Name      string         `json:"name"`
	Namespace string         `json:"namespace"`
	Rules     []SecurityRule `json:"rules"`
	Enabled   bool           `json:"enabled"`
	CreatedAt time.Time      `json:"createdAt"`
}

type SecurityRule struct {
	Action     string                 `json:"action"`
	Conditions map[string]interface{} `json:"conditions"`
	Targets    []string               `json:"targets"`
	Priority   int                    `json:"priority"`
}

type TrafficRule struct {
	Source      map[string]interface{}   `json:"source"`
	Destination map[string]interface{}   `json:"destination"`
	Match       map[string]interface{}   `json:"match,omitempty"`
	Route       []map[string]interface{} `json:"route,omitempty"`
	Fault       map[string]interface{}   `json:"fault,omitempty"`
	Timeout     string                   `json:"timeout,omitempty"`
	Retries     map[string]interface{}   `json:"retries,omitempty"`
}

type TraceData struct {
	TraceID       string                   `json:"traceId"`
	SpanID        string                   `json:"spanId"`
	ParentSpanID  string                   `json:"parentSpanId,omitempty"`
	OperationName string                   `json:"operationName"`
	StartTime     time.Time                `json:"startTime"`
	Duration      time.Duration            `json:"duration"`
	Tags          map[string]interface{}   `json:"tags,omitempty"`
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
	ResourceID       string        `json:"resourceId"`
	Period           time.Duration `json:"period"`
	OverallTrend     string        `json:"overallTrend"`
	AvailabilityRate float64       `json:"availabilityRate"`
	IncidentCount    int           `json:"incidentCount"`
	MTBF             time.Duration `json:"mtbf"` // Mean Time Between Failures
	MTTR             time.Duration `json:"mttr"` // Mean Time To Recovery
}

// Note: Stub implementations moved to adaptor.go to avoid duplicates
