//go:build disable_rag
// +build disable_rag

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

// StreamingRequest is defined in types.go

// BatchRequest is defined in types.go

// ProcessingResult is defined in processing.go

// ProcessingMetrics is defined in processing.go

// MetricsCollector is defined in metrics.go
// NewMetricsCollector is defined in metrics.go

// MetricsIntegrator for disable_rag builds
type MetricsIntegrator struct {
	prometheusMetrics *PrometheusMetricsStub
}

// PrometheusMetricsStub provides stub prometheus metrics (disable_rag builds)
type PrometheusMetricsStub struct{}

func (pm *PrometheusMetricsStub) RecordError(errorType string, details string) {}

// NewMetricsIntegrator stub for disable_rag builds
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

// TokenManager is defined as interface in types.go
// For disable_rag builds, use basic_token_manager.go implementation

// Stub types for disable_rag builds
type RelevanceScorer struct{}
func (rs *RelevanceScorer) GetMetrics() map[string]interface{} { return map[string]interface{}{} }

type RAGAwarePromptBuilder struct{}
func (rpb *RAGAwarePromptBuilder) GetMetrics() map[string]interface{} { return map[string]interface{}{} }

type RAGEnhancedProcessor struct{}
func (rep *RAGEnhancedProcessor) ProcessIntent(ctx context.Context, intent string) (string, error) {
	return "", fmt.Errorf("RAG functionality disabled with disable_rag build tag")
}

// Constructor functions for disable_rag builds
func NewRelevanceScorer() *RelevanceScorer { return &RelevanceScorer{} }
func NewRAGAwarePromptBuilder() *RAGAwarePromptBuilder { return &RAGAwarePromptBuilder{} }
func NewRAGEnhancedProcessor() *RAGEnhancedProcessor { return &RAGEnhancedProcessor{} }
// NewStreamingProcessor is defined in clean_stubs.go or streaming_processor.go

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

// ClientMetrics is defined in common_types.go

// SimpleTokenTracker is defined in common_types.go

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

// ContextBuilder is defined in clean_stubs.go

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
