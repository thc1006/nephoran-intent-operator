//go:build disable_rag || stub

package rag

import (
	"context"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// Core RAG types

type SearchResult = shared.SearchResult
type SearchQuery = shared.SearchQuery
type SearchResponse = shared.RAGResponse

type RAGRequest struct {
	Query            string
	MaxResults       int
	UseHybridSearch  bool
	EnableReranking  bool
	SearchFilters    map[string]interface{}
	MinConfidence    float64
}

type RAGResponse struct {
	Answer           string
	SourceDocuments  []*SearchResult
	Confidence       float32
	ProcessingTime   time.Duration
	RetrievalTime    time.Duration
	Query            string
	ProcessedAt      time.Time
}

type WeaviateClient interface {
	Search(ctx context.Context, query *SearchQuery) (*SearchResponse, error)
	IsHealthy() bool
	Close() error
}

type WeaviateConnectionPool interface {
	GetClient() (WeaviateClient, error)
	ReleaseClient(client WeaviateClient)
	GetMetrics() map[string]interface{}
	Close() error
}

type OptimizedRAGService interface {
	ProcessQuery(ctx context.Context, request *RAGRequest) (*RAGResponse, error)
	GetOptimizedMetrics() map[string]interface{}
	GetHealth() string
}

type WeaviateConfig struct {
	Host   string
	Scheme string
}

type PoolConfig struct{}
type OptimizedRAGConfig struct{}

func NewWeaviateConnectionPool(config *PoolConfig) (WeaviateConnectionPool, error) {
	return nil, nil
}

func NewOptimizedRAGService(config *OptimizedRAGConfig) (OptimizedRAGService, error) {
	return nil, nil
}

// Utility function
func convertSearchResults(results []SearchResult) []*SearchResult {
	converted := make([]*SearchResult, len(results))
	for i := range results {
		converted[i] = &results[i]
	}
	return converted
}

// Health status for Weaviate
type WeaviateHealthStatus struct {
	IsHealthy bool      `json:"is_healthy"`
	LastCheck time.Time `json:"last_check"`
	Version   string    `json:"version"`
	Details   string    `json:"details"`
	Message   string    `json:"message"`
}
