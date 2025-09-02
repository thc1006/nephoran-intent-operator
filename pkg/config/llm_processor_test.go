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
	assert.Equal(t, []string{}, cfg.AllowedOrigins) // Default to empty for security

	// Performance Configuration
	assert.Equal(t, 30*time.Second, cfg.RequestTimeout)
	assert.Equal(t, int64(1048576), cfg.MaxRequestSize)

	// OAuth2 Authentication Configuration
	assert.False(t, cfg.AuthEnabled) // Default disabled for easier testing
	assert.False(t, cfg.RequireAuth) // Default disabled for easier testing
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
				// Disable auth and CORS for basic testing
				cfg.AuthEnabled = false
				cfg.CORSEnabled = false
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
				// Disable auth and CORS for basic testing
				cfg.AuthEnabled = false
				cfg.CORSEnabled = false
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
				// Disable auth and CORS for basic testing
				cfg.AuthEnabled = false
				cfg.CORSEnabled = false
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
				// Disable CORS for basic testing
				cfg.CORSEnabled = false
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
				// Disable auth and CORS for basic testing
				cfg.AuthEnabled = false
				cfg.CORSEnabled = false
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
			errMsg:      "CIRCUIT_BREAKER_THRESHOLD should be reasonable (??0)",
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
	cfg.LLMAPIKey = ""               // Missing required API key
	cfg.AuthEnabled = true           // Auth enabled
	cfg.JWTSecretKey = ""            // But missing JWT secret
	cfg.APIKeyRequired = true        // API key required
	cfg.APIKey = ""                  // But missing API key
	cfg.MaxConcurrentStreams = 1500  // Too high
	cfg.MaxContextTokens = 50000     // Too high
	cfg.CircuitBreakerThreshold = 75 // Too high

	err := cfg.Validate()
	assert.Error(t, err)

	errMsg := err.Error()
	assert.Contains(t, errMsg, "OPENAI_API_KEY is required for non-mock/non-rag backends")
	assert.Contains(t, errMsg, "JWT_SECRET_KEY is required when authentication is enabled")
	assert.Contains(t, errMsg, "API_KEY is required when API key authentication is enabled")
	assert.Contains(t, errMsg, "MAX_CONCURRENT_STREAMS should not exceed 1000 for performance reasons")
	assert.Contains(t, errMsg, "MAX_CONTEXT_TOKENS should not exceed 32000 for most models")
	assert.Contains(t, errMsg, "CIRCUIT_BREAKER_THRESHOLD should be reasonable (??0)")
}

func TestLLMProcessorConfig_Validate_CORS(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func() *LLMProcessorConfig
		setupEnv    func(t *testing.T)
		description string
		wantErr     bool
		errMsg      string
	}{
		{
			name: "CORS disabled - no origins required",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "mock"
				cfg.AuthEnabled = false
				cfg.CORSEnabled = false
				cfg.AllowedOrigins = []string{}
				return cfg
			},
			description: "CORS disabled should not require allowed origins",
			wantErr:     false,
		},
		{
			name: "CORS enabled with valid origins",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "mock"
				cfg.AuthEnabled = false
				cfg.CORSEnabled = true
				cfg.AllowedOrigins = []string{"https://example.com", "http://localhost:3000"}
				return cfg
			},
			description: "CORS enabled with valid origins should be valid",
			wantErr:     false,
		},
		{
			name: "CORS enabled but no origins configured",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "mock"
				cfg.AuthEnabled = false
				cfg.CORSEnabled = true
				cfg.AllowedOrigins = []string{}
				return cfg
			},
			description: "CORS enabled without origins should be invalid",
			wantErr:     true,
			errMsg:      "LLM_ALLOWED_ORIGINS must be configured when CORS is enabled",
		},
		{
			name: "CORS enabled with wildcard in non-production",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "mock"
				cfg.AuthEnabled = false
				cfg.CORSEnabled = true
				cfg.AllowedOrigins = []string{"*"}
				return cfg
			},
			setupEnv: func(t *testing.T) {
				t.Setenv("LLM_ENVIRONMENT", "development")
			},
			description: "CORS enabled with wildcard in development should be valid",
			wantErr:     false,
		},
		{
			name: "CORS enabled with wildcard in production",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "mock"
				cfg.AuthEnabled = false
				cfg.CORSEnabled = true
				cfg.AllowedOrigins = []string{"*"}
				return cfg
			},
			setupEnv: func(t *testing.T) {
				t.Setenv("LLM_ENVIRONMENT", "production")
			},
			description: "CORS enabled with wildcard in production should be invalid",
			wantErr:     true,
			errMsg:      "wildcard origin '*' is not allowed in production environments",
		},
		{
			name: "CORS enabled with invalid origin format",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "mock"
				cfg.AuthEnabled = false
				cfg.CORSEnabled = true
				cfg.AllowedOrigins = []string{"invalid-origin", "https://valid.com"}
				return cfg
			},
			description: "CORS enabled with invalid origin format should be invalid",
			wantErr:     true,
			errMsg:      "origin 'invalid-origin' must start with http:// or https://",
		},
		{
			name: "CORS enabled with partial wildcard",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "mock"
				cfg.AuthEnabled = false
				cfg.CORSEnabled = true
				cfg.AllowedOrigins = []string{"https://*.example.com"}
				return cfg
			},
			description: "CORS enabled with partial wildcard should be invalid",
			wantErr:     true,
			errMsg:      "origin 'https://*.example.com' contains wildcard but is not exact wildcard '*'",
		},
		{
			name: "CORS enabled with empty origin in list",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "mock"
				cfg.AuthEnabled = false
				cfg.CORSEnabled = true
				cfg.AllowedOrigins = []string{"https://valid.com", "", "http://localhost:3000"}
				return cfg
			},
			description: "CORS enabled with empty origin in list should be valid (empty entries ignored)",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupEnv != nil {
				tt.setupEnv(t)
			}

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

func TestParseAllowedOrigins(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		setupEnv    func(t *testing.T)
		expected    []string
		wantErr     bool
		errMsg      string
		description string
	}{
		{
			name:        "single valid origin",
			input:       "https://example.com",
			expected:    []string{"https://example.com"},
			wantErr:     false,
			description: "Single valid HTTPS origin should parse correctly",
		},
		{
			name:        "multiple valid origins",
			input:       "https://example.com,http://localhost:3000,https://app.company.com",
			expected:    []string{"https://example.com", "http://localhost:3000", "https://app.company.com"},
			wantErr:     false,
			description: "Multiple valid origins should parse correctly",
		},
		{
			name:        "origins with spaces",
			input:       "https://example.com, http://localhost:3000 , https://app.company.com",
			expected:    []string{"https://example.com", "http://localhost:3000", "https://app.company.com"},
			wantErr:     false,
			description: "Origins with spaces should be trimmed correctly",
		},
		{
			name:  "wildcard in development",
			input: "*",
			setupEnv: func(t *testing.T) {
				t.Setenv("LLM_ENVIRONMENT", "development")
			},
			expected:    []string{"*"},
			wantErr:     false,
			description: "Wildcard should be allowed in development environment",
		},
		{
			name:  "wildcard in production",
			input: "*",
			setupEnv: func(t *testing.T) {
				t.Setenv("LLM_ENVIRONMENT", "production")
			},
			wantErr:     true,
			errMsg:      "wildcard origin '*' is not allowed in production environments",
			description: "Wildcard should be rejected in production environment",
		},
		{
			name:        "empty string",
			input:       "",
			wantErr:     true,
			errMsg:      "origins string cannot be empty",
			description: "Empty string should be rejected",
		},
		{
			name:        "only whitespace",
			input:       "   ",
			wantErr:     true,
			errMsg:      "origins string cannot be empty",
			description: "Whitespace-only string should be rejected",
		},
		{
			name:        "invalid origin format",
			input:       "invalid-origin,https://valid.com",
			wantErr:     true,
			errMsg:      "invalid origin 'invalid-origin': origin must start with http:// or https://",
			description: "Invalid origin format should be rejected",
		},
		{
			name:        "origins with empty values",
			input:       "https://example.com,,http://localhost:3000,",
			expected:    []string{"https://example.com", "http://localhost:3000"},
			wantErr:     false,
			description: "Empty values in list should be ignored",
		},
		{
			name:        "partial wildcard",
			input:       "https://*.example.com",
			wantErr:     true,
			errMsg:      "origin contains wildcard but is not exact wildcard '*'",
			description: "Partial wildcards should be rejected",
		},
		{
			name:        "origin with whitespace characters",
			input:       "https://example .com",
			wantErr:     true,
			errMsg:      "origin contains invalid whitespace characters",
			description: "Origins with whitespace should be rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupEnv != nil {
				tt.setupEnv(t)
			}

			result, err := parseAllowedOrigins(tt.input)

			if tt.wantErr {
				assert.Error(t, err, tt.description)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg, tt.description)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err, tt.description)
				assert.Equal(t, tt.expected, result, tt.description)
			}
		})
	}
}

func TestLoadLLMProcessorConfig_CORSConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		description string
		wantErr     bool
		errMsg      string
		checkConfig func(t *testing.T, cfg *LLMProcessorConfig)
	}{
		{
			name: "CORS disabled via environment",
			envVars: map[string]string{
				"LLM_BACKEND_TYPE": "mock",
				"MAX_REQUEST_SIZE": "10485760",
				"AUTH_ENABLED":     "false",
				"CORS_ENABLED":     "false",
			},
			description: "CORS disabled should not require origins",
			wantErr:     false,
			checkConfig: func(t *testing.T, cfg *LLMProcessorConfig) {
				assert.False(t, cfg.CORSEnabled)
				assert.Equal(t, []string{}, cfg.AllowedOrigins)
			},
		},
		{
			name: "CORS enabled with valid origins from environment",
			envVars: map[string]string{
				"LLM_BACKEND_TYPE":    "mock",
				"AUTH_ENABLED":        "false",
				"CORS_ENABLED":        "true",
				"LLM_ALLOWED_ORIGINS": "https://example.com,http://localhost:3000",
				"MAX_REQUEST_SIZE":    "10485760", // 10MB
			},
			description: "CORS enabled with valid origins should load correctly",
			wantErr:     false,
			checkConfig: func(t *testing.T, cfg *LLMProcessorConfig) {
				assert.True(t, cfg.CORSEnabled)
				assert.Equal(t, []string{"https://example.com", "http://localhost:3000"}, cfg.AllowedOrigins)
			},
		},
		{
			name: "CORS enabled without origins in development",
			envVars: map[string]string{
				"LLM_BACKEND_TYPE": "mock",
				"MAX_REQUEST_SIZE": "10485760",
				"AUTH_ENABLED":     "false",
				"CORS_ENABLED":     "true",
				"GO_ENV":           "development",
			},
			description: "CORS enabled without origins in development should use default localhost origins",
			wantErr:     false,
			checkConfig: func(t *testing.T, cfg *LLMProcessorConfig) {
				assert.True(t, cfg.CORSEnabled)
				assert.Equal(t, []string{"http://localhost:3000", "http://localhost:8080"}, cfg.AllowedOrigins)
			},
		},
		{
			name: "CORS enabled without origins in production",
			envVars: map[string]string{
				"LLM_BACKEND_TYPE": "mock",
				"MAX_REQUEST_SIZE": "10485760",
				"AUTH_ENABLED":     "false",
				"CORS_ENABLED":     "true",
				"LLM_ENVIRONMENT":  "production",
			},
			description: "CORS enabled without origins in production should fail validation",
			wantErr:     true,
			errMsg:      "LLM_ALLOWED_ORIGINS must be configured when CORS is enabled",
		},
		{
			name: "CORS enabled with invalid origins from environment",
			envVars: map[string]string{
				"LLM_BACKEND_TYPE":    "mock",
				"AUTH_ENABLED":        "false",
				"CORS_ENABLED":        "true",
				"LLM_ALLOWED_ORIGINS": "invalid-origin,https://valid.com",
				"MAX_REQUEST_SIZE":    "10485760", // 10MB
			},
			description: "CORS enabled with invalid origins should fail validation",
			wantErr:     true,
			errMsg:      "LLM_ALLOWED_ORIGINS: invalid origin 'invalid-origin'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean environment
			cleanupLLMProcessorEnv(t)

			// Set test environment variables using t.Setenv for automatic cleanup
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			cfg, err := LoadLLMProcessorConfig()

			if tt.wantErr {
				assert.Error(t, err, tt.description)
				assert.Nil(t, cfg)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg, tt.description)
				}
			} else {
				assert.NoError(t, err, tt.description)
				assert.NotNil(t, cfg)
				if tt.checkConfig != nil {
					tt.checkConfig(t, cfg)
				}
			}
		})
	}
}

func TestLoadLLMProcessorConfig_ValidConfiguration(t *testing.T) {
	// Clean environment
	cleanupLLMProcessorEnv(t)

	// Set minimal required environment variables for mock backend
	t.Setenv("LLM_BACKEND_TYPE", "mock")
	t.Setenv("AUTH_ENABLED", "false")
	t.Setenv("CORS_ENABLED", "false")

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
		"PORT":                           "9080",
		"LOG_LEVEL":                      "debug",
		"SERVICE_VERSION":                "v3.0.0",
		"GRACEFUL_SHUTDOWN_TIMEOUT":      "45s",
		"LLM_BACKEND_TYPE":               "mock",
		"LLM_MODEL_NAME":                 "gpt-3.5-turbo",
		"LLM_TIMEOUT_SECS":               "120",
		"LLM_MAX_TOKENS":                 "4096",
		"RAG_API_URL":                    "http://custom-rag:6001",
		"RAG_TIMEOUT":                    "60s",
		"RAG_ENABLED":                    "false",
		"STREAMING_ENABLED":              "false",
		"MAX_CONCURRENT_STREAMS":         "200",
		"STREAM_TIMEOUT":                 "10m",
		"ENABLE_CONTEXT_BUILDER":         "false",
		"MAX_CONTEXT_TOKENS":             "8000",
		"CONTEXT_TTL":                    "10m",
		"API_KEY_REQUIRED":               "true",
		"API_KEY":                        "test-api-key-123",
		"CORS_ENABLED":                   "false",
		"LLM_ALLOWED_ORIGINS":            "http://localhost:3000",
		"REQUEST_TIMEOUT":                "60s",
		"MAX_REQUEST_SIZE":               "2097152",
		"CIRCUIT_BREAKER_ENABLED":        "false",
		"CIRCUIT_BREAKER_THRESHOLD":      "10",
		"CIRCUIT_BREAKER_TIMEOUT":        "120s",
		"RATE_LIMIT_ENABLED":             "false",
		"RATE_LIMIT_REQUESTS_PER_MINUTE": "120",
		"RATE_LIMIT_BURST":               "20",
		"MAX_RETRIES":                    "5",
		"RETRY_DELAY":                    "2s",
		"RETRY_BACKOFF":                  "linear",
		"AUTH_ENABLED":                   "false",
		"REQUIRE_AUTH":                   "false",
		"ADMIN_USERS":                    "admin1,admin2",
		"OPERATOR_USERS":                 "operator1,operator2",
		"USE_KUBERNETES_SECRETS":         "false",
		"SECRET_NAMESPACE":               "custom-namespace",
	}

	for key, value := range envVars {
		t.Setenv(key, value)
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
	assert.Equal(t, []string{}, cfg.AllowedOrigins) // CORS disabled, so empty
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
				"MAX_REQUEST_SIZE": "10485760",
			},
			wantErr: true,
			errMsg:  "PORT: invalid port number",
		},
		{
			name: "invalid log level",
			envVars: map[string]string{
				"LOG_LEVEL":        "invalid",
				"LLM_BACKEND_TYPE": "mock",
				"MAX_REQUEST_SIZE": "10485760",
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
				"MAX_REQUEST_SIZE": "10485760",
				"RAG_API_URL":      "invalid-url",
			},
			wantErr: true,
			errMsg:  "RAG_API_URL: URL must start with http:// or https://",
		},
		{
			name: "invalid retry backoff",
			envVars: map[string]string{
				"LLM_BACKEND_TYPE": "mock",
				"MAX_REQUEST_SIZE": "10485760",
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

			// Set test environment variables using t.Setenv for automatic cleanup
			for key, value := range tt.envVars {
				t.Setenv(key, value)
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

func TestLLMProcessorConfig_TLSConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func() *LLMProcessorConfig
		setupFiles  func(t *testing.T) (certPath, keyPath string, cleanup func())
		description string
		wantErr     bool
		errMsg      string
	}{
		{
			name: "TLS disabled - no certificates required",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "mock"
				cfg.TLSEnabled = false
				cfg.TLSCertPath = ""
				cfg.TLSKeyPath = ""
				// Disable auth and CORS to avoid validation errors
				cfg.AuthEnabled = false
				cfg.CORSEnabled = false
				return cfg
			},
			setupFiles:  nil,
			description: "TLS disabled should not require certificate files",
			wantErr:     false,
		},
		{
			name: "TLS enabled with valid certificates",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "mock"
				cfg.TLSEnabled = true
				// Disable auth and CORS to avoid validation errors
				cfg.AuthEnabled = false
				cfg.CORSEnabled = false
				return cfg
			},
			setupFiles: func(t *testing.T) (certPath, keyPath string, cleanup func()) {
				return createTestTLSFiles(t)
			},
			description: "TLS enabled with valid certificate files should be valid",
			wantErr:     false,
		},
		{
			name: "TLS enabled but missing certificate path",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "mock"
				cfg.TLSEnabled = true
				cfg.TLSCertPath = ""
				cfg.TLSKeyPath = "/path/to/key.pem"
				cfg.AuthEnabled = false
				cfg.CORSEnabled = false
				return cfg
			},
			setupFiles:  nil,
			description: "TLS enabled but missing certificate path should be invalid",
			wantErr:     true,
			errMsg:      "TLS_CERT_PATH is required when TLS is enabled",
		},
		{
			name: "TLS enabled but missing key path",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "mock"
				cfg.TLSEnabled = true
				cfg.TLSCertPath = "/path/to/cert.pem"
				cfg.TLSKeyPath = ""
				cfg.AuthEnabled = false
				cfg.CORSEnabled = false
				return cfg
			},
			setupFiles:  nil,
			description: "TLS enabled but missing key path should be invalid",
			wantErr:     true,
			errMsg:      "TLS_KEY_PATH is required when TLS is enabled",
		},
		{
			name: "TLS enabled with non-existent certificate file",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "mock"
				cfg.TLSEnabled = true
				cfg.TLSCertPath = "/non/existent/cert.pem"
				cfg.TLSKeyPath = "/non/existent/key.pem"
				cfg.AuthEnabled = false
				cfg.CORSEnabled = false
				return cfg
			},
			setupFiles:  nil,
			description: "TLS enabled with non-existent certificate file should be invalid",
			wantErr:     true,
			errMsg:      "certificate file does not exist",
		},
		{
			name: "TLS enabled with non-existent key file",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "mock"
				cfg.TLSEnabled = true
				cfg.AuthEnabled = false
				cfg.CORSEnabled = false
				return cfg
			},
			setupFiles: func(t *testing.T) (certPath, keyPath string, cleanup func()) {
				certPath, _, cleanup = createTestTLSFiles(t)
				keyPath = "/non/existent/key.pem"
				return
			},
			description: "TLS enabled with non-existent key file should be invalid",
			wantErr:     true,
			errMsg:      "private key file does not exist",
		},
		{
			name: "TLS disabled but certificate paths provided",
			setupConfig: func() *LLMProcessorConfig {
				cfg := DefaultLLMProcessorConfig()
				cfg.LLMBackendType = "mock"
				cfg.TLSEnabled = false
				cfg.TLSCertPath = "/path/to/cert.pem"
				cfg.TLSKeyPath = "/path/to/key.pem"
				cfg.AuthEnabled = false
				cfg.CORSEnabled = false
				return cfg
			},
			setupFiles:  nil,
			description: "TLS disabled but certificate paths provided should warn about misconfiguration",
			wantErr:     true,
			errMsg:      "TLS certificate/key paths provided but TLS_ENABLED=false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.setupConfig()

			// Setup test files if needed
			var cleanup func()
			if tt.setupFiles != nil {
				certPath, keyPath, cleanupFunc := tt.setupFiles(t)
				cfg.TLSCertPath = certPath
				cfg.TLSKeyPath = keyPath
				cleanup = cleanupFunc
			}

			// Ensure cleanup happens
			if cleanup != nil {
				defer cleanup()
			}

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

func TestLoadLLMProcessorConfig_TLSFromEnvironment(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		setupFiles  func(t *testing.T) (certPath, keyPath string, cleanup func())
		description string
		wantErr     bool
		errMsg      string
		checkConfig func(t *testing.T, cfg *LLMProcessorConfig)
	}{
		{
			name: "TLS configuration loaded from environment - disabled",
			envVars: map[string]string{
				"LLM_BACKEND_TYPE": "mock",
				"MAX_REQUEST_SIZE": "10485760",
				"TLS_ENABLED":      "false",
				"AUTH_ENABLED":     "false",
				"CORS_ENABLED":     "false",
			},
			setupFiles:  nil,
			description: "TLS disabled via environment should be loaded correctly",
			wantErr:     false,
			checkConfig: func(t *testing.T, cfg *LLMProcessorConfig) {
				require.NotNil(t, cfg, "config should not be nil")
				assert.False(t, cfg.TLSEnabled)
				assert.Empty(t, cfg.TLSCertPath)
				assert.Empty(t, cfg.TLSKeyPath)
			},
		},
		{
			name: "TLS configuration loaded from environment - enabled with valid paths",
			envVars: map[string]string{
				"LLM_BACKEND_TYPE": "mock",
				"MAX_REQUEST_SIZE": "10485760",
				"TLS_ENABLED":      "true",
				"AUTH_ENABLED":     "false",
				"CORS_ENABLED":     "false",
			},
			setupFiles: func(t *testing.T) (certPath, keyPath string, cleanup func()) {
				return createTestTLSFiles(t)
			},
			description: "TLS enabled via environment with valid files should be loaded correctly",
			wantErr:     false,
			checkConfig: func(t *testing.T, cfg *LLMProcessorConfig) {
				require.NotNil(t, cfg, "config should not be nil")
				assert.True(t, cfg.TLSEnabled)
				assert.NotEmpty(t, cfg.TLSCertPath)
				assert.NotEmpty(t, cfg.TLSKeyPath)
			},
		},
		{
			name: "TLS enabled via environment but missing certificate file",
			envVars: map[string]string{
				"LLM_BACKEND_TYPE": "mock",
				"MAX_REQUEST_SIZE": "10485760",
				"TLS_ENABLED":      "true",
				"TLS_CERT_PATH":    "/non/existent/cert.pem",
				"TLS_KEY_PATH":     "/non/existent/key.pem",
				"AUTH_ENABLED":     "false",
				"CORS_ENABLED":     "false",
			},
			setupFiles:  nil,
			description: "TLS enabled but missing certificate files should fail validation",
			wantErr:     true,
			errMsg:      "certificate file does not exist",
		},
		{
			name: "TLS configuration with empty paths when enabled",
			envVars: map[string]string{
				"LLM_BACKEND_TYPE": "mock",
				"MAX_REQUEST_SIZE": "10485760",
				"TLS_ENABLED":      "true",
				"TLS_CERT_PATH":    "",
				"TLS_KEY_PATH":     "",
				"AUTH_ENABLED":     "false",
				"CORS_ENABLED":     "false",
			},
			setupFiles:  nil,
			description: "TLS enabled but empty paths should fail validation",
			wantErr:     true,
			errMsg:      "TLS_CERT_PATH is required when TLS is enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean environment
			cleanupLLMProcessorEnv(t)

			// Set base environment variables with all required defaults
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			// Setup test files if needed
			var cleanup func()
			if tt.setupFiles != nil {
				certPath, keyPath, cleanupFunc := tt.setupFiles(t)
				t.Setenv("TLS_CERT_PATH", certPath)
				t.Setenv("TLS_KEY_PATH", keyPath)
				cleanup = cleanupFunc
			}

			// Ensure cleanup happens
			if cleanup != nil {
				defer cleanup()
			}

			cfg, err := LoadLLMProcessorConfig()

			if tt.wantErr {
				assert.Error(t, err, tt.description)
				assert.Nil(t, cfg)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg, tt.description)
				}
			} else {
				assert.NoError(t, err, tt.description)
				assert.NotNil(t, cfg)
				if tt.checkConfig != nil {
					tt.checkConfig(t, cfg)
				}
			}
		})
	}
}

func TestValidateTLSFiles(t *testing.T) {
	tests := []struct {
		name        string
		setupFiles  func(t *testing.T) (certPath, keyPath string, cleanup func())
		description string
		wantErr     bool
		errMsg      string
	}{
		{
			name: "valid certificate and key files",
			setupFiles: func(t *testing.T) (certPath, keyPath string, cleanup func()) {
				return createTestTLSFiles(t)
			},
			description: "Valid certificate and key files should pass validation",
			wantErr:     false,
		},
		{
			name: "non-existent certificate file",
			setupFiles: func(t *testing.T) (certPath, keyPath string, cleanup func()) {
				_, keyPath, cleanup = createTestTLSFiles(t)
				certPath = "/non/existent/cert.pem"
				return
			},
			description: "Non-existent certificate file should fail validation",
			wantErr:     true,
			errMsg:      "certificate file does not exist",
		},
		{
			name: "non-existent key file",
			setupFiles: func(t *testing.T) (certPath, keyPath string, cleanup func()) {
				certPath, _, cleanup = createTestTLSFiles(t)
				keyPath = "/non/existent/key.pem"
				return
			},
			description: "Non-existent key file should fail validation",
			wantErr:     true,
			errMsg:      "private key file does not exist",
		},
		{
			name: "empty certificate path",
			setupFiles: func(t *testing.T) (certPath, keyPath string, cleanup func()) {
				_, keyPath, cleanup = createTestTLSFiles(t)
				certPath = ""
				return
			},
			description: "Empty certificate path should fail validation",
			wantErr:     true,
			errMsg:      "certificate file does not exist",
		},
		{
			name: "empty key path",
			setupFiles: func(t *testing.T) (certPath, keyPath string, cleanup func()) {
				certPath, _, cleanup = createTestTLSFiles(t)
				keyPath = ""
				return
			},
			description: "Empty key path should fail validation",
			wantErr:     true,
			errMsg:      "private key file does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			certPath, keyPath, cleanup := tt.setupFiles(t)
			if cleanup != nil {
				defer cleanup()
			}

			err := validateTLSFiles(certPath, keyPath)

			if tt.wantErr {
				assert.Error(t, err, tt.description)
				assert.Contains(t, err.Error(), tt.errMsg, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

func TestDefaultLLMProcessorConfig_TLSDefaults(t *testing.T) {
	cfg := DefaultLLMProcessorConfig()

	// TLS Configuration defaults
	assert.False(t, cfg.TLSEnabled, "TLS should be disabled by default")
	assert.Empty(t, cfg.TLSCertPath, "TLS certificate path should be empty by default")
	assert.Empty(t, cfg.TLSKeyPath, "TLS key path should be empty by default")
}

// createTestTLSFiles creates temporary certificate and key files for testing
func createTestTLSFiles(t *testing.T) (certPath, keyPath string, cleanup func()) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create certificate file
	certPath = tmpDir + "/server.crt"
	certContent := `-----BEGIN CERTIFICATE-----
MIICljCCAX4CCQCKOtxqIPjM9TANBgkqhkiG9w0BAQsFADCBjTELMAkGA1UEBhMC
VVMxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoMGEludGVybmV0IFdpZGdp
dHMgUHR5IEx0ZDEQMA4GA1UEAwwHdGVzdC1jYTEqMCgGCSqGSIb3DQEJARYbdGVz
dEBleGFtcGxlLmNvbTEEDAJVUzELMAkGA1UEBhMCVVMwHhcNMjQwMTAxMDAwMDAw
WhcNMjUwMTAxMDAwMDAwWjCBjTELMAkGA1UEBhMCVVMxEzARBgNVBAgMClNvbWUt
U3RhdGUxITAfBgNVBAoMGEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDEQMA4GA1UE
AwwHdGVzdC1jYTEqMCgGCSqGSIb3DQEJARYbdGVzdEBleGFtcGxlLmNvbTEEDAJV
UzELMAkGA1UEBhMCVVMwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC7
-----END CERTIFICATE-----`

	keyPath = tmpDir + "/server.key"
	keyContent := `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC7VJTUt9Us8cKB
wkG6oQQVduafWaNAe506lunHHH2xLDqxB3ISsOgTFVjrdwqz4F9S4kKO2KYzOGZh
oQQiCjRGD5mPjDfSKt9vq6ZV2V3xm1qhU8lXJ8kYgZLB0+9q1m9tQ9qxBgkqhkiG
9w0BAQEFAASCBKcwggSjAgEAAoIBAQC7VJTUt9Us8cKBwkG6oQQVduafWaNAe506
lunHHH2xLDqxB3ISsOgTFVjrdwqz4F9S4kKO2KYzOGZhoQQiCjRGD5mPjDfSKt9v
q6ZV2V3xm1qhU8lXJ8kYgZLB0+9q1m9tQ9qxBgkqhkiG9w0BAQEFAASCBKcwggSj
AgEAAoIBAQC7VJTUt9Us8cKBwkG6oQQVduafWaNAe506lunHHH2xLDqxB3ISsOgT
FVjrdwqz4F9S4kKO2KYzOGZhoQQiCjRGD5mPjDfSKt9vq6ZV2V3xm1qhU8lXJ8kY
gZLB0+9q1m9tQ9qxBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC7VJTUt9Us
8cKBwkG6oQQVduafWaNAe506lunHHH2xLDqxB3ISsOgTFVjrdwqz4F9S4kKO2KYz
OGZhoQQiCjRGD5mPjDfSKt9vq6ZV2V3xm1qhU8lXJ8kYgZLB0+9q1m9tQ9qx
-----END PRIVATE KEY-----`

	// Write certificate file
	err := os.WriteFile(certPath, []byte(certContent), 0o644)
	require.NoError(t, err)

	// Write key file
	err = os.WriteFile(keyPath, []byte(keyContent), 0o600)
	require.NoError(t, err)

	cleanup = func() {
		os.RemoveAll(tmpDir)
	}

	return certPath, keyPath, cleanup
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
		"LLM_ALLOWED_ORIGINS",
		"REQUEST_TIMEOUT",
		"MAX_REQUEST_SIZE",
		"HTTP_MAX_BODY",
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
		"TLS_ENABLED",
		"TLS_CERT_PATH",
		"TLS_KEY_PATH",
		"USE_KUBERNETES_SECRETS",
		"SECRET_NAMESPACE",
		// Environment detection variables (used by isDevelopmentEnvironment())
		"GO_ENV",
		"NODE_ENV",
		"ENVIRONMENT",
		"ENV",
		"APP_ENV",
		"LLM_ENVIRONMENT",
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
