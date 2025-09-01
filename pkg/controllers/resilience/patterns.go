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

package resiliencecontroller

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"

	"github.com/thc1006/nephoran-intent-operator/pkg/errors"
)

// ResilienceManager manages all resilience patterns for the system.

type ResilienceManager struct {

	// Pattern implementations.

	timeoutManager *TimeoutManager

	bulkheadManager *BulkheadManager

	circuitBreakerMgr *CircuitBreakerManager

	rateLimiterMgr *RateLimiterManager

	retryManager *RetryManager

	healthCheckManager *HealthCheckManager

	gracefulShutdown *GracefulShutdownManager

	// Configuration and state.

	config *ResilienceConfig

	logger logr.Logger

	metrics *ResilienceMetrics

	// Lifecycle.

	mutex sync.RWMutex

	started bool

	stopChan chan struct{}

	healthTicker *time.Ticker
}

// ResilienceConfig holds configuration for all resilience patterns.

type ResilienceConfig struct {

	// Global settings.

	DefaultTimeout time.Duration `json:"defaultTimeout"`

	MaxConcurrentOperations int `json:"maxConcurrentOperations"`

	HealthCheckInterval time.Duration `json:"healthCheckInterval"`

	// Timeout settings.

	TimeoutEnabled bool `json:"timeoutEnabled"`

	TimeoutConfigs map[string]*TimeoutConfig `json:"timeoutConfigs"`

	// Bulkhead settings.

	BulkheadEnabled bool `json:"bulkheadEnabled"`

	BulkheadConfigs map[string]*BulkheadConfig `json:"bulkheadConfigs"`

	// Circuit breaker settings.

	CircuitBreakerEnabled bool `json:"circuitBreakerEnabled"`

	CircuitBreakerConfigs map[string]*CircuitBreakerConfig `json:"circuitBreakerConfigs"`

	// Rate limiting settings.

	RateLimitingEnabled bool `json:"rateLimitingEnabled"`

	RateLimitConfigs map[string]*RateLimitConfig `json:"rateLimitConfigs"`

	// Retry settings.

	RetryEnabled bool `json:"retryEnabled"`

	RetryConfigs map[string]*RetryConfig `json:"retryConfigs"`

	// Health check settings.

	HealthCheckEnabled bool `json:"healthCheckEnabled"`

	HealthCheckConfigs map[string]*HealthCheckConfig `json:"healthCheckConfigs"`

	// Graceful shutdown settings.

	GracefulShutdownEnabled bool `json:"gracefulShutdownEnabled"`

	ShutdownTimeout time.Duration `json:"shutdownTimeout"`

	ShutdownHooks []string `json:"shutdownHooks"`
}

// CircuitBreakerMetrics tracks circuit breaker pattern metrics.

type CircuitBreakerMetrics struct {
	TotalRequests int64 `json:"totalRequests"`

	FailedRequests int64 `json:"failedRequests"`

	SuccessRate float64 `json:"successRate"`

	State string `json:"state"`

	StateChanges int64 `json:"stateChanges"`

	LastFailure time.Time `json:"lastFailure"`
}

// ResilienceMetrics tracks metrics for all resilience patterns.

type ResilienceMetrics struct {

	// Overall metrics.

	TotalOperations int64 `json:"totalOperations"`

	SuccessfulOperations int64 `json:"successfulOperations"`

	FailedOperations int64 `json:"failedOperations"`

	AverageLatency time.Duration `json:"averageLatency"`

	// Pattern-specific metrics.

	TimeoutMetrics *TimeoutMetrics `json:"timeoutMetrics"`

	BulkheadMetrics *BulkheadMetrics `json:"bulkheadMetrics"`

	CircuitBreakerMetrics *CircuitBreakerMetrics `json:"circuitBreakerMetrics"`

	RateLimitMetrics *RateLimitMetrics `json:"rateLimitMetrics"`

	RetryMetrics *RetryMetrics `json:"retryMetrics"`

	HealthCheckMetrics *HealthCheckMetrics `json:"healthCheckMetrics"`

	// Resource metrics.

	ResourceMetrics *ResourceMetrics `json:"resourceMetrics"`

	LastUpdated time.Time `json:"lastUpdated"`

	mutex sync.RWMutex
}

// TimeoutManager manages timeout patterns.

type TimeoutManager struct {
	configs map[string]*TimeoutConfig

	operations sync.Map // map[string]*TimeoutOperation

	metrics *TimeoutMetrics

	logger logr.Logger

	mutex sync.RWMutex
}

// TimeoutConfig configures timeout behavior.

type TimeoutConfig struct {
	Name string `json:"name"`

	DefaultTimeout time.Duration `json:"defaultTimeout"`

	MaxTimeout time.Duration `json:"maxTimeout"`

	MinTimeout time.Duration `json:"minTimeout"`

	TimeoutGradient bool `json:"timeoutGradient"` // Adjust timeout based on load

	AdaptiveTimeout bool `json:"adaptiveTimeout"` // Learn optimal timeout

}

// TimeoutOperation tracks an operation with timeout.

type TimeoutOperation struct {
	ID string

	StartTime time.Time

	Timeout time.Duration

	Context context.Context

	CancelFunc context.CancelFunc

	CompletedChan chan struct{}

	TimeoutChan chan struct{}

	mutex sync.RWMutex
}

// TimeoutMetrics tracks timeout-related metrics.

type TimeoutMetrics struct {
	TotalOperations int64 `json:"totalOperations"`

	TimeoutOperations int64 `json:"timeoutOperations"`

	AverageTimeout time.Duration `json:"averageTimeout"`

	TimeoutRate float64 `json:"timeoutRate"`

	AdaptiveAdjustments int64 `json:"adaptiveAdjustments"`

	OperationsByTimeout map[time.Duration]int64 `json:"operationsByTimeout"`

	mutex sync.RWMutex
}

// BulkheadManager manages bulkhead isolation patterns.

type BulkheadManager struct {
	bulkheads map[string]*Bulkhead

	metrics *BulkheadMetrics

	logger logr.Logger

	mutex sync.RWMutex
}

// BulkheadConfig configures bulkhead behavior.

type BulkheadConfig struct {
	Name string `json:"name"`

	MaxConcurrent int `json:"maxConcurrent"`

	QueueSize int `json:"queueSize"`

	Timeout time.Duration `json:"timeout"`

	RejectWhenFull bool `json:"rejectWhenFull"`

	MonitoringEnabled bool `json:"monitoringEnabled"`

	AutoScaling bool `json:"autoScaling"`

	ScalingThreshold float64 `json:"scalingThreshold"`
}

// Bulkhead implements resource isolation.

type Bulkhead struct {
	config *BulkheadConfig

	semaphore chan struct{}

	queue chan *BulkheadRequest

	activeRequests int64

	queuedRequests int64

	rejectedRequests int64

	completedRequests int64

	// Auto-scaling.

	loadMetrics *LoadMetrics

	scalingDecision chan ScalingDecision

	logger logr.Logger

	stopChan chan struct{}

	mutex sync.RWMutex
}

// BulkheadRequest represents a request to be processed in a bulkhead.

type BulkheadRequest struct {
	ID string

	Operation func(ctx context.Context) (interface{}, error)

	Context context.Context

	Priority int

	Timeout time.Duration

	ResultChan chan *BulkheadResult

	CreatedAt time.Time
}

// BulkheadResult represents the result of a bulkhead operation.

type BulkheadResult struct {
	Success bool

	Result interface{}

	Error error

	Duration time.Duration

	QueueTime time.Duration

	ExecuteTime time.Duration
}

// LoadMetrics tracks load for auto-scaling decisions.

type LoadMetrics struct {
	RequestRate float64

	ResponseTime time.Duration

	QueueUtilization float64

	ErrorRate float64

	ResourceUsage float64

	History []LoadMeasurement

	mutex sync.RWMutex
}

// LoadMeasurement represents a point-in-time load measurement.

type LoadMeasurement struct {
	Timestamp time.Time

	RequestRate float64

	ResponseTime time.Duration

	QueueUsage float64
}

// ScalingDecision represents an auto-scaling decision.

type ScalingDecision struct {
	Action ScalingAction

	TargetSize int

	Reason string

	Confidence float64

	Timestamp time.Time
}

// ScalingAction defines auto-scaling actions.

type ScalingAction string

const (

	// ScaleUp holds scaleup value.

	ScaleUp ScalingAction = "scale_up"

	// ScaleDown holds scaledown value.

	ScaleDown ScalingAction = "scale_down"

	// NoScale holds noscale value.

	NoScale ScalingAction = "no_scale"
)

// BulkheadMetrics tracks bulkhead-related metrics.

type BulkheadMetrics struct {
	TotalBulkheads int `json:"totalBulkheads"`

	ActiveRequests int64 `json:"activeRequests"`

	QueuedRequests int64 `json:"queuedRequests"`

	RejectedRequests int64 `json:"rejectedRequests"`

	CompletedRequests int64 `json:"completedRequests"`

	AverageQueueTime time.Duration `json:"averageQueueTime"`

	AverageExecuteTime time.Duration `json:"averageExecuteTime"`

	BulkheadDetails map[string]*BulkheadDetail `json:"bulkheadDetails"`

	mutex sync.RWMutex
}

// BulkheadDetail tracks detailed metrics for a specific bulkhead.

type BulkheadDetail struct {
	Name string `json:"name"`

	MaxConcurrent int `json:"maxConcurrent"`

	CurrentActive int64 `json:"currentActive"`

	CurrentQueued int64 `json:"currentQueued"`

	TotalProcessed int64 `json:"totalProcessed"`

	TotalRejected int64 `json:"totalRejected"`

	AverageLatency time.Duration `json:"averageLatency"`

	SuccessRate float64 `json:"successRate"`

	LastScalingAction *ScalingDecision `json:"lastScalingAction"`

	HealthStatus string `json:"healthStatus"`
}

// CircuitBreakerManager manages circuit breaker patterns.

type CircuitBreakerManager struct {
	circuitBreakers map[string]*AdvancedCircuitBreaker

	metrics *CircuitBreakerMetrics

	logger logr.Logger

	mutex sync.RWMutex
}

// CircuitBreakerConfig configures circuit breaker behavior.

type CircuitBreakerConfig struct {
	Name string `json:"name"`

	FailureThreshold int `json:"failureThreshold"`

	SuccessThreshold int `json:"successThreshold"`

	Timeout time.Duration `json:"timeout"`

	MaxHalfOpenRequests int `json:"maxHalfOpenRequests"`

	// Advanced features.

	AdaptiveThreshold bool `json:"adaptiveThreshold"`

	FailureRateThreshold float64 `json:"failureRateThreshold"`

	SlidingWindowSize time.Duration `json:"slidingWindowSize"`

	MinimumRequestThreshold int `json:"minimumRequestThreshold"`

	// Monitoring.

	MetricsEnabled bool `json:"metricsEnabled"`

	AlertingEnabled bool `json:"alertingEnabled"`

	HealthCheckEnabled bool `json:"healthCheckEnabled"`
}

// AdvancedCircuitBreaker implements advanced circuit breaker pattern.

type AdvancedCircuitBreaker struct {
	config *CircuitBreakerConfig

	state CircuitState

	lastStateChange time.Time

	// Counters.

	totalRequests int64

	successfulRequests int64

	failedRequests int64

	halfOpenRequests int64

	// Sliding window for failure rate calculation.

	requestWindow *SlidingWindow

	failureWindow *SlidingWindow

	// Adaptive threshold.

	adaptiveThreshold *AdaptiveThreshold

	// Health check.

	healthChecker HealthChecker

	// Callbacks.

	onStateChange func(from, to CircuitState)

	onOpen func(metrics CircuitMetrics)

	onHalfOpen func()

	onClose func()

	logger logr.Logger

	mutex sync.RWMutex
}

// CircuitState represents the state of a circuit breaker.

type CircuitState int

const (

	// CircuitClosed holds circuitclosed value.

	CircuitClosed CircuitState = iota

	// CircuitOpen holds circuitopen value.

	CircuitOpen

	// CircuitHalfOpen holds circuithalfopen value.

	CircuitHalfOpen
)

// String returns string representation of circuit state.

func (cs CircuitState) String() string {

	switch cs {

	case CircuitClosed:

		return "closed"

	case CircuitOpen:

		return "open"

	case CircuitHalfOpen:

		return "half_open"

	default:

		return "unknown"

	}

}

// SlidingWindow implements a time-based sliding window.

type SlidingWindow struct {
	windowSize time.Duration

	buckets []WindowBucket

	bucketSize time.Duration

	current int

	mutex sync.RWMutex
}

// WindowBucket represents a bucket in the sliding window.

type WindowBucket struct {
	Timestamp time.Time

	Count int64

	Success int64

	Failure int64
}

// AdaptiveThreshold dynamically adjusts circuit breaker thresholds.

type AdaptiveThreshold struct {
	baseThreshold int

	currentThreshold int

	adjustmentFactor float64

	learningRate float64

	recentPerformance []float64

	adjustmentHistory []ThresholdAdjustment

	mutex sync.RWMutex
}

// ThresholdAdjustment tracks threshold adjustments.

type ThresholdAdjustment struct {
	Timestamp time.Time

	OldThreshold int

	NewThreshold int

	Reason string

	PerformanceData map[string]float64
}

// CircuitMetrics provides metrics about circuit breaker state.

type CircuitMetrics struct {
	State CircuitState `json:"state"`

	TotalRequests int64 `json:"totalRequests"`

	SuccessfulRequests int64 `json:"successfulRequests"`

	FailedRequests int64 `json:"failedRequests"`

	FailureRate float64 `json:"failureRate"`

	LastStateChange time.Time `json:"lastStateChange"`

	TimeInState time.Duration `json:"timeInState"`

	AdaptiveThreshold int `json:"adaptiveThreshold"`
}

// HealthChecker defines interface for health checking.

type HealthChecker interface {
	CheckHealth(ctx context.Context) error

	IsHealthy() bool

	GetHealthMetrics() map[string]interface{}
}

// DefaultHealthChecker provides basic health checking.

type DefaultHealthChecker struct {
	checkFunc func(ctx context.Context) error

	lastCheckTime time.Time

	lastCheckResult error

	isHealthy bool

	mutex sync.RWMutex
}

// RateLimiterManager manages rate limiting patterns.

type RateLimiterManager struct {
	limiters map[string]*AdaptiveRateLimiter

	metrics *RateLimitMetrics

	logger logr.Logger

	mutex sync.RWMutex
}

// RateLimitConfig configures rate limiting behavior.

type RateLimitConfig struct {
	Name string `json:"name"`

	RequestsPerSecond float64 `json:"requestsPerSecond"`

	BurstSize int `json:"burstSize"`

	// Adaptive features.

	AdaptiveEnabled bool `json:"adaptiveEnabled"`

	MaxRate float64 `json:"maxRate"`

	MinRate float64 `json:"minRate"`

	AdjustmentWindow time.Duration `json:"adjustmentWindow"`

	// Monitoring.

	MonitoringEnabled bool `json:"monitoringEnabled"`

	AlertingEnabled bool `json:"alertingEnabled"`
}

// AdaptiveRateLimiter implements adaptive rate limiting.

type AdaptiveRateLimiter struct {
	config *RateLimitConfig

	tokenBucket *TokenBucket

	adaptiveEngine *RateAdaptationEngine

	metrics *RateLimiterMetrics

	logger logr.Logger

	mutex sync.RWMutex
}

// TokenBucket implements token bucket algorithm.

type TokenBucket struct {
	capacity int

	tokens int

	refillRate float64

	lastRefillTime time.Time

	mutex sync.Mutex
}

// RateAdaptationEngine adapts rate limits based on system performance.

type RateAdaptationEngine struct {
	currentRate float64

	targetLatency time.Duration

	performanceWindow *PerformanceWindow

	adjustmentHistory []RateAdjustment

	mutex sync.RWMutex
}

// PerformanceWindow tracks recent performance metrics.

type PerformanceWindow struct {
	measurements []PerformanceMeasurement

	windowSize time.Duration

	maxSize int

	mutex sync.RWMutex
}

// PerformanceMeasurement represents a performance measurement.

type PerformanceMeasurement struct {
	Timestamp time.Time

	Latency time.Duration

	SuccessRate float64

	ErrorRate float64

	QueueDepth int

	ResourceUsage map[string]float64
}

// RateAdjustment tracks rate limit adjustments.

type RateAdjustment struct {
	Timestamp time.Time

	OldRate float64

	NewRate float64

	Reason string

	Performance PerformanceMeasurement
}

// NewResilienceManager creates a new resilience manager.

func NewResilienceManager(config *ResilienceConfig, logger logr.Logger) *ResilienceManager {

	if config == nil {

		config = getDefaultResilienceConfig()

	}

	rm := &ResilienceManager{

		config: config,

		logger: logger.WithName("resilience-manager"),

		metrics: NewResilienceMetrics(),

		stopChan: make(chan struct{}),

		healthTicker: time.NewTicker(config.HealthCheckInterval),
	}

	// Initialize pattern managers.

	if config.TimeoutEnabled {

		rm.timeoutManager = NewTimeoutManager(config.TimeoutConfigs, logger)

	}

	if config.BulkheadEnabled {

		rm.bulkheadManager = NewBulkheadManager(config.BulkheadConfigs, logger)

	}

	if config.CircuitBreakerEnabled {

		rm.circuitBreakerMgr = NewCircuitBreakerManager(config.CircuitBreakerConfigs, logger)

	}

	if config.RateLimitingEnabled {

		rm.rateLimiterMgr = NewRateLimiterManager(config.RateLimitConfigs, logger)

	}

	if config.RetryEnabled {

		rm.retryManager = NewRetryManager(config.RetryConfigs, logger)

	}

	if config.HealthCheckEnabled {

		rm.healthCheckManager = NewHealthCheckManager(config.HealthCheckConfigs, logger)

	}

	if config.GracefulShutdownEnabled {

		rm.gracefulShutdown = NewGracefulShutdownManager(config.ShutdownTimeout, logger)

	}

	return rm

}

// Start starts the resilience manager and all pattern managers.

func (rm *ResilienceManager) Start(ctx context.Context) error {

	rm.mutex.Lock()

	defer rm.mutex.Unlock()

	if rm.started {

		return fmt.Errorf("resilience manager already started")

	}

	rm.logger.Info("Starting resilience manager")

	// Start pattern managers.

	if rm.timeoutManager != nil {

		go rm.timeoutManager.Start(ctx)

	}

	if rm.bulkheadManager != nil {

		go rm.bulkheadManager.Start(ctx)

	}

	if rm.circuitBreakerMgr != nil {

		go rm.circuitBreakerMgr.Start(ctx)

	}

	if rm.rateLimiterMgr != nil {

		go rm.rateLimiterMgr.Start(ctx)

	}

	if rm.retryManager != nil {

		go rm.retryManager.Start(ctx)

	}

	if rm.healthCheckManager != nil {

		go rm.healthCheckManager.Start(ctx)

	}

	if rm.gracefulShutdown != nil {

		go rm.gracefulShutdown.Start(ctx)

	}

	// Start monitoring.

	go rm.monitor(ctx)

	rm.started = true

	rm.logger.Info("Resilience manager started successfully")

	return nil

}

// Stop stops the resilience manager.

func (rm *ResilienceManager) Stop() error {

	rm.mutex.Lock()

	defer rm.mutex.Unlock()

	if !rm.started {

		return nil

	}

	rm.logger.Info("Stopping resilience manager")

	// Signal stop.

	close(rm.stopChan)

	rm.healthTicker.Stop()

	// Stop pattern managers.

	if rm.timeoutManager != nil {

		rm.timeoutManager.Stop()

	}

	if rm.bulkheadManager != nil {

		rm.bulkheadManager.Stop()

	}

	if rm.circuitBreakerMgr != nil {

		rm.circuitBreakerMgr.Stop()

	}

	if rm.rateLimiterMgr != nil {

		rm.rateLimiterMgr.Stop()

	}

	if rm.retryManager != nil {

		rm.retryManager.Stop()

	}

	if rm.healthCheckManager != nil {

		rm.healthCheckManager.Stop()

	}

	if rm.gracefulShutdown != nil {

		rm.gracefulShutdown.Stop()

	}

	rm.started = false

	rm.logger.Info("Resilience manager stopped")

	return nil

}

// ExecuteWithResilience executes an operation with all configured resilience patterns.

func (rm *ResilienceManager) ExecuteWithResilience(ctx context.Context, operationName string, operation func(ctx context.Context) (interface{}, error)) (interface{}, error) {

	startTime := time.Now()

	// Apply timeout.

	if rm.timeoutManager != nil {

		ctx = rm.timeoutManager.ApplyTimeout(ctx, operationName)

	}

	// Check rate limit.

	if rm.rateLimiterMgr != nil {

		if !rm.rateLimiterMgr.Allow(operationName) {

			rm.updateMetrics(operationName, false, time.Since(startTime), "rate_limited")

			return nil, errors.NewProcessingError("Operation rate limited", errors.CategoryRateLimit)

		}

	}

	// Check circuit breaker.

	if rm.circuitBreakerMgr != nil {

		if !rm.circuitBreakerMgr.CanExecute(operationName) {

			rm.updateMetrics(operationName, false, time.Since(startTime), "circuit_breaker_open")

			return nil, errors.NewProcessingError("Circuit breaker is open", errors.CategoryCircuitBreaker)

		}

	}

	// Execute with bulkhead if configured.

	if rm.bulkheadManager != nil {

		return rm.bulkheadManager.Execute(ctx, operationName, operation)

	}

	// Execute directly.

	result, err := operation(ctx)

	// Update metrics.

	success := err == nil

	duration := time.Since(startTime)

	rm.updateMetrics(operationName, success, duration, "")

	// Update circuit breaker.

	if rm.circuitBreakerMgr != nil {

		rm.circuitBreakerMgr.RecordResult(operationName, success, duration)

	}

	return result, err

}

// monitor performs periodic monitoring and maintenance.

func (rm *ResilienceManager) monitor(ctx context.Context) {

	for {

		select {

		case <-rm.healthTicker.C:

			rm.performHealthCheck()

			rm.collectMetrics()

		case <-rm.stopChan:

			return

		case <-ctx.Done():

			return

		}

	}

}

// performHealthCheck performs comprehensive health checking.

func (rm *ResilienceManager) performHealthCheck() {

	// Check all pattern managers.

	if rm.healthCheckManager != nil {

		rm.healthCheckManager.PerformHealthChecks()

	}

	// Check circuit breakers.

	if rm.circuitBreakerMgr != nil {

		rm.circuitBreakerMgr.CheckHealth()

	}

	// Check bulkheads.

	if rm.bulkheadManager != nil {

		rm.bulkheadManager.CheckHealth()

	}

	// Check rate limiters.

	if rm.rateLimiterMgr != nil {

		rm.rateLimiterMgr.CheckHealth()

	}

}

// collectMetrics collects metrics from all pattern managers.

func (rm *ResilienceManager) collectMetrics() {

	rm.metrics.mutex.Lock()

	defer rm.metrics.mutex.Unlock()

	// Collect timeout metrics.

	if rm.timeoutManager != nil {

		rm.metrics.TimeoutMetrics = rm.timeoutManager.GetMetrics()

	}

	// Collect bulkhead metrics.

	if rm.bulkheadManager != nil {

		rm.metrics.BulkheadMetrics = rm.bulkheadManager.GetMetrics()

	}

	// Collect circuit breaker metrics.

	if rm.circuitBreakerMgr != nil {

		rm.metrics.CircuitBreakerMetrics = rm.circuitBreakerMgr.GetMetrics()

	}

	// Collect rate limit metrics.

	if rm.rateLimiterMgr != nil {

		rm.metrics.RateLimitMetrics = rm.rateLimiterMgr.GetMetrics()

	}

	// Collect retry metrics.

	if rm.retryManager != nil {

		rm.metrics.RetryMetrics = rm.retryManager.GetMetrics()

	}

	// Collect health check metrics.

	if rm.healthCheckManager != nil {

		rm.metrics.HealthCheckMetrics = rm.healthCheckManager.GetMetrics()

	}

	rm.metrics.LastUpdated = time.Now()

}

// updateMetrics updates overall resilience metrics.

func (rm *ResilienceManager) updateMetrics(operationName string, success bool, duration time.Duration, reason string) {

	rm.metrics.mutex.Lock()

	defer rm.metrics.mutex.Unlock()

	atomic.AddInt64(&rm.metrics.TotalOperations, 1)

	if success {

		atomic.AddInt64(&rm.metrics.SuccessfulOperations, 1)

	} else {

		atomic.AddInt64(&rm.metrics.FailedOperations, 1)

	}

	// Update average latency.

	if rm.metrics.TotalOperations > 0 {

		totalLatency := time.Duration(rm.metrics.TotalOperations-1) * rm.metrics.AverageLatency

		totalLatency += duration

		rm.metrics.AverageLatency = totalLatency / time.Duration(rm.metrics.TotalOperations)

	}

}

// GetMetrics returns comprehensive resilience metrics.

func (rm *ResilienceManager) GetMetrics() *ResilienceMetrics {

	rm.metrics.mutex.RLock()

	defer rm.metrics.mutex.RUnlock()

	// Return a copy to prevent concurrent access issues.

	metricsCopy := &ResilienceMetrics{

		TotalOperations: rm.metrics.TotalOperations,

		SuccessfulOperations: rm.metrics.SuccessfulOperations,

		FailedOperations: rm.metrics.FailedOperations,

		AverageLatency: rm.metrics.AverageLatency,

		TimeoutMetrics: rm.metrics.TimeoutMetrics,

		BulkheadMetrics: rm.metrics.BulkheadMetrics,

		CircuitBreakerMetrics: rm.metrics.CircuitBreakerMetrics,

		RateLimitMetrics: rm.metrics.RateLimitMetrics,

		RetryMetrics: rm.metrics.RetryMetrics,

		HealthCheckMetrics: rm.metrics.HealthCheckMetrics,

		ResourceMetrics: rm.metrics.ResourceMetrics,

		LastUpdated: rm.metrics.LastUpdated,
	}

	return metricsCopy

}

// Helper functions and supporting implementations.

// getDefaultResilienceConfig returns default resilience configuration.

func getDefaultResilienceConfig() *ResilienceConfig {

	return &ResilienceConfig{

		DefaultTimeout: 30 * time.Second,

		MaxConcurrentOperations: 100,

		HealthCheckInterval: 30 * time.Second,

		TimeoutEnabled: true,

		TimeoutConfigs: make(map[string]*TimeoutConfig),

		BulkheadEnabled: true,

		BulkheadConfigs: make(map[string]*BulkheadConfig),

		CircuitBreakerEnabled: true,

		CircuitBreakerConfigs: make(map[string]*CircuitBreakerConfig),

		RateLimitingEnabled: true,

		RateLimitConfigs: make(map[string]*RateLimitConfig),

		RetryEnabled: true,

		RetryConfigs: make(map[string]*RetryConfig),

		HealthCheckEnabled: true,

		HealthCheckConfigs: make(map[string]*HealthCheckConfig),

		GracefulShutdownEnabled: true,

		ShutdownTimeout: 30 * time.Second,

		ShutdownHooks: make([]string, 0),
	}

}

// NewResilienceMetrics creates new resilience metrics.

func NewResilienceMetrics() *ResilienceMetrics {

	return &ResilienceMetrics{

		TimeoutMetrics: &TimeoutMetrics{OperationsByTimeout: make(map[time.Duration]int64)},

		BulkheadMetrics: &BulkheadMetrics{BulkheadDetails: make(map[string]*BulkheadDetail)},

		CircuitBreakerMetrics: &CircuitBreakerMetrics{},

		RateLimitMetrics: &RateLimitMetrics{},

		RetryMetrics: &RetryMetrics{},

		HealthCheckMetrics: &HealthCheckMetrics{},

		ResourceMetrics: &ResourceMetrics{},

		LastUpdated: time.Now(),
	}

}

// Define missing metric types (placeholder implementations).

type RetryMetrics struct {
	TotalRetries int64 `json:"totalRetries"`

	SuccessfulRetries int64 `json:"successfulRetries"`

	FailedRetries int64 `json:"failedRetries"`

	AverageAttempts float64 `json:"averageAttempts"`

	AverageLatency time.Duration `json:"averageLatency"`
}

// HealthCheckMetrics represents a healthcheckmetrics.

type HealthCheckMetrics struct {
	TotalChecks int64 `json:"totalChecks"`

	HealthyChecks int64 `json:"healthyChecks"`

	UnhealthyChecks int64 `json:"unhealthyChecks"`

	AverageLatency time.Duration `json:"averageLatency"`

	CheckFrequency float64 `json:"checkFrequency"`
}

// ResourceMetrics represents a resourcemetrics.

type ResourceMetrics struct {
	CPUUsage float64 `json:"cpuUsage"`

	MemoryUsage int64 `json:"memoryUsage"`

	NetworkUsage int64 `json:"networkUsage"`

	DiskUsage int64 `json:"diskUsage"`

	GoroutineCount int `json:"goroutineCount"`
}

// RetryConfig represents a retryconfig.

type RetryConfig struct {
	Name string `json:"name"`

	MaxAttempts int `json:"maxAttempts"`

	InitialDelay time.Duration `json:"initialDelay"`

	MaxDelay time.Duration `json:"maxDelay"`

	BackoffFactor float64 `json:"backoffFactor"`

	Jitter bool `json:"jitter"`
}

// HealthCheckConfig represents a healthcheckconfig.

type HealthCheckConfig struct {
	Name string `json:"name"`

	Interval time.Duration `json:"interval"`

	Timeout time.Duration `json:"timeout"`

	FailureThreshold int `json:"failureThreshold"`

	SuccessThreshold int `json:"successThreshold"`
}

// RateLimitMetrics represents a ratelimitmetrics.

type RateLimitMetrics struct {
	TotalRequests int64 `json:"totalRequests"`

	AllowedRequests int64 `json:"allowedRequests"`

	RejectedRequests int64 `json:"rejectedRequests"`

	CurrentRate float64 `json:"currentRate"`

	AverageRate float64 `json:"averageRate"`
}

// RateLimiterMetrics represents a ratelimitermetrics.

type RateLimiterMetrics struct {
	RequestsAllowed int64 `json:"requestsAllowed"`

	RequestsRejected int64 `json:"requestsRejected"`

	CurrentTokens int `json:"currentTokens"`

	RefillRate float64 `json:"refillRate"`
}

// Placeholder manager implementations (these would be fully implemented in production).

type (
	RetryManager struct{ logger logr.Logger }

	// HealthCheckManager represents a healthcheckmanager.

	HealthCheckManager struct{ logger logr.Logger }

	// GracefulShutdownManager represents a gracefulshutdownmanager.

	GracefulShutdownManager struct{ logger logr.Logger }
)

// NewBulkheadManager creates a new BulkheadManager.

func NewBulkheadManager(configs map[string]*BulkheadConfig, logger logr.Logger) *BulkheadManager {

	return &BulkheadManager{

		bulkheads: make(map[string]*Bulkhead),

		metrics: &BulkheadMetrics{},

		logger: logger,
	}

}

// NewCircuitBreakerManager creates a new CircuitBreakerManager.

func NewCircuitBreakerManager(configs map[string]*CircuitBreakerConfig, logger logr.Logger) *CircuitBreakerManager {

	return &CircuitBreakerManager{

		circuitBreakers: make(map[string]*AdvancedCircuitBreaker),

		metrics: &CircuitBreakerMetrics{},

		logger: logger,
	}

}

// NewRateLimiterManager creates a new RateLimiterManager.

func NewRateLimiterManager(configs map[string]*RateLimitConfig, logger logr.Logger) *RateLimiterManager {

	return &RateLimiterManager{

		limiters: make(map[string]*AdaptiveRateLimiter),

		metrics: &RateLimitMetrics{},

		logger: logger,
	}

}

// NewRetryManager performs newretrymanager operation.

func NewRetryManager(configs map[string]*RetryConfig, logger logr.Logger) *RetryManager {

	return &RetryManager{logger: logger}

}

// NewHealthCheckManager performs newhealthcheckmanager operation.

func NewHealthCheckManager(configs map[string]*HealthCheckConfig, logger logr.Logger) *HealthCheckManager {

	return &HealthCheckManager{logger: logger}

}

// NewGracefulShutdownManager performs newgracefulshutdownmanager operation.

func NewGracefulShutdownManager(timeout time.Duration, logger logr.Logger) *GracefulShutdownManager {

	return &GracefulShutdownManager{logger: logger}

}

// BulkheadManager methods.

func (bm *BulkheadManager) Start(ctx context.Context) { /* Implementation */ }

// Stop performs stop operation.

func (bm *BulkheadManager) Stop() { /* Implementation */ }

// GetMetrics performs getmetrics operation.

func (bm *BulkheadManager) GetMetrics() *BulkheadMetrics { return bm.metrics }

// Execute performs execute operation.

func (bm *BulkheadManager) Execute(ctx context.Context, operationName string, operation func(ctx context.Context) (interface{}, error)) (interface{}, error) {

	return operation(ctx) /* Implementation */

}

// CheckHealth performs checkhealth operation.

func (bm *BulkheadManager) CheckHealth() map[string]interface{} {

	return map[string]interface{}{"healthy": true}

}

// CircuitBreakerManager methods.

func (cbm *CircuitBreakerManager) Start(ctx context.Context) { /* Implementation */ }

// Stop performs stop operation.

func (cbm *CircuitBreakerManager) Stop() { /* Implementation */ }

// GetMetrics performs getmetrics operation.

func (cbm *CircuitBreakerManager) GetMetrics() *CircuitBreakerMetrics { return cbm.metrics }

// CanExecute performs canexecute operation.

func (cbm *CircuitBreakerManager) CanExecute(operationName string) bool {

	return true /* Implementation */

}

// RecordResult performs recordresult operation.

func (cbm *CircuitBreakerManager) RecordResult(operationName string, success bool, duration ...time.Duration) { /* Implementation */

}

// CheckHealth performs checkhealth operation.

func (cbm *CircuitBreakerManager) CheckHealth() map[string]interface{} {

	return map[string]interface{}{"healthy": true}

}

// RateLimiterManager methods.

func (rlm *RateLimiterManager) Start(ctx context.Context) { /* Implementation */ }

// Stop performs stop operation.

func (rlm *RateLimiterManager) Stop() { /* Implementation */ }

// GetMetrics performs getmetrics operation.

func (rlm *RateLimiterManager) GetMetrics() *RateLimitMetrics { return rlm.metrics }

// Allow performs allow operation.

func (rlm *RateLimiterManager) Allow(operationName string) bool { return true /* Implementation */ }

// CheckHealth performs checkhealth operation.

func (rlm *RateLimiterManager) CheckHealth() map[string]interface{} {

	return map[string]interface{}{"healthy": true}

}

// Start performs start operation.

func (rm *RetryManager) Start(ctx context.Context) { /* Implementation */ }

// Stop performs stop operation.

func (rm *RetryManager) Stop() { /* Implementation */ }

// GetMetrics performs getmetrics operation.

func (rm *RetryManager) GetMetrics() *RetryMetrics { return &RetryMetrics{} }

// Start performs start operation.

func (hcm *HealthCheckManager) Start(ctx context.Context) { /* Implementation */ }

// Stop performs stop operation.

func (hcm *HealthCheckManager) Stop() { /* Implementation */ }

// PerformHealthChecks performs performhealthchecks operation.

func (hcm *HealthCheckManager) PerformHealthChecks() { /* Implementation */ }

// GetMetrics performs getmetrics operation.

func (hcm *HealthCheckManager) GetMetrics() *HealthCheckMetrics { return &HealthCheckMetrics{} }

// Start performs start operation.

func (gsm *GracefulShutdownManager) Start(ctx context.Context) { /* Implementation */ }

// Stop performs stop operation.

func (gsm *GracefulShutdownManager) Stop() { /* Implementation */ }

// GetStats implementations for manager interfaces.

// GetStats returns statistics for BulkheadManager (interface-compatible method).

func (bm *BulkheadManager) GetStats() (map[string]interface{}, error) {

	bm.mutex.RLock()

	defer bm.mutex.RUnlock()

	stats := map[string]interface{}{

		"bulkhead_count": len(bm.bulkheads),

		"healthy": true,
	}

	return stats, nil

}

// GetStats returns statistics for CircuitBreakerManager (interface-compatible method).

func (cbm *CircuitBreakerManager) GetStats() (map[string]interface{}, error) {

	cbm.mutex.RLock()

	defer cbm.mutex.RUnlock()

	stats := map[string]interface{}{

		"circuit_breaker_count": len(cbm.circuitBreakers),

		"healthy": true,
	}

	return stats, nil

}

// GetStats returns statistics for RateLimiterManager (interface-compatible method).

func (rlm *RateLimiterManager) GetStats() (map[string]interface{}, error) {

	rlm.mutex.RLock()

	defer rlm.mutex.RUnlock()

	stats := map[string]interface{}{

		"rate_limiter_count": len(rlm.limiters),

		"healthy": true,
	}

	return stats, nil

}
