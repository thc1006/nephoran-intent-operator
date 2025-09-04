// Package monitoring provides alerting and notification capabilities for O-RAN systems
package monitoring

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// AlertRouter handles alert routing and delivery
type AlertRouter struct {
	rules    map[string]*AlertRule
	channels map[string]AlertChannel
	alerts   map[string]*Alert
	mutex    sync.RWMutex
	logger   logr.Logger
}

// AlertChannel represents a notification channel
type AlertChannel interface {
	ID() string
	Name() string
	Send(ctx context.Context, alert *Alert) error
	Test(ctx context.Context) error
}

// AlertStats holds alert statistics
type AlertStats struct {
	TotalAlerts       int64            `json:"total_alerts"`
	FiringAlerts      int64            `json:"firing_alerts"`
	ResolvedAlerts    int64            `json:"resolved_alerts"`
	SuppressedAlerts  int64            `json:"suppressed_alerts"`
	LastAlert         *time.Time       `json:"last_alert,omitempty"`
	AlertsByComponent map[string]int64 `json:"alerts_by_component"`
	AlertsBySeverity  map[string]int64 `json:"alerts_by_severity"`
}

// WebhookChannel sends alerts via HTTP webhook
type WebhookChannel struct {
	id       string
	name     string
	url      string
	headers  map[string]string
	template string
}

// SlackChannel sends alerts to Slack
type SlackChannel struct {
	id       string
	name     string
	webhook  string
	channel  string
	username string
}

// EmailChannel sends alerts via email
type EmailChannel struct {
	id       string
	name     string
	smtp     SMTPConfig
	from     string
	to       []string
	template string
}

// SMTPConfig holds SMTP configuration
type SMTPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	TLS      bool   `json:"tls"`
}

// NewAlertRouter creates a new alert router
func NewAlertRouter() *AlertRouter {
	return &AlertRouter{
		rules:    make(map[string]*AlertRule),
		channels: make(map[string]AlertChannel),
		alerts:   make(map[string]*Alert),
		logger:   log.Log.WithName("alert-router"),
	}
}

// AddRule adds an alert rule
func (ar *AlertRouter) AddRule(rule *AlertRule) error {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()

	if rule.ID == "" {
		return fmt.Errorf("rule ID cannot be empty")
	}

	if rule.Duration == 0 {
		rule.Duration = 5 * time.Minute
	}

	if rule.Cooldown == 0 {
		rule.Cooldown = 15 * time.Minute
	}

	ar.rules[rule.ID] = rule

	ar.logger.Info("Added alert rule",
		"ruleID", rule.ID,
		"name", rule.Name,
		"severity", rule.Severity,
		"component", rule.Component)

	return nil
}

// RemoveRule removes an alert rule
func (ar *AlertRouter) RemoveRule(ruleID string) error {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()

	if _, exists := ar.rules[ruleID]; !exists {
		return fmt.Errorf("rule not found: %s", ruleID)
	}

	delete(ar.rules, ruleID)

	ar.logger.Info("Removed alert rule", "ruleID", ruleID)
	return nil
}

// AddChannel adds an alert channel
func (ar *AlertRouter) AddChannel(channel AlertChannel) error {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()

	ar.channels[channel.ID()] = channel

	ar.logger.Info("Added alert channel",
		"channelID", channel.ID(),
		"name", channel.Name())

	return nil
}

// RemoveChannel removes an alert channel
func (ar *AlertRouter) RemoveChannel(channelID string) error {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()

	if _, exists := ar.channels[channelID]; !exists {
		return fmt.Errorf("channel not found: %s", channelID)
	}

	delete(ar.channels, channelID)

	ar.logger.Info("Removed alert channel", "channelID", channelID)
	return nil
}

// FireAlert fires an alert based on a rule (fixes line 910)
func (ar *AlertRouter) FireAlert(ctx context.Context, ruleID string, labels map[string]string, value float64) error {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()

	rule, exists := ar.rules[ruleID]
	if !exists {
		return fmt.Errorf("rule not found: %s", ruleID)
	}

	if !rule.Enabled {
		ar.logger.V(1).Info("Rule is disabled, skipping alert", "ruleID", ruleID)
		return nil
	}

	// Check cooldown period
	if rule.LastFired != nil && time.Since(*rule.LastFired) < rule.Cooldown {
		ar.logger.V(1).Info("Rule is in cooldown period, skipping alert",
			"ruleID", ruleID,
			"cooldown", rule.Cooldown.String())
		return nil
	}

	// Create alert (Alert type is properly defined in types.go)
	alert := &Alert{
		ID:          fmt.Sprintf("%s-%d", ruleID, time.Now().Unix()),
		Rule:        ruleID,
		Component:   rule.Component,
		Severity:    rule.Severity,
		Status:      "firing",
		StartsAt:    time.Now(),
		Labels:      make(map[string]string),
		Annotations: make(map[string]string),
		Message:     ar.formatAlertMessage(rule, value),
		Description: ar.formatAlertDescription(rule, labels, value),
		Fingerprint: ar.generateFingerprint(rule, labels),
	}

	// Copy rule labels and annotations
	for k, v := range rule.Labels {
		alert.Labels[k] = v
	}
	for k, v := range rule.Annotations {
		alert.Annotations[k] = v
	}

	// Add provided labels
	for k, v := range labels {
		alert.Labels[k] = v
	}

	// Store alert
	ar.alerts[alert.ID] = alert

	// Update rule last fired time
	now := time.Now()
	rule.LastFired = &now

	ar.logger.Info("Fired alert",
		"alertID", alert.ID,
		"ruleID", ruleID,
		"severity", alert.Severity,
		"component", alert.Component,
		"message", alert.Message)

	// Send to configured channels
	go ar.sendAlert(ctx, alert, rule)

	return nil
}

// sendAlert sends an alert to configured channels (fixes line 910)
func (ar *AlertRouter) sendAlert(ctx context.Context, alert *Alert, rule *AlertRule) {
	for _, channelID := range rule.Channels {
		channel, exists := ar.channels[channelID]
		if !exists {
			ar.logger.Error(fmt.Errorf("channel not found"), "Failed to send alert",
				"channelID", channelID, "alertID", alert.ID)
			continue
		}

		if err := channel.Send(ctx, alert); err != nil {
			ar.logger.Error(err, "Failed to send alert to channel",
				"channelID", channelID,
				"alertID", alert.ID,
				"channel", channel.Name())
		} else {
			ar.logger.Info("Sent alert to channel",
				"channelID", channelID,
				"alertID", alert.ID,
				"channel", channel.Name())
		}
	}
}

// ResolveAlert resolves a firing alert
func (ar *AlertRouter) ResolveAlert(alertID string) error {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()

	alert, exists := ar.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert not found: %s", alertID)
	}

	if alert.Status != "firing" {
		return fmt.Errorf("alert is not firing: %s", alertID)
	}

	now := time.Now()
	alert.Status = "resolved"
	alert.EndsAt = &now

	ar.logger.Info("Resolved alert",
		"alertID", alertID,
		"duration", now.Sub(alert.StartsAt).String())

	return nil
}

// SuppressAlert suppresses an alert
func (ar *AlertRouter) SuppressAlert(alertID string, reason string) error {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()

	alert, exists := ar.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert not found: %s", alertID)
	}

	alert.Status = "suppressed"
	alert.Silenced = true
	alert.Annotations["suppression_reason"] = reason

	ar.logger.Info("Suppressed alert",
		"alertID", alertID,
		"reason", reason)

	return nil
}

// AcknowledgeAlert acknowledges an alert
func (ar *AlertRouter) AcknowledgeAlert(alertID string, acknowledgedBy string) error {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()

	alert, exists := ar.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert not found: %s", alertID)
	}

	now := time.Now()
	alert.AckBy = acknowledgedBy
	alert.AckAt = &now
	alert.Annotations["acknowledged_by"] = acknowledgedBy
	alert.Annotations["acknowledged_at"] = now.Format(time.RFC3339)

	ar.logger.Info("Acknowledged alert",
		"alertID", alertID,
		"acknowledgedBy", acknowledgedBy)

	return nil
}

// GetAlert returns an alert by ID (Alert type fix)
func (ar *AlertRouter) GetAlert(alertID string) (*Alert, error) {
	ar.mutex.RLock()
	defer ar.mutex.RUnlock()

	alert, exists := ar.alerts[alertID]
	if !exists {
		return nil, fmt.Errorf("alert not found: %s", alertID)
	}

	// Return a copy to avoid concurrent access issues
	alertCopy := *alert
	return &alertCopy, nil
}

// ListAlerts returns all alerts with optional filtering
func (ar *AlertRouter) ListAlerts(status string, component string, severity string) []*Alert {
	ar.mutex.RLock()
	defer ar.mutex.RUnlock()

	var filtered []*Alert

	for _, alert := range ar.alerts {
		if status != "" && string(alert.Status) != status {
			continue
		}
		if component != "" && alert.Component != component {
			continue
		}
		if severity != "" && string(alert.Severity) != severity {
			continue
		}

		alertCopy := *alert
		filtered = append(filtered, &alertCopy)
	}

	return filtered
}

// GetStats returns alert statistics
func (ar *AlertRouter) GetStats() *AlertStats {
	ar.mutex.RLock()
	defer ar.mutex.RUnlock()

	stats := &AlertStats{
		AlertsByComponent: make(map[string]int64),
		AlertsBySeverity:  make(map[string]int64),
	}

	for _, alert := range ar.alerts {
		stats.TotalAlerts++

		switch alert.Status {
		case "firing":
			stats.FiringAlerts++
		case "resolved":
			stats.ResolvedAlerts++
		case "suppressed":
			stats.SuppressedAlerts++
		}

		stats.AlertsByComponent[alert.Component]++
		stats.AlertsBySeverity[string(alert.Severity)]++

		if stats.LastAlert == nil || alert.StartsAt.After(*stats.LastAlert) {
			stats.LastAlert = &alert.StartsAt
		}
	}

	return stats
}

// formatAlertMessage formats the alert message
func (ar *AlertRouter) formatAlertMessage(rule *AlertRule, value float64) string {
	return fmt.Sprintf("%s alert: %s (value: %.2f, threshold: %s)",
		rule.Severity, rule.Name, value, rule.Threshold)
}

// formatAlertDescription formats the alert description
func (ar *AlertRouter) formatAlertDescription(rule *AlertRule, labels map[string]string, value float64) string {
	desc := fmt.Sprintf("Alert triggered for rule '%s' in component '%s'. Current value %.2f vs threshold %s.",
		rule.Name, rule.Component, value, rule.Threshold)

	if len(labels) > 0 {
		desc += " Labels: "
		for k, v := range labels {
			desc += fmt.Sprintf("%s=%s ", k, v)
		}
	}

	return desc
}

// generateFingerprint generates a unique fingerprint for the alert
func (ar *AlertRouter) generateFingerprint(rule *AlertRule, labels map[string]string) string {
	data := fmt.Sprintf("%s:%s", rule.ID, rule.Component)
	for k, v := range labels {
		data += fmt.Sprintf(":%s=%s", k, v)
	}
	return fmt.Sprintf("%x", []byte(data)) // Simple hash, could use proper hash function
}

// Shutdown gracefully shuts down the alert router
func (ar *AlertRouter) Shutdown(ctx context.Context) error {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()

	// Resolve all firing alerts
	for _, alert := range ar.alerts {
		if alert.Status == "firing" {
			now := time.Now()
			alert.Status = "resolved"
			alert.EndsAt = &now
		}
	}

	ar.logger.Info("Alert router shutdown completed")
	return nil
}

// Channel implementations

// WebhookChannel implementation
func NewWebhookChannel(id, name, url string, headers map[string]string, template string) *WebhookChannel {
	return &WebhookChannel{
		id:       id,
		name:     name,
		url:      url,
		headers:  headers,
		template: template,
	}
}

func (wc *WebhookChannel) ID() string {
	return wc.id
}

func (wc *WebhookChannel) Name() string {
	return wc.name
}

func (wc *WebhookChannel) Send(ctx context.Context, alert *Alert) error {
	// TODO: Implement webhook sending
	return fmt.Errorf("webhook channel not implemented yet")
}

func (wc *WebhookChannel) Test(ctx context.Context) error {
	// TODO: Implement webhook testing
	return fmt.Errorf("webhook channel testing not implemented yet")
}

// SlackChannel implementation
func NewSlackChannel(id, name, webhook, channel, username string) *SlackChannel {
	return &SlackChannel{
		id:       id,
		name:     name,
		webhook:  webhook,
		channel:  channel,
		username: username,
	}
}

func (sc *SlackChannel) ID() string {
	return sc.id
}

func (sc *SlackChannel) Name() string {
	return sc.name
}

func (sc *SlackChannel) Send(ctx context.Context, alert *Alert) error {
	// TODO: Implement Slack sending
	return fmt.Errorf("slack channel not implemented yet")
}

func (sc *SlackChannel) Test(ctx context.Context) error {
	// TODO: Implement Slack testing
	return fmt.Errorf("slack channel testing not implemented yet")
}

// EmailChannel implementation
func NewEmailChannel(id, name, from string, to []string, smtp SMTPConfig, template string) *EmailChannel {
	return &EmailChannel{
		id:       id,
		name:     name,
		smtp:     smtp,
		from:     from,
		to:       to,
		template: template,
	}
}

func (ec *EmailChannel) ID() string {
	return ec.id
}

func (ec *EmailChannel) Name() string {
	return ec.name
}

func (ec *EmailChannel) Send(ctx context.Context, alert *Alert) error {
	// TODO: Implement email sending
	return fmt.Errorf("email channel not implemented yet")
}

func (ec *EmailChannel) Test(ctx context.Context) error {
	// TODO: Implement email testing
	return fmt.Errorf("email channel testing not implemented yet")
}
