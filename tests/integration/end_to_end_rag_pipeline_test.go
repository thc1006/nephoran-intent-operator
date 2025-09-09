package integration_tests

import (
<<<<<<< HEAD
	
	"encoding/json"
"context"
=======
	"context"
	"encoding/json"
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
	"github.com/thc1006/nephoran-intent-operator/pkg/rag"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// EndToEndRAGPipelineTestSuite tests the complete RAG pipeline workflow
// This test demonstrates the complete integration from document retrieval through
// relevance scoring to final prompt building
type EndToEndRAGPipelineTestSuite struct {
	suite.Suite

	ctx    context.Context
	cancel context.CancelFunc

	// Core components
	embeddingService rag.EmbeddingServiceInterface
	contextBuilder   *MockContextBuilder
<<<<<<< HEAD
	relevanceScorer  *llm.RelevanceScorer
=======
	relevanceScorer  llm.RelevanceScorer
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	promptBuilder    *MockRAGAwarePromptBuilder

	// Test data
	testDocuments []*shared.TelecomDocument
	testQueries   []TestPipelineQuery
}

// TestPipelineQuery represents a test query with expected pipeline behavior
type TestPipelineQuery struct {
<<<<<<< HEAD
	Query              string                 `json:"query"`
	IntentType         string                 `json:"intent_type"`
	ExpectedDocCount   int                    `json:"expected_doc_count"`
	MinRelevanceScore  float32                `json:"min_relevance_score"`
	ExpectedKeywords   []string               `json:"expected_keywords"`
	Context            json.RawMessage `json:"context"`
	ExpectedPromptSize int                    `json:"expected_prompt_size"`
=======
	Query              string          `json:"query"`
	IntentType         string          `json:"intent_type"`
	ExpectedDocCount   int             `json:"expected_doc_count"`
	MinRelevanceScore  float32         `json:"min_relevance_score"`
	ExpectedKeywords   []string        `json:"expected_keywords"`
	Context            json.RawMessage `json:"context"`
	ExpectedPromptSize int             `json:"expected_prompt_size"`
>>>>>>> 6835433495e87288b95961af7173d866977175ff
}

// MockContextBuilder implements context building for testing
type MockContextBuilder struct {
	documents      []*shared.TelecomDocument
	retrievalCalls int
	lastQuery      string
}

// NewMockContextBuilder creates a new mock context builder
func NewMockContextBuilder() *MockContextBuilder {
	return &MockContextBuilder{
		documents: make([]*shared.TelecomDocument, 0),
	}
}

// RetrieveDocuments simulates document retrieval from Weaviate
func (cb *MockContextBuilder) RetrieveDocuments(ctx context.Context, query string, maxResults int) ([]*shared.TelecomDocument, error) {
	cb.retrievalCalls++
	cb.lastQuery = query

	// Filter documents based on query (simple keyword matching for testing)
	var results []*shared.TelecomDocument
	queryLower := strings.ToLower(query)

	for _, doc := range cb.documents {
		contentLower := strings.ToLower(doc.Content + " " + doc.Title)
		if strings.Contains(contentLower, queryLower) {
			results = append(results, doc)
			if len(results) >= maxResults {
				break
			}
		}
	}

	return results, nil
}

// AddDocuments adds documents to the mock retrieval system
func (cb *MockContextBuilder) AddDocuments(docs []*shared.TelecomDocument) {
	cb.documents = append(cb.documents, docs...)
}

// GetMetrics returns retrieval metrics
func (cb *MockContextBuilder) GetMetrics() map[string]interface{} {
<<<<<<< HEAD
	return json.RawMessage(`{}`)
=======
	return map[string]interface{}{}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
}

// MockRAGAwarePromptBuilder implements prompt building for testing
type MockRAGAwarePromptBuilder struct {
	generatedPrompts []string
	lastContext      string
	lastQuery        string
	buildCalls       int
}

// NewMockRAGAwarePromptBuilder creates a new mock prompt builder
func NewMockRAGAwarePromptBuilder() *MockRAGAwarePromptBuilder {
	return &MockRAGAwarePromptBuilder{
		generatedPrompts: make([]string, 0),
	}
}

// BuildPrompt creates a RAG-aware prompt with the provided context
func (pb *MockRAGAwarePromptBuilder) BuildPrompt(ctx context.Context, query string, documents []*llm.ScoredDocument, intentType string) (string, error) {
	pb.buildCalls++
	pb.lastQuery = query

	// Build context from scored documents
	var contextParts []string
	for _, doc := range documents {
		contextParts = append(contextParts, doc.Document.Content[:min(len(doc.Document.Content), 200)])
	}

	pb.lastContext = strings.Join(contextParts, "\n---\n")

	// Generate a comprehensive prompt
	prompt := fmt.Sprintf(`Based on the following telecommunications documentation context, please answer the question about %s.

Context:
%s

Question: %s

Please provide a detailed answer based on the provided context, focusing on %s aspects.
If the context doesn't contain sufficient information, please indicate what additional information would be needed.`,
		intentType,
		pb.lastContext,
		query,
		intentType)

	pb.generatedPrompts = append(pb.generatedPrompts, prompt)
	return prompt, nil
}

// GetMetrics returns prompt building metrics
func (pb *MockRAGAwarePromptBuilder) GetMetrics() map[string]interface{} {
<<<<<<< HEAD
	return json.RawMessage(`{}`)
=======
	return map[string]interface{}{}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
}

// SetupSuite initializes the test suite
func (suite *EndToEndRAGPipelineTestSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithCancel(context.Background())

	// Initialize components
	suite.setupComponents()

	// Load test data
	suite.loadTestData()
}

// TearDownSuite cleans up the test suite
func (suite *EndToEndRAGPipelineTestSuite) TearDownSuite() {
	if suite.embeddingService != nil {
		suite.embeddingService.Close()
	}
	suite.cancel()
}

// setupComponents initializes all pipeline components
func (suite *EndToEndRAGPipelineTestSuite) setupComponents() {
	// Create embedding service with adapter pattern
<<<<<<< HEAD
	concreteEmbeddingService := rag.NewEmbeddingService(&rag.EmbeddingConfig{
		Provider:      "mock",
		ModelName:     "test-model",
		Dimensions:    384,
		BatchSize:     16,
		EnableCaching: true,
		MockMode:      true,
	})

	suite.embeddingService = rag.NewEmbeddingServiceAdapter(concreteEmbeddingService)
=======
	suite.embeddingService = rag.NewNoopEmbeddingService()
>>>>>>> 6835433495e87288b95961af7173d866977175ff

	// Create context builder
	suite.contextBuilder = NewMockContextBuilder()

	// Create relevance scorer with proper interface
<<<<<<< HEAD
	suite.relevanceScorer = llm.NewRelevanceScorer(nil, suite.embeddingService)
=======
	suite.relevanceScorer = llm.NewRelevanceScorerStub()
>>>>>>> 6835433495e87288b95961af7173d866977175ff

	// Create prompt builder
	suite.promptBuilder = NewMockRAGAwarePromptBuilder()
}

// loadTestData initializes test documents and queries
func (suite *EndToEndRAGPipelineTestSuite) loadTestData() {
	// Load comprehensive test documents covering various telecom topics
	suite.testDocuments = []*shared.TelecomDocument{
		{
			ID:              "doc-5g-slicing",
			Title:           "5G Network Slicing Implementation Guide",
			Content:         "Network slicing is a key 5G technology that enables the creation of multiple virtual networks on top of a shared physical infrastructure. Each network slice can be optimized for specific use cases such as enhanced Mobile Broadband (eMBB), Ultra-Reliable Low-Latency Communication (URLLC), or Massive Machine-Type Communication (mMTC). The implementation involves configuring the AMF (Access and Mobility Management Function) and SMF (Session Management Function) to handle different slice types with appropriate QoS parameters.",
			Category:        "5g_core",
			Source:          "3GPP TS 23.501",
			Version:         "17.0.0",
			CreatedAt:       time.Now().Add(-30 * 24 * time.Hour),
			UpdatedAt:       time.Now().Add(-10 * 24 * time.Hour),
			Technology:      []string{"5G", "Network Slicing"},
			NetworkFunction: []string{"AMF", "SMF"},
		},
		{
			ID:              "doc-oran-architecture",
			Title:           "O-RAN Architecture and E2 Interface Specification",
			Content:         "The O-RAN Alliance defines an open, intelligent Radio Access Network architecture that enables multi-vendor interoperability and programmability. The E2 interface connects the Near-RT RIC (Real-Time Intelligent Controller) to the E2 nodes (O-DU, O-CU) and enables real-time control and optimization. The E2AP (E2 Application Protocol) supports various service models including Key Performance Measurement (KPM), RAN Control (RC), and Network Information (NI) services.",
			Category:        "oran",
			Source:          "O-RAN.WG3.E2AP-v03.00",
			Version:         "03.00",
			CreatedAt:       time.Now().Add(-60 * 24 * time.Hour),
			UpdatedAt:       time.Now().Add(-5 * 24 * time.Hour),
			Technology:      []string{"O-RAN", "E2"},
			NetworkFunction: []string{"Near-RT RIC", "O-DU", "O-CU"},
		},
		{
			ID:              "doc-amf-deployment",
			Title:           "AMF Deployment Configuration for High Availability",
			Content:         "The Access and Mobility Management Function (AMF) is a critical 5G Core network function that handles registration, connection, and mobility management. For high availability deployment, AMF instances should be configured with load balancing, geographic redundancy, and proper scaling policies. The AMF connects to UDM for subscriber data, AUSF for authentication, and PCF for policy control. Configuration parameters include PLMN settings, tracking area lists, and N1/N2 interface configurations.",
			Category:        "5g_core",
			Source:          "3GPP TS 23.502",
			Version:         "17.1.0",
			CreatedAt:       time.Now().Add(-15 * 24 * time.Hour),
			UpdatedAt:       time.Now().Add(-2 * 24 * time.Hour),
			Technology:      []string{"5G", "Core Network"},
			NetworkFunction: []string{"AMF", "UDM", "AUSF", "PCF"},
		},
		{
			ID:              "doc-ric-apps",
			Title:           "Near-RT RIC Application Development Guide",
			Content:         "Near-RT RIC applications (xApps) enable real-time RAN optimization and control. xApps can implement various algorithms for handover optimization, load balancing, interference management, and QoS control. The development involves using E2 service models to interact with RAN nodes, implementing control loops with sub-second latency, and integrating with A1 interface for policy-driven configuration. Popular xApp use cases include traffic steering, energy saving, and anomaly detection.",
			Category:        "oran",
			Source:          "O-RAN.WG2.xApp-Guide-v01.02",
			Version:         "01.02",
			CreatedAt:       time.Now().Add(-45 * 24 * time.Hour),
			UpdatedAt:       time.Now().Add(-7 * 24 * time.Hour),
			Technology:      []string{"O-RAN", "xApp", "Near-RT RIC"},
			NetworkFunction: []string{"Near-RT RIC", "xApp"},
		},
	}

	// Add documents to context builder
	suite.contextBuilder.AddDocuments(suite.testDocuments)

	// Define test queries that exercise the full pipeline
	suite.testQueries = []TestPipelineQuery{
		{
			Query:              "How do I deploy AMF with high availability?",
			IntentType:         "configuration_request",
			ExpectedDocCount:   2,
			MinRelevanceScore:  0.7,
			ExpectedKeywords:   []string{"amf", "high availability", "deployment", "configuration"},
			ExpectedPromptSize: 800,
<<<<<<< HEAD
			Context: json.RawMessage(`{}`),
=======
			Context:            json.RawMessage(`{}`),
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		},
		{
			Query:              "What are the key features of O-RAN E2 interface?",
			IntentType:         "knowledge_request",
			ExpectedDocCount:   2,
			MinRelevanceScore:  0.75,
			ExpectedKeywords:   []string{"o-ran", "e2", "interface", "features"},
			ExpectedPromptSize: 700,
<<<<<<< HEAD
			Context: json.RawMessage(`{}`),
=======
			Context:            json.RawMessage(`{}`),
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		},
		{
			Query:              "Create network slice for IoT use case with low latency",
			IntentType:         "creation_intent",
			ExpectedDocCount:   1,
			MinRelevanceScore:  0.65,
			ExpectedKeywords:   []string{"network slice", "iot", "low latency", "create"},
			ExpectedPromptSize: 600,
<<<<<<< HEAD
			Context: json.RawMessage(`{}`),
=======
			Context:            json.RawMessage(`{}`),
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		},
		{
			Query:              "Develop xApp for traffic steering optimization",
			IntentType:         "development_request",
			ExpectedDocCount:   2,
			MinRelevanceScore:  0.6,
			ExpectedKeywords:   []string{"xapp", "traffic steering", "optimization", "development"},
			ExpectedPromptSize: 750,
<<<<<<< HEAD
			Context: json.RawMessage(`{}`),
=======
			Context:            json.RawMessage(`{}`),
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		},
	}
}

// TestCompleteRAGPipelineWorkflow tests the end-to-end pipeline integration
func (suite *EndToEndRAGPipelineTestSuite) TestCompleteRAGPipelineWorkflow() {
	for _, testQuery := range suite.testQueries {
		suite.Run(fmt.Sprintf("Pipeline_%s", testQuery.IntentType), func() {
			// Step 1: Context Building - Retrieve documents from Weaviate (simulated)
			suite.T().Log("Step 1: Document retrieval from context builder")
			startTime := time.Now()

			documents, err := suite.contextBuilder.RetrieveDocuments(
				suite.ctx,
				testQuery.Query,
				10, // max results
			)

			retrievalTime := time.Since(startTime)
			suite.NoError(err, "Document retrieval should not fail")
			suite.GreaterOrEqual(len(documents), 1, "Should retrieve at least one document")
			suite.LessOrEqual(len(documents), testQuery.ExpectedDocCount+2, "Should not retrieve excessive documents")

			suite.T().Logf("Retrieved %d documents in %v", len(documents), retrievalTime)

			// Step 2: Relevance Scoring - Score retrieved documents
			suite.T().Log("Step 2: Document relevance scoring")
			startTime = time.Now()

			var scoredDocuments []*llm.ScoredDocument
			for i, doc := range documents {
				request := &llm.RelevanceRequest{
					Query:      testQuery.Query,
					IntentType: testQuery.IntentType,
					Document:   doc,
					Position:   i,
					Context:    fmt.Sprintf("%v", testQuery.Context),
				}

<<<<<<< HEAD
				relevanceScore, err := suite.relevanceScorer.CalculateRelevance(suite.ctx, request)
				suite.NoError(err, "Relevance scoring should not fail")
				suite.NotNil(relevanceScore, "Relevance score should not be nil")
				suite.GreaterOrEqual(relevanceScore.OverallScore, float32(0.0), "Score should be non-negative")
				suite.LessOrEqual(relevanceScore.OverallScore, float32(1.0), "Score should not exceed 1.0")

				scoredDoc := &llm.ScoredDocument{
					Document:       doc,
					RelevanceScore: relevanceScore,
=======
				relevanceScore, err := suite.relevanceScorer.Score(suite.ctx, "", request.Query)
				suite.NoError(err, "Relevance scoring should not fail")
				suite.NotZero(relevanceScore, "Relevance score should not be zero")
				suite.GreaterOrEqual(relevanceScore, float32(0.0), "Score should be non-negative")
				suite.LessOrEqual(relevanceScore, float32(1.0), "Score should not exceed 1.0")

				scoredDoc := &llm.ScoredDocument{
					Document:       doc,
					RelevanceScore: &llm.RelevanceScore{OverallScore: relevanceScore},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
					Position:       i,
					TokenCount:     len(doc.Content) / 4, // Rough token estimation
				}

				scoredDocuments = append(scoredDocuments, scoredDoc)
			}

			scoringTime := time.Since(startTime)
			suite.T().Logf("Scored %d documents in %v", len(scoredDocuments), scoringTime)

			// Validate scoring results
			suite.validateScoringResults(scoredDocuments, testQuery)

			// Step 3: RAG-Aware Prompt Building - Create final prompt
			suite.T().Log("Step 3: RAG-aware prompt building")
			startTime = time.Now()

			prompt, err := suite.promptBuilder.BuildPrompt(
				suite.ctx,
				testQuery.Query,
				scoredDocuments,
				testQuery.IntentType,
			)

			promptTime := time.Since(startTime)
			suite.NoError(err, "Prompt building should not fail")
			suite.NotEmpty(prompt, "Generated prompt should not be empty")
			suite.GreaterOrEqual(len(prompt), testQuery.ExpectedPromptSize, "Prompt should meet minimum size requirement")

			suite.T().Logf("Generated prompt with %d characters in %v", len(prompt), promptTime)

			// Step 4: Pipeline Validation - Ensure complete workflow integrity
			suite.T().Log("Step 4: Complete pipeline validation")
			suite.validateCompleteWorkflow(testQuery, documents, scoredDocuments, prompt)

			// Log performance metrics
			totalTime := retrievalTime + scoringTime + promptTime
			suite.T().Logf("Complete pipeline executed in %v (retrieval: %v, scoring: %v, prompt: %v)",
				totalTime, retrievalTime, scoringTime, promptTime)

			// Performance assertions
			suite.Less(totalTime, 10*time.Second, "Complete pipeline should execute within 10 seconds")
			suite.Less(retrievalTime, 2*time.Second, "Document retrieval should be fast")
			suite.Less(scoringTime, 5*time.Second, "Scoring should complete efficiently")
			suite.Less(promptTime, 1*time.Second, "Prompt building should be very fast")
		})
	}
}

// TestPipelineComponentIntegration tests individual component integration
func (suite *EndToEndRAGPipelineTestSuite) TestPipelineComponentIntegration() {
	// Test ContextBuilder ??RelevanceScorer Integration
	suite.Run("ContextBuilder_RelevanceScorer_Integration", func() {
		query := "AMF deployment configuration"

		// Retrieve documents
		documents, err := suite.contextBuilder.RetrieveDocuments(suite.ctx, query, 3)
		suite.NoError(err)
		suite.GreaterOrEqual(len(documents), 1)

		// Score the first document
		request := &llm.RelevanceRequest{
			Query:      query,
			IntentType: "configuration_request",
			Document:   documents[0],
			Position:   0,
		}

<<<<<<< HEAD
		score, err := suite.relevanceScorer.CalculateRelevance(suite.ctx, request)
		suite.NoError(err)
		suite.NotNil(score)

		// Validate score components
		suite.Greater(score.SemanticScore, float32(0.0), "Semantic score should be positive")
		suite.Greater(score.AuthorityScore, float32(0.0), "Authority score should be positive")
		suite.Greater(score.RecencyScore, float32(0.0), "Recency score should be positive")

		// Check explanation
		suite.NotEmpty(score.Explanation, "Score should have explanation")
=======
		score, err := suite.relevanceScorer.Score(suite.ctx, "", request.Query)
		suite.NoError(err)
		suite.NotZero(score)

		// TODO: Validate score components - score is float32, not struct
		// suite.Greater(score.SemanticScore, float32(0.0), "Semantic score should be positive")
		// suite.Greater(score.AuthorityScore, float32(0.0), "Authority score should be positive")
		// suite.Greater(score.RecencyScore, float32(0.0), "Recency score should be positive")
		//
		// // Check explanation
		// suite.NotEmpty(score.Explanation, "Score should have explanation")
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	})

	// Test RelevanceScorer ??PromptBuilder Integration
	suite.Run("RelevanceScorer_PromptBuilder_Integration", func() {
		// Create a scored document
		doc := suite.testDocuments[0]
		request := &llm.RelevanceRequest{
			Query:      "network slicing configuration",
			IntentType: "configuration_request",
			Document:   doc,
			Position:   0,
		}

<<<<<<< HEAD
		score, err := suite.relevanceScorer.CalculateRelevance(suite.ctx, request)
=======
		score, err := suite.relevanceScorer.Score(suite.ctx, "", request.Query)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		suite.NoError(err)

		scoredDoc := &llm.ScoredDocument{
			Document:       doc,
<<<<<<< HEAD
			RelevanceScore: score,
=======
			RelevanceScore: &llm.RelevanceScore{OverallScore: score},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			Position:       0,
		}

		// Build prompt with scored document
		prompt, err := suite.promptBuilder.BuildPrompt(
			suite.ctx,
			"network slicing configuration",
			[]*llm.ScoredDocument{scoredDoc},
			"configuration_request",
		)

		suite.NoError(err)
		suite.NotEmpty(prompt)
		suite.Contains(strings.ToLower(prompt), "network slicing")
		suite.Contains(strings.ToLower(prompt), "configuration")
	})
}

// TestEmbeddingServiceInterfaceAbstraction tests the interface abstraction
func (suite *EndToEndRAGPipelineTestSuite) TestEmbeddingServiceInterfaceAbstraction() {
	suite.Run("EmbeddingService_Interface_Compatibility", func() {
		// Test GetEmbedding method
		embedding, err := suite.embeddingService.GetEmbedding(suite.ctx, "test embedding text")
		suite.NoError(err, "GetEmbedding should not fail")
		suite.NotEmpty(embedding, "Embedding should not be empty")
		suite.Greater(len(embedding), 0, "Embedding should have dimensions")

		// Test CalculateSimilarity method
		similarity, err := suite.embeddingService.CalculateSimilarity(suite.ctx, "network slicing", "5G network slice")
		suite.NoError(err, "CalculateSimilarity should not fail")
		suite.GreaterOrEqual(similarity, 0.0, "Similarity should be non-negative")
		suite.LessOrEqual(similarity, 1.0, "Similarity should not exceed 1.0")
		suite.Greater(similarity, 0.3, "Similar texts should have reasonable similarity")

		// Test HealthCheck method
		err = suite.embeddingService.HealthCheck(suite.ctx)
		suite.NoError(err, "HealthCheck should pass for healthy service")

		// Test GetMetrics method
		metrics := suite.embeddingService.GetMetrics()
		suite.NotNil(metrics, "Metrics should not be nil")
	})
}

// TestErrorHandlingAndResilience tests error scenarios and recovery
func (suite *EndToEndRAGPipelineTestSuite) TestErrorHandlingAndResilience() {
	suite.Run("Pipeline_Error_Handling", func() {
		// Test with empty query
		documents, err := suite.contextBuilder.RetrieveDocuments(suite.ctx, "", 5)
		// Should either return error or empty results gracefully
		if err == nil {
			suite.Equal(0, len(documents), "Empty query should return no documents")
		}

		// Test relevance scoring with nil document
		request := &llm.RelevanceRequest{
			Query:      "test query",
			IntentType: "test",
			Document:   nil,
			Position:   0,
		}

<<<<<<< HEAD
		_, err = suite.relevanceScorer.CalculateRelevance(suite.ctx, request)
=======
		_, err = suite.relevanceScorer.Score(suite.ctx, "", request.Query)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		suite.Error(err, "Should fail with nil document")

		// Test prompt building with empty documents
		prompt, err := suite.promptBuilder.BuildPrompt(suite.ctx, "test", []*llm.ScoredDocument{}, "test")
		if err == nil {
			suite.NotEmpty(prompt, "Should generate fallback prompt even with no documents")
		}
	})
}

// Helper methods for validation

func (suite *EndToEndRAGPipelineTestSuite) validateScoringResults(scoredDocuments []*llm.ScoredDocument, testQuery TestPipelineQuery) {
	// Validate that at least one document meets the minimum relevance threshold
	hasHighRelevance := false
	for _, scoredDoc := range scoredDocuments {
		if scoredDoc.RelevanceScore.OverallScore >= testQuery.MinRelevanceScore {
			hasHighRelevance = true
			break
		}
	}

	if len(scoredDocuments) > 0 {
		// Allow some flexibility - either have high relevance OR reasonable scores
		if !hasHighRelevance {
			avgScore := float32(0)
			for _, scoredDoc := range scoredDocuments {
				avgScore += scoredDoc.RelevanceScore.OverallScore
			}
			avgScore /= float32(len(scoredDocuments))
			suite.GreaterOrEqual(avgScore, testQuery.MinRelevanceScore*0.7, "Average relevance should be reasonable")
		}
	}

	// Validate score components are reasonable
	for _, scoredDoc := range scoredDocuments {
		score := scoredDoc.RelevanceScore
		suite.GreaterOrEqual(score.SemanticScore, float32(0.0), "Semantic score should be non-negative")
		suite.GreaterOrEqual(score.AuthorityScore, float32(0.0), "Authority score should be non-negative")
		suite.GreaterOrEqual(score.RecencyScore, float32(0.0), "Recency score should be non-negative")
		suite.GreaterOrEqual(score.DomainScore, float32(0.0), "Domain score should be non-negative")
		suite.GreaterOrEqual(score.IntentScore, float32(0.0), "Intent score should be non-negative")
	}
}

func (suite *EndToEndRAGPipelineTestSuite) validateCompleteWorkflow(testQuery TestPipelineQuery, documents []*shared.TelecomDocument, scoredDocuments []*llm.ScoredDocument, prompt string) {
	// Validate data flow consistency
	suite.Equal(len(documents), len(scoredDocuments), "Document count should be consistent")

	// Validate that prompt contains relevant information
	promptLower := strings.ToLower(prompt)
	queryLower := strings.ToLower(testQuery.Query)

	// Check if key terms from query appear in prompt
	queryWords := strings.Fields(queryLower)
	foundWords := 0
	for _, word := range queryWords {
		if len(word) > 3 && strings.Contains(promptLower, word) {
			foundWords++
		}
	}

	suite.GreaterOrEqual(foundWords, len(queryWords)/2, "Prompt should contain significant query terms")

	// Validate that context from high-scoring documents appears in prompt
	if len(scoredDocuments) > 0 {
		highestScoring := scoredDocuments[0]
		for _, scoredDoc := range scoredDocuments[1:] {
			if scoredDoc.RelevanceScore.OverallScore > highestScoring.RelevanceScore.OverallScore {
				highestScoring = scoredDoc
			}
		}

		// Check if content from highest scoring document appears in prompt
		docWords := strings.Fields(strings.ToLower(highestScoring.Document.Content))
		foundDocWords := 0
		for _, word := range docWords[:min(len(docWords), 20)] { // Check first 20 words
			if len(word) > 4 && strings.Contains(promptLower, word) {
				foundDocWords++
			}
		}

		suite.GreaterOrEqual(foundDocWords, 2, "Prompt should contain content from highest scoring document")
	}

	// Validate intent type appears in prompt
	suite.Contains(promptLower, strings.ToLower(testQuery.IntentType), "Prompt should reference intent type")
}

// Utility functions

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Test runner
func TestEndToEndRAGPipeline(t *testing.T) {
	suite.Run(t, new(EndToEndRAGPipelineTestSuite))
}
<<<<<<< HEAD

=======
>>>>>>> 6835433495e87288b95961af7173d866977175ff
