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
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
	"github.com/thc1006/nephoran-intent-operator/pkg/telecom"
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

func (m *MockDependencies) GetLLMClient() shared.ClientInterface {
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

func (m *MockGitClient) Clone(ctx context.Context, url, branch, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pushResults["Clone"]
}

func (m *MockGitClient) CommitAndPush(ctx context.Context, repoPath, message string, files map[string]string) (string, error) {
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

func (m *MockGitClient) Pull(ctx context.Context, repoPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pullResults["Pull"]
}

// MockLLMClient implements shared.ClientInterface for testing
type MockLLMClient struct {
	responses map[string]*shared.LLMResponse
	errors    map[string]error
	mu        sync.RWMutex
}

func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		responses: make(map[string]*shared.LLMResponse),
		errors:    make(map[string]error),
	}
}

func (m *MockLLMClient) ProcessIntent(ctx context.Context, intent string, metadata map[string]interface{}) (*shared.LLMResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if err := m.errors["ProcessIntent"]; err != nil {
		return nil, err
	}

	if response := m.responses["ProcessIntent"]; response != nil {
		return response, nil
	}

	// Default response for successful processing
	return &shared.LLMResponse{
		IntentType: "5G-Core-AMF",
		Confidence: 0.95,
		Parameters: map[string]interface{}{
			"replicas":       3,
			"scaling":        true,
			"cpu_request":    "500m",
			"memory_request": "1Gi",
		},
		Manifests: map[string]interface{}{
			"deployment": map[string]interface{}{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]interface{}{
					"name": "amf-deployment",
				},
				"spec": map[string]interface{}{
					"replicas": 3,
				},
			},
		},
		ProcessingTime: 1500,
		TokensUsed:     250,
		Model:          "gpt-4o-mini",
	}, nil
}

func (m *MockLLMClient) SetResponse(operation string, response *shared.LLMResponse) {
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
func CreateMockLLMResponse(intentType string, confidence float64) *shared.LLMResponse {
	return &shared.LLMResponse{
		IntentType: intentType,
		Confidence: confidence,
		Parameters: map[string]interface{}{
			"replicas":       3,
			"scaling":        true,
			"cpu_request":    "500m",
			"memory_request": "1Gi",
		},
		Manifests: map[string]interface{}{
			"deployment": map[string]interface{}{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]interface{}{
					"name": intentType + "-deployment",
				},
			},
		},
		ProcessingTime: 1500,
		TokensUsed:     250,
		Model:          "gpt-4o-mini",
	}
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
