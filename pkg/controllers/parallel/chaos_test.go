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
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nephio-project/nephoran-intent-operator/api/v1"
	"github.com/nephio-project/nephoran-intent-operator/pkg/controllers/resilience"
	"github.com/nephio-project/nephoran-intent-operator/pkg/monitoring"
)

// ChaosTestSuite performs chaos engineering tests to validate system resilience
type ChaosTestSuite struct {
	suite.Suite
	logger        logr.Logger
	engine        *ParallelProcessingEngine
	resilienceMgr *resilience.ResilienceManager
	errorTracker  *monitoring.ErrorTracker
	ctx           context.Context
	cancel        context.CancelFunc
	random        *rand.Rand
}

func (suite *ChaosTestSuite) SetupSuite() {
	zapLogger, _ := zap.NewDevelopment()
	suite.logger = zapr.NewLogger(zapLogger)
	suite.random = rand.New(rand.NewSource(time.Now().UnixNano()))

	suite.ctx, suite.cancel = context.WithCancel(context.Background())

	// Setup resilience manager with aggressive settings for chaos testing
	resilienceConfig := &resilience.ResilienceConfig{
		DefaultTimeout:          20 * time.Second,
		MaxConcurrentOperations: 200,
		HealthCheckInterval:     5 * time.Second,
		TimeoutEnabled:          true,
		BulkheadEnabled:         true,
		CircuitBreakerEnabled:   true,
		RateLimitingEnabled:     true,
		RetryEnabled:            true,
		HealthCheckEnabled:      true,
		TimeoutConfigs: map[string]*resilience.TimeoutConfig{
			"chaos_operation": {
				Name:            "chaos_operation",
				DefaultTimeout:  5 * time.Second,
				MaxTimeout:      30 * time.Second,
				MinTimeout:      500 * time.Millisecond,
				AdaptiveTimeout: true,
			},
		},
	}

	suite.resilienceMgr = resilience.NewResilienceManager(resilienceConfig, suite.logger)
	require.NoError(suite.T(), suite.resilienceMgr.Start(suite.ctx))

	// Setup error tracker for chaos monitoring
	errorConfig := &monitoring.ErrorTrackingConfig{
		EnablePrometheus:    true,
		EnableOpenTelemetry: true,
		AlertingEnabled:     false, // Disable alerting during chaos tests
		DashboardEnabled:    false,
		ReportsEnabled:      true,
	}

	var err error
	suite.errorTracker, err = monitoring.NewErrorTracker(errorConfig, suite.logger)
	require.NoError(suite.T(), err)

	// Setup processing engine with high capacity for chaos testing
	engineConfig := &ParallelProcessingConfig{
		MaxConcurrentIntents: 100,
		IntentPoolSize:       15,
		LLMPoolSize:          10,
		RAGPoolSize:          10,
		ResourcePoolSize:     15,
		ManifestPoolSize:     15,
		GitOpsPoolSize:       8,
		DeploymentPoolSize:   8,
		TaskQueueSize:        500,
		HealthCheckInterval:  3 * time.Second,
	}

	suite.engine, err = NewParallelProcessingEngine(
		engineConfig,
		suite.resilienceMgr,
		suite.errorTracker,
		suite.logger,
	)
	require.NoError(suite.T(), err)
	require.NoError(suite.T(), suite.engine.Start(suite.ctx))
}

func (suite *ChaosTestSuite) TearDownSuite() {
	if suite.engine != nil {
		suite.engine.Stop()
	}
	if suite.resilienceMgr != nil {
		suite.resilienceMgr.Stop()
	}
	suite.cancel()
}

// TestRandomFailureInjection injects random failures during processing
func (suite *ChaosTestSuite) TestRandomFailureInjection() {
	suite.T().Log("Starting random failure injection chaos test")

	numIntents := 20
	failureRate := 0.3 // 30% failure rate

	var wg sync.WaitGroup
	results := make([]bool, numIntents)
	startTime := time.Now()

	// Inject failures randomly during intent processing
	for i := 0; i < numIntents; i++ {
		wg.Add(1)
		go func(intentNum int) {
			defer wg.Done()

			intent := &v1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("chaos-intent-%d", intentNum),
					Namespace: "chaos-test",
				},
				Spec: v1.NetworkIntentSpec{
					IntentType: "chaos_test_deployment",
					Intent:     fmt.Sprintf("Chaos test intent %d with potential failures", intentNum),
					Priority:   v1.NetworkPriorityNormal,
				},
			}

			// Randomly introduce processing delays and failures
			if suite.random.Float64() < failureRate {
				// Inject failure by using very short timeout
				ctx, cancel := context.WithTimeout(suite.ctx, 100*time.Millisecond)
				defer cancel()
				_, _ = suite.engine.ProcessIntentWorkflow(ctx, intent)
				results[intentNum] = false
			} else {
				// Normal processing with potential for natural failures
				result, err := suite.engine.ProcessIntentWorkflow(suite.ctx, intent)
				results[intentNum] = (err == nil && result != nil && result.Success)
			}

			// Random processing delays to create chaos
			if suite.random.Float64() < 0.2 {
				time.Sleep(time.Duration(suite.random.Intn(1000)) * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	// Analyze chaos test results
	successCount := 0
	for _, success := range results {
		if success {
			successCount++
		}
	}

	suite.T().Logf("Chaos test completed in %v: %d/%d intents successful",
		duration, successCount, numIntents)

	// System should demonstrate resilience despite injected chaos
	// Even with 30% intentional failures, some processing should succeed
	assert.True(suite.T(), successCount > 0, "Some intents should succeed despite chaos")

	// Verify system health after chaos
	health := suite.engine.HealthCheck()
	assert.NoError(suite.T(), health, "System should remain healthy after chaos injection")

	// Check error tracking captured the chaos
	errorSummary := suite.errorTracker.GetErrorSummary()
	if errorSummary != nil {
		suite.T().Logf("Chaos error summary: %+v", errorSummary)
	}
}

// TestResourceExhaustion tests system behavior under resource exhaustion
func (suite *ChaosTestSuite) TestResourceExhaustion() {
	suite.T().Log("Starting resource exhaustion chaos test")

	// Create a massive burst of intents to exhaust resources
	burstSize := 150 // Exceed configured capacity
	var wg sync.WaitGroup

	startTime := time.Now()

	// Create resource exhaustion by submitting many intents simultaneously
	for i := 0; i < burstSize; i++ {
		wg.Add(1)
		go func(intentNum int) {
			defer wg.Done()

			intent := &v1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("burst-intent-%d", intentNum),
					Namespace: "chaos-test",
				},
				Spec: v1.NetworkIntentSpec{
					IntentType: "resource_intensive",
					Intent:     fmt.Sprintf("Resource intensive intent %d", intentNum),
					Priority:   v1.NetworkPriorityHigh,
				},
			}

			// Use context with timeout to prevent indefinite blocking
			ctx, cancel := context.WithTimeout(suite.ctx, 30*time.Second)
			defer cancel()

			_, _ = suite.engine.ProcessIntentWorkflow(ctx, intent)
		}(i)
	}

	// Monitor system during resource exhaustion
	monitorCtx, monitorCancel := context.WithCancel(suite.ctx)
	go suite.monitorSystemDuringChaos(monitorCtx)

	wg.Wait()
	monitorCancel()

	duration := time.Since(startTime)
	suite.T().Logf("Resource exhaustion test completed in %v", duration)

	// System should handle resource exhaustion gracefully
	// It may not process all requests, but it should not crash
	health := suite.engine.HealthCheck()
	assert.NoError(suite.T(), health, "System should survive resource exhaustion")

	// Check final metrics
	metrics := suite.engine.GetMetrics()
	suite.T().Logf("Post-exhaustion metrics: %+v", metrics)

	// Verify some level of throughput was maintained
	assert.True(suite.T(), metrics.TotalTasks > 0, "Some tasks should have been processed")
}

// TestNetworkPartitions simulates network partition scenarios
func (suite *ChaosTestSuite) TestNetworkPartitions() {
	suite.T().Log("Starting network partition chaos test")

	numIntents := 15
	var wg sync.WaitGroup
	partitionDuration := 5 * time.Second

	// Start processing intents
	for i := 0; i < numIntents; i++ {
		wg.Add(1)
		go func(intentNum int) {
			defer wg.Done()

			intent := &v1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("partition-intent-%d", intentNum),
					Namespace: "chaos-test",
				},
				Spec: v1.NetworkIntentSpec{
					IntentType: "network_partition_test",
					Intent:     fmt.Sprintf("Intent during network partition %d", intentNum),
					Priority:   v1.NetworkPriorityNormal,
				},
			}

			// Simulate network partition by introducing random delays and timeouts
			if suite.random.Float64() < 0.4 {
				// Simulate slow network
				time.Sleep(time.Duration(suite.random.Intn(2000)) * time.Millisecond)
			}

			ctx, cancel := context.WithTimeout(suite.ctx, 20*time.Second)
			defer cancel()

			_, _ = suite.engine.ProcessIntentWorkflow(ctx, intent)
		}(i)
	}

	// Simulate network partition by introducing system instability
	go func() {
		time.Sleep(2 * time.Second) // Let some processing start

		suite.T().Log("Simulating network partition...")

		// Create chaos by submitting conflicting high-priority tasks
		for j := 0; j < 10; j++ {
			go func(chaosNum int) {
				task := &Task{
					ID:        fmt.Sprintf("chaos-task-%d", chaosNum),
					IntentID:  "partition-chaos",
					Type:      TaskTypeLLMProcessing,
					Priority:  10, // Very high priority
					Status:    TaskStatusPending,
					InputData: map[string]interface{}{"intent": "chaos"},
					Timeout:   1 * time.Second, // Short timeout to create instability
				}

				_ = suite.engine.SubmitTask(task)
			}(j)
		}

		time.Sleep(partitionDuration)
		suite.T().Log("Network partition simulation ended")
	}()

	wg.Wait()

	// System should demonstrate resilience to network partitions
	health := suite.engine.HealthCheck()
	assert.NoError(suite.T(), health, "System should survive network partition simulation")

	// Check that error handling worked during partition
	errorSummary := suite.errorTracker.GetErrorSummary()
	if errorSummary != nil {
		suite.T().Logf("Network partition error summary: %+v", errorSummary)
	}
}

// TestCascadingFailures tests system behavior with cascading failures
func (suite *ChaosTestSuite) TestCascadingFailures() {
	suite.T().Log("Starting cascading failure chaos test")

	// Create intents with complex dependencies to trigger cascading failures
	baseIntent := &v1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cascading-base-intent",
			Namespace: "chaos-test",
		},
		Spec: v1.NetworkIntentSpec{
			IntentType: "cascading_failure_test",
			Intent:     "Base intent that will trigger cascading failures in dependent components",
			Priority:   v1.NetworkPriorityCritical,
		},
	}

	// Submit the base intent
	baseResult, baseErr := suite.engine.ProcessIntentWorkflow(suite.ctx, baseIntent)
	if baseErr != nil {
		suite.T().Logf("Base intent failed as expected: %v", baseErr)
	}

	// Now submit multiple dependent intents that may fail due to base failure
	numDependentIntents := 10
	var wg sync.WaitGroup

	for i := 0; i < numDependentIntents; i++ {
		wg.Add(1)
		go func(intentNum int) {
			defer wg.Done()

			dependentIntent := &v1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("cascading-dependent-%d", intentNum),
					Namespace: "chaos-test",
				},
				Spec: v1.NetworkIntentSpec{
					IntentType: "dependent_service",
					Intent:     fmt.Sprintf("Dependent intent %d that may fail due to base failure", intentNum),
					Priority:   v1.NetworkPriorityHigh,
				},
			}

			ctx, cancel := context.WithTimeout(suite.ctx, 15*time.Second)
			defer cancel()

			_, _ = suite.engine.ProcessIntentWorkflow(ctx, dependentIntent)
		}(i)
	}

	wg.Wait()

	// Verify system handled cascading failures gracefully
	health := suite.engine.HealthCheck()
	assert.NoError(suite.T(), health, "System should handle cascading failures gracefully")

	// Check error correlation for cascading failures
	if baseResult != nil && !baseResult.Success {
		cascadingErrors := suite.errorTracker.GetErrorsByPattern("cascading")
		suite.T().Logf("Detected %d cascading errors", len(cascadingErrors))
	}

	// System should demonstrate circuit breaker behavior
	metrics := suite.resilienceMgr.GetMetrics()
	if metrics.CircuitBreakerMetrics != nil {
		suite.T().Logf("Circuit breaker metrics after cascading failures: %+v", metrics.CircuitBreakerMetrics)
	}
}

// TestRecoveryAfterChaos tests system recovery after various chaos scenarios
func (suite *ChaosTestSuite) TestRecoveryAfterChaos() {
	suite.T().Log("Starting recovery after chaos test")

	// First, create chaos conditions
	chaosIntents := 20

	suite.T().Log("Creating chaos conditions...")
	var chaosWg sync.WaitGroup

	for i := 0; i < chaosIntents; i++ {
		chaosWg.Add(1)
		go func(intentNum int) {
			defer chaosWg.Done()

			intent := &v1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("chaos-condition-%d", intentNum),
					Namespace: "chaos-test",
				},
				Spec: v1.NetworkIntentSpec{
					IntentType: "chaos_inducer",
					Intent:     fmt.Sprintf("Chaos inducing intent %d", intentNum),
					Priority:   v1.NetworkPriorityNormal,
				},
			}

			// Create chaos with random timeouts and failures
			timeoutDuration := time.Duration(suite.random.Intn(1000)) * time.Millisecond
			ctx, cancel := context.WithTimeout(suite.ctx, timeoutDuration)
			defer cancel()

			_, _ = suite.engine.ProcessIntentWorkflow(ctx, intent)
		}(i)
	}

	chaosWg.Wait()
	suite.T().Log("Chaos conditions created, waiting for recovery...")

	// Allow time for system recovery
	recoveryTime := 10 * time.Second
	time.Sleep(recoveryTime)

	// Test recovery by processing normal intents
	suite.T().Log("Testing recovery with normal intents...")

	recoveryIntents := 5
	var recoveryWg sync.WaitGroup
	recoveryResults := make([]bool, recoveryIntents)

	for i := 0; i < recoveryIntents; i++ {
		recoveryWg.Add(1)
		go func(intentNum int) {
			defer recoveryWg.Done()

			intent := &v1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("recovery-intent-%d", intentNum),
					Namespace: "chaos-test",
				},
				Spec: v1.NetworkIntentSpec{
					IntentType: "recovery_test",
					Intent:     fmt.Sprintf("Recovery test intent %d", intentNum),
					Priority:   v1.NetworkPriorityNormal,
				},
			}

			result, err := suite.engine.ProcessIntentWorkflow(suite.ctx, intent)
			recoveryResults[intentNum] = (err == nil && result != nil && result.Success)
		}(i)
	}

	recoveryWg.Wait()

	// Analyze recovery success
	recoverySuccessCount := 0
	for _, success := range recoveryResults {
		if success {
			recoverySuccessCount++
		}
	}

	recoveryRate := float64(recoverySuccessCount) / float64(recoveryIntents)
	suite.T().Logf("Recovery test: %d/%d intents successful (%.2f%% recovery rate)",
		recoverySuccessCount, recoveryIntents, recoveryRate*100)

	// System should demonstrate good recovery capabilities
	assert.True(suite.T(), recoveryRate >= 0.6,
		"System should demonstrate good recovery (>60%% success rate)")

	// Verify system health after recovery
	health := suite.engine.HealthCheck()
	assert.NoError(suite.T(), health, "System should be healthy after recovery")

	// Check final system state
	finalMetrics := suite.engine.GetMetrics()
	suite.T().Logf("Final metrics after chaos and recovery: %+v", finalMetrics)

	// Verify system learned from chaos (adaptive behavior)
	if finalMetrics.AverageLatency > 0 {
		suite.T().Logf("Post-chaos average latency: %v", finalMetrics.AverageLatency)
	}
}

// monitorSystemDuringChaos monitors system metrics during chaos tests
func (suite *ChaosTestSuite) monitorSystemDuringChaos(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			metrics := suite.engine.GetMetrics()
			if metrics != nil {
				suite.T().Logf("Chaos monitoring - Tasks: %d, Success Rate: %.2f%%, Avg Latency: %v",
					metrics.TotalTasks, metrics.SuccessRate*100, metrics.AverageLatency)
			}

			health := suite.engine.HealthCheck()
			if health != nil {
				suite.T().Logf("Chaos monitoring - Health check failed: %v", health)
			}

		case <-ctx.Done():
			return
		}
	}
}

func TestChaosTestSuite(t *testing.T) {
	suite.Run(t, new(ChaosTestSuite))
}
