/*

Copyright 2025.



Licensed under the Apache License, Version 2.0 (the "License");

you may not use this file except in compliance with the License.

You may obtain a copy of the License at



    http://www.apache.org/licenses/LICENSE-2.0



Unless required by applicable law or agreed to in writing, software

distributed under the License is distributed on an "AS IS" BASIS,

WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.

See the License for the specific language governing permissions and

limitations under the License.

*/

// Package webui provides comprehensive Web UI integration for the Nephoran Intent Operator.
package webui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/cors"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/thc1006/nephoran-intent-operator/pkg/auth"
	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers"
	"github.com/thc1006/nephoran-intent-operator/pkg/multicluster"
	"github.com/thc1006/nephoran-intent-operator/pkg/packagerevision"
	"github.com/thc1006/nephoran-intent-operator/pkg/services"
)

// NephoranAPIServer provides comprehensive Web UI integration for the Nephoran Intent Operator.

type NephoranAPIServer struct {

	// Core dependencies.

	intentReconciler *controllers.NetworkIntentReconciler

	packageManager packagerevision.PackageRevisionManager

	clusterManager multicluster.ClusterPropagationManager

	llmProcessor *services.LLMProcessorService

	kubeClient kubernetes.Interface

	// Authentication and authorization.

	authMiddleware *auth.AuthMiddleware

	sessionManager *auth.SessionManager

	rbacManager *auth.RBACManager

	// Configuration and metrics.

	config *ServerConfig

	metrics *ServerMetrics

	logger logr.Logger

	// HTTP server and routing.

	server *http.Server

	router *mux.Router

	upgrader websocket.Upgrader

	// WebSocket and SSE connection management.

	wsConnections map[string]*WebSocketConnection

	sseConnections map[string]*SSEConnection

	connectionsMutex sync.RWMutex

	// Caching and performance.

	cache *Cache

	rateLimiter *RateLimiter

	// Background workers.

	shutdown chan struct{}

	wg sync.WaitGroup
}

// ServerConfig holds API server configuration.

type ServerConfig struct {

	// Server settings.

	Address string `json:"address"`

	Port int `json:"port"`

	ReadTimeout time.Duration `json:"read_timeout"`

	WriteTimeout time.Duration `json:"write_timeout"`

	IdleTimeout time.Duration `json:"idle_timeout"`

	ShutdownTimeout time.Duration `json:"shutdown_timeout"`

	// TLS settings.

	TLSEnabled bool `json:"tls_enabled"`

	TLSCertFile string `json:"tls_cert_file"`

	TLSKeyFile string `json:"tls_key_file"`

	// CORS settings.

	EnableCORS bool `json:"enable_cors"`

	AllowedOrigins []string `json:"allowed_origins"`

	AllowedMethods []string `json:"allowed_methods"`

	AllowedHeaders []string `json:"allowed_headers"`

	AllowCredentials bool `json:"allow_credentials"`

	// Rate limiting.

	EnableRateLimit bool `json:"enable_rate_limit"`

	RequestsPerMin int `json:"requests_per_min"`

	BurstSize int `json:"burst_size"`

	RateLimitWindow time.Duration `json:"rate_limit_window"`

	// Caching.

	EnableCaching bool `json:"enable_caching"`

	CacheSize int `json:"cache_size"`

	CacheTTL time.Duration `json:"cache_ttl"`

	// WebSocket and SSE.

	MaxWSConnections int `json:"max_ws_connections"`

	WSTimeout time.Duration `json:"ws_timeout"`

	SSETimeout time.Duration `json:"sse_timeout"`

	PingInterval time.Duration `json:"ping_interval"`

	// Pagination.

	DefaultPageSize int `json:"default_page_size"`

	MaxPageSize int `json:"max_page_size"`

	// Feature flags.

	EnableMetrics bool `json:"enable_metrics"`

	EnableProfiling bool `json:"enable_profiling"`

	EnableHealthCheck bool `json:"enable_health_check"`

	// Authentication configuration.

	AuthConfigFile string `json:"auth_config_file"`
}

// ServerMetrics contains Prometheus metrics for the API server.

type ServerMetrics struct {
	RequestsTotal *prometheus.CounterVec

	RequestDuration *prometheus.HistogramVec

	WSConnectionsActive prometheus.Gauge

	SSEConnectionsActive prometheus.Gauge

	CacheHits prometheus.Counter

	CacheMisses prometheus.Counter

	RateLimitExceeded prometheus.Counter
}

// WebSocketConnection represents an active WebSocket connection.

type WebSocketConnection struct {
	ID string

	UserID string

	Connection *websocket.Conn

	Send chan []byte

	Filters map[string]interface{}

	LastSeen time.Time
}

// SSEConnection represents an active Server-Sent Events connection.

type SSEConnection struct {
	ID string

	UserID string

	Writer http.ResponseWriter

	Flusher http.Flusher

	Filters map[string]interface{}

	LastSeen time.Time
}

// APIResponse represents a standard API response format.

type APIResponse struct {
	Success bool `json:"success"`

	Data interface{} `json:"data,omitempty"`

	Error *APIError `json:"error,omitempty"`

	Meta *Meta `json:"meta,omitempty"`

	Links *Links `json:"links,omitempty"`

	Timestamp time.Time `json:"timestamp"`
}

// APIError represents an API error response.

type APIError struct {
	Code string `json:"code"`

	Message string `json:"message"`

	Details interface{} `json:"details,omitempty"`

	RequestID string `json:"request_id,omitempty"`
}

// Meta contains response metadata for pagination and filtering.

type Meta struct {
	Page int `json:"page"`

	PageSize int `json:"page_size"`

	TotalPages int `json:"total_pages"`

	TotalItems int `json:"total_items"`
}

// Links contains HATEOAS navigation links.

type Links struct {
	Self string `json:"self"`

	First string `json:"first,omitempty"`

	Last string `json:"last,omitempty"`

	Previous string `json:"previous,omitempty"`

	Next string `json:"next,omitempty"`
}

// PaginationParams represents pagination parameters.

type PaginationParams struct {
	Page int `json:"page"`

	PageSize int `json:"page_size"`

	Sort string `json:"sort"`

	Order string `json:"order"`
}

// FilterParams represents filtering parameters.

type FilterParams struct {
	Status string `json:"status,omitempty"`

	Type string `json:"type,omitempty"`

	Priority string `json:"priority,omitempty"`

	Component string `json:"component,omitempty"`

	Cluster string `json:"cluster,omitempty"`

	Labels map[string]string `json:"labels,omitempty"`

	Since *time.Time `json:"since,omitempty"`

	Until *time.Time `json:"until,omitempty"`
}

// NewNephoranAPIServer creates a new API server instance.

func NewNephoranAPIServer(

	intentReconciler *controllers.NetworkIntentReconciler,

	packageManager packagerevision.PackageRevisionManager,

	clusterManager multicluster.ClusterPropagationManager,

	llmProcessor *services.LLMProcessorService,

	kubeClient kubernetes.Interface,

	config *ServerConfig,

) (*NephoranAPIServer, error) {

	if config == nil {

		config = getDefaultServerConfig()

	}

	logger := log.Log.WithName("nephoran-api-server")

	// Initialize metrics.

	metrics := initServerMetrics()

	prometheus.MustRegister(

		metrics.RequestsTotal,

		metrics.RequestDuration,

		metrics.WSConnectionsActive,

		metrics.SSEConnectionsActive,

		metrics.CacheHits,

		metrics.CacheMisses,

		metrics.RateLimitExceeded,
	)

	// Initialize cache if enabled.

	var cache *Cache

	if config.EnableCaching {

		cache = NewCache(config.CacheSize, config.CacheTTL)

	}

	// Initialize rate limiter if enabled.

	var rateLimiter *RateLimiter

	if config.EnableRateLimit {

		rateLimiter = NewRateLimiter(config.RequestsPerMin, config.BurstSize, config.RateLimitWindow)

	}

	server := &NephoranAPIServer{

		intentReconciler: intentReconciler,

		packageManager: packageManager,

		clusterManager: clusterManager,

		llmProcessor: llmProcessor,

		kubeClient: kubeClient,

		config: config,

		metrics: metrics,

		logger: logger,

		wsConnections: make(map[string]*WebSocketConnection),

		sseConnections: make(map[string]*SSEConnection),

		cache: cache,

		rateLimiter: rateLimiter,

		shutdown: make(chan struct{}),

		upgrader: websocket.Upgrader{

			CheckOrigin: func(r *http.Request) bool {

				return true // Configure based on CORS settings

			},

			ReadBufferSize: 1024,

			WriteBufferSize: 1024,
		},
	}

	// Initialize authentication middleware.

	if err := server.initializeAuth(); err != nil {

		return nil, fmt.Errorf("failed to initialize authentication: %w", err)

	}

	// Setup router and routes.

	server.setupRouter()

	// Create HTTP server.

	server.server = &http.Server{

		Addr: fmt.Sprintf("%s:%d", config.Address, config.Port),

		Handler: server.router,

		ReadTimeout: config.ReadTimeout,

		WriteTimeout: config.WriteTimeout,

		IdleTimeout: config.IdleTimeout,
	}

	// Start background workers.

	server.startBackgroundWorkers()

	logger.Info("Nephoran API Server initialized",

		"address", config.Address,

		"port", config.Port,

		"tls_enabled", config.TLSEnabled,

		"cors_enabled", config.EnableCORS,

		"rate_limit_enabled", config.EnableRateLimit,

		"caching_enabled", config.EnableCaching)

	return server, nil

}

// initializeAuth sets up authentication and authorization middleware.

func (s *NephoranAPIServer) initializeAuth() error {

	authConfig, err := auth.LoadAuthConfig(context.Background(), s.config.AuthConfigFile)

	if err != nil {

		return fmt.Errorf("failed to load auth config: %w", err)

	}

	if !authConfig.Enabled {

		s.logger.Info("Authentication disabled")

		return nil

	}

	// Initialize session manager.

	// Note: NewSessionManager requires config, jwtManager, rbacManager, logger - will be set after managers are created.

	// s.sessionManager = auth.NewSessionManager(authConfig, s.logger.WithName("session-manager")).

	// Initialize RBAC manager.

	// Create RBACManagerConfig from RBACConfig.

	rbacManagerConfig := &auth.RBACManagerConfig{

		CacheTTL: 15 * time.Minute,

		EnableHierarchy: true,

		DefaultDenyAll: true,

		PolicyEvaluation: "deny-overrides",

		MaxPolicyDepth: 10,

		EnableAuditLogging: true,
	}

	s.rbacManager = auth.NewRBACManager(rbacManagerConfig, nil)

	// Initialize JWT manager.

	// Note: NewJWTManager requires config, tokenStore, blacklist, logger.

	jwtManager, err := auth.NewJWTManager(context.Background(), &auth.JWTConfig{

		Issuer: "nephoran",

		DefaultTTL: 24 * time.Hour,

		RefreshTTL: 7 * 24 * time.Hour,
	}, nil, nil, nil)

	if err != nil {

		return fmt.Errorf("failed to create JWT manager: %w", err)

	}

	// Initialize auth middleware.

	middlewareConfig := &auth.MiddlewareConfig{

		SkipAuth: []string{

			"/health", "/metrics", "/openapi", "/docs",

			"/auth/login", "/auth/callback", "/auth/providers",
		},

		EnableCORS: s.config.EnableCORS,

		AllowedOrigins: s.config.AllowedOrigins,

		AllowedMethods: s.config.AllowedMethods,

		AllowedHeaders: s.config.AllowedHeaders,

		AllowCredentials: s.config.AllowCredentials,
	}

	s.authMiddleware = auth.NewAuthMiddleware(s.sessionManager, jwtManager, s.rbacManager, middlewareConfig)

	s.logger.Info("Authentication initialized successfully")

	return nil

}

// setupRouter configures the HTTP router with all endpoints.

func (s *NephoranAPIServer) setupRouter() {

	s.router = mux.NewRouter().StrictSlash(true)

	// Apply middleware.

	s.router.Use(s.loggingMiddleware)

	s.router.Use(s.metricsMiddleware)

	if s.rateLimiter != nil {

		s.router.Use(s.rateLimitMiddleware)

	}

	if s.authMiddleware != nil {

		s.router.Use(s.authMiddleware.AuthenticateMiddleware)

		s.router.Use(s.authMiddleware.RequestLoggingMiddleware)

	}

	// Setup API versioned routes.

	v1 := s.router.PathPrefix("/api/v1").Subrouter()

	// Intent Management APIs.

	s.setupIntentRoutes(v1)

	// Package Management APIs.

	s.setupPackageRoutes(v1)

	// Multi-Cluster APIs.

	s.setupClusterRoutes(v1)

	// Real-time APIs.

	s.setupRealtimeRoutes(v1)

	// Dashboard APIs.

	s.setupDashboardRoutes(v1)

	// System APIs.

	s.setupSystemRoutes(s.router)

	// Setup CORS if enabled.

	if s.config.EnableCORS {

		c := cors.New(cors.Options{

			AllowedOrigins: s.config.AllowedOrigins,

			AllowedMethods: s.config.AllowedMethods,

			AllowedHeaders: s.config.AllowedHeaders,

			AllowCredentials: s.config.AllowCredentials,
		})

		s.router = c.Handler(s.router).(*mux.Router)

	}

}

// Start starts the API server.

func (s *NephoranAPIServer) Start(ctx context.Context) error {

	s.logger.Info("Starting Nephoran API Server", "address", s.server.Addr)

	// Start server.

	go func() {

		var err error

		if s.config.TLSEnabled {

			err = s.server.ListenAndServeTLS(s.config.TLSCertFile, s.config.TLSKeyFile)

		} else {

			err = s.server.ListenAndServe()

		}

		if err != nil && err != http.ErrServerClosed {

			s.logger.Error(err, "API server failed to start")

		}

	}()

	// Wait for shutdown signal.

	<-ctx.Done()

	return s.Shutdown()

}

// Shutdown gracefully shuts down the API server.

func (s *NephoranAPIServer) Shutdown() error {

	s.logger.Info("Shutting down Nephoran API Server")

	// Signal background workers to stop.

	close(s.shutdown)

	// Wait for background workers to finish.

	s.wg.Wait()

	// Close WebSocket connections.

	s.connectionsMutex.Lock()

	for id, conn := range s.wsConnections {

		if err := conn.Connection.Close(); err != nil {

			s.logger.Error(err, "Failed to close WebSocket connection",

				"connection_id", id)

		}

		delete(s.wsConnections, id)

	}

	s.connectionsMutex.Unlock()

	// Shutdown HTTP server.

	ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)

	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {

		s.logger.Error(err, "Error during server shutdown")

		return err

	}

	s.logger.Info("Nephoran API Server shutdown complete")

	return nil

}

// Middleware functions.

func (s *NephoranAPIServer) loggingMiddleware(next http.Handler) http.Handler {

	// handlers.LoggingHandler expects io.Writer, but we have logr.Logger.

	// Use a simple logging wrapper instead.

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		start := time.Now()

		next.ServeHTTP(w, r)

		s.logger.Info("HTTP request processed",

			"method", r.Method,

			"path", r.URL.Path,

			"duration", time.Since(start))

	})

}

func (s *NephoranAPIServer) metricsMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		start := time.Now()

		// Wrap response writer to capture status code.

		wrapper := &responseWrapper{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapper, r)

		duration := time.Since(start)

		s.metrics.RequestsTotal.WithLabelValues(

			r.Method,

			r.URL.Path,

			fmt.Sprintf("%d", wrapper.statusCode),
		).Inc()

		s.metrics.RequestDuration.WithLabelValues(

			r.Method,

			r.URL.Path,
		).Observe(duration.Seconds())

	})

}

// Background workers.

func (s *NephoranAPIServer) startBackgroundWorkers() {

	// Connection cleanup worker.

	s.wg.Add(1)

	go s.connectionCleanupWorker()

	// Metrics collection worker.

	s.wg.Add(1)

	go s.metricsWorker()

}

func (s *NephoranAPIServer) connectionCleanupWorker() {

	defer s.wg.Done()

	ticker := time.NewTicker(30 * time.Second)

	defer ticker.Stop()

	for {

		select {

		case <-s.shutdown:

			return

		case <-ticker.C:

			s.cleanupStaleConnections()

		}

	}

}

func (s *NephoranAPIServer) metricsWorker() {

	defer s.wg.Done()

	ticker := time.NewTicker(10 * time.Second)

	defer ticker.Stop()

	for {

		select {

		case <-s.shutdown:

			return

		case <-ticker.C:

			s.updateConnectionMetrics()

		}

	}

}

func (s *NephoranAPIServer) cleanupStaleConnections() {

	threshold := time.Now().Add(-s.config.WSTimeout)

	s.connectionsMutex.Lock()

	defer s.connectionsMutex.Unlock()

	// Cleanup WebSocket connections.

	for id, conn := range s.wsConnections {

		if conn.LastSeen.Before(threshold) {

			conn.Connection.Close()

			delete(s.wsConnections, id)

		}

	}

	// Cleanup SSE connections.

	for id, conn := range s.sseConnections {

		if conn.LastSeen.Before(threshold) {

			delete(s.sseConnections, id)

		}

	}

}

func (s *NephoranAPIServer) updateConnectionMetrics() {

	s.connectionsMutex.RLock()

	wsCount := len(s.wsConnections)

	sseCount := len(s.sseConnections)

	s.connectionsMutex.RUnlock()

	s.metrics.WSConnectionsActive.Set(float64(wsCount))

	s.metrics.SSEConnectionsActive.Set(float64(sseCount))

}

// Utility functions.

func (s *NephoranAPIServer) writeJSONResponse(w http.ResponseWriter, status int, data interface{}) {

	response := &APIResponse{

		Success: status >= 200 && status < 300,

		Data: data,

		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(status)

	json.NewEncoder(w).Encode(response)

}

func (s *NephoranAPIServer) writeErrorResponse(w http.ResponseWriter, status int, code, message string) {

	response := &APIResponse{

		Success: false,

		Error: &APIError{

			Code: code,

			Message: message,
		},

		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(status)

	json.NewEncoder(w).Encode(response)

}

func (s *NephoranAPIServer) parsePaginationParams(r *http.Request) PaginationParams {

	params := PaginationParams{

		Page: 1,

		PageSize: s.config.DefaultPageSize,

		Sort: "created_at",

		Order: "desc",
	}

	if page := r.URL.Query().Get("page"); page != "" {

		fmt.Sscanf(page, "%d", &params.Page)

	}

	if pageSize := r.URL.Query().Get("page_size"); pageSize != "" {

		fmt.Sscanf(pageSize, "%d", &params.PageSize)

		if params.PageSize > s.config.MaxPageSize {

			params.PageSize = s.config.MaxPageSize

		}

	}

	if sort := r.URL.Query().Get("sort"); sort != "" {

		params.Sort = sort

	}

	if order := r.URL.Query().Get("order"); order != "" {

		params.Order = order

	}

	return params

}

func (s *NephoranAPIServer) parseFilterParams(r *http.Request) FilterParams {

	params := FilterParams{

		Labels: make(map[string]string),
	}

	if status := r.URL.Query().Get("status"); status != "" {

		params.Status = status

	}

	if intentType := r.URL.Query().Get("type"); intentType != "" {

		params.Type = intentType

	}

	if priority := r.URL.Query().Get("priority"); priority != "" {

		params.Priority = priority

	}

	if component := r.URL.Query().Get("component"); component != "" {

		params.Component = component

	}

	if cluster := r.URL.Query().Get("cluster"); cluster != "" {

		params.Cluster = cluster

	}

	// Parse label filters (format: label.key=value).

	for key, values := range r.URL.Query() {

		if strings.HasPrefix(key, "label.") {

			labelKey := strings.TrimPrefix(key, "label.")

			if len(values) > 0 {

				params.Labels[labelKey] = values[0]

			}

		}

	}

	return params

}

// Default configuration.

func getDefaultServerConfig() *ServerConfig {

	return &ServerConfig{

		Address: "0.0.0.0",

		Port: 8080,

		ReadTimeout: 30 * time.Second,

		WriteTimeout: 30 * time.Second,

		IdleTimeout: 120 * time.Second,

		ShutdownTimeout: 30 * time.Second,

		TLSEnabled: false,

		EnableCORS: true,

		AllowedOrigins: []string{"*"},

		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},

		AllowedHeaders: []string{"Authorization", "Content-Type", "X-Requested-With", "X-Session-ID"},

		AllowCredentials: true,

		EnableRateLimit: true,

		RequestsPerMin: 1000,

		BurstSize: 100,

		RateLimitWindow: time.Minute,

		EnableCaching: true,

		CacheSize: 1000,

		CacheTTL: 5 * time.Minute,

		MaxWSConnections: 100,

		WSTimeout: 5 * time.Minute,

		SSETimeout: 5 * time.Minute,

		PingInterval: 30 * time.Second,

		DefaultPageSize: 20,

		MaxPageSize: 100,

		EnableMetrics: true,

		EnableProfiling: false,

		EnableHealthCheck: true,

		AuthConfigFile: config.GetEnvOrDefault("AUTH_CONFIG_FILE", ""),
	}

}

// Initialize server metrics.

func initServerMetrics() *ServerMetrics {

	return &ServerMetrics{

		RequestsTotal: prometheus.NewCounterVec(

			prometheus.CounterOpts{

				Name: "nephoran_api_requests_total",

				Help: "Total number of API requests",
			},

			[]string{"method", "path", "status"},
		),

		RequestDuration: prometheus.NewHistogramVec(

			prometheus.HistogramOpts{

				Name: "nephoran_api_request_duration_seconds",

				Help: "Duration of API requests",
			},

			[]string{"method", "path"},
		),

		WSConnectionsActive: prometheus.NewGauge(

			prometheus.GaugeOpts{

				Name: "nephoran_api_websocket_connections_active",

				Help: "Number of active WebSocket connections",
			},
		),

		SSEConnectionsActive: prometheus.NewGauge(

			prometheus.GaugeOpts{

				Name: "nephoran_api_sse_connections_active",

				Help: "Number of active SSE connections",
			},
		),

		CacheHits: prometheus.NewCounter(

			prometheus.CounterOpts{

				Name: "nephoran_api_cache_hits_total",

				Help: "Total number of cache hits",
			},
		),

		CacheMisses: prometheus.NewCounter(

			prometheus.CounterOpts{

				Name: "nephoran_api_cache_misses_total",

				Help: "Total number of cache misses",
			},
		),

		RateLimitExceeded: prometheus.NewCounter(

			prometheus.CounterOpts{

				Name: "nephoran_api_rate_limit_exceeded_total",

				Help: "Total number of rate limit exceeded errors",
			},
		),
	}

}

// responseWrapper captures response status code for metrics.

type responseWrapper struct {
	http.ResponseWriter

	statusCode int
}

// WriteHeader performs writeheader operation.

func (rw *responseWrapper) WriteHeader(code int) {

	rw.statusCode = code

	rw.ResponseWriter.WriteHeader(code)

}

// getClientIP extracts client IP from request.

func getClientIP(r *http.Request) string {

	xff := r.Header.Get("X-Forwarded-For")

	if xff != "" {

		ips := strings.Split(xff, ",")

		return strings.TrimSpace(ips[0])

	}

	xri := r.Header.Get("X-Real-IP")

	if xri != "" {

		return xri

	}

	return r.RemoteAddr

}
