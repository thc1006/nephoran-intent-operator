//go:build !disable_rag && !test

package rag

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/errors"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// RAGService provides Retrieval-Augmented Generation capabilities for telecom domain
type RAGService struct {
	weaviateClient *WeaviateClient
	llmClient      shared.ClientInterface
	config         *RAGConfig
	logger         *slog.Logger
	metrics        *ServiceRAGMetrics
	errorBuilder   *errors.ErrorBuilder
	cache          *ServiceRAGCache
	mutex          sync.RWMutex
}

// RAGCache provides caching for RAG responses
type ServiceRAGCache struct {
	data    map[string]*CachedResponse
	mutex   sync.RWMutex
	config  *CacheConfig
	metrics *CacheMetrics
}

// CachedResponse represents a cached RAG response
type CachedResponse struct {
	Response    *RAGResponse
	CreatedAt   time.Time
	AccessCount int64
	LastAccess  time.Time
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	EnableCache     bool          `json:"enable_cache"`
	TTL             time.Duration `json:"ttl"`
	MaxSize         int           `json:"max_size"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
}

// CacheMetrics definition moved to embedding_support.go to avoid duplicates

// RAGConfig holds configuration for the RAG service
type RAGConfig struct {
	// Search configuration
	DefaultSearchLimit int     `json:"default_search_limit"`
	MaxSearchLimit     int     `json:"max_search_limit"`
	DefaultHybridAlpha float32 `json:"default_hybrid_alpha"`
	MinConfidenceScore float32 `json:"min_confidence_score"`

	// Context configuration
	MaxContextLength     int `json:"max_context_length"`
	ContextOverlapTokens int `json:"context_overlap_tokens"`

	// LLM configuration
	MaxLLMTokens int     `json:"max_llm_tokens"`
	Temperature  float32 `json:"temperature"`

	// Caching configuration
	EnableCaching bool          `json:"enable_caching"`
	CacheTTL      time.Duration `json:"cache_ttl"`

	// Performance tuning
	EnableReranking      bool `json:"enable_reranking"`
	RerankingTopK        int  `json:"reranking_top_k"`
	EnableQueryExpansion bool `json:"enable_query_expansion"`

	// Telecom-specific settings
	TelecomDomains   []string `json:"telecom_domains"`
	PreferredSources []string `json:"preferred_sources"`
	TechnologyFilter []string `json:"technology_filter"`
}

// RAGMetrics holds performance and usage metrics
type ServiceRAGMetrics struct {
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
	Query             string                 `json:"query"`
	Context           string                 `json:"context,omitempty"`
	IntentType        string                 `json:"intent_type,omitempty"`
	SearchFilters     map[string]interface{} `json:"search_filters,omitempty"`
	MaxResults        int                    `json:"max_results,omitempty"`
	MinConfidence     float32                `json:"min_confidence,omitempty"`
	UseHybridSearch   bool                   `json:"use_hybrid_search"`
	EnableReranking   bool                   `json:"enable_reranking"`
	IncludeSourceRefs bool                   `json:"include_source_refs"`
	ResponseFormat    string                 `json:"response_format,omitempty"`
	UserID            string                 `json:"user_id,omitempty"`
	SessionID         string                 `json:"session_id,omitempty"`
}

// RAGResponse represents the response from RAG processing
type RAGResponse struct {
	Answer          string                 `json:"answer"`
	SourceDocuments []*shared.SearchResult `json:"source_documents"`
	Confidence      float32                `json:"confidence"`
	ProcessingTime  time.Duration          `json:"processing_time"`
	RetrievalTime   time.Duration          `json:"retrieval_time"`
	GenerationTime  time.Duration          `json:"generation_time"`
	UsedCache       bool                   `json:"used_cache"`
	Query           string                 `json:"query"`
	IntentType      string                 `json:"intent_type,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	ProcessedAt     time.Time              `json:"processed_at"`
}

// NewRAGService creates a new RAG service instance
func NewRAGService(weaviateClient *WeaviateClient, llmClient shared.ClientInterface, config *RAGConfig) *RAGService {
	if config == nil {
		config = getDefaultRAGConfig()
	}

	logger := slog.Default().With("component", "rag-service")

	// Initialize cache
	cache := newRAGCache(&CacheConfig{
		EnableCache:     config.EnableCaching,
		TTL:             config.CacheTTL,
		MaxSize:         1000, // Default max size
		CleanupInterval: 5 * time.Minute,
	})

	service := &RAGService{
		weaviateClient: weaviateClient,
		llmClient:      llmClient,
		config:         config,
		logger:         logger,
		metrics:        &ServiceRAGMetrics{LastUpdated: time.Now()},
		errorBuilder:   errors.NewErrorBuilder("rag-service", "", logger),
		cache:          cache,
	}

	// Start cache cleanup goroutine
	if config.EnableCaching {
		go service.startCacheCleanup()
	}

	return service
}

// getDefaultRAGConfig returns default configuration for RAG service
func getDefaultRAGConfig() *RAGConfig {
	return &RAGConfig{
		DefaultSearchLimit:   10,
		MaxSearchLimit:       50,
		DefaultHybridAlpha:   0.7,
		MinConfidenceScore:   0.5,
		MaxContextLength:     8000,
		ContextOverlapTokens: 200,
		MaxLLMTokens:         4000,
		Temperature:          0.3,
		EnableCaching:        true,
		CacheTTL:             30 * time.Minute,
		EnableReranking:      true,
		RerankingTopK:        20,
		EnableQueryExpansion: true,
		TelecomDomains:       []string{"RAN", "Core", "Transport", "Management", "O-RAN"},
		PreferredSources:     []string{"3GPP", "O-RAN", "ETSI", "ITU"},
		TechnologyFilter:     []string{"5G", "4G", "O-RAN", "vRAN"},
	}
}

// ProcessQuery processes a RAG query and returns an enhanced response
func (rs *RAGService) ProcessQuery(ctx context.Context, request *RAGRequest) (*RAGResponse, error) {
	startTime := time.Now()

	// Update error builder operation
	eb := errors.NewErrorBuilder("rag-service", "ProcessQuery", rs.logger)

	// Check for context cancellation early
	select {
	case <-ctx.Done():
		return nil, eb.ContextCancelledError(ctx)
	default:
	}

	// Validate request with standardized errors
	if request == nil {
		return nil, eb.RequiredFieldError("request")
	}
	if request.Query == "" {
		return nil, eb.RequiredFieldError("query")
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
	rs.updateMetrics(func(m *ServiceRAGMetrics) {
		m.TotalQueries++
	})

	// Check cache first if enabled
	if rs.config.EnableCaching {
		cacheKey := rs.generateCacheKey(request)
		if cachedResponse := rs.cache.Get(cacheKey); cachedResponse != nil {
			rs.logger.Debug("Cache hit for RAG query", "cache_key", cacheKey)
			cachedResponse.UsedCache = true
			rs.updateMetrics(func(m *ServiceRAGMetrics) {
				m.SuccessfulQueries++
				m.CacheHitRate = float64(rs.cache.metrics.Hits) / float64(rs.cache.metrics.Hits+rs.cache.metrics.Misses)
			})
			return cachedResponse, nil
		}
		rs.logger.Debug("Cache miss for RAG query", "cache_key", cacheKey)
	}

	// Step 1: Retrieve relevant documents
	retrievalStart := time.Now()
	searchQuery := &SearchQuery{
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
		rs.updateMetrics(func(m *ServiceRAGMetrics) {
			m.FailedQueries++
		})

		// Check if context was cancelled during search
		select {
		case <-ctx.Done():
			return nil, eb.ContextCancelledError(ctx)
		default:
		}

		return nil, eb.ExternalServiceError("weaviate", err).
			WithMetadata("search_query", request.Query).
			WithMetadata("max_results", request.MaxResults)
	}
	retrievalTime := time.Since(retrievalStart)

	// Step 2: Convert local results to shared results and prepare context
	sharedResults := make([]*shared.SearchResult, len(searchResponse.Results))
	for i, result := range searchResponse.Results {
		sharedResults[i] = &shared.SearchResult{
			Document: &shared.TelecomDocument{
				ID:      result.Document.ID,
				Content: result.Document.Content,
				Source:  result.Document.Source,
			},
			Score: result.Score,
		}
	}
	context, contextMetadata := rs.prepareContext(sharedResults, request)

	// Step 3: Generate response using LLM
	generationStart := time.Now()
	llmPrompt := rs.buildLLMPrompt(request.Query, context, request.IntentType)

	llmResponse, err := rs.llmClient.ProcessIntent(ctx, llmPrompt)
	if err != nil {
		rs.updateMetrics(func(m *ServiceRAGMetrics) {
			m.FailedQueries++
		})

		// Check if context was cancelled during LLM processing
		select {
		case <-ctx.Done():
			return nil, eb.ContextCancelledError(ctx)
		default:
		}

		return nil, eb.ExternalServiceError("llm", err).
			WithMetadata("query", request.Query).
			WithMetadata("intent_type", request.IntentType).
			WithMetadata("context_length", len(context))
	}
	generationTime := time.Since(generationStart)

	// Step 4: Post-process and enhance the response
	enhancedResponse := rs.enhanceResponse(llmResponse, sharedResults, request)

	// Create RAG response
	ragResponse := &RAGResponse{
		Answer:          enhancedResponse,
		SourceDocuments: sharedResults,
		Confidence:      rs.calculateConfidence(sharedResults),
		ProcessingTime:  time.Since(startTime),
		RetrievalTime:   retrievalTime,
		GenerationTime:  generationTime,
		UsedCache:       false,
		Query:           request.Query,
		IntentType:      request.IntentType,
		ProcessedAt:     time.Now(),
		Metadata: map[string]interface{}{
			"context_length":   len(context),
			"documents_used":   len(searchResponse.Results),
			"search_took":      searchResponse.Took,
			"context_metadata": contextMetadata,
		},
	}

	// Cache the response if caching is enabled
	if rs.config.EnableCaching {
		cacheKey := rs.generateCacheKey(request)
		rs.cache.Set(cacheKey, ragResponse)
	}

	// Update success metrics
	rs.updateMetrics(func(m *ServiceRAGMetrics) {
		m.SuccessfulQueries++
		m.AverageLatency = (m.AverageLatency*time.Duration(m.SuccessfulQueries-1) + ragResponse.ProcessingTime) / time.Duration(m.SuccessfulQueries)
		m.AverageRetrievalTime = (m.AverageRetrievalTime*time.Duration(m.SuccessfulQueries-1) + retrievalTime) / time.Duration(m.SuccessfulQueries)
		m.AverageGenerationTime = (m.AverageGenerationTime*time.Duration(m.SuccessfulQueries-1) + generationTime) / time.Duration(m.SuccessfulQueries)
		m.CacheHitRate = float64(rs.cache.metrics.Hits) / float64(rs.cache.metrics.Hits+rs.cache.metrics.Misses)
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
		"context_truncated":    false,
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
func (rs *RAGService) updateMetrics(updater func(*ServiceRAGMetrics)) {
	rs.metrics.mutex.Lock()
	defer rs.metrics.mutex.Unlock()
	updater(rs.metrics)
}

// GetMetrics returns the current metrics
func (rs *RAGService) GetMetrics() *ServiceRAGMetrics {
	rs.metrics.mutex.RLock()
	defer rs.metrics.mutex.RUnlock()

	// Return a copy
	metrics := *rs.metrics
	return &metrics
}

// GetHealth returns the health status of the RAG service
func (rs *RAGService) GetHealth() map[string]interface{} {
	weaviateHealth := rs.weaviateClient.GetHealthStatus()

	// Determine overall health status
	status := "healthy"
	issues := make([]string, 0)

	// Check Weaviate health
	if !weaviateHealth.IsHealthy {
		status = "degraded"
		issues = append(issues, "weaviate_unhealthy")
	}

	// Check LLM client health
	if rs.llmClient == nil {
		status = "unhealthy"
		issues = append(issues, "llm_client_unavailable")
	}

	// Check cache health if enabled
	cacheHealth := "disabled"
	if rs.config.EnableCaching && rs.cache != nil {
		cacheHealth = "healthy"
		cacheMetrics := rs.cache.GetMetrics()
		if cacheMetrics.TotalItems > int64(rs.cache.config.MaxSize)*9/10 { // 90% full
			status = "degraded"
			issues = append(issues, "cache_near_full")
		}
	}

	// Check recent error rates
	metrics := rs.GetMetrics()
	if metrics.TotalQueries > 0 {
		errorRate := float64(metrics.FailedQueries) / float64(metrics.TotalQueries)
		if errorRate > 0.1 { // More than 10% error rate
			status = "degraded"
			issues = append(issues, "high_error_rate")
		}
		if errorRate > 0.5 { // More than 50% error rate
			status = "unhealthy"
		}
	}

	return map[string]interface{}{
		"status": status,
		"issues": issues,
		"weaviate": map[string]interface{}{
			"healthy":    weaviateHealth.IsHealthy,
			"last_check": weaviateHealth.LastCheck,
			"version":    weaviateHealth.Version,
		},
		"cache": map[string]interface{}{
			"status":  cacheHealth,
			"metrics": rs.cache.GetMetrics(),
		},
		"metrics":   metrics,
		"timestamp": time.Now(),
	}
}

// generateCacheKey generates a cache key for a RAG request
func (rs *RAGService) generateCacheKey(request *RAGRequest) string {
	// Create a deterministic key based on request parameters
	data := struct {
		Query         string                 `json:"query"`
		IntentType    string                 `json:"intent_type"`
		MaxResults    int                    `json:"max_results"`
		MinConfidence float32                `json:"min_confidence"`
		Filters       map[string]interface{} `json:"filters"`
		UseHybrid     bool                   `json:"use_hybrid"`
		EnableRerank  bool                   `json:"enable_rerank"`
	}{
		Query:         request.Query,
		IntentType:    request.IntentType,
		MaxResults:    request.MaxResults,
		MinConfidence: request.MinConfidence,
		Filters:       request.SearchFilters,
		UseHybrid:     request.UseHybridSearch,
		EnableRerank:  request.EnableReranking,
	}

	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

// startCacheCleanup starts the cache cleanup goroutine
func (rs *RAGService) startCacheCleanup() {
	ticker := time.NewTicker(rs.cache.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rs.cache.Cleanup()
		}
	}
}

// newRAGCache creates a new RAG cache
func newRAGCache(config *CacheConfig) *ServiceRAGCache {
	return &ServiceRAGCache{
		data:    make(map[string]*CachedResponse),
		config:  config,
		metrics: &CacheMetrics{},
	}
}

// Get retrieves a cached response
func (c *ServiceRAGCache) Get(key string) *RAGResponse {
	if !c.config.EnableCache {
		return nil
	}

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	cached, exists := c.data[key]
	if !exists {
		c.updateMetrics(func(m *CacheMetrics) { m.Misses++ })
		return nil
	}

	// Check if expired
	if time.Since(cached.CreatedAt) > c.config.TTL {
		// Remove expired entry
		go c.remove(key)
		c.updateMetrics(func(m *CacheMetrics) { m.Misses++ })
		return nil
	}

	// Update access info
	cached.AccessCount++
	cached.LastAccess = time.Now()

	c.updateMetrics(func(m *CacheMetrics) { m.Hits++ })
	return cached.Response
}

// Set stores a response in the cache
func (c *ServiceRAGCache) Set(key string, response *RAGResponse) {
	if !c.config.EnableCache {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if we need to evict items
	if len(c.data) >= c.config.MaxSize {
		c.evictLRU()
	}

	c.data[key] = &CachedResponse{
		Response:    response,
		CreatedAt:   time.Now(),
		AccessCount: 0,
		LastAccess:  time.Now(),
	}

	c.updateMetrics(func(m *CacheMetrics) { m.TotalItems++ })
}

// remove removes an item from cache
func (c *ServiceRAGCache) remove(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, exists := c.data[key]; exists {
		delete(c.data, key)
		c.updateMetrics(func(m *CacheMetrics) {
			m.TotalItems--
			m.Evictions++
		})
	}
}

// evictLRU evicts the least recently used item
func (c *ServiceRAGCache) evictLRU() {
	if len(c.data) == 0 {
		return
	}

	var oldestKey string
	var oldestTime time.Time = time.Now()

	for key, cached := range c.data {
		if cached.LastAccess.Before(oldestTime) {
			oldestTime = cached.LastAccess
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(c.data, oldestKey)
		c.updateMetrics(func(m *CacheMetrics) {
			m.TotalItems--
			m.Evictions++
		})
	}
}

// Cleanup removes expired entries
func (c *ServiceRAGCache) Cleanup() {
	if !c.config.EnableCache {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	expiredKeys := make([]string, 0)

	for key, cached := range c.data {
		if now.Sub(cached.CreatedAt) > c.config.TTL {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		delete(c.data, key)
		c.updateMetrics(func(m *CacheMetrics) {
			m.TotalItems--
			m.Evictions++
		})
	}
}

// GetMetrics returns cache metrics
func (c *ServiceRAGCache) GetMetrics() *CacheMetrics {
	c.metrics.mutex.RLock()
	defer c.metrics.mutex.RUnlock()

	metrics := *c.metrics
	return &metrics
}

// updateMetrics safely updates cache metrics
func (c *ServiceRAGCache) updateMetrics(updater func(*CacheMetrics)) {
	c.metrics.mutex.Lock()
	defer c.metrics.mutex.Unlock()
	updater(c.metrics)
}
