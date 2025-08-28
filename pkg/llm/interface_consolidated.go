package llm

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CONSOLIDATED INTERFACES - Simplified from over-engineered abstractions

// LLMProcessor is the main interface for LLM processing
type LLMProcessor interface {
	ProcessIntent(ctx context.Context, intent string) (string, error)
	GetMetrics() ClientMetrics
	Shutdown()
}

// BatchProcessorInterface handles batch processing of multiple intents
type BatchProcessorInterface interface {
	ProcessBatch(ctx context.Context, requests []*BatchRequest) ([]*ProcessingResult, error)
	GetMetrics() *ProcessingMetrics
}


// StreamingProcessor handles streaming requests (concrete implementation for disable_rag builds)
type StreamingProcessor struct {
	// Stub implementation fields

}

// CacheProvider provides caching functionality
type CacheProvider interface {
	Get(key string) (string, bool)
	Set(key, response string)
	Clear()
	Stop()
	GetStats() map[string]interface{}
}

// PromptGenerator generates prompts for different intent types
type PromptGenerator interface {
	GeneratePrompt(intentType, userIntent string) string
	ExtractParameters(intent string) map[string]interface{}
}

// ESSENTIAL TYPES ONLY - Consolidated from scattered definitions

// StreamingRequest represents a streaming request (stub for disable_rag builds)
type StreamingRequest struct {
	Content   string `json:"content"`
	Query     string `json:"query"`
	ModelName string `json:"model_name"`
	MaxTokens int    `json:"max_tokens"`
	EnableRAG bool   `json:"enable_rag"`
}

// BatchRequest represents a batch processing request (stub for disable_rag builds)
type BatchRequest struct {
	Content string `json:"content"`
}

// ProcessingResult represents a processing result (stub for disable_rag builds)
type ProcessingResult struct {
	Result string `json:"result"`
}

// ProcessingMetrics tracks processing metrics (stub for disable_rag builds)
type ProcessingMetrics struct {
	ProcessedCount int64 `json:"processed_count"`
}

// MetricsCollector collects metrics (stub for disable_rag builds)
type MetricsCollector struct {
	// Stub implementation
}

// NewMetricsCollector creates a new metrics collector (stub for disable_rag builds)
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{}
}

// MetricsIntegrator integrates metrics (stub for disable_rag builds)
type MetricsIntegrator struct {
	// Stub implementation
	prometheusMetrics *PrometheusMetricsStub
}

// PrometheusMetricsStub provides stub prometheus metrics (disable_rag builds)
type PrometheusMetricsStub struct{}

func (pm *PrometheusMetricsStub) RecordError(errorType string, details string) {}

// NewMetricsIntegrator creates a new metrics integrator (stub for disable_rag builds)
func NewMetricsIntegrator(collector *MetricsCollector) *MetricsIntegrator {
	return &MetricsIntegrator{
		prometheusMetrics: &PrometheusMetricsStub{},
	}
}

// Stub methods for MetricsIntegrator (disable_rag builds)
func (mi *MetricsIntegrator) RecordCircuitBreakerEvent(event string, state string, info string) {}
func (mi *MetricsIntegrator) RecordLLMRequest(backend string, model string, duration time.Duration, tokens int) {}
func (mi *MetricsIntegrator) RecordCacheOperation(operation string, backend string, hit bool) {}
func (mi *MetricsIntegrator) RecordRetryAttempt(model string) {}
func (mi *MetricsIntegrator) GetComprehensiveMetrics() map[string]interface{} { return nil }

// Additional stub types for disable_rag builds
type TokenManager struct{}
func (tm *TokenManager) GetSupportedModels() []string { return []string{} }

type RelevanceScorer struct{}
func (rs *RelevanceScorer) GetMetrics() map[string]interface{} { return map[string]interface{}{} }

type RAGAwarePromptBuilder struct{}
func (rpb *RAGAwarePromptBuilder) GetMetrics() map[string]interface{} { return map[string]interface{}{} }

type RAGEnhancedProcessor struct{}
func (rep *RAGEnhancedProcessor) ProcessIntent(ctx context.Context, intent string) (string, error) {
	return "", fmt.Errorf("RAG functionality disabled with disable_rag build tag")
}

// Constructor functions for disable_rag builds
func NewTokenManager() *TokenManager { return &TokenManager{} }
func NewRelevanceScorer() *RelevanceScorer { return &RelevanceScorer{} }
func NewRAGAwarePromptBuilder() *RAGAwarePromptBuilder { return &RAGAwarePromptBuilder{} }
func NewRAGEnhancedProcessor() *RAGEnhancedProcessor { return &RAGEnhancedProcessor{} }
func NewStreamingProcessor() *StreamingProcessor { return &StreamingProcessor{} }

// Stub methods for StreamingProcessor (disable_rag builds)
func (sp *StreamingProcessor) HandleStreamingRequest(w interface{}, r interface{}, req *StreamingRequest) error {
	// Stub implementation - just return an error indicating streaming is disabled
	return fmt.Errorf("streaming functionality is disabled with disable_rag build tag")
}

func (sp *StreamingProcessor) GetMetrics() map[string]interface{} {
	return map[string]interface{}{"streaming_disabled": true}
}

func (sp *StreamingProcessor) Shutdown(ctx context.Context) error {
	// Stub implementation - nothing to shutdown
	return nil
}

// ClientMetrics tracks client performance (consolidated from multiple files)
type ClientMetrics struct {
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

// BACKWARD COMPATIBILITY SECTION
// These maintain compatibility with existing code but should be phased out

// Config represents LLM client configuration (from old interface.go)
type Config struct {
	Endpoint string
	Timeout  time.Duration
	APIKey   string
	Model    string
}

// IntentRequest represents a request to process an intent (from old interface.go)
type IntentRequest struct {
	Intent      string                 `json:"intent"`
	Prompt      string                 `json:"prompt"`
	Context     map[string]interface{} `json:"context"`
	MaxTokens   int                    `json:"maxTokens"`
	Temperature float64                `json:"temperature"`
}

// IntentResponse represents the response from intent processing (from old interface.go)
type IntentResponse struct {
	Response   string                 `json:"response"`
	Confidence float64                `json:"confidence"`
	Tokens     int                    `json:"tokens"`
	Duration   time.Duration          `json:"duration"`
	Metadata   map[string]interface{} `json:"metadata"`
}


// STUB IMPLEMENTATIONS - Consolidated from stubs.go
// These provide default implementations for components not yet fully implemented

// ContextBuilder stub implementation (consolidated from stubs.go)
type ContextBuilder struct{}

func NewContextBuilder() *ContextBuilder {
	return &ContextBuilder{}
}

func (cb *ContextBuilder) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"context_builder_enabled": false,
		"status":                  "not_implemented",
	}
}

// RelevanceScorer implementation moved to relevance_scorer.go

// RAGAwarePromptBuilder implementation moved to rag_aware_prompt_builder.go

// UTILITY FUNCTIONS

// Use getDefaultCircuitBreakerConfig from circuit_breaker.go to avoid duplicates


// isValidKubernetesName validates Kubernetes resource names
func isValidKubernetesName(name string) bool {
	if len(name) == 0 || len(name) > 253 {
		return false
	}

	// Kubernetes names must match DNS subdomain format
	for i, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '.') {
			return false
		}
		if i == 0 && (r == '-' || r == '.') {
			return false
		}
		if i == len(name)-1 && (r == '-' || r == '.') {
			return false
		}
	}

	return true
}
