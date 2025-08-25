package config

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/nephoran-intent-operator/pkg/testutil"
)

func TestDefaultConfig(t *testing.T) {
	// Ensure test isolation
	envGuard := testutil.NewEnvironmentGuard(t)
	testutil.CleanupCommonEnvVars(t)
	
	ctx, cancel := testutil.ContextWithTimeout(t)
	defer cancel()
	
	// Test with clean environment
	select {
	case <-ctx.Done():
		t.Fatal("test timeout")
	default:
		cfg := DefaultConfig()
		require.NotNil(t, cfg, "default config should not be nil")

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
	
	// Verify cleanup
	_ = envGuard
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
	envGuard := testutil.NewEnvironmentGuard(t)
	testutil.CleanupCommonEnvVars(t)
	
	ctx, cancel := testutil.ContextWithTimeout(t)
	defer cancel()

	// Set required environment variables using environment guard
	envGuard.Set("OPENAI_API_KEY", "sk-test-api-key")

	select {
	case <-ctx.Done():
		t.Fatal("test timeout")
	default:
		cfg, err := LoadFromEnv()
		require.NoError(t, err)
		require.NotNil(t, cfg, "config should not be nil")
		assert.Equal(t, "sk-test-api-key", cfg.OpenAIAPIKey)
	}
}

func TestLoadFromEnv_EnvironmentOverrides(t *testing.T) {
	envGuard := testutil.NewEnvironmentGuard(t)
	testutil.CleanupCommonEnvVars(t)
	
	ctx, cancel := testutil.ContextWithTimeout(t)
	defer cancel()

	// Set test environment variables using environment guard
	envVars := map[string]string{
		"METRICS_ADDR":           ":9090",
		"PROBE_ADDR":             ":9091",
		"ENABLE_LEADER_ELECTION": "true",
		"LLM_PROCESSOR_URL":      "http://custom-llm:8080",
		"LLM_PROCESSOR_TIMEOUT":  "60s",
		"RAG_API_URL":            "http://custom-rag:5001",
		"RAG_API_URL_EXTERNAL":   "http://custom-rag-external:5001",
		"RAG_API_TIMEOUT":        "45s",
		"GIT_REPO_URL":           "https://github.com/custom/repo.git",
		"GIT_TOKEN":              "custom_token",
		"GIT_BRANCH":             "develop",
		"WEAVIATE_URL":           "http://custom-weaviate:8080",
		"WEAVIATE_INDEX":         "custom_index",
		"OPENAI_API_KEY":         "sk-custom-api-key",
		"OPENAI_MODEL":           "gpt-4",
		"OPENAI_EMBEDDING_MODEL": "text-embedding-ada-002",
		"NAMESPACE":              "custom-namespace",
		"CRD_PATH":               "custom/crds",
	}

	for key, value := range envVars {
		envGuard.Set(key, value)
	}

	select {
	case <-ctx.Done():
		t.Fatal("test timeout")
	default:
		cfg, err := LoadFromEnv()
		require.NoError(t, err)
		require.NotNil(t, cfg, "config should not be nil")

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
}

func TestLoadFromEnv_InvalidConfiguration(t *testing.T) {
	envGuard := testutil.NewEnvironmentGuard(t)
	testutil.CleanupCommonEnvVars(t)
	
	ctx, cancel := testutil.ContextWithTimeout(t)
	defer cancel()

	// Don't set required OPENAI_API_KEY
	select {
	case <-ctx.Done():
		t.Fatal("test timeout")
	default:
		cfg, err := LoadFromEnv()
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "OPENAI_API_KEY is required")
	}
	_ = envGuard
}

func TestLoadFromEnv_InvalidDurationValues(t *testing.T) {
	envGuard := testutil.NewEnvironmentGuard(t)
	testutil.CleanupCommonEnvVars(t)

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
			// Use dedicated environment guard for subtest
			subEnvGuard := testutil.NewEnvironmentGuard(t)
			testutil.CleanupCommonEnvVars(t)

			// Set required environment variables
			subEnvGuard.Set("OPENAI_API_KEY", "sk-test-api-key")
			subEnvGuard.Set(tt.envVar, tt.value)

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
			// Use dedicated environment guard for subtest
			subEnvGuard := testutil.NewEnvironmentGuard(t)
			testutil.CleanupCommonEnvVars(t)

			// Set required environment variables
			subEnvGuard.Set("OPENAI_API_KEY", "sk-test-api-key")
			subEnvGuard.Set(tt.envVar, tt.value)

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

// TestEnvironmentVariables tests the 8 specific environment variables
func TestEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name      string
		envVar    string
		testCases []struct {
			name     string
			value    string
			expected interface{}
			isError  bool
		}
	}{
		{
			name:   "ENABLE_NETWORK_INTENT",
			envVar: "ENABLE_NETWORK_INTENT",
			testCases: []struct {
				name     string
				value    string
				expected interface{}
				isError  bool
			}{
				{"default when unset", "", true, false}, // default: true
				{"true value", "true", true, false},
				{"false value", "false", false, false},
				{"1 value", "1", true, false},
				{"0 value", "0", false, false},
				{"yes value", "yes", true, false},
				{"no value", "no", false, false},
				{"on value", "on", true, false},
				{"off value", "off", false, false},
				{"enabled value", "enabled", true, false},
				{"disabled value", "disabled", false, false},
				{"invalid value uses default", "invalid", true, false}, // uses default
				{"empty string uses default", "", true, false},
				{"mixed case true", "TRUE", true, false},
				{"mixed case false", "False", false, false},
				{"whitespace trimmed", "  true  ", true, false},
			},
		},
		{
			name:   "ENABLE_LLM_INTENT",
			envVar: "ENABLE_LLM_INTENT",
			testCases: []struct {
				name     string
				value    string
				expected interface{}
				isError  bool
			}{
				{"default when unset", "", false, false}, // default: false
				{"true value", "true", true, false},
				{"false value", "false", false, false},
				{"1 value", "1", true, false},
				{"0 value", "0", false, false},
				{"invalid value uses default", "maybe", false, false}, // uses default
				{"whitespace trimmed", "  false  ", false, false},
			},
		},
		{
			name:   "LLM_TIMEOUT_SECS",
			envVar: "LLM_TIMEOUT_SECS",
			testCases: []struct {
				name     string
				value    string
				expected interface{}
				isError  bool
			}{
				{"default when unset", "", 15 * time.Second, false}, // default: 15s
				{"valid seconds", "30", 30 * time.Second, false},
				{"minimum value", "1", 1 * time.Second, false},
				{"large value", "300", 300 * time.Second, false},
				{"zero value uses default", "0", 15 * time.Second, false}, // uses default for non-positive
				{"negative value uses default", "-5", 15 * time.Second, false},
				{"invalid string uses default", "invalid", 15 * time.Second, false},
				{"empty string uses default", "", 15 * time.Second, false},
				{"float string uses default", "15.5", 15 * time.Second, false},             // Atoi fails on float
				{"whitespace not trimmed uses default", "  60  ", 15 * time.Second, false}, // GetEnvOrDefault doesn't trim, strconv.Atoi fails
			},
		},
		{
			name:   "LLM_MAX_RETRIES",
			envVar: "LLM_MAX_RETRIES",
			testCases: []struct {
				name     string
				value    string
				expected interface{}
				isError  bool
			}{
				{"default when unset", "", 2, false}, // default: 2
				{"valid retries", "5", 5, false},
				{"zero retries", "0", 0, false},
				{"maximum retries", "10", 10, false},
				{"negative value accepted", "-1", -1, false}, // GetIntEnv accepts negative values
				{"invalid string uses default", "invalid", 2, false},
				{"empty string uses default", "", 2, false},
				{"whitespace trimmed", "  3  ", 3, false},
			},
		},
		{
			name:   "LLM_CACHE_MAX_ENTRIES",
			envVar: "LLM_CACHE_MAX_ENTRIES",
			testCases: []struct {
				name     string
				value    string
				expected interface{}
				isError  bool
			}{
				{"default when unset", "", 512, false}, // default: 512
				{"valid entries", "1024", 1024, false},
				{"zero entries", "0", 0, false},
				{"large entries", "10000", 10000, false},
				{"negative value accepted", "-100", -100, false}, // GetIntEnv accepts negative values
				{"invalid string uses default", "many", 512, false},
				{"empty string uses default", "", 512, false},
				{"whitespace trimmed", "  256  ", 256, false},
			},
		},
		{
			name:   "HTTP_MAX_BODY",
			envVar: "HTTP_MAX_BODY",
			testCases: []struct {
				name     string
				value    string
				expected interface{}
				isError  bool
			}{
				{"default when unset", "", int64(1048576), false}, // default: 1MB
				{"valid size", "2097152", int64(2097152), false},  // 2MB
				{"zero size", "0", int64(0), false},
				{"large size", "10485760", int64(10485760), false},        // 10MB
				{"negative value accepted", "-1000", int64(-1000), false}, // GetInt64Env accepts negative values
				{"invalid string uses default", "unlimited", int64(1048576), false},
				{"empty string uses default", "", int64(1048576), false},
				{"whitespace trimmed", "  524288  ", int64(524288), false}, // 512KB
			},
		},
		{
			name:   "METRICS_ENABLED",
			envVar: "METRICS_ENABLED",
			testCases: []struct {
				name     string
				value    string
				expected interface{}
				isError  bool
			}{
				{"default when unset", "", false, false}, // default: false
				{"true value", "true", true, false},
				{"false value", "false", false, false},
				{"1 value", "1", true, false},
				{"0 value", "0", false, false},
				{"invalid value uses default", "maybe", false, false},
				{"whitespace trimmed", "  true  ", true, false},
			},
		},
		{
			name:   "METRICS_ALLOWED_IPS",
			envVar: "METRICS_ALLOWED_IPS",
			testCases: []struct {
				name     string
				value    string
				expected interface{}
				isError  bool
			}{
				{"default when unset", "", []string{}, false}, // default: empty
				{"single IP", "127.0.0.1", []string{"127.0.0.1"}, false},
				{"multiple IPs", "127.0.0.1,192.168.1.1,10.0.0.1", []string{"127.0.0.1", "192.168.1.1", "10.0.0.1"}, false},
				{"IPs with spaces", " 127.0.0.1 , 192.168.1.1 , 10.0.0.1 ", []string{"127.0.0.1", "192.168.1.1", "10.0.0.1"}, false},
				{"wildcard", "*", []string{"*"}, false},
				{"mixed IPs and wildcard", "127.0.0.1,*", []string{"127.0.0.1", "*"}, false},
				{"empty string uses default", "", []string{}, false},
				{"comma only gives empty", ",", []string{}, false},
				{"spaces only gives empty", "   ", []string{}, false},
				{"empty items filtered", "127.0.0.1,,192.168.1.1", []string{"127.0.0.1", "192.168.1.1"}, false},
				{"trailing comma", "127.0.0.1,", []string{"127.0.0.1"}, false},
				{"leading comma", ",127.0.0.1", []string{"127.0.0.1"}, false},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, tc := range test.testCases {
				t.Run(tc.name, func(t *testing.T) {
					// Clean environment
					testEnvGuard := testutil.NewEnvironmentGuard(t)
					testutil.CleanupCommonEnvVars(t)

					// Set required API key
					testEnvGuard.Set("OPENAI_API_KEY", "sk-test-key")

					// Set test environment variable if value is not empty (to test default behavior)
					if tc.value != "" {
						testEnvGuard.Set(test.envVar, tc.value)
					}

					cfg, err := LoadFromEnv()

					if tc.isError {
						assert.Error(t, err)
						return
					}

					require.NoError(t, err)
					require.NotNil(t, cfg)

					// Check the specific field based on the environment variable
					switch test.envVar {
					case "ENABLE_NETWORK_INTENT":
						assert.Equal(t, tc.expected, cfg.EnableNetworkIntent)
					case "ENABLE_LLM_INTENT":
						assert.Equal(t, tc.expected, cfg.EnableLLMIntent)
					case "LLM_TIMEOUT_SECS":
						assert.Equal(t, tc.expected, cfg.LLMTimeout)
					case "LLM_MAX_RETRIES":
						assert.Equal(t, tc.expected, cfg.LLMMaxRetries)
					case "LLM_CACHE_MAX_ENTRIES":
						assert.Equal(t, tc.expected, cfg.LLMCacheMaxEntries)
					case "HTTP_MAX_BODY":
						assert.Equal(t, tc.expected, cfg.HTTPMaxBody)
					case "METRICS_ENABLED":
						assert.Equal(t, tc.expected, cfg.MetricsEnabled)
					case "METRICS_ALLOWED_IPS":
						assert.Equal(t, tc.expected, cfg.MetricsAllowedIPs)
					}
				})
			}
		})
	}
}

// TestEnvironmentVariablesDefaultValues specifically tests that default values are correct
func TestEnvironmentVariablesDefaultValues(t *testing.T) {
	envGuard := testutil.NewEnvironmentGuard(t)
	testutil.CleanupCommonEnvVars(t)
	envGuard.Set("OPENAI_API_KEY", "sk-test-key")

	cfg, err := LoadFromEnv()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Test all 8 environment variables have correct defaults
	assert.Equal(t, true, cfg.EnableNetworkIntent, "ENABLE_NETWORK_INTENT default should be true")
	assert.Equal(t, false, cfg.EnableLLMIntent, "ENABLE_LLM_INTENT default should be false")
	assert.Equal(t, 15*time.Second, cfg.LLMTimeout, "LLM_TIMEOUT_SECS default should be 15 seconds")
	assert.Equal(t, 2, cfg.LLMMaxRetries, "LLM_MAX_RETRIES default should be 2")
	assert.Equal(t, 512, cfg.LLMCacheMaxEntries, "LLM_CACHE_MAX_ENTRIES default should be 512")
	assert.Equal(t, int64(1048576), cfg.HTTPMaxBody, "HTTP_MAX_BODY default should be 1048576 (1MB)")
	assert.Equal(t, false, cfg.MetricsEnabled, "METRICS_ENABLED default should be false")
	assert.Equal(t, []string{}, cfg.MetricsAllowedIPs, "METRICS_ALLOWED_IPS default should be empty slice")
}

// TestEnvironmentVariablesEdgeCases tests edge cases for the 8 environment variables
func TestEnvironmentVariablesEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setupEnv    func()
		validate    func(t *testing.T, cfg *Config)
		expectError bool
	}{
		{
			name: "LLM_TIMEOUT_SECS with very large value",
			setupEnv: func() {
				os.Setenv("LLM_TIMEOUT_SECS", "86400") // 24 hours
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 86400*time.Second, cfg.LLMTimeout)
			},
		},
		{
			name: "HTTP_MAX_BODY with maximum int64 value",
			setupEnv: func() {
				os.Setenv("HTTP_MAX_BODY", "9223372036854775807") // max int64
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, int64(9223372036854775807), cfg.HTTPMaxBody)
			},
		},
		{
			name: "METRICS_ALLOWED_IPS with complex whitespace",
			setupEnv: func() {
				os.Setenv("METRICS_ALLOWED_IPS", "\t127.0.0.1\n,\r192.168.1.1\t,   10.0.0.1   \n")
			},
			validate: func(t *testing.T, cfg *Config) {
				expected := []string{"127.0.0.1", "192.168.1.1", "10.0.0.1"}
				assert.Equal(t, expected, cfg.MetricsAllowedIPs)
			},
		},
		{
			name: "All boolean env vars with enable/disable values",
			setupEnv: func() {
				os.Setenv("ENABLE_NETWORK_INTENT", "disable")
				os.Setenv("ENABLE_LLM_INTENT", "enable")
				os.Setenv("METRICS_ENABLED", "enabled")
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.False(t, cfg.EnableNetworkIntent)
				assert.True(t, cfg.EnableLLMIntent)
				assert.True(t, cfg.MetricsEnabled)
			},
		},
		{
			name: "Integer env vars with boundary values",
			setupEnv: func() {
				os.Setenv("LLM_MAX_RETRIES", "0")
				os.Setenv("LLM_CACHE_MAX_ENTRIES", "1")
				os.Setenv("HTTP_MAX_BODY", "0")
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 0, cfg.LLMMaxRetries)
				assert.Equal(t, 1, cfg.LLMCacheMaxEntries)
				assert.Equal(t, int64(0), cfg.HTTPMaxBody)
			},
		},
		{
			name: "METRICS_ALLOWED_IPS with IPv6 addresses",
			setupEnv: func() {
				os.Setenv("METRICS_ALLOWED_IPS", "::1,2001:db8::1,127.0.0.1")
			},
			validate: func(t *testing.T, cfg *Config) {
				expected := []string{"::1", "2001:db8::1", "127.0.0.1"}
				assert.Equal(t, expected, cfg.MetricsAllowedIPs)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edgeEnvGuard := testutil.NewEnvironmentGuard(t)
			testutil.CleanupCommonEnvVars(t)
			edgeEnvGuard.Set("OPENAI_API_KEY", "sk-test-key")

			tt.setupEnv()

			cfg, err := LoadFromEnv()
			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cfg)

			tt.validate(t, cfg)
		})
	}
}

// TestSpecialLLMTimeoutSecsConversion tests the specific conversion logic for LLM_TIMEOUT_SECS
func TestSpecialLLMTimeoutSecsConversion(t *testing.T) {
	tests := []struct {
		name           string
		envValue       string
		expectedResult time.Duration
		description    string
	}{
		{
			name:           "valid positive integer",
			envValue:       "30",
			expectedResult: 30 * time.Second,
			description:    "Should convert seconds to time.Duration",
		},
		{
			name:           "zero value",
			envValue:       "0",
			expectedResult: 15 * time.Second, // default
			description:    "Should use default for zero or negative values",
		},
		{
			name:           "negative value",
			envValue:       "-5",
			expectedResult: 15 * time.Second, // default
			description:    "Should use default for negative values",
		},
		{
			name:           "invalid format",
			envValue:       "not-a-number",
			expectedResult: 15 * time.Second, // default
			description:    "Should use default for invalid format",
		},
		{
			name:           "float value",
			envValue:       "15.5",
			expectedResult: 15 * time.Second, // default because Atoi fails
			description:    "Should use default for float values (Atoi fails)",
		},
		{
			name:           "empty string",
			envValue:       "",
			expectedResult: 15 * time.Second, // default
			description:    "Should use default for empty string",
		},
		{
			name:           "large value",
			envValue:       "3600",
			expectedResult: 3600 * time.Second, // 1 hour
			description:    "Should handle large values correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeoutEnvGuard := testutil.NewEnvironmentGuard(t)
			testutil.CleanupCommonEnvVars(t)
			timeoutEnvGuard.Set("OPENAI_API_KEY", "sk-test-key")
			timeoutEnvGuard.Set("LLM_TIMEOUT_SECS", tt.envValue)

			cfg, err := LoadFromEnv()
			require.NoError(t, err)
			require.NotNil(t, cfg)

			assert.Equal(t, tt.expectedResult, cfg.LLMTimeout, tt.description)
		})
	}
}

// NOTE: The old cleanup functions have been replaced with testutil.NewEnvironmentGuard
// and testutil.CleanupCommonEnvVars for better test isolation and reliability.

// Helper function for backward compatibility
func cleanupEnv(t *testing.T) {
	testutil.CleanupCommonEnvVars(t)
}