// Package monitoring provides comprehensive monitoring capabilities for O-RAN Nephio systems
package monitoring

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// MonitoringManager coordinates all monitoring activities
type MonitoringManager struct {
	LatencyTracker      LatencyTracker
	ReportGenerator     ReportGenerator
	AvailabilityMonitor AvailabilityMonitor
	AlertManager        AlertManager
}

// LatencyTrackerInterface interface for tracking operation latencies
type LatencyTrackerInterface interface {
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

// Common types used across monitoring components

// PerformanceMetrics holds performance data
type PerformanceMetrics struct {
	Component     string                 `json:"component"`
	Timestamp     time.Time              `json:"timestamp"`
	Throughput    float64                `json:"throughput"`
	Latency       time.Duration          `json:"latency"`
	ErrorRate     float64                `json:"error_rate"`
	Availability  float64                `json:"availability"`
	CustomMetrics map[string]interface{} `json:"custom_metrics"`
}

// PerformanceReport represents a generated performance report
type PerformanceReport struct {
	ID          string               `json:"id"`
	Title       string               `json:"title"`
	GeneratedAt time.Time            `json:"generated_at"`
	TimeRange   TimeRange            `json:"time_range"`
	Summary     ReportSummary        `json:"summary"`
	Metrics     []PerformanceMetrics `json:"metrics"`
	Alerts      []AlertItem          `json:"alerts"`
	Content     string               `json:"content"`
	Format      string               `json:"format"`
}

// TimeRange defines a time range for reports
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ReportSummary provides high-level report statistics
type ReportSummary struct {
	TotalComponents int     `json:"total_components"`
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
	IncludeMetrics  []string           `json:"include_metrics"`
	TimeRange       TimeRange          `json:"time_range"`
	OutputFormat    string             `json:"output_format"`
	CustomFields    map[string]string  `json:"custom_fields"`
	ThresholdAlerts map[string]float64 `json:"threshold_alerts"`
}

// SyntheticCheck defines a synthetic check configuration
type SyntheticCheck struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Type         string            `json:"type"` // http, tcp, grpc
	Target       string            `json:"target"`
	Interval     time.Duration     `json:"interval"`
	Timeout      time.Duration     `json:"timeout"`
	Enabled      bool              `json:"enabled"`
	Headers      map[string]string `json:"headers,omitempty"`
	ExpectedCode int               `json:"expected_code,omitempty"`
	ExpectedBody string            `json:"expected_body,omitempty"`
	ThresholdMs  int               `json:"threshold_ms"`
}

// CheckResult represents the result of a synthetic check
type CheckResult struct {
	CheckID      string        `json:"check_id"`
	Timestamp    time.Time     `json:"timestamp"`
	Success      bool          `json:"success"`
	ResponseTime time.Duration `json:"response_time"`
	StatusCode   int           `json:"status_code,omitempty"`
	Error        string        `json:"error,omitempty"`
	Message      string        `json:"message"`
	Availability float64       `json:"availability"`
}

// Alert type is defined in alerting.go

// Note: CircularBuffer is defined in sla_components.go to avoid duplication

// TrendAnalyzer provides trend analysis for time-series data
type TrendAnalyzer struct {
	window time.Duration
	mu     sync.RWMutex
}

// NewTrendAnalyzer creates a new trend analyzer
func NewTrendAnalyzer() *TrendAnalyzer {
	return &TrendAnalyzer{
		window: 24 * time.Hour,
	}
}

// SeasonalityDetector detects seasonal patterns in metrics
type SeasonalityDetector struct {
	patterns []SeasonalPattern
	mu       sync.RWMutex
}

// NewSeasonalityDetector creates a new seasonality detector
func NewSeasonalityDetector() *SeasonalityDetector {
	return &SeasonalityDetector{
		patterns: make([]SeasonalPattern, 0),
	}
}

// GetSeasonalAdjustment method is implemented in error_tracking.go

// SeasonalPattern represents a detected seasonal pattern
type SeasonalPattern struct {
	Period    time.Duration `json:"period"`
	Amplitude float64       `json:"amplitude"`
	Phase     float64       `json:"phase"`
	Strength  float64       `json:"strength"`
}

// PredictionHistory tracks prediction accuracy over time
type PredictionHistory struct {
	predictions  []HistoricalPrediction
	actualValues []float64
	accuracy     *AccuracyTracker
	mu           sync.RWMutex
}

// HistoricalPrediction represents a past prediction for accuracy tracking
type HistoricalPrediction struct {
	Timestamp       time.Time `json:"timestamp"`
	PredictedValue  float64   `json:"predicted_value"`
	ActualValue     float64   `json:"actual_value"`
	AbsoluteError   float64   `json:"absolute_error"`
	RelativeError   float64   `json:"relative_error"`
	Confidence      float64   `json:"confidence"`
	ModelVersion    string    `json:"model_version"`
}

// AccuracyTracker tracks model accuracy over different time periods
type AccuracyTracker struct {
	dailyAccuracy   *CircularBuffer
	weeklyAccuracy  *CircularBuffer
	monthlyAccuracy *CircularBuffer
	mu              sync.RWMutex
}

// Prediction represents a model prediction with confidence
type Prediction struct {
	Value      float64   `json:"value"`
	Confidence float64   `json:"confidence"`
	Timestamp  time.Time `json:"timestamp"`
	Features   []float64 `json:"features"`
}

// PredictedViolation represents a predicted SLA violation
type PredictedViolation struct {
	MetricType  string        `json:"metric_type"`
	Probability float64       `json:"probability"`
	Severity    string        `json:"severity"`
	TimeToEvent time.Duration `json:"time_to_event"`
	Impact      string        `json:"impact"`
	Mitigation  []string      `json:"mitigation"`
	Timestamp   time.Time     `json:"timestamp"`
}

// TrainingDataSet represents a dataset for model training
type TrainingDataSet struct {
	Features   [][]float64 `json:"features"`
	Targets    []float64   `json:"targets"`
	Timestamps []time.Time `json:"timestamps"`
	Weights    []float64   `json:"weights"`
}

// ModelInterface defines the common interface for ML models
type ModelInterface interface {
	Train(ctx context.Context, data *TrainingDataSet) error
	Predict(ctx context.Context, features []float64) (*Prediction, error)
	GetAccuracy() float64
}
