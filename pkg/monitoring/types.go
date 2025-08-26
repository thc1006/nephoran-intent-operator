// Package monitoring provides comprehensive monitoring capabilities for O-RAN Nephio systems
package monitoring

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// MonitoringManager coordinates all monitoring activities
type MonitoringManager struct {
	LatencyTracker   LatencyTracker
	ReportGenerator  ReportGenerator
	AvailabilityMonitor AvailabilityMonitor
	AlertManager     AlertManager
}

// LatencyTracker interface for tracking operation latencies
type LatencyTracker interface {
	StartOperation(ctx context.Context, operationID, operationType, component string, attrs map[string]string) (context.Context, error)
	EndOperation(ctx context.Context, operationID string, success bool, errorMsg string) error
	GetSpanFromContext(ctx context.Context) trace.Span
	TrackOperation(ctx context.Context, operationID string, fn func(context.Context) error) error
	RecordLatency(operationType string, latency time.Duration, success bool)
	StartPeriodicCollection(ctx context.Context, interval time.Duration)
}

// ReportGenerator interface for generating performance reports
type ReportGenerator interface {
	GenerateReport(ctx context.Context, config *ReportConfig) (*PerformanceReport, error)
	RecordMetrics(component string, metrics *PerformanceMetrics) error
	GetMetrics(component string) (*PerformanceMetrics, error)
	ExportReport(report *PerformanceReport, format string) ([]byte, error)
}

// AvailabilityMonitor interface for monitoring service availability
type AvailabilityMonitor interface {
	AddCheck(check *SyntheticCheck) error
	RemoveCheck(checkID string) error
	GetCheckResult(checkID string) (*CheckResult, error)
	StartMonitoring(ctx context.Context)
	Shutdown(ctx context.Context) error
}

// AlertManager interface for managing alerts
type AlertManager interface {
	AddRule(rule *AlertRule) error
	FireAlert(ctx context.Context, ruleID string, labels map[string]string, value float64) error
	GetAlert(alertID string) (*Alert, error)
	ListAlerts(status, component, severity string) []*Alert
}

// Common types used across monitoring components

// PerformanceMetrics holds performance data
type PerformanceMetrics struct {
	Component     string                 `json:"component"`
	Timestamp     time.Time             `json:"timestamp"`
	Throughput    float64               `json:"throughput"`
	Latency       time.Duration         `json:"latency"`
	ErrorRate     float64               `json:"error_rate"`
	Availability  float64               `json:"availability"`
	CustomMetrics map[string]interface{} `json:"custom_metrics"`
}

// PerformanceReport represents a generated performance report
type PerformanceReport struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	GeneratedAt time.Time         `json:"generated_at"`
	TimeRange   TimeRange         `json:"time_range"`
	Summary     ReportSummary     `json:"summary"`
	Metrics     []PerformanceMetrics `json:"metrics"`
	Alerts      []AlertItem       `json:"alerts"`
	Content     string            `json:"content"`
	Format      string            `json:"format"`
}

// TimeRange defines a time range for reports
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ReportSummary provides high-level report statistics
type ReportSummary struct {
	TotalComponents  int     `json:"total_components"`
	AvgThroughput   float64 `json:"avg_throughput"`
	AvgLatency      string  `json:"avg_latency"`
	OverallHealth   float64 `json:"overall_health"`
	CriticalAlerts  int     `json:"critical_alerts"`
}

// AlertItem represents a performance alert
type AlertItem struct {
	Component   string    `json:"component"`
	Metric      string    `json:"metric"`
	Value       float64   `json:"value"`
	Threshold   float64   `json:"threshold"`
	Severity    string    `json:"severity"`
	Timestamp   time.Time `json:"timestamp"`
	Description string    `json:"description"`
}

// ReportConfig holds configuration for report generation
type ReportConfig struct {
	IncludeMetrics   []string          `json:"include_metrics"`
	TimeRange        TimeRange         `json:"time_range"`
	OutputFormat     string            `json:"output_format"`
	CustomFields     map[string]string `json:"custom_fields"`
	ThresholdAlerts  map[string]float64 `json:"threshold_alerts"`
}

// SyntheticCheck defines a synthetic check configuration
type SyntheticCheck struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Type        string        `json:"type"` // http, tcp, grpc
	Target      string        `json:"target"`
	Interval    time.Duration `json:"interval"`
	Timeout     time.Duration `json:"timeout"`
	Enabled     bool          `json:"enabled"`
	Headers     map[string]string `json:"headers,omitempty"`
	ExpectedCode int           `json:"expected_code,omitempty"`
	ExpectedBody string        `json:"expected_body,omitempty"`
	ThresholdMs  int           `json:"threshold_ms"`
}

// CheckResult represents the result of a synthetic check
type CheckResult struct {
	CheckID     string        `json:"check_id"`
	Timestamp   time.Time     `json:"timestamp"`
	Success     bool          `json:"success"`
	ResponseTime time.Duration `json:"response_time"`
	StatusCode   int          `json:"status_code,omitempty"`
	Error        string       `json:"error,omitempty"`
	Message     string       `json:"message"`
	Availability float64      `json:"availability"`
}

// Alert represents an alert instance
type Alert struct {
	ID          string            `json:"id"`
	Rule        string            `json:"rule"`
	Component   string            `json:"component"`
	Severity    string            `json:"severity"`
	Status      string            `json:"status"` // firing, resolved, suppressed
	StartsAt    time.Time         `json:"starts_at"`
	EndsAt      *time.Time        `json:"ends_at,omitempty"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Message     string            `json:"message"`
	Description string            `json:"description"`
	Fingerprint string            `json:"fingerprint"`
	Silenced    bool              `json:"silenced"`
	AckBy       string            `json:"ack_by,omitempty"`
	AckAt       *time.Time        `json:"ack_at,omitempty"`
}

// AlertRule defines conditions for triggering alerts
type AlertRule struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Query       string            `json:"query"`
	Condition   string            `json:"condition"`
	Threshold   float64           `json:"threshold"`
	Duration    time.Duration     `json:"duration"`
	Severity    string            `json:"severity"`
	Component   string            `json:"component"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Channels    []string          `json:"channels"`
	Enabled     bool              `json:"enabled"`
	Cooldown    time.Duration     `json:"cooldown"`
	LastFired   *time.Time        `json:"last_fired,omitempty"`
}