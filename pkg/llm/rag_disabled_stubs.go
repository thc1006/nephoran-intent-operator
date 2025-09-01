//go:build stub && !disable_rag

package llm

import (
	"context"
	"time"
)

// Stub types for when RAG is disabled.

// StubMetricsIntegrator stub.

type StubMetricsIntegrator struct{}

// NewMetricsIntegrator performs newmetricsintegrator operation.

func NewStubMetricsIntegrator(collector interface{}) *StubMetricsIntegrator {

	return &StubMetricsIntegrator{}

}

// MetricsCollector stub.

type MetricsCollector struct{}

// NewMetricsCollector performs newmetricscollector operation.

func NewMetricsCollector() *MetricsCollector {

	return &MetricsCollector{}

}

// Priority stub for batch processing.

type Priority int

const (

	// PriorityLow holds prioritylow value.

	PriorityLow Priority = iota

	// PriorityNormal holds prioritynormal value.

	PriorityNormal

	// PriorityHigh holds priorityhigh value.

	PriorityHigh

	// PriorityCritical holds prioritycritical value.

	PriorityCritical
)

// BatchResult stub.

type BatchResult struct {
	Index int `json:"index"`

	Result interface{} `json:"result"`

	Error error `json:"error"`

	Latency time.Duration `json:"latency"`
}

// StubBatchProcessorStats stub.

type StubBatchProcessorStats struct {
	TotalProcessed int64 `json:"total_processed"`

	SuccessfulBatch int64 `json:"successful_batch"`

	FailedBatch int64 `json:"failed_batch"`

	AverageLatency time.Duration `json:"average_latency"`

	QueueSize int `json:"queue_size"`

	ActiveWorkers int `json:"active_workers"`

	LastProcessedAt time.Time `json:"last_processed_at"`
}

// WeaviateConnectionPool stub.

type WeaviateConnectionPool struct{}

// EmbeddingServiceInterface stub.

type EmbeddingServiceInterface interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}

// EmbeddingService stub.

type EmbeddingService struct{}

// GenerateEmbedding performs generateembedding operation.

func (e *EmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {

	return nil, nil

}

// WeaviateClient stub.

type WeaviateClient struct{}

// TelecomDocument stub.

type TelecomDocument struct {
	ID string `json:"id"`

	Title string `json:"title"`

	Content string `json:"content"`

	Source string `json:"source"`

	Technology []string `json:"technology"`

	Version string `json:"version,omitempty"`

	Metadata map[string]interface{} `json:"metadata,omitempty"`

	Embedding []float32 `json:"embedding,omitempty"`

	CreatedAt time.Time `json:"created_at"`

	UpdatedAt time.Time `json:"updated_at"`
}

// SearchResponse stub.

type SearchResponse struct {
	Results []*SearchResult `json:"results"`

	Total int `json:"total"`

	Query string `json:"query"`

	ProcessedAt time.Time `json:"processed_at"`
}

// SearchResult stub.

type SearchResult struct {
	Document *TelecomDocument `json:"document"`

	Score float32 `json:"score"`

	Snippet string `json:"snippet,omitempty"`
}

// RAGService stub.

type RAGService struct{}

// RAGRequest stub.

type RAGRequest struct {
	Query string `json:"query"`

	IntentType string `json:"intent_type,omitempty"`

	MaxResults int `json:"max_results"`

	MinConfidence float32 `json:"min_confidence"`

	UseHybridSearch bool `json:"use_hybrid_search"`

	EnableReranking bool `json:"enable_reranking"`

	IncludeSourceRefs bool `json:"include_source_refs"`

	SearchFilters map[string]interface{} `json:"search_filters,omitempty"`
}

// RAGResponse stub.

type RAGResponse struct {
	Answer string `json:"answer"`

	Confidence float32 `json:"confidence"`

	SourceDocuments []*SearchResult `json:"source_documents"`

	RetrievalTime time.Duration `json:"retrieval_time"`

	GenerationTime time.Duration `json:"generation_time"`

	UsedCache bool `json:"used_cache"`
}

// SearchQuery stub.

type SearchQuery struct {
	Query string `json:"query"`

	MaxResults int `json:"max_results"`

	MinScore float32 `json:"min_score"`

	Filters map[string]interface{} `json:"filters,omitempty"`
}

// SimpleTokenTracker stub.

type StubSimpleTokenTracker struct{}

// NewSimpleTokenTracker performs newsimpletokentracker operation.

func NewStubSimpleTokenTracker() *StubSimpleTokenTracker {

	return &StubSimpleTokenTracker{}

}

// CircuitBreaker stub.

type CircuitBreaker struct{}

// Execute performs execute operation.

func (cb *CircuitBreaker) Execute(fn func() error) error {

	return fn()

}

// CircuitBreakerConfig stub.

type CircuitBreakerConfig struct {
	FailureThreshold int64 `json:"failure_threshold"`

	Timeout time.Duration `json:"timeout"`
}

// NewCircuitBreaker performs newcircuitbreaker operation.

func NewCircuitBreaker(name string, config *CircuitBreakerConfig) *CircuitBreaker {

	return &CircuitBreaker{}

}

// ResponseCache stub.

type ResponseCache struct{}

// NewResponseCache performs newresponsecache operation.

func NewResponseCache(ttl time.Duration, maxSize int) *ResponseCache {

	return &ResponseCache{}

}

// Get performs get operation.

func (rc *ResponseCache) Get(key string) (string, bool) {

	return "", false

}

// Set performs set operation.

func (rc *ResponseCache) Set(key, value string) {}

// Stop performs stop operation.

func (rc *ResponseCache) Stop() {}

// CircuitState stub.

type CircuitState int

const (

	// CircuitStateClosed holds circuitstateclosed value.

	CircuitStateClosed CircuitState = iota

	// CircuitStateOpen holds circuitstateopen value.

	CircuitStateOpen

	// CircuitStateHalfOpen holds circuitstatehalfopen value.

	CircuitStateHalfOpen
)

// CircuitBreakerError stub.

type CircuitBreakerError struct {
	State CircuitState

	Err error
}

// Error performs error operation.

func (e *CircuitBreakerError) Error() string {

	if e.Err != nil {

		return e.Err.Error()

	}

	return "circuit breaker error"

}

// Additional stub types for handlers compatibility.

type StreamingRequest struct {
	Query string `json:"query"`

	ModelName string `json:"model_name"`

	MaxTokens int `json:"max_tokens"`

	EnableRAG bool `json:"enable_rag"`
}

// CircuitBreakerManager represents a circuitbreakermanager.

type CircuitBreakerManager struct{}

// GetAllStats performs getallstats operation.

func (cbm *CircuitBreakerManager) GetAllStats() map[string]interface{} {

	return make(map[string]interface{})

}

// Get performs get operation.

func (cbm *CircuitBreakerManager) Get(name string) *CircuitBreaker {

	return &CircuitBreaker{}

}

// TokenManager represents a tokenmanager.

type TokenManager struct{}

// AllocateTokens performs allocatetokens operation.
func (tm *TokenManager) AllocateTokens(request string) (int, error) {
	return 0, nil
}

// ReleaseTokens performs releasetokens operation.
func (tm *TokenManager) ReleaseTokens(count int) error {
	return nil
}

// GetAvailableTokens performs getavailabletokens operation.
func (tm *TokenManager) GetAvailableTokens() int {
	return 0
}

// EstimateTokensForModel performs estimatetokensformodel operation.
func (tm *TokenManager) EstimateTokensForModel(model string, text string) (int, error) {
	return 0, nil
}

// SupportsSystemPrompt performs supportssystemprompt operation.
func (tm *TokenManager) SupportsSystemPrompt(model string) bool {
	return false
}

// SupportsChatFormat performs supportschatformat operation.
func (tm *TokenManager) SupportsChatFormat(model string) bool {
	return false
}

// SupportsStreaming performs supportsstreaming operation.
func (tm *TokenManager) SupportsStreaming(model string) bool {
	return false
}

// TruncateToFit performs truncatetofit operation.
func (tm *TokenManager) TruncateToFit(text string, maxTokens int, model string) (string, error) {
	return text, nil
}

// GetTokenCount performs gettokencount operation.
func (tm *TokenManager) GetTokenCount(text string) int {
	return 0
}

// ValidateModel performs validatemodel operation.
func (tm *TokenManager) ValidateModel(model string) error {
	return nil
}

// GetSupportedModels performs getsupportedmodels operation.

func (tm *TokenManager) GetSupportedModels() []string {

	return []string{}

}

// StubContextBuilder represents a contextbuilder.

type StubContextBuilder struct{}

// GetMetrics performs getmetrics operation.

func (cb *StubContextBuilder) GetMetrics() map[string]interface{} {

	return make(map[string]interface{})

}

// RelevanceScorer represents a relevancescorer.

type StubRelevanceScorer struct{}

// GetMetrics performs getmetrics operation.

func (rs *StubRelevanceScorer) GetMetrics() map[string]interface{} {

	return make(map[string]interface{})

}

// RAGAwarePromptBuilder represents a ragawarepromptbuilder.

type RAGAwarePromptBuilder struct{}

// GetMetrics performs getmetrics operation.

func (pb *RAGAwarePromptBuilder) GetMetrics() map[string]interface{} {

	return make(map[string]interface{})

}

// RAGEnhancedProcessor represents a ragenhancedprocessor.

type RAGEnhancedProcessor struct{}

// ProcessIntent performs processintent operation.

func (rep *RAGEnhancedProcessor) ProcessIntent(ctx context.Context, intent string) (string, error) {

	return "RAG is disabled", nil

}

// Stub constructors.

func NewCircuitBreakerManager() *CircuitBreakerManager {

	return &CircuitBreakerManager{}

}

// NewTokenManagerStub creates a stub token manager
// Note: The main NewTokenManager is provided by token_manager_default.go

func NewTokenManagerStub() TokenManager {

	return NewTokenManager()

}

// NewContextBuilder performs newcontextbuilder operation.

func NewContextBuilder() *ContextBuilder {

	return &ContextBuilder{}

}

// NewRelevanceScorer performs newrelevancescorer operation.

func NewRelevanceScorer() *RelevanceScorer {

	return &RelevanceScorer{}

}

// NewRAGAwarePromptBuilder performs newragawarepromptbuilder operation.

func NewRAGAwarePromptBuilder(tokenManager TokenManager, config interface{}) *RAGAwarePromptBuilder {

	return &RAGAwarePromptBuilder{}

}

// NewRAGEnhancedProcessor performs newragenhancedprocessor operation.

func NewRAGEnhancedProcessor(client *Client, weaviateClient interface{}, ragService interface{}, config interface{}) *RAGEnhancedProcessor {

	return &RAGEnhancedProcessor{}

}
