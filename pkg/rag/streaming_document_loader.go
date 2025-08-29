//go:build !disable_rag && !test

package rag

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// StreamingDocumentLoader provides memory-efficient document processing.

type StreamingDocumentLoader struct {
	logger *zap.Logger

	chunkingService *ChunkingService

	memoryThreshold int64 // bytes

	bufferSize int

	maxConcurrency int

	processPool *ProcessingPool

	metrics *streamingMetrics
}

// StreamingConfig holds configuration for streaming operations.

type StreamingConfig struct {
	MemoryThresholdMB int64

	BufferSizeKB int

	MaxConcurrency int

	BackpressureLimit int
}

// ProcessingPool definition moved to document_loader.go to avoid duplicates.

// streamingMetrics tracks performance metrics.

type streamingMetrics struct {
	documentsProcessed prometheus.Counter

	chunksProcessed prometheus.Counter

	bytesProcessed prometheus.Counter

	processingDuration prometheus.Histogram

	memoryUsage prometheus.Gauge

	backpressureEvents prometheus.Counter

	streamingThresholdHit prometheus.Counter
}

// Document definition moved to embedding_support.go to avoid duplicates.

// ProcessedChunk definition moved to pipeline.go to avoid duplicates.

// NewStreamingDocumentLoader creates a new streaming document loader.

func NewStreamingDocumentLoader(

	logger *zap.Logger,

	chunkingService *ChunkingService,

	config StreamingConfig,

) *StreamingDocumentLoader {

	if config.MemoryThresholdMB == 0 {

		config.MemoryThresholdMB = 100 // 100MB default

	}

	if config.BufferSizeKB == 0 {

		config.BufferSizeKB = 64 // 64KB default

	}

	if config.MaxConcurrency == 0 {

		config.MaxConcurrency = runtime.NumCPU()

	}

	metrics := &streamingMetrics{

		documentsProcessed: prometheus.NewCounter(prometheus.CounterOpts{

			Name: "rag_streaming_documents_processed_total",

			Help: "Total number of documents processed via streaming",
		}),

		chunksProcessed: prometheus.NewCounter(prometheus.CounterOpts{

			Name: "rag_streaming_chunks_processed_total",

			Help: "Total number of chunks processed",
		}),

		bytesProcessed: prometheus.NewCounter(prometheus.CounterOpts{

			Name: "rag_streaming_bytes_processed_total",

			Help: "Total bytes processed via streaming",
		}),

		processingDuration: prometheus.NewHistogram(prometheus.HistogramOpts{

			Name: "rag_streaming_processing_duration_seconds",

			Help: "Duration of document processing operations",

			Buckets: prometheus.DefBuckets,
		}),

		memoryUsage: prometheus.NewGauge(prometheus.GaugeOpts{

			Name: "rag_streaming_memory_usage_bytes",

			Help: "Current memory usage for streaming operations",
		}),

		backpressureEvents: prometheus.NewCounter(prometheus.CounterOpts{

			Name: "rag_streaming_backpressure_events_total",

			Help: "Total number of backpressure events",
		}),

		streamingThresholdHit: prometheus.NewCounter(prometheus.CounterOpts{

			Name: "rag_streaming_threshold_hit_total",

			Help: "Number of times streaming threshold was hit",
		}),
	}

	// Register metrics.

	prometheus.MustRegister(

		metrics.documentsProcessed,

		metrics.chunksProcessed,

		metrics.bytesProcessed,

		metrics.processingDuration,

		metrics.memoryUsage,

		metrics.backpressureEvents,

		metrics.streamingThresholdHit,
	)

	pool := &ProcessingPool{

		documentWorkers: make(chan func(), config.MaxConcurrency),

		chunkWorkers: make(chan func(), config.MaxConcurrency*2),

		embeddingWorkers: make(chan func(), config.MaxConcurrency),
	}

	// Start worker pools.

	for i := 0; i < config.MaxConcurrency; i++ {

		go pool.worker(pool.documentWorkers)

		go pool.worker(pool.chunkWorkers)

		go pool.worker(pool.embeddingWorkers)

	}

	loader := &StreamingDocumentLoader{

		logger: logger,

		chunkingService: chunkingService,

		memoryThreshold: config.MemoryThresholdMB * 1024 * 1024,

		bufferSize: config.BufferSizeKB * 1024,

		maxConcurrency: config.MaxConcurrency,

		processPool: pool,

		metrics: metrics,
	}

	// Start memory monitor.

	go loader.monitorMemory()

	return loader

}

// ProcessDocument processes a single document with streaming support.

func (l *StreamingDocumentLoader) ProcessDocument(

	ctx context.Context,

	doc Document,

	chunkProcessor func(ProcessedChunk) error,

) error {

	start := time.Now()

	defer func() {

		l.metrics.processingDuration.Observe(time.Since(start).Seconds())

		l.metrics.documentsProcessed.Inc()

	}()

	// Check if streaming is needed.

	if doc.Size > 0 && doc.Size > l.memoryThreshold {

		l.metrics.streamingThresholdHit.Inc()

		return l.processStreamingDocument(ctx, doc, chunkProcessor)

	}

	// For smaller documents, use regular processing.

	return l.processRegularDocument(ctx, doc, chunkProcessor)

}

// processStreamingDocument handles large documents with streaming.

func (l *StreamingDocumentLoader) processStreamingDocument(

	ctx context.Context,

	doc Document,

	chunkProcessor func(ProcessedChunk) error,

) error {

	l.logger.Info("Processing document via streaming",

		zap.String("document_id", doc.ID),

		zap.Int64("size_bytes", doc.Size))

	reader := bufio.NewReaderSize(strings.NewReader(doc.Content), l.bufferSize)

	var buffer []byte

	chunkIndex := 0

	bytesRead := int64(0)

	for {

		select {

		case <-ctx.Done():

			return ctx.Err()

		default:

		}

		// Read chunk with backpressure handling.

		chunk, err := l.readChunkWithBackpressure(reader, &bytesRead)

		if err == io.EOF {

			// Process final chunk if buffer has content.

			if len(buffer) > 0 {

				if err := l.processChunk(ctx, doc, string(buffer), chunkIndex, chunkProcessor); err != nil {

					return fmt.Errorf("failed to process final chunk: %w", err)

				}

			}

			break

		}

		if err != nil {

			return fmt.Errorf("failed to read chunk: %w", err)

		}

		buffer = append(buffer, chunk...)

		// Check if we have enough content for a chunk.

		if len(buffer) >= l.chunkingService.config.ChunkSize {

			content := string(buffer[:l.chunkingService.config.ChunkSize])

			buffer = buffer[l.chunkingService.config.ChunkSize:]

			if err := l.processChunk(ctx, doc, content, chunkIndex, chunkProcessor); err != nil {

				return fmt.Errorf("failed to process chunk %d: %w", chunkIndex, err)

			}

			chunkIndex++

		}

	}

	l.metrics.bytesProcessed.Add(float64(bytesRead))

	return nil

}

// processRegularDocument handles smaller documents in memory.

func (l *StreamingDocumentLoader) processRegularDocument(

	ctx context.Context,

	doc Document,

	chunkProcessor func(ProcessedChunk) error,

) error {

	// Read entire document.

	content, err := io.ReadAll(strings.NewReader(doc.Content))

	if err != nil {

		return fmt.Errorf("failed to read document: %w", err)

	}

	l.metrics.bytesProcessed.Add(float64(len(content)))

	// Convert to LoadedDocument for chunking.

	loadedDoc := &LoadedDocument{

		ID: doc.ID,

		Content: string(content),

		Size: int64(len(content)),

		LoadedAt: time.Now(),
	}

	// Chunk the document.

	chunks, err := l.chunkingService.ChunkDocument(ctx, loadedDoc)

	if err != nil {

		return fmt.Errorf("failed to chunk document: %w", err)

	}

	// Convert DocumentChunk objects to strings for parallel processing.

	chunkStrings := make([]string, len(chunks))

	for i, chunk := range chunks {

		chunkStrings[i] = chunk.CleanContent

	}

	// Process chunks in parallel.

	return l.processChunksParallel(ctx, doc, chunkStrings, chunkProcessor)

}

// processChunk processes a single chunk.

func (l *StreamingDocumentLoader) processChunk(

	ctx context.Context,

	doc Document,

	content string,

	position int,

	processor func(ProcessedChunk) error,

) error {

	chunk := ProcessedChunk{

		Content: content,

		Metadata: doc.Metadata,

		ChunkIndex: position,
	}

	l.metrics.chunksProcessed.Inc()

	return processor(chunk)

}

// processChunksParallel processes multiple chunks concurrently.

func (l *StreamingDocumentLoader) processChunksParallel(

	ctx context.Context,

	doc Document,

	chunks []string,

	processor func(ProcessedChunk) error,

) error {

	errChan := make(chan error, len(chunks))

	var wg sync.WaitGroup

	for i, chunk := range chunks {

		wg.Add(1)

		i := i

		chunk := chunk

		l.processPool.chunkWorkers <- func() {

			defer wg.Done()

			if err := l.processChunk(ctx, doc, chunk, i, processor); err != nil {

				errChan <- err

			}

		}

	}

	wg.Wait()

	close(errChan)

	// Check for errors.

	for err := range errChan {

		if err != nil {

			return err

		}

	}

	return nil

}

// readChunkWithBackpressure reads data with backpressure handling.

func (l *StreamingDocumentLoader) readChunkWithBackpressure(

	reader *bufio.Reader,

	bytesRead *int64,

) ([]byte, error) {

	// Check memory pressure.

	var memStats runtime.MemStats

	runtime.ReadMemStats(&memStats)

	if memStats.Alloc > uint64(l.memoryThreshold) {

		l.metrics.backpressureEvents.Inc()

		// Apply backpressure - wait for memory to be freed.

		time.Sleep(100 * time.Millisecond)

		runtime.GC()

	}

	// Read chunk.

	chunk := make([]byte, l.bufferSize)

	n, err := reader.Read(chunk)

	if n > 0 {

		atomic.AddInt64(bytesRead, int64(n))

		return chunk[:n], err

	}

	return nil, err

}

// ProcessBatch processes multiple documents concurrently.

func (l *StreamingDocumentLoader) ProcessBatch(

	ctx context.Context,

	documents []Document,

	chunkProcessor func(ProcessedChunk) error,

) error {

	errChan := make(chan error, len(documents))

	var wg sync.WaitGroup

	for _, doc := range documents {

		wg.Add(1)

		doc := doc

		l.processPool.documentWorkers <- func() {

			defer wg.Done()

			if err := l.ProcessDocument(ctx, doc, chunkProcessor); err != nil {

				errChan <- fmt.Errorf("failed to process document %s: %w", doc.ID, err)

			}

		}

	}

	wg.Wait()

	close(errChan)

	// Collect errors.

	var errs []error

	for err := range errChan {

		if err != nil {

			errs = append(errs, err)

		}

	}

	if len(errs) > 0 {

		return fmt.Errorf("batch processing failed with %d errors: %v", len(errs), errs)

	}

	return nil

}

// monitorMemory tracks memory usage.

func (l *StreamingDocumentLoader) monitorMemory() {

	ticker := time.NewTicker(10 * time.Second)

	defer ticker.Stop()

	for range ticker.C {

		var memStats runtime.MemStats

		runtime.ReadMemStats(&memStats)

		l.metrics.memoryUsage.Set(float64(memStats.Alloc))

	}

}

// worker processes tasks from a channel.

func (p *ProcessingPool) worker(tasks chan func()) {

	for task := range tasks {

		task()

	}

}

// Shutdown gracefully shuts down the processing pool.

func (l *StreamingDocumentLoader) Shutdown() {

	close(l.processPool.documentWorkers)

	close(l.processPool.chunkWorkers)

	close(l.processPool.embeddingWorkers)

	l.processPool.activeTasks.Wait()

}
