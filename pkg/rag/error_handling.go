//go:build !disable_rag && !test

package rag

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// ErrorHandler provides comprehensive error handling for RAG components
type ErrorHandler struct {
	config          *ErrorHandlingConfig
	logger          *slog.Logger
	metrics         *ErrorMetrics
	circuitBreakers map[string]*ComponentCircuitBreaker
	retryPolicies   map[string]*RetryPolicy
	errorCodes      map[error]ErrorCode
	mutex           sync.RWMutex
}

// ErrorHandlingConfig configures error handling behavior
type ErrorHandlingConfig struct {
	// Circuit breaker settings
	EnableCircuitBreaker       bool          `json:"enable_circuit_breaker"`
	CircuitBreakerThreshold    int           `json:"circuit_breaker_threshold"`
	CircuitBreakerTimeout      time.Duration `json:"circuit_breaker_timeout"`
	CircuitBreakerRecoveryTime time.Duration `json:"circuit_breaker_recovery_time"`

	// Retry settings
	EnableRetry        bool          `json:"enable_retry"`
	DefaultMaxRetries  int           `json:"default_max_retries"`
	DefaultRetryDelay  time.Duration `json:"default_retry_delay"`
	MaxRetryDelay      time.Duration `json:"max_retry_delay"`
	RetryBackoffFactor float64       `json:"retry_backoff_factor"`

	// Timeout settings
	DefaultTimeout    time.Duration            `json:"default_timeout"`
	ComponentTimeouts map[string]time.Duration `json:"component_timeouts"`

	// Fallback behavior
	EnableFallback  bool `json:"enable_fallback"`
	CascadeFailures bool `json:"cascade_failures"`

	// Error reporting
	EnableDetailedErrors bool    `json:"enable_detailed_errors"`
	ErrorStackTrace      bool    `json:"error_stack_trace"`
	ErrorSampling        bool    `json:"error_sampling"`
	ErrorSampleRate      float64 `json:"error_sample_rate"`

	// Recovery settings
	EnableAutoRecovery  bool          `json:"enable_auto_recovery"`
	RecoveryInterval    time.Duration `json:"recovery_interval"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
}

// ErrorMetrics tracks error statistics
type ErrorMetrics struct {
	// Error counters by component and type
	ErrorsByComponent *prometheus.CounterVec
	ErrorsByType      *prometheus.CounterVec
	ErrorsByCode      *prometheus.CounterVec

	// Circuit breaker metrics
	CircuitBreakerState *prometheus.GaugeVec
	CircuitBreakerTrips *prometheus.CounterVec

	// Retry metrics
	RetryAttempts  *prometheus.CounterVec
	RetrySuccesses *prometheus.CounterVec
	RetryFailures  *prometheus.CounterVec

	// Recovery metrics
	ComponentRecoveries *prometheus.CounterVec
	RecoveryLatency     *prometheus.HistogramVec

	// Error rate metrics
	ErrorRate       *prometheus.GaugeVec
	ErrorRateWindow time.Duration

	mutex sync.RWMutex
}

// ComponentCircuitBreaker implements circuit breaker pattern for components
type ComponentCircuitBreaker struct {
	component       string
	threshold       int
	timeout         time.Duration
	recoveryTime    time.Duration
	state           CircuitBreakerState
	failures        int
	successes       int
	lastFailureTime time.Time
	lastStateChange time.Time
	mutex           sync.RWMutex
}

// CircuitBreakerState represents circuit breaker states
type CircuitBreakerState int

const (
	CircuitBreakerClosed CircuitBreakerState = iota
	CircuitBreakerOpen
	CircuitBreakerHalfOpen
)

func (s CircuitBreakerState) String() string {
	switch s {
	case CircuitBreakerClosed:
		return "closed"
	case CircuitBreakerOpen:
		return "open"
	case CircuitBreakerHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// RetryPolicy defines retry behavior for different error types
type RetryPolicy struct {
	MaxRetries      int           `json:"max_retries"`
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffFactor   float64       `json:"backoff_factor"`
	RetryableErrors []ErrorCode   `json:"retryable_errors"`
	Jitter          bool          `json:"jitter"`
}

// ErrorCode represents standardized error codes
type ErrorCode string

const (
	// Connection errors
	ErrorCodeConnectionTimeout  ErrorCode = "CONNECTION_TIMEOUT"
	ErrorCodeConnectionRefused  ErrorCode = "CONNECTION_REFUSED"
	ErrorCodeConnectionLost     ErrorCode = "CONNECTION_LOST"
	ErrorCodeConnectionPoolFull ErrorCode = "CONNECTION_POOL_FULL"

	// Cache errors
	ErrorCodeCacheMiss       ErrorCode = "CACHE_MISS"
	ErrorCodeCacheTimeout    ErrorCode = "CACHE_TIMEOUT"
	ErrorCodeCacheEviction   ErrorCode = "CACHE_EVICTION"
	ErrorCodeCacheCorruption ErrorCode = "CACHE_CORRUPTION"
	ErrorCodeCacheOverflow   ErrorCode = "CACHE_OVERFLOW"

	// Query errors
	ErrorCodeQueryTimeout       ErrorCode = "QUERY_TIMEOUT"
	ErrorCodeQueryComplexity    ErrorCode = "QUERY_COMPLEXITY"
	ErrorCodeQueryMalformed     ErrorCode = "QUERY_MALFORMED"
	ErrorCodeQueryResourceLimit ErrorCode = "QUERY_RESOURCE_LIMIT"

	// Embedding errors
	ErrorCodeEmbeddingTimeout ErrorCode = "EMBEDDING_TIMEOUT"
	ErrorCodeEmbeddingQuota   ErrorCode = "EMBEDDING_QUOTA"
	ErrorCodeEmbeddingModel   ErrorCode = "EMBEDDING_MODEL"
	ErrorCodeEmbeddingSize    ErrorCode = "EMBEDDING_SIZE"

	// Resource errors
	ErrorCodeResourceExhausted ErrorCode = "RESOURCE_EXHAUSTED"
	ErrorCodeMemoryLimit       ErrorCode = "MEMORY_LIMIT"
	ErrorCodeDiskSpace         ErrorCode = "DISK_SPACE"
	ErrorCodeCPUThrotting      ErrorCode = "CPU_THROTTLING"

	// Data errors
	ErrorCodeDataCorruption ErrorCode = "DATA_CORRUPTION"
	ErrorCodeDataValidation ErrorCode = "DATA_VALIDATION"
	ErrorCodeDataFormat     ErrorCode = "DATA_FORMAT"
	ErrorCodeDataMissing    ErrorCode = "DATA_MISSING"

	// System errors
	ErrorCodeSystemFailure      ErrorCode = "SYSTEM_FAILURE"
	ErrorCodeInternalError      ErrorCode = "INTERNAL_ERROR"
	ErrorCodeConfigurationError ErrorCode = "CONFIGURATION_ERROR"
	ErrorCodeDependencyFailure  ErrorCode = "DEPENDENCY_FAILURE"
)

// RAGError represents a standardized error with context
type RAGError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Component  string                 `json:"component"`
	Operation  string                 `json:"operation"`
	Timestamp  time.Time              `json:"timestamp"`
	Context    map[string]interface{} `json:"context"`
	StackTrace string                 `json:"stack_trace,omitempty"`
	Cause      error                  `json:"cause,omitempty"`
	Retryable  bool                   `json:"retryable"`
	Severity   ErrorSeverity          `json:"severity"`
}

// ErrorSeverity represents error severity levels
type ErrorSeverity string

const (
	SeverityLow      ErrorSeverity = "low"
	SeverityMedium   ErrorSeverity = "medium"
	SeverityHigh     ErrorSeverity = "high"
	SeverityCritical ErrorSeverity = "critical"
)

// Error implements the error interface
func (e *RAGError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s in %s.%s: %v", e.Code, e.Message, e.Component, e.Operation, e.Cause)
	}
	return fmt.Sprintf("[%s] %s in %s.%s", e.Code, e.Message, e.Component, e.Operation)
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(config *ErrorHandlingConfig) *ErrorHandler {
	if config == nil {
		config = getDefaultErrorHandlingConfig()
	}

	eh := &ErrorHandler{
		config:          config,
		logger:          slog.Default().With("component", "error-handler"),
		metrics:         newErrorMetrics(),
		circuitBreakers: make(map[string]*ComponentCircuitBreaker),
		retryPolicies:   make(map[string]*RetryPolicy),
		errorCodes:      make(map[error]ErrorCode),
	}

	// Initialize circuit breakers for components
	components := []string{"weaviate", "redis", "embedding", "llm", "document-loader"}
	for _, component := range components {
		eh.circuitBreakers[component] = newComponentCircuitBreaker(
			component,
			config.CircuitBreakerThreshold,
			config.CircuitBreakerTimeout,
			config.CircuitBreakerRecoveryTime,
		)
	}

	// Initialize retry policies
	eh.initializeRetryPolicies()

	// Start background recovery monitoring
	if config.EnableAutoRecovery {
		go eh.startRecoveryMonitoring()
	}

	return eh
}

// HandleError processes an error and applies appropriate handling
func (eh *ErrorHandler) HandleError(ctx context.Context, err error, component, operation string) error {
	if err == nil {
		return nil
	}

	// Create RAG error with context
	ragError := eh.createRAGError(err, component, operation)

	// Record metrics
	eh.recordErrorMetrics(ragError)

	// Check circuit breaker
	if eh.config.EnableCircuitBreaker {
		if cb := eh.circuitBreakers[component]; cb != nil {
			cb.recordFailure()
			eh.updateCircuitBreakerMetrics(cb)
		}
	}

	// Apply error sampling if enabled
	if eh.config.ErrorSampling && !eh.shouldSampleError() {
		// Log but don't propagate for sampled errors
		eh.logger.Debug("Sampled error", "error", ragError.Error(), "component", component)
		return ragError
	}

	// Log error with appropriate level
	eh.logError(ragError)

	return ragError
}

// ExecuteWithErrorHandling executes a function with comprehensive error handling
func (eh *ErrorHandler) ExecuteWithErrorHandling(
	ctx context.Context,
	component, operation string,
	fn func(context.Context) error,
) error {
	// Check circuit breaker
	if eh.config.EnableCircuitBreaker {
		if cb := eh.circuitBreakers[component]; cb != nil {
			if !cb.allowRequest() {
				return eh.createCircuitBreakerError(component, operation)
			}
		}
	}

	// Apply timeout if configured
	timeout := eh.getTimeout(component)
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	// Execute with retry if enabled
	if eh.config.EnableRetry {
		return eh.executeWithRetry(ctx, component, operation, fn)
	}

	// Execute once
	err := fn(ctx)
	if err != nil {
		return eh.HandleError(ctx, err, component, operation)
	}

	// Record success
	if eh.config.EnableCircuitBreaker {
		if cb := eh.circuitBreakers[component]; cb != nil {
			cb.recordSuccess()
			eh.updateCircuitBreakerMetrics(cb)
		}
	}

	return nil
}

// executeWithRetry executes a function with retry logic
func (eh *ErrorHandler) executeWithRetry(
	ctx context.Context,
	component, operation string,
	fn func(context.Context) error,
) error {
	policy := eh.getRetryPolicy(component)
	if policy == nil {
		policy = eh.getDefaultRetryPolicy()
	}

	var lastErr error
	delay := policy.InitialDelay

	for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return eh.HandleError(ctx, ctx.Err(), component, operation)
		default:
		}

		// Execute function
		err := fn(ctx)

		if err == nil {
			// Success - record metrics and return
			if attempt > 0 {
				eh.metrics.RetrySuccesses.WithLabelValues(component, operation).Inc()
			}

			if eh.config.EnableCircuitBreaker {
				if cb := eh.circuitBreakers[component]; cb != nil {
					cb.recordSuccess()
					eh.updateCircuitBreakerMetrics(cb)
				}
			}

			return nil
		}

		lastErr = err
		eh.metrics.RetryAttempts.WithLabelValues(component, operation).Inc()

		// Check if error is retryable
		ragError := eh.createRAGError(err, component, operation)
		if !eh.isRetryable(ragError, policy) {
			break
		}

		// Don't retry on last attempt
		if attempt == policy.MaxRetries {
			break
		}

		// Wait before retry
		if delay > 0 {
			eh.logger.Debug("Retrying after delay",
				"component", component,
				"operation", operation,
				"attempt", attempt+1,
				"delay", delay,
				"error", err.Error(),
			)

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return eh.HandleError(ctx, ctx.Err(), component, operation)
			}
		}

		// Calculate next delay with backoff
		delay = time.Duration(float64(delay) * policy.BackoffFactor)
		if delay > policy.MaxDelay {
			delay = policy.MaxDelay
		}

		// Add jitter if enabled
		if policy.Jitter {
			jitter := time.Duration(float64(delay) * 0.1 * (2*eh.random() - 1))
			delay += jitter
		}
	}

	// All retries failed
	eh.metrics.RetryFailures.WithLabelValues(component, operation).Inc()

	if eh.config.EnableCircuitBreaker {
		if cb := eh.circuitBreakers[component]; cb != nil {
			cb.recordFailure()
			eh.updateCircuitBreakerMetrics(cb)
		}
	}

	return eh.HandleError(ctx, lastErr, component, operation)
}

// createRAGError creates a standardized RAG error
func (eh *ErrorHandler) createRAGError(err error, component, operation string) *RAGError {
	ragError := &RAGError{
		Code:      eh.getErrorCode(err),
		Message:   err.Error(),
		Component: component,
		Operation: operation,
		Timestamp: time.Now(),
		Context:   make(map[string]interface{}),
		Cause:     err,
		Retryable: eh.isErrorRetryable(err),
		Severity:  eh.getErrorSeverity(err),
	}

	// Add stack trace if enabled
	if eh.config.ErrorStackTrace {
		ragError.StackTrace = eh.getStackTrace()
	}

	// Add runtime context
	ragError.Context["goroutine_id"] = eh.getGoroutineID()
	ragError.Context["memory_stats"] = eh.getMemoryStats()

	return ragError
}

// getErrorCode maps an error to a standardized error code
func (eh *ErrorHandler) getErrorCode(err error) ErrorCode {
	if code, exists := eh.errorCodes[err]; exists {
		return code
	}

	// Pattern matching for common error types
	switch {
	case isTimeoutError(err):
		return ErrorCodeConnectionTimeout
	case isContextCancelled(err):
		return ErrorCodeQueryTimeout
	case isConnectionError(err):
		return ErrorCodeConnectionRefused
	case isResourceError(err):
		return ErrorCodeResourceExhausted
	case isValidationError(err):
		return ErrorCodeDataValidation
	default:
		return ErrorCodeInternalError
	}
}

// Circuit breaker methods

func newComponentCircuitBreaker(component string, threshold int, timeout, recoveryTime time.Duration) *ComponentCircuitBreaker {
	return &ComponentCircuitBreaker{
		component:    component,
		threshold:    threshold,
		timeout:      timeout,
		recoveryTime: recoveryTime,
		state:        CircuitBreakerClosed,
	}
}

func (cb *ComponentCircuitBreaker) allowRequest() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	switch cb.state {
	case CircuitBreakerClosed:
		return true
	case CircuitBreakerOpen:
		if time.Since(cb.lastStateChange) >= cb.timeout {
			cb.state = CircuitBreakerHalfOpen
			cb.successes = 0
			cb.lastStateChange = time.Now()
			return true
		}
		return false
	case CircuitBreakerHalfOpen:
		return true
	default:
		return false
	}
}

func (cb *ComponentCircuitBreaker) recordSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failures = 0

	if cb.state == CircuitBreakerHalfOpen {
		cb.successes++
		if cb.successes >= 3 { // Require 3 successes to close
			cb.state = CircuitBreakerClosed
			cb.lastStateChange = time.Now()
		}
	}
}

func (cb *ComponentCircuitBreaker) recordFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failures++
	cb.lastFailureTime = time.Now()

	if cb.state == CircuitBreakerClosed && cb.failures >= cb.threshold {
		cb.state = CircuitBreakerOpen
		cb.lastStateChange = time.Now()
	} else if cb.state == CircuitBreakerHalfOpen {
		cb.state = CircuitBreakerOpen
		cb.lastStateChange = time.Now()
	}
}

// Recovery and monitoring

func (eh *ErrorHandler) startRecoveryMonitoring() {
	ticker := time.NewTicker(eh.config.RecoveryInterval)
	defer ticker.Stop()

	for range ticker.C {
		eh.checkAndRecoverComponents()
	}
}

func (eh *ErrorHandler) checkAndRecoverComponents() {
	eh.mutex.RLock()
	circuitBreakers := make(map[string]*ComponentCircuitBreaker)
	for name, cb := range eh.circuitBreakers {
		circuitBreakers[name] = cb
	}
	eh.mutex.RUnlock()

	for component, cb := range circuitBreakers {
		if cb.state == CircuitBreakerOpen {
			eh.attemptComponentRecovery(component, cb)
		}
	}
}

func (eh *ErrorHandler) attemptComponentRecovery(component string, cb *ComponentCircuitBreaker) {
	eh.logger.Info("Attempting component recovery", "component", component)

	start := time.Now()
	recovered := eh.performHealthCheck(component)
	duration := time.Since(start)

	if recovered {
		cb.mutex.Lock()
		cb.state = CircuitBreakerHalfOpen
		cb.successes = 0
		cb.lastStateChange = time.Now()
		cb.mutex.Unlock()

		eh.metrics.ComponentRecoveries.WithLabelValues(component, "success").Inc()
		eh.metrics.RecoveryLatency.WithLabelValues(component).Observe(duration.Seconds())

		eh.logger.Info("Component recovery successful", "component", component, "duration", duration)
	} else {
		eh.metrics.ComponentRecoveries.WithLabelValues(component, "failed").Inc()
		eh.logger.Warn("Component recovery failed", "component", component, "duration", duration)
	}
}

func (eh *ErrorHandler) performHealthCheck(component string) bool {
	// This would perform actual health checks based on component type
	// For now, return true as a placeholder
	return true
}

// Utility methods

func (eh *ErrorHandler) getTimeout(component string) time.Duration {
	if timeout, exists := eh.config.ComponentTimeouts[component]; exists {
		return timeout
	}
	return eh.config.DefaultTimeout
}

func (eh *ErrorHandler) getRetryPolicy(component string) *RetryPolicy {
	if policy, exists := eh.retryPolicies[component]; exists {
		return policy
	}
	return nil
}

func (eh *ErrorHandler) getDefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:    eh.config.DefaultMaxRetries,
		InitialDelay:  eh.config.DefaultRetryDelay,
		MaxDelay:      eh.config.MaxRetryDelay,
		BackoffFactor: eh.config.RetryBackoffFactor,
		Jitter:        true,
		RetryableErrors: []ErrorCode{
			ErrorCodeConnectionTimeout,
			ErrorCodeConnectionLost,
			ErrorCodeQueryTimeout,
			ErrorCodeEmbeddingTimeout,
			ErrorCodeResourceExhausted,
		},
	}
}

func (eh *ErrorHandler) isRetryable(ragError *RAGError, policy *RetryPolicy) bool {
	if !ragError.Retryable {
		return false
	}

	for _, code := range policy.RetryableErrors {
		if ragError.Code == code {
			return true
		}
	}

	return false
}

func (eh *ErrorHandler) isErrorRetryable(err error) bool {
	// Determine if error is retryable based on type
	switch {
	case isTimeoutError(err):
		return true
	case isConnectionError(err):
		return true
	case isResourceError(err):
		return true
	case isValidationError(err):
		return false
	case isConfigurationError(err):
		return false
	default:
		return false
	}
}

func (eh *ErrorHandler) getErrorSeverity(err error) ErrorSeverity {
	switch {
	case isSystemError(err):
		return SeverityCritical
	case isConnectionError(err):
		return SeverityHigh
	case isResourceError(err):
		return SeverityMedium
	case isValidationError(err):
		return SeverityLow
	default:
		return SeverityMedium
	}
}

func (eh *ErrorHandler) shouldSampleError() bool {
	return eh.random() < eh.config.ErrorSampleRate
}

func (eh *ErrorHandler) random() float64 {
	// Simple random number generator
	return float64(time.Now().UnixNano()%1000) / 1000.0
}

func (eh *ErrorHandler) getStackTrace() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

func (eh *ErrorHandler) getGoroutineID() int64 {
	// Simplified goroutine ID extraction
	return int64(runtime.NumGoroutine())
}

func (eh *ErrorHandler) getMemoryStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"heap_alloc":  m.HeapAlloc,
		"heap_sys":    m.HeapSys,
		"stack_inuse": m.StackInuse,
		"stack_sys":   m.StackSys,
		"num_gc":      m.NumGC,
		"goroutines":  runtime.NumGoroutine(),
	}
}

// Error type checking functions

func isTimeoutError(err error) bool {
	return err == context.DeadlineExceeded
}

func isContextCancelled(err error) bool {
	return err == context.Canceled
}

func isConnectionError(err error) bool {
	// Check for common connection error patterns
	errStr := err.Error()
	connectionPatterns := []string{
		"connection refused",
		"connection timeout",
		"connection reset",
		"no route to host",
		"network unreachable",
	}

	for _, pattern := range connectionPatterns {
		if errorContains(errStr, pattern) {
			return true
		}
	}
	return false
}

func isResourceError(err error) bool {
	errStr := err.Error()
	resourcePatterns := []string{
		"out of memory",
		"resource exhausted",
		"quota exceeded",
		"rate limit",
		"too many requests",
	}

	for _, pattern := range resourcePatterns {
		if errorContains(errStr, pattern) {
			return true
		}
	}
	return false
}

func isValidationError(err error) bool {
	errStr := err.Error()
	validationPatterns := []string{
		"validation failed",
		"invalid input",
		"malformed",
		"parse error",
	}

	for _, pattern := range validationPatterns {
		if errorContains(errStr, pattern) {
			return true
		}
	}
	return false
}

func isConfigurationError(err error) bool {
	errStr := err.Error()
	configPatterns := []string{
		"configuration error",
		"missing configuration",
		"invalid configuration",
	}

	for _, pattern := range configPatterns {
		if errorContains(errStr, pattern) {
			return true
		}
	}
	return false
}

func isSystemError(err error) bool {
	errStr := err.Error()
	systemPatterns := []string{
		"system failure",
		"internal error",
		"panic",
		"fatal",
	}

	for _, pattern := range systemPatterns {
		if errorContains(errStr, pattern) {
			return true
		}
	}
	return false
}

func errorContains(str, substr string) bool {
	return len(str) >= len(substr) &&
		(len(substr) == 0 || str[0:len(substr)] == substr ||
			(len(str) > len(substr) && errorContains(str[1:], substr)))
}

// Metrics initialization

func newErrorMetrics() *ErrorMetrics {
	return &ErrorMetrics{
		ErrorsByComponent: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephoran_errors_by_component_total",
				Help: "Total errors by component",
			},
			[]string{"component", "operation"},
		),
		ErrorsByType: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephoran_errors_by_type_total",
				Help: "Total errors by type",
			},
			[]string{"error_type", "severity"},
		),
		ErrorsByCode: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephoran_errors_by_code_total",
				Help: "Total errors by error code",
			},
			[]string{"error_code", "component"},
		),
		CircuitBreakerState: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "nephoran_circuit_breaker_state",
				Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
			},
			[]string{"component"},
		),
		CircuitBreakerTrips: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephoran_circuit_breaker_trips_total",
				Help: "Total circuit breaker trips",
			},
			[]string{"component"},
		),
		RetryAttempts: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephoran_retry_attempts_total",
				Help: "Total retry attempts",
			},
			[]string{"component", "operation"},
		),
		RetrySuccesses: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephoran_retry_successes_total",
				Help: "Total successful retries",
			},
			[]string{"component", "operation"},
		),
		RetryFailures: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephoran_retry_failures_total",
				Help: "Total failed retries",
			},
			[]string{"component", "operation"},
		),
		ComponentRecoveries: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephoran_component_recoveries_total",
				Help: "Total component recovery attempts",
			},
			[]string{"component", "status"},
		),
		RecoveryLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "nephoran_recovery_latency_seconds",
				Help:    "Component recovery latency in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"component"},
		),
		ErrorRate: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "nephoran_error_rate",
				Help: "Error rate by component",
			},
			[]string{"component"},
		),
		ErrorRateWindow: 5 * time.Minute,
	}
}

func (eh *ErrorHandler) recordErrorMetrics(ragError *RAGError) {
	eh.metrics.ErrorsByComponent.WithLabelValues(ragError.Component, ragError.Operation).Inc()
	eh.metrics.ErrorsByType.WithLabelValues(string(ragError.Code), string(ragError.Severity)).Inc()
	eh.metrics.ErrorsByCode.WithLabelValues(string(ragError.Code), ragError.Component).Inc()
}

func (eh *ErrorHandler) updateCircuitBreakerMetrics(cb *ComponentCircuitBreaker) {
	stateValue := float64(cb.state)
	eh.metrics.CircuitBreakerState.WithLabelValues(cb.component).Set(stateValue)

	if cb.state == CircuitBreakerOpen {
		eh.metrics.CircuitBreakerTrips.WithLabelValues(cb.component).Inc()
	}
}

func (eh *ErrorHandler) logError(ragError *RAGError) {
	attrs := []interface{}{
		"error_code", ragError.Code,
		"component", ragError.Component,
		"operation", ragError.Operation,
		"severity", ragError.Severity,
		"retryable", ragError.Retryable,
	}

	if eh.config.EnableDetailedErrors {
		attrs = append(attrs, "context", ragError.Context)
		if ragError.StackTrace != "" {
			attrs = append(attrs, "stack_trace", ragError.StackTrace)
		}
	}

	switch ragError.Severity {
	case SeverityCritical:
		eh.logger.Error(ragError.Message, attrs...)
	case SeverityHigh:
		eh.logger.Warn(ragError.Message, attrs...)
	case SeverityMedium:
		eh.logger.Info(ragError.Message, attrs...)
	case SeverityLow:
		eh.logger.Debug(ragError.Message, attrs...)
	default:
		eh.logger.Info(ragError.Message, attrs...)
	}
}

func (eh *ErrorHandler) createCircuitBreakerError(component, operation string) error {
	return &RAGError{
		Code:      ErrorCodeDependencyFailure,
		Message:   fmt.Sprintf("Circuit breaker open for component %s", component),
		Component: component,
		Operation: operation,
		Timestamp: time.Now(),
		Context:   map[string]interface{}{"circuit_breaker": "open"},
		Retryable: false,
		Severity:  SeverityHigh,
	}
}

func (eh *ErrorHandler) initializeRetryPolicies() {
	// Weaviate retry policy
	eh.retryPolicies["weaviate"] = &RetryPolicy{
		MaxRetries:    3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      10 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
		RetryableErrors: []ErrorCode{
			ErrorCodeConnectionTimeout,
			ErrorCodeConnectionLost,
			ErrorCodeQueryTimeout,
		},
	}

	// Redis retry policy
	eh.retryPolicies["redis"] = &RetryPolicy{
		MaxRetries:    2,
		InitialDelay:  500 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 1.5,
		Jitter:        true,
		RetryableErrors: []ErrorCode{
			ErrorCodeConnectionTimeout,
			ErrorCodeConnectionLost,
			ErrorCodeCacheTimeout,
		},
	}

	// Embedding service retry policy
	eh.retryPolicies["embedding"] = &RetryPolicy{
		MaxRetries:    3,
		InitialDelay:  2 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
		RetryableErrors: []ErrorCode{
			ErrorCodeEmbeddingTimeout,
			ErrorCodeEmbeddingQuota,
			ErrorCodeResourceExhausted,
		},
	}
}

func getDefaultErrorHandlingConfig() *ErrorHandlingConfig {
	return &ErrorHandlingConfig{
		EnableCircuitBreaker:       true,
		CircuitBreakerThreshold:    5,
		CircuitBreakerTimeout:      60 * time.Second,
		CircuitBreakerRecoveryTime: 30 * time.Second,
		EnableRetry:                true,
		DefaultMaxRetries:          3,
		DefaultRetryDelay:          1 * time.Second,
		MaxRetryDelay:              30 * time.Second,
		RetryBackoffFactor:         2.0,
		DefaultTimeout:             30 * time.Second,
		ComponentTimeouts: map[string]time.Duration{
			"weaviate":  30 * time.Second,
			"redis":     5 * time.Second,
			"embedding": 60 * time.Second,
			"llm":       120 * time.Second,
		},
		EnableFallback:       true,
		CascadeFailures:      false,
		EnableDetailedErrors: true,
		ErrorStackTrace:      false, // Disable in production
		ErrorSampling:        false,
		ErrorSampleRate:      0.1,
		EnableAutoRecovery:   true,
		RecoveryInterval:     5 * time.Minute,
		HealthCheckInterval:  30 * time.Second,
	}
}
