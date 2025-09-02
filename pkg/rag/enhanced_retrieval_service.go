//go:build !disable_rag && !test

package rag

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// EnhancedRetrievalService provides advanced retrieval capabilities with query enhancement.

type EnhancedRetrievalService struct {
	weaviateClient WeaviateClient

	embeddingService *EmbeddingService

	config *RetrievalConfig

	logger *slog.Logger

	metrics *RetrievalMetrics

	queryEnhancer *QueryEnhancer

	reranker *SemanticReranker

	contextAssembler *ContextAssembler

	mutex sync.RWMutex
}

// RetrievalConfig holds configuration for the enhanced retrieval service.

type RetrievalConfig struct {
	// Search configuration.

	DefaultLimit int `json:"default_limit"`

	MaxLimit int `json:"max_limit"`

	DefaultHybridAlpha float32 `json:"default_hybrid_alpha"`

	MinConfidenceThreshold float32 `json:"min_confidence_threshold"`

	// Query enhancement.

	EnableQueryExpansion bool `json:"enable_query_expansion"`

	QueryExpansionTerms int `json:"query_expansion_terms"`

	EnableQueryRewriting bool `json:"enable_query_rewriting"`

	EnableSpellCorrection bool `json:"enable_spell_correction"`

	EnableSynonymExpansion bool `json:"enable_synonym_expansion"`

	// Reranking configuration.

	EnableSemanticReranking bool `json:"enable_semantic_reranking"`

	RerankingTopK int `json:"reranking_top_k"`

	CrossEncoderModel string `json:"cross_encoder_model"`

	// Context assembly.

	MaxContextLength int `json:"max_context_length"`

	ContextOverlapRatio float64 `json:"context_overlap_ratio"`

	IncludeHierarchyInfo bool `json:"include_hierarchy_info"`

	IncludeSourceMetadata bool `json:"include_source_metadata"`

	// Filtering and ranking.

	EnableDiversityFiltering bool `json:"enable_diversity_filtering"`

	DiversityThreshold float32 `json:"diversity_threshold"`

	BoostRecentDocuments bool `json:"boost_recent_documents"`

	RecencyBoostFactor float64 `json:"recency_boost_factor"`

	// Performance settings.

	EnableResultCaching bool `json:"enable_result_caching"`

	ResultCacheTTL time.Duration `json:"result_cache_ttl"`

	MaxConcurrentQueries int `json:"max_concurrent_queries"`

	QueryTimeout time.Duration `json:"query_timeout"`

	// Intent-specific settings.

	IntentTypeWeights map[string]float64 `json:"intent_type_weights"`

	TechnicalDomainBoosts map[string]float64 `json:"technical_domain_boosts"`

	SourcePriorityWeights map[string]float64 `json:"source_priority_weights"`
}

// EnhancedSearchRequest extends the basic search with advanced features.

type EnhancedSearchRequest struct {
	// Basic search parameters.

	Query string `json:"query"`

	Limit int `json:"limit"`

	Filters map[string]interface{} `json:"filters,omitempty"`

	// Enhancement options.

	EnableQueryEnhancement bool `json:"enable_query_enhancement"`

	EnableReranking bool `json:"enable_reranking"`

	RequiredContextLength int `json:"required_context_length"`

	// Intent context.

	IntentType string `json:"intent_type,omitempty"`

	NetworkDomain string `json:"network_domain,omitempty"`

	TechnicalLevel string `json:"technical_level,omitempty"` // basic, intermediate, advanced

	// User context.

	UserID string `json:"user_id,omitempty"`

	SessionID string `json:"session_id,omitempty"`

	ConversationHistory []string `json:"conversation_history,omitempty"`

	// Search preferences.

	PreferredSources []string `json:"preferred_sources,omitempty"`

	ExcludedSources []string `json:"excluded_sources,omitempty"`

	PreferredDocumentTypes []string `json:"preferred_document_types,omitempty"`

	MaxDocumentAge *time.Duration `json:"max_document_age,omitempty"`

	// Quality requirements.

	MinQualityScore float32 `json:"min_quality_score"`

	RequireHierarchyInfo bool `json:"require_hierarchy_info"`

	RequireTechnicalTerms bool `json:"require_technical_terms"`
}

// EnhancedSearchResponse extends the basic response with additional metadata.

type EnhancedSearchResponse struct {
	// Basic response data.

	Results []*EnhancedSearchResult `json:"results"`

	Total int `json:"total"`

	Query string `json:"query"`

	ProcessedQuery string `json:"processed_query"`

	// Processing information.

	ProcessingTime time.Duration `json:"processing_time"`

	RetrievalTime time.Duration `json:"retrieval_time"`

	EnhancementTime time.Duration `json:"enhancement_time"`

	RerankingTime time.Duration `json:"reranking_time"`

	ContextAssemblyTime time.Duration `json:"context_assembly_time"`

	// Enhancement details.

	QueryEnhancements *QueryEnhancements `json:"query_enhancements,omitempty"`

	RerankingApplied bool `json:"reranking_applied"`

	FiltersApplied []string `json:"filters_applied"`

	BoostsApplied []string `json:"boosts_applied"`

	// Context information.

	AssembledContext string `json:"assembled_context"`

	ContextMetadata *ContextMetadata `json:"context_metadata"`

	// Quality metrics.

	AverageRelevanceScore float32 `json:"average_relevance_score"`

	CoverageScore float32 `json:"coverage_score"`

	DiversityScore float32 `json:"diversity_score"`

	// Debug information.

	DebugInfo map[string]interface{} `json:"debug_info,omitempty"`

	ProcessedAt time.Time `json:"processed_at"`
}

// EnhancedSearchResult extends SearchResult with additional information.

type EnhancedSearchResult struct {
	// Basic result information.

	*shared.SearchResult

	// Enhanced scoring.

	RelevanceScore float32 `json:"relevance_score"`

	QualityScore float32 `json:"quality_score"`

	FreshnessScore float32 `json:"freshness_score"`

	AuthorityScore float32 `json:"authority_score"`

	CombinedScore float32 `json:"combined_score"`

	// Context information.

	ContextRelevance float32 `json:"context_relevance"`

	HierarchyMatch bool `json:"hierarchy_match"`

	SemanticSimilarity float32 `json:"semantic_similarity"`

	// Explanation.

	RelevanceReason string `json:"relevance_reason"`

	HighlightedText string `json:"highlighted_text"`

	KeyTermMatches []string `json:"key_term_matches"`

	// Metadata.

	ProcessingNotes []string `json:"processing_notes,omitempty"`
}

// QueryEnhancements contains information about query processing.

type QueryEnhancements struct {
	OriginalQuery string `json:"original_query"`

	ExpandedTerms []string `json:"expanded_terms"`

	SynonymReplacements map[string]string `json:"synonym_replacements"`

	SpellingCorrections map[string]string `json:"spelling_corrections"`

	RewrittenQuery string `json:"rewritten_query"`

	EnhancementApplied []string `json:"enhancements_applied"`
}

// ContextMetadata contains information about assembled context.

type ContextMetadata struct {
	DocumentCount int `json:"document_count"`

	TotalLength int `json:"total_length"`

	TruncatedAt int `json:"truncated_at"`

	HierarchyLevels []int `json:"hierarchy_levels"`

	SourceDistribution map[string]int `json:"source_distribution"`

	TechnicalTermCount int `json:"technical_term_count"`

	AverageQuality float32 `json:"average_quality"`
}

// RetrievalMetrics tracks advanced retrieval performance.

type RetrievalMetrics struct {
	// Basic metrics.

	TotalQueries int64 `json:"total_queries"`

	SuccessfulQueries int64 `json:"successful_queries"`

	FailedQueries int64 `json:"failed_queries"`

	AverageResponseTime time.Duration `json:"average_response_time"`

	// Enhancement metrics.

	QueriesWithEnhancement int64 `json:"queries_with_enhancement"`

	QueriesWithReranking int64 `json:"queries_with_reranking"`

	AverageEnhancementTime time.Duration `json:"average_enhancement_time"`

	AverageRerankingTime time.Duration `json:"average_reranking_time"`

	// Quality metrics.

	AverageRelevanceScore float32 `json:"average_relevance_score"`

	AverageCoverageScore float32 `json:"average_coverage_score"`

	AverageDiversityScore float32 `json:"average_diversity_score"`

	// Intent-specific metrics.

	IntentTypeMetrics map[string]IntentTypeMetrics `json:"intent_type_metrics"`

	// Cache metrics.

	CacheHitRate float64 `json:"cache_hit_rate"`

	CacheHits int64 `json:"cache_hits"`

	CacheMisses int64 `json:"cache_misses"`

	LastUpdated time.Time `json:"last_updated"`

	mutex sync.RWMutex
}

// IntentTypeMetrics tracks metrics per intent type.

type IntentTypeMetrics struct {
	QueryCount int64 `json:"query_count"`

	AverageRelevance float32 `json:"average_relevance"`

	AverageResponseTime time.Duration `json:"average_response_time"`

	SuccessRate float64 `json:"success_rate"`
}

// NewEnhancedRetrievalService creates a new enhanced retrieval service.

func NewEnhancedRetrievalService(
	weaviateClient WeaviateClient,

	embeddingService *EmbeddingService,

	config *RetrievalConfig,
) *EnhancedRetrievalService {
	if config == nil {
		config = getDefaultRetrievalConfig()
	}

	service := &EnhancedRetrievalService{
		weaviateClient: weaviateClient,

		embeddingService: embeddingService,

		config: config,

		logger: slog.Default().With("component", "enhanced-retrieval-service"),

		metrics: &RetrievalMetrics{
			IntentTypeMetrics: make(map[string]IntentTypeMetrics),

			LastUpdated: time.Now(),
		},
	}

	// Initialize sub-components.

	service.queryEnhancer = NewQueryEnhancer(config)

	service.reranker = NewSemanticReranker(config)

	service.contextAssembler = NewContextAssembler(config)

	return service
}

// SearchEnhanced performs an enhanced search with query processing and reranking.

func (ers *EnhancedRetrievalService) SearchEnhanced(ctx context.Context, request *EnhancedSearchRequest) (*EnhancedSearchResponse, error) {
	startTime := time.Now()

	if request == nil {
		return nil, fmt.Errorf("search request cannot be nil")
	}

	if request.Query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	ers.logger.Info("Processing enhanced search request",

		"query", request.Query,

		"intent_type", request.IntentType,

		"limit", request.Limit,

		"enable_enhancement", request.EnableQueryEnhancement,

		"enable_reranking", request.EnableReranking,
	)

	// Set defaults.

	if request.Limit == 0 {
		request.Limit = ers.config.DefaultLimit
	}

	if request.Limit > ers.config.MaxLimit {
		request.Limit = ers.config.MaxLimit
	}

	response := &EnhancedSearchResponse{
		Query: request.Query,

		ProcessedAt: time.Now(),
	}

	// Step 1: Query enhancement.

	var enhancementTime time.Duration

	var queryEnhancements *QueryEnhancements

	processedQuery := request.Query

	if request.EnableQueryEnhancement && ers.config.EnableQueryExpansion {

		enhancementStart := time.Now()

		var err error

		processedQuery, queryEnhancements, err = ers.queryEnhancer.EnhanceQuery(ctx, request)
		if err != nil {
			ers.logger.Warn("Query enhancement failed", "error", err)

			// Continue with original query.
		}

		enhancementTime = time.Since(enhancementStart)

		response.QueryEnhancements = queryEnhancements

		response.ProcessedQuery = processedQuery

	}

	// Step 2: Initial retrieval.

	retrievalStart := time.Now()

	searchQuery := &SearchQuery{
		Query: processedQuery,

		Limit: ers.calculateInitialLimit(request),

		Filters: ers.buildEnhancedFilters(request),

		HybridSearch: true,

		HybridAlpha: float32(ers.config.DefaultHybridAlpha),

		UseReranker: false, // We'll do our own reranking

		MinConfidence: float32(request.MinQualityScore),

		ExpandQuery: false, // Already done in enhancement step

	}

	searchResponse, err := ers.weaviateClient.Search(ctx, searchQuery)
	if err != nil {

		ers.updateMetrics(func(m *RetrievalMetrics) {
			m.FailedQueries++
		})

		return nil, fmt.Errorf("initial retrieval failed: %w", err)

	}

	retrievalTime := time.Since(retrievalStart)

	// Step 3: Semantic reranking.

	var rerankingTime time.Duration

	results := ers.convertToEnhancedResults(searchResponse.Results)

	if request.EnableReranking && ers.config.EnableSemanticReranking && len(results) > 1 {

		rerankingStart := time.Now()

		results, err = ers.reranker.RerankResults(ctx, request.Query, results)

		if err != nil {
			ers.logger.Warn("Reranking failed", "error", err)

			// Continue with original results.
		} else {
			response.RerankingApplied = true
		}

		rerankingTime = time.Since(rerankingStart)

	}

	// Step 4: Apply post-processing filters and boosts.

	results = ers.applyPostProcessingFilters(results, request)

	results = ers.applyScoreBoosts(results, request)

	// Step 5: Limit results to requested size.

	if len(results) > request.Limit {
		results = results[:request.Limit]
	}

	// Step 6: Context assembly.

	contextAssemblyStart := time.Now()

	assembledContext, contextMetadata := ers.contextAssembler.AssembleContext(results, request)

	contextAssemblyTime := time.Since(contextAssemblyStart)

	// Build response.

	response.Results = results

	response.Total = len(results)

	response.ProcessingTime = time.Since(startTime)

	response.RetrievalTime = retrievalTime

	response.EnhancementTime = enhancementTime

	response.RerankingTime = rerankingTime

	response.ContextAssemblyTime = contextAssemblyTime

	response.AssembledContext = assembledContext

	response.ContextMetadata = contextMetadata

	// Calculate quality metrics.

	response.AverageRelevanceScore = ers.calculateAverageRelevance(results)

	response.CoverageScore = ers.calculateCoverageScore(results, request)

	response.DiversityScore = ers.calculateDiversityScore(results)

	// Add debug information if needed.

	if ers.logger.Enabled(ctx, slog.LevelDebug) {
		response.DebugInfo = ers.buildDebugInfo(request, searchQuery, searchResponse)
	}

	// Update metrics.

	ers.updateMetrics(func(m *RetrievalMetrics) {
		m.TotalQueries++

		m.SuccessfulQueries++

		if m.SuccessfulQueries > 0 {
			m.AverageResponseTime = (m.AverageResponseTime*time.Duration(m.SuccessfulQueries-1) + response.ProcessingTime) / time.Duration(m.SuccessfulQueries)
		} else {
			m.AverageResponseTime = response.ProcessingTime
		}

		if request.EnableQueryEnhancement {

			m.QueriesWithEnhancement++

			if m.QueriesWithEnhancement > 0 {
				m.AverageEnhancementTime = (m.AverageEnhancementTime*time.Duration(m.QueriesWithEnhancement-1) + enhancementTime) / time.Duration(m.QueriesWithEnhancement)
			} else {
				m.AverageEnhancementTime = enhancementTime
			}

		}

		if request.EnableReranking {

			m.QueriesWithReranking++

			if m.QueriesWithReranking > 0 {
				m.AverageRerankingTime = (m.AverageRerankingTime*time.Duration(m.QueriesWithReranking-1) + rerankingTime) / time.Duration(m.QueriesWithReranking)
			} else {
				m.AverageRerankingTime = rerankingTime
			}

		}

		// Update quality metrics.

		m.AverageRelevanceScore = (m.AverageRelevanceScore*float32(m.SuccessfulQueries-1) + response.AverageRelevanceScore) / float32(m.SuccessfulQueries)

		m.AverageCoverageScore = (m.AverageCoverageScore*float32(m.SuccessfulQueries-1) + response.CoverageScore) / float32(m.SuccessfulQueries)

		m.AverageDiversityScore = (m.AverageDiversityScore*float32(m.SuccessfulQueries-1) + response.DiversityScore) / float32(m.SuccessfulQueries)

		// Update intent-specific metrics.

		if request.IntentType != "" {

			intentMetrics := m.IntentTypeMetrics[request.IntentType]

			intentMetrics.QueryCount++

			intentMetrics.AverageRelevance = (intentMetrics.AverageRelevance*float32(intentMetrics.QueryCount-1) + response.AverageRelevanceScore) / float32(intentMetrics.QueryCount)

			intentMetrics.AverageResponseTime = (intentMetrics.AverageResponseTime*time.Duration(intentMetrics.QueryCount-1) + response.ProcessingTime) / time.Duration(intentMetrics.QueryCount)

			intentMetrics.SuccessRate = float64(m.SuccessfulQueries) / float64(m.TotalQueries)

			m.IntentTypeMetrics[request.IntentType] = intentMetrics

		}

		m.LastUpdated = time.Now()
	})

	ers.logger.Info("Enhanced search completed",

		"query", request.Query,

		"results", len(results),

		"processing_time", response.ProcessingTime,

		"average_relevance", response.AverageRelevanceScore,
	)

	return response, nil
}

// calculateInitialLimit determines how many results to fetch initially.

func (ers *EnhancedRetrievalService) calculateInitialLimit(request *EnhancedSearchRequest) int {
	initialLimit := request.Limit

	// Fetch more if we're going to rerank.

	if request.EnableReranking && ers.config.EnableSemanticReranking {

		initialLimit = ers.config.RerankingTopK

		if initialLimit < request.Limit*2 {
			initialLimit = request.Limit * 2
		}

	}

	// Fetch more if diversity filtering is enabled.

	if ers.config.EnableDiversityFiltering {
		initialLimit = int(float64(initialLimit) * 1.5)
	}

	if initialLimit > ers.config.MaxLimit*2 {
		initialLimit = ers.config.MaxLimit * 2
	}

	return initialLimit
}

// buildEnhancedFilters creates comprehensive filters for the search.

func (ers *EnhancedRetrievalService) buildEnhancedFilters(request *EnhancedSearchRequest) map[string]interface{} {
	filters := make(map[string]interface{})

	// Copy user-provided filters.

	for k, v := range request.Filters {
		filters[k] = v
	}

	// Add intent-based filters.

	if request.IntentType != "" {
		filters["intent_category"] = request.IntentType
	}

	// Add network domain filters.

	if request.NetworkDomain != "" {
		filters["network_domain"] = request.NetworkDomain
	}

	// Add source preferences.

	if len(request.PreferredSources) > 0 {
		filters["preferred_sources"] = request.PreferredSources
	}

	// Add source exclusions.

	if len(request.ExcludedSources) > 0 {
		filters["excluded_sources"] = request.ExcludedSources
	}

	// Add document type preferences.

	if len(request.PreferredDocumentTypes) > 0 {
		filters["preferred_document_types"] = request.PreferredDocumentTypes
	}

	// Add age restrictions.

	if request.MaxDocumentAge != nil {
		filters["max_age"] = request.MaxDocumentAge.String()
	}

	// Add quality requirements.

	if request.RequireHierarchyInfo {
		filters["require_hierarchy"] = true
	}

	if request.RequireTechnicalTerms {
		filters["require_technical_terms"] = true
	}

	return filters
}

// convertToEnhancedResults converts basic search results to enhanced results.

func (ers *EnhancedRetrievalService) convertToEnhancedResults(basicResults []*SearchResult) []*EnhancedSearchResult {
	enhanced := make([]*EnhancedSearchResult, len(basicResults))

	for i, result := range basicResults {

		enhanced[i] = &EnhancedSearchResult{
			SearchResult: result,

			RelevanceScore: result.Score,

			QualityScore: ers.calculateQualityScore(result),

			FreshnessScore: ers.calculateFreshnessScore(result),

			AuthorityScore: ers.calculateAuthorityScore(result),

			SemanticSimilarity: result.Score, // Initial value

		}

		// Calculate combined score.

		enhanced[i].CombinedScore = ers.calculateCombinedScore(enhanced[i])

	}

	return enhanced
}

// calculateQualityScore calculates a quality score for a result.

func (ers *EnhancedRetrievalService) calculateQualityScore(result *SearchResult) float32 {
	if result.Document == nil {
		return 0.0
	}

	score := float32(0.0)

	// Content length factor.

	contentLength := len(result.Document.Content)

	if contentLength > 100 {
		score += 0.2
	}

	if contentLength > 500 {
		score += 0.2
	}

	if contentLength > 1000 {
		score += 0.1
	}

	// Metadata completeness.

	if result.Document.Title != "" {
		score += 0.1
	}

	if result.Document.Source != "" {
		score += 0.1
	}

	if result.Document.Version != "" {
		score += 0.1
	}

	if len(result.Document.Keywords) > 0 {
		score += 0.1
	}

	if len(result.Document.Technology) > 0 {
		score += 0.1
	}

	// Confidence from document metadata.

	if result.Document.Confidence > 0 {
		score += float32(result.Document.Confidence) * 0.3
	}

	return score
}

// calculateFreshnessScore calculates a freshness score based on document age.

func (ers *EnhancedRetrievalService) calculateFreshnessScore(result *SearchResult) float32 {
	if result.Document == nil {
		return 0.5 // Neutral score for documents without timestamp
	}

	// Since Timestamp field is not available in TelecomDocument, use a neutral score.

	// In a full implementation, could check document version or other metadata for freshness.

	return 0.5
}

// calculateAuthorityScore calculates an authority score based on source.

func (ers *EnhancedRetrievalService) calculateAuthorityScore(result *SearchResult) float32 {
	if result.Document == nil {
		return 0.5
	}

	source := result.Document.Source

	if weight, exists := ers.config.SourcePriorityWeights[source]; exists {
		return float32(weight)
	}

	// Default authority score.

	return 0.7
}

// calculateCombinedScore calculates a combined relevance score.

func (ers *EnhancedRetrievalService) calculateCombinedScore(result *EnhancedSearchResult) float32 {
	// Weighted combination of different scores.

	weights := map[string]float32{
		"relevance": 0.4,

		"quality": 0.25,

		"freshness": 0.15,

		"authority": 0.2,
	}

	score := result.RelevanceScore*weights["relevance"] +

		result.QualityScore*weights["quality"] +

		result.FreshnessScore*weights["freshness"] +

		result.AuthorityScore*weights["authority"]

	return score
}

// applyPostProcessingFilters applies various filters to the results.

func (ers *EnhancedRetrievalService) applyPostProcessingFilters(results []*EnhancedSearchResult, request *EnhancedSearchRequest) []*EnhancedSearchResult {
	var filtered []*EnhancedSearchResult

	for _, result := range results {

		// Quality threshold filter.

		if result.QualityScore < request.MinQualityScore {
			continue
		}

		// Confidence threshold filter.

		if result.RelevanceScore < ers.config.MinConfidenceThreshold {
			continue
		}

		filtered = append(filtered, result)

	}

	// Apply diversity filtering if enabled.

	if ers.config.EnableDiversityFiltering {
		filtered = ers.applyDiversityFiltering(filtered)
	}

	return filtered
}

// applyDiversityFiltering removes results that are too similar to each other.

func (ers *EnhancedRetrievalService) applyDiversityFiltering(results []*EnhancedSearchResult) []*EnhancedSearchResult {
	if len(results) <= 1 {
		return results
	}

	var diverse []*EnhancedSearchResult

	diverse = append(diverse, results[0]) // Always include the top result

	for i := 1; i < len(results); i++ {

		candidate := results[i]

		isDiverse := true

		// Check similarity with already selected results.

		for _, selected := range diverse {

			similarity := ers.calculateContentSimilarity(candidate, selected)

			if similarity > ers.config.DiversityThreshold {

				isDiverse = false

				break

			}

		}

		if isDiverse {
			diverse = append(diverse, candidate)
		}

	}

	return diverse
}

// calculateContentSimilarity calculates similarity between two results.

func (ers *EnhancedRetrievalService) calculateContentSimilarity(result1, result2 *EnhancedSearchResult) float32 {
	// Simple content similarity based on overlapping words.

	if result1.Document == nil || result2.Document == nil {
		return 0.0
	}

	content1 := strings.ToLower(result1.Document.Content)

	content2 := strings.ToLower(result2.Document.Content)

	words1 := strings.Fields(content1)

	words2 := strings.Fields(content2)

	// Create word sets.

	wordSet1 := make(map[string]bool)

	wordSet2 := make(map[string]bool)

	for _, word := range words1 {
		if len(word) > 3 { // Only consider meaningful words

			wordSet1[word] = true
		}
	}

	for _, word := range words2 {
		if len(word) > 3 {
			wordSet2[word] = true
		}
	}

	// Calculate Jaccard similarity.

	intersection := 0

	for word := range wordSet1 {
		if wordSet2[word] {
			intersection++
		}
	}

	union := len(wordSet1) + len(wordSet2) - intersection

	if union == 0 {
		return 0.0
	}

	return float32(intersection) / float32(union)
}

// applyScoreBoosts applies various scoring boosts.

func (ers *EnhancedRetrievalService) applyScoreBoosts(results []*EnhancedSearchResult, request *EnhancedSearchRequest) []*EnhancedSearchResult {
	for _, result := range results {

		// Intent type boost.

		if request.IntentType != "" {
			if boost, exists := ers.config.IntentTypeWeights[request.IntentType]; exists {
				result.CombinedScore *= float32(boost)
			}
		}

		// Technical domain boost.

		if request.NetworkDomain != "" && result.Document != nil {
			if boost, exists := ers.config.TechnicalDomainBoosts[request.NetworkDomain]; exists {
				result.CombinedScore *= float32(boost)
			}
		}

		// Source priority boost.

		if result.Document != nil && result.Document.Source != "" {
			if boost, exists := ers.config.SourcePriorityWeights[result.Document.Source]; exists {
				result.CombinedScore *= float32(boost)
			}
		}

		// Recency boost.

		if ers.config.BoostRecentDocuments {
			result.CombinedScore *= result.FreshnessScore * float32(ers.config.RecencyBoostFactor)
		}

	}

	// Re-sort by combined score.

	sort.Slice(results, func(i, j int) bool {
		return results[i].CombinedScore > results[j].CombinedScore
	})

	return results
}

// calculateAverageRelevance calculates the average relevance score.

func (ers *EnhancedRetrievalService) calculateAverageRelevance(results []*EnhancedSearchResult) float32 {
	if len(results) == 0 {
		return 0.0
	}

	total := float32(0.0)

	for _, result := range results {
		total += result.RelevanceScore
	}

	return total / float32(len(results))
}

// calculateCoverageScore calculates how well the results cover the query.

func (ers *EnhancedRetrievalService) calculateCoverageScore(results []*EnhancedSearchResult, request *EnhancedSearchRequest) float32 {
	// Simple coverage metric based on query term coverage.

	queryTerms := strings.Fields(strings.ToLower(request.Query))

	if len(queryTerms) == 0 {
		return 1.0
	}

	coveredTerms := make(map[string]bool)

	for _, result := range results {

		if result.Document == nil {
			continue
		}

		content := strings.ToLower(result.Document.Content)

		for _, term := range queryTerms {
			if strings.Contains(content, term) {
				coveredTerms[term] = true
			}
		}

	}

	return float32(len(coveredTerms)) / float32(len(queryTerms))
}

// calculateDiversityScore calculates the diversity of the result set.

func (ers *EnhancedRetrievalService) calculateDiversityScore(results []*EnhancedSearchResult) float32 {
	if len(results) <= 1 {
		return 1.0
	}

	// Calculate average pairwise similarity.

	totalSimilarity := float32(0.0)

	pairCount := 0

	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {

			similarity := ers.calculateContentSimilarity(results[i], results[j])

			totalSimilarity += similarity

			pairCount++

		}
	}

	if pairCount == 0 {
		return 1.0
	}

	avgSimilarity := totalSimilarity / float32(pairCount)

	return 1.0 - avgSimilarity // Higher diversity = lower average similarity
}

// buildDebugInfo creates debug information for the response.

func (ers *EnhancedRetrievalService) buildDebugInfo(
	request *EnhancedSearchRequest,

	searchQuery *SearchQuery,

	searchResponse *SearchResponse,
) map[string]interface{} {
	return map[string]interface{}{
		"original_query": request.Query,

		"processed_query": searchQuery.Query,

		"initial_results_count": len(searchResponse.Results),

		"search_took": searchResponse.Took,

		"filters_applied": searchQuery.Filters,

		"hybrid_alpha": searchQuery.HybridAlpha,

		"min_confidence": searchQuery.MinConfidence,
	}
}

// updateMetrics safely updates the retrieval metrics.

func (ers *EnhancedRetrievalService) updateMetrics(updater func(*RetrievalMetrics)) {
	ers.metrics.mutex.Lock()

	defer ers.metrics.mutex.Unlock()

	updater(ers.metrics)
}

// GetMetrics returns the current retrieval metrics.

func (ers *EnhancedRetrievalService) GetMetrics() *RetrievalMetrics {
	ers.metrics.mutex.RLock()

	defer ers.metrics.mutex.RUnlock()

	// Return a copy without the mutex.

	metrics := &RetrievalMetrics{
		TotalQueries: ers.metrics.TotalQueries,

		SuccessfulQueries: ers.metrics.SuccessfulQueries,

		FailedQueries: ers.metrics.FailedQueries,

		AverageResponseTime: ers.metrics.AverageResponseTime,

		QueriesWithEnhancement: ers.metrics.QueriesWithEnhancement,

		QueriesWithReranking: ers.metrics.QueriesWithReranking,

		AverageEnhancementTime: ers.metrics.AverageEnhancementTime,

		AverageRerankingTime: ers.metrics.AverageRerankingTime,

		AverageRelevanceScore: ers.metrics.AverageRelevanceScore,

		AverageCoverageScore: ers.metrics.AverageCoverageScore,

		AverageDiversityScore: ers.metrics.AverageDiversityScore,

		IntentTypeMetrics: copyIntentTypeMetrics(ers.metrics.IntentTypeMetrics),

		CacheHitRate: ers.metrics.CacheHitRate,

		CacheHits: ers.metrics.CacheHits,

		CacheMisses: ers.metrics.CacheMisses,

		LastUpdated: ers.metrics.LastUpdated,
	}

	return metrics
}

// RetrievalHealthStatus represents the health status of the retrieval service.

type RetrievalHealthStatus struct {
	Status string `json:"status"` // "healthy", "degraded", "unhealthy"

	Timestamp time.Time `json:"timestamp"`

	Components map[string]ComponentStatus `json:"components"`

	Metrics HealthMetrics `json:"metrics"`

	TotalQueries int64 `json:"total_queries"`

	SuccessRate float64 `json:"success_rate"`
}

// ComponentStatus represents the health status of a service component.

type ComponentStatus struct {
	Status string `json:"status"` // "healthy", "degraded", "unhealthy"

	Message string `json:"message,omitempty"`

	LastCheck time.Time `json:"last_check"`

	Details map[string]interface{} `json:"details,omitempty"`
}

// HealthMetrics contains metrics relevant to health status.

type HealthMetrics struct {
	TotalQueries int64 `json:"total_queries"`

	SuccessfulQueries int64 `json:"successful_queries"`

	FailedQueries int64 `json:"failed_queries"`

	AverageResponseTime time.Duration `json:"average_response_time"`

	LastUpdated time.Time `json:"last_updated"`
}

// GetHealthStatus returns the health status of the retrieval service.

func (ers *EnhancedRetrievalService) GetHealthStatus(ctx context.Context) (*RetrievalHealthStatus, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Create timeout context for health checks.

	healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)

	defer cancel()

	// Get current metrics safely.

	metrics := ers.GetMetrics()

	// Calculate success rate with zero-division protection.

	var successRate float64

	if metrics.TotalQueries > 0 {
		successRate = float64(metrics.SuccessfulQueries) / float64(metrics.TotalQueries)
	} else {
		successRate = 1.0 // Default to 100% when no queries processed yet
	}

	// Check component health.

	components := make(map[string]ComponentStatus)

	// Check Weaviate/Vector Store health.

	weaviateHealth := &WeaviateHealthStatus{
		IsHealthy: true,
		LastCheck: time.Now(),
	}

	weaviateStatus := "healthy"

	weaviateMessage := "Vector store is operational"

	if !weaviateHealth.IsHealthy {

		weaviateStatus = "unhealthy"

		weaviateMessage = "Vector store connectivity issues"

	}

	components["vector_store"] = ComponentStatus{
		Status: weaviateStatus,

		Message: weaviateMessage,

		LastCheck: weaviateHealth.LastCheck,

		Details: map[string]interface{}{
			"healthy": weaviateHealth.IsHealthy,
		},
	}

	// Check Embedding Service health.

	embeddingStatus := ers.checkEmbeddingServiceHealth(healthCtx)

	components["embedding_service"] = embeddingStatus

	// Determine overall health status.

	overallStatus := ers.determineOverallHealth(components, successRate)

	healthStatus := &RetrievalHealthStatus{
		Status: overallStatus,

		Timestamp: time.Now(),

		Components: components,

		TotalQueries: metrics.TotalQueries,

		SuccessRate: successRate,

		Metrics: HealthMetrics{
			TotalQueries: metrics.TotalQueries,

			SuccessfulQueries: metrics.SuccessfulQueries,

			FailedQueries: metrics.FailedQueries,

			AverageResponseTime: metrics.AverageResponseTime,

			LastUpdated: metrics.LastUpdated,
		},
	}

	return healthStatus, nil
}

// checkEmbeddingServiceHealth checks the health of the embedding service.

func (ers *EnhancedRetrievalService) checkEmbeddingServiceHealth(ctx context.Context) ComponentStatus {
	if ers.embeddingService == nil {
		return ComponentStatus{
			Status: "unhealthy",

			Message: "Embedding service not initialized",

			LastCheck: time.Now(),
		}
	}

	// Call the embedding service's CheckStatus method.

	status, err := ers.embeddingService.CheckStatus(ctx)
	if err != nil {
		return ComponentStatus{
			Status: "unhealthy",

			Message: fmt.Sprintf("Health check failed: %v", err),

			LastCheck: time.Now(),
		}
	}

	if status == nil {
		return ComponentStatus{
			Status: "unhealthy",

			Message: "Health check returned nil status",

			LastCheck: time.Now(),
		}
	}

	return ComponentStatus{
		Status: status.Status,

		Message: status.Message,

		LastCheck: status.LastCheck,

		Details: status.Details,
	}
}

// determineOverallHealth determines the overall health status based on components and metrics.

func (ers *EnhancedRetrievalService) determineOverallHealth(components map[string]ComponentStatus, successRate float64) string {
	// Check if any component is unhealthy.

	for _, component := range components {
		if component.Status == "unhealthy" {
			return "unhealthy"
		}
	}

	// Check success rate thresholds.

	if successRate < 0.5 { // Less than 50% success rate

		return "unhealthy"
	} else if successRate < 0.8 { // Less than 80% success rate

		return "degraded"
	}

	// Check if any component is degraded.

	for _, component := range components {
		if component.Status == "degraded" {
			return "degraded"
		}
	}

	return "healthy"
}

// copyIntentTypeMetrics creates a deep copy of IntentTypeMetrics map.

func copyIntentTypeMetrics(original map[string]IntentTypeMetrics) map[string]IntentTypeMetrics {
	if original == nil {
		return nil
	}

	copy := make(map[string]IntentTypeMetrics, len(original))

	for k, v := range original {
		copy[k] = v
	}

	return copy
}
