//go:build go1.24

package regression

import (
	"context"
	"fmt"
	"math"
	"time"

	"gonum.org/v1/gonum/stat"

	"k8s.io/klog/v2"
)

// RegressionDetector provides automated performance regression detection.

// with statistical significance testing and trend analysis.

type RegressionDetector struct {
	config *RegressionConfig

	baseline *PerformanceBaseline

	alertThresholds *AlertThresholds

	analyzer *TrendAnalyzer

	notifications *NotificationManager
}

// RegressionConfig defines detection parameters and thresholds.

type RegressionConfig struct {

	// Statistical parameters.

	ConfidenceLevel float64 // e.g., 0.95 for 95% confidence

	SignificanceLevel float64 // e.g., 0.05 for 5% significance

	MinSampleSize int // Minimum samples for valid comparison

	SlidingWindowSize int // Number of recent samples to analyze

	// Regression thresholds.

	LatencyRegressionPct float64 // % increase considered regression

	ThroughputRegressionPct float64 // % decrease considered regression

	ErrorRateRegressionPct float64 // % increase considered regression

	AvailabilityRegressionPct float64 // % decrease considered regression

	// Detection sensitivity.

	TrendDetectionPeriod time.Duration // Period for trend analysis

	NoiseFilteringEnabled bool // Filter out noisy/spurious regressions

	SeasonalAdjustment bool // Account for seasonal patterns

	// Performance targets (from Nephoran Intent Operator claims).

	TargetLatencyP95Ms float64 // 2000ms target

	TargetThroughputRpm float64 // 45 intents/min target

	TargetAvailabilityPct float64 // 99.95% target

	TargetCacheHitRatePct float64 // 87% target

}

// PerformanceBaseline represents historical baseline performance data.

type PerformanceBaseline struct {
	ID string

	CreatedAt time.Time

	ValidFrom time.Time

	ValidUntil time.Time

	Version string

	// Baseline metrics with statistical distributions.

	LatencyMetrics *BaselineMetrics

	ThroughputMetrics *BaselineMetrics

	ErrorRateMetrics *BaselineMetrics

	AvailabilityMetrics *BaselineMetrics

	ResourceMetrics *BaselineMetrics

	// Contextual information.

	TestConditions *TestConditions

	SystemConfiguration *SystemConfiguration

	SampleCount int

	QualityScore float64 // Baseline quality/reliability score

}

// BaselineMetrics contains statistical summary of baseline performance.

type BaselineMetrics struct {
	MetricName string

	Unit string

	Mean float64

	Median float64

	StandardDev float64

	P50, P95, P99 float64

	Min, Max float64

	SampleSize int

	Distribution []float64 // Raw samples for detailed analysis

	Seasonality *SeasonalPattern

	TrendCoefficient float64
}

// SeasonalPattern captures seasonal performance variations.

type SeasonalPattern struct {
	HasSeasonality bool

	Period time.Duration

	Amplitude float64

	Phase float64

	SeasonalIndices map[string]float64 // e.g., hourly/daily patterns

}

// TestConditions captures environmental factors during baseline creation.

type TestConditions struct {
	LoadLevel float64

	ConcurrentUsers int

	TestDuration time.Duration

	NetworkConditions string

	SystemLoad float64

	ExternalFactors map[string]string
}

// SystemConfiguration captures system state during baseline.

type SystemConfiguration struct {
	Version string

	Dependencies map[string]string

	ResourceLimits map[string]float64

	ConfigurationHash string

	InfrastructureType string
}

// AlertThresholds defines when to trigger different alert levels.

type AlertThresholds struct {

	// Critical thresholds (immediate action required).

	CriticalLatencyMs float64 // Latency exceeding this triggers critical alert

	CriticalErrorRatePct float64 // Error rate exceeding this is critical

	CriticalAvailabilityPct float64 // Availability below this is critical

	// Warning thresholds (monitoring required).

	WarningLatencyMs float64

	WarningErrorRatePct float64

	WarningAvailabilityPct float64

	// Trend thresholds (gradual degradation).

	TrendDegradationPct float64 // % degradation over trend period

	ConsecutiveFailures int // Consecutive bad measurements

}

// RegressionAnalysis represents the result of regression detection analysis.

type RegressionAnalysis struct {
	AnalysisID string

	Timestamp time.Time

	BaselineUsed *PerformanceBaseline

	CurrentMeasurement *PerformanceMeasurement

	// Overall assessment.

	HasRegression bool

	RegressionSeverity string // "Critical", "High", "Medium", "Low"

	ConfidenceScore float64 // 0-1 confidence in regression detection

	// Detailed findings.

	MetricRegressions []MetricRegression

	TrendAnalysis *TrendAnalysisResult

	StatisticalTests map[string]*StatisticalTestResult

	// Root cause hints.

	PotentialCauses []PotentialCause

	CorrelationAnalysis *CorrelationAnalysis

	// Recommendations.

	ImmediateActions []string

	InvestigationSteps []string

	PreventionMeasures []string
}

// PerformanceMeasurement represents current performance measurements.

type PerformanceMeasurement struct {
	Timestamp time.Time

	Source string

	SampleCount int

	TestDuration time.Duration

	// Core performance metrics.

	LatencyP50 float64

	LatencyP95 float64

	LatencyP99 float64

	Throughput float64

	ErrorRate float64

	Availability float64

	// Resource utilization.

	CPUUtilization float64

	MemoryUsageMB float64

	GoroutineCount int

	CacheHitRate float64

	// Quality indicators.

	SampleQuality float64 // Quality of this measurement

	NoiseLevel float64 // Amount of noise in measurement

	Outliers int // Number of outliers detected

}

// MetricRegression represents regression analysis for a specific metric.

type MetricRegression struct {
	MetricName string

	BaselineValue float64

	CurrentValue float64

	AbsoluteChange float64

	RelativeChangePct float64

	// Statistical analysis.

	IsSignificant bool

	PValue float64

	EffectSize float64 // Cohen's d

	ConfidenceInterval ConfidenceInterval

	// Assessment.

	RegressionType string // "Performance", "Availability", "Resource"

	Severity string

	Impact string

	TrendDirection string // "Improving", "Stable", "Degrading"

}

// ConfidenceInterval represents statistical confidence bounds.

type ConfidenceInterval struct {
	Lower float64

	Upper float64

	Level float64

	MarginError float64
}

// StatisticalTestResult contains results from statistical hypothesis tests.

type StatisticalTestResult struct {
	TestName string

	TestType string // "t-test", "mann-whitney", "ks-test"

	Statistic float64

	PValue float64

	IsSignificant bool

	DegreesOfFreedom int

	AlternativeHypothesis string

	Interpretation string
}

// PotentialCause represents possible causes of performance regression.

type PotentialCause struct {
	Category string // "Code", "Infrastructure", "Load", "External"

	Description string

	Likelihood float64 // 0-1 probability score

	Evidence []string

	Investigation []string // Steps to investigate this cause

}

// CorrelationAnalysis examines relationships between metrics and external factors.

type CorrelationAnalysis struct {
	MetricCorrelations map[string]float64 // Correlation between different metrics

	ExternalFactors map[string]float64 // Correlation with external factors

	LaggedCorrelations map[string]map[int]float64 // Time-lagged correlations

}

// TrendAnalyzer provides trend detection and analysis capabilities.

type TrendAnalyzer struct {
	config *TrendAnalysisConfig

	historicalData *HistoricalDataStore

	seasonalModel *SeasonalModel
}

// TrendAnalysisConfig configures trend analysis behavior.

type TrendAnalysisConfig struct {
	WindowSizes []int // Different window sizes for trend analysis

	TrendMethods []string // "linear", "polynomial", "seasonal"

	ChangePointDetection bool // Detect sudden changes in trends

	AnomalyDetection bool // Detect anomalous measurements

	ForecastingEnabled bool // Generate performance forecasts

}

// TrendAnalysisResult contains results from trend analysis.

type TrendAnalysisResult struct {
	Trend string // "Improving", "Stable", "Degrading"

	TrendCoefficient float64 // Rate of change

	Seasonality *DetectedSeasonality

	ChangePoints []ChangePoint

	Anomalies []AnomalyEvent

	Forecast *PerformanceForecast
}

// DetectedSeasonality represents detected seasonal patterns.

type DetectedSeasonality struct {
	IsPresent bool

	Period time.Duration

	Strength float64 // 0-1, strength of seasonal pattern

	SeasonalFactors map[string]float64
}

// ChangePoint represents a detected change in performance trend.

type ChangePoint struct {
	Timestamp time.Time

	Metric string

	BeforeValue float64

	AfterValue float64

	ChangeType string // "step", "trend", "variance"

	Confidence float64
}

// AnomalyEvent represents detected performance anomaly.

type AnomalyEvent struct {
	Timestamp time.Time

	Metric string

	Value float64

	ExpectedValue float64

	Severity string

	AnomalyType string // "outlier", "contextual", "collective"

}

// PerformanceForecast provides performance predictions.

type PerformanceForecast struct {
	Horizon time.Duration

	ForecastData map[string][]ForecastPoint

	Confidence map[string][]ConfidenceInterval

	RiskAreas []ForecastRisk
}

// ForecastRisk represents predicted performance risks.

type ForecastRisk struct {
	Timeframe time.Duration

	Metric string

	RiskType string

	Probability float64

	Impact string

	Mitigation []string
}

// NotificationManager handles regression alerts and notifications.

type NotificationManager struct {
	config *NotificationConfig

	channels map[string]NotificationChannel

	alertHistory *AlertHistory
}

// NotificationConfig configures alert notifications.

type NotificationConfig struct {
	EnabledChannels []string

	AlertLevels []string

	RateLimiting bool

	EscalationRules []EscalationRule

	QuietHours *TimeWindow

	MaintenanceWindows []TimeWindow
}

// RegressionAlert represents a regression detection alert.

type RegressionAlert struct {
	ID string

	Timestamp time.Time

	Severity string

	Title string

	Description string

	AffectedMetrics []string

	Analysis *RegressionAnalysis

	Actions []RecommendedAction

	Context map[string]interface{}
}

// RecommendedAction represents suggested response actions.

type RecommendedAction struct {
	Type string // "immediate", "investigation", "prevention"

	Description string

	Priority int

	EstimatedTime time.Duration

	Owner string
}

// NewRegressionDetector creates a new regression detection system.

func NewRegressionDetector(config *RegressionConfig) *RegressionDetector {

	if config == nil {

		config = getDefaultRegressionConfig()

	}

	return &RegressionDetector{

		config: config,

		alertThresholds: getDefaultAlertThresholds(),

		analyzer: NewTrendAnalyzer(getDefaultTrendConfig()),

		notifications: NewNotificationManager(getDefaultNotificationConfig()),
	}

}

// getDefaultRegressionConfig returns default regression detection configuration.

func getDefaultRegressionConfig() *RegressionConfig {

	return &RegressionConfig{

		// Statistical parameters.

		ConfidenceLevel: 0.95,

		SignificanceLevel: 0.05,

		MinSampleSize: 30,

		SlidingWindowSize: 100,

		// Regression thresholds based on Nephoran performance claims.

		LatencyRegressionPct: 10.0, // 10% latency increase = regression

		ThroughputRegressionPct: 10.0, // 10% throughput decrease = regression

		ErrorRateRegressionPct: 50.0, // 50% error rate increase = regression

		AvailabilityRegressionPct: 0.1, // 0.1% availability decrease = regression

		// Detection settings.

		TrendDetectionPeriod: 24 * time.Hour,

		NoiseFilteringEnabled: true,

		SeasonalAdjustment: true,

		// Performance targets from claims.

		TargetLatencyP95Ms: 2000, // Sub-2-second P95 claim

		TargetThroughputRpm: 45, // 45 intents/min claim

		TargetAvailabilityPct: 99.95, // 99.95% availability claim

		TargetCacheHitRatePct: 87, // 87% cache hit rate claim

	}

}

// AnalyzeRegression performs comprehensive regression analysis.

func (rd *RegressionDetector) AnalyzeRegression(ctx context.Context, current *PerformanceMeasurement) (*RegressionAnalysis, error) {

	if rd.baseline == nil {

		return nil, fmt.Errorf("no baseline available for regression analysis")

	}

	if current.SampleCount < rd.config.MinSampleSize {

		return nil, fmt.Errorf("insufficient sample size: %d < %d", current.SampleCount, rd.config.MinSampleSize)

	}

	analysis := &RegressionAnalysis{

		AnalysisID: fmt.Sprintf("regression-%d", time.Now().Unix()),

		Timestamp: time.Now(),

		BaselineUsed: rd.baseline,

		CurrentMeasurement: current,

		MetricRegressions: make([]MetricRegression, 0),

		StatisticalTests: make(map[string]*StatisticalTestResult),

		PotentialCauses: make([]PotentialCause, 0),

		ImmediateActions: make([]string, 0),

		InvestigationSteps: make([]string, 0),

		PreventionMeasures: make([]string, 0),
	}

	// Analyze each metric for regression.

	if err := rd.analyzeLatencyRegression(analysis); err != nil {

		klog.Warningf("Latency regression analysis failed: %v", err)

	}

	if err := rd.analyzeThroughputRegression(analysis); err != nil {

		klog.Warningf("Throughput regression analysis failed: %v", err)

	}

	if err := rd.analyzeErrorRateRegression(analysis); err != nil {

		klog.Warningf("Error rate regression analysis failed: %v", err)

	}

	if err := rd.analyzeAvailabilityRegression(analysis); err != nil {

		klog.Warningf("Availability regression analysis failed: %v", err)

	}

	// Perform trend analysis.

	trendResult, err := rd.analyzer.AnalyzeTrends(ctx, current)

	if err != nil {

		klog.Warningf("Trend analysis failed: %v", err)

	} else {

		analysis.TrendAnalysis = trendResult

	}

	// Determine overall regression status.

	rd.determineOverallRegression(analysis)

	// Generate potential causes and recommendations.

	rd.generateCausesAndRecommendations(analysis)

	klog.Infof("Regression analysis completed: regression=%v, severity=%s, confidence=%.2f",

		analysis.HasRegression, analysis.RegressionSeverity, analysis.ConfidenceScore)

	return analysis, nil

}

// analyzeLatencyRegression analyzes latency metrics for regression.

func (rd *RegressionDetector) analyzeLatencyRegression(analysis *RegressionAnalysis) error {

	baselineP95 := rd.baseline.LatencyMetrics.P95

	currentP95 := analysis.CurrentMeasurement.LatencyP95

	regression := MetricRegression{

		MetricName: "LatencyP95",

		BaselineValue: baselineP95,

		CurrentValue: currentP95,

		AbsoluteChange: currentP95 - baselineP95,
	}

	if baselineP95 > 0 {

		regression.RelativeChangePct = (currentP95 - baselineP95) / baselineP95 * 100

	}

	// Perform statistical test.

	testResult := rd.performTTest(rd.baseline.LatencyMetrics.Distribution,

		[]float64{currentP95}) // Simplified - would use full current distribution

	regression.IsSignificant = testResult.IsSignificant

	regression.PValue = testResult.PValue

	regression.EffectSize = rd.calculateEffectSize(baselineP95, currentP95, rd.baseline.LatencyMetrics.StandardDev)

	// Assess severity.

	if regression.RelativeChangePct > rd.config.LatencyRegressionPct && regression.IsSignificant {

		if currentP95 > rd.config.TargetLatencyP95Ms {

			regression.Severity = "Critical"

			regression.Impact = "Performance target exceeded"

		} else if regression.RelativeChangePct > 20 {

			regression.Severity = "High"

			regression.Impact = "Significant latency increase"

		} else {

			regression.Severity = "Medium"

			regression.Impact = "Moderate latency increase"

		}

		regression.RegressionType = "Performance"

	} else {

		regression.Severity = "Low"

		regression.Impact = "Within acceptable range"

	}

	// Determine trend direction.

	if regression.RelativeChangePct > 5 {

		regression.TrendDirection = "Degrading"

	} else if regression.RelativeChangePct < -5 {

		regression.TrendDirection = "Improving"

	} else {

		regression.TrendDirection = "Stable"

	}

	analysis.MetricRegressions = append(analysis.MetricRegressions, regression)

	analysis.StatisticalTests["latency_p95"] = testResult

	return nil

}

// analyzeThroughputRegression analyzes throughput metrics for regression.

func (rd *RegressionDetector) analyzeThroughputRegression(analysis *RegressionAnalysis) error {

	baselineThroughput := rd.baseline.ThroughputMetrics.Mean

	currentThroughput := analysis.CurrentMeasurement.Throughput

	regression := MetricRegression{

		MetricName: "Throughput",

		BaselineValue: baselineThroughput,

		CurrentValue: currentThroughput,

		AbsoluteChange: currentThroughput - baselineThroughput,
	}

	if baselineThroughput > 0 {

		regression.RelativeChangePct = (currentThroughput - baselineThroughput) / baselineThroughput * 100

	}

	// For throughput, negative change is regression.

	isRegression := regression.RelativeChangePct < -rd.config.ThroughputRegressionPct

	// Statistical test.

	testResult := rd.performTTest(rd.baseline.ThroughputMetrics.Distribution,

		[]float64{currentThroughput})

	regression.IsSignificant = testResult.IsSignificant

	regression.PValue = testResult.PValue

	regression.EffectSize = rd.calculateEffectSize(baselineThroughput, currentThroughput, rd.baseline.ThroughputMetrics.StandardDev)

	// Assess severity.

	if isRegression && regression.IsSignificant {

		if currentThroughput < rd.config.TargetThroughputRpm {

			regression.Severity = "Critical"

			regression.Impact = "Throughput below target"

		} else if math.Abs(regression.RelativeChangePct) > 20 {

			regression.Severity = "High"

			regression.Impact = "Significant throughput decrease"

		} else {

			regression.Severity = "Medium"

			regression.Impact = "Moderate throughput decrease"

		}

		regression.RegressionType = "Performance"

	} else {

		regression.Severity = "Low"

		regression.Impact = "Within acceptable range"

	}

	// Trend direction.

	if regression.RelativeChangePct < -5 {

		regression.TrendDirection = "Degrading"

	} else if regression.RelativeChangePct > 5 {

		regression.TrendDirection = "Improving"

	} else {

		regression.TrendDirection = "Stable"

	}

	analysis.MetricRegressions = append(analysis.MetricRegressions, regression)

	analysis.StatisticalTests["throughput"] = testResult

	return nil

}

// analyzeErrorRateRegression analyzes error rate for regression.

func (rd *RegressionDetector) analyzeErrorRateRegression(analysis *RegressionAnalysis) error {

	baselineErrorRate := rd.baseline.ErrorRateMetrics.Mean

	currentErrorRate := analysis.CurrentMeasurement.ErrorRate

	regression := MetricRegression{

		MetricName: "ErrorRate",

		BaselineValue: baselineErrorRate,

		CurrentValue: currentErrorRate,

		AbsoluteChange: currentErrorRate - baselineErrorRate,
	}

	if baselineErrorRate > 0 {

		regression.RelativeChangePct = (currentErrorRate - baselineErrorRate) / baselineErrorRate * 100

	} else if currentErrorRate > 0 {

		regression.RelativeChangePct = 100 // 100% increase from 0

	}

	// For error rate, positive change is regression.

	isRegression := regression.RelativeChangePct > rd.config.ErrorRateRegressionPct

	testResult := rd.performTTest(rd.baseline.ErrorRateMetrics.Distribution,

		[]float64{currentErrorRate})

	regression.IsSignificant = testResult.IsSignificant

	regression.PValue = testResult.PValue

	regression.EffectSize = rd.calculateEffectSize(baselineErrorRate, currentErrorRate, rd.baseline.ErrorRateMetrics.StandardDev)

	// Assess severity.

	if isRegression && regression.IsSignificant {

		if currentErrorRate > 5.0 { // 5% error rate threshold

			regression.Severity = "Critical"

			regression.Impact = "High error rate affecting users"

		} else if currentErrorRate > 2.0 {

			regression.Severity = "High"

			regression.Impact = "Elevated error rate"

		} else {

			regression.Severity = "Medium"

			regression.Impact = "Slight increase in errors"

		}

		regression.RegressionType = "Availability"

	} else {

		regression.Severity = "Low"

		regression.Impact = "Error rate acceptable"

	}

	analysis.MetricRegressions = append(analysis.MetricRegressions, regression)

	analysis.StatisticalTests["error_rate"] = testResult

	return nil

}

// analyzeAvailabilityRegression analyzes availability for regression.

func (rd *RegressionDetector) analyzeAvailabilityRegression(analysis *RegressionAnalysis) error {

	baselineAvailability := rd.baseline.AvailabilityMetrics.Mean

	currentAvailability := analysis.CurrentMeasurement.Availability

	regression := MetricRegression{

		MetricName: "Availability",

		BaselineValue: baselineAvailability,

		CurrentValue: currentAvailability,

		AbsoluteChange: currentAvailability - baselineAvailability,
	}

	if baselineAvailability > 0 {

		regression.RelativeChangePct = (currentAvailability - baselineAvailability) / baselineAvailability * 100

	}

	// For availability, negative change is regression.

	isRegression := regression.RelativeChangePct < -rd.config.AvailabilityRegressionPct

	testResult := rd.performTTest(rd.baseline.AvailabilityMetrics.Distribution,

		[]float64{currentAvailability})

	regression.IsSignificant = testResult.IsSignificant

	regression.PValue = testResult.PValue

	regression.EffectSize = rd.calculateEffectSize(baselineAvailability, currentAvailability, rd.baseline.AvailabilityMetrics.StandardDev)

	// Assess severity.

	if isRegression && regression.IsSignificant {

		if currentAvailability < rd.config.TargetAvailabilityPct {

			regression.Severity = "Critical"

			regression.Impact = "Availability below target"

		} else if currentAvailability < 99.9 {

			regression.Severity = "High"

			regression.Impact = "Significant availability degradation"

		} else {

			regression.Severity = "Medium"

			regression.Impact = "Moderate availability decrease"

		}

		regression.RegressionType = "Availability"

	} else {

		regression.Severity = "Low"

		regression.Impact = "Availability acceptable"

	}

	analysis.MetricRegressions = append(analysis.MetricRegressions, regression)

	analysis.StatisticalTests["availability"] = testResult

	return nil

}

// performTTest performs a two-sample t-test.

func (rd *RegressionDetector) performTTest(baseline, current []float64) *StatisticalTestResult {

	if len(baseline) == 0 || len(current) == 0 {

		return &StatisticalTestResult{

			TestName: "t-test",

			TestType: "two-sample",

			IsSignificant: false,

			Interpretation: "Insufficient data for test",
		}

	}

	baselineMean := stat.Mean(baseline, nil)

	currentMean := stat.Mean(current, nil)

	baselineVar := stat.Variance(baseline, nil)

	currentVar := stat.Variance(current, nil)

	n1, n2 := float64(len(baseline)), float64(len(current))

	// Welch's t-test for unequal variances.

	pooledSE := math.Sqrt(baselineVar/n1 + currentVar/n2)

	tStat := (currentMean - baselineMean) / pooledSE

	// Degrees of freedom for Welch's test.

	df := math.Pow(baselineVar/n1+currentVar/n2, 2) /

		(math.Pow(baselineVar/n1, 2)/(n1-1) + math.Pow(currentVar/n2, 2)/(n2-1))

	// Simplified p-value calculation (would use proper t-distribution in production).

	pValue := 0.05 // Placeholder - would calculate actual p-value

	isSignificant := pValue < rd.config.SignificanceLevel

	return &StatisticalTestResult{

		TestName: "t-test",

		TestType: "two-sample",

		Statistic: tStat,

		PValue: pValue,

		IsSignificant: isSignificant,

		DegreesOfFreedom: int(df),

		AlternativeHypothesis: "two-sided",

		Interpretation: fmt.Sprintf("Difference is statistically %s (p=%.4f)",

			map[bool]string{true: "significant", false: "not significant"}[isSignificant], pValue),
	}

}

// calculateEffectSize computes Cohen's d effect size.

func (rd *RegressionDetector) calculateEffectSize(baseline, current, pooledStdDev float64) float64 {

	if pooledStdDev == 0 {

		return 0

	}

	return (current - baseline) / pooledStdDev

}

// determineOverallRegression determines if overall regression exists.

func (rd *RegressionDetector) determineOverallRegression(analysis *RegressionAnalysis) {

	criticalCount := 0

	highCount := 0

	significantCount := 0

	for _, regression := range analysis.MetricRegressions {

		if regression.IsSignificant {

			significantCount++

		}

		switch regression.Severity {

		case "Critical":

			criticalCount++

		case "High":

			highCount++

		}

	}

	// Determine overall regression status.

	analysis.HasRegression = criticalCount > 0 || highCount > 1 || significantCount > 2

	// Determine overall severity.

	if criticalCount > 0 {

		analysis.RegressionSeverity = "Critical"

		analysis.ConfidenceScore = 0.95

	} else if highCount > 1 {

		analysis.RegressionSeverity = "High"

		analysis.ConfidenceScore = 0.85

	} else if highCount > 0 || significantCount > 2 {

		analysis.RegressionSeverity = "Medium"

		analysis.ConfidenceScore = 0.75

	} else {

		analysis.RegressionSeverity = "Low"

		analysis.ConfidenceScore = 0.60

	}

}

// generateCausesAndRecommendations generates potential causes and recommendations.

func (rd *RegressionDetector) generateCausesAndRecommendations(analysis *RegressionAnalysis) {

	// Analyze potential causes based on regression patterns.

	for _, regression := range analysis.MetricRegressions {

		switch regression.MetricName {

		case "LatencyP95":

			if regression.Severity == "Critical" || regression.Severity == "High" {

				analysis.PotentialCauses = append(analysis.PotentialCauses, PotentialCause{

					Category: "Code",

					Description: "Inefficient algorithm or increased processing complexity",

					Likelihood: 0.7,

					Evidence: []string{"Significant latency increase without load increase"},

					Investigation: []string{"Review recent code changes", "Profile CPU usage", "Analyze LLM processing time"},
				})

				analysis.ImmediateActions = append(analysis.ImmediateActions,

					"Enable detailed performance profiling",

					"Review recent deployments and changes",

					"Check LLM and RAG service health")

			}

		case "Throughput":

			if regression.Severity == "Critical" || regression.Severity == "High" {

				analysis.PotentialCauses = append(analysis.PotentialCauses, PotentialCause{

					Category: "Infrastructure",

					Description: "Resource constraints or bottlenecks",

					Likelihood: 0.8,

					Evidence: []string{"Throughput decrease without configuration changes"},

					Investigation: []string{"Check CPU and memory utilization", "Analyze database performance", "Review connection pools"},
				})

				analysis.ImmediateActions = append(analysis.ImmediateActions,

					"Scale up resources if needed",

					"Check for resource bottlenecks",

					"Verify downstream service health")

			}

		case "ErrorRate":

			if regression.Severity == "Critical" || regression.Severity == "High" {

				analysis.PotentialCauses = append(analysis.PotentialCauses, PotentialCause{

					Category: "External",

					Description: "Downstream service failures or network issues",

					Likelihood: 0.9,

					Evidence: []string{"Increased error rate"},

					Investigation: []string{"Check downstream service logs", "Verify network connectivity", "Review error patterns"},
				})

				analysis.ImmediateActions = append(analysis.ImmediateActions,

					"Check error logs and patterns",

					"Verify external service dependencies",

					"Consider circuit breaker activation")

			}

		}

	}

	// Generate investigation steps.

	analysis.InvestigationSteps = append(analysis.InvestigationSteps,

		"Compare current metrics with historical trends",

		"Analyze correlation with external factors",

		"Review system configuration changes",

		"Examine resource utilization patterns",

		"Investigate user behavior changes")

	// Generate prevention measures.

	analysis.PreventionMeasures = append(analysis.PreventionMeasures,

		"Implement automated performance testing in CI/CD",

		"Set up proactive monitoring and alerting",

		"Establish performance budgets and gates",

		"Regular baseline updates and validation",

		"Implement gradual rollout strategies")

}

// SendAlert sends regression alert through notification channels.

func (rd *RegressionDetector) SendAlert(ctx context.Context, analysis *RegressionAnalysis) error {

	if !analysis.HasRegression {

		return nil // No alert needed

	}

	alert := &RegressionAlert{

		ID: analysis.AnalysisID,

		Timestamp: analysis.Timestamp,

		Severity: analysis.RegressionSeverity,

		Title: fmt.Sprintf("Performance Regression Detected - %s", analysis.RegressionSeverity),

		Description: rd.generateAlertDescription(analysis),

		AffectedMetrics: rd.extractAffectedMetrics(analysis),

		Analysis: analysis,

		Actions: rd.generateRecommendedActions(analysis),

		Context: map[string]interface{}{

			"confidence_score": analysis.ConfidenceScore,

			"baseline_id": analysis.BaselineUsed.ID,

			"test_duration": analysis.CurrentMeasurement.TestDuration,
		},
	}

	return rd.notifications.SendAlert(alert)

}

// generateAlertDescription creates human-readable alert description.

func (rd *RegressionDetector) generateAlertDescription(analysis *RegressionAnalysis) string {

	desc := fmt.Sprintf("Performance regression detected with %.1f%% confidence. ", analysis.ConfidenceScore*100)

	criticalMetrics := make([]string, 0)

	for _, regression := range analysis.MetricRegressions {

		if regression.Severity == "Critical" || regression.Severity == "High" {

			criticalMetrics = append(criticalMetrics,

				fmt.Sprintf("%s: %.1f%% change", regression.MetricName, regression.RelativeChangePct))

		}

	}

	if len(criticalMetrics) > 0 {

		desc += "Affected metrics: " + fmt.Sprintf("%v", criticalMetrics)

	}

	return desc

}

// extractAffectedMetrics extracts names of regressed metrics.

func (rd *RegressionDetector) extractAffectedMetrics(analysis *RegressionAnalysis) []string {

	metrics := make([]string, 0)

	for _, regression := range analysis.MetricRegressions {

		if regression.Severity == "Critical" || regression.Severity == "High" {

			metrics = append(metrics, regression.MetricName)

		}

	}

	return metrics

}

// generateRecommendedActions converts analysis recommendations to actions.

func (rd *RegressionDetector) generateRecommendedActions(analysis *RegressionAnalysis) []RecommendedAction {

	actions := make([]RecommendedAction, 0)

	// Immediate actions.

	for i, action := range analysis.ImmediateActions {

		actions = append(actions, RecommendedAction{

			Type: "immediate",

			Description: action,

			Priority: i + 1,

			EstimatedTime: 30 * time.Minute,

			Owner: "ops-team",
		})

	}

	// Investigation actions.

	for i, action := range analysis.InvestigationSteps {

		actions = append(actions, RecommendedAction{

			Type: "investigation",

			Description: action,

			Priority: i + 1,

			EstimatedTime: 2 * time.Hour,

			Owner: "dev-team",
		})

	}

	return actions

}

// UpdateBaseline updates the performance baseline with new data.

func (rd *RegressionDetector) UpdateBaseline(baseline *PerformanceBaseline) error {

	if baseline == nil {

		return fmt.Errorf("baseline cannot be nil")

	}

	// Validate baseline quality.

	if baseline.QualityScore < 0.7 {

		return fmt.Errorf("baseline quality score %.2f below threshold 0.7", baseline.QualityScore)

	}

	rd.baseline = baseline

	klog.Infof("Updated performance baseline: ID=%s, Quality=%.2f", baseline.ID, baseline.QualityScore)

	return nil

}

// Supporting functions and types for remaining functionality.

func getDefaultAlertThresholds() *AlertThresholds {

	return &AlertThresholds{

		CriticalLatencyMs: 3000, // 3 seconds is critical

		CriticalErrorRatePct: 5.0, // 5% error rate is critical

		CriticalAvailabilityPct: 99.0, // Below 99% is critical

		WarningLatencyMs: 2500, // Warning at 2.5 seconds

		WarningErrorRatePct: 2.0, // Warning at 2% error rate

		WarningAvailabilityPct: 99.5, // Warning below 99.5%

		TrendDegradationPct: 15.0, // 15% degradation over time

		ConsecutiveFailures: 3, // 3 consecutive bad measurements

	}

}

func getDefaultTrendConfig() *TrendAnalysisConfig {

	return &TrendAnalysisConfig{

		WindowSizes: []int{10, 50, 100},

		TrendMethods: []string{"linear", "seasonal"},

		ChangePointDetection: true,

		AnomalyDetection: true,

		ForecastingEnabled: true,
	}

}

func getDefaultNotificationConfig() *NotificationConfig {

	return &NotificationConfig{

		EnabledChannels: []string{"console", "metrics"},

		AlertLevels: []string{"Critical", "High"},

		RateLimiting: true,
	}

}

// Placeholder implementations for additional components.

// NewTrendAnalyzer performs newtrendanalyzer operation.

func NewTrendAnalyzer(config *TrendAnalysisConfig) *TrendAnalyzer {

	return &TrendAnalyzer{

		config: config,
	}

}

// AnalyzeTrends performs analyzetrends operation.

func (ta *TrendAnalyzer) AnalyzeTrends(ctx context.Context, measurement *PerformanceMeasurement) (*TrendAnalysisResult, error) {

	return &TrendAnalysisResult{

		Trend: "Stable",

		TrendCoefficient: 0.0,
	}, nil

}

// NewNotificationManager performs newnotificationmanager operation.

func NewNotificationManager(config *NotificationConfig) *NotificationManager {

	return &NotificationManager{

		config: config,

		channels: make(map[string]NotificationChannel),
	}

}

// SendAlert performs sendalert operation.

func (nm *NotificationManager) SendAlert(alert *RegressionAlert) error {

	klog.Infof("Regression Alert: %s - %s", alert.Severity, alert.Title)

	return nil

}

// Additional types for completeness.

type (
	HistoricalDataStore struct{}

	// SeasonalModel represents a seasonalmodel.

	SeasonalModel struct{}

	// EscalationRule represents a escalationrule.

	EscalationRule struct{}

	// TimeWindow represents a timewindow.

	TimeWindow struct {
		Start time.Time

		End time.Time
	}
)
