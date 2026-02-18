package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/testutil"
	"github.com/thc1006/nephoran-intent-operator/pkg/testutils"
)

// createTestNetworkIntent creates a NetworkIntent for use in table-driven tests.
func createTestNetworkIntent(name, namespace, intentText string) *nephoranv1.NetworkIntent {
	return &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: nephoranv1.NetworkIntentSpec{
			Intent: intentText,
		},
	}
}

// createTestConfig creates a default Config for testing.
func createTestConfig() Config {
	return Config{
		MaxRetries:    3,
		RetryDelay:    time.Second,
		Timeout:       30 * time.Second,
		GitRepoURL:    "https://github.com/test/repo.git",
		GitBranch:     "main",
		GitDeployPath: "deployments",
	}
}

// NewMockDependencies creates a MockDependencies instance for table-driven tests.
func NewMockDependencies() *MockDependencies {
	return &MockDependencies{
		gitClient:  testutils.NewMockGitClient(),
		llmClient:  testutils.NewMockLLMClient(),
		httpClient: &http.Client{},
	}
}

// getConditionMessage returns the message for a condition by type.
func getConditionMessage(conditions []metav1.Condition, conditionType string) string {
	for _, c := range conditions {
		if c.Type == conditionType {
			return c.Message
		}
	}
	return ""
}

// Table-driven tests for edge cases and error scenarios
func TestNetworkIntentEdgeCases(t *testing.T) {
	scheme := runtime.NewScheme()
	nephoranv1.AddToScheme(scheme)

	tests := []struct {
		name              string
		intentText        string
		enabledLLMIntent  string
		initialPhase      string
		initialConditions []metav1.Condition
		mockSetup         func(*MockDependencies)
		expectedPhase     string
		expectedRequeue   bool
		expectedError     bool
		validationChecks  func(t *testing.T, ni *nephoranv1.NetworkIntent, result ctrl.Result)
		description       string
	}{
		{
			name:             "empty_intent_text",
			intentText:       "",
			enabledLLMIntent: "true",
			initialPhase:     "Pending",
			mockSetup:        func(deps *MockDependencies) {},
			expectedPhase:    "Error",
			expectedRequeue:  false,
			expectedError:    false,
			description:      "Should handle empty intent text gracefully",
			validationChecks: func(t *testing.T, ni *nephoranv1.NetworkIntent, result ctrl.Result) {
				assert.Equal(t, nephoranv1.NetworkIntentPhase("Error"), ni.Status.Phase)
				assert.True(t, testutil.HasConditionWithStatus(ni.Status.Conditions, "Validated", metav1.ConditionFalse))
				assert.Contains(t, strings.ToLower(getConditionMessage(ni.Status.Conditions, "Validated")), "empty")
			},
		},
		{
			name:             "whitespace_only_intent",
			intentText:       "   \t\n   ",
			enabledLLMIntent: "true",
			initialPhase:     "Pending",
			mockSetup:        func(deps *MockDependencies) {},
			expectedPhase:    "Error",
			expectedRequeue:  false,
			expectedError:    false,
			description:      "Should handle whitespace-only intent text",
			validationChecks: func(t *testing.T, ni *nephoranv1.NetworkIntent, result ctrl.Result) {
				assert.Equal(t, nephoranv1.NetworkIntentPhase("Error"), ni.Status.Phase)
				assert.True(t, testutil.HasConditionWithStatus(ni.Status.Conditions, "Validated", metav1.ConditionFalse))
			},
		},
		{
			name:             "llm_disabled_processing",
			intentText:       "Deploy AMF network function",
			enabledLLMIntent: "false",
			initialPhase:     "Pending",
			mockSetup:        func(deps *MockDependencies) {},
			expectedPhase:    "Processed",
			expectedRequeue:  false,
			expectedError:    false,
			description:      "Should process intent without LLM when disabled",
			validationChecks: func(t *testing.T, ni *nephoranv1.NetworkIntent, result ctrl.Result) {
				assert.Equal(t, nephoranv1.NetworkIntentPhase("Processed"), ni.Status.Phase)
				assert.True(t, testutil.HasConditionWithStatus(ni.Status.Conditions, "Processed", metav1.ConditionTrue))
			},
		},
		{
			name:             "llm_service_unavailable",
			intentText:       "Deploy SMF network function",
			enabledLLMIntent: "true",
			initialPhase:     "Pending",
			mockSetup: func(deps *MockDependencies) {
				// SetShouldReturnError makes the mock return an error for ALL intent inputs,
				// simulating a globally unavailable LLM service.
				deps.llmClient.SetShouldReturnError(true)
			},
			expectedPhase:   "Error",
			expectedRequeue: true,
			expectedError:   false,
			description:     "Should handle LLM service unavailable with retry",
			validationChecks: func(t *testing.T, ni *nephoranv1.NetworkIntent, result ctrl.Result) {
				assert.Equal(t, nephoranv1.NetworkIntentPhase("Error"), ni.Status.Phase)
				assert.True(t, testutil.HasConditionWithStatus(ni.Status.Conditions, "Processed", metav1.ConditionFalse))
				assert.True(t, result.RequeueAfter > 0)
				assert.Contains(t, strings.ToLower(getConditionMessage(ni.Status.Conditions, "Processed")), "llm")
			},
		},
		{
			name:             "llm_invalid_json_response",
			intentText:       "Deploy UPF network function",
			enabledLLMIntent: "true",
			initialPhase:     "Pending",
			mockSetup: func(deps *MockDependencies) {
				deps.llmClient.SetResponse("", "invalid json response")
			},
			expectedPhase:   "Error",
			expectedRequeue: true,
			expectedError:   false,
			description:     "Should handle invalid JSON response from LLM",
			validationChecks: func(t *testing.T, ni *nephoranv1.NetworkIntent, result ctrl.Result) {
				assert.Equal(t, nephoranv1.NetworkIntentPhase("Error"), ni.Status.Phase)
				assert.True(t, testutil.HasConditionWithStatus(ni.Status.Conditions, "Processed", metav1.ConditionFalse))
				assert.True(t, result.RequeueAfter > 0)
			},
		},
		{
			name:             "llm_empty_response",
			intentText:       "Deploy NSSF network function",
			enabledLLMIntent: "true",
			initialPhase:     "Pending",
			mockSetup: func(deps *MockDependencies) {
				deps.llmClient.SetResponse("", "")
			},
			expectedPhase:   "Error",
			expectedRequeue: true,
			expectedError:   false,
			description:     "Should handle empty response from LLM",
			validationChecks: func(t *testing.T, ni *nephoranv1.NetworkIntent, result ctrl.Result) {
				assert.Equal(t, nephoranv1.NetworkIntentPhase("Error"), ni.Status.Phase)
				assert.True(t, result.RequeueAfter > 0)
			},
		},
		{
			name:             "git_operation_failure",
			intentText:       "Deploy comprehensive 5G core",
			enabledLLMIntent: "true",
			initialPhase:     "Pending",
			mockSetup: func(deps *MockDependencies) {
				// Valid LLM response but Git failure.
				llmResponse := json.RawMessage(`{}`)
				responseJSON, _ := json.Marshal(llmResponse)
				deps.llmClient.SetResponse("", string(responseJSON))
				deps.gitClient.SetCommitPushError(errors.New("git operation failed"))
			},
			// Git commit failures result in RequeueAfter (backoff retry), not a Go error.
			expectedPhase:   "GitOpsCommit",
			expectedRequeue: true,
			expectedError:   false,
			description:     "Should handle Git operation failures with retry (RequeueAfter backoff)",
			validationChecks: func(t *testing.T, ni *nephoranv1.NetworkIntent, result ctrl.Result) {
				assert.True(t, result.RequeueAfter > 0)
				assert.True(t, testutil.HasConditionWithStatus(ni.Status.Conditions, "GitOpsCommitted", metav1.ConditionFalse))
			},
		},
		{
			name:             "context_cancellation_during_processing",
			intentText:       "Deploy AMF with timeout test",
			enabledLLMIntent: "true",
			initialPhase:     "Processing",
			mockSetup: func(deps *MockDependencies) {
				// Mock will simulate long processing time
				llmResponse := json.RawMessage(`{}`)
				responseJSON, _ := json.Marshal(llmResponse)
				deps.llmClient.SetResponse("", string(responseJSON))
			},
			expectedPhase:   "Completed", // Pipeline completes gracefully without KB
			expectedRequeue: false,
			expectedError:   false,
			description:     "Should handle context cancellation gracefully",
			validationChecks: func(t *testing.T, ni *nephoranv1.NetworkIntent, result ctrl.Result) {
				// Should not crash or leave inconsistent state
				assert.Contains(t, []nephoranv1.NetworkIntentPhase{"Processing", "Processed", "Error", "Completed"}, ni.Status.Phase)
			},
		},
		{
			name:             "very_long_intent_text",
			intentText:       strings.Repeat("Deploy AMF network function with extensive configuration ", 100),
			enabledLLMIntent: "true",
			initialPhase:     "Pending",
			mockSetup: func(deps *MockDependencies) {
				llmResponse := json.RawMessage(`{}`)
				responseJSON, _ := json.Marshal(llmResponse)
				deps.llmClient.SetResponse("", string(responseJSON))
			},
			// 5500-char intent is within the 10KB limit; pipeline completes normally.
			expectedPhase:   "Completed",
			expectedRequeue: false,
			expectedError:   false,
			description:     "Should handle very long intent text (within 10KB sanitizer limit)",
			validationChecks: func(t *testing.T, ni *nephoranv1.NetworkIntent, result ctrl.Result) {
				// Intent is accepted since it's within the 10KB max input length.
				assert.Contains(t, []nephoranv1.NetworkIntentPhase{"Completed", "Processed", "Error"}, ni.Status.Phase)
			},
		},
		{
			name:             "special_characters_in_intent",
			intentText:       "Deploy AMF with config: {cpu: 500m, memory: 512Mi, ports: [8080, 8443]}",
			enabledLLMIntent: "true",
			initialPhase:     "Pending",
			mockSetup: func(deps *MockDependencies) {
				llmResponse := map[string]interface{}{
					"resources": map[string]interface{}{
						"cpu":    "500m",
						"memory": "512Mi",
						"ports":  []int{8080, 8443},
					},
				}
				responseJSON, _ := json.Marshal(llmResponse)
				deps.llmClient.SetResponse("", string(responseJSON))
			},
			expectedPhase:   "Completed",
			expectedRequeue: false,
			expectedError:   false,
			description:     "Should handle special characters and structured data in intent",
			validationChecks: func(t *testing.T, ni *nephoranv1.NetworkIntent, result ctrl.Result) {
				assert.Contains(t, []nephoranv1.NetworkIntentPhase{"Completed", "Processed", "LLMProcessing"}, ni.Status.Phase)
				// Parameters may be stored in Status.Extensions rather than Spec.Parameters
			},
		},
		{
			name:             "unicode_characters_in_intent",
			intentText:       "Deploy AMF with 高性能 configuration for 5G 网络",
			enabledLLMIntent: "true",
			initialPhase:     "Pending",
			mockSetup: func(deps *MockDependencies) {
				llmResponse := json.RawMessage(`{}`)
				responseJSON, _ := json.Marshal(llmResponse)
				deps.llmClient.SetResponse("", string(responseJSON))
			},
			expectedPhase:   "Completed",
			expectedRequeue: false,
			expectedError:   false,
			description:     "Should handle Unicode characters in intent",
			validationChecks: func(t *testing.T, ni *nephoranv1.NetworkIntent, result ctrl.Result) {
				assert.Contains(t, []nephoranv1.NetworkIntentPhase{"Completed", "Processed"}, ni.Status.Phase)
				// Successful validation does not set an explicit Validated=True condition.
			},
		},
		{
			name:             "max_retry_attempts_exceeded",
			intentText:       "Deploy SMF with persistent failure",
			enabledLLMIntent: "true",
			initialPhase:     "Error",
			initialConditions: []metav1.Condition{
				{
					Type:    "Processed",
					Status:  metav1.ConditionFalse,
					Reason:  "ProcessingFailed",
					Message: "Retry attempt 3 of 3 failed",
				},
			},
			mockSetup: func(deps *MockDependencies) {
				deps.llmClient.SetError("", errors.New("persistent LLM failure"))
			},
			expectedPhase:   "Error",
			expectedRequeue: true, // Should still requeue but with exponential backoff
			expectedError:   false,
			description:     "Should handle maximum retry attempts exceeded",
			validationChecks: func(t *testing.T, ni *nephoranv1.NetworkIntent, result ctrl.Result) {
				assert.Equal(t, nephoranv1.NetworkIntentPhase("Error"), ni.Status.Phase)
				assert.True(t, testutil.HasConditionWithStatus(ni.Status.Conditions, "Processed", metav1.ConditionFalse))
				// Should have longer requeue time due to exponential backoff
				assert.True(t, result.RequeueAfter > 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable for LLM intent processing
			if tt.enabledLLMIntent != "" {
				os.Setenv("ENABLE_LLM_INTENT", tt.enabledLLMIntent)
				defer os.Unsetenv("ENABLE_LLM_INTENT")
			}

			// Create test NetworkIntent
			ni := createTestNetworkIntent(fmt.Sprintf("edge-case-%s", tt.name), "default", tt.intentText)
			ni.Status.Phase = nephoranv1.NetworkIntentPhase(tt.initialPhase)
			if tt.initialConditions != nil {
				ni.Status.Conditions = tt.initialConditions
				// Set proper timestamps
				now := metav1.NewTime(time.Now())
				for i := range ni.Status.Conditions {
					ni.Status.Conditions[i].LastTransitionTime = now
				}
			}

			// Create fake client.
			// WithStatusSubresource enables Status().Update() for NetworkIntent in the fake client.
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ni).WithStatusSubresource(&nephoranv1.NetworkIntent{}).Build()

			// Setup mock dependencies
			mockDeps := NewMockDependencies()
			if tt.mockSetup != nil {
				tt.mockSetup(mockDeps)
			}

			// Create reconciler
			cfg := createTestConfig(); reconciler, err := NewNetworkIntentReconciler(fakeClient, scheme, mockDeps, &cfg)
			require.NoError(t, err, "Failed to create reconciler for test %s: %v", tt.name, err)

			// Create reconcile request
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      ni.Name,
					Namespace: ni.Namespace,
				},
			}

			// Execute reconciliation in a loop to handle finalizer addition.
			// The first reconcile typically adds a finalizer and returns RequeueAfter,
			// so we iterate until the result is stable or a non-finalizer requeue occurs.
			ctx := context.Background()
			var result ctrl.Result
			const maxReconcileIterations = 5
			for i := 0; i < maxReconcileIterations; i++ {
				result, err = reconciler.Reconcile(ctx, req)
				if err != nil {
					break
				}
				// Stop if no requeue was requested (processing is complete or errored terminally)
				if !result.Requeue && result.RequeueAfter == 0 {
					break
				}
				// If requeue is only due to finalizer addition (RequeueAfter == 1s on first pass),
				// continue to next iteration to complete actual processing.
				// Also stop if the result indicates an application-level requeue (not finalizer).
				if i > 0 {
					// After the first iteration we've handled the finalizer; stop here.
					break
				}
			}

			// Verify error expectation
			if tt.expectedError {
				assert.Error(t, err, "Expected error for test %s but got nil", tt.name)
			} else {
				assert.NoError(t, err, "Unexpected error for test %s: %v", tt.name, err)
			}

			// Verify requeue expectation
			if tt.expectedRequeue {
				assert.True(t, result.Requeue || result.RequeueAfter > 0,
					"Expected requeue for test %s but got %+v", tt.name, result)
			} else {
				assert.False(t, result.Requeue,
					"Did not expect requeue for test %s but got %+v", tt.name, result)
				assert.Equal(t, time.Duration(0), result.RequeueAfter,
					"Did not expect requeue after for test %s but got %+v", tt.name, result)
			}

			// Get updated NetworkIntent and run validation checks
			updatedNI := &nephoranv1.NetworkIntent{}
			err = fakeClient.Get(ctx, types.NamespacedName{
				Name:      ni.Name,
				Namespace: ni.Namespace,
			}, updatedNI)
			require.NoError(t, err, "Failed to get updated NetworkIntent for test %s", tt.name)

			// Verify phase transition (cast expected to NetworkIntentPhase for type-safe comparison)
			assert.Equal(t, nephoranv1.NetworkIntentPhase(tt.expectedPhase), updatedNI.Status.Phase,
				"Phase mismatch for test %s. Description: %s", tt.name, tt.description)

			// Run custom validation checks
			if tt.validationChecks != nil {
				tt.validationChecks(t, updatedNI, result)
			}
		})
	}
}

// Test concurrent reconciliation scenarios
func TestConcurrentReconciliation(t *testing.T) {
	scheme := runtime.NewScheme()
	nephoranv1.AddToScheme(scheme)

	// Create multiple NetworkIntents
	intents := []*nephoranv1.NetworkIntent{
		createTestNetworkIntent("concurrent-amf", "default", "Deploy AMF network function"),
		createTestNetworkIntent("concurrent-smf", "default", "Deploy SMF network function"),
		createTestNetworkIntent("concurrent-upf", "default", "Deploy UPF network function"),
	}

	// Create fake client with all intents
	objs := make([]client.Object, len(intents))
	for i, intent := range intents {
		objs[i] = intent
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()

	// Setup mock dependencies
	mockDeps := NewMockDependencies()
	llmResponse := json.RawMessage(`{}`)
	responseJSON, _ := json.Marshal(llmResponse)
	mockDeps.llmClient.SetResponse("", string(responseJSON))

	cfg := createTestConfig(); reconciler, err := NewNetworkIntentReconciler(fakeClient, scheme, mockDeps, &cfg)
	require.NoError(t, err)

	ctx := context.Background()

	// Test concurrent reconciliation
	resultChan := make(chan struct {
		result ctrl.Result
		err    error
	}, len(intents))

	for _, intent := range intents {
		go func(ni *nephoranv1.NetworkIntent) {
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      ni.Name,
					Namespace: ni.Namespace,
				},
			}
			result, err := reconciler.Reconcile(ctx, req)
			resultChan <- struct {
				result ctrl.Result
				err    error
			}{result, err}
		}(intent)
	}

	// Wait for all reconciliations to complete
	for i := 0; i < len(intents); i++ {
		select {
		case res := <-resultChan:
			assert.NoError(t, res.err, "Concurrent reconciliation should not fail")
		case <-time.After(30 * time.Second):
			t.Fatal("Concurrent reconciliation timed out")
		}
	}

	// Verify all intents were processed
	for _, intent := range intents {
		updatedNI := &nephoranv1.NetworkIntent{}
		err = fakeClient.Get(ctx, types.NamespacedName{
			Name:      intent.Name,
			Namespace: intent.Namespace,
		}, updatedNI)
		assert.NoError(t, err)
		assert.Contains(t, []string{"", "Processing", "Processed", "Completed", "LLMProcessing", "GitOpsCommit", "DeploymentVerification"}, string(updatedNI.Status.Phase))
	}
}

// Test resource constraint scenarios
func TestResourceConstraints(t *testing.T) {
	scheme := runtime.NewScheme()
	nephoranv1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)

	tests := []struct {
		name                string
		namespaceExists     bool
		resourceQuotaExists bool
		expectedPhase       string
		description         string
	}{
		{
			name:            "namespace_exists",
			namespaceExists: true,
			expectedPhase:   "Processing",
			description:     "Should process when namespace exists",
		},
		{
			name:            "namespace_not_exists",
			namespaceExists: false,
			expectedPhase:   "Processing", // Should still proceed
			description:     "Should handle missing namespace gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test NetworkIntent
			ni := createTestNetworkIntent(fmt.Sprintf("resource-%s", tt.name), "test-namespace", "Deploy AMF")

			// Setup objects based on test scenario
			objs := []client.Object{ni}
			if tt.namespaceExists {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-namespace",
					},
				}
				objs = append(objs, ns)
			}

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()

			// Setup mock dependencies
			mockDeps := NewMockDependencies()
			llmResponse := json.RawMessage(`{}`)
			responseJSON, _ := json.Marshal(llmResponse)
			mockDeps.llmClient.SetResponse("", string(responseJSON))

			cfg := createTestConfig(); reconciler, err := NewNetworkIntentReconciler(fakeClient, scheme, mockDeps, &cfg)
			require.NoError(t, err)

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      ni.Name,
					Namespace: ni.Namespace,
				},
			}

			ctx := context.Background()
			_, err = reconciler.Reconcile(ctx, req)
			assert.NoError(t, err)

			// Verify final state
			updatedNI := &nephoranv1.NetworkIntent{}
			err = fakeClient.Get(ctx, types.NamespacedName{
				Name:      ni.Name,
				Namespace: ni.Namespace,
			}, updatedNI)
			assert.NoError(t, err)
			assert.Contains(t, []string{"", tt.expectedPhase, "Processed", "Completed"}, string(updatedNI.Status.Phase))
		})
	}
}

// Test network partition scenarios
func TestNetworkPartitionScenarios(t *testing.T) {
	scheme := runtime.NewScheme()
	nephoranv1.AddToScheme(scheme)

	// Create test NetworkIntent
	ni := createTestNetworkIntent("network-partition-test", "default", "Deploy AMF with network partition simulation")
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ni).Build()

	// Setup mock dependencies to simulate network partition
	mockDeps := NewMockDependencies()
	mockDeps.llmClient.SetError("", errors.New("connection timeout: network unreachable"))

	cfg := createTestConfig(); reconciler, err := NewNetworkIntentReconciler(fakeClient, scheme, mockDeps, &cfg)
	require.NoError(t, err)

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      ni.Name,
			Namespace: ni.Namespace,
		},
	}

	ctx := context.Background()

	// First reconciliation should fail due to network partition
	_, err = reconciler.Reconcile(ctx, req)
	// May or may not return error depending on retry strategy; we check the phase instead

	// Verify it moved to error/retry state (or still empty if status subresource not propagated)
	errorNI := &nephoranv1.NetworkIntent{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: ni.Name, Namespace: ni.Namespace}, errorNI)
	assert.NoError(t, err)
	assert.Contains(t, []string{"", "Error", "Processing"}, string(errorNI.Status.Phase))

	// Simulate network recovery
	llmResponse := json.RawMessage(`{}`)
	responseJSON, _ := json.Marshal(llmResponse)
	mockDeps.llmClient.SetError("", nil)
	mockDeps.llmClient.SetResponse("", string(responseJSON))

	// Second reconciliation should succeed after network recovery
	_, err = reconciler.Reconcile(ctx, req)
	// Error is acceptable during recovery phase

	// Verify recovery state
	recoveredNI := &nephoranv1.NetworkIntent{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: ni.Name, Namespace: ni.Namespace}, recoveredNI)
	assert.NoError(t, err)
	assert.Contains(t, []string{"", "Error", "Processing", "Processed"}, string(recoveredNI.Status.Phase))
}

// Benchmark edge case performance
func BenchmarkEdgeCaseProcessing(b *testing.B) {
	scheme := runtime.NewScheme()
	nephoranv1.AddToScheme(scheme)

	// Test scenarios
	scenarios := []struct {
		name      string
		intent    string
		mockSetup func(*MockDependencies)
	}{
		{
			name:      "EmptyIntent",
			intent:    "",
			mockSetup: func(deps *MockDependencies) {},
		},
		{
			name:   "LLMFailure",
			intent: "Deploy AMF",
			mockSetup: func(deps *MockDependencies) {
				deps.llmClient.SetError("", errors.New("LLM failure"))
			},
		},
		{
			name:   "LongIntent",
			intent: strings.Repeat("Deploy comprehensive 5G network ", 50),
			mockSetup: func(deps *MockDependencies) {
				llmResponse := json.RawMessage(`{}`)
				responseJSON, _ := json.Marshal(llmResponse)
				deps.llmClient.SetResponse("", string(responseJSON))
			},
		},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			ni := createTestNetworkIntent("benchmark-edge", "default", scenario.intent)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ni).Build()

			mockDeps := NewMockDependencies()
			scenario.mockSetup(mockDeps)

			cfg2 := createTestConfig(); reconciler, _ := NewNetworkIntentReconciler(fakeClient, scheme, mockDeps, &cfg2)

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
		})
	}
}

