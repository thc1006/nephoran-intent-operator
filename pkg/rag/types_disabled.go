//go:build disable_rag

package rag

import (
	"context"
	"time"
)

// Doc represents a document retrieved from the RAG system (stub).
type Doc struct {
	ID         string
	Content    string
	Confidence float64
	Metadata   map[string]interface{}
}

// RAGClient is the main interface for RAG operations (stub).
type RAGClient interface {
	// Retrieve performs a semantic search for relevant documents.
	Retrieve(ctx context.Context, query string) ([]Doc, error)
	// ProcessIntent processes an intent and returns a response.
	ProcessIntent(ctx context.Context, intent string) (string, error)
	// IsHealthy returns the health status of the RAG client.
	IsHealthy() bool
	// Initialize initializes the RAG client and its dependencies.
	Initialize(ctx context.Context) error
	// Shutdown gracefully shuts down the RAG client and releases resources.
	Shutdown(ctx context.Context) error
	// Health returns the health status of the client.
	Health(ctx context.Context) (*HealthStatus, error)
	// Close closes the client connection.
	Close() error
	// Query returns query results.
	Query(ctx context.Context, query string) ([]*Doc, error)
}

// RAGClientConfig holds configuration for RAG clients (stub).
type RAGClientConfig struct {
	// Common configuration.
	Enabled          bool
	MaxSearchResults int
	MinConfidence    float64
	// Weaviate-specific (used only when rag build tag is enabled).
	WeaviateURL    string
	WeaviateAPIKey string
	// LLM configuration.
	LLMEndpoint string
	LLMAPIKey   string
	MaxTokens   int
	Temperature float32
}

// HealthStatus represents the health status of a component (stub).
type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Details   map[string]string `json:"details,omitempty"`
}

// NewRAGClient creates a new RAG client based on build tags (stub).
func NewRAGClient(config *RAGClientConfig) RAGClient {
	return newRAGClientImpl(config)
}