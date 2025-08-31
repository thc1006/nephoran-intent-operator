//go:build ignore

package rag

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/thc1006/nephoran-intent-operator/pkg/types"
)

// MockWeaviateClientRST implements a mock for WeaviateClient (renamed to avoid conflict)
type MockWeaviateClientRST struct {
	searchResponse *SearchResponse
	searchError    error
	healthStatus   *HealthStatus
}

func (m *MockWeaviateClientRST) Search(ctx context.Context, query *SearchQuery) (*SearchResponse, error) {
	if m.searchError != nil {
		return nil, m.searchError
	}
	return m.searchResponse, nil
}

func (m *MockWeaviateClientRST) GetHealthStatus() *HealthStatus {
	if m.healthStatus != nil {
		return m.healthStatus
	}
	return &HealthStatus{
		IsHealthy: true,
		LastCheck: time.Now(),
		Version:   "1.0.0",
	}
}

// MockLLMClient implements a mock for types.ClientInterface
type MockLLMClient struct {
	processResponse string
	processError    error
}

func (m *MockLLMClient) ProcessIntent(ctx context.Context, prompt string) (string, error) {
	if m.processError != nil {
		return "", m.processError
	}
	return m.processResponse, nil
}

var _ = Describe("RAGService", func() {
	var (
		ragService   *RAGService
		mockWeaviate *MockWeaviateClientRST
		mockLLM      *MockLLMClient
		config       *RAGConfig
		ctx          context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Create mock clients
		mockWeaviate = &MockWeaviateClientRST{}
		mockLLM = &MockLLMClient{}

		// Create test configuration
		config = &RAGConfig{
			DefaultSearchLimit:   10,
			MaxSearchLimit:       50,
			DefaultHybridAlpha:   0.7,
			MinConfidenceScore:   0.5,
			MaxContextLength:     8000,
			ContextOverlapTokens: 200,
			MaxLLMTokens:         4000,
			Temperature:          0.3,
			EnableCaching:        true,
			CacheTTL:             30 * time.Minute,
			EnableReranking:      true,
			RerankingTopK:        20,
			EnableQueryExpansion: true,
			TelecomDomains:       []string{"RAN", "Core", "Transport"},
			PreferredSources:     []string{"3GPP", "O-RAN"},
			TechnologyFilter:     []string{"5G", "4G"},
		}

		// Create RAG service
		ragService = NewRAGService(mockWeaviate, mockLLM, config)
	})

	Describe("NewRAGService", func() {
		Context("when creating a new RAG service", func() {
			It("should create service with provided configuration", func() {
				service := NewRAGService(mockWeaviate, mockLLM, config)
				Expect(service).NotTo(BeNil())
				Expect(service.config).To(Equal(config))
				Expect(service.weaviateClient).To(Equal(mockWeaviate))
				Expect(service.llmClient).To(Equal(mockLLM))
			})

			It("should create service with default configuration when nil is provided", func() {
				service := NewRAGService(mockWeaviate, mockLLM, nil)
				Expect(service).NotTo(BeNil())
				Expect(service.config).NotTo(BeNil())
				Expect(service.config.DefaultSearchLimit).To(Equal(10))
				Expect(service.config.MaxSearchLimit).To(Equal(50))
			})

			It("should initialize metrics", func() {
				service := NewRAGService(mockWeaviate, mockLLM, config)
				metrics := service.GetMetrics()
				Expect(metrics).NotTo(BeNil())
				Expect(metrics.TotalQueries).To(Equal(int64(0)))
				Expect(metrics.SuccessfulQueries).To(Equal(int64(0)))
				Expect(metrics.FailedQueries).To(Equal(int64(0)))
			})
		})
	})

	Describe("ProcessQuery", func() {
		var request *RAGRequest

		BeforeEach(func() {
			request = &RAGRequest{
				Query:             "How to configure 5G RAN parameters?",
				IntentType:        "configuration",
				MaxResults:        5,
				MinConfidence:     0.6,
				UseHybridSearch:   true,
				EnableReranking:   true,
				IncludeSourceRefs: true,
				ResponseFormat:    "markdown",
				UserID:            "test-user",
				SessionID:         "test-session",
			}

			// Setup successful mock responses
			mockWeaviate.searchResponse = &SearchResponse{
				Results: []*SearchResult{
					{
						Document: &TelecomDocument{
							ID:       "doc1",
							Content:  "5G RAN configuration parameters include...",
							Source:   "3GPP TS 38.331",
							Title:    "RAN Configuration Guide",
							Category: "Configuration",
						},
						Score: 0.9,
					},
					{
						Document: &TelecomDocument{
							ID:       "doc2",
							Content:  "Additional RAN parameters for network optimization...",
							Source:   "O-RAN WG3",
							Title:    "Network Optimization",
							Category: "Optimization",
						},
						Score: 0.8,
					},
				},
				Took: 50 * time.Millisecond,
			}

			mockLLM.processResponse = "Based on the provided documentation, here are the key 5G RAN configuration parameters..."
		})

		Context("when processing a valid query", func() {
			It("should return a successful response", func() {
				response, err := ragService.ProcessQuery(ctx, request)

				Expect(err).ToNot(HaveOccurred())
				Expect(response).NotTo(BeNil())
				Expect(response.Answer).To(ContainSubstring("Based on the provided documentation"))
				Expect(response.Query).To(Equal(request.Query))
				Expect(response.IntentType).To(Equal(request.IntentType))
				Expect(len(response.SourceDocuments)).To(Equal(2))
				Expect(response.Confidence).To(BeNumerically(">", 0))
				Expect(response.ProcessingTime).To(BeNumerically(">", 0))
			})

			It("should include source references when requested", func() {
				request.IncludeSourceRefs = true
				response, err := ragService.ProcessQuery(ctx, request)

				Expect(err).ToNot(HaveOccurred())
				Expect(response.Answer).To(ContainSubstring("**Sources:**"))
				Expect(response.Answer).To(ContainSubstring("[1] 3GPP TS 38.331"))
			})

			It("should update metrics on successful query", func() {
				initialMetrics := ragService.GetMetrics()
				initialTotal := initialMetrics.TotalQueries
				initialSuccess := initialMetrics.SuccessfulQueries

				_, err := ragService.ProcessQuery(ctx, request)
				Expect(err).ToNot(HaveOccurred())

				finalMetrics := ragService.GetMetrics()
				Expect(finalMetrics.TotalQueries).To(Equal(initialTotal + 1))
				Expect(finalMetrics.SuccessfulQueries).To(Equal(initialSuccess + 1))
			})

			It("should apply default limits when not specified", func() {
				request.MaxResults = 0
				request.MinConfidence = 0

				response, err := ragService.ProcessQuery(ctx, request)
				Expect(err).ToNot(HaveOccurred())
				Expect(response).NotTo(BeNil())
			})

			It("should enforce maximum search limit", func() {
				request.MaxResults = 100 // Exceeds configured max of 50

				response, err := ragService.ProcessQuery(ctx, request)
				Expect(err).ToNot(HaveOccurred())
				Expect(response).NotTo(BeNil())
			})
		})

		Context("when handling invalid requests", func() {
			It("should return error for nil request", func() {
				response, err := ragService.ProcessQuery(ctx, nil)

				Expect(err).To(HaveOccurred())
				Expect(response).To(BeNil())
				Expect(err.Error()).To(ContainSubstring("request"))
			})

			It("should return error for empty query", func() {
				request.Query = ""
				response, err := ragService.ProcessQuery(ctx, request)

				Expect(err).To(HaveOccurred())
				Expect(response).To(BeNil())
				Expect(err.Error()).To(ContainSubstring("query"))
			})
		})

		Context("when search service fails", func() {
			BeforeEach(func() {
				mockWeaviate.searchError = errors.New("search service unavailable")
			})

			It("should return error and update failure metrics", func() {
				initialMetrics := ragService.GetMetrics()
				initialFailed := initialMetrics.FailedQueries

				response, err := ragService.ProcessQuery(ctx, request)

				Expect(err).To(HaveOccurred())
				Expect(response).To(BeNil())
				Expect(err.Error()).To(ContainSubstring("weaviate"))

				finalMetrics := ragService.GetMetrics()
				Expect(finalMetrics.FailedQueries).To(Equal(initialFailed + 1))
			})
		})

		Context("when LLM service fails", func() {
			BeforeEach(func() {
				mockLLM.processError = errors.New("LLM service unavailable")
			})

			It("should return error and update failure metrics", func() {
				initialMetrics := ragService.GetMetrics()
				initialFailed := initialMetrics.FailedQueries

				response, err := ragService.ProcessQuery(ctx, request)

				Expect(err).To(HaveOccurred())
				Expect(response).To(BeNil())
				Expect(err.Error()).To(ContainSubstring("llm"))

				finalMetrics := ragService.GetMetrics()
				Expect(finalMetrics.FailedQueries).To(Equal(initialFailed + 1))
			})
		})

		Context("when context is cancelled", func() {
			It("should handle context cancellation gracefully", func() {
				cancelCtx, cancel := context.WithCancel(ctx)
				cancel() // Cancel immediately

				response, err := ragService.ProcessQuery(cancelCtx, request)

				Expect(err).To(HaveOccurred())
				Expect(response).To(BeNil())
				Expect(err.Error()).To(ContainSubstring("context"))
			})

			It("should handle timeout", func() {
				timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
				defer cancel()

				// Add delay to mock to trigger timeout
				time.Sleep(2 * time.Millisecond)

				response, err := ragService.ProcessQuery(timeoutCtx, request)

				Expect(err).To(HaveOccurred())
				Expect(response).To(BeNil())
			})
		})
	})

	Describe("buildSearchFilters", func() {
		It("should build filters from request parameters", func() {
			request := &RAGRequest{
				IntentType: "configuration",
				SearchFilters: map[string]interface{}{
					"source":  "3GPP",
					"version": "R16",
				},
			}

			filters := ragService.buildSearchFilters(request)

			Expect(filters).To(HaveKeyWithValue("source", "3GPP"))
			Expect(filters).To(HaveKeyWithValue("version", "R16"))
			Expect(filters).To(HaveKeyWithValue("category", "Configuration"))
		})

		It("should handle different intent types", func() {
			testCases := []struct {
				intentType       string
				expectedCategory string
			}{
				{"configuration", "Configuration"},
				{"optimization", "Optimization"},
				{"troubleshooting", "Troubleshooting"},
				{"monitoring", "Monitoring"},
			}

			for _, tc := range testCases {
				request := &RAGRequest{IntentType: tc.intentType}
				filters := ragService.buildSearchFilters(request)
				Expect(filters).To(HaveKeyWithValue("category", tc.expectedCategory))
			}
		})

		It("should handle empty intent type", func() {
			request := &RAGRequest{IntentType: ""}
			filters := ragService.buildSearchFilters(request)
			Expect(filters).NotTo(HaveKey("category"))
		})
	})

	Describe("prepareContext", func() {
		var searchResults []*types.SearchResult

		BeforeEach(func() {
			searchResults = []*types.SearchResult{
				{
					Document: &types.TelecomDocument{
						ID:              "doc1",
						Content:         "First document content with technical details about 5G configuration.",
						Source:          "3GPP TS 38.331",
						Title:           "5G Configuration Guide",
						Category:        "Configuration",
						Version:         "R16",
						Technology:      []string{"5G", "NR"},
						NetworkFunction: []string{"gNB", "AMF"},
					},
					Score: 0.9,
				},
				{
					Document: &types.TelecomDocument{
						ID:      "doc2",
						Content: "Second document with additional information about network optimization parameters.",
						Source:  "O-RAN WG3",
						Title:   "Network Optimization",
					},
					Score: 0.8,
				},
			}
		})

		It("should prepare context from search results", func() {
			request := &RAGRequest{Query: "test query"}
			context, metadata := ragService.prepareContext(searchResults, request)

			Expect(context).To(ContainSubstring("Document 1:"))
			Expect(context).To(ContainSubstring("Document 2:"))
			Expect(context).To(ContainSubstring("Title: 5G Configuration Guide"))
			Expect(context).To(ContainSubstring("Source: 3GPP TS 38.331"))
			Expect(context).To(ContainSubstring("First document content"))
			Expect(context).To(ContainSubstring("Second document with additional"))

			Expect(metadata).To(HaveKeyWithValue("documents_considered", 2))
			Expect(metadata).To(HaveKeyWithValue("documents_used", 2))
			Expect(metadata).To(HaveKeyWithValue("context_truncated", false))
		})

		It("should include user context when provided", func() {
			request := &RAGRequest{
				Query:   "test query",
				Context: "Additional user context information",
			}
			context, _ := ragService.prepareContext(searchResults, request)

			Expect(context).To(ContainSubstring("Additional user context information"))
		})

		It("should handle context length limits", func() {
			// Create a very long document that would exceed context limit
			longContent := make([]byte, 50000) // Very long content
			for i := range longContent {
				longContent[i] = 'a'
			}

			longResults := []*types.SearchResult{
				{
					Document: &types.TelecomDocument{
						ID:      "long-doc",
						Content: string(longContent),
						Source:  "Long Document",
					},
					Score: 0.9,
				},
			}

			request := &RAGRequest{Query: "test query"}
			_, metadata := ragService.prepareContext(longResults, request)

			Expect(metadata).To(HaveKeyWithValue("context_truncated", true))
		})

		It("should handle empty search results", func() {
			request := &RAGRequest{Query: "test query"}
			context, metadata := ragService.prepareContext([]*types.SearchResult{}, request)

			Expect(context).To(BeEmpty())
			Expect(metadata).To(HaveKeyWithValue("documents_considered", 0))
			Expect(metadata).To(HaveKeyWithValue("documents_used", 0))
		})

		It("should handle nil documents gracefully", func() {
			resultsWithNil := []*types.SearchResult{
				{Document: nil, Score: 0.9},
				searchResults[0],
			}

			request := &RAGRequest{Query: "test query"}
			context, metadata := ragService.prepareContext(resultsWithNil, request)

			Expect(context).To(ContainSubstring("Document 1:"))
			Expect(metadata).To(HaveKeyWithValue("documents_considered", 2))
			Expect(metadata).To(HaveKeyWithValue("documents_used", 1))
		})
	})

	Describe("buildLLMPrompt", func() {
		It("should build a comprehensive prompt", func() {
			query := "How to configure 5G parameters?"
			context := "Document 1: Configuration details..."
			intentType := "configuration"

			prompt := ragService.buildLLMPrompt(query, context, intentType)

			Expect(prompt).To(ContainSubstring("expert telecommunications engineer"))
			Expect(prompt).To(ContainSubstring("Guidelines:"))
			Expect(prompt).To(ContainSubstring("Focus on configuration parameters"))
			Expect(prompt).To(ContainSubstring("Relevant Technical Documentation:"))
			Expect(prompt).To(ContainSubstring("Document 1: Configuration details..."))
			Expect(prompt).To(ContainSubstring("User Question:"))
			Expect(prompt).To(ContainSubstring("How to configure 5G parameters?"))
		})

		It("should include intent-specific instructions", func() {
			testCases := []struct {
				intentType      string
				expectedContent string
			}{
				{"configuration", "Focus on configuration parameters"},
				{"optimization", "Focus on performance optimization"},
				{"troubleshooting", "Focus on diagnostic procedures"},
				{"monitoring", "Focus on monitoring approaches"},
			}

			for _, tc := range testCases {
				prompt := ragService.buildLLMPrompt("test query", "test context", tc.intentType)
				Expect(prompt).To(ContainSubstring(tc.expectedContent))
			}
		})

		It("should handle empty context", func() {
			prompt := ragService.buildLLMPrompt("test query", "", "configuration")
			Expect(prompt).NotTo(ContainSubstring("Relevant Technical Documentation:"))
		})

		It("should handle empty intent type", func() {
			prompt := ragService.buildLLMPrompt("test query", "test context", "")
			Expect(prompt).NotTo(ContainSubstring("Focus on"))
		})
	})

	Describe("calculateConfidence", func() {
		It("should calculate confidence based on search results", func() {
			results := []*types.SearchResult{
				{Score: 0.9},
				{Score: 0.8},
				{Score: 0.7},
			}

			confidence := ragService.calculateConfidence(results)
			Expect(confidence).To(BeNumerically(">", 0))
			Expect(confidence).To(BeNumerically("<=", 1))
		})

		It("should return 0 for empty results", func() {
			confidence := ragService.calculateConfidence([]*types.SearchResult{})
			Expect(confidence).To(Equal(float32(0.0)))
		})

		It("should reduce confidence for few results", func() {
			singleResult := []*types.SearchResult{{Score: 0.9}}
			multipleResults := []*types.SearchResult{
				{Score: 0.9}, {Score: 0.8}, {Score: 0.7},
			}

			singleConfidence := ragService.calculateConfidence(singleResult)
			multipleConfidence := ragService.calculateConfidence(multipleResults)

			Expect(singleConfidence).To(BeNumerically("<", multipleConfidence))
		})

		It("should normalize high scores", func() {
			results := []*types.SearchResult{
				{Score: 10.0}, // Abnormally high score
			}

			confidence := ragService.calculateConfidence(results)
			Expect(confidence).To(BeNumerically("<=", 1.0))
		})
	})

	Describe("GetMetrics", func() {
		It("should return current metrics", func() {
			metrics := ragService.GetMetrics()

			Expect(metrics).NotTo(BeNil())
			Expect(metrics.TotalQueries).To(Equal(int64(0)))
			Expect(metrics.SuccessfulQueries).To(Equal(int64(0)))
			Expect(metrics.FailedQueries).To(Equal(int64(0)))
			Expect(metrics.AverageLatency).To(Equal(time.Duration(0)))
			Expect(metrics.LastUpdated).NotTo(BeZero())
		})

		It("should return a copy of metrics", func() {
			metrics1 := ragService.GetMetrics()
			metrics2 := ragService.GetMetrics()

			// Modify one copy
			metrics1.TotalQueries = 100

			// Other copy should be unaffected
			Expect(metrics2.TotalQueries).To(Equal(int64(0)))
		})
	})

	Describe("GetHealth", func() {
		It("should return health status", func() {
			health := ragService.GetHealth()

			Expect(health).To(HaveKey("status"))
			Expect(health).To(HaveKey("weaviate"))
			Expect(health).To(HaveKey("metrics"))

			weaviateHealth := health["weaviate"].(map[string]interface{})
			Expect(weaviateHealth).To(HaveKeyWithValue("healthy", true))
			Expect(weaviateHealth).To(HaveKey("version"))
		})

		It("should include weaviate health status", func() {
			mockWeaviate.healthStatus = &HealthStatus{
				IsHealthy: false,
				LastCheck: time.Now(),
				Version:   "1.2.3",
			}

			health := ragService.GetHealth()
			weaviateHealth := health["weaviate"].(map[string]interface{})
			Expect(weaviateHealth).To(HaveKeyWithValue("healthy", false))
			Expect(weaviateHealth).To(HaveKeyWithValue("version", "1.2.3"))
		})
	})

	Describe("enhanceResponse", func() {
		var sourceDocuments []*types.SearchResult

		BeforeEach(func() {
			sourceDocuments = []*types.SearchResult{
				{
					Document: &types.TelecomDocument{
						Source:  "3GPP TS 38.331",
						Title:   "5G Configuration",
						Version: "R16",
					},
					Score: 0.9,
				},
				{
					Document: &types.TelecomDocument{
						Source: "O-RAN WG3",
						Title:  "Network Optimization",
					},
					Score: 0.8,
				},
			}
		})

		It("should enhance response with source references when requested", func() {
			request := &RAGRequest{IncludeSourceRefs: true}
			llmResponse := "This is the LLM response."

			enhanced := ragService.enhanceResponse(llmResponse, sourceDocuments, request)

			Expect(enhanced).To(ContainSubstring("This is the LLM response."))
			Expect(enhanced).To(ContainSubstring("**Sources:**"))
			Expect(enhanced).To(ContainSubstring("[1] 3GPP TS 38.331 - 5G Configuration (R16)"))
			Expect(enhanced).To(ContainSubstring("[2] O-RAN WG3 - Network Optimization"))
		})

		It("should not enhance response when source references not requested", func() {
			request := &RAGRequest{IncludeSourceRefs: false}
			llmResponse := "This is the LLM response."

			enhanced := ragService.enhanceResponse(llmResponse, sourceDocuments, request)

			Expect(enhanced).To(Equal("This is the LLM response."))
			Expect(enhanced).NotTo(ContainSubstring("**Sources:**"))
		})

		It("should limit references to top 5", func() {
			// Create more than 5 source documents
			manyDocs := make([]*types.SearchResult, 10)
			for i := 0; i < 10; i++ {
				manyDocs[i] = &types.SearchResult{
					Document: &types.TelecomDocument{
						Source: "Source " + string(rune('A'+i)),
						Title:  "Title " + string(rune('A'+i)),
					},
					Score: 0.9 - float32(i)*0.1,
				}
			}

			request := &RAGRequest{IncludeSourceRefs: true}
			llmResponse := "Response"

			enhanced := ragService.enhanceResponse(llmResponse, manyDocs, request)

			// Should only have references [1] through [5]
			Expect(enhanced).To(ContainSubstring("[1]"))
			Expect(enhanced).To(ContainSubstring("[5]"))
			Expect(enhanced).NotTo(ContainSubstring("[6]"))
		})

		It("should handle empty source documents", func() {
			request := &RAGRequest{IncludeSourceRefs: true}
			llmResponse := "Response"

			enhanced := ragService.enhanceResponse(llmResponse, []*types.SearchResult{}, request)

			Expect(enhanced).To(Equal("Response"))
		})

		It("should handle nil documents in source list", func() {
			docsWithNil := []*types.SearchResult{
				{Document: nil, Score: 0.9},
				sourceDocuments[0],
			}

			request := &RAGRequest{IncludeSourceRefs: true}
			llmResponse := "Response"

			enhanced := ragService.enhanceResponse(llmResponse, docsWithNil, request)

			Expect(enhanced).To(ContainSubstring("**Sources:**"))
			Expect(enhanced).To(ContainSubstring("[1] 3GPP TS 38.331"))
		})
	})

	Describe("Edge Cases and Error Handling", func() {
		Context("when dealing with malformed data", func() {
			It("should handle malformed search results gracefully", func() {
				mockWeaviate.searchResponse = &SearchResponse{
					Results: []*SearchResult{
						{Document: nil, Score: 0.9}, // Nil document
						{
							Document: &TelecomDocument{
								ID:      "", // Empty ID
								Content: "", // Empty content
								Source:  "", // Empty source
							},
							Score: 0.0, // Zero score
						},
					},
					Took: 0,
				}

				request := &RAGRequest{
					Query:      "test query",
					MaxResults: 5,
				}

				response, err := ragService.ProcessQuery(ctx, request)
				Expect(err).ToNot(HaveOccurred())
				Expect(response).NotTo(BeNil())
			})

			It("should handle extremely long queries", func() {
				longQuery := make([]byte, 100000)
				for i := range longQuery {
					longQuery[i] = 'a'
				}

				request := &RAGRequest{
					Query:      string(longQuery),
					MaxResults: 5,
				}

				response, err := ragService.ProcessQuery(ctx, request)
				Expect(err).ToNot(HaveOccurred())
				Expect(response).NotTo(BeNil())
			})
		})

		Context("when dealing with concurrent requests", func() {
			It("should handle concurrent queries safely", func() {
				request := &RAGRequest{
					Query:      "concurrent test query",
					MaxResults: 5,
				}

				// Run multiple queries concurrently
				done := make(chan bool, 10)
				for i := 0; i < 10; i++ {
					go func() {
						defer GinkgoRecover()
						response, err := ragService.ProcessQuery(ctx, request)
						Expect(err).ToNot(HaveOccurred())
						Expect(response).NotTo(BeNil())
						done <- true
					}()
				}

				// Wait for all to complete
				for i := 0; i < 10; i++ {
					Eventually(done).Should(Receive())
				}

				// Check metrics were updated correctly
				metrics := ragService.GetMetrics()
				Expect(metrics.TotalQueries).To(Equal(int64(10)))
				Expect(metrics.SuccessfulQueries).To(Equal(int64(10)))
			})
		})
	})

	Describe("Default Configuration", func() {
		It("should have proper default values", func() {
			defaultConfig := getDefaultRAGConfig()

			Expect(defaultConfig.DefaultSearchLimit).To(Equal(10))
			Expect(defaultConfig.MaxSearchLimit).To(Equal(50))
			Expect(defaultConfig.DefaultHybridAlpha).To(Equal(float32(0.7)))
			Expect(defaultConfig.MinConfidenceScore).To(Equal(float32(0.5)))
			Expect(defaultConfig.MaxContextLength).To(Equal(8000))
			Expect(defaultConfig.EnableCaching).To(BeTrue())
			Expect(defaultConfig.EnableReranking).To(BeTrue())
			Expect(defaultConfig.EnableQueryExpansion).To(BeTrue())
			Expect(len(defaultConfig.TelecomDomains)).To(BeNumerically(">", 0))
			Expect(len(defaultConfig.PreferredSources)).To(BeNumerically(">", 0))
			Expect(len(defaultConfig.TechnologyFilter)).To(BeNumerically(">", 0))
		})
	})
})
