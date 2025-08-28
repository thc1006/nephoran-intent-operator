package o2

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
)

// InfrastructureResourceManager manages infrastructure resources and their lifecycles.
type InfrastructureResourceManager struct {
	storage    O2IMSStorage
	kubeClient client.Client
	logger     *logging.StructuredLogger
	config     *ResourceManagerConfig

	// Resource cache.
	resourceCache map[string]*models.Resource
	cacheMu       sync.RWMutex
	cacheExpiry   time.Duration

	// Resource lifecycle management.
	lifecycleManager  ResourceLifecycleManager
	provisioningQueue ProvisioningQueue

	// Resource monitoring and health.
	healthMonitor    ResourceHealthMonitor
	metricsCollector ResourceMetricsCollector

	// Resource discovery and synchronization.
	discoveryEnabled bool
	syncEnabled      bool

	// Operation tracking.
	operationTracker OperationTracker

	// Resource validation.
	validator ResourceValidator
}

// ResourceManagerConfig defines configuration for resource management.
type ResourceManagerConfig struct {
	// Cache configuration.
	CacheEnabled bool          `json:"cache_enabled"`
	CacheExpiry  time.Duration `json:"cache_expiry"`
	MaxCacheSize int           `json:"max_cache_size"`

	// Lifecycle management.
	DefaultProvisionTimeout   time.Duration `json:"default_provision_timeout"`
	DefaultTerminationTimeout time.Duration `json:"default_termination_timeout"`
	MaxConcurrentOperations   int           `json:"max_concurrent_operations"`

	// Discovery and synchronization.
	AutoDiscoveryEnabled bool          `json:"auto_discovery_enabled"`
	DiscoveryInterval    time.Duration `json:"discovery_interval"`
	StateSyncEnabled     bool          `json:"state_sync_enabled"`
	StateSyncInterval    time.Duration `json:"state_sync_interval"`

	// Health monitoring.
	HealthMonitoringEnabled bool          `json:"health_monitoring_enabled"`
	HealthCheckInterval     time.Duration `json:"health_check_interval"`
	HealthCheckTimeout      time.Duration `json:"health_check_timeout"`

	// Metrics collection.
	MetricsCollectionEnabled  bool          `json:"metrics_collection_enabled"`
	MetricsCollectionInterval time.Duration `json:"metrics_collection_interval"`
	MetricsRetentionPeriod    time.Duration `json:"metrics_retention_period"`

	// Resource validation.
	ValidationEnabled bool          `json:"validation_enabled"`
	StrictValidation  bool          `json:"strict_validation"`
	ValidationTimeout time.Duration `json:"validation_timeout"`

	// Error handling and retry.
	MaxRetries       int           `json:"max_retries"`
	RetryInterval    time.Duration `json:"retry_interval"`
	BackoffFactor    float64       `json:"backoff_factor"`
	MaxRetryInterval time.Duration `json:"max_retry_interval"`

	// Resource limits.
	MaxResourcesPerPool int `json:"max_resources_per_pool"`
	MaxResourcesPerType int `json:"max_resources_per_type"`

	// Cleanup and garbage collection.
	CleanupEnabled          bool          `json:"cleanup_enabled"`
	CleanupInterval         time.Duration `json:"cleanup_interval"`
	OrphanedResourceTimeout time.Duration `json:"orphaned_resource_timeout"`
}

// ResourceLifecycleManager defines the interface for resource lifecycle operations.
type ResourceLifecycleManager interface {
	// Provisioning operations.
	ProvisionResource(ctx context.Context, req *ProvisionResourceRequest) (*models.Resource, error)
	ConfigureResource(ctx context.Context, resourceID string, config *runtime.RawExtension) error
	StartResource(ctx context.Context, resourceID string) error
	StopResource(ctx context.Context, resourceID string) error
	RestartResource(ctx context.Context, resourceID string) error

	// Scaling operations.
	ScaleResource(ctx context.Context, resourceID string, req *ScaleResourceRequest) error
	AutoScale(ctx context.Context, resourceID string, policy *AutoScalingPolicy) error

	// Migration operations.
	MigrateResource(ctx context.Context, resourceID string, req *MigrateResourceRequest) error
	ValidateMigration(ctx context.Context, resourceID string, req *MigrateResourceRequest) (*MigrationValidationResult, error)

	// Backup and restore operations.
	BackupResource(ctx context.Context, resourceID string, req *BackupResourceRequest) (*BackupInfo, error)
	RestoreResource(ctx context.Context, resourceID string, backupID string) error
	ListBackups(ctx context.Context, resourceID string) ([]*BackupInfo, error)

	// Termination operations.
	TerminateResource(ctx context.Context, resourceID string, graceful bool) error
	ForceTerminate(ctx context.Context, resourceID string) error

	// State management.
	GetResourceState(ctx context.Context, resourceID string) (*ResourceStateInfo, error)
	UpdateResourceState(ctx context.Context, resourceID string, state string) error
	ValidateStateTransition(ctx context.Context, resourceID string, targetState string) error
}

// ProvisioningQueue defines the interface for managing resource provisioning queue.
type ProvisioningQueue interface {
	// Queue operations.
	EnqueueProvisioningRequest(ctx context.Context, req *ProvisioningQueueEntry) error
	DequeueProvisioningRequest(ctx context.Context) (*ProvisioningQueueEntry, error)
	GetQueueStatus(ctx context.Context) (*QueueStatus, error)

	// Priority management.
	SetRequestPriority(ctx context.Context, requestID string, priority int) error
	ReorderQueue(ctx context.Context, requestIDs []string) error

	// Monitoring and control.
	PauseQueue(ctx context.Context) error
	ResumeQueue(ctx context.Context) error
	ClearQueue(ctx context.Context) error

	// Retry and error handling.
	RetryFailedRequests(ctx context.Context) error
	RemoveFailedRequest(ctx context.Context, requestID string) error
}

// ResourceHealthMonitor defines the interface for resource health monitoring.
type ResourceHealthMonitor interface {
	// Health checking.
	CheckResourceHealth(ctx context.Context, resourceID string) (*models.ResourceHealthInfo, error)
	StartHealthMonitoring(ctx context.Context, resourceID string, config *HealthMonitoringConfig) error
	StopHealthMonitoring(ctx context.Context, resourceID string) error

	// Health policy management.
	SetHealthPolicy(ctx context.Context, resourceID string, policy *HealthPolicy) error
	GetHealthPolicy(ctx context.Context, resourceID string) (*HealthPolicy, error)

	// Health events and alerts.
	GetHealthEvents(ctx context.Context, resourceID string, filter *HealthEventFilter) ([]*HealthEvent, error)
	SubscribeToHealthEvents(ctx context.Context, resourceID string, callback HealthEventCallback) error

	// Health history and trends.
	GetHealthHistory(ctx context.Context, resourceID string, duration time.Duration) ([]*models.HealthHistoryEntry, error)
	GetHealthTrends(ctx context.Context, resourceID string, duration time.Duration) (*HealthTrends, error)
}

// ResourceMetricsCollector defines the interface for resource metrics collection.
type ResourceMetricsCollector interface {
	// Metrics collection.
	CollectResourceMetrics(ctx context.Context, resourceID string, metricNames []string) (*MetricsData, error)
	StartMetricsCollection(ctx context.Context, resourceID string, config *MetricsCollectionConfig) (string, error)
	StopMetricsCollection(ctx context.Context, collectionID string) error

	// Metrics aggregation.
	AggregateMetrics(ctx context.Context, resourceIDs []string, aggregationType string, timeWindow time.Duration) (*AggregatedMetrics, error)
	GetMetricsHistory(ctx context.Context, resourceID string, metricNames []string, duration time.Duration) (*MetricsHistory, error)

	// Metrics alerts.
	SetMetricsAlert(ctx context.Context, resourceID string, alert *MetricsAlert) error
	GetMetricsAlerts(ctx context.Context, resourceID string) ([]*MetricsAlert, error)

	// Custom metrics.
	RegisterCustomMetric(ctx context.Context, metric *CustomMetricDefinition) error
	UnregisterCustomMetric(ctx context.Context, metricName string) error
}

// OperationTracker defines the interface for tracking resource operations.
type OperationTracker interface {
	// Operation tracking.
	StartOperation(ctx context.Context, operation *ResourceOperation) (string, error)
	UpdateOperation(ctx context.Context, operationID string, status *OperationStatus) error
	CompleteOperation(ctx context.Context, operationID string, result *OperationResult) error
	FailOperation(ctx context.Context, operationID string, err error) error

	// Operation queries.
	GetOperation(ctx context.Context, operationID string) (*ResourceOperation, error)
	ListOperations(ctx context.Context, filter *OperationFilter) ([]*ResourceOperation, error)
	GetResourceOperations(ctx context.Context, resourceID string) ([]*ResourceOperation, error)

	// Operation management.
	CancelOperation(ctx context.Context, operationID string) error
	RetryOperation(ctx context.Context, operationID string) error
	CleanupCompletedOperations(ctx context.Context, olderThan time.Duration) error
}

// ResourceValidator defines the interface for resource validation.
type ResourceValidator interface {
	// Configuration validation.
	ValidateResourceConfiguration(ctx context.Context, resourceTypeID string, config *runtime.RawExtension) (*ValidationResult, error)
	ValidateResourcePlacement(ctx context.Context, placement *models.ResourcePlacement) (*ValidationResult, error)

	// Constraint validation.
	ValidateResourceConstraints(ctx context.Context, resourceID string, constraints []*ResourceConstraint) (*ValidationResult, error)
	ValidateCapacityConstraints(ctx context.Context, resourcePoolID string, requirements *models.ResourceRequirements) (*ValidationResult, error)

	// Dependency validation.
	ValidateResourceDependencies(ctx context.Context, resourceID string, dependencies []string) (*ValidationResult, error)
	ValidateCircularDependencies(ctx context.Context, resourceID string, dependencies []string) error

	// Policy validation.
	ValidatePolicyCompliance(ctx context.Context, resourceID string, policies []*ResourcePolicy) (*ValidationResult, error)
	ValidateSecurityPolicies(ctx context.Context, resourceID string) (*ValidationResult, error)
}

// Supporting data structures.

// AutoScalingPolicy defines auto-scaling behavior for resources.
type AutoScalingPolicy struct {
	PolicyID         string           `json:"policyId"`
	Enabled          bool             `json:"enabled"`
	MinInstances     int              `json:"minInstances"`
	MaxInstances     int              `json:"maxInstances"`
	TargetMetrics    []*ScalingMetric `json:"targetMetrics"`
	ScaleUpPolicy    *ScalePolicy     `json:"scaleUpPolicy,omitempty"`
	ScaleDownPolicy  *ScalePolicy     `json:"scaleDownPolicy,omitempty"`
	CooldownPeriod   time.Duration    `json:"cooldownPeriod"`
	EvaluationPeriod time.Duration    `json:"evaluationPeriod"`
}

// ScalingMetric defines a metric used for auto-scaling decisions.
type ScalingMetric struct {
	MetricName         string  `json:"metricName"`
	TargetValue        float64 `json:"targetValue"`
	MetricType         string  `json:"metricType"` // UTILIZATION, AVERAGE_VALUE, AVERAGE_UTILIZATION
	Threshold          float64 `json:"threshold"`
	ComparisonOperator string  `json:"comparisonOperator"` // GT, LT, GE, LE, EQ
}

// MigrationValidationResult represents the result of migration validation.
type MigrationValidationResult struct {
	Valid             bool                         `json:"valid"`
	EstimatedDuration time.Duration                `json:"estimatedDuration,omitempty"`
	RequiredResources *models.ResourceRequirements `json:"requiredResources,omitempty"`
	Warnings          []string                     `json:"warnings,omitempty"`
	Blockers          []string                     `json:"blockers,omitempty"`
	Recommendations   []string                     `json:"recommendations,omitempty"`
}

// ResourceStateInfo represents detailed state information for a resource.
type ResourceStateInfo struct {
	ResourceID       string                 `json:"resourceId"`
	CurrentState     string                 `json:"currentState"`
	PreviousState    string                 `json:"previousState,omitempty"`
	StateTransitions []*StateTransition     `json:"stateTransitions,omitempty"`
	StateMetadata    map[string]interface{} `json:"stateMetadata,omitempty"`
	LastStateChange  time.Time              `json:"lastStateChange"`
}

// StateTransition represents a state transition in resource lifecycle.
type StateTransition struct {
	FromState    string        `json:"fromState"`
	ToState      string        `json:"toState"`
	Timestamp    time.Time     `json:"timestamp"`
	Reason       string        `json:"reason,omitempty"`
	TriggeredBy  string        `json:"triggeredBy,omitempty"`
	Duration     time.Duration `json:"duration,omitempty"`
	Success      bool          `json:"success"`
	ErrorMessage string        `json:"errorMessage,omitempty"`
}

// ProvisioningQueueEntry represents an entry in the provisioning queue.
type ProvisioningQueueEntry struct {
	RequestID      string                    `json:"requestId"`
	ResourceName   string                    `json:"resourceName"`
	ResourceTypeID string                    `json:"resourceTypeId"`
	ResourcePoolID string                    `json:"resourcePoolId"`
	Priority       int                       `json:"priority"`
	Status         string                    `json:"status"` // QUEUED, PROCESSING, COMPLETED, FAILED
	Request        *ProvisionResourceRequest `json:"request"`
	QueuedAt       time.Time                 `json:"queuedAt"`
	StartedAt      *time.Time                `json:"startedAt,omitempty"`
	CompletedAt    *time.Time                `json:"completedAt,omitempty"`
	RetryCount     int                       `json:"retryCount"`
	LastError      string                    `json:"lastError,omitempty"`
	Metadata       map[string]interface{}    `json:"metadata,omitempty"`
}

// QueueStatus represents the status of the provisioning queue.
type QueueStatus struct {
	TotalEntries       int           `json:"totalEntries"`
	QueuedEntries      int           `json:"queuedEntries"`
	ProcessingEntries  int           `json:"processingEntries"`
	CompletedEntries   int           `json:"completedEntries"`
	FailedEntries      int           `json:"failedEntries"`
	AverageWaitTime    time.Duration `json:"averageWaitTime"`
	AverageProcessTime time.Duration `json:"averageProcessTime"`
	QueuePaused        bool          `json:"queuePaused"`
	LastProcessed      *time.Time    `json:"lastProcessed,omitempty"`
}

// HealthMonitoringConfig defines configuration for health monitoring.
type HealthMonitoringConfig struct {
	CheckInterval    time.Duration        `json:"checkInterval"`
	Timeout          time.Duration        `json:"timeout"`
	FailureThreshold int                  `json:"failureThreshold"`
	SuccessThreshold int                  `json:"successThreshold"`
	HealthChecks     []*HealthCheckConfig `json:"healthChecks"`
	AlertOnFailure   bool                 `json:"alertOnFailure"`
	AutoRemediation  bool                 `json:"autoRemediation"`
}

// HealthCheckConfig defines configuration for a specific health check.
type HealthCheckConfig struct {
	Name             string                 `json:"name"`
	Type             string                 `json:"type"` // HTTP, TCP, EXEC, GRPC
	Endpoint         string                 `json:"endpoint,omitempty"`
	Command          []string               `json:"command,omitempty"`
	Timeout          time.Duration          `json:"timeout"`
	ExpectedResponse string                 `json:"expectedResponse,omitempty"`
	FailureThreshold int                    `json:"failureThreshold"`
	Parameters       map[string]interface{} `json:"parameters,omitempty"`
}

// HealthPolicy defines health policies for resources.
type HealthPolicy struct {
	PolicyID           string              `json:"policyId"`
	ResourceID         string              `json:"resourceId"`
	Enabled            bool                `json:"enabled"`
	HealthThreshold    float64             `json:"healthThreshold"`
	Actions            []*HealthAction     `json:"actions"`
	NotificationConfig *NotificationConfig `json:"notificationConfig,omitempty"`
	EvaluationWindow   time.Duration       `json:"evaluationWindow"`
	GracePeriod        time.Duration       `json:"gracePeriod"`
}

// HealthAction defines actions to take based on health status.
type HealthAction struct {
	ActionType string                 `json:"actionType"` // ALERT, RESTART, SCALE, MIGRATE, TERMINATE
	Condition  string                 `json:"condition"`  // Health condition that triggers this action
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Priority   int                    `json:"priority"`
	Enabled    bool                   `json:"enabled"`
}

// HealthEvent represents a health-related event.
type HealthEvent struct {
	EventID        string                 `json:"eventId"`
	ResourceID     string                 `json:"resourceId"`
	EventType      string                 `json:"eventType"` // HEALTH_CHANGED, HEALTH_CHECK_FAILED, HEALTH_RECOVERED
	Severity       string                 `json:"severity"`
	Message        string                 `json:"message"`
	HealthStatus   string                 `json:"healthStatus"`
	PreviousStatus string                 `json:"previousStatus,omitempty"`
	Timestamp      time.Time              `json:"timestamp"`
	Details        map[string]interface{} `json:"details,omitempty"`
}

// HealthEventFilter defines filters for health events.
type HealthEventFilter struct {
	EventTypes []string   `json:"eventTypes,omitempty"`
	Severities []string   `json:"severities,omitempty"`
	StartTime  *time.Time `json:"startTime,omitempty"`
	EndTime    *time.Time `json:"endTime,omitempty"`
	Limit      int        `json:"limit,omitempty"`
	Offset     int        `json:"offset,omitempty"`
}

// HealthEventCallback defines callback function for health events.
type HealthEventCallback func(event *HealthEvent) error

// HealthTrends represents health trends over time.
type HealthTrends struct {
	ResourceID       string              `json:"resourceId"`
	Period           time.Duration       `json:"period"`
	OverallTrend     string              `json:"overallTrend"` // IMPROVING, STABLE, DEGRADING
	HealthScore      *TrendData          `json:"healthScore,omitempty"`
	AvailabilityRate float64             `json:"availabilityRate"`
	IncidentCount    int                 `json:"incidentCount"`
	MTBF             time.Duration       `json:"mtbf,omitempty"` // Mean Time Between Failures
	MTTR             time.Duration       `json:"mttr,omitempty"` // Mean Time To Recovery
	TrendData        []*HealthTrendPoint `json:"trendData,omitempty"`
}

// TrendData represents trend information for a metric.
type TrendData struct {
	CurrentValue  float64 `json:"currentValue"`
	PreviousValue float64 `json:"previousValue"`
	ChangePercent float64 `json:"changePercent"`
	Trend         string  `json:"trend"` // UP, DOWN, STABLE
}

// HealthTrendPoint represents a point in health trend data.
type HealthTrendPoint struct {
	Timestamp     time.Time `json:"timestamp"`
	HealthScore   float64   `json:"healthScore"`
	Status        string    `json:"status"`
	IncidentCount int       `json:"incidentCount"`
}

// AggregatedMetrics represents aggregated metrics across resources.
type AggregatedMetrics struct {
	AggregationType string             `json:"aggregationType"` // SUM, AVERAGE, MIN, MAX, COUNT
	TimeWindow      time.Duration      `json:"timeWindow"`
	ResourceCount   int                `json:"resourceCount"`
	Metrics         map[string]float64 `json:"metrics"`
	Timestamp       time.Time          `json:"timestamp"`
	GroupBy         []string           `json:"groupBy,omitempty"`
}

// MetricsHistory represents historical metrics data.
type MetricsHistory struct {
	ResourceID  string              `json:"resourceId"`
	MetricNames []string            `json:"metricNames"`
	Period      time.Duration       `json:"period"`
	DataPoints  []*MetricsDataPoint `json:"dataPoints"`
	Aggregation string              `json:"aggregation,omitempty"`
	Resolution  time.Duration       `json:"resolution"`
}

// MetricsDataPoint represents a single data point in metrics history.
type MetricsDataPoint struct {
	Timestamp time.Time          `json:"timestamp"`
	Values    map[string]float64 `json:"values"`
	Labels    map[string]string  `json:"labels,omitempty"`
}

// MetricsAlert defines an alert based on metrics.
type MetricsAlert struct {
	AlertID            string              `json:"alertId"`
	ResourceID         string              `json:"resourceId"`
	MetricName         string              `json:"metricName"`
	Condition          string              `json:"condition"` // GT, LT, GE, LE, EQ, NE
	Threshold          float64             `json:"threshold"`
	EvaluationWindow   time.Duration       `json:"evaluationWindow"`
	AlertAction        string              `json:"alertAction"` // NOTIFY, SCALE, RESTART
	Enabled            bool                `json:"enabled"`
	NotificationConfig *NotificationConfig `json:"notificationConfig,omitempty"`
	LastTriggered      *time.Time          `json:"lastTriggered,omitempty"`
}

// CustomMetricDefinition defines a custom metric.
type CustomMetricDefinition struct {
	Name             string                 `json:"name"`
	Description      string                 `json:"description,omitempty"`
	Unit             string                 `json:"unit"`
	MetricType       string                 `json:"metricType"`       // GAUGE, COUNTER, HISTOGRAM
	CollectionMethod string                 `json:"collectionMethod"` // PULL, PUSH, CALCULATED
	CollectionConfig map[string]interface{} `json:"collectionConfig,omitempty"`
	Labels           []string               `json:"labels,omitempty"`
}

// ResourceOperation represents a resource operation.
type ResourceOperation struct {
	OperationID   string                 `json:"operationId"`
	ResourceID    string                 `json:"resourceId"`
	OperationType string                 `json:"operationType"` // PROVISION, CONFIGURE, SCALE, MIGRATE, TERMINATE
	Status        *OperationStatus       `json:"status"`
	Request       interface{}            `json:"request,omitempty"`
	Result        *OperationResult       `json:"result,omitempty"`
	StartedAt     time.Time              `json:"startedAt"`
	CompletedAt   *time.Time             `json:"completedAt,omitempty"`
	CreatedBy     string                 `json:"createdBy,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// OperationStatus represents the status of an operation.
type OperationStatus struct {
	State               string     `json:"state"`    // PENDING, RUNNING, COMPLETED, FAILED, CANCELLED
	Progress            float64    `json:"progress"` // 0.0 to 1.0
	Message             string     `json:"message,omitempty"`
	CurrentStep         string     `json:"currentStep,omitempty"`
	TotalSteps          int        `json:"totalSteps,omitempty"`
	CompletedSteps      int        `json:"completedSteps,omitempty"`
	EstimatedCompletion *time.Time `json:"estimatedCompletion,omitempty"`
	LastUpdated         time.Time  `json:"lastUpdated"`
}

// OperationResult represents the result of an operation.
type OperationResult struct {
	Success    bool                   `json:"success"`
	ResourceID string                 `json:"resourceId,omitempty"`
	Message    string                 `json:"message,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Errors     []string               `json:"errors,omitempty"`
	Warnings   []string               `json:"warnings,omitempty"`
	Duration   time.Duration          `json:"duration"`
	OutputData interface{}            `json:"outputData,omitempty"`
}

// OperationFilter defines filters for operation queries.
type OperationFilter struct {
	ResourceIDs     []string   `json:"resourceIds,omitempty"`
	OperationTypes  []string   `json:"operationTypes,omitempty"`
	States          []string   `json:"states,omitempty"`
	CreatedBy       []string   `json:"createdBy,omitempty"`
	StartedAfter    *time.Time `json:"startedAfter,omitempty"`
	StartedBefore   *time.Time `json:"startedBefore,omitempty"`
	CompletedAfter  *time.Time `json:"completedAfter,omitempty"`
	CompletedBefore *time.Time `json:"completedBefore,omitempty"`
	Limit           int        `json:"limit,omitempty"`
	Offset          int        `json:"offset,omitempty"`
	SortBy          string     `json:"sortBy,omitempty"`
	SortOrder       string     `json:"sortOrder,omitempty"`
}

// ResourceConstraint defines a constraint on resource configuration or behavior.
type ResourceConstraint struct {
	ConstraintID string                 `json:"constraintId"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"` // CAPACITY, PLACEMENT, SECURITY, POLICY
	Description  string                 `json:"description,omitempty"`
	Expression   string                 `json:"expression"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	Severity     string                 `json:"severity"`  // ERROR, WARNING, INFO
	EnforceAt    string                 `json:"enforceAt"` // CREATION, RUNTIME, BOTH
}

// ResourcePolicy defines a policy that applies to resources.
type ResourcePolicy struct {
	PolicyID        string        `json:"policyId"`
	Name            string        `json:"name"`
	Type            string        `json:"type"` // SECURITY, COMPLIANCE, GOVERNANCE, OPERATIONAL
	Description     string        `json:"description,omitempty"`
	Rules           []*PolicyRule `json:"rules"`
	Scope           *PolicyScope  `json:"scope,omitempty"`
	Enabled         bool          `json:"enabled"`
	Priority        int           `json:"priority"`
	EnforcementMode string        `json:"enforcementMode"` // ENFORCED, MONITOR, DISABLED
}

// PolicyRule defines a rule within a policy.
type PolicyRule struct {
	RuleID     string                 `json:"ruleId"`
	Name       string                 `json:"name"`
	Condition  string                 `json:"condition"`
	Action     string                 `json:"action"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Enabled    bool                   `json:"enabled"`
}

// PolicyScope defines the scope of a policy.
type PolicyScope struct {
	ResourceTypes []string          `json:"resourceTypes,omitempty"`
	ResourcePools []string          `json:"resourcePools,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	Namespaces    []string          `json:"namespaces,omitempty"`
}

// NewInfrastructureResourceManager creates a new infrastructure resource manager.
func NewInfrastructureResourceManager(storage O2IMSStorage, kubeClient client.Client, logger *logging.StructuredLogger) *InfrastructureResourceManager {
	config := &ResourceManagerConfig{
		CacheEnabled:              true,
		CacheExpiry:               15 * time.Minute,
		MaxCacheSize:              1000,
		DefaultProvisionTimeout:   10 * time.Minute,
		DefaultTerminationTimeout: 5 * time.Minute,
		MaxConcurrentOperations:   50,
		AutoDiscoveryEnabled:      true,
		DiscoveryInterval:         5 * time.Minute,
		StateSyncEnabled:          true,
		StateSyncInterval:         30 * time.Second,
		HealthMonitoringEnabled:   true,
		HealthCheckInterval:       30 * time.Second,
		HealthCheckTimeout:        10 * time.Second,
		MetricsCollectionEnabled:  true,
		MetricsCollectionInterval: 30 * time.Second,
		MetricsRetentionPeriod:    24 * time.Hour,
		ValidationEnabled:         true,
		StrictValidation:          false,
		ValidationTimeout:         30 * time.Second,
		MaxRetries:                3,
		RetryInterval:             30 * time.Second,
		BackoffFactor:             2.0,
		MaxRetryInterval:          5 * time.Minute,
		MaxResourcesPerPool:       1000,
		MaxResourcesPerType:       500,
		CleanupEnabled:            true,
		CleanupInterval:           1 * time.Hour,
		OrphanedResourceTimeout:   24 * time.Hour,
	}

	return &InfrastructureResourceManager{
		storage:          storage,
		kubeClient:       kubeClient,
		logger:           logger,
		config:           config,
		resourceCache:    make(map[string]*models.Resource),
		cacheExpiry:      config.CacheExpiry,
		discoveryEnabled: config.AutoDiscoveryEnabled,
		syncEnabled:      config.StateSyncEnabled,
	}
}

// GetResources retrieves infrastructure resources with filtering support.
func (irm *InfrastructureResourceManager) GetResources(ctx context.Context, filter *models.ResourceFilter) ([]*models.Resource, error) {
	logger := log.FromContext(ctx)
	logger.Info("retrieving infrastructure resources", "filter", filter)

	// Check cache first if enabled and no specific filter.
	if irm.config.CacheEnabled && filter == nil {
		if resources := irm.getCachedResources(ctx); len(resources) > 0 {
			return resources, nil
		}
	}

	// Retrieve from storage.
	resources, err := irm.storage.ListResources(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	// Update cache if enabled.
	if irm.config.CacheEnabled {
		irm.updateResourceCache(resources)
	}

	// Enrich with real-time data if monitoring is enabled.
	if irm.config.HealthMonitoringEnabled || irm.config.MetricsCollectionEnabled {
		if err := irm.enrichResourcesWithRealTimeData(ctx, resources); err != nil {
			logger.Info("failed to enrich resources with real-time data", "error", err)
		}
	}

	logger.Info("retrieved infrastructure resources", "count", len(resources))
	return resources, nil
}

// GetResource retrieves a specific infrastructure resource.
func (irm *InfrastructureResourceManager) GetResource(ctx context.Context, resourceID string) (*models.Resource, error) {
	logger := log.FromContext(ctx)
	logger.Info("retrieving infrastructure resource", "resourceId", resourceID)

	// Check cache first.
	if irm.config.CacheEnabled {
		if resource := irm.getCachedResource(resourceID); resource != nil {
			return resource, nil
		}
	}

	// Retrieve from storage.
	resource, err := irm.storage.GetResource(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource %s: %w", resourceID, err)
	}

	// Update cache.
	if irm.config.CacheEnabled {
		irm.setCachedResource(resource)
	}

	// Enrich with real-time data.
	if irm.config.HealthMonitoringEnabled || irm.config.MetricsCollectionEnabled {
		if err := irm.enrichResourceWithRealTimeData(ctx, resource); err != nil {
			logger.Info("failed to enrich resource with real-time data", "error", err)
		}
	}

	return resource, nil
}

// CreateResource creates a new infrastructure resource.
func (irm *InfrastructureResourceManager) CreateResource(ctx context.Context, req *models.CreateResourceRequest) (*models.Resource, error) {
	logger := log.FromContext(ctx)
	logger.Info("creating infrastructure resource", "name", req.Name, "type", req.ResourceTypeID)

	// Start operation tracking.
	operation := &ResourceOperation{
		OperationID:   irm.generateOperationID(),
		OperationType: "CREATE",
		Request:       req,
		StartedAt:     time.Now(),
		Status: &OperationStatus{
			State:    "RUNNING",
			Progress: 0.0,
			Message:  "Creating resource",
		},
	}

	var operationID string
	if irm.operationTracker != nil {
		var err error
		operationID, err = irm.operationTracker.StartOperation(ctx, operation)
		if err != nil {
			logger.Info("failed to start operation tracking", "error", err)
		}
	}

	// Validate the request.
	if err := irm.validateCreateResourceRequest(ctx, req); err != nil {
		if operationID != "" {
			irm.operationTracker.FailOperation(ctx, operationID, err)
		}
		return nil, fmt.Errorf("invalid create resource request: %w", err)
	}

	// Create the resource.
	resource := &models.Resource{
		ResourceID:       irm.generateResourceID(req.Name, req.ResourceTypeID),
		Name:             req.Name,
		Description:      req.Description,
		ResourceTypeID:   req.ResourceTypeID,
		ResourcePoolID:   req.ResourcePoolID,
		ParentResourceID: req.ParentResourceID,
		Configuration:    req.Configuration,
		Placement:        req.PlacementConstraints,
		Tags:             req.Tags,
		Labels:           req.Labels,
		Extensions:       req.Extensions,
		Status: &models.ResourceStatus{
			State:           models.LifecycleStateProvisioning,
			Health:          models.ResourceHealthUnknown,
			ErrorMessage:    "Resource provisioning initiated",
			LastHealthCheck: time.Now(),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store the resource.
	if err := irm.storage.StoreResource(ctx, resource); err != nil {
		if operationID != "" {
			irm.operationTracker.FailOperation(ctx, operationID, err)
		}
		return nil, fmt.Errorf("failed to store resource: %w", err)
	}

	// Update operation tracking.
	if operationID != "" {
		operation.ResourceID = resource.ResourceID
		irm.operationTracker.UpdateOperation(ctx, operationID, &OperationStatus{
			State:    "RUNNING",
			Progress: 0.3,
			Message:  "Resource stored, initiating provisioning",
		})
	}

	// Update cache.
	if irm.config.CacheEnabled {
		irm.setCachedResource(resource)
	}

	// Initiate provisioning if lifecycle manager is available.
	if irm.lifecycleManager != nil {
		provisionReq := &ProvisionResourceRequest{
			Name:           req.Name,
			ResourceType:   req.ResourceTypeID,
			ResourcePoolID: req.ResourcePoolID,
			Configuration:  req.Configuration,
			Requirements:   nil, // Would be derived from resource type
			Tags:           req.Tags,
			Metadata:       req.Extensions,
		}

		if _, err := irm.lifecycleManager.ProvisionResource(ctx, provisionReq); err != nil {
			logger.Info("provisioning failed, resource will remain in provisioning state", "error", err)
			resource.Status.State = models.LifecycleStateFailed
			resource.Status.ErrorMessage = fmt.Sprintf("Provisioning failed: %v", err)
			irm.storage.UpdateResource(ctx, resource.ResourceID, map[string]interface{}{
				"status": resource.Status,
			})

			if operationID != "" {
				irm.operationTracker.FailOperation(ctx, operationID, err)
			}
		} else {
			resource.Status.State = models.LifecycleStateActive
			resource.Status.Health = models.ResourceHealthHealthy
			resource.Status.ErrorMessage = ""
			irm.storage.UpdateResource(ctx, resource.ResourceID, map[string]interface{}{
				"status": resource.Status,
			})

			if operationID != "" {
				irm.operationTracker.CompleteOperation(ctx, operationID, &OperationResult{
					Success:    true,
					ResourceID: resource.ResourceID,
					Message:    "Resource created successfully",
				})
			}
		}
	}

	// Start monitoring if enabled.
	if irm.config.HealthMonitoringEnabled && irm.healthMonitor != nil {
		if err := irm.startResourceMonitoring(ctx, resource.ResourceID); err != nil {
			logger.Info("failed to start resource monitoring", "error", err)
		}
	}

	logger.Info("infrastructure resource created", "resourceId", resource.ResourceID)
	return resource, nil
}

// UpdateResource updates an existing infrastructure resource.
func (irm *InfrastructureResourceManager) UpdateResource(ctx context.Context, resourceID string, req *models.UpdateResourceRequest) (*models.Resource, error) {
	logger := log.FromContext(ctx)
	logger.Info("updating infrastructure resource", "resourceId", resourceID)

	// Get current resource.
	resource, err := irm.GetResource(ctx, resourceID)
	if err != nil {
		return nil, err
	}

	// Start operation tracking.
	operation := &ResourceOperation{
		OperationID:   irm.generateOperationID(),
		ResourceID:    resourceID,
		OperationType: "UPDATE",
		Request:       req,
		StartedAt:     time.Now(),
		Status: &OperationStatus{
			State:    "RUNNING",
			Progress: 0.0,
			Message:  "Updating resource",
		},
	}

	var operationID string
	if irm.operationTracker != nil {
		operationID, err = irm.operationTracker.StartOperation(ctx, operation)
		if err != nil {
			logger.Info("failed to start operation tracking", "error", err)
		}
	}

	// Apply updates.
	updates := make(map[string]interface{})
	if req.Name != nil {
		resource.Name = *req.Name
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		resource.Description = *req.Description
		updates["description"] = *req.Description
	}
	if req.Configuration != nil {
		resource.Configuration = req.Configuration
		updates["configuration"] = req.Configuration
	}
	if req.Tags != nil {
		resource.Tags = req.Tags
		updates["tags"] = req.Tags
	}
	if req.Labels != nil {
		resource.Labels = req.Labels
		updates["labels"] = req.Labels
	}
	if req.Extensions != nil {
		resource.Extensions = req.Extensions
		updates["extensions"] = req.Extensions
	}

	resource.UpdatedAt = time.Now()
	updates["updated_at"] = resource.UpdatedAt

	// Update in storage.
	if err := irm.storage.UpdateResource(ctx, resourceID, updates); err != nil {
		if operationID != "" {
			irm.operationTracker.FailOperation(ctx, operationID, err)
		}
		return nil, fmt.Errorf("failed to update resource: %w", err)
	}

	// Update cache.
	if irm.config.CacheEnabled {
		irm.setCachedResource(resource)
	}

	// Apply configuration changes if lifecycle manager is available.
	if req.Configuration != nil && irm.lifecycleManager != nil {
		if err := irm.lifecycleManager.ConfigureResource(ctx, resourceID, req.Configuration); err != nil {
			logger.Info("configuration update failed", "error", err)
			if operationID != "" {
				irm.operationTracker.FailOperation(ctx, operationID, err)
			}
		}
	}

	// Complete operation tracking.
	if operationID != "" {
		irm.operationTracker.CompleteOperation(ctx, operationID, &OperationResult{
			Success:    true,
			ResourceID: resourceID,
			Message:    "Resource updated successfully",
		})
	}

	logger.Info("infrastructure resource updated", "resourceId", resourceID)
	return resource, nil
}

// DeleteResource deletes an infrastructure resource.
func (irm *InfrastructureResourceManager) DeleteResource(ctx context.Context, resourceID string) error {
	logger := log.FromContext(ctx)
	logger.Info("deleting infrastructure resource", "resourceId", resourceID)

	// Check if resource exists.
	resource, err := irm.GetResource(ctx, resourceID)
	if err != nil {
		return err
	}

	// Check for dependent resources.
	if err := irm.checkResourceDependencies(ctx, resourceID); err != nil {
		return fmt.Errorf("cannot delete resource with dependencies: %w", err)
	}

	// Start operation tracking.
	operation := &ResourceOperation{
		OperationID:   irm.generateOperationID(),
		ResourceID:    resourceID,
		OperationType: "DELETE",
		StartedAt:     time.Now(),
		Status: &OperationStatus{
			State:    "RUNNING",
			Progress: 0.0,
			Message:  "Deleting resource",
		},
	}

	var operationID string
	if irm.operationTracker != nil {
		operationID, err = irm.operationTracker.StartOperation(ctx, operation)
		if err != nil {
			logger.Info("failed to start operation tracking", "error", err)
		}
	}

	// Stop monitoring if enabled.
	if irm.config.HealthMonitoringEnabled && irm.healthMonitor != nil {
		if err := irm.stopResourceMonitoring(ctx, resourceID); err != nil {
			logger.Info("failed to stop resource monitoring", "error", err)
		}
	}

	// Terminate resource if lifecycle manager is available.
	if irm.lifecycleManager != nil {
		if err := irm.lifecycleManager.TerminateResource(ctx, resourceID, false); err != nil {
			logger.Info("resource termination failed, proceeding with deletion", "error", err)
		}
	}

	// Update resource status before deletion.
	resource.Status.State = models.LifecycleStateTerminating
	resource.Status.ErrorMessage = ""
	irm.storage.UpdateResource(ctx, resourceID, map[string]interface{}{
		"status": resource.Status,
	})

	// Delete from storage.
	if err := irm.storage.DeleteResource(ctx, resourceID); err != nil {
		if operationID != "" {
			irm.operationTracker.FailOperation(ctx, operationID, err)
		}
		return fmt.Errorf("failed to delete resource: %w", err)
	}

	// Remove from cache.
	if irm.config.CacheEnabled {
		irm.removeCachedResource(resourceID)
	}

	// Complete operation tracking.
	if operationID != "" {
		irm.operationTracker.CompleteOperation(ctx, operationID, &OperationResult{
			Success:    true,
			ResourceID: resourceID,
			Message:    "Resource deleted successfully",
		})
	}

	logger.Info("infrastructure resource deleted", "resourceId", resourceID)
	return nil
}

// Private helper methods.

// validateCreateResourceRequest validates a create resource request.
func (irm *InfrastructureResourceManager) validateCreateResourceRequest(ctx context.Context, req *models.CreateResourceRequest) error {
	if req.Name == "" {
		return fmt.Errorf("resource name is required")
	}
	if req.ResourceTypeID == "" {
		return fmt.Errorf("resource type ID is required")
	}
	if req.ResourcePoolID == "" {
		return fmt.Errorf("resource pool ID is required")
	}

	// Additional validation through validator if available.
	if irm.config.ValidationEnabled && irm.validator != nil {
		if req.Configuration != nil {
			if result, err := irm.validator.ValidateResourceConfiguration(ctx, req.ResourceTypeID, req.Configuration); err != nil {
				return fmt.Errorf("configuration validation failed: %w", err)
			} else if !result.Valid {
				return fmt.Errorf("configuration validation failed: %v", result.Errors)
			}
		}

		if req.PlacementConstraints != nil {
			if result, err := irm.validator.ValidateResourcePlacement(ctx, req.PlacementConstraints); err != nil {
				return fmt.Errorf("placement validation failed: %w", err)
			} else if !result.Valid {
				return fmt.Errorf("placement validation failed: %v", result.Errors)
			}
		}
	}

	return nil
}

// generateResourceID generates a unique resource ID.
func (irm *InfrastructureResourceManager) generateResourceID(name, resourceTypeID string) string {
	return fmt.Sprintf("%s-%s-%d", resourceTypeID, name, time.Now().Unix())
}

// generateOperationID generates a unique operation ID.
func (irm *InfrastructureResourceManager) generateOperationID() string {
	return fmt.Sprintf("op-%d", time.Now().UnixNano())
}

// checkResourceDependencies checks if there are resources depending on this resource.
func (irm *InfrastructureResourceManager) checkResourceDependencies(ctx context.Context, resourceID string) error {
	// Check for dependent resources.
	filter := &models.ResourceFilter{
		ParentResourceIDs: []string{resourceID},
		Limit:             1,
	}

	resources, err := irm.storage.ListResources(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to check dependencies: %w", err)
	}

	if len(resources) > 0 {
		return fmt.Errorf("resource has %d dependent resources", len(resources))
	}

	return nil
}

// startResourceMonitoring starts monitoring for a resource.
func (irm *InfrastructureResourceManager) startResourceMonitoring(ctx context.Context, resourceID string) error {
	if irm.healthMonitor == nil {
		return nil
	}

	config := &HealthMonitoringConfig{
		CheckInterval:    irm.config.HealthCheckInterval,
		Timeout:          irm.config.HealthCheckTimeout,
		FailureThreshold: 3,
		SuccessThreshold: 1,
		AlertOnFailure:   true,
		AutoRemediation:  false,
	}

	return irm.healthMonitor.StartHealthMonitoring(ctx, resourceID, config)
}

// stopResourceMonitoring stops monitoring for a resource.
func (irm *InfrastructureResourceManager) stopResourceMonitoring(ctx context.Context, resourceID string) error {
	if irm.healthMonitor == nil {
		return nil
	}

	return irm.healthMonitor.StopHealthMonitoring(ctx, resourceID)
}

// enrichResourcesWithRealTimeData enriches resources with real-time health and metrics data.
func (irm *InfrastructureResourceManager) enrichResourcesWithRealTimeData(ctx context.Context, resources []*models.Resource) error {
	for _, resource := range resources {
		if err := irm.enrichResourceWithRealTimeData(ctx, resource); err != nil {
			irm.logger.Error("failed to enrich resource with real-time data", "resourceId", resource.ResourceID, "error", err)
		}
	}
	return nil
}

// enrichResourceWithRealTimeData enriches a single resource with real-time data.
func (irm *InfrastructureResourceManager) enrichResourceWithRealTimeData(ctx context.Context, resource *models.Resource) error {
	// Enrich with health information.
	if irm.config.HealthMonitoringEnabled && irm.healthMonitor != nil {
		health, err := irm.healthMonitor.CheckResourceHealth(ctx, resource.ResourceID)
		if err == nil && health != nil {
			resource.Health = health
		}
	}

	// Enrich with metrics.
	if irm.config.MetricsCollectionEnabled && irm.metricsCollector != nil {
		metrics, err := irm.metricsCollector.CollectResourceMetrics(ctx, resource.ResourceID, []string{"cpu", "memory", "storage"})
		if err == nil && metrics != nil {
			resource.Metrics = metrics.Metrics
		}
	}

	return nil
}

// Cache management methods.

// getCachedResources returns all cached resources.
func (irm *InfrastructureResourceManager) getCachedResources(ctx context.Context) []*models.Resource {
	irm.cacheMu.RLock()
	defer irm.cacheMu.RUnlock()

	resources := make([]*models.Resource, 0, len(irm.resourceCache))
	for _, resource := range irm.resourceCache {
		resources = append(resources, resource)
	}
	return resources
}

// getCachedResource returns a cached resource by ID.
func (irm *InfrastructureResourceManager) getCachedResource(resourceID string) *models.Resource {
	irm.cacheMu.RLock()
	defer irm.cacheMu.RUnlock()
	return irm.resourceCache[resourceID]
}

// setCachedResource adds or updates a resource in the cache.
func (irm *InfrastructureResourceManager) setCachedResource(resource *models.Resource) {
	irm.cacheMu.Lock()
	defer irm.cacheMu.Unlock()
	irm.resourceCache[resource.ResourceID] = resource
}

// updateResourceCache updates multiple resources in the cache.
func (irm *InfrastructureResourceManager) updateResourceCache(resources []*models.Resource) {
	irm.cacheMu.Lock()
	defer irm.cacheMu.Unlock()
	for _, resource := range resources {
		irm.resourceCache[resource.ResourceID] = resource
	}
}

// removeCachedResource removes a resource from the cache.
func (irm *InfrastructureResourceManager) removeCachedResource(resourceID string) {
	irm.cacheMu.Lock()
	defer irm.cacheMu.Unlock()
	delete(irm.resourceCache, resourceID)
}

// Component setters.

// SetLifecycleManager sets the resource lifecycle manager.
func (irm *InfrastructureResourceManager) SetLifecycleManager(manager ResourceLifecycleManager) {
	irm.lifecycleManager = manager
}

// SetProvisioningQueue sets the provisioning queue.
func (irm *InfrastructureResourceManager) SetProvisioningQueue(queue ProvisioningQueue) {
	irm.provisioningQueue = queue
}

// SetHealthMonitor sets the resource health monitor.
func (irm *InfrastructureResourceManager) SetHealthMonitor(monitor ResourceHealthMonitor) {
	irm.healthMonitor = monitor
}

// SetMetricsCollector sets the resource metrics collector.
func (irm *InfrastructureResourceManager) SetMetricsCollector(collector ResourceMetricsCollector) {
	irm.metricsCollector = collector
}

// SetOperationTracker sets the operation tracker.
func (irm *InfrastructureResourceManager) SetOperationTracker(tracker OperationTracker) {
	irm.operationTracker = tracker
}

// SetValidator sets the resource validator.
func (irm *InfrastructureResourceManager) SetValidator(validator ResourceValidator) {
	irm.validator = validator
}

// GetConfig returns the current configuration.
func (irm *InfrastructureResourceManager) GetConfig() *ResourceManagerConfig {
	return irm.config
}

// UpdateConfig updates the configuration.
func (irm *InfrastructureResourceManager) UpdateConfig(config *ResourceManagerConfig) {
	irm.config = config
	irm.cacheExpiry = config.CacheExpiry
	irm.discoveryEnabled = config.AutoDiscoveryEnabled
	irm.syncEnabled = config.StateSyncEnabled
}
