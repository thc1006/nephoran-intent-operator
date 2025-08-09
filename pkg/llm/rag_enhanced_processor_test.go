package llm

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRAGEnhancedProcessor_ProcessIntent(t *testing.T) {
	// Create a mock base LLM client configuration
	config := &RAGProcessorConfig{
		EnableRAG:              true,
		RAGConfidenceThreshold: 0.6,
		FallbackToBase:         true,
		WeaviateURL:            "http://localhost:8080",
		WeaviateAPIKey:         "",
		MaxContextDocuments:    5,
		QueryTimeout:           30 * time.Second,
		TelecomKeywords: []string{
			"5G", "AMF", "SMF", "UPF", "O-RAN",
		},
		LLMEndpoint:   "http://localhost:8080/v1/chat/completions",
		LLMAPIKey:     "",
		LLMModelName:  "gpt-4o-mini",
		MaxTokens:     2048,
		Temperature:   0.0,
		EnableCaching: true,
		CacheTTL:      5 * time.Minute,
		MaxRetries:    3,
	}

	// Disable RAG for base functionality test
	config.EnableRAG = false
	processor := NewRAGEnhancedProcessorWithConfig(config)

	tests := []struct {
		name           string
		intent         string
		expectError    bool
		expectRAG      bool
		expectFallback bool
	}{
		{
			name:        "Simple deployment intent",
			intent:      "Deploy UPF network function with 3 replicas",
			expectError: false,
			expectRAG:   false, // RAG disabled in config
		},
		{
			name:        "Scaling intent",
			intent:      "Scale AMF to 5 replicas for high availability",
			expectError: false,
			expectRAG:   false,
		},
		{
			name:        "Complex O-RAN intent",
			intent:      "Configure O-RAN Near-RT RIC with xApp support",
			expectError: false,
			expectRAG:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := processor.ProcessIntent(ctx, tt.intent)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Since we're using a mock client, we expect empty results or specific errors
			// The main goal is to test the RAG logic flow
			if err != nil && !strings.Contains(err.Error(), "failed to send request") {
				t.Logf("Got expected network error: %v", err)
			}

			if result != "" {
				t.Logf("Got result: %s", result)
			}
		})
	}
}

func TestRAGEnhancedProcessor_ShouldUseRAG(t *testing.T) {
	config := getDefaultRAGProcessorConfig()
	processor := NewRAGEnhancedProcessorWithConfig(config)

	tests := []struct {
		name      string
		intent    string
		shouldRAG bool
	}{
		{
			name:      "5G keyword should trigger RAG",
			intent:    "Deploy 5G AMF network function",
			shouldRAG: true,
		},
		{
			name:      "O-RAN keyword should trigger RAG",
			intent:    "Configure O-RAN components",
			shouldRAG: true,
		},
		{
			name:      "How-to question should trigger RAG",
			intent:    "How to optimize network performance",
			shouldRAG: true,
		},
		{
			name:      "Generic intent without telecom keywords",
			intent:    "Create a simple web application",
			shouldRAG: false,
		},
		{
			name:      "Explanation request should trigger RAG",
			intent:    "Explain network slicing concepts",
			shouldRAG: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.shouldUseRAG(tt.intent)
			if result != tt.shouldRAG {
				t.Errorf("shouldUseRAG() = %v, want %v for intent: %s", result, tt.shouldRAG, tt.intent)
			}
		})
	}
}

func TestRAGEnhancedProcessor_ClassifyIntentType(t *testing.T) {
	config := getDefaultRAGProcessorConfig()
	processor := NewRAGEnhancedProcessorWithConfig(config)

	tests := []struct {
		name     string
		intent   string
		expected string
	}{
		{
			name:     "Deployment intent",
			intent:   "Deploy UPF network function",
			expected: "NetworkFunctionDeployment",
		},
		{
			name:     "Scaling intent",
			intent:   "Scale AMF to 5 replicas",
			expected: "NetworkFunctionScale",
		},
		{
			name:     "Create intent",
			intent:   "Create SMF instance",
			expected: "NetworkFunctionDeployment",
		},
		{
			name:     "Increase resources",
			intent:   "Increase CPU for UPF",
			expected: "NetworkFunctionScale",
		},
		{
			name:     "Generic intent defaults to deployment",
			intent:   "Configure network settings",
			expected: "NetworkFunctionDeployment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.classifyIntentType(tt.intent)
			if result != tt.expected {
				t.Errorf("classifyIntentType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRAGEnhancedProcessor_BuildEnhancedPrompt(t *testing.T) {
	config := getDefaultRAGProcessorConfig()
	processor := NewRAGEnhancedProcessorWithConfig(config)

	intent := "Deploy UPF network function"
	contextDocs := []map[string]interface{}{
		{
			"title":   "UPF Architecture Guide",
			"content": "UPF is a key component in 5G core network",
			"source":  "3GPP TS 23.501",
		},
	}

	prompt := processor.buildEnhancedPrompt(intent, contextDocs)

	// Check that the prompt contains key elements
	if !strings.Contains(prompt, "CONTEXT FROM KNOWLEDGE BASE") {
		t.Error("Enhanced prompt should contain context section")
	}

	if !strings.Contains(prompt, "UPF Architecture Guide") {
		t.Error("Enhanced prompt should contain context document title")
	}

	if !strings.Contains(prompt, "3GPP TS 23.501") {
		t.Error("Enhanced prompt should contain document source")
	}

	// Test with empty context
	emptyPrompt := processor.buildEnhancedPrompt(intent, []map[string]interface{}{})
	if strings.Contains(emptyPrompt, "CONTEXT FROM KNOWLEDGE BASE") {
		t.Error("Prompt without context should not contain context section")
	}
}

func TestRAGEnhancedProcessor_QueryVectorDatabase(t *testing.T) {
	config := getDefaultRAGProcessorConfig()
	processor := NewRAGEnhancedProcessorWithConfig(config)

	ctx := context.Background()
	query := "5G network architecture"

	// Test with mock data (Weaviate pool is nil in test environment)
	processor.weaviatePool = nil
	_, err := processor.queryVectorDatabase(ctx, query)
	if err == nil || !strings.Contains(err.Error(), "Weaviate connection pool not available") {
		t.Error("Should return error when Weaviate pool is not available")
	}

	// Test with mock setup
	// Since we're using mock data in the implementation, we can test that path
	config.EnableRAG = true
	processor2 := NewRAGEnhancedProcessorWithConfig(config)

	// Force weaviate pool to be available but use mock data
	if processor2.weaviatePool != nil {
		results, err := processor2.queryVectorDatabase(ctx, query)
		if err != nil {
			t.Errorf("Unexpected error with mock setup: %v", err)
		}
		if len(results) == 0 {
			t.Error("Should return mock data when properly configured")
		}
	}
}
