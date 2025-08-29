package o2

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/runtime"

	o2models "github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
)

// ===== SHARED UTILITY TYPES =====

// RetryPolicy defines retry behavior for operations
type RetryPolicy struct {
	MaxRetries      int           `json:"maxRetries"`
	RetryDelay      time.Duration `json:"retryDelay"`
	BackoffFactor   float64       `json:"backoffFactor"`
	MaxRetryDelay   time.Duration `json:"maxRetryDelay"`
	RetryConditions []string      `json:"retryConditions,omitempty"`
}

// ResourceRequirements defines resource requirements
type ResourceRequirements struct {
	CPU              string                 `json:"cpu,omitempty"`
	Memory           string                 `json:"memory,omitempty"`
	Storage          string                 `json:"storage,omitempty"`
	NetworkBandwidth string                 `json:"networkBandwidth,omitempty"`
	MinNodes         int                    `json:"minNodes,omitempty"`
	MaxNodes         int                    `json:"maxNodes,omitempty"`
	RequiredFeatures []string               `json:"requiredFeatures,omitempty"`
	Constraints      map[string]interface{} `json:"constraints,omitempty"`
	Affinity         *AffinityRules         `json:"affinity,omitempty"`
}

// AffinityRules represents affinity and anti-affinity rules
type AffinityRules struct {
	NodeAffinity    *NodeAffinity `json:"nodeAffinity,omitempty"`
	PodAffinity     *PodAffinity  `json:"podAffinity,omitempty"`
	PodAntiAffinity *PodAffinity  `json:"podAntiAffinity,omitempty"`
}

// NodeAffinity represents node affinity rules
type NodeAffinity struct {
	RequiredDuringSchedulingIgnoredDuringExecution  *NodeSelector              `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`
	PreferredDuringSchedulingIgnoredDuringExecution []*PreferredSchedulingTerm `json:"preferredDuringSchedulingIgnoredDuringExecution,omitempty"`
}

// NodeSelector represents a node selector
type NodeSelector struct {
	NodeSelectorTerms []*NodeSelectorTerm `json:"nodeSelectorTerms"`
}

// NodeSelectorTerm represents a node selector term
type NodeSelectorTerm struct {
	MatchExpressions []*NodeSelectorRequirement `json:"matchExpressions,omitempty"`
	MatchFields      []*NodeSelectorRequirement `json:"matchFields,omitempty"`
}

// NodeSelectorRequirement represents a node selector requirement
type NodeSelectorRequirement struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values,omitempty"`
}

// PreferredSchedulingTerm represents a preferred scheduling term
type PreferredSchedulingTerm struct {
	Weight     int32            `json:"weight"`
	Preference *NodeSelectorTerm `json:"preference"`
}

// PodAffinity represents pod affinity rules
type PodAffinity struct {
	RequiredDuringSchedulingIgnoredDuringExecution  []*PodAffinityTerm         `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`
	PreferredDuringSchedulingIgnoredDuringExecution []*WeightedPodAffinityTerm `json:"preferredDuringSchedulingIgnoredDuringExecution,omitempty"`
}

// PodAffinityTerm represents a pod affinity term
type PodAffinityTerm struct {
	LabelSelector *LabelSelector `json:"labelSelector,omitempty"`
	Namespaces    []string       `json:"namespaces,omitempty"`
	TopologyKey   string         `json:"topologyKey"`
}

// WeightedPodAffinityTerm represents a weighted pod affinity term
type WeightedPodAffinityTerm struct {
	Weight          int32            `json:"weight"`
	PodAffinityTerm *PodAffinityTerm `json:"podAffinityTerm"`
}

// LabelSelector represents a label selector
type LabelSelector struct {
	MatchLabels      map[string]string       `json:"matchLabels,omitempty"`
	MatchExpressions []*LabelSelectorRequirement `json:"matchExpressions,omitempty"`
}

// LabelSelectorRequirement represents a label selector requirement
type LabelSelectorRequirement struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values,omitempty"`
}

// ResourceStatus represents the status of a resource
type ResourceStatus struct {
	State          string                 `json:"state"`
	Phase          string                 `json:"phase"`
	Health         string                 `json:"health"`
	Conditions     []ResourceCondition    `json:"conditions,omitempty"`
	ErrorMessage   string                 `json:"errorMessage,omitempty"`
	LastUpdated    time.Time              `json:"lastUpdated"`
	Metrics        map[string]interface{} `json:"metrics,omitempty"`
	ResourceUsage  *ResourceUsage         `json:"resourceUsage,omitempty"`
}

// ResourceCondition represents a condition of the resource
type ResourceCondition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
	LastTransitionTime time.Time `json:"lastTransitionTime"`
	LastUpdateTime     time.Time `json:"lastUpdateTime,omitempty"`
}

// ResourceMetric represents a resource metric
type ResourceMetric struct {
	Used      string  `json:"used"`
	Available string  `json:"available"`
	Usage     float64 `json:"usage"` // percentage
}

// NetworkUsage represents network usage metrics
type NetworkUsage struct {
	BytesIn       int64   `json:"bytesIn"`
	BytesOut      int64   `json:"bytesOut"`
	PacketsIn     int64   `json:"packetsIn"`
	PacketsOut    int64   `json:"packetsOut"`
	ErrorsIn      int64   `json:"errorsIn"`
	ErrorsOut     int64   `json:"errorsOut"`
	Throughput    float64 `json:"throughput"`
}

// Resource management operation types

// ProvisionResourceRequest represents a request to provision a resource
type ProvisionResourceRequest struct {
	Name           string                 `json:"name"`
	ResourceType   string                 `json:"resourceType"`
	ResourcePoolID string                 `json:"resourcePoolId"`
	Provider       string                 `json:"provider,omitempty"`        // Added Provider field
	Configuration  *runtime.RawExtension  `json:"configuration"`
	Requirements   *ResourceRequirements  `json:"requirements,omitempty"`
	Tags           map[string]string      `json:"tags,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ScaleResourceRequest represents a request to scale a resource
type ScaleResourceRequest struct {
	ScaleType       string                 `json:"scaleType"` // UP, DOWN, OUT, IN
	TargetSize      int                    `json:"targetSize,omitempty"`
	TargetReplicas  int32                  `json:"targetReplicas,omitempty"`  // Added TargetReplicas field
	Percentage      int                    `json:"percentage,omitempty"`
	Resources       *ResourceRequirements  `json:"resources,omitempty"`
	Options         map[string]interface{} `json:"options,omitempty"`
}

// MigrateResourceRequest represents a request to migrate a resource
type MigrateResourceRequest struct {
	TargetPoolID        string                 `json:"targetPoolId"`
	TargetProvider      string                 `json:"targetProvider,omitempty"`   // Added TargetProvider field
	SourceProvider      string                 `json:"sourceProvider,omitempty"`   // Added SourceProvider field
	MigrationType       string                 `json:"migrationType"` // LIVE, COLD, HOT
	PreservePersistence bool                   `json:"preservePersistence"`
	Options             map[string]interface{} `json:"options,omitempty"`
}

// BackupResourceRequest represents a request to backup a resource
type BackupResourceRequest struct {
	BackupType      string                 `json:"backupType"` // FULL, INCREMENTAL, DIFFERENTIAL
	StorageLocation string                 `json:"storageLocation,omitempty"`
	Retention       time.Duration          `json:"retention,omitempty"`
	Compression     bool                   `json:"compression"`
	Options         map[string]interface{} `json:"options,omitempty"`
}

// BackupInfo represents information about a backup
type BackupInfo struct {
	BackupID    string                 `json:"backupId"`
	ResourceID  string                 `json:"resourceId"`
	BackupType  string                 `json:"backupType"`
	Status      string                 `json:"status"` // PENDING, RUNNING, COMPLETED, FAILED
	Size        int64                  `json:"size,omitempty"`
	Location    string                 `json:"location,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	CompletedAt *time.Time             `json:"completedAt,omitempty"`
	ExpiresAt   *time.Time             `json:"expiresAt,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ResourceHistoryEntry represents a history entry for a resource
type ResourceHistoryEntry struct {
	EntryID       string                 `json:"entryId"`
	ResourceID    string                 `json:"resourceId"`
	EventType     string                 `json:"eventType"` // CREATED, UPDATED, DELETED, STATE_CHANGED
	EventData     *runtime.RawExtension  `json:"eventData,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
	UserID        string                 `json:"userId,omitempty"`
	Description   string                 `json:"description,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// RestoreResourceRequest represents a request to restore a resource from backup
type RestoreResourceRequest struct {
	BackupID         string                 `json:"backupId"`
	ResourceName     string                 `json:"resourceName,omitempty"`
	ResourcePoolID   string                 `json:"resourcePoolId,omitempty"`
	RestoreType      string                 `json:"restoreType"` // FULL, PARTIAL
	RestoreLocation  string                 `json:"restoreLocation,omitempty"`
	OverwriteExisting bool                   `json:"overwriteExisting"`
	Options          map[string]interface{} `json:"options,omitempty"`
}

// ResourceConfigurationUpdate represents an update to a resource's configuration
type ResourceConfigurationUpdate struct {
	UpdateType    string                `json:"updateType"` // MERGE, REPLACE, PATCH
	Configuration *runtime.RawExtension `json:"configuration"`
	ValidateOnly  bool                  `json:"validateOnly"`
	Options       map[string]interface{} `json:"options,omitempty"`
}

// ResourceEventFilter represents filters for querying resource events
type ResourceEventFilter struct {
	ResourceIDs []string   `json:"resourceIds,omitempty"`
	EventTypes  []string   `json:"eventTypes,omitempty"`
	StartTime   *time.Time `json:"startTime,omitempty"`
	EndTime     *time.Time `json:"endTime,omitempty"`
	UserIDs     []string   `json:"userIds,omitempty"`
	Severity    []string   `json:"severity,omitempty"`
	Limit       int        `json:"limit,omitempty"`
	Offset      int        `json:"offset,omitempty"`
}

// UpdateSubscriptionRequest represents a request to update a subscription
type UpdateSubscriptionRequest struct {
	Callback    *string                `json:"callback,omitempty"`
	Filter      map[string]interface{} `json:"filter,omitempty"`
	EventTypes  []string               `json:"eventTypes,omitempty"`
	ExpiresAt   *time.Time             `json:"expiresAt,omitempty"`
	RetryPolicy *RetryPolicy           `json:"retryPolicy,omitempty"`
}

// Operation and workflow types

// OperationRequest represents a generic operation request
type OperationRequest struct {
	OperationType string                 `json:"operationType"`
	Parameters    map[string]interface{} `json:"parameters,omitempty"`
	Context       *OperationContext      `json:"context,omitempty"`
	Options       *OperationOptions      `json:"options,omitempty"`
}

// OperationContext represents the context for an operation
type OperationContext struct {
	UserID        string            `json:"userId,omitempty"`
	SessionID     string            `json:"sessionId,omitempty"`
	TraceID       string            `json:"traceId,omitempty"`
	CorrelationID string            `json:"correlationId,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// OperationOptions represents options for an operation
type OperationOptions struct {
	Async           bool          `json:"async"`
	Timeout         time.Duration `json:"timeout,omitempty"`
	RetryPolicy     *RetryPolicy  `json:"retryPolicy,omitempty"`
	ProgressCallback string        `json:"progressCallback,omitempty"`
	ValidateOnly    bool          `json:"validateOnly"`
}

// OperationProgress represents the progress of an operation
type OperationProgress struct {
	TotalSteps             int32         `json:"totalSteps"`
	CompletedSteps         int32         `json:"completedSteps"`
	CurrentStep            string        `json:"currentStep,omitempty"`
	PercentComplete        float64       `json:"percentComplete"`
	EstimatedTimeRemaining time.Duration `json:"estimatedTimeRemaining,omitempty"`
}

// Interface types for components

// Alerting defines the interface for alerting
type Alerting interface {
	CreateAlert(ctx context.Context, alert *Alert) error
	GetActiveAlerts(ctx context.Context) ([]*Alert, error)
	ResolveAlert(ctx context.Context, alertID string) error
}

// ResourceOperations defines the interface for resource operations
type ResourceOperations interface {
	ProvisionResource(ctx context.Context, req *ProvisionResourceRequest) (*o2models.Resource, error)
	ScaleResource(ctx context.Context, resourceID string, req *ScaleResourceRequest) error
	MigrateResource(ctx context.Context, resourceID string, req *MigrateResourceRequest) error
	TerminateResource(ctx context.Context, resourceID string) error
}

// Discovery and inventory support types

// DiscoveryRule represents a rule for resource discovery
type DiscoveryRule struct {
	RuleID      string                 `json:"ruleId"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // NETWORK_SCAN, API_POLL, EVENT_DRIVEN
	Enabled     bool                   `json:"enabled"`
	Schedule    string                 `json:"schedule,omitempty"` // cron expression
	Filter      *DiscoveryFilter       `json:"filter,omitempty"`
	Actions     []string               `json:"actions,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// DiscoveryFilter represents filters for discovery operations
type DiscoveryFilter struct {
	ResourceTypes []string               `json:"resourceTypes,omitempty"`
	Providers     []string               `json:"providers,omitempty"`
	Locations     []string               `json:"locations,omitempty"`
	Tags          map[string]string      `json:"tags,omitempty"`
	Properties    map[string]interface{} `json:"properties,omitempty"`
}

// Auto-scaling and lifecycle management types

// AutoScalePolicy represents an auto-scaling policy
type AutoScalePolicy struct {
	PolicyID        string                `json:"policyId"`
	ResourceID      string                `json:"resourceId"`
	Enabled         bool                  `json:"enabled"`
	MinReplicas     int32                 `json:"minReplicas"`
	MaxReplicas     int32                 `json:"maxReplicas"`
	TargetCPU       *int32                `json:"targetCpu,omitempty"`
	TargetMemory    *int32                `json:"targetMemory,omitempty"`
	ScaleUpPolicy   *AutoScaleRuleSet     `json:"scaleUpPolicy,omitempty"`
	ScaleDownPolicy *AutoScaleRuleSet     `json:"scaleDownPolicy,omitempty"`
	Metrics         []AutoScaleMetric     `json:"metrics,omitempty"`
	Behaviors       *AutoScaleBehaviors   `json:"behaviors,omitempty"`
}

// AutoScaleRuleSet represents a set of auto-scaling rules
type AutoScaleRuleSet struct {
	Rules                   []AutoScaleRule `json:"rules"`
	SelectPolicy            string          `json:"selectPolicy,omitempty"` // Max, Min, Average
	StabilizationWindowSecs int32           `json:"stabilizationWindowSeconds,omitempty"`
}

// AutoScaleRule represents an individual auto-scaling rule
type AutoScaleRule struct {
	Type          string `json:"type"`  // Pods, Percent
	Value         int32  `json:"value"`
	PeriodSeconds int32  `json:"periodSeconds"`
}

// AutoScaleMetric represents a metric for auto-scaling
type AutoScaleMetric struct {
	Type     string                 `json:"type"` // Resource, Pods, Object, External
	Name     string                 `json:"name"`
	Target   AutoScaleMetricTarget  `json:"target"`
	Selector map[string]string      `json:"selector,omitempty"`
}

// AutoScaleMetricTarget represents a target for auto-scaling metrics
type AutoScaleMetricTarget struct {
	Type               string `json:"type"` // Utilization, Value, AverageValue
	AverageUtilization *int32 `json:"averageUtilization,omitempty"`
	AverageValue       string `json:"averageValue,omitempty"`
	Value              string `json:"value,omitempty"`
}

// AutoScaleBehaviors represents scaling behaviors
type AutoScaleBehaviors struct {
	ScaleUp   *AutoScaleBehavior `json:"scaleUp,omitempty"`
	ScaleDown *AutoScaleBehavior `json:"scaleDown,omitempty"`
}

// AutoScaleBehavior represents scaling behavior
type AutoScaleBehavior struct {
	StabilizationWindowSeconds *int32           `json:"stabilizationWindowSeconds,omitempty"`
	SelectPolicy               string           `json:"selectPolicy,omitempty"`
	Policies                   []AutoScaleRule  `json:"policies,omitempty"`
}

// Lifecycle management types

// LifecyclePolicy represents a lifecycle policy for resources
type LifecyclePolicy struct {
	PolicyID       string                      `json:"policyId"`
	Name           string                      `json:"name"`
	ResourceTypes  []string                    `json:"resourceTypes"`
	Enabled        bool                        `json:"enabled"`
	Rules          []LifecycleRule             `json:"rules"`
	DefaultActions *LifecycleActionSet         `json:"defaultActions,omitempty"`
	Metadata       map[string]interface{}      `json:"metadata,omitempty"`
}

// LifecycleRule represents a lifecycle rule
type LifecycleRule struct {
	RuleID      string                 `json:"ruleId"`
	Name        string                 `json:"name"`
	Condition   string                 `json:"condition"`   // expression to evaluate
	Action      LifecycleAction        `json:"action"`
	Delay       time.Duration          `json:"delay,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// LifecycleAction represents a lifecycle action
type LifecycleAction struct {
	Type        string                 `json:"type"` // SCALE, MIGRATE, BACKUP, TERMINATE, NOTIFY
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	OnFailure   string                 `json:"onFailure,omitempty"` // CONTINUE, ABORT, RETRY
	RetryPolicy *RetryPolicy           `json:"retryPolicy,omitempty"`
}

// LifecycleActionSet represents a set of lifecycle actions
type LifecycleActionSet struct {
	PreTerminate  []LifecycleAction `json:"preTerminate,omitempty"`
	PostTerminate []LifecycleAction `json:"postTerminate,omitempty"`
	OnFailure     []LifecycleAction `json:"onFailure,omitempty"`
	OnSuccess     []LifecycleAction `json:"onSuccess,omitempty"`
}

// Validation and compliance types

// ValidationRule represents a validation rule
type ValidationRule struct {
	RuleID     string                 `json:"ruleId"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"` // SCHEMA, CONSTRAINT, POLICY, CUSTOM
	Expression string                 `json:"expression"`
	Message    string                 `json:"message,omitempty"`
	Severity   string                 `json:"severity"` // ERROR, WARNING, INFO
	Enabled    bool                   `json:"enabled"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

// ValidationInfo represents validation information
type ValidationInfo struct {
	RuleID  string `json:"ruleId"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

// Performance optimization types

// PerformanceProfile represents a performance profile for resources
type PerformanceProfile struct {
	ProfileID   string                     `json:"profileId"`
	Name        string                     `json:"name"`
	Description string                     `json:"description,omitempty"`
	Type        string                     `json:"type"` // CPU_OPTIMIZED, MEMORY_OPTIMIZED, BALANCED, CUSTOM
	Settings    PerformanceSettings        `json:"settings"`
	Constraints []PerformanceConstraint    `json:"constraints,omitempty"`
	Metadata    map[string]interface{}     `json:"metadata,omitempty"`
}

// PerformanceSettings represents performance settings
type PerformanceSettings struct {
	CPU              PerformanceSetting `json:"cpu,omitempty"`
	Memory           PerformanceSetting `json:"memory,omitempty"`
	Storage          PerformanceSetting `json:"storage,omitempty"`
	Network          PerformanceSetting `json:"network,omitempty"`
	CustomSettings   map[string]interface{} `json:"customSettings,omitempty"`
}

// PerformanceSetting represents an individual performance setting
type PerformanceSetting struct {
	MinValue     *float64               `json:"minValue,omitempty"`
	MaxValue     *float64               `json:"maxValue,omitempty"`
	DefaultValue *float64               `json:"defaultValue,omitempty"`
	Unit         string                 `json:"unit,omitempty"`
	Options      map[string]interface{} `json:"options,omitempty"`
}

// PerformanceConstraint represents a performance constraint
type PerformanceConstraint struct {
	Type        string  `json:"type"` // MIN, MAX, RANGE
	Metric      string  `json:"metric"`
	Value       float64 `json:"value"`
	Unit        string  `json:"unit,omitempty"`
	Description string  `json:"description,omitempty"`
}

// Cache and optimization types

// CacheConfig represents cache configuration
type CacheConfig struct {
	Enabled       bool          `json:"enabled"`
	TTL           time.Duration `json:"ttl"`
	MaxSize       int           `json:"maxSize"`
	EvictionPolicy string        `json:"evictionPolicy"` // LRU, LFU, FIFO
	Compression   bool          `json:"compression"`
	Persistence   bool          `json:"persistence"`
}

// LoadBalancerConfig represents load balancer configuration
type LoadBalancerConfig struct {
	Algorithm      string                 `json:"algorithm"` // ROUND_ROBIN, LEAST_CONNECTIONS, WEIGHTED, IP_HASH
	HealthCheck    LoadBalancerHealthCheck `json:"healthCheck"`
	SessionAffinity bool                   `json:"sessionAffinity"`
	Targets        []LoadBalancerTarget   `json:"targets"`
}

// LoadBalancerHealthCheck represents health check configuration for load balancer
type LoadBalancerHealthCheck struct {
	Enabled        bool          `json:"enabled"`
	Path           string        `json:"path,omitempty"`
	Port           int32         `json:"port,omitempty"`
	Protocol       string        `json:"protocol"` // HTTP, HTTPS, TCP
	Interval       time.Duration `json:"interval"`
	Timeout        time.Duration `json:"timeout"`
	HealthyThreshold   int       `json:"healthyThreshold"`
	UnhealthyThreshold int       `json:"unhealthyThreshold"`
}

// LoadBalancerTarget represents a target for load balancer
type LoadBalancerTarget struct {
	Address string `json:"address"`
	Port    int32  `json:"port"`
	Weight  int32  `json:"weight,omitempty"`
	Enabled bool   `json:"enabled"`
}

// Constants for common values
const (
	// Resource states
	ResourceStateCreating  = "CREATING"
	ResourceStateActive    = "ACTIVE"
	ResourceStateUpdating  = "UPDATING"
	ResourceStateFailed    = "FAILED"
	ResourceStateDeleting  = "DELETING"
	ResourceStateDeleted   = "DELETED"

	// Health states
	HealthStateHealthy   = "HEALTHY"
	HealthStateDegraded  = "DEGRADED"
	HealthStateUnhealthy = "UNHEALTHY"
	HealthStateUnknown   = "UNKNOWN"

	// Operation statuses
	OperationStatusPending   = "PENDING"
	OperationStatusRunning   = "RUNNING"
	OperationStatusCompleted = "COMPLETED"
	OperationStatusFailed    = "FAILED"
	OperationStatusCancelled = "CANCELLED"

	// Alert severities
	AlertSeverityCritical = "CRITICAL"
	AlertSeverityHigh     = "HIGH"
	AlertSeverityMedium   = "MEDIUM"
	AlertSeverityLow      = "LOW"
	AlertSeverityInfo     = "INFO"

	// Scale types
	ScaleTypeUp   = "UP"
	ScaleTypeDown = "DOWN"
	ScaleTypeOut  = "OUT"
	ScaleTypeIn   = "IN"

	// Migration types
	MigrationTypeLive = "LIVE"
	MigrationTypeCold = "COLD"
	MigrationTypeHot  = "HOT"

	// Backup types
	BackupTypeFull         = "FULL"
	BackupTypeIncremental  = "INCREMENTAL"
	BackupTypeDifferential = "DIFFERENTIAL"
)