package llm

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/rag"
)

// StreamingProcessor stub implementation
type StreamingProcessor struct{}

type StreamingRequest struct {
	Query     string `json:"query"`
	ModelName string `json:"model_name,omitempty"`
	MaxTokens int    `json:"max_tokens,omitempty"`
	EnableRAG bool   `json:"enable_rag,omitempty"`
}

func NewStreamingProcessor() *StreamingProcessor {
	return &StreamingProcessor{}
}

func (sp *StreamingProcessor) HandleStreamingRequest(w http.ResponseWriter, r *http.Request, req *StreamingRequest) error {
	// Stub implementation - return not implemented error
	http.Error(w, "Streaming not implemented", http.StatusNotImplemented)
	return nil
}

func (sp *StreamingProcessor) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"streaming_enabled": false,
		"status":           "not_implemented",
	}
}

func (sp *StreamingProcessor) Shutdown(ctx context.Context) error {
	// Stub implementation - nothing to shutdown
	return nil
}

// ContextBuilder stub implementation
type ContextBuilder struct{}

func NewContextBuilder() *ContextBuilder {
	return &ContextBuilder{}
}

func (cb *ContextBuilder) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"context_builder_enabled": false,
		"status":                 "not_implemented",
	}
}

// RelevanceScorer stub implementation
type RelevanceScorer struct{}

func NewRelevanceScorer() *RelevanceScorer {
	return &RelevanceScorer{}
}

func (rs *RelevanceScorer) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"relevance_scorer_enabled": false,
		"status":                  "not_implemented",
	}
}

// RAGAwarePromptBuilder stub implementation
type RAGAwarePromptBuilder struct{}

func NewRAGAwarePromptBuilder() *RAGAwarePromptBuilder {
	return &RAGAwarePromptBuilder{}
}

func (rpb *RAGAwarePromptBuilder) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"prompt_builder_enabled": false,
		"status":                "not_implemented",
	}
}

// RAGEnhancedProcessor provides LLM processing enhanced with RAG capabilities
type RAGEnhancedProcessor struct {
	baseClient     *Client
	weaviatePool   *rag.WeaviateConnectionPool
	promptEngine   *TelecomPromptEngine
	config         *RAGProcessorConfig
	logger         *slog.Logger
	circuitBreaker *CircuitBreaker
	cache          *ResponseCache
	mutex          sync.RWMutex
}

// RAGProcessorConfig holds configuration for the RAG-enhanced processor
type RAGProcessorConfig struct {
	// RAG configuration
	EnableRAG              bool                   `json:"enable_rag"`
	RAGConfidenceThreshold float32                `json:"rag_confidence_threshold"`
	FallbackToBase         bool                   `json:"fallback_to_base"`
	WeaviateURL            string                 `json:"weaviate_url"`
	WeaviateAPIKey         string                 `json:"weaviate_api_key"`
	
	// Query processing
	MaxContextDocuments    int                    `json:"max_context_documents"`
	QueryTimeout           time.Duration          `json:"query_timeout"`
	TelecomKeywords        []string               `json:"telecom_keywords"`
	
	// LLM configuration
	LLMEndpoint            string                 `json:"llm_endpoint"`
	LLMAPIKey              string                 `json:"llm_api_key"`
	LLMModelName           string                 `json:"llm_model_name"`
	MaxTokens              int                    `json:"max_tokens"`
	Temperature            float64                `json:"temperature"`
	
	// Cache and performance
	EnableCaching          bool                   `json:"enable_caching"`
	CacheTTL               time.Duration          `json:"cache_ttl"`
	MaxRetries             int                    `json:"max_retries"`
}

// getDefaultRAGProcessorConfig returns default configuration
func getDefaultRAGProcessorConfig() *RAGProcessorConfig {
	return &RAGProcessorConfig{
		EnableRAG:              true,
		RAGConfidenceThreshold: 0.6,
		FallbackToBase:         true,
		WeaviateURL:            "http://localhost:8080",
		WeaviateAPIKey:         "",
		MaxContextDocuments:    5,
		QueryTimeout:           30 * time.Second,
		TelecomKeywords: []string{
			"5G", "4G", "LTE", "NR", "gNB", "eNB", "AMF", "SMF", "UPF", "AUSF",
			"O-RAN", "RAN", "Core", "Transport", "3GPP", "ETSI", "ITU",
			"URLLC", "eMBB", "mMTC", "NSA", "SA", "PLMN", "TAC", "QCI", "QoS",
			"handover", "mobility", "bearer", "session", "procedure", "interface",
		},
		LLMEndpoint:    "http://localhost:8080/v1/chat/completions",
		LLMAPIKey:      "",
		LLMModelName:   "gpt-4o-mini",
		MaxTokens:      2048,
		Temperature:    0.0,
		EnableCaching:  true,
		CacheTTL:       5 * time.Minute,
		MaxRetries:     3,
	}
}

func NewRAGEnhancedProcessor() *RAGEnhancedProcessor {
	return NewRAGEnhancedProcessorWithConfig(nil)
}

func NewRAGEnhancedProcessorWithConfig(config *RAGProcessorConfig) *RAGEnhancedProcessor {
	if config == nil {
		config = getDefaultRAGProcessorConfig()
	}

	// Create base LLM client
	baseClient := NewClientWithConfig(config.LLMEndpoint, ClientConfig{
		APIKey:      config.LLMAPIKey,
		ModelName:   config.LLMModelName,
		MaxTokens:   config.MaxTokens,
		BackendType: "openai",
		Timeout:     config.QueryTimeout,
	})

	// Create Weaviate connection pool if RAG is enabled
	var weaviatePool *rag.WeaviateConnectionPool
	if config.EnableRAG {
		poolConfig := rag.DefaultPoolConfig()
		poolConfig.URL = config.WeaviateURL
		poolConfig.APIKey = config.WeaviateAPIKey
		poolConfig.RequestTimeout = config.QueryTimeout
		weaviatePool = rag.NewWeaviateConnectionPool(poolConfig)
		
		// Start the pool
		if err := weaviatePool.Start(); err != nil {
			slog.Error("Failed to start Weaviate connection pool", "error", err)
			weaviatePool = nil
		}
	}

	// Create circuit breaker
	circuitBreaker := NewCircuitBreaker("rag-processor", &CircuitBreakerConfig{
		FailureThreshold:      5,
		FailureRate:           0.5,
		MinimumRequestCount:   10,
		Timeout:               30 * time.Second,
		HalfOpenTimeout:       60 * time.Second,
		SuccessThreshold:      2,
		HalfOpenMaxRequests:   5,
		ResetTimeout:          60 * time.Second,
		SlidingWindowSize:     100,
		EnableHealthCheck:     false,
		HealthCheckInterval:   30 * time.Second,
		HealthCheckTimeout:    10 * time.Second,
	})

	// Create cache if enabled
	var cache *ResponseCache
	if config.EnableCaching {
		cache = NewResponseCache(config.CacheTTL, 1000)
	}

	return &RAGEnhancedProcessor{
		baseClient:     baseClient,
		weaviatePool:   weaviatePool,
		promptEngine:   NewTelecomPromptEngine(),
		config:         config,
		logger:         slog.Default().With("component", "rag-enhanced-processor"),
		circuitBreaker: circuitBreaker,
		cache:          cache,
	}
}

func (rep *RAGEnhancedProcessor) ProcessIntent(ctx context.Context, intent string) (string, error) {
	startTime := time.Now()
	rep.logger.Info("Processing intent", slog.String("intent", intent))

	// Check cache first if enabled
	if rep.cache != nil {
		cacheKey := fmt.Sprintf("rag:%s", intent)
		if cached, found := rep.cache.Get(cacheKey); found {
			rep.logger.Debug("Cache hit for RAG intent", slog.String("cache_key", cacheKey))
			return cached, nil
		}
	}

	// Determine if we should use RAG
	shouldUseRAG := rep.shouldUseRAG(intent)
	
	var result string
	var err error

	if shouldUseRAG && rep.config.EnableRAG && rep.weaviatePool != nil {
		// Try RAG-enhanced processing
		result, err = rep.processWithRAG(ctx, intent)
		if err != nil && rep.config.FallbackToBase {
			rep.logger.Warn("RAG processing failed, falling back to base client", slog.String("error", err.Error()))
			result, err = rep.processWithBase(ctx, intent)
		}
	} else {
		// Use base processing
		result, err = rep.processWithBase(ctx, intent)
	}

	if err != nil {
		rep.logger.Error("Intent processing failed", slog.String("error", err.Error()))
		return "", fmt.Errorf("failed to process intent: %w", err)
	}

	// Cache successful result if enabled
	if rep.cache != nil && result != "" {
		cacheKey := fmt.Sprintf("rag:%s", intent)
		rep.cache.Set(cacheKey, result)
	}

	processingTime := time.Since(startTime)
	rep.logger.Info("Intent processed successfully", 
		slog.Duration("processing_time", processingTime),
		slog.Bool("used_rag", shouldUseRAG && rep.config.EnableRAG && rep.weaviatePool != nil),
	)

	return result, nil
}

// shouldUseRAG determines if a query should use RAG enhancement
func (rep *RAGEnhancedProcessor) shouldUseRAG(intent string) bool {
	if !rep.config.EnableRAG {
		return false
	}

	intentLower := strings.ToLower(intent)
	
	// Check for telecom-specific keywords
	for _, keyword := range rep.config.TelecomKeywords {
		if strings.Contains(intentLower, strings.ToLower(keyword)) {
			return true
		}
	}

	// Check for patterns that benefit from RAG
	ragIndicators := []string{
		"how to", "what is", "explain", "configure", "setup", "troubleshoot",
		"optimize", "monitor", "standard", "specification", "procedure",
		"interface", "protocol", "parameter", "algorithm", "implementation",
		"best practice", "recommendation", "guideline",
	}

	for _, indicator := range ragIndicators {
		if strings.Contains(intentLower, indicator) {
			return true
		}
	}

	return false
}

// processWithRAG processes the intent using RAG enhancement
func (rep *RAGEnhancedProcessor) processWithRAG(ctx context.Context, intent string) (string, error) {
	rep.logger.Info("Processing with RAG enhancement", slog.String("intent", intent))

	// Execute with circuit breaker protection
	result, err := rep.circuitBreaker.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		return rep.executeRAGQuery(ctx, intent)
	})

	if err != nil {
		return "", fmt.Errorf("RAG processing failed: %w", err)
	}

	return result.(string), nil
}

// executeRAGQuery performs the actual RAG query execution
func (rep *RAGEnhancedProcessor) executeRAGQuery(ctx context.Context, intent string) (string, error) {
	// Step 1: Query vector database for relevant context
	contextDocs, err := rep.queryVectorDatabase(ctx, intent)
	if err != nil {
		return "", fmt.Errorf("vector database query failed: %w", err)
	}

	// Step 2: Build enhanced prompt with context
	enhancedPrompt := rep.buildEnhancedPrompt(intent, contextDocs)

	// Step 3: Send enhanced prompt to LLM
	response, err := rep.baseClient.ProcessIntent(ctx, enhancedPrompt)
	if err != nil {
		return "", fmt.Errorf("LLM processing failed: %w", err)
	}

	return response, nil
}

// queryVectorDatabase queries the Weaviate database for relevant documents
func (rep *RAGEnhancedProcessor) queryVectorDatabase(ctx context.Context, query string) ([]map[string]interface{}, error) {
	if rep.weaviatePool == nil {
		return nil, fmt.Errorf("Weaviate connection pool not available")
	}

	// For now, return mock data until we can properly configure Weaviate GraphQL
	// This allows the rest of the RAG pipeline to work
	mockResults := []map[string]interface{}{
		{
			"title":      "5G Core Network Architecture",
			"content":    "The 5G Core (5GC) network consists of several key functions including AMF, SMF, UPF, and others that work together to provide enhanced mobile services.",
			"source":     "3GPP TS 23.501",
			"category":   "Architecture",
			"confidence": 0.9,
		},
		{
			"title":      "Network Function Deployment Best Practices",
			"content":    "When deploying network functions in cloud-native environments, consider resource allocation, scaling policies, and high availability requirements.",
			"source":     "O-RAN Architecture Guide",
			"category":   "Deployment",
			"confidence": 0.8,
		},
	}

	rep.logger.Debug("Retrieved mock context documents", slog.Int("count", len(mockResults)))
	return mockResults, nil

	// TODO: Implement proper Weaviate integration
	// This would involve:
	// 1. Proper GraphQL field specification
	// 2. Vector similarity search
	// 3. Hybrid search capabilities
	// 4. Result ranking and filtering
}

// buildEnhancedPrompt creates an enhanced prompt with retrieved context
func (rep *RAGEnhancedProcessor) buildEnhancedPrompt(intent string, contextDocs []map[string]interface{}) string {
	// Classify intent type for appropriate system prompt
	intentType := rep.classifyIntentType(intent)
	systemPrompt := rep.promptEngine.GeneratePrompt(intentType, intent)

	// Add context section if we have relevant documents
	if len(contextDocs) > 0 {
		systemPrompt += "\n\n**CONTEXT FROM KNOWLEDGE BASE:**\n"
		systemPrompt += "Use the following telecom documentation and standards as context for your response:\n\n"
		
		for i, doc := range contextDocs {
			if i >= rep.config.MaxContextDocuments {
				break
			}
			
			title, _ := doc["title"].(string)
			content, _ := doc["content"].(string)
			source, _ := doc["source"].(string)
			
			// Truncate content if too long
			if len(content) > 1000 {
				content = content[:1000] + "..."
			}
			
			systemPrompt += fmt.Sprintf("Document %d - %s (Source: %s):\n%s\n\n", i+1, title, source, content)
		}
		
		systemPrompt += "**END CONTEXT**\n\n"
		systemPrompt += "Based on the above context and your telecom expertise, respond to the user's intent. " +
			"If the context provides relevant information, incorporate it into your response. " +
			"If the context doesn't contain relevant information, rely on your general knowledge but mention this limitation.\n\n"
	}

	return systemPrompt
}

// classifyIntentType attempts to classify the intent type
func (rep *RAGEnhancedProcessor) classifyIntentType(intent string) string {
	intentLower := strings.ToLower(intent)

	// Network function deployment patterns
	if strings.Contains(intentLower, "deploy") || strings.Contains(intentLower, "create") || 
	   strings.Contains(intentLower, "setup") || strings.Contains(intentLower, "install") {
		return "NetworkFunctionDeployment"
	}

	// Scaling patterns
	if strings.Contains(intentLower, "scale") || strings.Contains(intentLower, "increase") || 
	   strings.Contains(intentLower, "decrease") || strings.Contains(intentLower, "replicas") {
		return "NetworkFunctionScale"
	}

	// Default to deployment for general requests
	return "NetworkFunctionDeployment"
}

// processWithBase processes the intent using only the base LLM client
func (rep *RAGEnhancedProcessor) processWithBase(ctx context.Context, intent string) (string, error) {
	rep.logger.Info("Processing with base client", slog.String("intent", intent))
	
	response, err := rep.baseClient.ProcessIntent(ctx, intent)
	if err != nil {
		return "", fmt.Errorf("base client processing failed: %w", err)
	}

	return response, nil
}

// CircuitBreakerManager already exists in circuit_breaker.go - no need to redefine