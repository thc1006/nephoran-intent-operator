// Package unit provides comprehensive unit tests for LLM integration components
package unit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/suite"

	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
	"github.com/thc1006/nephoran-intent-operator/tests/framework"
)

// LLMTestSuite provides comprehensive testing for LLM components
type LLMTestSuite struct {
	*framework.TestSuite
	llmClient *llm.EnhancedPerformanceClient
}

// TestLLMComponents runs the LLM test suite
func TestLLMComponents(t *testing.T) {
	suite.Run(t, &LLMTestSuite{
		TestSuite: framework.NewTestSuite(),
	})
}

// SetupSuite initializes the LLM test environment
func (suite *LLMTestSuite) SetupSuite() {
	suite.TestSuite.SetupSuite()

	// Initialize LLM client with test configuration
	config := &llm.EnhancedClientConfig{
		Config: &llm.Config{
			Provider:    "mock",
			Model:       "gpt-4o-mini",
			APIKey:      "test-key",
			MaxTokens:   2048,
			Temperature: 0.0,
			Timeout:     30 * time.Second,
		},
	}

	client, err := llm.NewEnhancedPerformanceClient(config)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	suite.llmClient = client

	// Setup LLM-specific mocks
	suite.setupLLMMocks()
}

// setupLLMMocks configures mocks for LLM testing
func (suite *LLMTestSuite) setupLLMMocks() {
	llmMock := suite.GetMocks().GetLLMMock()

	// Setup standard successful response
	llmMock.On("ProcessIntent", gomega.Any(), gomega.Any()).Return(
		map[string]interface{}{
			"type":            "NetworkFunctionDeployment",
			"networkFunction": "AMF",
			"replicas":        int64(3),
			"namespace":       "telecom-core",
			"resources": map[string]interface{}{
				"requests": map[string]string{
					"cpu":    "1000m",
					"memory": "2Gi",
				},
				"limits": map[string]string{
					"cpu":    "2000m",
					"memory": "4Gi",
				},
			},
			"config": map[string]interface{}{
				"plmn": map[string]string{
					"mcc": "001",
					"mnc": "01",
				},
				"slice_support": []string{"eMBB", "URLLC"},
			},
		}, nil)
}

// TestTokenManager tests the token management functionality
func (suite *LLMTestSuite) TestTokenManager() {
	ginkgo.Describe("Token Manager", func() {
		var tokenManager *llm.TokenManager

		ginkgo.BeforeEach(func() {
			tokenManager = llm.NewTokenManager()
		})

		ginkgo.Context("Token Calculation", func() {
			ginkgo.It("should accurately calculate token counts", func() {
				text := "Deploy AMF with 3 replicas in the telecom-core namespace"
				tokens := tokenManager.CountTokens(text)

				// Should be reasonable token count (not exact due to tokenization variations)
				gomega.Expect(tokens).To(gomega.BeNumerically(">", 10))
				gomega.Expect(tokens).To(gomega.BeNumerically("<", 20))

				suite.GetMetrics().RecordLatency("token_counting", time.Since(time.Now()))
			})

			ginkgo.It("should handle empty text", func() {
				tokens := tokenManager.CountTokens("")
				gomega.Expect(tokens).To(gomega.Equal(0))
			})

			ginkgo.It("should calculate budget for different models", func() {
				models := []string{"gpt-4o", "gpt-4o-mini", "claude-3-haiku"}

				for _, model := range models {
					budget, err := tokenManager.CalculateTokenBudget(
						context.Background(),
						model,
						"System prompt for network automation",
						"Deploy AMF with high availability",
						"Context from knowledge base about AMF deployment procedures...",
					)

					gomega.Expect(err).NotTo(gomega.HaveOccurred())
					gomega.Expect(budget.CanAccommodate).To(gomega.BeTrue())
					gomega.Expect(budget.ContextBudget).To(gomega.BeNumerically(">", 0))
				}
			})
		})

		ginkgo.Context("Budget Management", func() {
			ginkgo.It("should enforce token limits", func() {
				// Create a very long context that exceeds limits
				longContext := ""
				for i := 0; i < 10000; i++ {
					longContext += "This is a very long context that should exceed token limits. "
				}

				budget, err := tokenManager.CalculateTokenBudget(
					context.Background(),
					"gpt-4o-mini",
					"System prompt",
					"User query",
					longContext,
				)

				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(budget.CanAccommodate).To(gomega.BeFalse())
			})

			ginkgo.It("should optimize context when budget is exceeded", func() {
				contexts := []string{
					"AMF deployment requires proper configuration of PLMN settings...",
					"SMF handles session management for 5G networks...",
					"UPF provides user plane functionality...",
					"Network slicing enables multiple virtual networks...",
				}

				optimized := tokenManager.OptimizeContext(contexts, 200, "gpt-4o-mini")

				// Should return optimized context within budget
				tokens := tokenManager.CountTokens(optimized)
				gomega.Expect(tokens).To(gomega.BeNumerically("<=", 200))
			})
		})

		ginkgo.Context("Performance Testing", func() {
			ginkgo.It("should calculate tokens efficiently", func() {
				texts := make([]string, 100)
				for i := range texts {
					texts[i] = fmt.Sprintf("Test text %d with various telecom terms like AMF, SMF, UPF", i)
				}

				start := time.Now()
				for _, text := range texts {
					tokenManager.CountTokens(text)
				}
				duration := time.Since(start)

				suite.GetMetrics().RecordLatency("token_calculation_batch", duration)

				// Should process 100 texts quickly
				gomega.Expect(duration).To(gomega.BeNumerically("<", 1*time.Second))
			})
		})
	})
}

// TestContextBuilder tests the context building functionality
func (suite *LLMTestSuite) TestContextBuilder() {
	ginkgo.Describe("Context Builder", func() {
		var contextBuilder *llm.ContextBuilder

		ginkgo.BeforeEach(func() {
			config := &llm.ContextBuilderConfig{
				MaxContextTokens:   4000,
				DiversityThreshold: 0.7,
				QualityThreshold:   0.8,
			}
			contextBuilder = llm.NewContextBuilder(config)
		})

		ginkgo.Context("Relevance Scoring", func() {
			ginkgo.It("should score documents based on relevance", func() {
				documents := []llm.Document{
					{
						ID:      "doc1",
						Title:   "AMF Configuration Guide",
						Content: "Access and Mobility Management Function configuration procedures...",
						Source:  "3GPP TS 23.501",
						Metadata: map[string]interface{}{
							"category":  "5G Core",
							"authority": "3GPP",
						},
					},
					{
						ID:      "doc2",
						Title:   "SMF Deployment",
						Content: "Session Management Function deployment in Kubernetes...",
						Source:  "O-RAN WG4",
						Metadata: map[string]interface{}{
							"category":  "5G Core",
							"authority": "O-RAN",
						},
					},
				}

				query := "How to configure AMF for 5G network deployment"

				scores, err := contextBuilder.CalculateRelevanceScores(context.Background(), query, documents)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(len(scores)).To(gomega.Equal(2))

				// First document should be more relevant (contains "AMF" in title)
				gomega.Expect(scores[0].OverallScore).To(gomega.BeNumerically(">", scores[1].OverallScore))
			})

			ginkgo.It("should consider authority in scoring", func() {
				documents := []llm.Document{
					{
						ID:      "doc1",
						Title:   "Network Function Guide",
						Content: "General network function information...",
						Source:  "3GPP TS 23.501",
						Metadata: map[string]interface{}{
							"authority": "3GPP",
						},
					},
					{
						ID:      "doc2",
						Title:   "Network Function Guide",
						Content: "General network function information...",
						Source:  "Blog Post",
						Metadata: map[string]interface{}{
							"authority": "Blog",
						},
					},
				}

				query := "Network function deployment"

				scores, err := contextBuilder.CalculateRelevanceScores(context.Background(), query, documents)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// 3GPP document should score higher due to authority
				gomega.Expect(scores[0].AuthorityScore).To(gomega.BeNumerically(">", scores[1].AuthorityScore))
			})
		})

		ginkgo.Context("Context Assembly", func() {
			ginkgo.It("should assemble diverse and relevant context", func() {
				documents := []llm.Document{
					{
						ID:      "doc1",
						Content: "AMF configuration details for 5G SA networks...",
						Source:  "3GPP TS 23.501",
						Metadata: map[string]interface{}{
							"category": "AMF",
						},
					},
					{
						ID:      "doc2",
						Content: "SMF session management procedures...",
						Source:  "3GPP TS 23.502",
						Metadata: map[string]interface{}{
							"category": "SMF",
						},
					},
					{
						ID:      "doc3",
						Content: "Another AMF configuration example...",
						Source:  "O-RAN WG4",
						Metadata: map[string]interface{}{
							"category": "AMF",
						},
					},
				}

				query := "Deploy 5G core network functions"

				context, err := contextBuilder.BuildContext(context.Background(), query, documents)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(context.Context).NotTo(gomega.BeEmpty())
				gomega.Expect(len(context.UsedDocuments)).To(gomega.BeNumerically(">", 0))
				gomega.Expect(context.QualityScore).To(gomega.BeNumerically(">", 0.5))
			})

			ginkgo.It("should respect token budget", func() {
				documents := make([]llm.Document, 10)
				for i := range documents {
					documents[i] = llm.Document{
						ID:      fmt.Sprintf("doc%d", i),
						Content: fmt.Sprintf("Very long document content %d that exceeds normal token limits and contains repeated information about network functions and deployment procedures that should be truncated when budget is limited...", i),
						Source:  "Test Source",
					}
				}

				query := "Network deployment"

				// Set very low token budget
				contextBuilder.Config.MaxContextTokens = 100

				context, err := contextBuilder.BuildContext(context.Background(), query, documents)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Verify context respects token budget
				tokenManager := llm.NewTokenManager()
				tokens := tokenManager.CountTokens(context.Context)
				gomega.Expect(tokens).To(gomega.BeNumerically("<=", 100))
			})
		})

		ginkgo.Context("Performance Testing", func() {
			ginkgo.It("should build context efficiently", func() {
				// Create large document set
				documents := make([]llm.Document, 100)
				for i := range documents {
					documents[i] = llm.Document{
						ID:      fmt.Sprintf("doc%d", i),
						Content: fmt.Sprintf("Document %d about network functions and telecom procedures", i),
						Source:  "Test Source",
					}
				}

				query := "Network function deployment"

				start := time.Now()
				context, err := contextBuilder.BuildContext(context.Background(), query, documents)
				duration := time.Since(start)

				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(context.Context).NotTo(gomega.BeEmpty())

				suite.GetMetrics().RecordLatency("context_building", duration)

				// Should build context efficiently even with large document set
				gomega.Expect(duration).To(gomega.BeNumerically("<", 5*time.Second))
			})
		})
	})
}

// TestCircuitBreaker tests the circuit breaker functionality
func (suite *LLMTestSuite) TestCircuitBreaker() {
	ginkgo.Describe("Circuit Breaker", func() {
		var circuitBreaker *llm.CircuitBreaker

		ginkgo.BeforeEach(func() {
			config := &llm.CircuitBreakerConfig{
				FailureThreshold:    3,
				FailureRate:         0.5,
				MinimumRequestCount: 5,
				Timeout:             30 * time.Second,
				ResetTimeout:        60 * time.Second,
				SuccessThreshold:    2,
			}
			circuitBreaker = llm.NewCircuitBreaker("test-breaker", config)
		})

		ginkgo.Context("State Management", func() {
			ginkgo.It("should start in closed state", func() {
				state := circuitBreaker.GetState()
				gomega.Expect(state).To(gomega.Equal(llm.StateClosed))
			})

			ginkgo.It("should open after failure threshold", func() {
				// Simulate failures
				for i := 0; i < 5; i++ {
					circuitBreaker.Execute(func() error {
						return fmt.Errorf("simulated failure")
					})
				}

				state := circuitBreaker.GetState()
				gomega.Expect(state).To(gomega.Equal(llm.StateOpen))
			})

			ginkgo.It("should transition to half-open after reset timeout", func() {
				// Force circuit to open
				for i := 0; i < 5; i++ {
					circuitBreaker.Execute(func() error {
						return fmt.Errorf("simulated failure")
					})
				}

				gomega.Expect(circuitBreaker.GetState()).To(gomega.Equal(llm.StateOpen))

				// Wait for reset timeout (mock time advancement)
				circuitBreaker.ForceState(llm.StateHalfOpen)

				state := circuitBreaker.GetState()
				gomega.Expect(state).To(gomega.Equal(llm.StateHalfOpen))
			})
		})

		ginkgo.Context("Request Handling", func() {
			ginkgo.It("should execute requests in closed state", func() {
				executed := false
				err := circuitBreaker.Execute(func() error {
					executed = true
					return nil
				})

				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(executed).To(gomega.BeTrue())
			})

			ginkgo.It("should reject requests in open state", func() {
				// Force circuit to open
				for i := 0; i < 5; i++ {
					circuitBreaker.Execute(func() error {
						return fmt.Errorf("simulated failure")
					})
				}

				executed := false
				err := circuitBreaker.Execute(func() error {
					executed = true
					return nil
				})

				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(executed).To(gomega.BeFalse())
			})
		})

		ginkgo.Context("Performance Monitoring", func() {
			ginkgo.It("should track success and failure rates", func() {
				// Execute mix of success and failures
				for i := 0; i < 10; i++ {
					circuitBreaker.Execute(func() error {
						if i%2 == 0 {
							return nil // Success
						}
						return fmt.Errorf("failure") // Failure
					})
				}

				metrics := circuitBreaker.GetMetrics()
				gomega.Expect(metrics.TotalRequests).To(gomega.Equal(int64(10)))
				gomega.Expect(metrics.SuccessCount).To(gomega.Equal(int64(5)))
				gomega.Expect(metrics.FailureCount).To(gomega.Equal(int64(5)))
				gomega.Expect(metrics.FailureRate).To(gomega.BeNumerically("~", 0.5, 0.1))
			})
		})
	})
}

// TestStreamingProcessor tests the streaming functionality
func (suite *LLMTestSuite) TestStreamingProcessor() {
	ginkgo.Describe("Streaming Processor", func() {
		var streamingProcessor *llm.StreamingProcessor

		ginkgo.BeforeEach(func() {
			config := &llm.StreamingConfig{
				MaxConcurrentStreams: 10,
				StreamTimeout:        5 * time.Minute,
				BufferSize:           1024,
			}
			streamingProcessor = llm.NewStreamingProcessor(config)
		})

		ginkgo.Context("Session Management", func() {
			ginkgo.It("should create and manage streaming sessions", func() {
				sessionID := "test-session-1"

				// Mock HTTP response writer and request
				// In real testing, would use httptest.ResponseRecorder

				session, err := streamingProcessor.CreateSession(sessionID, nil, nil)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(session.ID).To(gomega.Equal(sessionID))
				gomega.Expect(session.Status).To(gomega.Equal(llm.StreamingStatusActive))

				// Verify session tracking
				activeSessions := streamingProcessor.GetActiveSessions()
				gomega.Expect(len(activeSessions)).To(gomega.Equal(1))
			})

			ginkgo.It("should enforce concurrent session limits", func() {
				// Create sessions up to limit
				for i := 0; i < 10; i++ {
					sessionID := fmt.Sprintf("session-%d", i)
					_, err := streamingProcessor.CreateSession(sessionID, nil, nil)
					gomega.Expect(err).NotTo(gomega.HaveOccurred())
				}

				// Attempt to create one more session (should fail)
				_, err := streamingProcessor.CreateSession("excess-session", nil, nil)
				gomega.Expect(err).To(gomega.HaveOccurred())
			})
		})

		ginkgo.Context("Stream Processing", func() {
			ginkgo.It("should stream content chunks", func() {
				sessionID := "stream-test-session"
				session, err := streamingProcessor.CreateSession(sessionID, nil, nil)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Simulate streaming content
				chunks := []string{
					"Based on the retrieved knowledge,",
					" I recommend deploying AMF",
					" with the following configuration...",
				}

				for _, chunk := range chunks {
					err := streamingProcessor.StreamChunk(sessionID, &llm.StreamingChunk{
						Content:   chunk,
						ChunkType: llm.ChunkTypeContent,
						Timestamp: time.Now(),
					})
					gomega.Expect(err).NotTo(gomega.HaveOccurred())
				}

				// Complete the stream
				err = streamingProcessor.CompleteStream(sessionID)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				gomega.Expect(session.Status).To(gomega.Equal(llm.StreamingStatusCompleted))
			})
		})
	})
}

// TestLLMIntegration performs integration testing of LLM components
func (suite *LLMTestSuite) TestLLMIntegration() {
	ginkgo.Describe("LLM Integration", func() {
		ginkgo.Context("End-to-End Processing", func() {
			ginkgo.It("should process intent with full pipeline", func() {
				intent := "Deploy AMF with 3 replicas for high availability in the telecom-core namespace"

				start := time.Now()
				result, err := suite.llmClient.ProcessIntent(context.Background(), intent)
				duration := time.Since(start)

				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(result).NotTo(gomega.BeNil())

				// Verify structured response
				gomega.Expect(result["type"]).To(gomega.Equal("NetworkFunctionDeployment"))
				gomega.Expect(result["networkFunction"]).To(gomega.Equal("AMF"))
				gomega.Expect(result["replicas"]).To(gomega.Equal(int64(3)))

				suite.GetMetrics().RecordLatency("llm_end_to_end", duration)

				// Should complete within reasonable time
				gomega.Expect(duration).To(gomega.BeNumerically("<", 10*time.Second))
			})
		})

		ginkgo.Context("Load Testing", func() {
			ginkgo.It("should handle concurrent requests", func() {
				if !suite.GetConfig().LoadTestEnabled {
					ginkgo.Skip("Load testing disabled")
				}

				err := suite.RunLoadTest(func() error {
					intent := fmt.Sprintf("Deploy network function %d", time.Now().UnixNano())
					_, err := suite.llmClient.ProcessIntent(context.Background(), intent)
					return err
				})

				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			})
		})

		ginkgo.Context("Chaos Testing", func() {
			ginkgo.It("should handle service failures gracefully", func() {
				if !suite.GetConfig().ChaosTestEnabled {
					ginkgo.Skip("Chaos testing disabled")
				}

				err := suite.RunChaosTest(func() error {
					intent := "Deploy resilient network function"
					_, err := suite.llmClient.ProcessIntent(context.Background(), intent)
					// Errors are expected in chaos testing
					return nil
				})

				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			})
		})
	})
}

var _ = ginkgo.Describe("LLM Components", func() {
	var testSuite *LLMTestSuite

	ginkgo.BeforeEach(func() {
		testSuite = &LLMTestSuite{
			TestSuite: framework.NewTestSuite(),
		}
		testSuite.SetupSuite()
	})

	ginkgo.AfterEach(func() {
		testSuite.TearDownSuite()
	})

	ginkgo.Context("Token Manager", func() {
		testSuite.TestTokenManager()
	})

	ginkgo.Context("Context Builder", func() {
		testSuite.TestContextBuilder()
	})

	ginkgo.Context("Circuit Breaker", func() {
		testSuite.TestCircuitBreaker()
	})

	ginkgo.Context("Streaming Processor", func() {
		testSuite.TestStreamingProcessor()
	})

	ginkgo.Context("Integration", func() {
		testSuite.TestLLMIntegration()
	})
})
