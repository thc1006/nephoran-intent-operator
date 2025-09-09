package models

import (
	
	"encoding/json"
"time"

	"k8s.io/apimachinery/pkg/runtime"
)

// Core O2 IMS Data Models following O-RAN.WG6.O2ims-Interface-v01.01 specification.

// SystemInfo represents basic system information for the O2 IMS service.
type SystemInfo struct {
	ID                     string                 `json:"id"`
	Name                   string                 `json:"name"`
	Description            string                 `json:"description"`
	Version                string                 `json:"version"`
	APIVersions            []string               `json:"apiVersions"`
	SupportedResourceTypes []string               `json:"supportedResourceTypes"`
	Extensions             map[string]interface{} `json:"extensions,omitempty"`
	Timestamp              time.Time              `json:"timestamp"`
}

// ResourcePool represents a collection of infrastructure resources.

type ResourcePool struct {
	ResourcePoolID string `json:"resourcePoolId"`

	Name string `json:"name"`

	Description string `json:"description,omitempty"`

	Location string `json:"location,omitempty"`

	OCloudID string `json:"oCloudId"`

	GlobalLocationID string `json:"globalLocationId,omitempty"`

	Extensions map[string]interface{} `json:"extensions,omitempty"`

	// Nephoran-specific extensions.

	Provider string `json:"provider"`

	Region string `json:"region,omitempty"`

	Zone string `json:"zone,omitempty"`

	Capacity *ResourceCapacity `json:"capacity,omitempty"`

	Status *ResourcePoolStatus `json:"status,omitempty"`

	CreatedAt time.Time `json:"createdAt"`

	UpdatedAt time.Time `json:"updatedAt"`

	// Direct state access for test compatibility.

	State string `json:"state,omitempty"`
}

// ResourcePoolStatus represents the current status of a resource pool.

type ResourcePoolStatus struct {
	State string `json:"state"` // AVAILABLE, UNAVAILABLE, MAINTENANCE

	Health string `json:"health"` // HEALTHY, DEGRADED, UNHEALTHY

	Utilization float64 `json:"utilization"`

	LastHealthCheck time.Time `json:"lastHealthCheck"`

	ErrorMessage string `json:"errorMessage,omitempty"`
}

// ResourceCapacity represents resource capacity information.

type ResourceCapacity struct {
	CPU *ResourceMetric `json:"cpu,omitempty"`

	Memory *ResourceMetric `json:"memory,omitempty"`

	Storage *ResourceMetric `json:"storage,omitempty"`

	Network *ResourceMetric `json:"network,omitempty"`

	Accelerators *ResourceMetric `json:"accelerators,omitempty"`

	CustomResources map[string]*ResourceMetric `json:"customResources,omitempty"`
}

<<<<<<< HEAD
=======
// MetricData represents basic metric data with value and unit
type MetricData struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

>>>>>>> 6835433495e87288b95961af7173d866977175ff
// ResourceMetric represents a resource metric with total, available, and used values.

type ResourceMetric struct {
	Total string `json:"total"`

	Available string `json:"available"`

	Used string `json:"used"`

	Unit string `json:"unit"`

	Utilization float64 `json:"utilization"`
}

// AlarmDictionary defines alarm information for a resource type.

type AlarmDictionary struct {
	ID string `json:"id"`

	Name string `json:"name"`

	AlarmDefinitions []*AlarmDefinition `json:"alarmDefinitions"`
}

// AlarmDefinition defines a specific alarm type.

type AlarmDefinition struct {
	AlarmDefinitionID string `json:"alarmDefinitionId"`

	AlarmName string `json:"alarmName"`

	AlarmDescription string `json:"alarmDescription,omitempty"`

	ProposedRepairActions []string `json:"proposedRepairActions,omitempty"`

	AlarmAdditionalFields map[string]string `json:"alarmAdditionalFields,omitempty"`

	AlarmLastChange string `json:"alarmLastChange,omitempty"`
}

// ResourceStatus represents the current status of a resource.

type ResourceStatus struct {
	State string `json:"state"` // PENDING, ACTIVE, INACTIVE, FAILED, DELETING

	OperationalState string `json:"operationalState"` // ENABLED, DISABLED

	AdministrativeState string `json:"administrativeState"` // LOCKED, UNLOCKED, SHUTTINGDOWN

	UsageState string `json:"usageState"` // IDLE, ACTIVE, BUSY

	Health string `json:"health"` // HEALTHY, DEGRADED, UNHEALTHY, UNKNOWN

	LastHealthCheck time.Time `json:"lastHealthCheck"`

	ErrorMessage string `json:"errorMessage,omitempty"`

	Conditions []ResourceCondition `json:"conditions,omitempty"`

	Metrics json.RawMessage `json:"metrics,omitempty"`
}

// ResourceCondition represents a condition of the resource.

type ResourceCondition struct {
	Type string `json:"type"`

	Status string `json:"status"` // True, False, Unknown

	Reason string `json:"reason,omitempty"`

	Message string `json:"message,omitempty"`

	LastTransitionTime time.Time `json:"lastTransitionTime"`

	LastUpdateTime time.Time `json:"lastUpdateTime,omitempty"`
}

// Filter types for resource queries.

// ResourcePoolFilter defines filters for querying resource pools.

type ResourcePoolFilter struct {
	Names []string `json:"names,omitempty"`

	OCloudIDs []string `json:"oCloudIds,omitempty"`

	Providers []string `json:"providers,omitempty"`

	Regions []string `json:"regions,omitempty"`

	States []string `json:"states,omitempty"`

	HealthStates []string `json:"healthStates,omitempty"`

	Labels map[string]string `json:"labels,omitempty"`

	CreatedAfter *time.Time `json:"createdAfter,omitempty"`

	CreatedBefore *time.Time `json:"createdBefore,omitempty"`

	Limit int `json:"limit,omitempty"`

	Offset int `json:"offset,omitempty"`

	SortBy string `json:"sortBy,omitempty"`

	SortOrder string `json:"sortOrder,omitempty"` // ASC, DESC
}

// Request types for resource management operations.

// CreateResourcePoolRequest represents a request to create a resource pool.

type CreateResourcePoolRequest struct {
	Name string `json:"name"`

	Description string `json:"description,omitempty"`

	Location string `json:"location,omitempty"`

	OCloudID string `json:"oCloudId"`

	GlobalLocationID string `json:"globalLocationId,omitempty"`

	Provider string `json:"provider"`

	Region string `json:"region,omitempty"`

	Zone string `json:"zone,omitempty"`

	Configuration *runtime.RawExtension `json:"configuration,omitempty"`

	Extensions map[string]interface{} `json:"extensions,omitempty"`

	Metadata map[string]string `json:"metadata,omitempty"`
}

// UpdateResourcePoolRequest represents a request to update a resource pool.

type UpdateResourcePoolRequest struct {
	Name *string `json:"name,omitempty"`

	Description *string `json:"description,omitempty"`

	Location *string `json:"location,omitempty"`

	GlobalLocationID *string `json:"globalLocationId,omitempty"`

	Configuration *runtime.RawExtension `json:"configuration,omitempty"`

	Extensions map[string]interface{} `json:"extensions,omitempty"`

	Metadata map[string]string `json:"metadata,omitempty"`
}

// Node represents a compute node in the infrastructure inventory.

type Node struct {
	NodeID string `json:"nodeId"`

	Name string `json:"name"`

	Description string `json:"description,omitempty"`

	ResourcePoolID string `json:"resourcePoolId"`

	NodeType string `json:"nodeType"` // PHYSICAL, VIRTUAL, CONTAINER

	Status *NodeStatus `json:"status"`

	Capacity *ResourceCapacity `json:"capacity"`

	Architecture string `json:"architecture,omitempty"`

	OperatingSystem *OperatingSystemInfo `json:"operatingSystem,omitempty"`

	NetworkInterfaces []*NetworkInterface `json:"networkInterfaces,omitempty"`

	StorageDevices []*StorageDevice `json:"storageDevices,omitempty"`

	Accelerators []*AcceleratorDevice `json:"accelerators,omitempty"`

	Location *NodeLocation `json:"location,omitempty"`

	Extensions map[string]interface{} `json:"extensions,omitempty"`

	CreatedAt time.Time `json:"createdAt"`

	UpdatedAt time.Time `json:"updatedAt"`
}

// NodeStatus represents the current status of a node.

type NodeStatus struct {
	State string `json:"state"` // READY, NOT_READY, UNKNOWN

	Phase string `json:"phase"` // ACTIVE, INACTIVE, MAINTENANCE

	Health string `json:"health"` // HEALTHY, DEGRADED, UNHEALTHY

	LastHeartbeat time.Time `json:"lastHeartbeat"`

	Conditions []NodeCondition `json:"conditions,omitempty"`

	Metrics json.RawMessage `json:"metrics,omitempty"`

	Alarms []string `json:"alarms,omitempty"`
}

// NodeCondition represents a condition of the node.

type NodeCondition struct {
	Type string `json:"type"`

	Status string `json:"status"`

	Reason string `json:"reason,omitempty"`

	Message string `json:"message,omitempty"`

	LastTransitionTime time.Time `json:"lastTransitionTime"`
}

// OperatingSystemInfo represents operating system information.

type OperatingSystemInfo struct {
	Name string `json:"name"`

	Version string `json:"version"`

	KernelVersion string `json:"kernelVersion,omitempty"`

	Architecture string `json:"architecture,omitempty"`

	ContainerRuntime string `json:"containerRuntime,omitempty"`

	KubernetesVersion string `json:"kubernetesVersion,omitempty"`
}

// NetworkInterface represents a network interface on a node.

type NetworkInterface struct {
	Name string `json:"name"`

	Type string `json:"type"` // ETHERNET, WIRELESS, LOOPBACK

	MACAddress string `json:"macAddress,omitempty"`

	IPAddresses []string `json:"ipAddresses,omitempty"`

	Speed string `json:"speed,omitempty"`

	MTU int `json:"mtu,omitempty"`

	State string `json:"state"` // UP, DOWN, UNKNOWN
}

// StorageDevice represents a storage device on a node.

type StorageDevice struct {
	Name string `json:"name"`

	Type string `json:"type"` // HDD, SSD, NVME

	Size string `json:"size"`

	MountPoint string `json:"mountPoint,omitempty"`

	FileSystem string `json:"fileSystem,omitempty"`

	Available string `json:"available,omitempty"`

	Used string `json:"used,omitempty"`

	Utilization float64 `json:"utilization,omitempty"`
}

// AcceleratorDevice represents an accelerator device on a node.

type AcceleratorDevice struct {
	Name string `json:"name"`

	Type string `json:"type"` // GPU, FPGA, TPU, SR-IOV

	Model string `json:"model,omitempty"`

	Vendor string `json:"vendor,omitempty"`

	Memory string `json:"memory,omitempty"`

	Count int `json:"count"`

	Available int `json:"available"`

	Used int `json:"used"`

	Utilization float64 `json:"utilization,omitempty"`
}

// NodeLocation represents the physical location of a node.

type NodeLocation struct {
	Datacenter string `json:"datacenter,omitempty"`

	Rack string `json:"rack,omitempty"`

	Chassis string `json:"chassis,omitempty"`

	Slot string `json:"slot,omitempty"`

	Latitude float64 `json:"latitude,omitempty"`

	Longitude float64 `json:"longitude,omitempty"`

	Altitude float64 `json:"altitude,omitempty"`
}

// NodeFilter defines filters for querying nodes.

type NodeFilter struct {
	Names []string `json:"names,omitempty"`

	NodeTypes []string `json:"nodeTypes,omitempty"`

	ResourcePoolIDs []string `json:"resourcePoolIds,omitempty"`

	States []string `json:"states,omitempty"`

	HealthStates []string `json:"healthStates,omitempty"`

	Architectures []string `json:"architectures,omitempty"`

	Labels map[string]string `json:"labels,omitempty"`

	MinCPU string `json:"minCpu,omitempty"`

	MinMemory string `json:"minMemory,omitempty"`

	HasAccelerators *bool `json:"hasAccelerators,omitempty"`

	Limit int `json:"limit,omitempty"`

	Offset int `json:"offset,omitempty"`

	SortBy string `json:"sortBy,omitempty"`

	SortOrder string `json:"sortOrder,omitempty"`
}

// Common constants for resource management.

const (

	// Resource Pool States.

	ResourcePoolStateAvailable = "AVAILABLE"

	// ResourcePoolStateUnavailable holds resourcepoolstateunavailable value.

	ResourcePoolStateUnavailable = "UNAVAILABLE"

	// ResourcePoolStateMaintenance holds resourcepoolstatemaintenance value.

	ResourcePoolStateMaintenance = "MAINTENANCE"

	// Resource Pool Health States.

	ResourcePoolHealthHealthy = "HEALTHY"

	// ResourcePoolHealthDegraded holds resourcepoolhealthdegraded value.

	ResourcePoolHealthDegraded = "DEGRADED"

	// ResourcePoolHealthUnhealthy holds resourcepoolhealthunhealthy value.

	ResourcePoolHealthUnhealthy = "UNHEALTHY"

	// Resource States.

	ResourceStatePending = "PENDING"

	// ResourceStateActive holds resourcestateactive value.

	ResourceStateActive = "ACTIVE"

	// ResourceStateInactive holds resourcestateinactive value.

	ResourceStateInactive = "INACTIVE"

	// ResourceStateFailed holds resourcestatefailed value.

	ResourceStateFailed = "FAILED"

	// ResourceStateDeleting holds resourcestatedeleting value.

	ResourceStateDeleting = "DELETING"

	// Administrative States.

	AdminStateUnlocked = "UNLOCKED"

	// AdminStateLocked holds adminstatelocked value.

	AdminStateLocked = "LOCKED"

	// AdminStateShuttingdown holds adminstateshuttingdown value.

	AdminStateShuttingdown = "SHUTTINGDOWN"

	// Operational States.

	OpStateEnabled = "ENABLED"

	// OpStateDisabled holds opstatedisabled value.

	OpStateDisabled = "DISABLED"

	// Usage States.

	UsageStateIdle = "IDLE"

	// UsageStateActive holds usagestateactive value.

	UsageStateActive = "ACTIVE"

	// UsageStateBusy holds usagestatebusy value.

	UsageStateBusy = "BUSY"

	// Health States.

	HealthStateHealthy = "HEALTHY"

	// HealthStateDegraded holds healthstatedegraded value.

	HealthStateDegraded = "DEGRADED"

	// HealthStateUnhealthy holds healthstateunhealthy value.

	HealthStateUnhealthy = "UNHEALTHY"

	// HealthStateUnknown holds healthstateunknown value.

	HealthStateUnknown = "UNKNOWN"

	// Node States.

	NodeStateReady = "READY"

	// NodeStateNotReady holds nodestatenotready value.

	NodeStateNotReady = "NOT_READY"

	// NodeStateUnknown holds nodestateunknown value.

	NodeStateUnknown = "UNKNOWN"

	// Node Types.

	NodeTypePhysical = "PHYSICAL"

	// NodeTypeVirtual holds nodetypevirtual value.

	NodeTypeVirtual = "VIRTUAL"

	// NodeTypeContainer holds nodetypecontainer value.

	NodeTypeContainer = "CONTAINER"

	// Resource Categories (removed duplicates - see resource_types.go for category constants).

)
