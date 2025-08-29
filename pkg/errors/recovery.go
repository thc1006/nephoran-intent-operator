package errors

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// RetryPolicy defines how retries should be performed.

type RetryPolicy struct {
	MaxRetries int `json:"max_retries"`

	InitialDelay time.Duration `json:"initial_delay"`

	MaxDelay time.Duration `json:"max_delay"`

	BackoffMultiplier float64 `json:"backoff_multiplier"`

	Jitter bool `json:"jitter"`

	RetryableErrors []ErrorType `json:"retryable_errors"`

	NonRetryableErrors []ErrorType `json:"non_retryable_errors"`
}

// DefaultRetryPolicy returns a sensible default retry policy.

func DefaultRetryPolicy() *RetryPolicy {

	return &RetryPolicy{

		MaxRetries: 3,

		InitialDelay: time.Millisecond * 100,

		MaxDelay: time.Second * 30,

		BackoffMultiplier: 2.0,

		Jitter: true,

		RetryableErrors: []ErrorType{

			ErrorTypeNetwork,

			ErrorTypeTimeout,

			ErrorTypeExternal,

			ErrorTypeRateLimit,

			ErrorTypeResource,

			ErrorTypeDatabase,
		},

		NonRetryableErrors: []ErrorType{

			ErrorTypeValidation,

			ErrorTypeAuth,

			ErrorTypeUnauthorized,

			ErrorTypeForbidden,

			ErrorTypeNotFound,
		},
	}

}

// CircuitBreakerState represents the state of a circuit breaker.

type CircuitBreakerState int32

const (

	// CircuitBreakerClosed holds circuitbreakerclosed value.

	CircuitBreakerClosed CircuitBreakerState = iota

	// CircuitBreakerOpen holds circuitbreakeropen value.

	CircuitBreakerOpen

	// CircuitBreakerHalfOpen holds circuitbreakerhalfopen value.

	CircuitBreakerHalfOpen
)

// String performs string operation.

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

// CircuitBreakerConfig configures circuit breaker behavior.

type CircuitBreakerConfig struct {
	Name string `json:"name"`

	MaxFailures int `json:"max_failures"`

	Timeout time.Duration `json:"timeout"`

	ResetTimeout time.Duration `json:"reset_timeout"`

	HalfOpenMaxCalls int `json:"half_open_max_calls"`

	FailureThresholdRatio float64 `json:"failure_threshold_ratio"`

	MinRequestThreshold int `json:"min_request_threshold"`

	SlidingWindowSize time.Duration `json:"sliding_window_size"`

	ConsecutiveFailures bool `json:"consecutive_failures"`
}

// DefaultCircuitBreakerConfig returns sensible defaults.

func DefaultCircuitBreakerConfig(name string) *CircuitBreakerConfig {

	return &CircuitBreakerConfig{

		Name: name,

		MaxFailures: 5,

		Timeout: time.Second * 60,

		ResetTimeout: time.Second * 30,

		HalfOpenMaxCalls: 3,

		FailureThresholdRatio: 0.5,

		MinRequestThreshold: 10,

		SlidingWindowSize: time.Minute * 5,

		ConsecutiveFailures: false,
	}

}

// CircuitBreaker implements the circuit breaker pattern for error recovery.

type CircuitBreaker struct {
	config *CircuitBreakerConfig

	state int32 // CircuitBreakerState

	failures int64

	requests int64

	lastFailureTime int64

	halfOpenCalls int64

	mu sync.RWMutex

	onStateChange func(string, CircuitBreakerState, CircuitBreakerState)

	// Sliding window for failure tracking.

	failureWindow []time.Time

	windowMutex sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker.

func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {

	if config == nil {

		config = DefaultCircuitBreakerConfig("default")

	}

	return &CircuitBreaker{

		config: config,

		state: int32(CircuitBreakerClosed),

		failureWindow: make([]time.Time, 0),
	}

}

// SetStateChangeCallback sets a callback for state changes.

func (cb *CircuitBreaker) SetStateChangeCallback(callback func(string, CircuitBreakerState, CircuitBreakerState)) {

	cb.mu.Lock()

	defer cb.mu.Unlock()

	cb.onStateChange = callback

}

// Execute runs a function with circuit breaker protection.

func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {

	// Check if we can proceed.

	if err := cb.canProceed(); err != nil {

		return err

	}

	// Execute the function.

	atomic.AddInt64(&cb.requests, 1)

	start := time.Now()

	err := fn()

	latency := time.Since(start)

	// Record the result.

	cb.recordResult(err, latency)

	return err

}

// canProceed checks if the circuit breaker allows the request to proceed.

func (cb *CircuitBreaker) canProceed() error {

	state := CircuitBreakerState(atomic.LoadInt32(&cb.state))

	switch state {

	case CircuitBreakerClosed:

		return nil

	case CircuitBreakerOpen:

		lastFailure := time.Unix(0, atomic.LoadInt64(&cb.lastFailureTime))

		if time.Since(lastFailure) > cb.config.ResetTimeout {

			// Try to transition to half-open.

			if atomic.CompareAndSwapInt32(&cb.state, int32(CircuitBreakerOpen), int32(CircuitBreakerHalfOpen)) {

				atomic.StoreInt64(&cb.halfOpenCalls, 0)

				cb.notifyStateChange(CircuitBreakerOpen, CircuitBreakerHalfOpen)

			}

			return nil

		}

		return cb.createCircuitBreakerError("circuit breaker is open")

	case CircuitBreakerHalfOpen:

		if atomic.LoadInt64(&cb.halfOpenCalls) >= int64(cb.config.HalfOpenMaxCalls) {

			return cb.createCircuitBreakerError("circuit breaker half-open call limit exceeded")

		}

		atomic.AddInt64(&cb.halfOpenCalls, 1)

		return nil

	default:

		return cb.createCircuitBreakerError("unknown circuit breaker state")

	}

}

// recordResult records the result of a function execution.

func (cb *CircuitBreaker) recordResult(err error, latency time.Duration) {

	state := CircuitBreakerState(atomic.LoadInt32(&cb.state))

	if err != nil {

		cb.recordFailure()

		switch state {

		case CircuitBreakerClosed:

			if cb.shouldTripToOpen() {

				if atomic.CompareAndSwapInt32(&cb.state, int32(CircuitBreakerClosed), int32(CircuitBreakerOpen)) {

					atomic.StoreInt64(&cb.lastFailureTime, time.Now().UnixNano())

					cb.notifyStateChange(CircuitBreakerClosed, CircuitBreakerOpen)

				}

			}

		case CircuitBreakerHalfOpen:

			// On failure in half-open state, go back to open.

			if atomic.CompareAndSwapInt32(&cb.state, int32(CircuitBreakerHalfOpen), int32(CircuitBreakerOpen)) {

				atomic.StoreInt64(&cb.lastFailureTime, time.Now().UnixNano())

				cb.notifyStateChange(CircuitBreakerHalfOpen, CircuitBreakerOpen)

			}

		}

	} else {

		cb.recordSuccess()

		if state == CircuitBreakerHalfOpen {

			// Check if we should close the circuit.

			if atomic.LoadInt64(&cb.halfOpenCalls) >= int64(cb.config.HalfOpenMaxCalls) {

				if atomic.CompareAndSwapInt32(&cb.state, int32(CircuitBreakerHalfOpen), int32(CircuitBreakerClosed)) {

					atomic.StoreInt64(&cb.failures, 0)

					cb.notifyStateChange(CircuitBreakerHalfOpen, CircuitBreakerClosed)

					cb.clearFailureWindow()

				}

			}

		}

	}

}

// recordFailure records a failure.

func (cb *CircuitBreaker) recordFailure() {

	atomic.AddInt64(&cb.failures, 1)

	// Add to sliding window.

	cb.windowMutex.Lock()

	now := time.Now()

	cb.failureWindow = append(cb.failureWindow, now)

	// Clean old entries from sliding window.

	cutoff := now.Add(-cb.config.SlidingWindowSize)

	i := 0

	for i < len(cb.failureWindow) && cb.failureWindow[i].Before(cutoff) {

		i++

	}

	cb.failureWindow = cb.failureWindow[i:]

	cb.windowMutex.Unlock()

}

// recordSuccess records a successful operation.

func (cb *CircuitBreaker) recordSuccess() {

	// Reset consecutive failures if configured.

	if cb.config.ConsecutiveFailures {

		atomic.StoreInt64(&cb.failures, 0)

	}

}

// shouldTripToOpen determines if the circuit breaker should trip to open state.

func (cb *CircuitBreaker) shouldTripToOpen() bool {

	failures := atomic.LoadInt64(&cb.failures)

	requests := atomic.LoadInt64(&cb.requests)

	// Check consecutive failures.

	if cb.config.ConsecutiveFailures {

		return failures >= int64(cb.config.MaxFailures)

	}

	// Check failure ratio.

	if requests < int64(cb.config.MinRequestThreshold) {

		return false

	}

	cb.windowMutex.RLock()

	windowFailures := int64(len(cb.failureWindow))

	cb.windowMutex.RUnlock()

	failureRatio := float64(windowFailures) / float64(requests)

	return failureRatio >= cb.config.FailureThresholdRatio

}

// clearFailureWindow clears the failure tracking window.

func (cb *CircuitBreaker) clearFailureWindow() {

	cb.windowMutex.Lock()

	cb.failureWindow = cb.failureWindow[:0]

	cb.windowMutex.Unlock()

}

// createCircuitBreakerError creates a circuit breaker specific error.

func (cb *CircuitBreaker) createCircuitBreakerError(message string) error {

	return &ServiceError{

		Type: ErrorTypeResource,

		Code: "circuit_breaker_open",

		Message: message,

		Service: "circuit-breaker",

		Operation: "execute",

		Component: cb.config.Name,

		Category: CategorySystem,

		Severity: SeverityMedium,

		Impact: ImpactModerate,

		Retryable: true,

		Temporary: true,

		HTTPStatus: 503,

		Timestamp: time.Now(),

		CircuitBreaker: cb.config.Name,

		RetryAfter: cb.config.ResetTimeout,

		Metadata: map[string]interface{}{

			"circuit_breaker_state": CircuitBreakerState(atomic.LoadInt32(&cb.state)).String(),

			"failures": atomic.LoadInt64(&cb.failures),

			"requests": atomic.LoadInt64(&cb.requests),
		},
	}

}

// notifyStateChange notifies about state changes.

func (cb *CircuitBreaker) notifyStateChange(from, to CircuitBreakerState) {

	cb.mu.RLock()

	callback := cb.onStateChange

	cb.mu.RUnlock()

	if callback != nil {

		callback(cb.config.Name, from, to)

	}

}

// GetState returns the current circuit breaker state.

func (cb *CircuitBreaker) GetState() CircuitBreakerState {

	return CircuitBreakerState(atomic.LoadInt32(&cb.state))

}

// GetMetrics returns current circuit breaker metrics.

func (cb *CircuitBreaker) GetMetrics() map[string]interface{} {

	cb.windowMutex.RLock()

	windowFailures := len(cb.failureWindow)

	cb.windowMutex.RUnlock()

	return map[string]interface{}{

		"name": cb.config.Name,

		"state": cb.GetState().String(),

		"failures": atomic.LoadInt64(&cb.failures),

		"requests": atomic.LoadInt64(&cb.requests),

		"window_failures": windowFailures,

		"half_open_calls": atomic.LoadInt64(&cb.halfOpenCalls),
	}

}

// Reset resets the circuit breaker to its initial state.

func (cb *CircuitBreaker) Reset() {

	atomic.StoreInt32(&cb.state, int32(CircuitBreakerClosed))

	atomic.StoreInt64(&cb.failures, 0)

	atomic.StoreInt64(&cb.requests, 0)

	atomic.StoreInt64(&cb.lastFailureTime, 0)

	atomic.StoreInt64(&cb.halfOpenCalls, 0)

	cb.clearFailureWindow()

}

// RetryableOperation represents an operation that can be retried.

type RetryableOperation func() error

// RetryExecutor executes operations with retry logic.

type RetryExecutor struct {
	policy *RetryPolicy

	circuitBreaker *CircuitBreaker
}

// NewRetryExecutor creates a new retry executor.

func NewRetryExecutor(policy *RetryPolicy, circuitBreaker *CircuitBreaker) *RetryExecutor {

	if policy == nil {

		policy = DefaultRetryPolicy()

	}

	return &RetryExecutor{

		policy: policy,

		circuitBreaker: circuitBreaker,
	}

}

// Execute executes an operation with retry logic and circuit breaker protection.

func (re *RetryExecutor) Execute(ctx context.Context, operation RetryableOperation) error {

	if re.circuitBreaker != nil {

		return re.circuitBreaker.Execute(ctx, func() error {

			return re.executeWithRetry(ctx, operation)

		})

	}

	return re.executeWithRetry(ctx, operation)

}

// executeWithRetry executes an operation with retry logic.

func (re *RetryExecutor) executeWithRetry(ctx context.Context, operation RetryableOperation) error {

	var lastError error

	delay := re.policy.InitialDelay

	for attempt := 0; attempt <= re.policy.MaxRetries; attempt++ {

		select {

		case <-ctx.Done():

			return ctx.Err()

		default:

		}

		// Execute the operation.

		err := operation()

		if err == nil {

			return nil // Success

		}

		lastError = err

		// Check if we should retry.

		if !re.shouldRetry(err, attempt) {

			break

		}

		// Don't sleep after the last attempt.

		if attempt < re.policy.MaxRetries {

			select {

			case <-time.After(delay):

			case <-ctx.Done():

				return ctx.Err()

			}

			// Calculate next delay.

			delay = re.calculateNextDelay(delay, attempt)

		}

	}

	// Enhance the error with retry information.

	if serviceErr, ok := lastError.(*ServiceError); ok {

		serviceErr.RetryCount = re.policy.MaxRetries

		serviceErr.AddTag("retry_exhausted")

		return serviceErr

	}

	// Wrap non-ServiceError in a ServiceError.

	return &ServiceError{

		Type: ErrorTypeInternal,

		Code: "retry_exhausted",

		Message: fmt.Sprintf("Operation failed after %d retries", re.policy.MaxRetries),

		Service: "retry-executor",

		Operation: "execute",

		Category: CategorySystem,

		Severity: SeverityHigh,

		Impact: ImpactSevere,

		Retryable: false,

		Temporary: false,

		HTTPStatus: 500,

		Timestamp: time.Now(),

		Cause: lastError,

		RetryCount: re.policy.MaxRetries,

		Tags: []string{"retry_exhausted"},
	}

}

// shouldRetry determines if an error is retryable.

func (re *RetryExecutor) shouldRetry(err error, attempt int) bool {

	if attempt >= re.policy.MaxRetries {

		return false

	}

	// Check for non-retryable errors first.

	for _, nonRetryable := range re.policy.NonRetryableErrors {

		if isErrorOfType(err, nonRetryable) {

			return false

		}

	}

	// Check for explicitly retryable errors.

	for _, retryable := range re.policy.RetryableErrors {

		if isErrorOfType(err, retryable) {

			return true

		}

	}

	// Default to checking if it's a ServiceError with Retryable flag.

	if serviceErr, ok := err.(*ServiceError); ok {

		return serviceErr.IsRetryable()

	}

	return false

}

// calculateNextDelay calculates the next delay using exponential backoff with optional jitter.

func (re *RetryExecutor) calculateNextDelay(currentDelay time.Duration, attempt int) time.Duration {

	nextDelay := time.Duration(float64(currentDelay) * re.policy.BackoffMultiplier)

	if nextDelay > re.policy.MaxDelay {

		nextDelay = re.policy.MaxDelay

	}

	if re.policy.Jitter {

		// Add jitter to prevent thundering herd.

		jitter := time.Duration(rand.Int63n(int64(nextDelay) / 2))

		nextDelay = nextDelay + jitter

	}

	return nextDelay

}

// isErrorOfType checks if an error is of a specific type.

func isErrorOfType(err error, errorType ErrorType) bool {

	if serviceErr, ok := err.(*ServiceError); ok {

		return serviceErr.Type == errorType

	}

	return false

}

// BulkheadConfig configures bulkhead isolation patterns.

type BulkheadConfig struct {
	Name string `json:"name"`

	MaxConcurrent int `json:"max_concurrent"`

	QueueSize int `json:"queue_size"`

	Timeout time.Duration `json:"timeout"`

	RejectOverflow bool `json:"reject_overflow"`
}

// DefaultBulkheadConfig returns sensible defaults.

func DefaultBulkheadConfig(name string) *BulkheadConfig {

	return &BulkheadConfig{

		Name: name,

		MaxConcurrent: 10,

		QueueSize: 100,

		Timeout: time.Second * 30,

		RejectOverflow: true,
	}

}

// Bulkhead implements the bulkhead pattern for resource isolation.

type Bulkhead struct {
	config *BulkheadConfig

	semaphore chan struct{}

	queue chan func()

	active int64

	queued int64

	rejected int64

	completed int64

	mu sync.RWMutex
}

// NewBulkhead creates a new bulkhead.

func NewBulkhead(config *BulkheadConfig) *Bulkhead {

	if config == nil {

		config = DefaultBulkheadConfig("default")

	}

	b := &Bulkhead{

		config: config,

		semaphore: make(chan struct{}, config.MaxConcurrent),

		queue: make(chan func(), config.QueueSize),
	}

	// Start worker goroutines.

	for range config.MaxConcurrent {

		go b.worker()

	}

	return b

}

// Execute executes a function within the bulkhead.

func (b *Bulkhead) Execute(ctx context.Context, fn func() error) error {

	resultChan := make(chan error, 1)

	work := func() {

		atomic.AddInt64(&b.active, 1)

		defer atomic.AddInt64(&b.active, -1)

		defer atomic.AddInt64(&b.completed, 1)

		resultChan <- fn()

	}

	// Try to queue the work.

	select {

	case b.queue <- work:

		atomic.AddInt64(&b.queued, 1)

	default:

		// Queue is full.

		atomic.AddInt64(&b.rejected, 1)

		if b.config.RejectOverflow {

			return b.createBulkheadError("bulkhead queue is full")

		}

		// Block until we can queue (with timeout).

		select {

		case b.queue <- work:

			atomic.AddInt64(&b.queued, 1)

		case <-time.After(b.config.Timeout):

			return b.createBulkheadError("bulkhead queue timeout")

		case <-ctx.Done():

			return ctx.Err()

		}

	}

	// Wait for result.

	select {

	case err := <-resultChan:

		return err

	case <-ctx.Done():

		return ctx.Err()

	case <-time.After(b.config.Timeout):

		return b.createBulkheadError("bulkhead execution timeout")

	}

}

// worker processes work from the queue.

func (b *Bulkhead) worker() {

	for work := range b.queue {

		atomic.AddInt64(&b.queued, -1)

		work()

	}

}

// createBulkheadError creates a bulkhead-specific error.

func (b *Bulkhead) createBulkheadError(message string) error {

	return &ServiceError{

		Type: ErrorTypeResource,

		Code: "bulkhead_limit",

		Message: message,

		Service: "bulkhead",

		Operation: "execute",

		Component: b.config.Name,

		Category: CategorySystem,

		Severity: SeverityMedium,

		Impact: ImpactModerate,

		Retryable: true,

		Temporary: true,

		HTTPStatus: 503,

		Timestamp: time.Now(),

		Metadata: map[string]interface{}{

			"active": atomic.LoadInt64(&b.active),

			"queued": atomic.LoadInt64(&b.queued),

			"rejected": atomic.LoadInt64(&b.rejected),

			"completed": atomic.LoadInt64(&b.completed),
		},
	}

}

// GetMetrics returns current bulkhead metrics.

func (b *Bulkhead) GetMetrics() map[string]interface{} {

	return map[string]interface{}{

		"name": b.config.Name,

		"active": atomic.LoadInt64(&b.active),

		"queued": atomic.LoadInt64(&b.queued),

		"rejected": atomic.LoadInt64(&b.rejected),

		"completed": atomic.LoadInt64(&b.completed),
	}

}

// Close shuts down the bulkhead.

func (b *Bulkhead) Close() {

	close(b.queue)

}

// TimeoutConfig configures timeout behavior.

type TimeoutConfig struct {
	DefaultTimeout time.Duration `json:"default_timeout"`

	MaxTimeout time.Duration `json:"max_timeout"`

	MinTimeout time.Duration `json:"min_timeout"`
}

// DefaultTimeoutConfig returns sensible timeout defaults.

func DefaultTimeoutConfig() *TimeoutConfig {

	return &TimeoutConfig{

		DefaultTimeout: time.Second * 30,

		MaxTimeout: time.Minute * 5,

		MinTimeout: time.Millisecond * 100,
	}

}

// TimeoutExecutor executes operations with timeout protection.

type TimeoutExecutor struct {
	config *TimeoutConfig
}

// NewTimeoutExecutor creates a new timeout executor.

func NewTimeoutExecutor(config *TimeoutConfig) *TimeoutExecutor {

	if config == nil {

		config = DefaultTimeoutConfig()

	}

	return &TimeoutExecutor{config: config}

}

// ExecuteWithTimeout executes an operation with timeout protection.

func (te *TimeoutExecutor) ExecuteWithTimeout(ctx context.Context, timeout time.Duration, operation func(context.Context) error) error {

	if timeout <= 0 {

		timeout = te.config.DefaultTimeout

	}

	if timeout > te.config.MaxTimeout {

		timeout = te.config.MaxTimeout

	}

	if timeout < te.config.MinTimeout {

		timeout = te.config.MinTimeout

	}

	ctx, cancel := context.WithTimeout(ctx, timeout)

	defer cancel()

	errChan := make(chan error, 1)

	go func() {

		errChan <- operation(ctx)

	}()

	select {

	case err := <-errChan:

		return err

	case <-ctx.Done():

		if ctx.Err() == context.DeadlineExceeded {

			return &ServiceError{

				Type: ErrorTypeTimeout,

				Code: "operation_timeout",

				Message: fmt.Sprintf("Operation timed out after %v", timeout),

				Service: "timeout-executor",

				Operation: "execute",

				Category: CategorySystem,

				Severity: SeverityMedium,

				Impact: ImpactModerate,

				Retryable: true,

				Temporary: true,

				HTTPStatus: 504,

				Timestamp: time.Now(),

				Metadata: map[string]interface{}{

					"timeout": timeout.String(),
				},
			}

		}

		return ctx.Err()

	}

}
