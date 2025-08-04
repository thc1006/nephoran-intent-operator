package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
	
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// DynamicContextManager handles intelligent context injection with budget management
type DynamicContextManager struct {
	tokenManager      *TokenManager
	contextBuilder    *ContextBuilder
	promptRegistry    *TelecomPromptRegistry
	documentCache     *DocumentCache
	config            *DynamicContextConfig
	logger            *slog.Logger
	metrics           *ContextMetrics
	mu                sync.RWMutex
}

// DynamicContextConfig configures the dynamic context manager
type DynamicContextConfig struct {
	// Budget Management
	MaxContextTokens       int                        `json:"max_context_tokens"`
	MinContextTokens       int                        `json:"min_context_tokens"`
	TokenAllocationStrategy string                    `json:"token_allocation_strategy"` // "proportional", "priority", "adaptive"
	CompressionEnabled     bool                       `json:"compression_enabled"`
	CompressionRatio       float64                    `json:"compression_ratio"`
	
	// Context Prioritization
	PriorityWeights        map[string]float64         `json:"priority_weights"`
	RecencyBias            float64                    `json:"recency_bias"`
	RelevanceThreshold     float32                    `json:"relevance_threshold"`
	DiversityFactor        float64                    `json:"diversity_factor"`
	
	// Telecom-Specific Settings
	TelecomDomainPriority  map[string]float64         `json:"telecom_domain_priority"`
	StandardsPreference    []string                   `json:"standards_preference"`
	NetworkStateWeight     float64                    `json:"network_state_weight"`
	ComplianceWeight       float64                    `json:"compliance_weight"`
	
	// Multi-Tier Strategy
	TierDefinitions        []ContextTier              `json:"tier_definitions"`
	TierSelectionMode      string                     `json:"tier_selection_mode"` // "auto", "manual", "adaptive"
	
	// Performance Settings
	CacheEnabled           bool                       `json:"cache_enabled"`
	CacheTTL               time.Duration              `json:"cache_ttl"`
	ParallelProcessing     bool                       `json:"parallel_processing"`
	MaxProcessingTime      time.Duration              `json:"max_processing_time"`
}

// ContextTier defines a tier in multi-tier prompting strategy
type ContextTier struct {
	Name              string                 `json:"name"`
	MaxTokens         int                    `json:"max_tokens"`
	MinTokens         int                    `json:"min_tokens"`
	ComplexityLevel   string                 `json:"complexity_level"` // "simple", "moderate", "complex"
	IncludeExamples   bool                   `json:"include_examples"`
	IncludeMetadata   bool                   `json:"include_metadata"`
	CompressionLevel  string                 `json:"compression_level"` // "none", "light", "moderate", "heavy"
	RequiredSources   []string               `json:"required_sources"`
	Priority          int                    `json:"priority"`
}

// DocumentCache caches processed documents to improve performance
type DocumentCache struct {
	cache    map[string]*CachedDocument
	mu       sync.RWMutex
	maxSize  int
	ttl      time.Duration
}

// CachedDocument represents a cached document with metadata
type CachedDocument struct {
	Document        *shared.TelecomDocument `json:"document"`
	ProcessedText   string                  `json:"processed_text"`
	TokenCount      int                     `json:"token_count"`
	CompressedText  string                  `json:"compressed_text"`
	CompressedTokens int                    `json:"compressed_tokens"`
	LastAccessed    time.Time               `json:"last_accessed"`
	AccessCount     int                     `json:"access_count"`
}

// ContextInjectionRequest represents a request for dynamic context injection
type ContextInjectionRequest struct {
	Intent            string                      `json:"intent"`
	IntentType        string                      `json:"intent_type"`
	ModelName         string                      `json:"model_name"`
	NetworkState      *NetworkStateContext        `json:"network_state"`
	ComplianceReqs    []string                    `json:"compliance_requirements"`
	UserPreferences   map[string]interface{}      `json:"user_preferences"`
	MaxTokenBudget    int                         `json:"max_token_budget"`
	PreferredTier     string                      `json:"preferred_tier"`
	RequiredDocuments []string                    `json:"required_documents"`
	ExcludeDocuments  []string                    `json:"exclude_documents"`
}

// NetworkStateContext represents current network state information
type NetworkStateContext struct {
	ActiveSlices      []NetworkSliceInfo          `json:"active_slices"`
	NetworkFunctions  []NetworkFunctionInfo       `json:"network_functions"`
	CurrentLoad       map[string]float64          `json:"current_load"`
	ActiveAlarms      []NetworkAlarm              `json:"active_alarms"`
	TopologyInfo      map[string]interface{}      `json:"topology_info"`
	PerformanceMetrics map[string]interface{}     `json:"performance_metrics"`
}

// NetworkSliceInfo represents information about a network slice
type NetworkSliceInfo struct {
	SliceID          string                 `json:"slice_id"`
	SST              int                    `json:"sst"`
	SD               string                 `json:"sd"`
	Status           string                 `json:"status"`
	ActiveSessions   int                    `json:"active_sessions"`
	ResourceUsage    map[string]float64     `json:"resource_usage"`
}

// NetworkFunctionInfo represents information about a network function
type NetworkFunctionInfo struct {
	Name             string                 `json:"name"`
	Type             string                 `json:"type"`
	Version          string                 `json:"version"`
	Status           string                 `json:"status"`
	Namespace        string                 `json:"namespace"`
	Resources        map[string]interface{} `json:"resources"`
}

// NetworkAlarm represents an active network alarm
type NetworkAlarm struct {
	ID               string                 `json:"id"`
	Severity         string                 `json:"severity"`
	Component        string                 `json:"component"`
	Description      string                 `json:"description"`
	Timestamp        time.Time              `json:"timestamp"`
}

// ContextInjectionResult represents the result of context injection
type ContextInjectionResult struct {
	FinalPrompt      string                      `json:"final_prompt"`
	SystemPrompt     string                      `json:"system_prompt"`
	UserPrompt       string                      `json:"user_prompt"`
	InjectedContext  string                      `json:"injected_context"`
	TokenUsage       TokenUsageBreakdown         `json:"token_usage"`
	SelectedTier     string                      `json:"selected_tier"`
	DocumentsUsed    []*DocumentReference        `json:"documents_used"`
	CompressionApplied bool                     `json:"compression_applied"`
	ProcessingTime   time.Duration               `json:"processing_time"`
	QualityScore     float32                     `json:"quality_score"`
	Metadata         map[string]interface{}      `json:"metadata"`
}

// TokenUsageBreakdown provides detailed token usage information
type TokenUsageBreakdown struct {
	SystemPromptTokens   int     `json:"system_prompt_tokens"`
	UserPromptTokens     int     `json:"user_prompt_tokens"`
	ContextTokens        int     `json:"context_tokens"`
	NetworkStateTokens   int     `json:"network_state_tokens"`
	ExampleTokens        int     `json:"example_tokens"`
	TotalTokens          int     `json:"total_tokens"`
	BudgetUtilization    float64 `json:"budget_utilization"`
	CompressionSavings   int     `json:"compression_savings"`
}

// NewDynamicContextManager creates a new dynamic context manager
func NewDynamicContextManager(
	tokenManager *TokenManager,
	contextBuilder *ContextBuilder,
	promptRegistry *TelecomPromptRegistry,
	config *DynamicContextConfig,
) *DynamicContextManager {
	if config == nil {
		config = getDefaultDynamicContextConfig()
	}
	
	return &DynamicContextManager{
		tokenManager:   tokenManager,
		contextBuilder: contextBuilder,
		promptRegistry: promptRegistry,
		documentCache:  NewDocumentCache(1000, config.CacheTTL),
		config:         config,
		logger:         slog.Default().With("component", "dynamic-context-manager"),
		metrics:        &ContextMetrics{LastUpdated: time.Now()},
	}
}

// getDefaultDynamicContextConfig returns default configuration
func getDefaultDynamicContextConfig() *DynamicContextConfig {
	return &DynamicContextConfig{
		MaxContextTokens:        8000,
		MinContextTokens:        1000,
		TokenAllocationStrategy: "adaptive",
		CompressionEnabled:      true,
		CompressionRatio:        0.7,
		PriorityWeights: map[string]float64{
			"relevance":   0.4,
			"recency":     0.2,
			"authority":   0.2,
			"diversity":   0.1,
			"compliance":  0.1,
		},
		RecencyBias:        0.3,
		RelevanceThreshold: 0.4,
		DiversityFactor:    0.2,
		TelecomDomainPriority: map[string]float64{
			"O-RAN":     1.0,
			"5G-Core":   0.9,
			"RAN":       0.85,
			"Transport": 0.7,
			"OSS/BSS":   0.6,
		},
		StandardsPreference: []string{"3GPP", "O-RAN", "ETSI", "IETF"},
		NetworkStateWeight:  0.3,
		ComplianceWeight:    0.2,
		TierDefinitions: []ContextTier{
			{
				Name:             "minimal",
				MaxTokens:        2000,
				MinTokens:        500,
				ComplexityLevel:  "simple",
				IncludeExamples:  false,
				IncludeMetadata:  false,
				CompressionLevel: "heavy",
				Priority:         1,
			},
			{
				Name:             "standard",
				MaxTokens:        4000,
				MinTokens:        1000,
				ComplexityLevel:  "moderate",
				IncludeExamples:  true,
				IncludeMetadata:  true,
				CompressionLevel: "moderate",
				Priority:         2,
			},
			{
				Name:             "comprehensive",
				MaxTokens:        8000,
				MinTokens:        2000,
				ComplexityLevel:  "complex",
				IncludeExamples:  true,
				IncludeMetadata:  true,
				CompressionLevel: "light",
				Priority:         3,
			},
		},
		TierSelectionMode:  "adaptive",
		CacheEnabled:       true,
		CacheTTL:           15 * time.Minute,
		ParallelProcessing: true,
		MaxProcessingTime:  5 * time.Second,
	}
}

// InjectContext performs dynamic context injection with budget management
func (dcm *DynamicContextManager) InjectContext(ctx context.Context, request *ContextInjectionRequest) (*ContextInjectionResult, error) {
	startTime := time.Now()
	
	// Validate request
	if err := dcm.validateRequest(request); err != nil {
		return nil, fmt.Errorf("invalid context injection request: %w", err)
	}
	
	// Select appropriate tier
	selectedTier := dcm.selectContextTier(request)
	dcm.logger.Info("Selected context tier",
		"tier", selectedTier.Name,
		"max_tokens", selectedTier.MaxTokens,
		"complexity", selectedTier.ComplexityLevel,
	)
	
	// Get prompt template
	templates := dcm.promptRegistry.GetTemplatesByIntent(request.IntentType)
	if len(templates) == 0 {
		return nil, fmt.Errorf("no template found for intent type: %s", request.IntentType)
	}
	template := dcm.selectBestTemplate(templates, request)
	
	// Calculate token budget
	tokenBudget := dcm.calculateTokenBudget(request, selectedTier, template)
	
	// Build base prompts
	systemPrompt := template.SystemPrompt
	userPrompt := dcm.buildUserPrompt(template, request)
	
	// Estimate base token usage
	baseTokens := dcm.tokenManager.EstimateTokensForModel(systemPrompt+userPrompt, request.ModelName)
	availableForContext := tokenBudget.MaxContextTokens - baseTokens
	
	// Gather and inject context
	injectedContext, documentsUsed, contextTokens := dcm.gatherAndInjectContext(
		ctx, request, selectedTier, availableForContext,
	)
	
	// Add network state context if available
	networkStateContext, networkTokens := dcm.buildNetworkStateContext(request.NetworkState, request.ModelName)
	if networkStateContext != "" && networkTokens < availableForContext-contextTokens {
		injectedContext = networkStateContext + "\n\n" + injectedContext
		contextTokens += networkTokens
	}
	
	// Apply compression if needed
	compressionApplied := false
	compressionSavings := 0
	if selectedTier.CompressionLevel != "none" && contextTokens > tokenBudget.OptimalContextTokens {
		compressedContext, compressedTokens := dcm.compressContext(
			injectedContext, selectedTier.CompressionLevel, request.ModelName,
		)
		if compressedTokens < contextTokens {
			compressionSavings = contextTokens - compressedTokens
			injectedContext = compressedContext
			contextTokens = compressedTokens
			compressionApplied = true
		}
	}
	
	// Add examples if tier allows and budget permits
	exampleTokens := 0
	if selectedTier.IncludeExamples && len(template.Examples) > 0 {
		examples, exTokens := dcm.selectExamples(
			template.Examples, request, availableForContext-contextTokens, request.ModelName,
		)
		if len(examples) > 0 {
			exampleText := dcm.formatExamples(examples)
			userPrompt += "\n\n" + exampleText
			exampleTokens = exTokens
		}
	}
	
	// Build final prompt
	finalPrompt := dcm.assembleFinalPrompt(systemPrompt, userPrompt, injectedContext)
	
	// Calculate final token usage
	totalTokens := dcm.tokenManager.EstimateTokensForModel(finalPrompt, request.ModelName)
	
	// Create token usage breakdown
	tokenUsage := TokenUsageBreakdown{
		SystemPromptTokens: dcm.tokenManager.EstimateTokensForModel(systemPrompt, request.ModelName),
		UserPromptTokens:   dcm.tokenManager.EstimateTokensForModel(userPrompt, request.ModelName) - exampleTokens,
		ContextTokens:      contextTokens,
		NetworkStateTokens: networkTokens,
		ExampleTokens:      exampleTokens,
		TotalTokens:        totalTokens,
		BudgetUtilization:  float64(totalTokens) / float64(tokenBudget.MaxContextTokens),
		CompressionSavings: compressionSavings,
	}
	
	// Calculate quality score
	qualityScore := dcm.calculateQualityScore(
		documentsUsed, selectedTier, tokenUsage, compressionApplied,
	)
	
	// Create result
	result := &ContextInjectionResult{
		FinalPrompt:        finalPrompt,
		SystemPrompt:       systemPrompt,
		UserPrompt:         userPrompt,
		InjectedContext:    injectedContext,
		TokenUsage:         tokenUsage,
		SelectedTier:       selectedTier.Name,
		DocumentsUsed:      documentsUsed,
		CompressionApplied: compressionApplied,
		ProcessingTime:     time.Since(startTime),
		QualityScore:       qualityScore,
		Metadata: map[string]interface{}{
			"template_id":        template.ID,
			"network_state_included": networkStateContext != "",
			"examples_included":   exampleTokens > 0,
			"cache_hits":         0, // TODO: track cache hits
		},
	}
	
	dcm.logger.Info("Context injection completed",
		"tier", selectedTier.Name,
		"total_tokens", totalTokens,
		"quality_score", qualityScore,
		"processing_time", result.ProcessingTime,
	)
	
	return result, nil
}

// validateRequest validates the context injection request
func (dcm *DynamicContextManager) validateRequest(request *ContextInjectionRequest) error {
	if request.Intent == "" {
		return fmt.Errorf("intent cannot be empty")
	}
	if request.IntentType == "" {
		return fmt.Errorf("intent type cannot be empty")
	}
	if request.ModelName == "" {
		return fmt.Errorf("model name cannot be empty")
	}
	return nil
}

// selectContextTier selects the appropriate context tier
func (dcm *DynamicContextManager) selectContextTier(request *ContextInjectionRequest) *ContextTier {
	switch dcm.config.TierSelectionMode {
	case "manual":
		if request.PreferredTier != "" {
			for _, tier := range dcm.config.TierDefinitions {
				if tier.Name == request.PreferredTier {
					return &tier
				}
			}
		}
	case "adaptive":
		// Analyze intent complexity
		complexity := dcm.analyzeIntentComplexity(request)
		
		// Select tier based on complexity and budget
		for _, tier := range dcm.config.TierDefinitions {
			if tier.ComplexityLevel == complexity {
				if request.MaxTokenBudget == 0 || tier.MaxTokens <= request.MaxTokenBudget {
					return &tier
				}
			}
		}
	}
	
	// Default to standard tier
	for _, tier := range dcm.config.TierDefinitions {
		if tier.Name == "standard" {
			return &tier
		}
	}
	
	// Fallback to first tier
	return &dcm.config.TierDefinitions[0]
}

// analyzeIntentComplexity analyzes the complexity of an intent
func (dcm *DynamicContextManager) analyzeIntentComplexity(request *ContextInjectionRequest) string {
	// Simple heuristics for complexity analysis
	complexityScore := 0
	
	// Check intent length
	if len(request.Intent) > 200 {
		complexityScore += 2
	} else if len(request.Intent) > 100 {
		complexityScore += 1
	}
	
	// Check for technical terms
	technicalTerms := []string{
		"O-RAN", "xApp", "rApp", "Near-RT RIC", "Non-RT RIC",
		"network slice", "URLLC", "eMBB", "mMTC",
		"beamforming", "MIMO", "carrier aggregation",
		"E2 interface", "A1 policy", "O1 management",
	}
	
	lowerIntent := strings.ToLower(request.Intent)
	for _, term := range technicalTerms {
		if strings.Contains(lowerIntent, strings.ToLower(term)) {
			complexityScore++
		}
	}
	
	// Check network state complexity
	if request.NetworkState != nil {
		if len(request.NetworkState.ActiveSlices) > 3 {
			complexityScore += 2
		}
		if len(request.NetworkState.ActiveAlarms) > 0 {
			complexityScore += 1
		}
	}
	
	// Check compliance requirements
	if len(request.ComplianceReqs) > 2 {
		complexityScore += 2
	} else if len(request.ComplianceReqs) > 0 {
		complexityScore += 1
	}
	
	// Map score to complexity level
	if complexityScore >= 6 {
		return "complex"
	} else if complexityScore >= 3 {
		return "moderate"
	}
	return "simple"
}

// TokenBudget represents token allocation for different components
type TokenBudget struct {
	MaxContextTokens     int
	OptimalContextTokens int
	MinContextTokens     int
	SystemPromptTokens   int
	UserPromptTokens     int
	ExampleTokens        int
	NetworkStateTokens   int
}

// calculateTokenBudget calculates token budget allocation
func (dcm *DynamicContextManager) calculateTokenBudget(
	request *ContextInjectionRequest,
	tier *ContextTier,
	template *TelecomPromptTemplate,
) *TokenBudget {
	// Get model configuration
	modelConfig, _ := dcm.tokenManager.GetModelConfig(request.ModelName)
	if modelConfig == nil {
		// Use defaults
		modelConfig = &ModelTokenConfig{
			ContextWindow: 8192,
			MaxTokens:     4096,
		}
	}
	
	// Start with tier limits or request budget
	maxTokens := tier.MaxTokens
	if request.MaxTokenBudget > 0 && request.MaxTokenBudget < maxTokens {
		maxTokens = request.MaxTokenBudget
	}
	
	// Ensure we don't exceed model limits
	if maxTokens > modelConfig.ContextWindow-modelConfig.MaxTokens {
		maxTokens = modelConfig.ContextWindow - modelConfig.MaxTokens
	}
	
	// Calculate optimal allocation
	systemPromptTokens := dcm.tokenManager.EstimateTokensForModel(template.SystemPrompt, request.ModelName)
	
	// Reserve tokens for different components
	reservedForResponse := int(float64(modelConfig.MaxTokens) * 0.5) // Reserve 50% for response
	reservedForPrompt := systemPromptTokens + 500 // System prompt + user prompt buffer
	
	optimalContextTokens := maxTokens - reservedForResponse - reservedForPrompt
	if optimalContextTokens < tier.MinTokens {
		optimalContextTokens = tier.MinTokens
	}
	
	return &TokenBudget{
		MaxContextTokens:     maxTokens,
		OptimalContextTokens: optimalContextTokens,
		MinContextTokens:     tier.MinTokens,
		SystemPromptTokens:   systemPromptTokens,
		UserPromptTokens:     500, // Estimated
		ExampleTokens:        0,   // Will be calculated later
		NetworkStateTokens:   0,   // Will be calculated later
	}
}

// selectBestTemplate selects the most appropriate template
func (dcm *DynamicContextManager) selectBestTemplate(
	templates []*TelecomPromptTemplate,
	request *ContextInjectionRequest,
) *TelecomPromptTemplate {
	if len(templates) == 1 {
		return templates[0]
	}
	
	// Score templates based on relevance
	type scoredTemplate struct {
		template *TelecomPromptTemplate
		score    float64
	}
	
	scored := make([]scoredTemplate, len(templates))
	for i, template := range templates {
		score := 0.0
		
		// Domain match
		if domainPriority, exists := dcm.config.TelecomDomainPriority[template.Domain]; exists {
			score += domainPriority * 10
		}
		
		// Technology match
		intentLower := strings.ToLower(request.Intent)
		for _, tech := range template.Technology {
			if strings.Contains(intentLower, strings.ToLower(tech)) {
				score += 5
			}
		}
		
		// Compliance match
		for _, compliance := range request.ComplianceReqs {
			for _, templateCompliance := range template.ComplianceStandards {
				if compliance == templateCompliance {
					score += 3
				}
			}
		}
		
		scored[i] = scoredTemplate{template: template, score: score}
	}
	
	// Sort by score
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})
	
	return scored[0].template
}

// buildUserPrompt builds the user prompt from template and request
func (dcm *DynamicContextManager) buildUserPrompt(
	template *TelecomPromptTemplate,
	request *ContextInjectionRequest,
) string {
	variables := map[string]interface{}{
		"intent": request.Intent,
	}
	
	// Add optional variables
	if request.NetworkState != nil {
		variables["network_state"] = request.NetworkState
	}
	
	if len(request.ComplianceReqs) > 0 {
		variables["compliance_requirements"] = request.ComplianceReqs
	}
	
	// Add user preferences
	for k, v := range request.UserPreferences {
		variables[k] = v
	}
	
	prompt, _ := dcm.promptRegistry.BuildPrompt(template.ID, variables)
	
	// Extract just the user prompt part (after system prompt)
	parts := strings.Split(prompt, template.SystemPrompt)
	if len(parts) > 1 {
		return strings.TrimSpace(parts[1])
	}
	
	return request.Intent
}

// gatherAndInjectContext gathers relevant context within token budget
func (dcm *DynamicContextManager) gatherAndInjectContext(
	ctx context.Context,
	request *ContextInjectionRequest,
	tier *ContextTier,
	maxTokens int,
) (string, []*DocumentReference, int) {
	// This would integrate with RAG system to fetch relevant documents
	// For now, return placeholder
	contextParts := []string{}
	documents := []*DocumentReference{}
	totalTokens := 0
	
	// Add required documents if specified
	if len(request.RequiredDocuments) > 0 {
		for _, docID := range request.RequiredDocuments {
			// Fetch document from cache or storage
			if cachedDoc := dcm.documentCache.Get(docID); cachedDoc != nil {
				tokens := dcm.tokenManager.EstimateTokensForModel(cachedDoc.ProcessedText, request.ModelName)
				if totalTokens+tokens <= maxTokens {
					contextParts = append(contextParts, cachedDoc.ProcessedText)
					totalTokens += tokens
					
					documents = append(documents, &DocumentReference{
						ID:             cachedDoc.Document.ID,
						Title:          cachedDoc.Document.Title,
						Source:         cachedDoc.Document.Source,
						Category:       cachedDoc.Document.Category,
						RelevanceScore: 1.0, // Required document
						UsedTokens:     tokens,
					})
				}
			}
		}
	}
	
	// Add telecom-specific context based on intent
	if strings.Contains(strings.ToLower(request.Intent), "o-ran") {
		oranContext := `O-RAN Context:
- Current O-RAN Alliance specifications: G-release
- Supported interfaces: A1, E2, O1, O2, Open Fronthaul
- xApp framework version: 2.0
- Conflict mitigation: Enabled
- Multi-vendor support: Active`
		
		tokens := dcm.tokenManager.EstimateTokensForModel(oranContext, request.ModelName)
		if totalTokens+tokens <= maxTokens {
			contextParts = append(contextParts, oranContext)
			totalTokens += tokens
		}
	}
	
	context := strings.Join(contextParts, "\n\n---\n\n")
	return context, documents, totalTokens
}

// buildNetworkStateContext builds context from network state
func (dcm *DynamicContextManager) buildNetworkStateContext(
	state *NetworkStateContext,
	modelName string,
) (string, int) {
	if state == nil {
		return "", 0
	}
	
	var parts []string
	
	// Active slices summary
	if len(state.ActiveSlices) > 0 {
		parts = append(parts, "Active Network Slices:")
		for _, slice := range state.ActiveSlices {
			sliceInfo := fmt.Sprintf("- Slice %s (SST=%d, SD=%s): %s, %d active sessions",
				slice.SliceID, slice.SST, slice.SD, slice.Status, slice.ActiveSessions)
			parts = append(parts, sliceInfo)
		}
	}
	
	// Network functions summary
	if len(state.NetworkFunctions) > 0 {
		parts = append(parts, "\nDeployed Network Functions:")
		for _, nf := range state.NetworkFunctions {
			nfInfo := fmt.Sprintf("- %s (%s v%s): %s in %s",
				nf.Name, nf.Type, nf.Version, nf.Status, nf.Namespace)
			parts = append(parts, nfInfo)
		}
	}
	
	// Active alarms
	if len(state.ActiveAlarms) > 0 {
		parts = append(parts, "\nActive Alarms:")
		for _, alarm := range state.ActiveAlarms {
			alarmInfo := fmt.Sprintf("- [%s] %s: %s",
				alarm.Severity, alarm.Component, alarm.Description)
			parts = append(parts, alarmInfo)
		}
	}
	
	context := strings.Join(parts, "\n")
	tokens := dcm.tokenManager.EstimateTokensForModel(context, modelName)
	
	return context, tokens
}

// compressContext applies compression to reduce token usage
func (dcm *DynamicContextManager) compressContext(
	context string,
	compressionLevel string,
	modelName string,
) (string, int) {
	switch compressionLevel {
	case "light":
		// Remove extra whitespace and formatting
		compressed := strings.TrimSpace(context)
		compressed = strings.ReplaceAll(compressed, "\n\n", "\n")
		compressed = strings.ReplaceAll(compressed, "  ", " ")
		tokens := dcm.tokenManager.EstimateTokensForModel(compressed, modelName)
		return compressed, tokens
		
	case "moderate":
		// Remove redundant information and compress structure
		lines := strings.Split(context, "\n")
		var compressed []string
		seen := make(map[string]bool)
		
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !seen[trimmed] {
				compressed = append(compressed, trimmed)
				seen[trimmed] = true
			}
		}
		
		result := strings.Join(compressed, "\n")
		tokens := dcm.tokenManager.EstimateTokensForModel(result, modelName)
		return result, tokens
		
	case "heavy":
		// Aggressive compression - summarize content
		// In production, this could use an LLM to summarize
		words := strings.Fields(context)
		if len(words) > 100 {
			// Take first 30% and last 20% of content
			firstPart := strings.Join(words[:int(float64(len(words))*0.3)], " ")
			lastPart := strings.Join(words[int(float64(len(words))*0.8):], " ")
			compressed := firstPart + " [...content compressed...] " + lastPart
			tokens := dcm.tokenManager.EstimateTokensForModel(compressed, modelName)
			return compressed, tokens
		}
	}
	
	tokens := dcm.tokenManager.EstimateTokensForModel(context, modelName)
	return context, tokens
}

// selectExamples selects relevant examples within token budget
func (dcm *DynamicContextManager) selectExamples(
	examples []TelecomExample,
	request *ContextInjectionRequest,
	maxTokens int,
	modelName string,
) ([]TelecomExample, int) {
	if len(examples) == 0 || maxTokens < 100 {
		return nil, 0
	}
	
	// Score examples by relevance
	type scoredExample struct {
		example TelecomExample
		score   float64
	}
	
	scored := make([]scoredExample, len(examples))
	intentLower := strings.ToLower(request.Intent)
	
	for i, example := range examples {
		score := 0.0
		
		// Context similarity
		if strings.Contains(strings.ToLower(example.Context), intentLower) {
			score += 10
		}
		
		// Intent similarity
		exampleIntentLower := strings.ToLower(example.Intent)
		intentWords := strings.Fields(intentLower)
		for _, word := range intentWords {
			if strings.Contains(exampleIntentLower, word) {
				score += 1
			}
		}
		
		scored[i] = scoredExample{example: example, score: score}
	}
	
	// Sort by score
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})
	
	// Select examples that fit within budget
	selected := []TelecomExample{}
	totalTokens := 0
	
	for _, s := range scored {
		if s.score > 0 {
			exampleText := fmt.Sprintf("Example:\nContext: %s\nIntent: %s\nResponse: %s",
				s.example.Context, s.example.Intent, s.example.Response)
			tokens := dcm.tokenManager.EstimateTokensForModel(exampleText, modelName)
			
			if totalTokens+tokens <= maxTokens {
				selected = append(selected, s.example)
				totalTokens += tokens
				
				if len(selected) >= 2 { // Limit to 2 examples
					break
				}
			}
		}
	}
	
	return selected, totalTokens
}

// formatExamples formats examples for inclusion in prompt
func (dcm *DynamicContextManager) formatExamples(examples []TelecomExample) string {
	parts := []string{"## Examples:"}
	
	for i, example := range examples {
		parts = append(parts, fmt.Sprintf("\n### Example %d:", i+1))
		parts = append(parts, fmt.Sprintf("Context: %s", example.Context))
		parts = append(parts, fmt.Sprintf("Intent: %s", example.Intent))
		parts = append(parts, fmt.Sprintf("Response:\n%s", example.Response))
		
		if example.Explanation != "" {
			parts = append(parts, fmt.Sprintf("Explanation: %s", example.Explanation))
		}
	}
	
	return strings.Join(parts, "\n")
}

// assembleFinalPrompt assembles the final prompt
func (dcm *DynamicContextManager) assembleFinalPrompt(
	systemPrompt, userPrompt, context string,
) string {
	parts := []string{}
	
	if systemPrompt != "" {
		parts = append(parts, systemPrompt)
	}
	
	if context != "" {
		parts = append(parts, "## Context:", context)
	}
	
	if userPrompt != "" {
		parts = append(parts, "## Request:", userPrompt)
	}
	
	return strings.Join(parts, "\n\n")
}

// calculateQualityScore calculates the quality score of the injection
func (dcm *DynamicContextManager) calculateQualityScore(
	documents []*DocumentReference,
	tier *ContextTier,
	tokenUsage TokenUsageBreakdown,
	compressionApplied bool,
) float32 {
	score := float32(0.5) // Base score
	
	// Document quality contribution
	if len(documents) > 0 {
		var avgRelevance float32
		for _, doc := range documents {
			avgRelevance += doc.RelevanceScore
		}
		avgRelevance /= float32(len(documents))
		score += avgRelevance * 0.2
	}
	
	// Tier quality contribution
	switch tier.ComplexityLevel {
	case "comprehensive":
		score += 0.2
	case "standard":
		score += 0.15
	case "minimal":
		score += 0.1
	}
	
	// Token efficiency contribution
	if tokenUsage.BudgetUtilization > 0.8 && tokenUsage.BudgetUtilization < 0.95 {
		score += 0.1 // Good utilization
	}
	
	// Compression penalty
	if compressionApplied {
		score -= 0.05
	}
	
	// Cap score at 1.0
	if score > 1.0 {
		score = 1.0
	}
	
	return score
}

// NewDocumentCache creates a new document cache
func NewDocumentCache(maxSize int, ttl time.Duration) *DocumentCache {
	cache := &DocumentCache{
		cache:   make(map[string]*CachedDocument),
		maxSize: maxSize,
		ttl:     ttl,
	}
	
	// Start cleanup goroutine
	go cache.cleanupLoop()
	
	return cache
}

// Get retrieves a document from cache
func (dc *DocumentCache) Get(id string) *CachedDocument {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	
	if doc, exists := dc.cache[id]; exists {
		if time.Since(doc.LastAccessed) < dc.ttl {
			doc.LastAccessed = time.Now()
			doc.AccessCount++
			return doc
		}
	}
	
	return nil
}

// Set adds a document to cache
func (dc *DocumentCache) Set(id string, doc *CachedDocument) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	
	// Evict oldest if at capacity
	if len(dc.cache) >= dc.maxSize {
		dc.evictOldest()
	}
	
	doc.LastAccessed = time.Now()
	dc.cache[id] = doc
}

// evictOldest removes the least recently accessed document
func (dc *DocumentCache) evictOldest() {
	var oldestID string
	var oldestTime time.Time
	
	for id, doc := range dc.cache {
		if oldestID == "" || doc.LastAccessed.Before(oldestTime) {
			oldestID = id
			oldestTime = doc.LastAccessed
		}
	}
	
	if oldestID != "" {
		delete(dc.cache, oldestID)
	}
}

// cleanupLoop periodically removes expired entries
func (dc *DocumentCache) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		dc.mu.Lock()
		now := time.Now()
		
		for id, doc := range dc.cache {
			if now.Sub(doc.LastAccessed) > dc.ttl {
				delete(dc.cache, id)
			}
		}
		
		dc.mu.Unlock()
	}
}

// GetMetrics returns current context manager metrics
func (dcm *DynamicContextManager) GetMetrics() map[string]interface{} {
	dcm.mu.RLock()
	defer dcm.mu.RUnlock()
	
	return map[string]interface{}{
		"cache_size":        len(dcm.documentCache.cache),
		"context_metrics":   dcm.metrics,
		"tier_definitions":  dcm.config.TierDefinitions,
		"current_config":    dcm.config,
	}
}

// UpdateConfig updates the dynamic context configuration
func (dcm *DynamicContextManager) UpdateConfig(config *DynamicContextConfig) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}
	
	dcm.mu.Lock()
	defer dcm.mu.Unlock()
	
	dcm.config = config
	dcm.logger.Info("Dynamic context configuration updated")
	
	return nil
}