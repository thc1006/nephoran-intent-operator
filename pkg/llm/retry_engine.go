package llm

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// RetryEngine provides advanced retry capabilities with multiple strategies
type RetryEngine struct {
	config  RetryEngineConfig
	logger  *slog.Logger
	tracer  trace.Tracer
	metrics *RetryMetrics

	// Retry strategies
	strategies map[string]RetryStrategy

	// Adaptive retry parameters
	adaptiveConfig AdaptiveRetryConfig

	// Circuit breaker integration
	circuitBreaker *AdvancedCircuitBreaker
}

// RetryEngineConfig holds configuration for the retry engine
type RetryEngineConfig struct {
	DefaultStrategy      string        `json:"default_strategy"`
	MaxRetries           int           `json:"max_retries"`
	BaseDelay            time.Duration `json:"base_delay"`
	MaxDelay             time.Duration `json:"max_delay"`
	JitterEnabled        bool          `json:"jitter_enabled"`
	BackoffMultiplier    float64       `json:"backoff_multiplier"`
	EnableAdaptive       bool          `json:"enable_adaptive"`
	TimeoutMultiplier    float64       `json:"timeout_multiplier"`
	EnableClassification bool          `json:"enable_classification"`
}

// AdaptiveRetryConfig holds configuration for adaptive retry behavior
type AdaptiveRetryConfig struct {
	LearningRate     float64 `json:"learning_rate"`
	SuccessThreshold float64 `json:"success_threshold"`
	FailureThreshold float64 `json:"failure_threshold"`
	WindowSize       int     `json:"window_size"`
	MinRetries       int     `json:"min_retries"`
	MaxRetries       int     `json:"max_retries"`
}

// RetryStrategy defines a retry strategy interface
type RetryStrategy interface {
	GetDelay(attempt int, lastError error) time.Duration
	ShouldRetry(attempt int, err error) bool
	GetName() string
}

// RetryMetrics tracks retry-related metrics
type RetryMetrics struct {
	totalRetries        int64
	successfulRetries   int64
	failedRetries       int64
	adaptiveAdjustments int64
	strategyUsage       map[string]int64
	errorClassification map[string]int64
	mutex               sync.RWMutex
}

// RetryContext holds context for a retry operation
type RetryContext struct {
	OperationID     string
	StartTime       time.Time
	LastAttemptTime time.Time
	Attempts        int
	Errors          []error
	Strategy        string
	Metadata        map[string]interface{}
}

// RetryResult holds the result of a retry operation
type RetryResult struct {
	Success             bool
	FinalError          error
	TotalAttempts       int
	TotalDuration       time.Duration
	Strategy            string
	AdaptiveAdjustments int
}

// Error classification types
type ErrorClass string

const (
	ErrorClassTransient      ErrorClass = "transient"
	ErrorClassPermanent      ErrorClass = "permanent"
	ErrorClassThrottling     ErrorClass = "throttling"
	ErrorClassTimeout        ErrorClass = "timeout"
	ErrorClassNetworking     ErrorClass = "networking"
	ErrorClassAuthentication ErrorClass = "authentication"
	ErrorClassRateLimit      ErrorClass = "rate_limit"
	ErrorClassUnknown        ErrorClass = "unknown"
)

// Retry strategies
type ExponentialBackoffStrategy struct {
	baseDelay     time.Duration
	maxDelay      time.Duration
	multiplier    float64
	maxRetries    int
	jitterEnabled bool
}

type LinearBackoffStrategy struct {
	baseDelay  time.Duration
	maxDelay   time.Duration
	increment  time.Duration
	maxRetries int
}

type FixedDelayStrategy struct {
	delay      time.Duration
	maxRetries int
}

type AdaptiveStrategy struct {
	engine         *RetryEngine
	baseDelay      time.Duration
	maxDelay       time.Duration
	maxRetries     int
	successHistory []bool
	mutex          sync.RWMutex
}

// NewRetryEngine creates a new retry engine
func NewRetryEngine(config RetryEngineConfig, circuitBreaker *AdvancedCircuitBreaker) *RetryEngine {
	if config.DefaultStrategy == "" {
		config.DefaultStrategy = "exponential"
	}

	engine := &RetryEngine{
		config:         config,
		logger:         slog.Default().With("component", "retry-engine"),
		tracer:         otel.Tracer("nephoran-intent-operator/retry"),
		circuitBreaker: circuitBreaker,
		strategies:     make(map[string]RetryStrategy),
		metrics: &RetryMetrics{
			strategyUsage:       make(map[string]int64),
			errorClassification: make(map[string]int64),
		},
		adaptiveConfig: AdaptiveRetryConfig{
			LearningRate:     0.1,
			SuccessThreshold: 0.8,
			FailureThreshold: 0.3,
			WindowSize:       100,
			MinRetries:       1,
			MaxRetries:       10,
		},
	}

	// Initialize retry strategies
	engine.initializeStrategies()

	return engine
}

// initializeStrategies sets up the available retry strategies
func (re *RetryEngine) initializeStrategies() {
	// Exponential backoff strategy
	re.strategies["exponential"] = &ExponentialBackoffStrategy{
		baseDelay:     re.config.BaseDelay,
		maxDelay:      re.config.MaxDelay,
		multiplier:    re.config.BackoffMultiplier,
		maxRetries:    re.config.MaxRetries,
		jitterEnabled: re.config.JitterEnabled,
	}

	// Linear backoff strategy
	re.strategies["linear"] = &LinearBackoffStrategy{
		baseDelay:  re.config.BaseDelay,
		maxDelay:   re.config.MaxDelay,
		increment:  re.config.BaseDelay,
		maxRetries: re.config.MaxRetries,
	}

	// Fixed delay strategy
	re.strategies["fixed"] = &FixedDelayStrategy{
		delay:      re.config.BaseDelay,
		maxRetries: re.config.MaxRetries,
	}

	// Adaptive strategy
	if re.config.EnableAdaptive {
		re.strategies["adaptive"] = &AdaptiveStrategy{
			engine:         re,
			baseDelay:      re.config.BaseDelay,
			maxDelay:       re.config.MaxDelay,
			maxRetries:     re.config.MaxRetries,
			successHistory: make([]bool, 0, re.adaptiveConfig.WindowSize),
		}
	}
}

// ExecuteWithRetry executes an operation with retry logic
func (re *RetryEngine) ExecuteWithRetry(ctx context.Context, operation func() error) (*RetryResult, error) {
	return re.ExecuteWithRetryAndStrategy(ctx, operation, re.config.DefaultStrategy)
}

// ExecuteWithRetryAndStrategy executes an operation with a specific retry strategy
func (re *RetryEngine) ExecuteWithRetryAndStrategy(ctx context.Context, operation func() error, strategyName string) (*RetryResult, error) {
	ctx, span := re.tracer.Start(ctx, "retry_engine.execute")
	defer span.End()

	strategy, exists := re.strategies[strategyName]
	if !exists {
		return nil, fmt.Errorf("unknown retry strategy: %s", strategyName)
	}

	retryCtx := &RetryContext{
		OperationID: generateOperationID(),
		StartTime:   time.Now(),
		Strategy:    strategyName,
		Metadata:    make(map[string]interface{}),
	}

	span.SetAttributes(
		attribute.String("retry.strategy", strategyName),
		attribute.String("retry.operation_id", retryCtx.OperationID),
	)

	re.logger.Debug("Starting retry operation",
		"operation_id", retryCtx.OperationID,
		"strategy", strategyName,
	)

	var lastError error
	adaptiveAdjustments := 0

	for attempt := 0; attempt <= re.config.MaxRetries; attempt++ {
		retryCtx.Attempts = attempt + 1
		retryCtx.LastAttemptTime = time.Now()

		// Check if circuit breaker allows the request
		if re.circuitBreaker != nil && !re.circuitBreaker.IsRequestAllowed() {
			lastError = &CircuitBreakerError{
				State:   re.circuitBreaker.GetState(),
				Message: "circuit breaker is open",
			}
			break
		}

		// Execute the operation
		err := operation()
		retryCtx.Errors = append(retryCtx.Errors, err)

		if err == nil {
			// Success
			re.updateMetrics(strategyName, true, attempt, adaptiveAdjustments)

			result := &RetryResult{
				Success:             true,
				TotalAttempts:       attempt + 1,
				TotalDuration:       time.Since(retryCtx.StartTime),
				Strategy:            strategyName,
				AdaptiveAdjustments: adaptiveAdjustments,
			}

			span.SetAttributes(
				attribute.Bool("retry.success", true),
				attribute.Int("retry.attempts", attempt+1),
			)

			re.logger.Info("Retry operation succeeded",
				"operation_id", retryCtx.OperationID,
				"attempts", attempt+1,
				"duration", result.TotalDuration,
			)

			return result, nil
		}

		lastError = err

		// Classify the error
		errorClass := re.classifyError(err)
		re.updateErrorClassification(errorClass)

		span.SetAttributes(
			attribute.String("retry.error_class", string(errorClass)),
			attribute.String("retry.last_error", err.Error()),
		)

		// Check if we should retry
		if !strategy.ShouldRetry(attempt, err) {
			re.logger.Debug("Strategy determined no retry should be attempted",
				"operation_id", retryCtx.OperationID,
				"attempt", attempt+1,
				"error", err.Error(),
			)
			break
		}

		// Don't wait after the last attempt
		if attempt >= re.config.MaxRetries {
			break
		}

		// Calculate delay
		delay := strategy.GetDelay(attempt, err)

		// Apply adaptive adjustments
		if re.config.EnableAdaptive && strategyName == "adaptive" {
			originalDelay := delay
			delay = re.applyAdaptiveAdjustment(delay, errorClass, retryCtx)
			if delay != originalDelay {
				adaptiveAdjustments++
			}
		}

		re.logger.Debug("Retrying operation",
			"operation_id", retryCtx.OperationID,
			"attempt", attempt+1,
			"delay", delay,
			"error", err.Error(),
		)

		// Wait for the delay
		select {
		case <-ctx.Done():
			return &RetryResult{
				Success:       false,
				FinalError:    ctx.Err(),
				TotalAttempts: attempt + 1,
				TotalDuration: time.Since(retryCtx.StartTime),
				Strategy:      strategyName,
			}, ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	// All retries exhausted
	re.updateMetrics(strategyName, false, retryCtx.Attempts, adaptiveAdjustments)

	result := &RetryResult{
		Success:             false,
		FinalError:          lastError,
		TotalAttempts:       retryCtx.Attempts,
		TotalDuration:       time.Since(retryCtx.StartTime),
		Strategy:            strategyName,
		AdaptiveAdjustments: adaptiveAdjustments,
	}

	span.SetAttributes(
		attribute.Bool("retry.success", false),
		attribute.Int("retry.attempts", retryCtx.Attempts),
	)

	re.logger.Warn("Retry operation failed after all attempts",
		"operation_id", retryCtx.OperationID,
		"attempts", retryCtx.Attempts,
		"duration", result.TotalDuration,
		"final_error", lastError.Error(),
	)

	return result, lastError
}

// classifyError classifies an error for retry decision making
func (re *RetryEngine) classifyError(err error) ErrorClass {
	if err == nil {
		return ErrorClassUnknown
	}

	errorStr := strings.ToLower(err.Error())

	// Check for specific error patterns
	switch {
	case strings.Contains(errorStr, "timeout") || strings.Contains(errorStr, "deadline exceeded"):
		return ErrorClassTimeout
	case strings.Contains(errorStr, "rate limit") || strings.Contains(errorStr, "too many requests"):
		return ErrorClassRateLimit
	case strings.Contains(errorStr, "throttle") || strings.Contains(errorStr, "throttling"):
		return ErrorClassThrottling
	case strings.Contains(errorStr, "connection") || strings.Contains(errorStr, "network") || strings.Contains(errorStr, "dns"):
		return ErrorClassNetworking
	case strings.Contains(errorStr, "unauthorized") || strings.Contains(errorStr, "authentication") || strings.Contains(errorStr, "token"):
		return ErrorClassAuthentication
	case strings.Contains(errorStr, "temporary") || strings.Contains(errorStr, "service unavailable") || strings.Contains(errorStr, "bad gateway"):
		return ErrorClassTransient
	case strings.Contains(errorStr, "invalid") || strings.Contains(errorStr, "bad request") || strings.Contains(errorStr, "malformed"):
		return ErrorClassPermanent
	default:
		return ErrorClassUnknown
	}
}

// applyAdaptiveAdjustment applies adaptive adjustment to the delay
func (re *RetryEngine) applyAdaptiveAdjustment(baseDelay time.Duration, errorClass ErrorClass, ctx *RetryContext) time.Duration {
	// Adjust delay based on error class
	multiplier := 1.0

	switch errorClass {
	case ErrorClassRateLimit, ErrorClassThrottling:
		multiplier = 2.0 // Longer delays for rate limiting
	case ErrorClassNetworking, ErrorClassTimeout:
		multiplier = 1.5 // Moderate delays for network issues
	case ErrorClassAuthentication, ErrorClassPermanent:
		multiplier = 0.5 // Shorter delays for likely permanent issues
	case ErrorClassTransient:
		multiplier = 1.0 // Normal delays for transient issues
	}

	adjustedDelay := time.Duration(float64(baseDelay) * multiplier)

	// Ensure bounds
	if adjustedDelay > re.config.MaxDelay {
		adjustedDelay = re.config.MaxDelay
	}
	if adjustedDelay < time.Millisecond {
		adjustedDelay = time.Millisecond
	}

	return adjustedDelay
}

// updateMetrics updates retry metrics
func (re *RetryEngine) updateMetrics(strategy string, success bool, attempts, adaptiveAdjustments int) {
	re.metrics.mutex.Lock()
	defer re.metrics.mutex.Unlock()

	re.metrics.totalRetries += int64(attempts - 1) // Don't count the initial attempt
	re.metrics.strategyUsage[strategy]++
	re.metrics.adaptiveAdjustments += int64(adaptiveAdjustments)

	if success {
		re.metrics.successfulRetries++
	} else {
		re.metrics.failedRetries++
	}
}

// updateErrorClassification updates error classification metrics
func (re *RetryEngine) updateErrorClassification(errorClass ErrorClass) {
	re.metrics.mutex.Lock()
	defer re.metrics.mutex.Unlock()

	re.metrics.errorClassification[string(errorClass)]++
}

// GetMetrics returns current retry metrics
func (re *RetryEngine) GetMetrics() RetryEngineMetrics {
	re.metrics.mutex.RLock()
	defer re.metrics.mutex.RUnlock()

	strategyUsage := make(map[string]int64)
	for k, v := range re.metrics.strategyUsage {
		strategyUsage[k] = v
	}

	errorClassification := make(map[string]int64)
	for k, v := range re.metrics.errorClassification {
		errorClassification[k] = v
	}

	return RetryEngineMetrics{
		TotalRetries:        re.metrics.totalRetries,
		SuccessfulRetries:   re.metrics.successfulRetries,
		FailedRetries:       re.metrics.failedRetries,
		AdaptiveAdjustments: re.metrics.adaptiveAdjustments,
		StrategyUsage:       strategyUsage,
		ErrorClassification: errorClassification,
	}
}

// RetryEngineMetrics holds retry engine metrics
type RetryEngineMetrics struct {
	TotalRetries        int64            `json:"total_retries"`
	SuccessfulRetries   int64            `json:"successful_retries"`
	FailedRetries       int64            `json:"failed_retries"`
	AdaptiveAdjustments int64            `json:"adaptive_adjustments"`
	StrategyUsage       map[string]int64 `json:"strategy_usage"`
	ErrorClassification map[string]int64 `json:"error_classification"`
}

// Strategy implementations

// ExponentialBackoffStrategy implementation
func (s *ExponentialBackoffStrategy) GetDelay(attempt int, lastError error) time.Duration {
	delay := time.Duration(float64(s.baseDelay) * math.Pow(s.multiplier, float64(attempt)))

	if s.jitterEnabled {
		// Add jitter (±25% of delay)
		jitter := time.Duration(float64(delay) * 0.25 * (2.0*rand.Float64() - 1.0))
		delay += jitter
	}

	if delay > s.maxDelay {
		delay = s.maxDelay
	}

	return delay
}

func (s *ExponentialBackoffStrategy) ShouldRetry(attempt int, err error) bool {
	return attempt < s.maxRetries
}

func (s *ExponentialBackoffStrategy) GetName() string {
	return "exponential"
}

// LinearBackoffStrategy implementation
func (s *LinearBackoffStrategy) GetDelay(attempt int, lastError error) time.Duration {
	delay := s.baseDelay + time.Duration(attempt)*s.increment
	if delay > s.maxDelay {
		delay = s.maxDelay
	}
	return delay
}

func (s *LinearBackoffStrategy) ShouldRetry(attempt int, err error) bool {
	return attempt < s.maxRetries
}

func (s *LinearBackoffStrategy) GetName() string {
	return "linear"
}

// FixedDelayStrategy implementation
func (s *FixedDelayStrategy) GetDelay(attempt int, lastError error) time.Duration {
	return s.delay
}

func (s *FixedDelayStrategy) ShouldRetry(attempt int, err error) bool {
	return attempt < s.maxRetries
}

func (s *FixedDelayStrategy) GetName() string {
	return "fixed"
}

// AdaptiveStrategy implementation
func (s *AdaptiveStrategy) GetDelay(attempt int, lastError error) time.Duration {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Calculate success rate from recent history
	successRate := s.calculateSuccessRate()

	// Adjust delay based on success rate
	multiplier := 1.0
	if successRate < s.engine.adaptiveConfig.FailureThreshold {
		// High failure rate - increase delay
		multiplier = 2.0
	} else if successRate > s.engine.adaptiveConfig.SuccessThreshold {
		// High success rate - decrease delay
		multiplier = 0.5
	}

	delay := time.Duration(float64(s.baseDelay) * math.Pow(multiplier, float64(attempt)))
	if delay > s.maxDelay {
		delay = s.maxDelay
	}

	return delay
}

func (s *AdaptiveStrategy) ShouldRetry(attempt int, err error) bool {
	return attempt < s.maxRetries
}

func (s *AdaptiveStrategy) GetName() string {
	return "adaptive"
}

func (s *AdaptiveStrategy) calculateSuccessRate() float64 {
	if len(s.successHistory) == 0 {
		return 0.5 // Neutral assumption
	}

	successes := 0
	for _, success := range s.successHistory {
		if success {
			successes++
		}
	}

	return float64(successes) / float64(len(s.successHistory))
}

func (s *AdaptiveStrategy) recordResult(success bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.successHistory = append(s.successHistory, success)

	// Maintain window size
	if len(s.successHistory) > s.engine.adaptiveConfig.WindowSize {
		s.successHistory = s.successHistory[1:]
	}
}

// Helper functions
func generateOperationID() string {
	return fmt.Sprintf("op_%d", time.Now().UnixNano())
}
