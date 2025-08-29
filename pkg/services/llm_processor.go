
package services



import (

	"context"

	"fmt"

	"log/slog"



	"github.com/nephio-project/nephoran-intent-operator/pkg/config"

	"github.com/nephio-project/nephoran-intent-operator/pkg/handlers"

	"github.com/nephio-project/nephoran-intent-operator/pkg/health"

	"github.com/nephio-project/nephoran-intent-operator/pkg/llm"

)



// LLMProcessorService manages the lifecycle of LLM processor components.

type LLMProcessorService struct {

	config             *config.LLMProcessorConfig

	secretManager      *config.SecretManager

	processor          *handlers.IntentProcessor

	streamingProcessor *llm.StreamingProcessorStub

	circuitBreakerMgr  *llm.CircuitBreakerManager

	tokenManager       *llm.TokenManager

	contextBuilder     *llm.ContextBuilder

	relevanceScorer    *llm.RelevanceScorer

	promptBuilder      *llm.RAGAwarePromptBuilder

	logger             *slog.Logger

	healthChecker      *health.HealthChecker

}



// NewLLMProcessorService creates a new service instance.

func NewLLMProcessorService(config *config.LLMProcessorConfig, logger *slog.Logger) *LLMProcessorService {

	return &LLMProcessorService{

		config: config,

		logger: logger,

	}

}



// Initialize initializes all components of the LLM processor service.

func (s *LLMProcessorService) Initialize(ctx context.Context) error {

	s.logger.Info("Initializing LLM Processor service components")



	// Initialize secret manager if enabled.

	if err := s.initializeSecretManager(); err != nil {

		return fmt.Errorf("failed to initialize secret manager: %w", err)

	}



	// Initialize health checker.

	s.healthChecker = health.NewHealthChecker("llm-processor", s.config.ServiceVersion, s.logger)



	// Initialize core components.

	if err := s.initializeLLMComponents(ctx); err != nil {

		return fmt.Errorf("failed to initialize LLM components: %w", err)

	}



	// Register health checks.

	s.registerHealthChecks()



	s.logger.Info("LLM Processor service components initialized successfully")

	return nil

}



// initializeSecretManager initializes the Kubernetes secret manager if enabled.

func (s *LLMProcessorService) initializeSecretManager() error {

	if s.config.UseKubernetesSecrets {

		var err error

		s.secretManager, err = config.NewSecretManager(s.config.SecretNamespace)

		if err != nil {

			s.logger.Error("Failed to initialize secret manager", slog.String("error", err.Error()))

			s.logger.Info("Falling back to environment variables for secrets")

			return nil // Non-critical error, fallback to env vars

		}

		s.logger.Info("Secret manager initialized successfully",

			slog.String("namespace", s.config.SecretNamespace))

	}

	return nil

}



// initializeLLMComponents initializes all LLM-related components.

func (s *LLMProcessorService) initializeLLMComponents(ctx context.Context) error {

	// Initialize token manager.

	s.tokenManager = llm.NewTokenManager()



	// Initialize circuit breaker manager.

	s.circuitBreakerMgr = llm.NewCircuitBreakerManager(nil)



	// Load API keys securely.

	apiKeys, err := s.loadSecureAPIKeys(ctx)

	if err != nil {

		s.logger.Error("Failed to load API keys", slog.String("error", err.Error()))

		return fmt.Errorf("failed to load API keys: %w", err)

	}



	// Use the secure API key for LLM client.

	apiKey := apiKeys.OpenAI

	if apiKey == "" {

		apiKey = s.config.LLMAPIKey // fallback to config

	}



	// Validate configuration before creating client.

	if err := s.validateLLMConfig(apiKey); err != nil {

		return fmt.Errorf("LLM configuration validation failed: %w", err)

	}



	// Create LLM client configuration.

	clientConfig := llm.ClientConfig{

		APIKey:      apiKey,

		ModelName:   s.config.LLMModelName,

		MaxTokens:   s.config.LLMMaxTokens,

		BackendType: s.config.LLMBackendType,

		Timeout:     s.config.LLMTimeout,

	}



	// Create base LLM client.

	llmClient := llm.NewClientWithConfig(s.config.RAGAPIURL, clientConfig)

	if llmClient == nil {

		return fmt.Errorf("failed to create LLM client - nil client returned")

	}



	// Initialize relevance scorer stub.

	s.relevanceScorer = llm.NewRelevanceScorerStub()



	// Initialize context builder if enabled.

	if s.config.EnableContextBuilder {

		s.contextBuilder = llm.NewContextBuilderStub()

	}



	// Initialize prompt builder.

	s.promptBuilder = llm.NewRAGAwarePromptBuilder(s.tokenManager, nil)



	// Initialize RAG-enhanced processor if enabled (stubbed).

	var ragEnhanced interface{} = nil // Stubbed

	if s.config.RAGEnabled {

		// ragEnhanced = llm.NewRAGEnhancedProcessor(llmClient, nil, nil, nil) // Stubbed.

	}



	// Initialize streaming processor if enabled.

	if s.config.StreamingEnabled {

		// StreamingConfig doesn't exist, use basic streaming processor.

		s.streamingProcessor = llm.NewStreamingProcessor()

	}



	// Initialize main processor with circuit breaker.

	circuitBreaker := s.circuitBreakerMgr.GetOrCreate("llm-processor", nil)

	s.processor = &handlers.IntentProcessor{

		LLMClient:         llmClient,

		RAGEnhancedClient: ragEnhanced,

		CircuitBreaker:    circuitBreaker,

		Logger:            s.logger,

	}



	return nil

}



// validateLLMConfig validates the LLM configuration.

func (s *LLMProcessorService) validateLLMConfig(apiKey string) error {

	if apiKey == "" && s.config.LLMBackendType != "mock" && s.config.LLMBackendType != "rag" {

		return fmt.Errorf("API Key is required for non-mock/non-rag backends")

	}

	if s.config.LLMModelName == "" {

		return fmt.Errorf("model name is required")

	}

	if s.config.LLMMaxTokens <= 0 {

		return fmt.Errorf("max tokens must be greater than 0")

	}

	if s.config.LLMTimeout <= 0 {

		return fmt.Errorf("timeout must be greater than 0")

	}

	if s.config.RAGAPIURL == "" {

		return fmt.Errorf("RAG API URL is required but not configured")

	}

	return nil

}



// loadSecureAPIKeys loads API keys from files, Kubernetes secrets, or environment variables.

func (s *LLMProcessorService) loadSecureAPIKeys(ctx context.Context) (*config.APIKeys, error) {

	// Try file-based secrets first.

	fileAPIKeys, err := config.LoadFileBasedAPIKeysWithValidation()

	if err == nil && !fileAPIKeys.IsEmpty() {

		s.logger.Info("Loaded API keys from files")

		return fileAPIKeys, nil

	}

	if err != nil {

		s.logger.Info("Failed to load API keys from files", "error", err.Error())

	}



	// If file loading failed or returned empty keys, try Kubernetes secrets.

	if s.secretManager != nil {

		k8sAPIKeys, err := s.secretManager.GetAPIKeys(ctx)

		if err == nil && !k8sAPIKeys.IsEmpty() {

			s.logger.Info("Loaded API keys from Kubernetes secrets")

			return k8sAPIKeys, nil

		}

	}



	// Fall back to environment variables as last resort.

	s.logger.Info("Falling back to environment variables for API keys")

	return &config.APIKeys{

		OpenAI:    config.GetEnvOrDefault("OPENAI_API_KEY", ""),

		Weaviate:  config.GetEnvOrDefault("WEAVIATE_API_KEY", ""),

		Generic:   config.GetEnvOrDefault("API_KEY", ""),

		JWTSecret: config.GetEnvOrDefault("JWT_SECRET_KEY", ""),

	}, nil

}



// registerHealthChecks registers all health checks for the service.

func (s *LLMProcessorService) registerHealthChecks() {

	// Internal service health checks.

	s.healthChecker.RegisterCheck("service_status", func(ctx context.Context) *health.Check {

		return &health.Check{

			Status:  health.StatusHealthy,

			Message: "Service is running normally",

		}

	})



	// Circuit breaker health check.

	if s.circuitBreakerMgr != nil {

		s.healthChecker.RegisterCheck("circuit_breaker", func(ctx context.Context) *health.Check {

			stats := s.circuitBreakerMgr.GetAllStats()

			if len(stats) == 0 {

				return &health.Check{

					Status:  health.StatusHealthy,

					Message: "No circuit breakers registered",

				}

			}



			// Check if any circuit breakers are open.

			// Preallocate slice with expected capacity for performance.

			openBreakers := make([]string, 0, len(stats))

			for name, cbStats := range stats {

				if cbMap, ok := cbStats.(map[string]interface{}); ok {

					if state, ok := cbMap["state"].(string); ok && state == "open" {

						openBreakers = append(openBreakers, name)

					}

				}

			}



			if len(openBreakers) > 0 {

				return &health.Check{

					Status:  health.StatusUnhealthy,

					Message: fmt.Sprintf("Circuit breakers in open state: %v", openBreakers),

				}

			}



			return &health.Check{

				Status:  health.StatusHealthy,

				Message: fmt.Sprintf("All %d circuit breakers operational", len(stats)),

			}

		})

	}



	// Token manager health check.

	if s.tokenManager != nil {

		s.healthChecker.RegisterCheck("token_manager", func(ctx context.Context) *health.Check {

			models := s.tokenManager.GetSupportedModels()

			return &health.Check{

				Status:  health.StatusHealthy,

				Message: fmt.Sprintf("Token manager operational with %d supported models", len(models)),

				Metadata: map[string]interface{}{

					"supported_models": models,

				},

			}

		})

	}



	// Streaming processor health check.

	if s.streamingProcessor != nil {

		s.healthChecker.RegisterCheck("streaming_processor", func(ctx context.Context) *health.Check {

			metrics := s.streamingProcessor.GetMetrics()

			return &health.Check{

				Status:   health.StatusHealthy,

				Message:  "Streaming processor operational",

				Metadata: metrics,

			}

		})

	}



	// RAG API dependency check with smart endpoint detection.

	if s.config.RAGEnabled && s.config.RAGAPIURL != "" {

		_, healthEndpoint := s.config.GetEffectiveRAGEndpoints()

		s.healthChecker.RegisterDependency("rag_api", health.HTTPCheck("rag_api", healthEndpoint))

	}



	s.logger.Info("Health checks registered")

}



// GetComponents returns the initialized components.

func (s *LLMProcessorService) GetComponents() (

	*handlers.IntentProcessor,

	*llm.StreamingProcessorStub,

	*llm.CircuitBreakerManager,

	*llm.TokenManager,

	*llm.ContextBuilder,

	*llm.RelevanceScorer,

	*llm.RAGAwarePromptBuilder,

	*health.HealthChecker,

) {

	return s.processor, s.streamingProcessor, s.circuitBreakerMgr, s.tokenManager,

		s.contextBuilder, s.relevanceScorer, s.promptBuilder, s.healthChecker

}



// Shutdown gracefully shuts down the service.

func (s *LLMProcessorService) Shutdown(ctx context.Context) error {

	s.logger.Info("Shutting down LLM Processor service")



	// Mark service as not ready.

	if s.healthChecker != nil {

		s.healthChecker.SetReady(false)

	}



	// Shutdown streaming processor if enabled.

	if s.streamingProcessor != nil {

		if err := s.streamingProcessor.Shutdown(ctx); err != nil {

			s.logger.Error("Failed to shutdown streaming processor", slog.String("error", err.Error()))

		}

	}



	// Close circuit breakers.

	if s.circuitBreakerMgr != nil {

		s.circuitBreakerMgr.Shutdown()

	}



	s.logger.Info("LLM Processor service shutdown completed")

	return nil

}



// Note: Helper functions have been moved to pkg/config/env_helpers.go.

