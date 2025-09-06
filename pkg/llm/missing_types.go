// Package llm provides missing type definitions
package llm

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	types "github.com/thc1006/nephoran-intent-operator/pkg/shared/types"
)

// Config represents the configuration for LLM clients.
type Config struct {
	Provider    string        `json:"provider"`
	Model       string        `json:"model"`
	APIKey      string        `json:"api_key"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
	Timeout     time.Duration `json:"timeout"`
}

// EnhancedClient is an alias for EnhancedPerformanceClient for backward compatibility.
type EnhancedClient = EnhancedPerformanceClient

// NewEnhancedClient creates a new enhanced client from a simple config.
func NewEnhancedClient(config *Config) (*EnhancedClient, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Convert simple config to enhanced config
	enhancedConfig := &EnhancedClientConfig{
		BaseConfig: ClientConfig{
			APIKey:      config.APIKey,
			ModelName:   config.Model,
			MaxTokens:   config.MaxTokens,
			BackendType: config.Provider,
			Timeout:     config.Timeout,
		},
		PerformanceConfig:   getDefaultPerformanceConfig(),
		RetryConfig:         getDefaultRetryConfig(),
		BatchConfig:         getDefaultBatchConfig(),
		CircuitBreakerConfig: *getDefaultCircuitBreakerConfig(),
	}

	return NewEnhancedPerformanceClient(enhancedConfig)
}


// getDefaultRetryConfig returns a default retry configuration.
func getDefaultRetryConfig() RetryEngineConfig {
	return RetryEngineConfig{
		DefaultStrategy: "exponential",
		MaxRetries:      3,
		BaseDelay:       100 * time.Millisecond,
		MaxDelay:        5 * time.Second,
		JitterEnabled:   true,
	}
}

// getDefaultBatchConfig returns a default batch configuration.
func getDefaultBatchConfig() BatchConfig {
	return BatchConfig{
		MaxBatchSize:      10,
		BatchTimeout:      1 * time.Second,
		ConcurrentBatches: 5,
	}
}


// Task types
const (
	TaskTypeRAGProcessing   = "rag_processing"
	TaskTypeBatchProcessing = "batch_processing"
)

// MissingBatchProcessor interface for batch processing capabilities
type MissingBatchProcessor interface {
	ProcessRequest(ctx context.Context, intent, intentType, modelName string, priority Priority) (*BatchResult, error)
	GetStats() types.BatchProcessorStats
	Close() error
}

// MissingBatchProcessorStats holds batch processor statistics
type MissingBatchProcessorStats struct {
	ProcessedBatches  int64         `json:"processed_batches"`
	TotalRequests     int64         `json:"total_requests"`
	FailedRequests    int64         `json:"failed_requests"`
	AvgBatchSize      float64       `json:"avg_batch_size"`
	AvgProcessingTime time.Duration `json:"avg_processing_time"`
	CurrentQueueDepth int           `json:"current_queue_depth"`
	ActiveBatches     int           `json:"active_batches"`
	LastProcessedTime time.Time     `json:"last_processed_time"`
}

// Note: BatchProcessorConfig is defined in types.go

// MissingMetricsIntegrator integrates various metrics collectors
type MissingMetricsIntegrator struct {
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
	TotalRequests      int64         `json:"total_requests"`
	SuccessfulRequests int64         `json:"successful_requests"`
	FailedRequests     int64         `json:"failed_requests"`
	AvgResponseTime    time.Duration `json:"avg_response_time"`
	P95ResponseTime    time.Duration `json:"p95_response_time"`
	P99ResponseTime    time.Duration `json:"p99_response_time"`
	TotalTokens        int64         `json:"total_tokens"`
	ErrorRate          float64       `json:"error_rate"`
	LastUpdated        time.Time     `json:"last_updated"`
}

// NewMissingMetricsIntegrator creates a new metrics integrator
func NewMissingMetricsIntegrator(collector *MetricsCollector) *MissingMetricsIntegrator {
	mi := &MissingMetricsIntegrator{
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
func (mi *MissingMetricsIntegrator) aggregateMetrics() {
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
func (mi *MissingMetricsIntegrator) updateAggregatedStats() {
	mi.mutex.Lock()
	defer mi.mutex.Unlock()

	// Update aggregated stats from collector
	if mi.collector != nil {
		// For now, just update basic stats since collector doesn't have GetStats
		mi.aggregatedStats.LastUpdated = time.Now()

		// Calculate error rate
		if mi.aggregatedStats.TotalRequests > 0 {
			mi.aggregatedStats.ErrorRate = float64(mi.aggregatedStats.FailedRequests) / float64(mi.aggregatedStats.TotalRequests)
		}
	}
}

// RegisterCustomMetric registers a custom Prometheus metric
func (mi *MissingMetricsIntegrator) RegisterCustomMetric(name string, metric prometheus.Collector) error {
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
func (mi *MissingMetricsIntegrator) GetStats() *AggregatedStats {
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
func (mi *MissingMetricsIntegrator) RecordRequest(duration time.Duration, success bool, tokens int) {
	mi.mutex.Lock()
	defer mi.mutex.Unlock()

	mi.aggregatedStats.TotalRequests++
	if success {
		mi.aggregatedStats.SuccessfulRequests++
	} else {
		mi.aggregatedStats.FailedRequests++
	}
	mi.aggregatedStats.TotalTokens += int64(tokens)

	// Update average response time (simple rolling average)
	if mi.aggregatedStats.TotalRequests > 0 {
		prevAvg := mi.aggregatedStats.AvgResponseTime
		mi.aggregatedStats.AvgResponseTime = (prevAvg*time.Duration(mi.aggregatedStats.TotalRequests-1) + duration) / time.Duration(mi.aggregatedStats.TotalRequests)
	}
}

// Close gracefully shuts down the metrics integrator
func (mi *MissingMetricsIntegrator) Close() error {
	close(mi.stopChan)
	return nil
}

// RecordCircuitBreakerEvent records a circuit breaker event
func (mi *MissingMetricsIntegrator) RecordCircuitBreakerEvent(event string, oldState string, newState string) {
	// Record the event to metrics
	mi.logger.Info("Circuit breaker event", "event", event, "old_state", oldState, "new_state", newState)
}

// PrometheusMetrics returns Prometheus metrics (stub for now)
type PrometheusMetricsWrapper struct{}

func (p *PrometheusMetricsWrapper) RecordError(model, errorType string) {
	// Record error metric
}

func (mi *MissingMetricsIntegrator) prometheusMetrics() *PrometheusMetricsWrapper {
	return &PrometheusMetricsWrapper{}
}

// RecordLLMRequest records an LLM request
func (mi *MissingMetricsIntegrator) RecordLLMRequest(model string, duration time.Duration, tokens int, success bool) {
	mi.RecordRequest(duration, success, tokens)
}

// RecordCacheOperation records a cache operation
func (mi *MissingMetricsIntegrator) RecordCacheOperation(operation string, hit bool, duration time.Duration) {
	mi.mutex.Lock()
	defer mi.mutex.Unlock()
	if hit {
		mi.aggregatedStats.TotalRequests++ // Using as a proxy for cache hits for now
	}
}

// RecordRetryAttempt records a retry attempt (overloaded signatures for compatibility)
func (mi *MissingMetricsIntegrator) RecordRetryAttempt(args ...interface{}) {
	// Handle different call signatures
	if len(args) == 1 {
		// Called with just model string
		model := args[0].(string)
		mi.logger.Debug("Retry attempt", "model", model)
	} else if len(args) == 3 {
		// Called with attempt, success, duration
		attempt := args[0].(int)
		success := args[1].(bool)
		duration := args[2].(time.Duration)
		mi.logger.Debug("Retry attempt", "attempt", attempt, "success", success, "duration", duration)
	}
}

// GetComprehensiveMetrics returns comprehensive metrics
func (mi *MissingMetricsIntegrator) GetComprehensiveMetrics() map[string]interface{} {
	mi.mutex.RLock()
	defer mi.mutex.RUnlock()

	return make(map[string]interface{})
}

// RecordFallbackAttempt records a fallback attempt (overloaded signatures for compatibility)
func (mi *MissingMetricsIntegrator) RecordFallbackAttempt(args ...interface{}) {
	// Handle different call signatures
	if len(args) == 2 {
		// Called with originalModel, fallbackModel
		if originalModel, ok := args[0].(string); ok {
			if fallbackModel, ok := args[1].(string); ok {
				mi.logger.Debug("Fallback attempt", "original_model", originalModel, "fallback_model", fallbackModel)
			}
		}
		// Could also be provider, success bool
		if provider, ok := args[0].(string); ok {
			if success, ok := args[1].(bool); ok {
				mi.logger.Debug("Fallback attempt", "provider", provider, "success", success)
			}
		}
	}
}

// Note: ContextBuilder is defined in clean_stubs.go as ContextBuilderStub
// We'll create an alias for compatibility
type MissingContextBuilder = ContextBuilder

// Document represents a document in the knowledge base for RAG operations
type Document struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title,omitempty"`
	Content     string                 `json:"content"`
	Source      string                 `json:"source,omitempty"`
	Metadata    string                 `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at,omitempty"`
	UpdatedAt   time.Time              `json:"updated_at,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Category    string                 `json:"category,omitempty"`
	Priority    int                    `json:"priority,omitempty"`
	Embedding   []float32              `json:"embedding,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
}

// ContextBuilderConfig holds configuration for the context builder
type ContextBuilderConfig struct {
	MaxContextTokens   int     `json:"max_context_tokens"`
	DiversityThreshold float64 `json:"diversity_threshold"`
	QualityThreshold   float64 `json:"quality_threshold"`
	RelevanceWeight    float64 `json:"relevance_weight"`
	AuthorityWeight    float64 `json:"authority_weight"`
	FreshnessWeight    float64 `json:"freshness_weight"`
	MaxDocuments       int     `json:"max_documents"`
}

// RelevanceScore is defined in relevance_scorer.go - no need to redefine

// BuiltContext represents a context built from documents
type BuiltContext struct {
	Context       string     `json:"context"`
	UsedDocuments []Document `json:"used_documents"`
	QualityScore  float64    `json:"quality_score"`
	TokenCount    int        `json:"token_count"`
	BuildTime     time.Duration `json:"build_time"`
}

// NewContextBuilder creates a new context builder with the given configuration
func NewContextBuilder(config *ContextBuilderConfig) *ContextBuilder {
	return &ContextBuilder{
		Config: config,
	}
}
