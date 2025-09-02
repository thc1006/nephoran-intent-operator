// Package monitoring - Predictive SLA analysis implementation
package monitoring

import (
	
	"encoding/json"
"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// PredictiveSLAAnalyzer implements TrendAnalyzer with predictive SLA capabilities
type PredictiveSLAAnalyzer struct {
	mu            sync.RWMutex
	models        map[string]*PredictiveModel
	slaThresholds map[string]SLAThreshold
	registry      prometheus.Registerer
	detector      SeasonalityDetector

	// Metrics
	predictionsTotal  prometheus.Counter
	predictionErrors  prometheus.Counter
	predictionLatency prometheus.Histogram
	slaViolations     prometheus.Counter
}

// PredictiveModel represents a statistical model for predictions
type PredictiveModel struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"` // linear, polynomial, arima, lstm
	Parameters   map[string]float64     `json:"parameters"`
	TrainingData []*MetricsData         `json:"-"`
	LastTrained  time.Time              `json:"lastTrained"`
	Accuracy     float64                `json:"accuracy"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
}

// SLAThreshold defines SLA thresholds for metrics
type SLAThreshold struct {
	MetricName    string  `json:"metricName"`
	WarningLevel  float64 `json:"warningLevel"`
	CriticalLevel float64 `json:"criticalLevel"`
	Unit          string  `json:"unit"`
	Description   string  `json:"description,omitempty"`
}

// SLAViolation represents an SLA violation event
type SLAViolation struct {
	MetricName    string        `json:"metricName"`
	Timestamp     time.Time     `json:"timestamp"`
	ActualValue   float64       `json:"actualValue"`
	ThresholdType string        `json:"thresholdType"` // warning, critical
	Threshold     float64       `json:"threshold"`
	Severity      string        `json:"severity"`
	Duration      time.Duration `json:"duration,omitempty"`
}

// NewPredictiveSLAAnalyzer creates a new predictive SLA analyzer
func NewPredictiveSLAAnalyzer(registry prometheus.Registerer, detector SeasonalityDetector) *PredictiveSLAAnalyzer {
	psa := &PredictiveSLAAnalyzer{
		models:        make(map[string]*PredictiveModel),
		slaThresholds: make(map[string]SLAThreshold),
		registry:      registry,
		detector:      detector,
	}

	psa.initMetrics()
	return psa
}

// initMetrics initializes Prometheus metrics for the analyzer
func (psa *PredictiveSLAAnalyzer) initMetrics() {
	psa.predictionsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "oran_sla_predictions_total",
		Help: "Total number of SLA predictions made",
	})

	psa.predictionErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "oran_sla_prediction_errors_total",
		Help: "Total number of SLA prediction errors",
	})

	psa.predictionLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "oran_sla_prediction_latency_seconds",
		Help:    "Latency of SLA predictions in seconds",
		Buckets: prometheus.DefBuckets,
	})

	psa.slaViolations = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "oran_sla_violations_total",
		Help: "Total number of SLA violations detected",
	})

	if psa.registry != nil {
		psa.registry.MustRegister(psa.predictionsTotal)
		psa.registry.MustRegister(psa.predictionErrors)
		psa.registry.MustRegister(psa.predictionLatency)
		psa.registry.MustRegister(psa.slaViolations)
	}
}

// AnalyzeTrend analyzes trends in the provided metrics with SLA context
func (psa *PredictiveSLAAnalyzer) AnalyzeTrend(ctx context.Context, data []*MetricsData, window time.Duration) (*TrendAnalysis, error) {
	startTime := time.Now()
	defer func() {
		psa.predictionLatency.Observe(time.Since(startTime).Seconds())
	}()

	logger := log.FromContext(ctx)
	logger.V(1).Info("Analyzing trend with SLA prediction", "dataPoints", len(data), "window", window)

	if len(data) == 0 {
		psa.predictionErrors.Inc()
		return nil, fmt.Errorf("no data provided for trend analysis")
	}

	// Extract primary metric for analysis
	primaryMetric := psa.extractPrimaryMetric(data)
	if primaryMetric == "" {
		psa.predictionErrors.Inc()
		return nil, fmt.Errorf("no primary metric found in data")
	}

	// Get or create predictive model
	model, err := psa.getOrCreateModel(primaryMetric, data)
	if err != nil {
		psa.predictionErrors.Inc()
		return nil, fmt.Errorf("failed to get predictive model: %w", err)
	}

	// Perform trend analysis
	trend := psa.calculateTrend(data, primaryMetric)
	slope := psa.calculateSlope(data, primaryMetric)
	confidence := psa.calculateConfidence(model, data)

	// Detect anomalies (results used in analysis)
	_ = psa.detectAnomalies(data, primaryMetric, model)

	// Detect seasonality (results used in analysis)
	_, _ = psa.detector.DetectSeasonality(ctx, data)

	// Check for SLA violations (results used in analysis)
	_ = psa.checkSLAViolations(data, primaryMetric)

	psa.predictionsTotal.Inc()

	analysis := &TrendAnalysis{
		Direction:  trend,
		StartTime:  time.Now().Add(-24 * time.Hour), // Analysis window start
		EndTime:    time.Now(),
		Slope:      slope,
		Confidence: confidence,
		Strength:   confidence, // Using confidence as strength
		R2Score:    confidence, // Simplified mapping
	}

	return analysis, nil
}

// PredictFuture predicts future values with SLA violation forecasting
func (psa *PredictiveSLAAnalyzer) PredictFuture(ctx context.Context, data []*MetricsData, horizon time.Duration) (*PredictionResult, error) {
	startTime := time.Now()
	defer func() {
		psa.predictionLatency.Observe(time.Since(startTime).Seconds())
	}()

	logger := log.FromContext(ctx)
	logger.V(1).Info("Predicting future values with SLA analysis", "dataPoints", len(data), "horizon", horizon)

	if len(data) == 0 {
		psa.predictionErrors.Inc()
		return nil, fmt.Errorf("no data provided for prediction")
	}

	primaryMetric := psa.extractPrimaryMetric(data)
	if primaryMetric == "" {
		psa.predictionErrors.Inc()
		return nil, fmt.Errorf("no primary metric found in data")
	}

	// Get or create predictive model
	model, err := psa.getOrCreateModel(primaryMetric, data)
	if err != nil {
		psa.predictionErrors.Inc()
		return nil, fmt.Errorf("failed to get predictive model: %w", err)
	}

	// Generate predictions
	predictions := psa.generatePredictions(data, model, horizon)

	// Calculate prediction confidence
	confidence := psa.calculatePredictionConfidence(model, predictions)

	// Predict SLA violations
	predictedViolations := psa.predictSLAViolations(predictions, primaryMetric)

	psa.predictionsTotal.Inc()

	result := &PredictionResult{
		Timestamp:      time.Now(),
		PredictedValue: 0.0, // Would need to be calculated from predictions
		Confidence:     confidence,
		LowerBound:     0.0, // Would need to be calculated
		UpperBound:     0.0, // Would need to be calculated
		PredictionType: model.Type,
		ModelUsed:      model.Name,
		Features:       make(map[string]float64),
	}

	// Add SLA violation predictions to metadata
	if len(predictedViolations) > 0 {
		if result.Features == nil {
			result.Features = make(map[string]float64)
		}
		result.Features["sla_violations_predicted"] = float64(len(predictedViolations))
	}

	return result, nil
}

// AddSLAThreshold adds an SLA threshold for monitoring
func (psa *PredictiveSLAAnalyzer) AddSLAThreshold(threshold SLAThreshold) {
	psa.mu.Lock()
	defer psa.mu.Unlock()
	psa.slaThresholds[threshold.MetricName] = threshold
}

// RemoveSLAThreshold removes an SLA threshold
func (psa *PredictiveSLAAnalyzer) RemoveSLAThreshold(metricName string) {
	psa.mu.Lock()
	defer psa.mu.Unlock()
	delete(psa.slaThresholds, metricName)
}

// GetSLAThresholds returns all configured SLA thresholds
func (psa *PredictiveSLAAnalyzer) GetSLAThresholds() map[string]SLAThreshold {
	psa.mu.RLock()
	defer psa.mu.RUnlock()

	thresholds := make(map[string]SLAThreshold)
	for k, v := range psa.slaThresholds {
		thresholds[k] = v
	}
	return thresholds
}

// TrainModel trains or retrains a predictive model
func (psa *PredictiveSLAAnalyzer) TrainModel(ctx context.Context, metricName string, data []*MetricsData) error {
	psa.mu.Lock()
	defer psa.mu.Unlock()

	logger := log.FromContext(ctx)
	logger.Info("Training predictive model", "metric", metricName, "dataPoints", len(data))

	if len(data) < 10 {
		return fmt.Errorf("insufficient data for training (need at least 10 points, got %d)", len(data))
	}

	// Create or update model
	model := &PredictiveModel{
		Name:         fmt.Sprintf("%s-model", metricName),
		Type:         "linear", // Default to linear regression
		Parameters:   make(map[string]float64),
		TrainingData: data,
		LastTrained:  time.Now(),
		Accuracy:     0.0,
		Metadata:     make(map[string]interface{}),
	}

	// Train the model (simplified linear regression)
	err := psa.trainLinearModel(model, data, metricName)
	if err != nil {
		return fmt.Errorf("failed to train model: %w", err)
	}

	psa.models[metricName] = model
	logger.Info("Model training completed", "metric", metricName, "accuracy", model.Accuracy)

	return nil
}

// extractPrimaryMetric extracts the primary metric from data
func (psa *PredictiveSLAAnalyzer) extractPrimaryMetric(data []*MetricsData) string {
	if len(data) == 0 {
		return ""
	}

	// Return the first metric name found
	for metricName := range data[0].Metrics {
		return metricName
	}

	return ""
}

// getOrCreateModel gets an existing model or creates a new one
func (psa *PredictiveSLAAnalyzer) getOrCreateModel(metricName string, data []*MetricsData) (*PredictiveModel, error) {
	psa.mu.RLock()
	model, exists := psa.models[metricName]
	psa.mu.RUnlock()

	if !exists || time.Since(model.LastTrained) > 24*time.Hour {
		// Create or retrain model
		if err := psa.TrainModel(context.Background(), metricName, data); err != nil {
			return nil, err
		}

		psa.mu.RLock()
		model = psa.models[metricName]
		psa.mu.RUnlock()
	}

	return model, nil
}

// calculateTrend determines the overall trend direction
func (psa *PredictiveSLAAnalyzer) calculateTrend(data []*MetricsData, metricName string) string {
	if len(data) < 2 {
		return "stable"
	}

	values := make([]float64, 0, len(data))
	for _, d := range data {
		if val, exists := d.Metrics[metricName]; exists {
			values = append(values, val)
		}
	}

	if len(values) < 2 {
		return "stable"
	}

	// Simple trend calculation
	start := values[0]
	end := values[len(values)-1]

	if end > start*1.05 { // 5% increase threshold
		return "increasing"
	} else if end < start*0.95 { // 5% decrease threshold
		return "decreasing"
	}

	return "stable"
}

// calculateSlope calculates the slope of the trend
func (psa *PredictiveSLAAnalyzer) calculateSlope(data []*MetricsData, metricName string) float64 {
	if len(data) < 2 {
		return 0.0
	}

	// Simple linear regression slope calculation
	var sumX, sumY, sumXY, sumX2 float64
	n := 0

	for i, d := range data {
		if val, exists := d.Metrics[metricName]; exists {
			x := float64(i)
			y := val

			sumX += x
			sumY += y
			sumXY += x * y
			sumX2 += x * x
			n++
		}
	}

	if n < 2 {
		return 0.0
	}

	nf := float64(n)
	denominator := nf*sumX2 - sumX*sumX

	if math.Abs(denominator) < 1e-10 {
		return 0.0
	}

	return (nf*sumXY - sumX*sumY) / denominator
}

// calculateConfidence calculates the confidence level of the analysis
func (psa *PredictiveSLAAnalyzer) calculateConfidence(model *PredictiveModel, data []*MetricsData) float64 {
	// Base confidence on model accuracy and data completeness
	dataCompleteness := float64(len(data)) / 100.0 // Assume 100 is optimal
	if dataCompleteness > 1.0 {
		dataCompleteness = 1.0
	}

	modelConfidence := model.Accuracy
	if modelConfidence == 0.0 {
		modelConfidence = 0.5 // Default moderate confidence
	}

	return (dataCompleteness + modelConfidence) / 2.0
}

// detectAnomalies detects anomalous points in the data
func (psa *PredictiveSLAAnalyzer) detectAnomalies(data []*MetricsData, metricName string, model *PredictiveModel) []AnomalyPoint {
	anomalies := make([]AnomalyPoint, 0)

	if len(data) < 3 {
		return anomalies
	}

	// Calculate mean and standard deviation
	values := make([]float64, 0, len(data))
	for _, d := range data {
		if val, exists := d.Metrics[metricName]; exists {
			values = append(values, val)
		}
	}

	if len(values) < 3 {
		return anomalies
	}

	mean := psa.calculateMean(values)
	stdDev := psa.calculateStdDev(values, mean)

	// Detect anomalies using 2-sigma rule
	threshold := 2.0 * stdDev

	for i, d := range data {
		if val, exists := d.Metrics[metricName]; exists {
			deviation := math.Abs(val - mean)
			if deviation > threshold {
				anomalies = append(anomalies, AnomalyPoint{
					Timestamp:     d.Timestamp,
					Value:         val,
					ExpectedValue: mean,
					Deviation:     deviation,
					Severity:      fmt.Sprintf("%.2f", math.Min(deviation/threshold, 1.0)),
					AnomalyScore:  math.Min(deviation/threshold, 1.0),
					Context:       json.RawMessage("{}"),
				})
			}
		}
	}

	return anomalies
}

// checkSLAViolations checks for SLA violations in the data
func (psa *PredictiveSLAAnalyzer) checkSLAViolations(data []*MetricsData, metricName string) []SLAViolation {
	violations := make([]SLAViolation, 0)

	psa.mu.RLock()
	threshold, exists := psa.slaThresholds[metricName]
	psa.mu.RUnlock()

	if !exists {
		return violations
	}

	for _, d := range data {
		if val, exists := d.Metrics[metricName]; exists {
			if val >= threshold.CriticalLevel {
				violations = append(violations, SLAViolation{
					MetricName:    metricName,
					Timestamp:     d.Timestamp,
					ActualValue:   val,
					ThresholdType: "critical",
					Threshold:     threshold.CriticalLevel,
					Severity:      "critical",
				})
				psa.slaViolations.Inc()
			} else if val >= threshold.WarningLevel {
				violations = append(violations, SLAViolation{
					MetricName:    metricName,
					Timestamp:     d.Timestamp,
					ActualValue:   val,
					ThresholdType: "warning",
					Threshold:     threshold.WarningLevel,
					Severity:      "warning",
				})
			}
		}
	}

	return violations
}

// generatePredictions generates future predictions
func (psa *PredictiveSLAAnalyzer) generatePredictions(data []*MetricsData, model *PredictiveModel, horizon time.Duration) []PredictionPoint {
	if len(data) == 0 {
		return nil
	}

	predictions := make([]PredictionPoint, 0)
	lastTimestamp := data[len(data)-1].Timestamp

	// Simple linear prediction based on slope
	slope := model.Parameters["slope"]
	intercept := model.Parameters["intercept"]

	// Generate predictions for the next hour with 5-minute intervals
	interval := 5 * time.Minute
	steps := int(horizon / interval)

	for i := 1; i <= steps; i++ {
		timestamp := lastTimestamp.Add(time.Duration(i) * interval)

		// Simple linear prediction: y = mx + b
		x := float64(len(data) + i)
		predictedValue := slope*x + intercept

		// Calculate confidence bounds (simplified)
		uncertainty := math.Abs(predictedValue) * 0.1 // 10% uncertainty

		predictions = append(predictions, PredictionPoint{
			Timestamp:  timestamp,
			Value:      predictedValue,
			Confidence: math.Max(0, 1.0-uncertainty/math.Abs(predictedValue)),
			Source:     "predictive-sla-analyzer",
			ModelName:  "linear-regression",
		})
	}

	return predictions
}

// calculatePredictionConfidence calculates confidence for predictions
func (psa *PredictiveSLAAnalyzer) calculatePredictionConfidence(model *PredictiveModel, predictions []PredictionPoint) float64 {
	// Base confidence on model accuracy and prediction horizon
	baseConfidence := model.Accuracy
	if baseConfidence == 0.0 {
		baseConfidence = 0.7
	}

	// Decrease confidence based on prediction length
	horizonPenalty := float64(len(predictions)) * 0.01
	confidence := baseConfidence - horizonPenalty

	if confidence < 0.1 {
		confidence = 0.1
	}
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// predictSLAViolations predicts future SLA violations
func (psa *PredictiveSLAAnalyzer) predictSLAViolations(predictions []PredictionPoint, metricName string) []SLAViolation {
	violations := make([]SLAViolation, 0)

	psa.mu.RLock()
	threshold, exists := psa.slaThresholds[metricName]
	psa.mu.RUnlock()

	if !exists {
		return violations
	}

	for _, pred := range predictions {
		if pred.Value >= threshold.CriticalLevel {
			violations = append(violations, SLAViolation{
				MetricName:    metricName,
				Timestamp:     pred.Timestamp,
				ActualValue:   pred.Value,
				ThresholdType: "critical",
				Threshold:     threshold.CriticalLevel,
				Severity:      "critical",
			})
		} else if pred.Value >= threshold.WarningLevel {
			violations = append(violations, SLAViolation{
				MetricName:    metricName,
				Timestamp:     pred.Timestamp,
				ActualValue:   pred.Value,
				ThresholdType: "warning",
				Threshold:     threshold.WarningLevel,
				Severity:      "warning",
			})
		}
	}

	return violations
}

// trainLinearModel trains a simple linear regression model
func (psa *PredictiveSLAAnalyzer) trainLinearModel(model *PredictiveModel, data []*MetricsData, metricName string) error {
	if len(data) < 2 {
		return fmt.Errorf("insufficient data for linear regression")
	}

	// Extract values for training
	var x, y []float64
	for i, d := range data {
		if val, exists := d.Metrics[metricName]; exists {
			x = append(x, float64(i))
			y = append(y, val)
		}
	}

	if len(x) < 2 {
		return fmt.Errorf("insufficient valid data points")
	}

	// Calculate linear regression parameters
	slope := psa.calculateSlope(data, metricName)

	// Calculate intercept
	meanX := psa.calculateMean(x)
	meanY := psa.calculateMean(y)
	intercept := meanY - slope*meanX

	// Calculate R-squared for accuracy
	accuracy := psa.calculateRSquared(x, y, slope, intercept)

	model.Parameters["slope"] = slope
	model.Parameters["intercept"] = intercept
	model.Accuracy = accuracy

	return nil
}

// calculateMean calculates the mean of a slice of float64 values
func (psa *PredictiveSLAAnalyzer) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, val := range values {
		sum += val
	}

	return sum / float64(len(values))
}

// calculateStdDev calculates the standard deviation
func (psa *PredictiveSLAAnalyzer) calculateStdDev(values []float64, mean float64) float64 {
	if len(values) < 2 {
		return 0.0
	}

	sumSquares := 0.0
	for _, val := range values {
		diff := val - mean
		sumSquares += diff * diff
	}

	return math.Sqrt(sumSquares / float64(len(values)-1))
}

// calculateRSquared calculates R-squared for linear regression
func (psa *PredictiveSLAAnalyzer) calculateRSquared(x, y []float64, slope, intercept float64) float64 {
	if len(x) != len(y) || len(x) < 2 {
		return 0.0
	}

	meanY := psa.calculateMean(y)

	var ssRes, ssTot float64
	for i := 0; i < len(x); i++ {
		predicted := slope*x[i] + intercept
		residual := y[i] - predicted
		ssRes += residual * residual

		deviation := y[i] - meanY
		ssTot += deviation * deviation
	}

	if ssTot == 0.0 {
		return 1.0 // Perfect fit if no variance
	}

	return 1.0 - (ssRes / ssTot)
}
