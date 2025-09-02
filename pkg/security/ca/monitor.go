package ca

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// CAMonitor monitors Certificate Authority health and performance.

type CAMonitor struct {
	config *MonitoringConfig

	logger *logging.StructuredLogger

	metrics *CAMetrics

	alerts *AlertManager

	checks map[string]HealthCheck

	mu sync.RWMutex

	ctx context.Context

	cancel context.CancelFunc
}

// CAMetrics holds Prometheus metrics for CA operations.

type CAMetrics struct {
	// Certificate metrics.

	CertificatesIssued *prometheus.CounterVec

	CertificatesRevoked *prometheus.CounterVec

	CertificatesExpired *prometheus.CounterVec

	CertificateLifetime *prometheus.HistogramVec

	// Backend metrics.

	BackendHealth *prometheus.GaugeVec

	BackendOperations *prometheus.CounterVec

	BackendLatency *prometheus.HistogramVec

	BackendErrors *prometheus.CounterVec

	// Pool metrics.

	CertificatePoolSize *prometheus.GaugeVec

	PoolOperations *prometheus.CounterVec

	PoolCacheHitRate *prometheus.GaugeVec

	// Distribution metrics.

	DistributionJobs *prometheus.CounterVec

	DistributionLatency *prometheus.HistogramVec

	DistributionErrors *prometheus.CounterVec

	// Policy metrics.

	PolicyValidations *prometheus.CounterVec

	PolicyViolations *prometheus.CounterVec

	ApprovalRequests *prometheus.CounterVec

	// System metrics.

	ActiveSessions *prometheus.GaugeVec

	ResourceUsage *prometheus.GaugeVec

	ErrorRate *prometheus.GaugeVec
}

// AlertManager manages certificate-related alerts.

type AlertManager struct {
	config *AlertConfig

	logger *logging.StructuredLogger

	rules map[string]*AlertRule

	activeAlerts map[string]*Alert

	mu sync.RWMutex
}

// AlertConfig configures alert management.

type AlertConfig struct {
	Enabled bool `yaml:"enabled"`

	Rules []*AlertRule `yaml:"rules"`

	NotificationChannels []NotificationChannel `yaml:"notification_channels"`

	Thresholds *AlertThresholds `yaml:"thresholds"`

	Cooldown time.Duration `yaml:"cooldown"`

	MaxAlerts int `yaml:"max_alerts"`
}

// AlertRule defines an alert rule.

type AlertRule struct {
	Name string `yaml:"name"`

	Description string `yaml:"description"`

	Condition string `yaml:"condition"`

	Severity AlertSeverity `yaml:"severity"`

	Threshold float64 `yaml:"threshold"`

	Duration time.Duration `yaml:"duration"`

	Labels map[string]string `yaml:"labels"`

	Annotations map[string]string `yaml:"annotations"`

	Actions []AlertAction `yaml:"actions"`
}

// AlertSeverity represents alert severity levels.

type AlertSeverity string

const (

	// SeverityInfo holds severityinfo value.

	SeverityInfo AlertSeverity = "info"

	// SeverityWarning holds severitywarning value.

	SeverityWarning AlertSeverity = "warning"

	// SeverityError holds severityerror value.

	SeverityError AlertSeverity = "error"

	// SeverityCritical holds severitycritical value.

	SeverityCritical AlertSeverity = "critical"
)

// AlertAction defines what to do when an alert fires.

type AlertAction struct {
	Type string `yaml:"type"`

	Config map[string]string `yaml:"config"`
}

// NotificationChannel defines how to send alerts.

type NotificationChannel struct {
	Name string `yaml:"name"`

	Type string `yaml:"type"` // email, slack, webhook, pagerduty

	Config map[string]string `yaml:"config"`
}

// AlertThresholds defines default alert thresholds.

type AlertThresholds struct {
	CertificateExpiryDays int `yaml:"certificate_expiry_days"`

	BackendErrorRate float64 `yaml:"backend_error_rate"`

	PolicyViolationRate float64 `yaml:"policy_violation_rate"`

	DistributionFailureRate float64 `yaml:"distribution_failure_rate"`

	SystemResourceUsage float64 `yaml:"system_resource_usage"`
}

// Alert represents an active alert.

type Alert struct {
	ID string `json:"id"`

	Rule *AlertRule `json:"rule"`

	Status AlertStatus `json:"status"`

	StartsAt time.Time `json:"starts_at"`

	EndsAt time.Time `json:"ends_at,omitempty"`

	UpdatedAt time.Time `json:"updated_at"`

	Labels map[string]string `json:"labels"`

	Annotations map[string]string `json:"annotations"`

	Value float64 `json:"value"`

	GeneratorURL string `json:"generator_url,omitempty"`
}

// AlertStatus represents the status of an alert.

type AlertStatus string

const (

	// AlertStatusFiring holds alertstatusfiring value.

	AlertStatusFiring AlertStatus = "firing"

	// AlertStatusResolved holds alertstatusresolved value.

	AlertStatusResolved AlertStatus = "resolved"

	// AlertStatusSuppressed holds alertstatussuppressed value.

	AlertStatusSuppressed AlertStatus = "suppressed"
)

// HealthCheck represents a health check function.

type HealthCheck interface {
	Name() string

	Check(ctx context.Context) error

	Critical() bool
}

// BackendHealthCheck checks backend health.

type BackendHealthCheck struct {
	name string

	backend Backend
}

// Name performs name operation.

func (h *BackendHealthCheck) Name() string { return h.name }

// Critical performs critical operation.

func (h *BackendHealthCheck) Critical() bool { return true }

// Check performs check operation.

func (h *BackendHealthCheck) Check(ctx context.Context) error {
	return h.backend.HealthCheck(ctx)
}

// CertificateExpiryCheck checks for expiring certificates.

type CertificateExpiryCheck struct {
	pool *CertificatePool

	threshold time.Duration
}

// Name performs name operation.

func (h *CertificateExpiryCheck) Name() string { return "certificate_expiry" }

// Critical performs critical operation.

func (h *CertificateExpiryCheck) Critical() bool { return false }

// Check performs check operation.

func (h *CertificateExpiryCheck) Check(ctx context.Context) error {
	threshold := time.Now().Add(h.threshold)

	expiring, err := h.pool.GetExpiringCertificates(threshold)
	if err != nil {
		return err
	}

	if len(expiring) > 0 {
		return fmt.Errorf("found %d certificates expiring within %v", len(expiring), h.threshold)
	}

	return nil
}

// NewCAMonitor creates a new CA monitor.

func NewCAMonitor(config *MonitoringConfig, logger *logging.StructuredLogger) (*CAMonitor, error) {
	ctx, cancel := context.WithCancel(context.Background())

	monitor := &CAMonitor{
		config: config,

		logger: logger,

		checks: make(map[string]HealthCheck),

		ctx: ctx,

		cancel: cancel,
	}

	// Initialize metrics.

	monitor.metrics = initializeMetrics()

	// Initialize alert manager.

	if config.AlertingEnabled {

		alertConfig := &AlertConfig{
			Enabled: true,

			Thresholds: &AlertThresholds{
				CertificateExpiryDays: config.ExpirationWarningDays,

				BackendErrorRate: 0.05, // 5%

				PolicyViolationRate: 0.10, // 10%

				DistributionFailureRate: 0.05, // 5%

				SystemResourceUsage: 0.80, // 80%

			},

			Cooldown: 5 * time.Minute,

			MaxAlerts: 100,
		}

		alertManager, err := NewAlertManager(alertConfig, logger)
		if err != nil {

			cancel()

			return nil, fmt.Errorf("failed to initialize alert manager: %w", err)

		}

		monitor.alerts = alertManager

	}

	return monitor, nil
}

// Start starts the CA monitor.

func (m *CAMonitor) Start(ctx context.Context) {
	m.logger.Info("starting CA monitor")

	// Start health check loop.

	go m.runHealthChecks()

	// Start metrics collection loop.

	go m.runMetricsCollection()

	// Start alert processing loop.

	if m.alerts != nil {
		go m.alerts.Start(ctx)
	}

	<-ctx.Done()

	m.logger.Info("CA monitor stopped")
}

// Stop stops the CA monitor.

func (m *CAMonitor) Stop() {
	m.logger.Info("stopping CA monitor")

	m.cancel()
}

// AddHealthCheck adds a health check.

func (m *CAMonitor) AddHealthCheck(check HealthCheck) {
	m.mu.Lock()

	defer m.mu.Unlock()

	m.checks[check.Name()] = check

	m.logger.Info("health check added",

		"name", check.Name(),

		"critical", check.Critical())
}

// RemoveHealthCheck removes a health check.

func (m *CAMonitor) RemoveHealthCheck(name string) {
	m.mu.Lock()

	defer m.mu.Unlock()

	delete(m.checks, name)

	m.logger.Info("health check removed", "name", name)
}

// RecordCertificateIssued records a certificate issuance.

func (m *CAMonitor) RecordCertificateIssued(backend, tenantID string, duration time.Duration) {
	m.metrics.CertificatesIssued.WithLabelValues(backend, tenantID).Inc()

	m.metrics.CertificateLifetime.WithLabelValues(backend, tenantID).Observe(duration.Hours())
}

// RecordCertificateRevoked records a certificate revocation.

func (m *CAMonitor) RecordCertificateRevoked(backend, tenantID, reason string) {
	m.metrics.CertificatesRevoked.WithLabelValues(backend, tenantID, reason).Inc()
}

// RecordBackendOperation records a backend operation.

func (m *CAMonitor) RecordBackendOperation(backend, operation string, success bool, duration time.Duration) {
	status := "success"

	if !success {

		status = "error"

		m.metrics.BackendErrors.WithLabelValues(backend, operation).Inc()

	}

	m.metrics.BackendOperations.WithLabelValues(backend, operation, status).Inc()

	m.metrics.BackendLatency.WithLabelValues(backend, operation).Observe(duration.Seconds())
}

// RecordDistributionJob records a distribution job.

func (m *CAMonitor) RecordDistributionJob(status, targetType string, duration time.Duration) {
	m.metrics.DistributionJobs.WithLabelValues(status, targetType).Inc()

	m.metrics.DistributionLatency.WithLabelValues(targetType).Observe(duration.Seconds())

	if status == "failed" {
		m.metrics.DistributionErrors.WithLabelValues(targetType, "distribution_failed").Inc()
	}
}

// RecordPolicyValidation records a policy validation.

func (m *CAMonitor) RecordPolicyValidation(template string, valid bool, violationCount int) {
	status := "passed"

	if !valid {

		status = "failed"

		m.metrics.PolicyViolations.WithLabelValues(template, "validation_failed").Add(float64(violationCount))

	}

	m.metrics.PolicyValidations.WithLabelValues(template, status).Inc()
}

// UpdateBackendHealth updates backend health status.

func (m *CAMonitor) UpdateBackendHealth(backend string, healthy bool) {
	value := float64(0)

	if healthy {
		value = 1
	}

	m.metrics.BackendHealth.WithLabelValues(backend).Set(value)
}

// UpdatePoolMetrics updates certificate pool metrics.

func (m *CAMonitor) UpdatePoolMetrics(poolSize int, hitRate float64) {
	m.metrics.CertificatePoolSize.WithLabelValues("active").Set(float64(poolSize))

	m.metrics.PoolCacheHitRate.WithLabelValues("memory").Set(hitRate)
}

// GetHealthStatus returns overall health status.

func (m *CAMonitor) GetHealthStatus() map[string]interface{} {
	m.mu.RLock()

	defer m.mu.RUnlock()

	status := map[string]interface{}{
		"healthy": true,

		"checks": make(map[string]interface{}),

		"last_check": time.Now(),

		"alert_count": 0,
	}

	// Check all health checks.

	for name, check := range m.checks {

		err := check.Check(m.ctx)

		checkStatus := map[string]interface{}{
			"healthy": err == nil,

			"critical": check.Critical(),
		}

		if err != nil {

			checkStatus["error"] = err.Error()

			if check.Critical() {
				status["healthy"] = false
			}

		}

		status["checks"].(map[string]interface{})[name] = checkStatus

	}

	// Add alert count.

	if m.alerts != nil {
		status["alert_count"] = len(m.alerts.GetActiveAlerts())
	}

	return status
}

// Helper methods.

func (m *CAMonitor) runHealthChecks() {
	ticker := time.NewTicker(m.config.HealthCheckInterval)

	defer ticker.Stop()

	for {
		select {

		case <-m.ctx.Done():

			return

		case <-ticker.C:

			m.performHealthChecks()

		}
	}
}

func (m *CAMonitor) performHealthChecks() {
	m.mu.RLock()

	checks := make(map[string]HealthCheck)

	for name, check := range m.checks {
		checks[name] = check
	}

	m.mu.RUnlock()

	for name, check := range checks {

		start := time.Now()

		err := check.Check(m.ctx)

		duration := time.Since(start)

		if err != nil {

			m.logger.Warn("health check failed",

				"check", name,

				"error", err,

				"duration", duration,

				"critical", check.Critical())

			// Trigger alert if enabled.

			if m.alerts != nil {
				m.alerts.TriggerAlert("health_check_failed", map[string]string{
					"check": name,

					"error": err.Error(),

					"critical": fmt.Sprintf("%v", check.Critical()),
				})
			}

		} else {
			m.logger.Debug("health check passed",

				"check", name,

				"duration", duration)
		}

	}
}

func (m *CAMonitor) runMetricsCollection() {
	ticker := time.NewTicker(30 * time.Second) // Collect metrics every 30 seconds

	defer ticker.Stop()

	for {
		select {

		case <-m.ctx.Done():

			return

		case <-ticker.C:

			m.collectSystemMetrics()

		}
	}
}

func (m *CAMonitor) collectSystemMetrics() {
	// Collect system resource usage.

	// This would typically use runtime metrics, cgroup stats, etc.

	m.logger.Debug("collecting system metrics")

	// Example metrics collection (would be replaced with actual implementation).

	m.metrics.ResourceUsage.WithLabelValues("cpu").Set(0.25) // 25% CPU

	m.metrics.ResourceUsage.WithLabelValues("memory").Set(0.40) // 40% Memory
}

// Initialize Prometheus metrics.

func initializeMetrics() *CAMetrics {
	return &CAMetrics{
		CertificatesIssued: promauto.NewCounterVec(

			prometheus.CounterOpts{
				Name: "ca_certificates_issued_total",

				Help: "Total number of certificates issued",
			},

			[]string{"backend", "tenant_id"},
		),

		CertificatesRevoked: promauto.NewCounterVec(

			prometheus.CounterOpts{
				Name: "ca_certificates_revoked_total",

				Help: "Total number of certificates revoked",
			},

			[]string{"backend", "tenant_id", "reason"},
		),

		CertificatesExpired: promauto.NewCounterVec(

			prometheus.CounterOpts{
				Name: "ca_certificates_expired_total",

				Help: "Total number of certificates expired",
			},

			[]string{"backend", "tenant_id"},
		),

		CertificateLifetime: promauto.NewHistogramVec(

			prometheus.HistogramOpts{
				Name: "ca_certificate_lifetime_hours",

				Help: "Certificate lifetime in hours",

				Buckets: prometheus.ExponentialBuckets(24, 2, 10), // 1 day to ~1 year

			},

			[]string{"backend", "tenant_id"},
		),

		BackendHealth: promauto.NewGaugeVec(

			prometheus.GaugeOpts{
				Name: "ca_backend_healthy",

				Help: "Backend health status (1=healthy, 0=unhealthy)",
			},

			[]string{"backend"},
		),

		BackendOperations: promauto.NewCounterVec(

			prometheus.CounterOpts{
				Name: "ca_backend_operations_total",

				Help: "Total number of backend operations",
			},

			[]string{"backend", "operation", "status"},
		),

		BackendLatency: promauto.NewHistogramVec(

			prometheus.HistogramOpts{
				Name: "ca_backend_operation_duration_seconds",

				Help: "Backend operation duration in seconds",
			},

			[]string{"backend", "operation"},
		),

		BackendErrors: promauto.NewCounterVec(

			prometheus.CounterOpts{
				Name: "ca_backend_errors_total",

				Help: "Total number of backend errors",
			},

			[]string{"backend", "operation"},
		),

		CertificatePoolSize: promauto.NewGaugeVec(

			prometheus.GaugeOpts{
				Name: "ca_certificate_pool_size",

				Help: "Current certificate pool size",
			},

			[]string{"pool_type"},
		),

		PoolOperations: promauto.NewCounterVec(

			prometheus.CounterOpts{
				Name: "ca_pool_operations_total",

				Help: "Total number of certificate pool operations",
			},

			[]string{"operation", "status"},
		),

		PoolCacheHitRate: promauto.NewGaugeVec(

			prometheus.GaugeOpts{
				Name: "ca_pool_cache_hit_rate",

				Help: "Certificate pool cache hit rate",
			},

			[]string{"cache_type"},
		),

		DistributionJobs: promauto.NewCounterVec(

			prometheus.CounterOpts{
				Name: "ca_distribution_jobs_total",

				Help: "Total number of certificate distribution jobs",
			},

			[]string{"status", "target_type"},
		),

		DistributionLatency: promauto.NewHistogramVec(

			prometheus.HistogramOpts{
				Name: "ca_distribution_duration_seconds",

				Help: "Certificate distribution duration in seconds",
			},

			[]string{"target_type"},
		),

		DistributionErrors: promauto.NewCounterVec(

			prometheus.CounterOpts{
				Name: "ca_distribution_errors_total",

				Help: "Total number of distribution errors",
			},

			[]string{"target_type", "error_type"},
		),

		PolicyValidations: promauto.NewCounterVec(

			prometheus.CounterOpts{
				Name: "ca_policy_validations_total",

				Help: "Total number of policy validations",
			},

			[]string{"template", "status"},
		),

		PolicyViolations: promauto.NewCounterVec(

			prometheus.CounterOpts{
				Name: "ca_policy_violations_total",

				Help: "Total number of policy violations",
			},

			[]string{"template", "violation_type"},
		),

		ApprovalRequests: promauto.NewCounterVec(

			prometheus.CounterOpts{
				Name: "ca_approval_requests_total",

				Help: "Total number of approval requests",
			},

			[]string{"workflow", "status"},
		),

		ActiveSessions: promauto.NewGaugeVec(

			prometheus.GaugeOpts{
				Name: "ca_active_sessions",

				Help: "Number of active CA sessions",
			},

			[]string{"backend", "session_type"},
		),

		ResourceUsage: promauto.NewGaugeVec(

			prometheus.GaugeOpts{
				Name: "ca_resource_usage_ratio",

				Help: "Resource usage ratio (0-1)",
			},

			[]string{"resource_type"},
		),

		ErrorRate: promauto.NewGaugeVec(

			prometheus.GaugeOpts{
				Name: "ca_error_rate",

				Help: "Error rate (0-1)",
			},

			[]string{"component", "error_type"},
		),
	}
}

// AlertManager methods.

// NewAlertManager creates a new alert manager.

func NewAlertManager(config *AlertConfig, logger *logging.StructuredLogger) (*AlertManager, error) {
	am := &AlertManager{
		config: config,

		logger: logger,

		rules: make(map[string]*AlertRule),

		activeAlerts: make(map[string]*Alert),
	}

	// Load alert rules.

	for _, rule := range config.Rules {
		am.rules[rule.Name] = rule
	}

	return am, nil
}

// Start starts the alert manager.

func (am *AlertManager) Start(ctx context.Context) {
	am.logger.Info("starting alert manager")

	ticker := time.NewTicker(30 * time.Second)

	defer ticker.Stop()

	for {
		select {

		case <-ctx.Done():

			return

		case <-ticker.C:

			am.processAlerts()

		}
	}
}

// TriggerAlert triggers an alert.

func (am *AlertManager) TriggerAlert(ruleName string, labels map[string]string) {
	am.mu.Lock()

	defer am.mu.Unlock()

	rule, exists := am.rules[ruleName]

	if !exists {

		am.logger.Warn("unknown alert rule", "rule", ruleName)

		return

	}

	alertID := am.generateAlertID(rule, labels)

	// Check if alert already exists and is in cooldown.

	if existing, exists := am.activeAlerts[alertID]; exists {
		if time.Since(existing.UpdatedAt) < am.config.Cooldown {
			return // Still in cooldown
		}
	}

	alert := &Alert{
		ID: alertID,

		Rule: rule,

		Status: AlertStatusFiring,

		StartsAt: time.Now(),

		UpdatedAt: time.Now(),

		Labels: labels,

		Annotations: rule.Annotations,
	}

	am.activeAlerts[alertID] = alert

	am.logger.Warn("alert triggered",

		"rule", ruleName,

		"alert_id", alertID,

		"severity", rule.Severity,

		"labels", labels)

	// Send notifications.

	go am.sendNotifications(alert)
}

// GetActiveAlerts returns all active alerts.

func (am *AlertManager) GetActiveAlerts() []*Alert {
	am.mu.RLock()

	defer am.mu.RUnlock()

	alerts := make([]*Alert, 0, len(am.activeAlerts))

	for _, alert := range am.activeAlerts {
		alerts = append(alerts, alert)
	}

	return alerts
}

// Helper methods for AlertManager.

func (am *AlertManager) processAlerts() {
	am.mu.Lock()

	defer am.mu.Unlock()

	// Clean up old alerts.

	for id, alert := range am.activeAlerts {
		if alert.Status == AlertStatusResolved && time.Since(alert.UpdatedAt) > 24*time.Hour {
			delete(am.activeAlerts, id)
		}
	}
}

func (am *AlertManager) sendNotifications(alert *Alert) {
	for _, channel := range am.config.NotificationChannels {
		switch channel.Type {

		case "webhook":

			am.sendWebhookNotification(channel, alert)

		case "email":

			am.sendEmailNotification(channel, alert)

		case "slack":

			am.sendSlackNotification(channel, alert)

		}
	}
}

func (am *AlertManager) sendWebhookNotification(channel NotificationChannel, alert *Alert) {
	am.logger.Debug("sending webhook notification",

		"channel", channel.Name,

		"alert", alert.ID)

	// Implementation would send HTTP POST to webhook URL.
}

func (am *AlertManager) sendEmailNotification(channel NotificationChannel, alert *Alert) {
	am.logger.Debug("sending email notification",

		"channel", channel.Name,

		"alert", alert.ID)

	// Implementation would send email via SMTP.
}

func (am *AlertManager) sendSlackNotification(channel NotificationChannel, alert *Alert) {
	am.logger.Debug("sending Slack notification",

		"channel", channel.Name,

		"alert", alert.ID)

	// Implementation would send message to Slack webhook.
}

func (am *AlertManager) generateAlertID(rule *AlertRule, labels map[string]string) string {
	return fmt.Sprintf("%s-%x", rule.Name, time.Now().Unix())
}
