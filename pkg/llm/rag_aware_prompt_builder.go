package llm

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// RAGAwarePromptBuilder builds telecom-specific prompts with RAG context integration
type RAGAwarePromptBuilder struct {
	tokenManager         *TokenManager
	telecomQueryEnhancer *TelecomQueryEnhancer
	promptTemplates      *TelecomPromptTemplates
	config               *PromptBuilderConfig
	logger               *slog.Logger
	metrics              *PromptBuilderMetrics
	mutex                sync.RWMutex
}

// PromptBuilderConfig holds configuration for prompt building
type PromptBuilderConfig struct {
	// Template settings
	DefaultTemplate           string `json:"default_template"`
	EnableContextOptimization bool   `json:"enable_context_optimization"`
	MaxPromptTokens           int    `json:"max_prompt_tokens"`
	SystemPromptTokens        int    `json:"system_prompt_tokens"`

	// RAG integration
	ContextSeparator          string  `json:"context_separator"`
	SourceAttributionFormat   string  `json:"source_attribution_format"`
	ContextPreamble           string  `json:"context_preamble"`
	MaxContextSources         int     `json:"max_context_sources"`
	ContextRelevanceThreshold float32 `json:"context_relevance_threshold"`

	// Telecom enhancement
	EnableAbbreviationExpansion bool     `json:"enable_abbreviation_expansion"`
	EnableStandardsMapping      bool     `json:"enable_standards_mapping"`
	TelecomDomains              []string `json:"telecom_domains"`
	PreferredStandards          []string `json:"preferred_standards"`

	// Few-shot examples
	EnableFewShotExamples    bool   `json:"enable_few_shot_examples"`
	MaxFewShotExamples       int    `json:"max_few_shot_examples"`
	FewShotSelectionStrategy string `json:"few_shot_selection_strategy"`

	// Performance settings
	CachePrompts             bool          `json:"cache_prompts"`
	PromptCacheTTL           time.Duration `json:"prompt_cache_ttl"`
	EnableParallelProcessing bool          `json:"enable_parallel_processing"`
}

// PromptBuilderMetrics tracks prompt building performance
type PromptBuilderMetrics struct {
	TotalPrompts           int64         `json:"total_prompts"`
	CachedPrompts          int64         `json:"cached_prompts"`
	ContextOptimizations   int64         `json:"context_optimizations"`
	AbbreviationExpansions int64         `json:"abbreviation_expansions"`
	StandardsMappings      int64         `json:"standards_mappings"`
	FewShotExamplesUsed    int64         `json:"few_shot_examples_used"`
	AveragePromptTokens    int           `json:"average_prompt_tokens"`
	AverageProcessingTime  time.Duration `json:"average_processing_time"`
	LastUpdated            time.Time     `json:"last_updated"`
	mutex                  sync.RWMutex
}

// TelecomQueryEnhancer enhances queries with telecom-specific knowledge
type TelecomQueryEnhancer struct {
	abbreviations    map[string]string
	standardsMapping map[string]StandardInfo
	protocolMappings map[string]ProtocolInfo
	domainClassifier *DomainClassifier
	logger           *slog.Logger
}

// StandardInfo holds information about telecom standards
type StandardInfo struct {
	FullName     string   `json:"full_name"`
	Organization string   `json:"organization"`
	Domain       string   `json:"domain"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Keywords     []string `json:"keywords"`
}

// ProtocolInfo holds information about telecom protocols
type ProtocolInfo struct {
	FullName  string   `json:"full_name"`
	Layer     string   `json:"layer"`
	Function  string   `json:"function"`
	Standards []string `json:"standards"`
	Keywords  []string `json:"keywords"`
}

// DomainClassifier classifies queries into telecom domains
type DomainClassifier struct {
	domainKeywords map[string][]string
	mutex          sync.RWMutex
}

// TelecomPromptTemplates holds various prompt templates
type TelecomPromptTemplates struct {
	systemPrompts    map[string]string
	contextTemplates map[string]string
	fewShotExamples  map[string][]FewShotExample
	mutex            sync.RWMutex
}

// FewShotExample represents a few-shot learning example
type FewShotExample struct {
	Query      string                 `json:"query"`
	Context    string                 `json:"context"`
	Response   string                 `json:"response"`
	Domain     string                 `json:"domain"`
	IntentType string                 `json:"intent_type"`
	Relevance  float32                `json:"relevance"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// PromptRequest represents a request for prompt building
type PromptRequest struct {
	Query              string                 `json:"query"`
	IntentType         string                 `json:"intent_type"`
	ModelName          string                 `json:"model_name"`
	RAGContext         []*shared.SearchResult `json:"rag_context"`
	Domain             string                 `json:"domain,omitempty"`
	ExistingContext    string                 `json:"existing_context,omitempty"`
	TemplateType       string                 `json:"template_type,omitempty"`
	MaxTokens          int                    `json:"max_tokens,omitempty"`
	IncludeFewShot     bool                   `json:"include_few_shot"`
	CustomInstructions string                 `json:"custom_instructions,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// PromptResponse represents the response from prompt building
type PromptResponse struct {
	SystemPrompt         string                 `json:"system_prompt"`
	UserPrompt           string                 `json:"user_prompt"`
	FullPrompt           string                 `json:"full_prompt"`
	TokenCount           int                    `json:"token_count"`
	ContextSources       []string               `json:"context_sources"`
	EnhancedQuery        string                 `json:"enhanced_query"`
	FewShotExamples      []FewShotExample       `json:"few_shot_examples"`
	ProcessingTime       time.Duration          `json:"processing_time"`
	OptimizationsApplied []string               `json:"optimizations_applied"`
	Metadata             map[string]interface{} `json:"metadata"`
	CacheUsed            bool                   `json:"cache_used"`
}

// NewRAGAwarePromptBuilder creates a new RAG-aware prompt builder
func NewRAGAwarePromptBuilder(tokenManager *TokenManager, config *PromptBuilderConfig) *RAGAwarePromptBuilder {
	if config == nil {
		config = getDefaultPromptBuilderConfig()
	}

	return &RAGAwarePromptBuilder{
		tokenManager:         tokenManager,
		telecomQueryEnhancer: NewTelecomQueryEnhancer(),
		promptTemplates:      NewTelecomPromptTemplates(),
		config:               config,
		logger:               slog.Default().With("component", "rag-aware-prompt-builder"),
		metrics:              &PromptBuilderMetrics{LastUpdated: time.Now()},
	}
}

// getDefaultPromptBuilderConfig returns default configuration
func getDefaultPromptBuilderConfig() *PromptBuilderConfig {
	return &PromptBuilderConfig{
		DefaultTemplate:             "telecom_expert",
		EnableContextOptimization:   true,
		MaxPromptTokens:             4000,
		SystemPromptTokens:          500,
		ContextSeparator:            "\n\n---\n\n",
		SourceAttributionFormat:     "[{index}] {source} - {title}",
		ContextPreamble:             "Based on the following technical documentation:",
		MaxContextSources:           5,
		ContextRelevanceThreshold:   0.3,
		EnableAbbreviationExpansion: true,
		EnableStandardsMapping:      true,
		TelecomDomains:              []string{"RAN", "Core", "Transport", "Management", "O-RAN"},
		PreferredStandards:          []string{"3GPP", "O-RAN", "ETSI", "ITU"},
		EnableFewShotExamples:       true,
		MaxFewShotExamples:          3,
		FewShotSelectionStrategy:    "similarity",
		CachePrompts:                true,
		PromptCacheTTL:              15 * time.Minute,
		EnableParallelProcessing:    true,
	}
}

// BuildPrompt builds a RAG-enhanced prompt for telecom queries
func (pb *RAGAwarePromptBuilder) BuildPrompt(ctx context.Context, request *PromptRequest) (*PromptResponse, error) {
	startTime := time.Now()

	// Update metrics
	pb.updateMetrics(func(m *PromptBuilderMetrics) {
		m.TotalPrompts++
	})

	// Validate request
	if err := pb.validateRequest(request); err != nil {
		return nil, fmt.Errorf("invalid prompt request: %w", err)
	}

	pb.logger.Debug("Building RAG-aware prompt",
		"query", request.Query,
		"intent_type", request.IntentType,
		"model", request.ModelName,
		"context_sources", len(request.RAGContext),
	)

	var optimizations []string

	// Step 1: Enhance the query with telecom-specific knowledge
	enhancedQuery, queryOptimizations := pb.enhanceQuery(request.Query, request.IntentType)
	optimizations = append(optimizations, queryOptimizations...)

	// Step 2: Classify domain if not provided
	domain := request.Domain
	if domain == "" {
		domain = pb.telecomQueryEnhancer.ClassifyDomain(enhancedQuery)
	}

	// Step 3: Build system prompt
	systemPrompt, err := pb.buildSystemPrompt(request, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to build system prompt: %w", err)
	}

	// Step 4: Process RAG context
	contextText, contextSources, contextOptimizations := pb.processRAGContext(request.RAGContext, request.ModelName)
	optimizations = append(optimizations, contextOptimizations...)

	// Step 5: Select few-shot examples if enabled
	var fewShotExamples []FewShotExample
	if request.IncludeFewShot && pb.config.EnableFewShotExamples {
		fewShotExamples = pb.selectFewShotExamples(enhancedQuery, request.IntentType, domain)
		if len(fewShotExamples) > 0 {
			optimizations = append(optimizations, "few_shot_examples")
			pb.updateMetrics(func(m *PromptBuilderMetrics) {
				m.FewShotExamplesUsed += int64(len(fewShotExamples))
			})
		}
	}

	// Step 6: Build user prompt with context
	userPrompt := pb.buildUserPrompt(enhancedQuery, contextText, fewShotExamples, request.CustomInstructions)

	// Step 7: Combine into full prompt
	fullPrompt := pb.combinePrompts(systemPrompt, userPrompt, request.ModelName)

	// Step 8: Optimize for token budget
	if pb.config.EnableContextOptimization {
		maxTokens := request.MaxTokens
		if maxTokens == 0 {
			maxTokens = pb.config.MaxPromptTokens
		}

		optimizedPrompt, tokenOptimizations, err := pb.optimizeForTokens(fullPrompt, systemPrompt, userPrompt, maxTokens, request.ModelName)
		if err != nil {
			pb.logger.Warn("Token optimization failed", "error", err)
		} else {
			fullPrompt = optimizedPrompt
			optimizations = append(optimizations, tokenOptimizations...)
		}
	}

	// Calculate final token count
	tokenCount := pb.tokenManager.EstimateTokensForModel(fullPrompt, request.ModelName)

	// Create response
	response := &PromptResponse{
		SystemPrompt:         systemPrompt,
		UserPrompt:           userPrompt,
		FullPrompt:           fullPrompt,
		TokenCount:           tokenCount,
		ContextSources:       contextSources,
		EnhancedQuery:        enhancedQuery,
		FewShotExamples:      fewShotExamples,
		ProcessingTime:       time.Since(startTime),
		OptimizationsApplied: optimizations,
		CacheUsed:            false, // Cache implementation available but not used in this context
		Metadata: map[string]interface{}{
			"domain":                domain,
			"template_used":         pb.getTemplateType(request),
			"context_sources_count": len(request.RAGContext),
			"few_shot_count":        len(fewShotExamples),
		},
	}

	// Update metrics
	pb.updateMetrics(func(m *PromptBuilderMetrics) {
		m.AverageProcessingTime = (m.AverageProcessingTime*time.Duration(m.TotalPrompts-1) + response.ProcessingTime) / time.Duration(m.TotalPrompts)
		m.AveragePromptTokens = int((float64(m.AveragePromptTokens)*float64(m.TotalPrompts-1) + float64(tokenCount)) / float64(m.TotalPrompts))
		if len(optimizations) > 0 {
			m.ContextOptimizations++
		}
		m.LastUpdated = time.Now()
	})

	pb.logger.Info("RAG-aware prompt built successfully",
		"token_count", tokenCount,
		"optimizations", len(optimizations),
		"processing_time", response.ProcessingTime,
	)

	return response, nil
}

// validateRequest validates the prompt building request
func (pb *RAGAwarePromptBuilder) validateRequest(request *PromptRequest) error {
	if request == nil {
		return fmt.Errorf("prompt request cannot be nil")
	}
	if request.Query == "" {
		return fmt.Errorf("query cannot be empty")
	}
	if request.ModelName == "" {
		return fmt.Errorf("model name cannot be empty")
	}
	return nil
}

// enhanceQuery enhances the query with telecom-specific knowledge
func (pb *RAGAwarePromptBuilder) enhanceQuery(query, intentType string) (string, []string) {
	var optimizations []string
	enhancedQuery := query

	// Expand abbreviations
	if pb.config.EnableAbbreviationExpansion {
		expanded, changed := pb.telecomQueryEnhancer.ExpandAbbreviations(enhancedQuery)
		if changed {
			enhancedQuery = expanded
			optimizations = append(optimizations, "abbreviation_expansion")
			pb.updateMetrics(func(m *PromptBuilderMetrics) {
				m.AbbreviationExpansions++
			})
		}
	}

	// Map standards references
	if pb.config.EnableStandardsMapping {
		mapped, changed := pb.telecomQueryEnhancer.MapStandards(enhancedQuery)
		if changed {
			enhancedQuery = mapped
			optimizations = append(optimizations, "standards_mapping")
			pb.updateMetrics(func(m *PromptBuilderMetrics) {
				m.StandardsMappings++
			})
		}
	}

	// Add domain-specific context clues
	if intentType != "" {
		contextualEnhancement := pb.addContextualHints(enhancedQuery, intentType)
		if contextualEnhancement != enhancedQuery {
			enhancedQuery = contextualEnhancement
			optimizations = append(optimizations, "contextual_enhancement")
		}
	}

	return enhancedQuery, optimizations
}

// addContextualHints adds contextual hints based on intent type
func (pb *RAGAwarePromptBuilder) addContextualHints(query, intentType string) string {
	switch strings.ToLower(intentType) {
	case "configuration":
		if !strings.Contains(strings.ToLower(query), "configure") && !strings.Contains(strings.ToLower(query), "config") {
			return fmt.Sprintf("How to configure: %s", query)
		}
	case "troubleshooting":
		if !strings.Contains(strings.ToLower(query), "troubleshoot") && !strings.Contains(strings.ToLower(query), "debug") {
			return fmt.Sprintf("Troubleshooting issue: %s", query)
		}
	case "optimization":
		if !strings.Contains(strings.ToLower(query), "optimize") && !strings.Contains(strings.ToLower(query), "improve") {
			return fmt.Sprintf("How to optimize: %s", query)
		}
	case "monitoring":
		if !strings.Contains(strings.ToLower(query), "monitor") && !strings.Contains(strings.ToLower(query), "observe") {
			return fmt.Sprintf("How to monitor: %s", query)
		}
	}
	return query
}

// buildSystemPrompt builds the system prompt based on request parameters
func (pb *RAGAwarePromptBuilder) buildSystemPrompt(request *PromptRequest, domain string) (string, error) {
	templateType := pb.getTemplateType(request)
	template, err := pb.promptTemplates.GetSystemPrompt(templateType)
	if err != nil {
		return "", fmt.Errorf("failed to get system prompt template: %w", err)
	}

	// Replace template variables
	systemPrompt := template
	systemPrompt = strings.ReplaceAll(systemPrompt, "{domain}", domain)
	systemPrompt = strings.ReplaceAll(systemPrompt, "{intent_type}", request.IntentType)

	// Add model-specific optimizations
	if !pb.tokenManager.SupportsSystemPrompt(request.ModelName) {
		// For models that don't support system prompts, we'll integrate it into user prompt
		return "", nil
	}

	return systemPrompt, nil
}

// getTemplateType determines the appropriate template type
func (pb *RAGAwarePromptBuilder) getTemplateType(request *PromptRequest) string {
	if request.TemplateType != "" {
		return request.TemplateType
	}

	// Determine based on intent type and domain
	if request.IntentType != "" {
		switch strings.ToLower(request.IntentType) {
		case "configuration":
			return "telecom_configuration"
		case "troubleshooting":
			return "telecom_troubleshooting"
		case "optimization":
			return "telecom_optimization"
		case "monitoring":
			return "telecom_monitoring"
		}
	}

	return pb.config.DefaultTemplate
}

// processRAGContext processes RAG context and creates formatted context text
func (pb *RAGAwarePromptBuilder) processRAGContext(ragContext []*shared.SearchResult, modelName string) (string, []string, []string) {
	if len(ragContext) == 0 {
		return "", nil, nil
	}

	var optimizations []string
	var contextParts []string
	var sources []string

	// Filter by relevance threshold
	filteredContext := make([]*shared.SearchResult, 0)
	for _, result := range ragContext {
		if result.Score >= pb.config.ContextRelevanceThreshold {
			filteredContext = append(filteredContext, result)
		}
	}

	if len(filteredContext) < len(ragContext) {
		optimizations = append(optimizations, "relevance_filtering")
	}

	// Limit to max sources
	maxSources := pb.config.MaxContextSources
	if len(filteredContext) > maxSources {
		filteredContext = filteredContext[:maxSources]
		optimizations = append(optimizations, "source_limiting")
	}

	// Build context text
	for i, result := range filteredContext {
		if result.Document == nil {
			continue
		}

		// Format document for context
		docText := pb.formatDocumentForContext(result, i+1)
		contextParts = append(contextParts, docText)

		// Create source attribution - use simple string formatting
		source := fmt.Sprintf("[%d] %s - %s", i+1, result.Document.Source, result.Document.Title)
		sources = append(sources, source)
	}

	contextText := ""
	if len(contextParts) > 0 {
		contextText = pb.config.ContextPreamble + pb.config.ContextSeparator +
			strings.Join(contextParts, pb.config.ContextSeparator)
	}

	return contextText, sources, optimizations
}

// formatDocumentForContext formats a document for inclusion in context
func (pb *RAGAwarePromptBuilder) formatDocumentForContext(result *shared.SearchResult, index int) string {
	doc := result.Document

	var parts []string
	parts = append(parts, fmt.Sprintf("Source %d:", index))

	if doc.Title != "" {
		parts = append(parts, fmt.Sprintf("Title: %s", doc.Title))
	}
	if doc.Source != "" {
		parts = append(parts, fmt.Sprintf("Source: %s", doc.Source))
	}
	if doc.Version != "" {
		parts = append(parts, fmt.Sprintf("Version: %s", doc.Version))
	}
	if len(doc.Technology) > 0 {
		parts = append(parts, fmt.Sprintf("Technology: %s", strings.Join(doc.Technology, ", ")))
	}

	parts = append(parts, fmt.Sprintf("Relevance: %.3f", result.Score))
	parts = append(parts, "")
	parts = append(parts, doc.Content)

	return strings.Join(parts, "\n")
}

// selectFewShotExamples selects appropriate few-shot examples
func (pb *RAGAwarePromptBuilder) selectFewShotExamples(query, intentType, domain string) []FewShotExample {
	examples := pb.promptTemplates.GetFewShotExamples(intentType, domain)
	if len(examples) == 0 {
		return nil
	}

	// Score examples by similarity to query
	type scoredExample struct {
		example FewShotExample
		score   float32
	}

	var scoredExamples []scoredExample
	queryLower := strings.ToLower(query)

	for _, example := range examples {
		// Simple similarity scoring based on keyword overlap
		score := pb.calculateExampleSimilarity(queryLower, strings.ToLower(example.Query))
		scoredExamples = append(scoredExamples, scoredExample{
			example: example,
			score:   score,
		})
	}

	// Sort by score (highest first)
	for i := 0; i < len(scoredExamples)-1; i++ {
		for j := i + 1; j < len(scoredExamples); j++ {
			if scoredExamples[i].score < scoredExamples[j].score {
				scoredExamples[i], scoredExamples[j] = scoredExamples[j], scoredExamples[i]
			}
		}
	}

	// Select top examples
	maxExamples := pb.config.MaxFewShotExamples
	if len(scoredExamples) < maxExamples {
		maxExamples = len(scoredExamples)
	}

	selectedExamples := make([]FewShotExample, maxExamples)
	for i := 0; i < maxExamples; i++ {
		selectedExamples[i] = scoredExamples[i].example
	}

	return selectedExamples
}

// calculateExampleSimilarity calculates similarity between query and example
func (pb *RAGAwarePromptBuilder) calculateExampleSimilarity(query, exampleQuery string) float32 {
	queryWords := strings.Fields(query)
	exampleWords := strings.Fields(exampleQuery)

	if len(queryWords) == 0 || len(exampleWords) == 0 {
		return 0.0
	}

	matches := 0
	for _, qWord := range queryWords {
		for _, eWord := range exampleWords {
			if qWord == eWord {
				matches++
				break
			}
		}
	}

	return float32(matches) / float32(len(queryWords))
}

// buildUserPrompt builds the user prompt with context and examples
func (pb *RAGAwarePromptBuilder) buildUserPrompt(query, context string, fewShotExamples []FewShotExample, customInstructions string) string {
	var parts []string

	// Add few-shot examples first
	if len(fewShotExamples) > 0 {
		parts = append(parts, "Here are some examples of similar queries:")
		for i, example := range fewShotExamples {
			parts = append(parts, fmt.Sprintf("\nExample %d:", i+1))
			parts = append(parts, fmt.Sprintf("Q: %s", example.Query))
			parts = append(parts, fmt.Sprintf("A: %s", example.Response))
		}
		parts = append(parts, "")
	}

	// Add context if available
	if context != "" {
		parts = append(parts, context)
		parts = append(parts, "")
	}

	// Add custom instructions if provided
	if customInstructions != "" {
		parts = append(parts, "Additional Instructions:")
		parts = append(parts, customInstructions)
		parts = append(parts, "")
	}

	// Add the actual query
	parts = append(parts, "Question:")
	parts = append(parts, query)

	return strings.Join(parts, "\n")
}

// combinePrompts combines system and user prompts based on model capabilities
func (pb *RAGAwarePromptBuilder) combinePrompts(systemPrompt, userPrompt, modelName string) string {
	if !pb.tokenManager.SupportsChatFormat(modelName) {
		// For models that don't support chat format, combine into single prompt
		if systemPrompt != "" {
			return systemPrompt + "\n\n" + userPrompt
		}
		return userPrompt
	}

	// For chat models, we'll return the user prompt
	// The system prompt should be handled separately by the client
	return userPrompt
}

// optimizeForTokens optimizes the prompt to fit within token budget
func (pb *RAGAwarePromptBuilder) optimizeForTokens(fullPrompt, systemPrompt, userPrompt string, maxTokens int, modelName string) (string, []string, error) {
	currentTokens := pb.tokenManager.EstimateTokensForModel(fullPrompt, modelName)
	if currentTokens <= maxTokens {
		return fullPrompt, nil, nil
	}

	var optimizations []string
	optimizedPrompt := fullPrompt

	// Reserve tokens for system prompt
	availableTokens := maxTokens - pb.tokenManager.EstimateTokensForModel(systemPrompt, modelName)

	// Truncate user prompt if necessary
	if pb.tokenManager.EstimateTokensForModel(userPrompt, modelName) > availableTokens {
		truncatedUserPrompt := pb.tokenManager.TruncateToFit(userPrompt, availableTokens, modelName)
		optimizedPrompt = pb.combinePrompts(systemPrompt, truncatedUserPrompt, modelName)
		optimizations = append(optimizations, "user_prompt_truncation")
	}

	return optimizedPrompt, optimizations, nil
}

// updateMetrics safely updates metrics
func (pb *RAGAwarePromptBuilder) updateMetrics(updater func(*PromptBuilderMetrics)) {
	pb.metrics.mutex.Lock()
	defer pb.metrics.mutex.Unlock()
	updater(pb.metrics)
}

// GetMetrics returns current metrics
func (pb *RAGAwarePromptBuilder) GetMetrics() *PromptBuilderMetrics {
	pb.metrics.mutex.RLock()
	defer pb.metrics.mutex.RUnlock()

	metrics := *pb.metrics
	return &metrics
}

// NewTelecomQueryEnhancer creates a new telecom query enhancer
func NewTelecomQueryEnhancer() *TelecomQueryEnhancer {
	return &TelecomQueryEnhancer{
		abbreviations:    initializeAbbreviations(),
		standardsMapping: initializeStandardsMapping(),
		protocolMappings: initializeProtocolMappings(),
		domainClassifier: NewDomainClassifier(),
		logger:           slog.Default().With("component", "telecom-query-enhancer"),
	}
}

// ExpandAbbreviations expands telecom abbreviations in the query
func (tqe *TelecomQueryEnhancer) ExpandAbbreviations(query string) (string, bool) {
	expandedQuery := query
	changed := false

	for abbrev, expansion := range tqe.abbreviations {
		// Use word boundaries to avoid partial matches
		pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(abbrev))
		re := regexp.MustCompile(pattern)

		if re.MatchString(expandedQuery) {
			expandedQuery = re.ReplaceAllString(expandedQuery, fmt.Sprintf("%s (%s)", expansion, abbrev))
			changed = true
		}
	}

	return expandedQuery, changed
}

// MapStandards maps standards references to full information
func (tqe *TelecomQueryEnhancer) MapStandards(query string) (string, bool) {
	mappedQuery := query
	changed := false

	for pattern, info := range tqe.standardsMapping {
		re := regexp.MustCompile(pattern)
		if re.MatchString(mappedQuery) {
			enhancement := fmt.Sprintf(" (%s - %s)", info.Organization, info.FullName)
			mappedQuery = re.ReplaceAllString(mappedQuery, "${0}"+enhancement)
			changed = true
		}
	}

	return mappedQuery, changed
}

// ClassifyDomain classifies the query into telecom domains
func (tqe *TelecomQueryEnhancer) ClassifyDomain(query string) string {
	return tqe.domainClassifier.Classify(query)
}

// Initialize abbreviations map
func initializeAbbreviations() map[string]string {
	return map[string]string{
		"5G":   "Fifth Generation",
		"4G":   "Fourth Generation",
		"LTE":  "Long Term Evolution",
		"NR":   "New Radio",
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
		"RAN":  "Radio Access Network",
		"CN":   "Core Network",
		"PLMN": "Public Land Mobile Network",
		"QoS":  "Quality of Service",
		"KPI":  "Key Performance Indicator",
		"SLA":  "Service Level Agreement",
		"API":  "Application Programming Interface",
		"REST": "Representational State Transfer",
		"HTTP": "Hypertext Transfer Protocol",
		"JSON": "JavaScript Object Notation",
		"XML":  "eXtensible Markup Language",
		"YAML": "YAML Ain't Markup Language",
		"CLI":  "Command Line Interface",
		"GUI":  "Graphical User Interface",
		"SDN":  "Software Defined Networking",
		"NFV":  "Network Function Virtualization",
		"VNF":  "Virtual Network Function",
		"CNF":  "Cloud Native Function",
		"MANO": "Management and Orchestration",
		"VNFM": "VNF Manager",
		"NFVO": "NFV Orchestrator",
		"VIM":  "Virtual Infrastructure Manager",
	}
}

// Initialize standards mapping
func initializeStandardsMapping() map[string]StandardInfo {
	return map[string]StandardInfo{
		`TS\s*38\.\d+`: {
			FullName:     "5G NR Technical Specification",
			Organization: "3GPP",
			Domain:       "RAN",
			Description:  "5G New Radio specifications",
		},
		`TS\s*23\.\d+`: {
			FullName:     "System Architecture Technical Specification",
			Organization: "3GPP",
			Domain:       "Core",
			Description:  "System architecture and procedures",
		},
		`TS\s*29\.\d+`: {
			FullName:     "Protocol Technical Specification",
			Organization: "3GPP",
			Domain:       "Core",
			Description:  "Protocol specifications",
		},
		`O-RAN\.WG\d+`: {
			FullName:     "O-RAN Working Group Specification",
			Organization: "O-RAN Alliance",
			Domain:       "O-RAN",
			Description:  "Open RAN specifications",
		},
	}
}

// Initialize protocol mappings
func initializeProtocolMappings() map[string]ProtocolInfo {
	return map[string]ProtocolInfo{
		"NAS": {
			FullName:  "Non-Access Stratum",
			Layer:     "Layer 3",
			Function:  "Mobility and session management",
			Standards: []string{"TS 24.501"},
		},
		"RRC": {
			FullName:  "Radio Resource Control",
			Layer:     "Layer 3",
			Function:  "Radio resource management",
			Standards: []string{"TS 38.331"},
		},
		"NGAP": {
			FullName:  "NG Application Protocol",
			Layer:     "Application",
			Function:  "Interface between gNB and AMF",
			Standards: []string{"TS 38.413"},
		},
	}
}

// NewDomainClassifier creates a new domain classifier
func NewDomainClassifier() *DomainClassifier {
	return &DomainClassifier{
		domainKeywords: map[string][]string{
			"RAN":        {"radio", "antenna", "gnb", "enb", "cell", "handover", "mobility", "rf", "baseband"},
			"Core":       {"amf", "smf", "upf", "ausf", "udm", "pcf", "nrf", "session", "authentication"},
			"Transport":  {"ip", "mpls", "ethernet", "optical", "backhaul", "fronthaul", "routing", "switching"},
			"Management": {"orchestration", "automation", "monitoring", "configuration", "provisioning"},
			"O-RAN":      {"o-ran", "oran", "open", "disaggregated", "virtualized", "cloudified", "ric"},
		},
	}
}

// Classify classifies a query into telecom domains
func (dc *DomainClassifier) Classify(query string) string {
	queryLower := strings.ToLower(query)

	domainScores := make(map[string]int)

	// Score domains based on keyword matches
	for domain, keywords := range dc.domainKeywords {
		score := 0
		for _, keyword := range keywords {
			if strings.Contains(queryLower, keyword) {
				score++
			}
		}
		domainScores[domain] = score
	}

	// Find domain with highest score
	maxScore := 0
	bestDomain := "General"

	for domain, score := range domainScores {
		if score > maxScore {
			maxScore = score
			bestDomain = domain
		}
	}

	return bestDomain
}

// NewTelecomPromptTemplates creates new telecom prompt templates
func NewTelecomPromptTemplates() *TelecomPromptTemplates {
	return &TelecomPromptTemplates{
		systemPrompts:    initializeSystemPrompts(),
		contextTemplates: initializeContextTemplates(),
		fewShotExamples:  initializeFewShotExamples(),
	}
}

// GetSystemPrompt gets a system prompt by type
func (tpt *TelecomPromptTemplates) GetSystemPrompt(templateType string) (string, error) {
	tpt.mutex.RLock()
	defer tpt.mutex.RUnlock()

	template, exists := tpt.systemPrompts[templateType]
	if !exists {
		// Return default template
		if defaultTemplate, hasDefault := tpt.systemPrompts["telecom_expert"]; hasDefault {
			return defaultTemplate, nil
		}
		return "", fmt.Errorf("template not found: %s", templateType)
	}

	return template, nil
}

// GetFewShotExamples gets few-shot examples for intent type and domain
func (tpt *TelecomPromptTemplates) GetFewShotExamples(intentType, domain string) []FewShotExample {
	tpt.mutex.RLock()
	defer tpt.mutex.RUnlock()

	key := fmt.Sprintf("%s_%s", intentType, domain)
	if examples, exists := tpt.fewShotExamples[key]; exists {
		return examples
	}

	// Try with just intent type
	if examples, exists := tpt.fewShotExamples[intentType]; exists {
		return examples
	}

	// Return general examples
	if examples, exists := tpt.fewShotExamples["general"]; exists {
		return examples
	}

	return nil
}

// Initialize system prompts
func initializeSystemPrompts() map[string]string {
	return map[string]string{
		"telecom_expert": `You are an expert telecommunications engineer and system administrator with deep knowledge of 5G, 4G, O-RAN, and network infrastructure. Your role is to provide accurate, actionable answers based on technical documentation and industry best practices.

Guidelines:
1. Provide technically accurate information appropriate for telecom professionals
2. Reference specific standards, specifications, or working groups when relevant
3. Use precise technical terminology and explain complex concepts clearly
4. Focus on practical, implementable solutions
5. Consider the network domain (RAN, Core, Transport, Management) in your responses
6. Clearly distinguish between facts from documentation and general industry knowledge
7. When information is insufficient, state this clearly and suggest additional resources`,

		"telecom_configuration": `You are a telecommunications configuration specialist with expertise in network element configuration, parameter optimization, and system integration. Focus on providing detailed configuration guidance, parameter explanations, and best practices.

Your responses should include:
- Specific configuration parameters and their values
- Step-by-step configuration procedures
- Potential impacts of configuration changes
- Validation steps and testing recommendations
- Common configuration pitfalls to avoid`,

		"telecom_troubleshooting": `You are a telecommunications troubleshooting expert specializing in network fault analysis, performance issues, and system diagnostics. Your role is to help identify root causes and provide systematic resolution approaches.

Your responses should include:
- Systematic diagnostic procedures
- Key metrics and KPIs to examine
- Common root causes for similar issues
- Step-by-step resolution procedures
- Prevention measures and monitoring recommendations`,
	}
}

// Initialize context templates
func initializeContextTemplates() map[string]string {
	return map[string]string{
		"default":         "Based on the following technical documentation:",
		"troubleshooting": "Based on the following troubleshooting guides and technical documentation:",
		"configuration":   "Based on the following configuration guides and specifications:",
	}
}

// Initialize few-shot examples
func initializeFewShotExamples() map[string][]FewShotExample {
	return map[string][]FewShotExample{
		"configuration": {
			{
				Query:      "How do I configure QoS parameters for 5G network slicing?",
				Response:   "To configure QoS parameters for 5G network slicing, you need to define QoS flows with specific 5QI values, configure PDU session parameters, and set up policy rules...",
				Domain:     "Core",
				IntentType: "configuration",
				Relevance:  0.9,
			},
		},
		"troubleshooting": {
			{
				Query:      "How do I troubleshoot handover failures in 5G networks?",
				Response:   "To troubleshoot handover failures, first check the handover success rate KPIs, analyze X2/Xn interface status, verify neighbor cell configurations...",
				Domain:     "RAN",
				IntentType: "troubleshooting",
				Relevance:  0.95,
			},
		},
	}
}
