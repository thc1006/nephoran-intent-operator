package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/git"
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
	"github.com/thc1006/nephoran-intent-operator/pkg/telecom"
)

// Test constants
const (
	DefaultRetryDelay    = time.Second
	DefaultTimeout       = 30 * time.Second
	DefaultGitDeployPath = "networkintents"
)


// MockDependencies implements Dependencies interface for testing
type MockDependencies struct {
	mock.Mock
	gitClient            *MockGitClientComprehensive
	llmClient            *MockLLMClientComprehensive
	packageGenerator     *MockPackageGenerator
	httpClient           *http.Client
	eventRecorder        record.EventRecorder
	telecomKnowledgeBase *telecom.TelecomKnowledgeBase
	metricsCollector     *MockMetricsCollector
}

func NewMockDependencies() *MockDependencies {
	return &MockDependencies{
		gitClient:            NewMockGitClientComprehensive(),
		llmClient:            NewMockLLMClientComprehensive(),
		packageGenerator:     NewMockPackageGenerator(),
		httpClient:           &http.Client{Timeout: 30 * time.Second},
		eventRecorder:        &record.FakeRecorder{},
		telecomKnowledgeBase: telecom.NewTelecomKnowledgeBase(),
		metricsCollector:     NewMockMetricsCollector(),
	}
}

func (m *MockDependencies) GetGitClient() git.ClientInterface {
	return m.gitClient
}

func (m *MockDependencies) GetLLMClient() shared.ClientInterface {
	return m.llmClient
}

func (m *MockDependencies) GetPackageGenerator() *nephio.PackageGenerator {
	return nil // Return nil for now, can be enhanced later
}

func (m *MockDependencies) GetHTTPClient() *http.Client {
	return m.httpClient
}

func (m *MockDependencies) GetEventRecorder() record.EventRecorder {
	return m.eventRecorder
}

func (m *MockDependencies) GetTelecomKnowledgeBase() *telecom.TelecomKnowledgeBase {
	return m.telecomKnowledgeBase
}

func (m *MockDependencies) GetMetricsCollector() *monitoring.MetricsCollector {
	return nil // Return nil for now
}

// MockLLMClient for testing LLM integration
type MockLLMClientComprehensive struct {
	mock.Mock
	response  string
	err       error
	callCount int
	failCount int
	processed bool
}

func NewMockLLMClientComprehensive() *MockLLMClientComprehensive {
	return &MockLLMClientComprehensive{}
}

func (m *MockLLMClientComprehensive) ProcessIntent(ctx context.Context, intent string) (string, error) {
	m.callCount++
	if m.failCount > 0 && m.callCount <= m.failCount {
		return "", m.err
	}
	return m.response, nil
}

func (m *MockLLMClientComprehensive) SetResponse(response string) {
	m.response = response
}

func (m *MockLLMClientComprehensive) SetError(err error) {
	m.err = err
}

func (m *MockLLMClientComprehensive) SetFailCount(count int) {
	m.failCount = count
}

func (m *MockLLMClientComprehensive) Close() error {
	// No-op for mock
	return nil
}

func (m *MockLLMClientComprehensive) EstimateTokens(text string) int {
	// Simple token estimation for testing
	return len(text) / 4
}

func (m *MockLLMClientComprehensive) GetMaxTokens(modelName string) int {
	// Return a reasonable token limit for testing, ignore model name
	return 4096
}

func (m *MockLLMClientComprehensive) GetModelCapabilities(modelName string) (*shared.ModelCapabilities, error) {
	// Return basic capabilities for testing
	return &shared.ModelCapabilities{
		MaxTokens:         4096,
		SupportsChat:      true,
		SupportsFunction:  true,
		SupportsStreaming: false,
		CostPerToken:      0.001,
		Features:          map[string]interface{}{"test": true},
	}, nil
}

func (m *MockLLMClientComprehensive) GetSupportedModels() []string {
	return []string{"gpt-4", "gpt-3.5-turbo"}
}

func (m *MockLLMClientComprehensive) ValidateModel(modelName string) error {
	// All models are valid in testing
	return nil
}

func (m *MockLLMClientComprehensive) ProcessIntentStream(ctx context.Context, prompt string, chunks chan<- *shared.StreamingChunk) error {
	// Simple streaming implementation for testing
	chunks <- &shared.StreamingChunk{
		Content: "Test streaming response",
		IsLast:  true,
	}
	close(chunks)
	return nil
}

// MockGitClient for testing Git operations
type MockGitClientComprehensive struct {
	mock.Mock
	shouldFail bool
	filePaths  []string
}

func NewMockGitClientComprehensive() *MockGitClientComprehensive {
	return &MockGitClientComprehensive{}
}

func (m *MockGitClientComprehensive) CommitAndPush(files map[string]string, message string) (string, error) {
	args := m.Called(files, message)
	return args.String(0), args.Error(1)
}

func (m *MockGitClientComprehensive) CommitAndPushChanges(message string) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *MockGitClientComprehensive) InitRepo() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockGitClientComprehensive) RemoveDirectory(path string, commitMessage string) error {
	args := m.Called(path, commitMessage)
	return args.Error(0)
}

func (m *MockGitClientComprehensive) CommitFiles(files []string, message string) error {
	args := m.Called(files, message)
	return args.Error(0)
}

func (m *MockGitClientComprehensive) CreateBranch(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockGitClientComprehensive) SwitchBranch(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockGitClientComprehensive) GetCurrentBranch() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockGitClientComprehensive) ListBranches() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockGitClientComprehensive) GetFileContent(path string) ([]byte, error) {
	args := m.Called(path)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockGitClientComprehensive) Add(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *MockGitClientComprehensive) Remove(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *MockGitClientComprehensive) Move(oldPath, newPath string) error {
	args := m.Called(oldPath, newPath)
	return args.Error(0)
}

func (m *MockGitClientComprehensive) Restore(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *MockGitClientComprehensive) DeleteBranch(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockGitClientComprehensive) MergeBranch(sourceBranch, targetBranch string) error {
	args := m.Called(sourceBranch, targetBranch)
	return args.Error(0)
}

func (m *MockGitClientComprehensive) RebaseBranch(sourceBranch, targetBranch string) error {
	args := m.Called(sourceBranch, targetBranch)
	return args.Error(0)
}

func (m *MockGitClientComprehensive) CherryPick(commitHash string) error {
	args := m.Called(commitHash)
	return args.Error(0)
}

func (m *MockGitClientComprehensive) Reset(options git.ResetOptions) error {
	args := m.Called(options)
	return args.Error(0)
}

func (m *MockGitClientComprehensive) Clean(force bool) error {
	args := m.Called(force)
	return args.Error(0)
}

func (m *MockGitClientComprehensive) GetCommitHistory(options git.LogOptions) ([]git.CommitInfo, error) {
	args := m.Called(options)
	return args.Get(0).([]git.CommitInfo), args.Error(1)
}

func (m *MockGitClientComprehensive) CreateTag(name, message string) error {
	args := m.Called(name, message)
	return args.Error(0)
}

func (m *MockGitClientComprehensive) ListTags() ([]git.TagInfo, error) {
	args := m.Called()
	return args.Get(0).([]git.TagInfo), args.Error(1)
}

func (m *MockGitClientComprehensive) GetTagInfo(name string) (git.TagInfo, error) {
	args := m.Called(name)
	return args.Get(0).(git.TagInfo), args.Error(1)
}

func (m *MockGitClientComprehensive) CreatePullRequest(options git.PullRequestOptions) (git.PullRequestInfo, error) {
	args := m.Called(options)
	return args.Get(0).(git.PullRequestInfo), args.Error(1)
}

func (m *MockGitClientComprehensive) GetPullRequestStatus(id int) (string, error) {
	args := m.Called(id)
	return args.String(0), args.Error(1)
}

func (m *MockGitClientComprehensive) ApprovePullRequest(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockGitClientComprehensive) MergePullRequest(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockGitClientComprehensive) GetDiff(options git.DiffOptions) (string, error) {
	args := m.Called(options)
	return args.String(0), args.Error(1)
}

func (m *MockGitClientComprehensive) GetStatus() ([]git.StatusInfo, error) {
	args := m.Called()
	return args.Get(0).([]git.StatusInfo), args.Error(1)
}

func (m *MockGitClientComprehensive) ApplyPatch(patch string) error {
	args := m.Called(patch)
	return args.Error(0)
}

func (m *MockGitClientComprehensive) CreatePatch(options git.DiffOptions) (string, error) {
	args := m.Called(options)
	return args.String(0), args.Error(1)
}

func (m *MockGitClientComprehensive) GetRemotes() ([]git.RemoteInfo, error) {
	args := m.Called()
	return args.Get(0).([]git.RemoteInfo), args.Error(1)
}

func (m *MockGitClientComprehensive) AddRemote(name, url string) error {
	args := m.Called(name, url)
	return args.Error(0)
}

func (m *MockGitClientComprehensive) RemoveRemote(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockGitClientComprehensive) Fetch(remote string) error {
	args := m.Called(remote)
	return args.Error(0)
}

func (m *MockGitClientComprehensive) Pull(remote string) error {
	args := m.Called(remote)
	return args.Error(0)
}

func (m *MockGitClientComprehensive) Push(remote string) error {
	args := m.Called(remote)
	return args.Error(0)
}

func (m *MockGitClientComprehensive) GetLog(options git.LogOptions) ([]git.CommitInfo, error) {
	args := m.Called(options)
	return args.Get(0).([]git.CommitInfo), args.Error(1)
}

func (m *MockGitClientComprehensive) PushChanges() error {
	if m.shouldFail {
		return errors.New("git push failed")
	}
	return nil
}

func (m *MockGitClientComprehensive) Clone(url, branch string) error {
	return nil
}

func (m *MockGitClientComprehensive) DeleteFiles(filePaths []string) error {
	if m.shouldFail {
		return errors.New("git delete failed")
	}
	m.filePaths = filePaths
	return nil
}

func (m *MockGitClientComprehensive) SetShouldFail(shouldFail bool) {
	m.shouldFail = shouldFail
}

// MockPackageGenerator for testing Nephio package generation
type MockPackageGenerator struct {
	mock.Mock
}

func NewMockPackageGenerator() *MockPackageGenerator {
	return &MockPackageGenerator{}
}

// MockMetricsCollector for testing metrics collection
type MockMetricsCollector struct {
	mock.Mock
}

func NewMockMetricsCollector() *MockMetricsCollector {
	return &MockMetricsCollector{}
}

// Test helper functions
func createTestNetworkIntent(name, namespace, intent string) *nephoranv1.NetworkIntent {
	return &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: nephoranv1.NetworkIntentSpec{
			Intent: intent,
		},
		Status: nephoranv1.NetworkIntentStatus{
			Phase: "Pending",
		},
	}
}

func createTestConfig() *Config {
	return &Config{
		MaxRetries:      DefaultMaxRetries,
		RetryDelay:      DefaultRetryDelay,
		Timeout:         DefaultTimeout,
		GitRepoURL:      "https://github.com/test/test-repo.git",
		GitBranch:       "main",
		GitDeployPath:   DefaultGitDeployPath,
		LLMProcessorURL: "http://localhost:8080/process",
		UseNephioPorch:  false,
	}
}

// Comprehensive unit tests for NetworkIntent controller
func TestNewNetworkIntentReconciler(t *testing.T) {
	tests := []struct {
		name          string
		client        client.Client
		scheme        *runtime.Scheme
		deps          Dependencies
		config        *Config
		expectedError bool
	}{
		{
			name:          "successful creation",
			client:        fake.NewClientBuilder().Build(),
			scheme:        runtime.NewScheme(),
			deps:          NewMockDependencies(),
			config:        createTestConfig(),
			expectedError: false,
		},
		{
			name:          "nil client",
			client:        nil,
			scheme:        runtime.NewScheme(),
			deps:          NewMockDependencies(),
			config:        createTestConfig(),
			expectedError: true,
		},
		{
			name:          "nil scheme",
			client:        fake.NewClientBuilder().Build(),
			scheme:        nil,
			deps:          NewMockDependencies(),
			config:        createTestConfig(),
			expectedError: true,
		},
		{
			name:          "nil dependencies",
			client:        fake.NewClientBuilder().Build(),
			scheme:        runtime.NewScheme(),
			deps:          nil,
			config:        createTestConfig(),
			expectedError: true,
		},
		{
			name:          "nil config",
			client:        fake.NewClientBuilder().Build(),
			scheme:        runtime.NewScheme(),
			deps:          NewMockDependencies(),
			config:        nil,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reconciler, err := NewNetworkIntentReconciler(tt.client, tt.scheme, tt.deps, tt.config)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, reconciler)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, reconciler)
				assert.Equal(t, tt.client, reconciler.Client)
				assert.Equal(t, tt.scheme, reconciler.Scheme)
				assert.Equal(t, tt.deps, reconciler.deps)
				assert.Equal(t, tt.config, reconciler.config)
			}
		})
	}
}

func TestReconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	nephoranv1.AddToScheme(scheme)

	tests := []struct {
		name            string
		networkIntent   *nephoranv1.NetworkIntent
		mockSetup       func(*MockDependencies)
		enableLLMIntent string
		expectedResult  ctrl.Result
		expectedError   bool
		expectedPhase   string
		validationCheck func(t *testing.T, ni *nephoranv1.NetworkIntent)
	}{
		{
			name:          "successful reconciliation with LLM processing",
			networkIntent: createTestNetworkIntent("test-intent", "default", "Deploy AMF network function"),
			mockSetup: func(deps *MockDependencies) {
				llmResponse := map[string]interface{}{
					"action":    "deploy",
					"component": "amf",
					"namespace": "5g-core",
					"replicas":  1,
					"resources": map[string]interface{}{
						"cpu":    "500m",
						"memory": "512Mi",
					},
				}
				responseJSON, _ := json.Marshal(llmResponse)
				deps.llmClient.SetResponse(string(responseJSON))
			},
			enableLLMIntent: "true",
			expectedResult:  ctrl.Result{},
			expectedError:   false,
			expectedPhase:   "Processed",
			validationCheck: func(t *testing.T, ni *nephoranv1.NetworkIntent) {
				assert.Equal(t, "Processed", ni.Status.Phase)
				assert.True(t, isConditionTrueLocal(ni.Status.Conditions, "Processed"))
			},
		},
		{
			name:          "reconciliation with LLM disabled",
			networkIntent: createTestNetworkIntent("test-intent-no-llm", "default", "Deploy SMF network function"),
			mockSetup: func(deps *MockDependencies) {
				// No LLM setup needed when disabled
			},
			enableLLMIntent: "false",
			expectedResult:  ctrl.Result{},
			expectedError:   false,
			expectedPhase:   "Processed",
			validationCheck: func(t *testing.T, ni *nephoranv1.NetworkIntent) {
				assert.Equal(t, "Processed", ni.Status.Phase)
			},
		},
		{
			name:          "LLM processing failure with retry",
			networkIntent: createTestNetworkIntent("test-intent-fail", "default", "Deploy UPF network function"),
			mockSetup: func(deps *MockDependencies) {
				deps.llmClient.SetError(errors.New("LLM service unavailable"))
				deps.llmClient.SetFailCount(1)
			},
			enableLLMIntent: "true",
			expectedResult:  ctrl.Result{RequeueAfter: DefaultRetryDelay},
			expectedError:   false,
			expectedPhase:   "Error",
			validationCheck: func(t *testing.T, ni *nephoranv1.NetworkIntent) {
				assert.Equal(t, "Error", ni.Status.Phase)
				assert.True(t, isConditionFalse(ni.Status.Conditions, "Processed"))
			},
		},
		{
			name:            "empty intent handling",
			networkIntent:   createTestNetworkIntent("test-empty-intent", "default", ""),
			mockSetup:       func(deps *MockDependencies) {},
			enableLLMIntent: "true",
			expectedResult:  ctrl.Result{},
			expectedError:   false,
			expectedPhase:   "Error",
			validationCheck: func(t *testing.T, ni *nephoranv1.NetworkIntent) {
				assert.Equal(t, "Error", ni.Status.Phase)
				assert.Contains(t, getConditionMessage(ni.Status.Conditions, "Processed"), "empty")
			},
		},
		{
			name: "intent with finalizer deletion",
			networkIntent: func() *nephoranv1.NetworkIntent {
				ni := createTestNetworkIntent("test-finalizer", "default", "Deploy NSSF")
				ni.ObjectMeta.Finalizers = []string{NetworkIntentFinalizer}
				now := metav1.NewTime(time.Now())
				ni.ObjectMeta.DeletionTimestamp = &now
				return ni
			}(),
			mockSetup: func(deps *MockDependencies) {
				// Git client should handle cleanup
				deps.gitClient.SetShouldFail(false)
			},
			enableLLMIntent: "true",
			expectedResult:  ctrl.Result{},
			expectedError:   false,
			expectedPhase:   "Processed", // Phase before deletion
			validationCheck: func(t *testing.T, ni *nephoranv1.NetworkIntent) {
				// Should handle deletion path
				assert.NotNil(t, ni.ObjectMeta.DeletionTimestamp)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable for LLM intent processing
			if tt.enableLLMIntent != "" {
				os.Setenv("ENABLE_LLM_INTENT", tt.enableLLMIntent)
				defer os.Unsetenv("ENABLE_LLM_INTENT")
			}

			// Create fake client with the test NetworkIntent
			clientBuilder := fake.NewClientBuilder().WithScheme(scheme)
			if tt.networkIntent != nil {
				clientBuilder = clientBuilder.WithObjects(tt.networkIntent)
			}
			fakeClient := clientBuilder.Build()

			// Setup mock dependencies
			mockDeps := NewMockDependencies()
			if tt.mockSetup != nil {
				tt.mockSetup(mockDeps)
			}

			// Create reconciler
			reconciler, err := NewNetworkIntentReconciler(fakeClient, scheme, mockDeps, createTestConfig())
			require.NoError(t, err)

			// Create reconcile request
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      tt.networkIntent.Name,
					Namespace: tt.networkIntent.Namespace,
				},
			}

			// Execute reconciliation
			ctx := context.Background()
			result, err := reconciler.Reconcile(ctx, req)

			// Verify results
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedResult, result)

			// Get updated NetworkIntent and run validation checks
			if tt.validationCheck != nil && !tt.expectedError {
				updatedNI := &nephoranv1.NetworkIntent{}
				err = fakeClient.Get(ctx, types.NamespacedName{
					Name:      tt.networkIntent.Name,
					Namespace: tt.networkIntent.Namespace,
				}, updatedNI)
				if err == nil {
					tt.validationCheck(t, updatedNI)
				}
			}
		})
	}
}

func TestExtractIntentType(t *testing.T) {
	scheme := runtime.NewScheme()
	nephoranv1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	mockDeps := NewMockDependencies()
	reconciler, _ := NewNetworkIntentReconciler(fakeClient, scheme, mockDeps, createTestConfig())

	tests := []struct {
		name         string
		intent       string
		expectedType string
	}{
		{"embb slice", "Deploy eMBB slice for high bandwidth", "embb"},
		{"urllc slice", "Configure URLLC slice for low latency", "urllc"},
		{"mmtc slice", "Setup mMTC slice for IoT devices", "mmtc"},
		{"5gc deployment", "Deploy 5G core AMF function", "5gc"},
		{"oran deployment", "Setup O-RAN components", "oran"},
		{"unknown intent", "Do something generic", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := reconciler.extractIntentType(tt.intent)
			assert.Equal(t, tt.expectedType, actual)
		})
	}
}

func TestUpdatePhase(t *testing.T) {
	scheme := runtime.NewScheme()
	nephoranv1.AddToScheme(scheme)

	tests := []struct {
		name          string
		initialPhase  string
		targetPhase   string
		expectedError bool
		shouldUpdate  bool
	}{
		{
			name:          "phase change from Pending to Processing",
			initialPhase:  "Pending",
			targetPhase:   "Processing",
			expectedError: false,
			shouldUpdate:  true,
		},
		{
			name:          "no change needed",
			initialPhase:  "Processed",
			targetPhase:   "Processed",
			expectedError: false,
			shouldUpdate:  false,
		},
		{
			name:          "phase change from Error to Processing",
			initialPhase:  "Error",
			targetPhase:   "Processing",
			expectedError: false,
			shouldUpdate:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test NetworkIntent
			ni := createTestNetworkIntent("test-phase", "default", "test intent")
			ni.Status.Phase = nephoranv1.NetworkIntentPhase(tt.initialPhase)

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ni).Build()
			mockDeps := NewMockDependencies()
			reconciler, _ := NewNetworkIntentReconciler(fakeClient, scheme, mockDeps, createTestConfig())

			ctx := context.Background()
			err := reconciler.updatePhase(ctx, ni, tt.targetPhase)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.shouldUpdate {
					assert.Equal(t, tt.targetPhase, ni.Status.Phase)
				} else {
					assert.Equal(t, tt.initialPhase, ni.Status.Phase)
				}
			}
		})
	}
}

// Test helper functions for conditions
func isConditionTrueLocal(conditions []metav1.Condition, conditionType string) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition.Status == metav1.ConditionTrue
		}
	}
	return false
}

func isConditionFalse(conditions []metav1.Condition, conditionType string) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition.Status == metav1.ConditionFalse
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

func TestProcessingContext(t *testing.T) {
	tests := []struct {
		name              string
		intentType        string
		extractedEntities map[string]interface{}
		telecomContext    map[string]interface{}
		expectedValid     bool
	}{
		{
			name:       "valid 5gc context",
			intentType: "5gc",
			extractedEntities: map[string]interface{}{
				"component": "amf",
				"action":    "deploy",
			},
			telecomContext: map[string]interface{}{
				"network_functions": []string{"amf"},
				"deployment_type":   "production",
			},
			expectedValid: true,
		},
		{
			name:              "empty context",
			intentType:        "unknown",
			extractedEntities: map[string]interface{}{},
			telecomContext:    map[string]interface{}{},
			expectedValid:     true, // Empty context is still valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &ProcessingContext{
				StartTime:         time.Now(),
				CurrentPhase:      PhaseLLMProcessing,
				IntentType:        tt.intentType,
				ExtractedEntities: tt.extractedEntities,
				TelecomContext:    tt.telecomContext,
			}

			assert.NotNil(t, ctx)
			assert.Equal(t, tt.intentType, ctx.IntentType)
			assert.Equal(t, PhaseLLMProcessing, ctx.CurrentPhase)
			assert.NotZero(t, ctx.StartTime)
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkReconcile(b *testing.B) {
	scheme := runtime.NewScheme()
	nephoranv1.AddToScheme(scheme)

	ni := createTestNetworkIntent("benchmark-intent", "default", "Deploy AMF network function")
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ni).Build()

	mockDeps := NewMockDependencies()
	llmResponse := map[string]interface{}{
		"action":    "deploy",
		"component": "amf",
		"namespace": "5g-core",
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

func BenchmarkExtractIntentType(b *testing.B) {
	scheme := runtime.NewScheme()
	nephoranv1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	mockDeps := NewMockDependencies()
	reconciler, _ := NewNetworkIntentReconciler(fakeClient, scheme, mockDeps, createTestConfig())

	intents := []string{
		"Deploy eMBB slice for high bandwidth applications",
		"Configure URLLC slice for low latency requirements",
		"Setup 5G core AMF network function",
		"Deploy O-RAN components for edge computing",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, intent := range intents {
			reconciler.extractIntentType(intent)
		}
	}
}
