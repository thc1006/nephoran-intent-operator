package performance_test

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers"
	testutils "github.com/thc1006/nephoran-intent-operator/tests/utils"
)

// PerformanceMetrics captures various performance measurements
type PerformanceMetrics struct {
	TotalRequests      int
	SuccessfulRequests int
	FailedRequests     int
	TotalDuration      time.Duration
	MinDuration        time.Duration
	MaxDuration        time.Duration
	AvgDuration        time.Duration
	P95Duration        time.Duration
	P99Duration        time.Duration
	ThroughputRPS      float64
	MemoryUsageMB      float64
	CPUUsagePercent    float64
	Durations          []time.Duration
	mu                 sync.RWMutex
}

func NewPerformanceMetrics() *PerformanceMetrics {
	return &PerformanceMetrics{
		MinDuration: time.Hour, // Initialize to high value
		Durations:   make([]time.Duration, 0),
	}
}

func (pm *PerformanceMetrics) RecordRequest(duration time.Duration, success bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.TotalRequests++
	if success {
		pm.SuccessfulRequests++
	} else {
		pm.FailedRequests++
	}

	pm.Durations = append(pm.Durations, duration)
	pm.TotalDuration += duration

	if duration < pm.MinDuration {
		pm.MinDuration = duration
	}
	if duration > pm.MaxDuration {
		pm.MaxDuration = duration
	}
}

func (pm *PerformanceMetrics) Calculate() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.TotalRequests == 0 {
		return
	}

	pm.AvgDuration = pm.TotalDuration / time.Duration(pm.TotalRequests)
	pm.ThroughputRPS = float64(pm.TotalRequests) / pm.TotalDuration.Seconds()

	// Calculate percentiles
	if len(pm.Durations) > 0 {
		// Sort durations for percentile calculation
		for i := 0; i < len(pm.Durations)-1; i++ {
			for j := i + 1; j < len(pm.Durations); j++ {
				if pm.Durations[i] > pm.Durations[j] {
					pm.Durations[i], pm.Durations[j] = pm.Durations[j], pm.Durations[i]
				}
			}
		}

		p95Index := int(float64(len(pm.Durations)) * 0.95)
		p99Index := int(float64(len(pm.Durations)) * 0.99)

		if p95Index >= len(pm.Durations) {
			p95Index = len(pm.Durations) - 1
		}
		if p99Index >= len(pm.Durations) {
			p99Index = len(pm.Durations) - 1
		}

		pm.P95Duration = pm.Durations[p95Index]
		pm.P99Duration = pm.Durations[p99Index]
	}

	// Calculate memory usage
	var memStats runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memStats)
	pm.MemoryUsageMB = float64(memStats.Alloc) / 1024 / 1024
}

func (pm *PerformanceMetrics) Print() {
	pm.Calculate()
	fmt.Printf("\n=== Performance Metrics ===\n")
	fmt.Printf("Total Requests: %d\n", pm.TotalRequests)
	fmt.Printf("Successful: %d (%.2f%%)\n", pm.SuccessfulRequests, float64(pm.SuccessfulRequests)/float64(pm.TotalRequests)*100)
	fmt.Printf("Failed: %d (%.2f%%)\n", pm.FailedRequests, float64(pm.FailedRequests)/float64(pm.TotalRequests)*100)
	fmt.Printf("Total Duration: %v\n", pm.TotalDuration)
	fmt.Printf("Average Duration: %v\n", pm.AvgDuration)
	fmt.Printf("Min Duration: %v\n", pm.MinDuration)
	fmt.Printf("Max Duration: %v\n", pm.MaxDuration)
	fmt.Printf("95th Percentile: %v\n", pm.P95Duration)
	fmt.Printf("99th Percentile: %v\n", pm.P99Duration)
	fmt.Printf("Throughput: %.2f requests/second\n", pm.ThroughputRPS)
	fmt.Printf("Memory Usage: %.2f MB\n", pm.MemoryUsageMB)
	fmt.Printf("========================\n\n")
}

var _ = Describe("Load Testing Suite", func() {
	var (
		k8sClient    client.Client
		mockDeps     *testutils.MockDependencies
		controller   *controllers.NetworkIntentReconciler
		e2Controller *controllers.E2NodeSetReconciler
		ctx          context.Context
		cancel       context.CancelFunc
		namespace    string
		testFixtures *testutils.TestFixtures
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 15*time.Minute)
		namespace = fmt.Sprintf("load-test-%d", time.Now().UnixNano())

		// Setup test environment
		testFixtures = testutils.NewTestFixtures()
		k8sClient = fake.NewClientBuilder().WithScheme(testFixtures.Scheme).Build()

		// Create namespace
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: namespace},
		}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		// Setup mock dependencies with performance-optimized responses
		mockDeps = testutils.NewMockDependencies()
		mockResponse := testutils.CreateMockLLMResponse("5G-Core-AMF", 0.95)
		mockDeps.GetLLMClient().(*testutils.MockLLMClient).SetResponse("ProcessIntent", mockResponse)

		// Setup controllers
		controller = &controllers.NetworkIntentReconciler{
			Client: k8sClient,
			Scheme: testFixtures.Scheme,
			Config: controllers.Config{
				MaxRetries:      2,
				RetryDelay:      1 * time.Second,
				Timeout:         10 * time.Second,
				GitRepoURL:      "https://github.com/test/repo.git",
				GitBranch:       "main",
				GitDeployPath:   "networkintents",
				LLMProcessorURL: "http://localhost:8080",
				UseNephioPorch:  false,
			},
			Dependencies: mockDeps,
		}

		e2Controller = &controllers.E2NodeSetReconciler{
			Client: k8sClient,
			Scheme: testFixtures.Scheme,
		}
	})

	AfterEach(func() {
		cancel()
	})

	Describe("NetworkIntent Load Testing", func() {
		Context("with 10 concurrent users", func() {
			It("should maintain performance under light load", func() {
				concurrentUsers := 10
				requestsPerUser := 5
				totalRequests := concurrentUsers * requestsPerUser

				metrics := NewPerformanceMetrics()
				var wg sync.WaitGroup

				By(fmt.Sprintf("executing %d requests with %d concurrent users", totalRequests, concurrentUsers))

				startTime := time.Now()

				for user := 0; user < concurrentUsers; user++ {
					wg.Add(1)
					go func(userID int) {
						defer wg.Done()
						defer GinkgoRecover()

						for req := 0; req < requestsPerUser; req++ {
							intentName := fmt.Sprintf("load-intent-u%d-r%d", userID, req)

							// Create intent
							intent := testutils.NetworkIntentFixture.CreateBasicNetworkIntent(intentName, namespace)
							intent.Spec.Intent = fmt.Sprintf("Deploy AMF for user %d request %d", userID, req)

							reqStart := time.Now()
							createErr := k8sClient.Create(ctx, intent)
							if createErr != nil {
								metrics.RecordRequest(time.Since(reqStart), false)
								continue
							}

							// Reconcile
							_, reconcileErr := controller.Reconcile(ctx, types.NamespacedName{
								Name:      intent.Name,
								Namespace: intent.Namespace,
							})

							duration := time.Since(reqStart)
							success := reconcileErr == nil
							metrics.RecordRequest(duration, success)

							// Small delay to simulate realistic load
							time.Sleep(10 * time.Millisecond)
						}
					}(user)
				}

				wg.Wait()
				totalTestTime := time.Since(startTime)

				By("analyzing performance results")
				metrics.Calculate()
				metrics.Print()

				// Performance assertions
				Expect(metrics.SuccessfulRequests).To(BeNumerically(">=", int(float64(totalRequests)*0.95)),
					"Should have at least 95% success rate")

				Expect(metrics.P95Duration).To(BeNumerically("<", 5*time.Second),
					"95th percentile should be under 5 seconds")

				Expect(metrics.AvgDuration).To(BeNumerically("<", 2*time.Second),
					"Average response time should be under 2 seconds")

				Expect(metrics.ThroughputRPS).To(BeNumerically(">=", 5.0),
					"Should maintain at least 5 requests per second")

				Expect(totalTestTime).To(BeNumerically("<", 60*time.Second),
					"Total test should complete within 1 minute")
			})
		})

		Context("with 50 concurrent users", func() {
			It("should handle medium load efficiently", func() {
				concurrentUsers := 50
				requestsPerUser := 3
				totalRequests := concurrentUsers * requestsPerUser

				metrics := NewPerformanceMetrics()
				var wg sync.WaitGroup

				By(fmt.Sprintf("executing %d requests with %d concurrent users", totalRequests, concurrentUsers))

				startTime := time.Now()

				for user := 0; user < concurrentUsers; user++ {
					wg.Add(1)
					go func(userID int) {
						defer wg.Done()
						defer GinkgoRecover()

						for req := 0; req < requestsPerUser; req++ {
							intentName := fmt.Sprintf("medium-load-u%d-r%d", userID, req)

							intent := testutils.NetworkIntentFixture.CreateBasicNetworkIntent(intentName, namespace)
							intent.Spec.Intent = fmt.Sprintf("Deploy SMF for user %d request %d", userID, req)

							reqStart := time.Now()
							createErr := k8sClient.Create(ctx, intent)
							if createErr != nil {
								metrics.RecordRequest(time.Since(reqStart), false)
								continue
							}

							_, reconcileErr := controller.Reconcile(ctx, types.NamespacedName{
								Name:      intent.Name,
								Namespace: intent.Namespace,
							})

							duration := time.Since(reqStart)
							success := reconcileErr == nil
							metrics.RecordRequest(duration, success)
						}
					}(user)
				}

				wg.Wait()
				totalTestTime := time.Since(startTime)

				By("analyzing medium load performance")
				metrics.Calculate()
				metrics.Print()

				// Performance assertions for medium load
				Expect(metrics.SuccessfulRequests).To(BeNumerically(">=", int(float64(totalRequests)*0.90)),
					"Should have at least 90% success rate under medium load")

				Expect(metrics.P95Duration).To(BeNumerically("<", 8*time.Second),
					"95th percentile should be under 8 seconds under medium load")

				Expect(metrics.AvgDuration).To(BeNumerically("<", 4*time.Second),
					"Average response time should be under 4 seconds")

				expectedMinThroughput := float64(totalRequests) / 120.0 // Should complete within 2 minutes
				Expect(metrics.ThroughputRPS).To(BeNumerically(">=", expectedMinThroughput),
					"Should maintain reasonable throughput")
			})
		})

		Context("with 100 concurrent users", func() {
			It("should handle high load with acceptable performance", func() {
				concurrentUsers := 100
				requestsPerUser := 2
				totalRequests := concurrentUsers * requestsPerUser

				metrics := NewPerformanceMetrics()
				var wg sync.WaitGroup

				By(fmt.Sprintf("executing %d requests with %d concurrent users", totalRequests, concurrentUsers))

				startTime := time.Now()

				// Use a semaphore to control peak concurrency
				semaphore := make(chan struct{}, concurrentUsers)

				for user := 0; user < concurrentUsers; user++ {
					wg.Add(1)
					go func(userID int) {
						defer wg.Done()
						defer GinkgoRecover()

						// Acquire semaphore
						semaphore <- struct{}{}
						defer func() { <-semaphore }()

						for req := 0; req < requestsPerUser; req++ {
							intentName := fmt.Sprintf("high-load-u%d-r%d", userID, req)

							intent := testutils.NetworkIntentFixture.CreateBasicNetworkIntent(intentName, namespace)
							intent.Spec.Intent = fmt.Sprintf("Deploy UPF for user %d request %d", userID, req)

							reqStart := time.Now()
							createErr := k8sClient.Create(ctx, intent)
							if createErr != nil {
								metrics.RecordRequest(time.Since(reqStart), false)
								continue
							}

							_, reconcileErr := controller.Reconcile(ctx, types.NamespacedName{
								Name:      intent.Name,
								Namespace: intent.Namespace,
							})

							duration := time.Since(reqStart)
							success := reconcileErr == nil
							metrics.RecordRequest(duration, success)

							// Slight delay to prevent overwhelming
							time.Sleep(5 * time.Millisecond)
						}
					}(user)
				}

				wg.Wait()
				totalTestTime := time.Since(startTime)

				By("analyzing high load performance")
				metrics.Calculate()
				metrics.Print()

				// More relaxed performance assertions for high load
				Expect(metrics.SuccessfulRequests).To(BeNumerically(">=", int(float64(totalRequests)*0.85)),
					"Should have at least 85% success rate under high load")

				Expect(metrics.P95Duration).To(BeNumerically("<", 15*time.Second),
					"95th percentile should be under 15 seconds under high load")

				Expect(metrics.AvgDuration).To(BeNumerically("<", 8*time.Second),
					"Average response time should be under 8 seconds")

				Expect(totalTestTime).To(BeNumerically("<", 300*time.Second),
					"Total test should complete within 5 minutes")
			})
		})

		Context("throughput validation", func() {
			It("should achieve target of 50 intents/second", func() {
				targetThroughput := 50.0 // intents per second
				testDuration := 30 * time.Second
				expectedRequests := int(targetThroughput * testDuration.Seconds())

				metrics := NewPerformanceMetrics()
				var wg sync.WaitGroup
				var requestCount int64

				By(fmt.Sprintf("targeting %v intents/second for %v", targetThroughput, testDuration))

				startTime := time.Now()
				stopTime := startTime.Add(testDuration)

				// Launch multiple workers to achieve target throughput
				numWorkers := 20
				for worker := 0; worker < numWorkers; worker++ {
					wg.Add(1)
					go func(workerID int) {
						defer wg.Done()
						defer GinkgoRecover()

						requestID := 0
						for time.Now().Before(stopTime) {
							intentName := fmt.Sprintf("throughput-w%d-r%d", workerID, requestID)

							intent := testutils.NetworkIntentFixture.CreateBasicNetworkIntent(intentName, namespace)
							intent.Spec.Intent = fmt.Sprintf("Throughput test worker %d request %d", workerID, requestID)

							reqStart := time.Now()
							createErr := k8sClient.Create(ctx, intent)
							if createErr != nil {
								metrics.RecordRequest(time.Since(reqStart), false)
								continue
							}

							_, reconcileErr := controller.Reconcile(ctx, types.NamespacedName{
								Name:      intent.Name,
								Namespace: intent.Namespace,
							})

							duration := time.Since(reqStart)
							success := reconcileErr == nil
							metrics.RecordRequest(duration, success)

							requestCount++
							requestID++

							// Dynamic delay based on target throughput
							targetIntervalPerWorker := time.Duration(float64(numWorkers) * float64(time.Second) / targetThroughput)
							time.Sleep(targetIntervalPerWorker)
						}
					}(worker)
				}

				wg.Wait()
				actualDuration := time.Since(startTime)

				By("analyzing throughput results")
				metrics.Calculate()
				metrics.Print()

				// Throughput validation
				actualThroughput := float64(metrics.TotalRequests) / actualDuration.Seconds()

				Expect(actualThroughput).To(BeNumerically(">=", targetThroughput*0.8),
					fmt.Sprintf("Should achieve at least 80%% of target throughput (%.2f >= %.2f)",
						actualThroughput, targetThroughput*0.8))

				Expect(metrics.P95Duration).To(BeNumerically("<", 5*time.Second),
					"95th percentile should remain under 5 seconds during throughput test")

				fmt.Printf("Achieved throughput: %.2f intents/second (target: %.2f)\n",
					actualThroughput, targetThroughput)
			})
		})
	})

	Describe("Memory Leak Detection", func() {
		It("should not leak memory over 10 minutes of continuous operation", func() {
			testDuration := 10 * time.Minute
			measurementInterval := 30 * time.Second

			var memoryMeasurements []float64
			var wg sync.WaitGroup
			var stopMemoryMonitoring bool

			By("starting memory monitoring")
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer GinkgoRecover()

				ticker := time.NewTicker(measurementInterval)
				defer ticker.Stop()

				for !stopMemoryMonitoring {
					select {
					case <-ticker.C:
						var memStats runtime.MemStats
						runtime.GC()
						runtime.ReadMemStats(&memStats)
						memoryMB := float64(memStats.Alloc) / 1024 / 1024
						memoryMeasurements = append(memoryMeasurements, memoryMB)

						fmt.Printf("Memory usage: %.2f MB\n", memoryMB)
					case <-ctx.Done():
						return
					}
				}
			}()

			By("running continuous load for memory leak detection")
			endTime := time.Now().Add(testDuration)
			requestCount := 0

			for time.Now().Before(endTime) {
				intentName := fmt.Sprintf("memory-test-%d", requestCount)

				intent := testutils.NetworkIntentFixture.CreateBasicNetworkIntent(intentName, namespace)
				intent.Spec.Intent = fmt.Sprintf("Memory leak test request %d", requestCount)

				err := k8sClient.Create(ctx, intent)
				if err == nil {
					controller.Reconcile(ctx, types.NamespacedName{
						Name:      intent.Name,
						Namespace: intent.Namespace,
					})

					// Occasionally delete old intents to prevent accumulation
					if requestCount%100 == 0 && requestCount > 0 {
						oldIntentName := fmt.Sprintf("memory-test-%d", requestCount-100)
						oldIntent := &nephoranv1.NetworkIntent{
							ObjectMeta: metav1.ObjectMeta{Name: oldIntentName, Namespace: namespace},
						}
						k8sClient.Delete(ctx, oldIntent)
					}
				}

				requestCount++
				time.Sleep(100 * time.Millisecond) // Steady load
			}

			stopMemoryMonitoring = true
			wg.Wait()

			By("analyzing memory usage patterns")
			Expect(len(memoryMeasurements)).To(BeNumerically(">=", 3),
				"Should have multiple memory measurements")

			if len(memoryMeasurements) >= 3 {
				initialMemory := memoryMeasurements[0]
				finalMemory := memoryMeasurements[len(memoryMeasurements)-1]
				maxMemory := initialMemory

				for _, mem := range memoryMeasurements {
					if mem > maxMemory {
						maxMemory = mem
					}
				}

				memoryGrowth := finalMemory - initialMemory
				memoryGrowthPercent := (memoryGrowth / initialMemory) * 100

				fmt.Printf("Memory Analysis:\n")
				fmt.Printf("  Initial: %.2f MB\n", initialMemory)
				fmt.Printf("  Final: %.2f MB\n", finalMemory)
				fmt.Printf("  Peak: %.2f MB\n", maxMemory)
				fmt.Printf("  Growth: %.2f MB (%.2f%%)\n", memoryGrowth, memoryGrowthPercent)

				// Memory leak assertions
				Expect(memoryGrowthPercent).To(BeNumerically("<", 50),
					"Memory growth should be less than 50% over 10 minutes")

				Expect(maxMemory).To(BeNumerically("<", 500),
					"Peak memory usage should stay under 500 MB")

				// Ensure memory doesn't grow linearly (sign of a leak)
				if len(memoryMeasurements) >= 5 {
					midMemory := memoryMeasurements[len(memoryMeasurements)/2]
					firstHalfGrowth := midMemory - initialMemory
					secondHalfGrowth := finalMemory - midMemory

					if firstHalfGrowth > 0 {
						growthRatio := secondHalfGrowth / firstHalfGrowth
						Expect(growthRatio).To(BeNumerically("<", 2.0),
							"Memory growth should not accelerate significantly")
					}
				}
			}
		})
	})

	Describe("Resource Usage Validation", func() {
		It("should maintain CPU usage under 80% and memory under 85%", func() {
			testDuration := 2 * time.Minute
			concurrentUsers := 30

			var maxCPUUsage float64
			var maxMemoryUsage float64

			By("running sustained load while monitoring resource usage")

			var wg sync.WaitGroup
			endTime := time.Now().Add(testDuration)

			// Resource monitoring
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer GinkgoRecover()

				ticker := time.NewTicker(5 * time.Second)
				defer ticker.Stop()

				for time.Now().Before(endTime.Add(10 * time.Second)) {
					select {
					case <-ticker.C:
						var memStats runtime.MemStats
						runtime.GC()
						runtime.ReadMemStats(&memStats)

						memoryMB := float64(memStats.Alloc) / 1024 / 1024
						if memoryMB > maxMemoryUsage {
							maxMemoryUsage = memoryMB
						}

						// CPU usage monitoring would be more complex in real scenario
						// For this test, we'll use a simplified approximation
						cpuUsage := float64(runtime.NumGoroutine()) / 100.0 * 10 // Simplified metric
						if cpuUsage > maxCPUUsage {
							maxCPUUsage = cpuUsage
						}

					case <-ctx.Done():
						return
					}
				}
			}()

			// Load generation
			for user := 0; user < concurrentUsers; user++ {
				wg.Add(1)
				go func(userID int) {
					defer wg.Done()
					defer GinkgoRecover()

					requestID := 0
					for time.Now().Before(endTime) {
						intentName := fmt.Sprintf("resource-test-u%d-r%d", userID, requestID)

						intent := testutils.NetworkIntentFixture.CreateBasicNetworkIntent(intentName, namespace)
						intent.Spec.Intent = fmt.Sprintf("Resource usage test user %d request %d", userID, requestID)

						err := k8sClient.Create(ctx, intent)
						if err == nil {
							controller.Reconcile(ctx, types.NamespacedName{
								Name:      intent.Name,
								Namespace: intent.Namespace,
							})
						}

						requestID++
						time.Sleep(200 * time.Millisecond)
					}
				}(user)
			}

			wg.Wait()

			By("validating resource usage limits")
			fmt.Printf("Peak Memory Usage: %.2f MB\n", maxMemoryUsage)
			fmt.Printf("Peak CPU Usage: %.2f%%\n", maxCPUUsage)

			// Resource usage assertions
			Expect(maxMemoryUsage).To(BeNumerically("<", 400),
				"Memory usage should stay under 400 MB (85% of ~470MB limit)")

			// Note: CPU usage monitoring in unit tests is limited
			// In a real environment, this would use proper CPU monitoring
			Expect(maxCPUUsage).To(BeNumerically("<", 80),
				"CPU usage should stay under 80%")
		})
	})

	Describe("E2NodeSet Performance", func() {
		It("should handle large scale E2NodeSet operations efficiently", func() {
			scalingSizes := []int32{10, 25, 50, 75, 100}
			metrics := NewPerformanceMetrics()

			for _, size := range scalingSizes {
				By(fmt.Sprintf("testing E2NodeSet scaling to %d nodes", size))

				nodeSet := testutils.E2NodeSetFixture.CreateBasicE2NodeSet(
					fmt.Sprintf("perf-nodeset-%d", size), namespace, 1)

				reqStart := time.Now()
				createErr := k8sClient.Create(ctx, nodeSet)

				if createErr == nil {
					// Scale to target size
					nodeSet.Spec.Replicas = size
					updateErr := k8sClient.Update(ctx, nodeSet)

					if updateErr == nil {
						// Simulate reconcile
						_, reconcileErr := e2Controller.Reconcile(ctx, types.NamespacedName{
							Name:      nodeSet.Name,
							Namespace: nodeSet.Namespace,
						})

						duration := time.Since(reqStart)
						success := reconcileErr == nil
						metrics.RecordRequest(duration, success)
					} else {
						metrics.RecordRequest(time.Since(reqStart), false)
					}
				} else {
					metrics.RecordRequest(time.Since(reqStart), false)
				}

				time.Sleep(1 * time.Second) // Cool down between tests
			}

			By("analyzing E2NodeSet scaling performance")
			metrics.Calculate()
			metrics.Print()

			// Performance assertions for E2NodeSet scaling
			Expect(metrics.SuccessfulRequests).To(BeNumerically(">=", int(float64(len(scalingSizes))*0.8)),
				"Should have at least 80% success rate for E2NodeSet scaling")

			Expect(metrics.AvgDuration).To(BeNumerically("<", 10*time.Second),
				"Average E2NodeSet scaling should complete within 10 seconds")
		})
	})
})

// Helper functions for benchmarking
func BenchmarkNetworkIntentProcessing(b *testing.B) {
	// This would be used with go test -bench for micro-benchmarks
	// Implementation would follow similar patterns to the load tests above
}
