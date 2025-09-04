//go:build !stub && !disable_rag

package llm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/oauth2"
	"golang.org/x/time/rate"
)

// OAuth2TokenManager provides comprehensive token management with OAuth2 support
type OAuth2TokenManager struct {
	// Core configuration
	maxTokens       int
	availableTokens int
	allocated       map[string]int
	mu              sync.RWMutex

	// OAuth2 configuration
	oauth2Config *oauth2.Config
	token        *oauth2.Token
	tokenMu      sync.RWMutex

	// Rate limiting
	rateLimiter *rate.Limiter

	// Token caching
	tokenCache   map[string]*CachedTokenEstimate
	cacheMu      sync.RWMutex
	cacheTimeout time.Duration

	// Model-specific configurations
	modelConfigs map[string]*TokenModelConfig

	// Metrics and monitoring
	metrics *TokenMetrics
	logger  *slog.Logger
}

// CachedTokenEstimate stores cached token estimates with expiration
type CachedTokenEstimate struct {
	Tokens    int
	Model     string
	TextHash  string
	Timestamp time.Time
	ExpiresAt time.Time
}

// TokenModelConfig stores model-specific token configuration
type TokenModelConfig struct {
	Name              string
	TokensPerChar     float64
	MaxTokens         int
	SupportsSystem    bool
	SupportsChat      bool
	SupportsStreaming bool
	InputCostPer1K    float64
	OutputCostPer1K   float64
	TokenizerType     string
	ContextWindow     int
}

// TokenMetrics stores token usage metrics
type TokenMetrics struct {
	TotalAllocated   int64
	TotalReleased    int64
	CacheHits        int64
	CacheMisses      int64
	RateLimitHits    int64
	ModelEstimations map[string]int64
	mu               sync.RWMutex
}

// OAuth2Config represents OAuth2 configuration for token management
type OAuth2Config struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	Scopes       []string
}

// NewOAuth2TokenManager creates a new OAuth2-enabled token manager
func NewOAuth2TokenManager(maxTokens int, oauth2Cfg *OAuth2Config, logger *slog.Logger) *OAuth2TokenManager {
	if logger == nil {
		logger = slog.Default()
	}

	tm := &OAuth2TokenManager{
		maxTokens:       maxTokens,
		availableTokens: maxTokens,
		allocated:       make(map[string]int),
		tokenCache:      make(map[string]*CachedTokenEstimate),
		cacheTimeout:    5 * time.Minute,
		modelConfigs:    getDefaultModelConfigs(),
		rateLimiter:     rate.NewLimiter(rate.Limit(100), 1000), // 100 requests/sec, burst of 1000
		metrics:         &TokenMetrics{ModelEstimations: make(map[string]int64)},
		logger:          logger,
	}

	if oauth2Cfg != nil {
		tm.oauth2Config = &oauth2.Config{
			ClientID:     oauth2Cfg.ClientID,
			ClientSecret: oauth2Cfg.ClientSecret,
			Endpoint: oauth2.Endpoint{
				TokenURL: oauth2Cfg.TokenURL,
			},
			Scopes: oauth2Cfg.Scopes,
		}
	}

	// Start cleanup goroutine for cache
	go tm.cleanupCache()

	return tm
}

// NewTokenManagerOAuth2 creates a new token manager with OAuth2 support
// Note: The main NewTokenManager is provided by token_manager_default.go
func NewTokenManagerOAuth2() TokenManager {
	return NewOAuth2TokenManager(100000, nil, nil)
}

// AllocateTokens allocates tokens for a request with rate limiting
func (tm *OAuth2TokenManager) AllocateTokens(request string) (int, error) {
	// Check rate limit
	if !tm.rateLimiter.Allow() {
		tm.metrics.mu.Lock()
		tm.metrics.RateLimitHits++
		tm.metrics.mu.Unlock()
		return 0, fmt.Errorf("rate limit exceeded")
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Estimate tokens using enhanced algorithm
	estimatedTokens := tm.estimateTokensAdvanced(request)
	if estimatedTokens < 1 {
		estimatedTokens = 1
	}

	if tm.availableTokens < estimatedTokens {
		return 0, fmt.Errorf("insufficient tokens available: need %d, have %d", estimatedTokens, tm.availableTokens)
	}

	tm.availableTokens -= estimatedTokens
	requestID := fmt.Sprintf("req_%d_%d", time.Now().UnixNano(), len(tm.allocated))
	tm.allocated[requestID] = estimatedTokens

	tm.metrics.mu.Lock()
	tm.metrics.TotalAllocated += int64(estimatedTokens)
	tm.metrics.mu.Unlock()

	tm.logger.Debug("Tokens allocated", "request_id", requestID, "tokens", estimatedTokens)

	return estimatedTokens, nil
}

// CalculateTokenBudget calculates the recommended token budget for a given context
func (tm *OAuth2TokenManager) CalculateTokenBudget(context string, requirements map[string]interface{}) (int, error) {
	// Base token budget calculation
	baseTokens := len(strings.Fields(context)) * 2 // Rough estimation: 2 tokens per word

	// Add buffer for requirements
	if requirements != nil {
		if complexity, ok := requirements["complexity"]; ok {
			if complexityStr, ok := complexity.(string); ok {
				switch complexityStr {
				case "high":
					baseTokens = int(float64(baseTokens) * 1.5)
				case "medium":
					baseTokens = int(float64(baseTokens) * 1.2)
				}
			}
		}
	}

	// Ensure minimum budget
	if baseTokens < 100 {
		baseTokens = 100
	}

	// Ensure it's within our limits
	if baseTokens > tm.maxTokens {
		baseTokens = tm.maxTokens
	}

	return baseTokens, nil
}

// ReleaseTokens releases previously allocated tokens
func (tm *OAuth2TokenManager) ReleaseTokens(count int) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.availableTokens += count
	if tm.availableTokens > tm.maxTokens {
		tm.availableTokens = tm.maxTokens
	}

	tm.metrics.mu.Lock()
	tm.metrics.TotalReleased += int64(count)
	tm.metrics.mu.Unlock()

	tm.logger.Debug("Tokens released", "tokens", count)

	return nil
}

// GetAvailableTokens returns the number of available tokens
func (tm *OAuth2TokenManager) GetAvailableTokens() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return tm.availableTokens
}

// EstimateTokensForModel estimates token count for a specific model with caching
func (tm *OAuth2TokenManager) EstimateTokensForModel(model string, text string) (int, error) {
	if text == "" {
		return 0, nil
	}

	// Generate cache key
	hash := sha256.Sum256([]byte(model + ":" + text))
	cacheKey := hex.EncodeToString(hash[:])[:16]

	// Check cache first
	tm.cacheMu.RLock()
	if cached, exists := tm.tokenCache[cacheKey]; exists && time.Now().Before(cached.ExpiresAt) {
		tm.cacheMu.RUnlock()
		tm.metrics.mu.Lock()
		tm.metrics.CacheHits++
		tm.metrics.mu.Unlock()
		return cached.Tokens, nil
	}
	tm.cacheMu.RUnlock()

	// Cache miss - calculate tokens
	tm.metrics.mu.Lock()
	tm.metrics.CacheMisses++
	tm.metrics.ModelEstimations[model]++
	tm.metrics.mu.Unlock()

	var tokens int
	var err error

	// Get model configuration
	if config, exists := tm.modelConfigs[strings.ToLower(model)]; exists {
		tokens = tm.estimateTokensForModelConfig(text, config)
	} else {
		// Fallback to advanced estimation
		tokens = tm.estimateTokensAdvanced(text)
	}

	// Cache the result
	tm.cacheMu.Lock()
	tm.tokenCache[cacheKey] = &CachedTokenEstimate{
		Tokens:    tokens,
		Model:     model,
		TextHash:  cacheKey,
		Timestamp: time.Now(),
		ExpiresAt: time.Now().Add(tm.cacheTimeout),
	}
	tm.cacheMu.Unlock()

	return tokens, err
}

// estimateTokensForModelConfig estimates tokens based on model-specific configuration
func (tm *OAuth2TokenManager) estimateTokensForModelConfig(text string, config *TokenModelConfig) int {
	switch config.TokenizerType {
	case "gpt":
		return tm.estimateTokensGPT(text)
	case "claude":
		return tm.estimateTokensClaude(text)
	case "llama":
		return tm.estimateTokensLlama(text)
	default:
		return tm.estimateTokensAdvanced(text)
	}
}

// estimateTokensGPT provides GPT-specific token estimation
func (tm *OAuth2TokenManager) estimateTokensGPT(text string) int {
	// GPT tokenization approximation
	words := strings.Fields(text)
	tokens := 0

	for _, word := range words {
		// GPT tokenizer patterns
		if len(word) <= 3 {
			tokens += 1
		} else if len(word) <= 6 {
			tokens += 1
		} else if len(word) <= 10 {
			tokens += 2
		} else {
			tokens += 3
		}

		// Add tokens for punctuation and special characters
		for _, r := range word {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
				tokens += 1
			}
		}
	}

	// Add overhead for special tokens and formatting
	overhead := int(math.Max(float64(len(text))/200, 5))
	return tokens + overhead
}

// estimateTokensClaude provides Claude-specific token estimation
func (tm *OAuth2TokenManager) estimateTokensClaude(text string) int {
	// Claude tokenization is similar to GPT but slightly more efficient
	baseTokens := tm.estimateTokensGPT(text)
	return int(float64(baseTokens) * 0.85) // Claude is ~15% more efficient
}

// estimateTokensLlama provides Llama-specific token estimation
func (tm *OAuth2TokenManager) estimateTokensLlama(text string) int {
	// Llama tokenization patterns
	runeCount := utf8.RuneCountInString(text)
	wordCount := len(strings.Fields(text))

	// Llama tends to use more tokens for the same text
	baseTokens := int(float64(wordCount) * 1.3)

	// Add tokens for character complexity
	complexityTokens := runeCount / 5

	return baseTokens + complexityTokens + 10 // Base overhead
}

// estimateTokensAdvanced provides advanced token estimation algorithm
func (tm *OAuth2TokenManager) estimateTokensAdvanced(text string) int {
	if text == "" {
		return 0
	}

	words := strings.Fields(text)
	tokens := 0

	for _, word := range words {
		// Base token count based on word length and complexity
		wordLen := len(word)
		runeCount := utf8.RuneCountInString(word)

		if wordLen <= 4 {
			tokens += 1
		} else if wordLen <= 8 {
			tokens += 2
		} else if wordLen <= 12 {
			tokens += 3
		} else {
			tokens += 4
		}

		// Adjust for Unicode complexity
		if runeCount != wordLen {
			tokens += (wordLen - runeCount + 3) / 4
		}

		// Add tokens for special patterns
		if hasDigits(word) {
			tokens += 1
		}

		if hasPunctuation(word) {
			tokens += 1
		}

		if hasUpperCase(word) {
			tokens += 1
		}
	}

	// Add base overhead for context and formatting
	baseOverhead := int(math.Max(float64(len(text))/100, 10))

	return tokens + baseOverhead
}

// SupportsSystemPrompt checks if a model supports system prompts
func (tm *OAuth2TokenManager) SupportsSystemPrompt(model string) bool {
	if config, exists := tm.modelConfigs[strings.ToLower(model)]; exists {
		return config.SupportsSystem
	}

	// Default heuristics for unknown models
	model = strings.ToLower(model)
	systemSupportPatterns := []string{
		"gpt-", "claude-", "llama", "mistral", "gemini", "anthropic",
		"openai", "google", "meta", "huggingface", "chat", "instruct",
	}

	for _, pattern := range systemSupportPatterns {
		if strings.Contains(model, pattern) {
			return true
		}
	}

	return false
}

// SupportsChatFormat checks if a model supports chat format
func (tm *OAuth2TokenManager) SupportsChatFormat(model string) bool {
	if config, exists := tm.modelConfigs[strings.ToLower(model)]; exists {
		return config.SupportsChat
	}

	// Default heuristics
	model = strings.ToLower(model)
	chatPatterns := []string{
		"gpt-", "claude-", "llama", "mistral", "gemini", "chat",
		"instruct", "conversational", "turbo",
	}

	for _, pattern := range chatPatterns {
		if strings.Contains(model, pattern) {
			return true
		}
	}

	return false
}

// SupportsStreaming checks if a model supports streaming responses
func (tm *OAuth2TokenManager) SupportsStreaming(model string) bool {
	if config, exists := tm.modelConfigs[strings.ToLower(model)]; exists {
		return config.SupportsStreaming
	}

	// Default heuristics
	model = strings.ToLower(model)
	streamingPatterns := []string{
		"gpt-", "claude-", "gemini", "api", "turbo", "pro",
		"openai", "anthropic", "google",
	}

	for _, pattern := range streamingPatterns {
		if strings.Contains(model, pattern) {
			return true
		}
	}

	return false
}

// TruncateToFit truncates text to fit within token limits with smart truncation
func (tm *OAuth2TokenManager) TruncateToFit(text string, maxTokens int, model string) (string, error) {
	if maxTokens <= 0 {
		return "", fmt.Errorf("maxTokens must be positive, got %d", maxTokens)
	}

	currentTokens, err := tm.EstimateTokensForModel(model, text)
	if err != nil {
		return "", fmt.Errorf("failed to estimate tokens: %w", err)
	}

	if currentTokens <= maxTokens {
		return text, nil
	}

	// Smart truncation with multiple strategies
	return tm.smartTruncate(text, maxTokens, model, currentTokens)
}

// smartTruncate implements intelligent text truncation
func (tm *OAuth2TokenManager) smartTruncate(text string, maxTokens int, model string, currentTokens int) (string, error) {
	if maxTokens < 10 {
		return "", fmt.Errorf("maxTokens too small for meaningful truncation: %d", maxTokens)
	}

	// Reserve tokens for truncation indicator
	reserveTokens := 5
	targetTokens := maxTokens - reserveTokens

	// Strategy 1: Truncate by sentence boundaries
	sentences := splitSentences(text)
	if len(sentences) > 1 {
		if truncated := tm.truncateBySentences(sentences, targetTokens, model); truncated != "" {
			return truncated + "\n\n[Content truncated...]", nil
		}
	}

	// Strategy 2: Truncate by paragraph boundaries
	paragraphs := strings.Split(text, "\n\n")
	if len(paragraphs) > 1 {
		if truncated := tm.truncateByParagraphs(paragraphs, targetTokens, model); truncated != "" {
			return truncated + "\n\n[Content truncated...]", nil
		}
	}

	// Strategy 3: Word boundary truncation with binary search
	words := strings.Fields(text)
	if len(words) > 10 {
		if truncated := tm.truncateByWords(words, targetTokens, model); truncated != "" {
			return truncated + "... [truncated]", nil
		}
	}

	// Strategy 4: Character-based truncation as fallback
	ratio := float64(targetTokens) / float64(currentTokens)
	targetLength := int(float64(len(text)) * ratio * 0.8) // Safety margin

	if targetLength <= 0 || targetLength >= len(text) {
		return text, nil
	}

	// Find nearest word boundary
	truncated := text[:targetLength]
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > targetLength/2 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "...", nil
}

// GetTokenCount returns estimated token count (simplified)
func (tm *OAuth2TokenManager) GetTokenCount(text string) int {
	return tm.estimateTokensAdvanced(text)
}

// ValidateModel validates if a model is supported and configured
func (tm *OAuth2TokenManager) ValidateModel(model string) error {
	if model == "" {
		return fmt.Errorf("model name cannot be empty")
	}

	// Check against configured models
	if _, exists := tm.modelConfigs[strings.ToLower(model)]; exists {
		return nil
	}

	// Check against known patterns
	model = strings.ToLower(model)
	validPatterns := []string{
		"gpt", "claude", "llama", "mistral", "gemini", "anthropic",
		"openai", "google", "meta", "huggingface", "mock", "test",
	}

	for _, pattern := range validPatterns {
		if strings.Contains(model, pattern) {
			return nil
		}
	}

	return fmt.Errorf("unsupported model: %s", model)
}

// GetSupportedModels returns list of all supported models
func (tm *OAuth2TokenManager) GetSupportedModels() []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var models []string
	for model := range tm.modelConfigs {
		models = append(models, model)
	}

	// Add common model names not in config
	additionalModels := []string{
		"mock-model", "test-model",
	}

	return append(models, additionalModels...)
}

// GetOAuth2Token retrieves or refreshes OAuth2 token
func (tm *OAuth2TokenManager) GetOAuth2Token(ctx context.Context) (*oauth2.Token, error) {
	tm.tokenMu.Lock()
	defer tm.tokenMu.Unlock()

	if tm.oauth2Config == nil {
		return nil, fmt.Errorf("OAuth2 not configured")
	}

	if tm.token != nil && tm.token.Valid() {
		return tm.token, nil
	}

	// Get new token using client credentials flow
	token, err := tm.oauth2Config.PasswordCredentialsToken(ctx, "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth2 token: %w", err)
	}

	tm.token = token
	tm.logger.Info("OAuth2 token refreshed", "expires_at", token.Expiry)

	return token, nil
}

// GetMetrics returns current token usage metrics
func (tm *OAuth2TokenManager) GetMetrics() *TokenMetrics {
	tm.metrics.mu.RLock()
	defer tm.metrics.mu.RUnlock()

	// Return a copy of metrics
	metrics := &TokenMetrics{
		TotalAllocated:   tm.metrics.TotalAllocated,
		TotalReleased:    tm.metrics.TotalReleased,
		CacheHits:        tm.metrics.CacheHits,
		CacheMisses:      tm.metrics.CacheMisses,
		RateLimitHits:    tm.metrics.RateLimitHits,
		ModelEstimations: make(map[string]int64),
	}

	for model, count := range tm.metrics.ModelEstimations {
		metrics.ModelEstimations[model] = count
	}

	return metrics
}

// cleanupCache removes expired cache entries
func (tm *OAuth2TokenManager) cleanupCache() {
	ticker := time.NewTicker(tm.cacheTimeout)
	defer ticker.Stop()

	for range ticker.C {
		tm.cacheMu.Lock()
		now := time.Now()
		for key, cached := range tm.tokenCache {
			if now.After(cached.ExpiresAt) {
				delete(tm.tokenCache, key)
			}
		}
		tm.cacheMu.Unlock()
	}
}

// Helper functions

func hasDigits(s string) bool {
	for _, r := range s {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func hasPunctuation(s string) bool {
	for _, r := range s {
		if unicode.IsPunct(r) {
			return true
		}
	}
	return false
}

func hasUpperCase(s string) bool {
	for _, r := range s {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

func splitSentences(text string) []string {
	re := regexp.MustCompile(`[.!?]+\s+`)
	return re.Split(text, -1)
}

func (tm *OAuth2TokenManager) truncateBySentences(sentences []string, maxTokens int, model string) string {
	var result strings.Builder
	totalTokens := 0

	for _, sentence := range sentences {
		tokens, _ := tm.EstimateTokensForModel(model, sentence)
		if totalTokens+tokens > maxTokens {
			break
		}
		result.WriteString(sentence)
		result.WriteString(". ")
		totalTokens += tokens
	}

	return strings.TrimSpace(result.String())
}

func (tm *OAuth2TokenManager) truncateByParagraphs(paragraphs []string, maxTokens int, model string) string {
	var result strings.Builder
	totalTokens := 0

	for _, paragraph := range paragraphs {
		tokens, _ := tm.EstimateTokensForModel(model, paragraph)
		if totalTokens+tokens > maxTokens {
			break
		}
		result.WriteString(paragraph)
		result.WriteString("\n\n")
		totalTokens += tokens
	}

	return strings.TrimSpace(result.String())
}

func (tm *OAuth2TokenManager) truncateByWords(words []string, maxTokens int, model string) string {
	// Binary search for optimal word count
	left, right := 0, len(words)
	bestResult := ""

	for left <= right {
		mid := (left + right) / 2
		candidate := strings.Join(words[:mid], " ")
		tokens, _ := tm.EstimateTokensForModel(model, candidate)

		if tokens <= maxTokens {
			bestResult = candidate
			left = mid + 1
		} else {
			right = mid - 1
		}
	}

	return bestResult
}

// getDefaultModelConfigs returns default configurations for known models
func getDefaultModelConfigs() map[string]*TokenModelConfig {
	return map[string]*TokenModelConfig{
		"gpt-3.5-turbo": {
			Name:              "gpt-3.5-turbo",
			TokensPerChar:     0.25,
			MaxTokens:         4096,
			SupportsSystem:    true,
			SupportsChat:      true,
			SupportsStreaming: true,
			InputCostPer1K:    0.0015,
			OutputCostPer1K:   0.002,
			TokenizerType:     "gpt",
			ContextWindow:     16385,
		},
		"gpt-4": {
			Name:              "gpt-4",
			TokensPerChar:     0.25,
			MaxTokens:         8192,
			SupportsSystem:    true,
			SupportsChat:      true,
			SupportsStreaming: true,
			InputCostPer1K:    0.03,
			OutputCostPer1K:   0.06,
			TokenizerType:     "gpt",
			ContextWindow:     8192,
		},
		"gpt-4-turbo": {
			Name:              "gpt-4-turbo",
			TokensPerChar:     0.25,
			MaxTokens:         4096,
			SupportsSystem:    true,
			SupportsChat:      true,
			SupportsStreaming: true,
			InputCostPer1K:    0.01,
			OutputCostPer1K:   0.03,
			TokenizerType:     "gpt",
			ContextWindow:     128000,
		},
		"claude-3-opus": {
			Name:              "claude-3-opus",
			TokensPerChar:     0.21,
			MaxTokens:         4096,
			SupportsSystem:    true,
			SupportsChat:      true,
			SupportsStreaming: true,
			InputCostPer1K:    0.015,
			OutputCostPer1K:   0.075,
			TokenizerType:     "claude",
			ContextWindow:     200000,
		},
		"claude-3-sonnet": {
			Name:              "claude-3-sonnet",
			TokensPerChar:     0.21,
			MaxTokens:         4096,
			SupportsSystem:    true,
			SupportsChat:      true,
			SupportsStreaming: true,
			InputCostPer1K:    0.003,
			OutputCostPer1K:   0.015,
			TokenizerType:     "claude",
			ContextWindow:     200000,
		},
		"claude-3-haiku": {
			Name:              "claude-3-haiku",
			TokensPerChar:     0.21,
			MaxTokens:         4096,
			SupportsSystem:    true,
			SupportsChat:      true,
			SupportsStreaming: true,
			InputCostPer1K:    0.00025,
			OutputCostPer1K:   0.00125,
			TokenizerType:     "claude",
			ContextWindow:     200000,
		},
		"llama-2-7b": {
			Name:              "llama-2-7b",
			TokensPerChar:     0.3,
			MaxTokens:         4096,
			SupportsSystem:    false,
			SupportsChat:      true,
			SupportsStreaming: false,
			InputCostPer1K:    0.0002,
			OutputCostPer1K:   0.0002,
			TokenizerType:     "llama",
			ContextWindow:     4096,
		},
		"llama-2-13b": {
			Name:              "llama-2-13b",
			TokensPerChar:     0.3,
			MaxTokens:         4096,
			SupportsSystem:    false,
			SupportsChat:      true,
			SupportsStreaming: false,
			InputCostPer1K:    0.0002,
			OutputCostPer1K:   0.0002,
			TokenizerType:     "llama",
			ContextWindow:     4096,
		},
		"llama-2-70b": {
			Name:              "llama-2-70b",
			TokensPerChar:     0.3,
			MaxTokens:         4096,
			SupportsSystem:    false,
			SupportsChat:      true,
			SupportsStreaming: false,
			InputCostPer1K:    0.0006,
			OutputCostPer1K:   0.0006,
			TokenizerType:     "llama",
			ContextWindow:     4096,
		},
		"gemini-pro": {
			Name:              "gemini-pro",
			TokensPerChar:     0.25,
			MaxTokens:         8192,
			SupportsSystem:    true,
			SupportsChat:      true,
			SupportsStreaming: true,
			InputCostPer1K:    0.0005,
			OutputCostPer1K:   0.0015,
			TokenizerType:     "gpt",
			ContextWindow:     32768,
		},
		"mistral-7b": {
			Name:              "mistral-7b",
			TokensPerChar:     0.28,
			MaxTokens:         4096,
			SupportsSystem:    true,
			SupportsChat:      true,
			SupportsStreaming: false,
			InputCostPer1K:    0.0002,
			OutputCostPer1K:   0.0002,
			TokenizerType:     "llama",
			ContextWindow:     8192,
		},
		"mistral-8x7b": {
			Name:              "mistral-8x7b",
			TokensPerChar:     0.28,
			MaxTokens:         4096,
			SupportsSystem:    true,
			SupportsChat:      true,
			SupportsStreaming: false,
			InputCostPer1K:    0.0007,
			OutputCostPer1K:   0.0007,
			TokenizerType:     "llama",
			ContextWindow:     32768,
		},
	}
}

// CountTokens is an alias for GetTokenCount for consistency
func (tm *OAuth2TokenManager) CountTokens(text string) int {
	return tm.GetTokenCount(text)
}

// CalculateTokenBudgetAdvanced calculates advanced token budget with detailed context analysis
func (tm *OAuth2TokenManager) CalculateTokenBudgetAdvanced(ctx context.Context, model, systemPrompt, userQuery, contextData string) (*TokenBudget, error) {
	// Estimate tokens for each component using model-specific estimation
	systemTokens, _ := tm.EstimateTokensForModel(model, systemPrompt)
	userTokens, _ := tm.EstimateTokensForModel(model, userQuery)
	contextTokens, _ := tm.EstimateTokensForModel(model, contextData)
	
	// Get model configuration for limits
	config, exists := tm.modelConfigs[model]
	maxTokens := 4096 // Default
	if exists {
		maxTokens = config.ContextWindow
	}
	
	totalUsed := systemTokens + userTokens + contextTokens
	responseBudget := maxTokens - totalUsed
	
	// Check if we can accommodate the request
	canAccommodate := responseBudget > 100 // Need at least 100 tokens for response
	
	// Calculate context budget (80% of remaining tokens after system and user)
	contextBudget := int(float64(maxTokens-systemTokens-userTokens) * 0.8)
	if contextBudget < 0 {
		contextBudget = 0
	}
	
	return &TokenBudget{
		CanAccommodate: canAccommodate,
		ContextBudget:  contextBudget,
		SystemTokens:   systemTokens,
		UserTokens:     userTokens,
		ContextTokens:  contextTokens,
		ResponseBudget: responseBudget,
		TotalUsed:      totalUsed,
		MaxTokens:      maxTokens,
	}, nil
}

// OptimizeContext optimizes contexts to fit within token limits using model-specific estimation
func (tm *OAuth2TokenManager) OptimizeContext(contexts []string, maxTokens int, model string) string {
	if len(contexts) == 0 {
		return ""
	}
	
	// Start with all contexts concatenated
	combined := strings.Join(contexts, "\n\n")
	
	// If it fits within limits, return as-is
	currentTokens, _ := tm.EstimateTokensForModel(model, combined)
	if currentTokens <= maxTokens {
		return combined
	}
	
	// Otherwise, truncate contexts from the end until we fit
	totalContexts := len(contexts)
	for i := totalContexts - 1; i >= 0; i-- {
		reduced := strings.Join(contexts[:i+1], "\n\n")
		tokens, _ := tm.EstimateTokensForModel(model, reduced)
		if tokens <= maxTokens {
			if i < totalContexts-1 {
				reduced += "\n\n[... additional context truncated ...]"
			}
			return reduced
		}
	}
	
	// If even a single context is too large, truncate it
	if len(contexts) > 0 {
		truncated, err := tm.TruncateToFit(contexts[0], maxTokens-50, model) // Reserve 50 tokens for truncation message
		if err != nil {
			return "[Context too large to process]"
		}
		return truncated
	}
	
	return ""
}
