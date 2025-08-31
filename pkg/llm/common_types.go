package llm

import (
	"sync"
	"time"
)

// CommonClientMetrics tracks client performance (used by both disable_rag and regular builds)
type CommonClientMetrics struct {
	RequestsTotal    int64         `json:"requests_total"`
	RequestsSuccess  int64         `json:"requests_success"`
	RequestsFailure  int64         `json:"requests_failure"`
	TotalLatency     time.Duration `json:"total_latency"`
	CacheHits        int64         `json:"cache_hits"`
	CacheMisses      int64         `json:"cache_misses"`
	RetryAttempts    int64         `json:"retry_attempts"`
	FallbackAttempts int64         `json:"fallback_attempts"`
	mutex            sync.RWMutex
}

// SimpleTokenTracker tracks token usage and costs
type SimpleTokenTracker struct {
	totalTokens  int64
	totalCost    float64
	requestCount int64
	mutex        sync.RWMutex
}

// NewSimpleTokenTracker creates a new simple token tracker
func NewSimpleTokenTracker() *SimpleTokenTracker {
	return &SimpleTokenTracker{}
}

// RecordUsage records token usage
func (tt *SimpleTokenTracker) RecordUsage(tokens int) {
	tt.mutex.Lock()
	defer tt.mutex.Unlock()

	tt.totalTokens += int64(tokens)
	tt.requestCount++

	// Simple cost calculation (adjust based on model pricing)
	costPerToken := 0.0001 // Example: $0.0001 per token
	tt.totalCost += float64(tokens) * costPerToken
}

// GetStats returns token usage statistics
func (tt *SimpleTokenTracker) GetStats() map[string]interface{} {
	tt.mutex.RLock()
	defer tt.mutex.RUnlock()

	avgTokensPerRequest := float64(0)
	if tt.requestCount > 0 {
		avgTokensPerRequest = float64(tt.totalTokens) / float64(tt.requestCount)
	}

	return map[string]interface{}{
		"total_tokens":           tt.totalTokens,
		"total_cost":             tt.totalCost,
		"request_count":          tt.requestCount,
		"avg_tokens_per_request": avgTokensPerRequest,
	}
}