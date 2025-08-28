package monitoringhealth

import (
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/health"
)

// EnhancedHealthResponse represents a comprehensive health check response
type EnhancedHealthResponse struct {
	Service       string        `json:"service"`
	Version       string        `json:"version"`
	Timestamp     time.Time     `json:"timestamp"`
	Context       HealthContext `json:"context"`
	OverallStatus health.Status `json:"overall_status"`
	WeightedScore float64       `json:"weighted_score"`

	// Check results
	Checks        []EnhancedCheck            `json:"checks"`
	TierSummaries map[HealthTier]TierSummary `json:"tier_summaries"`

	// Execution information
	ExecutionMetrics ExecutionMetrics `json:"execution_metrics"`

	// Additional metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// TierSummary provides aggregated information for a health tier
type TierSummary struct {
	Tier            HealthTier `json:"tier"`
	TotalChecks     int        `json:"total_checks"`
	HealthyChecks   int        `json:"healthy_checks"`
	UnhealthyChecks int        `json:"unhealthy_checks"`
	DegradedChecks  int        `json:"degraded_checks"`
	UnknownChecks   int        `json:"unknown_checks"`
	TotalWeight     float64    `json:"total_weight"`
	WeightedScore   float64    `json:"weighted_score"`
	AverageScore    float64    `json:"average_score"`
}

// ExecutionMetrics contains metrics about the health check execution
type ExecutionMetrics struct {
	StartTime         time.Time     `json:"start_time"`
	Duration          time.Duration `json:"duration"`
	CheckCount        int           `json:"check_count"`
	ParallelExecution bool          `json:"parallel_execution"`
	CacheHitRate      float64       `json:"cache_hit_rate,omitempty"`
}

// HealthTrend represents historical health trend data
type HealthTrend struct {
	CheckName   string             `json:"check_name"`
	TimeWindow  time.Duration      `json:"time_window"`
	DataPoints  []HealthDataPoint  `json:"data_points"`
	Trend       TrendDirection     `json:"trend"`
	Stability   StabilityLevel     `json:"stability"`
	Predictions []HealthPrediction `json:"predictions,omitempty"`
}

// HealthDataPoint represents a single data point in health history
type HealthDataPoint struct {
	Timestamp time.Time     `json:"timestamp"`
	Status    health.Status `json:"status"`
	Score     float64       `json:"score"`
	Duration  time.Duration `json:"duration"`
}

// TrendDirection indicates the direction of health trend
type TrendDirection string

const (
	TrendImproving TrendDirection = "improving"
	TrendDegrading TrendDirection = "degrading"
	TrendStable    TrendDirection = "stable"
	TrendUnknown   TrendDirection = "unknown"
	TrendVolatile  TrendDirection = "volatile"
)

// StabilityLevel indicates the stability of health metrics
type StabilityLevel string

const (
	StabilityHigh    StabilityLevel = "high"
	StabilityMedium  StabilityLevel = "medium"
	StabilityLow     StabilityLevel = "low"
	StabilityUnknown StabilityLevel = "unknown"
)

// HealthPrediction represents a predicted future health state
type HealthPrediction struct {
	PredictedTime   time.Time     `json:"predicted_time"`
	PredictedStatus health.Status `json:"predicted_status"`
	PredictedScore  float64       `json:"predicted_score"`
	Confidence      float64       `json:"confidence"`
	Reasoning       string        `json:"reasoning,omitempty"`
}

// HealthAlert represents a health-based alert
type HealthAlert struct {
	ID          string          `json:"id"`
	CheckName   string          `json:"check_name"`
	AlertType   HealthAlertType `json:"alert_type"`
	Severity    AlertSeverity   `json:"severity"`
	Status      AlertStatus     `json:"status"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Timestamp   time.Time       `json:"timestamp"`

	// Trigger conditions
	Threshold   float64 `json:"threshold,omitempty"`
	ActualValue float64 `json:"actual_value,omitempty"`

	// Context
	CurrentHealth *EnhancedCheck         `json:"current_health,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// HealthAlertType represents different types of health alerts
type HealthAlertType string

const (
	AlertTypeStatusChange        HealthAlertType = "status_change"
	AlertTypeScoreThreshold      HealthAlertType = "score_threshold"
	AlertTypeLatencyThreshold    HealthAlertType = "latency_threshold"
	AlertTypeConsecutiveFailures HealthAlertType = "consecutive_failures"
	AlertTypePredictiveFailure   HealthAlertType = "predictive_failure"
)

// AlertSeverity represents the severity of an alert

// AlertSeverity represents the severity of an alert
type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "info"
	SeverityWarning  AlertSeverity = "warning"
	SeverityError    AlertSeverity = "error"
	SeverityCritical AlertSeverity = "critical"
)

// AlertStatus represents the status of an alert
type AlertStatus string

const (
	AlertStatusActive     AlertStatus = "active"
	AlertStatusResolved   AlertStatus = "resolved"
	AlertStatusSuppressed AlertStatus = "suppressed"
)

// DependencyGraph represents health dependencies between components
type DependencyGraph struct {
	Nodes []DependencyNode `json:"nodes"`
	Edges []DependencyEdge `json:"edges"`
}

// DependencyNode represents a component in the dependency graph
type DependencyNode struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	Tier     HealthTier             `json:"tier"`
	Status   health.Status          `json:"status"`
	Score    float64                `json:"score"`
	Critical bool                   `json:"critical"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// DependencyEdge represents a dependency relationship
type DependencyEdge struct {
	Source   string                 `json:"source"`
	Target   string                 `json:"target"`
	Type     string                 `json:"type"`
	Weight   float64                `json:"weight"`
	Critical bool                   `json:"critical"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// HealthImpactAnalysis provides analysis of health issues and their impact
type HealthImpactAnalysis struct {
	CheckName          string              `json:"check_name"`
	CurrentStatus      health.Status       `json:"current_status"`
	ImpactLevel        ImpactLevel         `json:"impact_level"`
	AffectedComponents []string            `json:"affected_components"`
	RootCauses         []RootCause         `json:"root_causes"`
	RecommendedActions []RecommendedAction `json:"recommended_actions"`
	EstimatedRecovery  time.Duration       `json:"estimated_recovery"`
	BusinessImpact     BusinessImpact      `json:"business_impact"`
}

// ImpactLevel represents the level of impact from a health issue
type ImpactLevel string

const (
	ImpactNone     ImpactLevel = "none"
	ImpactLow      ImpactLevel = "low"
	ImpactMedium   ImpactLevel = "medium"
	ImpactHigh     ImpactLevel = "high"
	ImpactCritical ImpactLevel = "critical"
)

// RootCause represents a potential root cause of health issues
type RootCause struct {
	Category    string   `json:"category"`
	Description string   `json:"description"`
	Confidence  float64  `json:"confidence"`
	Evidence    []string `json:"evidence"`
}

// RecommendedAction represents a recommended action to address health issues
type RecommendedAction struct {
	Action      string         `json:"action"`
	Priority    ActionPriority `json:"priority"`
	Description string         `json:"description"`
	Automated   bool           `json:"automated"`
	ETA         time.Duration  `json:"eta,omitempty"`
}

// ActionPriority represents the priority of recommended actions
type ActionPriority string

const (
	PriorityImmediate ActionPriority = "immediate"
	PriorityUrgent    ActionPriority = "urgent"
	PriorityNormal    ActionPriority = "normal"
	PriorityLow       ActionPriority = "low"
)

// BusinessImpact represents the business impact of health issues
type BusinessImpact struct {
	Level             ImpactLevel   `json:"level"`
	Description       string        `json:"description"`
	AffectedServices  []string      `json:"affected_services"`
	EstimatedDowntime time.Duration `json:"estimated_downtime,omitempty"`
	EstimatedCost     float64       `json:"estimated_cost,omitempty"`
	UserImpact        string        `json:"user_impact,omitempty"`
}

// HealthMetricsSnapshot represents a snapshot of health metrics at a point in time
type HealthMetricsSnapshot struct {
	Timestamp       time.Time              `json:"timestamp"`
	OverallScore    float64                `json:"overall_score"`
	OverallStatus   health.Status          `json:"overall_status"`
	TierScores      map[HealthTier]float64 `json:"tier_scores"`
	ComponentScores map[string]float64     `json:"component_scores"`
	FailureRate     float64                `json:"failure_rate"`
	AverageLatency  time.Duration          `json:"average_latency"`
	ActiveAlerts    int                    `json:"active_alerts"`
	CriticalIssues  int                    `json:"critical_issues"`
}

// HealthConfiguration represents the configuration for the enhanced health system
type HealthConfiguration struct {
	// Global settings
	DefaultTimeout  time.Duration `json:"default_timeout"`
	DefaultInterval time.Duration `json:"default_interval"`
	WorkerCount     int           `json:"worker_count"`

	// Caching
	CacheEnabled bool          `json:"cache_enabled"`
	CacheExpiry  time.Duration `json:"cache_expiry"`

	// Scoring
	LatencyPenaltyEnabled bool               `json:"latency_penalty_enabled"`
	LatencyThresholds     []LatencyThreshold `json:"latency_thresholds"`

	// History and trends
	HistoryRetention     time.Duration `json:"history_retention"`
	HistoryMaxPoints     int           `json:"history_max_points"`
	TrendAnalysisEnabled bool          `json:"trend_analysis_enabled"`

	// Alerting
	AlertingEnabled bool             `json:"alerting_enabled"`
	AlertThresholds []AlertThreshold `json:"alert_thresholds"`

	// Context-specific overrides
	ContextOverrides map[HealthContext]ContextOverride `json:"context_overrides"`
}

// LatencyThreshold represents a latency threshold configuration
type LatencyThreshold struct {
	Threshold time.Duration `json:"threshold"`
	Penalty   float64       `json:"penalty"`
}

// AlertThreshold represents an alert threshold configuration
type AlertThreshold struct {
	CheckName string        `json:"check_name"`
	Tier      HealthTier    `json:"tier"`
	Metric    string        `json:"metric"`
	Threshold float64       `json:"threshold"`
	Operator  string        `json:"operator"` // "gt", "lt", "eq", "ne"
	Severity  AlertSeverity `json:"severity"`
}

// ContextOverride represents context-specific configuration overrides
type ContextOverride struct {
	Timeout          time.Duration      `json:"timeout"`
	Interval         time.Duration      `json:"interval"`
	WeightAdjustment map[string]float64 `json:"weight_adjustment"`
}
