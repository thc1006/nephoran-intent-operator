package providers

import (
	"time"
)

// ClusterSpec defines the specification for creating a Kubernetes cluster
type ClusterSpec struct {
	Name          string            `json:"name"`
	Version       string            `json:"version"`
	NodeCount     int               `json:"nodeCount"`
	NodeSize      string            `json:"nodeSize"`
	Region        string            `json:"region,omitempty"`
	Zones         []string          `json:"zones,omitempty"`
	NetworkConfig *NetworkConfig    `json:"networkConfig,omitempty"`
	AddOns        []string          `json:"addOns,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	Annotations   map[string]string `json:"annotations,omitempty"`
}

// ClusterResource represents a managed Kubernetes cluster
type ClusterResource struct {
	Resource
	Endpoint   string             `json:"endpoint"`
	Version    string             `json:"version"`
	NodeCount  int                `json:"nodeCount"`
	Status     ClusterStatus      `json:"status"`
	Conditions []ClusterCondition `json:"conditions,omitempty"`
}

// ClusterStatus represents the status of a cluster
type ClusterStatus string

const (
	ClusterStatusProvisioning ClusterStatus = "provisioning"
	ClusterStatusRunning      ClusterStatus = "running"
	ClusterStatusUpgrading    ClusterStatus = "upgrading"
	ClusterStatusScaling      ClusterStatus = "scaling"
	ClusterStatusError        ClusterStatus = "error"
	ClusterStatusTerminating  ClusterStatus = "terminating"
)

// ClusterCondition represents a condition of cluster status
type ClusterCondition struct {
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	Reason      string    `json:"reason,omitempty"`
	Message     string    `json:"message,omitempty"`
	LastUpdated time.Time `json:"lastUpdated"`
}

// NetworkConfig defines network configuration for resources
type NetworkConfig struct {
	CIDR          string   `json:"cidr,omitempty"`
	SubnetCIDRs   []string `json:"subnetCIDRs,omitempty"`
	ServiceCIDR   string   `json:"serviceCIDR,omitempty"`
	PodCIDR       string   `json:"podCIDR,omitempty"`
	DNSServers    []string `json:"dnsServers,omitempty"`
	EnablePrivate bool     `json:"enablePrivate,omitempty"`
}

// NetworkSpec defines the specification for creating a network
type NetworkSpec struct {
	Name   string            `json:"name"`
	Type   string            `json:"type"` // vpc, subnet, security-group
	CIDR   string            `json:"cidr,omitempty"`
	Region string            `json:"region,omitempty"`
	Labels map[string]string `json:"labels,omitempty"`
	Rules  []SecurityRule    `json:"rules,omitempty"`
}

// NetworkResource represents a managed network
type NetworkResource struct {
	Resource
	Type        string   `json:"type"`
	CIDR        string   `json:"cidr,omitempty"`
	Subnets     []Subnet `json:"subnets,omitempty"`
	Routes      []Route  `json:"routes,omitempty"`
	Attachments []string `json:"attachments,omitempty"` // attached resource IDs
}

// SecurityRule defines network security rules
type SecurityRule struct {
	Protocol  string `json:"protocol"` // tcp, udp, icmp
	Port      string `json:"port,omitempty"`
	Source    string `json:"source,omitempty"`
	Target    string `json:"target,omitempty"`
	Action    string `json:"action"`    // allow, deny
	Direction string `json:"direction"` // inbound, outbound
}

// Subnet represents a network subnet
type Subnet struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	CIDR             string `json:"cidr"`
	AvailabilityZone string `json:"availabilityZone,omitempty"`
}

// Route represents a network route
type Route struct {
	Destination string `json:"destination"`
	NextHop     string `json:"nextHop"`
	Interface   string `json:"interface,omitempty"`
}

// NetworkTopology represents network connectivity information
type NetworkTopology struct {
	NetworkID   string               `json:"networkId"`
	Nodes       []TopologyNode       `json:"nodes"`
	Connections []TopologyConnection `json:"connections"`
	Metadata    map[string]string    `json:"metadata,omitempty"`
}

// TopologyNode represents a node in network topology
type TopologyNode struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Type     string            `json:"type"`
	Address  string            `json:"address,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// TopologyConnection represents a connection between nodes
type TopologyConnection struct {
	Source   string            `json:"source"`
	Target   string            `json:"target"`
	Type     string            `json:"type"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// VolumeSpec defines the specification for creating a volume
type VolumeSpec struct {
	Name             string            `json:"name"`
	Size             int64             `json:"size"` // size in bytes
	Type             string            `json:"type"` // ssd, hdd, nvme
	Region           string            `json:"region,omitempty"`
	AvailabilityZone string            `json:"availabilityZone,omitempty"`
	Encrypted        bool              `json:"encrypted,omitempty"`
	Labels           map[string]string `json:"labels,omitempty"`
}

// VolumeResource represents a managed persistent volume
type VolumeResource struct {
	Resource
	Size             int64    `json:"size"`
	Type             string   `json:"type"`
	Encrypted        bool     `json:"encrypted"`
	AvailabilityZone string   `json:"availabilityZone,omitempty"`
	AttachedTo       string   `json:"attachedTo,omitempty"` // node ID if attached
	Snapshots        []string `json:"snapshots,omitempty"`  // snapshot IDs
}

// SnapshotResource represents a volume snapshot
type SnapshotResource struct {
	Resource
	VolumeID    string `json:"volumeId"`
	Size        int64  `json:"size"`
	Description string `json:"description,omitempty"`
}

// MetricsQuery defines parameters for metrics collection
type MetricsQuery struct {
	MetricNames []string          `json:"metricNames"`
	StartTime   time.Time         `json:"startTime"`
	EndTime     time.Time         `json:"endTime"`
	Interval    time.Duration     `json:"interval,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// MetricsResult contains collected metrics data
type MetricsResult struct {
	ResourceID string             `json:"resourceId"`
	Timestamp  time.Time          `json:"timestamp"`
	Metrics    map[string]float64 `json:"metrics"`
	Labels     map[string]string  `json:"labels,omitempty"`
}

// LogsQuery defines parameters for log collection
type LogsQuery struct {
	StartTime time.Time         `json:"startTime"`
	EndTime   time.Time         `json:"endTime"`
	Level     string            `json:"level,omitempty"` // debug, info, warn, error
	Limit     int               `json:"limit,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// LogsResult contains collected log data
type LogsResult struct {
	ResourceID string     `json:"resourceId"`
	Logs       []LogEntry `json:"logs"`
	TotalCount int        `json:"totalCount"`
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time         `json:"timestamp"`
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// AlertSpec defines the specification for creating an alert
type AlertSpec struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Query       string            `json:"query"`
	Threshold   float64           `json:"threshold"`
	Comparison  string            `json:"comparison"` // gt, lt, eq, ne
	Duration    time.Duration     `json:"duration"`
	Actions     []AlertAction     `json:"actions,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// AlertResource represents a monitoring alert
type AlertResource struct {
	Resource
	Query     string        `json:"query"`
	Threshold float64       `json:"threshold"`
	State     string        `json:"state"` // firing, pending, inactive
	LastFired *time.Time    `json:"lastFired,omitempty"`
	Actions   []AlertAction `json:"actions,omitempty"`
}

// AlertAction defines an action to take when alert fires
type AlertAction struct {
	Type   string                 `json:"type"` // webhook, email, slack
	Config map[string]interface{} `json:"config"`
}

// HealthStatus represents the health status of a resource
type HealthStatus struct {
	ResourceID string            `json:"resourceId"`
	Status     string            `json:"status"` // healthy, warning, critical, unknown
	Checks     []HealthCheck     `json:"checks"`
	Timestamp  time.Time         `json:"timestamp"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// HealthCheck represents a single health check result
type HealthCheck struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"` // pass, fail, warn
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}
