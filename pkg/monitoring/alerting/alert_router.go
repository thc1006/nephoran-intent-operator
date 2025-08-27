// Package alerting provides intelligent alert routing and deduplication
// to reduce alert fatigue and ensure rapid incident response.
package alerting

import (
	"context"
	"crypto/md5"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// AlertRouter provides intelligent alert routing with deduplication,
// priority scoring, and context-aware notification delivery
type AlertRouter struct {
	logger *logging.StructuredLogger
	config *AlertRouterConfig

	// Routing infrastructure
	routingRules         []*RoutingRule
	notificationChannels map[string]NotificationChannel

	// Deduplication and correlation
	alertFingerprints map[string]*AlertGroup
	correlationEngine *CorrelationEngine

	// Priority and impact analysis
	priorityCalculator *PriorityCalculator
	impactAnalyzer     *ImpactAnalyzer

	// Geographic and timezone routing
	geographicRouter *GeographicRouter
	timezoneManager  *TimezoneManager

	// Alert enrichment
	contextEnricher *ContextEnricher
	runbookManager  *RunbookManager

	// Performance tracking
	metrics      *AlertRouterMetrics
	routingStats *RoutingStatistics

	// State management
	started    bool
	stopCh     chan struct{}
	alertQueue chan *EnrichedAlert
	mu         sync.RWMutex
}

// AlertRouterConfig holds configuration for the alert router
type AlertRouterConfig struct {
	// Core routing settings
	MaxConcurrentAlerts int           `yaml:"max_concurrent_alerts"`
	DeduplicationWindow time.Duration `yaml:"deduplication_window"`
	CorrelationWindow   time.Duration `yaml:"correlation_window"`

	// Queue management
	AlertQueueSize    int           `yaml:"alert_queue_size"`
	ProcessingWorkers int           `yaml:"processing_workers"`
	MaxRetries        int           `yaml:"max_retries"`
	RetryBackoff      time.Duration `yaml:"retry_backoff"`

	// Notification settings
	NotificationTimeout time.Duration `yaml:"notification_timeout"`
	MaxNotificationRate int           `yaml:"max_notification_rate"`
	BatchNotifications  bool          `yaml:"batch_notifications"`
	BatchWindow         time.Duration `yaml:"batch_window"`

	// Geographic routing
	EnableGeographicRouting bool   `yaml:"enable_geographic_routing"`
	DefaultTimezone         string `yaml:"default_timezone"`
	BusinessHoursStart      int    `yaml:"business_hours_start"`
	BusinessHoursEnd        int    `yaml:"business_hours_end"`

	// Performance settings
	MetricCacheSize   int           `yaml:"metric_cache_size"`
	EnrichmentTimeout time.Duration `yaml:"enrichment_timeout"`
}

// RoutingRule defines how alerts should be routed based on criteria
type RoutingRule struct {
	Name       string             `yaml:"name"`
	Priority   int                `yaml:"priority"`
	Enabled    bool               `yaml:"enabled"`
	Conditions []RoutingCondition `yaml:"conditions"`
	Actions    []RoutingAction    `yaml:"actions"`
	Schedule   *RoutingSchedule   `yaml:"schedule,omitempty"`
	RateLimit  *RateLimit         `yaml:"rate_limit,omitempty"`
}

// RoutingCondition defines criteria for applying a routing rule
type RoutingCondition struct {
	Field    string   `yaml:"field"`    // sla_type, severity, component, etc.
	Operator string   `yaml:"operator"` // equals, contains, matches, greater_than
	Values   []string `yaml:"values"`
	Negate   bool     `yaml:"negate"`
}

// RoutingAction defines what to do when a rule matches
type RoutingAction struct {
	Type       string            `yaml:"type"`   // notify, escalate, suppress, transform
	Target     string            `yaml:"target"` // channel name, escalation policy
	Parameters map[string]string `yaml:"parameters"`
	Delay      time.Duration     `yaml:"delay,omitempty"`
}

// RoutingSchedule defines time-based routing constraints
type RoutingSchedule struct {
	Timezone        string   `yaml:"timezone"`
	BusinessHours   bool     `yaml:"business_hours"`
	Weekdays        []string `yaml:"weekdays"`
	ExcludedDates   []string `yaml:"excluded_dates"`
	MaintenanceMode bool     `yaml:"maintenance_mode"`
}

// RateLimit defines rate limiting for notifications
type RateLimit struct {
	MaxAlerts    int           `yaml:"max_alerts"`
	Window       time.Duration `yaml:"window"`
	BurstAllowed int           `yaml:"burst_allowed"`
}

// EnrichedAlert extends SLAAlert with routing context
type EnrichedAlert struct {
	*SLAAlert

	// Routing metadata
	RoutingDecision *RoutingDecision    `json:"routing_decision"`
	Priority        int                 `json:"priority"`
	BusinessImpact  BusinessImpactScore `json:"business_impact"`

	// Enrichment data
	RelatedIncidents []string        `json:"related_incidents"`
	SimilarAlerts    []string        `json:"similar_alerts"`
	RunbookActions   []RunbookAction `json:"runbook_actions"`

	// Geographic context
	AffectedRegions []string     `json:"affected_regions"`
	TimezoneInfo    TimezoneInfo `json:"timezone_info"`

	// Processing metadata
	ProcessedAt       time.Time     `json:"processed_at"`
	EnrichmentLatency time.Duration `json:"enrichment_latency"`
	RoutingLatency    time.Duration `json:"routing_latency"`
}

// RoutingDecision contains the routing decision for an alert
type RoutingDecision struct {
	MatchedRules      []string   `json:"matched_rules"`
	SelectedChannels  []string   `json:"selected_channels"`
	EscalationPolicy  string     `json:"escalation_policy,omitempty"`
	Suppressed        bool       `json:"suppressed"`
	SuppressionReason string     `json:"suppression_reason,omitempty"`
	DelayUntil        *time.Time `json:"delay_until,omitempty"`
}

// BusinessImpactScore quantifies business impact of an alert
type BusinessImpactScore struct {
	OverallScore     float64 `json:"overall_score"`
	UserImpact       float64 `json:"user_impact"`
	RevenueImpact    float64 `json:"revenue_impact"`
	ReputationImpact float64 `json:"reputation_impact"`
	ServiceTier      string  `json:"service_tier"`
}

// AlertGroup represents a group of related alerts for deduplication
type AlertGroup struct {
	ID               string      `json:"id"`
	Fingerprint      string      `json:"fingerprint"`
	Representative   *SLAAlert   `json:"representative"`
	Members          []*SLAAlert `json:"members"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
	NotificationSent bool        `json:"notification_sent"`
}

// CorrelationEngine identifies related alerts
type CorrelationEngine struct {
	logger       *logging.StructuredLogger
	config       *AlertRouterConfig
	correlations map[string]*AlertCorrelation
	mu           sync.RWMutex
}

// AlertCorrelation represents correlated alerts
type AlertCorrelation struct {
	ID              string    `json:"id"`
	Alerts          []string  `json:"alerts"`
	CorrelationType string    `json:"correlation_type"`
	Confidence      float64   `json:"confidence"`
	CreatedAt       time.Time `json:"created_at"`
}

// PriorityCalculator calculates alert priorities
type PriorityCalculator struct {
	logger *logging.StructuredLogger
	config *AlertRouterConfig
}

// ImpactAnalyzer analyzes business impact of alerts
type ImpactAnalyzer struct {
	logger         *logging.StructuredLogger
	impactDatabase map[string]ImpactProfile
	mu             sync.RWMutex
}

// ImpactProfile defines impact characteristics for components/services
type ImpactProfile struct {
	Component           string   `json:"component"`
	ServiceTier         string   `json:"service_tier"`
	UserImpactFactor    float64  `json:"user_impact_factor"`
	RevenueImpactFactor float64  `json:"revenue_impact_factor"`
	DependentServices   []string `json:"dependent_services"`
}

// GeographicRouter handles geographic and timezone-based routing
type GeographicRouter struct {
	logger    *logging.StructuredLogger
	regions   map[string]RegionInfo
	fallbacks []string
}

// RegionInfo contains information about a geographic region
type RegionInfo struct {
	Name          string       `json:"name"`
	Timezone      string       `json:"timezone"`
	OnCallTeams   []string     `json:"oncall_teams"`
	BusinessHours ScheduleInfo `json:"business_hours"`
}

// ScheduleInfo defines schedule information
type ScheduleInfo struct {
	Start    int   `json:"start"`    // Hour of day (0-23)
	End      int   `json:"end"`      // Hour of day (0-23)
	Weekdays []int `json:"weekdays"` // Days of week (0=Sunday)
}

// TimezoneManager manages timezone-aware operations
type TimezoneManager struct {
	logger    *logging.StructuredLogger
	timezones map[string]*time.Location
}

// TimezoneInfo provides timezone context
type TimezoneInfo struct {
	Timezone      string    `json:"timezone"`
	LocalTime     time.Time `json:"local_time"`
	BusinessHours bool      `json:"business_hours"`
	OnCallTeam    string    `json:"oncall_team,omitempty"`
}

// ContextEnricher enriches alerts with additional context
type ContextEnricher struct {
	logger *logging.StructuredLogger
	config *AlertRouterConfig
}

// RunbookManager manages runbook actions and recommendations
type RunbookManager struct {
	logger   *logging.StructuredLogger
	runbooks map[string]Runbook
}

// Runbook defines automated and manual actions for alerts
type Runbook struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Actions       []RunbookAction `json:"actions"`
	Prerequisites []string        `json:"prerequisites"`
}

// RunbookAction defines a specific action in a runbook
type RunbookAction struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"` // manual, automated, diagnostic
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Command     string            `json:"command,omitempty"`
	Parameters  map[string]string `json:"parameters,omitempty"`
	Timeout     time.Duration     `json:"timeout,omitempty"`
}

// NotificationChannel represents a notification destination
type NotificationChannel struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Type      string            `json:"type"` // slack, email, webhook, pagerduty, teams
	Config    map[string]string `json:"config"`
	Enabled   bool              `json:"enabled"`
	RateLimit *RateLimit        `json:"rate_limit,omitempty"`
	Filters   []AlertFilter     `json:"filters,omitempty"`
}

// AlertFilter filters alerts for notification channels
type AlertFilter struct {
	Field    string `json:"field"`    // severity, component, sla_type
	Operator string `json:"operator"` // equals, contains, matches, greater_than
	Value    string `json:"value"`
	Negate   bool   `json:"negate"`
}

// RoutingStatistics tracks routing performance metrics
type RoutingStatistics struct {
	TotalAlertsRouted     int64            `json:"total_alerts_routed"`
	AlertsDeduped         int64            `json:"alerts_deduped"`
	AlertsCorrelated      int64            `json:"alerts_correlated"`
	NotificationsSent     int64            `json:"notifications_sent"`
	NotificationsFailed   int64            `json:"notifications_failed"`
	AverageRoutingLatency time.Duration    `json:"average_routing_latency"`
	RuleMatchCounts       map[string]int64 `json:"rule_match_counts"`
}

// AlertRouterMetrics contains Prometheus metrics for the alert router
type AlertRouterMetrics struct {
	AlertsRouted        *prometheus.CounterVec
	AlertsDeduped       *prometheus.CounterVec
	NotificationsSent   *prometheus.CounterVec
	NotificationsFailed *prometheus.CounterVec
	RoutingLatency      *prometheus.HistogramVec
	RuleMatches         *prometheus.CounterVec
	QueueDepth          prometheus.Gauge
}

// DefaultAlertRouterConfig returns production-ready alert router configuration
func DefaultAlertRouterConfig() *AlertRouterConfig {
	return &AlertRouterConfig{
		// Core routing settings
		MaxConcurrentAlerts: 100,
		DeduplicationWindow: 5 * time.Minute,
		CorrelationWindow:   10 * time.Minute,

		// Queue management
		AlertQueueSize:    1000,
		ProcessingWorkers: 5,
		MaxRetries:        3,
		RetryBackoff:      30 * time.Second,

		// Notification settings
		NotificationTimeout: 30 * time.Second,
		MaxNotificationRate: 50, // per minute
		BatchNotifications:  true,
		BatchWindow:         2 * time.Minute,

		// Geographic routing
		EnableGeographicRouting: true,
		DefaultTimezone:         "UTC",
		BusinessHoursStart:      9,  // 9 AM
		BusinessHoursEnd:        17, // 5 PM

		// Performance settings
		MetricCacheSize:   500,
		EnrichmentTimeout: 10 * time.Second,
	}
}

// NewAlertRouter creates a new intelligent alert router
func NewAlertRouter(config *AlertRouterConfig, logger *logging.StructuredLogger) (*AlertRouter, error) {
	if config == nil {
		config = DefaultAlertRouterConfig()
	}

	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	// Initialize metrics
	metrics := &AlertRouterMetrics{
		AlertsRouted: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "alert_router_alerts_routed_total",
			Help: "Total number of alerts routed",
		}, []string{"rule", "channel", "result"}),

		AlertsDeduped: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "alert_router_alerts_deduped_total",
			Help: "Total number of alerts deduplicated",
		}, []string{"fingerprint_type"}),

		NotificationsSent: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "alert_router_notifications_sent_total",
			Help: "Total number of notifications sent",
		}, []string{"channel", "type"}),

		NotificationsFailed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "alert_router_notifications_failed_total",
			Help: "Total number of notifications failed",
		}, []string{"channel", "reason"}),

		RoutingLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "alert_router_routing_latency_seconds",
			Help:    "Alert routing latency in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
		}, []string{"stage"}),

		RuleMatches: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "alert_router_rule_matches_total",
			Help: "Total number of rule matches",
		}, []string{"rule_name"}),

		QueueDepth: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "alert_router_queue_depth",
			Help: "Number of alerts in the routing queue",
		}),
	}

	// Register metrics with duplicate handling
	routerMetrics := []prometheus.Collector{
		metrics.AlertsRouted,
		metrics.AlertsDeduped,
		metrics.NotificationsSent,
		metrics.NotificationsFailed,
		metrics.RoutingLatency,
		metrics.RuleMatches,
		metrics.QueueDepth,
	}

	// Register each metric, ignoring duplicate registration errors
	for _, metric := range routerMetrics {
		if err := prometheus.Register(metric); err != nil {
			if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
				// Only propagate non-duplicate errors
				logger.Error("Failed to register router metric", "error", err)
			}
		}
	}

	ar := &AlertRouter{
		logger:               logger.WithComponent("alert-router"),
		config:               config,
		routingRules:         make([]*RoutingRule, 0),
		notificationChannels: make(map[string]NotificationChannel),
		alertFingerprints:    make(map[string]*AlertGroup),
		metrics:              metrics,
		routingStats: &RoutingStatistics{
			RuleMatchCounts: make(map[string]int64),
		},
		stopCh:     make(chan struct{}),
		alertQueue: make(chan *EnrichedAlert, config.AlertQueueSize),
	}

	// Initialize sub-components
	ar.correlationEngine = &CorrelationEngine{
		logger:       logger.WithComponent("correlation-engine"),
		config:       config,
		correlations: make(map[string]*AlertCorrelation),
	}

	ar.priorityCalculator = &PriorityCalculator{
		logger: logger.WithComponent("priority-calculator"),
		config: config,
	}

	ar.impactAnalyzer = &ImpactAnalyzer{
		logger:         logger.WithComponent("impact-analyzer"),
		impactDatabase: make(map[string]ImpactProfile),
	}

	ar.geographicRouter = &GeographicRouter{
		logger:    logger.WithComponent("geographic-router"),
		regions:   make(map[string]RegionInfo),
		fallbacks: []string{"default"},
	}

	ar.timezoneManager = &TimezoneManager{
		logger:    logger.WithComponent("timezone-manager"),
		timezones: make(map[string]*time.Location),
	}

	ar.contextEnricher = &ContextEnricher{
		logger: logger.WithComponent("context-enricher"),
		config: config,
	}

	ar.runbookManager = &RunbookManager{
		logger:   logger.WithComponent("runbook-manager"),
		runbooks: make(map[string]Runbook),
	}

	// Load default routing rules and channels
	ar.loadDefaultRoutingRules()
	ar.loadDefaultNotificationChannels()
	ar.loadDefaultImpactProfiles()

	return ar, nil
}

// Start initializes the alert router
func (ar *AlertRouter) Start(ctx context.Context) error {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	if ar.started {
		return fmt.Errorf("alert router already started")
	}

	ar.logger.InfoWithContext("Starting alert router",
		"routing_rules", len(ar.routingRules),
		"notification_channels", len(ar.notificationChannels),
		"processing_workers", ar.config.ProcessingWorkers,
	)

	// Start processing workers
	for i := 0; i < ar.config.ProcessingWorkers; i++ {
		go ar.processingWorker(ctx, i)
	}

	// Start background processes
	go ar.deduplicationCleanupLoop(ctx)
	go ar.metricsUpdateLoop(ctx)

	ar.started = true
	ar.logger.InfoWithContext("Alert router started successfully")

	return nil
}

// Stop shuts down the alert router
func (ar *AlertRouter) Stop(ctx context.Context) error {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	if !ar.started {
		return nil
	}

	ar.logger.InfoWithContext("Stopping alert router")
	close(ar.stopCh)
	close(ar.alertQueue)

	ar.started = false
	ar.logger.InfoWithContext("Alert router stopped")

	return nil
}

// Route processes and routes an SLA alert
func (ar *AlertRouter) Route(ctx context.Context, alert *SLAAlert) error {
	startTime := time.Now()

	// Enrich the alert with additional context
	enrichedAlert, err := ar.enrichAlert(ctx, alert)
	if err != nil {
		ar.logger.ErrorWithContext("Failed to enrich alert", err,
			slog.String("alert_id", alert.ID),
		)
		// Continue with non-enriched alert
		enrichedAlert = &EnrichedAlert{
			SLAAlert:          alert,
			ProcessedAt:       time.Now(),
			EnrichmentLatency: time.Since(startTime),
		}
	}

	// Add to processing queue
	select {
	case ar.alertQueue <- enrichedAlert:
		ar.metrics.QueueDepth.Set(float64(len(ar.alertQueue)))
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("alert routing queue is full")
	}
}

// processingWorker processes alerts from the queue
func (ar *AlertRouter) processingWorker(ctx context.Context, workerID int) {
	ar.logger.DebugWithContext("Starting alert processing worker",
		slog.Int("worker_id", workerID),
	)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ar.stopCh:
			return
		case enrichedAlert, ok := <-ar.alertQueue:
			if !ok {
				return
			}

			ar.processAlert(ctx, enrichedAlert, workerID)
			ar.metrics.QueueDepth.Set(float64(len(ar.alertQueue)))
		}
	}
}

// processAlert processes a single enriched alert
func (ar *AlertRouter) processAlert(ctx context.Context, enrichedAlert *EnrichedAlert, workerID int) {
	startTime := time.Now()
	defer func() {
		routingLatency := time.Since(startTime)
		enrichedAlert.RoutingLatency = routingLatency
		ar.metrics.RoutingLatency.WithLabelValues("total").Observe(routingLatency.Seconds())
	}()

	ar.logger.DebugWithContext("Processing alert",
		slog.String("alert_id", enrichedAlert.ID),
		slog.Int("worker_id", workerID),
	)

	// Step 1: Deduplication
	if deduplicated := ar.deduplicateAlert(enrichedAlert); deduplicated {
		ar.metrics.AlertsDeduped.WithLabelValues("duplicate").Inc()
		ar.logger.DebugWithContext("Alert deduplicated",
			slog.String("alert_id", enrichedAlert.ID),
		)
		return
	}

	// Step 2: Correlation with existing alerts
	ar.correlateAlert(enrichedAlert.SLAAlert)

	// Step 3: Priority calculation
	priority, _ := ar.priorityCalculator.CalculatePriority(enrichedAlert.SLAAlert)
	enrichedAlert.Priority = ar.convertPriorityToInt(priority)

	// Step 4: Business impact analysis
	impact, _ := ar.impactAnalyzer.AnalyzeImpact(enrichedAlert.SLAAlert)
	businessImpact := BusinessImpactScore{
		OverallScore:     50.0,
		UserImpact:       40.0,
		RevenueImpact:    30.0,
		ReputationImpact: 20.0,
		ServiceTier:      impact.Severity,
	}
	enrichedAlert.BusinessImpact = businessImpact

	// Step 5: Apply routing rules
	routingDecision, err := ar.applyRoutingRules(enrichedAlert)
	if err != nil {
		ar.logger.ErrorWithContext("Failed to apply routing rules", err,
			slog.String("alert_id", enrichedAlert.ID),
		)
		// Use fallback routing
		routingDecision = &RoutingDecision{
			MatchedRules:     []string{"fallback"},
			SelectedChannels: []string{"default"},
			Suppressed:       false,
		}
	}
	enrichedAlert.RoutingDecision = routingDecision

	// Step 6: Check for suppression
	if routingDecision.Suppressed {
		ar.logger.InfoWithContext("Alert suppressed",
			slog.String("alert_id", enrichedAlert.ID),
			slog.String("reason", routingDecision.SuppressionReason),
		)
		return
	}

	// Step 7: Send notifications
	ar.sendNotifications(ctx, enrichedAlert)

	ar.routingStats.TotalAlertsRouted++
}

// enrichAlert enriches the alert with additional context
func (ar *AlertRouter) enrichAlert(ctx context.Context, alert *SLAAlert) (*EnrichedAlert, error) {
	enrichmentCtx, cancel := context.WithTimeout(ctx, ar.config.EnrichmentTimeout)
	defer cancel()

	enriched := &EnrichedAlert{
		SLAAlert:    alert,
		ProcessedAt: time.Now(),
	}

	// Add related incidents
	relatedIncidents, err := ar.contextEnricher.FindRelatedIncidents(enrichmentCtx, alert)
	if err != nil {
		ar.logger.WarnWithContext("Failed to find related incidents",
			slog.String("alert_id", alert.ID),
			slog.String("error", err.Error()),
		)
	} else {
		// Convert incidents to string array for backward compatibility
		var incidentIDs []string
		for _, incident := range relatedIncidents {
			incidentIDs = append(incidentIDs, incident.ID)
		}
		enriched.RelatedIncidents = incidentIDs
	}

	// Add geographic context
	if ar.config.EnableGeographicRouting && ar.geographicRouter != nil {
		// TODO: Implement GetContextForAlert method
		enriched.AffectedRegions = []string{}
		enriched.TimezoneInfo = TimezoneInfo{
			Timezone:      "UTC",
			LocalTime:     time.Now(),
			BusinessHours: true,
		}
	}

	// Add runbook actions
	if ar.runbookManager != nil {
		// TODO: Implement GetActionsForAlert method
		enriched.RunbookActions = []RunbookAction{}
	}

	return enriched, nil
}

// deduplicateAlert checks if the alert is a duplicate and handles deduplication
func (ar *AlertRouter) deduplicateAlert(alert *EnrichedAlert) bool {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	fingerprint := ar.generateDeduplicationFingerprint(alert.SLAAlert)

	existingGroup, exists := ar.alertFingerprints[fingerprint]
	if !exists {
		// Create new alert group
		ar.alertFingerprints[fingerprint] = &AlertGroup{
			ID:             fmt.Sprintf("group-%x", md5.Sum([]byte(fingerprint))),
			Fingerprint:    fingerprint,
			Representative: alert.SLAAlert,
			Members:        []*SLAAlert{alert.SLAAlert},
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		return false
	}

	// Check if within deduplication window
	if time.Since(existingGroup.UpdatedAt) <= ar.config.DeduplicationWindow {
		// Add to existing group
		existingGroup.Members = append(existingGroup.Members, alert.SLAAlert)
		existingGroup.UpdatedAt = time.Now()

		// Update representative if new alert is more severe
		if ar.isMoreSevere(string(alert.Severity), string(existingGroup.Representative.Severity)) {
			existingGroup.Representative = alert.SLAAlert
		}

		return true // Deduplicated
	}

	// Outside window, create new group
	ar.alertFingerprints[fingerprint] = &AlertGroup{
		ID:             fmt.Sprintf("group-%x", md5.Sum([]byte(fmt.Sprintf("%s-%d", fingerprint, time.Now().Unix())))),
		Fingerprint:    fingerprint,
		Representative: alert.SLAAlert,
		Members:        []*SLAAlert{alert.SLAAlert},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return false
}

// generateDeduplicationFingerprint creates a fingerprint for deduplication
func (ar *AlertRouter) generateDeduplicationFingerprint(alert *SLAAlert) string {
	// Create fingerprint based on SLA type, component, and general characteristics
	components := []string{
		string(alert.SLAType),
		alert.Context.Component,
		alert.Context.Service,
		string(alert.Severity),
	}

	// Add relevant labels
	relevantLabels := []string{"region", "environment", "cluster"}
	for _, label := range relevantLabels {
		if value, exists := alert.Labels[label]; exists {
			components = append(components, fmt.Sprintf("%s=%s", label, value))
		}
	}

	sort.Strings(components)
	data := strings.Join(components, "|")
	return fmt.Sprintf("%x", md5.Sum([]byte(data)))
}

// applyRoutingRules applies routing rules to determine alert routing
func (ar *AlertRouter) applyRoutingRules(alert *EnrichedAlert) (*RoutingDecision, error) {
	decision := &RoutingDecision{
		MatchedRules:     make([]string, 0),
		SelectedChannels: make([]string, 0),
	}

	// Sort rules by priority (higher priority first)
	sortedRules := make([]*RoutingRule, len(ar.routingRules))
	copy(sortedRules, ar.routingRules)
	sort.Slice(sortedRules, func(i, j int) bool {
		return sortedRules[i].Priority > sortedRules[j].Priority
	})

	// Evaluate rules in priority order
	for _, rule := range sortedRules {
		if !rule.Enabled {
			continue
		}

		// Check if rule matches
		if ar.evaluateRuleConditions(alert.SLAAlert, rule.Conditions) {
			decision.MatchedRules = append(decision.MatchedRules, rule.Name)
			ar.routingStats.RuleMatchCounts[rule.Name]++
			ar.metrics.RuleMatches.WithLabelValues(rule.Name).Inc()

			// Process rule actions
			for _, action := range rule.Actions {
				switch action.Type {
				case "notify":
					if ar.isChannelAvailable(action.Target, alert.SLAAlert) {
						decision.SelectedChannels = append(decision.SelectedChannels, action.Target)
					}
				case "escalate":
					decision.EscalationPolicy = action.Target
				case "suppress":
					decision.Suppressed = true
					decision.SuppressionReason = action.Parameters["reason"]
				case "delay":
					if delayStr, exists := action.Parameters["duration"]; exists {
						if delay, err := time.ParseDuration(delayStr); err == nil {
							delayUntil := time.Now().Add(delay)
							decision.DelayUntil = &delayUntil
						}
					}
				}
			}

			// If rule has stop processing flag, break
			if stopProcessing, exists := rule.Actions[0].Parameters["stop_processing"]; exists && stopProcessing == "true" {
				break
			}
		}
	}

	// If no channels selected, use default
	if len(decision.SelectedChannels) == 0 && !decision.Suppressed {
		decision.SelectedChannels = []string{"default"}
	}

	return decision, nil
}

// sendNotifications sends notifications to selected channels
func (ar *AlertRouter) sendNotifications(ctx context.Context, alert *EnrichedAlert) {
	for _, channelName := range alert.RoutingDecision.SelectedChannels {
		channel, exists := ar.notificationChannels[channelName]
		if !exists {
			ar.logger.WarnWithContext("Notification channel not found",
				slog.String("channel", channelName),
				slog.String("alert_id", alert.ID),
			)
			continue
		}

		// Send notification asynchronously
		go ar.sendNotificationToChannel(ctx, alert, channel)
	}
}

// sendNotificationToChannel sends notification to a specific channel
func (ar *AlertRouter) sendNotificationToChannel(ctx context.Context, alert *EnrichedAlert, channel NotificationChannel) {
	notificationCtx, cancel := context.WithTimeout(ctx, ar.config.NotificationTimeout)
	defer cancel()

	// Apply channel-specific filters
	if !ar.passesChannelFilters(alert, channel.Filters) {
		ar.logger.DebugWithContext("Alert filtered out by channel",
			slog.String("alert_id", alert.ID),
			slog.String("channel", channel.Name),
		)
		return
	}

	// Check rate limits
	if channel.RateLimit != nil && !ar.checkRateLimit(channel.Name, *channel.RateLimit) {
		ar.logger.WarnWithContext("Channel rate limit exceeded",
			slog.String("channel", channel.Name),
			slog.String("alert_id", alert.ID),
		)
		ar.metrics.NotificationsFailed.WithLabelValues(channel.Name, "rate_limit").Inc()
		return
	}

	// Send notification based on channel type
	var err error
	switch channel.Type {
	case "slack":
		err = ar.sendSlackNotification(notificationCtx, alert, channel)
	case "email":
		err = ar.sendEmailNotification(notificationCtx, alert, channel)
	case "webhook":
		err = ar.sendWebhookNotification(notificationCtx, alert, channel)
	case "pagerduty":
		err = ar.sendPagerDutyNotification(notificationCtx, alert, channel)
	case "teams":
		err = ar.sendTeamsNotification(notificationCtx, alert, channel)
	default:
		err = fmt.Errorf("unsupported channel type: %s", channel.Type)
	}

	if err != nil {
		ar.logger.ErrorWithContext("Failed to send notification", err,
			slog.String("channel", channel.Name),
			slog.String("channel_type", channel.Type),
			slog.String("alert_id", alert.ID),
		)
		ar.metrics.NotificationsFailed.WithLabelValues(channel.Name, "send_error").Inc()
		ar.routingStats.NotificationsFailed++
	} else {
		ar.logger.DebugWithContext("Notification sent successfully",
			slog.String("channel", channel.Name),
			slog.String("channel_type", channel.Type),
			slog.String("alert_id", alert.ID),
		)
		ar.metrics.NotificationsSent.WithLabelValues(channel.Name, channel.Type).Inc()
		ar.routingStats.NotificationsSent++
	}
}

// Missing AlertRouter method implementations

func (ar *AlertRouter) loadDefaultRoutingRules() error {
	ar.logger.InfoWithContext("Loading default routing rules")
	// TODO: Implement default routing rules loading
	return nil
}

func (ar *AlertRouter) loadDefaultNotificationChannels() error {
	ar.logger.InfoWithContext("Loading default notification channels")
	// TODO: Implement default notification channels loading
	return nil
}

func (ar *AlertRouter) loadDefaultImpactProfiles() error {
	ar.logger.InfoWithContext("Loading default impact profiles")
	// TODO: Implement default impact profiles loading
	return nil
}

func (ar *AlertRouter) deduplicationCleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ar.logger.DebugWithContext("Running deduplication cleanup")
		}
	}
}

func (ar *AlertRouter) metricsUpdateLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ar.logger.DebugWithContext("Updating metrics")
		}
	}
}

func (ar *AlertRouter) correlateAlert(alert *SLAAlert) error {
	ar.logger.DebugWithContext("Correlating alert", "alertID", alert.ID)
	// TODO: Implement alert correlation
	return nil
}

func (ar *AlertRouter) getFallbackRouting() *RoutingRule {
	return &RoutingRule{Name: "fallback", Priority: 50}
}

// Helper methods for routing evaluation

func (ar *AlertRouter) evaluateRuleConditions(alert *SLAAlert, conditions []RoutingCondition) bool {
	for _, condition := range conditions {
		if !ar.evaluateCondition(alert, condition) {
			return false
		}
	}
	return true
}

func (ar *AlertRouter) evaluateCondition(alert *SLAAlert, condition RoutingCondition) bool {
	var value string

	switch condition.Field {
	case "sla_type":
		value = string(alert.SLAType)
	case "severity":
		value = string(alert.Severity)
	case "component":
		value = alert.Context.Component
	case "service":
		value = alert.Context.Service
	default:
		// Check labels
		if labelValue, exists := alert.Labels[condition.Field]; exists {
			value = labelValue
		} else {
			return condition.Negate
		}
	}

	matches := false
	for _, expectedValue := range condition.Values {
		switch condition.Operator {
		case "equals":
			if value == expectedValue {
				matches = true
				break
			}
		case "contains":
			if strings.Contains(value, expectedValue) {
				matches = true
				break
			}
		case "matches":
			// Simple regex matching (simplified)
			if strings.Contains(value, expectedValue) {
				matches = true
				break
			}
		}
	}

	if condition.Negate {
		return !matches
	}
	return matches
}

func (ar *AlertRouter) isChannelAvailable(channelName string, alert *SLAAlert) bool {
	channel, exists := ar.notificationChannels[channelName]
	if !exists || !channel.Enabled {
		return false
	}

	// Apply channel filters
	return ar.passesChannelFilters(&EnrichedAlert{SLAAlert: alert}, channel.Filters)
}

func (ar *AlertRouter) passesChannelFilters(alert *EnrichedAlert, filters []AlertFilter) bool {
	for _, filter := range filters {
		if !ar.evaluateFilter(alert, filter) {
			return false
		}
	}
	return true
}

func (ar *AlertRouter) evaluateFilter(alert *EnrichedAlert, filter AlertFilter) bool {
	var value string

	switch filter.Field {
	case "severity":
		value = string(alert.Severity)
	case "component":
		value = alert.Context.Component
	case "sla_type":
		value = string(alert.SLAType)
	default:
		if labelValue, exists := alert.Labels[filter.Field]; exists {
			value = labelValue
		} else {
			return filter.Negate
		}
	}

	matches := false
	switch filter.Operator {
	case "equals":
		matches = value == filter.Value
	case "contains":
		matches = strings.Contains(value, filter.Value)
	case "matches":
		matches = strings.Contains(value, filter.Value) // Simplified regex
	}

	if filter.Negate {
		return !matches
	}
	return matches
}

func (ar *AlertRouter) checkRateLimit(channelName string, limit RateLimit) bool {
	// Simplified rate limiting - in production would use more sophisticated algorithm
	return true
}

func (ar *AlertRouter) isMoreSevere(a, b string) bool {
	severityOrder := map[string]int{
		"info":     1,
		"warning":  2,
		"major":    3,
		"critical": 4,
		"urgent":   5,
	}

	return severityOrder[a] > severityOrder[b]
}

func (ar *AlertRouter) convertPriorityToInt(priority string) int {
	switch priority {
	case "low":
		return 1
	case "medium":
		return 2
	case "high":
		return 3
	case "urgent":
		return 4
	default:
		return 2
	}
}

// Notification sending methods (simplified implementations)

func (ar *AlertRouter) sendSlackNotification(ctx context.Context, alert *EnrichedAlert, channel NotificationChannel) error {
	ar.logger.InfoWithContext("Sending Slack notification",
		slog.String("alert_id", alert.ID),
		slog.String("channel", channel.Name),
	)
	return nil
}

func (ar *AlertRouter) sendEmailNotification(ctx context.Context, alert *EnrichedAlert, channel NotificationChannel) error {
	ar.logger.InfoWithContext("Sending email notification",
		slog.String("alert_id", alert.ID),
		slog.String("channel", channel.Name),
	)
	return nil
}

func (ar *AlertRouter) sendWebhookNotification(ctx context.Context, alert *EnrichedAlert, channel NotificationChannel) error {
	ar.logger.InfoWithContext("Sending webhook notification",
		slog.String("alert_id", alert.ID),
		slog.String("channel", channel.Name),
	)
	return nil
}

func (ar *AlertRouter) sendPagerDutyNotification(ctx context.Context, alert *EnrichedAlert, channel NotificationChannel) error {
	ar.logger.InfoWithContext("Sending PagerDuty notification",
		slog.String("alert_id", alert.ID),
		slog.String("channel", channel.Name),
	)
	return nil
}

func (ar *AlertRouter) sendTeamsNotification(ctx context.Context, alert *EnrichedAlert, channel NotificationChannel) error {
	ar.logger.InfoWithContext("Sending Teams notification",
		slog.String("alert_id", alert.ID),
		slog.String("channel", channel.Name),
	)
	return nil
}

// Missing helper type method implementations

func (pc *PriorityCalculator) CalculatePriority(alert *SLAAlert) (string, error) {
	if alert.Severity == "critical" {
		return "high", nil
	}
	return "medium", nil
}

func (ia *ImpactAnalyzer) AnalyzeImpact(alert *SLAAlert) (*ImpactAnalysis, error) {
	return &ImpactAnalysis{Severity: string(alert.Severity), AffectedServices: []string{}}, nil
}

func (ce *ContextEnricher) FindRelatedIncidents(ctx context.Context, alert *SLAAlert) ([]*Incident, error) {
	return []*Incident{}, nil
}

// Helper types for the stubs above

type ImpactAnalysis struct {
	Severity         string   `json:"severity"`
	AffectedServices []string `json:"affected_services"`
}

type Incident struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}
