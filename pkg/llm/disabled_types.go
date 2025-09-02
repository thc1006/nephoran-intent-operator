//go:build disable_rag

package llm

import (
	"context"
	"time"
)

// Stub types for when RAG is disabled.

// ResponseCache provides a stub response cache implementation
type ResponseCache struct{}

// CacheMetrics provides stub cache metrics
type CacheMetrics struct {
	Hits   int64
	Misses int64
	Size   int64
}

// CacheState represents cache state
type CacheState struct {
	Enabled bool
	Size    int
}

// BatchProcessorConfig provides configuration for batch processing (stub)
type BatchProcessorConfig struct {
	BatchSize   int
	MaxWaitTime time.Duration
	WorkerCount int
}

// LLMLatencyDataPoint represents a latency data point (stub)
type LLMLatencyDataPoint struct {
	Timestamp time.Time
	Latency   time.Duration
	Model     string
}

// Priority represents request priority (stub)
type Priority int

const (
	LowPriority Priority = iota
	MediumPriority
	HighPriority
)

// BatchResult represents batch processing result (stub)
type BatchResult struct {
	Results []interface{}
	Errors  []error
}

// BatchProcessorStats provides batch processor statistics (stub)
type BatchProcessorStats struct {
	TotalProcessed int64
	TotalErrors    int64
	AverageLatency time.Duration
}

// CircuitBreaker provides stub circuit breaker implementation
type CircuitBreaker struct{}

// CircuitBreakerConfig provides circuit breaker configuration (stub)
type CircuitBreakerConfig struct {
	MaxFailures int
	Timeout     time.Duration
}

// MetricsCollector provides metrics collection (stub)
type MetricsCollector struct{}

// ContextBuilder provides context building functionality (stub)
type ContextBuilder struct{}

// ProcessingContext provides processing context (stub)
type ProcessingContext struct {
	RequestID string
	Timestamp time.Time
}

// Stub methods for ResponseCache
func (r *ResponseCache) Get(key string) (interface{}, bool) {
	return nil, false
}

func (r *ResponseCache) Set(key string, value interface{}) {}

func (r *ResponseCache) Delete(key string) {}

func (r *ResponseCache) Clear() {}

// Stub methods for CircuitBreaker
func (cb *CircuitBreaker) Execute(req func() (interface{}, error)) (interface{}, error) {
	return req()
}

func (cb *CircuitBreaker) IsOpen() bool {
	return false
}

func (cb *CircuitBreaker) Reset() {}

// Stub methods for MetricsCollector
func (mc *MetricsCollector) Collect(metric string, value float64) {}

func (mc *MetricsCollector) GetMetrics() map[string]interface{} {
	return make(map[string]interface{})
}

// Stub methods for ContextBuilder
func (cb *ContextBuilder) Build(ctx context.Context) *ProcessingContext {
	return &ProcessingContext{
		RequestID: "stub-request",
		Timestamp: time.Now(),
	}
}
