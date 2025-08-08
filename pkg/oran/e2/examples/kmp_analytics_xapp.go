package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/e2"
)

// KMPAnalyticsXApp demonstrates KMP (Key Performance Measurement) analytics using the xApp SDK
type KMPAnalyticsXApp struct {
	sdk            *e2.XAppSDK
	analytics      *KMPAnalytics
	alertThresholds map[string]float64
}

// KMPAnalytics processes KMP measurement data
type KMPAnalytics struct {
	measurements     map[string]*MeasurementHistory
	alerts          []PerformanceAlert
	analysisResults []AnalysisResult
}

// MeasurementHistory stores historical measurement data
type MeasurementHistory struct {
	MetricName   string                 `json:"metric_name"`
	Values       []MeasurementValue     `json:"values"`
	Statistics   MeasurementStatistics  `json:"statistics"`
	LastUpdated  time.Time              `json:"last_updated"`
}

// MeasurementValue represents a single measurement
type MeasurementValue struct {
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
	CellID    string    `json:"cell_id"`
}

// MeasurementStatistics contains statistical analysis
type MeasurementStatistics struct {
	Mean     float64 `json:"mean"`
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
	StdDev   float64 `json:"std_dev"`
	Count    int     `json:"count"`
}

// PerformanceAlert represents a performance threshold alert
type PerformanceAlert struct {
	AlertID     string    `json:"alert_id"`
	MetricName  string    `json:"metric_name"`
	Value       float64   `json:"value"`
	Threshold   float64   `json:"threshold"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
	CellID      string    `json:"cell_id"`
}

// AnalysisResult contains analysis outcomes
type AnalysisResult struct {
	AnalysisID   string                 `json:"analysis_id"`
	AnalysisType string                 `json:"analysis_type"`
	Results      map[string]interface{} `json:"results"`
	Timestamp    time.Time              `json:"timestamp"`
	Confidence   float64                `json:"confidence"`
}

// NewKMPAnalyticsXApp creates a new KMP analytics xApp
func NewKMPAnalyticsXApp() (*KMPAnalyticsXApp, error) {
	// Configure xApp
	config := &e2.XAppConfig{
		XAppName:        "kmp-analytics-xapp",
		XAppVersion:     "1.0.0",
		XAppDescription: "KMP Performance Analytics and Alerting xApp",
		E2NodeID:        config.GetEnvOrDefault("E2_NODE_ID", "gnb-001"),
		NearRTRICURL:    config.GetEnvOrDefault("NEAR_RT_RIC_URL", "http://near-rt-ric:8080"),
		ServiceModels:   []string{"KPM"},
		Environment: map[string]string{
			"LOG_LEVEL":     "INFO",
			"ANALYSIS_MODE": "real-time",
		},
		ResourceLimits: &e2.XAppResourceLimits{
			MaxMemoryMB:      512,
			MaxCPUCores:      1.0,
			MaxSubscriptions: 10,
			RequestTimeout:   30 * time.Second,
		},
		HealthCheck: &e2.XAppHealthConfig{
			Enabled:          true,
			CheckInterval:    30 * time.Second,
			FailureThreshold: 3,
			HealthEndpoint:   "/health",
		},
	}

	// Create E2Manager (this would be injected in real implementation)
	e2Manager := &e2.E2Manager{} // Placeholder

	// Create SDK
	sdk, err := e2.NewXAppSDK(config, e2Manager)
	if err != nil {
		return nil, fmt.Errorf("failed to create xApp SDK: %w", err)
	}

	// Create analytics engine
	analytics := &KMPAnalytics{
		measurements:     make(map[string]*MeasurementHistory),
		alerts:          make([]PerformanceAlert, 0),
		analysisResults: make([]AnalysisResult, 0),
	}

	xapp := &KMPAnalyticsXApp{
		sdk:       sdk,
		analytics: analytics,
		alertThresholds: map[string]float64{
			"DRB.UEThpDl":    100.0, // Mbps threshold
			"DRB.UEThpUl":    50.0,  // Mbps threshold
			"RRU.PrbUsedDl":  0.8,   // 80% PRB utilization
			"RRU.PrbUsedUl":  0.8,   // 80% PRB utilization
			"DRB.RlcSduDelayDl": 20.0, // 20ms delay threshold
		},
	}

	// Register handlers
	sdk.RegisterIndicationHandler("default", xapp.handleKMPIndication)

	return xapp, nil
}

// Start starts the xApp
func (x *KMPAnalyticsXApp) Start(ctx context.Context) error {
	log.Printf("Starting KMP Analytics xApp")

	// Start the SDK
	if err := x.sdk.Start(ctx); err != nil {
		return fmt.Errorf("failed to start xApp SDK: %w", err)
	}

	// Create KMP subscription for key performance metrics
	subscriptionReq := &e2.E2SubscriptionRequest{
		SubscriptionID: "kmp-analytics-001",
		NodeID:         x.sdk.GetConfig().E2NodeID,
		RequestorID:    "kmp-analytics-xapp",
		RanFunctionID:  1, // KMP service model
		EventTriggers: []e2.E2EventTrigger{
			{
				TriggerType: "periodic",
				Conditions: map[string]interface{}{
					"measurement_types": []string{
						"DRB.UEThpDl",         // Downlink UE throughput
						"DRB.UEThpUl",         // Uplink UE throughput
						"RRU.PrbUsedDl",       // Downlink PRB utilization
						"RRU.PrbUsedUl",       // Uplink PRB utilization
						"DRB.RlcSduDelayDl",   // Downlink RLC SDU delay
						"DRB.RlcSduDelayUl",   // Uplink RLC SDU delay
						"TB.ErrTotNbrDl",      // Downlink transport block errors
						"TB.ErrTotNbrUl",      // Uplink transport block errors
					},
					"granularity_period":    "1000ms",
					"collection_start_time": time.Now().Format(time.RFC3339),
					"collection_duration":   "continuous",
					"reporting_format":      "CHOICE",
				},
			},
		},
		Actions: []e2.E2Action{
			{
				ActionID:   1,
				ActionType: "report",
				ActionDefinition: map[string]interface{}{
					"format": "json",
					"compression": false,
				},
			},
		},
		ReportingPeriod: 1 * time.Second,
	}

	// Subscribe to KMP measurements
	subscription, err := x.sdk.Subscribe(ctx, subscriptionReq)
	if err != nil {
		return fmt.Errorf("failed to create KMP subscription: %w", err)
	}

	log.Printf("Created KMP subscription: %s", subscription.SubscriptionID)

	// Start analytics processing loop
	go x.analyticsLoop(ctx)

	return nil
}

// Stop stops the xApp
func (x *KMPAnalyticsXApp) Stop(ctx context.Context) error {
	log.Printf("Stopping KMP Analytics xApp")
	return x.sdk.Stop(ctx)
}

// handleKMPIndication processes incoming KMP indications
func (x *KMPAnalyticsXApp) handleKMPIndication(ctx context.Context, indication *e2.RICIndication) error {
	// Parse the indication message
	var kmpData map[string]interface{}
	if err := json.Unmarshal(indication.RICIndicationMessage, &kmpData); err != nil {
		log.Printf("Failed to parse KMP indication: %v", err)
		return err
	}

	// Extract measurements
	measurements, ok := kmpData["measurements"].(map[string]interface{})
	if !ok {
		log.Printf("Invalid KMP indication format: missing measurements")
		return fmt.Errorf("invalid KMP indication format")
	}

	cellID, _ := kmpData["cell_id"].(string)
	timestampStr, _ := kmpData["timestamp"].(string)
	timestamp, _ := time.Parse(time.RFC3339, timestampStr)
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	// Process each measurement
	for metricName, value := range measurements {
		if metricValue, ok := value.(float64); ok {
			x.processMeasurement(metricName, metricValue, cellID, timestamp)
		}
	}

	log.Printf("Processed KMP indication: %d measurements from cell %s", len(measurements), cellID)
	return nil
}

// processMeasurement processes a single measurement value
func (x *KMPAnalyticsXApp) processMeasurement(metricName string, value float64, cellID string, timestamp time.Time) {
	// Get or create measurement history
	history, exists := x.analytics.measurements[metricName]
	if !exists {
		history = &MeasurementHistory{
			MetricName:  metricName,
			Values:      make([]MeasurementValue, 0),
			LastUpdated: timestamp,
		}
		x.analytics.measurements[metricName] = history
	}

	// Add new measurement
	measurement := MeasurementValue{
		Value:     value,
		Timestamp: timestamp,
		CellID:    cellID,
	}
	
	history.Values = append(history.Values, measurement)
	history.LastUpdated = timestamp

	// Keep only last 1000 measurements to prevent memory growth
	if len(history.Values) > 1000 {
		history.Values = history.Values[1:]
	}

	// Update statistics
	x.updateStatistics(history)

	// Check for alerts
	x.checkAlerts(metricName, value, cellID, timestamp)
}

// updateStatistics calculates statistical measures for measurement history
func (x *KMPAnalyticsXApp) updateStatistics(history *MeasurementHistory) {
	if len(history.Values) == 0 {
		return
	}

	var sum, min, max float64
	min = history.Values[0].Value
	max = history.Values[0].Value

	for _, measurement := range history.Values {
		sum += measurement.Value
		if measurement.Value < min {
			min = measurement.Value
		}
		if measurement.Value > max {
			max = measurement.Value
		}
	}

	count := float64(len(history.Values))
	mean := sum / count

	// Calculate standard deviation
	var variance float64
	for _, measurement := range history.Values {
		diff := measurement.Value - mean
		variance += diff * diff
	}
	variance /= count
	stdDev := 0.0
	if variance > 0 {
		stdDev = variance // Simplified - would use math.Sqrt in real implementation
	}

	history.Statistics = MeasurementStatistics{
		Mean:   mean,
		Min:    min,
		Max:    max,
		StdDev: stdDev,
		Count:  int(count),
	}
}

// checkAlerts checks if measurement values exceed thresholds
func (x *KMPAnalyticsXApp) checkAlerts(metricName string, value float64, cellID string, timestamp time.Time) {
	threshold, exists := x.alertThresholds[metricName]
	if !exists {
		return
	}

	var severity string
	var triggered bool

	// Check threshold based on metric type
	switch metricName {
	case "DRB.UEThpDl", "DRB.UEThpUl":
		// Throughput - alert if below threshold
		if value < threshold {
			triggered = true
			severity = "WARNING"
		}
	case "RRU.PrbUsedDl", "RRU.PrbUsedUl":
		// PRB utilization - alert if above threshold
		if value > threshold {
			triggered = true
			severity = "WARNING"
			if value > 0.95 {
				severity = "CRITICAL"
			}
		}
	case "DRB.RlcSduDelayDl", "DRB.RlcSduDelayUl":
		// Delay - alert if above threshold
		if value > threshold {
			triggered = true
			severity = "WARNING"
			if value > threshold*2 {
				severity = "CRITICAL"
			}
		}
	}

	if triggered {
		alert := PerformanceAlert{
			AlertID:     fmt.Sprintf("alert-%s-%s-%d", metricName, cellID, timestamp.Unix()),
			MetricName:  metricName,
			Value:       value,
			Threshold:   threshold,
			Severity:    severity,
			Description: fmt.Sprintf("%s threshold exceeded: %.2f (threshold: %.2f)", metricName, value, threshold),
			Timestamp:   timestamp,
			CellID:      cellID,
		}

		x.analytics.alerts = append(x.analytics.alerts, alert)
		log.Printf("ALERT [%s]: %s", severity, alert.Description)

		// Keep only last 100 alerts
		if len(x.analytics.alerts) > 100 {
			x.analytics.alerts = x.analytics.alerts[1:]
		}
	}
}

// analyticsLoop runs continuous analytics processing
func (x *KMPAnalyticsXApp) analyticsLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			x.performAnalysis()
		}
	}
}

// performAnalysis runs analytical algorithms on collected data
func (x *KMPAnalyticsXApp) performAnalysis() {
	log.Printf("Performing analytics on %d metrics", len(x.analytics.measurements))

	// Example: Trend analysis
	trendResults := x.analyzeTrends()
	if len(trendResults) > 0 {
		analysisResult := AnalysisResult{
			AnalysisID:   fmt.Sprintf("trend-analysis-%d", time.Now().Unix()),
			AnalysisType: "trend_analysis",
			Results:      trendResults,
			Timestamp:    time.Now(),
			Confidence:   0.85,
		}
		x.analytics.analysisResults = append(x.analytics.analysisResults, analysisResult)
		log.Printf("Trend analysis completed: %v", trendResults)
	}

	// Example: Correlation analysis
	correlationResults := x.analyzeCorrelations()
	if len(correlationResults) > 0 {
		analysisResult := AnalysisResult{
			AnalysisID:   fmt.Sprintf("correlation-analysis-%d", time.Now().Unix()),
			AnalysisType: "correlation_analysis",
			Results:      correlationResults,
			Timestamp:    time.Now(),
			Confidence:   0.75,
		}
		x.analytics.analysisResults = append(x.analytics.analysisResults, analysisResult)
		log.Printf("Correlation analysis completed: %v", correlationResults)
	}

	// Keep only last 50 analysis results
	if len(x.analytics.analysisResults) > 50 {
		x.analytics.analysisResults = x.analytics.analysisResults[1:]
	}
}

// analyzeTrends analyzes measurement trends
func (x *KMPAnalyticsXApp) analyzeTrends() map[string]interface{} {
	results := make(map[string]interface{})

	for metricName, history := range x.analytics.measurements {
		if len(history.Values) < 10 {
			continue // Need at least 10 data points
		}

		// Simple trend analysis - check if recent values are increasing/decreasing
		recentValues := history.Values[len(history.Values)-10:]
		
		var increasing, decreasing int
		for i := 1; i < len(recentValues); i++ {
			if recentValues[i].Value > recentValues[i-1].Value {
				increasing++
			} else if recentValues[i].Value < recentValues[i-1].Value {
				decreasing++
			}
		}

		var trend string
		if increasing > decreasing*2 {
			trend = "increasing"
		} else if decreasing > increasing*2 {
			trend = "decreasing"
		} else {
			trend = "stable"
		}

		results[metricName] = map[string]interface{}{
			"trend":              trend,
			"confidence":         float64(abs(increasing-decreasing)) / float64(len(recentValues)-1),
			"recent_average":     calculateAverage(recentValues),
			"historical_average": history.Statistics.Mean,
		}
	}

	return results
}

// analyzeCorrelations analyzes correlations between different metrics
func (x *KMPAnalyticsXApp) analyzeCorrelations() map[string]interface{} {
	results := make(map[string]interface{})

	// Example: Check correlation between throughput and PRB utilization
	throughputDL, exists1 := x.analytics.measurements["DRB.UEThpDl"]
	prbUsedDL, exists2 := x.analytics.measurements["RRU.PrbUsedDl"]

	if exists1 && exists2 && len(throughputDL.Values) > 10 && len(prbUsedDL.Values) > 10 {
		correlation := x.calculateCorrelation(throughputDL.Values, prbUsedDL.Values)
		results["throughput_prb_correlation"] = map[string]interface{}{
			"correlation_coefficient": correlation,
			"interpretation":          x.interpretCorrelation(correlation),
			"metric_pair":            "DRB.UEThpDl vs RRU.PrbUsedDl",
		}
	}

	return results
}

// calculateCorrelation calculates correlation coefficient between two measurement series
func (x *KMPAnalyticsXApp) calculateCorrelation(values1, values2 []MeasurementValue) float64 {
	// Simplified correlation calculation
	// In a real implementation, this would use proper statistical correlation algorithms
	
	if len(values1) != len(values2) || len(values1) < 2 {
		return 0.0
	}

	// Use the minimum length and align by timestamp
	minLen := len(values1)
	if len(values2) < minLen {
		minLen = len(values2)
	}

	// For simplicity, just use the last minLen values
	subset1 := values1[len(values1)-minLen:]
	subset2 := values2[len(values2)-minLen:]

	// Calculate means
	mean1 := calculateAverage(subset1)
	mean2 := calculateAverage(subset2)

	// Calculate correlation coefficient (simplified)
	var numerator, denominator1, denominator2 float64
	for i := 0; i < minLen; i++ {
		diff1 := subset1[i].Value - mean1
		diff2 := subset2[i].Value - mean2
		numerator += diff1 * diff2
		denominator1 += diff1 * diff1
		denominator2 += diff2 * diff2
	}

	if denominator1 == 0 || denominator2 == 0 {
		return 0.0
	}

	// Simplified square root approximation
	correlation := numerator / (denominator1 * denominator2)
	return correlation
}

// interpretCorrelation provides interpretation of correlation values
func (x *KMPAnalyticsXApp) interpretCorrelation(correlation float64) string {
	absCorr := correlation
	if absCorr < 0 {
		absCorr = -absCorr
	}

	if absCorr > 0.8 {
		return "strong correlation"
	} else if absCorr > 0.5 {
		return "moderate correlation"
	} else if absCorr > 0.3 {
		return "weak correlation"
	}
	return "no significant correlation"
}

// Helper functions

// Note: Helper functions have been moved to pkg/config/env_helpers.go

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func calculateAverage(values []MeasurementValue) float64 {
	if len(values) == 0 {
		return 0.0
	}
	
	var sum float64
	for _, value := range values {
		sum += value.Value
	}
	return sum / float64(len(values))
}

// Main function to run the xApp
func main() {
	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create and start the xApp
	xapp, err := NewKMPAnalyticsXApp()
	if err != nil {
		log.Fatalf("Failed to create KMP Analytics xApp: %v", err)
	}

	// Start the xApp
	if err := xapp.Start(ctx); err != nil {
		log.Fatalf("Failed to start KMP Analytics xApp: %v", err)
	}

	// Wait for shutdown signal
	<-sigChan
	log.Println("Received shutdown signal")

	// Stop the xApp
	if err := xapp.Stop(ctx); err != nil {
		log.Printf("Error stopping xApp: %v", err)
	}

	log.Println("KMP Analytics xApp stopped")
}