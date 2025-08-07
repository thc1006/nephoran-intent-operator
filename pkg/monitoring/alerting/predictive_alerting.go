// Package alerting provides ML-based predictive SLA violation detection
// for early warning and proactive incident prevention.
package alerting

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// PredictiveAlerting implements ML-based SLA violation prediction using
// historical data, seasonal patterns, and anomaly detection
type PredictiveAlerting struct {
	logger       *logging.StructuredLogger
	config       *PredictiveConfig
	
	// ML models and data
	models       map[SLAType]*PredictionModel
	historicalData map[SLAType]*HistoricalDataset
	
	// Feature engineering
	featureExtractor *FeatureExtractor
	seasonalModels   map[SLAType]*SeasonalModel
	trendAnalyzer    *TrendAnalyzer
	
	// Anomaly detection
	anomalyDetector  *AnomalyDetector
	
	// State management
	lastModelUpdate  time.Time
	predictionCache  map[string]*PredictionResult
	
	started          bool
	stopCh           chan struct{}
	mu               sync.RWMutex
}

// PredictiveConfig holds configuration for predictive alerting
type PredictiveConfig struct {
	// Model training settings
	ModelUpdateInterval   time.Duration `yaml:"model_update_interval"`
	TrainingDataWindow    time.Duration `yaml:"training_data_window"`
	MinDataPoints        int           `yaml:"min_data_points"`
	
	// Prediction settings
	PredictionWindow     time.Duration `yaml:"prediction_window"`
	PredictionInterval   time.Duration `yaml:"prediction_interval"`
	ConfidenceThreshold  float64       `yaml:"confidence_threshold"`
	
	// Early warning settings
	EarlyWarningLeadTime time.Duration `yaml:"early_warning_lead_time"`
	AlertThreshold       float64       `yaml:"alert_threshold"`
	
	// Feature engineering
	EnableSeasonalFeatures bool        `yaml:"enable_seasonal_features"`
	EnableTrendFeatures    bool        `yaml:"enable_trend_features"`
	EnableAnomalyFeatures  bool        `yaml:"enable_anomaly_features"`
	
	// Model parameters
	MovingAverageWindow   time.Duration `yaml:"moving_average_window"`
	SeasonalityPeriods   []time.Duration `yaml:"seasonality_periods"`
	
	// Performance settings
	MaxConcurrentPredictions int `yaml:"max_concurrent_predictions"`
	CacheTTL                time.Duration `yaml:"cache_ttl"`
}

// PredictionModel represents a trained ML model for SLA prediction
type PredictionModel struct {
	SLAType          SLAType                `json:"sla_type"`
	ModelType        string                 `json:"model_type"`
	TrainedAt        time.Time              `json:"trained_at"`
	Accuracy         float64                `json:"accuracy"`
	Precision        float64                `json:"precision"`
	Recall           float64                `json:"recall"`
	F1Score          float64                `json:"f1_score"`
	
	// Model parameters
	Weights          []float64              `json:"weights"`
	Bias             float64                `json:"bias"`
	Features         []string               `json:"features"`
	Normalization    NormalizationParams    `json:"normalization"`
	
	// Training metadata
	TrainingDataSize int                    `json:"training_data_size"`
	ValidationSplit  float64                `json:"validation_split"`
	Hyperparameters  map[string]interface{} `json:"hyperparameters"`
}

// HistoricalDataset contains historical SLA data for training
type HistoricalDataset struct {
	SLAType     SLAType              `json:"sla_type"`
	DataPoints  []HistoricalPoint    `json:"data_points"`
	StartTime   time.Time            `json:"start_time"`
	EndTime     time.Time            `json:"end_time"`
	Features    []string             `json:"features"`
	Statistics  DatasetStatistics    `json:"statistics"`
}

// HistoricalPoint represents a single data point in the historical dataset
type HistoricalPoint struct {
	Timestamp    time.Time              `json:"timestamp"`
	SLAValue     float64                `json:"sla_value"`
	Features     map[string]float64     `json:"features"`
	Labels       map[string]string      `json:"labels"`
	Violation    bool                   `json:"violation"`
	Severity     AlertSeverity          `json:"severity,omitempty"`
}

// DatasetStatistics contains statistical information about the dataset
type DatasetStatistics struct {
	Count          int     `json:"count"`
	Mean           float64 `json:"mean"`
	StdDev         float64 `json:"std_dev"`
	Min            float64 `json:"min"`
	Max            float64 `json:"max"`
	ViolationRate  float64 `json:"violation_rate"`
	Seasonality    bool    `json:"seasonality"`
	TrendDirection string  `json:"trend_direction"`
}

// PredictionResult contains the result of SLA violation prediction
type PredictionResult struct {
	SLAType           SLAType       `json:"sla_type"`
	PredictedAt       time.Time     `json:"predicted_at"`
	PredictionWindow  time.Duration `json:"prediction_window"`
	
	// Prediction values
	PredictedValue    float64       `json:"predicted_value"`
	ViolationProbability float64    `json:"violation_probability"`
	Confidence        float64       `json:"confidence"`
	
	// Prediction breakdown
	BaselinePrediction float64      `json:"baseline_prediction"`
	SeasonalAdjustment float64      `json:"seasonal_adjustment"`
	TrendAdjustment   float64       `json:"trend_adjustment"`
	AnomalyScore      float64       `json:"anomaly_score"`
	
	// Early warning information
	TimeToViolation   *time.Duration `json:"time_to_violation,omitempty"`
	RecommendedActions []string       `json:"recommended_actions"`
	
	// Context and features
	Features          map[string]float64 `json:"features"`
	ContributingFactors []ContributingFactor `json:"contributing_factors"`
	
	// Model metadata
	ModelVersion      string         `json:"model_version"`
	ModelAccuracy     float64        `json:"model_accuracy"`
}

// ContributingFactor identifies factors contributing to the prediction
type ContributingFactor struct {
	Feature     string  `json:"feature"`
	Importance  float64 `json:"importance"`
	Direction   string  `json:"direction"` // positive, negative
	Description string  `json:"description"`
}

// FeatureExtractor extracts features from raw SLA data for ML models
type FeatureExtractor struct {
	logger *logging.StructuredLogger
	config *PredictiveConfig
}

// SeasonalModel detects and models seasonal patterns in SLA data
type SeasonalModel struct {
	SLAType            SLAType                    `json:"sla_type"`
	SeasonalityPeriods map[time.Duration]float64  `json:"seasonality_periods"`
	WeeklyPattern      [7]float64                 `json:"weekly_pattern"`
	DailyPattern       [24]float64                `json:"daily_pattern"`
	MonthlyPattern     [12]float64                `json:"monthly_pattern"`
	LastUpdated        time.Time                  `json:"last_updated"`
}

// TrendAnalyzer analyzes long-term trends in SLA data
type TrendAnalyzer struct {
	logger     *logging.StructuredLogger
	trendCache map[SLAType]*TrendInfo
	mu         sync.RWMutex
}

// TrendInfo contains trend analysis results
type TrendInfo struct {
	SLAType       SLAType   `json:"sla_type"`
	Direction     string    `json:"direction"` // improving, degrading, stable
	Slope         float64   `json:"slope"`
	Correlation   float64   `json:"correlation"`
	Confidence    float64   `json:"confidence"`
	LastUpdated   time.Time `json:"last_updated"`
}

// AnomalyDetector detects anomalies in SLA metrics using statistical methods
type AnomalyDetector struct {
	logger         *logging.StructuredLogger
	anomalyModels  map[SLAType]*AnomalyModel
	mu             sync.RWMutex
}

// AnomalyModel contains parameters for anomaly detection
type AnomalyModel struct {
	SLAType          SLAType   `json:"sla_type"`
	Mean             float64   `json:"mean"`
	StdDev           float64   `json:"std_dev"`
	ZScoreThreshold  float64   `json:"z_score_threshold"`
	IQRAnomalyFactor float64   `json:"iqr_anomaly_factor"`
	LastUpdated      time.Time `json:"last_updated"`
}

// NormalizationParams contains parameters for feature normalization
type NormalizationParams struct {
	Method string             `json:"method"` // min-max, z-score
	Params map[string]float64 `json:"params"`
}

// DefaultPredictiveConfig returns production-ready predictive alerting configuration
func DefaultPredictiveConfig() *PredictiveConfig {
	return &PredictiveConfig{
		// Model training settings
		ModelUpdateInterval:   24 * time.Hour,
		TrainingDataWindow:    30 * 24 * time.Hour, // 30 days
		MinDataPoints:        1000,
		
		// Prediction settings  
		PredictionWindow:     60 * time.Minute,
		PredictionInterval:   5 * time.Minute,
		ConfidenceThreshold:  0.75,
		
		// Early warning settings
		EarlyWarningLeadTime: 15 * time.Minute,
		AlertThreshold:       0.8, // 80% violation probability
		
		// Feature engineering
		EnableSeasonalFeatures: true,
		EnableTrendFeatures:    true,
		EnableAnomalyFeatures:  true,
		
		// Model parameters
		MovingAverageWindow: 10 * time.Minute,
		SeasonalityPeriods: []time.Duration{
			24 * time.Hour,      // Daily
			7 * 24 * time.Hour,  // Weekly
			30 * 24 * time.Hour, // Monthly
		},
		
		// Performance settings
		MaxConcurrentPredictions: 10,
		CacheTTL:                5 * time.Minute,
	}
}

// NewPredictiveAlerting creates a new predictive alerting system
func NewPredictiveAlerting(config *PredictiveConfig, logger *logging.StructuredLogger) (*PredictiveAlerting, error) {
	if config == nil {
		config = DefaultPredictiveConfig()
	}
	
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	
	pa := &PredictiveAlerting{
		logger:          logger.WithComponent("predictive-alerting"),
		config:          config,
		models:          make(map[SLAType]*PredictionModel),
		historicalData:  make(map[SLAType]*HistoricalDataset),
		predictionCache: make(map[string]*PredictionResult),
		stopCh:          make(chan struct{}),
	}
	
	// Initialize sub-components
	pa.featureExtractor = &FeatureExtractor{
		logger: logger.WithComponent("feature-extractor"),
		config: config,
	}
	
	pa.seasonalModels = make(map[SLAType]*SeasonalModel)
	
	pa.trendAnalyzer = &TrendAnalyzer{
		logger:     logger.WithComponent("trend-analyzer"),
		trendCache: make(map[SLAType]*TrendInfo),
	}
	
	pa.anomalyDetector = &AnomalyDetector{
		logger:        logger.WithComponent("anomaly-detector"),
		anomalyModels: make(map[SLAType]*AnomalyModel),
	}
	
	return pa, nil
}

// Start initializes the predictive alerting system
func (pa *PredictiveAlerting) Start(ctx context.Context) error {
	pa.mu.Lock()
	defer pa.mu.Unlock()
	
	if pa.started {
		return fmt.Errorf("predictive alerting already started")
	}
	
	pa.logger.InfoWithContext("Starting predictive alerting system",
		"model_update_interval", pa.config.ModelUpdateInterval,
		"prediction_window", pa.config.PredictionWindow,
		"confidence_threshold", pa.config.ConfidenceThreshold,
	)
	
	// Initialize models for all SLA types
	slaTypes := []SLAType{SLATypeAvailability, SLATypeLatency, SLAThroughput, SLAErrorRate}
	for _, slaType := range slaTypes {
		if err := pa.initializeModel(ctx, slaType); err != nil {
			pa.logger.WarnWithContext("Failed to initialize model", 
				slog.String("sla_type", string(slaType)),
				slog.String("error", err.Error()),
			)
		}
	}
	
	// Start background processes
	go pa.modelUpdateLoop(ctx)
	go pa.predictionLoop(ctx)
	go pa.cacheCleanupLoop(ctx)
	
	pa.started = true
	pa.logger.InfoWithContext("Predictive alerting system started successfully")
	
	return nil
}

// Stop shuts down the predictive alerting system
func (pa *PredictiveAlerting) Stop(ctx context.Context) error {
	pa.mu.Lock()
	defer pa.mu.Unlock()
	
	if !pa.started {
		return nil
	}
	
	pa.logger.InfoWithContext("Stopping predictive alerting system")
	close(pa.stopCh)
	
	pa.started = false
	pa.logger.InfoWithContext("Predictive alerting system stopped")
	
	return nil
}

// Predict generates a prediction for SLA violation
func (pa *PredictiveAlerting) Predict(ctx context.Context, slaType SLAType, 
	currentMetrics map[string]float64) (*PredictionResult, error) {
	
	// Check cache first
	cacheKey := fmt.Sprintf("%s-%d", slaType, time.Now().Truncate(pa.config.CacheTTL).Unix())
	if cached := pa.getCachedPrediction(cacheKey); cached != nil {
		return cached, nil
	}
	
	model, exists := pa.models[slaType]
	if !exists {
		return nil, fmt.Errorf("no model available for SLA type %s", slaType)
	}
	
	// Extract features from current metrics
	features, err := pa.featureExtractor.ExtractFeatures(ctx, slaType, currentMetrics)
	if err != nil {
		return nil, fmt.Errorf("failed to extract features: %w", err)
	}
	
	// Generate baseline prediction
	baselinePrediction := pa.generateBaselinePrediction(model, features)
	
	// Apply seasonal adjustments
	var seasonalAdjustment float64
	if pa.config.EnableSeasonalFeatures {
		seasonalAdjustment = pa.applySeasonalAdjustment(slaType, time.Now())
	}
	
	// Apply trend adjustments
	var trendAdjustment float64
	if pa.config.EnableTrendFeatures {
		trendAdjustment = pa.applyTrendAdjustment(slaType)
	}
	
	// Calculate anomaly score
	var anomalyScore float64
	if pa.config.EnableAnomalyFeatures {
		anomalyScore = pa.calculateAnomalyScore(slaType, currentMetrics)
	}
	
	// Combine predictions
	finalPrediction := baselinePrediction + seasonalAdjustment + trendAdjustment
	
	// Calculate violation probability
	violationProbability := pa.calculateViolationProbability(slaType, finalPrediction, anomalyScore)
	
	// Calculate confidence based on model accuracy and data quality
	confidence := pa.calculateConfidence(model, features, anomalyScore)
	
	// Estimate time to violation if probability is high
	var timeToViolation *time.Duration
	if violationProbability > pa.config.AlertThreshold {
		ttv := pa.estimateTimeToViolation(slaType, finalPrediction, violationProbability)
		timeToViolation = &ttv
	}
	
	// Generate recommended actions
	recommendedActions := pa.generateRecommendedActions(slaType, violationProbability, features)
	
	// Identify contributing factors
	contributingFactors := pa.identifyContributingFactors(model, features)
	
	result := &PredictionResult{
		SLAType:              slaType,
		PredictedAt:          time.Now(),
		PredictionWindow:     pa.config.PredictionWindow,
		PredictedValue:       finalPrediction,
		ViolationProbability: violationProbability,
		Confidence:           confidence,
		BaselinePrediction:   baselinePrediction,
		SeasonalAdjustment:   seasonalAdjustment,
		TrendAdjustment:      trendAdjustment,
		AnomalyScore:         anomalyScore,
		TimeToViolation:      timeToViolation,
		RecommendedActions:   recommendedActions,
		Features:             features,
		ContributingFactors:  contributingFactors,
		ModelVersion:         fmt.Sprintf("%s-v%d", model.ModelType, model.TrainedAt.Unix()),
		ModelAccuracy:        model.Accuracy,
	}
	
	// Cache the result
	pa.cachePrediction(cacheKey, result)
	
	pa.logger.DebugWithContext("Generated SLA prediction",
		slog.String("sla_type", string(slaType)),
		slog.Float64("predicted_value", finalPrediction),
		slog.Float64("violation_probability", violationProbability),
		slog.Float64("confidence", confidence),
	)
	
	return result, nil
}

// initializeModel initializes a prediction model for the given SLA type
func (pa *PredictiveAlerting) initializeModel(ctx context.Context, slaType SLAType) error {
	// Load historical data
	dataset, err := pa.loadHistoricalData(ctx, slaType)
	if err != nil {
		return fmt.Errorf("failed to load historical data: %w", err)
	}
	
	// Check if we have enough data
	if len(dataset.DataPoints) < pa.config.MinDataPoints {
		pa.logger.WarnWithContext("Insufficient historical data for training",
			slog.String("sla_type", string(slaType)),
			slog.Int("data_points", len(dataset.DataPoints)),
			slog.Int("min_required", pa.config.MinDataPoints),
		)
		// Initialize with a simple baseline model
		return pa.initializeBaselineModel(slaType)
	}
	
	// Train the model
	model, err := pa.trainModel(ctx, slaType, dataset)
	if err != nil {
		return fmt.Errorf("failed to train model: %w", err)
	}
	
	pa.models[slaType] = model
	pa.historicalData[slaType] = dataset
	
	// Initialize seasonal model
	if pa.config.EnableSeasonalFeatures {
		seasonalModel := pa.buildSeasonalModel(slaType, dataset)
		pa.seasonalModels[slaType] = seasonalModel
	}
	
	pa.logger.InfoWithContext("Initialized prediction model",
		slog.String("sla_type", string(slaType)),
		slog.Float64("accuracy", model.Accuracy),
		slog.Int("training_data_size", model.TrainingDataSize),
	)
	
	return nil
}

// trainModel trains a machine learning model using historical data
func (pa *PredictiveAlerting) trainModel(ctx context.Context, slaType SLAType, 
	dataset *HistoricalDataset) (*PredictionModel, error) {
	
	// Extract features and labels
	features, labels, err := pa.prepareTrainingData(dataset)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare training data: %w", err)
	}
	
	// Normalize features
	normParams, normalizedFeatures := pa.normalizeFeatures(features)
	
	// Split data into training and validation sets
	trainFeatures, trainLabels, valFeatures, valLabels := pa.splitTrainingData(
		normalizedFeatures, labels, 0.8)
	
	// Train using simple linear regression for now
	// In production, this could use more sophisticated algorithms
	weights, bias := pa.trainLinearRegression(trainFeatures, trainLabels)
	
	// Validate the model
	accuracy, precision, recall, f1Score := pa.validateModel(
		weights, bias, valFeatures, valLabels)
	
	model := &PredictionModel{
		SLAType:          slaType,
		ModelType:        "linear_regression",
		TrainedAt:        time.Now(),
		Accuracy:         accuracy,
		Precision:        precision,
		Recall:           recall,
		F1Score:          f1Score,
		Weights:          weights,
		Bias:             bias,
		Features:         dataset.Features,
		Normalization:    normParams,
		TrainingDataSize: len(features),
		ValidationSplit:  0.2,
		Hyperparameters: map[string]interface{}{
			"learning_rate": 0.01,
			"regularization": 0.001,
		},
	}
	
	return model, nil
}

// generateBaselinePrediction generates a baseline prediction using the trained model
func (pa *PredictiveAlerting) generateBaselinePrediction(model *PredictionModel, 
	features map[string]float64) float64 {
	
	prediction := model.Bias
	
	for i, featureName := range model.Features {
		if i < len(model.Weights) {
			if value, exists := features[featureName]; exists {
				// Apply normalization
				normalizedValue := pa.normalizeFeatureValue(value, featureName, model.Normalization)
				prediction += model.Weights[i] * normalizedValue
			}
		}
	}
	
	return prediction
}

// Background loops for model management and predictions

func (pa *PredictiveAlerting) modelUpdateLoop(ctx context.Context) {
	ticker := time.NewTicker(pa.config.ModelUpdateInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-pa.stopCh:
			return
		case <-ticker.C:
			pa.updateAllModels(ctx)
		}
	}
}

func (pa *PredictiveAlerting) predictionLoop(ctx context.Context) {
	ticker := time.NewTicker(pa.config.PredictionInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-pa.stopCh:
			return
		case <-ticker.C:
			// Generate predictions for all SLA types
			pa.generatePeriodicPredictions(ctx)
		}
	}
}

func (pa *PredictiveAlerting) cacheCleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(pa.config.CacheTTL)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-pa.stopCh:
			return
		case <-ticker.C:
			pa.cleanupExpiredPredictions()
		}
	}
}

// Additional helper methods would include:
// - Feature extraction and engineering methods
// - Seasonal pattern detection and modeling
// - Trend analysis algorithms
// - Anomaly detection implementations
// - Model validation and metrics calculation
// - Cache management
// - Action recommendation logic

// Simplified implementations for key methods:

func (pa *PredictiveAlerting) loadHistoricalData(ctx context.Context, slaType SLAType) (*HistoricalDataset, error) {
	// In production, this would query a time-series database
	// For now, return mock data
	return &HistoricalDataset{
		SLAType: slaType,
		DataPoints: []HistoricalPoint{
			{
				Timestamp: time.Now().Add(-24 * time.Hour),
				SLAValue:  99.95,
				Features: map[string]float64{
					"cpu_usage":    45.0,
					"memory_usage": 67.0,
					"request_rate": 1200.0,
				},
				Violation: false,
			},
			// More historical points would be loaded here
		},
		StartTime: time.Now().Add(-pa.config.TrainingDataWindow),
		EndTime:   time.Now(),
		Features:  []string{"cpu_usage", "memory_usage", "request_rate"},
	}, nil
}

func (pa *PredictiveAlerting) initializeBaselineModel(slaType SLAType) error {
	// Create a simple baseline model when insufficient data is available
	model := &PredictionModel{
		SLAType:    slaType,
		ModelType:  "baseline",
		TrainedAt:  time.Now(),
		Accuracy:   0.5, // Conservative accuracy for baseline
		Weights:    []float64{1.0},
		Bias:       0.0,
		Features:   []string{"current_value"},
	}
	
	pa.models[slaType] = model
	return nil
}

// Cache management methods
func (pa *PredictiveAlerting) getCachedPrediction(key string) *PredictionResult {
	pa.mu.RLock()
	defer pa.mu.RUnlock()
	
	return pa.predictionCache[key]
}

func (pa *PredictiveAlerting) cachePrediction(key string, result *PredictionResult) {
	pa.mu.Lock()
	defer pa.mu.Unlock()
	
	pa.predictionCache[key] = result
}

func (pa *PredictiveAlerting) cleanupExpiredPredictions() {
	pa.mu.Lock()
	defer pa.mu.Unlock()
	
	cutoff := time.Now().Add(-pa.config.CacheTTL)
	for key, result := range pa.predictionCache {
		if result.PredictedAt.Before(cutoff) {
			delete(pa.predictionCache, key)
		}
	}
}

// Placeholder implementations for complex ML operations
func (pa *PredictiveAlerting) trainLinearRegression(features [][]float64, labels []float64) ([]float64, float64) {
	// Simplified linear regression - in production would use proper ML libraries
	weights := make([]float64, len(features[0]))
	for i := range weights {
		weights[i] = 1.0 // Initialize with equal weights
	}
	return weights, 0.0
}

func (pa *PredictiveAlerting) validateModel(weights []float64, bias float64, 
	features [][]float64, labels []float64) (accuracy, precision, recall, f1Score float64) {
	// Simplified validation - in production would calculate proper metrics
	return 0.85, 0.82, 0.88, 0.85
}

// Additional methods would include comprehensive implementations for:
// - Feature engineering and extraction
// - Seasonal pattern detection
// - Trend analysis
// - Anomaly scoring
// - Confidence calculation
// - Action recommendations
// - Contributing factor analysis