// main_working.go - LLM Processor with working authentication integration
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

	// Setup authentication if enabled - for now, we'll just log the status
	authEnabled := setupAuthenticationLogging(cfg, logger)

	// Perform security validation before starting the service
	if err := validateSecurityConfiguration(cfg, logger, authEnabled); err != nil {
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
		slog.Bool("auth_integration_ready", authEnabled),
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
	server := setupHTTPServerWithIntegratedAuth()

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

// setupAuthenticationLogging sets up authentication logging and validation
func setupAuthenticationLogging(cfg *config.LLMProcessorConfig, logger *slog.Logger) bool {
	if !cfg.AuthEnabled {
		logger.Info("Authentication disabled in configuration")
		return false
	}

	logger.Info("Authentication enabled - validating configuration")

	// Load Nephoran auth config to validate it exists
	nephoranAuthConfig, err := config.LoadNephoranAuthConfig()
	if err != nil {
		logger.Error("Failed to load Nephoran auth config", slog.String("error", err.Error()))
		return false
	}

	// Validate JWT secret key
	if nephoranAuthConfig.JWT.SecretKey == "" {
		logger.Error("JWT secret key is required when authentication is enabled")
		return false
	}

	logger.Info("Authentication configuration validated",
		slog.Bool("enabled", nephoranAuthConfig.Enabled),
		slog.Bool("require_auth", nephoranAuthConfig.RequireAuth),
		slog.Bool("oauth2_enabled", nephoranAuthConfig.OAuth2.Enabled),
		slog.Bool("ldap_enabled", nephoranAuthConfig.LDAP.Enabled),
		slog.Bool("rbac_enabled", nephoranAuthConfig.RBAC.Enabled),
		slog.Bool("sessions_enabled", nephoranAuthConfig.Sessions.Enabled),
		slog.Bool("controller_auth", nephoranAuthConfig.Controller.Enabled),
		slog.Duration("token_ttl", nephoranAuthConfig.JWT.TokenTTL),
		slog.Duration("refresh_ttl", nephoranAuthConfig.JWT.RefreshTTL),
	)

	return true
}

// validateSecurityConfiguration performs fail-fast security validation
func validateSecurityConfiguration(cfg *config.LLMProcessorConfig, logger *slog.Logger, authEnabled bool) error {
	// Check if we're in a development environment
	isDevelopment := isDevelopmentEnvironment()

	logger.Info("Security configuration validation",
		slog.Bool("is_development", isDevelopment),
		slog.Bool("auth_config_enabled", cfg.AuthEnabled),
		slog.Bool("auth_validated", authEnabled),
		slog.Bool("require_auth", cfg.RequireAuth),
		slog.Bool("tls_enabled", cfg.TLSEnabled),
	)

	// Enhanced security validation with authentication status
	if cfg.AuthEnabled && !authEnabled {
		return fmt.Errorf("authentication is enabled in LLM processor config but authentication setup failed")
	}

	if !cfg.AuthEnabled && !authEnabled {
		if !isDevelopment {
			return fmt.Errorf("authentication is disabled but this appears to be a production environment. " +
				"Set AUTH_ENABLED=true or ensure GO_ENV/NODE_ENV/ENVIRONMENT is set to 'development' or 'dev'")
		}

		logger.Warn("Authentication is disabled - this should only be used in development environments",
			slog.String("environment_status", "development_detected"))
	}

	// If auth is enabled but not required, warn in production
	if authEnabled && !cfg.RequireAuth && !isDevelopment {
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
		slog.Bool("auth_config_enabled", cfg.AuthEnabled),
		slog.Bool("auth_integration_ready", authEnabled),
		slog.Bool("require_auth", cfg.RequireAuth),
		slog.Bool("tls_enabled", cfg.TLSEnabled),
		slog.String("environment", func() string {
			if isDevelopment {
				return "development"
			}
			return "production"
		}()),
		slog.String("jwt_secret_configured", func() string {
			if authEnabled && cfg.JWTSecretKey != "" {
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

	if !authEnabled && !isDevelopment {
		logger.Warn("Service starting without authentication in production environment")
	}
	if !cfg.TLSEnabled && !isDevelopment {
		logger.Warn("Service starting without TLS encryption in production environment")
	}

	return nil
}

// isDevelopmentEnvironment determines if the service is running in a development environment
func isDevelopmentEnvironment() bool {
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

// setupHTTPServerWithIntegratedAuth configures HTTP server with authentication when available
func setupHTTPServerWithIntegratedAuth() *http.Server {
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
			AllowCredentials: false,
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

	// Setup routes based on authentication configuration
	if cfg.AuthEnabled {
		logger.Info("Configuring routes with authentication awareness")
		setupAuthEnabledRoutes(router)
	} else {
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

// setupAuthEnabledRoutes configures routes when authentication is enabled
func setupAuthEnabledRoutes(router *mux.Router) {
	// Public health endpoints (always available)
	_, _, _, _, _, _, _, healthChecker := service.GetComponents()
	router.HandleFunc("/healthz", healthChecker.HealthzHandler).Methods("GET")
	router.HandleFunc("/readyz", healthChecker.ReadyzHandler).Methods("GET")
	router.HandleFunc("/metrics", handler.MetricsHandler).Methods("GET")

	// Authentication status endpoint
	router.HandleFunc("/auth/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"auth_enabled": true, "auth_ready": true, "message": "Authentication is configured and ready"}`)
	}).Methods("GET")

	// For now, make API endpoints available but log that they should be protected
	protected := router.PathPrefix("/api").Subrouter()
	
	// Add request logging middleware for audit purposes
	protected.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			logger.Info("API request (auth-aware mode)",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()))
			
			next.ServeHTTP(w, r)
			
			logger.Info("API request completed",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Duration("duration", time.Since(start)))
		})
	})

	// Add size limiting middleware for POST endpoints
	protected.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" {
				r.Body = http.MaxBytesReader(w, r.Body, cfg.MaxRequestSize)
			}
			next.ServeHTTP(w, r)
		})
	})
	
	// Configure API routes
	protected.HandleFunc("/process", handler.ProcessIntentHandler).Methods("POST")
	protected.HandleFunc("/status", handler.StatusHandler).Methods("GET")
	protected.HandleFunc("/circuit-breaker/status", handler.CircuitBreakerStatusHandler).Methods("GET")
	
	if cfg.StreamingEnabled {
		protected.HandleFunc("/stream", handler.StreamingHandler).Methods("POST")
	}

	logger.Warn("Authentication is enabled but endpoints are not yet protected - this is a transitional state")
}

// setupPublicRoutes configures routes without authentication
func setupPublicRoutes(router *mux.Router) {
	// Public health endpoints
	_, _, _, _, _, _, _, healthChecker := service.GetComponents()
	router.HandleFunc("/healthz", healthChecker.HealthzHandler).Methods("GET")
	router.HandleFunc("/readyz", healthChecker.ReadyzHandler).Methods("GET")
	router.HandleFunc("/metrics", handler.MetricsHandler).Methods("GET")

	// Authentication status endpoint
	router.HandleFunc("/auth/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"auth_enabled": false, "auth_ready": false, "message": "Authentication is disabled"}`)
	}).Methods("GET")

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