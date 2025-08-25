// Copyright 2024 Nephoran Intent Operator Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package reporting

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"sync"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
)

// SLATarget represents an SLA target and its configuration
type SLATarget struct {
	Name        string        `json:"name"`
	Type        string        `json:"type"` // availability, latency, throughput
	Target      float64       `json:"target"`
	Unit        string        `json:"unit"`
	Query       string        `json:"query"`
	Window      time.Duration `json:"window"`
	ErrorBudget float64       `json:"error_budget"`
}

// SLAStatus represents the current status of an SLA
type SLAStatus struct {
	Target           SLATarget      `json:"target"`
	CurrentValue     float64        `json:"current_value"`
	ComplianceStatus string         `json:"compliance_status"` // compliant, at_risk, violation
	ErrorBudgetUsed  float64        `json:"error_budget_used"`
	BurnRate         float64        `json:"burn_rate"`
	TimeToExhaustion *time.Time     `json:"time_to_exhaustion,omitempty"`
	LastUpdated      time.Time      `json:"last_updated"`
	History          []DataPoint    `json:"history"`
	Violations       []SLAViolation `json:"violations"`
}

// SLAViolation represents an SLA violation event
type SLAViolation struct {
	ID             string                 `json:"id"`
	Target         string                 `json:"target"`
	StartTime      time.Time              `json:"start_time"`
	EndTime        *time.Time             `json:"end_time,omitempty"`
	Duration       time.Duration          `json:"duration"`
	Severity       string                 `json:"severity"`
	ImpactValue    float64                `json:"impact_value"`
	RootCause      string                 `json:"root_cause"`
	Resolution     string                 `json:"resolution"`
	BusinessImpact BusinessImpact         `json:"business_impact"`
	Metadata       map[string]interface{} `json:"metadata"`
}


// DataPoint represents a time-series data point
type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// SLAReport represents a comprehensive SLA report
type SLAReport struct {
	ID              string                 `json:"id"`
	GeneratedAt     time.Time              `json:"generated_at"`
	ReportType      string                 `json:"report_type"` // daily, weekly, monthly, ad_hoc
	Period          Period                 `json:"period"`
	OverallSLAScore float64                `json:"overall_sla_score"`
	SLAStatuses     []SLAStatus            `json:"sla_statuses"`
	Summary         ReportSummary          `json:"summary"`
	Trends          ReportTrends           `json:"trends"`
	Recommendations []Recommendation       `json:"recommendations"`
	BusinessMetrics BusinessMetrics        `json:"business_metrics"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// Period represents a time period for reporting
type Period struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Duration  string    `json:"duration"`
}

// ReportSummary provides high-level summary statistics
type ReportSummary struct {
	TotalViolations     int     `json:"total_violations"`
	CriticalViolations  int     `json:"critical_violations"`
	AverageAvailability float64 `json:"average_availability"`
	AverageLatency      float64 `json:"average_latency"`
	AverageThroughput   float64 `json:"average_throughput"`
	MTTR                float64 `json:"mttr"` // Mean Time to Resolution in minutes
	MTBF                float64 `json:"mtbf"` // Mean Time Between Failures in hours
}

// ReportTrends shows trending information
type ReportTrends struct {
	AvailabilityTrend string  `json:"availability_trend"` // improving, stable, degrading
	LatencyTrend      string  `json:"latency_trend"`
	ThroughputTrend   string  `json:"throughput_trend"`
	ViolationTrend    string  `json:"violation_trend"`
	TrendConfidence   float64 `json:"trend_confidence"`
}


// BusinessMetrics provides business context for SLA performance
type BusinessMetrics struct {
	TotalRevenue         float64 `json:"total_revenue"`
	RevenueAtRisk        float64 `json:"revenue_at_risk"`
	CustomerSatisfaction float64 `json:"customer_satisfaction"`
	CompetitivePosition  string  `json:"competitive_position"`
}

// SLAReporter manages SLA monitoring and reporting
type SLAReporter struct {
	promClient      v1.API
	slaTargets      []SLATarget
	violationStore  ViolationStore
	reportGenerator ReportGenerator
	logger          *logrus.Logger
	mu              sync.RWMutex
	currentStatuses map[string]*SLAStatus
	stopCh          chan struct{}
}

// ViolationStore interface for storing SLA violations
type ViolationStore interface {
	Store(violation SLAViolation) error
	Get(id string) (*SLAViolation, error)
	List(target string, since time.Time) ([]SLAViolation, error)
	Update(violation SLAViolation) error
}

// ReportGenerator interface for generating reports
type ReportGenerator interface {
	GeneratePDF(report SLAReport) ([]byte, error)
	GenerateHTML(report SLAReport) (string, error)
	GenerateCSV(report SLAReport) (string, error)
	GenerateJSON(report SLAReport) ([]byte, error)
}

// NewSLAReporter creates a new SLA reporter instance
func NewSLAReporter(promClient v1.API, logger *logrus.Logger) *SLAReporter {
	return &SLAReporter{
		promClient:      promClient,
		logger:          logger,
		currentStatuses: make(map[string]*SLAStatus),
		stopCh:          make(chan struct{}),
		slaTargets: []SLATarget{
			{
				Name:        "System Availability",
				Type:        "availability",
				Target:      99.95,
				Unit:        "percent",
				Query:       "100 * (1 - rate(nephoran_intent_failures_total[5m]) / rate(nephoran_intent_requests_total[5m]))",
				Window:      time.Hour * 24 * 30, // 30 days
				ErrorBudget: 0.05,                // 0.05% error budget
			},
			{
				Name:        "Intent Processing Latency",
				Type:        "latency",
				Target:      2.0,
				Unit:        "seconds",
				Query:       "histogram_quantile(0.95, rate(nephoran_intent_processing_duration_seconds_bucket[5m]))",
				Window:      time.Hour * 24,
				ErrorBudget: 0.1, // 10% above target is violation
			},
			{
				Name:        "Intent Throughput",
				Type:        "throughput",
				Target:      45.0,
				Unit:        "intents/minute",
				Query:       "rate(nephoran_intent_processed_total[1m]) * 60",
				Window:      time.Hour * 4,
				ErrorBudget: 0.2, // 20% below target is violation
			},
		},
	}
}

// Start begins the SLA monitoring process
func (s *SLAReporter) Start(ctx context.Context) error {
	s.logger.Info("Starting SLA Reporter")

	// Initialize current statuses
	for _, target := range s.slaTargets {
		s.currentStatuses[target.Name] = &SLAStatus{
			Target:           target,
			ComplianceStatus: "unknown",
			LastUpdated:      time.Now(),
			History:          make([]DataPoint, 0),
			Violations:       make([]SLAViolation, 0),
		}
	}

	// Start monitoring loop
	go s.monitoringLoop(ctx)

	// Start violation detection
	go s.violationDetectionLoop(ctx)

	return nil
}

// Stop stops the SLA monitoring process
func (s *SLAReporter) Stop() {
	s.logger.Info("Stopping SLA Reporter")
	close(s.stopCh)
}

// GetCurrentStatus returns the current SLA status for all targets
func (s *SLAReporter) GetCurrentStatus() []SLAStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	statuses := make([]SLAStatus, 0, len(s.currentStatuses))
	for _, status := range s.currentStatuses {
		statuses = append(statuses, *status)
	}

	return statuses
}

// GetSLAStatus returns the current status for a specific SLA target
func (s *SLAReporter) GetSLAStatus(targetName string) (*SLAStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status, exists := s.currentStatuses[targetName]
	if !exists {
		return nil, fmt.Errorf("SLA target %s not found", targetName)
	}

	return status, nil
}

// GenerateReport generates a comprehensive SLA report
func (s *SLAReporter) GenerateReport(reportType string, period Period) (*SLAReport, error) {
	s.logger.WithFields(logrus.Fields{
		"report_type": reportType,
		"period":      period,
	}).Info("Generating SLA report")

	report := &SLAReport{
		ID:          fmt.Sprintf("sla-report-%d", time.Now().Unix()),
		GeneratedAt: time.Now(),
		ReportType:  reportType,
		Period:      period,
		SLAStatuses: s.GetCurrentStatus(),
		Metadata:    make(map[string]interface{}),
	}

	// Calculate overall SLA score
	report.OverallSLAScore = s.calculateOverallSLAScore(report.SLAStatuses)

	// Generate summary
	summary, err := s.generateSummary(period)
	if err != nil {
		return nil, fmt.Errorf("failed to generate summary: %v", err)
	}
	report.Summary = summary

	// Analyze trends
	trends, err := s.analyzeTrends(period)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze trends: %v", err)
	}
	report.Trends = trends

	// Generate recommendations
	report.Recommendations = s.generateRecommendations(report.SLAStatuses, trends)

	// Calculate business metrics
	businessMetrics, err := s.calculateBusinessMetrics(period)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate business metrics: %v", err)
	}
	report.BusinessMetrics = businessMetrics

	return report, nil
}

// monitoringLoop continuously monitors SLA targets
func (s *SLAReporter) monitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.updateSLAStatuses(ctx)
		}
	}
}

// violationDetectionLoop detects and handles SLA violations
func (s *SLAReporter) violationDetectionLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute) // Check for violations every minute
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.detectViolations(ctx)
		}
	}
}

// updateSLAStatuses updates the current status of all SLA targets
func (s *SLAReporter) updateSLAStatuses(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, target := range s.slaTargets {
		status := s.currentStatuses[target.Name]

		// Query current value
		value, err := s.queryCurrentValue(ctx, target.Query)
		if err != nil {
			s.logger.WithError(err).WithField("target", target.Name).Error("Failed to query current value")
			continue
		}

		// Update status
		status.CurrentValue = value
		status.LastUpdated = time.Now()

		// Add to history (keep last 100 points)
		dataPoint := DataPoint{
			Timestamp: time.Now(),
			Value:     value,
		}
		status.History = append(status.History, dataPoint)
		if len(status.History) > 100 {
			status.History = status.History[1:]
		}

		// Calculate error budget usage
		status.ErrorBudgetUsed = s.calculateErrorBudgetUsage(target, value)

		// Calculate burn rate
		status.BurnRate = s.calculateBurnRate(status.History)

		// Update compliance status
		status.ComplianceStatus = s.determineComplianceStatus(target, value, status.ErrorBudgetUsed)

		// Calculate time to exhaustion if burning error budget
		if status.BurnRate > 0 {
			remainingBudget := target.ErrorBudget - status.ErrorBudgetUsed
			if remainingBudget > 0 {
				timeToExhaustion := time.Now().Add(time.Duration(float64(time.Hour) * remainingBudget / status.BurnRate))
				status.TimeToExhaustion = &timeToExhaustion
			}
		}
	}
}

// queryCurrentValue queries Prometheus for the current value of a metric
func (s *SLAReporter) queryCurrentValue(ctx context.Context, query string) (float64, error) {
	result, warnings, err := s.promClient.Query(ctx, query, time.Now())
	if err != nil {
		return 0, err
	}

	if len(warnings) > 0 {
		s.logger.WithField("warnings", warnings).Warn("Prometheus query returned warnings")
	}

	switch result.Type() {
	case model.ValVector:
		vector := result.(model.Vector)
		if len(vector) == 0 {
			return 0, fmt.Errorf("no data returned from query")
		}
		return float64(vector[0].Value), nil
	case model.ValScalar:
		scalar := result.(*model.Scalar)
		return float64(scalar.Value), nil
	default:
		return 0, fmt.Errorf("unexpected result type: %v", result.Type())
	}
}

// calculateErrorBudgetUsage calculates how much of the error budget has been used
func (s *SLAReporter) calculateErrorBudgetUsage(target SLATarget, currentValue float64) float64 {
	switch target.Type {
	case "availability":
		if currentValue >= target.Target {
			return 0 // No error budget used if above target
		}
		shortage := target.Target - currentValue
		return shortage / target.ErrorBudget
	case "latency":
		if currentValue <= target.Target {
			return 0 // No error budget used if below target
		}
		excess := currentValue - target.Target
		maxAllowed := target.Target * (1 + target.ErrorBudget)
		return excess / (maxAllowed - target.Target)
	case "throughput":
		if currentValue >= target.Target {
			return 0 // No error budget used if above target
		}
		shortage := target.Target - currentValue
		minAllowed := target.Target * (1 - target.ErrorBudget)
		return shortage / (target.Target - minAllowed)
	default:
		return 0
	}
}

// calculateBurnRate calculates the current error budget burn rate
func (s *SLAReporter) calculateBurnRate(history []DataPoint) float64 {
	if len(history) < 2 {
		return 0
	}

	// Calculate burn rate based on last 10 points
	start := len(history) - 10
	if start < 0 {
		start = 0
	}

	recent := history[start:]
	if len(recent) < 2 {
		return 0
	}

	// Simple linear regression to calculate rate of change
	var sumX, sumY, sumXY, sumX2 float64
	n := float64(len(recent))

	for i, point := range recent {
		x := float64(i)
		y := point.Value
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	if n*sumX2-sumX*sumX == 0 {
		return 0
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	return slope
}

// determineComplianceStatus determines the current compliance status
func (s *SLAReporter) determineComplianceStatus(target SLATarget, currentValue, errorBudgetUsed float64) string {
	if errorBudgetUsed <= 0 {
		return "compliant"
	} else if errorBudgetUsed < 0.8 {
		return "at_risk"
	} else {
		return "violation"
	}
}

// detectViolations detects and handles SLA violations
func (s *SLAReporter) detectViolations(ctx context.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, status := range s.currentStatuses {
		if status.ComplianceStatus == "violation" {
			// Check if this is a new violation or ongoing
			lastViolation := s.getLastViolation(status.Target.Name)
			if lastViolation == nil || lastViolation.EndTime != nil {
				// New violation
				violation := SLAViolation{
					ID:          fmt.Sprintf("violation-%s-%d", status.Target.Name, time.Now().Unix()),
					Target:      status.Target.Name,
					StartTime:   time.Now(),
					Severity:    s.determineSeverity(status.ErrorBudgetUsed),
					ImpactValue: status.CurrentValue,
					Metadata:    make(map[string]interface{}),
				}

				if s.violationStore != nil {
					if err := s.violationStore.Store(violation); err != nil {
						s.logger.WithError(err).Error("Failed to store violation")
					}
				}

				s.logger.WithFields(logrus.Fields{
					"target":    violation.Target,
					"violation": violation.ID,
					"severity":  violation.Severity,
				}).Warn("SLA violation detected")
			}
		}
	}
}

// getLastViolation gets the last violation for a target
func (s *SLAReporter) getLastViolation(target string) *SLAViolation {
	if s.violationStore == nil {
		return nil
	}

	violations, err := s.violationStore.List(target, time.Now().Add(-24*time.Hour))
	if err != nil || len(violations) == 0 {
		return nil
	}

	// Sort by start time and return the most recent
	sort.Slice(violations, func(i, j int) bool {
		return violations[i].StartTime.After(violations[j].StartTime)
	})

	return &violations[0]
}

// determineSeverity determines the severity of a violation
func (s *SLAReporter) determineSeverity(errorBudgetUsed float64) string {
	if errorBudgetUsed >= 2.0 {
		return "critical"
	} else if errorBudgetUsed >= 1.5 {
		return "high"
	} else if errorBudgetUsed >= 1.0 {
		return "medium"
	} else {
		return "low"
	}
}

// calculateOverallSLAScore calculates an overall SLA compliance score
func (s *SLAReporter) calculateOverallSLAScore(statuses []SLAStatus) float64 {
	if len(statuses) == 0 {
		return 0
	}

	var totalScore float64
	for _, status := range statuses {
		// Score based on compliance status
		var score float64
		switch status.ComplianceStatus {
		case "compliant":
			score = 100
		case "at_risk":
			score = 80
		case "violation":
			score = math.Max(0, 60-status.ErrorBudgetUsed*20)
		default:
			score = 50
		}
		totalScore += score
	}

	return totalScore / float64(len(statuses))
}

// generateSummary generates a summary of SLA performance for the period
func (s *SLAReporter) generateSummary(period Period) (ReportSummary, error) {
	// This would typically query historical data from Prometheus
	// For now, return a placeholder implementation
	return ReportSummary{
		TotalViolations:     0,
		CriticalViolations:  0,
		AverageAvailability: 99.97,
		AverageLatency:      1.8,
		AverageThroughput:   47.2,
		MTTR:                15.5,
		MTBF:                168.0,
	}, nil
}

// analyzeTrends analyzes trends in SLA performance
func (s *SLAReporter) analyzeTrends(period Period) (ReportTrends, error) {
	// This would typically analyze historical data
	// For now, return a placeholder implementation
	return ReportTrends{
		AvailabilityTrend: "stable",
		LatencyTrend:      "improving",
		ThroughputTrend:   "improving",
		ViolationTrend:    "stable",
		TrendConfidence:   0.85,
	}, nil
}

// generateRecommendations generates actionable recommendations
func (s *SLAReporter) generateRecommendations(statuses []SLAStatus, trends ReportTrends) []Recommendation {
	recommendations := make([]Recommendation, 0)

	// Analyze each SLA status for recommendations
	for _, status := range statuses {
		if status.ComplianceStatus == "violation" || status.ComplianceStatus == "at_risk" {
			recommendation := Recommendation{
				Type:     "reliability",
				Priority: "high",
				Title:    fmt.Sprintf("Address %s SLA Risk", status.Target.Name),
				Description: fmt.Sprintf("The %s SLA is currently %s with %.1f%% error budget used",
					status.Target.Name, status.ComplianceStatus, status.ErrorBudgetUsed*100),
				Action: "Investigate root cause and implement corrective measures",
				Impact: "Prevent SLA violation and maintain service quality",
			}
			recommendations = append(recommendations, recommendation)
		}
	}

	// Add trend-based recommendations
	if trends.LatencyTrend == "degrading" {
		recommendations = append(recommendations, Recommendation{
			Type:        "performance",
			Priority:    "medium",
			Title:       "Latency Performance Degrading",
			Description: "Response latency shows a degrading trend",
			Action:      "Review recent changes and optimize performance bottlenecks",
			Impact:      "Maintain responsive user experience",
		})
	}

	return recommendations
}

// calculateBusinessMetrics calculates business impact metrics
func (s *SLAReporter) calculateBusinessMetrics(period Period) (BusinessMetrics, error) {
	// This would typically integrate with business systems
	// For now, return a placeholder implementation
	return BusinessMetrics{
		TotalRevenue:         1000000.0,
		RevenueAtRisk:        5000.0,
		CustomerSatisfaction: 4.2,
		CompetitivePosition:  "strong",
	}, nil
}

// ServeHTTP provides an HTTP interface for the SLA reporter
func (s *SLAReporter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/sla/status":
		s.handleSLAStatus(w, r)
	case "/sla/report":
		s.handleSLAReport(w, r)
	case "/sla/violations":
		s.handleViolations(w, r)
	default:
		http.NotFound(w, r)
	}
}

// handleSLAStatus handles requests for current SLA status
func (s *SLAReporter) handleSLAStatus(w http.ResponseWriter, r *http.Request) {
	statuses := s.GetCurrentStatus()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statuses)
}

// handleSLAReport handles requests for SLA reports
func (s *SLAReporter) handleSLAReport(w http.ResponseWriter, r *http.Request) {
	reportType := r.URL.Query().Get("type")
	if reportType == "" {
		reportType = "daily"
	}

	// Parse period from query parameters
	period := Period{
		StartTime: time.Now().Add(-24 * time.Hour),
		EndTime:   time.Now(),
		Duration:  "24h",
	}

	if start := r.URL.Query().Get("start"); start != "" {
		if t, err := time.Parse(time.RFC3339, start); err == nil {
			period.StartTime = t
		}
	}

	if end := r.URL.Query().Get("end"); end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			period.EndTime = t
		}
	}

	report, err := s.GenerateReport(reportType, period)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	format := r.URL.Query().Get("format")
	switch format {
	case "pdf":
		if s.reportGenerator != nil {
			data, err := s.reportGenerator.GeneratePDF(*report)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/pdf")
			w.Write(data)
		} else {
			http.Error(w, "PDF generation not available", http.StatusNotImplemented)
		}
	case "html":
		if s.reportGenerator != nil {
			html, err := s.reportGenerator.GenerateHTML(*report)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(html))
		} else {
			http.Error(w, "HTML generation not available", http.StatusNotImplemented)
		}
	case "csv":
		if s.reportGenerator != nil {
			csv, err := s.reportGenerator.GenerateCSV(*report)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/csv")
			w.Write([]byte(csv))
		} else {
			http.Error(w, "CSV generation not available", http.StatusNotImplemented)
		}
	default:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(report)
	}
}

// handleViolations handles requests for SLA violations
func (s *SLAReporter) handleViolations(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	since := time.Now().Add(-24 * time.Hour)

	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = t
		}
	}

	if s.violationStore == nil {
		http.Error(w, "Violation store not configured", http.StatusNotImplemented)
		return
	}

	violations, err := s.violationStore.List(target, since)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(violations)
}
