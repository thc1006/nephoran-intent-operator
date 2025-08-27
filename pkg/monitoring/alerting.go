package monitoring

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// AlertSeverity represents the severity of an alert
type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "info"
	SeverityWarning  AlertSeverity = "warning"
	SeverityError    AlertSeverity = "error"
	SeverityCritical AlertSeverity = "critical"
	SeverityLow      AlertSeverity = "low"
	SeverityMedium   AlertSeverity = "medium"
	SeverityHigh     AlertSeverity = "high"

	// Aliases for compatibility
	AlertSeverityInfo     = SeverityInfo
	AlertSeverityWarning  = SeverityWarning
	AlertSeverityError    = SeverityError
	AlertSeverityCritical = SeverityCritical
)

// AlertState represents the state of an alert
type AlertState string

const (
	AlertStateFiring   AlertState = "firing"
	AlertStateResolved AlertState = "resolved"
	AlertStateSilenced AlertState = "silenced"
)

// Alert represents an alert instance
type Alert struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Rule         string                 `json:"rule"`
	Component    string                 `json:"component"`
	Severity     AlertSeverity          `json:"severity"`
	State        AlertState             `json:"state"`
	Status       AlertState             `json:"status"`
	Message      string                 `json:"message"`
	Silenced     bool                   `json:"silenced"`
	AckBy        string                 `json:"ack_by,omitempty"`
	AckAt        *time.Time             `json:"ack_at,omitempty"`
	Labels       map[string]string      `json:"labels"`
	Annotations  map[string]string      `json:"annotations"`
	StartsAt     time.Time              `json:"starts_at"`
	EndsAt       *time.Time             `json:"ends_at,omitempty"`
	GeneratorURL string                 `json:"generator_url,omitempty"`
	Fingerprint  string                 `json:"fingerprint"`
	Value        interface{}            `json:"value,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
}

// AlertRule defines conditions for triggering alerts
type AlertRule struct {
	ID              string             `json:"id"`
	Name            string             `json:"name"`
	Description     string             `json:"description"`
	Component       string             `json:"component"`
	Condition       AlertConditionFunc `json:"-"`
	Severity        AlertSeverity      `json:"severity"`
	Duration        time.Duration      `json:"duration"`
	Cooldown        time.Duration      `json:"cooldown"`
	Threshold       float64            `json:"threshold"`
	Channels        []string           `json:"channels"`
	Labels          map[string]string  `json:"labels"`
	Annotations     map[string]string  `json:"annotations"`
	Enabled         bool               `json:"enabled"`
	LastEvaluated   time.Time          `json:"last_evaluated"`
	LastFired       *time.Time         `json:"last_fired,omitempty"`
	EvaluationCount int64              `json:"evaluation_count"`
	AlertCount      int64              `json:"alert_count"`
}

// AlertConditionFunc evaluates whether an alert should fire
type AlertConditionFunc func(ctx context.Context) (bool, interface{}, error)

// NotificationChannel represents a notification destination
type NotificationChannel struct {
	Name    string            `json:"name"`
	Type    string            `json:"type"` // slack, email, webhook, pagerduty
	Config  map[string]string `json:"config"`
	Enabled bool              `json:"enabled"`
	Filters []AlertFilter     `json:"filters"`
}

// AlertFilter filters alerts for notification channels
type AlertFilter struct {
	Field    string `json:"field"`    // severity, component, etc.
	Operator string `json:"operator"` // equals, contains, matches
	Value    string `json:"value"`
}

// AlertManager manages alerts and notifications
type AlertManager struct {
	mu                   sync.RWMutex
	rules                map[string]*AlertRule
	activeAlerts         map[string]*Alert
	notificationChannels map[string]*NotificationChannel
	evaluationInterval   time.Duration
	running              bool
	stopCh               chan struct{}
	logger               *StructuredLogger
	metricsRecorder      *MetricsRecorder
	httpClient           *http.Client
}

// AlertManagerConfig holds configuration for the alert manager
type AlertManagerConfig struct {
	EvaluationInterval   time.Duration         `json:"evaluation_interval"`
	NotificationChannels []NotificationChannel `json:"notification_channels"`
	DefaultLabels        map[string]string     `json:"default_labels"`
	ExternalURL          string                `json:"external_url"`
}

// NewAlertManager creates a new alert manager
func NewAlertManager(config *AlertManagerConfig, logger *StructuredLogger, metricsRecorder *MetricsRecorder) *AlertManager {
	if config == nil {
		config = &AlertManagerConfig{
			EvaluationInterval: 30 * time.Second,
		}
	}

	am := &AlertManager{
		rules:                make(map[string]*AlertRule),
		activeAlerts:         make(map[string]*Alert),
		notificationChannels: make(map[string]*NotificationChannel),
		evaluationInterval:   config.EvaluationInterval,
		stopCh:               make(chan struct{}),
		logger:               logger,
		metricsRecorder:      metricsRecorder,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Register notification channels
	for _, channel := range config.NotificationChannels {
		am.RegisterNotificationChannel(&channel)
	}

	return am
}

// RegisterAlertRule registers a new alert rule
func (am *AlertManager) RegisterAlertRule(rule *AlertRule) {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.rules[rule.Name] = rule

	if am.logger != nil {
		am.logger.Info(context.Background(), "Alert rule registered",
			slog.String("rule_name", rule.Name),
			slog.String("severity", string(rule.Severity)),
			slog.Bool("enabled", rule.Enabled),
		)
	}
}

// RegisterNotificationChannel registers a notification channel
func (am *AlertManager) RegisterNotificationChannel(channel *NotificationChannel) {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.notificationChannels[channel.Name] = channel

	if am.logger != nil {
		am.logger.Info(context.Background(), "Notification channel registered",
			slog.String("channel_name", channel.Name),
			slog.String("channel_type", channel.Type),
			slog.Bool("enabled", channel.Enabled),
		)
	}
}

// Start starts the alert manager
func (am *AlertManager) Start(ctx context.Context) error {
	am.mu.Lock()
	if am.running {
		am.mu.Unlock()
		return fmt.Errorf("alert manager is already running")
	}
	am.running = true
	am.mu.Unlock()

	// Register default alert rules
	am.registerDefaultAlertRules()

	// Start evaluation loop
	go am.evaluationLoop(ctx)

	if am.logger != nil {
		am.logger.Info(ctx, "Alert manager started",
			slog.Duration("evaluation_interval", am.evaluationInterval),
			slog.Int("rules_count", len(am.rules)),
			slog.Int("channels_count", len(am.notificationChannels)),
		)
	}

	return nil
}

// Stop stops the alert manager
func (am *AlertManager) Stop() {
	am.mu.Lock()
	defer am.mu.Unlock()

	if !am.running {
		return
	}

	close(am.stopCh)
	am.running = false

	if am.logger != nil {
		am.logger.Info(context.Background(), "Alert manager stopped")
	}
}

// evaluationLoop runs the alert evaluation loop
func (am *AlertManager) evaluationLoop(ctx context.Context) {
	ticker := time.NewTicker(am.evaluationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-am.stopCh:
			return
		case <-ticker.C:
			am.evaluateRules(ctx)
		}
	}
}

// evaluateRules evaluates all alert rules
func (am *AlertManager) evaluateRules(ctx context.Context) {
	am.mu.RLock()
	rules := make([]*AlertRule, 0, len(am.rules))
	for _, rule := range am.rules {
		if rule.Enabled {
			rules = append(rules, rule)
		}
	}
	am.mu.RUnlock()

	for _, rule := range rules {
		am.evaluateRule(ctx, rule)
	}
}

// evaluateRule evaluates a single alert rule
func (am *AlertManager) evaluateRule(ctx context.Context, rule *AlertRule) {
	defer func() {
		rule.LastEvaluated = time.Now()
		rule.EvaluationCount++
	}()

	// Evaluate condition
	shouldFire, value, err := rule.Condition(ctx)
	if err != nil {
		if am.logger != nil {
			am.logger.Error(ctx, "Failed to evaluate alert rule", err,
				slog.String("rule_name", rule.Name),
			)
		}
		return
	}

	alertID := am.generateAlertID(rule)

	am.mu.Lock()
	existingAlert, exists := am.activeAlerts[alertID]
	am.mu.Unlock()

	if shouldFire {
		if !exists {
			// Create new alert
			alert := &Alert{
				ID:          alertID,
				Name:        rule.Name,
				Description: rule.Description,
				Severity:    rule.Severity,
				State:       AlertStateFiring,
				Labels:      am.copyLabels(rule.Labels),
				Annotations: am.copyLabels(rule.Annotations),
				StartsAt:    time.Now(),
				Fingerprint: am.generateFingerprint(rule),
				Value:       value,
			}

			am.mu.Lock()
			am.activeAlerts[alertID] = alert
			am.mu.Unlock()

			rule.AlertCount++

			// Send notifications
			am.sendNotifications(ctx, alert)

			if am.logger != nil {
				am.logger.Warn(ctx, "Alert fired",
					slog.String("alert_id", alert.ID),
					slog.String("alert_name", alert.Name),
					slog.String("severity", string(alert.Severity)),
					slog.Any("value", value),
				)
			}

			// Record metrics
			if am.metricsRecorder != nil {
				am.metricsRecorder.RecordControllerHealth("alerting", rule.Name, false)
			}
		}
	} else if exists && existingAlert.State == AlertStateFiring {
		// Resolve existing alert
		now := time.Now()
		existingAlert.State = AlertStateResolved
		existingAlert.EndsAt = &now

		// Send resolution notifications
		am.sendNotifications(ctx, existingAlert)

		if am.logger != nil {
			am.logger.Info(ctx, "Alert resolved",
				slog.String("alert_id", existingAlert.ID),
				slog.String("alert_name", existingAlert.Name),
				slog.Duration("duration", now.Sub(existingAlert.StartsAt)),
			)
		}

		// Remove from active alerts after a delay
		go func() {
			time.Sleep(5 * time.Minute)
			am.mu.Lock()
			delete(am.activeAlerts, alertID)
			am.mu.Unlock()
		}()

		// Record metrics
		if am.metricsRecorder != nil {
			am.metricsRecorder.RecordControllerHealth("alerting", rule.Name, true)
		}
	}
}

// sendNotifications sends alert notifications to configured channels
func (am *AlertManager) sendNotifications(ctx context.Context, alert *Alert) {
	am.mu.RLock()
	channels := make([]*NotificationChannel, 0, len(am.notificationChannels))
	for _, channel := range am.notificationChannels {
		if channel.Enabled && am.shouldNotify(alert, channel) {
			channels = append(channels, channel)
		}
	}
	am.mu.RUnlock()

	for _, channel := range channels {
		go am.sendNotification(ctx, alert, channel)
	}
}

// shouldNotify determines if an alert should be sent to a channel based on filters
func (am *AlertManager) shouldNotify(alert *Alert, channel *NotificationChannel) bool {
	for _, filter := range channel.Filters {
		if !am.matchFilter(alert, filter) {
			return false
		}
	}
	return true
}

// matchFilter checks if an alert matches a filter
func (am *AlertManager) matchFilter(alert *Alert, filter AlertFilter) bool {
	var value string

	switch filter.Field {
	case "severity":
		value = string(alert.Severity)
	case "name":
		value = alert.Name
	case "state":
		value = string(alert.State)
	default:
		if labelValue, exists := alert.Labels[filter.Field]; exists {
			value = labelValue
		} else {
			return false
		}
	}

	switch filter.Operator {
	case "equals":
		return value == filter.Value
	case "contains":
		return strings.Contains(value, filter.Value)
	case "matches":
		// Simple wildcard matching
		return strings.Contains(value, strings.Replace(filter.Value, "*", "", -1))
	default:
		return false
	}
}

// sendNotification sends a notification to a specific channel
func (am *AlertManager) sendNotification(ctx context.Context, alert *Alert, channel *NotificationChannel) {
	switch channel.Type {
	case "slack":
		am.sendSlackNotification(ctx, alert, channel)
	case "webhook":
		am.sendWebhookNotification(ctx, alert, channel)
	case "email":
		am.sendEmailNotification(ctx, alert, channel)
	case "pagerduty":
		am.sendPagerDutyNotification(ctx, alert, channel)
	default:
		if am.logger != nil {
			am.logger.Warn(ctx, "Unknown notification channel type",
				slog.String("channel_type", channel.Type),
				slog.String("channel_name", channel.Name),
			)
		}
	}
}

// sendSlackNotification sends a Slack notification
func (am *AlertManager) sendSlackNotification(ctx context.Context, alert *Alert, channel *NotificationChannel) {
	webhookURL, exists := channel.Config["webhook_url"]
	if !exists {
		am.logger.Error(ctx, "Slack webhook URL not configured", nil,
			slog.String("channel_name", channel.Name),
		)
		return
	}

	color := am.getAlertColor(alert.Severity)
	stateEmoji := am.getStateEmoji(alert.State)

	payload := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"color":     color,
				"title":     fmt.Sprintf("%s %s", stateEmoji, alert.Name),
				"text":      alert.Description,
				"timestamp": alert.StartsAt.Unix(),
				"fields": []map[string]interface{}{
					{
						"title": "Severity",
						"value": string(alert.Severity),
						"short": true,
					},
					{
						"title": "State",
						"value": string(alert.State),
						"short": true,
					},
				},
			},
		},
	}

	if alert.Value != nil {
		attachment := payload["attachments"].([]map[string]interface{})[0]
		fields := attachment["fields"].([]map[string]interface{})
		fields = append(fields, map[string]interface{}{
			"title": "Value",
			"value": fmt.Sprintf("%v", alert.Value),
			"short": true,
		})
		attachment["fields"] = fields
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		am.logger.Error(ctx, "Failed to marshal Slack payload", err,
			slog.String("channel_name", channel.Name),
		)
		return
	}

	resp, err := am.httpClient.Post(webhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		am.logger.Error(ctx, "Failed to send Slack notification", err,
			slog.String("channel_name", channel.Name),
		)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		am.logger.Error(ctx, "Slack notification failed", nil,
			slog.String("channel_name", channel.Name),
			slog.Int("status_code", resp.StatusCode),
		)
		return
	}

	if am.logger != nil {
		am.logger.Debug(ctx, "Slack notification sent successfully",
			slog.String("channel_name", channel.Name),
			slog.String("alert_id", alert.ID),
		)
	}
}

// sendWebhookNotification sends a generic webhook notification
func (am *AlertManager) sendWebhookNotification(ctx context.Context, alert *Alert, channel *NotificationChannel) {
	webhookURL, exists := channel.Config["url"]
	if !exists {
		am.logger.Error(ctx, "Webhook URL not configured", nil,
			slog.String("channel_name", channel.Name),
		)
		return
	}

	jsonPayload, err := json.Marshal(alert)
	if err != nil {
		am.logger.Error(ctx, "Failed to marshal webhook payload", err,
			slog.String("channel_name", channel.Name),
		)
		return
	}

	resp, err := am.httpClient.Post(webhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		am.logger.Error(ctx, "Failed to send webhook notification", err,
			slog.String("channel_name", channel.Name),
		)
		return
	}
	defer resp.Body.Close()

	if am.logger != nil {
		am.logger.Debug(ctx, "Webhook notification sent",
			slog.String("channel_name", channel.Name),
			slog.String("alert_id", alert.ID),
			slog.Int("status_code", resp.StatusCode),
		)
	}
}

// sendEmailNotification sends an email notification (placeholder implementation)
func (am *AlertManager) sendEmailNotification(ctx context.Context, alert *Alert, channel *NotificationChannel) {
	// Implementation would integrate with SMTP server
	if am.logger != nil {
		am.logger.Info(ctx, "Email notification would be sent",
			slog.String("channel_name", channel.Name),
			slog.String("alert_id", alert.ID),
		)
	}
}

// sendPagerDutyNotification sends a PagerDuty notification (placeholder implementation)
func (am *AlertManager) sendPagerDutyNotification(ctx context.Context, alert *Alert, channel *NotificationChannel) {
	// Implementation would integrate with PagerDuty API
	if am.logger != nil {
		am.logger.Info(ctx, "PagerDuty notification would be sent",
			slog.String("channel_name", channel.Name),
			slog.String("alert_id", alert.ID),
		)
	}
}

// GetActiveAlerts returns all active alerts
func (am *AlertManager) GetActiveAlerts() []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	alerts := make([]*Alert, 0, len(am.activeAlerts))
	for _, alert := range am.activeAlerts {
		if alert.State == AlertStateFiring {
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

// GetAlertRules returns all registered alert rules
func (am *AlertManager) GetAlertRules() []*AlertRule {
	am.mu.RLock()
	defer am.mu.RUnlock()

	rules := make([]*AlertRule, 0, len(am.rules))
	for _, rule := range am.rules {
		rules = append(rules, rule)
	}

	return rules
}

// registerDefaultAlertRules registers default alert rules for the Nephoran system
func (am *AlertManager) registerDefaultAlertRules() {
	// NetworkIntent processing failure rate
	am.RegisterAlertRule(&AlertRule{
		Name:        "NetworkIntentHighFailureRate",
		Description: "NetworkIntent processing failure rate is high",
		Severity:    AlertSeverityWarning,
		Duration:    2 * time.Minute,
		Labels: map[string]string{
			"component": "networkintent-controller",
			"type":      "performance",
		},
		Annotations: map[string]string{
			"summary":     "High NetworkIntent failure rate detected",
			"description": "NetworkIntent processing failure rate is above 10% for the last 5 minutes",
			"runbook":     "https://docs.nephoran.com/runbooks/networkintent-failures",
		},
		Enabled: true,
		Condition: func(ctx context.Context) (bool, interface{}, error) {
			// This would typically query Prometheus metrics
			// For now, return false as a placeholder
			return false, nil, nil
		},
	})

	// LLM service down
	am.RegisterAlertRule(&AlertRule{
		Name:        "LLMServiceDown",
		Description: "LLM service is not responding",
		Severity:    AlertSeverityCritical,
		Duration:    1 * time.Minute,
		Labels: map[string]string{
			"component": "llm-service",
			"type":      "availability",
		},
		Annotations: map[string]string{
			"summary":     "LLM service is down",
			"description": "The LLM processing service is not responding to health checks",
			"runbook":     "https://docs.nephoran.com/runbooks/llm-service-down",
		},
		Enabled: true,
		Condition: func(ctx context.Context) (bool, interface{}, error) {
			// This would check LLM service health
			return false, nil, nil
		},
	})

	// High memory usage
	am.RegisterAlertRule(&AlertRule{
		Name:        "HighMemoryUsage",
		Description: "System memory usage is critically high",
		Severity:    AlertSeverityWarning,
		Duration:    5 * time.Minute,
		Labels: map[string]string{
			"component": "system",
			"type":      "resource",
		},
		Annotations: map[string]string{
			"summary":     "High memory usage detected",
			"description": "System memory usage is above 85% for the last 5 minutes",
			"runbook":     "https://docs.nephoran.com/runbooks/high-memory-usage",
		},
		Enabled: true,
		Condition: func(ctx context.Context) (bool, interface{}, error) {
			// This would check memory metrics
			return false, nil, nil
		},
	})

	// O-RAN interface connection issues
	am.RegisterAlertRule(&AlertRule{
		Name:        "ORANInterfaceDown",
		Description: "O-RAN interface connection is down",
		Severity:    AlertSeverityCritical,
		Duration:    2 * time.Minute,
		Labels: map[string]string{
			"component": "oran-interface",
			"type":      "connectivity",
		},
		Annotations: map[string]string{
			"summary":     "O-RAN interface connection down",
			"description": "One or more O-RAN interface connections are down",
			"runbook":     "https://docs.nephoran.com/runbooks/oran-interface-down",
		},
		Enabled: true,
		Condition: func(ctx context.Context) (bool, interface{}, error) {
			// This would check O-RAN interface connectivity
			return false, nil, nil
		},
	})
}

// Utility functions

// generateAlertID generates a unique alert ID
func (am *AlertManager) generateAlertID(rule *AlertRule) string {
	return fmt.Sprintf("%s-%d", rule.Name, time.Now().Unix())
}

// generateFingerprint generates a fingerprint for an alert
func (am *AlertManager) generateFingerprint(rule *AlertRule) string {
	return fmt.Sprintf("%s-%s", rule.Name, rule.Description)
}

// copyLabels creates a copy of labels map
func (am *AlertManager) copyLabels(labels map[string]string) map[string]string {
	if labels == nil {
		return make(map[string]string)
	}

	copy := make(map[string]string, len(labels))
	for k, v := range labels {
		copy[k] = v
	}
	return copy
}

// getAlertColor returns color for Slack notifications based on severity
func (am *AlertManager) getAlertColor(severity AlertSeverity) string {
	switch severity {
	case AlertSeverityInfo:
		return "good"
	case AlertSeverityWarning:
		return "warning"
	case AlertSeverityError:
		return "danger"
	case AlertSeverityCritical:
		return "danger"
	default:
		return "warning"
	}
}

// getStateEmoji returns emoji for alert state
func (am *AlertManager) getStateEmoji(state AlertState) string {
	switch state {
	case AlertStateFiring:
		return "🔥"
	case AlertStateResolved:
		return "✅"
	case AlertStateSilenced:
		return "🔇"
	default:
		return "❓"
	}
}
