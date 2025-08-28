package models

import (
	"time"
)

// ResourceInstance represents a specific instance of a resource type.
type ResourceInstance struct {
	ResourceInstanceID   string                 `json:"resourceInstanceId"`
	ResourceTypeID       string                 `json:"resourceTypeId"`
	ResourcePoolID       string                 `json:"resourcePoolId"`
	OCloudID             string                 `json:"oCloudId"`
	Name                 string                 `json:"name"`
	Description          string                 `json:"description,omitempty"`
	ParentID             string                 `json:"parentId,omitempty"`
	State                string                 `json:"state"`                // INSTANTIATED, TERMINATED
	OperationalStatus    string                 `json:"operationalStatus"`    // ENABLED, DISABLED
	AdministrativeStatus string                 `json:"administrativeStatus"` // LOCKED, UNLOCKED, SHUTTINGDOWN
	UsageStatus          string                 `json:"usageStatus"`          // IDLE, ACTIVE, BUSY
	Extensions           map[string]interface{} `json:"extensions,omitempty"`

	// Nephoran-specific extensions.
	Provider    string                 `json:"provider,omitempty"`
	Region      string                 `json:"region,omitempty"`
	Zone        string                 `json:"zone,omitempty"`
	Status      *ResourceStatus        `json:"status,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Annotations map[string]string      `json:"annotations,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

// ResourceMetrics represents metrics collected from a resource.
type ResourceMetrics struct {
	ResourceID    string                  `json:"resourceId"`
	Timestamp     time.Time               `json:"timestamp"`
	Metrics       map[string]interface{}  `json:"metrics"`
	CPU           *MetricValue            `json:"cpu,omitempty"`
	Memory        *MetricValue            `json:"memory,omitempty"`
	Storage       *MetricValue            `json:"storage,omitempty"`
	Network       *NetworkMetrics         `json:"network,omitempty"`
	CustomMetrics map[string]*MetricValue `json:"customMetrics,omitempty"`
}

// MetricValue represents a single metric value with metadata.
type MetricValue struct {
	Value       float64           `json:"value"`
	Unit        string            `json:"unit"`
	Timestamp   time.Time         `json:"timestamp"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// NetworkMetrics represents network-specific metrics.
type NetworkMetrics struct {
	BytesIn     *MetricValue `json:"bytesIn,omitempty"`
	BytesOut    *MetricValue `json:"bytesOut,omitempty"`
	PacketsIn   *MetricValue `json:"packetsIn,omitempty"`
	PacketsOut  *MetricValue `json:"packetsOut,omitempty"`
	Connections *MetricValue `json:"connections,omitempty"`
	Latency     *MetricValue `json:"latency,omitempty"`
	Throughput  *MetricValue `json:"throughput,omitempty"`
}

// CreateResourceInstanceRequest represents a request to create a resource instance.
type CreateResourceInstanceRequest struct {
	Name           string                 `json:"name"`
	Description    string                 `json:"description,omitempty"`
	ResourceTypeID string                 `json:"resourceTypeId"`
	ResourcePoolID string                 `json:"resourcePoolId"`
	OCloudID       string                 `json:"oCloudId"`
	ParentID       string                 `json:"parentId,omitempty"`
	Configuration  map[string]interface{} `json:"configuration,omitempty"`
	Extensions     map[string]interface{} `json:"extensions,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Labels         map[string]string      `json:"labels,omitempty"`
	Annotations    map[string]string      `json:"annotations,omitempty"`
}

// UpdateResourceInstanceRequest represents a request to update a resource instance.
type UpdateResourceInstanceRequest struct {
	Name                 *string                `json:"name,omitempty"`
	Description          *string                `json:"description,omitempty"`
	OperationalStatus    *string                `json:"operationalStatus,omitempty"`
	AdministrativeStatus *string                `json:"administrativeStatus,omitempty"`
	Configuration        map[string]interface{} `json:"configuration,omitempty"`
	Extensions           map[string]interface{} `json:"extensions,omitempty"`
	Metadata             map[string]interface{} `json:"metadata,omitempty"`
	Labels               map[string]string      `json:"labels,omitempty"`
	Annotations          map[string]string      `json:"annotations,omitempty"`
}

// ResourceInstanceFilter defines filters for querying resource instances.
type ResourceInstanceFilter struct {
	Names                  []string          `json:"names,omitempty"`
	ResourceTypeIDs        []string          `json:"resourceTypeIds,omitempty"`
	ResourcePoolIDs        []string          `json:"resourcePoolIds,omitempty"`
	OCloudIDs              []string          `json:"oCloudIds,omitempty"`
	ParentIDs              []string          `json:"parentIds,omitempty"`
	States                 []string          `json:"states,omitempty"`
	OperationalStatuses    []string          `json:"operationalStatuses,omitempty"`
	AdministrativeStatuses []string          `json:"administrativeStatuses,omitempty"`
	UsageStatuses          []string          `json:"usageStatuses,omitempty"`
	Providers              []string          `json:"providers,omitempty"`
	Regions                []string          `json:"regions,omitempty"`
	Zones                  []string          `json:"zones,omitempty"`
	Labels                 map[string]string `json:"labels,omitempty"`
	CreatedAfter           *time.Time        `json:"createdAfter,omitempty"`
	CreatedBefore          *time.Time        `json:"createdBefore,omitempty"`
	Limit                  int               `json:"limit,omitempty"`
	Offset                 int               `json:"offset,omitempty"`
	SortBy                 string            `json:"sortBy,omitempty"`
	SortOrder              string            `json:"sortOrder,omitempty"` // ASC, DESC
}

// Constants for resource instance states and statuses.
const (
	// Resource instance states.
	ResourceInstanceStateInstantiated = "INSTANTIATED"
	// ResourceInstanceStateTerminated holds resourceinstancestateterminated value.
	ResourceInstanceStateTerminated = "TERMINATED"

	// Operational statuses.
	ResourceInstanceOperationalStatusEnabled = "ENABLED"
	// ResourceInstanceOperationalStatusDisabled holds resourceinstanceoperationalstatusdisabled value.
	ResourceInstanceOperationalStatusDisabled = "DISABLED"

	// Administrative statuses.
	ResourceInstanceAdministrativeStatusLocked = "LOCKED"
	// ResourceInstanceAdministrativeStatusUnlocked holds resourceinstanceadministrativestatusunlocked value.
	ResourceInstanceAdministrativeStatusUnlocked = "UNLOCKED"
	// ResourceInstanceAdministrativeStatusShuttingDown holds resourceinstanceadministrativestatusshuttingdown value.
	ResourceInstanceAdministrativeStatusShuttingDown = "SHUTTINGDOWN"

	// Usage statuses.
	ResourceInstanceUsageStatusIdle = "IDLE"
	// ResourceInstanceUsageStatusActive holds resourceinstanceusagestatusactive value.
	ResourceInstanceUsageStatusActive = "ACTIVE"
	// ResourceInstanceUsageStatusBusy holds resourceinstanceusagestatusbusy value.
	ResourceInstanceUsageStatusBusy = "BUSY"
)
