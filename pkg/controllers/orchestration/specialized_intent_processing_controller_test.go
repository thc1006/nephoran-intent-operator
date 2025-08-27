/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/interfaces"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
	"github.com/thc1006/nephoran-intent-operator/pkg/rag"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// Mock implementations for testing
type MockLLMClient struct {
	responses         map[string]string
	shouldReturnError bool
	responseDelay     time.Duration
	tokenUsage        int
}

func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		responses: map[string]string{
			"default": `{
				"network_functions": [
					{
						"name": "amf-instance",
						"type": "amf",
						"image": "nephoran/amf:v1.0.0",
						"replicas": 2,
						"resources": {
							"requests": {"cpu": "500m", "memory": "1Gi"},
							"limits": {"cpu": "2", "memory": "4Gi"}
						}
					}
				],
				"deployment_pattern": "high-availability",
				"confidence": 0.95,
				"resources": {"cpu": "1", "memory": "2Gi", "storage": "20Gi"}
			}`,
			"low-confidence": `{
				"network_functions": [],
				"deployment_pattern": "unknown",
				"confidence": 0.3,
				"resources": {}
			}`,
			"5g-deployment": `{
				"network_functions": [
					{
						"name": "amf",
						"type": "amf",
						"replicas": 3,
						"resources": {"requests": {"cpu": "1", "memory": "2Gi"}}
					},
					{
						"name": "smf",
						"type": "smf",
						"replicas": 2,
						"resources": {"requests": {"cpu": "500m", "memory": "1Gi"}}
					},
					{
						"name": "upf",
						"type": "upf",
						"replicas": 1,
						"resources": {"requests": {"cpu": "2", "memory": "4Gi"}}
					}
				],
				"deployment_pattern": "production",
				"confidence": 0.92
			}`,
		},
		shouldReturnError: false,
		responseDelay:     10 * time.Millisecond,
		tokenUsage:        100,
	}
}

func (m *MockLLMClient) ProcessIntent(ctx context.Context, prompt string) (string, error) {
	if m.shouldReturnError {
		return "", fmt.Errorf("mock LLM error")
	}

	if m.responseDelay > 0 {
		time.Sleep(m.responseDelay)
	}

	// Return specific response based on prompt content
	if strings.Contains(prompt, "5g") || strings.Contains(prompt, "5G") {
		return m.responses["5g-deployment"], nil
	}
	if strings.Contains(prompt, "low confidence") {
		return m.responses["low-confidence"], nil
	}

	return m.responses["default"], nil
}

func (m *MockLLMClient) SetShouldReturnError(shouldError bool) {
	m.shouldReturnError = shouldError
}

func (m *MockLLMClient) SetResponseDelay(delay time.Duration) {
	m.responseDelay = delay
}

func (m *MockLLMClient) SetResponse(key, response string) {
	m.responses[key] = response
}

type IntentProcessingMockRAGService struct {
	documents         []map[string]interface{}
	shouldReturnError bool
	queryDelay        time.Duration
	maxSimilarity     float64
}

func NewMockRAGService() *IntentProcessingMockRAGService {
	return &IntentProcessingMockRAGService{
		documents: []map[string]interface{}{
			{
				"content":    "AMF (Access and Mobility Management Function) is a key component of 5G Core Network",
				"metadata":   map[string]interface{}{"source": "3GPP TS 23.501", "section": "6.2.2"},
				"similarity": 0.9,
			},
			{
				"content":    "SMF (Session Management Function) handles PDU sessions in 5G networks",
				"metadata":   map[string]interface{}{"source": "3GPP TS 23.502", "section": "4.3.2"},
				"similarity": 0.85,
			},
		},
		shouldReturnError: false,
		queryDelay:        5 * time.Millisecond,
		maxSimilarity:     0.9,
	}
}

func (m *IntentProcessingMockRAGService) ProcessQuery(ctx context.Context, req *rag.RAGRequest) (*rag.RAGResponse, error) {
	if m.shouldReturnError {
		return nil, fmt.Errorf("mock RAG error")
	}

	if m.queryDelay > 0 {
		time.Sleep(m.queryDelay)
	}

	// Filter documents based on query
	var filteredDocs []map[string]interface{}
	avgSimilarity := 0.0

	for _, doc := range m.documents {
		if similarity, ok := doc["similarity"].(float64); ok && similarity >= float64(req.MinConfidence) {
			filteredDocs = append(filteredDocs, doc)
			avgSimilarity += similarity
		}
	}

	if len(filteredDocs) > 0 {
		avgSimilarity /= float64(len(filteredDocs))
	}

	// Limit results
	if req.MaxResults > 0 && len(filteredDocs) > req.MaxResults {
		filteredDocs = filteredDocs[:req.MaxResults]
	}

	// Convert filteredDocs to SearchResult format
	var searchResults []*shared.SearchResult
	for _, doc := range filteredDocs {
		// Create a basic SearchResult for the mock
		searchResult := &shared.SearchResult{
			Document: nil, // Could be nil for mock
			Score:    float32(m.maxSimilarity),
			Distance: 0.0,
			Metadata: doc,
		}
		searchResults = append(searchResults, searchResult)
	}

	return &rag.RAGResponse{
		Answer:          "Mock RAG Answer",
		SourceDocuments: searchResults,
		Confidence:      float32(avgSimilarity),
		ProcessingTime:  m.queryDelay,
		RetrievalTime:   m.queryDelay / 2,
		GenerationTime:  m.queryDelay / 2,
		UsedCache:       false,
		Query:           req.Query,
		IntentType:      req.IntentType,
		Metadata: map[string]interface{}{
			"queryTime": m.queryDelay,
			"totalDocs": len(m.documents),
		},
		ProcessedAt: time.Now(),
	}, nil
}

func (m *IntentProcessingMockRAGService) SetShouldReturnError(shouldError bool) {
	m.shouldReturnError = shouldError
}

func (m *IntentProcessingMockRAGService) SetQueryDelay(delay time.Duration) {
	m.queryDelay = delay
}

func (m *IntentProcessingMockRAGService) SetDocuments(docs []map[string]interface{}) {
	m.documents = docs
}

type MockPromptEngine struct {
	shouldReturnError bool
}

func NewMockPromptEngine() *MockPromptEngine {
	return &MockPromptEngine{
		shouldReturnError: false,
	}
}

func (m *MockPromptEngine) BuildIntentProcessingPrompt(intent string, ragContext map[string]interface{}) (string, error) {
	if m.shouldReturnError {
		return "", fmt.Errorf("mock prompt engine error")
	}

	prompt := fmt.Sprintf(`
Process the following telecommunications intent: %s

Context from knowledge base: %v

Please provide a JSON response with:
- network_functions: array of required network functions
- deployment_pattern: suggested deployment pattern
- confidence: confidence score (0.0 to 1.0)
- resources: resource requirements
`, intent, ragContext)

	return prompt, nil
}

func (m *MockPromptEngine) SetShouldReturnError(shouldError bool) {
	m.shouldReturnError = shouldError
}

type MockStreamingProcessor struct {
	shouldReturnError bool
	chunks            []string
}

func NewMockStreamingProcessor() *MockStreamingProcessor {
	return &MockStreamingProcessor{
		shouldReturnError: false,
		chunks: []string{
			`{"network_functions": [`,
			`{"name": "amf", "type": "amf", "replicas": 2}`,
			`], "confidence": 0.9}`,
		},
	}
}

func (m *MockStreamingProcessor) ProcessIntentStreaming(ctx context.Context, prompt string) (<-chan string, <-chan error) {
	responseChan := make(chan string, len(m.chunks))
	errorChan := make(chan error, 1)

	go func() {
		defer close(responseChan)
		defer close(errorChan)

		if m.shouldReturnError {
			errorChan <- fmt.Errorf("mock streaming error")
			return
		}

		for _, chunk := range m.chunks {
			select {
			case responseChan <- chunk:
				time.Sleep(1 * time.Millisecond) // Simulate streaming delay
			case <-ctx.Done():
				return
			}
		}
	}()

	return responseChan, errorChan
}

func (m *MockStreamingProcessor) SetShouldReturnError(shouldError bool) {
	m.shouldReturnError = shouldError
}

func (m *MockStreamingProcessor) SetChunks(chunks []string) {
	m.chunks = chunks
}

type MockPerformanceOptimizer struct {
	shouldReturnError bool
}

func NewMockPerformanceOptimizer() *MockPerformanceOptimizer {
	return &MockPerformanceOptimizer{
		shouldReturnError: false,
	}
}

func (m *MockPerformanceOptimizer) OptimizePerformance() error {
	if m.shouldReturnError {
		return fmt.Errorf("mock performance optimizer error")
	}
	return nil
}

func (m *MockPerformanceOptimizer) SetShouldReturnError(shouldError bool) {
	m.shouldReturnError = shouldError
}

type MockCircuitBreaker struct {
	isOpen            bool
	shouldReturnError bool
	executionCount    int
}

func NewMockCircuitBreaker() *MockCircuitBreaker {
	return &MockCircuitBreaker{
		isOpen:            false,
		shouldReturnError: false,
		executionCount:    0,
	}
}

func (m *MockCircuitBreaker) Execute(fn func() (interface{}, error)) (interface{}, error) {
	m.executionCount++

	if m.isOpen {
		return nil, fmt.Errorf("circuit breaker is open")
	}

	if m.shouldReturnError {
		return nil, fmt.Errorf("mock circuit breaker error")
	}

	return fn()
}

func (m *MockCircuitBreaker) State() string {
	if m.isOpen {
		return "Open"
	}
	return "Closed"
}

func (m *MockCircuitBreaker) SetOpen(open bool) {
	m.isOpen = open
}

func (m *MockCircuitBreaker) SetShouldReturnError(shouldError bool) {
	m.shouldReturnError = shouldError
}

func (m *MockCircuitBreaker) GetExecutionCount() int {
	return m.executionCount
}

var _ = Describe("SpecializedIntentProcessingController", func() {
	var (
		ctx                      context.Context
		controller               *SpecializedIntentProcessingController
		fakeClient               client.Client
		logger                   logr.Logger
		scheme                   *runtime.Scheme
		fakeRecorder             *record.FakeRecorder
		networkIntent            *nephoranv1.NetworkIntent
		mockLLMClient            *MockLLMClient
		mockRAGService           *MockRAGService
		mockPromptEngine         *MockPromptEngine
		mockStreamingProcessor   *MockStreamingProcessor
		mockPerformanceOptimizer *MockPerformanceOptimizer
		mockCircuitBreaker       *MockCircuitBreaker
		config                   IntentProcessingConfig
	)

	BeforeEach(func() {
		ctx = context.Background()
		logger = zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true))

		// Create scheme and add types
		scheme = runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		Expect(nephoranv1.AddToScheme(scheme)).To(Succeed())

		// Create fake client and recorder
		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			Build()
		fakeRecorder = record.NewFakeRecorder(100)

		// Create mock services
		mockLLMClient = NewMockLLMClient()
		mockRAGService = NewMockRAGService()
		mockPromptEngine = NewMockPromptEngine()
		mockStreamingProcessor = NewMockStreamingProcessor()
		mockPerformanceOptimizer = NewMockPerformanceOptimizer()
		mockCircuitBreaker = NewMockCircuitBreaker()

		// Create configuration
		config = IntentProcessingConfig{
			LLMEndpoint:           "http://mock-llm:8080",
			LLMAPIKey:             "mock-api-key",
			LLMModel:              "gpt-4o-mini",
			MaxTokens:             1000,
			Temperature:           0.7,
			RAGEndpoint:           "http://mock-rag:8080",
			MaxContextChunks:      5,
			SimilarityThreshold:   0.7,
			StreamingEnabled:      false,
			CacheEnabled:          true,
			CacheTTL:              30 * time.Minute,
			MaxRetries:            3,
			Timeout:               30 * time.Second,
			CircuitBreakerEnabled: true,
			FailureThreshold:      5,
			RecoveryTimeout:       60 * time.Second,
		}

		// Create controller with mock dependencies
		controller = &SpecializedIntentProcessingController{
			Client:               fakeClient,
			Scheme:               scheme,
			Recorder:             fakeRecorder,
			Logger:               logger,
			LLMClient:            mockLLMClient,
			RAGService:           mockRAGService,
			PromptEngine:         mockPromptEngine,
			StreamingProcessor:   mockStreamingProcessor,
			PerformanceOptimizer: mockPerformanceOptimizer,
			Config:               config,
			SupportedIntents:     []string{"5g-deployment", "network-slice", "cnf-deployment"},
			ConfidenceThreshold:  0.7,
			metrics:              NewIntentProcessingMetrics(),
			circuitBreaker:       mockCircuitBreaker,
			stopChan:             make(chan struct{}),
			healthStatus: interfaces.HealthStatus{
				Status:      "Healthy",
				Message:     "Controller initialized for testing",
				LastChecked: time.Now(),
			},
		}

		// Initialize cache
		controller.cache = &IntentProcessingCache{
			entries:    make(map[string]*CacheEntry),
			ttl:        config.CacheTTL,
			maxEntries: 1000,
		}

		// Create test NetworkIntent
		networkIntent = &nephoranv1.NetworkIntent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-intent",
				Namespace: "test-namespace",
				UID:       "test-uid-12345",
			},
			Spec: nephoranv1.NetworkIntentSpec{
				Intent:     "Deploy high-availability AMF for 5G core network",
				IntentType: "5g-deployment",
				TargetComponents: []nephoranv1.TargetComponent{
					{
						Type:     "amf",
						Version:  "v1.0.0",
						Replicas: 2,
					},
				},
			},
			Status: nephoranv1.NetworkIntentStatus{
				ProcessingPhase: interfaces.PhaseLLMProcessing,
			},
		}
	})

	AfterEach(func() {
		if controller != nil && controller.started {
			controller.Stop(ctx)
		}
	})

	Describe("Controller Initialization", func() {
		It("should initialize with proper configuration", func() {
			Expect(controller).NotTo(BeNil())
			Expect(controller.LLMClient).NotTo(BeNil())
			Expect(controller.RAGService).NotTo(BeNil())
			Expect(controller.PromptEngine).NotTo(BeNil())
			Expect(controller.StreamingProcessor).NotTo(BeNil())
			Expect(controller.PerformanceOptimizer).NotTo(BeNil())
			Expect(controller.Config.LLMEndpoint).To(Equal("http://mock-llm:8080"))
			Expect(controller.ConfidenceThreshold).To(Equal(0.7))
			Expect(controller.SupportedIntents).To(ContainElement("5g-deployment"))
			Expect(controller.cache).NotTo(BeNil())
		})

		It("should start and stop successfully", func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(controller.started).To(BeTrue())

			err = controller.Stop(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(controller.started).To(BeFalse())
		})

		It("should return supported intent types", func() {
			supportedTypes := controller.GetSupportedIntentTypes()
			Expect(supportedTypes).To(ContainElements("5g-deployment", "network-slice", "cnf-deployment"))
		})
	})

	Describe("Intent Processing", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should process intent successfully with high confidence", func() {
			result, err := controller.ProcessIntent(ctx, networkIntent)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.Success).To(BeTrue())
			Expect(result.NextPhase).To(Equal(interfaces.PhaseResourcePlanning))

			// Verify response data
			Expect(result.Data).To(HaveKey("llmResponse"))
			Expect(result.Data).To(HaveKey("ragContext"))
			Expect(result.Data).To(HaveKey("confidence"))
			Expect(result.Data).To(HaveKey("correlationId"))

			// Verify metrics
			Expect(result.Metrics).To(HaveKey("processing_time_ms"))
			Expect(result.Metrics).To(HaveKey("llm_latency_ms"))
			Expect(result.Metrics).To(HaveKey("rag_latency_ms"))
			Expect(result.Metrics).To(HaveKey("confidence"))

			// Verify confidence is above threshold
			confidence := result.Data["confidence"].(float64)
			Expect(confidence).To(BeNumerically(">=", controller.ConfidenceThreshold))
		})

		It("should process 5G deployment intent with multiple NFs", func() {
			networkIntent.Spec.Intent = "Deploy complete 5G core with AMF, SMF, and UPF"

			result, err := controller.ProcessIntent(ctx, networkIntent)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeTrue())

			// Verify LLM response contains multiple NFs
			llmResponse := result.Data["llmResponse"].(map[string]interface{})
			networkFunctions := llmResponse["network_functions"].([]interface{})
			Expect(len(networkFunctions)).To(BeNumerically(">=", 3))

			// Verify confidence
			confidence := result.Data["confidence"].(float64)
			Expect(confidence).To(BeNumerically(">=", 0.9))
		})

		It("should reject intent with low confidence", func() {
			// Configure mock to return low confidence
			lowConfidenceResponse := `{
				"network_functions": [],
				"deployment_pattern": "unknown", 
				"confidence": 0.3
			}`
			mockLLMClient.SetResponse("default", lowConfidenceResponse)

			result, err := controller.ProcessIntent(ctx, networkIntent)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeFalse())
			Expect(result.ErrorCode).To(Equal("LOW_CONFIDENCE"))
			Expect(result.ErrorMessage).To(ContainSubstring("confidence"))
		})

		It("should handle LLM service failure with circuit breaker", func() {
			mockLLMClient.SetShouldReturnError(true)

			result, err := controller.ProcessIntent(ctx, networkIntent)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeFalse())
			Expect(result.ErrorCode).To(Equal("LLM_PROCESSING_ERROR"))

			// Verify circuit breaker was used
			Expect(mockCircuitBreaker.GetExecutionCount()).To(BeNumerically(">=", 1))
		})

		It("should handle RAG service failure gracefully", func() {
			mockRAGService.SetShouldReturnError(true)

			// Should continue processing without RAG context
			result, err := controller.ProcessIntent(ctx, networkIntent)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeTrue())

			// RAG context should be empty
			ragContext := result.Data["ragContext"].(map[string]interface{})
			Expect(len(ragContext)).To(Equal(0))
		})

		It("should validate intent text properly", func() {
			// Test empty intent
			emptyIntent := &nephoranv1.NetworkIntent{
				ObjectMeta: networkIntent.ObjectMeta,
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "",
				},
			}

			result, err := controller.ProcessIntent(ctx, emptyIntent)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeFalse())
			Expect(result.ErrorCode).To(Equal("VALIDATION_ERROR"))

			// Test very long intent
			longIntent := strings.Repeat("a", 10001)
			longIntentObj := &nephoranv1.NetworkIntent{
				ObjectMeta: networkIntent.ObjectMeta,
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: longIntent,
				},
			}

			result, err = controller.ProcessIntent(ctx, longIntentObj)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeFalse())
			Expect(result.ErrorCode).To(Equal("VALIDATION_ERROR"))

			// Test non-telecom intent
			nonTelecomIntent := &nephoranv1.NetworkIntent{
				ObjectMeta: networkIntent.ObjectMeta,
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Make me a sandwich",
				},
			}

			result, err = controller.ProcessIntent(ctx, nonTelecomIntent)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeFalse())
			Expect(result.ErrorCode).To(Equal("VALIDATION_ERROR"))
		})
	})

	Describe("Streaming Processing", func() {
		BeforeEach(func() {
			controller.Config.StreamingEnabled = true
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should process intent with streaming LLM", func() {
			result, err := controller.ProcessIntent(ctx, networkIntent)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeTrue())

			// Verify streaming was used (indicated by successful processing)
			llmResponse := result.Data["llmResponse"].(map[string]interface{})
			Expect(llmResponse).To(HaveKey("network_functions"))
			Expect(llmResponse).To(HaveKey("confidence"))
		})

		It("should handle streaming errors", func() {
			mockStreamingProcessor.SetShouldReturnError(true)

			result, err := controller.ProcessIntent(ctx, networkIntent)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeFalse())
			Expect(result.ErrorCode).To(Equal("LLM_PROCESSING_ERROR"))
		})

		It("should handle context cancellation during streaming", func() {
			// Create context with short timeout
			streamingCtx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
			defer cancel()

			result, err := controller.ProcessIntent(streamingCtx, networkIntent)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeFalse())
		})
	})

	Describe("Caching", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should cache successful results", func() {
			// First processing - should not use cache
			result1, err := controller.ProcessIntent(ctx, networkIntent)
			Expect(err).NotTo(HaveOccurred())
			Expect(result1.Success).To(BeTrue())
			Expect(result1.Metrics["cache_hit"]).To(Equal(float64(0)))

			// Second processing - should use cache
			result2, err := controller.ProcessIntent(ctx, networkIntent)
			Expect(err).NotTo(HaveOccurred())
			Expect(result2.Success).To(BeTrue())
			Expect(result2.Metrics["cache_hit"]).To(Equal(float64(1)))
		})

		It("should handle cache expiration", func() {
			// Set very short cache TTL
			controller.cache.ttl = 1 * time.Millisecond

			// First processing
			result1, err := controller.ProcessIntent(ctx, networkIntent)
			Expect(err).NotTo(HaveOccurred())
			Expect(result1.Success).To(BeTrue())

			// Wait for cache to expire
			time.Sleep(5 * time.Millisecond)

			// Second processing - should not use cache
			result2, err := controller.ProcessIntent(ctx, networkIntent)
			Expect(err).NotTo(HaveOccurred())
			Expect(result2.Success).To(BeTrue())
			Expect(result2.Metrics["cache_hit"]).To(Equal(float64(0)))
		})

		It("should handle cache size limit", func() {
			// Set small cache size limit
			controller.cache.maxEntries = 1

			// Create first intent
			result1, err := controller.ProcessIntent(ctx, networkIntent)
			Expect(err).NotTo(HaveOccurred())
			Expect(result1.Success).To(BeTrue())

			// Create second intent with different text
			networkIntent2 := networkIntent.DeepCopy()
			networkIntent2.Name = "test-intent-2"
			networkIntent2.Spec.Intent = "Different intent for cache testing"

			result2, err := controller.ProcessIntent(ctx, networkIntent2)
			Expect(err).NotTo(HaveOccurred())
			Expect(result2.Success).To(BeTrue())

			// Cache should be limited to 1 entry
			Expect(len(controller.cache.entries)).To(Equal(1))
		})
	})

	Describe("RAG Enhancement", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should enhance intent with RAG context", func() {
			ragContext, err := controller.EnhanceWithRAG(ctx, networkIntent.Spec.Intent)
			Expect(err).NotTo(HaveOccurred())
			Expect(ragContext).NotTo(BeNil())
			Expect(ragContext).To(HaveKey("relevant_documents"))
			Expect(ragContext).To(HaveKey("chunk_count"))
			Expect(ragContext).To(HaveKey("max_similarity"))

			// Verify documents were retrieved
			documents := ragContext["relevant_documents"].([]map[string]interface{})
			Expect(len(documents)).To(BeNumerically(">=", 1))
		})

		It("should handle empty RAG service gracefully", func() {
			// Create controller without RAG service
			controller.RAGService = nil

			ragContext, err := controller.EnhanceWithRAG(ctx, networkIntent.Spec.Intent)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(ragContext)).To(Equal(0))
		})

		It("should filter documents by similarity threshold", func() {
			// Set high similarity threshold
			controller.Config.SimilarityThreshold = 0.95

			ragContext, err := controller.EnhanceWithRAG(ctx, networkIntent.Spec.Intent)
			Expect(err).NotTo(HaveOccurred())

			documents := ragContext["relevant_documents"].([]map[string]interface{})
			// Should have fewer documents due to high threshold
			Expect(len(documents)).To(BeNumerically("<=", 1))
		})
	})

	Describe("Phase Controller Interface", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should process LLM processing phase", func() {
			result, err := controller.ProcessPhase(ctx, networkIntent, interfaces.PhaseLLMProcessing)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeTrue())
			Expect(result.NextPhase).To(Equal(interfaces.PhaseResourcePlanning))
		})

		It("should reject unsupported phases", func() {
			result, err := controller.ProcessPhase(ctx, networkIntent, interfaces.PhaseResourcePlanning)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeFalse())
			Expect(result.ErrorMessage).To(ContainSubstring("unsupported phase"))
		})

		It("should return correct dependencies", func() {
			deps := controller.GetDependencies()
			Expect(deps).To(ContainElement(interfaces.PhaseIntentReceived))
		})

		It("should return correct blocked phases", func() {
			blocked := controller.GetBlockedPhases()
			expectedPhases := []interfaces.ProcessingPhase{
				interfaces.PhaseResourcePlanning,
				interfaces.PhaseManifestGeneration,
				interfaces.PhaseGitOpsCommit,
				interfaces.PhaseDeploymentVerification,
			}
			Expect(blocked).To(ContainElements(expectedPhases))
		})
	})

	Describe("Health and Metrics", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return health status", func() {
			health, err := controller.GetHealthStatus(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(health.Status).To(Equal("Healthy"))
			Expect(health.Metrics).To(HaveKey("totalProcessed"))
			Expect(health.Metrics).To(HaveKey("successRate"))
		})

		It("should return controller metrics", func() {
			metrics, err := controller.GetMetrics(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(metrics).To(HaveKey("total_processed"))
			Expect(metrics).To(HaveKey("successful_processed"))
			Expect(metrics).To(HaveKey("failed_processed"))
			Expect(metrics).To(HaveKey("success_rate"))
			Expect(metrics).To(HaveKey("average_latency_ms"))
		})

		It("should update metrics after processing", func() {
			initialMetrics, err := controller.GetMetrics(ctx)
			Expect(err).NotTo(HaveOccurred())
			initialTotal := initialMetrics["total_processed"]

			// Process an intent
			_, err = controller.ProcessIntent(ctx, networkIntent)
			Expect(err).NotTo(HaveOccurred())

			updatedMetrics, err := controller.GetMetrics(ctx)
			Expect(err).NotTo(HaveOccurred())
			updatedTotal := updatedMetrics["total_processed"]

			Expect(updatedTotal).To(Equal(initialTotal + 1))
		})

		It("should track session metrics", func() {
			intentID := networkIntent.Name

			// Process intent to create session
			_, err := controller.ProcessIntent(ctx, networkIntent)
			Expect(err).NotTo(HaveOccurred())

			// Get phase status (session should be cleaned up, but we can verify the pattern)
			status, err := controller.GetPhaseStatus(ctx, intentID)
			Expect(err).NotTo(HaveOccurred())
			Expect(status.Phase).To(Equal(interfaces.PhaseLLMProcessing))
		})
	})

	Describe("Error Handling and Recovery", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle prompt engine failure", func() {
			mockPromptEngine.SetShouldReturnError(true)

			result, err := controller.ProcessIntent(ctx, networkIntent)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeFalse())
			Expect(result.ErrorCode).To(Equal("LLM_PROCESSING_ERROR"))
		})

		It("should handle malformed LLM response", func() {
			mockLLMClient.SetResponse("default", `{invalid json`)

			result, err := controller.ProcessIntent(ctx, networkIntent)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeFalse())
			Expect(result.ErrorCode).To(Equal("LLM_PROCESSING_ERROR"))
		})

		It("should handle circuit breaker open state", func() {
			mockCircuitBreaker.SetOpen(true)

			result, err := controller.ProcessIntent(ctx, networkIntent)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeFalse())
			Expect(result.ErrorCode).To(Equal("LLM_PROCESSING_ERROR"))
		})

		It("should handle phase error with retry logic", func() {
			intentID := networkIntent.Name
			testError := fmt.Errorf("test processing error")

			err := controller.HandlePhaseError(ctx, intentID, testError)
			Expect(err).NotTo(HaveOccurred()) // Should return nil to indicate retry
		})

		It("should handle phase error with max retries exceeded", func() {
			intentID := networkIntent.Name
			testError := fmt.Errorf("test processing error")

			// Create a session with max retries already reached
			session := &ProcessingSession{
				IntentID:   intentID,
				RetryCount: controller.Config.MaxRetries,
			}
			controller.activeProcessing.Store(intentID, session)

			err := controller.HandlePhaseError(ctx, intentID, testError)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("test processing error"))
		})
	})

	Describe("Concurrent Processing", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle multiple concurrent intents", func() {
			numIntents := 10
			results := make(chan *interfaces.ProcessingResult, numIntents)
			errors := make(chan error, numIntents)

			// Process multiple intents concurrently
			for i := 0; i < numIntents; i++ {
				go func(index int) {
					defer GinkgoRecover()
					intent := networkIntent.DeepCopy()
					intent.Name = fmt.Sprintf("concurrent-intent-%d", index)
					intent.UID = fmt.Sprintf("uid-%d", index)

					result, err := controller.ProcessIntent(ctx, intent)
					if err != nil {
						errors <- err
					} else {
						results <- result
					}
				}(i)
			}

			// Collect results
			successCount := 0
			errorCount := 0
			timeout := time.After(30 * time.Second)

			for i := 0; i < numIntents; i++ {
				select {
				case result := <-results:
					if result.Success {
						successCount++
					}
				case <-errors:
					errorCount++
				case <-timeout:
					Fail("Timeout waiting for concurrent processing to complete")
				}
			}

			// Verify all intents were processed
			Expect(successCount + errorCount).To(Equal(numIntents))
			// Most should succeed (allow for some failures due to test conditions)
			Expect(successCount).To(BeNumerically(">=", numIntents/2))
		})

		It("should maintain thread safety under concurrent load", func() {
			numGoroutines := 50
			done := make(chan bool, numGoroutines)

			// Launch goroutines that access controller state
			for i := 0; i < numGoroutines; i++ {
				go func(index int) {
					defer GinkgoRecover()
					defer func() { done <- true }()

					// Mix of operations
					switch index % 4 {
					case 0:
						// Process intent
						intent := networkIntent.DeepCopy()
						intent.Name = fmt.Sprintf("thread-safety-intent-%d", index)
						controller.ProcessIntent(ctx, intent)

					case 1:
						// Get health status
						controller.GetHealthStatus(ctx)

					case 2:
						// Get metrics
						controller.GetMetrics(ctx)

					case 3:
						// Get supported intent types
						controller.GetSupportedIntentTypes()
					}
				}(i)
			}

			// Wait for all goroutines to complete
			for i := 0; i < numGoroutines; i++ {
				Eventually(done).Should(Receive())
			}
		})
	})

	Describe("Background Operations", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should perform background cleanup", func() {
			intentID := "cleanup-test-intent"

			// Create an old session
			oldSession := &ProcessingSession{
				IntentID:  intentID,
				StartTime: time.Now().Add(-2 * time.Hour),
			}
			controller.activeProcessing.Store(intentID, oldSession)

			// Verify session exists
			_, exists := controller.activeProcessing.Load(intentID)
			Expect(exists).To(BeTrue())

			// Trigger cleanup
			controller.cleanupExpiredSessions()

			// Verify session was removed
			_, exists = controller.activeProcessing.Load(intentID)
			Expect(exists).To(BeFalse())
		})

		It("should perform cache cleanup", func() {
			// Add expired entry to cache
			expiredHash := "expired-entry"
			controller.cache.entries[expiredHash] = &CacheEntry{
				Timestamp: time.Now().Add(-1 * time.Hour),
			}

			// Verify entry exists
			Expect(controller.cache.entries).To(HaveKey(expiredHash))

			// Trigger cache cleanup
			controller.cleanupExpiredCache()

			// Verify expired entry was removed
			Expect(controller.cache.entries).NotTo(HaveKey(expiredHash))
		})

		It("should perform health monitoring", func() {
			initialHealth := controller.healthStatus

			// Trigger health check
			controller.performHealthCheck()

			// Verify health status was updated
			Expect(controller.healthStatus.LastChecked).To(BeTemporally(">", initialHealth.LastChecked))
		})
	})
})

func TestSpecializedIntentProcessingController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SpecializedIntentProcessingController Suite")
}
