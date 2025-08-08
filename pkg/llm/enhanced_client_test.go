package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEnhancedClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Enhanced LLM Client Suite")
}

var _ = Describe("Enhanced LLM Client Tests", func() {
	var (
		enhancedClient *EnhancedClient
		mockServer     *httptest.Server
		config         EnhancedClientConfig
	)

	BeforeEach(func() {
		// Create mock server
		mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]interface{}{
				"type":      "NetworkFunctionDeployment",
				"name":      "test-nf",
				"namespace": "default",
				"spec": map[string]interface{}{
					"replicas": float64(1),
					"image":    "test:latest",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))

		config = EnhancedClientConfig{
			ClientConfig: ClientConfig{
				APIKey:      "test-key",
				ModelName:   "test-model",
				MaxTokens:   1000,
				BackendType: "test",
				Timeout:     30 * time.Second,
			},
			CircuitBreakerThreshold: 5,
			CircuitBreakerTimeout:   time.Minute,
			RateLimitTokens:         10,
			RateLimitRefillRate:     1,
			HealthCheckInterval:     time.Minute,
			HealthCheckTimeout:      10 * time.Second,
		}

		enhancedClient = NewEnhancedClient(mockServer.URL, config)
	})

	AfterEach(func() {
		enhancedClient.healthChecker.Stop()
		mockServer.Close()
	})

	Context("Circuit Breaker", func() {
		It("should start in closed state", func() {
			state := enhancedClient.circuitBreaker.GetState()
			Expect(state).To(Equal(CircuitClosed))
		})

		It("should open after threshold failures", func() {
			// Create failing server
			failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			}))
			defer failingServer.Close()

			failingClient := NewEnhancedClient(failingServer.URL, config)
			defer failingClient.healthChecker.Stop()

			// Trigger failures
			for i := 0; i < int(config.CircuitBreakerThreshold); i++ {
				_, err := failingClient.ProcessIntentWithEnhancements(context.Background(), "test intent")
				Expect(err).To(HaveOccurred())
			}

			// Circuit should be open now
			state := failingClient.circuitBreaker.GetState()
			Expect(state).To(Equal(CircuitOpen))
		})

		It("should reject requests when open", func() {
			// Manually open circuit
			enhancedClient.circuitBreaker.state = CircuitOpen

			_, err := enhancedClient.ProcessIntentWithEnhancements(context.Background(), "test intent")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("circuit breaker is open"))
		})
	})

	Context("Rate Limiting", func() {
		It("should allow requests within rate limit", func() {
			result, err := enhancedClient.ProcessIntentWithEnhancements(context.Background(), "test intent")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeEmpty())
		})

		It("should reject requests when rate limit exceeded", func() {
			// Exhaust all tokens
			for i := 0; i < int(config.RateLimitTokens); i++ {
				enhancedClient.rateLimiter.Allow()
			}

			_, err := enhancedClient.ProcessIntentWithEnhancements(context.Background(), "test intent")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("rate limit exceeded"))
		})

		It("should refill tokens over time", func() {
			// Exhaust tokens
			for i := 0; i < int(config.RateLimitTokens); i++ {
				enhancedClient.rateLimiter.Allow()
			}

			initialTokens := enhancedClient.rateLimiter.GetTokens()
			Expect(initialTokens).To(Equal(int64(0)))

			// Wait for refill
			time.Sleep(2 * time.Second)

			currentTokens := enhancedClient.rateLimiter.GetTokens()
			Expect(currentTokens).To(BeNumerically(">", initialTokens))
		})
	})

	Context("Health Checking", func() {
		It("should track health status", func() {
			// Give some time for health check to run
			time.Sleep(100 * time.Millisecond)

			health := enhancedClient.healthChecker.GetHealth()
			Expect(health).ToNot(BeEmpty())
		})

		It("should detect unhealthy backends", func() {
			// Create failing server
			failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			}))
			defer failingServer.Close()

			failingConfig := config
			failingConfig.HealthCheckInterval = 100 * time.Millisecond

			failingClient := NewEnhancedClient(failingServer.URL, failingConfig)
			defer failingClient.healthChecker.Stop()

			// Wait for health check
			time.Sleep(200 * time.Millisecond)

			health := failingClient.healthChecker.GetHealth()
			for _, status := range health {
				Expect(status.Available).To(BeFalse())
			}
		})
	})

	Context("Enhanced Metrics", func() {
		It("should provide comprehensive metrics", func() {
			// Process some requests
			enhancedClient.ProcessIntentWithEnhancements(context.Background(), "test intent 1")
			enhancedClient.ProcessIntentWithEnhancements(context.Background(), "test intent 2")

			metrics := enhancedClient.GetEnhancedMetrics()
			Expect(metrics).To(HaveKey("base_metrics"))
			Expect(metrics).To(HaveKey("circuit_breaker"))
			Expect(metrics).To(HaveKey("rate_limiter"))
			Expect(metrics).To(HaveKey("health_status"))

			circuitBreakerMetrics := metrics["circuit_breaker"].(map[string]interface{})
			Expect(circuitBreakerMetrics).To(HaveKey("state"))
		})
	})

	Context("Caching", func() {
		It("should cache responses", func() {
			ctx := context.Background()
			intent := "test intent for caching"

			// First request should miss cache
			start1 := time.Now()
			result1, err1 := enhancedClient.ProcessIntentWithEnhancements(ctx, intent)
			duration1 := time.Since(start1)

			Expect(err1).ToNot(HaveOccurred())
			Expect(result1).ToNot(BeEmpty())

			// Second request should hit cache and be faster
			start2 := time.Now()
			result2, err2 := enhancedClient.ProcessIntentWithEnhancements(ctx, intent)
			duration2 := time.Since(start2)

			Expect(err2).ToNot(HaveOccurred())
			Expect(result2).To(Equal(result1))
			Expect(duration2).To(BeNumerically("<", duration1))
		})
	})

	Context("Fallback URLs", func() {
		It("should try fallback URLs on primary failure", func() {
			// Create a primary failing server and a working fallback
			failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			}))
			defer failingServer.Close()

			workingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				response := map[string]interface{}{
					"type":      "NetworkFunctionDeployment",
					"name":      "fallback-nf",
					"namespace": "default",
					"spec": map[string]interface{}{
						"replicas": float64(1),
						"image":    "fallback:latest",
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer workingServer.Close()

			failingClient := NewEnhancedClient(failingServer.URL, config)
			defer failingClient.healthChecker.Stop()

			// Set fallback URL
			failingClient.SetFallbackURLs([]string{workingServer.URL})

			result, err := failingClient.ProcessIntentWithEnhancements(context.Background(), "test intent")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(ContainSubstring("fallback-nf"))
		})
	})
})

var _ = Describe("Cache Manager Tests", func() {
	var cacheManager *CacheManager

	BeforeEach(func() {
		cacheManager = NewCacheManager(time.Minute, 100, false, false, nil)
	})

	Context("Basic Caching", func() {
		It("should store and retrieve responses", func() {
			key := "test-key"
			response := "test response"

			cacheManager.SetWithTags(key, response, []string{"tag1", "tag2"})

			retrieved, metadata, found := cacheManager.GetWithMetadata(key)
			Expect(found).To(BeTrue())
			Expect(retrieved).To(Equal(response))
			Expect(metadata).To(HaveKey("cached"))
			Expect(metadata["cached"]).To(BeTrue())
		})

		It("should invalidate by pattern", func() {
			// Store multiple entries
			cacheManager.SetWithTags("test-key-1", "response1", []string{"test"})
			cacheManager.SetWithTags("test-key-2", "response2", []string{"test"})
			cacheManager.SetWithTags("other-key", "response3", []string{"other"})

			// Invalidate by pattern
			count := cacheManager.InvalidateByPattern("test")
			Expect(count).To(Equal(2))

			// Check that test entries are gone but other remains
			_, _, found1 := cacheManager.GetWithMetadata("test-key-1")
			_, _, found2 := cacheManager.GetWithMetadata("test-key-2")
			_, _, found3 := cacheManager.GetWithMetadata("other-key")

			Expect(found1).To(BeFalse())
			Expect(found2).To(BeFalse())
			Expect(found3).To(BeTrue())
		})
	})
})

var _ = Describe("Response Enhancer Tests", func() {
	var enhancer *ResponseEnhancer

	BeforeEach(func() {
		enhancer = NewResponseEnhancer()
	})

	Context("Response Enhancement", func() {
		It("should enhance responses with default enrichments", func() {
			originalResponse := `{
				"type": "NetworkFunctionDeployment",
				"name": "test-nf",
				"namespace": "default",
				"spec": {
					"replicas": 1,
					"image": "test:latest"
				}
			}`

			enhanced, err := enhancer.EnhanceResponse(originalResponse, "NetworkFunctionDeployment")
			Expect(err).ToNot(HaveOccurred())

			var enhancedMap map[string]interface{}
			err = json.Unmarshal([]byte(enhanced), &enhancedMap)
			Expect(err).ToNot(HaveOccurred())

			Expect(enhancedMap).To(HaveKey("processing_metadata"))
			Expect(enhancedMap).To(HaveKey("response_id"))

			processingMetadata := enhancedMap["processing_metadata"].(map[string]interface{})
			Expect(processingMetadata["enhanced"]).To(BeTrue())
		})

		It("should apply custom enrichment rules", func() {
			// Add custom enrichment rule
			enhancer.AddEnrichmentRule("NetworkFunctionDeployment", func(response map[string]interface{}) map[string]interface{} {
				response["custom_field"] = "custom_value"
				return response
			})

			originalResponse := `{
				"type": "NetworkFunctionDeployment",
				"name": "test-nf",
				"namespace": "default",
				"spec": {
					"replicas": 1,
					"image": "test:latest"
				}
			}`

			enhanced, err := enhancer.EnhanceResponse(originalResponse, "NetworkFunctionDeployment")
			Expect(err).ToNot(HaveOccurred())

			var enhancedMap map[string]interface{}
			err = json.Unmarshal([]byte(enhanced), &enhancedMap)
			Expect(err).ToNot(HaveOccurred())

			Expect(enhancedMap).To(HaveKey("custom_field"))
			Expect(enhancedMap["custom_field"]).To(Equal("custom_value"))
		})
	})
})

var _ = Describe("Request Context Manager Tests", func() {
	var contextManager *RequestContextManager

	BeforeEach(func() {
		contextManager = NewRequestContextManager()
	})

	Context("Context Management", func() {
		It("should create and manage request contexts", func() {
			ctx := contextManager.CreateContext("test intent", "user123", "session456")

			Expect(ctx.ID).ToNot(BeEmpty())
			Expect(ctx.Intent).To(Equal("test intent"))
			Expect(ctx.UserID).To(Equal("user123"))
			Expect(ctx.SessionID).To(Equal("session456"))

			// Should be able to retrieve it
			retrieved, found := contextManager.GetContext(ctx.ID)
			Expect(found).To(BeTrue())
			Expect(retrieved.Intent).To(Equal(ctx.Intent))
		})

		It("should track active requests", func() {
			ctx1 := contextManager.CreateContext("intent 1", "user1", "session1")
			ctx2 := contextManager.CreateContext("intent 2", "user2", "session2")

			activeRequests := contextManager.GetActiveRequests()
			Expect(len(activeRequests)).To(Equal(2))

			// Complete one request
			contextManager.CompleteContext(ctx1.ID)

			activeRequests = contextManager.GetActiveRequests()
			Expect(len(activeRequests)).To(Equal(1))
			Expect(activeRequests[0].ID).To(Equal(ctx2.ID))
		})
	})
})

// Benchmark tests
var _ = Describe("Performance Benchmarks", func() {
	var (
		enhancedClient *EnhancedClient
		mockServer     *httptest.Server
	)

	BeforeEach(func() {
		mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate some processing time
			time.Sleep(10 * time.Millisecond)

			response := map[string]interface{}{
				"type":      "NetworkFunctionDeployment",
				"name":      "benchmark-nf",
				"namespace": "default",
				"spec": map[string]interface{}{
					"replicas": float64(1),
					"image":    "benchmark:latest",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))

		config := EnhancedClientConfig{
			ClientConfig: ClientConfig{
				APIKey:      "test-key",
				ModelName:   "test-model",
				MaxTokens:   1000,
				BackendType: "test",
				Timeout:     30 * time.Second,
			},
			CircuitBreakerThreshold: 100,
			CircuitBreakerTimeout:   time.Minute,
			RateLimitTokens:         1000,
			RateLimitRefillRate:     100,
			HealthCheckInterval:     time.Minute,
			HealthCheckTimeout:      10 * time.Second,
		}

		enhancedClient = NewEnhancedClient(mockServer.URL, config)
	})

	AfterEach(func() {
		enhancedClient.healthChecker.Stop()
		mockServer.Close()
	})

	It("should handle concurrent requests efficiently", func() {
		const numRequests = 100
		const numWorkers = 10

		var wg sync.WaitGroup
		results := make(chan error, numRequests)
		requests := make(chan int, numRequests)

		// Fill request channel
		for i := 0; i < numRequests; i++ {
			requests <- i
		}
		close(requests)

		start := time.Now()

		// Start workers
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range requests {
					_, err := enhancedClient.ProcessIntentWithEnhancements(
						context.Background(),
						fmt.Sprintf("benchmark intent %d", time.Now().UnixNano()),
					)
					results <- err
				}
			}()
		}

		wg.Wait()
		close(results)

		duration := time.Since(start)

		// Collect results
		var errors []error
		for err := range results {
			if err != nil {
				errors = append(errors, err)
			}
		}

		fmt.Printf("Processed %d requests in %v\n", numRequests, duration)
		fmt.Printf("Requests per second: %.2f\n", float64(numRequests)/duration.Seconds())
		fmt.Printf("Errors: %d\n", len(errors))

		Expect(len(errors)).To(BeNumerically("<", numRequests/10)) // Less than 10% errors
		Expect(duration).To(BeNumerically("<", 30*time.Second))    // Should complete within 30 seconds
	})

	It("should demonstrate cache performance benefits", func() {
		ctx := context.Background()
		intent := "performance test intent"

		// Measure uncached request
		start := time.Now()
		_, err := enhancedClient.ProcessIntentWithEnhancements(ctx, intent)
		uncachedDuration := time.Since(start)

		Expect(err).ToNot(HaveOccurred())

		// Measure cached request
		start = time.Now()
		_, err = enhancedClient.ProcessIntentWithEnhancements(ctx, intent)
		cachedDuration := time.Since(start)

		Expect(err).ToNot(HaveOccurred())

		fmt.Printf("Uncached request: %v\n", uncachedDuration)
		fmt.Printf("Cached request: %v\n", cachedDuration)
		fmt.Printf("Cache speedup: %.2fx\n", float64(uncachedDuration)/float64(cachedDuration))

		// Cached request should be significantly faster
		Expect(cachedDuration).To(BeNumerically("<", uncachedDuration/2))
	})

	It("should measure memory usage", func() {
		const numRequests = 1000

		// Get initial metrics
		initialMetrics := enhancedClient.GetEnhancedMetrics()
		baseMetrics := initialMetrics["base_metrics"].(ClientMetrics)
		initialRequests := baseMetrics.RequestsTotal

		// Process many requests
		for i := 0; i < numRequests; i++ {
			enhancedClient.ProcessIntentWithEnhancements(
				context.Background(),
				fmt.Sprintf("memory test intent %d", i),
			)
		}

		// Get final metrics
		finalMetrics := enhancedClient.GetEnhancedMetrics()
		finalBaseMetrics := finalMetrics["base_metrics"].(ClientMetrics)
		finalRequests := finalBaseMetrics.RequestsTotal

		requestsProcessed := finalRequests - initialRequests

		fmt.Printf("Processed %d requests\n", requestsProcessed)
		fmt.Printf("Cache hits: %d\n", finalBaseMetrics.CacheHits)
		fmt.Printf("Cache misses: %d\n", finalBaseMetrics.CacheMisses)

		if finalBaseMetrics.CacheHits+finalBaseMetrics.CacheMisses > 0 {
			hitRatio := float64(finalBaseMetrics.CacheHits) / float64(finalBaseMetrics.CacheHits+finalBaseMetrics.CacheMisses)
			fmt.Printf("Cache hit ratio: %.2f%%\n", hitRatio*100)
		}

		Expect(requestsProcessed).To(BeNumerically(">=", numRequests))
	})
})
