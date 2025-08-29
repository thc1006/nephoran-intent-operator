package testutils

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"k8s.io/client-go/tools/record"

	"github.com/thc1006/nephoran-intent-operator/pkg/git"
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio"
	"github.com/thc1006/nephoran-intent-operator/pkg/telecom"
	"github.com/thc1006/nephoran-intent-operator/pkg/types"
)

// MockDependencies implements the Dependencies interface for testing
type MockDependencies struct {
	gitClient            *MockGitClient
	llmClient            *MockLLMClient
	packageGenerator     *MockPackageGenerator
	httpClient           *http.Client
	eventRecorder        *record.FakeRecorder
	telecomKnowledgeBase *telecom.TelecomKnowledgeBase
	metricsCollector     *MockMetricsCollector

	// Control behavior
	simulateErrors  map[string]error
	operationDelays map[string]time.Duration
	callCounts      map[string]int
	mu              sync.RWMutex
}

// NewMockDependencies creates a new mock dependencies instance
func NewMockDependencies() *MockDependencies {
	return &MockDependencies{
		gitClient:        NewMockGitClient(),
		llmClient:        NewMockLLMClient(),
		packageGenerator: NewMockPackageGenerator(),
		httpClient:       &http.Client{Timeout: 30 * time.Second},
		eventRecorder:    record.NewFakeRecorder(100),
		metricsCollector: NewMockMetricsCollector(),
		simulateErrors:   make(map[string]error),
		operationDelays:  make(map[string]time.Duration),
		callCounts:       make(map[string]int),
	}
}

// Implement Dependencies interface
func (m *MockDependencies) GetGitClient() git.ClientInterface {
	m.incrementCallCount("GetGitClient")
	return m.gitClient
}

func (m *MockDependencies) GetLLMClient() types.ClientInterface {
	m.incrementCallCount("GetLLMClient")
	return m.llmClient
}

func (m *MockDependencies) GetPackageGenerator() *nephio.PackageGenerator {
	m.incrementCallCount("GetPackageGenerator")
	return &nephio.PackageGenerator{} // Return a real instance for simplicity
}

func (m *MockDependencies) GetHTTPClient() *http.Client {
	m.incrementCallCount("GetHTTPClient")
	return m.httpClient
}

func (m *MockDependencies) GetEventRecorder() record.EventRecorder {
	m.incrementCallCount("GetEventRecorder")
	return m.eventRecorder
}

func (m *MockDependencies) GetTelecomKnowledgeBase() *telecom.TelecomKnowledgeBase {
	m.incrementCallCount("GetTelecomKnowledgeBase")
	if m.telecomKnowledgeBase == nil {
		m.telecomKnowledgeBase = &telecom.TelecomKnowledgeBase{}
	}
	return m.telecomKnowledgeBase
}

func (m *MockDependencies) GetMetricsCollector() *monitoring.MetricsCollector {
	m.incrementCallCount("GetMetricsCollector")
	return &monitoring.MetricsCollector{} // Return a real instance for simplicity
}

// Control methods
func (m *MockDependencies) SetError(operation string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateErrors[operation] = err
}

func (m *MockDependencies) SetDelay(operation string, delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.operationDelays[operation] = delay
}

func (m *MockDependencies) GetCallCount(operation string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCounts[operation]
}

func (m *MockDependencies) incrementCallCount(operation string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCounts[operation]++
}

func (m *MockDependencies) simulateDelay(operation string) {
	m.mu.RLock()
	delay, exists := m.operationDelays[operation]
	m.mu.RUnlock()

	if exists && delay > 0 {
		time.Sleep(delay)
	}
}

func (m *MockDependencies) checkError(operation string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.simulateErrors[operation]
}

// MockGitClient implements git.ClientInterface for testing
type MockGitClient struct {
	commits     []string
	pushResults map[string]error
	pullResults map[string]error
	files       map[string]string
	mu          sync.RWMutex
}

func NewMockGitClient() *MockGitClient {
	return &MockGitClient{
		commits:     make([]string, 0),
		pushResults: make(map[string]error),
		pullResults: make(map[string]error),
		files:       make(map[string]string),
	}
}

func (m *MockGitClient) InitRepo() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pushResults["InitRepo"]
}

func (m *MockGitClient) CommitAndPush(files map[string]string, message string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	commitHash := fmt.Sprintf("commit-%d", len(m.commits))
	m.commits = append(m.commits, commitHash)

	for path, content := range files {
		m.files[path] = content
	}

	if err := m.pushResults["CommitAndPush"]; err != nil {
		return "", err
	}
	return commitHash, nil
}

func (m *MockGitClient) CommitAndPushChanges(message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pushResults["CommitAndPushChanges"]
}

func (m *MockGitClient) RemoveDirectory(path string, commitMessage string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pushResults["RemoveDirectory"]
}

func (m *MockGitClient) RemoveAndPush(path string, commitMessage string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.pushResults["RemoveAndPush"]; err != nil {
		return "", err
	}

	commitHash := fmt.Sprintf("commit-%d", len(m.commits))
	m.commits = append(m.commits, commitHash)
	return commitHash, nil
}

func (m *MockGitClient) CommitFiles(ctx context.Context, files map[string]string, message string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	commitHash := fmt.Sprintf("commit-%d", len(m.commits))
	m.commits = append(m.commits, commitHash)

	for path, content := range files {
		m.files[path] = content
	}

	if err := m.pushResults["CommitFiles"]; err != nil {
		return "", err
	}
	return commitHash, nil
}

// Branch operations
func (m *MockGitClient) CreateBranch(ctx context.Context, branchName, baseBranch string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pushResults["CreateBranch"]
}

func (m *MockGitClient) SwitchBranch(ctx context.Context, branchName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pushResults["SwitchBranch"]
}

func (m *MockGitClient) GetCurrentBranch(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.pushResults["GetCurrentBranch"]; err != nil {
		return "", err
	}
	return "main", nil
}

func (m *MockGitClient) ListBranches(ctx context.Context) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.pushResults["ListBranches"]; err != nil {
		return nil, err
	}
	return []string{"main", "develop"}, nil
}

// File operations
func (m *MockGitClient) GetFileContent(ctx context.Context, filePath string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.pushResults["GetFileContent"]; err != nil {
		return "", err
	}
	if content, exists := m.files[filePath]; exists {
		return content, nil
	}
	return "", fmt.Errorf("file not found: %s", filePath)
}

func (m *MockGitClient) DeleteFile(ctx context.Context, filePath, message string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.pushResults["DeleteFile"]; err != nil {
		return "", err
	}
	delete(m.files, filePath)
	commitHash := fmt.Sprintf("commit-%d", len(m.commits))
	m.commits = append(m.commits, commitHash)
	return commitHash, nil
}

// Repository operations
func (m *MockGitClient) Push(ctx context.Context, branch string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pushResults["Push"]
}

func (m *MockGitClient) Pull(ctx context.Context, branch string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pullResults[branch]
}

func (m *MockGitClient) GetStatus(ctx context.Context) (*git.StatusInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.pushResults["GetStatus"]; err != nil {
		return nil, err
	}
	return &git.StatusInfo{Clean: true}, nil
}

func (m *MockGitClient) GetCommitHistory(ctx context.Context, limit int) ([]git.CommitInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.pushResults["GetCommitHistory"]; err != nil {
		return nil, err
	}
	return []git.CommitInfo{}, nil
}

// Pull request operations
func (m *MockGitClient) CreatePullRequest(ctx context.Context, options *git.PullRequestOptions) (*git.PullRequestInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.pushResults["CreatePullRequest"]; err != nil {
		return nil, err
	}
	return &git.PullRequestInfo{Number: 1, URL: "https://github.com/test/repo/pull/1"}, nil
}

// MockLLMClient implements types.ClientInterface for testing
type MockLLMClient struct {
	responses map[string]string
	errors    map[string]error
	mu        sync.RWMutex
}

func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		responses: make(map[string]string),
		errors:    make(map[string]error),
	}
}

func (m *MockLLMClient) ProcessIntent(ctx context.Context, prompt string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if err := m.errors["ProcessIntent"]; err != nil {
		return "", err
	}

	if response := m.responses["ProcessIntent"]; response != "" {
		return response, nil
	}

	// Default response for successful processing
	return "Mock LLM response for intent processing", nil
}

func (m *MockLLMClient) ProcessIntentStream(ctx context.Context, prompt string, chunks chan<- *types.StreamingChunk) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if err := m.errors["ProcessIntentStream"]; err != nil {
		return err
	}

	// Send a mock streaming chunk
	select {
	case chunks <- &types.StreamingChunk{
		Content: "Mock streaming response",
		IsLast:  true,
	}:
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

func (m *MockLLMClient) GetSupportedModels() []string {
	return []string{"mock-model-1", "mock-model-2"}
}

func (m *MockLLMClient) GetModelCapabilities(modelName string) (*types.ModelCapabilities, error) {
	return &types.ModelCapabilities{
		MaxTokens:    4096,
		SupportsChat: true,
	}, nil
}

func (m *MockLLMClient) ValidateModel(modelName string) error {
	if modelName == "invalid-model" {
		return fmt.Errorf("model %s not supported", modelName)
	}
	return nil
}

func (m *MockLLMClient) EstimateTokens(text string) int {
	// Simple estimation: roughly 4 characters per token
	return len(text) / 4
}

func (m *MockLLMClient) GetMaxTokens() int {
	return 4096
}

func (m *MockLLMClient) SupportsStreaming() bool {
	return true
}

func (m *MockLLMClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Clear state on close
	m.responses = make(map[string]string)
	m.errors = make(map[string]error)
	return nil
}

func (m *MockLLMClient) SetResponse(operation string, response string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[operation] = response
}

func (m *MockLLMClient) SetError(operation string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors[operation] = err
}

// MockPackageGenerator implements basic package generation for testing
type MockPackageGenerator struct {
	packages map[string]string
	errors   map[string]error
	mu       sync.RWMutex
}

func NewMockPackageGenerator() *MockPackageGenerator {
	return &MockPackageGenerator{
		packages: make(map[string]string),
		errors:   make(map[string]error),
	}
}

func (m *MockPackageGenerator) GeneratePackage(intentType string, parameters map[string]interface{}) (map[string]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if err := m.errors["GeneratePackage"]; err != nil {
		return nil, err
	}

	// Generate mock package files
	files := map[string]string{
		"Kptfile": `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: ` + intentType + `
pipeline:
  mutators: []
  validators: []`,
		"deployment.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: ` + intentType + `-deployment
spec:
  replicas: ` + fmt.Sprintf("%v", parameters["replicas"]) + `
  selector:
    matchLabels:
      app: ` + intentType + `
  template:
    metadata:
      labels:
        app: ` + intentType + `
    spec:
      containers:
      - name: ` + intentType + `
        image: ` + intentType + `:latest`,
	}

	return files, nil
}

func (m *MockPackageGenerator) SetError(operation string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors[operation] = err
}

// MockMetricsCollector implements metrics collection for testing
type MockMetricsCollector struct {
	metrics map[string]float64
	labels  map[string]map[string]string
	mu      sync.RWMutex
}

func NewMockMetricsCollector() *MockMetricsCollector {
	return &MockMetricsCollector{
		metrics: make(map[string]float64),
		labels:  make(map[string]map[string]string),
	}
}

func (m *MockMetricsCollector) RecordMetric(name string, value float64, labels map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics[name] = value
	m.labels[name] = labels
}

func (m *MockMetricsCollector) GetMetric(name string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metrics[name]
}

func (m *MockMetricsCollector) GetLabels(name string) map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.labels[name]
}

// MockHTTPClient provides HTTP client mocking utilities
type MockHTTPClient struct {
	responses map[string]*http.Response
	errors    map[string]error
	mu        sync.RWMutex
}

func NewMockHTTPClient() *MockHTTPClient {
	return &MockHTTPClient{
		responses: make(map[string]*http.Response),
		errors:    make(map[string]error),
	}
}

// Helper functions for creating mock responses
func CreateMockLLMResponse(intentType string, confidence float64) string {
	return fmt.Sprintf("Mock LLM response for intent type: %s with confidence: %.2f", intentType, confidence)
}

// Performance testing utilities
type PerformanceTracker struct {
	startTimes map[string]time.Time
	durations  map[string]time.Duration
	mu         sync.RWMutex
}

func NewPerformanceTracker() *PerformanceTracker {
	return &PerformanceTracker{
		startTimes: make(map[string]time.Time),
		durations:  make(map[string]time.Duration),
	}
}

func (p *PerformanceTracker) Start(operation string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.startTimes[operation] = time.Now()
}

func (p *PerformanceTracker) Stop(operation string) time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()

	if start, exists := p.startTimes[operation]; exists {
		duration := time.Since(start)
		p.durations[operation] = duration
		delete(p.startTimes, operation)
		return duration
	}
	return 0
}

func (p *PerformanceTracker) GetDuration(operation string) time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.durations[operation]
}

func (p *PerformanceTracker) GetAllDurations() map[string]time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string]time.Duration)
	for k, v := range p.durations {
		result[k] = v
	}
	return result
}

// Concurrent testing utilities
type ConcurrentTestRunner struct {
	workers    int
	operations []func() error
	results    []error
	mu         sync.Mutex
	wg         sync.WaitGroup
}

func NewConcurrentTestRunner(workers int) *ConcurrentTestRunner {
	return &ConcurrentTestRunner{
		workers:    workers,
		operations: make([]func() error, 0),
		results:    make([]error, 0),
	}
}

func (c *ConcurrentTestRunner) AddOperation(op func() error) {
	c.operations = append(c.operations, op)
}

func (c *ConcurrentTestRunner) Run() []error {
	jobs := make(chan func() error, len(c.operations))

	// Start workers
	for i := 0; i < c.workers; i++ {
		c.wg.Add(1)
		go c.worker(jobs)
	}

	// Send jobs
	for _, op := range c.operations {
		jobs <- op
	}
	close(jobs)

	// Wait for completion
	c.wg.Wait()

	return c.results
}

func (c *ConcurrentTestRunner) worker(jobs <-chan func() error) {
	defer c.wg.Done()

	for op := range jobs {
		err := op()
		c.mu.Lock()
		c.results = append(c.results, err)
		c.mu.Unlock()
	}
}

// Memory testing utilities
type MemoryTracker struct {
	initialMem uint64
	peakMem    uint64
	mu         sync.RWMutex
}

func NewMemoryTracker() *MemoryTracker {
	return &MemoryTracker{}
}

func (m *MemoryTracker) Start() {
	// Memory tracking would be implemented using runtime package
	// This is a simplified version for the test framework
}

func (m *MemoryTracker) Stop() uint64 {
	// Return peak memory usage
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.peakMem
}
