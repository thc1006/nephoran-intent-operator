package testutils

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/git"
	"github.com/thc1006/nephoran-intent-operator/pkg/types"
)

// MockLLMClient provides a mock implementation of the LLM client interface
type MockLLMClient struct {
	responses       map[string]string
	errors          map[string]error
	processingDelay time.Duration
	callCount       int
	lastIntent      string
}

// NewMockLLMClient creates a new mock LLM client
func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		responses:       make(map[string]string),
		errors:          make(map[string]error),
		processingDelay: 100 * time.Millisecond,
	}
}

// ProcessIntent implements the LLM client interface
func (m *MockLLMClient) ProcessIntent(ctx context.Context, intent string) (string, error) {
	m.callCount++
	m.lastIntent = intent

	// Simulate processing delay
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(m.processingDelay):
	}

	// Check for specific error responses
	if err, exists := m.errors[intent]; exists {
		return "", err
	}

	// Check for specific responses
	if response, exists := m.responses[intent]; exists {
		return response, nil
	}

	// Generate default response based on intent content
	return m.generateDefaultResponse(intent), nil
}

// SetResponse sets a specific response for a given intent
func (m *MockLLMClient) SetResponse(intent, response string) {
	m.responses[intent] = response
}

// SetError sets an error to be returned for a specific intent
func (m *MockLLMClient) SetError(intent string, err error) {
	m.errors[intent] = err
}

// SetProcessingDelay sets the simulated processing delay
func (m *MockLLMClient) SetProcessingDelay(delay time.Duration) {
	m.processingDelay = delay
}

// GetCallCount returns the number of times ProcessIntent was called
func (m *MockLLMClient) GetCallCount() int {
	return m.callCount
}

// GetLastIntent returns the last intent that was processed
func (m *MockLLMClient) GetLastIntent() string {
	return m.lastIntent
}

// Reset clears all mock state
func (m *MockLLMClient) Reset() {
	m.responses = make(map[string]string)
	m.errors = make(map[string]error)
	m.callCount = 0
	m.lastIntent = ""
	m.processingDelay = 100 * time.Millisecond
}

// generateDefaultResponse creates a default response based on intent content
func (m *MockLLMClient) generateDefaultResponse(intent string) string {
	lowerIntent := strings.ToLower(intent)

	// Determine response type based on intent content
	if strings.Contains(lowerIntent, "scale") || strings.Contains(lowerIntent, "increase") || strings.Contains(lowerIntent, "decrease") {
		return m.generateScaleResponse(intent)
	}

	return m.generateDeploymentResponse(intent)
}

// generateDeploymentResponse generates a default deployment response
func (m *MockLLMClient) generateDeploymentResponse(intent string) string {
	lowerIntent := strings.ToLower(intent)

	// Determine network function type
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
		"type":      "NetworkFunctionDeployment",
		"name":      fmt.Sprintf("%s-deployment", nfType),
		"namespace": namespace,
		"spec": map[string]interface{}{
			"replicas": 1,
			"image":    fmt.Sprintf("registry.local/%s:latest", nfType),
			"resources": map[string]interface{}{
				"requests": map[string]string{"cpu": "500m", "memory": "1Gi"},
				"limits":   map[string]string{"cpu": "1000m", "memory": "2Gi"},
			},
		},
	}

	jsonBytes, _ := json.Marshal(response)
	return string(jsonBytes)
}

// generateScaleResponse generates a default scaling response
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
		"type":      "NetworkFunctionScale",
		"name":      fmt.Sprintf("%s-deployment", nfType),
		"namespace": namespace,
		"spec": map[string]interface{}{
			"scaling": map[string]interface{}{
				"horizontal": map[string]interface{}{
					"replicas": 3,
				},
			},
		},
	}

	jsonBytes, _ := json.Marshal(response)
	return string(jsonBytes)
}

// ProcessIntentStream implements the shared.ClientInterface
func (m *MockLLMClient) ProcessIntentStream(ctx context.Context, prompt string, chunks chan<- *types.StreamingChunk) error {
	// For testing, just send the full response as a single chunk
	response, err := m.ProcessIntent(ctx, prompt)
	if err != nil {
		return err
	}

	chunk := &types.StreamingChunk{
		Content:   response,
		IsLast:    true,
		Metadata:  make(map[string]interface{}),
		Timestamp: time.Now(),
	}

	select {
	case chunks <- chunk:
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// GetSupportedModels implements the shared.ClientInterface
func (m *MockLLMClient) GetSupportedModels() []string {
	return []string{"gpt-3.5-turbo", "gpt-4", "claude-3"}
}

// GetModelCapabilities implements the shared.ClientInterface
func (m *MockLLMClient) GetModelCapabilities(modelName string) (*types.ModelCapabilities, error) {
	return &types.ModelCapabilities{
		MaxTokens:         4096,
		SupportsChat:      true,
		SupportsFunction:  true,
		SupportsStreaming: true,
		CostPerToken:      0.001,
		Features:          make(map[string]interface{}),
	}, nil
}

// ValidateModel implements the shared.ClientInterface
func (m *MockLLMClient) ValidateModel(modelName string) error {
	supported := m.GetSupportedModels()
	for _, model := range supported {
		if model == modelName {
			return nil
		}
	}
	return fmt.Errorf("unsupported model: %s", modelName)
}

// EstimateTokens implements the shared.ClientInterface
func (m *MockLLMClient) EstimateTokens(text string) int {
	// Simple estimation: roughly 4 characters per token
	return len(text) / 4
}

// GetMaxTokens implements the shared.ClientInterface
func (m *MockLLMClient) GetMaxTokens(modelName string) int {
	return 4096 // Default max tokens for testing
}

// Close implements the shared.ClientInterface
func (m *MockLLMClient) Close() error {
	return nil // Nothing to close in mock
}

// MockGitClient provides a mock implementation of the Git client interface
type MockGitClient struct {
	files           map[string]string
	commits         []string
	initError       error
	commitPushError error
	callLog         []string
	commitCount     int
	lastCommitHash  string
}

// NewMockGitClient creates a new mock Git client
func NewMockGitClient() *MockGitClient {
	return &MockGitClient{
		files:          make(map[string]string),
		commits:        make([]string, 0),
		callLog:        make([]string, 0),
		lastCommitHash: "initial-commit-hash",
	}
}

// InitRepo implements the Git client interface
func (m *MockGitClient) InitRepo() error {
	m.callLog = append(m.callLog, "InitRepo()")

	if m.initError != nil {
		return m.initError
	}

	return nil
}

// CommitAndPush implements the Git client interface
func (m *MockGitClient) CommitAndPush(files map[string]string, message string) (string, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("CommitAndPush(%d files, %s)", len(files), message))
	m.commitCount++

	if m.commitPushError != nil {
		return "", m.commitPushError
	}

	// Store files
	for path, content := range files {
		m.files[path] = content
	}

	// Store commit message
	m.commits = append(m.commits, message)

	// Generate commit hash
	commitHash := fmt.Sprintf("commit-hash-%d", m.commitCount)
	m.lastCommitHash = commitHash

	return commitHash, nil
}

// CommitAndPushChanges implements the Git client interface
func (m *MockGitClient) CommitAndPushChanges(message string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("CommitAndPushChanges(%s)", message))
	m.commitCount++

	if m.commitPushError != nil {
		return m.commitPushError
	}

	// Store commit message
	m.commits = append(m.commits, message)

	// Generate commit hash
	commitHash := fmt.Sprintf("commit-hash-%d", m.commitCount)
	m.lastCommitHash = commitHash

	return nil
}

// RemoveDirectory implements the Git client interface
func (m *MockGitClient) RemoveDirectory(path string, commitMessage string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("RemoveDirectory(%s, %s)", path, commitMessage))
	m.commitCount++

	if m.commitPushError != nil {
		return m.commitPushError
	}

	// Remove files that start with the path
	for filePath := range m.files {
		if strings.HasPrefix(filePath, path) {
			delete(m.files, filePath)
		}
	}

	// Store commit message
	m.commits = append(m.commits, commitMessage)

	return nil
}

// RemoveAndPush implements the Git client interface
func (m *MockGitClient) RemoveAndPush(path string, commitMessage string) (string, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("RemoveAndPush(%s, %s)", path, commitMessage))
	m.commitCount++

	if m.commitPushError != nil {
		return "", m.commitPushError
	}

	// Remove files that start with the path
	for filePath := range m.files {
		if strings.HasPrefix(filePath, path) {
			delete(m.files, filePath)
		}
	}

	// Store commit message and generate a new commit hash
	m.commits = append(m.commits, commitMessage)
	m.lastCommitHash = fmt.Sprintf("remove-commit-%d", m.commitCount)

	return m.lastCommitHash, nil
}

func (m *MockGitClient) CommitFiles(ctx context.Context, files map[string]string, message string) (string, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("CommitFiles(%d files, %s)", len(files), message))
	m.commitCount++

	if m.commitPushError != nil {
		return "", m.commitPushError
	}

	// Store files
	for path, content := range files {
		m.files[path] = content
	}

	// Store commit message
	m.commits = append(m.commits, message)

	// Generate commit hash
	commitHash := fmt.Sprintf("commit-hash-%d", m.commitCount)
	m.lastCommitHash = commitHash

	return commitHash, nil
}

// Branch operations
func (m *MockGitClient) CreateBranch(ctx context.Context, branchName, baseBranch string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("CreateBranch(%s, %s)", branchName, baseBranch))
	return m.commitPushError
}

func (m *MockGitClient) SwitchBranch(ctx context.Context, branchName string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("SwitchBranch(%s)", branchName))
	return m.commitPushError
}

func (m *MockGitClient) GetCurrentBranch(ctx context.Context) (string, error) {
	m.callLog = append(m.callLog, "GetCurrentBranch()")
	if m.commitPushError != nil {
		return "", m.commitPushError
	}
	return "main", nil
}

func (m *MockGitClient) ListBranches(ctx context.Context) ([]string, error) {
	m.callLog = append(m.callLog, "ListBranches()")
	if m.commitPushError != nil {
		return nil, m.commitPushError
	}
	return []string{"main", "develop"}, nil
}

// File operations  
func (m *MockGitClient) GetFileContent(ctx context.Context, filePath string) (string, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("GetFileContent(%s)", filePath))
	if m.commitPushError != nil {
		return "", m.commitPushError
	}
	if content, exists := m.files[filePath]; exists {
		return content, nil
	}
	return "", fmt.Errorf("file not found: %s", filePath)
}

func (m *MockGitClient) DeleteFile(ctx context.Context, filePath, message string) (string, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("DeleteFile(%s, %s)", filePath, message))
	m.commitCount++
	if m.commitPushError != nil {
		return "", m.commitPushError
	}
	delete(m.files, filePath)
	commitHash := fmt.Sprintf("commit-hash-%d", m.commitCount)
	m.lastCommitHash = commitHash
	m.commits = append(m.commits, message)
	return commitHash, nil
}

// Repository operations
func (m *MockGitClient) Push(ctx context.Context, branch string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("Push(%s)", branch))
	return m.commitPushError
}

func (m *MockGitClient) Pull(ctx context.Context, branch string) error {
	m.callLog = append(m.callLog, fmt.Sprintf("Pull(%s)", branch))
	return m.commitPushError
}

func (m *MockGitClient) GetStatus(ctx context.Context) (*git.StatusInfo, error) {
	m.callLog = append(m.callLog, "GetStatus()")
	if m.commitPushError != nil {
		return nil, m.commitPushError
	}
	return &git.StatusInfo{Clean: true}, nil
}

func (m *MockGitClient) GetCommitHistory(ctx context.Context, limit int) ([]git.CommitInfo, error) {
	m.callLog = append(m.callLog, fmt.Sprintf("GetCommitHistory(%d)", limit))
	if m.commitPushError != nil {
		return nil, m.commitPushError
	}
	return []git.CommitInfo{}, nil
}

// Pull request operations
func (m *MockGitClient) CreatePullRequest(ctx context.Context, options *git.PullRequestOptions) (*git.PullRequestInfo, error) {
	m.callLog = append(m.callLog, "CreatePullRequest()")
	if m.commitPushError != nil {
		return nil, m.commitPushError
	}
	return &git.PullRequestInfo{Number: 1, URL: "https://github.com/test/repo/pull/1"}, nil
}

// SetInitError sets an error to be returned by InitRepo operations
func (m *MockGitClient) SetInitError(err error) {
	m.initError = err
}

// SetCommitPushError sets an error to be returned by CommitAndPush operations
func (m *MockGitClient) SetCommitPushError(err error) {
	m.commitPushError = err
}

// GetCallLog returns the log of all method calls
func (m *MockGitClient) GetCallLog() []string {
	return m.callLog
}

// GetCommitCount returns the number of commits made
func (m *MockGitClient) GetCommitCount() int {
	return m.commitCount
}

// GetFileContentString returns the content of a file in the mock repository (legacy method)
func (m *MockGitClient) GetFileContentString(filePath string) string {
	if content, exists := m.files[filePath]; exists {
		return content
	}
	return ""
}

// GetCommits returns all commits made to the mock repository
func (m *MockGitClient) GetCommits() []string {
	return m.commits
}

// Reset clears all mock state
func (m *MockGitClient) Reset() {
	m.files = make(map[string]string)
	m.commits = make([]string, 0)
	m.callLog = make([]string, 0)
	m.commitCount = 0
	m.initError = nil
	m.commitPushError = nil
	m.lastCommitHash = "initial-commit-hash"
}

// Ensure mock clients implement the expected interfaces
var _ types.ClientInterface = (*MockLLMClient)(nil)
var _ git.ClientInterface = (*MockGitClient)(nil)
