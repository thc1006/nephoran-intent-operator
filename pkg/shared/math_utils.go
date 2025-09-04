package shared

import "math"

// Sqrt calculates the square root of x
// Consolidated from pkg/rag/optimized_rag_pipeline.go and pkg/rag/embedding_service_interface.go
func Sqrt(x float64) float64 {
	return math.Sqrt(x)
}

// Additional mathematical utilities for RAG and LLM operations

// CosineSimilarity calculates the cosine similarity between two vectors
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (Sqrt(normA) * Sqrt(normB))
}

// EuclideanDistance calculates the Euclidean distance between two vectors
func EuclideanDistance(a, b []float32) float64 {
	if len(a) != len(b) {
		return math.Inf(1)
	}

	var sum float64
	for i := range a {
		diff := float64(a[i] - b[i])
		sum += diff * diff
	}

	return Sqrt(sum)
}

// Normalize normalizes a vector to unit length
func Normalize(vector []float32) []float32 {
	var norm float64
	for _, v := range vector {
		norm += float64(v * v)
	}

	norm = Sqrt(norm)
	if norm == 0 {
		return vector
	}

	normalized := make([]float32, len(vector))
	for i, v := range vector {
		normalized[i] = float32(float64(v) / norm)
	}

	return normalized
}
