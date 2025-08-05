package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/interfaces"
)

// Config holds all configuration for the Nephoran Intent Operator
type Config struct {
	// Controller configuration
	MetricsAddr          string
	ProbeAddr            string
	EnableLeaderElection bool

	// LLM Processor configuration
	LLMProcessorURL     string
	LLMProcessorTimeout time.Duration

	// RAG API configuration
	RAGAPIURLInternal string
	RAGAPIURLExternal string
	RAGAPITimeout     time.Duration

	// Git integration configuration
	GitRepoURL  string
	GitToken    string
	GitTokenPath string  // Path to file containing Git token
	GitBranch   string

	// Weaviate configuration
	WeaviateURL   string
	WeaviateIndex string

	// OpenAI configuration
	OpenAIAPIKey         string
	OpenAIModel          string
	OpenAIEmbeddingModel string

	// Kubernetes configuration
	Namespace string
	CRDPath   string
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		MetricsAddr:          ":8080",
		ProbeAddr:            ":8081",
		EnableLeaderElection: false,

		LLMProcessorURL:     "http://llm-processor.default.svc.cluster.local:8080",
		LLMProcessorTimeout: 30 * time.Second,

		RAGAPIURLInternal: "http://rag-api.default.svc.cluster.local:5001",
		RAGAPIURLExternal: "http://localhost:5001",
		RAGAPITimeout:     30 * time.Second,

		GitBranch: "main",

		WeaviateURL:   "http://weaviate.default.svc.cluster.local:8080",
		WeaviateIndex: "telecom_knowledge",

		OpenAIModel:          "gpt-4o-mini",
		OpenAIEmbeddingModel: "text-embedding-3-large",

		Namespace: "default",
		CRDPath:   "deployments/crds",
	}
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() (*Config, error) {
	cfg := DefaultConfig()

	// Override defaults with environment variables if they exist
	if val := os.Getenv("METRICS_ADDR"); val != "" {
		cfg.MetricsAddr = val
	}

	if val := os.Getenv("PROBE_ADDR"); val != "" {
		cfg.ProbeAddr = val
	}

	if val := os.Getenv("ENABLE_LEADER_ELECTION"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			cfg.EnableLeaderElection = b
		}
	}

	if val := os.Getenv("LLM_PROCESSOR_URL"); val != "" {
		cfg.LLMProcessorURL = val
	}

	if val := os.Getenv("LLM_PROCESSOR_TIMEOUT"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.LLMProcessorTimeout = d
		}
	}

	if val := os.Getenv("RAG_API_URL"); val != "" {
		cfg.RAGAPIURLInternal = val
	}

	if val := os.Getenv("RAG_API_URL_EXTERNAL"); val != "" {
		cfg.RAGAPIURLExternal = val
	}

	if val := os.Getenv("RAG_API_TIMEOUT"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.RAGAPITimeout = d
		}
	}

	if val := os.Getenv("GIT_REPO_URL"); val != "" {
		cfg.GitRepoURL = val
	}

	// Check for token file path first
	if val := os.Getenv("GIT_TOKEN_PATH"); val != "" {
		cfg.GitTokenPath = val
		// Try to read the token from file
		if tokenData, err := os.ReadFile(val); err == nil {
			cfg.GitToken = strings.TrimSpace(string(tokenData))
		}
		// If file read fails, GitToken will remain empty and fallback will occur
	}

	// Fallback to direct environment variable if no token loaded from file
	if cfg.GitToken == "" {
		if val := os.Getenv("GIT_TOKEN"); val != "" {
			cfg.GitToken = val
		}
	}

	if val := os.Getenv("GIT_BRANCH"); val != "" {
		cfg.GitBranch = val
	}

	if val := os.Getenv("WEAVIATE_URL"); val != "" {
		cfg.WeaviateURL = val
	}

	if val := os.Getenv("WEAVIATE_INDEX"); val != "" {
		cfg.WeaviateIndex = val
	}

	if val := os.Getenv("OPENAI_API_KEY"); val != "" {
		cfg.OpenAIAPIKey = val
	}

	if val := os.Getenv("OPENAI_MODEL"); val != "" {
		cfg.OpenAIModel = val
	}

	if val := os.Getenv("OPENAI_EMBEDDING_MODEL"); val != "" {
		cfg.OpenAIEmbeddingModel = val
	}

	if val := os.Getenv("NAMESPACE"); val != "" {
		cfg.Namespace = val
	}

	if val := os.Getenv("CRD_PATH"); val != "" {
		cfg.CRDPath = val
	}

	// Validate required configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// Validate checks that required configuration is present
func (c *Config) Validate() error {
	var errors []string

	// Always required configuration
	if c.OpenAIAPIKey == "" {
		errors = append(errors, "OPENAI_API_KEY is required")
	}

	// Git features validation - enabled when Git integration is intended
	// Git features are considered enabled when GitRepoURL is configured or
	// when other Git-related config suggests Git usage
	if c.isGitFeatureEnabled() {
		if c.GitRepoURL == "" {
			errors = append(errors, "GIT_REPO_URL is required when Git features are enabled")
		}
	}

	// LLM processing validation - enabled when LLM processing is intended
	// LLM processing is considered enabled when LLMProcessorURL is configured
	if c.isLLMProcessingEnabled() {
		if c.LLMProcessorURL == "" {
			errors = append(errors, "LLM_PROCESSOR_URL is required when LLM processing is enabled")
		}
	}

	// RAG features validation - enabled when RAG features are intended
	// RAG features are considered enabled when RAG API URL is configured or
	// when RAG-related infrastructure is configured
	if c.isRAGFeatureEnabled() {
		if c.RAGAPIURLInternal == "" {
			errors = append(errors, "RAG_API_URL_INTERNAL is required when RAG features are enabled")
		}
	}

	// Return validation errors if any
	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// isGitFeatureEnabled checks if Git features are enabled based on configuration
func (c *Config) isGitFeatureEnabled() bool {
	// Git features are enabled when GitRepoURL is set or when Git token is provided
	// (indicating intention to use Git even if URL is missing - validation error)
	return c.GitRepoURL != "" || c.GitToken != ""
}

// isLLMProcessingEnabled checks if LLM processing is enabled based on configuration
func (c *Config) isLLMProcessingEnabled() bool {
	// LLM processing is enabled when LLMProcessorURL is set
	// This follows the pattern seen in networkintent_constructor.go
	return c.LLMProcessorURL != ""
}

// isRAGFeatureEnabled checks if RAG features are enabled based on configuration
func (c *Config) isRAGFeatureEnabled() bool {
	// RAG features are enabled when RAG API URL is set or when Weaviate is configured
	// (indicating intention to use RAG even if internal URL is missing - validation error)
	return c.RAGAPIURLInternal != "" || c.RAGAPIURLExternal != "" || c.WeaviateURL != ""
}

// GetRAGAPIURL returns the appropriate RAG API URL based on environment
func (c *Config) GetRAGAPIURL(useInternal bool) string {
	if useInternal {
		return c.RAGAPIURLInternal
	}
	return c.RAGAPIURLExternal
}

// ConfigProvider interface methods

// GetLLMProcessorURL returns the LLM processor URL
func (c *Config) GetLLMProcessorURL() string {
	return c.LLMProcessorURL
}

// GetLLMProcessorTimeout returns the LLM processor timeout
func (c *Config) GetLLMProcessorTimeout() time.Duration {
	return c.LLMProcessorTimeout
}

// GetGitRepoURL returns the Git repository URL
func (c *Config) GetGitRepoURL() string {
	return c.GitRepoURL
}

// GetGitToken returns the Git token
func (c *Config) GetGitToken() string {
	return c.GitToken
}

// GetGitBranch returns the Git branch
func (c *Config) GetGitBranch() string {
	return c.GitBranch
}

// GetWeaviateURL returns the Weaviate URL
func (c *Config) GetWeaviateURL() string {
	return c.WeaviateURL
}

// GetWeaviateIndex returns the Weaviate index name
func (c *Config) GetWeaviateIndex() string {
	return c.WeaviateIndex
}

// GetOpenAIAPIKey returns the OpenAI API key
func (c *Config) GetOpenAIAPIKey() string {
	return c.OpenAIAPIKey
}

// GetOpenAIModel returns the OpenAI model
func (c *Config) GetOpenAIModel() string {
	return c.OpenAIModel
}

// GetOpenAIEmbeddingModel returns the OpenAI embedding model
func (c *Config) GetOpenAIEmbeddingModel() string {
	return c.OpenAIEmbeddingModel
}

// GetNamespace returns the Kubernetes namespace
func (c *Config) GetNamespace() string {
	return c.Namespace
}

// Ensure Config implements interfaces.ConfigProvider
var _ interfaces.ConfigProvider = (*Config)(nil)
