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

// Additional AlertState values not in escalation_engine.go
const (
	AlertStateSuppressed   AlertState = "suppressed"
	AlertStateAcknowledged AlertState = "acknowledged"
)

// SLAAlertExtended extends the base SLAAlert with additional context
// (Using composition to avoid redeclaration)
type SLAAlertExtended struct {
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
	Context     EnrichedAlertContext `json:"context"`

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

// EnrichedAlertContext provides enriched context for the alert
// Extends the base AlertContext from escalation_engine.go with additional fields
type EnrichedAlertContext struct {
	AlertContext    // Embed the base AlertContext from escalation_engine.go
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

// MaintenanceWindowExtended extends base MaintenanceWindow with additional fields
// (MaintenanceWindow is already defined in escalation_engine.go)
type MaintenanceWindowExtended struct {
	MaintenanceWindow
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

	// Register metrics
	prometheus.MustRegister(
		metrics.AlertsGenerated,
		metrics.AlertsResolved,
		metrics.AlertsFalsePositive,
		metrics.SLACompliance,
		metrics.ErrorBudgetBurn,
		metrics.ActiveAlerts,
		metrics.BusinessImpactScore,
		metrics.RevenueAtRisk,
	)

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
		&PredictiveConfig{
			ModelUpdateInterval: 24 * time.Hour,
			PredictionWindow:    60 * time.Minute,
			ConfidenceThreshold: 0.75,
			TrainingDataWindow:  30 * 24 * time.Hour,
		},
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

// GetActiveAlerts returns all currently active alerts
func (sam *SLAAlertManager) GetActiveAlerts() []*SLAAlert {
	sam.mu.RLock()
	defer sam.mu.RUnlock()
	
	alerts := make([]*SLAAlert, 0, len(sam.activeAlerts))
	for _, alert := range sam.activeAlerts {
		alerts = append(alerts, alert)
	}
	
	return alerts
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
	businessImpact := sam.calculateBusinessImpact(slaType, target, currentValue, errorBudget.ConsumedPercent)

	// Create unified SLAAlert with all fields
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
		StartsAt:       now,
		UpdatedAt:      now,
		Fingerprint:    sam.generateFingerprint(slaType, window),
		Hash:           sam.generateHash(alertID, now),
		BusinessImpact: BusinessImpactScore{
			OverallScore:     businessImpact.RevenueImpact / 1000.0, // Convert to 0-1 scale
			UserImpact:       float64(businessImpact.AffectedUsers) / 10000.0, // Normalize user count
			RevenueImpact:    businessImpact.RevenueImpact,
			ReputationImpact: 0.0, // Calculate based on severity if needed
			ServiceTier:      businessImpact.ServiceTier,
			CustomerFacing:   businessImpact.CustomerFacing,
		},
		Context: AlertContext{
			Component:      context.Component,
			Service:        context.Service,
			Environment:    context.Environment,
			Region:         context.Region,
			Cluster:        context.Cluster,
			Namespace:      context.Namespace,
			ResourceType:   context.ResourceType,
			ResourceName:   context.ResourceName,
			RelatedMetrics: sam.extractMetricNames(context.RelatedMetrics),
		},
		CreatedAt: now,
		Metadata:  make(map[string]string),
	}

	return alert
}

// processAlert processes a new alert through the routing and escalation pipeline
func (sam *SLAAlertManager) processAlert(ctx context.Context, alert *SLAAlert) {
	sam.mu.Lock()

	// Check for existing alert using generated fingerprint
	fingerprint := sam.generateFingerprint(alert.SLAType, AlertWindow{})
	existingAlert, exists := sam.activeAlerts[fingerprint]
	if exists {
		// Update existing alert with available fields
		existingAlert.UpdatedAt = time.Now()
		existingAlert.State = alert.State
		sam.mu.Unlock()
		return
	}

	// Check if alert should be suppressed
	if sam.shouldSuppressAlert(alert.SLAType) {
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
	sam.activeAlerts[fingerprint] = alert
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

// getCurrentSLAValue retrieves the current SLA value for the given target
func (sam *SLAAlertManager) getCurrentSLAValue(ctx context.Context, slaType SLAType) (float64, error) {
	// In production, this would query monitoring systems like Prometheus
	// For now, return a mock value
	return 0.995, nil
}

// isWindowViolating checks if the current window is violating the SLA
func (sam *SLAAlertManager) isWindowViolating(burnRates BurnRateInfo, window AlertWindow) bool {
	// Check if any burn rate exceeds the threshold for this window
	if burnRates.ShortWindow.BurnRate > window.BurnRate {
		return true
	}
	if burnRates.MediumWindow.BurnRate > window.BurnRate {
		return true
	}
	if burnRates.LongWindow.BurnRate > window.BurnRate {
		return true
	}
	return false
}

// calculateCompliance calculates the compliance percentage for the SLA
func (sam *SLAAlertManager) calculateCompliance(currentValue float64, target SLATarget) float64 {
	return currentValue * 100
}

// calculateErrorBudget calculates the remaining error budget
func (sam *SLAAlertManager) calculateErrorBudget(currentValue float64, target SLATarget, burnRates BurnRateInfo) ErrorBudgetInfo {
	remainingBudget := 1 - target.Target
	consumedBudget := target.Target - currentValue
	if consumedBudget < 0 {
		consumedBudget = 0
	}
	
	return ErrorBudgetInfo{
		Total:           remainingBudget,
		Remaining:       remainingBudget - consumedBudget,
		Consumed:        consumedBudget,
		ConsumedPercent: (consumedBudget / remainingBudget) * 100,
		TimeToExhaustion: nil, // Would be calculated based on burn rate
	}
}

// extractMetricNames extracts metric names from MetricSnapshot slice
func (sam *SLAAlertManager) extractMetricNames(snapshots []MetricSnapshot) []string {
	names := make([]string, len(snapshots))
	for i, snapshot := range snapshots {
		names[i] = snapshot.Name
	}
	return names
}

// generateRelatedMetrics generates related metric snapshots based on SLA type
func (sam *SLAAlertManager) generateRelatedMetrics(slaType SLAType, currentValue float64) []MetricSnapshot {
	now := time.Now()
	
	switch slaType {
	case SLATypeAvailability:
		return []MetricSnapshot{
			{Name: "http_requests_total", Value: currentValue, Timestamp: now},
			{Name: "http_request_duration_seconds", Value: 0.5, Timestamp: now},
			{Name: "up", Value: 1.0, Timestamp: now},
		}
	case SLATypeLatency:
		return []MetricSnapshot{
			{Name: "http_request_duration_seconds_p95", Value: currentValue / 1000, Timestamp: now},
			{Name: "http_request_duration_seconds_p99", Value: (currentValue / 1000) * 1.2, Timestamp: now},
		}
	case SLAThroughput:
		return []MetricSnapshot{
			{Name: "http_requests_per_second", Value: currentValue, Timestamp: now},
			{Name: "active_connections", Value: 100, Timestamp: now},
		}
	case SLAErrorRate:
		return []MetricSnapshot{
			{Name: "http_requests_total", Value: 1000, Timestamp: now},
			{Name: "http_requests_error_total", Value: currentValue * 10, Timestamp: now},
		}
	default:
		return []MetricSnapshot{}
	}
}

// enrichAlertContext adds additional context to the alert
func (sam *SLAAlertManager) enrichAlertContext(ctx context.Context, slaType SLAType, currentValue float64) EnrichedAlertContext {
	// Add additional context based on the target and current conditions
	// Generate metric snapshots based on SLA type
	relatedMetrics := sam.generateRelatedMetrics(slaType, currentValue)
	
	return EnrichedAlertContext{
		AlertContext: AlertContext{
			Component:    "nephoran-operator",
			Service:      string(slaType),
			Region:       "default",
			Environment:  "production",
		},
		RelatedMetrics:  relatedMetrics,
		RecentIncidents: []IncidentSummary{},
		DashboardLinks:  []string{},
		LogQueries:      []LogQuery{},
	}
}

// calculateBusinessImpact calculates the business impact of the SLA violation
func (sam *SLAAlertManager) calculateBusinessImpact(slaType SLAType, target SLATarget, currentValue float64, errorBudget float64) BusinessImpactInfo {
	// Check if target has CustomerFacing flag
	isCustomerFacing := target.CustomerFacing
	deficit := target.Target - currentValue
	severity := "none"
	if deficit > 0 {
		if deficit < 0.001 {
			severity = "low"
		} else if deficit < 0.005 {
			severity = "medium"
		} else {
			severity = "high"
		}
	}
	
	return BusinessImpactInfo{
		Severity:       severity,
		AffectedUsers:  0, // Would be calculated from actual metrics
		RevenueImpact:  0.0,
		SLABreach:      deficit > 0,
		CustomerFacing: isCustomerFacing,
		ServiceTier:    "production",
	}
}

// generateAlertName generates a descriptive name for the alert
func (sam *SLAAlertManager) generateAlertName(slaType SLAType, severity AlertSeverity) string {
	return fmt.Sprintf("SLA Violation - %s (%s)", string(slaType), string(severity))
}

// generateAlertDescription generates a detailed description for the alert
func (sam *SLAAlertManager) generateAlertDescription(slaType SLAType, currentValue float64, target SLATarget, window AlertWindow) string {
	return fmt.Sprintf("SLA %s is violating threshold %.3f with current value %.3f",
		string(slaType), target.Target, currentValue)
}

// generateAlertLabels generates labels for the alert
func (sam *SLAAlertManager) generateAlertLabels(slaType SLAType, target SLATarget, window AlertWindow) map[string]string {
	return map[string]string{
		"sla_name":    string(slaType),
		"sla_type":    string(slaType),
		"severity":    string(window.Severity),
		"window":      "alert-window",
	}
}

// generateAlertAnnotations generates annotations for the alert
func (sam *SLAAlertManager) generateAlertAnnotations(slaType SLAType, target SLATarget, window AlertWindow) map[string]string {
	return map[string]string{
		"description": fmt.Sprintf("SLA %s violation", string(slaType)),
		"runbook":     sam.getRunbookURL(target),
		"dashboard":   sam.getDashboardURL(target),
	}
}

// calculateSeverity calculates severity based on business impact
func (sam *SLAAlertManager) calculateSeverity(businessImpact string) string {
	switch businessImpact {
	case "high":
		return "critical"
	case "medium":
		return "warning"
	case "low":
		return "info"
	default:
		return "info"
	}
}

// getRunbookURL returns the runbook URL for the SLA target
func (sam *SLAAlertManager) getRunbookURL(target SLATarget) string {
	return "https://runbooks.example.com/sla/default"
}

// getDashboardURL returns the dashboard URL for the SLA target
func (sam *SLAAlertManager) getDashboardURL(target SLATarget) string {
	return "https://dashboards.example.com/sla/default"
}

// generateRunbookURL returns the runbook URL for the SLA target
func (sam *SLAAlertManager) generateRunbookURL(target SLATarget) string {
	return "https://runbooks.example.com/sla/default"
}

// generateRunbookSteps returns runbook steps for the alert
func (sam *SLAAlertManager) generateRunbookSteps(slaType SLAType, severity AlertSeverity) []string {
	return []string{
		fmt.Sprintf("Check %s metrics dashboard", string(slaType)),
		"Verify system health status",
		"Review recent deployments",
		"Check error logs",
		"Contact on-call engineer if needed",
	}
}

// shouldSuppressAlert checks if an alert should be suppressed due to maintenance
func (sam *SLAAlertManager) shouldSuppressAlert(slaType SLAType) bool {
	// Check maintenance windows
	now := time.Now()
	for _, window := range sam.maintenanceWindows {
		if now.After(window.StartTime) && now.Before(window.EndTime) {
			// MaintenanceWindow doesn't have Services field in the base definition
			// For now, suppress all alerts during any maintenance window
			return true
		}
	}
	return false
}

// updateMaintenanceWindows updates the maintenance windows configuration
func (sam *SLAAlertManager) updateMaintenanceWindows() {
	// In production, this would load from configuration or API
	// For now, use empty maintenance windows
	sam.maintenanceWindows = []*MaintenanceWindow{}
}

// updateMetrics updates the internal metrics
func (sam *SLAAlertManager) updateMetrics() {
	// Update alerting metrics
	if sam.metrics != nil && sam.metrics.AlertsGenerated != nil {
		sam.metrics.AlertsGenerated.WithLabelValues("sla_violation").Inc()
	}
}

// cleanupOldAlerts removes old resolved alerts from memory
func (sam *SLAAlertManager) cleanupOldAlerts() {
	cutoff := time.Now().Add(-24 * time.Hour) // Keep alerts for 24 hours
	sam.mu.Lock()
	defer sam.mu.Unlock()
	
	for id, alert := range sam.activeAlerts {
		// SLAAlert doesn't have EndsAt field, so check UpdatedAt instead
		if alert.UpdatedAt.Before(cutoff) {
			delete(sam.activeAlerts, id)
		}
	}
}

// convertToExtended converts a base SLAAlert to SLAAlertExtended for internal processing
func (sam *SLAAlertManager) convertToExtended(alert *SLAAlert) *SLAAlertExtended {
	return &SLAAlertExtended{
		ID:          alert.ID,
		SLAType:     alert.SLAType,
		Name:        alert.Name,
		Description: alert.Description,
		Severity:    alert.Severity,
		State:       alert.State,
		
		// Set default values for missing fields
		SLATarget:    0.0,
		CurrentValue: 0.0,
		Threshold:    0.0,
		ErrorBudget:  ErrorBudgetInfo{},
		BurnRate:     BurnRateInfo{},
		
		Labels:      alert.Labels,
		Annotations: make(map[string]string),
		Context: EnrichedAlertContext{
			AlertContext: alert.Context,
		},
		
		StartsAt:  alert.CreatedAt,
		UpdatedAt: alert.UpdatedAt,
		
		Fingerprint: sam.generateFingerprint(alert.SLAType, AlertWindow{}),
		Hash:        sam.generateHash(alert.ID, alert.CreatedAt),
		
		BusinessImpact: BusinessImpactInfo{
			Severity: "medium",
		},
	}
}

