// Package validation provides regression dashboard and reporting capabilities.

package test_validation

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
)

// RegressionDashboard provides comprehensive visualization and reporting for regression testing.

type RegressionDashboard struct {
	framework *RegressionFramework

	config *RegressionConfig

	templatePath string
}

// NewRegressionDashboard creates a new regression dashboard.

func NewRegressionDashboard(framework *RegressionFramework, config *RegressionConfig) *RegressionDashboard {
	return &RegressionDashboard{
		framework: framework,

		config: config,

		templatePath: "./templates", // Default template path

	}
}

// DashboardData aggregates all data needed for dashboard generation.

type DashboardData struct {
	// Metadata.

	GeneratedAt time.Time `json:"generated_at"`

	SystemInfo *SystemInfo `json:"system_info"`

	Configuration *RegressionConfig `json:"configuration"`

	// Current status.

	LatestDetection *RegressionDetection `json:"latest_detection"`

	OverallStatus string `json:"overall_status"` // "healthy", "warning", "critical"

	// Historical data.

	RegressionHistory []*RegressionDetection `json:"regression_history"`

	TrendAnalysis *TrendAnalysis `json:"trend_analysis"`

	// Baseline information.

	Baselines []*BaselineSnapshot `json:"baselines"`

	CurrentBaseline *BaselineSnapshot `json:"current_baseline"`

	// Statistics.

	Statistics *DashboardStatistics `json:"statistics"`

	// Performance metrics.

	Metrics map[string]*MetricSeries `json:"metrics"`

	// Alert information.

	ActiveAlerts []*AlertInfo `json:"active_alerts"`

	RecentAlerts []*AlertInfo `json:"recent_alerts"`
}

// SystemInfo provides system and environment information.

type SystemInfo struct {
	SystemName string `json:"system_name"`

	Version string `json:"version"`

	Environment string `json:"environment"`

	LastDeployment time.Time `json:"last_deployment"`

	UpTime string `json:"uptime"`
}

// DashboardStatistics provides summary statistics.

type DashboardStatistics struct {
	// Regression statistics.

	TotalRegressions int `json:"total_regressions"`

	RegressionsThisWeek int `json:"regressions_this_week"`

	RegressionsThisMonth int `json:"regressions_this_month"`

	AverageResolutionTime string `json:"average_resolution_time"`

	// Quality trends.

	QualityScore float64 `json:"quality_score"`

	QualityTrend string `json:"quality_trend"` // "improving", "stable", "degrading"

	TestStability float64 `json:"test_stability"`

	// Performance statistics.

	AverageLatency string `json:"average_latency"`

	AverageThroughput float64 `json:"average_throughput"`

	AverageAvailability float64 `json:"average_availability"`

	// Baseline statistics.

	BaselineCount int `json:"baseline_count"`

	OldestBaseline string `json:"oldest_baseline"`

	NewestBaseline string `json:"newest_baseline"`

	BaselineFrequency string `json:"baseline_frequency"`
}

// MetricSeries represents time-series data for a metric.

type MetricSeries struct {
	MetricName string `json:"metric_name"`

	Unit string `json:"unit"`

	DataPoints []MetricDataPoint `json:"data_points"`

	Trend string `json:"trend"`

	LastValue float64 `json:"last_value"`

	MinValue float64 `json:"min_value"`

	MaxValue float64 `json:"max_value"`

	Threshold float64 `json:"threshold"`
}

// MetricDataPoint represents a single metric measurement.

type MetricDataPoint struct {
	Timestamp time.Time `json:"timestamp"`

	Value float64 `json:"value"`

	Baseline bool `json:"baseline"`

	Anomaly bool `json:"anomaly"`
}

// AlertInfo provides information about alerts.

type AlertInfo struct {
	ID string `json:"id"`

	Type string `json:"type"`

	Severity string `json:"severity"`

	Title string `json:"title"`

	Description string `json:"description"`

	Timestamp time.Time `json:"timestamp"`

	Status string `json:"status"` // "active", "acknowledged", "resolved"

	Owner string `json:"owner"`
}

// GenerateDashboard creates comprehensive regression dashboard.

func (rd *RegressionDashboard) GenerateDashboard(outputPath string) error {
	ginkgo.By("Generating comprehensive regression dashboard")

	// Collect all dashboard data.

	dashboardData, err := rd.collectDashboardData()
	if err != nil {
		return fmt.Errorf("failed to collect dashboard data: %w", err)
	}

	// Generate JSON report.

	if err := rd.generateJSONDashboard(dashboardData, outputPath); err != nil {
		return fmt.Errorf("failed to generate JSON dashboard: %w", err)
	}

	// Generate HTML dashboard.

	if err := rd.generateHTMLDashboard(dashboardData, outputPath); err != nil {
		return fmt.Errorf("failed to generate HTML dashboard: %w", err)
	}

	// Generate Prometheus metrics.

	if err := rd.generatePrometheusMetrics(dashboardData, outputPath); err != nil {
		return fmt.Errorf("failed to generate Prometheus metrics: %w", err)
	}

	// Generate CSV exports for analysis.

	if err := rd.generateCSVExports(dashboardData, outputPath); err != nil {
		return fmt.Errorf("failed to generate CSV exports: %w", err)
	}

	ginkgo.By(fmt.Sprintf("Regression dashboard generated: %s", outputPath))

	return nil
}

// collectDashboardData gathers all data needed for the dashboard.

func (rd *RegressionDashboard) collectDashboardData() (*DashboardData, error) {
	data := &DashboardData{
		GeneratedAt: time.Now(),

		SystemInfo: rd.getSystemInfo(),

		Configuration: rd.config,

		Statistics: &DashboardStatistics{},

		Metrics: make(map[string]*MetricSeries),

		ActiveAlerts: []*AlertInfo{},

		RecentAlerts: []*AlertInfo{},
	}

	// Load regression history.

	history, err := rd.framework.GetRegressionHistory(30) // Last 30 days
	if err != nil {
		return nil, fmt.Errorf("failed to load regression history: %w", err)
	}

	data.RegressionHistory = history

	if len(history) > 0 {
		data.LatestDetection = history[0] // Most recent
	}

	// Load baselines.

	baselines, err := rd.framework.baselineManager.LoadAllBaselines()
	if err != nil {
		return nil, fmt.Errorf("failed to load baselines: %w", err)
	}

	data.Baselines = baselines

	if len(baselines) > 0 {
		data.CurrentBaseline = baselines[0] // Most recent baseline
	}

	// Generate trend analysis.

	if len(baselines) >= rd.config.MinimumSamples {

		trends, err := rd.framework.GenerateRegressionTrends()
		if err != nil {
			return nil, fmt.Errorf("failed to generate trend analysis: %w", err)
		}

		data.TrendAnalysis = trends

	}

	// Calculate statistics.

	data.Statistics = rd.calculateStatistics(history, baselines, data.TrendAnalysis)

	// Generate metric series.

	data.Metrics = rd.generateMetricSeries(baselines)

	// Determine overall status.

	data.OverallStatus = rd.determineOverallStatus(data.LatestDetection, data.TrendAnalysis)

	// Collect alert information.

	data.ActiveAlerts, data.RecentAlerts = rd.collectAlertInfo(history)

	return data, nil
}

// generateJSONDashboard creates JSON dashboard report.

func (rd *RegressionDashboard) generateJSONDashboard(data *DashboardData, outputPath string) error {
	jsonPath := filepath.Join(outputPath, "regression-dashboard.json")

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(jsonPath, jsonData, 0o640)
}

// generateHTMLDashboard creates HTML dashboard.

func (rd *RegressionDashboard) generateHTMLDashboard(data *DashboardData, outputPath string) error {
	htmlPath := filepath.Join(outputPath, "regression-dashboard.html")

	htmlTemplate := rd.getHTMLTemplate()

	tmpl, err := template.New("dashboard").Parse(htmlTemplate)
	if err != nil {
		return err
	}

	file, err := os.Create(htmlPath)
	if err != nil {
		return err
	}

	defer file.Close()

	return tmpl.Execute(file, data)
}

// generatePrometheusMetrics creates Prometheus metrics export.

func (rd *RegressionDashboard) generatePrometheusMetrics(data *DashboardData, outputPath string) error {
	metricsPath := filepath.Join(outputPath, "regression-metrics.prom")

	var metrics strings.Builder

	// System metrics.

	metrics.WriteString("# HELP nephoran_regression_total_score Total regression score\n")

	metrics.WriteString("# TYPE nephoran_regression_total_score gauge\n")

	if data.LatestDetection != nil && data.CurrentBaseline != nil {
		metrics.WriteString(fmt.Sprintf("nephoran_regression_total_score %d\n", data.CurrentBaseline.Results.TotalScore))
	}

	// Regression count metrics.

	metrics.WriteString("# HELP nephoran_regressions_detected_total Total regressions detected\n")

	metrics.WriteString("# TYPE nephoran_regressions_detected_total counter\n")

	metrics.WriteString(fmt.Sprintf("nephoran_regressions_detected_total %d\n", data.Statistics.TotalRegressions))

	// Performance metrics.

	if data.CurrentBaseline != nil {

		results := data.CurrentBaseline.Results

		metrics.WriteString("# HELP nephoran_p95_latency_seconds P95 latency in seconds\n")

		metrics.WriteString("# TYPE nephoran_p95_latency_seconds gauge\n")

		metrics.WriteString(fmt.Sprintf("nephoran_p95_latency_seconds %f\n", results.P95Latency.Seconds()))

		metrics.WriteString("# HELP nephoran_throughput_intents_per_minute Throughput in intents per minute\n")

		metrics.WriteString("# TYPE nephoran_throughput_intents_per_minute gauge\n")

		metrics.WriteString(fmt.Sprintf("nephoran_throughput_intents_per_minute %f\n", results.ThroughputAchieved))

		metrics.WriteString("# HELP nephoran_availability_percent Availability percentage\n")

		metrics.WriteString("# TYPE nephoran_availability_percent gauge\n")

		metrics.WriteString(fmt.Sprintf("nephoran_availability_percent %f\n", results.AvailabilityAchieved))

	}

	// Quality trend metrics.

	if data.TrendAnalysis != nil && data.TrendAnalysis.OverallTrend != nil {

		qualityScore := data.Statistics.QualityScore

		metrics.WriteString("# HELP nephoran_quality_score Overall quality score\n")

		metrics.WriteString("# TYPE nephoran_quality_score gauge\n")

		metrics.WriteString(fmt.Sprintf("nephoran_quality_score %f\n", qualityScore))

	}

	return os.WriteFile(metricsPath, []byte(metrics.String()), 0o640)
}

// generateCSVExports creates CSV files for data analysis.

func (rd *RegressionDashboard) generateCSVExports(data *DashboardData, outputPath string) error {
	// Export regression history.

	if err := rd.exportRegressionHistoryCSV(data.RegressionHistory, outputPath); err != nil {
		return err
	}

	// Export performance metrics.

	if err := rd.exportPerformanceMetricsCSV(data.Metrics, outputPath); err != nil {
		return err
	}

	// Export baseline comparison.

	if err := rd.exportBaselineComparisonCSV(data.Baselines, outputPath); err != nil {
		return err
	}

	return nil
}

// Helper methods for data collection and analysis.

func (rd *RegressionDashboard) getSystemInfo() *SystemInfo {
	return &SystemInfo{
		SystemName: "Nephoran Intent Operator",

		Version: "v1.0.0", // Would be retrieved from build info

		Environment: rd.getEnvironmentName(),

		LastDeployment: time.Now().Add(-24 * time.Hour), // Placeholder

		UpTime: "72h", // Placeholder

	}
}

func (rd *RegressionDashboard) getEnvironmentName() string {
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		return env
	}

	return "development"
}

func (rd *RegressionDashboard) calculateStatistics(history []*RegressionDetection, baselines []*BaselineSnapshot, trends *TrendAnalysis) *DashboardStatistics {
	stats := &DashboardStatistics{
		BaselineCount: len(baselines),
	}

	// Calculate regression statistics.

	now := time.Now()

	weekAgo := now.AddDate(0, 0, -7)

	monthAgo := now.AddDate(0, -1, 0)

	for _, detection := range history {
		if detection.HasRegression {

			stats.TotalRegressions++

			if detection.ComparisonTime.After(weekAgo) {
				stats.RegressionsThisWeek++
			}

			if detection.ComparisonTime.After(monthAgo) {
				stats.RegressionsThisMonth++
			}

		}
	}

	// Calculate averages from baselines.

	if len(baselines) > 0 {

		var latencySum time.Duration

		var throughputSum, availabilitySum float64

		validSamples := 0

		for _, baseline := range baselines {
			if baseline.Results.P95Latency > 0 {

				latencySum += baseline.Results.P95Latency

				throughputSum += baseline.Results.ThroughputAchieved

				availabilitySum += baseline.Results.AvailabilityAchieved

				validSamples++

			}
		}

		if validSamples > 0 {

			stats.AverageLatency = (latencySum / time.Duration(validSamples)).String()

			stats.AverageThroughput = throughputSum / float64(validSamples)

			stats.AverageAvailability = availabilitySum / float64(validSamples)

		}

		// Baseline age information.

		oldest := baselines[len(baselines)-1]

		newest := baselines[0]

		stats.OldestBaseline = oldest.Timestamp.Format("2006-01-02 15:04")

		stats.NewestBaseline = newest.Timestamp.Format("2006-01-02 15:04")

		// Calculate quality score and trends.

		if newest.Results != nil {
			stats.QualityScore = float64(newest.Results.TotalScore)
		}

		if trends != nil && trends.OverallTrend != nil {
			stats.QualityTrend = trends.OverallTrend.Direction
		}

	}

	return stats
}

func (rd *RegressionDashboard) generateMetricSeries(baselines []*BaselineSnapshot) map[string]*MetricSeries {
	metrics := make(map[string]*MetricSeries)

	if len(baselines) == 0 {
		return metrics
	}

	// P95 Latency series.

	p95Points := []MetricDataPoint{}

	for _, baseline := range baselines {
		if baseline.Results.P95Latency > 0 {
			p95Points = append(p95Points, MetricDataPoint{
				Timestamp: baseline.Timestamp,

				Value: float64(baseline.Results.P95Latency.Nanoseconds()),

				Baseline: true,
			})
		}
	}

	if len(p95Points) > 0 {

		metrics["p95_latency"] = &MetricSeries{
			MetricName: "P95 Latency",

			Unit: "nanoseconds",

			DataPoints: p95Points,

			Trend: rd.calculateTrend(p95Points),

			LastValue: p95Points[len(p95Points)-1].Value,

			Threshold: float64((2 * time.Second).Nanoseconds()),
		}

		// Calculate min/max.

		var minVal, maxVal float64

		for i, point := range p95Points {
			if i == 0 {

				minVal = point.Value

				maxVal = point.Value

			} else {

				if point.Value < minVal {
					minVal = point.Value
				}

				if point.Value > maxVal {
					maxVal = point.Value
				}

			}
		}

		metrics["p95_latency"].MinValue = minVal

		metrics["p95_latency"].MaxValue = maxVal

	}

	// Throughput series.

	throughputPoints := []MetricDataPoint{}

	for _, baseline := range baselines {
		if baseline.Results.ThroughputAchieved > 0 {
			throughputPoints = append(throughputPoints, MetricDataPoint{
				Timestamp: baseline.Timestamp,

				Value: baseline.Results.ThroughputAchieved,

				Baseline: true,
			})
		}
	}

	if len(throughputPoints) > 0 {
		metrics["throughput"] = &MetricSeries{
			MetricName: "Throughput",

			Unit: "intents/minute",

			DataPoints: throughputPoints,

			Trend: rd.calculateTrend(throughputPoints),

			LastValue: throughputPoints[len(throughputPoints)-1].Value,

			Threshold: 45.0,
		}
	}

	// Overall score series.

	scorePoints := []MetricDataPoint{}

	for _, baseline := range baselines {
		scorePoints = append(scorePoints, MetricDataPoint{
			Timestamp: baseline.Timestamp,

			Value: float64(baseline.Results.TotalScore),

			Baseline: true,
		})
	}

	if len(scorePoints) > 0 {
		metrics["total_score"] = &MetricSeries{
			MetricName: "Total Score",

			Unit: "points",

			DataPoints: scorePoints,

			Trend: rd.calculateTrend(scorePoints),

			LastValue: scorePoints[len(scorePoints)-1].Value,

			Threshold: 90.0,
		}
	}

	return metrics
}

func (rd *RegressionDashboard) calculateTrend(points []MetricDataPoint) string {
	if len(points) < 2 {
		return "stable"
	}

	first := points[0].Value

	last := points[len(points)-1].Value

	diff := (last - first) / first * 100

	if diff > 5 {
		return "improving"
	} else if diff < -5 {
		return "degrading"
	}

	return "stable"
}

func (rd *RegressionDashboard) determineOverallStatus(latestDetection *RegressionDetection, trends *TrendAnalysis) string {
	if latestDetection == nil {
		return "unknown"
	}

	if !latestDetection.HasRegression {

		// Check trends for early warning.

		if trends != nil && trends.OverallTrend != nil {
			if trends.OverallTrend.Direction == "degrading" {
				return "warning"
			}
		}

		return "healthy"

	}

	// Has regressions - classify severity.

	switch latestDetection.RegressionSeverity {

	case "critical":

		return "critical"

	case "high":

		return "critical"

	case "medium":

		return "warning"

	default:

		return "warning"

	}
}

func (rd *RegressionDashboard) collectAlertInfo(history []*RegressionDetection) ([]*AlertInfo, []*AlertInfo) {
	var activeAlerts, recentAlerts []*AlertInfo

	// For demonstration, create sample alerts based on regressions.

	for _, detection := range history {
		if detection.HasRegression {

			alert := &AlertInfo{
				ID: fmt.Sprintf("regression-%s", detection.ComparisonTime.Format("20060102-150405")),

				Type: "regression",

				Severity: detection.RegressionSeverity,

				Title: fmt.Sprintf("Regression Detected (%s)", detection.RegressionSeverity),

				Description: fmt.Sprintf("Quality regression detected with %d total issues",

					len(detection.PerformanceRegressions)+len(detection.FunctionalRegressions)+len(detection.SecurityRegressions)+len(detection.ProductionRegressions)),

				Timestamp: detection.ComparisonTime,

				Status: "active", // Would be determined by actual alert tracking

				Owner: "DevOps Team",
			}

			// Recent alerts are from last 7 days.

			if detection.ComparisonTime.After(time.Now().AddDate(0, 0, -7)) {

				recentAlerts = append(recentAlerts, alert)

				// Active alerts are unresolved critical/high severity.

				if detection.RegressionSeverity == "critical" || detection.RegressionSeverity == "high" {
					activeAlerts = append(activeAlerts, alert)
				}

			}

		}
	}

	return activeAlerts, recentAlerts
}

// CSV export methods.

func (rd *RegressionDashboard) exportRegressionHistoryCSV(history []*RegressionDetection, outputPath string) error {
	csvPath := filepath.Join(outputPath, "regression-history.csv")

	file, err := os.Create(csvPath)
	if err != nil {
		return err
	}

	defer file.Close()

	// Write CSV header.

	file.WriteString("timestamp,baseline_id,has_regression,severity,performance_count,functional_count,security_count,production_count\n")

	// Write data rows.

	for _, detection := range history {
		fmt.Fprintf(file, "%s,%s,%t,%s,%d,%d,%d,%d\n",

			detection.ComparisonTime.Format(time.RFC3339),

			detection.BaselineID,

			detection.HasRegression,

			detection.RegressionSeverity,

			len(detection.PerformanceRegressions),

			len(detection.FunctionalRegressions),

			len(detection.SecurityRegressions),

			len(detection.ProductionRegressions))
	}

	return nil
}

func (rd *RegressionDashboard) exportPerformanceMetricsCSV(metrics map[string]*MetricSeries, outputPath string) error {
	csvPath := filepath.Join(outputPath, "performance-metrics.csv")

	file, err := os.Create(csvPath)
	if err != nil {
		return err
	}

	defer file.Close()

	// Write CSV header.

	file.WriteString("timestamp,metric_name,value,unit,threshold_exceeded\n")

	// Write data rows.

	for _, metric := range metrics {
		for _, point := range metric.DataPoints {

			thresholdExceeded := point.Value > metric.Threshold

			fmt.Fprintf(file, "%s,%s,%f,%s,%t\n",

				point.Timestamp.Format(time.RFC3339),

				metric.MetricName,

				point.Value,

				metric.Unit,

				thresholdExceeded)

		}
	}

	return nil
}

func (rd *RegressionDashboard) exportBaselineComparisonCSV(baselines []*BaselineSnapshot, outputPath string) error {
	csvPath := filepath.Join(outputPath, "baseline-comparison.csv")

	file, err := os.Create(csvPath)
	if err != nil {
		return err
	}

	defer file.Close()

	// Write CSV header.

	file.WriteString("timestamp,baseline_id,total_score,functional_score,performance_score,security_score,production_score,p95_latency_ms,throughput,availability\n")

	// Sort baselines by timestamp.

	sort.Slice(baselines, func(i, j int) bool {
		return baselines[i].Timestamp.Before(baselines[j].Timestamp)
	})

	// Write data rows.

	for _, baseline := range baselines {
		fmt.Fprintf(file, "%s,%s,%d,%d,%d,%d,%d,%f,%f,%f\n",

			baseline.Timestamp.Format(time.RFC3339),

			baseline.ID,

			baseline.Results.TotalScore,

			baseline.Results.FunctionalScore,

			baseline.Results.PerformanceScore,

			baseline.Results.SecurityScore,

			baseline.Results.ProductionScore,

			float64(baseline.Results.P95Latency.Nanoseconds())/1e6, // Convert to milliseconds

			baseline.Results.ThroughputAchieved,

			baseline.Results.AvailabilityAchieved)
	}

	return nil
}

// HTML template for dashboard.

func (rd *RegressionDashboard) getHTMLTemplate() string {
	return `<!DOCTYPE html>

<html>

<head>

    <title>Nephoran Intent Operator - Regression Dashboard</title>

    <meta charset="utf-8">

    <meta name="viewport" content="width=device-width, initial-scale=1">

    <style>

        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background-color: #f5f5f5; }

        .header { background: #333; color: white; padding: 20px; margin: -20px -20px 20px -20px; }

        .status-{{ .OverallStatus }} { padding: 10px; border-radius: 5px; margin: 10px 0; }

        .status-healthy { background-color: #d4edda; border: 1px solid #c3e6cb; color: #155724; }

        .status-warning { background-color: #fff3cd; border: 1px solid #ffeaa7; color: #856404; }

        .status-critical { background-color: #f8d7da; border: 1px solid #f5c6cb; color: #721c24; }

        .metrics-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; margin: 20px 0; }

        .metric-card { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }

        .metric-value { font-size: 2em; font-weight: bold; color: #333; }

        .metric-label { color: #666; margin-bottom: 5px; }

        .regression-list { background: white; padding: 20px; border-radius: 8px; margin: 20px 0; }

        .regression-item { padding: 10px; border-left: 4px solid #dc3545; margin: 10px 0; background: #f8f9fa; }

        .footer { text-align: center; margin-top: 40px; color: #666; font-size: 0.9em; }

    </style>

</head>

<body>

    <div class="header">

        <h1>Nephoran Intent Operator - Regression Dashboard</h1>

        <p>Generated: {{ .GeneratedAt.Format "2006-01-02 15:04:05 MST" }}</p>

    </div>



    <div class="status-{{ .OverallStatus }}">

        <h2>System Status: {{ .OverallStatus | title }}</h2>

        {{- if .LatestDetection }}

        {{- if .LatestDetection.HasRegression }}

        <p>Latest regression detected: {{ .LatestDetection.RegressionSeverity }} severity</p>

        {{- else }}

        <p>No regressions detected in latest validation</p>

        {{- end }}

        {{- end }}

    </div>



    <div class="metrics-grid">

        <div class="metric-card">

            <div class="metric-label">Quality Score</div>

            <div class="metric-value">{{ printf "%.1f" .Statistics.QualityScore }}</div>

            <div>Trend: {{ .Statistics.QualityTrend }}</div>

        </div>

        

        <div class="metric-card">

            <div class="metric-label">Total Regressions</div>

            <div class="metric-value">{{ .Statistics.TotalRegressions }}</div>

            <div>This week: {{ .Statistics.RegressionsThisWeek }}</div>

        </div>

        

        <div class="metric-card">

            <div class="metric-label">Average Latency</div>

            <div class="metric-value">{{ .Statistics.AverageLatency }}</div>

        </div>

        

        <div class="metric-card">

            <div class="metric-label">Average Throughput</div>

            <div class="metric-value">{{ printf "%.1f" .Statistics.AverageThroughput }}</div>

            <div>intents/minute</div>

        </div>

        

        <div class="metric-card">

            <div class="metric-label">Availability</div>

            <div class="metric-value">{{ printf "%.2f%%" .Statistics.AverageAvailability }}</div>

        </div>

        

        <div class="metric-card">

            <div class="metric-label">Baselines</div>

            <div class="metric-value">{{ .Statistics.BaselineCount }}</div>

            <div>Latest: {{ .Statistics.NewestBaseline }}</div>

        </div>

    </div>



    {{- if .ActiveAlerts }}

    <div class="regression-list">

        <h3>Active Alerts ({{ len .ActiveAlerts }})</h3>

        {{- range .ActiveAlerts }}

        <div class="regression-item">

            <strong>{{ .Title }}</strong> - {{ .Severity | title }}

            <br>{{ .Description }}

            <br><small>{{ .Timestamp.Format "2006-01-02 15:04" }}</small>

        </div>

        {{- end }}

    </div>

    {{- end }}



    <div class="footer">

        <p>Nephoran Intent Operator Regression Dashboard</p>

        <p>System: {{ .SystemInfo.SystemName }} {{ .SystemInfo.Version }} | Environment: {{ .SystemInfo.Environment }}</p>

    </div>

</body>

</html>`
}
