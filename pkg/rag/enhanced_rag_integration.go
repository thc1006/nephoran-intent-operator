//go:build !disable_rag && !test

package rag

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// EnhancedRAGService integrates all enhanced components
type EnhancedRAGService struct {
	streamingProcessor *StreamingDocumentProcessor
	parallelProcessor  *ParallelChunkProcessor
	costAwareEmbedding *CostAwareEmbeddingServiceAdapter
	vectorStore        VectorStore
	config             *EnhancedRAGConfig
	logger             *slog.Logger
	metrics            *enhancedRAGMetricsInternal
	mu                 sync.RWMutex
}

// EnhancedRAGConfig holds configuration for the enhanced RAG service
type EnhancedRAGConfig struct {
	// Component configurations
	StreamingConfig     *StreamingConfig     `json:"streaming_config"`
	ParallelChunkConfig *ParallelChunkConfig `json:"parallel_chunk_config"`
	CostOptimizerConfig *CostOptimizerConfig `json:"cost_optimizer_config"`

	// Processing options
	UseStreamingForLarge bool  `json:"use_streaming_for_large"`
	LargeFileThreshold   int64 `json:"large_file_threshold"`
	UseParallelChunking  bool  `json:"use_parallel_chunking"`

	// Vector store configuration
	VectorStoreType     string `json:"vector_store_type"` // "weaviate", "qdrant", "pinecone"
	VectorStoreEndpoint string `json:"vector_store_endpoint"`
	VectorStoreAPIKey   string `json:"vector_store_api_key"`

	// Performance settings
	MaxConcurrentIngestion int           `json:"max_concurrent_ingestion"`
	IngestionTimeout       time.Duration `json:"ingestion_timeout"`
	EnableMetrics          bool          `json:"enable_metrics"`
}

// EnhancedRAGMetrics tracks comprehensive metrics (public view without mutex)
type EnhancedRAGMetrics struct {
	// Document metrics
	DocumentsIngested   int64   `json:"documents_ingested"`
	DocumentsFailed     int64   `json:"documents_failed"`
	TotalDocumentSize   int64   `json:"total_document_size"`
	AverageDocumentSize float64 `json:"average_document_size"`

	// Processing metrics
	TotalChunksCreated     int64 `json:"total_chunks_created"`
	TotalEmbeddingsCreated int64 `json:"total_embeddings_created"`
	StreamingProcessed     int64 `json:"streaming_processed"`
	ParallelProcessed      int64 `json:"parallel_processed"`

	// Cost metrics
	TotalCost              float64            `json:"total_cost"`
	CostByProvider         map[string]float64 `json:"cost_by_provider"`
	AverageCostPerDocument float64            `json:"average_cost_per_document"`

	// Performance metrics
	AverageIngestionTime    time.Duration `json:"average_ingestion_time"`
	ThroughputDocsPerSecond float64       `json:"throughput_docs_per_second"`
	ThroughputMBPerSecond   float64       `json:"throughput_mb_per_second"`

	// Error metrics
	ErrorsByType  map[string]int64 `json:"errors_by_type"`
	LastError     error            `json:"last_error,omitempty"`
	LastErrorTime time.Time        `json:"last_error_time,omitempty"`
}

// enhancedRAGMetricsInternal tracks comprehensive metrics (internal with mutex)
type enhancedRAGMetricsInternal struct {
	// Document metrics
	DocumentsIngested   int64   `json:"documents_ingested"`
	DocumentsFailed     int64   `json:"documents_failed"`
	TotalDocumentSize   int64   `json:"total_document_size"`
	AverageDocumentSize float64 `json:"average_document_size"`

	// Processing metrics
	TotalChunksCreated     int64 `json:"total_chunks_created"`
	TotalEmbeddingsCreated int64 `json:"total_embeddings_created"`
	StreamingProcessed     int64 `json:"streaming_processed"`
	ParallelProcessed      int64 `json:"parallel_processed"`

	// Cost metrics
	TotalCost              float64            `json:"total_cost"`
	CostByProvider         map[string]float64 `json:"cost_by_provider"`
	AverageCostPerDocument float64            `json:"average_cost_per_document"`

	// Performance metrics
	AverageIngestionTime    time.Duration `json:"average_ingestion_time"`
	ThroughputDocsPerSecond float64       `json:"throughput_docs_per_second"`
	ThroughputMBPerSecond   float64       `json:"throughput_mb_per_second"`

	// Error metrics
	ErrorsByType  map[string]int64 `json:"errors_by_type"`
	LastError     error            `json:"last_error,omitempty"`
	LastErrorTime time.Time        `json:"last_error_time,omitempty"`

	mu sync.RWMutex
}

// NewEnhancedRAGService creates a new enhanced RAG service
func NewEnhancedRAGService(config *EnhancedRAGConfig) (*EnhancedRAGService, error) {
	logger := slog.Default().With("component", "enhanced-rag-service")

	// Initialize chunking service
	chunkingConfig := getDefaultChunkingConfig()
	chunkingService := NewChunkingService(chunkingConfig)

	// Initialize cost-aware service
	zapLogger := zap.NewNop() // Use nop logger for now
	baseCostAwareService := NewCostAwareEmbeddingService(zapLogger)
	costAwareEmbedding := NewCostAwareEmbeddingServiceAdapter(baseCostAwareService)

	// Initialize streaming processor
	streamingProcessor := NewStreamingDocumentProcessor(
		config.StreamingConfig,
		chunkingService,
		costAwareEmbedding,
	)

	// Initialize parallel processor
	parallelProcessor := NewParallelChunkProcessorExt(
		config.ParallelChunkConfig,
		chunkingService,
		costAwareEmbedding,
	)

	// Initialize vector store
	vectorStore, err := createVectorStore(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create vector store: %w", err)
	}

	service := &EnhancedRAGService{
		streamingProcessor: streamingProcessor,
		parallelProcessor:  parallelProcessor,
		costAwareEmbedding: costAwareEmbedding,
		vectorStore:        vectorStore,
		config:             config,
		logger:             logger,
		metrics: &enhancedRAGMetricsInternal{
			CostByProvider: make(map[string]float64),
			ErrorsByType:   make(map[string]int64),
		},
	}

	// Start metrics collection if enabled
	if config.EnableMetrics {
		go service.metricsCollector()
	}

	return service, nil
}

// IngestDocument ingests a document using the appropriate processing strategy
func (ers *EnhancedRAGService) IngestDocument(ctx context.Context, docPath string, metadata map[string]interface{}) error {
	startTime := time.Now()

	ers.logger.Info("Starting document ingestion",
		"path", docPath,
		"metadata", metadata)

	// Load document metadata
	fileInfo, err := os.Stat(docPath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	doc := &LoadedDocument{
		ID:         generateDocumentID(docPath),
		SourcePath: docPath,
		Size:       fileInfo.Size(),
		LoadedAt:   time.Now(),
		Metadata: &DocumentMetadata{
			Source: "local",
			Custom: metadata,
		},
	}

	// Determine processing strategy
	var chunks []*DocumentChunk
	var processingErr error

	if ers.shouldUseStreaming(doc) {
		// Use streaming for large documents
		ers.logger.Info("Using streaming processor for large document",
			"doc_id", doc.ID,
			"size", doc.Size)

		_, err := ers.streamingProcessor.ProcessDocumentStream(ctx, doc)
		if err != nil {
			processingErr = fmt.Errorf("streaming processing failed: %w", err)
		} else {
			ers.updateMetrics(func(m *enhancedRAGMetricsInternal) {
				m.StreamingProcessed++
			})
			// Note: StreamingProcessor handles chunks internally
			// We would need to modify it to return chunks for vector store
		}
	} else if ers.config.UseParallelChunking {
		// Use parallel processing for medium documents
		ers.logger.Info("Using parallel processor",
			"doc_id", doc.ID,
			"size", doc.Size)

		// First load the document content
		loader := NewDocumentLoader(nil)
		loadedDoc, err := loader.LoadDocument(ctx, docPath)
		if err != nil {
			processingErr = fmt.Errorf("failed to load document: %w", err)
		} else {
			chunks, err = ers.parallelProcessor.ProcessDocumentChunks(ctx, loadedDoc)
			if err != nil {
				processingErr = fmt.Errorf("parallel processing failed: %w", err)
			} else {
				ers.updateMetrics(func(m *enhancedRAGMetricsInternal) {
					m.ParallelProcessed++
				})
			}
		}
	} else {
		// Use standard processing for small documents
		ers.logger.Info("Using standard processor",
			"doc_id", doc.ID,
			"size", doc.Size)

		// Load and process normally
		loader := NewDocumentLoader(nil)
		loadedDoc, err := loader.LoadDocument(ctx, docPath)
		if err != nil {
			processingErr = fmt.Errorf("failed to load document: %w", err)
		} else {
			chunkingService := NewChunkingService(nil)
			chunks, err = chunkingService.ChunkDocument(ctx, loadedDoc)
			if err != nil {
				processingErr = fmt.Errorf("chunking failed: %w", err)
			}
		}
	}

	if processingErr != nil {
		ers.recordError(processingErr)
		return processingErr
	}

	// Generate embeddings for chunks (if not already done)
	var embeddingResponse *EmbeddingResponseExt
	if len(chunks) > 0 {
		ers.logger.Info("Generating embeddings for chunks",
			"chunk_count", len(chunks))

		// Extract texts for embedding
		texts := make([]string, len(chunks))
		for i, chunk := range chunks {
			texts[i] = chunk.CleanContent
		}

		// Use cost-aware embedding service
		embeddingRequest := &EmbeddingRequestExt{
			Texts:     texts,
			UseCache:  true,
			RequestID: fmt.Sprintf("doc_%s", doc.ID),
			Metadata: map[string]interface{}{
				"document_id": doc.ID,
				"chunk_count": len(chunks),
			},
		}

		var err error
		embeddingResponse, err = ers.costAwareEmbedding.GenerateEmbeddingsOptimized(ctx, embeddingRequest)
		if err != nil {
			ers.recordError(fmt.Errorf("embedding generation failed: %w", err))
			return err
		}

		// Store in vector database
		err = ers.storeChunksWithEmbeddings(ctx, chunks, embeddingResponse.Embeddings)
		if err != nil {
			ers.recordError(fmt.Errorf("vector store failed: %w", err))
			return err
		}

		// Update cost metrics
		ers.updateMetrics(func(m *enhancedRAGMetricsInternal) {
			m.TotalCost += embeddingResponse.TokenUsage.EstimatedCost
			if embeddingResponse.ModelUsed != "" {
				m.CostByProvider[embeddingResponse.ModelUsed] += embeddingResponse.TokenUsage.EstimatedCost
			}
		})
	}

	// Update metrics
	processingTime := time.Since(startTime)
	ers.updateMetrics(func(m *enhancedRAGMetricsInternal) {
		m.DocumentsIngested++
		m.TotalDocumentSize += doc.Size
		m.AverageDocumentSize = float64(m.TotalDocumentSize) / float64(m.DocumentsIngested)
		m.TotalChunksCreated += int64(len(chunks))
		m.TotalEmbeddingsCreated += int64(len(chunks))

		// Update timing metrics
		if m.DocumentsIngested == 1 {
			m.AverageIngestionTime = processingTime
		} else {
			m.AverageIngestionTime = time.Duration(
				(int64(m.AverageIngestionTime)*(m.DocumentsIngested-1) + int64(processingTime)) / m.DocumentsIngested,
			)
		}

		// Calculate throughput
		if processingTime > 0 {
			m.ThroughputDocsPerSecond = float64(m.DocumentsIngested) / time.Since(startTime).Seconds()
			m.ThroughputMBPerSecond = float64(m.TotalDocumentSize) / 1024 / 1024 / time.Since(startTime).Seconds()
		}

		if m.DocumentsIngested > 0 {
			m.AverageCostPerDocument = m.TotalCost / float64(m.DocumentsIngested)
		}
	})

	cost := 0.0
	if embeddingResponse != nil {
		cost = embeddingResponse.TokenUsage.EstimatedCost
	}
	ers.logger.Info("Document ingestion completed",
		"doc_id", doc.ID,
		"chunks_created", len(chunks),
		"processing_time", processingTime,
		"cost", cost)

	return nil
}

// IngestBatch ingests multiple documents concurrently
func (ers *EnhancedRAGService) IngestBatch(ctx context.Context, docPaths []string, metadata map[string]interface{}) error {
	if len(docPaths) == 0 {
		return nil
	}

	ers.logger.Info("Starting batch ingestion",
		"document_count", len(docPaths))

	// Create semaphore for concurrency control
	sem := make(chan struct{}, ers.config.MaxConcurrentIngestion)

	// Process documents concurrently
	var wg sync.WaitGroup
	errors := make(chan error, len(docPaths))

	for _, docPath := range docPaths {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Create timeout context
			timeoutCtx, cancel := context.WithTimeout(ctx, ers.config.IngestionTimeout)
			defer cancel()

			// Ingest document
			if err := ers.IngestDocument(timeoutCtx, path, metadata); err != nil {
				errors <- fmt.Errorf("failed to ingest %s: %w", path, err)
			}
		}(docPath)
	}

	// Wait for completion
	wg.Wait()
	close(errors)

	// Collect errors
	var errs []error
	for err := range errors {
		errs = append(errs, err)
		ers.updateMetrics(func(m *enhancedRAGMetricsInternal) {
			m.DocumentsFailed++
		})
	}

	if len(errs) > 0 {
		return fmt.Errorf("batch ingestion had %d errors: %v", len(errs), errs)
	}

	return nil
}

// Query performs a RAG query using the enhanced system
func (ers *EnhancedRAGService) Query(ctx context.Context, query string, options *QueryOptions) (*QueryResponse, error) {
	startTime := time.Now()

	ers.logger.Info("Processing query", "query", query)

	// Generate query embedding using cost-aware service
	embeddingRequest := &EmbeddingRequestExt{
		Texts:     []string{query},
		UseCache:  true,
		RequestID: fmt.Sprintf("query_%d", time.Now().UnixNano()),
		Priority:  10, // High priority for queries
	}

	embeddingResponse, err := ers.costAwareEmbedding.GenerateEmbeddingsOptimized(ctx, embeddingRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	if len(embeddingResponse.Embeddings) == 0 {
		return nil, fmt.Errorf("no embedding generated for query")
	}

	queryEmbedding := embeddingResponse.Embeddings[0]

	// Search vector store
	searchResults, err := ers.vectorStore.Search(ctx, queryEmbedding, options.TopK)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Construct response
	response := &QueryResponse{
		Query:          query,
		Results:        searchResults,
		ProcessingTime: time.Since(startTime),
		EmbeddingCost:  embeddingResponse.TokenUsage.EstimatedCost,
		ProviderUsed:   embeddingResponse.ModelUsed,
	}

	ers.logger.Info("Query completed",
		"processing_time", response.ProcessingTime,
		"results", len(response.Results),
		"cost", response.EmbeddingCost)

	return response, nil
}

// GetMetrics returns current service metrics
func (ers *EnhancedRAGService) GetMetrics() EnhancedRAGMetrics {
	ers.metrics.mu.RLock()
	defer ers.metrics.mu.RUnlock()

	// Create a copy without the mutex
	metrics := EnhancedRAGMetrics{
		DocumentsIngested:       ers.metrics.DocumentsIngested,
		DocumentsFailed:         ers.metrics.DocumentsFailed,
		TotalDocumentSize:       ers.metrics.TotalDocumentSize,
		AverageDocumentSize:     ers.metrics.AverageDocumentSize,
		TotalChunksCreated:      ers.metrics.TotalChunksCreated,
		TotalEmbeddingsCreated:  ers.metrics.TotalEmbeddingsCreated,
		StreamingProcessed:      ers.metrics.StreamingProcessed,
		ParallelProcessed:       ers.metrics.ParallelProcessed,
		TotalCost:               ers.metrics.TotalCost,
		AverageCostPerDocument:  ers.metrics.AverageCostPerDocument,
		AverageIngestionTime:    ers.metrics.AverageIngestionTime,
		ThroughputDocsPerSecond: ers.metrics.ThroughputDocsPerSecond,
		ThroughputMBPerSecond:   ers.metrics.ThroughputMBPerSecond,
		LastError:               ers.metrics.LastError,
		LastErrorTime:           ers.metrics.LastErrorTime,
		// Do not copy the mutex field
	}

	// Copy maps
	if ers.metrics.CostByProvider != nil {
		metrics.CostByProvider = make(map[string]float64)
		for k, v := range ers.metrics.CostByProvider {
			metrics.CostByProvider[k] = v
		}
	}

	if ers.metrics.ErrorsByType != nil {
		metrics.ErrorsByType = make(map[string]int64)
		for k, v := range ers.metrics.ErrorsByType {
			metrics.ErrorsByType[k] = v
		}
	}

	return metrics
}

// GetComponentMetrics returns metrics from all components
func (ers *EnhancedRAGService) GetComponentMetrics() map[string]interface{} {
	return map[string]interface{}{
		"streaming":      ers.streamingProcessor.GetMetrics(),
		"parallel":       ers.parallelProcessor.GetMetrics(),
		"embedding":      map[string]interface{}{"status": "ok"},
		"cost_optimizer": map[string]interface{}{"status": "ok"},
		"enhanced_rag":   ers.GetMetrics(),
	}
}

// Shutdown gracefully shuts down the service
func (ers *EnhancedRAGService) Shutdown(timeout time.Duration) error {
	ers.logger.Info("Shutting down enhanced RAG service")

	// Shutdown components
	var wg sync.WaitGroup
	errors := make(chan error, 3)

	// Shutdown streaming processor
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := ers.streamingProcessor.Shutdown(timeout); err != nil {
			errors <- fmt.Errorf("streaming processor shutdown error: %w", err)
		}
	}()

	// Shutdown parallel processor
	wg.Add(1)
	go func() {
		defer wg.Done()
		ers.parallelProcessor.Shutdown()
	}()

	// Close vector store connection
	wg.Add(1)
	go func() {
		defer wg.Done()
		if closer, ok := ers.vectorStore.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				errors <- fmt.Errorf("vector store close error: %w", err)
			}
		}
	}()

	// Wait for completion
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
		close(errors)
	}()

	select {
	case <-done:
		// Check for errors
		var shutdownErrors []error
		for err := range errors {
			shutdownErrors = append(shutdownErrors, err)
		}
		if len(shutdownErrors) > 0 {
			return fmt.Errorf("shutdown errors: %v", shutdownErrors)
		}
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("shutdown timeout exceeded")
	}
}

// Helper methods

func (ers *EnhancedRAGService) shouldUseStreaming(doc *LoadedDocument) bool {
	return ers.config.UseStreamingForLarge && doc.Size >= ers.config.LargeFileThreshold
}

func (ers *EnhancedRAGService) storeChunksWithEmbeddings(ctx context.Context, chunks []*DocumentChunk, embeddings [][]float32) error {
	if len(chunks) != len(embeddings) {
		return fmt.Errorf("chunk count (%d) does not match embedding count (%d)", len(chunks), len(embeddings))
	}

	// Store each chunk with its embedding
	for i, chunk := range chunks {
		err := ers.vectorStore.Store(ctx, chunk.ID, embeddings[i], map[string]interface{}{
			"document_id":   chunk.DocumentID,
			"content":       chunk.Content,
			"chunk_index":   chunk.ChunkIndex,
			"section_title": chunk.SectionTitle,
			"chunk_type":    chunk.ChunkType,
			"quality_score": chunk.QualityScore,
		})
		if err != nil {
			return fmt.Errorf("failed to store chunk %s: %w", chunk.ID, err)
		}
	}

	return nil
}

func (ers *EnhancedRAGService) updateMetrics(updater func(*enhancedRAGMetricsInternal)) {
	ers.metrics.mu.Lock()
	defer ers.metrics.mu.Unlock()
	updater(ers.metrics)
}

func (ers *EnhancedRAGService) recordError(err error) {
	ers.updateMetrics(func(m *enhancedRAGMetricsInternal) {
		errorType := fmt.Sprintf("%T", err)
		m.ErrorsByType[errorType]++
		m.LastError = err
		m.LastErrorTime = time.Now()
	})
}

func (ers *EnhancedRAGService) metricsCollector() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		metrics := ers.GetComponentMetrics()
		ers.logger.Info("Enhanced RAG metrics update",
			"metrics", metrics)
	}
}

// Helper functions

// generateDocumentID is now defined in document_loader.go

func detectFileType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".pdf":
		return "pdf"
	case ".txt":
		return "text"
	case ".md":
		return "markdown"
	case ".json":
		return "json"
	case ".xml":
		return "xml"
	default:
		return "unknown"
	}
}

func createVectorStore(config *EnhancedRAGConfig) (VectorStore, error) {
	// This is a placeholder - in real implementation, create appropriate vector store
	// based on config.VectorStoreType
	switch config.VectorStoreType {
	case "weaviate":
		// return NewWeaviateVectorStore(config.VectorStoreEndpoint, config.VectorStoreAPIKey)
		return nil, fmt.Errorf("weaviate vector store not implemented in this example")
	case "qdrant":
		// return NewQdrantVectorStore(config.VectorStoreEndpoint, config.VectorStoreAPIKey)
		return nil, fmt.Errorf("qdrant vector store not implemented in this example")
	default:
		return NewMockVectorStore(), nil
	}
}

// VectorStore interface
type VectorStore interface {
	Store(ctx context.Context, id string, embedding []float32, metadata map[string]interface{}) error
	Search(ctx context.Context, queryEmbedding []float32, topK int) ([]*EnhancedSearchResult, error)
	Delete(ctx context.Context, id string) error
}

// SearchResult represents a vector search result
// EnhancedSearchResult definition moved to enhanced_retrieval_service.go to avoid duplication

// QueryOptions holds options for RAG queries
type QueryOptions struct {
	TopK            int                    `json:"top_k"`
	ScoreThreshold  float32                `json:"score_threshold"`
	Filters         map[string]interface{} `json:"filters"`
	IncludeMetadata bool                   `json:"include_metadata"`
}

// QueryResponse represents a RAG query response
type QueryResponse struct {
	Query          string                  `json:"query"`
	Results        []*EnhancedSearchResult `json:"results"`
	ProcessingTime time.Duration           `json:"processing_time"`
	EmbeddingCost  float64                 `json:"embedding_cost"`
	ProviderUsed   string                  `json:"provider_used"`
}

// MockVectorStore is a simple in-memory vector store for testing
type MockVectorStore struct {
	data map[string]*vectorEntry
	mu   sync.RWMutex
}

type vectorEntry struct {
	embedding []float32
	metadata  map[string]interface{}
}

func NewMockVectorStore() *MockVectorStore {
	return &MockVectorStore{
		data: make(map[string]*vectorEntry),
	}
}

func (m *MockVectorStore) Store(ctx context.Context, id string, embedding []float32, metadata map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[id] = &vectorEntry{
		embedding: embedding,
		metadata:  metadata,
	}
	return nil
}

func (m *MockVectorStore) Search(ctx context.Context, queryEmbedding []float32, topK int) ([]*EnhancedSearchResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Simple cosine similarity search
	var results []*EnhancedSearchResult

	for id, entry := range m.data {
		score := cosineSimilarity(queryEmbedding, entry.embedding)
		results = append(results, &EnhancedSearchResult{
			SearchResult: &SearchResult{
				ID:         id,
				Score:      score,
				Metadata:   entry.metadata,
				Content:    "", // Add default empty content
				Confidence: float64(score),
			},
			RelevanceScore: score,
			QualityScore:   0.5, // Default value
			FreshnessScore: 0.5, // Default value
			AuthorityScore: 0.5, // Default value
			CombinedScore:  score,
		})
	}

	// Sort by score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Return top K
	if len(results) > topK {
		results = results[:topK]
	}

	return results, nil
}

func (m *MockVectorStore) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, id)
	return nil
}

func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}
