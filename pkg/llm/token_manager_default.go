// Default token manager implementation (no build tags)
// This ensures NewTokenManager is always available
package llm

import (
	"fmt"
	"strings"
	"sync"
)

// DefaultTokenManager provides a basic token management implementation
// that is always available regardless of build tags
type DefaultTokenManager struct {
	mutex sync.RWMutex
}

// NewTokenManager creates a new token manager
// This is the default implementation that's always available
func NewTokenManager() TokenManager {
	return &DefaultTokenManager{}
}

// AllocateTokens estimates token allocation for a request
func (dtm *DefaultTokenManager) AllocateTokens(request string) (int, error) {
	if request == "" {
		return 0, fmt.Errorf("request cannot be empty")
	}
	
	// Simple token estimation: ~4 chars per token
	estimatedTokens := len(request) / 4
	if estimatedTokens == 0 {
		estimatedTokens = 1
	}
	
	return estimatedTokens, nil
}

// ReleaseTokens releases allocated tokens
func (dtm *DefaultTokenManager) ReleaseTokens(tokens int) error {
	// No-op for basic implementation
	return nil
}

// GetAvailableTokens returns the number of available tokens
func (dtm *DefaultTokenManager) GetAvailableTokens() int {
	// Return a large number for basic implementation
	return 1000000
}

// EstimateTokensForModel estimates tokens for a specific model
func (dtm *DefaultTokenManager) EstimateTokensForModel(model string, text string) (int, error) {
	return dtm.EstimateTokens(text), nil
}

// EstimateTokens estimates the number of tokens in a string
func (dtm *DefaultTokenManager) EstimateTokens(text string) int {
	if text == "" {
		return 0
	}
	
	// Simple estimation: split by whitespace and punctuation
	// This is a rough approximation
	words := strings.Fields(text)
	tokenCount := 0
	
	for _, word := range words {
		// Rough estimate: each word is about 1.3 tokens
		tokenCount++
		if len(word) > 10 {
			// Longer words might be multiple tokens
			tokenCount += len(word) / 10
		}
	}
	
	if tokenCount == 0 && text != "" {
		tokenCount = 1
	}
	
	return tokenCount
}

// ValidateTokenLimit checks if the text exceeds token limits
func (dtm *DefaultTokenManager) ValidateTokenLimit(text string, maxTokens int) error {
	estimated := dtm.EstimateTokens(text)
	if estimated > maxTokens {
		return fmt.Errorf("estimated tokens (%d) exceeds limit (%d)", estimated, maxTokens)
	}
	return nil
}

// SupportsSystemPrompt checks if a model supports system prompts
func (dtm *DefaultTokenManager) SupportsSystemPrompt(model string) bool {
	return true // Default implementation supports system prompts
}

// SupportsChatFormat checks if a model supports chat format
func (dtm *DefaultTokenManager) SupportsChatFormat(model string) bool {
	return true // Default implementation supports chat format
}

// SupportsStreaming checks if a model supports streaming
func (dtm *DefaultTokenManager) SupportsStreaming(model string) bool {
	return false // Default implementation doesn't support streaming
}

// TruncateToFit truncates text to fit within token limits
func (dtm *DefaultTokenManager) TruncateToFit(text string, maxTokens int, model string) (string, error) {
	currentTokens := dtm.EstimateTokens(text)
	if currentTokens <= maxTokens {
		return text, nil
	}
	
	// Simple truncation by character ratio
	ratio := float64(maxTokens) / float64(currentTokens)
	targetLength := int(float64(len(text)) * ratio * 0.9) // Safety margin
	
	if targetLength <= 0 {
		return "", fmt.Errorf("maxTokens too small: %d", maxTokens)
	}
	if targetLength >= len(text) {
		return text, nil
	}
	
	// Find nearest word boundary
	truncated := text[:targetLength]
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > targetLength/2 {
		truncated = truncated[:lastSpace]
	}
	
	return truncated + "...", nil
}

// GetTokenCount returns estimated token count
func (dtm *DefaultTokenManager) GetTokenCount(text string) int {
	return dtm.EstimateTokens(text)
}

// ValidateModel validates if a model is supported
func (dtm *DefaultTokenManager) ValidateModel(model string) error {
	if model == "" {
		return fmt.Errorf("model name cannot be empty")
	}
	return nil // Default implementation accepts any non-empty model
}

// GetSupportedModels returns list of supported models
func (dtm *DefaultTokenManager) GetSupportedModels() []string {
	return []string{"default", "mock-model", "test-model"}
}

// GetTokenUsage returns current token usage statistics
func (dtm *DefaultTokenManager) GetTokenUsage() TokenUsageInfo {
	return TokenUsageInfo{
		PromptTokens:     0,
		CompletionTokens: 0,
		TotalTokens:      0,
	}
}

// ResetUsage resets token usage statistics
func (dtm *DefaultTokenManager) ResetUsage() {
	// No-op for basic implementation
}