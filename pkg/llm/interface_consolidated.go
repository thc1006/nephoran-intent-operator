//go:build !disable_rag
// +build !disable_rag

package llm

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// CONSOLIDATED INTERFACES - Simplified from over-engineered abstractions

// LLMProcessor is the main interface for LLM processing
type LLMProcessor interface {
	ProcessIntent(ctx context.Context, request *ProcessingRequest) (*ProcessingResponse, error)
	GetMetrics() ClientMetrics
	Shutdown()
}

// Processor is an alias for backward compatibility
type Processor = LLMProcessor

// BatchProcessor handles batch processing of multiple intents
type BatchProcessor interface {
	ProcessRequest(ctx context.Context, intent, intentType, modelName string, priority Priority) (*BatchResult, error)
	GetStats() BatchProcessorStats
	Close() error
}

// StreamingProcessor handles streaming requests (concrete implementation for disable_rag builds)
type StreamingProcessor struct {
	// Stub implementation fields
}

// CacheProvider provides caching functionality
type CacheProvider interface {
	Get(key string) (string, bool)
	Set(key, response string)
	Clear()
	Stop()
	GetStats() map[string]interface{}
}

// PromptGenerator generates prompts for different intent types
type PromptGenerator interface {
	GeneratePrompt(intentType, userIntent string) string
	ExtractParameters(intent string) map[string]interface{}
}

// NOTE: The following types are defined in their respective files:
// - ClientMetrics: client_consolidated.go
// - Priority, BatchResult, BatchProcessorStats: batch_processor.go

// ProcessingRequest represents a request for LLM processing
type ProcessingRequest struct {
	ID                string                 `json:"id"`
	Intent            string                 `json:"intent"`
	IntentType        string                 `json:"intent_type,omitempty"`
	Context           string                 `json:"context,omitempty"`
	SystemPrompt      string                 `json:"system_prompt,omitempty"`
	UserPrompt        string                 `json:"user_prompt,omitempty"`
	ModelName         string                 `json:"model_name,omitempty"`
	Model             string                 `json:"model,omitempty"` // Alias for ModelName
	MaxTokens         int                    `json:"max_tokens,omitempty"`
	Temperature       float32                `json:"temperature,omitempty"`
	Priority          Priority               `json:"priority,omitempty"`
	RequestID         string                 `json:"request_id,omitempty"`
	ProcessingTimeout time.Duration          `json:"processing_timeout,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// ProcessingResponse represents a response from LLM processing
type ProcessingResponse struct {
	ID                  string                 `json:"id"`
	Response            string                 `json:"response"`
	ProcessedParameters string                 `json:"processed_parameters,omitempty"` // JSON string of parameters
	Confidence          float32                `json:"confidence"`
	TokensUsed          int                    `json:"tokens_used"`
	Cost                float64                `json:"cost"`
	ProcessingTime      time.Duration          `json:"processing_time"`
	ModelUsed           string                 `json:"model_used"`
	CacheHit            bool                   `json:"cache_hit"`
	IntentType          string                 `json:"intent_type,omitempty"`
	ExtractedIntent     string                 `json:"extracted_intent,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
}

// StreamingRequest represents a request for streaming LLM processing
type StreamingRequest struct {
	Query       string                 `json:"query"`
	Context     string                 `json:"context,omitempty"`
	ModelName   string                 `json:"model_name,omitempty"`
	Stream      bool                   `json:"stream"`
	SessionID   string                 `json:"session_id,omitempty"`
	EnableRAG   bool                   `json:"enable_rag"`
	IntentType  string                 `json:"intent_type,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Temperature float32                `json:"temperature,omitempty"`
	ClientID    string                 `json:"client_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// WeaviateConnectionPool is a stub type for the connection pool
type WeaviateConnectionPool struct {
	// Stub implementation - no actual fields needed
}

// ContextBuilder provides context building functionality for RAG systems
type ContextBuilder struct {
	weaviatePool *WeaviateConnectionPool // Using our type alias
	config       *ContextBuilderConfig
	logger       *slog.Logger // Using concrete type instead of interface{}
	metrics      *ContextBuilderMetrics
	mutex        sync.RWMutex
}

// ContextBuilderConfig holds configuration for context building
type ContextBuilderConfig struct {
	DefaultMaxDocs        int           `json:"default_max_docs"`
	MaxContextLength      int           `json:"max_context_length"`
	MinConfidenceScore    float32       `json:"min_confidence_score"`
	QueryTimeout          time.Duration `json:"query_timeout"`
	EnableHybridSearch    bool          `json:"enable_hybrid_search"`
	HybridAlpha           float32       `json:"hybrid_alpha"`
	TelecomKeywords       []string      `json:"telecom_keywords"`
	QueryExpansionEnabled bool          `json:"query_expansion_enabled"`
}

// ContextBuilderMetrics tracks context building performance
type ContextBuilderMetrics struct {
	TotalQueries          int64         `json:"total_queries"`
	SuccessfulQueries     int64         `json:"successful_queries"`
	FailedQueries         int64         `json:"failed_queries"`
	AverageQueryDuration  time.Duration `json:"average_query_duration"`
	AverageDocumentsFound int           `json:"average_documents_found"`
	CacheHits             int64         `json:"cache_hits"`
	CacheMisses           int64         `json:"cache_misses"`
	TotalLatency          time.Duration `json:"total_latency"`
	mutex                 sync.RWMutex
}

// RelevanceScorer provides relevance scoring functionality
type RelevanceScorer struct {
	config          *RelevanceScorerConfig
	logger          *slog.Logger // Using concrete type instead of interface{}
	embeddings      interface{}  // rag.EmbeddingServiceInterface
	domainKnowledge interface{}  // *TelecomDomainKnowledge
	metrics         *ScoringMetrics
	mutex           sync.RWMutex
}

// RelevanceScorerConfig holds configuration for relevance scoring
type RelevanceScorerConfig struct {
	// Scoring weights
	SemanticWeight        float64 `json:"semantic_weight"`
	AuthorityWeight       float64 `json:"authority_weight"`
	RecencyWeight         float64 `json:"recency_weight"`
	DomainWeight          float64 `json:"domain_weight"`
	IntentAlignmentWeight float64 `json:"intent_alignment_weight"`

	// Additional configuration fields
	MinSemanticSimilarity float64            `json:"min_semantic_similarity"`
	UseEmbeddingDistance  bool               `json:"use_embedding_distance"`
	AuthorityScores       map[string]float64 `json:"authority_scores"`
	StandardsMultiplier   float64            `json:"standards_multiplier"`
	RecencyHalfLife       time.Duration      `json:"recency_half_life"`
	MaxAge                time.Duration      `json:"max_age"`
	CacheScores           bool               `json:"cache_scores"`
	ScoreCacheTTL         time.Duration      `json:"score_cache_ttl"`
	ParallelProcessing    bool               `json:"parallel_processing"`
	MaxProcessingTime     time.Duration      `json:"max_processing_time"`
}

// ScoringMetrics tracks scoring performance
type ScoringMetrics struct {
	TotalScores        int64         `json:"total_scores"`
	AverageScoringTime time.Duration `json:"average_scoring_time"`
	CacheHitRate       float64       `json:"cache_hit_rate"`
	SemanticScores     int64         `json:"semantic_scores"`
	AuthorityScores    int64         `json:"authority_scores"`
	RecencyScores      int64         `json:"recency_scores"`
	DomainScores       int64         `json:"domain_scores"`
	IntentScores       int64         `json:"intent_scores"`
	LastUpdated        time.Time     `json:"last_updated"`
	mutex              sync.RWMutex
}

// SimpleRelevanceScorer provides a simple relevance scoring implementation
type SimpleRelevanceScorer struct {
	embeddingService interface{} // rag.EmbeddingServiceInterface
	legacyEmbedding  interface{} // *rag.EmbeddingService
	logger           *slog.Logger
	metrics          *SimpleRelevanceScorerMetrics
	mutex            sync.RWMutex
}

// SimpleRelevanceScorerMetrics tracks simple scoring performance
type SimpleRelevanceScorerMetrics struct {
	TotalScores      int64         `json:"total_scores"`
	SuccessfulScores int64         `json:"successful_scores"`
	FailedScores     int64         `json:"failed_scores"`
	AverageLatency   time.Duration `json:"average_latency"`
	EmbeddingCalls   int64         `json:"embedding_calls"`
	FallbackUses     int64         `json:"fallback_uses"`
	LastUpdated      time.Time     `json:"last_updated"`
	mutex            sync.RWMutex
}

// Additional types needed by various components

// SimpleTokenTracker tracks token usage statistics
type SimpleTokenTracker struct {
	totalTokens  int64
	totalCost    float64
	requestCount int64
	mutex        sync.RWMutex
}

// NewSimpleTokenTracker creates a new token tracker
func NewSimpleTokenTracker() *SimpleTokenTracker {
	return &SimpleTokenTracker{}
}

// RecordUsage records token usage
func (tt *SimpleTokenTracker) RecordUsage(tokens int) {
	tt.mutex.Lock()
	defer tt.mutex.Unlock()
	tt.totalTokens += int64(tokens)
	tt.requestCount++
	// Simple cost calculation (adjust based on model pricing)
	costPerToken := 0.0001 // Example: $0.0001 per token
	tt.totalCost += float64(tokens) * costPerToken
}

// GetStats returns usage statistics
func (tt *SimpleTokenTracker) GetStats() map[string]interface{} {
	tt.mutex.RLock()
	defer tt.mutex.RUnlock()
	avgTokensPerRequest := float64(0)
	if tt.requestCount > 0 {
		avgTokensPerRequest = float64(tt.totalTokens) / float64(tt.requestCount)
	}
	return map[string]interface{}{
		"total_tokens":           tt.totalTokens,
		"total_cost":             tt.totalCost,
		"request_count":          tt.requestCount,
		"avg_tokens_per_request": avgTokensPerRequest,
	}
}

// RequestContext contains context for LLM requests
type RequestContext struct {
	ID        string                 // Unique identifier for this context
	RequestID string                 // Request ID
	UserID    string                 // User identifier
	SessionID string                 // Session identifier
	Priority  int                    // Request priority
	Intent    string                 // User intent
	Metadata  map[string]interface{} // Additional metadata
	StartTime time.Time              // Request start time
	Deadline  time.Time              // Request deadline
}

// HealthChecker performs health checks on endpoints
type HealthChecker interface {
	CheckHealth(ctx context.Context, endpoint string) error
}

// EndpointPool manages a pool of service endpoints
type EndpointPool interface {
	GetHealthyEndpoint() (string, error)
	ReportError(endpoint string, err error)
	GetAllEndpoints() []string
}

// BatchProcessorConfig contains batch processor configuration
type BatchProcessorConfig struct {
	MaxBatchSize      int
	MaxWaitTime       time.Duration
	MaxWorkers        int
	QueueSize         int
	RetryAttempts     int
	RetryDelay        time.Duration
	Priority          int
	ProcessingTimeout time.Duration
}

// TokenManager manages token counting and limits
type TokenManager struct {
	maxTokens     int
	tokensPerWord float64
	mutex         sync.RWMutex
}

// NewTokenManager creates a new token manager
func NewTokenManager() *TokenManager {
	return &TokenManager{
		maxTokens:     8192,
		tokensPerWord: 1.3, // Average tokens per word
	}
}

// CountTokens estimates token count from text
func (tm *TokenManager) CountTokens(text string) int {
	// Simple approximation: count words and multiply by average tokens per word
	words := len(strings.Fields(text))
	return int(float64(words) * tm.tokensPerWord)
}

// EstimateTokensForModel estimates tokens for a specific model
func (tm *TokenManager) EstimateTokensForModel(text string, model string) int {
	// For now, use the same estimation for all models
	return tm.CountTokens(text)
}

// SupportsSystemPrompt checks if model supports system prompts
func (tm *TokenManager) SupportsSystemPrompt(model string) bool {
	// Most modern models support system prompts
	return true
}

// SupportsChatFormat checks if model supports chat format
func (tm *TokenManager) SupportsChatFormat(model string) bool {
	// Most modern models support chat format
	return true
}

// TruncateToFit truncates text to fit within token limit
func (tm *TokenManager) TruncateToFit(text string, maxTokens int, model string) string {
	// Model parameter is for compatibility, using same logic for all models
	tokens := tm.CountTokens(text)
	if tokens <= maxTokens {
		return text
	}
	// Simple truncation by character ratio
	ratio := float64(maxTokens) / float64(tokens)
	targetLen := int(float64(len(text)) * ratio * 0.95) // 95% to ensure we're under limit
	if targetLen > len(text) {
		return text
	}
	return text[:targetLen] + "..."
}

// SupportsStreaming checks if model supports streaming
func (tm *TokenManager) SupportsStreaming(model string) bool {
	// Most modern models support streaming
	return true
}

// GetSupportedModels returns list of supported models
func (tm *TokenManager) GetSupportedModels() []string {
	return []string{
		"gpt-3.5-turbo",
		"gpt-4",
		"claude-2",
		"claude-3",
		"llama2",
		"mistral",
	}
}

// StreamingContextManager manages streaming context
type StreamingContextManager struct {
	contexts map[string]interface{}
	mutex    sync.RWMutex
}

// NewStreamingContextManager creates a new streaming context manager
func NewStreamingContextManager(tokenManager *TokenManager, contextOverhead time.Duration) *StreamingContextManager {
	// Parameters are for compatibility but not used in stub implementation
	return &StreamingContextManager{
		contexts: make(map[string]interface{}),
	}
}

// Close closes the streaming context manager
func (scm *StreamingContextManager) Close() {
	// No resources to clean up in stub implementation
}

// GetMetrics returns metrics for the ContextBuilder
func (cb *ContextBuilder) GetMetrics() map[string]interface{} {
	if cb == nil || cb.metrics == nil {
		return map[string]interface{}{
			"status": "disabled",
		}
	}

	cb.metrics.mutex.RLock()
	defer cb.metrics.mutex.RUnlock()

	return map[string]interface{}{
		"total_queries":           cb.metrics.TotalQueries,
		"successful_queries":      cb.metrics.SuccessfulQueries,
		"failed_queries":          cb.metrics.FailedQueries,
		"average_query_duration":  cb.metrics.AverageQueryDuration.String(),
		"average_documents_found": cb.metrics.AverageDocumentsFound,
		"cache_hits":              cb.metrics.CacheHits,
		"cache_misses":            cb.metrics.CacheMisses,
		"total_latency":           cb.metrics.TotalLatency.String(),
	}
}

// Document represents a document for context building
type Document struct {
	ID       string                 `json:"id"`
	Title    string                 `json:"title"`
	Content  string                 `json:"content"`
	Source   string                 `json:"source"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// GetMetrics returns metrics for the RelevanceScorer
func (rs *RelevanceScorer) GetMetrics() map[string]interface{} {
	if rs == nil || rs.metrics == nil {
		return map[string]interface{}{
			"status": "disabled",
		}
	}

	rs.metrics.mutex.RLock()
	defer rs.metrics.mutex.RUnlock()

	return map[string]interface{}{
		"total_scores":         rs.metrics.TotalScores,
		"average_scoring_time": rs.metrics.AverageScoringTime.String(),
		"cache_hit_rate":       rs.metrics.CacheHitRate,
		"semantic_scores":      rs.metrics.SemanticScores,
		"authority_scores":     rs.metrics.AuthorityScores,
		"recency_scores":       rs.metrics.RecencyScores,
		"domain_scores":        rs.metrics.DomainScores,
		"intent_scores":        rs.metrics.IntentScores,
		"last_updated":         rs.metrics.LastUpdated.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// IntentRequest represents a legacy request structure (backward compatibility)
type IntentRequest = ProcessingRequest

// IntentResponse represents a legacy response structure (backward compatibility)
type IntentResponse = ProcessingResponse
