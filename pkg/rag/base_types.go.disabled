package rag

import (
	"context"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// Base type aliases to avoid conflicts and provide common interface
type SearchResult = shared.SearchResult
type SearchQuery = shared.SearchQuery

// Doc represents a document with vector embedding
type Doc struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Vector   []float32              `json:"vector,omitempty"`
}

// WeaviateClient interface for all implementations
type WeaviateClient interface {
	Search(ctx context.Context, query *SearchQuery) (*SearchResponse, error)
	IsHealthy() bool
	Close() error
}

// SearchResponse represents search results
type SearchResponse struct {
	Results     []*SearchResult        `json:"results"`
	Total       int                    `json:"total"`
	Took        time.Duration          `json:"took"`
	Query       string                 `json:"query"`
	ProcessedAt time.Time              `json:"processed_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RAGClientConfig holds configuration for RAG client
type RAGClientConfig struct {
	WeaviateHost     string        `json:"weaviate_host"`
	WeaviateScheme   string        `json:"weaviate_scheme"`
	APIKey           string        `json:"api_key"`
	Timeout          time.Duration `json:"timeout"`
	MaxRetries       int           `json:"max_retries"`
	EnableMetrics    bool          `json:"enable_metrics"`
	ConnectionPooled bool          `json:"connection_pooled"`
}

// RAGClient interface for all implementations
type RAGClient interface {
	Search(ctx context.Context, query string, options *SearchOptions) ([]*Doc, error)
	IsHealthy() bool
	Close() error
}

// SearchOptions provides options for search operations
type SearchOptions struct {
	Limit            int                    `json:"limit,omitempty"`
	Offset           int                    `json:"offset,omitempty"`
	MinConfidence    float64                `json:"min_confidence,omitempty"`
	UseHybridSearch  bool                   `json:"use_hybrid_search"`
	EnableReranking  bool                   `json:"enable_reranking"`
	Filters          map[string]interface{} `json:"filters,omitempty"`
}

// OptimizedRAGMetrics provides metrics for RAG operations
type OptimizedRAGMetrics struct {
	TotalRequests      int64         `json:"total_requests"`
	SuccessfulRequests int64         `json:"successful_requests"`
	FailedRequests     int64         `json:"failed_requests"`
	AverageLatency     time.Duration `json:"average_latency"`
	LastUpdated        time.Time     `json:"last_updated"`
}

// PoolMetrics provides connection pool metrics
type PoolMetrics struct {
	ActiveConnections    int32         `json:"active_connections"`
	IdleConnections      int32         `json:"idle_connections"`
	TotalConnections     int32         `json:"total_connections"`
	ConnectionsCreated   int64         `json:"connections_created"`
	ConnectionsDestroyed int64         `json:"connections_destroyed"`
	AverageWaitTime      time.Duration `json:"average_wait_time"`
	LastUpdated          time.Time     `json:"last_updated"`
}

// WeaviateConfig holds Weaviate connection configuration  
type WeaviateConfig struct {
	Host           string        `json:"host"`
	Scheme         string        `json:"scheme"`
	APIKey         string        `json:"api_key"`
	Timeout        time.Duration `json:"timeout"`
	MaxRetries     int           `json:"max_retries"`
	EnableMetrics  bool          `json:"enable_metrics"`
	ConnectionPool *PoolConfig   `json:"connection_pool,omitempty"`
}

// PoolConfig holds connection pool configuration
type PoolConfig struct {
	MaxConnections int           `json:"max_connections"`
	MinConnections int           `json:"min_connections"`
	MaxIdleTime    time.Duration `json:"max_idle_time"`
	MaxLifetime    time.Duration `json:"max_lifetime"`
}

// ConnectionPoolInterface interface for connection pooling
type ConnectionPoolInterface interface {
	GetClient() (WeaviateClient, error)
	ReleaseClient(client WeaviateClient)
	GetMetrics() *PoolMetrics
	Close() error
}

// WeaviateConnectionPool interface for backward compatibility
type WeaviateConnectionPool interface {
	GetClient() (WeaviateClient, error)
	ReleaseClient(client WeaviateClient) 
	GetMetrics() map[string]interface{}
	Close() error
}

// OptimizedRAGService interface for optimized RAG operations
type OptimizedRAGService interface {
	ProcessQuery(ctx context.Context, request *RAGRequest) (*RAGResponse, error)
	GetOptimizedMetrics() *OptimizedRAGMetrics
	IsHealthy() bool
	Close() error
}

// OptimizedRAGConfig holds configuration for optimized RAG service
type OptimizedRAGConfig struct {
	WeaviateConfig    *WeaviateConfig `json:"weaviate_config"`
	ConnectionPool    *PoolConfig     `json:"connection_pool"`
	SearchConfig      *SearchConfig   `json:"search_config"`
	CacheConfig       *CacheConfig    `json:"cache_config"`
	PerformanceConfig *PerfConfig     `json:"performance_config"`
}

// SearchConfig holds search-specific configuration
type SearchConfig struct {
	DefaultLimit       int     `json:"default_limit"`
	MaxLimit           int     `json:"max_limit"`
	MinConfidence      float64 `json:"min_confidence"`
	HybridAlpha        float64 `json:"hybrid_alpha"`
	EnableReranking    bool    `json:"enable_reranking"`
	RerankingThreshold float64 `json:"reranking_threshold"`
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	Enabled         bool          `json:"enabled"`
	TTL             time.Duration `json:"ttl"`
	MaxSize         int           `json:"max_size"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
}

// PerfConfig holds performance tuning configuration
type PerfConfig struct {
	MaxConcurrency     int           `json:"max_concurrency"`
	RequestTimeout     time.Duration `json:"request_timeout"`
	BatchSize          int           `json:"batch_size"`
	EnablePipelining   bool          `json:"enable_pipelining"`
	EnableMetrics      bool          `json:"enable_metrics"`
}