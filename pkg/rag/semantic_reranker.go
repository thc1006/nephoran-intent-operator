//go:build !disable_rag && !test

package rag

import (
	"context"
	"fmt"
	"hash/fnv"
	"log/slog"
	"math"
	"sort"
	"strings"
	"sync"
)

// SemanticReranker provides semantic reranking of search results.

type SemanticReranker struct {
	config *RetrievalConfig

	logger *slog.Logger

	embeddingService *EmbeddingService

	crossEncoder *CrossEncoder

	mutex sync.RWMutex
}

// CrossEncoder provides cross-encoder scoring for query-document pairs.

type CrossEncoder struct {
	modelName string

	initialized bool

	scoreCache map[string]float32

	cacheMutex sync.RWMutex
}

// RerankingScore contains detailed scoring information.

type RerankingScore struct {
	SemanticSimilarity float32 `json:"semantic_similarity"`

	CrossEncoderScore float32 `json:"cross_encoder_score"`

	LexicalSimilarity float32 `json:"lexical_similarity"`

	StructuralRelevance float32 `json:"structural_relevance"`

	ContextualRelevance float32 `json:"contextual_relevance"`

	CombinedScore float32 `json:"combined_score"`

	Explanation string `json:"explanation"`
}

// NewSemanticReranker creates a new semantic reranker.

func NewSemanticReranker(config *RetrievalConfig) *SemanticReranker {

	return &SemanticReranker{

		config: config,

		logger: slog.Default().With("component", "semantic-reranker"),

		crossEncoder: &CrossEncoder{

			modelName: config.CrossEncoderModel,

			scoreCache: make(map[string]float32),
		},
	}

}

// RerankResults reranks search results using semantic similarity.

func (sr *SemanticReranker) RerankResults(ctx context.Context, query string, results []*EnhancedSearchResult) ([]*EnhancedSearchResult, error) {

	if len(results) <= 1 {

		return results, nil

	}

	sr.logger.Debug("Starting semantic reranking",

		"query", query,

		"result_count", len(results),
	)

	// Calculate detailed scores for each result.

	for i, result := range results {

		score, err := sr.calculateDetailedScore(ctx, query, result)

		if err != nil {

			sr.logger.Warn("Failed to calculate detailed score",

				"result_index", i,

				"error", err,
			)

			continue

		}

		// Update result with new scores.

		result.SemanticSimilarity = score.SemanticSimilarity

		result.ContextRelevance = score.ContextualRelevance

		result.CombinedScore = score.CombinedScore

		result.RelevanceReason = score.Explanation

		// Add processing note.

		if result.ProcessingNotes == nil {

			result.ProcessingNotes = []string{}

		}

		result.ProcessingNotes = append(result.ProcessingNotes, "semantic_reranked")

	}

	// Sort by combined score.

	sort.Slice(results, func(i, j int) bool {

		return results[i].CombinedScore > results[j].CombinedScore

	})

	sr.logger.Info("Semantic reranking completed",

		"query", query,

		"reranked_count", len(results),
	)

	return results, nil

}

// calculateDetailedScore calculates a comprehensive relevance score.

func (sr *SemanticReranker) calculateDetailedScore(ctx context.Context, query string, result *EnhancedSearchResult) (*RerankingScore, error) {

	if result.Document == nil {

		return &RerankingScore{}, fmt.Errorf("result document is nil")

	}

	score := &RerankingScore{}

	// 1. Semantic similarity (using embeddings if available).

	score.SemanticSimilarity = sr.calculateSemanticSimilarity(query, result.Document.Content)

	// 2. Cross-encoder score (if cross-encoder is available).

	crossScore, err := sr.calculateCrossEncoderScore(ctx, query, result.Document.Content)

	if err != nil {

		sr.logger.Debug("Cross-encoder scoring failed", "error", err)

		crossScore = score.SemanticSimilarity // Fallback to semantic similarity

	}

	score.CrossEncoderScore = crossScore

	// 3. Lexical similarity (keyword matching).

	score.LexicalSimilarity = sr.calculateLexicalSimilarity(query, result.Document.Content)

	// 4. Structural relevance (document structure and metadata).

	score.StructuralRelevance = sr.calculateStructuralRelevance(query, result)

	// 5. Contextual relevance (hierarchy and domain context).

	score.ContextualRelevance = sr.calculateContextualRelevance(query, result)

	// 6. Combined score with weights.

	score.CombinedScore = sr.calculateCombinedScore(score)

	// 7. Generate explanation.

	score.Explanation = sr.generateScoreExplanation(score, result)

	return score, nil

}

// calculateSemanticSimilarity calculates semantic similarity using embeddings.

func (sr *SemanticReranker) calculateSemanticSimilarity(query, content string) float32 {

	// This is a simplified implementation.

	// In practice, you would use actual embeddings and cosine similarity.

	// For now, use a heuristic based on term overlap and TF-IDF-like scoring.

	queryTerms := sr.extractImportantTerms(query)

	contentTerms := sr.extractImportantTerms(content)

	if len(queryTerms) == 0 || len(contentTerms) == 0 {

		return 0.0

	}

	// Calculate term overlap with TF-IDF weighting.

	overlap := 0.0

	totalWeight := 0.0

	for queryTerm, queryWeight := range queryTerms {

		totalWeight += queryWeight

		if contentWeight, exists := contentTerms[queryTerm]; exists {

			overlap += queryWeight * contentWeight

		}

	}

	if totalWeight == 0 {

		return 0.0

	}

	similarity := float32(overlap / totalWeight)

	if similarity > 1.0 {

		similarity = 1.0

	}

	return similarity

}

// extractImportantTerms extracts important terms with weights.

func (sr *SemanticReranker) extractImportantTerms(text string) map[string]float64 {

	terms := make(map[string]float64)

	words := strings.Fields(strings.ToLower(text))

	// Count term frequencies.

	termFreq := make(map[string]int)

	for _, word := range words {

		// Clean word.

		word = strings.Trim(word, ".,!?;:()[]{}\"'")

		if len(word) > 2 { // Only consider meaningful words

			termFreq[word]++

		}

	}

	// Calculate TF-IDF-like weights.

	totalWords := len(words)

	for term, freq := range termFreq {

		// Simple TF weight.

		tf := float64(freq) / float64(totalWords)

		// Boost technical terms.

		idf := 1.0

		if sr.isTechnicalTerm(term) {

			idf = 2.0

		}

		terms[term] = tf * idf

	}

	return terms

}

// isTechnicalTerm checks if a term is likely a technical term.

func (sr *SemanticReranker) isTechnicalTerm(term string) bool {

	// Check for acronyms (all caps).

	if len(term) >= 2 && strings.ToUpper(term) == term {

		return true

	}

	// Check for technical patterns.

	technicalPatterns := []string{

		"config", "parameter", "protocol", "interface", "algorithm",

		"network", "wireless", "radio", "frequency", "signal",

		"bandwidth", "latency", "throughput", "performance",

		"optimization", "authentication", "encryption", "security",
	}

	for _, pattern := range technicalPatterns {

		if strings.Contains(term, pattern) {

			return true

		}

	}

	return false

}

// calculateCrossEncoderScore calculates score using a cross-encoder model.

func (sr *SemanticReranker) calculateCrossEncoderScore(ctx context.Context, query, content string) (float32, error) {

	// Check cache first.

	cacheKey := sr.generateCacheKey(query, content)

	if score, exists := sr.crossEncoder.getCachedScore(cacheKey); exists {

		return score, nil

	}

	// For now, return a placeholder score based on lexical similarity.

	// In a full implementation, this would call an actual cross-encoder model.

	score := sr.calculateLexicalSimilarity(query, content)

	// Cache the score.

	sr.crossEncoder.setCachedScore(cacheKey, score)

	return score, nil

}

// generateCacheKey generates a cache key for cross-encoder scores using fast FNV hash.

func (sr *SemanticReranker) generateCacheKey(query, content string) string {

	h := fnv.New64a()

	h.Write([]byte(query))

	h.Write([]byte("|"))

	h.Write([]byte(content))

	return fmt.Sprintf("%016x", h.Sum64())

}

// calculateLexicalSimilarity calculates lexical similarity using keyword matching.

func (sr *SemanticReranker) calculateLexicalSimilarity(query, content string) float32 {

	queryWords := sr.tokenize(strings.ToLower(query))

	contentWords := sr.tokenize(strings.ToLower(content))

	if len(queryWords) == 0 {

		return 0.0

	}

	// Create word frequency maps.

	queryFreq := make(map[string]int)

	contentFreq := make(map[string]int)

	for _, word := range queryWords {

		queryFreq[word]++

	}

	for _, word := range contentWords {

		contentFreq[word]++

	}

	// Calculate cosine similarity.

	dotProduct := 0.0

	queryMagnitude := 0.0

	contentMagnitude := 0.0

	for word, qFreq := range queryFreq {

		cFreq := contentFreq[word]

		dotProduct += float64(qFreq * cFreq)

		queryMagnitude += float64(qFreq * qFreq)

	}

	for _, cFreq := range contentFreq {

		contentMagnitude += float64(cFreq * cFreq)

	}

	if queryMagnitude == 0 || contentMagnitude == 0 {

		return 0.0

	}

	similarity := dotProduct / (math.Sqrt(queryMagnitude) * math.Sqrt(contentMagnitude))

	return float32(similarity)

}

// tokenize splits text into meaningful tokens.

func (sr *SemanticReranker) tokenize(text string) []string {

	// Split on whitespace and punctuation.

	words := strings.FieldsFunc(text, func(c rune) bool {

		return !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9'))

	})

	// Filter out very short words and common stop words.

	stopWords := map[string]bool{

		"the": true, "and": true, "or": true, "but": true, "in": true,

		"on": true, "at": true, "to": true, "for": true, "of": true,

		"with": true, "by": true, "is": true, "are": true, "was": true,

		"were": true, "be": true, "been": true, "have": true, "has": true,

		"had": true, "do": true, "does": true, "did": true, "will": true,

		"would": true, "could": true, "should": true, "may": true, "might": true,

		"can": true, "must": true, "shall": true, "a": true, "an": true,
	}

	var filtered []string

	for _, word := range words {

		if len(word) > 2 && !stopWords[word] {

			filtered = append(filtered, word)

		}

	}

	return filtered

}

// calculateStructuralRelevance calculates relevance based on document structure.

func (sr *SemanticReranker) calculateStructuralRelevance(query string, result *EnhancedSearchResult) float32 {

	score := float32(0.0)

	if result.Document == nil {

		return score

	}

	doc := result.Document

	// Title relevance.

	if doc.Title != "" {

		titleSim := sr.calculateLexicalSimilarity(query, doc.Title)

		score += titleSim * 0.3 // Title is important

	}

	// Keywords relevance.

	if len(doc.Keywords) > 0 {

		keywordText := strings.Join(doc.Keywords, " ")

		keywordSim := sr.calculateLexicalSimilarity(query, keywordText)

		score += keywordSim * 0.2

	}

	// Source authority (higher weight for authoritative sources).

	authorityBoost := float32(0.0)

	switch doc.Source {

	case "3GPP":

		authorityBoost = 0.2

	case "O-RAN":

		authorityBoost = 0.15

	case "ETSI":

		authorityBoost = 0.1

	case "ITU":

		authorityBoost = 0.05

	}

	score += authorityBoost

	// Document type relevance.

	if doc.DocumentType != "" {

		switch strings.ToLower(doc.DocumentType) {

		case "specification", "standard":

			score += 0.1 // High value for specs

		case "technical report", "report":

			score += 0.05

		}

	}

	// Confidence score from document metadata.

	if doc.Confidence > 0 {

		score += float32(doc.Confidence) * 0.1

	}

	// Normalize score.

	if score > 1.0 {

		score = 1.0

	}

	return score

}

// calculateContextualRelevance calculates relevance based on context.

func (sr *SemanticReranker) calculateContextualRelevance(query string, result *EnhancedSearchResult) float32 {

	score := float32(0.0)

	if result.Document == nil {

		return score

	}

	doc := result.Document

	// Technology relevance.

	if len(doc.Technology) > 0 {

		for _, tech := range doc.Technology {

			if strings.Contains(strings.ToLower(query), strings.ToLower(tech)) {

				score += 0.15

			}

		}

	}

	// Network function relevance.

	if len(doc.NetworkFunction) > 0 {

		for _, nf := range doc.NetworkFunction {

			if strings.Contains(strings.ToLower(query), strings.ToLower(nf)) {

				score += 0.15

			}

		}

	}

	// Use case relevance.

	if doc.UseCase != "" {

		if strings.Contains(strings.ToLower(query), strings.ToLower(doc.UseCase)) {

			score += 0.1

		}

	}

	// Category relevance.

	if doc.Category != "" {

		if strings.Contains(strings.ToLower(query), strings.ToLower(doc.Category)) {

			score += 0.1

		}

	}

	// Recency boost.

	if !doc.Timestamp.IsZero() {

		age := float32(result.FreshnessScore)

		score += age * 0.1

	}

	// Normalize score.

	if score > 1.0 {

		score = 1.0

	}

	return score

}

// calculateCombinedScore calculates the final combined score.

func (sr *SemanticReranker) calculateCombinedScore(score *RerankingScore) float32 {

	// Define weights for different scoring components.

	weights := map[string]float32{

		"semantic": 0.3,

		"cross": 0.25,

		"lexical": 0.2,

		"structural": 0.15,

		"contextual": 0.1,
	}

	combined := score.SemanticSimilarity*weights["semantic"] +

		score.CrossEncoderScore*weights["cross"] +

		score.LexicalSimilarity*weights["lexical"] +

		score.StructuralRelevance*weights["structural"] +

		score.ContextualRelevance*weights["contextual"]

	return combined

}

// generateScoreExplanation generates a human-readable explanation of the score.

func (sr *SemanticReranker) generateScoreExplanation(score *RerankingScore, result *EnhancedSearchResult) string {

	var explanations []string

	// Identify the strongest scoring components.

	if score.SemanticSimilarity > 0.7 {

		explanations = append(explanations, "high semantic similarity")

	}

	if score.CrossEncoderScore > 0.7 {

		explanations = append(explanations, "strong cross-encoder relevance")

	}

	if score.LexicalSimilarity > 0.7 {

		explanations = append(explanations, "excellent keyword match")

	}

	if score.StructuralRelevance > 0.7 {

		explanations = append(explanations, "authoritative source")

	}

	if score.ContextualRelevance > 0.7 {

		explanations = append(explanations, "contextually relevant")

	}

	// Add document-specific information.

	if result.Document != nil {

		if result.Document.Source != "" && result.Document.Source != "Unknown" {

			explanations = append(explanations, fmt.Sprintf("from %s", result.Document.Source))

		}

		if len(result.Document.Technology) > 0 {

			explanations = append(explanations, fmt.Sprintf("covers %s", strings.Join(result.Document.Technology, ", ")))

		}

	}

	if len(explanations) == 0 {

		return "moderate relevance"

	}

	return strings.Join(explanations, "; ")

}

// CrossEncoder methods.

// getCachedScore retrieves a cached cross-encoder score.

func (ce *CrossEncoder) getCachedScore(key string) (float32, bool) {

	ce.cacheMutex.RLock()

	defer ce.cacheMutex.RUnlock()

	score, exists := ce.scoreCache[key]

	return score, exists

}

// setCachedScore stores a cross-encoder score in cache.

func (ce *CrossEncoder) setCachedScore(key string, score float32) {

	ce.cacheMutex.Lock()

	defer ce.cacheMutex.Unlock()

	// Limit cache size.

	if len(ce.scoreCache) > 10000 {

		// Simple cache eviction - remove 25% of entries.

		count := 0

		for k := range ce.scoreCache {

			delete(ce.scoreCache, k)

			count++

			if count > 2500 {

				break

			}

		}

	}

	ce.scoreCache[key] = score

}

// clearCache clears the cross-encoder cache.

func (ce *CrossEncoder) clearCache() {

	ce.cacheMutex.Lock()

	defer ce.cacheMutex.Unlock()

	ce.scoreCache = make(map[string]float32)

}
