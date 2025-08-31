// Package monitoring provides types and utilities for O-RAN monitoring and observability
package monitoring

import (
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

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// Metric represents a single metric with its metadata
type Metric struct {
	Name        string            `json:"name" yaml:"name"`
	Type        MetricType        `json:"type" yaml:"type"`
	Value       float64           `json:"value" yaml:"value"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Timestamp   time.Time         `json:"timestamp" yaml:"timestamp"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
}

// Alert represents a monitoring alert
type Alert struct {
	Name        string            `json:"name" yaml:"name"`
	Severity    AlertSeverity     `json:"severity" yaml:"severity"`
	Message     string            `json:"message" yaml:"message"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Timestamp   time.Time         `json:"timestamp" yaml:"timestamp"`
	Resolved    bool              `json:"resolved" yaml:"resolved"`
	ResolvedAt  *time.Time        `json:"resolvedAt,omitempty" yaml:"resolvedAt,omitempty"`
	Fingerprint string            `json:"fingerprint" yaml:"fingerprint"`
}

// HealthCheck represents a health check configuration
type HealthCheck struct {
	Name        string        `json:"name" yaml:"name"`
	Endpoint    string        `json:"endpoint" yaml:"endpoint"`
	Method      string        `json:"method" yaml:"method"`
	Interval    time.Duration `json:"interval" yaml:"interval"`
	Timeout     time.Duration `json:"timeout" yaml:"timeout"`
	HealthyCode int           `json:"healthyCode" yaml:"healthyCode"`
	Headers     []Header      `json:"headers,omitempty" yaml:"headers,omitempty"`
}

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
	QueryType    string   `json:"queryType,omitempty"`
	ExpectedIPs  []string `json:"expectedIPs,omitempty"`
	ExpectedCNAME string  `json:"expectedCNAME,omitempty"`
	Nameserver   string   `json:"nameserver,omitempty"`
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
	Name      string        `json:"name"`
	Condition string        `json:"condition"`
	Threshold float64       `json:"threshold"`
	Duration  metav1.Duration `json:"duration"`
	Severity  AlertSeverity `json:"severity"`
	Message   string        `json:"message"`
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
	SuccessRate float64 `json:"successRate,omitempty"`

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
	CheckID      string  `json:"check_id,omitempty"`
	Message      string  `json:"message,omitempty"`
	Availability float64 `json:"availability,omitempty"`

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

	// Extended fields from sla_monitoring_architecture.go
	IntentProcessingTests *IntentProcessingTestSuite `json:"intent_processing_tests,omitempty"`
	APIEndpointTests      *APIEndpointTestSuite      `json:"api_endpoint_tests,omitempty"`
	UserJourneyTests      *UserJourneyTestSuite      `json:"user_journey_tests,omitempty"`
	TestScheduler         *TestScheduler             `json:"test_scheduler,omitempty"`
	TestExecutor          *TestExecutor              `json:"test_executor,omitempty"`
	ResultProcessor       *TestResultProcessor       `json:"result_processor,omitempty"`
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

// Note: IntentProcessingTestSuite, APIEndpointTestSuite, UserJourneyTestSuite, 
// TestScheduler, TestExecutor, and TestResultProcessor are defined in 
// sla_monitoring_architecture.go to avoid duplicates

// Missing types that need to be defined
type AlertRule struct {
	Name        string            `json:"name"`
	Expression  string            `json:"expression"`
	Severity    AlertSeverity     `json:"severity"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type MetricsCollector interface {
	CollectMetrics() ([]*Metric, error)
	Start() error
	Stop() error
}

type HealthChecker interface {
	CheckHealth() (HealthStatus, error)
}

// NotificationChannel represents a notification delivery channel
type NotificationChannel struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Type    string            `json:"type"` // slack, email, webhook, pagerduty, teams
	Config  map[string]string `json:"config,omitempty"`
	Enabled bool              `json:"enabled"`
}