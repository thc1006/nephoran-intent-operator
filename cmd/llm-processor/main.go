package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/thc1006/nephoran-intent-operator/pkg/auth"
	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/handlers"
	"github.com/thc1006/nephoran-intent-operator/pkg/middleware"
	"github.com/thc1006/nephoran-intent-operator/pkg/services"
)

var (
	cfg       *config.LLMProcessorConfig
	logger    *slog.Logger
	service   *services.LLMProcessorService
	handler   *handlers.LLMProcessorHandler
	startTime = time.Now()
)

func main() {
	// Initialize structured logger
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Load configuration with validation
	var err error
	cfg, err = config.LoadLLMProcessorConfig()
	if err != nil {
		logger.Error("Failed to load configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Update logger level based on configuration
	logger = createLoggerWithLevel(cfg.LogLevel)

	// Perform security validation before starting the service
	if err := validateSecurityConfiguration(cfg, logger); err != nil {
		logger.Error("Security validation failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("Starting LLM Processor service",
		slog.String("version", cfg.ServiceVersion),
		slog.String("port", cfg.Port),
		slog.String("backend_type", cfg.LLMBackendType),
		slog.String("model", cfg.LLMModelName),
		slog.Bool("auth_enabled", cfg.AuthEnabled),
		slog.Bool("require_auth", cfg.RequireAuth),
	)

	// Initialize service components
	service = services.NewLLMProcessorService(cfg, logger)

	ctx := context.Background()
	if err := service.Initialize(ctx); err != nil {
		logger.Error("Failed to initialize service", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Get initialized components
	processor, streamingProcessor, circuitBreakerMgr, tokenManager, contextBuilder, relevanceScorer, promptBuilder, healthChecker := service.GetComponents()

	// Create handler with initialized components
	handler = handlers.NewLLMProcessorHandler(
		cfg, processor, streamingProcessor, circuitBreakerMgr,
		tokenManager, contextBuilder, relevanceScorer, promptBuilder,
		logger, healthChecker, startTime,
	)

	// Set up HTTP server
	server := setupHTTPServer()

	// Mark service as ready
	healthChecker.SetReady(true)

	// Start server
	go func() {
		logger.Info("Server starting", slog.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed to start", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Server shutting down...")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.GracefulShutdown)
	defer cancel()

	// Shutdown service components
	if err := service.Shutdown(shutdownCtx); err != nil {
		logger.Error("Service shutdown failed", slog.String("error", err.Error()))
	}

	// Shutdown HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", slog.String("error", err.Error()))
	}

	logger.Info("Server exited")
}

// validateSecurityConfiguration performs fail-fast security validation
// to prevent accidental deployment of unauthenticated services in production
func validateSecurityConfiguration(cfg *config.LLMProcessorConfig, logger *slog.Logger) error {
	// Check if we're in a development environment
	isDevelopment := isDevelopmentEnvironment()

	logger.Info("Security configuration validation",
		slog.Bool("is_development", isDevelopment),
		slog.Bool("auth_enabled", cfg.AuthEnabled),
		slog.Bool("require_auth", cfg.RequireAuth),
	)

	// If authentication is disabled, only allow in development environments
	if !cfg.AuthEnabled {
		if !isDevelopment {
			return fmt.Errorf("authentication is disabled but this appears to be a production environment. " +
				"Set AUTH_ENABLED=true or ensure GO_ENV/NODE_ENV/ENVIRONMENT is set to 'development' or 'dev'")
		}

		logger.Warn("Authentication is disabled - this should only be used in development environments",
			slog.String("environment_status", "development_detected"))
	}

	// Additional security check: if auth is enabled but not required, warn in production
	if cfg.AuthEnabled && !cfg.RequireAuth && !isDevelopment {
		logger.Warn("Authentication is enabled but not required in production environment. " +
			"Consider setting REQUIRE_AUTH=true for enhanced security")
	}

	// Log security status for audit purposes
	if cfg.AuthEnabled {
		logger.Info("Authentication security status",
			slog.Bool("auth_enabled", true),
			slog.Bool("require_auth", cfg.RequireAuth),
			slog.String("jwt_secret_configured", func() string {
				if cfg.JWTSecretKey != "" {
					return "yes"
				}
				return "no"
			}()),
		)
	} else {
		logger.Warn("Service starting without authentication - development mode only")
	}

	return nil
}

// isDevelopmentEnvironment determines if the service is running in a development environment
// by checking common environment variables used to indicate development/staging environments
func isDevelopmentEnvironment() bool {
	// Check common environment indicators
	envVars := []string{"GO_ENV", "NODE_ENV", "ENVIRONMENT", "ENV", "APP_ENV"}

	for _, envVar := range envVars {
		value := strings.ToLower(os.Getenv(envVar))
		switch value {
		case "development", "dev", "local", "test", "testing":
			return true
		case "production", "prod", "staging", "stage":
			return false
		}
	}

	// If no environment variable is set, default to production for safety
	// This ensures fail-safe behavior where authentication is required by default
	return false
}

// createLoggerWithLevel creates a logger with the specified level
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
		Level:     logLevel,
		AddSource: true,
	}))
}

// setupHTTPServer configures and returns the HTTP server
func setupHTTPServer() *http.Server {
	router := mux.NewRouter()

	// Initialize request size limiting middleware
	requestSizeLimiter := middleware.NewRequestSizeLimiter(cfg.MaxRequestSize, logger)
	
	logger.Info("Request size limiting enabled",
		slog.Int64("max_request_size_bytes", cfg.MaxRequestSize),
		slog.String("max_request_size_human", fmt.Sprintf("%.2f MB", float64(cfg.MaxRequestSize)/(1024*1024))))

	// Initialize OAuth2 middleware if enabled
	var authMiddleware *auth.AuthMiddleware
	if cfg.AuthEnabled {
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

		authMiddleware = auth.NewAuthMiddleware(oauth2Config, []byte(cfg.JWTSecretKey))

		// OAuth2 authentication routes (no size limits needed for these)
		router.HandleFunc("/auth/login/{provider}", authMiddleware.LoginHandler).Methods("GET")
		router.HandleFunc("/auth/callback/{provider}", authMiddleware.CallbackHandler).Methods("GET")
		router.HandleFunc("/auth/refresh", authMiddleware.RefreshHandler).Methods("POST")
		router.HandleFunc("/auth/logout", authMiddleware.LogoutHandler).Methods("POST")
		router.HandleFunc("/auth/userinfo", authMiddleware.UserInfoHandler).Methods("GET")

		logger.Info("OAuth2 authentication enabled",
			slog.Int("providers", len(oauth2Config.Providers)))
	}

	// Public health endpoints (no authentication required)
	_, _, _, _, _, _, _, healthChecker := service.GetComponents()
	router.HandleFunc("/healthz", healthChecker.HealthzHandler).Methods("GET")
	router.HandleFunc("/readyz", healthChecker.ReadyzHandler).Methods("GET")
	router.HandleFunc("/metrics", handler.MetricsHandler).Methods("GET")

	// Protected endpoints
	if cfg.AuthEnabled && cfg.RequireAuth {
		// Apply authentication middleware to protected routes
		protectedRouter := router.PathPrefix("/").Subrouter()
		protectedRouter.Use(authMiddleware.Authenticate)

		// Main processing endpoint - requires operator role and size limits
		protectedRouter.HandleFunc("/process", 
			middleware.MaxBytesHandler(cfg.MaxRequestSize, logger, handler.ProcessIntentHandler)).Methods("POST")
		protectedRouter.Use(authMiddleware.RequireOperator())

		// Streaming endpoint - requires operator role and size limits
		if cfg.StreamingEnabled {
			protectedRouter.HandleFunc("/stream", 
				middleware.MaxBytesHandler(cfg.MaxRequestSize, logger, handler.StreamingHandler)).Methods("POST")
		}

		// Admin endpoints - requires admin role (no size limits for status endpoints)
		adminRouter := protectedRouter.PathPrefix("/admin").Subrouter()
		adminRouter.Use(authMiddleware.RequireAdmin())
		adminRouter.HandleFunc("/status", handler.StatusHandler).Methods("GET")
		adminRouter.HandleFunc("/circuit-breaker/status", handler.CircuitBreakerStatusHandler).Methods("GET")

	} else {
		// No authentication required - direct routes with size limits for POST endpoints
		router.HandleFunc("/process", 
			middleware.MaxBytesHandler(cfg.MaxRequestSize, logger, handler.ProcessIntentHandler)).Methods("POST")
		router.HandleFunc("/status", handler.StatusHandler).Methods("GET")
		router.HandleFunc("/circuit-breaker/status", handler.CircuitBreakerStatusHandler).Methods("GET")

		if cfg.StreamingEnabled {
			router.HandleFunc("/stream", 
				middleware.MaxBytesHandler(cfg.MaxRequestSize, logger, handler.StreamingHandler)).Methods("POST")
		}
	}

	return &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  cfg.RequestTimeout,
		WriteTimeout: cfg.RequestTimeout,
		IdleTimeout:  2 * time.Minute,
	}
}
