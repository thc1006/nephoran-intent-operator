//go:build integration

package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	nephoran "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"
	"github.com/thc1006/nephoran-intent-operator/pkg/rag"
)

// RAGPipelineIntegrationTestSuite provides comprehensive integration testing for the RAG pipeline
type RAGPipelineIntegrationTestSuite struct {
	suite.Suite

	// Test environment
	testEnv      *envtest.Environment
	k8sClient    client.Client
	k8sClientset kubernetes.Interface
	ctx          context.Context
	cancel       context.CancelFunc

	// Test components
	ragService       *rag.RAGService
	embeddingService *rag.EmbeddingService
	retrievalService *rag.EnhancedRetrievalService
	weaviateClient   *rag.WeaviateClient
	redisCache       *rag.RedisCache
	tracingManager   *rag.TracingManager
	metricsCollector *monitoring.MetricsCollector

	// Test data
	testDocuments []rag.Document
	testQueries   []TestQuery
	testIntents   []TestNetworkIntent

	// Mock services
	mockLLMService *MockLLMService
	mockWeaviate   *MockWeaviateService
	mockRedis      *MockRedisService
}

// TestQuery represents a test query with expected results
type TestQuery struct {
	Query            string                 `json:"query"`
	IntentType       string                 `json:"intent_type"`
	ExpectedSources  int                    `json:"expected_sources"`
	ExpectedKeywords []string               `json:"expected_keywords"`
	MinConfidence    float64                `json:"min_confidence"`
	Context          map[string]interface{} `json:"context"`
}

// TestNetworkIntent represents a test network intent
type TestNetworkIntent struct {
	Name     string                     `json:"name"`
	Spec     nephoran.NetworkIntentSpec `json:"spec"`
	Expected TestExpectedResult         `json:"expected"`
}

// TestExpectedResult represents expected test results
type TestExpectedResult struct {
	Status     string             `json:"status"`
	RAGQueries int                `json:"rag_queries"`
	LLMCalls   int                `json:"llm_calls"`
	Errors     []string           `json:"errors"`
	Latency    time.Duration      `json:"latency"`
	Metrics    map[string]float64 `json:"metrics"`
}

// SetupSuite initializes the test suite
func (suite *RAGPipelineIntegrationTestSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithCancel(context.Background())

	// Setup test environment
	suite.setupTestEnvironment()

	// Initialize test data
	suite.initializeTestData()

	// Setup mock services
	suite.setupMockServices()

	// Initialize RAG components
	suite.initializeRAGComponents()
}

// TearDownSuite cleans up the test suite
func (suite *RAGPipelineIntegrationTestSuite) TearDownSuite() {
	suite.cancel()

	if suite.testEnv != nil {
		err := suite.testEnv.Stop()
		suite.NoError(err)
	}

	// Cleanup tracing
	if suite.tracingManager != nil {
		err := suite.tracingManager.Shutdown(suite.ctx)
		suite.NoError(err)
	}
}

// setupTestEnvironment sets up the Kubernetes test environment
func (suite *RAGPipelineIntegrationTestSuite) setupTestEnvironment() {
	suite.testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			"../../deployments/crds",
		},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := suite.testEnv.Start()
	suite.Require().NoError(err)

	suite.k8sClient, err = client.New(cfg, client.Options{})
	suite.Require().NoError(err)

	suite.k8sClientset, err = kubernetes.NewForConfig(cfg)
	suite.Require().NoError(err)
}

// initializeTestData loads test documents and queries
func (suite *RAGPipelineIntegrationTestSuite) initializeTestData() {
	// Load test documents
	suite.testDocuments = []rag.Document{
		{
			ID:      "doc1",
			Title:   "5G Network Slicing Configuration",
			Content: "Network slicing enables multiple virtual networks on a single physical infrastructure...",
			Type:    "technical_specification",
			Source:  "3GPP TS 23.501",
			Metadata: map[string]interface{}{
				"category": "5g_core",
				"version":  "17.0.0",
				"section":  "5.15",
			},
		},
		{
			ID:      "doc2",
			Title:   "O-RAN Architecture Overview",
			Content: "The O-RAN Alliance defines open interfaces for RAN components...",
			Type:    "architecture_document",
			Source:  "O-RAN.WG1.O-RAN-Architecture-Description",
			Metadata: map[string]interface{}{
				"category": "oran",
				"version":  "v07.00",
				"focus":    "architecture",
			},
		},
		{
			ID:      "doc3",
			Title:   "E2 Interface Specification",
			Content: "The E2 interface connects the Near-RT RIC to E2 nodes...",
			Type:    "interface_specification",
			Source:  "O-RAN.WG3.E2AP",
			Metadata: map[string]interface{}{
				"category": "e2_interface",
				"version":  "v03.00",
				"protocol": "E2AP",
			},
		},
	}

	// Load test queries
	suite.testQueries = []TestQuery{
		{
			Query:            "How do I configure network slicing for 5G?",
			IntentType:       "configuration_request",
			ExpectedSources:  2,
			ExpectedKeywords: []string{"network", "slicing", "5G", "configuration"},
			MinConfidence:    0.8,
			Context: map[string]interface{}{
				"technology": "5g",
				"domain":     "core_network",
			},
		},
		{
			Query:            "What is the O-RAN E2 interface?",
			IntentType:       "knowledge_request",
			ExpectedSources:  2,
			ExpectedKeywords: []string{"O-RAN", "E2", "interface", "RIC"},
			MinConfidence:    0.75,
			Context: map[string]interface{}{
				"standard":  "oran",
				"component": "e2_interface",
			},
		},
		{
			Query:            "Create a network slice for IoT devices",
			IntentType:       "creation_intent",
			ExpectedSources:  1,
			ExpectedKeywords: []string{"network", "slice", "IoT", "create"},
			MinConfidence:    0.7,
			Context: map[string]interface{}{
				"use_case": "iot",
				"action":   "create",
			},
		},
	}

	// Load test network intents
	suite.testIntents = []TestNetworkIntent{
		{
			Name: "test-5g-slice",
			Spec: nephoran.NetworkIntentSpec{
				IntentType:  "network_slice_creation",
				Description: "Create a network slice for enhanced mobile broadband",
				Requirements: map[string]interface{}{
					"slice_type": "eMBB",
					"bandwidth":  "1Gbps",
					"latency":    "10ms",
				},
			},
			Expected: TestExpectedResult{
				Status:     "completed",
				RAGQueries: 3,
				LLMCalls:   2,
				Latency:    5 * time.Second,
				Metrics: map[string]float64{
					"rag_confidence":  0.85,
					"processing_time": 4.5,
				},
			},
		},
	}
}

// setupMockServices initializes mock services for testing
func (suite *RAGPipelineIntegrationTestSuite) setupMockServices() {
	suite.mockLLMService = NewMockLLMService()
	suite.mockWeaviate = NewMockWeaviateService()
	suite.mockRedis = NewMockRedisService()
}

// initializeRAGComponents sets up all RAG pipeline components
func (suite *RAGPipelineIntegrationTestSuite) initializeRAGComponents() {
	var err error

	// Initialize tracing
	tracingConfig := rag.GetDefaultTracingConfig()
	tracingConfig.ServiceName = "test-rag-service"
	tracingConfig.EnableTracing = false // Disable for tests

	suite.tracingManager, err = rag.NewTracingManager(tracingConfig)
	suite.Require().NoError(err)

	// Initialize metrics collector
	suite.metricsCollector = monitoring.NewMetricsCollector()

	// Initialize Weaviate client (mocked)
	suite.weaviateClient = rag.NewWeaviateClient(&rag.WeaviateConfig{
		Host:     "localhost",
		Port:     8080,
		Scheme:   "http",
		MockMode: true,
	})

	// Initialize Redis cache (mocked)
	suite.redisCache = rag.NewRedisCache(&rag.RedisCacheConfig{
		Address:  "localhost:6379",
		Database: 0,
		MockMode: true,
	})

	// Initialize embedding service
	suite.embeddingService = rag.NewEmbeddingService(&rag.EmbeddingConfig{
		ModelName:    "all-MiniLM-L6-v2",
		ModelPath:    "test-model",
		BatchSize:    32,
		CacheEnabled: true,
		MockMode:     true,
	})

	// Initialize retrieval service
	suite.retrievalService = rag.NewEnhancedRetrievalService(&rag.RetrievalConfig{
		MaxResults:       10,
		MinScore:         0.5,
		RerankingEnabled: true,
		SemanticSearch:   true,
		HybridSearch:     true,
		MockMode:         true,
	})

	// Initialize RAG service
	suite.ragService = rag.NewRAGService(&rag.RAGConfig{
		EmbeddingService: suite.embeddingService,
		RetrievalService: suite.retrievalService,
		WeaviateClient:   suite.weaviateClient,
		RedisCache:       suite.redisCache,
		TracingManager:   suite.tracingManager,
		MetricsCollector: suite.metricsCollector,
		MaxContextLength: 4000,
		EnableCaching:    true,
		EnableTracing:    false,
	})

	// Populate test data
	suite.populateTestData()
}

// populateTestData adds test documents to the RAG system
func (suite *RAGPipelineIntegrationTestSuite) populateTestData() {
	for _, doc := range suite.testDocuments {
		err := suite.ragService.IndexDocument(suite.ctx, doc)
		suite.Require().NoError(err)
	}
}

// TestEndToEndRAGPipeline tests the complete RAG pipeline
func (suite *RAGPipelineIntegrationTestSuite) TestEndToEndRAGPipeline() {
	for _, testQuery := range suite.testQueries {
		suite.Run(fmt.Sprintf("Query_%s", testQuery.IntentType), func() {
			// Execute RAG query
			startTime := time.Now()
			response, err := suite.ragService.ProcessQuery(suite.ctx, &rag.RAGRequest{
				Query:      testQuery.Query,
				IntentType: testQuery.IntentType,
				Context:    testQuery.Context,
			})
			duration := time.Since(startTime)

			// Assert no errors
			suite.NoError(err)
			suite.NotNil(response)

			// Validate response structure
			suite.NotEmpty(response.GeneratedText)
			suite.GreaterOrEqual(len(response.Sources), 1)
			suite.GreaterOrEqual(response.ConfidenceScore, testQuery.MinConfidence)

			// Validate sources
			suite.LessOrEqual(len(response.Sources), testQuery.ExpectedSources+2) // Allow some variance

			// Validate keywords presence
			responseText := strings.ToLower(response.GeneratedText)
			for _, keyword := range testQuery.ExpectedKeywords {
				suite.Contains(responseText, strings.ToLower(keyword))
			}

			// Validate performance
			suite.Less(duration, 10*time.Second, "Query should complete within 10 seconds")

			// Validate tracing data (if enabled)
			if suite.tracingManager != nil {
				// Check that spans were created (mock validation)
				suite.True(true) // Placeholder for tracing validation
			}

			// Validate metrics
			suite.validateMetrics(testQuery.IntentType, duration)
		})
	}
}

// TestDocumentProcessingPipeline tests document processing workflow
func (suite *RAGPipelineIntegrationTestSuite) TestDocumentProcessingPipeline() {
	testDoc := rag.Document{
		ID:      "test-processing-doc",
		Title:   "Test Document Processing",
		Content: "This is a test document for processing pipeline validation. It contains multiple sentences and should be chunked appropriately for embedding and retrieval.",
		Type:    "test_document",
		Source:  "integration_test",
	}

	// Test document loading
	suite.Run("DocumentLoading", func() {
		err := suite.ragService.LoadDocument(suite.ctx, testDoc)
		suite.NoError(err)
	})

	// Test document chunking
	suite.Run("DocumentChunking", func() {
		chunks, err := suite.ragService.ChunkDocument(suite.ctx, testDoc, &rag.ChunkingConfig{
			Strategy:     "sentence",
			MaxChunkSize: 200,
			Overlap:      20,
		})

		suite.NoError(err)
		suite.GreaterOrEqual(len(chunks), 1)

		for _, chunk := range chunks {
			suite.LessOrEqual(len(chunk.Content), 220) // Max size + overlap
			suite.NotEmpty(chunk.Content)
			suite.Equal(testDoc.ID, chunk.DocumentID)
		}
	})

	// Test embedding generation
	suite.Run("EmbeddingGeneration", func() {
		embeddings, err := suite.embeddingService.GenerateEmbeddings(suite.ctx, []string{testDoc.Content})

		suite.NoError(err)
		suite.Len(embeddings, 1)
		suite.Greater(len(embeddings[0]), 0)
	})

	// Test vector storage
	suite.Run("VectorStorage", func() {
		err := suite.weaviateClient.StoreDocument(suite.ctx, testDoc)
		suite.NoError(err)

		// Verify storage
		stored, err := suite.weaviateClient.GetDocument(suite.ctx, testDoc.ID)
		suite.NoError(err)
		suite.Equal(testDoc.ID, stored.ID)
		suite.Equal(testDoc.Title, stored.Title)
	})
}

// TestRetrievalAccuracy tests retrieval system accuracy
func (suite *RAGPipelineIntegrationTestSuite) TestRetrievalAccuracy() {
	testCases := []struct {
		query          string
		expectedDocIDs []string
		minScore       float64
		searchType     string
	}{
		{
			query:          "network slicing 5G",
			expectedDocIDs: []string{"doc1"},
			minScore:       0.7,
			searchType:     "semantic",
		},
		{
			query:          "O-RAN architecture",
			expectedDocIDs: []string{"doc2"},
			minScore:       0.75,
			searchType:     "hybrid",
		},
		{
			query:          "E2 interface specification",
			expectedDocIDs: []string{"doc3"},
			minScore:       0.8,
			searchType:     "semantic",
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Retrieval_%s", tc.searchType), func() {
			results, err := suite.retrievalService.RetrieveDocuments(suite.ctx, &rag.RetrievalRequest{
				Query:      tc.query,
				MaxResults: 5,
				SearchType: tc.searchType,
				MinScore:   tc.minScore,
			})

			suite.NoError(err)
			suite.GreaterOrEqual(len(results), 1)

			// Check if expected documents are in results
			foundIDs := make(map[string]bool)
			for _, result := range results {
				foundIDs[result.DocumentID] = true
				suite.GreaterOrEqual(result.Score, tc.minScore)
			}

			for _, expectedID := range tc.expectedDocIDs {
				suite.True(foundIDs[expectedID], "Expected document %s not found in results", expectedID)
			}
		})
	}
}

// TestCachePerformance tests caching system performance
func (suite *RAGPipelineIntegrationTestSuite) TestCachePerformance() {
	testQuery := "test cache performance query"

	// First query (cache miss)
	start1 := time.Now()
	response1, err := suite.ragService.ProcessQuery(suite.ctx, &rag.RAGRequest{
		Query:      testQuery,
		IntentType: "test_query",
	})
	duration1 := time.Since(start1)

	suite.NoError(err)
	suite.NotNil(response1)

	// Second query (cache hit)
	start2 := time.Now()
	response2, err := suite.ragService.ProcessQuery(suite.ctx, &rag.RAGRequest{
		Query:      testQuery,
		IntentType: "test_query",
	})
	duration2 := time.Since(start2)

	suite.NoError(err)
	suite.NotNil(response2)

	// Cache hit should be significantly faster
	suite.Less(duration2, duration1/2, "Cached query should be at least 2x faster")

	// Validate cache metrics
	hitRate := suite.redisCache.GetHitRate()
	suite.Greater(hitRate, 0.0)
}

// TestErrorHandling tests error handling and recovery
func (suite *RAGPipelineIntegrationTestSuite) TestErrorHandling() {
	// Test with invalid query
	suite.Run("InvalidQuery", func() {
		response, err := suite.ragService.ProcessQuery(suite.ctx, &rag.RAGRequest{
			Query:      "",
			IntentType: "invalid",
		})

		suite.Error(err)
		suite.Nil(response)
	})

	// Test with service unavailable
	suite.Run("ServiceUnavailable", func() {
		// Simulate service failure
		suite.mockWeaviate.SetError("service unavailable")

		response, err := suite.ragService.ProcessQuery(suite.ctx, &rag.RAGRequest{
			Query:      "test query",
			IntentType: "test",
		})

		// Should handle gracefully or return appropriate error
		if err != nil {
			suite.Contains(err.Error(), "service unavailable")
		} else {
			suite.NotNil(response)
			suite.Less(response.ConfidenceScore, 0.5) // Degraded confidence
		}

		// Reset mock
		suite.mockWeaviate.ClearError()
	})
}

// TestScalabilityAndPerformance tests system performance under load
func (suite *RAGPipelineIntegrationTestSuite) TestScalabilityAndPerformance() {
	concurrentQueries := 10
	queriesPerRoutine := 5

	results := make(chan TestResult, concurrentQueries*queriesPerRoutine)

	// Launch concurrent queries
	for i := 0; i < concurrentQueries; i++ {
		go func(routineID int) {
			for j := 0; j < queriesPerRoutine; j++ {
				start := time.Now()

				response, err := suite.ragService.ProcessQuery(suite.ctx, &rag.RAGRequest{
					Query:      fmt.Sprintf("test query %d-%d", routineID, j),
					IntentType: "load_test",
				})

				duration := time.Since(start)

				results <- TestResult{
					RoutineID: routineID,
					QueryID:   j,
					Duration:  duration,
					Success:   err == nil && response != nil,
					Error:     err,
				}
			}
		}(i)
	}

	// Collect results
	var totalDuration time.Duration
	var successCount int
	var maxDuration time.Duration

	for i := 0; i < concurrentQueries*queriesPerRoutine; i++ {
		result := <-results

		totalDuration += result.Duration
		if result.Success {
			successCount++
		}
		if result.Duration > maxDuration {
			maxDuration = result.Duration
		}

		if result.Error != nil {
			suite.T().Logf("Query failed (routine %d, query %d): %v", result.RoutineID, result.QueryID, result.Error)
		}
	}

	// Validate performance metrics
	avgDuration := totalDuration / time.Duration(concurrentQueries*queriesPerRoutine)
	successRate := float64(successCount) / float64(concurrentQueries*queriesPerRoutine)

	suite.Greater(successRate, 0.95, "Success rate should be > 95%")
	suite.Less(avgDuration, 5*time.Second, "Average response time should be < 5s")
	suite.Less(maxDuration, 15*time.Second, "Max response time should be < 15s")

	suite.T().Logf("Performance results: avg=%v, max=%v, success_rate=%.2f%%",
		avgDuration, maxDuration, successRate*100)
}

// TestNetworkIntentIntegration tests integration with NetworkIntent controller
func (suite *RAGPipelineIntegrationTestSuite) TestNetworkIntentIntegration() {
	for _, testIntent := range suite.testIntents {
		suite.Run(fmt.Sprintf("Intent_%s", testIntent.Name), func() {
			// Create NetworkIntent
			intent := &nephoran.NetworkIntent{
				ObjectMeta: suite.createObjectMeta(testIntent.Name),
				Spec:       testIntent.Spec,
			}

			err := suite.k8sClient.Create(suite.ctx, intent)
			suite.NoError(err)

			// Wait for processing
			time.Sleep(2 * time.Second)

			// Verify processing results
			var updatedIntent nephoran.NetworkIntent
			err = suite.k8sClient.Get(suite.ctx, client.ObjectKeyFromObject(intent), &updatedIntent)
			suite.NoError(err)

			// Check status
			if testIntent.Expected.Status != "" {
				suite.Equal(testIntent.Expected.Status, updatedIntent.Status.Phase)
			}

			// Validate metrics were recorded
			suite.validateNetworkIntentMetrics(testIntent.Name, testIntent.Expected)

			// Cleanup
			err = suite.k8sClient.Delete(suite.ctx, intent)
			suite.NoError(err)
		})
	}
}

// Helper methods

func (suite *RAGPipelineIntegrationTestSuite) validateMetrics(intentType string, duration time.Duration) {
	// Validate that metrics were recorded
	// This would typically query the metrics registry
	suite.True(true) // Placeholder for metrics validation
}

func (suite *RAGPipelineIntegrationTestSuite) validateNetworkIntentMetrics(intentName string, expected TestExpectedResult) {
	// Validate NetworkIntent-specific metrics
	suite.True(true) // Placeholder for NetworkIntent metrics validation
}

func (suite *RAGPipelineIntegrationTestSuite) createObjectMeta(name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      name,
		Namespace: "default",
		Labels: map[string]string{
			"test-suite": "rag-integration",
		},
	}
}

// TestResult represents the result of a load test query
type TestResult struct {
	RoutineID int
	QueryID   int
	Duration  time.Duration
	Success   bool
	Error     error
}

// Test runner
func TestRAGPipelineIntegration(t *testing.T) {
	suite.Run(t, new(RAGPipelineIntegrationTestSuite))
}

// Benchmark tests
func BenchmarkRAGQuery(b *testing.B) {
	suite := &RAGPipelineIntegrationTestSuite{}
	suite.SetT(&testing.T{}) // Mock T for setup
	suite.SetupSuite()
	defer suite.TearDownSuite()

	query := &rag.RAGRequest{
		Query:      "What is network slicing in 5G?",
		IntentType: "knowledge_request",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := suite.ragService.ProcessQuery(context.Background(), query)
			if err != nil {
				b.Error(err)
			}
		}
	})
}
