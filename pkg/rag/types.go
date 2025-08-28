// Package rag provides Retrieval-Augmented Generation interfaces
// This file contains interface definitions that allow conditional compilation
// with or without Weaviate dependencies
package rag

import (
	"context"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
	"github.com/thc1006/nephoran-intent-operator/pkg/types"
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

	// Query performs a query against the RAG system
	Query(ctx context.Context, query string) (interface{}, error)
}

// Service interface for compatibility with CNF package
type Service interface {
	Query(ctx context.Context, query string) (interface{}, error)
}

// SearchResult is an alias to shared.SearchResult for compatibility
type SearchResult = shared.SearchResult

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

// Note: TokenUsage is defined in embedding_service.go

// WeaviateClient stub for conditional compilation
type WeaviateClient struct {
	// This is a stub - actual implementation in weaviate_client_complex.go with build tag
}

// TelecomDocument is an alias to types.TelecomDocument for backwards compatibility
// This is used in conditional compilation where the actual implementation may differ
type TelecomDocument = types.TelecomDocument

// WeaviateHealthStatus represents the health status of Weaviate (stub for conditional compilation)
type WeaviateHealthStatus struct {
	IsHealthy bool                   `json:"is_healthy"`
	LastCheck time.Time              `json:"last_check"`
	Details   map[string]interface{} `json:"details"`
	Version   string                 `json:"version,omitempty"`
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

// AddDocument is a stub method for conditional compilation
func (w *WeaviateClient) AddDocument(ctx context.Context, doc *TelecomDocument) error {
	// Stub implementation - actual implementation in weaviate_client_complex.go with build tag
	return nil
}

// Close is a stub method for conditional compilation
func (w *WeaviateClient) Close() error {
	// Stub implementation - actual implementation in weaviate_client_complex.go with build tag
	return nil
}

// WeaviateConfig holds Weaviate configuration
type WeaviateConfig struct {
	Host    string            `json:"host"`
	Scheme  string            `json:"scheme"`
	APIKey  string            `json:"api_key"`
	Timeout time.Duration     `json:"timeout"`
	Headers map[string]string `json:"headers"`
	Retries int               `json:"retries"`
}

// SearchQuery is an alias to types.SearchQuery for compatibility
type SearchQuery = types.SearchQuery

// SearchResponse represents a search response
type SearchResponse struct {
	Results []*SearchResult `json:"results"`
	Took    time.Duration   `json:"took"`
	Total   int64           `json:"total"`
	Query   string          `json:"query"`
}

// Note: OptimizedRAGPipeline, OptimizedRAGService, OptimizedRAGConfig, and OptimizedRAGMetrics
// are defined in their respective files to avoid redeclaration errors

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

// newRAGClientImpl is implemented in:
// - client_noop.go (for !rag build tag)
// - weaviate_client_complex.go (for rag build tag)

// QueryRequest represents a query request to the RAG system
type QueryRequest struct {
	Query       string                 `json:"query"`
	Limit       int                    `json:"limit"`
	Filters     map[string]interface{} `json:"filters"`
	Context     map[string]interface{} `json:"context"`
	IntentType  string                 `json:"intent_type"`
	SessionID   string                 `json:"session_id"`
	ClientID    string                 `json:"client_id"`
	Metadata    map[string]interface{} `json:"metadata"`
	Timeout     time.Duration          `json:"timeout"`
}

// Document represents a document in the RAG system (alias to Doc for compatibility)
type Document = Doc

// RetrievalRequest represents a request for document retrieval
type RetrievalRequest struct {
	Query       string                 `json:"query"`
	Limit       int                    `json:"limit,omitempty"`
	Filters     map[string]interface{} `json:"filters,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	IntentType  string                 `json:"intent_type,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	ClientID    string                 `json:"client_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Timeout     time.Duration          `json:"timeout,omitempty"`
}

// RetrievalResponse represents a response from document retrieval
type RetrievalResponse struct {
	Documents   []Doc                  `json:"documents"`
	Query       string                 `json:"query"`
	TotalFound  int                    `json:"total_found"`
	ProcessTime time.Duration          `json:"process_time"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Error       string                 `json:"error,omitempty"`
}