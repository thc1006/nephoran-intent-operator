//go:build !disable_rag && !test

package rag

import (
	"context"
	"crypto/md5"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// OptimizedRAGPipeline provides a high-performance RAG processing pipeline.

type OptimizedRAGPipeline struct {
	config *RAGPipelineConfig

	semanticCache *SemanticCache

	queryPreprocessor *QueryPreprocessor

	resultAggregator *ResultAggregator

	embeddingCache EmbeddingCache

	weaviateClient WeaviateClient

	batchSearchClient *OptimizedBatchSearchClient

	connectionPool *OptimizedConnectionPool

	logger *slog.Logger

	metrics *RAGPipelineMetrics
}

// RAGPipelineConfig holds configuration for the optimized RAG pipeline.

type RAGPipelineConfig struct {

	// Caching configuration.

	EnableSemanticCache bool `json:"enable_semantic_cache"`

	SemanticCacheSize int `json:"semantic_cache_size"`

	SemanticCacheTTL time.Duration `json:"semantic_cache_ttl"`

	SemanticSimilarityThreshold float32 `json:"semantic_similarity_threshold"`

	// Query preprocessing.

	EnableQueryPreprocessing bool `json:"enable_query_preprocessing"`

	EnableQueryExpansion bool `json:"enable_query_expansion"`

	EnableQueryNormalization bool `json:"enable_query_normalization"`

	EnableTelecomNER bool `json:"enable_telecom_ner"`

	// Result processing.

	EnableResultAggregation bool `json:"enable_result_aggregation"`

	EnableResultDeduplication bool `json:"enable_result_deduplication"`

	EnableResultRanking bool `json:"enable_result_ranking"`

	MaxResultsPerQuery int `json:"max_results_per_query"`

	// Performance optimizations.

	EnableBatchProcessing bool `json:"enable_batch_processing"`

	BatchSize int `json:"batch_size"`

	MaxConcurrency int `json:"max_concurrency"`

	ProcessingTimeout time.Duration `json:"processing_timeout"`

	// Embedding caching.

	EnableEmbeddingCache bool `json:"enable_embedding_cache"`

	EmbeddingCacheSize int `json:"embedding_cache_size"`

	EmbeddingCacheTTL time.Duration `json:"embedding_cache_ttl"`

	// Quality thresholds.

	MinSemanticSimilarity float32 `json:"min_semantic_similarity"`

	MinResultConfidence float32 `json:"min_result_confidence"`

	MaxLatencyTarget time.Duration `json:"max_latency_target"`
}

// SemanticCache provides intelligent semantic-aware caching.

type SemanticCache struct {
	entries map[string]*SemanticCacheEntry

	vectors map[string][]float32

	config *RAGPipelineConfig

	mutex sync.RWMutex

	metrics *SemanticCacheMetrics

	logger *slog.Logger
}

// SemanticCacheEntry represents a cached semantic result.

type SemanticCacheEntry struct {
	Query string `json:"query"`

	QueryVector []float32 `json:"query_vector"`

	Result *RAGResponse `json:"result"`

	CreatedAt time.Time `json:"created_at"`

	LastAccessed time.Time `json:"last_accessed"`

	AccessCount int64 `json:"access_count"`

	SemanticHash string `json:"semantic_hash"`

	OriginalQueries []string `json:"original_queries"`
}

// SemanticCacheMetrics tracks semantic cache performance.

type SemanticCacheMetrics struct {
	TotalQueries int64 `json:"total_queries"`

	SemanticHits int64 `json:"semantic_hits"`

	ExactHits int64 `json:"exact_hits"`

	Misses int64 `json:"misses"`

	SemanticHitRate float64 `json:"semantic_hit_rate"`

	AverageSemanticScore float32 `json:"average_semantic_score"`

	CacheEvictions int64 `json:"cache_evictions"`

	mutex sync.RWMutex
}

// QueryPreprocessor handles intelligent query preprocessing.

type QueryPreprocessor struct {
	config *RAGPipelineConfig

	telecomTerms map[string][]string

	acronymExpansions map[string]string

	synonyms map[string][]string

	stopWords map[string]bool

	nerPatterns map[string]string

	logger *slog.Logger

	metrics *QueryPreprocessorMetrics
}

// QueryPreprocessorMetrics tracks preprocessing performance.

type QueryPreprocessorMetrics struct {
	TotalQueries int64 `json:"total_queries"`

	ProcessedQueries int64 `json:"processed_queries"`

	ExpandedQueries int64 `json:"expanded_queries"`

	NormalizedQueries int64 `json:"normalized_queries"`

	ExtractedEntities int64 `json:"extracted_entities"`

	AverageProcessingTime time.Duration `json:"average_processing_time"`

	mutex sync.RWMutex
}

// ResultAggregator handles intelligent result aggregation and ranking.

type ResultAggregator struct {
	config *RAGPipelineConfig

	logger *slog.Logger

	metrics *ResultAggregatorMetrics
}

// ResultAggregatorMetrics tracks aggregation performance.

type ResultAggregatorMetrics struct {
	TotalAggregations int64 `json:"total_aggregations"`

	DeduplicatedResults int64 `json:"deduplicated_results"`

	RankedResults int64 `json:"ranked_results"`

	AverageAggregationTime time.Duration `json:"average_aggregation_time"`

	mutex sync.RWMutex
}

// Note: EmbeddingCache interface and related types are defined in embedding_service.go.

// RAGPipelineMetrics tracks overall pipeline performance.

type RAGPipelineMetrics struct {
	TotalPipelineRuns int64 `json:"total_pipeline_runs"`

	SuccessfulRuns int64 `json:"successful_runs"`

	FailedRuns int64 `json:"failed_runs"`

	AverageLatency time.Duration `json:"average_latency"`

	LatencyP95 time.Duration `json:"latency_p95"`

	LatencyP99 time.Duration `json:"latency_p99"`

	CacheUtilization float64 `json:"cache_utilization"`

	BatchingEfficiency float64 `json:"batching_efficiency"`

	PreprocessingTime time.Duration `json:"preprocessing_time"`

	SearchTime time.Duration `json:"search_time"`

	AggregationTime time.Duration `json:"aggregation_time"`

	mutex sync.RWMutex
}

// ProcessedQuery represents a preprocessed query.

type ProcessedQuery struct {
	OriginalQuery string `json:"original_query"`

	NormalizedQuery string `json:"normalized_query"`

	ExpandedQuery string `json:"expanded_query"`

	ExtractedEntities map[string]string `json:"extracted_entities"`

	Synonyms []string `json:"synonyms"`

	TelecomTerms []string `json:"telecom_terms"`

	ProcessingTime time.Duration `json:"processing_time"`
}

// NewOptimizedRAGPipeline creates a new optimized RAG pipeline.

func NewOptimizedRAGPipeline(

	weaviateClient WeaviateClient,

	batchSearchClient *OptimizedBatchSearchClient,

	connectionPool *OptimizedConnectionPool,

	config *RAGPipelineConfig,

) *OptimizedRAGPipeline {

	if config == nil {

		config = getDefaultRAGPipelineConfig()

	}

	logger := slog.Default().With("component", "optimized-rag-pipeline")

	// Initialize semantic cache.

	semanticCache := &SemanticCache{

		entries: make(map[string]*SemanticCacheEntry),

		vectors: make(map[string][]float32),

		config: config,

		metrics: &SemanticCacheMetrics{},

		logger: logger,
	}

	// Initialize query preprocessor.

	queryPreprocessor := &QueryPreprocessor{

		config: config,

		telecomTerms: initializeTelecomTerms(),

		acronymExpansions: initializeAcronymExpansions(),

		synonyms: initializeSynonyms(),

		stopWords: initializeStopWords(),

		nerPatterns: initializeNERPatterns(),

		logger: logger,

		metrics: &QueryPreprocessorMetrics{},
	}

	// Initialize result aggregator.

	resultAggregator := &ResultAggregator{

		config: config,

		logger: logger,

		metrics: &ResultAggregatorMetrics{},
	}

	// Initialize embedding cache - use the in-memory implementation.

	embeddingCache := NewInMemoryCache(int64(config.EmbeddingCacheSize))

	pipeline := &OptimizedRAGPipeline{

		config: config,

		semanticCache: semanticCache,

		queryPreprocessor: queryPreprocessor,

		resultAggregator: resultAggregator,

		embeddingCache: embeddingCache,

		weaviateClient: weaviateClient,

		batchSearchClient: batchSearchClient,

		connectionPool: connectionPool,

		logger: logger,

		metrics: &RAGPipelineMetrics{},
	}

	// Start background maintenance.

	go pipeline.startBackgroundMaintenance()

	logger.Info("Optimized RAG pipeline created",

		"semantic_cache_enabled", config.EnableSemanticCache,

		"preprocessing_enabled", config.EnableQueryPreprocessing,

		"batch_processing_enabled", config.EnableBatchProcessing,
	)

	return pipeline

}

// getDefaultRAGPipelineConfig returns default pipeline configuration.

func getDefaultRAGPipelineConfig() *RAGPipelineConfig {

	return &RAGPipelineConfig{

		EnableSemanticCache: true,

		SemanticCacheSize: 10000,

		SemanticCacheTTL: time.Hour,

		SemanticSimilarityThreshold: 0.85,

		EnableQueryPreprocessing: true,

		EnableQueryExpansion: true,

		EnableQueryNormalization: true,

		EnableTelecomNER: true,

		EnableResultAggregation: true,

		EnableResultDeduplication: true,

		EnableResultRanking: true,

		MaxResultsPerQuery: 50,

		EnableBatchProcessing: true,

		BatchSize: 20,

		MaxConcurrency: 10,

		ProcessingTimeout: 30 * time.Second,

		EnableEmbeddingCache: true,

		EmbeddingCacheSize: 5000,

		EmbeddingCacheTTL: 2 * time.Hour,

		MinSemanticSimilarity: 0.7,

		MinResultConfidence: 0.6,

		MaxLatencyTarget: 500 * time.Millisecond,
	}

}

// ProcessQuery processes a query through the optimized RAG pipeline.

func (p *OptimizedRAGPipeline) ProcessQuery(ctx context.Context, request *RAGRequest) (*RAGResponse, error) {

	startTime := time.Now()

	// Step 1: Check semantic cache.

	if p.config.EnableSemanticCache {

		if cached := p.semanticCache.GetSimilar(request.Query); cached != nil {

			p.updatePipelineMetrics(true, time.Since(startTime), "cache_hit")

			p.logger.Debug("Semantic cache hit", "query", request.Query, "similarity", cached.SemanticHash)

			return cached.Result, nil

		}

	}

	// Step 2: Preprocess query.

	var processedQuery *ProcessedQuery

	if p.config.EnableQueryPreprocessing {

		processedQuery = p.queryPreprocessor.Process(request.Query)

		request.Query = processedQuery.ExpandedQuery // Use expanded query for search

	}

	// Step 3: Execute search with optimization.

	searchResponse, err := p.executeOptimizedSearch(ctx, request)

	if err != nil {

		p.updatePipelineMetrics(false, time.Since(startTime), "search_error")

		return nil, fmt.Errorf("optimized search failed: %w", err)

	}

	// Step 4: Aggregate and rank results.

	if p.config.EnableResultAggregation {

		searchResponse = p.resultAggregator.AggregateResults(searchResponse)

	}

	// Step 5: Cache the result.

	if p.config.EnableSemanticCache {

		p.semanticCache.Store(request.Query, searchResponse)

	}

	p.updatePipelineMetrics(true, time.Since(startTime), "success")

	p.logger.Info("RAG pipeline completed",

		"query", request.Query,

		"processing_time", time.Since(startTime),

		"results", len(searchResponse.SourceDocuments),
	)

	return searchResponse, nil

}

// ProcessBatch processes multiple queries in an optimized batch.

func (p *OptimizedRAGPipeline) ProcessBatch(ctx context.Context, requests []*RAGRequest) ([]*RAGResponse, error) {

	if len(requests) == 0 {

		return nil, fmt.Errorf("no requests provided")

	}

	startTime := time.Now()

	// Step 1: Check semantic cache for all requests.

	var cachedResponses []*RAGResponse

	var remainingRequests []*RAGRequest

	if p.config.EnableSemanticCache {

		for _, request := range requests {

			if cached := p.semanticCache.GetSimilar(request.Query); cached != nil {

				cachedResponses = append(cachedResponses, cached.Result)

			} else {

				remainingRequests = append(remainingRequests, request)

			}

		}

	} else {

		remainingRequests = requests

	}

	// Step 2: Batch preprocess remaining queries.

	var processedQueries []*ProcessedQuery

	if p.config.EnableQueryPreprocessing && len(remainingRequests) > 0 {

		processedQueries = p.queryPreprocessor.ProcessBatch(remainingRequests)

		// Update requests with processed queries.

		for i, processed := range processedQueries {

			if i < len(remainingRequests) {

				remainingRequests[i].Query = processed.ExpandedQuery

			}

		}

	}

	// Step 3: Execute batch search.

	var freshResponses []*RAGResponse

	if len(remainingRequests) > 0 {

		responses, err := p.executeBatchSearch(ctx, remainingRequests)

		if err != nil {

			p.updatePipelineMetrics(false, time.Since(startTime), "batch_search_error")

			return nil, fmt.Errorf("batch search failed: %w", err)

		}

		freshResponses = responses

	}

	// Step 4: Combine cached and fresh responses.

	allResponses := append(cachedResponses, freshResponses...)

	// Step 5: Aggregate results if enabled.

	if p.config.EnableResultAggregation {

		for i, response := range allResponses {

			allResponses[i] = p.resultAggregator.AggregateResults(response)

		}

	}

	// Step 6: Cache fresh results.

	if p.config.EnableSemanticCache {

		for i, response := range freshResponses {

			if i < len(remainingRequests) {

				p.semanticCache.Store(remainingRequests[i].Query, response)

			}

		}

	}

	p.updatePipelineMetrics(true, time.Since(startTime), "batch_success")

	p.logger.Info("RAG batch pipeline completed",

		"total_requests", len(requests),

		"cached_responses", len(cachedResponses),

		"fresh_responses", len(freshResponses),

		"processing_time", time.Since(startTime),
	)

	return allResponses, nil

}

// executeOptimizedSearch performs optimized single query search.

func (p *OptimizedRAGPipeline) executeOptimizedSearch(ctx context.Context, request *RAGRequest) (*RAGResponse, error) {

	// Create search query.

	searchQuery := &SearchQuery{

		Query: request.Query,

		Limit: request.MaxResults,

		Filters: request.SearchFilters,

		HybridSearch: request.UseHybridSearch,

		UseReranker: request.EnableReranking,

		MinConfidence: float64(request.MinConfidence),
	}

	// Execute search.

	searchResponse, err := p.weaviateClient.Search(ctx, searchQuery)

	if err != nil {

		return nil, err

	}

	// Convert to RAG response.

	ragResponse := &RAGResponse{

		Answer: "", // Will be filled by LLM processing

		SourceDocuments: make([]*shared.SearchResult, len(searchResponse.Results)),

		Confidence: p.calculateAggregateConfidence(searchResponse.Results),

		ProcessingTime: searchResponse.Took,

		RetrievalTime: searchResponse.Took,

		Query: request.Query,

		ProcessedAt: time.Now(),
	}

	// Convert search results.

	for i, result := range searchResponse.Results {

		ragResponse.SourceDocuments[i] = &shared.SearchResult{

			Document: result.Document,

			Score: result.Score,
		}

	}

	return ragResponse, nil

}

// executeBatchSearch performs optimized batch search.

func (p *OptimizedRAGPipeline) executeBatchSearch(ctx context.Context, requests []*RAGRequest) ([]*RAGResponse, error) {

	// Convert to search queries.

	searchQueries := make([]*SearchQuery, len(requests))

	for i, request := range requests {

		searchQueries[i] = &SearchQuery{

			Query: request.Query,

			Limit: request.MaxResults,

			Filters: request.SearchFilters,

			HybridSearch: request.UseHybridSearch,

			UseReranker: request.EnableReranking,

			MinConfidence: float64(request.MinConfidence),
		}

	}

	// Execute batch search.

	batchRequest := &BatchSearchRequest{

		Queries: searchQueries,

		MaxConcurrency: p.config.MaxConcurrency,

		EnableAggregation: p.config.EnableResultAggregation,
	}

	batchResponse, err := p.batchSearchClient.BatchSearch(ctx, batchRequest)

	if err != nil {

		return nil, err

	}

	// Convert to RAG responses.

	ragResponses := make([]*RAGResponse, len(batchResponse.Results))

	for i, searchResponse := range batchResponse.Results {

		ragResponses[i] = &RAGResponse{

			Answer: "",

			SourceDocuments: make([]*shared.SearchResult, len(searchResponse.Results)),

			Confidence: p.calculateAggregateConfidence(searchResponse.Results),

			ProcessingTime: searchResponse.Took,

			RetrievalTime: searchResponse.Took,

			Query: requests[i].Query,

			ProcessedAt: time.Now(),
		}

		// Convert search results.

		for j, result := range searchResponse.Results {

			ragResponses[i].SourceDocuments[j] = &shared.SearchResult{

				Document: result.Document,

				Score: result.Score,
			}

		}

	}

	return ragResponses, nil

}

// Semantic Cache Implementation.

// GetSimilar retrieves semantically similar cached results.

func (c *SemanticCache) GetSimilar(query string) *SemanticCacheEntry {

	if !c.config.EnableSemanticCache {

		return nil

	}

	c.mutex.RLock()

	defer c.mutex.RUnlock()

	c.metrics.mutex.Lock()

	c.metrics.TotalQueries++

	c.metrics.mutex.Unlock()

	// First try exact match.

	if entry, exists := c.entries[query]; exists {

		c.updateCacheHit(entry, true)

		return entry

	}

	// Then try semantic similarity.

	queryVector := c.generateQueryVector(query) // Placeholder implementation

	bestMatch := c.findBestSemanticMatch(query, queryVector)

	if bestMatch != nil {

		c.updateCacheHit(bestMatch, false)

		return bestMatch

	}

	c.metrics.mutex.Lock()

	c.metrics.Misses++

	c.metrics.mutex.Unlock()

	return nil

}

// Store caches a result with semantic awareness.

func (c *SemanticCache) Store(query string, result *RAGResponse) {

	if !c.config.EnableSemanticCache {

		return

	}

	c.mutex.Lock()

	defer c.mutex.Unlock()

	// Generate semantic hash and vector.

	queryVector := c.generateQueryVector(query)

	semanticHash := c.generateSemanticHash(query, queryVector)

	entry := &SemanticCacheEntry{

		Query: query,

		QueryVector: queryVector,

		Result: result,

		CreatedAt: time.Now(),

		LastAccessed: time.Now(),

		AccessCount: 0,

		SemanticHash: semanticHash,

		OriginalQueries: []string{query},
	}

	c.entries[query] = entry

	c.vectors[semanticHash] = queryVector

	// Evict old entries if necessary.

	if len(c.entries) > c.config.SemanticCacheSize {

		c.evictOldest()

	}

}

// Helper methods for semantic cache.

func (c *SemanticCache) generateQueryVector(query string) []float32 {

	// Placeholder implementation - would use actual embedding model.

	vector := make([]float32, 384) // Example dimension

	for i := range vector {

		vector[i] = 0.1 // Placeholder values

	}

	return vector

}

func (c *SemanticCache) generateSemanticHash(query string, vector []float32) string {

	// Combine query text and vector signature for hash.

	data := fmt.Sprintf("%s_%f_%f_%f", query, vector[0], vector[len(vector)/2], vector[len(vector)-1])

	return fmt.Sprintf("%x", md5.Sum([]byte(data)))

}

func (c *SemanticCache) findBestSemanticMatch(query string, queryVector []float32) *SemanticCacheEntry {

	var bestMatch *SemanticCacheEntry

	var bestSimilarity float32

	for _, entry := range c.entries {

		if time.Since(entry.CreatedAt) > c.config.SemanticCacheTTL {

			continue // Skip expired entries

		}

		similarity := c.calculateCosineSimilarity(queryVector, entry.QueryVector)

		if similarity > bestSimilarity && similarity >= c.config.SemanticSimilarityThreshold {

			bestSimilarity = similarity

			bestMatch = entry

		}

	}

	return bestMatch

}

func (c *SemanticCache) calculateCosineSimilarity(vec1, vec2 []float32) float32 {

	if len(vec1) != len(vec2) {

		return 0

	}

	var dotProduct, norm1, norm2 float32

	for i := range vec1 {

		dotProduct += vec1[i] * vec2[i]

		norm1 += vec1[i] * vec1[i]

		norm2 += vec2[i] * vec2[i]

	}

	if norm1 == 0 || norm2 == 0 {

		return 0

	}

	return dotProduct / (float32(sqrt(float64(norm1))) * float32(sqrt(float64(norm2))))

}

func sqrt(x float64) float64 {

	// Simple square root implementation.

	if x == 0 {

		return 0

	}

	// Use Newton's method for square root approximation.

	z := x

	for i := 0; i < 10; i++ {

		z = (z + x/z) / 2

	}

	return z

}

func (c *SemanticCache) updateCacheHit(entry *SemanticCacheEntry, exact bool) {

	entry.LastAccessed = time.Now()

	entry.AccessCount++

	c.metrics.mutex.Lock()

	defer c.metrics.mutex.Unlock()

	if exact {

		c.metrics.ExactHits++

	} else {

		c.metrics.SemanticHits++

	}

}

func (c *SemanticCache) evictOldest() {

	var oldestKey string

	var oldestTime time.Time = time.Now()

	for key, entry := range c.entries {

		if entry.LastAccessed.Before(oldestTime) {

			oldestTime = entry.LastAccessed

			oldestKey = key

		}

	}

	if oldestKey != "" {

		delete(c.entries, oldestKey)

		c.metrics.mutex.Lock()

		c.metrics.CacheEvictions++

		c.metrics.mutex.Unlock()

	}

}

// Query Preprocessor Implementation.

// Process preprocesses a single query.

func (p *QueryPreprocessor) Process(query string) *ProcessedQuery {

	startTime := time.Now()

	processed := &ProcessedQuery{

		OriginalQuery: query,

		ExtractedEntities: make(map[string]string),
	}

	// Step 1: Normalize query.

	if p.config.EnableQueryNormalization {

		processed.NormalizedQuery = p.normalizeQuery(query)

	} else {

		processed.NormalizedQuery = query

	}

	// Step 2: Extract telecom entities.

	if p.config.EnableTelecomNER {

		processed.ExtractedEntities = p.extractTelecomEntities(processed.NormalizedQuery)

	}

	// Step 3: Expand query with synonyms and related terms.

	if p.config.EnableQueryExpansion {

		processed.ExpandedQuery = p.expandQuery(processed.NormalizedQuery, processed.ExtractedEntities)

		processed.Synonyms = p.findSynonyms(processed.NormalizedQuery)

		processed.TelecomTerms = p.findTelecomTerms(processed.NormalizedQuery)

	} else {

		processed.ExpandedQuery = processed.NormalizedQuery

	}

	processed.ProcessingTime = time.Since(startTime)

	p.updatePreprocessorMetrics(processed)

	return processed

}

// ProcessBatch preprocesses multiple queries efficiently.

func (p *QueryPreprocessor) ProcessBatch(requests []*RAGRequest) []*ProcessedQuery {

	results := make([]*ProcessedQuery, len(requests))

	// Use goroutines for parallel processing.

	var wg sync.WaitGroup

	for i, request := range requests {

		wg.Add(1)

		go func(index int, req *RAGRequest) {

			defer wg.Done()

			results[index] = p.Process(req.Query)

		}(i, request)

	}

	wg.Wait()

	return results

}

// Helper methods for query preprocessing.

func (p *QueryPreprocessor) normalizeQuery(query string) string {

	// Convert to lowercase.

	normalized := strings.ToLower(query)

	// Expand acronyms.

	for acronym, expansion := range p.acronymExpansions {

		normalized = strings.ReplaceAll(normalized, acronym, expansion)

	}

	// Remove stop words.

	words := strings.Fields(normalized)

	var filteredWords []string

	for _, word := range words {

		if !p.stopWords[word] {

			filteredWords = append(filteredWords, word)

		}

	}

	return strings.Join(filteredWords, " ")

}

func (p *QueryPreprocessor) extractTelecomEntities(query string) map[string]string {

	entities := make(map[string]string)

	// Use pattern matching for telecom entity recognition.

	for entityType, pattern := range p.nerPatterns {

		// Simplified pattern matching - would use regex in real implementation.

		if strings.Contains(query, pattern) {

			entities[entityType] = pattern

		}

	}

	return entities

}

func (p *QueryPreprocessor) expandQuery(query string, entities map[string]string) string {

	expanded := query

	// Add synonyms.

	synonyms := p.findSynonyms(query)

	if len(synonyms) > 0 {

		expanded += " " + strings.Join(synonyms, " ")

	}

	// Add related telecom terms.

	telecomTerms := p.findTelecomTerms(query)

	if len(telecomTerms) > 0 {

		expanded += " " + strings.Join(telecomTerms, " ")

	}

	return expanded

}

func (p *QueryPreprocessor) findSynonyms(query string) []string {

	var synonyms []string

	words := strings.Fields(query)

	for _, word := range words {

		if synList, exists := p.synonyms[word]; exists {

			synonyms = append(synonyms, synList...)

		}

	}

	return synonyms

}

func (p *QueryPreprocessor) findTelecomTerms(query string) []string {

	var terms []string

	for term, relatedTerms := range p.telecomTerms {

		if strings.Contains(query, term) {

			terms = append(terms, relatedTerms...)

		}

	}

	return terms

}

func (p *QueryPreprocessor) updatePreprocessorMetrics(processed *ProcessedQuery) {

	p.metrics.mutex.Lock()

	defer p.metrics.mutex.Unlock()

	p.metrics.TotalQueries++

	p.metrics.ProcessedQueries++

	if processed.ExpandedQuery != processed.OriginalQuery {

		p.metrics.ExpandedQueries++

	}

	if processed.NormalizedQuery != processed.OriginalQuery {

		p.metrics.NormalizedQueries++

	}

	p.metrics.ExtractedEntities += int64(len(processed.ExtractedEntities))

	// Update average processing time.

	if p.metrics.ProcessedQueries == 1 {

		p.metrics.AverageProcessingTime = processed.ProcessingTime

	} else {

		p.metrics.AverageProcessingTime = (p.metrics.AverageProcessingTime*time.Duration(p.metrics.ProcessedQueries-1) +

			processed.ProcessingTime) / time.Duration(p.metrics.ProcessedQueries)

	}

}

// Result Aggregator Implementation.

// AggregateResults performs intelligent result aggregation.

func (r *ResultAggregator) AggregateResults(response *RAGResponse) *RAGResponse {

	if !r.config.EnableResultAggregation {

		return response

	}

	startTime := time.Now()

	// Step 1: Deduplicate results if enabled.

	if r.config.EnableResultDeduplication {

		response.SourceDocuments = r.deduplicateResults(response.SourceDocuments)

	}

	// Step 2: Rank results if enabled.

	if r.config.EnableResultRanking {

		response.SourceDocuments = r.rankResults(response.SourceDocuments)

	}

	// Step 3: Limit results to maximum.

	if len(response.SourceDocuments) > r.config.MaxResultsPerQuery {

		response.SourceDocuments = response.SourceDocuments[:r.config.MaxResultsPerQuery]

	}

	r.updateAggregatorMetrics(response, time.Since(startTime))

	return response

}

func (r *ResultAggregator) deduplicateResults(results []*shared.SearchResult) []*shared.SearchResult {

	seen := make(map[string]*shared.SearchResult)

	var deduplicated []*shared.SearchResult

	for _, result := range results {

		if result.Document == nil {

			continue

		}

		key := result.Document.ID

		if existing, exists := seen[key]; exists {

			// Keep the one with higher score.

			if result.Score > existing.Score {

				seen[key] = result

			}

		} else {

			seen[key] = result

			deduplicated = append(deduplicated, result)

		}

	}

	return deduplicated

}

func (r *ResultAggregator) rankResults(results []*shared.SearchResult) []*shared.SearchResult {

	// Sort by score in descending order.

	sort.Slice(results, func(i, j int) bool {

		return results[i].Score > results[j].Score

	})

	return results

}

func (r *ResultAggregator) updateAggregatorMetrics(response *RAGResponse, duration time.Duration) {

	r.metrics.mutex.Lock()

	defer r.metrics.mutex.Unlock()

	r.metrics.TotalAggregations++

	r.metrics.RankedResults += int64(len(response.SourceDocuments))

	// Update average aggregation time.

	if r.metrics.TotalAggregations == 1 {

		r.metrics.AverageAggregationTime = duration

	} else {

		r.metrics.AverageAggregationTime = (r.metrics.AverageAggregationTime*time.Duration(r.metrics.TotalAggregations-1) +

			duration) / time.Duration(r.metrics.TotalAggregations)

	}

}

// Helper initialization functions.

func initializeTelecomTerms() map[string][]string {

	return map[string][]string{

		"5G": {"fifth generation", "nr", "new radio"},

		"AMF": {"access and mobility management function"},

		"SMF": {"session management function"},

		"UPF": {"user plane function"},

		"gNB": {"next generation node b"},
	}

}

func initializeAcronymExpansions() map[string]string {

	return map[string]string{

		"amf": "access and mobility management function",

		"smf": "session management function",

		"upf": "user plane function",

		"gnb": "next generation node b",
	}

}

func initializeSynonyms() map[string][]string {

	return map[string][]string{

		"configuration": {"config", "setup", "settings"},

		"optimization": {"tuning", "optimization", "enhancement"},

		"network": {"telecom", "telecommunications", "cellular"},
	}

}

func initializeStopWords() map[string]bool {

	return map[string]bool{

		"the": true, "a": true, "an": true, "and": true, "or": true,

		"but": true, "in": true, "on": true, "at": true, "to": true,

		"for": true, "of": true, "with": true, "by": true, "is": true,
	}

}

func initializeNERPatterns() map[string]string {

	return map[string]string{

		"network_function": "AMF|SMF|UPF|AUSF|UDM|NSSF",

		"interface": "N1|N2|N3|N4|N6|A1|O1|O2|E2",

		"technology": "5G|4G|LTE|NR|O-RAN|vRAN",
	}

}

// Utility methods.

func (p *OptimizedRAGPipeline) calculateAggregateConfidence(results []*SearchResult) float32 {

	if len(results) == 0 {

		return 0.0

	}

	var totalScore float32

	for _, result := range results {

		totalScore += result.Score

	}

	return totalScore / float32(len(results))

}

func (p *OptimizedRAGPipeline) updatePipelineMetrics(success bool, duration time.Duration, operation string) {

	p.metrics.mutex.Lock()

	defer p.metrics.mutex.Unlock()

	p.metrics.TotalPipelineRuns++

	if success {

		p.metrics.SuccessfulRuns++

	} else {

		p.metrics.FailedRuns++

	}

	// Update average latency.

	if p.metrics.TotalPipelineRuns == 1 {

		p.metrics.AverageLatency = duration

	} else {

		p.metrics.AverageLatency = (p.metrics.AverageLatency*time.Duration(p.metrics.TotalPipelineRuns-1) +

			duration) / time.Duration(p.metrics.TotalPipelineRuns)

	}

	p.logger.Debug("Pipeline metrics updated", "operation", operation, "duration", duration, "success", success)

}

func (p *OptimizedRAGPipeline) startBackgroundMaintenance() {

	ticker := time.NewTicker(time.Hour)

	defer ticker.Stop()

	for {

		select {

		case <-ticker.C:

			p.performMaintenance()

		}

	}

}

func (p *OptimizedRAGPipeline) performMaintenance() {

	// Clean up expired cache entries.

	if p.config.EnableSemanticCache {

		p.semanticCache.cleanupExpired()

	}

	// Clean up embedding cache.

	if p.config.EnableEmbeddingCache {

		// Note: cleanup is handled by the cache implementation.

	}

	p.logger.Debug("Background maintenance completed")

}

func (c *SemanticCache) cleanupExpired() {

	c.mutex.Lock()

	defer c.mutex.Unlock()

	now := time.Now()

	var keysToDelete []string

	for key, entry := range c.entries {

		if now.Sub(entry.CreatedAt) > c.config.SemanticCacheTTL {

			keysToDelete = append(keysToDelete, key)

		}

	}

	for _, key := range keysToDelete {

		delete(c.entries, key)

		c.metrics.mutex.Lock()

		c.metrics.CacheEvictions++

		c.metrics.mutex.Unlock()

	}

	c.logger.Debug("Semantic cache cleanup completed", "evicted", len(keysToDelete))

}

// Note: EmbeddingCache cleanup is handled by the cache implementation.

// GetMetrics returns all pipeline metrics.

func (p *OptimizedRAGPipeline) GetMetrics() map[string]interface{} {

	return map[string]interface{}{

		"pipeline": p.metrics,

		"semantic_cache": p.semanticCache.metrics,

		"preprocessor": p.queryPreprocessor.metrics,

		"aggregator": p.resultAggregator.metrics,

		"embedding_cache": p.embeddingCache.Stats(),
	}

}

// Close cleans up pipeline resources.

func (p *OptimizedRAGPipeline) Close() error {

	p.logger.Info("Closing optimized RAG pipeline")

	return nil

}
