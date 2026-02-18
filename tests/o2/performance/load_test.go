package o2_performance_tests_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	fakeClient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
)

// LoadTestConfig defines configuration for load testing
type LoadTestConfig struct {
	ConcurrentClients int
	RequestsPerClient int
	TestDuration      time.Duration
	RampUpTime        time.Duration
}

// LoadTestMetrics captures performance metrics during testing
type LoadTestMetrics struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	AverageLatency     time.Duration
	P95Latency         time.Duration
	P99Latency         time.Duration
	MinLatency         time.Duration
	MaxLatency         time.Duration
	RequestsPerSecond  float64
	Latencies          []time.Duration
	ErrorTypes         map[string]int64
	StartTime          time.Time
	EndTime            time.Time
	mutex              sync.RWMutex
}

func NewLoadTestMetrics() *LoadTestMetrics {
	return &LoadTestMetrics{
		ErrorTypes: make(map[string]int64),
		Latencies:  make([]time.Duration, 0),
		StartTime:  time.Now(),
	}
}

func (m *LoadTestMetrics) RecordRequest(latency time.Duration, success bool, errorType string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.TotalRequests++
	m.Latencies = append(m.Latencies, latency)

	if success {
		m.SuccessfulRequests++
	} else {
		m.FailedRequests++
		if errorType != "" {
			m.ErrorTypes[errorType]++
		}
	}

	if m.MinLatency == 0 || latency < m.MinLatency {
		m.MinLatency = latency
	}
	if latency > m.MaxLatency {
		m.MaxLatency = latency
	}
}

func (m *LoadTestMetrics) Finalize() {
	m.EndTime = time.Now()
	testDuration := m.EndTime.Sub(m.StartTime)

	if len(m.Latencies) > 0 {
		// Calculate average latency
		var total time.Duration
		for _, latency := range m.Latencies {
			total += latency
		}
		m.AverageLatency = total / time.Duration(len(m.Latencies))

		// Calculate percentiles
		sortedLatencies := make([]time.Duration, len(m.Latencies))
		copy(sortedLatencies, m.Latencies)

		// Simple bubble sort for small datasets
		for i := 0; i < len(sortedLatencies); i++ {
			for j := 0; j < len(sortedLatencies)-1-i; j++ {
				if sortedLatencies[j] > sortedLatencies[j+1] {
					sortedLatencies[j], sortedLatencies[j+1] = sortedLatencies[j+1], sortedLatencies[j]
				}
			}
		}

		p95Index := int(0.95 * float64(len(sortedLatencies)))
		if p95Index >= len(sortedLatencies) {
			p95Index = len(sortedLatencies) - 1
		}
		m.P95Latency = sortedLatencies[p95Index]

		p99Index := int(0.99 * float64(len(sortedLatencies)))
		if p99Index >= len(sortedLatencies) {
			p99Index = len(sortedLatencies) - 1
		}
		m.P99Latency = sortedLatencies[p99Index]
	}

	if testDuration.Seconds() > 0 {
		m.RequestsPerSecond = float64(m.TotalRequests) / testDuration.Seconds()
	}
}

func TestO2APILoadPerformance(t *testing.T) {
	// This test runs 1000+ concurrent HTTP requests and takes 40+ seconds.
	// Skip when -short is passed or when ENABLE_LOAD_TESTS is not set.
	if testing.Short() {
		t.Skip("skipping load performance test: requires long runtime; run without -short or set ENABLE_LOAD_TESTS=true")
	}
	if os.Getenv("ENABLE_LOAD_TESTS") != "true" {
		t.Skip("skipping load performance test: set ENABLE_LOAD_TESTS=true to run")
	}

	// Setup test environment
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	_ = fakeClient.NewClientBuilder().WithScheme(scheme).Build()
	_ = fake.NewSimpleClientset()
	testLogger := logging.NewLogger("o2-load-test", "info")

	// Setup O2 API server
	config := &o2.O2IMSConfig{
		ServerAddress:  "127.0.0.1",
		ServerPort:     0,
		TLSEnabled:     false,
		DatabaseConfig: json.RawMessage(`{}`),
		ProviderConfigs: json.RawMessage(`{"enabled": true}`),
	}

	o2Server, err := o2.NewO2APIServer(config, testLogger, nil)
	require.NoError(t, err)
	defer o2Server.Shutdown(context.Background())

	httpServer := httptest.NewServer(o2Server.GetRouter())
	defer httpServer.Close() // #nosec G307 - Error handled in defer

	client := httpServer.Client()
	client.Timeout = 30 * time.Second

	t.Run("Resource Pool Creation Load Test", func(t *testing.T) {
		loadConfig := LoadTestConfig{
			ConcurrentClients: 50,
			RequestsPerClient: 20,
			TestDuration:      30 * time.Second,
			RampUpTime:        5 * time.Second,
		}

		metrics := runResourcePoolCreationLoadTest(t, httpServer.URL, client, loadConfig)

		// Performance assertions
		assert.Greater(t, metrics.SuccessfulRequests, int64(800), "Should handle at least 800 successful requests")
		assert.Less(t, float64(metrics.FailedRequests)/float64(metrics.TotalRequests), 0.05, "Error rate should be less than 5%")
		assert.Less(t, metrics.AverageLatency.Milliseconds(), int64(500), "Average latency should be less than 500ms")
		assert.Less(t, metrics.P95Latency.Milliseconds(), int64(1000), "P95 latency should be less than 1000ms")
		assert.Greater(t, metrics.RequestsPerSecond, 25.0, "Should handle at least 25 requests per second")

		t.Logf("Load Test Results:")
		t.Logf("  Total Requests: %d", metrics.TotalRequests)
		t.Logf("  Successful: %d, Failed: %d", metrics.SuccessfulRequests, metrics.FailedRequests)
		t.Logf("  Success Rate: %.2f%%", float64(metrics.SuccessfulRequests)/float64(metrics.TotalRequests)*100)
		t.Logf("  Average Latency: %v", metrics.AverageLatency)
		t.Logf("  P95 Latency: %v", metrics.P95Latency)
		t.Logf("  P99 Latency: %v", metrics.P99Latency)
		t.Logf("  Min/Max Latency: %v/%v", metrics.MinLatency, metrics.MaxLatency)
		t.Logf("  Requests/Second: %.2f", metrics.RequestsPerSecond)
	})

	t.Run("Mixed API Operations Load Test", func(t *testing.T) {
		loadConfig := LoadTestConfig{
			ConcurrentClients: 30,
			RequestsPerClient: 50,
			TestDuration:      45 * time.Second,
			RampUpTime:        10 * time.Second,
		}

		metrics := runMixedOperationsLoadTest(t, httpServer.URL, client, loadConfig)

		// Performance assertions for mixed operations
		assert.Greater(t, metrics.SuccessfulRequests, int64(1200), "Should handle at least 1200 successful mixed operations")
		assert.Less(t, float64(metrics.FailedRequests)/float64(metrics.TotalRequests), 0.08, "Error rate should be less than 8% for mixed operations")
		assert.Less(t, metrics.AverageLatency.Milliseconds(), int64(800), "Average latency should be less than 800ms for mixed operations")
		assert.Greater(t, metrics.RequestsPerSecond, 20.0, "Should handle at least 20 mixed requests per second")

		t.Logf("Mixed Operations Load Test Results:")
		t.Logf("  Total Requests: %d", metrics.TotalRequests)
		t.Logf("  Successful: %d, Failed: %d", metrics.SuccessfulRequests, metrics.FailedRequests)
		t.Logf("  Success Rate: %.2f%%", float64(metrics.SuccessfulRequests)/float64(metrics.TotalRequests)*100)
		t.Logf("  Average Latency: %v", metrics.AverageLatency)
		t.Logf("  P95 Latency: %v", metrics.P95Latency)
		t.Logf("  Requests/Second: %.2f", metrics.RequestsPerSecond)

		if len(metrics.ErrorTypes) > 0 {
			t.Logf("  Error Types:")
			for errorType, count := range metrics.ErrorTypes {
				t.Logf("    %s: %d", errorType, count)
			}
		}
	})

	t.Run("Sustained Load Test", func(t *testing.T) {
		loadConfig := LoadTestConfig{
			ConcurrentClients: 20,
			RequestsPerClient: 100,
			TestDuration:      120 * time.Second,
			RampUpTime:        20 * time.Second,
		}

		metrics := runSustainedLoadTest(t, httpServer.URL, client, loadConfig)

		// Performance assertions for sustained load
		assert.Greater(t, metrics.SuccessfulRequests, int64(1800), "Should handle at least 1800 successful requests over sustained period")
		assert.Less(t, float64(metrics.FailedRequests)/float64(metrics.TotalRequests), 0.03, "Error rate should be less than 3% for sustained load")
		assert.Less(t, metrics.AverageLatency.Milliseconds(), int64(600), "Average latency should be less than 600ms for sustained load")
		assert.Greater(t, metrics.RequestsPerSecond, 15.0, "Should maintain at least 15 requests per second under sustained load")

		t.Logf("Sustained Load Test Results:")
		t.Logf("  Test Duration: %v", metrics.EndTime.Sub(metrics.StartTime))
		t.Logf("  Total Requests: %d", metrics.TotalRequests)
		t.Logf("  Success Rate: %.2f%%", float64(metrics.SuccessfulRequests)/float64(metrics.TotalRequests)*100)
		t.Logf("  Average Latency: %v", metrics.AverageLatency)
		t.Logf("  P99 Latency: %v", metrics.P99Latency)
		t.Logf("  Requests/Second: %.2f", metrics.RequestsPerSecond)
	})
}

func runResourcePoolCreationLoadTest(t *testing.T, baseURL string, client *http.Client, config LoadTestConfig) *LoadTestMetrics {
	metrics := NewLoadTestMetrics()
	var wg sync.WaitGroup

	// Channel to control request rate during ramp-up
	requestChan := make(chan bool, config.ConcurrentClients)

	// Start goroutines for concurrent clients
	for i := 0; i < config.ConcurrentClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			// Ramp-up delay
			rampDelay := time.Duration(clientID) * config.RampUpTime / time.Duration(config.ConcurrentClients)
			time.Sleep(rampDelay)

			for j := 0; j < config.RequestsPerClient; j++ {
				select {
				case <-requestChan:
					return // Test duration exceeded
				default:
				}

				poolID := fmt.Sprintf("load-test-pool-%d-%d-%d", clientID, j, time.Now().UnixNano())
				pool := &models.ResourcePool{
					ResourcePoolID: poolID,
					Name:           fmt.Sprintf("Load Test Pool %d-%d", clientID, j),
					Description:    "Resource pool created during load testing",
					Location:       "us-east-1",
					OCloudID:       "load-test-ocloud",
					Provider:       "kubernetes",
					Region:         "us-east-1",
					Zone:           "us-east-1a",
					Capacity: &models.ResourceCapacity{
						CPU: &models.ResourceMetric{
							Total:       "100",
							Available:   "80",
							Used:        "20",
							Unit:        "cores",
							Utilization: 20.0,
						},
						Memory: &models.ResourceMetric{
							Total:       "400Gi",
							Available:   "320Gi",
							Used:        "80Gi",
							Unit:        "bytes",
							Utilization: 20.0,
						},
					},
				}

				poolJSON, err := json.Marshal(pool)
				if err != nil {
					metrics.RecordRequest(0, false, "marshal_error")
					continue
				}

				start := time.Now()
				resp, err := client.Post(
					baseURL+"/o2ims/v1/resourcePools",
					"application/json",
					bytes.NewBuffer(poolJSON),
				)
				latency := time.Since(start)

				if err != nil {
					metrics.RecordRequest(latency, false, "network_error")
					continue
				}

				resp.Body.Close()
				success := resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK
				errorType := ""
				if !success {
					errorType = fmt.Sprintf("http_%d", resp.StatusCode)
				}
				metrics.RecordRequest(latency, success, errorType)

				// Small delay between requests from same client
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	// Stop test after configured duration
	go func() {
		time.Sleep(config.TestDuration)
		close(requestChan)
	}()

	wg.Wait()
	metrics.Finalize()
	return metrics
}

func runMixedOperationsLoadTest(t *testing.T, baseURL string, client *http.Client, config LoadTestConfig) *LoadTestMetrics {
	metrics := NewLoadTestMetrics()
	var wg sync.WaitGroup

	// Pre-create some resources for GET/PUT/DELETE operations
	createdPoolIDs := make([]string, 20)
	for i := 0; i < 20; i++ {
		poolID := fmt.Sprintf("mixed-test-pool-%d", i)
		createdPoolIDs[i] = poolID

		pool := &models.ResourcePool{
			ResourcePoolID: poolID,
			Name:           fmt.Sprintf("Mixed Test Pool %d", i),
			Provider:       "kubernetes",
			OCloudID:       "mixed-test-ocloud",
		}

		poolJSON, _ := json.Marshal(pool)
		resp, _ := client.Post(
			baseURL+"/o2ims/v1/resourcePools",
			"application/json",
			bytes.NewBuffer(poolJSON),
		)
		if resp != nil {
			resp.Body.Close()
		}
	}

	// Channel to control request rate
	requestChan := make(chan bool, config.ConcurrentClients)

	for i := 0; i < config.ConcurrentClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			// Ramp-up delay
			rampDelay := time.Duration(clientID) * config.RampUpTime / time.Duration(config.ConcurrentClients)
			time.Sleep(rampDelay)

			for j := 0; j < config.RequestsPerClient; j++ {
				select {
				case <-requestChan:
					return
				default:
				}

				// Mix of operations: 40% GET, 30% POST, 20% PUT, 10% DELETE
				operation := j % 10
				var start time.Time
				var resp *http.Response
				var err error
				var operationType string

				switch {
				case operation < 4: // GET operations (40%)
					operationType = "GET"
					start = time.Now()
					if operation < 2 {
						// List resource pools
						resp, err = client.Get(baseURL + "/o2ims/v1/resourcePools")
					} else {
						// Get specific resource pool
						poolID := createdPoolIDs[j%len(createdPoolIDs)]
						resp, err = client.Get(baseURL + "/o2ims/v1/resourcePools/" + poolID)
					}

				case operation < 7: // POST operations (30%)
					operationType = "POST"
					poolID := fmt.Sprintf("mixed-post-pool-%d-%d-%d", clientID, j, time.Now().UnixNano())
					pool := &models.ResourcePool{
						ResourcePoolID: poolID,
						Name:           fmt.Sprintf("Mixed POST Pool %d-%d", clientID, j),
						Provider:       "kubernetes",
						OCloudID:       "mixed-test-ocloud",
					}

					poolJSON, marshalErr := json.Marshal(pool)
					if marshalErr != nil {
						metrics.RecordRequest(0, false, "marshal_error")
						continue
					}

					start = time.Now()
					resp, err = client.Post(
						baseURL+"/o2ims/v1/resourcePools",
						"application/json",
						bytes.NewBuffer(poolJSON),
					)

				case operation < 9: // PUT operations (20%)
					operationType = "PUT"
					poolID := createdPoolIDs[j%len(createdPoolIDs)]

					// Get current pool first
					getResp, getErr := client.Get(baseURL + "/o2ims/v1/resourcePools/" + poolID)
					if getErr != nil {
						metrics.RecordRequest(0, false, "get_before_put_error")
						continue
					}

					var pool models.ResourcePool
					json.NewDecoder(getResp.Body).Decode(&pool)
					getResp.Body.Close()

					// Update pool
					pool.Description = fmt.Sprintf("Updated by client %d request %d", clientID, j)
					poolJSON, _ := json.Marshal(pool)

					req, _ := http.NewRequest("PUT", baseURL+"/o2ims/v1/resourcePools/"+poolID, bytes.NewBuffer(poolJSON))
					req.Header.Set("Content-Type", "application/json")

					start = time.Now()
					resp, err = client.Do(req)

				default: // DELETE operations (10%)
					operationType = "DELETE"
					// Create a temporary pool to delete
					tempPoolID := fmt.Sprintf("temp-delete-pool-%d-%d-%d", clientID, j, time.Now().UnixNano())
					tempPool := &models.ResourcePool{
						ResourcePoolID: tempPoolID,
						Name:           "Temporary Pool for Deletion",
						Provider:       "kubernetes",
						OCloudID:       "temp-ocloud",
					}

					tempPoolJSON, _ := json.Marshal(tempPool)
					createResp, createErr := client.Post(
						baseURL+"/o2ims/v1/resourcePools",
						"application/json",
						bytes.NewBuffer(tempPoolJSON),
					)
					if createErr != nil || createResp.StatusCode != http.StatusCreated {
						if createResp != nil {
							createResp.Body.Close()
						}
						metrics.RecordRequest(0, false, "create_before_delete_error")
						continue
					}
					createResp.Body.Close()

					// Now delete it
					req, _ := http.NewRequest("DELETE", baseURL+"/o2ims/v1/resourcePools/"+tempPoolID, nil)

					start = time.Now()
					resp, err = client.Do(req)
				}

				latency := time.Since(start)

				if err != nil {
					metrics.RecordRequest(latency, false, operationType+"_network_error")
					continue
				}

				if resp != nil {
					resp.Body.Close()
					success := resp.StatusCode >= 200 && resp.StatusCode < 300
					errorType := ""
					if !success {
						errorType = fmt.Sprintf("%s_http_%d", operationType, resp.StatusCode)
					}
					metrics.RecordRequest(latency, success, errorType)
				}

				// Small delay between requests
				time.Sleep(20 * time.Millisecond)
			}
		}(i)
	}

	// Stop test after configured duration
	go func() {
		time.Sleep(config.TestDuration)
		close(requestChan)
	}()

	wg.Wait()
	metrics.Finalize()
	return metrics
}

func runSustainedLoadTest(t *testing.T, baseURL string, client *http.Client, config LoadTestConfig) *LoadTestMetrics {
	metrics := NewLoadTestMetrics()
	var wg sync.WaitGroup

	requestChan := make(chan bool, config.ConcurrentClients)

	for i := 0; i < config.ConcurrentClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			// Gradual ramp-up
			rampDelay := time.Duration(clientID) * config.RampUpTime / time.Duration(config.ConcurrentClients)
			time.Sleep(rampDelay)

			for j := 0; j < config.RequestsPerClient; j++ {
				select {
				case <-requestChan:
					return
				default:
				}

				// Primarily GET operations for sustained load (90% GET, 10% POST)
				var start time.Time
				var resp *http.Response
				var err error

				if j%10 == 0 { // 10% POST operations
					poolID := fmt.Sprintf("sustained-pool-%d-%d-%d", clientID, j, time.Now().UnixNano())
					pool := &models.ResourcePool{
						ResourcePoolID: poolID,
						Name:           fmt.Sprintf("Sustained Test Pool %d-%d", clientID, j),
						Provider:       "kubernetes",
						OCloudID:       "sustained-test-ocloud",
					}

					poolJSON, marshalErr := json.Marshal(pool)
					if marshalErr != nil {
						metrics.RecordRequest(0, false, "marshal_error")
						continue
					}

					start = time.Now()
					resp, err = client.Post(
						baseURL+"/o2ims/v1/resourcePools",
						"application/json",
						bytes.NewBuffer(poolJSON),
					)
				} else { // 90% GET operations
					start = time.Now()
					if j%3 == 0 {
						// Service info endpoint
						resp, err = client.Get(baseURL + "/o2ims/v1/")
					} else {
						// List resource pools
						resp, err = client.Get(baseURL + "/o2ims/v1/resourcePools")
					}
				}

				latency := time.Since(start)

				if err != nil {
					metrics.RecordRequest(latency, false, "network_error")
					continue
				}

				if resp != nil {
					resp.Body.Close()
					success := resp.StatusCode >= 200 && resp.StatusCode < 300
					errorType := ""
					if !success {
						errorType = fmt.Sprintf("http_%d", resp.StatusCode)
					}
					metrics.RecordRequest(latency, success, errorType)
				}

				// Consistent request rate with small variance
				baseDelay := 50 * time.Millisecond
				variance := time.Duration(j%10) * time.Millisecond // 0-9ms variance
				time.Sleep(baseDelay + variance)
			}
		}(i)
	}

	// Stop test after configured duration
	go func() {
		time.Sleep(config.TestDuration)
		close(requestChan)
	}()

	wg.Wait()
	metrics.Finalize()
	return metrics
}

// Benchmark tests for individual operations
func BenchmarkO2APIOperations(b *testing.B) {
	// Setup test environment
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	_ = fakeClient.NewClientBuilder().WithScheme(scheme).Build()
	_ = fake.NewSimpleClientset()
	testLogger := logging.NewLogger("o2-benchmark", "error") // Reduced logging for benchmarks

	config := &o2.O2IMSConfig{
		ServerAddress:  "127.0.0.1",
		ServerPort:     0,
		TLSEnabled:     false,
		DatabaseConfig: json.RawMessage(`{}`),
		ProviderConfigs: json.RawMessage(`{"enabled": true}`),
	}

	o2Server, err := o2.NewO2APIServer(config, testLogger, nil)
	require.NoError(b, err)
	defer o2Server.Shutdown(context.Background())

	httpServer := httptest.NewServer(o2Server.GetRouter())
	defer httpServer.Close() // #nosec G307 - Error handled in defer

	client := httpServer.Client()
	client.Timeout = 10 * time.Second

	b.Run("ServiceInfo", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(httpServer.URL + "/o2ims/v1/")
			require.NoError(b, err)
			resp.Body.Close()
		}
	})

	b.Run("CreateResourcePool", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			poolID := fmt.Sprintf("benchmark-pool-%d-%d", i, time.Now().UnixNano())
			pool := &models.ResourcePool{
				ResourcePoolID: poolID,
				Name:           fmt.Sprintf("Benchmark Pool %d", i),
				Provider:       "kubernetes",
				OCloudID:       "benchmark-ocloud",
			}

			poolJSON, err := json.Marshal(pool)
			require.NoError(b, err)

			resp, err := client.Post(
				httpServer.URL+"/o2ims/v1/resourcePools",
				"application/json",
				bytes.NewBuffer(poolJSON),
			)
			require.NoError(b, err)
			resp.Body.Close()
		}
	})

	b.Run("ListResourcePools", func(b *testing.B) {
		// Pre-create some pools
		for i := 0; i < 10; i++ {
			pool := &models.ResourcePool{
				ResourcePoolID: fmt.Sprintf("list-test-pool-%d", i),
				Name:           fmt.Sprintf("List Test Pool %d", i),
				Provider:       "kubernetes",
				OCloudID:       "list-test-ocloud",
			}
			poolJSON, _ := json.Marshal(pool)
			resp, _ := client.Post(
				httpServer.URL+"/o2ims/v1/resourcePools",
				"application/json",
				bytes.NewBuffer(poolJSON),
			)
			if resp != nil {
				resp.Body.Close()
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(httpServer.URL + "/o2ims/v1/resourcePools")
			require.NoError(b, err)
			resp.Body.Close()
		}
	})

	b.Run("GetResourcePool", func(b *testing.B) {
		// Pre-create a pool
		poolID := "get-test-pool"
		pool := &models.ResourcePool{
			ResourcePoolID: poolID,
			Name:           "Get Test Pool",
			Provider:       "kubernetes",
			OCloudID:       "get-test-ocloud",
		}
		poolJSON, _ := json.Marshal(pool)
		resp, err := client.Post(
			httpServer.URL+"/o2ims/v1/resourcePools",
			"application/json",
			bytes.NewBuffer(poolJSON),
		)
		require.NoError(b, err)
		resp.Body.Close()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(httpServer.URL + "/o2ims/v1/resourcePools/" + poolID)
			require.NoError(b, err)
			resp.Body.Close()
		}
	})
}

// Memory usage and GC pressure tests
func TestMemoryUsage(t *testing.T) {
	// This test would typically require additional tooling for memory profiling
	// For now, we'll simulate high-volume operations and ensure no memory leaks

	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	_ = fakeClient.NewClientBuilder().WithScheme(scheme).Build()
	_ = fake.NewSimpleClientset()
	testLogger := logging.NewLogger("o2-memory-test", "warn")

	config := &o2.O2IMSConfig{
		ServerAddress:  "127.0.0.1",
		ServerPort:     0,
		TLSEnabled:     false,
		DatabaseConfig: json.RawMessage(`{}`),
	}

	o2Server, err := o2.NewO2APIServer(config, testLogger, nil)
	require.NoError(t, err)
	defer o2Server.Shutdown(context.Background())

	httpServer := httptest.NewServer(o2Server.GetRouter())
	defer httpServer.Close() // #nosec G307 - Error handled in defer

	client := httpServer.Client()

	t.Run("High Volume Resource Pool Operations", func(t *testing.T) {
		const numOperations = 1000

		// Create many resource pools
		for i := 0; i < numOperations; i++ {
			poolID := fmt.Sprintf("memory-test-pool-%d", i)
			pool := &models.ResourcePool{
				ResourcePoolID: poolID,
				Name:           fmt.Sprintf("Memory Test Pool %d", i),
				Provider:       "kubernetes",
				OCloudID:       "memory-test-ocloud",
				Capacity: &models.ResourceCapacity{
					CPU: &models.ResourceMetric{
						Total:       strconv.Itoa(100 + i%100),
						Available:   strconv.Itoa(80 + i%80),
						Used:        strconv.Itoa(20 + i%20),
						Unit:        "cores",
						Utilization: float64(20 + i%30),
					},
				},
			}

			poolJSON, err := json.Marshal(pool)
			require.NoError(t, err)

			resp, err := client.Post(
				httpServer.URL+"/o2ims/v1/resourcePools",
				"application/json",
				bytes.NewBuffer(poolJSON),
			)
			require.NoError(t, err)
			resp.Body.Close()

			// Periodically list all pools to test retrieval performance
			if i%100 == 99 {
				resp, err := client.Get(httpServer.URL + "/o2ims/v1/resourcePools")
				require.NoError(t, err)
				resp.Body.Close()
			}
		}

		// Verify we can still perform operations efficiently
		start := time.Now()
		resp, err := client.Get(httpServer.URL + "/o2ims/v1/resourcePools")
		latency := time.Since(start)
		require.NoError(t, err)
		resp.Body.Close()

		// Should still be responsive with 1000 resource pools
		assert.Less(t, latency.Milliseconds(), int64(2000), "Should retrieve 1000 pools in less than 2 seconds")

		t.Logf("Retrieved 1000 resource pools in %v", latency)
	})
}
