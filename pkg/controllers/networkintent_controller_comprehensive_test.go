package controllers

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/git"
	gitfake "github.com/thc1006/nephoran-intent-operator/pkg/git/fake"
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
	"github.com/thc1006/nephoran-intent-operator/pkg/telecom"
)

const (
	// NetworkIntentFinalizer is the finalizer used by NetworkIntent resources
	NetworkIntentFinalizer = "networkintent.nephoran.io/finalizer"
)

// fakeLLMClient implements shared.ClientInterface for testing
type fakeLLMClient struct {
	shouldFail              bool
	shouldReturnEmpty       bool
	shouldReturnInvalidJSON bool
	response                string
	callCount               int
}

func (f *fakeLLMClient) ProcessIntent(ctx context.Context, intent string) (string, error) {
	f.callCount++
	if f.shouldFail {
		return "", fmt.Errorf("fake LLM processing failure")
	}
	if f.shouldReturnEmpty {
		return "", nil
	}
	if f.shouldReturnInvalidJSON {
		return "invalid json {", nil
	}
	if f.response != "" {
		return f.response, nil
	}
	// Default successful response
	return `{"deployment_type": "5g", "latency_requirement": "1ms", "bandwidth": "1Gbps"}`, nil
}

func (f *fakeLLMClient) Reset() {
	f.shouldFail = false
	f.shouldReturnEmpty = false
	f.shouldReturnInvalidJSON = false
	f.response = ""
	f.callCount = 0
}

// Implement remaining methods from shared.ClientInterface
func (f *fakeLLMClient) ProcessRequest(ctx context.Context, request *shared.LLMRequest) (*shared.LLMResponse, error) {
	f.callCount++
	if f.shouldFail {
		return nil, fmt.Errorf("fake LLM request processing failure")
	}
	return &shared.LLMResponse{
		Content: f.response,
	}, nil
}

func (f *fakeLLMClient) ProcessStreamingRequest(ctx context.Context, request *shared.LLMRequest) (<-chan *shared.StreamingChunk, error) {
	ch := make(chan *shared.StreamingChunk, 1)
	close(ch)
	return ch, nil
}

func (f *fakeLLMClient) HealthCheck(ctx context.Context) error {
	if f.shouldFail {
		return fmt.Errorf("fake health check failure")
	}
	return nil
}

func (f *fakeLLMClient) GetStatus() shared.ClientStatus {
	return shared.ClientStatusHealthy
}

func (f *fakeLLMClient) GetModelCapabilities() shared.ModelCapabilities {
	return shared.ModelCapabilities{}
}

func (f *fakeLLMClient) GetEndpoint() string {
	return "http://fake-llm-endpoint"
}

func (f *fakeLLMClient) Close() error {
	return nil
}

// SetError configures the fake client to return an error
func (f *fakeLLMClient) SetError(shouldFail bool) {
	f.shouldFail = shouldFail
}

// SetResponse configures the fake client to return a specific response
func (f *fakeLLMClient) SetResponse(response string) {
	f.response = response
}

// fakePackageGenerator implements nephio.PackageGenerator for testing
type fakePackageGenerator struct {
	shouldFail bool
	callCount  int
}

func (f *fakePackageGenerator) GeneratePackage(intent *nephoranv1.NetworkIntent) (map[string]string, error) {
	f.callCount++
	if f.shouldFail {
		return nil, fmt.Errorf("fake package generation failure")
	}
	return map[string]string{
		"deployment.yaml": "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test-config",
	}, nil
}

func (f *fakePackageGenerator) Reset() {
	f.shouldFail = false
	f.callCount = 0
}

// fakeHTTPClient implements http.Client behavior for testing
type fakeHTTPClient struct {
	shouldFail bool
	callCount  int
}

func (f *fakeHTTPClient) Do(req *http.Request) (*http.Response, error) {
	f.callCount++
	if f.shouldFail {
		return nil, fmt.Errorf("fake HTTP client failure")
	}
	return &http.Response{StatusCode: http.StatusOK}, nil
}

func (f *fakeHTTPClient) Reset() {
	f.shouldFail = false
	f.callCount = 0
}

// fakeDependencies implements Dependencies interface for testing
type fakeDependencies struct {
	gitClient        *gitfake.Client
	llmClient        *fakeLLMClient
	packageGenerator *fakePackageGenerator
	httpClient       *fakeHTTPClient
	eventRecorder    record.EventRecorder
}

func newFakeDependencies() *fakeDependencies {
	return &fakeDependencies{
		gitClient:        gitfake.NewClient(),
		llmClient:        &fakeLLMClient{},
		packageGenerator: &fakePackageGenerator{},
		httpClient:       &fakeHTTPClient{},
		eventRecorder:    &record.FakeRecorder{},
	}
}

func (f *fakeDependencies) GetGitClient() git.ClientInterface {
	return f.gitClient
}

func (f *fakeDependencies) GetLLMClient() shared.ClientInterface {
	return f.llmClient
}

func (f *fakeDependencies) GetPackageGenerator() *nephio.PackageGenerator {
	// Return nil since we're using our fake directly in the controller
	return nil
}

func (f *fakeDependencies) GetHTTPClient() *http.Client {
	// Can't return our fake directly due to interface mismatch, return nil for now
	return nil
}

func (f *fakeDependencies) GetEventRecorder() record.EventRecorder {
	return f.eventRecorder
}

func (f *fakeDependencies) GetMetricsCollector() monitoring.MetricsCollector {
	return nil // Return nil for testing purposes
}

func (f *fakeDependencies) GetTelecomKnowledgeBase() *telecom.TelecomKnowledgeBase {
	return nil // Return nil for testing purposes
}

func (f *fakeDependencies) Reset() {
	f.gitClient.Reset()
	f.llmClient.Reset()
	f.packageGenerator.Reset()
	f.httpClient.Reset()
}

func TestNetworkIntentController_Reconcile(t *testing.T) {
	testCases := []struct {
		name            string
		networkIntent   *nephoranv1.NetworkIntent
		existingObjects []client.Object
		depsSetup       func(*fakeDependencies)
		config          *Config
		expectedResult  ctrl.Result
		expectedError   bool
		expectedCalls   func(*testing.T, *fakeDependencies)
		expectedStatus  func(*testing.T, *nephoranv1.NetworkIntent)
		description     string
	}{
		{
			name: "successful_end_to_end_processing",
			networkIntent: &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-intent",
					Namespace: "default",
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy a 5G network with ultra-low latency requirements",
				},
			},
			existingObjects: []client.Object{},
			depsSetup: func(deps *fakeDependencies) {
				// LLM should succeed with valid JSON
				deps.llmClient.response = `{"deployment_type": "5g", "latency_requirement": "1ms"}`
				// Git operations should succeed
				deps.gitClient.CommitHash = "abc123def456"
			},
			config: &Config{
				MaxRetries:      3,
				RetryDelay:      10 * time.Second,
				Timeout:         5 * time.Minute,
				GitRepoURL:      "https://git.example.com/repo",
				GitBranch:       "main",
				GitDeployPath:   "networkintents",
				LLMProcessorURL: "http://llm.example.com",
				UseNephioPorch:  false,
			},
			expectedResult: ctrl.Result{},
			expectedError:  false,
			expectedCalls: func(t *testing.T, deps *fakeDependencies) {
				assert.Equal(t, 1, deps.llmClient.callCount, "LLM should be called once")
				assert.Contains(t, deps.gitClient.CallHistory, "InitRepo", "Git InitRepo should be called")
				assert.True(t, containsCallPattern(deps.gitClient.CallHistory, "CommitAndPush"), "Git CommitAndPush should be called")
			},
			expectedStatus: func(t *testing.T, intent *nephoranv1.NetworkIntent) {
				assert.Equal(t, "Completed", intent.Status.Phase, "Phase should be Completed")
				assert.True(t, isConditionTrue(intent.Status.Conditions, "Processed"), "Processed condition should be true")
				assert.True(t, isConditionTrue(intent.Status.Conditions, "Deployed"), "Deployed condition should be true")
				assert.NotEmpty(t, intent.Status.GitCommitHash, "GitCommitHash should be set")
			},
			description: "Should successfully process intent and deploy via GitOps",
		},
		{
			name: "llm_processing_failure_with_retry",
			networkIntent: &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-intent",
					Namespace: "default",
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy a test network",
				},
			},
			existingObjects: []client.Object{},
			depsSetup: func(deps *fakeDependencies) {
				deps.llmClient.shouldFail = true
			},
			config: &Config{
				MaxRetries: 3,
				RetryDelay: 1 * time.Second,
				Timeout:    5 * time.Minute,
			},
			expectedResult: ctrl.Result{RequeueAfter: 1 * time.Second},
			expectedError:  false,
			expectedCalls: func(t *testing.T, deps *fakeDependencies) {
				assert.Equal(t, 1, deps.llmClient.callCount, "LLM should be called once")
				assert.Empty(t, deps.gitClient.CallHistory, "Git should not be called on LLM failure")
			},
			expectedStatus: func(t *testing.T, intent *nephoranv1.NetworkIntent) {
				assert.NotEqual(t, "Completed", intent.Status.Phase, "Phase should not be Completed")
				assert.False(t, isConditionTrue(intent.Status.Conditions, "Processed"), "Processed condition should be false")
			},
			description: "Should handle LLM processing failure with retry",
		},
		{
			name: "empty_intent_validation_failure",
			networkIntent: &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-intent",
					Namespace: "default",
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "", // Empty intent
				},
			},
			existingObjects: []client.Object{},
			depsSetup:       func(deps *fakeDependencies) {},
			config: &Config{
				MaxRetries: 3,
				RetryDelay: 1 * time.Second,
				Timeout:    5 * time.Minute,
			},
			expectedResult: ctrl.Result{},
			expectedError:  true,
			expectedCalls: func(t *testing.T, deps *fakeDependencies) {
				assert.Equal(t, 0, deps.llmClient.callCount, "LLM should not be called for empty intent")
				assert.Empty(t, deps.gitClient.CallHistory, "Git should not be called for invalid intent")
			},
			expectedStatus: func(t *testing.T, intent *nephoranv1.NetworkIntent) {
				assert.False(t, isConditionTrue(intent.Status.Conditions, "Processed"), "Processed condition should be false")
			},
			description: "Should fail validation for empty intent",
		},
		{
			name: "llm_returns_invalid_json",
			networkIntent: &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-intent",
					Namespace: "default",
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy something",
				},
			},
			existingObjects: []client.Object{},
			depsSetup: func(deps *fakeDependencies) {
				deps.llmClient.shouldReturnInvalidJSON = true
			},
			config: &Config{
				MaxRetries: 3,
				RetryDelay: 1 * time.Second,
				Timeout:    5 * time.Minute,
			},
			expectedResult: ctrl.Result{RequeueAfter: 1 * time.Second},
			expectedError:  false,
			expectedCalls: func(t *testing.T, deps *fakeDependencies) {
				assert.Equal(t, 1, deps.llmClient.callCount, "LLM should be called once")
				assert.Empty(t, deps.gitClient.CallHistory, "Git should not be called on JSON parse failure")
			},
			expectedStatus: func(t *testing.T, intent *nephoranv1.NetworkIntent) {
				assert.False(t, isConditionTrue(intent.Status.Conditions, "Processed"), "Processed condition should be false")
			},
			description: "Should handle invalid JSON response from LLM with retry",
		},
		{
			name: "git_deployment_failure_with_retry",
			networkIntent: &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-intent",
					Namespace:  "default",
					Finalizers: []string{NetworkIntentFinalizer},
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent:     "Deploy a test network",
					Parameters: &runtime.RawExtension{Raw: []byte(`{"deployment_type": "test"}`)},
				},
				Status: nephoranv1.NetworkIntentStatus{
					Conditions: []metav1.Condition{
						{
							Type:   "Processed",
							Status: metav1.ConditionTrue,
							Reason: "LLMProcessingSucceeded",
						},
					},
				},
			},
			existingObjects: []client.Object{},
			depsSetup: func(deps *fakeDependencies) {
				deps.gitClient.ShouldFailCommitAndPush = true
			},
			config: &Config{
				MaxRetries:    3,
				RetryDelay:    1 * time.Second,
				Timeout:       5 * time.Minute,
				GitRepoURL:    "https://git.example.com/repo",
				GitDeployPath: "networkintents",
			},
			expectedResult: ctrl.Result{RequeueAfter: 1 * time.Second},
			expectedError:  false,
			expectedCalls: func(t *testing.T, deps *fakeDependencies) {
				assert.Equal(t, 0, deps.llmClient.callCount, "LLM should not be called for already processed intent")
				assert.Contains(t, deps.gitClient.CallHistory, "InitRepo", "Git InitRepo should be called")
				assert.True(t, containsCallPattern(deps.gitClient.CallHistory, "CommitAndPush"), "Git CommitAndPush should be attempted")
			},
			expectedStatus: func(t *testing.T, intent *nephoranv1.NetworkIntent) {
				assert.True(t, isConditionTrue(intent.Status.Conditions, "Processed"), "Processed condition should remain true")
				assert.False(t, isConditionTrue(intent.Status.Conditions, "Deployed"), "Deployed condition should be false")
			},
			description: "Should handle Git deployment failure with retry",
		},
		{
			name: "max_retries_exceeded",
			networkIntent: &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-intent",
					Namespace: "default",
					Annotations: map[string]string{
						"nephoran.com/llm-processing-retry-count": "3",
					},
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy a test network",
				},
			},
			existingObjects: []client.Object{},
			depsSetup: func(deps *fakeDependencies) {
				deps.llmClient.shouldFail = true
			},
			config: &Config{
				MaxRetries: 3,
				RetryDelay: 1 * time.Second,
				Timeout:    5 * time.Minute,
			},
			expectedResult: ctrl.Result{},
			expectedError:  true,
			expectedCalls: func(t *testing.T, deps *fakeDependencies) {
				assert.Equal(t, 0, deps.llmClient.callCount, "LLM should not be called when max retries exceeded")
			},
			expectedStatus: func(t *testing.T, intent *nephoranv1.NetworkIntent) {
				assert.False(t, isConditionTrue(intent.Status.Conditions, "Processed"), "Processed condition should be false")
			},
			description: "Should stop retrying after max retries exceeded",
		},
		{
			name: "successful_deletion_with_cleanup",
			networkIntent: &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-intent",
					Namespace:         "default",
					Finalizers:        []string{NetworkIntentFinalizer},
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy a test network",
				},
			},
			existingObjects: []client.Object{},
			depsSetup: func(deps *fakeDependencies) {
				// Cleanup should succeed
			},
			config: &Config{
				MaxRetries:    3,
				RetryDelay:    1 * time.Second,
				GitRepoURL:    "https://git.example.com/repo",
				GitDeployPath: "networkintents",
			},
			expectedResult: ctrl.Result{},
			expectedError:  false,
			expectedCalls: func(t *testing.T, deps *fakeDependencies) {
				assert.Equal(t, 0, deps.llmClient.callCount, "LLM should not be called during deletion")
				assert.Contains(t, deps.gitClient.CallHistory, "InitRepo", "Git InitRepo should be called for cleanup")
				assert.True(t, containsCallPattern(deps.gitClient.CallHistory, "RemoveDirectory"), "Git RemoveDirectory should be called")
			},
			expectedStatus: func(t *testing.T, intent *nephoranv1.NetworkIntent) {
				assert.NotContains(t, intent.Finalizers, NetworkIntentFinalizer, "Finalizer should be removed")
			},
			description: "Should successfully delete with proper cleanup",
		},
		{
			name: "resource_not_found",
			networkIntent: &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nonexistent-intent",
					Namespace: "default",
				},
			},
			existingObjects: []client.Object{}, // Don't add the NetworkIntent
			depsSetup:       func(deps *fakeDependencies) {},
			config:          &Config{},
			expectedResult:  ctrl.Result{},
			expectedError:   false,
			expectedCalls: func(t *testing.T, deps *fakeDependencies) {
				assert.Equal(t, 0, deps.llmClient.callCount, "LLM should not be called for nonexistent resource")
				assert.Empty(t, deps.gitClient.CallHistory, "Git should not be called for nonexistent resource")
			},
			expectedStatus: func(t *testing.T, intent *nephoranv1.NetworkIntent) {
				// No status checks for nonexistent resource
			},
			description: "Should handle resource not found gracefully",
		},
		{
			name: "already_completed_intent",
			networkIntent: &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-intent",
					Namespace:  "default",
					Finalizers: []string{NetworkIntentFinalizer},
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy a test network",
				},
				Status: nephoranv1.NetworkIntentStatus{
					Phase: "Completed",
					Conditions: []metav1.Condition{
						{
							Type:   "Processed",
							Status: metav1.ConditionTrue,
							Reason: "LLMProcessingSucceeded",
						},
						{
							Type:   "Deployed",
							Status: metav1.ConditionTrue,
							Reason: "GitDeploymentSucceeded",
						},
					},
				},
			},
			existingObjects: []client.Object{},
			depsSetup:       func(deps *fakeDependencies) {},
			config:          &Config{},
			expectedResult:  ctrl.Result{},
			expectedError:   false,
			expectedCalls: func(t *testing.T, deps *fakeDependencies) {
				assert.Equal(t, 0, deps.llmClient.callCount, "LLM should not be called for completed intent")
				assert.Empty(t, deps.gitClient.CallHistory, "Git should not be called for completed intent")
			},
			expectedStatus: func(t *testing.T, intent *nephoranv1.NetworkIntent) {
				assert.Equal(t, "Completed", intent.Status.Phase, "Phase should remain Completed")
			},
			description: "Should skip processing for already completed intent",
		},
		{
			name: "nil_llm_client",
			networkIntent: &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-intent",
					Namespace: "default",
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy a test network",
				},
			},
			existingObjects: []client.Object{},
			depsSetup: func(deps *fakeDependencies) {
				deps.llmClient = nil
			},
			config: &Config{
				MaxRetries: 3,
				RetryDelay: 1 * time.Second,
			},
			expectedResult: ctrl.Result{},
			expectedError:  true,
			expectedCalls: func(t *testing.T, deps *fakeDependencies) {
				// No calls should be made to nil client
			},
			expectedStatus: func(t *testing.T, intent *nephoranv1.NetworkIntent) {
				assert.False(t, isConditionTrue(intent.Status.Conditions, "Processed"), "Processed condition should be false")
			},
			description: "Should handle nil LLM client gracefully",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup the scheme
			s := scheme.Scheme
			err := nephoranv1.AddToScheme(s)
			require.NoError(t, err)

			// Create fake dependencies
			deps := newFakeDependencies()
			tc.depsSetup(deps)

			// Add the NetworkIntent to existing objects if it's not a "not found" test
			objects := tc.existingObjects
			if tc.name != "resource_not_found" {
				objects = append(objects, tc.networkIntent)
			}

			// Create fake client with existing objects
			fakeClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(objects...).
				WithStatusSubresource(&nephoranv1.NetworkIntent{}).
				Build()

			// Create reconciler
			reconciler, err := NewNetworkIntentReconciler(fakeClient, s, deps, tc.config)
			require.NoError(t, err)

			// Execute reconcile
			ctx := context.Background()
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      tc.networkIntent.Name,
					Namespace: tc.networkIntent.Namespace,
				},
			}

			result, err := reconciler.Reconcile(ctx, req)

			// Verify results
			if tc.expectedError {
				assert.Error(t, err, tc.description)
			} else {
				assert.NoError(t, err, tc.description)
			}

			assert.Equal(t, tc.expectedResult, result, tc.description)

			// Verify expected calls
			tc.expectedCalls(t, deps)

			// Verify status if object exists
			if tc.name != "resource_not_found" {
				var updatedIntent nephoranv1.NetworkIntent
				err = fakeClient.Get(ctx, req.NamespacedName, &updatedIntent)
				if err == nil {
					tc.expectedStatus(t, &updatedIntent)
				}
			}
		})
	}
}

func TestNetworkIntentController_ConfigValidation(t *testing.T) {
	testCases := []struct {
		name          string
		config        *Config
		expectedError bool
		description   string
	}{
		{
			name:          "nil_config_uses_defaults",
			config:        nil,
			expectedError: false,
			description:   "Should accept nil config and use defaults",
		},
		{
			name: "valid_config",
			config: &Config{
				MaxRetries:    5,
				RetryDelay:    30 * time.Second,
				Timeout:       10 * time.Minute,
				GitDeployPath: "custom-path",
			},
			expectedError: false,
			description:   "Should accept valid config",
		},
		{
			name: "max_retries_too_high",
			config: &Config{
				MaxRetries: 15, // Exceeds MaxAllowedRetries (10)
			},
			expectedError: true,
			description:   "Should reject config with too many max retries",
		},
		{
			name: "retry_delay_too_high",
			config: &Config{
				RetryDelay: 2 * time.Hour, // Exceeds MaxAllowedRetryDelay (1 hour)
			},
			expectedError: true,
			description:   "Should reject config with too high retry delay",
		},
		{
			name: "zero_values_use_defaults",
			config: &Config{
				MaxRetries: 0,
				RetryDelay: 0,
				Timeout:    0,
			},
			expectedError: false,
			description:   "Should use defaults for zero values",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := scheme.Scheme
			err := nephoranv1.AddToScheme(s)
			require.NoError(t, err)

			fakeClient := fake.NewClientBuilder().WithScheme(s).Build()
			deps := newFakeDependencies()

			reconciler, err := NewNetworkIntentReconciler(fakeClient, s, deps, tc.config)

			if tc.expectedError {
				assert.Error(t, err, tc.description)
				assert.Nil(t, reconciler, "Reconciler should be nil on config validation error")
			} else {
				assert.NoError(t, err, tc.description)
				assert.NotNil(t, reconciler, "Reconciler should not be nil on successful creation")
			}
		})
	}
}

func TestNetworkIntentController_ConcurrentReconciles(t *testing.T) {
	// Test concurrent reconcile operations
	s := scheme.Scheme
	err := nephoranv1.AddToScheme(s)
	require.NoError(t, err)

	intent := &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-intent",
			Namespace: "default",
		},
		Spec: nephoranv1.NetworkIntentSpec{
			Intent: "Deploy a test network",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(intent).
		WithStatusSubresource(&nephoranv1.NetworkIntent{}).
		Build()

	deps := newFakeDependencies()
	config := &Config{
		MaxRetries: 3,
		RetryDelay: 1 * time.Second,
		Timeout:    5 * time.Minute,
	}

	reconciler, err := NewNetworkIntentReconciler(fakeClient, s, deps, config)
	require.NoError(t, err)

	ctx := context.Background()
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      intent.Name,
			Namespace: intent.Namespace,
		},
	}

	// Run concurrent reconciles
	done := make(chan bool, 2)
	errors := make(chan error, 2)

	for i := 0; i < 2; i++ {
		go func() {
			defer func() { done <- true }()
			_, err := reconciler.Reconcile(ctx, req)
			if err != nil {
				errors <- err
			}
		}()
	}

	// Wait for completion
	for i := 0; i < 2; i++ {
		<-done
	}

	// Check for errors
	select {
	case err := <-errors:
		t.Logf("Concurrent reconcile error (may be acceptable): %v", err)
	default:
		// No errors
	}

	// Verify final state consistency
	var finalIntent nephoranv1.NetworkIntent
	err = fakeClient.Get(ctx, req.NamespacedName, &finalIntent)
	require.NoError(t, err)

	// Should have finalizer
	assert.Contains(t, finalIntent.Finalizers, NetworkIntentFinalizer, "Finalizer should be present")
}

// Helper functions

func containsCallPattern(history []string, pattern string) bool {
	for _, call := range history {
		if len(call) >= len(pattern) && call[:len(pattern)] == pattern {
			return true
		}
	}
	return false
}
