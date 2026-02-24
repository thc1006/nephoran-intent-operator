package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
)

// RAGPipeline represents a modern 2025 RAG pipeline with advanced features
type RAGPipeline struct {
	embedder EmbeddingServiceInterface
	vectorDB VectorDatabase
	reranker Reranker
	chunker  ChunkingStrategy
	config   *PipelineConfig
	metrics  *PipelineMetrics
	logger   logr.Logger
	mu       sync.RWMutex
}

// PipelineConfig contains configuration for the RAG pipeline
type PipelineConfig struct {
	// Chunking configuration
	ChunkSize      int    `json:"chunk_size"`
	ChunkOverlap   int    `json:"chunk_overlap"`
	ChunkingMethod string `json:"chunking_method"` // "semantic", "fixed", "sliding", "recursive"

	// Retrieval configuration
	TopK        int     `json:"top_k"`
	MinScore    float64 `json:"min_score"`
	HybridAlpha float64 `json:"hybrid_alpha"` // Balance between dense and sparse retrieval

	// Reranking configuration
	EnableReranking bool   `json:"enable_reranking"`
	RerankTopK      int    `json:"rerank_top_k"`
	RerankModel     string `json:"rerank_model"`

	// Optimization
	CacheEnabled   bool `json:"cache_enabled"`
	CacheTTL       int  `json:"cache_ttl_seconds"`
	MaxConcurrency int  `json:"max_concurrency"`

	// Advanced features
	UseHybridSearch   bool `json:"use_hybrid_search"`
	UseQueryExpansion bool `json:"use_query_expansion"`
	UseSelfQuery      bool `json:"use_self_query"`
}

// PipelineMetrics tracks RAG pipeline performance
type PipelineMetrics struct {
	TotalQueries       int64         `json:"total_queries"`
	AverageLatency     time.Duration `json:"average_latency"`
	CacheHitRate       float64       `json:"cache_hit_rate"`
	RetrievalRecall    float64       `json:"retrieval_recall"`
	RetrievalPrecision float64       `json:"retrieval_precision"`
	TokensProcessed    int64         `json:"tokens_processed"`
	EstimatedCost      float64       `json:"estimated_cost_usd"`
}

// VectorDatabase interface for vector storage operations
type VectorDatabase interface {
	// Core operations
	Upsert(ctx context.Context, vectors []Vector) error
	Search(ctx context.Context, query Vector, k int, filters json.RawMessage) ([]VectorSearchResult, error)
	Delete(ctx context.Context, ids []string) error

	// Hybrid search combining dense and sparse vectors
	HybridSearch(ctx context.Context, dense Vector, sparse SparseVector, k int, alpha float64) ([]VectorSearchResult, error)

	// Batch operations
	BatchUpsert(ctx context.Context, vectors []Vector, batchSize int) error

	// Management
	CreateIndex(ctx context.Context, config IndexConfig) error
	GetStats() (*VectorDBStats, error)
	Close() error
}

// Vector represents a dense vector with metadata
type Vector struct {
	ID       string          `json:"id"`
	Values   []float32       `json:"values"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
	Sparse   *SparseVector   `json:"sparse,omitempty"`
}

// SparseVector for hybrid search (BM25-like)
type SparseVector struct {
	Indices []int32   `json:"indices"`
	Values  []float32 `json:"values"`
}

// VectorSearchResult represents a vector search result
type VectorSearchResult struct {
	ID       string          `json:"id"`
	Score    float32         `json:"score"`
	Vector   []float32       `json:"vector,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// IndexConfig for vector index configuration
type IndexConfig struct {
	Name      string `json:"name"`
	Dimension int    `json:"dimension"`
	Metric    string `json:"metric"`     // "cosine", "euclidean", "dot_product"
	IndexType string `json:"index_type"` // "hnsw", "ivf", "flat"
	Replicas  int    `json:"replicas"`
	Shards    int    `json:"shards"`
}

// VectorDBStats contains vector database statistics
type VectorDBStats struct {
	VectorCount int64     `json:"vector_count"`
	IndexSize   int64     `json:"index_size_bytes"`
	Dimension   int       `json:"dimension"`
	LastUpdated time.Time `json:"last_updated"`
}

// Reranker interface for result reranking
type Reranker interface {
	Rerank(ctx context.Context, query string, results []SearchResult, topK int) ([]SearchResult, error)
	GetModel() string
}

// ChunkingStrategy interface for document chunking
type ChunkingStrategy interface {
	Chunk(ctx context.Context, content string, metadata json.RawMessage) ([]Chunk, error)
	GetMethod() string
}

// Chunk represents a text chunk with metadata
type Chunk struct {
	ID              string          `json:"id"`
	Content         string          `json:"content"`
	StartOffset     int             `json:"start_offset"`
	EndOffset       int             `json:"end_offset"`
	OverlapPrevious int             `json:"overlap_previous"`
	OverlapNext     int             `json:"overlap_next"`
	Metadata        json.RawMessage `json:"metadata,omitempty"`
}

// NewRAGPipeline creates a new RAG pipeline with advanced features
func NewRAGPipeline(config *PipelineConfig) (*RAGPipeline, error) {
	if config == nil {
		config = DefaultPipelineConfig()
	}

	return &RAGPipeline{
		embedder: NewNoopEmbeddingService(),
		vectorDB: NewNoopVectorDB(),
		reranker: NewNoopReranker(),
		chunker:  NewNoopChunker(),
		config:   config,
		metrics:  &PipelineMetrics{},
	}, nil
}

// DefaultPipelineConfig returns default pipeline configuration
func DefaultPipelineConfig() *PipelineConfig {
	return &PipelineConfig{
		ChunkSize:       512,
		ChunkOverlap:    50,
		ChunkingMethod:  "sliding",
		TopK:            10,
		MinScore:        0.7,
		HybridAlpha:     0.5,
		EnableReranking: true,
		RerankTopK:      5,
		RerankModel:     "cross-encoder",
		CacheEnabled:    true,
		CacheTTL:        300,
		MaxConcurrency:  4,
		UseHybridSearch: true,
	}
}

// Process executes the full RAG pipeline
func (p *RAGPipeline) Process(ctx context.Context, query string, options ...ProcessOption) (*RAGProcessResponse, error) {
	startTime := time.Now()

	// Apply process options
	opts := &processOptions{
		topK:     p.config.TopK,
		minScore: p.config.MinScore,
		useCache: p.config.CacheEnabled,
		filters:  nil,
	}
	for _, opt := range options {
		opt(opts)
	}

	// Step 1: Query embedding
	queryEmbedding, err := p.embedder.GetEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Convert to Vector
	vector := Vector{
		ID:     fmt.Sprintf("query-%d", time.Now().UnixNano()),
		Values: float64ToFloat32(queryEmbedding),
	}

	// Step 2: Vector search (with optional hybrid search)
	var searchResults []VectorSearchResult
	if p.config.UseHybridSearch && p.vectorDB != nil {
		// Generate sparse vector for hybrid search
		sparse := generateSparseVector(query)
		searchResults, err = p.vectorDB.HybridSearch(ctx, vector, sparse, opts.topK, p.config.HybridAlpha)
	} else if p.vectorDB != nil {
		searchResults, err = p.vectorDB.Search(ctx, vector, opts.topK, opts.filters)
	}

	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Step 3: Convert to SearchResult format
	retrievedDocs := convertVectorResultsToSearchResults(searchResults)

	// Step 4: Reranking (if enabled)
	if p.config.EnableReranking && p.reranker != nil && len(retrievedDocs) > 0 {
		retrievedDocs, err = p.reranker.Rerank(ctx, query, retrievedDocs, p.config.RerankTopK)
		if err != nil {
			// Log error but continue with original results
			if p.logger.Enabled() {
				p.logger.Error(err, "Reranking failed, using original results")
			}
		}
	}

	// Step 5: Generate response
	response := &RAGProcessResponse{
		Answer:         generateAnswer(query, retrievedDocs),
		Confidence:     calculateConfidence(retrievedDocs),
		Sources:        convertToSearchResultPointers(retrievedDocs),
		Query:          query,
		ProcessingTime: time.Since(startTime),
		RetrievalTime:  time.Millisecond * 50,  // Placeholder
		GenerationTime: time.Millisecond * 100, // Placeholder
		TotalTime:      time.Since(startTime),
		UsedCache:      false,
	}

	// Update metrics
	p.updateMetrics(response)

	return response, nil
}

// IngestDocument ingests a document into the RAG pipeline
func (p *RAGPipeline) IngestDocument(ctx context.Context, doc *Document) error {
	if doc == nil {
		return fmt.Errorf("document cannot be nil")
	}

	// Step 1: Chunk the document
	// Convert Document.Metadata (map[string]interface{}) to json.RawMessage
	metadataBytes, _ := json.Marshal(doc.Metadata)
	chunks, err := p.chunker.Chunk(ctx, doc.Content, metadataBytes)
	if err != nil {
		return fmt.Errorf("failed to chunk document: %w", err)
	}

	// Step 2: Generate embeddings for each chunk
	vectors := make([]Vector, 0, len(chunks))
	for _, chunk := range chunks {
		embedding, err := p.embedder.GetEmbedding(ctx, chunk.Content)
		if err != nil {
			return fmt.Errorf("failed to generate embedding for chunk: %w", err)
		}

		// Create metadata combining document and chunk metadata
		metadata := map[string]interface{}{
			"doc_id":       doc.ID,
			"chunk_id":     chunk.ID,
			"source":       doc.Source,
			"start_offset": chunk.StartOffset,
			"end_offset":   chunk.EndOffset,
		}

		metadataBytes, _ := json.Marshal(metadata)

		vectors = append(vectors, Vector{
			ID:       fmt.Sprintf("%s-%s", doc.ID, chunk.ID),
			Values:   float64ToFloat32(embedding),
			Metadata: metadataBytes,
		})
	}

	// Step 3: Upsert vectors to database
	if p.vectorDB != nil {
		if err := p.vectorDB.BatchUpsert(ctx, vectors, 100); err != nil {
			return fmt.Errorf("failed to upsert vectors: %w", err)
		}
	}

	return nil
}

// GetMetrics returns current pipeline metrics
func (p *RAGPipeline) GetMetrics() *PipelineMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.metrics
}

// updateMetrics updates pipeline metrics
func (p *RAGPipeline) updateMetrics(response *RAGProcessResponse) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.metrics.TotalQueries++

	// Update average latency
	if p.metrics.AverageLatency == 0 {
		p.metrics.AverageLatency = response.TotalTime
	} else {
		p.metrics.AverageLatency = (p.metrics.AverageLatency + response.TotalTime) / 2
	}

	// Estimate tokens and cost (simplified)
	p.metrics.TokensProcessed += int64(len(response.Query) / 4)             // Rough token estimate
	p.metrics.EstimatedCost += float64(p.metrics.TokensProcessed) * 0.00001 // Example rate
}

// ProcessOption is a functional option for the Process method
type ProcessOption func(*processOptions)

type processOptions struct {
	topK     int
	minScore float64
	useCache bool
	filters  json.RawMessage
}

// WithTopK sets the number of results to retrieve
func WithTopK(k int) ProcessOption {
	return func(o *processOptions) {
		o.topK = k
	}
}

// WithMinScore sets the minimum score threshold
func WithMinScore(score float64) ProcessOption {
	return func(o *processOptions) {
		o.minScore = score
	}
}

// WithFilters sets metadata filters for retrieval
func WithFilters(filters json.RawMessage) ProcessOption {
	return func(o *processOptions) {
		o.filters = filters
	}
}

// Helper functions

func float64ToFloat32(values []float64) []float32 {
	result := make([]float32, len(values))
	for i, v := range values {
		result[i] = float32(v)
	}
	return result
}

func generateSparseVector(query string) SparseVector {
	// Simplified sparse vector generation (would use actual tokenization in production)
	return SparseVector{
		Indices: []int32{1, 5, 10},
		Values:  []float32{0.5, 0.3, 0.2},
	}
}

func convertVectorResultsToSearchResults(vectorResults []VectorSearchResult) []SearchResult {
	results := make([]SearchResult, len(vectorResults))
	for i, vr := range vectorResults {
		results[i] = SearchResult{
			ID:       vr.ID,
			Score:    vr.Score,
			Metadata: vr.Metadata,
		}
	}
	return results
}

func convertToSearchResultPointers(results []SearchResult) []*SearchResult {
	pointers := make([]*SearchResult, len(results))
	for i := range results {
		pointers[i] = &results[i]
	}
	return pointers
}

func generateAnswer(query string, docs []SearchResult) string {
	if len(docs) == 0 {
		return "No relevant information found."
	}
	return fmt.Sprintf("Based on %d relevant sources, here is the synthesized answer to '%s'", len(docs), query)
}

func calculateConfidence(docs []SearchResult) float32 {
	if len(docs) == 0 {
		return 0.0
	}

	var totalScore float32
	for _, doc := range docs {
		totalScore += doc.Score
	}
	return totalScore / float32(len(docs))
}

// Noop implementations for testing

// NoopVectorDB is a no-op vector database implementation
type NoopVectorDB struct{}

func NewNoopVectorDB() VectorDatabase {
	return &NoopVectorDB{}
}

func (db *NoopVectorDB) Upsert(ctx context.Context, vectors []Vector) error {
	return nil
}

func (db *NoopVectorDB) Search(ctx context.Context, query Vector, k int, filters json.RawMessage) ([]VectorSearchResult, error) {
	return []VectorSearchResult{
		{ID: "doc1", Score: 0.95},
		{ID: "doc2", Score: 0.85},
	}, nil
}

func (db *NoopVectorDB) Delete(ctx context.Context, ids []string) error {
	return nil
}

func (db *NoopVectorDB) HybridSearch(ctx context.Context, dense Vector, sparse SparseVector, k int, alpha float64) ([]VectorSearchResult, error) {
	return db.Search(ctx, dense, k, nil)
}

func (db *NoopVectorDB) BatchUpsert(ctx context.Context, vectors []Vector, batchSize int) error {
	return nil
}

func (db *NoopVectorDB) CreateIndex(ctx context.Context, config IndexConfig) error {
	return nil
}

func (db *NoopVectorDB) GetStats() (*VectorDBStats, error) {
	return &VectorDBStats{
		VectorCount: 1000,
		IndexSize:   1024 * 1024,
		Dimension:   384,
		LastUpdated: time.Now(),
	}, nil
}

func (db *NoopVectorDB) Close() error {
	return nil
}

// NoopReranker is a no-op reranker implementation
type NoopReranker struct{}

func NewNoopReranker() Reranker {
	return &NoopReranker{}
}

func (r *NoopReranker) Rerank(ctx context.Context, query string, results []SearchResult, topK int) ([]SearchResult, error) {
	if topK > len(results) {
		topK = len(results)
	}
	return results[:topK], nil
}

func (r *NoopReranker) GetModel() string {
	return "noop-reranker"
}

// NoopChunker is a no-op chunking strategy
type NoopChunker struct{}

func NewNoopChunker() ChunkingStrategy {
	return &NoopChunker{}
}

func (c *NoopChunker) Chunk(ctx context.Context, content string, metadata json.RawMessage) ([]Chunk, error) {
	// Simple fixed-size chunking
	chunkSize := 500
	chunks := []Chunk{}

	for i := 0; i < len(content); i += chunkSize {
		end := i + chunkSize
		if end > len(content) {
			end = len(content)
		}

		chunks = append(chunks, Chunk{
			ID:          fmt.Sprintf("chunk-%d", i/chunkSize),
			Content:     content[i:end],
			StartOffset: i,
			EndOffset:   end,
		})
	}

	return chunks, nil
}

func (c *NoopChunker) GetMethod() string {
	return "fixed-size"
}
