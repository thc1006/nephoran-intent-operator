package performance_tests

import (
	
	"encoding/json"
"time"
)

// ResourceUsageMetrics contains resource usage statistics
type ResourceUsageMetrics struct {
	CPUUsage    float64                `json:"cpu_usage"`
	MemoryUsage float64                `json:"memory_usage"`
	DiskIO      float64                `json:"disk_io"`
	NetworkIO   float64                `json:"network_io"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    json.RawMessage `json:"metadata"`
}

// TimePointMetric represents a point in time with a value (avoiding redeclaration)
type TimePointMetric struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}
