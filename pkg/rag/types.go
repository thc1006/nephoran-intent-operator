// Package rag provides Retrieval-Augmented Generation interfaces
// This file contains interface definitions that allow conditional compilation
// with or without Weaviate dependencies
package rag

import (
	"context"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// Doc represents a document retrieved from the RAG system
type Doc struct {
	ID         string
	Content    string
	Confidence float64
	Metadata   map[string]interface{}
}

// RAGClient is the main interface for RAG operations
// This allows for different implementations (Weaviate, no-op, etc.)
type RAGClient interface {
	// Retrieve performs a semantic search for relevant documents
	Retrieve(ctx context.Context, query string) ([]Doc, error)

	// Initialize initializes the RAG client and its dependencies
	Initialize(ctx context.Context) error

	// Shutdown gracefully shuts down the RAG client and releases resources
	Shutdown(ctx context.Context) error
}

// SearchResult represents a search result from the RAG system
// Deprecated: Use Doc instead
type SearchResult struct {
	ID         string
	Content    string
	Confidence float64
	Metadata   map[string]interface{}
	// Additional fields for compatibility
	Score      float32 `json:"score"`
	Document   *shared.TelecomDocument `json:"document"`
}

// RAGClientConfig holds configuration for RAG clients
type RAGClientConfig struct {
	// Common configuration
	Enabled          bool
	MaxSearchResults int
	MinConfidence    float64

	// Weaviate-specific (used only when rag build tag is enabled)
	WeaviateURL    string
	WeaviateAPIKey string

	// LLM configuration
	LLMEndpoint string
	LLMAPIKey   string
	MaxTokens   int
	Temperature float32
}

// TokenUsage tracks token usage for cost tracking
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	EstimatedCost    float64
}

// WeaviateClient stub for conditional compilation
type WeaviateClient struct {
	// This is a stub - actual implementation in weaviate_client_complex.go with build tag
}

// WeaviateHealthStatus represents the health status of Weaviate (stub for conditional compilation)
type WeaviateHealthStatus struct {
	IsHealthy bool                   `json:"is_healthy"`
	LastCheck time.Time              `json:"last_check"`
	Details   map[string]interface{} `json:"details"`
}

// Search is a stub method for conditional compilation
func (w *WeaviateClient) Search(ctx context.Context, query *SearchQuery) (*SearchResponse, error) {
	// Stub implementation - actual implementation in weaviate_client_complex.go with build tag
	return &SearchResponse{}, nil
}

// GetHealthStatus is a stub method for conditional compilation
func (w *WeaviateClient) GetHealthStatus() *WeaviateHealthStatus {
	// Stub implementation - actual implementation in weaviate_client_complex.go with build tag
	return &WeaviateHealthStatus{
		IsHealthy: true,
		LastCheck: time.Now(),
		Details:   make(map[string]interface{}),
	}
}

// Close is a stub method for conditional compilation
func (w *WeaviateClient) Close() error {
	// Stub implementation - actual implementation in weaviate_client_complex.go with build tag
	return nil
}

// WeaviateConfig holds Weaviate configuration
type WeaviateConfig struct {
	Host     string        `json:"host"`
	Scheme   string        `json:"scheme"`
	APIKey   string        `json:"api_key"`
	Timeout  time.Duration `json:"timeout"`
	Headers  map[string]string `json:"headers"`
	Retries  int           `json:"retries"`
}

// SearchQuery represents a search query
type SearchQuery struct {
	Query         string                 `json:"query"`
	Limit         int                    `json:"limit"`
	Filters       map[string]interface{} `json:"filters"`
	HybridSearch  bool                   `json:"hybrid_search"`
	UseReranker   bool                   `json:"use_reranker"`
	MinConfidence float32                `json:"min_confidence"`
	HybridAlpha   *float32               `json:"hybrid_alpha,omitempty"`
	ExpandQuery   bool                   `json:"expand_query"`
}

// SearchResponse represents a search response
type SearchResponse struct {
	Results []*SearchResult `json:"results"`
	Took    time.Duration   `json:"took"`
	Total   int64           `json:"total"`
	Query   string          `json:"query"`
}

// Missing types for compilation
type OptimizedRAGPipeline struct {
	// Placeholder for optimized pipeline
}

type OptimizedRAGService struct {
	// Placeholder for optimized service
}

type OptimizedRAGConfig struct {
	// Placeholder for optimized config
}

type OptimizedRAGMetrics struct {
	// Placeholder for optimized metrics
}

// Note: ProcessedChunk and DocumentChunk are defined in their respective files
// to avoid redeclaration errors

// NewWeaviateClient creates a new Weaviate client (stub for conditional compilation)
func NewWeaviateClient(config *WeaviateConfig) (*WeaviateClient, error) {
	// Stub implementation - actual implementation in weaviate_client_complex.go with build tag
	return &WeaviateClient{}, nil
}

// NewRAGClient creates a new RAG client based on build tags
// With "rag" build tag: returns Weaviate-based implementation
// Without "rag" build tag: returns no-op implementation
func NewRAGClient(config *RAGClientConfig) RAGClient {
	// This function will be implemented differently in:
	// - weaviate_client.go (with //go:build rag)
	// - noop/client.go (no build tag)
	return newRAGClientImpl(config)
}
