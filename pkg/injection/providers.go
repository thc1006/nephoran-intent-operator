package injection

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/git"
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio"
	"github.com/thc1006/nephoran-intent-operator/pkg/security"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
	"github.com/thc1006/nephoran-intent-operator/pkg/telecom"
)

// Error types for dependency injection.

type ErrMissingGitConfig struct {
	Field string
}

// Error performs error operation.

func (e ErrMissingGitConfig) Error() string {
	return fmt.Sprintf("missing required Git configuration: %s", e.Field)
}

// ErrMissingLLMConfig represents a errmissingllmconfig.

type ErrMissingLLMConfig struct {
	Field string
}

// Error performs error operation.

func (e ErrMissingLLMConfig) Error() string {
	return fmt.Sprintf("missing required LLM configuration: %s", e.Field)
}

// SimpleHTTPClient implements the shared.ClientInterface for basic HTTP clients.

type SimpleHTTPClient struct {
	BaseURL string

	Client *http.Client
}

// ProcessIntent performs processintent operation.

func (c *SimpleHTTPClient) ProcessIntent(ctx context.Context, prompt string) (string, error) {
	// Basic implementation - this should be replaced with actual LLM client.

	return "Processed intent: " + prompt, nil
}

// ProcessIntentStream performs processintentstream operation.

func (c *SimpleHTTPClient) ProcessIntentStream(ctx context.Context, prompt string, chunks chan<- *shared.StreamingChunk) error {
	// Basic implementation - this should be replaced with actual streaming client.

	chunk := &shared.StreamingChunk{
		Content: "Streamed response for: " + prompt,

		IsLast: true,

		Timestamp: time.Now(),
	}

	chunks <- chunk

	close(chunks)

	return nil
}

// GetSupportedModels performs getsupportedmodels operation.

func (c *SimpleHTTPClient) GetSupportedModels() []string {
	return []string{"gpt-4o-mini", "claude-3-haiku"}
}

// GetModelCapabilities performs getmodelcapabilities operation.

func (c *SimpleHTTPClient) GetModelCapabilities(modelName string) (*shared.ModelCapabilities, error) {
	return &shared.ModelCapabilities{
		MaxTokens: 4096,

		SupportsChat: true,

		SupportsFunction: false,

		SupportsStreaming: true,

		CostPerToken: 0.0001,
	}, nil
}

// ValidateModel performs validatemodel operation.

func (c *SimpleHTTPClient) ValidateModel(modelName string) error {
	return nil
}

// EstimateTokens performs estimatetokens operation.

func (c *SimpleHTTPClient) EstimateTokens(text string) int {
	return len(text) / 4 // Rough estimate
}

// GetMaxTokens performs getmaxtokens operation.

func (c *SimpleHTTPClient) GetMaxTokens(modelName string) int {
	return 4096
}

// Close performs close operation.

func (c *SimpleHTTPClient) Close() error {
	return nil
}

// HTTPClientProvider creates an HTTP client instance.

func HTTPClientProvider(c *Container) (interface{}, error) {
	config := c.GetConfig()

	client := &http.Client{
		Timeout: config.DefaultTimeout,

		Transport: &http.Transport{
			MaxIdleConns: 100,

			MaxIdleConnsPerHost: 10,

			IdleConnTimeout: 90 * time.Second,
		},
	}

	return client, nil
}

// GitClientProvider creates a Git client instance.

func GitClientProvider(c *Container) (interface{}, error) {
	gitConfig := &git.ClientConfig{
		RepoURL: os.Getenv("GIT_REPO_URL"),

		Branch: os.Getenv("GIT_BRANCH"),

		Token: os.Getenv("GIT_TOKEN"),

		TokenPath: os.Getenv("GIT_TOKEN_PATH"),

		RepoPath: os.Getenv("GIT_REPO_PATH"),

		ConcurrentPushLimit: 4,
	}

	// Validate configuration.

	if gitConfig.RepoURL == "" {
		gitConfig.RepoURL = "https://github.com/example/nephoran-packages.git"
	}

	if gitConfig.Branch == "" {
		gitConfig.Branch = "main"
	}

	if gitConfig.RepoPath == "" {
		gitConfig.RepoPath = "/tmp/nephoran-git"
	}

	client := git.NewClientFromConfig(gitConfig)

	return client, nil
}

// LLMClientProvider creates an LLM client instance.

func LLMClientProvider(c *Container) (interface{}, error) {
	llmURL := os.Getenv("LLM_PROCESSOR_URL")

	if llmURL == "" {
		// Return nil if no LLM processor URL is configured.

		return nil, nil
	}

	// Get HTTP client dependency.

	httpClient := c.GetHTTPClient()

	// Create a simple HTTP client that implements the shared.ClientInterface.

	client := &SimpleHTTPClient{
		BaseURL: llmURL,

		Client: httpClient,
	}

	return client, nil
}

// PackageGeneratorProvider creates a Nephio package generator instance.

func PackageGeneratorProvider(c *Container) (interface{}, error) {
	// NewPackageGenerator doesn't require configuration parameters.

	generator, err := nephio.NewPackageGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to create package generator: %w", err)
	}

	return generator, nil
}

// TelecomKnowledgeBaseProvider creates a telecom knowledge base instance.

func TelecomKnowledgeBaseProvider(c *Container) (interface{}, error) {
	knowledgeBase := telecom.NewTelecomKnowledgeBase()

	return knowledgeBase, nil
}

// MetricsCollectorProvider creates a metrics collector instance.

func MetricsCollectorProvider(c *Container) (interface{}, error) {
	collector := monitoring.NewMetricsCollector()

	return collector, nil
}

// LLMSanitizerProvider creates an LLM sanitizer instance.

func LLMSanitizerProvider(c *Container) (interface{}, error) {
	config := c.GetConfig()

	sanitizerConfig := &security.SanitizerConfig{
		MaxInputLength: config.MaxInputLength,

		MaxOutputLength: config.MaxOutputLength,

		AllowedDomains: config.AllowedDomains,

		BlockedKeywords: config.BlockedKeywords,

		ContextBoundary: config.ContextBoundary,

		SystemPrompt: config.SystemPrompt,
	}

	sanitizer := security.NewLLMSanitizer(sanitizerConfig)

	return sanitizer, nil
}

// Additional helper providers for complex configurations.

// SecureHTTPClientProvider creates an HTTP client with enhanced security settings.

func SecureHTTPClientProvider(c *Container) (interface{}, error) {
	config := c.GetConfig()

	transport := &http.Transport{
		MaxIdleConns: 50,

		MaxIdleConnsPerHost: 5,

		IdleConnTimeout: 60 * time.Second,

		DisableCompression: false,

		DisableKeepAlives: false,
	}

	client := &http.Client{
		Timeout: config.DefaultTimeout,

		Transport: transport,
	}

	return client, nil
}

// CircuitBreakerHTTPClientProvider creates an HTTP client with circuit breaker.

func CircuitBreakerHTTPClientProvider(c *Container) (interface{}, error) {
	config := c.GetConfig()

	// Create base HTTP client.

	baseClient := &http.Client{
		Timeout: config.DefaultTimeout,

		Transport: &http.Transport{
			MaxIdleConns: 100,

			MaxIdleConnsPerHost: 10,

			IdleConnTimeout: 90 * time.Second,
		},
	}

	// TODO: Wrap with circuit breaker when implemented.

	// This would integrate with the resilience package.

	return baseClient, nil
}

// ConfiguredGitClientProvider creates a Git client with full configuration.

func ConfiguredGitClientProvider(c *Container) (interface{}, error) {
	gitConfig := &git.ClientConfig{
		RepoURL: getEnvWithDefault("GIT_REPO_URL", ""),

		Branch: getEnvWithDefault("GIT_BRANCH", "main"),

		Token: os.Getenv("GIT_TOKEN"), // Keep sensitive tokens from env only

		TokenPath: os.Getenv("GIT_TOKEN_PATH"),

		RepoPath: getEnvWithDefault("GIT_REPO_PATH", "/tmp/nephoran-git"),

		ConcurrentPushLimit: 4,
	}

	// Validation with better error messages.

	if gitConfig.RepoURL == "" {
		return nil, ErrMissingGitConfig{Field: "GIT_REPO_URL"}
	}

	client := git.NewClientFromConfig(gitConfig)

	return client, nil
}

// ProductionLLMClientProvider creates an LLM client with production settings.

func ProductionLLMClientProvider(c *Container) (interface{}, error) {
	llmURL := os.Getenv("LLM_PROCESSOR_URL")

	if llmURL == "" {
		return nil, ErrMissingLLMConfig{Field: "LLM_PROCESSOR_URL"}
	}

	// Get HTTP client with circuit breaker.

	httpClient, err := c.Get("circuit_breaker_http_client")
	if err != nil {
		// Fallback to regular HTTP client.

		httpClient = c.GetHTTPClient()
	}

	// Create a simple HTTP client with production configuration.

	// This should be replaced with actual production LLM client.

	client := &SimpleHTTPClient{
		BaseURL: llmURL,

		Client: httpClient.(*http.Client),
	}

	return client, nil
}

// getEnvWithDefault returns environment variable value or default.

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return defaultValue
}
