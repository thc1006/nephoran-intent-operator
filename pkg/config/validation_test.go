package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigValidateFeatureRequirements tests the enhanced validation logic
// for specific feature requirements based on configuration state
func TestConfigValidateFeatureRequirements(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func() *Config
		wantErr     bool
		errContains []string
	}{
		{
			name: "missing GitRepoURL when git features enabled by token",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.GitRepoURL = ""
				cfg.GitToken = "github_token_123"
				return cfg
			},
			wantErr:     true,
			errContains: []string{"GIT_REPO_URL is required when Git features are enabled"},
		},
		{
			name: "missing RAGAPIURLInternal when RAG features enabled by WeaviateURL",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.RAGAPIURLInternal = ""
				cfg.RAGAPIURLExternal = ""
				cfg.WeaviateURL = "http://weaviate:8080"
				return cfg
			},
			wantErr:     true,
			errContains: []string{"RAG_API_URL_INTERNAL is required when RAG features are enabled"},
		},
		{
			name: "missing RAGAPIURLInternal when RAG features enabled by external URL",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.RAGAPIURLInternal = ""
				cfg.RAGAPIURLExternal = "http://external-rag:5001"
				cfg.WeaviateURL = ""
				return cfg
			},
			wantErr:     true,
			errContains: []string{"RAG_API_URL_INTERNAL is required when RAG features are enabled"},
		},
		{
			name: "multiple missing required fields",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = ""
				cfg.GitRepoURL = ""
				cfg.GitToken = "github_token_123"
				cfg.RAGAPIURLInternal = ""
				cfg.WeaviateURL = "http://weaviate:8080"
				return cfg
			},
			wantErr: true,
			errContains: []string{
				"OPENAI_API_KEY is required",
				"GIT_REPO_URL is required when Git features are enabled",
				"RAG_API_URL_INTERNAL is required when RAG features are enabled",
			},
		},
		{
			name: "all features properly configured",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.GitRepoURL = "https://github.com/test/repo.git"
				cfg.GitToken = "github_token_123"
				cfg.LLMProcessorURL = "http://llm-processor:8080"
				cfg.RAGAPIURLInternal = "http://rag-api:5001"
				cfg.WeaviateURL = "http://weaviate:8080"
				return cfg
			},
			wantErr: false,
		},
		{
			name: "features disabled by empty URLs",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.GitRepoURL = ""
				cfg.GitToken = ""
				cfg.LLMProcessorURL = ""
				cfg.RAGAPIURLInternal = ""
				cfg.RAGAPIURLExternal = ""
				cfg.WeaviateURL = ""
				return cfg
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.setupConfig()
			err := cfg.Validate()

			if tt.wantErr {
				require.Error(t, err)
				for _, errText := range tt.errContains {
					assert.Contains(t, err.Error(), errText, "Expected error to contain: %s", errText)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestConfigFeatureDetection tests the helper methods that detect if features are enabled
func TestConfigFeatureDetection(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func() *Config
		expectGit   bool
		expectLLM   bool
		expectRAG   bool
	}{
		{
			name: "git enabled by URL only",
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
			expectGit: true,
			expectLLM: false,
			expectRAG: false,
		},
		{
			name: "git enabled by token only",
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
			expectGit: true,
			expectLLM: false,
			expectRAG: false,
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
			expectGit: false,
			expectLLM: true,
			expectRAG: false,
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
			expectGit: false,
			expectLLM: false,
			expectRAG: true,
		},
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
			expectGit: false,
			expectLLM: false,
			expectRAG: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.setupConfig()

			assert.Equal(t, tt.expectGit, cfg.isGitFeatureEnabled(),
				"Git feature detection failed")
			assert.Equal(t, tt.expectLLM, cfg.isLLMProcessingEnabled(),
				"LLM processing detection failed")
			assert.Equal(t, tt.expectRAG, cfg.isRAGFeatureEnabled(),
				"RAG feature detection failed")
		})
	}
}
