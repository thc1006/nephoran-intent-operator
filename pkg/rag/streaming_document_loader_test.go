package rag

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

// MockChunkingService for testing
type MockChunkingService struct {
	config *ChunkingConfig
}

func (m *MockChunkingService) ChunkDocument(ctx context.Context, doc *LoadedDocument) ([]*DocumentChunk, error) {
	// Simple mock implementation
	chunks := []*DocumentChunk{
		{
			ID:         doc.ID + "_chunk_0",
			DocumentID: doc.ID,
			Content:    doc.Content,
			ChunkIndex: 0,
		},
	}
	return chunks, nil
}

func (m *MockChunkingService) cleanChunkContent(content string) string {
	return strings.TrimSpace(content)
}

func (m *MockChunkingService) countWords(content string) int {
	return len(strings.Fields(content))
}

// MockEmbeddingService for testing
type MockEmbeddingService struct{}

func (m *MockEmbeddingService) GenerateEmbeddings(ctx context.Context, request *EmbeddingRequest) (*EmbeddingResponse, error) {
	embeddings := make([][]float32, len(request.Texts))
	for i := range embeddings {
		embeddings[i] = []float32{0.1, 0.2, 0.3} // Mock embedding
	}

	return &EmbeddingResponse{
		Embeddings:     embeddings,
		TokenUsage:     TokenUsage{TotalTokens: len(request.Texts) * 10},
		ProcessingTime: 100 * time.Millisecond,
	}, nil
}

func (m *MockEmbeddingService) GenerateEmbeddingsForChunks(ctx context.Context, chunks []*DocumentChunk) error {
	return nil
}

func TestStreamingDocumentProcessor_ProcessDocumentStream(t *testing.T) {
	config := &StreamingConfig{
		StreamBufferSize:       10,
		ChunkBufferSize:        10,
		MaxConcurrentDocs:      2,
		MaxConcurrentChunks:    4,
		StreamingThreshold:     1024,              // 1KB threshold
		MaxMemoryUsage:         100 * 1024 * 1024, // 100MB
		EnableParallelChunking: true,
		ChunkBatchSize:         2,
		EmbeddingBatchSize:     2,
		ProcessingTimeout:      30 * time.Second,
		MaxRetries:             3,
		RetryBackoff:           100 * time.Millisecond,
		ErrorThreshold:         0.1,
		MemoryCheckInterval:    1 * time.Second,
		BackpressureThreshold:  0.8,
	}

	chunkingService := &MockChunkingService{
		config: &ChunkingConfig{
			ChunkSize: 100,
		},
	}

	embeddingService := &MockEmbeddingService{}

	processor := NewStreamingDocumentProcessor(config, chunkingService, embeddingService)
	defer processor.Shutdown(5 * time.Second)

	tests := []struct {
		name    string
		doc     *LoadedDocument
		wantErr bool
	}{
		{
			name: "Small document (no streaming)",
			doc: &LoadedDocument{
				ID:      "test_small",
				Content: "This is a small test document.",
				Size:    30,
			},
			wantErr: false,
		},
		{
			name: "Large document (streaming)",
			doc: &LoadedDocument{
				ID:      "test_large",
				Content: strings.Repeat("This is a large test document. ", 100),
				Size:    3100,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := processor.ProcessDocumentStream(ctx, tt.doc)

			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessDocumentStream() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result.DocumentID != tt.doc.ID {
					t.Errorf("Expected document ID %s, got %s", tt.doc.ID, result.DocumentID)
				}

				if result.ChunksCreated == 0 {
					t.Error("Expected at least one chunk to be created")
				}

				if !result.Success {
					t.Error("Expected processing to be successful")
				}
			}
		})
	}
}

func TestStreamingDocumentProcessor_Backpressure(t *testing.T) {
	config := &StreamingConfig{
		StreamBufferSize:      2,
		ChunkBufferSize:       2,
		MaxConcurrentDocs:     1,
		MaxConcurrentChunks:   1,
		StreamingThreshold:    0,    // Always stream
		MaxMemoryUsage:        1024, // Very low to trigger backpressure
		BackpressureThreshold: 0.5,
		ProcessingTimeout:     5 * time.Second,
		MemoryCheckInterval:   100 * time.Millisecond,
	}

	chunkingService := &MockChunkingService{
		config: &ChunkingConfig{
			ChunkSize: 100,
		},
	}

	embeddingService := &MockEmbeddingService{}

	processor := NewStreamingDocumentProcessor(config, chunkingService, embeddingService)
	defer processor.Shutdown(5 * time.Second)

	// Simulate high memory usage
	processor.memoryMonitor.AllocateMemory(500) // 50% of max

	// Start processing
	go func() {
		time.Sleep(200 * time.Millisecond)
		// Allocate more memory to trigger backpressure
		processor.memoryMonitor.AllocateMemory(300) // Now at 80%
	}()

	// Process document
	doc := &LoadedDocument{
		ID:      "test_backpressure",
		Content: strings.Repeat("Test content. ", 50),
		Size:    700,
	}

	ctx := context.Background()
	result, err := processor.ProcessDocumentStream(ctx, doc)

	if err != nil {
		t.Fatalf("ProcessDocumentStream() error = %v", err)
	}

	// Check that backpressure was triggered
	metrics := processor.GetMetrics()
	if metrics.BackpressureEvents == 0 {
		t.Error("Expected backpressure events to be triggered")
	}

	if result.ChunksCreated == 0 {
		t.Error("Expected chunks to be created despite backpressure")
	}
}

func TestStreamingDocumentProcessor_ConcurrentProcessing(t *testing.T) {
	config := &StreamingConfig{
		StreamBufferSize:       10,
		ChunkBufferSize:        20,
		MaxConcurrentDocs:      3,
		MaxConcurrentChunks:    6,
		StreamingThreshold:     0, // Always stream
		MaxMemoryUsage:         100 * 1024 * 1024,
		ProcessingTimeout:      10 * time.Second,
		EnableParallelChunking: true,
		ChunkBatchSize:         3,
		EmbeddingBatchSize:     3,
	}

	chunkingService := &MockChunkingService{
		config: &ChunkingConfig{
			ChunkSize: 50,
		},
	}

	embeddingService := &MockEmbeddingService{}

	processor := NewStreamingDocumentProcessor(config, chunkingService, embeddingService)
	defer processor.Shutdown(5 * time.Second)

	// Process multiple documents concurrently
	numDocs := 5
	results := make(chan *ProcessingResult, numDocs)
	errors := make(chan error, numDocs)

	for i := 0; i < numDocs; i++ {
		go func(idx int) {
			doc := &LoadedDocument{
				ID:      fmt.Sprintf("test_concurrent_%d", idx),
				Content: strings.Repeat(fmt.Sprintf("Document %d content. ", idx), 20),
				Size:    int64(20 * (15 + len(fmt.Sprintf("%d", idx)))),
			}

			ctx := context.Background()
			result, err := processor.ProcessDocumentStream(ctx, doc)
			if err != nil {
				errors <- err
			} else {
				results <- result
			}
		}(i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < numDocs; i++ {
		select {
		case result := <-results:
			if result.Success {
				successCount++
			}
		case err := <-errors:
			t.Errorf("Concurrent processing error: %v", err)
		case <-time.After(15 * time.Second):
			t.Fatal("Timeout waiting for concurrent processing")
		}
	}

	if successCount != numDocs {
		t.Errorf("Expected %d successful documents, got %d", numDocs, successCount)
	}

	// Check metrics
	metrics := processor.GetMetrics()
	if metrics.DocumentsProcessed != int64(numDocs) {
		t.Errorf("Expected %d documents processed, got %d", numDocs, metrics.DocumentsProcessed)
	}
}

func TestStreamingDocumentProcessor_ErrorHandling(t *testing.T) {
	config := &StreamingConfig{
		StreamBufferSize:    5,
		ChunkBufferSize:     5,
		MaxConcurrentDocs:   1,
		MaxConcurrentChunks: 1,
		StreamingThreshold:  0,
		ProcessingTimeout:   2 * time.Second,
		MaxRetries:          2,
		RetryBackoff:        100 * time.Millisecond,
		ErrorThreshold:      0.5,
		MaxMemoryUsage:      100 * 1024 * 1024,
	}

	// Create a failing embedding service
	failingEmbeddingService := &FailingEmbeddingService{
		failureRate: 0.6, // 60% failure rate
	}

	chunkingService := &MockChunkingService{
		config: &ChunkingConfig{
			ChunkSize: 50,
		},
	}

	processor := NewStreamingDocumentProcessor(config, chunkingService, failingEmbeddingService)
	defer processor.Shutdown(5 * time.Second)

	doc := &LoadedDocument{
		ID:      "test_error",
		Content: "Test document for error handling.",
		Size:    34,
	}

	ctx := context.Background()
	result, err := processor.ProcessDocumentStream(ctx, doc)

	// Should succeed due to retries
	if err != nil {
		t.Fatalf("ProcessDocumentStream() error = %v", err)
	}

	// Check error metrics
	metrics := processor.GetMetrics()
	if metrics.ErrorCount == 0 {
		t.Error("Expected some errors to be recorded")
	}

	if result.Success {
		// If successful, check that retries happened
		if len(result.Errors) == 0 && metrics.ErrorCount > 0 {
			t.Log("Document processed successfully after retries")
		}
	}
}

// FailingEmbeddingService simulates an embedding service with failures
type FailingEmbeddingService struct {
	failureRate float64
	callCount   int
}

func (f *FailingEmbeddingService) GenerateEmbeddings(ctx context.Context, request *EmbeddingRequest) (*EmbeddingResponse, error) {
	f.callCount++

	// Fail based on failure rate
	if float64(f.callCount%100)/100.0 < f.failureRate {
		return nil, fmt.Errorf("simulated embedding failure")
	}

	embeddings := make([][]float32, len(request.Texts))
	for i := range embeddings {
		embeddings[i] = []float32{0.1, 0.2, 0.3}
	}

	return &EmbeddingResponse{
		Embeddings:     embeddings,
		TokenUsage:     TokenUsage{TotalTokens: len(request.Texts) * 10},
		ProcessingTime: 100 * time.Millisecond,
	}, nil
}

func (f *FailingEmbeddingService) GenerateEmbeddingsForChunks(ctx context.Context, chunks []*DocumentChunk) error {
	f.callCount++
	if float64(f.callCount%100)/100.0 < f.failureRate {
		return fmt.Errorf("simulated chunk embedding failure")
	}
	return nil
}

// MockReader for testing streaming
type MockReader struct {
	content []byte
	pos     int
}

func NewMockReader(content string) *MockReader {
	return &MockReader{
		content: []byte(content),
		pos:     0,
	}
}

func (m *MockReader) Read(p []byte) (n int, err error) {
	if m.pos >= len(m.content) {
		return 0, io.EOF
	}

	n = copy(p, m.content[m.pos:])
	m.pos += n
	return n, nil
}

func (m *MockReader) Close() error {
	return nil
}

func TestStreamingDocumentProcessor_StreamingBehavior(t *testing.T) {
	config := &StreamingConfig{
		StreamBufferSize:    10,
		ChunkBufferSize:     10,
		MaxConcurrentDocs:   1,
		MaxConcurrentChunks: 2,
		StreamingThreshold:  0, // Always stream
		ProcessingTimeout:   5 * time.Second,
		MaxMemoryUsage:      100 * 1024 * 1024,
	}

	chunkingService := &MockChunkingService{
		config: &ChunkingConfig{
			ChunkSize: 50,
		},
	}

	embeddingService := &MockEmbeddingService{}

	processor := NewStreamingDocumentProcessor(config, chunkingService, embeddingService)

	// Override openDocumentStream for testing
	originalOpen := processor.openDocumentStream
	processor.openDocumentStream = func(path string) (io.ReadCloser, error) {
		// Return a mock reader with test content
		content := strings.Repeat("Line of test content.\n", 10)
		return NewMockReader(content), nil
	}
	defer func() {
		processor.openDocumentStream = originalOpen
		processor.Shutdown(5 * time.Second)
	}()

	doc := &LoadedDocument{
		ID:         "test_streaming",
		SourcePath: "mock://test.txt",
		Size:       220,
	}

	ctx := context.Background()
	result, err := processor.ProcessDocumentStream(ctx, doc)

	if err != nil {
		t.Fatalf("ProcessDocumentStream() error = %v", err)
	}

	if result.ChunksCreated == 0 {
		t.Error("Expected chunks to be created from streamed content")
	}

	metrics := processor.GetMetrics()
	if metrics.BytesProcessed == 0 {
		t.Error("Expected bytes to be processed")
	}

	t.Logf("Processed %d bytes into %d chunks", metrics.BytesProcessed, result.ChunksCreated)
}

func BenchmarkStreamingDocumentProcessor(b *testing.B) {
	config := &StreamingConfig{
		StreamBufferSize:       100,
		ChunkBufferSize:        100,
		MaxConcurrentDocs:      4,
		MaxConcurrentChunks:    8,
		StreamingThreshold:     1024,
		MaxMemoryUsage:         100 * 1024 * 1024,
		EnableParallelChunking: true,
		ChunkBatchSize:         10,
		EmbeddingBatchSize:     10,
		ProcessingTimeout:      30 * time.Second,
	}

	chunkingService := &MockChunkingService{
		config: &ChunkingConfig{
			ChunkSize: 500,
		},
	}

	embeddingService := &MockEmbeddingService{}

	processor := NewStreamingDocumentProcessor(config, chunkingService, embeddingService)
	defer processor.Shutdown(5 * time.Second)

	// Create test document
	doc := &LoadedDocument{
		ID:      "bench_doc",
		Content: strings.Repeat("This is a test sentence for benchmarking. ", 1000),
		Size:    42000,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		_, err := processor.ProcessDocumentStream(ctx, doc)
		if err != nil {
			b.Fatalf("Benchmark error: %v", err)
		}
	}

	b.StopTimer()

	metrics := processor.GetMetrics()
	b.Logf("Processed %d documents, %d chunks, %d bytes",
		metrics.DocumentsProcessed,
		metrics.ChunksProcessed,
		metrics.BytesProcessed)
}

// Add fmt import for Sprintf
func init() {
	// This is handled by the import statement at the top
}
