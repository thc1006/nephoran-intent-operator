package llm

import (
	"context"
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

// BatchProcessor handles batch processing of multiple intents
type BatchProcessor interface {
	ProcessBatch(ctx context.Context, requests []*BatchRequest) ([]*ProcessingResult, error)
	GetMetrics() *ProcessingMetrics
}

// StreamingProcessor handles streaming requests
type StreamingProcessor interface {
	HandleStreamingRequest(w interface{}, r interface{}, req *StreamingRequest) error
	GetMetrics() map[string]interface{}
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

// CircuitBreakerConfig holds configuration for circuit breaker (consolidated)
type CircuitBreakerConfig struct {
	FailureThreshold    int64         `json:"failure_threshold"`
	FailureRate         float64       `json:"failure_rate"`
	MinimumRequestCount int64         `json:"minimum_request_count"`
	Timeout             time.Duration `json:"timeout"`
	HalfOpenTimeout     time.Duration `json:"half_open_timeout"`
	SuccessThreshold    int64         `json:"success_threshold"`
	HalfOpenMaxRequests int64         `json:"half_open_max_requests"`
	ResetTimeout        time.Duration `json:"reset_timeout"`
	SlidingWindowSize   int           `json:"sliding_window_size"`
	EnableHealthCheck   bool          `json:"enable_health_check"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	HealthCheckTimeout  time.Duration `json:"health_check_timeout"`
}

// TokenTracker tracks token usage and costs
type TokenTracker struct {
	totalTokens  int64
	totalCost    float64
	requestCount int64
	mutex        sync.RWMutex
}

// NewTokenTracker creates a new token tracker
func NewTokenTracker() *TokenTracker {
	return &TokenTracker{}
}

// RecordUsage records token usage
func (tt *TokenTracker) RecordUsage(tokens int) {
	tt.mutex.Lock()
	defer tt.mutex.Unlock()

	tt.totalTokens += int64(tokens)
	tt.requestCount++

	// Simple cost calculation (adjust based on model pricing)
	costPerToken := 0.0001 // Example: $0.0001 per token
	tt.totalCost += float64(tokens) * costPerToken
}

// GetStats returns token usage statistics
func (tt *TokenTracker) GetStats() map[string]interface{} {
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

// RelevanceScorer stub implementation (consolidated from stubs.go)
type RelevanceScorer struct{}

func NewRelevanceScorer() *RelevanceScorer {
	return &RelevanceScorer{}
}

func (rs *RelevanceScorer) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"relevance_scorer_enabled": false,
		"status":                   "not_implemented",
	}
}

// RAGAwarePromptBuilder stub implementation (consolidated from stubs.go)
type RAGAwarePromptBuilder struct{}

func NewRAGAwarePromptBuilder() *RAGAwarePromptBuilder {
	return &RAGAwarePromptBuilder{}
}

func (rpb *RAGAwarePromptBuilder) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"prompt_builder_enabled": false,
		"status":                 "not_implemented",
	}
}

// UTILITY FUNCTIONS

// getDefaultCircuitBreakerConfig returns default circuit breaker configuration
func getDefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		FailureThreshold:    5,
		FailureRate:         0.5,
		MinimumRequestCount: 10,
		Timeout:             30 * time.Second,
		HalfOpenTimeout:     60 * time.Second,
		SuccessThreshold:    3,
		HalfOpenMaxRequests: 5,
		ResetTimeout:        60 * time.Second,
		SlidingWindowSize:   100,
		EnableHealthCheck:   false,
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  10 * time.Second,
	}
}

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
