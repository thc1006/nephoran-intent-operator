package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Enhanced Client Edge Cases", func() {
	var (
		enhancedClient *EnhancedClient
		mockServer     *httptest.Server
		config         EnhancedClientConfig
	)

	BeforeEach(func() {
		// Create mock server with successful responses
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
				Timeout:     10 * time.Second, // Shorter timeout for edge case testing
			},
			CircuitBreakerThreshold: 3,
			CircuitBreakerTimeout:   time.Second,
			RateLimitTokens:         5,
			RateLimitRefillRate:     1,
			HealthCheckInterval:     100 * time.Millisecond,
			HealthCheckTimeout:      time.Second,
		}

		enhancedClient = NewEnhancedClient(mockServer.URL, config)
	})

	AfterEach(func() {
		enhancedClient.healthChecker.Stop()
		mockServer.Close()
	})

	Context("Concurrent Reconciliation Edge Cases", func() {
		It("should handle concurrent requests without race conditions", func() {
			const numConcurrentRequests = 50
			var wg sync.WaitGroup
			errors := make([]error, numConcurrentRequests)
			results := make([]string, numConcurrentRequests)

			// Launch concurrent requests
			for i := 0; i < numConcurrentRequests; i++ {
				wg.Add(1)
				go func(index int) {
					defer wg.Done()
					result, err := enhancedClient.ProcessIntentWithEnhancements(
						context.Background(),
						"concurrent test intent",
					)
					errors[index] = err
					results[index] = result
				}(i)
			}

			wg.Wait()

			// Check results
			successCount := 0
			for i := 0; i < numConcurrentRequests; i++ {
				if errors[i] == nil {
					successCount++
					Expect(results[i]).ToNot(BeEmpty())
				}
			}

			// Should have some successful requests (rate limiting may cause some failures)
			Expect(successCount).To(BeNumerically(">", 0))
		})

		It("should handle concurrent health checks safely", func() {
			const numHealthChecks = 20
			var wg sync.WaitGroup

			// Run concurrent health checks
			for i := 0; i < numHealthChecks; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					// Trigger health check manually
					enhancedClient.healthChecker.checkHealth()
				}()
			}

			wg.Wait()

			// Health check should still work after concurrent access
			health := enhancedClient.healthChecker.GetHealth()
			Expect(health).ToNot(BeEmpty())
		})

		It("should handle concurrent circuit breaker state changes", func() {
			// Create a failing server for this test
			failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			}))
			defer failingServer.Close()

			failingClient := NewEnhancedClient(failingServer.URL, config)
			defer failingClient.healthChecker.Stop()

			const numRequests = 10
			var wg sync.WaitGroup
			results := make([]error, numRequests)

			// Launch concurrent requests that will trigger circuit breaker
			for i := 0; i < numRequests; i++ {
				wg.Add(1)
				go func(index int) {
					defer wg.Done()
					_, err := failingClient.ProcessIntentWithEnhancements(
						context.Background(),
						"failing test intent",
					)
					results[index] = err
				}(i)
			}

			wg.Wait()

			// All requests should have errors
			for _, err := range results {
				Expect(err).To(HaveOccurred())
			}

			// Circuit breaker should be open
			Eventually(func() CircuitState {
				return failingClient.circuitBreaker.GetState()
			}, time.Second).Should(Equal(CircuitOpen))
		})
	})

	Context("Network Timeout Scenarios", func() {
		It("should handle request timeouts gracefully", func() {
			// Create a slow server that exceeds client timeout
			slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(15 * time.Second) // Longer than client timeout
				w.WriteHeader(http.StatusOK)
			}))
			defer slowServer.Close()

			slowClient := NewEnhancedClient(slowServer.URL, config)
			defer slowClient.healthChecker.Stop()

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			result, err := slowClient.ProcessIntentWithEnhancements(ctx, "timeout test")

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeEmpty())

			// Should be a timeout error
			var enhancedErr *EnhancedError
			Expect(errors.As(err, &enhancedErr)).To(BeTrue())
			Expect(enhancedErr.Type).To(Equal(ErrorTypeTimeout))
		})

		It("should handle connection refused errors", func() {
			// Use a non-existent server
			brokenClient := NewEnhancedClient("http://localhost:65535", config)
			defer brokenClient.healthChecker.Stop()

			result, err := brokenClient.ProcessIntentWithEnhancements(
				context.Background(),
				"connection refused test",
			)

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeEmpty())

			// Should be a network error
			var enhancedErr *EnhancedError
			Expect(errors.As(err, &enhancedErr)).To(BeTrue())
			Expect(enhancedErr.Type).To(Equal(ErrorTypeNetwork))
		})

		It("should handle network interruptions during requests", func() {
			// Create a server that closes connection mid-request
			unreliableServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Write partial response then close
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"partial":`))
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
				// Simulate connection close
				hj, _ := w.(http.Hijacker)
				conn, _, _ := hj.Hijack()
				conn.Close()
			}))
			defer unreliableServer.Close()

			unreliableClient := NewEnhancedClient(unreliableServer.URL, config)
			defer unreliableClient.healthChecker.Stop()

			result, err := unreliableClient.ProcessIntentWithEnhancements(
				context.Background(),
				"network interruption test",
			)

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeEmpty())
		})
	})

	Context("Resource Exhaustion Scenarios", func() {
		It("should handle memory pressure gracefully", func() {
			// Create requests with large payloads
			largeIntent := string(make([]byte, 10*1024*1024)) // 10MB intent

			result, err := enhancedClient.ProcessIntentWithEnhancements(
				context.Background(),
				largeIntent,
			)

			// Should either succeed or fail gracefully
			if err != nil {
				var enhancedErr *EnhancedError
				Expect(errors.As(err, &enhancedErr)).To(BeTrue())
			} else {
				Expect(result).ToNot(BeEmpty())
			}
		})

		It("should handle cache overflow", func() {
			cacheManager := NewCacheManager(time.Minute, 5, false, false, nil) // Small cache

			// Fill cache beyond capacity
			for i := 0; i < 10; i++ {
				key := fmt.Sprintf("key-%d", i)
				response := fmt.Sprintf("response-%d", i)
				cacheManager.SetWithTags(key, response, []string{"test"})
			}

			// Check that oldest items were evicted
			_, _, found := cacheManager.GetWithMetadata("key-0")
			Expect(found).To(BeFalse()) // Should be evicted

			// Recent items should still be there
			_, _, found = cacheManager.GetWithMetadata("key-9")
			Expect(found).To(BeTrue())
		})

		It("should handle rate limit exhaustion", func() {
			// Exhaust rate limit quickly
			const numRequests = 20
			errors := make([]error, numRequests)

			for i := 0; i < numRequests; i++ {
				_, err := enhancedClient.ProcessIntentWithEnhancements(
					context.Background(),
					fmt.Sprintf("rate limit test %d", i),
				)
				errors[i] = err
			}

			// Should have rate limit errors
			rateLimitErrors := 0
			for _, err := range errors {
				if err != nil {
					var enhancedErr *EnhancedError
					if errors.As(err, &enhancedErr) && enhancedErr.Type == ErrorTypeRateLimit {
						rateLimitErrors++
					}
				}
			}

			Expect(rateLimitErrors).To(BeNumerically(">", 0))
		})
	})

	Context("Large Payload Processing", func() {
		It("should handle very large responses", func() {
			// Create server that returns large responses
			largeResponseServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				largeSpec := make(map[string]interface{})
				largeSpec["replicas"] = float64(1)
				largeSpec["image"] = "test:latest"

				// Add large configuration
				largeConfig := make([]string, 1000)
				for i := 0; i < 1000; i++ {
					largeConfig[i] = fmt.Sprintf("config-item-%d", i)
				}
				largeSpec["large_config"] = largeConfig

				response := map[string]interface{}{
					"type":      "NetworkFunctionDeployment",
					"name":      "large-nf",
					"namespace": "default",
					"spec":      largeSpec,
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer largeResponseServer.Close()

			largeClient := NewEnhancedClient(largeResponseServer.URL, config)
			defer largeClient.healthChecker.Stop()

			result, err := largeClient.ProcessIntentWithEnhancements(
				context.Background(),
				"large response test",
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeEmpty())
			Expect(len(result)).To(BeNumerically(">", 1000)) // Should be a large response
		})

		It("should handle malformed JSON responses", func() {
			// Create server that returns malformed JSON
			malformedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"type": "NetworkFunctionDeployment", "malformed": `)) // Incomplete JSON
			}))
			defer malformedServer.Close()

			malformedClient := NewEnhancedClient(malformedServer.URL, config)
			defer malformedClient.healthChecker.Stop()

			result, err := malformedClient.ProcessIntentWithEnhancements(
				context.Background(),
				"malformed JSON test",
			)

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeEmpty())
		})

		It("should handle empty responses", func() {
			// Create server that returns empty responses
			emptyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				// Don't write any content
			}))
			defer emptyServer.Close()

			emptyClient := NewEnhancedClient(emptyServer.URL, config)
			defer emptyClient.healthChecker.Stop()

			result, err := emptyClient.ProcessIntentWithEnhancements(
				context.Background(),
				"empty response test",
			)

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeEmpty())
		})
	})

	Context("Async Processing Edge Cases", func() {
		It("should handle async cancellation", func() {
			ctx, cancel := context.WithCancel(context.Background())

			// Start async processing
			resultChan := enhancedClient.ProcessIntentWithEnhancementsAsync(ctx, "async cancel test")

			// Cancel immediately
			cancel()

			// Should complete (might succeed or fail due to cancellation)
			select {
			case result := <-resultChan:
				// Either succeeded before cancellation or failed due to cancellation
				if result.Error != nil {
					var enhancedErr *EnhancedError
					if errors.As(result.Error, &enhancedErr) {
						Expect(enhancedErr.Type).To(Or(Equal(ErrorTypeTimeout), Equal(ErrorTypeLLM)))
					}
				}
			case <-time.After(5 * time.Second):
				Fail("Async processing should have completed")
			}
		})

		It("should handle multiple concurrent async operations", func() {
			const numAsync = 10
			resultChans := make([]<-chan AsyncProcessingResult, numAsync)

			// Start multiple async operations
			for i := 0; i < numAsync; i++ {
				resultChans[i] = enhancedClient.ProcessIntentWithEnhancementsAsync(
					context.Background(),
					fmt.Sprintf("async concurrent test %d", i),
				)
			}

			// Collect all results
			successCount := 0
			for i := 0; i < numAsync; i++ {
				select {
				case result := <-resultChans[i]:
					if result.Error == nil {
						successCount++
						Expect(result.Result).ToNot(BeEmpty())
					}
				case <-time.After(10 * time.Second):
					Fail(fmt.Sprintf("Async operation %d timed out", i))
				}
			}

			// Should have some successful operations
			Expect(successCount).To(BeNumerically(">", 0))
		})
	})

	Context("Cache Adaptive TTL Edge Cases", func() {
		It("should adjust TTL based on access patterns", func() {
			cacheManager := NewCacheManager(time.Minute, 100, false, false, nil)

			key := "adaptive-ttl-test"
			response := "test response"

			// Set initial item
			cacheManager.SetWithTags(key, response, []string{"test"})

			// Access it multiple times to increase access count
			for i := 0; i < 15; i++ {
				_, _, found := cacheManager.GetWithMetadata(key)
				Expect(found).To(BeTrue())
				time.Sleep(10 * time.Millisecond) // Small delay between accesses
			}

			// Calculate adaptive TTL - should be longer due to high access count
			adaptiveTTL := cacheManager.calculateAdaptiveTTL(key)
			Expect(adaptiveTTL).To(BeNumerically(">", time.Minute))
		})

		It("should handle cache statistics overflow", func() {
			cacheManager := NewCacheManager(time.Minute, 100, false, false, nil)

			// Create many cache entries to test stats management
			for i := 0; i < 200; i++ {
				key := fmt.Sprintf("overflow-test-%d", i)
				response := fmt.Sprintf("response-%d", i)
				cacheManager.SetWithTags(key, response, []string{"test"})

				// Access some entries multiple times
				if i < 50 {
					cacheManager.GetWithMetadata(key)
				}
			}

			// Should handle overflow gracefully (evicting least used items)
			Expect(cacheManager.currentSize).To(BeNumerically("<=", 100))
		})
	})
})
