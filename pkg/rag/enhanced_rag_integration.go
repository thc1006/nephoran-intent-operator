//go:build !disable_rag




package rag



import (

	"context"

	"encoding/json"

	"fmt"

	"log"

	"log/slog"

	"math/rand"

	"net/http"

	"sort"

	"sync"

	"time"



	"github.com/nephio-project/nephoran-intent-operator/pkg/shared"

)



// Enhanced RAG Integration with Multiple Providers.

// This file provides a comprehensive RAG implementation with support for multiple vector stores,.

// embedding providers, and optimization strategies.



// RAGProvider defines the interface for RAG providers.

type RAGProvider interface {

	// Search performs semantic search.

	Search(ctx context.Context, query string, options *QueryOptions) (*QueryResponse, error)



	// Index stores documents for retrieval.

	Index(ctx context.Context, documents []*shared.TelecomDocument) error



	// GetHealth returns provider health status.

	GetHealth() ProviderHealth



	// GetMetrics returns provider performance metrics.

	GetMetrics() ProviderMetrics

}



// EmbeddingProvider defines the interface for embedding generation.

type EmbeddingProvider interface {

	// GenerateEmbedding creates embeddings for text.

	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)



	// GenerateBatchEmbeddings creates embeddings for multiple texts.

	GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error)



	// GetDimensions returns the embedding dimension size.

	GetDimensions() int



	// IsHealthy returns the health status of the provider (for health monitoring compatibility).

	IsHealthy() bool



	// GetLatency returns the average latency of the provider (for health monitoring compatibility).

	GetLatency() time.Duration

}



// VectorStore defines the interface for vector storage.

type VectorStore interface {

	// Store saves vectors with metadata.

	Store(ctx context.Context, vectors []Vector) error



	// Search finds similar vectors.

	Search(ctx context.Context, query []float32, limit int) ([]VectorSearchResult, error)



	// Delete removes vectors by ID.

	Delete(ctx context.Context, ids []string) error



	// GetStats returns storage statistics.

	GetStats() VectorStoreStats

}



// Core Types.



// Vector represents a document with its embedding.

type Vector struct {

	ID        string                  `json:"id"`

	Embedding []float32               `json:"embedding"`

	Metadata  map[string]interface{}  `json:"metadata"`

	Document  *shared.TelecomDocument `json:"document,omitempty"`

}



// VectorSearchResult represents a search result with similarity score.

type VectorSearchResult struct {

	Vector *Vector `json:"vector"`

	Score  float32 `json:"score"`

}



// VectorStoreStats provides storage statistics.

type VectorStoreStats struct {

	TotalVectors int64                  `json:"total_vectors"`

	IndexSize    int64                  `json:"index_size"`

	Metadata     map[string]interface{} `json:"metadata,omitempty"`

}



// Provider Health and Metrics.



// ProviderHealth represents the health status of a provider.

type ProviderHealth struct {

	IsHealthy bool          `json:"is_healthy"`

	LastCheck time.Time     `json:"last_check"`

	Latency   time.Duration `json:"latency"`

	ErrorRate float64       `json:"error_rate"`

	Details   string        `json:"details,omitempty"`

}



// ProviderMetrics contains performance metrics for a provider.

type ProviderMetrics struct {

	RequestCount    int64         `json:"request_count"`

	AverageLatency  time.Duration `json:"average_latency"`

	ErrorCount      int64         `json:"error_count"`

	CacheHitRate    float64       `json:"cache_hit_rate"`

	LastRequestTime time.Time     `json:"last_request_time"`

}



// Enhanced RAG Manager.



// EnhancedRAGManager manages multiple RAG providers with intelligent routing.

type EnhancedRAGManager struct {

	providers          map[string]RAGProvider

	embeddingProviders map[string]EmbeddingProvider

	vectorStores       map[string]VectorStore

	router             *ProviderRouter

	cache              *RAGCache

	metrics            *RAGMetrics

	config             *EnhancedRAGConfig

	mu                 sync.RWMutex

	logger             *slog.Logger

}



// EnhancedRAGConfig holds configuration for the RAG manager.

type EnhancedRAGConfig struct {

	// Provider Selection Strategy.

	ProviderStrategy string `json:"provider_strategy"` // "round_robin", "failover", "performance_based"

	DefaultProvider  string `json:"default_provider"`



	// Caching Configuration.

	CacheEnabled bool          `json:"cache_enabled"`

	CacheTTL     time.Duration `json:"cache_ttl"`

	CacheMaxSize int           `json:"cache_max_size"`



	// Performance Settings.

	MaxConcurrentQueries int           `json:"max_concurrent_queries"`

	QueryTimeout         time.Duration `json:"query_timeout"`



	// Retry Configuration.

	MaxRetries int           `json:"max_retries"`

	RetryDelay time.Duration `json:"retry_delay"`



	// Health Check Settings.

	HealthCheckInterval time.Duration `json:"health_check_interval"`

}



// NewEnhancedRAGManager creates a new enhanced RAG manager.

func NewEnhancedRAGManager(config *EnhancedRAGConfig) *EnhancedRAGManager {

	if config == nil {

		config = &EnhancedRAGConfig{

			ProviderStrategy:     "performance_based",

			CacheEnabled:         true,

			CacheTTL:             5 * time.Minute,

			CacheMaxSize:         1000,

			MaxConcurrentQueries: 10,

			QueryTimeout:         30 * time.Second,

			MaxRetries:           3,

			RetryDelay:           time.Second,

			HealthCheckInterval:  time.Minute,

		}

	}



	manager := &EnhancedRAGManager{

		providers:          make(map[string]RAGProvider),

		embeddingProviders: make(map[string]EmbeddingProvider),

		vectorStores:       make(map[string]VectorStore),

		config:             config,

		logger:             slog.Default().With("component", "enhanced-rag-manager"),

	}



	// Initialize components.

	manager.router = NewProviderRouter(config)

	manager.cache = NewRAGCache(config.CacheMaxSize, config.CacheTTL)

	manager.metrics = NewRAGMetrics()



	// Start health checking.

	go manager.startHealthChecking()



	return manager

}



// RegisterProvider registers a new RAG provider.

func (erm *EnhancedRAGManager) RegisterProvider(name string, provider RAGProvider) {

	erm.mu.Lock()

	defer erm.mu.Unlock()



	erm.providers[name] = provider

	erm.router.AddProvider(name, provider)



	erm.logger.Info("Registered RAG provider", "provider", name)

}



// RegisterEmbeddingProvider registers a new embedding provider.

func (erm *EnhancedRAGManager) RegisterEmbeddingProvider(name string, provider EmbeddingProvider) {

	erm.mu.Lock()

	defer erm.mu.Unlock()



	erm.embeddingProviders[name] = provider

	erm.logger.Info("Registered embedding provider", "provider", name)

}



// RegisterVectorStore registers a new vector store.

func (erm *EnhancedRAGManager) RegisterVectorStore(name string, store VectorStore) {

	erm.mu.Lock()

	defer erm.mu.Unlock()



	erm.vectorStores[name] = store

	erm.logger.Info("Registered vector store", "store", name)

}



// Query performs an enhanced RAG query with provider selection and caching.

func (erm *EnhancedRAGManager) Query(ctx context.Context, query string, options *QueryOptions) (*QueryResponse, error) {

	startTime := time.Now()



	// Check cache first.

	if erm.config.CacheEnabled {

		if cached := erm.cache.Get(query); cached != nil {

			erm.metrics.RecordCacheHit()

			return cached, nil

		}

	}



	// Select optimal provider.

	provider := erm.router.SelectProvider(ctx, query)

	if provider == nil {

		return nil, fmt.Errorf("no healthy providers available")

	}



	// Execute query with retries.

	var response *QueryResponse

	var err error



	for attempt := 0; attempt <= erm.config.MaxRetries; attempt++ {

		queryCtx, cancel := context.WithTimeout(ctx, erm.config.QueryTimeout)

		response, err = provider.Search(queryCtx, query, options)

		cancel()



		if err == nil {

			break

		}



		if attempt < erm.config.MaxRetries {

			erm.logger.Warn("Query attempt failed, retrying",

				"attempt", attempt+1,

				"error", err,

				"delay", erm.config.RetryDelay)



			select {

			case <-ctx.Done():

				return nil, ctx.Err()

			case <-time.After(erm.config.RetryDelay):

				// Continue to next attempt.

			}

		}

	}



	if err != nil {

		erm.metrics.RecordError()

		return nil, fmt.Errorf("query failed after %d attempts: %w", erm.config.MaxRetries+1, err)

	}



	// Update metrics.

	duration := time.Since(startTime)

	erm.metrics.RecordQuery(duration)



	// Cache successful response.

	if erm.config.CacheEnabled && response != nil {

		erm.cache.Set(query, response)

	}



	response.ProcessingTime = duration

	return response, nil

}



// IndexDocuments indexes documents across all registered vector stores.

func (erm *EnhancedRAGManager) IndexDocuments(ctx context.Context, documents []*shared.TelecomDocument) error {

	if len(documents) == 0 {

		return nil

	}



	erm.mu.RLock()

	providers := make([]RAGProvider, 0, len(erm.providers))

	for _, provider := range erm.providers {

		providers = append(providers, provider)

	}

	erm.mu.RUnlock()



	// Index in parallel across all providers.

	errCh := make(chan error, len(providers))



	for _, provider := range providers {

		go func(p RAGProvider) {

			err := p.Index(ctx, documents)

			errCh <- err

		}(provider)

	}



	var errors []error

	for range len(providers) {

		if err := <-errCh; err != nil {

			errors = append(errors, err)

		}

	}



	if len(errors) > 0 {

		return fmt.Errorf("indexing failed in %d providers: %v", len(errors), errors)

	}



	erm.logger.Info("Indexed documents successfully", "count", len(documents))

	return nil

}



// GetHealth returns the health status of all providers.

func (erm *EnhancedRAGManager) GetHealth() map[string]ProviderHealth {

	erm.mu.RLock()

	defer erm.mu.RUnlock()



	health := make(map[string]ProviderHealth)

	for name, provider := range erm.providers {

		health[name] = provider.GetHealth()

	}



	return health

}



// GetMetrics returns aggregated metrics for all providers.

func (erm *EnhancedRAGManager) GetMetrics() map[string]ProviderMetrics {

	erm.mu.RLock()

	defer erm.mu.RUnlock()



	metrics := make(map[string]ProviderMetrics)

	for name, provider := range erm.providers {

		metrics[name] = provider.GetMetrics()

	}



	return metrics

}



// startHealthChecking runs periodic health checks.

func (erm *EnhancedRAGManager) startHealthChecking() {

	ticker := time.NewTicker(erm.config.HealthCheckInterval)

	defer ticker.Stop()



	for {

		select {

		case <-ticker.C:

			erm.checkProviderHealth()

		}

	}

}



// checkProviderHealth checks the health of all providers.

func (erm *EnhancedRAGManager) checkProviderHealth() {

	erm.mu.RLock()

	providers := make(map[string]RAGProvider)

	for name, provider := range erm.providers {

		providers[name] = provider

	}

	erm.mu.RUnlock()



	for name, provider := range providers {

		health := provider.GetHealth()

		if !health.IsHealthy {

			erm.logger.Warn("Provider health check failed",

				"provider", name,

				"details", health.Details)

		}

	}

}



// Provider Router for intelligent provider selection.



// ProviderRouter handles intelligent routing to optimal providers.

type ProviderRouter struct {

	providers  map[string]RAGProvider

	strategy   string

	roundRobin int

	mu         sync.RWMutex

}



// NewProviderRouter creates a new provider router.

func NewProviderRouter(config *EnhancedRAGConfig) *ProviderRouter {

	return &ProviderRouter{

		providers: make(map[string]RAGProvider),

		strategy:  config.ProviderStrategy,

	}

}



// AddProvider adds a provider to the router.

func (pr *ProviderRouter) AddProvider(name string, provider RAGProvider) {

	pr.mu.Lock()

	defer pr.mu.Unlock()

	pr.providers[name] = provider

}



// SelectProvider selects the optimal provider based on strategy.

func (pr *ProviderRouter) SelectProvider(ctx context.Context, query string) RAGProvider {

	pr.mu.RLock()

	defer pr.mu.RUnlock()



	if len(pr.providers) == 0 {

		return nil

	}



	switch pr.strategy {

	case "round_robin":

		return pr.selectRoundRobin()

	case "performance_based":

		return pr.selectByPerformance()

	case "failover":

		return pr.selectFailover()

	default:

		return pr.selectFailover()

	}

}



func (pr *ProviderRouter) selectRoundRobin() RAGProvider {

	providers := make([]RAGProvider, 0, len(pr.providers))

	for _, provider := range pr.providers {

		providers = append(providers, provider)

	}



	if len(providers) == 0 {

		return nil

	}



	provider := providers[pr.roundRobin%len(providers)]

	pr.roundRobin++

	return provider

}



func (pr *ProviderRouter) selectByPerformance() RAGProvider {

	type providerScore struct {

		provider RAGProvider

		score    float64

	}



	var scores []providerScore

	for _, provider := range pr.providers {

		health := provider.GetHealth()

		metrics := provider.GetMetrics()



		if !health.IsHealthy {

			continue

		}



		// Score based on latency and error rate.

		score := 1.0 / (1.0 + float64(health.Latency.Milliseconds()))

		score *= (1.0 - health.ErrorRate)

		score *= metrics.CacheHitRate



		scores = append(scores, providerScore{provider, score})

	}



	if len(scores) == 0 {

		return nil

	}



	// Sort by score (descending).

	sort.Slice(scores, func(i, j int) bool {

		return scores[i].score > scores[j].score

	})



	return scores[0].provider

}



func (pr *ProviderRouter) selectFailover() RAGProvider {

	for _, provider := range pr.providers {

		health := provider.GetHealth()

		if health.IsHealthy {

			return provider

		}

	}

	return nil

}



// RAG Cache for query result caching.



// RAGCache provides caching for query results.

type RAGCache struct {

	cache   map[string]*enhancedCacheEntry

	maxSize int

	ttl     time.Duration

	mu      sync.RWMutex

}



type enhancedCacheEntry struct {

	response  *QueryResponse

	timestamp time.Time

}



// NewRAGCache creates a new RAG cache.

func NewRAGCache(maxSize int, ttl time.Duration) *RAGCache {

	cache := &RAGCache{

		cache:   make(map[string]*enhancedCacheEntry),

		maxSize: maxSize,

		ttl:     ttl,

	}



	// Start cleanup routine.

	go cache.startCleanup()



	return cache

}



// Get retrieves a cached response.

func (rc *RAGCache) Get(query string) *QueryResponse {

	rc.mu.RLock()

	defer rc.mu.RUnlock()



	entry, exists := rc.cache[query]

	if !exists {

		return nil

	}



	if time.Since(entry.timestamp) > rc.ttl {

		return nil

	}



	return entry.response

}



// Set stores a response in cache.

func (rc *RAGCache) Set(query string, response *QueryResponse) {

	rc.mu.Lock()

	defer rc.mu.Unlock()



	// Evict if at capacity.

	if len(rc.cache) >= rc.maxSize {

		rc.evictOldest()

	}



	rc.cache[query] = &enhancedCacheEntry{

		response:  response,

		timestamp: time.Now(),

	}

}



func (rc *RAGCache) evictOldest() {

	var oldestKey string

	var oldestTime time.Time

	first := true



	for key, entry := range rc.cache {

		if first || entry.timestamp.Before(oldestTime) {

			oldestKey = key

			oldestTime = entry.timestamp

			first = false

		}

	}



	if oldestKey != "" {

		delete(rc.cache, oldestKey)

	}

}



func (rc *RAGCache) startCleanup() {

	ticker := time.NewTicker(rc.ttl / 2) // Cleanup twice per TTL

	defer ticker.Stop()



	for {

		select {

		case <-ticker.C:

			rc.cleanup()

		}

	}

}



func (rc *RAGCache) cleanup() {

	rc.mu.Lock()

	defer rc.mu.Unlock()



	now := time.Now()

	for query, entry := range rc.cache {

		if now.Sub(entry.timestamp) > rc.ttl {

			delete(rc.cache, query)

		}

	}

}



// RAG Metrics for performance tracking.



// RAGMetrics tracks performance metrics.

type RAGMetrics struct {

	queryCount   int64

	errorCount   int64

	cacheHits    int64

	totalLatency time.Duration

	mu           sync.RWMutex

}



// NewRAGMetrics creates a new metrics tracker.

func NewRAGMetrics() *RAGMetrics {

	return &RAGMetrics{}

}



// RecordQuery records a successful query.

func (rm *RAGMetrics) RecordQuery(duration time.Duration) {

	rm.mu.Lock()

	defer rm.mu.Unlock()



	rm.queryCount++

	rm.totalLatency += duration

}



// RecordError records a query error.

func (rm *RAGMetrics) RecordError() {

	rm.mu.Lock()

	defer rm.mu.Unlock()



	rm.errorCount++

}



// RecordCacheHit records a cache hit.

func (rm *RAGMetrics) RecordCacheHit() {

	rm.mu.Lock()

	defer rm.mu.Unlock()



	rm.cacheHits++

}



// GetStats returns current metrics.

func (rm *RAGMetrics) GetStats() RAGStats {

	rm.mu.RLock()

	defer rm.mu.RUnlock()



	var avgLatency time.Duration

	if rm.queryCount > 0 {

		avgLatency = rm.totalLatency / time.Duration(rm.queryCount)

	}



	var errorRate float64

	totalRequests := rm.queryCount + rm.errorCount

	if totalRequests > 0 {

		errorRate = float64(rm.errorCount) / float64(totalRequests)

	}



	var cacheHitRate float64

	if totalRequests > 0 {

		cacheHitRate = float64(rm.cacheHits) / float64(totalRequests)

	}



	return RAGStats{

		QueryCount:     rm.queryCount,

		ErrorCount:     rm.errorCount,

		CacheHits:      rm.cacheHits,

		AverageLatency: avgLatency,

		ErrorRate:      errorRate,

		CacheHitRate:   cacheHitRate,

	}

}



// RAGStats contains aggregated statistics.

type RAGStats struct {

	QueryCount     int64         `json:"query_count"`

	ErrorCount     int64         `json:"error_count"`

	CacheHits      int64         `json:"cache_hits"`

	AverageLatency time.Duration `json:"average_latency"`

	ErrorRate      float64       `json:"error_rate"`

	CacheHitRate   float64       `json:"cache_hit_rate"`

}



// Type aliases for shared types.

type SearchResult = shared.SearchResult



// QueryOptions holds options for RAG queries.

type QueryOptions struct {

	TopK            int                    `json:"top_k"`

	ScoreThreshold  float32                `json:"score_threshold"`

	Filters         map[string]interface{} `json:"filters"`

	IncludeMetadata bool                   `json:"include_metadata"`

}



// QueryResponse represents a RAG query response.

type QueryResponse struct {

	Query          string          `json:"query"`

	Results        []*SearchResult `json:"results"`

	ProcessingTime time.Duration   `json:"processing_time"`

	EmbeddingCost  float64         `json:"embedding_cost"`

	ProviderUsed   string          `json:"provider_used"`

}



// MockVectorStore is a simple in-memory vector store for testing.

type MockVectorStore struct {

	data map[string]*vectorEntry

	mu   sync.RWMutex

}



type vectorEntry struct {

	Vector    *Vector   `json:"vector"`

	Timestamp time.Time `json:"timestamp"`

}



// NewMockVectorStore creates a new mock vector store.

func NewMockVectorStore() *MockVectorStore {

	return &MockVectorStore{

		data: make(map[string]*vectorEntry),

	}

}



// Store implements VectorStore interface.

func (mvs *MockVectorStore) Store(ctx context.Context, vectors []Vector) error {

	mvs.mu.Lock()

	defer mvs.mu.Unlock()



	for _, vector := range vectors {

		mvs.data[vector.ID] = &vectorEntry{

			Vector:    &vector,

			Timestamp: time.Now(),

		}

	}



	return nil

}



// Search implements VectorStore interface.

func (mvs *MockVectorStore) Search(ctx context.Context, query []float32, limit int) ([]VectorSearchResult, error) {

	mvs.mu.RLock()

	defer mvs.mu.RUnlock()



	var results []VectorSearchResult



	for _, entry := range mvs.data {

		// Simple cosine similarity calculation.

		score := cosineSimilarity(query, entry.Vector.Embedding)



		results = append(results, VectorSearchResult{

			Vector: entry.Vector,

			Score:  score,

		})

	}



	// Sort by score (descending).

	sort.Slice(results, func(i, j int) bool {

		return results[i].Score > results[j].Score

	})



	// Limit results.

	if limit > 0 && len(results) > limit {

		results = results[:limit]

	}



	return results, nil

}



// Delete implements VectorStore interface.

func (mvs *MockVectorStore) Delete(ctx context.Context, ids []string) error {

	mvs.mu.Lock()

	defer mvs.mu.Unlock()



	for _, id := range ids {

		delete(mvs.data, id)

	}



	return nil

}



// GetStats implements VectorStore interface.

func (mvs *MockVectorStore) GetStats() VectorStoreStats {

	mvs.mu.RLock()

	defer mvs.mu.RUnlock()



	return VectorStoreStats{

		TotalVectors: int64(len(mvs.data)),

		IndexSize:    int64(len(mvs.data) * 1024), // Approximate size

	}

}



// cosineSimilarity calculates cosine similarity between two vectors.

func cosineSimilarity(a, b []float32) float32 {

	if len(a) != len(b) {

		return 0

	}



	var dotProduct, normA, normB float32



	for i := range len(a) {

		dotProduct += a[i] * b[i]

		normA += a[i] * a[i]

		normB += b[i] * b[i]

	}



	if normA == 0 || normB == 0 {

		return 0

	}



	return dotProduct / (float32(normA) * float32(normB))

}



// Mock RAG Provider for testing.



// MockRAGProvider is a simple mock implementation of RAGProvider.

type MockRAGProvider struct {

	name        string

	isHealthy   bool

	documents   []*shared.TelecomDocument

	mu          sync.RWMutex

	metrics     ProviderMetrics

	lastRequest time.Time

}



// NewMockRAGProvider creates a new mock RAG provider.

func NewMockRAGProvider(name string) *MockRAGProvider {

	return &MockRAGProvider{

		name:      name,

		isHealthy: true,

		documents: make([]*shared.TelecomDocument, 0),

		metrics: ProviderMetrics{

			CacheHitRate: 0.8,

		},

	}

}



// Search implements RAGProvider interface.

func (mrp *MockRAGProvider) Search(ctx context.Context, query string, options *QueryOptions) (*QueryResponse, error) {

	mrp.mu.Lock()

	defer mrp.mu.Unlock()



	mrp.lastRequest = time.Now()

	mrp.metrics.RequestCount++



	// Simple mock search logic.

	var results []*SearchResult



	for _, doc := range mrp.documents {

		// Simple keyword matching for mock.

		if contains(doc.Content, query) {

			// Create SearchResult using the correct shared.SearchResult struct.

			results = append(results, &SearchResult{

				Document: doc,

				Score:    rand.Float32(),

				Distance: rand.Float32(),

				Metadata: map[string]interface{}{"source": "mock", "provider": mrp.name},

			})

		}



		if options != nil && options.TopK > 0 && len(results) >= options.TopK {

			break

		}

	}



	return &QueryResponse{

		Query:         query,

		Results:       results,

		ProviderUsed:  mrp.name,

		EmbeddingCost: 0.01,

	}, nil

}



// Index implements RAGProvider interface.

func (mrp *MockRAGProvider) Index(ctx context.Context, documents []*shared.TelecomDocument) error {

	mrp.mu.Lock()

	defer mrp.mu.Unlock()



	mrp.documents = append(mrp.documents, documents...)

	return nil

}



// GetHealth implements RAGProvider interface.

func (mrp *MockRAGProvider) GetHealth() ProviderHealth {

	mrp.mu.RLock()

	defer mrp.mu.RUnlock()



	return ProviderHealth{

		IsHealthy: mrp.isHealthy,

		LastCheck: time.Now(),

		Latency:   10 * time.Millisecond,

		ErrorRate: 0.01,

		Details:   fmt.Sprintf("Mock provider %s", mrp.name),

	}

}



// GetMetrics implements RAGProvider interface.

func (mrp *MockRAGProvider) GetMetrics() ProviderMetrics {

	mrp.mu.RLock()

	defer mrp.mu.RUnlock()



	metrics := mrp.metrics

	metrics.LastRequestTime = mrp.lastRequest

	metrics.AverageLatency = 10 * time.Millisecond



	return metrics

}



// SetHealthy sets the health status for testing.

func (mrp *MockRAGProvider) SetHealthy(healthy bool) {

	mrp.mu.Lock()

	defer mrp.mu.Unlock()

	mrp.isHealthy = healthy

}



// contains is a simple helper function for string matching.

func contains(text, substring string) bool {

	return len(substring) == 0 || len(text) >= len(substring) &&

		(text == substring || text[:len(substring)] == substring ||

			text[len(text)-len(substring):] == substring ||

			findSubstring(text, substring))

}



func findSubstring(text, substring string) bool {

	for i := 0; i <= len(text)-len(substring); i++ {

		if text[i:i+len(substring)] == substring {

			return true

		}

	}

	return false

}



// HTTP Handlers for RAG API.



// RAGHandler provides HTTP endpoints for RAG operations.

type RAGHandler struct {

	manager *EnhancedRAGManager

	logger  *slog.Logger

}



// NewRAGHandler creates a new RAG HTTP handler.

func NewRAGHandler(manager *EnhancedRAGManager) *RAGHandler {

	return &RAGHandler{

		manager: manager,

		logger:  slog.Default().With("component", "rag-handler"),

	}

}



// HandleQuery handles RAG query requests.

func (rh *RAGHandler) HandleQuery(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return

	}



	var request struct {

		Query   string        `json:"query"`

		Options *QueryOptions `json:"options"`

	}



	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {

		http.Error(w, "Invalid request body", http.StatusBadRequest)

		return

	}



	if request.Query == "" {

		http.Error(w, "Query is required", http.StatusBadRequest)

		return

	}



	response, err := rh.manager.Query(r.Context(), request.Query, request.Options)

	if err != nil {

		rh.logger.Error("Query failed", "error", err)

		http.Error(w, "Query failed", http.StatusInternalServerError)

		return

	}



	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(response)

}



// HandleHealth handles health check requests.

func (rh *RAGHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return

	}



	health := rh.manager.GetHealth()



	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(health)

}



// HandleMetrics handles metrics requests.

func (rh *RAGHandler) HandleMetrics(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return

	}



	metrics := rh.manager.GetMetrics()



	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(metrics)

}



// Example Usage and Integration.



// RAGIntegrationExample demonstrates how to use the enhanced RAG system.

func RAGIntegrationExample() {

	// Create configuration.

	config := &EnhancedRAGConfig{

		ProviderStrategy:     "performance_based",

		DefaultProvider:      "weaviate",

		CacheEnabled:         true,

		CacheTTL:             5 * time.Minute,

		MaxConcurrentQueries: 10,

		QueryTimeout:         30 * time.Second,

		MaxRetries:           3,

		RetryDelay:           time.Second,

		HealthCheckInterval:  time.Minute,

	}



	// Create RAG manager.

	manager := NewEnhancedRAGManager(config)



	// Register providers.

	weaviateProvider := NewMockRAGProvider("weaviate")

	chromaProvider := NewMockRAGProvider("chroma")



	manager.RegisterProvider("weaviate", weaviateProvider)

	manager.RegisterProvider("chroma", chromaProvider)



	// Index some documents.

	documents := []*shared.TelecomDocument{

		{

			ID:      "doc1",

			Title:   "5G Network Architecture",

			Content: "5G networks use a service-based architecture...",

		},

		{

			ID:      "doc2",

			Title:   "O-RAN Alliance Specifications",

			Content: "Open RAN specifications define disaggregated RAN...",

		},

	}



	ctx := context.Background()

	if err := manager.IndexDocuments(ctx, documents); err != nil {

		log.Printf("Failed to index documents: %v", err)

		return

	}



	// Perform queries.

	queryOptions := &QueryOptions{

		TopK:            5,

		ScoreThreshold:  0.7,

		IncludeMetadata: true,

	}



	response, err := manager.Query(ctx, "5G network architecture", queryOptions)

	if err != nil {

		log.Printf("Query failed: %v", err)

		return

	}



	log.Printf("Query successful: found %d results in %v",

		len(response.Results), response.ProcessingTime)



	// Check health.

	health := manager.GetHealth()

	for provider, status := range health {

		log.Printf("Provider %s health: %v", provider, status.IsHealthy)

	}



	// Get metrics.

	metrics := manager.GetMetrics()

	for provider, metric := range metrics {

		log.Printf("Provider %s metrics: %+v", provider, metric)

	}

}

