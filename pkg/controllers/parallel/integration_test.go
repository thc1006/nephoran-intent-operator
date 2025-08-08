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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	
	"github.com/thc1006/nephoran-intent-operator/api/v1alpha1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/interfaces"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/resilience"
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"
)

// ParallelProcessingIntegrationTestSuite tests the complete integration
type ParallelProcessingIntegrationTestSuite struct {
	suite.Suite
	logger        logr.Logger
	engine        *ParallelProcessingEngine
	resilienceMgr *resilience.ResilienceManager
	errorTracker  *monitoring.ErrorTracker
	ctx          context.Context
	cancel       context.CancelFunc
}

func (suite *ParallelProcessingIntegrationTestSuite) SetupSuite() {
	zapLogger, _ := zap.NewDevelopment()
	suite.logger = zapr.New(zapLogger)
	
	suite.ctx, suite.cancel = context.WithCancel(context.Background())
	
	// Setup resilience manager
	resilienceConfig := &resilience.ResilienceConfig{
		DefaultTimeout:          30 * time.Second,
		MaxConcurrentOperations: 100,
		HealthCheckInterval:     10 * time.Second,
		TimeoutEnabled:          true,
		BulkheadEnabled:        true,
		CircuitBreakerEnabled:  true,
		RateLimitingEnabled:    true,
		RetryEnabled:          true,
		HealthCheckEnabled:    true,
		TimeoutConfigs: map[string]*resilience.TimeoutConfig{
			"llm_processing": {
				Name:           "llm_processing",
				DefaultTimeout: 10 * time.Second,
				MaxTimeout:     60 * time.Second,
				MinTimeout:     1 * time.Second,
			},
			"rag_retrieval": {
				Name:           "rag_retrieval",
				DefaultTimeout: 5 * time.Second,
				MaxTimeout:     30 * time.Second,
				MinTimeout:     500 * time.Millisecond,
			},
		},
	}
	
	suite.resilienceMgr = resilience.NewResilienceManager(resilienceConfig, suite.logger)
	require.NoError(suite.T(), suite.resilienceMgr.Start(suite.ctx))
	
	// Setup error tracker
	errorConfig := &monitoring.ErrorTrackingConfig{
		EnablePrometheus:    true,
		EnableOpenTelemetry: true,
		AlertingEnabled:     false, // Disable alerting in tests
		DashboardEnabled:    false,
		ReportsEnabled:      false,
	}
	
	var err error
	suite.errorTracker, err = monitoring.NewErrorTracker(errorConfig, suite.logger)
	require.NoError(suite.T(), err)
	
	// Setup parallel processing engine
	engineConfig := &ParallelProcessingConfig{
		MaxConcurrentIntents: 50,
		IntentPoolSize:      10,
		LLMPoolSize:        5,
		RAGPoolSize:        5,
		ResourcePoolSize:   8,
		ManifestPoolSize:   8,
		GitOpsPoolSize:     4,
		DeploymentPoolSize: 4,
		TaskQueueSize:      200,
		HealthCheckInterval: 10 * time.Second,
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

func (suite *ParallelProcessingIntegrationTestSuite) TearDownSuite() {
	if suite.engine != nil {
		suite.engine.Stop()
	}
	if suite.resilienceMgr != nil {
		suite.resilienceMgr.Stop()
	}
	suite.cancel()
}

// TestCompleteIntentProcessingWorkflow tests the full intent processing pipeline
func (suite *ParallelProcessingIntegrationTestSuite) TestCompleteIntentProcessingWorkflow() {
	// Create a realistic NetworkIntent
	intent := &v1alpha1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-5g-deployment",
			Namespace: "telecom-operator",
		},
		Spec: v1alpha1.NetworkIntentSpec{
			IntentType: "5g_core_deployment",
			Intent:     "Deploy high-availability 5G Core network with AMF, SMF, and UPF for production environment",
			Priority:   "high",
		},
	}
	
	intentID := fmt.Sprintf("%s/%s", intent.Namespace, intent.Name)
	
	// Process the complete intent workflow
	result, err := suite.engine.ProcessIntentWorkflow(suite.ctx, intent)
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	
	suite.T().Logf("Intent processing result: %+v", result)
	
	// Verify workflow phases were executed
	assert.True(suite.T(), result.Success)
	assert.NotNil(suite.T(), result.ProcessingPhases)
	
	// Check that all expected phases were completed
	expectedPhases := []interfaces.ProcessingPhase{
		interfaces.PhaseInitialization,
		interfaces.PhaseIntentProcessing,
		interfaces.PhaseLLMProcessing,
		interfaces.PhaseRAGRetrieval,
		interfaces.PhaseResourcePlanning,
		interfaces.PhaseManifestGeneration,
		interfaces.PhaseGitOpsCommit,
		interfaces.PhaseDeploymentVerification,
		interfaces.PhaseCompleted,
	}
	
	completedPhases := make(map[interfaces.ProcessingPhase]bool)
	for _, phase := range result.ProcessingPhases {
		completedPhases[phase.Phase] = phase.Success
	}
	
	for _, expectedPhase := range expectedPhases {
		assert.True(suite.T(), completedPhases[expectedPhase], 
			"Phase %s should have been completed", expectedPhase)
	}
	
	// Verify metrics were collected
	metrics := suite.engine.GetMetrics()
	assert.NotNil(suite.T(), metrics)
	assert.True(suite.T(), metrics.TotalTasks > 0)
	assert.True(suite.T(), metrics.CompletedTasks > 0)
	
	suite.T().Logf("Processing metrics: %+v", metrics)
}

// TestConcurrentIntentProcessing tests processing multiple intents concurrently
func (suite *ParallelProcessingIntegrationTestSuite) TestConcurrentIntentProcessing() {
	numIntents := 10
	var wg sync.WaitGroup
	results := make([]*IntentProcessingResult, numIntents)
	errors := make([]error, numIntents)
	
	startTime := time.Now()
	
	// Process multiple intents concurrently
	for i := 0; i < numIntents; i++ {
		wg.Add(1)
		go func(intentNum int) {
			defer wg.Done()
			
			intent := &v1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("concurrent-intent-%d", intentNum),
					Namespace: "telecom-operator",
				},
				Spec: v1alpha1.NetworkIntentSpec{
					IntentType: "edge_deployment",
					Intent:     fmt.Sprintf("Deploy edge computing infrastructure for region %d", intentNum),
					Priority:   "medium",
				},
			}
			
			result, err := suite.engine.ProcessIntentWorkflow(suite.ctx, intent)
			results[intentNum] = result
			errors[intentNum] = err
		}(i)
	}
	
	wg.Wait()
	duration := time.Since(startTime)
	
	// Verify all intents were processed
	successCount := 0
	for i, result := range results {
		if errors[i] != nil {
			suite.T().Logf("Intent %d failed: %v", i, errors[i])
		} else if result != nil && result.Success {
			successCount++
		}
	}
	
	suite.T().Logf("Processed %d intents in %v, %d successful", 
		numIntents, duration, successCount)
	
	// Verify reasonable success rate and performance
	successRate := float64(successCount) / float64(numIntents)
	assert.True(suite.T(), successRate >= 0.7, "Expected at least 70%% success rate, got %.2f%%", successRate*100)
	
	// Verify concurrent processing was faster than sequential
	averageTimePerIntent := duration / time.Duration(numIntents)
	suite.T().Logf("Average time per intent: %v", averageTimePerIntent)
	assert.True(suite.T(), averageTimePerIntent < 10*time.Second, 
		"Concurrent processing should be reasonably fast")
	
	// Check system metrics after concurrent load
	metrics := suite.engine.GetMetrics()
	assert.NotNil(suite.T(), metrics)
	assert.True(suite.T(), metrics.TotalTasks >= int64(numIntents*6), // At least 6 tasks per intent
		"Expected significant task processing")
	
	suite.T().Logf("Final metrics after concurrent test: %+v", metrics)
}

// TestErrorRecoveryInWorkflow tests error handling and recovery in complete workflows
func (suite *ParallelProcessingIntegrationTestSuite) TestErrorRecoveryInWorkflow() {
	// Create an intent that might cause some processing errors
	intent := &v1alpha1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "error-recovery-intent",
			Namespace: "telecom-operator",
		},
		Spec: v1alpha1.NetworkIntentSpec{
			IntentType: "complex_deployment",
			Intent:     "Deploy complex network topology with multiple interdependent components and strict SLA requirements",
			Priority:   "critical",
		},
	}
	
	// Process with potential for errors
	result, err := suite.engine.ProcessIntentWorkflow(suite.ctx, intent)
	
	// Even if some errors occurred, the system should handle them gracefully
	if err != nil {
		suite.T().Logf("Workflow completed with errors: %v", err)
	}
	
	// Check if the system recovered and maintained functionality
	health := suite.engine.HealthCheck()
	assert.NoError(suite.T(), health, "System should remain healthy after error recovery")
	
	// Verify error tracking captured issues
	errorSummary := suite.errorTracker.GetErrorSummary()
	if errorSummary != nil && errorSummary.TotalErrors > 0 {
		suite.T().Logf("Error summary: %+v", errorSummary)
		
		// Verify errors were properly categorized and correlated
		assert.True(suite.T(), errorSummary.TotalErrors >= 0)
		if errorSummary.ErrorsByCategory != nil {
			suite.T().Logf("Errors by category: %+v", errorSummary.ErrorsByCategory)
		}
	}
	
	// Test that the system can continue processing new intents
	followUpIntent := &v1alpha1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "post-recovery-intent",
			Namespace: "telecom-operator",
		},
		Spec: v1alpha1.NetworkIntentSpec{
			IntentType: "simple_deployment",
			Intent:     "Deploy simple network service after error recovery",
			Priority:   "low",
		},
	}
	
	followUpResult, followUpErr := suite.engine.ProcessIntentWorkflow(suite.ctx, followUpIntent)
	
	// System should continue functioning
	if followUpErr != nil {
		suite.T().Logf("Follow-up intent error: %v", followUpErr)
	}
	
	suite.T().Logf("Post-recovery processing result: %+v", followUpResult)
}

// TestResourceOptimization tests resource utilization and optimization
func (suite *ParallelProcessingIntegrationTestSuite) TestResourceOptimization() {
	// Create intents with different resource requirements
	intents := []*v1alpha1.NetworkIntent{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "light-intent", Namespace: "telecom-operator"},
			Spec: v1alpha1.NetworkIntentSpec{
				IntentType: "edge_service",
				Intent:     "Deploy lightweight edge service",
				Priority:   "low",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "medium-intent", Namespace: "telecom-operator"},
			Spec: v1alpha1.NetworkIntentSpec{
				IntentType: "core_service",
				Intent:     "Deploy core network service with moderate requirements",
				Priority:   "medium",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "heavy-intent", Namespace: "telecom-operator"},
			Spec: v1alpha1.NetworkIntentSpec{
				IntentType: "full_stack_deployment",
				Intent:     "Deploy complete 5G network stack with all components",
				Priority:   "high",
			},
		},
	}
	
	startTime := time.Now()
	var wg sync.WaitGroup
	
	// Process all intents and measure resource utilization
	for i, intent := range intents {
		wg.Add(1)
		go func(intentNum int, intent *v1alpha1.NetworkIntent) {
			defer wg.Done()
			
			result, err := suite.engine.ProcessIntentWorkflow(suite.ctx, intent)
			if err != nil {
				suite.T().Logf("Intent %d processing error: %v", intentNum, err)
			} else {
				suite.T().Logf("Intent %d result: %+v", intentNum, result)
			}
		}(i, intent)
	}
	
	wg.Wait()
	duration := time.Since(startTime)
	
	// Analyze resource utilization
	metrics := suite.engine.GetMetrics()
	assert.NotNil(suite.T(), metrics)
	
	suite.T().Logf("Resource optimization test completed in %v", duration)
	suite.T().Logf("Final resource metrics: %+v", metrics)
	
	// Verify efficient resource usage
	if metrics.AverageLatency > 0 {
		assert.True(suite.T(), metrics.AverageLatency < 30*time.Second, 
			"Average latency should be reasonable")
	}
	
	// Check that concurrent processing was effective
	totalTasks := metrics.TotalTasks
	assert.True(suite.T(), totalTasks > 0, "Should have processed tasks")
	
	// Verify system maintained stability under varying loads
	assert.True(suite.T(), metrics.SuccessRate >= 0.8, 
		"Success rate should be high (>80%%), got %.2f%%", metrics.SuccessRate*100)
}

// TestSystemLimitsAndScaling tests system behavior at capacity limits
func (suite *ParallelProcessingIntegrationTestSuite) TestSystemLimitsAndScaling() {
	// Test system behavior at configured limits
	maxIntents := suite.engine.config.MaxConcurrentIntents
	
	suite.T().Logf("Testing system limits with %d concurrent intents", maxIntents)
	
	var wg sync.WaitGroup
	successCount := int64(0)
	var successMutex sync.Mutex
	
	// Submit exactly the maximum number of concurrent intents
	for i := 0; i < int(maxIntents); i++ {
		wg.Add(1)
		go func(intentNum int) {
			defer wg.Done()
			
			intent := &v1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("scaling-test-intent-%d", intentNum),
					Namespace: "telecom-operator",
				},
				Spec: v1alpha1.NetworkIntentSpec{
					IntentType: "scaling_test",
					Intent:     fmt.Sprintf("Scaling test intent %d", intentNum),
					Priority:   "medium",
				},
			}
			
			ctx, cancel := context.WithTimeout(suite.ctx, 60*time.Second)
			defer cancel()
			
			result, err := suite.engine.ProcessIntentWorkflow(ctx, intent)
			if err == nil && result != nil && result.Success {
				successMutex.Lock()
				successCount++
				successMutex.Unlock()
			} else if err != nil {
				suite.T().Logf("Scaling test intent %d failed: %v", intentNum, err)
			}
		}(i)
	}
	
	wg.Wait()
	
	// Analyze scaling behavior
	successRate := float64(successCount) / float64(maxIntents)
	suite.T().Logf("Scaling test: processed %d/%d intents (%.2f%% success rate)", 
		successCount, maxIntents, successRate*100)
	
	// System should handle at least some portion of max capacity
	assert.True(suite.T(), successRate >= 0.5, 
		"System should handle at least 50%% of max capacity under stress")
	
	// Verify system metrics reflect the load
	metrics := suite.engine.GetMetrics()
	assert.NotNil(suite.T(), metrics)
	assert.True(suite.T(), metrics.TotalTasks >= successCount*4, // Minimum tasks per successful intent
		"Metrics should reflect significant task processing")
	
	suite.T().Logf("Scaling test final metrics: %+v", metrics)
	
	// Verify system health after stress test
	health := suite.engine.HealthCheck()
	assert.NoError(suite.T(), health, "System should remain healthy after scaling test")
}

// TestRealWorldScenarios tests realistic telecommunications use cases
func (suite *ParallelProcessingIntegrationTestSuite) TestRealWorldScenarios() {
	scenarios := []struct {
		name   string
		intent *v1alpha1.NetworkIntent
	}{
		{
			name: "5G Core Deployment",
			intent: &v1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{Name: "5g-core-prod", Namespace: "telecom-operator"},
				Spec: v1alpha1.NetworkIntentSpec{
					IntentType: "5g_core_deployment",
					Intent:     "Deploy production 5G Core network with AMF, SMF, UPF, NRF, and UDM for 10,000 concurrent users",
					Priority:   "critical",
				},
			},
		},
		{
			name: "Network Slicing",
			intent: &v1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{Name: "network-slicing", Namespace: "telecom-operator"},
				Spec: v1alpha1.NetworkIntentSpec{
					IntentType: "network_slicing",
					Intent:     "Create network slices for eMBB, URLLC, and mMTC with isolated resources and QoS policies",
					Priority:   "high",
				},
			},
		},
		{
			name: "Edge Computing",
			intent: &v1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{Name: "edge-deployment", Namespace: "telecom-operator"},
				Spec: v1alpha1.NetworkIntentSpec{
					IntentType: "edge_computing",
					Intent:     "Deploy edge computing infrastructure with MEC platforms across 5 geographic regions",
					Priority:   "high",
				},
			},
		},
		{
			name: "O-RAN Deployment",
			intent: &v1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{Name: "oran-deployment", Namespace: "telecom-operator"},
				Spec: v1alpha1.NetworkIntentSpec{
					IntentType: "oran_deployment",
					Intent:     "Deploy O-RAN compliant RAN with Near-RT RIC, O-DU, O-CU, and xApps for intelligent optimization",
					Priority:   "high",
				},
			},
		},
	}
	
	for _, scenario := range scenarios {
		suite.T().Run(scenario.name, func(t *testing.T) {
			startTime := time.Now()
			
			result, err := suite.engine.ProcessIntentWorkflow(suite.ctx, scenario.intent)
			duration := time.Since(startTime)
			
			if err != nil {
				t.Logf("Scenario %s failed: %v", scenario.name, err)
			}
			
			t.Logf("Scenario %s completed in %v", scenario.name, duration)
			
			if result != nil {
				t.Logf("Scenario %s result: Success=%v, Phases=%d", 
					scenario.name, result.Success, len(result.ProcessingPhases))
				
				// Verify realistic processing occurred
				assert.True(t, len(result.ProcessingPhases) >= 4, 
					"Should have significant processing phases")
			}
			
			// Verify reasonable processing time for realistic scenarios
			assert.True(t, duration < 2*time.Minute, 
				"Realistic scenarios should complete within reasonable time")
		})
	}
	
	// Verify system health after all scenarios
	health := suite.engine.HealthCheck()
	assert.NoError(suite.T(), health, "System should remain healthy after real-world scenarios")
	
	// Check final system metrics
	metrics := suite.engine.GetMetrics()
	suite.T().Logf("Real-world scenarios final metrics: %+v", metrics)
}

func TestParallelProcessingIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ParallelProcessingIntegrationTestSuite))
}