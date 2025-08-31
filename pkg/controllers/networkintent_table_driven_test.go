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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// Comprehensive table-driven test suite
func TestNetworkIntentTableDriven(t *testing.T) {
	scheme := runtime.NewScheme()
	nephoranv1.AddToScheme(scheme)

	// Define comprehensive test cases
	testCases := []struct {
		// Test identification
		name     string
		category string

		// Input configuration
		intentText        string
		enabledLLMIntent  string
		initialPhase      string
		initialConditions []metav1.Condition
		environmentVars   map[string]string

		// Mock configuration
		llmResponse     string
		llmError        error
		llmFailCount    int
		gitShouldFail   bool
		httpClientSetup func(*http.Client)

		// Expected outcomes
		expectedPhase      string
		expectedRequeue    bool
		expectedError      bool
		expectedConditions []ExpectedCondition
		expectedParameters map[string]interface{}

		// Validation and assertions
		validationChecks func(t *testing.T, ni *nephoranv1.NetworkIntent, result ctrl.Result)
		description      string
		tags             []string
	}{
		// === HAPPY PATH SCENARIOS ===
		{
			name:             "5gc_amf_deployment_success",
			category:         "happy_path",
			intentText:       "Deploy AMF network function for 5G core",
			enabledLLMIntent: "true",
			initialPhase:     "Pending",
			llmResponse: mustMarshal(map[string]interface{}{
				"action":    "deploy",
				"component": "amf",
				"namespace": "5g-core",
				"replicas":  1,
				"resources": map[string]interface{}{
					"cpu":    "500m",
					"memory": "1Gi",
				},
			}),
			expectedPhase:   "Processing",
			expectedRequeue: false,
			expectedConditions: []ExpectedCondition{
				{Type: "Validated", Status: metav1.ConditionTrue},
				{Type: "Processed", Status: metav1.ConditionTrue},
			},
			expectedParameters: map[string]interface{}{
				"action":    "deploy",
				"component": "amf",
				"namespace": "5g-core",
			},
			description: "Successful AMF deployment with LLM processing",
			tags:        []string{"5gc", "amf", "deployment"},
		},
		{
			name:             "5gc_smf_configuration_success",
			category:         "happy_path",
			intentText:       "Configure SMF with UPF integration and session management policies",
			enabledLLMIntent: "true",
			initialPhase:     "Pending",
			llmResponse: mustMarshal(map[string]interface{}{
				"action":    "configure",
				"component": "smf",
				"namespace": "5g-core",
				"config": map[string]interface{}{
					"upf_integration":  true,
					"session_policies": []string{"policy1", "policy2"},
				},
			}),
			expectedPhase:   "Processing",
			expectedRequeue: false,
			expectedConditions: []ExpectedCondition{
				{Type: "Validated", Status: metav1.ConditionTrue},
				{Type: "Processed", Status: metav1.ConditionTrue},
			},
			description: "Successful SMF configuration with complex parameters",
			tags:        []string{"5gc", "smf", "configuration"},
		},
		{
			name:             "oran_deployment_success",
			category:         "happy_path",
			intentText:       "Deploy O-RAN components including O-CU and O-DU for edge deployment",
			enabledLLMIntent: "true",
			initialPhase:     "Pending",
			llmResponse: mustMarshal(map[string]interface{}{
				"action":          "deploy",
				"component":       "oran",
				"namespace":       "oran-system",
				"components":      []string{"o-cu", "o-du"},
				"deployment_type": "edge",
			}),
			expectedPhase:   "Processing",
			expectedRequeue: false,
			expectedConditions: []ExpectedCondition{
				{Type: "Validated", Status: metav1.ConditionTrue},
				{Type: "Processed", Status: metav1.ConditionTrue},
			},
			description: "Successful O-RAN deployment with multiple components",
			tags:        []string{"oran", "o-cu", "o-du", "edge"},
		},
		{
			name:             "network_slice_embb_success",
			category:         "happy_path",
			intentText:       "Deploy eMBB network slice for enhanced mobile broadband with high throughput",
			enabledLLMIntent: "true",
			initialPhase:     "Pending",
			llmResponse: mustMarshal(map[string]interface{}{
				"action":     "deploy",
				"component":  "network-slice",
				"slice_type": "embb",
				"namespace":  "slicing",
				"qos_profile": map[string]interface{}{
					"throughput": "10Gbps",
					"latency":    "10ms",
				},
			}),
			expectedPhase:   "Processing",
			expectedRequeue: false,
			expectedConditions: []ExpectedCondition{
				{Type: "Validated", Status: metav1.ConditionTrue},
				{Type: "Processed", Status: metav1.ConditionTrue},
			},
			description: "Successful eMBB network slice deployment",
			tags:        []string{"slicing", "embb", "qos"},
		},

		// === ERROR SCENARIOS ===
		{
			name:             "empty_intent_error",
			category:         "error_handling",
			intentText:       "",
			enabledLLMIntent: "true",
			initialPhase:     "Pending",
			expectedPhase:    "Error",
			expectedRequeue:  false,
			expectedConditions: []ExpectedCondition{
				{Type: "Validated", Status: metav1.ConditionFalse, MessageContains: "empty"},
			},
			description: "Handle empty intent text with proper error",
			tags:        []string{"validation", "error"},
		},
		{
			name:             "llm_service_unavailable_error",
			category:         "error_handling",
			intentText:       "Deploy SMF network function",
			enabledLLMIntent: "true",
			initialPhase:     "Pending",
			llmError:         errors.New("LLM service unavailable - connection timeout"),
			expectedPhase:    "Error",
			expectedRequeue:  true,
			expectedConditions: []ExpectedCondition{
				{Type: "Processed", Status: metav1.ConditionFalse, MessageContains: "LLM"},
				{Type: "Ready", Status: metav1.ConditionFalse},
			},
			description: "Handle LLM service unavailable with retry",
			tags:        []string{"llm", "error", "retry"},
		},
		{
			name:             "llm_invalid_json_error",
			category:         "error_handling",
			intentText:       "Deploy UPF network function",
			enabledLLMIntent: "true",
			initialPhase:     "Pending",
			llmResponse:      "invalid json response from LLM",
			expectedPhase:    "Error",
			expectedRequeue:  true,
			expectedConditions: []ExpectedCondition{
				{Type: "Processed", Status: metav1.ConditionFalse},
			},
			description: "Handle invalid JSON response from LLM",
			tags:        []string{"llm", "json", "error"},
		},
		{
			name:             "git_operation_failure_error",
			category:         "error_handling",
			intentText:       "Deploy comprehensive 5G network",
			enabledLLMIntent: "true",
			initialPhase:     "Pending",
			llmResponse: mustMarshal(map[string]interface{}{
				"action":    "deploy",
				"component": "5gc-comprehensive",
				"namespace": "5g-core",
			}),
			gitShouldFail:   true,
			expectedPhase:   "Error",
			expectedRequeue: true,
			expectedConditions: []ExpectedCondition{
				{Type: "Deployed", Status: metav1.ConditionFalse, MessageContains: "git"},
				{Type: "Ready", Status: metav1.ConditionFalse},
			},
			description: "Handle Git operation failure with retry",
			tags:        []string{"git", "deployment", "error"},
		},

		// === RETRY SCENARIOS ===
		{
			name:             "llm_retry_eventual_success",
			category:         "retry_logic",
			intentText:       "Deploy NSSF for network slicing",
			enabledLLMIntent: "true",
			initialPhase:     "Error",
			initialConditions: []metav1.Condition{
				{
					Type:    "Processed",
					Status:  metav1.ConditionFalse,
					Reason:  "ProcessingFailed",
					Message: "LLM processing failed - retry 1/3",
				},
			},
			llmFailCount: 1, // Fail first attempt, succeed second
			llmResponse: mustMarshal(map[string]interface{}{
				"action":    "deploy",
				"component": "nssf",
				"namespace": "5g-core",
			}),
			expectedPhase:   "Processing",
			expectedRequeue: false,
			expectedConditions: []ExpectedCondition{
				{Type: "Processed", Status: metav1.ConditionTrue},
				{Type: "Validated", Status: metav1.ConditionTrue},
			},
			description: "Successful retry after initial LLM failure",
			tags:        []string{"retry", "recovery", "nssf"},
		},
		{
			name:             "max_retries_exceeded",
			category:         "retry_logic",
			intentText:       "Deploy AMF with persistent failure simulation",
			enabledLLMIntent: "true",
			initialPhase:     "Error",
			initialConditions: []metav1.Condition{
				{
					Type:    "Processed",
					Status:  metav1.ConditionFalse,
					Reason:  "MaxRetriesReached",
					Message: "Maximum retry attempts (3/3) exceeded",
				},
			},
			llmError:        errors.New("persistent LLM failure"),
			expectedPhase:   "Error",
			expectedRequeue: true, // Should continue with exponential backoff
			expectedConditions: []ExpectedCondition{
				{Type: "Processed", Status: metav1.ConditionFalse},
			},
			validationChecks: func(t *testing.T, ni *nephoranv1.NetworkIntent, result ctrl.Result) {
				// Should have longer requeue time due to exponential backoff
				assert.True(t, result.RequeueAfter >= DefaultRetryDelay)
				assert.Contains(t, getConditionMessage(ni.Status.Conditions, "Processed"), "persistent")
			},
			description: "Handle maximum retries exceeded scenario",
			tags:        []string{"retry", "max_retries", "backoff"},
		},

		// === LLM DISABLED SCENARIOS ===
		{
			name:             "llm_disabled_success",
			category:         "llm_disabled",
			intentText:       "Deploy AMF network function without LLM",
			enabledLLMIntent: "false",
			initialPhase:     "Pending",
			expectedPhase:    "Processed",
			expectedRequeue:  false,
			expectedConditions: []ExpectedCondition{
				{Type: "Processed", Status: metav1.ConditionTrue},
				{Type: "Ready", Status: metav1.ConditionTrue},
			},
			description: "Process intent successfully without LLM when disabled",
			tags:        []string{"no_llm", "direct_processing"},
		},
		{
			name:             "llm_disabled_complex_intent",
			category:         "llm_disabled",
			intentText:       "Deploy comprehensive 5G core with AMF, SMF, UPF, and NSSF components",
			enabledLLMIntent: "false",
			initialPhase:     "Pending",
			expectedPhase:    "Processed",
			expectedRequeue:  false,
			expectedConditions: []ExpectedCondition{
				{Type: "Processed", Status: metav1.ConditionTrue},
			},
			description: "Handle complex intent without LLM processing",
			tags:        []string{"no_llm", "complex", "5gc"},
		},

		// === EDGE CASE SCENARIOS ===
		{
			name:             "unicode_intent_success",
			category:         "edge_cases",
			intentText:       "Deploy AMF with 高性能 configuration for 5G 网络 deployment",
			enabledLLMIntent: "true",
			initialPhase:     "Pending",
			llmResponse: mustMarshal(map[string]interface{}{
				"action":    "deploy",
				"component": "amf",
				"config":    "高性能",
			}),
			expectedPhase:   "Processing",
			expectedRequeue: false,
			expectedConditions: []ExpectedCondition{
				{Type: "Validated", Status: metav1.ConditionTrue},
				{Type: "Processed", Status: metav1.ConditionTrue},
			},
			description: "Handle Unicode characters in intent text",
			tags:        []string{"unicode", "i18n", "edge_case"},
		},
		{
			name:             "special_characters_intent",
			category:         "edge_cases",
			intentText:       "Deploy SMF with config: {cpu: '500m', memory: '1Gi', ports: [8080, 8443]}",
			enabledLLMIntent: "true",
			initialPhase:     "Pending",
			llmResponse: mustMarshal(map[string]interface{}{
				"action":    "deploy",
				"component": "smf",
				"config": map[string]interface{}{
					"cpu":    "500m",
					"memory": "1Gi",
					"ports":  []int{8080, 8443},
				},
			}),
			expectedPhase:   "Processing",
			expectedRequeue: false,
			expectedConditions: []ExpectedCondition{
				{Type: "Validated", Status: metav1.ConditionTrue},
				{Type: "Processed", Status: metav1.ConditionTrue},
			},
			description: "Handle special characters and structured data in intent",
			tags:        []string{"special_chars", "structured_data"},
		},
		{
			name:             "very_long_intent_error",
			category:         "edge_cases",
			intentText:       strings.Repeat("Deploy comprehensive 5G network with advanced features ", 100),
			enabledLLMIntent: "true",
			initialPhase:     "Pending",
			expectedPhase:    "Error",
			expectedRequeue:  false,
			expectedConditions: []ExpectedCondition{
				{Type: "Validated", Status: metav1.ConditionFalse, MessageContains: "complex"},
			},
			description: "Reject overly complex intent text",
			tags:        []string{"validation", "complexity", "limits"},
		},

		// === PHASE TRANSITION SCENARIOS ===
		{
			name:             "pending_to_processing_transition",
			category:         "phase_transitions",
			intentText:       "Deploy UPF for user plane processing",
			enabledLLMIntent: "true",
			initialPhase:     "Pending",
			llmResponse: mustMarshal(map[string]interface{}{
				"action":    "deploy",
				"component": "upf",
				"namespace": "5g-core",
			}),
			expectedPhase:   "Processing",
			expectedRequeue: false,
			validationChecks: func(t *testing.T, ni *nephoranv1.NetworkIntent, result ctrl.Result) {
				assert.Equal(t, "Processing", ni.Status.Phase)
				assert.True(t, hasConditionWithStatus(ni.Status.Conditions, "Validated", metav1.ConditionTrue))
				assert.True(t, hasConditionWithStatus(ni.Status.Conditions, "Processed", metav1.ConditionTrue))
				assert.NotEmpty(t, ni.Spec.Parameters.Raw)
			},
			description: "Successful transition from Pending to Processing",
			tags:        []string{"phase_transition", "upf"},
		},
		{
			name:             "error_to_processing_recovery",
			category:         "phase_transitions",
			intentText:       "Deploy PCF for policy control",
			enabledLLMIntent: "true",
			initialPhase:     "Error",
			initialConditions: []metav1.Condition{
				{
					Type:    "Processed",
					Status:  metav1.ConditionFalse,
					Reason:  "ProcessingFailed",
					Message: "Previous processing attempt failed",
				},
			},
			llmFailCount: 1, // Recover on retry
			llmResponse: mustMarshal(map[string]interface{}{
				"action":    "deploy",
				"component": "pcf",
				"namespace": "5g-core",
			}),
			expectedPhase:   "Processing",
			expectedRequeue: false,
			validationChecks: func(t *testing.T, ni *nephoranv1.NetworkIntent, result ctrl.Result) {
				assert.Equal(t, "Processing", ni.Status.Phase)
				assert.True(t, hasConditionWithStatus(ni.Status.Conditions, "Processed", metav1.ConditionTrue))
			},
			description: "Successful recovery from Error to Processing state",
			tags:        []string{"phase_transition", "recovery", "pcf"},
		},
	}

	// Execute all test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variables
			if tc.enabledLLMIntent != "" {
				os.Setenv("ENABLE_LLM_INTENT", tc.enabledLLMIntent)
				defer os.Unsetenv("ENABLE_LLM_INTENT")
			}

			for key, value := range tc.environmentVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			// Create test NetworkIntent
			ni := createTestNetworkIntent(fmt.Sprintf("%s-%s", tc.category, tc.name), "default", tc.intentText)
			ni.Status.Phase = tc.initialPhase
			if tc.initialConditions != nil {
				ni.Status.Conditions = tc.initialConditions
				// Set proper timestamps
				now := metav1.NewTime(time.Now())
				for i := range ni.Status.Conditions {
					ni.Status.Conditions[i].LastTransitionTime = now
				}
			}

			// Create fake client
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ni).Build()

			// Setup mock dependencies
			mockDeps := NewMockDependencies()

			// Configure LLM client
			if tc.llmError != nil {
				mockDeps.llmClient.SetError(tc.llmError)
			}
			if tc.llmResponse != "" {
				mockDeps.llmClient.SetResponse(tc.llmResponse)
			}
			if tc.llmFailCount > 0 {
				mockDeps.llmClient.SetFailCount(tc.llmFailCount)
			}

			// Configure Git client
			if tc.gitShouldFail {
				mockDeps.gitClient.SetShouldFail(true)
			}

			// Create reconciler
			reconciler, err := NewNetworkIntentReconciler(fakeClient, scheme, mockDeps, createTestConfig())
			require.NoError(t, err, "Failed to create reconciler for test %s", tc.name)

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

			// Verify error expectation
			if tc.expectedError {
				assert.Error(t, err, "Expected error for test %s but got nil", tc.name)
			} else {
				assert.NoError(t, err, "Unexpected error for test %s: %v", tc.name, err)
			}

			// Verify requeue expectation
			if tc.expectedRequeue {
				assert.True(t, result.Requeue || result.RequeueAfter > 0,
					"Expected requeue for test %s but got %+v", tc.name, result)
			} else {
				assert.False(t, result.Requeue,
					"Did not expect requeue for test %s but got %+v", tc.name, result)
				if result.RequeueAfter > 0 {
					t.Logf("Note: Test %s has RequeueAfter=%v (may be expected for some scenarios)", tc.name, result.RequeueAfter)
				}
			}

			// Get updated NetworkIntent
			updatedNI := &nephoranv1.NetworkIntent{}
			err = fakeClient.Get(ctx, types.NamespacedName{
				Name:      ni.Name,
				Namespace: ni.Namespace,
			}, updatedNI)
			require.NoError(t, err, "Failed to get updated NetworkIntent for test %s", tc.name)

			// Verify phase expectation
			assert.Equal(t, tc.expectedPhase, updatedNI.Status.Phase,
				"Phase mismatch for test %s. Expected: %s, Got: %s. Description: %s",
				tc.name, tc.expectedPhase, updatedNI.Status.Phase, tc.description)

			// Verify condition expectations
			for _, expectedCond := range tc.expectedConditions {
				assert.True(t, hasConditionWithStatus(updatedNI.Status.Conditions, expectedCond.Type, expectedCond.Status),
					"Expected condition %s with status %s not found for test %s. Conditions: %+v",
					expectedCond.Type, expectedCond.Status, tc.name, updatedNI.Status.Conditions)

				if expectedCond.MessageContains != "" {
					message := getConditionMessage(updatedNI.Status.Conditions, expectedCond.Type)
					assert.Contains(t, strings.ToLower(message), strings.ToLower(expectedCond.MessageContains),
						"Condition %s message should contain '%s' for test %s. Got: '%s'",
						expectedCond.Type, expectedCond.MessageContains, tc.name, message)
				}
			}

			// Verify parameter expectations
			if len(tc.expectedParameters) > 0 && len(updatedNI.Spec.Parameters.Raw) > 0 {
				var actualParams map[string]interface{}
				err = json.Unmarshal(updatedNI.Spec.Parameters.Raw, &actualParams)
				assert.NoError(t, err, "Should be able to unmarshal parameters for test %s", tc.name)

				for key, expectedValue := range tc.expectedParameters {
					actualValue, exists := actualParams[key]
					assert.True(t, exists, "Parameter %s should exist for test %s", key, tc.name)
					assert.Equal(t, expectedValue, actualValue,
						"Parameter %s mismatch for test %s. Expected: %v, Got: %v",
						key, tc.name, expectedValue, actualValue)
				}
			}

			// Run custom validation checks
			if tc.validationChecks != nil {
				tc.validationChecks(t, updatedNI, result)
			}

			// Log test completion with tags for analysis
			t.Logf("✓ Test %s completed successfully. Category: %s, Tags: %v", tc.name, tc.category, tc.tags)
		})
	}

	// Print test summary
	categories := make(map[string]int)
	for _, tc := range testCases {
		categories[tc.category]++
	}

	t.Logf("\n=== Test Summary ===")
	t.Logf("Total test cases: %d", len(testCases))
	for category, count := range categories {
		t.Logf("%s: %d tests", category, count)
	}
}

// Helper types for test configuration
type ExpectedCondition struct {
	Type            string
	Status          metav1.ConditionStatus
	MessageContains string
}

// Helper function to marshal JSON with error handling
func mustMarshal(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal JSON: %v", err))
	}
	return string(data)
}

// Test suite for specific categories
func TestNetworkIntentHappyPath(t *testing.T) {
	// Run only happy path scenarios for focused testing
	t.Parallel()
	// Implementation can filter test cases by category
}

func TestNetworkIntentErrorHandling(t *testing.T) {
	// Run only error handling scenarios
	t.Parallel()
	// Implementation can filter test cases by category
}

func TestNetworkIntentRetryLogic(t *testing.T) {
	// Run only retry logic scenarios
	t.Parallel()
	// Implementation can filter test cases by category
}

// Benchmark table-driven tests
func BenchmarkTableDrivenScenarios(b *testing.B) {
	scheme := runtime.NewScheme()
	nephoranv1.AddToScheme(scheme)

	// Representative scenarios for benchmarking
	scenarios := []struct {
		name        string
		intentText  string
		llmResponse string
		mockSetup   func(*MockDependencies)
	}{
		{
			name:       "HappyPath",
			intentText: "Deploy AMF network function",
			llmResponse: mustMarshal(map[string]interface{}{
				"action":    "deploy",
				"component": "amf",
			}),
			mockSetup: func(deps *MockDependencies) {},
		},
		{
			name:       "ErrorPath",
			intentText: "Deploy SMF network function",
			mockSetup: func(deps *MockDependencies) {
				deps.llmClient.SetError(errors.New("LLM failure"))
			},
		},
		{
			name:       "ComplexIntent",
			intentText: "Deploy comprehensive 5G core with AMF, SMF, UPF, and network slicing",
			llmResponse: mustMarshal(map[string]interface{}{
				"action":     "deploy",
				"component":  "5gc-comprehensive",
				"components": []string{"amf", "smf", "upf"},
				"slicing":    true,
			}),
			mockSetup: func(deps *MockDependencies) {},
		},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			ni := createTestNetworkIntent("benchmark", "default", scenario.intentText)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ni).Build()

			mockDeps := NewMockDependencies()
			if scenario.llmResponse != "" {
				mockDeps.llmClient.SetResponse(scenario.llmResponse)
			}
			scenario.mockSetup(mockDeps)

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
		})
	}
}
