package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
<<<<<<< HEAD
	"sync"
=======
	"strings"
	"sync"
	"sync/atomic"
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// BenchmarkHTTPHandler_IngestEndpoint benchmarks the ingest endpoint
func BenchmarkHTTPHandler_IngestEndpoint(b *testing.B) {
	tempDir := b.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")

	server := createBenchmarkServer(b, handoffDir)
	defer server.Close() // #nosec G307 - Error handled in defer

	intent := map[string]interface{}{
<<<<<<< HEAD
		"intent": "Deploy nginx with 3 replicas",
=======
		"intent_type": "scaling",
		"target": "nginx",
		"namespace": "default",
		"replicas": 3,
		"source": "test",
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	}

	intentJSON, err := json.Marshal(intent)
	require.NoError(b, err)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := http.Post(server.URL+"/ingest", "application/json", bytes.NewBuffer(intentJSON))
			if err != nil {
				b.Fatal(err)
			}
<<<<<<< HEAD
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
=======
			_, _ = io.Copy(io.Discard, resp.Body) // #nosec G104 - Test discard
			_ = resp.Body.Close() // #nosec G104 - Test cleanup
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		}
	})
}

// BenchmarkHTTPHandler_HealthEndpoint benchmarks the health endpoint
func BenchmarkHTTPHandler_HealthEndpoint(b *testing.B) {
	tempDir := b.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")

	server := createBenchmarkServer(b, handoffDir)
	defer server.Close() // #nosec G307 - Error handled in defer

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := http.Get(server.URL + "/health")
			if err != nil {
				b.Fatal(err)
			}
<<<<<<< HEAD
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
=======
			_, _ = io.Copy(io.Discard, resp.Body) // #nosec G104 - Test discard
			_ = resp.Body.Close() // #nosec G104 - Test cleanup
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		}
	})
}

// BenchmarkHTTPHandler_MetricsEndpoint benchmarks the metrics endpoint
func BenchmarkHTTPHandler_MetricsEndpoint(b *testing.B) {
	tempDir := b.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")

	server := createBenchmarkServer(b, handoffDir)
	defer server.Close() // #nosec G307 - Error handled in defer

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := http.Get(server.URL + "/metrics")
			if err != nil {
				b.Fatal(err)
			}
<<<<<<< HEAD
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
=======
			_, _ = io.Copy(io.Discard, resp.Body) // #nosec G104 - Test discard
			_ = resp.Body.Close() // #nosec G104 - Test cleanup
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		}
	})
}

// BenchmarkJSONMarshal benchmarks JSON marshaling performance
func BenchmarkJSONMarshal(b *testing.B) {
	intent := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      "test-intent",
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"cpu":    "100m",
			"memory": "128Mi",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(intent)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkJSONUnmarshal benchmarks JSON unmarshaling performance
func BenchmarkJSONUnmarshal(b *testing.B) {
	intentJSON := []byte(`{
		"apiVersion": "intent.nephoran.com/v1alpha1",
		"kind": "NetworkIntent",
		"metadata": {
			"name": "test-intent",
			"namespace": "default"
		},
		"spec": {
			"intent": "Deploy nginx with 3 replicas",
			"replicas": 3,
			"resources": {
				"cpu": "100m",
				"memory": "128Mi"
			}
		}
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var intent map[string]interface{}
		err := json.Unmarshal(intentJSON, &intent)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkConcurrentRequests benchmarks concurrent request handling
func BenchmarkConcurrentRequests(b *testing.B) {
	tempDir := b.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")

	server := createBenchmarkServer(b, handoffDir)
	defer server.Close() // #nosec G307 - Error handled in defer

	intent := map[string]interface{}{
<<<<<<< HEAD
		"intent": "Benchmark concurrent processing",
=======
		"intent_type": "scaling",
		"target": "benchmark-app",
		"namespace": "default",
		"replicas": 2,
		"source": "test",
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	}

	intentJSON, err := json.Marshal(intent)
	require.NoError(b, err)

	// Test different levels of concurrency
	concurrencyLevels := []int{1, 10, 100}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency-%d", concurrency), func(b *testing.B) {
			sem := make(chan struct{}, concurrency)
			var wg sync.WaitGroup

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sem <- struct{}{}
				wg.Add(1)

				go func() {
					defer func() {
						<-sem
						wg.Done()
					}()

					resp, err := http.Post(server.URL+"/ingest", "application/json", bytes.NewBuffer(intentJSON))
					if err != nil {
						b.Error(err)
						return
					}
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}()
			}

			wg.Wait()
		})
	}
}

// BenchmarkMemoryUsage benchmarks memory usage patterns
func BenchmarkMemoryUsage(b *testing.B) {
	tempDir := b.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")

	server := createBenchmarkServer(b, handoffDir)
	defer server.Close() // #nosec G307 - Error handled in defer

	// Create large intent payload to test memory usage
	largeIntent := map[string]interface{}{
<<<<<<< HEAD
		"intent":   "Large intent for memory testing",
		"metadata": make(map[string]interface{}),
	}

	// Add many fields to create a large payload
=======
		"intent": "Large intent for memory testing",
		"spec": map[string]interface{}{
			"metadata": make(map[string]interface{}),
		},
	}

	// Add many fields to create a large payload
	// Fixed: properly access the nested metadata field
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	metadata := largeIntent["spec"].(map[string]interface{})["metadata"].(map[string]interface{})
	for i := 0; i < 1000; i++ {
		metadata[fmt.Sprintf("field_%d", i)] = fmt.Sprintf("value_%d", i)
	}

	intentJSON, err := json.Marshal(largeIntent)
	require.NoError(b, err)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		resp, err := http.Post(server.URL+"/ingest", "application/json", bytes.NewBuffer(intentJSON))
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// BenchmarkResponseTime benchmarks response time distribution
func BenchmarkResponseTime(b *testing.B) {
	tempDir := b.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")

	server := createBenchmarkServer(b, handoffDir)
	defer server.Close() // #nosec G307 - Error handled in defer

	intent := map[string]interface{}{
<<<<<<< HEAD
		"intent": "Response time benchmark",
=======
		"intent_type": "scaling",
		"target": "response-app",
		"namespace": "default",
		"replicas": 1,
		"source": "test",
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	}

	intentJSON, err := json.Marshal(intent)
	require.NoError(b, err)

	var totalDuration time.Duration
	var maxDuration time.Duration
	minDuration := time.Hour // Initialize to high value

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()

		resp, err := http.Post(server.URL+"/ingest", "application/json", bytes.NewBuffer(intentJSON))
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		duration := time.Since(start)
		totalDuration += duration

		if duration > maxDuration {
			maxDuration = duration
		}
		if duration < minDuration {
			minDuration = duration
		}
	}

	avgDuration := totalDuration / time.Duration(b.N)
	b.ReportMetric(float64(avgDuration.Nanoseconds()), "avg_ns/op")
	b.ReportMetric(float64(minDuration.Nanoseconds()), "min_ns/op")
	b.ReportMetric(float64(maxDuration.Nanoseconds()), "max_ns/op")
}

// BenchmarkThroughput benchmarks request throughput
func BenchmarkThroughput(b *testing.B) {
	tempDir := b.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")

	server := createBenchmarkServer(b, handoffDir)
	defer server.Close() // #nosec G307 - Error handled in defer

	intent := map[string]interface{}{
<<<<<<< HEAD
		"intent": "Throughput benchmark",
=======
		"intent_type": "scaling",
		"target": "throughput-app",
		"namespace": "default",
		"replicas": 1,
		"source": "test",
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	}

	intentJSON, err := json.Marshal(intent)
	require.NoError(b, err)

	// Test throughput over time
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var requestCount int64
	var wg sync.WaitGroup

	// Spawn multiple workers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					resp, err := http.Post(server.URL+"/ingest", "application/json", bytes.NewBuffer(intentJSON))
					if err != nil {
						continue
					}
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
<<<<<<< HEAD
					requestCount++
=======
					// Use atomic operation to safely increment counter across goroutines
					atomic.AddInt64(&requestCount, 1)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
				}
			}
		}()
	}

	wg.Wait()

<<<<<<< HEAD
	throughput := float64(requestCount) / 5.0 // requests per second
=======
	// Use atomic Load to safely read the final value
	finalCount := atomic.LoadInt64(&requestCount)
	throughput := float64(finalCount) / 5.0 // requests per second
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	b.ReportMetric(throughput, "req/sec")
}

// createBenchmarkServer creates a lightweight server for benchmarking
<<<<<<< HEAD
=======
// 
// Thread Safety: This function creates handlers that may be called concurrently.
// The requestCounter variable is protected using atomic operations (atomic.AddInt64
// and atomic.LoadInt64) to prevent race conditions when multiple goroutines
// increment or read the counter simultaneously.
//
// Race Detection: To run tests with race detection enabled:
//   - Linux CI: CGO_ENABLED=1 go test -race ./...
//   - The CI workflows already have CGO_ENABLED=1 configured for race detection
>>>>>>> 6835433495e87288b95961af7173d866977175ff
func createBenchmarkServer(tb testing.TB, handoffDir string) *httptest.Server {
	tb.Helper()

	// Create a more realistic server for benchmarking
	mux := http.NewServeMux()

<<<<<<< HEAD
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
=======
	// Declare request counter as an atomic variable at the top
	// to be accessible from all handlers
	// IMPORTANT: Use atomic operations (atomic.AddInt64, atomic.LoadInt64) 
	// when accessing this variable to prevent race conditions
	var requestCounter int64

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`)) // #nosec G104 - Test response
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	})

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)

<<<<<<< HEAD
=======
		// Use atomic Load to safely read the current request count
		currentCount := atomic.LoadInt64(&requestCounter)
		
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		metrics := fmt.Sprintf(`# HELP intent_ingest_requests_total Total number of ingested intents
# TYPE intent_ingest_requests_total counter
intent_ingest_requests_total %d

# HELP intent_ingest_request_duration_seconds Duration of intent ingestion requests
# TYPE intent_ingest_request_duration_seconds histogram
intent_ingest_request_duration_seconds_bucket{le="0.001"} %d
intent_ingest_request_duration_seconds_bucket{le="0.01"} %d
intent_ingest_request_duration_seconds_bucket{le="0.1"} %d
intent_ingest_request_duration_seconds_bucket{le="+Inf"} %d
intent_ingest_request_duration_seconds_sum %f
intent_ingest_request_duration_seconds_count %d
`,
<<<<<<< HEAD
			time.Now().Unix(), // Simulated counter
=======
			currentCount, // Use actual request counter value
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			10, 50, 100, 100,  // Simulated histogram buckets
			float64(time.Now().UnixNano())/1e9, // Simulated sum
			100,                                // Simulated count
		)

<<<<<<< HEAD
		w.Write([]byte(metrics))
	})

	var requestCounter int64
=======
		_, _ = w.Write([]byte(metrics)) // #nosec G104 - Test response
	})

>>>>>>> 6835433495e87288b95961af7173d866977175ff
	mux.HandleFunc("/ingest", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Simulate processing time
		time.Sleep(time.Microsecond * 100)

		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var intent map[string]interface{}
		if err := json.Unmarshal(body, &intent); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

<<<<<<< HEAD
		// Simulate file creation without actually writing to disk for performance
		requestCounter++
=======
		// Use atomic operation to safely increment counter across goroutines
		// This prevents race conditions in high concurrency scenarios
		atomic.AddInt64(&requestCounter, 1)
>>>>>>> 6835433495e87288b95961af7173d866977175ff

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := json.RawMessage(`{}`)

		json.NewEncoder(w).Encode(response)
	})

	return httptest.NewServer(mux)
}

// FuzzHTTPIngest fuzzes the ingest endpoint with various inputs
func FuzzHTTPIngest(f *testing.F) {
	tempDir := f.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")

	server := createBenchmarkServer(f, handoffDir)
	defer server.Close() // #nosec G307 - Error handled in defer

	// Add seed corpus
	f.Add(`{"spec":{"intent":"test"}}`)
	f.Add(`{"apiVersion":"intent.nephoran.com/v1alpha1","kind":"NetworkIntent"}`)
	f.Add(`{}`)
	f.Add(`{"invalid":"json"`)
	f.Add(`null`)
	f.Add(`""`)

	f.Fuzz(func(t *testing.T, input string) {
		resp, err := http.Post(server.URL+"/ingest", "application/json", bytes.NewBufferString(input))
		if err != nil {
			return // Network errors are acceptable in fuzzing
		}
		defer resp.Body.Close() // #nosec G307 - Error handled in defer

		// Read response to ensure no panic
		io.Copy(io.Discard, resp.Body)

		// Valid responses should be 200, 400, or 500
		validStatusCodes := []int{
			http.StatusOK,
			http.StatusBadRequest,
			http.StatusInternalServerError,
		}

		validStatus := false
		for _, code := range validStatusCodes {
			if resp.StatusCode == code {
				validStatus = true
				break
			}
		}

		if !validStatus {
			t.Errorf("Unexpected status code: %d", resp.StatusCode)
		}
	})
}

<<<<<<< HEAD
=======
// TestConcurrentCounterAccess verifies that request counter is thread-safe
func TestConcurrentCounterAccess(t *testing.T) {
	tempDir := t.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")
	
	server := createBenchmarkServer(t, handoffDir)
	defer server.Close()
	
	const numGoroutines = 50
	const requestsPerGoroutine = 20
	
	var wg sync.WaitGroup
	
	// Test concurrent POST requests to /ingest
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			
			intent := map[string]interface{}{
				"intent_type": "scaling",
				"target":      fmt.Sprintf("app-%d", id),
				"namespace":   "default",
				"replicas":    1,
				"source":      "test",
			}
			
			intentJSON, _ := json.Marshal(intent)
			
			for j := 0; j < requestsPerGoroutine; j++ {
				resp, err := http.Post(server.URL+"/ingest", "application/json", bytes.NewBuffer(intentJSON))
				if err != nil {
					// Don't fail the test for connection errors in high concurrency
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}(i)
	}
	
	// Test concurrent GET requests to /metrics (which reads the counter)
	// Run these in parallel to test race conditions between reads and writes
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			
			for j := 0; j < 5; j++ {
				resp, err := http.Get(server.URL + "/metrics")
				if err != nil {
					// Don't fail the test for connection errors
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				
				// Small delay to spread out requests
				time.Sleep(time.Millisecond * 10)
			}
		}()
	}
	
	wg.Wait()
	
	// Give a small delay to ensure all requests are processed
	time.Sleep(time.Millisecond * 100)
	
	// Verify final counter value through metrics endpoint
	resp, err := http.Get(server.URL + "/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	
	// The metrics should contain a reasonable request count
	// Due to potential connection failures, we check that the counter is > 0
	// and that it doesn't exceed the maximum possible value
	metricsStr := string(body)
	maxCount := numGoroutines * requestsPerGoroutine
	
	// Parse the actual count from metrics
	if strings.Contains(metricsStr, "intent_ingest_requests_total") {
		// Extract the number after "intent_ingest_requests_total "
		lines := strings.Split(metricsStr, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "intent_ingest_requests_total ") {
				var count int64
				_, err := fmt.Sscanf(line, "intent_ingest_requests_total %d", &count)
				if err == nil {
					if count > 0 && count <= int64(maxCount) {
						t.Logf("Counter value is %d (max possible: %d)", count, maxCount)
					} else {
						t.Errorf("Counter value %d is outside expected range (1-%d)", count, maxCount)
					}
				}
				break
			}
		}
	} else {
		t.Error("Metrics endpoint did not return request counter")
		t.Logf("Metrics output:\n%s", metricsStr)
	}
}

>>>>>>> 6835433495e87288b95961af7173d866977175ff
// Test specific performance characteristics
func TestPerformanceCharacteristics(t *testing.T) {
	tempDir := t.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")

	server := createBenchmarkServer(t, handoffDir)
	defer server.Close() // #nosec G307 - Error handled in defer

	t.Run("response time is under 100ms", func(t *testing.T) {
		intent := map[string]interface{}{
<<<<<<< HEAD
			"intent": "Performance test intent",
=======
			"intent_type": "scaling",
			"target": "perf-test-app",
			"namespace": "default",
			"replicas": 1,
			"source": "test",
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		}

		intentJSON, err := json.Marshal(intent)
		require.NoError(t, err)

		start := time.Now()
		resp, err := http.Post(server.URL+"/ingest", "application/json", bytes.NewBuffer(intentJSON))
		duration := time.Since(start)

		require.NoError(t, err)
		defer resp.Body.Close() // #nosec G307 - Error handled in defer

		if duration > 100*time.Millisecond {
			t.Errorf("Response time %v exceeds 100ms threshold", duration)
		}
	})

	t.Run("handles high concurrency without errors", func(t *testing.T) {
		const concurrency = 50
		const requestsPerWorker = 10

		intent := map[string]interface{}{
<<<<<<< HEAD
			"intent": "Concurrency test intent",
=======
			"intent_type": "scaling",
			"target": "concurrency-app",
			"namespace": "default",
			"replicas": 1,
			"source": "test",
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		}

		intentJSON, err := json.Marshal(intent)
		require.NoError(t, err)

		var wg sync.WaitGroup
		errorCh := make(chan error, concurrency*requestsPerWorker)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < requestsPerWorker; j++ {
					resp, err := http.Post(server.URL+"/ingest", "application/json", bytes.NewBuffer(intentJSON))
					if err != nil {
						errorCh <- err
						continue
					}

					if resp.StatusCode != http.StatusOK {
						errorCh <- fmt.Errorf("worker %d request %d: unexpected status %d", workerID, j, resp.StatusCode)
					}

					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}
			}(i)
		}

		wg.Wait()
		close(errorCh)

		var errors []error
		for err := range errorCh {
			errors = append(errors, err)
		}

		if len(errors) > 0 {
			t.Errorf("Found %d errors during concurrent execution: %v", len(errors), errors[0])
		}
	})
}
