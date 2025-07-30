package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// RAGService provides Retrieval-Augmented Generation capabilities for telecom domain
type RAGService struct {
	weaviateClient *WeaviateClient
	llmClient      shared.ClientInterface
	config         *RAGConfig
	logger         *slog.Logger
	metrics        *RAGMetrics
	mutex          sync.RWMutex
}

// RAGConfig holds configuration for the RAG service
type RAGConfig struct {
	// Search configuration
	DefaultSearchLimit     int     `json:"default_search_limit"`
	MaxSearchLimit        int     `json:"max_search_limit"`
	DefaultHybridAlpha    float32 `json:"default_hybrid_alpha"`
	MinConfidenceScore    float32 `json:"min_confidence_score"`
	
	// Context configuration
	MaxContextLength      int     `json:"max_context_length"`
	ContextOverlapTokens  int     `json:"context_overlap_tokens"`
	
	// LLM configuration
	MaxLLMTokens         int     `json:"max_llm_tokens"`
	Temperature          float32 `json:"temperature"`
	
	// Caching configuration
	EnableCaching        bool          `json:"enable_caching"`
	CacheTTL            time.Duration `json:"cache_ttl"`
	
	// Performance tuning
	EnableReranking      bool    `json:"enable_reranking"`
	RerankingTopK       int     `json:"reranking_top_k"`
	EnableQueryExpansion bool    `json:"enable_query_expansion"`
	
	// Telecom-specific settings
	TelecomDomains      []string `json:"telecom_domains"`
	PreferredSources    []string `json:"preferred_sources"`
	TechnologyFilter    []string `json:"technology_filter"`
}

// RAGMetrics holds performance and usage metrics
type RAGMetrics struct {
	TotalQueries          int64         `json:"total_queries"`
	SuccessfulQueries     int64         `json:"successful_queries"`
	FailedQueries         int64         `json:"failed_queries"`
	AverageLatency        time.Duration `json:"average_latency"`
	AverageRetrievalTime  time.Duration `json:"average_retrieval_time"`
	AverageGenerationTime time.Duration `json:"average_generation_time"`
	CacheHitRate          float64       `json:"cache_hit_rate"`
	LastUpdated           time.Time     `json:"last_updated"`
	mutex                 sync.RWMutex
}

// RAGRequest represents a request for RAG processing
type RAGRequest struct {
	Query               string                 `json:"query"`
	Context             string                 `json:"context,omitempty"`
	IntentType          string                 `json:"intent_type,omitempty"`
	SearchFilters       map[string]interface{} `json:"search_filters,omitempty"`
	MaxResults          int                    `json:"max_results,omitempty"`
	MinConfidence       float32                `json:"min_confidence,omitempty"`
	UseHybridSearch     bool                   `json:"use_hybrid_search"`
	EnableReranking     bool                   `json:"enable_reranking"`
	IncludeSourceRefs   bool                   `json:"include_source_refs"`
	ResponseFormat      string                 `json:"response_format,omitempty"`
	UserID              string                 `json:"user_id,omitempty"`
	SessionID           string                 `json:"session_id,omitempty"`
}

// RAGResponse represents the response from RAG processing
type RAGResponse struct {
	Answer              string                 `json:"answer"`
	SourceDocuments     []*shared.SearchResult `json:"source_documents"`
	Confidence          float32                `json:"confidence"`
	ProcessingTime      time.Duration          `json:"processing_time"`
	RetrievalTime       time.Duration          `json:"retrieval_time"`
	GenerationTime      time.Duration          `json:"generation_time"`
	UsedCache           bool                   `json:"used_cache"`
	Query               string                 `json:"query"`
	IntentType          string                 `json:"intent_type,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
	ProcessedAt         time.Time              `json:"processed_at"`
}

// NewRAGService creates a new RAG service instance
func NewRAGService(weaviateClient *WeaviateClient, llmClient shared.ClientInterface, config *RAGConfig) *RAGService {
	if config == nil {
		config = getDefaultRAGConfig()
	}

	return &RAGService{
		weaviateClient: weaviateClient,
		llmClient:      llmClient,
		config:         config,
		logger:         slog.Default().With("component", "rag-service"),
		metrics:        &RAGMetrics{LastUpdated: time.Now()},
	}
}

// getDefaultRAGConfig returns default configuration for RAG service
func getDefaultRAGConfig() *RAGConfig {
	return &RAGConfig{
		DefaultSearchLimit:    10,
		MaxSearchLimit:       50,
		DefaultHybridAlpha:   0.7,
		MinConfidenceScore:   0.5,
		MaxContextLength:     8000,
		ContextOverlapTokens: 200,
		MaxLLMTokens:         4000,
		Temperature:          0.3,
		EnableCaching:        true,
		CacheTTL:            30 * time.Minute,
		EnableReranking:      true,
		RerankingTopK:       20,
		EnableQueryExpansion: true,
		TelecomDomains:      []string{"RAN", "Core", "Transport", "Management", "O-RAN"},
		PreferredSources:    []string{"3GPP", "O-RAN", "ETSI", "ITU"},
		TechnologyFilter:    []string{"5G", "4G", "O-RAN", "vRAN"},
	}
}

// ProcessQuery processes a RAG query and returns an enhanced response
func (rs *RAGService) ProcessQuery(ctx context.Context, request *RAGRequest) (*RAGResponse, error) {
	startTime := time.Now()
	
	// Validate request
	if request == nil {
		return nil, fmt.Errorf("RAG request cannot be nil")
	}
	if request.Query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	// Set defaults
	if request.MaxResults == 0 {
		request.MaxResults = rs.config.DefaultSearchLimit
	}
	if request.MaxResults > rs.config.MaxSearchLimit {
		request.MaxResults = rs.config.MaxSearchLimit
	}
	if request.MinConfidence == 0 {
		request.MinConfidence = rs.config.MinConfidenceScore
	}

	rs.logger.Info("Processing RAG query",
		"query", request.Query,
		"intent_type", request.IntentType,
		"max_results", request.MaxResults,
	)

	// Update metrics
	rs.updateMetrics(func(m *RAGMetrics) {
		m.TotalQueries++
	})

	// Step 1: Retrieve relevant documents
	retrievalStart := time.Now()
	searchQuery := &shared.SearchQuery{
		Query:         request.Query,
		Limit:         request.MaxResults,
		Filters:       rs.buildSearchFilters(request),
		HybridSearch:  request.UseHybridSearch,
		HybridAlpha:   rs.config.DefaultHybridAlpha,
		UseReranker:   request.EnableReranking && rs.config.EnableReranking,
		MinConfidence: request.MinConfidence,
		ExpandQuery:   rs.config.EnableQueryExpansion,
	}

	searchResponse, err := rs.weaviateClient.Search(ctx, searchQuery)
	if err != nil {
		rs.updateMetrics(func(m *RAGMetrics) {
			m.FailedQueries++
		})
		return nil, fmt.Errorf("failed to retrieve documents: %w", err)
	}
	retrievalTime := time.Since(retrievalStart)

	// Step 2: Prepare context from retrieved documents
	context, contextMetadata := rs.prepareContext(searchResponse.Results, request)

	// Step 3: Generate response using LLM
	generationStart := time.Now()
	llmPrompt := rs.buildLLMPrompt(request.Query, context, request.IntentType)
	
	llmResponse, err := rs.llmClient.ProcessIntent(ctx, llmPrompt)
	if err != nil {
		rs.updateMetrics(func(m *RAGMetrics) {
			m.FailedQueries++
		})
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}
	generationTime := time.Since(generationStart)

	// Step 4: Post-process and enhance the response
	enhancedResponse := rs.enhanceResponse(llmResponse, searchResponse.Results, request)

	// Create RAG response
	ragResponse := &RAGResponse{
		Answer:          enhancedResponse,
		SourceDocuments: searchResponse.Results,
		Confidence:      rs.calculateConfidence(searchResponse.Results),
		ProcessingTime:  time.Since(startTime),
		RetrievalTime:   retrievalTime,
		GenerationTime:  generationTime,
		UsedCache:       false, // TODO: implement caching
		Query:           request.Query,
		IntentType:      request.IntentType,
		ProcessedAt:     time.Now(),
		Metadata: map[string]interface{}{
			"context_length":    len(context),
			"documents_used":    len(searchResponse.Results),
			"search_took":       searchResponse.Took,
			"context_metadata":  contextMetadata,
		},
	}

	// Update success metrics
	rs.updateMetrics(func(m *RAGMetrics) {
		m.SuccessfulQueries++
		m.AverageLatency = (m.AverageLatency*time.Duration(m.SuccessfulQueries-1) + ragResponse.ProcessingTime) / time.Duration(m.SuccessfulQueries)
		m.AverageRetrievalTime = (m.AverageRetrievalTime*time.Duration(m.SuccessfulQueries-1) + retrievalTime) / time.Duration(m.SuccessfulQueries)
		m.AverageGenerationTime = (m.AverageGenerationTime*time.Duration(m.SuccessfulQueries-1) + generationTime) / time.Duration(m.SuccessfulQueries)
		m.LastUpdated = time.Now()
	})

	rs.logger.Info("RAG query processed successfully",
		"query", request.Query,
		"processing_time", ragResponse.ProcessingTime,
		"documents_used", len(ragResponse.SourceDocuments),
		"confidence", ragResponse.Confidence,
	)

	return ragResponse, nil
}

// buildSearchFilters constructs search filters based on the request and configuration
func (rs *RAGService) buildSearchFilters(request *RAGRequest) map[string]interface{} {
	filters := make(map[string]interface{})

	// Add user-provided filters
	for k, v := range request.SearchFilters {
		filters[k] = v
	}

	// Add intent-type specific filters
	if request.IntentType != "" {
		switch strings.ToLower(request.IntentType) {
		case "configuration":
			filters["category"] = "Configuration"
		case "optimization":
			filters["category"] = "Optimization"
		case "troubleshooting":
			filters["category"] = "Troubleshooting"
		case "monitoring":
			filters["category"] = "Monitoring"
		}
	}

	// Add technology filters if configured
	if len(rs.config.TechnologyFilter) > 0 {
		// This would be implemented as an OR filter in a full implementation
		// For now, we'll just add a comment indicating the intention
		// filters["technology"] = rs.config.TechnologyFilter
	}

	return filters
}

// prepareContext creates a context string from retrieved documents
func (rs *RAGService) prepareContext(results []*shared.SearchResult, request *RAGRequest) (string, map[string]interface{}) {
	var contextParts []string
	var totalTokens int
	documentsUsed := 0
	
	metadata := map[string]interface{}{
		"documents_considered": len(results),
		"context_truncated":   false,
	}

	for i, result := range results {
		if result.Document == nil {
			continue
		}

		// Estimate token count (rough approximation: 1 token ≈ 4 characters)
		contentTokens := len(result.Document.Content) / 4
		
		// Check if adding this document would exceed the context limit
		if totalTokens+contentTokens > rs.config.MaxContextLength {
			metadata["context_truncated"] = true
			break
		}

		// Format document for context
		docContext := rs.formatDocumentForContext(result, i+1)
		contextParts = append(contextParts, docContext)
		totalTokens += len(docContext) / 4
		documentsUsed++
	}

	metadata["documents_used"] = documentsUsed
	metadata["estimated_tokens"] = totalTokens

	context := strings.Join(contextParts, "\n\n---\n\n")
	
	// Add user context if provided
	if request.Context != "" {
		context = request.Context + "\n\n" + context
	}

	return context, metadata
}

// formatDocumentForContext formats a document for inclusion in the LLM context
func (rs *RAGService) formatDocumentForContext(result *shared.SearchResult, index int) string {
	doc := result.Document
	
	var parts []string
	
	// Add document header
	parts = append(parts, fmt.Sprintf("Document %d:", index))
	
	if doc.Title != "" {
		parts = append(parts, fmt.Sprintf("Title: %s", doc.Title))
	}
	
	if doc.Source != "" {
		parts = append(parts, fmt.Sprintf("Source: %s", doc.Source))
	}
	
	if doc.Category != "" {
		parts = append(parts, fmt.Sprintf("Category: %s", doc.Category))
	}
	
	if doc.Version != "" {
		parts = append(parts, fmt.Sprintf("Version: %s", doc.Version))
	}
	
	if len(doc.Technology) > 0 {
		parts = append(parts, fmt.Sprintf("Technologies: %s", strings.Join(doc.Technology, ", ")))
	}
	
	if len(doc.NetworkFunction) > 0 {
		parts = append(parts, fmt.Sprintf("Network Functions: %s", strings.Join(doc.NetworkFunction, ", ")))
	}
	
	parts = append(parts, fmt.Sprintf("Relevance Score: %.3f", result.Score))
	parts = append(parts, "")
	parts = append(parts, doc.Content)
	
	return strings.Join(parts, "\n")
}

// buildLLMPrompt constructs the prompt for the LLM
func (rs *RAGService) buildLLMPrompt(query, context, intentType string) string {
	var promptParts []string
	
	// System prompt
	promptParts = append(promptParts, `You are an expert telecommunications engineer and system administrator with deep knowledge of 5G, 4G, O-RAN, and network infrastructure. Your role is to provide accurate, actionable answers based on the provided technical documentation.

Guidelines:
1. Base your answers primarily on the provided context documents
2. Clearly distinguish between factual information from the documents and your general knowledge
3. Use precise technical terminology appropriate for telecom professionals
4. When relevant, reference specific standards, specifications, or working groups
5. If the context doesn't contain sufficient information, state this clearly
6. Provide practical, actionable guidance when possible
7. Consider the network domain (RAN, Core, Transport) when formulating responses`)

	// Intent-specific instructions
	if intentType != "" {
		switch strings.ToLower(intentType) {
		case "configuration":
			promptParts = append(promptParts, "\nFocus on configuration parameters, procedures, and best practices.")
		case "optimization":
			promptParts = append(promptParts, "\nFocus on performance optimization techniques, KPI improvements, and tuning recommendations.")
		case "troubleshooting":
			promptParts = append(promptParts, "\nFocus on diagnostic procedures, common issues, and resolution steps.")
		case "monitoring":
			promptParts = append(promptParts, "\nFocus on monitoring approaches, key metrics, and alerting strategies.")
		}
	}

	// Context section
	if context != "" {
		promptParts = append(promptParts, "\n\nRelevant Technical Documentation:")
		promptParts = append(promptParts, context)
	}

	// Query section
	promptParts = append(promptParts, "\n\nUser Question:")
	promptParts = append(promptParts, query)

	// Response instruction
	promptParts = append(promptParts, "\n\nPlease provide a comprehensive answer based on the documentation above. Include specific references to standards or specifications when applicable.")

	return strings.Join(promptParts, "\n")
}

// enhanceResponse post-processes the LLM response to add source references and formatting
func (rs *RAGService) enhanceResponse(llmResponse string, sourceDocuments []*shared.SearchResult, request *RAGRequest) string {
	response := llmResponse

	// Add source references if requested
	if request.IncludeSourceRefs && len(sourceDocuments) > 0 {
		var references []string
		
		for i, result := range sourceDocuments {
			if result.Document != nil && i < 5 { // Limit to top 5 references
				ref := fmt.Sprintf("[%d] %s", i+1, result.Document.Source)
				if result.Document.Title != "" {
					ref += fmt.Sprintf(" - %s", result.Document.Title)
				}
				if result.Document.Version != "" {
					ref += fmt.Sprintf(" (%s)", result.Document.Version)
				}
				references = append(references, ref)
			}
		}
		
		if len(references) > 0 {
			response += "\n\n**Sources:**\n" + strings.Join(references, "\n")
		}
	}

	return response
}

// calculateConfidence calculates an overall confidence score for the response
func (rs *RAGService) calculateConfidence(results []*shared.SearchResult) float32 {
	if len(results) == 0 {
		return 0.0
	}

	var totalScore float32
	var weights float32
	
	for i, result := range results {
		// Weight decreases with position (first result has highest weight)
		weight := 1.0 / float32(i+1)
		totalScore += result.Score * weight
		weights += weight
	}

	confidence := totalScore / weights
	
	// Normalize to 0-1 range and apply some adjustments
	if confidence > 1.0 {
		confidence = 1.0
	}
	
	// Reduce confidence if we have few results
	if len(results) < 3 {
		confidence *= 0.8
	}

	return confidence
}

// updateMetrics safely updates the metrics
func (rs *RAGService) updateMetrics(updater func(*RAGMetrics)) {
	rs.metrics.mutex.Lock()
	defer rs.metrics.mutex.Unlock()
	updater(rs.metrics)
}

// GetMetrics returns the current metrics
func (rs *RAGService) GetMetrics() *RAGMetrics {
	rs.metrics.mutex.RLock()
	defer rs.metrics.mutex.RUnlock()
	
	// Return a copy
	metrics := *rs.metrics
	return &metrics
}

// GetHealth returns the health status of the RAG service
func (rs *RAGService) GetHealth() map[string]interface{} {
	weaviateHealth := rs.weaviateClient.GetHealthStatus()
	
	return map[string]interface{}{
		"status": "healthy", // TODO: implement proper health checking
		"weaviate": map[string]interface{}{
			"healthy":    weaviateHealth.IsHealthy,
			"last_check": weaviateHealth.LastCheck,
			"version":    weaviateHealth.Version,
		},
		"metrics": rs.GetMetrics(),
	}
}