package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, ":8080", cfg.MetricsAddr)
	assert.Equal(t, ":8081", cfg.ProbeAddr)
	assert.False(t, cfg.EnableLeaderElection)
	assert.Equal(t, "http://llm-processor.default.svc.cluster.local:8080", cfg.LLMProcessorURL)
	assert.Equal(t, 30*time.Second, cfg.LLMProcessorTimeout)
	assert.Equal(t, "http://rag-api.default.svc.cluster.local:5001", cfg.RAGAPIURLInternal)
	assert.Equal(t, "http://localhost:5001", cfg.RAGAPIURLExternal)
	assert.Equal(t, 30*time.Second, cfg.RAGAPITimeout)
	assert.Equal(t, "main", cfg.GitBranch)
	assert.Equal(t, "http://weaviate.default.svc.cluster.local:8080", cfg.WeaviateURL)
	assert.Equal(t, "telecom_knowledge", cfg.WeaviateIndex)
	assert.Equal(t, "gpt-4o-mini", cfg.OpenAIModel)
	assert.Equal(t, "text-embedding-3-large", cfg.OpenAIEmbeddingModel)
	assert.Equal(t, "default", cfg.Namespace)
	assert.Equal(t, "deployments/crds", cfg.CRDPath)
}

func TestConfig_Validate_RequiredFields(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func() *Config
		wantErr     bool
		errMsg      string
	}{
		{
			name: "valid config with all required fields",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				return cfg
			},
			wantErr: false,
		},
		{
			name: "invalid config missing OpenAI API key",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = ""
				return cfg
			},
			wantErr: true,
			errMsg:  "OPENAI_API_KEY is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.setupConfig()
			err := cfg.Validate()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_Validate_EnhancedValidation(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func() *Config
		description string
		wantErr     bool
		errMsg      string
	}{
		{
			name: "valid config with git features enabled",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.GitRepoURL = "https://github.com/test/repo.git"
				cfg.GitToken = "github_token_123"
				return cfg
			},
			description: "Git features are enabled when GitRepoURL is provided",
			wantErr:     false,
		},
		{
			name: "valid config with git features disabled",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.GitRepoURL = ""
				cfg.GitToken = ""
				return cfg
			},
			description: "Git features are disabled when GitRepoURL is empty",
			wantErr:     false,
		},
		{
			name: "valid config with LLM processing enabled",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.LLMProcessorURL = "http://llm-processor:8080"
				return cfg
			},
			description: "LLM processing is enabled when LLMProcessorURL is provided",
			wantErr:     false,
		},
		{
			name: "valid config with LLM processing disabled",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.LLMProcessorURL = ""
				return cfg
			},
			description: "LLM processing is disabled when LLMProcessorURL is empty",
			wantErr:     false,
		},
		{
			name: "valid config with RAG features enabled",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.RAGAPIURLInternal = "http://rag-api:5001"
				cfg.WeaviateURL = "http://weaviate:8080"
				return cfg
			},
			description: "RAG features are enabled when RAGAPIURLInternal is provided",
			wantErr:     false,
		},
		{
			name: "valid config with RAG features disabled",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.RAGAPIURLInternal = ""
				cfg.WeaviateURL = ""
				return cfg
			},
			description: "RAG features are disabled when RAGAPIURLInternal is empty",
			wantErr:     false,
		},
		{
			name: "valid config with all features enabled",
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
			description: "All features can be enabled simultaneously",
			wantErr:     false,
		},
		{
			name: "valid config with all features disabled",
			setupConfig: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAIAPIKey = "sk-test-api-key"
				cfg.GitRepoURL = ""
				cfg.GitToken = ""
				cfg.LLMProcessorURL = ""
				cfg.RAGAPIURLInternal = ""
				cfg.WeaviateURL = ""
				return cfg
			},
			description: "All features can be disabled simultaneously",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.setupConfig()
			err := cfg.Validate()

			if tt.wantErr {
				assert.Error(t, err, tt.description)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg, tt.description)
				}
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

func TestLoadFromEnv_ValidConfiguration(t *testing.T) {
	// Clean environment
	cleanupEnv(t)

	// Set required environment variables
	os.Setenv("OPENAI_API_KEY", "sk-test-api-key")

	cfg, err := LoadFromEnv()
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "sk-test-api-key", cfg.OpenAIAPIKey)
}

func TestLoadFromEnv_EnvironmentOverrides(t *testing.T) {
	// Clean environment
	cleanupEnv(t)

	// Set test environment variables
	envVars := map[string]string{
		"METRICS_ADDR":              ":9090",
		"PROBE_ADDR":                ":9091",
		"ENABLE_LEADER_ELECTION":    "true",
		"LLM_PROCESSOR_URL":         "http://custom-llm:8080",
		"LLM_PROCESSOR_TIMEOUT":     "60s",
		"RAG_API_URL":               "http://custom-rag:5001",
		"RAG_API_URL_EXTERNAL":      "http://custom-rag-external:5001",
		"RAG_API_TIMEOUT":           "45s",
		"GIT_REPO_URL":              "https://github.com/custom/repo.git",
		"GIT_TOKEN":                 "custom_token",
		"GIT_BRANCH":                "develop",
		"WEAVIATE_URL":              "http://custom-weaviate:8080",
		"WEAVIATE_INDEX":            "custom_index",
		"OPENAI_API_KEY":            "sk-custom-api-key",
		"OPENAI_MODEL":              "gpt-4",
		"OPENAI_EMBEDDING_MODEL":    "text-embedding-ada-002",
		"NAMESPACE":                 "custom-namespace",
		"CRD_PATH":                  "custom/crds",
	}

	for key, value := range envVars {
		os.Setenv(key, value)
	}

	cfg, err := LoadFromEnv()
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify all environment variables were loaded correctly
	assert.Equal(t, ":9090", cfg.MetricsAddr)
	assert.Equal(t, ":9091", cfg.ProbeAddr)
	assert.True(t, cfg.EnableLeaderElection)
	assert.Equal(t, "http://custom-llm:8080", cfg.LLMProcessorURL)
	assert.Equal(t, 60*time.Second, cfg.LLMProcessorTimeout)
	assert.Equal(t, "http://custom-rag:5001", cfg.RAGAPIURLInternal)
	assert.Equal(t, "http://custom-rag-external:5001", cfg.RAGAPIURLExternal)
	assert.Equal(t, 45*time.Second, cfg.RAGAPITimeout)
	assert.Equal(t, "https://github.com/custom/repo.git", cfg.GitRepoURL)
	assert.Equal(t, "custom_token", cfg.GitToken)
	assert.Equal(t, "develop", cfg.GitBranch)
	assert.Equal(t, "http://custom-weaviate:8080", cfg.WeaviateURL)
	assert.Equal(t, "custom_index", cfg.WeaviateIndex)
	assert.Equal(t, "sk-custom-api-key", cfg.OpenAIAPIKey)
	assert.Equal(t, "gpt-4", cfg.OpenAIModel)
	assert.Equal(t, "text-embedding-ada-002", cfg.OpenAIEmbeddingModel)
	assert.Equal(t, "custom-namespace", cfg.Namespace)
	assert.Equal(t, "custom/crds", cfg.CRDPath)
}

func TestLoadFromEnv_InvalidConfiguration(t *testing.T) {
	// Clean environment
	cleanupEnv(t)

	// Don't set required OPENAI_API_KEY
	cfg, err := LoadFromEnv()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "OPENAI_API_KEY is required")
}

func TestLoadFromEnv_InvalidDurationValues(t *testing.T) {
	// Clean environment
	cleanupEnv(t)

	tests := []struct {
		name    string
		envVar  string
		value   string
		wantErr bool
	}{
		{
			name:    "valid duration",
			envVar:  "LLM_PROCESSOR_TIMEOUT",
			value:   "30s",
			wantErr: false,
		},
		{
			name:    "invalid duration format",
			envVar:  "LLM_PROCESSOR_TIMEOUT",
			value:   "invalid",
			wantErr: false, // Should use default value
		},
		{
			name:    "valid RAG timeout",
			envVar:  "RAG_API_TIMEOUT",
			value:   "1m",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean environment first
			cleanupEnv(t)

			// Set required environment variables
			os.Setenv("OPENAI_API_KEY", "sk-test-api-key")
			os.Setenv(tt.envVar, tt.value)

			cfg, err := LoadFromEnv()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
			}
		})
	}
}

func TestLoadFromEnv_InvalidBooleanValues(t *testing.T) {
	// Clean environment
	cleanupEnv(t)

	tests := []struct {
		name         string
		envVar       string
		value        string
		expectedBool bool
	}{
		{
			name:         "true string",
			envVar:       "ENABLE_LEADER_ELECTION",
			value:        "true",
			expectedBool: true,
		},
		{
			name:         "false string",
			envVar:       "ENABLE_LEADER_ELECTION",
			value:        "false",
			expectedBool: false,
		},
		{
			name:         "invalid boolean uses default",
			envVar:       "ENABLE_LEADER_ELECTION",
			value:        "invalid",
			expectedBool: false, // Default value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean environment first
			cleanupEnv(t)

			// Set required environment variables
			os.Setenv("OPENAI_API_KEY", "sk-test-api-key")
			os.Setenv(tt.envVar, tt.value)

			cfg, err := LoadFromEnv()
			require.NoError(t, err)
			assert.NotNil(t, cfg)
			assert.Equal(t, tt.expectedBool, cfg.EnableLeaderElection)
		})
	}
}

func TestConfig_GetRAGAPIURL(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RAGAPIURLInternal = "http://internal-rag:5001"
	cfg.RAGAPIURLExternal = "http://external-rag:5001"

	tests := []struct {
		name        string
		useInternal bool
		expected    string
	}{
		{
			name:        "use internal URL",
			useInternal: true,
			expected:    "http://internal-rag:5001",
		},
		{
			name:        "use external URL",
			useInternal: false,
			expected:    "http://external-rag:5001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cfg.GetRAGAPIURL(tt.useInternal)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// cleanupEnv cleans up environment variables that might affect tests
func cleanupEnv(t *testing.T) {
	envVars := []string{
		"METRICS_ADDR",
		"PROBE_ADDR",
		"ENABLE_LEADER_ELECTION",
		"LLM_PROCESSOR_URL",
		"LLM_PROCESSOR_TIMEOUT",
		"RAG_API_URL",
		"RAG_API_URL_EXTERNAL",
		"RAG_API_TIMEOUT",
		"GIT_REPO_URL",
		"GIT_TOKEN",
		"GIT_BRANCH",
		"WEAVIATE_URL",
		"WEAVIATE_INDEX",
		"OPENAI_API_KEY",
		"OPENAI_MODEL",
		"OPENAI_EMBEDDING_MODEL",
		"NAMESPACE",
		"CRD_PATH",
	}

	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}

	t.Cleanup(func() {
		for _, envVar := range envVars {
			os.Unsetenv(envVar)
		}
	})
}