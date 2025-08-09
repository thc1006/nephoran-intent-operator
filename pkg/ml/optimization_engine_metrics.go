//go:build ml && !test

package ml

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MLOptimizationMetrics provides Prometheus metrics for the ML optimization engine
var (
	// Request metrics
	optimizationRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ml_optimization_requests_total",
			Help: "Total number of optimization requests processed",
		},
		[]string{"intent_type", "status"},
	)

	optimizationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ml_optimization_duration_seconds",
			Help:    "Duration of optimization requests in seconds",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // 0.1s to ~100s
		},
		[]string{"intent_type", "phase"},
	)

	// Data gathering metrics
	dataGatheringDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ml_data_gathering_duration_seconds",
			Help:    "Duration of historical data gathering in seconds",
			Buckets: prometheus.ExponentialBuckets(0.01, 2, 10), // 10ms to ~10s
		},
		[]string{"query_type"},
	)

	prometheusQueryErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ml_prometheus_query_errors_total",
			Help: "Total number of Prometheus query errors",
		},
		[]string{"query_type", "error_type"},
	)

	// Memory metrics
	dataPointsInMemory = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "ml_data_points_in_memory",
			Help: "Current number of data points stored in memory",
		},
	)

	memoryUsageBytes = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ml_memory_usage_bytes",
			Help: "Memory usage by component in bytes",
		},
		[]string{"component"},
	)

	// Model metrics
	modelAccuracy = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ml_model_accuracy",
			Help: "Current accuracy of ML models",
		},
		[]string{"model_type"},
	)

	modelPredictionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ml_model_prediction_duration_seconds",
			Help:    "Duration of model predictions in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
		},
		[]string{"model_type"},
	)

	modelTrainingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ml_model_training_duration_seconds",
			Help:    "Duration of model training in seconds",
			Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1s to ~1000s
		},
		[]string{"model_type"},
	)

	// Cache metrics
	cacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ml_cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"cache_type"},
	)

	cacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ml_cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"cache_type"},
	)

	cacheSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ml_cache_size_entries",
			Help: "Current number of entries in cache",
		},
		[]string{"cache_type"},
	)

	// Recommendation quality metrics
	recommendationConfidence = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "ml_recommendation_confidence_score",
			Help:    "Confidence scores of generated recommendations",
			Buckets: prometheus.LinearBuckets(0, 0.1, 11), // 0.0 to 1.0
		},
	)

	optimizationPotential = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "ml_optimization_potential_score",
			Help:    "Optimization potential scores",
			Buckets: prometheus.LinearBuckets(0, 0.1, 11), // 0.0 to 1.0
		},
	)

	// Resource utilization metrics
	cpuUtilization = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ml_resource_cpu_utilization_percent",
			Help: "CPU utilization percentage by component",
		},
		[]string{"component"},
	)

	goroutineCount = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "ml_goroutine_count",
			Help: "Current number of goroutines",
		},
	)

	// Circuit breaker metrics
	circuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ml_circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
		},
		[]string{"breaker_name"},
	)

	circuitBreakerTrips = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ml_circuit_breaker_trips_total",
			Help: "Total number of circuit breaker trips",
		},
		[]string{"breaker_name"},
	)
)

// RecordOptimizationRequest records metrics for an optimization request
func RecordOptimizationRequest(intentType, status string, duration float64) {
	optimizationRequests.WithLabelValues(intentType, status).Inc()
	optimizationDuration.WithLabelValues(intentType, "total").Observe(duration)
}

// RecordDataGathering records metrics for data gathering operations
func RecordDataGathering(queryType string, duration float64) {
	dataGatheringDuration.WithLabelValues(queryType).Observe(duration)
}

// RecordPrometheusError records Prometheus query errors
func RecordPrometheusError(queryType, errorType string) {
	prometheusQueryErrors.WithLabelValues(queryType, errorType).Inc()
}

// UpdateMemoryMetrics updates memory usage metrics
func UpdateMemoryMetrics(component string, bytes float64) {
	memoryUsageBytes.WithLabelValues(component).Set(bytes)
}

// UpdateModelMetrics updates model performance metrics
func UpdateModelMetrics(modelType string, accuracy float64) {
	modelAccuracy.WithLabelValues(modelType).Set(accuracy)
}

// RecordModelPrediction records model prediction metrics
func RecordModelPrediction(modelType string, duration float64) {
	modelPredictionDuration.WithLabelValues(modelType).Observe(duration)
}

// RecordCacheMetrics records cache performance metrics
func RecordCacheMetrics(cacheType string, hit bool) {
	if hit {
		cacheHits.WithLabelValues(cacheType).Inc()
	} else {
		cacheMisses.WithLabelValues(cacheType).Inc()
	}
}

// UpdateCacheSize updates cache size metrics
func UpdateCacheSize(cacheType string, size float64) {
	cacheSize.WithLabelValues(cacheType).Set(size)
}

// RecordRecommendationQuality records recommendation quality metrics
func RecordRecommendationQuality(confidence, potential float64) {
	recommendationConfidence.Observe(confidence)
	optimizationPotential.Observe(potential)
}

// UpdateResourceMetrics updates resource utilization metrics
func UpdateResourceMetrics(component string, cpuPercent float64, goroutines int) {
	cpuUtilization.WithLabelValues(component).Set(cpuPercent)
	goroutineCount.Set(float64(goroutines))
}

// UpdateCircuitBreakerMetrics updates circuit breaker metrics
func UpdateCircuitBreakerMetrics(breakerName string, state int, trips int64) {
	circuitBreakerState.WithLabelValues(breakerName).Set(float64(state))
	if trips > 0 {
		circuitBreakerTrips.WithLabelValues(breakerName).Add(float64(trips))
	}
}

// GrafanaDashboardJSON provides a Grafana dashboard configuration for ML optimization metrics
const GrafanaDashboardJSON = `{
  "dashboard": {
    "title": "ML Optimization Engine Performance",
    "uid": "ml-optimization-perf",
    "version": 1,
    "panels": [
      {
        "title": "Request Rate",
        "gridPos": {"h": 8, "w": 12, "x": 0, "y": 0},
        "targets": [{
          "expr": "rate(ml_optimization_requests_total[5m])"
        }]
      },
      {
        "title": "Request Duration (P95)",
        "gridPos": {"h": 8, "w": 12, "x": 12, "y": 0},
        "targets": [{
          "expr": "histogram_quantile(0.95, rate(ml_optimization_duration_seconds_bucket[5m]))"
        }]
      },
      {
        "title": "Memory Usage by Component",
        "gridPos": {"h": 8, "w": 12, "x": 0, "y": 8},
        "targets": [{
          "expr": "ml_memory_usage_bytes"
        }]
      },
      {
        "title": "Model Accuracy",
        "gridPos": {"h": 8, "w": 12, "x": 12, "y": 8},
        "targets": [{
          "expr": "ml_model_accuracy"
        }]
      },
      {
        "title": "Cache Hit Rate",
        "gridPos": {"h": 8, "w": 12, "x": 0, "y": 16},
        "targets": [{
          "expr": "rate(ml_cache_hits_total[5m]) / (rate(ml_cache_hits_total[5m]) + rate(ml_cache_misses_total[5m]))"
        }]
      },
      {
        "title": "Prometheus Query Errors",
        "gridPos": {"h": 8, "w": 12, "x": 12, "y": 16},
        "targets": [{
          "expr": "rate(ml_prometheus_query_errors_total[5m])"
        }]
      },
      {
        "title": "Data Points in Memory",
        "gridPos": {"h": 8, "w": 12, "x": 0, "y": 24},
        "targets": [{
          "expr": "ml_data_points_in_memory"
        }]
      },
      {
        "title": "Circuit Breaker State",
        "gridPos": {"h": 8, "w": 12, "x": 12, "y": 24},
        "targets": [{
          "expr": "ml_circuit_breaker_state"
        }]
      }
    ]
  }
}`

// PrometheusRecordingRules provides recording rules for common queries
const PrometheusRecordingRules = `
groups:
  - name: ml_optimization_recording
    interval: 30s
    rules:
      # Request rate
      - record: ml:optimization_request_rate
        expr: rate(ml_optimization_requests_total[5m])
      
      # Success rate
      - record: ml:optimization_success_rate
        expr: |
          rate(ml_optimization_requests_total{status="success"}[5m]) /
          rate(ml_optimization_requests_total[5m])
      
      # P95 latency
      - record: ml:optimization_duration_p95
        expr: histogram_quantile(0.95, rate(ml_optimization_duration_seconds_bucket[5m]))
      
      # Cache hit rate
      - record: ml:cache_hit_rate
        expr: |
          rate(ml_cache_hits_total[5m]) /
          (rate(ml_cache_hits_total[5m]) + rate(ml_cache_misses_total[5m]))
      
      # Model prediction rate
      - record: ml:model_prediction_rate
        expr: rate(ml_model_prediction_duration_seconds_count[5m])
      
      # Memory efficiency
      - record: ml:memory_per_datapoint
        expr: |
          sum(ml_memory_usage_bytes) /
          ml_data_points_in_memory
`

// AlertingRules provides Prometheus alerting rules
const AlertingRules = `
groups:
  - name: ml_optimization_alerts
    rules:
      - alert: MLOptimizationHighLatency
        expr: ml:optimization_duration_p95 > 5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "ML optimization latency is high"
          description: "P95 latency {{ $value }}s exceeds 5s threshold"
      
      - alert: MLOptimizationHighErrorRate
        expr: 1 - ml:optimization_success_rate > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "ML optimization error rate is high"
          description: "Error rate {{ $value }} exceeds 5% threshold"
      
      - alert: MLMemoryUsageHigh
        expr: sum(ml_memory_usage_bytes) > 5e9
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "ML engine memory usage is high"
          description: "Memory usage {{ $value | humanize }} exceeds 5GB"
      
      - alert: MLCacheHitRateLow
        expr: ml:cache_hit_rate < 0.5
        for: 10m
        labels:
          severity: info
        annotations:
          summary: "ML cache hit rate is low"
          description: "Cache hit rate {{ $value }} is below 50%"
      
      - alert: MLCircuitBreakerOpen
        expr: ml_circuit_breaker_state > 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "ML circuit breaker is open"
          description: "Circuit breaker {{ $labels.breaker_name }} is in state {{ $value }}"
`
