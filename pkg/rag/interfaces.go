// Package rag provides Retrieval-Augmented Generation interfaces
package rag

import (
	"context"
)

// EmbeddingServiceInterface defines the interface for embedding services
type EmbeddingServiceInterface interface {
	// Embed generates embeddings for the given text
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch generates embeddings for multiple texts
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

	// CalculateSimilarity calculates semantic similarity between two texts
	CalculateSimilarity(ctx context.Context, text1, text2 string) (float32, error)

	// GetDimension returns the dimension of the embeddings
	GetDimension() int

	// GetModel returns the model name being used
	GetModel() string

	// Close gracefully shuts down the embedding service
	Close() error

	// GetUsage returns token usage statistics
	GetUsage() EmbeddingTokenUsage

	// ResetUsage resets the token usage counters
	ResetUsage()

	// HealthCheck verifies the service is operational
	HealthCheck(ctx context.Context) error

	// GetCostEstimate returns cost estimate for the given text
	GetCostEstimate(text string) float64

	// GetMetrics returns service metrics
	GetMetrics() *EmbeddingMetrics
}

// Note: EmbeddingMetrics is defined in embedding_service.go
// Note: EmbeddingTokenUsage is defined in embedding_service.go