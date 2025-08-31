package monitoringtypes

import (
	"time"
)

// LatencyReport contains end-to-end latency measurements
type LatencyReport struct {
	TraceID            string
	StartTime          time.Time
	EndTime            time.Time
	TotalLatency       time.Duration
	InterfaceLatencies map[string]time.Duration
	Spans              []SpanReport
}

// SpanReport contains information about a single span
type SpanReport struct {
	Name       string
	StartTime  time.Time
	EndTime    time.Time
	Duration   time.Duration
	Attributes map[string]interface{}
}

// CheckResult represents the result of a synthetic check
type CheckResult struct {
	CheckID    string
	CheckType  string
	Target     string
	Success    bool
	Latency    time.Duration
	Error      error
	Timestamp  time.Time
	Attributes map[string]interface{}
	Details    map[string]interface{}
}

// Alert represents a monitoring alert
type Alert struct {
	ID           string
	Name         string
	Description  string
	Severity     string
	Status       string
	Labels       map[string]string
	Annotations  map[string]string
	StartsAt     time.Time
	EndsAt       time.Time
	GeneratorURL string
}

// PerformanceReport contains system performance metrics
type PerformanceReport struct {
	Timestamp   time.Time
	ServiceName string
	Metrics     map[string]float64
	Latencies   map[string]time.Duration
	ErrorRates  map[string]float64
	Throughput  map[string]float64
}

// MonitoringMetadata contains metadata for monitoring operations
type MonitoringMetadata struct {
	Namespace   string
	Service     string
	Version     string
	Environment string
	Labels      map[string]string
}

// HealthStatus represents the health status of a component
type HealthStatus struct {
	Component string
	Status    string
	Message   string
	Timestamp time.Time
	Checks    []HealthCheck
}

// HealthCheck represents an individual health check
type HealthCheck struct {
	Name     string
	Status   string
	Message  string
	Duration time.Duration
	Error    error
}
