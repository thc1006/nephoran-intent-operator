package o2models

import (
	"time"
)

// Core O2 IMS Data Models following O-RAN.WG6.O2ims-Interface-v01.01 specification

// ResourcePool represents a collection of infrastructure resources
type ResourcePool struct {
	ResourcePoolID   string                 `json:"resourcePoolId"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description,omitempty"`
	Location         string                 `json:"location,omitempty"`
	OCloudID         string                 `json:"oCloudId"`
	GlobalLocationID string                 `json:"globalLocationId,omitempty"`
	Extensions       map[string]interface{} `json:"extensions,omitempty"`

	// Nephoran-specific extensions
	Provider  string              `json:"provider"`
	Region    string              `json:"region,omitempty"`
	Zone      string              `json:"zone,omitempty"`
	Capacity  *ResourceCapacity   `json:"capacity,omitempty"`
	Status    *ResourcePoolStatus `json:"status,omitempty"`
	CreatedAt time.Time           `json:"createdAt"`
	UpdatedAt time.Time           `json:"updatedAt"`
}

// ResourcePoolStatus represents the current status of a resource pool
type ResourcePoolStatus struct {
	State           string    `json:"state"`  // AVAILABLE, UNAVAILABLE, MAINTENANCE
	Health          string    `json:"health"` // HEALTHY, DEGRADED, UNHEALTHY
	Utilization     float64   `json:"utilization"`
	LastHealthCheck time.Time `json:"lastHealthCheck"`
	ErrorMessage    string    `json:"errorMessage,omitempty"`
}

// Note: ResourceCapacity, ResourceMetric, AlarmDictionary, AlarmDefinition, 
// ResourceStatus, and ResourceCondition are defined in resource_types.go
// Node represents a compute node in the infrastructure inventory
type Node struct {
	NodeID            string                 `json:"nodeId"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description,omitempty"`
	ResourcePoolID    string                 `json:"resourcePoolId"`
	NodeType          string                 `json:"nodeType"` // PHYSICAL, VIRTUAL, CONTAINER
	Status            *NodeStatus            `json:"status"`
	Capacity          *ResourceCapacity      `json:"capacity"`
	Architecture      string                 `json:"architecture,omitempty"`
	OperatingSystem   *OperatingSystemInfo   `json:"operatingSystem,omitempty"`
	NetworkInterfaces []*NetworkInterface    `json:"networkInterfaces,omitempty"`
	StorageDevices    []*StorageDevice       `json:"storageDevices,omitempty"`
	Accelerators      []*AcceleratorDevice   `json:"accelerators,omitempty"`
	Location          *NodeLocation          `json:"location,omitempty"`
	Labels            map[string]string      `json:"labels,omitempty"`
	Annotations       map[string]string      `json:"annotations,omitempty"`
	CreatedAt         time.Time              `json:"createdAt"`
	UpdatedAt         time.Time              `json:"updatedAt"`
}

// NodeStatus represents the current status of a compute node
type NodeStatus struct {
	State           string               `json:"state"`  // READY, NOT_READY, UNKNOWN
	Phase           string               `json:"phase"`  // ACTIVE, INACTIVE, MAINTENANCE, PROVISIONING
	Health          string               `json:"health"` // HEALTHY, DEGRADED, UNHEALTHY
	Conditions      []*NodeCondition     `json:"conditions,omitempty"`
	Utilization     *ResourceUtilization `json:"utilization,omitempty"`
	LastHeartbeat   time.Time            `json:"lastHeartbeat"`
	LastHealthCheck time.Time            `json:"lastHealthCheck"`
	ErrorMessage    string               `json:"errorMessage,omitempty"`
	Events          []*NodeEvent         `json:"events,omitempty"`
}

// NodeCondition represents a condition of a node
type NodeCondition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	LastTransitionTime time.Time `json:"lastTransitionTime"`
	LastHeartbeatTime  time.Time `json:"lastHeartbeatTime"`
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
}

// NodeEvent represents an event related to a node
type NodeEvent struct {
	ID             string                 `json:"id"`
	Type           string                 `json:"type"` // NORMAL, WARNING, ERROR
	Reason         string                 `json:"reason"`
	Message        string                 `json:"message"`
	Source         string                 `json:"source"`
	FirstTimestamp time.Time              `json:"firstTimestamp"`
	LastTimestamp  time.Time              `json:"lastTimestamp"`
	Count          int32                  `json:"count"`
	AdditionalData map[string]interface{} `json:"additionalData,omitempty"`
}

// ResourceUtilization represents the current utilization of resources
type ResourceUtilization struct {
	CPU              *ResourceMetric        `json:"cpu,omitempty"`
	Memory           *ResourceMetric        `json:"memory,omitempty"`
	Storage          *ResourceMetric        `json:"storage,omitempty"`
	EphemeralStorage *ResourceMetric        `json:"ephemeralStorage,omitempty"`
	Pods             *ResourceMetric        `json:"pods,omitempty"`
	GPU              *ResourceMetric        `json:"gpu,omitempty"`
	Network          *NetworkUtilization    `json:"network,omitempty"`
	CustomResources  map[string]interface{} `json:"customResources,omitempty"`
	Timestamp        time.Time              `json:"timestamp"`
}

// NetworkUtilization represents network utilization metrics
type NetworkUtilization struct {
	RxBytes   int64   `json:"rxBytes"`
	TxBytes   int64   `json:"txBytes"`
	RxPackets int64   `json:"rxPackets"`
	TxPackets int64   `json:"txPackets"`
	RxErrors  int64   `json:"rxErrors,omitempty"`
	TxErrors  int64   `json:"txErrors,omitempty"`
	Bandwidth string  `json:"bandwidth,omitempty"`
	Latency   float64 `json:"latency,omitempty"`
}

// OperatingSystemInfo represents information about the operating system
type OperatingSystemInfo struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Distribution string `json:"distribution,omitempty"`
	Kernel       string `json:"kernel,omitempty"`
	Architecture string `json:"architecture,omitempty"`
}

// NetworkInterface represents a network interface
type NetworkInterface struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"` // ETHERNET, WIFI, LOOPBACK
	MacAddress  string   `json:"macAddress,omitempty"`
	IPAddresses []string `json:"ipAddresses,omitempty"`
	MTU         int      `json:"mtu,omitempty"`
	Speed       string   `json:"speed,omitempty"`
	State       string   `json:"state"` // UP, DOWN
}

// StorageDevice represents a storage device
type StorageDevice struct {
	Name       string `json:"name"`
	Type       string `json:"type"` // SSD, HDD, NVME
	Size       string `json:"size"`
	Model      string `json:"model,omitempty"`
	Vendor     string `json:"vendor,omitempty"`
	SerialNo   string `json:"serialNo,omitempty"`
	MountPoint string `json:"mountPoint,omitempty"`
	FileSystem string `json:"fileSystem,omitempty"`
	Health     string `json:"health,omitempty"`
}

// AcceleratorDevice represents an accelerator device (GPU, FPGA, etc.)
type AcceleratorDevice struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"` // GPU, FPGA, TPU
	Model    string                 `json:"model"`
	Vendor   string                 `json:"vendor,omitempty"`
	Memory   string                 `json:"memory,omitempty"`
	PCIAddr  string                 `json:"pciAddr,omitempty"`
	Driver   string                 `json:"driver,omitempty"`
	Status   string                 `json:"status"` // ACTIVE, INACTIVE, ERROR
	Features map[string]interface{} `json:"features,omitempty"`
}

// NodeLocation represents the physical location of a node
type NodeLocation struct {
	Region         string  `json:"region"`
	AvailableZone  string  `json:"availabilityZone,omitempty"`
	DataCenter     string  `json:"dataCenter,omitempty"`
	Rack           string  `json:"rack,omitempty"`
	Row            string  `json:"row,omitempty"`
	Latitude       float64 `json:"latitude,omitempty"`
	Longitude      float64 `json:"longitude,omitempty"`
	Address        string  `json:"address,omitempty"`
	Building       string  `json:"building,omitempty"`
	Floor          string  `json:"floor,omitempty"`
	Room           string  `json:"room,omitempty"`
	Position       string  `json:"position,omitempty"`
	AdministrativeRegion string `json:"administrativeRegion,omitempty"`
}

// ResourceElement represents an element within a resource
type ResourceElement struct {
	ElementID   string                 `json:"elementId"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Type        string                 `json:"type"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
	Status      *ResourceStatus        `json:"status,omitempty"`
}

// OCloud represents an O-Cloud infrastructure domain
type OCloud struct {
	OCloudID         string                 `json:"oCloudId"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description,omitempty"`
	ServiceURI       string                 `json:"serviceUri"`
	SupportedFeatures []string               `json:"supportedFeatures,omitempty"`
	Extensions       map[string]interface{} `json:"extensions,omitempty"`

	// Nephoran-specific extensions
	Region       string            `json:"region,omitempty"`
	Provider     string            `json:"provider,omitempty"`
	Version      string            `json:"version,omitempty"`
	Capabilities []string          `json:"capabilities,omitempty"`
	Status       *OCloudStatus     `json:"status,omitempty"`
	ResourcePools []*ResourcePool   `json:"resourcePools,omitempty"`
	DeploymentManagers []*DeploymentManager `json:"deploymentManagers,omitempty"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
}

// OCloudStatus represents the status of an O-Cloud
type OCloudStatus struct {
	State           string    `json:"state"`  // AVAILABLE, UNAVAILABLE, MAINTENANCE
	Health          string    `json:"health"` // HEALTHY, DEGRADED, UNHEALTHY
	Connectivity    string    `json:"connectivity"` // CONNECTED, DISCONNECTED, INTERMITTENT
	LastHealthCheck time.Time `json:"lastHealthCheck"`
	ErrorMessage    string    `json:"errorMessage,omitempty"`
}

// Filter types and Request types for resource management operations are defined in resource_types.go

// Constants for resource management

const (
	// Resource Pool States
	ResourcePoolStateAvailable   = "AVAILABLE"
	ResourcePoolStateUnavailable = "UNAVAILABLE"
	ResourcePoolStateMaintenance = "MAINTENANCE"

	// Node Types
	NodeTypePhysical  = "PHYSICAL"
	NodeTypeVirtual   = "VIRTUAL"
	NodeTypeContainer = "CONTAINER"

	// Node States
	NodeStateReady     = "READY"
	NodeStateNotReady  = "NOT_READY"
	NodeStateUnknown   = "UNKNOWN"

	// Node Phases
	NodePhaseActive       = "ACTIVE"
	NodePhaseInactive     = "INACTIVE"
	NodePhaseMaintenance  = "MAINTENANCE"
	NodePhaseProvisioning = "PROVISIONING"

	// Health States
	HealthStateHealthy   = "HEALTHY"
	HealthStateDegraded  = "DEGRADED"
	HealthStateUnhealthy = "UNHEALTHY"
	HealthStateUnknown   = "UNKNOWN"

	// O-Cloud States
	OCloudStateAvailable   = "AVAILABLE"
	OCloudStateUnavailable = "UNAVAILABLE"
	OCloudStateMaintenance = "MAINTENANCE"

	// Connectivity States
	ConnectivityStateConnected     = "CONNECTED"
	ConnectivityStateDisconnected  = "DISCONNECTED"
	ConnectivityStateIntermittent  = "INTERMITTENT"

	// Network Interface States
	NetworkInterfaceStateUp   = "UP"
	NetworkInterfaceStateDown = "DOWN"

	// Network Interface Types
	NetworkInterfaceTypeEthernet = "ETHERNET"
	NetworkInterfaceTypeWifi     = "WIFI"
	NetworkInterfaceTypeLoopback = "LOOPBACK"

	// Storage Device Types
	StorageDeviceTypeSSD  = "SSD"
	StorageDeviceTypeHDD  = "HDD"
	StorageDeviceTypeNVME = "NVME"

	// Accelerator Device Types
	AcceleratorTypeGPU  = "GPU"
	AcceleratorTypeFPGA = "FPGA"
	AcceleratorTypeTPU  = "TPU"

	// Accelerator Device Statuses
	AcceleratorStatusActive   = "ACTIVE"
	AcceleratorStatusInactive = "INACTIVE"
	AcceleratorStatusError    = "ERROR"

	// Event Types
	NodeEventTypeNormal  = "NORMAL"
	NodeEventTypeWarning = "WARNING"
	NodeEventTypeError   = "ERROR"
)
