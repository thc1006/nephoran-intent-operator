package resilience

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"
)

// CircuitBreakerConfig holds configuration for circuit breaker behavior.

type CircuitBreakerConfig struct {

	// Failure threshold - number of failures to trigger open state.

	FailureThreshold int `json:"failure_threshold" yaml:"failure_threshold"`

	// Recovery timeout - how long to wait before transitioning from open to half-open.

	RecoveryTimeout time.Duration `json:"recovery_timeout" yaml:"recovery_timeout"`

	// Success threshold - number of successes needed to close circuit from half-open.

	SuccessThreshold int `json:"success_threshold" yaml:"success_threshold"`

	// Request timeout for individual operations.

	RequestTimeout time.Duration `json:"request_timeout" yaml:"request_timeout"`

	// Maximum requests allowed in half-open state.

	HalfOpenMaxRequests int `json:"half_open_max_requests" yaml:"half_open_max_requests"`

	// Minimum requests before failure rate calculation.

	MinimumRequests int `json:"minimum_requests" yaml:"minimum_requests"`

	// Failure rate threshold (0.0 to 1.0).

	FailureRate float64 `json:"failure_rate" yaml:"failure_rate"`
}

// DefaultCircuitBreakerConfig returns the default configuration for circuit breaker.

func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {

	return &CircuitBreakerConfig{

		FailureThreshold: 5,

		RecoveryTimeout: 60 * time.Second,

		SuccessThreshold: 3,

		RequestTimeout: 30 * time.Second,

		HalfOpenMaxRequests: 5,

		MinimumRequests: 10,

		FailureRate: 0.5,
	}

}

// CircuitBreakerState represents the state of the circuit breaker.

type CircuitBreakerState string

const (

	// StateClosed holds stateclosed value.

	StateClosed CircuitBreakerState = "closed"

	// StateOpen holds stateopen value.

	StateOpen CircuitBreakerState = "open"

	// StateHalfOpen holds statehalfopen value.

	StateHalfOpen CircuitBreakerState = "half-open"
)

// CircuitBreakerMetrics holds metrics for circuit breaker monitoring.

type CircuitBreakerMetrics struct {
	ServiceName string `json:"service_name"`

	State CircuitBreakerState `json:"state"`

	TotalRequests int64 `json:"total_requests"`

	SuccessfulRequests int64 `json:"successful_requests"`

	FailedRequests int64 `json:"failed_requests"`

	RejectedRequests int64 `json:"rejected_requests"`

	FailureRate float64 `json:"failure_rate"`

	LastStateChange time.Time `json:"last_state_change"`

	AverageLatency time.Duration `json:"average_latency"`
}

// LLMCircuitBreaker wraps the existing LLM circuit breaker with a streamlined interface.

type LLMCircuitBreaker struct {
	name string

	circuitBreaker *llm.CircuitBreaker

	config *CircuitBreakerConfig

	metricsCollector *monitoring.MetricsCollector

	mutex sync.RWMutex
}

// NewLLMCircuitBreaker creates a new circuit breaker for LLM operations.

func NewLLMCircuitBreaker(name string, config *CircuitBreakerConfig, metricsCollector *monitoring.MetricsCollector) *LLMCircuitBreaker {

	if config == nil {

		config = DefaultCircuitBreakerConfig()

	}

	// Convert to LLM circuit breaker config.

	llmConfig := &llm.CircuitBreakerConfig{

		FailureThreshold: int64(config.FailureThreshold),

		FailureRate: config.FailureRate,

		MinimumRequestCount: int64(config.MinimumRequests),

		Timeout: config.RequestTimeout,

		HalfOpenTimeout: config.RecoveryTimeout,

		SuccessThreshold: int64(config.SuccessThreshold),

		HalfOpenMaxRequests: int64(config.HalfOpenMaxRequests),

		ResetTimeout: config.RecoveryTimeout,

		SlidingWindowSize: 100,

		EnableHealthCheck: false,

		HealthCheckInterval: 30 * time.Second,

		HealthCheckTimeout: 10 * time.Second,
	}

	llmCB := llm.NewCircuitBreaker(name, llmConfig)

	return &LLMCircuitBreaker{

		name: name,

		circuitBreaker: llmCB,

		config: config,

		metricsCollector: metricsCollector,
	}

}

// Execute executes an operation through the circuit breaker with timeout and fallback.

func (cb *LLMCircuitBreaker) Execute(ctx context.Context, operation func(context.Context) (interface{}, error)) (interface{}, error) {

	// Create a timeout context.

	timeoutCtx, cancel := context.WithTimeout(ctx, cb.config.RequestTimeout)

	defer cancel()

	// Wrap the operation to match the circuit breaker interface.

	circuitOperation := func(ctx context.Context) (interface{}, error) {

		return operation(ctx)

	}

	// Execute through circuit breaker.

	result, err := cb.circuitBreaker.Execute(timeoutCtx, circuitOperation)

	// Update metrics.

	cb.updateMetrics(err != nil, time.Since(time.Now()))

	return result, err

}

// ExecuteWithFallback executes an operation with a fallback function when circuit is open.

func (cb *LLMCircuitBreaker) ExecuteWithFallback(

	ctx context.Context,

	operation func(context.Context) (interface{}, error),

	fallback func(context.Context, error) (interface{}, error),

) (interface{}, error) {

	result, err := cb.Execute(ctx, operation)

	// If circuit breaker rejected the request, use fallback.

	if err != nil && cb.IsOpen() {

		if fallback != nil {

			return fallback(ctx, err)

		}

		return nil, fmt.Errorf("circuit breaker is open and no fallback provided: %w", err)

	}

	return result, err

}

// IsOpen returns true if the circuit breaker is in open state.

func (cb *LLMCircuitBreaker) IsOpen() bool {

	return cb.circuitBreaker.IsOpen()

}

// IsClosed returns true if the circuit breaker is in closed state.

func (cb *LLMCircuitBreaker) IsClosed() bool {

	return cb.circuitBreaker.IsClosed()

}

// IsHalfOpen returns true if the circuit breaker is in half-open state.

func (cb *LLMCircuitBreaker) IsHalfOpen() bool {

	return cb.circuitBreaker.IsHalfOpen()

}

// GetState returns the current state of the circuit breaker.

func (cb *LLMCircuitBreaker) GetState() CircuitBreakerState {

	if cb.circuitBreaker.IsOpen() {

		return StateOpen

	} else if cb.circuitBreaker.IsHalfOpen() {

		return StateHalfOpen

	}

	return StateClosed

}

// GetMetrics returns current circuit breaker metrics.

func (cb *LLMCircuitBreaker) GetMetrics() CircuitBreakerMetrics {

	cb.mutex.RLock()

	defer cb.mutex.RUnlock()

	llmMetrics := cb.circuitBreaker.GetMetrics()

	return CircuitBreakerMetrics{

		ServiceName: cb.name,

		State: cb.GetState(),

		TotalRequests: llmMetrics.TotalRequests,

		SuccessfulRequests: llmMetrics.SuccessfulRequests,

		FailedRequests: llmMetrics.FailedRequests,

		RejectedRequests: llmMetrics.RejectedRequests,

		FailureRate: llmMetrics.FailureRate,

		LastStateChange: llmMetrics.LastStateChange,

		AverageLatency: llmMetrics.AverageLatency,
	}

}

// Reset manually resets the circuit breaker to closed state.

func (cb *LLMCircuitBreaker) Reset() {

	cb.circuitBreaker.Reset()

}

// ForceOpen manually forces the circuit breaker to open state.

func (cb *LLMCircuitBreaker) ForceOpen() {

	cb.circuitBreaker.ForceOpen()

}

// updateMetrics updates internal metrics and publishes to metrics collector.

func (cb *LLMCircuitBreaker) updateMetrics(failed bool, latency time.Duration) {

	if cb.metricsCollector == nil {

		return

	}

	// Update Prometheus metrics.

	_ = cb.name // avoid unused variable

	if failed {

		cb.metricsCollector.RecordLLMRequestError(cb.name, "circuit_breaker_failure")

	} else {

		cb.metricsCollector.RecordLLMRequest(cb.name, "success", latency, 0)

	}

	// Record circuit breaker state.

	stateValue := 0.0

	switch cb.GetState() {

	case StateClosed:

		stateValue = 0.0

	case StateHalfOpen:

		stateValue = 0.5

	case StateOpen:

		stateValue = 1.0

	}

	// Use a gauge metric for circuit breaker state.

	if gauge := cb.metricsCollector.GetGauge("circuit_breaker_state"); gauge != nil {

		gauge.Set(stateValue)

	}

}

// CircuitBreakerManager manages multiple circuit breakers for different services.

type CircuitBreakerManager struct {
	circuitBreakers map[string]*LLMCircuitBreaker

	defaultConfig *CircuitBreakerConfig

	metricsCollector *monitoring.MetricsCollector

	mutex sync.RWMutex
}

// NewCircuitBreakerManager creates a new circuit breaker manager.

func NewCircuitBreakerManager(defaultConfig *CircuitBreakerConfig, metricsCollector *monitoring.MetricsCollector) *CircuitBreakerManager {

	if defaultConfig == nil {

		defaultConfig = DefaultCircuitBreakerConfig()

	}

	return &CircuitBreakerManager{

		circuitBreakers: make(map[string]*LLMCircuitBreaker),

		defaultConfig: defaultConfig,

		metricsCollector: metricsCollector,
	}

}

// GetOrCreateCircuitBreaker gets an existing circuit breaker or creates a new one.

func (cbm *CircuitBreakerManager) GetOrCreateCircuitBreaker(serviceName string, config *CircuitBreakerConfig) *LLMCircuitBreaker {

	cbm.mutex.Lock()

	defer cbm.mutex.Unlock()

	if cb, exists := cbm.circuitBreakers[serviceName]; exists {

		return cb

	}

	if config == nil {

		config = cbm.defaultConfig

	}

	cb := NewLLMCircuitBreaker(serviceName, config, cbm.metricsCollector)

	cbm.circuitBreakers[serviceName] = cb

	return cb

}

// GetCircuitBreaker gets an existing circuit breaker by name.

func (cbm *CircuitBreakerManager) GetCircuitBreaker(serviceName string) (*LLMCircuitBreaker, bool) {

	cbm.mutex.RLock()

	defer cbm.mutex.RUnlock()

	cb, exists := cbm.circuitBreakers[serviceName]

	return cb, exists

}

// GetAllMetrics returns metrics for all circuit breakers.

func (cbm *CircuitBreakerManager) GetAllMetrics() map[string]CircuitBreakerMetrics {

	cbm.mutex.RLock()

	defer cbm.mutex.RUnlock()

	metrics := make(map[string]CircuitBreakerMetrics)

	for name, cb := range cbm.circuitBreakers {

		metrics[name] = cb.GetMetrics()

	}

	return metrics

}

// GetStats returns statistics for all circuit breakers (interface-compatible method).

func (cbm *CircuitBreakerManager) GetStats() (map[string]interface{}, error) {

	metrics := cbm.GetAllMetrics()

	stats := make(map[string]interface{})

	for name, metric := range metrics {

		stats[name] = metric

	}

	return stats, nil

}

// ResetAll resets all circuit breakers.

func (cbm *CircuitBreakerManager) ResetAll() {

	cbm.mutex.RLock()

	defer cbm.mutex.RUnlock()

	for _, cb := range cbm.circuitBreakers {

		cb.Reset()

	}

}

// ListCircuitBreakers returns the names of all circuit breakers.

func (cbm *CircuitBreakerManager) ListCircuitBreakers() []string {

	cbm.mutex.RLock()

	defer cbm.mutex.RUnlock()

	names := make([]string, 0, len(cbm.circuitBreakers))

	for name := range cbm.circuitBreakers {

		names = append(names, name)

	}

	return names

}
