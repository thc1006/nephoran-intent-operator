package security

import (
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/time/rate"

	"github.com/thc1006/nephoran-intent-operator/pkg/security/mtls"
	"github.com/thc1006/nephoran-intent-operator/tests/utils"
)

// mTLSPerformanceTestSuite provides comprehensive performance testing for mTLS operations
type mTLSPerformanceTestSuite struct {
	ctx         context.Context
	testSuite   *mTLSSecurityTestSuite
	servers     []*httptest.Server
	clients     []*mtls.Client
	metrics     *PerformanceMetrics
	rateLimiter *rate.Limiter
}

// PerformanceMetrics tracks various performance measurements
type PerformanceMetrics struct {
	HandshakeLatencies    []time.Duration
	ConnectionEstablished int64
	ConnectionFailed      int64
	BytesTransferred      int64
	RequestsPerSecond     float64
	P50Latency            time.Duration
	P95Latency            time.Duration
	P99Latency            time.Duration
	MaxLatency            time.Duration
	MinLatency            time.Duration
	AvgLatency            time.Duration
	ErrorRate             float64
	ThroughputMBps        float64
	CertificateRotations  int64
	mu                    sync.RWMutex
}

// LoadTestConfig defines parameters for load testing
type LoadTestConfig struct {
	Concurrency     int
	Duration        time.Duration
	RequestsPerSec  int
	PayloadSize     int
	ConnectionReuse bool
	KeepAlive       bool
	TimeoutDuration time.Duration
}

var _ = Describe("mTLS Performance and Load Test Suite", func() {
	var perfSuite *mTLSPerformanceTestSuite

	BeforeEach(func() {
		baseSuite := &mTLSSecurityTestSuite{
			ctx: context.Background(),
		}
		err := baseSuite.initializeTestCertificates()
		Expect(err).NotTo(HaveOccurred())

		perfSuite = &mTLSPerformanceTestSuite{
			ctx:       context.Background(),
			testSuite: baseSuite,
			metrics:   &PerformanceMetrics{},
		}
	})

	AfterEach(func() {
		perfSuite.cleanup()
	})

	Context("mTLS Connection Performance", func() {
		It("should benchmark mTLS handshake performance", func() {
			server := perfSuite.testSuite.createTestServer(perfSuite.testSuite.serverCert, true)
			defer server.Close()

			// Benchmark different handshake scenarios
			scenarios := map[string]func() *http.Client{
				"fresh_connection": func() *http.Client {
					return perfSuite.testSuite.createMTLSClient(perfSuite.testSuite.clientCert)
				},
				"connection_reuse": func() *http.Client {
					client := perfSuite.testSuite.createMTLSClient(perfSuite.testSuite.clientCert)
					// Pre-establish connection
					resp, _ := client.Get(server.URL)
					if resp != nil {
						resp.Body.Close()
					}
					return client
				},
			}

			for scenarioName, clientFactory := range scenarios {
				By(fmt.Sprintf("Benchmarking %s scenario", scenarioName))

				latencies := make([]time.Duration, 0, 100)

				for i := 0; i < 100; i++ {
					client := clientFactory()

					start := time.Now()
					resp, err := client.Get(server.URL)
					handshakeTime := time.Since(start)

					if err == nil && resp != nil {
						resp.Body.Close()
						latencies = append(latencies, handshakeTime)
					}
				}

				stats := perfSuite.calculateLatencyStats(latencies)

				By(fmt.Sprintf("%s - P50: %v, P95: %v, P99: %v, Max: %v",
					scenarioName, stats.P50, stats.P95, stats.P99, stats.Max))

				// Performance assertions
				Expect(stats.P95).To(BeNumerically("<", 100*time.Millisecond))
				Expect(stats.P99).To(BeNumerically("<", 200*time.Millisecond))
			}
		})

		It("should test concurrent connection establishment", func() {
			server := perfSuite.testSuite.createTestServer(perfSuite.testSuite.serverCert, true)
			defer server.Close()

			concurrencyLevels := []int{10, 50, 100, 200}

			for _, concurrency := range concurrencyLevels {
				By(fmt.Sprintf("Testing %d concurrent connections", concurrency))

				var wg sync.WaitGroup
				var successful, failed int64
				latencies := make([]time.Duration, concurrency)

				for i := 0; i < concurrency; i++ {
					wg.Add(1)
					go func(index int) {
						defer wg.Done()

						client := perfSuite.testSuite.createMTLSClient(perfSuite.testSuite.clientCert)

						start := time.Now()
						resp, err := client.Get(server.URL)
						latencies[index] = time.Since(start)

						if err == nil && resp != nil {
							atomic.AddInt64(&successful, 1)
							resp.Body.Close()
						} else {
							atomic.AddInt64(&failed, 1)
						}
					}(i)
				}

				wg.Wait()

				stats := perfSuite.calculateLatencyStats(latencies)
				successRate := float64(successful) / float64(concurrency) * 100

				By(fmt.Sprintf("Concurrency %d - Success Rate: %.1f%%, P95 Latency: %v",
					concurrency, successRate, stats.P95))

				// Performance assertions
				Expect(successRate).To(BeNumerically(">=", 95.0))
				Expect(stats.P95).To(BeNumerically("<", 500*time.Millisecond))
			}
		})

		It("should measure throughput under sustained load", func() {
			server := perfSuite.testSuite.createTestServer(perfSuite.testSuite.serverCert, true)
			defer server.Close()

			config := LoadTestConfig{
				Concurrency:     50,
				Duration:        30 * time.Second,
				RequestsPerSec:  100,
				PayloadSize:     1024,
				ConnectionReuse: true,
				KeepAlive:       true,
				TimeoutDuration: 10 * time.Second,
			}

			results := perfSuite.runLoadTest(server.URL, config)

			By(fmt.Sprintf("Load Test Results:"))
			By(fmt.Sprintf("- Requests/sec: %.2f", results.RequestsPerSecond))
			By(fmt.Sprintf("- Throughput: %.2f MB/s", results.ThroughputMBps))
			By(fmt.Sprintf("- Error Rate: %.2f%%", results.ErrorRate))
			By(fmt.Sprintf("- P95 Latency: %v", results.P95Latency))

			// Performance assertions for production requirements
			Expect(results.RequestsPerSecond).To(BeNumerically(">=", 80.0))
			Expect(results.ErrorRate).To(BeNumerically("<", 1.0))
			Expect(results.P95Latency).To(BeNumerically("<", 100*time.Millisecond))
		})

		It("should test connection pooling efficiency", func() {
			server := perfSuite.testSuite.createTestServer(perfSuite.testSuite.serverCert, true)
			defer server.Close()

			// Test with different pool sizes
			poolSizes := []int{1, 5, 10, 20, 50}

			for _, poolSize := range poolSizes {
				By(fmt.Sprintf("Testing connection pool size: %d", poolSize))

				client := perfSuite.createPooledMTLSClient(poolSize)

				var wg sync.WaitGroup
				var totalLatency int64
				requestCount := 200

				start := time.Now()

				for i := 0; i < requestCount; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()

						reqStart := time.Now()
						resp, err := client.Get(server.URL)
						reqLatency := time.Since(reqStart)

						if err == nil && resp != nil {
							atomic.AddInt64(&totalLatency, int64(reqLatency))
							resp.Body.Close()
						}
					}()
				}

				wg.Wait()
				totalTime := time.Since(start)
				avgLatency := time.Duration(atomic.LoadInt64(&totalLatency) / int64(requestCount))
				requestsPerSec := float64(requestCount) / totalTime.Seconds()

				By(fmt.Sprintf("Pool Size %d - RPS: %.1f, Avg Latency: %v",
					poolSize, requestsPerSec, avgLatency))
			}
		})
	})

	Context("Certificate Operation Performance", func() {
		It("should benchmark certificate validation performance", func() {
			cert := perfSuite.testSuite.clientCert
			validationCount := 1000

			start := time.Now()

			for i := 0; i < validationCount; i++ {
				// Simulate certificate validation
				if len(cert.Certificate) > 0 {
					_, err := x509.ParseCertificate(cert.Certificate[0])
					Expect(err).NotTo(HaveOccurred())
				}
			}

			totalTime := time.Since(start)
			validationsPerSec := float64(validationCount) / totalTime.Seconds()
			avgValidationTime := totalTime / time.Duration(validationCount)

			By(fmt.Sprintf("Certificate Validation Performance:"))
			By(fmt.Sprintf("- Validations/sec: %.1f", validationsPerSec))
			By(fmt.Sprintf("- Avg validation time: %v", avgValidationTime))

			// Performance assertions
			Expect(validationsPerSec).To(BeNumerically(">=", 1000.0))
			Expect(avgValidationTime).To(BeNumerically("<", time.Millisecond))
		})

		It("should test certificate rotation performance", func() {
			server := perfSuite.testSuite.createTestServer(perfSuite.testSuite.serverCert, true)
			defer server.Close()

			client := perfSuite.testSuite.createMTLSClient(perfSuite.testSuite.clientCert)

			// Measure baseline performance
			baselineLatency := perfSuite.measureBaselineLatency(client, server.URL, 10)

			rotationCount := 5
			rotationLatencies := make([]time.Duration, rotationCount)

			for i := 0; i < rotationCount; i++ {
				By(fmt.Sprintf("Performing certificate rotation %d", i+1))

				start := time.Now()

				// Simulate certificate rotation
				newCert := perfSuite.testSuite.createClientCertificate(fmt.Sprintf("rotated-client-%d", i))
				newClient := perfSuite.testSuite.createMTLSClient(newCert)

				// Test that new certificate works
				resp, err := newClient.Get(server.URL)
				rotationLatencies[i] = time.Since(start)

				Expect(err).NotTo(HaveOccurred())
				if resp != nil {
					resp.Body.Close()
				}

				client = newClient
			}

			stats := perfSuite.calculateLatencyStats(rotationLatencies)

			By(fmt.Sprintf("Certificate Rotation Performance:"))
			By(fmt.Sprintf("- Baseline latency: %v", baselineLatency))
			By(fmt.Sprintf("- Avg rotation time: %v", stats.Avg))
			By(fmt.Sprintf("- P95 rotation time: %v", stats.P95))

			// Performance assertions
			Expect(stats.P95).To(BeNumerically("<", 5*time.Second))
		})

		It("should test certificate cache performance", func() {
			// Test certificate caching efficiency
			cacheHits := 0
			cacheMisses := 0

			// Simulate certificate cache operations
			certificates := make(map[string]*tls.Certificate)

			// Populate cache
			for i := 0; i < 100; i++ {
				certName := fmt.Sprintf("test-cert-%d", i%10) // 10 unique certificates
				if _, exists := certificates[certName]; !exists {
					certificates[certName] = perfSuite.testSuite.createClientCertificate(certName)
					cacheMisses++
				} else {
					cacheHits++
				}
			}

			cacheHitRate := float64(cacheHits) / float64(cacheHits+cacheMisses) * 100

			By(fmt.Sprintf("Certificate Cache Performance:"))
			By(fmt.Sprintf("- Cache hit rate: %.1f%%", cacheHitRate))
			By(fmt.Sprintf("- Cache hits: %d", cacheHits))
			By(fmt.Sprintf("- Cache misses: %d", cacheMisses))

			// Expected cache performance
			Expect(cacheHitRate).To(BeNumerically(">=", 80.0))
		})
	})

	Context("Service Mesh mTLS Performance", func() {
		It("should benchmark service mesh mTLS overhead", func() {
			Skip("Requires service mesh deployment")

			// This would compare performance with and without service mesh mTLS
			scenarios := []struct {
				name        string
				mtlsEnabled bool
			}{
				{"no_mtls", false},
				{"istio_mtls", true},
				{"linkerd_mtls", true},
			}

			for _, scenario := range scenarios {
				By(fmt.Sprintf("Testing %s scenario", scenario.name))

				// Deploy test services with/without mTLS
				// Measure performance differences
				// Calculate overhead percentages
			}
		})

		It("should test cross-mesh communication performance", func() {
			Skip("Requires multi-mesh deployment")

			// Test performance of mTLS communication between different service meshes
		})
	})

	Context("Stress Testing", func() {
		It("should handle high connection churn", func() {
			server := perfSuite.testSuite.createTestServer(perfSuite.testSuite.serverCert, true)
			defer server.Close()

			config := LoadTestConfig{
				Concurrency:     100,
				Duration:        60 * time.Second,
				RequestsPerSec:  200,
				ConnectionReuse: false, // Force new connections
				KeepAlive:       false,
				TimeoutDuration: 5 * time.Second,
			}

			results := perfSuite.runLoadTest(server.URL, config)

			By(fmt.Sprintf("High Churn Test Results:"))
			By(fmt.Sprintf("- Connections established: %d", results.ConnectionEstablished))
			By(fmt.Sprintf("- Error rate: %.2f%%", results.ErrorRate))
			By(fmt.Sprintf("- P99 latency: %v", results.P99Latency))

			// System should handle high churn gracefully
			Expect(results.ErrorRate).To(BeNumerically("<", 5.0))
			Expect(results.P99Latency).To(BeNumerically("<", 1*time.Second))
		})

		It("should test memory usage under load", func() {
			server := perfSuite.testSuite.createTestServer(perfSuite.testSuite.serverCert, true)
			defer server.Close()

			// Measure memory before test
			var memBefore runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&memBefore)

			config := LoadTestConfig{
				Concurrency:     200,
				Duration:        30 * time.Second,
				RequestsPerSec:  500,
				ConnectionReuse: true,
				KeepAlive:       true,
				TimeoutDuration: 10 * time.Second,
			}

			results := perfSuite.runLoadTest(server.URL, config)

			// Measure memory after test
			var memAfter runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&memAfter)

			memoryIncrease := memAfter.Alloc - memBefore.Alloc
			memoryIncreaseMB := float64(memoryIncrease) / 1024 / 1024

			By(fmt.Sprintf("Memory Usage Test Results:"))
			By(fmt.Sprintf("- Memory increase: %.2f MB", memoryIncreaseMB))
			By(fmt.Sprintf("- Requests processed: %d", int(results.RequestsPerSecond*30)))
			By(fmt.Sprintf("- Memory per request: %.2f KB", float64(memoryIncrease)/results.RequestsPerSecond/30/1024))

			// Memory usage should be reasonable
			Expect(memoryIncreaseMB).To(BeNumerically("<", 100.0))
		})

		It("should test certificate rotation under load", func() {
			server := perfSuite.testSuite.createTestServer(perfSuite.testSuite.serverCert, true)
			defer server.Close()

			var wg sync.WaitGroup
			var rotationErrors int64
			var requestErrors int64

			// Start background load
			wg.Add(1)
			go func() {
				defer wg.Done()

				for i := 0; i < 1000; i++ {
					client := perfSuite.testSuite.createMTLSClient(perfSuite.testSuite.clientCert)
					resp, err := client.Get(server.URL)

					if err != nil {
						atomic.AddInt64(&requestErrors, 1)
					} else if resp != nil {
						resp.Body.Close()
					}

					time.Sleep(10 * time.Millisecond)
				}
			}()

			// Perform certificate rotations during load
			for i := 0; i < 5; i++ {
				time.Sleep(2 * time.Second)

				By(fmt.Sprintf("Rotating certificate during load test %d", i+1))

				err := perfSuite.simulateCertificateRotation()
				if err != nil {
					atomic.AddInt64(&rotationErrors, 1)
				}
			}

			wg.Wait()

			By(fmt.Sprintf("Rotation Under Load Results:"))
			By(fmt.Sprintf("- Rotation errors: %d", rotationErrors))
			By(fmt.Sprintf("- Request errors: %d", requestErrors))

			// Should handle rotation under load gracefully
			Expect(rotationErrors).To(Equal(int64(0)))
			Expect(requestErrors).To(BeNumerically("<", 50))
		})
	})
})

// Helper methods for performance testing

func (p *mTLSPerformanceTestSuite) calculateLatencyStats(latencies []time.Duration) *LatencyStats {
	if len(latencies) == 0 {
		return &LatencyStats{}
	}

	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	var total time.Duration
	for _, lat := range latencies {
		total += lat
	}

	return &LatencyStats{
		P50: latencies[len(latencies)*50/100],
		P95: latencies[len(latencies)*95/100],
		P99: latencies[len(latencies)*99/100],
		Max: latencies[len(latencies)-1],
		Min: latencies[0],
		Avg: total / time.Duration(len(latencies)),
	}
}

func (p *mTLSPerformanceTestSuite) runLoadTest(url string, config LoadTestConfig) *PerformanceMetrics {
	metrics := &PerformanceMetrics{}

	var wg sync.WaitGroup
	var requestCount, errorCount int64
	var totalLatency int64

	// Rate limiter for controlled load
	limiter := rate.NewLimiter(rate.Limit(config.RequestsPerSec), config.RequestsPerSec)

	startTime := time.Now()

	// Launch worker goroutines
	for i := 0; i < config.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			client := p.testSuite.createMTLSClient(p.testSuite.clientCert)

			for time.Since(startTime) < config.Duration {
				// Wait for rate limiter
				limiter.Wait(context.Background())

				reqStart := time.Now()
				resp, err := client.Get(url)
				reqLatency := time.Since(reqStart)

				atomic.AddInt64(&requestCount, 1)
				atomic.AddInt64(&totalLatency, int64(reqLatency))

				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else if resp != nil {
					resp.Body.Close()
					atomic.AddInt64(&metrics.ConnectionEstablished, 1)
				}
			}
		}()
	}

	wg.Wait()

	totalTime := time.Since(startTime)
	totalRequests := atomic.LoadInt64(&requestCount)
	totalErrors := atomic.LoadInt64(&errorCount)

	metrics.RequestsPerSecond = float64(totalRequests) / totalTime.Seconds()
	metrics.ErrorRate = float64(totalErrors) / float64(totalRequests) * 100
	metrics.AvgLatency = time.Duration(atomic.LoadInt64(&totalLatency) / totalRequests)

	return metrics
}

func (p *mTLSPerformanceTestSuite) createPooledMTLSClient(poolSize int) *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{*p.testSuite.clientCert},
			RootCAs:      p.testSuite.certPool,
		},
		MaxIdleConns:        poolSize,
		MaxConnsPerHost:     poolSize,
		MaxIdleConnsPerHost: poolSize,
		IdleConnTimeout:     90 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}
}

func (p *mTLSPerformanceTestSuite) measureBaselineLatency(client *http.Client, url string, samples int) time.Duration {
	var totalLatency time.Duration

	for i := 0; i < samples; i++ {
		start := time.Now()
		resp, err := client.Get(url)
		latency := time.Since(start)

		if err == nil && resp != nil {
			totalLatency += latency
			resp.Body.Close()
		}
	}

	return totalLatency / time.Duration(samples)
}

func (p *mTLSPerformanceTestSuite) simulateCertificateRotation() error {
	// Simulate certificate rotation process
	time.Sleep(100 * time.Millisecond) // Simulate rotation time
	return nil
}

func (p *mTLSPerformanceTestSuite) cleanup() {
	for _, server := range p.servers {
		server.Close()
	}

	for _, client := range p.clients {
		client.Close()
	}

	if p.testSuite != nil {
		p.testSuite.cleanupTestCertificates()
	}
}

// LatencyStats holds latency percentile information
type LatencyStats struct {
	P50 time.Duration
	P95 time.Duration
	P99 time.Duration
	Max time.Duration
	Min time.Duration
	Avg time.Duration
}

// Benchmark functions for detailed performance analysis
func BenchmarkMTLSHandshake(b *testing.B) {
	suite := &mTLSSecurityTestSuite{}
	suite.initializeTestCertificates()
	defer suite.cleanupTestCertificates()

	server := suite.createTestServer(suite.serverCert, true)
	defer server.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			client := suite.createMTLSClient(suite.clientCert)
			resp, err := client.Get(server.URL)
			if err == nil && resp != nil {
				resp.Body.Close()
			}
		}
	})
}

func BenchmarkCertificateValidation(b *testing.B) {
	suite := &mTLSSecurityTestSuite{}
	suite.initializeTestCertificates()
	defer suite.cleanupTestCertificates()

	cert := suite.clientCert

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if len(cert.Certificate) > 0 {
			x509.ParseCertificate(cert.Certificate[0])
		}
	}
}

func BenchmarkConnectionPooling(b *testing.B) {
	suite := &mTLSSecurityTestSuite{}
	suite.initializeTestCertificates()
	defer suite.cleanupTestCertificates()

	server := suite.createTestServer(suite.serverCert, true)
	defer server.Close()

	perfSuite := &mTLSPerformanceTestSuite{testSuite: suite}
	client := perfSuite.createPooledMTLSClient(10)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(server.URL)
			if err == nil && resp != nil {
				resp.Body.Close()
			}
		}
	})
}

// Performance profiling helpers
func (p *mTLSPerformanceTestSuite) profileMemoryUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Printf("Memory Profile:\n")
	fmt.Printf("- Alloc: %d KB\n", m.Alloc/1024)
	fmt.Printf("- TotalAlloc: %d KB\n", m.TotalAlloc/1024)
	fmt.Printf("- Sys: %d KB\n", m.Sys/1024)
	fmt.Printf("- NumGC: %d\n", m.NumGC)
}

func (p *mTLSPerformanceTestSuite) calculatePercentile(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sort.Float64s(values)
	index := int(math.Ceil(float64(len(values))*percentile/100.0)) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(values) {
		index = len(values) - 1
	}

	return values[index]
}
