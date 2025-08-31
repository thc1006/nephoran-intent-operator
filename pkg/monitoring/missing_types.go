package monitoring

import "time"

// TraceSpan represents a distributed tracing span
type TraceSpan struct {
	TraceID     string                 `json:"trace_id"`
	SpanID      string                 `json:"span_id"`
	ParentID    string                 `json:"parent_id,omitempty"`
	OperationName string               `json:"operation_name"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     time.Time              `json:"end_time,omitempty"`
	Duration    time.Duration          `json:"duration"`
	Status      SpanStatus             `json:"status"`
	Tags        map[string]interface{} `json:"tags,omitempty"`
	Logs        []SpanLog              `json:"logs,omitempty"`
}

// SpanStatus represents the status of a tracing span
type SpanStatus int

const (
	SpanStatusUnknown SpanStatus = iota
	SpanStatusOK
	SpanStatusError
	SpanStatusTimeout
	SpanStatusCancelled
)

// SpanLog represents a log entry within a span
type SpanLog struct {
	Timestamp time.Time              `json:"timestamp"`
	Fields    map[string]interface{} `json:"fields"`
}

// ComponentHealth represents the health status of a system component
type ComponentHealth struct {
	Name        string                 `json:"name"`
	Status      HealthStatus           `json:"status"`
	LastChecked time.Time              `json:"last_checked"`
	Message     string                 `json:"message,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Metrics     map[string]float64     `json:"metrics,omitempty"`
}

// HealthStatus represents component health status
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// MetricsData represents time-series metrics data
type MetricsData struct {
	Timestamp time.Time              `json:"timestamp"`
	Value     float64                `json:"value"`
	Labels    map[string]string      `json:"labels,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ReportSummary represents a summary of a performance report
type ReportSummary struct {
	TotalMetrics   int                    `json:"total_metrics"`
	AverageValue   float64                `json:"average_value"`
	MaxValue       float64                `json:"max_value"`
	MinValue       float64                `json:"min_value"`
	TrendDirection string                 `json:"trend_direction"`
	KeyInsights    []string               `json:"key_insights,omitempty"`
	Recommendations []string              `json:"recommendations,omitempty"`
	Statistics     map[string]interface{} `json:"statistics,omitempty"`
}

// TrendAnalysis represents analysis of metric trends
type TrendAnalysis struct {
	Direction      string    `json:"direction"` // "up", "down", "stable"
	Confidence     float64   `json:"confidence"`
	ChangeRate     float64   `json:"change_rate"`
	Significance   string    `json:"significance"` // "high", "medium", "low"
	StartValue     float64   `json:"start_value"`
	EndValue       float64   `json:"end_value"`
	Analysis       string    `json:"analysis,omitempty"`
	RecommendedActions []string `json:"recommended_actions,omitempty"`
}

// PredictionResult represents the result of a prediction analysis
type PredictionResult struct {
	PredictedValue float64    `json:"predicted_value"`
	Confidence     float64    `json:"confidence"`
	PredictionTime time.Time  `json:"prediction_time"`
	UpperBound     float64    `json:"upper_bound"`
	LowerBound     float64    `json:"lower_bound"`
	ModelUsed      string     `json:"model_used"`
	Features       map[string]float64 `json:"features,omitempty"`
}

// AnomalyPoint represents an anomaly detection result
type AnomalyPoint struct {
	Timestamp      time.Time              `json:"timestamp"`
	Value          float64                `json:"value"`
	ExpectedValue  float64                `json:"expected_value"`
	AnomalyScore   float64                `json:"anomaly_score"`
	Severity       string                 `json:"severity"` // "low", "medium", "high"
	Description    string                 `json:"description,omitempty"`
	Context        map[string]interface{} `json:"context,omitempty"`
}

// PredictionPoint represents a point in a prediction series
type PredictionPoint struct {
	Time           time.Time `json:"time"`
	Value          float64   `json:"value"`
	UpperBound     float64   `json:"upper_bound"`
	LowerBound     float64   `json:"lower_bound"`
	Confidence     float64   `json:"confidence"`
}