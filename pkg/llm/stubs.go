package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/rag"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
)

// StreamingProcessor handles streaming requests with server-sent events
type StreamingProcessor struct {
	httpClient *http.Client
	ragAPIURL  string
	logger     *slog.Logger
	mutex      sync.RWMutex
}

// StreamingRequest represents a streaming request payload
type StreamingRequest struct {
	Query     string `json:"query"`
	ModelName string `json:"model_name,omitempty"`
	MaxTokens int    `json:"max_tokens,omitempty"`
	EnableRAG bool   `json:"enable_rag,omitempty"`
}

func NewStreamingProcessor() *StreamingProcessor {
	return &StreamingProcessor{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		ragAPIURL: "http://rag-api:8080",
		logger:    slog.Default().With("component", "streaming-processor"),
	}
}

func (sp *StreamingProcessor) HandleStreamingRequest(w http.ResponseWriter, r *http.Request, req *StreamingRequest) error {
	sp.logger.Info("Handling streaming request", slog.String("query", req.Query))

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create request to RAG API stream endpoint
	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	streamURL := sp.ragAPIURL + "/stream"
	httpReq, err := http.NewRequestWithContext(r.Context(), "POST", streamURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create stream request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	// Execute the request
	resp, err := sp.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to connect to RAG API stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("RAG API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Stream the response using bufio.Scanner
	scanner := bufio.NewScanner(resp.Body)
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported")
	}

	for scanner.Scan() {
		line := scanner.Text()

		// Forward the SSE event to client
		fmt.Fprintf(w, "%s\n", line)

		// Flush after each line for SSE
		if line == "" {
			flusher.Flush()
		}

		// Check if client disconnected
		select {
		case <-r.Context().Done():
			sp.logger.Info("Client disconnected from stream")
			return nil
		default:
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stream: %w", err)
	}

	sp.logger.Info("Streaming request completed successfully")
	return nil
}

func (sp *StreamingProcessor) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"streaming_enabled": true,
		"status":            "active",
		"rag_api_url":       sp.ragAPIURL,
	}
}

func (sp *StreamingProcessor) Shutdown(ctx context.Context) error {
	sp.logger.Info("Shutting down streaming processor")
	return nil
}

// ContextBuilder provides RAG context building capabilities
type ContextBuilder struct {
	weaviatePool *rag.WeaviateConnectionPool
	logger       *slog.Logger
	config       *ContextBuilderConfig
	metrics      *ContextBuilderMetrics
	mutex        sync.RWMutex
}

// ContextBuilderConfig holds configuration for the context builder
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

func NewContextBuilder() *ContextBuilder {
	return NewContextBuilderWithPool(nil)
}

func NewContextBuilderWithPool(pool *rag.WeaviateConnectionPool) *ContextBuilder {
	config := &ContextBuilderConfig{
		DefaultMaxDocs:        5,
		MaxContextLength:      8192,
		MinConfidenceScore:    0.6,
		QueryTimeout:          30 * time.Second,
		EnableHybridSearch:    true,
		HybridAlpha:           0.7,
		QueryExpansionEnabled: true,
		TelecomKeywords: []string{
			"5G", "4G", "LTE", "NR", "gNB", "eNB", "AMF", "SMF", "UPF", "AUSF",
			"O-RAN", "RAN", "Core", "Transport", "3GPP", "ETSI", "ITU",
			"URLLC", "eMBB", "mMTC", "NSA", "SA", "PLMN", "TAC", "QCI", "QoS",
			"handover", "mobility", "bearer", "session", "procedure", "interface",
			"network slice", "slicing", "orchestration", "deployment",
		},
	}

	return &ContextBuilder{
		weaviatePool: pool,
		logger:       slog.Default().With("component", "context-builder"),
		config:       config,
		metrics:      &ContextBuilderMetrics{},
	}
}

// BuildContext retrieves and builds context from the RAG system using semantic search
func (cb *ContextBuilder) BuildContext(ctx context.Context, intent string, maxDocs int) ([]map[string]any, error) {
	startTime := time.Now()

	// Update metrics
	cb.updateMetrics(func(m *ContextBuilderMetrics) {
		m.TotalQueries++
	})

	// Validate inputs
	if intent == "" {
		cb.updateMetrics(func(m *ContextBuilderMetrics) {
			m.FailedQueries++
		})
		return nil, fmt.Errorf("intent cannot be empty")
	}

	if maxDocs <= 0 {
		maxDocs = cb.config.DefaultMaxDocs
	}

	// Check if we have a connection pool
	if cb.weaviatePool == nil {
		cb.logger.Warn("No Weaviate connection pool available, returning empty context")
		cb.updateMetrics(func(m *ContextBuilderMetrics) {
			m.FailedQueries++
		})
		return []map[string]any{}, nil
	}

	// Enhance the query with telecom-specific context if enabled
	enhancedQuery := intent
	if cb.config.QueryExpansionEnabled {
		enhancedQuery = cb.expandQuery(intent)
	}

	// Create search query with timeout context
	queryCtx, cancel := context.WithTimeout(ctx, cb.config.QueryTimeout)
	defer cancel()

	// Perform semantic search using the connection pool
	var searchResults []*shared.SearchResult
	var searchErr error

	err := cb.weaviatePool.WithConnection(queryCtx, func(client *weaviate.Client) error {
		// Build GraphQL query directly
		var gqlQuery *graphql.GetObjectsBuilder

		if cb.config.EnableHybridSearch {
			// Use hybrid search (vector + keyword)
			gqlQuery = client.GraphQL().Get().
				WithClassName("TelecomKnowledge").
				WithHybrid(
					client.GraphQL().HybridArgumentBuilder().
						WithQuery(enhancedQuery).
						WithAlpha(cb.config.HybridAlpha),
				)
		} else {
			// Use pure vector search
			gqlQuery = client.GraphQL().Get().
				WithClassName("TelecomKnowledge").
				WithNearText(
					client.GraphQL().NearTextArgumentBuilder().
						WithConcepts([]string{enhancedQuery}),
				)
		}

		// Define fields to retrieve
		fields := []graphql.Field{
			{Name: "content"},
			{Name: "title"},
			{Name: "source"},
			{Name: "category"},
			{Name: "version"},
			{Name: "keywords"},
			{Name: "language"},
			{Name: "documentType"},
			{Name: "networkFunction"},
			{Name: "technology"},
			{Name: "useCase"},
			{Name: "confidence"},
			{Name: "_additional", Fields: []graphql.Field{
				{Name: "id"},
				{Name: "score"},
				{Name: "distance"},
			}},
		}

		// Execute the query
		result, err := gqlQuery.
			WithFields(fields...).
			WithLimit(maxDocs).
			Do(queryCtx)

		if err != nil {
			searchErr = err
			return err
		}

		// Parse results
		searchResults = make([]*shared.SearchResult, 0)
		if result.Data != nil {
			if data, ok := result.Data["Get"].(map[string]interface{}); ok {
				if telecomData, ok := data["TelecomKnowledge"].([]interface{}); ok {
					for _, item := range telecomData {
						if itemMap, ok := item.(map[string]interface{}); ok {
							searchResult := cb.parseSearchResult(itemMap)
							if searchResult != nil && searchResult.Document.Confidence >= cb.config.MinConfidenceScore {
								searchResults = append(searchResults, searchResult)
							}
						}
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		cb.logger.Error("Failed to get connection from pool", "error", err)
		cb.updateMetrics(func(m *ContextBuilderMetrics) {
			m.FailedQueries++
		})
		return nil, fmt.Errorf("failed to connect to vector database: %w", err)
	}

	if searchErr != nil {
		cb.logger.Error("Search query failed", "error", searchErr, "query", enhancedQuery)
		cb.updateMetrics(func(m *ContextBuilderMetrics) {
			m.FailedQueries++
		})
		return nil, fmt.Errorf("semantic search failed: %w", searchErr)
	}

	// Convert search results to context format
	contextDocs := make([]map[string]any, 0, len(searchResults))
	totalContentLength := 0

	for _, result := range searchResults {
		if result == nil || result.Document == nil {
			continue
		}

		doc := result.Document

		// Check content length limits
		if totalContentLength+len(doc.Content) > cb.config.MaxContextLength {
			cb.logger.Debug("Context length limit reached, truncating results",
				"current_length", totalContentLength,
				"max_length", cb.config.MaxContextLength,
			)
			break
		}

		// Build context document
		contextDoc := map[string]any{
			"id":            doc.ID,
			"title":         doc.Title,
			"content":       doc.Content,
			"source":        doc.Source,
			"category":      doc.Category,
			"version":       doc.Version,
			"language":      doc.Language,
			"document_type": doc.DocumentType,
			"confidence":    doc.Confidence,
			"score":         result.Score,
			"distance":      result.Distance,
		}

		// Add array fields if they exist
		if len(doc.Keywords) > 0 {
			contextDoc["keywords"] = doc.Keywords
		}
		if len(doc.NetworkFunction) > 0 {
			contextDoc["network_function"] = doc.NetworkFunction
		}
		if len(doc.Technology) > 0 {
			contextDoc["technology"] = doc.Technology
		}
		if len(doc.UseCase) > 0 {
			contextDoc["use_case"] = doc.UseCase
		}

		// Add metadata if available
		if doc.Metadata != nil && len(doc.Metadata) > 0 {
			contextDoc["metadata"] = doc.Metadata
		}

		contextDocs = append(contextDocs, contextDoc)
		totalContentLength += len(doc.Content)
	}

	// Update metrics with successful query
	duration := time.Since(startTime)
	cb.updateMetrics(func(m *ContextBuilderMetrics) {
		m.SuccessfulQueries++
		m.TotalLatency += duration
		if m.TotalQueries > 0 {
			m.AverageQueryDuration = time.Duration(int64(m.TotalLatency) / m.TotalQueries)
		}
		if m.SuccessfulQueries > 0 {
			currentAvg := m.AverageDocumentsFound
			newCount := len(contextDocs)
			m.AverageDocumentsFound = (currentAvg*int(m.SuccessfulQueries-1) + newCount) / int(m.SuccessfulQueries)
		}
	})

	cb.logger.Info("Context building completed",
		"intent", intent,
		"enhanced_query", enhancedQuery,
		"documents_found", len(contextDocs),
		"total_content_length", totalContentLength,
		"duration", duration,
	)

	return contextDocs, nil
}

// expandQuery enhances the query with telecom-specific context
func (cb *ContextBuilder) expandQuery(query string) string {
	// Convert to lowercase for matching
	lowerQuery := strings.ToLower(query)

	// Find relevant telecom keywords that might enhance the query
	var relevantKeywords []string
	for _, keyword := range cb.config.TelecomKeywords {
		if !strings.Contains(lowerQuery, strings.ToLower(keyword)) {
			// Add related keywords based on context
			if cb.isRelatedKeyword(lowerQuery, keyword) {
				relevantKeywords = append(relevantKeywords, keyword)
			}
		}
	}

	// Limit the number of additional keywords to avoid query bloat
	if len(relevantKeywords) > 3 {
		relevantKeywords = relevantKeywords[:3]
	}

	// Enhance the query if relevant keywords were found
	if len(relevantKeywords) > 0 {
		enhanced := fmt.Sprintf("%s %s", query, strings.Join(relevantKeywords, " "))
		cb.logger.Debug("Enhanced query with telecom keywords",
			"original", query,
			"enhanced", enhanced,
			"added_keywords", relevantKeywords,
		)
		return enhanced
	}

	return query
}

// isRelatedKeyword determines if a keyword is contextually related to the query
func (cb *ContextBuilder) isRelatedKeyword(query, keyword string) bool {
	// Define keyword relationships for telecom domain
	relations := map[string][]string{
		"deploy":    {"orchestration", "5G", "Core", "RAN"},
		"create":    {"deployment", "network", "function"},
		"configure": {"QoS", "bearer", "session", "interface"},
		"network":   {"5G", "4G", "Core", "RAN", "slice"},
		"function":  {"AMF", "SMF", "UPF", "network"},
		"slice":     {"network slice", "slicing", "QoS", "orchestration"},
		"amf":       {"5G", "Core", "session", "mobility"},
		"smf":       {"5G", "Core", "PDU", "session"},
		"upf":       {"5G", "Core", "user plane", "bearer"},
	}

	lowerKeyword := strings.ToLower(keyword)
	for queryTerm, relatedTerms := range relations {
		if strings.Contains(query, queryTerm) {
			for _, related := range relatedTerms {
				if strings.Contains(lowerKeyword, strings.ToLower(related)) {
					return true
				}
			}
		}
	}

	return false
}

// updateMetrics safely updates metrics with a function
func (cb *ContextBuilder) updateMetrics(updateFunc func(*ContextBuilderMetrics)) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	updateFunc(cb.metrics)
}

func (cb *ContextBuilder) GetMetrics() map[string]interface{} {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	return map[string]interface{}{
		"context_builder_enabled":   true,
		"status":                    "active",
		"total_queries":             cb.metrics.TotalQueries,
		"successful_queries":        cb.metrics.SuccessfulQueries,
		"failed_queries":            cb.metrics.FailedQueries,
		"success_rate":              cb.getSuccessRate(),
		"average_query_duration_ms": cb.metrics.AverageQueryDuration.Milliseconds(),
		"average_documents_found":   cb.metrics.AverageDocumentsFound,
		"cache_hit_rate":            cb.getCacheHitRate(),
		"config": map[string]interface{}{
			"default_max_docs":        cb.config.DefaultMaxDocs,
			"max_context_length":      cb.config.MaxContextLength,
			"min_confidence_score":    cb.config.MinConfidenceScore,
			"enable_hybrid_search":    cb.config.EnableHybridSearch,
			"query_expansion_enabled": cb.config.QueryExpansionEnabled,
		},
	}
}

// getSuccessRate calculates the success rate percentage
func (cb *ContextBuilder) getSuccessRate() float64 {
	if cb.metrics.TotalQueries == 0 {
		return 0.0
	}
	return float64(cb.metrics.SuccessfulQueries) / float64(cb.metrics.TotalQueries) * 100.0
}

// getCacheHitRate calculates the cache hit rate percentage
func (cb *ContextBuilder) getCacheHitRate() float64 {
	totalCacheOps := cb.metrics.CacheHits + cb.metrics.CacheMisses
	if totalCacheOps == 0 {
		return 0.0
	}
	return float64(cb.metrics.CacheHits) / float64(totalCacheOps) * 100.0
}

// parseSearchResult converts a GraphQL result item to a SearchResult
func (cb *ContextBuilder) parseSearchResult(item map[string]interface{}) *shared.SearchResult {
	doc := &shared.TelecomDocument{}
	result := &shared.SearchResult{Document: doc}

	// Parse document fields
	if val, ok := item["content"].(string); ok {
		doc.Content = val
	}
	if val, ok := item["title"].(string); ok {
		doc.Title = val
	}
	if val, ok := item["source"].(string); ok {
		doc.Source = val
	}
	if val, ok := item["category"].(string); ok {
		doc.Category = val
	}
	if val, ok := item["version"].(string); ok {
		doc.Version = val
	}
	if val, ok := item["confidence"].(float64); ok {
		doc.Confidence = float32(val)
	}
	if val, ok := item["language"].(string); ok {
		doc.Language = val
	}
	if val, ok := item["documentType"].(string); ok {
		doc.DocumentType = val
	}

	// Parse array fields
	if val, ok := item["keywords"].([]interface{}); ok {
		keywords := make([]string, 0, len(val))
		for _, keyword := range val {
			if keywordStr, ok := keyword.(string); ok {
				keywords = append(keywords, keywordStr)
			}
		}
		doc.Keywords = keywords
	}

	if val, ok := item["networkFunction"].([]interface{}); ok {
		networkFunctions := make([]string, 0, len(val))
		for _, nf := range val {
			if nfStr, ok := nf.(string); ok {
				networkFunctions = append(networkFunctions, nfStr)
			}
		}
		doc.NetworkFunction = networkFunctions
	}

	if val, ok := item["technology"].([]interface{}); ok {
		technologies := make([]string, 0, len(val))
		for _, tech := range val {
			if techStr, ok := tech.(string); ok {
				technologies = append(technologies, techStr)
			}
		}
		doc.Technology = technologies
	}

	if val, ok := item["useCase"].([]interface{}); ok {
		useCases := make([]string, 0, len(val))
		for _, uc := range val {
			if ucStr, ok := uc.(string); ok {
				useCases = append(useCases, ucStr)
			}
		}
		doc.UseCase = useCases
	}

	// Parse additional fields
	if additional, ok := item["_additional"].(map[string]interface{}); ok {
		if id, ok := additional["id"].(string); ok {
			doc.ID = id
		}
		if score, ok := additional["score"].(float64); ok {
			result.Score = float32(score)
		}
		if distance, ok := additional["distance"].(float64); ok {
			result.Distance = float32(distance)
		}
	}

	// Set default confidence if not provided
	if doc.Confidence == 0 {
		doc.Confidence = 0.5 // Default confidence
	}

	return result
}

// RelevanceScorer stub implementation
type RelevanceScorer struct {
	impl *SimpleRelevanceScorer
}

func NewRelevanceScorer() *RelevanceScorer {
	return &RelevanceScorer{
		impl: NewSimpleRelevanceScorer(),
	}
}

// Score calculates the relevance score between a document and intent using semantic similarity
func (rs *RelevanceScorer) Score(ctx context.Context, doc string, intent string) (float32, error) {
	return rs.impl.Score(ctx, doc, intent)
}

func (rs *RelevanceScorer) GetMetrics() map[string]interface{} {
	return rs.impl.GetMetrics()
}

// RAGAwarePromptBuilder stub implementation
type RAGAwarePromptBuilder struct {
	config      *RAGPromptBuilderConfig
	logger      *slog.Logger
	metrics     *RAGPromptBuilderMetrics
	tokenLimits map[string]int
	mutex       sync.RWMutex
}

// RAGPromptBuilderConfig holds configuration for RAG-aware prompt building
type RAGPromptBuilderConfig struct {
	MaxPromptTokens           int     `json:"max_prompt_tokens"`
	MaxContextTokens          int     `json:"max_context_tokens"`
	ContextRelevanceThreshold float32 `json:"context_relevance_threshold"`
	MaxContextSources         int     `json:"max_context_sources"`
	TruncationStrategy        string  `json:"truncation_strategy"`
	EnableMetrics             bool    `json:"enable_metrics"`
	ContextSeparator          string  `json:"context_separator"`
	SourceAttributionFormat   string  `json:"source_attribution_format"`
	SystemPromptTemplate      string  `json:"system_prompt_template"`
}

// RAGPromptBuilderMetrics tracks prompt building metrics
type RAGPromptBuilderMetrics struct {
	TotalPrompts          int64         `json:"total_prompts"`
	ContextDocsUsed       int64         `json:"context_docs_used"`
	TruncatedPrompts      int64         `json:"truncated_prompts"`
	AverageTokens         float64       `json:"average_tokens"`
	AverageProcessingTime time.Duration `json:"average_processing_time"`
	mutex                 sync.RWMutex
}

func NewRAGAwarePromptBuilder() *RAGAwarePromptBuilder {
	config := &RAGPromptBuilderConfig{
		MaxPromptTokens:           4000,
		MaxContextTokens:          2500,
		ContextRelevanceThreshold: 0.3,
		MaxContextSources:         5,
		TruncationStrategy:        "smart",
		EnableMetrics:             true,
		ContextSeparator:          "\n\n---\n\n",
		SourceAttributionFormat:   "[{index}] Source: {source} | Title: {title} | Relevance: {score:.3f}",
		SystemPromptTemplate:      telecomSystemPromptTemplate,
	}

	tokenLimits := map[string]int{
		"gpt-4":           8192,
		"gpt-4-turbo":     128000,
		"gpt-4o":          128000,
		"gpt-4o-mini":     128000,
		"gpt-3.5-turbo":   16384,
		"claude-3":        200000,
		"claude-3-haiku":  200000,
		"claude-3-sonnet": 200000,
		"claude-3-opus":   200000,
		"default":         4000,
	}

	return &RAGAwarePromptBuilder{
		config:      config,
		logger:      slog.Default().With("component", "rag-aware-prompt-builder"),
		metrics:     &RAGPromptBuilderMetrics{},
		tokenLimits: tokenLimits,
	}
}

// Build creates a comprehensive prompt using intent and context documents from RAG system
func (rpb *RAGAwarePromptBuilder) Build(intent string, ctxDocs []map[string]any) string {
	startTime := time.Now()

	// Update metrics
	rpb.updateMetrics(func(m *RAGPromptBuilderMetrics) {
		m.TotalPrompts++
	})

	// Validate inputs
	if intent == "" {
		rpb.logger.Warn("Empty intent provided to prompt builder")
		return rpb.buildBasicPrompt(intent)
	}

	rpb.logger.Debug("Building RAG-aware prompt",
		"intent", intent,
		"context_docs_count", len(ctxDocs),
	)

	// Process and rank context documents
	processedContext := rpb.processContextDocuments(ctxDocs)

	// Build structured prompt template
	promptBuilder := strings.Builder{}

	// 1. System prompt with telecommunications expertise
	systemPrompt := rpb.buildSystemPrompt()
	promptBuilder.WriteString(systemPrompt)
	promptBuilder.WriteString("\n\n")

	// 2. Context section with relevant RAG documents
	if len(processedContext) > 0 {
		contextSection := rpb.buildContextSection(processedContext)
		promptBuilder.WriteString(contextSection)
		promptBuilder.WriteString("\n\n")
	}

	// 3. Telecommunications domain guidance
	domainGuidance := rpb.buildDomainGuidance(intent)
	promptBuilder.WriteString(domainGuidance)
	promptBuilder.WriteString("\n\n")

	// 4. Intent processing instructions
	intentInstructions := rpb.buildIntentInstructions(intent)
	promptBuilder.WriteString(intentInstructions)

	// 5. Apply intelligent truncation if needed
	finalPrompt := promptBuilder.String()
	finalPrompt = rpb.applyTokenTruncation(finalPrompt)

	// Update metrics
	processingTime := time.Since(startTime)
	tokenCount := rpb.estimateTokenCount(finalPrompt)

	rpb.updateMetrics(func(m *RAGPromptBuilderMetrics) {
		m.ContextDocsUsed += int64(len(processedContext))
		m.AverageProcessingTime = (m.AverageProcessingTime*time.Duration(m.TotalPrompts-1) + processingTime) / time.Duration(m.TotalPrompts)
		m.AverageTokens = (m.AverageTokens*float64(m.TotalPrompts-1) + float64(tokenCount)) / float64(m.TotalPrompts)
	})

	rpb.logger.Info("RAG-aware prompt built successfully",
		"token_count", tokenCount,
		"context_sources", len(processedContext),
		"processing_time", processingTime,
	)

	return finalPrompt
}

// processContextDocuments processes and ranks context documents from RAG system
func (rpb *RAGAwarePromptBuilder) processContextDocuments(ctxDocs []map[string]any) []map[string]any {
	if len(ctxDocs) == 0 {
		return nil
	}

	// Filter documents by relevance threshold
	relevantDocs := make([]map[string]any, 0)
	for _, doc := range ctxDocs {
		if rpb.isRelevantDocument(doc) {
			relevantDocs = append(relevantDocs, doc)
		}
	}

	// Sort by relevance score (highest first)
	rpb.sortDocumentsByRelevance(relevantDocs)

	// Limit to maximum sources
	maxSources := rpb.config.MaxContextSources
	if len(relevantDocs) > maxSources {
		relevantDocs = relevantDocs[:maxSources]
	}

	rpb.logger.Debug("Processed context documents",
		"total_docs", len(ctxDocs),
		"relevant_docs", len(relevantDocs),
		"max_sources", maxSources,
	)

	return relevantDocs
}

// isRelevantDocument checks if a document meets relevance criteria
func (rpb *RAGAwarePromptBuilder) isRelevantDocument(doc map[string]any) bool {
	// Check confidence/score threshold
	if score, exists := doc["score"]; exists {
		if scoreFloat, ok := score.(float64); ok {
			if float32(scoreFloat) < rpb.config.ContextRelevanceThreshold {
				return false
			}
		}
	}

	// Check confidence field
	if confidence, exists := doc["confidence"]; exists {
		if confidenceFloat, ok := confidence.(float64); ok {
			if float32(confidenceFloat) < rpb.config.ContextRelevanceThreshold {
				return false
			}
		}
	}

	// Check that document has content
	if content, exists := doc["content"]; exists {
		if contentStr, ok := content.(string); ok {
			if len(strings.TrimSpace(contentStr)) == 0 {
				return false
			}
		} else {
			return false
		}
	} else {
		return false
	}

	return true
}

// sortDocumentsByRelevance sorts documents by relevance score (highest first)
func (rpb *RAGAwarePromptBuilder) sortDocumentsByRelevance(docs []map[string]any) {
	// Simple bubble sort by score/confidence (highest first)
	for i := 0; i < len(docs)-1; i++ {
		for j := i + 1; j < len(docs); j++ {
			score1 := rpb.getDocumentScore(docs[i])
			score2 := rpb.getDocumentScore(docs[j])
			if score1 < score2 {
				docs[i], docs[j] = docs[j], docs[i]
			}
		}
	}
}

// getDocumentScore extracts relevance score from document
func (rpb *RAGAwarePromptBuilder) getDocumentScore(doc map[string]any) float64 {
	// Try score field first
	if score, exists := doc["score"]; exists {
		if scoreFloat, ok := score.(float64); ok {
			return scoreFloat
		}
		if scoreFloat32, ok := score.(float32); ok {
			return float64(scoreFloat32)
		}
	}

	// Try confidence field
	if confidence, exists := doc["confidence"]; exists {
		if confidenceFloat, ok := confidence.(float64); ok {
			return confidenceFloat
		}
		if confidenceFloat32, ok := confidence.(float32); ok {
			return float64(confidenceFloat32)
		}
	}

	// Default score
	return 0.5
}

// buildSystemPrompt creates the system prompt with telecommunications expertise
func (rpb *RAGAwarePromptBuilder) buildSystemPrompt() string {
	return rpb.config.SystemPromptTemplate
}

// buildContextSection creates the context section from processed documents
func (rpb *RAGAwarePromptBuilder) buildContextSection(docs []map[string]any) string {
	if len(docs) == 0 {
		return ""
	}

	var contextBuilder strings.Builder
	contextBuilder.WriteString("## Technical Documentation Context\n\n")
	contextBuilder.WriteString("The following technical documentation is relevant to your request:\n\n")

	for i, doc := range docs {
		contextBuilder.WriteString(rpb.formatContextDocument(doc, i+1))
		if i < len(docs)-1 {
			contextBuilder.WriteString(rpb.config.ContextSeparator)
		}
	}

	return contextBuilder.String()
}

// formatContextDocument formats a single context document for the prompt
func (rpb *RAGAwarePromptBuilder) formatContextDocument(doc map[string]any, index int) string {
	var docBuilder strings.Builder

	// Document header with attribution
	score := rpb.getDocumentScore(doc)
	source := rpb.getStringValue(doc, "source", "Unknown Source")
	title := rpb.getStringValue(doc, "title", "Untitled Document")

	attribution := strings.ReplaceAll(rpb.config.SourceAttributionFormat, "{index}", fmt.Sprintf("%d", index))
	attribution = strings.ReplaceAll(attribution, "{source}", source)
	attribution = strings.ReplaceAll(attribution, "{title}", title)
	attribution = strings.ReplaceAll(attribution, "{score:.3f}", fmt.Sprintf("%.3f", score))

	docBuilder.WriteString(attribution)
	docBuilder.WriteString("\n\n")

	// Document metadata
	if category := rpb.getStringValue(doc, "category", ""); category != "" {
		docBuilder.WriteString(fmt.Sprintf("**Category:** %s\n", category))
	}
	if version := rpb.getStringValue(doc, "version", ""); version != "" {
		docBuilder.WriteString(fmt.Sprintf("**Version:** %s\n", version))
	}
	if documentType := rpb.getStringValue(doc, "document_type", ""); documentType != "" {
		docBuilder.WriteString(fmt.Sprintf("**Type:** %s\n", documentType))
	}

	// Technology and network function tags
	if technologies := rpb.getStringArrayValue(doc, "technology"); len(technologies) > 0 {
		docBuilder.WriteString(fmt.Sprintf("**Technologies:** %s\n", strings.Join(technologies, ", ")))
	}
	if networkFunctions := rpb.getStringArrayValue(doc, "network_function"); len(networkFunctions) > 0 {
		docBuilder.WriteString(fmt.Sprintf("**Network Functions:** %s\n", strings.Join(networkFunctions, ", ")))
	}

	docBuilder.WriteString("\n")

	// Document content
	content := rpb.getStringValue(doc, "content", "")
	if content != "" {
		// Truncate content if too long
		maxContentLength := 1500 // Reserve tokens for other parts
		if len(content) > maxContentLength {
			content = content[:maxContentLength] + "...[truncated]"
			rpb.updateMetrics(func(m *RAGPromptBuilderMetrics) {
				m.TruncatedPrompts++
			})
		}
		docBuilder.WriteString("**Content:**\n")
		docBuilder.WriteString(content)
	}

	return docBuilder.String()
}

// buildDomainGuidance creates telecommunications domain-specific guidance
func (rpb *RAGAwarePromptBuilder) buildDomainGuidance(intent string) string {
	var guidanceBuilder strings.Builder
	guidanceBuilder.WriteString("## Telecommunications Domain Guidance\n\n")

	// Classify intent domain
	domain := rpb.classifyIntentDomain(intent)

	switch domain {
	case "RAN":
		guidanceBuilder.WriteString("**Radio Access Network (RAN) Focus:**\n")
		guidanceBuilder.WriteString("- Consider gNodeB configurations, cell planning, and radio parameters\n")
		guidanceBuilder.WriteString("- Address mobility management, handover procedures, and interference\n")
		guidanceBuilder.WriteString("- Include O-RAN interfaces (A1, O1, E2) and disaggregated architecture\n")
		guidanceBuilder.WriteString("- Reference 3GPP TS 38.xxx series specifications\n")

	case "Core":
		guidanceBuilder.WriteString("**5G Core Network Focus:**\n")
		guidanceBuilder.WriteString("- Focus on network functions: AMF, SMF, UPF, AUSF, UDM, PCF, NSSF\n")
		guidanceBuilder.WriteString("- Address service-based architecture and NF interactions\n")
		guidanceBuilder.WriteString("- Include network slicing, QoS flows, and session management\n")
		guidanceBuilder.WriteString("- Reference 3GPP TS 23.xxx and TS 29.xxx specifications\n")

	case "Transport":
		guidanceBuilder.WriteString("**Transport Network Focus:**\n")
		guidanceBuilder.WriteString("- Address IP/MPLS backbone, routing protocols, and QoS mechanisms\n")
		guidanceBuilder.WriteString("- Include fronthaul/backhaul connectivity and latency requirements\n")
		guidanceBuilder.WriteString("- Consider network synchronization and timing distribution\n")
		guidanceBuilder.WriteString("- Reference transport-specific standards and best practices\n")

	case "Management":
		guidanceBuilder.WriteString("**Network Management & Orchestration Focus:**\n")
		guidanceBuilder.WriteString("- Address FCAPS (Fault, Configuration, Accounting, Performance, Security)\n")
		guidanceBuilder.WriteString("- Include automation, orchestration, and lifecycle management\n")
		guidanceBuilder.WriteString("- Consider cloud-native principles and Kubernetes deployment\n")
		guidanceBuilder.WriteString("- Reference ETSI NFV MANO and cloud management standards\n")

	default:
		guidanceBuilder.WriteString("**General Telecommunications Focus:**\n")
		guidanceBuilder.WriteString("- Consider the complete network architecture and end-to-end flows\n")
		guidanceBuilder.WriteString("- Address interoperability between different network domains\n")
		guidanceBuilder.WriteString("- Include standards compliance and best practices\n")
		guidanceBuilder.WriteString("- Consider both technical and operational aspects\n")
	}

	return guidanceBuilder.String()
}

// buildIntentInstructions creates specific instructions for intent processing
func (rpb *RAGAwarePromptBuilder) buildIntentInstructions(intent string) string {
	var instructionsBuilder strings.Builder
	instructionsBuilder.WriteString("## Intent Processing Instructions\n\n")

	instructionsBuilder.WriteString("**User Intent:** ")
	instructionsBuilder.WriteString(intent)
	instructionsBuilder.WriteString("\n\n")

	instructionsBuilder.WriteString("**Processing Guidelines:**\n")
	instructionsBuilder.WriteString("1. **Analyze the intent** for specific requirements, constraints, and objectives\n")
	instructionsBuilder.WriteString("2. **Apply technical context** from the provided documentation\n")
	instructionsBuilder.WriteString("3. **Generate actionable recommendations** with specific parameters and configurations\n")
	instructionsBuilder.WriteString("4. **Include implementation steps** with proper sequencing and dependencies\n")
	instructionsBuilder.WriteString("5. **Address potential risks** and mitigation strategies\n")
	instructionsBuilder.WriteString("6. **Validate against standards** and industry best practices\n")
	instructionsBuilder.WriteString("7. **Provide measurable outcomes** and success criteria\n\n")

	// Intent-specific instructions
	intentType := rpb.classifyIntentType(intent)
	switch intentType {
	case "deployment":
		instructionsBuilder.WriteString("**Deployment-Specific Instructions:**\n")
		instructionsBuilder.WriteString("- Specify resource requirements and dependencies\n")
		instructionsBuilder.WriteString("- Include configuration parameters and initial settings\n")
		instructionsBuilder.WriteString("- Address scaling, redundancy, and high availability\n")
		instructionsBuilder.WriteString("- Provide validation and testing procedures\n")

	case "configuration":
		instructionsBuilder.WriteString("**Configuration-Specific Instructions:**\n")
		instructionsBuilder.WriteString("- Provide specific parameter values and ranges\n")
		instructionsBuilder.WriteString("- Include validation commands and verification steps\n")
		instructionsBuilder.WriteString("- Address configuration dependencies and order of operations\n")
		instructionsBuilder.WriteString("- Mention rollback procedures if changes fail\n")

	case "troubleshooting":
		instructionsBuilder.WriteString("**Troubleshooting-Specific Instructions:**\n")
		instructionsBuilder.WriteString("- Provide systematic diagnostic procedures\n")
		instructionsBuilder.WriteString("- Include specific KPIs and metrics to check\n")
		instructionsBuilder.WriteString("- Address common root causes and resolution steps\n")
		instructionsBuilder.WriteString("- Suggest preventive measures and monitoring\n")

	case "optimization":
		instructionsBuilder.WriteString("**Optimization-Specific Instructions:**\n")
		instructionsBuilder.WriteString("- Identify performance bottlenecks and improvement opportunities\n")
		instructionsBuilder.WriteString("- Provide specific tuning parameters and recommended values\n")
		instructionsBuilder.WriteString("- Include measurement baselines and target metrics\n")
		instructionsBuilder.WriteString("- Address trade-offs and potential side effects\n")
	}

	return instructionsBuilder.String()
}

// Helper methods for domain and intent classification
func (rpb *RAGAwarePromptBuilder) classifyIntentDomain(intent string) string {
	intentLower := strings.ToLower(intent)

	// RAN keywords
	ranKeywords := []string{"gnb", "enb", "cell", "radio", "antenna", "handover", "mobility", "rf", "o-ran", "oran"}
	for _, keyword := range ranKeywords {
		if strings.Contains(intentLower, keyword) {
			return "RAN"
		}
	}

	// Core keywords
	coreKeywords := []string{"amf", "smf", "upf", "ausf", "udm", "pcf", "nrf", "nssf", "core", "5gc", "session", "slice"}
	for _, keyword := range coreKeywords {
		if strings.Contains(intentLower, keyword) {
			return "Core"
		}
	}

	// Transport keywords
	transportKeywords := []string{"transport", "ip", "mpls", "routing", "switching", "backhaul", "fronthaul"}
	for _, keyword := range transportKeywords {
		if strings.Contains(intentLower, keyword) {
			return "Transport"
		}
	}

	// Management keywords
	managementKeywords := []string{"orchestration", "management", "monitoring", "automation", "lifecycle", "mano"}
	for _, keyword := range managementKeywords {
		if strings.Contains(intentLower, keyword) {
			return "Management"
		}
	}

	return "General"
}

func (rpb *RAGAwarePromptBuilder) classifyIntentType(intent string) string {
	intentLower := strings.ToLower(intent)

	if strings.Contains(intentLower, "deploy") || strings.Contains(intentLower, "install") || strings.Contains(intentLower, "create") {
		return "deployment"
	}
	if strings.Contains(intentLower, "configure") || strings.Contains(intentLower, "config") || strings.Contains(intentLower, "setup") {
		return "configuration"
	}
	if strings.Contains(intentLower, "troubleshoot") || strings.Contains(intentLower, "debug") || strings.Contains(intentLower, "fix") || strings.Contains(intentLower, "issue") {
		return "troubleshooting"
	}
	if strings.Contains(intentLower, "optimize") || strings.Contains(intentLower, "improve") || strings.Contains(intentLower, "tune") || strings.Contains(intentLower, "enhance") {
		return "optimization"
	}

	return "general"
}

// applyTokenTruncation applies intelligent truncation to stay within token limits
func (rpb *RAGAwarePromptBuilder) applyTokenTruncation(prompt string) string {
	tokenCount := rpb.estimateTokenCount(prompt)
	maxTokens := rpb.config.MaxPromptTokens

	if tokenCount <= maxTokens {
		return prompt
	}

	rpb.logger.Debug("Applying token truncation",
		"current_tokens", tokenCount,
		"max_tokens", maxTokens,
		"strategy", rpb.config.TruncationStrategy,
	)

	switch rpb.config.TruncationStrategy {
	case "smart":
		return rpb.smartTruncation(prompt, maxTokens)
	case "tail":
		return rpb.tailTruncation(prompt, maxTokens)
	default:
		return rpb.simpleTruncation(prompt, maxTokens)
	}
}

// smartTruncation applies intelligent truncation preserving important sections
func (rpb *RAGAwarePromptBuilder) smartTruncation(prompt string, maxTokens int) string {
	// Split prompt into sections
	sections := strings.Split(prompt, "## ")
	if len(sections) <= 1 {
		return rpb.simpleTruncation(prompt, maxTokens)
	}

	// Preserve system prompt and intent instructions
	var preservedSections []string
	var optionalSections []string

	for i, section := range sections {
		if i == 0 {
			preservedSections = append(preservedSections, section) // System prompt
			continue
		}

		sectionHeader := strings.ToLower(section)
		if strings.HasPrefix(sectionHeader, "intent processing") {
			preservedSections = append(preservedSections, "## "+section)
		} else {
			optionalSections = append(optionalSections, "## "+section)
		}
	}

	// Build truncated prompt
	var builder strings.Builder
	for _, section := range preservedSections {
		builder.WriteString(section)
		builder.WriteString("\n\n")
	}

	// Add optional sections if they fit
	reservedTokens := rpb.estimateTokenCount(builder.String())
	availableTokens := maxTokens - reservedTokens

	for _, section := range optionalSections {
		sectionTokens := rpb.estimateTokenCount(section)
		if sectionTokens <= availableTokens {
			builder.WriteString(section)
			builder.WriteString("\n\n")
			availableTokens -= sectionTokens
		}
	}

	result := strings.TrimSpace(builder.String())
	rpb.updateMetrics(func(m *RAGPromptBuilderMetrics) {
		m.TruncatedPrompts++
	})

	return result
}

// tailTruncation truncates from the end
func (rpb *RAGAwarePromptBuilder) tailTruncation(prompt string, maxTokens int) string {
	// Simple character-based truncation (rough approximation)
	avgCharsPerToken := 4
	maxChars := maxTokens * avgCharsPerToken

	if len(prompt) <= maxChars {
		return prompt
	}

	rpb.updateMetrics(func(m *RAGPromptBuilderMetrics) {
		m.TruncatedPrompts++
	})

	return prompt[:maxChars] + "...[truncated]"
}

// simpleTruncation applies basic truncation
func (rpb *RAGAwarePromptBuilder) simpleTruncation(prompt string, maxTokens int) string {
	return rpb.tailTruncation(prompt, maxTokens)
}

// estimateTokenCount provides rough token count estimation
func (rpb *RAGAwarePromptBuilder) estimateTokenCount(text string) int {
	// Simple estimation: ~4 characters per token on average
	return len(text) / 4
}

// buildBasicPrompt creates a basic prompt when no context is available
func (rpb *RAGAwarePromptBuilder) buildBasicPrompt(intent string) string {
	var builder strings.Builder

	builder.WriteString(rpb.buildSystemPrompt())
	builder.WriteString("\n\n")
	builder.WriteString("## User Intent\n\n")
	if intent != "" {
		builder.WriteString(intent)
	} else {
		builder.WriteString("No specific intent provided.")
	}

	return builder.String()
}

// Utility methods for extracting values from context documents
func (rpb *RAGAwarePromptBuilder) getStringValue(doc map[string]any, key string, defaultValue string) string {
	if value, exists := doc[key]; exists {
		if strValue, ok := value.(string); ok {
			return strValue
		}
	}
	return defaultValue
}

func (rpb *RAGAwarePromptBuilder) getStringArrayValue(doc map[string]any, key string) []string {
	if value, exists := doc[key]; exists {
		if arrayValue, ok := value.([]interface{}); ok {
			result := make([]string, 0, len(arrayValue))
			for _, item := range arrayValue {
				if strItem, ok := item.(string); ok {
					result = append(result, strItem)
				}
			}
			return result
		}
		if arrayValue, ok := value.([]string); ok {
			return arrayValue
		}
	}
	return nil
}

// updateMetrics safely updates metrics
func (rpb *RAGAwarePromptBuilder) updateMetrics(updater func(*RAGPromptBuilderMetrics)) {
	if !rpb.config.EnableMetrics {
		return
	}
	rpb.mutex.Lock()
	defer rpb.mutex.Unlock()
	updater(rpb.metrics)
}

func (rpb *RAGAwarePromptBuilder) GetMetrics() map[string]interface{} {
	rpb.mutex.RLock()
	defer rpb.mutex.RUnlock()

	return map[string]interface{}{
		"prompt_builder_enabled":     true,
		"status":                     "active",
		"total_prompts":              rpb.metrics.TotalPrompts,
		"context_docs_used":          rpb.metrics.ContextDocsUsed,
		"truncated_prompts":          rpb.metrics.TruncatedPrompts,
		"average_tokens":             rpb.metrics.AverageTokens,
		"average_processing_time_ms": rpb.metrics.AverageProcessingTime.Milliseconds(),
		"config": map[string]interface{}{
			"max_prompt_tokens":   rpb.config.MaxPromptTokens,
			"max_context_tokens":  rpb.config.MaxContextTokens,
			"max_context_sources": rpb.config.MaxContextSources,
			"truncation_strategy": rpb.config.TruncationStrategy,
		},
	}
}

// telecomSystemPromptTemplate provides the default system prompt for telecommunications
const telecomSystemPromptTemplate = `You are an expert telecommunications engineer and network architect with deep expertise in:

**5G and Next-Generation Networks:**
- 5G Core Network functions (AMF, SMF, UPF, AUSF, UDM, PCF, NRF, NSSF)
- Service-based architecture and network function interactions
- Network slicing, QoS flows, and session management procedures
- 3GPP specifications and standards compliance

**O-RAN and Radio Access Networks:**
- Open RAN architecture and disaggregated network functions
- O-RAN interfaces (A1, O1, O2, E2) and their implementations
- RAN Intelligent Controller (RIC) applications and xApps
- Radio resource management and optimization

**Cloud-Native Network Functions:**
- Containerized and virtualized network function deployment
- Kubernetes orchestration and cloud-native principles
- Microservices architecture and service mesh integration
- DevOps practices for network function lifecycle management

**Network Orchestration and Management:**
- ETSI NFV MANO framework and orchestration platforms
- Automated network provisioning and configuration management
- Network monitoring, analytics, and performance optimization
- Intent-driven networking and closed-loop automation

**Your Role:**
Provide precise, actionable technical guidance based on industry standards, best practices, and the specific context provided. Focus on practical implementation details, configuration parameters, and operational considerations. When providing recommendations:

1. Reference specific standards and specifications where applicable
2. Include concrete configuration examples and parameter values
3. Address potential issues and provide troubleshooting guidance
4. Consider scalability, security, and performance implications
5. Ensure recommendations align with cloud-native and automation principles

**Response Style:**
- Technical accuracy and precision are paramount
- Use industry-standard terminology and conventions
- Provide structured, actionable recommendations
- Include relevant context about impacts and dependencies
- Distinguish between mandatory requirements and best practices`
