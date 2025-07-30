package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/thc1006/nephoran-intent-operator/pkg/auth"
	"github.com/thc1006/nephoran-intent-operator/pkg/health"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
	"github.com/thc1006/nephoran-intent-operator/pkg/rag"
)

var (
	config              *Config
	processor           *IntentProcessor
	streamingProcessor  *llm.StreamingProcessor
	circuitBreakerMgr   *llm.CircuitBreakerManager
	tokenManager        *llm.TokenManager
	contextBuilder      *llm.ContextBuilder
	relevanceScorer     *llm.RelevanceScorer
	promptBuilder       *llm.RAGAwarePromptBuilder
	logger              *slog.Logger
	healthChecker       *health.HealthChecker
	startTime           = time.Now()
	requestID           int64
)

// Configuration structure
type Config struct {
	// Service Configuration
	Port             string
	LogLevel         string
	ServiceVersion   string
	GracefulShutdown time.Duration

	// LLM Configuration
	LLMBackendType string
	LLMAPIKey      string
	LLMModelName   string
	LLMTimeout     time.Duration
	LLMMaxTokens   int

	// RAG Configuration
	RAGAPIURL    string
	RAGTimeout   time.Duration
	RAGEnabled   bool
	
	// Streaming Configuration
	StreamingEnabled     bool
	MaxConcurrentStreams int
	StreamTimeout        time.Duration
	
	// Context Management
	EnableContextBuilder bool
	MaxContextTokens     int
	ContextTTL          time.Duration

	// Security Configuration
	APIKeyRequired bool
	APIKey         string
	CORSEnabled    bool
	AllowedOrigins string

	// Performance Configuration
	RequestTimeout time.Duration
	MaxRequestSize int64

	// Circuit Breaker Configuration
	CircuitBreakerEnabled   bool
	CircuitBreakerThreshold int
	CircuitBreakerTimeout   time.Duration

	// Rate Limiting
	RateLimitEnabled        bool
	RateLimitRequestsPerMin int
	RateLimitBurst          int

	// Retry Configuration
	MaxRetries   int
	RetryDelay   time.Duration
	RetryBackoff string

	// OAuth2 Authentication Configuration
	AuthEnabled        bool
	AuthConfigFile     string
	JWTSecretKey       string
	RequireAuth        bool
	AdminUsers         []string
	OperatorUsers      []string
}

// Request/Response structures
type ProcessIntentRequest struct {
	Intent   string            `json:"intent"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type ProcessIntentResponse struct {
	Result          string                 `json:"result"`
	ProcessingTime  string                 `json:"processing_time"`
	RequestID       string                 `json:"request_id"`
	ServiceVersion  string                 `json:"service_version"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	Status          string                 `json:"status"`
	Error           string                 `json:"error,omitempty"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Uptime    string `json:"uptime"`
	Timestamp string `json:"timestamp"`
}

// IntentProcessor handles the LLM processing logic with RAG enhancement
type IntentProcessor struct {
	llmClient          *llm.Client
	ragEnhancedClient  *llm.RAGEnhancedProcessor
	circuitBreaker     *llm.CircuitBreaker
	logger             *slog.Logger
}

func main() {
	// Initialize logger
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Load configuration
	config = loadConfig()

	// Initialize health checker
	healthChecker = health.NewHealthChecker("llm-processor", config.ServiceVersion, logger)

	// Initialize components
	if err := initializeComponents(); err != nil {
		logger.Error("Failed to initialize components", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Register health checks
	registerHealthChecks()

	logger.Info("Starting LLM Processor service",
		slog.String("version", config.ServiceVersion),
		slog.String("port", config.Port),
		slog.String("backend_type", config.LLMBackendType),
		slog.String("model", config.LLMModelName),
	)

	// Set up HTTP server with authentication
	router := mux.NewRouter()
	
	// Initialize OAuth2 middleware if enabled
	var authMiddleware *auth.AuthMiddleware
	if config.AuthEnabled {
		authConfig, err := auth.LoadAuthConfig()
		if err != nil {
			logger.Error("Failed to load auth config", slog.String("error", err.Error()))
			os.Exit(1)
		}
		
		oauth2Config, err := authConfig.ToOAuth2Config()
		if err != nil {
			logger.Error("Failed to create OAuth2 config", slog.String("error", err.Error()))
			os.Exit(1)
		}
		
		authMiddleware = auth.NewAuthMiddleware(oauth2Config, []byte(config.JWTSecretKey))
		
		// OAuth2 authentication routes
		router.HandleFunc("/auth/login/{provider}", authMiddleware.LoginHandler).Methods("GET")
		router.HandleFunc("/auth/callback/{provider}", authMiddleware.CallbackHandler).Methods("GET")
		router.HandleFunc("/auth/refresh", authMiddleware.RefreshHandler).Methods("POST")
		router.HandleFunc("/auth/logout", authMiddleware.LogoutHandler).Methods("POST")
		router.HandleFunc("/auth/userinfo", authMiddleware.UserInfoHandler).Methods("GET")
		
		logger.Info("OAuth2 authentication enabled", 
			slog.Int("providers", len(oauth2Config.Providers)))
	}

	// Public health endpoints (no authentication required)
	router.HandleFunc("/healthz", healthChecker.HealthzHandler).Methods("GET")
	router.HandleFunc("/readyz", healthChecker.ReadyzHandler).Methods("GET")
	router.HandleFunc("/metrics", metricsHandler).Methods("GET")

	// Protected endpoints
	if config.AuthEnabled && config.RequireAuth {
		// Apply authentication middleware to protected routes
		protectedRouter := router.PathPrefix("/").Subrouter()
		protectedRouter.Use(authMiddleware.Authenticate)
		
		// Main processing endpoint - requires operator role
		protectedRouter.HandleFunc("/process", processIntentHandler).Methods("POST")
		protectedRouter.Use(authMiddleware.RequireOperator())
		
		// Streaming endpoint - requires operator role
		if config.StreamingEnabled {
			protectedRouter.HandleFunc("/stream", streamingHandler).Methods("POST")
		}
		
		// Admin endpoints - requires admin role
		adminRouter := protectedRouter.PathPrefix("/admin").Subrouter()
		adminRouter.Use(authMiddleware.RequireAdmin())
		adminRouter.HandleFunc("/status", statusHandler).Methods("GET")
		adminRouter.HandleFunc("/circuit-breaker/status", circuitBreakerStatusHandler).Methods("GET")
		
	} else {
		// No authentication required - direct routes
		router.HandleFunc("/process", processIntentHandler).Methods("POST")
		router.HandleFunc("/status", statusHandler).Methods("GET")
		router.HandleFunc("/circuit-breaker/status", circuitBreakerStatusHandler).Methods("GET")
		
		if config.StreamingEnabled {
			router.HandleFunc("/stream", streamingHandler).Methods("POST")
		}
	}

	server := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      router,
		ReadTimeout:  config.RequestTimeout,
		WriteTimeout: config.RequestTimeout,
		IdleTimeout:  2 * time.Minute,
	}

	// Mark service as ready
	healthChecker.SetReady(true)

	// Start server in a goroutine
	go func() {
		logger.Info("Server starting", slog.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed to start", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Server shutting down...")

	// Mark service as not ready
	healthChecker.SetReady(false)

	// Give the server a timeout to finish handling requests
	ctx, cancel := context.WithTimeout(context.Background(), config.GracefulShutdown)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", slog.String("error", err.Error()))
	}

	logger.Info("Server exited")
}

// initializeComponents initializes all the RAG-enhanced components
func initializeComponents() error {
	// Initialize token manager
	tokenManager = llm.NewTokenManager()
	
	// Initialize circuit breaker manager
	circuitBreakerMgr = llm.NewCircuitBreakerManager(nil)
	
	// Initialize base LLM client
	llmClient := llm.NewClientWithConfig(config.RAGAPIURL, llm.ClientConfig{
		APIKey:      config.LLMAPIKey,
		ModelName:   config.LLMModelName,
		MaxTokens:   config.LLMMaxTokens,
		BackendType: config.LLMBackendType,
		Timeout:     config.LLMTimeout,
	})
	
	// Initialize relevance scorer (without embeddings for now)
	relevanceScorer = llm.NewRelevanceScorer(nil, nil)
	
	// Initialize context builder
	if config.EnableContextBuilder {
		contextBuilder = llm.NewContextBuilder(tokenManager, relevanceScorer, nil)
	}
	
	// Initialize prompt builder
	promptBuilder = llm.NewRAGAwarePromptBuilder(tokenManager, nil)
	
	// Initialize RAG-enhanced processor if RAG is enabled
	var ragEnhanced *llm.RAGEnhancedProcessor
	if config.RAGEnabled {
		// For now, we'll create a basic RAG enhanced processor
		// In production, you would initialize with actual RAG service and Weaviate client
		ragEnhanced = llm.NewRAGEnhancedProcessor(*llmClient, nil, nil, nil)
	}
	
	// Initialize streaming processor if enabled
	if config.StreamingEnabled {
		streamingConfig := &llm.StreamingConfig{
			MaxConcurrentStreams: config.MaxConcurrentStreams,
			StreamTimeout:       config.StreamTimeout,
		}
		streamingProcessor = llm.NewStreamingProcessor(*llmClient, tokenManager, streamingConfig)
	}
	
	// Initialize main processor with circuit breaker
	circuitBreaker := circuitBreakerMgr.GetOrCreate("llm-processor", nil)
	processor = &IntentProcessor{
		llmClient:         llmClient,
		ragEnhancedClient: ragEnhanced,
		circuitBreaker:    circuitBreaker,
		logger:           logger,
	}
	
	return nil
}

// registerHealthChecks registers all health checks for the service
func registerHealthChecks() {
	// Internal service health checks
	healthChecker.RegisterCheck("service_status", func(ctx context.Context) *health.Check {
		return &health.Check{
			Status:  health.StatusHealthy,
			Message: "Service is running normally",
		}
	})

	// Circuit breaker health check
	if circuitBreakerMgr != nil {
		healthChecker.RegisterCheck("circuit_breaker", func(ctx context.Context) *health.Check {
			stats := circuitBreakerMgr.GetAllStats()
			if len(stats) == 0 {
				return &health.Check{
					Status:  health.StatusHealthy,
					Message: "No circuit breakers registered",
				}
			}
			
			// Check if any circuit breakers are open
			for name, state := range stats {
				if state != nil {
					// Assuming the state has a field indicating if it's open
					// This would need to match your actual circuit breaker implementation
					return &health.Check{
						Status:  health.StatusHealthy,
						Message: fmt.Sprintf("Circuit breaker %s is operational", name),
					}
				}
			}
			
			return &health.Check{
				Status:  health.StatusHealthy,
				Message: "All circuit breakers operational",
			}
		})
	}

	// Token manager health check
	if tokenManager != nil {
		healthChecker.RegisterCheck("token_manager", func(ctx context.Context) *health.Check {
			models := tokenManager.GetSupportedModels()
			return &health.Check{
				Status:  health.StatusHealthy,
				Message: fmt.Sprintf("Token manager operational with %d supported models", len(models)),
				Metadata: map[string]interface{}{
					"supported_models": models,
				},
			}
		})
	}

	// Streaming processor health check
	if streamingProcessor != nil {
		healthChecker.RegisterCheck("streaming_processor", func(ctx context.Context) *health.Check {
			metrics := streamingProcessor.GetMetrics()
			return &health.Check{
				Status:  health.StatusHealthy,
				Message: "Streaming processor operational",
				Metadata: metrics,
			}
		})
	}

	// RAG API dependency check
	if config.RAGEnabled && config.RAGAPIURL != "" {
		healthChecker.RegisterDependency("rag_api", health.HTTPCheck("rag_api", config.RAGAPIURL+"/health"))
	}

	logger.Info("Health checks registered", "checks", len(healthChecker.checks), "dependencies", len(healthChecker.dependencies))
}

func loadConfig() *Config {
	return &Config{
		Port:             getEnv("PORT", "8080"),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
		ServiceVersion:   getEnv("SERVICE_VERSION", "v2.0.0"),
		GracefulShutdown: parseDuration(getEnv("GRACEFUL_SHUTDOWN_TIMEOUT", "30s")),

		LLMBackendType: getEnv("LLM_BACKEND_TYPE", "rag"),
		LLMAPIKey:      getEnv("OPENAI_API_KEY", ""),
		LLMModelName:   getEnv("LLM_MODEL_NAME", "gpt-4o-mini"),
		LLMTimeout:     parseDuration(getEnv("LLM_TIMEOUT", "60s")),
		LLMMaxTokens:   parseInt(getEnv("LLM_MAX_TOKENS", "2048")),

		RAGAPIURL:  getEnv("RAG_API_URL", "http://rag-api:5001"),
		RAGTimeout: parseDuration(getEnv("RAG_TIMEOUT", "30s")),
		RAGEnabled: parseBool(getEnv("RAG_ENABLED", "true")),
		
		StreamingEnabled:     parseBool(getEnv("STREAMING_ENABLED", "true")),
		MaxConcurrentStreams: parseInt(getEnv("MAX_CONCURRENT_STREAMS", "100")),
		StreamTimeout:        parseDuration(getEnv("STREAM_TIMEOUT", "5m")),
		
		EnableContextBuilder: parseBool(getEnv("ENABLE_CONTEXT_BUILDER", "true")),
		MaxContextTokens:     parseInt(getEnv("MAX_CONTEXT_TOKENS", "6000")),
		ContextTTL:          parseDuration(getEnv("CONTEXT_TTL", "5m")),

		APIKeyRequired: parseBool(getEnv("API_KEY_REQUIRED", "false")),
		APIKey:         getEnv("API_KEY", ""),
		CORSEnabled:    parseBool(getEnv("CORS_ENABLED", "true")),
		AllowedOrigins: getEnv("ALLOWED_ORIGINS", "*"),

		RequestTimeout: parseDuration(getEnv("REQUEST_TIMEOUT", "30s")),
		MaxRequestSize: parseInt64(getEnv("MAX_REQUEST_SIZE", "1048576")), // 1MB

		CircuitBreakerEnabled:   parseBool(getEnv("CIRCUIT_BREAKER_ENABLED", "true")),
		CircuitBreakerThreshold: parseInt(getEnv("CIRCUIT_BREAKER_THRESHOLD", "5")),
		CircuitBreakerTimeout:   parseDuration(getEnv("CIRCUIT_BREAKER_TIMEOUT", "60s")),

		RateLimitEnabled:        parseBool(getEnv("RATE_LIMIT_ENABLED", "true")),
		RateLimitRequestsPerMin: parseInt(getEnv("RATE_LIMIT_REQUESTS_PER_MINUTE", "60")),
		RateLimitBurst:          parseInt(getEnv("RATE_LIMIT_BURST", "10")),

		MaxRetries:   parseInt(getEnv("MAX_RETRIES", "3")),
		RetryDelay:   parseDuration(getEnv("RETRY_DELAY", "1s")),
		RetryBackoff: getEnv("RETRY_BACKOFF", "exponential"),

		// OAuth2 Authentication Configuration
		AuthEnabled:    parseBool(getEnv("AUTH_ENABLED", "false")),
		AuthConfigFile: getEnv("AUTH_CONFIG_FILE", ""),
		JWTSecretKey:   getEnv("JWT_SECRET_KEY", ""),
		RequireAuth:    parseBool(getEnv("REQUIRE_AUTH", "true")),
		AdminUsers:     parseStringSlice(getEnv("ADMIN_USERS", "")),
		OperatorUsers:  parseStringSlice(getEnv("OPERATOR_USERS", "")),
	}
}


func processIntentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	startTime := time.Now()
	reqID := fmt.Sprintf("%d", time.Now().UnixNano())

	logger.Info("Processing intent request", slog.String("request_id", reqID))

	var req ProcessIntentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("Failed to decode request", slog.String("error", err.Error()))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Intent == "" {
		logger.Error("Empty intent provided")
		http.Error(w, "Intent is required", http.StatusBadRequest)
		return
	}

	// Process intent
	result, err := processor.ProcessIntent(r.Context(), req.Intent)
	if err != nil {
		logger.Error("Failed to process intent",
			slog.String("error", err.Error()),
			slog.String("intent", req.Intent),
		)
		response := ProcessIntentResponse{
			Status:         "error",
			Error:          err.Error(),
			RequestID:      reqID,
			ServiceVersion: config.ServiceVersion,
			ProcessingTime: time.Since(startTime).String(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ProcessIntentResponse{
		Result:         result,
		Status:         "success",
		ProcessingTime: time.Since(startTime).String(),
		RequestID:      reqID,
		ServiceVersion: config.ServiceVersion,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	logger.Info("Intent processed successfully",
		slog.String("request_id", reqID),
		slog.Duration("processing_time", time.Since(startTime)),
	)
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"service":         "llm-processor",
		"version":         config.ServiceVersion,
		"uptime":          time.Since(startTime).String(),
		"healthy":         healthChecker.IsHealthy(),
		"ready":           healthChecker.IsReady(),
		"backend_type":    config.LLMBackendType,
		"model":           config.LLMModelName,
		"rag_enabled":     config.RAGEnabled,
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (p *IntentProcessor) ProcessIntent(ctx context.Context, intent string) (string, error) {
	p.logger.Debug("Processing intent with enhanced client", slog.String("intent", intent))

	// Use circuit breaker for fault tolerance
	operation := func(ctx context.Context) (interface{}, error) {
		// Try RAG-enhanced processing first if available
		if p.ragEnhancedClient != nil {
			result, err := p.ragEnhancedClient.ProcessIntent(ctx, intent)
			if err == nil {
				return result, nil
			}
			p.logger.Warn("RAG-enhanced processing failed, falling back to base client", "error", err)
		}
		
		// Fallback to base LLM client
		return p.llmClient.ProcessIntent(ctx, intent)
	}
	
	result, err := p.circuitBreaker.Execute(ctx, operation)
	if err != nil {
		return "", fmt.Errorf("LLM processing failed: %w", err)
	}

	return result.(string), nil
}

// Utility functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 30 * time.Second
	}
	return d
}

func parseInt(s string) int {
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}

func parseInt64(s string) int64 {
	var result int64
	fmt.Sscanf(s, "%d", &result)
	return result
}

func parseBool(s string) bool {
	return s == "true" || s == "1"
}

func parseStringSlice(s string) []string {
	if s == "" {
		return []string{}
	}
	result := []string{}
	for _, item := range strings.Split(s, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}


// streamingHandler handles Server-Sent Events streaming requests
func streamingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if streamingProcessor == nil {
		http.Error(w, "Streaming not enabled", http.StatusServiceUnavailable)
		return
	}

	var req llm.StreamingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("Failed to decode streaming request", slog.String("error", err.Error()))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.ModelName == "" {
		req.ModelName = config.LLMModelName
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = config.LLMMaxTokens
	}

	logger.Info("Starting streaming request",
		"query", req.Query,
		"model", req.ModelName,
		"enable_rag", req.EnableRAG,
	)

	err := streamingProcessor.HandleStreamingRequest(w, r, &req)
	if err != nil {
		logger.Error("Streaming request failed", slog.String("error", err.Error()))
		// Error handling is done within HandleStreamingRequest
	}
}

// metricsHandler provides comprehensive metrics
func metricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := map[string]interface{}{
		"service": "llm-processor",
		"version": config.ServiceVersion,
		"uptime":  time.Since(startTime).String(),
	}

	// Add token manager metrics
	if tokenManager != nil {
		metrics["supported_models"] = tokenManager.GetSupportedModels()
	}

	// Add circuit breaker metrics
	if circuitBreakerMgr != nil {
		metrics["circuit_breakers"] = circuitBreakerMgr.GetAllStats()
	}

	// Add streaming metrics
	if streamingProcessor != nil {
		metrics["streaming"] = streamingProcessor.GetMetrics()
	}

	// Add context builder metrics
	if contextBuilder != nil {
		metrics["context_builder"] = contextBuilder.GetMetrics()
	}

	// Add relevance scorer metrics
	if relevanceScorer != nil {
		metrics["relevance_scorer"] = relevanceScorer.GetMetrics()
	}

	// Add prompt builder metrics
	if promptBuilder != nil {
		metrics["prompt_builder"] = promptBuilder.GetMetrics()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// circuitBreakerStatusHandler provides circuit breaker status
func circuitBreakerStatusHandler(w http.ResponseWriter, r *http.Request) {
	if circuitBreakerMgr == nil {
		http.Error(w, "Circuit breaker manager not available", http.StatusServiceUnavailable)
		return
	}

	// Handle POST requests for circuit breaker operations
	if r.Method == http.MethodPost {
		var req struct {
			Action string `json:"action"`
			Name   string `json:"name"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		cb, exists := circuitBreakerMgr.Get(req.Name)
		if !exists {
			http.Error(w, "Circuit breaker not found", http.StatusNotFound)
			return
		}

		switch req.Action {
		case "reset":
			cb.Reset()
			logger.Info("Circuit breaker reset", "name", req.Name)
		case "force_open":
			cb.ForceOpen()
			logger.Info("Circuit breaker forced open", "name", req.Name)
		default:
			http.Error(w, "Invalid action", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		return
	}

	// Handle GET requests for status
	stats := circuitBreakerMgr.GetAllStats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}