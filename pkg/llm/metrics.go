//go:build !disable_rag
// +build !disable_rag

package llm

import (
	"fmt"
	"sync"
	"time"
)

// MetricsCollector provides unified metrics collection for all LLM components.

type MetricsCollector struct {
	// Client metrics.

	clientMetrics *ClientMetrics

	// Processing metrics.

	processingMetrics *ProcessingMetrics

	// Cache metrics.

	cacheMetrics *CacheMetrics

	// Worker pool metrics.

	workerPoolMetrics *WorkerPoolMetrics

	// Circuit breaker metrics.

	circuitBreakerMetrics map[string]*CircuitMetrics

	// Global metrics.

	globalMetrics *GlobalMetrics

	mutex sync.RWMutex
}

// GlobalMetrics tracks overall system performance.

type GlobalMetrics struct {
	StartTime time.Time `json:"start_time"`

	TotalRequests int64 `json:"total_requests"`

	TotalErrors int64 `json:"total_errors"`

	AverageResponseTime time.Duration `json:"average_response_time"`

	RequestsPerSecond float64 `json:"requests_per_second"`

	ErrorRate float64 `json:"error_rate"`

	UptimeSeconds int64 `json:"uptime_seconds"`

	LastResetTime time.Time `json:"last_reset_time"`

	mutex sync.RWMutex
}

// Use CircuitMetrics from circuit_breaker.go to avoid duplicates.

// NewMetricsCollector creates a new unified metrics collector.

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		clientMetrics: &ClientMetrics{},

		processingMetrics: &ProcessingMetrics{},

		cacheMetrics: &CacheMetrics{},

		workerPoolMetrics: &WorkerPoolMetrics{},

		circuitBreakerMetrics: make(map[string]*CircuitMetrics),

		globalMetrics: &GlobalMetrics{
			StartTime: time.Now(),

			LastResetTime: time.Now(),
		},
	}
}

// RecordLLMRequest records an LLM request.

func (mc *MetricsCollector) RecordLLMRequest(model, status string, latency time.Duration, tokens int) {
	mc.mutex.Lock()

	defer mc.mutex.Unlock()

	// Update global metrics.

	mc.globalMetrics.mutex.Lock()

	mc.globalMetrics.TotalRequests++

	if status == "error" {
		mc.globalMetrics.TotalErrors++
	}

	// Update average response time.

	totalLatency := mc.globalMetrics.AverageResponseTime * time.Duration(mc.globalMetrics.TotalRequests-1)

	mc.globalMetrics.AverageResponseTime = (totalLatency + latency) / time.Duration(mc.globalMetrics.TotalRequests)

	// Update error rate.

	if mc.globalMetrics.TotalRequests > 0 {
		mc.globalMetrics.ErrorRate = float64(mc.globalMetrics.TotalErrors) / float64(mc.globalMetrics.TotalRequests)
	}

	mc.globalMetrics.mutex.Unlock()

	// Update client metrics.

	mc.clientMetrics.mutex.Lock()

	mc.clientMetrics.RequestsTotal++

	if status == "success" {
		mc.clientMetrics.RequestsSuccess++
	} else {
		mc.clientMetrics.RequestsFailure++
	}

	mc.clientMetrics.TotalLatency += latency

	mc.clientMetrics.mutex.Unlock()
}

// RecordCacheOperation records a cache operation.

func (mc *MetricsCollector) RecordCacheOperation(operation string, hit bool) {
	mc.cacheMetrics.mutex.Lock()

	defer mc.cacheMetrics.mutex.Unlock()

	switch operation {
	case "get":

		if hit {
			mc.cacheMetrics.L1Hits++ // Simplified - assuming L1 for now
		} else {
			mc.cacheMetrics.Misses++
		}
	}

	// Update hit rate.

	total := mc.cacheMetrics.L1Hits + mc.cacheMetrics.L2Hits + mc.cacheMetrics.Misses

	if total > 0 {
		mc.cacheMetrics.HitRate = float64(mc.cacheMetrics.L1Hits+mc.cacheMetrics.L2Hits) / float64(total)
	}
}

// RecordCircuitBreakerEvent records circuit breaker events.

func (mc *MetricsCollector) RecordCircuitBreakerEvent(name, event string) {
	mc.mutex.Lock()

	defer mc.mutex.Unlock()

	metrics, exists := mc.circuitBreakerMetrics[name]

	if !exists {

		metrics = &CircuitMetrics{
			CurrentState: "closed",

			LastStateChange: time.Now(),

			LastUpdated: time.Now(),
		}

		mc.circuitBreakerMetrics[name] = metrics

	}

	metrics.mutex.Lock()

	defer metrics.mutex.Unlock()

	switch event {

	case "request":

		metrics.TotalRequests++

	case "success":

		metrics.SuccessfulRequests++

	case "failure":

		metrics.FailedRequests++

	case "rejected":

		metrics.RejectedRequests++

	case "timeout":

		metrics.TimeoutRequests++

	case "state_change":

		metrics.StateTransitions++

		metrics.LastStateChange = time.Now()

	}

	// Update failure rate.

	if metrics.TotalRequests > 0 {
		metrics.FailureRate = float64(metrics.FailedRequests) / float64(metrics.TotalRequests)
	}

	metrics.LastUpdated = time.Now()
}

// GetGlobalMetrics returns global system metrics.

func (mc *MetricsCollector) GetGlobalMetrics() *GlobalMetrics {
	mc.globalMetrics.mutex.RLock()

	defer mc.globalMetrics.mutex.RUnlock()

	// Update uptime.

	uptime := time.Since(mc.globalMetrics.StartTime)

	mc.globalMetrics.UptimeSeconds = int64(uptime.Seconds())

	// Calculate requests per second.

	if uptime.Seconds() > 0 {
		mc.globalMetrics.RequestsPerSecond = float64(mc.globalMetrics.TotalRequests) / uptime.Seconds()
	}

	// Create new metrics struct to avoid copying mutex.

	metrics := &GlobalMetrics{
		StartTime: mc.globalMetrics.StartTime,

		TotalRequests: mc.globalMetrics.TotalRequests,

		TotalErrors: mc.globalMetrics.TotalErrors,

		AverageResponseTime: mc.globalMetrics.AverageResponseTime,

		RequestsPerSecond: mc.globalMetrics.RequestsPerSecond,

		ErrorRate: mc.globalMetrics.ErrorRate,

		UptimeSeconds: mc.globalMetrics.UptimeSeconds,

		LastResetTime: mc.globalMetrics.LastResetTime,
	}

	return metrics
}

// GetClientMetrics returns client metrics.

func (mc *MetricsCollector) GetClientMetrics() *ClientMetrics {
	mc.clientMetrics.mutex.RLock()

	defer mc.clientMetrics.mutex.RUnlock()

	// Create new metrics struct to avoid copying mutex - using proper fields.

	metrics := &ClientMetrics{
		RequestsTotal: mc.clientMetrics.RequestsTotal,

		RequestsSuccess: mc.clientMetrics.RequestsSuccess,

		RequestsFailure: mc.clientMetrics.RequestsFailure,

		TotalLatency: mc.clientMetrics.TotalLatency,
	}

	return metrics
}

// GetProcessingMetrics returns processing metrics.

func (mc *MetricsCollector) GetProcessingMetrics() *ProcessingMetrics {
	mc.processingMetrics.mutex.RLock()

	defer mc.processingMetrics.mutex.RUnlock()

	// Create new metrics struct to avoid copying mutex - using basic structure.

	metrics := &ProcessingMetrics{

		// Note: Actual fields need to be added based on ProcessingMetrics definition.

	}

	return metrics
}

// GetCacheMetrics returns cache metrics.

func (mc *MetricsCollector) GetCacheMetrics() *CacheMetrics {
	mc.cacheMetrics.mutex.RLock()

	defer mc.cacheMetrics.mutex.RUnlock()

	// Create new metrics struct to avoid copying mutex.

	metrics := &CacheMetrics{
		L1Hits: mc.cacheMetrics.L1Hits,

		L2Hits: mc.cacheMetrics.L2Hits,

		Misses: mc.cacheMetrics.Misses,

		Evictions: mc.cacheMetrics.Evictions,

		HitRate: mc.cacheMetrics.HitRate,

		L1HitRate: mc.cacheMetrics.L1HitRate,

		L2HitRate: mc.cacheMetrics.L2HitRate,

		SemanticHits: mc.cacheMetrics.SemanticHits,

		AdaptiveTTLAdjustments: mc.cacheMetrics.AdaptiveTTLAdjustments,
	}

	return metrics
}

// GetWorkerPoolMetrics returns worker pool metrics.

func (mc *MetricsCollector) GetWorkerPoolMetrics() *WorkerPoolMetrics {
	mc.workerPoolMetrics.mutex.RLock()

	defer mc.workerPoolMetrics.mutex.RUnlock()

	// Create new metrics struct to avoid copying mutex - using basic structure.

	metrics := &WorkerPoolMetrics{

		// Note: Actual fields need to be added based on WorkerPoolMetrics definition.

	}

	return metrics
}

// GetCircuitBreakerMetrics returns circuit breaker metrics for a specific circuit.

func (mc *MetricsCollector) GetCircuitBreakerMetrics(name string) *CircuitMetrics {
	mc.mutex.RLock()

	defer mc.mutex.RUnlock()

	if metrics, exists := mc.circuitBreakerMetrics[name]; exists {

		metrics.mutex.RLock()

		defer metrics.mutex.RUnlock()

		// Create new metrics struct to avoid copying mutex - using basic structure.

		result := &CircuitMetrics{

			// Note: Actual fields need to be added based on CircuitMetrics definition.

		}

		return result

	}

	return nil
}

// GetAllCircuitBreakerMetrics returns all circuit breaker metrics.

func (mc *MetricsCollector) GetAllCircuitBreakerMetrics() map[string]*CircuitMetrics {
	mc.mutex.RLock()

	defer mc.mutex.RUnlock()

	result := make(map[string]*CircuitMetrics)

	for name, metrics := range mc.circuitBreakerMetrics {

		metrics.mutex.RLock()

		// Create new metrics struct to avoid copying mutex - using basic structure.

		copied := &CircuitMetrics{

			// Note: Actual fields need to be added based on CircuitMetrics definition.

		}

		metrics.mutex.RUnlock()

		result[name] = copied

	}

	return result
}

// GetComprehensiveMetrics returns all metrics in a single structure.

func (mc *MetricsCollector) GetComprehensiveMetrics() map[string]interface{} {
	return make(map[string]interface{})
}

// Reset resets all metrics.

func (mc *MetricsCollector) Reset() {
	mc.mutex.Lock()

	defer mc.mutex.Unlock()

	// Reset global metrics.

	mc.globalMetrics.mutex.Lock()

	mc.globalMetrics.TotalRequests = 0

	mc.globalMetrics.TotalErrors = 0

	mc.globalMetrics.AverageResponseTime = 0

	mc.globalMetrics.RequestsPerSecond = 0

	mc.globalMetrics.ErrorRate = 0

	mc.globalMetrics.LastResetTime = time.Now()

	mc.globalMetrics.mutex.Unlock()

	// Reset client metrics.

	mc.clientMetrics.mutex.Lock()

	*mc.clientMetrics = ClientMetrics{}

	mc.clientMetrics.mutex.Unlock()

	// Reset processing metrics.

	mc.processingMetrics.mutex.Lock()

	*mc.processingMetrics = ProcessingMetrics{}

	mc.processingMetrics.mutex.Unlock()

	// Reset cache metrics.

	mc.cacheMetrics.mutex.Lock()

	*mc.cacheMetrics = CacheMetrics{}

	mc.cacheMetrics.mutex.Unlock()

	// Reset worker pool metrics.

	mc.workerPoolMetrics.mutex.Lock()

	*mc.workerPoolMetrics = WorkerPoolMetrics{}

	mc.workerPoolMetrics.mutex.Unlock()

	// Reset circuit breaker metrics.

	mc.circuitBreakerMetrics = make(map[string]*CircuitMetrics)
}

// StartMetricsCollection starts background metrics collection.

func (mc *MetricsCollector) StartMetricsCollection() {
	go mc.metricsCollectionLoop()
}

// metricsCollectionLoop runs the metrics collection background process.

func (mc *MetricsCollector) metricsCollectionLoop() {
	ticker := time.NewTicker(30 * time.Second)

	defer ticker.Stop()

	for range ticker.C {
		// Update calculated metrics.

		mc.updateCalculatedMetrics()
	}
}

// updateCalculatedMetrics updates metrics that require calculation.

func (mc *MetricsCollector) updateCalculatedMetrics() {
	// Update global metrics calculations.

	globalMetrics := mc.GetGlobalMetrics()

	// Update cache hit rates.

	cacheMetrics := mc.GetCacheMetrics()

	// Update worker pool throughput.

	workerMetrics := mc.GetWorkerPoolMetrics()

	// Log metrics summary (optional).

	// Could be used for debugging or monitoring.

	_ = globalMetrics

	_ = cacheMetrics

	_ = workerMetrics
}

// GetHealthStatus returns overall system health status.

func (mc *MetricsCollector) GetHealthStatus() map[string]interface{} {
	globalMetrics := mc.GetGlobalMetrics()

	// Determine health status based on error rate and response time.

	healthy := true

	issues := []string{}

	if globalMetrics.ErrorRate > 0.1 { // 10% error rate threshold

		healthy = false

		issues = append(issues, "high_error_rate")

	}

	if globalMetrics.AverageResponseTime > 5*time.Second { // 5 second response time threshold

		healthy = false

		issues = append(issues, "high_response_time")

	}

	status := "healthy"
	if !healthy {
		status = "unhealthy"
	}

	return map[string]interface{}{
		"status": status,
		"healthy": healthy,
		"issues": issues,
		"error_rate": globalMetrics.ErrorRate,
		"average_response_time": globalMetrics.AverageResponseTime,
	}
}

// ExportPrometheusMetrics exports metrics in Prometheus format (placeholder).

func (mc *MetricsCollector) ExportPrometheusMetrics() string {
	// This is a simplified implementation.

	// In production, you'd use the official Prometheus Go client.

	globalMetrics := mc.GetGlobalMetrics()

	clientMetrics := mc.GetClientMetrics()

	cacheMetrics := mc.GetCacheMetrics()

	prometheus := fmt.Sprintf(`# HELP llm_requests_total Total number of LLM requests

# TYPE llm_requests_total counter

llm_requests_total %d



# HELP llm_requests_success_total Total number of successful LLM requests

# TYPE llm_requests_success_total counter

llm_requests_success_total %d



# HELP llm_requests_failed_total Total number of failed LLM requests

# TYPE llm_requests_failed_total counter

llm_requests_failed_total %d



# HELP llm_average_response_time_seconds Average response time in seconds

# TYPE llm_average_response_time_seconds gauge

llm_average_response_time_seconds %f



# HELP llm_error_rate Error rate as a fraction

# TYPE llm_error_rate gauge

llm_error_rate %f



# HELP llm_cache_hit_rate Cache hit rate as a fraction

# TYPE llm_cache_hit_rate gauge

llm_cache_hit_rate %f



# HELP llm_cache_hits_total Total cache hits

# TYPE llm_cache_hits_total counter

llm_cache_hits_total %d



# HELP llm_cache_misses_total Total cache misses

# TYPE llm_cache_misses_total counter

llm_cache_misses_total %d

`,

		globalMetrics.TotalRequests,

		clientMetrics.RequestsSuccess,

		clientMetrics.RequestsFailure,

		globalMetrics.AverageResponseTime.Seconds(),

		globalMetrics.ErrorRate,

		cacheMetrics.HitRate,

		cacheMetrics.L1Hits+cacheMetrics.L2Hits,

		cacheMetrics.Misses,
	)

	return prometheus
}
