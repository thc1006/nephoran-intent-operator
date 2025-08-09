package auth

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/thc1006/nephoran-intent-operator/pkg/config"
)

// This file demonstrates how to integrate the EnhancedOAuth2Manager
// into the existing main.go setupHTTPServer function

// EnhancedSetupHTTPServer shows the improved server setup using the enhanced OAuth2Manager
func EnhancedSetupHTTPServer(cfg *config.LLMProcessorConfig, handler interface{}, logger *slog.Logger) *http.Server {
	router := mux.NewRouter()

	// Apply middlewares in the correct order (same as before):
	// 1. Redact Logger, 2. Security Headers, 3. CORS, 4. Rate Limiter
	// ... (middleware setup code remains the same)

	// Create enhanced OAuth2Manager configuration from LLMProcessorConfig
	enhancedConfig := &EnhancedOAuth2ManagerConfig{
		// OAuth2 configuration
		Enabled:          cfg.AuthEnabled,
		AuthConfigFile:   cfg.AuthConfigFile,
		JWTSecretKey:     cfg.JWTSecretKey,
		RequireAuth:      cfg.RequireAuth,
		AdminUsers:       cfg.AdminUsers,
		OperatorUsers:    cfg.OperatorUsers,
		StreamingEnabled: cfg.StreamingEnabled,
		MaxRequestSize:   cfg.MaxRequestSize,

		// Public endpoint configuration
		ExposeMetricsPublicly:  cfg.ExposeMetricsPublicly,
		MetricsAllowedCIDRs:    cfg.MetricsAllowedCIDRs,
		HealthEndpointsEnabled: true, // Usually always enabled for K8s orchestration
	}

	// Validate configuration
	if err := enhancedConfig.Validate(); err != nil {
		logger.Error("Enhanced OAuth2Manager configuration validation failed", slog.String("error", err.Error()))
		// Handle error appropriately
		return nil
	}

	// Create enhanced OAuth2Manager
	enhancedOAuth2Manager, err := NewEnhancedOAuth2Manager(enhancedConfig, logger)
	if err != nil {
		logger.Error("Failed to create EnhancedOAuth2Manager", slog.String("error", err.Error()))
		// Handle error appropriately
		return nil
	}

	// Prepare handlers - extract health checker from service components
	// _, _, _, _, _, _, _, healthChecker := service.GetComponents()
	
	// Create public route handlers
	publicHandlers := &PublicRouteHandlers{
		// Health:  healthChecker.HealthzHandler,  // You'll need to get these from your service
		// Ready:   healthChecker.ReadyzHandler,
		// Metrics: handler.MetricsHandler,        // You'll need to get this from your handler
	}

	// Create protected route handlers  
	protectedHandlers := &ProtectedRouteHandlers{
		// ProcessIntent:        handler.ProcessIntentHandler,
		// Status:               handler.StatusHandler, 
		// CircuitBreakerStatus: handler.CircuitBreakerStatusHandler,
		// StreamingHandler:     handler.StreamingHandler,
	}

	// Configure all routes in one centralized call
	if err := enhancedOAuth2Manager.ConfigureAllRoutes(router, publicHandlers, protectedHandlers); err != nil {
		logger.Error("Failed to configure routes", slog.String("error", err.Error()))
		return nil
	}

	return &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  cfg.RequestTimeout,
		WriteTimeout: cfg.RequestTimeout,
		IdleTimeout:  2 * time.Minute,
	}
}

// Migration example showing how to transition from the current implementation
func MigrationExample() {
	// BEFORE (current main.go approach):
	// 
	// // Setup OAuth2 routes if enabled
	// oauth2Manager.SetupRoutes(router)
	//
	// // Public health endpoints (no authentication required)
	// router.HandleFunc("/healthz", healthChecker.HealthzHandler).Methods("GET")
	// router.HandleFunc("/readyz", healthChecker.ReadyzHandler).Methods("GET")
	//
	// // Configure metrics endpoint with IP allowlist if not publicly exposed
	// if cfg.ExposeMetricsPublicly {
	//     router.HandleFunc("/metrics", handler.MetricsHandler).Methods("GET")
	// } else {
	//     router.HandleFunc("/metrics", createIPAllowlistHandler(...)).Methods("GET")
	// }
	//
	// // Create handlers with MaxBytesHandler applied to POST endpoints
	// handlers := oauth2Manager.CreateHandlersWithSizeLimit(...)
	// oauth2Manager.ConfigureProtectedRoutes(router, handlers)

	// AFTER (enhanced approach):
	//
	// // Create enhanced manager
	// enhancedManager, err := NewEnhancedOAuth2Manager(enhancedConfig, logger)
	// 
	// // Prepare handlers
	// publicHandlers := &PublicRouteHandlers{...}
	// protectedHandlers := &ProtectedRouteHandlers{...}
	//
	// // Configure all routes in one centralized call  
	// enhancedManager.ConfigureAllRoutes(router, publicHandlers, protectedHandlers)
}

// Benefits of the enhanced approach:

// 1. CENTRALIZED CONFIGURATION
//    All route setup logic is now in one place (ConfigureAllRoutes method)
//    instead of scattered across main.go

// 2. BETTER SEPARATION OF CONCERNS  
//    - Public routes (health, metrics) have their own handler struct
//    - Protected routes have their own handler struct
//    - OAuth2 routes are handled internally
//    - IP allowlisting logic is encapsulated

// 3. IMPROVED MAINTAINABILITY
//    - Adding new public routes requires only updating PublicRouteHandlers
//    - Adding new protected routes requires only updating ProtectedRouteHandlers
//    - Route security policies are centralized in the manager

// 4. ENHANCED CONFIGURABILITY
//    - Health endpoints can be disabled if needed (HealthEndpointsEnabled)
//    - Metrics security is centrally managed
//    - Streaming endpoints are conditionally registered

// 5. BETTER TESTING
//    - Each route type can be tested independently
//    - IP allowlisting logic can be unit tested
//    - Handler creation can be mocked easily

// 6. BACKWARD COMPATIBILITY
//    - Legacy methods (IsEnabled, RequiresAuth) are preserved
//    - Existing handler interfaces are maintained
//    - Migration can be done incrementally