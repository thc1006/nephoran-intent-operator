/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package monitoring

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// AlertSeverity represents alert severity levels
// Consolidated from alerting.go, sla_components.go, and distributed_tracing.go
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityError    AlertSeverity = "error"
	AlertSeverityCritical AlertSeverity = "critical"
	// Additional severity levels from distributed_tracing.go
	SeverityLow    AlertSeverity = "low"
	SeverityMedium AlertSeverity = "medium"
	SeverityHigh   AlertSeverity = "high"
)

// TrendAnalyzer analyzes error trends and patterns
// Consolidated from error_tracking.go and predictive_sla_analyzer.go
type TrendAnalyzer struct {
	// From error_tracking.go
	timeSeriesData map[string]*TimeSeries
	trendModels    map[string]*TrendModel
	logger         *zap.Logger
	mutex          sync.RWMutex

	// From predictive_sla_analyzer.go
	windowSizes []time.Duration
	confidence  float64
}

// AnomalyDetector detects anomalous error patterns
// Consolidated from error_tracking.go, predictive_sla_analyzer.go, and sla_components.go
type AnomalyDetector struct {
	// From error_tracking.go
	detectors  map[string]*AnomalyDetectorModel
	baselines  map[string]*Baseline
	alertRules []AnomalyAlertRule
	logger     *zap.Logger
	mutex      sync.RWMutex

	// From predictive_sla_analyzer.go
	algorithm        string // "isolation_forest", "one_class_svm", "local_outlier_factor"
	sensitivityLevel float64
	windowSize       time.Duration
	trainingSize     int
}

// TrendModel represents a trend model
// Consolidated from error_tracking.go, predictive_sla_analyzer.go, and sla_components.go
type TrendModel struct {
	// From error_tracking.go
	Name        string             `json:"name"`
	ModelType   TrendModelType     `json:"modelType"`
	Parameters  map[string]float64 `json:"parameters"`
	Accuracy    float64            `json:"accuracy"`
	LastTrained time.Time          `json:"lastTrained"`
	Predictions []TrendPrediction  `json:"predictions"`

	// From predictive_sla_analyzer.go
	slope      float64
	intercept  float64
	r2         float64
	direction  string // "increasing", "decreasing", "stable"
	confidence float64

	// From sla_components.go
	Type            string // "linear", "polynomial", "seasonal"
	PredictionRange time.Duration
	LastUpdated     time.Time
}

// SeasonalityDetector detects seasonal patterns in data
// Consolidated from predictive_sla_analyzer.go and sla_components.go
type SeasonalityDetector struct {
	// From predictive_sla_analyzer.go
	patterns      []SeasonalPattern
	detectionAlgo string // "fft", "autocorr", "stl"
	minPeriod     time.Duration
	maxPeriod     time.Duration

	// From sla_components.go
	detectionWindow time.Duration
	confidence      float64
}

// SeasonalPattern represents a seasonal pattern
// Consolidated from predictive_sla_analyzer.go and sla_components.go
type SeasonalPattern struct {
	// From predictive_sla_analyzer.go
	period    time.Duration
	amplitude float64
	phase     float64
	strength  float64
	detected  bool

	// From sla_components.go
	Name   string
	Period time.Duration
	Phase  float64
}

// TimeSeries represents time series data
// Consolidated from error_tracking.go and sla_components.go
type TimeSeries struct {
	// From error_tracking.go
	Name        string          `json:"name"`
	DataPoints  []DataPoint     `json:"dataPoints"`
	Aggregation AggregationType `json:"aggregation"`
	Resolution  time.Duration   `json:"resolution"`

	// From sla_components.go
	buffer     *CircularBuffer
	aggregator func([]float64) float64
	mutex      sync.RWMutex
}

// Supporting types that were referenced in the consolidated types above

// TrendModelType defines trend model types
type TrendModelType string

const (
	ModelLinear      TrendModelType = "linear"
	ModelExponential TrendModelType = "exponential"
	ModelSeasonal    TrendModelType = "seasonal"
	ModelARIMA       TrendModelType = "arima"
)

// TrendPrediction represents a trend prediction
type TrendPrediction struct {
	Timestamp  time.Time `json:"timestamp"`
	Value      float64   `json:"value"`
	Confidence float64   `json:"confidence"`
	LowerBound float64   `json:"lowerBound"`
	UpperBound float64   `json:"upperBound"`
}

// AnomalyDetectorModel represents an anomaly detection model
type AnomalyDetectorModel struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	Algorithm         AnomalyAlgorithm       `json:"algorithm"`
	Sensitivity       float64                `json:"sensitivity"`
	ThresholdStdDev   float64                `json:"thresholdStdDev"`
	Parameters        map[string]interface{} `json:"parameters"`
	LastTrained       time.Time              `json:"lastTrained"`
	DetectionAccuracy float64                `json:"detectionAccuracy"`
}

// AnomalyAlgorithm defines anomaly detection algorithms
type AnomalyAlgorithm string

const (
	AlgorithmIsolationForest AnomalyAlgorithm = "isolation_forest"
	AlgorithmOneClassSVM     AnomalyAlgorithm = "one_class_svm"
	AlgorithmStatistical     AnomalyAlgorithm = "statistical"
	AlgorithmLocalOutlier    AnomalyAlgorithm = "local_outlier_factor"
	AlgorithmAutoEncoder     AnomalyAlgorithm = "autoencoder"
)

// Baseline represents normal behavior baseline
type Baseline struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	MetricName  string             `json:"metricName"`
	Mean        float64            `json:"mean"`
	StandardDev float64            `json:"standardDev"`
	Min         float64            `json:"min"`
	Max         float64            `json:"max"`
	Percentiles map[string]float64 `json:"percentiles"`
	SampleSize  int64              `json:"sampleSize"`
	LastUpdated time.Time          `json:"lastUpdated"`
	TimeWindow  time.Duration      `json:"timeWindow"`
}

// AnomalyAlertRule defines rules for anomaly alerts
type AnomalyAlertRule struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	MetricName string        `json:"metricName"`
	Threshold  float64       `json:"threshold"`
	Operator   Operator      `json:"operator"`
	Severity   AlertSeverity `json:"severity"`
	Enabled    bool          `json:"enabled"`
}

// Operator defines comparison operators
type Operator string

const (
	OperatorGreaterThan    Operator = "gt"
	OperatorLessThan       Operator = "lt"
	OperatorEqual          Operator = "eq"
	OperatorNotEqual       Operator = "ne"
	OperatorGreaterOrEqual Operator = "gte"
	OperatorLessOrEqual    Operator = "lte"
)

// DataPoint represents a single data point
type DataPoint struct {
	Timestamp time.Time         `json:"timestamp"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels"`
}

// AggregationType defines aggregation types
type AggregationType string

const (
	AggregationSum     AggregationType = "sum"
	AggregationAverage AggregationType = "average"
	AggregationMax     AggregationType = "max"
	AggregationMin     AggregationType = "min"
	AggregationCount   AggregationType = "count"
)

// CircularBuffer provides efficient ring buffer for time series data
// This type was referenced by TimeSeries but defined in sla_components.go
type CircularBuffer struct {
	data     []float64
	times    []time.Time
	capacity int
	size     int
	head     int
	tail     int
	mu       sync.RWMutex
}

// NewCircularBuffer creates a new circular buffer
func NewCircularBuffer(capacity int) *CircularBuffer {
	return &CircularBuffer{
		data:     make([]float64, capacity),
		times:    make([]time.Time, capacity),
		capacity: capacity,
	}
}

// Add adds a new value to the circular buffer
func (cb *CircularBuffer) Add(timestamp time.Time, value float64) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.data[cb.head] = value
	cb.times[cb.head] = timestamp

	cb.head = (cb.head + 1) % cb.capacity

	if cb.size < cb.capacity {
		cb.size++
	} else {
		cb.tail = (cb.tail + 1) % cb.capacity
	}
}

// GetRecent returns the most recent n values
func (cb *CircularBuffer) GetRecent(n int) ([]time.Time, []float64) {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if n > cb.size {
		n = cb.size
	}

	times := make([]time.Time, n)
	values := make([]float64, n)

	for i := 0; i < n; i++ {
		idx := (cb.head - 1 - i + cb.capacity) % cb.capacity
		times[n-1-i] = cb.times[idx]
		values[n-1-i] = cb.data[idx]
	}

	return times, values
}

// NewTimeSeries creates a new generic TimeSeries
func NewTimeSeries(capacity int) *TimeSeries {
	return &TimeSeries{
		buffer: NewCircularBuffer(capacity),
		aggregator: func(values []float64) float64 {
			if len(values) == 0 {
				return 0.0
			}
			sum := 0.0
			for _, v := range values {
				sum += v
			}
			return sum / float64(len(values))
		},
	}
}

// Add adds a value to the time series
func (ts *TimeSeries) Add(timestamp time.Time, value float64) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()
	ts.buffer.Add(timestamp, value)
}

// GetRecent returns the most recent values
func (ts *TimeSeries) GetRecent(n int) ([]time.Time, []float64) {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()
	return ts.buffer.GetRecent(n)
}

// Aggregate returns aggregated value over recent points
func (ts *TimeSeries) Aggregate(n int) float64 {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()
	_, values := ts.buffer.GetRecent(n)
	return ts.aggregator(values)
}

// NewTrendAnalyzer creates a new TrendAnalyzer
func NewTrendAnalyzer(logger *zap.Logger) *TrendAnalyzer {
	return &TrendAnalyzer{
		timeSeriesData: make(map[string]*TimeSeries),
		trendModels:    make(map[string]*TrendModel),
		logger:         logger,
		windowSizes:    []time.Duration{15 * time.Minute, 1 * time.Hour, 4 * time.Hour, 24 * time.Hour},
		confidence:     0.95,
	}
}

// NewAnomalyDetector creates a new AnomalyDetector
func NewAnomalyDetector(logger *zap.Logger) *AnomalyDetector {
	return &AnomalyDetector{
		detectors:        make(map[string]*AnomalyDetectorModel),
		baselines:        make(map[string]*Baseline),
		alertRules:       make([]AnomalyAlertRule, 0),
		logger:           logger,
		algorithm:        "isolation_forest",
		sensitivityLevel: 0.1,
		windowSize:       1 * time.Hour,
		trainingSize:     1000,
	}
}

// NewSeasonalityDetector creates a new SeasonalityDetector
func NewSeasonalityDetector() *SeasonalityDetector {
	return &SeasonalityDetector{
		patterns:        make([]SeasonalPattern, 0),
		detectionAlgo:   "autocorr",
		minPeriod:       15 * time.Minute,
		maxPeriod:       7 * 24 * time.Hour, // Weekly patterns
		detectionWindow: 1 * time.Hour,
		confidence:      0.95,
	}
}

// GetSeasonalAdjustment returns seasonal adjustment for the given timestamp and horizon
func (sd *SeasonalityDetector) GetSeasonalAdjustment(timestamp time.Time, horizon time.Duration) float64 {
	// Mock seasonal adjustment
	hour := timestamp.Hour()
	// Simple daily pattern: higher latency during business hours
	if hour >= 9 && hour <= 17 {
		return 100.0 // Add 100ms during business hours
	}
	return -50.0 // Reduce 50ms during off-hours
}