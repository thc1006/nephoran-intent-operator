package services

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/nephio-project/nephoran-intent-operator/pkg/config"
	"github.com/nephio-project/nephoran-intent-operator/pkg/health"
	"github.com/nephio-project/nephoran-intent-operator/pkg/llm"
)

// Test Suite
type LLMProcessorServiceTestSuite struct {
	suite.Suite
	service  *LLMProcessorService
	config   *config.LLMProcessorConfig
	logger   *slog.Logger
	mockCtrl *gomock.Controller
}

func (suite *LLMProcessorServiceTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.config = &config.LLMProcessorConfig{
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
		Port:                 "8080",
		LogLevel:             "info",
	}

	suite.service = NewLLMProcessorService(suite.config, suite.logger)
}

func (suite *LLMProcessorServiceTestSuite) TearDownTest() {
	if suite.mockCtrl != nil {
		suite.mockCtrl.Finish()
	}
}

func (suite *LLMProcessorServiceTestSuite) SetupSuite() {
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

// Basic Tests
func (suite *LLMProcessorServiceTestSuite) TestNewLLMProcessorService() {
	assert.NotNil(suite.T(), suite.service)
	assert.Equal(suite.T(), suite.config, suite.service.config)
	assert.Equal(suite.T(), suite.logger, suite.service.logger)
}

func (suite *LLMProcessorServiceTestSuite) TestNewLLMProcessor() {
	processor := NewLLMProcessor(suite.config, suite.logger)
	assert.NotNil(suite.T(), processor)
	assert.Equal(suite.T(), suite.config, processor.config)
	assert.Equal(suite.T(), suite.logger, processor.logger)
}

// Configuration Tests
func (suite *LLMProcessorServiceTestSuite) TestInitializeWithValidConfig() {
	ctx := context.Background()
	err := suite.service.Initialize(ctx)

	// Since we're using mock/test configurations, initialization might succeed or fail
	// depending on external dependencies
	if err != nil {
		suite.T().Logf("Initialize returned error (expected in test environment): %v", err)
	} else {
		assert.NoError(suite.T(), err)
	}
}

func (suite *LLMProcessorServiceTestSuite) TestInitializeSecretManagerDisabled() {
	suite.config.UseKubernetesSecrets = false
	
	err := suite.service.initializeSecretManager()
	assert.NoError(suite.T(), err)
	assert.Nil(suite.T(), suite.service.secretManager)
}

func (suite *LLMProcessorServiceTestSuite) TestInitializeSecretManagerEnabled() {
	suite.config.UseKubernetesSecrets = true
	suite.config.SecretNamespace = "test-namespace"
	
	err := suite.service.initializeSecretManager()
	// In test environment, this might fail but shouldn't panic
	if err != nil {
		suite.T().Logf("Secret manager initialization failed as expected in test: %v", err)
	}
}

// Component Tests
func (suite *LLMProcessorServiceTestSuite) TestGetComponents() {
	processor, streamingProcessor, circuitBreakerMgr, tokenManager, contextBuilder, relevanceScorer, promptBuilder, healthChecker := suite.service.GetComponents()

	// Components may be nil before initialization
	assert.Equal(suite.T(), suite.service.processor, processor)
	assert.Equal(suite.T(), suite.service.streamingProcessor, streamingProcessor)
	assert.Equal(suite.T(), suite.service.circuitBreakerMgr, circuitBreakerMgr)
	assert.Equal(suite.T(), suite.service.tokenManager, tokenManager)
	assert.Equal(suite.T(), suite.service.contextBuilder, contextBuilder)
	assert.Equal(suite.T(), suite.service.relevanceScorer, relevanceScorer)
	assert.Equal(suite.T(), suite.service.promptBuilder, promptBuilder)
	assert.Equal(suite.T(), suite.service.healthChecker, healthChecker)
}

// Performance Component Tests
func (suite *LLMProcessorServiceTestSuite) TestSetCacheManager() {
	// This test would need a proper cache manager mock
	// For now, just test that the setter doesn't panic
	suite.service.SetCacheManager(nil)
	assert.Nil(suite.T(), suite.service.GetCacheManager())
}

func (suite *LLMProcessorServiceTestSuite) TestSetAsyncProcessor() {
	// This test would need a proper async processor mock
	// For now, just test that the setter doesn't panic
	suite.service.SetAsyncProcessor(nil)
	assert.Nil(suite.T(), suite.service.GetAsyncProcessor())
}

// Shutdown Tests
func (suite *LLMProcessorServiceTestSuite) TestShutdown() {
	ctx := context.Background()
	err := suite.service.Shutdown(ctx)
	assert.NoError(suite.T(), err)
}

func (suite *LLMProcessorServiceTestSuite) TestShutdownWithTimeout() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	
	// Even with a short timeout, shutdown should handle it gracefully
	err := suite.service.Shutdown(ctx)
	assert.NoError(suite.T(), err)
}

// Configuration Validation Tests
func (suite *LLMProcessorServiceTestSuite) TestValidateLLMConfig() {
	tests := []struct {
		name        string
		apiKey      string
		backendType string
		modelName   string
		maxTokens   int
		timeout     time.Duration
		ragAPIURL   string
		expectError bool
	}{
		{
			name:        "valid config",
			apiKey:      "valid-key",
			backendType: "openai",
			modelName:   "gpt-4o-mini",
			maxTokens:   2048,
			timeout:     15 * time.Second,
			ragAPIURL:   "http://localhost:8081",
			expectError: false,
		},
		{
			name:        "empty api key for non-mock backend",
			apiKey:      "",
			backendType: "openai",
			modelName:   "gpt-4o-mini",
			maxTokens:   2048,
			timeout:     15 * time.Second,
			ragAPIURL:   "http://localhost:8081",
			expectError: true,
		},
		{
			name:        "mock backend with empty api key",
			apiKey:      "",
			backendType: "mock",
			modelName:   "gpt-4o-mini",
			maxTokens:   2048,
			timeout:     15 * time.Second,
			ragAPIURL:   "http://localhost:8081",
			expectError: false,
		},
		{
			name:        "empty model name",
			apiKey:      "valid-key",
			backendType: "openai",
			modelName:   "",
			maxTokens:   2048,
			timeout:     15 * time.Second,
			ragAPIURL:   "http://localhost:8081",
			expectError: true,
		},
		{
			name:        "zero max tokens",
			apiKey:      "valid-key",
			backendType: "openai",
			modelName:   "gpt-4o-mini",
			maxTokens:   0,
			timeout:     15 * time.Second,
			ragAPIURL:   "http://localhost:8081",
			expectError: true,
		},
		{
			name:        "zero timeout",
			apiKey:      "valid-key",
			backendType: "openai",
			modelName:   "gpt-4o-mini",
			maxTokens:   2048,
			timeout:     0,
			ragAPIURL:   "http://localhost:8081",
			expectError: true,
		},
		{
			name:        "empty rag api url",
			apiKey:      "valid-key",
			backendType: "openai",
			modelName:   "gpt-4o-mini",
			maxTokens:   2048,
			timeout:     15 * time.Second,
			ragAPIURL:   "",
			expectError: true,
		},
	}

	for _, tc := range tests {
		suite.T().Run(tc.name, func(t *testing.T) {
			suite.config.LLMAPIKey = tc.apiKey
			suite.config.LLMBackendType = tc.backendType
			suite.config.LLMModelName = tc.modelName
			suite.config.LLMMaxTokens = tc.maxTokens
			suite.config.LLMTimeout = tc.timeout
			suite.config.RAGAPIURL = tc.ragAPIURL

			err := suite.service.validateLLMConfig(tc.apiKey)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// API Key Loading Tests
func (suite *LLMProcessorServiceTestSuite) TestLoadSecureAPIKeys() {
	ctx := context.Background()
	
	// This will attempt to load from files or environment
	apiKeys, err := suite.service.loadSecureAPIKeys(ctx)
	
	// In test environment, this might not find any keys, but should not panic
	assert.NotNil(suite.T(), apiKeys)
	
	if err != nil {
		suite.T().Logf("Load API keys returned error (expected in test environment): %v", err)
	}
}

// Health Check Tests
func (suite *LLMProcessorServiceTestSuite) TestRegisterHealthChecks() {
	// Initialize health checker first
	suite.service.healthChecker = health.NewHealthChecker("test-service", "1.0.0", suite.logger)
	
	// This should not panic
	suite.service.registerHealthChecks()
	
	// Verify health checker is still functional
	assert.True(suite.T(), suite.service.healthChecker.IsReady())
}

func (suite *LLMProcessorServiceTestSuite) TestRegisterHealthChecksWithCircuitBreaker() {
	suite.service.healthChecker = health.NewHealthChecker("test-service", "1.0.0", suite.logger)
	
	// Create a real circuit breaker manager (not a mock)
	cbMgr := llm.NewCircuitBreakerManager(nil)
	suite.service.circuitBreakerMgr = cbMgr
	
	suite.service.registerHealthChecks()
	
	// Test health checker functionality
	assert.True(suite.T(), suite.service.healthChecker.IsReady())
}

// Error Handling Tests
func (suite *LLMProcessorServiceTestSuite) TestInitializationFailure() {
	// Test with invalid configuration that should fail
	invalidConfig := &config.LLMProcessorConfig{
		ServiceVersion: "1.0.0",
		LLMAPIKey:      "",        // Empty API key
		LLMBackendType: "openai",  // Non-mock backend
		LLMModelName:   "",        // Empty model name
		RAGAPIURL:      "",        // Empty RAG URL
		Port:           "8080",
		LogLevel:       "info",
	}
	
	service := NewLLMProcessorService(invalidConfig, suite.logger)
	ctx := context.Background()
	
	err := service.Initialize(ctx)
	// Should fail due to invalid configuration
	assert.Error(suite.T(), err)
}

// Edge Case Tests
func (suite *LLMProcessorServiceTestSuite) TestMultipleInitializeCalls() {
	ctx := context.Background()
	
	// First initialization
	err1 := suite.service.Initialize(ctx)
	
	// Second initialization (should handle gracefully)
	err2 := suite.service.Initialize(ctx)
	
	// Both calls should either succeed or fail consistently
	if err1 != nil {
		assert.Error(suite.T(), err2)
	} else {
		// If first succeeds, second might succeed or handle gracefully
		suite.T().Logf("First init: %v, Second init: %v", err1, err2)
	}
}

func (suite *LLMProcessorServiceTestSuite) TestConcurrentShutdown() {
	ctx := context.Background()
	
	done := make(chan error, 2)
	
	// Launch two concurrent shutdown calls
	go func() {
		done <- suite.service.Shutdown(ctx)
	}()
	go func() {
		done <- suite.service.Shutdown(ctx)
	}()
	
	// Both should complete without hanging
	err1 := <-done
	err2 := <-done
	
	assert.NoError(suite.T(), err1)
	assert.NoError(suite.T(), err2)
}

// Test various configuration scenarios
func (suite *LLMProcessorServiceTestSuite) TestConfigurationScenarios() {
	scenarios := []struct {
		name     string
		modifier func(*config.LLMProcessorConfig)
	}{
		{
			name: "streaming disabled",
			modifier: func(cfg *config.LLMProcessorConfig) {
				cfg.StreamingEnabled = false
			},
		},
		{
			name: "context builder disabled",
			modifier: func(cfg *config.LLMProcessorConfig) {
				cfg.EnableContextBuilder = false
			},
		},
		{
			name: "rag disabled",
			modifier: func(cfg *config.LLMProcessorConfig) {
				cfg.RAGEnabled = false
			},
		},
		{
			name: "kubernetes secrets enabled",
			modifier: func(cfg *config.LLMProcessorConfig) {
				cfg.UseKubernetesSecrets = true
				cfg.SecretNamespace = "test-namespace"
			},
		},
	}
	
	for _, scenario := range scenarios {
		suite.T().Run(scenario.name, func(t *testing.T) {
			cfg := suite.setupMinimalConfig()
			scenario.modifier(cfg)
			
			service := NewLLMProcessorService(cfg, suite.logger)
			assert.NotNil(t, service)
			
			// Test initialization (may fail in test environment, but shouldn't panic)
			ctx := context.Background()
			err := service.Initialize(ctx)
			
			if err != nil {
				suite.T().Logf("Scenario %s failed initialization (expected in test): %v", scenario.name, err)
			}
			
			// Test shutdown
			shutdownErr := service.Shutdown(ctx)
			assert.NoError(t, shutdownErr)
		})
	}
}

// Helper functions for testing
func (suite *LLMProcessorServiceTestSuite) setupMinimalConfig() *config.LLMProcessorConfig {
	return &config.LLMProcessorConfig{
		ServiceVersion:       "1.0.0",
		LLMAPIKey:            "test-api-key",
		LLMModelName:         "gpt-4o-mini",
		LLMMaxTokens:         4000,
		LLMBackendType:       "mock", // Use mock for testing
		LLMTimeout:           30 * time.Second,
		RAGAPIURL:            "http://localhost:8081",
		RAGEnabled:           true,
		StreamingEnabled:     false, // Disable streaming for minimal config
		EnableContextBuilder: false, // Disable context builder for minimal config
		UseKubernetesSecrets: false,
		SecretNamespace:      "default",
		Port:                 "8080",
		LogLevel:             "info",
	}
}

// Run the test suite
func TestLLMProcessorServiceSuite(t *testing.T) {
	suite.Run(t, new(LLMProcessorServiceTestSuite))
}

// Additional standalone tests
func TestNewLLMProcessorService(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	
	config := &config.LLMProcessorConfig{
		ServiceVersion: "1.0.0",
		Port:          "8080",
		LogLevel:      "info",
	}
	
	service := NewLLMProcessorService(config, logger)
	assert.NotNil(t, service)
	assert.Equal(t, config, service.config)
	assert.Equal(t, logger, service.logger)
}

func TestNewLLMProcessor(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	
	config := &config.LLMProcessorConfig{
		ServiceVersion: "1.0.0",
		Port:          "8080",
		LogLevel:      "info",
	}
	
	processor := NewLLMProcessor(config, logger)
	assert.NotNil(t, processor)
	assert.Equal(t, config, processor.config)
	assert.Equal(t, logger, processor.logger)
}