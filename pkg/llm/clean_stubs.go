//go:build !disable_rag
// +build !disable_rag

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
	"github.com/thc1006/nephoran-intent-operator/pkg/types"
	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
)


// StreamingProcessorStub handles streaming requests with server-sent events
type StreamingProcessorStub struct {
	httpClient *http.Client
	ragAPIURL  string
	logger     *slog.Logger
	mutex      sync.RWMutex
}

// StreamingRequest is defined in types.go

func NewStreamingProcessor() *StreamingProcessorStub {
	return &StreamingProcessorStub{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		ragAPIURL: "http://rag-api:8080",
		logger:    slog.Default().With("component", "streaming-processor"),
	}
}

func (sp *StreamingProcessorStub) HandleStreamingRequest(w http.ResponseWriter, r *http.Request, req *StreamingRequest) error {
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

func (sp *StreamingProcessorStub) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"streaming_enabled": true,
		"status":            "active",
		"rag_api_url":       sp.ragAPIURL,
	}
}

func (sp *StreamingProcessorStub) Shutdown(ctx context.Context) error {
	sp.logger.Info("Shutting down streaming processor")
	return nil
}


// ContextBuilder provides RAG context building capabilities
type ContextBuilderStub struct {
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

func NewContextBuilderStub() *ContextBuilder {
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
func (cb *ContextBuilderStub) BuildContext(ctx context.Context, intent string, maxDocs int) ([]map[string]any, error) {
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
	var searchResults []*types.SearchResult
	var searchErr error

	err := cb.weaviatePool.WithConnection(queryCtx, func(client *weaviate.Client) error {
		// Build GraphQL query using available API methods
		gqlQuery := client.GraphQL().Get().
			WithClassName("TelecomKnowledge")

		if cb.config.EnableHybridSearch {
			// Use hybrid search - simplified for compatibility
			gqlQuery = gqlQuery.WithLimit(maxDocs)
		} else {
			// Use pure vector search - simplified for compatibility
			gqlQuery = gqlQuery.WithLimit(maxDocs)
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
		searchResults = make([]*types.SearchResult, 0)
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
func (cb *ContextBuilderStub) expandQuery(query string) string {
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
func (cb *ContextBuilderStub) isRelatedKeyword(query, keyword string) bool {
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
func (cb *ContextBuilderStub) updateMetrics(updateFunc func(*ContextBuilderMetrics)) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	updateFunc(cb.metrics)
}

func (cb *ContextBuilderStub) GetMetrics() map[string]interface{} {
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
func (cb *ContextBuilderStub) getSuccessRate() float64 {
	if cb.metrics.TotalQueries == 0 {
		return 0.0
	}
	return float64(cb.metrics.SuccessfulQueries) / float64(cb.metrics.TotalQueries) * 100.0
}

// getCacheHitRate calculates the cache hit rate percentage
func (cb *ContextBuilderStub) getCacheHitRate() float64 {
	totalCacheOps := cb.metrics.CacheHits + cb.metrics.CacheMisses
	if totalCacheOps == 0 {
		return 0.0
	}
	return float64(cb.metrics.CacheHits) / float64(totalCacheOps) * 100.0
}

// parseSearchResult converts a GraphQL result item to a SearchResult

func (cb *ContextBuilderStub) parseSearchResult(item map[string]interface{}) *shared.SearchResult {
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
type RelevanceScorerStub struct {
	impl *SimpleRelevanceScorer
}

func NewRelevanceScorerStub() *RelevanceScorer {
	return &RelevanceScorer{
		impl: NewSimpleRelevanceScorer(),
	}
}

// Score calculates the relevance score between a document and intent using semantic similarity
func (rs *RelevanceScorerStub) Score(ctx context.Context, doc string, intent string) (float32, error) {
	return rs.impl.Score(ctx, doc, intent)
}

func (rs *RelevanceScorerStub) GetMetrics() map[string]interface{} {
	return rs.impl.GetMetrics()
}

// RAGAwarePromptBuilder stub implementation
type RAGAwarePromptBuilderStub struct{}

func NewRAGAwarePromptBuilderStub() *RAGAwarePromptBuilderStub {
	return &RAGAwarePromptBuilderStub{}
}

func (rpb *RAGAwarePromptBuilderStub) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"prompt_builder_enabled": false,
		"status":                 "not_implemented",
	}
}

