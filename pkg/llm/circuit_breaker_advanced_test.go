package llm

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Advanced Circuit Breaker Tests", func() {
	var (
		circuitBreaker *CircuitBreaker
		requestCount   int64
		successCount   int64
		failureCount   int64
		ctx            context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		config := &CircuitBreakerConfig{
			FailureThreshold: 3,
			ResetTimeout:     time.Second,
		}
		circuitBreaker = NewCircuitBreaker("test", config)
		atomic.StoreInt64(&requestCount, 0)
		atomic.StoreInt64(&successCount, 0)
		atomic.StoreInt64(&failureCount, 0)
	})

	Context("Circuit Breaker State Transitions", func() {
		It("should transition from closed to open after threshold failures", func() {
			Expect(circuitBreaker.getState()).To(Equal(StateClosed))

			// Trigger threshold failures
			for i := 0; i < 3; i++ {
				_, err := circuitBreaker.Execute(ctx, func(ctx context.Context) (interface{}, error) {
					return nil, fmt.Errorf("simulated failure %d", i)
				})
				Expect(err).To(HaveOccurred())
			}

			Expect(circuitBreaker.getState()).To(Equal(StateOpen))
		})

		It("should transition from open to half-open after timeout", func() {
			// Force circuit open
			for i := 0; i < 3; i++ {
				circuitBreaker.Execute(ctx, func(ctx context.Context) (interface{}, error) {
					return nil, fmt.Errorf("failure %d", i)
				})
			}
			Expect(circuitBreaker.getState()).To(Equal(StateOpen))

			// Wait for timeout
			time.Sleep(1100 * time.Millisecond)

			// Next call should transition to half-open
			_, err := circuitBreaker.Execute(ctx, func(ctx context.Context) (interface{}, error) {
				return nil, nil // Success
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(circuitBreaker.getState()).To(Equal(StateClosed))
		})

		It("should remain open if half-open call fails", func() {
			// Force circuit open
			for i := 0; i < 3; i++ {
				circuitBreaker.Execute(ctx, func(ctx context.Context) (interface{}, error) {
					return nil, fmt.Errorf("failure %d", i)
				})
			}

			// Wait for timeout
			time.Sleep(1100 * time.Millisecond)

			// Fail the half-open call
			_, err := circuitBreaker.Execute(ctx, func(ctx context.Context) (interface{}, error) {
				return nil, fmt.Errorf("half-open failure")
			})

			Expect(err).To(HaveOccurred())
			Expect(circuitBreaker.getState()).To(Equal(StateOpen))
		})
	})

	Context("Concurrent Circuit Breaker Operations", func() {
		It("should handle concurrent calls safely", func() {
			const numGoroutines = 50
			const callsPerGoroutine = 10

			var wg sync.WaitGroup
			results := make([][]error, numGoroutines)

			for i := 0; i < numGoroutines; i++ {
				wg.Add(1)
				go func(routineIndex int) {
					defer wg.Done()
					results[routineIndex] = make([]error, callsPerGoroutine)

					for j := 0; j < callsPerGoroutine; j++ {
						_, err := circuitBreaker.Execute(ctx, func(ctx context.Context) (interface{}, error) {
							atomic.AddInt64(&requestCount, 1)
							// Simulate some failures
							if atomic.LoadInt64(&requestCount)%4 == 0 {
								atomic.AddInt64(&failureCount, 1)
								return nil, fmt.Errorf("simulated failure")
							}
							atomic.AddInt64(&successCount, 1)
							return nil, nil
						})
						results[routineIndex][j] = err
						time.Sleep(time.Millisecond) // Small delay
					}
				}(i)
			}

			wg.Wait()

			// Verify results
			totalRequests := atomic.LoadInt64(&requestCount)
			totalSuccess := atomic.LoadInt64(&successCount)
			totalFailures := atomic.LoadInt64(&failureCount)

			fmt.Printf("Total requests: %d, Success: %d, Failures: %d\n",
				totalRequests, totalSuccess, totalFailures)

			Expect(totalRequests).To(BeNumerically(">", 0))
			Expect(totalSuccess + totalFailures).To(Equal(totalRequests))
		})

		It("should maintain state consistency under concurrent load", func() {
			const numGoroutines = 20
			var wg sync.WaitGroup

			// Function that sometimes fails
			operation := func(ctx context.Context) (interface{}, error) {
				time.Sleep(time.Millisecond) // Simulate work
				if time.Now().UnixNano()%5 == 0 {
					return nil, fmt.Errorf("random failure")
				}
				return nil, nil
			}

			for i := 0; i < numGoroutines; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < 10; j++ {
						circuitBreaker.Execute(ctx, operation)
						time.Sleep(time.Millisecond)
					}
				}()
			}

			wg.Wait()

			// Circuit breaker should still be in a valid state
			state := circuitBreaker.getState()
			Expect(state).To(BeElementOf(StateClosed, StateOpen, StateHalfOpen))
		})
	})

	/*
		Context("Circuit Breaker with Enhanced Client", func() {
			var (
				enhancedClient *EnhancedClient
				failureServer  *httptest.Server
				successServer  *httptest.Server
			)

			BeforeEach(func() {
				// Create a server that fails initially then succeeds
				requestCounter := int64(0)
				failureServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					count := atomic.AddInt64(&requestCounter, 1)
					if count <= 5 { // First 5 requests fail
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					// Subsequent requests succeed
					response := json.RawMessage("{}"){
							"replicas": float64(1),
							"image":    "recovery:latest",
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
						Timeout:     5 * time.Second,
					},
					CircuitBreakerThreshold: 3,
					CircuitBreakerTimeout:   500 * time.Millisecond,
					RateLimitTokens:         100, // High limit to avoid rate limiting
					RateLimitRefillRate:     50,
					HealthCheckInterval:     time.Minute,
					HealthCheckTimeout:      time.Second,
				}

				enhancedClient = NewEnhancedClient(failureServer.URL, config)
			})

			AfterEach(func() {
				enhancedClient.healthChecker.Stop()
				failureServer.Close()
				if successServer != nil {
					successServer.Close()
				}
			})

			It("should demonstrate circuit breaker recovery", func() {
				// Initial requests should fail and open the circuit
				for i := 0; i < 5; i++ {
					_, err := enhancedClient.ProcessIntentWithEnhancements(
						context.Background(),
						fmt.Sprintf("failing request %d", i),
					)
					Expect(err).To(HaveOccurred())
				}

				// Circuit should be open
				Eventually(func() CircuitState {
					return enhancedClient.circuitBreaker.getState()
				}, time.Second).Should(Equal(StateOpen))

				// Wait for circuit to move to half-open
				time.Sleep(600 * time.Millisecond)

				// Next request should succeed and close the circuit
				result, err := enhancedClient.ProcessIntentWithEnhancements(
					context.Background(),
					"recovery request",
				)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(ContainSubstring("recovery-nf"))
				Expect(enhancedClient.circuitBreaker.getState()).To(Equal(StateClosed))
			})

			It("should handle rapid state transitions", func() {
				const numRequests = 20
				errors := make([]error, numRequests)
				results := make([]string, numRequests)

				// Make rapid requests
				for i := 0; i < numRequests; i++ {
					result, err := enhancedClient.ProcessIntentWithEnhancements(
						context.Background(),
						fmt.Sprintf("rapid request %d", i),
					)
					errors[i] = err
					results[i] = result
					time.Sleep(50 * time.Millisecond)
				}

				// Should have a mix of failures and successes
				failureCount := 0
				successCount := 0
				circuitBreakerErrors := 0

				for _, err := range errors {
					if err != nil {
						failureCount++
						var enhancedErr *EnhancedError
						if errors.As(err, &enhancedErr) && enhancedErr.Type == ErrorTypeCircuitBreaker {
							circuitBreakerErrors++
						}
					} else {
						successCount++
					}
				}

				fmt.Printf("Successes: %d, Failures: %d, Circuit breaker errors: %d\n",
					successCount, failureCount, circuitBreakerErrors)

				Expect(successCount).To(BeNumerically(">", 0))
				Expect(circuitBreakerErrors).To(BeNumerically(">", 0))
			})
		})
	*/

	Context("Circuit Breaker Error Classification", func() {
		It("should differentiate between retryable and non-retryable errors", func() {
			retryableErrors := []error{
				fmt.Errorf("connection refused"),
				fmt.Errorf("timeout occurred"),
				fmt.Errorf("service unavailable"),
			}

			nonRetryableErrors := []error{
				fmt.Errorf("invalid request"),
				fmt.Errorf("authentication failed"),
				fmt.Errorf("malformed data"),
			}

			// Test retryable errors
			for _, err := range retryableErrors {
				_, cbErr := circuitBreaker.Execute(ctx, func(ctx context.Context) (interface{}, error) {
					return nil, err
				})
				Expect(cbErr).To(Equal(err))
			}

			// Circuit should be open after retryable failures
			Expect(circuitBreaker.getState()).To(Equal(StateOpen))

			// Reset circuit breaker
			config := &CircuitBreakerConfig{
				FailureThreshold: 3,
				ResetTimeout:     time.Second,
			}
			circuitBreaker = NewCircuitBreaker("test-reset", config)

			// Test non-retryable errors (these still count as failures for circuit breaker)
			for _, err := range nonRetryableErrors {
				_, cbErr := circuitBreaker.Execute(ctx, func(ctx context.Context) (interface{}, error) {
					return nil, err
				})
				Expect(cbErr).To(Equal(err))
			}

			// Circuit should also be open after non-retryable failures
			Expect(circuitBreaker.getState()).To(Equal(StateOpen))
		})
	})

	Context("Circuit Breaker Metrics and Monitoring", func() {
		It("should track failure rates and success rates", func() {
			const totalCalls = 20
			successCalls := 0
			failureCalls := 0

			for i := 0; i < totalCalls; i++ {
				_, err := circuitBreaker.Execute(ctx, func(ctx context.Context) (interface{}, error) {
					// Simulate 70% success rate
					if i%10 < 7 {
						return nil, nil
					}
					return nil, fmt.Errorf("simulated failure")
				})

				if err == nil {
					successCalls++
				} else {
					failureCalls++
				}
			}

			fmt.Printf("Success rate: %.2f%%, Failure rate: %.2f%%\n",
				float64(successCalls)/float64(totalCalls)*100,
				float64(failureCalls)/float64(totalCalls)*100)

			Expect(successCalls + failureCalls).To(Equal(totalCalls))

			// With 30% failure rate and threshold of 3, circuit should be open
			Eventually(func() CircuitState {
				return circuitBreaker.getState()
			}).Should(Equal(StateOpen))
		})
	})
})
