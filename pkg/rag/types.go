//go:build !disable_rag && !test

package rag

import (
	"context"
	"time"
)

// EmbeddingProvider interface for embedding providers
type EmbeddingProvider interface {
	GetEmbeddings(ctx context.Context, texts []string) ([][]float64, error)
	IsHealthy() bool
	GetLatency() time.Duration
}

// TokenUsage definition moved to embedding_service.go to avoid duplicates

// EmbeddingRequestExt represents extended embedding request
type EmbeddingRequestExt struct {
	Texts     []string
	UseCache  bool
	RequestID string
	Priority  int
	Metadata  map[string]interface{}
}

// EmbeddingResponseExt represents extended embedding response
type EmbeddingResponseExt struct {
	Embeddings [][]float32
	TokenUsage *TokenUsage
	ModelUsed  string
}

// CostOptimizerConfig holds cost optimizer configuration
type CostOptimizerConfig struct {
	MaxCostPerRequest float64
}

// ParallelChunkConfig holds configuration for parallel chunk processor
type ParallelChunkConfig struct {
	MaxWorkers int
}

// StreamingProcessorConfig holds configuration for streaming processor
type StreamingProcessorConfig struct {
	BufferSize int
}

// StreamingDocumentProcessor processes documents using streaming
type StreamingDocumentProcessor struct {
	config *StreamingProcessorConfig
}

// NewStreamingDocumentProcessor creates a new streaming processor
func NewStreamingDocumentProcessor(
	config *StreamingConfig,
	chunkingService *ChunkingService,
	embeddingService interface{},
) *StreamingDocumentProcessor {
	return &StreamingDocumentProcessor{
		config: &StreamingProcessorConfig{BufferSize: 1024},
	}
}

// ProcessDocumentStream processes a document stream
func (sdp *StreamingDocumentProcessor) ProcessDocumentStream(ctx context.Context, doc *LoadedDocument) (interface{}, error) {
	// Stub implementation
	return nil, nil
}

// GetMetrics returns streaming processor metrics
func (sdp *StreamingDocumentProcessor) GetMetrics() interface{} {
	return map[string]interface{}{"status": "ok"}
}

// Shutdown shuts down the streaming processor
func (sdp *StreamingDocumentProcessor) Shutdown(timeout time.Duration) error {
	return nil
}

// NewParallelChunkProcessorExt creates a new parallel chunk processor with extended config
func NewParallelChunkProcessorExt(
	config *ParallelChunkConfig,
	chunkingService *ChunkingService,
	embeddingService interface{},
) *ParallelChunkProcessor {
	return nil // Stub implementation
}

// CostAwareEmbeddingServiceAdapter adapts CostAwareEmbeddingService to expected interface
type CostAwareEmbeddingServiceAdapter struct {
	service *CostAwareEmbeddingService
}

// NewCostAwareEmbeddingServiceAdapter creates an adapter
func NewCostAwareEmbeddingServiceAdapter(service *CostAwareEmbeddingService) *CostAwareEmbeddingServiceAdapter {
	return &CostAwareEmbeddingServiceAdapter{service: service}
}

// GenerateEmbeddingsOptimized generates embeddings with cost optimization
func (caesa *CostAwareEmbeddingServiceAdapter) GenerateEmbeddingsOptimized(ctx context.Context, request *EmbeddingRequestExt) (*EmbeddingResponseExt, error) {
	// Convert request format
	embeddingRequest := CostAwareEmbeddingRequest{
		Text:            request.Texts[0], // Take first text for now
		MaxBudget:       10.0,
		QualityRequired: 0.8,
		LatencyBudget:   5 * time.Second,
	}

	// Call the actual service
	response, err := caesa.service.GetEmbeddings(ctx, embeddingRequest)
	if err != nil {
		return nil, err
	}

	// Convert response format
	embeddings := make([][]float32, len(request.Texts))
	for i := range request.Texts {
		embeddings[i] = make([]float32, len(response.Embeddings))
		for j, val := range response.Embeddings {
			embeddings[i][j] = float32(val)
		}
	}

	return &EmbeddingResponseExt{
		Embeddings: embeddings,
		TokenUsage: &TokenUsage{EstimatedCost: response.Cost},
		ModelUsed:  response.Provider,
	}, nil
}

// Helper functions

func generateChunkID(docID string, index int) string {
	return docID + "_chunk_" + string(rune(index))
}

// generateDocumentID definition moved to enhanced_rag_integration.go to avoid duplicates
