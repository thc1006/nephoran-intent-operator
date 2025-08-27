// Package validation provides regression detection engine for identifying quality degradations
package validation

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
)

// RegressionDetectionEngine analyzes current results against baselines to detect regressions
type RegressionDetectionEngine struct {
	config *RegressionConfig

	// Thresholds for different regression severities
	severityThresholds map[string]float64
}

// NewRegressionDetectionEngine creates a new regression detection engine
func NewRegressionDetectionEngine(config *RegressionConfig) *RegressionDetectionEngine {
	return &RegressionDetectionEngine{
		config: config,
		severityThresholds: map[string]float64{
			"low":      5.0,  // 5% degradation
			"medium":   15.0, // 15% degradation
			"high":     30.0, // 30% degradation
			"critical": 50.0, // 50% degradation
		},
	}
}

// DetectRegressions analyzes current results against baseline to detect regressions
func (rde *RegressionDetectionEngine) DetectRegressions(baseline *BaselineSnapshot, current *ValidationResults) (*RegressionDetection, error) {
	ginkgo.By("Analyzing results for regression detection")

	detection := &RegressionDetection{
		HasRegression:           false,
		RegressionSeverity:      "none",
		BaselineID:              baseline.ID,
		ComparisonTime:          time.Now(),
		ExecutionContext:        "regression-analysis",
		PerformanceRegressions:  []*PerformanceRegression{},
		FunctionalRegressions:   []*FunctionalRegression{},
		SecurityRegressions:     []*SecurityRegression{},
		ProductionRegressions:   []*ProductionRegression{},
		StatisticalSignificance: 95.0,
		ConfidenceLevel:         rde.config.StatisticalConfidence,
	}

	// Detect performance regressions
	perfRegressions := rde.detectPerformanceRegressions(baseline, current)
	detection.PerformanceRegressions = perfRegressions

	// Detect functional regressions
	funcRegressions := rde.detectFunctionalRegressions(baseline, current)
	detection.FunctionalRegressions = funcRegressions

	// Detect security regressions
	secRegressions := rde.detectSecurityRegressions(baseline, current)
	detection.SecurityRegressions = secRegressions

	// Detect production readiness regressions
	prodRegressions := rde.detectProductionRegressions(baseline, current)
	detection.ProductionRegressions = prodRegressions

	// Determine overall regression status and severity
	detection.HasRegression = len(perfRegressions) > 0 || len(funcRegressions) > 0 ||
		len(secRegressions) > 0 || len(prodRegressions) > 0

	if detection.HasRegression {
		detection.RegressionSeverity = rde.calculateOverallSeverity(detection)
	}

	ginkgo.By(fmt.Sprintf("Regression detection completed: %s",
		func() string {
			if detection.HasRegression {
				return fmt.Sprintf("REGRESSION DETECTED (%s severity)", detection.RegressionSeverity)
			}
			return "NO REGRESSION"
		}()))

	return detection, nil
}

// detectPerformanceRegressions identifies performance degradations
func (rde *RegressionDetectionEngine) detectPerformanceRegressions(baseline *BaselineSnapshot, current *ValidationResults) []*PerformanceRegression {
	var regressions []*PerformanceRegression

	// Check P95 Latency regression
	if baseline.Results.P95Latency > 0 && current.P95Latency > 0 {
		degradation := rde.calculatePercentDegradation(
			float64(baseline.Results.P95Latency.Nanoseconds()),
			float64(current.P95Latency.Nanoseconds()),
		)

		if degradation > rde.config.PerformanceThreshold {
			regression := &PerformanceRegression{
				MetricName:         "P95 Latency",
				BaselineValue:      float64(baseline.Results.P95Latency.Nanoseconds()),
				CurrentValue:       float64(current.P95Latency.Nanoseconds()),
				DegradationPercent: degradation,
				Severity:           rde.classifySeverity(degradation),
				Impact:             rde.describeLatencyImpact(degradation, baseline.Results.P95Latency, current.P95Latency),
				Recommendation:     rde.getLatencyRecommendation(degradation),
				DetectionTime:      time.Now(),
			}
			regressions = append(regressions, regression)
		}
	}

	// Check P99 Latency regression
	if baseline.Results.P99Latency > 0 && current.P99Latency > 0 {
		degradation := rde.calculatePercentDegradation(
			float64(baseline.Results.P99Latency.Nanoseconds()),
			float64(current.P99Latency.Nanoseconds()),
		)

		if degradation > rde.config.PerformanceThreshold {
			regression := &PerformanceRegression{
				MetricName:         "P99 Latency",
				BaselineValue:      float64(baseline.Results.P99Latency.Nanoseconds()),
				CurrentValue:       float64(current.P99Latency.Nanoseconds()),
				DegradationPercent: degradation,
				Severity:           rde.classifySeverity(degradation),
				Impact:             rde.describeLatencyImpact(degradation, baseline.Results.P99Latency, current.P99Latency),
				Recommendation:     rde.getLatencyRecommendation(degradation),
				DetectionTime:      time.Now(),
			}
			regressions = append(regressions, regression)
		}
	}

	// Check Throughput regression (inverted - lower is worse)
	if baseline.Results.ThroughputAchieved > 0 && current.ThroughputAchieved > 0 {
		// For throughput, we want to detect decreases (negative degradation)
		if current.ThroughputAchieved < baseline.Results.ThroughputAchieved {
			degradation := ((baseline.Results.ThroughputAchieved - current.ThroughputAchieved) / baseline.Results.ThroughputAchieved) * 100

			if degradation > rde.config.PerformanceThreshold {
				regression := &PerformanceRegression{
					MetricName:         "Throughput",
					BaselineValue:      baseline.Results.ThroughputAchieved,
					CurrentValue:       current.ThroughputAchieved,
					DegradationPercent: degradation,
					Severity:           rde.classifySeverity(degradation),
					Impact:             rde.describeThroughputImpact(degradation, baseline.Results.ThroughputAchieved, current.ThroughputAchieved),
					Recommendation:     rde.getThroughputRecommendation(degradation),
					DetectionTime:      time.Now(),
				}
				regressions = append(regressions, regression)
			}
		}
	}

	// Check Availability regression (inverted - lower is worse)
	if baseline.Results.AvailabilityAchieved > 0 && current.AvailabilityAchieved > 0 {
		if current.AvailabilityAchieved < baseline.Results.AvailabilityAchieved {
			reduction := baseline.Results.AvailabilityAchieved - current.AvailabilityAchieved

			// Any availability reduction is significant
			if reduction > 0.01 { // 0.01% reduction threshold
				regression := &PerformanceRegression{
					MetricName:         "Availability",
					BaselineValue:      baseline.Results.AvailabilityAchieved,
					CurrentValue:       current.AvailabilityAchieved,
					DegradationPercent: (reduction / baseline.Results.AvailabilityAchieved) * 100,
					Severity:           "high", // Availability regression is always high severity
					Impact:             rde.describeAvailabilityImpact(reduction, baseline.Results.AvailabilityAchieved, current.AvailabilityAchieved),
					Recommendation:     rde.getAvailabilityRecommendation(reduction),
					DetectionTime:      time.Now(),
				}
				regressions = append(regressions, regression)
			}
		}
	}

	return regressions
}

// detectFunctionalRegressions identifies functional test failures and degradations
func (rde *RegressionDetectionEngine) detectFunctionalRegressions(baseline *BaselineSnapshot, current *ValidationResults) []*FunctionalRegression {
	var regressions []*FunctionalRegression

	// Check overall functional score regression
	if baseline.Results.FunctionalScore > 0 && current.FunctionalScore > 0 {
		reduction := float64(baseline.Results.FunctionalScore - current.FunctionalScore)
		reductionPercent := (reduction / float64(baseline.Results.FunctionalScore)) * 100

		if reductionPercent > rde.config.FunctionalThreshold {
			regression := &FunctionalRegression{
				TestCategory:     "Overall Functional",
				BaselinePassRate: (float64(baseline.Results.FunctionalScore) / 50.0) * 100, // Assuming 50 is max functional score
				CurrentPassRate:  (float64(current.FunctionalScore) / 50.0) * 100,
				FailedTests:      rde.identifyFailedFunctionalTests(baseline, current),
				NewFailures:      rde.identifyNewFunctionalFailures(baseline, current),
				Severity:         rde.classifySeverity(reductionPercent),
				Impact:           rde.describeFunctionalImpact(reductionPercent, baseline.Results.FunctionalScore, current.FunctionalScore),
				Recommendation:   rde.getFunctionalRecommendation(reductionPercent),
				DetectionTime:    time.Now(),
			}
			regressions = append(regressions, regression)
		}
	}

	// Check specific test category regressions
	categories := []string{"intent-processing", "llm-rag", "porch-integration", "multi-cluster", "oran-interfaces"}
	for _, category := range categories {
		if baselineResult, exists := baseline.Results.TestResults[category]; exists {
			if currentResult, exists := current.TestResults[category]; exists {
				baselinePassRate := (float64(baselineResult.PassedTests) / float64(baselineResult.TotalTests)) * 100
				currentPassRate := (float64(currentResult.PassedTests) / float64(currentResult.TotalTests)) * 100

				reduction := baselinePassRate - currentPassRate
				if reduction > rde.config.FunctionalThreshold {
					regression := &FunctionalRegression{
						TestCategory:     category,
						BaselinePassRate: baselinePassRate,
						CurrentPassRate:  currentPassRate,
						FailedTests:      rde.extractFailedTestNames(currentResult),
						NewFailures:      rde.compareTestFailures(baselineResult, currentResult),
						Severity:         rde.classifySeverity(reduction),
						Impact:           rde.describeCategoryImpact(category, reduction),
						Recommendation:   rde.getCategoryRecommendation(category, reduction),
						DetectionTime:    time.Now(),
					}
					regressions = append(regressions, regression)
				}
			}
		}
	}

	return regressions
}

// detectSecurityRegressions identifies new security vulnerabilities and compliance failures
func (rde *RegressionDetectionEngine) detectSecurityRegressions(baseline *BaselineSnapshot, current *ValidationResults) []*SecurityRegression {
	var regressions []*SecurityRegression

	// Check for new security vulnerabilities
	baselineFindings := make(map[string]*SecurityFinding)
	for _, finding := range baseline.Results.SecurityFindings {
		baselineFindings[finding.Type] = finding
	}

	for _, currentFinding := range current.SecurityFindings {
		if baselineFinding, exists := baselineFindings[currentFinding.Type]; exists {
			// Existing finding - check for severity increase
			if rde.isSecuritySeverityWorse(currentFinding.Severity, baselineFinding.Severity) {
				regression := &SecurityRegression{
					Finding:            currentFinding,
					IsNewVulnerability: false,
					SeverityChange:     fmt.Sprintf("%s → %s", baselineFinding.Severity, currentFinding.Severity),
					Impact:             rde.describeSecurityImpact(currentFinding, false),
					Recommendation:     rde.getSecurityRecommendation(currentFinding),
					DetectionTime:      time.Now(),
				}
				regressions = append(regressions, regression)
			}
		} else {
			// New security finding
			if !currentFinding.Passed {
				regression := &SecurityRegression{
					Finding:            currentFinding,
					IsNewVulnerability: true,
					SeverityChange:     fmt.Sprintf("None → %s", currentFinding.Severity),
					Impact:             rde.describeSecurityImpact(currentFinding, true),
					Recommendation:     rde.getSecurityRecommendation(currentFinding),
					DetectionTime:      time.Now(),
				}
				regressions = append(regressions, regression)
			}
		}
	}

	// Check overall security score regression
	if baseline.Results.SecurityScore > current.SecurityScore {
		reduction := baseline.Results.SecurityScore - current.SecurityScore
		if reduction > 0 { // Any security score reduction is a regression
			// Create a general security regression if no specific findings explain the drop
			if len(regressions) == 0 {
				regression := &SecurityRegression{
					Finding: &SecurityFinding{
						Type:        "General Security",
						Severity:    "medium",
						Description: fmt.Sprintf("Overall security score decreased from %d to %d", baseline.Results.SecurityScore, current.SecurityScore),
						Component:   "Security Framework",
						Remediation: "Review all security test failures and compliance issues",
						Passed:      false,
					},
					IsNewVulnerability: false,
					SeverityChange:     "Degradation",
					Impact:             "Overall security posture has deteriorated",
					Recommendation:     "Conduct comprehensive security review and address all failing security tests",
					DetectionTime:      time.Now(),
				}
				regressions = append(regressions, regression)
			}
		}
	}

	return regressions
}

// detectProductionRegressions identifies production readiness degradations
func (rde *RegressionDetectionEngine) detectProductionRegressions(baseline *BaselineSnapshot, current *ValidationResults) []*ProductionRegression {
	var regressions []*ProductionRegression

	// Check production readiness score regression
	if baseline.Results.ProductionScore > current.ProductionScore {
		reduction := float64(baseline.Results.ProductionScore - current.ProductionScore)
		if reduction > 0 { // Any production score reduction is significant
			regression := &ProductionRegression{
				Metric:         "Production Readiness Score",
				BaselineValue:  float64(baseline.Results.ProductionScore),
				CurrentValue:   float64(current.ProductionScore),
				Impact:         rde.describeProductionScoreImpact(reduction),
				Recommendation: rde.getProductionScoreRecommendation(reduction),
				DetectionTime:  time.Now(),
			}
			regressions = append(regressions, regression)
		}
	}

	// Check specific production metrics if available
	if baseline.ProductionBaselines != nil && current.ReliabilityMetrics != nil {
		// MTTR regression (higher is worse)
		if current.ReliabilityMetrics.MTTR > baseline.ProductionBaselines.MTTR {
			increase := float64(current.ReliabilityMetrics.MTTR-baseline.ProductionBaselines.MTTR) / float64(baseline.ProductionBaselines.MTTR) * 100
			if increase > rde.config.ProductionThreshold {
				regression := &ProductionRegression{
					Metric:         "Mean Time To Recovery (MTTR)",
					BaselineValue:  float64(baseline.ProductionBaselines.MTTR.Seconds()),
					CurrentValue:   float64(current.ReliabilityMetrics.MTTR.Seconds()),
					Impact:         "Increased recovery time affects service reliability",
					Recommendation: "Review incident response procedures and automation",
					DetectionTime:  time.Now(),
				}
				regressions = append(regressions, regression)
			}
		}

		// Error rate regression (higher is worse)
		if current.ReliabilityMetrics.ErrorRate > baseline.ProductionBaselines.ErrorRate {
			increase := ((current.ReliabilityMetrics.ErrorRate - baseline.ProductionBaselines.ErrorRate) / baseline.ProductionBaselines.ErrorRate) * 100
			if increase > rde.config.ProductionThreshold {
				regression := &ProductionRegression{
					Metric:         "Error Rate",
					BaselineValue:  baseline.ProductionBaselines.ErrorRate,
					CurrentValue:   current.ReliabilityMetrics.ErrorRate,
					Impact:         "Increased error rate affects user experience and system reliability",
					Recommendation: "Investigate root causes of increased errors and implement fixes",
					DetectionTime:  time.Now(),
				}
				regressions = append(regressions, regression)
			}
		}
	}

	return regressions
}

// Helper methods for impact description and recommendations

func (rde *RegressionDetectionEngine) calculatePercentDegradation(baseline, current float64) float64 {
	if baseline <= 0 {
		return 0
	}
	return ((current - baseline) / baseline) * 100
}

func (rde *RegressionDetectionEngine) classifySeverity(degradationPercent float64) string {
	absPercent := math.Abs(degradationPercent)

	if absPercent >= rde.severityThresholds["critical"] {
		return "critical"
	} else if absPercent >= rde.severityThresholds["high"] {
		return "high"
	} else if absPercent >= rde.severityThresholds["medium"] {
		return "medium"
	}
	return "low"
}

func (rde *RegressionDetectionEngine) calculateOverallSeverity(detection *RegressionDetection) string {
	maxSeverity := "low"
	severityOrder := map[string]int{"low": 1, "medium": 2, "high": 3, "critical": 4}

	// Check performance regressions
	for _, reg := range detection.PerformanceRegressions {
		if severityOrder[reg.Severity] > severityOrder[maxSeverity] {
			maxSeverity = reg.Severity
		}
	}

	// Check functional regressions
	for _, reg := range detection.FunctionalRegressions {
		if severityOrder[reg.Severity] > severityOrder[maxSeverity] {
			maxSeverity = reg.Severity
		}
	}

	// Security regressions are always at least medium severity
	if len(detection.SecurityRegressions) > 0 && severityOrder[maxSeverity] < severityOrder["medium"] {
		maxSeverity = "medium"
	}

	// Production regressions are always at least high severity
	if len(detection.ProductionRegressions) > 0 && severityOrder[maxSeverity] < severityOrder["high"] {
		maxSeverity = "high"
	}

	return maxSeverity
}

// Impact description methods
func (rde *RegressionDetectionEngine) describeLatencyImpact(degradation float64, baseline, current time.Duration) string {
	return fmt.Sprintf("Latency increased by %.1f%% (%v → %v), potentially affecting user experience",
		degradation, baseline, current)
}

func (rde *RegressionDetectionEngine) describeThroughputImpact(reduction, baseline, current float64) string {
	return fmt.Sprintf("Throughput decreased by %.1f%% (%.1f → %.1f intents/min), reducing system capacity",
		reduction, baseline, current)
}

func (rde *RegressionDetectionEngine) describeAvailabilityImpact(reduction, baseline, current float64) string {
	return fmt.Sprintf("Availability decreased by %.2f%% (%.2f%% → %.2f%%), violating SLA targets",
		reduction, baseline, current)
}

func (rde *RegressionDetectionEngine) describeFunctionalImpact(reductionPercent float64, baseline, current int) string {
	return fmt.Sprintf("Functional tests degraded by %.1f%% (%d → %d points), indicating feature regressions",
		reductionPercent, baseline, current)
}

func (rde *RegressionDetectionEngine) describeCategoryImpact(category string, reduction float64) string {
	return fmt.Sprintf("%s functionality degraded by %.1f%%, affecting system reliability",
		strings.ReplaceAll(category, "-", " "), reduction)
}

func (rde *RegressionDetectionEngine) describeSecurityImpact(finding *SecurityFinding, isNew bool) string {
	if isNew {
		return fmt.Sprintf("New %s security vulnerability detected: %s", finding.Severity, finding.Description)
	}
	return fmt.Sprintf("Security vulnerability severity increased: %s", finding.Description)
}

func (rde *RegressionDetectionEngine) describeProductionScoreImpact(reduction float64) string {
	return fmt.Sprintf("Production readiness score decreased by %.0f points, affecting deployment confidence", reduction)
}

// Recommendation methods
func (rde *RegressionDetectionEngine) getLatencyRecommendation(degradation float64) string {
	if degradation > 50 {
		return "Critical latency regression - investigate immediately for performance bottlenecks, resource constraints, or algorithmic changes"
	} else if degradation > 25 {
		return "Significant latency increase - profile application performance and optimize critical paths"
	}
	return "Monitor latency trends and consider performance optimizations"
}

func (rde *RegressionDetectionEngine) getThroughputRecommendation(reduction float64) string {
	if reduction > 50 {
		return "Critical throughput loss - check for resource limits, bottlenecks, or concurrency issues"
	} else if reduction > 25 {
		return "Significant throughput reduction - analyze load balancing and resource allocation"
	}
	return "Monitor throughput patterns and consider capacity optimizations"
}

func (rde *RegressionDetectionEngine) getAvailabilityRecommendation(reduction float64) string {
	return "Availability regression detected - review health checks, failover mechanisms, and incident response procedures immediately"
}

func (rde *RegressionDetectionEngine) getFunctionalRecommendation(reductionPercent float64) string {
	if reductionPercent > 25 {
		return "Critical functional regression - halt deployments and investigate failing tests immediately"
	}
	return "Review and fix failing functional tests before next release"
}

func (rde *RegressionDetectionEngine) getCategoryRecommendation(category string, reduction float64) string {
	return fmt.Sprintf("Review and fix %s test failures - focus on critical path functionality",
		strings.ReplaceAll(category, "-", " "))
}

func (rde *RegressionDetectionEngine) getSecurityRecommendation(finding *SecurityFinding) string {
	return fmt.Sprintf("Address security issue immediately: %s. %s", finding.Description, finding.Remediation)
}

func (rde *RegressionDetectionEngine) getProductionScoreRecommendation(reduction float64) string {
	return "Review production readiness checklist and address failing reliability, monitoring, or disaster recovery tests"
}

// Helper methods for test analysis
func (rde *RegressionDetectionEngine) identifyFailedFunctionalTests(baseline *BaselineSnapshot, current *ValidationResults) []string {
	// This would extract failed test names from current results
	return []string{} // Placeholder
}

func (rde *RegressionDetectionEngine) identifyNewFunctionalFailures(baseline *BaselineSnapshot, current *ValidationResults) []string {
	// This would identify tests that passed in baseline but failed in current
	return []string{} // Placeholder
}

func (rde *RegressionDetectionEngine) extractFailedTestNames(result *TestCategoryResult) []string {
	var failedTests []string
	for _, detail := range result.Details {
		if detail.Status == "failed" {
			failedTests = append(failedTests, detail.TestName)
		}
	}
	return failedTests
}

func (rde *RegressionDetectionEngine) compareTestFailures(baseline, current *TestCategoryResult) []string {
	baselineFailures := make(map[string]bool)
	for _, detail := range baseline.Details {
		if detail.Status == "failed" {
			baselineFailures[detail.TestName] = true
		}
	}

	var newFailures []string
	for _, detail := range current.Details {
		if detail.Status == "failed" && !baselineFailures[detail.TestName] {
			newFailures = append(newFailures, detail.TestName)
		}
	}

	return newFailures
}

func (rde *RegressionDetectionEngine) isSecuritySeverityWorse(current, baseline string) bool {
	severityOrder := map[string]int{
		"low":      1,
		"medium":   2,
		"high":     3,
		"critical": 4,
	}

	return severityOrder[strings.ToLower(current)] > severityOrder[strings.ToLower(baseline)]
}
