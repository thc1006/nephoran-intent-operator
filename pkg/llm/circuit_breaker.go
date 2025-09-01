//go:build !disable_rag
// +build !disable_rag

package llm

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// CircuitBreaker implements the circuit breaker pattern for fault tolerance.

type CircuitBreaker struct {
	name string

	config *CircuitBreakerConfig

	state CircuitState

	failureCount int64

	successCount int64

	requestCount int64

	lastFailureTime time.Time

	lastSuccessTime time.Time

	stateChangeTime time.Time

	halfOpenRequests int64

	logger *slog.Logger

	metrics *CircuitMetrics

	mutex sync.RWMutex

	onStateChange func(name string, from, to CircuitState)
}

// CircuitBreakerConfig holds configuration for circuit breaker.

type CircuitBreakerConfig struct {

	// Failure threshold.

	FailureThreshold int64 `json:"failure_threshold"`

	FailureRate float64 `json:"failure_rate"` // 0.0 to 1.0

	MinimumRequestCount int64 `json:"minimum_request_count"`

	// Timeout settings.

	Timeout time.Duration `json:"timeout"`

	HalfOpenTimeout time.Duration `json:"half_open_timeout"`

	// Recovery settings.

	SuccessThreshold int64 `json:"success_threshold"`

	HalfOpenMaxRequests int64 `json:"half_open_max_requests"`

	// Reset settings.

	ResetTimeout time.Duration `json:"reset_timeout"`

	SlidingWindowSize int `json:"sliding_window_size"`

	// Health check.

	EnableHealthCheck bool `json:"enable_health_check"`

	HealthCheckInterval time.Duration `json:"health_check_interval"`

	HealthCheckTimeout time.Duration `json:"health_check_timeout"`

	// Advanced circuit breaker settings.

	MaxConcurrentRequests int `json:"max_concurrent_requests"`

	EnableAdaptiveTimeout bool `json:"enable_adaptive_timeout"`
}

// CircuitState represents the state of the circuit breaker.

type CircuitState int

const (

	// StateClosed holds stateclosed value.

	StateClosed CircuitState = iota

	// StateOpen holds stateopen value.

	StateOpen

	// StateHalfOpen holds statehalfopen value.

	StateHalfOpen
)

// String performs string operation.

func (s CircuitState) String() string {

	switch s {

	case StateClosed:

		return "Closed"

	case StateOpen:

		return "Open"

	case StateHalfOpen:

		return "HalfOpen"

	default:

		return "Unknown"

	}

}

// CircuitMetrics tracks circuit breaker performance.

type CircuitMetrics struct {
	TotalRequests int64 `json:"total_requests"`

	SuccessfulRequests int64 `json:"successful_requests"`

	FailedRequests int64 `json:"failed_requests"`

	RejectedRequests int64 `json:"rejected_requests"`

	TimeoutRequests int64 `json:"timeout_requests"`

	StateTransitions int64 `json:"state_transitions"`

	CurrentState string `json:"current_state"`

	LastStateChange time.Time `json:"last_state_change"`

	FailureRate float64 `json:"failure_rate"`

	AverageLatency time.Duration `json:"average_latency"`

	LastUpdated time.Time `json:"last_updated"`

	mutex sync.RWMutex
}

// CircuitBreakerError represents an error from circuit breaker.

type CircuitBreakerError struct {
	CircuitName string

	State CircuitState

	Message string
}

// Error performs error operation.

func (e *CircuitBreakerError) Error() string {

	return fmt.Sprintf("circuit breaker '%s' is %s: %s", e.CircuitName, e.State.String(), e.Message)

}

// CircuitOperation represents an operation to be executed through circuit breaker.

type CircuitOperation func(context.Context) (interface{}, error)

// NewCircuitBreaker creates a new circuit breaker.

func NewCircuitBreaker(name string, config *CircuitBreakerConfig) *CircuitBreaker {

	if config == nil {

		config = getDefaultCircuitBreakerConfig()

	}

	cb := &CircuitBreaker{

		name: name,

		config: config,

		state: StateClosed,

		stateChangeTime: time.Now(),

		logger: slog.Default().With("component", "circuit-breaker", "name", name),

		metrics: &CircuitMetrics{

			CurrentState: StateClosed.String(),

			LastStateChange: time.Now(),

			LastUpdated: time.Now(),
		},
	}

	// Start health check routine if enabled.

	if config.EnableHealthCheck {

		go cb.healthCheckRoutine()

	}

	return cb

}

// getDefaultCircuitBreakerConfig returns default configuration.

func getDefaultCircuitBreakerConfig() *CircuitBreakerConfig {

	return &CircuitBreakerConfig{

		FailureThreshold: 5,

		FailureRate: 0.5,

		MinimumRequestCount: 10,

		Timeout: 30 * time.Second,

		HalfOpenTimeout: 60 * time.Second,

		SuccessThreshold: 3,

		HalfOpenMaxRequests: 5,

		ResetTimeout: 60 * time.Second,

		SlidingWindowSize: 100,

		EnableHealthCheck: false,

		HealthCheckInterval: 30 * time.Second,

		HealthCheckTimeout: 10 * time.Second,
	}

}

// Call executes a simple function through the circuit breaker.

// This method provides a simpler interface compared to Execute for functions that don't need context.

func (cb *CircuitBreaker) Call(fn func() error) error {

	ctx, cancel := context.WithTimeout(context.Background(), cb.config.Timeout)

	defer cancel()

	_, err := cb.Execute(ctx, func(context.Context) (interface{}, error) {

		return nil, fn()

	})

	return err

}

// ExecuteSimple provides a simpler Execute interface for functions that return only an error.

// This is needed for compatibility with test frameworks that expect Execute(func() error) error.

func (cb *CircuitBreaker) ExecuteSimple(fn func() error) error {

	return cb.Call(fn)

}

// Execute executes an operation through the circuit breaker.

func (cb *CircuitBreaker) Execute(ctx context.Context, operation CircuitOperation) (interface{}, error) {

	startTime := time.Now()

	// Check if circuit is open.

	if !cb.canExecute() {

		cb.updateMetrics(func(m *CircuitMetrics) {

			m.RejectedRequests++

			m.TotalRequests++

		})

		return nil, &CircuitBreakerError{

			CircuitName: cb.name,

			State: cb.getState(),

			Message: "circuit breaker is open",
		}

	}

	// Create timeout context.

	operationCtx, cancel := context.WithTimeout(ctx, cb.config.Timeout)

	defer cancel()

	// Execute operation.

	result, err := cb.executeWithTimeout(operationCtx, operation)

	// Update circuit state based on result.

	cb.recordResult(err, time.Since(startTime))

	return result, err

}

// canExecute checks if the circuit breaker allows execution.

func (cb *CircuitBreaker) canExecute() bool {

	cb.mutex.RLock()

	defer cb.mutex.RUnlock()

	switch cb.state {

	case StateClosed:

		return true

	case StateOpen:

		// Check if we should transition to half-open.

		if time.Since(cb.stateChangeTime) >= cb.config.ResetTimeout {

			cb.transitionToHalfOpen()

			return true

		}

		return false

	case StateHalfOpen:

		// Allow limited requests in half-open state.

		return cb.halfOpenRequests < cb.config.HalfOpenMaxRequests

	default:

		return false

	}

}

// executeWithTimeout executes operation with timeout handling.

func (cb *CircuitBreaker) executeWithTimeout(ctx context.Context, operation CircuitOperation) (interface{}, error) {

	resultChan := make(chan interface{}, 1)

	errorChan := make(chan error, 1)

	go func() {

		defer func() {

			if r := recover(); r != nil {

				errorChan <- fmt.Errorf("operation panicked: %v", r)

			}

		}()

		result, err := operation(ctx)

		if err != nil {

			errorChan <- err

		} else {

			resultChan <- result

		}

	}()

	select {

	case result := <-resultChan:

		return result, nil

	case err := <-errorChan:

		return nil, err

	case <-ctx.Done():

		cb.updateMetrics(func(m *CircuitMetrics) {

			m.TimeoutRequests++

		})

		return nil, fmt.Errorf("operation timeout: %w", ctx.Err())

	}

}

// recordResult records the result of an operation and updates circuit state.

func (cb *CircuitBreaker) recordResult(err error, latency time.Duration) {

	cb.mutex.Lock()

	defer cb.mutex.Unlock()

	cb.requestCount++

	if err != nil {

		cb.failureCount++

		cb.lastFailureTime = time.Now()

		cb.updateMetrics(func(m *CircuitMetrics) {

			m.FailedRequests++

			m.TotalRequests++

		})

		// Check if we should open the circuit.

		if cb.shouldOpenCircuit() {

			cb.transitionToOpen()

		}

	} else {

		cb.successCount++

		cb.lastSuccessTime = time.Now()

		cb.updateMetrics(func(m *CircuitMetrics) {

			m.SuccessfulRequests++

			m.TotalRequests++

			// Update average latency.

			totalLatency := m.AverageLatency * time.Duration(m.SuccessfulRequests-1)

			m.AverageLatency = (totalLatency + latency) / time.Duration(m.SuccessfulRequests)

		})

		// Check if we should close the circuit (from half-open state).

		if cb.state == StateHalfOpen && cb.shouldCloseCircuit() {

			cb.transitionToClosed()

		}

	}

	// Update failure rate.

	if cb.requestCount >= cb.config.MinimumRequestCount {

		failureRate := float64(cb.failureCount) / float64(cb.requestCount)

		cb.updateMetrics(func(m *CircuitMetrics) {

			m.FailureRate = failureRate

		})

	}

	if cb.state == StateHalfOpen {

		cb.halfOpenRequests++

	}

}

// shouldOpenCircuit determines if the circuit should be opened.

func (cb *CircuitBreaker) shouldOpenCircuit() bool {

	if cb.state != StateClosed {

		return false

	}

	// Check failure threshold.

	if cb.failureCount >= cb.config.FailureThreshold {

		return true

	}

	// Check failure rate.

	if cb.requestCount >= cb.config.MinimumRequestCount {

		failureRate := float64(cb.failureCount) / float64(cb.requestCount)

		if failureRate >= cb.config.FailureRate {

			return true

		}

	}

	return false

}

// shouldCloseCircuit determines if the circuit should be closed (from half-open).

func (cb *CircuitBreaker) shouldCloseCircuit() bool {

	if cb.state != StateHalfOpen {

		return false

	}

	// Check success threshold in half-open state.

	recentSuccesses := cb.getRecentSuccesses()

	return recentSuccesses >= cb.config.SuccessThreshold

}

// getRecentSuccesses gets the number of recent successful requests.

func (cb *CircuitBreaker) getRecentSuccesses() int64 {

	// Simplified implementation - in production, use sliding window.

	if cb.state == StateHalfOpen {

		return cb.successCount

	}

	return 0

}

// transitionToOpen transitions circuit to open state.

func (cb *CircuitBreaker) transitionToOpen() {

	oldState := cb.state

	cb.state = StateOpen

	cb.stateChangeTime = time.Now()

	cb.halfOpenRequests = 0

	cb.updateMetrics(func(m *CircuitMetrics) {

		m.StateTransitions++

		m.CurrentState = StateOpen.String()

		m.LastStateChange = cb.stateChangeTime

		m.LastUpdated = time.Now()

	})

	cb.logger.Warn("Circuit breaker opened",

		"failure_count", cb.failureCount,

		"request_count", cb.requestCount,

		"failure_rate", fmt.Sprintf("%.2f%%", cb.metrics.FailureRate*100),
	)

	if cb.onStateChange != nil {

		go cb.onStateChange(cb.name, oldState, StateOpen)

	}

}

// transitionToHalfOpen transitions circuit to half-open state.

func (cb *CircuitBreaker) transitionToHalfOpen() {

	oldState := cb.state

	cb.state = StateHalfOpen

	cb.stateChangeTime = time.Now()

	cb.halfOpenRequests = 0

	cb.successCount = 0 // Reset for half-open evaluation

	cb.failureCount = 0

	cb.updateMetrics(func(m *CircuitMetrics) {

		m.StateTransitions++

		m.CurrentState = StateHalfOpen.String()

		m.LastStateChange = cb.stateChangeTime

		m.LastUpdated = time.Now()

	})

	cb.logger.Info("Circuit breaker transitioned to half-open")

	if cb.onStateChange != nil {

		go cb.onStateChange(cb.name, oldState, StateHalfOpen)

	}

}

// transitionToClosed transitions circuit to closed state.

func (cb *CircuitBreaker) transitionToClosed() {

	oldState := cb.state

	cb.state = StateClosed

	cb.stateChangeTime = time.Now()

	cb.halfOpenRequests = 0

	cb.failureCount = 0 // Reset counters

	cb.requestCount = 0

	cb.updateMetrics(func(m *CircuitMetrics) {

		m.StateTransitions++

		m.CurrentState = StateClosed.String()

		m.LastStateChange = cb.stateChangeTime

		m.FailureRate = 0

		m.LastUpdated = time.Now()

	})

	cb.logger.Info("Circuit breaker closed - service recovered")

	if cb.onStateChange != nil {

		go cb.onStateChange(cb.name, oldState, StateClosed)

	}

}

// healthCheckRoutine performs periodic health checks.

func (cb *CircuitBreaker) healthCheckRoutine() {

	ticker := time.NewTicker(cb.config.HealthCheckInterval)

	defer ticker.Stop()

	for range ticker.C {

		if cb.getState() == StateOpen {

			if cb.performHealthCheck() {

				cb.mutex.Lock()

				cb.transitionToHalfOpen()

				cb.mutex.Unlock()

			}

		}

	}

}

// performHealthCheck performs a health check operation.

func (cb *CircuitBreaker) performHealthCheck() bool {

	// This is a placeholder - in a real implementation, you would perform.

	// an actual health check against the protected service.

	cb.logger.Debug("Performing health check")

	// Simulate health check.

	ctx, cancel := context.WithTimeout(context.Background(), cb.config.HealthCheckTimeout)

	defer cancel()

	// In a real implementation, this would be an actual health check call.

	select {

	case <-time.After(100 * time.Millisecond): // Simulate quick success

		return true

	case <-ctx.Done():

		return false

	}

}

// getState safely gets the current state.

func (cb *CircuitBreaker) getState() CircuitState {

	cb.mutex.RLock()

	defer cb.mutex.RUnlock()

	return cb.state

}

// GetStats returns current circuit breaker statistics.

func (cb *CircuitBreaker) GetStats() map[string]interface{} {

	cb.mutex.RLock()

	defer cb.mutex.RUnlock()

	return map[string]interface{}{

		"name": cb.name,

		"state": cb.state.String(),

		"failure_count": cb.failureCount,

		"success_count": cb.successCount,

		"request_count": cb.requestCount,

		"failure_rate": cb.metrics.FailureRate,

		"state_change_time": cb.stateChangeTime,

		"last_failure_time": cb.lastFailureTime,

		"last_success_time": cb.lastSuccessTime,

		"half_open_requests": cb.halfOpenRequests,
	}

}

// updateMetrics safely updates metrics.

func (cb *CircuitBreaker) updateMetrics(updater func(*CircuitMetrics)) {

	cb.metrics.mutex.Lock()

	defer cb.metrics.mutex.Unlock()

	updater(cb.metrics)

}

// GetMetrics returns current circuit breaker metrics.

func (cb *CircuitBreaker) GetMetrics() *CircuitMetrics {

	cb.metrics.mutex.RLock()

	defer cb.metrics.mutex.RUnlock()

	// Return a copy without the mutex to avoid copying lock values.

	return &CircuitMetrics{

		TotalRequests: cb.metrics.TotalRequests,

		SuccessfulRequests: cb.metrics.SuccessfulRequests,

		FailedRequests: cb.metrics.FailedRequests,

		RejectedRequests: cb.metrics.RejectedRequests,

		TimeoutRequests: cb.metrics.TimeoutRequests,

		StateTransitions: cb.metrics.StateTransitions,

		CurrentState: cb.metrics.CurrentState,

		LastStateChange: cb.metrics.LastStateChange,

		FailureRate: cb.metrics.FailureRate,

		AverageLatency: cb.metrics.AverageLatency,

		LastUpdated: cb.metrics.LastUpdated,
	}

}

// SetStateChangeCallback sets a callback for state changes.

func (cb *CircuitBreaker) SetStateChangeCallback(callback func(name string, from, to CircuitState)) {

	cb.mutex.Lock()

	defer cb.mutex.Unlock()

	cb.onStateChange = callback

}

// Reset manually resets the circuit breaker to closed state.

func (cb *CircuitBreaker) Reset() {

	cb.mutex.Lock()

	defer cb.mutex.Unlock()

	oldState := cb.state

	cb.state = StateClosed

	cb.stateChangeTime = time.Now()

	cb.failureCount = 0

	cb.successCount = 0

	cb.requestCount = 0

	cb.halfOpenRequests = 0

	cb.updateMetrics(func(m *CircuitMetrics) {

		m.StateTransitions++

		m.CurrentState = StateClosed.String()

		m.LastStateChange = cb.stateChangeTime

		m.FailureRate = 0

		m.LastUpdated = time.Now()

	})

	cb.logger.Info("Circuit breaker manually reset")

	if cb.onStateChange != nil {

		go cb.onStateChange(cb.name, oldState, StateClosed)

	}

}

// ForceOpen manually forces the circuit breaker to open state.

func (cb *CircuitBreaker) ForceOpen() {

	cb.mutex.Lock()

	defer cb.mutex.Unlock()

	if cb.state != StateOpen {

		cb.transitionToOpen()

		cb.logger.Warn("Circuit breaker manually forced open")

	}

}

// IsOpen checks if the circuit breaker is in open state.

func (cb *CircuitBreaker) IsOpen() bool {

	return cb.getState() == StateOpen

}

// IsClosed checks if the circuit breaker is in closed state.

func (cb *CircuitBreaker) IsClosed() bool {

	return cb.getState() == StateClosed

}

// IsHalfOpen checks if the circuit breaker is in half-open state.

func (cb *CircuitBreaker) IsHalfOpen() bool {

	return cb.getState() == StateHalfOpen

}

// CircuitBreakerManager manages multiple circuit breakers.

type CircuitBreakerManager struct {
	circuitBreakers map[string]*CircuitBreaker

	defaultConfig *CircuitBreakerConfig

	logger *slog.Logger

	mutex sync.RWMutex
}

// NewCircuitBreakerManager creates a new circuit breaker manager.

func NewCircuitBreakerManager(defaultConfig *CircuitBreakerConfig) *CircuitBreakerManager {

	if defaultConfig == nil {

		defaultConfig = getDefaultCircuitBreakerConfig()

	}

	return &CircuitBreakerManager{

		circuitBreakers: make(map[string]*CircuitBreaker),

		defaultConfig: defaultConfig,

		logger: slog.Default().With("component", "circuit-breaker-manager"),
	}

}

// GetOrCreate gets an existing circuit breaker or creates a new one.

func (cbm *CircuitBreakerManager) GetOrCreate(name string, config *CircuitBreakerConfig) *CircuitBreaker {

	cbm.mutex.Lock()

	defer cbm.mutex.Unlock()

	if cb, exists := cbm.circuitBreakers[name]; exists {

		return cb

	}

	if config == nil {

		config = cbm.defaultConfig

	}

	cb := NewCircuitBreaker(name, config)

	cbm.circuitBreakers[name] = cb

	cbm.logger.Info("Created new circuit breaker", "name", name)

	return cb

}

// Get gets an existing circuit breaker.

func (cbm *CircuitBreakerManager) Get(name string) (*CircuitBreaker, bool) {

	cbm.mutex.RLock()

	defer cbm.mutex.RUnlock()

	cb, exists := cbm.circuitBreakers[name]

	return cb, exists

}

// Remove removes a circuit breaker.

func (cbm *CircuitBreakerManager) Remove(name string) {

	cbm.mutex.Lock()

	defer cbm.mutex.Unlock()

	delete(cbm.circuitBreakers, name)

	cbm.logger.Info("Removed circuit breaker", "name", name)

}

// List returns all circuit breaker names.

func (cbm *CircuitBreakerManager) List() []string {

	cbm.mutex.RLock()

	defer cbm.mutex.RUnlock()

	names := make([]string, 0, len(cbm.circuitBreakers))

	for name := range cbm.circuitBreakers {

		names = append(names, name)

	}

	return names

}

// GetAllStats returns statistics for all circuit breakers.

func (cbm *CircuitBreakerManager) GetAllStats() map[string]interface{} {

	cbm.mutex.RLock()

	defer cbm.mutex.RUnlock()

	stats := make(map[string]interface{})

	for name, cb := range cbm.circuitBreakers {

		stats[name] = cb.GetStats()

	}

	return stats

}

// GetStats returns statistics for all circuit breakers (interface-compatible method).

func (cbm *CircuitBreakerManager) GetStats() (map[string]interface{}, error) {

	return cbm.GetAllStats(), nil

}

// Shutdown gracefully shuts down all circuit breakers.

func (cbm *CircuitBreakerManager) Shutdown() {

	cbm.mutex.Lock()

	defer cbm.mutex.Unlock()

	for name := range cbm.circuitBreakers {

		cbm.logger.Debug("Shutting down circuit breaker", "name", name)

	}

	// Clear all circuit breakers.

	cbm.circuitBreakers = make(map[string]*CircuitBreaker)

}

// ResetAll resets all circuit breakers.

func (cbm *CircuitBreakerManager) ResetAll() {

	cbm.mutex.RLock()

	circuitBreakers := make([]*CircuitBreaker, 0, len(cbm.circuitBreakers))

	for _, cb := range cbm.circuitBreakers {

		circuitBreakers = append(circuitBreakers, cb)

	}

	cbm.mutex.RUnlock()

	for _, cb := range circuitBreakers {

		cb.Reset()

	}

	cbm.logger.Info("Reset all circuit breakers", "count", len(circuitBreakers))

}
