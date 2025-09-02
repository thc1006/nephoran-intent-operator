/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package monitoring

import (
	
	"encoding/json"
"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// ErrorTrackingConfig defines configuration for error tracking
type ErrorTrackingConfig struct {
	// EnablePrometheus enables Prometheus metrics collection
	EnablePrometheus bool

	// PrometheusEnabled enables Prometheus metrics
	PrometheusEnabled bool

	// PrometheusNamespace sets the Prometheus namespace
	PrometheusNamespace string

	// ErrorBufferSize sets the error buffer size
	ErrorBufferSize int

	// EnableOpenTelemetry enables OpenTelemetry metrics collection
	EnableOpenTelemetry bool

	// OpenTelemetryEnabled enables OpenTelemetry
	OpenTelemetryEnabled bool

	// ServiceName sets the service name for tracing
	ServiceName string

	// ProcessingWorkers sets number of processing workers
	ProcessingWorkers int

	// AlertingEnabled enables alerting on error thresholds
	AlertingEnabled bool

	// NotificationChannels defines notification channels
	NotificationChannels []NotificationChannel

	// DashboardEnabled enables dashboard metrics export
	DashboardEnabled bool

	// DashboardPort sets the dashboard port
	DashboardPort int

	// AnalyticsEnabled enables analytics
	AnalyticsEnabled bool

	// PredictionEnabled enables prediction
	PredictionEnabled bool

	// AnomalyDetectionEnabled enables anomaly detection
	AnomalyDetectionEnabled bool

	// ReportsEnabled enables error reports generation
	ReportsEnabled bool

	// Enabled enables the error tracking system
	Enabled bool

	// MetricsPort sets the metrics port
	MetricsPort int

	// AlertRules defines alert rules
	AlertRules []AlertRule

	// MetricsRetentionDays sets metrics retention
	MetricsRetentionDays int

	// TracesRetentionDays sets traces retention
	TracesRetentionDays int

	// ReportsRetentionDays sets reports retention
	ReportsRetentionDays int

	// ErrorRetentionDuration defines how long to retain error records
	ErrorRetentionDuration time.Duration

	// AlertThresholds defines error rate thresholds for alerting
	AlertThresholds map[string]float64
}

// ErrorSummary provides a summary of tracked errors
type ErrorSummary struct {
	// TotalErrors is the total number of errors tracked
	TotalErrors int64 `json:"totalErrors"`

	// ErrorRate is the current error rate per minute
	ErrorRate float64 `json:"errorRate"`

	// TopErrors contains the most frequent error patterns
	TopErrors []ErrorPattern `json:"topErrors"`

	// LastError contains details of the most recent error
	LastError *ErrorRecord `json:"lastError,omitempty"`

	// TimeWindow is the time window for the summary
	TimeWindow time.Duration `json:"timeWindow"`
}

// ErrorPattern represents a pattern of errors
type ErrorPattern struct {
	// Pattern is the error pattern string
	Pattern string `json:"pattern"`

	// Count is the number of occurrences
	Count int64 `json:"count"`

	// FirstSeen is when this pattern was first observed
	FirstSeen time.Time `json:"firstSeen"`

	// LastSeen is when this pattern was last observed
	LastSeen time.Time `json:"lastSeen"`

	// Components affected by this error pattern
	Components []string `json:"components"`
}

// ErrorRecord represents a single error record
type ErrorRecord struct {
	// ID is unique identifier for the error
	ID string `json:"id"`

	// Timestamp when the error occurred
	Timestamp time.Time `json:"timestamp"`

	// Component that generated the error
	Component string `json:"component"`

	// ErrorType categorizes the error
	ErrorType string `json:"errorType"`

	// Message is the error message
	Message string `json:"message"`

	// Severity indicates the error severity
	Severity string `json:"severity"`

	// Context provides additional error context
	Context json.RawMessage `json:"context,omitempty"`

	// StackTrace contains the stack trace if available
	StackTrace string `json:"stackTrace,omitempty"`

	// Tags for categorization and filtering
	Tags []string `json:"tags,omitempty"`

	// CorrelationID for tracing related errors
	CorrelationID string `json:"correlationId,omitempty"`

	// IntentID for linking errors to specific intents
	IntentID string `json:"intentId,omitempty"`
}

// ErrorTracker tracks and analyzes errors for monitoring and alerting
type ErrorTracker struct {
	config   *ErrorTrackingConfig
	logger   logr.Logger
	mu       sync.RWMutex
	errors   []ErrorRecord
	patterns map[string]*ErrorPattern

	// Prometheus metrics
	errorCounter   prometheus.Counter
	errorHistogram prometheus.Histogram
	patternGauge   *prometheus.GaugeVec

	// OpenTelemetry metrics
	otelErrorCounter metric.Int64Counter
	otelErrorHisto   metric.Int64Histogram

	// Internal state
	startTime   time.Time
	lastCleanup time.Time
}

// NewErrorTracker creates a new ErrorTracker instance
func NewErrorTracker(config *ErrorTrackingConfig, logger logr.Logger) (*ErrorTracker, error) {
	if config == nil {
		config = &ErrorTrackingConfig{
			EnablePrometheus:       true,
			EnableOpenTelemetry:    true,
			AlertingEnabled:        true,
			DashboardEnabled:       true,
			ReportsEnabled:         true,
			ErrorRetentionDuration: 24 * time.Hour,
			AlertThresholds: map[string]float64{
				"error_rate": 10.0, // errors per minute
			},
		}
	}

	tracker := &ErrorTracker{
		config:    config,
		logger:    logger,
		errors:    make([]ErrorRecord, 0),
		patterns:  make(map[string]*ErrorPattern),
		startTime: time.Now(),
	}

	// Initialize Prometheus metrics if enabled
	if config.EnablePrometheus {
		tracker.errorCounter = promauto.NewCounter(prometheus.CounterOpts{
			Name: "nephoran_errors_total",
			Help: "Total number of errors tracked",
		})

		tracker.errorHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
			Name: "nephoran_error_processing_duration_seconds",
			Help: "Time spent processing errors",
		})

		tracker.patternGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "nephoran_error_patterns",
			Help: "Number of error patterns by type",
		}, []string{"pattern", "component"})
	}

	// Initialize OpenTelemetry metrics if enabled
	if config.EnableOpenTelemetry {
		meter := otel.Meter("nephoran.monitoring.error_tracker")

		var err error
		tracker.otelErrorCounter, err = meter.Int64Counter(
			"nephoran.errors.count",
			metric.WithDescription("Total number of errors tracked"),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTEL error counter: %w", err)
		}

		tracker.otelErrorHisto, err = meter.Int64Histogram(
			"nephoran.error.processing.duration",
			metric.WithDescription("Error processing duration in milliseconds"),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTEL error histogram: %w", err)
		}
	}

	logger.Info("Error tracker initialized", "config", config)
	return tracker, nil
}

// TrackError records a new error for tracking and analysis
func (et *ErrorTracker) TrackError(ctx context.Context, err error, component, errorType string, context map[string]interface{}) {
	if err == nil {
		return
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		if et.config.EnableOpenTelemetry && et.otelErrorHisto != nil {
			et.otelErrorHisto.Record(ctx, duration.Milliseconds(),
				metric.WithAttributes(
					attribute.String("component", component),
					attribute.String("error_type", errorType),
				))
		}
	}()

	// Convert context map to JSON
	contextJSON := json.RawMessage(`{}`)
	if context != nil && len(context) > 0 {
		if contextBytes, err := json.Marshal(context); err == nil {
			contextJSON = contextBytes
		}
	}

	record := ErrorRecord{
		ID:        fmt.Sprintf("%s-%d", component, time.Now().UnixNano()),
		Timestamp: time.Now(),
		Component: component,
		ErrorType: errorType,
		Message:   err.Error(),
		Severity:  et.determineSeverity(err, errorType),
		Context:   contextJSON,
		Tags:      et.generateTags(component, errorType, err),
	}

	et.mu.Lock()
	defer et.mu.Unlock()

	// Add to error list
	et.errors = append(et.errors, record)

	// Update error patterns
	pattern := et.extractPattern(err.Error())
	if existing, exists := et.patterns[pattern]; exists {
		existing.Count++
		existing.LastSeen = time.Now()
		existing.Components = et.addUniqueComponent(existing.Components, component)
	} else {
		et.patterns[pattern] = &ErrorPattern{
			Pattern:    pattern,
			Count:      1,
			FirstSeen:  time.Now(),
			LastSeen:   time.Now(),
			Components: []string{component},
		}
	}

	// Update Prometheus metrics
	if et.config.EnablePrometheus {
		et.errorCounter.Inc()
		if et.patternGauge != nil {
			et.patternGauge.WithLabelValues(pattern, component).Inc()
		}
	}

	// Update OpenTelemetry metrics
	if et.config.EnableOpenTelemetry && et.otelErrorCounter != nil {
		et.otelErrorCounter.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("component", component),
				attribute.String("error_type", errorType),
				attribute.String("severity", record.Severity),
			))
	}

	// Cleanup old errors if needed
	et.cleanupOldErrors()

	et.logger.Error(err, "Error tracked",
		"component", component,
		"errorType", errorType,
		"pattern", pattern,
		"errorId", record.ID)
}

// GetErrorSummary returns a summary of tracked errors
func (et *ErrorTracker) GetErrorSummary() *ErrorSummary {
	et.mu.RLock()
	defer et.mu.RUnlock()

	if len(et.errors) == 0 {
		return &ErrorSummary{
			TotalErrors: 0,
			ErrorRate:   0,
			TopErrors:   make([]ErrorPattern, 0),
			TimeWindow:  time.Since(et.startTime),
		}
	}

	// Calculate error rate (errors per minute)
	timeWindow := time.Since(et.startTime)
	errorRate := float64(len(et.errors)) / timeWindow.Minutes()

	// Get top error patterns
	topErrors := et.getTopErrorPatterns(5)

	// Get last error
	var lastError *ErrorRecord
	if len(et.errors) > 0 {
		lastError = &et.errors[len(et.errors)-1]
	}

	return &ErrorSummary{
		TotalErrors: int64(len(et.errors)),
		ErrorRate:   errorRate,
		TopErrors:   topErrors,
		LastError:   lastError,
		TimeWindow:  timeWindow,
	}
}

// GetErrorsByPattern returns errors matching a specific pattern
func (et *ErrorTracker) GetErrorsByPattern(pattern string) []ErrorRecord {
	et.mu.RLock()
	defer et.mu.RUnlock()

	var matchingErrors []ErrorRecord
	for _, err := range et.errors {
		if et.extractPattern(err.Message) == pattern {
			matchingErrors = append(matchingErrors, err)
		}
	}

	return matchingErrors
}

// GetErrorsByComponent returns errors from a specific component
func (et *ErrorTracker) GetErrorsByComponent(component string) []ErrorRecord {
	et.mu.RLock()
	defer et.mu.RUnlock()

	var componentErrors []ErrorRecord
	for _, err := range et.errors {
		if err.Component == component {
			componentErrors = append(componentErrors, err)
		}
	}

	return componentErrors
}

// GetErrorPatterns returns all tracked error patterns
func (et *ErrorTracker) GetErrorPatterns() []ErrorPattern {
	et.mu.RLock()
	defer et.mu.RUnlock()

	patterns := make([]ErrorPattern, 0, len(et.patterns))
	for _, pattern := range et.patterns {
		patterns = append(patterns, *pattern)
	}

	return patterns
}

// Reset clears all tracked errors and patterns
func (et *ErrorTracker) Reset() {
	et.mu.Lock()
	defer et.mu.Unlock()

	et.errors = make([]ErrorRecord, 0)
	et.patterns = make(map[string]*ErrorPattern)
	et.startTime = time.Now()
	et.lastCleanup = time.Now()

	et.logger.Info("Error tracker reset")
}

// Health returns the health status of the error tracker
func (et *ErrorTracker) Health() error {
	et.mu.RLock()
	defer et.mu.RUnlock()

	// Check if we're tracking too many errors (potential memory issue)
	if len(et.errors) > 10000 {
		return fmt.Errorf("too many errors tracked: %d", len(et.errors))
	}

	// Check error rate thresholds
	if et.config.AlertThresholds != nil {
		if threshold, exists := et.config.AlertThresholds["error_rate"]; exists {
			timeWindow := time.Since(et.startTime)
			currentRate := float64(len(et.errors)) / timeWindow.Minutes()
			if currentRate > threshold {
				return fmt.Errorf("error rate too high: %.2f errors/min (threshold: %.2f)", currentRate, threshold)
			}
		}
	}

	return nil
}

// extractPattern extracts a pattern from an error message for grouping
func (et *ErrorTracker) extractPattern(message string) string {
	// Simple pattern extraction - in production, this could be more sophisticated
	// For now, just use the first 100 characters as a pattern
	if len(message) > 100 {
		return message[:100] + "..."
	}
	return message
}

// determineSeverity determines the severity of an error
func (et *ErrorTracker) determineSeverity(err error, errorType string) string {
	message := err.Error()

	// High severity patterns
	if contains(message, []string{"panic", "fatal", "critical", "crash", "deadlock"}) {
		return "critical"
	}

	// Medium severity patterns
	if contains(message, []string{"timeout", "connection", "failed", "error"}) {
		return "high"
	}

	// Low severity patterns
	if contains(message, []string{"warning", "deprecated", "retry"}) {
		return "medium"
	}

	return "low"
}

// generateTags generates tags for error categorization
func (et *ErrorTracker) generateTags(component, errorType string, err error) []string {
	tags := []string{component, errorType}

	message := err.Error()
	if contains(message, []string{"network", "connection", "socket"}) {
		tags = append(tags, "network")
	}
	if contains(message, []string{"timeout", "deadline"}) {
		tags = append(tags, "timeout")
	}
	if contains(message, []string{"memory", "oom"}) {
		tags = append(tags, "memory")
	}
	if contains(message, []string{"cpu", "resource"}) {
		tags = append(tags, "resource")
	}

	return tags
}

// getTopErrorPatterns returns the top N error patterns by frequency
func (et *ErrorTracker) getTopErrorPatterns(limit int) []ErrorPattern {
	patterns := make([]ErrorPattern, 0, len(et.patterns))
	for _, pattern := range et.patterns {
		patterns = append(patterns, *pattern)
	}

	// Sort by count (descending)
	for i := 0; i < len(patterns)-1; i++ {
		for j := 0; j < len(patterns)-i-1; j++ {
			if patterns[j].Count < patterns[j+1].Count {
				patterns[j], patterns[j+1] = patterns[j+1], patterns[j]
			}
		}
	}

	if len(patterns) > limit {
		patterns = patterns[:limit]
	}

	return patterns
}

// cleanupOldErrors removes errors older than the retention duration
func (et *ErrorTracker) cleanupOldErrors() {
	now := time.Now()
	if now.Sub(et.lastCleanup) < time.Hour {
		return // Only cleanup once per hour
	}

	cutoff := now.Add(-et.config.ErrorRetentionDuration)
	var keptErrors []ErrorRecord

	for _, err := range et.errors {
		if err.Timestamp.After(cutoff) {
			keptErrors = append(keptErrors, err)
		}
	}

	removedCount := len(et.errors) - len(keptErrors)
	et.errors = keptErrors
	et.lastCleanup = now

	if removedCount > 0 {
		et.logger.Info("Cleaned up old errors", "removed", removedCount, "retained", len(keptErrors))
	}
}

// addUniqueComponent adds a component to the list if not already present
func (et *ErrorTracker) addUniqueComponent(components []string, component string) []string {
	for _, existing := range components {
		if existing == component {
			return components
		}
	}
	return append(components, component)
}

// GetErrorsByCorrelation returns errors filtered by correlation ID
func (et *ErrorTracker) GetErrorsByCorrelation(correlationID string) []ErrorRecord {
	et.mu.RLock()
	defer et.mu.RUnlock()

	var matched []ErrorRecord
	for _, record := range et.errors {
		if record.CorrelationID == correlationID {
			matched = append(matched, record)
		}
	}

	return matched
}

// GetMetrics returns current error tracking metrics
func (et *ErrorTracker) GetMetrics() map[string]interface{} {
	et.mu.RLock()
	defer et.mu.RUnlock()

	metrics := make(map[string]interface{})
	metrics["total_errors"] = len(et.errors)
	metrics["total_patterns"] = len(et.patterns)

	// Count errors by severity
	severityCounts := make(map[string]int)
	for _, record := range et.errors {
		severityCounts[record.Severity]++
	}
	metrics["errors_by_severity"] = severityCounts

	// Count errors by component
	componentCounts := make(map[string]int)
	for _, record := range et.errors {
		componentCounts[record.Component]++
	}
	metrics["errors_by_component"] = componentCounts

	metrics["uptime_seconds"] = time.Since(et.startTime).Seconds()

	return metrics
}

// contains checks if a string contains any of the given substrings
func contains(s string, substrings []string) bool {
	for _, substring := range substrings {
		if len(s) >= len(substring) {
			for i := 0; i <= len(s)-len(substring); i++ {
				if s[i:i+len(substring)] == substring {
					return true
				}
			}
		}
	}
	return false
}

