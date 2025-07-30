package shared

import (
	"context"
	"time"
)

// ClientInterface defines the interface for LLM clients
// This interface is shared between packages to avoid circular dependencies
type ClientInterface interface {
	ProcessIntent(ctx context.Context, prompt string) (string, error)
	ProcessIntentStream(ctx context.Context, prompt string, chunks chan<- *StreamingChunk) error
	GetSupportedModels() []string
	GetModelCapabilities(modelName string) (*ModelCapabilities, error)
	ValidateModel(modelName string) error
	EstimateTokens(text string) int
	GetMaxTokens(modelName string) int
	Close() error
}

// StreamingChunk represents a chunk of streamed response
type StreamingChunk struct {
	Content   string
	IsLast    bool
	Metadata  map[string]interface{}
	Timestamp time.Time
}

// ModelCapabilities describes what a model can do
type ModelCapabilities struct {
	MaxTokens        int                    `json:"max_tokens"`
	SupportsChat     bool                   `json:"supports_chat"`
	SupportsFunction bool                   `json:"supports_function"`
	SupportsStreaming bool                  `json:"supports_streaming"`
	CostPerToken     float64                `json:"cost_per_token"`
	Features         map[string]interface{} `json:"features"`
}

// TelecomDocument represents a document in the telecom knowledge base
type TelecomDocument struct {
	ID              string                 `json:"id"`
	Title           string                 `json:"title"`
	Content         string                 `json:"content"`
	Source          string                 `json:"source"`
	Category        string                 `json:"category"`
	Version         string                 `json:"version"`
	Keywords        []string               `json:"keywords"`
	Language        string                 `json:"language"`
	DocumentType    string                 `json:"document_type"`
	NetworkFunction []string               `json:"network_function"`
	Technology      []string               `json:"technology"`
	UseCase         []string               `json:"use_case"`
	Confidence      float32                `json:"confidence"`
	Metadata        map[string]interface{} `json:"metadata"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// SearchResult represents a search result from the vector database
type SearchResult struct {
	Document *TelecomDocument `json:"document"`
	Score    float32          `json:"score"`
	Distance float32          `json:"distance"`
	Metadata map[string]interface{} `json:"metadata"`
}