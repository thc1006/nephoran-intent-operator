//go:build !disable_rag
// +build !disable_rag

package llm

import (
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// prometheusOnce ensures metrics are only registered once.
	prometheusOnce sync.Once

	// prometheusMetrics holds the registered Prometheus metrics.
	prometheusMetrics *PrometheusMetrics

	// testMode indicates if we're running in test mode
	testMode bool

	// testMutex protects test mode operations
	testMutex sync.RWMutex

	// metricsRegistry allows test isolation
	metricsRegistry prometheus.Registerer = prometheus.DefaultRegisterer
)

// PrometheusMetrics holds all LLM-related Prometheus metrics.

type PrometheusMetrics struct {

	// Counter metrics.

	RequestsTotal *prometheus.CounterVec

	ErrorsTotal *prometheus.CounterVec

	CacheHitsTotal *prometheus.CounterVec

	CacheMissesTotal *prometheus.CounterVec

	FallbackAttemptsTotal *prometheus.CounterVec

	RetryAttemptsTotal *prometheus.CounterVec

	// Histogram metrics.

	ProcessingDurationSeconds *prometheus.HistogramVec

	// Registration flag.
	registered bool

	// Thread-safe access protection.
	mutex sync.RWMutex

	// Test registry for isolation.
	testRegistry prometheus.Registerer
}

// NewPrometheusMetrics creates and registers LLM Prometheus metrics if enabled.
func NewPrometheusMetrics() *PrometheusMetrics {
	testMutex.RLock()
	isTest := testMode
	testMutex.RUnlock()

	if isTest {
		// In test mode, create fresh metrics for each test
		return createAndRegisterMetricsForTest()
	}

	prometheusOnce.Do(func() {
		if isMetricsEnabled() {
			prometheusMetrics = createAndRegisterMetrics()
		} else {
			prometheusMetrics = &PrometheusMetrics{registered: false}
		}
	})

	return prometheusMetrics
}

// isMetricsEnabled checks if metrics are enabled via environment variable.

func isMetricsEnabled() bool {

	return os.Getenv("METRICS_ENABLED") == "true"

}

// SetTestMode enables/disables test mode for metrics isolation.
func SetTestMode(enabled bool) {
	testMutex.Lock()
	defer testMutex.Unlock()
	testMode = enabled
	if enabled {
		// Create a new registry for test isolation
		metricsRegistry = prometheus.NewRegistry()
	} else {
		metricsRegistry = prometheus.DefaultRegisterer
	}
}

// ResetMetricsForTest resets metrics state for testing (only works in test mode).
func ResetMetricsForTest() {
	testMutex.Lock()
	defer testMutex.Unlock()
	if testMode {
		// Create new test registry
		metricsRegistry = prometheus.NewRegistry()
		prometheusMetrics = nil
		prometheusOnce = sync.Once{}
	}
}

// createAndRegisterMetricsForTest creates metrics with a test registry.
func createAndRegisterMetricsForTest() *PrometheusMetrics {
	if !isMetricsEnabled() {
		return &PrometheusMetrics{registered: false}
	}

	testMutex.RLock()
	registry := metricsRegistry
	testMutex.RUnlock()

	return createMetricsWithRegistry(registry)
}

// createAndRegisterMetrics creates and registers all LLM Prometheus metrics.
func createAndRegisterMetrics() *PrometheusMetrics {
	return createMetricsWithRegistry(prometheus.DefaultRegisterer)
}

// createMetricsWithRegistry creates metrics with the specified registry.
func createMetricsWithRegistry(registry prometheus.Registerer) *PrometheusMetrics {
	// Create a promauto factory with the specified registry
	factory := promauto.With(registry)

	pm := &PrometheusMetrics{
		// Counters with appropriate labels.
		RequestsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "nephoran_llm_requests_total",
			Help: "Total number of LLM requests by model and status",
		}, []string{"model", "status"}),

		ErrorsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "nephoran_llm_errors_total",
			Help: "Total number of LLM errors by model and error type",
		}, []string{"model", "error_type"}),

		CacheHitsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "nephoran_llm_cache_hits_total",
			Help: "Total number of LLM cache hits by model",
		}, []string{"model"}),

		CacheMissesTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "nephoran_llm_cache_misses_total",
			Help: "Total number of LLM cache misses by model",
		}, []string{"model"}),

		FallbackAttemptsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "nephoran_llm_fallback_attempts_total",
			Help: "Total number of LLM fallback attempts by original model and fallback model",
		}, []string{"original_model", "fallback_model"}),

		RetryAttemptsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "nephoran_llm_retry_attempts_total",
			Help: "Total number of LLM retry attempts by model",
		}, []string{"model"}),

		// Histogram for processing duration.
		ProcessingDurationSeconds: factory.NewHistogramVec(prometheus.HistogramOpts{
			Name: "nephoran_llm_processing_duration_seconds",
			Help: "Duration of LLM processing requests by model and status",
			Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0, 60.0, 120.0},
		}, []string{"model", "status"}),

		registered:    true,
		testRegistry: registry,
	}

	// Only register with controller-runtime metrics if not in test mode
	testMutex.RLock()
	isTest := testMode
	testMutex.RUnlock()

	if !isTest {
		// Register with controller-runtime metrics registry for production
		metrics.Registry.MustRegister(
			pm.RequestsTotal,
			pm.ErrorsTotal,
			pm.CacheHitsTotal,
			pm.CacheMissesTotal,
			pm.FallbackAttemptsTotal,
			pm.RetryAttemptsTotal,
			pm.ProcessingDurationSeconds,
		)
	}

	return pm
}

// GetPrometheusMetrics returns the singleton instance.

func GetPrometheusMetrics() *PrometheusMetrics {

	if prometheusMetrics == nil {

		return NewPrometheusMetrics()

	}

	return prometheusMetrics

}

// RecordRequest records an LLM request.

func (pm *PrometheusMetrics) RecordRequest(model, status string, duration time.Duration) {

	if !pm.isRegistered() {

		return

	}

	pm.RequestsTotal.WithLabelValues(model, status).Inc()

	pm.ProcessingDurationSeconds.WithLabelValues(model, status).Observe(duration.Seconds())

}

// RecordError records an LLM error.

func (pm *PrometheusMetrics) RecordError(model, errorType string) {

	if !pm.isRegistered() {

		return

	}

	pm.ErrorsTotal.WithLabelValues(model, errorType).Inc()

}

// RecordCacheHit records a cache hit.

func (pm *PrometheusMetrics) RecordCacheHit(model string) {

	if !pm.isRegistered() {

		return

	}

	pm.CacheHitsTotal.WithLabelValues(model).Inc()

}

// RecordCacheMiss records a cache miss.

func (pm *PrometheusMetrics) RecordCacheMiss(model string) {

	if !pm.isRegistered() {

		return

	}

	pm.CacheMissesTotal.WithLabelValues(model).Inc()

}

// RecordFallbackAttempt records a fallback attempt.

func (pm *PrometheusMetrics) RecordFallbackAttempt(originalModel, fallbackModel string) {

	if !pm.isRegistered() {

		return

	}

	pm.FallbackAttemptsTotal.WithLabelValues(originalModel, fallbackModel).Inc()

}

// RecordRetryAttempt records a retry attempt.

func (pm *PrometheusMetrics) RecordRetryAttempt(model string) {

	if !pm.isRegistered() {

		return

	}

	pm.RetryAttemptsTotal.WithLabelValues(model).Inc()

}

// isRegistered safely checks if metrics are registered.

func (pm *PrometheusMetrics) isRegistered() bool {

	if pm == nil {

		return false

	}

	pm.mutex.RLock()

	defer pm.mutex.RUnlock()

	return pm.registered

}

// MetricsIntegrator integrates Prometheus metrics with the existing MetricsCollector.

type MetricsIntegrator struct {
	collector *MetricsCollector

	prometheusMetrics *PrometheusMetrics
}

// NewMetricsIntegrator creates a new metrics integrator.

func NewMetricsIntegrator(collector *MetricsCollector) *MetricsIntegrator {

	return &MetricsIntegrator{

		collector: collector,

		prometheusMetrics: GetPrometheusMetrics(),
	}

}

// RecordLLMRequest records an LLM request to both metrics systems.

func (mi *MetricsIntegrator) RecordLLMRequest(model, status string, latency time.Duration, tokens int) {

	// Record to existing MetricsCollector.

	mi.collector.RecordLLMRequest(model, status, latency, tokens)

	// Record to Prometheus metrics.

	mi.prometheusMetrics.RecordRequest(model, status, latency)

	// Record error if status indicates failure.

	if status == "error" || status == "failed" {

		mi.prometheusMetrics.RecordError(model, "processing_error")

	}

}

// RecordCacheOperation records cache operations to both metrics systems.

func (mi *MetricsIntegrator) RecordCacheOperation(model, operation string, hit bool) {

	// Record to existing MetricsCollector.

	mi.collector.RecordCacheOperation(operation, hit)

	// Record to Prometheus metrics.

	if hit {

		mi.prometheusMetrics.RecordCacheHit(model)

	} else {

		mi.prometheusMetrics.RecordCacheMiss(model)

	}

}

// RecordFallbackAttempt records fallback attempts to both metrics systems.

func (mi *MetricsIntegrator) RecordFallbackAttempt(originalModel, fallbackModel string) {

	// Increment fallback in existing collector's client metrics.

	// This requires accessing the client metrics - we'll add a method for this.

	mi.collector.clientMetrics.mutex.Lock()

	mi.collector.clientMetrics.FallbackAttempts++

	mi.collector.clientMetrics.mutex.Unlock()

	// Record to Prometheus metrics.

	mi.prometheusMetrics.RecordFallbackAttempt(originalModel, fallbackModel)

}

// RecordRetryAttempt records retry attempts to both metrics systems.

func (mi *MetricsIntegrator) RecordRetryAttempt(model string) {

	// Increment retry in existing collector's client metrics.

	mi.collector.clientMetrics.mutex.Lock()

	mi.collector.clientMetrics.RetryAttempts++

	mi.collector.clientMetrics.mutex.Unlock()

	// Record to Prometheus metrics.

	mi.prometheusMetrics.RecordRetryAttempt(model)

}

// RecordCircuitBreakerEvent records circuit breaker events with error categorization.

func (mi *MetricsIntegrator) RecordCircuitBreakerEvent(name, event, model string) {

	// Record to existing MetricsCollector.

	mi.collector.RecordCircuitBreakerEvent(name, event)

	// Map circuit breaker events to error types for Prometheus.

	switch event {

	case "rejected":

		mi.prometheusMetrics.RecordError(model, "circuit_breaker_rejected")

	case "timeout":

		mi.prometheusMetrics.RecordError(model, "timeout")

	case "failure":

		mi.prometheusMetrics.RecordError(model, "request_failure")

	}

}

// ResetMetrics resets both metrics systems (useful for testing).

func (mi *MetricsIntegrator) ResetMetrics() {

	mi.collector.Reset()

	// Note: Prometheus metrics cannot be reset easily as they're cumulative.

	// This is by design - use different metric names/labels for testing.

}

// GetComprehensiveMetrics returns metrics from both systems.

func (mi *MetricsIntegrator) GetComprehensiveMetrics() map[string]interface{} {

	result := mi.collector.GetComprehensiveMetrics()

	result["prometheus_enabled"] = mi.prometheusMetrics.isRegistered()

	result["metrics_enabled_env"] = isMetricsEnabled()

	return result

}
