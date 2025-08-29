//go:build !disable_rag
// +build !disable_rag

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/thc1006/nephoran-intent-operator/pkg/auth"
	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/handlers"
	"github.com/thc1006/nephoran-intent-operator/pkg/health"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
)

// ServiceManager manages the overall service lifecycle and components.
type ServiceManager struct {
	config             *config.LLMProcessorConfig
	logger             *slog.Logger
	healthChecker      *health.HealthChecker
	secretManager      *config.SecretManager
	oauth2Manager      *auth.OAuth2Manager
	processor          *handlers.IntentProcessor
	streamingProcessor interface {
		HandleStreamingRequest(w http.ResponseWriter, r *http.Request, req *llm.StreamingRequest) error
		GetMetrics() map[string]interface{}
	}
	circuitBreakerMgr *llm.CircuitBreakerManager
	tokenManager      *llm.TokenManager
	contextBuilder    *llm.ContextBuilder
	relevanceScorer   *llm.RelevanceScorer
	promptBuilder     *llm.RAGAwarePromptBuilder
}

// NewServiceManager creates a new service manager.
func NewServiceManager(config *config.LLMProcessorConfig, logger *slog.Logger) *ServiceManager {
	return &ServiceManager{
		config: config,
		logger: logger,
	}
}

// Initialize initializes all service components.
func (sm *ServiceManager) Initialize(ctx context.Context) error {
	// Initialize health checker.
	sm.healthChecker = health.NewHealthChecker("llm-processor", sm.config.ServiceVersion, sm.logger)

	// Initialize secret manager.
	if err := sm.initializeSecretManager(); err != nil {
		return fmt.Errorf("failed to initialize secret manager: %w", err)
	}

	// Initialize OAuth2 manager.
	if err := sm.initializeOAuth2Manager(); err != nil {
		return fmt.Errorf("failed to initialize OAuth2 manager: %w", err)
	}

	// Initialize processing components.
	if err := sm.initializeProcessingComponents(ctx); err != nil {
		return fmt.Errorf("failed to initialize processing components: %w", err)
	}

	// Register health checks.
	sm.registerHealthChecks()

	sm.logger.Info("Service manager initialized successfully")
	return nil
}

// initializeSecretManager initializes the secret manager.
func (sm *ServiceManager) initializeSecretManager() error {
	var err error
	if sm.config.UseKubernetesSecrets {
		sm.secretManager, err = config.NewSecretManager(sm.config.SecretNamespace)
		if err != nil {
			sm.logger.Error("Failed to initialize secret manager", slog.String("error", err.Error()))
			sm.logger.Info("Falling back to environment variables for secrets")
			return nil // Continue without secret manager
		}
		sm.logger.Info("Secret manager initialized successfully",
			slog.String("namespace", sm.config.SecretNamespace))
	}
	return nil
}

// initializeOAuth2Manager initializes the OAuth2 manager.
func (sm *ServiceManager) initializeOAuth2Manager() error {
	oauth2Config := &auth.OAuth2ManagerConfig{
		Enabled:        sm.config.AuthEnabled,
		AuthConfigFile: sm.config.AuthConfigFile,
		JWTSecretKey:   sm.config.JWTSecretKey,
		RequireAuth:    sm.config.RequireAuth,
		AdminUsers:     sm.config.AdminUsers,
		OperatorUsers:  sm.config.OperatorUsers,
	}

	if err := oauth2Config.Validate(); err != nil {
		return err
	}

	var err error
	sm.oauth2Manager, err = auth.NewOAuth2Manager(oauth2Config, sm.logger)
	if err != nil {
		return fmt.Errorf("failed to create OAuth2 manager: %w", err)
	}

	return nil
}

// initializeProcessingComponents initializes all LLM and RAG processing components.
func (sm *ServiceManager) initializeProcessingComponents(ctx context.Context) error {
	// Initialize token manager.
	sm.tokenManager = llm.NewTokenManager()

	// Initialize circuit breaker manager.
	sm.circuitBreakerMgr = llm.NewCircuitBreakerManager(nil)

	// Validate configuration.
	if sm.config.RAGAPIURL == "" {
		return fmt.Errorf("RAG API URL is required but not configured")
	}

	// Load API keys securely.
	apiKeys, err := sm.loadSecureAPIKeys(ctx)
	if err != nil {
		return fmt.Errorf("failed to load API keys: %w", err)
	}

	// Use the secure API key for LLM client.
	apiKey := apiKeys.OpenAI
	if apiKey == "" {
		apiKey = sm.config.LLMAPIKey // fallback to config
	}

	clientConfig := llm.ClientConfig{
		APIKey:      apiKey,
		ModelName:   sm.config.LLMModelName,
		MaxTokens:   sm.config.LLMMaxTokens,
		BackendType: sm.config.LLMBackendType,
		Timeout:     sm.config.LLMTimeout,
	}

	// Validate client configuration.
	if err := sm.validateClientConfig(clientConfig); err != nil {
		return err
	}

	llmClient := llm.NewClientWithConfig(sm.config.RAGAPIURL, clientConfig)
	if llmClient == nil {
		return fmt.Errorf("failed to create LLM client - nil client returned")
	}

	// Initialize supporting components.
	sm.relevanceScorer = llm.NewRelevanceScorerStub()

	if sm.config.EnableContextBuilder {
		sm.contextBuilder = llm.NewContextBuilderStub()
	}

	sm.promptBuilder = llm.NewRAGAwarePromptBuilder(sm.tokenManager, nil)

	// Initialize RAG-enhanced processor if enabled (stubbed).
	var ragEnhanced interface{} = nil // Stubbed
	if sm.config.RAGEnabled {
		// ragEnhanced = llm.NewRAGEnhancedProcessor(llmClient, nil, nil, nil) // Stubbed.
	}

	// Initialize streaming processor if enabled.
	if sm.config.StreamingEnabled {
		// Use stub implementation for now.
		sm.streamingProcessor = llm.NewStreamingProcessor()
	}

	// Initialize main processor with circuit breaker.
	circuitBreaker := sm.circuitBreakerMgr.GetOrCreate("llm-processor", nil)
	sm.processor = &handlers.IntentProcessor{
		LLMClient:         llmClient,
		RAGEnhancedClient: ragEnhanced,
		CircuitBreaker:    circuitBreaker,
		Logger:            sm.logger,
	}

	return nil
}

// registerHealthChecks registers all health checks for the service.
func (sm *ServiceManager) registerHealthChecks() {
	// Internal service health checks.
	sm.healthChecker.RegisterCheck("service_status", func(ctx context.Context) *health.Check {
		return &health.Check{
			Status:  health.StatusHealthy,
			Message: "Service is running normally",
		}
	})

	// Circuit breaker health check.
	sm.healthChecker.RegisterCheck("circuit_breaker", func(ctx context.Context) *health.Check {
		if sm.circuitBreakerMgr == nil {
			return &health.Check{
				Status:  health.StatusHealthy,
				Message: "No circuit breakers registered",
			}
		}

		stats := sm.circuitBreakerMgr.GetAllStats()
		if len(stats) == 0 {
			return &health.Check{
				Status:  health.StatusHealthy,
				Message: "No circuit breakers registered",
			}
		}

		// Check if any circuit breakers are open.
		// Preallocate slice with expected capacity for performance.
		openBreakers := make([]string, 0, len(stats))
		for name, state := range stats {
			if cbStats, ok := state.(map[string]interface{}); ok {
				if cbState, exists := cbStats["state"]; exists && cbState == "open" {
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
			Message: "All circuit breakers operational",
		}
	})

	// Token manager health check.
	if sm.tokenManager != nil {
		sm.healthChecker.RegisterCheck("token_manager", func(ctx context.Context) *health.Check {
			models := sm.tokenManager.GetSupportedModels()
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
	if sm.streamingProcessor != nil {
		sm.healthChecker.RegisterCheck("streaming_processor", func(ctx context.Context) *health.Check {
			metrics := sm.streamingProcessor.GetMetrics()
			return &health.Check{
				Status:   health.StatusHealthy,
				Message:  "Streaming processor operational",
				Metadata: metrics,
			}
		})
	}

	// RAG API dependency check with smart endpoint detection.
	if sm.config.RAGEnabled && sm.config.RAGAPIURL != "" {
		_, healthEndpoint := sm.config.GetEffectiveRAGEndpoints()
		sm.healthChecker.RegisterDependency("rag_api", health.HTTPCheck("rag_api", healthEndpoint))
	}

	sm.logger.Info("Health checks registered")
}

// loadSecureAPIKeys loads API keys from Kubernetes secrets or environment variables.
func (sm *ServiceManager) loadSecureAPIKeys(ctx context.Context) (*config.APIKeys, error) {
	if sm.secretManager == nil {
		// Fall back to environment variables.
		return &config.APIKeys{
			OpenAI:    getEnvString("OPENAI_API_KEY", ""),
			Weaviate:  getEnvString("WEAVIATE_API_KEY", ""),
			Generic:   getEnvString("API_KEY", ""),
			JWTSecret: getEnvString("JWT_SECRET_KEY", ""),
		}, nil
	}

	return sm.secretManager.GetAPIKeys(ctx)
}

// validateClientConfig validates the LLM client configuration.
func (sm *ServiceManager) validateClientConfig(config llm.ClientConfig) error {
	if config.APIKey == "" && config.BackendType != "mock" {
		return fmt.Errorf("API Key is required for non-mock backends")
	}
	if config.ModelName == "" {
		return fmt.Errorf("model name is required")
	}
	if config.MaxTokens <= 0 {
		return fmt.Errorf("max tokens must be greater than 0")
	}
	if config.Timeout <= 0 {
		return fmt.Errorf("timeout must be greater than 0")
	}
	return nil
}

// CreateRouter creates and configures the HTTP router.
func (sm *ServiceManager) CreateRouter() *mux.Router {
	router := mux.NewRouter()

	// Setup OAuth2 routes.
	sm.oauth2Manager.SetupRoutes(router)

	// Public health endpoints (no authentication required).
	router.HandleFunc("/healthz", sm.healthChecker.HealthzHandler).Methods("GET")
	router.HandleFunc("/readyz", sm.healthChecker.ReadyzHandler).Methods("GET")
	router.HandleFunc("/metrics", sm.metricsHandler).Methods("GET")

	// Setup protected/unprotected routes based on configuration.
	handlers := &auth.RouteHandlers{
		ProcessIntent:        sm.processIntentHandler,
		Status:               sm.statusHandler,
		CircuitBreakerStatus: sm.circuitBreakerStatusHandler,
		Metrics:              sm.metricsHandler,
	}

	if sm.config.StreamingEnabled {
		handlers.StreamingHandler = sm.streamingHandler
	}

	sm.oauth2Manager.ConfigureProtectedRoutes(router, handlers)

	return router
}

// CreateServer creates the HTTP server.
func (sm *ServiceManager) CreateServer(router *mux.Router) *http.Server {
	return &http.Server{
		Addr:         ":" + sm.config.Port,
		Handler:      router,
		ReadTimeout:  sm.config.RequestTimeout,
		WriteTimeout: sm.config.RequestTimeout,
		IdleTimeout:  2 * time.Minute,
	}
}

// MarkReady marks the service as ready.
func (sm *ServiceManager) MarkReady() {
	sm.healthChecker.SetReady(true)
}

// MarkNotReady marks the service as not ready.
func (sm *ServiceManager) MarkNotReady() {
	sm.healthChecker.SetReady(false)
}

// GetHealthChecker returns the health checker.
func (sm *ServiceManager) GetHealthChecker() *health.HealthChecker {
	return sm.healthChecker
}

// GetOAuth2Manager returns the OAuth2 manager.
func (sm *ServiceManager) GetOAuth2Manager() *auth.OAuth2Manager {
	return sm.oauth2Manager
}

// GetProcessor returns the intent processor.
func (sm *ServiceManager) GetProcessor() *handlers.IntentProcessor {
	return sm.processor
}

// GetStreamingProcessor returns the streaming processor.
func (sm *ServiceManager) GetStreamingProcessor() interface {
	HandleStreamingRequest(w http.ResponseWriter, r *http.Request, req *llm.StreamingRequest) error
	GetMetrics() map[string]interface{}
} {
	return sm.streamingProcessor
}

// GetCircuitBreakerMgr returns the circuit breaker manager.
func (sm *ServiceManager) GetCircuitBreakerMgr() *llm.CircuitBreakerManager {
	return sm.circuitBreakerMgr
}

// getEnvString gets a string environment variable with a default value.
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// HTTP Handlers.

// processIntentHandler handles intent processing requests.
func (sm *ServiceManager) processIntentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	startTime := time.Now()
	reqID := fmt.Sprintf("%d", time.Now().UnixNano())

	sm.logger.Info("Processing intent request", slog.String("request_id", reqID))

	var req handlers.ProcessIntentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sm.logger.Error("Failed to decode request", slog.String("error", err.Error()))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Intent == "" {
		sm.logger.Error("Empty intent provided")
		http.Error(w, "Intent is required", http.StatusBadRequest)
		return
	}

	// Process intent.
	result, err := sm.processor.ProcessIntent(r.Context(), req.Intent, req.Metadata)
	if err != nil {
		sm.logger.Error("Failed to process intent",
			slog.String("error", err.Error()),
			slog.String("intent", req.Intent),
		)
		response := handlers.ProcessIntentResponse{
			Status:         "error",
			Error:          err.Error(),
			RequestID:      reqID,
			ServiceVersion: sm.config.ServiceVersion,
			ProcessingTime: time.Since(startTime).String(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		// FIXME: Adding error check for json encoder per errcheck linter.

		if err := json.NewEncoder(w).Encode(response); err != nil {

			sm.logger.Error("Error encoding JSON", slog.String("error", err.Error()))

			return

		}
		return
	}

	response := handlers.ProcessIntentResponse{
		Result:         result.Result,
		Status:         "success",
		ProcessingTime: time.Since(startTime).String(),
		RequestID:      reqID,
		ServiceVersion: sm.config.ServiceVersion,
	}

	w.Header().Set("Content-Type", "application/json")
	// FIXME: Adding error check for json encoder per errcheck linter.

	if err := json.NewEncoder(w).Encode(response); err != nil {

		sm.logger.Error("Error encoding JSON", slog.String("error", err.Error()))

		return

	}

	sm.logger.Info("Intent processed successfully",
		slog.String("request_id", reqID),
		slog.Duration("processing_time", time.Since(startTime)),
	)
}

// statusHandler provides service status information.
func (sm *ServiceManager) statusHandler(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"service":        "llm-processor",
		"version":        sm.config.ServiceVersion,
		"uptime":         time.Since(startTime).String(),
		"healthy":        sm.healthChecker.IsHealthy(),
		"ready":          sm.healthChecker.IsReady(),
		"backend_type":   sm.config.LLMBackendType,
		"model":          sm.config.LLMModelName,
		"rag_enabled":    sm.config.RAGEnabled,
		"authentication": sm.oauth2Manager.GetAuthenticationInfo(),
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	// FIXME: Adding error check for json encoder per errcheck linter.

	if err := json.NewEncoder(w).Encode(status); err != nil {

		sm.logger.Error("Error encoding JSON", slog.String("error", err.Error()))

		return

	}
}

// metricsHandler provides comprehensive metrics.
func (sm *ServiceManager) metricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := map[string]interface{}{
		"service": "llm-processor",
		"version": sm.config.ServiceVersion,
		"uptime":  time.Since(startTime).String(),
	}

	// Add token manager metrics.
	if sm.tokenManager != nil {
		metrics["supported_models"] = sm.tokenManager.GetSupportedModels()
	}

	// Add circuit breaker metrics.
	if sm.circuitBreakerMgr != nil {
		metrics["circuit_breakers"] = sm.circuitBreakerMgr.GetAllStats()
	}

	// Add streaming metrics.
	if sm.streamingProcessor != nil {
		metrics["streaming"] = sm.streamingProcessor.GetMetrics()
	}

	// Add context builder metrics.
	if sm.contextBuilder != nil {
		metrics["context_builder"] = sm.contextBuilder.GetMetrics()
	}

	// Add relevance scorer metrics.
	if sm.relevanceScorer != nil {
		metrics["relevance_scorer"] = sm.relevanceScorer.GetMetrics()
	}

	// Add prompt builder metrics.
	if sm.promptBuilder != nil {
		metrics["prompt_builder"] = sm.promptBuilder.GetMetrics()
	}

	w.Header().Set("Content-Type", "application/json")
	// FIXME: Adding error check for json encoder per errcheck linter.

	if err := json.NewEncoder(w).Encode(metrics); err != nil {

		sm.logger.Error("Error encoding JSON", slog.String("error", err.Error()))

		return

	}
}

// streamingHandler handles Server-Sent Events streaming requests.
func (sm *ServiceManager) streamingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if sm.streamingProcessor == nil {
		http.Error(w, "Streaming not enabled", http.StatusServiceUnavailable)
		return
	}

	var req llm.StreamingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sm.logger.Error("Failed to decode streaming request", slog.String("error", err.Error()))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	// Set defaults.
	if req.ModelName == "" {
		req.ModelName = sm.config.LLMModelName
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = sm.config.LLMMaxTokens
	}

	sm.logger.Info("Starting streaming request",
		slog.String("query", req.Query),
		slog.String("model", req.ModelName),
		slog.Bool("enable_rag", req.EnableRAG),
	)

	err := sm.streamingProcessor.HandleStreamingRequest(w, r, &req)
	if err != nil {
		sm.logger.Error("Streaming request failed", slog.String("error", err.Error()))
		// Error handling is done within HandleStreamingRequest.
	}
}

// circuitBreakerStatusHandler provides circuit breaker status and controls.
func (sm *ServiceManager) circuitBreakerStatusHandler(w http.ResponseWriter, r *http.Request) {
	if sm.circuitBreakerMgr == nil {
		http.Error(w, "Circuit breaker manager not available", http.StatusServiceUnavailable)
		return
	}

	// Handle POST requests for circuit breaker operations.
	if r.Method == http.MethodPost {
		var req struct {
			Action string `json:"action"`
			Name   string `json:"name"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		cb, exists := sm.circuitBreakerMgr.Get(req.Name)
		if !exists {
			http.Error(w, "Circuit breaker not found", http.StatusNotFound)
			return
		}

		switch req.Action {
		case "reset":
			cb.Reset()
			sm.logger.Info("Circuit breaker reset", slog.String("name", req.Name))
		case "force_open":
			cb.ForceOpen()
			sm.logger.Info("Circuit breaker forced open", slog.String("name", req.Name))
		default:
			http.Error(w, "Invalid action", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		// FIXME: Adding error check for json encoder per errcheck linter.

		if err := json.NewEncoder(w).Encode(map[string]string{"status": "success"}); err != nil {

			sm.logger.Error("Error encoding JSON", slog.String("error", err.Error()))

			return

		}
		return
	}

	// Handle GET requests for status.
	stats := sm.circuitBreakerMgr.GetAllStats()
	w.Header().Set("Content-Type", "application/json")
	// FIXME: Adding error check for json encoder per errcheck linter.

	if err := json.NewEncoder(w).Encode(stats); err != nil {

		sm.logger.Error("Error encoding JSON", slog.String("error", err.Error()))

		return

	}
}
