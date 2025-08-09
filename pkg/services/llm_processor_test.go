package services

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/handlers"
	"github.com/thc1006/nephoran-intent-operator/pkg/health"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
)

// LLMProcessorServiceTestSuite provides comprehensive test coverage for LLMProcessorService
type LLMProcessorServiceTestSuite struct {
	suite.Suite
	ctx      context.Context
	cancel   context.CancelFunc
	mockCtrl *gomock.Controller
	logger   *slog.Logger
	service  *LLMProcessorService
	config   *config.LLMProcessorConfig
}

func TestLLMProcessorServiceSuite(t *testing.T) {
	suite.Run(t, new(LLMProcessorServiceTestSuite))
}

func (suite *LLMProcessorServiceTestSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithTimeout(context.Background(), 30*time.Second)
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func (suite *LLMProcessorServiceTestSuite) TearDownSuite() {
	suite.cancel()
}

func (suite *LLMProcessorServiceTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.config = &config.LLMProcessorConfig{
		Enabled:              true,
		ServiceVersion:       "1.0.0",
		LLMAPIKey:            "test-api-key",
		LLMModelName:         "gpt-4o-mini",
		LLMMaxTokens:         4000,
		LLMBackendType:       "openai",
		LLMTimeout:           30 * time.Second,
		RAGAPIURL:            "http://localhost:8081",
		RAGEnabled:           true,
		StreamingEnabled:     true,
		EnableContextBuilder: true,
		UseKubernetesSecrets: false,
		SecretNamespace:      "default",
	}

	suite.service = NewLLMProcessorService(suite.config, suite.logger)
}

func (suite *LLMProcessorServiceTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *LLMProcessorServiceTestSuite) TestNewLLMProcessorService() {
	service := NewLLMProcessorService(suite.config, suite.logger)

	assert.NotNil(suite.T(), service)
	assert.Equal(suite.T(), suite.config, service.config)
	assert.Equal(suite.T(), suite.logger, service.logger)
	assert.Nil(suite.T(), service.processor)     // Should be nil before initialization
	assert.Nil(suite.T(), service.healthChecker) // Should be nil before initialization
}

func (suite *LLMProcessorServiceTestSuite) TestInitialize_Success() {
	// Set environment variables for API keys
	os.Setenv("OPENAI_API_KEY", "test-openai-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	err := suite.service.Initialize(suite.ctx)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), suite.service.processor)
	assert.NotNil(suite.T(), suite.service.healthChecker)
	assert.NotNil(suite.T(), suite.service.tokenManager)
	assert.NotNil(suite.T(), suite.service.circuitBreakerMgr)
	assert.NotNil(suite.T(), suite.service.contextBuilder)
	assert.NotNil(suite.T(), suite.service.relevanceScorer)
	assert.NotNil(suite.T(), suite.service.promptBuilder)
	assert.NotNil(suite.T(), suite.service.streamingProcessor)
}

func (suite *LLMProcessorServiceTestSuite) TestInitialize_WithKubernetesSecrets() {
	suite.config.UseKubernetesSecrets = true
	suite.config.SecretNamespace = "test-namespace"

	// This will fail in unit test environment without K8s, but we can test the flow
	err := suite.service.Initialize(suite.ctx)

	// Should still succeed by falling back to environment variables
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), suite.service.healthChecker)
}

func (suite *LLMProcessorServiceTestSuite) TestInitialize_DisabledFeatures() {
	suite.config.RAGEnabled = false
	suite.config.StreamingEnabled = false
	suite.config.EnableContextBuilder = false

	os.Setenv("OPENAI_API_KEY", "test-openai-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	err := suite.service.Initialize(suite.ctx)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), suite.service.processor)
	assert.Nil(suite.T(), suite.service.streamingProcessor)
	assert.Nil(suite.T(), suite.service.contextBuilder)
}

func (suite *LLMProcessorServiceTestSuite) TestValidateLLMConfig_Success() {
	testCases := []struct {
		name      string
		apiKey    string
		configMod func(*config.LLMProcessorConfig)
	}{
		{
			name:   "Valid OpenAI config",
			apiKey: "sk-test123",
			configMod: func(c *config.LLMProcessorConfig) {
				c.LLMBackendType = "openai"
				c.LLMModelName = "gpt-4o-mini"
				c.LLMMaxTokens = 4000
				c.LLMTimeout = 30 * time.Second
				c.RAGAPIURL = "http://localhost:8081"
			},
		},
		{
			name:   "Valid mock backend",
			apiKey: "",
			configMod: func(c *config.LLMProcessorConfig) {
				c.LLMBackendType = "mock"
				c.LLMModelName = "mock-model"
				c.LLMMaxTokens = 1000
				c.LLMTimeout = 10 * time.Second
				c.RAGAPIURL = "http://localhost:8081"
			},
		},
		{
			name:   "Valid RAG backend",
			apiKey: "",
			configMod: func(c *config.LLMProcessorConfig) {
				c.LLMBackendType = "rag"
				c.LLMModelName = "rag-model"
				c.LLMMaxTokens = 2000
				c.LLMTimeout = 15 * time.Second
				c.RAGAPIURL = "http://localhost:8081"
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			config := *suite.config // Copy config
			tc.configMod(&config)

			service := NewLLMProcessorService(&config, suite.logger)
			err := service.validateLLMConfig(tc.apiKey)

			assert.NoError(suite.T(), err)
		})
	}
}

func (suite *LLMProcessorServiceTestSuite) TestValidateLLMConfig_Errors() {
	testCases := []struct {
		name      string
		apiKey    string
		configMod func(*config.LLMProcessorConfig)
		errorMsg  string
	}{
		{
			name:   "Missing API key for OpenAI",
			apiKey: "",
			configMod: func(c *config.LLMProcessorConfig) {
				c.LLMBackendType = "openai"
			},
			errorMsg: "API Key is required",
		},
		{
			name:   "Empty model name",
			apiKey: "test-key",
			configMod: func(c *config.LLMProcessorConfig) {
				c.LLMModelName = ""
			},
			errorMsg: "model name is required",
		},
		{
			name:   "Invalid max tokens",
			apiKey: "test-key",
			configMod: func(c *config.LLMProcessorConfig) {
				c.LLMMaxTokens = 0
			},
			errorMsg: "max tokens must be greater than 0",
		},
		{
			name:   "Invalid timeout",
			apiKey: "test-key",
			configMod: func(c *config.LLMProcessorConfig) {
				c.LLMTimeout = 0
			},
			errorMsg: "timeout must be greater than 0",
		},
		{
			name:   "Missing RAG API URL",
			apiKey: "test-key",
			configMod: func(c *config.LLMProcessorConfig) {
				c.RAGAPIURL = ""
			},
			errorMsg: "RAG API URL is required",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			config := *suite.config // Copy config
			tc.configMod(&config)

			service := NewLLMProcessorService(&config, suite.logger)
			err := service.validateLLMConfig(tc.apiKey)

			assert.Error(suite.T(), err)
			assert.Contains(suite.T(), err.Error(), tc.errorMsg)
		})
	}
}

func (suite *LLMProcessorServiceTestSuite) TestLoadSecureAPIKeys_EnvironmentVariables() {
	// Set environment variables
	os.Setenv("OPENAI_API_KEY", "env-openai-key")
	os.Setenv("WEAVIATE_API_KEY", "env-weaviate-key")
	os.Setenv("JWT_SECRET_KEY", "env-jwt-secret")
	defer func() {
		os.Unsetenv("OPENAI_API_KEY")
		os.Unsetenv("WEAVIATE_API_KEY")
		os.Unsetenv("JWT_SECRET_KEY")
	}()

	apiKeys, err := suite.service.loadSecureAPIKeys(suite.ctx)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), apiKeys)
	assert.Equal(suite.T(), "env-openai-key", apiKeys.OpenAI)
	assert.Equal(suite.T(), "env-weaviate-key", apiKeys.Weaviate)
	assert.Equal(suite.T(), "env-jwt-secret", apiKeys.JWTSecret)
}

func (suite *LLMProcessorServiceTestSuite) TestLoadSecureAPIKeys_DefaultValues() {
	// Clear environment variables
	os.Unsetenv("OPENAI_API_KEY")
	os.Unsetenv("WEAVIATE_API_KEY")
	os.Unsetenv("API_KEY")
	os.Unsetenv("JWT_SECRET_KEY")

	apiKeys, err := suite.service.loadSecureAPIKeys(suite.ctx)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), apiKeys)
	assert.Equal(suite.T(), "", apiKeys.OpenAI)
	assert.Equal(suite.T(), "", apiKeys.Weaviate)
	assert.Equal(suite.T(), "", apiKeys.Generic)
	assert.Equal(suite.T(), "", apiKeys.JWTSecret)
}

func (suite *LLMProcessorServiceTestSuite) TestGetComponents() {
	// Initialize the service first
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	err := suite.service.Initialize(suite.ctx)
	require.NoError(suite.T(), err)

	processor, streamingProcessor, circuitBreakerMgr, tokenManager,
		contextBuilder, relevanceScorer, promptBuilder, healthChecker := suite.service.GetComponents()

	assert.NotNil(suite.T(), processor)
	assert.NotNil(suite.T(), streamingProcessor)
	assert.NotNil(suite.T(), circuitBreakerMgr)
	assert.NotNil(suite.T(), tokenManager)
	assert.NotNil(suite.T(), contextBuilder)
	assert.NotNil(suite.T(), relevanceScorer)
	assert.NotNil(suite.T(), promptBuilder)
	assert.NotNil(suite.T(), healthChecker)
}

func (suite *LLMProcessorServiceTestSuite) TestShutdown() {
	// Initialize the service first
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	err := suite.service.Initialize(suite.ctx)
	require.NoError(suite.T(), err)

	// Shutdown should complete without error
	err = suite.service.Shutdown(suite.ctx)
	assert.NoError(suite.T(), err)

	// Health checker should be marked as not ready
	assert.False(suite.T(), suite.service.healthChecker.IsReady())
}

func (suite *LLMProcessorServiceTestSuite) TestShutdown_WithoutInitialization() {
	// Should handle shutdown gracefully even if not initialized
	err := suite.service.Shutdown(suite.ctx)
	assert.NoError(suite.T(), err)
}

// Table-driven tests for different configuration scenarios
func (suite *LLMProcessorServiceTestSuite) TestInitialize_ConfigurationScenarios() {
	testCases := []struct {
		name        string
		configMod   func(*config.LLMProcessorConfig)
		envVars     map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name: "Full featured configuration",
			configMod: func(c *config.LLMProcessorConfig) {
				c.RAGEnabled = true
				c.StreamingEnabled = true
				c.EnableContextBuilder = true
			},
			envVars: map[string]string{
				"OPENAI_API_KEY": "test-openai-key",
			},
			expectError: false,
		},
		{
			name: "Minimal configuration",
			configMod: func(c *config.LLMProcessorConfig) {
				c.RAGEnabled = false
				c.StreamingEnabled = false
				c.EnableContextBuilder = false
			},
			envVars: map[string]string{
				"OPENAI_API_KEY": "test-openai-key",
			},
			expectError: false,
		},
		{
			name: "Mock backend configuration",
			configMod: func(c *config.LLMProcessorConfig) {
				c.LLMBackendType = "mock"
			},
			envVars:     map[string]string{},
			expectError: false,
		},
		{
			name: "Invalid model name",
			configMod: func(c *config.LLMProcessorConfig) {
				c.LLMModelName = ""
			},
			envVars: map[string]string{
				"OPENAI_API_KEY": "test-openai-key",
			},
			expectError: true,
			errorMsg:    "model name is required",
		},
		{
			name: "Invalid max tokens",
			configMod: func(c *config.LLMProcessorConfig) {
				c.LLMMaxTokens = -1
			},
			envVars: map[string]string{
				"OPENAI_API_KEY": "test-openai-key",
			},
			expectError: true,
			errorMsg:    "max tokens must be greater than 0",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Set up environment variables
			for key, value := range tc.envVars {
				os.Setenv(key, value)
			}
			defer func() {
				for key := range tc.envVars {
					os.Unsetenv(key)
				}
			}()

			// Modify config
			config := *suite.config // Copy config
			tc.configMod(&config)

			service := NewLLMProcessorService(&config, suite.logger)
			err := service.Initialize(suite.ctx)

			if tc.expectError {
				assert.Error(suite.T(), err)
				if tc.errorMsg != "" {
					assert.Contains(suite.T(), err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(suite.T(), err)
				assert.NotNil(suite.T(), service.processor)
				assert.NotNil(suite.T(), service.healthChecker)
			}
		})
	}
}

func (suite *LLMProcessorServiceTestSuite) TestHealthChecksRegistration() {
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	err := suite.service.Initialize(suite.ctx)
	require.NoError(suite.T(), err)

	// Verify health checks are registered by checking health
	response := suite.service.healthChecker.Check(suite.ctx)

	assert.NotNil(suite.T(), response)
	assert.NotEmpty(suite.T(), response.Checks)

	// Look for expected health checks
	checkNames := make([]string, len(response.Checks))
	for i, check := range response.Checks {
		checkNames[i] = check.Name
	}

	assert.Contains(suite.T(), checkNames, "service_status")
	assert.Contains(suite.T(), checkNames, "circuit_breaker")
	assert.Contains(suite.T(), checkNames, "token_manager")

	// If streaming is enabled, check for streaming processor health check
	if suite.config.StreamingEnabled {
		assert.Contains(suite.T(), checkNames, "streaming_processor")
	}
}

func (suite *LLMProcessorServiceTestSuite) TestHealthChecksRegistration_WithRAGDependency() {
	suite.config.RAGEnabled = true
	suite.config.RAGAPIURL = "http://localhost:8081"

	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	err := suite.service.Initialize(suite.ctx)
	require.NoError(suite.T(), err)

	// Check that RAG API dependency is registered
	response := suite.service.healthChecker.Check(suite.ctx)

	assert.NotNil(suite.T(), response)
	// Dependencies should be registered
	depNames := make([]string, len(response.Dependencies))
	for i, dep := range response.Dependencies {
		depNames[i] = dep.Name
	}

	assert.Contains(suite.T(), depNames, "rag_api")
}

// Edge cases and error handling tests
func (suite *LLMProcessorServiceTestSuite) TestEdgeCases() {
	testCases := []struct {
		name        string
		setupFunc   func(*LLMProcessorService)
		testFunc    func(*LLMProcessorService) error
		expectError bool
		errorMsg    string
	}{
		{
			name: "Initialize with nil config",
			setupFunc: func(s *LLMProcessorService) {
				s.config = nil
			},
			testFunc: func(s *LLMProcessorService) error {
				return s.Initialize(suite.ctx)
			},
			expectError: true,
		},
		{
			name: "Shutdown before initialize",
			setupFunc: func(s *LLMProcessorService) {
				// No setup needed
			},
			testFunc: func(s *LLMProcessorService) error {
				return s.Shutdown(suite.ctx)
			},
			expectError: false, // Should handle gracefully
		},
		{
			name: "Initialize with context cancellation",
			setupFunc: func(s *LLMProcessorService) {
				// Setup normal config
			},
			testFunc: func(s *LLMProcessorService) error {
				cancelCtx, cancel := context.WithCancel(suite.ctx)
				cancel() // Cancel immediately
				return s.Initialize(cancelCtx)
			},
			expectError: false, // Context cancellation handled in loadSecureAPIKeys
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			service := NewLLMProcessorService(suite.config, suite.logger)
			tc.setupFunc(service)

			err := tc.testFunc(service)

			if tc.expectError {
				assert.Error(suite.T(), err)
				if tc.errorMsg != "" {
					assert.Contains(suite.T(), err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(suite.T(), err)
			}
		})
	}
}

// Concurrent operations testing
func (suite *LLMProcessorServiceTestSuite) TestConcurrentOperations() {
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	err := suite.service.Initialize(suite.ctx)
	require.NoError(suite.T(), err)

	// Test concurrent health checks
	numGoroutines := 10
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			response := suite.service.healthChecker.Check(suite.ctx)
			if response == nil {
				errors <- assert.AnError
				return
			}
			errors <- nil
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		err := <-errors
		assert.NoError(suite.T(), err)
	}
}

func (suite *LLMProcessorServiceTestSuite) TestIntegrationFlow() {
	// Test complete service lifecycle
	os.Setenv("OPENAI_API_KEY", "test-integration-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	// Step 1: Initialize
	err := suite.service.Initialize(suite.ctx)
	require.NoError(suite.T(), err)

	// Step 2: Verify components are available
	processor, streamingProcessor, circuitBreakerMgr, tokenManager,
		contextBuilder, relevanceScorer, promptBuilder, healthChecker := suite.service.GetComponents()

	assert.NotNil(suite.T(), processor)
	assert.NotNil(suite.T(), streamingProcessor)
	assert.NotNil(suite.T(), circuitBreakerMgr)
	assert.NotNil(suite.T(), tokenManager)
	assert.NotNil(suite.T(), contextBuilder)
	assert.NotNil(suite.T(), relevanceScorer)
	assert.NotNil(suite.T(), promptBuilder)
	assert.NotNil(suite.T(), healthChecker)

	// Step 3: Verify health checks work
	response := healthChecker.Check(suite.ctx)
	assert.NotNil(suite.T(), response)
	assert.NotEmpty(suite.T(), response.Checks)

	// Step 4: Shutdown gracefully
	err = suite.service.Shutdown(suite.ctx)
	assert.NoError(suite.T(), err)

	// Step 5: Verify service is marked as not ready
	assert.False(suite.T(), healthChecker.IsReady())
}

// Helper functions for testing
func (suite *LLMProcessorServiceTestSuite) setupMinimalConfig() *config.LLMProcessorConfig {
	return &config.LLMProcessorConfig{
		Enabled:              true,
		ServiceVersion:       "1.0.0",
		LLMAPIKey:            "test-api-key",
		LLMModelName:         "gpt-4o-mini",
		LLMMaxTokens:         1000,
		LLMBackendType:       "mock",
		LLMTimeout:           10 * time.Second,
		RAGAPIURL:            "http://localhost:8081",
		RAGEnabled:           false,
		StreamingEnabled:     false,
		EnableContextBuilder: false,
		UseKubernetesSecrets: false,
	}
}

// Mock implementations and helper types

// IntentProcessor interface for testing
type MockIntentProcessor struct {
	LLMClient         interface{}
	RAGEnhancedClient interface{}
	CircuitBreaker    interface{}
	Logger            *slog.Logger
}

// Mock implementations for missing types
func (suite *LLMProcessorServiceTestSuite) createMockComponents() {
	// These would be actual mock implementations in a real test suite
	// For now, we'll use placeholder implementations
}

// Test utilities
func (suite *LLMProcessorServiceTestSuite) assertComponentsInitialized(service *LLMProcessorService) {
	assert.NotNil(suite.T(), service.processor, "processor should be initialized")
	assert.NotNil(suite.T(), service.healthChecker, "health checker should be initialized")
	assert.NotNil(suite.T(), service.tokenManager, "token manager should be initialized")
	assert.NotNil(suite.T(), service.circuitBreakerMgr, "circuit breaker manager should be initialized")
	assert.NotNil(suite.T(), service.relevanceScorer, "relevance scorer should be initialized")
	assert.NotNil(suite.T(), service.promptBuilder, "prompt builder should be initialized")
}

func (suite *LLMProcessorServiceTestSuite) assertOptionalComponentsInitialized(service *LLMProcessorService, config *config.LLMProcessorConfig) {
	if config.StreamingEnabled {
		assert.NotNil(suite.T(), service.streamingProcessor, "streaming processor should be initialized when enabled")
	} else {
		assert.Nil(suite.T(), service.streamingProcessor, "streaming processor should not be initialized when disabled")
	}

	if config.EnableContextBuilder {
		assert.NotNil(suite.T(), service.contextBuilder, "context builder should be initialized when enabled")
	} else {
		assert.Nil(suite.T(), service.contextBuilder, "context builder should not be initialized when disabled")
	}
}

// TestCircuitBreakerHealthCheck tests the circuit breaker health check functionality
func (suite *LLMProcessorServiceTestSuite) TestCircuitBreakerHealthCheck() {
	// Initialize service for testing
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	err := suite.service.Initialize(suite.ctx)
	require.NoError(suite.T(), err)

	testCases := []struct {
		name            string
		stats           map[string]interface{}
		expectedStatus  health.Status
		expectedMessage string
		checkContains   []string
	}{
		{
			name:            "No circuit breakers registered",
			stats:           map[string]interface{}{},
			expectedStatus:  health.StatusHealthy,
			expectedMessage: "No circuit breakers registered",
		},
		{
			name: "All circuit breakers closed",
			stats: map[string]interface{}{
				"llm-service": map[string]interface{}{
					"state":    "closed",
					"failures": 0,
				},
				"rag-service": map[string]interface{}{
					"state":    "closed",
					"failures": 1,
				},
			},
			expectedStatus:  health.StatusHealthy,
			expectedMessage: "All 2 circuit breakers operational",
		},
		{
			name: "Circuit breakers in half-open state",
			stats: map[string]interface{}{
				"llm-service": map[string]interface{}{
					"state":    "half-open",
					"failures": 2,
				},
			},
			expectedStatus:  health.StatusHealthy,
			expectedMessage: "All 1 circuit breakers operational",
		},
		{
			name: "Single circuit breaker open",
			stats: map[string]interface{}{
				"llm-service": map[string]interface{}{
					"state":    "open",
					"failures": 5,
				},
			},
			expectedStatus: health.StatusUnhealthy,
			checkContains:  []string{"Circuit breakers in open state:", "llm-service"},
		},
		{
			name: "Multiple circuit breakers with one open",
			stats: map[string]interface{}{
				"llm-service": map[string]interface{}{
					"state":    "closed",
					"failures": 0,
				},
				"rag-service": map[string]interface{}{
					"state":    "open",
					"failures": 5,
				},
				"token-service": map[string]interface{}{
					"state":    "half-open",
					"failures": 2,
				},
			},
			expectedStatus: health.StatusUnhealthy,
			checkContains:  []string{"Circuit breakers in open state:", "rag-service"},
		},
		{
			name: "Multiple open circuit breakers",
			stats: map[string]interface{}{
				"llm-service": map[string]interface{}{
					"state":    "open",
					"failures": 3,
				},
				"rag-service": map[string]interface{}{
					"state":    "open",
					"failures": 7,
				},
				"token-service": map[string]interface{}{
					"state":    "closed",
					"failures": 0,
				},
			},
			expectedStatus: health.StatusUnhealthy,
			checkContains:  []string{"Circuit breakers in open state:", "llm-service", "rag-service"},
		},
		{
			name: "Circuit breaker with malformed stats (missing state)",
			stats: map[string]interface{}{
				"llm-service": map[string]interface{}{
					"failures": 0,
					// Missing "state" field
				},
				"rag-service": map[string]interface{}{
					"state":    "closed",
					"failures": 1,
				},
			},
			expectedStatus:  health.StatusHealthy,
			expectedMessage: "All 2 circuit breakers operational",
		},
		{
			name: "Circuit breaker with non-string state",
			stats: map[string]interface{}{
				"llm-service": map[string]interface{}{
					"state":    123, // Invalid type
					"failures": 2,
				},
			},
			expectedStatus:  health.StatusHealthy,
			expectedMessage: "All 1 circuit breakers operational",
		},
		{
			name: "Circuit breaker with non-map stats",
			stats: map[string]interface{}{
				"llm-service": "invalid-stats", // Should be map
				"rag-service": map[string]interface{}{
					"state":    "closed",
					"failures": 0,
				},
			},
			expectedStatus:  health.StatusHealthy,
			expectedMessage: "All 2 circuit breakers operational",
		},
		{
			name: "Mixed valid and invalid circuit breakers with open state",
			stats: map[string]interface{}{
				"llm-service": map[string]interface{}{
					"state":    "open",
					"failures": 5,
				},
				"invalid-service": "invalid-stats",
				"rag-service": map[string]interface{}{
					"state":    "closed",
					"failures": 0,
				},
			},
			expectedStatus: health.StatusUnhealthy,
			checkContains:  []string{"Circuit breakers in open state:", "llm-service"},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Mock circuit breaker manager
			mockCBMgr := &MockCircuitBreakerManagerForLLM{
				stats: tc.stats,
			}

			// Replace the circuit breaker manager temporarily
			originalCBMgr := suite.service.circuitBreakerMgr
			suite.service.circuitBreakerMgr = mockCBMgr
			defer func() {
				suite.service.circuitBreakerMgr = originalCBMgr
			}()

			// Re-register health checks with the mock
			suite.service.registerHealthChecks()

			// Execute the circuit breaker health check
			result := suite.service.healthChecker.RunCheck(suite.ctx, "circuit_breaker")

			// Verify the result
			require.NotNil(suite.T(), result)
			assert.Equal(suite.T(), tc.expectedStatus, result.Status)

			if tc.expectedMessage != "" {
				assert.Equal(suite.T(), tc.expectedMessage, result.Message)
			}

			if len(tc.checkContains) > 0 {
				for _, expectedStr := range tc.checkContains {
					assert.Contains(suite.T(), result.Message, expectedStr)
				}
			}
		})
	}
}

// TestCircuitBreakerHealthCheckEdgeCases tests edge cases for circuit breaker health checks
func (suite *LLMProcessorServiceTestSuite) TestCircuitBreakerHealthCheckEdgeCases() {
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	err := suite.service.Initialize(suite.ctx)
	require.NoError(suite.T(), err)

	testCases := []struct {
		name           string
		cbManager      interface{} // Can be nil or mock
		expectedStatus health.Status
		checkContains  string
	}{
		{
			name:           "No circuit breaker manager",
			cbManager:      nil,
			expectedStatus: health.StatusHealthy,
			checkContains:  "No circuit breakers registered",
		},
		{
			name: "Circuit breaker manager returns error",
			cbManager: &MockCircuitBreakerManagerErrorForLLM{
				shouldError: true,
			},
			expectedStatus: health.StatusHealthy,
			checkContains:  "No circuit breakers registered",
		},
		{
			name: "Empty stats from circuit breaker manager",
			cbManager: &MockCircuitBreakerManagerForLLM{
				stats: map[string]interface{}{},
			},
			expectedStatus: health.StatusHealthy,
			checkContains:  "No circuit breakers registered",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Replace the circuit breaker manager
			originalCBMgr := suite.service.circuitBreakerMgr
			suite.service.circuitBreakerMgr = tc.cbManager
			defer func() {
				suite.service.circuitBreakerMgr = originalCBMgr
			}()

			// Re-register health checks
			suite.service.registerHealthChecks()

			// Execute the health check
			result := suite.service.healthChecker.RunCheck(suite.ctx, "circuit_breaker")

			// Verify the result
			require.NotNil(suite.T(), result)
			assert.Equal(suite.T(), tc.expectedStatus, result.Status)
			assert.Contains(suite.T(), result.Message, tc.checkContains)
		})
	}
}

// Mock implementations for testing

// MockCircuitBreakerManagerForLLM mocks circuit breaker manager for LLM processor tests
type MockCircuitBreakerManagerForLLM struct {
	stats map[string]interface{}
}

func (m *MockCircuitBreakerManagerForLLM) GetStats() (map[string]interface{}, error) {
	if m.stats == nil {
		return map[string]interface{}{}, nil
	}
	return m.stats, nil
}

// MockCircuitBreakerManagerErrorForLLM mocks circuit breaker manager that returns errors
type MockCircuitBreakerManagerErrorForLLM struct {
	shouldError bool
}

func (m *MockCircuitBreakerManagerErrorForLLM) GetStats() (map[string]interface{}, error) {
	if m.shouldError {
		return nil, assert.AnError
	}
	return map[string]interface{}{}, nil
}
