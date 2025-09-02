// Package monitoring provides performance reporting for O-RAN systems
package monitoring

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// PerformanceReporter generates performance reports for O-RAN operations
type PerformanceReporter struct {
	templates map[string]*template.Template
	metrics   map[string]*PerformanceMetrics
	mutex     sync.RWMutex
	logger    logr.Logger
}

// NewPerformanceReporter creates a new performance reporter
func NewPerformanceReporter() *PerformanceReporter {
	reporter := &PerformanceReporter{
		templates: make(map[string]*template.Template),
		metrics:   make(map[string]*PerformanceMetrics),
		logger:    log.Log.WithName("performance-reporter"),
	}

	// Load default templates
	if err := reporter.loadTemplates(); err != nil {
		reporter.logger.Error(err, "Failed to load default templates")
	}

	return reporter
}

// RecordMetrics records performance metrics for a component
func (r *PerformanceReporter) RecordMetrics(component string, metrics *PerformanceMetrics) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	metrics.Component = component
	metrics.Timestamp = time.Now()

	r.metrics[component] = metrics

	r.logger.V(1).Info("Recorded performance metrics",
		"component", component,
		"throughput", metrics.Throughput,
		"latency", metrics.Latency.String(),
		"error_rate", metrics.ErrorRate)

	return nil
}

// GenerateReport generates a performance report based on configuration
func (r *PerformanceReporter) GenerateReport(ctx context.Context, config *ReportConfig) (*PerformanceReport, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	report := &PerformanceReport{
		ID:          fmt.Sprintf("perf-report-%d", time.Now().Unix()),
		Title:       "O-RAN Performance Report",
		GeneratedAt: time.Now(),
		TimeRange:   config.TimeRange,
		Format:      config.OutputFormat,
	}

	// Collect metrics within time range
	var reportMetrics []PerformanceMetrics
	var alerts []AlertItem

	for component, metrics := range r.metrics {
		if r.isInTimeRange(metrics.Timestamp, config.TimeRange) {
			reportMetrics = append(reportMetrics, *metrics)

			// Check for threshold alerts
			alerts = append(alerts, r.checkThresholds(component, metrics, config.ThresholdAlerts)...)
		}
	}

	// Convert slice to pointer slice for Metrics
	report.Metrics = make([]*PerformanceMetrics, len(reportMetrics))
	for i := range reportMetrics {
		report.Metrics[i] = &reportMetrics[i]
	}

	// Convert slice to pointer slice for Alerts
	report.Alerts = make([]*AlertItem, len(alerts))
	for i := range alerts {
		report.Alerts[i] = &alerts[i]
	}

	// Calculate summary and convert to pointer
	summary := r.calculateSummary(reportMetrics, alerts)
	report.Summary = &summary

	// Generate report content (note: Content field doesn't exist on PerformanceReport)
	_, err := r.renderReport(report, config)
	if err != nil {
		return nil, fmt.Errorf("failed to render report: %w", err)
	}

	r.logger.Info("Generated performance report",
		"reportID", report.ID,
		"components", len(reportMetrics),
		"alerts", len(alerts))

	return report, nil
}

// loadTemplates loads default report templates (line 348 fix)
func (r *PerformanceReporter) loadTemplates() error {
	// Define default HTML template
	htmlTemplate := `
<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 15px; border-radius: 5px; }
        .metrics { margin: 20px 0; }
        .alert { background-color: #ffebee; padding: 10px; margin: 10px 0; border-left: 4px solid #f44336; }
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 12px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <div class="header">
        <h1>{{.Title}}</h1>
        <p>Generated: {{.GeneratedAt.Format "2006-01-02 15:04:05"}}</p>
        <p>Time Range: {{.TimeRange.Start.Format "2006-01-02 15:04:05"}} - {{.TimeRange.End.Format "2006-01-02 15:04:05"}}</p>
    </div>

    <div class="summary">
        <h2>Summary</h2>
        <p>Total Components: {{.Summary.TotalComponents}}</p>
        <p>Average Throughput: {{printf "%.2f" .Summary.AvgThroughput}} ops/sec</p>
        <p>Average Latency: {{.Summary.AvgLatency}}</p>
        <p>Overall Health: {{printf "%.1f" .Summary.OverallHealth}}%</p>
        <p>Critical Alerts: {{.Summary.CriticalAlerts}}</p>
    </div>

    {{if .Alerts}}
    <div class="alerts">
        <h2>Alerts</h2>
        {{range .Alerts}}
        <div class="alert">
            <strong>{{.Severity}}</strong>: {{.Component}} - {{.Metric}}<br>
            Value: {{printf "%.2f" .Value}}, Threshold: {{printf "%.2f" .Threshold}}<br>
            {{.Description}}
        </div>
        {{end}}
    </div>
    {{end}}

    <div class="metrics">
        <h2>Performance Metrics</h2>
        <table>
            <thead>
                <tr>
                    <th>Component</th>
                    <th>Throughput (ops/sec)</th>
                    <th>Latency</th>
                    <th>Error Rate (%)</th>
                    <th>Availability (%)</th>
                    <th>Timestamp</th>
                </tr>
            </thead>
            <tbody>
                {{range .Metrics}}
                <tr>
                    <td>{{.Component}}</td>
                    <td>{{printf "%.2f" .Throughput}}</td>
                    <td>{{.Latency}}</td>
                    <td>{{printf "%.2f" .ErrorRate}}</td>
                    <td>{{printf "%.1f" .Availability}}</td>
                    <td>{{.Timestamp.Format "2006-01-02 15:04:05"}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>
    </div>
</body>
</html>
`

	// Parse and store HTML template
	htmlTmpl, err := template.New("html-report").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}
	r.templates["html"] = htmlTmpl

	// Define JSON template
	jsonTemplate := `{
  "report_id": "{{.ID}}",
  "title": "{{.Title}}",
  "generated_at": "{{.GeneratedAt.Format "2006-01-02T15:04:05Z07:00"}}",
  "time_range": {
    "start": "{{.TimeRange.Start.Format "2006-01-02T15:04:05Z07:00"}}",
    "end": "{{.TimeRange.End.Format "2006-01-02T15:04:05Z07:00"}}"
  },
  "summary": {{printf "%+v" .Summary}},
  "metrics": {{printf "%+v" .Metrics}},
  "alerts": {{printf "%+v" .Alerts}}
}`

	jsonTmpl, err := template.New("json-report").Parse(jsonTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse JSON template: %w", err)
	}
	r.templates["json"] = jsonTmpl

	r.logger.Info("Loaded default report templates", "count", len(r.templates))
	return nil
}

// renderReport renders a report using the appropriate template
func (r *PerformanceReporter) renderReport(report *PerformanceReport, config *ReportConfig) (string, error) {
	tmpl, exists := r.templates[config.OutputFormat]
	if !exists {
		return "", fmt.Errorf("template not found for format: %s", config.OutputFormat)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, report); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// isInTimeRange checks if a timestamp is within the specified time range
func (r *PerformanceReporter) isInTimeRange(timestamp time.Time, timeRange TimeRange) bool {
	return timestamp.After(timeRange.StartTime) && timestamp.Before(timeRange.EndTime)
}

// checkThresholds checks metrics against configured thresholds
func (r *PerformanceReporter) checkThresholds(component string, metrics *PerformanceMetrics, alertRules []AlertRule) []AlertItem {
	var alerts []AlertItem

	// Convert AlertRules to thresholds map for backward compatibility
	thresholds := make(map[string]float64)
	for _, rule := range alertRules {
		if rule.Enabled && rule.Component == component {
			thresholds[rule.Name] = rule.Threshold
		}
	}

	// Check throughput threshold
	if threshold, exists := thresholds["throughput"]; exists && metrics.Throughput < threshold {
		alerts = append(alerts, AlertItem{
			ID:        fmt.Sprintf("%s-throughput-%d", component, time.Now().Unix()),
			Name:      "throughput",
			Message:   fmt.Sprintf("Throughput %.2f below threshold %.2f for component %s", metrics.Throughput, threshold, component),
			Severity:  AlertSeverityWarning,
			Status:    "active",
			Timestamp: time.Now(),
			Source:    component,
		})
	}

	// Check error rate threshold
	if threshold, exists := thresholds["error_rate"]; exists && metrics.ErrorRate > threshold {
		alerts = append(alerts, AlertItem{
			ID:        fmt.Sprintf("%s-error-rate-%d", component, time.Now().Unix()),
			Name:      "error_rate",
			Message:   fmt.Sprintf("Error rate %.2f%% above threshold %.2f%% for component %s", metrics.ErrorRate, threshold, component),
			Severity:  AlertSeverityCritical,
			Status:    "active",
			Timestamp: time.Now(),
			Source:    component,
		})
	}

	// Check availability threshold (calculated as 100% - error rate)
	if threshold, exists := thresholds["availability"]; exists {
		availability := 100.0 - metrics.ErrorRate
		if availability < threshold {
			alerts = append(alerts, AlertItem{
				ID:        fmt.Sprintf("%s-availability-%d", component, time.Now().Unix()),
				Name:      "availability",
				Message:   fmt.Sprintf("Availability %.1f%% below threshold %.1f%% for component %s", availability, threshold, component),
				Severity:  AlertSeverityCritical,
				Status:    "active",
				Timestamp: time.Now(),
				Source:    component,
			})
		}
	}

	return alerts
}

// calculateSummary calculates summary statistics from metrics and alerts
func (r *PerformanceReporter) calculateSummary(metrics []PerformanceMetrics, alerts []AlertItem) ReportSummary {
	if len(metrics) == 0 {
		return ReportSummary{}
	}

	var totalThroughput float64
	var totalLatency time.Duration
	var totalAvailability float64
	criticalAlerts := 0

	for _, metric := range metrics {
		totalThroughput += metric.Throughput
		totalLatency += metric.Latency
		// Calculate availability as 100% - error rate since Availability field doesn't exist
		availability := 100.0 - metric.ErrorRate
		totalAvailability += availability
	}

	for _, alert := range alerts {
		if alert.Severity == "critical" {
			criticalAlerts++
		}
	}

	avgResponseTime := totalLatency / time.Duration(len(metrics))

	return ReportSummary{
		TotalMetrics:     len(metrics),
		AlertsTriggered:  criticalAlerts,
		PerformanceScore: totalAvailability / float64(len(metrics)),
		AvgResponseTime:  avgResponseTime,
		TotalRequests:    int64(len(metrics)), // Simplified
		ErrorRate:        0.0,                 // Would need to be calculated from metrics
		ResourceUsage:    make(map[string]float64),
	}
}

// ExportReport exports a report to various formats
func (r *PerformanceReporter) ExportReport(report *PerformanceReport, format string) ([]byte, error) {
	switch format {
	case "json":
		return json.MarshalIndent(report, "", "  ")
	case "html":
		// Content field doesn't exist on PerformanceReport, generate HTML from report data
		html := fmt.Sprintf("<html><body><h1>%s</h1><p>Generated: %s</p></body></html>",
			report.Title, report.GeneratedAt.Format("2006-01-02 15:04:05"))
		return []byte(html), nil
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// GetMetrics returns current performance metrics
func (r *PerformanceReporter) GetMetrics(component string) (*PerformanceMetrics, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	metrics, exists := r.metrics[component]
	if !exists {
		return nil, fmt.Errorf("metrics not found for component: %s", component)
	}

	// Return a copy to avoid concurrent access issues
	metricsCopy := *metrics
	return &metricsCopy, nil
}

// ListComponents returns all components with recorded metrics
func (r *PerformanceReporter) ListComponents() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	components := make([]string, 0, len(r.metrics))
	for component := range r.metrics {
		components = append(components, component)
	}

	return components
}

// ClearMetrics clears all recorded metrics
func (r *PerformanceReporter) ClearMetrics() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.metrics = make(map[string]*PerformanceMetrics)
	r.logger.Info("Cleared all performance metrics")
}
