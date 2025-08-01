package llm

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestProcessingPipeline(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Processing Pipeline Suite")
}

var _ = Describe("Processing Pipeline Tests", func() {
	var pipeline *ProcessingPipeline

	BeforeEach(func() {
		config := PipelineConfig{
			EnablePreprocessing:          true,
			EnableContextEnrichment:      true,
			EnableResponseTransformation: true,
			EnablePostprocessing:         true,
			MaxProcessingTime:            30 * time.Second,
			ValidationStrictness:         "normal",
			CacheEnabled:                 true,
			AsyncProcessing:              false,
		}
		pipeline = NewProcessingPipeline(config)
	})

	Context("Intent Preprocessing", func() {
		It("should normalize network function names", func() {
			preprocessor := NewIntentPreprocessor()

			testCases := []struct {
				input    string
				expected string
			}{
				{"Deploy UPF with 3 replicas", "Deploy User Plane Function with 3 replicas"},
				{"Setup AMF in core network", "Setup Access and Mobility Management Function in core network"},
				{"Configure near-rt ric for optimization", "Configure Near Real-Time RAN Intelligent Controller for optimization"},
			}

			for _, tc := range testCases {
				result, err := preprocessor.Preprocess(tc.input)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(tc.expected))
			}
		})

		It("should handle empty intents", func() {
			preprocessor := NewIntentPreprocessor()

			_, err := preprocessor.Preprocess("")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty intent"))
		})

		It("should normalize whitespace", func() {
			preprocessor := NewIntentPreprocessor()

			result, err := preprocessor.Preprocess("Deploy    UPF   with   multiple   spaces")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal("Deploy User Plane Function with multiple spaces"))
		})
	})

	Context("Intent Classification", func() {
		It("should classify deployment intents correctly", func() {
			classifier := NewIntentClassifier()

			testCases := []struct {
				intent     string
				expected   string
				confidence float64
			}{
				{"Deploy UPF network function", "NetworkFunctionDeployment", 0.8},
				{"Create AMF instance", "NetworkFunctionDeployment", 0.8},
				{"Install SMF service", "NetworkFunctionDeployment", 0.8},
				{"Setup Near-RT RIC", "NetworkFunctionDeployment", 0.8},
			}

			for _, tc := range testCases {
				result, err := classifier.Classify(tc.intent)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.IntentType).To(Equal(tc.expected))
				Expect(result.Confidence).To(Equal(tc.confidence))
			}
		})

		It("should classify scaling intents correctly", func() {
			classifier := NewIntentClassifier()

			testCases := []struct {
				intent   string
				expected string
			}{
				{"Scale AMF to 5 replicas", "NetworkFunctionScale"},
				{"Increase UPF instances", "NetworkFunctionScale"},
				{"Resize memory allocation", "NetworkFunctionScale"},
				{"Decrease deployment size", "NetworkFunctionScale"},
			}

			for _, tc := range testCases {
				result, err := classifier.Classify(tc.intent)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.IntentType).To(Equal(tc.expected))
			}
		})

		It("should detect network functions", func() {
			classifier := NewIntentClassifier()

			testCases := []struct {
				intent     string
				expectedNF string
			}{
				{"Deploy User Plane Function", "UPF"},
				{"Setup Access and Mobility Management", "AMF"},
				{"Configure Session Management Function", "SMF"},
				{"Install Near Real-Time RIC", "Near-RT-RIC"},
			}

			for _, tc := range testCases {
				result, err := classifier.Classify(tc.intent)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.NetworkFunction).To(Equal(tc.expectedNF))
			}
		})

		It("should determine operation types", func() {
			classifier := NewIntentClassifier()

			testCases := []struct {
				intent            string
				expectedOperation string
			}{
				{"Deploy UPF with high availability", "HighAvailability"},
				{"Setup AMF at edge location", "EdgeDeployment"},
				{"Configure SMF in core network", "CoreDeployment"},
			}

			for _, tc := range testCases {
				result, err := classifier.Classify(tc.intent)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.OperationType).To(Equal(tc.expectedOperation))
			}
		})

		It("should assign priority levels", func() {
			classifier := NewIntentClassifier()

			testCases := []struct {
				intent           string
				expectedPriority string
			}{
				{"Urgent deployment of AMF", "High"},
				{"Critical UPF scaling needed", "High"},
				{"Background SMF update", "Low"},
				{"Regular PCF deployment", "Medium"},
			}

			for _, tc := range testCases {
				result, err := classifier.Classify(tc.intent)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.Priority).To(Equal(tc.expectedPriority))
			}
		})
	})

	Context("Input Validation", func() {
		It("should validate intent length", func() {
			validator := NewInputValidator("strict")

			// Too short
			result := validator.Validate("short")
			Expect(result.Valid).To(BeFalse())
			Expect(len(result.Errors)).To(BeNumerically(">", 0))

			// Good length
			result = validator.Validate("This is a properly sized intent for testing validation")
			Expect(result.Valid).To(BeTrue())
		})

		It("should detect malicious content", func() {
			validator := NewInputValidator("strict")

			maliciousIntents := []string{
				"Deploy UPF; DROP TABLE users;",
				"SELECT * FROM secrets UNION ALL",
				"<script>alert('xss')</script>",
			}

			for _, intent := range maliciousIntents {
				result := validator.Validate(intent)
				// Should detect SQL injection patterns
				hasSecurityError := false
				for _, err := range result.Errors {
					if err.Code == "NoSQLInjection" {
						hasSecurityError = true
						break
					}
				}
				Expect(hasSecurityError).To(BeTrue(), "Should detect malicious content in: %s", intent)
			}
		})

		It("should calculate validation scores", func() {
			validator := NewInputValidator("normal")

			// Perfect intent
			result := validator.Validate("Deploy UPF network function with high availability in core network")
			Expect(result.Score).To(Equal(1.0))

			// Intent with issues should have lower score
			result = validator.Validate("short")
			Expect(result.Score).To(BeNumerically("<", 1.0))
		})
	})

	Context("Context Enrichment", func() {
		It("should enrich with network topology", func() {
			enricher := NewContextEnricher()

			ctx := &ProcessingContext{
				RequestID: "test-123",
				Intent:    "Deploy UPF in edge location",
				Classification: ClassificationResult{
					IntentType:      "NetworkFunctionDeployment",
					NetworkFunction: "UPF",
					OperationType:   "EdgeDeployment",
				},
			}

			enrichment, err := enricher.Enrich(context.Background(), ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(enrichment.NetworkTopology).ToNot(BeNil())
			Expect(enrichment.NetworkTopology.Region).ToNot(BeEmpty())
			Expect(len(enrichment.NetworkTopology.NetworkSlices)).To(BeNumerically(">", 0))
		})

		It("should enrich with deployment context", func() {
			enricher := NewContextEnricher()

			ctx := &ProcessingContext{
				RequestID: "test-456",
				Intent:    "Scale AMF to 5 replicas",
				Classification: ClassificationResult{
					IntentType:      "NetworkFunctionScale",
					NetworkFunction: "AMF",
				},
			}

			enrichment, err := enricher.Enrich(context.Background(), ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(enrichment.DeploymentContext).ToNot(BeNil())
			Expect(enrichment.DeploymentContext.Environment).ToNot(BeEmpty())
			Expect(enrichment.DeploymentContext.Cluster).ToNot(BeEmpty())
		})

		It("should cache enrichment data", func() {
			enricher := NewContextEnricher()

			ctx := &ProcessingContext{
				RequestID: "test-cache",
				Intent:    "Deploy UPF",
				Classification: ClassificationResult{
					IntentType:      "NetworkFunctionDeployment",
					NetworkFunction: "UPF",
				},
			}

			// First call
			start1 := time.Now()
			enrichment1, err1 := enricher.Enrich(context.Background(), ctx)
			duration1 := time.Since(start1)

			Expect(err1).ToNot(HaveOccurred())

			// Second call with same classification should be faster (cached)
			start2 := time.Now()
			enrichment2, err2 := enricher.Enrich(context.Background(), ctx)
			duration2 := time.Since(start2)

			Expect(err2).ToNot(HaveOccurred())
			Expect(enrichment2).To(Equal(enrichment1))
			Expect(duration2).To(BeNumerically("<", duration1))
		})
	})

	Context("Response Transformation", func() {
		It("should transform deployment responses", func() {
			transformer := NewResponseTransformer()

			response := map[string]interface{}{
				"type":      "NetworkFunctionDeployment",
				"name":      "test-upf",
				"namespace": "5g-core",
				"metadata":  map[string]interface{}{},
				"spec": map[string]interface{}{
					"replicas": 3,
					"image":    "upf:latest",
				},
			}

			transformed, err := transformer.Transform(response, "NetworkFunctionDeployment")
			Expect(err).ToNot(HaveOccurred())

			metadata := transformed["metadata"].(map[string]interface{})
			Expect(metadata).To(HaveKey("deployment_strategy"))
			Expect(metadata["deployment_strategy"]).To(Equal("RollingUpdate"))
		})

		It("should transform scaling responses", func() {
			transformer := NewResponseTransformer()

			response := map[string]interface{}{
				"type":      "NetworkFunctionScale",
				"name":      "test-amf",
				"namespace": "5g-core",
				"spec": map[string]interface{}{
					"scaling": map[string]interface{}{
						"horizontal": map[string]interface{}{
							"replicas": 5,
						},
					},
				},
			}

			transformed, err := transformer.Transform(response, "NetworkFunctionScale")
			Expect(err).ToNot(HaveOccurred())

			spec := transformed["spec"].(map[string]interface{})
			scaling := spec["scaling"].(map[string]interface{})
			Expect(scaling).To(HaveKey("strategy"))
			Expect(scaling["strategy"]).To(Equal("gradual"))
		})
	})

	Context("Response Postprocessing", func() {
		It("should add processing metadata", func() {
			postprocessor := NewResponsePostprocessor()

			response := map[string]interface{}{
				"type": "NetworkFunctionDeployment",
				"name": "test-nf",
			}

			ctx := &ProcessingContext{
				RequestID: "test-post-123",
				Intent:    "Deploy test NF",
				Classification: ClassificationResult{
					IntentType: "NetworkFunctionDeployment",
					Confidence: 0.95,
				},
				ProcessingStart: time.Now().Add(-100 * time.Millisecond),
			}

			processed, err := postprocessor.Postprocess(response, ctx)
			Expect(err).ToNot(HaveOccurred())

			Expect(processed).To(HaveKey("processing_metadata"))
			Expect(processed).To(HaveKey("validation_result"))
			Expect(processed).To(HaveKey("tracking"))

			metadata := processed["processing_metadata"].(map[string]interface{})
			Expect(metadata["request_id"]).To(Equal("test-post-123"))
			Expect(metadata["confidence_score"]).To(Equal(0.95))
		})
	})

	Context("Full Pipeline Integration", func() {
		It("should process intents end-to-end", func() {
			ctx := context.Background()
			metadata := map[string]interface{}{
				"user_id":    "test-user",
				"session_id": "test-session",
			}

			result, err := pipeline.ProcessIntent(ctx, "Deploy UPF network function with 3 replicas for high availability", metadata)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())
			Expect(result.Success).To(BeTrue())
			Expect(result.ProcessingContext.RequestID).ToNot(BeEmpty())
			Expect(result.ProcessingContext.Classification.IntentType).To(Equal("NetworkFunctionDeployment"))
			Expect(result.ProcessingContext.Classification.NetworkFunction).To(Equal("UPF"))
			Expect(result.ProcessingContext.EnrichmentData).ToNot(BeNil())
		})

		It("should handle validation failures in strict mode", func() {
			strictConfig := PipelineConfig{
				EnablePreprocessing:          true,
				EnableContextEnrichment:      true,
				EnableResponseTransformation: true,
				EnablePostprocessing:         true,
				MaxProcessingTime:            30 * time.Second,
				ValidationStrictness:         "strict",
				CacheEnabled:                 true,
				AsyncProcessing:              false,
			}
			strictPipeline := NewProcessingPipeline(strictConfig)

			ctx := context.Background()
			metadata := map[string]interface{}{}

			// Too short intent should fail in strict mode
			_, err := strictPipeline.ProcessIntent(ctx, "short", metadata)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("validation failed"))
		})

		It("should track processing metrics", func() {
			ctx := context.Background()
			metadata := map[string]interface{}{}

			// Process several intents
			intents := []string{
				"Deploy UPF network function with high availability",
				"Scale AMF to 5 replicas for increased capacity",
				"Configure SMF in edge deployment",
			}

			for _, intent := range intents {
				result, err := pipeline.ProcessIntent(ctx, intent, metadata)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.Success).To(BeTrue())
			}

			metrics := pipeline.GetMetrics()
			Expect(metrics.TotalRequests).To(Equal(int64(len(intents))))
			Expect(metrics.SuccessfulRequests).To(Equal(int64(len(intents))))
			Expect(metrics.FailedRequests).To(Equal(int64(0)))
		})
	})

	Context("Telecom Knowledge Base", func() {
		It("should provide network function specifications", func() {
			kb := NewTelecomKnowledgeBase()

			spec, found := kb.GetNetworkFunctionSpec("UPF")
			Expect(found).To(BeTrue())
			Expect(spec.Name).To(Equal("User Plane Function"))
			Expect(spec.Type).To(Equal("5G Core"))
			Expect(spec.DefaultResources).To(HaveKey("cpu"))
			Expect(spec.DefaultResources).To(HaveKey("memory"))
			Expect(len(spec.RequiredPorts)).To(BeNumerically(">", 0))
		})

		It("should handle unknown network functions", func() {
			kb := NewTelecomKnowledgeBase()

			_, found := kb.GetNetworkFunctionSpec("UNKNOWN_NF")
			Expect(found).To(BeFalse())
		})
	})
})

// Performance tests for pipeline
var _ = Describe("Processing Pipeline Performance", func() {
	var pipeline *ProcessingPipeline

	BeforeEach(func() {
		config := PipelineConfig{
			EnablePreprocessing:          true,
			EnableContextEnrichment:      true,
			EnableResponseTransformation: true,
			EnablePostprocessing:         true,
			MaxProcessingTime:            30 * time.Second,
			ValidationStrictness:         "normal",
			CacheEnabled:                 true,
			AsyncProcessing:              false,
		}
		pipeline = NewProcessingPipeline(config)
	})

	It("should process intents within time limits", func() {
		ctx := context.Background()
		metadata := map[string]interface{}{}

		testIntents := []string{
			"Deploy UPF network function with 3 replicas",
			"Scale AMF to 5 instances for high availability",
			"Configure SMF in edge deployment with GPU acceleration",
			"Setup Near-RT RIC with xApp support",
			"Install O-DU with DPDK optimization",
		}

		for _, intent := range testIntents {
			start := time.Now()
			result, err := pipeline.ProcessIntent(ctx, intent, metadata)
			duration := time.Since(start)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Success).To(BeTrue())
			Expect(duration).To(BeNumerically("<", 5*time.Second), "Processing should complete within 5 seconds for intent: %s", intent)
		}
	})

	It("should demonstrate caching benefits", func() {
		ctx := context.Background()
		metadata := map[string]interface{}{}
		intent := "Deploy UPF network function with high availability"

		// First processing (no cache)
		start1 := time.Now()
		result1, err1 := pipeline.ProcessIntent(ctx, intent, metadata)
		duration1 := time.Since(start1)

		Expect(err1).ToNot(HaveOccurred())
		Expect(result1.Success).To(BeTrue())

		// Second processing (with cache for enrichment)
		start2 := time.Now()
		result2, err2 := pipeline.ProcessIntent(ctx, intent, metadata)
		duration2 := time.Since(start2)

		Expect(err2).ToNot(HaveOccurred())
		Expect(result2.Success).To(BeTrue())

		// Should be faster due to cached enrichment data
		Expect(duration2).To(BeNumerically("<=", duration1))
	})

	It("should handle concurrent processing", func() {
		const numWorkers = 10
		const numRequestsPerWorker = 20

		ctx := context.Background()
		metadata := map[string]interface{}{}

		results := make(chan error, numWorkers*numRequestsPerWorker)
		var wg sync.WaitGroup

		start := time.Now()

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				for j := 0; j < numRequestsPerWorker; j++ {
					intent := fmt.Sprintf("Deploy UPF-%d-%d with 3 replicas", workerID, j)
					_, err := pipeline.ProcessIntent(ctx, intent, metadata)
					results <- err
				}
			}(i)
		}

		wg.Wait()
		close(results)

		duration := time.Since(start)

		// Count errors
		errorCount := 0
		for err := range results {
			if err != nil {
				errorCount++
			}
		}

		totalRequests := numWorkers * numRequestsPerWorker
		Expect(errorCount).To(Equal(0), "All requests should succeed")
		Expect(duration).To(BeNumerically("<", 30*time.Second), "Concurrent processing should complete within 30 seconds")

		// Check metrics
		metrics := pipeline.GetMetrics()
		Expect(metrics.TotalRequests).To(BeNumerically(">=", int64(totalRequests)))
		Expect(metrics.SuccessfulRequests).To(Equal(metrics.TotalRequests))
	})
})
