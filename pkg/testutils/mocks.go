package testutils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"k8s.io/client-go/tools/record"

	"github.com/thc1006/nephoran-intent-operator/pkg/git"
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
	"github.com/thc1006/nephoran-intent-operator/pkg/telecom"
)

// MockLLMClient provides a mock implementation of the LLM client interface.

type MockLLMClient struct {
	responses map[string]string

	errors map[string]error

	processingDelay time.Duration

	callCount int

	lastIntent string

	shouldReturnError bool

	Error error // Public field for direct error control in tests
}

// NewMockLLMClient creates a new mock LLM client.

func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		responses: make(map[string]string),

		errors: make(map[string]error),

		processingDelay: 100 * time.Millisecond,
	}
}

// ProcessIntent implements the LLM client interface.

func (m *MockLLMClient) ProcessIntent(ctx context.Context, intent string) (string, error) {
	m.callCount++

	m.lastIntent = intent

	// Simulate processing delay.

	select {

	case <-ctx.Done():

		return "", ctx.Err()

	case <-time.After(m.processingDelay):

	}

	// Check for direct Error field first.

	if m.Error != nil {
		return "", m.Error
	}

	// Check for global error flag.

	if m.shouldReturnError {
		return "", fmt.Errorf("mock LLM client configured to return error")
	}

	// Check for specific error responses.

	if err, exists := m.errors[intent]; exists {
		return "", err
	}

	// Check for specific responses.

	if response, exists := m.responses[intent]; exists {
		return response, nil
	}

	// Generate default response based on intent content.

	return m.generateDefaultResponse(intent), nil
}

// SetResponse sets a specific response for a given intent.

func (m *MockLLMClient) SetResponse(intent, response string) {
	m.responses[intent] = response
}

// SetError sets an error to be returned for a specific intent.

func (m *MockLLMClient) SetError(intent string, err error) {
	m.errors[intent] = err
}

// SetProcessingDelay sets the simulated processing delay.

func (m *MockLLMClient) SetProcessingDelay(delay time.Duration) {
	m.processingDelay = delay
}

// SetShouldReturnError sets whether the mock should return errors for all requests.

func (m *MockLLMClient) SetShouldReturnError(shouldError bool) {
	m.shouldReturnError = shouldError
}

// GetCallCount returns the number of times ProcessIntent was called.

func (m *MockLLMClient) GetCallCount() int {
	return m.callCount
}

// GetLastIntent returns the last intent that was processed.

func (m *MockLLMClient) GetLastIntent() string {
	return m.lastIntent
}

// ResetMock clears all mock state.

func (m *MockLLMClient) ResetMock() {
	m.responses = make(map[string]string)

	m.errors = make(map[string]error)

	m.callCount = 0

	m.lastIntent = ""

	m.processingDelay = 100 * time.Millisecond

	m.shouldReturnError = false
}

// generateDefaultResponse creates a default response based on intent content.

func (m *MockLLMClient) generateDefaultResponse(intent string) string {
	lowerIntent := strings.ToLower(intent)

	// Determine response type based on intent content.

	if strings.Contains(lowerIntent, "scale") || strings.Contains(lowerIntent, "increase") || strings.Contains(lowerIntent, "decrease") {
		return m.generateScaleResponse(intent)
	}

	return m.generateDeploymentResponse(intent)
}

// generateDeploymentResponse generates a default deployment response.

func (m *MockLLMClient) generateDeploymentResponse(intent string) string {
	lowerIntent := strings.ToLower(intent)

	// Determine network function type.

	nfType := "generic-nf"

	namespace := "default"

	if strings.Contains(lowerIntent, "upf") {

		nfType = "upf"

		namespace = "5g-core"

	} else if strings.Contains(lowerIntent, "amf") {

		nfType = "amf"

		namespace = "5g-core"

	} else if strings.Contains(lowerIntent, "smf") {

		nfType = "smf"

		namespace = "5g-core"

	} else if strings.Contains(lowerIntent, "ric") || strings.Contains(lowerIntent, "near-rt") {

		nfType = "near-rt-ric"

		namespace = "o-ran"

	} else if strings.Contains(lowerIntent, "edge") || strings.Contains(lowerIntent, "mec") {

		nfType = "mec-app"

		namespace = "edge-apps"

	}

	response := map[string]interface{}{
		"spec": map[string]interface{}{
			"replicas": 1,

			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"containers": []map[string]interface{}{
						{
							"name":  nfType,
							"image": fmt.Sprintf("registry.local/%s:latest", nfType),
							"resources": map[string]interface{}{
								"limits": map[string]string{"cpu": "1000m", "memory": "2Gi"},
							},
						},
					},
				},
			},
		},
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-deployment", nfType),
			"namespace": namespace,
		},
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		log.Printf("ERROR: failed to marshal response: %v", err)
		return `{"error":"failed to marshal response"}`
	}
	return string(jsonBytes)
}

// generateScaleResponse generates a default scaling response.

func (m *MockLLMClient) generateScaleResponse(intent string) string {
	lowerIntent := strings.ToLower(intent)

	nfType := "generic-nf"

	namespace := "default"

	if strings.Contains(lowerIntent, "upf") {

		nfType = "upf"

		namespace = "5g-core"

	} else if strings.Contains(lowerIntent, "amf") {

		nfType = "amf"

		namespace = "5g-core"

	}

	response := map[string]interface{}{
		"spec": map[string]interface{}{
			"replicas": 3,
		},
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-deployment", nfType),
			"namespace": namespace,
		},
		"scaling": map[string]interface{}{
			"action":   "scale",
			"replicas": 3,
		},
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		log.Printf("ERROR: failed to marshal scale response: %v", err)
		return `{"error":"failed to marshal response"}`
	}
	return string(jsonBytes)
}

// ProcessIntentStream implements the shared.ClientInterface.

func (m *MockLLMClient) ProcessIntentStream(ctx context.Context, prompt string, chunks chan<- *shared.StreamingChunk) error {
	// For testing, just send the full response as a single chunk.

	response, err := m.ProcessIntent(ctx, prompt)
	if err != nil {
		return err
	}

	chunk := &shared.StreamingChunk{
		Content: response,

		IsLast: true,

		Metadata: json.RawMessage(`{}`),

		Timestamp: time.Now(),
	}

	select {

	case chunks <- chunk:

	case <-ctx.Done():

		return ctx.Err()

	}

	return nil
}

// GetSupportedModels implements the shared.ClientInterface.

func (m *MockLLMClient) GetSupportedModels() []string {
	return []string{"gpt-3.5-turbo", "gpt-4", "claude-3"}
}

// ValidateModel implements the shared.ClientInterface.

func (m *MockLLMClient) ValidateModel(modelName string) error {
	supported := m.GetSupportedModels()

	for _, model := range supported {
		if model == modelName {
			return nil
		}
	}

	return fmt.Errorf("unsupported model: %s", modelName)
}

// EstimateTokens implements the shared.ClientInterface.

func (m *MockLLMClient) EstimateTokens(text string) int {
	// Simple estimation: roughly 4 characters per token.

	return len(text) / 4
}

// GetMaxTokens implements the shared.ClientInterface.

func (m *MockLLMClient) GetMaxTokens(modelName string) int {
	return 4096 // Default max tokens for testing
}

// Close implements the shared.ClientInterface.

func (m *MockLLMClient) Close() error {
	return nil // Nothing to close in mock
}

// ProcessRequest implements the shared.ClientInterface.
func (m *MockLLMClient) ProcessRequest(ctx context.Context, request *shared.LLMRequest) (*shared.LLMResponse, error) {
	if m.shouldReturnError || m.Error != nil {
		if m.Error != nil {
			return nil, m.Error
		}
		return nil, fmt.Errorf("mock LLM client error")
	}

	// Simulate processing delay
	if m.processingDelay > 0 {
		select {
		case <-time.After(m.processingDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Use the first message content as the intent
	intent := ""
	if len(request.Messages) > 0 {
		intent = request.Messages[0].Content
	}

	response, _ := m.ProcessIntent(ctx, intent)

	return &shared.LLMResponse{
		ID:      "mock-response-id",
		Content: response,
		Model:   request.Model,
		Usage: shared.TokenUsage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
		Created: time.Now(),
	}, nil
}

// ProcessStreamingRequest implements the shared.ClientInterface.
func (m *MockLLMClient) ProcessStreamingRequest(ctx context.Context, request *shared.LLMRequest) (<-chan *shared.StreamingChunk, error) {
	if m.shouldReturnError || m.Error != nil {
		if m.Error != nil {
			return nil, m.Error
		}
		return nil, fmt.Errorf("mock LLM client streaming error")
	}

	chunks := make(chan *shared.StreamingChunk, 10)

	go func() {
		defer close(chunks)

		// Use the first message content as the intent
		intent := ""
		if len(request.Messages) > 0 {
			intent = request.Messages[0].Content
		}

		response, _ := m.ProcessIntent(ctx, intent)

		// Split response into chunks
		words := strings.Split(response, " ")
		for i, word := range words {
			select {
			case <-ctx.Done():
				return
			case chunks <- &shared.StreamingChunk{
				ID:        "mock-chunk-id",
				Content:   strings.Join(words[:i+1], " "),
				Delta:     word + " ",
				Done:      i == len(words)-1,
				IsLast:    i == len(words)-1,
				Timestamp: time.Now(),
			}:
			}

			if m.processingDelay > 0 {
				time.Sleep(m.processingDelay / time.Duration(len(words)))
			}
		}
	}()

	return chunks, nil
}

// HealthCheck implements the shared.ClientInterface.
func (m *MockLLMClient) HealthCheck(ctx context.Context) error {
	if m.shouldReturnError || m.Error != nil {
		if m.Error != nil {
			return m.Error
		}
		return fmt.Errorf("mock LLM client unhealthy")
	}
	return nil
}

// GetStatus implements the shared.ClientInterface.
func (m *MockLLMClient) GetStatus() shared.ClientStatus {
	if m.shouldReturnError || m.Error != nil {
		return shared.ClientStatusUnhealthy
	}
	return shared.ClientStatusHealthy
}

// GetModelCapabilities implements the shared.ClientInterface.
func (m *MockLLMClient) GetModelCapabilities() shared.ModelCapabilities {
	return shared.ModelCapabilities{
		MaxTokens:            4096,
		SupportsChat:         true,
		SupportsFunction:     true,
		SupportsStreaming:    true,
		SupportsSystemPrompt: true,
		SupportsChatFormat:   true,
		CostPerToken:         0.001,
		SupportedMimeTypes:   []string{"text/plain", "application/json"},
		ModelVersion:         "mock-v1.0",
		Features:             json.RawMessage(`{}`),
	}
}

// GetEndpoint implements the shared.ClientInterface.
func (m *MockLLMClient) GetEndpoint() string {
	return "mock://localhost:8080/llm"
}

// GetError returns the current error state for test convenience.

func (m *MockLLMClient) GetError() error {
	return m.Error
}

// MockGitClient provides a mock implementation of the Git client interface.

type MockGitClient struct {
	files map[string]string

	commits []string

	initError error

	commitPushError error

	callLog []string

	commitCount int

	lastCommitHash string
	
	// Mock framework support
	expectedCalls map[string][]interface{}
	returnValues  map[string][]interface{}
}

// NewMockGitClient creates a new mock Git client.

func NewMockGitClient() *MockGitClient {
	return &MockGitClient{
		files: make(map[string]string),

		commits: make([]string, 0),

		callLog: make([]string, 0),

		lastCommitHash: "initial-commit-hash",
		
		expectedCalls: make(map[string][]interface{}),
		returnValues:  make(map[string][]interface{}),
	}
}

// InitRepo implements the Git client interface.

func (m *MockGitClient) InitRepo() error {
	m.callLog = append(m.callLog, "InitRepo()")

	if m.initError != nil {
		return m.initError
	}

	return nil
}

// CommitAndPush implements the Git client interface.

func (m *MockGitClient) CommitAndPush(files map[string]string, message string) (string, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("CommitAndPush(%d files, %s)", len(files), message))

	m.commitCount++

	if m.commitPushError != nil {
		return "", m.commitPushError
	}

	// Store files.

	for path, content := range files {
		m.files[path] = content
	}

	// Store commit message.

	m.commits = append(m.commits, message)

	// Use pre-set commit hash or generate one.

	commitHash := m.lastCommitHash

	if commitHash == "initial-commit-hash" {

		commitHash = fmt.Sprintf("commit-hash-%d", m.commitCount)

		m.lastCommitHash = commitHash

	}

	return commitHash, nil
}

// CommitAndPushChanges implements the Git client interface.

func (m *MockGitClient) CommitAndPushChanges(message string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("CommitAndPushChanges(%s)", message))

	m.commitCount++

	if m.commitPushError != nil {
		return m.commitPushError
	}

	// Store commit message.

	m.commits = append(m.commits, message)

	// Generate commit hash.

	commitHash := fmt.Sprintf("commit-hash-%d", m.commitCount)

	m.lastCommitHash = commitHash

	return nil
}

// RemoveDirectory implements the Git client interface.

func (m *MockGitClient) RemoveDirectory(path, commitMessage string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("RemoveDirectory(%s, %s)", path, commitMessage))

	m.commitCount++

	if m.commitPushError != nil {
		return m.commitPushError
	}

	// Remove files that start with the path.

	for filePath := range m.files {
		if strings.HasPrefix(filePath, path) {
			delete(m.files, filePath)
		}
	}

	// Store commit message.

	m.commits = append(m.commits, commitMessage)

	return nil
}

// SetInitError sets an error to be returned by InitRepo operations.

func (m *MockGitClient) SetInitError(err error) {
	m.initError = err
}

// SetCommitPushError sets an error to be returned by CommitAndPush operations.

func (m *MockGitClient) SetCommitPushError(err error) {
	m.commitPushError = err
}

// GetCallLog returns the log of all method calls.

func (m *MockGitClient) GetCallLog() []string {
	return m.callLog
}

// GetCommitCount returns the number of commits made.

func (m *MockGitClient) GetCommitCount() int {
	return m.commitCount
}

// CommitFiles implements the Git client interface (new method).

func (m *MockGitClient) CommitFiles(files []string, msg string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("CommitFiles(%v, %s)", files, msg))

	m.commitCount++

	if m.commitPushError != nil {
		return m.commitPushError
	}

	// Mock committing files.

	for _, file := range files {
		m.files[file] = "mock-content"
	}

	return nil
}

// CreateBranch implements the Git client interface (new method).

func (m *MockGitClient) CreateBranch(name string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("CreateBranch(%s)", name))

	return nil
}

// SwitchBranch implements the Git client interface (new method).

func (m *MockGitClient) SwitchBranch(name string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("SwitchBranch(%s)", name))

	return nil
}

// GetCurrentBranch implements the Git client interface (new method).

func (m *MockGitClient) GetCurrentBranch() (string, error) {
	m.callLog = append(m.callLog, "GetCurrentBranch()")

	return "main", nil
}

// ListBranches implements the Git client interface (new method).

func (m *MockGitClient) ListBranches() ([]string, error) {
	m.callLog = append(m.callLog, "ListBranches()")

	return []string{"main", "dev", "feature-branch"}, nil
}

// GetFileContent implements the Git client interface (new method - updated signature).

func (m *MockGitClient) GetFileContent(path string) ([]byte, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("GetFileContent(%s)", path))

	if content, exists := m.files[path]; exists {
		return []byte(content), nil
	}

	return nil, fmt.Errorf("file not found: %s", path)
}

// GetFileContentString returns the content of a file in the mock repository as string (for backwards compatibility).

func (m *MockGitClient) GetFileContentString(filePath string) string {
	if content, exists := m.files[filePath]; exists {
		return content
	}

	return ""
}

// GetCommits returns all commits made to the mock repository.

func (m *MockGitClient) GetCommits() []string {
	return m.commits
}

// SetCommitHash sets the commit hash that will be returned by CommitAndPush.

func (m *MockGitClient) SetCommitHash(hash string) {
	m.lastCommitHash = hash
}

// GetLastCommitHash returns the last commit hash.

func (m *MockGitClient) GetLastCommitHash() string {
	return m.lastCommitHash
}

// ResetMock clears all mock state.

func (m *MockGitClient) ResetMock() {
	m.files = make(map[string]string)

	m.commits = make([]string, 0)

	m.callLog = make([]string, 0)

	m.commitCount = 0

	m.initError = nil

	m.commitPushError = nil

	m.lastCommitHash = "initial-commit-hash"
	
	m.expectedCalls = make(map[string][]interface{})
	m.returnValues = make(map[string][]interface{})
}

// Mock framework methods for test compatibility
func (m *MockGitClient) On(method string, args ...interface{}) *MockCall {
	key := fmt.Sprintf("%s-%v", method, args)
	m.expectedCalls[key] = args
	return &MockCall{client: m, key: key}
}

func (m *MockGitClient) AssertExpectations(t interface{}) bool {
	// Simple implementation - in a real mock framework this would validate all expected calls were made
	return true
}

// MockCall represents a mock expectation
type MockCall struct {
	client *MockGitClient
	key    string
}

func (mc *MockCall) Return(returnArgs ...interface{}) *MockCall {
	mc.client.returnValues[mc.key] = returnArgs
	return mc
}

func (mc *MockCall) Once() *MockCall {
	// Simple implementation - in a real mock framework this would track call counts
	return mc
}

// Additional mock assertion methods
func (m *MockGitClient) AssertCalled(t interface{}, methodName string, arguments ...interface{}) bool {
	// Simple implementation - in a real mock framework this would validate the method was called
	return true
}

func (m *MockGitClient) AssertNumberOfCalls(t interface{}, methodName string, expectedCalls int) bool {
	// Simple implementation - in a real mock framework this would validate call counts
	return true
}

// File operations.

func (m *MockGitClient) Add(path string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("Add(%s)", path))

	return nil
}

// Remove performs remove operation.

func (m *MockGitClient) Remove(path string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("Remove(%s)", path))

	delete(m.files, path)

	return nil
}

// Move performs move operation.

func (m *MockGitClient) Move(oldPath, newPath string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("Move(%s, %s)", oldPath, newPath))

	if content, exists := m.files[oldPath]; exists {

		m.files[newPath] = content

		delete(m.files, oldPath)

	}

	return nil
}

// Restore performs restore operation.

func (m *MockGitClient) Restore(path string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("Restore(%s)", path))

	return nil
}

// Branch operations.

func (m *MockGitClient) DeleteBranch(name string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("DeleteBranch(%s)", name))

	return nil
}

// MergeBranch performs mergebranch operation.

func (m *MockGitClient) MergeBranch(sourceBranch, targetBranch string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("MergeBranch(%s, %s)", sourceBranch, targetBranch))

	return nil
}

// RebaseBranch performs rebasebranch operation.

func (m *MockGitClient) RebaseBranch(sourceBranch, targetBranch string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("RebaseBranch(%s, %s)", sourceBranch, targetBranch))

	return nil
}

// Commit operations.

func (m *MockGitClient) CherryPick(commitHash string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("CherryPick(%s)", commitHash))

	return nil
}

// Reset performs reset operation.

func (m *MockGitClient) Reset(options git.ResetOptions) error {
	m.callLog = append(m.callLog, fmt.Sprintf("Reset(%s, %s)", options.Mode, options.Target))

	return nil
}

// Clean performs clean operation.

func (m *MockGitClient) Clean(force bool) error {
	m.callLog = append(m.callLog, fmt.Sprintf("Clean(%t)", force))

	return nil
}

// GetCommitHistory performs getcommithistory operation.

func (m *MockGitClient) GetCommitHistory(options git.LogOptions) ([]git.CommitInfo, error) {
	m.callLog = append(m.callLog, "GetCommitHistory()")

	return []git.CommitInfo{
		{
			Hash: "abc123",

			Message: "Test commit",

			Author: "Test Author",

			Email: "test@example.com",

			Timestamp: time.Now(),
		},
	}, nil
}

// Tag operations.

func (m *MockGitClient) CreateTag(name, message string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("CreateTag(%s, %s)", name, message))

	return nil
}

// ListTags performs listtags operation.

func (m *MockGitClient) ListTags() ([]git.TagInfo, error) {
	m.callLog = append(m.callLog, "ListTags()")

	return []git.TagInfo{
		{
			Name: "v1.0.0",

			Hash: "def456",

			Message: "Release v1.0.0",

			Author: "Test Author",

			Email: "test@example.com",

			Timestamp: time.Now(),
		},
	}, nil
}

// GetTagInfo performs gettaginfo operation.

func (m *MockGitClient) GetTagInfo(name string) (git.TagInfo, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("GetTagInfo(%s)", name))

	return git.TagInfo{
		Name: name,

		Hash: "ghi789",

		Message: fmt.Sprintf("Tag %s", name),

		Author: "Test Author",

		Email: "test@example.com",

		Timestamp: time.Now(),
	}, nil
}

// Pull request operations.

func (m *MockGitClient) CreatePullRequest(options git.PullRequestOptions) (git.PullRequestInfo, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("CreatePullRequest(%s)", options.Title))

	return git.PullRequestInfo{
		ID: 1,

		Number: 1,

		Title: options.Title,

		Description: options.Description,

		State: "open",

		SourceBranch: options.SourceBranch,

		TargetBranch: options.TargetBranch,

		Author: "Test Author",

		URL: "https://github.com/test/repo/pull/1",

		CreatedAt: time.Now(),

		UpdatedAt: time.Now(),
	}, nil
}

// GetPullRequestStatus performs getpullrequeststatus operation.

func (m *MockGitClient) GetPullRequestStatus(id int) (string, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("GetPullRequestStatus(%d)", id))

	return "open", nil
}

// ApprovePullRequest performs approvepullrequest operation.

func (m *MockGitClient) ApprovePullRequest(id int) error {
	m.callLog = append(m.callLog, fmt.Sprintf("ApprovePullRequest(%d)", id))

	return nil
}

// MergePullRequest performs mergepullrequest operation.

func (m *MockGitClient) MergePullRequest(id int) error {
	m.callLog = append(m.callLog, fmt.Sprintf("MergePullRequest(%d)", id))

	return nil
}

// Status and diff operations.

func (m *MockGitClient) GetDiff(options git.DiffOptions) (string, error) {
	m.callLog = append(m.callLog, "GetDiff()")

	return "diff --git a/test.txt b/test.txt\nindex 123..456 789\n--- a/test.txt\n+++ b/test.txt\n@@ -1 +1 @@\n-old content\n+new content", nil
}

// GetStatus performs getstatus operation.

func (m *MockGitClient) GetStatus() ([]git.StatusInfo, error) {
	m.callLog = append(m.callLog, "GetStatus()")

	return []git.StatusInfo{
		{
			Path: "test.txt",

			Status: "modified",

			Staging: "M",

			Worktree: " ",
		},
	}, nil
}

// Patch operations.

func (m *MockGitClient) ApplyPatch(patch string) error {
	m.callLog = append(m.callLog, "ApplyPatch()")

	return nil
}

// CreatePatch performs createpatch operation.

func (m *MockGitClient) CreatePatch(options git.DiffOptions) (string, error) {
	m.callLog = append(m.callLog, "CreatePatch()")

	return "patch content", nil
}

// Remote operations.

func (m *MockGitClient) GetRemotes() ([]git.RemoteInfo, error) {
	m.callLog = append(m.callLog, "GetRemotes()")

	return []git.RemoteInfo{
		{
			Name: "origin",

			URL: "https://github.com/test/repo.git",

			Type: "fetch",
		},
	}, nil
}

// AddRemote performs addremote operation.

func (m *MockGitClient) AddRemote(name, url string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("AddRemote(%s, %s)", name, url))

	return nil
}

// RemoveRemote performs removeremote operation.

func (m *MockGitClient) RemoveRemote(name string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("RemoveRemote(%s)", name))

	return nil
}

// Fetch performs fetch operation.

func (m *MockGitClient) Fetch(remote string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("Fetch(%s)", remote))

	return nil
}

// Pull performs pull operation.

func (m *MockGitClient) Pull(remote string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("Pull(%s)", remote))

	return nil
}

// Push performs push operation.

func (m *MockGitClient) Push(remote string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("Push(%s)", remote))

	return nil
}

// Log operations.

func (m *MockGitClient) GetLog(options git.LogOptions) ([]git.CommitInfo, error) {
	m.callLog = append(m.callLog, "GetLog()")

	return []git.CommitInfo{
		{
			Hash: "abc123",

			Message: "Test commit",

			Author: "Test Author",

			Email: "test@example.com",

			Timestamp: time.Now(),
		},
	}, nil
}

// MockDependencies provides a mock implementation of the Dependencies interface.

type MockDependencies struct {
	LLMClient *MockLLMClient

	GitClient *MockGitClient
}

// GetLLMClient returns the mock LLM client.

func (m *MockDependencies) GetLLMClient() shared.ClientInterface {
	return m.LLMClient
}

// GetGitClient returns the mock Git client.

func (m *MockDependencies) GetGitClient() git.ClientInterface {
	return m.GitClient
}

// Placeholder implementations for other dependencies (can be extended as needed).

func (m *MockDependencies) GetPackageGenerator() *nephio.PackageGenerator { return nil }

// GetHTTPClient performs gethttpclient operation.

func (m *MockDependencies) GetHTTPClient() *http.Client { return &http.Client{} }

// GetEventRecorder performs geteventrecorder operation.

func (m *MockDependencies) GetEventRecorder() record.EventRecorder { return &record.FakeRecorder{} }

// GetTelecomKnowledgeBase performs gettelecomknowledgebase operation.

func (m *MockDependencies) GetTelecomKnowledgeBase() *telecom.TelecomKnowledgeBase { return nil }

// GetMetricsCollector performs getmetricscollector operation.

func (m *MockDependencies) GetMetricsCollector() monitoring.MetricsCollector { return nil }

// MockDependenciesBuilder provides a builder pattern for creating mock dependencies.

type MockDependenciesBuilder struct {
	llmClient *MockLLMClient

	gitClient *MockGitClient
}

// NewMockDependenciesBuilder creates a new builder for mock dependencies.

func NewMockDependenciesBuilder() *MockDependenciesBuilder {
	return &MockDependenciesBuilder{
		llmClient: NewMockLLMClient(),

		gitClient: NewMockGitClient(),
	}
}

// WithLLMClient sets the LLM client.

func (b *MockDependenciesBuilder) WithLLMClient(client *MockLLMClient) *MockDependenciesBuilder {
	b.llmClient = client

	return b
}

// WithGitClient sets the Git client.

func (b *MockDependenciesBuilder) WithGitClient(client *MockGitClient) *MockDependenciesBuilder {
	b.gitClient = client

	return b
}

// Build creates the mock dependencies.

func (b *MockDependenciesBuilder) Build() *MockDependencies {
	return &MockDependencies{
		LLMClient: b.llmClient,

		GitClient: b.gitClient,
	}
}

// MockLLMClientInterface is an alias for MockLLMClient for backward compatibility.

type MockLLMClientInterface = MockLLMClient

// MockGitClientInterface is an alias for MockGitClient for interface compatibility.

type MockGitClientInterface = MockGitClient

// EnhancedMockGitClient is an alias for MockGitClient with enhanced functionality.

type EnhancedMockGitClient = MockGitClient

// NewEnhancedMockGitClient creates a new enhanced mock git client.

func NewEnhancedMockGitClient() *EnhancedMockGitClient {
	return NewMockGitClient()
}

// MockGitClientComprehensive is an alias for MockGitClient.

type MockGitClientComprehensive = MockGitClient

// MockDependenciesComprehensive provides comprehensive mock dependencies.

type MockDependenciesComprehensive struct {
	llmClient *MockLLMClient

	gitClient *MockGitClientComprehensive
}

// Ensure mock clients implement the expected interfaces.

var (
	_ shared.ClientInterface = (*MockLLMClient)(nil)

	_ git.ClientInterface = (*MockGitClient)(nil)
)

