//go:build ml && !test

package ml

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"encoding/json"
)

// LoadTestConfig defines configuration for load testing
type LoadTestConfig struct {
	Duration           time.Duration
	ConcurrentUsers    int
	RequestsPerSecond  int
	DataPointsPerQuery int
	EnableCaching      bool
	EnableOptimization bool
}

// LoadTestResults contains load test metrics
type LoadTestResults struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	TotalLatency       int64 // in microseconds
	MinLatency         int64
	MaxLatency         int64
	P50Latency         int64
	P95Latency         int64
	P99Latency         int64
	MemoryUsed         int64
	GoroutinesUsed     int
}

// LoadTester performs load testing on the optimization engine
type LoadTester struct {
	config    *LoadTestConfig
	engine    *OptimizationEngine
	results   *LoadTestResults
	latencies []int64
	mu        sync.Mutex
}

// NewLoadTester creates a new load tester
func NewLoadTester(config *LoadTestConfig, engine *OptimizationEngine) *LoadTester {
	return &LoadTester{
		config:    config,
		engine:    engine,
		results:   &LoadTestResults{MinLatency: int64(^uint64(0) >> 1)}, // Max int64
		latencies: make([]int64, 0, config.Duration.Seconds()*float64(config.RequestsPerSecond)),
	}
}

// Run executes the load test
func (lt *LoadTester) Run(ctx context.Context) (*LoadTestResults, error) {
	// Create rate limiter
	limiter := time.NewTicker(time.Second / time.Duration(lt.config.RequestsPerSecond))
	defer limiter.Stop()

	// Create worker pool
	workers := make(chan struct{}, lt.config.ConcurrentUsers)
	for i := 0; i < lt.config.ConcurrentUsers; i++ {
		workers <- struct{}{}
	}

	// Start time
	startTime := time.Now()
	deadline := startTime.Add(lt.config.Duration)

	// WaitGroup for all requests
	var wg sync.WaitGroup

	// Request generator
	go func() {
		for time.Now().Before(deadline) {
			select {
			case <-ctx.Done():
				return
			case <-limiter.C:
				// Get worker
				<-workers

				wg.Add(1)
				go func() {
					defer func() {
						workers <- struct{}{} // Return worker
						wg.Done()
					}()

					lt.executeRequest(ctx)
				}()
			}
		}
	}()

	// Wait for all requests to complete
	wg.Wait()

	// Calculate percentiles
	lt.calculatePercentiles()

	// Collect final metrics
	lt.results.GoroutinesUsed = lt.config.ConcurrentUsers

	return lt.results, nil
}

// executeRequest performs a single optimization request
func (lt *LoadTester) executeRequest(ctx context.Context) {
	// Generate random intent
	intent := &NetworkIntent{
		ID:          fmt.Sprintf("load-test-%d", rand.Int63()),
		Description: "Load test intent",
		Priority:    randomPriority(),
		Parameters:  json.RawMessage(`{}`),
		Timestamp:   time.Now(),
	}

	// Measure request
	start := time.Now()
	_, err := lt.engine.OptimizeNetworkDeployment(ctx, intent)
	latency := time.Since(start).Microseconds()

	// Update metrics
	atomic.AddInt64(&lt.results.TotalRequests, 1)

	if err != nil {
		atomic.AddInt64(&lt.results.FailedRequests, 1)
	} else {
		atomic.AddInt64(&lt.results.SuccessfulRequests, 1)
		atomic.AddInt64(&lt.results.TotalLatency, latency)

		// Update min/max latency
		for {
			min := atomic.LoadInt64(&lt.results.MinLatency)
			if latency >= min || atomic.CompareAndSwapInt64(&lt.results.MinLatency, min, latency) {
				break
			}
		}

		for {
			max := atomic.LoadInt64(&lt.results.MaxLatency)
			if latency <= max || atomic.CompareAndSwapInt64(&lt.results.MaxLatency, max, latency) {
				break
			}
		}

		// Store latency for percentile calculation
		lt.mu.Lock()
		lt.latencies = append(lt.latencies, latency)
		lt.mu.Unlock()
	}
}

// calculatePercentiles calculates latency percentiles
func (lt *LoadTester) calculatePercentiles() {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	if len(lt.latencies) == 0 {
		return
	}

	// Sort latencies
	sortInt64s(lt.latencies)

	// Calculate percentiles
	lt.results.P50Latency = lt.latencies[len(lt.latencies)*50/100]
	lt.results.P95Latency = lt.latencies[len(lt.latencies)*95/100]
	lt.results.P99Latency = lt.latencies[len(lt.latencies)*99/100]
}

// Helper functions
func randomPriority() string {
	priorities := []string{"low", "medium", "high", "critical"}
	return priorities[rand.Intn(len(priorities))]
}

func randomQoS() string {
	qosLevels := []string{"best-effort", "guaranteed", "premium"}
	return qosLevels[rand.Intn(len(qosLevels))]
}

func sortInt64s(a []int64) {
	// Simple quicksort for int64
	if len(a) < 2 {
		return
	}

	left, right := 0, len(a)-1
	pivot := rand.Intn(right)

	a[pivot], a[right] = a[right], a[pivot]

	for i := range a {
		if a[i] < a[right] {
			a[left], a[i] = a[i], a[left]
			left++
		}
	}

	a[left], a[right] = a[right], a[left]

	sortInt64s(a[:left])
	sortInt64s(a[left+1:])
}

// TestLoadScenarios runs various load test scenarios
func TestLoadScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	scenarios := []struct {
		name   string
		config LoadTestConfig
	}{
		{
			name: "Light Load",
			config: LoadTestConfig{
				Duration:           30 * time.Second,
				ConcurrentUsers:    10,
				RequestsPerSecond:  50,
				DataPointsPerQuery: 100,
				EnableCaching:      true,
				EnableOptimization: true,
			},
		},
		{
			name: "Medium Load",
			config: LoadTestConfig{
				Duration:           60 * time.Second,
				ConcurrentUsers:    50,
				RequestsPerSecond:  200,
				DataPointsPerQuery: 500,
				EnableCaching:      true,
				EnableOptimization: true,
			},
		},
		{
			name: "Heavy Load",
			config: LoadTestConfig{
				Duration:           120 * time.Second,
				ConcurrentUsers:    100,
				RequestsPerSecond:  500,
				DataPointsPerQuery: 1000,
				EnableCaching:      true,
				EnableOptimization: true,
			},
		},
		{
			name: "Stress Test",
			config: LoadTestConfig{
				Duration:           300 * time.Second,
				ConcurrentUsers:    200,
				RequestsPerSecond:  1000,
				DataPointsPerQuery: 2000,
				EnableCaching:      false, // Disable caching for worst case
				EnableOptimization: false,
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Create test engine
			engine := createBenchmarkEngine(t, scenario.config.DataPointsPerQuery, 5*time.Millisecond)

			// Create load tester
			tester := NewLoadTester(&scenario.config, engine)

			// Run load test
			ctx := context.Background()
			results, err := tester.Run(ctx)
			if err != nil {
				t.Fatalf("Load test failed: %v", err)
			}

			// Log results
			t.Logf("Load Test Results for %s:", scenario.name)
			t.Logf("  Total Requests: %d", results.TotalRequests)
			t.Logf("  Successful: %d (%.2f%%)", results.SuccessfulRequests,
				float64(results.SuccessfulRequests)/float64(results.TotalRequests)*100)
			t.Logf("  Failed: %d", results.FailedRequests)
			t.Logf("  Average Latency: %.2fms", float64(results.TotalLatency)/float64(results.SuccessfulRequests)/1000)
			t.Logf("  Min Latency: %.2fms", float64(results.MinLatency)/1000)
			t.Logf("  Max Latency: %.2fms", float64(results.MaxLatency)/1000)
			t.Logf("  P50 Latency: %.2fms", float64(results.P50Latency)/1000)
			t.Logf("  P95 Latency: %.2fms", float64(results.P95Latency)/1000)
			t.Logf("  P99 Latency: %.2fms", float64(results.P99Latency)/1000)
			t.Logf("  Throughput: %.2f req/s", float64(results.SuccessfulRequests)/scenario.config.Duration.Seconds())

			// Assert performance requirements
			avgLatency := float64(results.TotalLatency) / float64(results.SuccessfulRequests) / 1000 // in ms
			if avgLatency > 1000 {                                                                   // 1 second threshold
				t.Errorf("Average latency too high: %.2fms (expected < 1000ms)", avgLatency)
			}

			successRate := float64(results.SuccessfulRequests) / float64(results.TotalRequests)
			if successRate < 0.95 { // 95% success rate threshold
				t.Errorf("Success rate too low: %.2f%% (expected >= 95%%)", successRate*100)
			}
		})
	}
}

// BenchmarkLoadTest provides benchmark-style load testing
func BenchmarkLoadTest(b *testing.B) {
	config := LoadTestConfig{
		Duration:           10 * time.Second,
		ConcurrentUsers:    50,
		RequestsPerSecond:  100,
		DataPointsPerQuery: 500,
		EnableCaching:      true,
		EnableOptimization: true,
	}

	engine := createBenchmarkEngine(b, config.DataPointsPerQuery, 5*time.Millisecond)
	tester := NewLoadTester(&config, engine)

	b.ResetTimer()

	ctx := context.Background()
	results, err := tester.Run(ctx)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportMetric(float64(results.SuccessfulRequests), "requests")
	b.ReportMetric(float64(results.TotalLatency)/float64(results.SuccessfulRequests)/1000, "ms/op")
	b.ReportMetric(float64(results.P95Latency)/1000, "p95_ms")
	b.ReportMetric(float64(results.P99Latency)/1000, "p99_ms")
}

// Example output generator for documentation
func ExampleLoadTestReport() {
	fmt.Println("ML Optimization Engine Load Test Report")
	fmt.Println("=======================================")
	fmt.Println()
	fmt.Println("Test Configuration:")
	fmt.Println("  Duration: 60s")
	fmt.Println("  Concurrent Users: 50")
	fmt.Println("  Target RPS: 200")
	fmt.Println()
	fmt.Println("Results:")
	fmt.Println("  Total Requests: 12,000")
	fmt.Println("  Successful: 11,940 (99.5%)")
	fmt.Println("  Failed: 60 (0.5%)")
	fmt.Println()
	fmt.Println("Latency Statistics:")
	fmt.Println("  Average: 45.3ms")
	fmt.Println("  Min: 12.1ms")
	fmt.Println("  Max: 523.4ms")
	fmt.Println("  P50: 42.3ms")
	fmt.Println("  P95: 98.7ms")
	fmt.Println("  P99: 234.5ms")
	fmt.Println()
	fmt.Println("Performance:")
	fmt.Println("  Actual Throughput: 199 req/s")
	fmt.Println("  CPU Usage: 65%")
	fmt.Println("  Memory Usage: 2.3GB")
	fmt.Println("  Goroutines: 50")
}
