package health

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/thc1006/nephoran-intent-operator/pkg/health"
)

// HealthPredictor provides predictive health monitoring and early warning capabilities
type HealthPredictor struct {
	// Core configuration
	logger           *slog.Logger
	serviceName      string
	
	// Data sources
	aggregator       *HealthAggregator
	dependencyTracker *DependencyHealthTracker
	
	// ML models and algorithms
	models           map[string]*PredictionModel
	modelsMu         sync.RWMutex
	
	// Early warning system
	earlyWarning     *EarlyWarningSystem
	
	// Resource exhaustion detection
	resourceMonitor  *ResourceExhaustionMonitor
	
	// Seasonal pattern detection
	seasonalDetector *SeasonalPatternDetector
	
	// Prediction configuration
	config           *PredictorConfig
	
	// Historical data for ML training
	trainingData     map[string]*ModelTrainingData
	trainingMu       sync.RWMutex
	
	// Metrics
	predictorMetrics *PredictorMetrics
}

// PredictorConfig holds configuration for health prediction
type PredictorConfig struct {
	// Prediction horizons
	DefaultHorizon       time.Duration       `json:"default_horizon"`
	MaxHorizon           time.Duration       `json:"max_horizon"`
	PredictionInterval   time.Duration       `json:"prediction_interval"`
	
	// Model configuration
	ModelUpdateInterval  time.Duration       `json:"model_update_interval"`
	MinTrainingDataSize  int                 `json:"min_training_data_size"`
	ModelAccuracyThreshold float64           `json:"model_accuracy_threshold"`
	
	// Early warning thresholds
	EarlyWarningEnabled  bool                `json:"early_warning_enabled"`
	WarningThreshold     float64             `json:"warning_threshold"`
	CriticalThreshold    float64             `json:"critical_threshold"`
	
	// Resource exhaustion detection
	ResourceMonitoringEnabled bool           `json:"resource_monitoring_enabled"`
	ResourceThresholds   map[string]float64  `json:"resource_thresholds"`
	
	// Seasonal detection
	SeasonalDetectionEnabled bool            `json:"seasonal_detection_enabled"`
	SeasonalMinPeriod    time.Duration       `json:"seasonal_min_period"`
	SeasonalMaxPeriod    time.Duration       `json:"seasonal_max_period"`
	
	// Confidence thresholds
	MinPredictionConfidence float64          `json:"min_prediction_confidence"`
	HighConfidenceThreshold float64          `json:"high_confidence_threshold"`
}

// PredictionModel represents a machine learning model for health prediction
type PredictionModel struct {
	ID                string              `json:"id"`
	Component         string              `json:"component"`
	Algorithm         ModelAlgorithm      `json:"algorithm"`
	Features          []string            `json:"features"`
	Accuracy          float64             `json:"accuracy"`
	LastTrained       time.Time           `json:"last_trained"`
	TrainingDataSize  int                 `json:"training_data_size"`
	Hyperparameters   map[string]interface{} `json:"hyperparameters"`
	
	// Model state
	weights           []float64           `json:"-"`
	featureScalers    map[string]*FeatureScaler `json:"-"`
	
	// Performance metrics
	mae               float64             `json:"mae"`  // Mean Absolute Error
	rmse              float64             `json:"rmse"` // Root Mean Square Error
	r2Score           float64             `json:"r2_score"` // R-squared score
}

// ModelAlgorithm represents different ML algorithms for prediction
type ModelAlgorithm string

const (
	AlgorithmLinearRegression    ModelAlgorithm = "linear_regression"
	AlgorithmPolynomialRegression ModelAlgorithm = "polynomial_regression"
	AlgorithmMovingAverage       ModelAlgorithm = "moving_average"
	AlgorithmExponentialSmoothing ModelAlgorithm = "exponential_smoothing"
	AlgorithmARIMA               ModelAlgorithm = "arima"
	AlgorithmNeuralNetwork       ModelAlgorithm = "neural_network"
)

// FeatureScaler handles feature normalization
type FeatureScaler struct {
	Mean     float64 `json:"mean"`
	StdDev   float64 `json:"std_dev"`
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
	Method   ScalingMethod `json:"method"`
}

// ScalingMethod defines feature scaling methods
type ScalingMethod string

const (
	ScalingStandardization ScalingMethod = "standardization"
	ScalingMinMaxScaling  ScalingMethod = "minmax_scaling"
	ScalingRobustScaling  ScalingMethod = "robust_scaling"
)

// ModelTrainingData holds training data for ML models
type ModelTrainingData struct {
	Component     string                `json:"component"`
	Features      []FeatureVector       `json:"features"`
	Targets       []float64             `json:"targets"`
	Timestamps    []time.Time           `json:"timestamps"`
	LastUpdated   time.Time             `json:"last_updated"`
	DataQuality   DataQualityMetrics    `json:"data_quality"`
}

// FeatureVector represents a feature vector for ML training
type FeatureVector struct {
	Values    []float64           `json:"values"`
	Names     []string            `json:"names"`
	Timestamp time.Time           `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// DataQualityMetrics tracks the quality of training data
type DataQualityMetrics struct {
	Completeness    float64 `json:"completeness"`    // Percentage of non-null values
	Consistency     float64 `json:"consistency"`     // Data consistency score
	Accuracy        float64 `json:"accuracy"`        // Estimated data accuracy
	Timeliness      float64 `json:"timeliness"`      // Data freshness score
	Outliers        int     `json:"outliers"`        // Number of outliers detected
	MissingValues   int     `json:"missing_values"`  // Number of missing values
}

// EarlyWarningSystem provides early warning capabilities
type EarlyWarningSystem struct {
	logger              *slog.Logger
	config              *EarlyWarningConfig
	activeWarnings      map[string]*HealthWarning
	warningMu           sync.RWMutex
	alertChannel        chan *HealthWarning
	
	// Threshold monitoring
	thresholdMonitors   map[string]*ThresholdMonitor
	
	// Pattern detection
	anomalyDetector     *AnomalyDetector
}

// EarlyWarningConfig configures the early warning system
type EarlyWarningConfig struct {
	Enabled                bool                `json:"enabled"`
	WarningThreshold       float64             `json:"warning_threshold"`
	CriticalThreshold      float64             `json:"critical_threshold"`
	PredictionHorizon      time.Duration       `json:"prediction_horizon"`
	MinConfidence          float64             `json:"min_confidence"`
	SuppressionDuration    time.Duration       `json:"suppression_duration"`
	EscalationRules        []EscalationRule    `json:"escalation_rules"`
}

// HealthWarning represents a health warning or alert
type HealthWarning struct {
	ID                string              `json:"id"`
	Component         string              `json:"component"`
	Type              WarningType         `json:"type"`
	Severity          WarningSeverity     `json:"severity"`
	Title             string              `json:"title"`
	Description       string              `json:"description"`
	PredictedTime     time.Time           `json:"predicted_time"`
	Confidence        float64             `json:"confidence"`
	CurrentHealth     *EnhancedCheck      `json:"current_health,omitempty"`
	PredictedHealth   *HealthPrediction   `json:"predicted_health,omitempty"`
	RootCause         *RootCauseAnalysis  `json:"root_cause,omitempty"`
	Recommendations   []RecommendedAction `json:"recommendations"`
	
	// Warning lifecycle
	CreatedAt         time.Time           `json:"created_at"`
	LastUpdated       time.Time           `json:"last_updated"`
	Status            WarningStatus       `json:"status"`
	EscalationLevel   int                 `json:"escalation_level"`
	
	// Metadata
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// WarningType defines the type of warning
type WarningType string

const (
	WarningTypeHealthDegradation    WarningType = "health_degradation"
	WarningTypeResourceExhaustion   WarningType = "resource_exhaustion"
	WarningTypeDependencyFailure    WarningType = "dependency_failure"
	WarningTypePerformanceDegradation WarningType = "performance_degradation"
	WarningTypeAnomalyDetected      WarningType = "anomaly_detected"
	WarningTypeSeasonalAnomaly      WarningType = "seasonal_anomaly"
)

// WarningSeverity defines warning severity levels
type WarningSeverity string

const (
	SeverityLow      WarningSeverity = "low"
	SeverityMedium   WarningSeverity = "medium"
	SeverityHigh     WarningSeverity = "high"
	SeverityCritical WarningSeverity = "critical"
)

// WarningStatus defines the status of a warning
type WarningStatus string

const (
	WarningStatusActive     WarningStatus = "active"
	WarningStatusResolved   WarningStatus = "resolved"
	WarningStatusSuppressed WarningStatus = "suppressed"
	WarningStatusEscalated  WarningStatus = "escalated"
)

// EscalationRule defines when and how warnings should be escalated
type EscalationRule struct {
	Level           int           `json:"level"`
	Duration        time.Duration `json:"duration"`
	SeverityFilter  []WarningSeverity `json:"severity_filter"`
	Actions         []string      `json:"actions"`
	NotificationChannels []string `json:"notification_channels"`
}

// RootCauseAnalysis provides analysis of potential root causes
type RootCauseAnalysis struct {
	PrimaryCandidate    RootCauseCandidate   `json:"primary_candidate"`
	AlternativeCandidates []RootCauseCandidate `json:"alternative_candidates"`
	CorrelatedFactors   []CorrelationFactor  `json:"correlated_factors"`
	Confidence          float64              `json:"confidence"`
	AnalysisMethod      string               `json:"analysis_method"`
}

// RootCauseCandidate represents a potential root cause
type RootCauseCandidate struct {
	Category        string   `json:"category"`
	Description     string   `json:"description"`
	Probability     float64  `json:"probability"`
	Evidence        []string `json:"evidence"`
	Component       string   `json:"component,omitempty"`
	Dependency      string   `json:"dependency,omitempty"`
}

// CorrelationFactor represents correlated factors in root cause analysis
type CorrelationFactor struct {
	Factor          string  `json:"factor"`
	Correlation     float64 `json:"correlation"`
	TimeOffset      time.Duration `json:"time_offset"`
	Description     string  `json:"description"`
}

// ThresholdMonitor monitors specific thresholds for a component
type ThresholdMonitor struct {
	Component       string    `json:"component"`
	Metric          string    `json:"metric"`
	WarningThreshold float64  `json:"warning_threshold"`
	CriticalThreshold float64 `json:"critical_threshold"`
	LastValue       float64   `json:"last_value"`
	LastChecked     time.Time `json:"last_checked"`
	ConsecutiveViolations int `json:"consecutive_violations"`
}

// AnomalyDetector detects anomalies in health patterns
type AnomalyDetector struct {
	logger              *slog.Logger
	algorithms          []AnomalyAlgorithm
	sensitivityLevel    AnomalySensitivity
	historicalBaseline  map[string]*BaselineModel
}

// AnomalyAlgorithm defines different anomaly detection algorithms
type AnomalyAlgorithm string

const (
	AnomalyStatisticalOutlier    AnomalyAlgorithm = "statistical_outlier"
	AnomalyIsolationForest      AnomalyAlgorithm = "isolation_forest"
	AnomalyOneClassSVM          AnomalyAlgorithm = "one_class_svm"
	AnomalyLocalOutlierFactor   AnomalyAlgorithm = "local_outlier_factor"
	AnomalySeasonalESD          AnomalyAlgorithm = "seasonal_esd"
)

// AnomalySensitivity defines sensitivity levels for anomaly detection
type AnomalySensitivity string

const (
	SensitivityLow    AnomalySensitivity = "low"
	SensitivityMedium AnomalySensitivity = "medium"
	SensitivityHigh   AnomalySensitivity = "high"
)

// BaselineModel represents a baseline model for anomaly detection
type BaselineModel struct {
	Component       string    `json:"component"`
	Mean            float64   `json:"mean"`
	StdDev          float64   `json:"std_dev"`
	Percentiles     map[int]float64 `json:"percentiles"`
	LastUpdated     time.Time `json:"last_updated"`
	DataPoints      int       `json:"data_points"`
}

// ResourceExhaustionMonitor monitors resource exhaustion patterns
type ResourceExhaustionMonitor struct {
	logger           *slog.Logger
	resourceTrackers map[string]*ResourceTracker
	predictions      map[string]*ResourceExhaustionPrediction
	mu               sync.RWMutex
}

// ResourceTracker tracks resource usage patterns
type ResourceTracker struct {
	ResourceType    ResourceType      `json:"resource_type"`
	Component       string            `json:"component"`
	CurrentUsage    float64           `json:"current_usage"`
	MaxCapacity     float64           `json:"max_capacity"`
	UtilizationRate float64           `json:"utilization_rate"`
	GrowthRate      float64           `json:"growth_rate"`
	History         []ResourceDataPoint `json:"history"`
	LastUpdated     time.Time         `json:"last_updated"`
}

// ResourceType defines different types of resources to monitor
type ResourceType string

const (
	ResourceMemory      ResourceType = "memory"
	ResourceCPU         ResourceType = "cpu"
	ResourceDisk        ResourceType = "disk"
	ResourceNetwork     ResourceType = "network"
	ResourceConnections ResourceType = "connections"
	ResourceFileHandles ResourceType = "file_handles"
)

// ResourceDataPoint represents a resource usage data point
type ResourceDataPoint struct {
	Timestamp   time.Time `json:"timestamp"`
	Usage       float64   `json:"usage"`
	Capacity    float64   `json:"capacity"`
	Utilization float64   `json:"utilization"`
}

// ResourceExhaustionPrediction predicts when a resource will be exhausted
type ResourceExhaustionPrediction struct {
	ResourceType        ResourceType  `json:"resource_type"`
	Component           string        `json:"component"`
	PredictedExhaustion time.Time     `json:"predicted_exhaustion"`
	Confidence          float64       `json:"confidence"`
	CurrentTrend        string        `json:"current_trend"`
	RecommendedActions  []RecommendedAction `json:"recommended_actions"`
	TimeToExhaustion    time.Duration `json:"time_to_exhaustion"`
}

// SeasonalPatternDetector detects seasonal patterns in health data
type SeasonalPatternDetector struct {
	logger                  *slog.Logger
	detectedPatterns        map[string][]SeasonalPattern
	patternsMu              sync.RWMutex
	minPeriod               time.Duration
	maxPeriod               time.Duration
	minConfidence           float64
}

// PredictorMetrics contains Prometheus metrics for the health predictor
type PredictorMetrics struct {
	PredictionAccuracy      *prometheus.GaugeVec
	PredictionLatency       prometheus.Histogram
	ModelTrainingTime       prometheus.Histogram
	EarlyWarningsGenerated  *prometheus.CounterVec
	AnomaliesDetected       *prometheus.CounterVec
	ResourceExhaustionWarnings *prometheus.CounterVec
	ModelPerformanceMetrics *prometheus.GaugeVec
}

// HealthPredictionResult represents the result of health prediction
type HealthPredictionResult struct {
	Component           string                      `json:"component"`
	Timestamp           time.Time                   `json:"timestamp"`
	PredictionHorizon   time.Duration               `json:"prediction_horizon"`
	
	// Predictions
	Predictions         []HealthPrediction          `json:"predictions"`
	
	// Early warnings
	Warnings            []HealthWarning             `json:"warnings,omitempty"`
	
	// Resource exhaustion predictions
	ResourcePredictions []ResourceExhaustionPrediction `json:"resource_predictions,omitempty"`
	
	// Anomalies detected
	Anomalies           []DetectedAnomaly           `json:"anomalies,omitempty"`
	
	// Seasonal patterns
	SeasonalPatterns    []SeasonalPattern           `json:"seasonal_patterns,omitempty"`
	
	// Model information
	ModelUsed           *PredictionModel            `json:"model_used,omitempty"`
	ModelAccuracy       float64                     `json:"model_accuracy"`
	
	// Confidence and quality
	OverallConfidence   float64                     `json:"overall_confidence"`
	DataQuality         DataQualityMetrics          `json:"data_quality"`
}

// DetectedAnomaly represents an anomaly in health data
type DetectedAnomaly struct {
	Component       string              `json:"component"`
	Timestamp       time.Time           `json:"timestamp"`
	Value           float64             `json:"value"`
	ExpectedValue   float64             `json:"expected_value"`
	AnomalyScore    float64             `json:"anomaly_score"`
	Algorithm       AnomalyAlgorithm    `json:"algorithm"`
	Severity        WarningSeverity     `json:"severity"`
	Description     string              `json:"description"`
	Context         map[string]interface{} `json:"context,omitempty"`
}

// NewHealthPredictor creates a new health predictor
func NewHealthPredictor(serviceName string, aggregator *HealthAggregator, dependencyTracker *DependencyHealthTracker, logger *slog.Logger) *HealthPredictor {
	if logger == nil {
		logger = slog.Default()
	}
	
	predictor := &HealthPredictor{
		logger:            logger.With("component", "health_predictor"),
		serviceName:       serviceName,
		aggregator:        aggregator,
		dependencyTracker: dependencyTracker,
		models:            make(map[string]*PredictionModel),
		trainingData:      make(map[string]*ModelTrainingData),
		config:            defaultPredictorConfig(),
		predictorMetrics:  initializePredictorMetrics(),
	}
	
	// Initialize early warning system
	predictor.earlyWarning = &EarlyWarningSystem{
		logger:            logger.With("component", "early_warning"),
		config:            defaultEarlyWarningConfig(),
		activeWarnings:    make(map[string]*HealthWarning),
		alertChannel:      make(chan *HealthWarning, 100),
		thresholdMonitors: make(map[string]*ThresholdMonitor),
		anomalyDetector: &AnomalyDetector{
			logger:              logger.With("component", "anomaly_detector"),
			algorithms:          []AnomalyAlgorithm{AnomalyStatisticalOutlier, AnomalySeasonalESD},
			sensitivityLevel:    SensitivityMedium,
			historicalBaseline:  make(map[string]*BaselineModel),
		},
	}
	
	// Initialize resource exhaustion monitor
	predictor.resourceMonitor = &ResourceExhaustionMonitor{
		logger:           logger.With("component", "resource_monitor"),
		resourceTrackers: make(map[string]*ResourceTracker),
		predictions:      make(map[string]*ResourceExhaustionPrediction),
	}
	
	// Initialize seasonal pattern detector
	predictor.seasonalDetector = &SeasonalPatternDetector{
		logger:           logger.With("component", "seasonal_detector"),
		detectedPatterns: make(map[string][]SeasonalPattern),
		minPeriod:        time.Hour,
		maxPeriod:        7 * 24 * time.Hour, // 1 week
		minConfidence:    0.7,
	}
	
	// Initialize default models
	predictor.initializeDefaultModels()
	
	return predictor
}

// defaultPredictorConfig returns default predictor configuration
func defaultPredictorConfig() *PredictorConfig {
	return &PredictorConfig{
		DefaultHorizon:               4 * time.Hour,
		MaxHorizon:                   24 * time.Hour,
		PredictionInterval:           15 * time.Minute,
		ModelUpdateInterval:          time.Hour,
		MinTrainingDataSize:          50,
		ModelAccuracyThreshold:       0.7,
		EarlyWarningEnabled:          true,
		WarningThreshold:             0.7,
		CriticalThreshold:            0.5,
		ResourceMonitoringEnabled:    true,
		ResourceThresholds: map[string]float64{
			"memory": 0.8,
			"cpu":    0.8,
			"disk":   0.9,
		},
		SeasonalDetectionEnabled:     true,
		SeasonalMinPeriod:           time.Hour,
		SeasonalMaxPeriod:           7 * 24 * time.Hour,
		MinPredictionConfidence:      0.6,
		HighConfidenceThreshold:      0.8,
	}
}

// defaultEarlyWarningConfig returns default early warning configuration
func defaultEarlyWarningConfig() *EarlyWarningConfig {
	return &EarlyWarningConfig{
		Enabled:             true,
		WarningThreshold:    0.7,
		CriticalThreshold:   0.5,
		PredictionHorizon:   2 * time.Hour,
		MinConfidence:       0.7,
		SuppressionDuration: 30 * time.Minute,
		EscalationRules: []EscalationRule{
			{
				Level:               1,
				Duration:            15 * time.Minute,
				SeverityFilter:      []WarningSeverity{SeverityHigh, SeverityCritical},
				Actions:             []string{"notify_on_call"},
				NotificationChannels: []string{"slack", "email"},
			},
			{
				Level:               2,
				Duration:            30 * time.Minute,
				SeverityFilter:      []WarningSeverity{SeverityCritical},
				Actions:             []string{"page_management", "create_incident"},
				NotificationChannels: []string{"pagerduty", "phone"},
			},
		},
	}
}

// initializePredictorMetrics initializes Prometheus metrics
func initializePredictorMetrics() *PredictorMetrics {
	return &PredictorMetrics{
		PredictionAccuracy: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "health_prediction_accuracy",
			Help: "Accuracy of health predictions by model and component",
		}, []string{"component", "model", "metric"}),
		
		PredictionLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "health_prediction_latency_seconds",
			Help:    "Latency of health prediction operations",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 8),
		}),
		
		ModelTrainingTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "model_training_time_seconds",
			Help:    "Time taken to train prediction models",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 10),
		}),
		
		EarlyWarningsGenerated: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "early_warnings_generated_total",
			Help: "Total number of early warnings generated",
		}, []string{"component", "warning_type", "severity"}),
		
		AnomaliesDetected: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "anomalies_detected_total",
			Help: "Total number of anomalies detected",
		}, []string{"component", "algorithm", "severity"}),
		
		ResourceExhaustionWarnings: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "resource_exhaustion_warnings_total",
			Help: "Total number of resource exhaustion warnings",
		}, []string{"component", "resource_type"}),
		
		ModelPerformanceMetrics: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "model_performance_metrics",
			Help: "Performance metrics for prediction models",
		}, []string{"component", "model", "metric"}),
	}
}

// PredictHealth performs comprehensive health prediction
func (hp *HealthPredictor) PredictHealth(ctx context.Context, component string, horizon time.Duration) (*HealthPredictionResult, error) {
	start := time.Now()
	
	if horizon > hp.config.MaxHorizon {
		horizon = hp.config.MaxHorizon
	}
	if horizon <= 0 {
		horizon = hp.config.DefaultHorizon
	}
	
	result := &HealthPredictionResult{
		Component:         component,
		Timestamp:         start,
		PredictionHorizon: horizon,
		Predictions:       []HealthPrediction{},
		Warnings:          []HealthWarning{},
	}
	
	// Get or create model for component
	model, err := hp.getOrCreateModel(component)
	if err != nil {
		hp.logger.Error("Failed to get prediction model", "error", err, "component", component)
		return result, fmt.Errorf("failed to get prediction model: %w", err)
	}
	
	result.ModelUsed = model
	result.ModelAccuracy = model.Accuracy
	
	// Get training data and update if needed
	trainingData, err := hp.getTrainingData(component)
	if err != nil {
		hp.logger.Error("Failed to get training data", "error", err, "component", component)
		return result, fmt.Errorf("failed to get training data: %w", err)
	}
	
	result.DataQuality = trainingData.DataQuality
	
	// Generate predictions using the model
	predictions, confidence, err := hp.generatePredictions(model, trainingData, horizon)
	if err != nil {
		hp.logger.Error("Failed to generate predictions", "error", err, "component", component)
		return result, fmt.Errorf("failed to generate predictions: %w", err)
	}
	
	result.Predictions = predictions
	result.OverallConfidence = confidence
	
	// Check for early warnings if enabled
	if hp.config.EarlyWarningEnabled {
		warnings := hp.checkEarlyWarnings(component, predictions, confidence)
		result.Warnings = warnings
	}
	
	// Check for resource exhaustion if enabled
	if hp.config.ResourceMonitoringEnabled {
		resourcePredictions := hp.checkResourceExhaustion(component)
		result.ResourcePredictions = resourcePredictions
	}
	
	// Detect anomalies
	anomalies := hp.detectAnomalies(component, trainingData)
	result.Anomalies = anomalies
	
	// Detect seasonal patterns if enabled
	if hp.config.SeasonalDetectionEnabled {
		patterns := hp.detectSeasonalPatterns(component, trainingData)
		result.SeasonalPatterns = patterns
	}
	
	// Record metrics
	duration := time.Since(start)
	hp.predictorMetrics.PredictionLatency.Observe(duration.Seconds())
	hp.predictorMetrics.PredictionAccuracy.WithLabelValues(component, string(model.Algorithm), "overall").Set(confidence)
	
	// Record warnings and anomalies
	for _, warning := range result.Warnings {
		hp.predictorMetrics.EarlyWarningsGenerated.WithLabelValues(component, string(warning.Type), string(warning.Severity)).Inc()
	}
	
	for _, anomaly := range result.Anomalies {
		hp.predictorMetrics.AnomaliesDetected.WithLabelValues(component, string(anomaly.Algorithm), string(anomaly.Severity)).Inc()
	}
	
	hp.logger.Debug("Health prediction completed",
		"component", component,
		"duration", duration,
		"predictions", len(result.Predictions),
		"confidence", confidence,
		"warnings", len(result.Warnings))
	
	return result, nil
}

// getOrCreateModel gets an existing model or creates a new one for a component
func (hp *HealthPredictor) getOrCreateModel(component string) (*PredictionModel, error) {
	hp.modelsMu.RLock()
	if model, exists := hp.models[component]; exists {
		hp.modelsMu.RUnlock()
		
		// Check if model needs retraining
		if time.Since(model.LastTrained) > hp.config.ModelUpdateInterval {
			return hp.retrainModel(component, model)
		}
		
		return model, nil
	}
	hp.modelsMu.RUnlock()
	
	// Create new model
	return hp.createModel(component)
}

// createModel creates a new prediction model for a component
func (hp *HealthPredictor) createModel(component string) (*PredictionModel, error) {
	model := &PredictionModel{
		ID:              fmt.Sprintf("%s-model-%d", component, time.Now().Unix()),
		Component:       component,
		Algorithm:       AlgorithmLinearRegression, // Default algorithm
		Features:        []string{"score", "latency", "error_rate", "timestamp"},
		Accuracy:        0.0,
		LastTrained:     time.Time{},
		TrainingDataSize: 0,
		Hyperparameters: map[string]interface{}{
			"learning_rate": 0.01,
			"regularization": 0.001,
		},
		weights:        []float64{},
		featureScalers: make(map[string]*FeatureScaler),
	}
	
	hp.modelsMu.Lock()
	hp.models[component] = model
	hp.modelsMu.Unlock()
	
	hp.logger.Info("Created new prediction model",
		"component", component,
		"algorithm", model.Algorithm,
		"features", model.Features)
	
	return model, nil
}

// retrainModel retrains an existing model with fresh data
func (hp *HealthPredictor) retrainModel(component string, model *PredictionModel) (*PredictionModel, error) {
	start := time.Now()
	
	// Get fresh training data
	trainingData, err := hp.getTrainingData(component)
	if err != nil {
		return model, fmt.Errorf("failed to get training data: %w", err)
	}
	
	if len(trainingData.Features) < hp.config.MinTrainingDataSize {
		hp.logger.Warn("Insufficient training data for model retraining",
			"component", component,
			"data_size", len(trainingData.Features),
			"min_required", hp.config.MinTrainingDataSize)
		return model, nil
	}
	
	// Train the model based on algorithm
	err = hp.trainModel(model, trainingData)
	if err != nil {
		return model, fmt.Errorf("failed to train model: %w", err)
	}
	
	// Update model metadata
	model.LastTrained = time.Now()
	model.TrainingDataSize = len(trainingData.Features)
	
	// Record training time
	trainingDuration := time.Since(start)
	hp.predictorMetrics.ModelTrainingTime.Observe(trainingDuration.Seconds())
	
	// Update model in registry
	hp.modelsMu.Lock()
	hp.models[component] = model
	hp.modelsMu.Unlock()
	
	hp.logger.Info("Model retrained successfully",
		"component", component,
		"algorithm", model.Algorithm,
		"accuracy", model.Accuracy,
		"training_duration", trainingDuration,
		"data_size", model.TrainingDataSize)
	
	return model, nil
}

// trainModel trains a model using the specified algorithm
func (hp *HealthPredictor) trainModel(model *PredictionModel, trainingData *ModelTrainingData) error {
	switch model.Algorithm {
	case AlgorithmLinearRegression:
		return hp.trainLinearRegression(model, trainingData)
	case AlgorithmMovingAverage:
		return hp.trainMovingAverage(model, trainingData)
	case AlgorithmExponentialSmoothing:
		return hp.trainExponentialSmoothing(model, trainingData)
	default:
		return fmt.Errorf("unsupported algorithm: %s", model.Algorithm)
	}
}

// trainLinearRegression trains a linear regression model
func (hp *HealthPredictor) trainLinearRegression(model *PredictionModel, trainingData *ModelTrainingData) error {
	if len(trainingData.Features) == 0 {
		return fmt.Errorf("no training data available")
	}
	
	// Prepare feature matrix and target vector
	numFeatures := len(trainingData.Features[0].Values)
	numSamples := len(trainingData.Features)
	
	// Initialize weights (simple approach)
	model.weights = make([]float64, numFeatures+1) // +1 for bias
	
	// Simple linear regression using normal equations (simplified)
	// In production, use proper ML libraries like golearn or TensorFlow
	
	// Calculate means
	var targetMean float64
	featureMeans := make([]float64, numFeatures)
	
	for i, target := range trainingData.Targets {
		targetMean += target
		for j, feature := range trainingData.Features[i].Values {
			featureMeans[j] += feature
		}
	}
	
	targetMean /= float64(numSamples)
	for j := range featureMeans {
		featureMeans[j] /= float64(numSamples)
	}
	
	// Simple weight calculation (this is a simplified approach)
	for j := 0; j < numFeatures; j++ {
		var numerator, denominator float64
		
		for i := range trainingData.Features {
			featureDiff := trainingData.Features[i].Values[j] - featureMeans[j]
			targetDiff := trainingData.Targets[i] - targetMean
			
			numerator += featureDiff * targetDiff
			denominator += featureDiff * featureDiff
		}
		
		if denominator > 0 {
			model.weights[j] = numerator / denominator
		}
	}
	
	// Calculate bias
	var biasSum float64
	for i := range trainingData.Features {
		prediction := 0.0
		for j, feature := range trainingData.Features[i].Values {
			prediction += model.weights[j] * feature
		}
		biasSum += trainingData.Targets[i] - prediction
	}
	model.weights[numFeatures] = biasSum / float64(numSamples) // bias term
	
	// Calculate accuracy metrics
	model.Accuracy, model.mae, model.rmse, model.r2Score = hp.calculateModelAccuracy(model, trainingData)
	
	return nil
}

// trainMovingAverage trains a moving average model
func (hp *HealthPredictor) trainMovingAverage(model *PredictionModel, trainingData *ModelTrainingData) error {
	windowSize := 10 // Default window size
	if ws, exists := model.Hyperparameters["window_size"].(int); exists {
		windowSize = ws
	}
	
	// Store window size in weights for prediction
	model.weights = []float64{float64(windowSize)}
	
	// Calculate accuracy using moving average predictions
	model.Accuracy, model.mae, model.rmse, model.r2Score = hp.calculateModelAccuracy(model, trainingData)
	
	return nil
}

// trainExponentialSmoothing trains an exponential smoothing model
func (hp *HealthPredictor) trainExponentialSmoothing(model *PredictionModel, trainingData *ModelTrainingData) error {
	alpha := 0.3 // Default smoothing parameter
	if a, exists := model.Hyperparameters["alpha"].(float64); exists {
		alpha = a
	}
	
	// Store alpha in weights for prediction
	model.weights = []float64{alpha}
	
	// Calculate accuracy
	model.Accuracy, model.mae, model.rmse, model.r2Score = hp.calculateModelAccuracy(model, trainingData)
	
	return nil
}

// calculateModelAccuracy calculates model accuracy metrics
func (hp *HealthPredictor) calculateModelAccuracy(model *PredictionModel, trainingData *ModelTrainingData) (float64, float64, float64, float64) {
	if len(trainingData.Features) == 0 {
		return 0, 0, 0, 0
	}
	
	var totalError, totalSquaredError, totalVariance float64
	targetMean := 0.0
	
	// Calculate target mean
	for _, target := range trainingData.Targets {
		targetMean += target
	}
	targetMean /= float64(len(trainingData.Targets))
	
	// Calculate predictions and errors
	for i, features := range trainingData.Features {
		actual := trainingData.Targets[i]
		predicted := hp.makePrediction(model, features.Values)
		
		error := math.Abs(actual - predicted)
		squaredError := math.Pow(actual-predicted, 2)
		variance := math.Pow(actual-targetMean, 2)
		
		totalError += error
		totalSquaredError += squaredError
		totalVariance += variance
	}
	
	n := float64(len(trainingData.Targets))
	mae := totalError / n
	rmse := math.Sqrt(totalSquaredError / n)
	
	// R-squared calculation
	var r2Score float64
	if totalVariance > 0 {
		r2Score = 1 - (totalSquaredError / totalVariance)
	}
	
	// Overall accuracy (simplified metric)
	accuracy := math.Max(0, r2Score)
	
	return accuracy, mae, rmse, r2Score
}

// makePrediction makes a prediction using the trained model
func (hp *HealthPredictor) makePrediction(model *PredictionModel, features []float64) float64 {
	switch model.Algorithm {
	case AlgorithmLinearRegression:
		return hp.predictLinearRegression(model, features)
	case AlgorithmMovingAverage:
		return hp.predictMovingAverage(model, features)
	case AlgorithmExponentialSmoothing:
		return hp.predictExponentialSmoothing(model, features)
	default:
		return 0.5 // Default prediction
	}
}

// predictLinearRegression makes prediction using linear regression
func (hp *HealthPredictor) predictLinearRegression(model *PredictionModel, features []float64) float64 {
	if len(model.weights) == 0 || len(features) == 0 {
		return 0.5
	}
	
	prediction := 0.0
	
	// Apply weights to features
	for i, feature := range features {
		if i < len(model.weights)-1 {
			prediction += model.weights[i] * feature
		}
	}
	
	// Add bias term
	if len(model.weights) > len(features) {
		prediction += model.weights[len(model.weights)-1]
	}
	
	// Clamp between 0 and 1
	if prediction < 0 {
		prediction = 0
	} else if prediction > 1 {
		prediction = 1
	}
	
	return prediction
}

// predictMovingAverage makes prediction using moving average
func (hp *HealthPredictor) predictMovingAverage(model *PredictionModel, features []float64) float64 {
	if len(features) == 0 {
		return 0.5
	}
	
	windowSize := 10
	if len(model.weights) > 0 {
		windowSize = int(model.weights[0])
	}
	
	// Use recent values for moving average
	startIdx := len(features) - windowSize
	if startIdx < 0 {
		startIdx = 0
	}
	
	sum := 0.0
	count := 0
	for i := startIdx; i < len(features); i++ {
		sum += features[i]
		count++
	}
	
	if count == 0 {
		return 0.5
	}
	
	return sum / float64(count)
}

// predictExponentialSmoothing makes prediction using exponential smoothing
func (hp *HealthPredictor) predictExponentialSmoothing(model *PredictionModel, features []float64) float64 {
	if len(features) == 0 {
		return 0.5
	}
	
	alpha := 0.3
	if len(model.weights) > 0 {
		alpha = model.weights[0]
	}
	
	// Start with first value
	smoothed := features[0]
	
	// Apply exponential smoothing
	for i := 1; i < len(features); i++ {
		smoothed = alpha*features[i] + (1-alpha)*smoothed
	}
	
	return smoothed
}

// generatePredictions generates future health predictions
func (hp *HealthPredictor) generatePredictions(model *PredictionModel, trainingData *ModelTrainingData, horizon time.Duration) ([]HealthPrediction, float64, error) {
	if len(trainingData.Features) == 0 {
		return nil, 0, fmt.Errorf("no training data available")
	}
	
	// Calculate number of prediction points
	interval := hp.config.PredictionInterval
	numPredictions := int(horizon / interval)
	if numPredictions == 0 {
		numPredictions = 1
	}
	
	predictions := make([]HealthPrediction, numPredictions)
	
	// Get the most recent features as baseline
	lastFeatures := trainingData.Features[len(trainingData.Features)-1]
	
	// Generate predictions
	for i := 0; i < numPredictions; i++ {
		futureTime := time.Now().Add(time.Duration(i+1) * interval)
		
		// Make prediction (simplified - in practice would use time series analysis)
		predictedScore := hp.makePrediction(model, lastFeatures.Values)
		
		// Add some uncertainty over time
		uncertainty := float64(i) * 0.05 // Increasing uncertainty
		confidence := math.Max(0.1, model.Accuracy-uncertainty)
		
		// Convert score to status
		var predictedStatus health.Status
		if predictedScore >= 0.9 {
			predictedStatus = health.StatusHealthy
		} else if predictedScore >= 0.6 {
			predictedStatus = health.StatusDegraded
		} else {
			predictedStatus = health.StatusUnhealthy
		}
		
		predictions[i] = HealthPrediction{
			PredictedTime:   futureTime,
			PredictedStatus: predictedStatus,
			PredictedScore:  predictedScore,
			Confidence:      confidence,
			Reasoning:       fmt.Sprintf("Predicted using %s model", model.Algorithm),
		}
	}
	
	// Calculate overall confidence
	overallConfidence := model.Accuracy
	if len(predictions) > 0 {
		confidenceSum := 0.0
		for _, pred := range predictions {
			confidenceSum += pred.Confidence
		}
		overallConfidence = confidenceSum / float64(len(predictions))
	}
	
	return predictions, overallConfidence, nil
}

// getTrainingData gets or creates training data for a component
func (hp *HealthPredictor) getTrainingData(component string) (*ModelTrainingData, error) {
	hp.trainingMu.RLock()
	if data, exists := hp.trainingData[component]; exists {
		hp.trainingMu.RUnlock()
		
		// Check if data needs update
		if time.Since(data.LastUpdated) < 5*time.Minute {
			return data, nil
		}
	} else {
		hp.trainingMu.RUnlock()
	}
	
	// Create or update training data
	return hp.updateTrainingData(component)
}

// updateTrainingData updates training data for a component
func (hp *HealthPredictor) updateTrainingData(component string) (*ModelTrainingData, error) {
	data := &ModelTrainingData{
		Component:   component,
		Features:    []FeatureVector{},
		Targets:     []float64{},
		Timestamps:  []time.Time{},
		LastUpdated: time.Now(),
		DataQuality: DataQualityMetrics{
			Completeness: 1.0,
			Consistency:  1.0,
			Accuracy:     1.0,
			Timeliness:   1.0,
		},
	}
	
	// Get historical data from aggregator
	if hp.aggregator != nil {
		history := hp.aggregator.GetCheckHistory(component, 100)
		
		for _, point := range history {
			featureVector := FeatureVector{
				Values: []float64{
					point.Score,
					point.Duration.Seconds(),
					float64(point.ConsecutiveFails),
					float64(point.Timestamp.Unix()),
				},
				Names:     []string{"score", "latency", "error_rate", "timestamp"},
				Timestamp: point.Timestamp,
			}
			
			data.Features = append(data.Features, featureVector)
			data.Targets = append(data.Targets, point.Score)
			data.Timestamps = append(data.Timestamps, point.Timestamp)
		}
	}
	
	// Calculate data quality metrics
	hp.calculateDataQuality(data)
	
	// Store training data
	hp.trainingMu.Lock()
	hp.trainingData[component] = data
	hp.trainingMu.Unlock()
	
	return data, nil
}

// calculateDataQuality calculates data quality metrics
func (hp *HealthPredictor) calculateDataQuality(data *ModelTrainingData) {
	if len(data.Features) == 0 {
		return
	}
	
	// Calculate completeness (percentage of non-null values)
	nonNullCount := 0
	totalValues := 0
	
	for _, feature := range data.Features {
		for _, value := range feature.Values {
			totalValues++
			if !math.IsNaN(value) && !math.IsInf(value, 0) {
				nonNullCount++
			}
		}
	}
	
	if totalValues > 0 {
		data.DataQuality.Completeness = float64(nonNullCount) / float64(totalValues)
	}
	
	// Calculate timeliness (how recent is the data)
	if len(data.Timestamps) > 0 {
		latest := data.Timestamps[len(data.Timestamps)-1]
		age := time.Since(latest)
		
		// Timeliness score decreases with age
		data.DataQuality.Timeliness = math.Max(0, 1.0-age.Hours()/24.0) // 1.0 for today, 0.0 for 24h+ old
	}
	
	// Other quality metrics would be calculated here in a full implementation
}

// checkEarlyWarnings checks for early warning conditions
func (hp *HealthPredictor) checkEarlyWarnings(component string, predictions []HealthPrediction, confidence float64) []HealthWarning {
	var warnings []HealthWarning
	
	if !hp.config.EarlyWarningEnabled || confidence < hp.config.MinPredictionConfidence {
		return warnings
	}
	
	// Check each prediction for warning conditions
	for _, prediction := range predictions {
		if prediction.PredictedScore < hp.config.CriticalThreshold {
			warning := HealthWarning{
				ID:            fmt.Sprintf("warning-%s-%d", component, time.Now().Unix()),
				Component:     component,
				Type:          WarningTypeHealthDegradation,
				Severity:      SeverityCritical,
				Title:         fmt.Sprintf("Critical health degradation predicted for %s", component),
				Description:   fmt.Sprintf("Health score predicted to drop to %.2f at %s", prediction.PredictedScore, prediction.PredictedTime.Format(time.RFC3339)),
				PredictedTime: prediction.PredictedTime,
				Confidence:    prediction.Confidence,
				PredictedHealth: &prediction,
				CreatedAt:     time.Now(),
				Status:        WarningStatusActive,
				Recommendations: []RecommendedAction{
					{
						Action:      "investigate_degradation",
						Priority:    PriorityImmediate,
						Description: "Investigate potential causes of health degradation",
						Automated:   false,
						ETA:         15 * time.Minute,
					},
				},
			}
			warnings = append(warnings, warning)
		} else if prediction.PredictedScore < hp.config.WarningThreshold {
			warning := HealthWarning{
				ID:            fmt.Sprintf("warning-%s-%d", component, time.Now().Unix()),
				Component:     component,
				Type:          WarningTypeHealthDegradation,
				Severity:      SeverityHigh,
				Title:         fmt.Sprintf("Health degradation predicted for %s", component),
				Description:   fmt.Sprintf("Health score predicted to drop to %.2f at %s", prediction.PredictedScore, prediction.PredictedTime.Format(time.RFC3339)),
				PredictedTime: prediction.PredictedTime,
				Confidence:    prediction.Confidence,
				PredictedHealth: &prediction,
				CreatedAt:     time.Now(),
				Status:        WarningStatusActive,
				Recommendations: []RecommendedAction{
					{
						Action:      "monitor_closely",
						Priority:    PriorityUrgent,
						Description: "Monitor component health closely",
						Automated:   true,
						ETA:         5 * time.Minute,
					},
				},
			}
			warnings = append(warnings, warning)
		}
	}
	
	return warnings
}

// checkResourceExhaustion checks for resource exhaustion predictions
func (hp *HealthPredictor) checkResourceExhaustion(component string) []ResourceExhaustionPrediction {
	var predictions []ResourceExhaustionPrediction
	
	hp.resourceMonitor.mu.RLock()
	defer hp.resourceMonitor.mu.RUnlock()
	
	// Check each resource tracker for the component
	for trackerKey, tracker := range hp.resourceMonitor.resourceTrackers {
		if tracker.Component != component {
			continue
		}
		
		// Predict resource exhaustion based on current trend
		if tracker.GrowthRate > 0 && tracker.UtilizationRate > 0.7 {
			remainingCapacity := tracker.MaxCapacity - tracker.CurrentUsage
			timeToExhaustion := time.Duration(remainingCapacity/tracker.GrowthRate) * time.Hour
			
			if timeToExhaustion < 24*time.Hour { // Warn if exhaustion within 24 hours
				prediction := ResourceExhaustionPrediction{
					ResourceType:        tracker.ResourceType,
					Component:           component,
					PredictedExhaustion: time.Now().Add(timeToExhaustion),
					Confidence:          0.8, // Simplified confidence calculation
					CurrentTrend:        "increasing",
					TimeToExhaustion:    timeToExhaustion,
					RecommendedActions: []RecommendedAction{
						{
							Action:      fmt.Sprintf("scale_%s", tracker.ResourceType),
							Priority:    PriorityUrgent,
							Description: fmt.Sprintf("Scale %s resources before exhaustion", tracker.ResourceType),
							Automated:   true,
							ETA:         30 * time.Minute,
						},
					},
				}
				predictions = append(predictions, prediction)
				
				// Record metric
				hp.predictorMetrics.ResourceExhaustionWarnings.WithLabelValues(component, string(tracker.ResourceType)).Inc()
			}
		}
	}
	
	return predictions
}

// detectAnomalies detects anomalies in health data
func (hp *HealthPredictor) detectAnomalies(component string, trainingData *ModelTrainingData) []DetectedAnomaly {
	var anomalies []DetectedAnomaly
	
	if len(trainingData.Targets) < 10 {
		return anomalies // Need sufficient data for anomaly detection
	}
	
	// Statistical outlier detection
	anomalies = append(anomalies, hp.detectStatisticalOutliers(component, trainingData)...)
	
	return anomalies
}

// detectStatisticalOutliers detects statistical outliers in the data
func (hp *HealthPredictor) detectStatisticalOutliers(component string, trainingData *ModelTrainingData) []DetectedAnomaly {
	var anomalies []DetectedAnomaly
	
	// Calculate mean and standard deviation
	var sum, sumSquares float64
	n := float64(len(trainingData.Targets))
	
	for _, value := range trainingData.Targets {
		sum += value
		sumSquares += value * value
	}
	
	mean := sum / n
	variance := (sumSquares / n) - (mean * mean)
	stdDev := math.Sqrt(variance)
	
	// Detect outliers (values more than 3 standard deviations from mean)
	threshold := 3.0
	
	for i, value := range trainingData.Targets {
		if len(trainingData.Timestamps) <= i {
			continue
		}
		
		deviation := math.Abs(value - mean)
		if deviation > threshold*stdDev {
			severity := SeverityMedium
			if deviation > 4*stdDev {
				severity = SeverityHigh
			}
			
			anomaly := DetectedAnomaly{
				Component:     component,
				Timestamp:     trainingData.Timestamps[i],
				Value:         value,
				ExpectedValue: mean,
				AnomalyScore:  deviation / stdDev,
				Algorithm:     AnomalyStatisticalOutlier,
				Severity:      severity,
				Description:   fmt.Sprintf("Value %.3f deviates %.1f standard deviations from mean %.3f", value, deviation/stdDev, mean),
			}
			anomalies = append(anomalies, anomaly)
		}
	}
	
	return anomalies
}

// detectSeasonalPatterns detects seasonal patterns in health data
func (hp *HealthPredictor) detectSeasonalPatterns(component string, trainingData *ModelTrainingData) []SeasonalPattern {
	var patterns []SeasonalPattern
	
	if len(trainingData.Timestamps) < 50 {
		return patterns // Need sufficient data for pattern detection
	}
	
	// Simplified seasonal pattern detection
	// In a full implementation, this would use FFT or autocorrelation
	
	// Check for daily patterns (simplified)
	dailyPattern := hp.checkDailyPattern(trainingData)
	if dailyPattern != nil {
		patterns = append(patterns, *dailyPattern)
	}
	
	return patterns
}

// checkDailyPattern checks for daily seasonal patterns
func (hp *HealthPredictor) checkDailyPattern(trainingData *ModelTrainingData) *SeasonalPattern {
	// Group data by hour of day
	hourlyData := make(map[int][]float64)
	
	for i, timestamp := range trainingData.Timestamps {
		if i >= len(trainingData.Targets) {
			continue
		}
		
		hour := timestamp.Hour()
		hourlyData[hour] = append(hourlyData[hour], trainingData.Targets[i])
	}
	
	// Calculate hourly averages
	hourlyAverages := make([]float64, 24)
	for hour := 0; hour < 24; hour++ {
		if values, exists := hourlyData[hour]; exists && len(values) > 0 {
			sum := 0.0
			for _, value := range values {
				sum += value
			}
			hourlyAverages[hour] = sum / float64(len(values))
		}
	}
	
	// Check if there's a significant daily pattern
	var totalVariation float64
	globalMean := 0.0
	validHours := 0
	
	for _, avg := range hourlyAverages {
		if avg > 0 {
			globalMean += avg
			validHours++
		}
	}
	
	if validHours == 0 {
		return nil
	}
	
	globalMean /= float64(validHours)
	
	for _, avg := range hourlyAverages {
		if avg > 0 {
			totalVariation += math.Abs(avg - globalMean)
		}
	}
	
	// If variation is significant, consider it a pattern
	if totalVariation/globalMean > 0.1 { // 10% variation threshold
		return &SeasonalPattern{
			Period:      24 * time.Hour,
			Amplitude:   totalVariation / float64(validHours),
			Phase:       0, // Simplified
			Confidence:  0.7, // Simplified confidence
			Description: "Daily pattern detected in health metrics",
		}
	}
	
	return nil
}

// initializeDefaultModels initializes default models for common components
func (hp *HealthPredictor) initializeDefaultModels() {
	defaultComponents := []string{
		"llm-processor",
		"rag-api",
		"weaviate",
		"kubernetes-api",
	}
	
	for _, component := range defaultComponents {
		_, err := hp.createModel(component)
		if err != nil {
			hp.logger.Error("Failed to create default model",
				"component", component,
				"error", err)
		}
	}
}

// GetActiveWarnings returns all currently active warnings
func (hp *HealthPredictor) GetActiveWarnings() []HealthWarning {
	hp.earlyWarning.warningMu.RLock()
	defer hp.earlyWarning.warningMu.RUnlock()
	
	var activeWarnings []HealthWarning
	for _, warning := range hp.earlyWarning.activeWarnings {
		if warning.Status == WarningStatusActive {
			activeWarnings = append(activeWarnings, *warning)
		}
	}
	
	return activeWarnings
}

// GetModelPerformance returns performance metrics for all models
func (hp *HealthPredictor) GetModelPerformance() map[string]*PredictionModel {
	hp.modelsMu.RLock()
	defer hp.modelsMu.RUnlock()
	
	result := make(map[string]*PredictionModel)
	for component, model := range hp.models {
		// Return a copy to avoid race conditions
		modelCopy := *model
		result[component] = &modelCopy
	}
	
	return result
}

// UpdateResourceUsage updates resource usage data for monitoring
func (hp *HealthPredictor) UpdateResourceUsage(component string, resourceType ResourceType, usage, capacity float64) {
	hp.resourceMonitor.mu.Lock()
	defer hp.resourceMonitor.mu.Unlock()
	
	key := fmt.Sprintf("%s-%s", component, resourceType)
	
	tracker, exists := hp.resourceMonitor.resourceTrackers[key]
	if !exists {
		tracker = &ResourceTracker{
			ResourceType: resourceType,
			Component:    component,
			MaxCapacity:  capacity,
			History:      []ResourceDataPoint{},
		}
		hp.resourceMonitor.resourceTrackers[key] = tracker
	}
	
	// Update current values
	prevUsage := tracker.CurrentUsage
	tracker.CurrentUsage = usage
	tracker.MaxCapacity = capacity
	tracker.UtilizationRate = usage / capacity
	tracker.LastUpdated = time.Now()
	
	// Calculate growth rate (simplified)
	if prevUsage > 0 {
		tracker.GrowthRate = (usage - prevUsage) / prevUsage
	}
	
	// Add to history
	dataPoint := ResourceDataPoint{
		Timestamp:   time.Now(),
		Usage:       usage,
		Capacity:    capacity,
		Utilization: tracker.UtilizationRate,
	}
	tracker.History = append(tracker.History, dataPoint)
	
	// Keep only last 100 points
	if len(tracker.History) > 100 {
		tracker.History = tracker.History[len(tracker.History)-100:]
	}
}