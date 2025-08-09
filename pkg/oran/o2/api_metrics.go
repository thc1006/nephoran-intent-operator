package o2

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// newAPIMetrics creates and registers Prometheus metrics for the O2 API server
func newAPIMetrics(registry *prometheus.Registry) *APIMetrics {
	metrics := &APIMetrics{
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "nephoran",
				Subsystem: "o2_ims_api",
				Name:      "requests_total",
				Help:      "Total number of HTTP requests processed by the O2 IMS API",
			},
			[]string{"method", "endpoint", "status_code"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "nephoran",
				Subsystem: "o2_ims_api",
				Name:      "request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),
		activeConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "nephoran",
				Subsystem: "o2_ims_api",
				Name:      "active_connections",
				Help:      "Number of active HTTP connections",
			},
		),
		resourceOperations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "nephoran",
				Subsystem: "o2_ims_api",
				Name:      "resource_operations_total",
				Help:      "Total number of resource operations performed",
			},
			[]string{"operation", "resource_type", "provider", "status"},
		),
		errorRate: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "nephoran",
				Subsystem: "o2_ims_api",
				Name:      "errors_total",
				Help:      "Total number of errors by type",
			},
			[]string{"error_type", "endpoint"},
		),
		responseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "nephoran",
				Subsystem: "o2_ims_api",
				Name:      "response_size_bytes",
				Help:      "HTTP response size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 6), // 100B to 100MB
			},
			[]string{"method", "endpoint"},
		),
	}

	// Register metrics with the registry
	registry.MustRegister(
		metrics.requestsTotal,
		metrics.requestDuration,
		metrics.activeConnections,
		metrics.resourceOperations,
		metrics.errorRate,
		metrics.responseSize,
	)

	return metrics
}

// RecordRequest records metrics for an HTTP request
func (m *APIMetrics) RecordRequest(method, endpoint string, statusCode int, duration time.Duration, responseSize int64) {
	statusStr := strconv.Itoa(statusCode)
	m.requestsTotal.WithLabelValues(method, endpoint, statusStr).Inc()
	m.requestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
	m.responseSize.WithLabelValues(method, endpoint).Observe(float64(responseSize))
}

// RecordResourceOperation records metrics for resource operations
func (m *APIMetrics) RecordResourceOperation(operation, resourceType, provider, status string) {
	m.resourceOperations.WithLabelValues(operation, resourceType, provider, status).Inc()
}

// RecordError records error metrics
func (m *APIMetrics) RecordError(errorType, endpoint string) {
	m.errorRate.WithLabelValues(errorType, endpoint).Inc()
}

// SetActiveConnections updates the active connections gauge
func (m *APIMetrics) SetActiveConnections(count float64) {
	m.activeConnections.Set(count)
}
