package shared

import (
	"context"
	"os"
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
	MaxTokens         int                    `json:"max_tokens"`
	SupportsChat      bool                   `json:"supports_chat"`
	SupportsFunction  bool                   `json:"supports_function"`
	SupportsStreaming bool                   `json:"supports_streaming"`
	CostPerToken      float64                `json:"cost_per_token"`
	Features          map[string]interface{} `json:"features"`
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
	Document *TelecomDocument       `json:"document"`
	Score    float32                `json:"score"`
	Distance float32                `json:"distance"`
	Metadata map[string]interface{} `json:"metadata"`
}

// SearchQuery represents a search query to the vector database
type SearchQuery struct {
	Query         string                 `json:"query"`
	Limit         int                    `json:"limit"`
	Filters       map[string]interface{} `json:"filters,omitempty"`
	HybridSearch  bool                   `json:"hybrid_search"`
	HybridAlpha   float32                `json:"hybrid_alpha"`
	UseReranker   bool                   `json:"use_reranker"`
	MinConfidence float32                `json:"min_confidence"`
	ExpandQuery   bool                   `json:"expand_query"`
}

// SearchResponse represents the response from a search operation
type SearchResponse struct {
	Results []*SearchResult `json:"results"`
	Took    int64           `json:"took"`
	Total   int64           `json:"total"`
}

// ComponentType represents different types of components in the system
type ComponentType string

const (
	ComponentTypeLLMProcessor       ComponentType = "llm-processor"
	ComponentTypeResourcePlanner    ComponentType = "resource-planner"
	ComponentTypeManifestGenerator  ComponentType = "manifest-generator"
	ComponentTypeGitOpsController   ComponentType = "gitops-controller"
	ComponentTypeDeploymentVerifier ComponentType = "deployment-verifier"
)

// ComponentStatus represents the status of a component
type ComponentStatus struct {
	Type       ComponentType          `json:"type"`
	Name       string                 `json:"name"`
	Status     string                 `json:"status"`
	Healthy    bool                   `json:"healthy"`
	LastUpdate time.Time              `json:"lastUpdate"`
	Version    string                 `json:"version,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Metrics    map[string]float64     `json:"metrics,omitempty"`
	Errors     []string               `json:"errors,omitempty"`
}

// SystemHealth represents the overall health of the system
type SystemHealth struct {
	OverallStatus  string                      `json:"overallStatus"`
	Healthy        bool                        `json:"healthy"`
	Components     map[string]*ComponentStatus `json:"components"`
	LastUpdate     time.Time                   `json:"lastUpdate"`
	ActiveIntents  int                         `json:"activeIntents"`
	ProcessingRate float64                     `json:"processingRate"`
	ErrorRate      float64                     `json:"errorRate"`
	ResourceUsage  ResourceUsage               `json:"resourceUsage"`
}

// ResourceUsage represents resource utilization
type ResourceUsage struct {
	CPUPercent        float64 `json:"cpuPercent"`
	MemoryPercent     float64 `json:"memoryPercent"`
	DiskPercent       float64 `json:"diskPercent"`
	NetworkInMBps     float64 `json:"networkInMBps"`
	NetworkOutMBps    float64 `json:"networkOutMBps"`
	ActiveConnections int     `json:"activeConnections"`
}

// ==== CONSOLIDATED TYPES TO RESOLVE REDECLARATIONS ====

// CircuitBreakerConfig holds configuration for circuit breaker (consolidated from pkg/llm)
type CircuitBreakerConfig struct {
	FailureThreshold    int64         `json:"failure_threshold"`
	FailureRate         float64       `json:"failure_rate"`
	MinimumRequestCount int64         `json:"minimum_request_count"`
	Timeout             time.Duration `json:"timeout"`
	HalfOpenTimeout     time.Duration `json:"half_open_timeout"`
	SuccessThreshold    int64         `json:"success_threshold"`
	HalfOpenMaxRequests int64         `json:"half_open_max_requests"`
	ResetTimeout        time.Duration `json:"reset_timeout"`
	SlidingWindowSize   int           `json:"sliding_window_size"`
	EnableHealthCheck   bool          `json:"enable_health_check"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	HealthCheckTimeout  time.Duration `json:"health_check_timeout"`
	// Advanced features
	EnableAdaptiveTimeout bool `json:"enable_adaptive_timeout"`
	MaxConcurrentRequests int  `json:"max_concurrent_requests"`
}

// BatchSearchRequest represents a batch of search requests (consolidated from pkg/rag)
type BatchSearchRequest struct {
	Queries           []*SearchQuery         `json:"queries"`
	MaxConcurrency    int                    `json:"max_concurrency"`
	EnableAggregation bool                   `json:"enable_aggregation"`
	DeduplicationKey  string                 `json:"deduplication_key"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// BatchSearchResponse represents the response from batch search (consolidated from pkg/rag)
type BatchSearchResponse struct {
	Results             []*SearchResponse      `json:"results"`
	AggregatedResults   []*SearchResult        `json:"aggregated_results"`
	TotalProcessingTime time.Duration          `json:"total_processing_time"`
	ParallelQueries     int                    `json:"parallel_queries"`
	CacheHits           int                    `json:"cache_hits"`
	Metadata            map[string]interface{} `json:"metadata"`
}

// EmbeddingCacheInterface defines the interface for embedding caches (consolidated from pkg/rag)
type EmbeddingCacheInterface interface {
	Get(key string) ([]float32, bool)
	Set(key string, embedding []float32, ttl time.Duration) error
	Delete(key string) error
	Clear() error
	Stats() EmbeddingCacheStats
}

// EmbeddingCacheStats represents embedding cache statistics (consolidated from pkg/rag)
type EmbeddingCacheStats struct {
	Size    int64   `json:"size"`
	Hits    int64   `json:"hits"`
	Misses  int64   `json:"misses"`
	HitRate float64 `json:"hit_rate"`
}

// EmbeddingCacheEntry represents a cached embedding entry (consolidated from pkg/rag)
type EmbeddingCacheEntry struct {
	Text        string        `json:"text"`
	Vector      []float32     `json:"vector"`
	CreatedAt   time.Time     `json:"created_at"`
	AccessCount int64         `json:"access_count"`
	TTL         time.Duration `json:"ttl"`
}

// GetEnv is a utility function to get environment variables with default values
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
