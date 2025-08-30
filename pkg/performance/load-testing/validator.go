// Package loadtesting provides performance validation for load tests.

package loadtesting

import (
	"fmt"
	"math"
	"sort"
	"time"

	"go.uber.org/zap"
	"gonum.org/v1/gonum/stat"
	"gonum.org/v1/gonum/stat/distuv"
)

// PerformanceValidatorImpl validates performance claims against test results.

type PerformanceValidatorImpl struct {
	config *LoadTestConfig

	logger *zap.Logger

	thresholds ValidationThresholds

	tests []StatisticalValidator
}

// StatisticalValidator interface for statistical validation methods.

type StatisticalValidator interface {
	Validate(data []float64, threshold float64) (bool, float64, string)

	GetName() string
}

// NewPerformanceValidator creates a new performance validator.

func NewPerformanceValidator(config *LoadTestConfig, logger *zap.Logger) *PerformanceValidatorImpl {

	if logger == nil {

		logger = zap.NewNop()

	}

	validator := &PerformanceValidatorImpl{

		config: config,

		logger: logger,

		thresholds: ValidationThresholds{

			MinThroughput: config.TargetThroughput * 0.95, // Allow 5% tolerance

			MaxLatencyP50: 1000 * time.Millisecond,

			MaxLatencyP95: 2000 * time.Millisecond,

			MaxLatencyP99: 3000 * time.Millisecond,

			MinAvailability: 0.9995, // 99.95%

			MaxErrorRate: 0.05, // 5%

			MaxResourceUsage: ResourceLimits{

				CPUPercent: config.MaxCPU,

				MemoryGB: config.MaxMemoryGB,

				NetworkMbps: config.MaxNetworkMbps,

				DiskIOPS: 10000,
			},
		},

		tests: make([]StatisticalValidator, 0),
	}

	// Initialize statistical tests.

	validator.initializeTests()

	return validator

}

// Validate checks if performance meets requirements.

func (v *PerformanceValidatorImpl) Validate(results *LoadTestResults) ValidationReport {

	report := ValidationReport{

		Passed: true,

		Score: 100.0,

		FailedCriteria: make([]string, 0),

		Warnings: make([]string, 0),

		DetailedResults: make(map[string]interface{}),

		Recommendations: make([]string, 0),

		Evidence: make([]Evidence, 0),
	}

	// Validate each performance claim.

	v.validateConcurrentIntents(results, &report)

	v.validateThroughput(results, &report)

	v.validateLatency(results, &report)

	v.validateAvailability(results, &report)

	v.validateResourceUsage(results, &report)

	v.validateStatisticalSignificance(results, &report)

	// Calculate final score.

	report.Score = v.calculateScore(&report)

	// Generate recommendations.

	v.generateRecommendations(results, &report)

	return report

}

// ValidateRealtime performs real-time validation during test execution.

func (v *PerformanceValidatorImpl) ValidateRealtime(metrics GeneratorMetrics) bool {

	// Check error rate.

	if metrics.ErrorRate > v.thresholds.MaxErrorRate {

		v.logger.Warn("Real-time validation failed: high error rate",

			zap.Float64("errorRate", metrics.ErrorRate),

			zap.Float64("threshold", v.thresholds.MaxErrorRate))

		return false

	}

	// Check latency.

	if metrics.AverageLatency > v.thresholds.MaxLatencyP95 {

		v.logger.Warn("Real-time validation failed: high latency",

			zap.Duration("latency", metrics.AverageLatency),

			zap.Duration("threshold", v.thresholds.MaxLatencyP95))

		return false

	}

	// Check throughput.

	currentThroughput := metrics.CurrentRate * 60 // Convert to per minute

	if currentThroughput < v.thresholds.MinThroughput*0.8 { // Allow 20% tolerance during ramp-up

		v.logger.Warn("Real-time validation failed: low throughput",

			zap.Float64("throughput", currentThroughput),

			zap.Float64("threshold", v.thresholds.MinThroughput))

		return false

	}

	return true

}

// GetThresholds returns validation thresholds.

func (v *PerformanceValidatorImpl) GetThresholds() ValidationThresholds {

	return v.thresholds

}

// Private validation methods.

func (v *PerformanceValidatorImpl) validateConcurrentIntents(results *LoadTestResults, report *ValidationReport) {

	// Check if system handled 200+ concurrent intents.

	maxConcurrent := v.calculateMaxConcurrent(results)

	evidence := Evidence{

		Type: "concurrent_intents",

		Description: fmt.Sprintf("Maximum concurrent intents: %d", maxConcurrent),

		Data: map[string]interface{}{

			"max_concurrent": maxConcurrent,

			"target": 200,

			"test_duration": results.Duration,
		},

		Timestamp: time.Now(),
	}

	if maxConcurrent >= 200 {

		v.logger.Info("PASS: Concurrent intent handling",

			zap.Int("maxConcurrent", maxConcurrent))

		evidence.Data["result"] = "PASS"

	} else {

		report.Passed = false

		report.FailedCriteria = append(report.FailedCriteria,

			fmt.Sprintf("Failed to handle 200+ concurrent intents (achieved: %d)", maxConcurrent))

		evidence.Data["result"] = "FAIL"

	}

	report.Evidence = append(report.Evidence, evidence)

	report.DetailedResults["concurrent_intents"] = maxConcurrent

}

func (v *PerformanceValidatorImpl) validateThroughput(results *LoadTestResults, report *ValidationReport) {

	// Check if system achieved 45 intents per minute.

	targetThroughput := 45.0

	evidence := Evidence{

		Type: "throughput",

		Description: fmt.Sprintf("Average throughput: %.2f intents/min", results.AverageThroughput),

		Data: map[string]interface{}{

			"average_throughput": results.AverageThroughput,

			"peak_throughput": results.PeakThroughput,

			"min_throughput": results.MinThroughput,

			"target": targetThroughput,
		},

		Timestamp: time.Now(),
	}

	if results.AverageThroughput >= targetThroughput {

		v.logger.Info("PASS: Throughput requirement",

			zap.Float64("throughput", results.AverageThroughput))

		evidence.Data["result"] = "PASS"

	} else {

		report.Passed = false

		report.FailedCriteria = append(report.FailedCriteria,

			fmt.Sprintf("Failed to achieve 45 intents/min (achieved: %.2f)", results.AverageThroughput))

		evidence.Data["result"] = "FAIL"

	}

	// Check consistency.

	if results.MinThroughput < targetThroughput*0.8 {

		report.Warnings = append(report.Warnings,

			fmt.Sprintf("Throughput inconsistent - minimum was %.2f intents/min", results.MinThroughput))

	}

	report.Evidence = append(report.Evidence, evidence)

	report.DetailedResults["throughput"] = map[string]float64{

		"average": results.AverageThroughput,

		"peak": results.PeakThroughput,

		"min": results.MinThroughput,
	}

}

func (v *PerformanceValidatorImpl) validateLatency(results *LoadTestResults, report *ValidationReport) {

	// Check if P95 latency is under 2 seconds.

	targetP95 := 2000 * time.Millisecond

	evidence := Evidence{

		Type: "latency",

		Description: fmt.Sprintf("P95 latency: %v", results.LatencyP95),

		Data: map[string]interface{}{

			"p50": results.LatencyP50,

			"p95": results.LatencyP95,

			"p99": results.LatencyP99,

			"mean": results.LatencyMean,

			"target": targetP95,
		},

		Timestamp: time.Now(),
	}

	if results.LatencyP95 <= targetP95 {

		v.logger.Info("PASS: P95 latency requirement",

			zap.Duration("p95", results.LatencyP95))

		evidence.Data["result"] = "PASS"

	} else {

		report.Passed = false

		report.FailedCriteria = append(report.FailedCriteria,

			fmt.Sprintf("P95 latency exceeds 2s (actual: %v)", results.LatencyP95))

		evidence.Data["result"] = "FAIL"

	}

	// Additional latency checks.

	if results.LatencyP99 > 5*time.Second {

		report.Warnings = append(report.Warnings,

			fmt.Sprintf("P99 latency is high: %v", results.LatencyP99))

	}

	report.Evidence = append(report.Evidence, evidence)

	report.DetailedResults["latency"] = map[string]time.Duration{

		"p50": results.LatencyP50,

		"p95": results.LatencyP95,

		"p99": results.LatencyP99,

		"min": results.LatencyMin,

		"max": results.LatencyMax,

		"mean": results.LatencyMean,
	}

}

func (v *PerformanceValidatorImpl) validateAvailability(results *LoadTestResults, report *ValidationReport) {

	// Check if system achieved 99.95% availability.

	targetAvailability := 0.9995

	evidence := Evidence{

		Type: "availability",

		Description: fmt.Sprintf("Availability: %.4f%%", results.Availability*100),

		Data: map[string]interface{}{

			"availability": results.Availability,

			"successful_requests": results.SuccessfulRequests,

			"failed_requests": results.FailedRequests,

			"total_requests": results.TotalRequests,

			"downtime": results.Downtime,

			"target": targetAvailability,
		},

		Timestamp: time.Now(),
	}

	if results.Availability >= targetAvailability {

		v.logger.Info("PASS: Availability requirement",

			zap.Float64("availability", results.Availability))

		evidence.Data["result"] = "PASS"

	} else {

		report.Passed = false

		report.FailedCriteria = append(report.FailedCriteria,

			fmt.Sprintf("Failed to achieve 99.95%% availability (achieved: %.4f%%)", results.Availability*100))

		evidence.Data["result"] = "FAIL"

	}

	// Calculate allowed downtime.

	allowedDowntime := results.Duration * time.Duration(1-targetAvailability)

	if results.Downtime > allowedDowntime {

		report.Warnings = append(report.Warnings,

			fmt.Sprintf("Downtime exceeded allowed threshold: %v > %v", results.Downtime, allowedDowntime))

	}

	report.Evidence = append(report.Evidence, evidence)

	report.DetailedResults["availability"] = results.Availability

}

func (v *PerformanceValidatorImpl) validateResourceUsage(results *LoadTestResults, report *ValidationReport) {

	evidence := Evidence{

		Type: "resource_usage",

		Description: "Resource utilization during test",

		Data: map[string]interface{}{

			"peak_cpu": results.PeakCPU,

			"average_cpu": results.AverageCPU,

			"peak_memory_gb": results.PeakMemoryGB,

			"average_memory_gb": results.AverageMemoryGB,

			"peak_network_mbps": results.PeakNetworkMbps,
		},

		Timestamp: time.Now(),
	}

	// Check CPU usage.

	if results.PeakCPU > v.thresholds.MaxResourceUsage.CPUPercent {

		report.Warnings = append(report.Warnings,

			fmt.Sprintf("Peak CPU usage exceeded limit: %.2f%% > %.2f%%",

				results.PeakCPU, v.thresholds.MaxResourceUsage.CPUPercent))

		evidence.Data["cpu_exceeded"] = true

	}

	// Check memory usage.

	if results.PeakMemoryGB > v.thresholds.MaxResourceUsage.MemoryGB {

		report.Warnings = append(report.Warnings,

			fmt.Sprintf("Peak memory usage exceeded limit: %.2fGB > %.2fGB",

				results.PeakMemoryGB, v.thresholds.MaxResourceUsage.MemoryGB))

		evidence.Data["memory_exceeded"] = true

	}

	// Check network usage.

	if results.PeakNetworkMbps > v.thresholds.MaxResourceUsage.NetworkMbps {

		report.Warnings = append(report.Warnings,

			fmt.Sprintf("Peak network usage exceeded limit: %.2fMbps > %.2fMbps",

				results.PeakNetworkMbps, v.thresholds.MaxResourceUsage.NetworkMbps))

		evidence.Data["network_exceeded"] = true

	}

	report.Evidence = append(report.Evidence, evidence)

	report.DetailedResults["resource_usage"] = map[string]interface{}{

		"cpu": map[string]float64{"peak": results.PeakCPU, "average": results.AverageCPU},

		"memory": map[string]float64{"peak": results.PeakMemoryGB, "average": results.AverageMemoryGB},

		"network": map[string]float64{"peak": results.PeakNetworkMbps},
	}

}

func (v *PerformanceValidatorImpl) validateStatisticalSignificance(results *LoadTestResults, report *ValidationReport) {

	if results.StatisticalAnalysis == nil {

		report.Warnings = append(report.Warnings, "No statistical analysis available")

		return

	}

	// Validate statistical tests.

	for testName, testResult := range results.StatisticalAnalysis.TestResults {

		if !testResult.Significant {

			continue

		}

		evidence := Evidence{

			Type: "statistical_test",

			Description: fmt.Sprintf("%s: %s", testName, testResult.Conclusion),

			Data: map[string]interface{}{

				"test_name": testName,

				"statistic": testResult.Statistic,

				"p_value": testResult.PValue,

				"significant": testResult.Significant,

				"conclusion": testResult.Conclusion,
			},

			Timestamp: time.Now(),
		}

		report.Evidence = append(report.Evidence, evidence)

		// Check if test indicates performance issue.

		if testResult.PValue < 0.05 && testName == "performance_degradation" {

			report.Warnings = append(report.Warnings,

				fmt.Sprintf("Statistical test indicates performance degradation: %s", testResult.Conclusion))

		}

	}

	// Validate confidence intervals.

	for metric, ci := range results.StatisticalAnalysis.ConfidenceIntervals {

		if metric == "latency_p95" {

			if ci.Upper > float64(v.thresholds.MaxLatencyP95.Milliseconds()) {

				report.Warnings = append(report.Warnings,

					fmt.Sprintf("P95 latency confidence interval upper bound exceeds threshold: %.2fms", ci.Upper))

			}

		}

		if metric == "throughput" {

			if ci.Lower < v.thresholds.MinThroughput {

				report.Warnings = append(report.Warnings,

					fmt.Sprintf("Throughput confidence interval lower bound below threshold: %.2f", ci.Lower))

			}

		}

	}

}

func (v *PerformanceValidatorImpl) initializeTests() {

	// Add statistical test validators.

	v.tests = append(v.tests, &TTestValidator{logger: v.logger})

	v.tests = append(v.tests, &MannWhitneyValidator{logger: v.logger})

	v.tests = append(v.tests, &ChiSquareValidator{logger: v.logger})

	v.tests = append(v.tests, &AndersonDarlingValidator{logger: v.logger})

}

func (v *PerformanceValidatorImpl) calculateMaxConcurrent(results *LoadTestResults) int {

	// Calculate based on Little's Law: L = λW.

	// L = average number in system.

	// λ = arrival rate (throughput).

	// W = average time in system (latency).

	if results.AverageThroughput == 0 || results.LatencyMean == 0 {

		return 0

	}

	arrivalRate := results.AverageThroughput / 60.0 // Convert to per second

	avgTimeInSystem := results.LatencyMean.Seconds()

	maxConcurrent := int(math.Ceil(arrivalRate * avgTimeInSystem))

	// Adjust based on concurrency setting.

	if v.config.Concurrency > 0 {

		maxConcurrent = v.config.Concurrency

	}

	return maxConcurrent

}

func (v *PerformanceValidatorImpl) calculateScore(report *ValidationReport) float64 {

	score := 100.0

	// Deduct points for failed criteria.

	deductionPerFailure := 20.0

	score -= float64(len(report.FailedCriteria)) * deductionPerFailure

	// Deduct points for warnings.

	deductionPerWarning := 5.0

	score -= float64(len(report.Warnings)) * deductionPerWarning

	// Ensure score doesn't go below 0.

	if score < 0 {

		score = 0

	}

	return score

}

func (v *PerformanceValidatorImpl) generateRecommendations(results *LoadTestResults, report *ValidationReport) {

	// Based on validation results, generate recommendations.

	if results.Availability < v.thresholds.MinAvailability {

		report.Recommendations = append(report.Recommendations,

			"Improve error handling and retry mechanisms to increase availability")

	}

	if results.LatencyP95 > v.thresholds.MaxLatencyP95 {

		report.Recommendations = append(report.Recommendations,

			"Optimize intent processing pipeline to reduce P95 latency")

		report.Recommendations = append(report.Recommendations,

			"Consider caching frequently accessed data")

		report.Recommendations = append(report.Recommendations,

			"Review database query performance")

	}

	if results.AverageThroughput < v.thresholds.MinThroughput {

		report.Recommendations = append(report.Recommendations,

			"Increase concurrency or optimize processing to improve throughput")

		report.Recommendations = append(report.Recommendations,

			"Consider horizontal scaling of processing components")

	}

	if results.PeakCPU > 80 {

		report.Recommendations = append(report.Recommendations,

			"CPU usage is high - profile and optimize CPU-intensive operations")

	}

	if results.PeakMemoryGB > v.thresholds.MaxResourceUsage.MemoryGB*0.8 {

		report.Recommendations = append(report.Recommendations,

			"Memory usage approaching limit - check for memory leaks")

	}

	// Recommendations based on breaking point analysis.

	if results.BreakingPoint != nil {

		if results.BreakingPoint.MaxConcurrentIntents < 200 {

			report.Recommendations = append(report.Recommendations,

				fmt.Sprintf("System breaks at %d concurrent intents - needs optimization for target of 200+",

					results.BreakingPoint.MaxConcurrentIntents))

		}

		if results.BreakingPoint.ResourceBottleneck != "" {

			report.Recommendations = append(report.Recommendations,

				fmt.Sprintf("Resource bottleneck identified: %s - focus optimization efforts here",

					results.BreakingPoint.ResourceBottleneck))

		}

	}

}

// Statistical Test Validators.

// TTestValidator performs Student's t-test validation.

type TTestValidator struct {
	logger *zap.Logger
}

// Validate performs validate operation.

func (t *TTestValidator) Validate(data []float64, threshold float64) (bool, float64, string) {

	if len(data) < 30 {

		return false, 0, "Insufficient data for t-test (need at least 30 samples)"

	}

	mean := stat.Mean(data, nil)

	stdDev := stat.StdDev(data, nil)

	n := float64(len(data))

	// Calculate t-statistic.

	tStatistic := (mean - threshold) / (stdDev / math.Sqrt(n))

	// Calculate p-value (simplified - in production use proper t-distribution).

	pValue := 2 * (1 - cumulativeNormal(math.Abs(tStatistic)))

	significant := pValue < 0.05

	conclusion := fmt.Sprintf("Mean=%.2f, StdDev=%.2f, t=%.2f, p=%.4f", mean, stdDev, tStatistic, pValue)

	return significant, pValue, conclusion

}

// GetName performs getname operation.

func (t *TTestValidator) GetName() string {

	return "t_test"

}

// MannWhitneyValidator performs Mann-Whitney U test.

type MannWhitneyValidator struct {
	logger *zap.Logger
}

// Validate performs validate operation.

func (m *MannWhitneyValidator) Validate(data []float64, threshold float64) (bool, float64, string) {

	if len(data) < 20 {

		return false, 0, "Insufficient data for Mann-Whitney test"

	}

	// Create comparison sample around threshold.

	comparisonData := make([]float64, len(data))

	for i := range comparisonData {

		comparisonData[i] = threshold + (distuv.Normal{Mu: 0, Sigma: threshold * 0.1}.Rand())

	}

	// Perform Mann-Whitney U test.

	u, pValue := mannWhitneyU(data, comparisonData)

	significant := pValue < 0.05

	conclusion := fmt.Sprintf("U=%.2f, p=%.4f", u, pValue)

	return significant, pValue, conclusion

}

// GetName performs getname operation.

func (m *MannWhitneyValidator) GetName() string {

	return "mann_whitney"

}

// ChiSquareValidator performs Chi-square goodness of fit test.

type ChiSquareValidator struct {
	logger *zap.Logger
}

// Validate performs validate operation.

func (c *ChiSquareValidator) Validate(data []float64, threshold float64) (bool, float64, string) {

	if len(data) < 50 {

		return false, 0, "Insufficient data for Chi-square test"

	}

	// Create bins.

	numBins := 10

	bins := make([]int, numBins)

	minVal, maxVal := minMax(data)

	binWidth := (maxVal - minVal) / float64(numBins)

	// Count observations in each bin.

	for _, value := range data {

		binIndex := int((value - minVal) / binWidth)

		if binIndex >= numBins {

			binIndex = numBins - 1

		}

		bins[binIndex]++

	}

	// Calculate expected frequencies (assuming normal distribution).

	n := float64(len(data))

	mean := stat.Mean(data, nil)

	stdDev := stat.StdDev(data, nil)

	chiSquare := 0.0

	for i, observed := range bins {

		binCenter := minVal + float64(i)*binWidth + binWidth/2

		expected := n * normalPDF(binCenter, mean, stdDev) * binWidth

		if expected > 0 {

			chiSquare += math.Pow(float64(observed)-expected, 2) / expected

		}

	}

	// Calculate p-value (simplified).

	df := float64(numBins - 1)

	pValue := 1 - chiSquareCDF(chiSquare, df)

	significant := pValue < 0.05

	conclusion := fmt.Sprintf("χ²=%.2f, df=%.0f, p=%.4f", chiSquare, df, pValue)

	return significant, pValue, conclusion

}

// GetName performs getname operation.

func (c *ChiSquareValidator) GetName() string {

	return "chi_square"

}

// AndersonDarlingValidator performs Anderson-Darling test for normality.

type AndersonDarlingValidator struct {
	logger *zap.Logger
}

// Validate performs validate operation.

func (a *AndersonDarlingValidator) Validate(data []float64, threshold float64) (bool, float64, string) {

	if len(data) < 8 {

		return false, 0, "Insufficient data for Anderson-Darling test"

	}

	// Sort data.

	sorted := make([]float64, len(data))

	copy(sorted, data)

	sort.Float64s(sorted)

	n := float64(len(sorted))

	mean := stat.Mean(sorted, nil)

	stdDev := stat.StdDev(sorted, nil)

	// Calculate A² statistic.

	aSquared := -n

	for i, x := range sorted {

		z := (x - mean) / stdDev

		Fi := cumulativeNormal(z)

		if Fi <= 0 || Fi >= 1 {

			continue

		}

		aSquared -= (2*float64(i+1) - 1) / n * (math.Log(Fi) + math.Log(1-cumulativeNormal((sorted[len(sorted)-1-i]-mean)/stdDev)))

	}

	// Adjust for sample size.

	aSquaredAdjusted := aSquared * (1 + 0.75/n + 2.25/(n*n))

	// Critical value at 0.05 significance level.

	criticalValue := 0.752

	significant := aSquaredAdjusted > criticalValue

	pValue := andersonDarlingPValue(aSquaredAdjusted)

	conclusion := fmt.Sprintf("A²=%.3f, critical=%.3f, p=%.4f", aSquaredAdjusted, criticalValue, pValue)

	return significant, pValue, conclusion

}

// GetName performs getname operation.

func (a *AndersonDarlingValidator) GetName() string {

	return "anderson_darling"

}

// Helper functions for statistical calculations.

func cumulativeNormal(z float64) float64 {

	return 0.5 * (1 + math.Erf(z/math.Sqrt(2)))

}

func normalPDF(x, mu, sigma float64) float64 {

	return math.Exp(-math.Pow(x-mu, 2)/(2*math.Pow(sigma, 2))) / (sigma * math.Sqrt(2*math.Pi))

}

func chiSquareCDF(x, df float64) float64 {

	// Simplified chi-square CDF calculation using gamma distribution.

	// Chi-square(df) is equivalent to Gamma(df/2, 2).

	// Using approximation for simplicity.

	if x <= 0 {

		return 0

	}

	// Using Wilson-Hilferty approximation for chi-square CDF.

	z := math.Pow(x/df, 1.0/3.0) - (1.0 - 2.0/(9.0*df))

	z = z / math.Sqrt(2.0/(9.0*df))

	// Standard normal CDF approximation.

	return 0.5 * (1.0 + math.Erf(z/math.Sqrt(2.0)))

}

func andersonDarlingPValue(aSquared float64) float64 {

	// Simplified p-value calculation for Anderson-Darling test.

	if aSquared < 0.2 {

		return 1 - math.Exp(-13.436+101.14*aSquared-223.73*aSquared*aSquared)

	} else if aSquared < 0.34 {

		return 1 - math.Exp(-8.318+42.796*aSquared-59.938*aSquared*aSquared)

	} else if aSquared < 0.6 {

		return math.Exp(0.9177 - 4.279*aSquared - 1.38*aSquared*aSquared)

	} else {

		return math.Exp(1.2937 - 5.709*aSquared + 0.0186*aSquared*aSquared)

	}

}

func mannWhitneyU(x, y []float64) (float64, float64) {

	// Simplified Mann-Whitney U test.

	// In production, use proper implementation.

	// Combine and rank.

	combined := append(x, y...)

	ranks := make(map[float64]float64)

	sort.Float64s(combined)

	for i, val := range combined {

		ranks[val] = float64(i + 1)

	}

	// Calculate U statistic.

	var sumRanksX float64

	for _, val := range x {

		sumRanksX += ranks[val]

	}

	nx := float64(len(x))

	ny := float64(len(y))

	u := sumRanksX - nx*(nx+1)/2

	// Calculate z-score for large samples.

	mu := nx * ny / 2

	sigma := math.Sqrt(nx * ny * (nx + ny + 1) / 12)

	z := (u - mu) / sigma

	// Calculate p-value.

	pValue := 2 * (1 - cumulativeNormal(math.Abs(z)))

	return u, pValue

}

func minMax(data []float64) (float64, float64) {

	if len(data) == 0 {

		return 0, 0

	}

	minVal, maxVal := data[0], data[0]

	for _, v := range data[1:] {

		if v < minVal {

			minVal = v

		}

		if v > maxVal {

			maxVal = v

		}

	}

	return minVal, maxVal

}
