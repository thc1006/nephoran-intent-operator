//go:build integrated && !fast_build && !disable_rag
// +build integrated,!fast_build,!disable_rag

// main_integrated.go - LLM Processor with integrated authentication
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
	"github.com/nephio-project/nephoran-intent-operator/pkg/auth"
	"github.com/nephio-project/nephoran-intent-operator/pkg/config"
	"github.com/nephio-project/nephoran-intent-operator/pkg/handlers"
	"github.com/nephio-project/nephoran-intent-operator/pkg/middleware"
	"github.com/nephio-project/nephoran-intent-operator/pkg/services"
)

var (
	cfg             *config.LLMProcessorConfig
	logger          *slog.Logger
	service         *services.LLMProcessorService
	handler         *handlers.LLMProcessorHandler
	authIntegration *auth.NephoranAuthIntegration
	startTime       = time.Now()
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

	// Setup authentication if enabled
	authIntegration, err = setupAuthentication(cfg, logger)
	if err != nil {
		logger.Error("Failed to setup authentication", slog.String("error", err.Error()))
		os.Exit(1)
	}

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
		slog.Bool("tls_enabled", cfg.TLSEnabled),
		slog.Bool("nephoran_auth_integrated", authIntegration != nil),
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

	// Set up HTTP server with integrated authentication
	server := setupHTTPServerWithAuth()

	// Mark service as ready
	healthChecker.SetReady(true)

	// Start server
	go func() {
		if cfg.TLSEnabled {
			logger.Info("Server starting with TLS",
				slog.String("addr", server.Addr),
				slog.String("cert_path", cfg.TLSCertPath),
				slog.String("key_path", cfg.TLSKeyPath))
			if err := server.ListenAndServeTLS(cfg.TLSCertPath, cfg.TLSKeyPath); err != nil && err != http.ErrServerClosed {
				logger.Error("TLS server failed to start", slog.String("error", err.Error()))
				os.Exit(1)
			}
		} else {
			logger.Info("Server starting (HTTP only)",
				slog.String("addr", server.Addr))
			logger.Warn("Server running in HTTP mode - consider enabling TLS for production use",
				slog.String("tls_config", "set TLS_ENABLED=true, TLS_CERT_PATH, and TLS_KEY_PATH to enable TLS"))
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("Server failed to start", slog.String("error", err.Error()))
				os.Exit(1)
			}
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

	// Shutdown authentication components if enabled
	if authIntegration != nil {
		logger.Info("Shutting down authentication integration")
		// Note: Add shutdown method to NephoranAuthIntegration if needed
	}

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

func setupAuthentication(cfg *config.LLMProcessorConfig, logger *slog.Logger) (*auth.NephoranAuthIntegration, error) {
	if !cfg.AuthEnabled {
		logger.Info("Authentication disabled in configuration")
		return nil, nil
	}

	logger.Info("Setting up Nephoran authentication integration")

	nephoranAuthConfig, err := config.LoadNephoranAuthConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load Nephoran auth config: %w", err)
	}

	// Map Nephoran auth config to NephoranAuthIntegration config
	authConfig := &auth.NephoranAuthConfig{
		AuthConfig: &auth.AuthConfig{
			Enabled:      nephoranAuthConfig.Enabled,
			JWTSecretKey: nephoranAuthConfig.JWT.SecretKey,
			TokenTTL:     nephoranAuthConfig.JWT.TokenTTL,
			RefreshTTL:   nephoranAuthConfig.JWT.RefreshTTL,
			Providers:    make(map[string]auth.ProviderConfig),
			AdminUsers:   []string{"admin@nephoran.com"},
		},
		ControllerAuth: &auth.ControllerAuthConfig{
			Enabled: nephoranAuthConfig.Controller.Enabled,
		},
		EndpointProtection: &auth.EndpointProtectionConfig{
			RequireAuth: nephoranAuthConfig.RequireAuth,
		},
		NephoranRBAC: &auth.NephoranRBACConfig{
			Enabled: nephoranAuthConfig.RBAC.Enabled,
		},
	}

	// Validate JWT secret key
	if authConfig.AuthConfig.JWTSecretKey == "" {
		return nil, fmt.Errorf("JWT secret key is required when authentication is enabled")
	}

	logger.Info("Nephoran authentication configuration loaded",
		slog.Bool("enabled", authConfig.AuthConfig.Enabled),
		slog.Bool("require_auth", authConfig.EndpointProtection.RequireAuth),
		slog.Bool("controller_auth", authConfig.ControllerAuth.Enabled),
		slog.Bool("rbac_enabled", authConfig.NephoranRBAC.Enabled),
		slog.Duration("token_ttl", authConfig.AuthConfig.TokenTTL),
		slog.Duration("refresh_ttl", authConfig.AuthConfig.RefreshTTL),
	)

	return auth.NewNephoranAuthIntegration(authConfig, nil, logger)
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
		slog.Bool("tls_enabled", cfg.TLSEnabled),
		slog.Bool("nephoran_auth_integrated", authIntegration != nil),
	)

	// Enhanced security validation with Nephoran integration
	if !cfg.AuthEnabled && authIntegration == nil {
		if !isDevelopment {
			return fmt.Errorf("authentication is disabled and no Nephoran auth integration found, but this appears to be a production environment. " +
				"Set AUTH_ENABLED=true or ensure GO_ENV/NODE_ENV/ENVIRONMENT is set to 'development' or 'dev'")
		}

		logger.Warn("Authentication is disabled - this should only be used in development environments",
			slog.String("environment_status", "development_detected"))
	}

	// If auth is enabled but not required, warn in production
	if (cfg.AuthEnabled || authIntegration != nil) && !cfg.RequireAuth && !isDevelopment {
		logger.Warn("Authentication is available but not required in production environment. " +
			"Consider setting REQUIRE_AUTH=true for enhanced security")
	}

	// TLS security validation
	if !cfg.TLSEnabled && !isDevelopment {
		logger.Warn("TLS is disabled in production environment. " +
			"Consider enabling TLS by setting TLS_ENABLED=true and providing certificate files for enhanced security")
	}

	// Log comprehensive security status for audit purposes
	logger.Info("Security status summary",
		slog.Bool("auth_enabled", cfg.AuthEnabled),
		slog.Bool("nephoran_auth_integrated", authIntegration != nil),
		slog.Bool("require_auth", cfg.RequireAuth),
		slog.Bool("tls_enabled", cfg.TLSEnabled),
		slog.String("environment", func() string {
			if isDevelopment {
				return "development"
			}
			return "production"
		}()),
		slog.String("jwt_secret_configured", func() string {
			if (cfg.AuthEnabled || authIntegration != nil) && cfg.JWTSecretKey != "" {
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

	if !cfg.AuthEnabled && authIntegration == nil && !isDevelopment {
		logger.Warn("Service starting without authentication in production environment")
	}
	if !cfg.TLSEnabled && !isDevelopment {
		logger.Warn("Service starting without TLS encryption in production environment")
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

// setupHTTPServerWithAuth configures and returns the HTTP server with integrated authentication
func setupHTTPServerWithAuth() *http.Server {
	router := mux.NewRouter()

	logger.Info("Request size limiting enabled",
		slog.Int64("max_request_size_bytes", cfg.MaxRequestSize),
		slog.String("max_request_size_human", fmt.Sprintf("%.2f MB", float64(cfg.MaxRequestSize)/(1024*1024))))

	// Initialize CORS middleware if enabled
	var corsMiddleware *middleware.CORSMiddleware
	if cfg.CORSEnabled {
		corsConfig := middleware.CORSConfig{
			AllowedOrigins:   cfg.AllowedOrigins,
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Requested-With", "Accept", "Origin"},
			AllowCredentials: false, // Security: disabled by default
			MaxAge:           24 * time.Hour,
		}

		if err := middleware.ValidateConfig(corsConfig); err != nil {
			logger.Error("Invalid CORS configuration", slog.String("error", err.Error()))
			os.Exit(1)
		}

		corsMiddleware = middleware.NewCORSMiddleware(corsConfig, logger)
		logger.Info("CORS middleware enabled",
			slog.Any("allowed_origins", cfg.AllowedOrigins),
			slog.Bool("credentials_allowed", corsConfig.AllowCredentials))
	}

	// Apply CORS middleware to all routes if enabled
	if corsMiddleware != nil {
		router.Use(corsMiddleware.Middleware)
	}

	// Setup authentication routes and middleware based on integration type
	if authIntegration != nil {
		// Use Nephoran authentication integration
		logger.Info("Configuring routes with Nephoran authentication integration")
		setupNephoranAuthRoutes(router)
	} else if cfg.AuthEnabled {
		// Fallback to original OAuth2Manager for backward compatibility
		logger.Info("Configuring routes with legacy OAuth2Manager")
		setupLegacyAuthRoutes(router)
	} else {
		// No authentication
		logger.Info("Configuring routes without authentication")
		setupPublicRoutes(router)
	}

	return &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  cfg.RequestTimeout,
		WriteTimeout: cfg.RequestTimeout,
		IdleTimeout:  2 * time.Minute,
	}
}

// setupNephoranAuthRoutes configures routes with Nephoran authentication integration
func setupNephoranAuthRoutes(router *mux.Router) {
	// Public health endpoints (always available)
	_, _, _, _, _, _, _, healthChecker := service.GetComponents()
	router.HandleFunc("/healthz", healthChecker.HealthzHandler).Methods("GET")
	router.HandleFunc("/readyz", healthChecker.ReadyzHandler).Methods("GET")
	router.HandleFunc("/metrics", handler.MetricsHandler).Methods("GET")

	// TODO: Implement Nephoran auth route setup when methods are available
	// For now, setup protected routes with placeholder middleware
	logger.Warn("Nephoran authentication routes not fully implemented - using placeholder setup")

	// Protected endpoints
	protected := router.PathPrefix("/api").Subrouter()

	// Add size limiting middleware for POST endpoints
	protected.Use(func(next http.Handler) http.Handler {
		return http.MaxBytesReader(nil, nil, cfg.MaxRequestSize)
	})

	// Configure routes
	protected.HandleFunc("/process", handler.ProcessIntentHandler).Methods("POST")
	protected.HandleFunc("/status", handler.StatusHandler).Methods("GET")
	protected.HandleFunc("/circuit-breaker/status", handler.CircuitBreakerStatusHandler).Methods("GET")

	if cfg.StreamingEnabled {
		protected.HandleFunc("/stream", handler.StreamingHandler).Methods("POST")
	}
}

// setupLegacyAuthRoutes configures routes with legacy OAuth2Manager for backward compatibility
func setupLegacyAuthRoutes(router *mux.Router) {
	// Initialize OAuth2Manager
	oauth2Config := &auth.OAuth2ManagerConfig{
		Enabled:          cfg.AuthEnabled,
		AuthConfigFile:   cfg.AuthConfigFile,
		JWTSecretKey:     cfg.JWTSecretKey,
		RequireAuth:      cfg.RequireAuth,
		StreamingEnabled: cfg.StreamingEnabled,
		MaxRequestSize:   cfg.MaxRequestSize,
	}

	oauth2Manager, err := auth.NewOAuth2Manager(oauth2Config, logger)
	if err != nil {
		logger.Error("Failed to create OAuth2Manager", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Setup OAuth2 routes
	oauth2Manager.SetupRoutes(router)

	// Public health endpoints
	_, _, _, _, _, _, _, healthChecker := service.GetComponents()
	router.HandleFunc("/healthz", healthChecker.HealthzHandler).Methods("GET")
	router.HandleFunc("/readyz", healthChecker.ReadyzHandler).Methods("GET")
	router.HandleFunc("/metrics", handler.MetricsHandler).Methods("GET")

	// Create handlers with MaxBytesHandler applied to POST endpoints
	handlers := oauth2Manager.CreateHandlersWithSizeLimit(
		handler.ProcessIntentHandler,
		handler.StatusHandler,
		handler.CircuitBreakerStatusHandler,
		handler.StreamingHandler,
		handler.MetricsHandler,
	)

	// Configure protected routes using OAuth2Manager
	oauth2Manager.ConfigureProtectedRoutes(router, handlers)
}

// setupPublicRoutes configures routes without authentication
func setupPublicRoutes(router *mux.Router) {
	// Public health endpoints
	_, _, _, _, _, _, _, healthChecker := service.GetComponents()
	router.HandleFunc("/healthz", healthChecker.HealthzHandler).Methods("GET")
	router.HandleFunc("/readyz", healthChecker.ReadyzHandler).Methods("GET")
	router.HandleFunc("/metrics", handler.MetricsHandler).Methods("GET")

	// Create size-limited handlers for POST endpoints
	sizeLimitedProcessHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, cfg.MaxRequestSize)
		handler.ProcessIntentHandler(w, r)
	})

	sizeLimitedStreamHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, cfg.MaxRequestSize)
		handler.StreamingHandler(w, r)
	})

	// Public API endpoints
	router.HandleFunc("/api/process", sizeLimitedProcessHandler).Methods("POST")
	router.HandleFunc("/api/status", handler.StatusHandler).Methods("GET")
	router.HandleFunc("/api/circuit-breaker/status", handler.CircuitBreakerStatusHandler).Methods("GET")

	if cfg.StreamingEnabled {
		router.HandleFunc("/api/stream", sizeLimitedStreamHandler).Methods("POST")
	}

	logger.Warn("Service configured without authentication - all endpoints are publicly accessible")
}
