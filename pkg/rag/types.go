// Package rag provides Retrieval-Augmented Generation interfaces
// This file contains interface definitions that allow conditional compilation
// with or without Weaviate dependencies
package rag

import (
	"context"
)

// RAGClient is the main interface for RAG operations
// This allows for different implementations (Weaviate, no-op, etc.)
type RAGClient interface {
	// ProcessIntent processes an intent with RAG enhancement
	ProcessIntent(ctx context.Context, intent string) (string, error)
	
	// Search performs a semantic search for relevant documents
	Search(ctx context.Context, query string, limit int) ([]SearchResult, error)
	
	// Initialize sets up the RAG client
	Initialize(ctx context.Context) error
	
	// Shutdown gracefully shuts down the client
	Shutdown(ctx context.Context) error
	
	// IsHealthy checks if the RAG service is healthy
	IsHealthy() bool
}

// SearchResult represents a search result from the RAG system
type SearchResult struct {
	ID         string
	Content    string
	Confidence float64
	Metadata   map[string]interface{}
}

// RAGClientConfig holds configuration for RAG clients
type RAGClientConfig struct {
	// Common configuration
	Enabled          bool
	MaxSearchResults int
	MinConfidence    float64
	
	// Weaviate-specific (used only when rag build tag is enabled)
	WeaviateURL      string
	WeaviateAPIKey   string
	
	// LLM configuration
	LLMEndpoint      string
	LLMAPIKey        string
	MaxTokens        int
	Temperature      float32
}

// TokenUsage tracks token usage for cost tracking
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	EstimatedCost    float64
}

// NewRAGClient creates a new RAG client based on build tags
// With "rag" build tag: returns Weaviate-based implementation
// Without "rag" build tag: returns no-op implementation
func NewRAGClient(config *RAGClientConfig) RAGClient {
	// This function will be implemented differently in:
	// - client_weaviate.go (with //go:build rag)
	// - client_noop.go (with //go:build !rag)
	return newRAGClientImpl(config)
}