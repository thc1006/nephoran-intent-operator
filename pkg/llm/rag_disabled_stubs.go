//go:build stubs

package llm

import (
	"context"
	"time"
)

// Stub types for when RAG is disabled

// MetricsIntegrator stub
type MetricsIntegrator struct{}

func NewMetricsIntegrator(collector interface{}) *MetricsIntegrator {
	return &MetricsIntegrator{}
}

// MetricsCollector stub
type MetricsCollector struct{}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{}
}

// Priority stub for batch processing
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// BatchResult stub
type BatchResult struct {
	Index   int         `json:"index"`
	Result  interface{} `json:"result"`
	Error   error       `json:"error"`
	Latency time.Duration `json:"latency"`
}

// BatchProcessorStats stub
type BatchProcessorStats struct {
	TotalProcessed   int64         `json:"total_processed"`
	SuccessfulBatch  int64         `json:"successful_batch"`
	FailedBatch      int64         `json:"failed_batch"`
	AverageLatency   time.Duration `json:"average_latency"`
	QueueSize        int           `json:"queue_size"`
	ActiveWorkers    int           `json:"active_workers"`
	LastProcessedAt  time.Time     `json:"last_processed_at"`
}

// WeaviateConnectionPool stub
type WeaviateConnectionPool struct{}

// EmbeddingServiceInterface stub
type EmbeddingServiceInterface interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}

// EmbeddingService stub
type EmbeddingService struct{}

func (e *EmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	return nil, nil
}

// WeaviateClient stub
type WeaviateClient struct{}

// TelecomDocument stub
type TelecomDocument struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Content     string            `json:"content"`
	Source      string            `json:"source"`
	Technology  []string          `json:"technology"`
	Version     string            `json:"version,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Embedding   []float32         `json:"embedding,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// SearchResponse stub
type SearchResponse struct {
	Results     []*SearchResult `json:"results"`
	Total       int             `json:"total"`
	Query       string          `json:"query"`
	ProcessedAt time.Time       `json:"processed_at"`
}

// SearchResult stub
type SearchResult struct {
	Document *TelecomDocument `json:"document"`
	Score    float32          `json:"score"`
	Snippet  string           `json:"snippet,omitempty"`
}

// RAGService stub
type RAGService struct{}

// RAGRequest stub
type RAGRequest struct {
	Query             string                 `json:"query"`
	IntentType        string                 `json:"intent_type,omitempty"`
	MaxResults        int                    `json:"max_results"`
	MinConfidence     float32                `json:"min_confidence"`
	UseHybridSearch   bool                   `json:"use_hybrid_search"`
	EnableReranking   bool                   `json:"enable_reranking"`
	IncludeSourceRefs bool                   `json:"include_source_refs"`
	SearchFilters     map[string]interface{} `json:"search_filters,omitempty"`
}

// RAGResponse stub
type RAGResponse struct {
	Answer          string          `json:"answer"`
	Confidence      float32         `json:"confidence"`
	SourceDocuments []*SearchResult `json:"source_documents"`
	RetrievalTime   time.Duration   `json:"retrieval_time"`
	GenerationTime  time.Duration   `json:"generation_time"`
	UsedCache       bool            `json:"used_cache"`
}

// SearchQuery stub
type SearchQuery struct {
	Query       string                 `json:"query"`
	MaxResults  int                    `json:"max_results"`
	MinScore    float32                `json:"min_score"`
	Filters     map[string]interface{} `json:"filters,omitempty"`
}

// SimpleTokenTracker stub
type SimpleTokenTracker struct{}

func NewSimpleTokenTracker() *SimpleTokenTracker {
	return &SimpleTokenTracker{}
}

// CircuitBreaker stub
type CircuitBreaker struct{}

func (cb *CircuitBreaker) Execute(fn func() error) error {
	return fn()
}

// CircuitBreakerConfig stub  
type CircuitBreakerConfig struct {
	FailureThreshold int64         `json:"failure_threshold"`
	Timeout          time.Duration `json:"timeout"`
}

func NewCircuitBreaker(name string, config *CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{}
}

// ResponseCache stub
type ResponseCache struct{}

func NewResponseCache(ttl time.Duration, maxSize int) *ResponseCache {
	return &ResponseCache{}
}

func (rc *ResponseCache) Get(key string) (string, bool) {
	return "", false
}

func (rc *ResponseCache) Set(key, value string) {}

func (rc *ResponseCache) Stop() {}

// CircuitState stub
type CircuitState int

const (
	CircuitStateClosed CircuitState = iota
	CircuitStateOpen
	CircuitStateHalfOpen
)

// CircuitBreakerError stub
type CircuitBreakerError struct {
	State CircuitState
	Err   error
}

func (e *CircuitBreakerError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return "circuit breaker error"
}

// Additional stub types for handlers compatibility
type StreamingRequest struct {
	Query      string `json:"query"`
	ModelName  string `json:"model_name"`
	MaxTokens  int    `json:"max_tokens"`
	EnableRAG  bool   `json:"enable_rag"`
}

type CircuitBreakerManager struct{}

func (cbm *CircuitBreakerManager) GetAllStats() map[string]interface{} {
	return make(map[string]interface{})
}

func (cbm *CircuitBreakerManager) Get(name string) *CircuitBreaker {
	return &CircuitBreaker{}
}

type TokenManager struct{}

func (tm *TokenManager) GetSupportedModels() []string {
	return []string{}
}

type ContextBuilder struct{}

func (cb *ContextBuilder) GetMetrics() map[string]interface{} {
	return make(map[string]interface{})
}

type RelevanceScorer struct{}

func (rs *RelevanceScorer) GetMetrics() map[string]interface{} {
	return make(map[string]interface{})
}

type RAGAwarePromptBuilder struct{}

func (pb *RAGAwarePromptBuilder) GetMetrics() map[string]interface{} {
	return make(map[string]interface{})
}

type RAGEnhancedProcessor struct{}

func (rep *RAGEnhancedProcessor) ProcessIntent(ctx context.Context, intent string) (string, error) {
	return "RAG is disabled", nil
}

// Stub constructors
func NewCircuitBreakerManager() *CircuitBreakerManager {
	return &CircuitBreakerManager{}
}

func NewTokenManager() *TokenManager {
	return &TokenManager{}
}

func NewContextBuilder() *ContextBuilder {
	return &ContextBuilder{}
}

func NewRelevanceScorer() *RelevanceScorer {
	return &RelevanceScorer{}
}

func NewRAGAwarePromptBuilder(tokenManager *TokenManager, config interface{}) *RAGAwarePromptBuilder {
	return &RAGAwarePromptBuilder{}
}

func NewRAGEnhancedProcessor(client *Client, weaviateClient interface{}, ragService interface{}, config interface{}) *RAGEnhancedProcessor {
	return &RAGEnhancedProcessor{}
}

