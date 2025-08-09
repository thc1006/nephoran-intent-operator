package automation

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Metrics for failure prediction
	failurePredictionAccuracy = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "nephoran_failure_prediction_accuracy",
		Help: "Accuracy of failure prediction models",
	}, []string{"component", "model_type"})

	failurePredictionLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "nephoran_failure_prediction_duration_seconds",
		Help:    "Duration of failure prediction operations",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
	}, []string{"component", "operation"})

	predictedFailuresProbability = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "nephoran_predicted_failure_probability",
		Help: "Probability of predicted failures for components",
	}, []string{"component", "failure_type"})
)

// NewFailurePrediction creates a new failure prediction system
func NewFailurePrediction(config *SelfHealingConfig, logger *slog.Logger) (*FailurePrediction, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	fp := &FailurePrediction{
		config:               config,
		logger:               logger,
		predictionModels:     make(map[string]*PredictionModel),
		failureProbabilities: make(map[string]float64),
		predictionAccuracy:   make(map[string]float64),
		historicalData:       NewHistoricalDataStore(),
		anomalyDetector:      NewAnomalyDetector(),
	}

	// Initialize prediction models for each component
	for componentName := range config.ComponentConfigs {
		model := &PredictionModel{
			Name:             fmt.Sprintf("%s-predictor", componentName),
			Component:        componentName,
			ModelType:        "ARIMA", // Time series analysis for telecom workloads
			Features:         []string{"cpu_usage", "memory_usage", "error_rate", "response_time", "queue_depth"},
			Accuracy:         0.75, // Initial accuracy
			LastTraining:     time.Now().Add(-24 * time.Hour),
			PredictionWindow: 30 * time.Minute,
			Thresholds: map[string]float64{
				"failure_probability": 0.8,
				"anomaly_score":       0.7,
				"degradation_rate":    0.6,
			},
		}
		fp.predictionModels[componentName] = model
		fp.failureProbabilities[componentName] = 0.0
		fp.predictionAccuracy[componentName] = model.Accuracy
	}

	return fp, nil
}

// Start starts the failure prediction system
func (fp *FailurePrediction) Start(ctx context.Context) {
	fp.logger.Info("Starting failure prediction system")

	// Start prediction loop
	go fp.runPredictionLoop(ctx)

	// Start model training loop
	go fp.runModelTrainingLoop(ctx)

	// Start anomaly detection
	go fp.anomalyDetector.Start(ctx)

	fp.logger.Info("Failure prediction system started successfully")
}

// runPredictionLoop runs the main prediction loop
func (fp *FailurePrediction) runPredictionLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute) // Predict every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fp.performPredictions(ctx)
		}
	}
}

// runModelTrainingLoop periodically retrains models
func (fp *FailurePrediction) runModelTrainingLoop(ctx context.Context) {
	ticker := time.NewTicker(6 * time.Hour) // Retrain every 6 hours
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fp.retrainModels(ctx)
		}
	}
}

// performPredictions performs failure predictions for all components
func (fp *FailurePrediction) performPredictions(ctx context.Context) {
	start := time.Now()
	defer func() {
		failurePredictionLatency.WithLabelValues("all", "prediction").Observe(time.Since(start).Seconds())
	}()

	fp.logger.Debug("Performing failure predictions")

	fp.mu.Lock()
	defer fp.mu.Unlock()

	for componentName, model := range fp.predictionModels {
		probability := fp.predictFailureProbability(ctx, componentName, model)
		fp.failureProbabilities[componentName] = probability

		// Update metrics
		predictedFailuresProbability.WithLabelValues(componentName, "overall").Set(probability)
		failurePredictionAccuracy.WithLabelValues(componentName, model.ModelType).Set(model.Accuracy)

		// Log high-probability predictions
		if probability > model.Thresholds["failure_probability"] {
			fp.logger.Warn("High failure probability predicted",
				"component", componentName,
				"probability", probability,
				"model", model.ModelType)

			// Record prediction for accuracy tracking
			fp.recordPrediction(componentName, probability, time.Now().Add(model.PredictionWindow))
		}
	}
}

// predictFailureProbability predicts failure probability for a component
func (fp *FailurePrediction) predictFailureProbability(ctx context.Context, componentName string, model *PredictionModel) float64 {
	start := time.Now()
	defer func() {
		failurePredictionLatency.WithLabelValues(componentName, "single_prediction").Observe(time.Since(start).Seconds())
	}()

	// Get recent historical data
	historicalData := fp.historicalData.GetRecentData(componentName, 24*time.Hour)
	if len(historicalData) < 10 {
		// Not enough data for prediction
		return 0.0
	}

	// Perform prediction based on model type
	switch model.ModelType {
	case "ARIMA":
		return fp.predictWithARIMA(historicalData, model)
	case "NEURAL_NETWORK":
		return fp.predictWithNeuralNetwork(historicalData, model)
	case "LINEAR_REGRESSION":
		return fp.predictWithLinearRegression(historicalData, model)
	default:
		return fp.predictWithARIMA(historicalData, model) // Default to ARIMA
	}
}

// predictWithARIMA performs ARIMA-based time series prediction
func (fp *FailurePrediction) predictWithARIMA(data []DataPoint, model *PredictionModel) float64 {
	if len(data) < 20 {
		return 0.0
	}

	// Simplified ARIMA implementation for demonstration
	// In production, would use proper time series analysis library

	// Calculate trend and seasonality
	trend := fp.calculateTrend(data)
	seasonality := fp.calculateSeasonality(data)
	volatility := fp.calculateVolatility(data)

	// Combine factors for failure probability
	failureProbability := 0.0

	// High upward trend in error rate or resource usage
	if trend["error_rate"] > 0.05 {
		failureProbability += 0.3
	}
	if trend["cpu_usage"] > 0.1 || trend["memory_usage"] > 0.1 {
		failureProbability += 0.2
	}
	if trend["response_time"] > 0.1 {
		failureProbability += 0.2
	}

	// High volatility indicates instability
	if volatility > 0.3 {
		failureProbability += 0.2
	}

	// Seasonal patterns suggesting problems
	if seasonality > 0.5 {
		failureProbability += 0.1
	}

	// Ensure probability is within [0, 1]
	return math.Min(1.0, math.Max(0.0, failureProbability))
}

// predictWithNeuralNetwork performs neural network-based prediction
func (fp *FailurePrediction) predictWithNeuralNetwork(data []DataPoint, model *PredictionModel) float64 {
	// Simplified neural network simulation
	// In production, would use proper ML framework like TensorFlow or PyTorch

	if len(data) < 50 {
		return 0.0
	}

	// Feature extraction
	features := fp.extractFeatures(data)

	// Simulate neural network prediction
	// Using weighted combination of features
	weights := map[string]float64{
		"cpu_trend":     0.25,
		"memory_trend":  0.20,
		"error_rate":    0.30,
		"response_time": 0.15,
		"queue_depth":   0.10,
	}

	prediction := 0.0
	for feature, weight := range weights {
		if value, exists := features[feature]; exists {
			prediction += value * weight
		}
	}

	// Apply sigmoid activation function
	return 1.0 / (1.0 + math.Exp(-prediction))
}

// predictWithLinearRegression performs linear regression-based prediction
func (fp *FailurePrediction) predictWithLinearRegression(data []DataPoint, model *PredictionModel) float64 {
	if len(data) < 10 {
		return 0.0
	}

	// Simple linear regression on recent trend
	n := len(data)
	recentData := data[max(0, n-20):] // Use last 20 data points

	// Calculate linear regression for key metrics
	errorRateSlope := fp.calculateLinearSlope(recentData, "error_rate")
	responseTimeSlope := fp.calculateLinearSlope(recentData, "response_time")
	cpuSlope := fp.calculateLinearSlope(recentData, "cpu_usage")

	// Combine slopes to predict failure probability
	failureProbability := 0.0

	if errorRateSlope > 0.01 {
		failureProbability += 0.4
	}
	if responseTimeSlope > 50 { // 50ms increase per data point
		failureProbability += 0.3
	}
	if cpuSlope > 0.05 { // 5% increase per data point
		failureProbability += 0.3
	}

	return math.Min(1.0, failureProbability)
}

// retrainModels retrains prediction models with recent data
func (fp *FailurePrediction) retrainModels(ctx context.Context) {
	fp.logger.Info("Retraining prediction models")

	fp.mu.Lock()
	defer fp.mu.Unlock()

	for componentName, model := range fp.predictionModels {
		start := time.Now()

		// Get training data (last 7 days)
		trainingData := fp.historicalData.GetRecentData(componentName, 7*24*time.Hour)
		if len(trainingData) < 100 {
			fp.logger.Warn("Insufficient training data", "component", componentName, "data_points", len(trainingData))
			continue
		}

		// Calculate new model accuracy based on recent predictions
		newAccuracy := fp.calculateModelAccuracy(componentName, 24*time.Hour)
		if newAccuracy > 0 {
			model.Accuracy = newAccuracy
			fp.predictionAccuracy[componentName] = newAccuracy
		}

		// Update model parameters based on training data
		fp.updateModelParameters(model, trainingData)

		model.LastTraining = time.Now()

		fp.logger.Info("Model retrained",
			"component", componentName,
			"accuracy", model.Accuracy,
			"training_data_points", len(trainingData),
			"duration", time.Since(start))
	}
}

// GetFailureProbabilities returns current failure probabilities
func (fp *FailurePrediction) GetFailureProbabilities() map[string]float64 {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	probabilities := make(map[string]float64)
	for k, v := range fp.failureProbabilities {
		probabilities[k] = v
	}
	return probabilities
}

// GetPredictionAccuracy returns prediction accuracy for each component
func (fp *FailurePrediction) GetPredictionAccuracy() map[string]float64 {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	accuracy := make(map[string]float64)
	for k, v := range fp.predictionAccuracy {
		accuracy[k] = v
	}
	return accuracy
}

// Helper methods

func (fp *FailurePrediction) calculateTrend(data []DataPoint) map[string]float64 {
	trends := make(map[string]float64)

	if len(data) < 2 {
		return trends
	}

	// Calculate simple trend as slope over time
	for _, metric := range []string{"error_rate", "cpu_usage", "memory_usage", "response_time"} {
		slope := fp.calculateLinearSlope(data, metric)
		trends[metric] = slope
	}

	return trends
}

func (fp *FailurePrediction) calculateSeasonality(data []DataPoint) float64 {
	// Simplified seasonality calculation
	// In production, would use FFT or autocorrelation
	if len(data) < 24 {
		return 0.0
	}

	// Check for 24-hour patterns (hourly data)
	var seasonalVariance float64
	for i := 24; i < len(data); i++ {
		diff := math.Abs(data[i].Metrics["cpu_usage"] - data[i-24].Metrics["cpu_usage"])
		seasonalVariance += diff
	}

	return seasonalVariance / float64(len(data)-24)
}

func (fp *FailurePrediction) calculateVolatility(data []DataPoint) float64 {
	if len(data) < 2 {
		return 0.0
	}

	// Calculate coefficient of variation for CPU usage
	var sum, sumSquares float64
	for _, point := range data {
		value := point.Metrics["cpu_usage"]
		sum += value
		sumSquares += value * value
	}

	n := float64(len(data))
	mean := sum / n
	variance := (sumSquares / n) - (mean * mean)

	if mean == 0 {
		return 0.0
	}

	return math.Sqrt(variance) / mean // Coefficient of variation
}

func (fp *FailurePrediction) extractFeatures(data []DataPoint) map[string]float64 {
	features := make(map[string]float64)

	if len(data) == 0 {
		return features
	}

	// Recent values
	recent := data[len(data)-1]
	features["cpu_usage"] = recent.Metrics["cpu_usage"]
	features["memory_usage"] = recent.Metrics["memory_usage"]
	features["error_rate"] = recent.Metrics["error_rate"]
	features["response_time"] = recent.Metrics["response_time"]

	// Trends
	trends := fp.calculateTrend(data)
	for metric, trend := range trends {
		features[metric+"_trend"] = trend
	}

	// Volatility
	features["volatility"] = fp.calculateVolatility(data)

	return features
}

func (fp *FailurePrediction) calculateLinearSlope(data []DataPoint, metric string) float64 {
	n := len(data)
	if n < 2 {
		return 0.0
	}

	// Simple linear regression: y = mx + b, return m (slope)
	var sumX, sumY, sumXY, sumXX float64

	for i, point := range data {
		x := float64(i)
		y := point.Metrics[metric]
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}

	nFloat := float64(n)
	denominator := nFloat*sumXX - sumX*sumX
	if denominator == 0 {
		return 0.0
	}

	return (nFloat*sumXY - sumX*sumY) / denominator
}

func (fp *FailurePrediction) calculateModelAccuracy(componentName string, period time.Duration) float64 {
	// Get prediction history for the component
	predictions := fp.historicalData.GetPredictionHistory(componentName, period)
	if len(predictions) == 0 {
		return 0.0
	}

	correctPredictions := 0
	for _, prediction := range predictions {
		if prediction.WasAccurate {
			correctPredictions++
		}
	}

	return float64(correctPredictions) / float64(len(predictions))
}

func (fp *FailurePrediction) updateModelParameters(model *PredictionModel, trainingData []DataPoint) {
	// Update model thresholds based on training data statistics
	if len(trainingData) == 0 {
		return
	}

	// Calculate statistics for threshold adjustment
	var errorRates, responseTimes []float64
	for _, point := range trainingData {
		errorRates = append(errorRates, point.Metrics["error_rate"])
		responseTimes = append(responseTimes, point.Metrics["response_time"])
	}

	// Update thresholds based on percentiles
	model.Thresholds["failure_probability"] = fp.calculatePercentile(errorRates, 0.95)
	model.Thresholds["anomaly_score"] = fp.calculatePercentile(responseTimes, 0.90)
}

func (fp *FailurePrediction) calculatePercentile(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	// Simple percentile calculation (would use sort in production)
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	// Use mean as approximation (simplified)
	return mean * (1.0 + percentile)
}

func (fp *FailurePrediction) recordPrediction(componentName string, probability float64, predictedTime time.Time) {
	// Record prediction for later accuracy evaluation
	fp.historicalData.RecordPrediction(componentName, PredictionRecord{
		Timestamp:     time.Now(),
		Component:     componentName,
		Probability:   probability,
		PredictedTime: predictedTime,
		WasAccurate:   false, // Will be updated when prediction time arrives
	})
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Supporting types for historical data storage

type DataPoint struct {
	Timestamp time.Time          `json:"timestamp"`
	Component string             `json:"component"`
	Metrics   map[string]float64 `json:"metrics"`
}

type PredictionRecord struct {
	Timestamp     time.Time `json:"timestamp"`
	Component     string    `json:"component"`
	Probability   float64   `json:"probability"`
	PredictedTime time.Time `json:"predicted_time"`
	WasAccurate   bool      `json:"was_accurate"`
}

// HistoricalDataStore manages historical data for ML training
type HistoricalDataStore struct {
	mu            sync.RWMutex
	data          map[string][]DataPoint
	predictions   map[string][]PredictionRecord
	maxDataPoints int
}

func NewHistoricalDataStore() *HistoricalDataStore {
	return &HistoricalDataStore{
		data:          make(map[string][]DataPoint),
		predictions:   make(map[string][]PredictionRecord),
		maxDataPoints: 10000, // Keep last 10k data points per component
	}
}

func (hds *HistoricalDataStore) GetRecentData(component string, duration time.Duration) []DataPoint {
	hds.mu.RLock()
	defer hds.mu.RUnlock()

	data, exists := hds.data[component]
	if !exists {
		return []DataPoint{}
	}

	cutoff := time.Now().Add(-duration)
	var recent []DataPoint

	for _, point := range data {
		if point.Timestamp.After(cutoff) {
			recent = append(recent, point)
		}
	}

	return recent
}

func (hds *HistoricalDataStore) GetPredictionHistory(component string, duration time.Duration) []PredictionRecord {
	hds.mu.RLock()
	defer hds.mu.RUnlock()

	predictions, exists := hds.predictions[component]
	if !exists {
		return []PredictionRecord{}
	}

	cutoff := time.Now().Add(-duration)
	var recent []PredictionRecord

	for _, pred := range predictions {
		if pred.Timestamp.After(cutoff) {
			recent = append(recent, pred)
		}
	}

	return recent
}

func (hds *HistoricalDataStore) RecordPrediction(component string, record PredictionRecord) {
	hds.mu.Lock()
	defer hds.mu.Unlock()

	if _, exists := hds.predictions[component]; !exists {
		hds.predictions[component] = make([]PredictionRecord, 0)
	}

	hds.predictions[component] = append(hds.predictions[component], record)

	// Limit size
	if len(hds.predictions[component]) > hds.maxDataPoints {
		hds.predictions[component] = hds.predictions[component][1:]
	}
}

// AnomalyDetector detects anomalies in system behavior
type AnomalyDetector struct {
	mu         sync.RWMutex
	thresholds map[string]float64
}

func NewAnomalyDetector() *AnomalyDetector {
	return &AnomalyDetector{
		thresholds: map[string]float64{
			"cpu_spike":     0.9,  // 90% CPU usage
			"memory_spike":  0.85, // 85% memory usage
			"error_burst":   0.1,  // 10% error rate
			"latency_spike": 2.0,  // 2x normal latency
		},
	}
}

func (ad *AnomalyDetector) Start(ctx context.Context) {
	// Anomaly detection would run continuously
	// For now, it's a placeholder
}
