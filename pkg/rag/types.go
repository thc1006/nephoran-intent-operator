package rag

import (
	"context"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// Type aliases to shared types
type SearchResult = shared.SearchResult
type SearchQuery = shared.SearchQuery
type SearchResponse = shared.RAGResponse

// WeaviateClient interface for RAG operations
type WeaviateClient interface {
	Search(ctx context.Context, query *SearchQuery) (*SearchResponse, error)
	IsHealthy() bool
	Close() error
}

// Additional types for compatibility
type HNSWOptimizer interface{}
type OptimizedRAGConfig struct{}
type OptimizedRAGService interface{}
type WeaviateConnectionPool interface{}
type PoolConfig struct{}
type WeaviateConfig struct {
	Host   string
	Scheme string
}
type Doc interface{}
type RAGClientConfig struct{}
type RAGClient interface{}
type QueryPattern interface{}
type OptimizationResult interface{}
type OptimizedRAGMetrics interface{}

// Placeholder types for module resolution
type RagContext struct{}
type RagProcessor interface{}
