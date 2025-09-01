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
	"time"

	"github.com/go-logr/logr"
	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"
)

// ResilienceConfig defines configuration for resilience patterns
type ResilienceConfig struct {
	// DefaultTimeout for operations
	DefaultTimeout time.Duration

	// MaxConcurrentOperations limits concurrent operations
	MaxConcurrentOperations int

	// HealthCheckInterval for periodic health checks
	HealthCheckInterval time.Duration

	// Feature flags for enabling/disabling patterns
	TimeoutEnabled        bool
	BulkheadEnabled       bool
	CircuitBreakerEnabled bool
	RateLimitingEnabled   bool
	RetryEnabled          bool
	HealthCheckEnabled    bool

	// Timeout configurations for different operation types
	TimeoutConfigs map[string]*TimeoutConfig

	// Circuit breaker configurations
	CircuitBreakerConfigs map[string]*CircuitBreakerConfig

	// Rate limiting configurations
	RateLimitConfigs map[string]*RateLimitConfig

	// Retry configurations
	RetryConfigs map[string]*RetryConfig
}

// TimeoutConfig defines timeout configuration for a specific operation
type TimeoutConfig struct {
	Name            string
	DefaultTimeout  time.Duration
	MaxTimeout      time.Duration
	MinTimeout      time.Duration
	AdaptiveTimeout bool
	TimeoutGradient bool
}

// CircuitBreakerConfig defines circuit breaker configuration
type CircuitBreakerConfig struct {
	Name          string
	MaxRequests   uint32
	Interval      time.Duration
	Timeout       time.Duration
	ReadyToTrip   func(counts gobreaker.Counts) bool
	OnStateChange func(name string, from gobreaker.State, to gobreaker.State)
	IsSuccessful  func(err error) bool
}

// RateLimitConfig defines rate limiting configuration
type RateLimitConfig struct {
	Name      string
	Limit     rate.Limit
	Burst     int
	PerSecond bool
}

// RetryConfig defines retry configuration
type RetryConfig struct {
	Name           string
	MaxAttempts    int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	Multiplier     float64
	Jitter         bool
}

// ResilienceMetrics contains metrics about resilience patterns
type ResilienceMetrics struct {
	// Timeout metrics
	TimeoutMetrics *TimeoutMetrics

	// Circuit breaker metrics
	CircuitBreakerMetrics *CircuitBreakerMetrics

	// Rate limiting metrics
	RateLimitMetrics *RateLimitMetrics

	// Retry metrics
	RetryMetrics *RetryMetrics

	// Bulkhead metrics
	BulkheadMetrics *BulkheadMetrics

	// Overall health metrics
	HealthMetrics *HealthMetrics
}

// TimeoutMetrics contains timeout-related metrics
type TimeoutMetrics struct {
	TotalOperations     int64
	TimeoutOperations   int64
	AverageLatency      time.Duration
	TimeoutRate         float64
	AdaptiveAdjustments int64
	OperationsByTimeout map[time.Duration]int64
	mutex               sync.RWMutex
}

// CircuitBreakerMetrics contains circuit breaker metrics
type CircuitBreakerMetrics struct {
	State           string
	TotalRequests   int64
	SuccessfulReqs  int64
	FailedRequests  int64
	ConsecutiveSucc int64
	ConsecutiveFail int64
}

// RateLimitMetrics contains rate limiting metrics
type RateLimitMetrics struct {
	AllowedRequests int64
	DroppedRequests int64
	CurrentQPS      float64
	BurstCapacity   int
}

// RetryMetrics contains retry-related metrics
type RetryMetrics struct {
	TotalRetries        int64
	SuccessfulRetries   int64
	FailedRetries       int64
	AverageRetryLatency time.Duration
}

// BulkheadMetrics contains bulkhead pattern metrics
type BulkheadMetrics struct {
	ActiveOperations   int
	QueuedOperations   int
	RejectedOperations int64
	MaxConcurrency     int
}

// HealthMetrics contains overall health metrics
type HealthMetrics struct {
	OverallHealthy    bool
	ComponentsHealthy map[string]bool
	LastHealthCheck   time.Time
	HealthCheckErrors int64
}

// ResilienceManager implements resilience patterns for distributed systems
type ResilienceManager struct {
	config *ResilienceConfig
	logger logr.Logger
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc

	// Pattern implementations
	circuitBreakers map[string]*gobreaker.CircuitBreaker
	rateLimiters    map[string]*rate.Limiter
	semaphores      map[string]*semaphore
	timeouts        map[string]*TimeoutConfig

	// Metrics
	metrics *ResilienceMetrics

	// Internal state
	running      bool
	healthTicker *time.Ticker
	healthChecks map[string]HealthCheckFunc
}

// HealthCheckFunc defines a function that performs a health check
type HealthCheckFunc func(ctx context.Context) error

// semaphore implements a counting semaphore for bulkhead pattern
type semaphore struct {
	ch chan struct{}
	mu sync.Mutex
}

// newSemaphore creates a new semaphore with the given capacity
func newSemaphore(capacity int) *semaphore {
	return &semaphore{
		ch: make(chan struct{}, capacity),
	}
}

// Acquire acquires a permit from the semaphore
func (s *semaphore) Acquire(ctx context.Context) error {
	select {
	case s.ch <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Release releases a permit back to the semaphore
func (s *semaphore) Release() {
	select {
	case <-s.ch:
	default:
		// Should not happen if used correctly
	}
}

// Available returns the number of available permits
func (s *semaphore) Available() int {
	return cap(s.ch) - len(s.ch)
}

// NewResilienceManager creates a new ResilienceManager
func NewResilienceManager(config *ResilienceConfig, logger logr.Logger) *ResilienceManager {
	if config == nil {
		config = &ResilienceConfig{
			DefaultTimeout:          30 * time.Second,
			MaxConcurrentOperations: 100,
			HealthCheckInterval:     30 * time.Second,
			TimeoutEnabled:          true,
			BulkheadEnabled:         true,
			CircuitBreakerEnabled:   true,
			RateLimitingEnabled:     true,
			RetryEnabled:            true,
			HealthCheckEnabled:      true,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	rm := &ResilienceManager{
		config:          config,
		logger:          logger,
		ctx:             ctx,
		cancel:          cancel,
		circuitBreakers: make(map[string]*gobreaker.CircuitBreaker),
		rateLimiters:    make(map[string]*rate.Limiter),
		semaphores:      make(map[string]*semaphore),
		timeouts:        make(map[string]*TimeoutConfig),
		healthChecks:    make(map[string]HealthCheckFunc),
		metrics:         &ResilienceMetrics{},
	}

	rm.initializeMetrics()
	rm.initializePatterns()

	return rm
}

// Start starts the resilience manager and background processes
func (rm *ResilienceManager) Start(ctx context.Context) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.running {
		return fmt.Errorf("resilience manager is already running")
	}

	rm.running = true

	// Start health check ticker if enabled
	if rm.config.HealthCheckEnabled {
		rm.healthTicker = time.NewTicker(rm.config.HealthCheckInterval)
		go rm.healthCheckLoop()
	}

	rm.logger.Info("Resilience manager started", "config", rm.config)
	return nil
}

// Stop stops the resilience manager
func (rm *ResilienceManager) Stop() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if !rm.running {
		return
	}

	rm.running = false
	rm.cancel()

	if rm.healthTicker != nil {
		rm.healthTicker.Stop()
	}

	rm.logger.Info("Resilience manager stopped")
}

// ExecuteWithTimeout executes a function with timeout protection
func (rm *ResilienceManager) ExecuteWithTimeout(ctx context.Context, operationType string, fn func(ctx context.Context) error) error {
	if !rm.config.TimeoutEnabled {
		return fn(ctx)
	}

	config := rm.getTimeoutConfig(operationType)

	timeoutCtx, cancel := context.WithTimeout(ctx, config.DefaultTimeout)
	defer cancel()

	start := time.Now()
	err := fn(timeoutCtx)
	duration := time.Since(start)

	rm.updateTimeoutMetrics(duration, err != nil && timeoutCtx.Err() == context.DeadlineExceeded)

	return err
}

// ExecuteWithCircuitBreaker executes a function with circuit breaker protection
func (rm *ResilienceManager) ExecuteWithCircuitBreaker(ctx context.Context, breakerName string, fn func() (interface{}, error)) (interface{}, error) {
	if !rm.config.CircuitBreakerEnabled {
		return fn()
	}

	breaker := rm.getCircuitBreaker(breakerName)
	return breaker.Execute(fn)
}

// ExecuteWithRateLimit executes a function with rate limiting
func (rm *ResilienceManager) ExecuteWithRateLimit(ctx context.Context, limiterName string, fn func() error) error {
	if !rm.config.RateLimitingEnabled {
		return fn()
	}

	limiter := rm.getRateLimiter(limiterName)

	if !limiter.Allow() {
		rm.updateRateLimitMetrics(false)
		return fmt.Errorf("rate limit exceeded for %s", limiterName)
	}

	rm.updateRateLimitMetrics(true)
	return fn()
}

// ExecuteWithBulkhead executes a function with bulkhead pattern (concurrency limiting)
func (rm *ResilienceManager) ExecuteWithBulkhead(ctx context.Context, bulkheadName string, fn func() error) error {
	if !rm.config.BulkheadEnabled {
		return fn()
	}

	sem := rm.getSemaphore(bulkheadName)

	if err := sem.Acquire(ctx); err != nil {
		rm.updateBulkheadMetrics(bulkheadName, false)
		return fmt.Errorf("failed to acquire bulkhead permit: %w", err)
	}
	defer sem.Release()

	rm.updateBulkheadMetrics(bulkheadName, true)
	return fn()
}

// ExecuteWithRetry executes a function with retry logic
func (rm *ResilienceManager) ExecuteWithRetry(ctx context.Context, retryName string, fn func() error) error {
	if !rm.config.RetryEnabled {
		return fn()
	}

	config := rm.getRetryConfig(retryName)
	var lastErr error

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			rm.updateRetryMetrics(attempt, true)
			return nil
		}

		if attempt < config.MaxAttempts-1 {
			backoff := rm.calculateBackoff(config, attempt)
			select {
			case <-time.After(backoff):
				continue
			case <-ctx.Done():
				rm.updateRetryMetrics(attempt, false)
				return ctx.Err()
			}
		}
	}

	rm.updateRetryMetrics(config.MaxAttempts, false)
	return lastErr
}

// RegisterHealthCheck registers a health check function
func (rm *ResilienceManager) RegisterHealthCheck(name string, healthCheck HealthCheckFunc) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.healthChecks[name] = healthCheck
	rm.logger.Info("Health check registered", "name", name)
}

// GetMetrics returns current resilience metrics
func (rm *ResilienceManager) GetMetrics() *ResilienceMetrics {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// Return a copy to avoid concurrent access issues
	return &ResilienceMetrics{
		TimeoutMetrics:        rm.metrics.TimeoutMetrics,
		CircuitBreakerMetrics: rm.metrics.CircuitBreakerMetrics,
		RateLimitMetrics:      rm.metrics.RateLimitMetrics,
		RetryMetrics:          rm.metrics.RetryMetrics,
		BulkheadMetrics:       rm.metrics.BulkheadMetrics,
		HealthMetrics:         rm.metrics.HealthMetrics,
	}
}

// Health returns the overall health status
func (rm *ResilienceManager) Health() error {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if !rm.running {
		return fmt.Errorf("resilience manager is not running")
	}

	if rm.metrics.HealthMetrics != nil && !rm.metrics.HealthMetrics.OverallHealthy {
		return fmt.Errorf("health checks failing")
	}

	return nil
}

// initializeMetrics initializes all metrics structures
func (rm *ResilienceManager) initializeMetrics() {
	rm.metrics = &ResilienceMetrics{
		TimeoutMetrics: &TimeoutMetrics{},
		CircuitBreakerMetrics: &CircuitBreakerMetrics{
			State: "closed",
		},
		RateLimitMetrics: &RateLimitMetrics{},
		RetryMetrics:     &RetryMetrics{},
		BulkheadMetrics: &BulkheadMetrics{
			MaxConcurrency: rm.config.MaxConcurrentOperations,
		},
		HealthMetrics: &HealthMetrics{
			OverallHealthy:    true,
			ComponentsHealthy: make(map[string]bool),
			LastHealthCheck:   time.Now(),
		},
	}
}

// initializePatterns initializes resilience patterns
func (rm *ResilienceManager) initializePatterns() {
	// Initialize default timeout configs
	if rm.config.TimeoutConfigs != nil {
		for name, config := range rm.config.TimeoutConfigs {
			rm.timeouts[name] = config
		}
	}

	// Initialize default circuit breakers
	if rm.config.CircuitBreakerConfigs != nil {
		for name, config := range rm.config.CircuitBreakerConfigs {
			rm.createCircuitBreaker(name, config)
		}
	}

	// Initialize default rate limiters
	if rm.config.RateLimitConfigs != nil {
		for name, config := range rm.config.RateLimitConfigs {
			rm.createRateLimiter(name, config)
		}
	}

	// Initialize default semaphores
	rm.semaphores["default"] = newSemaphore(rm.config.MaxConcurrentOperations)
}

// getTimeoutConfig gets or creates a timeout configuration
func (rm *ResilienceManager) getTimeoutConfig(operationType string) *TimeoutConfig {
	if config, exists := rm.timeouts[operationType]; exists {
		return config
	}

	// Create default timeout config
	config := &TimeoutConfig{
		Name:           operationType,
		DefaultTimeout: rm.config.DefaultTimeout,
		MaxTimeout:     rm.config.DefaultTimeout * 2,
		MinTimeout:     time.Second,
	}
	rm.timeouts[operationType] = config
	return config
}

// getCircuitBreaker gets or creates a circuit breaker
func (rm *ResilienceManager) getCircuitBreaker(name string) *gobreaker.CircuitBreaker {
	if breaker, exists := rm.circuitBreakers[name]; exists {
		return breaker
	}

	// Create default circuit breaker
	config := &CircuitBreakerConfig{
		Name:        name,
		MaxRequests: 10,
		Interval:    time.Minute,
		Timeout:     30 * time.Second,
	}
	return rm.createCircuitBreaker(name, config)
}

// getRateLimiter gets or creates a rate limiter
func (rm *ResilienceManager) getRateLimiter(name string) *rate.Limiter {
	if limiter, exists := rm.rateLimiters[name]; exists {
		return limiter
	}

	// Create default rate limiter
	limiter := rate.NewLimiter(rate.Limit(10), 10) // 10 requests per second, burst of 10
	rm.rateLimiters[name] = limiter
	return limiter
}

// getSemaphore gets or creates a semaphore
func (rm *ResilienceManager) getSemaphore(name string) *semaphore {
	if sem, exists := rm.semaphores[name]; exists {
		return sem
	}

	// Create default semaphore
	sem := newSemaphore(rm.config.MaxConcurrentOperations)
	rm.semaphores[name] = sem
	return sem
}

// getRetryConfig gets retry configuration
func (rm *ResilienceManager) getRetryConfig(name string) *RetryConfig {
	if rm.config.RetryConfigs != nil {
		if config, exists := rm.config.RetryConfigs[name]; exists {
			return config
		}
	}

	// Return default retry config
	return &RetryConfig{
		Name:           name,
		MaxAttempts:    3,
		InitialBackoff: time.Second,
		MaxBackoff:     30 * time.Second,
		Multiplier:     2.0,
		Jitter:         true,
	}
}

// createCircuitBreaker creates a new circuit breaker
func (rm *ResilienceManager) createCircuitBreaker(name string, config *CircuitBreakerConfig) *gobreaker.CircuitBreaker {
	settings := gobreaker.Settings{
		Name:        config.Name,
		MaxRequests: config.MaxRequests,
		Interval:    config.Interval,
		Timeout:     config.Timeout,
	}

	if config.ReadyToTrip != nil {
		settings.ReadyToTrip = config.ReadyToTrip
	}

	if config.OnStateChange != nil {
		settings.OnStateChange = config.OnStateChange
	}

	if config.IsSuccessful != nil {
		settings.IsSuccessful = config.IsSuccessful
	}

	breaker := gobreaker.NewCircuitBreaker(settings)
	rm.circuitBreakers[name] = breaker
	return breaker
}

// createRateLimiter creates a new rate limiter
func (rm *ResilienceManager) createRateLimiter(name string, config *RateLimitConfig) *rate.Limiter {
	limiter := rate.NewLimiter(config.Limit, config.Burst)
	rm.rateLimiters[name] = limiter
	return limiter
}

// calculateBackoff calculates backoff duration for retry
func (rm *ResilienceManager) calculateBackoff(config *RetryConfig, attempt int) time.Duration {
	backoff := time.Duration(float64(config.InitialBackoff) * (config.Multiplier * float64(attempt)))

	if backoff > config.MaxBackoff {
		backoff = config.MaxBackoff
	}

	if config.Jitter {
		// Add up to 25% jitter
		jitter := time.Duration(float64(backoff) * 0.25 * (0.5 - float64(time.Now().UnixNano()%2)))
		backoff += jitter
	}

	return backoff
}

// Update metrics methods
func (rm *ResilienceManager) updateTimeoutMetrics(duration time.Duration, timedOut bool) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.metrics.TimeoutMetrics.TotalOperations++
	if timedOut {
		rm.metrics.TimeoutMetrics.TimeoutOperations++
	}

	// Update average latency (simple moving average)
	totalOps := rm.metrics.TimeoutMetrics.TotalOperations
	currentAvg := rm.metrics.TimeoutMetrics.AverageLatency
	rm.metrics.TimeoutMetrics.AverageLatency = time.Duration(
		(int64(currentAvg)*(totalOps-1) + int64(duration)) / totalOps,
	)
}

func (rm *ResilienceManager) updateRateLimitMetrics(allowed bool) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if allowed {
		rm.metrics.RateLimitMetrics.AllowedRequests++
	} else {
		rm.metrics.RateLimitMetrics.DroppedRequests++
	}
}

func (rm *ResilienceManager) updateBulkheadMetrics(bulkheadName string, acquired bool) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if acquired {
		rm.metrics.BulkheadMetrics.ActiveOperations++
	} else {
		rm.metrics.BulkheadMetrics.RejectedOperations++
	}
}

func (rm *ResilienceManager) updateRetryMetrics(attempts int, successful bool) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.metrics.RetryMetrics.TotalRetries += int64(attempts)
	if successful {
		rm.metrics.RetryMetrics.SuccessfulRetries++
	} else {
		rm.metrics.RetryMetrics.FailedRetries++
	}
}

// healthCheckLoop runs periodic health checks
func (rm *ResilienceManager) healthCheckLoop() {
	for {
		select {
		case <-rm.healthTicker.C:
			rm.performHealthChecks()
		case <-rm.ctx.Done():
			return
		}
	}
}

// performHealthChecks executes all registered health checks
func (rm *ResilienceManager) performHealthChecks() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	overallHealthy := true
	for name, healthCheck := range rm.healthChecks {
		ctx, cancel := context.WithTimeout(rm.ctx, 10*time.Second)
		err := healthCheck(ctx)
		cancel()

		healthy := err == nil
		rm.metrics.HealthMetrics.ComponentsHealthy[name] = healthy

		if !healthy {
			overallHealthy = false
			rm.metrics.HealthMetrics.HealthCheckErrors++
			rm.logger.Error(err, "Health check failed", "component", name)
		}
	}

	rm.metrics.HealthMetrics.OverallHealthy = overallHealthy
	rm.metrics.HealthMetrics.LastHealthCheck = time.Now()
}

// TimeoutManager manages timeout operations across the system
type TimeoutManager struct {
	configs    map[string]*TimeoutConfig
	operations sync.Map // map[string]*TimeoutOperation
	metrics    *TimeoutMetrics
	mu         sync.RWMutex
	mutex      sync.Mutex
	logger     logr.Logger
}

// TimeoutOperation represents a timeout operation
type TimeoutOperation struct {
	ID             string            `json:"id"`
	Duration       time.Duration     `json:"duration"`
	StartTime      time.Time         `json:"startTime"`
	Cancelled      bool              `json:"cancelled"`
	Timeout        time.Duration     `json:"timeout"`
	Context        context.Context   `json:"-"`
	CancelFunc     context.CancelFunc `json:"-"`
	CompletedChan  chan bool         `json:"-"`
	TimeoutChan    chan bool         `json:"-"`
	mutex          sync.RWMutex      `json:"-"`
}

// AverageTimeout returns the average timeout duration from metrics
func (tm *TimeoutMetrics) AverageTimeout() time.Duration {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	
	if tm.TotalOperations == 0 {
		return 0
	}
	
	// Calculate weighted average based on operation timeouts
	var totalDuration time.Duration
	var totalOps int64
	
	for timeout, count := range tm.OperationsByTimeout {
		totalDuration += time.Duration(int64(timeout) * count)
		totalOps += count
	}
	
	if totalOps == 0 {
		return tm.AverageLatency
	}
	
	return totalDuration / time.Duration(totalOps)
}
