package analysis

import (
	"math"
	"sort"

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
	
	sortedValues := make([]float64, len(values))
	copy(sortedValues, values)
	sort.Float64s(sortedValues)
	
	return DescriptiveStats{
		Mean:         stat.Mean(values, nil),
		Median:       median(sortedValues),
		StdDev:       stat.StdDev(values, nil),
		Min:          min(values),
		Max:          max(values),
		Percentile95: percentile(sortedValues, 0.95),
	}
}

// OutlierDetection uses multiple methods for robust outlier identification
func (sa *StatisticalAnalyzer) OutlierDetection() OutlierResult {
	values := sa.ExtractValues()
	if len(values) == 0 {
		return OutlierResult{}
	}

	sortedValues := make([]float64, len(values))
	copy(sortedValues, values)
	sort.Float64s(sortedValues)

	// IQR Method
	q1 := percentile(sortedValues, 0.25)
	q3 := percentile(sortedValues, 0.75)
	iqr := q3 - q1
	lowerBound := q1 - 1.5*iqr
	upperBound := q3 + 1.5*iqr

	// Z-Score Method
	mean := stat.Mean(values, nil)
	stdDev := stat.StdDev(values, nil)
	zScoreOutliers := make([]float64, 0)
	iqrOutliers := make([]float64, 0)

	for _, val := range values {
		if stdDev > 0 {
			zScore := (val - mean) / stdDev
			if math.Abs(zScore) > 3 {
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

// Helper functions for statistical calculations

// median calculates the median of a sorted slice
func median(sortedValues []float64) float64 {
	n := len(sortedValues)
	if n == 0 {
		return 0
	}
	if n%2 == 0 {
		return (sortedValues[n/2-1] + sortedValues[n/2]) / 2
	}
	return sortedValues[n/2]
}

// percentile calculates the percentile of a sorted slice
func percentile(sortedValues []float64, p float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}
	if p <= 0 {
		return sortedValues[0]
	}
	if p >= 1 {
		return sortedValues[len(sortedValues)-1]
	}
	
	index := p * float64(len(sortedValues)-1)
	lower := int(index)
	upper := lower + 1
	
	if upper >= len(sortedValues) {
		return sortedValues[lower]
	}
	
	weight := index - float64(lower)
	return sortedValues[lower]*(1-weight) + sortedValues[upper]*weight
}

// min finds the minimum value in a slice
func min(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	minVal := values[0]
	for _, v := range values[1:] {
		if v < minVal {
			minVal = v
		}
	}
	return minVal
}

// max finds the maximum value in a slice
func max(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	maxVal := values[0]
	for _, v := range values[1:] {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
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
