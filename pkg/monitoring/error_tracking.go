// Copyright 2024 Nephio Contributors
// Licensed under the Apache License, Version 2.0

package monitoring

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ErrorTracker tracks and analyzes errors across the system
type ErrorTracker struct {
	mu          sync.RWMutex
	errors      map[string]*ErrorSummary
	anomalyDetector *AnomalyDetector
	logger      *zap.Logger
	maxErrors   int
}

// ErrorSummary contains aggregated error information
type ErrorSummary struct {
	Type        string    `json:"type"`
	Count       int64     `json:"count"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	Severity    string    `json:"severity"`
	Component   string    `json:"component"`
	Description string    `json:"description"`
	Samples     []string  `json:"samples,omitempty"`
}

// ErrorTrackingConfig configures the error tracker
type ErrorTrackingConfig struct {
	MaxErrors           int           `json:"max_errors" yaml:"maxErrors"`
	AnomalyThreshold    float64       `json:"anomaly_threshold" yaml:"anomalyThreshold"`
	TimeWindow          time.Duration `json:"time_window" yaml:"timeWindow"`
	CleanupInterval     time.Duration `json:"cleanup_interval" yaml:"cleanupInterval"`
	EnableAnomalyDetect bool          `json:"enable_anomaly_detect" yaml:"enableAnomalyDetect"`
}

// NewErrorTracker creates a new error tracker
func NewErrorTracker(config ErrorTrackingConfig, logger *zap.Logger) *ErrorTracker {
	if logger == nil {
		logger = zap.NewNop()
	}

	tracker := &ErrorTracker{
		errors:    make(map[string]*ErrorSummary),
		logger:    logger,
		maxErrors: config.MaxErrors,
	}

	if config.EnableAnomalyDetect {
		tracker.anomalyDetector = NewAnomalyDetector(config.AnomalyThreshold, config.TimeWindow)
	}

	// Start cleanup routine
	if config.CleanupInterval > 0 {
		go tracker.cleanupRoutine(config.CleanupInterval)
	}

	return tracker
}

// TrackError records an error occurrence
func (et *ErrorTracker) TrackError(errorType, component, description string, severity string) {
	et.mu.Lock()
	defer et.mu.Unlock()

	key := fmt.Sprintf("%s:%s", component, errorType)
	
	if summary, exists := et.errors[key]; exists {
		summary.Count++
		summary.LastSeen = time.Now()
		if len(summary.Samples) < 10 {
			summary.Samples = append(summary.Samples, description)
		}
	} else {
		// Check if we've reached max errors
		if len(et.errors) >= et.maxErrors {
			et.logger.Warn("Max errors reached, dropping oldest",
				zap.Int("max_errors", et.maxErrors))
			et.removeOldestError()
		}

		et.errors[key] = &ErrorSummary{
			Type:        errorType,
			Count:       1,
			FirstSeen:   time.Now(),
			LastSeen:    time.Now(),
			Severity:    severity,
			Component:   component,
			Description: description,
			Samples:     []string{description},
		}
	}

	// Check for anomalies
	if et.anomalyDetector != nil {
		isAnomaly := et.anomalyDetector.DetectAnomaly(key, 1.0)
		if isAnomaly {
			et.logger.Warn("Error anomaly detected",
				zap.String("error_key", key),
				zap.String("component", component),
				zap.String("type", errorType))
		}
	}
}

// GetErrorSummary returns aggregated error information
func (et *ErrorTracker) GetErrorSummary() map[string]*ErrorSummary {
	et.mu.RLock()
	defer et.mu.RUnlock()

	result := make(map[string]*ErrorSummary, len(et.errors))
	for key, summary := range et.errors {
		// Deep copy
		result[key] = &ErrorSummary{
			Type:        summary.Type,
			Count:       summary.Count,
			FirstSeen:   summary.FirstSeen,
			LastSeen:    summary.LastSeen,
			Severity:    summary.Severity,
			Component:   summary.Component,
			Description: summary.Description,
			Samples:     append([]string(nil), summary.Samples...),
		}
	}

	return result
}

// ClearErrors clears all tracked errors
func (et *ErrorTracker) ClearErrors() {
	et.mu.Lock()
	defer et.mu.Unlock()

	et.errors = make(map[string]*ErrorSummary)
	if et.anomalyDetector != nil {
		et.anomalyDetector.Clear()
	}
}

// removeOldestError removes the oldest error entry
func (et *ErrorTracker) removeOldestError() {
	var oldestKey string
	var oldestTime time.Time = time.Now()

	for key, summary := range et.errors {
		if summary.FirstSeen.Before(oldestTime) {
			oldestTime = summary.FirstSeen
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(et.errors, oldestKey)
	}
}

// cleanupRoutine periodically cleans up old errors
func (et *ErrorTracker) cleanupRoutine(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		et.cleanup()
	}
}

// cleanup removes errors older than 24 hours
func (et *ErrorTracker) cleanup() {
	et.mu.Lock()
	defer et.mu.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour)
	for key, summary := range et.errors {
		if summary.LastSeen.Before(cutoff) {
			delete(et.errors, key)
		}
	}
}

// AnomalyDetector detects anomalies in error patterns
type AnomalyDetector struct {
	mu         sync.RWMutex
	threshold  float64
	timeWindow time.Duration
	timeSeries map[string]*TimeSeries
}

// NewAnomalyDetector creates a new anomaly detector
func NewAnomalyDetector(threshold float64, timeWindow time.Duration) *AnomalyDetector {
	return &AnomalyDetector{
		threshold:  threshold,
		timeWindow: timeWindow,
		timeSeries: make(map[string]*TimeSeries),
	}
}

// DetectAnomaly checks if the given value is anomalous
func (ad *AnomalyDetector) DetectAnomaly(key string, value float64) bool {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	ts, exists := ad.timeSeries[key]
	if !exists {
		ts = NewTimeSeries(ad.timeWindow)
		ad.timeSeries[key] = ts
	}

	return ts.IsAnomaly(value, ad.threshold)
}

// Clear clears all time series data
func (ad *AnomalyDetector) Clear() {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	ad.timeSeries = make(map[string]*TimeSeries)
}

// TimeSeries maintains a time series of values for anomaly detection
type TimeSeries struct {
	mu         sync.RWMutex
	values     *CircularBuffer
	timeWindow time.Duration
	lastUpdate time.Time
}

// NewTimeSeries creates a new time series
func NewTimeSeries(timeWindow time.Duration) *TimeSeries {
	return &TimeSeries{
		values:     NewCircularBuffer(1000), // Keep last 1000 values
		timeWindow: timeWindow,
		lastUpdate: time.Now(),
	}
}

// IsAnomaly checks if the value is anomalous using simple statistical method
func (ts *TimeSeries) IsAnomaly(value, threshold float64) bool {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.values.Add(value)
	ts.lastUpdate = time.Now()

	if ts.values.Size() < 10 {
		return false // Need at least 10 data points
	}

	mean, stddev := ts.values.Stats()
	
	// Z-score based anomaly detection
	zScore := (value - mean) / stddev
	return zScore > threshold || zScore < -threshold
}

// CircularBuffer implements a circular buffer for storing values
type CircularBuffer struct {
	mu     sync.RWMutex
	data   []float64
	head   int
	size   int
	maxSize int
}

// NewCircularBuffer creates a new circular buffer
func NewCircularBuffer(maxSize int) *CircularBuffer {
	return &CircularBuffer{
		data:    make([]float64, maxSize),
		maxSize: maxSize,
	}
}

// Add adds a value to the buffer
func (cb *CircularBuffer) Add(value float64) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.data[cb.head] = value
	cb.head = (cb.head + 1) % cb.maxSize
	
	if cb.size < cb.maxSize {
		cb.size++
	}
}

// Size returns the current size of the buffer
func (cb *CircularBuffer) Size() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.size
}

// Stats calculates mean and standard deviation
func (cb *CircularBuffer) Stats() (mean, stddev float64) {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if cb.size == 0 {
		return 0, 0
	}

	// Calculate mean
	sum := 0.0
	for i := 0; i < cb.size; i++ {
		sum += cb.data[i]
	}
	mean = sum / float64(cb.size)

	// Calculate standard deviation
	if cb.size < 2 {
		return mean, 0
	}

	sumSquares := 0.0
	for i := 0; i < cb.size; i++ {
		diff := cb.data[i] - mean
		sumSquares += diff * diff
	}
	
	stddev = (sumSquares / float64(cb.size-1))
	if stddev > 0 {
		stddev = float64(int(stddev*1000000)) / 1000000 // Simple sqrt approximation
	}

	return mean, stddev
}

// LatencyTracker tracks request latencies
type LatencyTracker struct {
	mu        sync.RWMutex
	latencies map[string]*CircularBuffer
	maxSize   int
}

// NewLatencyTracker creates a new latency tracker
func NewLatencyTracker(maxSize int) *LatencyTracker {
	return &LatencyTracker{
		latencies: make(map[string]*CircularBuffer),
		maxSize:   maxSize,
	}
}

// RecordLatency records a latency measurement
func (lt *LatencyTracker) RecordLatency(operation string, latency time.Duration) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	if _, exists := lt.latencies[operation]; !exists {
		lt.latencies[operation] = NewCircularBuffer(lt.maxSize)
	}

	lt.latencies[operation].Add(float64(latency.Milliseconds()))
}

// GetLatencyStats returns latency statistics for an operation
func (lt *LatencyTracker) GetLatencyStats(operation string) (mean, stddev float64, count int) {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	buffer, exists := lt.latencies[operation]
	if !exists {
		return 0, 0, 0
	}

	mean, stddev = buffer.Stats()
	count = buffer.Size()
	return
}

// Shutdown gracefully shuts down the error tracker
func (et *ErrorTracker) Shutdown(ctx context.Context) error {
	et.logger.Info("Shutting down error tracker")
	
	// Clear all data
	et.ClearErrors()
	
	return nil
}