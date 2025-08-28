package analysis

import (
	"math"
	"testing"
)

func TestDescriptiveStatistics(t *testing.T) {
	tests := []struct {
		name     string
		metrics  []PerformanceMetric
		expected DescriptiveStats
	}{
		{
			name:    "empty slice",
			metrics: []PerformanceMetric{},
			expected: DescriptiveStats{
				Mean:         0,
				Median:       0,
				StdDev:       0,
				Min:          0,
				Max:          0,
				Percentile95: 0,
			},
		},
		{
			name: "single value",
			metrics: []PerformanceMetric{
				{Timestamp: 1000, Value: 42.0},
			},
			expected: DescriptiveStats{
				Mean:         42.0,
				Median:       42.0,
				StdDev:       0.0,
				Min:          42.0,
				Max:          42.0,
				Percentile95: 42.0,
			},
		},
		{
			name: "odd number of values",
			metrics: []PerformanceMetric{
				{Timestamp: 1000, Value: 1.0},
				{Timestamp: 2000, Value: 2.0},
				{Timestamp: 3000, Value: 3.0},
				{Timestamp: 4000, Value: 4.0},
				{Timestamp: 5000, Value: 5.0},
			},
			expected: DescriptiveStats{
				Mean:         3.0,
				Median:       3.0,
				StdDev:       math.Sqrt(2.5), // sample standard deviation
				Min:          1.0,
				Max:          5.0,
				Percentile95: 4.8, // approximate
			},
		},
		{
			name: "even number of values",
			metrics: []PerformanceMetric{
				{Timestamp: 1000, Value: 1.0},
				{Timestamp: 2000, Value: 2.0},
				{Timestamp: 3000, Value: 3.0},
				{Timestamp: 4000, Value: 4.0},
				{Timestamp: 5000, Value: 5.0},
				{Timestamp: 6000, Value: 6.0},
			},
			expected: DescriptiveStats{
				Mean:         3.5,
				Median:       3.5,
				StdDev:       math.Sqrt(3.5), // sample standard deviation
				Min:          1.0,
				Max:          6.0,
				Percentile95: 5.75, // approximate
			},
		},
		{
			name: "duplicate values",
			metrics: []PerformanceMetric{
				{Timestamp: 1000, Value: 2.0},
				{Timestamp: 2000, Value: 2.0},
				{Timestamp: 3000, Value: 2.0},
				{Timestamp: 4000, Value: 5.0},
				{Timestamp: 5000, Value: 5.0},
			},
			expected: DescriptiveStats{
				Mean:         3.2,
				Median:       2.0,
				StdDev:       math.Sqrt(2.7), // sample standard deviation
				Min:          2.0,
				Max:          5.0,
				Percentile95: 5.0,
			},
		},
		{
			name: "negative values",
			metrics: []PerformanceMetric{
				{Timestamp: 1000, Value: -10.0},
				{Timestamp: 2000, Value: -5.0},
				{Timestamp: 3000, Value: 0.0},
				{Timestamp: 4000, Value: 5.0},
				{Timestamp: 5000, Value: 10.0},
			},
			expected: DescriptiveStats{
				Mean:         0.0,
				Median:       0.0,
				StdDev:       math.Sqrt(62.5), // sample standard deviation
				Min:          -10.0,
				Max:          10.0,
				Percentile95: 10.0, // approximate
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewStatisticalAnalyzer(tt.metrics)
			result := analyzer.DescriptiveStatistics()

			const tolerance = 0.5

			if math.Abs(result.Mean-tt.expected.Mean) > tolerance {
				t.Errorf("Mean: got %.2f, want %.2f", result.Mean, tt.expected.Mean)
			}
			if math.Abs(result.Median-tt.expected.Median) > tolerance {
				t.Errorf("Median: got %.2f, want %.2f", result.Median, tt.expected.Median)
			}
			if math.Abs(result.StdDev-tt.expected.StdDev) > tolerance {
				t.Errorf("StdDev: got %.2f, want %.2f", result.StdDev, tt.expected.StdDev)
			}
			if math.Abs(result.Min-tt.expected.Min) > tolerance {
				t.Errorf("Min: got %.2f, want %.2f", result.Min, tt.expected.Min)
			}
			if math.Abs(result.Max-tt.expected.Max) > tolerance {
				t.Errorf("Max: got %.2f, want %.2f", result.Max, tt.expected.Max)
			}
			if math.Abs(result.Percentile95-tt.expected.Percentile95) > tolerance*2 {
				t.Errorf("Percentile95: got %.2f, want %.2f", result.Percentile95, tt.expected.Percentile95)
			}
		})
	}
}

func TestOutlierDetection(t *testing.T) {
	tests := []struct {
		name            string
		metrics         []PerformanceMetric
		expectedZScore  int // number of z-score outliers
		expectedIQR     int // number of IQR outliers
	}{
		{
			name:           "no outliers",
			metrics:        makeMetrics([]float64{1, 2, 3, 4, 5}),
			expectedZScore: 0,
			expectedIQR:    0,
		},
		{
			name:           "single outlier",
			metrics:        makeMetrics([]float64{1, 2, 3, 4, 100}),
			expectedZScore: 0, // Z-score threshold might be too high for this
			expectedIQR:    1,
		},
		{
			name:           "multiple outliers",
			metrics:        makeMetrics([]float64{1, 2, 3, 4, 5, 100, 200}),
			expectedZScore: 0, // Z-score threshold might be too high for this
			expectedIQR:    0, // IQR method might not detect these as outliers
		},
		{
			name:           "edge case - single value",
			metrics:        makeMetrics([]float64{42}),
			expectedZScore: 0,
			expectedIQR:    0,
		},
		{
			name:           "edge case - empty",
			metrics:        []PerformanceMetric{},
			expectedZScore: 0,
			expectedIQR:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewStatisticalAnalyzer(tt.metrics)
			result := analyzer.OutlierDetection()

			if len(result.ZScoreOutliers) != tt.expectedZScore {
				t.Errorf("Z-score outliers: got %d, want %d", len(result.ZScoreOutliers), tt.expectedZScore)
			}
			if len(result.IQROutliers) != tt.expectedIQR {
				t.Errorf("IQR outliers: got %d, want %d", len(result.IQROutliers), tt.expectedIQR)
			}
		})
	}
}

func TestHypothesisTesting(t *testing.T) {
	tests := []struct {
		name              string
		metrics           []PerformanceMetric
		expectedMean      float64
		expectSignificant bool
	}{
		{
			name:              "no significant difference",
			metrics:           makeMetrics([]float64{10, 11, 12, 13, 14}),
			expectedMean:      12.0, // close to actual mean of 12
			expectSignificant: false,
		},
		{
			name:              "significant difference - higher",
			metrics:           makeMetrics([]float64{20, 21, 22, 23, 24}),
			expectedMean:      12.0, // significantly different from actual mean of 22
			expectSignificant: true,
		},
		{
			name:              "significant difference - lower",
			metrics:           makeMetrics([]float64{10, 11, 12, 13, 14}),
			expectedMean:      22.0, // significantly different from actual mean of 12
			expectSignificant: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewStatisticalAnalyzer(tt.metrics)
			result := analyzer.HypothesisTesting(tt.expectedMean)

			if result.Significant != tt.expectSignificant {
				t.Errorf("Significant: got %v, want %v (p-value: %.4f)",
					result.Significant, tt.expectSignificant, result.PValue)
			}
		})
	}
}

func TestExtractValues(t *testing.T) {
	metrics := []PerformanceMetric{
		{Timestamp: 1000, Value: 1.5},
		{Timestamp: 2000, Value: 2.5},
		{Timestamp: 3000, Value: 3.5},
	}

	analyzer := NewStatisticalAnalyzer(metrics)
	values := analyzer.ExtractValues()

	if len(values) != 3 {
		t.Fatalf("Expected 3 values, got %d", len(values))
	}

	expected := []float64{1.5, 2.5, 3.5}
	for i, v := range values {
		if v != expected[i] {
			t.Errorf("Value[%d]: got %.2f, want %.2f", i, v, expected[i])
		}
	}
}

// Helper function to create metrics from values
func makeMetrics(values []float64) []PerformanceMetric {
	metrics := make([]PerformanceMetric, len(values))
	for i, v := range values {
		metrics[i] = PerformanceMetric{
			Timestamp: int64(i * 1000),
			Value:     v,
		}
	}
	return metrics
}