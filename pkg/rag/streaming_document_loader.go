package rag

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ledongthuc/pdf"
)

// StreamingDocumentProcessor handles streaming document processing with concurrent chunk handling
type StreamingDocumentProcessor struct {
	config            *StreamingConfig
	logger            *slog.Logger
	chunkingService   *ChunkingService
	embeddingService  *EmbeddingService
	memoryMonitor     *MemoryMonitor
	processingPool    *ProcessingPool
	metrics           *StreamingMetrics
	
	// Channels for streaming pipeline
	documentStream    chan *StreamingDocument
	chunkStream       chan *StreamingChunk
	embeddingStream   chan *EmbeddingTask
	resultStream      chan *ProcessingResult
	errorStream       chan error
	
	// Control channels
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
}

// StreamingConfig holds configuration for streaming document processing
type StreamingConfig struct {
	// Streaming parameters
	StreamBufferSize      int           `json:"stream_buffer_size"`       // Size of streaming buffers
	ChunkBufferSize       int           `json:"chunk_buffer_size"`        // Size of chunk processing buffer
	MaxConcurrentDocs     int           `json:"max_concurrent_docs"`      // Max documents processed concurrently
	MaxConcurrentChunks   int           `json:"max_concurrent_chunks"`    // Max chunks processed concurrently
	StreamingThreshold    int64         `json:"streaming_threshold"`      // File size threshold for streaming
	
	// Memory management
	MaxMemoryUsage        int64         `json:"max_memory_usage"`         // Maximum memory usage in bytes
	MemoryCheckInterval   time.Duration `json:"memory_check_interval"`    // How often to check memory usage
	BackpressureThreshold float64       `json:"backpressure_threshold"`   // Memory usage % to trigger backpressure
	
	// Processing configuration
	EnableParallelChunking bool         `json:"enable_parallel_chunking"` // Enable parallel chunk processing
	ChunkBatchSize        int           `json:"chunk_batch_size"`         // Chunks to process in batch
	EmbeddingBatchSize    int           `json:"embedding_batch_size"`     // Embeddings to generate in batch
	ProcessingTimeout     time.Duration `json:"processing_timeout"`       // Timeout for processing
	
	// Error handling
	MaxRetries            int           `json:"max_retries"`              // Maximum retries for failed operations
	RetryBackoff          time.Duration `json:"retry_backoff"`            // Backoff duration between retries
	ErrorThreshold        float64       `json:"error_threshold"`          // Error rate threshold to stop processing
}

// StreamingDocument represents a document being processed in streaming mode
type StreamingDocument struct {
	ID              string
	SourcePath      string
	Reader          io.Reader
	Size            int64
	Metadata        *DocumentMetadata
	ProcessingStart time.Time
}

// StreamingChunk represents a chunk being processed
type StreamingChunk struct {
	DocumentID      string
	Chunk           *DocumentChunk
	SequenceNumber  int
	ProcessingTime  time.Duration
}

// EmbeddingTask represents an embedding generation task
type EmbeddingTask struct {
	ChunkID         string
	Text            string
	Priority        int
	RetryCount      int
	SourceDocID     string
}

// ProcessingResult represents the result of processing a document
type ProcessingResult struct {
	DocumentID      string
	ChunksCreated   int
	EmbeddingsCreated int
	ProcessingTime  time.Duration
	Errors          []error
	Success         bool
}

// StreamingMetrics tracks streaming processing metrics
type StreamingMetrics struct {
	DocumentsProcessed    int64
	ChunksProcessed       int64
	EmbeddingsGenerated   int64
	BytesProcessed        int64
	ErrorCount            int64
	BackpressureEvents    int64
	AverageProcessingTime time.Duration
	PeakMemoryUsage       int64
	CurrentMemoryUsage    int64
	mutex                 sync.RWMutex
}

// NewStreamingDocumentProcessor creates a new streaming document processor
func NewStreamingDocumentProcessor(config *StreamingConfig, chunkingService *ChunkingService, embeddingService *EmbeddingService) *StreamingDocumentProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	
	processor := &StreamingDocumentProcessor{
		config:           config,
		logger:           slog.Default().With("component", "streaming-processor"),
		chunkingService:  chunkingService,
		embeddingService: embeddingService,
		memoryMonitor:    NewMemoryMonitor(config.MaxMemoryUsage),
		processingPool:   NewProcessingPool(config.MaxConcurrentDocs),
		metrics:          &StreamingMetrics{},
		
		// Initialize channels
		documentStream:   make(chan *StreamingDocument, config.StreamBufferSize),
		chunkStream:      make(chan *StreamingChunk, config.ChunkBufferSize),
		embeddingStream:  make(chan *EmbeddingTask, config.ChunkBufferSize),
		resultStream:     make(chan *ProcessingResult, config.MaxConcurrentDocs),
		errorStream:      make(chan error, 100),
		
		ctx:              ctx,
		cancel:           cancel,
	}
	
	// Start processing pipelines
	processor.startPipelines()
	
	return processor
}

// startPipelines starts all processing pipelines
func (sdp *StreamingDocumentProcessor) startPipelines() {
	// Start document processing workers
	for i := 0; i < sdp.config.MaxConcurrentDocs; i++ {
		sdp.wg.Add(1)
		go sdp.documentProcessor(i)
	}
	
	// Start chunk processing workers
	chunkWorkers := sdp.config.MaxConcurrentChunks
	if chunkWorkers == 0 {
		chunkWorkers = sdp.config.MaxConcurrentDocs * 2
	}
	for i := 0; i < chunkWorkers; i++ {
		sdp.wg.Add(1)
		go sdp.chunkProcessor(i)
	}
	
	// Start embedding generation workers
	embeddingWorkers := sdp.config.MaxConcurrentDocs
	for i := 0; i < embeddingWorkers; i++ {
		sdp.wg.Add(1)
		go sdp.embeddingProcessor(i)
	}
	
	// Start memory monitor
	sdp.wg.Add(1)
	go sdp.memoryMonitorWorker()
	
	// Start error handler
	sdp.wg.Add(1)
	go sdp.errorHandler()
}

// ProcessDocumentStream processes a document using streaming approach
func (sdp *StreamingDocumentProcessor) ProcessDocumentStream(ctx context.Context, doc *LoadedDocument) (*ProcessingResult, error) {
	startTime := time.Now()
	
	// Check if streaming is needed
	if doc.Size < sdp.config.StreamingThreshold {
		// Process normally for small documents
		return sdp.processSmallDocument(ctx, doc)
	}
	
	// Create streaming document
	streamingDoc := &StreamingDocument{
		ID:              doc.ID,
		SourcePath:      doc.SourcePath,
		Size:            doc.Size,
		Metadata:        doc.Metadata,
		ProcessingStart: startTime,
	}
	
	// Open file for streaming
	reader, err := sdp.openDocumentStream(doc.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open document stream: %w", err)
	}
	defer reader.Close()
	
	streamingDoc.Reader = reader
	
	// Submit to processing pipeline
	select {
	case sdp.documentStream <- streamingDoc:
		sdp.logger.Info("Document submitted for streaming processing", 
			"doc_id", doc.ID, 
			"size", doc.Size)
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(sdp.config.ProcessingTimeout):
		return nil, fmt.Errorf("timeout submitting document for processing")
	}
	
	// Wait for processing result
	select {
	case result := <-sdp.resultStream:
		if result.DocumentID == doc.ID {
			return result, nil
		}
		// Wrong result, this shouldn't happen
		return nil, fmt.Errorf("received result for wrong document")
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(sdp.config.ProcessingTimeout):
		return nil, fmt.Errorf("timeout waiting for processing result")
	}
}

// documentProcessor processes documents from the stream
func (sdp *StreamingDocumentProcessor) documentProcessor(workerID int) {
	defer sdp.wg.Done()
	
	for {
		select {
		case doc := <-sdp.documentStream:
			sdp.processStreamingDocument(doc)
		case <-sdp.ctx.Done():
			return
		}
	}
}

// processStreamingDocument processes a single streaming document
func (sdp *StreamingDocumentProcessor) processStreamingDocument(doc *StreamingDocument) {
	result := &ProcessingResult{
		DocumentID:     doc.ID,
		ProcessingTime: time.Since(doc.ProcessingStart),
	}
	
	// Check memory before processing
	if !sdp.checkMemoryAvailable() {
		sdp.handleBackpressure()
	}
	
	// Process document in chunks
	chunkCount := 0
	errorCount := 0
	scanner := bufio.NewScanner(doc.Reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 64KB initial, 1MB max
	
	var currentChunk strings.Builder
	currentChunkSize := 0
	
	for scanner.Scan() {
		line := scanner.Text()
		lineSize := len(line) + 1 // +1 for newline
		
		// Check if adding this line would exceed chunk size
		if currentChunkSize+lineSize > sdp.chunkingService.config.ChunkSize {
			// Process current chunk
			if currentChunk.Len() > 0 {
				chunk := sdp.createChunkFromContent(doc, currentChunk.String(), chunkCount)
				select {
				case sdp.chunkStream <- chunk:
					chunkCount++
				case <-sdp.ctx.Done():
					result.Errors = append(result.Errors, sdp.ctx.Err())
					sdp.resultStream <- result
					return
				}
				
				// Reset for next chunk
				currentChunk.Reset()
				currentChunkSize = 0
			}
		}
		
		currentChunk.WriteString(line)
		currentChunk.WriteString("\n")
		currentChunkSize += lineSize
		
		// Update metrics
		atomic.AddInt64(&sdp.metrics.BytesProcessed, int64(lineSize))
	}
	
	// Process final chunk
	if currentChunk.Len() > 0 {
		chunk := sdp.createChunkFromContent(doc, currentChunk.String(), chunkCount)
		select {
		case sdp.chunkStream <- chunk:
			chunkCount++
		case <-sdp.ctx.Done():
			result.Errors = append(result.Errors, sdp.ctx.Err())
		}
	}
	
	if err := scanner.Err(); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("scanner error: %w", err))
		errorCount++
	}
	
	// Update result
	result.ChunksCreated = chunkCount
	result.Success = errorCount == 0
	result.ProcessingTime = time.Since(doc.ProcessingStart)
	
	// Update metrics
	atomic.AddInt64(&sdp.metrics.DocumentsProcessed, 1)
	if errorCount > 0 {
		atomic.AddInt64(&sdp.metrics.ErrorCount, int64(errorCount))
	}
	
	// Send result
	select {
	case sdp.resultStream <- result:
	case <-sdp.ctx.Done():
		sdp.logger.Warn("Failed to send processing result", "doc_id", doc.ID)
	}
}

// chunkProcessor processes chunks from the stream
func (sdp *StreamingDocumentProcessor) chunkProcessor(workerID int) {
	defer sdp.wg.Done()
	
	// Batch chunks for efficient processing
	batch := make([]*StreamingChunk, 0, sdp.config.ChunkBatchSize)
	batchTimer := time.NewTimer(100 * time.Millisecond)
	defer batchTimer.Stop()
	
	for {
		select {
		case chunk := <-sdp.chunkStream:
			batch = append(batch, chunk)
			
			if len(batch) >= sdp.config.ChunkBatchSize {
				sdp.processBatchedChunks(batch)
				batch = make([]*StreamingChunk, 0, sdp.config.ChunkBatchSize)
				batchTimer.Reset(100 * time.Millisecond)
			}
			
		case <-batchTimer.C:
			if len(batch) > 0 {
				sdp.processBatchedChunks(batch)
				batch = make([]*StreamingChunk, 0, sdp.config.ChunkBatchSize)
			}
			batchTimer.Reset(100 * time.Millisecond)
			
		case <-sdp.ctx.Done():
			// Process remaining batch
			if len(batch) > 0 {
				sdp.processBatchedChunks(batch)
			}
			return
		}
	}
}

// processBatchedChunks processes a batch of chunks
func (sdp *StreamingDocumentProcessor) processBatchedChunks(batch []*StreamingChunk) {
	// Apply any chunk-level processing (e.g., enhancement, validation)
	for _, streamingChunk := range batch {
		// Create embedding task
		task := &EmbeddingTask{
			ChunkID:     streamingChunk.Chunk.ID,
			Text:        streamingChunk.Chunk.CleanContent,
			Priority:    1,
			SourceDocID: streamingChunk.DocumentID,
		}
		
		select {
		case sdp.embeddingStream <- task:
			atomic.AddInt64(&sdp.metrics.ChunksProcessed, 1)
		case <-sdp.ctx.Done():
			return
		}
	}
}

// embeddingProcessor generates embeddings for chunks
func (sdp *StreamingDocumentProcessor) embeddingProcessor(workerID int) {
	defer sdp.wg.Done()
	
	// Batch embedding tasks for efficient processing
	batch := make([]*EmbeddingTask, 0, sdp.config.EmbeddingBatchSize)
	batchTimer := time.NewTimer(200 * time.Millisecond)
	defer batchTimer.Stop()
	
	for {
		select {
		case task := <-sdp.embeddingStream:
			batch = append(batch, task)
			
			if len(batch) >= sdp.config.EmbeddingBatchSize {
				sdp.processBatchedEmbeddings(batch)
				batch = make([]*EmbeddingTask, 0, sdp.config.EmbeddingBatchSize)
				batchTimer.Reset(200 * time.Millisecond)
			}
			
		case <-batchTimer.C:
			if len(batch) > 0 {
				sdp.processBatchedEmbeddings(batch)
				batch = make([]*EmbeddingTask, 0, sdp.config.EmbeddingBatchSize)
			}
			batchTimer.Reset(200 * time.Millisecond)
			
		case <-sdp.ctx.Done():
			// Process remaining batch
			if len(batch) > 0 {
				sdp.processBatchedEmbeddings(batch)
			}
			return
		}
	}
}

// processBatchedEmbeddings generates embeddings for a batch of tasks
func (sdp *StreamingDocumentProcessor) processBatchedEmbeddings(batch []*EmbeddingTask) {
	if len(batch) == 0 {
		return
	}
	
	// Extract texts for embedding
	texts := make([]string, len(batch))
	chunkIDs := make([]string, len(batch))
	for i, task := range batch {
		texts[i] = task.Text
		chunkIDs[i] = task.ChunkID
	}
	
	// Create embedding request
	request := &EmbeddingRequest{
		Texts:     texts,
		ChunkIDs:  chunkIDs,
		UseCache:  true,
		RequestID: fmt.Sprintf("stream_batch_%d", time.Now().UnixNano()),
	}
	
	// Generate embeddings with retry logic
	maxRetries := sdp.config.MaxRetries
	for attempt := 0; attempt <= maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(sdp.ctx, sdp.config.ProcessingTimeout)
		response, err := sdp.embeddingService.GenerateEmbeddings(ctx, request)
		cancel()
		
		if err == nil {
			atomic.AddInt64(&sdp.metrics.EmbeddingsGenerated, int64(len(response.Embeddings)))
			sdp.logger.Debug("Embeddings generated for batch", 
				"batch_size", len(batch),
				"cache_hits", response.CacheHits)
			break
		}
		
		// Handle error
		sdp.logger.Warn("Failed to generate embeddings", 
			"attempt", attempt+1,
			"error", err)
		
		if attempt < maxRetries {
			// Exponential backoff
			backoff := sdp.config.RetryBackoff * time.Duration(1<<uint(attempt))
			time.Sleep(backoff)
		} else {
			// Max retries exceeded, log error
			sdp.errorStream <- fmt.Errorf("failed to generate embeddings after %d attempts: %w", maxRetries+1, err)
			atomic.AddInt64(&sdp.metrics.ErrorCount, 1)
		}
	}
}

// memoryMonitorWorker monitors memory usage and triggers backpressure
func (sdp *StreamingDocumentProcessor) memoryMonitorWorker() {
	defer sdp.wg.Done()
	
	ticker := time.NewTicker(sdp.config.MemoryCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			currentUsage, maxUsage := sdp.memoryMonitor.GetMemoryUsage()
			
			// Update metrics
			sdp.updateMetrics(func(m *StreamingMetrics) {
				m.CurrentMemoryUsage = currentUsage
				if currentUsage > m.PeakMemoryUsage {
					m.PeakMemoryUsage = currentUsage
				}
			})
			
			// Check for backpressure threshold
			usagePercent := float64(currentUsage) / float64(maxUsage)
			if usagePercent > sdp.config.BackpressureThreshold {
				sdp.handleBackpressure()
			}
			
		case <-sdp.ctx.Done():
			return
		}
	}
}

// handleBackpressure handles memory backpressure
func (sdp *StreamingDocumentProcessor) handleBackpressure() {
	atomic.AddInt64(&sdp.metrics.BackpressureEvents, 1)
	sdp.logger.Warn("Memory backpressure detected, slowing down processing")
	
	// Simple backpressure: sleep to allow memory to be freed
	time.Sleep(500 * time.Millisecond)
	
	// Force garbage collection
	runtime.GC()
}

// errorHandler handles errors from the processing pipeline
func (sdp *StreamingDocumentProcessor) errorHandler() {
	defer sdp.wg.Done()
	
	errorCounts := make(map[string]int)
	
	for {
		select {
		case err := <-sdp.errorStream:
			sdp.logger.Error("Processing error", "error", err)
			
			// Track error types
			errorType := fmt.Sprintf("%T", err)
			errorCounts[errorType]++
			
			// Check if error threshold is exceeded
			totalErrors := 0
			for _, count := range errorCounts {
				totalErrors += count
			}
			
			errorRate := float64(totalErrors) / float64(sdp.metrics.DocumentsProcessed)
			if errorRate > sdp.config.ErrorThreshold {
				sdp.logger.Error("Error threshold exceeded, stopping processing",
					"error_rate", errorRate,
					"threshold", sdp.config.ErrorThreshold)
				sdp.cancel()
			}
			
		case <-sdp.ctx.Done():
			return
		}
	}
}

// Helper methods

func (sdp *StreamingDocumentProcessor) openDocumentStream(path string) (io.ReadCloser, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	return file, nil
}

func (sdp *StreamingDocumentProcessor) createChunkFromContent(doc *StreamingDocument, content string, sequenceNum int) *StreamingChunk {
	chunk := &DocumentChunk{
		ID:             fmt.Sprintf("%s_chunk_%d", doc.ID, sequenceNum),
		DocumentID:     doc.ID,
		Content:        content,
		CleanContent:   sdp.chunkingService.cleanChunkContent(content),
		ChunkIndex:     sequenceNum,
		CharacterCount: len(content),
		WordCount:      sdp.chunkingService.countWords(content),
		ProcessedAt:    time.Now(),
		ChunkType:      "text",
		DocumentMetadata: doc.Metadata,
	}
	
	return &StreamingChunk{
		DocumentID:     doc.ID,
		Chunk:          chunk,
		SequenceNumber: sequenceNum,
	}
}

func (sdp *StreamingDocumentProcessor) checkMemoryAvailable() bool {
	current, max := sdp.memoryMonitor.GetMemoryUsage()
	return float64(current)/float64(max) < sdp.config.BackpressureThreshold
}

func (sdp *StreamingDocumentProcessor) processSmallDocument(ctx context.Context, doc *LoadedDocument) (*ProcessingResult, error) {
	// For small documents, use the regular chunking service
	chunks, err := sdp.chunkingService.ChunkDocument(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("failed to chunk document: %w", err)
	}
	
	// Generate embeddings
	err = sdp.embeddingService.GenerateEmbeddingsForChunks(ctx, chunks)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}
	
	return &ProcessingResult{
		DocumentID:        doc.ID,
		ChunksCreated:     len(chunks),
		EmbeddingsCreated: len(chunks),
		ProcessingTime:    time.Since(time.Now()),
		Success:           true,
	}, nil
}

func (sdp *StreamingDocumentProcessor) updateMetrics(updater func(*StreamingMetrics)) {
	sdp.metrics.mutex.Lock()
	defer sdp.metrics.mutex.Unlock()
	updater(sdp.metrics)
}

// GetMetrics returns current streaming metrics
func (sdp *StreamingDocumentProcessor) GetMetrics() StreamingMetrics {
	sdp.metrics.mutex.RLock()
	defer sdp.metrics.mutex.RUnlock()
	return *sdp.metrics
}

// Shutdown gracefully shuts down the streaming processor
func (sdp *StreamingDocumentProcessor) Shutdown(timeout time.Duration) error {
	sdp.logger.Info("Shutting down streaming processor")
	
	// Signal shutdown
	sdp.cancel()
	
	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		sdp.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		sdp.logger.Info("Streaming processor shut down successfully")
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("shutdown timeout exceeded")
	}
}