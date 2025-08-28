// Package llm provides missing type definitions
package llm

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/thc1006/nephoran-intent-operator/pkg/rag"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
	types "github.com/thc1006/nephoran-intent-operator/pkg/shared/types"
	"log/slog"
)

// BatchProcessor interface for batch processing capabilities
type BatchProcessor interface {
	ProcessRequest(ctx context.Context, intent, intentType, modelName string, priority Priority) (*BatchResult, error)
	GetStats() types.BatchProcessorStats
	Close() error
}

// BatchProcessorStats holds batch processor statistics
type BatchProcessorStats struct {
	ProcessedBatches    int64         `json:"processed_batches"`
	TotalRequests       int64         `json:"total_requests"`
	FailedRequests      int64         `json:"failed_requests"`
	AvgBatchSize        float64       `json:"avg_batch_size"`
	AvgProcessingTime   time.Duration `json:"avg_processing_time"`
	CurrentQueueDepth   int           `json:"current_queue_depth"`
	ActiveBatches       int           `json:"active_batches"`
	LastProcessedTime   time.Time     `json:"last_processed_time"`
}

// Note: BatchProcessorConfig is defined in types.go

// MetricsIntegrator integrates various metrics collectors
type MetricsIntegrator struct {
	collector       *MetricsCollector
	prometheusReg   prometheus.Registerer
	customMetrics   map[string]prometheus.Collector
	mutex           sync.RWMutex
	logger          *slog.Logger
	updateInterval  time.Duration
	stopChan        chan struct{}
	aggregatedStats *AggregatedStats
}

// AggregatedStats holds aggregated statistics
type AggregatedStats struct {
	TotalRequests     int64         `json:"total_requests"`
	SuccessfulRequests int64        `json:"successful_requests"`
	FailedRequests    int64         `json:"failed_requests"`
	AvgResponseTime   time.Duration `json:"avg_response_time"`
	P95ResponseTime   time.Duration `json:"p95_response_time"`
	P99ResponseTime   time.Duration `json:"p99_response_time"`
	TotalTokens       int64         `json:"total_tokens"`
	ErrorRate         float64       `json:"error_rate"`
	LastUpdated       time.Time     `json:"last_updated"`
}

// NewMetricsIntegrator creates a new metrics integrator
func NewMetricsIntegrator(collector *MetricsCollector) *MetricsIntegrator {
	mi := &MetricsIntegrator{
		collector:       collector,
		customMetrics:   make(map[string]prometheus.Collector),
		logger:          slog.Default().With("component", "metrics-integrator"),
		updateInterval:  30 * time.Second,
		stopChan:        make(chan struct{}),
		aggregatedStats: &AggregatedStats{},
	}
	
	// Start background metrics aggregation
	go mi.aggregateMetrics()
	
	return mi
}

// aggregateMetrics runs in background to aggregate metrics
func (mi *MetricsIntegrator) aggregateMetrics() {
	ticker := time.NewTicker(mi.updateInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			mi.updateAggregatedStats()
		case <-mi.stopChan:
			return
		}
	}
}

// updateAggregatedStats updates the aggregated statistics
func (mi *MetricsIntegrator) updateAggregatedStats() {
	mi.mutex.Lock()
	defer mi.mutex.Unlock()
	
	// Update aggregated stats from collector
	if mi.collector != nil {
		stats := mi.collector.GetStats()
		mi.aggregatedStats.TotalRequests = stats.TotalRequests
		mi.aggregatedStats.SuccessfulRequests = stats.SuccessfulRequests
		mi.aggregatedStats.FailedRequests = stats.FailedRequests
		mi.aggregatedStats.TotalTokens = stats.TotalTokens
		mi.aggregatedStats.LastUpdated = time.Now()
		
		// Calculate error rate
		if mi.aggregatedStats.TotalRequests > 0 {
			mi.aggregatedStats.ErrorRate = float64(mi.aggregatedStats.FailedRequests) / float64(mi.aggregatedStats.TotalRequests)
		}
	}
}

// RegisterCustomMetric registers a custom Prometheus metric
func (mi *MetricsIntegrator) RegisterCustomMetric(name string, metric prometheus.Collector) error {
	mi.mutex.Lock()
	defer mi.mutex.Unlock()
	
	if _, exists := mi.customMetrics[name]; exists {
		mi.logger.Warn("Metric already registered", "name", name)
		return nil
	}
	
	mi.customMetrics[name] = metric
	if mi.prometheusReg != nil {
		return mi.prometheusReg.Register(metric)
	}
	return nil
}

// GetStats returns current statistics
func (mi *MetricsIntegrator) GetStats() *AggregatedStats {
	mi.mutex.RLock()
	defer mi.mutex.RUnlock()
	
	// Return a copy to avoid data races
	return &AggregatedStats{
		TotalRequests:      mi.aggregatedStats.TotalRequests,
		SuccessfulRequests: mi.aggregatedStats.SuccessfulRequests,
		FailedRequests:     mi.aggregatedStats.FailedRequests,
		AvgResponseTime:    mi.aggregatedStats.AvgResponseTime,
		P95ResponseTime:    mi.aggregatedStats.P95ResponseTime,
		P99ResponseTime:    mi.aggregatedStats.P99ResponseTime,
		TotalTokens:        mi.aggregatedStats.TotalTokens,
		ErrorRate:          mi.aggregatedStats.ErrorRate,
		LastUpdated:        mi.aggregatedStats.LastUpdated,
	}
}

// RecordRequest records a request metric
func (mi *MetricsIntegrator) RecordRequest(duration time.Duration, success bool, tokens int) {
	if mi.collector != nil {
		mi.collector.RecordRequest("default", duration, success, tokens)
	}
	
	mi.mutex.Lock()
	defer mi.mutex.Unlock()
	
	mi.aggregatedStats.TotalRequests++
	if success {
		mi.aggregatedStats.SuccessfulRequests++
	} else {
		mi.aggregatedStats.FailedRequests++
	}
	mi.aggregatedStats.TotalTokens += int64(tokens)
}

// Close gracefully shuts down the metrics integrator
func (mi *MetricsIntegrator) Close() error {
	close(mi.stopChan)
	return nil
}

// RecordCircuitBreakerEvent records a circuit breaker event
func (mi *MetricsIntegrator) RecordCircuitBreakerEvent(event string, oldState string, newState string) {
	// Record the event to metrics
	mi.logger.Info("Circuit breaker event", "event", event, "old_state", oldState, "new_state", newState)
}

// PrometheusMetrics returns Prometheus metrics (stub for now)
type PrometheusMetricsWrapper struct{}

func (p *PrometheusMetricsWrapper) RecordError(err error) {
	// Record error metric
}

func (mi *MetricsIntegrator) prometheusMetrics() *PrometheusMetricsWrapper {
	return &PrometheusMetricsWrapper{}
}

// RecordLLMRequest records an LLM request
func (mi *MetricsIntegrator) RecordLLMRequest(model string, duration time.Duration, tokens int, success bool) {
	mi.RecordRequest(duration, success, tokens)
}

// RecordCacheOperation records a cache operation
func (mi *MetricsIntegrator) RecordCacheOperation(operation string, hit bool, duration time.Duration) {
	mi.mutex.Lock()
	defer mi.mutex.Unlock()
	if hit {
		mi.aggregatedStats.TotalRequests++ // Using as a proxy for cache hits for now
	}
}

// RecordRetryAttempt records a retry attempt
func (mi *MetricsIntegrator) RecordRetryAttempt(attempt int, success bool, duration time.Duration) {
	// Record retry metrics
	mi.logger.Debug("Retry attempt", "attempt", attempt, "success", success, "duration", duration)
}

// Note: ContextBuilder is defined in clean_stubs.go as ContextBuilderStub
// We'll create an alias for compatibility
type ContextBuilder = ContextBuilderStub