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

package errors

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
)

// ErrorSeverity defines the severity levels for errors
type ErrorSeverity string

const (
	SeverityCritical ErrorSeverity = "critical"
	SeverityHigh     ErrorSeverity = "high"
	SeverityMedium   ErrorSeverity = "medium"
	SeverityLow      ErrorSeverity = "low"
	SeverityInfo     ErrorSeverity = "info"
)

// ErrorCategory defines categories for error classification
type ErrorCategory string

const (
	CategoryValidation    ErrorCategory = "validation"
	CategoryLLMProcessing ErrorCategory = "llm_processing"
	CategoryRAGRetrieval  ErrorCategory = "rag_retrieval"
	CategoryResourcePlan  ErrorCategory = "resource_planning"
	CategoryManifestGen   ErrorCategory = "manifest_generation"
	CategoryGitOps        ErrorCategory = "gitops"
	CategoryDeployment    ErrorCategory = "deployment"
	CategoryNetwork       ErrorCategory = "network"
	CategorySecurity      ErrorCategory = "security"
	CategoryConfiguration ErrorCategory = "configuration"
	CategoryInfrastructure ErrorCategory = "infrastructure"
	CategoryTimeout       ErrorCategory = "timeout"
	CategoryRateLimit     ErrorCategory = "rate_limit"
	CategoryCircuitBreaker ErrorCategory = "circuit_breaker"
	CategoryDependency    ErrorCategory = "dependency"
	CategoryCapacity      ErrorCategory = "capacity"
	CategoryInternal      ErrorCategory = "internal"
)

// ErrorAction defines possible actions to take when an error occurs
type ErrorAction string

const (
	ActionRetry           ErrorAction = "retry"
	ActionRetryWithBackoff ErrorAction = "retry_with_backoff"
	ActionCircuitBreaker  ErrorAction = "circuit_breaker"
	ActionFailFast        ErrorAction = "fail_fast"
	ActionDegrade         ErrorAction = "degrade"
	ActionFallback        ErrorAction = "fallback"
	ActionEscalate        ErrorAction = "escalate"
	ActionIgnore          ErrorAction = "ignore"
	ActionQuarantine      ErrorAction = "quarantine"
)

// ProcessingError represents a comprehensive error with context and recovery information
type ProcessingError struct {
	// Core error information
	ID            string        `json:"id"`
	Code          string        `json:"code"`
	Message       string        `json:"message"`
	Category      ErrorCategory `json:"category"`
	Severity      ErrorSeverity `json:"severity"`
	Phase         string        `json:"phase"`
	Component     string        `json:"component"`
	Operation     string        `json:"operation"`
	
	// Contextual information
	IntentID      string                 `json:"intentId,omitempty"`
	CorrelationID string                 `json:"correlationId,omitempty"`
	UserID        string                 `json:"userId,omitempty"`
	RequestID     string                 `json:"requestId,omitempty"`
	TraceID       string                 `json:"traceId,omitempty"`
	SpanID        string                 `json:"spanId,omitempty"`
	Context       map[string]interface{} `json:"context,omitempty"`
	
	// Error chain and causality
	Cause         error           `json:"-"`
	CauseMessage  string          `json:"cause,omitempty"`
	ErrorChain    []ErrorDetails  `json:"errorChain,omitempty"`
	
	// Recovery and retry information
	Retryable     bool            `json:"retryable"`
	RetryCount    int             `json:"retryCount"`
	MaxRetries    int             `json:"maxRetries"`
	RetryAfter    *time.Duration  `json:"retryAfter,omitempty"`
	BackoffFactor float64         `json:"backoffFactor,omitempty"`
	
	// Action and resolution
	RecommendedAction ErrorAction   `json:"recommendedAction"`
	PossibleActions   []ErrorAction `json:"possibleActions,omitempty"`
	AutoRecoverable   bool          `json:"autoRecoverable"`
	RecoveryStrategy  string        `json:"recoveryStrategy,omitempty"`
	
	// Timing and occurrence information
	Timestamp     time.Time       `json:"timestamp"`
	FirstOccurred time.Time       `json:"firstOccurred"`
	LastOccurred  time.Time       `json:"lastOccurred"`
	OccurrenceCount int           `json:"occurrenceCount"`
	
	// Impact and metrics
	Impact        ErrorImpact     `json:"impact"`
	Metrics       ErrorMetrics    `json:"metrics"`
	
	// Resource and environment information
	ResourceType  string          `json:"resourceType,omitempty"`
	ResourceName  string          `json:"resourceName,omitempty"`
	Namespace     string          `json:"namespace,omitempty"`
	NodeName      string          `json:"nodeName,omitempty"`
	Environment   string          `json:"environment,omitempty"`
	
	mutex         sync.RWMutex
}

// ErrorDetails provides detailed information about individual errors in a chain
type ErrorDetails struct {
	Message   string                 `json:"message"`
	Code      string                 `json:"code,omitempty"`
	Component string                 `json:"component"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// ErrorImpact describes the impact of an error
type ErrorImpact struct {
	UserExperience    string   `json:"userExperience"`    // degraded, unavailable, normal
	ServiceLevel      string   `json:"serviceLevel"`      // critical, major, minor
	AffectedUsers     int64    `json:"affectedUsers"`
	AffectedResources []string `json:"affectedResources,omitempty"`
	BusinessImpact    string   `json:"businessImpact,omitempty"`
	SLAViolation      bool     `json:"slaViolation"`
	EstimatedDowntime *time.Duration `json:"estimatedDowntime,omitempty"`
}

// ErrorMetrics contains performance and statistical metrics related to the error
type ErrorMetrics struct {
	ProcessingTimeMs  int64             `json:"processingTimeMs"`
	MemoryUsageMB     int64             `json:"memoryUsageMB"`
	CPUUsagePercent   float64           `json:"cpuUsagePercent"`
	NetworkLatencyMs  int64             `json:"networkLatencyMs,omitempty"`
	DatabaseLatencyMs int64             `json:"databaseLatencyMs,omitempty"`
	QueueDepth        int64             `json:"queueDepth,omitempty"`
	CustomMetrics     map[string]float64 `json:"customMetrics,omitempty"`
}

// ErrorAggregator collects and correlates errors across the system
type ErrorAggregator struct {
	errors           map[string]*ProcessingError
	correlatedErrors map[string][]*ProcessingError
	errorCounts      map[string]int64
	errorRates       map[string]*ErrorRateTracker
	mutex            sync.RWMutex
	logger           logr.Logger
	
	// Configuration
	MaxErrorHistory  int           `json:"maxErrorHistory"`
	CorrelationWindow time.Duration `json:"correlationWindow"`
	CleanupInterval  time.Duration `json:"cleanupInterval"`
	
	// Channels for async processing
	errorChan        chan *ProcessingError
	stopChan         chan struct{}
	started          bool
}

// ErrorRateTracker tracks error rates over time
type ErrorRateTracker struct {
	WindowSize     time.Duration `json:"windowSize"`
	BucketCount    int           `json:"bucketCount"`
	buckets        []ErrorBucket
	currentBucket  int
	mutex          sync.RWMutex
}

// ErrorBucket represents a time bucket for error rate calculation
type ErrorBucket struct {
	StartTime    time.Time `json:"startTime"`
	ErrorCount   int64     `json:"errorCount"`
	RequestCount int64     `json:"requestCount"`
}

// ErrorCorrelationRule defines rules for correlating related errors
type ErrorCorrelationRule struct {
	Name          string        `json:"name"`
	Pattern       string        `json:"pattern"`
	TimeWindow    time.Duration `json:"timeWindow"`
	Categories    []ErrorCategory `json:"categories"`
	Threshold     int           `json:"threshold"`
	Action        ErrorAction   `json:"action"`
	Enabled       bool          `json:"enabled"`
}

// RecoveryContext provides context for error recovery operations
type RecoveryContext struct {
	ErrorID         string                 `json:"errorId"`
	IntentID        string                 `json:"intentId"`
	CorrelationID   string                 `json:"correlationId"`
	Phase           string                 `json:"phase"`
	Component       string                 `json:"component"`
	
	// Recovery state
	RecoveryAttempt int                    `json:"recoveryAttempt"`
	RecoveryType    string                 `json:"recoveryType"`
	RecoveryData    map[string]interface{} `json:"recoveryData"`
	
	// Context data
	OriginalContext map[string]interface{} `json:"originalContext"`
	RecoveryContext map[string]interface{} `json:"recoveryContext"`
	
	// Timing
	StartTime       time.Time              `json:"startTime"`
	LastAttempt     time.Time              `json:"lastAttempt"`
	NextAttempt     time.Time              `json:"nextAttempt"`
	Timeout         time.Duration          `json:"timeout"`
	
	// Tracking
	Logger          logr.Logger            `json:"-"`
	Metrics         map[string]float64     `json:"metrics,omitempty"`
}

// Error implements the error interface
func (e *ProcessingError) Error() string {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	if e.CauseMessage != "" {
		return fmt.Sprintf("[%s] %s: %s (cause: %s)", e.Code, e.Component, e.Message, e.CauseMessage)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Code, e.Component, e.Message)
}

// Unwrap returns the underlying cause
func (e *ProcessingError) Unwrap() error {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.Cause
}

// AddToChain adds an error to the error chain
func (e *ProcessingError) AddToChain(message, code, component string, context map[string]interface{}) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	e.ErrorChain = append(e.ErrorChain, ErrorDetails{
		Message:   message,
		Code:      code,
		Component: component,
		Context:   context,
		Timestamp: time.Now(),
	})
}

// IncrementRetry increments the retry count and updates timing
func (e *ProcessingError) IncrementRetry() {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	e.RetryCount++
	e.LastOccurred = time.Now()
	e.OccurrenceCount++
	
	// Calculate next retry delay with exponential backoff
	if e.BackoffFactor > 0 {
		baseDelay := time.Second
		if e.RetryAfter != nil {
			baseDelay = *e.RetryAfter
		}
		
		delay := time.Duration(float64(baseDelay) * 
			(e.BackoffFactor * float64(e.RetryCount)))
		e.RetryAfter = &delay
	}
}

// IsRetryExhausted checks if retry attempts are exhausted
func (e *ProcessingError) IsRetryExhausted() bool {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.RetryCount >= e.MaxRetries
}

// GetRetryDelay returns the delay before next retry
func (e *ProcessingError) GetRetryDelay() time.Duration {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	if e.RetryAfter != nil {
		return *e.RetryAfter
	}
	return 0
}

// UpdateImpact updates the error impact information
func (e *ProcessingError) UpdateImpact(impact ErrorImpact) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.Impact = impact
}

// AddMetric adds a custom metric to the error
func (e *ProcessingError) AddMetric(key string, value float64) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	if e.Metrics.CustomMetrics == nil {
		e.Metrics.CustomMetrics = make(map[string]float64)
	}
	e.Metrics.CustomMetrics[key] = value
}

// NewProcessingError creates a new ProcessingError
func NewProcessingError(code, message, component, operation string, category ErrorCategory, severity ErrorSeverity) *ProcessingError {
	now := time.Now()
	
	return &ProcessingError{
		ID:              generateErrorID(),
		Code:            code,
		Message:         message,
		Category:        category,
		Severity:        severity,
		Component:       component,
		Operation:       operation,
		Timestamp:       now,
		FirstOccurred:   now,
		LastOccurred:    now,
		OccurrenceCount: 1,
		Retryable:       isRetryableByDefault(category),
		MaxRetries:      getDefaultMaxRetries(category),
		RecommendedAction: getDefaultAction(category, severity),
		Context:         make(map[string]interface{}),
		ErrorChain:      make([]ErrorDetails, 0),
	}
}

// NewErrorAggregator creates a new ErrorAggregator
func NewErrorAggregator(logger logr.Logger) *ErrorAggregator {
	return &ErrorAggregator{
		errors:           make(map[string]*ProcessingError),
		correlatedErrors: make(map[string][]*ProcessingError),
		errorCounts:      make(map[string]int64),
		errorRates:       make(map[string]*ErrorRateTracker),
		logger:           logger,
		MaxErrorHistory:  10000,
		CorrelationWindow: 5 * time.Minute,
		CleanupInterval:  15 * time.Minute,
		errorChan:        make(chan *ProcessingError, 1000),
		stopChan:         make(chan struct{}),
	}
}

// Start starts the error aggregator background processing
func (ea *ErrorAggregator) Start(ctx context.Context) error {
	ea.mutex.Lock()
	defer ea.mutex.Unlock()
	
	if ea.started {
		return fmt.Errorf("error aggregator already started")
	}
	
	ea.started = true
	
	// Start background goroutines
	go ea.processErrors(ctx)
	go ea.cleanup(ctx)
	go ea.correlateErrors(ctx)
	
	ea.logger.Info("Error aggregator started")
	return nil
}

// Stop stops the error aggregator
func (ea *ErrorAggregator) Stop() error {
	ea.mutex.Lock()
	defer ea.mutex.Unlock()
	
	if !ea.started {
		return nil
	}
	
	close(ea.stopChan)
	ea.started = false
	
	ea.logger.Info("Error aggregator stopped")
	return nil
}

// RecordError records a new error
func (ea *ErrorAggregator) RecordError(err *ProcessingError) {
	if !ea.started {
		ea.logger.Error(fmt.Errorf("error aggregator not started"), "Cannot record error")
		return
	}
	
	select {
	case ea.errorChan <- err:
		// Successfully queued
	default:
		// Channel full, log warning
		ea.logger.Error(fmt.Errorf("error channel full"), "Dropping error", "errorId", err.ID)
	}
}

// GetError retrieves an error by ID
func (ea *ErrorAggregator) GetError(errorID string) (*ProcessingError, bool) {
	ea.mutex.RLock()
	defer ea.mutex.RUnlock()
	
	err, exists := ea.errors[errorID]
	return err, exists
}

// GetCorrelatedErrors returns errors correlated with the given error
func (ea *ErrorAggregator) GetCorrelatedErrors(errorID string) []*ProcessingError {
	ea.mutex.RLock()
	defer ea.mutex.RUnlock()
	
	return ea.correlatedErrors[errorID]
}

// GetErrorStatistics returns error statistics
func (ea *ErrorAggregator) GetErrorStatistics() map[string]interface{} {
	ea.mutex.RLock()
	defer ea.mutex.RUnlock()
	
	stats := make(map[string]interface{})
	
	// Basic counts
	stats["totalErrors"] = len(ea.errors)
	stats["errorCounts"] = ea.errorCounts
	
	// Category breakdown
	categoryStats := make(map[string]int)
	severityStats := make(map[string]int)
	
	for _, err := range ea.errors {
		categoryStats[string(err.Category)]++
		severityStats[string(err.Severity)]++
	}
	
	stats["errorsByCategory"] = categoryStats
	stats["errorsBySeverity"] = severityStats
	
	// Error rates
	rates := make(map[string]float64)
	for key, tracker := range ea.errorRates {
		rates[key] = tracker.GetCurrentRate()
	}
	stats["errorRates"] = rates
	
	return stats
}

// processErrors processes errors from the channel
func (ea *ErrorAggregator) processErrors(ctx context.Context) {
	for {
		select {
		case err := <-ea.errorChan:
			ea.storeError(err)
			ea.updateErrorRates(err)
			
		case <-ea.stopChan:
			return
			
		case <-ctx.Done():
			return
		}
	}
}

// storeError stores an error in the aggregator
func (ea *ErrorAggregator) storeError(err *ProcessingError) {
	ea.mutex.Lock()
	defer ea.mutex.Unlock()
	
	// Check if this is a duplicate error (same correlation ID and code)
	key := fmt.Sprintf("%s-%s", err.CorrelationID, err.Code)
	if existingErr, exists := ea.errors[key]; exists {
		// Update existing error
		existingErr.IncrementRetry()
		existingErr.LastOccurred = err.Timestamp
		return
	}
	
	// Store new error
	ea.errors[err.ID] = err
	
	// Update counts
	countKey := fmt.Sprintf("%s-%s", err.Category, err.Severity)
	ea.errorCounts[countKey]++
	
	// Enforce maximum history
	if len(ea.errors) > ea.MaxErrorHistory {
		ea.evictOldestErrors()
	}
}

// updateErrorRates updates error rate tracking
func (ea *ErrorAggregator) updateErrorRates(err *ProcessingError) {
	key := fmt.Sprintf("%s-%s", err.Component, err.Category)
	
	ea.mutex.Lock()
	defer ea.mutex.Unlock()
	
	if tracker, exists := ea.errorRates[key]; exists {
		tracker.RecordError()
	} else {
		tracker := NewErrorRateTracker(time.Minute, 60) // 1-minute buckets, 1-hour window
		tracker.RecordError()
		ea.errorRates[key] = tracker
	}
}

// cleanup performs periodic cleanup of old errors
func (ea *ErrorAggregator) cleanup(ctx context.Context) {
	ticker := time.NewTicker(ea.CleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			ea.performCleanup()
			
		case <-ea.stopChan:
			return
			
		case <-ctx.Done():
			return
		}
	}
}

// performCleanup removes old errors and updates statistics
func (ea *ErrorAggregator) performCleanup() {
	ea.mutex.Lock()
	defer ea.mutex.Unlock()
	
	cutoff := time.Now().Add(-24 * time.Hour) // Keep 24 hours of errors
	
	toDelete := make([]string, 0)
	for id, err := range ea.errors {
		if err.Timestamp.Before(cutoff) {
			toDelete = append(toDelete, id)
		}
	}
	
	for _, id := range toDelete {
		delete(ea.errors, id)
	}
	
	if len(toDelete) > 0 {
		ea.logger.Info("Cleaned up old errors", "count", len(toDelete))
	}
}

// correlateErrors identifies correlated errors based on patterns
func (ea *ErrorAggregator) correlateErrors(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			ea.performCorrelation()
			
		case <-ea.stopChan:
			return
			
		case <-ctx.Done():
			return
		}
	}
}

// performCorrelation correlates related errors
func (ea *ErrorAggregator) performCorrelation() {
	ea.mutex.Lock()
	defer ea.mutex.Unlock()
	
	// Simple correlation based on correlation ID and time window
	correlationGroups := make(map[string][]*ProcessingError)
	
	for _, err := range ea.errors {
		if err.CorrelationID != "" {
			correlationGroups[err.CorrelationID] = append(correlationGroups[err.CorrelationID], err)
		}
	}
	
	// Update correlated errors
	for correlationID, errors := range correlationGroups {
		if len(errors) > 1 {
			for _, err := range errors {
				ea.correlatedErrors[err.ID] = errors
			}
		}
	}
}

// evictOldestErrors removes the oldest errors when limit is exceeded
func (ea *ErrorAggregator) evictOldestErrors() {
	// Simple LRU eviction based on timestamp
	var oldestID string
	var oldestTime time.Time
	
	for id, err := range ea.errors {
		if oldestID == "" || err.Timestamp.Before(oldestTime) {
			oldestID = id
			oldestTime = err.Timestamp
		}
	}
	
	if oldestID != "" {
		delete(ea.errors, oldestID)
	}
}

// NewErrorRateTracker creates a new error rate tracker
func NewErrorRateTracker(windowSize time.Duration, bucketCount int) *ErrorRateTracker {
	tracker := &ErrorRateTracker{
		WindowSize:  windowSize,
		BucketCount: bucketCount,
		buckets:     make([]ErrorBucket, bucketCount),
	}
	
	// Initialize buckets
	now := time.Now()
	bucketDuration := windowSize / time.Duration(bucketCount)
	
	for i := 0; i < bucketCount; i++ {
		tracker.buckets[i] = ErrorBucket{
			StartTime: now.Add(-time.Duration(bucketCount-i) * bucketDuration),
		}
	}
	
	return tracker
}

// RecordError records an error occurrence
func (t *ErrorRateTracker) RecordError() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	
	now := time.Now()
	bucketDuration := t.WindowSize / time.Duration(t.BucketCount)
	
	// Find the current bucket
	for i := range t.buckets {
		bucketStart := t.buckets[i].StartTime
		bucketEnd := bucketStart.Add(bucketDuration)
		
		if now.After(bucketStart) && now.Before(bucketEnd) {
			t.buckets[i].ErrorCount++
			return
		}
	}
	
	// If we're here, we need to rotate buckets
	t.rotateBuckets(now, bucketDuration)
	t.buckets[t.BucketCount-1].ErrorCount++
}

// GetCurrentRate returns the current error rate (errors per second)
func (t *ErrorRateTracker) GetCurrentRate() float64 {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	
	var totalErrors int64
	for _, bucket := range t.buckets {
		totalErrors += bucket.ErrorCount
	}
	
	return float64(totalErrors) / t.WindowSize.Seconds()
}

// rotateBuckets rotates the buckets to accommodate new time windows
func (t *ErrorRateTracker) rotateBuckets(now time.Time, bucketDuration time.Duration) {
	// Shift all buckets left and create new bucket at the end
	copy(t.buckets[0:], t.buckets[1:])
	t.buckets[t.BucketCount-1] = ErrorBucket{
		StartTime:  now.Truncate(bucketDuration),
		ErrorCount: 0,
	}
}

// Helper functions

// generateErrorID generates a unique error ID
func generateErrorID() string {
	return fmt.Sprintf("err_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond())
}

// isRetryableByDefault returns true if errors in the category are retryable by default
func isRetryableByDefault(category ErrorCategory) bool {
	retryableCategories := []ErrorCategory{
		CategoryLLMProcessing,
		CategoryRAGRetrieval,
		CategoryResourcePlan,
		CategoryManifestGen,
		CategoryGitOps,
		CategoryNetwork,
		CategoryTimeout,
		CategoryRateLimit,
		CategoryDependency,
		CategoryCapacity,
	}
	
	for _, retryable := range retryableCategories {
		if category == retryable {
			return true
		}
	}
	
	return false
}

// getDefaultMaxRetries returns default max retries for a category
func getDefaultMaxRetries(category ErrorCategory) int {
	switch category {
	case CategoryValidation, CategorySecurity:
		return 1
	case CategoryLLMProcessing, CategoryRAGRetrieval:
		return 3
	case CategoryNetwork, CategoryTimeout:
		return 5
	case CategoryDependency, CategoryCapacity:
		return 2
	default:
		return 3
	}
}

// getDefaultAction returns the default action for a category and severity
func getDefaultAction(category ErrorCategory, severity ErrorSeverity) ErrorAction {
	if severity == SeverityCritical {
		return ActionEscalate
	}
	
	switch category {
	case CategoryValidation, CategorySecurity:
		return ActionFailFast
	case CategoryLLMProcessing, CategoryRAGRetrieval:
		return ActionRetryWithBackoff
	case CategoryNetwork, CategoryTimeout:
		return ActionRetryWithBackoff
	case CategoryCircuitBreaker:
		return ActionCircuitBreaker
	case CategoryDependency:
		return ActionFallback
	default:
		return ActionRetry
	}
}