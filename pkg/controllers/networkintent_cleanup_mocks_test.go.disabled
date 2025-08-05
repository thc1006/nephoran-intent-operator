package controllers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/stretchr/testify/mock"
	"github.com/thc1006/nephoran-intent-operator/pkg/git"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
	"k8s.io/client-go/tools/record"
)

// MockDependencies implements the Dependencies interface for testing
type MockDependencies struct {
	gitClient        git.ClientInterface
	llmClient        shared.ClientInterface
	packageGenerator *nephio.PackageGenerator
	httpClient       *http.Client
	eventRecorder    record.EventRecorder
}

func (m *MockDependencies) GetGitClient() git.ClientInterface {
	return m.gitClient
}

func (m *MockDependencies) GetLLMClient() shared.ClientInterface {
	return m.llmClient
}

func (m *MockDependencies) GetPackageGenerator() *nephio.PackageGenerator {
	return m.packageGenerator
}

func (m *MockDependencies) GetHTTPClient() *http.Client {
	return m.httpClient
}

func (m *MockDependencies) GetEventRecorder() record.EventRecorder {
	return m.eventRecorder
}

// MockGitClientInterface is a mock implementation of git.ClientInterface
type MockGitClientInterface struct {
	mock.Mock
}

func (m *MockGitClientInterface) CommitAndPush(files map[string]string, message string) (string, error) {
	args := m.Called(files, message)
	return args.String(0), args.Error(1)
}

func (m *MockGitClientInterface) CommitAndPushChanges(message string) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *MockGitClientInterface) InitRepo() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockGitClientInterface) RemoveDirectory(path string, commitMessage string) error {
	args := m.Called(path, commitMessage)
	return args.Error(0)
}

// MockLLMClientInterface is a mock implementation of shared.ClientInterface
type MockLLMClientInterface struct {
	mock.Mock
}

func (m *MockLLMClientInterface) ProcessIntent(ctx interface{}, intent string) (string, error) {
	args := m.Called(ctx, intent)
	return args.String(0), args.Error(1)
}

// Enhanced mock implementations with more detailed behavior control

// EnhancedMockGitClient provides more sophisticated mock behavior
type EnhancedMockGitClient struct {
	mock.Mock
	callCounts       map[string]int
	failureCounts    map[string]int
	maxFailures      map[string]int
	simulateLatency  bool
	latencyDuration  int // milliseconds
}

func NewEnhancedMockGitClient() *EnhancedMockGitClient {
	return &EnhancedMockGitClient{
		callCounts:    make(map[string]int),
		failureCounts: make(map[string]int),
		maxFailures:   make(map[string]int),
	}
}

func (m *EnhancedMockGitClient) SetMaxFailures(method string, count int) {
	m.maxFailures[method] = count
}

func (m *EnhancedMockGitClient) SetLatencySimulation(enabled bool, durationMs int) {
	m.simulateLatency = enabled
	m.latencyDuration = durationMs
}

func (m *EnhancedMockGitClient) RemoveDirectory(path string, commitMessage string) error {
	method := "RemoveDirectory"
	m.callCounts[method]++
	
	// Simulate transient failures
	if maxFails, exists := m.maxFailures[method]; exists {
		if m.failureCounts[method] < maxFails {
			m.failureCounts[method]++
			args := m.Called(path, commitMessage)
			return args.Error(0)
		}
	}
	
	args := m.Called(path, commitMessage)
	return args.Error(0)
}

func (m *EnhancedMockGitClient) CommitAndPushChanges(message string) error {
	method := "CommitAndPushChanges"
	m.callCounts[method]++
	
	// Simulate transient failures
	if maxFails, exists := m.maxFailures[method]; exists {
		if m.failureCounts[method] < maxFails {
			m.failureCounts[method]++
			args := m.Called(message)
			return args.Error(0)
		}
	}
	
	args := m.Called(message)
	return args.Error(0)
}

func (m *EnhancedMockGitClient) CommitAndPush(files map[string]string, message string) (string, error) {
	method := "CommitAndPush"
	m.callCounts[method]++
	
	args := m.Called(files, message)
	return args.String(0), args.Error(1)
}

func (m *EnhancedMockGitClient) InitRepo() error {
	method := "InitRepo"
	m.callCounts[method]++
	
	args := m.Called()
	return args.Error(0)
}

func (m *EnhancedMockGitClient) GetCallCount(method string) int {
	return m.callCounts[method]
}

func (m *EnhancedMockGitClient) GetFailureCount(method string) int {
	return m.failureCounts[method]
}

// ScenarioBasedMockGitClient allows defining specific scenarios for testing
type ScenarioBasedMockGitClient struct {
	mock.Mock
	scenarios map[string]*GitScenario
}

type GitScenario struct {
	Name            string
	RemoveDirError  error
	CommitError     error
	CallSequence    []string
	ExpectedCalls   int
	ActualCalls     int
}

func NewScenarioBasedMockGitClient() *ScenarioBasedMockGitClient {
	return &ScenarioBasedMockGitClient{
		scenarios: make(map[string]*GitScenario),
	}
}

func (m *ScenarioBasedMockGitClient) AddScenario(name string, scenario *GitScenario) {
	m.scenarios[name] = scenario
}

func (m *ScenarioBasedMockGitClient) ExecuteScenario(scenarioName string) {
	scenario, exists := m.scenarios[scenarioName]
	if !exists {
		return
	}
	
	// Set up expectations based on scenario
	if scenario.RemoveDirError != nil {
		m.On("RemoveDirectory", mock.Anything, mock.Anything).Return(scenario.RemoveDirError)
	} else {
		m.On("RemoveDirectory", mock.Anything, mock.Anything).Return(nil)
	}
	
	if scenario.CommitError != nil {
		m.On("CommitAndPushChanges", mock.Anything).Return(scenario.CommitError)
	} else {
		m.On("CommitAndPushChanges", mock.Anything).Return(nil)
	}
}

func (m *ScenarioBasedMockGitClient) RemoveDirectory(path string, commitMessage string) error {
	args := m.Called(path, commitMessage)
	return args.Error(0)
}

func (m *ScenarioBasedMockGitClient) CommitAndPushChanges(message string) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *ScenarioBasedMockGitClient) CommitAndPush(files map[string]string, message string) (string, error) {
	args := m.Called(files, message)
	return args.String(0), args.Error(1)
}

func (m *ScenarioBasedMockGitClient) InitRepo() error {
	args := m.Called()
	return args.Error(0)
}

// MockDependenciesBuilder provides a fluent interface for building mock dependencies
type MockDependenciesBuilder struct {
	deps *MockDependencies
}

func NewMockDependenciesBuilder() *MockDependenciesBuilder {
	return &MockDependenciesBuilder{
		deps: &MockDependencies{
			httpClient:    &http.Client{},
			eventRecorder: &record.FakeRecorder{Events: make(chan string, 100)},
		},
	}
}

func (b *MockDependenciesBuilder) WithGitClient(client git.ClientInterface) *MockDependenciesBuilder {
	b.deps.gitClient = client
	return b
}

func (b *MockDependenciesBuilder) WithLLMClient(client shared.ClientInterface) *MockDependenciesBuilder {
	b.deps.llmClient = client
	return b
}

func (b *MockDependenciesBuilder) WithPackageGenerator(generator *nephio.PackageGenerator) *MockDependenciesBuilder {
	b.deps.packageGenerator = generator
	return b
}

func (b *MockDependenciesBuilder) WithHTTPClient(client *http.Client) *MockDependenciesBuilder {
	b.deps.httpClient = client
	return b
}

func (b *MockDependenciesBuilder) WithEventRecorder(recorder record.EventRecorder) *MockDependenciesBuilder {
	b.deps.eventRecorder = recorder
	return b
}

func (b *MockDependenciesBuilder) Build() *MockDependencies {
	return b.deps
}

// Test utilities for mock validation

// GitClientCallValidator helps validate git client call patterns
type GitClientCallValidator struct {
	expectedCalls map[string][]interface{}
	actualCalls   map[string][]interface{}
}

func NewGitClientCallValidator() *GitClientCallValidator {
	return &GitClientCallValidator{
		expectedCalls: make(map[string][]interface{}),
		actualCalls:   make(map[string][]interface{}),
	}
}

func (v *GitClientCallValidator) ExpectRemoveDirectory(path, commitMessage string) *GitClientCallValidator {
	v.expectedCalls["RemoveDirectory"] = []interface{}{path, commitMessage}
	return v
}

func (v *GitClientCallValidator) ExpectCommitAndPushChanges(message string) *GitClientCallValidator {
	v.expectedCalls["CommitAndPushChanges"] = []interface{}{message}
	return v
}

func (v *GitClientCallValidator) RecordCall(method string, args ...interface{}) {
	v.actualCalls[method] = args
}

func (v *GitClientCallValidator) Validate() error {
	for method, expectedArgs := range v.expectedCalls {
		actualArgs, exists := v.actualCalls[method]
		if !exists {
			return fmt.Errorf("expected call to %s was not made", method)
		}
		
		if len(expectedArgs) != len(actualArgs) {
			return fmt.Errorf("method %s: expected %d args, got %d", method, len(expectedArgs), len(actualArgs))
		}
		
		for i, expected := range expectedArgs {
			if expected != actualArgs[i] {
				return fmt.Errorf("method %s: arg %d expected %v, got %v", method, i, expected, actualArgs[i])
			}
		}
	}
	return nil
}

// Common error types for testing
var (
	ErrGitAuthenticationFailed = errors.New("SSH key authentication failed")
	ErrGitNetworkTimeout      = errors.New("network timeout connecting to remote repository")
	ErrGitRepositoryCorrupted = errors.New("repository is corrupted or locked")
	ErrGitDirectoryNotFound   = errors.New("directory not found in repository")
	ErrGitPushRejected        = errors.New("push rejected by remote repository")
	ErrGitNoChangesToCommit   = errors.New("no changes to commit")
)

// Test scenario factory functions
func CreateAuthFailureScenario() *GitScenario {
	return &GitScenario{
		Name:           "AuthFailure",
		RemoveDirError: ErrGitAuthenticationFailed,
		CommitError:    nil,
		ExpectedCalls:  1,
	}
}

func CreateNetworkTimeoutScenario() *GitScenario {
	return &GitScenario{
		Name:           "NetworkTimeout",
		RemoveDirError: nil,
		CommitError:    ErrGitNetworkTimeout,
		ExpectedCalls:  2,
	}
}

func CreateRepositoryCorruptionScenario() *GitScenario {
	return &GitScenario{
		Name:           "RepositoryCorruption",
		RemoveDirError: ErrGitRepositoryCorrupted,
		CommitError:    nil,
		ExpectedCalls:  1,
	}
}

func CreateSuccessfulCleanupScenario() *GitScenario {
	return &GitScenario{
		Name:           "SuccessfulCleanup",
		RemoveDirError: nil,
		CommitError:    nil,
		ExpectedCalls:  2,
	}
}

func CreateTransientFailureScenario() *GitScenario {
	return &GitScenario{
		Name:           "TransientFailure",
		RemoveDirError: ErrGitNetworkTimeout, // First call fails
		CommitError:    nil,                  // Second call succeeds
		ExpectedCalls:  2,
	}
}