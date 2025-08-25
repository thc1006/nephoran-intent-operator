package shared

import (
	"context"
	"time"
)

// ClientInterface defines the standard LLM client interface
type ClientInterface interface {
	// Basic operations
	ProcessRequest(ctx context.Context, request *LLMRequest) (*LLMResponse, error)
	ProcessStreamingRequest(ctx context.Context, request *LLMRequest) (<-chan *StreamingChunk, error)
	
	// Health and status
	HealthCheck(ctx context.Context) error
	GetStatus() ClientStatus
	
	// Configuration
	GetModelCapabilities() ModelCapabilities
	GetEndpoint() string
	Close() error
}

// LLMRequest represents a request to an LLM service
type LLMRequest struct {
	Model       string                 `json:"model"`
	Messages    []ChatMessage          `json:"messages"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Temperature float32                `json:"temperature,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// LLMResponse represents a response from an LLM service
type LLMResponse struct {
	ID      string      `json:"id"`
	Content string      `json:"content"`
	Model   string      `json:"model"`
	Usage   TokenUsage  `json:"usage"`
	Created time.Time   `json:"created"`
	Error   *LLMError   `json:"error,omitempty"`
}

// StreamingChunk represents a chunk of streamed response
type StreamingChunk struct {
	ID      string    `json:"id"`
	Content string    `json:"content"`
	Delta   string    `json:"delta"`
	Done    bool      `json:"done"`
	Error   *LLMError `json:"error,omitempty"`
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Role    string `json:"role"`    // system, user, assistant
	Content string `json:"content"`
}

// TokenUsage represents token usage information
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// LLMError represents an error from LLM service
type LLMError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

// ClientStatus represents the status of a client
type ClientStatus string

const (
	ClientStatusHealthy     ClientStatus = "healthy"
	ClientStatusUnhealthy   ClientStatus = "unhealthy"
	ClientStatusUnavailable ClientStatus = "unavailable"
	ClientStatusMaintenance ClientStatus = "maintenance"
)

// ModelCapabilities represents model capabilities
type ModelCapabilities struct {
	SupportsStreaming   bool     `json:"supports_streaming"`
	SupportsSystemPrompt bool    `json:"supports_system_prompt"`
	SupportsChatFormat  bool     `json:"supports_chat_format"`
	MaxTokens          int      `json:"max_tokens"`
	SupportedMimeTypes []string `json:"supported_mime_types"`
	ModelVersion       string   `json:"model_version"`
}