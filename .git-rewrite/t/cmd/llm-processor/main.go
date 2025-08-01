package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
)

var (
	config    *Config
	processor *IntentProcessor
	logger    *slog.Logger
	startTime = time.Now()
	healthMux sync.RWMutex
	healthy   = true
	readyMux  sync.RWMutex
	ready     = false
	requestID int64
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

// IntentProcessor handles the LLM processing logic
type IntentProcessor struct {
	llmClient *llm.Client
	logger    *slog.Logger
}

func main() {
	// Initialize logger
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Load configuration
	config = loadConfig()

	// Initialize LLM client
	llmClient := llm.NewClientWithConfig(config.RAGAPIURL, llm.ClientConfig{
		APIKey:      config.LLMAPIKey,
		ModelName:   config.LLMModelName,
		MaxTokens:   config.LLMMaxTokens,
		BackendType: config.LLMBackendType,
		Timeout:     config.LLMTimeout,
	})

	// Initialize processor
	processor = &IntentProcessor{
		llmClient: llmClient,
		logger:    logger,
	}

	logger.Info("Starting LLM Processor service",
		slog.String("version", config.ServiceVersion),
		slog.String("port", config.Port),
		slog.String("backend_type", config.LLMBackendType),
		slog.String("model", config.LLMModelName),
	)

	// Set up HTTP server
	mux := http.NewServeMux()

	// Health endpoints
	mux.HandleFunc("/healthz", healthzHandler)
	mux.HandleFunc("/readyz", readyzHandler)

	// Main processing endpoint
	mux.HandleFunc("/process", processIntentHandler)

	// Status endpoint
	mux.HandleFunc("/status", statusHandler)

	server := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      mux,
		ReadTimeout:  config.RequestTimeout,
		WriteTimeout: config.RequestTimeout,
		IdleTimeout:  2 * time.Minute,
	}

	// Mark service as ready
	markReady(true)

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
	markReady(false)

	// Give the server a timeout to finish handling requests
	ctx, cancel := context.WithTimeout(context.Background(), config.GracefulShutdown)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", slog.String("error", err.Error()))
	}

	logger.Info("Server exited")
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
	}
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	healthMux.RLock()
	isHealthy := healthy
	healthMux.RUnlock()

	response := HealthResponse{
		Status:    "healthy",
		Version:   config.ServiceVersion,
		Uptime:    time.Since(startTime).String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if !isHealthy {
		response.Status = "unhealthy"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func readyzHandler(w http.ResponseWriter, r *http.Request) {
	readyMux.RLock()
	isReady := ready
	readyMux.RUnlock()

	response := HealthResponse{
		Status:    "ready",
		Version:   config.ServiceVersion,
		Uptime:    time.Since(startTime).String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if !isReady {
		response.Status = "not ready"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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
		"healthy":         getHealthStatus(),
		"ready":           getReadyStatus(),
		"backend_type":    config.LLMBackendType,
		"model":           config.LLMModelName,
		"rag_enabled":     config.RAGEnabled,
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (p *IntentProcessor) ProcessIntent(ctx context.Context, intent string) (string, error) {
	p.logger.Debug("Processing intent with LLM client", slog.String("intent", intent))

	result, err := p.llmClient.ProcessIntent(ctx, intent)
	if err != nil {
		return "", fmt.Errorf("LLM processing failed: %w", err)
	}

	return result, nil
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

func markReady(isReady bool) {
	readyMux.Lock()
	ready = isReady
	readyMux.Unlock()
}

func getHealthStatus() bool {
	healthMux.RLock()
	defer healthMux.RUnlock()
	return healthy
}

func getReadyStatus() bool {
	readyMux.RLock()
	defer readyMux.RUnlock()
	return ready
}