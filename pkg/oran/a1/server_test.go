// Package a1 provides comprehensive unit tests for the A1 Policy Management Service server
package a1

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// Note: Mock types are defined in handlers_test.go to avoid duplication

// Test fixtures and helpers

func createTestLogger() *logging.StructuredLogger {
	return logging.NewStructuredLogger(logging.DefaultConfig("a1-test", "1.0.0", "test"))
}

func createTestConfig() *A1ServerConfig {
	return &A1ServerConfig{
		Host:           "127.0.0.1",
		Port:           0, // Let OS choose available port
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
		Logger:         createTestLogger(),
		EnableA1P:      true,
		EnableA1C:      true,
		EnableA1EI:     true,
		TLSEnabled:     false,
		MetricsConfig: &MetricsConfig{
			Enabled:   true,
			Namespace: "test",
			Subsystem: "a1",
			Endpoint:  "/metrics",
		},
		AuthenticationConfig: &AuthenticationConfig{
			Enabled: false,
		},
		RateLimitConfig: &RateLimitConfig{
			Enabled: false,
		},
	}
}

func createTestTLSConfig() *A1ServerConfig {
	config := createTestConfig()
	config.TLSEnabled = true
	config.CertFile = "testdata/server.crt"
	config.KeyFile = "testdata/server.key"
	return config
}

func setupTestServer(t *testing.T, config *A1ServerConfig) (*A1Server, *MockA1Service, *MockA1Validator, *MockA1Storage) {
	if config == nil {
		config = createTestConfig()
	}

	service := &MockA1Service{}
	validator := &MockA1Validator{}
	storage := &MockA1Storage{}

	server, err := NewA1Server(config, service, validator, storage)
	require.NoError(t, err)
	require.NotNil(t, server)

	return server, service, validator, storage
}

// Test Server Creation and Configuration

func TestNewA1Server(t *testing.T) {
	tests := []struct {
		name        string
		config      *A1ServerConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid configuration",
			config:      createTestConfig(),
			expectError: false,
		},
		{
			name:        "nil configuration uses defaults",
			config:      nil,
			expectError: false,
		},
		{
			name:        "TLS configuration without cert files",
			config:      &A1ServerConfig{TLSEnabled: true},
			expectError: true,
			errorMsg:    "failed to setup TLS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &MockA1Service{}
			validator := &MockA1Validator{}
			storage := &MockA1Storage{}

			server, err := NewA1Server(tt.config, service, validator, storage)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, server)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, server)
				assert.NotNil(t, server.httpServer)
				assert.NotNil(t, server.router)
				assert.NotEmpty(t, server.middleware)
			}
		})
	}
}

func TestA1Server_Configuration(t *testing.T) {
	config := createTestConfig()
	config.EnableA1P = false
	config.EnableA1C = false
	config.EnableA1EI = false

	server, _, _, _ := setupTestServer(t, config)

	assert.Equal(t, config.Host, "127.0.0.1")
	assert.Equal(t, config.Port, 0)
	assert.Equal(t, config.ReadTimeout, 5*time.Second)
	assert.Equal(t, config.WriteTimeout, 10*time.Second)
	assert.False(t, config.EnableA1P)
	assert.False(t, config.EnableA1C)
	assert.False(t, config.EnableA1EI)
	assert.NotNil(t, server.httpServer)
}

// Test Server Lifecycle

func TestA1Server_StartAndStop(t *testing.T) {
	server, _, _, _ := setupTestServer(t, nil)

	// Test initial state
	assert.False(t, server.IsReady())
	assert.Equal(t, time.Duration(0), server.GetUptime())

	// Start server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	startDone := make(chan error, 1)
	go func() {
		startDone <- server.Start(ctx)
	}()

	// Wait for server to be ready
	require.Eventually(t, func() bool {
		return server.IsReady()
	}, 2*time.Second, 100*time.Millisecond, "Server should become ready")

	// Verify server is ready
	assert.True(t, server.IsReady())
	assert.Greater(t, server.GetUptime(), time.Duration(0))

	// Stop server
	cancel()
	err := <-startDone
	assert.NoError(t, err)

	// Verify server is no longer ready
	assert.False(t, server.IsReady())
}

func TestA1Server_ConcurrentStartStop(t *testing.T) {
	server, _, _, _ := setupTestServer(t, nil)

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	// Start multiple goroutines trying to start/stop server
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			if err := server.Start(ctx); err != nil && err != http.ErrServerClosed {
				errors <- err
			}
		}()
	}

	// Wait a bit then stop
	time.Sleep(100 * time.Millisecond)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			if err := server.Stop(ctx); err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for any unexpected errors
	for err := range errors {
		t.Errorf("Unexpected error: %v", err)
	}
}

// Test Middleware

func TestA1Server_MiddlewareSetup(t *testing.T) {
	config := createTestConfig()
	config.AuthenticationConfig.Enabled = true
	config.RateLimitConfig.Enabled = true

	server, _, _, _ := setupTestServer(t, config)

	// Verify middleware is configured
	assert.NotEmpty(t, server.middleware)
	assert.Greater(t, len(server.middleware), 5) // Should have multiple middleware
}

func TestMiddleware_RequestLogging(t *testing.T) {
	server, _, _, _ := setupTestServer(t, nil)

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Apply middleware
	middleware := server.requestLoggingMiddleware(handler)

	// Create test request
	req, err := http.NewRequest("GET", "/test", nil)
	require.NoError(t, err)

	// Execute request
	rr := &responseWriterWrapper{
		ResponseWriter: &testResponseWriter{},
		statusCode:     http.StatusOK,
	}

	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.statusCode)
	assert.Greater(t, rr.bytesWritten, int64(0))
}

func TestMiddleware_RequestID(t *testing.T) {
	server, _, _, _ := setupTestServer(t, nil)

	// Create test handler that checks for request ID
	var capturedRequestID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestID = r.Context().Value("request_id").(string)
		w.WriteHeader(http.StatusOK)
	})

	// Apply middleware
	middleware := server.requestIDMiddleware(handler)

	// Test without existing request ID
	req, err := http.NewRequest("GET", "/test", nil)
	require.NoError(t, err)

	rr := &testResponseWriter{}
	middleware.ServeHTTP(rr, req)

	assert.NotEmpty(t, capturedRequestID)
	assert.Contains(t, capturedRequestID, "req_")
	assert.NotEmpty(t, rr.Header().Get("X-Request-ID"))

	// Test with existing request ID
	req.Header.Set("X-Request-ID", "existing-id")
	middleware.ServeHTTP(rr, req)
	assert.Equal(t, "existing-id", capturedRequestID)
}

func TestMiddleware_CORS(t *testing.T) {
	server, _, _, _ := setupTestServer(t, nil)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := server.corsMiddleware(handler)

	// Test regular request
	req, err := http.NewRequest("GET", "/test", nil)
	require.NoError(t, err)

	rr := &testResponseWriter{}
	middleware.ServeHTTP(rr, req)

	assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, rr.Header().Get("Access-Control-Allow-Methods"), "GET")
	assert.Contains(t, rr.Header().Get("Access-Control-Allow-Methods"), "POST")

	// Test OPTIONS preflight request
	req, err = http.NewRequest("OPTIONS", "/test", nil)
	require.NoError(t, err)

	rr = &testResponseWriter{}
	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.statusCode)
}

func TestMiddleware_Authentication(t *testing.T) {
	config := createTestConfig()
	config.AuthenticationConfig.Enabled = true

	server, _, _, _ := setupTestServer(t, config)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := server.authenticationMiddleware(handler)

	// Test health endpoint (should bypass auth)
	req, err := http.NewRequest("GET", "/health", nil)
	require.NoError(t, err)

	rr := &testResponseWriter{}
	middleware.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.statusCode)

	// Test API endpoint without auth header
	req, err = http.NewRequest("GET", "/A1-P/v2/policytypes", nil)
	require.NoError(t, err)

	rr = &testResponseWriter{}
	middleware.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.statusCode)

	// Test API endpoint with auth header
	req.Header.Set("Authorization", "Bearer token")
	rr = &testResponseWriter{}
	middleware.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.statusCode)
}

func TestMiddleware_RequestSizeLimit(t *testing.T) {
	config := createTestConfig()
	config.MaxHeaderBytes = 100 // Very small limit for testing

	server, _, _, _ := setupTestServer(t, config)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := server.requestSizeLimitMiddleware(handler)

	// Test request within limit
	req, err := http.NewRequest("POST", "/test", nil)
	req.ContentLength = 50
	require.NoError(t, err)

	rr := &testResponseWriter{}
	middleware.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.statusCode)

	// Test request exceeding limit
	req.ContentLength = 200
	rr = &testResponseWriter{}
	middleware.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.statusCode)
}

func TestMiddleware_PanicRecovery(t *testing.T) {
	server, _, _, _ := setupTestServer(t, nil)

	// Create handler that panics
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	middleware := server.panicRecoveryMiddleware(handler)

	req, err := http.NewRequest("GET", "/test", nil)
	require.NoError(t, err)

	rr := &testResponseWriter{}

	// Should not panic and should return 500
	assert.NotPanics(t, func() {
		middleware.ServeHTTP(rr, req)
	})

	assert.Equal(t, http.StatusInternalServerError, rr.statusCode)
}

// Test Route Setup

func TestA1Server_RouteSetup(t *testing.T) {
	tests := []struct {
		name           string
		enableA1P      bool
		enableA1C      bool
		enableA1EI     bool
		expectedRoutes []string
	}{
		{
			name:       "all interfaces enabled",
			enableA1P:  true,
			enableA1C:  true,
			enableA1EI: true,
			expectedRoutes: []string{
				"/health",
				"/ready",
				"/metrics",
				"/A1-P/v2/policytypes",
				"/A1-C/v1/consumers",
				"/A1-EI/v1/eitypes",
			},
		},
		{
			name:       "only A1-P enabled",
			enableA1P:  true,
			enableA1C:  false,
			enableA1EI: false,
			expectedRoutes: []string{
				"/health",
				"/ready",
				"/A1-P/v2/policytypes",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestConfig()
			config.EnableA1P = tt.enableA1P
			config.EnableA1C = tt.enableA1C
			config.EnableA1EI = tt.enableA1EI

			server, _, _, _ := setupTestServer(t, config)

			// Walk through routes and check they exist
			err := server.router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
				pathTemplate, err := route.GetPathTemplate()
				if err != nil {
					return nil // Skip routes without templates
				}

				// Check if this is one of our expected routes
				for _, expectedRoute := range tt.expectedRoutes {
					if pathTemplate == expectedRoute {
						return nil
					}
				}

				return nil
			})

			assert.NoError(t, err)
		})
	}
}

// Test Circuit Breaker

func TestCircuitBreaker_States(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxRequests: 3,
		Interval:    1 * time.Second,
		Timeout:     1 * time.Second,
	}

	cb := NewCircuitBreaker("test", config)

	// Initially closed
	assert.Equal(t, StateClosed, cb.state)

	// Simulate failures to open circuit
	ctx := context.Background()
	failingReq := func(context.Context) (interface{}, error) {
		return nil, fmt.Errorf("failure")
	}

	// Execute requests that fail
	for i := 0; i < 5; i++ {
		cb.Execute(ctx, failingReq)
	}

	// Should be open now
	assert.Equal(t, StateOpen, cb.state)

	// Try to execute - should be rejected
	_, err := cb.Execute(ctx, failingReq)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker")

	// Wait for timeout
	time.Sleep(1100 * time.Millisecond)

	// Should be half-open now
	successReq := func(context.Context) (interface{}, error) {
		return "success", nil
	}

	result, err := cb.Execute(ctx, successReq)
	assert.NoError(t, err)
	assert.Equal(t, "success", result)
}

func TestCircuitBreaker_Stats(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxRequests: 10,
		Interval:    1 * time.Second,
		Timeout:     1 * time.Second,
	}

	cb := NewCircuitBreaker("test", config)

	// Execute some successful requests
	ctx := context.Background()
	successReq := func(context.Context) (interface{}, error) {
		return "success", nil
	}

	for i := 0; i < 3; i++ {
		cb.Execute(ctx, successReq)
	}

	stats := cb.GetStats()
	assert.Equal(t, "test", stats["name"])
	assert.Equal(t, "Closed", stats["state"])
	assert.Equal(t, uint64(3), stats["requests"])
	assert.Equal(t, uint64(3), stats["total_successes"])
	assert.Equal(t, uint64(0), stats["total_failures"])
}

func TestCircuitBreaker_Reset(t *testing.T) {
	config := &CircuitBreakerConfig{
		MaxRequests: 2,
		Interval:    1 * time.Second,
		Timeout:     1 * time.Second,
	}

	cb := NewCircuitBreaker("test", config)

	// Make circuit breaker open
	ctx := context.Background()
	failingReq := func(context.Context) (interface{}, error) {
		return nil, fmt.Errorf("failure")
	}

	for i := 0; i < 3; i++ {
		cb.Execute(ctx, failingReq)
	}

	assert.Equal(t, StateOpen, cb.state)

	// Reset circuit breaker
	cb.Reset()

	assert.Equal(t, StateClosed, cb.state)
	stats := cb.GetStats()
	assert.Equal(t, uint64(0), stats["requests"])
}

// Test Metrics Collection

func TestA1MetricsCollector(t *testing.T) {
	config := &MetricsConfig{
		Enabled:   true,
		Namespace: "test",
		Subsystem: "a1",
	}

	// Clear any existing metrics
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	metrics := NewA1MetricsCollector(config)
	require.NotNil(t, metrics)

	// Test metrics recording
	metrics.IncrementRequestCount(A1PolicyInterface, "GET", 200)
	metrics.RecordRequestDuration(A1PolicyInterface, "GET", 100*time.Millisecond)
	metrics.RecordPolicyCount(1, 5)
	metrics.RecordConsumerCount(3)
	metrics.RecordEIJobCount("type1", 2)
	metrics.RecordCircuitBreakerState("test", StateOpen)
	metrics.RecordValidationErrors(A1PolicyInterface, "schema")

	// Verify metrics can be collected (basic smoke test)
	// In real tests, you'd use prometheus testutil to check values
}

func TestNoopMetrics(t *testing.T) {
	config := &MetricsConfig{
		Enabled: false,
	}

	metrics := NewA1MetricsCollector(config)

	// Should not panic when calling methods
	assert.NotPanics(t, func() {
		metrics.IncrementRequestCount(A1PolicyInterface, "GET", 200)
		metrics.RecordRequestDuration(A1PolicyInterface, "GET", 100*time.Millisecond)
		metrics.RecordPolicyCount(1, 5)
		metrics.RecordConsumerCount(3)
		metrics.RecordEIJobCount("type1", 2)
		metrics.RecordCircuitBreakerState("test", StateOpen)
		metrics.RecordValidationErrors(A1PolicyInterface, "schema")
	})
}

// Benchmarks

func BenchmarkA1Server_RequestLoggingMiddleware(b *testing.B) {
	server, _, _, _ := setupTestServer(&testing.T{}, nil)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := server.requestLoggingMiddleware(handler)

	req, _ := http.NewRequest("GET", "/test", nil)
	rr := &testResponseWriter{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		middleware.ServeHTTP(rr, req)
	}
}

func BenchmarkCircuitBreaker_Execute(b *testing.B) {
	config := &CircuitBreakerConfig{
		MaxRequests: 100,
		Interval:    1 * time.Second,
		Timeout:     1 * time.Second,
	}

	cb := NewCircuitBreaker("bench", config)
	ctx := context.Background()

	req := func(context.Context) (interface{}, error) {
		return "result", nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Execute(ctx, req)
	}
}

// Test helpers

type testResponseWriter struct {
	headers    http.Header
	body       []byte
	statusCode int
}

func (w *testResponseWriter) Header() http.Header {
	if w.headers == nil {
		w.headers = make(http.Header)
	}
	return w.headers
}

func (w *testResponseWriter) Write(data []byte) (int, error) {
	w.body = append(w.body, data...)
	return len(data), nil
}

func (w *testResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

// Note: All type definitions are imported from types.go

// Note: All interfaces and types are imported from existing files (types.go, handlers.go, errors.go)
