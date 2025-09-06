package test_validation

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// Missing types required by sla_validation_test.go

// DowntimeEvent represents a downtime incident
type DowntimeEvent struct {
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
	Cause     string        `json:"cause"`
	Impact    string        `json:"impact"`
}

// CrossValidationResult contains cross-validation analysis results
type CrossValidationResult struct {
	ConsistencyScore  float64  `json:"consistency_score"`
	AgreementRate     float64  `json:"agreement_rate"`
	MaxDeviation      float64  `json:"max_deviation"`
	ValidationMethods []string `json:"validation_methods"`
}

// IndependentMeasurement represents a measurement from an independent method
type IndependentMeasurement struct {
	Method    string    `json:"method"`
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
	Quality   float64   `json:"quality"`
}

// LatencyDistribution contains latency distribution analysis
type LatencyDistribution struct {
	P50    float64 `json:"p50"`
	P90    float64 `json:"p90"`
	P95    float64 `json:"p95"`
	P99    float64 `json:"p99"`
	P99_9  float64 `json:"p99_9"`
	Mean   float64 `json:"mean"`
	StdDev float64 `json:"std_dev"`
}

// LatencyStats contains detailed latency statistics for a component
type LatencyStats struct {
	Mean   float64 `json:"mean"`
	P95    float64 `json:"p95"`
	P99    float64 `json:"p99"`
	Count  int64   `json:"count"`
	StdDev float64 `json:"std_dev"`
}

// TailLatencyAnalysis contains analysis of tail latencies
type TailLatencyAnalysis struct {
	TailLatency float64 `json:"tail_latency"`
	Outliers    int     `json:"outliers"`
	Severity    string  `json:"severity"`
}

// LatencyTrends contains temporal latency trend analysis
type LatencyTrends struct {
	Trend      string  `json:"trend"` // "increasing", "decreasing", "stable"
	Slope      float64 `json:"slope"`
	R2         float64 `json:"r2"`
	Confidence float64 `json:"confidence"`
}

// LatencyEvent represents a specific latency event
type LatencyEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Latency   float64   `json:"latency"`
	Component string    `json:"component"`
	Severity  string    `json:"severity"`
}

// BottleneckAnalysis contains bottleneck identification results
type BottleneckAnalysis struct {
	PrimaryBottleneck  string             `json:"primary_bottleneck"`
	BottleneckSeverity float64            `json:"bottleneck_severity"`
	ComponentContrib   map[string]float64 `json:"component_contrib"`
	Recommendations    []string           `json:"recommendations"`
}

// ScalabilityMetrics contains scalability analysis results
type ScalabilityMetrics struct {
	LinearityScore   float64 `json:"linearity_score"`
	BreakingPoint    float64 `json:"breaking_point"`
	EfficiencyFactor float64 `json:"efficiency_factor"`
	RecommendedLimit float64 `json:"recommended_limit"`
}

// StatisticalSummary contains overall statistical summary
type StatisticalSummary struct {
	TotalSamples     int64   `json:"total_samples"`
	ValidSamples     int64   `json:"valid_samples"`
	OutlierCount     int     `json:"outlier_count"`
	ConfidenceLevel  float64 `json:"confidence_level"`
	StatisticalPower float64 `json:"statistical_power"`
}

// QualityAssessment contains data quality assessment
type QualityAssessment struct {
	DataQualityScore float64 `json:"data_quality_score"`
	CompletenessRate float64 `json:"completeness_rate"`
	AccuracyScore    float64 `json:"accuracy_score"`
	ReliabilityScore float64 `json:"reliability_score"`
}

// EvidencePackage contains all validation evidence
// Note: ValidationEvidence is defined in sla_validation_test.go
type EvidencePackage struct {
	MetricData        []interface{} `json:"metric_data"`
	LogFiles          []interface{} `json:"log_files"`
	TraceData         []interface{} `json:"trace_data"`
	Screenshots       []interface{} `json:"screenshots"`
	ConfigSnapshots   []interface{} `json:"config_snapshots"`
	ValidationReports []interface{} `json:"validation_reports"`
}

// Additional helper types for missing functionality

// P95Analysis contains P95 latency analysis
type P95Analysis struct {
	Value              float64     `json:"value"`
	ConfidenceInterval interface{} `json:"confidence_interval"` // Will be *ConfidenceInterval from main test file
	SampleSize         int         `json:"sample_size"`
	ValidationMethod   string      `json:"validation_method"`
}

// SustainedThroughput contains sustained throughput analysis
type SustainedThroughput struct {
	Value              float64       `json:"value"`
	Duration           time.Duration `json:"duration"`
	ConfidenceInterval interface{}   `json:"confidence_interval"` // Will be *ConfidenceInterval from main test file
	ValidationMethod   string        `json:"validation_method"`
}

// ErrorBudget contains error budget calculations
type ErrorBudget struct {
	Percentage          float64 `json:"percentage"`
	MinutesPerMonth     float64 `json:"minutes_per_month"`
	ConsumedPercentage  float64 `json:"consumed_percentage"`
	RemainingPercentage float64 `json:"remaining_percentage"`
}

// ErrorBudgetMeasurement contains measured error budget data
type ErrorBudgetMeasurement struct {
	ConsumedPercentage  float64       `json:"consumed_percentage"`
	RemainingPercentage float64       `json:"remaining_percentage"`
	TotalDowntime       time.Duration `json:"total_downtime"`
	MeasurementPeriod   time.Duration `json:"measurement_period"`
}

// AvailabilityScore contains availability scoring
type AvailabilityScore struct {
	Score      float64            `json:"score"`
	Components map[string]float64 `json:"components"`
	Timestamp  time.Time          `json:"timestamp"`
}

// LatencyScore contains latency scoring
type LatencyScore struct {
	Score      float64            `json:"score"`
	Components map[string]float64 `json:"components"`
	Timestamp  time.Time          `json:"timestamp"`
}

// ThroughputScore contains throughput scoring
type ThroughputScore struct {
	Score      float64            `json:"score"`
	Components map[string]float64 `json:"components"`
	Timestamp  time.Time          `json:"timestamp"`
}

// Additional types for SLA functionality - removed duplicates as they exist in validator_interfaces.go

// Core types needed by sla_methods.go

// SLAValidationTestSuite is the main test suite (forward declaration)
// The actual implementation is in sla_validation_test.go
// SLAValidationTestSuite is defined in sla_validation_test.go

// SLAValidationConfig placeholder for compilation
type SLAValidationConfig struct {
	AvailabilityClaim    float64       `json:"availability_claim"`
	AvailabilityAccuracy float64       `json:"availability_accuracy"`
	LatencyP95Claim      time.Duration `json:"latency_p95_claim"`
	ThroughputClaim      float64       `json:"throughput_claim"`
	ThroughputAccuracy   float64       `json:"throughput_accuracy"`
	ConfidenceLevel      float64       `json:"confidence_level"`
}

// MeasurementSet contains a collection of measurements for validation
type MeasurementSet struct {
	Measurements   []float64 `json:"measurements"`
	Timestamps     []int64   `json:"timestamps"`
	Labels         []string  `json:"labels,omitempty"`
	AggregatedData json.RawMessage `json:"aggregated_data,omitempty"`
	
	// Statistical properties (calculated by calculateMeasurementStatistics)
	Mean        float64         `json:"mean"`
	Median      float64         `json:"median"`
	StdDev      float64         `json:"std_dev"`
	Min         float64         `json:"min"`
	Max         float64         `json:"max"`
	Percentiles map[int]float64 `json:"percentiles"`

	// Quality metrics
	OutlierCount int     `json:"outlier_count"`
	MissingData  int     `json:"missing_data"`
	QualityScore float64 `json:"quality_score"`
}

// StatisticalAnalyzer performs statistical analysis on measurement data
type StatisticalAnalyzer struct {
	Config          *AnalyzerConfig `json:"config,omitempty"`
	confidenceLevel float64         `json:"confidence_level"`
}

// StatisticalAnalysis contains statistical analysis results
type StatisticalAnalysis struct {
	Mean         float64             `json:"mean"`
	StdDev       float64             `json:"std_dev"`
	Median       float64             `json:"median"`
	Mode         float64             `json:"mode"`
	Percentiles  map[string]float64  `json:"percentiles"`
	Distribution string              `json:"distribution"`
	Confidence   *ConfidenceInterval `json:"confidence"`
}

// ConfidenceInterval represents a statistical confidence interval
type ConfidenceInterval struct {
	Lower      float64 `json:"lower"`
	Upper      float64 `json:"upper"`
	Level      float64 `json:"level"`
	MarginOfError float64 `json:"margin_of_error"`
}

// AnalyzerConfig holds configuration for statistical analysis
type AnalyzerConfig struct {
	ConfidenceLevel  float64 `json:"confidence_level"`
	SignificanceLevel float64 `json:"significance_level"`
	SampleSize       int     `json:"sample_size"`
	Method           string  `json:"method"`
}

// ClaimVerifier verifies SLA claims against measurements
type ClaimVerifier struct {
	Config *VerifierConfig         `json:"config,omitempty"`
	claims map[string]*SLAClaim    `json:"claims,omitempty"`
	mutex  sync.RWMutex           `json:"-"`
}

// ClaimVerification contains claim verification results
type ClaimVerification struct {
	Claim     string  `json:"claim"`
	Verified  bool    `json:"verified"`
	Evidence  string  `json:"evidence"`
	Score     float64 `json:"score"`
	Deviation float64 `json:"deviation"`
}

// VerifierConfig holds configuration for claim verification
type VerifierConfig struct {
	Threshold   float64 `json:"threshold"`
	Method      string  `json:"method"`
	StrictMode  bool    `json:"strict_mode"`
}

// SLAClaim represents an SLA claim to be validated
type SLAClaim struct {
	Type        string  `json:"type"`
	Metric      string  `json:"metric"`
	Target      float64 `json:"target"`
	Threshold   float64 `json:"threshold"`
	Description string  `json:"description"`
}

// Mock implementations for testing
type mockAvailabilityValidator struct{}

func (m *mockAvailabilityValidator) ValidateAvailability(ctx context.Context) (*MeasurementSet, error) {
	return &MeasurementSet{}, nil
}

type mockLatencyValidator struct{}

func (m *mockLatencyValidator) ValidateLatency(ctx context.Context) (*MeasurementSet, error) {
	return &MeasurementSet{}, nil
}

type mockThroughputValidator struct{}

func (m *mockThroughputValidator) ValidateThroughput(ctx context.Context) (*MeasurementSet, error) {
	return &MeasurementSet{}, nil
}
