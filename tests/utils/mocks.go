// Package testutils provides mock implementations and test utilities for the Nephoran operator.

package testutils

import (
	
	"encoding/json"
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

func (m *MockDependencies) GetMetricsCollector() monitoring.MetricsCollector {
	m.incrementCallCount("GetMetricsCollector")

	return m.metricsCollector
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

	Parameters json.RawMessage `json:"parameters"`

	Manifests json.RawMessage `json:"manifests"`

	ProcessingTime int64 `json:"processing_time"`

	TokensUsed int `json:"tokens_used"`

	Model string `json:"model"`
}

// MockLLMClient implements shared.ClientInterface for testing.
type MockLLMClient struct {
	responses map[string]*LLMResponse
	errors    map[string]error
	mu        sync.RWMutex
}

// Compile-time interface compliance check
var _ shared.ClientInterface = (*MockLLMClient)(nil)

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

// Legacy method for backward compatibility - returns model capabilities with error
func (m *MockLLMClient) GetModelCapabilitiesLegacy(modelName string) (*shared.ModelCapabilities, error) {
	caps := m.GetModelCapabilities()
	return &caps, nil
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

// GetEndpoint implements shared.ClientInterface.GetEndpoint
func (m *MockLLMClient) GetEndpoint() string {
	return "https://mock-llm-endpoint.local"
}

// ProcessRequest implements shared.ClientInterface.ProcessRequest
func (m *MockLLMClient) ProcessRequest(ctx context.Context, request *shared.LLMRequest) (*shared.LLMResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if err := m.errors["ProcessRequest"]; err != nil {
		return nil, err
	}

	// Return a mock response
	return &shared.LLMResponse{
		ID:      "mock-response-123",
		Content: "Mock LLM response content",
		Model:   request.Model,
		Usage: shared.TokenUsage{
			PromptTokens:     100,
			CompletionTokens: 150,
			TotalTokens:      250,
		},
		Created: time.Now(),
	}, nil
}

// ProcessStreamingRequest implements shared.ClientInterface.ProcessStreamingRequest
func (m *MockLLMClient) ProcessStreamingRequest(ctx context.Context, request *shared.LLMRequest) (<-chan *shared.StreamingChunk, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if err := m.errors["ProcessStreamingRequest"]; err != nil {
		return nil, err
	}

	chunks := make(chan *shared.StreamingChunk, 2)

	go func() {
		defer close(chunks)
		chunks <- &shared.StreamingChunk{
			ID:        "mock-chunk-1",
			Content:   "Mock streaming chunk 1",
			Delta:     "Mock streaming chunk 1",
			Done:      false,
			Timestamp: time.Now(),
		}
		chunks <- &shared.StreamingChunk{
			ID:        "mock-chunk-2",
			Content:   "Mock streaming chunk 2",
			Delta:     " chunk 2",
			Done:      true,
			IsLast:    true,
			Timestamp: time.Now(),
		}
	}()

	return chunks, nil
}

// HealthCheck performs a health check on the mock client.

func (m *MockLLMClient) HealthCheck(ctx context.Context) error {
	m.mu.RLock()

	defer m.mu.RUnlock()

	return m.errors["HealthCheck"]
}

// GetStatus returns the status of the mock client.

func (m *MockLLMClient) GetStatus() shared.ClientStatus {
	return shared.ClientStatusHealthy
}

// GetModelCapabilities implements shared.ClientInterface.GetModelCapabilities
func (m *MockLLMClient) GetModelCapabilities() shared.ModelCapabilities {
	return shared.ModelCapabilities{
		SupportsStreaming:    true,
		SupportsSystemPrompt: true,
		SupportsChatFormat:   true,
		SupportsChat:         true,
		SupportsFunction:     true,
		MaxTokens:            4096,
		CostPerToken:         0.0001,
		SupportedMimeTypes:   []string{"text/plain", "application/json"},
		ModelVersion:         "1.0.0",
		Features:             make(map[string]interface{}),
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

	started bool
}

// NewMockMetricsCollector performs newmockmetricscollector operation.

func NewMockMetricsCollector() *MockMetricsCollector {
	return &MockMetricsCollector{
		metrics: make(map[string]float64),

		labels: make(map[string]map[string]string),

		started: false,
	}
}

// CollectMetrics implements MetricsCollector interface.

func (m *MockMetricsCollector) CollectMetrics() ([]*monitoring.Metric, error) {
	m.mu.RLock()

	defer m.mu.RUnlock()

	metrics := make([]*monitoring.Metric, 0, len(m.metrics))

	for name, value := range m.metrics {
		metrics = append(metrics, &monitoring.Metric{
			Name: name,

			Type: monitoring.MetricTypeGauge,

			Value: value,

			Labels: m.labels[name],

			Timestamp: time.Now(),
		})
	}

	return metrics, nil
}

// Start implements MetricsCollector interface.

func (m *MockMetricsCollector) Start() error {
	m.mu.Lock()

	defer m.mu.Unlock()

	m.started = true

	return nil
}

// Stop implements MetricsCollector interface.

func (m *MockMetricsCollector) Stop() error {
	m.mu.Lock()

	defer m.mu.Unlock()

	m.started = false

	return nil
}

// RegisterMetrics implements MetricsCollector interface.

func (m *MockMetricsCollector) RegisterMetrics(registry interface{}) error {
	return nil // No-op for mock
}

// UpdateControllerHealth implements MetricsCollector interface.

func (m *MockMetricsCollector) UpdateControllerHealth(controllerName, component string, healthy bool) {
	value := 0.0

	if healthy {
		value = 1.0
	}

	m.RecordMetric("controller_health", value, map[string]string{
		"controller": controllerName,

		"component": component,
	})
}

// RecordKubernetesAPILatency implements MetricsCollector interface.

func (m *MockMetricsCollector) RecordKubernetesAPILatency(latency time.Duration) {
	m.RecordMetric("kubernetes_api_latency_seconds", latency.Seconds(), nil)
}

// UpdateNetworkIntentStatus implements MetricsCollector interface.

func (m *MockMetricsCollector) UpdateNetworkIntentStatus(name, namespace, intentType, status string) {
	m.RecordMetric("network_intent_status", 1.0, map[string]string{
		"name": name,

		"namespace": namespace,

		"intent_type": intentType,

		"status": status,
	})
}

// RecordNetworkIntentProcessed implements MetricsCollector interface.

func (m *MockMetricsCollector) RecordNetworkIntentProcessed(intentType, status string, duration time.Duration) {
	m.RecordMetric("network_intent_processed_total", 1.0, map[string]string{
		"intent_type": intentType,

		"status": status,
	})

	m.RecordMetric("network_intent_processing_duration_seconds", duration.Seconds(), map[string]string{
		"intent_type": intentType,

		"status": status,
	})
}

// RecordNetworkIntentRetry implements MetricsCollector interface.

func (m *MockMetricsCollector) RecordNetworkIntentRetry(name, namespace, reason string) {
	m.RecordMetric("network_intent_retries_total", 1.0, map[string]string{
		"name": name,

		"namespace": namespace,

		"reason": reason,
	})
}

// RecordLLMRequest implements MetricsCollector interface.

func (m *MockMetricsCollector) RecordLLMRequest(model, status string, duration time.Duration, tokensUsed int) {
	m.RecordMetric("llm_requests_total", 1.0, map[string]string{
		"model": model,

		"status": status,
	})

	m.RecordMetric("llm_request_duration_seconds", duration.Seconds(), map[string]string{
		"model": model,

		"status": status,
	})

	m.RecordMetric("llm_tokens_used_total", float64(tokensUsed), map[string]string{
		"model": model,
	})
}

// RecordE2NodeSetOperation implements MetricsCollector interface.

func (m *MockMetricsCollector) RecordE2NodeSetOperation(operation string, duration time.Duration) {
	m.RecordMetric("e2nodeset_operations_total", 1.0, map[string]string{
		"operation": operation,
	})

	m.RecordMetric("e2nodeset_operation_duration_seconds", duration.Seconds(), map[string]string{
		"operation": operation,
	})
}

// UpdateE2NodeSetReplicas implements MetricsCollector interface.

func (m *MockMetricsCollector) UpdateE2NodeSetReplicas(name, namespace, status string, count int) {
	m.RecordMetric("e2nodeset_replicas", float64(count), map[string]string{
		"name": name,

		"namespace": namespace,

		"status": status,
	})
}

// RecordE2NodeSetScaling implements MetricsCollector interface.

func (m *MockMetricsCollector) RecordE2NodeSetScaling(name, namespace, direction string) {
	m.RecordMetric("e2nodeset_scaling_events_total", 1.0, map[string]string{
		"name": name,

		"namespace": namespace,

		"direction": direction,
	})
}

// RecordORANInterfaceRequest implements MetricsCollector interface.

func (m *MockMetricsCollector) RecordORANInterfaceRequest(interfaceType, operation, status string, duration time.Duration) {
	m.RecordMetric("oran_interface_requests_total", 1.0, map[string]string{
		"interface_type": interfaceType,

		"operation": operation,

		"status": status,
	})

	m.RecordMetric("oran_interface_request_duration_seconds", duration.Seconds(), map[string]string{
		"interface_type": interfaceType,

		"operation": operation,

		"status": status,
	})
}

// RecordORANInterfaceError implements MetricsCollector interface.

func (m *MockMetricsCollector) RecordORANInterfaceError(interfaceType, operation, errorType string) {
	m.RecordMetric("oran_interface_errors_total", 1.0, map[string]string{
		"interface_type": interfaceType,

		"operation": operation,

		"error_type": errorType,
	})
}

// UpdateORANConnectionStatus implements MetricsCollector interface.

func (m *MockMetricsCollector) UpdateORANConnectionStatus(interfaceType, endpoint string, connected bool) {
	value := 0.0

	if connected {
		value = 1.0
	}

	m.RecordMetric("oran_connection_status", value, map[string]string{
		"interface_type": interfaceType,

		"endpoint": endpoint,
	})
}

// UpdateORANPolicyInstances implements MetricsCollector interface.

func (m *MockMetricsCollector) UpdateORANPolicyInstances(policyType, status string, count int) {
	m.RecordMetric("oran_policy_instances", float64(count), map[string]string{
		"policy_type": policyType,

		"status": status,
	})
}

// RecordRAGOperation implements MetricsCollector interface.

func (m *MockMetricsCollector) RecordRAGOperation(duration time.Duration, cacheHit bool) {
	cacheStatus := "miss"

	if cacheHit {
		cacheStatus = "hit"
	}

	m.RecordMetric("rag_operations_total", 1.0, map[string]string{
		"cache_status": cacheStatus,
	})

	m.RecordMetric("rag_operation_duration_seconds", duration.Seconds(), map[string]string{
		"cache_status": cacheStatus,
	})
}

// UpdateRAGDocumentsIndexed implements MetricsCollector interface.

func (m *MockMetricsCollector) UpdateRAGDocumentsIndexed(count int) {
	m.RecordMetric("rag_documents_indexed_total", float64(count), nil)
}

// RecordGitOpsOperation implements MetricsCollector interface.

func (m *MockMetricsCollector) RecordGitOpsOperation(operation string, duration time.Duration, success bool) {
	status := "failure"

	if success {
		status = "success"
	}

	m.RecordMetric("gitops_operations_total", 1.0, map[string]string{
		"operation": operation,

		"status": status,
	})

	m.RecordMetric("gitops_operation_duration_seconds", duration.Seconds(), map[string]string{
		"operation": operation,

		"status": status,
	})
}

// UpdateGitOpsSyncStatus implements MetricsCollector interface.

func (m *MockMetricsCollector) UpdateGitOpsSyncStatus(repository, branch string, inSync bool) {
	value := 0.0

	if inSync {
		value = 1.0
	}

	m.RecordMetric("gitops_sync_status", value, map[string]string{
		"repository": repository,

		"branch": branch,
	})
}

// UpdateResourceUtilization implements MetricsCollector interface.

func (m *MockMetricsCollector) UpdateResourceUtilization(resourceType, unit string, value float64) {
	m.RecordMetric("resource_utilization", value, map[string]string{
		"resource_type": resourceType,

		"unit": unit,
	})
}

// UpdateWorkerQueueMetrics implements MetricsCollector interface.

func (m *MockMetricsCollector) UpdateWorkerQueueMetrics(queueName string, depth int, latency time.Duration) {
	m.RecordMetric("worker_queue_depth", float64(depth), map[string]string{
		"queue_name": queueName,
	})

	m.RecordMetric("worker_queue_latency_seconds", latency.Seconds(), map[string]string{
		"queue_name": queueName,
	})
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

// GetCounter returns a counter interface
func (m *MockMetricsCollector) GetCounter(name string) interface{} {
	return m.GetMetric(name)
}

// GetGauge returns a gauge interface
func (m *MockMetricsCollector) GetGauge(name string) interface{} {
	return m.GetMetric(name)
}

// GetHistogram returns a histogram interface
func (m *MockMetricsCollector) GetHistogram(name string) interface{} {
	return m.GetMetric(name)
}

// RecordCNFDeployment records CNF deployment metrics
func (m *MockMetricsCollector) RecordCNFDeployment(functionName string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.metrics == nil {
		m.metrics = make(map[string]float64)
	}

	key := fmt.Sprintf("cnf_deployment_duration_%s", functionName)
	m.metrics[key] = duration.Seconds()
}

// RecordHTTPRequest records HTTP request metrics
func (m *MockMetricsCollector) RecordHTTPRequest(method, endpoint, status string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.metrics == nil {
		m.metrics = make(map[string]float64)
	}

	key := fmt.Sprintf("http_request_duration_%s_%s_%s", method, endpoint, status)
	m.metrics[key] = duration.Seconds()
}

// RecordSSEStream records server-sent events stream metrics
func (m *MockMetricsCollector) RecordSSEStream(endpoint string, connected bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.metrics == nil {
		m.metrics = make(map[string]float64)
	}

	key := fmt.Sprintf("sse_stream_connection_%s", endpoint)
	if connected {
		m.metrics[key] = 1.0
	} else {
		m.metrics[key] = 0.0
	}
}

// RecordLLMRequestError records LLM request error metrics
func (m *MockMetricsCollector) RecordLLMRequestError(model, errorType string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.metrics == nil {
		m.metrics = make(map[string]float64)
	}

	key := fmt.Sprintf("llm_request_errors_%s_%s", model, errorType)
	m.metrics[key] = m.metrics[key] + 1.0
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

		Parameters: json.RawMessage(`{}`),

		Manifests: json.RawMessage(`{
			"apiVersion": "apps/v1",
			"kind": "Deployment", 
			"metadata": {}
		}`),

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

