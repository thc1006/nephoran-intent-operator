package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the Nephoran Intent Operator
type Config struct {
	// Controller configuration
	MetricsAddr          string
	ProbeAddr           string
	EnableLeaderElection bool
	
	// LLM Processor configuration
	LLMProcessorURL     string
	LLMProcessorTimeout time.Duration
	
	// RAG API configuration
	RAGAPIURLInternal   string
	RAGAPIURLExternal   string
	RAGAPITimeout      time.Duration
	
	// Git integration configuration
	GitRepoURL         string
	GitToken           string
	GitBranch          string
	
	// Weaviate configuration
	WeaviateURL        string
	WeaviateIndex      string
	
	// OpenAI configuration
	OpenAIAPIKey       string
	OpenAIModel        string
	OpenAIEmbeddingModel string
	
	// Kubernetes configuration
	Namespace          string
	CRDPath           string
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		MetricsAddr:          ":8080",
		ProbeAddr:           ":8081",
		EnableLeaderElection: false,
		
		LLMProcessorURL:     "http://llm-processor.default.svc.cluster.local:8080",
		LLMProcessorTimeout: 30 * time.Second,
		
		RAGAPIURLInternal:   "http://rag-api.default.svc.cluster.local:5001",
		RAGAPIURLExternal:   "http://localhost:5001",
		RAGAPITimeout:      30 * time.Second,
		
		GitBranch:          "main",
		
		WeaviateURL:        "http://weaviate.default.svc.cluster.local:8080",
		WeaviateIndex:      "telecom_knowledge",
		
		OpenAIModel:        "gpt-4o-mini",
		OpenAIEmbeddingModel: "text-embedding-3-large",
		
		Namespace:          "default",
		CRDPath:           "deployments/crds",
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
	
	if val := os.Getenv("GIT_TOKEN"); val != "" {
		cfg.GitToken = val
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
	if c.OpenAIAPIKey == "" {
		return fmt.Errorf("OPENAI_API_KEY is required")
	}
	
	return nil
}

// GetRAGAPIURL returns the appropriate RAG API URL based on environment
func (c *Config) GetRAGAPIURL(useInternal bool) string {
	if useInternal {
		return c.RAGAPIURLInternal
	}
	return c.RAGAPIURLExternal
}