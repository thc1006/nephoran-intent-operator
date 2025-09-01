package models

import "time"

// RuntimeMetadata represents runtime metadata for managed resources
type RuntimeMetadata struct {
    ResourceID        string    `json:"resource_id"`
    CreatedAt         time.Time `json:"created_at"`
    UpdatedAt         time.Time `json:"updated_at"`
    State             string    `json:"state"`
    HealthStatus      string    `json:"health_status"`
    LastHealthCheck   time.Time `json:"last_health_check"`
    OperationalStatus string    `json:"operational_status"`
    Tags              []string  `json:"tags,omitempty"`
    Version           string    `json:"version,omitempty"`
    Annotations       map[string]string `json:"annotations,omitempty"`
}

// RuntimeStatus represents the detailed runtime status of a resource
type RuntimeStatus struct {
    Metadata        RuntimeMetadata `json:"metadata"`
    ResourceConfig  map[string]interface{} `json:"resource_config"`
    CurrentLoad     float64 `json:"current_load,omitempty"`
    MemoryUsage     float64 `json:"memory_usage,omitempty"`
    CPUUtilization  float64 `json:"cpu_utilization,omitempty"`
    NetworkTraffic  NetworkTrafficMetrics `json:"network_traffic,omitempty"`
}

// NetworkTrafficMetrics represents network-related runtime metrics
type NetworkTrafficMetrics struct {
    InboundBandwidth  float64 `json:"inbound_bandwidth,omitempty"`
    OutboundBandwidth float64 `json:"outbound_bandwidth,omitempty"`
    PacketsReceived   int64   `json:"packets_received,omitempty"`
    PacketsSent       int64   `json:"packets_sent,omitempty"`
}

// HealthStatus represents the health status of a resource or component
type HealthStatus struct {
    Status      string    `json:"status"`
    Message     string    `json:"message,omitempty"`
    LastCheck   time.Time `json:"last_check"`
    Details     map[string]interface{} `json:"details,omitempty"`
}