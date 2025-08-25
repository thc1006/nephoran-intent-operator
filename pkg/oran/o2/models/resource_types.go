package models

import (
	"time"

	"k8s.io/apimachinery/pkg/runtime"
)

// Resource Type Management Models following O-RAN.WG6.O2ims-Interface-v01.01

// ResourceType represents a type of infrastructure resource following O2 IMS specification
type ResourceType struct {
	ResourceTypeID  string                 `json:"resourceTypeId"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description,omitempty"`
	Vendor          string                 `json:"vendor,omitempty"`
	Model           string                 `json:"model,omitempty"`
	Version         string                 `json:"version,omitempty"`
	AlarmDictionary *AlarmDictionary       `json:"alarmDictionary,omitempty"`
	Extensions      map[string]interface{} `json:"extensions,omitempty"`

	// Resource specifications (compatible with both old and new formats)
	Specifications   *ResourceTypeSpec `json:"specifications,omitempty"`
	SupportedActions []string          `json:"supportedActions,omitempty"`

	// Legacy fields for backward compatibility
	Category      string `json:"category,omitempty"`      // COMPUTE, STORAGE, NETWORK, ACCELERATOR
	ResourceClass string `json:"resourceClass,omitempty"` // PHYSICAL, VIRTUAL, LOGICAL
	ResourceKind  string `json:"resourceKind,omitempty"`  // SERVER, SWITCH, STORAGE_ARRAY

	// Configuration and schema
	YANGModel    *YANGModelReference   `json:"yangModel,omitempty"`
	ConfigSchema *runtime.RawExtension `json:"configSchema,omitempty"`
	StateSchema  *runtime.RawExtension `json:"stateSchema,omitempty"`

	// Capabilities and features
	SupportedOperations []string             `json:"supportedOperations,omitempty"`
	Capabilities        []ResourceCapability `json:"capabilities,omitempty"`
	Features            []ResourceFeature    `json:"features,omitempty"`

	// Resource characteristics
	ResourceLimits     *ResourceLimits     `json:"resourceLimits,omitempty"`
	PerformanceProfile *PerformanceProfile `json:"performanceProfile,omitempty"`

	// Lifecycle and management
	LifecycleStates   []string `json:"lifecycleStates,omitempty"`
	ManagementMethods []string `json:"managementMethods,omitempty"`

	// Compatibility and requirements
	Dependencies  []*TypeDependency    `json:"dependencies,omitempty"`
	Compatibility []*CompatibilityRule `json:"compatibility,omitempty"`
	Requirements  []*TypeRequirement   `json:"requirements,omitempty"`

	// Extensions and metadata
	VendorExtensions map[string]interface{} `json:"vendorExtensions,omitempty"`
	Tags             map[string]string      `json:"tags,omitempty"`
	Labels           map[string]string      `json:"labels,omitempty"`

	// Lifecycle information
	Status    string    `json:"status"` // ACTIVE, DEPRECATED, OBSOLETE
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	CreatedBy string    `json:"createdBy,omitempty"`
	UpdatedBy string    `json:"updatedBy,omitempty"`
}

// ResourceTypeSpec defines the specifications for a resource type
type ResourceTypeSpec struct {
	Category         string                 `json:"category"` // COMPUTE, STORAGE, NETWORK, ACCELERATOR
	MinResources     map[string]string      `json:"minResources,omitempty"`
	MaxResources     map[string]string      `json:"maxResources,omitempty"`
	DefaultResources map[string]string      `json:"defaultResources,omitempty"`
	Properties       map[string]interface{} `json:"properties,omitempty"`
	Constraints      []string               `json:"constraints,omitempty"`
}

// AlarmDictionary defines alarm information for a resource type
type AlarmDictionary struct {
	ID               string             `json:"id"`
	Name             string             `json:"name"`
	AlarmDefinitions []*AlarmDefinition `json:"alarmDefinitions"`
}

// AlarmDefinition defines a specific alarm type
type AlarmDefinition struct {
	AlarmDefinitionID     string            `json:"alarmDefinitionId"`
	AlarmName             string            `json:"alarmName"`
	AlarmDescription      string            `json:"alarmDescription,omitempty"`
	ProposedRepairActions []string          `json:"proposedRepairActions,omitempty"`
	AlarmAdditionalFields map[string]string `json:"alarmAdditionalFields,omitempty"`
	AlarmLastChange       string            `json:"alarmLastChange,omitempty"`
}

// YANGModelReference represents a reference to a YANG model
type YANGModelReference struct {
	ModelName      string `json:"modelName"`
	Version        string `json:"version"`
	Namespace      string `json:"namespace,omitempty"`
	Module         string `json:"module,omitempty"`
	Revision       string `json:"revision,omitempty"`
	SchemaLocation string `json:"schemaLocation,omitempty"`
}

// ResourceCapability represents a capability of a resource type
type ResourceCapability struct {
	Name         string                  `json:"name"`
	Type         string                  `json:"type"` // FUNCTIONAL, PERFORMANCE, OPERATIONAL
	Description  string                  `json:"description,omitempty"`
	Parameters   map[string]interface{}  `json:"parameters,omitempty"`
	Constraints  []*CapabilityConstraint `json:"constraints,omitempty"`
	SupportLevel string                  `json:"supportLevel,omitempty"` // MANDATORY, OPTIONAL, CONDITIONAL
}

// ResourceFeature represents a feature supported by a resource type
type ResourceFeature struct {
	Name          string                 `json:"name"`
	Description   string                 `json:"description,omitempty"`
	FeatureType   string                 `json:"featureType"`  // STANDARD, VENDOR_SPECIFIC, EXPERIMENTAL
	SupportLevel  string                 `json:"supportLevel"` // FULL, PARTIAL, EXPERIMENTAL
	Dependencies  []string               `json:"dependencies,omitempty"`
	Constraints   []*FeatureConstraint   `json:"constraints,omitempty"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
}

// ResourceLimits defines limits and constraints for resources of this type
type ResourceLimits struct {
	// Quantity limits
	MinInstances     *int `json:"minInstances,omitempty"`
	MaxInstances     *int `json:"maxInstances,omitempty"`
	DefaultInstances *int `json:"defaultInstances,omitempty"`

	// Resource limits
	CPULimits     *ResourceLimit `json:"cpuLimits,omitempty"`
	MemoryLimits  *ResourceLimit `json:"memoryLimits,omitempty"`
	StorageLimits *ResourceLimit `json:"storageLimits,omitempty"`
	NetworkLimits *NetworkLimits `json:"networkLimits,omitempty"`

	// Custom limits
	CustomLimits map[string]*ResourceLimit `json:"customLimits,omitempty"`
}

// ResourceLimit defines limits for a specific resource metric
type ResourceLimit struct {
	MinValue     string   `json:"minValue,omitempty"`
	MaxValue     string   `json:"maxValue,omitempty"`
	DefaultValue string   `json:"defaultValue,omitempty"`
	Unit         string   `json:"unit"`
	Constraints  []string `json:"constraints,omitempty"`
}

// NetworkLimits defines network-specific limits
type NetworkLimits struct {
	MaxBandwidth       string   `json:"maxBandwidth,omitempty"`
	MaxConnections     *int     `json:"maxConnections,omitempty"`
	MaxPorts           *int     `json:"maxPorts,omitempty"`
	SupportedProtocols []string `json:"supportedProtocols,omitempty"`
}

// PerformanceProfile defines expected performance characteristics
type PerformanceProfile struct {
	// Compute performance
	CPUPerformance    *PerformanceMetric `json:"cpuPerformance,omitempty"`
	MemoryPerformance *PerformanceMetric `json:"memoryPerformance,omitempty"`

	// Storage performance
	StorageIOPS       *PerformanceMetric `json:"storageIops,omitempty"`
	StorageThroughput *PerformanceMetric `json:"storageThroughput,omitempty"`
	StorageLatency    *PerformanceMetric `json:"storageLatency,omitempty"`

	// Network performance
	NetworkThroughput *PerformanceMetric `json:"networkThroughput,omitempty"`
	NetworkLatency    *PerformanceMetric `json:"networkLatency,omitempty"`
	NetworkPacketRate *PerformanceMetric `json:"networkPacketRate,omitempty"`

	// Custom performance metrics
	CustomMetrics map[string]*PerformanceMetric `json:"customMetrics,omitempty"`
}

// PerformanceMetric defines a performance metric with expected values
type PerformanceMetric struct {
	MetricName   string            `json:"metricName"`
	Unit         string            `json:"unit"`
	TypicalValue string            `json:"typicalValue,omitempty"`
	MinValue     string            `json:"minValue,omitempty"`
	MaxValue     string            `json:"maxValue,omitempty"`
	Percentiles  map[string]string `json:"percentiles,omitempty"` // P50, P95, P99
	Conditions   []string          `json:"conditions,omitempty"`
}

// TypeDependency represents a dependency between resource types
type TypeDependency struct {
	DependencyType string                  `json:"dependencyType"` // REQUIRES, CONFLICTS, RECOMMENDS
	ResourceTypeID string                  `json:"resourceTypeId"`
	Name           string                  `json:"name,omitempty"`
	Version        string                  `json:"version,omitempty"`
	Cardinality    string                  `json:"cardinality,omitempty"` // ONE, MANY, ZERO_OR_ONE, ZERO_OR_MANY
	Description    string                  `json:"description,omitempty"`
	Constraints    []*DependencyConstraint `json:"constraints,omitempty"`
}

// CompatibilityRule defines compatibility rules for resource types
type CompatibilityRule struct {
	RuleType      string `json:"ruleType"` // COMPATIBLE, INCOMPATIBLE, CONDITIONAL
	TargetTypeID  string `json:"targetTypeId,omitempty"`
	TargetVersion string `json:"targetVersion,omitempty"`
	Condition     string `json:"condition,omitempty"`
	Description   string `json:"description,omitempty"`
	Severity      string `json:"severity,omitempty"` // ERROR, WARNING, INFO
}

// TypeRequirement defines requirements for deploying resources of this type
type TypeRequirement struct {
	RequirementType string                   `json:"requirementType"` // HARDWARE, SOFTWARE, NETWORK, SECURITY
	Name            string                   `json:"name"`
	Description     string                   `json:"description,omitempty"`
	Mandatory       bool                     `json:"mandatory"`
	Values          []string                 `json:"values,omitempty"`
	Constraints     []*RequirementConstraint `json:"constraints,omitempty"`
	Validation      *RequirementValidation   `json:"validation,omitempty"`
}

// CapabilityConstraint defines constraints for resource capabilities
type CapabilityConstraint struct {
	ConstraintType string      `json:"constraintType"` // RANGE, ENUM, PATTERN, CUSTOM
	Parameter      string      `json:"parameter"`
	Values         []string    `json:"values,omitempty"`
	MinValue       interface{} `json:"minValue,omitempty"`
	MaxValue       interface{} `json:"maxValue,omitempty"`
	Pattern        string      `json:"pattern,omitempty"`
	Expression     string      `json:"expression,omitempty"`
	ErrorMessage   string      `json:"errorMessage,omitempty"`
}

// FeatureConstraint defines constraints for resource features
type FeatureConstraint struct {
	ConstraintType string      `json:"constraintType"`
	Condition      string      `json:"condition"`
	Value          interface{} `json:"value,omitempty"`
	Message        string      `json:"message,omitempty"`
}

// DependencyConstraint defines constraints for type dependencies
type DependencyConstraint struct {
	ConstraintType string      `json:"constraintType"`
	Property       string      `json:"property"`
	Operator       string      `json:"operator"` // EQ, NE, GT, LT, GE, LE, IN, NOT_IN
	Value          interface{} `json:"value"`
	Description    string      `json:"description,omitempty"`
}

// RequirementConstraint defines constraints for type requirements
type RequirementConstraint struct {
	ConstraintType string      `json:"constraintType"`
	Field          string      `json:"field"`
	Operator       string      `json:"operator"`
	Value          interface{} `json:"value"`
	Message        string      `json:"message,omitempty"`
}

// RequirementValidation defines validation rules for requirements
type RequirementValidation struct {
	ValidationType   string                 `json:"validationType"` // SCRIPT, API, MANUAL
	ValidationScript string                 `json:"validationScript,omitempty"`
	ValidationAPI    string                 `json:"validationApi,omitempty"`
	Timeout          time.Duration          `json:"timeout,omitempty"`
	RetryPolicy      *ValidationRetryPolicy `json:"retryPolicy,omitempty"`
}

// ValidationRetryPolicy defines retry policy for requirement validation
type ValidationRetryPolicy struct {
	MaxRetries       int           `json:"maxRetries"`
	RetryInterval    time.Duration `json:"retryInterval"`
	BackoffFactor    float64       `json:"backoffFactor"`
	MaxRetryInterval time.Duration `json:"maxRetryInterval"`
}

// Resource represents an infrastructure resource instance following O2 IMS specification
type Resource struct {
	ResourceID       string `json:"resourceId"`
	Name             string `json:"name"`
	Description      string `json:"description,omitempty"`
	ResourceTypeID   string `json:"resourceTypeId"`
	ResourcePoolID   string `json:"resourcePoolId"`
	ParentResourceID string `json:"parentResourceId,omitempty"`

	// Resource state and configuration
	Status        *ResourceStatus       `json:"status"`
	Configuration *runtime.RawExtension `json:"configuration,omitempty"`
	State         *runtime.RawExtension `json:"state,omitempty"`

	// Resource allocation and capacity
	AllocatedCapacity *ResourceCapacity `json:"allocatedCapacity,omitempty"`
	UtilizedCapacity  *ResourceCapacity `json:"utilizedCapacity,omitempty"`

	// Relationships and dependencies
	Dependencies  []string                `json:"dependencies,omitempty"`
	Dependents    []string                `json:"dependents,omitempty"`
	Relationships []*ResourceRelationship `json:"relationships,omitempty"`

	// Location and placement
	Location  *ResourceLocation  `json:"location,omitempty"`
	Placement *ResourcePlacement `json:"placement,omitempty"`

	// Monitoring and health
	Health  *ResourceHealthInfo    `json:"health,omitempty"`
	Metrics map[string]interface{} `json:"metrics,omitempty"`
	Alarms  []*ResourceAlarm       `json:"alarms,omitempty"`

	// Lifecycle and management
	LifecycleState string                  `json:"lifecycleState,omitempty"`
	ManagementInfo *ResourceManagementInfo `json:"managementInfo,omitempty"`

	// Extensions and metadata
	VendorExtensions map[string]interface{} `json:"vendorExtensions,omitempty"`
	Extensions       map[string]interface{} `json:"extensions,omitempty"`
	Tags             map[string]string      `json:"tags,omitempty"`
	Labels           map[string]string      `json:"labels,omitempty"`

	// Lifecycle information
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	CreatedBy string    `json:"createdBy,omitempty"`
	UpdatedBy string    `json:"updatedBy,omitempty"`
}

// ResourceStatus represents the current status of a resource
type ResourceStatus struct {
	State               string                 `json:"state"`               // PENDING, ACTIVE, INACTIVE, FAILED, DELETING
	OperationalState    string                 `json:"operationalState"`    // ENABLED, DISABLED
	AdministrativeState string                 `json:"administrativeState"` // LOCKED, UNLOCKED, SHUTTINGDOWN
	UsageState          string                 `json:"usageState"`          // IDLE, ACTIVE, BUSY
	Health              string                 `json:"health"`              // HEALTHY, DEGRADED, UNHEALTHY, UNKNOWN
	LastHealthCheck     time.Time              `json:"lastHealthCheck"`
	ErrorMessage        string                 `json:"errorMessage,omitempty"`
	Conditions          []ResourceCondition    `json:"conditions,omitempty"`
	Metrics             map[string]interface{} `json:"metrics,omitempty"`
}

// ResourceCondition represents a condition of the resource
type ResourceCondition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"` // True, False, Unknown
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
	LastTransitionTime time.Time `json:"lastTransitionTime"`
	LastUpdateTime     time.Time `json:"lastUpdateTime,omitempty"`
}

// ResourceCapacity represents resource capacity information
type ResourceCapacity struct {
	CPU             *ResourceMetric            `json:"cpu,omitempty"`
	Memory          *ResourceMetric            `json:"memory,omitempty"`
	Storage         *ResourceMetric            `json:"storage,omitempty"`
	Network         *ResourceMetric            `json:"network,omitempty"`
	Accelerators    *ResourceMetric            `json:"accelerators,omitempty"`
	CustomResources map[string]*ResourceMetric `json:"customResources,omitempty"`
}

// ResourceMetric represents a resource metric with total, available, and used values
type ResourceMetric struct {
	Total       string  `json:"total"`
	Available   string  `json:"available"`
	Used        string  `json:"used"`
	Unit        string  `json:"unit"`
	Utilization float64 `json:"utilization"`
}

// ResourceRelationship represents a relationship between resources
type ResourceRelationship struct {
	RelationshipType string                 `json:"relationshipType"` // CONTAINS, USES, CONNECTS_TO, DEPENDS_ON
	TargetResourceID string                 `json:"targetResourceId"`
	Properties       map[string]interface{} `json:"properties,omitempty"`
	Description      string                 `json:"description,omitempty"`
}

// ResourceLocation represents the location of a resource
type ResourceLocation struct {
	// Geographic location
	Country    string `json:"country,omitempty"`
	Region     string `json:"region,omitempty"`
	City       string `json:"city,omitempty"`
	DataCenter string `json:"dataCenter,omitempty"`
	Zone       string `json:"zone,omitempty"`

	// Coordinates
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`

	// Physical location
	Building string `json:"building,omitempty"`
	Floor    string `json:"floor,omitempty"`
	Room     string `json:"room,omitempty"`
	Rack     string `json:"rack,omitempty"`
	Position string `json:"position,omitempty"`

	// Logical location
	Cluster   string `json:"cluster,omitempty"`
	Namespace string `json:"namespace,omitempty"`

	// Additional properties
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// ResourcePlacement represents placement constraints and preferences
type ResourcePlacement struct {
	// Affinity rules
	NodeAffinity     *NodeAffinity     `json:"nodeAffinity,omitempty"`
	ResourceAffinity *ResourceAffinity `json:"resourceAffinity,omitempty"`

	// Anti-affinity rules
	AntiAffinity *AntiAffinityRules `json:"antiAffinity,omitempty"`

	// Placement constraints
	Constraints []*PlacementConstraint `json:"constraints,omitempty"`
	Preferences []*PlacementPreference `json:"preferences,omitempty"`

	// Topology
	TopologyKey    string                    `json:"topologyKey,omitempty"`
	TopologySpread *TopologySpreadConstraint `json:"topologySpread,omitempty"`
}

// NodeAffinity represents node affinity rules (from helper_types.go but included here for completeness)
type NodeAffinity struct {
	RequiredDuringSchedulingIgnoredDuringExecution  []*NodeSelectorTerm        `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`
	PreferredDuringSchedulingIgnoredDuringExecution []*PreferredSchedulingTerm `json:"preferredDuringSchedulingIgnoredDuringExecution,omitempty"`
}

// NodeSelectorTerm represents a node selector term
type NodeSelectorTerm struct {
	MatchExpressions []*NodeSelectorRequirement `json:"matchExpressions,omitempty"`
	MatchFields      []*NodeSelectorRequirement `json:"matchFields,omitempty"`
}

// NodeSelectorRequirement represents a node selector requirement
type NodeSelectorRequirement struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"` // In, NotIn, Exists, DoesNotExist, Gt, Lt
	Values   []string `json:"values,omitempty"`
}

// PreferredSchedulingTerm represents a preferred scheduling term
type PreferredSchedulingTerm struct {
	Weight     int32             `json:"weight"`
	Preference *NodeSelectorTerm `json:"preference"`
}

// ResourceAffinity represents affinity rules based on resource properties
type ResourceAffinity struct {
	RequiredRules  []*ResourceAffinityRule `json:"requiredRules,omitempty"`
	PreferredRules []*ResourceAffinityRule `json:"preferredRules,omitempty"`
}

// ResourceAffinityRule defines a resource affinity rule
type ResourceAffinityRule struct {
	Weight      int32    `json:"weight,omitempty"`
	Property    string   `json:"property"`
	Operator    string   `json:"operator"`
	Values      []string `json:"values,omitempty"`
	TopologyKey string   `json:"topologyKey,omitempty"`
}

// AntiAffinityRules represents anti-affinity rules
type AntiAffinityRules struct {
	ResourceTypes []string                  `json:"resourceTypes,omitempty"`
	ResourceIDs   []string                  `json:"resourceIds,omitempty"`
	TopologyKeys  []string                  `json:"topologyKeys,omitempty"`
	Constraints   []*AntiAffinityConstraint `json:"constraints,omitempty"`
}

// AntiAffinityConstraint defines an anti-affinity constraint
type AntiAffinityConstraint struct {
	Type     string   `json:"type"` // HARD, SOFT
	Property string   `json:"property"`
	Operator string   `json:"operator"`
	Values   []string `json:"values,omitempty"`
	Scope    string   `json:"scope"` // NODE, RACK, ZONE, REGION
}

// PlacementConstraint defines a placement constraint
type PlacementConstraint struct {
	ConstraintType string      `json:"constraintType"` // LOCATION, CAPACITY, NETWORK, SECURITY
	Property       string      `json:"property"`
	Operator       string      `json:"operator"`
	Value          interface{} `json:"value"`
	Required       bool        `json:"required"`
	Description    string      `json:"description,omitempty"`
}

// PlacementPreference defines a placement preference
type PlacementPreference struct {
	PreferenceType string      `json:"preferenceType"`
	Property       string      `json:"property"`
	PreferredValue interface{} `json:"preferredValue"`
	Weight         int32       `json:"weight"`
	Description    string      `json:"description,omitempty"`
}

// TopologySpreadConstraint defines topology spread constraints
type TopologySpreadConstraint struct {
	MaxSkew           int32          `json:"maxSkew"`
	TopologyKey       string         `json:"topologyKey"`
	WhenUnsatisfiable string         `json:"whenUnsatisfiable"` // DO_NOT_SCHEDULE, SCHEDULE_ANYWAY
	LabelSelector     *LabelSelector `json:"labelSelector,omitempty"`
}

// LabelSelector represents a label selector
type LabelSelector struct {
	MatchLabels      map[string]string           `json:"matchLabels,omitempty"`
	MatchExpressions []*LabelSelectorRequirement `json:"matchExpressions,omitempty"`
}

// LabelSelectorRequirement represents a label selector requirement
type LabelSelectorRequirement struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"` // In, NotIn, Exists, DoesNotExist
	Values   []string `json:"values,omitempty"`
}

// ResourceHealthInfo represents health information for a resource
type ResourceHealthInfo struct {
	OverallHealth string                `json:"overallHealth"` // HEALTHY, DEGRADED, UNHEALTHY, UNKNOWN
	HealthScore   float64               `json:"healthScore,omitempty"`
	LastCheckTime time.Time             `json:"lastCheckTime"`
	HealthChecks  []*HealthCheckInfo    `json:"healthChecks,omitempty"`
	HealthHistory []*HealthHistoryEntry `json:"healthHistory,omitempty"`
}

// HealthCheckInfo represents information about a specific health check
type HealthCheckInfo struct {
	CheckName string                 `json:"checkName"`
	CheckType string                 `json:"checkType"` // LIVENESS, READINESS, STARTUP
	Status    string                 `json:"status"`
	Message   string                 `json:"message,omitempty"`
	LastCheck time.Time              `json:"lastCheck"`
	Duration  time.Duration          `json:"duration"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// HealthHistoryEntry represents a health history entry
type HealthHistoryEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Health      string                 `json:"health"`
	HealthScore float64                `json:"healthScore"`
	Reason      string                 `json:"reason,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// ResourceAlarm represents an alarm associated with a resource
type ResourceAlarm struct {
	AlarmID   string                 `json:"alarmId"`
	AlarmType string                 `json:"alarmType"`
	Severity  string                 `json:"severity"`
	Status    string                 `json:"status"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	RaisedAt  time.Time              `json:"raisedAt"`
	ClearedAt *time.Time             `json:"clearedAt,omitempty"`
}

// ResourceManagementInfo represents management information for a resource
type ResourceManagementInfo struct {
	ManagedBy          string                 `json:"managedBy,omitempty"`
	ManagementAPI      string                 `json:"managementApi,omitempty"`
	ManagementEndpoint string                 `json:"managementEndpoint,omitempty"`
	Credentials        map[string]interface{} `json:"credentials,omitempty"`
	ManagementState    string                 `json:"managementState,omitempty"`
	LastManaged        time.Time              `json:"lastManaged,omitempty"`
}

// Filter types for resource queries

// ResourceTypeFilter defines filters for resource type queries
type ResourceTypeFilter struct {
	Names           []string          `json:"names,omitempty"`
	Vendors         []string          `json:"vendors,omitempty"`
	Models          []string          `json:"models,omitempty"`    // Added missing field
	Versions        []string          `json:"versions,omitempty"`
	Categories      []string          `json:"categories,omitempty"`
	ResourceClasses []string          `json:"resourceClasses,omitempty"`
	ResourceKinds   []string          `json:"resourceKinds,omitempty"`
	Status          []string          `json:"status,omitempty"`
	Capabilities    []string          `json:"capabilities,omitempty"`
	Features        []string          `json:"features,omitempty"`
	Tags            map[string]string `json:"tags,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
	CreatedAfter    *time.Time        `json:"createdAfter,omitempty"`
	CreatedBefore   *time.Time        `json:"createdBefore,omitempty"`
	Limit           int               `json:"limit,omitempty"`
	Offset          int               `json:"offset,omitempty"`
	SortBy          string            `json:"sortBy,omitempty"`
	SortOrder       string            `json:"sortOrder,omitempty"`
}

// ResourceFilter defines filters for resource queries
type ResourceFilter struct {
	Names             []string          `json:"names,omitempty"`
	ResourceTypeIDs   []string          `json:"resourceTypeIds,omitempty"`
	ResourcePoolIDs   []string          `json:"resourcePoolIds,omitempty"`
	ParentResourceIDs []string          `json:"parentResourceIds,omitempty"`
	LifecycleStates   []string          `json:"lifecycleStates,omitempty"`
	HealthStates      []string          `json:"healthStates,omitempty"`
	Locations         []*LocationFilter `json:"locations,omitempty"`
	Tags              map[string]string `json:"tags,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
	CreatedAfter      *time.Time        `json:"createdAfter,omitempty"`
	CreatedBefore     *time.Time        `json:"createdBefore,omitempty"`
	Limit             int               `json:"limit,omitempty"`
	Offset            int               `json:"offset,omitempty"`
	SortBy            string            `json:"sortBy,omitempty"`
	SortOrder         string            `json:"sortOrder,omitempty"`
}

// LocationFilter defines location-based filtering (from subscription models but included here for completeness)
type LocationFilter struct {
	Regions     []string        `json:"regions,omitempty"`
	Zones       []string        `json:"zones,omitempty"`
	DataCenters []string        `json:"dataCenters,omitempty"`
	Coordinates *GeographicArea `json:"coordinates,omitempty"`
}

// GeographicArea defines a geographic area for filtering (from subscription models)
type GeographicArea struct {
	CenterLatitude  float64      `json:"centerLatitude"`
	CenterLongitude float64      `json:"centerLongitude"`
	Radius          float64      `json:"radius"` // in kilometers
	BoundingBox     *BoundingBox `json:"boundingBox,omitempty"`
}

// BoundingBox defines a rectangular geographic boundary (from subscription models)
type BoundingBox struct {
	NorthLatitude float64 `json:"northLatitude"`
	SouthLatitude float64 `json:"southLatitude"`
	EastLongitude float64 `json:"eastLongitude"`
	WestLongitude float64 `json:"westLongitude"`
}

// Request types for resource management operations

// CreateResourceRequest represents a request to create a resource
type CreateResourceRequest struct {
	Name                 string                 `json:"name"`
	Description          string                 `json:"description,omitempty"`
	ResourceTypeID       string                 `json:"resourceTypeId"`
	ResourcePoolID       string                 `json:"resourcePoolId"`
	ParentResourceID     string                 `json:"parentResourceId,omitempty"`
	Configuration        *runtime.RawExtension  `json:"configuration,omitempty"`
	PlacementConstraints *ResourcePlacement     `json:"placementConstraints,omitempty"`
	Tags                 map[string]string      `json:"tags,omitempty"`
	Labels               map[string]string      `json:"labels,omitempty"`
	Extensions           map[string]interface{} `json:"extensions,omitempty"`
	Metadata             map[string]string      `json:"metadata,omitempty"`
}

// UpdateResourceRequest represents a request to update a resource
type UpdateResourceRequest struct {
	Name          *string                `json:"name,omitempty"`
	Description   *string                `json:"description,omitempty"`
	Configuration *runtime.RawExtension  `json:"configuration,omitempty"`
	Tags          map[string]string      `json:"tags,omitempty"`
	Labels        map[string]string      `json:"labels,omitempty"`
	Extensions    map[string]interface{} `json:"extensions,omitempty"`
	Metadata      map[string]string      `json:"metadata,omitempty"`
}

// Constants for resource types and resources

const (
	// Resource type categories
	ResourceCategoryCompute     = "COMPUTE"
	ResourceCategoryStorage     = "STORAGE"
	ResourceCategoryNetwork     = "NETWORK"
	ResourceCategoryAccelerator = "ACCELERATOR"
	ResourceCategorySecurity    = "SECURITY"
	ResourceCategoryMonitoring  = "MONITORING"

	// Resource classes
	ResourceClassPhysical  = "PHYSICAL"
	ResourceClassVirtual   = "VIRTUAL"
	ResourceClassLogical   = "LOGICAL"
	ResourceClassContainer = "CONTAINER"

	// Resource kinds
	ResourceKindServer       = "SERVER"
	ResourceKindSwitch       = "SWITCH"
	ResourceKindRouter       = "ROUTER"
	ResourceKindStorageArray = "STORAGE_ARRAY"
	ResourceKindLoadBalancer = "LOAD_BALANCER"
	ResourceKindFirewall     = "FIREWALL"
	ResourceKindGPU          = "GPU"
	ResourceKindFPGA         = "FPGA"

	// Resource type status
	ResourceTypeStatusActive     = "ACTIVE"
	ResourceTypeStatusDeprecated = "DEPRECATED"
	ResourceTypeStatusObsolete   = "OBSOLETE"

	// Lifecycle states
	LifecycleStateProvisioning = "PROVISIONING"
	LifecycleStateActive       = "ACTIVE"
	LifecycleStateInactive     = "INACTIVE"
	LifecycleStateUpdating     = "UPDATING"
	LifecycleStateScaling      = "SCALING"
	LifecycleStateMaintenance  = "MAINTENANCE"
	LifecycleStateFailed       = "FAILED"
	LifecycleStateTerminating  = "TERMINATING"
	LifecycleStateTerminated   = "TERMINATED"

	// Resource health states
	ResourceHealthHealthy   = "HEALTHY"
	ResourceHealthDegraded  = "DEGRADED"
	ResourceHealthUnhealthy = "UNHEALTHY"
	ResourceHealthUnknown   = "UNKNOWN"

	// Capability types
	CapabilityTypeFunctional  = "FUNCTIONAL"
	CapabilityTypePerformance = "PERFORMANCE"
	CapabilityTypeOperational = "OPERATIONAL"

	// Support levels
	SupportLevelMandatory    = "MANDATORY"
	SupportLevelOptional     = "OPTIONAL"
	SupportLevelConditional  = "CONDITIONAL"
	SupportLevelFull         = "FULL"
	SupportLevelPartial      = "PARTIAL"
	SupportLevelExperimental = "EXPERIMENTAL"

	// Feature types
	FeatureTypeStandard       = "STANDARD"
	FeatureTypeVendorSpecific = "VENDOR_SPECIFIC"
	FeatureTypeExperimental   = "EXPERIMENTAL"

	// Dependency types
	DependencyTypeRequires   = "REQUIRES"
	DependencyTypeConflicts  = "CONFLICTS"
	DependencyTypeRecommends = "RECOMMENDS"

	// Compatibility rule types
	CompatibilityRuleCompatible   = "COMPATIBLE"
	CompatibilityRuleIncompatible = "INCOMPATIBLE"
	CompatibilityRuleConditional  = "CONDITIONAL"

	// Requirement types
	RequirementTypeHardware = "HARDWARE"
	RequirementTypeSoftware = "SOFTWARE"
	RequirementTypeNetwork  = "NETWORK"
	RequirementTypeSecurity = "SECURITY"
	RequirementTypePolicy   = "POLICY"

	// Relationship types
	RelationshipTypeContains   = "CONTAINS"
	RelationshipTypeUses       = "USES"
	RelationshipTypeConnectsTo = "CONNECTS_TO"
	RelationshipTypeDependsOn  = "DEPENDS_ON"
	RelationshipTypeHostedBy   = "HOSTED_BY"
	RelationshipTypeManages    = "MANAGES"
)