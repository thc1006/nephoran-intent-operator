//go:build integration

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// Phase transition integration tests
func TestNetworkIntentPhaseTransitions(t *testing.T) {
	scheme := runtime.NewScheme()
	nephoranv1.AddToScheme(scheme)

	tests := []struct {
		name               string
		initialPhase       string
		initialConditions  []metav1.Condition
		intent             string
		mockSetup          func(*MockDependencies)
		expectedPhase      string
		expectedConditions []string // Condition types that should be present
		expectedRequeue    bool
		validationChecks   func(t *testing.T, ni *nephoranv1.NetworkIntent)
	}{
		{
			name:         "Pending to Processing transition",
			initialPhase: "Pending",
			initialConditions: []metav1.Condition{
				{
					Type:   "Ready",
					Status: metav1.ConditionFalse,
					Reason: "Pending",
				},
			},
			intent: "Deploy AMF network function for 5G core",
			mockSetup: func(deps *MockDependencies) {
				llmResponse := map[string]interface{}{
					"action":    "deploy",
					"component": "amf",
					"namespace": "5g-core",
					"replicas":  1,
				}
				responseJSON, _ := json.Marshal(llmResponse)
				deps.llmClient.SetResponse(string(responseJSON))
			},
			expectedPhase:      "Processing",
			expectedConditions: []string{"Validated", "Ready"},
			expectedRequeue:    false,
			validationChecks: func(t *testing.T, ni *nephoranv1.NetworkIntent) {
				assert.Equal(t, "Processing", ni.Status.Phase)
				assert.True(t, hasCondition(ni.Status.Conditions, "Validated"))
				assert.NotEmpty(t, ni.Spec.Parameters.Raw)
			},
		},
		{
			name:         "Processing to Error on LLM failure",
			initialPhase: "Processing",
			initialConditions: []metav1.Condition{
				{
					Type:   "Ready",
					Status: metav1.ConditionFalse,
					Reason: "Processing",
				},
			},
			intent: "Deploy SMF network function",
			mockSetup: func(deps *MockDependencies) {
				deps.llmClient.SetError(fmt.Errorf("LLM service temporarily unavailable"))
			},
			expectedPhase:      "Error",
			expectedConditions: []string{"Processed", "Ready"},
			expectedRequeue:    true,
			validationChecks: func(t *testing.T, ni *nephoranv1.NetworkIntent) {
				assert.Equal(t, "Error", ni.Status.Phase)
				assert.True(t, hasConditionWithStatus(ni.Status.Conditions, "Processed", metav1.ConditionFalse))
				assert.True(t, hasConditionWithStatus(ni.Status.Conditions, "Ready", metav1.ConditionFalse))
				assert.Contains(t, getConditionMessage(ni.Status.Conditions, "Processed"), "LLM")
			},
		},
		{
			name:         "Error to Processing on retry success",
			initialPhase: "Error",
			initialConditions: []metav1.Condition{
				{
					Type:    "Processed",
					Status:  metav1.ConditionFalse,
					Reason:  "LLMProcessingFailed",
					Message: "LLM service temporarily unavailable",
				},
				{
					Type:   "Ready",
					Status: metav1.ConditionFalse,
					Reason: "Error",
				},
			},
			intent: "Deploy UPF network function",
			mockSetup: func(deps *MockDependencies) {
				// First call fails, second call succeeds
				deps.llmClient.SetFailCount(1)
				llmResponse := map[string]interface{}{
					"action":    "deploy",
					"component": "upf",
					"namespace": "5g-core",
				}
				responseJSON, _ := json.Marshal(llmResponse)
				deps.llmClient.SetResponse(string(responseJSON))
			},
			expectedPhase:      "Processing",
			expectedConditions: []string{"Processed", "Ready", "Validated"},
			expectedRequeue:    false,
			validationChecks: func(t *testing.T, ni *nephoranv1.NetworkIntent) {
				assert.Equal(t, "Processing", ni.Status.Phase)
				assert.True(t, hasConditionWithStatus(ni.Status.Conditions, "Processed", metav1.ConditionTrue))
				assert.NotEmpty(t, ni.Spec.Parameters.Raw)
			},
		},
		{
			name:         "Processing to Processed on successful completion",
			initialPhase: "Processing",
			initialConditions: []metav1.Condition{
				{
					Type:   "Validated",
					Status: metav1.ConditionTrue,
					Reason: "ValidationSuccessful",
				},
				{
					Type:   "Ready",
					Status: metav1.ConditionFalse,
					Reason: "Processing",
				},
			},
			intent: "Configure NSSF for network slicing",
			mockSetup: func(deps *MockDependencies) {
				llmResponse := map[string]interface{}{
					"action":    "configure",
					"component": "nssf",
					"namespace": "5g-core",
					"slicing_config": map[string]interface{}{
						"slice_types": []string{"embb", "urllc"},
					},
				}
				responseJSON, _ := json.Marshal(llmResponse)
				deps.llmClient.SetResponse(string(responseJSON))
				// Mock successful git operations
				deps.gitClient.SetShouldFail(false)
			},
			expectedPhase:      "Processed",
			expectedConditions: []string{"Processed", "Ready", "Validated", "Deployed"},
			expectedRequeue:    false,
			validationChecks: func(t *testing.T, ni *nephoranv1.NetworkIntent) {
				assert.Equal(t, "Processed", ni.Status.Phase)
				assert.True(t, hasConditionWithStatus(ni.Status.Conditions, "Processed", metav1.ConditionTrue))
				assert.True(t, hasConditionWithStatus(ni.Status.Conditions, "Ready", metav1.ConditionTrue))
				assert.True(t, hasConditionWithStatus(ni.Status.Conditions, "Deployed", metav1.ConditionTrue))
			},
		},
		{
			name:         "Processing to Error on Git failure",
			initialPhase: "Processing",
			initialConditions: []metav1.Condition{
				{
					Type:   "Validated",
					Status: metav1.ConditionTrue,
					Reason: "ValidationSuccessful",
				},
				{
					Type:   "Processed",
					Status: metav1.ConditionTrue,
					Reason: "LLMProcessingSuccessful",
				},
			},
			intent: "Deploy 5G core network functions",
			mockSetup: func(deps *MockDependencies) {
				llmResponse := map[string]interface{}{
					"action":    "deploy",
					"component": "5gc-core",
					"namespace": "5g-core",
				}
				responseJSON, _ := json.Marshal(llmResponse)
				deps.llmClient.SetResponse(string(responseJSON))
				// Mock git failure
				deps.gitClient.SetShouldFail(true)
			},
			expectedPhase:      "Error",
			expectedConditions: []string{"Deployed", "Ready"},
			expectedRequeue:    true,
			validationChecks: func(t *testing.T, ni *nephoranv1.NetworkIntent) {
				assert.Equal(t, "Error", ni.Status.Phase)
				assert.True(t, hasConditionWithStatus(ni.Status.Conditions, "Deployed", metav1.ConditionFalse))
				assert.True(t, hasConditionWithStatus(ni.Status.Conditions, "Ready", metav1.ConditionFalse))
				assert.Contains(t, getConditionMessage(ni.Status.Conditions, "Deployed"), "git")
			},
		},
		{
			name:               "Empty intent handling",
			initialPhase:       "Pending",
			initialConditions:  []metav1.Condition{},
			intent:             "",
			mockSetup:          func(deps *MockDependencies) {},
			expectedPhase:      "Error",
			expectedConditions: []string{"Validated", "Ready"},
			expectedRequeue:    false,
			validationChecks: func(t *testing.T, ni *nephoranv1.NetworkIntent) {
				assert.Equal(t, "Error", ni.Status.Phase)
				assert.True(t, hasConditionWithStatus(ni.Status.Conditions, "Validated", metav1.ConditionFalse))
				assert.Contains(t, getConditionMessage(ni.Status.Conditions, "Validated"), "empty")
			},
		},
		{
			name:         "Maximum retries reached",
			initialPhase: "Error",
			initialConditions: []metav1.Condition{
				{
					Type:    "Processed",
					Status:  metav1.ConditionFalse,
					Reason:  "MaxRetriesReached",
					Message: "Maximum retry attempts exceeded",
				},
			},
			intent: "Deploy AMF with retry test",
			mockSetup: func(deps *MockDependencies) {
				// Always fail to simulate max retries scenario
				deps.llmClient.SetError(fmt.Errorf("persistent failure"))
			},
			expectedPhase:      "Error",
			expectedConditions: []string{"Processed", "Ready"},
			expectedRequeue:    true, // Will still requeue but with longer delay
			validationChecks: func(t *testing.T, ni *nephoranv1.NetworkIntent) {
				assert.Equal(t, "Error", ni.Status.Phase)
				assert.True(t, hasConditionWithStatus(ni.Status.Conditions, "Processed", metav1.ConditionFalse))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test NetworkIntent with initial phase and conditions
			ni := createTestNetworkIntent(fmt.Sprintf("test-%s", tt.name), "default", tt.intent)
			ni.Status.Phase = tt.initialPhase
			ni.Status.Conditions = tt.initialConditions

			// Update condition timestamps
			now := metav1.NewTime(time.Now())
			for i := range ni.Status.Conditions {
				ni.Status.Conditions[i].LastTransitionTime = now
			}

			// Setup fake client
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ni).Build()

			// Setup mock dependencies
			mockDeps := NewMockDependencies()
			tt.mockSetup(mockDeps)

			// Create reconciler
			reconciler, err := NewNetworkIntentReconciler(fakeClient, scheme, mockDeps, createTestConfig())
			require.NoError(t, err)

			// Create reconcile request
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      ni.Name,
					Namespace: ni.Namespace,
				},
			}

			// Execute reconciliation
			ctx := context.Background()
			result, err := reconciler.Reconcile(ctx, req)
			require.NoError(t, err)

			// Verify requeue expectation
			if tt.expectedRequeue {
				assert.True(t, result.Requeue || result.RequeueAfter > 0, "Expected requeue but got %+v", result)
			} else {
				assert.False(t, result.Requeue, "Did not expect requeue but got %+v", result)
				assert.Equal(t, time.Duration(0), result.RequeueAfter, "Did not expect requeue after but got %+v", result)
			}

			// Get updated NetworkIntent
			updatedNI := &nephoranv1.NetworkIntent{}
			err = fakeClient.Get(ctx, types.NamespacedName{
				Name:      ni.Name,
				Namespace: ni.Namespace,
			}, updatedNI)
			require.NoError(t, err)

			// Verify phase transition
			assert.Equal(t, tt.expectedPhase, updatedNI.Status.Phase)

			// Verify expected conditions are present
			for _, condType := range tt.expectedConditions {
				assert.True(t, hasCondition(updatedNI.Status.Conditions, condType),
					"Expected condition %s not found in conditions: %+v", condType, updatedNI.Status.Conditions)
			}

			// Run custom validation checks
			if tt.validationChecks != nil {
				tt.validationChecks(t, updatedNI)
			}
		})
	}
}

// Test complex multi-phase processing pipeline
func TestNetworkIntentMultiPhaseProcessing(t *testing.T) {
	scheme := runtime.NewScheme()
	nephoranv1.AddToScheme(scheme)

	// Create test NetworkIntent
	ni := createTestNetworkIntent("multi-phase-test", "default", "Deploy comprehensive 5G core with AMF, SMF, and UPF")
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ni).Build()

	// Setup mock dependencies for successful flow
	mockDeps := NewMockDependencies()
	llmResponse := map[string]interface{}{
		"action":     "deploy",
		"component":  "5gc-core",
		"namespace":  "5g-core",
		"components": []string{"amf", "smf", "upf"},
		"resources": map[string]interface{}{
			"cpu":    "2000m",
			"memory": "4Gi",
		},
	}
	responseJSON, _ := json.Marshal(llmResponse)
	mockDeps.llmClient.SetResponse(string(responseJSON))
	mockDeps.gitClient.SetShouldFail(false)

	// Create reconciler
	reconciler, err := NewNetworkIntentReconciler(fakeClient, scheme, mockDeps, createTestConfig())
	require.NoError(t, err)

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      ni.Name,
			Namespace: ni.Namespace,
		},
	}

	ctx := context.Background()

	// Execute multiple reconciliation cycles to simulate pipeline progression
	phaseProgression := []string{}
	maxCycles := 10

	for cycle := 0; cycle < maxCycles; cycle++ {
		// Execute reconciliation
		result, err := reconciler.Reconcile(ctx, req)
		require.NoError(t, err, "Reconciliation failed at cycle %d", cycle)

		// Get current state
		currentNI := &nephoranv1.NetworkIntent{}
		err = fakeClient.Get(ctx, types.NamespacedName{
			Name:      ni.Name,
			Namespace: ni.Namespace,
		}, currentNI)
		require.NoError(t, err)

		// Track phase progression
		if len(phaseProgression) == 0 || phaseProgression[len(phaseProgression)-1] != currentNI.Status.Phase {
			phaseProgression = append(phaseProgression, currentNI.Status.Phase)
		}

		// Check if we've reached a terminal state
		if currentNI.Status.Phase == "Processed" || currentNI.Status.Phase == "Error" {
			if !result.Requeue && result.RequeueAfter == 0 {
				break
			}
		}

		// Prevent infinite loops in testing
		if cycle >= maxCycles-1 {
			t.Logf("Reached maximum test cycles (%d), current phase: %s", maxCycles, currentNI.Status.Phase)
			break
		}

		// Add small delay to prevent tight loops
		time.Sleep(10 * time.Millisecond)
	}

	// Verify final state
	finalNI := &nephoranv1.NetworkIntent{}
	err = fakeClient.Get(ctx, types.NamespacedName{
		Name:      ni.Name,
		Namespace: ni.Namespace,
	}, finalNI)
	require.NoError(t, err)

	// Assertions for successful multi-phase processing
	t.Logf("Phase progression: %v", phaseProgression)
	assert.Contains(t, []string{"Processed", "Processing"}, finalNI.Status.Phase,
		"Expected final phase to be Processed or Processing, got %s", finalNI.Status.Phase)

	// Verify that we progressed through expected phases
	assert.Contains(t, phaseProgression, "Pending", "Should start with Pending phase")
	if finalNI.Status.Phase == "Processed" {
		assert.Contains(t, phaseProgression, "Processing", "Should progress through Processing phase")
	}

	// Verify key conditions are set appropriately
	if finalNI.Status.Phase == "Processed" {
		assert.True(t, hasConditionWithStatus(finalNI.Status.Conditions, "Ready", metav1.ConditionTrue),
			"Ready condition should be True for Processed phase")
		assert.True(t, hasConditionWithStatus(finalNI.Status.Conditions, "Processed", metav1.ConditionTrue),
			"Processed condition should be True for Processed phase")
	}

	// Verify parameters were extracted
	if len(finalNI.Spec.Parameters.Raw) > 0 {
		var extractedParams map[string]interface{}
		err = json.Unmarshal(finalNI.Spec.Parameters.Raw, &extractedParams)
		assert.NoError(t, err, "Should be able to unmarshal extracted parameters")
		assert.Equal(t, "deploy", extractedParams["action"], "Action should be extracted correctly")
	}
}

// Test phase transition error recovery
func TestPhaseTransitionErrorRecovery(t *testing.T) {
	scheme := runtime.NewScheme()
	nephoranv1.AddToScheme(scheme)

	tests := []struct {
		name               string
		initialPhase       string
		failureSetup       func(*MockDependencies)
		recoverySetup      func(*MockDependencies)
		expectedRecovery   bool
		expectedFinalPhase string
	}{
		{
			name:         "LLM failure with recovery",
			initialPhase: "Processing",
			failureSetup: func(deps *MockDependencies) {
				deps.llmClient.SetError(fmt.Errorf("temporary LLM failure"))
				deps.llmClient.SetFailCount(2)
			},
			recoverySetup: func(deps *MockDependencies) {
				llmResponse := map[string]interface{}{
					"action":    "deploy",
					"component": "amf",
				}
				responseJSON, _ := json.Marshal(llmResponse)
				deps.llmClient.SetResponse(string(responseJSON))
			},
			expectedRecovery:   true,
			expectedFinalPhase: "Processing",
		},
		{
			name:         "Git failure with recovery",
			initialPhase: "Processing",
			failureSetup: func(deps *MockDependencies) {
				// Setup successful LLM but failing Git
				llmResponse := map[string]interface{}{
					"action":    "deploy",
					"component": "smf",
				}
				responseJSON, _ := json.Marshal(llmResponse)
				deps.llmClient.SetResponse(string(responseJSON))
				deps.gitClient.SetShouldFail(true)
			},
			recoverySetup: func(deps *MockDependencies) {
				// Fix git operations
				deps.gitClient.SetShouldFail(false)
			},
			expectedRecovery:   true,
			expectedFinalPhase: "Processed",
		},
		{
			name:         "Persistent failure no recovery",
			initialPhase: "Processing",
			failureSetup: func(deps *MockDependencies) {
				deps.llmClient.SetError(fmt.Errorf("persistent failure"))
			},
			recoverySetup: func(deps *MockDependencies) {
				// No recovery - keep the failure
			},
			expectedRecovery:   false,
			expectedFinalPhase: "Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test NetworkIntent
			ni := createTestNetworkIntent(fmt.Sprintf("recovery-%s", tt.name), "default", "Deploy test network function")
			ni.Status.Phase = tt.initialPhase
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ni).Build()

			// Setup mock dependencies for failure
			mockDeps := NewMockDependencies()
			tt.failureSetup(mockDeps)

			reconciler, err := NewNetworkIntentReconciler(fakeClient, scheme, mockDeps, createTestConfig())
			require.NoError(t, err)

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      ni.Name,
					Namespace: ni.Namespace,
				},
			}
			ctx := context.Background()

			// First reconciliation should result in error
			result1, err := reconciler.Reconcile(ctx, req)
			require.NoError(t, err)

			// Check that it moved to error state and is scheduled for retry
			errorNI := &nephoranv1.NetworkIntent{}
			err = fakeClient.Get(ctx, types.NamespacedName{Name: ni.Name, Namespace: ni.Namespace}, errorNI)
			require.NoError(t, err)
			assert.Equal(t, "Error", errorNI.Status.Phase)
			assert.True(t, result1.Requeue || result1.RequeueAfter > 0)

			// Setup recovery conditions
			tt.recoverySetup(mockDeps)

			// Second reconciliation should attempt recovery
			result2, err := reconciler.Reconcile(ctx, req)
			require.NoError(t, err)

			// Verify final state
			finalNI := &nephoranv1.NetworkIntent{}
			err = fakeClient.Get(ctx, types.NamespacedName{Name: ni.Name, Namespace: ni.Namespace}, finalNI)
			require.NoError(t, err)

			if tt.expectedRecovery {
				assert.NotEqual(t, "Error", finalNI.Status.Phase, "Should recover from error state")
				if tt.expectedFinalPhase != "" {
					// Allow some flexibility in final phase for multi-step processes
					assert.Contains(t, []string{tt.expectedFinalPhase, "Processing"}, finalNI.Status.Phase)
				}
			} else {
				assert.Equal(t, tt.expectedFinalPhase, finalNI.Status.Phase)
				assert.True(t, result2.Requeue || result2.RequeueAfter > 0, "Should continue retrying")
			}
		})
	}
}

// Helper functions for condition checking
func hasCondition(conditions []metav1.Condition, conditionType string) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return true
		}
	}
	return false
}

func hasConditionWithStatus(conditions []metav1.Condition, conditionType string, status metav1.ConditionStatus) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition.Status == status
		}
	}
	return false
}

func getConditionMessage(conditions []metav1.Condition, conditionType string) string {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition.Message
		}
	}
	return ""
}

// Benchmark phase transition performance
func BenchmarkPhaseTransition(b *testing.B) {
	scheme := runtime.NewScheme()
	nephoranv1.AddToScheme(scheme)

	ni := createTestNetworkIntent("benchmark-transition", "default", "Deploy AMF network function")
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ni).Build()

	mockDeps := NewMockDependencies()
	llmResponse := map[string]interface{}{
		"action":    "deploy",
		"component": "amf",
	}
	responseJSON, _ := json.Marshal(llmResponse)
	mockDeps.llmClient.SetResponse(string(responseJSON))

	reconciler, _ := NewNetworkIntentReconciler(fakeClient, scheme, mockDeps, createTestConfig())

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      ni.Name,
			Namespace: ni.Namespace,
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reconciler.Reconcile(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}
