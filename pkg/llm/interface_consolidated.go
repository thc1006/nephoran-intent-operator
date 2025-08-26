package llm

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CONSOLIDATED INTERFACES - Simplified from over-engineered abstractions

// LLMProcessor is the main interface for LLM processing
type LLMProcessor interface {
	ProcessIntent(ctx context.Context, intent string) (string, error)
	GetMetrics() ClientMetrics
	Shutdown()
}

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

// ESSENTIAL TYPES ONLY - Consolidated from scattered definitions

// Note: StreamingRequest and BatchRequest types are defined in their respective implementation files

// Note: ProcessingResult, ProcessingMetrics, MetricsCollector, MetricsIntegrator types are defined in their respective files

// Note: MetricsIntegrator methods are defined in prometheus_metrics.go

// Missing type definitions for compilation
type RequestContext struct {
	ID        string
	Intent    string
	StartTime time.Time
	Metadata  map[string]interface{}
}

type HealthChecker struct {
	// Stub implementation
}

type EndpointPool struct {
	// Stub implementation
}

type BatchProcessorConfig struct {
	// Stub implementation
}

type TokenManager struct {
	// Stub implementation
	maxTokens int
}

func NewTokenManager() *TokenManager {
	return &TokenManager{maxTokens: 4096}
}

func (tm *TokenManager) GetSupportedModels() []string { return []string{"gpt-4o-mini"} }

func (tm *TokenManager) CountTokens(text string) int {
	// Simple estimation: ~4 characters per token
	return len(text) / 4
}

func (tm *TokenManager) CalculateTokenBudget(ctx context.Context, model, systemPrompt, userPrompt, context string) (TokenBudget, error) {
	systemTokens := tm.CountTokens(systemPrompt)
	userTokens := tm.CountTokens(userPrompt)
	contextTokens := tm.CountTokens(context)
	totalInput := systemTokens + userTokens + contextTokens
	availableOutput := tm.maxTokens - totalInput - 100 // Reserve 100 tokens
	
	if availableOutput < 0 {
		availableOutput = 0
	}
	
	return TokenBudget{
		SystemTokens:    systemTokens,
		UserTokens:      userTokens,
		ContextTokens:   contextTokens,
		TotalInput:      totalInput,
		AvailableOutput: availableOutput,
		MaxTokens:       tm.maxTokens,
	}, nil
}

func (tm *TokenManager) OptimizeContext(contexts []string, maxTokens int, model string) []string {
	var result []string
	currentTokens := 0
	
	for _, ctx := range contexts {
		tokens := tm.CountTokens(ctx)
		if currentTokens+tokens <= maxTokens {
			result = append(result, ctx)
			currentTokens += tokens
		} else {
			break
		}
	}
	
	return result
}

// EstimateTokensForModel estimates tokens for a specific model
func (tm *TokenManager) EstimateTokensForModel(text, modelName string) int {
	// For now, use the same estimation regardless of model
	// This could be enhanced to have model-specific token counting
	return tm.CountTokens(text)
}

// SupportsSystemPrompt checks if the model supports system prompts
func (tm *TokenManager) SupportsSystemPrompt(modelName string) bool {
	// Most modern models support system prompts
	// Could be enhanced with a model-specific capability map
	return true
}

// SupportsChatFormat checks if the model supports chat format
func (tm *TokenManager) SupportsChatFormat(modelName string) bool {
	// Most modern models support chat format
	return true
}

// TruncateToFit truncates text to fit within token limits
func (tm *TokenManager) TruncateToFit(text string, maxTokens int, modelName string) string {
	currentTokens := tm.EstimateTokensForModel(text, modelName)
	if currentTokens <= maxTokens {
		return text
	}
	
	// Rough approximation: truncate proportionally
	ratio := float64(maxTokens) / float64(currentTokens)
	newLength := int(float64(len(text)) * ratio)
	
	if newLength < len(text) {
		return text[:newLength]
	}
	return text
}

type TokenBudget struct {
	SystemTokens    int
	UserTokens      int
	ContextTokens   int
	TotalInput      int
	AvailableOutput int
	MaxTokens       int
}

type StreamingContextManager struct {
	// Stub implementation
}

type Document struct {
	ID       string                 `json:"id"`
	Title    string                 `json:"title"`
	Content  string                 `json:"content"`
	Source   string                 `json:"source"`
	Metadata map[string]interface{} `json:"metadata"`
}

// ContextBuilder is defined in clean_stubs.go

func (cb *ContextBuilder) BuildContext(ctx context.Context, query string, documents []Document) (string, error) {
	scores, err := cb.CalculateRelevanceScores(ctx, query, documents)
	if err != nil {
		return "", err
	}
	
	var context string
	currentTokens := 0
	
	for i, doc := range documents {
		if scores[i] >= float64(cb.config.MinConfidenceScore) {
			docContent := doc.Title + "\n" + doc.Content
			docTokens := cb.tokenManager.CountTokens(docContent)
			
			if currentTokens+docTokens <= cb.config.MaxContextLength {
				if context != "" {
					context += "\n---\n"
				}
				context += docContent
				currentTokens += docTokens
			} else {
				break
			}
		}
	}
	
	return context, nil
}

func (cb *ContextBuilder) CalculateRelevanceScores(ctx context.Context, query string, documents []Document) ([]float64, error) {
	scores := make([]float64, len(documents))
	
	for i, doc := range documents {
		// Simple scoring based on title and content matching
		score := 0.0
		if len(doc.Title) > 0 && len(query) > 0 {
			score = 0.5 + float64(len(doc.Content)%100)/200.0 // Mock relevance score
		}
		scores[i] = score
	}
	
	return scores, nil
}

// CircuitBreaker types are defined in circuit_breaker.go

// Stub methods for StreamingProcessor (disable_rag builds)
func (sp *StreamingProcessor) HandleStreamingRequest(w interface{}, r interface{}, req *StreamingRequest) error {
	// Stub implementation - just return an error indicating streaming is disabled
	return fmt.Errorf("streaming functionality is disabled with disable_rag build tag")
}

func (sp *StreamingProcessor) GetMetrics() map[string]interface{} {
	return map[string]interface{}{"streaming_disabled": true}
}

func (sp *StreamingProcessor) Shutdown(ctx context.Context) error {
	// Stub implementation - nothing to shutdown
	return nil
}

// Note: ClientMetrics type is defined in llm.go


// SimpleTokenTracker tracks token usage and costs
type SimpleTokenTracker struct {
	totalTokens  int64
	totalCost    float64
	requestCount int64
	mutex        sync.RWMutex
}

// NewSimpleTokenTracker creates a new simple token tracker
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

// GetStats returns token usage statistics
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

// BACKWARD COMPATIBILITY SECTION
// These maintain compatibility with existing code but should be phased out

// Config represents LLM client configuration (from old interface.go)
type Config struct {
	Endpoint string
	Timeout  time.Duration
	APIKey   string
	Model    string
}

// IntentRequest represents a request to process an intent (from old interface.go)
type IntentRequest struct {
	Intent      string                 `json:"intent"`
	Prompt      string                 `json:"prompt"`
	Context     map[string]interface{} `json:"context"`
	MaxTokens   int                    `json:"maxTokens"`
	Temperature float64                `json:"temperature"`
}

// IntentResponse represents the response from intent processing (from old interface.go)
type IntentResponse struct {
	Response   string                 `json:"response"`
	Confidence float64                `json:"confidence"`
	Tokens     int                    `json:"tokens"`
	Duration   time.Duration          `json:"duration"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// STUB IMPLEMENTATIONS - Consolidated from stubs.go
// These provide default implementations for components not yet fully implemented

// ContextBuilder stub implementation (consolidated from stubs.go)
type ContextBuilder struct{
	config       *ContextBuilderConfig
	tokenManager *TokenManager
}

// ContextBuilderConfig is defined in clean_stubs.go

func NewContextBuilder() *ContextBuilder {
	return &ContextBuilder{
		config: &ContextBuilderConfig{
			DefaultMaxDocs:        10,
			MaxContextLength:      2000,
			MinConfidenceScore:    0.5,
			QueryTimeout:          30 * time.Second,
			EnableHybridSearch:    false,
			HybridAlpha:           0.5,
			TelecomKeywords:       []string{"amf", "upf", "smf", "ric"},
			QueryExpansionEnabled: false,
		},
		tokenManager: NewTokenManager(),
	}
}

func (cb *ContextBuilder) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"context_builder_enabled": false,
		"status":                  "not_implemented",
	}
}

// RelevanceScorer implementation moved to relevance_scorer.go

// RAGAwarePromptBuilder implementation moved to rag_aware_prompt_builder.go

// UTILITY FUNCTIONS

// Use getDefaultCircuitBreakerConfig from circuit_breaker.go to avoid duplicates

// isValidKubernetesName validates Kubernetes resource names
func isValidKubernetesName(name string) bool {
	if len(name) == 0 || len(name) > 253 {
		return false
	}

	// Kubernetes names must match DNS subdomain format
	for i, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '.') {
			return false
		}
		if i == 0 && (r == '-' || r == '.') {
			return false
		}
		if i == len(name)-1 && (r == '-' || r == '.') {
			return false
		}
	}

	return true
}
