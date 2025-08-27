package llm

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// AdvancedCircuitBreaker provides enhanced circuit breaker functionality
type AdvancedCircuitBreaker struct {
	// Configuration
	failureThreshold      int64
	successThreshold      int64
	timeout               time.Duration
	maxConcurrentRequests int64

	// State management
	state              int32 // Use int32 for atomic operations, convert to/from CircuitState
	failureCount       int64
	successCount       int64
	lastFailureTime    int64
	concurrentRequests int64

	// Statistics
	totalRequests  int64
	totalFailures  int64
	totalSuccesses int64
	stateChanges   int64

	// Adaptive features
	adaptiveTimeout time.Duration
	enableAdaptive  bool
	latencyHistory  []time.Duration
	latencyMutex    sync.RWMutex

	// Observers
	stateChangeCallbacks []StateChangeCallback
	mutex                sync.RWMutex

	// Tracing
	tracer trace.Tracer
}

// Use state constants from circuit_breaker.go to avoid duplicates

// StateChangeCallback is called when circuit breaker state changes
type StateChangeCallback func(oldState, newState CircuitState, reason string)

// Use CircuitBreakerError from circuit_breaker.go to avoid duplicates

// NewAdvancedCircuitBreaker creates a new advanced circuit breaker
func NewAdvancedCircuitBreaker(config CircuitBreakerConfig) *AdvancedCircuitBreaker {
	return &AdvancedCircuitBreaker{
		failureThreshold:      int64(config.FailureThreshold),
		successThreshold:      int64(config.SuccessThreshold),
		timeout:               config.Timeout,
		maxConcurrentRequests: int64(config.MaxConcurrentRequests),
		adaptiveTimeout:       config.Timeout,
		enableAdaptive:        config.EnableAdaptiveTimeout,
		latencyHistory:        make([]time.Duration, 0, 100),
		stateChangeCallbacks:  make([]StateChangeCallback, 0),
		tracer:                otel.Tracer("nephoran-intent-operator/circuit-breaker"),
	}
}

// Execute executes an operation through the circuit breaker
func (cb *AdvancedCircuitBreaker) Execute(ctx context.Context, operation func() error) error {
	// Start tracing
	ctx, span := cb.tracer.Start(ctx, "circuit_breaker.execute")
	defer span.End()

	// Check if we can execute
	if !cb.allowRequest() {
		state := atomic.LoadInt32(&cb.state)
		span.SetAttributes(
			attribute.Int("circuit.state", int(state)),
			attribute.Bool("circuit.allowed", false),
		)
		return &CircuitBreakerError{
			CircuitName: "advanced",
			State:       CircuitState(state),
			Message:     fmt.Sprintf("circuit breaker is %s", cb.getStateName(CircuitState(state))),
		}
	}

	// Increment concurrent requests
	atomic.AddInt64(&cb.concurrentRequests, 1)
	defer atomic.AddInt64(&cb.concurrentRequests, -1)

	// Record total requests
	atomic.AddInt64(&cb.totalRequests, 1)

	// Execute with timing
	start := time.Now()
	err := operation()
	duration := time.Since(start)

	// Record latency for adaptive timeout
	if cb.enableAdaptive {
		cb.recordLatency(duration)
	}

	span.SetAttributes(
		attribute.Int64("operation.duration_ms", duration.Milliseconds()),
		attribute.Bool("operation.success", err == nil),
	)

	// Update state based on result
	if err != nil {
		cb.onFailure()
		atomic.AddInt64(&cb.totalFailures, 1)
	} else {
		cb.onSuccess()
		atomic.AddInt64(&cb.totalSuccesses, 1)
	}

	return err
}

// allowRequest determines if a request should be allowed
func (cb *AdvancedCircuitBreaker) allowRequest() bool {
	state := CircuitState(atomic.LoadInt32(&cb.state))
	now := time.Now().UnixNano()

	switch state {
	case StateClosed:
		return true

	case StateOpen:
		// Check if timeout has passed
		lastFailure := atomic.LoadInt64(&cb.lastFailureTime)
		timeout := cb.getEffectiveTimeout()
		if now-lastFailure > timeout.Nanoseconds() {
			// Try to transition to half-open
			if atomic.CompareAndSwapInt32(&cb.state, int32(StateOpen), int32(StateHalfOpen)) {
				atomic.StoreInt64(&cb.successCount, 0)
				cb.notifyStateChange(StateOpen, StateHalfOpen, "timeout_reached")
			}
			return true
		}
		return false

	case StateHalfOpen:
		// Allow limited concurrent requests
		concurrent := atomic.LoadInt64(&cb.concurrentRequests)
		return concurrent < cb.maxConcurrentRequests/2 // Allow half the normal capacity

	default:
		return false
	}
}

// onFailure handles failure events
func (cb *AdvancedCircuitBreaker) onFailure() {
	atomic.StoreInt64(&cb.lastFailureTime, time.Now().UnixNano())
	failureCount := atomic.AddInt64(&cb.failureCount, 1)

	state := CircuitState(atomic.LoadInt32(&cb.state))

	// Transition to open if threshold exceeded
	if (state == StateClosed || state == StateHalfOpen) && failureCount >= cb.failureThreshold {
		if atomic.CompareAndSwapInt32(&cb.state, int32(state), int32(StateOpen)) {
			atomic.StoreInt64(&cb.successCount, 0)
			cb.notifyStateChange(state, StateOpen, "failure_threshold_exceeded")
		}
	}
}

// onSuccess handles success events
func (cb *AdvancedCircuitBreaker) onSuccess() {
	successCount := atomic.AddInt64(&cb.successCount, 1)
	state := CircuitState(atomic.LoadInt32(&cb.state))

	// Reset failure count on success
	atomic.StoreInt64(&cb.failureCount, 0)

	// Transition from half-open to closed if success threshold reached
	if state == StateHalfOpen && successCount >= cb.successThreshold {
		if atomic.CompareAndSwapInt32(&cb.state, int32(StateHalfOpen), int32(StateClosed)) {
			atomic.StoreInt64(&cb.successCount, 0)
			cb.notifyStateChange(StateHalfOpen, StateClosed, "success_threshold_reached")
		}
	}
}

// recordLatency records latency for adaptive timeout calculation
func (cb *AdvancedCircuitBreaker) recordLatency(duration time.Duration) {
	cb.latencyMutex.Lock()
	defer cb.latencyMutex.Unlock()

	cb.latencyHistory = append(cb.latencyHistory, duration)

	// Keep only last 100 measurements
	if len(cb.latencyHistory) > 100 {
		cb.latencyHistory = cb.latencyHistory[1:]
	}

	// Update adaptive timeout every 10 measurements
	if len(cb.latencyHistory)%10 == 0 {
		cb.updateAdaptiveTimeout()
	}
}

// updateAdaptiveTimeout calculates new adaptive timeout based on latency history
func (cb *AdvancedCircuitBreaker) updateAdaptiveTimeout() {
	if len(cb.latencyHistory) < 10 {
		return
	}

	// Calculate P95 latency
	sorted := make([]time.Duration, len(cb.latencyHistory))
	copy(sorted, cb.latencyHistory)

	// Simple sort for small arrays
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	p95Index := int(float64(len(sorted)) * 0.95)
	p95Latency := sorted[p95Index]

	// Set adaptive timeout to P95 + 50% buffer
	newTimeout := p95Latency + (p95Latency / 2)

	// Ensure reasonable bounds
	minTimeout := 1 * time.Second
	maxTimeout := 60 * time.Second

	if newTimeout < minTimeout {
		newTimeout = minTimeout
	} else if newTimeout > maxTimeout {
		newTimeout = maxTimeout
	}

	cb.adaptiveTimeout = newTimeout
}

// getEffectiveTimeout returns the current effective timeout
func (cb *AdvancedCircuitBreaker) getEffectiveTimeout() time.Duration {
	if cb.enableAdaptive {
		return cb.adaptiveTimeout
	}
	return cb.timeout
}

// GetState returns the current circuit breaker state
func (cb *AdvancedCircuitBreaker) GetState() CircuitState {
	return CircuitState(atomic.LoadInt32(&cb.state))
}

// GetStateName returns the human-readable state name
func (cb *AdvancedCircuitBreaker) getStateName(state CircuitState) string {
	return state.String()
}

// GetStats returns current circuit breaker statistics
func (cb *AdvancedCircuitBreaker) GetStats() CircuitBreakerStats {
	return CircuitBreakerStats{
		State:              cb.GetState(),
		StateName:          cb.getStateName(cb.GetState()),
		TotalRequests:      atomic.LoadInt64(&cb.totalRequests),
		TotalFailures:      atomic.LoadInt64(&cb.totalFailures),
		TotalSuccesses:     atomic.LoadInt64(&cb.totalSuccesses),
		FailureCount:       atomic.LoadInt64(&cb.failureCount),
		SuccessCount:       atomic.LoadInt64(&cb.successCount),
		ConcurrentRequests: atomic.LoadInt64(&cb.concurrentRequests),
		StateChanges:       atomic.LoadInt64(&cb.stateChanges),
		EffectiveTimeout:   cb.getEffectiveTimeout(),
		LastFailureTime:    time.Unix(0, atomic.LoadInt64(&cb.lastFailureTime)),
	}
}

// CircuitBreakerStats holds circuit breaker statistics
type CircuitBreakerStats struct {
	State              CircuitState  `json:"state"`
	StateName          string        `json:"state_name"`
	TotalRequests      int64         `json:"total_requests"`
	TotalFailures      int64         `json:"total_failures"`
	TotalSuccesses     int64         `json:"total_successes"`
	FailureCount       int64         `json:"failure_count"`
	SuccessCount       int64         `json:"success_count"`
	ConcurrentRequests int64         `json:"concurrent_requests"`
	StateChanges       int64         `json:"state_changes"`
	EffectiveTimeout   time.Duration `json:"effective_timeout"`
	LastFailureTime    time.Time     `json:"last_failure_time"`
}

// AddStateChangeCallback adds a callback for state changes
func (cb *AdvancedCircuitBreaker) AddStateChangeCallback(callback StateChangeCallback) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	cb.stateChangeCallbacks = append(cb.stateChangeCallbacks, callback)
}

// notifyStateChange notifies all registered callbacks of state changes
func (cb *AdvancedCircuitBreaker) notifyStateChange(oldState, newState CircuitState, reason string) {
	atomic.AddInt64(&cb.stateChanges, 1)

	cb.mutex.RLock()
	callbacks := make([]StateChangeCallback, len(cb.stateChangeCallbacks))
	copy(callbacks, cb.stateChangeCallbacks)
	cb.mutex.RUnlock()

	for _, callback := range callbacks {
		go callback(oldState, newState, reason)
	}
}

// Reset resets the circuit breaker to its initial state
func (cb *AdvancedCircuitBreaker) Reset() {
	oldState := CircuitState(atomic.SwapInt32(&cb.state, int32(StateClosed)))
	atomic.StoreInt64(&cb.failureCount, 0)
	atomic.StoreInt64(&cb.successCount, 0)
	atomic.StoreInt64(&cb.lastFailureTime, 0)
	atomic.StoreInt64(&cb.concurrentRequests, 0)

	cb.latencyMutex.Lock()
	cb.latencyHistory = cb.latencyHistory[:0]
	cb.latencyMutex.Unlock()

	if oldState != StateClosed {
		cb.notifyStateChange(oldState, StateClosed, "manual_reset")
	}
}

// ForceOpen forces the circuit breaker to open state
func (cb *AdvancedCircuitBreaker) ForceOpen() {
	oldState := CircuitState(atomic.SwapInt32(&cb.state, int32(StateOpen)))
	atomic.StoreInt64(&cb.lastFailureTime, time.Now().UnixNano())

	if oldState != StateOpen {
		cb.notifyStateChange(oldState, StateOpen, "forced_open")
	}
}

// ForceClose forces the circuit breaker to closed state
func (cb *AdvancedCircuitBreaker) ForceClose() {
	oldState := CircuitState(atomic.SwapInt32(&cb.state, int32(StateClosed)))
	atomic.StoreInt64(&cb.failureCount, 0)
	atomic.StoreInt64(&cb.successCount, 0)

	if oldState != StateClosed {
		cb.notifyStateChange(oldState, StateClosed, "forced_close")
	}
}

// UpdateTimeout updates the circuit breaker timeout
func (cb *AdvancedCircuitBreaker) UpdateTimeout(timeout time.Duration) {
	if cb.enableAdaptive {
		cb.adaptiveTimeout = timeout
	} else {
		cb.timeout = timeout
	}
}

// IsRequestAllowed checks if a request would be allowed without executing it
func (cb *AdvancedCircuitBreaker) IsRequestAllowed() bool {
	return cb.allowRequest()
}

// GetFailureRate returns the current failure rate
func (cb *AdvancedCircuitBreaker) GetFailureRate() float64 {
	totalRequests := atomic.LoadInt64(&cb.totalRequests)
	if totalRequests == 0 {
		return 0
	}

	totalFailures := atomic.LoadInt64(&cb.totalFailures)
	return float64(totalFailures) / float64(totalRequests)
}

// GetSuccessRate returns the current success rate
func (cb *AdvancedCircuitBreaker) GetSuccessRate() float64 {
	return 1.0 - cb.GetFailureRate()
}

// HealthCheck performs a health check on the circuit breaker
func (cb *AdvancedCircuitBreaker) HealthCheck() map[string]interface{} {
	stats := cb.GetStats()

	return map[string]interface{}{
		"state":               stats.StateName,
		"healthy":             stats.State == StateClosed,
		"failure_rate":        cb.GetFailureRate(),
		"success_rate":        cb.GetSuccessRate(),
		"concurrent_requests": stats.ConcurrentRequests,
		"effective_timeout":   stats.EffectiveTimeout.String(),
		"total_requests":      stats.TotalRequests,
		"state_changes":       stats.StateChanges,
	}
}
