//go:build disable_rag
// +build disable_rag

package llm

import (
	"context"
	"time"
)

// DISABLED RAG INTERFACES - Minimal stub implementations

// LLMProcessor is the main interface for LLM processing
type LLMProcessor interface {
	ProcessIntent(ctx context.Context, request *ProcessingRequest) (*ProcessingResponse, error)
	GetMetrics() ClientMetrics
	Shutdown()
}

// Processor is an alias for backward compatibility
type Processor = LLMProcessor

// BatchProcessor handles batch processing of multiple intents (disabled)
type BatchProcessor interface {
	ProcessRequest(ctx context.Context, intent, intentType, modelName string, priority Priority) (*BatchResult, error)
	GetStats() BatchProcessorStats
	Close() error
}

// StreamingProcessor handles streaming requests (disabled)
type StreamingProcessor struct {
	// Stub implementation fields
}

// CacheProvider provides caching functionality (disabled)
type CacheProvider interface {
	Get(key string) (string, bool)
	Set(key, response string)
	Clear()
	Stop()
	GetStats() map[string]interface{}
}

// PromptGenerator generates prompts for different intent types (disabled)
type PromptGenerator interface {
	GeneratePrompt(intentType, userIntent string) string
	ExtractParameters(intent string) map[string]interface{}
}

// ProcessingRequest represents a processing request (stub)
type ProcessingRequest struct {
	Intent      string            `json:"intent"`
	IntentType  string            `json:"intent_type"`
	ModelName   string            `json:"model_name"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature float32           `json:"temperature,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// ProcessingResponse represents a processing response (stub)
type ProcessingResponse struct {
	Content      string            `json:"content"`
	TokensUsed   int               `json:"tokens_used"`
	ProcessingTime time.Duration   `json:"processing_time"`
	ModelUsed    string            `json:"model_used"`
	Cached       bool              `json:"cached"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}