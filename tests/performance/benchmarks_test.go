// Package performance provides comprehensive performance testing and benchmarking
package performance_tests

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nephranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
	"github.com/thc1006/nephoran-intent-operator/tests/framework"
)

// BenchmarkTestSuite provides comprehensive performance benchmarking
type BenchmarkTestSuite struct {
	*framework.TestSuite
}

// TestBenchmarks runs the benchmark test suite
func TestBenchmarks(t *testing.T) {
	suite.Run(t, &BenchmarkTestSuite{
		TestSuite: framework.NewTestSuite(&framework.TestConfig{
			LoadTestEnabled:  true,
			MaxConcurrency:   1000,
			TestDuration:     5 * time.Minute,
			ChaosTestEnabled: false,
			MockExternalAPIs: true,
		}),
	})
}

// BenchmarkIntentProcessing benchmarks intent processing performance
func BenchmarkIntentProcessing(b *testing.B) {
	testSuite := &BenchmarkTestSuite{
		TestSuite: framework.NewTestSuite(),
	}
	testSuite.SetupSuite()
	defer testSuite.TearDownSuite()

	controller := &controllers.NetworkIntentReconciler{
		Client: testSuite.GetK8sClient(),
		Scheme: testSuite.GetK8sClient().Scheme(),
	}

	intents := make([]*nephranv1.NetworkIntent, b.N)
	for i := 0; i < b.N; i++ {
		intents[i] = &nephranv1.NetworkIntent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("benchmark-intent-%d", i),
				Namespace: "default",
				Labels: map[string]string{
					"benchmark": "true",
				},
			},
			Spec: nephranv1.NetworkIntentSpec{
				Description: fmt.Sprintf("Benchmark intent %d for AMF deployment with 3 replicas", i),
				Priority:    "medium",
			},
		}
	}

	b.ResetTimer()
	b.StartTimer()

	// Benchmark parallel processing
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i >= len(intents) {
				i = 0
			}

			intent := intents[i]
			i++

			// Create intent
			err := testSuite.GetK8sClient().Create(context.Background(), intent)
			if err != nil {
				b.Errorf("Failed to create intent: %v", err)
				continue
			}

			// Process intent
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      intent.Name,
					Namespace: intent.Namespace,
				},
			}

			_, err = controller.Reconcile(context.Background(), req)
			if err != nil {
				b.Errorf("Failed to reconcile intent: %v", err)
			}
		}
	})

	b.StopTimer()
}

// BenchmarkLLMTokenManagement benchmarks token management operations
func BenchmarkLLMTokenManagement(b *testing.B) {
	tokenManager := llm.NewTokenManager()

	texts := []string{
		"Deploy AMF with 3 replicas in the telecom-core namespace for high availability",
		"Configure SMF for session management with PFCP support and N4 interface",
		"Setup UPF with user plane functionality for 5G SA network deployment",
		"Implement network slicing for eMBB, URLLC, and mMTC use cases",
		"Configure gNodeB with RRC, PDCP, RLC, and MAC layer implementations",
	}

	b.ResetTimer()

	b.Run("TokenCounting", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				if i >= len(texts) {
					i = 0
				}
				tokenManager.CountTokens(texts[i])
				i++
			}
		})
	})

	b.Run("BudgetCalculation", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := tokenManager.CalculateTokenBudget(
					context.Background(),
					"gpt-4o-mini",
					"System prompt for network automation",
					"Deploy network function",
					"Context from knowledge base...",
				)
				if err != nil {
					b.Errorf("Budget calculation failed: %v", err)
				}
			}
		})
	})

	b.Run("ContextOptimization", func(b *testing.B) {
		contexts := []string{
			"AMF deployment procedures for 5G SA networks...",
			"SMF session management configuration details...",
			"UPF user plane functionality implementation...",
			"Network slicing architecture and deployment...",
			"O-RAN interface specifications and protocols...",
		}

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				tokenManager.OptimizeContext(contexts, 1000, "gpt-4o-mini")
			}
		})
	})
}

// BenchmarkContextBuilding benchmarks context building performance
func BenchmarkContextBuilding(b *testing.B) {
	config := &llm.ContextBuilderConfig{
		MaxContextTokens:   4000,
		DiversityThreshold: 0.7,
		QualityThreshold:   0.8,
	}
	contextBuilder := llm.NewContextBuilder(config)

	// Create test documents
	documents := make([]llm.Document, 1000)
	for i := range documents {
		documents[i] = llm.Document{
			ID:      fmt.Sprintf("doc%d", i),
			Title:   fmt.Sprintf("Telecom Document %d", i),
			Content: fmt.Sprintf("This is document %d about network functions, AMF, SMF, UPF deployment procedures and 5G SA network configuration details...", i),
			Source:  "3GPP TS 23.501",
			Metadata: map[string]interface{}{
				"category":  "5G Core",
				"authority": "3GPP",
				"recency":   0.9,
			},
		}
	}

	queries := []string{
		"Deploy AMF for 5G network",
		"Configure SMF session management",
		"Setup UPF user plane",
		"Implement network slicing",
		"O-RAN interface configuration",
	}

	b.ResetTimer()

	b.Run("RelevanceScoring", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			queryIndex := 0
			for pb.Next() {
				if queryIndex >= len(queries) {
					queryIndex = 0
				}

				_, err := contextBuilder.CalculateRelevanceScores(
					context.Background(),
					queries[queryIndex],
					documents[:100], // Limit to 100 docs for benchmark
				)
				if err != nil {
					b.Errorf("Relevance scoring failed: %v", err)
				}
				queryIndex++
			}
		})
	})

	b.Run("ContextAssembly", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			queryIndex := 0
			for pb.Next() {
				if queryIndex >= len(queries) {
					queryIndex = 0
				}

				_, err := contextBuilder.BuildContext(
					context.Background(),
					queries[queryIndex],
					documents[:50], // Limit for performance
				)
				if err != nil {
					b.Errorf("Context building failed: %v", err)
				}
				queryIndex++
			}
		})
	})
}

// BenchmarkCircuitBreaker benchmarks circuit breaker performance
func BenchmarkCircuitBreaker(b *testing.B) {
	config := &llm.CircuitBreakerConfig{
		FailureThreshold:    5,
		FailureRate:         0.5,
		MinimumRequestCount: 10,
		Timeout:             30 * time.Second,
		ResetTimeout:        60 * time.Second,
		SuccessThreshold:    3,
	}

	circuitBreaker := llm.NewCircuitBreaker("benchmark-breaker", config)

	b.ResetTimer()

	b.Run("SuccessfulRequests", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				err := circuitBreaker.Execute(func() error {
					// Simulate successful operation
					time.Sleep(1 * time.Millisecond)
					return nil
				})
				if err != nil {
					b.Errorf("Circuit breaker execution failed: %v", err)
				}
			}
		})
	})

	b.Run("MixedRequests", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				circuitBreaker.Execute(func() error {
					if i%10 < 2 { // 20% failure rate
						return fmt.Errorf("simulated failure")
					}
					time.Sleep(1 * time.Millisecond)
					return nil
				})
				i++
			}
		})
	})
}

// TestLoadTesting performs comprehensive load testing
func (suite *BenchmarkTestSuite) TestLoadTesting() {
	ginkgo.Describe("Load Testing", func() {
		ginkgo.Context("Intent Processing Load", func() {
			ginkgo.It("should handle 1000 concurrent intents", func() {
				if !suite.GetConfig().LoadTestEnabled {
					ginkgo.Skip("Load testing disabled")
				}

				controller := &controllers.NetworkIntentReconciler{
					Client: suite.GetK8sClient(),
					Scheme: suite.GetK8sClient().Scheme(),
				}

				intentCounter := int64(0)

				err := suite.RunLoadTest(func() error {
					intentNum := atomic.AddInt64(&intentCounter, 1)

					intent := &nephranv1.NetworkIntent{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("load-test-intent-%d", intentNum),
							Namespace: "default",
						},
						Spec: nephranv1.NetworkIntentSpec{
							Description: fmt.Sprintf("Load test intent %d for performance testing", intentNum),
							Priority:    "medium",
						},
					}

					// Create intent
					err := suite.GetK8sClient().Create(context.Background(), intent)
					if err != nil {
						return err
					}

					// Process intent
					req := reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      intent.Name,
							Namespace: intent.Namespace,
						},
					}

					_, err = controller.Reconcile(context.Background(), req)
					return err
				})

				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Verify load test results
				loadResults := suite.GetMetrics().loadTestResults
				gomega.Expect(len(loadResults)).To(gomega.BeNumerically(">", 0))

				for _, result := range loadResults {
					fmt.Printf("Load Test: %s\n", result.TestName)
					fmt.Printf("  Total Requests: %d\n", result.TotalRequests)
					fmt.Printf("  Successful: %d\n", result.SuccessfulReq)
					fmt.Printf("  Failed: %d\n", result.FailedReq)
					fmt.Printf("  Error Rate: %.2f%%\n", result.ErrorRate*100)
					fmt.Printf("  Throughput: %.2f req/s\n", result.Throughput)
					fmt.Printf("  Avg Latency: %v\n", result.AvgLatency)
					fmt.Printf("  P95 Latency: %v\n", result.P95Latency)
					fmt.Printf("  P99 Latency: %v\n", result.P99Latency)

					// Performance assertions
					gomega.Expect(result.ErrorRate).To(gomega.BeNumerically("<", 0.05))           // <5% error rate
					gomega.Expect(result.Throughput).To(gomega.BeNumerically(">", 10))            // >10 req/s
					gomega.Expect(result.P95Latency).To(gomega.BeNumerically("<", 5*time.Second)) // P95 < 5s
				}
			})
		})

		ginkgo.Context("Memory Pressure Testing", func() {
			ginkgo.It("should handle memory pressure gracefully", func() {
				if !suite.GetConfig().LoadTestEnabled {
					ginkgo.Skip("Load testing disabled")
				}

				// Monitor memory before test
				var initialMem runtime.MemStats
				runtime.ReadMemStats(&initialMem)

				// Create memory pressure by processing large contexts
				largeContexts := make([]string, 1000)
				for i := range largeContexts {
					largeContexts[i] = generateLargeContext(10000) // 10KB each
				}

				contextBuilder := llm.NewContextBuilder(&llm.ContextBuilderConfig{
					MaxContextTokens: 8000,
				})

				documents := make([]llm.Document, 100)
				for i := range documents {
					documents[i] = llm.Document{
						ID:      fmt.Sprintf("doc%d", i),
						Content: largeContexts[i%len(largeContexts)],
						Source:  "Memory Test",
					}
				}

				// Process contexts under memory pressure
				for i := 0; i < 100; i++ {
					_, err := contextBuilder.BuildContext(
						context.Background(),
						"memory pressure test query",
						documents,
					)
					gomega.Expect(err).NotTo(gomega.HaveOccurred())
				}

				// Monitor memory after test
				var finalMem runtime.MemStats
				runtime.ReadMemStats(&finalMem)
				runtime.GC() // Force garbage collection

				var afterGCMem runtime.MemStats
				runtime.ReadMemStats(&afterGCMem)

				// Memory should be manageable
				memoryGrowth := afterGCMem.Alloc - initialMem.Alloc
				fmt.Printf("Memory growth: %d bytes\n", memoryGrowth)

				// Should not exceed 100MB growth
				gomega.Expect(memoryGrowth).To(gomega.BeNumerically("<", 100*1024*1024))
			})
		})

		ginkgo.Context("Concurrent Processing", func() {
			ginkgo.It("should maintain performance under high concurrency", func() {
				if !suite.GetConfig().LoadTestEnabled {
					ginkgo.Skip("Load testing disabled")
				}

				const numGoroutines = 100
				const requestsPerGoroutine = 10

				tokenManager := llm.NewTokenManager()

				var wg sync.WaitGroup
				latencies := make(chan time.Duration, numGoroutines*requestsPerGoroutine)

				wg.Add(numGoroutines)

				for i := 0; i < numGoroutines; i++ {
					go func(goroutineID int) {
						defer wg.Done()

						for j := 0; j < requestsPerGoroutine; j++ {
							start := time.Now()

							text := fmt.Sprintf("Goroutine %d request %d: Deploy network function with specific configuration", goroutineID, j)
							tokenManager.CountTokens(text)

							latency := time.Since(start)
							latencies <- latency
						}
					}(i)
				}

				wg.Wait()
				close(latencies)

				// Analyze latencies
				var totalLatency time.Duration
				var maxLatency time.Duration
				count := 0

				for latency := range latencies {
					totalLatency += latency
					if latency > maxLatency {
						maxLatency = latency
					}
					count++
				}

				avgLatency := totalLatency / time.Duration(count)

				fmt.Printf("Concurrent Processing Results:\n")
				fmt.Printf("  Total Requests: %d\n", count)
				fmt.Printf("  Average Latency: %v\n", avgLatency)
				fmt.Printf("  Max Latency: %v\n", maxLatency)

				// Performance assertions
				gomega.Expect(avgLatency).To(gomega.BeNumerically("<", 10*time.Millisecond))
				gomega.Expect(maxLatency).To(gomega.BeNumerically("<", 100*time.Millisecond))
			})
		})
	})
}

// TestMemoryProfiling performs memory profiling and leak detection
func (suite *BenchmarkTestSuite) TestMemoryProfiling() {
	ginkgo.Describe("Memory Profiling", func() {
		ginkgo.It("should not have memory leaks in long-running operations", func() {
			// Baseline memory measurement
			runtime.GC()
			time.Sleep(100 * time.Millisecond)

			var baseline runtime.MemStats
			runtime.ReadMemStats(&baseline)

			// Perform many operations
			tokenManager := llm.NewTokenManager()

			for i := 0; i < 10000; i++ {
				text := fmt.Sprintf("Operation %d: Network function deployment with configuration", i)
				tokenManager.CountTokens(text)

				// Periodically force GC to detect leaks
				if i%1000 == 0 {
					runtime.GC()
				}
			}

			// Final measurement
			runtime.GC()
			time.Sleep(100 * time.Millisecond)

			var final runtime.MemStats
			runtime.ReadMemStats(&final)

			// Check for memory leaks
			memoryGrowth := final.Alloc - baseline.Alloc
			fmt.Printf("Memory growth after 10k operations: %d bytes\n", memoryGrowth)

			// Should not grow significantly (allow 1MB growth)
			gomega.Expect(memoryGrowth).To(gomega.BeNumerically("<", 1024*1024))

			// Check heap size
			heapGrowth := final.HeapAlloc - baseline.HeapAlloc
			fmt.Printf("Heap growth: %d bytes\n", heapGrowth)
			gomega.Expect(heapGrowth).To(gomega.BeNumerically("<", 2*1024*1024))
		})
	})
}

// Helper functions for benchmarking

// generateLargeContext creates a context string of specified size
func generateLargeContext(sizeBytes int) string {
	content := "This is telecom content for testing memory usage with network functions like AMF, SMF, UPF, and deployment procedures. "

	result := ""
	for len(result) < sizeBytes {
		result += content
	}

	return result[:sizeBytes]
}

// BenchmarkResult stores benchmark execution results
type BenchmarkResult struct {
	Name                string
	OperationsPerSec    float64
	AvgLatencyNs        int64
	P95LatencyNs        int64
	P99LatencyNs        int64
	AllocationsPerOp    int64
	BytesAllocatedPerOp int64
	MemoryFootprintMB   float64
}

// RunCustomBenchmark runs a custom benchmark and returns results
func RunCustomBenchmark(name string, benchFunc func() error, iterations int, concurrency int) *BenchmarkResult {
	latencies := make([]time.Duration, 0, iterations)

	var totalAllocs int64
	var totalBytes int64

	start := time.Now()

	if concurrency <= 1 {
		// Sequential execution
		for i := 0; i < iterations; i++ {
			var m1, m2 runtime.MemStats
			runtime.ReadMemStats(&m1)

			opStart := time.Now()
			err := benchFunc()
			latency := time.Since(opStart)

			runtime.ReadMemStats(&m2)

			if err != nil {
				continue
			}

			latencies = append(latencies, latency)
			totalAllocs += int64(m2.Mallocs - m1.Mallocs)
			totalBytes += int64(m2.TotalAlloc - m1.TotalAlloc)
		}
	} else {
		// Concurrent execution
		var wg sync.WaitGroup
		latencyChan := make(chan time.Duration, iterations)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for j := 0; j < iterations/concurrency; j++ {
					opStart := time.Now()
					err := benchFunc()
					latency := time.Since(opStart)

					if err == nil {
						latencyChan <- latency
					}
				}
			}()
		}

		wg.Wait()
		close(latencyChan)

		for latency := range latencyChan {
			latencies = append(latencies, latency)
		}
	}

	totalDuration := time.Since(start)

	// Calculate statistics
	if len(latencies) == 0 {
		return &BenchmarkResult{Name: name}
	}

	// Sort latencies for percentile calculation
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	var totalLatency time.Duration
	for _, lat := range latencies {
		totalLatency += lat
	}

	avgLatency := totalLatency / time.Duration(len(latencies))
	p95Index := int(float64(len(latencies)) * 0.95)
	p99Index := int(float64(len(latencies)) * 0.99)

	if p95Index >= len(latencies) {
		p95Index = len(latencies) - 1
	}
	if p99Index >= len(latencies) {
		p99Index = len(latencies) - 1
	}

	return &BenchmarkResult{
		Name:                name,
		OperationsPerSec:    float64(len(latencies)) / totalDuration.Seconds(),
		AvgLatencyNs:        avgLatency.Nanoseconds(),
		P95LatencyNs:        latencies[p95Index].Nanoseconds(),
		P99LatencyNs:        latencies[p99Index].Nanoseconds(),
		AllocationsPerOp:    totalAllocs / int64(len(latencies)),
		BytesAllocatedPerOp: totalBytes / int64(len(latencies)),
	}
}

var _ = ginkgo.Describe("Performance Benchmarks", func() {
	var testSuite *BenchmarkTestSuite

	ginkgo.BeforeEach(func() {
		testSuite = &BenchmarkTestSuite{
			TestSuite: framework.NewTestSuite(&framework.TestConfig{
				LoadTestEnabled: true,
				MaxConcurrency:  100,
				TestDuration:    2 * time.Minute,
			}),
		}
		testSuite.SetupSuite()
	})

	ginkgo.AfterEach(func() {
		testSuite.TearDownSuite()
	})

	ginkgo.Context("Load Testing", func() {
		testSuite.TestLoadTesting()
	})

	ginkgo.Context("Memory Profiling", func() {
		testSuite.TestMemoryProfiling()
	})
})
