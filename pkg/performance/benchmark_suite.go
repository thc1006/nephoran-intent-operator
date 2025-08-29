// Package performance provides comprehensive benchmarking and performance optimization for the Nephoran Intent Operator.

package performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"k8s.io/client-go/util/workqueue"
)

// BenchmarkSuite provides comprehensive performance testing capabilities.

type BenchmarkSuite struct {
	metrics *MetricsCollector

	optimizer *OptimizationEngine

	profiler *Profiler

	baselines map[string]PerformanceBaseline

	results []BenchmarkResult

	mu sync.RWMutex

	rateLimiter workqueue.RateLimiter
}

// PerformanceBaseline represents expected performance characteristics.

type PerformanceBaseline struct {
	Name string

	MaxLatencyP50 time.Duration

	MaxLatencyP95 time.Duration

	MaxLatencyP99 time.Duration

	MinThroughput float64

	MaxMemoryMB float64

	MaxCPUPercent float64

	MaxGoroutines int

	RegressionTolerance float64 // Percentage tolerance for regression

}

// BenchmarkResult captures the results of a benchmark run.

type BenchmarkResult struct {
	Name string

	StartTime time.Time

	EndTime time.Time

	Duration time.Duration

	Iterations int64

	TotalRequests int64

	SuccessCount int64

	ErrorCount int64

	Latencies []time.Duration

	LatencyP50 time.Duration

	LatencyP95 time.Duration

	LatencyP99 time.Duration

	Throughput float64

	AvgMemoryMB float64

	PeakMemoryMB float64

	AvgCPUPercent float64

	PeakCPUPercent float64

	MaxGoroutines int

	Errors []error

	ResourceUsage ResourceMetrics

	Passed bool

	RegressionInfo string
}

// ResourceMetrics tracks resource usage during benchmarks.

type ResourceMetrics struct {
	CPUSamples []float64

	MemorySamples []uint64

	GoroutineSamples []int

	GCStats runtime.MemStats

	NetworkIO NetworkMetrics

	DiskIO DiskMetrics
}

// NetworkMetrics tracks network I/O during benchmarks.

type NetworkMetrics struct {
	BytesSent uint64

	BytesReceived uint64

	RequestCount uint64

	ErrorCount uint64
}

// DiskMetrics tracks disk I/O during benchmarks.

type DiskMetrics struct {
	BytesRead uint64

	BytesWritten uint64

	ReadOps uint64

	WriteOps uint64
}

// NewBenchmarkSuite creates a new benchmark suite with default baselines.

func NewBenchmarkSuite() *BenchmarkSuite {

	return &BenchmarkSuite{

		metrics: NewMetricsCollector(),

		optimizer: NewOptimizationEngine(),

		profiler: NewProfiler(),

		baselines: getDefaultBaselines(),

		results: make([]BenchmarkResult, 0),

		rateLimiter: workqueue.NewItemExponentialFailureRateLimiter(time.Millisecond*100, time.Second*10),
	}

}

// getDefaultBaselines returns the default performance baselines.

func getDefaultBaselines() map[string]PerformanceBaseline {

	return map[string]PerformanceBaseline{

		"single_intent": {

			Name: "Single Intent Processing",

			MaxLatencyP50: 15 * time.Second,

			MaxLatencyP95: 25 * time.Second,

			MaxLatencyP99: 30 * time.Second,

			MinThroughput: 1.0,

			MaxMemoryMB: 500,

			MaxCPUPercent: 80,

			MaxGoroutines: 100,

			RegressionTolerance: 10.0,
		},

		"concurrent_10": {

			Name: "10 Concurrent Users",

			MaxLatencyP50: 3 * time.Second,

			MaxLatencyP95: 5 * time.Second,

			MaxLatencyP99: 8 * time.Second,

			MinThroughput: 8.0,

			MaxMemoryMB: 1000,

			MaxCPUPercent: 150,

			MaxGoroutines: 200,

			RegressionTolerance: 15.0,
		},

		"e2nodeset_scaling": {

			Name: "E2NodeSet Scaling to 100 Nodes",

			MaxLatencyP50: 60 * time.Second,

			MaxLatencyP95: 80 * time.Second,

			MaxLatencyP99: 90 * time.Second,

			MinThroughput: 1.1, // nodes per second

			MaxMemoryMB: 2000,

			MaxCPUPercent: 200,

			MaxGoroutines: 500,

			RegressionTolerance: 20.0,
		},

		"memory_stability": {

			Name: "10-Minute Memory Stability",

			MaxLatencyP50: 5 * time.Second,

			MaxLatencyP95: 10 * time.Second,

			MaxLatencyP99: 15 * time.Second,

			MinThroughput: 5.0,

			MaxMemoryMB: 1500,

			MaxCPUPercent: 100,

			MaxGoroutines: 300,

			RegressionTolerance: 5.0,
		},

		"high_throughput": {

			Name: "High Throughput (50 intents/sec)",

			MaxLatencyP50: 2 * time.Second,

			MaxLatencyP95: 5 * time.Second,

			MaxLatencyP99: 10 * time.Second,

			MinThroughput: 50.0,

			MaxMemoryMB: 3000,

			MaxCPUPercent: 400,

			MaxGoroutines: 1000,

			RegressionTolerance: 10.0,
		},
	}

}

// RunSingleIntentBenchmark tests single intent processing performance.

func (bs *BenchmarkSuite) RunSingleIntentBenchmark(ctx context.Context, intentFunc func() error) (*BenchmarkResult, error) {

	baseline := bs.baselines["single_intent"]

	result := &BenchmarkResult{

		Name: baseline.Name,

		StartTime: time.Now(),
	}

	// Start resource monitoring.

	stopMonitor := bs.startResourceMonitoring(ctx, result)

	defer stopMonitor()

	// Run the benchmark.

	start := time.Now()

	err := intentFunc()

	latency := time.Since(start)

	result.EndTime = time.Now()

	result.Duration = result.EndTime.Sub(result.StartTime)

	result.Iterations = 1

	result.TotalRequests = 1

	result.Latencies = []time.Duration{latency}

	if err != nil {

		result.ErrorCount = 1

		result.Errors = []error{err}

	} else {

		result.SuccessCount = 1

	}

	// Calculate metrics.

	bs.calculateMetrics(result)

	// Check against baseline.

	result.Passed = bs.checkBaseline(result, baseline)

	bs.mu.Lock()

	bs.results = append(bs.results, *result)

	bs.mu.Unlock()

	return result, nil

}

// RunConcurrentBenchmark tests concurrent user scenarios.

func (bs *BenchmarkSuite) RunConcurrentBenchmark(ctx context.Context, users int, duration time.Duration, workload func() error) (*BenchmarkResult, error) {

	baselineKey := fmt.Sprintf("concurrent_%d", users)

	baseline, exists := bs.baselines[baselineKey]

	if !exists {

		baseline = bs.baselines["concurrent_10"] // Use default concurrent baseline

	}

	result := &BenchmarkResult{

		Name: fmt.Sprintf("Concurrent Users: %d", users),

		StartTime: time.Now(),

		Latencies: make([]time.Duration, 0),
	}

	// Start resource monitoring.

	stopMonitor := bs.startResourceMonitoring(ctx, result)

	defer stopMonitor()

	var wg sync.WaitGroup

	var totalRequests int64

	var successCount int64

	var errorCount int64

	latencyChan := make(chan time.Duration, 10000)

	errorChan := make(chan error, 1000)

	// Create worker pool.

	for i := range users {

		wg.Add(1)

		go func(workerID int) {

			defer wg.Done()

			endTime := time.Now().Add(duration)

			for time.Now().Before(endTime) {

				select {

				case <-ctx.Done():

					return

				default:

					// Apply rate limiting.

					bs.rateLimiter.When("request")

					start := time.Now()

					err := workload()

					latency := time.Since(start)

					atomic.AddInt64(&totalRequests, 1)

					latencyChan <- latency

					if err != nil {

						atomic.AddInt64(&errorCount, 1)

						select {

						case errorChan <- err:

						default:

						}

					} else {

						atomic.AddInt64(&successCount, 1)

					}

				}

			}

		}(i)

	}

	// Wait for all workers to complete.

	go func() {

		wg.Wait()

		close(latencyChan)

		close(errorChan)

	}()

	// Collect latencies.

	for latency := range latencyChan {

		result.Latencies = append(result.Latencies, latency)

	}

	// Collect errors.

	for err := range errorChan {

		result.Errors = append(result.Errors, err)

	}

	result.EndTime = time.Now()

	result.Duration = result.EndTime.Sub(result.StartTime)

	result.TotalRequests = totalRequests

	result.SuccessCount = successCount

	result.ErrorCount = errorCount

	// Calculate metrics.

	bs.calculateMetrics(result)

	// Check against baseline.

	result.Passed = bs.checkBaseline(result, baseline)

	bs.mu.Lock()

	bs.results = append(bs.results, *result)

	bs.mu.Unlock()

	return result, nil

}

// RunE2NodeSetScalingBenchmark tests E2NodeSet scaling performance.

func (bs *BenchmarkSuite) RunE2NodeSetScalingBenchmark(ctx context.Context, nodeCount int, scaleFunc func(int) error) (*BenchmarkResult, error) {

	baseline := bs.baselines["e2nodeset_scaling"]

	result := &BenchmarkResult{

		Name: fmt.Sprintf("E2NodeSet Scaling to %d nodes", nodeCount),

		StartTime: time.Now(),
	}

	// Start resource monitoring.

	stopMonitor := bs.startResourceMonitoring(ctx, result)

	defer stopMonitor()

	// Run scaling benchmark.

	start := time.Now()

	err := scaleFunc(nodeCount)

	latency := time.Since(start)

	result.EndTime = time.Now()

	result.Duration = result.EndTime.Sub(result.StartTime)

	result.Iterations = 1

	result.TotalRequests = int64(nodeCount)

	result.Latencies = []time.Duration{latency}

	if err != nil {

		result.ErrorCount = 1

		result.Errors = []error{err}

	} else {

		result.SuccessCount = int64(nodeCount)

	}

	// Calculate throughput as nodes per second.

	result.Throughput = float64(nodeCount) / latency.Seconds()

	// Calculate other metrics.

	bs.calculateMetrics(result)

	// Check against baseline.

	result.Passed = bs.checkBaseline(result, baseline)

	bs.mu.Lock()

	bs.results = append(bs.results, *result)

	bs.mu.Unlock()

	return result, nil

}

// RunMemoryStabilityBenchmark tests for memory leaks over time.

func (bs *BenchmarkSuite) RunMemoryStabilityBenchmark(ctx context.Context, duration time.Duration, workload func() error) (*BenchmarkResult, error) {

	baseline := bs.baselines["memory_stability"]

	result := &BenchmarkResult{

		Name: fmt.Sprintf("Memory Stability Test (%v)", duration),

		StartTime: time.Now(),

		Latencies: make([]time.Duration, 0),
	}

	// Start resource monitoring with higher frequency.

	stopMonitor := bs.startResourceMonitoring(ctx, result)

	defer stopMonitor()

	// Force initial GC to get baseline.

	runtime.GC()

	var initialMem runtime.MemStats

	runtime.ReadMemStats(&initialMem)

	var totalRequests int64

	var successCount int64

	var errorCount int64

	endTime := time.Now().Add(duration)

	for time.Now().Before(endTime) {

		select {

		case <-ctx.Done():

			break

		default:

			start := time.Now()

			err := workload()

			latency := time.Since(start)

			totalRequests++

			result.Latencies = append(result.Latencies, latency)

			if err != nil {

				errorCount++

				result.Errors = append(result.Errors, err)

			} else {

				successCount++

			}

			// Periodic GC to check for leaks.

			if totalRequests%100 == 0 {

				runtime.GC()

			}

		}

	}

	// Final GC and memory check.

	runtime.GC()

	var finalMem runtime.MemStats

	runtime.ReadMemStats(&finalMem)

	result.EndTime = time.Now()

	result.Duration = result.EndTime.Sub(result.StartTime)

	result.TotalRequests = totalRequests

	result.SuccessCount = successCount

	result.ErrorCount = errorCount

	// Check for memory leak.

	memoryGrowth := float64(finalMem.HeapAlloc-initialMem.HeapAlloc) / (1024 * 1024)

	if memoryGrowth > baseline.MaxMemoryMB*0.1 { // 10% growth threshold

		result.RegressionInfo = fmt.Sprintf("Potential memory leak detected: %.2f MB growth", memoryGrowth)

	}

	// Calculate metrics.

	bs.calculateMetrics(result)

	// Check against baseline.

	result.Passed = bs.checkBaseline(result, baseline)

	bs.mu.Lock()

	bs.results = append(bs.results, *result)

	bs.mu.Unlock()

	return result, nil

}

// RunRealisticTelecomWorkload runs a realistic telecom deployment scenario.

func (bs *BenchmarkSuite) RunRealisticTelecomWorkload(ctx context.Context, scenario TelecomScenario) (*BenchmarkResult, error) {

	result := &BenchmarkResult{

		Name: fmt.Sprintf("Telecom Workload: %s", scenario.Name),

		StartTime: time.Now(),

		Latencies: make([]time.Duration, 0),
	}

	// Start resource monitoring.

	stopMonitor := bs.startResourceMonitoring(ctx, result)

	defer stopMonitor()

	// Execute scenario steps.

	for _, step := range scenario.Steps {

		start := time.Now()

		err := step.Execute()

		latency := time.Since(start)

		result.TotalRequests++

		result.Latencies = append(result.Latencies, latency)

		if err != nil {

			result.ErrorCount++

			result.Errors = append(result.Errors, fmt.Errorf("step %s failed: %w", step.Name, err))

		} else {

			result.SuccessCount++

		}

		// Wait between steps if specified.

		if step.Delay > 0 {

			time.Sleep(step.Delay)

		}

	}

	result.EndTime = time.Now()

	result.Duration = result.EndTime.Sub(result.StartTime)

	// Calculate metrics.

	bs.calculateMetrics(result)

	// Use high throughput baseline for realistic scenarios.

	baseline := bs.baselines["high_throughput"]

	result.Passed = bs.checkBaseline(result, baseline)

	bs.mu.Lock()

	bs.results = append(bs.results, *result)

	bs.mu.Unlock()

	return result, nil

}

// TelecomScenario represents a realistic telecom deployment scenario.

type TelecomScenario struct {
	Name string

	Description string

	Steps []ScenarioStep
}

// ScenarioStep represents a single step in a telecom scenario.

type ScenarioStep struct {
	Name string

	Execute func() error

	Delay time.Duration
}

// startResourceMonitoring starts monitoring resources during benchmark.

func (bs *BenchmarkSuite) startResourceMonitoring(ctx context.Context, result *BenchmarkResult) func() {

	stopChan := make(chan struct{})

	go func() {

		ticker := time.NewTicker(100 * time.Millisecond)

		defer ticker.Stop()

		for {

			select {

			case <-ctx.Done():

				return

			case <-stopChan:

				return

			case <-ticker.C:

				// Collect CPU usage.

				cpuPercent := bs.metrics.GetCPUUsage()

				result.ResourceUsage.CPUSamples = append(result.ResourceUsage.CPUSamples, cpuPercent)

				// Collect memory usage.

				var memStats runtime.MemStats

				runtime.ReadMemStats(&memStats)

				result.ResourceUsage.MemorySamples = append(result.ResourceUsage.MemorySamples, memStats.Alloc)

				// Collect goroutine count.

				goroutineCount := runtime.NumGoroutine()

				result.ResourceUsage.GoroutineSamples = append(result.ResourceUsage.GoroutineSamples, goroutineCount)

				// Update peaks.

				memoryMB := float64(memStats.Alloc) / (1024 * 1024)

				if memoryMB > result.PeakMemoryMB {

					result.PeakMemoryMB = memoryMB

				}

				if cpuPercent > result.PeakCPUPercent {

					result.PeakCPUPercent = cpuPercent

				}

				if goroutineCount > result.MaxGoroutines {

					result.MaxGoroutines = goroutineCount

				}

			}

		}

	}()

	return func() {

		close(stopChan)

	}

}

// calculateMetrics calculates performance metrics from raw data.

func (bs *BenchmarkSuite) calculateMetrics(result *BenchmarkResult) {

	if len(result.Latencies) == 0 {

		return

	}

	// Calculate latency percentiles.

	analyzer := bs.metrics.analyzer

	result.LatencyP50 = analyzer.CalculatePercentile(result.Latencies, 50)

	result.LatencyP95 = analyzer.CalculatePercentile(result.Latencies, 95)

	result.LatencyP99 = analyzer.CalculatePercentile(result.Latencies, 99)

	// Calculate throughput.

	if result.Duration > 0 {

		result.Throughput = float64(result.SuccessCount) / result.Duration.Seconds()

	}

	// Calculate average memory.

	if len(result.ResourceUsage.MemorySamples) > 0 {

		var totalMem uint64

		for _, mem := range result.ResourceUsage.MemorySamples {

			totalMem += mem

		}

		result.AvgMemoryMB = float64(totalMem) / float64(len(result.ResourceUsage.MemorySamples)) / (1024 * 1024)

	}

	// Calculate average CPU.

	if len(result.ResourceUsage.CPUSamples) > 0 {

		var totalCPU float64

		for _, cpu := range result.ResourceUsage.CPUSamples {

			totalCPU += cpu

		}

		result.AvgCPUPercent = totalCPU / float64(len(result.ResourceUsage.CPUSamples))

	}

}

// checkBaseline checks if results meet baseline requirements.

func (bs *BenchmarkSuite) checkBaseline(result *BenchmarkResult, baseline PerformanceBaseline) bool {

	passed := true

	var regressions []string

	// Check latency.

	if result.LatencyP50 > baseline.MaxLatencyP50 {

		passed = false

		regressions = append(regressions, fmt.Sprintf("P50 latency %.2fs exceeds baseline %.2fs",

			result.LatencyP50.Seconds(), baseline.MaxLatencyP50.Seconds()))

	}

	if result.LatencyP95 > baseline.MaxLatencyP95 {

		passed = false

		regressions = append(regressions, fmt.Sprintf("P95 latency %.2fs exceeds baseline %.2fs",

			result.LatencyP95.Seconds(), baseline.MaxLatencyP95.Seconds()))

	}

	if result.LatencyP99 > baseline.MaxLatencyP99 {

		passed = false

		regressions = append(regressions, fmt.Sprintf("P99 latency %.2fs exceeds baseline %.2fs",

			result.LatencyP99.Seconds(), baseline.MaxLatencyP99.Seconds()))

	}

	// Check throughput.

	if result.Throughput < baseline.MinThroughput {

		passed = false

		regressions = append(regressions, fmt.Sprintf("Throughput %.2f below baseline %.2f",

			result.Throughput, baseline.MinThroughput))

	}

	// Check memory.

	if result.PeakMemoryMB > baseline.MaxMemoryMB {

		passed = false

		regressions = append(regressions, fmt.Sprintf("Peak memory %.2fMB exceeds baseline %.2fMB",

			result.PeakMemoryMB, baseline.MaxMemoryMB))

	}

	// Check CPU.

	if result.PeakCPUPercent > baseline.MaxCPUPercent {

		passed = false

		regressions = append(regressions, fmt.Sprintf("Peak CPU %.2f%% exceeds baseline %.2f%%",

			result.PeakCPUPercent, baseline.MaxCPUPercent))

	}

	// Check goroutines.

	if result.MaxGoroutines > baseline.MaxGoroutines {

		passed = false

		regressions = append(regressions, fmt.Sprintf("Max goroutines %d exceeds baseline %d",

			result.MaxGoroutines, baseline.MaxGoroutines))

	}

	if len(regressions) > 0 {

		result.RegressionInfo = fmt.Sprintf("Performance regressions detected: %v", regressions)

	}

	return passed

}

// GetResults returns all benchmark results.

func (bs *BenchmarkSuite) GetResults() []BenchmarkResult {

	bs.mu.RLock()

	defer bs.mu.RUnlock()

	return bs.results

}

// GenerateReport generates a comprehensive performance report.

func (bs *BenchmarkSuite) GenerateReport() string {

	bs.mu.RLock()

	defer bs.mu.RUnlock()

	report := "=== Nephoran Intent Operator Performance Report ===\n\n"

	for _, result := range bs.results {

		report += fmt.Sprintf("Benchmark: %s\n", result.Name)

		report += fmt.Sprintf("  Status: %s\n", getStatusString(result.Passed))

		report += fmt.Sprintf("  Duration: %v\n", result.Duration)

		report += fmt.Sprintf("  Total Requests: %d\n", result.TotalRequests)

		report += fmt.Sprintf("  Success Rate: %.2f%%\n", float64(result.SuccessCount)/float64(result.TotalRequests)*100)

		report += fmt.Sprintf("  Throughput: %.2f req/s\n", result.Throughput)

		report += fmt.Sprintf("  Latency P50: %v\n", result.LatencyP50)

		report += fmt.Sprintf("  Latency P95: %v\n", result.LatencyP95)

		report += fmt.Sprintf("  Latency P99: %v\n", result.LatencyP99)

		report += fmt.Sprintf("  Avg Memory: %.2f MB\n", result.AvgMemoryMB)

		report += fmt.Sprintf("  Peak Memory: %.2f MB\n", result.PeakMemoryMB)

		report += fmt.Sprintf("  Avg CPU: %.2f%%\n", result.AvgCPUPercent)

		report += fmt.Sprintf("  Peak CPU: %.2f%%\n", result.PeakCPUPercent)

		report += fmt.Sprintf("  Max Goroutines: %d\n", result.MaxGoroutines)

		if result.RegressionInfo != "" {

			report += fmt.Sprintf("  Regression Info: %s\n", result.RegressionInfo)

		}

		if len(result.Errors) > 0 {

			report += fmt.Sprintf("  Errors: %d\n", len(result.Errors))

		}

		report += "\n"

	}

	return report

}

// getStatusString returns a status string based on passed flag.

func getStatusString(passed bool) string {

	if passed {

		return "PASSED"

	}

	return "FAILED"

}

// ExportMetrics exports benchmark results to Prometheus.

func (bs *BenchmarkSuite) ExportMetrics(registry *prometheus.Registry) {

	bs.mu.RLock()

	defer bs.mu.RUnlock()

	for _, result := range bs.results {

		labels := prometheus.Labels{

			"benchmark": result.Name,

			"status": getStatusString(result.Passed),
		}

		// Export latency metrics.

		latencyGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{

			Name: "benchmark_latency_seconds",

			Help: "Benchmark latency in seconds",
		}, []string{"benchmark", "status", "percentile"})

		latencyGauge.With(prometheus.Labels{"benchmark": result.Name, "status": getStatusString(result.Passed), "percentile": "p50"}).Set(result.LatencyP50.Seconds())

		latencyGauge.With(prometheus.Labels{"benchmark": result.Name, "status": getStatusString(result.Passed), "percentile": "p95"}).Set(result.LatencyP95.Seconds())

		latencyGauge.With(prometheus.Labels{"benchmark": result.Name, "status": getStatusString(result.Passed), "percentile": "p99"}).Set(result.LatencyP99.Seconds())

		registry.MustRegister(latencyGauge)

		// Export throughput.

		throughputGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{

			Name: "benchmark_throughput_rps",

			Help: "Benchmark throughput in requests per second",
		}, []string{"benchmark", "status"})

		throughputGauge.With(labels).Set(result.Throughput)

		registry.MustRegister(throughputGauge)

		// Export resource metrics.

		memoryGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{

			Name: "benchmark_memory_mb",

			Help: "Benchmark memory usage in MB",
		}, []string{"benchmark", "status", "type"})

		memoryGauge.With(prometheus.Labels{"benchmark": result.Name, "status": getStatusString(result.Passed), "type": "avg"}).Set(result.AvgMemoryMB)

		memoryGauge.With(prometheus.Labels{"benchmark": result.Name, "status": getStatusString(result.Passed), "type": "peak"}).Set(result.PeakMemoryMB)

		registry.MustRegister(memoryGauge)

		cpuGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{

			Name: "benchmark_cpu_percent",

			Help: "Benchmark CPU usage percentage",
		}, []string{"benchmark", "status", "type"})

		cpuGauge.With(prometheus.Labels{"benchmark": result.Name, "status": getStatusString(result.Passed), "type": "avg"}).Set(result.AvgCPUPercent)

		cpuGauge.With(prometheus.Labels{"benchmark": result.Name, "status": getStatusString(result.Passed), "type": "peak"}).Set(result.PeakCPUPercent)

		registry.MustRegister(cpuGauge)

	}

}

// CompareWithBaseline compares current results with historical baselines.

func (bs *BenchmarkSuite) CompareWithBaseline(current, historical *BenchmarkResult) (float64, string) {

	if historical == nil {

		return 0, "No historical data available"

	}

	// Calculate performance delta.

	latencyDelta := (current.LatencyP95.Seconds() - historical.LatencyP95.Seconds()) / historical.LatencyP95.Seconds() * 100

	throughputDelta := (current.Throughput - historical.Throughput) / historical.Throughput * 100

	memoryDelta := (current.PeakMemoryMB - historical.PeakMemoryMB) / historical.PeakMemoryMB * 100

	cpuDelta := (current.PeakCPUPercent - historical.PeakCPUPercent) / historical.PeakCPUPercent * 100

	summary := "Performance comparison:\n"

	summary += fmt.Sprintf("  Latency P95: %.2f%% (%.2fs -> %.2fs)\n", latencyDelta, historical.LatencyP95.Seconds(), current.LatencyP95.Seconds())

	summary += fmt.Sprintf("  Throughput: %.2f%% (%.2f -> %.2f rps)\n", throughputDelta, historical.Throughput, current.Throughput)

	summary += fmt.Sprintf("  Memory: %.2f%% (%.2f -> %.2f MB)\n", memoryDelta, historical.PeakMemoryMB, current.PeakMemoryMB)

	summary += fmt.Sprintf("  CPU: %.2f%% (%.2f%% -> %.2f%%)\n", cpuDelta, historical.PeakCPUPercent, current.PeakCPUPercent)

	// Calculate overall performance score (negative is better for latency/resources, positive for throughput).

	overallDelta := (-latencyDelta + throughputDelta - memoryDelta - cpuDelta) / 4

	return overallDelta, summary

}
