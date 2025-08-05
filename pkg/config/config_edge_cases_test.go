package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate_GitFeatureEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func() *Config
		description string
		wantErr     bool
		errMsg      string
	}{
		{
			name: "git features enabled by token only - missing GitRepoURL",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.GitRepoURL = ""
				cfg.GitToken = "github_token_123"
				return cfg
			},
			description: "Git token present but GitRepoURL missing should fail",
			wantErr:     true,
			errMsg:      "GIT_REPO_URL is required when Git features are enabled",
		},
		{
			name: "git features enabled by URL only - no token",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.GitRepoURL = "https://github.com/test/repo.git"
				cfg.GitToken = ""
				return cfg
			},
			description: "GitRepoURL present without token should be valid (public repo)",
			wantErr:     false,
		},
		{
			name: "git features disabled - no URL or token",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.GitRepoURL = ""
				cfg.GitToken = ""
				return cfg
			},
			description: "No git configuration should be valid",
			wantErr:     false,
		},
		{
			name: "git features enabled with both URL and token",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.GitRepoURL = "https://github.com/test/private-repo.git"
				cfg.GitToken = "github_token_123"
				return cfg
			},
			description: "Both GitRepoURL and token present should be valid",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.setupConfig()
			err := cfg.Validate()

			if tt.wantErr {
				assert.Error(t, err, tt.description)
				assert.Contains(t, err.Error(), tt.errMsg, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

func TestConfig_Validate_LLMFeatureEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func() *Config
		description string
		wantErr     bool
		errMsg      string
	}{
		{
			name: "LLM processing explicitly disabled",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.LLMProcessorURL = ""
				return cfg
			},
			description: "Empty LLMProcessorURL should disable LLM processing",
			wantErr:     false,
		},
		{
			name: "LLM processing enabled with valid URL",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.LLMProcessorURL = "http://custom-llm:8080"
				return cfg
			},
			description: "Valid LLMProcessorURL should enable LLM processing",
			wantErr:     false,
		},
		{
			name: "LLM processing enabled with HTTPS URL",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.LLMProcessorURL = "https://secure-llm.example.com:8443"
				return cfg
			},
			description: "HTTPS LLMProcessorURL should be valid",
			wantErr:     false,
		},
		{
			name: "LLM processing with custom timeout",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.LLMProcessorURL = "http://llm:8080"
				cfg.LLMProcessorTimeout = 2 * time.Minute
				return cfg
			},
			description: "Custom LLM timeout should be valid",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.setupConfig()
			err := cfg.Validate()

			if tt.wantErr {
				assert.Error(t, err, tt.description)
				assert.Contains(t, err.Error(), tt.errMsg, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

func TestConfig_Validate_RAGFeatureEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func() *Config
		description string
		wantErr     bool
		errMsg      string
	}{
		{
			name: "RAG features enabled by WeaviateURL only - missing internal URL",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.RAGAPIURLInternal = ""
				cfg.RAGAPIURLExternal = ""
				cfg.WeaviateURL = "http://weaviate:8080"
				return cfg
			},
			description: "WeaviateURL present but RAGAPIURLInternal missing should fail",
			wantErr:     true,
			errMsg:      "RAG_API_URL_INTERNAL is required when RAG features are enabled",
		},
		{
			name: "RAG features enabled by external URL only - missing internal URL",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.RAGAPIURLInternal = ""
				cfg.RAGAPIURLExternal = "http://external-rag:5001"
				cfg.WeaviateURL = ""
				return cfg
			},
			description: "RAGAPIURLExternal present but RAGAPIURLInternal missing should fail",
			wantErr:     true,
			errMsg:      "RAG_API_URL_INTERNAL is required when RAG features are enabled",
		},
		{
			name: "RAG features enabled by internal URL only",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.RAGAPIURLInternal = "http://rag-api:5001"
				cfg.RAGAPIURLExternal = ""
				cfg.WeaviateURL = ""
				return cfg
			},
			description: "RAGAPIURLInternal present should be sufficient",
			wantErr:     false,
		},
		{
			name: "RAG features fully configured",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.RAGAPIURLInternal = "http://rag-api:5001"
				cfg.RAGAPIURLExternal = "http://external-rag:5001"
				cfg.WeaviateURL = "http://weaviate:8080"
				cfg.WeaviateIndex = "knowledge_base"
				return cfg
			},
			description: "Full RAG configuration should be valid",
			wantErr:     false,
		},
		{
			name: "RAG features disabled - all URLs empty",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.RAGAPIURLInternal = ""
				cfg.RAGAPIURLExternal = ""
				cfg.WeaviateURL = ""
				return cfg
			},
			description: "All RAG URLs empty should disable RAG features",
			wantErr:     false,
		},
		{
			name: "RAG features with HTTPS URLs",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.RAGAPIURLInternal = "https://secure-rag-internal:5001"
				cfg.RAGAPIURLExternal = "https://secure-rag-external:5001"
				cfg.WeaviateURL = "https://secure-weaviate:8080"
				return cfg
			},
			description: "HTTPS RAG URLs should be valid",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.setupConfig()
			err := cfg.Validate()

			if tt.wantErr {
				assert.Error(t, err, tt.description)
				assert.Contains(t, err.Error(), tt.errMsg, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

func TestConfig_Validate_FeatureCombinations(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func() *Config
		description string
		wantErr     bool
		errMsg      string
	}{
		{
			name: "all features enabled - complete configuration",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.GitRepoURL = "https://github.com/test/repo.git"
				cfg.GitToken = "github_token_123"
				cfg.LLMProcessorURL = "http://llm-processor:8080"
				cfg.RAGAPIURLInternal = "http://rag-api:5001"
				cfg.RAGAPIURLExternal = "http://external-rag:5001"
				cfg.WeaviateURL = "http://weaviate:8080"
				return cfg
			},
			description: "All features fully configured should be valid",
			wantErr:     false,
		},
		{
			name: "mixed configuration - git enabled, others disabled",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.GitRepoURL = "https://github.com/test/repo.git"
				cfg.GitToken = "github_token_123"
				cfg.LLMProcessorURL = ""
				cfg.RAGAPIURLInternal = ""
				cfg.RAGAPIURLExternal = ""
				cfg.WeaviateURL = ""
				return cfg
			},
			description: "Only Git features enabled should be valid",
			wantErr:     false,
		},
		{
			name: "mixed configuration - git partially enabled, others disabled",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.GitRepoURL = ""
				cfg.GitToken = "github_token_123" // This enables git features
				cfg.LLMProcessorURL = ""
				cfg.RAGAPIURLInternal = ""
				cfg.RAGAPIURLExternal = ""
				cfg.WeaviateURL = ""
				return cfg
			},
			description: "Git token without URL should fail validation",
			wantErr:     true,
			errMsg:      "GIT_REPO_URL is required when Git features are enabled",
		},
		{
			name: "mixed configuration - RAG partially enabled, others disabled",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.GitRepoURL = ""
				cfg.GitToken = ""
				cfg.LLMProcessorURL = ""
				cfg.RAGAPIURLInternal = ""
				cfg.RAGAPIURLExternal = ""
				cfg.WeaviateURL = "http://weaviate:8080" // This enables RAG features
				return cfg
			},
			description: "Weaviate URL without RAG API URL should fail validation",
			wantErr:     true,
			errMsg:      "RAG_API_URL_INTERNAL is required when RAG features are enabled",
		},
		{
			name: "multiple validation errors",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = ""                                  // Missing required field
				cfg.GitRepoURL = ""                                    // Missing when git enabled
				cfg.GitToken = "github_token_123"                     // This enables git features
				cfg.LLMProcessorURL = ""                               // LLM disabled (OK)
				cfg.RAGAPIURLInternal = ""                             // Missing when RAG enabled
				cfg.WeaviateURL = "http://weaviate:8080"               // This enables RAG features
				return cfg
			},
			description: "Multiple validation errors should be reported",
			wantErr:     true,
			errMsg:      "OPENAI_API_KEY is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.setupConfig()
			err := cfg.Validate()

			if tt.wantErr {
				assert.Error(t, err, tt.description)
				assert.Contains(t, err.Error(), tt.errMsg, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

func TestConfig_FeatureFlagHelpers(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func() *Config
		description string
		expectGit   bool
		expectLLM   bool
		expectRAG   bool
	}{
		{
			name: "all features disabled",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.GitRepoURL = ""
				cfg.GitToken = ""
				cfg.LLMProcessorURL = ""
				cfg.RAGAPIURLInternal = ""
				cfg.RAGAPIURLExternal = ""
				cfg.WeaviateURL = ""
				return cfg
			},
			description: "All features should be detected as disabled",
			expectGit:   false,
			expectLLM:   false,
			expectRAG:   false,
		},
		{
			name: "git enabled by URL",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.GitRepoURL = "https://github.com/test/repo.git"
				cfg.GitToken = ""
				cfg.LLMProcessorURL = ""
				cfg.RAGAPIURLInternal = ""
				cfg.RAGAPIURLExternal = ""
				cfg.WeaviateURL = ""
				return cfg
			},
			description: "Git should be detected as enabled by URL",
			expectGit:   true,
			expectLLM:   false,
			expectRAG:   false,
		},
		{
			name: "git enabled by token",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.GitRepoURL = ""
				cfg.GitToken = "github_token_123"
				cfg.LLMProcessorURL = ""
				cfg.RAGAPIURLInternal = ""
				cfg.RAGAPIURLExternal = ""
				cfg.WeaviateURL = ""
				return cfg
			},
			description: "Git should be detected as enabled by token",
			expectGit:   true,
			expectLLM:   false,
			expectRAG:   false,
		},
		{
			name: "LLM enabled",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.GitRepoURL = ""
				cfg.GitToken = ""
				cfg.LLMProcessorURL = "http://llm-processor:8080"
				cfg.RAGAPIURLInternal = ""
				cfg.RAGAPIURLExternal = ""
				cfg.WeaviateURL = ""
				return cfg
			},
			description: "LLM should be detected as enabled",
			expectGit:   false,
			expectLLM:   true,
			expectRAG:   false,
		},
		{
			name: "RAG enabled by internal URL",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.GitRepoURL = ""
				cfg.GitToken = ""
				cfg.LLMProcessorURL = ""
				cfg.RAGAPIURLInternal = "http://rag-api:5001"
				cfg.RAGAPIURLExternal = ""
				cfg.WeaviateURL = ""
				return cfg
			},
			description: "RAG should be detected as enabled by internal URL",
			expectGit:   false,
			expectLLM:   false,
			expectRAG:   true,
		},
		{
			name: "RAG enabled by external URL",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.GitRepoURL = ""
				cfg.GitToken = ""
				cfg.LLMProcessorURL = ""
				cfg.RAGAPIURLInternal = ""
				cfg.RAGAPIURLExternal = "http://external-rag:5001"
				cfg.WeaviateURL = ""
				return cfg
			},
			description: "RAG should be detected as enabled by external URL",
			expectGit:   false,
			expectLLM:   false,
			expectRAG:   true,
		},
		{
			name: "RAG enabled by Weaviate URL",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.GitRepoURL = ""
				cfg.GitToken = ""
				cfg.LLMProcessorURL = ""
				cfg.RAGAPIURLInternal = ""
				cfg.RAGAPIURLExternal = ""
				cfg.WeaviateURL = "http://weaviate:8080"
				return cfg
			},
			description: "RAG should be detected as enabled by Weaviate URL",
			expectGit:   false,
			expectLLM:   false,
			expectRAG:   true,
		},
		{
			name: "all features enabled",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.GitRepoURL = "https://github.com/test/repo.git"
				cfg.GitToken = "github_token_123"
				cfg.LLMProcessorURL = "http://llm-processor:8080"
				cfg.RAGAPIURLInternal = "http://rag-api:5001"
				cfg.RAGAPIURLExternal = "http://external-rag:5001"
				cfg.WeaviateURL = "http://weaviate:8080"
				return cfg
			},
			description: "All features should be detected as enabled",
			expectGit:   true,
			expectLLM:   true,
			expectRAG:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.setupConfig()

			assert.Equal(t, tt.expectGit, cfg.isGitFeatureEnabled(), tt.description+" - Git detection")
			assert.Equal(t, tt.expectLLM, cfg.isLLMProcessingEnabled(), tt.description+" - LLM detection")
			assert.Equal(t, tt.expectRAG, cfg.isRAGFeatureEnabled(), tt.description+" - RAG detection")
		})
	}
}

func TestLoadFromEnv_EdgeCasesAndErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		setupEnv    func()
		description string
		wantErr     bool
		errMsg      string
	}{
		{
			name: "missing required OpenAI API key",
			setupEnv: func() {
				cleanupEnv(t)
				// Don't set OPENAI_API_KEY
			},
			description: "Missing required field should fail validation",
			wantErr:     true,
			errMsg:      "OPENAI_API_KEY is required",
		},
		{
			name: "git feature partially configured via env",
			setupEnv: func() {
				cleanupEnv(t)
				os.Setenv("OPENAI_API_KEY", "sk-test-key")
				os.Setenv("GIT_TOKEN", "github_token_123")
				// Don't set GIT_REPO_URL - this should trigger validation error
			},
			description: "Git token without repo URL should fail validation",
			wantErr:     true,
			errMsg:      "GIT_REPO_URL is required when Git features are enabled",
		},
		{
			name: "RAG feature partially configured via env",
			setupEnv: func() {
				cleanupEnv(t)
				os.Setenv("OPENAI_API_KEY", "sk-test-key")
				os.Setenv("WEAVIATE_URL", "http://weaviate:8080")
				// Don't set RAG_API_URL - this should trigger validation error
			},
			description: "Weaviate URL without RAG API URL should fail validation",
			wantErr:     true,
			errMsg:      "RAG_API_URL_INTERNAL is required when RAG features are enabled",
		},
		{
			name: "multiple validation errors via env",
			setupEnv: func() {
				cleanupEnv(t)
				// Missing OPENAI_API_KEY
				os.Setenv("GIT_TOKEN", "github_token_123")
				// Missing GIT_REPO_URL
				os.Setenv("WEAVIATE_URL", "http://weaviate:8080")
				// Missing RAG_API_URL
			},
			description: "Multiple configuration errors should be reported",
			wantErr:     true,
			errMsg:      "configuration validation failed",
		},
		{
			name: "valid minimal configuration via env",
			setupEnv: func() {
				cleanupEnv(t)
				os.Setenv("OPENAI_API_KEY", "sk-test-key")
			},
			description: "Minimal valid configuration should succeed",
			wantErr:     false,
		},
		{
			name: "valid full configuration via env",
			setupEnv: func() {
				cleanupEnv(t)
				os.Setenv("OPENAI_API_KEY", "sk-test-key")
				os.Setenv("GIT_REPO_URL", "https://github.com/test/repo.git")
				os.Setenv("GIT_TOKEN", "github_token_123")
				os.Setenv("LLM_PROCESSOR_URL", "http://llm-processor:8080")
				os.Setenv("RAG_API_URL", "http://rag-api:5001")
				os.Setenv("WEAVIATE_URL", "http://weaviate:8080")
			},
			description: "Full valid configuration should succeed",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()

			cfg, err := LoadFromEnv()
			if tt.wantErr {
				assert.Error(t, err, tt.description)
				assert.Nil(t, cfg, tt.description)
				assert.Contains(t, err.Error(), tt.errMsg, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
				assert.NotNil(t, cfg, tt.description)
			}
		})
	}
}

func TestConfig_Validate_DescriptiveErrorMessages(t *testing.T) {
	// Test that error messages are descriptive and helpful
	cfg := DefaultConfig()
	cfg.OpenAIAPIKey = ""                       // Missing required
	cfg.GitToken = "token"                       // Enables git but URL missing
	cfg.WeaviateURL = "http://weaviate:8080"     // Enables RAG but internal URL missing

	err := cfg.Validate()
	require.Error(t, err)

	errMsg := err.Error()
	
	// Check that all expected errors are present
	assert.Contains(t, errMsg, "OPENAI_API_KEY is required")
	assert.Contains(t, errMsg, "GIT_REPO_URL is required when Git features are enabled")
	assert.Contains(t, errMsg, "RAG_API_URL_INTERNAL is required when RAG features are enabled")
	
	// Check that the error message indicates it's a validation failure
	assert.Contains(t, errMsg, "configuration validation failed")
}