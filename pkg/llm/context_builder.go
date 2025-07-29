package llm

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/rag"
)

// ContextBuilder manages context injection and assembly for RAG-enhanced LLM processing
type ContextBuilder struct {
	tokenManager    *TokenManager
	relevanceScorer *RelevanceScorer
	config          *ContextBuilderConfig
	logger          *slog.Logger
	metrics         *ContextMetrics
	mutex           sync.RWMutex
}

// ContextBuilderConfig holds configuration for context building
type ContextBuilderConfig struct {
	// Token management
	MaxContextTokens       int     `json:"max_context_tokens"`
	TokenOverheadBuffer    int     `json:"token_overhead_buffer"`
	ContextTruncationRatio float64 `json:"context_truncation_ratio"`
	
	// Relevance settings
	MinRelevanceScore     float32 `json:"min_relevance_score"`
	RelevanceDecayFactor  float64 `json:"relevance_decay_factor"`
	DiversityWeight       float64 `json:"diversity_weight"`
	RecencyWeight         float64 `json:"recency_weight"`
	AuthorityWeight       float64 `json:"authority_weight"`
	
	// Source attribution
	IncludeSourceRefs     bool    `json:"include_source_refs"`
	MaxSourceRefs         int     `json:"max_source_refs"`
	SourceRefFormat       string  `json:"source_ref_format"`
	
	// Context formatting
	UseStructuredFormat   bool    `json:"use_structured_format"`
	IncludeMetadata       bool    `json:"include_metadata"`
	SeparatorPattern      string  `json:"separator_pattern"`
	
	// Telecom-specific settings
	PreferStandardsDocs   bool     `json:"prefer_standards_docs"`
	RequiredSources       []string `json:"required_sources"`
	TechnologyFilters     []string `json:"technology_filters"`
	DomainPriorities      map[string]float64 `json:"domain_priorities"`
	
	// Performance settings
	MaxProcessingTime     time.Duration `json:"max_processing_time"`
	EnableParallelScoring bool          `json:"enable_parallel_scoring"`
	CacheContexts         bool          `json:"cache_contexts"`
}

// ContextMetrics tracks context building performance
type ContextMetrics struct {
	TotalRequests         int64         `json:"total_requests"`
	SuccessfulBuilds      int64         `json:"successful_builds"`
	FailedBuilds          int64         `json:"failed_builds"`
	AverageBuildTime      time.Duration `json:"average_build_time"`
	AverageContextSize    int           `json:"average_context_size"`
	AverageDocumentsUsed  int           `json:"average_documents_used"`
	TruncationRate        float64       `json:"truncation_rate"`
	CacheHitRate          float64       `json:"cache_hit_rate"`
	LastUpdated           time.Time     `json:"last_updated"`
	mutex                 sync.RWMutex
}

// ContextResult represents the result of context building
type ContextResult struct {
	Context             string                 `json:"context"`
	TokenCount          int                    `json:"token_count"`
	DocumentsUsed       []*DocumentReference   `json:"documents_used"`
	SourceReferences    []string               `json:"source_references"`
	RelevanceScores     []float32              `json:"relevance_scores"`
	TruncationApplied   bool                   `json:"truncation_applied"`
	ProcessingTime      time.Duration          `json:"processing_time"`
	QualityScore        float32                `json:"quality_score"`
	Metadata            map[string]interface{} `json:"metadata"`
	CacheUsed           bool                   `json:"cache_used"`
}

// DocumentReference represents a reference to a source document
type DocumentReference struct {
	ID               string                 `json:"id"`
	Title            string                 `json:"title"`
	Source           string                 `json:"source"`
	Version          string                 `json:"version"`
	Category         string                 `json:"category"`
	Technology       []string               `json:"technology"`
	NetworkFunction  []string               `json:"network_function"`
	RelevanceScore   float32                `json:"relevance_score"`
	UsedTokens       int                    `json:"used_tokens"`
	Position         int                    `json:"position"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// ContextRequest represents a request for context building
type ContextRequest struct {
	Query            string             `json:"query"`
	IntentType       string             `json:"intent_type"`
	ModelName        string             `json:"model_name"`
	MaxTokens        int                `json:"max_tokens"`
	SearchResults    []*rag.SearchResult `json:"search_results"`
	ExistingContext  string             `json:"existing_context"`
	RequireSources   []string           `json:"require_sources"`
	ExcludeSources   []string           `json:"exclude_sources"`
	PreferRecent     bool               `json:"prefer_recent"`
	RequestID        string             `json:"request_id"`
}

// NewContextBuilder creates a new context builder
func NewContextBuilder(tokenManager *TokenManager, relevanceScorer *RelevanceScorer, config *ContextBuilderConfig) *ContextBuilder {
	if config == nil {
		config = getDefaultContextBuilderConfig()
	}
	
	return &ContextBuilder{
		tokenManager:    tokenManager,
		relevanceScorer: relevanceScorer,
		config:          config,
		logger:          slog.Default().With("component", "context-builder"),
		metrics:         &ContextMetrics{LastUpdated: time.Now()},
	}
}

// getDefaultContextBuilderConfig returns default configuration
func getDefaultContextBuilderConfig() *ContextBuilderConfig {
	return &ContextBuilderConfig{
		MaxContextTokens:       6000,
		TokenOverheadBuffer:    500,
		ContextTruncationRatio: 0.9,
		MinRelevanceScore:      0.3,
		RelevanceDecayFactor:   0.85,
		DiversityWeight:        0.3,
		RecencyWeight:          0.2,
		AuthorityWeight:        0.4,
		IncludeSourceRefs:      true,
		MaxSourceRefs:          5,
		SourceRefFormat:        "[{index}] {source} - {title} ({version})",
		UseStructuredFormat:    true,
		IncludeMetadata:        true,
		SeparatorPattern:       "\n\n---\n\n",
		PreferStandardsDocs:    true,
		RequiredSources:        []string{"3GPP", "O-RAN", "ETSI"},
		TechnologyFilters:      []string{"5G", "4G", "O-RAN"},
		DomainPriorities: map[string]float64{
			"RAN":       1.0,
			"Core":      0.9,
			"Transport": 0.8,
			"Management": 0.7,
		},
		MaxProcessingTime:     5 * time.Second,
		EnableParallelScoring: true,
		CacheContexts:         true,
	}
}

// BuildContext creates an optimized context from search results
func (cb *ContextBuilder) BuildContext(ctx context.Context, request *ContextRequest) (*ContextResult, error) {
	startTime := time.Now()
	
	// Update metrics
	cb.updateMetrics(func(m *ContextMetrics) {
		m.TotalRequests++
	})
	
	// Validate request
	if err := cb.validateRequest(request); err != nil {
		cb.updateMetrics(func(m *ContextMetrics) {
			m.FailedBuilds++
		})
		return nil, fmt.Errorf("invalid context request: %w", err)
	}
	
	cb.logger.Debug("Building context",
		"query", request.Query,
		"intent_type", request.IntentType,
		"model", request.ModelName,
		"documents", len(request.SearchResults),
	)
	
	// Calculate token budget
	availableTokens := request.MaxTokens
	if availableTokens == 0 {
		availableTokens = cb.config.MaxContextTokens
	}
	availableTokens -= cb.config.TokenOverheadBuffer
	
	// Score and rank documents
	scoredDocs, err := cb.scoreAndRankDocuments(ctx, request)
	if err != nil {
		cb.updateMetrics(func(m *ContextMetrics) {
			m.FailedBuilds++
		})
		return nil, fmt.Errorf("failed to score documents: %w", err)
	}
	
	// Select documents that fit within token budget
	selectedDocs, totalTokens := cb.selectDocumentsForContext(scoredDocs, availableTokens, request.ModelName)
	
	// Build the final context
	contextText, sourceRefs := cb.assembleContext(selectedDocs, request)
	
	// Calculate final token count
	finalTokens := cb.tokenManager.EstimateTokensForModel(contextText, request.ModelName)
	
	// Apply truncation if necessary
	truncationApplied := false
	if finalTokens > availableTokens {
		contextText = cb.tokenManager.TruncateToFit(contextText, availableTokens, request.ModelName)
		finalTokens = cb.tokenManager.EstimateTokensForModel(contextText, request.ModelName)
		truncationApplied = true
	}
	
	// Calculate quality score
	qualityScore := cb.calculateQualityScore(selectedDocs, truncationApplied, len(request.SearchResults))
	
	// Create document references
	docRefs := cb.createDocumentReferences(selectedDocs)
	
	// Create result
	result := &ContextResult{
		Context:           contextText,
		TokenCount:        finalTokens,
		DocumentsUsed:     docRefs,
		SourceReferences:  sourceRefs,
		RelevanceScores:   cb.extractRelevanceScores(selectedDocs),
		TruncationApplied: truncationApplied,
		ProcessingTime:    time.Since(startTime),
		QualityScore:      qualityScore,
		CacheUsed:         false, // TODO: implement caching
		Metadata: map[string]interface{}{
			"available_tokens":    availableTokens,
			"total_documents":     len(request.SearchResults),
			"selected_documents":  len(selectedDocs),
			"truncation_applied":  truncationApplied,
			"average_relevance":   cb.calculateAverageRelevance(selectedDocs),
		},
	}
	
	// Update success metrics
	cb.updateMetrics(func(m *ContextMetrics) {
		m.SuccessfulBuilds++
		m.AverageBuildTime = (m.AverageBuildTime*time.Duration(m.SuccessfulBuilds-1) + result.ProcessingTime) / time.Duration(m.SuccessfulBuilds)
		m.AverageContextSize = int((float64(m.AverageContextSize)*float64(m.SuccessfulBuilds-1) + float64(finalTokens)) / float64(m.SuccessfulBuilds))
		m.AverageDocumentsUsed = int((float64(m.AverageDocumentsUsed)*float64(m.SuccessfulBuilds-1) + float64(len(selectedDocs))) / float64(m.SuccessfulBuilds))
		if truncationApplied {
			m.TruncationRate = (m.TruncationRate*float64(m.SuccessfulBuilds-1) + 1.0) / float64(m.SuccessfulBuilds)
		} else {
			m.TruncationRate = (m.TruncationRate*float64(m.SuccessfulBuilds-1) + 0.0) / float64(m.SuccessfulBuilds)
		}
		m.LastUpdated = time.Now()
	})
	
	cb.logger.Info("Context built successfully",
		"request_id", request.RequestID,
		"final_tokens", finalTokens,
		"documents_used", len(selectedDocs),
		"quality_score", fmt.Sprintf("%.3f", qualityScore),
		"processing_time", result.ProcessingTime,
	)
	
	return result, nil
}

// validateRequest validates the context building request
func (cb *ContextBuilder) validateRequest(request *ContextRequest) error {
	if request == nil {
		return fmt.Errorf("context request cannot be nil")
	}
	if request.Query == "" {
		return fmt.Errorf("query cannot be empty")
	}
	if request.ModelName == "" {
		return fmt.Errorf("model name cannot be empty")
	}
	if len(request.SearchResults) == 0 {
		return fmt.Errorf("no search results provided")
	}
	return nil
}

// scoreAndRankDocuments scores and ranks documents based on relevance
func (cb *ContextBuilder) scoreAndRankDocuments(ctx context.Context, request *ContextRequest) ([]*ScoredDocument, error) {
	if cb.relevanceScorer == nil {
		return nil, fmt.Errorf("relevance scorer not available")
	}
	
	scoredDocs := make([]*ScoredDocument, 0, len(request.SearchResults))
	
	for i, searchResult := range request.SearchResults {
		if searchResult.Document == nil {
			continue
		}
		
		// Create relevance request
		relevanceReq := &RelevanceRequest{
			Query:         request.Query,
			IntentType:    request.IntentType,
			Document:      searchResult.Document,
			Position:      i,
			OriginalScore: searchResult.Score,
			Context:       request.ExistingContext,
		}
		
		// Calculate relevance score
		score, err := cb.relevanceScorer.CalculateRelevance(ctx, relevanceReq)
		if err != nil {
			cb.logger.Warn("Failed to calculate relevance score",
				"document", searchResult.Document.ID,
				"error", err,
			)
			// Use original score as fallback
			score = &RelevanceScore{
				OverallScore:    searchResult.Score,
				SemanticScore:   searchResult.Score,
				AuthorityScore:  0.5,
				RecencyScore:    0.5,
				DomainScore:     0.5,
				IntentScore:     0.5,
			}
		}
		
		scoredDoc := &ScoredDocument{
			Document:       searchResult.Document,
			RelevanceScore: score,
			OriginalScore:  searchResult.Score,
			Position:       i,
			TokenCount:     cb.tokenManager.EstimateTokensForModel(searchResult.Document.Content, request.ModelName),
		}
		
		// Apply filters
		if cb.shouldIncludeDocument(scoredDoc, request) {
			scoredDocs = append(scoredDocs, scoredDoc)
		}
	}
	
	// Sort by overall relevance score
	sort.Slice(scoredDocs, func(i, j int) bool {
		return scoredDocs[i].RelevanceScore.OverallScore > scoredDocs[j].RelevanceScore.OverallScore
	})
	
	return scoredDocs, nil
}

// shouldIncludeDocument determines if a document should be included
func (cb *ContextBuilder) shouldIncludeDocument(doc *ScoredDocument, request *ContextRequest) bool {
	// Check minimum relevance threshold
	if doc.RelevanceScore.OverallScore < cb.config.MinRelevanceScore {
		return false
	}
	
	// Check required sources
	if len(request.RequireSources) > 0 {
		found := false
		for _, requiredSource := range request.RequireSources {
			if strings.Contains(strings.ToLower(doc.Document.Source), strings.ToLower(requiredSource)) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check excluded sources
	for _, excludedSource := range request.ExcludeSources {
		if strings.Contains(strings.ToLower(doc.Document.Source), strings.ToLower(excludedSource)) {
			return false
		}
	}
	
	return true
}

// selectDocumentsForContext selects documents that fit within the token budget
func (cb *ContextBuilder) selectDocumentsForContext(scoredDocs []*ScoredDocument, maxTokens int, modelName string) ([]*ScoredDocument, int) {
	selected := make([]*ScoredDocument, 0)
	totalTokens := 0
	
	// Diversity tracking to avoid too many similar documents
	sourceCount := make(map[string]int)
	categoryCount := make(map[string]int)
	
	for _, doc := range scoredDocs {
		// Estimate tokens including formatting overhead
		docTokens := doc.TokenCount + 50 // Overhead for formatting
		
		// Check if we can fit this document
		if totalTokens+docTokens > maxTokens {
			// Try to fit a truncated version
			remainingTokens := maxTokens - totalTokens
			if remainingTokens > 200 { // Only if we have reasonable space
				truncatedContent := cb.tokenManager.TruncateToFit(doc.Document.Content, remainingTokens-50, modelName)
				if truncatedContent != "" {
					// Create a copy with truncated content
					truncatedDoc := *doc
					truncatedDoc.Document = &rag.TelecomDocument{
						ID:              doc.Document.ID,
						Title:           doc.Document.Title,
						Content:         truncatedContent,
						Source:          doc.Document.Source,
						Category:        doc.Document.Category,
						Version:         doc.Document.Version,
						Technology:      doc.Document.Technology,
						NetworkFunction: doc.Document.NetworkFunction,
						Metadata:        doc.Document.Metadata,
						CreatedAt:       doc.Document.CreatedAt,
						UpdatedAt:       doc.Document.UpdatedAt,
					}
					truncatedDoc.TokenCount = cb.tokenManager.EstimateTokensForModel(truncatedContent, modelName)
					selected = append(selected, &truncatedDoc)
					totalTokens += truncatedDoc.TokenCount + 50
				}
			}
			break
		}
		
		// Apply diversity constraints
		if cb.shouldApplyDiversityConstraints(doc, sourceCount, categoryCount) {
			continue
		}
		
		selected = append(selected, doc)
		totalTokens += docTokens
		
		// Update diversity tracking
		sourceCount[doc.Document.Source]++
		categoryCount[doc.Document.Category]++
	}
	
	return selected, totalTokens
}

// shouldApplyDiversityConstraints checks if diversity constraints should be applied
func (cb *ContextBuilder) shouldApplyDiversityConstraints(doc *ScoredDocument, sourceCount, categoryCount map[string]int) bool {
	// Limit documents from the same source
	if sourceCount[doc.Document.Source] >= 3 {
		return true
	}
	
	// Limit documents from the same category
	if categoryCount[doc.Document.Category] >= 2 {
		return true
	}
	
	return false
}

// assembleContext creates the final context text from selected documents
func (cb *ContextBuilder) assembleContext(docs []*ScoredDocument, request *ContextRequest) (string, []string) {
	var contextParts []string
	var sourceRefs []string
	
	// Add existing context if provided
	if request.ExistingContext != "" {
		contextParts = append(contextParts, request.ExistingContext)
	}
	
	// Add document contexts
	for i, doc := range docs {
		var docParts []string
		
		if cb.config.UseStructuredFormat {
			// Structured format with metadata
			docParts = append(docParts, fmt.Sprintf("Document %d:", i+1))
			
			if cb.config.IncludeMetadata {
				if doc.Document.Title != "" {
					docParts = append(docParts, fmt.Sprintf("Title: %s", doc.Document.Title))
				}
				if doc.Document.Source != "" {
					docParts = append(docParts, fmt.Sprintf("Source: %s", doc.Document.Source))
				}
				if doc.Document.Category != "" {
					docParts = append(docParts, fmt.Sprintf("Category: %s", doc.Document.Category))
				}
				if doc.Document.Version != "" {
					docParts = append(docParts, fmt.Sprintf("Version: %s", doc.Document.Version))
				}
				if len(doc.Document.Technology) > 0 {
					docParts = append(docParts, fmt.Sprintf("Technologies: %s", strings.Join(doc.Document.Technology, ", ")))
				}
				docParts = append(docParts, fmt.Sprintf("Relevance: %.3f", doc.RelevanceScore.OverallScore))
				docParts = append(docParts, "")
			}
		}
		
		docParts = append(docParts, doc.Document.Content)
		contextParts = append(contextParts, strings.Join(docParts, "\n"))
		
		// Create source reference
		if cb.config.IncludeSourceRefs && i < cb.config.MaxSourceRefs {
			sourceRef := cb.formatSourceReference(doc, i+1)
			sourceRefs = append(sourceRefs, sourceRef)
		}
	}
	
	context := strings.Join(contextParts, cb.config.SeparatorPattern)
	return context, sourceRefs
}

// formatSourceReference formats a source reference
func (cb *ContextBuilder) formatSourceReference(doc *ScoredDocument, index int) string {
	format := cb.config.SourceRefFormat
	format = strings.ReplaceAll(format, "{index}", fmt.Sprintf("%d", index))
	format = strings.ReplaceAll(format, "{source}", doc.Document.Source)
	format = strings.ReplaceAll(format, "{title}", doc.Document.Title)
	format = strings.ReplaceAll(format, "{version}", doc.Document.Version)
	format = strings.ReplaceAll(format, "{category}", doc.Document.Category)
	return format
}

// calculateQualityScore calculates an overall quality score for the context
func (cb *ContextBuilder) calculateQualityScore(docs []*ScoredDocument, truncated bool, totalDocs int) float32 {
	if len(docs) == 0 {
		return 0.0
	}
	
	// Base score from average relevance
	var totalRelevance float32
	for _, doc := range docs {
		totalRelevance += doc.RelevanceScore.OverallScore
	}
	avgRelevance := totalRelevance / float32(len(docs))
	
	// Coverage score (how many of the available documents we used)
	coverageScore := float32(len(docs)) / float32(totalDocs)
	if coverageScore > 1.0 {
		coverageScore = 1.0
	}
	
	// Diversity score (variety of sources and categories)
	sources := make(map[string]bool)
	categories := make(map[string]bool)
	for _, doc := range docs {
		sources[doc.Document.Source] = true
		categories[doc.Document.Category] = true
	}
	diversityScore := (float32(len(sources)) + float32(len(categories))) / float32(len(docs)*2)
	
	// Quality factors
	qualityScore := avgRelevance*0.5 + coverageScore*0.3 + diversityScore*0.2
	
	// Penalty for truncation
	if truncated {
		qualityScore *= 0.9
	}
	
	return qualityScore
}

// createDocumentReferences creates document reference objects
func (cb *ContextBuilder) createDocumentReferences(docs []*ScoredDocument) []*DocumentReference {
	refs := make([]*DocumentReference, len(docs))
	
	for i, doc := range docs {
		refs[i] = &DocumentReference{
			ID:              doc.Document.ID,
			Title:           doc.Document.Title,
			Source:          doc.Document.Source,
			Version:         doc.Document.Version,
			Category:        doc.Document.Category,
			Technology:      doc.Document.Technology,
			NetworkFunction: doc.Document.NetworkFunction,
			RelevanceScore:  doc.RelevanceScore.OverallScore,
			UsedTokens:      doc.TokenCount,
			Position:        i,
			Metadata: map[string]interface{}{
				"semantic_score":  doc.RelevanceScore.SemanticScore,
				"authority_score": doc.RelevanceScore.AuthorityScore,
				"recency_score":   doc.RelevanceScore.RecencyScore,
				"domain_score":    doc.RelevanceScore.DomainScore,
				"intent_score":    doc.RelevanceScore.IntentScore,
				"original_score":  doc.OriginalScore,
			},
		}
	}
	
	return refs
}

// extractRelevanceScores extracts relevance scores from scored documents
func (cb *ContextBuilder) extractRelevanceScores(docs []*ScoredDocument) []float32 {
	scores := make([]float32, len(docs))
	for i, doc := range docs {
		scores[i] = doc.RelevanceScore.OverallScore
	}
	return scores
}

// calculateAverageRelevance calculates the average relevance score
func (cb *ContextBuilder) calculateAverageRelevance(docs []*ScoredDocument) float32 {
	if len(docs) == 0 {
		return 0.0
	}
	
	var total float32
	for _, doc := range docs {
		total += doc.RelevanceScore.OverallScore
	}
	
	return total / float32(len(docs))
}

// updateMetrics safely updates metrics
func (cb *ContextBuilder) updateMetrics(updater func(*ContextMetrics)) {
	cb.metrics.mutex.Lock()
	defer cb.metrics.mutex.Unlock()
	updater(cb.metrics)
}

// GetMetrics returns current metrics
func (cb *ContextBuilder) GetMetrics() *ContextMetrics {
	cb.metrics.mutex.RLock()
	defer cb.metrics.mutex.RUnlock()
	
	metrics := *cb.metrics
	return &metrics
}

// GetConfig returns the current configuration
func (cb *ContextBuilder) GetConfig() *ContextBuilderConfig {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	
	config := *cb.config
	return &config
}

// UpdateConfig updates the configuration
func (cb *ContextBuilder) UpdateConfig(config *ContextBuilderConfig) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}
	
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	cb.config = config
	cb.logger.Info("Context builder configuration updated")
	
	return nil
}

// OptimizeForModel optimizes context building for a specific model
func (cb *ContextBuilder) OptimizeForModel(modelName string, request *ContextRequest) (*ContextRequest, error) {
	// Get model capabilities
	supportsStructured := cb.tokenManager.SupportsChatFormat(modelName)
	maxTokens := 4000 // Default, could be retrieved from token manager
	
	// Create optimized request
	optimizedRequest := *request
	optimizedRequest.ModelName = modelName
	
	if request.MaxTokens == 0 {
		optimizedRequest.MaxTokens = maxTokens
	}
	
	// Adjust configuration based on model capabilities
	if !supportsStructured && cb.config.UseStructuredFormat {
		cb.logger.Debug("Disabling structured format for model", "model", modelName)
		// Would create a temporary config here in a full implementation
	}
	
	return &optimizedRequest, nil
}