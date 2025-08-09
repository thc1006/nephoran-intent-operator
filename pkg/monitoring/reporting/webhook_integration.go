// Package reporting provides webhook integration capabilities for automated
// performance reporting and real-time notifications
package reporting

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

// WebhookManager manages webhook integrations for performance reporting
type WebhookManager struct {
	logger   *zap.Logger
	client   *http.Client
	webhooks map[string]WebhookConfig

	// Metrics
	webhookRequests prometheus.Counter
	webhookFailures prometheus.Counter
	webhookLatency  prometheus.Histogram
	webhookRetries  prometheus.Counter
}

// WebhookPayload represents the standard webhook payload structure
type WebhookPayload struct {
	// Standard fields
	EventType   string    `json:"event_type"`
	Timestamp   time.Time `json:"timestamp"`
	Source      string    `json:"source"`
	Environment string    `json:"environment"`
	Severity    string    `json:"severity"`

	// Performance report data
	Report  *PerformanceReport `json:"report,omitempty"`
	Summary *ExecutiveSummary  `json:"summary,omitempty"`
	Claims  []ClaimValidation  `json:"claims,omitempty"`
	Alerts  []AlertInfo        `json:"alerts,omitempty"`

	// Custom data
	CustomData map[string]interface{} `json:"custom_data,omitempty"`
}

// SlackPayload represents Slack-specific webhook format
type SlackPayload struct {
	Channel     string            `json:"channel,omitempty"`
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Text        string            `json:"text"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

// SlackAttachment represents Slack message attachment
type SlackAttachment struct {
	Color      string        `json:"color,omitempty"`
	Title      string        `json:"title,omitempty"`
	Text       string        `json:"text,omitempty"`
	Fields     []SlackField  `json:"fields,omitempty"`
	Actions    []SlackAction `json:"actions,omitempty"`
	Timestamp  int64         `json:"ts,omitempty"`
	Footer     string        `json:"footer,omitempty"`
	FooterIcon string        `json:"footer_icon,omitempty"`
}

// SlackField represents Slack attachment field
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// SlackAction represents Slack interactive action
type SlackAction struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	URL   string `json:"url,omitempty"`
	Style string `json:"style,omitempty"`
}

// TeamsPayload represents Microsoft Teams webhook format
type TeamsPayload struct {
	Type       string         `json:"@type"`
	Context    string         `json:"@context"`
	ThemeColor string         `json:"themeColor,omitempty"`
	Summary    string         `json:"summary"`
	Sections   []TeamsSection `json:"sections,omitempty"`
	Actions    []TeamsAction  `json:"potentialAction,omitempty"`
}

// TeamsSection represents Teams message section
type TeamsSection struct {
	ActivityTitle    string      `json:"activityTitle,omitempty"`
	ActivitySubtitle string      `json:"activitySubtitle,omitempty"`
	ActivityText     string      `json:"activityText,omitempty"`
	ActivityImage    string      `json:"activityImage,omitempty"`
	Facts            []TeamsFact `json:"facts,omitempty"`
	Markdown         bool        `json:"markdown,omitempty"`
}

// TeamsFact represents Teams fact
type TeamsFact struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// TeamsAction represents Teams action
type TeamsAction struct {
	Type    string              `json:"@type"`
	Name    string              `json:"name"`
	Targets []map[string]string `json:"targets,omitempty"`
}

// PagerDutyPayload represents PagerDuty webhook format
type PagerDutyPayload struct {
	RoutingKey  string                `json:"routing_key"`
	EventAction string                `json:"event_action"`
	DedupKey    string                `json:"dedup_key,omitempty"`
	Payload     PagerDutyEventPayload `json:"payload"`
	Links       []PagerDutyLink       `json:"links,omitempty"`
	Images      []PagerDutyImage      `json:"images,omitempty"`
}

// PagerDutyEventPayload represents PagerDuty event payload
type PagerDutyEventPayload struct {
	Summary   string                 `json:"summary"`
	Source    string                 `json:"source"`
	Severity  string                 `json:"severity"`
	Timestamp time.Time              `json:"timestamp"`
	Component string                 `json:"component,omitempty"`
	Group     string                 `json:"group,omitempty"`
	Class     string                 `json:"class,omitempty"`
	Details   map[string]interface{} `json:"custom_details,omitempty"`
}

// PagerDutyLink represents PagerDuty link
type PagerDutyLink struct {
	Href string `json:"href"`
	Text string `json:"text"`
}

// PagerDutyImage represents PagerDuty image
type PagerDutyImage struct {
	Src  string `json:"src"`
	Href string `json:"href,omitempty"`
	Alt  string `json:"alt,omitempty"`
}

// WebhookEvent represents different types of webhook events
type WebhookEvent struct {
	Type       string
	Report     *PerformanceReport
	Alert      *AlertInfo
	Regression *RegressionAnalysis
	CustomData map[string]interface{}
}

// NewWebhookManager creates a new webhook manager
func NewWebhookManager(logger *zap.Logger, webhooks map[string]WebhookConfig) *WebhookManager {
	// Initialize metrics
	webhookRequests := promauto.NewCounter(prometheus.CounterOpts{
		Name: "nephoran_webhook_requests_total",
		Help: "Total number of webhook requests sent",
	})

	webhookFailures := promauto.NewCounter(prometheus.CounterOpts{
		Name: "nephoran_webhook_failures_total",
		Help: "Total number of webhook failures",
	})

	webhookLatency := promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "nephoran_webhook_duration_seconds",
		Help:    "Webhook request duration in seconds",
		Buckets: prometheus.DefBuckets,
	})

	webhookRetries := promauto.NewCounter(prometheus.CounterOpts{
		Name: "nephoran_webhook_retries_total",
		Help: "Total number of webhook retries",
	})

	return &WebhookManager{
		logger:          logger,
		client:          &http.Client{Timeout: 30 * time.Second},
		webhooks:        webhooks,
		webhookRequests: webhookRequests,
		webhookFailures: webhookFailures,
		webhookLatency:  webhookLatency,
		webhookRetries:  webhookRetries,
	}
}

// SendPerformanceReport sends performance report to all configured webhooks
func (wm *WebhookManager) SendPerformanceReport(ctx context.Context, report *PerformanceReport) error {
	event := WebhookEvent{
		Type:   "performance_report",
		Report: report,
	}

	var allErrors []error
	for name, webhook := range wm.webhooks {
		if !webhook.Enabled {
			continue
		}

		if err := wm.sendWebhook(ctx, name, webhook, event); err != nil {
			wm.logger.Error("Failed to send webhook",
				zap.String("webhook", name),
				zap.Error(err))
			allErrors = append(allErrors, err)
		}
	}

	if len(allErrors) > 0 {
		return fmt.Errorf("failed to send %d webhooks: %v", len(allErrors), allErrors)
	}

	return nil
}

// SendAlert sends alert notification to configured webhooks
func (wm *WebhookManager) SendAlert(ctx context.Context, alert *AlertInfo) error {
	event := WebhookEvent{
		Type:  "alert",
		Alert: alert,
	}

	var allErrors []error
	for name, webhook := range wm.webhooks {
		if !webhook.Enabled {
			continue
		}

		// Check if webhook should receive alerts
		if !wm.shouldSendAlert(webhook, alert) {
			continue
		}

		if err := wm.sendWebhook(ctx, name, webhook, event); err != nil {
			wm.logger.Error("Failed to send alert webhook",
				zap.String("webhook", name),
				zap.String("alert", alert.Name),
				zap.Error(err))
			allErrors = append(allErrors, err)
		}
	}

	if len(allErrors) > 0 {
		return fmt.Errorf("failed to send alert webhooks: %v", allErrors)
	}

	return nil
}

// SendRegressionAlert sends regression detection notification
func (wm *WebhookManager) SendRegressionAlert(ctx context.Context, regression *RegressionAnalysis) error {
	if !regression.RegressionDetected {
		return nil // No regression to report
	}

	event := WebhookEvent{
		Type:       "regression_detected",
		Regression: regression,
	}

	var allErrors []error
	for name, webhook := range wm.webhooks {
		if !webhook.Enabled {
			continue
		}

		if err := wm.sendWebhook(ctx, name, webhook, event); err != nil {
			wm.logger.Error("Failed to send regression webhook",
				zap.String("webhook", name),
				zap.Error(err))
			allErrors = append(allErrors, err)
		}
	}

	if len(allErrors) > 0 {
		return fmt.Errorf("failed to send regression webhooks: %v", allErrors)
	}

	return nil
}

// sendWebhook sends a webhook with retry logic
func (wm *WebhookManager) sendWebhook(ctx context.Context, name string, webhook WebhookConfig, event WebhookEvent) error {
	timer := prometheus.NewTimer(wm.webhookLatency)
	defer timer.ObserveDuration()

	var lastErr error
	maxRetries := webhook.Retries
	if maxRetries == 0 {
		maxRetries = 3 // Default retry count
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			wm.webhookRetries.Inc()
			backoff := time.Duration(attempt*attempt) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				// Continue with retry
			}
		}

		wm.webhookRequests.Inc()

		// Create payload based on webhook format
		payload, contentType, err := wm.createPayload(webhook, event)
		if err != nil {
			lastErr = fmt.Errorf("failed to create payload: %w", err)
			continue
		}

		// Create HTTP request
		req, err := http.NewRequestWithContext(ctx, webhook.Method, webhook.URL, bytes.NewReader(payload))
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		// Set headers
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("User-Agent", "Nephoran-Performance-Monitor/1.0")

		// Add custom headers
		for key, value := range webhook.Headers {
			req.Header.Set(key, value)
		}

		// Add authentication if configured
		if err := wm.addAuthentication(req, webhook, payload); err != nil {
			lastErr = fmt.Errorf("failed to add authentication: %w", err)
			continue
		}

		// Send request with timeout
		client := wm.client
		if webhook.Timeout > 0 {
			client = &http.Client{Timeout: webhook.Timeout}
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to send request: %w", err)
			continue
		}

		// Read response
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Check response status
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			wm.logger.Info("Webhook sent successfully",
				zap.String("webhook", name),
				zap.String("url", webhook.URL),
				zap.Int("status", resp.StatusCode),
				zap.Int("attempt", attempt+1))
			return nil
		}

		lastErr = fmt.Errorf("webhook failed with status %d: %s", resp.StatusCode, string(body))
		wm.logger.Warn("Webhook failed",
			zap.String("webhook", name),
			zap.Int("status", resp.StatusCode),
			zap.String("response", string(body)),
			zap.Int("attempt", attempt+1))
	}

	wm.webhookFailures.Inc()
	return fmt.Errorf("webhook failed after %d attempts: %w", maxRetries+1, lastErr)
}

// createPayload creates the appropriate payload format for the webhook
func (wm *WebhookManager) createPayload(webhook WebhookConfig, event WebhookEvent) ([]byte, string, error) {
	switch strings.ToLower(webhook.Format) {
	case "slack":
		return wm.createSlackPayload(webhook, event)
	case "teams":
		return wm.createTeamsPayload(webhook, event)
	case "pagerduty":
		return wm.createPagerDutyPayload(webhook, event)
	case "json":
		return wm.createJSONPayload(webhook, event)
	case "custom":
		return wm.createCustomPayload(webhook, event)
	default:
		return wm.createJSONPayload(webhook, event)
	}
}

// createSlackPayload creates Slack-formatted webhook payload
func (wm *WebhookManager) createSlackPayload(webhook WebhookConfig, event WebhookEvent) ([]byte, string, error) {
	var payload SlackPayload
	var color string

	switch event.Type {
	case "performance_report":
		if event.Report == nil {
			return nil, "", fmt.Errorf("missing performance report")
		}

		// Determine color based on performance score
		score := event.Report.ExecutiveSummary.OverallScore
		if score >= 90 {
			color = "good"
		} else if score >= 70 {
			color = "warning"
		} else {
			color = "danger"
		}

		payload = SlackPayload{
			Channel:   "#performance-alerts",
			Username:  "Nephoran Performance Monitor",
			IconEmoji: ":chart_with_upwards_trend:",
			Text:      "📊 Performance Report Generated",
			Attachments: []SlackAttachment{
				{
					Color: color,
					Title: fmt.Sprintf("Performance Report - %s", event.Report.Period),
					Fields: []SlackField{
						{
							Title: "Overall Score",
							Value: fmt.Sprintf("%.1f%% (%s)", score, event.Report.ExecutiveSummary.PerformanceGrade),
							Short: true,
						},
						{
							Title: "System Health",
							Value: event.Report.ExecutiveSummary.SystemHealth,
							Short: true,
						},
						{
							Title: "SLA Compliance",
							Value: fmt.Sprintf("%.2f%%", event.Report.ExecutiveSummary.SLACompliance),
							Short: true,
						},
						{
							Title: "Trend",
							Value: event.Report.ExecutiveSummary.TrendDirection,
							Short: true,
						},
					},
					Actions: []SlackAction{
						{
							Type:  "button",
							Text:  "View Dashboard",
							URL:   "https://grafana.nephoran.com/d/nephoran-executive-perf",
							Style: "primary",
						},
					},
					Footer:    "Nephoran Performance Monitoring",
					Timestamp: event.Report.Timestamp.Unix(),
				},
			},
		}

		// Add performance claims status
		if len(event.Report.PerformanceClaims) > 0 {
			claimsText := "*Performance Claims Status:*\n"
			for _, claim := range event.Report.PerformanceClaims {
				status := "✅"
				if claim.Status == "FAIL" {
					status = "❌"
				}
				claimsText += fmt.Sprintf("%s %s: %s (Target: %s)\n",
					status, claim.Name, claim.Actual, claim.Target)
			}

			payload.Attachments = append(payload.Attachments, SlackAttachment{
				Color: color,
				Title: "Performance Claims Validation",
				Text:  claimsText,
			})
		}

	case "alert":
		if event.Alert == nil {
			return nil, "", fmt.Errorf("missing alert")
		}

		// Determine color based on severity
		switch strings.ToLower(event.Alert.Severity) {
		case "critical":
			color = "danger"
		case "warning":
			color = "warning"
		default:
			color = "good"
		}

		payload = SlackPayload{
			Channel:   "#performance-alerts",
			Username:  "Nephoran Alert Manager",
			IconEmoji: ":warning:",
			Text:      fmt.Sprintf("🚨 %s Alert: %s", strings.ToUpper(event.Alert.Severity), event.Alert.Name),
			Attachments: []SlackAttachment{
				{
					Color: color,
					Title: event.Alert.Name,
					Text:  event.Alert.Description,
					Fields: []SlackField{
						{
							Title: "Severity",
							Value: event.Alert.Severity,
							Short: true,
						},
						{
							Title: "Component",
							Value: event.Alert.Component,
							Short: true,
						},
						{
							Title: "Count",
							Value: fmt.Sprintf("%d", event.Alert.Count),
							Short: true,
						},
						{
							Title: "Duration",
							Value: event.Alert.Duration.String(),
							Short: true,
						},
					},
					Footer:    "Nephoran Alert Manager",
					Timestamp: event.Alert.LastFired.Unix(),
				},
			},
		}

	case "regression_detected":
		if event.Regression == nil {
			return nil, "", fmt.Errorf("missing regression analysis")
		}

		color = "danger" // Regressions are always critical

		payload = SlackPayload{
			Channel:   "#performance-alerts",
			Username:  "Nephoran Regression Detector",
			IconEmoji: ":rotating_light:",
			Text:      "🔴 Performance Regression Detected",
			Attachments: []SlackAttachment{
				{
					Color: color,
					Title: "Performance Regression Alert",
					Text:  fmt.Sprintf("Regression severity: %s", event.Regression.RegressionSeverity),
					Fields: []SlackField{
						{
							Title: "Regression Detected",
							Value: fmt.Sprintf("%t", event.Regression.RegressionDetected),
							Short: true,
						},
						{
							Title: "Severity",
							Value: event.Regression.RegressionSeverity,
							Short: true,
						},
					},
					Footer: "Nephoran Regression Detector",
				},
			},
		}

		// Add performance changes
		if len(event.Regression.PerformanceChange) > 0 {
			changesText := "*Performance Changes:*\n"
			for metric, change := range event.Regression.PerformanceChange {
				changesText += fmt.Sprintf("• %s: %+.2f%%\n", metric, change)
			}
			payload.Attachments[0].Text += "\n" + changesText
		}
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal Slack payload: %w", err)
	}

	return data, "application/json", nil
}

// createTeamsPayload creates Microsoft Teams webhook payload
func (wm *WebhookManager) createTeamsPayload(webhook WebhookConfig, event WebhookEvent) ([]byte, string, error) {
	var payload TeamsPayload
	var themeColor string

	switch event.Type {
	case "performance_report":
		if event.Report == nil {
			return nil, "", fmt.Errorf("missing performance report")
		}

		score := event.Report.ExecutiveSummary.OverallScore
		if score >= 90 {
			themeColor = "0078D4" // Blue
		} else if score >= 70 {
			themeColor = "FF8C00" // Orange
		} else {
			themeColor = "D13438" // Red
		}

		payload = TeamsPayload{
			Type:       "MessageCard",
			Context:    "https://schema.org/extensions",
			ThemeColor: themeColor,
			Summary:    "Nephoran Performance Report",
			Sections: []TeamsSection{
				{
					ActivityTitle:    "📊 Performance Report Generated",
					ActivitySubtitle: fmt.Sprintf("Report Period: %s", event.Report.Period),
					Facts: []TeamsFact{
						{
							Name:  "Overall Score",
							Value: fmt.Sprintf("%.1f%% (%s)", score, event.Report.ExecutiveSummary.PerformanceGrade),
						},
						{
							Name:  "System Health",
							Value: event.Report.ExecutiveSummary.SystemHealth,
						},
						{
							Name:  "SLA Compliance",
							Value: fmt.Sprintf("%.2f%%", event.Report.ExecutiveSummary.SLACompliance),
						},
						{
							Name:  "Trend Direction",
							Value: event.Report.ExecutiveSummary.TrendDirection,
						},
					},
					Markdown: true,
				},
			},
			Actions: []TeamsAction{
				{
					Type: "OpenUri",
					Name: "View Dashboard",
					Targets: []map[string]string{
						{
							"os":  "default",
							"uri": "https://grafana.nephoran.com/d/nephoran-executive-perf",
						},
					},
				},
			},
		}

	case "alert":
		if event.Alert == nil {
			return nil, "", fmt.Errorf("missing alert")
		}

		switch strings.ToLower(event.Alert.Severity) {
		case "critical":
			themeColor = "D13438" // Red
		case "warning":
			themeColor = "FF8C00" // Orange
		default:
			themeColor = "0078D4" // Blue
		}

		payload = TeamsPayload{
			Type:       "MessageCard",
			Context:    "https://schema.org/extensions",
			ThemeColor: themeColor,
			Summary:    fmt.Sprintf("Nephoran Alert: %s", event.Alert.Name),
			Sections: []TeamsSection{
				{
					ActivityTitle:    fmt.Sprintf("🚨 %s Alert", strings.ToUpper(event.Alert.Severity)),
					ActivitySubtitle: event.Alert.Name,
					ActivityText:     event.Alert.Description,
					Facts: []TeamsFact{
						{
							Name:  "Severity",
							Value: event.Alert.Severity,
						},
						{
							Name:  "Component",
							Value: event.Alert.Component,
						},
						{
							Name:  "Count",
							Value: fmt.Sprintf("%d", event.Alert.Count),
						},
						{
							Name:  "Duration",
							Value: event.Alert.Duration.String(),
						},
					},
					Markdown: true,
				},
			},
		}

	case "regression_detected":
		if event.Regression == nil {
			return nil, "", fmt.Errorf("missing regression analysis")
		}

		payload = TeamsPayload{
			Type:       "MessageCard",
			Context:    "https://schema.org/extensions",
			ThemeColor: "D13438", // Red for regressions
			Summary:    "Performance Regression Detected",
			Sections: []TeamsSection{
				{
					ActivityTitle:    "🔴 Performance Regression Detected",
					ActivitySubtitle: fmt.Sprintf("Severity: %s", event.Regression.RegressionSeverity),
					Facts: []TeamsFact{
						{
							Name:  "Regression Status",
							Value: fmt.Sprintf("%t", event.Regression.RegressionDetected),
						},
						{
							Name:  "Severity Level",
							Value: event.Regression.RegressionSeverity,
						},
					},
					Markdown: true,
				},
			},
		}
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal Teams payload: %w", err)
	}

	return data, "application/json", nil
}

// createPagerDutyPayload creates PagerDuty webhook payload
func (wm *WebhookManager) createPagerDutyPayload(webhook WebhookConfig, event WebhookEvent) ([]byte, string, error) {
	var payload PagerDutyPayload

	// PagerDuty routing key should be in webhook headers or config
	routingKey, ok := webhook.Headers["X-Routing-Key"]
	if !ok {
		return nil, "", fmt.Errorf("PagerDuty routing key not configured")
	}

	switch event.Type {
	case "alert":
		if event.Alert == nil {
			return nil, "", fmt.Errorf("missing alert")
		}

		// Map severity levels
		severity := "info"
		switch strings.ToLower(event.Alert.Severity) {
		case "critical":
			severity = "critical"
		case "warning":
			severity = "warning"
		}

		payload = PagerDutyPayload{
			RoutingKey:  routingKey,
			EventAction: "trigger",
			DedupKey:    fmt.Sprintf("nephoran-%s-%s", event.Alert.Component, event.Alert.Name),
			Payload: PagerDutyEventPayload{
				Summary:   fmt.Sprintf("Nephoran: %s - %s", event.Alert.Name, event.Alert.Component),
				Source:    "nephoran-performance-monitor",
				Severity:  severity,
				Timestamp: event.Alert.LastFired,
				Component: event.Alert.Component,
				Group:     "performance",
				Class:     "performance-monitoring",
				Details: map[string]interface{}{
					"alert_name":    event.Alert.Name,
					"alert_count":   event.Alert.Count,
					"duration":      event.Alert.Duration.String(),
					"description":   event.Alert.Description,
					"dashboard_url": "https://grafana.nephoran.com/d/nephoran-executive-perf",
				},
			},
			Links: []PagerDutyLink{
				{
					Href: "https://grafana.nephoran.com/d/nephoran-executive-perf",
					Text: "View Performance Dashboard",
				},
			},
		}

	case "regression_detected":
		if event.Regression == nil {
			return nil, "", fmt.Errorf("missing regression analysis")
		}

		payload = PagerDutyPayload{
			RoutingKey:  routingKey,
			EventAction: "trigger",
			DedupKey:    "nephoran-regression-detected",
			Payload: PagerDutyEventPayload{
				Summary:   "Nephoran: Performance Regression Detected",
				Source:    "nephoran-performance-monitor",
				Severity:  "critical",
				Timestamp: time.Now(),
				Component: "performance-monitoring",
				Group:     "regression-detection",
				Class:     "performance-regression",
				Details: map[string]interface{}{
					"regression_severity": event.Regression.RegressionSeverity,
					"performance_changes": event.Regression.PerformanceChange,
					"baseline_period":     event.Regression.BaselineComparison.BaselinePeriod,
					"dashboard_url":       "https://grafana.nephoran.com/d/nephoran-regression-analysis",
				},
			},
			Links: []PagerDutyLink{
				{
					Href: "https://grafana.nephoran.com/d/nephoran-regression-analysis",
					Text: "View Regression Analysis Dashboard",
				},
			},
		}

	default:
		return nil, "", fmt.Errorf("unsupported event type for PagerDuty: %s", event.Type)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal PagerDuty payload: %w", err)
	}

	return data, "application/json", nil
}

// createJSONPayload creates standard JSON webhook payload
func (wm *WebhookManager) createJSONPayload(webhook WebhookConfig, event WebhookEvent) ([]byte, string, error) {
	payload := WebhookPayload{
		EventType:   event.Type,
		Timestamp:   time.Now(),
		Source:      "nephoran-performance-monitor",
		Environment: "production", // TODO: make configurable
		Report:      event.Report,
		CustomData:  event.CustomData,
	}

	// Set severity based on event type
	switch event.Type {
	case "alert":
		if event.Alert != nil {
			payload.Severity = event.Alert.Severity
			payload.Alerts = []AlertInfo{*event.Alert}
		}
	case "regression_detected":
		payload.Severity = "critical"
	case "performance_report":
		if event.Report != nil && len(event.Report.PerformanceClaims) > 0 {
			failedClaims := 0
			for _, claim := range event.Report.PerformanceClaims {
				if claim.Status == "FAIL" {
					failedClaims++
				}
			}

			if failedClaims > 0 {
				payload.Severity = "warning"
			} else {
				payload.Severity = "info"
			}

			payload.Summary = &event.Report.ExecutiveSummary
			payload.Claims = event.Report.PerformanceClaims
		}
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal JSON payload: %w", err)
	}

	return data, "application/json", nil
}

// createCustomPayload creates custom webhook payload using templates
func (wm *WebhookManager) createCustomPayload(webhook WebhookConfig, event WebhookEvent) ([]byte, string, error) {
	templateName := webhook.Templates[event.Type]
	if templateName == "" {
		return nil, "", fmt.Errorf("no template configured for event type: %s", event.Type)
	}

	// Load and execute template
	tmpl := template.New(templateName)

	// TODO: Load template content from file or configuration
	// This would typically load from a template file or configuration
	templateContent := `{
		"event": "{{ .Type }}",
		"timestamp": "{{ .Timestamp }}",
		{{ if .Report }}
		"performance_score": {{ .Report.ExecutiveSummary.OverallScore }},
		"system_health": "{{ .Report.ExecutiveSummary.SystemHealth }}",
		{{ end }}
		{{ if .Alert }}
		"alert_name": "{{ .Alert.Name }}",
		"alert_severity": "{{ .Alert.Severity }}",
		{{ end }}
		"source": "nephoran"
	}`

	tmpl, err := tmpl.Parse(templateContent)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, event); err != nil {
		return nil, "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), "application/json", nil
}

// addAuthentication adds authentication to the webhook request
func (wm *WebhookManager) addAuthentication(req *http.Request, webhook WebhookConfig, payload []byte) error {
	// HMAC signature for webhook verification
	if secret, ok := webhook.Headers["X-Webhook-Secret"]; ok && secret != "" {
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(payload)
		signature := "sha256=" + hex.EncodeToString(h.Sum(nil))
		req.Header.Set("X-Hub-Signature-256", signature)
	}

	// Bearer token authentication
	if token, ok := webhook.Headers["Authorization"]; ok && token != "" {
		if !strings.HasPrefix(token, "Bearer ") {
			token = "Bearer " + token
		}
		req.Header.Set("Authorization", token)
	}

	// API key authentication
	if apiKey, ok := webhook.Headers["X-API-Key"]; ok && apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	return nil
}

// shouldSendAlert determines if an alert should be sent to a specific webhook
func (wm *WebhookManager) shouldSendAlert(webhook WebhookConfig, alert *AlertInfo) bool {
	// Check severity filter
	if severityFilter, ok := webhook.Headers["X-Min-Severity"]; ok {
		minSeverity := parseSeverity(severityFilter)
		alertSeverity := parseSeverity(alert.Severity)
		if alertSeverity < minSeverity {
			return false
		}
	}

	// Check component filter
	if componentFilter, ok := webhook.Headers["X-Component-Filter"]; ok {
		components := strings.Split(componentFilter, ",")
		found := false
		for _, component := range components {
			if strings.TrimSpace(component) == alert.Component {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// parseSeverity converts severity string to numeric value for comparison
func parseSeverity(severity string) int {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical":
		return 3
	case "warning":
		return 2
	case "info":
		return 1
	default:
		return 0
	}
}
