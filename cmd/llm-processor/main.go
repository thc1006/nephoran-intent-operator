package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"

	"github.com/thc1006/nephoran-intent-operator/internal/llm/providers"
	"github.com/thc1006/nephoran-intent-operator/pkg/auth"
	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/handlers"
	"github.com/thc1006/nephoran-intent-operator/pkg/middleware"
	"github.com/thc1006/nephoran-intent-operator/pkg/services"
)

// ProviderAdapter adapts the new Provider interface to work with the existing LLMClient interface
type ProviderAdapter struct {
	provider providers.Provider
	logger   *slog.Logger
}

// NewProviderAdapter creates a new adapter for the provider
func NewProviderAdapter(provider providers.Provider, logger *slog.Logger) *ProviderAdapter {
	return &ProviderAdapter{
		provider: provider,
		logger:   logger,
	}
}

// ProcessIntent adapts the provider interface to the expected LLMClient interface
func (a *ProviderAdapter) ProcessIntent(ctx context.Context, intent string) (string, error) {
	response, err := a.provider.ProcessIntent(ctx, intent)
	if err != nil {
		return "", err
	}

	// Return the structured JSON from the provider
	return string(response.JSON), nil
}

// createProviderIntentHandler creates a new HTTP handler that uses the provider directly
func createProviderIntentHandler(provider providers.Provider, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Method not allowed",
				"message": "Only POST requests are supported",
			})
			return
		}

		// Read request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("Failed to read request body", slog.String("error", err.Error()))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Invalid request body",
				"message": err.Error(),
			})
			return
		}

		// Parse request
		var req ProcessIntentRequest
		if err := json.Unmarshal(body, &req); err != nil {
			logger.Error("Failed to parse request JSON", slog.String("error", err.Error()))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Invalid JSON",
				"message": err.Error(),
			})
			return
		}

		// Validate input
		if req.Intent == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Validation failed",
				"message": "Intent field is required",
			})
			return
		}

		// Process intent using provider
		ctx := r.Context()
		startTime := time.Now()
		response, err := provider.ProcessIntent(ctx, req.Intent)
		processingTime := time.Since(startTime)

		if err != nil {
			logger.Error("Provider processing failed", 
				slog.String("error", err.Error()),
				slog.String("intent", req.Intent[:min(100, len(req.Intent))]))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Processing failed",
				"message": err.Error(),
			})
			return
		}

		// Create success response
		successResponse := map[string]interface{}{
			"status": "success",
			"result": json.RawMessage(response.JSON),
			"processing_time": processingTime.String(),
			"provider_metadata": map[string]interface{}{
				"provider": response.Metadata.Provider,
				"model": response.Metadata.Model,
				"tokens_used": response.Metadata.TokensUsed,
				"confidence": response.Metadata.Confidence,
				"warnings": response.Metadata.Warnings,
			},
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}

		// Return response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(successResponse)

		logger.Info("Provider intent processed successfully",
			slog.String("provider", response.Metadata.Provider),
			slog.String("model", response.Metadata.Model),
			slog.Duration("processing_time", processingTime),
			slog.String("intent_preview", req.Intent[:min(50, len(req.Intent))]))
	}
}

// Config is a simplified configuration struct for testing compatibility.
// Production code uses config.LLMProcessorConfig.
type Config struct {
	Port             string
	LogLevel         string
	ServiceVersion   string
	GracefulShutdown time.Duration

	LLMBackendType string
	LLMAPIKey      string
	LLMModelName   string
	LLMTimeout     time.Duration
	LLMMaxTokens   int

	OpenAIAPIURL string
	OpenAIAPIKey string

	RAGEnabled bool

	CircuitBreakerEnabled   bool
	CircuitBreakerThreshold int
	CircuitBreakerTimeout   time.Duration

	MaxRetries   int
	RetryDelay   time.Duration
	RetryBackoff string

	MetricsEnabled bool
}

// IntentProcessor is a simplified processor for testing compatibility.
// Production code uses handlers.IntentProcessor.
type IntentProcessor struct {
	Config   *Config
	Provider providers.Provider
}

// NewIntentProcessor creates a new IntentProcessor with provider.
func NewIntentProcessor(config *Config, provider providers.Provider) *IntentProcessor {
	return &IntentProcessor{
		Config:   config,
		Provider: provider,
	}
}

// ProcessIntent processes an intent string and returns the result using the provider interface.
func (p *IntentProcessor) ProcessIntent(ctx context.Context, intent string) (string, error) {
	if intent == "" {
		return "", fmt.Errorf("validation failed: intent cannot be empty")
	}
	if len(intent) > 2000 {
		return "", fmt.Errorf("intent too long: maximum 2000 characters")
	}

	if p.Provider == nil {
		return "", fmt.Errorf("provider not configured")
	}

	// Use the new provider interface to process intent
	intentResponse, err := p.Provider.ProcessIntent(ctx, intent)
	if err != nil {
		return "", fmt.Errorf("provider processing failed: %w", err)
	}

	// Create response with provider metadata and structured intent
	response := map[string]interface{}{
		"type":             "NetworkIntent",
		"original_intent":  intent,
		"intent_json":      intentResponse.JSON,
		"processing_metadata": map[string]interface{}{
			"provider":         intentResponse.Metadata.Provider,
			"model":            intentResponse.Metadata.Model,
			"processing_time":  intentResponse.Metadata.ProcessingTime.String(),
			"tokens_used":      intentResponse.Metadata.TokensUsed,
			"confidence":       intentResponse.Metadata.Confidence,
			"warnings":         intentResponse.Metadata.Warnings,
		},
	}

	// Convert to JSON string
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	return string(jsonBytes), nil
}

// createHandoffIntentHandler creates an HTTP handler that accepts plain text intent,
// processes it with the LLM provider, and writes the result to handoff directory
func createHandoffIntentHandler(provider providers.Provider, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Type", "application/json")

		// Read request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("Failed to read request body", slog.String("error", err.Error()))
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		intentText := strings.TrimSpace(string(body))
		if intentText == "" {
			http.Error(w, "Intent text cannot be empty", http.StatusBadRequest)
			return
		}

		logger.Info("Processing handoff intent", slog.String("intent", intentText))

		// Process with LLM provider
		ctx := r.Context()
		response, err := provider.ProcessIntent(ctx, intentText)
		if err != nil {
			logger.Error("Provider processing failed", slog.String("error", err.Error()))
			http.Error(w, "Intent processing failed", http.StatusInternalServerError)
			return
		}

		// Create handoff directory if it doesn't exist
		handoffDir := "./handoff"
		if err := os.MkdirAll(handoffDir, 0755); err != nil {
			logger.Error("Failed to create handoff directory", slog.String("error", err.Error()))
			http.Error(w, "Failed to create handoff directory", http.StatusInternalServerError)
			return
		}

		// Create handoff filename with timestamp
		timestamp := time.Now().Format("20060102-150405")
		filename := fmt.Sprintf("intent-%s.json", timestamp)
		filepath := fmt.Sprintf("%s/%s", handoffDir, filename)

		// Write JSON to handoff file
		if err := os.WriteFile(filepath, []byte(response.JSON), 0644); err != nil {
			logger.Error("Failed to write handoff file", 
				slog.String("file", filepath),
				slog.String("error", err.Error()))
			http.Error(w, "Failed to write handoff file", http.StatusInternalServerError)
			return
		}

		logger.Info("Handoff file created",
			slog.String("file", filepath),
			slog.String("provider", response.Metadata.Provider))

		// Return success response
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"handoff_file": filename,
			"provider": response.Metadata.Provider,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}
}

// NetworkIntentRequest represents a request for processing network intents.
// Used for test compatibility with the intent processing pipeline.
type NetworkIntentRequest struct {
	Spec struct {
		Intent string `json:"intent"`
	} `json:"spec"`
}

type NetworkIntentResponse struct {
	Type               string          `json:"type"`
	Name               string          `json:"name"`
	Namespace          string          `json:"namespace"`
	OriginalIntent     string          `json:"original_intent"`
	Spec               json.RawMessage `json:"spec"`
	ProcessingMetadata struct {
		ModelUsed       string  `json:"modelUsed"`
		ConfidenceScore float64 `json:"confidenceScore"`
	} `json:"processing_metadata"`
}

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Time    string `json:"time"`
}

type ReadinessResponse struct {
	Status       string            `json:"status"`
	Dependencies map[string]string `json:"dependencies"`
}

type ErrorResponse struct {
	ErrorCode string `json:"errorCode"`
	Message   string `json:"message"`
}

// ProcessIntentRequest represents a request for processing intents using the provider
type ProcessIntentRequest struct {
	Intent   string            `json:"intent"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

var (
	cfg *config.LLMProcessorConfig

	logger *slog.Logger

	service *services.LLMProcessorService

	handler *handlers.LLMProcessorHandler

	startTime = time.Now()

	postRateLimiter *middleware.PostOnlyRateLimiter // Declare at package level for shutdown

)

func main() {
	// Initialize structured logger.

	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Load configuration with validation.

	var err error

	cfg, err = config.LoadLLMProcessorConfig()
	if err != nil {

		logger.Error("Failed to load configuration", slog.String("error", err.Error()))

		log.Fatal(1)

	}

	// Update logger level based on configuration.

	logger = createLoggerWithLevel(cfg.LogLevel)

	// Perform security validation before starting the service.

	if err := validateSecurityConfiguration(cfg, logger); err != nil {

		logger.Error("Security validation failed", slog.String("error", err.Error()))

		log.Fatal(1)

	}

	logger.Info("Starting LLM Processor service",

		slog.String("version", cfg.ServiceVersion),

		slog.String("port", cfg.Port),

		slog.String("backend_type", cfg.LLMBackendType),

		slog.String("model", cfg.LLMModelName),

		slog.Bool("auth_enabled", cfg.AuthEnabled),

		slog.Bool("require_auth", cfg.RequireAuth),

		slog.Bool("tls_enabled", cfg.TLSEnabled),

		slog.Bool("rate_limit_enabled", cfg.RateLimitEnabled),

		slog.Int("rate_limit_qps", cfg.RateLimitQPS),

		slog.Int("rate_limit_burst", cfg.RateLimitBurstTokens),
	)

	// Initialize LLM provider from environment.
	provider, err := providers.CreateFromEnvironment()
	if err != nil {
		logger.Error("Failed to create LLM provider", slog.String("error", err.Error()))
		log.Fatal(1)
	}
	defer func() {
		if closeErr := provider.Close(); closeErr != nil {
			logger.Warn("Failed to close provider", slog.String("error", closeErr.Error()))
		}
	}()

	// Log provider information.
	providerInfo := provider.GetProviderInfo()
	logger.Info("LLM Provider initialized",
		slog.String("provider", providerInfo.Name),
		slog.String("version", providerInfo.Version),
		slog.String("description", providerInfo.Description),
		slog.Bool("requires_auth", providerInfo.RequiresAuth),
		slog.Any("supported_features", providerInfo.SupportedFeatures),
	)

	// Initialize service components.

	service = services.NewLLMProcessorService(cfg, logger)

	ctx := context.Background()

	if err := service.Initialize(ctx); err != nil {

		logger.Error("Failed to initialize service", slog.String("error", err.Error()))

		log.Fatal(1)

	}

	// Get initialized components.

	processor, _, circuitBreakerMgr, tokenManager, contextBuilder, relevanceScorer, promptBuilder, healthChecker := service.GetComponents()

	// Create handler with initialized components.

	handler = handlers.NewLLMProcessorHandler(

		cfg, processor, nil, circuitBreakerMgr,

		tokenManager, contextBuilder, relevanceScorer, promptBuilder,

		logger, healthChecker, startTime,
	)

	// Set up HTTP server.

	server := setupHTTPServer(provider, providerInfo)

	// Mark service as ready.

	healthChecker.SetReady(true)

	// Start server.

	go func() {
		if cfg.TLSEnabled {

			logger.Info("Server starting with TLS",

				slog.String("addr", server.Addr),

				slog.String("cert_path", cfg.TLSCertPath),

				slog.String("key_path", cfg.TLSKeyPath))

			if err := server.ListenAndServeTLS(cfg.TLSCertPath, cfg.TLSKeyPath); err != nil && err != http.ErrServerClosed {

				logger.Error("TLS server failed to start", slog.String("error", err.Error()))

				log.Fatal(1)

			}

		} else {

			logger.Info("Server starting (HTTP only)",

				slog.String("addr", server.Addr))

			logger.Warn("Server running in HTTP mode - consider enabling TLS for production use",

				slog.String("tls_config", "set TLS_ENABLED=true, TLS_CERT_PATH, and TLS_KEY_PATH to enable TLS"))

			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {

				logger.Error("Server failed to start", slog.String("error", err.Error()))

				log.Fatal(1)

			}

		}
	}()

	// Wait for shutdown signal.

	quit := make(chan os.Signal, 1)

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit

	logger.Info("Server shutting down...")

	// Graceful shutdown.

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.GracefulShutdown)

	defer cancel()

	// Shutdown rate limiter.

	if postRateLimiter != nil {

		logger.Info("Stopping rate limiter...")

		postRateLimiter.Stop()

	}

	// Shutdown service components.

	if err := service.Shutdown(shutdownCtx); err != nil {
		logger.Error("Service shutdown failed", slog.String("error", err.Error()))
	}

	// Shutdown HTTP server.

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", slog.String("error", err.Error()))
	}

	logger.Info("Server exited")
}

// validateSecurityConfiguration performs fail-fast security validation.

// to prevent accidental deployment of unauthenticated services in production.

func validateSecurityConfiguration(cfg *config.LLMProcessorConfig, logger *slog.Logger) error {
	// Check if we're in a development environment.

	isDevelopment := isDevelopmentEnvironment()

	logger.Info("Security configuration validation",

		slog.Bool("is_development", isDevelopment),

		slog.Bool("auth_enabled", cfg.AuthEnabled),

		slog.Bool("require_auth", cfg.RequireAuth),

		slog.Bool("tls_enabled", cfg.TLSEnabled),
	)

	// If authentication is disabled, only allow in development environments.

	if !cfg.AuthEnabled {

		if !isDevelopment {
			return fmt.Errorf("authentication is disabled but this appears to be a production environment. " +

				"Set AUTH_ENABLED=true or ensure GO_ENV/NODE_ENV/ENVIRONMENT is set to 'development' or 'dev'")
		}

		logger.Warn("Authentication is disabled - this should only be used in development environments",

			slog.String("environment_status", "development_detected"))

	}

	// Additional security check: if auth is enabled but not required, warn in production.

	if cfg.AuthEnabled && !cfg.RequireAuth && !isDevelopment {
		logger.Warn("Authentication is enabled but not required in production environment. " +

			"Consider setting REQUIRE_AUTH=true for enhanced security")
	}

	// TLS security validation.

	if !cfg.TLSEnabled && !isDevelopment {
		logger.Warn("TLS is disabled in production environment. " +

			"Consider enabling TLS by setting TLS_ENABLED=true and providing certificate files for enhanced security")
	}

	// Log security status for audit purposes.

	logger.Info("Security status summary",

		slog.Bool("auth_enabled", cfg.AuthEnabled),

		slog.Bool("require_auth", cfg.RequireAuth),

		slog.Bool("tls_enabled", cfg.TLSEnabled),

		slog.String("environment", func() string {
			if isDevelopment {
				return "development"
			}

			return "production"
		}()),

		slog.String("jwt_secret_configured", func() string {
			if cfg.AuthEnabled && cfg.JWTSecretKey != "" {
				return "yes"
			}

			return "no"
		}()),

		slog.String("tls_certificates_configured", func() string {
			if cfg.TLSEnabled && cfg.TLSCertPath != "" && cfg.TLSKeyPath != "" {
				return "yes"
			}

			return "no"
		}()),
	)

	if !cfg.AuthEnabled && !isDevelopment {
		logger.Warn("Service starting without authentication in production environment")
	}

	if !cfg.TLSEnabled && !isDevelopment {
		logger.Warn("Service starting without TLS encryption in production environment")
	}

	return nil
}

// isDevelopmentEnvironment determines if the service is running in a development environment.

// by checking common environment variables used to indicate development/staging environments.

func isDevelopmentEnvironment() bool {
	// Check common environment indicators.

	envVars := []string{"GO_ENV", "NODE_ENV", "ENVIRONMENT", "ENV", "APP_ENV"}

	for _, envVar := range envVars {

		value := strings.ToLower(config.GetEnvOrDefault(envVar, ""))

		switch value {

		case "development", "dev", "local", "test", "testing":

			return true

		case "production", "prod", "staging", "stage":

			return false

		}

	}

	// If no environment variable is set, default to production for safety.

	// This ensures fail-safe behavior where authentication is required by default.

	return false
}

// createLoggerWithLevel creates a logger with the specified level.

func createLoggerWithLevel(level string) *slog.Logger {
	var logLevel slog.Level

	switch level {

	case "debug":

		logLevel = slog.LevelDebug

	case "info":

		logLevel = slog.LevelInfo

	case "warn":

		logLevel = slog.LevelWarn

	case "error":

		logLevel = slog.LevelError

	default:

		logLevel = slog.LevelInfo

	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,

		AddSource: true,
	}))
}

// setupHTTPServer configures and returns the HTTP server.

func setupHTTPServer(provider providers.Provider, providerInfo providers.ProviderInfo) *http.Server {
	router := mux.NewRouter()

	// Initialize request size limiter middleware (uses HTTP_MAX_BODY env var via config).

	requestSizeLimiter := middleware.NewRequestSizeLimiter(cfg.MaxRequestSize, logger)

	logger.Info("Request size limiting enabled",

		slog.Int64("max_request_size_bytes", cfg.MaxRequestSize),

		slog.String("max_request_size_human", fmt.Sprintf("%.2f MB", float64(cfg.MaxRequestSize)/(1024*1024))))

	// Initialize CORS middleware if enabled.

	var corsMiddleware *middleware.CORSMiddleware

	if cfg.CORSEnabled {

		corsConfig := middleware.CORSConfig{
			AllowedOrigins: cfg.AllowedOrigins,

			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},

			AllowedHeaders: []string{"Content-Type", "Authorization", "X-Requested-With", "Accept", "Origin"},

			AllowCredentials: false, // Security: disabled by default

			MaxAge: 24 * time.Hour,
		}

		if err := middleware.ValidateConfig(corsConfig); err != nil {

			logger.Error("Invalid CORS configuration", slog.String("error", err.Error()))

			log.Fatal(1)

		}

		corsMiddleware = middleware.NewCORSMiddleware(corsConfig, logger)

		logger.Info("CORS middleware enabled",

			slog.Any("allowed_origins", cfg.AllowedOrigins),

			slog.Bool("credentials_allowed", corsConfig.AllowCredentials))

	}

	// Initialize rate limiter middleware for POST endpoints only.

	// var postRateLimiter *middleware.PostOnlyRateLimiter // Now declared at package level.

	if cfg.RateLimitEnabled {

		rateLimiterConfig := middleware.RateLimiterConfig{
			QPS: cfg.RateLimitQPS,

			Burst: cfg.RateLimitBurstTokens,

			CleanupInterval: 10 * time.Minute,

			IPTimeout: 1 * time.Hour,
		}

		postRateLimiter = middleware.NewPostOnlyRateLimiter(rateLimiterConfig, logger)

		logger.Info("Rate limiter enabled for POST endpoints",

			slog.Int("qps", cfg.RateLimitQPS),

			slog.Int("burst", cfg.RateLimitBurstTokens))

	}

	// Initialize redact logger middleware.

	redactLoggerConfig := middleware.DefaultRedactLoggerConfig()

	// Customize configuration based on log level and environment.

	redactLoggerConfig.LogLevel = func() slog.Level {
		switch cfg.LogLevel {

		case "debug":

			return slog.LevelDebug

		case "info":

			return slog.LevelInfo

		case "warn":

			return slog.LevelWarn

		case "error":

			return slog.LevelError

		default:

			return slog.LevelInfo

		}
	}()

	redactLoggerConfig.LogRequestBody = cfg.LogLevel == "debug"

	redactLoggerConfig.LogResponseBody = cfg.LogLevel == "debug"

	redactLogger, err := middleware.NewRedactLogger(redactLoggerConfig, logger)
	if err != nil {

		logger.Error("Failed to create redact logger middleware", slog.String("error", err.Error()))

		log.Fatal(1)

	}

	// Initialize security headers middleware with enhanced configuration.

	securityHeadersConfig := middleware.DefaultSecurityHeadersConfig()

	// Enable HSTS only if TLS is enabled.

	securityHeadersConfig.EnableHSTS = cfg.TLSEnabled

	// Enhanced security headers for API service.

	securityHeadersConfig.ContentTypeOptions = true // Always set nosniff

	securityHeadersConfig.FrameOptions = "DENY" // Prevent clickjacking

	securityHeadersConfig.ContentSecurityPolicy = "default-src 'none'" // Strict CSP for API

	securityHeaders := middleware.NewSecurityHeaders(securityHeadersConfig, logger)

	logger.Info("Middlewares configured",

		slog.Bool("redact_logger_enabled", redactLoggerConfig.Enabled),

		slog.Bool("security_headers_enabled", true),

		slog.Bool("hsts_enabled", securityHeadersConfig.EnableHSTS),

		slog.String("log_level", redactLoggerConfig.LogLevel.String()))

	// Initialize Enhanced OAuth2Manager for centralized route configuration.

	enhancedOAuth2Config := &auth.EnhancedOAuth2ManagerConfig{
		// OAuth2 configuration.

		Enabled: cfg.AuthEnabled,

		AuthConfigFile: cfg.AuthConfigFile,

		JWTSecretKey: cfg.JWTSecretKey,

		RequireAuth: cfg.RequireAuth,

		StreamingEnabled: cfg.StreamingEnabled,

		MaxRequestSize: cfg.MaxRequestSize,

		// Public route configuration.

		ExposeMetricsPublicly: cfg.ExposeMetricsPublicly,

		MetricsAllowedCIDRs: cfg.MetricsAllowedCIDRs,

		HealthEndpointsEnabled: true, // Always enable health endpoints for Kubernetes

	}

	oauth2Manager, err := auth.NewEnhancedOAuth2Manager(enhancedOAuth2Config, logger)
	if err != nil {

		logger.Error("Failed to create Enhanced OAuth2Manager", slog.String("error", err.Error()))

		log.Fatal(1)

	}

	// Apply middlewares in the correct order:.

	// 1. Request Size Limiter (first, to prevent oversized bodies early).

	router.Use(requestSizeLimiter.Middleware)

	// 2. Redact Logger (to log all requests).

	router.Use(redactLogger.Middleware)

	// 3. Security Headers (early, to set headers on all responses).

	router.Use(securityHeaders.Middleware)

	// 4. CORS (after security headers).

	if corsMiddleware != nil {
		router.Use(corsMiddleware.Middleware)
	}

	// 5. Rate Limiter (for POST endpoints only, before authentication).

	if postRateLimiter != nil {
		router.Use(postRateLimiter.Middleware)
	}

	// Get health checker component.

	_, _, _, _, _, _, _, healthChecker := service.GetComponents()

	// Prepare public route handlers.

	// Apply metrics access control if metrics endpoint is enabled.

	var metricsHandler http.HandlerFunc

	if cfg.MetricsEnabled {
		if len(cfg.MetricsAllowedIPs) > 0 {
			// Wrap metrics handler with IP access control.

			metricsHandler = createIPRestrictedHandler(handler.MetricsHandler, cfg.MetricsAllowedIPs, logger)
		} else {
			metricsHandler = handler.MetricsHandler
		}
	} else {
		// Metrics disabled - return 404.

		metricsHandler = func(w http.ResponseWriter, r *http.Request) {
			logger.Warn("Metrics endpoint accessed but disabled",

				slog.String("remote_addr", r.RemoteAddr),

				slog.String("path", r.URL.Path))

			http.NotFound(w, r)
		}
	}

	publicHandlers := &auth.PublicRouteHandlers{
		Health: healthChecker.HealthzHandler,

		Ready: healthChecker.ReadyzHandler,

		Metrics: metricsHandler,
	}

	// Create provider-based intent handler
	providerIntentHandler := createProviderIntentHandler(provider, logger)

	// Prepare protected route handlers.

	protectedHandlers := &auth.ProtectedRouteHandlers{
		ProcessIntent: handler.ProcessIntentHandler,

		Status: handler.StatusHandler,

		CircuitBreakerStatus: handler.CircuitBreakerStatusHandler,

		StreamingHandler: handler.StreamingHandler,
	}

	// Configure ALL routes through the centralized OAuth2Manager.

	if err := oauth2Manager.ConfigureAllRoutes(router, publicHandlers, protectedHandlers); err != nil {

		logger.Error("Failed to configure routes", slog.String("error", err.Error()))

		log.Fatal(1)

	}

	// Add provider-based intent processing endpoint
	// This endpoint uses the new provider interface directly
	router.HandleFunc("/api/v2/process-intent", providerIntentHandler).Methods("POST")

	// Add handoff directory endpoint for acceptance testing
	// This endpoint accepts plain text and writes to handoff directory
	handoffHandler := createHandoffIntentHandler(provider, logger)
	router.HandleFunc("/intent", handoffHandler).Methods("POST")

	logger.Info("Provider-based endpoints configured",
		slog.String("endpoint1", "/api/v2/process-intent"),
		slog.String("endpoint2", "/intent"),
		slog.String("method", "POST"),
		slog.String("provider", providerInfo.Name))

	return &http.Server{
		Addr: ":" + cfg.Port,

		Handler: router,

		ReadTimeout: cfg.RequestTimeout,

		WriteTimeout: cfg.RequestTimeout,

		IdleTimeout: 2 * time.Minute,
	}
}

// createIPRestrictedHandler wraps a handler with IP-based access control.

func createIPRestrictedHandler(handler http.HandlerFunc, allowedIPs []string, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract client IP.

		clientIP := getClientIP(r)

		// Check if IP is in allowed list.

		allowed := false

		for _, ip := range allowedIPs {
			if clientIP == ip {

				allowed = true

				break

			}
		}

		if !allowed {

			logger.Warn("Metrics access denied - IP not in allowed list",

				slog.String("client_ip", clientIP),

				slog.String("remote_addr", r.RemoteAddr),

				slog.String("path", r.URL.Path))

			w.Header().Set("Content-Type", "application/json")

			w.WriteHeader(http.StatusForbidden)

			_, _ = fmt.Fprintf(w, `{"error":"Access forbidden","message":"Your IP address is not authorized to access this endpoint"}`)

			return

		}

		// Log successful access.

		logger.Debug("Metrics access granted",

			slog.String("client_ip", clientIP),

			slog.String("path", r.URL.Path))

		// Call the original handler.

		handler(w, r)
	}
}

// getClientIP extracts the client IP address from the request.

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (common with proxies/load balancers).

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {

		// X-Forwarded-For can contain multiple IPs, take the first one.

		parts := strings.Split(xff, ",")

		if len(parts) > 0 {

			ip := strings.TrimSpace(parts[0])

			if ip != "" {
				return ip
			}

		}

	}

	// Check X-Real-IP header (nginx).

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr.

	// RemoteAddr might be in the format "IP:port", so we need to extract just the IP.

	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}

	// If SplitHostPort fails, it might already be just an IP.

	return r.RemoteAddr
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

