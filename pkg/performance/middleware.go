package performance

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// HTTPMiddleware provides performance monitoring for HTTP requests
type HTTPMiddleware struct {
	metrics *Metrics
}

// NewHTTPMiddleware creates a new HTTP middleware for performance monitoring
func NewHTTPMiddleware(metrics *Metrics) *HTTPMiddleware {
	return &HTTPMiddleware{
		metrics: metrics,
	}
}

// Handler returns an HTTP middleware that records request metrics
func (m *HTTPMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Increment in-flight requests
		m.metrics.RequestInFlight.Inc()
		defer m.metrics.RequestInFlight.Dec()

		// Create a response writer wrapper to capture status code
		wrapper := &responseWriterWrapper{ResponseWriter: w, statusCode: 200}
		
		// Process the request
		next.ServeHTTP(wrapper, r)
		
		// Record metrics
		duration := time.Since(start)
		endpoint := m.getEndpoint(r)
		status := strconv.Itoa(wrapper.statusCode)
		
		m.metrics.RecordRequestDuration(r.Method, endpoint, status, duration)
	})
}

// responseWriterWrapper wraps http.ResponseWriter to capture status code
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// Write ensures WriteHeader is called
func (w *responseWriterWrapper) Write(data []byte) (int, error) {
	if w.statusCode == 0 {
		w.WriteHeader(200)
	}
	return w.ResponseWriter.Write(data)
}

// getEndpoint extracts the endpoint pattern from the request
func (m *HTTPMiddleware) getEndpoint(r *http.Request) string {
	// Try to get the route pattern from mux
	if route := mux.CurrentRoute(r); route != nil {
		if template, err := route.GetPathTemplate(); err == nil && template != "" {
			return template
		}
	}
	
	// Fallback to sanitized URL path
	path := r.URL.Path
	if path == "" {
		path = "/"
	}
	return path
}

// JSONMiddleware provides performance monitoring for JSON operations
type JSONMiddleware struct {
	metrics *Metrics
}

// NewJSONMiddleware creates a new JSON middleware for performance monitoring
func NewJSONMiddleware(metrics *Metrics) *JSONMiddleware {
	return &JSONMiddleware{
		metrics: metrics,
	}
}

// MarshalWithMetrics marshals JSON with performance tracking
func (m *JSONMiddleware) MarshalWithMetrics(v interface{}) ([]byte, error) {
	start := time.Now()
	data, err := JSONMarshal(v)
	duration := time.Since(start)
	
	m.metrics.RecordJSONMarshal(duration, err)
	return data, err
}

// UnmarshalWithMetrics unmarshals JSON with performance tracking
func (m *JSONMiddleware) UnmarshalWithMetrics(data []byte, v interface{}) error {
	start := time.Now()
	err := JSONUnmarshal(data, v)
	duration := time.Since(start)
	
	m.metrics.RecordJSONUnmarshal(duration, err)
	return err
}

// DatabaseMiddleware provides performance monitoring for database operations
type DatabaseMiddleware struct {
	metrics *Metrics
}

// NewDatabaseMiddleware creates a new database middleware for performance monitoring
func NewDatabaseMiddleware(metrics *Metrics) *DatabaseMiddleware {
	return &DatabaseMiddleware{
		metrics: metrics,
	}
}

// RecordQuery records database query performance
func (m *DatabaseMiddleware) RecordQuery(queryType, table string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)
	
	m.metrics.RecordDBQuery(queryType, table, duration, err)
	return err
}

// CacheMiddleware provides performance monitoring for cache operations
type CacheMiddleware struct {
	metrics *Metrics
}

// NewCacheMiddleware creates a new cache middleware for performance monitoring
func NewCacheMiddleware(metrics *Metrics) *CacheMiddleware {
	return &CacheMiddleware{
		metrics: metrics,
	}
}

// RecordHit records a cache hit
func (m *CacheMiddleware) RecordHit(cacheName string) {
	m.metrics.RecordCacheHit(cacheName)
}

// RecordMiss records a cache miss
func (m *CacheMiddleware) RecordMiss(cacheName string) {
	m.metrics.RecordCacheMiss(cacheName)
}

// RecordEviction records a cache eviction
func (m *CacheMiddleware) RecordEviction(cacheName string) {
	m.metrics.RecordCacheEviction(cacheName)
}

// UpdateSize updates the cache size
func (m *CacheMiddleware) UpdateSize(cacheName string, sizeBytes float64) {
	m.metrics.UpdateCacheSize(cacheName, sizeBytes)
}

// IntentMiddleware provides performance monitoring for intent processing
type IntentMiddleware struct {
	metrics *Metrics
}

// NewIntentMiddleware creates a new intent middleware for performance monitoring
func NewIntentMiddleware(metrics *Metrics) *IntentMiddleware {
	return &IntentMiddleware{
		metrics: metrics,
	}
}

// ProcessIntent processes an intent with performance tracking
func (m *IntentMiddleware) ProcessIntent(intentType string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)
	
	status := "success"
	if err != nil {
		status = "error"
	}
	
	m.metrics.RecordIntentProcessing(intentType, status, duration)
	return err
}

// ScalingMiddleware provides performance monitoring for scaling operations
type ScalingMiddleware struct {
	metrics *Metrics
}

// NewScalingMiddleware creates a new scaling middleware for performance monitoring
func NewScalingMiddleware(metrics *Metrics) *ScalingMiddleware {
	return &ScalingMiddleware{
		metrics: metrics,
	}
}

// RecordScaling records a scaling operation with performance tracking
func (m *ScalingMiddleware) RecordScaling(direction, resourceType string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)
	
	status := "success"
	if err != nil {
		status = "error"
	}
	
	m.metrics.RecordScalingOperation(direction, resourceType, status, duration)
	return err
}