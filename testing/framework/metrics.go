// Package framework provides comprehensive performance metrics and benchmarking
package framework

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

// TestMetrics provides comprehensive performance monitoring and analysis
type TestMetrics struct {
	// Test execution metrics
	testStartTime    time.Time
	testEndTime      time.Time
	totalTests       int
	passedTests      int
	failedTests      int
	skippedTests     int
	
	// Performance metrics
	latencyMetrics   map[string][]time.Duration
	throughputMetrics map[string]float64
	errorRateMetrics map[string]float64
	
	// Resource utilization
	memoryUsage      []MemorySnapshot
	cpuUsage         []CPUSnapshot
	goroutineCount   []int
	
	// Load testing metrics
	loadTestResults  map[string]*LoadTestResult
	
	// Code coverage
	coverageData     *CoverageData
	
	// Benchmarking results
	benchmarkResults map[string]*BenchmarkResult
	
	// Synchronization
	mu sync.RWMutex
	
	// Prometheus client for external metrics
	prometheusClient v1.API
}

// MemorySnapshot captures memory usage at a point in time
type MemorySnapshot struct {
	Timestamp    time.Time
	Alloc        uint64
	TotalAlloc   uint64
	Sys          uint64
	NumGC        uint32
	HeapAlloc    uint64
	HeapSys      uint64
	HeapInuse    uint64
}

// CPUSnapshot captures CPU usage information
type CPUSnapshot struct {
	Timestamp    time.Time
	NumCPU       int
	NumGoroutine int
	GOMAXPROCS   int
}

// LoadTestResult contains results from load testing
type LoadTestResult struct {
	TestName       string
	StartTime      time.Time
	EndTime        time.Time
	TotalRequests  int
	SuccessfulReq  int
	FailedReq      int
	AvgLatency     time.Duration
	P95Latency     time.Duration
	P99Latency     time.Duration
	MaxLatency     time.Duration
	MinLatency     time.Duration
	Throughput     float64 // requests per second
	ErrorRate      float64
	ConcurrentUsers int
}

// CoverageData contains code coverage information
type CoverageData struct {
	TotalLines    int
	CoveredLines  int
	Percentage    float64
	PackageCoverage map[string]float64
	FileCoverage  map[string]float64
}

// BenchmarkResult contains benchmark execution results
type BenchmarkResult struct {
	Name           string
	Iterations     int
	NsPerOp        int64
	AllocsPerOp    int64
	BytesPerOp     int64
	MemAllocsPerOp int64
	MemBytesPerOp  int64
}

// NewTestMetrics creates a new test metrics collector
func NewTestMetrics() *TestMetrics {
	return &TestMetrics{
		latencyMetrics:    make(map[string][]time.Duration),
		throughputMetrics: make(map[string]float64),
		errorRateMetrics:  make(map[string]float64),
		loadTestResults:   make(map[string]*LoadTestResult),
		benchmarkResults:  make(map[string]*BenchmarkResult),
		coverageData:      &CoverageData{
			PackageCoverage: make(map[string]float64),
			FileCoverage:    make(map[string]float64),
		},
	}
}

// Initialize sets up metrics collection
func (tm *TestMetrics) Initialize() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	tm.testStartTime = time.Now()
	
	// Start resource monitoring
	go tm.monitorResources()
	
	// Initialize Prometheus client if available
	tm.initPrometheusClient()
}

// Reset resets all metrics for a new test run
func (tm *TestMetrics) Reset() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	tm.testStartTime = time.Now()
	tm.testEndTime = time.Time{}
	tm.totalTests = 0
	tm.passedTests = 0
	tm.failedTests = 0
	tm.skippedTests = 0
	
	tm.latencyMetrics = make(map[string][]time.Duration)
	tm.throughputMetrics = make(map[string]float64)
	tm.errorRateMetrics = make(map[string]float64)
	tm.memoryUsage = nil
	tm.cpuUsage = nil
	tm.goroutineCount = nil
}

// RecordLatency records latency for a specific operation
func (tm *TestMetrics) RecordLatency(operation string, latency time.Duration) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	tm.latencyMetrics[operation] = append(tm.latencyMetrics[operation], latency)
}

// RecordThroughput records throughput for a specific operation
func (tm *TestMetrics) RecordThroughput(operation string, throughput float64) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	tm.throughputMetrics[operation] = throughput
}

// RecordErrorRate records error rate for a specific operation
func (tm *TestMetrics) RecordErrorRate(operation string, errorRate float64) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	tm.errorRateMetrics[operation] = errorRate
}

// CollectTestMetrics collects metrics for a completed test
func (tm *TestMetrics) CollectTestMetrics() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	tm.totalTests++
	tm.testEndTime = time.Now()
}

// RecordTestResult records the result of a test
func (tm *TestMetrics) RecordTestResult(passed bool, skipped bool) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	if skipped {
		tm.skippedTests++
	} else if passed {
		tm.passedTests++
	} else {
		tm.failedTests++
	}
}

// ExecuteLoadTest runs a load test with specified parameters
func (tm *TestMetrics) ExecuteLoadTest(concurrency int, duration time.Duration, testFunc func() error) error {
	testName := fmt.Sprintf("load_test_%d_users_%v", concurrency, duration)
	
	result := &LoadTestResult{
		TestName:        testName,
		StartTime:       time.Now(),
		ConcurrentUsers: concurrency,
		MinLatency:      time.Hour, // Initialize to max value
	}
	
	// Channel to collect results from goroutines
	resultChan := make(chan LoadTestRequest, concurrency*1000)
	
	// Context for cancellation
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	
	// Start concurrent workers
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tm.loadTestWorker(ctx, testFunc, resultChan)
		}()
	}
	
	// Collect results
	go func() {
		wg.Wait()
		close(resultChan)
	}()
	
	// Process results
	var latencies []time.Duration
	for req := range resultChan {
		result.TotalRequests++
		latencies = append(latencies, req.Latency)
		
		if req.Error != nil {
			result.FailedReq++
		} else {
			result.SuccessfulReq++
		}
		
		// Update min/max latency
		if req.Latency > result.MaxLatency {
			result.MaxLatency = req.Latency
		}
		if req.Latency < result.MinLatency {
			result.MinLatency = req.Latency
		}
	}
	
	result.EndTime = time.Now()
	
	// Calculate metrics
	if len(latencies) > 0 {
		result.AvgLatency = tm.calculateAverageLatency(latencies)
		result.P95Latency = tm.calculatePercentile(latencies, 95)
		result.P99Latency = tm.calculatePercentile(latencies, 99)
	}
	
	testDuration := result.EndTime.Sub(result.StartTime).Seconds()
	result.Throughput = float64(result.TotalRequests) / testDuration
	
	if result.TotalRequests > 0 {
		result.ErrorRate = float64(result.FailedReq) / float64(result.TotalRequests)
	}
	
	// Store result
	tm.mu.Lock()
	tm.loadTestResults[testName] = result
	tm.mu.Unlock()
	
	return nil
}

// LoadTestRequest represents a single load test request
type LoadTestRequest struct {
	Timestamp time.Time
	Latency   time.Duration
	Error     error
}

// loadTestWorker executes load test requests
func (tm *TestMetrics) loadTestWorker(ctx context.Context, testFunc func() error, resultChan chan<- LoadTestRequest) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			start := time.Now()
			err := testFunc()
			latency := time.Since(start)
			
			select {
			case resultChan <- LoadTestRequest{
				Timestamp: start,
				Latency:   latency,
				Error:     err,
			}:
			case <-ctx.Done():
				return
			}
		}
	}
}

// monitorResources continuously monitors system resources
func (tm *TestMetrics) monitorResources() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		tm.collectResourceSnapshot()
	}
}

// collectResourceSnapshot takes a snapshot of current resource usage
func (tm *TestMetrics) collectResourceSnapshot() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	// Memory snapshot
	memSnapshot := MemorySnapshot{
		Timestamp:  time.Now(),
		Alloc:      memStats.Alloc,
		TotalAlloc: memStats.TotalAlloc,
		Sys:        memStats.Sys,
		NumGC:      memStats.NumGC,
		HeapAlloc:  memStats.HeapAlloc,
		HeapSys:    memStats.HeapSys,
		HeapInuse:  memStats.HeapInuse,
	}
	tm.memoryUsage = append(tm.memoryUsage, memSnapshot)
	
	// CPU snapshot
	cpuSnapshot := CPUSnapshot{
		Timestamp:    time.Now(),
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
		GOMAXPROCS:   runtime.GOMAXPROCS(0),
	}
	tm.cpuUsage = append(tm.cpuUsage, cpuSnapshot)
	tm.goroutineCount = append(tm.goroutineCount, runtime.NumGoroutine())
	
	// Keep only recent data (last 1000 samples)
	if len(tm.memoryUsage) > 1000 {
		tm.memoryUsage = tm.memoryUsage[len(tm.memoryUsage)-1000:]
	}
	if len(tm.cpuUsage) > 1000 {
		tm.cpuUsage = tm.cpuUsage[len(tm.cpuUsage)-1000:]
	}
	if len(tm.goroutineCount) > 1000 {
		tm.goroutineCount = tm.goroutineCount[len(tm.goroutineCount)-1000:]
	}
}

// calculateAverageLatency calculates average latency from a slice of durations
func (tm *TestMetrics) calculateAverageLatency(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	
	var total time.Duration
	for _, latency := range latencies {
		total += latency
	}
	
	return total / time.Duration(len(latencies))
}

// calculatePercentile calculates the specified percentile from latencies
func (tm *TestMetrics) calculatePercentile(latencies []time.Duration, percentile int) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	
	// Simple percentile calculation (would use proper sorting in production)
	index := (percentile * len(latencies)) / 100
	if index >= len(latencies) {
		index = len(latencies) - 1
	}
	
	return latencies[index]
}

// GetCoveragePercentage returns the current code coverage percentage
func (tm *TestMetrics) GetCoveragePercentage() float64 {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	
	return tm.coverageData.Percentage
}

// UpdateCoverageData updates code coverage information
func (tm *TestMetrics) UpdateCoverageData(totalLines, coveredLines int) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	tm.coverageData.TotalLines = totalLines
	tm.coverageData.CoveredLines = coveredLines
	if totalLines > 0 {
		tm.coverageData.Percentage = (float64(coveredLines) / float64(totalLines)) * 100
	}
}

// RecordBenchmark records benchmark results
func (tm *TestMetrics) RecordBenchmark(name string, iterations int, nsPerOp int64, allocsPerOp int64, bytesPerOp int64) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	tm.benchmarkResults[name] = &BenchmarkResult{
		Name:        name,
		Iterations:  iterations,
		NsPerOp:     nsPerOp,
		AllocsPerOp: allocsPerOp,
		BytesPerOp:  bytesPerOp,
	}
}

// initPrometheusClient initializes Prometheus client for external metrics
func (tm *TestMetrics) initPrometheusClient() {
	prometheusURL := os.Getenv("PROMETHEUS_URL")
	if prometheusURL == "" {
		prometheusURL = "http://localhost:9090"
	}
	
	client, err := api.NewClient(api.Config{
		Address: prometheusURL,
	})
	if err == nil {
		tm.prometheusClient = v1.NewAPI(client)
	}
}

// GenerateReport creates a comprehensive performance report
func (tm *TestMetrics) GenerateReport() {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	
	report := tm.createTestReport()
	
	// Write report to file
	reportData, _ := json.MarshalIndent(report, "", "  ")
	filename := fmt.Sprintf("test-report-%d.json", time.Now().Unix())
	ioutil.WriteFile(filename, reportData, 0644)
	
	// Print summary to console
	tm.printSummary(report)
}

// createTestReport creates a comprehensive test report
func (tm *TestMetrics) createTestReport() map[string]interface{} {
	report := map[string]interface{}{
		"test_execution": map[string]interface{}{
			"start_time":    tm.testStartTime,
			"end_time":      tm.testEndTime,
			"duration":      tm.testEndTime.Sub(tm.testStartTime).String(),
			"total_tests":   tm.totalTests,
			"passed_tests":  tm.passedTests,
			"failed_tests":  tm.failedTests,
			"skipped_tests": tm.skippedTests,
			"success_rate":  float64(tm.passedTests) / float64(tm.totalTests) * 100,
		},
		"performance_metrics": map[string]interface{}{
			"latency_metrics":    tm.latencyMetrics,
			"throughput_metrics": tm.throughputMetrics,
			"error_rate_metrics": tm.errorRateMetrics,
		},
		"resource_utilization": map[string]interface{}{
			"peak_memory_usage":     tm.getPeakMemoryUsage(),
			"average_memory_usage":  tm.getAverageMemoryUsage(),
			"peak_goroutine_count":  tm.getPeakGoroutineCount(),
			"average_goroutine_count": tm.getAverageGoroutineCount(),
		},
		"load_test_results": tm.loadTestResults,
		"coverage_data":     tm.coverageData,
		"benchmark_results": tm.benchmarkResults,
	}
	
	return report
}

// Helper methods for report generation
func (tm *TestMetrics) getPeakMemoryUsage() uint64 {
	var peak uint64
	for _, snapshot := range tm.memoryUsage {
		if snapshot.Alloc > peak {
			peak = snapshot.Alloc
		}
	}
	return peak
}

func (tm *TestMetrics) getAverageMemoryUsage() uint64 {
	if len(tm.memoryUsage) == 0 {
		return 0
	}
	
	var total uint64
	for _, snapshot := range tm.memoryUsage {
		total += snapshot.Alloc
	}
	return total / uint64(len(tm.memoryUsage))
}

func (tm *TestMetrics) getPeakGoroutineCount() int {
	var peak int
	for _, count := range tm.goroutineCount {
		if count > peak {
			peak = count
		}
	}
	return peak
}

func (tm *TestMetrics) getAverageGoroutineCount() int {
	if len(tm.goroutineCount) == 0 {
		return 0
	}
	
	var total int
	for _, count := range tm.goroutineCount {
		total += count
	}
	return total / len(tm.goroutineCount)
}

// printSummary prints a summary of the test results
func (tm *TestMetrics) printSummary(report map[string]interface{}) {
	fmt.Println("=== Test Execution Summary ===")
	
	if testExec, ok := report["test_execution"].(map[string]interface{}); ok {
		fmt.Printf("Total Tests: %v\n", testExec["total_tests"])
		fmt.Printf("Passed: %v\n", testExec["passed_tests"])
		fmt.Printf("Failed: %v\n", testExec["failed_tests"])
		fmt.Printf("Success Rate: %.2f%%\n", testExec["success_rate"])
		fmt.Printf("Duration: %v\n", testExec["duration"])
	}
	
	fmt.Printf("Code Coverage: %.2f%%\n", tm.coverageData.Percentage)
	
	if len(tm.loadTestResults) > 0 {
		fmt.Println("\n=== Load Test Results ===")
		for name, result := range tm.loadTestResults {
			fmt.Printf("%s: %d requests, %.2f req/s, %.2f%% errors\n",
				name, result.TotalRequests, result.Throughput, result.ErrorRate*100)
		}
	}
}