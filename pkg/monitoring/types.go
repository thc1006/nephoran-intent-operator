// Package monitoring provides types and utilities for O-RAN monitoring and observability
package monitoring

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MetricType defines the type of metric being collected
type MetricType string

const (
	// Counter metrics that only increase
	MetricTypeCounter MetricType = "counter"
	// Gauge metrics that can increase or decrease
	MetricTypeGauge MetricType = "gauge"
	// Histogram metrics for latency/size distributions
	MetricTypeHistogram MetricType = "histogram"
	// Summary metrics with quantiles
	MetricTypeSummary MetricType = "summary"
)

// AlertSeverity defines the severity level of alerts
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

// HealthStatus is defined in health_checker_impl.go

// Metric represents a single metric with its metadata
type Metric struct {
	Name        string            `json:"name" yaml:"name"`
	Type        MetricType        `json:"type" yaml:"type"`
	Value       string            `json:"value" yaml:"value"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Timestamp   time.Time         `json:"timestamp" yaml:"timestamp"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
}

// Alert represents a monitoring alert
type Alert struct {
	ID          string            `json:"id" yaml:"id"`
	Name        string            `json:"name" yaml:"name"`
	Rule        string            `json:"rule,omitempty" yaml:"rule,omitempty"` // Store rule ID as string
	Component   string            `json:"component" yaml:"component"`
	Source      string            `json:"source" yaml:"source"`
	Severity    AlertSeverity     `json:"severity" yaml:"severity"`
	Status      string            `json:"status" yaml:"status"`
	Message     string            `json:"message" yaml:"message"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	StartsAt    time.Time         `json:"starts_at" yaml:"starts_at"`
	EndsAt      *time.Time        `json:"ends_at,omitempty" yaml:"ends_at,omitempty"`
	Silenced    bool              `json:"silenced" yaml:"silenced"`
	AckBy       string            `json:"ack_by,omitempty" yaml:"ack_by,omitempty"`
	AckAt       *time.Time        `json:"ack_at,omitempty" yaml:"ack_at,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	Timestamp   time.Time         `json:"timestamp" yaml:"timestamp"`
	Resolved    bool              `json:"resolved" yaml:"resolved"`
	ResolvedAt  *time.Time        `json:"resolvedAt,omitempty" yaml:"resolvedAt,omitempty"`
	Fingerprint string            `json:"fingerprint" yaml:"fingerprint"`
}

// HealthCheck is defined in health_checker_impl.go

// Header represents an HTTP header
type Header struct {
	Name  string `json:"name" yaml:"name"`
	Value string `json:"value" yaml:"value"`
}

// SyntheticCheck represents a synthetic monitoring check (unified definition)
type SyntheticCheck struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SyntheticCheckSpec   `json:"spec,omitempty"`
	Status SyntheticCheckStatus `json:"status,omitempty"`

	// Fields for backwards compatibility with existing code
	ID           string            `json:"id,omitempty"`
	Name         string            `json:"name,omitempty"`
	Type         string            `json:"type,omitempty"`
	Target       string            `json:"target,omitempty"`
	Interval     time.Duration     `json:"interval,omitempty"`
	Timeout      time.Duration     `json:"timeout,omitempty"`
	ThresholdMs  int64             `json:"threshold_ms,omitempty"`
	ExpectedCode int               `json:"expected_code,omitempty"`
	ExpectedBody string            `json:"expected_body,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	Enabled      bool              `json:"enabled,omitempty"`
}

// SyntheticCheckSpec defines the desired state of SyntheticCheck
type SyntheticCheckSpec struct {
	// Name of the synthetic check
	Name string `json:"name"`

	// Type of check (http, tcp, dns, grpc)
	Type string `json:"type"`

	// Target endpoint or resource to check
	Target string `json:"target"`

	// Interval between checks
	Interval metav1.Duration `json:"interval"`

	// Timeout for each check
	Timeout metav1.Duration `json:"timeout"`

	// HTTP-specific configuration
	HTTP *HTTPCheckSpec `json:"http,omitempty"`

	// TCP-specific configuration
	TCP *TCPCheckSpec `json:"tcp,omitempty"`

	// DNS-specific configuration
	DNS *DNSCheckSpec `json:"dns,omitempty"`

	// GRPC-specific configuration
	GRPC *GRPCCheckSpec `json:"grpc,omitempty"`

	// Labels to apply to metrics
	Labels map[string]string `json:"labels,omitempty"`

	// Alert conditions
	Alerts []AlertCondition `json:"alerts,omitempty"`
}

// HTTPCheckSpec defines HTTP-specific check parameters
type HTTPCheckSpec struct {
	Method          string            `json:"method,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
	Body            string            `json:"body,omitempty"`
	ExpectedStatus  int               `json:"expectedStatus,omitempty"`
	ExpectedContent string            `json:"expectedContent,omitempty"`
	FollowRedirects bool              `json:"followRedirects,omitempty"`
	TLSConfig       *TLSConfig        `json:"tlsConfig,omitempty"`
}

// TCPCheckSpec defines TCP-specific check parameters
type TCPCheckSpec struct {
	Port      int        `json:"port"`
	TLSConfig *TLSConfig `json:"tlsConfig,omitempty"`
}

// DNSCheckSpec defines DNS-specific check parameters
type DNSCheckSpec struct {
	QueryType     string   `json:"queryType,omitempty"`
	ExpectedIPs   []string `json:"expectedIPs,omitempty"`
	ExpectedCNAME string   `json:"expectedCNAME,omitempty"`
	Nameserver    string   `json:"nameserver,omitempty"`
}

// GRPCCheckSpec defines gRPC-specific check parameters
type GRPCCheckSpec struct {
	Service   string     `json:"service"`
	Method    string     `json:"method"`
	TLSConfig *TLSConfig `json:"tlsConfig,omitempty"`
}

// TLSConfig defines TLS configuration for checks
type TLSConfig struct {
	InsecureSkipVerify bool   `json:"insecureSkipVerify,omitempty"`
	ServerName         string `json:"serverName,omitempty"`
	CACert             string `json:"caCert,omitempty"`
	ClientCert         string `json:"clientCert,omitempty"`
	ClientKey          string `json:"clientKey,omitempty"`
}

// AlertCondition defines conditions for triggering alerts
type AlertCondition struct {
	Name      string          `json:"name"`
	Condition string          `json:"condition"`
	Threshold string          `json:"threshold"`
	Duration  metav1.Duration `json:"duration"`
	Severity  AlertSeverity   `json:"severity"`
	Message   string          `json:"message"`
}

// SyntheticCheckStatus defines the observed state of SyntheticCheck
type SyntheticCheckStatus struct {
	// Conditions represent the current state of the check
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastCheck timestamp of the last check execution
	LastCheck *metav1.Time `json:"lastCheck,omitempty"`

	// LastResult of the last check
	LastResult *CheckResult `json:"lastResult,omitempty"`

	// SuccessRate over the last window
	SuccessRate string `json:"successRate,omitempty"`

	// AverageLatency over the last window
	AverageLatency metav1.Duration `json:"averageLatency,omitempty"`

	// ActiveAlerts currently firing
	ActiveAlerts []string `json:"activeAlerts,omitempty"`
}

// CheckResult represents the result of a synthetic check execution (unified definition)
type CheckResult struct {
	// Timestamp when the check was executed
	Timestamp time.Time `json:"timestamp"`

	// Success indicates if the check passed
	Success bool `json:"success"`

	// Latency/ResponseTime of the check execution
	ResponseTime time.Duration `json:"response_time"`
	Latency      time.Duration `json:"latency,omitempty"` // Backwards compatibility

	// StatusCode for HTTP checks
	StatusCode int `json:"status_code,omitempty"`

	// ResponseSize in bytes
	ResponseSize int64 `json:"responseSize,omitempty"`

	// Error message if the check failed
	Error string `json:"error,omitempty"`

	// Additional fields for compatibility
	CheckID      string `json:"check_id,omitempty"`
	Message      string `json:"message,omitempty"`
	Availability string `json:"availability,omitempty"`

	// Metadata contains additional check-specific information
	Metadata map[string]string `json:"metadata,omitempty"`

	// DNSLookupTime for network checks
	DNSLookupTime metav1.Duration `json:"dnsLookupTime,omitempty"`

	// ConnectTime for TCP/HTTP checks
	ConnectTime metav1.Duration `json:"connectTime,omitempty"`

	// TLSHandshakeTime for TLS checks
	TLSHandshakeTime metav1.Duration `json:"tlsHandshakeTime,omitempty"`

	// FirstByteTime for HTTP checks
	FirstByteTime metav1.Duration `json:"firstByteTime,omitempty"`

	// Certificate information for TLS checks
	Certificate *CertificateInfo `json:"certificate,omitempty"`
}

// SyntheticMonitor manages synthetic monitoring checks (unified definition)
type SyntheticMonitor struct {
	checks     map[string]*SyntheticCheck
	results    map[string]*CheckResult
	mutex      sync.RWMutex
	logger     *slog.Logger
	httpClient *http.Client

	// Note: Extended test suite fields have been moved to sla_monitoring_architecture.go
	// to avoid circular dependencies. Use SLAMonitoringArchitecture for advanced testing.
}

// CertificateInfo contains information about TLS certificates
type CertificateInfo struct {
	Subject   string      `json:"subject"`
	Issuer    string      `json:"issuer"`
	NotBefore metav1.Time `json:"notBefore"`
	NotAfter  metav1.Time `json:"notAfter"`
	DNSNames  []string    `json:"dnsNames,omitempty"`
}

// SyntheticCheckList contains a list of SyntheticCheck
type SyntheticCheckList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SyntheticCheck `json:"items"`
}

// Note: Advanced test suite types are defined in sla_monitoring_architecture.go
// to maintain proper separation of concerns and avoid circular dependencies.

// Missing types that need to be defined
type AlertRule struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Expression  string            `json:"expression"`
	Severity    AlertSeverity     `json:"severity"`
	Duration    time.Duration     `json:"duration"`
	Cooldown    time.Duration     `json:"cooldown"`
	Component   string            `json:"component"`
	Enabled     bool              `json:"enabled"`
	LastFired   *time.Time        `json:"last_fired,omitempty"`
	Channels    []string          `json:"channels,omitempty"`
	Threshold   string            `json:"threshold"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// UPDATED METRICS COLLECTOR INTERFACE - FIXED METHOD SIGNATURES
type MetricsCollector interface {
	// Basic collection method with context
	CollectMetrics(ctx context.Context, source string) (*MetricsData, error)
	Start(ctx context.Context) error                                              // FIXED: Added context parameter
	Stop() error
	RegisterMetrics(registry interface{}) error

	// Controller instrumentation methods
	UpdateControllerHealth(controllerName, component string, healthy bool)
	RecordKubernetesAPILatency(latency time.Duration)

	// NetworkIntent metrics
	UpdateNetworkIntentStatus(name, namespace, intentType, status string)
	RecordNetworkIntentProcessed(intentType, status string, duration time.Duration)
	RecordNetworkIntentRetry(name, namespace, reason string)

	// LLM metrics
	RecordLLMRequest(model, status string, duration time.Duration, tokensUsed int)

	// CNF deployment metrics
	RecordCNFDeployment(functionName string, duration time.Duration)

	// E2NodeSet metrics
	RecordE2NodeSetOperation(operation string, duration time.Duration)
	UpdateE2NodeSetReplicas(name, namespace, status string, count int)
	RecordE2NodeSetScaling(name, namespace, direction string)

	// O-RAN interface metrics
	RecordORANInterfaceRequest(interfaceType, operation, status string, duration time.Duration)
	RecordORANInterfaceError(interfaceType, operation, errorType string)
	UpdateORANConnectionStatus(interfaceType, endpoint string, connected bool)
	UpdateORANPolicyInstances(policyType, status string, count int)

	// RAG metrics
	RecordRAGOperation(duration time.Duration, cacheHit bool)
	UpdateRAGDocumentsIndexed(count int)

	// GitOps metrics
	RecordGitOpsOperation(operation string, duration time.Duration, success bool)
	UpdateGitOpsSyncStatus(repository, branch string, inSync bool)

	// System metrics
	UpdateResourceUtilization(resourceType, unit string, value float64)
	UpdateWorkerQueueMetrics(queueName string, depth int, latency time.Duration)

	// Missing HTTP and streaming metrics methods
	RecordHTTPRequest(method, endpoint, status string, duration time.Duration)
	RecordSSEStream(endpoint string, connected bool)
	RecordLLMRequestError(model, errorType string)

	// Prometheus-style metrics getters
	GetGauge(name string) interface{}
	GetHistogram(name string) interface{}
	GetCounter(name string) interface{}
}

type HealthChecker interface {
	CheckHealth(ctx context.Context) (*ComponentHealth, error)
	GetName() string
	Start(ctx context.Context) error
	Stop() error
}

// NotificationChannel represents a notification delivery channel
type NotificationChannel struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Type    string            `json:"type"` // slack, email, webhook, pagerduty, teams
	Config  map[string]string `json:"config,omitempty"`
	Enabled bool              `json:"enabled"`
}

// TraceSpan represents a distributed tracing span
type TraceSpan struct {
	TraceID       string            `json:"trace_id"`
	SpanID        string            `json:"span_id"`
	ParentSpanID  string            `json:"parent_span_id,omitempty"`
	OperationName string            `json:"operation_name"`
	StartTime     time.Time         `json:"start_time"`
	EndTime       time.Time         `json:"end_time"`
	Duration      time.Duration     `json:"duration"`
	Tags          map[string]string `json:"tags,omitempty"`
	Status        string            `json:"status"`
	ServiceName   string            `json:"service_name,omitempty"`
}

// ComponentHealth represents the health status of a component
type ComponentHealth struct {
	Name        string            `json:"name"`
	Status      HealthStatus      `json:"status"`
	Message     string            `json:"message,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Details     interface{}       `json:"details,omitempty"`
	LastChecked time.Time         `json:"last_checked,omitempty"`
}

// MetricsData represents collected metrics data
type MetricsData struct {
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Namespace string                 `json:"namespace,omitempty"`
	Pod       string                 `json:"pod,omitempty"`
	Container string                 `json:"container,omitempty"`
	Metrics   map[string]string      `json:"metrics"`
	Labels    map[string]string      `json:"labels,omitempty"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}

// NWDAFAnalytics represents NWDAF analytics data
type NWDAFAnalytics struct {
	AnalyticsID   string                 `json:"analytics_id"`
	Type          NWDAFAnalyticsType     `json:"type"`
	AnalyticsType NWDAFAnalyticsType     `json:"analytics_type"`
	Timestamp     time.Time              `json:"timestamp"`
	Data          json.RawMessage `json:"data"`
	Consumer      string                 `json:"consumer"`
	Confidence    string                 `json:"confidence"`
	Validity      time.Duration          `json:"validity"`
}

// NWDAFAnalyticsType defines the type of NWDAF analytics
type NWDAFAnalyticsType string

const (
	NWDAFAnalyticsTypeLoad        NWDAFAnalyticsType = "load"
	NWDAFAnalyticsTypeLoadLevel   NWDAFAnalyticsType = "load_level"
	NWDAFAnalyticsTypeNFLoad      NWDAFAnalyticsType = "nf_load"
	NWDAFAnalyticsTypeServiceExp  NWDAFAnalyticsType = "service_exp"
	NWDAFAnalyticsTypeUEComm      NWDAFAnalyticsType = "ue_comm"
	NWDAFAnalyticsTypeNetworkPerf NWDAFAnalyticsType = "network_perf"
	NWDAFAnalyticsTypeUEMobility  NWDAFAnalyticsType = "ue_mobility"
	NWDAFAnalyticsTypeQoS         NWDAFAnalyticsType = "qos"
)

// PerformanceMetrics represents performance monitoring data
type PerformanceMetrics struct {
	ServiceName   string             `json:"service_name"`
	Component     string             `json:"component"`
	Timestamp     time.Time          `json:"timestamp"`
	ResponseTime  time.Duration      `json:"response_time"`
	Latency       time.Duration      `json:"latency"`
	Throughput    string             `json:"throughput"`
	ErrorRate     string             `json:"error_rate"`
	ResourceUsage map[string]string  `json:"resource_usage,omitempty"`
}

// SeasonalityDetector represents seasonal pattern detection
type SeasonalityDetector struct {
	Pattern    string        `json:"pattern"`
	Confidence float64       `json:"confidence"`
	Period     time.Duration `json:"period"`
	Amplitude  float64       `json:"amplitude"`
}

// DetectSeasonality detects seasonal patterns in the data
func (sd *SeasonalityDetector) DetectSeasonality(ctx context.Context, data interface{}) (*SeasonalityInfo, error) {
	// Stub implementation - would contain actual seasonality detection logic
	return &SeasonalityInfo{
		Pattern:    sd.Pattern,
		Period:     sd.Period,
		Confidence: sd.Confidence,
		Amplitude:  sd.Amplitude,
		Detected:   sd.Confidence > 0.5,
	}, nil
}

// SystemHealth represents overall system health
type SystemHealth struct {
	OverallStatus HealthStatus                `json:"overall_status"`
	Components    map[string]*ComponentHealth `json:"components"`
	LastUpdated   time.Time                   `json:"last_updated"`
	HealthScore   float64                     `json:"health_score"`
	Issues        []string                    `json:"issues,omitempty"`
}

// ReportConfig represents configuration for performance reports
type ReportConfig struct {
	Title           string            `json:"title"`
	Description     string            `json:"description"`
	TimeRange       TimeRange         `json:"time_range"`
	Metrics         []string          `json:"metrics"`
	Filters         map[string]string `json:"filters,omitempty"`
	Format          string            `json:"format"`
	OutputFormat    string            `json:"output_format"`
	Recipients      []string          `json:"recipients,omitempty"`
	ThresholdAlerts []AlertRule       `json:"threshold_alerts,omitempty"`
}

// PerformanceReport represents a performance analysis report
type PerformanceReport struct {
	ID              string                `json:"id"`
	Title           string                `json:"title"`
	GeneratedAt     time.Time             `json:"generated_at"`
	TimeRange       TimeRange             `json:"time_range"`
	Format          string                `json:"format"`
	Summary         *ReportSummary        `json:"summary"`
	Metrics         []*PerformanceMetrics `json:"metrics"`
	Alerts          []*AlertItem          `json:"alerts,omitempty"`
	Recommendations []string              `json:"recommendations,omitempty"`
}

// TimeRange represents a time range for queries
type TimeRange struct {
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
}

// AlertItem represents an alert in reports
type AlertItem struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Severity  AlertSeverity `json:"severity"`
	Status    string        `json:"status"`
	Timestamp time.Time     `json:"timestamp"`
	Message   string        `json:"message"`
	Source    string        `json:"source,omitempty"`
	Resolved  bool          `json:"resolved"`
}

// ReportSummary represents a summary of performance metrics
type ReportSummary struct {
	TotalMetrics     int                `json:"total_metrics"`
	AlertsTriggered  int                `json:"alerts_triggered"`
	PerformanceScore float64            `json:"performance_score"`
	AvgResponseTime  time.Duration      `json:"avg_response_time"`
	TotalRequests    int64              `json:"total_requests"`
	ErrorRate        float64            `json:"error_rate"`
	ResourceUsage    map[string]string  `json:"resource_usage,omitempty"`
}

// TrendAnalysis represents trend analysis results
type TrendAnalysis struct {
	Direction  string    `json:"direction"`  // "increasing", "decreasing", "stable"
	Strength   float64   `json:"strength"`   // 0-1 scale
	Confidence float64   `json:"confidence"` // 0-1 scale
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Slope      float64   `json:"slope"`
	R2Score    float64   `json:"r2_score"`
}

// PredictionResult represents prediction analysis results
type PredictionResult struct {
	Timestamp      time.Time          `json:"timestamp"`
	PredictedValue float64            `json:"predicted_value"`
	Confidence     float64            `json:"confidence"`
	LowerBound     float64            `json:"lower_bound"`
	UpperBound     float64            `json:"upper_bound"`
	PredictionType string             `json:"prediction_type"`
	ModelUsed      string             `json:"model_used"`
	Features       map[string]float64 `json:"features,omitempty"`
}

// AnomalyPoint represents an anomaly detection result
type AnomalyPoint struct {
	Timestamp     time.Time       `json:"timestamp"`
	Value         float64         `json:"value"`
	ExpectedValue float64         `json:"expected_value"`
	Deviation     float64         `json:"deviation"`
	Severity      float64         `json:"severity"`
	AnomalyScore  float64         `json:"anomaly_score"`
	Context       json.RawMessage `json:"context,omitempty"`
}

// PredictionPoint represents a single prediction data point
type PredictionPoint struct {
	Timestamp  time.Time `json:"timestamp"`
	Value      float64   `json:"value"`
	Confidence float64   `json:"confidence"`
	Source     string    `json:"source"`
	ModelName  string    `json:"model_name"`
}

// SeasonalityInfo represents seasonality detection results
type SeasonalityInfo struct {
	Pattern    string        `json:"pattern"`
	Period     time.Duration `json:"period"`
	Confidence float64       `json:"confidence"`
	Amplitude  float64       `json:"amplitude"`
	Phase      float64       `json:"phase,omitempty"`
	Detected   bool          `json:"detected"`
}