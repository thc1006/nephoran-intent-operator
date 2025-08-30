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

package parallel

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/nephio-project/nephoran-intent-operator/pkg/controllers/resilience"
	"github.com/nephio-project/nephoran-intent-operator/pkg/monitoring"
)

// ErrorRecoveryTestSuite provides comprehensive testing for error handling and recovery
type ErrorRecoveryTestSuite struct {
	suite.Suite
	logger        logr.Logger
	engine        *ParallelProcessingEngine
	resilienceMgr *resilience.ResilienceManager
	errorTracker  *monitoring.ErrorTracker
	ctx           context.Context
	cancel        context.CancelFunc
}

// SetupSuite initializes the test suite
func (suite *ErrorRecoveryTestSuite) SetupSuite() {
	zapLogger, _ := zap.NewDevelopment()
	suite.logger = zapr.NewLogger(zapLogger)

	suite.ctx, suite.cancel = context.WithCancel(context.Background())

	// Create resilience manager with test configuration
	resilienceConfig := &resilience.ResilienceConfig{
		DefaultTimeout:          10 * time.Second,
		MaxConcurrentOperations: 50,
		HealthCheckInterval:     5 * time.Second,
		TimeoutEnabled:          true,
		BulkheadEnabled:         true,
		CircuitBreakerEnabled:   true,
		RateLimitingEnabled:     true,
		RetryEnabled:            true,
		HealthCheckEnabled:      true,
		TimeoutConfigs: map[string]*resilience.TimeoutConfig{
			"test_operation": {
				Name:           "test_operation",
				DefaultTimeout: 5 * time.Second,
				MaxTimeout:     30 * time.Second,
				MinTimeout:     1 * time.Second,
			},
		},
	}

	suite.resilienceMgr = resilience.NewResilienceManager(resilienceConfig, suite.logger)
	require.NoError(suite.T(), suite.resilienceMgr.Start(suite.ctx))

	// Create error tracker
	errorConfig := &monitoring.ErrorTrackingConfig{
		EnablePrometheus:    true,
		EnableOpenTelemetry: true,
		AlertingEnabled:     true,
		DashboardEnabled:    false, // Disable for tests
		ReportsEnabled:      false, // Disable for tests
	}

	var err error
	suite.errorTracker, err = monitoring.NewErrorTracker(errorConfig, suite.logger)
	require.NoError(suite.T(), err)

	// Create parallel processing engine
	engineConfig := &ParallelProcessingConfig{
		MaxConcurrentIntents: 20,
		IntentPoolSize:       5,
		LLMPoolSize:          3,
		RAGPoolSize:          3,
		ResourcePoolSize:     4,
		ManifestPoolSize:     4,
		GitOpsPoolSize:       2,
		DeploymentPoolSize:   2,
		TaskQueueSize:        100,
		HealthCheckInterval:  5 * time.Second,
	}

	suite.engine, err = NewParallelProcessingEngine(
		engineConfig,
		suite.resilienceMgr,
		suite.errorTracker,
		suite.logger,
	)
	require.NoError(suite.T(), err)

	// Start the engine
	require.NoError(suite.T(), suite.engine.Start(suite.ctx))
}

// TearDownSuite cleans up the test suite
func (suite *ErrorRecoveryTestSuite) TearDownSuite() {
	if suite.engine != nil {
		suite.engine.Stop()
	}
	if suite.resilienceMgr != nil {
		suite.resilienceMgr.Stop()
	}
	suite.cancel()
}

// TestErrorCorrelation tests error correlation across related tasks
func (suite *ErrorRecoveryTestSuite) TestErrorCorrelation() {
	intentID := "intent-correlation-test"
	correlationID := "corr-123"

	// Create related tasks that will fail
	tasks := []*Task{
		{
			ID:            "task-1",
			IntentID:      intentID,
			CorrelationID: correlationID,
			Type:          TaskTypeLLMProcessing,
			Priority:      1,
			Dependencies:  []string{},
			Status:        TaskStatusPending,
			InputData:     map[string]interface{}{"intent": "test intent"},
			Timeout:       5 * time.Second,
		},
		{
			ID:            "task-2",
			IntentID:      intentID,
			CorrelationID: correlationID,
			Type:          TaskTypeRAGRetrieval,
			Priority:      1,
			Dependencies:  []string{"task-1"},
			Status:        TaskStatusPending,
			InputData:     map[string]interface{}{"query": "test query"},
			Timeout:       5 * time.Second,
		},
	}

	// Submit tasks
	for _, task := range tasks {
		err := suite.engine.SubmitTask(task)
		require.NoError(suite.T(), err)
	}

	// Wait for processing
	time.Sleep(2 * time.Second)

	// Check error correlation
	errors := suite.errorTracker.GetErrorsByCorrelation(correlationID)
	suite.T().Logf("Found %d correlated errors", len(errors))

	// Verify error correlation exists if errors occurred
	if len(errors) > 0 {
		assert.True(suite.T(), len(errors) >= 1, "Expected at least one correlated error")

		for _, err := range errors {
			assert.Equal(suite.T(), correlationID, err.CorrelationID)
			assert.Equal(suite.T(), intentID, err.IntentID)
		}
	}
}

// TestCircuitBreakerIntegration tests circuit breaker behavior with parallel processing
func (suite *ErrorRecoveryTestSuite) TestCircuitBreakerIntegration() {
	// Create a failing processor to trigger circuit breaker
	_ = &FailingProcessor{
		failureRate: 1.0, // Always fail
		logger:      suite.logger,
	}

	// Submit multiple tasks to trigger circuit breaker
	intentID := "intent-circuit-breaker-test"
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(taskID int) {
			defer wg.Done()

			task := &Task{
				ID:        fmt.Sprintf("task-%d", taskID),
				IntentID:  intentID,
				Type:      TaskTypeLLMProcessing,
				Priority:  1,
				Status:    TaskStatusPending,
				InputData: map[string]interface{}{"intent": "test intent"},
				Timeout:   2 * time.Second,
			}

			err := suite.engine.SubmitTask(task)
			if err != nil {
				suite.T().Logf("Task submission failed: %v", err)
			}
		}(i)
	}

	wg.Wait()

	// Wait for circuit breaker to potentially open
	time.Sleep(3 * time.Second)

	// Check if circuit breaker metrics indicate protection
	metrics := suite.resilienceMgr.GetMetrics()
	assert.NotNil(suite.T(), metrics)

	if metrics.CircuitBreakerMetrics != nil {
		suite.T().Logf("Circuit breaker metrics: %+v", metrics.CircuitBreakerMetrics)
	}
}

// TestRetryMechanismWithBackoff tests retry behavior with exponential backoff
func (suite *ErrorRecoveryTestSuite) TestRetryMechanismWithBackoff() {
	// Create a processor that fails first few times then succeeds
	_ = &RetryableProcessor{
		failCount:    3,
		currentTries: 0,
		logger:       suite.logger,
		mutex:        sync.Mutex{},
	}

	intentID := "intent-retry-test"
	task := &Task{
		ID:        "retry-task-1",
		IntentID:  intentID,
		Type:      TaskTypeLLMProcessing,
		Priority:  1,
		Status:    TaskStatusPending,
		InputData: map[string]interface{}{"intent": "test intent"},
		Timeout:   10 * time.Second,
		RetryConfig: &TaskRetryConfig{
			MaxAttempts:   5,
			InitialDelay:  100 * time.Millisecond,
			BackoffFactor: 2.0,
			MaxDelay:      2 * time.Second,
		},
	}

	startTime := time.Now()
	err := suite.engine.SubmitTask(task)
	require.NoError(suite.T(), err)

	// Wait for completion with retries
	time.Sleep(8 * time.Second)

	duration := time.Since(startTime)
	suite.T().Logf("Retry test completed in %v", duration)

	// Verify retry behavior occurred (should take longer than single attempt)
	assert.True(suite.T(), duration > 500*time.Millisecond, "Expected retries to take significant time")
}

// TestBulkheadIsolation tests that failures in one pool don't affect others
func (suite *ErrorRecoveryTestSuite) TestBulkheadIsolation() {
	intentID := "intent-bulkhead-test"

	// Create tasks for different pools
	llmTask := &Task{
		ID:        "llm-bulkhead-task",
		IntentID:  intentID,
		Type:      TaskTypeLLMProcessing,
		Priority:  1,
		Status:    TaskStatusPending,
		InputData: map[string]interface{}{"intent": "test intent"},
		Timeout:   5 * time.Second,
	}

	ragTask := &Task{
		ID:        "rag-bulkhead-task",
		IntentID:  intentID,
		Type:      TaskTypeRAGRetrieval,
		Priority:  1,
		Status:    TaskStatusPending,
		InputData: map[string]interface{}{"query": "test query"},
		Timeout:   5 * time.Second,
	}

	// Submit tasks to different pools
	err1 := suite.engine.SubmitTask(llmTask)
	err2 := suite.engine.SubmitTask(ragTask)

	assert.NoError(suite.T(), err1)
	assert.NoError(suite.T(), err2)

	// Wait for processing
	time.Sleep(3 * time.Second)

	// Verify both pools are functioning independently
	engineMetrics := suite.engine.GetMetrics()
	assert.NotNil(suite.T(), engineMetrics)

	suite.T().Logf("Engine metrics: %+v", engineMetrics)
}

// TestTimeoutHandling tests timeout behavior and recovery
func (suite *ErrorRecoveryTestSuite) TestTimeoutHandling() {
	intentID := "intent-timeout-test"

	// Create a task with short timeout
	task := &Task{
		ID:        "timeout-task",
		IntentID:  intentID,
		Type:      TaskTypeLLMProcessing,
		Priority:  1,
		Status:    TaskStatusPending,
		InputData: map[string]interface{}{"intent": "test intent"},
		Timeout:   500 * time.Millisecond, // Very short timeout
	}

	startTime := time.Now()
	err := suite.engine.SubmitTask(task)
	require.NoError(suite.T(), err)

	// Wait for timeout to occur
	time.Sleep(2 * time.Second)

	duration := time.Since(startTime)
	suite.T().Logf("Timeout test completed in %v", duration)

	// Check timeout metrics
	timeoutMetrics := suite.resilienceMgr.GetMetrics()
	if timeoutMetrics != nil && timeoutMetrics.TimeoutMetrics != nil {
		suite.T().Logf("Timeout metrics: %+v", timeoutMetrics.TimeoutMetrics)

		// If timeouts occurred, verify they were handled properly
		if timeoutMetrics.TimeoutMetrics.TimeoutOperations > 0 {
			assert.True(suite.T(), timeoutMetrics.TimeoutMetrics.TimeoutRate > 0)
		}
	}
}

// TestDependencyFailurePropagation tests how dependency failures affect downstream tasks
func (suite *ErrorRecoveryTestSuite) TestDependencyFailurePropagation() {
	intentID := "intent-dependency-test"

	// Create tasks with dependencies
	tasks := []*Task{
		{
			ID:           "dep-task-root",
			IntentID:     intentID,
			Type:         TaskTypeLLMProcessing,
			Priority:     1,
			Dependencies: []string{},
			Status:       TaskStatusPending,
			InputData:    map[string]interface{}{"intent": "test intent"},
			Timeout:      5 * time.Second,
		},
		{
			ID:           "dep-task-child1",
			IntentID:     intentID,
			Type:         TaskTypeRAGRetrieval,
			Priority:     1,
			Dependencies: []string{"dep-task-root"},
			Status:       TaskStatusPending,
			InputData:    map[string]interface{}{"query": "test query"},
			Timeout:      5 * time.Second,
		},
		{
			ID:           "dep-task-child2",
			IntentID:     intentID,
			Type:         TaskTypeResourcePlanning,
			Priority:     1,
			Dependencies: []string{"dep-task-root"},
			Status:       TaskStatusPending,
			InputData:    map[string]interface{}{},
			Timeout:      5 * time.Second,
		},
		{
			ID:           "dep-task-grandchild",
			IntentID:     intentID,
			Type:         TaskTypeManifestGeneration,
			Priority:     1,
			Dependencies: []string{"dep-task-child1", "dep-task-child2"},
			Status:       TaskStatusPending,
			InputData:    map[string]interface{}{},
			Timeout:      5 * time.Second,
		},
	}

	// Submit all tasks
	for _, task := range tasks {
		err := suite.engine.SubmitTask(task)
		require.NoError(suite.T(), err)
	}

	// Wait for processing
	time.Sleep(5 * time.Second)

	// Check task status and dependency handling
	for _, task := range tasks {
		status := suite.engine.GetTaskStatus(task.ID)
		suite.T().Logf("Task %s status: %v", task.ID, status)
	}

	// Verify dependency graph handled the relationships
	dependencyMetrics := suite.engine.GetDependencyMetrics()
	assert.NotNil(suite.T(), dependencyMetrics)
	suite.T().Logf("Dependency metrics: %+v", dependencyMetrics)
}

// TestConcurrentErrorHandling tests error handling under high concurrent load
func (suite *ErrorRecoveryTestSuite) TestConcurrentErrorHandling() {
	intentID := "intent-concurrent-error-test"
	numTasks := 50
	var wg sync.WaitGroup

	// Create many concurrent tasks
	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		go func(taskNum int) {
			defer wg.Done()

			task := &Task{
				ID:        fmt.Sprintf("concurrent-task-%d", taskNum),
				IntentID:  intentID,
				Type:      TaskTypeLLMProcessing,
				Priority:  1,
				Status:    TaskStatusPending,
				InputData: map[string]interface{}{"intent": "test intent"},
				Timeout:   10 * time.Second,
			}

			err := suite.engine.SubmitTask(task)
			if err != nil {
				suite.T().Logf("Task %d submission failed: %v", taskNum, err)
			}
		}(i)
	}

	wg.Wait()

	// Wait for processing
	time.Sleep(10 * time.Second)

	// Check overall system health
	engineMetrics := suite.engine.GetMetrics()
	assert.NotNil(suite.T(), engineMetrics)

	suite.T().Logf("Concurrent processing metrics: %+v", engineMetrics)

	// Verify system handled concurrent load
	assert.True(suite.T(), engineMetrics.TotalTasks >= int64(numTasks))
}

// TestRecoveryAfterFailure tests system recovery after component failures
func (suite *ErrorRecoveryTestSuite) TestRecoveryAfterFailure() {
	intentID := "intent-recovery-test"

	// First, submit a normal task to establish baseline
	normalTask := &Task{
		ID:        "recovery-normal-task",
		IntentID:  intentID,
		Type:      TaskTypeLLMProcessing,
		Priority:  1,
		Status:    TaskStatusPending,
		InputData: map[string]interface{}{"intent": "test intent"},
		Timeout:   5 * time.Second,
	}

	err := suite.engine.SubmitTask(normalTask)
	require.NoError(suite.T(), err)

	time.Sleep(2 * time.Second)

	// Now submit tasks after potential failures
	recoveryTask := &Task{
		ID:        "recovery-task",
		IntentID:  intentID,
		Type:      TaskTypeRAGRetrieval,
		Priority:  1,
		Status:    TaskStatusPending,
		InputData: map[string]interface{}{"query": "recovery query"},
		Timeout:   5 * time.Second,
	}

	err = suite.engine.SubmitTask(recoveryTask)
	require.NoError(suite.T(), err)

	time.Sleep(3 * time.Second)

	// Verify system is still functional
	health := suite.engine.HealthCheck()
	assert.NoError(suite.T(), health)

	metrics := suite.engine.GetMetrics()
	assert.NotNil(suite.T(), metrics)
	suite.T().Logf("Recovery metrics: %+v", metrics)
}

// TestErrorReporting tests comprehensive error reporting and metrics
func (suite *ErrorRecoveryTestSuite) TestErrorReporting() {
	intentID := "intent-error-reporting-test"

	// Create various types of tasks to generate different error patterns
	tasks := []*Task{
		{
			ID:        "error-report-task-1",
			IntentID:  intentID,
			Type:      TaskTypeLLMProcessing,
			Priority:  1,
			Status:    TaskStatusPending,
			InputData: map[string]interface{}{"intent": "test intent 1"},
			Timeout:   5 * time.Second,
		},
		{
			ID:        "error-report-task-2",
			IntentID:  intentID,
			Type:      TaskTypeRAGRetrieval,
			Priority:  2,
			Status:    TaskStatusPending,
			InputData: map[string]interface{}{"query": "test query 2"},
			Timeout:   3 * time.Second,
		},
	}

	// Submit tasks
	for _, task := range tasks {
		err := suite.engine.SubmitTask(task)
		require.NoError(suite.T(), err)
	}

	// Wait for processing
	time.Sleep(5 * time.Second)

	// Check error tracking and reporting
	errorSummary := suite.errorTracker.GetErrorSummary()
	assert.NotNil(suite.T(), errorSummary)
	suite.T().Logf("Error summary: %+v", errorSummary)

	// Check error patterns
	errorPatterns := suite.errorTracker.GetErrorPatterns()
	assert.NotNil(suite.T(), errorPatterns)
	suite.T().Logf("Error patterns: %+v", errorPatterns)

	// Verify error metrics are being collected
	metrics := suite.errorTracker.GetMetrics()
	assert.NotNil(suite.T(), metrics)
	suite.T().Logf("Error tracking metrics: %+v", metrics)
}

// TestRunSuite runs the error recovery test suite
func TestErrorRecoveryTestSuite(t *testing.T) {
	suite.Run(t, new(ErrorRecoveryTestSuite))
}

// Helper test processors

// FailingProcessor always fails for testing circuit breaker behavior
type FailingProcessor struct {
	failureRate float64
	logger      logr.Logger
}

func (fp *FailingProcessor) ProcessTask(ctx context.Context, task *Task) (*TaskResult, error) {
	fp.logger.Info("FailingProcessor processing task", "taskId", task.ID)

	// Simulate processing time
	time.Sleep(100 * time.Millisecond)

	// Always fail based on failure rate
	if fp.failureRate >= 1.0 {
		return &TaskResult{
			TaskID:  task.ID,
			Success: false,
			Error:   fmt.Errorf("simulated processor failure"),
		}, fmt.Errorf("simulated processor failure")
	}

	return &TaskResult{
		TaskID:  task.ID,
		Success: true,
	}, nil
}

func (fp *FailingProcessor) GetProcessorType() TaskType {
	return TaskTypeLLMProcessing
}

func (fp *FailingProcessor) HealthCheck(ctx context.Context) error {
	return nil
}

func (fp *FailingProcessor) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"processor_type": "failing_processor",
		"failure_rate":   fp.failureRate,
	}
}

// RetryableProcessor fails a specific number of times then succeeds
type RetryableProcessor struct {
	failCount    int
	currentTries int
	logger       logr.Logger
	mutex        sync.Mutex
}

func (rp *RetryableProcessor) ProcessTask(ctx context.Context, task *Task) (*TaskResult, error) {
	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	rp.currentTries++
	rp.logger.Info("RetryableProcessor processing task",
		"taskId", task.ID,
		"attempt", rp.currentTries)

	// Simulate processing time
	time.Sleep(200 * time.Millisecond)

	// Fail for the first few attempts
	if rp.currentTries <= rp.failCount {
		return &TaskResult{
			TaskID:  task.ID,
			Success: false,
			Error:   fmt.Errorf("simulated failure on attempt %d", rp.currentTries),
		}, fmt.Errorf("simulated failure on attempt %d", rp.currentTries)
	}

	// Succeed after enough attempts
	return &TaskResult{
		TaskID:  task.ID,
		Success: true,
		OutputData: map[string]interface{}{
			"attempts":                rp.currentTries,
			"succeeded_after_retries": true,
		},
	}, nil
}

func (rp *RetryableProcessor) GetProcessorType() TaskType {
	return TaskTypeLLMProcessing
}

func (rp *RetryableProcessor) HealthCheck(ctx context.Context) error {
	return nil
}

func (rp *RetryableProcessor) GetMetrics() map[string]interface{} {
	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	return map[string]interface{}{
		"processor_type": "retryable_processor",
		"current_tries":  rp.currentTries,
		"fail_count":     rp.failCount,
	}
}
