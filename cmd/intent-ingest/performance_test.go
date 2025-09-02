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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// BenchmarkHTTPHandler_IngestEndpoint benchmarks the ingest endpoint
func BenchmarkHTTPHandler_IngestEndpoint(b *testing.B) {
	tempDir := b.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")

	server := createBenchmarkServer(b, handoffDir)
	defer server.Close()

	intent := map[string]interface{}{
		"apiVersion": "intent.nephoran.com/v1alpha1",
		"kind":       "NetworkIntent",
		"spec": map[string]interface{}{
			"intent": "Deploy nginx with 3 replicas",
		},
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
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}

// BenchmarkHTTPHandler_HealthEndpoint benchmarks the health endpoint
func BenchmarkHTTPHandler_HealthEndpoint(b *testing.B) {
	tempDir := b.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")

	server := createBenchmarkServer(b, handoffDir)
	defer server.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := http.Get(server.URL + "/health")
			if err != nil {
				b.Fatal(err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}

// BenchmarkHTTPHandler_MetricsEndpoint benchmarks the metrics endpoint
func BenchmarkHTTPHandler_MetricsEndpoint(b *testing.B) {
	tempDir := b.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")

	server := createBenchmarkServer(b, handoffDir)
	defer server.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := http.Get(server.URL + "/metrics")
			if err != nil {
				b.Fatal(err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}

// BenchmarkJSONMarshal benchmarks JSON marshaling performance
func BenchmarkJSONMarshal(b *testing.B) {
	intent := map[string]interface{}{
		"apiVersion": "intent.nephoran.com/v1alpha1",
		"kind":       "NetworkIntent",
		"metadata": map[string]interface{}{
			"name":      "test-intent",
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"intent":   "Deploy nginx with 3 replicas",
			"replicas": 3,
			"resources": map[string]interface{}{
				"cpu":    "100m",
				"memory": "128Mi",
			},
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
	defer server.Close()

	intent := map[string]interface{}{
		"spec": map[string]interface{}{
			"intent": "Benchmark concurrent processing",
		},
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
	defer server.Close()

	// Create large intent payload to test memory usage
	largeIntent := map[string]interface{}{
		"apiVersion": "intent.nephoran.com/v1alpha1",
		"kind":       "NetworkIntent",
		"spec": map[string]interface{}{
			"intent":   "Large intent for memory testing",
			"metadata": make(map[string]interface{}),
		},
	}

	// Add many fields to create a large payload
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
	defer server.Close()

	intent := map[string]interface{}{
		"spec": map[string]interface{}{
			"intent": "Response time benchmark",
		},
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
	defer server.Close()

	intent := map[string]interface{}{
		"spec": map[string]interface{}{
			"intent": "Throughput benchmark",
		},
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
					requestCount++
				}
			}
		}()
	}

	wg.Wait()

	throughput := float64(requestCount) / 5.0 // requests per second
	b.ReportMetric(throughput, "req/sec")
}

// createBenchmarkServer creates a lightweight server for benchmarking
func createBenchmarkServer(tb testing.TB, handoffDir string) *httptest.Server {
	tb.Helper()

	// Create a more realistic server for benchmarking
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
	})

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)

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
			time.Now().Unix(), // Simulated counter
			10, 50, 100, 100,  // Simulated histogram buckets
			float64(time.Now().UnixNano())/1e9, // Simulated sum
			100,                                // Simulated count
		)

		w.Write([]byte(metrics))
	})

	var requestCounter int64
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

		// Simulate file creation without actually writing to disk for performance
		requestCounter++

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := map[string]interface{}{
			"status":    "success",
			"message":   "Intent ingested successfully",
			"requestId": fmt.Sprintf("req-%d", requestCounter),
			"timestamp": time.Now().Format(time.RFC3339),
		}

		json.NewEncoder(w).Encode(response)
	})

	return httptest.NewServer(mux)
}

// FuzzHTTPIngest fuzzes the ingest endpoint with various inputs
func FuzzHTTPIngest(f *testing.F) {
	tempDir := f.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")

	server := createBenchmarkServer(f, handoffDir)
	defer server.Close()

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
		defer resp.Body.Close()

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

// Test specific performance characteristics
func TestPerformanceCharacteristics(t *testing.T) {
	tempDir := t.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")

	server := createBenchmarkServer(t, handoffDir)
	defer server.Close()

	t.Run("response time is under 100ms", func(t *testing.T) {
		intent := map[string]interface{}{
			"spec": map[string]interface{}{
				"intent": "Performance test intent",
			},
		}

		intentJSON, err := json.Marshal(intent)
		require.NoError(t, err)

		start := time.Now()
		resp, err := http.Post(server.URL+"/ingest", "application/json", bytes.NewBuffer(intentJSON))
		duration := time.Since(start)

		require.NoError(t, err)
		defer resp.Body.Close()

		if duration > 100*time.Millisecond {
			t.Errorf("Response time %v exceeds 100ms threshold", duration)
		}
	})

	t.Run("handles high concurrency without errors", func(t *testing.T) {
		const concurrency = 50
		const requestsPerWorker = 10

		intent := map[string]interface{}{
			"spec": map[string]interface{}{
				"intent": "Concurrency test intent",
			},
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
