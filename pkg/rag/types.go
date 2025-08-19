// Package rag provides Retrieval-Augmented Generation interfaces
// This file contains interface definitions that allow conditional compilation
// with or without Weaviate dependencies
package rag

import (
	"context"
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

// Note: SearchResult is defined in enhanced_rag_integration.go

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

// NewRAGClient creates a new RAG client based on build tags
// With "rag" build tag: returns Weaviate-based implementation
// Without "rag" build tag: returns no-op implementation
func NewRAGClient(config *RAGClientConfig) RAGClient {
	// This function will be implemented differently in:
	// - weaviate_client.go (with //go:build rag)
	// - noop/client.go (no build tag)
	return newRAGClientImpl(config)
}
