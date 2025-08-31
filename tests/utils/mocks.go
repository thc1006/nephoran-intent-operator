// Package testutils provides mock implementations and test utilities for the Nephoran operator.

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

// MockDependencies implements the Dependencies interface for testing.

type MockDependencies struct {
	gitClient *MockGitClient

	llmClient *MockLLMClient

	packageGenerator *MockPackageGenerator

	httpClient *http.Client

	eventRecorder *record.FakeRecorder

	telecomKnowledgeBase *telecom.TelecomKnowledgeBase

	metricsCollector *MockMetricsCollector

	// Control behavior.

	simulateErrors map[string]error

	operationDelays map[string]time.Duration

	callCounts map[string]int

	mu sync.RWMutex
}

// NewMockDependencies creates a new mock dependencies instance.

func NewMockDependencies() *MockDependencies {

	return &MockDependencies{

		gitClient: NewMockGitClient(),

		llmClient: NewMockLLMClient(),

		packageGenerator: NewMockPackageGenerator(),

		httpClient: &http.Client{Timeout: 30 * time.Second},

		eventRecorder: record.NewFakeRecorder(100),

		metricsCollector: NewMockMetricsCollector(),

		simulateErrors: make(map[string]error),

		operationDelays: make(map[string]time.Duration),

		callCounts: make(map[string]int),
	}

}

// GetGitClient returns the mock Git client interface.

func (m *MockDependencies) GetGitClient() git.ClientInterface {

	m.incrementCallCount("GetGitClient")

	return m.gitClient

}

// GetLLMClient returns the mock LLM client interface.

func (m *MockDependencies) GetLLMClient() shared.ClientInterface {

	m.incrementCallCount("GetLLMClient")

	return m.llmClient

}

// GetPackageGenerator returns a mock Nephio package generator.

func (m *MockDependencies) GetPackageGenerator() *nephio.PackageGenerator {

	m.incrementCallCount("GetPackageGenerator")

	return &nephio.PackageGenerator{} // Return a real instance for simplicity

}

// GetHTTPClient performs gethttpclient operation.

func (m *MockDependencies) GetHTTPClient() *http.Client {

	m.incrementCallCount("GetHTTPClient")

	return m.httpClient

}

// GetEventRecorder performs geteventrecorder operation.

func (m *MockDependencies) GetEventRecorder() record.EventRecorder {

	m.incrementCallCount("GetEventRecorder")

	return m.eventRecorder

}

// GetTelecomKnowledgeBase performs gettelecomknowledgebase operation.

func (m *MockDependencies) GetTelecomKnowledgeBase() *telecom.TelecomKnowledgeBase {

	m.incrementCallCount("GetTelecomKnowledgeBase")

	if m.telecomKnowledgeBase == nil {

		m.telecomKnowledgeBase = &telecom.TelecomKnowledgeBase{}

	}

	return m.telecomKnowledgeBase

}

// GetMetricsCollector performs getmetricscollector operation.

func (m *MockDependencies) GetMetricsCollector() *monitoring.MetricsCollector {

	m.incrementCallCount("GetMetricsCollector")

	return &monitoring.MetricsCollector{} // Return a real instance for simplicity

}

// Control methods.

func (m *MockDependencies) SetError(operation string, err error) {

	m.mu.Lock()

	defer m.mu.Unlock()

	m.simulateErrors[operation] = err

}

// SetDelay performs setdelay operation.

func (m *MockDependencies) SetDelay(operation string, delay time.Duration) {

	m.mu.Lock()

	defer m.mu.Unlock()

	m.operationDelays[operation] = delay

}

// GetCallCount performs getcallcount operation.

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

// MockGitClient implements git.ClientInterface for testing.

type MockGitClient struct {
	commits []string

	pushResults map[string]error

	pullResults map[string]error

	files map[string]string

	mu sync.RWMutex
}

// NewMockGitClient performs newmockgitclient operation.

func NewMockGitClient() *MockGitClient {

	return &MockGitClient{

		commits: make([]string, 0),

		pushResults: make(map[string]error),

		pullResults: make(map[string]error),

		files: make(map[string]string),
	}

}

// Clone performs clone operation.

func (m *MockGitClient) Clone(ctx context.Context, url, branch, path string) error {

	m.mu.Lock()

	defer m.mu.Unlock()

	return m.pushResults["Clone"]

}

// CommitAndPush performs commitandpush operation.

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

// Pull performs pull operation.

func (m *MockGitClient) Pull(ctx context.Context, repoPath string) error {

	m.mu.Lock()

	defer m.mu.Unlock()

	return m.pullResults["Pull"]

}

// CommitAndPushChanges performs commitandpushchanges operation.

func (m *MockGitClient) CommitAndPushChanges(message string) error {

	m.mu.Lock()

	defer m.mu.Unlock()

	commitHash := fmt.Sprintf("commit-%d", len(m.commits))

	m.commits = append(m.commits, commitHash)

	if err := m.pushResults["CommitAndPushChanges"]; err != nil {

		return err

	}

	return nil

}

// InitRepo performs initrepo operation.

func (m *MockGitClient) InitRepo() error {

	m.mu.Lock()

	defer m.mu.Unlock()

	return m.pushResults["InitRepo"]

}

// RemoveDirectory performs removedirectory operation.

func (m *MockGitClient) RemoveDirectory(path, commitMessage string) error {

	m.mu.Lock()

	defer m.mu.Unlock()

	return m.pushResults["RemoveDirectory"]

}

// CommitFiles performs commitfiles operation.

func (m *MockGitClient) CommitFiles(files []string, msg string) error {

	m.mu.Lock()

	defer m.mu.Unlock()

	commitHash := fmt.Sprintf("commit-%d", len(m.commits))

	m.commits = append(m.commits, commitHash)

	if err := m.pushResults["CommitFiles"]; err != nil {

		return err

	}

	return nil

}

// CreateBranch performs createbranch operation.

func (m *MockGitClient) CreateBranch(name string) error {

	m.mu.Lock()

	defer m.mu.Unlock()

	return m.pushResults["CreateBranch"]

}

// SwitchBranch performs switchbranch operation.

func (m *MockGitClient) SwitchBranch(name string) error {

	m.mu.Lock()

	defer m.mu.Unlock()

	return m.pushResults["SwitchBranch"]

}

// GetCurrentBranch performs getcurrentbranch operation.

func (m *MockGitClient) GetCurrentBranch() (string, error) {

	m.mu.Lock()

	defer m.mu.Unlock()

	if err := m.pushResults["GetCurrentBranch"]; err != nil {

		return "", err

	}

	return "main", nil

}

// ListBranches performs listbranches operation.

func (m *MockGitClient) ListBranches() ([]string, error) {

	m.mu.Lock()

	defer m.mu.Unlock()

	if err := m.pushResults["ListBranches"]; err != nil {

		return nil, err

	}

	return []string{"main", "dev"}, nil

}

// GetFileContent performs getfilecontent operation.

func (m *MockGitClient) GetFileContent(path string) ([]byte, error) {

	m.mu.Lock()

	defer m.mu.Unlock()

	if err := m.pushResults["GetFileContent"]; err != nil {

		return nil, err

	}

	if content, exists := m.files[path]; exists {

		return []byte(content), nil

	}

	return []byte("mock content"), nil

}

// LLMResponse represents a response from the LLM processor.

type LLMResponse struct {
	IntentType string `json:"intent_type"`

	Confidence float64 `json:"confidence"`

	Parameters map[string]interface{} `json:"parameters"`

	Manifests map[string]interface{} `json:"manifests"`

	ProcessingTime int64 `json:"processing_time"`

	TokensUsed int `json:"tokens_used"`

	Model string `json:"model"`
}

// MockLLMClient implements shared.ClientInterface for testing.

type MockLLMClient struct {
	responses map[string]*LLMResponse

	errors map[string]error

	mu sync.RWMutex
}

// NewMockLLMClient performs newmockllmclient operation.

func NewMockLLMClient() *MockLLMClient {

	return &MockLLMClient{

		responses: make(map[string]*LLMResponse),

		errors: make(map[string]error),
	}

}

// ProcessIntent performs processintent operation.

func (m *MockLLMClient) ProcessIntent(ctx context.Context, prompt string) (string, error) {

	m.mu.RLock()

	defer m.mu.RUnlock()

	if err := m.errors["ProcessIntent"]; err != nil {

		return "", err

	}

	if response := m.responses["ProcessIntent"]; response != nil {

		return fmt.Sprintf("Intent: %s, Confidence: %.2f", response.IntentType, response.Confidence), nil

	}

	// Default response for successful processing.

	return "Intent processed successfully: 5G-Core-AMF with 95% confidence", nil

}

// ProcessIntentStream performs processintentstream operation.

func (m *MockLLMClient) ProcessIntentStream(ctx context.Context, prompt string, chunks chan<- *shared.StreamingChunk) error {

	m.mu.RLock()

	defer m.mu.RUnlock()

	if err := m.errors["ProcessIntentStream"]; err != nil {

		return err

	}

	// Send mock streaming chunks.

	go func() {

		defer close(chunks)

		chunks <- &shared.StreamingChunk{Content: "Processing intent...", IsLast: false}

		chunks <- &shared.StreamingChunk{Content: "Intent processed successfully", IsLast: true}

	}()

	return nil

}

// GetSupportedModels performs getsupportedmodels operation.

func (m *MockLLMClient) GetSupportedModels() []string {

	return []string{"gpt-4o-mini", "gpt-4o", "claude-3-haiku"}

}

// GetModelCapabilities performs getmodelcapabilities operation.

func (m *MockLLMClient) GetModelCapabilities(modelName string) (*shared.ModelCapabilities, error) {

	return &shared.ModelCapabilities{

		MaxTokens: 4096,

		SupportsChat: true,

		SupportsFunction: true,

		SupportsStreaming: true,

		CostPerToken: 0.0001,

		Features: make(map[string]interface{}),
	}, nil

}

// ValidateModel performs validatemodel operation.

func (m *MockLLMClient) ValidateModel(modelName string) error {

	supportedModels := m.GetSupportedModels()

	for _, model := range supportedModels {

		if model == modelName {

			return nil

		}

	}

	return fmt.Errorf("unsupported model: %s", modelName)

}

// EstimateTokens performs estimatetokens operation.

func (m *MockLLMClient) EstimateTokens(text string) int {

	// Simple estimation: roughly 4 characters per token.

	return len(text) / 4

}

// GetMaxTokens performs getmaxtokens operation.

func (m *MockLLMClient) GetMaxTokens(modelName string) int {

	switch modelName {

	case "gpt-4o-mini":

		return 128000

	case "gpt-4o":

		return 128000

	case "claude-3-haiku":

		return 200000

	default:

		return 4096

	}

}

// Close performs close operation.

func (m *MockLLMClient) Close() error {

	m.mu.Lock()

	defer m.mu.Unlock()

	// Clean up resources.

	m.responses = make(map[string]*LLMResponse)

	m.errors = make(map[string]error)

	return nil

}

// SetResponse performs setresponse operation.

func (m *MockLLMClient) SetResponse(operation string, response *LLMResponse) {

	m.mu.Lock()

	defer m.mu.Unlock()

	m.responses[operation] = response

}

// SetError performs seterror operation.

func (m *MockLLMClient) SetError(operation string, err error) {

	m.mu.Lock()

	defer m.mu.Unlock()

	m.errors[operation] = err

}

// MockPackageGenerator implements basic package generation for testing.

type MockPackageGenerator struct {
	packages map[string]string

	errors map[string]error

	mu sync.RWMutex
}

// NewMockPackageGenerator performs newmockpackagegenerator operation.

func NewMockPackageGenerator() *MockPackageGenerator {

	return &MockPackageGenerator{

		packages: make(map[string]string),

		errors: make(map[string]error),
	}

}

// GeneratePackage performs generatepackage operation.

func (m *MockPackageGenerator) GeneratePackage(intentType string, parameters map[string]interface{}) (map[string]string, error) {

	m.mu.RLock()

	defer m.mu.RUnlock()

	if err := m.errors["GeneratePackage"]; err != nil {

		return nil, err

	}

	// Generate mock package files.

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

// SetError performs seterror operation.

func (m *MockPackageGenerator) SetError(operation string, err error) {

	m.mu.Lock()

	defer m.mu.Unlock()

	m.errors[operation] = err

}

// MockMetricsCollector implements metrics collection for testing.

type MockMetricsCollector struct {
	metrics map[string]float64

	labels map[string]map[string]string

	mu sync.RWMutex
}

// NewMockMetricsCollector performs newmockmetricscollector operation.

func NewMockMetricsCollector() *MockMetricsCollector {

	return &MockMetricsCollector{

		metrics: make(map[string]float64),

		labels: make(map[string]map[string]string),
	}

}

// RecordMetric performs recordmetric operation.

func (m *MockMetricsCollector) RecordMetric(name string, value float64, labels map[string]string) {

	m.mu.Lock()

	defer m.mu.Unlock()

	m.metrics[name] = value

	m.labels[name] = labels

}

// GetMetric performs getmetric operation.

func (m *MockMetricsCollector) GetMetric(name string) float64 {

	m.mu.RLock()

	defer m.mu.RUnlock()

	return m.metrics[name]

}

// GetLabels performs getlabels operation.

func (m *MockMetricsCollector) GetLabels(name string) map[string]string {

	m.mu.RLock()

	defer m.mu.RUnlock()

	return m.labels[name]

}

// MockHTTPClient provides HTTP client mocking utilities.

type MockHTTPClient struct {
	responses map[string]*http.Response

	errors map[string]error

	mu sync.RWMutex
}

// NewMockHTTPClient performs newmockhttpclient operation.

func NewMockHTTPClient() *MockHTTPClient {

	return &MockHTTPClient{

		responses: make(map[string]*http.Response),

		errors: make(map[string]error),
	}

}

// Helper functions for creating mock responses.

func CreateMockLLMResponse(intentType string, confidence float64) *LLMResponse {

	return &LLMResponse{

		IntentType: intentType,

		Confidence: confidence,

		Parameters: map[string]interface{}{

			"replicas": 3,

			"scaling": true,

			"cpu_request": "500m",

			"memory_request": "1Gi",
		},

		Manifests: map[string]interface{}{

			"deployment": map[string]interface{}{

				"apiVersion": "apps/v1",

				"kind": "Deployment",

				"metadata": map[string]interface{}{

					"name": intentType + "-deployment",
				},
			},
		},

		ProcessingTime: 1500,

		TokensUsed: 250,

		Model: "gpt-4o-mini",
	}

}

// Performance testing utilities.

type PerformanceTracker struct {
	startTimes map[string]time.Time

	durations map[string]time.Duration

	mu sync.RWMutex
}

// NewPerformanceTracker performs newperformancetracker operation.

func NewPerformanceTracker() *PerformanceTracker {

	return &PerformanceTracker{

		startTimes: make(map[string]time.Time),

		durations: make(map[string]time.Duration),
	}

}

// Start performs start operation.

func (p *PerformanceTracker) Start(operation string) {

	p.mu.Lock()

	defer p.mu.Unlock()

	p.startTimes[operation] = time.Now()

}

// Stop performs stop operation.

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

// GetDuration performs getduration operation.

func (p *PerformanceTracker) GetDuration(operation string) time.Duration {

	p.mu.RLock()

	defer p.mu.RUnlock()

	return p.durations[operation]

}

// GetAllDurations performs getalldurations operation.

func (p *PerformanceTracker) GetAllDurations() map[string]time.Duration {

	p.mu.RLock()

	defer p.mu.RUnlock()

	result := make(map[string]time.Duration)

	for k, v := range p.durations {

		result[k] = v

	}

	return result

}

// Concurrent testing utilities.

type ConcurrentTestRunner struct {
	workers int

	operations []func() error

	results []error

	mu sync.Mutex

	wg sync.WaitGroup
}

// NewConcurrentTestRunner performs newconcurrenttestrunner operation.

func NewConcurrentTestRunner(workers int) *ConcurrentTestRunner {

	return &ConcurrentTestRunner{

		workers: workers,

		operations: make([]func() error, 0),

		results: make([]error, 0),
	}

}

// AddOperation performs addoperation operation.

func (c *ConcurrentTestRunner) AddOperation(op func() error) {

	c.operations = append(c.operations, op)

}

// Run performs run operation.

func (c *ConcurrentTestRunner) Run() []error {

	jobs := make(chan func() error, len(c.operations))

	// Start workers.

	for range c.workers {

		c.wg.Add(1)

		go c.worker(jobs)

	}

	// Send jobs.

	for _, op := range c.operations {

		jobs <- op

	}

	close(jobs)

	// Wait for completion.

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

// Memory testing utilities.

type MemoryTracker struct {
	initialMem uint64

	peakMem uint64

	mu sync.RWMutex
}

// NewMemoryTracker performs newmemorytracker operation.

func NewMemoryTracker() *MemoryTracker {

	return &MemoryTracker{}

}

// Start performs start operation.

func (m *MemoryTracker) Start() {

	// Memory tracking would be implemented using runtime package.

	// This is a simplified version for the test framework.

}

// Stop performs stop operation.

func (m *MemoryTracker) Stop() uint64 {

	// Return peak memory usage.

	m.mu.RLock()

	defer m.mu.RUnlock()

	return m.peakMem

}
