// Package validation provides alert system for regression notifications.

package validation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
)

// AlertSystem handles notifications for regression detections.

type AlertSystem struct {
	config *RegressionConfig

	client *http.Client
}

// NewAlertSystem creates a new alert system.

func NewAlertSystem(config *RegressionConfig) *AlertSystem {

	return &AlertSystem{

		config: config,

		client: &http.Client{

			Timeout: 30 * time.Second,
		},
	}

}

// AlertMessage represents a regression alert message.

type AlertMessage struct {

	// Alert metadata.

	ID string `json:"id"`

	Timestamp time.Time `json:"timestamp"`

	Severity string `json:"severity"`

	Type string `json:"type"` // "regression", "threshold_breach", "anomaly"

	// Context information.

	System string `json:"system"`

	Component string `json:"component"`

	Environment string `json:"environment"`

	// Alert content.

	Title string `json:"title"`

	Description string `json:"description"`

	Impact string `json:"impact"`

	// Regression details.

	BaselineID string `json:"baseline_id,omitempty"`

	RegressionData *RegressionDetection `json:"regression_data,omitempty"`

	// Recommendations.

	Recommendations []AlertRecommendation `json:"recommendations"`

	// Routing information.

	Recipients []string `json:"recipients"`

	Channels []string `json:"channels"`

	Priority string `json:"priority"` // "low", "medium", "high", "critical"

	// Metadata for tracking.

	Tags map[string]string `json:"tags"`

	Links []AlertLink `json:"links"`
}

// AlertRecommendation provides actionable guidance.

type AlertRecommendation struct {
	Action string `json:"action"`

	Description string `json:"description"`

	Priority string `json:"priority"`

	Owner string `json:"owner,omitempty"`

	Deadline string `json:"deadline,omitempty"`
}

// AlertLink provides relevant URLs.

type AlertLink struct {
	Name string `json:"name"`

	URL string `json:"url"`

	Type string `json:"type"` // "dashboard", "runbook", "logs", "metrics"

}

// SlackAlert represents a Slack-formatted alert.

type SlackAlert struct {
	Channel string `json:"channel"`

	Username string `json:"username,omitempty"`

	IconEmoji string `json:"icon_emoji,omitempty"`

	Text string `json:"text"`

	Attachments []SlackAttachment `json:"attachments"`
}

// SlackAttachment represents a Slack message attachment.

type SlackAttachment struct {
	Color string `json:"color"`

	Title string `json:"title"`

	Text string `json:"text"`

	Fields []SlackField `json:"fields"`

	Footer string `json:"footer"`

	Timestamp int64 `json:"ts"`

	Actions []SlackAction `json:"actions,omitempty"`
}

// SlackField represents a field in a Slack attachment.

type SlackField struct {
	Title string `json:"title"`

	Value string `json:"value"`

	Short bool `json:"short"`
}

// SlackAction represents an interactive button in Slack.

type SlackAction struct {
	Type string `json:"type"`

	Text string `json:"text"`

	URL string `json:"url,omitempty"`

	Style string `json:"style,omitempty"`
}

// EmailAlert represents an email alert.

type EmailAlert struct {
	To []string `json:"to"`

	Subject string `json:"subject"`

	Body string `json:"body"`

	BodyHTML string `json:"body_html"`

	Headers map[string]string `json:"headers"`

	Attachments []EmailAttachment `json:"attachments"`
}

// EmailAttachment represents an email attachment.

type EmailAttachment struct {
	Filename string `json:"filename"`

	ContentType string `json:"content_type"`

	Content []byte `json:"content"`
}

// WebhookAlert represents a generic webhook alert.

type WebhookAlert struct {
	URL string `json:"url"`

	Method string `json:"method"`

	Headers map[string]string `json:"headers"`

	Payload interface{} `json:"payload"`
}

// SendRegressionAlert sends comprehensive regression alerts through configured channels.

func (as *AlertSystem) SendRegressionAlert(detection *RegressionDetection) error {

	if !as.config.EnableAlerting {

		ginkgo.By("Alerting disabled - skipping regression alert")

		return nil

	}

	ginkgo.By(fmt.Sprintf("Sending regression alert: %s severity", detection.RegressionSeverity))

	// Create alert message.

	alert := as.createRegressionAlertMessage(detection)

	var errors []string

	// Send Slack alerts.

	if as.config.AlertSlackChannel != "" {

		if err := as.sendSlackAlert(alert); err != nil {

			errors = append(errors, fmt.Sprintf("Slack: %v", err))

		}

	}

	// Send email alerts.

	if len(as.config.AlertEmailRecipients) > 0 {

		if err := as.sendEmailAlert(alert); err != nil {

			errors = append(errors, fmt.Sprintf("Email: %v", err))

		}

	}

	// Send webhook alerts.

	if as.config.AlertWebhookURL != "" {

		if err := as.sendWebhookAlert(alert); err != nil {

			errors = append(errors, fmt.Sprintf("Webhook: %v", err))

		}

	}

	if len(errors) > 0 {

		return fmt.Errorf("alert sending failed: %s", strings.Join(errors, "; "))

	}

	ginkgo.By("Regression alert sent successfully")

	return nil

}

// SendThresholdAlert sends alerts for threshold breaches.

func (as *AlertSystem) SendThresholdAlert(metric string, current, threshold float64, severity string) error {

	if !as.config.EnableAlerting {

		return nil

	}

	alert := &AlertMessage{

		ID: as.generateAlertID("threshold", metric),

		Timestamp: time.Now(),

		Severity: severity,

		Type: "threshold_breach",

		System: "Nephoran Intent Operator",

		Component: metric,

		Environment: "production", // Could be configurable

		Title: fmt.Sprintf("Threshold Breach: %s", metric),

		Description: fmt.Sprintf("Metric %s exceeded threshold: %.2f > %.2f", metric, current, threshold),

		Impact: as.describeThresholdImpact(metric, current, threshold),

		Priority: as.mapSeverityToPriority(severity),

		Recommendations: as.getThresholdRecommendations(metric, current, threshold),

		Tags: map[string]string{

			"metric": metric,

			"type": "threshold",

			"severity": severity,
		},
	}

	return as.sendAlert(alert)

}

// SendAnomalyAlert sends alerts for detected anomalies.

func (as *AlertSystem) SendAnomalyAlert(anomaly *TrendAnomaly) error {

	if !as.config.EnableAlerting {

		return nil

	}

	alert := &AlertMessage{

		ID: as.generateAlertID("anomaly", anomaly.MetricName),

		Timestamp: time.Now(),

		Severity: anomaly.Severity,

		Type: "anomaly",

		System: "Nephoran Intent Operator",

		Component: anomaly.MetricName,

		Environment: "production",

		Title: fmt.Sprintf("Anomaly Detected: %s", anomaly.MetricName),

		Description: fmt.Sprintf("Unusual value detected for %s: %.2f (expected: %.2f, deviation: %.2f)",

			anomaly.MetricName, anomaly.ActualValue, anomaly.ExpectedValue, anomaly.Deviation),

		Impact: fmt.Sprintf("Potential impact: %s", anomaly.PossibleCause),

		Priority: as.mapSeverityToPriority(anomaly.Severity),

		Recommendations: as.getAnomalyRecommendations(anomaly),

		Tags: map[string]string{

			"metric": anomaly.MetricName,

			"type": "anomaly",

			"severity": anomaly.Severity,
		},
	}

	return as.sendAlert(alert)

}

// createRegressionAlertMessage creates a comprehensive alert message for regressions.

func (as *AlertSystem) createRegressionAlertMessage(detection *RegressionDetection) *AlertMessage {

	alert := &AlertMessage{

		ID: as.generateAlertID("regression", "comprehensive"),

		Timestamp: time.Now(),

		Severity: detection.RegressionSeverity,

		Type: "regression",

		System: "Nephoran Intent Operator",

		Component: "Validation Suite",

		Environment: "production",

		Title: fmt.Sprintf("Quality Regression Detected (%s)", strings.ToUpper(detection.RegressionSeverity)),

		Description: as.createRegressionDescription(detection),

		Impact: as.assessRegressionImpact(detection),

		BaselineID: detection.BaselineID,

		RegressionData: detection,

		Priority: as.mapSeverityToPriority(detection.RegressionSeverity),

		Recommendations: as.getRegressionRecommendations(detection),

		Recipients: as.config.AlertEmailRecipients,

		Channels: []string{as.config.AlertSlackChannel},

		Tags: map[string]string{

			"baseline_id": detection.BaselineID,

			"type": "regression",

			"severity": detection.RegressionSeverity,

			"categories": as.getCategoriesString(detection),
		},

		Links: []AlertLink{

			{

				Name: "Regression Dashboard",

				URL: "https://dashboard.example.com/regression", // Would be configurable

				Type: "dashboard",
			},

			{

				Name: "Troubleshooting Runbook",

				URL: "https://runbook.example.com/regression", // Would be configurable

				Type: "runbook",
			},
		},
	}

	return alert

}

// sendAlert routes alert to appropriate channels.

func (as *AlertSystem) sendAlert(alert *AlertMessage) error {

	var errors []string

	if as.config.AlertSlackChannel != "" {

		if err := as.sendSlackAlert(alert); err != nil {

			errors = append(errors, fmt.Sprintf("Slack: %v", err))

		}

	}

	if len(as.config.AlertEmailRecipients) > 0 {

		if err := as.sendEmailAlert(alert); err != nil {

			errors = append(errors, fmt.Sprintf("Email: %v", err))

		}

	}

	if as.config.AlertWebhookURL != "" {

		if err := as.sendWebhookAlert(alert); err != nil {

			errors = append(errors, fmt.Sprintf("Webhook: %v", err))

		}

	}

	if len(errors) > 0 {

		return fmt.Errorf("alert sending failed: %s", strings.Join(errors, "; "))

	}

	return nil

}

// sendSlackAlert sends alert to Slack.

func (as *AlertSystem) sendSlackAlert(alert *AlertMessage) error {

	slackAlert := as.formatSlackAlert(alert)

	payload, err := json.Marshal(slackAlert)

	if err != nil {

		return fmt.Errorf("failed to marshal Slack alert: %w", err)

	}

	// For this implementation, we'll use a webhook URL.

	// In production, this would use the Slack API.

	webhookURL := "https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK" // Placeholder

	resp, err := as.client.Post(webhookURL, "application/json", bytes.NewBuffer(payload))

	if err != nil {

		return fmt.Errorf("failed to send Slack alert: %w", err)

	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("ERROR: failed to read error response body: %v", err)
			return fmt.Errorf("Slack API returned status %d (failed to read response)", resp.StatusCode)
		}
		return fmt.Errorf("Slack API returned status %d: %s", resp.StatusCode, string(body))

	}

	ginkgo.By("Slack alert sent successfully")

	return nil

}

// sendEmailAlert sends alert via email.

func (as *AlertSystem) sendEmailAlert(alert *AlertMessage) error {

	emailAlert := as.formatEmailAlert(alert)

	// In production, this would integrate with an email service like SendGrid, AWS SES, etc.

	// For now, we'll just log the email content.

	ginkgo.By(fmt.Sprintf("Email alert prepared for recipients: %v", emailAlert.To))

	ginkgo.By(fmt.Sprintf("Subject: %s", emailAlert.Subject))

	// Placeholder for actual email sending.

	return nil

}

// sendWebhookAlert sends alert to a webhook endpoint.

func (as *AlertSystem) sendWebhookAlert(alert *AlertMessage) error {

	webhook := &WebhookAlert{

		URL: as.config.AlertWebhookURL,

		Method: "POST",

		Headers: map[string]string{

			"Content-Type": "application/json",

			"User-Agent": "Nephoran-Intent-Operator/1.0",
		},

		Payload: alert,
	}

	payload, err := json.Marshal(webhook.Payload)

	if err != nil {

		return fmt.Errorf("failed to marshal webhook payload: %w", err)

	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer cancel()

	req, err := http.NewRequestWithContext(ctx, webhook.Method, webhook.URL, bytes.NewBuffer(payload))

	if err != nil {

		return fmt.Errorf("failed to create webhook request: %w", err)

	}

	for key, value := range webhook.Headers {

		req.Header.Set(key, value)

	}

	resp, err := as.client.Do(req)

	if err != nil {

		return fmt.Errorf("failed to send webhook alert: %w", err)

	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("ERROR: failed to read webhook error response body: %v", err)
			return fmt.Errorf("webhook returned status %d (failed to read response)", resp.StatusCode)
		}
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(body))

	}

	ginkgo.By("Webhook alert sent successfully")

	return nil

}

// formatSlackAlert formats alert for Slack.

func (as *AlertSystem) formatSlackAlert(alert *AlertMessage) *SlackAlert {

	color := as.getSeverityColor(alert.Severity)

	fields := []SlackField{

		{

			Title: "Severity",

			Value: strings.ToUpper(alert.Severity),

			Short: true,
		},

		{

			Title: "Component",

			Value: alert.Component,

			Short: true,
		},

		{

			Title: "Environment",

			Value: alert.Environment,

			Short: true,
		},

		{

			Title: "Time",

			Value: alert.Timestamp.Format("2006-01-02 15:04:05 MST"),

			Short: true,
		},
	}

	// Add regression-specific fields.

	if alert.RegressionData != nil {

		fields = append(fields,

			SlackField{

				Title: "Performance Regressions",

				Value: fmt.Sprintf("%d detected", len(alert.RegressionData.PerformanceRegressions)),

				Short: true,
			},

			SlackField{

				Title: "Functional Regressions",

				Value: fmt.Sprintf("%d detected", len(alert.RegressionData.FunctionalRegressions)),

				Short: true,
			},

			SlackField{

				Title: "Security Regressions",

				Value: fmt.Sprintf("%d detected", len(alert.RegressionData.SecurityRegressions)),

				Short: true,
			},

			SlackField{

				Title: "Production Regressions",

				Value: fmt.Sprintf("%d detected", len(alert.RegressionData.ProductionRegressions)),

				Short: true,
			},
		)

	}

	// Add recommendations.

	if len(alert.Recommendations) > 0 {

		recText := ""

		for i, rec := range alert.Recommendations {

			if i < 3 { // Limit to first 3 recommendations

				recText += fmt.Sprintf("• %s\n", rec.Description)

			}

		}

		fields = append(fields, SlackField{

			Title: "Recommendations",

			Value: recText,

			Short: false,
		})

	}

	actions := []SlackAction{}

	for _, link := range alert.Links {

		actions = append(actions, SlackAction{

			Type: "button",

			Text: link.Name,

			URL: link.URL,

			Style: "primary",
		})

	}

	return &SlackAlert{

		Channel: as.config.AlertSlackChannel,

		Username: "Nephoran Quality Bot",

		IconEmoji: ":warning:",

		Text: fmt.Sprintf("<!channel> %s", alert.Title),

		Attachments: []SlackAttachment{

			{

				Color: color,

				Title: alert.Title,

				Text: alert.Description,

				Fields: fields,

				Footer: "Nephoran Intent Operator",

				Timestamp: alert.Timestamp.Unix(),

				Actions: actions,
			},
		},
	}

}

// formatEmailAlert formats alert for email.

func (as *AlertSystem) formatEmailAlert(alert *AlertMessage) *EmailAlert {

	subject := fmt.Sprintf("[%s] %s", strings.ToUpper(alert.Severity), alert.Title)

	body := fmt.Sprintf(`

Regression Alert: %s



Description:

%s



Impact:

%s



Details:

- Timestamp: %s

- Component: %s

- Environment: %s

- Severity: %s

- Baseline ID: %s



`, alert.Title, alert.Description, alert.Impact,

		alert.Timestamp.Format("2006-01-02 15:04:05 MST"),

		alert.Component, alert.Environment, alert.Severity, alert.BaselineID)

	if len(alert.Recommendations) > 0 {

		body += "\nRecommended Actions:\n"

		for i, rec := range alert.Recommendations {

			body += fmt.Sprintf("%d. %s\n", i+1, rec.Description)

		}

	}

	if len(alert.Links) > 0 {

		body += "\nUseful Links:\n"

		for _, link := range alert.Links {

			body += fmt.Sprintf("- %s: %s\n", link.Name, link.URL)

		}

	}

	body += "\n--\nGenerated by Nephoran Intent Operator\n"

	return &EmailAlert{

		To: alert.Recipients,

		Subject: subject,

		Body: body,

		Headers: map[string]string{

			"X-Priority": as.getEmailPriority(alert.Priority),
		},
	}

}

// Helper methods.

func (as *AlertSystem) generateAlertID(alertType, component string) string {

	timestamp := time.Now().Format("20060102-150405")

	return fmt.Sprintf("%s-%s-%s", alertType, component, timestamp)

}

func (as *AlertSystem) createRegressionDescription(detection *RegressionDetection) string {

	parts := []string{}

	if len(detection.PerformanceRegressions) > 0 {

		parts = append(parts, fmt.Sprintf("%d performance regression(s)", len(detection.PerformanceRegressions)))

		// Add most significant performance regression.

		for _, reg := range detection.PerformanceRegressions {

			if reg.Severity == "critical" || reg.Severity == "high" {

				parts = append(parts, fmt.Sprintf("- %s degraded by %.1f%%", reg.MetricName, reg.DegradationPercent))

				break

			}

		}

	}

	if len(detection.FunctionalRegressions) > 0 {

		parts = append(parts, fmt.Sprintf("%d functional regression(s)", len(detection.FunctionalRegressions)))

	}

	if len(detection.SecurityRegressions) > 0 {

		parts = append(parts, fmt.Sprintf("%d security regression(s)", len(detection.SecurityRegressions)))

	}

	if len(detection.ProductionRegressions) > 0 {

		parts = append(parts, fmt.Sprintf("%d production readiness regression(s)", len(detection.ProductionRegressions)))

	}

	return strings.Join(parts, ", ")

}

func (as *AlertSystem) assessRegressionImpact(detection *RegressionDetection) string {

	impacts := []string{}

	// Assess performance impact.

	for _, reg := range detection.PerformanceRegressions {

		if reg.Severity == "critical" || reg.Severity == "high" {

			impacts = append(impacts, reg.Impact)

		}

	}

	// Assess functional impact.

	for _, reg := range detection.FunctionalRegressions {

		if reg.Severity == "critical" || reg.Severity == "high" {

			impacts = append(impacts, reg.Impact)

		}

	}

	// Security regressions are always high impact.

	if len(detection.SecurityRegressions) > 0 {

		impacts = append(impacts, "Security posture compromised")

	}

	// Production regressions are always high impact.

	if len(detection.ProductionRegressions) > 0 {

		impacts = append(impacts, "Production readiness affected")

	}

	if len(impacts) == 0 {

		return "System quality degradation detected"

	}

	return strings.Join(impacts, "; ")

}

func (as *AlertSystem) getRegressionRecommendations(detection *RegressionDetection) []AlertRecommendation {

	var recommendations []AlertRecommendation

	// Performance recommendations.

	for _, reg := range detection.PerformanceRegressions {

		if reg.Severity == "critical" || reg.Severity == "high" {

			recommendations = append(recommendations, AlertRecommendation{

				Action: "investigate_performance",

				Description: reg.Recommendation,

				Priority: reg.Severity,

				Owner: "Performance Team",

				Deadline: as.calculateDeadline(reg.Severity),
			})

		}

	}

	// Functional recommendations.

	for _, reg := range detection.FunctionalRegressions {

		if reg.Severity == "critical" || reg.Severity == "high" {

			recommendations = append(recommendations, AlertRecommendation{

				Action: "fix_functional_tests",

				Description: reg.Recommendation,

				Priority: reg.Severity,

				Owner: "Development Team",

				Deadline: as.calculateDeadline(reg.Severity),
			})

		}

	}

	// Security recommendations.

	for _, reg := range detection.SecurityRegressions {

		recommendations = append(recommendations, AlertRecommendation{

			Action: "address_security_issue",

			Description: reg.Recommendation,

			Priority: "high", // Security is always high priority

			Owner: "Security Team",

			Deadline: "24h", // Security issues have tight deadlines

		})

	}

	// Production recommendations.

	for _, reg := range detection.ProductionRegressions {

		recommendations = append(recommendations, AlertRecommendation{

			Action: "fix_production_issue",

			Description: reg.Recommendation,

			Priority: "high",

			Owner: "SRE Team",

			Deadline: as.calculateDeadline("high"),
		})

	}

	// General recommendation if no specific ones.

	if len(recommendations) == 0 {

		recommendations = append(recommendations, AlertRecommendation{

			Action: "investigate_regression",

			Description: "Review regression report and address identified issues",

			Priority: detection.RegressionSeverity,

			Owner: "Development Team",

			Deadline: as.calculateDeadline(detection.RegressionSeverity),
		})

	}

	return recommendations

}

func (as *AlertSystem) getThresholdRecommendations(metric string, current, threshold float64) []AlertRecommendation {

	return []AlertRecommendation{

		{

			Action: "investigate_threshold_breach",

			Description: fmt.Sprintf("Investigate why %s exceeded threshold and implement corrective actions", metric),

			Priority: "medium",

			Owner: "Operations Team",

			Deadline: "4h",
		},
	}

}

func (as *AlertSystem) getAnomalyRecommendations(anomaly *TrendAnomaly) []AlertRecommendation {

	return []AlertRecommendation{

		{

			Action: "investigate_anomaly",

			Description: fmt.Sprintf("Investigate unusual behavior in %s: %s", anomaly.MetricName, anomaly.PossibleCause),

			Priority: anomaly.Severity,

			Owner: "Development Team",

			Deadline: as.calculateDeadline(anomaly.Severity),
		},
	}

}

func (as *AlertSystem) mapSeverityToPriority(severity string) string {

	switch strings.ToLower(severity) {

	case "critical":

		return "critical"

	case "high":

		return "high"

	case "medium":

		return "medium"

	default:

		return "low"

	}

}

func (as *AlertSystem) getSeverityColor(severity string) string {

	switch strings.ToLower(severity) {

	case "critical":

		return "#FF0000" // Red

	case "high":

		return "#FF6600" // Orange

	case "medium":

		return "#FFCC00" // Yellow

	default:

		return "#00CC00" // Green

	}

}

func (as *AlertSystem) getEmailPriority(priority string) string {

	switch strings.ToLower(priority) {

	case "critical":

		return "1"

	case "high":

		return "2"

	case "medium":

		return "3"

	default:

		return "4"

	}

}

func (as *AlertSystem) calculateDeadline(severity string) string {

	switch strings.ToLower(severity) {

	case "critical":

		return "1h"

	case "high":

		return "4h"

	case "medium":

		return "24h"

	default:

		return "72h"

	}

}

func (as *AlertSystem) getCategoriesString(detection *RegressionDetection) string {

	categories := []string{}

	if len(detection.PerformanceRegressions) > 0 {

		categories = append(categories, "performance")

	}

	if len(detection.FunctionalRegressions) > 0 {

		categories = append(categories, "functional")

	}

	if len(detection.SecurityRegressions) > 0 {

		categories = append(categories, "security")

	}

	if len(detection.ProductionRegressions) > 0 {

		categories = append(categories, "production")

	}

	return strings.Join(categories, ",")

}

func (as *AlertSystem) describeThresholdImpact(metric string, current, threshold float64) string {

	return fmt.Sprintf("Performance degradation detected - %s is %.1f%% above acceptable threshold",

		metric, ((current-threshold)/threshold)*100)

}
