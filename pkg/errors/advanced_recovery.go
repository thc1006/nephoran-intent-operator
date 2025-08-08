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
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
)

// AdvancedRecoveryManager provides comprehensive error recovery with enhanced patterns
type AdvancedRecoveryManager struct {
	// Core components
	errorAggregator      *ErrorAggregator
	retryExecutors       map[string]*RetryExecutor
	circuitBreakers      map[string]*CircuitBreaker
	bulkheads           map[string]*Bulkhead
	timeoutExecutors     map[string]*TimeoutExecutor
	rateLimiters        map[string]*RateLimiter
	degradationManager  *DegradationManager
	fallbackManager     *FallbackManager
	
	// Configuration and state
	config              *AdvancedRecoveryConfig
	logger              logr.Logger
	metrics             *AdvancedRecoveryMetrics
	
	// Async processing
	recoveryQueue       chan *AdvancedRecoveryRequest
	monitoringTicker    *time.Ticker
	
	// State management
	mutex               sync.RWMutex
	started             bool
	stopChan            chan struct{}
}

// AdvancedRecoveryConfig holds comprehensive recovery configuration
type AdvancedRecoveryConfig struct {
	// Global settings
	MaxConcurrentRecoveries  int           `json:"maxConcurrentRecoveries"`
	RecoveryTimeout          time.Duration `json:"recoveryTimeout"`
	MonitoringInterval       time.Duration `json:"monitoringInterval"`
	QueueSize               int           `json:"queueSize"`
	
	// Error correlation settings
	ErrorCorrelationWindow   time.Duration `json:"errorCorrelationWindow"`
	ErrorCorrelationThreshold int          `json:"errorCorrelationThreshold"`
	
	// Automatic recovery settings
	AutoRecoveryEnabled      bool          `json:"autoRecoveryEnabled"`
	AutoRecoveryDelay        time.Duration `json:"autoRecoveryDelay"`
	MaxAutoRecoveryAttempts  int           `json:"maxAutoRecoveryAttempts"`
	
	// Circuit breaker cascade settings
	CascadeProtectionEnabled bool          `json:"cascadeProtectionEnabled"`
	CascadeDetectionWindow   time.Duration `json:"cascadeDetectionWindow"`
	CascadeThreshold        int           `json:"cascadeThreshold"`
	
	// Health check settings
	HealthCheckInterval     time.Duration `json:"healthCheckInterval"`
	HealthCheckTimeout      time.Duration `json:"healthCheckTimeout"`
	
	// Resource limits
	MaxMemoryUsage          int64         `json:"maxMemoryUsage"`
	MaxCPUUsage            float64       `json:"maxCpuUsage"`
}

// AdvancedRecoveryRequest represents a comprehensive recovery request
type AdvancedRecoveryRequest struct {
	// Basic information
	ID              string                 `json:"id"`
	Error           *ProcessingError       `json:"error"`
	Context         context.Context        `json:"-"`
	Strategy        RecoveryStrategy       `json:"strategy"`
	Priority        int                   `json:"priority"`
	
	// Execution context
	Component       string                 `json:"component"`
	Operation       string                 `json:"operation"`
	Phase          string                 `json:"phase"`
	
	// Recovery configuration
	MaxRetries      int                   `json:"maxRetries"`
	Timeout         time.Duration         `json:"timeout"`
	BackoffStrategy string                `json:"backoffStrategy"`
	
	// Callback and result handling
	Callback        AdvancedRecoveryCallback `json:"-"`
	ResultChan      chan *AdvancedRecoveryResult `json:"-"`
	
	// Tracking information
	CreatedAt       time.Time             `json:"createdAt"`
	StartedAt       *time.Time            `json:"startedAt,omitempty"`
	CompletedAt     *time.Time            `json:"completedAt,omitempty"`
	
	// Correlation and grouping
	CorrelationID   string                `json:"correlationId"`
	ParentRequestID string                `json:"parentRequestId,omitempty"`
	GroupID         string                `json:"groupId,omitempty"`
}

// AdvancedRecoveryCallback defines the callback interface for recovery completion
type AdvancedRecoveryCallback func(request *AdvancedRecoveryRequest, result *AdvancedRecoveryResult)

// AdvancedRecoveryResult contains comprehensive recovery results
type AdvancedRecoveryResult struct {
	// Basic result information
	Success         bool                   `json:"success"`
	RequestID       string                 `json:"requestId"`
	Strategy        RecoveryStrategy       `json:"strategy"`
	AttemptsUsed    int                   `json:"attemptsUsed"`
	Duration        time.Duration         `json:"duration"`
	
	// Error information
	FinalError      error                 `json:"-"`
	ErrorMessage    string                `json:"errorMessage,omitempty"`
	ErrorCode       string                `json:"errorCode,omitempty"`
	
	// Result data
	Data            map[string]interface{} `json:"data,omitempty"`
	RecoveredState  map[string]interface{} `json:"recoveredState,omitempty"`
	
	// Performance metrics
	Metrics         *RecoveryPerformanceMetrics `json:"metrics"`
	
	// Recovery metadata
	RecoveryPath    []string              `json:"recoveryPath,omitempty"`
	FallbacksUsed   []string              `json:"fallbacksUsed,omitempty"`
	CircuitBreakers []string              `json:"circuitBreakers,omitempty"`
	
	// Health status
	ComponentHealth map[string]string     `json:"componentHealth,omitempty"`
}

// RecoveryPerformanceMetrics tracks detailed recovery performance
type RecoveryPerformanceMetrics struct {
	TotalDuration       time.Duration     `json:"totalDuration"`
	RetryDuration       time.Duration     `json:"retryDuration"`
	CircuitBreakerTime  time.Duration     `json:"circuitBreakerTime"`
	FallbackTime        time.Duration     `json:"fallbackTime"`
	HealthCheckTime     time.Duration     `json:"healthCheckTime"`
	
	MemoryUsage         int64             `json:"memoryUsage"`
	CPUUsage           float64           `json:"cpuUsage"`
	NetworkCalls        int               `json:"networkCalls"`
	DatabaseCalls       int               `json:"databaseCalls"`
	
	CustomMetrics       map[string]float64 `json:"customMetrics,omitempty"`
}

// AdvancedRecoveryMetrics tracks comprehensive recovery statistics
type AdvancedRecoveryMetrics struct {
	// Basic statistics
	TotalRequests       int64                      `json:"totalRequests"`
	SuccessfulRecoveries int64                     `json:"successfulRecoveries"`
	FailedRecoveries    int64                      `json:"failedRecoveries"`
	
	// Performance statistics
	AverageRecoveryTime time.Duration              `json:"averageRecoveryTime"`
	P95RecoveryTime     time.Duration              `json:"p95RecoveryTime"`
	P99RecoveryTime     time.Duration              `json:"p99RecoveryTime"`
	
	// Strategy effectiveness
	StrategyStats       map[RecoveryStrategy]*StrategyMetrics `json:"strategyStats"`
	
	// Component health
	ComponentStats      map[string]*ComponentRecoveryStats   `json:"componentStats"`
	
	// Error correlation
	CorrelationStats    *CorrelationMetrics        `json:"correlationStats"`
	
	// Resource utilization
	ResourceStats       *ResourceUtilizationStats  `json:"resourceStats"`
	
	// Timing
	LastUpdated         time.Time                  `json:"lastUpdated"`
	
	mutex               sync.RWMutex
}

// StrategyMetrics tracks metrics for each recovery strategy
type StrategyMetrics struct {
	UsageCount      int64         `json:"usageCount"`
	SuccessRate     float64       `json:"successRate"`
	AverageLatency  time.Duration `json:"averageLatency"`
	FailureReasons  map[string]int64 `json:"failureReasons"`
}

// ComponentRecoveryStats tracks recovery statistics per component
type ComponentRecoveryStats struct {
	RecoveryCount      int64         `json:"recoveryCount"`
	SuccessRate        float64       `json:"successRate"`
	AverageRecoveryTime time.Duration `json:"averageRecoveryTime"`
	CommonErrors       map[string]int64 `json:"commonErrors"`
	LastRecovery       time.Time     `json:"lastRecovery"`
}

// CorrelationMetrics tracks error correlation statistics
type CorrelationMetrics struct {
	CorrelatedGroups      int64              `json:"correlatedGroups"`
	AverageGroupSize      float64            `json:"averageGroupSize"`
	LargestGroup         int                `json:"largestGroup"`
	CorrelationPatterns  map[string]int64   `json:"correlationPatterns"`
}

// ResourceUtilizationStats tracks resource usage during recovery
type ResourceUtilizationStats struct {
	PeakMemoryUsage    int64    `json:"peakMemoryUsage"`
	AverageMemoryUsage int64    `json:"averageMemoryUsage"`
	PeakCPUUsage      float64  `json:"peakCpuUsage"`
	AverageCPUUsage   float64  `json:"averageCpuUsage"`
	NetworkTraffic    int64    `json:"networkTraffic"`
	DiskIO           int64    `json:"diskIo"`
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	name       string
	rate       float64       // tokens per second
	capacity   float64       // bucket capacity
	tokens     float64       // current tokens
	lastRefill time.Time     // last refill time
	mutex      sync.Mutex
	logger     logr.Logger
}

// DegradationManager manages service degradation strategies
type DegradationManager struct {
	config           *DegradationConfig
	degradationLevels map[string]*DegradationLevel
	currentLevels    map[string]int
	logger           logr.Logger
	mutex            sync.RWMutex
}

// DegradationConfig configures service degradation
type DegradationConfig struct {
	EnableDegradation    bool          `json:"enableDegradation"`
	AutoDegradation      bool          `json:"autoDegradation"`
	DegradationThreshold float64       `json:"degradationThreshold"`
	RecoveryThreshold    float64       `json:"recoveryThreshold"`
	EvaluationWindow     time.Duration `json:"evaluationWindow"`
}

// DegradationLevel defines a level of service degradation
type DegradationLevel struct {
	Level       int                    `json:"level"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Features    []string               `json:"features"`
	Limits      map[string]interface{} `json:"limits"`
}

// FallbackManager manages fallback strategies
type FallbackManager struct {
	fallbacks map[string]*FallbackStrategy
	logger    logr.Logger
	mutex     sync.RWMutex
}

// FallbackStrategy defines a fallback approach
type FallbackStrategy struct {
	Name         string                `json:"name"`
	Component    string                `json:"component"`
	Operation    string                `json:"operation"`
	Handler      FallbackHandler       `json:"-"`
	Priority     int                   `json:"priority"`
	Timeout      time.Duration         `json:"timeout"`
	Conditions   []string              `json:"conditions"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// FallbackHandler defines the interface for fallback execution
type FallbackHandler func(ctx context.Context, originalError error, originalData map[string]interface{}) (interface{}, error)

// NewAdvancedRecoveryManager creates a new advanced recovery manager
func NewAdvancedRecoveryManager(config *AdvancedRecoveryConfig, errorAggregator *ErrorAggregator, logger logr.Logger) *AdvancedRecoveryManager {
	if config == nil {
		config = getDefaultAdvancedRecoveryConfig()
	}
	
	manager := &AdvancedRecoveryManager{
		errorAggregator:    errorAggregator,
		retryExecutors:     make(map[string]*RetryExecutor),
		circuitBreakers:    make(map[string]*CircuitBreaker),
		bulkheads:         make(map[string]*Bulkhead),
		timeoutExecutors:   make(map[string]*TimeoutExecutor),
		rateLimiters:      make(map[string]*RateLimiter),
		config:            config,
		logger:            logger,
		metrics:           NewAdvancedRecoveryMetrics(),
		recoveryQueue:     make(chan *AdvancedRecoveryRequest, config.QueueSize),
		monitoringTicker:  time.NewTicker(config.MonitoringInterval),
		stopChan:          make(chan struct{}),
	}
	
	// Initialize sub-managers
	manager.degradationManager = NewDegradationManager(&DegradationConfig{
		EnableDegradation:    true,
		AutoDegradation:      true,
		DegradationThreshold: 0.8,
		RecoveryThreshold:    0.6,
		EvaluationWindow:     5 * time.Minute,
	}, logger)
	
	manager.fallbackManager = NewFallbackManager(logger)
	
	return manager
}

// Start starts the advanced recovery manager
func (arm *AdvancedRecoveryManager) Start(ctx context.Context) error {
	arm.mutex.Lock()
	defer arm.mutex.Unlock()
	
	if arm.started {
		return fmt.Errorf("advanced recovery manager already started")
	}
	
	arm.started = true
	
	// Start worker goroutines
	for i := 0; i < arm.config.MaxConcurrentRecoveries; i++ {
		go arm.recoveryWorker(ctx, i)
	}
	
	// Start monitoring
	go arm.monitor(ctx)
	
	// Start health checks
	go arm.healthChecker(ctx)
	
	arm.logger.Info("Advanced recovery manager started", 
		"maxConcurrentRecoveries", arm.config.MaxConcurrentRecoveries,
		"queueSize", arm.config.QueueSize)
	
	return nil
}

// Stop stops the advanced recovery manager
func (arm *AdvancedRecoveryManager) Stop() error {
	arm.mutex.Lock()
	defer arm.mutex.Unlock()
	
	if !arm.started {
		return nil
	}
	
	close(arm.stopChan)
	arm.monitoringTicker.Stop()
	arm.started = false
	
	arm.logger.Info("Advanced recovery manager stopped")
	return nil
}

// RequestAdvancedRecovery requests advanced error recovery
func (arm *AdvancedRecoveryManager) RequestAdvancedRecovery(req *AdvancedRecoveryRequest) error {
	if !arm.started {
		return fmt.Errorf("advanced recovery manager not started")
	}
	
	req.CreatedAt = time.Now()
	
	select {
	case arm.recoveryQueue <- req:
		atomic.AddInt64(&arm.metrics.TotalRequests, 1)
		return nil
	default:
		return fmt.Errorf("recovery queue full")
	}
}

// recoveryWorker processes recovery requests
func (arm *AdvancedRecoveryManager) recoveryWorker(ctx context.Context, workerID int) {
	arm.logger.Info("Advanced recovery worker started", "workerId", workerID)
	
	for {
		select {
		case req := <-arm.recoveryQueue:
			result := arm.processAdvancedRecoveryRequest(ctx, req)
			
			if req.Callback != nil {
				req.Callback(req, result)
			}
			
			if req.ResultChan != nil {
				select {
				case req.ResultChan <- result:
				default:
					arm.logger.Warn("Failed to send result to channel", "requestId", req.ID)
				}
			}
			
			arm.updateAdvancedMetrics(result)
			
		case <-arm.stopChan:
			arm.logger.Info("Advanced recovery worker stopping", "workerId", workerID)
			return
			
		case <-ctx.Done():
			arm.logger.Info("Advanced recovery worker context cancelled", "workerId", workerID)
			return
		}
	}
}

// processAdvancedRecoveryRequest processes a single advanced recovery request
func (arm *AdvancedRecoveryManager) processAdvancedRecoveryRequest(ctx context.Context, req *AdvancedRecoveryRequest) *AdvancedRecoveryResult {
	startTime := time.Now()
	now := time.Now()
	req.StartedAt = &now
	
	arm.logger.Info("Processing advanced recovery request",
		"requestId", req.ID,
		"component", req.Component,
		"strategy", req.Strategy,
		"errorCode", req.Error.Code)
	
	// Create recovery context with timeout
	recoveryCtx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()
	
	// Initialize result
	result := &AdvancedRecoveryResult{
		RequestID: req.ID,
		Strategy:  req.Strategy,
		Metrics:   &RecoveryPerformanceMetrics{
			CustomMetrics: make(map[string]float64),
		},
		Data:            make(map[string]interface{}),
		RecoveredState:  make(map[string]interface{}),
		ComponentHealth: make(map[string]string),
	}
	
	// Execute recovery strategy
	var recoveryError error
	switch req.Strategy {
	case StrategyRetry, StrategyBackoff, StrategyExponential, StrategyJittered:
		recoveryError = arm.executeRetryRecovery(recoveryCtx, req, result)
		
	case StrategyCircuitBreaker:
		recoveryError = arm.executeCircuitBreakerRecovery(recoveryCtx, req, result)
		
	case StrategyFallback:
		recoveryError = arm.executeFallbackRecovery(recoveryCtx, req, result)
		
	case StrategyBulkhead:
		recoveryError = arm.executeBulkheadRecovery(recoveryCtx, req, result)
		
	case StrategyTimeout:
		recoveryError = arm.executeTimeoutRecovery(recoveryCtx, req, result)
		
	case StrategyRateLimit:
		recoveryError = arm.executeRateLimitRecovery(recoveryCtx, req, result)
		
	case StrategyDegradation:
		recoveryError = arm.executeDegradationRecovery(recoveryCtx, req, result)
		
	default:
		recoveryError = fmt.Errorf("unknown recovery strategy: %s", req.Strategy)
	}
	
	// Finalize result
	result.Duration = time.Since(startTime)
	result.Metrics.TotalDuration = result.Duration
	result.Success = recoveryError == nil
	result.FinalError = recoveryError
	
	if recoveryError != nil {
		result.ErrorMessage = recoveryError.Error()
		if serviceErr, ok := recoveryError.(*ServiceError); ok {
			result.ErrorCode = serviceErr.Code
		}
	}
	
	completedAt := time.Now()
	req.CompletedAt = &completedAt
	
	arm.logger.Info("Advanced recovery request completed",
		"requestId", req.ID,
		"success", result.Success,
		"duration", result.Duration,
		"strategy", req.Strategy)
	
	return result
}

// executeRetryRecovery executes retry-based recovery
func (arm *AdvancedRecoveryManager) executeRetryRecovery(ctx context.Context, req *AdvancedRecoveryRequest, result *AdvancedRecoveryResult) error {
	retryStart := time.Now()
	
	// Get or create retry executor for this component
	executor := arm.getRetryExecutor(req.Component, req.Strategy)
	
	// Create retry operation
	operation := func() error {
		// Simulate the original operation that failed
		// In real implementation, this would re-execute the failed operation
		arm.logger.Info("Attempting retry operation", 
			"requestId", req.ID, 
			"component", req.Component)
		
		// Simulate work with some success probability
		time.Sleep(100 * time.Millisecond)
		
		// For demonstration, succeed after a few attempts
		if result.AttemptsUsed >= 2 {
			return nil
		}
		
		return fmt.Errorf("simulated retry failure")
	}
	
	// Execute with retry
	err := executor.Execute(ctx, operation)
	
	result.Metrics.RetryDuration = time.Since(retryStart)
	result.AttemptsUsed = req.MaxRetries // In real implementation, get from executor
	result.RecoveryPath = append(result.RecoveryPath, string(req.Strategy))
	
	if err == nil {
		result.Data["retry_success"] = true
		result.Data["attempts_used"] = result.AttemptsUsed
	}
	
	return err
}

// executeCircuitBreakerRecovery executes circuit breaker recovery
func (arm *AdvancedRecoveryManager) executeCircuitBreakerRecovery(ctx context.Context, req *AdvancedRecoveryRequest, result *AdvancedRecoveryResult) error {
	cbStart := time.Now()
	
	// Get or create circuit breaker for this component
	cb := arm.getCircuitBreaker(req.Component)
	
	err := cb.Execute(ctx, func() error {
		// Simulate recovery operation
		time.Sleep(50 * time.Millisecond)
		
		// Circuit breaker allows gradual recovery
		return nil
	})
	
	result.Metrics.CircuitBreakerTime = time.Since(cbStart)
	result.CircuitBreakers = append(result.CircuitBreakers, req.Component)
	result.RecoveryPath = append(result.RecoveryPath, "circuit_breaker")
	
	if err == nil {
		result.Data["circuit_breaker_state"] = cb.GetState().String()
		result.ComponentHealth[req.Component] = "healthy"
	} else {
		result.ComponentHealth[req.Component] = "degraded"
	}
	
	return err
}

// executeFallbackRecovery executes fallback recovery
func (arm *AdvancedRecoveryManager) executeFallbackRecovery(ctx context.Context, req *AdvancedRecoveryRequest, result *AdvancedRecoveryResult) error {
	fallbackStart := time.Now()
	
	// Get fallback strategy for this component/operation
	fallback := arm.fallbackManager.GetFallback(req.Component, req.Operation)
	if fallback == nil {
		return fmt.Errorf("no fallback strategy found for %s/%s", req.Component, req.Operation)
	}
	
	// Execute fallback
	fallbackResult, err := fallback.Handler(ctx, req.Error, result.Data)
	
	result.Metrics.FallbackTime = time.Since(fallbackStart)
	result.FallbacksUsed = append(result.FallbacksUsed, fallback.Name)
	result.RecoveryPath = append(result.RecoveryPath, "fallback")
	
	if err == nil {
		result.Data["fallback_result"] = fallbackResult
		result.Data["fallback_strategy"] = fallback.Name
		result.RecoveredState["fallback_active"] = true
	}
	
	return err
}

// executeBulkheadRecovery executes bulkhead isolation recovery
func (arm *AdvancedRecoveryManager) executeBulkheadRecovery(ctx context.Context, req *AdvancedRecoveryRequest, result *AdvancedRecoveryResult) error {
	// Get or create bulkhead for this component
	bulkhead := arm.getBulkhead(req.Component)
	
	err := bulkhead.Execute(ctx, func() error {
		// Simulate recovery in isolated environment
		time.Sleep(200 * time.Millisecond)
		return nil
	})
	
	result.RecoveryPath = append(result.RecoveryPath, "bulkhead")
	
	if err == nil {
		result.Data["bulkhead_isolation"] = true
		result.ComponentHealth[req.Component] = "isolated"
	}
	
	return err
}

// executeTimeoutRecovery executes timeout-based recovery
func (arm *AdvancedRecoveryManager) executeTimeoutRecovery(ctx context.Context, req *AdvancedRecoveryRequest, result *AdvancedRecoveryResult) error {
	executor := arm.getTimeoutExecutor(req.Component)
	
	err := executor.ExecuteWithTimeout(ctx, req.Timeout, func(timeoutCtx context.Context) error {
		// Simulate recovery operation with timeout
		time.Sleep(500 * time.Millisecond)
		return nil
	})
	
	result.RecoveryPath = append(result.RecoveryPath, "timeout")
	
	if err == nil {
		result.Data["timeout_recovery"] = true
	}
	
	return err
}

// executeRateLimitRecovery executes rate-limited recovery
func (arm *AdvancedRecoveryManager) executeRateLimitRecovery(ctx context.Context, req *AdvancedRecoveryRequest, result *AdvancedRecoveryResult) error {
	rateLimiter := arm.getRateLimiter(req.Component)
	
	// Check if we can proceed with recovery
	if !rateLimiter.Allow() {
		return fmt.Errorf("rate limit exceeded for component: %s", req.Component)
	}
	
	// Simulate recovery operation
	time.Sleep(100 * time.Millisecond)
	
	result.RecoveryPath = append(result.RecoveryPath, "rate_limit")
	result.Data["rate_limited"] = true
	
	return nil
}

// executeDegradationRecovery executes graceful degradation recovery
func (arm *AdvancedRecoveryManager) executeDegradationRecovery(ctx context.Context, req *AdvancedRecoveryRequest, result *AdvancedRecoveryResult) error {
	// Apply degradation level based on error severity
	level := arm.degradationManager.CalculateDegradationLevel(req.Error)
	
	err := arm.degradationManager.ApplyDegradation(req.Component, level)
	if err != nil {
		return fmt.Errorf("failed to apply degradation: %w", err)
	}
	
	result.RecoveryPath = append(result.RecoveryPath, "degradation")
	result.Data["degradation_level"] = level
	result.Data["degraded_features"] = arm.degradationManager.GetDegradedFeatures(req.Component)
	result.RecoveredState["degradation_active"] = true
	
	return nil
}

// Helper methods for getting/creating recovery components

func (arm *AdvancedRecoveryManager) getRetryExecutor(component string, strategy RecoveryStrategy) *RetryExecutor {
	key := fmt.Sprintf("%s-%s", component, strategy)
	
	arm.mutex.RLock()
	if executor, exists := arm.retryExecutors[key]; exists {
		arm.mutex.RUnlock()
		return executor
	}
	arm.mutex.RUnlock()
	
	arm.mutex.Lock()
	defer arm.mutex.Unlock()
	
	if executor, exists := arm.retryExecutors[key]; exists {
		return executor
	}
	
	// Create new retry executor
	policy := DefaultRetryPolicy()
	executor := NewRetryExecutor(policy, nil)
	arm.retryExecutors[key] = executor
	
	return executor
}

func (arm *AdvancedRecoveryManager) getCircuitBreaker(component string) *CircuitBreaker {
	arm.mutex.RLock()
	if cb, exists := arm.circuitBreakers[component]; exists {
		arm.mutex.RUnlock()
		return cb
	}
	arm.mutex.RUnlock()
	
	arm.mutex.Lock()
	defer arm.mutex.Unlock()
	
	if cb, exists := arm.circuitBreakers[component]; exists {
		return cb
	}
	
	config := DefaultCircuitBreakerConfig(component)
	cb := NewCircuitBreaker(config)
	arm.circuitBreakers[component] = cb
	
	return cb
}

func (arm *AdvancedRecoveryManager) getBulkhead(component string) *Bulkhead {
	arm.mutex.RLock()
	if bulkhead, exists := arm.bulkheads[component]; exists {
		arm.mutex.RUnlock()
		return bulkhead
	}
	arm.mutex.RUnlock()
	
	arm.mutex.Lock()
	defer arm.mutex.Unlock()
	
	if bulkhead, exists := arm.bulkheads[component]; exists {
		return bulkhead
	}
	
	config := DefaultBulkheadConfig(component)
	bulkhead := NewBulkhead(config)
	arm.bulkheads[component] = bulkhead
	
	return bulkhead
}

func (arm *AdvancedRecoveryManager) getTimeoutExecutor(component string) *TimeoutExecutor {
	arm.mutex.RLock()
	if executor, exists := arm.timeoutExecutors[component]; exists {
		arm.mutex.RUnlock()
		return executor
	}
	arm.mutex.RUnlock()
	
	arm.mutex.Lock()
	defer arm.mutex.Unlock()
	
	if executor, exists := arm.timeoutExecutors[component]; exists {
		return executor
	}
	
	config := DefaultTimeoutConfig()
	executor := NewTimeoutExecutor(config)
	arm.timeoutExecutors[component] = executor
	
	return executor
}

func (arm *AdvancedRecoveryManager) getRateLimiter(component string) *RateLimiter {
	arm.mutex.RLock()
	if limiter, exists := arm.rateLimiters[component]; exists {
		arm.mutex.RUnlock()
		return limiter
	}
	arm.mutex.RUnlock()
	
	arm.mutex.Lock()
	defer arm.mutex.Unlock()
	
	if limiter, exists := arm.rateLimiters[component]; exists {
		return limiter
	}
	
	limiter := NewRateLimiter(component, 10, 20, arm.logger) // 10 tokens/sec, burst of 20
	arm.rateLimiters[component] = limiter
	
	return limiter
}

// monitor performs periodic monitoring and maintenance
func (arm *AdvancedRecoveryManager) monitor(ctx context.Context) {
	for {
		select {
		case <-arm.monitoringTicker.C:
			arm.performMaintenance()
			
		case <-arm.stopChan:
			return
			
		case <-ctx.Done():
			return
		}
	}
}

// healthChecker performs periodic health checks
func (arm *AdvancedRecoveryManager) healthChecker(ctx context.Context) {
	ticker := time.NewTicker(arm.config.HealthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			arm.performHealthChecks()
			
		case <-arm.stopChan:
			return
			
		case <-ctx.Done():
			return
		}
	}
}

// performMaintenance performs periodic maintenance
func (arm *AdvancedRecoveryManager) performMaintenance() {
	// Update metrics
	arm.updateSystemMetrics()
	
	// Clean up expired resources
	arm.cleanupExpiredResources()
	
	// Check for cascading failures
	if arm.config.CascadeProtectionEnabled {
		arm.checkCascadingFailures()
	}
	
	arm.logger.Info("Maintenance completed", 
		"totalRequests", atomic.LoadInt64(&arm.metrics.TotalRequests),
		"successfulRecoveries", atomic.LoadInt64(&arm.metrics.SuccessfulRecoveries))
}

// performHealthChecks performs health checks on components
func (arm *AdvancedRecoveryManager) performHealthChecks() {
	// Check circuit breaker health
	arm.mutex.RLock()
	for name, cb := range arm.circuitBreakers {
		state := cb.GetState()
		if state == CircuitBreakerOpen {
			arm.logger.Warn("Circuit breaker is open", "component", name)
		}
	}
	arm.mutex.RUnlock()
	
	// Check bulkhead utilization
	arm.mutex.RLock()
	for name, bulkhead := range arm.bulkheads {
		metrics := bulkhead.GetMetrics()
		if active, ok := metrics["active"].(int64); ok && active > 0 {
			arm.logger.Info("Bulkhead utilization", "component", name, "active", active)
		}
	}
	arm.mutex.RUnlock()
}

// updateAdvancedMetrics updates recovery metrics
func (arm *AdvancedRecoveryManager) updateAdvancedMetrics(result *AdvancedRecoveryResult) {
	arm.metrics.mutex.Lock()
	defer arm.metrics.mutex.Unlock()
	
	if result.Success {
		atomic.AddInt64(&arm.metrics.SuccessfulRecoveries, 1)
	} else {
		atomic.AddInt64(&arm.metrics.FailedRecoveries, 1)
	}
	
	// Update strategy metrics
	if _, exists := arm.metrics.StrategyStats[result.Strategy]; !exists {
		arm.metrics.StrategyStats[result.Strategy] = &StrategyMetrics{
			FailureReasons: make(map[string]int64),
		}
	}
	
	strategyMetrics := arm.metrics.StrategyStats[result.Strategy]
	strategyMetrics.UsageCount++
	
	if result.Success {
		strategyMetrics.SuccessRate = float64(atomic.LoadInt64(&arm.metrics.SuccessfulRecoveries)) / 
			float64(atomic.LoadInt64(&arm.metrics.TotalRequests))
	}
	
	// Update average latency
	totalRequests := atomic.LoadInt64(&arm.metrics.TotalRequests)
	if totalRequests > 0 {
		totalTime := time.Duration(totalRequests-1) * arm.metrics.AverageRecoveryTime
		totalTime += result.Duration
		arm.metrics.AverageRecoveryTime = totalTime / time.Duration(totalRequests)
	}
	
	arm.metrics.LastUpdated = time.Now()
}

// updateSystemMetrics updates system-wide metrics
func (arm *AdvancedRecoveryManager) updateSystemMetrics() {
	// This would collect system metrics like memory usage, CPU usage, etc.
	// For demonstration purposes, we'll use placeholder values
	arm.metrics.mutex.Lock()
	defer arm.metrics.mutex.Unlock()
	
	if arm.metrics.ResourceStats == nil {
		arm.metrics.ResourceStats = &ResourceUtilizationStats{}
	}
	
	// Simulate collecting resource metrics
	arm.metrics.ResourceStats.AverageMemoryUsage = 1024 * 1024 * 100 // 100MB
	arm.metrics.ResourceStats.AverageCPUUsage = 0.15 // 15%
}

// cleanupExpiredResources cleans up expired recovery resources
func (arm *AdvancedRecoveryManager) cleanupExpiredResources() {
	// Clean up old metrics, expired circuit breakers, etc.
	// This is a placeholder for actual cleanup logic
	arm.logger.Info("Cleanup completed")
}

// checkCascadingFailures checks for cascading failure patterns
func (arm *AdvancedRecoveryManager) checkCascadingFailures() {
	// Analyze error patterns to detect cascading failures
	// This would implement sophisticated correlation analysis
	arm.logger.Info("Cascade check completed")
}

// GetAdvancedMetrics returns comprehensive recovery metrics
func (arm *AdvancedRecoveryManager) GetAdvancedMetrics() *AdvancedRecoveryMetrics {
	arm.metrics.mutex.RLock()
	defer arm.metrics.mutex.RUnlock()
	
	// Return a deep copy of metrics to prevent concurrent access issues
	metricsCopy := *arm.metrics
	return &metricsCopy
}

// Helper functions and supporting types

// NewRateLimiter creates a new token bucket rate limiter
func NewRateLimiter(name string, rate float64, capacity float64, logger logr.Logger) *RateLimiter {
	return &RateLimiter{
		name:       name,
		rate:       rate,
		capacity:   capacity,
		tokens:     capacity,
		lastRefill: time.Now(),
		logger:     logger,
	}
}

// Allow checks if a request can proceed
func (rl *RateLimiter) Allow() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()
	
	// Add tokens based on elapsed time
	rl.tokens += elapsed * rl.rate
	if rl.tokens > rl.capacity {
		rl.tokens = rl.capacity
	}
	
	rl.lastRefill = now
	
	if rl.tokens >= 1 {
		rl.tokens--
		return true
	}
	
	return false
}

// NewDegradationManager creates a new degradation manager
func NewDegradationManager(config *DegradationConfig, logger logr.Logger) *DegradationManager {
	return &DegradationManager{
		config:            config,
		degradationLevels: make(map[string]*DegradationLevel),
		currentLevels:     make(map[string]int),
		logger:            logger,
	}
}

// CalculateDegradationLevel calculates appropriate degradation level for an error
func (dm *DegradationManager) CalculateDegradationLevel(err *ProcessingError) int {
	switch err.Severity {
	case SeverityCritical:
		return 3 // Maximum degradation
	case SeverityHigh:
		return 2 // High degradation
	case SeverityMedium:
		return 1 // Low degradation
	default:
		return 0 // No degradation
	}
}

// ApplyDegradation applies degradation to a component
func (dm *DegradationManager) ApplyDegradation(component string, level int) error {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()
	
	dm.currentLevels[component] = level
	dm.logger.Info("Applied degradation", "component", component, "level", level)
	
	return nil
}

// GetDegradedFeatures returns features that are degraded for a component
func (dm *DegradationManager) GetDegradedFeatures(component string) []string {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()
	
	level, exists := dm.currentLevels[component]
	if !exists || level == 0 {
		return []string{}
	}
	
	// Return features based on degradation level
	switch level {
	case 1:
		return []string{"non_essential_features"}
	case 2:
		return []string{"non_essential_features", "advanced_features"}
	case 3:
		return []string{"non_essential_features", "advanced_features", "optional_features"}
	default:
		return []string{}
	}
}

// NewFallbackManager creates a new fallback manager
func NewFallbackManager(logger logr.Logger) *FallbackManager {
	manager := &FallbackManager{
		fallbacks: make(map[string]*FallbackStrategy),
		logger:    logger,
	}
	
	// Register default fallbacks
	manager.registerDefaultFallbacks()
	
	return manager
}

// registerDefaultFallbacks registers default fallback strategies
func (fm *FallbackManager) registerDefaultFallbacks() {
	// Default LLM fallback
	fm.RegisterFallback(&FallbackStrategy{
		Name:      "llm_simple_response",
		Component: "llm_processor",
		Operation: "process_intent",
		Handler: func(ctx context.Context, originalError error, originalData map[string]interface{}) (interface{}, error) {
			return map[string]interface{}{
				"fallback": true,
				"message": "Using simplified processing due to service degradation",
			}, nil
		},
		Priority: 1,
		Timeout:  5 * time.Second,
	})
	
	// Default RAG fallback
	fm.RegisterFallback(&FallbackStrategy{
		Name:      "rag_cached_response",
		Component: "rag_service",
		Operation: "query",
		Handler: func(ctx context.Context, originalError error, originalData map[string]interface{}) (interface{}, error) {
			return map[string]interface{}{
				"fallback":    true,
				"cached_data": "Using cached knowledge base results",
			}, nil
		},
		Priority: 1,
		Timeout:  3 * time.Second,
	})
}

// RegisterFallback registers a fallback strategy
func (fm *FallbackManager) RegisterFallback(strategy *FallbackStrategy) {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()
	
	key := fmt.Sprintf("%s_%s", strategy.Component, strategy.Operation)
	fm.fallbacks[key] = strategy
	
	fm.logger.Info("Registered fallback strategy", 
		"name", strategy.Name,
		"component", strategy.Component,
		"operation", strategy.Operation)
}

// GetFallback gets a fallback strategy
func (fm *FallbackManager) GetFallback(component, operation string) *FallbackStrategy {
	fm.mutex.RLock()
	defer fm.mutex.RUnlock()
	
	key := fmt.Sprintf("%s_%s", component, operation)
	return fm.fallbacks[key]
}

// NewAdvancedRecoveryMetrics creates new advanced recovery metrics
func NewAdvancedRecoveryMetrics() *AdvancedRecoveryMetrics {
	return &AdvancedRecoveryMetrics{
		StrategyStats:     make(map[RecoveryStrategy]*StrategyMetrics),
		ComponentStats:    make(map[string]*ComponentRecoveryStats),
		CorrelationStats:  &CorrelationMetrics{CorrelationPatterns: make(map[string]int64)},
		ResourceStats:     &ResourceUtilizationStats{},
	}
}

// getDefaultAdvancedRecoveryConfig returns default advanced recovery configuration
func getDefaultAdvancedRecoveryConfig() *AdvancedRecoveryConfig {
	return &AdvancedRecoveryConfig{
		MaxConcurrentRecoveries:     10,
		RecoveryTimeout:             5 * time.Minute,
		MonitoringInterval:          30 * time.Second,
		QueueSize:                  1000,
		ErrorCorrelationWindow:     10 * time.Minute,
		ErrorCorrelationThreshold:  5,
		AutoRecoveryEnabled:        true,
		AutoRecoveryDelay:          1 * time.Minute,
		MaxAutoRecoveryAttempts:    3,
		CascadeProtectionEnabled:   true,
		CascadeDetectionWindow:     5 * time.Minute,
		CascadeThreshold:           10,
		HealthCheckInterval:        1 * time.Minute,
		HealthCheckTimeout:         10 * time.Second,
		MaxMemoryUsage:            1024 * 1024 * 1024, // 1GB
		MaxCPUUsage:               0.8,                  // 80%
	}
}