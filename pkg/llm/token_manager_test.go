package llm

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTokenManager(t *testing.T) {
	tm := NewTokenManager()
	assert.NotNil(t, tm)
	assert.NotNil(t, tm.modelConfigs)
	assert.True(t, len(tm.modelConfigs) > 0)
}

func TestTokenManager_EstimateTokens(t *testing.T) {
	tm := NewTokenManager()

	testCases := []struct {
		text     string
		expected int
		model    string
	}{
		{"Hello world", 2, "gpt-4o-mini"},
		{"", 0, "gpt-4o-mini"},
		{"This is a longer sentence with more words that should result in more tokens.", 15, "gpt-4o-mini"},
		{"Deploy AMF with 3 replicas in production namespace", 9, "gpt-4o-mini"},
		{"Configure 5G network slice for enhanced mobile broadband applications", 10, "gpt-4o-mini"},
	}

	for _, tc := range testCases {
		tokens := tm.EstimateTokens(tc.text, tc.model)
		// Allow some variance in token estimation
		assert.InDelta(t, tc.expected, tokens, float64(tc.expected)*0.3, 
			"Token estimation for '%s' should be approximately %d, got %d", tc.text, tc.expected, tokens)
	}
}

func TestTokenManager_CalculateTokenBudget(t *testing.T) {
	tm := NewTokenManager()
	ctx := context.Background()

	systemPrompt := "You are a helpful assistant for network operations."
	userPrompt := "Deploy AMF with 3 replicas"
	ragContext := "AMF (Access and Mobility Management Function) is a key component of 5G core network..."

	budget, err := tm.CalculateTokenBudget(ctx, "gpt-4o-mini", systemPrompt, userPrompt, ragContext)
	
	assert.NoError(t, err)
	assert.NotNil(t, budget)
	assert.True(t, budget.SystemPromptTokens > 0)
	assert.True(t, budget.UserPromptTokens > 0)
	assert.True(t, budget.ContextTokens > 0)
	assert.True(t, budget.MaxResponseTokens > 0)
	assert.True(t, budget.TotalUsedTokens > 0)
	assert.True(t, budget.RemainingTokens > 0)
	assert.True(t, budget.CanAccommodate)
}

func TestTokenManager_CalculateTokenBudget_ExceedsLimit(t *testing.T) {
	tm := NewTokenManager()
	ctx := context.Background()

	// Create very long context that exceeds token limit
	longContext := strings.Repeat("This is a very long context that will exceed the token limit. ", 1000)

	budget, err := tm.CalculateTokenBudget(ctx, "gpt-4o-mini", "System prompt", "User prompt", longContext)
	
	assert.NoError(t, err)
	assert.NotNil(t, budget)
	assert.False(t, budget.CanAccommodate)
	assert.Equal(t, 0, budget.RemainingTokens)
}

func TestTokenManager_OptimizeContext(t *testing.T) {
	tm := NewTokenManager()

	contexts := []RetrievedContext{
		{
			Content:   "First context about AMF deployment with detailed configuration parameters.",
			Relevance: 0.9,
			Source:    "3GPP TS 23.501",
		},
		{
			Content:   "Second context about UPF configuration and performance optimization.",
			Relevance: 0.8,
			Source:    "3GPP TS 23.502",
		},
		{
			Content:   "Third context about network slicing with comprehensive implementation guide.",
			Relevance: 0.7,
			Source:    "O-RAN WG1",
		},
		{
			Content:   "Fourth context with less relevant information about general networking.",
			Relevance: 0.5,
			Source:    "RFC 7950",
		},
	}

	// Test with sufficient budget
	optimized := tm.OptimizeContext(contexts, 500, "gpt-4o-mini")
	assert.True(t, len(optimized) > 0)
	assert.True(t, len(optimized) <= len(contexts))

	// Verify contexts are sorted by relevance
	for i := 1; i < len(optimized); i++ {
		assert.True(t, optimized[i-1].Relevance >= optimized[i].Relevance)
	}

	// Test with very limited budget
	optimized = tm.OptimizeContext(contexts, 50, "gpt-4o-mini")
	assert.True(t, len(optimized) >= 1) // Should keep at least the most relevant one
	assert.Equal(t, 0.9, optimized[0].Relevance) // Should be the highest relevance
}

func TestTokenManager_OptimizeContext_EmptyInput(t *testing.T) {
	tm := NewTokenManager()

	// Test with empty contexts
	optimized := tm.OptimizeContext([]RetrievedContext{}, 1000, "gpt-4o-mini")
	assert.Empty(t, optimized)

	// Test with zero budget
	contexts := []RetrievedContext{
		{Content: "Test content", Relevance: 0.8, Source: "test"},
	}
	optimized = tm.OptimizeContext(contexts, 0, "gpt-4o-mini")
	assert.Empty(t, optimized)
}

func TestTokenManager_GetModelConfig(t *testing.T) {
	tm := NewTokenManager()

	// Test existing model
	config, exists := tm.GetModelConfig("gpt-4o-mini")
	assert.True(t, exists)
	assert.NotNil(t, config)
	assert.True(t, config.MaxTokens > 0)
	assert.True(t, config.ContextWindow > 0)

	// Test non-existing model
	config, exists = tm.GetModelConfig("non-existent-model")
	assert.False(t, exists)
	assert.Nil(t, config)
}

func TestTokenManager_RegisterModel(t *testing.T) {
	tm := NewTokenManager()

	newConfig := &ModelTokenConfig{
		MaxTokens:            2048,
		ContextWindow:        8192,
		ReservedTokens:       512,
		PromptTokens:         256,
		SafetyMargin:         0.1,
		TokensPerChar:        0.25,
		TokensPerWord:        1.3,
		SupportsChatFormat:   true,
		SupportsSystemPrompt: true,
		SupportsStreaming:    false,
	}

	err := tm.RegisterModel("test-model", newConfig)
	assert.NoError(t, err)

	// Verify model is registered
	config, exists := tm.GetModelConfig("test-model")
	assert.True(t, exists)
	assert.Equal(t, newConfig, config)

	// Test registering existing model (should update)
	updatedConfig := &ModelTokenConfig{
		MaxTokens:     4096,
		ContextWindow: 16384,
	}

	err = tm.RegisterModel("test-model", updatedConfig)
	assert.NoError(t, err)

	config, exists = tm.GetModelConfig("test-model")
	assert.True(t, exists)
	assert.Equal(t, updatedConfig, config)
}

func TestTokenManager_EstimateTokens_DifferentModels(t *testing.T) {
	tm := NewTokenManager()

	text := "Deploy AMF network function with high availability configuration"

	models := []string{"gpt-4o", "gpt-4o-mini", "claude-3-haiku"}

	for _, model := range models {
		tokens := tm.EstimateTokens(text, model)
		assert.True(t, tokens > 0, "Model %s should estimate tokens for text", model)
		
		// Verify different models may have different estimations
		if model == "gpt-4o-mini" {
			// This is our baseline - should be reasonable
			assert.InDelta(t, 11, tokens, 3, "GPT-4o-mini estimation should be around 11 tokens")
		}
	}
}

func TestTokenManager_EstimateTokens_TechnicalContent(t *testing.T) {
	tm := NewTokenManager()

	testCases := []struct {
		name        string
		content     string
		expectHigh  bool
		description string
	}{
		{
			name: "code_snippet",
			content: `
func deployAMF(replicas int) error {
    return k8s.Deploy("amf", replicas)
}`,
			expectHigh:  true,
			description: "Code should have higher token density",
		},
		{
			name: "yaml_config",
			content: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: amf-deployment
spec:
  replicas: 3`,
			expectHigh:  true,
			description: "YAML should have higher token density",
		},
		{
			name: "json_config",
			content: `{"networkFunction": "AMF", "replicas": 3, "namespace": "5g-core"}`,
			expectHigh:  true,
			description: "JSON should have higher token density",
		},
		{
			name: "natural_language",
			content: "Please deploy the AMF network function with three replicas in the production environment.",
			expectHigh:  false,
			description: "Natural language has normal token density",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tokens := tm.EstimateTokens(tc.content, "gpt-4o-mini")
			words := len(strings.Fields(tc.content))
			ratio := float64(tokens) / float64(words)

			if tc.expectHigh {
				assert.True(t, ratio > 1.0, "%s: Expected high token-to-word ratio, got %.2f", tc.description, ratio)
			} else {
				assert.True(t, ratio <= 1.5, "%s: Expected normal token-to-word ratio, got %.2f", tc.description, ratio)
			}
		})
	}
}

func TestTokenManager_TruncateToFit(t *testing.T) {
	tm := NewTokenManager()

	longText := strings.Repeat("This is a sentence that will be repeated many times. ", 100)
	
	// Test truncation to fit within budget
	truncated := tm.TruncateToFit(longText, 50, "gpt-4o-mini")
	
	truncatedTokens := tm.EstimateTokens(truncated, "gpt-4o-mini")
	assert.True(t, truncatedTokens <= 50, "Truncated text should fit within token budget")
	assert.True(t, len(truncated) < len(longText), "Truncated text should be shorter than original")
	
	// Test with text that's already short enough
	shortText := "Short text"
	truncated = tm.TruncateToFit(shortText, 100, "gpt-4o-mini")
	assert.Equal(t, shortText, truncated, "Short text should not be truncated")
}

func TestTokenManager_CalculateCost(t *testing.T) {
	tm := NewTokenManager()

	// Test cost calculation for different models
	testCases := []struct {
		model        string
		inputTokens  int
		outputTokens int
		expectCost   bool
	}{
		{"gpt-4o", 1000, 500, true},
		{"gpt-4o-mini", 1000, 500, true},
		{"claude-3-haiku", 1000, 500, true},
		{"unknown-model", 1000, 500, false},
	}

	for _, tc := range testCases {
		cost, err := tm.CalculateCost(tc.model, tc.inputTokens, tc.outputTokens)
		
		if tc.expectCost {
			assert.NoError(t, err)
			assert.True(t, cost > 0, "Cost should be positive for model %s", tc.model)
		} else {
			assert.Error(t, err)
			assert.Equal(t, 0.0, cost)
		}
	}
}

func TestTokenManager_GetSupportedModels(t *testing.T) {
	tm := NewTokenManager()

	models := tm.GetSupportedModels()
	assert.True(t, len(models) > 0, "Should have supported models")

	// Check for expected models
	expectedModels := []string{"gpt-4o", "gpt-4o-mini", "claude-3-haiku"}
	for _, expected := range expectedModels {
		assert.Contains(t, models, expected, "Should contain model %s", expected)
	}
}

func TestTokenManager_IsValidModel(t *testing.T) {
	tm := NewTokenManager()

	validModels := []string{"gpt-4o", "gpt-4o-mini", "claude-3-haiku"}
	for _, model := range validModels {
		assert.True(t, tm.IsValidModel(model), "Model %s should be valid", model)
	}

	invalidModels := []string{"invalid-model", "", "gpt-5", "unknown"}
	for _, model := range invalidModels {
		assert.False(t, tm.IsValidModel(model), "Model %s should be invalid", model)
	}
}

func TestTokenManager_ContextOptimization_WordBoundary(t *testing.T) {
	tm := NewTokenManager()

	// Test that truncation respects word boundaries
	text := "This is a complete sentence that should be truncated at word boundaries not in the middle of words."
	
	// Truncate to a size that would normally cut through a word
	truncated := tm.TruncateToFit(text, 10, "gpt-4o-mini")
	
	// Verify truncation ends at word boundary
	assert.True(t, strings.HasSuffix(truncated, " ") || !strings.Contains(truncated, " "), 
		"Truncated text should end at word boundary")
	
	words := strings.Fields(truncated)
	for _, word := range words {
		assert.True(t, len(word) > 0, "No empty words should exist")
	}
}

func TestTokenManager_EdgeCases(t *testing.T) {
	tm := NewTokenManager()

	// Test with empty strings
	assert.Equal(t, 0, tm.EstimateTokens("", "gpt-4o-mini"))
	
	// Test with only whitespace
	assert.Equal(t, 0, tm.EstimateTokens("   \n\t  ", "gpt-4o-mini"))
	
	// Test with special characters
	tokens := tm.EstimateTokens("!@#$%^&*()_+-=[]{}|;':\",./<>?", "gpt-4o-mini")
	assert.True(t, tokens > 0, "Special characters should have token count")
	
	// Test with unicode characters
	tokens = tm.EstimateTokens("Hello 世界 🌍", "gpt-4o-mini")
	assert.True(t, tokens > 0, "Unicode characters should be handled")
	
	// Test truncation with very small budget
	truncated := tm.TruncateToFit("This is a test", 1, "gpt-4o-mini")
	assert.True(t, len(truncated) > 0, "Should return at least something even with tiny budget")
}

func TestTokenManager_Budget_SafetyMargin(t *testing.T) {
	tm := NewTokenManager()
	ctx := context.Background()

	// Test that safety margin is applied correctly
	budget, err := tm.CalculateTokenBudget(ctx, "gpt-4o-mini", "System", "User", "Context")
	
	assert.NoError(t, err)
	assert.NotNil(t, budget)
	
	// The remaining tokens should account for safety margin
	config, _ := tm.GetModelConfig("gpt-4o-mini")
	expectedMax := int(float64(config.ContextWindow) * (1.0 - config.SafetyMargin))
	
	assert.True(t, budget.TotalUsedTokens+budget.RemainingTokens <= expectedMax,
		"Total budget should not exceed context window minus safety margin")
}

// Benchmark tests
func BenchmarkTokenManager_EstimateTokens(b *testing.B) {
	tm := NewTokenManager()
	text := "Deploy AMF network function with 3 replicas in production namespace for enhanced mobile broadband service"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tm.EstimateTokens(text, "gpt-4o-mini")
	}
}

func BenchmarkTokenManager_CalculateTokenBudget(b *testing.B) {
	tm := NewTokenManager()
	ctx := context.Background()
	
	systemPrompt := "You are a helpful assistant for network operations."
	userPrompt := "Deploy AMF with 3 replicas"
	ragContext := "AMF (Access and Mobility Management Function) is a key component of 5G core network that handles access and mobility management procedures."
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tm.CalculateTokenBudget(ctx, "gpt-4o-mini", systemPrompt, userPrompt, ragContext)
	}
}

func BenchmarkTokenManager_OptimizeContext(b *testing.B) {
	tm := NewTokenManager()
	
	contexts := make([]RetrievedContext, 10)
	for i := 0; i < 10; i++ {
		contexts[i] = RetrievedContext{
			Content:   strings.Repeat("This is context content for testing optimization. ", 20),
			Relevance: float64(10-i) / 10.0, // Decreasing relevance
			Source:    "test-source",
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tm.OptimizeContext(contexts, 1000, "gpt-4o-mini")
	}
}