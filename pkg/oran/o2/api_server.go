// Package o2 implements O-RAN Infrastructure Management Service (IMS) API Server
// Following O-RAN.WG6.O2ims-Interface-v01.01 specification
package o2

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/thc1006/nephoran-intent-operator/pkg/auth"
	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/middleware"
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/providers"
)

// O2APIServer represents the O2 IMS RESTful API server
type O2APIServer struct {
	// Configuration
	config *O2IMSConfig
	logger *logging.StructuredLogger

	// Core services
	imsService      O2IMSService
	resourceManager ResourceManager
	inventoryService InventoryService
	monitoringService MonitoringService

	// HTTP server components
	router     *mux.Router
	httpServer *http.Server

	// Middleware components
	authMiddleware    auth.AuthMiddleware
	corsMiddleware    *middleware.CORSMiddleware
	rateLimitMiddleware http.Handler
	
	// Metrics and monitoring
	metricsRegistry *prometheus.Registry
	metrics         *APIMetrics
	healthChecker   *HealthChecker

	// Provider registry
	providerRegistry *providers.ProviderRegistry

	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
}

// APIMetrics contains Prometheus metrics for the API server
type APIMetrics struct {
	requestsTotal      *prometheus.CounterVec
	requestDuration    *prometheus.HistogramVec
	activeConnections  prometheus.Gauge
	resourceOperations *prometheus.CounterVec
	errorRate          *prometheus.CounterVec
	responseSize       *prometheus.HistogramVec
}

// HealthChecker manages health checking for the API server
type HealthChecker struct {
	config         *HealthCheckConfig
	healthChecks   map[string]ComponentHealthCheck
	lastHealthData *HealthCheck
	ticker         *time.Ticker
	stopCh         chan struct{}
}

// ComponentHealthCheck represents a health check for a component
type ComponentHealthCheck func(ctx context.Context) ComponentCheck

// NewO2APIServer creates a new O2 IMS API server instance
func NewO2APIServer(config *O2IMSConfig) (*O2APIServer, error) {
	if config == nil {
		config = DefaultO2IMSConfig()
	}

	if config.Logger == nil {
		config.Logger = logging.NewStructuredLogger(
			logging.WithService("o2-ims-api"),
			logging.WithVersion("1.0.0"),
		)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Initialize metrics registry
	metricsRegistry := prometheus.NewRegistry()
	metrics := newAPIMetrics(metricsRegistry)

	// Initialize provider registry
	providerRegistry := providers.NewProviderRegistry(config.CloudProviders)

	// Initialize health checker
	healthChecker, err := newHealthChecker(config.HealthCheckConfig, config.Logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create health checker: %w", err)
	}

	server := &O2APIServer{
		config:           config,
		logger:           config.Logger,
		ctx:              ctx,
		cancel:           cancel,
		metricsRegistry:  metricsRegistry,
		metrics:          metrics,
		healthChecker:    healthChecker,
		providerRegistry: providerRegistry,
	}

	// Initialize core services
	if err := server.initializeServices(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	// Initialize HTTP server components
	if err := server.initializeHTTPServer(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize HTTP server: %w", err)
	}

	return server, nil
}

// initializeServices initializes the core O2 IMS services
func (s *O2APIServer) initializeServices() error {
	s.logger.Info("initializing O2 IMS services")

	// Initialize storage if configured
	storage, err := s.initializeStorage()
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Initialize core services with proper dependency injection
	s.imsService = NewO2IMSServiceImpl(s.config, storage, s.providerRegistry, s.logger)
	s.resourceManager = NewResourceManagerImpl(s.config, s.providerRegistry, s.logger)
	s.inventoryService = NewInventoryServiceImpl(s.config, s.providerRegistry, s.logger)
	s.monitoringService = NewMonitoringServiceImpl(s.config, s.logger)

	// Register health checks for services
	s.healthChecker.RegisterHealthCheck("ims-service", s.imsServiceHealthCheck)
	s.healthChecker.RegisterHealthCheck("resource-manager", s.resourceManagerHealthCheck)
	s.healthChecker.RegisterHealthCheck("inventory-service", s.inventoryServiceHealthCheck)
	s.healthChecker.RegisterHealthCheck("monitoring-service", s.monitoringServiceHealthCheck)

	return nil
}

// initializeHTTPServer initializes the HTTP server and routing
func (s *O2APIServer) initializeHTTPServer() error {
	s.logger.Info("initializing HTTP server", 
		"port", s.config.Port,
		"tls_enabled", s.config.TLSEnabled)

	// Initialize router
	s.router = mux.NewRouter()

	// Initialize middleware
	if err := s.initializeMiddleware(); err != nil {
		return fmt.Errorf("failed to initialize middleware: %w", err)
	}

	// Setup routes
	s.setupRoutes()

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.httpServer = &http.Server{
		Addr:           addr,
		Handler:        s.router,
		ReadTimeout:    s.config.ReadTimeout,
		WriteTimeout:   s.config.WriteTimeout,
		IdleTimeout:    s.config.IdleTimeout,
		MaxHeaderBytes: s.config.MaxHeaderBytes,
	}

	return nil
}

// initializeMiddleware initializes HTTP middleware stack
func (s *O2APIServer) initializeMiddleware() error {
	s.logger.Info("initializing middleware stack")

	// Initialize authentication middleware
	if s.config.AuthenticationConfig != nil && s.config.AuthenticationConfig.Enabled {
		authMiddleware, err := auth.NewJWTMiddleware(&auth.JWTConfig{
			Secret:         s.config.AuthenticationConfig.JWTSecret,
			TokenExpiry:    s.config.AuthenticationConfig.TokenExpiry,
			AllowedIssuers: s.config.AuthenticationConfig.AllowedIssuers,
			RequiredClaims: s.config.AuthenticationConfig.RequiredClaims,
		}, s.logger.Logger)
		if err != nil {
			return fmt.Errorf("failed to initialize auth middleware: %w", err)
		}
		s.authMiddleware = authMiddleware
	}

	// Initialize CORS middleware
	if s.config.SecurityConfig != nil && s.config.SecurityConfig.CORSEnabled {
		corsConfig := middleware.CORSConfig{
			AllowedOrigins:   s.config.SecurityConfig.CORSAllowedOrigins,
			AllowedMethods:   s.config.SecurityConfig.CORSAllowedMethods,
			AllowedHeaders:   s.config.SecurityConfig.CORSAllowedHeaders,
			AllowCredentials: s.config.AuthenticationConfig != nil && s.config.AuthenticationConfig.Enabled,
		}
		
		if err := middleware.ValidateConfig(corsConfig); err != nil {
			return fmt.Errorf("invalid CORS config: %w", err)
		}
		
		s.corsMiddleware = middleware.NewCORSMiddleware(corsConfig, s.logger.Logger)
	}

	// Initialize rate limiting middleware
	if s.config.SecurityConfig != nil && s.config.SecurityConfig.RateLimitConfig != nil && s.config.SecurityConfig.RateLimitConfig.Enabled {
		rateLimitMiddleware := s.createRateLimitMiddleware()
		s.rateLimitMiddleware = rateLimitMiddleware
	}

	return nil
}

// setupRoutes configures all API routes following O-RAN O2 specification
func (s *O2APIServer) setupRoutes() {
	s.logger.Info("setting up API routes")

	// Apply global middleware
	s.router.Use(s.loggingMiddleware)
	s.router.Use(s.metricsMiddleware)
	s.router.Use(s.recoveryMiddleware)

	if s.corsMiddleware != nil {
		s.router.Use(s.corsMiddleware.Middleware)
	}

	if s.rateLimitMiddleware != nil {
		s.router.Use(func(next http.Handler) http.Handler {
			return s.rateLimitMiddleware
		})
	}

	// API versioning - O2 IMS v1.0
	apiV1 := s.router.PathPrefix("/ims/v1").Subrouter()

	// Apply authentication middleware to protected routes
	if s.authMiddleware != nil {
		apiV1.Use(s.authMiddleware.Middleware)
	}

	// Service information endpoints
	s.router.HandleFunc("/ims/info", s.handleGetServiceInfo).Methods("GET")
	s.router.HandleFunc("/health", s.handleHealthCheck).Methods("GET")
	s.router.HandleFunc("/ready", s.handleReadinessCheck).Methods("GET")

	// Metrics endpoint (usually should be on separate port in production)
	if s.config.MetricsConfig != nil && s.config.MetricsConfig.Enabled {
		s.router.Handle("/metrics", promhttp.HandlerFor(s.metricsRegistry, promhttp.HandlerOpts{}))
	}

	// Resource Pool Management - O2 IMS Core API
	apiV1.HandleFunc("/resourcePools", s.handleGetResourcePools).Methods("GET")
	apiV1.HandleFunc("/resourcePools/{resourcePoolId}", s.handleGetResourcePool).Methods("GET")
	apiV1.HandleFunc("/resourcePools", s.handleCreateResourcePool).Methods("POST")
	apiV1.HandleFunc("/resourcePools/{resourcePoolId}", s.handleUpdateResourcePool).Methods("PUT")
	apiV1.HandleFunc("/resourcePools/{resourcePoolId}", s.handleDeleteResourcePool).Methods("DELETE")

	// Resource Type Management
	apiV1.HandleFunc("/resourceTypes", s.handleGetResourceTypes).Methods("GET")
	apiV1.HandleFunc("/resourceTypes/{resourceTypeId}", s.handleGetResourceType).Methods("GET")
	apiV1.HandleFunc("/resourceTypes", s.handleCreateResourceType).Methods("POST")
	apiV1.HandleFunc("/resourceTypes/{resourceTypeId}", s.handleUpdateResourceType).Methods("PUT")
	apiV1.HandleFunc("/resourceTypes/{resourceTypeId}", s.handleDeleteResourceType).Methods("DELETE")

	// Resource Management
	apiV1.HandleFunc("/resources", s.handleGetResources).Methods("GET")
	apiV1.HandleFunc("/resources/{resourceId}", s.handleGetResource).Methods("GET")
	apiV1.HandleFunc("/resources", s.handleCreateResource).Methods("POST")
	apiV1.HandleFunc("/resources/{resourceId}", s.handleUpdateResource).Methods("PUT")
	apiV1.HandleFunc("/resources/{resourceId}", s.handleDeleteResource).Methods("DELETE")

	// Resource Health and Monitoring
	apiV1.HandleFunc("/resources/{resourceId}/health", s.handleGetResourceHealth).Methods("GET")
	apiV1.HandleFunc("/resources/{resourceId}/alarms", s.handleGetResourceAlarms).Methods("GET")
	apiV1.HandleFunc("/resources/{resourceId}/metrics", s.handleGetResourceMetrics).Methods("GET")

	// Deployment Template Management
	apiV1.HandleFunc("/deploymentTemplates", s.handleGetDeploymentTemplates).Methods("GET")
	apiV1.HandleFunc("/deploymentTemplates/{templateId}", s.handleGetDeploymentTemplate).Methods("GET")
	apiV1.HandleFunc("/deploymentTemplates", s.handleCreateDeploymentTemplate).Methods("POST")
	apiV1.HandleFunc("/deploymentTemplates/{templateId}", s.handleUpdateDeploymentTemplate).Methods("PUT")
	apiV1.HandleFunc("/deploymentTemplates/{templateId}", s.handleDeleteDeploymentTemplate).Methods("DELETE")

	// Deployment Management
	apiV1.HandleFunc("/deployments", s.handleGetDeployments).Methods("GET")
	apiV1.HandleFunc("/deployments/{deploymentId}", s.handleGetDeployment).Methods("GET")
	apiV1.HandleFunc("/deployments", s.handleCreateDeployment).Methods("POST")
	apiV1.HandleFunc("/deployments/{deploymentId}", s.handleUpdateDeployment).Methods("PUT")
	apiV1.HandleFunc("/deployments/{deploymentId}", s.handleDeleteDeployment).Methods("DELETE")

	// Subscription and Event Management
	apiV1.HandleFunc("/subscriptions", s.handleCreateSubscription).Methods("POST")
	apiV1.HandleFunc("/subscriptions", s.handleGetSubscriptions).Methods("GET")
	apiV1.HandleFunc("/subscriptions/{subscriptionId}", s.handleGetSubscription).Methods("GET")
	apiV1.HandleFunc("/subscriptions/{subscriptionId}", s.handleUpdateSubscription).Methods("PUT")
	apiV1.HandleFunc("/subscriptions/{subscriptionId}", s.handleDeleteSubscription).Methods("DELETE")

	// Cloud Provider Management (Nephoran extension)
	apiV1.HandleFunc("/cloudProviders", s.handleRegisterCloudProvider).Methods("POST")
	apiV1.HandleFunc("/cloudProviders", s.handleGetCloudProviders).Methods("GET")
	apiV1.HandleFunc("/cloudProviders/{providerId}", s.handleGetCloudProvider).Methods("GET")
	apiV1.HandleFunc("/cloudProviders/{providerId}", s.handleUpdateCloudProvider).Methods("PUT")
	apiV1.HandleFunc("/cloudProviders/{providerId}", s.handleRemoveCloudProvider).Methods("DELETE")

	// Resource Lifecycle Operations (Extended API)
	apiV1.HandleFunc("/resources/{resourceId}/provision", s.handleProvisionResource).Methods("POST")
	apiV1.HandleFunc("/resources/{resourceId}/configure", s.handleConfigureResource).Methods("POST")
	apiV1.HandleFunc("/resources/{resourceId}/scale", s.handleScaleResource).Methods("POST")
	apiV1.HandleFunc("/resources/{resourceId}/migrate", s.handleMigrateResource).Methods("POST")
	apiV1.HandleFunc("/resources/{resourceId}/backup", s.handleBackupResource).Methods("POST")
	apiV1.HandleFunc("/resources/{resourceId}/restore/{backupId}", s.handleRestoreResource).Methods("POST")
	apiV1.HandleFunc("/resources/{resourceId}/terminate", s.handleTerminateResource).Methods("POST")

	// Infrastructure Discovery and Inventory (Extended API)
	apiV1.HandleFunc("/discovery/infrastructure/{providerId}", s.handleDiscoverInfrastructure).Methods("POST")
	apiV1.HandleFunc("/inventory/sync", s.handleSyncInventory).Methods("POST")
	apiV1.HandleFunc("/inventory/assets", s.handleGetAssets).Methods("GET")
	apiV1.HandleFunc("/inventory/assets/{assetId}", s.handleGetAsset).Methods("GET")

	s.logger.Info("API routes configured successfully")
}

// Start starts the O2 IMS API server
func (s *O2APIServer) Start(ctx context.Context) error {
	s.logger.Info("starting O2 IMS API server")

	// Start background services
	if err := s.startBackgroundServices(ctx); err != nil {
		return fmt.Errorf("failed to start background services: %w", err)
	}

	// Start HTTP server
	go func() {
		var err error
		if s.config.TLSEnabled {
			s.logger.Info("starting HTTPS server",
				"addr", s.httpServer.Addr,
				"cert_file", s.config.CertFile)
			err = s.httpServer.ListenAndServeTLS(s.config.CertFile, s.config.KeyFile)
		} else {
			s.logger.Info("starting HTTP server",
				"addr", s.httpServer.Addr)
			err = s.httpServer.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP server error", "error", err)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	return s.Shutdown(context.Background())
}

// Shutdown gracefully shuts down the API server
func (s *O2APIServer) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down O2 IMS API server")

	// Cancel server context
	s.cancel()

	// Shutdown HTTP server
	if s.httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("HTTP server shutdown error", "error", err)
		}
	}

	// Stop background services
	s.stopBackgroundServices()

	s.logger.Info("O2 IMS API server shutdown completed")
	return nil
}

// startBackgroundServices starts background services like health checking
func (s *O2APIServer) startBackgroundServices(ctx context.Context) error {
	s.logger.Info("starting background services")

	// Start health checker
	if s.healthChecker != nil {
		go s.healthChecker.Start(ctx)
	}

	// Start metrics collection
	if s.config.MetricsConfig != nil && s.config.MetricsConfig.Enabled {
		go s.startMetricsCollection(ctx)
	}

	return nil
}

// stopBackgroundServices stops all background services
func (s *O2APIServer) stopBackgroundServices() {
	s.logger.Info("stopping background services")

	if s.healthChecker != nil {
		s.healthChecker.Stop()
	}
}

// startMetricsCollection starts periodic metrics collection
func (s *O2APIServer) startMetricsCollection(ctx context.Context) {
	ticker := time.NewTicker(s.config.MetricsConfig.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.collectMetrics()
		}
	}
}

// collectMetrics collects and updates metrics
func (s *O2APIServer) collectMetrics() {
	// Update active connections metric
	// This would typically be implemented with actual connection tracking
	s.metrics.activeConnections.Set(0) // Placeholder

	// Additional metrics collection would be implemented here
}

// Utility methods for extracting path parameters and query parameters

// getPathParam extracts a path parameter from the request
func (s *O2APIServer) getPathParam(r *http.Request, param string) string {
	vars := mux.Vars(r)
	return vars[param]
}

// getQueryParam extracts a query parameter from the request
func (s *O2APIServer) getQueryParam(r *http.Request, param string) string {
	return r.URL.Query().Get(param)
}

// getQueryParamInt extracts an integer query parameter
func (s *O2APIServer) getQueryParamInt(r *http.Request, param string, defaultValue int) int {
	value := r.URL.Query().Get(param)
	if value == "" {
		return defaultValue
	}
	
	if intValue, err := strconv.Atoi(value); err == nil {
		return intValue
	}
	
	return defaultValue
}

// getQueryParamBool extracts a boolean query parameter
func (s *O2APIServer) getQueryParamBool(r *http.Request, param string, defaultValue bool) bool {
	value := r.URL.Query().Get(param)
	if value == "" {
		return defaultValue
	}
	
	if boolValue, err := strconv.ParseBool(value); err == nil {
		return boolValue
	}
	
	return defaultValue
}

// parseResourcePoolFilter parses resource pool filter from query parameters
func (s *O2APIServer) parseResourcePoolFilter(r *http.Request) *models.ResourcePoolFilter {
	filter := &models.ResourcePoolFilter{
		Limit:  s.getQueryParamInt(r, "limit", 100),
		Offset: s.getQueryParamInt(r, "offset", 0),
	}

	if names := r.URL.Query().Get("names"); names != "" {
		filter.Names = []string{names} // Simple implementation - could be enhanced for multiple names
	}

	if locations := r.URL.Query().Get("locations"); locations != "" {
		filter.Locations = []string{locations}
	}

	if providers := r.URL.Query().Get("providers"); providers != "" {
		filter.Providers = []string{providers}
	}

	return filter
}

// Helper methods for service health checks
func (s *O2APIServer) imsServiceHealthCheck(ctx context.Context) ComponentCheck {
	// Implement actual health check for IMS service
	return ComponentCheck{
		Name:      "ims-service",
		Status:    "UP",
		Message:   "IMS service is healthy",
		Timestamp: time.Now(),
	}
}

func (s *O2APIServer) resourceManagerHealthCheck(ctx context.Context) ComponentCheck {
	// Implement actual health check for resource manager
	return ComponentCheck{
		Name:      "resource-manager",
		Status:    "UP",
		Message:   "Resource manager is healthy",
		Timestamp: time.Now(),
	}
}

func (s *O2APIServer) inventoryServiceHealthCheck(ctx context.Context) ComponentCheck {
	// Implement actual health check for inventory service
	return ComponentCheck{
		Name:      "inventory-service",
		Status:    "UP",
		Message:   "Inventory service is healthy",
		Timestamp: time.Now(),
	}
}

func (s *O2APIServer) monitoringServiceHealthCheck(ctx context.Context) ComponentCheck {
	// Implement actual health check for monitoring service
	return ComponentCheck{
		Name:      "monitoring-service",
		Status:    "UP",
		Message:   "Monitoring service is healthy",
		Timestamp: time.Now(),
	}
}