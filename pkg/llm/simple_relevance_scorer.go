package llm

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/rag"
)

// SimpleRelevanceScorer provides a simple implementation of relevance scoring
// This implementation focuses on the core requirements:
// 1. Use rag.EmbeddingServiceInterface to compute embeddings and cosine similarity 
// 2. Handle error cases gracefully
// 3. Add comprehensive metrics tracking
// 4. Follow existing patterns from ContextBuilder
type SimpleRelevanceScorer struct {
	embeddingService    rag.EmbeddingServiceInterface // New interface-based field
	legacyEmbedding     *rag.EmbeddingService        // Legacy field for backward compatibility
	logger              *slog.Logger
	metrics             *SimpleRelevanceScorerMetrics
	mutex               sync.RWMutex
}

// SimpleRelevanceScorerMetrics tracks scoring performance
type SimpleRelevanceScorerMetrics struct {
	TotalScores      int64         `json:"total_scores"`
	SuccessfulScores int64         `json:"successful_scores"`
	FailedScores     int64         `json:"failed_scores"`
	AverageLatency   time.Duration `json:"average_latency"`
	EmbeddingCalls   int64         `json:"embedding_calls"`
	FallbackUses     int64         `json:"fallback_uses"`
	LastUpdated      time.Time     `json:"last_updated"`
	mutex            sync.RWMutex
}

// NewSimpleRelevanceScorer creates a new simple relevance scorer
func NewSimpleRelevanceScorer() *SimpleRelevanceScorer {
	return &SimpleRelevanceScorer{
		logger: slog.Default().With("component", "simple-relevance-scorer"),
		metrics: &SimpleRelevanceScorerMetrics{
			LastUpdated: time.Now(),
		},
	}
}

// NewSimpleRelevanceScorerWithEmbeddingService creates a scorer with a legacy embedding service
// This method is deprecated. Use NewSimpleRelevanceScorerWithEmbeddingInterface instead.
func NewSimpleRelevanceScorerWithEmbeddingService(embeddingService *rag.EmbeddingService) *SimpleRelevanceScorer {
	// Create adapter for backward compatibility
	var embeddingInterface rag.EmbeddingServiceInterface
	if embeddingService != nil {
		embeddingInterface = rag.NewEmbeddingServiceAdapter(embeddingService)
	}
	
	return &SimpleRelevanceScorer{
		embeddingService: embeddingInterface,
		legacyEmbedding:  embeddingService,
		logger:           slog.Default().With("component", "simple-relevance-scorer"),
		metrics: &SimpleRelevanceScorerMetrics{
			LastUpdated: time.Now(),
		},
	}
}

// NewSimpleRelevanceScorerWithEmbeddingInterface creates a scorer with an embedding service interface
func NewSimpleRelevanceScorerWithEmbeddingInterface(embeddingService rag.EmbeddingServiceInterface) *SimpleRelevanceScorer {
	return &SimpleRelevanceScorer{
		embeddingService: embeddingService,
		logger:           slog.Default().With("component", "simple-relevance-scorer"),
		metrics: &SimpleRelevanceScorerMetrics{
			LastUpdated: time.Now(),
		},
	}
}

// Score calculates the relevance score between a document and intent
func (srs *SimpleRelevanceScorer) Score(ctx context.Context, doc string, intent string) (float32, error) {
	startTime := time.Now()

	// Update metrics
	srs.updateMetrics(func(m *SimpleRelevanceScorerMetrics) {
		m.TotalScores++
	})

	// Validate inputs
	if err := srs.validateInputs(doc, intent); err != nil {
		srs.updateMetrics(func(m *SimpleRelevanceScorerMetrics) {
			m.FailedScores++
		})
		return 0.0, fmt.Errorf("invalid inputs: %w", err)
	}

	srs.logger.Debug("Computing relevance score",
		"doc_length", len(doc),
		"intent_length", len(intent),
	)

	var score float32
	var err error

	// Try to use embedding service if available
	if srs.embeddingService != nil {
		score, err = srs.calculateSemanticScore(ctx, doc, intent)
		if err != nil {
			srs.logger.Warn("Semantic scoring failed, falling back to keyword similarity", "error", err)
			// Fallback to simple similarity
			score = srs.calculateKeywordSimilarity(doc, intent)
			srs.updateMetrics(func(m *SimpleRelevanceScorerMetrics) {
				m.FallbackUses++
			})
		} else {
			srs.updateMetrics(func(m *SimpleRelevanceScorerMetrics) {
				m.EmbeddingCalls++
			})
		}
	} else {
		// No embedding service available, use keyword similarity
		srs.logger.Debug("No embedding service available, using keyword similarity")
		score = srs.calculateKeywordSimilarity(doc, intent)
		srs.updateMetrics(func(m *SimpleRelevanceScorerMetrics) {
			m.FallbackUses++
		})
	}

	// Update metrics
	processingTime := time.Since(startTime)
	srs.updateMetrics(func(m *SimpleRelevanceScorerMetrics) {
		m.SuccessfulScores++
		if m.SuccessfulScores > 0 {
			m.AverageLatency = (m.AverageLatency*time.Duration(m.SuccessfulScores-1) +
				processingTime) / time.Duration(m.SuccessfulScores)
		} else {
			m.AverageLatency = processingTime
		}
		m.LastUpdated = time.Now()
	})

	srs.logger.Debug("Relevance score computed successfully",
		"score", fmt.Sprintf("%.4f", score),
		"processing_time", processingTime,
	)

	return score, nil
}

// calculateSemanticScore uses the embedding service to calculate semantic similarity
func (srs *SimpleRelevanceScorer) calculateSemanticScore(ctx context.Context, doc string, intent string) (float32, error) {
	if srs.embeddingService == nil {
		return 0, fmt.Errorf("embedding service not available")
	}

	// Use the interface method directly for similarity calculation
	similarity, err := srs.embeddingService.CalculateSimilarity(ctx, doc, intent)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate similarity: %w", err)
	}

	// Convert to float32 and ensure it's in [0,1] range
	score := float32(similarity)
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score, nil
}

// cosineSimilarity calculates cosine similarity between two vectors
func (srs *SimpleRelevanceScorer) cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		srs.logger.Warn("Vector dimensions don't match", "a_len", len(a), "b_len", len(b))
		return 0.0
	}

	if len(a) == 0 {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// calculateKeywordSimilarity calculates similarity based on keyword overlap
func (srs *SimpleRelevanceScorer) calculateKeywordSimilarity(doc string, intent string) float32 {
	docWords := srs.extractWords(strings.ToLower(doc))
	intentWords := srs.extractWords(strings.ToLower(intent))

	if len(intentWords) == 0 {
		return 0.0
	}

	// Count overlapping keywords
	docWordSet := make(map[string]bool)
	for _, word := range docWords {
		docWordSet[word] = true
	}

	matchCount := 0
	for _, word := range intentWords {
		if docWordSet[word] {
			matchCount++
		}
	}

	// Calculate base similarity
	similarity := float32(matchCount) / float32(len(intentWords))

	// Apply telecommunications domain boosts
	similarity = srs.applyTelecomBoosts(doc, intent, similarity)

	// Cap at 1.0
	if similarity > 1.0 {
		similarity = 1.0
	}

	return similarity
}

// applyTelecomBoosts applies domain-specific boosts for telecommunications terms
func (srs *SimpleRelevanceScorer) applyTelecomBoosts(doc string, intent string, baseSimilarity float32) float32 {
	docLower := strings.ToLower(doc)
	intentLower := strings.ToLower(intent)

	// Define telecom-specific terms and their boost factors
	telecomTerms := map[string]float32{
		"5g":            1.2,
		"o-ran":         1.15,
		"amf":           1.1,
		"smf":           1.1,
		"upf":           1.1,
		"deployment":    1.08,
		"configuration": 1.06,
		"network":       1.05,
		"function":      1.04,
		"core":          1.03,
	}

	boost := float32(1.0)
	matchedTerms := 0

	for term, factor := range telecomTerms {
		if strings.Contains(docLower, term) && strings.Contains(intentLower, term) {
			boost *= factor
			matchedTerms++
		}
	}

	// Apply diminishing returns for multiple matches
	if matchedTerms > 0 {
		// Limit boost to reasonable levels
		if boost > 1.5 {
			boost = 1.5
		}
		return baseSimilarity * boost
	}

	return baseSimilarity
}

// extractWords extracts meaningful words from text, filtering out stop words
func (srs *SimpleRelevanceScorer) extractWords(text string) []string {
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
		"be": true, "been": true, "being": true, "have": true, "has": true, "had": true,
	}

	words := strings.Fields(text)
	var filteredWords []string

	for _, word := range words {
		// Clean word
		cleaned := strings.Trim(word, ".,!?;:()[]{}\"'")
		cleaned = strings.ToLower(cleaned)

		// Filter out stop words and short words
		if len(cleaned) > 2 && !stopWords[cleaned] {
			filteredWords = append(filteredWords, cleaned)
		}
	}

	return filteredWords
}

// validateInputs validates the input parameters
func (srs *SimpleRelevanceScorer) validateInputs(doc string, intent string) error {
	if doc == "" {
		return fmt.Errorf("document content cannot be empty")
	}
	if intent == "" {
		return fmt.Errorf("intent cannot be empty")
	}
	if len(doc) > 100000 { // 100KB limit
		return fmt.Errorf("document too large: %d characters (max: 100000)", len(doc))
	}
	if len(intent) > 10000 { // 10KB limit
		return fmt.Errorf("intent too large: %d characters (max: 10000)", len(intent))
	}
	return nil
}

// updateMetrics safely updates metrics
func (srs *SimpleRelevanceScorer) updateMetrics(updater func(*SimpleRelevanceScorerMetrics)) {
	srs.metrics.mutex.Lock()
	defer srs.metrics.mutex.Unlock()
	updater(srs.metrics)
}

// GetMetrics returns current metrics and status
func (srs *SimpleRelevanceScorer) GetMetrics() map[string]interface{} {
	srs.metrics.mutex.RLock()
	defer srs.metrics.mutex.RUnlock()

	successRate := 0.0
	if srs.metrics.TotalScores > 0 {
		successRate = float64(srs.metrics.SuccessfulScores) / float64(srs.metrics.TotalScores) * 100.0
	}

	return map[string]interface{}{
		"relevance_scorer_enabled":    true,
		"status":                     "active",
		"implementation":             "simple",
		"embedding_service_available": srs.embeddingService != nil,
		"using_interface_abstraction": true,
		"legacy_compatibility":        srs.legacyEmbedding != nil,
		"total_scores":               srs.metrics.TotalScores,
		"successful_scores":          srs.metrics.SuccessfulScores,
		"failed_scores":              srs.metrics.FailedScores,
		"success_rate":               fmt.Sprintf("%.2f%%", successRate),
		"embedding_calls":            srs.metrics.EmbeddingCalls,
		"fallback_uses":              srs.metrics.FallbackUses,
		"average_processing_time_ms": srs.metrics.AverageLatency.Milliseconds(),
		"last_updated":               srs.metrics.LastUpdated.Format(time.RFC3339),
	}
}