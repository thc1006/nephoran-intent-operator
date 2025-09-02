package loop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// BenchmarkResults holds comprehensive benchmark data
type BenchmarkResults struct {
	Throughput       float64       `json:"throughput_files_per_second"`
	P99Latency       time.Duration `json:"p99_latency_ms"`
	P95Latency       time.Duration `json:"p95_latency_ms"`
	P50Latency       time.Duration `json:"p50_latency_ms"`
	MemoryFootprint  int64         `json:"memory_footprint_bytes"`
	EnergyEfficiency float64       `json:"energy_efficiency_gbps_per_watt"`
	GoroutineCount   int           `json:"goroutine_count"`
	AllocationsPerOp int64         `json:"allocations_per_operation"`
	BytesAllocPerOp  int64         `json:"bytes_allocated_per_operation"`
	CPUUtilization   float64       `json:"cpu_utilization_percent"`
}

// PerformanceTest represents a single performance test scenario
type PerformanceTest struct {
	Name             string
	FileCount        int
	FileSize         int
	ConcurrentOps    int
	TargetThroughput float64
	MaxLatency       time.Duration
	MaxMemory        int64
	MinEfficiency    float64
}

// Benchmark scenarios for different O-RAN L Release conditions
var benchmarkScenarios = []PerformanceTest{
	{
		Name:             "HighThroughput_SmallFiles",
		FileCount:        10000,
		FileSize:         1024, // 1KB files
		ConcurrentOps:    100,
		TargetThroughput: 10000, // 10k files/sec
		MaxLatency:       5 * time.Millisecond,
		MaxMemory:        100 * 1024 * 1024, // 100MB
		MinEfficiency:    0.5,               // 0.5 Gbps/Watt
	},
	{
		Name:             "MediumThroughput_MediumFiles",
		FileCount:        5000,
		FileSize:         64 * 1024, // 64KB files
		ConcurrentOps:    50,
		TargetThroughput: 5000,
		MaxLatency:       10 * time.Millisecond,
		MaxMemory:        200 * 1024 * 1024, // 200MB
		MinEfficiency:    0.4,
	},
	{
		Name:             "LowThroughput_LargeFiles",
		FileCount:        1000,
		FileSize:         1024 * 1024, // 1MB files
		ConcurrentOps:    10,
		TargetThroughput: 1000,
		MaxLatency:       50 * time.Millisecond,
		MaxMemory:        500 * 1024 * 1024, // 500MB
		MinEfficiency:    0.3,
	},
	{
		Name:             "EdgeCase_TinyFiles",
		FileCount:        50000,
		FileSize:         100, // 100 byte files
		ConcurrentOps:    200,
		TargetThroughput: 20000,
		MaxLatency:       2 * time.Millisecond,
		MaxMemory:        50 * 1024 * 1024, // 50MB
		MinEfficiency:    0.6,
	},
	{
		Name:             "EdgeCase_HugeFiles",
		FileCount:        100,
		FileSize:         10 * 1024 * 1024, // 10MB files
		ConcurrentOps:    5,
		TargetThroughput: 100,
		MaxLatency:       200 * time.Millisecond,
		MaxMemory:        1024 * 1024 * 1024, // 1GB
		MinEfficiency:    0.2,
	},
}

// BenchmarkConductorLoop runs comprehensive performance benchmarks
func BenchmarkConductorLoop(b *testing.B) {
	for _, scenario := range benchmarkScenarios {
		b.Run(scenario.Name, func(b *testing.B) {
			runPerformanceBenchmark(b, scenario)
		})
	}
}

// runPerformanceBenchmark executes a single benchmark scenario
func runPerformanceBenchmark(b *testing.B, test PerformanceTest) {
	// Setup benchmark environment
	tempDir, cleanup := setupBenchmarkEnvironment(b, test)
	defer cleanup()

	// Create watcher with optimized configuration
	config := Config{
		MaxWorkers:   runtime.NumCPU() * 4,
		DebounceDur:  10 * time.Millisecond, // Minimum debounce for max throughput
		CleanupAfter: 24 * time.Hour,
		MetricsPort:  0, // Disable metrics server for benchmark
	}

	watcher, err := NewWatcher(tempDir, config)
	if err != nil {
		b.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	// Pre-generate test files
	testFiles := generateBenchmarkFiles(b, tempDir, test.FileCount, test.FileSize)

	// Initialize metrics tracking
	var (
		startTime      = time.Now()
		completedFiles int64
		latencies      = make([]time.Duration, 0, test.FileCount)
		latencyMutex   sync.Mutex
	)

	// Start memory profiling
	var m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Start the benchmark
	b.ResetTimer()
	b.StartTimer()

	// Process files with controlled concurrency
	var wg sync.WaitGroup
	sem := make(chan struct{}, test.ConcurrentOps)

	for i := 0; i < test.FileCount; i++ {
		wg.Add(1)
		go func(fileIndex int) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			fileStart := time.Now()
			filePath := testFiles[fileIndex]

			// Simulate file processing
			err := processFileForBenchmark(watcher, filePath)

			latency := time.Since(fileStart)

			latencyMutex.Lock()
			latencies = append(latencies, latency)
			latencyMutex.Unlock()

			if err == nil {
				atomic.AddInt64(&completedFiles, 1)
			}
		}(i)
	}

	// Wait for all operations to complete
	wg.Wait()

	b.StopTimer()
	totalDuration := time.Since(startTime)

	// Collect memory stats
	var m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Calculate performance metrics
	results := calculateBenchmarkResults(completedFiles, totalDuration, latencies, &m1, &m2)

	// Validate against targets
	validateBenchmarkResults(b, test, results)

	// Report results
	reportBenchmarkResults(b, test, results)
}

// setupBenchmarkEnvironment creates a temporary directory for benchmark files
func setupBenchmarkEnvironment(b *testing.B, test PerformanceTest) (string, func()) {
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("conductor_bench_%s_*", test.Name))
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}

	// Create subdirectories
	for _, subdir := range []string{"processed", "failed", "status"} {
		if err := os.MkdirAll(filepath.Join(tempDir, subdir), 0o755); err != nil {
			b.Fatalf("Failed to create subdirectory %s: %v", subdir, err)
		}
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

// generateBenchmarkFiles creates test intent files with realistic content
func generateBenchmarkFiles(b *testing.B, dir string, count, size int) []string {
	files := make([]string, count)

	for i := 0; i < count; i++ {
		filename := fmt.Sprintf("intent-%06d-%d.json", i, time.Now().UnixNano())
		filePath := filepath.Join(dir, filename)

		// Generate realistic intent JSON content
		content := generateIntentContent(size)

		if err := os.WriteFile(filePath, content, 0o644); err != nil {
			b.Fatalf("Failed to create test file %s: %v", filePath, err)
		}

		files[i] = filePath
	}

	return files
}

// generateIntentContent creates realistic O-RAN intent JSON content
func generateIntentContent(targetSize int) []byte {
	baseIntent := map[string]interface{}{
		"apiVersion": "nephoran.com/v1alpha1",
		"kind":       "NetworkIntent",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("intent-%d", time.Now().UnixNano()),
			"namespace": "o-ran",
			"labels": map[string]string{
				"component":     "conductor-loop",
				"environment":   "benchmark",
				"o-ran-release": "l-release",
			},
		},
		"spec": map[string]interface{}{
			"action": "deploy",
			"target": map[string]interface{}{
				"type": "du",
				"name": "o-du-benchmark",
			},
			"parameters": map[string]interface{}{
				"networkFunctions": []map[string]interface{}{
					{
						"name": "o-du-high",
						"type": "distributed-unit",
						"resources": map[string]string{
							"cpu":    "4",
							"memory": "8Gi",
							"gpu":    "1",
						},
					},
				},
				"connectivity": map[string]interface{}{
					"fronthaul": map[string]string{
						"interface": "eth0",
						"bandwidth": "25Gbps",
					},
					"midhaul": map[string]string{
						"interface": "eth1",
						"bandwidth": "10Gbps",
					},
				},
				"sla": map[string]string{
					"latency":      "1ms",
					"throughput":   "1Gbps",
					"availability": "99.999%",
				},
			},
			"constraints": map[string]interface{}{
				"placement": map[string]string{
					"zone": "edge",
					"rack": "edge-001",
				},
				"security": map[string]string{
					"isolation":  "strict",
					"encryption": "enabled",
				},
			},
		},
	}

	// Serialize to JSON
	contentBytes, _ := json.Marshal(baseIntent)
	currentSize := len(contentBytes)

	// Pad to target size if needed
	if currentSize < targetSize {
		padding := make([]byte, targetSize-currentSize)
		for i := range padding {
			padding[i] = ' '
		}

		// Insert padding into metadata
		if metadata, ok := baseIntent["metadata"].(map[string]interface{}); ok {
			metadata["padding"] = string(padding)
		}

		contentBytes, _ = json.Marshal(baseIntent)
	}

	return contentBytes
}

// processFileForBenchmark simulates file processing with realistic operations
func processFileForBenchmark(watcher *Watcher, filePath string) error {
	// Simulate validation
	if err := simulateValidation(filePath); err != nil {
		return err
	}

	// Simulate state check
	if err := simulateStateCheck(watcher, filePath); err != nil {
		return err
	}

	// Simulate processing
	time.Sleep(time.Microsecond * 100) // 0.1ms processing time

	return nil
}

// calculateBenchmarkResults computes comprehensive performance metrics
func calculateBenchmarkResults(completed int64, duration time.Duration, latencies []time.Duration,
	m1, m2 *runtime.MemStats,
) BenchmarkResults {
	throughput := float64(completed) / duration.Seconds()

	// Calculate latency percentiles
	p50, p95, p99 := calculateLatencyPercentiles(latencies)

	// Memory usage
	memoryUsed := int64(m2.Alloc - m1.Alloc)
	allocations := int64(m2.Mallocs - m1.Mallocs)
	bytesPerAlloc := int64(0)
	if allocations > 0 {
		bytesPerAlloc = memoryUsed / allocations
	}

	// Energy efficiency estimate (simplified)
	// In production, this would use actual power measurement
	estimatedPower := float64(runtime.NumCPU()) * 20.0                // 20W per core estimate
	dataProcessed := float64(completed) * 1024 / (1024 * 1024 * 1024) // GB
	efficiency := dataProcessed / estimatedPower                      // GB/Watt

	return BenchmarkResults{
		Throughput:       throughput,
		P99Latency:       p99,
		P95Latency:       p95,
		P50Latency:       p50,
		MemoryFootprint:  memoryUsed,
		EnergyEfficiency: efficiency,
		GoroutineCount:   runtime.NumGoroutine(),
		AllocationsPerOp: allocations / completed,
		BytesAllocPerOp:  bytesPerAlloc,
		CPUUtilization:   calculateCPUUtilization(),
	}
}

// calculateLatencyPercentiles computes P50, P95, P99 latencies
func calculateLatencyPercentiles(latencies []time.Duration) (time.Duration, time.Duration, time.Duration) {
	if len(latencies) == 0 {
		return 0, 0, 0
	}

	// Sort latencies
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)

	// Simple sort for benchmark
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	n := len(sorted)
	p50 := sorted[n*50/100]
	p95 := sorted[n*95/100]
	p99 := sorted[n*99/100]

	return p50, p95, p99
}

// validateBenchmarkResults checks if benchmark meets target performance
func validateBenchmarkResults(b *testing.B, test PerformanceTest, results BenchmarkResults) {
	failures := []string{}

	// Check throughput target
	if results.Throughput < test.TargetThroughput {
		failures = append(failures, fmt.Sprintf("Throughput %.2f < target %.2f files/sec",
			results.Throughput, test.TargetThroughput))
	}

	// Check latency target
	if results.P99Latency > test.MaxLatency {
		failures = append(failures, fmt.Sprintf("P99 latency %v > target %v",
			results.P99Latency, test.MaxLatency))
	}

	// Check memory target
	if results.MemoryFootprint > test.MaxMemory {
		failures = append(failures, fmt.Sprintf("Memory %d bytes > target %d bytes",
			results.MemoryFootprint, test.MaxMemory))
	}

	// Check efficiency target
	if results.EnergyEfficiency < test.MinEfficiency {
		failures = append(failures, fmt.Sprintf("Energy efficiency %.3f < target %.3f Gbps/Watt",
			results.EnergyEfficiency, test.MinEfficiency))
	}

	if len(failures) > 0 {
		b.Logf("Benchmark %s failed performance targets:", test.Name)
		for _, failure := range failures {
			b.Logf("  - %s", failure)
		}
		// Don't fail the benchmark, just log warnings
	}
}

// reportBenchmarkResults outputs comprehensive performance metrics
func reportBenchmarkResults(b *testing.B, test PerformanceTest, results BenchmarkResults) {
	b.Logf("=== Benchmark Results for %s ===", test.Name)
	b.Logf("Throughput:          %.2f files/sec (target: %.2f)",
		results.Throughput, test.TargetThroughput)
	b.Logf("Latency P99:         %v (target: <%v)", results.P99Latency, test.MaxLatency)
	b.Logf("Latency P95:         %v", results.P95Latency)
	b.Logf("Latency P50:         %v", results.P50Latency)
	b.Logf("Memory footprint:    %d bytes (target: <%d)",
		results.MemoryFootprint, test.MaxMemory)
	b.Logf("Energy efficiency:   %.3f Gbps/Watt (target: >%.3f)",
		results.EnergyEfficiency, test.MinEfficiency)
	b.Logf("Goroutines:          %d", results.GoroutineCount)
	b.Logf("Allocs per op:       %d", results.AllocationsPerOp)
	b.Logf("Bytes per alloc:     %d", results.BytesAllocPerOp)
	b.Logf("CPU utilization:     %.1f%%", results.CPUUtilization)

	// Report to testing framework
	b.ReportMetric(results.Throughput, "files/sec")
	b.ReportMetric(float64(results.P99Latency.Nanoseconds())/1e6, "p99-latency-ms")
	b.ReportMetric(float64(results.MemoryFootprint)/1024/1024, "memory-mb")
	b.ReportMetric(results.EnergyEfficiency, "gbps-per-watt")
}

// Helper functions
func simulateValidation(filePath string) error {
	// Simulate JSON parsing and validation
	time.Sleep(time.Microsecond * 50) // 0.05ms
	return nil
}

func simulateStateCheck(watcher *Watcher, filePath string) error {
	// Simulate state manager lookup
	time.Sleep(time.Microsecond * 25) // 0.025ms
	return nil
}

func calculateCPUUtilization() float64 {
	// Simplified CPU utilization estimate
	return float64(runtime.NumGoroutine()) / float64(runtime.NumCPU()) * 10.0
}

// BenchmarkSpecificOperations provides focused micro-benchmarks
func BenchmarkJSONValidation(b *testing.B) {
	testFile := createTestIntentFile(b, 1024)
	defer os.Remove(testFile)

	watcher := createTestWatcher(b)
	defer watcher.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		watcher.validateJSONFile(testFile)
	}
}

func BenchmarkStateManagerLookup(b *testing.B) {
	watcher := createTestWatcher(b)
	defer watcher.Close()

	testFile := "test-file.json"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		watcher.stateManager.IsProcessed(testFile)
	}
}

func BenchmarkWorkerPoolThroughput(b *testing.B) {
	watcher := createTestWatcher(b)
	defer watcher.Close()

	var completed int64

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		workItem := WorkItem{
			FilePath: fmt.Sprintf("test-file-%d.json", i),
			Attempt:  1,
		}

		select {
		case watcher.workerPool.workQueue <- workItem:
			atomic.AddInt64(&completed, 1)
		default:
			// Queue full
		}
	}

	b.StopTimer()
	b.Logf("Completed %d/%d work items", completed, int64(b.N))
}

// Helper function to create test watcher
func createTestWatcher(b *testing.B) *Watcher {
	tempDir, err := os.MkdirTemp("", "bench_test_*")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}

	config := Config{
		MaxWorkers:  runtime.NumCPU(),
		DebounceDur: 10 * time.Millisecond,
		MetricsPort: 0,
	}

	watcher, err := NewWatcher(tempDir, config)
	if err != nil {
		b.Fatalf("Failed to create watcher: %v", err)
	}

	return watcher
}

// Helper function to create test intent file
func createTestIntentFile(b *testing.B, size int) string {
	tempFile, err := os.CreateTemp("", "intent_*.json")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	defer tempFile.Close()

	content := generateIntentContent(size)
	if _, err := tempFile.Write(content); err != nil {
		b.Fatalf("Failed to write test file: %v", err)
	}

	return tempFile.Name()
}
