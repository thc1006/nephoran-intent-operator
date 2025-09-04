// Package validation provides trend analysis capabilities for regression testing.

package test_validation

import (
	
	"encoding/json"
"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
)

// TrendAnalyzer performs statistical analysis of validation trends over time.

type TrendAnalyzer struct {
	config *RegressionConfig
}

// NewTrendAnalyzer creates a new trend analyzer.

func NewTrendAnalyzer(config *RegressionConfig) *TrendAnalyzer {
	return &TrendAnalyzer{
		config: config,
	}
}

// TrendAnalysis contains comprehensive trend analysis results.

type TrendAnalysis struct {
	// Overall trends.

	OverallTrend *OverallTrendAnalysis `json:"overall_trend"`

	// Category-specific trends.

	PerformanceTrends map[string]*PerformanceTrend `json:"performance_trends"`

	FunctionalTrends map[string]*FunctionalTrend `json:"functional_trends"`

	SecurityTrends *SecurityTrend `json:"security_trends"`

	ProductionTrends *ProductionTrend `json:"production_trends"`

	// Statistical insights.

	SeasonalPatterns map[string]*SeasonalPattern `json:"seasonal_patterns"`

	Anomalies []*TrendAnomaly `json:"anomalies"`

	Predictions map[string]*TrendPrediction `json:"predictions"`

	// Analysis metadata.

	AnalysisTime time.Time `json:"analysis_time"`

	SampleSize int `json:"sample_size"`

	TimeRange TimeRange `json:"time_range"`

	ConfidenceLevel float64 `json:"confidence_level"`
}

// OverallTrendAnalysis provides high-level trend insights.

type OverallTrendAnalysis struct {
	Direction string `json:"direction"` // "improving", "stable", "degrading"

	Confidence float64 `json:"confidence"`

	TrendStrength string `json:"trend_strength"` // "weak", "moderate", "strong"

	QualityTrajectory string `json:"quality_trajectory"` // "positive", "neutral", "negative"

	PredictedDirection string `json:"predicted_direction"`

	TimeToTarget *time.Duration `json:"time_to_target,omitempty"`

	RiskLevel string `json:"risk_level"` // "low", "medium", "high", "critical"
}

// PerformanceTrend tracks performance metric trends.

type PerformanceTrend struct {
	MetricName string `json:"metric_name"`

	Direction string `json:"direction"`

	TrendSlope float64 `json:"trend_slope"`

	Correlation float64 `json:"correlation"`

	Volatility float64 `json:"volatility"`

	HistoricalValues []TrendDataPoint `json:"historical_values"`

	MovingAverage []float64 `json:"moving_average"`

	ThresholdBreaches []ThresholdBreach `json:"threshold_breaches"`

	PredictedValue float64 `json:"predicted_value"`

	PredictionInterval [2]float64 `json:"prediction_interval"`
}

// FunctionalTrend tracks functional test trends.

type FunctionalTrend struct {
	Category string `json:"category"`

	PassRateTrend string `json:"pass_rate_trend"`

	TestStability float64 `json:"test_stability"`

	FlakeyTests []string `json:"flakey_tests"`

	ChronicFailures []string `json:"chronic_failures"`

	NewTestImpact float64 `json:"new_test_impact"`

	HistoricalPassRate []TrendDataPoint `json:"historical_pass_rate"`

	TrendConfidence float64 `json:"trend_confidence"`
}

// SecurityTrend tracks security posture trends.

type SecurityTrend struct {
	VulnerabilityTrend string `json:"vulnerability_trend"`

	ComplianceTrend string `json:"compliance_trend"`

	CriticalVulnHistory []TrendDataPoint `json:"critical_vuln_history"`

	ComplianceScoreHistory []TrendDataPoint `json:"compliance_score_history"`

	NewVulnerabilityRate float64 `json:"new_vulnerability_rate"`

	RemediationEffectiveness float64 `json:"remediation_effectiveness"`

	SecurityPostureTrend string `json:"security_posture_trend"`
}

// ProductionTrend tracks production readiness trends.

type ProductionTrend struct {
	ReadinessTrend string `json:"readiness_trend"`

	AvailabilityTrend string `json:"availability_trend"`

	ReliabilityTrend string `json:"reliability_trend"`

	MTBFTrend []TrendDataPoint `json:"mtbf_trend"`

	MTTRTrend []TrendDataPoint `json:"mttr_trend"`

	IncidentFrequency float64 `json:"incident_frequency"`

	RecoveryEfficiency float64 `json:"recovery_efficiency"`
}

// TrendDataPoint represents a single data point in a trend.

type TrendDataPoint struct {
	Timestamp time.Time `json:"timestamp"`

	Value float64 `json:"value"`

	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// SeasonalPattern identifies recurring patterns.

type SeasonalPattern struct {
	Pattern string `json:"pattern"` // "daily", "weekly", "monthly"

	Strength float64 `json:"strength"`

	Phase float64 `json:"phase"`

	Amplitude float64 `json:"amplitude"`

	Detected bool `json:"detected"`

	NextPeak time.Time `json:"next_peak"`

	NextTrough time.Time `json:"next_trough"`
}

// TrendAnomaly represents an unusual data point.

type TrendAnomaly struct {
	Timestamp time.Time `json:"timestamp"`

	MetricName string `json:"metric_name"`

	ExpectedValue float64 `json:"expected_value"`

	ActualValue float64 `json:"actual_value"`

	Deviation float64 `json:"deviation"`

	Severity string `json:"severity"`

	PossibleCause string `json:"possible_cause"`
}

// TrendPrediction provides future value predictions.

type TrendPrediction struct {
	MetricName string `json:"metric_name"`

	PredictedValue float64 `json:"predicted_value"`

	ConfidenceInterval [2]float64 `json:"confidence_interval"`

	TimeHorizon time.Duration `json:"time_horizon"`

	Accuracy float64 `json:"accuracy"`

	Model string `json:"model"` // "linear", "exponential", "polynomial"
}

// ThresholdBreach tracks when metrics exceeded thresholds.

type ThresholdBreach struct {
	Timestamp time.Time `json:"timestamp"`

	ThresholdType string `json:"threshold_type"` // "warning", "critical"

	Value float64 `json:"value"`

	Threshold float64 `json:"threshold"`

	Duration time.Duration `json:"duration"`
}

// TimeRange defines the analysis time period.

type TimeRange struct {
	Start time.Time `json:"start"`

	End time.Time `json:"end"`
}

// AnalyzeTrends compares current results with baseline and generates trend analysis.

func (ta *TrendAnalyzer) AnalyzeTrends(baseline *BaselineSnapshot, current *ValidationResults) *TrendAnalysis {
	ginkgo.By("Performing trend analysis")

	analysis := &TrendAnalysis{
		PerformanceTrends: make(map[string]*PerformanceTrend),

		FunctionalTrends: make(map[string]*FunctionalTrend),

		SeasonalPatterns: make(map[string]*SeasonalPattern),

		Anomalies: []*TrendAnomaly{},

		Predictions: make(map[string]*TrendPrediction),

		AnalysisTime: time.Now(),

		SampleSize: 2, // baseline + current

		ConfidenceLevel: ta.config.StatisticalConfidence,

		TimeRange: TimeRange{
			Start: baseline.Timestamp,

			End: time.Now(),
		},
	}

	// Analyze overall trends.

	analysis.OverallTrend = ta.analyzeOverallTrend(baseline, current)

	// Analyze performance trends.

	analysis.PerformanceTrends = ta.analyzePerformanceTrends(baseline, current)

	// Analyze functional trends.

	analysis.FunctionalTrends = ta.analyzeFunctionalTrends(baseline, current)

	// Analyze security trends.

	analysis.SecurityTrends = ta.analyzeSecurityTrends(baseline, current)

	// Analyze production trends.

	analysis.ProductionTrends = ta.analyzeProductionTrends(baseline, current)

	ginkgo.By("Trend analysis completed")

	return analysis
}

// GenerateTrends creates comprehensive trend analysis from multiple baselines.

func (ta *TrendAnalyzer) GenerateTrends(baselines []*BaselineSnapshot) *TrendAnalysis {
	ginkgo.By(fmt.Sprintf("Generating comprehensive trends from %d baselines", len(baselines)))

	if len(baselines) < ta.config.MinimumSamples {

		ginkgo.By(fmt.Sprintf("Insufficient samples for trend analysis: %d < %d", len(baselines), ta.config.MinimumSamples))

		return ta.createMinimalTrendAnalysis(baselines)

	}

	// Sort baselines by timestamp.

	sort.Slice(baselines, func(i, j int) bool {
		return baselines[i].Timestamp.Before(baselines[j].Timestamp)
	})

	analysis := &TrendAnalysis{
		PerformanceTrends: make(map[string]*PerformanceTrend),

		FunctionalTrends: make(map[string]*FunctionalTrend),

		SeasonalPatterns: make(map[string]*SeasonalPattern),

		Anomalies: []*TrendAnomaly{},

		Predictions: make(map[string]*TrendPrediction),

		AnalysisTime: time.Now(),

		SampleSize: len(baselines),

		ConfidenceLevel: ta.config.StatisticalConfidence,

		TimeRange: TimeRange{
			Start: baselines[0].Timestamp,

			End: baselines[len(baselines)-1].Timestamp,
		},
	}

	// Generate comprehensive trend analysis.

	analysis.OverallTrend = ta.generateOverallTrends(baselines)

	analysis.PerformanceTrends = ta.generatePerformanceTrends(baselines)

	analysis.FunctionalTrends = ta.generateFunctionalTrends(baselines)

	analysis.SecurityTrends = ta.generateSecurityTrends(baselines)

	analysis.ProductionTrends = ta.generateProductionTrends(baselines)

	analysis.SeasonalPatterns = ta.detectSeasonalPatterns(baselines)

	analysis.Anomalies = ta.detectAnomalies(baselines)

	analysis.Predictions = ta.generatePredictions(baselines)

	ginkgo.By("Comprehensive trend analysis completed")

	return analysis
}

// analyzeOverallTrend determines the overall quality trajectory.

func (ta *TrendAnalyzer) analyzeOverallTrend(baseline *BaselineSnapshot, current *ValidationResults) *OverallTrendAnalysis {
	scoreDiff := current.TotalScore - baseline.Results.TotalScore

	direction := "stable"

	trajectory := "neutral"

	riskLevel := "low"

	if scoreDiff > 2 {

		direction = "improving"

		trajectory = "positive"

	} else if scoreDiff < -2 {

		direction = "degrading"

		trajectory = "negative"

		riskLevel = "medium"

		if scoreDiff < -5 {
			riskLevel = "high"
		}

	}

	return &OverallTrendAnalysis{
		Direction: direction,

		Confidence: 85.0, // With only 2 samples, confidence is limited

		TrendStrength: "moderate",

		QualityTrajectory: trajectory,

		PredictedDirection: direction,

		RiskLevel: riskLevel,
	}
}

// analyzePerformanceTrends analyzes performance metric trends.

func (ta *TrendAnalyzer) analyzePerformanceTrends(baseline *BaselineSnapshot, current *ValidationResults) map[string]*PerformanceTrend {
	trends := make(map[string]*PerformanceTrend)

	// P95 Latency trend.

	if baseline.Results.P95Latency > 0 && current.P95Latency > 0 {

		baselineNs := float64(baseline.Results.P95Latency.Nanoseconds())

		currentNs := float64(current.P95Latency.Nanoseconds())

		direction := ta.determineDirection(baselineNs, currentNs, true) // Higher is worse for latency

		slope := ta.calculateSlope([]TrendDataPoint{
			{Timestamp: baseline.Timestamp, Value: baselineNs},

			{Timestamp: time.Now(), Value: currentNs},
		})

		trends["p95_latency"] = &PerformanceTrend{
			MetricName: "P95 Latency",

			Direction: direction,

			TrendSlope: slope,

			Correlation: 1.0, // Perfect correlation with only 2 points

			Volatility: math.Abs(currentNs-baselineNs) / baselineNs,

			HistoricalValues: []TrendDataPoint{
				{Timestamp: baseline.Timestamp, Value: baselineNs},

				{Timestamp: time.Now(), Value: currentNs},
			},

			PredictedValue: ta.predictNextValue([]float64{baselineNs, currentNs}),
		}

	}

	// Throughput trend.

	if baseline.Results.ThroughputAchieved > 0 && current.ThroughputAchieved > 0 {

		direction := ta.determineDirection(baseline.Results.ThroughputAchieved, current.ThroughputAchieved, false) // Higher is better

		slope := ta.calculateSlope([]TrendDataPoint{
			{Timestamp: baseline.Timestamp, Value: baseline.Results.ThroughputAchieved},

			{Timestamp: time.Now(), Value: current.ThroughputAchieved},
		})

		trends["throughput"] = &PerformanceTrend{
			MetricName: "Throughput",

			Direction: direction,

			TrendSlope: slope,

			Correlation: 1.0,

			Volatility: math.Abs(current.ThroughputAchieved-baseline.Results.ThroughputAchieved) / baseline.Results.ThroughputAchieved,

			HistoricalValues: []TrendDataPoint{
				{Timestamp: baseline.Timestamp, Value: baseline.Results.ThroughputAchieved},

				{Timestamp: time.Now(), Value: current.ThroughputAchieved},
			},

			PredictedValue: ta.predictNextValue([]float64{baseline.Results.ThroughputAchieved, current.ThroughputAchieved}),
		}

	}

	// Availability trend.

	if baseline.Results.AvailabilityAchieved > 0 && current.AvailabilityAchieved > 0 {

		direction := ta.determineDirection(baseline.Results.AvailabilityAchieved, current.AvailabilityAchieved, false) // Higher is better

		slope := ta.calculateSlope([]TrendDataPoint{
			{Timestamp: baseline.Timestamp, Value: baseline.Results.AvailabilityAchieved},

			{Timestamp: time.Now(), Value: current.AvailabilityAchieved},
		})

		trends["availability"] = &PerformanceTrend{
			MetricName: "Availability",

			Direction: direction,

			TrendSlope: slope,

			Correlation: 1.0,

			Volatility: math.Abs(current.AvailabilityAchieved-baseline.Results.AvailabilityAchieved) / baseline.Results.AvailabilityAchieved,

			HistoricalValues: []TrendDataPoint{
				{Timestamp: baseline.Timestamp, Value: baseline.Results.AvailabilityAchieved},

				{Timestamp: time.Now(), Value: current.AvailabilityAchieved},
			},

			PredictedValue: ta.predictNextValue([]float64{baseline.Results.AvailabilityAchieved, current.AvailabilityAchieved}),
		}

	}

	return trends
}

// analyzeFunctionalTrends analyzes functional test trends.

func (ta *TrendAnalyzer) analyzeFunctionalTrends(baseline *BaselineSnapshot, current *ValidationResults) map[string]*FunctionalTrend {
	trends := make(map[string]*FunctionalTrend)

	// Overall functional trend.

	baselinePassRate := float64(baseline.Results.FunctionalScore) / 50.0 * 100

	currentPassRate := float64(current.FunctionalScore) / 50.0 * 100

	trends["overall"] = &FunctionalTrend{
		Category: "Overall Functional",

		PassRateTrend: ta.determineDirection(baselinePassRate, currentPassRate, false),

		TestStability: ta.calculateStability([]float64{baselinePassRate, currentPassRate}),

		FlakeyTests: []string{}, // Would need test history to identify

		ChronicFailures: []string{}, // Would need test history to identify

		NewTestImpact: 0.0, // Would need test diff to calculate

		HistoricalPassRate: []TrendDataPoint{
			{Timestamp: baseline.Timestamp, Value: baselinePassRate},

			{Timestamp: time.Now(), Value: currentPassRate},
		},

		TrendConfidence: 85.0,
	}

	return trends
}

// analyzeSecurityTrends analyzes security posture trends.

func (ta *TrendAnalyzer) analyzeSecurityTrends(baseline *BaselineSnapshot, current *ValidationResults) *SecurityTrend {
	baselineVulns := len(baseline.Results.SecurityFindings)

	currentVulns := len(current.SecurityFindings)

	vulnTrend := ta.determineDirection(float64(baselineVulns), float64(currentVulns), true) // Higher is worse

	baselineScore := float64(baseline.Results.SecurityScore)

	currentScore := float64(current.SecurityScore)

	complianceTrend := ta.determineDirection(baselineScore, currentScore, false) // Higher is better

	return &SecurityTrend{
		VulnerabilityTrend: vulnTrend,

		ComplianceTrend: complianceTrend,

		CriticalVulnHistory: []TrendDataPoint{
			{Timestamp: baseline.Timestamp, Value: float64(ta.countCriticalVulns(baseline.Results.SecurityFindings))},

			{Timestamp: time.Now(), Value: float64(ta.countCriticalVulns(current.SecurityFindings))},
		},

		ComplianceScoreHistory: []TrendDataPoint{
			{Timestamp: baseline.Timestamp, Value: baselineScore},

			{Timestamp: time.Now(), Value: currentScore},
		},

		NewVulnerabilityRate: ta.calculateVulnRate(baseline, current),

		RemediationEffectiveness: ta.calculateRemediationEffectiveness(baseline, current),

		SecurityPostureTrend: complianceTrend,
	}
}

// analyzeProductionTrends analyzes production readiness trends.

func (ta *TrendAnalyzer) analyzeProductionTrends(baseline *BaselineSnapshot, current *ValidationResults) *ProductionTrend {
	baselineScore := float64(baseline.Results.ProductionScore)

	currentScore := float64(current.ProductionScore)

	readinessTrend := ta.determineDirection(baselineScore, currentScore, false) // Higher is better

	availabilityTrend := ta.determineDirection(baseline.Results.AvailabilityAchieved, current.AvailabilityAchieved, false)

	return &ProductionTrend{
		ReadinessTrend: readinessTrend,

		AvailabilityTrend: availabilityTrend,

		ReliabilityTrend: "stable", // Would need more data

		MTBFTrend: []TrendDataPoint{
			{Timestamp: baseline.Timestamp, Value: float64(24 * time.Hour)}, // Default assumption

			{Timestamp: time.Now(), Value: float64(24 * time.Hour)},
		},

		MTTRTrend: []TrendDataPoint{
			{Timestamp: baseline.Timestamp, Value: float64(5 * time.Minute)}, // Default assumption

			{Timestamp: time.Now(), Value: float64(5 * time.Minute)},
		},

		IncidentFrequency: 0.1, // Default assumption

		RecoveryEfficiency: 95.0, // Default assumption

	}
}

// Helper methods for more comprehensive trend analysis with multiple baselines.

// generateOverallTrends creates overall trend analysis from multiple baselines.

func (ta *TrendAnalyzer) generateOverallTrends(baselines []*BaselineSnapshot) *OverallTrendAnalysis {
	scores := make([]float64, len(baselines))

	for i, baseline := range baselines {
		scores[i] = float64(baseline.Results.TotalScore)
	}

	slope := ta.calculateLinearTrendSlope(baselines, scores)

	correlation := ta.calculateCorrelation(baselines, scores)

	direction := "stable"

	trajectory := "neutral"

	strength := "weak"

	riskLevel := "low"

	if math.Abs(slope) > 0.1 {

		if slope > 0 {

			direction = "improving"

			trajectory = "positive"

		} else {

			direction = "degrading"

			trajectory = "negative"

			riskLevel = "medium"

		}

		if math.Abs(slope) > 0.5 {

			strength = "strong"

			if slope < 0 {
				riskLevel = "high"
			}

		} else if math.Abs(slope) > 0.25 {
			strength = "moderate"
		}

	}

	return &OverallTrendAnalysis{
		Direction: direction,

		Confidence: math.Abs(correlation) * 100,

		TrendStrength: strength,

		QualityTrajectory: trajectory,

		PredictedDirection: direction,

		RiskLevel: riskLevel,
	}
}

// generatePerformanceTrends creates performance trends from multiple baselines.

func (ta *TrendAnalyzer) generatePerformanceTrends(baselines []*BaselineSnapshot) map[string]*PerformanceTrend {
	trends := make(map[string]*PerformanceTrend)

	// Extract P95 latency values.

	p95Values := ta.extractP95LatencyValues(baselines)

	if len(p95Values) >= ta.config.MinimumSamples {
		trends["p95_latency"] = ta.createPerformanceTrend("P95 Latency", baselines, p95Values, true)
	}

	// Extract throughput values.

	throughputValues := ta.extractThroughputValues(baselines)

	if len(throughputValues) >= ta.config.MinimumSamples {
		trends["throughput"] = ta.createPerformanceTrend("Throughput", baselines, throughputValues, false)
	}

	// Extract availability values.

	availabilityValues := ta.extractAvailabilityValues(baselines)

	if len(availabilityValues) >= ta.config.MinimumSamples {
		trends["availability"] = ta.createPerformanceTrend("Availability", baselines, availabilityValues, false)
	}

	return trends
}

// generateFunctionalTrends creates functional trends from multiple baselines.

func (ta *TrendAnalyzer) generateFunctionalTrends(baselines []*BaselineSnapshot) map[string]*FunctionalTrend {
	trends := make(map[string]*FunctionalTrend)

	functionalScores := make([]float64, len(baselines))

	for i, baseline := range baselines {
		functionalScores[i] = float64(baseline.Results.FunctionalScore) / 50.0 * 100
	}

	trends["overall"] = &FunctionalTrend{
		Category: "Overall Functional",

		PassRateTrend: ta.determineTrendDirection(functionalScores, false),

		TestStability: ta.calculateStability(functionalScores),

		HistoricalPassRate: ta.createTrendDataPoints(baselines, functionalScores),

		TrendConfidence: ta.calculateTrendConfidence(functionalScores),
	}

	return trends
}

// generateSecurityTrends creates security trends from multiple baselines.

func (ta *TrendAnalyzer) generateSecurityTrends(baselines []*BaselineSnapshot) *SecurityTrend {
	vulnCounts := make([]float64, len(baselines))

	securityScores := make([]float64, len(baselines))

	for i, baseline := range baselines {

		vulnCounts[i] = float64(len(baseline.Results.SecurityFindings))

		securityScores[i] = float64(baseline.Results.SecurityScore)

	}

	return &SecurityTrend{
		VulnerabilityTrend: ta.determineTrendDirection(vulnCounts, true),

		ComplianceTrend: ta.determineTrendDirection(securityScores, false),

		CriticalVulnHistory: ta.createTrendDataPoints(baselines, vulnCounts),

		ComplianceScoreHistory: ta.createTrendDataPoints(baselines, securityScores),

		SecurityPostureTrend: ta.determineTrendDirection(securityScores, false),
	}
}

// generateProductionTrends creates production trends from multiple baselines.

func (ta *TrendAnalyzer) generateProductionTrends(baselines []*BaselineSnapshot) *ProductionTrend {
	productionScores := make([]float64, len(baselines))

	availabilityScores := make([]float64, len(baselines))

	for i, baseline := range baselines {

		productionScores[i] = float64(baseline.Results.ProductionScore)

		availabilityScores[i] = baseline.Results.AvailabilityAchieved

	}

	return &ProductionTrend{
		ReadinessTrend: ta.determineTrendDirection(productionScores, false),

		AvailabilityTrend: ta.determineTrendDirection(availabilityScores, false),

		ReliabilityTrend: "stable",

		MTBFTrend: ta.createTrendDataPoints(baselines, productionScores), // Simplified

		MTTRTrend: ta.createTrendDataPoints(baselines, productionScores), // Simplified

	}
}

// detectSeasonalPatterns identifies recurring patterns in the data.

func (ta *TrendAnalyzer) detectSeasonalPatterns(baselines []*BaselineSnapshot) map[string]*SeasonalPattern {
	patterns := make(map[string]*SeasonalPattern)

	// For now, return empty patterns as we'd need more sophisticated time series analysis.

	// This would typically use FFT or autocorrelation analysis.

	return patterns
}

// detectAnomalies identifies unusual data points.

func (ta *TrendAnalyzer) detectAnomalies(baselines []*BaselineSnapshot) []*TrendAnomaly {
	var anomalies []*TrendAnomaly

	// Simple anomaly detection using standard deviation.

	scores := make([]float64, len(baselines))

	for i, baseline := range baselines {
		scores[i] = float64(baseline.Results.TotalScore)
	}

	mean, stdDev := ta.calculateMeanAndStdDev(scores)

	threshold := 2.0 * stdDev // 2-sigma threshold

	for _, baseline := range baselines {

		score := float64(baseline.Results.TotalScore)

		deviation := math.Abs(score - mean)

		if deviation > threshold {

			severity := "medium"

			if deviation > 3.0*stdDev {
				severity = "high"
			}

			anomaly := &TrendAnomaly{
				Timestamp: baseline.Timestamp,

				MetricName: "Total Score",

				ExpectedValue: mean,

				ActualValue: score,

				Deviation: deviation,

				Severity: severity,

				PossibleCause: ta.inferAnomalyCause(score, mean),
			}

			anomalies = append(anomalies, anomaly)

		}

	}

	return anomalies
}

// generatePredictions creates future value predictions.

func (ta *TrendAnalyzer) generatePredictions(baselines []*BaselineSnapshot) map[string]*TrendPrediction {
	predictions := make(map[string]*TrendPrediction)

	// Predict total score trend.

	scores := make([]float64, len(baselines))

	for i, baseline := range baselines {
		scores[i] = float64(baseline.Results.TotalScore)
	}

	nextValue := ta.predictNextValue(scores)

	confidence := ta.calculatePredictionConfidence(scores)

	predictions["total_score"] = &TrendPrediction{
		MetricName: "Total Score",

		PredictedValue: nextValue,

		ConfidenceInterval: [2]float64{
			nextValue - confidence,

			nextValue + confidence,
		},

		TimeHorizon: 7 * 24 * time.Hour, // 1 week

		Accuracy: 85.0, // Estimated based on model complexity

		Model: "linear",
	}

	return predictions
}

// Utility methods.

func (ta *TrendAnalyzer) createMinimalTrendAnalysis(baselines []*BaselineSnapshot) *TrendAnalysis {
	return &TrendAnalysis{
		OverallTrend: &OverallTrendAnalysis{
			Direction: "stable",

			Confidence: 0.0,

			TrendStrength: "insufficient_data",

			QualityTrajectory: "neutral",

			RiskLevel: "unknown",
		},

		PerformanceTrends: make(map[string]*PerformanceTrend),

		FunctionalTrends: make(map[string]*FunctionalTrend),

		SeasonalPatterns: make(map[string]*SeasonalPattern),

		Anomalies: []*TrendAnomaly{},

		Predictions: make(map[string]*TrendPrediction),

		AnalysisTime: time.Now(),

		SampleSize: len(baselines),

		ConfidenceLevel: 0.0,
	}
}

func (ta *TrendAnalyzer) determineDirection(baseline, current float64, higherIsWorse bool) string {
	diff := current - baseline

	threshold := baseline * 0.05 // 5% threshold

	if math.Abs(diff) < threshold {
		return "stable"
	}

	if higherIsWorse {

		if diff > 0 {
			return "degrading"
		}

		return "improving"

	} else {

		if diff > 0 {
			return "improving"
		}

		return "degrading"

	}
}

func (ta *TrendAnalyzer) determineTrendDirection(values []float64, higherIsWorse bool) string {
	if len(values) < 2 {
		return "stable"
	}

	slope := ta.calculateLinearSlope(values)

	threshold := 0.01 // Minimal threshold for trend detection

	if math.Abs(slope) < threshold {
		return "stable"
	}

	if higherIsWorse {

		if slope > 0 {
			return "degrading"
		}

		return "improving"

	} else {

		if slope > 0 {
			return "improving"
		}

		return "degrading"

	}
}

func (ta *TrendAnalyzer) calculateSlope(points []TrendDataPoint) float64 {
	if len(points) < 2 {
		return 0
	}

	x1 := float64(points[0].Timestamp.Unix())

	y1 := points[0].Value

	x2 := float64(points[len(points)-1].Timestamp.Unix())

	y2 := points[len(points)-1].Value

	if x2 == x1 {
		return 0
	}

	return (y2 - y1) / (x2 - x1)
}

func (ta *TrendAnalyzer) calculateLinearSlope(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	n := float64(len(values))

	sumX := 0.0

	sumY := 0.0

	sumXY := 0.0

	sumX2 := 0.0

	for i, value := range values {

		x := float64(i)

		sumX += x

		sumY += value

		sumXY += x * value

		sumX2 += x * x

	}

	// Calculate slope using least squares method.

	denominator := n*sumX2 - sumX*sumX

	if denominator == 0 {
		return 0
	}

	return (n*sumXY - sumX*sumY) / denominator
}

func (ta *TrendAnalyzer) calculateLinearTrendSlope(baselines []*BaselineSnapshot, values []float64) float64 {
	if len(baselines) != len(values) || len(values) < 2 {
		return 0
	}

	// Convert timestamps to numeric values for regression.

	times := make([]float64, len(baselines))

	baseTime := baselines[0].Timestamp.Unix()

	for i, baseline := range baselines {
		times[i] = float64(baseline.Timestamp.Unix() - baseTime)
	}

	return ta.calculateRegressionSlope(times, values)
}

func (ta *TrendAnalyzer) calculateRegressionSlope(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 2 {
		return 0
	}

	n := float64(len(x))

	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0

	for i := range x {

		sumX += x[i]

		sumY += y[i]

		sumXY += x[i] * y[i]

		sumX2 += x[i] * x[i]

	}

	denominator := n*sumX2 - sumX*sumX

	if denominator == 0 {
		return 0
	}

	return (n*sumXY - sumX*sumY) / denominator
}

func (ta *TrendAnalyzer) calculateCorrelation(baselines []*BaselineSnapshot, values []float64) float64 {
	if len(baselines) != len(values) || len(values) < 2 {
		return 0
	}

	times := make([]float64, len(baselines))

	baseTime := baselines[0].Timestamp.Unix()

	for i, baseline := range baselines {
		times[i] = float64(baseline.Timestamp.Unix() - baseTime)
	}

	return ta.calculatePearsonCorrelation(times, values)
}

func (ta *TrendAnalyzer) calculatePearsonCorrelation(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 2 {
		return 0
	}

	n := float64(len(x))

	sumX, sumY, sumXY, sumX2, sumY2 := 0.0, 0.0, 0.0, 0.0, 0.0

	for i := range x {

		sumX += x[i]

		sumY += y[i]

		sumXY += x[i] * y[i]

		sumX2 += x[i] * x[i]

		sumY2 += y[i] * y[i]

	}

	numerator := n*sumXY - sumX*sumY

	denominator := math.Sqrt((n*sumX2 - sumX*sumX) * (n*sumY2 - sumY*sumY))

	if denominator == 0 {
		return 0
	}

	return numerator / denominator
}

func (ta *TrendAnalyzer) calculateStability(values []float64) float64 {
	if len(values) < 2 {
		return 100.0
	}

	_, stdDev := ta.calculateMeanAndStdDev(values)

	mean := ta.calculateMean(values)

	if mean == 0 {
		return 0.0
	}

	// Coefficient of variation (inverted for stability).

	cv := stdDev / math.Abs(mean)

	stability := math.Max(0, 100.0-cv*100.0)

	return stability
}

func (ta *TrendAnalyzer) calculateTrendConfidence(values []float64) float64 {
	if len(values) < ta.config.MinimumSamples {
		return 0.0
	}

	// Simple confidence based on sample size and stability.

	stability := ta.calculateStability(values)

	sampleWeight := math.Min(100.0, float64(len(values))/float64(ta.config.MinimumSamples)*50.0)

	return (stability + sampleWeight) / 2.0
}

func (ta *TrendAnalyzer) predictNextValue(values []float64) float64 {
	if len(values) < 2 {

		if len(values) == 1 {
			return values[0]
		}

		return 0

	}

	// Simple linear prediction.

	slope := ta.calculateLinearSlope(values)

	return values[len(values)-1] + slope
}

func (ta *TrendAnalyzer) calculatePredictionConfidence(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	_, stdDev := ta.calculateMeanAndStdDev(values)

	return stdDev * 1.96 // 95% confidence interval
}

func (ta *TrendAnalyzer) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0

	for _, v := range values {
		sum += v
	}

	return sum / float64(len(values))
}

func (ta *TrendAnalyzer) calculateMeanAndStdDev(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}

	mean := ta.calculateMean(values)

	if len(values) == 1 {
		return mean, 0
	}

	sumSquaredDiff := 0.0

	for _, v := range values {

		diff := v - mean

		sumSquaredDiff += diff * diff

	}

	variance := sumSquaredDiff / float64(len(values)-1)

	stdDev := math.Sqrt(variance)

	return mean, stdDev
}

func (ta *TrendAnalyzer) countCriticalVulns(findings []*SecurityFinding) int {
	count := 0

	for _, finding := range findings {
		if strings.EqualFold(finding.Severity, "critical") {
			count++
		}
	}

	return count
}

func (ta *TrendAnalyzer) calculateVulnRate(baseline *BaselineSnapshot, current *ValidationResults) float64 {
	timeDiff := time.Since(baseline.Timestamp).Hours()

	if timeDiff <= 0 {
		return 0
	}

	newVulns := len(current.SecurityFindings) - len(baseline.Results.SecurityFindings)

	if newVulns < 0 {
		newVulns = 0
	}

	// Rate per day.

	return float64(newVulns) / (timeDiff / 24.0)
}

func (ta *TrendAnalyzer) calculateRemediationEffectiveness(baseline *BaselineSnapshot, current *ValidationResults) float64 {
	baselineFailedFindings := 0

	for _, finding := range baseline.Results.SecurityFindings {
		if !finding.Passed {
			baselineFailedFindings++
		}
	}

	currentFailedFindings := 0

	for _, finding := range current.SecurityFindings {
		if !finding.Passed {
			currentFailedFindings++
		}
	}

	if baselineFailedFindings == 0 {
		return 100.0 // Perfect if no initial failures
	}

	remediated := baselineFailedFindings - currentFailedFindings

	if remediated < 0 {
		remediated = 0
	}

	return float64(remediated) / float64(baselineFailedFindings) * 100.0
}

func (ta *TrendAnalyzer) inferAnomalyCause(actual, expected float64) string {
	if actual < expected {
		return "Possible performance regression or test failure"
	}

	return "Possible improvement or test variation"
}

// Helper methods for extracting values from baselines.

func (ta *TrendAnalyzer) extractP95LatencyValues(baselines []*BaselineSnapshot) []float64 {
	var values []float64

	for _, baseline := range baselines {
		if baseline.Results.P95Latency > 0 {
			values = append(values, float64(baseline.Results.P95Latency.Nanoseconds()))
		}
	}

	return values
}

func (ta *TrendAnalyzer) extractThroughputValues(baselines []*BaselineSnapshot) []float64 {
	var values []float64

	for _, baseline := range baselines {
		if baseline.Results.ThroughputAchieved > 0 {
			values = append(values, baseline.Results.ThroughputAchieved)
		}
	}

	return values
}

func (ta *TrendAnalyzer) extractAvailabilityValues(baselines []*BaselineSnapshot) []float64 {
	var values []float64

	for _, baseline := range baselines {
		if baseline.Results.AvailabilityAchieved > 0 {
			values = append(values, baseline.Results.AvailabilityAchieved)
		}
	}

	return values
}

func (ta *TrendAnalyzer) createPerformanceTrend(name string, baselines []*BaselineSnapshot, values []float64, higherIsWorse bool) *PerformanceTrend {
	if len(baselines) != len(values) {
		return nil
	}

	slope := ta.calculateLinearTrendSlope(baselines, values)

	correlation := ta.calculateCorrelation(baselines, values)

	return &PerformanceTrend{
		MetricName: name,

		Direction: ta.determineTrendDirection(values, higherIsWorse),

		TrendSlope: slope,

		Correlation: correlation,

		Volatility: ta.calculateStability(values),

		HistoricalValues: ta.createTrendDataPoints(baselines, values),

		PredictedValue: ta.predictNextValue(values),
	}
}

func (ta *TrendAnalyzer) createTrendDataPoints(baselines []*BaselineSnapshot, values []float64) []TrendDataPoint {
	if len(baselines) != len(values) {
		return []TrendDataPoint{}
	}

	points := make([]TrendDataPoint, len(baselines))

	for i := range baselines {
		points[i] = TrendDataPoint{
			Timestamp: baselines[i].Timestamp,

			Value: values[i],
		}
	}

	return points
}
