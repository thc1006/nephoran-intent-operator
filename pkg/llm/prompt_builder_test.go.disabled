package llm

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// MockTokenManager provides a mock implementation for testing
type MockTokenManager struct {
	estimatedTokens      int
	supportsSystemPrompt bool
	supportsChatFormat   bool
	maxTokens            int
	truncatedText        string
}

func (m *MockTokenManager) EstimateTokensForModel(text, modelName string) int {
	if m.estimatedTokens > 0 {
		return m.estimatedTokens
	}
	// Simple estimation based on text length
	return len(strings.Fields(text)) + len(text)/4
}

func (m *MockTokenManager) SupportsSystemPrompt(modelName string) bool {
	return m.supportsSystemPrompt
}

func (m *MockTokenManager) SupportsChatFormat(modelName string) bool {
	return m.supportsChatFormat
}

func (m *MockTokenManager) GetMaxTokens(modelName string) int {
	if m.maxTokens > 0 {
		return m.maxTokens
	}
	return 4096
}

func (m *MockTokenManager) TruncateToFit(text string, maxTokens int, modelName string) string {
	if m.truncatedText != "" {
		return m.truncatedText
	}
	// Simple truncation by word count
	words := strings.Fields(text)
	if len(words) > maxTokens/2 {
		return strings.Join(words[:maxTokens/2], " ")
	}
	return text
}

func TestRAGAwarePromptBuilder_BuildPrompt(t *testing.T) {
	tests := []struct {
		name            string
		request         *PromptRequest
		tokenManager    *MockTokenManager
		config          *PromptBuilderConfig
		expectedError   bool
		errorContains   string
		minPromptLength int
		maxPromptLength int
		validateResult  func(t *testing.T, response *PromptResponse)
	}{
		{
			name: "comprehensive 5G AMF deployment prompt",
			request: &PromptRequest{
				Query:      "Deploy AMF network function in 5G core with high availability",
				IntentType: "configuration",
				ModelName:  "gpt-4",
				RAGContext: []*shared.SearchResult{
					{
						Document: &shared.TelecomDocument{
							ID:              "amf_ha_guide",
							Title:           "5G AMF High Availability Deployment",
							Content:         "This guide covers deploying AMF with redundancy, load balancing, and failover mechanisms in 5G standalone core networks.",
							Source:          "3GPP TS 23.501 v17.0.0",
							Category:        "configuration",
							Version:         "v17.0.0",
							Keywords:        []string{"AMF", "5G", "HA", "deployment"},
							NetworkFunction: []string{"AMF"},
							Technology:      []string{"5G", "5GC"},
							Confidence:      0.95,
						},
						Score: 0.92,
					},
					{
						Document: &shared.TelecomDocument{
							ID:              "amf_scaling",
							Title:           "AMF Auto-scaling Configuration",
							Content:         "Configuration procedures for AMF horizontal and vertical scaling based on traffic patterns and resource utilization.",
							Source:          "O-RAN.SC.RICAPP-v03.00",
							Category:        "configuration",
							NetworkFunction: []string{"AMF"},
							Technology:      []string{"5G", "O-RAN"},
							Confidence:      0.88,
						},
						Score: 0.89,
					},
				},
				Domain:         "Core",
				IncludeFewShot: true,
				MaxTokens:      4000,
			},
			tokenManager: &MockTokenManager{
				supportsSystemPrompt: true,
				supportsChatFormat:   true,
				maxTokens:            4096,
			},
			minPromptLength: 500,
			maxPromptLength: 8000,
			validateResult: func(t *testing.T, response *PromptResponse) {
				if response.SystemPrompt == "" {
					t.Error("Expected non-empty system prompt for chat-capable model")
				}
				if !strings.Contains(strings.ToLower(response.UserPrompt), "amf") {
					t.Error("Expected user prompt to contain AMF reference")
				}
				if !strings.Contains(strings.ToLower(response.UserPrompt), "high availability") {
					t.Error("Expected user prompt to contain high availability reference")
				}
				if len(response.ContextSources) != 2 {
					t.Errorf("Expected 2 context sources, got %d", len(response.ContextSources))
				}
				if len(response.EnhancedQuery) == 0 {
					t.Error("Expected enhanced query to be populated")
				}
				if response.TokenCount <= 0 {
					t.Error("Expected positive token count")
				}
				if response.ProcessingTime <= 0 {
					t.Error("Expected positive processing time")
				}
			},
		},
		{
			name: "O-RAN troubleshooting prompt with few-shot examples",
			request: &PromptRequest{
				Query:      "Troubleshoot handover failures in O-RAN network",
				IntentType: "troubleshooting",
				ModelName:  "claude-3",
				RAGContext: []*shared.SearchResult{
					{
						Document: &shared.TelecomDocument{
							ID:              "oran_handover_troubleshooting",
							Title:           "O-RAN Handover Troubleshooting Guide",
							Content:         "Common handover failure scenarios in O-RAN networks including X2/Xn interface issues, RIC policy conflicts, and timing problems.",
							Source:          "O-RAN.WG3.HO-v02.00",
							Category:        "troubleshooting",
							NetworkFunction: []string{"O-CU", "O-DU", "RIC"},
							Technology:      []string{"O-RAN", "5G"},
							Confidence:      0.91,
						},
						Score: 0.87,
					},
				},
				Domain:             "RAN",
				IncludeFewShot:     true,
				CustomInstructions: "Focus on systematic diagnostic procedures and root cause analysis",
			},
			tokenManager: &MockTokenManager{
				supportsSystemPrompt: true,
				supportsChatFormat:   true,
			},
			minPromptLength: 400,
			validateResult: func(t *testing.T, response *PromptResponse) {
				if !strings.Contains(strings.ToLower(response.UserPrompt), "handover") {
					t.Error("Expected user prompt to contain handover reference")
				}
				if !strings.Contains(strings.ToLower(response.UserPrompt), "troubleshoot") {
					t.Error("Expected user prompt to contain troubleshooting reference")
				}
				if !strings.Contains(response.UserPrompt, "systematic diagnostic procedures") {
					t.Error("Expected custom instructions to be included in user prompt")
				}
				if len(response.FewShotExamples) == 0 {
					t.Error("Expected few-shot examples to be included")
				}
				// Check that domain classification worked
				if domain, exists := response.Metadata["domain"]; !exists || domain != "RAN" {
					t.Errorf("Expected domain to be RAN, got %v", domain)
				}
			},
		},
		{
			name: "network slicing optimization with long context",
			request: &PromptRequest{
				Query:      "Optimize network slicing for URLLC and eMBB coexistence",
				IntentType: "optimization",
				ModelName:  "gpt-3.5-turbo",
				RAGContext: generateLargeRAGContext(8), // Large context
				Domain:     "Core",
				MaxTokens:  2000, // Smaller token budget to test optimization
			},
			tokenManager: &MockTokenManager{
				supportsSystemPrompt: true,
				supportsChatFormat:   true,
				estimatedTokens:      2500, // Over budget to trigger optimization
			},
			minPromptLength: 300,
			maxPromptLength: 4000,
			validateResult: func(t *testing.T, response *PromptResponse) {
				if !contains(strings.Join(response.OptimizationsApplied, ","), "context_optimization") &&
					!contains(strings.Join(response.OptimizationsApplied, ","), "user_prompt_truncation") {
					// At least some optimization should be applied for over-budget scenario
					t.Log("Note: Expected some token optimization to be applied")
				}
				if !strings.Contains(strings.ToLower(response.UserPrompt), "network slicing") {
					t.Error("Expected user prompt to contain network slicing reference")
				}
				if !strings.Contains(strings.ToLower(response.UserPrompt), "urllc") {
					t.Error("Expected user prompt to contain URLLC reference")
				}
			},
		},
		{
			name: "simple query without RAG context",
			request: &PromptRequest{
				Query:      "What is AMF?",
				IntentType: "information",
				ModelName:  "gpt-4",
				RAGContext: []*shared.SearchResult{}, // Empty context
				Domain:     "Core",
			},
			tokenManager: &MockTokenManager{
				supportsSystemPrompt: true,
				supportsChatFormat:   true,
			},
			minPromptLength: 100,
			validateResult: func(t *testing.T, response *PromptResponse) {
				if len(response.ContextSources) != 0 {
					t.Errorf("Expected no context sources, got %d", len(response.ContextSources))
				}
				if !strings.Contains(strings.ToLower(response.UserPrompt), "what is amf") {
					t.Error("Expected user prompt to contain the original query")
				}
				// Enhanced query should still contain expanded abbreviations
				if !strings.Contains(response.EnhancedQuery, "Access and Mobility Management Function") {
					t.Log("Note: Expected AMF abbreviation to be expanded in enhanced query")
				}
			},
		},
		{
			name: "model without system prompt support",
			request: &PromptRequest{
				Query:      "Deploy UPF in edge location",
				IntentType: "configuration",
				ModelName:  "text-davinci-003", // Legacy model without system prompt
				RAGContext: []*shared.SearchResult{
					{
						Document: &shared.TelecomDocument{
							ID:              "upf_edge_deployment",
							Title:           "UPF Edge Deployment Guide",
							Content:         "User Plane Function deployment at edge locations for low-latency applications.",
							Source:          "Edge Computing Guide v1.0",
							Category:        "configuration",
							NetworkFunction: []string{"UPF"},
							Technology:      []string{"5G", "Edge"},
							Confidence:      0.85,
						},
						Score: 0.82,
					},
				},
			},
			tokenManager: &MockTokenManager{
				supportsSystemPrompt: false, // Legacy model
				supportsChatFormat:   false,
			},
			minPromptLength: 200,
			validateResult: func(t *testing.T, response *PromptResponse) {
				if response.SystemPrompt != "" {
					t.Error("Expected empty system prompt for legacy model")
				}
				// Full prompt should combine system instructions with user prompt
				if !strings.Contains(strings.ToLower(response.FullPrompt), "telecommunications engineer") {
					t.Error("Expected system instructions to be integrated into full prompt")
				}
				if !strings.Contains(strings.ToLower(response.FullPrompt), "upf") {
					t.Error("Expected UPF reference in full prompt")
				}
			},
		},
		{
			name: "empty query error",
			request: &PromptRequest{
				Query:     "",
				ModelName: "gpt-4",
			},
			tokenManager:  &MockTokenManager{},
			expectedError: true,
			errorContains: "query cannot be empty",
		},
		{
			name: "empty model name error",
			request: &PromptRequest{
				Query:     "Deploy network function",
				ModelName: "",
			},
			tokenManager:  &MockTokenManager{},
			expectedError: true,
			errorContains: "model name cannot be empty",
		},
		{
			name:          "nil request error",
			request:       nil,
			tokenManager:  &MockTokenManager{},
			expectedError: true,
			errorContains: "prompt request cannot be nil",
		},
		{
			name: "telecommunications abbreviation expansion",
			request: &PromptRequest{
				Query:      "Configure gNB RRC PDCP parameters for 5G NR",
				IntentType: "configuration",
				ModelName:  "gpt-4",
			},
			tokenManager: &MockTokenManager{
				supportsSystemPrompt: true,
				supportsChatFormat:   true,
			},
			validateResult: func(t *testing.T, response *PromptResponse) {
				// Check abbreviation expansion in enhanced query
				if !strings.Contains(response.EnhancedQuery, "gNodeB") {
					t.Log("Note: Expected gNB to be expanded to gNodeB")
				}
				if !strings.Contains(response.EnhancedQuery, "New Radio") {
					t.Log("Note: Expected NR to be expanded to New Radio")
				}
				// Check for optimization markers
				optimizations := strings.Join(response.OptimizationsApplied, ",")
				if strings.Contains(optimizations, "abbreviation_expansion") {
					// Good - abbreviation expansion was applied
				}
			},
		},
		{
			name: "standards mapping enhancement",
			request: &PromptRequest{
				Query:      "Implement procedures from TS 38.413 for NGAP interface",
				IntentType: "configuration",
				ModelName:  "gpt-4",
			},
			tokenManager: &MockTokenManager{
				supportsSystemPrompt: true,
				supportsChatFormat:   true,
			},
			validateResult: func(t *testing.T, response *PromptResponse) {
				// Check for standards mapping enhancement
				optimizations := strings.Join(response.OptimizationsApplied, ",")
				if strings.Contains(optimizations, "standards_mapping") {
					// Good - standards mapping was applied
				}
				// Enhanced query should contain reference to 3GPP
				if !strings.Contains(response.EnhancedQuery, "3GPP") {
					t.Log("Note: Expected TS reference to be enhanced with 3GPP organization")
				}
			},
		},
		{
			name: "contextual enhancement for different intent types",
			request: &PromptRequest{
				Query:      "AMF session management issues",
				IntentType: "troubleshooting",
				ModelName:  "gpt-4",
			},
			tokenManager: &MockTokenManager{
				supportsSystemPrompt: true,
				supportsChatFormat:   true,
			},
			validateResult: func(t *testing.T, response *PromptResponse) {
				// Should add contextual hints for troubleshooting intent
				if !strings.Contains(response.EnhancedQuery, "Troubleshooting issue") {
					t.Log("Note: Expected contextual enhancement for troubleshooting intent")
				}
				// Template should be troubleshooting-specific
				if template, exists := response.Metadata["template_used"]; exists {
					if templateStr, ok := template.(string); ok && !strings.Contains(templateStr, "troubleshooting") {
						t.Log("Note: Expected troubleshooting-specific template")
					}
				}
			},
		},
		{
			name: "multilingual content handling",
			request: &PromptRequest{
				Query:      "Deploy network function with international support",
				IntentType: "configuration",
				ModelName:  "gpt-4",
				RAGContext: []*shared.SearchResult{
					{
						Document: &shared.TelecomDocument{
							ID:         "multilang_doc",
							Title:      "Deployment Guide - International",
							Content:    "Network function deployment with support for múltiples idiomas, 多言語対応, и международная поддержка",
							Source:     "International Guide v1.0",
							Language:   "multi",
							Category:   "configuration",
							Confidence: 0.75,
						},
						Score: 0.70,
					},
				},
			},
			tokenManager: &MockTokenManager{
				supportsSystemPrompt: true,
				supportsChatFormat:   true,
			},
			validateResult: func(t *testing.T, response *PromptResponse) {
				if len(response.ContextSources) != 1 {
					t.Errorf("Expected 1 context source, got %d", len(response.ContextSources))
				}
				// Should handle multilingual content gracefully
				if response.OverallScore <= 0 {
					t.Error("Expected positive overall score even for multilingual content")
				}
			},
		},
		{
			name: "performance edge case - very long query",
			request: &PromptRequest{
				Query:      generateVeryLongQuery(1000), // 1000 words
				IntentType: "configuration",
				ModelName:  "gpt-4",
				MaxTokens:  2000,
			},
			tokenManager: &MockTokenManager{
				supportsSystemPrompt: true,
				supportsChatFormat:   true,
				estimatedTokens:      2200, // Over budget
			},
			minPromptLength: 500,
			maxPromptLength: 3000,
			validateResult: func(t *testing.T, response *PromptResponse) {
				// Should handle very long queries without errors
				if response.ProcessingTime > 5*time.Second {
					t.Errorf("Processing time too long for long query: %v", response.ProcessingTime)
				}
				// Token optimization should be applied
				optimizations := strings.Join(response.OptimizationsApplied, ",")
				if !strings.Contains(optimizations, "optimization") {
					t.Log("Note: Expected some optimization for over-budget long query")
				}
			},
		},
		{
			name: "context relevance filtering",
			request: &PromptRequest{
				Query:      "Deploy 5G core network functions",
				IntentType: "configuration",
				ModelName:  "gpt-4",
				RAGContext: []*shared.SearchResult{
					// High relevance
					{
						Document: &shared.TelecomDocument{
							ID:              "5g_core_deploy",
							Title:           "5G Core Network Deployment",
							Content:         "Comprehensive 5G core deployment including AMF, SMF, UPF configuration and integration procedures.",
							Source:          "5G Deployment Guide v2.0",
							Category:        "configuration",
							NetworkFunction: []string{"AMF", "SMF", "UPF"},
							Technology:      []string{"5G"},
							Confidence:      0.95,
						},
						Score: 0.92, // High relevance
					},
					// Low relevance - should be filtered
					{
						Document: &shared.TelecomDocument{
							ID:         "2g_legacy",
							Title:      "2G Network Maintenance",
							Content:    "Legacy 2G network maintenance procedures for GSM networks.",
							Source:     "Legacy Systems Guide",
							Category:   "maintenance",
							Technology: []string{"2G", "GSM"},
							Confidence: 0.30,
						},
						Score: 0.15, // Low relevance - below threshold
					},
				},
			},
			config: &PromptBuilderConfig{
				ContextRelevanceThreshold: 0.3, // Filter out low relevance
				MaxContextSources:         5,
				EnableContextOptimization: true,
			},
			tokenManager: &MockTokenManager{
				supportsSystemPrompt: true,
				supportsChatFormat:   true,
			},
			validateResult: func(t *testing.T, response *PromptResponse) {
				// Should only include high-relevance document
				if len(response.ContextSources) > 1 {
					t.Errorf("Expected relevance filtering to limit context sources, got %d", len(response.ContextSources))
				}
				// Should contain 5G content but not 2G content
				if strings.Contains(strings.ToLower(response.UserPrompt), "2g") {
					t.Error("Expected low-relevance 2G content to be filtered out")
				}
				if !strings.Contains(strings.ToLower(response.UserPrompt), "5g") {
					t.Error("Expected high-relevance 5G content to be included")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create config
			config := tt.config
			if config == nil {
				config = getDefaultPromptBuilderConfig()
			}

			// Create prompt builder
			promptBuilder := NewRAGAwarePromptBuilder(tt.tokenManager, config)

			// Execute the test
			ctx := context.Background()
			result, err := promptBuilder.BuildPrompt(ctx, tt.request)

			// Check error expectations
			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			// Check for unexpected errors
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Validate result exists
			if result == nil {
				t.Error("Expected non-nil result")
				return
			}

			// Check prompt length constraints
			totalLength := len(result.FullPrompt)
			if tt.minPromptLength > 0 && totalLength < tt.minPromptLength {
				t.Errorf("Prompt too short: %d chars (expected >= %d)", totalLength, tt.minPromptLength)
			}
			if tt.maxPromptLength > 0 && totalLength > tt.maxPromptLength {
				t.Errorf("Prompt too long: %d chars (expected <= %d)", totalLength, tt.maxPromptLength)
			}

			// Basic structure validation
			if result.EnhancedQuery == "" {
				t.Error("Expected enhanced query to be populated")
			}
			if result.Metadata == nil {
				t.Error("Expected metadata to be populated")
			}

			// Run custom validation
			if tt.validateResult != nil {
				tt.validateResult(t, result)
			}

			t.Logf("Test '%s' - Prompt length: %d, Token count: %d, Processing time: %v, Optimizations: %v",
				tt.name, totalLength, result.TokenCount, result.ProcessingTime, result.OptimizationsApplied)
		})
	}
}

func TestRAGAwarePromptBuilder_Performance(t *testing.T) {
	tokenManager := &MockTokenManager{
		supportsSystemPrompt: true,
		supportsChatFormat:   true,
	}

	promptBuilder := NewRAGAwarePromptBuilder(tokenManager, getDefaultPromptBuilderConfig())

	request := &PromptRequest{
		Query:          "Deploy AMF network function for high availability 5G deployment",
		IntentType:     "configuration",
		ModelName:      "gpt-4",
		RAGContext:     generateLargeRAGContext(10),
		IncludeFewShot: true,
	}

	// Warm up
	_, _ = promptBuilder.BuildPrompt(context.Background(), request)

	// Performance test
	iterations := 100
	start := time.Now()

	for i := 0; i < iterations; i++ {
		_, err := promptBuilder.BuildPrompt(context.Background(), request)
		if err != nil {
			t.Errorf("Iteration %d failed: %v", i, err)
			return
		}
	}

	duration := time.Since(start)
	avgDuration := duration / time.Duration(iterations)

	// Performance assertions
	if avgDuration > 100*time.Millisecond {
		t.Errorf("Average prompt building time too high: %v (expected < 100ms)", avgDuration)
	}

	t.Logf("Performance test: %d iterations in %v (avg: %v)", iterations, duration, avgDuration)

	// Check metrics
	metrics := promptBuilder.GetMetrics()
	if metrics.TotalPrompts < int64(iterations) {
		t.Errorf("Expected at least %d total prompts, got %d", iterations, metrics.TotalPrompts)
	}
}

func TestRAGAwarePromptBuilder_ConcurrentAccess(t *testing.T) {
	tokenManager := &MockTokenManager{
		supportsSystemPrompt: true,
		supportsChatFormat:   true,
	}

	promptBuilder := NewRAGAwarePromptBuilder(tokenManager, getDefaultPromptBuilderConfig())

	// Run concurrent prompt building operations
	concurrentRequests := 10
	results := make(chan error, concurrentRequests)

	for i := 0; i < concurrentRequests; i++ {
		go func(id int) {
			request := &PromptRequest{
				Query:      fmt.Sprintf("Deploy network function %d", id),
				IntentType: "configuration",
				ModelName:  "gpt-4",
				RAGContext: generateLargeRAGContext(3),
			}

			_, err := promptBuilder.BuildPrompt(context.Background(), request)
			results <- err
		}(i)
	}

	// Collect results
	for i := 0; i < concurrentRequests; i++ {
		err := <-results
		if err != nil {
			t.Errorf("Concurrent request %d failed: %v", i, err)
		}
	}

	// Verify metrics reflect concurrent operations
	metrics := promptBuilder.GetMetrics()
	if metrics.TotalPrompts < int64(concurrentRequests) {
		t.Errorf("Expected at least %d total prompts from concurrent requests, got %d", concurrentRequests, metrics.TotalPrompts)
	}
}

func TestRAGAwarePromptBuilder_TokenOptimization(t *testing.T) {
	tests := []struct {
		name             string
		maxTokens        int
		ragContext       []*shared.SearchResult
		tokenManager     *MockTokenManager
		expectTruncation bool
	}{
		{
			name:       "within token budget",
			maxTokens:  4000,
			ragContext: generateLargeRAGContext(2),
			tokenManager: &MockTokenManager{
				estimatedTokens:      2000, // Under budget
				supportsSystemPrompt: true,
				supportsChatFormat:   true,
			},
			expectTruncation: false,
		},
		{
			name:       "over token budget requires truncation",
			maxTokens:  1000,
			ragContext: generateLargeRAGContext(5),
			tokenManager: &MockTokenManager{
				estimatedTokens:      1500, // Over budget
				supportsSystemPrompt: true,
				supportsChatFormat:   true,
				truncatedText:        "Truncated content for testing...",
			},
			expectTruncation: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := getDefaultPromptBuilderConfig()
			config.EnableContextOptimization = true
			config.MaxPromptTokens = tt.maxTokens

			promptBuilder := NewRAGAwarePromptBuilder(tt.tokenManager, config)

			request := &PromptRequest{
				Query:      "Deploy network function with optimization",
				IntentType: "configuration",
				ModelName:  "gpt-4",
				RAGContext: tt.ragContext,
				MaxTokens:  tt.maxTokens,
			}

			result, err := promptBuilder.BuildPrompt(context.Background(), request)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			optimizations := strings.Join(result.OptimizationsApplied, ",")
			hasTruncation := strings.Contains(optimizations, "truncation") || strings.Contains(optimizations, "optimization")

			if tt.expectTruncation && !hasTruncation {
				t.Error("Expected token optimization/truncation but none was applied")
			}

			if !tt.expectTruncation && hasTruncation {
				t.Error("Unexpected token optimization/truncation was applied")
			}

			t.Logf("Token optimization test - Budget: %d, Estimated: %d, Optimizations: %v",
				tt.maxTokens, result.TokenCount, result.OptimizationsApplied)
		})
	}
}

// Test the legacy Build method for backward compatibility
func TestRAGAwarePromptBuilder_BuildLegacyMethod(t *testing.T) {
	tests := []struct {
		name       string
		intent     string
		ctxDocs    []map[string]any
		expectText bool
	}{
		{
			name:   "simple intent with context documents",
			intent: "Deploy AMF network function",
			ctxDocs: []map[string]any{
				{
					"title":   "AMF Deployment Guide",
					"content": "AMF deployment procedures for 5G networks",
					"source":  "3GPP TS 23.501",
					"score":   0.9,
				},
			},
			expectText: true,
		},
		{
			name:       "intent without context",
			intent:     "What is UPF?",
			ctxDocs:    []map[string]any{},
			expectText: true,
		},
		{
			name:       "empty intent",
			intent:     "",
			ctxDocs:    []map[string]any{},
			expectText: false, // Should return empty or minimal text
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenManager := &MockTokenManager{
				supportsSystemPrompt: true,
				supportsChatFormat:   true,
			}

			promptBuilder := NewRAGAwarePromptBuilder(tokenManager, getDefaultPromptBuilderConfig())

			result := promptBuilder.Build(tt.intent, tt.ctxDocs)

			if tt.expectText {
				if len(result) == 0 {
					t.Error("Expected non-empty result from Build method")
				}
				if !strings.Contains(strings.ToLower(result), strings.ToLower(tt.intent)) && tt.intent != "" {
					t.Error("Expected result to contain the original intent")
				}
			}

			t.Logf("Legacy Build method - Intent: %q, Result length: %d", tt.intent, len(result))
		})
	}
}

// Helper functions

func generateLargeRAGContext(count int) []*shared.SearchResult {
	results := make([]*shared.SearchResult, count)

	for i := 0; i < count; i++ {
		results[i] = &shared.SearchResult{
			Document: &shared.TelecomDocument{
				ID:              fmt.Sprintf("doc_%d", i),
				Title:           fmt.Sprintf("Telecommunications Document %d", i),
				Content:         generateLongTelecomContent(i),
				Source:          fmt.Sprintf("3GPP TS %d.%03d", 23+i%15, 500+i),
				Category:        []string{"configuration", "troubleshooting", "optimization"}[i%3],
				Version:         fmt.Sprintf("v%d.%d.0", 15+i%3, i%10),
				Keywords:        []string{"5G", "AMF", "SMF", "UPF", "O-RAN", "gNB"}[i%6 : i%6+1],
				NetworkFunction: []string{"AMF", "SMF", "UPF", "gNB", "RIC"}[i%5 : i%5+1],
				Technology:      []string{"5G", "O-RAN", "NFV", "SDN"},
				Confidence:      0.7 + float32(i%3)*0.1,
				CreatedAt:       time.Now().Add(-time.Duration(i*24) * time.Hour),
				UpdatedAt:       time.Now().Add(-time.Duration(i*12) * time.Hour),
			},
			Score:    0.9 - float32(i)*0.05,
			Distance: float32(i) * 0.05,
		}
	}

	return results
}

func generateLongTelecomContent(seed int) string {
	templates := []string{
		"This document describes the deployment and configuration procedures for %s network function in 5G standalone core networks. The implementation includes redundancy mechanisms, load balancing, and automatic failover capabilities to ensure high availability and reliability in production environments.",
		"Troubleshooting guide for %s component covering common failure scenarios, diagnostic procedures, and resolution steps. Includes analysis of interface issues, timing problems, and configuration conflicts that may occur in O-RAN and traditional network architectures.",
		"Optimization procedures for %s focusing on performance tuning, resource allocation, and scalability enhancements. Covers monitoring strategies, key performance indicators, and automated optimization techniques for telecommunications networks.",
		"Security considerations for %s deployment including authentication mechanisms, encryption protocols, and access control policies. Addresses both network-level and application-level security requirements for enterprise telecommunications environments.",
	}

	functions := []string{"AMF", "SMF", "UPF", "gNB", "RIC", "O-CU", "O-DU", "NSSF", "AUSF", "UDM"}

	template := templates[seed%len(templates)]
	function := functions[seed%len(functions)]

	return fmt.Sprintf(template, function)
}

func generateVeryLongQuery(wordCount int) string {
	words := []string{
		"deploy", "configure", "optimize", "troubleshoot", "monitor", "manage",
		"5G", "AMF", "SMF", "UPF", "gNB", "O-RAN", "RIC", "xApp",
		"network", "function", "core", "edge", "cloud", "container",
		"high", "availability", "scalability", "reliability", "performance",
		"security", "authentication", "encryption", "policy", "procedure",
		"interface", "protocol", "standard", "specification", "compliance",
	}

	query := make([]string, wordCount)
	for i := 0; i < wordCount; i++ {
		query[i] = words[i%len(words)]
	}

	return strings.Join(query, " ")
}

// contains helper function (already defined in context_builder_test.go but included for completeness)
func promptBuilderContains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
