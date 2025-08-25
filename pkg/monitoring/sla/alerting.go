package sla

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// AlertManager provides SLA violation alerting with multi-window burn rate detection
// and predictive violation analysis
type AlertManager struct {
	config  *AlertManagerConfig
	logger  *logging.StructuredLogger
	started atomic.Bool

	// Alert rules and conditions
	alertRules      map[string]*AlertRule
	burnRateRules   []*BurnRateRule
	alertConditions *AlertConditionEvaluator

	// Alert state management
	activeAlerts map[string]*ActiveAlert
	alertHistory *AlertHistory
	silences     *SilenceManager

	// Predictive alerting
	predictor     *ViolationPredictor
	trendAnalyzer *TrendAnalyzer

	// Notification management
	notificationMgr *NotificationManager
	escalationMgr   *EscalationManager

	// Performance metrics
	metrics         *AlertManagerMetrics
	alertsGenerated atomic.Uint64
	alertsResolved  atomic.Uint64
	falsePositives  atomic.Uint64

	// Background processing
	evaluationTicker *time.Ticker
	cleanupTicker    *time.Ticker
	stopCh           chan struct{}
	wg               sync.WaitGroup

	// Synchronization
	mu sync.RWMutex
}

// AlertManagerConfig holds configuration for the alert manager
type AlertManagerConfig struct {
	// Evaluation settings
	EvaluationInterval  time.Duration `yaml:"evaluation_interval"`
	NotificationTimeout time.Duration `yaml:"notification_timeout"`
	MaxPendingAlerts    int           `yaml:"max_pending_alerts"`
	DeduplicationWindow time.Duration `yaml:"deduplication_window"`

	// Burn rate alert configuration
	BurnRateWindows      []BurnRateWindow `yaml:"burn_rate_windows"`
	ErrorBudgetThreshold float64          `yaml:"error_budget_threshold"`

	// Predictive alerting
	PredictionEnabled   bool          `yaml:"prediction_enabled"`
	PredictionHorizon   time.Duration `yaml:"prediction_horizon"`
	ConfidenceThreshold float64       `yaml:"confidence_threshold"`

	// Notification settings
	NotificationChannels []NotificationChannel `yaml:"notification_channels"`
	EscalationPolicies   []EscalationPolicy    `yaml:"escalation_policies"`

	// Alert retention
	AlertRetentionPeriod time.Duration `yaml:"alert_retention_period"`
	HistoryMaxSize       int           `yaml:"history_max_size"`
}

// DefaultAlertManagerConfig returns optimized default configuration
func DefaultAlertManagerConfig() *AlertManagerConfig {
	return &AlertManagerConfig{
		EvaluationInterval:  30 * time.Second,
		NotificationTimeout: 30 * time.Second,
		MaxPendingAlerts:    1000,
		DeduplicationWindow: 5 * time.Minute,

		BurnRateWindows: []BurnRateWindow{
			{Name: "fast", LongWindow: 1 * time.Hour, ShortWindow: 5 * time.Minute, Factor: 14.4},
			{Name: "slow", LongWindow: 6 * time.Hour, ShortWindow: 30 * time.Minute, Factor: 6.0},
		},
		ErrorBudgetThreshold: 10.0, // Alert when 10% error budget remaining

		PredictionEnabled:   true,
		PredictionHorizon:   2 * time.Hour,
		ConfidenceThreshold: 0.8,

		NotificationChannels: []NotificationChannel{
			{Type: "webhook", URL: "http://alertmanager:9093/api/v1/alerts", Enabled: true},
			{Type: "log", Enabled: true},
		},

		EscalationPolicies: []EscalationPolicy{
			{
				Name:    "default",
				Enabled: true,
				Levels: []EscalationLevel{
					{Duration: 5 * time.Minute, Severity: "warning"},
					{Duration: 15 * time.Minute, Severity: "critical"},
					{Duration: 30 * time.Minute, Severity: "emergency"},
				},
			},
		},

		AlertRetentionPeriod: 7 * 24 * time.Hour, // 7 days
		HistoryMaxSize:       10000,
	}
}

// BurnRateWindow defines a time window for burn rate calculation
type BurnRateWindow struct {
	Name        string        `yaml:"name"`
	LongWindow  time.Duration `yaml:"long_window"`
	ShortWindow time.Duration `yaml:"short_window"`
	Factor      float64       `yaml:"factor"`
	Severity    string        `yaml:"severity"`
}

// NotificationChannel defines a notification destination
type NotificationChannel struct {
	Type    string                 `yaml:"type"` // webhook, email, slack, pagerduty
	URL     string                 `yaml:"url,omitempty"`
	Enabled bool                   `yaml:"enabled"`
	Config  map[string]interface{} `yaml:"config,omitempty"`
}

// EscalationPolicy defines how alerts escalate over time
type EscalationPolicy struct {
	Name    string            `yaml:"name"`
	Enabled bool              `yaml:"enabled"`
	Levels  []EscalationLevel `yaml:"levels"`
}

// EscalationLevel defines a single escalation level
type EscalationLevel struct {
	Duration time.Duration `yaml:"duration"`
	Severity string        `yaml:"severity"`
	Actions  []string      `yaml:"actions,omitempty"`
}

// AlertRule defines conditions for generating alerts
type AlertRule struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Metric      string            `json:"metric"`
	Condition   string            `json:"condition"` // "gt", "lt", "eq", "ne"
	Threshold   float64           `json:"threshold"`
	Duration    time.Duration     `json:"duration"`
	Severity    string            `json:"severity"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Enabled     bool              `json:"enabled"`
}

// BurnRateRule defines burn rate alerting conditions
type BurnRateRule struct {
	Name        string        `json:"name"`
	SLOTarget   float64       `json:"slo_target"`
	LongWindow  time.Duration `json:"long_window"`
	ShortWindow time.Duration `json:"short_window"`
	BurnFactor  float64       `json:"burn_factor"`
	Severity    string        `json:"severity"`
	ErrorBudget float64       `json:"error_budget"`
	Enabled     bool          `json:"enabled"`
}

// ActiveAlert represents an active alert condition
type ActiveAlert struct {
	ID          string            `json:"id"`
	Rule        *AlertRule        `json:"rule"`
	Status      AlertStatus       `json:"status"`
	StartsAt    time.Time         `json:"starts_at"`
	EndsAt      time.Time         `json:"ends_at,omitempty"`
	Value       float64           `json:"value"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Fingerprint string            `json:"fingerprint"`

	// Escalation tracking
	EscalationLevel int       `json:"escalation_level"`
	LastEscalated   time.Time `json:"last_escalated,omitempty"`

	// Notification tracking
	NotificationsSent map[string]time.Time `json:"notifications_sent"`
	SilencedUntil     time.Time            `json:"silenced_until,omitempty"`

	mu sync.RWMutex
}

// AlertStatus represents the current status of an alert
type AlertStatus string

const (
	AlertStatusFiring   AlertStatus = "firing"
	AlertStatusResolved AlertStatus = "resolved"
	AlertStatusSilenced AlertStatus = "silenced"
	AlertStatusPending  AlertStatus = "pending"
)

// AlertManagerMetrics contains Prometheus metrics for the alert manager
type AlertManagerMetrics struct {
	// Alert generation metrics
	AlertsGenerated   *prometheus.CounterVec
	AlertsResolved    *prometheus.CounterVec
	AlertDuration     *prometheus.HistogramVec
	ActiveAlertsGauge *prometheus.GaugeVec

	// Burn rate metrics
	BurnRateViolations *prometheus.CounterVec
	ErrorBudgetBurn    *prometheus.GaugeVec
	BurnRateLatency    prometheus.Histogram

	// Prediction metrics
	PredictedViolations *prometheus.CounterVec
	PredictionAccuracy  *prometheus.GaugeVec
	PredictionLatency   prometheus.Histogram

	// Notification metrics
	NotificationsSent   *prometheus.CounterVec
	NotificationLatency *prometheus.HistogramVec
	NotificationErrors  *prometheus.CounterVec

	// Performance metrics
	EvaluationLatency prometheus.Histogram
	EvaluationErrors  prometheus.Counter
	RuleEvaluations   *prometheus.CounterVec
}

// AlertConditionEvaluator evaluates alert conditions against current metrics
type AlertConditionEvaluator struct {
	rules       map[string]*AlertRule
	burnRates   []*BurnRateRule
	sliProvider SLIProvider
	mu          sync.RWMutex
}

// SLIProvider provides current SLI values for alert evaluation
type SLIProvider interface {
	GetCurrentAvailability() float64
	GetCurrentLatencyP95() time.Duration
	GetCurrentThroughput() float64
	GetCurrentErrorRate() float64
	GetErrorBudgetRemaining() float64
	GetBurnRate(window time.Duration) float64
}

// AlertHistory tracks historical alert data
type AlertHistory struct {
	alerts  []*HistoricalAlert
	maxSize int
	mu      sync.RWMutex
}

// HistoricalAlert represents a historical alert record
type HistoricalAlert struct {
	Alert      *ActiveAlert  `json:"alert"`
	ResolvedAt time.Time     `json:"resolved_at"`
	Duration   time.Duration `json:"duration"`
	MTTR       time.Duration `json:"mttr"` // Mean Time To Resolution
}

// SilenceManager manages alert silencing
type SilenceManager struct {
	silences map[string]*Silence
	mu       sync.RWMutex
}

// Silence represents an alert silence rule
type Silence struct {
	ID        string            `json:"id"`
	Matchers  map[string]string `json:"matchers"`
	StartsAt  time.Time         `json:"starts_at"`
	EndsAt    time.Time         `json:"ends_at"`
	CreatedBy string            `json:"created_by"`
	Comment   string            `json:"comment"`
}

// ViolationPredictor performs predictive analysis for SLA violations
type ViolationPredictor struct {
	models         map[string]*PredictionModel
	features       *FeatureExtractor
	confidenceCalc *ConfidenceCalculator
	enabled        bool
	horizon        time.Duration
	threshold      float64
	mu             sync.RWMutex
}

// PredictionModel represents a machine learning model for violation prediction
type PredictionModel struct {
	Name        string    `json:"name"`
	Type        string    `json:"type"` // linear, exponential, seasonal
	Accuracy    float64   `json:"accuracy"`
	LastTrained time.Time `json:"last_trained"`
	Features    []string  `json:"features"`
}

// TrendAnalyzer analyzes trends in SLI metrics
type TrendAnalyzer struct {
	windowSize time.Duration
	trendData  map[string]*TrendData
	mu         sync.RWMutex
}

// TrendData tracks trend information for a metric
type TrendData struct {
	Metric     string    `json:"metric"`
	Slope      float64   `json:"slope"`
	Direction  string    `json:"direction"` // increasing, decreasing, stable
	Strength   float64   `json:"strength"`  // 0.0-1.0
	LastUpdate time.Time `json:"last_update"`
}

// NotificationManager handles alert notifications
type NotificationManager struct {
	channels []NotificationChannel
	timeout  time.Duration
	mu       sync.RWMutex
}

// EscalationManager handles alert escalation
type EscalationManager struct {
	policies map[string]*EscalationPolicy
	mu       sync.RWMutex
}

// NewAlertManager creates a new SLA violation alert manager
func NewAlertManager(config *AlertManagerConfig, logger *logging.StructuredLogger) (*AlertManager, error) {
	if config == nil {
		config = DefaultAlertManagerConfig()
	}

	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	// Initialize metrics
	metrics := &AlertManagerMetrics{
		AlertsGenerated: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "sla_alertmanager_alerts_generated_total",
			Help: "Total number of alerts generated",
		}, []string{"rule", "severity", "slo_type"}),

		AlertsResolved: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "sla_alertmanager_alerts_resolved_total",
			Help: "Total number of alerts resolved",
		}, []string{"rule", "severity", "resolution_reason"}),

		AlertDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "sla_alertmanager_alert_duration_seconds",
			Help:    "Duration of active alerts",
			Buckets: prometheus.ExponentialBuckets(60, 2, 12), // 1min to ~68 hours
		}, []string{"rule", "severity"}),

		ActiveAlertsGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "sla_alertmanager_active_alerts",
			Help: "Number of currently active alerts",
		}, []string{"severity", "slo_type"}),

		BurnRateViolations: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "sla_alertmanager_burn_rate_violations_total",
			Help: "Total number of burn rate violations",
		}, []string{"window", "severity"}),

		ErrorBudgetBurn: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "sla_alertmanager_error_budget_burn_rate",
			Help: "Current error budget burn rate",
		}, []string{"window", "slo_type"}),

		BurnRateLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "sla_alertmanager_burn_rate_calculation_latency_seconds",
			Help:    "Latency of burn rate calculations",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
		}),

		PredictedViolations: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "sla_alertmanager_predicted_violations_total",
			Help: "Total number of predicted SLA violations",
		}, []string{"model", "confidence_level"}),

		PredictionAccuracy: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "sla_alertmanager_prediction_accuracy_percent",
			Help: "Accuracy of violation predictions",
		}, []string{"model", "time_horizon"}),

		PredictionLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "sla_alertmanager_prediction_latency_seconds",
			Help:    "Latency of violation prediction calculations",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 12),
		}),

		NotificationsSent: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "sla_alertmanager_notifications_sent_total",
			Help: "Total number of notifications sent",
		}, []string{"channel", "severity", "status"}),

		NotificationLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "sla_alertmanager_notification_latency_seconds",
			Help:    "Latency of notification delivery",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 10),
		}, []string{"channel"}),

		NotificationErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "sla_alertmanager_notification_errors_total",
			Help: "Total number of notification errors",
		}, []string{"channel", "error_type"}),

		EvaluationLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "sla_alertmanager_evaluation_latency_seconds",
			Help:    "Latency of alert rule evaluation",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 12),
		}),

		EvaluationErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "sla_alertmanager_evaluation_errors_total",
			Help: "Total number of alert evaluation errors",
		}),

		RuleEvaluations: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "sla_alertmanager_rule_evaluations_total",
			Help: "Total number of rule evaluations",
		}, []string{"rule", "result"}),
	}

	// Initialize components
	alertHistory := &AlertHistory{
		alerts:  make([]*HistoricalAlert, 0),
		maxSize: config.HistoryMaxSize,
	}

	silenceManager := &SilenceManager{
		silences: make(map[string]*Silence),
	}

	predictor := &ViolationPredictor{
		models:    make(map[string]*PredictionModel),
		features:  NewFeatureExtractor(),
		enabled:   config.PredictionEnabled,
		horizon:   config.PredictionHorizon,
		threshold: config.ConfidenceThreshold,
	}

	trendAnalyzer := &TrendAnalyzer{
		windowSize: 1 * time.Hour,
		trendData:  make(map[string]*TrendData),
	}

	notificationMgr := &NotificationManager{
		channels: config.NotificationChannels,
		timeout:  config.NotificationTimeout,
	}

	escalationMgr := &EscalationManager{
		policies: make(map[string]*EscalationPolicy),
	}

	// Initialize escalation policies
	for _, policy := range config.EscalationPolicies {
		policyCopy := policy
		escalationMgr.policies[policy.Name] = &policyCopy
	}

	// Initialize alert condition evaluator
	alertConditions := &AlertConditionEvaluator{
		rules:     make(map[string]*AlertRule),
		burnRates: make([]*BurnRateRule, 0),
	}

	// Initialize default alert rules
	defaultRules := createDefaultAlertRules(config)
	for _, rule := range defaultRules {
		alertConditions.rules[rule.Name] = rule
	}

	// Initialize burn rate rules
	burnRateRules := createBurnRateRules(config)
	alertConditions.burnRates = burnRateRules

	alertManager := &AlertManager{
		config:          config,
		logger:          logger.WithComponent("alert-manager"),
		alertRules:      alertConditions.rules,
		burnRateRules:   burnRateRules,
		alertConditions: alertConditions,
		activeAlerts:    make(map[string]*ActiveAlert),
		alertHistory:    alertHistory,
		silences:        silenceManager,
		predictor:       predictor,
		trendAnalyzer:   trendAnalyzer,
		notificationMgr: notificationMgr,
		escalationMgr:   escalationMgr,
		metrics:         metrics,
		stopCh:          make(chan struct{}),
	}

	return alertManager, nil
}

// Start begins the alert manager
func (am *AlertManager) Start(ctx context.Context) error {
	if am.started.Load() {
		return fmt.Errorf("alert manager already started")
	}

	am.logger.InfoWithContext("Starting SLA alert manager",
		"evaluation_interval", am.config.EvaluationInterval,
		"max_pending_alerts", am.config.MaxPendingAlerts,
		"prediction_enabled", am.config.PredictionEnabled,
	)

	// Initialize tickers
	am.evaluationTicker = time.NewTicker(am.config.EvaluationInterval)
	am.cleanupTicker = time.NewTicker(1 * time.Hour)

	// Start background processes
	am.wg.Add(1)
	go am.runEvaluationLoop(ctx)

	am.wg.Add(1)
	go am.runCleanupLoop(ctx)

	am.wg.Add(1)
	go am.runEscalationLoop(ctx)

	if am.config.PredictionEnabled {
		am.wg.Add(1)
		go am.runPredictionLoop(ctx)
	}

	am.started.Store(true)
	am.logger.InfoWithContext("SLA alert manager started successfully")

	return nil
}

// Stop gracefully stops the alert manager
func (am *AlertManager) Stop(ctx context.Context) error {
	if !am.started.Load() {
		return nil
	}

	am.logger.InfoWithContext("Stopping SLA alert manager")

	// Stop tickers
	if am.evaluationTicker != nil {
		am.evaluationTicker.Stop()
	}
	if am.cleanupTicker != nil {
		am.cleanupTicker.Stop()
	}

	// Signal stop
	close(am.stopCh)

	// Wait for goroutines
	am.wg.Wait()

	am.logger.InfoWithContext("SLA alert manager stopped")

	return nil
}

// EvaluateAlerts evaluates all alert rules against current SLI values
func (am *AlertManager) EvaluateAlerts(sliProvider SLIProvider) error {
	start := time.Now()
	defer func() {
		am.metrics.EvaluationLatency.Observe(time.Since(start).Seconds())
	}()

	am.alertConditions.sliProvider = sliProvider

	// Evaluate standard alert rules
	for _, rule := range am.alertRules {
		if !rule.Enabled {
			continue
		}

		if err := am.evaluateRule(rule, sliProvider); err != nil {
			am.logger.ErrorWithContext("Failed to evaluate alert rule",
				err,
				"rule", rule.Name,
			)
			am.metrics.EvaluationErrors.Inc()
		}
	}

	// Evaluate burn rate rules
	for _, burnRule := range am.burnRateRules {
		if !burnRule.Enabled {
			continue
		}

		if err := am.evaluateBurnRateRule(burnRule, sliProvider); err != nil {
			am.logger.ErrorWithContext("Failed to evaluate burn rate rule",
				err,
				"rule", burnRule.Name,
			)
		}
	}

	return nil
}

// SetSLIProvider sets the SLI provider for alert evaluation
func (am *AlertManager) SetSLIProvider(provider SLIProvider) {
	am.alertConditions.mu.Lock()
	defer am.alertConditions.mu.Unlock()
	am.alertConditions.sliProvider = provider
}

// GetActiveAlerts returns current active alerts
func (am *AlertManager) GetActiveAlerts() []*ActiveAlert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	alerts := make([]*ActiveAlert, 0, len(am.activeAlerts))
	for _, alert := range am.activeAlerts {
		alert.mu.RLock()
		
		// Deep copy the alert without copying the mutex
		alertCopy := &ActiveAlert{
			ID:                alert.ID,
			Rule:              alert.Rule,
			Status:            alert.Status,
			StartsAt:          alert.StartsAt,
			EndsAt:            alert.EndsAt,
			Value:             alert.Value,
			Labels:            make(map[string]string),
			Annotations:       make(map[string]string),
			Fingerprint:       alert.Fingerprint,
			EscalationLevel:   alert.EscalationLevel,
			LastEscalated:     alert.LastEscalated,
			NotificationsSent: make(map[string]time.Time),
			SilencedUntil:     alert.SilencedUntil,
		}
		
		// Copy maps
		for k, v := range alert.Labels {
			alertCopy.Labels[k] = v
		}
		for k, v := range alert.Annotations {
			alertCopy.Annotations[k] = v
		}
		for k, v := range alert.NotificationsSent {
			alertCopy.NotificationsSent[k] = v
		}
		
		alert.mu.RUnlock()
		alerts = append(alerts, alertCopy)
	}

	return alerts
}

// SilenceAlert creates a silence for matching alerts
func (am *AlertManager) SilenceAlert(matchers map[string]string, duration time.Duration, createdBy, comment string) (string, error) {
	silenceID := fmt.Sprintf("silence-%d", time.Now().UnixNano())

	silence := &Silence{
		ID:        silenceID,
		Matchers:  matchers,
		StartsAt:  time.Now(),
		EndsAt:    time.Now().Add(duration),
		CreatedBy: createdBy,
		Comment:   comment,
	}

	am.silences.mu.Lock()
	am.silences.silences[silenceID] = silence
	am.silences.mu.Unlock()

	am.logger.InfoWithContext("Alert silence created",
		"silence_id", silenceID,
		"duration", duration,
		"created_by", createdBy,
		"matchers", matchers,
	)

	return silenceID, nil
}

// runEvaluationLoop runs the main alert evaluation loop
func (am *AlertManager) runEvaluationLoop(ctx context.Context) {
	defer am.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-am.stopCh:
			return
		case <-am.evaluationTicker.C:
			if am.alertConditions.sliProvider != nil {
				am.EvaluateAlerts(am.alertConditions.sliProvider)
			}
		}
	}
}

// runCleanupLoop performs periodic cleanup of resolved alerts
func (am *AlertManager) runCleanupLoop(ctx context.Context) {
	defer am.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-am.stopCh:
			return
		case <-am.cleanupTicker.C:
			am.performCleanup()
		}
	}
}

// runEscalationLoop handles alert escalation
func (am *AlertManager) runEscalationLoop(ctx context.Context) {
	defer am.wg.Done()

	escalationTicker := time.NewTicker(1 * time.Minute)
	defer escalationTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-am.stopCh:
			return
		case <-escalationTicker.C:
			am.processEscalations()
		}
	}
}

// runPredictionLoop runs predictive violation analysis
func (am *AlertManager) runPredictionLoop(ctx context.Context) {
	defer am.wg.Done()

	predictionTicker := time.NewTicker(5 * time.Minute)
	defer predictionTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-am.stopCh:
			return
		case <-predictionTicker.C:
			if am.alertConditions.sliProvider != nil {
				am.performPredictiveAnalysis()
			}
		}
	}
}

// evaluateRule evaluates a single alert rule
func (am *AlertManager) evaluateRule(rule *AlertRule, sliProvider SLIProvider) error {
	_ = time.Now()
	defer func() {
		am.metrics.RuleEvaluations.WithLabelValues(rule.Name, "completed").Inc()
	}()

	// Get current metric value
	var currentValue float64
	switch rule.Metric {
	case "availability":
		currentValue = sliProvider.GetCurrentAvailability()
	case "latency_p95":
		currentValue = float64(sliProvider.GetCurrentLatencyP95().Milliseconds())
	case "throughput":
		currentValue = sliProvider.GetCurrentThroughput()
	case "error_rate":
		currentValue = sliProvider.GetCurrentErrorRate()
	default:
		return fmt.Errorf("unsupported metric: %s", rule.Metric)
	}

	// Check condition
	conditionMet := am.evaluateCondition(rule.Condition, currentValue, rule.Threshold)

	if conditionMet {
		am.handleAlertFiring(rule, currentValue)
	} else {
		am.handleAlertResolution(rule.Name)
	}

	return nil
}

// evaluateBurnRateRule evaluates a burn rate rule
func (am *AlertManager) evaluateBurnRateRule(rule *BurnRateRule, sliProvider SLIProvider) error {
	start := time.Now()
	defer func() {
		am.metrics.BurnRateLatency.Observe(time.Since(start).Seconds())
	}()

	// Calculate burn rates for long and short windows
	longBurnRate := sliProvider.GetBurnRate(rule.LongWindow)
	shortBurnRate := sliProvider.GetBurnRate(rule.ShortWindow)

	// Check if both windows exceed the burn rate threshold
	threshold := rule.BurnFactor * (100.0 - rule.SLOTarget) / 100.0

	longExceeded := longBurnRate > threshold
	shortExceeded := shortBurnRate > threshold

	if longExceeded && shortExceeded {
		am.handleBurnRateViolation(rule, longBurnRate, shortBurnRate)
	}

	// Update metrics
	am.metrics.ErrorBudgetBurn.WithLabelValues("long", rule.Name).Set(longBurnRate)
	am.metrics.ErrorBudgetBurn.WithLabelValues("short", rule.Name).Set(shortBurnRate)

	return nil
}

// evaluateCondition evaluates an alert condition
func (am *AlertManager) evaluateCondition(condition string, value, threshold float64) bool {
	switch condition {
	case "gt":
		return value > threshold
	case "lt":
		return value < threshold
	case "eq":
		return value == threshold
	case "ne":
		return value != threshold
	case "gte":
		return value >= threshold
	case "lte":
		return value <= threshold
	default:
		return false
	}
}

// handleAlertFiring creates or updates a firing alert
func (am *AlertManager) handleAlertFiring(rule *AlertRule, value float64) {
	am.mu.Lock()
	defer am.mu.Unlock()

	fingerprint := am.generateFingerprint(rule, rule.Labels)

	alert, exists := am.activeAlerts[fingerprint]
	if !exists {
		// Create new alert
		alert = &ActiveAlert{
			ID:                fmt.Sprintf("alert-%d", time.Now().UnixNano()),
			Rule:              rule,
			Status:            AlertStatusFiring,
			StartsAt:          time.Now(),
			Value:             value,
			Labels:            rule.Labels,
			Annotations:       rule.Annotations,
			Fingerprint:       fingerprint,
			NotificationsSent: make(map[string]time.Time),
		}

		am.activeAlerts[fingerprint] = alert

		// Send notification
		am.sendNotification(alert)

		// Update metrics
		am.alertsGenerated.Add(1)
		am.metrics.AlertsGenerated.WithLabelValues(rule.Name, rule.Severity, rule.Metric).Inc()
		am.metrics.ActiveAlertsGauge.WithLabelValues(rule.Severity, rule.Metric).Inc()

		am.logger.WarnWithContext("Alert firing",
			"alert_id", alert.ID,
			"rule", rule.Name,
			"value", value,
			"threshold", rule.Threshold,
		)
	} else {
		// Update existing alert
		alert.mu.Lock()
		alert.Value = value
		alert.mu.Unlock()
	}
}

// handleAlertResolution resolves an active alert
func (am *AlertManager) handleAlertResolution(ruleName string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	for fingerprint, alert := range am.activeAlerts {
		if alert.Rule.Name == ruleName && alert.Status == AlertStatusFiring {
			alert.mu.Lock()
			alert.Status = AlertStatusResolved
			alert.EndsAt = time.Now()
			duration := alert.EndsAt.Sub(alert.StartsAt)
			alert.mu.Unlock()

			// Move to history
			am.moveToHistory(alert)

			// Remove from active alerts
			delete(am.activeAlerts, fingerprint)

			// Update metrics
			am.alertsResolved.Add(1)
			am.metrics.AlertsResolved.WithLabelValues(alert.Rule.Name, alert.Rule.Severity, "condition_resolved").Inc()
			am.metrics.AlertDuration.WithLabelValues(alert.Rule.Name, alert.Rule.Severity).Observe(duration.Seconds())
			am.metrics.ActiveAlertsGauge.WithLabelValues(alert.Rule.Severity, alert.Rule.Metric).Dec()

			am.logger.InfoWithContext("Alert resolved",
				"alert_id", alert.ID,
				"rule", alert.Rule.Name,
				"duration", duration,
			)

			break
		}
	}
}

// handleBurnRateViolation handles burn rate violations
func (am *AlertManager) handleBurnRateViolation(rule *BurnRateRule, longRate, shortRate float64) {
	fingerprint := fmt.Sprintf("burn-rate-%s", rule.Name)

	am.mu.Lock()
	defer am.mu.Unlock()

	_, exists := am.activeAlerts[fingerprint]
	if !exists {
		// Create burn rate alert
		alert := &ActiveAlert{
			ID:          fmt.Sprintf("burn-rate-%d", time.Now().UnixNano()),
			Status:      AlertStatusFiring,
			StartsAt:    time.Now(),
			Value:       longRate,
			Fingerprint: fingerprint,
			Labels: map[string]string{
				"alertname":  "BurnRateViolation",
				"rule":       rule.Name,
				"severity":   rule.Severity,
				"long_rate":  fmt.Sprintf("%.2f", longRate),
				"short_rate": fmt.Sprintf("%.2f", shortRate),
			},
			Annotations: map[string]string{
				"summary":     fmt.Sprintf("Error budget burn rate violation for %s", rule.Name),
				"description": fmt.Sprintf("Long window burn rate: %.2f, Short window burn rate: %.2f", longRate, shortRate),
			},
			NotificationsSent: make(map[string]time.Time),
		}

		am.activeAlerts[fingerprint] = alert

		// Send notification
		am.sendNotification(alert)

		// Update metrics
		am.metrics.BurnRateViolations.WithLabelValues(rule.Name, rule.Severity).Inc()

		am.logger.WarnWithContext("Burn rate violation detected",
			"rule", rule.Name,
			"long_rate", longRate,
			"short_rate", shortRate,
			"severity", rule.Severity,
		)
	}
}

// sendNotification sends a notification for an alert
func (am *AlertManager) sendNotification(alert *ActiveAlert) {
	// Check if alert is silenced
	if am.isAlertSilenced(alert) {
		return
	}

	for _, channel := range am.notificationMgr.channels {
		if !channel.Enabled {
			continue
		}

		start := time.Now()

		err := am.deliverNotification(channel, alert)

		latency := time.Since(start)
		am.metrics.NotificationLatency.WithLabelValues(channel.Type).Observe(latency.Seconds())

		status := "success"
		if err != nil {
			status = "error"
			am.metrics.NotificationErrors.WithLabelValues(channel.Type, "delivery_failed").Inc()
			am.logger.ErrorWithContext("Failed to send notification",
				err,
				"channel", channel.Type,
				"alert_id", alert.ID,
			)
		} else {
			alert.mu.Lock()
			alert.NotificationsSent[channel.Type] = time.Now()
			alert.mu.Unlock()
		}

		am.metrics.NotificationsSent.WithLabelValues(channel.Type, alert.Rule.Severity, status).Inc()
	}
}

// deliverNotification delivers a notification to a specific channel
func (am *AlertManager) deliverNotification(channel NotificationChannel, alert *ActiveAlert) error {
	switch channel.Type {
	case "webhook":
		return am.sendWebhookNotification(channel.URL, alert)
	case "log":
		return am.sendLogNotification(alert)
	default:
		return fmt.Errorf("unsupported notification channel: %s", channel.Type)
	}
}

// sendWebhookNotification sends a webhook notification
func (am *AlertManager) sendWebhookNotification(url string, alert *ActiveAlert) error {
	// Placeholder implementation - would send HTTP POST to webhook URL
	am.logger.InfoWithContext("Webhook notification sent",
		"url", url,
		"alert_id", alert.ID,
		"severity", alert.Rule.Severity,
	)
	return nil
}

// sendLogNotification logs the alert notification
func (am *AlertManager) sendLogNotification(alert *ActiveAlert) error {
	am.logger.WarnWithContext("SLA Alert Notification",
		"alert_id", alert.ID,
		"rule", alert.Rule.Name,
		"severity", alert.Rule.Severity,
		"value", alert.Value,
		"labels", alert.Labels,
		"annotations", alert.Annotations,
	)
	return nil
}

// isAlertSilenced checks if an alert is currently silenced
func (am *AlertManager) isAlertSilenced(alert *ActiveAlert) bool {
	am.silences.mu.RLock()
	defer am.silences.mu.RUnlock()

	now := time.Now()

	for _, silence := range am.silences.silences {
		if silence.EndsAt.Before(now) {
			continue // Silence expired
		}

		// Check if alert matches silence matchers
		if am.matchesMatchers(alert.Labels, silence.Matchers) {
			alert.mu.Lock()
			alert.SilencedUntil = silence.EndsAt
			alert.mu.Unlock()
			return true
		}
	}

	return false
}

// matchesMatchers checks if alert labels match silence matchers
func (am *AlertManager) matchesMatchers(labels, matchers map[string]string) bool {
	for key, value := range matchers {
		if labels[key] != value {
			return false
		}
	}
	return true
}

// performCleanup removes old alerts and silences
func (am *AlertManager) performCleanup() {
	now := time.Now()
	cutoff := now.Add(-am.config.AlertRetentionPeriod)

	// Clean up expired silences
	am.silences.mu.Lock()
	for id, silence := range am.silences.silences {
		if silence.EndsAt.Before(now) {
			delete(am.silences.silences, id)
		}
	}
	am.silences.mu.Unlock()

	// Clean up old alert history
	am.alertHistory.mu.Lock()
	cleanedAlerts := make([]*HistoricalAlert, 0)
	for _, alert := range am.alertHistory.alerts {
		if alert.ResolvedAt.After(cutoff) {
			cleanedAlerts = append(cleanedAlerts, alert)
		}
	}
	am.alertHistory.alerts = cleanedAlerts
	am.alertHistory.mu.Unlock()
}

// processEscalations handles alert escalation based on policies
func (am *AlertManager) processEscalations() {
	am.mu.RLock()
	defer am.mu.RUnlock()

	now := time.Now()

	for _, alert := range am.activeAlerts {
		if alert.Status != AlertStatusFiring {
			continue
		}

		alert.mu.RLock()
		alertAge := now.Sub(alert.StartsAt)
		currentLevel := alert.EscalationLevel
		alert.mu.RUnlock()

		// Find applicable escalation policy
		policy := am.escalationMgr.policies["default"] // Use default policy
		if policy != nil && policy.Enabled {
			for i, level := range policy.Levels {
				if i > currentLevel && alertAge >= level.Duration {
					am.escalateAlert(alert, i, level)
					break
				}
			}
		}
	}
}

// escalateAlert escalates an alert to a higher level
func (am *AlertManager) escalateAlert(alert *ActiveAlert, level int, escalationLevel EscalationLevel) {
	alert.mu.Lock()
	alert.EscalationLevel = level
	alert.LastEscalated = time.Now()

	// Update severity if specified
	if escalationLevel.Severity != "" {
		alert.Rule.Severity = escalationLevel.Severity
	}
	alert.mu.Unlock()

	am.logger.WarnWithContext("Alert escalated",
		"alert_id", alert.ID,
		"new_level", level,
		"new_severity", escalationLevel.Severity,
		"age", time.Since(alert.StartsAt),
	)

	// Send escalation notification
	am.sendNotification(alert)
}

// performPredictiveAnalysis performs predictive violation analysis
func (am *AlertManager) performPredictiveAnalysis() {
	start := time.Now()
	defer func() {
		am.metrics.PredictionLatency.Observe(time.Since(start).Seconds())
	}()

	if !am.predictor.enabled {
		return
	}

	// This would implement sophisticated ML-based prediction
	// For now, we'll use trend analysis
	predictions := am.trendAnalyzer.analyzeViolationRisk(am.alertConditions.sliProvider)

	for _, prediction := range predictions {
		if prediction.Confidence >= am.predictor.threshold {
			am.handlePredictedViolation(prediction)
		}
	}
}

// handlePredictedViolation handles a predicted SLA violation
func (am *AlertManager) handlePredictedViolation(prediction *ViolationPrediction) {
	confidenceLevel := "high"
	if prediction.Confidence < 0.9 {
		confidenceLevel = "medium"
	}
	if prediction.Confidence < 0.7 {
		confidenceLevel = "low"
	}

	am.metrics.PredictedViolations.WithLabelValues(prediction.Model, confidenceLevel).Inc()

	am.logger.WarnWithContext("Predicted SLA violation",
		"violation_type", prediction.Type,
		"predicted_time", prediction.PredictedTime,
		"confidence", prediction.Confidence,
		"model", prediction.Model,
	)
}

// generateFingerprint generates a unique fingerprint for an alert
func (am *AlertManager) generateFingerprint(rule *AlertRule, labels map[string]string) string {
	fingerprint := rule.Name
	for key, value := range labels {
		fingerprint += fmt.Sprintf(":%s=%s", key, value)
	}
	return fingerprint
}

// moveToHistory moves an alert to historical storage
func (am *AlertManager) moveToHistory(alert *ActiveAlert) {
	historicalAlert := &HistoricalAlert{
		Alert:      alert,
		ResolvedAt: alert.EndsAt,
		Duration:   alert.EndsAt.Sub(alert.StartsAt),
		MTTR:       alert.EndsAt.Sub(alert.StartsAt), // Simplified MTTR calculation
	}

	am.alertHistory.mu.Lock()
	defer am.alertHistory.mu.Unlock()

	am.alertHistory.alerts = append(am.alertHistory.alerts, historicalAlert)

	// Trim history if too large
	if len(am.alertHistory.alerts) > am.alertHistory.maxSize {
		am.alertHistory.alerts = am.alertHistory.alerts[1:]
	}
}

// Helper functions and placeholder implementations

func createDefaultAlertRules(config *AlertManagerConfig) []*AlertRule {
	return []*AlertRule{
		{
			Name:        "AvailabilityViolation",
			Description: "Availability SLO violation",
			Metric:      "availability",
			Condition:   "lt",
			Threshold:   99.95,
			Duration:    2 * time.Minute,
			Severity:    "critical",
			Labels:      map[string]string{"alertname": "AvailabilityViolation"},
			Annotations: map[string]string{"summary": "Availability below SLO target"},
			Enabled:     true,
		},
		{
			Name:        "LatencyViolation",
			Description: "P95 latency SLO violation",
			Metric:      "latency_p95",
			Condition:   "gt",
			Threshold:   2000, // 2 seconds in milliseconds
			Duration:    1 * time.Minute,
			Severity:    "warning",
			Labels:      map[string]string{"alertname": "LatencyViolation"},
			Annotations: map[string]string{"summary": "P95 latency above SLO target"},
			Enabled:     true,
		},
		{
			Name:        "ErrorRateViolation",
			Description: "Error rate SLO violation",
			Metric:      "error_rate",
			Condition:   "gt",
			Threshold:   0.1,
			Duration:    5 * time.Minute,
			Severity:    "warning",
			Labels:      map[string]string{"alertname": "ErrorRateViolation"},
			Annotations: map[string]string{"summary": "Error rate above SLO target"},
			Enabled:     true,
		},
	}
}

func createBurnRateRules(config *AlertManagerConfig) []*BurnRateRule {
	rules := make([]*BurnRateRule, 0)

	for _, window := range config.BurnRateWindows {
		rule := &BurnRateRule{
			Name:        fmt.Sprintf("BurnRate_%s", window.Name),
			SLOTarget:   99.95,
			LongWindow:  window.LongWindow,
			ShortWindow: window.ShortWindow,
			BurnFactor:  window.Factor,
			Severity:    window.Severity,
			ErrorBudget: 100.0 - 99.95,
			Enabled:     true,
		}
		rules = append(rules, rule)
	}

	return rules
}

// Placeholder types and functions for predictive analysis
type ViolationPrediction struct {
	Type          string    `json:"type"`
	PredictedTime time.Time `json:"predicted_time"`
	Confidence    float64   `json:"confidence"`
	Model         string    `json:"model"`
}

type FeatureExtractor struct{}
type ConfidenceCalculator struct{}

func NewFeatureExtractor() *FeatureExtractor { return &FeatureExtractor{} }

func (ta *TrendAnalyzer) analyzeViolationRisk(sliProvider SLIProvider) []*ViolationPrediction {
	// Placeholder implementation
	return []*ViolationPrediction{}
}
