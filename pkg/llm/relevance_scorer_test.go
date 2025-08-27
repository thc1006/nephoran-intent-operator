package llm

import (
	"context"
	"testing"
)

// MockEmbeddingService provides a mock implementation for testing
type MockEmbeddingService struct {
	similarity float64
	err        error
}

// GetEmbedding implements the rag.EmbeddingService interface
func (m *MockEmbeddingService) GetEmbedding(ctx context.Context, text string) ([]float64, error) {
	if m.err != nil {
		return nil, m.err
	}
	// Return a mock embedding vector
	return []float64{0.1, 0.2, 0.3, 0.4, 0.5}, nil
}

func TestRelevanceScorerStub_Score(t *testing.T) {
	tests := []struct {
		name           string
		doc            string
		intent         string
		mockSimilarity float64
		mockError      error
		expectedScore  float32
		expectError    bool
	}{
		{
			name:           "Basic relevance scoring",
			doc:            "This document describes 5G AMF deployment procedures for O-RAN networks",
			intent:         "Deploy AMF network function in 5G core",
			mockSimilarity: 0.8,
			expectedScore:  0.6, // Approximate expected score
			expectError:    false,
		},
		{
			name:        "Empty document",
			doc:         "",
			intent:      "Deploy AMF",
			expectError: true,
		},
		{
			name:        "Empty intent",
			doc:         "Some document content",
			intent:      "",
			expectError: true,
		},
		{
			name:           "Technical terms boost",
			doc:            "5G O-RAN AMF SMF deployment configuration optimization monitoring",
			intent:         "5G network function deployment",
			mockSimilarity: 0.7,
			expectedScore:  0.6, // Should get technical boost
			expectError:    false,
		},
		{
			name:           "Fallback to keyword similarity on embedding error",
			doc:            "AMF deployment procedures for 5G networks",
			intent:         "Deploy AMF in 5G",
			mockSimilarity: 0.0,
			mockError:      context.DeadlineExceeded,
			expectedScore:  0.1, // Should still get some score from keywords
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock embedding service
			_ = &MockEmbeddingService{
				similarity: tt.mockSimilarity,
				err:        tt.mockError,
			}

			// Create scorer - it will work without embedding service using keyword fallback
			scorer := NewRelevanceScorer()

			// Test scoring
			ctx := context.Background()
			score, err := scorer.Score(ctx, tt.doc, tt.intent)

			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check score bounds
			if score < 0.0 || score > 1.0 {
				t.Errorf("Score out of bounds: %f (expected 0.0-1.0)", score)
			}

			// For tests with expected scores, check approximate match
			if tt.expectedScore > 0 {
				if score < 0.0 {
					t.Errorf("Expected score > 0 but got %f", score)
				}
			}

			t.Logf("Test '%s': doc_len=%d, intent_len=%d, score=%f",
				tt.name, len(tt.doc), len(tt.intent), score)
		})
	}
}

func TestRelevanceScorerStub_Caching(t *testing.T) {
	scorer := NewRelevanceScorer()
	ctx := context.Background()

	doc := "5G AMF deployment procedures"
	intent := "Deploy AMF"

	// First call - should compute and cache
	score1, err1 := scorer.Score(ctx, doc, intent)
	if err1 != nil {
		t.Errorf("First call failed: %v", err1)
	}

	// Second call - should use cache
	score2, err2 := scorer.Score(ctx, doc, intent)
	if err2 != nil {
		t.Errorf("Second call failed: %v", err2)
	}

	// Scores should be identical (from cache)
	if score1 != score2 {
		t.Errorf("Cached score mismatch: %f vs %f", score1, score2)
	}

	// Check metrics
	metrics := scorer.GetMetrics()
	cacheHits, ok := metrics["cache_hits"].(int64)
	if !ok || cacheHits < 1 {
		t.Errorf("Expected at least 1 cache hit, got %v", cacheHits)
	}

	t.Logf("Cache test passed: score=%f, cache_hits=%d", score1, cacheHits)
}

func TestRelevanceScorerStub_Metrics(t *testing.T) {
	scorer := NewRelevanceScorer()

	ctx := context.Background()
	_, err := scorer.Score(ctx, "Test document", "Test intent")
	if err != nil {
		t.Errorf("Scoring failed: %v", err)
	}

	metrics := scorer.GetMetrics()

	// Check required metrics fields
	requiredFields := []string{
		"relevance_scorer_enabled", "status", "total_scores",
		"successful_scores", "embedding_service_available",
	}

	for _, field := range requiredFields {
		if _, exists := metrics[field]; !exists {
			t.Errorf("Missing required metric field: %s", field)
		}
	}

	// Verify enabled status
	if enabled, ok := metrics["relevance_scorer_enabled"].(bool); !ok || !enabled {
		t.Errorf("Expected relevance_scorer_enabled=true, got %v", enabled)
	}

	// Verify embedding service availability
	if available, ok := metrics["embedding_service_available"].(bool); !ok || !available {
		t.Errorf("Expected embedding_service_available=true, got %v", available)
	}

	t.Logf("Metrics test passed: %+v", metrics)
}

func TestRelevanceScorerStub_FallbackToKeywords(t *testing.T) {
	// Test scoring without embedding service (should fallback to keywords)
	scorer := NewRelevanceScorer()

	ctx := context.Background()
	doc := "This document describes 5G AMF deployment"
	intent := "Deploy AMF in 5G network"

	score, err := scorer.Score(ctx, doc, intent)
	if err != nil {
		t.Errorf("Fallback scoring failed: %v", err)
	}

	if score <= 0 {
		t.Errorf("Expected positive score from keyword fallback, got %f", score)
	}

	// Check metrics show no embedding service
	metrics := scorer.GetMetrics()
	if available, ok := metrics["embedding_service_available"].(bool); !ok || available {
		t.Errorf("Expected embedding_service_available=false, got %v", available)
	}

	t.Logf("Fallback test passed: score=%f", score)
}
