// Package observability provides types for service mesh observability.

package observability

import (
	
	"encoding/json"
"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/servicemesh/abstraction"
)

// MTLSReport represents a comprehensive mTLS status report.

type MTLSReport struct {
	Timestamp time.Time `json:"timestamp"`

	Provider abstraction.ServiceMeshProvider `json:"provider"`

	TotalServices int `json:"totalServices"`

	MTLSEnabledServices int `json:"mtlsEnabledServices"`

	Coverage float64 `json:"coverage"`

	CertificateStatus []CertificateMetrics `json:"certificateStatus"`

	HealthScore float64 `json:"healthScore"`

	Recommendations []string `json:"recommendations"`
}

// CertificateMetrics represents certificate metrics.

type CertificateMetrics struct {
	Service string `json:"service"`

	Namespace string `json:"namespace"`

	Valid bool `json:"valid"`

	ExpiryDate time.Time `json:"expiryDate"`

	DaysUntilExpiry int `json:"daysUntilExpiry"`

	NeedsRotation bool `json:"needsRotation"`
}

// DependencyVisualization represents service dependency visualization data.

type DependencyVisualization struct {
	Namespace string `json:"namespace"`

	Timestamp time.Time `json:"timestamp"`

	Nodes []VisualizationNode `json:"nodes"`

	Edges []VisualizationEdge `json:"edges"`

	HasCycles bool `json:"hasCycles"`

	Cycles [][]string `json:"cycles,omitempty"`
}

// VisualizationNode represents a node in the dependency visualization.

type VisualizationNode struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Type string `json:"type"`

	MTLSEnabled bool `json:"mtlsEnabled"`

	Metrics NodeMetrics `json:"metrics"`
}

// NodeMetrics represents metrics for a node.

type NodeMetrics struct {
	RequestRate float64 `json:"requestRate"`

	ErrorRate float64 `json:"errorRate"`

	P99Latency float64 `json:"p99Latency"`
}

// VisualizationEdge represents an edge in the dependency visualization.

type VisualizationEdge struct {
	Source string `json:"source"`

	Target string `json:"target"`

	Protocol string `json:"protocol"`

	MTLSEnabled bool `json:"mtlsEnabled"`

	Metrics EdgeMetrics `json:"metrics"`
}

// EdgeMetrics represents metrics for an edge.

type EdgeMetrics struct {
	RequestRate float64 `json:"requestRate"`

	ErrorRate float64 `json:"errorRate"`

	Latency float64 `json:"latency"`
}

// ServiceMeshMetrics represents aggregated service mesh metrics.

type ServiceMeshMetrics struct {
	Timestamp time.Time `json:"timestamp"`

	MTLSConnections int64 `json:"mtlsConnections"`

	CertificateRotations int64 `json:"certificateRotations"`

	PolicyApplications int64 `json:"policyApplications"`

	AverageLatency float64 `json:"averageLatency"`

	P99Latency float64 `json:"p99Latency"`

	ErrorRate float64 `json:"errorRate"`

	ThroughputRPS float64 `json:"throughputRps"`

	ActiveServices int `json:"activeServices"`

	HealthyServices int `json:"healthyServices"`

	UnhealthyServices int `json:"unhealthyServices"`
}

// SecurityEvent represents a security-related event in the mesh.

type SecurityEvent struct {
	Timestamp time.Time `json:"timestamp"`

	Type string `json:"type"`

	Severity string `json:"severity"`

	Source string `json:"source"`

	Destination string `json:"destination"`

	Message string `json:"message"`

	Details json.RawMessage `json:"details,omitempty"`
}

// PerformanceMetrics represents performance metrics for the mesh.

type PerformanceMetrics struct {
	Timestamp time.Time `json:"timestamp"`

	ServiceMetrics map[string]ServicePerformance `json:"serviceMetrics"`

	OverallLatency LatencyDistribution `json:"overallLatency"`

	OverallThroughput float64 `json:"overallThroughput"`

	ErrorRate float64 `json:"errorRate"`
}

// ServicePerformance represents performance metrics for a single service.

type ServicePerformance struct {
	Service string `json:"service"`

	Namespace string `json:"namespace"`

	RequestRate float64 `json:"requestRate"`

	ErrorRate float64 `json:"errorRate"`

	Latency LatencyDistribution `json:"latency"`

	CPUUsage float64 `json:"cpuUsage"`

	MemoryUsage float64 `json:"memoryUsage"`

	ActiveConns int `json:"activeConnections"`
}

// LatencyDistribution represents latency distribution metrics.

type LatencyDistribution struct {
	P50 float64 `json:"p50"`

	P90 float64 `json:"p90"`

	P95 float64 `json:"p95"`

	P99 float64 `json:"p99"`

	Mean float64 `json:"mean"`

	Max float64 `json:"max"`

	Min float64 `json:"min"`
}

// TracingData represents distributed tracing data.

type TracingData struct {
	TraceID string `json:"traceId"`

	SpanID string `json:"spanId"`

	ParentID string `json:"parentId,omitempty"`

	Service string `json:"service"`

	Operation string `json:"operation"`

	StartTime time.Time `json:"startTime"`

	Duration float64 `json:"duration"`

	Status string `json:"status"`

	Tags map[string]string `json:"tags,omitempty"`

	Events []TraceEvent `json:"events,omitempty"`
}

// TraceEvent represents an event within a trace.

type TraceEvent struct {
	Timestamp time.Time `json:"timestamp"`

	Name string `json:"name"`

	Attributes map[string]string `json:"attributes,omitempty"`
}

// AlertConfiguration represents alert configuration.

type AlertConfiguration struct {
	Name string `json:"name"`

	Type string `json:"type"`

	Condition AlertCondition `json:"condition"`

	Actions []AlertAction `json:"actions"`

	Enabled bool `json:"enabled"`

	Labels map[string]string `json:"labels,omitempty"`
}

// AlertCondition represents an alert condition.

type AlertCondition struct {
	Metric string `json:"metric"`

	Operator string `json:"operator"`

	Threshold float64 `json:"threshold"`

	Duration string `json:"duration"`
}

// AlertAction represents an action to take when alert triggers.

type AlertAction struct {
	Type string `json:"type"`

	Config map[string]string `json:"config"`
}

// Dashboard represents a monitoring dashboard configuration.

type Dashboard struct {
	Name string `json:"name"`

	Description string `json:"description"`

	Panels []DashboardPanel `json:"panels"`

	Variables []DashboardVariable `json:"variables,omitempty"`

	RefreshRate string `json:"refreshRate"`
}

// DashboardPanel represents a panel in a dashboard.

type DashboardPanel struct {
	ID string `json:"id"`

	Title string `json:"title"`

	Type string `json:"type"`

	Query string `json:"query"`

	Position PanelPosition `json:"position"`

	Options json.RawMessage `json:"options,omitempty"`
}

// PanelPosition represents the position of a panel.

type PanelPosition struct {
	X int `json:"x"`

	Y int `json:"y"`

	Width int `json:"width"`

	Height int `json:"height"`
}

// DashboardVariable represents a dashboard variable.

type DashboardVariable struct {
	Name string `json:"name"`

	Type string `json:"type"`

	Query string `json:"query,omitempty"`

	Options []string `json:"options,omitempty"`

	Default string `json:"default,omitempty"`
}
