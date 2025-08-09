package rag

import (
	"context"
	"fmt"
	"time"
)

// EmbeddingServiceInterface defines the interface for embedding services
// This interface provides a clean abstraction for different embedding implementations
type EmbeddingServiceInterface interface {
	// GetEmbedding generates a single embedding for the given text
	GetEmbedding(ctx context.Context, text string) ([]float64, error)

	// CalculateSimilarity calculates semantic similarity between two texts
	CalculateSimilarity(ctx context.Context, text1, text2 string) (float64, error)

	// GenerateEmbeddings generates multiple embeddings in batch
	GenerateEmbeddings(ctx context.Context, request *EmbeddingRequest) (*EmbeddingResponse, error)

	// HealthCheck verifies the service is operational
	HealthCheck(ctx context.Context) error

	// GetMetrics returns service metrics
	GetMetrics() *EmbeddingMetrics

	// Close releases resources
	Close() error
}

// EmbeddingServiceAdapter adapts the concrete EmbeddingService to the interface
// This provides backward compatibility while enabling proper interface abstraction
type EmbeddingServiceAdapter struct {
	service *EmbeddingService
}

// NewEmbeddingServiceAdapter creates a new adapter for the concrete EmbeddingService
func NewEmbeddingServiceAdapter(service *EmbeddingService) EmbeddingServiceInterface {
	return &EmbeddingServiceAdapter{
		service: service,
	}
}

// GetEmbedding implements EmbeddingServiceInterface
func (a *EmbeddingServiceAdapter) GetEmbedding(ctx context.Context, text string) ([]float64, error) {
	if a.service == nil {
		return nil, NewEmbeddingServiceError("service not initialized")
	}

	request := &EmbeddingRequest{
		Texts:     []string{text},
		UseCache:  true,
		RequestID: generateRequestID("single_embedding"),
		Metadata: map[string]interface{}{
			"operation": "single_embedding",
			"timestamp": time.Now(),
		},
	}

	response, err := a.service.GenerateEmbeddings(ctx, request)
	if err != nil {
		return nil, NewEmbeddingServiceError("failed to generate embedding: %w", err)
	}

	if len(response.Embeddings) == 0 {
		return nil, NewEmbeddingServiceError("no embeddings returned")
	}

	// Convert []float32 to []float64 for interface compatibility
	embedding := make([]float64, len(response.Embeddings[0]))
	for i, v := range response.Embeddings[0] {
		embedding[i] = float64(v)
	}

	return embedding, nil
}

// CalculateSimilarity implements EmbeddingServiceInterface
func (a *EmbeddingServiceAdapter) CalculateSimilarity(ctx context.Context, text1, text2 string) (float64, error) {
	if a.service == nil {
		return 0, NewEmbeddingServiceError("service not initialized")
	}

	// Generate embeddings for both texts
	request := &EmbeddingRequest{
		Texts:     []string{text1, text2},
		UseCache:  true,
		RequestID: generateRequestID("similarity_calculation"),
		Metadata: map[string]interface{}{
			"operation": "similarity_calculation",
			"timestamp": time.Now(),
		},
	}

	response, err := a.service.GenerateEmbeddings(ctx, request)
	if err != nil {
		return 0, NewEmbeddingServiceError("failed to generate embeddings for similarity: %w", err)
	}

	if len(response.Embeddings) < 2 {
		return 0, NewEmbeddingServiceError("insufficient embeddings for similarity calculation")
	}

	// Calculate cosine similarity
	embedding1 := response.Embeddings[0]
	embedding2 := response.Embeddings[1]

	similarity := calculateCosineSimilarity(embedding1, embedding2)
	return float64(similarity), nil
}

// GenerateEmbeddings implements EmbeddingServiceInterface
func (a *EmbeddingServiceAdapter) GenerateEmbeddings(ctx context.Context, request *EmbeddingRequest) (*EmbeddingResponse, error) {
	if a.service == nil {
		return nil, NewEmbeddingServiceError("service not initialized")
	}

	return a.service.GenerateEmbeddings(ctx, request)
}

// HealthCheck implements EmbeddingServiceInterface
func (a *EmbeddingServiceAdapter) HealthCheck(ctx context.Context) error {
	if a.service == nil {
		return NewEmbeddingServiceError("service not initialized")
	}

	status, err := a.service.CheckStatus(ctx)
	if err != nil {
		return NewEmbeddingServiceError("health check failed: %w", err)
	}

	if status.Status != "healthy" {
		return NewEmbeddingServiceError("service is not healthy: %s", status.Message)
	}

	return nil
}

// GetMetrics implements EmbeddingServiceInterface
func (a *EmbeddingServiceAdapter) GetMetrics() *EmbeddingMetrics {
	if a.service == nil {
		return &EmbeddingMetrics{
			LastUpdated: time.Now(),
		}
	}

	return a.service.GetMetrics()
}

// Close implements EmbeddingServiceInterface
func (a *EmbeddingServiceAdapter) Close() error {
	if a.service == nil {
		return nil
	}

	// The concrete EmbeddingService doesn't have a Close method,
	// so we don't need to do anything here.
	// In a production environment, you might want to add cleanup logic.
	return nil
}

// Helper functions

// calculateCosineSimilarity calculates cosine similarity between two float32 vectors
func calculateCosineSimilarity(a, b []float32) float32 {
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

	return dotProduct / (float32(sqrt(float64(normA))) * float32(sqrt(float64(normB))))
}

// sqrt implements square root for float64
func sqrt(x float64) float64 {
	if x < 0 {
		return 0
	}

	// Simple Newton-Raphson method for square root
	if x == 0 {
		return 0
	}

	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}

	return z
}

// generateRequestID generates a unique request ID
func generateRequestID(operation string) string {
	return operation + "_" + time.Now().Format("20060102150405") + "_" + time.Now().Format("000000")
}

// EmbeddingServiceError represents errors from the embedding service
type EmbeddingServiceError struct {
	message string
	cause   error
}

// NewEmbeddingServiceError creates a new embedding service error
func NewEmbeddingServiceError(format string, args ...interface{}) *EmbeddingServiceError {
	var message string
	var cause error

	if len(args) > 0 {
		// Check if the last argument is an error
		if lastArg, ok := args[len(args)-1].(error); ok {
			cause = lastArg
			message = fmt.Sprintf(format, args[:len(args)-1]...)
		} else {
			message = fmt.Sprintf(format, args...)
		}
	} else {
		message = format
	}

	return &EmbeddingServiceError{
		message: message,
		cause:   cause,
	}
}

// Error implements error interface
func (e *EmbeddingServiceError) Error() string {
	if e.cause != nil {
		return e.message + ": " + e.cause.Error()
	}
	return e.message
}

// Unwrap returns the underlying error
func (e *EmbeddingServiceError) Unwrap() error {
	return e.cause
}
