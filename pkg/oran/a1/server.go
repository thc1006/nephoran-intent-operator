// Package a1 implements the main O-RAN A1 Policy Management Service HTTP server
// This module provides a production-ready HTTP/2 server with all three A1 interfaces (A1-P, A1-C, A1-EI)
package a1

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// A1Server represents the main A1 Policy Management Service server
type A1Server struct {
	config    *A1ServerConfig
	logger    *logging.StructuredLogger
	handlers  *A1Handlers
	service   A1Service
	validator A1Validator
	storage   A1Storage
	metrics   A1Metrics

	// HTTP server components
	httpServer *http.Server
	router     *mux.Router
	middleware []Middleware

	// Lifecycle management
	shutdownTimeout time.Duration
	startTime       time.Time
	isReady         bool
	mutex           sync.RWMutex

	// Circuit breaker for outbound calls
	circuitBreaker map[string]*CircuitBreaker

	// TLS configuration
	tlsConfig *tls.Config
}

// Middleware represents a middleware function
type Middleware func(http.Handler) http.Handler

// CircuitBreaker implements the circuit breaker pattern for fault tolerance
type CircuitBreaker struct {
	name          string
	maxRequests   uint32
	interval      time.Duration
	timeout       time.Duration
	readyToTrip   func(counts Counts) bool
	onStateChange func(name string, from State, to State)

	mutex  sync.Mutex
	state  State
	counts Counts
	expiry time.Time
}

// A1MetricsCollector implements A1Metrics interface with Prometheus
type A1MetricsCollector struct {
	requestCount        *prometheus.CounterVec
	requestDuration     *prometheus.HistogramVec
	policyCount         *prometheus.GaugeVec
	consumerCount       prometheus.Gauge
	eiJobCount          *prometheus.GaugeVec
	circuitBreakerState *prometheus.GaugeVec
	validationErrors    *prometheus.CounterVec
}

// NewA1Server creates a new A1 server instance
func NewA1Server(config *A1ServerConfig, service A1Service, validator A1Validator, storage A1Storage) (*A1Server, error) {
	if config == nil {
		config = DefaultA1ServerConfig()
	}

	if config.Logger == nil {
		config.Logger = logging.NewStructuredLogger(logging.DefaultConfig("a1-server", "1.0.0", "production"))
	}

	logger := config.Logger.WithComponent("a1-server")

	// Create metrics collector
	metrics := NewA1MetricsCollector(config.MetricsConfig)

	// Create handlers
	handlers := NewA1Handlers(service, validator, storage, metrics, logger, config)

	// Create router with middleware
	router := mux.NewRouter()

	server := &A1Server{
		config:          config,
		logger:          logger,
		handlers:        handlers,
		service:         service,
		validator:       validator,
		storage:         storage,
		metrics:         metrics,
		router:          router,
		middleware:      []Middleware{},
		shutdownTimeout: 30 * time.Second,
		circuitBreaker:  make(map[string]*CircuitBreaker),
	}

	// Setup TLS if enabled
	if config.TLSEnabled {
		if err := server.setupTLS(); err != nil {
			return nil, fmt.Errorf("failed to setup TLS: %w", err)
		}
	}

	// Setup middleware
	server.setupMiddleware()

	// Setup routes
	server.setupRoutes()

	// Create HTTP server
	server.httpServer = &http.Server{
		Addr:           fmt.Sprintf("%s:%d", config.Host, config.Port),
		Handler:        server.router,
		ReadTimeout:    config.ReadTimeout,
		WriteTimeout:   config.WriteTimeout,
		IdleTimeout:    config.IdleTimeout,
		MaxHeaderBytes: config.MaxHeaderBytes,
		TLSConfig:      server.tlsConfig,
		ErrorLog:       nil, // We'll handle errors through structured logging
	}

	// Enable HTTP/2
	if err := http.ConfigureTransport(server.httpServer); err != nil {
		logger.WarnWithContext("Failed to configure HTTP/2 transport", "error", err)
	}

	return server, nil
}

// Start starts the A1 server
func (s *A1Server) Start(ctx context.Context) error {
	s.mutex.Lock()
	s.startTime = time.Now()
	s.mutex.Unlock()

	s.logger.InfoWithContext("Starting A1 Policy Management Service",
		"addr", s.httpServer.Addr,
		"tls_enabled", s.config.TLSEnabled,
		"a1p_enabled", s.config.EnableA1P,
		"a1c_enabled", s.config.EnableA1C,
		"a1ei_enabled", s.config.EnableA1EI,
	)

	// Create listener
	listener, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	// Use error group for graceful shutdown
	g, gCtx := errgroup.WithContext(ctx)

	// Start HTTP server
	g.Go(func() error {
		s.mutex.Lock()
		s.isReady = true
		s.mutex.Unlock()

		s.logger.InfoWithContext("A1 server is ready and accepting connections",
			"addr", listener.Addr().String(),
		)

		if s.config.TLSEnabled {
			return s.httpServer.ServeTLS(listener, s.config.CertFile, s.config.KeyFile)
		}
		return s.httpServer.Serve(listener)
	})

	// Handle graceful shutdown
	g.Go(func() error {
		<-gCtx.Done()
		s.logger.InfoWithContext("Shutting down A1 server gracefully")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()

		return s.httpServer.Shutdown(shutdownCtx)
	})

	if err := g.Wait(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	s.logger.InfoWithContext("A1 server shutdown complete")
	return nil
}

// Stop stops the A1 server
func (s *A1Server) Stop(ctx context.Context) error {
	s.mutex.Lock()
	s.isReady = false
	s.mutex.Unlock()

	s.logger.InfoWithContext("Stopping A1 server")

	shutdownCtx, cancel := context.WithTimeout(ctx, s.shutdownTimeout)
	defer cancel()

	return s.httpServer.Shutdown(shutdownCtx)
}

// IsReady returns whether the server is ready to accept requests
func (s *A1Server) IsReady() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.isReady
}

// GetUptime returns the server uptime
func (s *A1Server) GetUptime() time.Duration {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return time.Since(s.startTime)
}

// setupTLS configures TLS for the server
func (s *A1Server) setupTLS() error {
	if s.config.CertFile == "" || s.config.KeyFile == "" {
		return fmt.Errorf("TLS enabled but cert_file or key_file not configured")
	}

	// Load certificate
	cert, err := tls.LoadX509KeyPair(s.config.CertFile, s.config.KeyFile)
	if err != nil {
		return fmt.Errorf("failed to load TLS certificate: %w", err)
	}

	// Configure TLS
	s.tlsConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
		PreferServerCipherSuites: true,
		NextProtos:               []string{"h2", "http/1.1"}, // Enable HTTP/2
	}

	return nil
}

// setupMiddleware configures middleware chain
func (s *A1Server) setupMiddleware() {
	// Request logging middleware
	s.middleware = append(s.middleware, s.requestLoggingMiddleware)

	// Request ID middleware
	s.middleware = append(s.middleware, s.requestIDMiddleware)

	// CORS middleware
	s.middleware = append(s.middleware, s.corsMiddleware)

	// Authentication middleware (if enabled)
	if s.config.AuthenticationConfig.Enabled {
		s.middleware = append(s.middleware, s.authenticationMiddleware)
	}

	// Rate limiting middleware (if enabled)
	if s.config.RateLimitConfig.Enabled {
		s.middleware = append(s.middleware, s.rateLimitMiddleware)
	}

	// Request size limiting middleware
	s.middleware = append(s.middleware, s.requestSizeLimitMiddleware)

	// Circuit breaker middleware
	s.middleware = append(s.middleware, s.circuitBreakerMiddleware)

	// Panic recovery middleware
	s.middleware = append(s.middleware, s.panicRecoveryMiddleware)

	// Error handling middleware
	s.middleware = append(s.middleware, ErrorMiddleware(DefaultErrorHandler))

	// Apply middleware to router
	handler := http.Handler(s.router)
	for i := len(s.middleware) - 1; i >= 0; i-- {
		handler = s.middleware[i](handler)
	}
	s.router.PathPrefix("/").Handler(handler)
}

// setupRoutes configures all A1 interface routes
func (s *A1Server) setupRoutes() {
	// Health and readiness endpoints
	s.router.HandleFunc("/health", s.handlers.HealthCheckHandler).Methods("GET")
	s.router.HandleFunc("/ready", s.handlers.ReadinessCheckHandler).Methods("GET")

	// Metrics endpoint (if enabled)
	if s.config.MetricsConfig.Enabled {
		s.router.Handle(s.config.MetricsConfig.Endpoint, promhttp.Handler()).Methods("GET")
	}

	// A1-P Policy Interface routes (v2)
	if s.config.EnableA1P {
		s.setupA1PolicyRoutes()
	}

	// A1-C Consumer Interface routes (v1)
	if s.config.EnableA1C {
		s.setupA1ConsumerRoutes()
	}

	// A1-EI Enrichment Information Interface routes (v1)
	if s.config.EnableA1EI {
		s.setupA1EnrichmentRoutes()
	}
}

// setupA1PolicyRoutes configures A1-P interface routes
func (s *A1Server) setupA1PolicyRoutes() {
	a1pRouter := s.router.PathPrefix("/A1-P/v2").Subrouter()

	// Policy types
	a1pRouter.HandleFunc("/policytypes", s.handlers.HandleGetPolicyTypes).Methods("GET")
	a1pRouter.HandleFunc("/policytypes/{policy_type_id:[0-9]+}", s.handlers.HandleGetPolicyType).Methods("GET")
	a1pRouter.HandleFunc("/policytypes/{policy_type_id:[0-9]+}", s.handlers.HandleCreatePolicyType).Methods("PUT")
	a1pRouter.HandleFunc("/policytypes/{policy_type_id:[0-9]+}", s.handlers.HandleDeletePolicyType).Methods("DELETE")

	// Policy instances
	a1pRouter.HandleFunc("/policytypes/{policy_type_id:[0-9]+}/policies", s.handlers.HandleGetPolicyInstances).Methods("GET")
	a1pRouter.HandleFunc("/policytypes/{policy_type_id:[0-9]+}/policies/{policy_id}", s.handlers.HandleGetPolicyInstance).Methods("GET")
	a1pRouter.HandleFunc("/policytypes/{policy_type_id:[0-9]+}/policies/{policy_id}", s.handlers.HandleCreatePolicyInstance).Methods("PUT")
	a1pRouter.HandleFunc("/policytypes/{policy_type_id:[0-9]+}/policies/{policy_id}", s.handlers.HandleDeletePolicyInstance).Methods("DELETE")

	// Policy status
	a1pRouter.HandleFunc("/policytypes/{policy_type_id:[0-9]+}/policies/{policy_id}/status", s.handlers.HandleGetPolicyStatus).Methods("GET")

	s.logger.InfoWithContext("A1-P Policy interface routes configured")
}

// setupA1ConsumerRoutes configures A1-C interface routes
func (s *A1Server) setupA1ConsumerRoutes() {
	a1cRouter := s.router.PathPrefix("/A1-C/v1").Subrouter()

	// Consumers
	a1cRouter.HandleFunc("/consumers", s.handlers.HandleListConsumers).Methods("GET")
	a1cRouter.HandleFunc("/consumers/{consumer_id}", s.handlers.HandleGetConsumer).Methods("GET")
	a1cRouter.HandleFunc("/consumers/{consumer_id}", s.handlers.HandleRegisterConsumer).Methods("POST")
	a1cRouter.HandleFunc("/consumers/{consumer_id}", s.handlers.HandleUnregisterConsumer).Methods("DELETE")

	s.logger.InfoWithContext("A1-C Consumer interface routes configured")
}

// setupA1EnrichmentRoutes configures A1-EI interface routes
func (s *A1Server) setupA1EnrichmentRoutes() {
	a1eiRouter := s.router.PathPrefix("/A1-EI/v1").Subrouter()

	// EI types
	a1eiRouter.HandleFunc("/eitypes", s.handlers.HandleGetEITypes).Methods("GET")
	a1eiRouter.HandleFunc("/eitypes/{ei_type_id}", s.handlers.HandleGetEIType).Methods("GET")
	a1eiRouter.HandleFunc("/eitypes/{ei_type_id}", s.handlers.HandleCreateEIType).Methods("PUT")
	a1eiRouter.HandleFunc("/eitypes/{ei_type_id}", s.handlers.HandleDeleteEIType).Methods("DELETE")

	// EI jobs
	a1eiRouter.HandleFunc("/eijobs", s.handlers.HandleGetEIJobs).Methods("GET")
	a1eiRouter.HandleFunc("/eijobs/{ei_job_id}", s.handlers.HandleGetEIJob).Methods("GET")
	a1eiRouter.HandleFunc("/eijobs/{ei_job_id}", s.handlers.HandleCreateEIJob).Methods("PUT")
	a1eiRouter.HandleFunc("/eijobs/{ei_job_id}", s.handlers.HandleDeleteEIJob).Methods("DELETE")
	a1eiRouter.HandleFunc("/eijobs/{ei_job_id}/status", s.handlers.HandleGetEIJobStatus).Methods("GET")

	s.logger.InfoWithContext("A1-EI Enrichment Information interface routes configured")
}

// Middleware implementations

// requestLoggingMiddleware logs HTTP requests and responses
func (s *A1Server) requestLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code and size
		wrapper := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Process request
		next.ServeHTTP(wrapper, r)

		// Log request
		duration := time.Since(start)
		s.logger.LogHTTPRequest(
			r.Method,
			r.URL.Path,
			wrapper.statusCode,
			duration,
			wrapper.bytesWritten,
		)
	})
}

// requestIDMiddleware adds request ID to context
func (s *A1Server) requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("req_%d", time.Now().UnixNano())
			w.Header().Set("X-Request-ID", requestID)
		}

		// Add to context
		ctx := context.WithValue(r.Context(), "request_id", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// corsMiddleware handles CORS headers
func (s *A1Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID, X-Correlation-ID")
		w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID, Location")
		w.Header().Set("Access-Control-Max-Age", "3600")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// authenticationMiddleware handles authentication (placeholder)
func (s *A1Server) authenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for health and metrics endpoints
		if r.URL.Path == "/health" || r.URL.Path == "/ready" || r.URL.Path == s.config.MetricsConfig.Endpoint {
			next.ServeHTTP(w, r)
			return
		}

		// Extract authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			WriteA1Error(w, NewAuthenticationRequiredError())
			return
		}

		// TODO: Implement actual authentication logic based on config
		// For now, just check that header is present

		next.ServeHTTP(w, r)
	})
}

// rateLimitMiddleware implements rate limiting (placeholder)
func (s *A1Server) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement actual rate limiting logic
		// For now, just pass through
		next.ServeHTTP(w, r)
	})
}

// requestSizeLimitMiddleware limits request body size
func (s *A1Server) requestSizeLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > int64(s.config.MaxHeaderBytes) {
			WriteA1Error(w, NewInvalidRequestError("Request body too large"))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// circuitBreakerMiddleware implements circuit breaker pattern (placeholder)
func (s *A1Server) circuitBreakerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement circuit breaker logic
		next.ServeHTTP(w, r)
	})
}

// panicRecoveryMiddleware recovers from panics
func (s *A1Server) panicRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.logger.ErrorWithContext("Request panic recovered",
					fmt.Errorf("panic: %v", rec),
					"method", r.Method,
					"path", r.URL.Path,
					"remote_addr", r.RemoteAddr,
				)

				WriteA1Error(w, NewInternalServerError("Internal server error", nil))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// responseWriterWrapper wraps http.ResponseWriter to capture response metrics
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

func (w *responseWriterWrapper) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriterWrapper) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	w.bytesWritten += int64(n)
	return n, err
}

// NewA1MetricsCollector creates a new metrics collector
func NewA1MetricsCollector(config *MetricsConfig) A1Metrics {
	if config == nil || !config.Enabled {
		return &noopMetrics{}
	}

	namespace := config.Namespace
	subsystem := config.Subsystem

	collector := &A1MetricsCollector{
		requestCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "requests_total",
				Help:      "Total number of A1 requests",
			},
			[]string{"interface", "method", "status_code"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "request_duration_seconds",
				Help:      "A1 request duration",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"interface", "method"},
		),
		policyCount: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "policy_instances_total",
				Help:      "Total number of policy instances",
			},
			[]string{"policy_type_id"},
		),
		consumerCount: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "consumers_total",
				Help:      "Total number of registered consumers",
			},
		),
		eiJobCount: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "ei_jobs_total",
				Help:      "Total number of EI jobs",
			},
			[]string{"ei_type_id"},
		),
		circuitBreakerState: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "circuit_breaker_state",
				Help:      "Circuit breaker state (0=closed, 1=half-open, 2=open)",
			},
			[]string{"name"},
		),
		validationErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "validation_errors_total",
				Help:      "Total number of validation errors",
			},
			[]string{"interface", "error_type"},
		),
	}

	// Register metrics
	prometheus.MustRegister(
		collector.requestCount,
		collector.requestDuration,
		collector.policyCount,
		collector.consumerCount,
		collector.eiJobCount,
		collector.circuitBreakerState,
		collector.validationErrors,
	)

	return collector
}

// A1MetricsCollector method implementations

func (m *A1MetricsCollector) IncrementRequestCount(interface_ A1Interface, method string, statusCode int) {
	m.requestCount.WithLabelValues(string(interface_), method, fmt.Sprintf("%d", statusCode)).Inc()
}

func (m *A1MetricsCollector) RecordRequestDuration(interface_ A1Interface, method string, duration time.Duration) {
	m.requestDuration.WithLabelValues(string(interface_), method).Observe(duration.Seconds())
}

func (m *A1MetricsCollector) RecordPolicyCount(policyTypeID int, instanceCount int) {
	m.policyCount.WithLabelValues(fmt.Sprintf("%d", policyTypeID)).Set(float64(instanceCount))
}

func (m *A1MetricsCollector) RecordConsumerCount(consumerCount int) {
	m.consumerCount.Set(float64(consumerCount))
}

func (m *A1MetricsCollector) RecordEIJobCount(eiTypeID string, jobCount int) {
	m.eiJobCount.WithLabelValues(eiTypeID).Set(float64(jobCount))
}

func (m *A1MetricsCollector) RecordCircuitBreakerState(name string, state State) {
	m.circuitBreakerState.WithLabelValues(name).Set(float64(state))
}

func (m *A1MetricsCollector) RecordValidationErrors(interface_ A1Interface, errorType string) {
	m.validationErrors.WithLabelValues(string(interface_), errorType).Inc()
}

// noopMetrics provides a no-op implementation of A1Metrics
type noopMetrics struct{}

func (m *noopMetrics) IncrementRequestCount(A1Interface, string, int)           {}
func (m *noopMetrics) RecordRequestDuration(A1Interface, string, time.Duration) {}
func (m *noopMetrics) RecordPolicyCount(int, int)                               {}
func (m *noopMetrics) RecordConsumerCount(int)                                  {}
func (m *noopMetrics) RecordEIJobCount(string, int)                             {}
func (m *noopMetrics) RecordCircuitBreakerState(string, State)                  {}
func (m *noopMetrics) RecordValidationErrors(A1Interface, string)               {}

// Circuit Breaker implementation

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, config *CircuitBreakerConfig) *CircuitBreaker {
	if config.ReadyToTrip == nil {
		config.ReadyToTrip = func(counts Counts) bool {
			return counts.Requests >= config.MaxRequests &&
				float64(counts.TotalFailures)/float64(counts.Requests) >= 0.5
		}
	}

	return &CircuitBreaker{
		name:          name,
		maxRequests:   config.MaxRequests,
		interval:      config.Interval,
		timeout:       config.Timeout,
		readyToTrip:   config.ReadyToTrip,
		onStateChange: config.OnStateChange,
		state:         StateClosed,
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, req func(context.Context) (interface{}, error)) (interface{}, error) {
	generation, err := cb.beforeRequest()
	if err != nil {
		return nil, err
	}

	defer func() {
		e := recover()
		if e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()

	result, err := req(ctx)
	cb.afterRequest(generation, err == nil)
	return result, err
}

// beforeRequest checks if request should be allowed
func (cb *CircuitBreaker) beforeRequest() (uint64, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)

	if state == StateOpen {
		return generation, fmt.Errorf("circuit breaker %s is open", cb.name)
	} else if state == StateHalfOpen && cb.counts.Requests >= cb.maxRequests {
		return generation, fmt.Errorf("circuit breaker %s is half-open with too many requests", cb.name)
	}

	cb.counts.onRequest()
	return generation, nil
}

// afterRequest updates circuit breaker state after request completion
func (cb *CircuitBreaker) afterRequest(before uint64, success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)
	if generation != before {
		return
	}

	if success {
		cb.onSuccess(state, now)
	} else {
		cb.onFailure(state, now)
	}
}

// currentState returns the current state and generation
func (cb *CircuitBreaker) currentState(now time.Time) (State, uint64) {
	switch cb.state {
	case StateClosed:
		if !cb.expiry.IsZero() && cb.expiry.Before(now) {
			cb.toNewGeneration(now)
		}
	case StateOpen:
		if cb.expiry.Before(now) {
			cb.setState(StateHalfOpen, now)
		}
	}
	return cb.state, cb.counts.Requests
}

// onSuccess handles successful request
func (cb *CircuitBreaker) onSuccess(state State, now time.Time) {
	switch state {
	case StateClosed:
		cb.counts.onSuccess()
	case StateHalfOpen:
		cb.counts.onSuccess()
		if cb.counts.ConsecutiveSuccesses >= cb.maxRequests {
			cb.setState(StateClosed, now)
		}
	}
}

// onFailure handles failed request
func (cb *CircuitBreaker) onFailure(state State, now time.Time) {
	switch state {
	case StateClosed:
		cb.counts.onFailure()
		if cb.readyToTrip(cb.counts) {
			cb.setState(StateOpen, now)
		}
	case StateHalfOpen:
		cb.setState(StateOpen, now)
	}
}

// setState changes the circuit breaker state
func (cb *CircuitBreaker) setState(state State, now time.Time) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state

	cb.toNewGeneration(now)

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, prev, state)
	}
}

// toNewGeneration resets counters for new generation
func (cb *CircuitBreaker) toNewGeneration(now time.Time) {
	cb.counts = Counts{}

	var zero time.Time
	switch cb.state {
	case StateClosed:
		if cb.interval == 0 {
			cb.expiry = zero
		} else {
			cb.expiry = now.Add(cb.interval)
		}
	case StateOpen:
		cb.expiry = now.Add(cb.timeout)
	default: // StateHalfOpen
		cb.expiry = zero
	}
}

// Counts method implementations

func (c *Counts) onRequest() {
	c.Requests++
}

func (c *Counts) onSuccess() {
	c.TotalSuccesses++
	c.ConsecutiveSuccesses++
	c.ConsecutiveFailures = 0
}

func (c *Counts) onFailure() {
	c.TotalFailures++
	c.ConsecutiveFailures++
	c.ConsecutiveSuccesses = 0
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	return map[string]interface{}{
		"name":                  cb.name,
		"state":                 cb.state.String(),
		"requests":              cb.counts.Requests,
		"total_successes":       cb.counts.TotalSuccesses,
		"total_failures":        cb.counts.TotalFailures,
		"consecutive_successes": cb.counts.ConsecutiveSuccesses,
		"consecutive_failures":  cb.counts.ConsecutiveFailures,
	}
}

// Reset manually resets the circuit breaker
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.toNewGeneration(time.Now())
	cb.state = StateClosed
}
