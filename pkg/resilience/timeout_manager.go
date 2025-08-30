package resilience

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/nephio-project/nephoran-intent-operator/pkg/monitoring"
)

// TimeoutConfig holds timeout configuration for different operation types.

type TimeoutConfig struct {

	// LLM processing timeout.

	LLMTimeout time.Duration `json:"llm_timeout" yaml:"llm_timeout"`

	// Git operations timeout.

	GitTimeout time.Duration `json:"git_timeout" yaml:"git_timeout"`

	// Kubernetes API timeout.

	KubernetesTimeout time.Duration `json:"kubernetes_timeout" yaml:"kubernetes_timeout"`

	// Package generation timeout.

	PackageGenerationTimeout time.Duration `json:"package_generation_timeout" yaml:"package_generation_timeout"`

	// RAG system timeout.

	RAGTimeout time.Duration `json:"rag_timeout" yaml:"rag_timeout"`

	// Overall reconciliation timeout.

	ReconciliationTimeout time.Duration `json:"reconciliation_timeout" yaml:"reconciliation_timeout"`

	// Default timeout for unspecified operations.

	DefaultTimeout time.Duration `json:"default_timeout" yaml:"default_timeout"`
}

// DefaultTimeoutConfig returns the default timeout configuration.

func DefaultTimeoutConfig() *TimeoutConfig {

	return &TimeoutConfig{

		LLMTimeout: 30 * time.Second,

		GitTimeout: 60 * time.Second,

		KubernetesTimeout: 30 * time.Second,

		PackageGenerationTimeout: 120 * time.Second,

		RAGTimeout: 15 * time.Second,

		ReconciliationTimeout: 300 * time.Second,

		DefaultTimeout: 30 * time.Second,
	}

}

// OperationType represents different types of operations that can timeout.

type OperationType string

const (

	// OperationTypeLLM holds operationtypellm value.

	OperationTypeLLM OperationType = "llm"

	// OperationTypeGit holds operationtypegit value.

	OperationTypeGit OperationType = "git"

	// OperationTypeKubernetes holds operationtypekubernetes value.

	OperationTypeKubernetes OperationType = "kubernetes"

	// OperationTypePackageGeneration holds operationtypepackagegeneration value.

	OperationTypePackageGeneration OperationType = "package_generation"

	// OperationTypeRAG holds operationtyperag value.

	OperationTypeRAG OperationType = "rag"

	// OperationTypeReconciliation holds operationtypereconciliation value.

	OperationTypeReconciliation OperationType = "reconciliation"

	// OperationTypeDefault holds operationtypedefault value.

	OperationTypeDefault OperationType = "default"
)

// TimeoutManager manages timeouts for various operations.

type TimeoutManager struct {
	config *TimeoutConfig

	metricsCollector *monitoring.MetricsCollector

	activeContexts map[string]context.CancelFunc

	mutex sync.RWMutex
}

// NewTimeoutManager creates a new timeout manager.

func NewTimeoutManager(config *TimeoutConfig, metricsCollector *monitoring.MetricsCollector) *TimeoutManager {

	if config == nil {

		config = DefaultTimeoutConfig()

	}

	return &TimeoutManager{

		config: config,

		metricsCollector: metricsCollector,

		activeContexts: make(map[string]context.CancelFunc),
	}

}

// GetTimeoutForOperation returns the configured timeout for a specific operation type.

func (tm *TimeoutManager) GetTimeoutForOperation(operationType OperationType) time.Duration {

	switch operationType {

	case OperationTypeLLM:

		return tm.config.LLMTimeout

	case OperationTypeGit:

		return tm.config.GitTimeout

	case OperationTypeKubernetes:

		return tm.config.KubernetesTimeout

	case OperationTypePackageGeneration:

		return tm.config.PackageGenerationTimeout

	case OperationTypeRAG:

		return tm.config.RAGTimeout

	case OperationTypeReconciliation:

		return tm.config.ReconciliationTimeout

	default:

		return tm.config.DefaultTimeout

	}

}

// CreateTimeoutContext creates a context with timeout for the specified operation type.

func (tm *TimeoutManager) CreateTimeoutContext(parent context.Context, operationType OperationType) (context.Context, context.CancelFunc) {

	timeout := tm.GetTimeoutForOperation(operationType)

	return context.WithTimeout(parent, timeout)

}

// CreateTimeoutContextWithID creates a context with timeout and tracks it with an ID.

func (tm *TimeoutManager) CreateTimeoutContextWithID(parent context.Context, operationType OperationType, contextID string) (context.Context, context.CancelFunc) {

	timeout := tm.GetTimeoutForOperation(operationType)

	ctx, cancel := context.WithTimeout(parent, timeout)

	// Track the context.

	tm.mutex.Lock()

	tm.activeContexts[contextID] = cancel

	tm.mutex.Unlock()

	// Return a cancel function that also removes tracking.

	wrappedCancel := func() {

		tm.mutex.Lock()

		delete(tm.activeContexts, contextID)

		tm.mutex.Unlock()

		cancel()

	}

	return ctx, wrappedCancel

}

// ExecuteWithTimeout executes a function with timeout for the specified operation type.

func (tm *TimeoutManager) ExecuteWithTimeout(

	ctx context.Context,

	operationType OperationType,

	operation func(context.Context) (interface{}, error),

) (interface{}, error) {

	startTime := time.Now()

	// Create timeout context.

	timeoutCtx, cancel := tm.CreateTimeoutContext(ctx, operationType)

	defer cancel()

	// Channel to capture result.

	resultChan := make(chan interface{}, 1)

	errorChan := make(chan error, 1)

	// Execute operation in goroutine.

	go func() {

		defer func() {

			if r := recover(); r != nil {

				errorChan <- fmt.Errorf("operation panicked: %v", r)

			}

		}()

		result, err := operation(timeoutCtx)

		if err != nil {

			errorChan <- err

		} else {

			resultChan <- result

		}

	}()

	// Wait for result or timeout.

	select {

	case result := <-resultChan:

		duration := time.Since(startTime)

		tm.recordSuccess(operationType, duration)

		return result, nil

	case err := <-errorChan:

		duration := time.Since(startTime)

		tm.recordError(operationType, duration, err)

		return nil, err

	case <-timeoutCtx.Done():

		duration := time.Since(startTime)

		timeoutErr := fmt.Errorf("operation timed out after %v: %w", tm.GetTimeoutForOperation(operationType), timeoutCtx.Err())

		tm.recordTimeout(operationType, duration)

		return nil, timeoutErr

	}

}

// ExecuteWithCustomTimeout executes a function with a custom timeout.

func (tm *TimeoutManager) ExecuteWithCustomTimeout(

	ctx context.Context,

	timeout time.Duration,

	operation func(context.Context) (interface{}, error),

) (interface{}, error) {

	startTime := time.Now()

	// Create timeout context.

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)

	defer cancel()

	// Channel to capture result.

	resultChan := make(chan interface{}, 1)

	errorChan := make(chan error, 1)

	// Execute operation in goroutine.

	go func() {

		defer func() {

			if r := recover(); r != nil {

				errorChan <- fmt.Errorf("operation panicked: %v", r)

			}

		}()

		result, err := operation(timeoutCtx)

		if err != nil {

			errorChan <- err

		} else {

			resultChan <- result

		}

	}()

	// Wait for result or timeout.

	select {

	case result := <-resultChan:

		duration := time.Since(startTime)

		tm.recordSuccess(OperationTypeDefault, duration)

		return result, nil

	case err := <-errorChan:

		duration := time.Since(startTime)

		tm.recordError(OperationTypeDefault, duration, err)

		return nil, err

	case <-timeoutCtx.Done():

		duration := time.Since(startTime)

		timeoutErr := fmt.Errorf("operation timed out after %v: %w", timeout, timeoutCtx.Err())

		tm.recordTimeout(OperationTypeDefault, duration)

		return nil, timeoutErr

	}

}

// ExecuteWithGracefulDegradation executes an operation with graceful degradation on timeout.

func (tm *TimeoutManager) ExecuteWithGracefulDegradation(

	ctx context.Context,

	operationType OperationType,

	operation func(context.Context) (interface{}, error),

	fallback func(context.Context, error) (interface{}, error),

) (interface{}, error) {

	result, err := tm.ExecuteWithTimeout(ctx, operationType, operation)

	// If timeout occurred and fallback is provided, use fallback.

	if err != nil && isTimeoutError(err) && fallback != nil {

		return fallback(ctx, err)

	}

	return result, err

}

// CancelContext cancels a tracked context by ID.

func (tm *TimeoutManager) CancelContext(contextID string) bool {

	tm.mutex.Lock()

	defer tm.mutex.Unlock()

	if cancel, exists := tm.activeContexts[contextID]; exists {

		cancel()

		delete(tm.activeContexts, contextID)

		return true

	}

	return false

}

// GetActiveContextCount returns the number of active tracked contexts.

func (tm *TimeoutManager) GetActiveContextCount() int {

	tm.mutex.RLock()

	defer tm.mutex.RUnlock()

	return len(tm.activeContexts)

}

// CancelAllContexts cancels all tracked contexts.

func (tm *TimeoutManager) CancelAllContexts() {

	tm.mutex.Lock()

	defer tm.mutex.Unlock()

	for contextID, cancel := range tm.activeContexts {

		cancel()

		delete(tm.activeContexts, contextID)

	}

}

// UpdateConfig updates the timeout configuration.

func (tm *TimeoutManager) UpdateConfig(config *TimeoutConfig) {

	tm.mutex.Lock()

	defer tm.mutex.Unlock()

	tm.config = config

}

// GetConfig returns the current timeout configuration.

func (tm *TimeoutManager) GetConfig() *TimeoutConfig {

	tm.mutex.RLock()

	defer tm.mutex.RUnlock()

	// Return a copy to prevent external modification.

	return &TimeoutConfig{

		LLMTimeout: tm.config.LLMTimeout,

		GitTimeout: tm.config.GitTimeout,

		KubernetesTimeout: tm.config.KubernetesTimeout,

		PackageGenerationTimeout: tm.config.PackageGenerationTimeout,

		RAGTimeout: tm.config.RAGTimeout,

		ReconciliationTimeout: tm.config.ReconciliationTimeout,

		DefaultTimeout: tm.config.DefaultTimeout,
	}

}

// recordSuccess records a successful operation.

func (tm *TimeoutManager) recordSuccess(operationType OperationType, duration time.Duration) {

	if tm.metricsCollector == nil {

		return

	}

	_ = operationType // avoid unused variable

	// Record operation duration.

	if histogram := tm.metricsCollector.GetHistogram("operation_duration_seconds"); histogram != nil {

		histogram.Observe(duration.Seconds())

	}

	// Increment success counter.

	if counter := tm.metricsCollector.GetCounter("operations_total"); counter != nil {

		counter.Inc()

	}

}

// recordError records a failed operation.

func (tm *TimeoutManager) recordError(operationType OperationType, duration time.Duration, err error) {

	if tm.metricsCollector == nil {

		return

	}

	_ = operationType // avoid unused variable

	// Record operation duration.

	if histogram := tm.metricsCollector.GetHistogram("operation_duration_seconds"); histogram != nil {

		histogram.Observe(duration.Seconds())

	}

	// Increment error counter.

	if counter := tm.metricsCollector.GetCounter("operations_total"); counter != nil {

		counter.Inc()

	}

}

// recordTimeout records a timed out operation.

func (tm *TimeoutManager) recordTimeout(operationType OperationType, duration time.Duration) {

	if tm.metricsCollector == nil {

		return

	}

	_ = operationType // avoid unused variable

	// Record operation duration.

	if histogram := tm.metricsCollector.GetHistogram("operation_duration_seconds"); histogram != nil {

		histogram.Observe(duration.Seconds())

	}

	// Increment timeout counter.

	if counter := tm.metricsCollector.GetCounter("operations_total"); counter != nil {

		counter.Inc()

	}

}

// isTimeoutError checks if an error is a timeout error.

func isTimeoutError(err error) bool {

	if err == nil {

		return false

	}

	// Check for context timeout.

	if errors.Is(err, context.DeadlineExceeded) {

		return true

	}

	// Check if error message contains timeout information.

	errStr := err.Error()

	return contains(errStr, "timeout") || contains(errStr, "deadline exceeded") || contains(errStr, "context deadline exceeded")

}

// contains is a helper function to check if a string contains a substring.

func contains(str, substr string) bool {

	return len(str) >= len(substr) && (str == substr || (len(str) > len(substr) &&

		(indexOf(str, substr) >= 0)))

}

// indexOf is a helper function to find the index of a substring.

func indexOf(str, substr string) int {

	for i := 0; i <= len(str)-len(substr); i++ {

		if str[i:i+len(substr)] == substr {

			return i

		}

	}

	return -1

}
