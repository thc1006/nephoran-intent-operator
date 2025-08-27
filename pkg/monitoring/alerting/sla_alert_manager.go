// Package alerting provides comprehensive SLA violation alerting for the Nephoran Intent Operator.
// This system provides intelligent early warning, reduces alert fatigue, and enables rapid incident response
// through multi-window burn rate alerting, predictive violation detection, and intelligent escalation.
package alerting

import (
	"context"
	"crypto/md5"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sony/gobreaker"
	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// SLAType represents different SLA metrics we monitor
type SLAType string

const (
	SLATypeAvailability SLAType = "availability"
	SLATypeLatency      SLAType = "latency"
	SLAThroughput       SLAType = "throughput"
	SLAErrorRate        SLAType = "error_rate"
)

// AlertSeverity represents alert severity levels aligned with SRE practices
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityMajor    AlertSeverity = "major"
	AlertSeverityCritical AlertSeverity = "critical"
	AlertSeverityUrgent   AlertSeverity = "urgent"
)

// AlertState represents the lifecycle state of an alert
type AlertState string

const (
	AlertStateFiring       AlertState = "firing"
	AlertStatePending      AlertState = "pending"
	AlertStateResolved     AlertState = "resolved"
	AlertStateSuppressed   AlertState = "suppressed"
	AlertStateAcknowledged AlertState = "acknowledged"
)

// SLAAlert represents an SLA violation alert with enriched context
type SLAAlert struct {
	ID          string        `json:"id"`
	SLAType     SLAType       `json:"sla_type"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Severity    AlertSeverity `json:"severity"`
	State       AlertState    `json:"state"`

	// SLA-specific fields
	SLATarget    float64         `json:"sla_target"`
	CurrentValue float64         `json:"current_value"`
	Threshold    float64         `json:"threshold"`
	ErrorBudget  ErrorBudgetInfo `json:"error_budget"`
	BurnRate     BurnRateInfo    `json:"burn_rate"`

	// Metadata
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Context     AlertContext      `json:"context"`

	// Timing
	StartsAt       time.Time  `json:"starts_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	EndsAt         *time.Time `json:"ends_at,omitempty"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`

	// Processing
	Fingerprint string `json:"fingerprint"`
	Hash        string `json:"hash"`

	// Business impact
	BusinessImpact BusinessImpactInfo `json:"business_impact"`

	// Runbook information
	RunbookURL   string   `json:"runbook_url,omitempty"`
	RunbookSteps []string `json:"runbook_steps,omitempty"`
}

// ErrorBudgetInfo contains error budget consumption details
type ErrorBudgetInfo struct {
	Total            float64        `json:"total"`
	Remaining        float64        `json:"remaining"`
	Consumed         float64        `json:"consumed"`
	ConsumedPercent  float64        `json:"consumed_percent"`
	TimeToExhaustion *time.Duration `json:"time_to_exhaustion,omitempty"`
}

// BurnRateInfo contains burn rate analysis
type BurnRateInfo struct {
	ShortWindow   BurnRateWindow `json:"short_window"`
	MediumWindow  BurnRateWindow `json:"medium_window"`
	LongWindow    BurnRateWindow `json:"long_window"`
	CurrentRate   float64        `json:"current_rate"`
	PredictedRate float64        `json:"predicted_rate"`
}

// BurnRateWindow represents a specific burn rate measurement window
type BurnRateWindow struct {
	Duration    time.Duration `json:"duration"`
	BurnRate    float64       `json:"burn_rate"`
	Threshold   float64       `json:"threshold"`
	IsViolating bool          `json:"is_violating"`
}

// AlertContext provides enriched context for the alert
type AlertContext struct {
	Component       string            `json:"component"`
	Service         string            `json:"service"`
	Region          string            `json:"region"`
	Environment     string            `json:"environment"`
	RelatedMetrics  []MetricSnapshot  `json:"related_metrics"`
	RecentIncidents []IncidentSummary `json:"recent_incidents"`
	DashboardLinks  []string          `json:"dashboard_links"`
	LogQueries      []LogQuery        `json:"log_queries"`
}

// MetricSnapshot provides related metric context
type MetricSnapshot struct {
	Name      string    `json:"name"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
	Timestamp time.Time `json:"timestamp"`
}

// IncidentSummary provides context about related incidents
type IncidentSummary struct {
	ID         string     `json:"id"`
	Title      string     `json:"title"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

// LogQuery provides relevant log query context
type LogQuery struct {
	Description string `json:"description"`
	Query       string `json:"query"`
	Source      string `json:"source"`
}

// BusinessImpactInfo provides business context for the alert
type BusinessImpactInfo struct {
	Severity       string  `json:"severity"`
	AffectedUsers  int64   `json:"affected_users"`
	RevenueImpact  float64 `json:"revenue_impact"`
	SLABreach      bool    `json:"sla_breach"`
	CustomerFacing bool    `json:"customer_facing"`
	ServiceTier    string  `json:"service_tier"`
}

// SLAAlertManager provides comprehensive SLA violation alerting with intelligent
// early warning, noise reduction, and rapid incident response capabilities
type SLAAlertManager struct {
	// Core components
	logger             *logging.StructuredLogger
	burnRateCalculator *BurnRateCalculator
	predictiveAlerting *PredictiveAlerting
	alertRouter        *AlertRouter
	escalationEngine   *EscalationEngine

	// Configuration
	config     *SLAAlertConfig
	slaTargets map[SLAType]SLATarget

	// State management
	activeAlerts       map[string]*SLAAlert
	suppressedAlerts   map[string]*SLAAlert
	alertHistory       []*SLAAlert
	maintenanceWindows []*MaintenanceWindow

	// External integrations
	prometheusClient v1.API
	circuitBreaker   *gobreaker.CircuitBreaker

	// Metrics
	metrics *SLAAlertMetrics

	// Synchronization
	mu      sync.RWMutex
	started bool
	stopCh  chan struct{}
}

// SLAAlertConfig holds configuration for the SLA alert manager
type SLAAlertConfig struct {
	// Evaluation intervals
	EvaluationInterval time.Duration `yaml:"evaluation_interval"`
	BurnRateInterval   time.Duration `yaml:"burn_rate_interval"`
	PredictiveInterval time.Duration `yaml:"predictive_interval"`

	// Alert management
	MaxActiveAlerts     int           `yaml:"max_active_alerts"`
	AlertRetention      time.Duration `yaml:"alert_retention"`
	DeduplicationWindow time.Duration `yaml:"deduplication_window"`
	CorrelationWindow   time.Duration `yaml:"correlation_window"`

	// Notification settings
	NotificationTimeout time.Duration `yaml:"notification_timeout"`
	MaxNotificationRate int           `yaml:"max_notification_rate"`
	SuppressionEnabled  bool          `yaml:"suppression_enabled"`

	// Business context
	EnableBusinessImpact bool    `yaml:"enable_business_impact"`
	RevenuePerUser       float64 `yaml:"revenue_per_user"`

	// Integration settings
	PrometheusURL  string `yaml:"prometheus_url"`
	GrafanaURL     string `yaml:"grafana_url"`
	RunbookBaseURL string `yaml:"runbook_base_url"`

	// Performance settings
	MetricCacheSize int           `yaml:"metric_cache_size"`
	QueryTimeout    time.Duration `yaml:"query_timeout"`
}

// SLATarget defines target values and thresholds for an SLA
type SLATarget struct {
	Target         float64       `yaml:"target"`          // SLA target (e.g., 99.95%)
	ErrorBudget    float64       `yaml:"error_budget"`    // Error budget period
	Windows        []AlertWindow `yaml:"windows"`         // Multi-window thresholds
	BusinessTier   string        `yaml:"business_tier"`   // Business criticality
	CustomerFacing bool          `yaml:"customer_facing"` // Customer impact
}

// AlertWindow defines a specific alerting window with thresholds
type AlertWindow struct {
	ShortWindow time.Duration `yaml:"short_window"`
	LongWindow  time.Duration `yaml:"long_window"`
	BurnRate    float64       `yaml:"burn_rate"`
	Severity    AlertSeverity `yaml:"severity"`
}

// MaintenanceWindow defines a period during which alerts should be suppressed
type MaintenanceWindow struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	StartTime  time.Time         `json:"start_time"`
	EndTime    time.Time         `json:"end_time"`
	Services   []string          `json:"services"`
	Components []string          `json:"components"`
	Labels     map[string]string `json:"labels"`
	CreatedBy  string            `json:"created_by"`
}

// SLAAlertMetrics contains Prometheus metrics for the SLA alert manager
type SLAAlertMetrics struct {
	// Alert generation metrics
	AlertsGenerated     *prometheus.CounterVec
	AlertsResolved      *prometheus.CounterVec
	AlertsFalsePositive *prometheus.CounterVec
	AlertsAcknowledged  *prometheus.CounterVec

	// Alert quality metrics
	AlertPrecision  *prometheus.GaugeVec
	AlertRecall     *prometheus.GaugeVec
	MeanTimeToAlert *prometheus.HistogramVec
	MeanTimeToAck   *prometheus.HistogramVec

	// SLA compliance metrics
	SLACompliance      *prometheus.GaugeVec
	ErrorBudgetBurn    *prometheus.GaugeVec
	BurnRateViolations *prometheus.CounterVec

	// System metrics
	ActiveAlerts        *prometheus.GaugeVec
	AlertProcessingTime *prometheus.HistogramVec
	NotificationsSent   *prometheus.CounterVec
	NotificationsFailed *prometheus.CounterVec

	// Business metrics
	BusinessImpactScore *prometheus.GaugeVec
	CustomerImpact      *prometheus.GaugeVec
	RevenueAtRisk       prometheus.Gauge
}

// DefaultSLAAlertConfig returns production-ready configuration
func DefaultSLAAlertConfig() *SLAAlertConfig {
	return &SLAAlertConfig{
		// Evaluation intervals optimized for rapid detection
		EvaluationInterval: 30 * time.Second,
		BurnRateInterval:   10 * time.Second,
		PredictiveInterval: 60 * time.Second,

		// Alert management settings
		MaxActiveAlerts:     1000,
		AlertRetention:      7 * 24 * time.Hour,
		DeduplicationWindow: 5 * time.Minute,
		CorrelationWindow:   10 * time.Minute,

		// Notification settings
		NotificationTimeout: 30 * time.Second,
		MaxNotificationRate: 100,
		SuppressionEnabled:  true,

		// Business context
		EnableBusinessImpact: true,
		RevenuePerUser:       10.0,

		// Performance settings
		MetricCacheSize: 1000,
		QueryTimeout:    10 * time.Second,
	}
}

// DefaultSLATargets returns production SLA targets for Nephoran Intent Operator
func DefaultSLATargets() map[SLAType]SLATarget {
	return map[SLAType]SLATarget{
		SLATypeAvailability: {
			Target:         99.95,
			ErrorBudget:    4.32, // minutes per month for 99.95%
			BusinessTier:   "critical",
			CustomerFacing: true,
			Windows: []AlertWindow{
				{
					ShortWindow: 1 * time.Hour,
					LongWindow:  5 * time.Minute,
					BurnRate:    14.4, // Exhausts budget in 2 hours
					Severity:    AlertSeverityUrgent,
				},
				{
					ShortWindow: 6 * time.Hour,
					LongWindow:  30 * time.Minute,
					BurnRate:    6, // Exhausts budget in 5 hours
					Severity:    AlertSeverityCritical,
				},
				{
					ShortWindow: 24 * time.Hour,
					LongWindow:  2 * time.Hour,
					BurnRate:    3, // Exhausts budget in 1 day
					Severity:    AlertSeverityMajor,
				},
			},
		},
		SLATypeLatency: {
			Target:         2000, // 2 seconds P95
			ErrorBudget:    5.0,  // 5% budget for latency violations
			BusinessTier:   "high",
			CustomerFacing: true,
			Windows: []AlertWindow{
				{
					ShortWindow: 15 * time.Minute,
					LongWindow:  2 * time.Minute,
					BurnRate:    10,
					Severity:    AlertSeverityCritical,
				},
				{
					ShortWindow: 1 * time.Hour,
					LongWindow:  10 * time.Minute,
					BurnRate:    5,
					Severity:    AlertSeverityMajor,
				},
			},
		},
		SLAThroughput: {
			Target:         45, // 45 intents per minute
			ErrorBudget:    10.0,
			BusinessTier:   "medium",
			CustomerFacing: false,
			Windows: []AlertWindow{
				{
					ShortWindow: 10 * time.Minute,
					LongWindow:  2 * time.Minute,
					BurnRate:    5,
					Severity:    AlertSeverityMajor,
				},
				{
					ShortWindow: 30 * time.Minute,
					LongWindow:  10 * time.Minute,
					BurnRate:    2,
					Severity:    AlertSeverityWarning,
				},
			},
		},
		SLAErrorRate: {
			Target:         0.1, // 0.1% error rate
			ErrorBudget:    2.0,
			BusinessTier:   "critical",
			CustomerFacing: true,
			Windows: []AlertWindow{
				{
					ShortWindow: 5 * time.Minute,
					LongWindow:  1 * time.Minute,
					BurnRate:    15,
					Severity:    AlertSeverityUrgent,
				},
				{
					ShortWindow: 30 * time.Minute,
					LongWindow:  5 * time.Minute,
					BurnRate:    8,
					Severity:    AlertSeverityCritical,
				},
			},
		},
	}
}

// NewSLAAlertManager creates a new SLA alert manager with comprehensive monitoring capabilities
func NewSLAAlertManager(config *SLAAlertConfig, logger *logging.StructuredLogger) (*SLAAlertManager, error) {
	if config == nil {
		config = DefaultSLAAlertConfig()
	}

	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	// Initialize Prometheus client
	var prometheusClient v1.API
	if config.PrometheusURL != "" {
		client, err := api.NewClient(api.Config{
			Address: config.PrometheusURL,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
		}
		prometheusClient = v1.NewAPI(client)
	}

	// Initialize circuit breaker for external dependencies
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "sla-alert-manager",
		MaxRequests: 10,
		Interval:    10 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
	})

	// Initialize metrics
	metrics := &SLAAlertMetrics{
		AlertsGenerated: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "sla_alerts_generated_total",
			Help: "Total number of SLA alerts generated",
		}, []string{"sla_type", "severity", "component"}),

		AlertsResolved: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "sla_alerts_resolved_total",
			Help: "Total number of SLA alerts resolved",
		}, []string{"sla_type", "severity", "duration_bucket"}),

		AlertsFalsePositive: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "sla_alerts_false_positive_total",
			Help: "Total number of false positive SLA alerts",
		}, []string{"sla_type", "reason"}),

		SLACompliance: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "sla_compliance_percentage",
			Help: "Current SLA compliance percentage",
		}, []string{"sla_type", "component"}),

		ErrorBudgetBurn: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "sla_error_budget_burn_rate",
			Help: "Current error budget burn rate",
		}, []string{"sla_type", "window"}),

		ActiveAlerts: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "sla_active_alerts",
			Help: "Number of active SLA alerts",
		}, []string{"sla_type", "severity", "state"}),

		BusinessImpactScore: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "sla_business_impact_score",
			Help: "Business impact score for SLA violations",
		}, []string{"sla_type", "tier"}),

		RevenueAtRisk: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "sla_revenue_at_risk_dollars",
			Help: "Revenue at risk due to SLA violations",
		}),
	}

	// Register metrics with duplicate handling
	metricsToRegister := []prometheus.Collector{
		metrics.AlertsGenerated,
		metrics.AlertsResolved,
		metrics.AlertsFalsePositive,
		metrics.SLACompliance,
		metrics.ErrorBudgetBurn,
		metrics.ActiveAlerts,
		metrics.BusinessImpactScore,
		metrics.RevenueAtRisk,
	}

	// Register each metric, ignoring duplicate registration errors
	for _, metric := range metricsToRegister {
		if err := prometheus.Register(metric); err != nil {
			if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
				// Only propagate non-duplicate errors
				logger.Error("Failed to register metric", "error", err)
			}
		}
	}

	sam := &SLAAlertManager{
		logger:             logger.WithComponent("sla-alert-manager"),
		config:             config,
		slaTargets:         DefaultSLATargets(),
		activeAlerts:       make(map[string]*SLAAlert),
		suppressedAlerts:   make(map[string]*SLAAlert),
		alertHistory:       make([]*SLAAlert, 0),
		maintenanceWindows: make([]*MaintenanceWindow, 0),
		prometheusClient:   prometheusClient,
		circuitBreaker:     cb,
		metrics:            metrics,
		stopCh:             make(chan struct{}),
	}

	// Initialize sub-components
	var err error

	sam.burnRateCalculator, err = NewBurnRateCalculator(
		&BurnRateConfig{
			SLATargets:        sam.slaTargets,
			EvaluationWindows: []time.Duration{5 * time.Minute, 30 * time.Minute, 2 * time.Hour},
			PrometheusClient:  prometheusClient,
		},
		logger.WithComponent("burn-rate-calculator"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize burn rate calculator: %w", err)
	}

	sam.predictiveAlerting, err = NewPredictiveAlerting(
		DefaultPredictiveConfig(),
		logger.WithComponent("predictive-alerting"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize predictive alerting: %w", err)
	}

	sam.alertRouter, err = NewAlertRouter(
		&AlertRouterConfig{
			MaxConcurrentAlerts: config.MaxActiveAlerts,
			DeduplicationWindow: config.DeduplicationWindow,
			CorrelationWindow:   config.CorrelationWindow,
		},
		logger.WithComponent("alert-router"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize alert router: %w", err)
	}

	sam.escalationEngine, err = NewEscalationEngine(
		&EscalationConfig{
			DefaultEscalationDelay: 15 * time.Minute,
			MaxEscalationLevels:    4,
			AutoResolutionEnabled:  true,
		},
		logger.WithComponent("escalation-engine"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize escalation engine: %w", err)
	}

	return sam, nil
}

// Start begins the SLA alert manager with all monitoring capabilities
func (sam *SLAAlertManager) Start(ctx context.Context) error {
	sam.mu.Lock()
	defer sam.mu.Unlock()

	if sam.started {
		return fmt.Errorf("SLA alert manager already started")
	}

	sam.logger.InfoWithContext("Starting SLA alert manager",
		"evaluation_interval", sam.config.EvaluationInterval,
		"max_active_alerts", sam.config.MaxActiveAlerts,
		"sla_targets", len(sam.slaTargets),
	)

	// Start sub-components
	if err := sam.burnRateCalculator.Start(ctx); err != nil {
		return fmt.Errorf("failed to start burn rate calculator: %w", err)
	}

	if err := sam.predictiveAlerting.Start(ctx); err != nil {
		return fmt.Errorf("failed to start predictive alerting: %w", err)
	}

	if err := sam.alertRouter.Start(ctx); err != nil {
		return fmt.Errorf("failed to start alert router: %w", err)
	}

	if err := sam.escalationEngine.Start(ctx); err != nil {
		return fmt.Errorf("failed to start escalation engine: %w", err)
	}

	// Start monitoring loops
	go sam.evaluationLoop(ctx)
	go sam.maintenanceWindowCheck(ctx)
	go sam.metricsUpdateLoop(ctx)
	go sam.alertCleanupLoop(ctx)

	sam.started = true
	sam.logger.InfoWithContext("SLA alert manager started successfully")

	return nil
}

// Stop gracefully shuts down the SLA alert manager
func (sam *SLAAlertManager) Stop(ctx context.Context) error {
	sam.mu.Lock()
	defer sam.mu.Unlock()

	if !sam.started {
		return nil
	}

	sam.logger.InfoWithContext("Stopping SLA alert manager")

	// Stop monitoring loops
	close(sam.stopCh)

	// Stop sub-components
	if err := sam.escalationEngine.Stop(ctx); err != nil {
		sam.logger.ErrorWithContext("Failed to stop escalation engine", err)
	}

	if err := sam.alertRouter.Stop(ctx); err != nil {
		sam.logger.ErrorWithContext("Failed to stop alert router", err)
	}

	if err := sam.predictiveAlerting.Stop(ctx); err != nil {
		sam.logger.ErrorWithContext("Failed to stop predictive alerting", err)
	}

	if err := sam.burnRateCalculator.Stop(ctx); err != nil {
		sam.logger.ErrorWithContext("Failed to stop burn rate calculator", err)
	}

	sam.started = false
	sam.logger.InfoWithContext("SLA alert manager stopped")

	return nil
}

// evaluationLoop runs the main alert evaluation loop
func (sam *SLAAlertManager) evaluationLoop(ctx context.Context) {
	ticker := time.NewTicker(sam.config.EvaluationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sam.stopCh:
			return
		case <-ticker.C:
			sam.evaluateAllSLAs(ctx)
		}
	}
}

// evaluateAllSLAs evaluates all configured SLAs for violations
func (sam *SLAAlertManager) evaluateAllSLAs(ctx context.Context) {
	for slaType, target := range sam.slaTargets {
		sam.evaluateSLA(ctx, slaType, target)
	}
}

// evaluateSLA evaluates a specific SLA for violations
func (sam *SLAAlertManager) evaluateSLA(ctx context.Context, slaType SLAType, target SLATarget) {
	// Get current SLA metrics
	currentValue, err := sam.getCurrentSLAValue(ctx, slaType)
	if err != nil {
		sam.logger.ErrorWithContext("Failed to get current SLA value", err,
			slog.String("sla_type", string(slaType)),
		)
		return
	}

	// Calculate burn rates for all windows
	burnRates, err := sam.burnRateCalculator.Calculate(ctx, slaType)
	if err != nil {
		sam.logger.ErrorWithContext("Failed to calculate burn rates", err,
			slog.String("sla_type", string(slaType)),
		)
		return
	}

	// Check each alerting window
	for _, window := range target.Windows {
		isViolating := sam.isWindowViolating(burnRates, window)

		if isViolating {
			alert := sam.createAlert(ctx, slaType, target, window, currentValue, burnRates)
			sam.processAlert(ctx, alert)
		}
	}

	// Update compliance metrics
	compliance := sam.calculateCompliance(currentValue, target)
	sam.metrics.SLACompliance.WithLabelValues(string(slaType), "overall").Set(compliance)
}

// createAlert creates a new SLA alert with full context
func (sam *SLAAlertManager) createAlert(ctx context.Context, slaType SLAType, target SLATarget,
	window AlertWindow, currentValue float64, burnRates BurnRateInfo) *SLAAlert {

	now := time.Now()
	alertID := sam.generateAlertID(slaType, window, now)

	// Calculate error budget information
	errorBudget := sam.calculateErrorBudget(currentValue, target, burnRates)

	// Enrich context
	context := sam.enrichAlertContext(ctx, slaType, currentValue)

	// Calculate business impact
	businessImpact := sam.calculateBusinessImpact(slaType, target, currentValue, errorBudget)

	alert := &SLAAlert{
		ID:             alertID,
		SLAType:        slaType,
		Name:           sam.generateAlertName(slaType, window.Severity),
		Description:    sam.generateAlertDescription(slaType, currentValue, target, window),
		Severity:       window.Severity,
		State:          AlertStateFiring,
		SLATarget:      target.Target,
		CurrentValue:   currentValue,
		Threshold:      window.BurnRate,
		ErrorBudget:    errorBudget,
		BurnRate:       burnRates,
		Labels:         sam.generateAlertLabels(slaType, target, window),
		Annotations:    sam.generateAlertAnnotations(slaType, target, window),
		Context:        context,
		StartsAt:       now,
		UpdatedAt:      now,
		Fingerprint:    sam.generateFingerprint(slaType, window),
		Hash:           sam.generateHash(alertID, now),
		BusinessImpact: businessImpact,
		RunbookURL:     sam.generateRunbookURL(slaType, window.Severity),
		RunbookSteps:   sam.generateRunbookSteps(slaType, window.Severity),
	}

	return alert
}

// processAlert processes a new alert through the routing and escalation pipeline
func (sam *SLAAlertManager) processAlert(ctx context.Context, alert *SLAAlert) {
	sam.mu.Lock()

	// Check for existing alert
	existingAlert, exists := sam.activeAlerts[alert.Fingerprint]
	if exists {
		// Update existing alert
		existingAlert.CurrentValue = alert.CurrentValue
		existingAlert.BurnRate = alert.BurnRate
		existingAlert.ErrorBudget = alert.ErrorBudget
		existingAlert.UpdatedAt = time.Now()
		existingAlert.BusinessImpact = alert.BusinessImpact
		sam.mu.Unlock()
		return
	}

	// Check if alert should be suppressed
	if sam.shouldSuppressAlert(alert) {
		sam.suppressedAlerts[alert.ID] = alert
		sam.mu.Unlock()
		sam.logger.InfoWithContext("Alert suppressed",
			slog.String("alert_id", alert.ID),
			slog.String("sla_type", string(alert.SLAType)),
			slog.String("reason", "maintenance_window"),
		)
		return
	}

	// Add to active alerts
	sam.activeAlerts[alert.Fingerprint] = alert
	sam.mu.Unlock()

	// Record metrics
	sam.metrics.AlertsGenerated.WithLabelValues(
		string(alert.SLAType),
		string(alert.Severity),
		alert.Context.Component,
	).Inc()

	sam.metrics.ActiveAlerts.WithLabelValues(
		string(alert.SLAType),
		string(alert.Severity),
		string(alert.State),
	).Inc()

	// Route alert for notifications
	if err := sam.alertRouter.Route(ctx, alert); err != nil {
		sam.logger.ErrorWithContext("Failed to route alert", err,
			slog.String("alert_id", alert.ID),
		)
	}

	// Start escalation process
	if err := sam.escalationEngine.StartEscalation(ctx, alert); err != nil {
		sam.logger.ErrorWithContext("Failed to start escalation", err,
			slog.String("alert_id", alert.ID),
		)
	}

	sam.logger.WarnWithContext("SLA alert fired",
		slog.String("alert_id", alert.ID),
		slog.String("sla_type", string(alert.SLAType)),
		slog.String("severity", string(alert.Severity)),
		slog.Float64("current_value", alert.CurrentValue),
		slog.Float64("sla_target", alert.SLATarget),
		slog.Float64("error_budget_remaining", alert.ErrorBudget.Remaining),
	)
}

// generateAlertID creates a unique identifier for the alert
func (sam *SLAAlertManager) generateAlertID(slaType SLAType, window AlertWindow, timestamp time.Time) string {
	return fmt.Sprintf("sla-%s-%s-%d",
		string(slaType),
		string(window.Severity),
		timestamp.Unix())
}

// generateFingerprint creates a fingerprint for alert deduplication
func (sam *SLAAlertManager) generateFingerprint(slaType SLAType, window AlertWindow) string {
	data := fmt.Sprintf("%s-%s-%s",
		string(slaType),
		string(window.Severity),
		window.ShortWindow.String())
	return fmt.Sprintf("%x", md5.Sum([]byte(data)))
}

// generateHash creates a unique hash for the alert
func (sam *SLAAlertManager) generateHash(alertID string, timestamp time.Time) string {
	data := fmt.Sprintf("%s-%d", alertID, timestamp.UnixNano())
	return fmt.Sprintf("%x", md5.Sum([]byte(data)))[:8]
}

// Helper functions for alert generation and management continue in the next section...
// Additional methods would include alert lifecycle management, metric calculations,
// context enrichment, business impact assessment, and maintenance window handling.

// maintenanceWindowCheck checks for active maintenance windows
func (sam *SLAAlertManager) maintenanceWindowCheck(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sam.stopCh:
			return
		case <-ticker.C:
			sam.updateMaintenanceWindows()
		}
	}
}

// metricsUpdateLoop periodically updates metrics
func (sam *SLAAlertManager) metricsUpdateLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sam.stopCh:
			return
		case <-ticker.C:
			sam.updateMetrics()
		}
	}
}

// alertCleanupLoop cleans up old resolved alerts
func (sam *SLAAlertManager) alertCleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sam.stopCh:
			return
		case <-ticker.C:
			sam.cleanupOldAlerts()
		}
	}
}

// getCurrentSLAValue retrieves the current value for the specified SLA type
func (sam *SLAAlertManager) getCurrentSLAValue(ctx context.Context, slaType SLAType) (float64, error) {
	if sam.prometheusClient == nil {
		// Return mock values for testing
		switch slaType {
		case SLATypeAvailability:
			return 99.90, nil // 99.90% availability
		case SLATypeLatency:
			return 1800, nil // 1.8 seconds
		case SLAThroughput:
			return 42, nil // 42 intents per minute
		case SLAErrorRate:
			return 0.15, nil // 0.15% error rate
		default:
			return 0, fmt.Errorf("unknown SLA type: %s", slaType)
		}
	}
	
	// Get metric name for SLA type
	metricName := sam.getMetricNameForSLAType(slaType)
	
	_, _, err := sam.prometheusClient.Query(ctx, metricName, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to query Prometheus for %s: %w", slaType, err)
	}
	
	// Parse result and return value
	// Implementation would depend on Prometheus result format
	// For now, return mock value
	return sam.getMockValueForSLAType(slaType), nil
}

// getMockValueForSLAType returns mock values for testing
func (sam *SLAAlertManager) getMockValueForSLAType(slaType SLAType) float64 {
	switch slaType {
	case SLATypeAvailability:
		return 99.90 // 99.90% availability
	case SLATypeLatency:
		return 1800 // 1.8 seconds
	case SLAThroughput:
		return 42 // 42 intents per minute
	case SLAErrorRate:
		return 0.15 // 0.15% error rate
	default:
		return 0
	}
}

// isWindowViolating checks if burn rates violate the specified window thresholds
func (sam *SLAAlertManager) isWindowViolating(burnRates BurnRateInfo, window AlertWindow) bool {
	// Check if any window burn rate exceeds threshold
	shortWindowViolating := burnRates.ShortWindow.BurnRate > window.BurnRate
	mediumWindowViolating := burnRates.MediumWindow.BurnRate > window.BurnRate
	longWindowViolating := burnRates.LongWindow.BurnRate > window.BurnRate
	
	// Return true if any window is violating
	return shortWindowViolating || mediumWindowViolating || longWindowViolating
}

// calculateCompliance calculates SLA compliance percentage
func (sam *SLAAlertManager) calculateCompliance(currentValue float64, target SLATarget) float64 {
	switch {
	case target.Target >= 90: // Availability or similar metrics
		return (currentValue / target.Target) * 100
	default: // Latency or error rate metrics (lower is better)
		if target.Target == 0 {
			return 100 // Perfect score if target is 0
		}
		compliance := (1 - (currentValue / target.Target)) * 100
		if compliance < 0 {
			return 0
		}
		return compliance
	}
}

// calculateErrorBudget calculates error budget information
func (sam *SLAAlertManager) calculateErrorBudget(currentValue float64, target SLATarget, burnRates BurnRateInfo) ErrorBudgetInfo {
	// Calculate total error budget for the period (monthly)
	totalBudget := target.ErrorBudget
	
	// Calculate consumed budget based on current burn rate
	consumed := burnRates.CurrentRate * totalBudget / 100
	remaining := totalBudget - consumed
	consumedPercent := (consumed / totalBudget) * 100
	
	errorBudget := ErrorBudgetInfo{
		Total:           totalBudget,
		Remaining:       remaining,
		Consumed:        consumed,
		ConsumedPercent: consumedPercent,
	}
	
	// Calculate time to exhaustion if current burn rate continues
	if burnRates.CurrentRate > 0 && remaining > 0 {
		hoursToExhaustion := remaining / burnRates.CurrentRate
		timeToExhaustion := time.Duration(hoursToExhaustion * float64(time.Hour))
		errorBudget.TimeToExhaustion = &timeToExhaustion
	}
	
	return errorBudget
}

// enrichAlertContext enriches alert with additional context information
func (sam *SLAAlertManager) enrichAlertContext(ctx context.Context, slaType SLAType, currentValue float64) AlertContext {
	context := AlertContext{
		Component:       "nephoran-intent-operator",
		Service:         string(slaType),
		Environment:     "production",
		Region:          "us-east-1",
		RelatedMetrics:  make([]MetricSnapshot, 0),
		RecentIncidents: make([]IncidentSummary, 0),
		DashboardLinks:  make([]string, 0),
		LogQueries:      make([]LogQuery, 0),
	}
	
	// Add related metrics
	context.RelatedMetrics = append(context.RelatedMetrics, MetricSnapshot{
		Name:      string(slaType) + "_current_value",
		Value:     currentValue,
		Unit:      sam.getUnitForSLAType(slaType),
		Timestamp: time.Now(),
	})
	
	// Add dashboard links
	context.DashboardLinks = append(context.DashboardLinks, 
		fmt.Sprintf("%s/d/nephoran-sla/nephoran-sla-dashboard", sam.config.GrafanaURL))
	
	// Add log queries
	context.LogQueries = append(context.LogQueries, LogQuery{
		Description: fmt.Sprintf("Recent %s errors", slaType),
		Query:       fmt.Sprintf("level=error component=%s", context.Component),
		Source:      "kubernetes",
	})
	
	return context
}

// calculateBusinessImpact calculates business impact for the alert
func (sam *SLAAlertManager) calculateBusinessImpact(slaType SLAType, target SLATarget, currentValue float64, errorBudget ErrorBudgetInfo) BusinessImpactInfo {
	businessImpact := BusinessImpactInfo{
		ServiceTier:    target.BusinessTier,
		CustomerFacing: target.CustomerFacing,
		SLABreach:      errorBudget.ConsumedPercent > 100,
	}
	
	// Calculate severity based on error budget consumption
	switch {
	case errorBudget.ConsumedPercent > 90:
		businessImpact.Severity = "critical"
	case errorBudget.ConsumedPercent > 75:
		businessImpact.Severity = "major"
	case errorBudget.ConsumedPercent > 50:
		businessImpact.Severity = "minor"
	default:
		businessImpact.Severity = "low"
	}
	
	// Estimate affected users (simplified calculation)
	if target.CustomerFacing {
		businessImpact.AffectedUsers = 1000 // Base user count
		if errorBudget.ConsumedPercent > 75 {
			businessImpact.AffectedUsers *= int64(errorBudget.ConsumedPercent / 25)
		}
	}
	
	// Calculate revenue impact if business impact tracking is enabled
	if sam.config.EnableBusinessImpact && businessImpact.AffectedUsers > 0 {
		businessImpact.RevenueImpact = float64(businessImpact.AffectedUsers) * sam.config.RevenuePerUser
	}
	
	return businessImpact
}

// generateAlertName generates a descriptive name for the alert
func (sam *SLAAlertManager) generateAlertName(slaType SLAType, severity AlertSeverity) string {
	return fmt.Sprintf("%s SLA %s Alert", 
		sam.capitalizeFirst(string(slaType)), 
		sam.capitalizeFirst(string(severity)))
}

// generateAlertDescription generates a detailed description for the alert
func (sam *SLAAlertManager) generateAlertDescription(slaType SLAType, currentValue float64, target SLATarget, window AlertWindow) string {
	return fmt.Sprintf("SLA violation detected for %s. Current value: %.2f, Target: %.2f, Burn rate threshold: %.2f exceeded over %v window.",
		slaType, currentValue, target.Target, window.BurnRate, window.ShortWindow)
}

// generateAlertLabels generates labels for the alert
func (sam *SLAAlertManager) generateAlertLabels(slaType SLAType, target SLATarget, window AlertWindow) map[string]string {
	return map[string]string{
		"sla_type":        string(slaType),
		"severity":        string(window.Severity),
		"business_tier":   target.BusinessTier,
		"customer_facing": fmt.Sprintf("%t", target.CustomerFacing),
		"component":       "nephoran-intent-operator",
		"alert_type":      "sla_violation",
	}
}

// generateAlertAnnotations generates annotations for the alert
func (sam *SLAAlertManager) generateAlertAnnotations(slaType SLAType, target SLATarget, window AlertWindow) map[string]string {
	annotations := map[string]string{
		"summary":     fmt.Sprintf("SLA violation for %s", slaType),
		"description": fmt.Sprintf("Burn rate threshold %.2f exceeded for %v", window.BurnRate, window.ShortWindow),
	}
	
	if sam.config.GrafanaURL != "" {
		annotations["dashboard"] = fmt.Sprintf("%s/d/nephoran-sla/nephoran-sla-dashboard", sam.config.GrafanaURL)
	}
	
	if sam.config.RunbookBaseURL != "" {
		annotations["runbook"] = fmt.Sprintf("%s/sla-%s", sam.config.RunbookBaseURL, slaType)
	}
	
	return annotations
}

// Helper methods

// getMetricNameForSLAType returns the Prometheus metric name for an SLA type
func (sam *SLAAlertManager) getMetricNameForSLAType(slaType SLAType) string {
	switch slaType {
	case SLATypeAvailability:
		return "nephoran_availability_percentage"
	case SLATypeLatency:
		return "nephoran_latency_p95_seconds"
	case SLAThroughput:
		return "nephoran_throughput_intents_per_minute"
	case SLAErrorRate:
		return "nephoran_error_rate_percentage"
	default:
		return "unknown_metric"
	}
}

// getUnitForSLAType returns the unit for an SLA type
func (sam *SLAAlertManager) getUnitForSLAType(slaType SLAType) string {
	switch slaType {
	case SLATypeAvailability:
		return "percentage"
	case SLATypeLatency:
		return "milliseconds"
	case SLAThroughput:
		return "per_minute"
	case SLAErrorRate:
		return "percentage"
	default:
		return "unknown"
	}
}

// capitalizeFirst capitalizes the first letter of a string
func (sam *SLAAlertManager) capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return fmt.Sprintf("%c%s", s[0]-32, s[1:])
}

// generateRunbookURL generates runbook URL for the alert
func (sam *SLAAlertManager) generateRunbookURL(slaType SLAType, severity AlertSeverity) string {
	if sam.config.RunbookBaseURL == "" {
		return ""
	}
	return fmt.Sprintf("%s/sla-%s-%s", sam.config.RunbookBaseURL, slaType, severity)
}

// generateRunbookSteps generates runbook steps for the alert
func (sam *SLAAlertManager) generateRunbookSteps(slaType SLAType, severity AlertSeverity) []string {
	steps := []string{
		"1. Check system health dashboard",
		"2. Review recent deployments and changes",
		"3. Analyze error logs and metrics",
	}
	
	switch slaType {
	case SLATypeAvailability:
		steps = append(steps, "4. Check service status and endpoints", "5. Verify infrastructure health")
	case SLATypeLatency:
		steps = append(steps, "4. Check performance metrics", "5. Review database query performance")
	case SLAThroughput:
		steps = append(steps, "4. Check resource utilization", "5. Review queue depths and processing rates")
	case SLAErrorRate:
		steps = append(steps, "4. Check error logs and patterns", "5. Review recent code changes")
	}
	
	if severity == AlertSeverityUrgent || severity == AlertSeverityCritical {
		steps = append(steps, "6. Escalate to on-call engineer", "7. Consider emergency rollback if needed")
	}
	
	return steps
}

// shouldSuppressAlert checks if an alert should be suppressed
func (sam *SLAAlertManager) shouldSuppressAlert(alert *SLAAlert) bool {
	// Check for active maintenance windows
	now := time.Now()
	for _, window := range sam.maintenanceWindows {
		if now.After(window.StartTime) && now.Before(window.EndTime) {
			// Check if alert matches maintenance window criteria
			if sam.alertMatchesMaintenanceWindow(alert, window) {
				return true
			}
		}
	}
	
	// Check suppression rules based on alert characteristics
	if alert.ErrorBudget.ConsumedPercent < 10.0 {
		// Suppress minor alerts when error budget consumption is low
		return true
	}
	
	return false
}

// alertMatchesMaintenanceWindow checks if an alert matches maintenance window criteria
func (sam *SLAAlertManager) alertMatchesMaintenanceWindow(alert *SLAAlert, window *MaintenanceWindow) bool {
	// Check if alert's component is in maintenance
	for _, component := range window.Components {
		if alert.Context.Component == component {
			return true
		}
	}
	
	// Check if alert's service is in maintenance
	for _, service := range window.Services {
		if alert.Context.Service == service {
			return true
		}
	}
	
	// Check labels matching
	for key, value := range window.Labels {
		if alertValue, exists := alert.Labels[key]; exists && alertValue == value {
			return true
		}
	}
	
	return false
}

// updateMaintenanceWindows updates active maintenance windows
func (sam *SLAAlertManager) updateMaintenanceWindows() {
	sam.mu.Lock()
	defer sam.mu.Unlock()
	
	now := time.Now()
	activeWindows := make([]*MaintenanceWindow, 0)
	
	// Filter out expired maintenance windows
	for _, window := range sam.maintenanceWindows {
		if now.Before(window.EndTime) {
			activeWindows = append(activeWindows, window)
		} else {
			sam.logger.InfoWithContext("Maintenance window expired",
				slog.String("window_id", window.ID),
				slog.String("name", window.Name),
			)
		}
	}
	
	sam.maintenanceWindows = activeWindows
	
	sam.logger.DebugWithContext("Updated maintenance windows",
		slog.Int("active_windows", len(sam.maintenanceWindows)),
	)
}

// updateMetrics updates internal metrics
func (sam *SLAAlertManager) updateMetrics() {
	sam.mu.RLock()
	defer sam.mu.RUnlock()
	
	// Update active alerts count
	for slaType, severity := range map[SLAType]AlertSeverity{
		SLATypeAvailability: AlertSeverityCritical,
		SLATypeLatency:      AlertSeverityMajor,
		SLAThroughput:       AlertSeverityWarning,
		SLAErrorRate:        AlertSeverityUrgent,
	} {
		count := 0
		for _, alert := range sam.activeAlerts {
			if alert.SLAType == slaType && alert.Severity == severity {
				count++
			}
		}
		sam.metrics.ActiveAlerts.WithLabelValues(
			string(slaType),
			string(severity),
			string(AlertStateFiring),
		).Set(float64(count))
	}
	
	// Update business impact metrics
	var totalRevenue float64
	for _, alert := range sam.activeAlerts {
		if alert.BusinessImpact.RevenueImpact > 0 {
			totalRevenue += alert.BusinessImpact.RevenueImpact
		}
		
		sam.metrics.BusinessImpactScore.WithLabelValues(
			string(alert.SLAType),
			alert.BusinessImpact.ServiceTier,
		).Set(alert.BusinessImpact.RevenueImpact)
		
		if alert.BusinessImpact.CustomerFacing {
			sam.metrics.CustomerImpact.WithLabelValues(
				string(alert.SLAType),
				"customer_facing",
			).Set(float64(alert.BusinessImpact.AffectedUsers))
		}
	}
	
	sam.metrics.RevenueAtRisk.Set(totalRevenue)
	
	sam.logger.DebugWithContext("Updated SLA alert metrics",
		slog.Int("active_alerts", len(sam.activeAlerts)),
		slog.Float64("total_revenue_at_risk", totalRevenue),
	)
}

// cleanupOldAlerts removes old resolved alerts from history
func (sam *SLAAlertManager) cleanupOldAlerts() {
	sam.mu.Lock()
	defer sam.mu.Unlock()
	
	now := time.Now()
	cutoff := now.Add(-sam.config.AlertRetention)
	
	// Clean up alert history
	var recentHistory []*SLAAlert
	for _, alert := range sam.alertHistory {
		if alert.UpdatedAt.After(cutoff) {
			recentHistory = append(recentHistory, alert)
		}
	}
	
	removedCount := len(sam.alertHistory) - len(recentHistory)
	sam.alertHistory = recentHistory
	
	// Clean up suppressed alerts
	var activeSuppressed = make(map[string]*SLAAlert)
	for id, alert := range sam.suppressedAlerts {
		if alert.UpdatedAt.After(cutoff) {
			activeSuppressed[id] = alert
		}
	}
	
	removedSuppressedCount := len(sam.suppressedAlerts) - len(activeSuppressed)
	sam.suppressedAlerts = activeSuppressed
	
	if removedCount > 0 || removedSuppressedCount > 0 {
		sam.logger.InfoWithContext("Cleaned up old alerts",
			slog.Int("removed_from_history", removedCount),
			slog.Int("removed_suppressed", removedSuppressedCount),
			slog.Int("remaining_history", len(sam.alertHistory)),
			slog.Int("remaining_suppressed", len(sam.suppressedAlerts)),
		)
	}
}
