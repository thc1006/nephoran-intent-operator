package validation

import (
	"fmt"
	"math"
	"sort"
	"time"

	"gonum.org/v1/gonum/stat"
	"gonum.org/v1/gonum/stat/distuv"
)

// StatisticalValidator provides comprehensive statistical analysis and validation
// for performance benchmarks with proper confidence intervals and regression testing
type StatisticalValidator struct {
	confidenceLevel  float64
	significanceLevel float64
	minSampleSize    int
	powerThreshold   float64
}

// ValidationConfig defines parameters for statistical validation
type ValidationConfig struct {
	ConfidenceLevel   float64 // e.g., 0.95 for 95% confidence
	SignificanceLevel float64 // e.g., 0.05 for 5% significance
	MinSampleSize     int     // Minimum samples for valid statistics
	PowerThreshold    float64 // Minimum statistical power required
	EffectSizeThreshold float64 // Minimum effect size to detect
}

// PerformanceMetrics represents a collection of performance measurements
type PerformanceMetrics struct {
	Name        string
	Values      []float64
	Unit        string
	Timestamp   time.Time
	SampleSize  int
	TestConfig  *ValidationConfig
}

// StatisticalSummary contains comprehensive statistical analysis
type StatisticalSummary struct {
	Metric              string
	SampleSize          int
	Mean                float64
	Median              float64
	StandardDeviation   float64
	StandardError       float64
	Variance            float64
	Skewness            float64
	Kurtosis            float64
	Min                 float64
	Max                 float64
	Range               float64
	InterquartileRange  float64
	Percentiles         map[float64]float64
	ConfidenceInterval  *ConfidenceInterval
	OutlierAnalysis     *OutlierAnalysis
	NormalityTest       *NormalityTestResult
	PowerAnalysis       *PowerAnalysis
}

// ConfidenceInterval represents statistical confidence bounds
type ConfidenceInterval struct {
	Level      float64
	LowerBound float64
	UpperBound float64
	MarginOfError float64
}

// OutlierAnalysis identifies and analyzes statistical outliers
type OutlierAnalysis struct {
	Method            string // "IQR", "Z-Score", "Modified Z-Score"
	OutlierCount      int
	OutlierPercentage float64
	OutlierIndices    []int
	OutlierValues     []float64
	LowerBound        float64
	UpperBound        float64
}

// NormalityTestResult contains results from normality testing
type NormalityTestResult struct {
	TestName    string
	Statistic   float64
	PValue      float64
	IsNormal    bool
	Conclusion  string
}

// PowerAnalysis evaluates the statistical power of the test
type PowerAnalysis struct {
	EffectSize       float64
	Power            float64
	RequiredSamples  int
	DetectableEffect float64
	Recommendation   string
}

// RegressionAnalysis compares current metrics against baseline/historical data
type RegressionAnalysis struct {
	Metric              string
	CurrentValue        float64
	BaselineValue       float64
	RelativeChange      float64
	AbsoluteChange      float64
	IsRegression        bool
	Significance        float64
	ConfidenceInterval  *ConfidenceInterval
	RegressionSeverity  string
	TTestResult         *TTestResult
}

// TTestResult contains results from t-test analysis
type TTestResult struct {
	TestType    string // "one-sample", "two-sample", "paired"
	Statistic   float64
	DegreesOfFreedom int
	PValue      float64
	IsSignificant bool
	EffectSize  float64
}

// ValidationResult represents the overall validation outcome
type ValidationResult struct {
	Metric              string
	TargetValue         float64
	ActualValue         float64
	Passed              bool
	ConfidenceLevel     float64
	StatisticalPower    float64
	MarginOfError       float64
	RequiredImprovement float64
	Recommendations     []string
}

// NewStatisticalValidator creates a new statistical validator with default settings
func NewStatisticalValidator(config *ValidationConfig) *StatisticalValidator {
	if config == nil {
		config = &ValidationConfig{
			ConfidenceLevel:     0.95,
			SignificanceLevel:   0.05,
			MinSampleSize:       30,
			PowerThreshold:      0.80,
			EffectSizeThreshold: 0.2, // Small to medium effect size
		}
	}
	
	return &StatisticalValidator{
		confidenceLevel:   config.ConfidenceLevel,
		significanceLevel: config.SignificanceLevel,
		minSampleSize:     config.MinSampleSize,
		powerThreshold:    config.PowerThreshold,
	}
}

// AnalyzeMetrics performs comprehensive statistical analysis on performance metrics
func (sv *StatisticalValidator) AnalyzeMetrics(metrics *PerformanceMetrics) (*StatisticalSummary, error) {
	if len(metrics.Values) < sv.minSampleSize {
		return nil, fmt.Errorf("insufficient sample size: %d < %d", len(metrics.Values), sv.minSampleSize)
	}
	
	// Create a copy and sort for percentile calculations
	values := make([]float64, len(metrics.Values))
	copy(values, metrics.Values)
	sort.Float64s(values)
	
	summary := &StatisticalSummary{
		Metric:     metrics.Name,
		SampleSize: len(values),
	}
	
	// Basic descriptive statistics
	summary.Mean = stat.Mean(values, nil)
	summary.Median = stat.Quantile(0.5, stat.Empirical, values, nil)
	summary.StandardDeviation = stat.StdDev(values, nil)
	summary.StandardError = summary.StandardDeviation / math.Sqrt(float64(len(values)))
	summary.Variance = stat.Variance(values, nil)
	summary.Min = values[0]
	summary.Max = values[len(values)-1]
	summary.Range = summary.Max - summary.Min
	
	// Advanced statistical measures
	summary.Skewness = sv.calculateSkewness(values, summary.Mean, summary.StandardDeviation)
	summary.Kurtosis = sv.calculateKurtosis(values, summary.Mean, summary.StandardDeviation)
	
	// Percentiles
	summary.Percentiles = map[float64]float64{
		5:    stat.Quantile(0.05, stat.Empirical, values, nil),
		10:   stat.Quantile(0.10, stat.Empirical, values, nil),
		25:   stat.Quantile(0.25, stat.Empirical, values, nil),
		50:   summary.Median,
		75:   stat.Quantile(0.75, stat.Empirical, values, nil),
		90:   stat.Quantile(0.90, stat.Empirical, values, nil),
		95:   stat.Quantile(0.95, stat.Empirical, values, nil),
		99:   stat.Quantile(0.99, stat.Empirical, values, nil),
		99.9: stat.Quantile(0.999, stat.Empirical, values, nil),
	}
	
	q1 := summary.Percentiles[25]
	q3 := summary.Percentiles[75]
	summary.InterquartileRange = q3 - q1
	
	// Confidence interval
	summary.ConfidenceInterval = sv.calculateConfidenceInterval(values, summary.Mean, summary.StandardError)
	
	// Outlier analysis
	summary.OutlierAnalysis = sv.detectOutliers(values, q1, q3, summary.InterquartileRange)
	
	// Normality testing
	summary.NormalityTest = sv.testNormality(values)
	
	// Power analysis
	summary.PowerAnalysis = sv.performPowerAnalysis(len(values), summary.StandardDeviation)
	
	return summary, nil
}

// CompareToBaseline performs regression analysis against historical baseline
func (sv *StatisticalValidator) CompareToBaseline(current *PerformanceMetrics, baseline *PerformanceMetrics) (*RegressionAnalysis, error) {
	if len(current.Values) < sv.minSampleSize || len(baseline.Values) < sv.minSampleSize {
		return nil, fmt.Errorf("insufficient sample size for comparison")
	}
	
	currentMean := stat.Mean(current.Values, nil)
	baselineMean := stat.Mean(baseline.Values, nil)
	
	analysis := &RegressionAnalysis{
		Metric:         current.Name,
		CurrentValue:   currentMean,
		BaselineValue:  baselineMean,
		AbsoluteChange: currentMean - baselineMean,
	}
	
	if baselineMean != 0 {
		analysis.RelativeChange = (currentMean - baselineMean) / baselineMean * 100
	}
	
	// Perform two-sample t-test
	analysis.TTestResult = sv.performTTest(current.Values, baseline.Values)
	
	// Determine if this represents a significant regression
	// For performance metrics, we typically care about increases (regressions)
	threshold := 0.05 // 5% threshold for considering regression
	analysis.IsRegression = analysis.RelativeChange > threshold && analysis.TTestResult.IsSignificant
	
	// Calculate significance level
	analysis.Significance = analysis.TTestResult.PValue
	
	// Determine regression severity
	if analysis.IsRegression {
		if analysis.RelativeChange > 20 {
			analysis.RegressionSeverity = "Critical"
		} else if analysis.RelativeChange > 10 {
			analysis.RegressionSeverity = "High"
		} else if analysis.RelativeChange > 5 {
			analysis.RegressionSeverity = "Medium"
		} else {
			analysis.RegressionSeverity = "Low"
		}
	}
	
	return analysis, nil
}

// ValidateTarget validates if current performance meets specified target
func (sv *StatisticalValidator) ValidateTarget(metrics *PerformanceMetrics, target float64, comparison string) (*ValidationResult, error) {
	summary, err := sv.AnalyzeMetrics(metrics)
	if err != nil {
		return nil, err
	}
	
	result := &ValidationResult{
		Metric:          metrics.Name,
		TargetValue:     target,
		ActualValue:     summary.Mean,
		ConfidenceLevel: sv.confidenceLevel,
		StatisticalPower: summary.PowerAnalysis.Power,
		MarginOfError:   summary.ConfidenceInterval.MarginOfError,
		Recommendations: make([]string, 0),
	}
	
	// Determine if target is met based on comparison type and confidence interval
	switch comparison {
	case "less_than", "<":
		// For latency metrics - want actual < target
		result.Passed = summary.ConfidenceInterval.UpperBound < target
		if !result.Passed {
			result.RequiredImprovement = summary.ConfidenceInterval.UpperBound - target
		}
	case "greater_than", ">":
		// For throughput metrics - want actual > target
		result.Passed = summary.ConfidenceInterval.LowerBound > target
		if !result.Passed {
			result.RequiredImprovement = target - summary.ConfidenceInterval.LowerBound
		}
	case "approximately", "≈":
		// For hit rates, availability - want actual ≈ target (within margin)
		margin := target * 0.02 // 2% margin
		result.Passed = math.Abs(summary.Mean-target) <= margin
		if !result.Passed {
			result.RequiredImprovement = math.Abs(summary.Mean - target)
		}
	default:
		return nil, fmt.Errorf("invalid comparison type: %s", comparison)
	}
	
	// Generate recommendations
	if !result.Passed {
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("Performance target not met with %.1f%% confidence", sv.confidenceLevel*100))
		
		if summary.PowerAnalysis.Power < sv.powerThreshold {
			result.Recommendations = append(result.Recommendations,
				fmt.Sprintf("Consider increasing sample size to %d for better statistical power", 
					summary.PowerAnalysis.RequiredSamples))
		}
		
		if summary.OutlierAnalysis.OutlierPercentage > 5 {
			result.Recommendations = append(result.Recommendations,
				fmt.Sprintf("%.1f%% outliers detected - investigate potential causes", 
					summary.OutlierAnalysis.OutlierPercentage))
		}
		
		if !summary.NormalityTest.IsNormal {
			result.Recommendations = append(result.Recommendations,
				"Data is not normally distributed - consider non-parametric analysis")
		}
	}
	
	return result, nil
}

// calculateConfidenceInterval computes confidence interval using t-distribution
func (sv *StatisticalValidator) calculateConfidenceInterval(values []float64, mean, standardError float64) *ConfidenceInterval {
	n := len(values)
	df := n - 1 // degrees of freedom
	
	// t-critical value for given confidence level
	alpha := 1 - sv.confidenceLevel
	tDist := distuv.StudentsT{Mu: 0, Sigma: 1, Nu: float64(df)}
	tCritical := tDist.Quantile(1 - alpha/2)
	
	marginOfError := tCritical * standardError
	
	return &ConfidenceInterval{
		Level:         sv.confidenceLevel,
		LowerBound:    mean - marginOfError,
		UpperBound:    mean + marginOfError,
		MarginOfError: marginOfError,
	}
}

// detectOutliers identifies outliers using IQR method
func (sv *StatisticalValidator) detectOutliers(values []float64, q1, q3, iqr float64) *OutlierAnalysis {
	lowerBound := q1 - 1.5*iqr
	upperBound := q3 + 1.5*iqr
	
	outliers := make([]float64, 0)
	indices := make([]int, 0)
	
	for i, value := range values {
		if value < lowerBound || value > upperBound {
			outliers = append(outliers, value)
			indices = append(indices, i)
		}
	}
	
	return &OutlierAnalysis{
		Method:            "IQR",
		OutlierCount:      len(outliers),
		OutlierPercentage: float64(len(outliers)) / float64(len(values)) * 100,
		OutlierIndices:    indices,
		OutlierValues:     outliers,
		LowerBound:        lowerBound,
		UpperBound:        upperBound,
	}
}

// testNormality performs Shapiro-Wilk test for normality (simplified implementation)
func (sv *StatisticalValidator) testNormality(values []float64) *NormalityTestResult {
	// For sample sizes > 50, use alternative test
	// This is a simplified implementation - in production, use proper statistical libraries
	
	n := len(values)
	if n < 3 || n > 5000 {
		return &NormalityTestResult{
			TestName:   "Shapiro-Wilk",
			Conclusion: "Sample size outside valid range for normality testing",
			IsNormal:   false,
		}
	}
	
	// Simplified normality check using skewness and kurtosis
	mean := stat.Mean(values, nil)
	stdDev := stat.StdDev(values, nil)
	
	skewness := sv.calculateSkewness(values, mean, stdDev)
	kurtosis := sv.calculateKurtosis(values, mean, stdDev)
	
	// Rule of thumb: data is approximately normal if:
	// |skewness| < 2 and |kurtosis - 3| < 2
	isNormal := math.Abs(skewness) < 2 && math.Abs(kurtosis-3) < 2
	
	conclusion := "Data appears normally distributed"
	if !isNormal {
		conclusion = fmt.Sprintf("Data shows non-normal distribution (skewness=%.3f, excess kurtosis=%.3f)", 
			skewness, kurtosis-3)
	}
	
	return &NormalityTestResult{
		TestName:   "Simplified Normality Check",
		Statistic:  skewness, // Using skewness as primary statistic
		IsNormal:   isNormal,
		Conclusion: conclusion,
	}
}

// performPowerAnalysis calculates statistical power and required sample size
func (sv *StatisticalValidator) performPowerAnalysis(currentSampleSize int, standardDeviation float64) *PowerAnalysis {
	// Effect size calculation (Cohen's d)
	// Small effect: 0.2, Medium: 0.5, Large: 0.8
	effectSizes := []float64{0.2, 0.5, 0.8}
	
	analysis := &PowerAnalysis{}
	
	// Calculate power for medium effect size (0.5)
	effectSize := 0.5
	analysis.EffectSize = effectSize
	
	// Simplified power calculation using normal approximation
	// In practice, use proper power analysis libraries
	alpha := sv.significanceLevel
	zAlpha := sv.normalQuantile(1 - alpha/2)
	zBeta := sv.normalQuantile(sv.powerThreshold)
	
	// Required sample size for desired power
	requiredN := int(math.Pow((zAlpha+zBeta)/effectSize, 2) * 2)
	analysis.RequiredSamples = requiredN
	
	// Calculate actual power with current sample size
	if currentSampleSize > 0 {
		actualZBeta := effectSize*math.Sqrt(float64(currentSampleSize)/2) - zAlpha
		analysis.Power = sv.normalCDF(actualZBeta)
	}
	
	// Minimum detectable effect with current sample
	if currentSampleSize > 0 {
		analysis.DetectableEffect = (zAlpha + zBeta) / math.Sqrt(float64(currentSampleSize)/2)
	}
	
	// Generate recommendation
	if analysis.Power < sv.powerThreshold {
		analysis.Recommendation = fmt.Sprintf("Increase sample size to %d for %.1f%% power to detect medium effects", 
			requiredN, sv.powerThreshold*100)
	} else {
		analysis.Recommendation = fmt.Sprintf("Current sample size provides adequate power (%.1f%%)", analysis.Power*100)
	}
	
	return analysis
}

// performTTest performs two-sample t-test
func (sv *StatisticalValidator) performTTest(sample1, sample2 []float64) *TTestResult {
	n1, n2 := len(sample1), len(sample2)
	
	mean1 := stat.Mean(sample1, nil)
	mean2 := stat.Mean(sample2, nil)
	
	var1 := stat.Variance(sample1, nil)
	var2 := stat.Variance(sample2, nil)
	
	// Welch's t-test (unequal variances)
	pooledSE := math.Sqrt(var1/float64(n1) + var2/float64(n2))
	tStatistic := (mean1 - mean2) / pooledSE
	
	// Degrees of freedom for Welch's t-test
	df := math.Pow(var1/float64(n1)+var2/float64(n2), 2) / 
		(math.Pow(var1/float64(n1), 2)/float64(n1-1) + math.Pow(var2/float64(n2), 2)/float64(n2-1))
	
	// Calculate p-value (simplified)
	tDist := distuv.StudentsT{Mu: 0, Sigma: 1, Nu: df}
	pValue := 2 * (1 - tDist.CDF(math.Abs(tStatistic)))
	
	// Effect size (Cohen's d)
	pooledStd := math.Sqrt(((float64(n1-1))*var1 + (float64(n2-1))*var2) / float64(n1+n2-2))
	effectSize := (mean1 - mean2) / pooledStd
	
	return &TTestResult{
		TestType:         "two-sample",
		Statistic:        tStatistic,
		DegreesOfFreedom: int(df),
		PValue:           pValue,
		IsSignificant:    pValue < sv.significanceLevel,
		EffectSize:       effectSize,
	}
}

// Helper mathematical functions

func (sv *StatisticalValidator) calculateSkewness(values []float64, mean, stdDev float64) float64 {
	n := float64(len(values))
	var sum float64
	
	for _, value := range values {
		standardized := (value - mean) / stdDev
		sum += math.Pow(standardized, 3)
	}
	
	return sum / n
}

func (sv *StatisticalValidator) calculateKurtosis(values []float64, mean, stdDev float64) float64 {
	n := float64(len(values))
	var sum float64
	
	for _, value := range values {
		standardized := (value - mean) / stdDev
		sum += math.Pow(standardized, 4)
	}
	
	return sum / n
}

// normalQuantile returns the quantile of the standard normal distribution
func (sv *StatisticalValidator) normalQuantile(p float64) float64 {
	normal := distuv.Normal{Mu: 0, Sigma: 1}
	return normal.Quantile(p)
}

// normalCDF returns the cumulative distribution function of standard normal
func (sv *StatisticalValidator) normalCDF(x float64) float64 {
	normal := distuv.Normal{Mu: 0, Sigma: 1}
	return normal.CDF(x)
}

// GenerateReport creates a comprehensive statistical report
func (sv *StatisticalValidator) GenerateReport(summaries []*StatisticalSummary, validations []*ValidationResult, regressions []*RegressionAnalysis) string {
	report := "COMPREHENSIVE PERFORMANCE VALIDATION REPORT\n"
	report += "============================================\n\n"
	
	// Executive Summary
	report += "EXECUTIVE SUMMARY\n"
	report += "-----------------\n"
	
	totalTargets := len(validations)
	passedTargets := 0
	for _, validation := range validations {
		if validation.Passed {
			passedTargets++
		}
	}
	
	passRate := float64(passedTargets) / float64(totalTargets) * 100
	report += fmt.Sprintf("Performance Targets Met: %d/%d (%.1f%%)\n", passedTargets, totalTargets, passRate)
	
	if len(regressions) > 0 {
		regressionCount := 0
		for _, regression := range regressions {
			if regression.IsRegression {
				regressionCount++
			}
		}
		report += fmt.Sprintf("Performance Regressions Detected: %d/%d\n", regressionCount, len(regressions))
	}
	
	report += fmt.Sprintf("Statistical Confidence Level: %.1f%%\n\n", sv.confidenceLevel*100)
	
	// Detailed Results
	report += "DETAILED PERFORMANCE ANALYSIS\n"
	report += "=============================\n\n"
	
	for i, summary := range summaries {
		validation := validations[i]
		
		report += fmt.Sprintf("Metric: %s\n", summary.Metric)
		report += fmt.Sprintf("Target: %.2f | Actual: %.2f | Status: %s\n", 
			validation.TargetValue, validation.ActualValue, 
			map[bool]string{true: "✓ PASSED", false: "✗ FAILED"}[validation.Passed])
		
		report += fmt.Sprintf("Sample Size: %d | Statistical Power: %.1f%%\n", 
			summary.SampleSize, summary.PowerAnalysis.Power*100)
		
		report += fmt.Sprintf("Mean: %.2f ± %.2f (95%% CI: [%.2f, %.2f])\n", 
			summary.Mean, summary.ConfidenceInterval.MarginOfError,
			summary.ConfidenceInterval.LowerBound, summary.ConfidenceInterval.UpperBound)
		
		report += fmt.Sprintf("P95: %.2f | P99: %.2f | Outliers: %.1f%%\n", 
			summary.Percentiles[95], summary.Percentiles[99], summary.OutlierAnalysis.OutlierPercentage)
		
		if len(validation.Recommendations) > 0 {
			report += "Recommendations:\n"
			for _, rec := range validation.Recommendations {
				report += fmt.Sprintf("  • %s\n", rec)
			}
		}
		
		report += "\n"
	}
	
	// Regression Analysis
	if len(regressions) > 0 {
		report += "REGRESSION ANALYSIS\n"
		report += "==================\n\n"
		
		for _, regression := range regressions {
			status := "No Change"
			if regression.IsRegression {
				status = fmt.Sprintf("REGRESSION (%s)", regression.RegressionSeverity)
			} else if regression.RelativeChange < -5 {
				status = "IMPROVEMENT"
			}
			
			report += fmt.Sprintf("%s: %s\n", regression.Metric, status)
			report += fmt.Sprintf("  Change: %.2f%% (%.2f → %.2f)\n", 
				regression.RelativeChange, regression.BaselineValue, regression.CurrentValue)
			report += fmt.Sprintf("  Statistical Significance: p = %.4f\n", regression.Significance)
			report += fmt.Sprintf("  Effect Size: %.2f\n\n", regression.TTestResult.EffectSize)
		}
	}
	
	return report
}

// BatchValidation performs validation on multiple metrics simultaneously
type BatchValidation struct {
	validator *StatisticalValidator
	results   []*ValidationResult
	summaries []*StatisticalSummary
}

func NewBatchValidation(validator *StatisticalValidator) *BatchValidation {
	return &BatchValidation{
		validator: validator,
		results:   make([]*ValidationResult, 0),
		summaries: make([]*StatisticalSummary, 0),
	}
}

func (bv *BatchValidation) AddTarget(metrics *PerformanceMetrics, target float64, comparison string) error {
	summary, err := bv.validator.AnalyzeMetrics(metrics)
	if err != nil {
		return err
	}
	
	validation, err := bv.validator.ValidateTarget(metrics, target, comparison)
	if err != nil {
		return err
	}
	
	bv.summaries = append(bv.summaries, summary)
	bv.results = append(bv.results, validation)
	
	return nil
}

func (bv *BatchValidation) GetResults() ([]*ValidationResult, []*StatisticalSummary) {
	return bv.results, bv.summaries
}

func (bv *BatchValidation) GenerateReport() string {
	return bv.validator.GenerateReport(bv.summaries, bv.results, nil)
}