package llm

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"strings"
	"sync"
	"time"
)

// TokenManager manages token budgets and limits for different LLM models
type TokenManager struct {
	modelConfigs map[string]*ModelTokenConfig
	logger       *slog.Logger
	mutex        sync.RWMutex
}

// ModelTokenConfig holds token configuration for a specific model
type ModelTokenConfig struct {
	MaxTokens              int     `json:"max_tokens"`
	ContextWindow         int     `json:"context_window"`
	ReservedTokens        int     `json:"reserved_tokens"`        // Tokens reserved for response
	PromptTokens          int     `json:"prompt_tokens"`          // Tokens reserved for system prompt
	SafetyMargin          float64 `json:"safety_margin"`          // Safety margin (0.1 = 10%)
	TokensPerChar         float64 `json:"tokens_per_char"`        // Rough estimation
	TokensPerWord         float64 `json:"tokens_per_word"`        // More accurate estimation
	SupportsChatFormat    bool    `json:"supports_chat_format"`
	SupportsSystemPrompt  bool    `json:"supports_system_prompt"`
	SupportsStreaming     bool    `json:"supports_streaming"`
}

// TokenBudget represents available token allocation for a request
type TokenBudget struct {
	TotalAvailable    int     `json:"total_available"`
	ContextBudget     int     `json:"context_budget"`
	PromptBudget      int     `json:"prompt_budget"`
	ResponseBudget    int     `json:"response_budget"`
	UtilizationRatio  float64 `json:"utilization_ratio"`
	ModelName         string  `json:"model_name"`
	EstimatedUsage    int     `json:"estimated_usage"`
	CanAccommodate    bool    `json:"can_accommodate"`
}

// TokenUsage tracks actual token consumption
type TokenUsage struct {
	PromptTokens      int       `json:"prompt_tokens"`
	ResponseTokens    int       `json:"response_tokens"`
	TotalTokens       int       `json:"total_tokens"`
	ContextTokens     int       `json:"context_tokens"`
	EstimatedCost     float64   `json:"estimated_cost"`
	ProcessingTime    time.Duration `json:"processing_time"`
	ModelName         string    `json:"model_name"`
	RequestID         string    `json:"request_id"`
	Timestamp         time.Time `json:"timestamp"`
}

// NewTokenManager creates a new token manager with default model configurations
func NewTokenManager() *TokenManager {
	tm := &TokenManager{
		modelConfigs: make(map[string]*ModelTokenConfig),
		logger:       slog.Default().With("component", "token-manager"),
	}

	// Initialize default model configurations
	tm.initializeDefaultConfigs()
	
	return tm
}

// initializeDefaultConfigs sets up default configurations for common models
func (tm *TokenManager) initializeDefaultConfigs() {
	defaultConfigs := map[string]*ModelTokenConfig{
		"gpt-4o": {
			MaxTokens:             4096,
			ContextWindow:         128000,
			ReservedTokens:        1024,
			PromptTokens:          512,
			SafetyMargin:          0.1,
			TokensPerChar:         0.25,
			TokensPerWord:         1.3,
			SupportsChatFormat:    true,
			SupportsSystemPrompt:  true,
			SupportsStreaming:     true,
		},
		"gpt-4o-mini": {
			MaxTokens:             4096,
			ContextWindow:         128000,
			ReservedTokens:        1024,
			PromptTokens:          512,
			SafetyMargin:          0.1,
			TokensPerChar:         0.25,
			TokensPerWord:         1.3,
			SupportsChatFormat:    true,
			SupportsSystemPrompt:  true,
			SupportsStreaming:     true,
		},
		"gpt-3.5-turbo": {
			MaxTokens:             4096,
			ContextWindow:         16384,
			ReservedTokens:        1024,
			PromptTokens:          512,
			SafetyMargin:          0.1,
			TokensPerChar:         0.25,
			TokensPerWord:         1.3,
			SupportsChatFormat:    true,
			SupportsSystemPrompt:  true,
			SupportsStreaming:     true,
		},
		"mistral-7b": {
			MaxTokens:             4096,
			ContextWindow:         32768,
			ReservedTokens:        1024,
			PromptTokens:          256,
			SafetyMargin:          0.15,
			TokensPerChar:         0.3,
			TokensPerWord:         1.4,
			SupportsChatFormat:    true,
			SupportsSystemPrompt:  false,
			SupportsStreaming:     true,
		},
		"llama-2-70b": {
			MaxTokens:             4096,
			ContextWindow:         4096,
			ReservedTokens:        1024,
			PromptTokens:          256,
			SafetyMargin:          0.15,
			TokensPerChar:         0.3,
			TokensPerWord:         1.4,
			SupportsChatFormat:    false,
			SupportsSystemPrompt:  false,
			SupportsStreaming:     false,
		},
		"claude-3-haiku": {
			MaxTokens:             4096,
			ContextWindow:         200000,
			ReservedTokens:        1024,
			PromptTokens:          512,
			SafetyMargin:          0.1,
			TokensPerChar:         0.24,
			TokensPerWord:         1.2,
			SupportsChatFormat:    true,
			SupportsSystemPrompt:  true,
			SupportsStreaming:     true,
		},
	}

	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	
	for modelName, config := range defaultConfigs {
		tm.modelConfigs[modelName] = config
	}
}

// RegisterModel registers a new model configuration
func (tm *TokenManager) RegisterModel(modelName string, config *ModelTokenConfig) error {
	if config == nil {
		return fmt.Errorf("model configuration cannot be nil")
	}
	
	if err := tm.validateModelConfig(config); err != nil {
		return fmt.Errorf("invalid model configuration: %w", err)
	}

	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	
	tm.modelConfigs[modelName] = config
	tm.logger.Info("Registered new model configuration", "model", modelName)
	
	return nil
}

// validateModelConfig validates a model configuration
func (tm *TokenManager) validateModelConfig(config *ModelTokenConfig) error {
	if config.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens must be positive")
	}
	if config.ContextWindow <= 0 {
		return fmt.Errorf("context_window must be positive")
	}
	if config.ReservedTokens < 0 {
		return fmt.Errorf("reserved_tokens cannot be negative")
	}
	if config.PromptTokens < 0 {
		return fmt.Errorf("prompt_tokens cannot be negative")
	}
	if config.SafetyMargin < 0 || config.SafetyMargin > 1 {
		return fmt.Errorf("safety_margin must be between 0 and 1")
	}
	if config.TokensPerChar <= 0 {
		return fmt.Errorf("tokens_per_char must be positive")
	}
	if config.TokensPerWord <= 0 {
		return fmt.Errorf("tokens_per_word must be positive")
	}
	
	return nil
}

// GetModelConfig retrieves configuration for a specific model
func (tm *TokenManager) GetModelConfig(modelName string) (*ModelTokenConfig, error) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	
	config, exists := tm.modelConfigs[modelName]
	if !exists {
		return nil, fmt.Errorf("model configuration not found: %s", modelName)
	}
	
	// Return a copy to prevent external modification
	configCopy := *config
	return &configCopy, nil
}

// CalculateTokenBudget calculates available token budget for a request
func (tm *TokenManager) CalculateTokenBudget(ctx context.Context, modelName string, systemPrompt, userPrompt, ragContext string) (*TokenBudget, error) {
	config, err := tm.GetModelConfig(modelName)
	if err != nil {
		return nil, err
	}

	// Estimate token usage for different components
	systemPromptTokens := tm.EstimateTokens(systemPrompt)
	userPromptTokens := tm.EstimateTokens(userPrompt)
	contextTokens := tm.EstimateTokens(ragContext)
	
	totalPromptTokens := systemPromptTokens + userPromptTokens + contextTokens
	
	// Calculate available budget
	availableForContext := config.ContextWindow - config.ReservedTokens - int(float64(config.ContextWindow)*config.SafetyMargin)
	availableForResponse := config.MaxTokens
	
	// Check if current usage exceeds limits
	canAccommodate := totalPromptTokens <= availableForContext
	
	budget := &TokenBudget{
		TotalAvailable:   availableForContext,
		ContextBudget:    availableForContext - totalPromptTokens,
		PromptBudget:     totalPromptTokens,
		ResponseBudget:   availableForResponse,
		UtilizationRatio: float64(totalPromptTokens) / float64(availableForContext),
		ModelName:        modelName,
		EstimatedUsage:   totalPromptTokens,
		CanAccommodate:   canAccommodate,
	}

	tm.logger.Debug("Calculated token budget",
		"model", modelName,
		"prompt_tokens", totalPromptTokens,
		"context_budget", budget.ContextBudget,
		"utilization", fmt.Sprintf("%.2f%%", budget.UtilizationRatio*100),
		"can_accommodate", canAccommodate,
	)

	return budget, nil
}

// EstimateTokens estimates the number of tokens in a text string
func (tm *TokenManager) EstimateTokens(text string) int {
	if text == "" {
		return 0
	}

	// Use a combination of character and word-based estimation for better accuracy
	charCount := len(text)
	wordCount := len(strings.Fields(text))
	
	// Default estimation (fallback if no model-specific config)
	tokensPerChar := 0.25
	tokensPerWord := 1.3
	
	// Use character-based estimation for short texts, word-based for longer ones
	var estimation float64
	if wordCount > 0 {
		estimation = float64(wordCount) * tokensPerWord
	} else {
		estimation = float64(charCount) * tokensPerChar
	}
	
	// Apply some adjustment for code/technical content
	if tm.isCodeOrTechnicalContent(text) {
		estimation *= 1.2 // Technical content tends to use more tokens
	}
	
	return int(math.Ceil(estimation))
}

// EstimateTokensForModel estimates tokens using model-specific parameters
func (tm *TokenManager) EstimateTokensForModel(text, modelName string) int {
	if text == "" {
		return 0
	}

	config, err := tm.GetModelConfig(modelName)
	if err != nil {
		tm.logger.Warn("Model config not found, using default estimation", "model", modelName)
		return tm.EstimateTokens(text)
	}

	charCount := len(text)
	wordCount := len(strings.Fields(text))
	
	var estimation float64
	if wordCount > 0 {
		estimation = float64(wordCount) * config.TokensPerWord
	} else {
		estimation = float64(charCount) * config.TokensPerChar
	}
	
	// Apply technical content adjustment
	if tm.isCodeOrTechnicalContent(text) {
		estimation *= 1.2
	}
	
	return int(math.Ceil(estimation))
}

// isCodeOrTechnicalContent detects if text contains code or technical content
func (tm *TokenManager) isCodeOrTechnicalContent(text string) bool {
	// Simple heuristics to detect technical content
	codeIndicators := []string{
		"```", "function", "class", "def ", "import ", "from ", 
		"<", ">", "{", "}", "[", "]", "->", "=>", "::", 
		"3GPP", "O-RAN", "gNB", "eNB", "API", "HTTP", "JSON", "XML",
		"kubectl", "docker", "kubernetes", "yaml", "json",
	}
	
	lowerText := strings.ToLower(text)
	indicatorCount := 0
	
	for _, indicator := range codeIndicators {
		if strings.Contains(lowerText, strings.ToLower(indicator)) {
			indicatorCount++
		}
	}
	
	// Consider it technical if it has multiple indicators
	return indicatorCount >= 2
}

// TruncateToFit truncates content to fit within token budget
func (tm *TokenManager) TruncateToFit(content string, maxTokens int, modelName string) string {
	currentTokens := tm.EstimateTokensForModel(content, modelName)
	
	if currentTokens <= maxTokens {
		return content
	}
	
	// Calculate approximate ratio to truncate
	targetRatio := float64(maxTokens) / float64(currentTokens)
	targetLength := int(float64(len(content)) * targetRatio * 0.9) // 90% to be safe
	
	if targetLength >= len(content) {
		return content
	}
	
	// Try to truncate at word boundaries
	words := strings.Fields(content)
	if len(words) == 0 {
		return content[:targetLength]
	}
	
	// Estimate words to keep
	targetWords := int(float64(len(words)) * targetRatio * 0.9)
	if targetWords <= 0 {
		targetWords = 1
	}
	
	truncated := strings.Join(words[:targetWords], " ")
	
	// Verify the truncation worked
	if tm.EstimateTokensForModel(truncated, modelName) > maxTokens {
		// If still too long, fallback to character truncation
		return content[:targetLength]
	}
	
	return truncated
}

// OptimizeContext optimizes RAG context to fit within token budget
func (tm *TokenManager) OptimizeContext(contexts []string, maxTokens int, modelName string) []string {
	if len(contexts) == 0 {
		return contexts
	}
	
	var optimized []string
	currentTokens := 0
	
	// Sort contexts by estimated relevance (assume first ones are most relevant)
	for _, context := range contexts {
		contextTokens := tm.EstimateTokensForModel(context, modelName)
		
		if currentTokens+contextTokens <= maxTokens {
			optimized = append(optimized, context)
			currentTokens += contextTokens
		} else {
			// Try to fit a truncated version of this context
			remainingTokens := maxTokens - currentTokens
			if remainingTokens > 100 { // Only if we have reasonable space left
				truncated := tm.TruncateToFit(context, remainingTokens, modelName)
				if truncated != "" {
					optimized = append(optimized, truncated)
				}
			}
			break
		}
	}
	
	tm.logger.Debug("Optimized context",
		"original_contexts", len(contexts),
		"optimized_contexts", len(optimized),
		"final_tokens", currentTokens,
		"max_tokens", maxTokens,
	)
	
	return optimized
}

// CreateTokenUsage creates a token usage record
func (tm *TokenManager) CreateTokenUsage(modelName, requestID string, promptTokens, responseTokens, contextTokens int, processingTime time.Duration) *TokenUsage {
	totalTokens := promptTokens + responseTokens
	
	// Estimate cost (simplified - in production this would use actual pricing)
	var estimatedCost float64
	switch {
	case strings.Contains(strings.ToLower(modelName), "gpt-4"):
		estimatedCost = float64(promptTokens)*0.00003 + float64(responseTokens)*0.00006
	case strings.Contains(strings.ToLower(modelName), "gpt-3.5"):
		estimatedCost = float64(totalTokens) * 0.000002
	default:
		estimatedCost = float64(totalTokens) * 0.000001 // Generic estimate
	}
	
	return &TokenUsage{
		PromptTokens:   promptTokens,
		ResponseTokens: responseTokens,
		TotalTokens:    totalTokens,
		ContextTokens:  contextTokens,
		EstimatedCost:  estimatedCost,
		ProcessingTime: processingTime,
		ModelName:      modelName,
		RequestID:      requestID,
		Timestamp:      time.Now(),
	}
}

// GetSupportedModels returns a list of all registered models
func (tm *TokenManager) GetSupportedModels() []string {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	
	models := make([]string, 0, len(tm.modelConfigs))
	for modelName := range tm.modelConfigs {
		models = append(models, modelName)
	}
	
	return models
}

// SupportsStreaming checks if a model supports streaming
func (tm *TokenManager) SupportsStreaming(modelName string) bool {
	config, err := tm.GetModelConfig(modelName)
	if err != nil {
		return false
	}
	return config.SupportsStreaming
}

// SupportsChatFormat checks if a model supports chat format
func (tm *TokenManager) SupportsChatFormat(modelName string) bool {
	config, err := tm.GetModelConfig(modelName)
	if err != nil {
		return false
	}
	return config.SupportsChatFormat
}

// SupportsSystemPrompt checks if a model supports system prompts
func (tm *TokenManager) SupportsSystemPrompt(modelName string) bool {
	config, err := tm.GetModelConfig(modelName)
	if err != nil {
		return false
	}
	return config.SupportsSystemPrompt
}

// ValidateTokenBudget validates that a request can fit within model limits
func (tm *TokenManager) ValidateTokenBudget(budget *TokenBudget) error {
	if budget == nil {
		return fmt.Errorf("token budget cannot be nil")
	}
	
	if !budget.CanAccommodate {
		return fmt.Errorf("request exceeds token limit: estimated %d tokens, available %d tokens", 
			budget.EstimatedUsage, budget.TotalAvailable)
	}
	
	if budget.UtilizationRatio > 0.95 {
		tm.logger.Warn("High token utilization",
			"model", budget.ModelName,
			"utilization", fmt.Sprintf("%.2f%%", budget.UtilizationRatio*100),
		)
	}
	
	return nil
}

// EstimateAccurateTokens provides more accurate token estimation using regex patterns
func (tm *TokenManager) EstimateAccurateTokens(text string) int {
	if text == "" {
		return 0
	}
	
	// More sophisticated token estimation using patterns
	// This is still an approximation but better than simple word/char counting
	
	// Handle common patterns that affect tokenization
	patterns := []struct {
		pattern *regexp.Regexp
		factor  float64
	}{
		{regexp.MustCompile(`\b\d+\.\d+\.\d+\.\d+\b`), 4.0},      // IP addresses
		{regexp.MustCompile(`\b[A-Z]{2,}\b`), 1.5},               // Acronyms
		{regexp.MustCompile(`\b\w+@\w+\.\w+\b`), 3.0},            // Email addresses
		{regexp.MustCompile(`https?://\S+`), 4.0},                // URLs
		{regexp.MustCompile(`[{}[\]().,;:!?-]`), 1.0},            // Punctuation
		{regexp.MustCompile(`\b\w{10,}\b`), 2.0},                 // Long words
	}
	
	tokenCount := 0.0
	processedText := text
	
	for _, pattern := range patterns {
		matches := pattern.pattern.FindAllString(processedText, -1)
		for _, match := range matches {
			tokenCount += float64(len(strings.Fields(match))) * pattern.factor
			processedText = strings.ReplaceAll(processedText, match, " ")
		}
	}
	
	// Count remaining regular words
	remainingWords := strings.Fields(processedText)
	tokenCount += float64(len(remainingWords)) * 1.3
	
	return int(math.Ceil(tokenCount))
}