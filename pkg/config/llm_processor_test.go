package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultLLMProcessorConfig(t *testing.T) {
	cfg := DefaultLLMProcessorConfig()

	// Service Configuration
	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "v2.0.0", cfg.ServiceVersion)
	assert.Equal(t, 30*time.Second, cfg.GracefulShutdown)

	// LLM Configuration
	assert.Equal(t, "rag", cfg.LLMBackendType)
	assert.Equal(t, "gpt-4o-mini", cfg.LLMModelName)
	assert.Equal(t, 60*time.Second, cfg.LLMTimeout)
	assert.Equal(t, 2048, cfg.LLMMaxTokens)

	// RAG Configuration
	assert.Equal(t, "http://rag-api:5001", cfg.RAGAPIURL)
	assert.Equal(t, 30*time.Second, cfg.RAGTimeout)
	assert.True(t, cfg.RAGEnabled)

	// Streaming Configuration
	assert.True(t, cfg.StreamingEnabled)
	assert.Equal(t, 100, cfg.MaxConcurrentStreams)
	assert.Equal(t, 5*time.Minute, cfg.StreamTimeout)

	// Context Management
	assert.True(t, cfg.EnableContextBuilder)
	assert.Equal(t, 6000, cfg.MaxContextTokens)
	assert.Equal(t, 5*time.Minute, cfg.ContextTTL)

	// Security Configuration
	assert.False(t, cfg.APIKeyRequired)
	assert.True(t, cfg.CORSEnabled)
	assert.Equal(t, "*", cfg.AllowedOrigins)

	// Performance Configuration
	assert.Equal(t, 30*time.Second, cfg.RequestTimeout)
	assert.Equal(t, int64(1048576), cfg.MaxRequestSize)

	// OAuth2 Authentication Configuration
	assert.False(t, cfg.AuthEnabled)
	assert.True(t, cfg.RequireAuth)
	assert.Empty(t, cfg.AdminUsers)
	assert.Empty(t, cfg.OperatorUsers)

	// Secret Management Configuration
	assert.True(t, cfg.UseKubernetesSecrets)
	assert.Equal(t, "nephoran-system", cfg.SecretNamespace)
}

func TestLLMProcessorConfig_Validate_RequiredFields(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func() *LLMProcessorConfig
		description string
		wantErr     bool
		errMsg      string
	}{
		{
			name: "valid config with mock backend",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "mock"
				cfg.LLMAPIKey = ""
				return cfg
			},
			description: "Mock backend should not require API key",
			wantErr:     false,
		},
		{
			name: "valid config with rag backend",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "rag"
				cfg.LLMAPIKey = ""
				cfg.RAGEnabled = true
				cfg.RAGAPIURL = "http://rag-api:5001"
				return cfg
			},
			description: "RAG backend should not require LLM API key",
			wantErr:     false,
		},
		{
			name: "valid config with openai backend and API key",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "openai"
				cfg.LLMAPIKey = "sk-test-api-key"
				return cfg
			},
			description: "OpenAI backend with API key should be valid",
			wantErr:     false,
		},
		{
			name: "invalid config with openai backend missing API key",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "openai"
				cfg.LLMAPIKey = ""
				return cfg
			},
			description: "OpenAI backend should require API key",
			wantErr:     true,
			errMsg:      "OPENAI_API_KEY is required for non-mock/non-rag backends",
		},
		{
			name: "invalid config with mistral backend missing API key",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "mistral"
				cfg.LLMAPIKey = ""
				return cfg
			},
			description: "Mistral backend should require API key",
			wantErr:     true,
			errMsg:      "OPENAI_API_KEY is required for non-mock/non-rag backends",
		},
		{
			name: "valid config with auth disabled",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "mock"
				cfg.AuthEnabled = false
				cfg.JWTSecretKey = ""
				return cfg
			},
			description: "Authentication disabled should not require JWT secret",
			wantErr:     false,
		},
		{
			name: "invalid config with auth enabled but missing JWT secret",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "mock"
				cfg.AuthEnabled = true
				cfg.JWTSecretKey = ""
				return cfg
			},
			description: "Authentication enabled should require JWT secret",
			wantErr:     true,
			errMsg:      "JWT_SECRET_KEY is required when authentication is enabled",
		},
		{
			name: "valid config with API key authentication disabled",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "mock"
				cfg.APIKeyRequired = false
				cfg.APIKey = ""
				return cfg
			},
			description: "API key authentication disabled should not require API key",
			wantErr:     false,
		},
		{
			name: "invalid config with API key authentication enabled but missing API key",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "mock"
				cfg.APIKeyRequired = true
				cfg.APIKey = ""
				return cfg
			},
			description: "API key authentication enabled should require API key",
			wantErr:     true,
			errMsg:      "API_KEY is required when API key authentication is enabled",
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

func TestLLMProcessorConfig_Validate_RAGFeatureFlags(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func() *LLMProcessorConfig
		description string
		wantErr     bool
		errMsg      string
	}{
		{
			name: "valid config with RAG enabled and URL provided",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "rag"
				cfg.RAGEnabled = true
				cfg.RAGAPIURL = "http://rag-api:5001"
				return cfg
			},
			description: "RAG enabled with valid URL should be valid",
			wantErr:     false,
		},
		{
			name: "valid config with RAG disabled",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "openai"
				cfg.LLMAPIKey = "sk-test-api-key"
				cfg.RAGEnabled = false
				cfg.RAGAPIURL = ""
				return cfg
			},
			description: "RAG disabled should not require RAG URL",
			wantErr:     false,
		},
		{
			name: "valid config with RAG disabled but URL provided",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "openai"
				cfg.LLMAPIKey = "sk-test-api-key"
				cfg.RAGEnabled = false
				cfg.RAGAPIURL = "http://rag-api:5001"
				return cfg
			},
			description: "RAG disabled with URL provided should still be valid",
			wantErr:     false,
		},
		{
			name: "valid config with streaming enabled and RAG enabled",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "rag"
				cfg.RAGEnabled = true
				cfg.RAGAPIURL = "http://rag-api:5001"
				cfg.StreamingEnabled = true
				cfg.MaxConcurrentStreams = 50
				return cfg
			},
			description: "RAG and streaming can be enabled together",
			wantErr:     false,
		},
		{
			name: "valid config with context builder enabled and RAG enabled",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "rag"
				cfg.RAGEnabled = true
				cfg.RAGAPIURL = "http://rag-api:5001"
				cfg.EnableContextBuilder = true
				cfg.MaxContextTokens = 4000
				return cfg
			},
			description: "RAG and context builder can be enabled together",
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

func TestLLMProcessorConfig_Validate_LogicalConstraints(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func() *LLMProcessorConfig
		description string
		wantErr     bool
		errMsg      string
	}{
		{
			name: "valid concurrent streams limit",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "mock"
				cfg.MaxConcurrentStreams = 500
				return cfg
			},
			description: "Reasonable concurrent streams limit should be valid",
			wantErr:     false,
		},
		{
			name: "invalid concurrent streams limit too high",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "mock"
				cfg.MaxConcurrentStreams = 1500
				return cfg
			},
			description: "Excessive concurrent streams limit should be invalid",
			wantErr:     true,
			errMsg:      "MAX_CONCURRENT_STREAMS should not exceed 1000 for performance reasons",
		},
		{
			name: "valid context tokens limit",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "mock"
				cfg.MaxContextTokens = 16000
				return cfg
			},
			description: "Reasonable context tokens limit should be valid",
			wantErr:     false,
		},
		{
			name: "invalid context tokens limit too high",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "mock"
				cfg.MaxContextTokens = 50000
				return cfg
			},
			description: "Excessive context tokens limit should be invalid",
			wantErr:     true,
			errMsg:      "MAX_CONTEXT_TOKENS should not exceed 32000 for most models",
		},
		{
			name: "valid circuit breaker threshold",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "mock"
				cfg.CircuitBreakerThreshold = 25
				return cfg
			},
			description: "Reasonable circuit breaker threshold should be valid",
			wantErr:     false,
		},
		{
			name: "invalid circuit breaker threshold too high",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "mock"
				cfg.CircuitBreakerThreshold = 75
				return cfg
			},
			description: "Excessive circuit breaker threshold should be invalid",
			wantErr:     true,
			errMsg:      "CIRCUIT_BREAKER_THRESHOLD should be reasonable (≤50)",
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

func TestLLMProcessorConfig_Validate_MultipleErrors(t *testing.T) {
	cfg := DefaultLLMProcessorConfig()
	cfg.LLMBackendType = "openai"
	cfg.LLMAPIKey = ""                           // Missing required API key
	cfg.AuthEnabled = true                       // Auth enabled
	cfg.JWTSecretKey = ""                        // But missing JWT secret
	cfg.APIKeyRequired = true                    // API key required
	cfg.APIKey = ""                              // But missing API key
	cfg.MaxConcurrentStreams = 1500              // Too high
	cfg.MaxContextTokens = 50000                 // Too high
	cfg.CircuitBreakerThreshold = 75             // Too high

	err := cfg.Validate()
	assert.Error(t, err)

	errMsg := err.Error()
	assert.Contains(t, errMsg, "OPENAI_API_KEY is required for non-mock/non-rag backends")
	assert.Contains(t, errMsg, "JWT_SECRET_KEY is required when authentication is enabled")
	assert.Contains(t, errMsg, "API_KEY is required when API key authentication is enabled")
	assert.Contains(t, errMsg, "MAX_CONCURRENT_STREAMS should not exceed 1000 for performance reasons")
	assert.Contains(t, errMsg, "MAX_CONTEXT_TOKENS should not exceed 32000 for most models")
	assert.Contains(t, errMsg, "CIRCUIT_BREAKER_THRESHOLD should be reasonable (≤50)")
}

func TestLoadLLMProcessorConfig_ValidConfiguration(t *testing.T) {
	// Clean environment
	cleanupLLMProcessorEnv(t)

	// Set minimal required environment variables for mock backend
	os.Setenv("LLM_BACKEND_TYPE", "mock")

	cfg, err := LoadLLMProcessorConfig()
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "mock", cfg.LLMBackendType)
}

func TestLoadLLMProcessorConfig_EnvironmentOverrides(t *testing.T) {
	// Clean environment
	cleanupLLMProcessorEnv(t)

	// Set test environment variables
	envVars := map[string]string{
		"PORT":                         "9080",
		"LOG_LEVEL":                    "debug",
		"SERVICE_VERSION":             "v3.0.0",
		"GRACEFUL_SHUTDOWN_TIMEOUT":   "45s",
		"LLM_BACKEND_TYPE":            "mock",
		"LLM_MODEL_NAME":              "gpt-3.5-turbo",
		"LLM_TIMEOUT":                 "120s",
		"LLM_MAX_TOKENS":              "4096",
		"RAG_API_URL":                 "http://custom-rag:6001",
		"RAG_TIMEOUT":                 "60s",
		"RAG_ENABLED":                 "false",
		"STREAMING_ENABLED":           "false",
		"MAX_CONCURRENT_STREAMS":      "200",
		"STREAM_TIMEOUT":              "10m",
		"ENABLE_CONTEXT_BUILDER":      "false",
		"MAX_CONTEXT_TOKENS":          "8000",
		"CONTEXT_TTL":                 "10m",
		"API_KEY_REQUIRED":            "true",
		"CORS_ENABLED":                "false",
		"ALLOWED_ORIGINS":             "http://localhost:3000",
		"REQUEST_TIMEOUT":             "60s",
		"MAX_REQUEST_SIZE":            "2097152",
		"CIRCUIT_BREAKER_ENABLED":     "false",
		"CIRCUIT_BREAKER_THRESHOLD":   "10",
		"CIRCUIT_BREAKER_TIMEOUT":     "120s",
		"RATE_LIMIT_ENABLED":          "false",
		"RATE_LIMIT_REQUESTS_PER_MINUTE": "120",
		"RATE_LIMIT_BURST":            "20",
		"MAX_RETRIES":                 "5",
		"RETRY_DELAY":                 "2s",
		"RETRY_BACKOFF":               "linear",
		"AUTH_ENABLED":                "false",
		"REQUIRE_AUTH":                "false",
		"ADMIN_USERS":                 "admin1,admin2",
		"OPERATOR_USERS":              "operator1,operator2",
		"USE_KUBERNETES_SECRETS":      "false",
		"SECRET_NAMESPACE":            "custom-namespace",
	}

	for key, value := range envVars {
		os.Setenv(key, value)
	}

	cfg, err := LoadLLMProcessorConfig()
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify all environment variables were loaded correctly
	assert.Equal(t, "9080", cfg.Port)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "v3.0.0", cfg.ServiceVersion)
	assert.Equal(t, 45*time.Second, cfg.GracefulShutdown)
	assert.Equal(t, "mock", cfg.LLMBackendType)
	assert.Equal(t, "gpt-3.5-turbo", cfg.LLMModelName)
	assert.Equal(t, 120*time.Second, cfg.LLMTimeout)
	assert.Equal(t, 4096, cfg.LLMMaxTokens)
	assert.Equal(t, "http://custom-rag:6001", cfg.RAGAPIURL)
	assert.Equal(t, 60*time.Second, cfg.RAGTimeout)
	assert.False(t, cfg.RAGEnabled)
	assert.False(t, cfg.StreamingEnabled)
	assert.Equal(t, 200, cfg.MaxConcurrentStreams)
	assert.Equal(t, 10*time.Minute, cfg.StreamTimeout)
	assert.False(t, cfg.EnableContextBuilder)
	assert.Equal(t, 8000, cfg.MaxContextTokens)
	assert.Equal(t, 10*time.Minute, cfg.ContextTTL)
	assert.True(t, cfg.APIKeyRequired)
	assert.False(t, cfg.CORSEnabled)
	assert.Equal(t, "http://localhost:3000", cfg.AllowedOrigins)
	assert.Equal(t, 60*time.Second, cfg.RequestTimeout)
	assert.Equal(t, int64(2097152), cfg.MaxRequestSize)
	assert.False(t, cfg.CircuitBreakerEnabled)
	assert.Equal(t, 10, cfg.CircuitBreakerThreshold)
	assert.Equal(t, 120*time.Second, cfg.CircuitBreakerTimeout)
	assert.False(t, cfg.RateLimitEnabled)
	assert.Equal(t, 120, cfg.RateLimitRequestsPerMin)
	assert.Equal(t, 20, cfg.RateLimitBurst)
	assert.Equal(t, 5, cfg.MaxRetries)
	assert.Equal(t, 2*time.Second, cfg.RetryDelay)
	assert.Equal(t, "linear", cfg.RetryBackoff)
	assert.False(t, cfg.AuthEnabled)
	assert.False(t, cfg.RequireAuth)
	assert.Equal(t, []string{"admin1", "admin2"}, cfg.AdminUsers)
	assert.Equal(t, []string{"operator1", "operator2"}, cfg.OperatorUsers)
	assert.False(t, cfg.UseKubernetesSecrets)
	assert.Equal(t, "custom-namespace", cfg.SecretNamespace)
}

func TestLoadLLMProcessorConfig_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
		errMsg  string
	}{
		{
			name: "invalid port",
			envVars: map[string]string{
				"PORT":             "99999",
				"LLM_BACKEND_TYPE": "mock",
			},
			wantErr: true,
			errMsg:  "PORT: invalid port number",
		},
		{
			name: "invalid log level",
			envVars: map[string]string{
				"LOG_LEVEL":        "invalid",
				"LLM_BACKEND_TYPE": "mock",
			},
			wantErr: true,
			errMsg:  "LOG_LEVEL: invalid log level",
		},
		{
			name: "invalid LLM backend type",
			envVars: map[string]string{
				"LLM_BACKEND_TYPE": "invalid",
			},
			wantErr: true,
			errMsg:  "LLM_BACKEND_TYPE: invalid LLM backend type",
		},
		{
			name: "invalid URL format",
			envVars: map[string]string{
				"LLM_BACKEND_TYPE": "mock",
				"RAG_API_URL":      "invalid-url",
			},
			wantErr: true,
			errMsg:  "RAG_API_URL: URL must start with http:// or https://",
		},
		{
			name: "invalid retry backoff",
			envVars: map[string]string{
				"LLM_BACKEND_TYPE": "mock",
				"RETRY_BACKOFF":    "invalid",
			},
			wantErr: true,
			errMsg:  "RETRY_BACKOFF: invalid retry backoff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean environment
			cleanupLLMProcessorEnv(t)

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			cfg, err := LoadLLMProcessorConfig()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, cfg)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
			}
		})
	}
}

func TestParseStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "single item",
			input:    "item1",
			expected: []string{"item1"},
		},
		{
			name:     "multiple items",
			input:    "item1,item2,item3",
			expected: []string{"item1", "item2", "item3"},
		},
		{
			name:     "items with spaces",
			input:    "item1, item2 , item3",
			expected: []string{"item1", "item2", "item3"},
		},
		{
			name:     "items with empty values",
			input:    "item1,,item3,",
			expected: []string{"item1", "item3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseStringSlice(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// cleanupLLMProcessorEnv cleans up LLM processor specific environment variables
func cleanupLLMProcessorEnv(t *testing.T) {
	envVars := []string{
		"PORT",
		"LOG_LEVEL",
		"SERVICE_VERSION",
		"GRACEFUL_SHUTDOWN_TIMEOUT",
		"LLM_BACKEND_TYPE",
		"LLM_API_KEY",
		"LLM_MODEL_NAME",
		"LLM_TIMEOUT",
		"LLM_MAX_TOKENS",
		"RAG_API_URL",
		"RAG_TIMEOUT",
		"RAG_ENABLED",
		"STREAMING_ENABLED",
		"MAX_CONCURRENT_STREAMS",
		"STREAM_TIMEOUT",
		"ENABLE_CONTEXT_BUILDER",
		"MAX_CONTEXT_TOKENS",
		"CONTEXT_TTL",
		"API_KEY_REQUIRED",
		"API_KEY",
		"CORS_ENABLED",
		"ALLOWED_ORIGINS",
		"REQUEST_TIMEOUT",
		"MAX_REQUEST_SIZE",
		"CIRCUIT_BREAKER_ENABLED",
		"CIRCUIT_BREAKER_THRESHOLD",
		"CIRCUIT_BREAKER_TIMEOUT",
		"RATE_LIMIT_ENABLED",
		"RATE_LIMIT_REQUESTS_PER_MINUTE",
		"RATE_LIMIT_BURST",
		"MAX_RETRIES",
		"RETRY_DELAY",
		"RETRY_BACKOFF",
		"AUTH_ENABLED",
		"AUTH_CONFIG_FILE",
		"JWT_SECRET_KEY",
		"REQUIRE_AUTH",
		"ADMIN_USERS",
		"OPERATOR_USERS",
		"USE_KUBERNETES_SECRETS",
		"SECRET_NAMESPACE",
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