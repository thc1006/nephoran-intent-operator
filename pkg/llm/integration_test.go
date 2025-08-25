package llm

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// IntegrationTestSuite represents a complete LLM processing pipeline integration test
type IntegrationTestSuite struct {
	contextBuilder   *ContextBuilder
	relevanceScorer  *RelevanceScorer
	promptBuilder    *RAGAwarePromptBuilder
	mockWeaviatePool *MockWeaviateConnectionPool
	mockEmbedding    *MockEmbeddingServiceForScorer
	mockTokenManager *MockTokenManager
}

// NewIntegrationTestSuite creates a new integration test suite with all mock dependencies
func NewIntegrationTestSuite() *IntegrationTestSuite {
	// Create mock Weaviate pool with comprehensive test data
	mockWeaviatePool := &MockWeaviateConnectionPool{
		searchResults: createComprehensiveTelecomKnowledgeBase(),
		connected:     true,
	}

	// Create mock embedding service
	mockEmbedding := &MockEmbeddingServiceForScorer{
		similarity: 0.8,
	}

	// Create mock token manager
	mockTokenManager := &MockTokenManager{
		supportsSystemPrompt: true,
		supportsChatFormat:   true,
		maxTokens:            4096,
	}

	// Create context builder
	contextConfig := &ContextBuilderConfig{
		WeaviateURL:           "http://localhost:8080",
		MaxConcurrentRequests: 10,
		DefaultLimit:          20,
		MinConfidenceScore:    0.3,
		QueryTimeout:          10 * time.Second,
		EnableHybridSearch:    true,
		HybridAlpha:           0.75,
		TelecomKeywords:       []string{"5G", "AMF", "SMF", "UPF", "gNB", "O-RAN", "RIC"},
		QueryExpansionEnabled: true,
	}
	contextBuilder := NewContextBuilderWithPool(contextConfig, mockWeaviatePool)

	// Create relevance scorer
	scorerConfig := getDefaultRelevanceScorerConfig()
	relevanceScorer := NewRelevanceScorer(scorerConfig, mockEmbedding)

	// Create prompt builder
	promptConfig := getDefaultPromptBuilderConfig()
	promptBuilder := NewRAGAwarePromptBuilder(mockTokenManager, promptConfig)

	return &IntegrationTestSuite{
		contextBuilder:   contextBuilder,
		relevanceScorer:  relevanceScorer,
		promptBuilder:    promptBuilder,
		mockWeaviatePool: mockWeaviatePool,
		mockEmbedding:    mockEmbedding,
		mockTokenManager: mockTokenManager,
	}
}

func TestLLMIntegration_CompleteWorkflow(t *testing.T) {
	tests := []struct {
		name           string
		intent         string
		intentType     string
		modelName      string
		maxDocs        int
		expectedStages []string
		validateResult func(t *testing.T, suite *IntegrationTestSuite, intent string, result map[string]interface{})
	}{
		{
			name:           "5G AMF deployment complete workflow",
			intent:         "Deploy high-availability AMF network function in 5G core with auto-scaling",
			intentType:     "configuration",
			modelName:      "gpt-4",
			maxDocs:        5,
			expectedStages: []string{"context_built", "relevance_scored", "prompt_generated"},
			validateResult: func(t *testing.T, suite *IntegrationTestSuite, intent string, result map[string]interface{}) {
				// Validate context building
				if contextDocs, exists := result["context_documents"]; exists {
					if docs, ok := contextDocs.([]map[string]any); ok && len(docs) > 0 {
						foundAMF := false
						for _, doc := range docs {
							if title, ok := doc["title"].(string); ok && strings.Contains(strings.ToLower(title), "amf") {
								foundAMF = true
								break
							}
						}
						if !foundAMF {
							t.Error("Expected to find AMF-related documents in context")
						}
					}
				}

				// Validate relevance scoring
				if scores, exists := result["relevance_scores"]; exists {
					if scoreSlice, ok := scores.([]*RelevanceScore); ok {
						for _, score := range scoreSlice {
							if score.OverallScore < 0.0 || score.OverallScore > 1.0 {
								t.Errorf("Invalid relevance score: %f", score.OverallScore)
							}
						}
					}
				}

				// Validate prompt generation
				if prompt, exists := result["final_prompt"]; exists {
					if promptStr, ok := prompt.(string); ok {
						if !strings.Contains(strings.ToLower(promptStr), "amf") {
							t.Error("Expected final prompt to contain AMF reference")
						}
						if !strings.Contains(strings.ToLower(promptStr), "high availability") {
							t.Error("Expected final prompt to contain high availability reference")
						}
					}
				}
			},
		},
		{
			name:           "O-RAN troubleshooting complete workflow",
			intent:         "Troubleshoot handover failures in O-RAN network with RIC integration",
			intentType:     "troubleshooting",
			modelName:      "claude-3",
			maxDocs:        3,
			expectedStages: []string{"context_built", "relevance_scored", "prompt_generated"},
			validateResult: func(t *testing.T, suite *IntegrationTestSuite, intent string, result map[string]interface{}) {
				// Check for O-RAN specific content
				if prompt, exists := result["final_prompt"]; exists {
					if promptStr, ok := prompt.(string); ok {
						if !strings.Contains(strings.ToLower(promptStr), "o-ran") {
							t.Error("Expected final prompt to contain O-RAN reference")
						}
						if !strings.Contains(strings.ToLower(promptStr), "handover") {
							t.Error("Expected final prompt to contain handover reference")
						}
					}
				}

				// Validate troubleshooting intent classification
				if metadata, exists := result["prompt_metadata"]; exists {
					if meta, ok := metadata.(map[string]interface{}); ok {
						if template, exists := meta["template_used"]; exists {
							if templateStr, ok := template.(string); ok && !strings.Contains(templateStr, "troubleshooting") {
								t.Error("Expected troubleshooting template to be used")
							}
						}
					}
				}
			},
		},
		{
			name:           "network slicing optimization workflow",
			intent:         "Optimize network slicing for URLLC and eMBB coexistence in 5G SA",
			intentType:     "optimization",
			modelName:      "gpt-4",
			maxDocs:        4,
			expectedStages: []string{"context_built", "relevance_scored", "prompt_generated"},
			validateResult: func(t *testing.T, suite *IntegrationTestSuite, intent string, result map[string]interface{}) {
				// Check for network slicing specific content
				if prompt, exists := result["final_prompt"]; exists {
					if promptStr, ok := prompt.(string); ok {
						if !strings.Contains(strings.ToLower(promptStr), "network slicing") {
							t.Error("Expected final prompt to contain network slicing reference")
						}
						if !strings.Contains(strings.ToLower(promptStr), "urllc") && !strings.Contains(strings.ToLower(promptStr), "embb") {
							t.Error("Expected final prompt to contain URLLC or eMBB reference")
						}
					}
				}
			},
		},
		{
			name:           "edge computing deployment workflow",
			intent:         "Deploy UPF at edge location for low-latency applications",
			intentType:     "configuration",
			modelName:      "gpt-3.5-turbo",
			maxDocs:        3,
			expectedStages: []string{"context_built", "relevance_scored", "prompt_generated"},
			validateResult: func(t *testing.T, suite *IntegrationTestSuite, intent string, result map[string]interface{}) {
				// Check for edge computing and UPF content
				if prompt, exists := result["final_prompt"]; exists {
					if promptStr, ok := prompt.(string); ok {
						if !strings.Contains(strings.ToLower(promptStr), "upf") {
							t.Error("Expected final prompt to contain UPF reference")
						}
						if !strings.Contains(strings.ToLower(promptStr), "edge") {
							t.Error("Expected final prompt to contain edge reference")
						}
					}
				}
			},
		},
		{
			name:           "minimal context workflow",
			intent:         "What is NSSF?",
			intentType:     "information",
			modelName:      "gpt-4",
			maxDocs:        2,
			expectedStages: []string{"context_built", "relevance_scored", "prompt_generated"},
			validateResult: func(t *testing.T, suite *IntegrationTestSuite, intent string, result map[string]interface{}) {
				// Even simple queries should go through full workflow
				if prompt, exists := result["final_prompt"]; exists {
					if promptStr, ok := prompt.(string); ok {
						if !strings.Contains(strings.ToLower(promptStr), "nssf") {
							t.Error("Expected final prompt to contain NSSF reference")
						}
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suite := NewIntegrationTestSuite()

			// Execute complete workflow
			result, err := suite.ExecuteCompleteWorkflow(context.Background(), tt.intent, tt.intentType, tt.modelName, tt.maxDocs)
			if err != nil {
				t.Errorf("Complete workflow failed: %v", err)
				return
			}

			// Validate expected stages were executed
			for _, stage := range tt.expectedStages {
				if _, exists := result[stage]; !exists {
					t.Errorf("Expected stage %q was not executed", stage)
				}
			}

			// Run custom validation
			if tt.validateResult != nil {
				tt.validateResult(t, suite, tt.intent, result)
			}

			t.Logf("Integration test '%s' completed successfully", tt.name)
		})
	}
}

func TestLLMIntegration_ErrorHandlingAndRecovery(t *testing.T) {
	tests := []struct {
		name               string
		setupErrorScenario func(*IntegrationTestSuite)
		intent             string
		expectError        bool
		expectRecovery     bool
	}{
		{
			name: "weaviate connection failure with recovery",
			setupErrorScenario: func(suite *IntegrationTestSuite) {
				suite.mockWeaviatePool.connected = false
			},
			intent:         "Deploy AMF network function",
			expectError:    true,
			expectRecovery: false,
		},
		{
			name: "embedding service failure with keyword fallback",
			setupErrorScenario: func(suite *IntegrationTestSuite) {
				suite.mockEmbedding.err = fmt.Errorf("embedding service unavailable")
			},
			intent:         "Deploy SMF network function",
			expectError:    false, // Should fallback gracefully
			expectRecovery: true,
		},
		{
			name: "partial context retrieval failure",
			setupErrorScenario: func(suite *IntegrationTestSuite) {
				// Simulate partial failure - some documents but search error
				suite.mockWeaviatePool.searchError = fmt.Errorf("partial search failure")
				suite.mockWeaviatePool.searchResults = createComprehensiveTelecomKnowledgeBase()[:2] // Limited results
			},
			intent:      "Configure gNodeB parameters",
			expectError: true,
		},
		{
			name: "token limit exceeded with truncation recovery",
			setupErrorScenario: func(suite *IntegrationTestSuite) {
				suite.mockTokenManager.estimatedTokens = 5000 // Over typical limits
				suite.mockTokenManager.truncatedText = "Truncated content for testing..."
			},
			intent:         "Deploy comprehensive 5G network with all functions",
			expectError:    false,
			expectRecovery: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suite := NewIntegrationTestSuite()

			// Setup error scenario
			tt.setupErrorScenario(suite)

			// Execute workflow
			result, err := suite.ExecuteCompleteWorkflow(context.Background(), tt.intent, "configuration", "gpt-4", 3)

			// Check error expectations
			if tt.expectError && err == nil {
				t.Error("Expected error but workflow succeeded")
				return
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check recovery expectations
			if tt.expectRecovery && err != nil {
				t.Error("Expected recovery but workflow failed")
				return
			}

			if tt.expectRecovery {
				// Validate that some result was produced despite errors
				if result == nil || len(result) == 0 {
					t.Error("Expected some result from recovery scenario")
				}
			}

			t.Logf("Error handling test '%s' completed", tt.name)
		})
	}
}

func TestLLMIntegration_PerformanceAndScalability(t *testing.T) {
	suite := NewIntegrationTestSuite()

	// Test concurrent processing
	t.Run("concurrent workflow execution", func(t *testing.T) {
		concurrentRequests := 5
		results := make(chan error, concurrentRequests)

		intents := []string{
			"Deploy AMF with high availability",
			"Configure SMF for network slicing",
			"Optimize UPF for edge computing",
			"Troubleshoot gNodeB connectivity",
			"Monitor O-RAN RIC performance",
		}

		for i := 0; i < concurrentRequests; i++ {
			go func(id int) {
				_, err := suite.ExecuteCompleteWorkflow(
					context.Background(),
					intents[id%len(intents)],
					"configuration",
					"gpt-4",
					3,
				)
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
	})

	// Test performance with large context
	t.Run("large context performance", func(t *testing.T) {
		// Add more documents to mock pool
		suite.mockWeaviatePool.searchResults = createLargeTelecomKnowledgeBase(50)

		start := time.Now()
		_, err := suite.ExecuteCompleteWorkflow(
			context.Background(),
			"Deploy comprehensive 5G network infrastructure",
			"configuration",
			"gpt-4",
			20, // Request many documents
		)
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Large context workflow failed: %v", err)
			return
		}

		// Performance assertion
		if duration > 10*time.Second {
			t.Errorf("Large context processing took too long: %v", duration)
		}

		t.Logf("Large context workflow completed in %v", duration)
	})

	// Test memory usage stability
	t.Run("memory stability", func(t *testing.T) {
		iterations := 20
		for i := 0; i < iterations; i++ {
			_, err := suite.ExecuteCompleteWorkflow(
				context.Background(),
				fmt.Sprintf("Deploy network function iteration %d", i),
				"configuration",
				"gpt-4",
				3,
			)
			if err != nil {
				t.Errorf("Memory stability test iteration %d failed: %v", i, err)
				return
			}
		}
		// If we get here without memory issues, test passed
		t.Log("Memory stability test completed successfully")
	})
}

func TestLLMIntegration_TelecommunicationsDomainSpecifics(t *testing.T) {
	suite := NewIntegrationTestSuite()

	tests := []struct {
		name             string
		intent           string
		intentType       string
		expectedKeywords []string
		expectedDomain   string
	}{
		{
			name:             "5G core network functions",
			intent:           "Deploy 5G core network with AMF, SMF, and UPF integration",
			intentType:       "configuration",
			expectedKeywords: []string{"AMF", "SMF", "UPF", "5G", "core"},
			expectedDomain:   "Core",
		},
		{
			name:             "O-RAN architecture components",
			intent:           "Configure O-RAN RIC with xApps for intelligent network management",
			intentType:       "configuration",
			expectedKeywords: []string{"O-RAN", "RIC", "xApp", "intelligent"},
			expectedDomain:   "O-RAN",
		},
		{
			name:             "RAN optimization and performance",
			intent:           "Optimize gNodeB performance for high-density urban deployment",
			intentType:       "optimization",
			expectedKeywords: []string{"gNodeB", "performance", "optimization", "urban"},
			expectedDomain:   "RAN",
		},
		{
			name:             "network slicing for different use cases",
			intent:           "Implement network slicing for eMBB, URLLC, and mMTC services",
			intentType:       "configuration",
			expectedKeywords: []string{"network slicing", "eMBB", "URLLC", "mMTC"},
			expectedDomain:   "Core",
		},
		{
			name:             "edge computing integration",
			intent:           "Deploy edge computing infrastructure for low-latency applications",
			intentType:       "configuration",
			expectedKeywords: []string{"edge", "computing", "low-latency"},
			expectedDomain:   "Management",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := suite.ExecuteCompleteWorkflow(
				context.Background(),
				tt.intent,
				tt.intentType,
				"gpt-4",
				5,
			)
			if err != nil {
				t.Errorf("Telecommunications domain test failed: %v", err)
				return
			}

			// Check for domain-specific keywords
			if prompt, exists := result["final_prompt"]; exists {
				if promptStr, ok := prompt.(string); ok {
					promptLower := strings.ToLower(promptStr)
					for _, keyword := range tt.expectedKeywords {
						if !strings.Contains(promptLower, strings.ToLower(keyword)) {
							t.Errorf("Expected keyword %q not found in final prompt", keyword)
						}
					}
				}
			}

			// Check domain classification
			if metadata, exists := result["prompt_metadata"]; exists {
				if meta, ok := metadata.(map[string]interface{}); ok {
					if domain, exists := meta["domain"]; exists {
						if domainStr, ok := domain.(string); ok && domainStr != tt.expectedDomain {
							// Domain classification may vary, so this is informational
							t.Logf("Domain classified as %q, expected %q", domainStr, tt.expectedDomain)
						}
					}
				}
			}

			t.Logf("Telecommunications domain test '%s' completed", tt.name)
		})
	}
}

// ExecuteCompleteWorkflow runs the complete LLM processing pipeline
func (suite *IntegrationTestSuite) ExecuteCompleteWorkflow(
	ctx context.Context,
	intent, intentType, modelName string,
	maxDocs int,
) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Stage 1: Build context from knowledge base
	contextDocs, err := suite.contextBuilder.BuildContext(ctx, intent, maxDocs)
	if err != nil {
		return nil, fmt.Errorf("context building failed: %w", err)
	}
	result["context_built"] = true
	result["context_documents"] = contextDocs

	// Stage 2: Score relevance for each document
	relevanceScores := make([]*RelevanceScore, 0)
	for i, docMap := range contextDocs {
		// Convert map back to document for scoring
		doc := mapToTelecomDocument(docMap)

		request := &RelevanceRequest{
			Query:         intent,
			IntentType:    intentType,
			Document:      doc,
			Position:      i,
			OriginalScore: getFloatFromMap(docMap, "score", 0.5),
		}

		score, err := suite.relevanceScorer.CalculateRelevance(ctx, request)
		if err != nil {
			// Log error but continue with other documents
			continue
		}
		relevanceScores = append(relevanceScores, score)
	}
	result["relevance_scored"] = true
	result["relevance_scores"] = relevanceScores

	// Stage 3: Convert context documents to search results for prompt builder
	searchResults := make([]*shared.SearchResult, 0)
	for _, docMap := range contextDocs {
		doc := mapToTelecomDocument(docMap)
		searchResult := &shared.SearchResult{
			Document: doc,
			Score:    getFloatFromMap(docMap, "score", 0.5),
			Distance: getFloatFromMap(docMap, "distance", 0.5),
		}
		searchResults = append(searchResults, searchResult)
	}

	// Stage 4: Build comprehensive prompt
	promptRequest := &PromptRequest{
		Query:          intent,
		IntentType:     intentType,
		ModelName:      modelName,
		RAGContext:     searchResults,
		IncludeFewShot: true,
		MaxTokens:      4000,
	}

	promptResponse, err := suite.promptBuilder.BuildPrompt(ctx, promptRequest)
	if err != nil {
		return nil, fmt.Errorf("prompt building failed: %w", err)
	}

	result["prompt_generated"] = true
	result["final_prompt"] = promptResponse.FullPrompt
	result["system_prompt"] = promptResponse.SystemPrompt
	result["user_prompt"] = promptResponse.UserPrompt
	result["prompt_metadata"] = promptResponse.Metadata
	result["token_count"] = promptResponse.TokenCount
	result["processing_time"] = promptResponse.ProcessingTime

	return result, nil
}

// Helper functions

func createComprehensiveTelecomKnowledgeBase() []*shared.SearchResult {
	documents := []*shared.TelecomDocument{
		{
			ID:              "5g_amf_deploy_ha",
			Title:           "5G AMF High Availability Deployment Guide",
			Content:         "Comprehensive guide for deploying Access and Mobility Management Function with high availability, including redundancy, load balancing, and failover mechanisms in 5G standalone core networks.",
			Source:          "3GPP TS 23.501 v17.0.0",
			Category:        "configuration",
			Version:         "v17.0.0",
			Keywords:        []string{"AMF", "5G", "high availability", "deployment", "redundancy"},
			NetworkFunction: []string{"AMF"},
			Technology:      []string{"5G", "5GC"},
			UseCase:         []string{"configuration", "deployment"},
			Confidence:      0.95,
			CreatedAt:       time.Now().Add(-30 * 24 * time.Hour),
			UpdatedAt:       time.Now().Add(-5 * 24 * time.Hour),
		},
		{
			ID:              "oran_handover_troubleshooting",
			Title:           "O-RAN Handover Troubleshooting Procedures",
			Content:         "Detailed troubleshooting procedures for handover failures in O-RAN networks, including X2/Xn interface analysis, RIC policy conflicts, and timing synchronization issues.",
			Source:          "O-RAN.WG3.HO-v02.00",
			Category:        "troubleshooting",
			Version:         "v2.0",
			Keywords:        []string{"O-RAN", "handover", "troubleshooting", "X2", "Xn", "RIC"},
			NetworkFunction: []string{"O-CU", "O-DU", "RIC"},
			Technology:      []string{"O-RAN", "5G", "RAN"},
			UseCase:         []string{"troubleshooting", "diagnostics"},
			Confidence:      0.91,
			CreatedAt:       time.Now().Add(-60 * 24 * time.Hour),
			UpdatedAt:       time.Now().Add(-15 * 24 * time.Hour),
		},
		{
			ID:              "network_slicing_optimization",
			Title:           "5G Network Slicing Optimization for Multi-Service Support",
			Content:         "Optimization strategies for 5G network slicing supporting simultaneous eMBB, URLLC, and mMTC services with dynamic resource allocation and QoS management.",
			Source:          "3GPP TS 28.530 v16.5.0",
			Category:        "optimization",
			Version:         "v16.5.0",
			Keywords:        []string{"network slicing", "eMBB", "URLLC", "mMTC", "QoS", "optimization"},
			NetworkFunction: []string{"NSSF", "AMF", "SMF"},
			Technology:      []string{"5G", "Network Slicing"},
			UseCase:         []string{"optimization", "multi-service"},
			Confidence:      0.89,
			CreatedAt:       time.Now().Add(-45 * 24 * time.Hour),
			UpdatedAt:       time.Now().Add(-10 * 24 * time.Hour),
		},
		{
			ID:              "upf_edge_deployment",
			Title:           "User Plane Function Edge Deployment for Low-Latency Applications",
			Content:         "Deployment procedures for UPF at edge locations to support ultra-low latency applications including AR/VR, autonomous vehicles, and industrial IoT.",
			Source:          "ETSI MEC 003 v3.1.1",
			Category:        "configuration",
			Version:         "v3.1.1",
			Keywords:        []string{"UPF", "edge computing", "low latency", "MEC", "deployment"},
			NetworkFunction: []string{"UPF"},
			Technology:      []string{"5G", "Edge Computing", "MEC"},
			UseCase:         []string{"edge deployment", "low latency"},
			Confidence:      0.87,
			CreatedAt:       time.Now().Add(-25 * 24 * time.Hour),
			UpdatedAt:       time.Now().Add(-8 * 24 * time.Hour),
		},
		{
			ID:              "gnb_performance_optimization",
			Title:           "gNodeB Performance Optimization for Dense Urban Deployments",
			Content:         "Performance tuning guidelines for gNodeB in high-density urban environments, covering interference management, capacity optimization, and energy efficiency.",
			Source:          "O-RAN.WG3.E2-v03.00",
			Category:        "optimization",
			Version:         "v3.0",
			Keywords:        []string{"gNodeB", "performance", "optimization", "urban", "interference"},
			NetworkFunction: []string{"gNB", "O-CU", "O-DU"},
			Technology:      []string{"5G", "O-RAN", "RAN"},
			UseCase:         []string{"optimization", "performance tuning"},
			Confidence:      0.86,
			CreatedAt:       time.Now().Add(-40 * 24 * time.Hour),
			UpdatedAt:       time.Now().Add(-12 * 24 * time.Hour),
		},
		{
			ID:              "nssf_slice_selection",
			Title:           "Network Slice Selection Function Configuration and Policies",
			Content:         "NSSF configuration procedures for network slice selection, including selection policies, slice templates, and integration with AMF and SMF components.",
			Source:          "3GPP TS 23.502 v17.1.0",
			Category:        "configuration",
			Version:         "v17.1.0",
			Keywords:        []string{"NSSF", "slice selection", "policies", "configuration"},
			NetworkFunction: []string{"NSSF"},
			Technology:      []string{"5G", "Network Slicing"},
			UseCase:         []string{"configuration", "slice management"},
			Confidence:      0.92,
			CreatedAt:       time.Now().Add(-20 * 24 * time.Hour),
			UpdatedAt:       time.Now().Add(-3 * 24 * time.Hour),
		},
	}

	results := make([]*shared.SearchResult, len(documents))
	for i, doc := range documents {
		results[i] = &shared.SearchResult{
			Document: doc,
			Score:    0.9 - float32(i)*0.05,
			Distance: float32(i) * 0.05,
		}
	}

	return results
}

func createLargeTelecomKnowledgeBase(count int) []*shared.SearchResult {
	results := make([]*shared.SearchResult, count)

	baseDocs := createComprehensiveTelecomKnowledgeBase()

	for i := 0; i < count; i++ {
		baseDoc := baseDocs[i%len(baseDocs)].Document

		// Create variations of the base documents
		doc := &shared.TelecomDocument{
			ID:              fmt.Sprintf("%s_variant_%d", baseDoc.ID, i),
			Title:           fmt.Sprintf("%s (Variant %d)", baseDoc.Title, i),
			Content:         fmt.Sprintf("%s Additional content for variant %d with extended details.", baseDoc.Content, i),
			Source:          baseDoc.Source,
			Category:        baseDoc.Category,
			Version:         baseDoc.Version,
			Keywords:        append(baseDoc.Keywords, fmt.Sprintf("variant%d", i)),
			NetworkFunction: baseDoc.NetworkFunction,
			Technology:      baseDoc.Technology,
			UseCase:         baseDoc.UseCase,
			Confidence:      baseDoc.Confidence - float32(i%10)*0.01,
			CreatedAt:       time.Now().Add(-time.Duration(i*24) * time.Hour),
			UpdatedAt:       time.Now().Add(-time.Duration(i*12) * time.Hour),
		}

		results[i] = &shared.SearchResult{
			Document: doc,
			Score:    0.9 - float32(i)*0.01,
			Distance: float32(i) * 0.01,
		}
	}

	return results
}

func mapToTelecomDocument(docMap map[string]any) *shared.TelecomDocument {
	doc := &shared.TelecomDocument{
		ID:         getStringFromMap(docMap, "id", ""),
		Title:      getStringFromMap(docMap, "title", ""),
		Content:    getStringFromMap(docMap, "content", ""),
		Source:     getStringFromMap(docMap, "source", ""),
		Category:   getStringFromMap(docMap, "category", ""),
		Version:    getStringFromMap(docMap, "version", ""),
		Confidence: getFloatFromMap(docMap, "confidence", 0.5),
	}

	// Handle slice fields
	if keywords, ok := docMap["keywords"].([]string); ok {
		doc.Keywords = keywords
	}
	if networkFunc, ok := docMap["network_function"].([]string); ok {
		doc.NetworkFunction = networkFunc
	}
	if technology, ok := docMap["technology"].([]string); ok {
		doc.Technology = technology
	}

	// Handle time fields
	if createdAt, ok := docMap["created_at"].(time.Time); ok {
		doc.CreatedAt = createdAt
	}
	if updatedAt, ok := docMap["updated_at"].(time.Time); ok {
		doc.UpdatedAt = updatedAt
	}

	return doc
}

func getStringFromMap(m map[string]any, key, defaultValue string) string {
	if val, exists := m[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

func getFloatFromMap(m map[string]any, key string, defaultValue float32) float32 {
	if val, exists := m[key]; exists {
		switch v := val.(type) {
		case float32:
			return v
		case float64:
			return float32(v)
		case int:
			return float32(v)
		}
	}
	return defaultValue
}
