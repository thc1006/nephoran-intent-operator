package analysis

import (
	"math"
	"sort"

	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/stat"
	"gonum.org/v1/gonum/stat/distuv"
)

// PerformanceMetric represents a single performance measurement
type PerformanceMetric struct {
	Timestamp int64   // Unix timestamp
	Value     float64 // Metric value
}

// StatisticalAnalyzer provides advanced statistical analysis capabilities
type StatisticalAnalyzer struct {
	metrics []PerformanceMetric
}

// NewStatisticalAnalyzer creates a new analyzer instance
func NewStatisticalAnalyzer(metrics []PerformanceMetric) *StatisticalAnalyzer {
	return &StatisticalAnalyzer{metrics: metrics}
}

// ExtractValues converts metrics to a slice of float64 values
func (sa *StatisticalAnalyzer) ExtractValues() []float64 {
	values := make([]float64, len(sa.metrics))
	for i, m := range sa.metrics {
		values[i] = m.Value
	}
	return values
}

// DescriptiveStatistics computes comprehensive statistical descriptions
func (sa *StatisticalAnalyzer) DescriptiveStatistics() DescriptiveStats {
	values := sa.ExtractValues()
	if len(values) == 0 {
		return DescriptiveStats{}
	}

	// Handle single value case
	if len(values) == 1 {
		val := values[0]
		return DescriptiveStats{
			Mean:         val,
			Median:       val,
			StdDev:       0.0,
			Min:          val,
			Max:          val,
			Percentile95: val,
		}
	}

	// Create a copy and sort for percentile calculations
	sortedValues := make([]float64, len(values))
	copy(sortedValues, values)
	sort.Float64s(sortedValues)

	return DescriptiveStats{
		Mean:         stat.Mean(values, nil),
		Median:       stat.Quantile(0.5, stat.Empirical, sortedValues, nil),
		StdDev:       stat.StdDev(values, nil),
		Min:          floats.Min(values),
		Max:          floats.Max(values),
		Percentile95: stat.Quantile(0.95, stat.Empirical, sortedValues, nil),
	}
}

// OutlierDetection uses multiple methods for robust outlier identification
func (sa *StatisticalAnalyzer) OutlierDetection() OutlierResult {
	values := sa.ExtractValues()
	if len(values) <= 1 {
		return OutlierResult{
			ZScoreOutliers: []float64{},
			IQROutliers:    []float64{},
		}
	}

	// Create sorted copy for quantile calculations
	sortedValues := make([]float64, len(values))
	copy(sortedValues, values)
	sort.Float64s(sortedValues)

	// IQR Method
	q1 := stat.Quantile(0.25, stat.Empirical, sortedValues, nil)
	q3 := stat.Quantile(0.75, stat.Empirical, sortedValues, nil)
	iqr := q3 - q1
	lowerBound := q1 - 1.5*iqr
	upperBound := q3 + 1.5*iqr

	// Z-Score Method
	mean := stat.Mean(values, nil)
	stdDev := stat.StdDev(values, nil)
	zScoreOutliers := make([]float64, 0)
	iqrOutliers := make([]float64, 0)

	// Avoid division by zero for standard deviation
	for _, val := range values {
		if stdDev > 0 {
			zScore := (val - mean) / stdDev
			if math.Abs(zScore) > 2.5 {
				zScoreOutliers = append(zScoreOutliers, val)
			}
		}
		if val < lowerBound || val > upperBound {
			iqrOutliers = append(iqrOutliers, val)
		}
	}

	return OutlierResult{
		ZScoreOutliers: zScoreOutliers,
		IQROutliers:    iqrOutliers,
	}
}

// HypothesisTesting conducts statistical hypothesis tests
func (sa *StatisticalAnalyzer) HypothesisTesting(expectedMean float64) HypothesisResult {
	values := sa.ExtractValues()
	mean := stat.Mean(values, nil)
	stdDev := stat.StdDev(values, nil)

	// One-sample t-test
	t := (mean - expectedMean) / (stdDev / math.Sqrt(float64(len(values))))
	df := float64(len(values) - 1)

	// Two-tailed test at 0.05 significance level
	tDist := distuv.StudentsT{Nu: df}
	pValue := 2 * (1 - tDist.CDF(math.Abs(t)))

	return HypothesisResult{
		TestStatistic:    t,
		DegreesOfFreedom: df,
		PValue:           pValue,
		Significant:      pValue < 0.05,
	}
}

// Structs for returning complex results
type DescriptiveStats struct {
	Mean         float64
	Median       float64
	StdDev       float64
	Min          float64
	Max          float64
	Percentile95 float64
}

type OutlierResult struct {
	ZScoreOutliers []float64
	IQROutliers    []float64
}

type HypothesisResult struct {
	TestStatistic    float64
	DegreesOfFreedom float64
	PValue           float64
	Significant      bool
}
