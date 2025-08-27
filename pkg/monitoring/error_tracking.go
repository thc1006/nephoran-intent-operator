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
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/thc1006/nephoran-intent-operator/pkg/errors"
)

// TrendAnalyzer analyzes trends in error patterns
type TrendAnalyzer struct {
	logger logr.Logger
	mutex  sync.RWMutex
}

// SeasonalityDetector detects seasonal patterns
type SeasonalityDetector struct {
	logger logr.Logger
}

// SeasonalPattern represents a seasonal pattern
type SeasonalPattern struct {
	Pattern string `json:"pattern"`
}

// TimeSeries represents time series data
type TimeSeries struct {
	Timestamps []time.Time `json:"timestamps"`
	Values     []float64   `json:"values"`
	maxSize    int
	mutex      sync.RWMutex
}

// NewTimeSeries creates a new TimeSeries with specified maximum size
func NewTimeSeries(maxSize int) *TimeSeries {
	return &TimeSeries{
		Timestamps: make([]time.Time, 0, maxSize),
		Values:     make([]float64, 0, maxSize),
		maxSize:    maxSize,
	}
}

// Add adds a new data point to the time series
func (ts *TimeSeries) Add(timestamp time.Time, value float64) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	ts.Timestamps = append(ts.Timestamps, timestamp)
	ts.Values = append(ts.Values, value)

	// Keep only the most recent maxSize points
	if len(ts.Timestamps) > ts.maxSize {
		ts.Timestamps = ts.Timestamps[len(ts.Timestamps)-ts.maxSize:]
		ts.Values = ts.Values[len(ts.Values)-ts.maxSize:]
	}
}

// GetRecent returns the most recent n data points
func (ts *TimeSeries) GetRecent(n int) ([]time.Time, []float64) {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()

	size := len(ts.Timestamps)
	if n > size {
		n = size
	}

	if n <= 0 {
		return []time.Time{}, []float64{}
	}

	startIdx := size - n
	return ts.Timestamps[startIdx:], ts.Values[startIdx:]
}

// TrendModel represents a trend model
type TrendModel struct {
	ModelType string `json:"model_type"`
}

// NewTrendAnalyzer creates a new trend analyzer
func NewTrendAnalyzer() *TrendAnalyzer {
	return &TrendAnalyzer{
		logger: logr.Discard(),
		mutex:  sync.RWMutex{},
	}
}

// NewSeasonalityDetector creates a new seasonality detector
func NewSeasonalityDetector() *SeasonalityDetector {
	return &SeasonalityDetector{
		logger: logr.Discard(),
	}
}

// GetSeasonalAdjustment returns seasonal adjustment for the given time and horizon
func (sd *SeasonalityDetector) GetSeasonalAdjustment(timestamp time.Time, horizon time.Duration) float64 {
	// Mock seasonal adjustment based on hour of day
	hour := timestamp.Hour()

	// Simple seasonal pattern: higher values during business hours
	if hour >= 9 && hour <= 17 {
		return 0.1 // 10% adjustment during business hours
	}

	return -0.05 // 5% negative adjustment during off-hours
}

// ErrorTrackingSystem provides comprehensive error monitoring and observability
type ErrorTrackingSystem struct {
	// Core components
	errorAggregator *errors.ErrorAggregator
	recoveryManager *errors.AdvancedRecoveryManager

	// Monitoring backends
	prometheusMetrics   *PrometheusErrorMetrics
	openTelemetryTracer trace.Tracer
	openTelemetryMeter  metric.Meter

	// Alerting
	alertManager        *AlertManager
	notificationManager *NotificationManager

	// Dashboard and reporting
	dashboardManager *DashboardManager
	reportGenerator  *ErrorReportGenerator

	// Configuration and state
	config *ErrorTrackingConfig
	logger logr.Logger

	// Error processing
	errorStream       chan *ErrorEvent
	processingWorkers []*ErrorProcessingWorker

	// Analytics and ML
	errorAnalyzer    *ErrorAnalyzer
	predictionEngine *ErrorPredictionEngine
	anomalyDetector  *AnomalyDetector

	// State management
	mutex    sync.RWMutex
	started  bool
	stopChan chan struct{}
}

// ErrorTrackingConfig holds configuration for error tracking

// ErrorEvent represents an error event for monitoring
type ErrorEvent struct {
	// Basic information
	EventID   string                  `json:"eventId"`
	Timestamp time.Time               `json:"timestamp"`
	Error     *errors.ProcessingError `json:"error"`

	// Context information
	TraceID       string `json:"traceId"`
	SpanID        string `json:"spanId"`
	CorrelationID string `json:"correlationId"`
	RequestID     string `json:"requestId"`

	// Processing information
	Component string `json:"component"`
	Operation string `json:"operation"`
	Phase     string `json:"phase"`

	// Recovery information
	RecoveryAttempts int           `json:"recoveryAttempts"`
	RecoverySuccess  bool          `json:"recoverySuccess"`
	RecoveryDuration time.Duration `json:"recoveryDuration"`

	// Impact assessment
	ImpactLevel    string                 `json:"impactLevel"`
	AffectedUsers  int64                  `json:"affectedUsers"`
	BusinessImpact map[string]interface{} `json:"businessImpact"`

	// Additional metadata
	Tags       map[string]string      `json:"tags"`
	Labels     map[string]string      `json:"labels"`
	CustomData map[string]interface{} `json:"customData"`
}

// PrometheusErrorMetrics provides Prometheus metrics for error tracking
type PrometheusErrorMetrics struct {
	// Error counters
	errorsTotal       *prometheus.CounterVec
	errorsByCategory  *prometheus.CounterVec
	errorsBySeverity  *prometheus.CounterVec
	errorsByComponent *prometheus.CounterVec

	// Recovery metrics
	recoveryAttemptsTotal *prometheus.CounterVec
	recoverySuccessTotal  *prometheus.CounterVec
	recoveryDurationHist  *prometheus.HistogramVec

	// Performance metrics
	errorProcessingDuration *prometheus.HistogramVec
	errorProcessingRate     *prometheus.GaugeVec

	// System health metrics
	activeErrors       *prometheus.GaugeVec
	errorRate          *prometheus.GaugeVec
	meanTimeToRecovery *prometheus.GaugeVec

	// Circuit breaker metrics
	circuitBreakerState *prometheus.GaugeVec
	circuitBreakerTrips *prometheus.CounterVec

	registry prometheus.Registerer
}

// NewPrometheusErrorMetrics creates a new PrometheusErrorMetrics instance
func NewPrometheusErrorMetrics(namespace string) *PrometheusErrorMetrics {
	metrics := &PrometheusErrorMetrics{
		registry: prometheus.DefaultRegisterer,
	}

	// Initialize error counters
	metrics.errorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "errors_total",
			Help:      "Total number of errors tracked",
		},
		[]string{"error_id", "category", "severity", "component"},
	)

	metrics.errorsByCategory = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "errors_by_category_total",
			Help:      "Total errors grouped by category",
		},
		[]string{"category"},
	)

	metrics.errorsBySeverity = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "errors_by_severity_total",
			Help:      "Total errors grouped by severity",
		},
		[]string{"severity"},
	)

	metrics.errorsByComponent = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "errors_by_component_total",
			Help:      "Total errors grouped by component",
		},
		[]string{"component"},
	)

	// Initialize recovery metrics
	metrics.recoveryAttemptsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "recovery_attempts_total",
			Help:      "Total number of error recovery attempts",
		},
		[]string{"error_id", "strategy"},
	)

	metrics.recoverySuccessTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "recovery_success_total",
			Help:      "Total number of successful error recoveries",
		},
		[]string{"error_id", "strategy"},
	)

	metrics.recoveryDurationHist = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "recovery_duration_seconds",
			Help:      "Duration of error recovery attempts",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"error_id", "strategy"},
	)

	// Initialize performance metrics
	metrics.errorProcessingDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "error_processing_duration_seconds",
			Help:      "Duration of error processing",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	metrics.errorProcessingRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "error_processing_rate",
			Help:      "Rate of error processing",
		},
		[]string{"operation"},
	)

	// Initialize system health metrics
	metrics.activeErrors = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "active_errors",
			Help:      "Current number of active errors",
		},
		[]string{"category"},
	)

	metrics.errorRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "error_rate",
			Help:      "Current error rate",
		},
		[]string{"window"},
	)

	metrics.meanTimeToRecovery = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "mean_time_to_recovery_seconds",
			Help:      "Mean time to recovery for errors",
		},
		[]string{"category"},
	)

	// Initialize circuit breaker metrics
	metrics.circuitBreakerState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "circuit_breaker_state",
			Help:      "Current state of circuit breakers (0=closed, 1=open, 2=half-open)",
		},
		[]string{"circuit"},
	)

	metrics.circuitBreakerTrips = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "circuit_breaker_trips_total",
			Help:      "Total number of circuit breaker trips",
		},
		[]string{"circuit", "reason"},
	)

	// Register all metrics
	metrics.registry.MustRegister(
		metrics.errorsTotal,
		metrics.errorsByCategory,
		metrics.errorsBySeverity,
		metrics.errorsByComponent,
		metrics.recoveryAttemptsTotal,
		metrics.recoverySuccessTotal,
		metrics.recoveryDurationHist,
		metrics.errorProcessingDuration,
		metrics.errorProcessingRate,
		metrics.activeErrors,
		metrics.errorRate,
		metrics.meanTimeToRecovery,
		metrics.circuitBreakerState,
		metrics.circuitBreakerTrips,
	)

	return metrics
}

// Note: AlertRule and AlertSeverity types are defined in alerting.go

// Note: NotificationChannel type is defined in alerting.go

// NotificationChannelType defines notification channel types
type NotificationChannelType string

const (
	ChannelEmail     NotificationChannelType = "email"
	ChannelSlack     NotificationChannelType = "slack"
	ChannelPagerDuty NotificationChannelType = "pagerduty"
	ChannelWebhook   NotificationChannelType = "webhook"
	ChannelSMS       NotificationChannelType = "sms"
)

// Note: AlertManager type is defined in alerting.go

// ActiveAlert represents a currently active alert
type ActiveAlert struct {
	ID                string            `json:"id"`
	Rule              AlertRule         `json:"rule"`
	FireTime          time.Time         `json:"fireTime"`
	LastEvaluation    time.Time         `json:"lastEvaluation"`
	Value             float64           `json:"value"`
	Labels            map[string]string `json:"labels"`
	Annotations       map[string]string `json:"annotations"`
	Status            AlertStatus       `json:"status"`
	NotificationsSent int               `json:"notificationsSent"`
}

// AlertStatus represents the status of an alert
type AlertStatus string

const (
	AlertStatusPending  AlertStatus = "pending"
	AlertStatusFiring   AlertStatus = "firing"
	AlertStatusResolved AlertStatus = "resolved"
	AlertStatusSilenced AlertStatus = "silenced"
)

// NotificationManager manages alert notifications
type NotificationManager struct {
	channels    []NotificationChannel
	templates   map[NotificationChannelType]*NotificationTemplate
	rateLimiter *NotificationRateLimiter

	logger logr.Logger
	mutex  sync.RWMutex
}

// NotificationTemplate defines message templates for different channels
type NotificationTemplate struct {
	Subject   string   `json:"subject"`
	Body      string   `json:"body"`
	Format    string   `json:"format"` // text, html, markdown, json
	Variables []string `json:"variables"`
}

// NotificationRateLimiter prevents notification spam
type NotificationRateLimiter struct {
	limits      map[string]*RateLimit // per channel limits
	globalLimit *RateLimit
	mutex       sync.RWMutex
}

// RateLimit defines rate limiting parameters
type RateLimit struct {
	MaxNotifications int           `json:"maxNotifications"`
	TimeWindow       time.Duration `json:"timeWindow"`
	BurstAllowed     int           `json:"burstAllowed"`

	// State
	notifications []time.Time
	mutex         sync.RWMutex
}

// ErrorProcessingWorker processes error events
type ErrorProcessingWorker struct {
	id             int
	errorStream    chan *ErrorEvent
	trackingSystem *ErrorTrackingSystem

	// Processing state
	processedEvents int64
	lastActivity    time.Time

	logger   logr.Logger
	stopChan chan struct{}
}

// DashboardManager provides web dashboard for error tracking
type DashboardManager struct {
	config         *DashboardConfig
	webServer      *WebServer
	templateEngine *TemplateEngine

	// Dashboard components
	errorSummary    *ErrorSummaryWidget
	errorTimeline   *TimelineWidget
	componentHealth *HealthWidget
	alertsWidget    *AlertsWidget
	recoveryStats   *RecoveryStatsWidget

	logger logr.Logger
}

// DashboardConfig configures the dashboard
type DashboardConfig struct {
	Port            int               `json:"port"`
	Path            string            `json:"path"`
	Title           string            `json:"title"`
	RefreshInterval int               `json:"refreshInterval"`
	Theme           string            `json:"theme"`
	EnableAuth      bool              `json:"enableAuth"`
	BasicAuthUsers  map[string]string `json:"basicAuthUsers"`
}

// ErrorReportGenerator generates error reports
type ErrorReportGenerator struct {
	templates        map[string]*ReportTemplate
	scheduledReports []ScheduledReport

	logger logr.Logger
	mutex  sync.RWMutex
}

// ReportTemplate defines report structure
type ReportTemplate struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Format      ReportFormat           `json:"format"`
	Sections    []ReportSection        `json:"sections"`
	TimeRange   time.Duration          `json:"timeRange"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ReportFormat defines report output formats
type ReportFormat string

const (
	FormatHTML ReportFormat = "html"
	FormatPDF  ReportFormat = "pdf"
	FormatJSON ReportFormat = "json"
	FormatCSV  ReportFormat = "csv"
)

// ReportSection defines a section of a report
type ReportSection struct {
	Name       string                 `json:"name"`
	Type       SectionType            `json:"type"`
	Query      string                 `json:"query"`
	Parameters map[string]interface{} `json:"parameters"`
}

// SectionType defines types of report sections
type SectionType string

const (
	SectionSummary  SectionType = "summary"
	SectionChart    SectionType = "chart"
	SectionTable    SectionType = "table"
	SectionMetrics  SectionType = "metrics"
	SectionAnalysis SectionType = "analysis"
)

// ScheduledReport defines a scheduled report
type ScheduledReport struct {
	ID         string     `json:"id"`
	Template   string     `json:"template"`
	Schedule   string     `json:"schedule"` // cron expression
	Recipients []string   `json:"recipients"`
	Enabled    bool       `json:"enabled"`
	LastRun    *time.Time `json:"lastRun"`
	NextRun    time.Time  `json:"nextRun"`
}

// ErrorAnalyzer performs advanced error analysis
type ErrorAnalyzer struct {
	patterns         []ErrorPattern
	correlationRules []CorrelationRule
	trendAnalyzer    *TrendAnalyzer

	logger logr.Logger
	mutex  sync.RWMutex
}

// CorrelationRule defines how errors are correlated
type CorrelationRule struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	Conditions     []string          `json:"conditions"`
	TimeWindow     time.Duration     `json:"timeWindow"`
	MinOccurrences int               `json:"minOccurrences"`
	Action         CorrelationAction `json:"action"`
}

// CorrelationAction defines actions for correlated errors
type CorrelationAction struct {
	Type       ActionType             `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
}

// ActionType defines correlation action types
type ActionType string

const (
	ActionAlert          ActionType = "alert"
	ActionAutoRecover    ActionType = "auto_recover"
	ActionEscalate       ActionType = "escalate"
	ActionSuppressAlerts ActionType = "suppress_alerts"
)

// TrendAnalyzer is defined in types.go

// TimeSeries is defined in types.go

// DataPoint is defined in types.go

// AggregationType is defined in types.go

// TrendModel is defined in types.go

// TrendModelType is defined in types.go

// TrendPrediction is defined in types.go

// ErrorPredictionEngine predicts future errors
type ErrorPredictionEngine struct {
	models       map[string]*PredictionModel
	features     []FeatureExtractor
	trainingData *TrainingDataset

	logger logr.Logger
	mutex  sync.RWMutex
}

// PredictionModel represents an ML model for error prediction
type PredictionModel struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	ModelType    PredictionModelType    `json:"modelType"`
	Features     []string               `json:"features"`
	Accuracy     float64                `json:"accuracy"`
	Precision    float64                `json:"precision"`
	Recall       float64                `json:"recall"`
	F1Score      float64                `json:"f1Score"`
	LastTrained  time.Time              `json:"lastTrained"`
	TrainingSize int                    `json:"trainingSize"`
	Parameters   map[string]interface{} `json:"parameters"`
}

// PredictionModelType defines prediction model types
type PredictionModelType string

const (
	ModelLogisticRegression PredictionModelType = "logistic_regression"
	ModelRandomForest       PredictionModelType = "random_forest"
	ModelSVM                PredictionModelType = "svm"
	ModelNeuralNetwork      PredictionModelType = "neural_network"
	ModelXGBoost            PredictionModelType = "xgboost"
)

// FeatureExtractor extracts features for ML models
type FeatureExtractor interface {
	ExtractFeatures(event *ErrorEvent) (map[string]float64, error)
	GetFeatureNames() []string
}

// TrainingDataset holds training data for ML models
type TrainingDataset struct {
	Features   []map[string]float64     `json:"features"`
	Labels     []float64                `json:"labels"`
	Metadata   []map[string]interface{} `json:"metadata"`
	Size       int                      `json:"size"`
	LastUpdate time.Time                `json:"lastUpdate"`
}

// AnomalyDetector is defined in types.go

// AnomalyDetectorModel is defined in types.go

// AnomalyAlgorithm is defined in types.go

// Baseline is defined in types.go

// AnomalyAlertRule is defined in types.go

// Operator is defined in types.go

// NewErrorTrackingSystem creates a new error tracking system
func NewErrorTrackingSystem(config *ErrorTrackingConfig, errorAggregator *errors.ErrorAggregator, logger logr.Logger) *ErrorTrackingSystem {
	if config == nil {
		config = getDefaultErrorTrackingConfig()
	}

	ets := &ErrorTrackingSystem{
		errorAggregator: errorAggregator,
		config:          config,
		logger:          logger.WithName("error-tracking"),
		errorStream:     make(chan *ErrorEvent, config.ErrorBufferSize),
		stopChan:        make(chan struct{}),
	}

	// Initialize Prometheus metrics if enabled
	if config.PrometheusEnabled {
		ets.prometheusMetrics = NewPrometheusErrorMetrics(config.PrometheusNamespace)
	}

	// Initialize OpenTelemetry if enabled
	if config.OpenTelemetryEnabled {
		ets.openTelemetryTracer = otel.Tracer(config.ServiceName)
		ets.openTelemetryMeter = otel.Meter(config.ServiceName)
	}

	// Initialize alerting if enabled
	if config.AlertingEnabled {
		alertConfig := &AlertManagerConfig{
			EvaluationInterval: 30 * time.Second,
		}
		ets.alertManager = NewAlertManager(alertConfig, nil, nil)
		ets.notificationManager = NewNotificationManager(config.NotificationChannels, logger)
	}

	// Initialize dashboard if enabled
	if config.DashboardEnabled {
		dashboardConfig := &DashboardConfig{
			Port:            config.DashboardPort,
			Path:            "/dashboard",
			Title:           "Error Tracking Dashboard",
			RefreshInterval: 30,
			Theme:           "dark",
		}
		ets.dashboardManager = NewDashboardManager(dashboardConfig, logger)
	}

	// Initialize analytics if enabled
	if config.AnalyticsEnabled {
		ets.errorAnalyzer = NewErrorAnalyzer(logger)

		if config.PredictionEnabled {
			ets.predictionEngine = NewErrorPredictionEngine(logger)
		}

		if config.AnomalyDetectionEnabled {
			ets.anomalyDetector = NewAnomalyDetectorWithLogr(logger)
		}
	}

	// Initialize report generator
	ets.reportGenerator = NewErrorReportGenerator(logger)

	// Create processing workers
	ets.processingWorkers = make([]*ErrorProcessingWorker, config.ProcessingWorkers)
	for i := 0; i < config.ProcessingWorkers; i++ {
		ets.processingWorkers[i] = NewErrorProcessingWorker(i, ets.errorStream, ets, logger)
	}

	return ets
}

// Start starts the error tracking system
func (ets *ErrorTrackingSystem) Start(ctx context.Context) error {
	ets.mutex.Lock()
	defer ets.mutex.Unlock()

	if ets.started {
		return fmt.Errorf("error tracking system already started")
	}

	ets.logger.Info("Starting error tracking system")

	// Start processing workers
	for _, worker := range ets.processingWorkers {
		go worker.Start(ctx)
	}

	// Start alert manager
	if ets.alertManager != nil {
		go ets.alertManager.Start(ctx)
	}

	// Start dashboard
	if ets.dashboardManager != nil {
		go ets.dashboardManager.Start(ctx)
	}

	// Start background tasks
	go ets.metricsCollector(ctx)
	go ets.reportScheduler(ctx)

	ets.started = true
	ets.logger.Info("Error tracking system started successfully")

	return nil
}

// Stop stops the error tracking system
func (ets *ErrorTrackingSystem) Stop() error {
	ets.mutex.Lock()
	defer ets.mutex.Unlock()

	if !ets.started {
		return nil
	}

	ets.logger.Info("Stopping error tracking system")

	// Signal stop
	close(ets.stopChan)

	// Stop components
	if ets.alertManager != nil {
		ets.alertManager.Stop()
	}

	if ets.dashboardManager != nil {
		ets.dashboardManager.Stop()
	}

	// Stop workers
	for _, worker := range ets.processingWorkers {
		worker.Stop()
	}

	ets.started = false
	ets.logger.Info("Error tracking system stopped")

	return nil
}

// TrackError tracks an error event
func (ets *ErrorTrackingSystem) TrackError(ctx context.Context, err *errors.ProcessingError) error {
	if !ets.started {
		return fmt.Errorf("error tracking system not started")
	}

	// Extract trace information
	span := trace.SpanFromContext(ctx)
	spanContext := span.SpanContext()

	// Create error event
	event := &ErrorEvent{
		EventID:       fmt.Sprintf("err_%d", time.Now().UnixNano()),
		Timestamp:     time.Now(),
		Error:         err,
		TraceID:       spanContext.TraceID().String(),
		SpanID:        spanContext.SpanID().String(),
		CorrelationID: err.CorrelationID,
		Component:     err.Component,
		Operation:     err.Operation,
		Phase:         err.Phase,
		ImpactLevel:   string(err.Severity),
		Tags:          make(map[string]string),
		Labels:        make(map[string]string),
		CustomData:    make(map[string]interface{}),
	}

	// Add additional context
	event.Tags["category"] = string(err.Category)
	event.Tags["severity"] = string(err.Severity)
	event.Tags["component"] = err.Component
	event.Labels["error_code"] = err.Code

	// Queue for processing
	select {
	case ets.errorStream <- event:
		return nil
	default:
		ets.logger.V(1).Info("Error stream buffer full, dropping error event", "errorId", err.ID)
		return fmt.Errorf("error stream buffer full")
	}
}

// TrackRecovery tracks an error recovery event
func (ets *ErrorTrackingSystem) TrackRecovery(ctx context.Context, errorID string, success bool, duration time.Duration, attempts int) error {
	if !ets.started {
		return nil
	}

	// Update Prometheus metrics
	if ets.prometheusMetrics != nil {
		labels := prometheus.Labels{
			"error_id": errorID,
			"success":  fmt.Sprintf("%t", success),
		}

		ets.prometheusMetrics.recoveryAttemptsTotal.With(labels).Add(float64(attempts))

		if success {
			ets.prometheusMetrics.recoverySuccessTotal.With(labels).Inc()
		}

		ets.prometheusMetrics.recoveryDurationHist.With(labels).Observe(duration.Seconds())
	}

	// Record OpenTelemetry metrics
	if ets.openTelemetryMeter != nil {
		attrs := []attribute.KeyValue{
			attribute.String("error_id", errorID),
			attribute.Bool("success", success),
		}

		if counter, err := ets.openTelemetryMeter.Int64Counter("error_recovery_attempts"); err == nil {
			counter.Add(ctx, int64(attempts), metric.WithAttributes(attrs...))
		}

		if histogram, err := ets.openTelemetryMeter.Float64Histogram("error_recovery_duration"); err == nil {
			histogram.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
		}
	}

	return nil
}

// metricsCollector periodically collects and updates metrics
func (ets *ErrorTrackingSystem) metricsCollector(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ets.collectMetrics(ctx)

		case <-ets.stopChan:
			return

		case <-ctx.Done():
			return
		}
	}
}

// collectMetrics collects metrics from various sources
func (ets *ErrorTrackingSystem) collectMetrics(ctx context.Context) {
	// Collect error aggregator metrics
	if ets.errorAggregator != nil {
		// Mock implementation - in real implementation, would use actual statistics
		totalErrors := 100
		errorCounts := map[string]int64{
			"validation": 50,
			"timeout":    30,
			"network":    20,
		}

		// Update Prometheus metrics
		if ets.prometheusMetrics != nil {
			ets.prometheusMetrics.activeErrors.WithLabelValues("total").Set(float64(totalErrors))

			for category, count := range errorCounts {
				ets.prometheusMetrics.errorsByCategory.WithLabelValues(category).Add(float64(count))
			}
		}
	}

	// Collect recovery manager metrics if available
	if ets.recoveryManager != nil {
		recoveryMetrics := ets.recoveryManager.GetAdvancedMetrics()

		if ets.prometheusMetrics != nil {
			ets.prometheusMetrics.meanTimeToRecovery.WithLabelValues("overall").Set(float64(recoveryMetrics.AverageRecoveryTime.Milliseconds()))
		}
	}
}

// reportScheduler schedules and generates reports
func (ets *ErrorTrackingSystem) reportScheduler(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute) // Check every minute
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if ets.reportGenerator != nil {
				ets.reportGenerator.CheckScheduledReports(ctx)
			}

		case <-ets.stopChan:
			return

		case <-ctx.Done():
			return
		}
	}
}

// GetMetrics returns comprehensive error tracking metrics
func (ets *ErrorTrackingSystem) GetMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	// Collect worker metrics
	workerMetrics := make([]map[string]interface{}, len(ets.processingWorkers))
	for i, worker := range ets.processingWorkers {
		workerMetrics[i] = map[string]interface{}{
			"id":               worker.id,
			"processed_events": worker.processedEvents,
			"last_activity":    worker.lastActivity,
			"status":           "active",
		}
	}
	metrics["workers"] = workerMetrics

	// Collect alert metrics
	if ets.alertManager != nil {
		metrics["alerts"] = ets.alertManager.GetMetrics()
	}

	// Collect analytics metrics
	if ets.errorAnalyzer != nil {
		metrics["analytics"] = ets.errorAnalyzer.GetMetrics()
	}

	return metrics
}

// GetHealthStatus returns the health status of the error tracking system
func (ets *ErrorTrackingSystem) GetHealthStatus() map[string]interface{} {
	health := map[string]interface{}{
		"status":     "healthy",
		"timestamp":  time.Now(),
		"components": map[string]interface{}{},
	}

	components := health["components"].(map[string]interface{})

	// Check worker health
	activeWorkers := 0
	for _, worker := range ets.processingWorkers {
		if time.Since(worker.lastActivity) < 5*time.Minute {
			activeWorkers++
		}
	}

	components["workers"] = map[string]interface{}{
		"total":   len(ets.processingWorkers),
		"active":  activeWorkers,
		"healthy": activeWorkers > 0,
	}

	// Check alert manager health
	if ets.alertManager != nil {
		components["alerting"] = map[string]interface{}{
			"healthy": true, // Would implement actual health check
		}
	}

	// Check dashboard health
	if ets.dashboardManager != nil {
		components["dashboard"] = map[string]interface{}{
			"healthy": true, // Would implement actual health check
		}
	}

	// Overall health
	allHealthy := true
	for _, component := range components {
		if comp, ok := component.(map[string]interface{}); ok {
			if healthy, exists := comp["healthy"]; exists {
				if !healthy.(bool) {
					allHealthy = false
					break
				}
			}
		}
	}

	if !allHealthy {
		health["status"] = "unhealthy"
	}

	return health
}

// Helper functions

// getDefaultErrorTrackingConfig returns default configuration
func getDefaultErrorTrackingConfig() *ErrorTrackingConfig {
	return &ErrorTrackingConfig{
		Enabled:                 true,
		ProcessingWorkers:       3,
		ErrorBufferSize:         1000,
		PrometheusEnabled:       true,
		PrometheusNamespace:     "nephoran_errors",
		MetricsPort:             8080,
		OpenTelemetryEnabled:    true,
		ServiceName:             "nephoran-intent-operator",
		AlertingEnabled:         true,
		AlertRules:              make([]AlertRule, 0),
		NotificationChannels:    make([]NotificationChannel, 0),
		DashboardEnabled:        true,
		DashboardPort:           8090,
		AnalyticsEnabled:        true,
		PredictionEnabled:       false, // Disabled by default
		AnomalyDetectionEnabled: false, // Disabled by default
		MetricsRetentionDays:    30,
		TracesRetentionDays:     7,
		ReportsRetentionDays:    90,
	}
}

// Placeholder implementations for referenced types
type WebServer struct{ logger logr.Logger }
type TemplateEngine struct{ logger logr.Logger }
type ErrorSummaryWidget struct{ logger logr.Logger }
type TimelineWidget struct{ logger logr.Logger }
type HealthWidget struct{ logger logr.Logger }
type AlertsWidget struct{ logger logr.Logger }
type RecoveryStatsWidget struct{ logger logr.Logger }

// Placeholder constructors
// Note: NewAlertManager function is defined in alerting.go

func NewNotificationManager(channels []NotificationChannel, logger logr.Logger) *NotificationManager {
	return &NotificationManager{logger: logger}
}

func NewDashboardManager(config *DashboardConfig, logger logr.Logger) *DashboardManager {
	return &DashboardManager{logger: logger}
}

func NewErrorReportGenerator(logger logr.Logger) *ErrorReportGenerator {
	return &ErrorReportGenerator{logger: logger}
}

func NewErrorAnalyzer(logger logr.Logger) *ErrorAnalyzer {
	return &ErrorAnalyzer{
		trendAnalyzer: NewTrendAnalyzer(),
		logger:        logger,
	}
}

func NewErrorPredictionEngine(logger logr.Logger) *ErrorPredictionEngine {
	return &ErrorPredictionEngine{logger: logger}
}

func NewAnomalyDetectorWithLogr(logger logr.Logger) *AnomalyDetector {
	// Use a default NWDAF config
	config := &NWDAFConfig{}
	detector := NewAnomalyDetector(config, logger)
	return detector
}

// Note: NewAnomalyDetector is defined in nwdaf_analytics_engine.go

func NewErrorProcessingWorker(id int, stream chan *ErrorEvent, ets *ErrorTrackingSystem, logger logr.Logger) *ErrorProcessingWorker {
	return &ErrorProcessingWorker{
		id:             id,
		errorStream:    stream,
		trackingSystem: ets,
		logger:         logger,
		stopChan:       make(chan struct{}),
	}
}

// Placeholder methods
// Note: AlertManager Start/Stop methods are defined in alerting.go
func (am *AlertManager) GetMetrics() map[string]interface{} { return make(map[string]interface{}) }

// Note: AlertManager Start/Stop methods are defined in alerting.go

func (dm *DashboardManager) Start(ctx context.Context) { /* Implementation */ }
func (dm *DashboardManager) Stop()                     { /* Implementation */ }

func (erg *ErrorReportGenerator) CheckScheduledReports(ctx context.Context) { /* Implementation */ }

func (ea *ErrorAnalyzer) GetMetrics() map[string]interface{} { return make(map[string]interface{}) }

func (epw *ErrorProcessingWorker) Start(ctx context.Context) { /* Implementation */ }
func (epw *ErrorProcessingWorker) Stop()                     { /* Implementation */ }

// Note: ErrorTrackingConfig, ErrorPattern, AlertRule, AlertSeverity, NotificationChannel
// AlertManagerConfig, NWDAFConfig and related types are defined in other monitoring files to avoid duplication

// InferenceConfig represents model inference configuration
type InferenceConfig struct {
	BatchSize           int           `json:"batch_size"`
	ConfidenceThreshold float64       `json:"confidence_threshold"`
	InferenceInterval   time.Duration `json:"inference_interval"`
}

// Note: SLAMonitoringConfig, AnomalyDetector and related types are also defined elsewhere
