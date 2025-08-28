//go:build !disable_rag && !test

package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
	"golang.org/x/sync/errgroup"
)

// BatchSearchRequest represents a batch of search requests
type BatchSearchRequest struct {
	Queries           []*SearchQuery         `json:"queries"`
	MaxConcurrency    int                    `json:"max_concurrency"`
	EnableAggregation bool                   `json:"enable_aggregation"`
	DeduplicationKey  string                 `json:"deduplication_key"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// BatchSearchResponse represents the response from batch search
type BatchSearchResponse struct {
	Results             []*SearchResponse      `json:"results"`
	AggregatedResults   []*shared.SearchResult `json:"aggregated_results"`
	TotalProcessingTime time.Duration          `json:"total_processing_time"`
	ParallelQueries     int                    `json:"parallel_queries"`
	CacheHits           int                    `json:"cache_hits"`
	Metadata            map[string]interface{} `json:"metadata"`
}

// SearchBatch represents a batch of related queries
type SearchBatch struct {
	ID          string         `json:"id"`
	Queries     []*SearchQuery `json:"queries"`
	Priority    int            `json:"priority"`
	CreatedAt   time.Time      `json:"created_at"`
	MaxWaitTime time.Duration  `json:"max_wait_time"`
}

// BatchOptimizer handles batching optimization
type BatchOptimizer struct {
	maxBatchSize        int
	maxWaitTime         time.Duration
	similarityThreshold float32
	batchQueue          chan *SearchBatch
	processingQueue     chan *BatchSearchRequest
	results             map[string]chan *BatchSearchResponse
	mutex               sync.RWMutex
	logger              *slog.Logger
}

// OptimizedBatchSearchClient provides high-performance batch searching
type OptimizedBatchSearchClient struct {
	client      *WeaviateClient
	optimizer   *BatchOptimizer
	queryCache  *QueryResultCache
	config      *BatchSearchConfig
	logger      *slog.Logger
	metrics     *BatchSearchMetrics
	vectorCache sync.Map  // Cache for computed vectors
	queryPool   sync.Pool // Pool for query objects
}

// BatchSearchConfig holds configuration for batch searching
type BatchSearchConfig struct {
	MaxBatchSize            int           `json:"max_batch_size"`
	MaxConcurrency          int           `json:"max_concurrency"`
	MaxWaitTime             time.Duration `json:"max_wait_time"`
	EnableVectorCaching     bool          `json:"enable_vector_caching"`
	EnableQueryOptimization bool          `json:"enable_query_optimization"`
	EnableDeduplication     bool          `json:"enable_deduplication"`
	SimilarityThreshold     float32       `json:"similarity_threshold"`
	CacheSize               int           `json:"cache_size"`
	CacheTTL                time.Duration `json:"cache_ttl"`
}

// BatchSearchMetrics tracks batch search performance
type BatchSearchMetrics struct {
	TotalBatches         int64         `json:"total_batches"`
	TotalQueries         int64         `json:"total_queries"`
	AverageLatency       time.Duration `json:"average_latency"`
	ThroughputQPS        float64       `json:"throughput_qps"`
	CacheHitRate         float64       `json:"cache_hit_rate"`
	ParallelizationRatio float64       `json:"parallelization_ratio"`
	DeduplicationSavings float64       `json:"deduplication_savings"`
	mutex                sync.RWMutex
}

// QueryResultCache provides intelligent caching for batch queries
type QueryResultCache struct {
	data     sync.Map
	ttl      time.Duration
	maxSize  int
	eviction chan string
	logger   *slog.Logger
}

// CachedQueryResult represents a cached query result
type CachedQueryResult struct {
	Result      *SearchResponse `json:"result"`
	CreatedAt   time.Time       `json:"created_at"`
	AccessCount int64           `json:"access_count"`
	Vector      []float32       `json:"vector"`
}

// NewOptimizedBatchSearchClient creates a new optimized batch search client
func NewOptimizedBatchSearchClient(weaviateClient *WeaviateClient, config *BatchSearchConfig) *OptimizedBatchSearchClient {
	if config == nil {
		config = getDefaultBatchSearchConfig()
	}

	logger := slog.Default().With("component", "optimized-batch-search")

	cache := &QueryResultCache{
		ttl:      config.CacheTTL,
		maxSize:  config.CacheSize,
		eviction: make(chan string, 1000),
		logger:   logger,
	}

	optimizer := &BatchOptimizer{
		maxBatchSize:        config.MaxBatchSize,
		maxWaitTime:         config.MaxWaitTime,
		similarityThreshold: config.SimilarityThreshold,
		batchQueue:          make(chan *SearchBatch, 100),
		processingQueue:     make(chan *BatchSearchRequest, 50),
		results:             make(map[string]chan *BatchSearchResponse),
		logger:              logger,
	}

	client := &OptimizedBatchSearchClient{
		client:     weaviateClient,
		optimizer:  optimizer,
		queryCache: cache,
		config:     config,
		logger:     logger,
		metrics:    &BatchSearchMetrics{},
	}

	// Initialize query pool for object reuse
	client.queryPool.New = func() interface{} {
		return &SearchQuery{}
	}

	// Start background processing
	go client.startBatchProcessor()
	go client.startCacheEviction()

	return client
}

// getDefaultBatchSearchConfig returns default batch search configuration
func getDefaultBatchSearchConfig() *BatchSearchConfig {
	return &BatchSearchConfig{
		MaxBatchSize:            50,
		MaxConcurrency:          10,
		MaxWaitTime:             100 * time.Millisecond,
		EnableVectorCaching:     true,
		EnableQueryOptimization: true,
		EnableDeduplication:     true,
		SimilarityThreshold:     0.95,
		CacheSize:               1000,
		CacheTTL:                5 * time.Minute,
	}
}

// BatchSearch performs optimized batch searching
func (c *OptimizedBatchSearchClient) BatchSearch(ctx context.Context, request *BatchSearchRequest) (*BatchSearchResponse, error) {
	startTime := time.Now()

	if request == nil || len(request.Queries) == 0 {
		return nil, fmt.Errorf("batch search request cannot be nil or empty")
	}

	// Set defaults
	if request.MaxConcurrency == 0 {
		request.MaxConcurrency = c.config.MaxConcurrency
	}

	c.logger.Info("Starting batch search",
		"num_queries", len(request.Queries),
		"max_concurrency", request.MaxConcurrency,
	)

	// Step 1: Deduplicate and optimize queries
	optimizedQueries := c.optimizeQueries(request.Queries)

	// Step 2: Check cache for existing results
	cachedResults, remainingQueries := c.checkCache(ctx, optimizedQueries)

	// Step 3: Execute remaining queries in parallel
	freshResults, err := c.executeParallelSearch(ctx, remainingQueries, request.MaxConcurrency)
	if err != nil {
		return nil, fmt.Errorf("parallel search failed: %w", err)
	}

	// Step 4: Combine cached and fresh results
	allResults := append(cachedResults, freshResults...)

	// Step 5: Aggregate results if requested
	var aggregatedResults []*shared.SearchResult
	if request.EnableAggregation {
		aggregatedResults = c.aggregateResults(allResults)
	}

	response := &BatchSearchResponse{
		Results:             allResults,
		AggregatedResults:   aggregatedResults,
		TotalProcessingTime: time.Since(startTime),
		ParallelQueries:     len(remainingQueries),
		CacheHits:           len(cachedResults),
		Metadata: map[string]interface{}{
			"original_queries":  len(request.Queries),
			"optimized_queries": len(optimizedQueries),
			"cache_hit_rate":    float64(len(cachedResults)) / float64(len(optimizedQueries)),
		},
	}

	// Update metrics
	c.updateMetrics(response)

	c.logger.Info("Batch search completed",
		"total_queries", len(request.Queries),
		"cache_hits", len(cachedResults),
		"fresh_queries", len(remainingQueries),
		"processing_time", response.TotalProcessingTime,
	)

	return response, nil
}

// optimizeQueries deduplicates and optimizes queries
func (c *OptimizedBatchSearchClient) optimizeQueries(queries []*SearchQuery) []*SearchQuery {
	if !c.config.EnableQueryOptimization {
		return queries
	}

	seen := make(map[string]*SearchQuery)
	optimized := make([]*SearchQuery, 0, len(queries))

	for _, query := range queries {
		// Create deduplication key
		key := c.createQueryKey(query)

		if existing, exists := seen[key]; exists {
			// Merge with existing query if similar
			if c.areSimilarQueries(query, existing) {
				continue
			}
		}

		seen[key] = query
		optimized = append(optimized, query)
	}

	return optimized
}

// checkCache checks for cached results
func (c *OptimizedBatchSearchClient) checkCache(ctx context.Context, queries []*SearchQuery) ([]*SearchResponse, []*SearchQuery) {
	if !c.config.EnableVectorCaching {
		return nil, queries
	}

	var cachedResults []*SearchResponse
	var remainingQueries []*SearchQuery

	for _, query := range queries {
		key := c.createQueryKey(query)

		if cached, found := c.queryCache.data.Load(key); found {
			if cachedResult, ok := cached.(*CachedQueryResult); ok {
				// Check if still valid
				if time.Since(cachedResult.CreatedAt) < c.queryCache.ttl {
					cachedResults = append(cachedResults, cachedResult.Result)
					cachedResult.AccessCount++
					continue
				}
				// Remove expired entry
				c.queryCache.data.Delete(key)
			}
		}

		remainingQueries = append(remainingQueries, query)
	}

	return cachedResults, remainingQueries
}

// executeParallelSearch executes queries in parallel
func (c *OptimizedBatchSearchClient) executeParallelSearch(ctx context.Context, queries []*SearchQuery, maxConcurrency int) ([]*SearchResponse, error) {
	if len(queries) == 0 {
		return nil, nil
	}

	results := make([]*SearchResponse, len(queries))

	// Create error group with concurrency limit
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrency)

	// Process queries in parallel
	for i, query := range queries {
		i, query := i, query // Capture loop variables

		g.Go(func() error {
			result, err := c.client.Search(ctx, query)
			if err != nil {
				return fmt.Errorf("search failed for query %d: %w", i, err)
			}

			results[i] = result

			// Cache the result
			if c.config.EnableVectorCaching {
				key := c.createQueryKey(query)
				cachedResult := &CachedQueryResult{
					Result:      result,
					CreatedAt:   time.Now(),
					AccessCount: 1,
				}
				c.queryCache.data.Store(key, cachedResult)
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}

// aggregateResults aggregates and deduplicates search results
func (c *OptimizedBatchSearchClient) aggregateResults(responses []*SearchResponse) []*shared.SearchResult {
	seen := make(map[string]*shared.SearchResult)
	var aggregated []*shared.SearchResult

	for _, response := range responses {
		for _, result := range response.Results {
			if result.Document == nil {
				continue
			}

			key := result.Document.ID
			if existing, exists := seen[key]; exists {
				// Combine scores using max
				if result.Score > existing.Score {
					existing.Score = result.Score
				}
			} else {
				// Convert local SearchResult to shared.SearchResult
				sharedResult := &shared.SearchResult{
					Document: result.Document, // Assuming Document can be directly assigned
					Score:    result.Score,
				}
				seen[key] = sharedResult
				aggregated = append(aggregated, sharedResult)
			}
		}
	}

	return aggregated
}

// ParallelMultiQuery executes multiple different query types in parallel
func (c *OptimizedBatchSearchClient) ParallelMultiQuery(ctx context.Context, queries map[string]*SearchQuery) (map[string]*SearchResponse, error) {
	results := make(map[string]*SearchResponse)
	resultsMutex := sync.Mutex{}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(c.config.MaxConcurrency)

	for queryType, query := range queries {
		queryType, query := queryType, query // Capture loop variables

		g.Go(func() error {
			result, err := c.client.Search(ctx, query)
			if err != nil {
				return fmt.Errorf("search failed for query type %s: %w", queryType, err)
			}

			resultsMutex.Lock()
			results[queryType] = result
			resultsMutex.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}

// StreamingBatchSearch provides streaming results for large batches
func (c *OptimizedBatchSearchClient) StreamingBatchSearch(ctx context.Context, queries []*SearchQuery, resultChan chan<- *SearchResponse) error {
	defer close(resultChan)

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(c.config.MaxConcurrency)

	for _, query := range queries {
		query := query // Capture loop variable

		g.Go(func() error {
			result, err := c.client.Search(ctx, query)
			if err != nil {
				c.logger.Error("streaming search failed", "error", err, "query", query.Query)
				return nil // Don't fail the entire batch for one query
			}

			select {
			case resultChan <- result:
			case <-ctx.Done():
				return ctx.Err()
			}

			return nil
		})
	}

	return g.Wait()
}

// Helper methods

func (c *OptimizedBatchSearchClient) createQueryKey(query *SearchQuery) string {
	key := fmt.Sprintf("%s|%d|%t|%.2f|%t",
		query.Query, query.Limit, query.HybridSearch,
		query.HybridAlpha, query.UseReranker)

	// Add filters to key
	if len(query.Filters) > 0 {
		if filtersJSON, err := json.Marshal(query.Filters); err == nil {
			key += "|" + string(filtersJSON)
		}
	}

	return key
}

func (c *OptimizedBatchSearchClient) areSimilarQueries(q1, q2 *SearchQuery) bool {
	// Simple similarity check - can be enhanced with semantic similarity
	return q1.Query == q2.Query &&
		q1.HybridSearch == q2.HybridSearch &&
		q1.UseReranker == q2.UseReranker
}

func (c *OptimizedBatchSearchClient) updateMetrics(response *BatchSearchResponse) {
	c.metrics.mutex.Lock()
	defer c.metrics.mutex.Unlock()

	c.metrics.TotalBatches++
	c.metrics.TotalQueries += int64(len(response.Results))

	// Update average latency
	if c.metrics.TotalBatches == 1 {
		c.metrics.AverageLatency = response.TotalProcessingTime
	} else {
		c.metrics.AverageLatency = (c.metrics.AverageLatency*time.Duration(c.metrics.TotalBatches-1) +
			response.TotalProcessingTime) / time.Duration(c.metrics.TotalBatches)
	}

	// Update throughput (queries per second)
	c.metrics.ThroughputQPS = float64(c.metrics.TotalQueries) /
		(float64(c.metrics.AverageLatency.Nanoseconds()) / 1e9)

	// Update cache hit rate
	if len(response.Results) > 0 {
		c.metrics.CacheHitRate = float64(response.CacheHits) / float64(len(response.Results))
	}
}

func (c *OptimizedBatchSearchClient) startBatchProcessor() {
	// Background batch processing logic would go here
	// This would handle queuing and batching of requests
}

func (c *OptimizedBatchSearchClient) startCacheEviction() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.evictExpiredCacheEntries()
		case key := <-c.queryCache.eviction:
			c.queryCache.data.Delete(key)
		}
	}
}

func (c *OptimizedBatchSearchClient) evictExpiredCacheEntries() {
	now := time.Now()
	var toEvict []string

	c.queryCache.data.Range(func(key, value interface{}) bool {
		if cached, ok := value.(*CachedQueryResult); ok {
			if now.Sub(cached.CreatedAt) > c.queryCache.ttl {
				toEvict = append(toEvict, key.(string))
			}
		}
		return true
	})

	for _, key := range toEvict {
		c.queryCache.data.Delete(key)
	}

	if len(toEvict) > 0 {
		c.logger.Debug("Evicted expired cache entries", "count", len(toEvict))
	}
}

// GetMetrics returns current batch search metrics
func (c *OptimizedBatchSearchClient) GetMetrics() *BatchSearchMetrics {
	c.metrics.mutex.RLock()
	defer c.metrics.mutex.RUnlock()

	// Return a copy without the mutex
	metrics := &BatchSearchMetrics{
		TotalBatches:         c.metrics.TotalBatches,
		TotalQueries:         c.metrics.TotalQueries,
		AverageLatency:       c.metrics.AverageLatency,
		ThroughputQPS:        c.metrics.ThroughputQPS,
		CacheHitRate:         c.metrics.CacheHitRate,
		ParallelizationRatio: c.metrics.ParallelizationRatio,
		DeduplicationSavings: c.metrics.DeduplicationSavings,
	}
	return metrics
}

// Close cleans up resources
func (c *OptimizedBatchSearchClient) Close() error {
	c.logger.Info("Closing optimized batch search client")
	// Cleanup resources
	return nil
}
