//go:build integration

package performance_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
)

// Performance testing thresholds and SLA requirements
const (
	// API Response Time SLAs
	P50_THRESHOLD = 100 * time.Millisecond  // 50th percentile under 100ms
	P95_THRESHOLD = 500 * time.Millisecond  // 95th percentile under 500ms
	P99_THRESHOLD = 1000 * time.Millisecond // 99th percentile under 1s

	// Throughput SLAs
	MIN_THROUGHPUT_RPS = 100 // Minimum 100 requests per second

	// Scalability SLAs
	MAX_CONCURRENT_INTENTS  = 200             // Support 200 concurrent intents
	SUSTAINED_LOAD_DURATION = 5 * time.Minute // Sustain load for 5 minutes
)

// PerformanceMetrics tracks performance statistics
type PerformanceMetrics struct {
	requestCount  int64
	totalDuration int64 // nanoseconds
	minDuration   int64
	maxDuration   int64
	errorCount    int64
	durations     []time.Duration
	mutex         sync.RWMutex
}

func NewPerformanceMetrics() *PerformanceMetrics {
	return &PerformanceMetrics{
		minDuration: int64(^uint64(0) >> 1), // max int64
		durations:   make([]time.Duration, 0, 10000),
	}
}

func (pm *PerformanceMetrics) RecordRequest(duration time.Duration, success bool) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	durationNs := duration.Nanoseconds()

	atomic.AddInt64(&pm.requestCount, 1)
	atomic.AddInt64(&pm.totalDuration, durationNs)

	if durationNs < pm.minDuration {
		pm.minDuration = durationNs
	}
	if durationNs > pm.maxDuration {
		pm.maxDuration = durationNs
	}

	if !success {
		atomic.AddInt64(&pm.errorCount, 1)
	}

	pm.durations = append(pm.durations, duration)
}

func (pm *PerformanceMetrics) GetStatistics() map[string]interface{} {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	if len(pm.durations) == 0 {
		return map[string]interface{}{}
	}

	// Sort durations for percentile calculations
	sortedDurations := make([]time.Duration, len(pm.durations))
	copy(sortedDurations, pm.durations)

	// Simple sort implementation for percentiles
	for i := 0; i < len(sortedDurations); i++ {
		for j := i + 1; j < len(sortedDurations); j++ {
			if sortedDurations[i] > sortedDurations[j] {
				sortedDurations[i], sortedDurations[j] = sortedDurations[j], sortedDurations[i]
			}
		}
	}

	count := len(sortedDurations)
	p50 := sortedDurations[count*50/100]
	p95 := sortedDurations[count*95/100]
	p99 := sortedDurations[count*99/100]

	return map[string]interface{}{
		"requestCount":  pm.requestCount,
		"errorCount":    pm.errorCount,
		"errorRate":     float64(pm.errorCount) / float64(pm.requestCount) * 100,
		"avgDuration":   time.Duration(pm.totalDuration / pm.requestCount),
		"minDuration":   time.Duration(pm.minDuration),
		"maxDuration":   time.Duration(pm.maxDuration),
		"p50Duration":   p50,
		"p95Duration":   p95,
		"p99Duration":   p99,
		"throughputRPS": float64(pm.requestCount) / (time.Duration(pm.totalDuration).Seconds()),
	}
}

var _ = Describe("O2 Performance and Load Testing Suite", func() {
	var (
		namespace       *corev1.Namespace
		testCtx         context.Context
		o2Server        *o2.O2APIServer
		httpTestServer  *httptest.Server
		testClient      *http.Client
		metricsRegistry *prometheus.Registry
		testLogger      *logging.StructuredLogger
	)

	BeforeEach(func() {
		namespace = CreateTestNamespace()
		var cancel context.CancelFunc
		testCtx, cancel = context.WithTimeout(ctx, 30*time.Minute)
		DeferCleanup(cancel)

		testLogger = logging.NewLogger("o2-performance-test", "info") // Less verbose for performance tests
		metricsRegistry = prometheus.NewRegistry()

		// Optimized configuration for performance testing
		config := &o2.O2IMSConfig{
			ServerAddress: "127.0.0.1",
			ServerPort:    0,
			TLSEnabled:    false,
			DatabaseConfig: map[string]interface{}{
				"type":            "memory",
				"database":        "o2_perf_test_db",
				"connectionPool":  100,
				"maxIdleConns":    25,
				"maxOpenConns":    100,
				"connMaxLifetime": "1h",
			},
			PerformanceConfig: map[string]interface{}{
				"maxConcurrentRequests": 500,
				"requestTimeout":        "30s",
				"keepAliveTimeout":      "60s",
				"idleTimeout":           "120s",
				"readTimeout":           "30s",
				"writeTimeout":          "30s",
				"enableCompression":     true,
				"cacheEnabled":          true,
				"cacheSize":             1000,
				"cacheTTL":              "5m",
			},
		}

		var err error
		o2Server, err = o2.NewO2APIServer(config, testLogger, metricsRegistry)
		Expect(err).NotTo(HaveOccurred())

		httpTestServer = httptest.NewServer(o2Server.GetRouter())

		// Optimized HTTP client for performance testing
		testClient = &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
		}

		DeferCleanup(func() {
			httpTestServer.Close()
			if o2Server != nil {
				o2Server.Shutdown(testCtx)
			}
		})
	})

	Describe("API Response Time Benchmarks", func() {
		Context("when measuring single request performance", func() {
			It("should meet P50, P95, and P99 response time SLAs", func() {
				metrics := NewPerformanceMetrics()
				numRequests := 1000

				By(fmt.Sprintf("performing %d sequential requests to service info endpoint", numRequests))
				for i := 0; i < numRequests; i++ {
					start := time.Now()
					resp, err := testClient.Get(httpTestServer.URL + "/o2ims/v1/")
					duration := time.Since(start)

					success := err == nil && resp.StatusCode == http.StatusOK
					if resp != nil {
						resp.Body.Close()
					}

					metrics.RecordRequest(duration, success)
				}

				stats := metrics.GetStatistics()

				By("validating response time SLAs")
				Expect(stats["p50Duration"].(time.Duration)).To(BeNumerically("<", P50_THRESHOLD),
					fmt.Sprintf("P50 response time %v exceeds threshold %v", stats["p50Duration"], P50_THRESHOLD))

				Expect(stats["p95Duration"].(time.Duration)).To(BeNumerically("<", P95_THRESHOLD),
					fmt.Sprintf("P95 response time %v exceeds threshold %v", stats["p95Duration"], P95_THRESHOLD))

				Expect(stats["p99Duration"].(time.Duration)).To(BeNumerically("<", P99_THRESHOLD),
					fmt.Sprintf("P99 response time %v exceeds threshold %v", stats["p99Duration"], P99_THRESHOLD))

				By("validating error rate")
				Expect(stats["errorRate"].(float64)).To(BeNumerically("<", 1.0),
					fmt.Sprintf("Error rate %.2f%% exceeds 1%% threshold", stats["errorRate"]))

				By("logging performance statistics")
				testLogger.Info("API Performance Statistics",
					"requestCount", stats["requestCount"],
					"errorCount", stats["errorCount"],
					"errorRate", fmt.Sprintf("%.2f%%", stats["errorRate"]),
					"avgDuration", stats["avgDuration"],
					"p50Duration", stats["p50Duration"],
					"p95Duration", stats["p95Duration"],
					"p99Duration", stats["p99Duration"],
				)
			})

			It("should maintain performance under complex query operations", func() {
				metrics := NewPerformanceMetrics()

				// Create test data first
				By("setting up test data for complex queries")
				numPools := 50
				createdPools := make([]string, numPools)

				for i := 0; i < numPools; i++ {
					poolID := fmt.Sprintf("perf-test-pool-%d-%d", i, time.Now().UnixNano())
					createdPools[i] = poolID

					pool := &models.ResourcePool{
						ResourcePoolID: poolID,
						Name:           fmt.Sprintf("Performance Test Pool %d", i),
						Provider:       []string{"kubernetes", "aws", "azure"}[i%3],
						Location:       []string{"us-west-2", "us-east-1", "eu-west-1"}[i%3],
						OCloudID:       "perf-test-ocloud",
						Capacity: &models.ResourceCapacity{
							CPU: &models.ResourceMetric{
								Total:       fmt.Sprintf("%d", 10+i),
								Available:   fmt.Sprintf("%d", 8+i),
								Used:        "2",
								Unit:        "cores",
								Utilization: 20.0,
							},
						},
						Extensions: map[string]interface{}{
							"performanceTest": true,
							"iteration":       i,
						},
					}

					poolJSON, err := json.Marshal(pool)
					Expect(err).NotTo(HaveOccurred())

					resp, err := testClient.Post(
						httpTestServer.URL+"/o2ims/v1/resourcePools",
						"application/json",
						bytes.NewBuffer(poolJSON),
					)
					Expect(err).NotTo(HaveOccurred())
					resp.Body.Close()
				}

				DeferCleanup(func() {
					for _, poolID := range createdPools {
						req, _ := http.NewRequest("DELETE", httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID, nil)
						testClient.Do(req)
					}
				})

				By("performing complex query operations")
				queryTests := []struct {
					name string
					url  string
				}{
					{"Filter by provider", "/o2ims/v1/resourcePools?filter=provider,eq,kubernetes"},
					{"Filter by location", "/o2ims/v1/resourcePools?filter=location,eq,us-west-2"},
					{"Pagination", "/o2ims/v1/resourcePools?limit=10&offset=0"},
					{"Multiple filters", "/o2ims/v1/resourcePools?filter=provider,eq,aws;location,eq,us-east-1"},
					{"Sorting", "/o2ims/v1/resourcePools?sort=name,asc"},
				}

				for _, test := range queryTests {
					By(fmt.Sprintf("testing %s query performance", test.name))

					for i := 0; i < 100; i++ {
						start := time.Now()
						resp, err := testClient.Get(httpTestServer.URL + test.url)
						duration := time.Since(start)

						success := err == nil && resp.StatusCode == http.StatusOK
						if resp != nil {
							resp.Body.Close()
						}

						metrics.RecordRequest(duration, success)
					}
				}

				stats := metrics.GetStatistics()

				By("validating complex query performance")
				Expect(stats["p95Duration"].(time.Duration)).To(BeNumerically("<", 2*P95_THRESHOLD),
					fmt.Sprintf("Complex query P95 response time %v exceeds threshold", stats["p95Duration"]))

				Expect(stats["errorRate"].(float64)).To(BeNumerically("<", 2.0),
					fmt.Sprintf("Complex query error rate %.2f%% too high", stats["errorRate"]))
			})
		})
	})

	Describe("Concurrent Load Testing", func() {
		Context("when handling concurrent requests", func() {
			It("should support high concurrent request loads", func() {
				concurrencyLevels := []int{10, 50, 100, 200}

				for _, concurrency := range concurrencyLevels {
					By(fmt.Sprintf("testing %d concurrent requests", concurrency))

					metrics := NewPerformanceMetrics()
					var wg sync.WaitGroup

					startTime := time.Now()

					for i := 0; i < concurrency; i++ {
						wg.Add(1)
						go func(workerID int) {
							defer wg.Done()

							// Each worker performs multiple requests
							for j := 0; j < 10; j++ {
								requestStart := time.Now()
								resp, err := testClient.Get(httpTestServer.URL + "/o2ims/v1/")
								duration := time.Since(requestStart)

								success := err == nil && resp != nil && resp.StatusCode == http.StatusOK
								if resp != nil {
									resp.Body.Close()
								}

								metrics.RecordRequest(duration, success)
							}
						}(i)
					}

					wg.Wait()
					totalTestDuration := time.Since(startTime)

					stats := metrics.GetStatistics()
					actualThroughput := float64(stats["requestCount"].(int64)) / totalTestDuration.Seconds()

					By(fmt.Sprintf("validating performance at %d concurrency", concurrency))

					// Adjust thresholds based on concurrency level
					maxP95 := P95_THRESHOLD
					if concurrency > 100 {
						maxP95 = 2 * P95_THRESHOLD // Allow higher latency for very high concurrency
					}

					Expect(stats["p95Duration"].(time.Duration)).To(BeNumerically("<", maxP95),
						fmt.Sprintf("Concurrency %d: P95 response time %v exceeds threshold %v",
							concurrency, stats["p95Duration"], maxP95))

					Expect(stats["errorRate"].(float64)).To(BeNumerically("<", 5.0),
						fmt.Sprintf("Concurrency %d: Error rate %.2f%% too high", concurrency, stats["errorRate"]))

					Expect(actualThroughput).To(BeNumerically(">", MIN_THROUGHPUT_RPS/2),
						fmt.Sprintf("Concurrency %d: Throughput %.1f RPS too low", concurrency, actualThroughput))

					testLogger.Info("Concurrent Load Test Results",
						"concurrency", concurrency,
						"totalRequests", stats["requestCount"],
						"duration", totalTestDuration,
						"throughput", fmt.Sprintf("%.1f RPS", actualThroughput),
						"p95Latency", stats["p95Duration"],
						"errorRate", fmt.Sprintf("%.2f%%", stats["errorRate"]),
					)
				}
			})

			It("should handle sustained high load", func() {
				By("performing sustained load test for 5 minutes")

				metrics := NewPerformanceMetrics()
				stopChan := make(chan struct{})
				var wg sync.WaitGroup

				// Start sustained load
				numWorkers := 50
				for i := 0; i < numWorkers; i++ {
					wg.Add(1)
					go func(workerID int) {
						defer wg.Done()

						for {
							select {
							case <-stopChan:
								return
							default:
								start := time.Now()
								resp, err := testClient.Get(httpTestServer.URL + "/o2ims/v1/")
								duration := time.Since(start)

								success := err == nil && resp != nil && resp.StatusCode == http.StatusOK
								if resp != nil {
									resp.Body.Close()
								}

								metrics.RecordRequest(duration, success)

								// Small delay to prevent overwhelming
								time.Sleep(50 * time.Millisecond)
							}
						}
					}(i)
				}

				// Run for sustained duration
				time.Sleep(SUSTAINED_LOAD_DURATION)
				close(stopChan)
				wg.Wait()

				stats := metrics.GetStatistics()

				By("validating sustained load performance")
				Expect(stats["errorRate"].(float64)).To(BeNumerically("<", 3.0),
					fmt.Sprintf("Sustained load error rate %.2f%% too high", stats["errorRate"]))

				Expect(stats["p99Duration"].(time.Duration)).To(BeNumerically("<", 2*P99_THRESHOLD),
					fmt.Sprintf("Sustained load P99 response time %v exceeds threshold", stats["p99Duration"]))

				totalDuration := SUSTAINED_LOAD_DURATION
				actualThroughput := float64(stats["requestCount"].(int64)) / totalDuration.Seconds()
				Expect(actualThroughput).To(BeNumerically(">", MIN_THROUGHPUT_RPS/3),
					fmt.Sprintf("Sustained throughput %.1f RPS too low", actualThroughput))

				testLogger.Info("Sustained Load Test Results",
					"duration", totalDuration,
					"totalRequests", stats["requestCount"],
					"throughput", fmt.Sprintf("%.1f RPS", actualThroughput),
					"p99Latency", stats["p99Duration"],
					"errorRate", fmt.Sprintf("%.2f%%", stats["errorRate"]),
				)
			})
		})
	})

	Describe("Resource-Intensive Operations Performance", func() {
		Context("when performing CRUD operations under load", func() {
			It("should maintain performance during resource pool operations", func() {
				metrics := NewPerformanceMetrics()
				numWorkers := 20
				operationsPerWorker := 25

				var wg sync.WaitGroup
				createdPools := make(chan string, numWorkers*operationsPerWorker)

				By("performing concurrent resource pool CRUD operations")

				for i := 0; i < numWorkers; i++ {
					wg.Add(1)
					go func(workerID int) {
						defer wg.Done()

						for j := 0; j < operationsPerWorker; j++ {
							poolID := fmt.Sprintf("perf-crud-pool-%d-%d-%d", workerID, j, time.Now().UnixNano())

							// CREATE operation
							pool := &models.ResourcePool{
								ResourcePoolID: poolID,
								Name:           fmt.Sprintf("Performance CRUD Pool %d-%d", workerID, j),
								Provider:       "kubernetes",
								OCloudID:       "perf-test-ocloud",
							}

							poolJSON, err := json.Marshal(pool)
							if err != nil {
								continue
							}

							start := time.Now()
							resp, err := testClient.Post(
								httpTestServer.URL+"/o2ims/v1/resourcePools",
								"application/json",
								bytes.NewBuffer(poolJSON),
							)
							createDuration := time.Since(start)

							success := err == nil && resp != nil &&
								(resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK)
							if resp != nil {
								resp.Body.Close()
							}

							metrics.RecordRequest(createDuration, success)

							if success {
								createdPools <- poolID

								// READ operation
								start = time.Now()
								resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/resourcePools/" + poolID)
								readDuration := time.Since(start)

								readSuccess := err == nil && resp != nil && resp.StatusCode == http.StatusOK
								if resp != nil {
									resp.Body.Close()
								}

								metrics.RecordRequest(readDuration, readSuccess)

								// UPDATE operation
								updateData := map[string]interface{}{
									"description": fmt.Sprintf("Updated pool %d-%d", workerID, j),
								}
								updateJSON, err := json.Marshal(updateData)
								if err == nil {
									start = time.Now()
									req, err := http.NewRequest("PATCH",
										httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID,
										bytes.NewBuffer(updateJSON))
									if err == nil {
										req.Header.Set("Content-Type", "application/json")
										resp, err = testClient.Do(req)
										updateDuration := time.Since(start)

										updateSuccess := err == nil && resp != nil && resp.StatusCode == http.StatusOK
										if resp != nil {
											resp.Body.Close()
										}

										metrics.RecordRequest(updateDuration, updateSuccess)
									}
								}
							}
						}
					}(i)
				}

				wg.Wait()
				close(createdPools)

				// Cleanup created resources
				DeferCleanup(func() {
					for poolID := range createdPools {
						req, _ := http.NewRequest("DELETE", httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID, nil)
						testClient.Do(req)
					}
				})

				stats := metrics.GetStatistics()

				By("validating CRUD operation performance")
				Expect(stats["p95Duration"].(time.Duration)).To(BeNumerically("<", 3*P95_THRESHOLD),
					fmt.Sprintf("CRUD P95 response time %v exceeds threshold", stats["p95Duration"]))

				Expect(stats["errorRate"].(float64)).To(BeNumerically("<", 10.0),
					fmt.Sprintf("CRUD error rate %.2f%% too high", stats["errorRate"]))

				testLogger.Info("CRUD Performance Test Results",
					"totalOperations", stats["requestCount"],
					"p95Latency", stats["p95Duration"],
					"avgLatency", stats["avgDuration"],
					"errorRate", fmt.Sprintf("%.2f%%", stats["errorRate"]),
				)
			})
		})
	})

	Describe("Memory and Resource Utilization", func() {
		Context("when monitoring resource consumption", func() {
			It("should maintain stable memory usage under load", func() {
				By("performing memory stress test")

				// Create many resources to test memory usage
				numResources := 1000
				createdPools := make([]string, numResources)

				startTime := time.Now()

				for i := 0; i < numResources; i++ {
					poolID := fmt.Sprintf("memory-test-pool-%d", i)
					createdPools[i] = poolID

					pool := &models.ResourcePool{
						ResourcePoolID: poolID,
						Name:           fmt.Sprintf("Memory Test Pool %d", i),
						Provider:       "kubernetes",
						OCloudID:       "memory-test-ocloud",
						Extensions: map[string]interface{}{
							"largeData": strings.Repeat("x", 1000), // 1KB of data per resource
							"iteration": i,
							"metadata": map[string]interface{}{
								"createdAt": time.Now().Format(time.RFC3339),
								"testType":  "memory-stress",
							},
						},
					}

					poolJSON, err := json.Marshal(pool)
					Expect(err).NotTo(HaveOccurred())

					resp, err := testClient.Post(
						httpTestServer.URL+"/o2ims/v1/resourcePools",
						"application/json",
						bytes.NewBuffer(poolJSON),
					)
					Expect(err).NotTo(HaveOccurred())
					Expect(resp.StatusCode).To(Or(Equal(http.StatusCreated), Equal(http.StatusOK)))
					resp.Body.Close()

					if i%100 == 0 {
						testLogger.Info("Memory test progress", "created", i+1, "total", numResources)
					}
				}

				creationTime := time.Since(startTime)

				By("performing operations on all resources")

				// Test retrieval performance with large dataset
				metrics := NewPerformanceMetrics()

				start := time.Now()
				resp, err := testClient.Get(httpTestServer.URL + "/o2ims/v1/resourcePools")
				listDuration := time.Since(start)

				success := err == nil && resp != nil && resp.StatusCode == http.StatusOK
				if resp != nil {
					var pools []models.ResourcePool
					json.NewDecoder(resp.Body).Decode(&pools)
					resp.Body.Close()

					// Verify we can retrieve all resources
					memoryTestPools := 0
					for _, pool := range pools {
						if pool.Extensions != nil {
							if testType, ok := pool.Extensions["testType"].(string); ok && testType == "memory-stress" {
								memoryTestPools++
							}
						}
					}
					Expect(memoryTestPools).To(BeNumerically(">=", numResources*0.9)) // Allow some tolerance
				}

				metrics.RecordRequest(listDuration, success)

				// Test individual resource access
				for i := 0; i < 100; i++ { // Sample 100 random resources
					poolID := createdPools[i*10] // Every 10th resource

					start = time.Now()
					resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/resourcePools/" + poolID)
					getDuration := time.Since(start)

					getSuccess := err == nil && resp != nil && resp.StatusCode == http.StatusOK
					if resp != nil {
						resp.Body.Close()
					}

					metrics.RecordRequest(getDuration, getSuccess)
				}

				stats := metrics.GetStatistics()

				By("validating memory stress test performance")
				Expect(stats["p95Duration"].(time.Duration)).To(BeNumerically("<", 5*P95_THRESHOLD),
					fmt.Sprintf("Memory stress P95 response time %v exceeds threshold", stats["p95Duration"]))

				// List operation should still be reasonably fast even with many resources
				Expect(listDuration).To(BeNumerically("<", 10*time.Second),
					fmt.Sprintf("List operation took too long: %v", listDuration))

				testLogger.Info("Memory Stress Test Results",
					"resourceCount", numResources,
					"creationTime", creationTime,
					"listDuration", listDuration,
					"sampleP95", stats["p95Duration"],
					"errorRate", fmt.Sprintf("%.2f%%", stats["errorRate"]),
				)

				DeferCleanup(func() {
					By("cleaning up memory test resources")
					for i, poolID := range createdPools {
						req, _ := http.NewRequest("DELETE", httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID, nil)
						testClient.Do(req)

						if i%100 == 0 {
							testLogger.Info("Cleanup progress", "deleted", i+1, "total", numResources)
						}
					}
				})
			})
		})
	})

	Describe("Scalability Validation", func() {
		Context("when testing system scalability limits", func() {
			It("should support 200+ concurrent intent processing", func() {
				By("testing maximum concurrent intent processing")

				maxConcurrent := MAX_CONCURRENT_INTENTS
				metrics := NewPerformanceMetrics()

				var wg sync.WaitGroup
				semaphore := make(chan struct{}, maxConcurrent)

				startTime := time.Now()

				// Launch workers up to the maximum concurrent limit
				for i := 0; i < maxConcurrent; i++ {
					wg.Add(1)
					go func(intentID int) {
						defer wg.Done()

						// Acquire semaphore
						semaphore <- struct{}{}
						defer func() { <-semaphore }()

						// Simulate intent processing by creating and managing a resource
						poolID := fmt.Sprintf("intent-pool-%d", intentID)

						pool := &models.ResourcePool{
							ResourcePoolID: poolID,
							Name:           fmt.Sprintf("Intent Pool %d", intentID),
							Provider:       "kubernetes",
							OCloudID:       "scalability-test-ocloud",
							Extensions: map[string]interface{}{
								"intentID":        intentID,
								"processingStart": time.Now().Format(time.RFC3339),
							},
						}

						poolJSON, err := json.Marshal(pool)
						if err != nil {
							metrics.RecordRequest(0, false)
							return
						}

						// Create resource (intent processing step 1)
						start := time.Now()
						resp, err := testClient.Post(
							httpTestServer.URL+"/o2ims/v1/resourcePools",
							"application/json",
							bytes.NewBuffer(poolJSON),
						)

						success := err == nil && resp != nil &&
							(resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK)
						if resp != nil {
							resp.Body.Close()
						}

						if !success {
							metrics.RecordRequest(time.Since(start), false)
							return
						}

						// Verify resource (intent processing step 2)
						resp, err = testClient.Get(httpTestServer.URL + "/o2ims/v1/resourcePools/" + poolID)
						if resp != nil {
							resp.Body.Close()
						}

						// Update resource (intent processing step 3)
						updateData := map[string]interface{}{
							"extensions": map[string]interface{}{
								"intentID":           intentID,
								"processingComplete": time.Now().Format(time.RFC3339),
								"status":             "processed",
							},
						}

						updateJSON, err := json.Marshal(updateData)
						if err == nil {
							req, err := http.NewRequest("PATCH",
								httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID,
								bytes.NewBuffer(updateJSON))
							if err == nil {
								req.Header.Set("Content-Type", "application/json")
								resp, err := testClient.Do(req)
								if resp != nil {
									resp.Body.Close()
								}
							}
						}

						totalDuration := time.Since(start)
						metrics.RecordRequest(totalDuration, true)

						// Cleanup
						req, _ := http.NewRequest("DELETE", httpTestServer.URL+"/o2ims/v1/resourcePools/"+poolID, nil)
						testClient.Do(req)

					}(i)
				}

				wg.Wait()
				totalDuration := time.Since(startTime)

				stats := metrics.GetStatistics()

				By("validating concurrent intent processing performance")

				successRate := 100.0 - stats["errorRate"].(float64)
				Expect(successRate).To(BeNumerically(">", 95.0),
					fmt.Sprintf("Success rate %.1f%% too low for %d concurrent intents", successRate, maxConcurrent))

				Expect(stats["p95Duration"].(time.Duration)).To(BeNumerically("<", 10*P95_THRESHOLD),
					fmt.Sprintf("Intent processing P95 duration %v too high", stats["p95Duration"]))

				avgIntentDuration := stats["avgDuration"].(time.Duration)
				Expect(avgIntentDuration).To(BeNumerically("<", 5*time.Second),
					fmt.Sprintf("Average intent processing time %v too high", avgIntentDuration))

				testLogger.Info("Scalability Test Results",
					"maxConcurrent", maxConcurrent,
					"totalDuration", totalDuration,
					"successRate", fmt.Sprintf("%.1f%%", successRate),
					"avgIntentDuration", avgIntentDuration,
					"p95IntentDuration", stats["p95Duration"],
					"totalIntents", stats["requestCount"],
				)
			})
		})
	})
})

// Helper function to create test namespace
func CreateTestNamespace() *corev1.Namespace {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "o2-perf-test-",
		},
	}

	// In a real implementation, this would use the k8s client
	// For testing, we'll just return the namespace object
	return namespace
}
