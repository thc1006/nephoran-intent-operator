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

// RelevanceScorer calculates multi-factor relevance scores for documents
type RelevanceScorer struct {
	config          *RelevanceScorerConfig
	logger          *slog.Logger
	embeddings      EmbeddingService // Interface for semantic similarity
	domainKnowledge *TelecomDomainKnowledge
	metrics         *ScoringMetrics
	mutex           sync.RWMutex
}

// RelevanceScorerConfig holds configuration for relevance scoring
type RelevanceScorerConfig struct {
	// Scoring weights (must sum to 1.0)
	SemanticWeight      float64 `json:"semantic_weight"`
	AuthorityWeight     float64 `json:"authority_weight"`
	RecencyWeight       float64 `json:"recency_weight"`
	DomainWeight        float64 `json:"domain_weight"`
	IntentAlignmentWeight float64 `json:"intent_alignment_weight"`
	
	// Semantic similarity settings
	MinSemanticSimilarity float64 `json:"min_semantic_similarity"`
	UseEmbeddingDistance  bool    `json:"use_embedding_distance"`
	
	// Authority scoring
	AuthorityScores    map[string]float64 `json:"authority_scores"`
	StandardsMultiplier float64           `json:"standards_multiplier"`
	
	// Recency scoring
	RecencyHalfLife     time.Duration `json:"recency_half_life"`
	MaxAge              time.Duration `json:"max_age"`
	
	// Domain specificity
	DomainKeywords      map[string][]string `json:"domain_keywords"`
	TechnologyBoosts    map[string]float64  `json:"technology_boosts"`
	
	// Intent alignment
	IntentPatterns      map[string][]string `json:"intent_patterns"`
	ContextualBoosts    map[string]float64  `json:"contextual_boosts"`
	
	// Performance settings
	CacheScores         bool          `json:"cache_scores"`
	ScoreCacheTTL       time.Duration `json:"score_cache_ttl"`
	ParallelProcessing  bool          `json:"parallel_processing"`
	MaxProcessingTime   time.Duration `json:"max_processing_time"`
}

// ScoringMetrics tracks scoring performance
type ScoringMetrics struct {
	TotalScores          int64         `json:"total_scores"`
	AverageScoringTime   time.Duration `json:"average_scoring_time"`
	CacheHitRate         float64       `json:"cache_hit_rate"`
	SemanticScores       int64         `json:"semantic_scores"`
	AuthorityScores      int64         `json:"authority_scores"`
	RecencyScores        int64         `json:"recency_scores"`
	DomainScores         int64         `json:"domain_scores"`
	IntentScores         int64         `json:"intent_scores"`
	LastUpdated          time.Time     `json:"last_updated"`
	mutex                sync.RWMutex
}

// RelevanceScore represents a multi-factor relevance score
type RelevanceScore struct {
	OverallScore    float32                `json:"overall_score"`
	SemanticScore   float32                `json:"semantic_score"`
	AuthorityScore  float32                `json:"authority_score"`
	RecencyScore    float32                `json:"recency_score"`
	DomainScore     float32                `json:"domain_score"`
	IntentScore     float32                `json:"intent_score"`
	Explanation     string                 `json:"explanation"`
	Factors         map[string]interface{} `json:"factors"`
	ProcessingTime  time.Duration          `json:"processing_time"`
	CacheUsed       bool                   `json:"cache_used"`
}

// RelevanceRequest represents a relevance scoring request
type RelevanceRequest struct {
	Query         string             `json:"query"`
	IntentType    string             `json:"intent_type"`
	Document      *rag.TelecomDocument `json:"document"`
	Position      int                `json:"position"`
	OriginalScore float32            `json:"original_score"`
	Context       string             `json:"context"`
	UserProfile   map[string]interface{} `json:"user_profile,omitempty"`
}

// ScoredDocument represents a document with relevance scoring
type ScoredDocument struct {
	Document       *rag.TelecomDocument `json:"document"`
	RelevanceScore *RelevanceScore      `json:"relevance_score"`
	OriginalScore  float32              `json:"original_score"`
	Position       int                  `json:"position"`
	TokenCount     int                  `json:"token_count"`
}

// EmbeddingService interface for semantic similarity calculations
type EmbeddingService interface {
	CalculateSimilarity(ctx context.Context, text1, text2 string) (float64, error)
	GetEmbedding(ctx context.Context, text string) ([]float64, error)
}

// TelecomDomainKnowledge provides domain-specific knowledge for scoring
type TelecomDomainKnowledge struct {
	Standards      map[string]float64 `json:"standards"`
	Technologies   map[string]float64 `json:"technologies"`
	Organizations  map[string]float64 `json:"organizations"`
	Protocols      map[string]float64 `json:"protocols"`
	Abbreviations  map[string]string  `json:"abbreviations"`
}

// NewRelevanceScorer creates a new relevance scorer
func NewRelevanceScorer(config *RelevanceScorerConfig, embeddings EmbeddingService) *RelevanceScorer {
	if config == nil {
		config = getDefaultRelevanceScorerConfig()
	}
	
	return &RelevanceScorer{
		config:          config,
		logger:          slog.Default().With("component", "relevance-scorer"),
		embeddings:      embeddings,
		domainKnowledge: initializeTelecomDomainKnowledge(),
		metrics:         &ScoringMetrics{LastUpdated: time.Now()},
	}
}

// getDefaultRelevanceScorerConfig returns default configuration
func getDefaultRelevanceScorerConfig() *RelevanceScorerConfig {
	return &RelevanceScorerConfig{
		SemanticWeight:           0.35,
		AuthorityWeight:          0.25,
		RecencyWeight:            0.15,
		DomainWeight:             0.15,
		IntentAlignmentWeight:    0.10,
		MinSemanticSimilarity:    0.3,
		UseEmbeddingDistance:     true,
		AuthorityScores: map[string]float64{
			"3GPP":     1.0,
			"ETSI":     0.95,
			"ITU":      0.9,
			"O-RAN":    0.9,
			"IEEE":     0.85,
			"IETF":     0.8,
			"GSMA":     0.75,
		},
		StandardsMultiplier: 1.2,
		RecencyHalfLife:     365 * 24 * time.Hour, // 1 year
		MaxAge:              5 * 365 * 24 * time.Hour, // 5 years
		DomainKeywords: map[string][]string{
			"RAN": {"ran", "radio", "antenna", "gnb", "enb", "cell", "handover", "mobility"},
			"Core": {"core", "amf", "smf", "upf", "ausf", "udm", "pcf", "nrf", "session"},
			"Transport": {"transport", "ip", "mpls", "ethernet", "optical", "backhaul", "fronthaul"},
			"Management": {"management", "orchestration", "automation", "monitoring", "configuration"},
			"O-RAN": {"o-ran", "oran", "open", "disaggregated", "virtualized", "cloudified"},
		},
		TechnologyBoosts: map[string]float64{
			"5G":     1.2,
			"4G":     1.0,
			"O-RAN":  1.15,
			"vRAN":   1.1,
			"Cloud":  1.05,
		},
		IntentPatterns: map[string][]string{
			"configuration": {"configure", "setup", "parameter", "setting", "config"},
			"optimization": {"optimize", "performance", "tuning", "efficiency", "improve"},
			"troubleshooting": {"troubleshoot", "debug", "error", "issue", "problem", "fault"},
			"monitoring": {"monitor", "observe", "track", "measure", "kpi", "metric"},
		},
		ContextualBoosts: map[string]float64{
			"implementation": 1.1,
			"procedure":      1.05,
			"specification":  1.15,
			"standard":       1.2,
		},
		CacheScores:        true,
		ScoreCacheTTL:      30 * time.Minute,
		ParallelProcessing: true,
		MaxProcessingTime:  2 * time.Second,
	}
}

// initializeTelecomDomainKnowledge creates domain knowledge base
func initializeTelecomDomainKnowledge() *TelecomDomainKnowledge {
	return &TelecomDomainKnowledge{
		Standards: map[string]float64{
			"TS 38":    1.0,  // 5G NR standards
			"TS 36":    0.9,  // LTE standards
			"TS 23":    0.95, // System architecture
			"TS 29":    0.9,  // Protocol specifications
			"TR":       0.7,  // Technical reports
			"O-RAN.WG": 0.95, // O-RAN working group specs
		},
		Technologies: map[string]float64{
			"5G NR":        1.0,
			"LTE":          0.9,
			"O-RAN":        0.95,
			"vRAN":         0.9,
			"Cloud-RAN":    0.85,
			"Network Slicing": 0.95,
			"Edge Computing": 0.9,
			"NFV":          0.85,
			"SDN":          0.8,
		},
		Organizations: map[string]float64{
			"3GPP":         1.0,
			"O-RAN Alliance": 0.95,
			"ETSI":         0.9,
			"ITU":          0.9,
			"IEEE":         0.85,
			"IETF":         0.8,
			"GSMA":         0.75,
			"TIA":          0.7,
		},
		Protocols: map[string]float64{
			"NAS":          1.0,
			"RRC":          1.0,
			"PDCP":         0.95,
			"RLC":          0.95,
			"MAC":          0.95,
			"PHY":          0.9,
			"NGAP":         0.9,
			"SCTP":         0.8,
			"GTP":          0.9,
		},
		Abbreviations: map[string]string{
			"gNB":  "gNodeB",
			"eNB":  "eNodeB",
			"AMF":  "Access and Mobility Management Function",
			"SMF":  "Session Management Function",
			"UPF":  "User Plane Function",
			"AUSF": "Authentication Server Function",
			"UDM":  "Unified Data Management",
			"PCF":  "Policy Control Function",
			"NRF":  "Network Repository Function",
			"NSSF": "Network Slice Selection Function",
		},
	}
}

// CalculateRelevance calculates the overall relevance score for a document
func (rs *RelevanceScorer) CalculateRelevance(ctx context.Context, request *RelevanceRequest) (*RelevanceScore, error) {
	startTime := time.Now()
	
	// Update metrics
	rs.updateMetrics(func(m *ScoringMetrics) {
		m.TotalScores++
	})
	
	// Validate request
	if err := rs.validateRequest(request); err != nil {
		return nil, fmt.Errorf("invalid relevance request: %w", err)
	}
	
	rs.logger.Debug("Calculating relevance score",
		"document_id", request.Document.ID,
		"query", request.Query,
		"intent_type", request.IntentType,
	)
	
	// Calculate individual scores
	var scores struct {
		semantic  float32
		authority float32
		recency   float32
		domain    float32
		intent    float32
	}
	
	var err error
	
	// Semantic similarity score
	scores.semantic, err = rs.calculateSemanticScore(ctx, request)
	if err != nil {
		rs.logger.Warn("Failed to calculate semantic score", "error", err)
		scores.semantic = 0.5 // Fallback
	}
	rs.updateMetrics(func(m *ScoringMetrics) { m.SemanticScores++ })
	
	// Authority score
	scores.authority = rs.calculateAuthorityScore(request)
	rs.updateMetrics(func(m *ScoringMetrics) { m.AuthorityScores++ })
	
	// Recency score
	scores.recency = rs.calculateRecencyScore(request)
	rs.updateMetrics(func(m *ScoringMetrics) { m.RecencyScores++ })
	
	// Domain specificity score
	scores.domain = rs.calculateDomainScore(request)
	rs.updateMetrics(func(m *ScoringMetrics) { m.DomainScores++ })
	
	// Intent alignment score
	scores.intent = rs.calculateIntentScore(request)
	rs.updateMetrics(func(m *ScoringMetrics) { m.IntentScores++ })
	
	// Calculate weighted overall score
	overallScore := float32(
		float64(scores.semantic)*rs.config.SemanticWeight +
		float64(scores.authority)*rs.config.AuthorityWeight +
		float64(scores.recency)*rs.config.RecencyWeight +
		float64(scores.domain)*rs.config.DomainWeight +
		float64(scores.intent)*rs.config.IntentAlignmentWeight,
	)
	
	// Apply position penalty (later results get slightly lower scores)
	positionPenalty := 1.0 - (float64(request.Position) * 0.01)
	if positionPenalty < 0.8 {
		positionPenalty = 0.8
	}
	overallScore *= float32(positionPenalty)
	
	// Create explanation
	explanation := rs.generateExplanation(scores, request)
	
	// Create factors map
	factors := map[string]interface{}{
		"semantic_similarity": scores.semantic,
		"source_authority":    scores.authority,
		"document_recency":    scores.recency,
		"domain_specificity":  scores.domain,
		"intent_alignment":    scores.intent,
		"position_penalty":    positionPenalty,
		"original_score":      request.OriginalScore,
		"weights": map[string]float64{
			"semantic":  rs.config.SemanticWeight,
			"authority": rs.config.AuthorityWeight,
			"recency":   rs.config.RecencyWeight,
			"domain":    rs.config.DomainWeight,
			"intent":    rs.config.IntentAlignmentWeight,
		},
	}
	
	processingTime := time.Since(startTime)
	
	relevanceScore := &RelevanceScore{
		OverallScore:   overallScore,
		SemanticScore:  scores.semantic,
		AuthorityScore: scores.authority,
		RecencyScore:   scores.recency,
		DomainScore:    scores.domain,
		IntentScore:    scores.intent,
		Explanation:    explanation,
		Factors:        factors,
		ProcessingTime: processingTime,
		CacheUsed:      false, // TODO: implement caching
	}
	
	// Update metrics
	rs.updateMetrics(func(m *ScoringMetrics) {
		m.AverageScoringTime = (m.AverageScoringTime*time.Duration(m.TotalScores-1) + processingTime) / time.Duration(m.TotalScores)
		m.LastUpdated = time.Now()
	})
	
	rs.logger.Debug("Relevance score calculated",
		"document_id", request.Document.ID,
		"overall_score", fmt.Sprintf("%.3f", overallScore),
		"processing_time", processingTime,
	)
	
	return relevanceScore, nil
}

// validateRequest validates the relevance scoring request
func (rs *RelevanceScorer) validateRequest(request *RelevanceRequest) error {
	if request == nil {
		return fmt.Errorf("relevance request cannot be nil")
	}
	if request.Query == "" {
		return fmt.Errorf("query cannot be empty")
	}
	if request.Document == nil {
		return fmt.Errorf("document cannot be nil")
	}
	return nil
}

// calculateSemanticScore calculates semantic similarity between query and document
func (rs *RelevanceScorer) calculateSemanticScore(ctx context.Context, request *RelevanceRequest) (float32, error) {
	if rs.embeddings == nil || !rs.config.UseEmbeddingDistance {
		// Fallback to keyword-based similarity
		return rs.calculateKeywordSimilarity(request), nil
	}
	
	// Calculate semantic similarity using embeddings
	similarity, err := rs.embeddings.CalculateSimilarity(ctx, request.Query, request.Document.Content)
	if err != nil {
		rs.logger.Warn("Embedding similarity calculation failed", "error", err)
		return rs.calculateKeywordSimilarity(request), nil
	}
	
	// Normalize and apply minimum threshold
	semanticScore := float32(similarity)
	if semanticScore < float32(rs.config.MinSemanticSimilarity) {
		semanticScore = float32(rs.config.MinSemanticSimilarity)
	}
	
	// Boost for exact matches
	if rs.hasExactMatches(request.Query, request.Document.Content) {
		semanticScore *= 1.1
		if semanticScore > 1.0 {
			semanticScore = 1.0
		}
	}
	
	return semanticScore, nil
}

// calculateKeywordSimilarity calculates similarity based on keyword overlap
func (rs *RelevanceScorer) calculateKeywordSimilarity(request *RelevanceRequest) float32 {
	queryWords := rs.extractKeywords(strings.ToLower(request.Query))
	docWords := rs.extractKeywords(strings.ToLower(request.Document.Content))
	
	if len(queryWords) == 0 {
		return 0.0
	}
	
	matches := 0
	for _, qWord := range queryWords {
		for _, dWord := range docWords {
			if qWord == dWord {
				matches++
				break
			}
		}
	}
	
	similarity := float32(matches) / float32(len(queryWords))
	
	// Boost for technical terms and abbreviations
	for term := range rs.domainKnowledge.Abbreviations {
		if strings.Contains(strings.ToLower(request.Query), strings.ToLower(term)) &&
		   strings.Contains(strings.ToLower(request.Document.Content), strings.ToLower(term)) {
			similarity *= 1.2
		}
	}
	
	if similarity > 1.0 {
		similarity = 1.0
	}
	
	return similarity
}

// hasExactMatches checks for exact phrase matches
func (rs *RelevanceScorer) hasExactMatches(query, content string) bool {
	queryLower := strings.ToLower(query)
	contentLower := strings.ToLower(content)
	
	// Check for exact phrase matches
	words := strings.Fields(queryLower)
	if len(words) >= 2 {
		for i := 0; i <= len(words)-2; i++ {
			phrase := strings.Join(words[i:i+2], " ")
			if strings.Contains(contentLower, phrase) {
				return true
			}
		}
	}
	
	return false
}

// extractKeywords extracts relevant keywords from text
func (rs *RelevanceScorer) extractKeywords(text string) []string {
	// Simple keyword extraction (in production, use more sophisticated NLP)
	words := strings.Fields(text)
	keywords := make([]string, 0)
	
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
		"be": true, "been": true, "being": true, "have": true, "has": true, "had": true,
		"do": true, "does": true, "did": true, "will": true, "would": true, "could": true,
		"should": true, "may": true, "might": true, "must": true, "can": true,
	}
	
	for _, word := range words {
		word = strings.Trim(word, ".,!?;:()[]{}\"'")
		if len(word) > 2 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}
	
	return keywords
}

// calculateAuthorityScore calculates authority score based on source credibility
func (rs *RelevanceScorer) calculateAuthorityScore(request *RelevanceRequest) float32 {
	doc := request.Document
	
	// Base authority from source
	baseAuthority := 0.5 // Default
	sourceLower := strings.ToLower(doc.Source)
	
	for source, score := range rs.config.AuthorityScores {
		if strings.Contains(sourceLower, strings.ToLower(source)) {
			baseAuthority = score
			break
		}
	}
	
	// Check for organization authority
	for org, score := range rs.domainKnowledge.Organizations {
		if strings.Contains(sourceLower, strings.ToLower(org)) {
			if score > baseAuthority {
				baseAuthority = score
			}
		}
	}
	
	// Boost for standards documents
	if rs.isStandardsDocument(doc) {
		baseAuthority *= rs.config.StandardsMultiplier
	}
	
	// Version authority (newer versions of specs are more authoritative)
	if doc.Version != "" {
		versionBoost := rs.calculateVersionAuthority(doc.Version)
		baseAuthority *= versionBoost
	}
	
	if baseAuthority > 1.0 {
		baseAuthority = 1.0
	}
	
	return float32(baseAuthority)
}

// isStandardsDocument checks if document is a standards document
func (rs *RelevanceScorer) isStandardsDocument(doc *rag.TelecomDocument) bool {
	indicators := []string{"TS ", "TR ", "RFC ", "IEEE ", "O-RAN.WG", "specification", "standard"}
	
	titleLower := strings.ToLower(doc.Title)
	sourceLower := strings.ToLower(doc.Source)
	
	for _, indicator := range indicators {
		if strings.Contains(titleLower, strings.ToLower(indicator)) ||
		   strings.Contains(sourceLower, strings.ToLower(indicator)) {
			return true
		}
	}
	
	return false
}

// calculateVersionAuthority calculates authority boost based on version
func (rs *RelevanceScorer) calculateVersionAuthority(version string) float64 {
	// Simple version authority calculation
	// In production, this would be more sophisticated
	versionLower := strings.ToLower(version)
	
	if strings.Contains(versionLower, "latest") || strings.Contains(versionLower, "current") {
		return 1.1
	}
	
	// Look for version numbers and assume higher is better
	// This is a simplified approach
	if strings.Contains(versionLower, "v17") || strings.Contains(versionLower, "rel-17") {
		return 1.05
	}
	if strings.Contains(versionLower, "v16") || strings.Contains(versionLower, "rel-16") {
		return 1.02
	}
	if strings.Contains(versionLower, "v15") || strings.Contains(versionLower, "rel-15") {
		return 1.0
	}
	
	return 1.0
}

// calculateRecencyScore calculates recency score based on document age
func (rs *RelevanceScorer) calculateRecencyScore(request *RelevanceRequest) float32 {
	doc := request.Document
	
	// Use UpdatedAt if available, otherwise CreatedAt
	var docTime time.Time
	if !doc.UpdatedAt.IsZero() {
		docTime = doc.UpdatedAt
	} else if !doc.CreatedAt.IsZero() {
		docTime = doc.CreatedAt
	} else {
		// No timestamp available, assume medium recency
		return 0.5
	}
	
	age := time.Since(docTime)
	
	// Apply exponential decay based on half-life
	if age > rs.config.MaxAge {
		return 0.1 // Very old documents get minimum score
	}
	
	// Calculate exponential decay
	halfLife := rs.config.RecencyHalfLife
	decay := math.Exp(-float64(age) * math.Ln2 / float64(halfLife))
	
	recencyScore := float32(decay)
	
	// Boost for very recent documents
	if age < 30*24*time.Hour { // Less than 30 days
		recencyScore *= 1.1
		if recencyScore > 1.0 {
			recencyScore = 1.0
		}
	}
	
	return recencyScore
}

// calculateDomainScore calculates domain specificity score
func (rs *RelevanceScorer) calculateDomainScore(request *RelevanceRequest) float32 {
	doc := request.Document
	query := strings.ToLower(request.Query)
	content := strings.ToLower(doc.Content + " " + doc.Title)
	
	domainScore := 0.0
	matchCount := 0
	
	// Check domain keywords
	for domain, keywords := range rs.config.DomainKeywords {
		domainMatch := false
		for _, keyword := range keywords {
			if strings.Contains(query, keyword) && strings.Contains(content, keyword) {
				domainMatch = true
				matchCount++
			}
		}
		if domainMatch {
			domainScore += 0.3 // Each domain match adds to score
		}
	}
	
	// Check technology boosts
	for tech, boost := range rs.config.TechnologyBoosts {
		if strings.Contains(query, strings.ToLower(tech)) && 
		   strings.Contains(content, strings.ToLower(tech)) {
			domainScore += 0.2 * boost
			matchCount++
		}
	}
	
	// Check for technology in document metadata
	for _, tech := range doc.Technology {
		if strings.Contains(query, strings.ToLower(tech)) {
			if boost, exists := rs.config.TechnologyBoosts[tech]; exists {
				domainScore += 0.2 * boost
			} else {
				domainScore += 0.1
			}
			matchCount++
		}
	}
	
	// Check network functions
	for _, nf := range doc.NetworkFunction {
		if strings.Contains(query, strings.ToLower(nf)) {
			domainScore += 0.15
			matchCount++
		}
	}
	
	// Normalize based on matches found
	if matchCount > 0 {
		domainScore = domainScore / float64(matchCount)
	}
	
	if domainScore > 1.0 {
		domainScore = 1.0
	}
	
	return float32(domainScore)
}

// calculateIntentScore calculates intent alignment score
func (rs *RelevanceScorer) calculateIntentScore(request *RelevanceRequest) float32 {
	if request.IntentType == "" {
		return 0.5 // Default when no intent specified
	}
	
	intentLower := strings.ToLower(request.IntentType)
	query := strings.ToLower(request.Query)
	content := strings.ToLower(request.Document.Content + " " + request.Document.Title)
	
	intentScore := 0.0
	
	// Check intent-specific patterns
	if patterns, exists := rs.config.IntentPatterns[intentLower]; exists {
		matchCount := 0
		for _, pattern := range patterns {
			if strings.Contains(query, pattern) && strings.Contains(content, pattern) {
				matchCount++
			}
		}
		if matchCount > 0 {
			intentScore += 0.4 * float64(matchCount) / float64(len(patterns))
		}
	}
	
	// Check contextual boosts
	for context, boost := range rs.config.ContextualBoosts {
		if strings.Contains(content, context) {
			intentScore += 0.2 * boost
		}
	}
	
	// Category alignment
	docCategory := strings.ToLower(request.Document.Category)
	if docCategory == intentLower {
		intentScore += 0.3
	} else if strings.Contains(docCategory, intentLower) || strings.Contains(intentLower, docCategory) {
		intentScore += 0.2
	}
	
	if intentScore > 1.0 {
		intentScore = 1.0
	}
	
	return float32(intentScore)
}

// generateExplanation generates a human-readable explanation of the scoring
func (rs *RelevanceScorer) generateExplanation(scores struct {
	semantic, authority, recency, domain, intent float32
}, request *RelevanceRequest) string {
	var explanations []string
	
	if scores.semantic > 0.7 {
		explanations = append(explanations, "High semantic similarity to query")
	} else if scores.semantic > 0.4 {
		explanations = append(explanations, "Moderate semantic similarity to query")
	} else {
		explanations = append(explanations, "Low semantic similarity to query")
	}
	
	if scores.authority > 0.8 {
		explanations = append(explanations, "High authority source")
	} else if scores.authority > 0.6 {
		explanations = append(explanations, "Credible source")
	}
	
	if scores.recency > 0.7 {
		explanations = append(explanations, "Recent document")
	} else if scores.recency < 0.3 {
		explanations = append(explanations, "Older document")
	}
	
	if scores.domain > 0.6 {
		explanations = append(explanations, "Strong domain relevance")
	}
	
	if scores.intent > 0.6 {
		explanations = append(explanations, "Good intent alignment")
	}
	
	return strings.Join(explanations, "; ")
}

// updateMetrics safely updates metrics
func (rs *RelevanceScorer) updateMetrics(updater func(*ScoringMetrics)) {
	rs.metrics.mutex.Lock()
	defer rs.metrics.mutex.Unlock()
	updater(rs.metrics)
}

// GetMetrics returns current metrics
func (rs *RelevanceScorer) GetMetrics() *ScoringMetrics {
	rs.metrics.mutex.RLock()
	defer rs.metrics.mutex.RUnlock()
	
	metrics := *rs.metrics
	return &metrics
}

// GetConfig returns the current configuration
func (rs *RelevanceScorer) GetConfig() *RelevanceScorerConfig {
	rs.mutex.RLock()
	defer rs.mutex.RUnlock()
	
	config := *rs.config
	return &config
}

// UpdateConfig updates the configuration
func (rs *RelevanceScorer) UpdateConfig(config *RelevanceScorerConfig) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}
	
	// Validate weights sum to 1.0
	totalWeight := config.SemanticWeight + config.AuthorityWeight + config.RecencyWeight + 
					config.DomainWeight + config.IntentAlignmentWeight
	if math.Abs(totalWeight - 1.0) > 0.01 {
		return fmt.Errorf("scoring weights must sum to 1.0, got %.3f", totalWeight)
	}
	
	rs.mutex.Lock()
	defer rs.mutex.Unlock()
	
	rs.config = config
	rs.logger.Info("Relevance scorer configuration updated")
	
	return nil
}