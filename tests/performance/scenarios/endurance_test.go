package scenarios

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/performance"
	"k8s.io/klog/v2"
)

// TestLongRunningStability tests system stability over extended periods
func TestLongRunningStability(t *testing.T) {
	suite := performance.NewBenchmarkSuite()
	ctx := context.Background()

	// Test configurations for different durations
	configs := []struct {
		Name          string
		Duration      time.Duration
		WorkloadRate  int // Requests per second
		CheckInterval time.Duration
		MaxMemGrowth  float64 // Maximum allowed memory growth percentage
		MaxGoroutines int
		MaxErrorRate  float64
	}{
		{
			Name:          "10 Minute Stability",
			Duration:      10 * time.Minute,
			WorkloadRate:  10,
			CheckInterval: 1 * time.Minute,
			MaxMemGrowth:  10.0,
			MaxGoroutines: 500,
			MaxErrorRate:  0.01, // 1%
		},
		{
			Name:          "1 Hour Endurance",
			Duration:      1 * time.Hour,
			WorkloadRate:  5,
			CheckInterval: 5 * time.Minute,
			MaxMemGrowth:  20.0,
			MaxGoroutines: 1000,
			MaxErrorRate:  0.02,
		},
		{
			Name:          "24 Hour Soak Test",
			Duration:      24 * time.Hour,
			WorkloadRate:  2,
			CheckInterval: 30 * time.Minute,
			MaxMemGrowth:  30.0,
			MaxGoroutines: 2000,
			MaxErrorRate:  0.03,
		},
	}

	for _, config := range configs {
		// Skip long tests in short mode
		if testing.Short() && config.Duration > 10*time.Minute {
			t.Skipf("Skipping %s in short mode", config.Name)
			continue
		}

		t.Run(config.Name, func(t *testing.T) {
			result := runEnduranceTest(ctx, suite, config)

			// Validate results
			if result.MemoryLeakDetected {
				t.Errorf("%s: Memory leak detected - growth: %.2f%%",
					config.Name, result.MemoryGrowthPercent)
			}

			if result.GoroutineLeakDetected {
				t.Errorf("%s: Goroutine leak detected - peak: %d",
					config.Name, result.PeakGoroutines)
			}

			if result.ErrorRate > config.MaxErrorRate {
				t.Errorf("%s: Error rate %.2f%% exceeded threshold %.2f%%",
					config.Name, result.ErrorRate*100, config.MaxErrorRate*100)
			}

			if !result.PerformanceStable {
				t.Errorf("%s: Performance degraded over time", config.Name)
			}

			t.Logf("%s Results:", config.Name)
			t.Logf("  Duration: %v", result.ActualDuration)
			t.Logf("  Total Requests: %d", result.TotalRequests)
			t.Logf("  Success Rate: %.2f%%", (1-result.ErrorRate)*100)
			t.Logf("  Memory Growth: %.2f%%", result.MemoryGrowthPercent)
			t.Logf("  Peak Goroutines: %d", result.PeakGoroutines)
			t.Logf("  Avg Response Time: %v", result.AvgResponseTime)
			t.Logf("  P99 Response Time: %v", result.P99ResponseTime)
			t.Logf("  Performance Stable: %v", result.PerformanceStable)
		})
	}
}

// TestMemoryLeakDetection specifically tests for memory leaks
func TestMemoryLeakDetection(t *testing.T) {
	ctx := context.Background()

	// Run memory-intensive operations
	workload := func() error {
		// Simulate operations that might leak memory
		data := make([]byte, 1024*1024) // 1MB allocation
		for i := range data {
			data[i] = byte(i % 256)
		}

		// Process data (simulate work)
		time.Sleep(10 * time.Millisecond)

		// In a leak scenario, we might forget to release resources
		// Here we properly let it go out of scope
		return nil
	}

	suite := performance.NewBenchmarkSuite()
	result, err := suite.RunMemoryStabilityBenchmark(ctx, 10*time.Minute, workload)

	if err != nil {
		t.Fatalf("Memory stability test failed: %v", err)
	}

	if !result.Passed {
		t.Errorf("Memory leak detected: %s", result.RegressionInfo)
	}

	// Analyze memory profile
	profiler := performance.NewProfiler()
	memProfile, err := profiler.CaptureMemoryProfile()
	if err != nil {
		t.Logf("Failed to capture memory profile: %v", err)
	} else {
		t.Logf("Memory profile saved to: %s", memProfile)
	}

	// Check for specific leak patterns
	leaks := profiler.DetectMemoryLeaks()
	if len(leaks) > 0 {
		t.Errorf("Detected %d potential memory leaks:", len(leaks))
		for _, leak := range leaks {
			t.Logf("  - %s: %d bytes at %s",
				leak.Description, leak.AllocBytes, leak.Location)
		}
	}
}

// TestGoroutineLeakDetection tests for goroutine leaks
func TestGoroutineLeakDetection(t *testing.T) {
	initialGoroutines := runtime.NumGoroutine()

	// Run operations that spawn goroutines
	var wg sync.WaitGroup
	stopChan := make(chan struct{})

	// Properly managed goroutines
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			select {
			case <-stopChan:
				return
			case <-time.After(5 * time.Second):
				// Simulate work
			}
		}(i)
	}

	// Let goroutines run
	time.Sleep(1 * time.Second)

	// Check for leaks before cleanup
	midGoroutines := runtime.NumGoroutine()
	t.Logf("Goroutines - Initial: %d, Mid-test: %d", initialGoroutines, midGoroutines)

	// Cleanup
	close(stopChan)
	wg.Wait()

	// Wait for goroutines to fully terminate
	time.Sleep(1 * time.Second)

	// Final check
	finalGoroutines := runtime.NumGoroutine()
	leaked := finalGoroutines - initialGoroutines

	if leaked > 10 { // Allow small variance
		t.Errorf("Goroutine leak detected: %d goroutines leaked", leaked)

		// Capture goroutine profile for debugging
		profiler := performance.NewProfiler()
		profile, err := profiler.CaptureGoroutineProfile()
		if err == nil {
			t.Logf("Goroutine profile saved to: %s", profile)
		}
	}

	t.Logf("Goroutines - Initial: %d, Final: %d, Leaked: %d",
		initialGoroutines, finalGoroutines, leaked)
}

// TestResourceExhaustion tests system behavior under resource exhaustion
func TestResourceExhaustion(t *testing.T) {
	ctx := context.Background()

	scenarios := []struct {
		Name         string
		ResourceType string
		ExhaustFunc  func(context.Context) error
		RecoverFunc  func() error
		MaxRecovery  time.Duration
	}{
		{
			Name:         "CPU Exhaustion",
			ResourceType: "CPU",
			ExhaustFunc:  exhaustCPU,
			RecoverFunc:  recoverCPU,
			MaxRecovery:  30 * time.Second,
		},
		{
			Name:         "Memory Exhaustion",
			ResourceType: "Memory",
			ExhaustFunc:  exhaustMemory,
			RecoverFunc:  recoverMemory,
			MaxRecovery:  45 * time.Second,
		},
		{
			Name:         "Goroutine Exhaustion",
			ResourceType: "Goroutines",
			ExhaustFunc:  exhaustGoroutines,
			RecoverFunc:  recoverGoroutines,
			MaxRecovery:  20 * time.Second,
		},
		{
			Name:         "File Descriptor Exhaustion",
			ResourceType: "FileDescriptors",
			ExhaustFunc:  exhaustFileDescriptors,
			RecoverFunc:  recoverFileDescriptors,
			MaxRecovery:  15 * time.Second,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			// Monitor system during exhaustion
			result := testResourceExhaustion(ctx, scenario)

			if !result.Survived {
				t.Errorf("%s: System crashed under %s exhaustion",
					scenario.Name, scenario.ResourceType)
			}

			if result.RecoveryTime > scenario.MaxRecovery {
				t.Errorf("%s: Recovery time %v exceeded max %v",
					scenario.Name, result.RecoveryTime, scenario.MaxRecovery)
			}

			t.Logf("%s Results:", scenario.Name)
			t.Logf("  Survived: %v", result.Survived)
			t.Logf("  Recovery Time: %v", result.RecoveryTime)
			t.Logf("  Degraded Services: %v", result.DegradedServices)
			t.Logf("  Error Messages: %d", len(result.Errors))
		})
	}
}

// TestPerformanceDegradation tests for gradual performance degradation
func TestPerformanceDegradation(t *testing.T) {
	ctx := context.Background()
	suite := performance.NewBenchmarkSuite()

	// Collect performance metrics over time
	intervals := []time.Duration{
		1 * time.Minute,
		5 * time.Minute,
		10 * time.Minute,
		30 * time.Minute,
		1 * time.Hour,
	}

	var previousLatency time.Duration
	degradationDetected := false

	for i, interval := range intervals {
		if testing.Short() && interval > 5*time.Minute {
			break
		}

		// Run workload for interval
		workload := func() error {
			// Simulate typical operation
			time.Sleep(50 * time.Millisecond)
			return nil
		}

		result, err := suite.RunConcurrentBenchmark(ctx, 10, interval, workload)
		if err != nil {
			t.Fatalf("Benchmark at %v failed: %v", interval, err)
		}

		// Check for degradation
		if i > 0 && result.LatencyP95 > time.Duration(float64(previousLatency)*1.2) { // 20% degradation
			degradationDetected = true
			t.Logf("Performance degradation detected at %v: P95 increased from %v to %v",
				interval, previousLatency, result.LatencyP95)
		}

		previousLatency = result.LatencyP95

		t.Logf("Performance at %v:", interval)
		t.Logf("  P50 Latency: %v", result.LatencyP50)
		t.Logf("  P95 Latency: %v", result.LatencyP95)
		t.Logf("  P99 Latency: %v", result.LatencyP99)
		t.Logf("  Throughput: %.2f req/s", result.Throughput)
		t.Logf("  Memory: %.2f MB", result.PeakMemoryMB)
	}

	if degradationDetected {
		t.Error("Performance degradation detected over time")
	}
}

// EnduranceTestResult captures endurance test results
type EnduranceTestResult struct {
	ActualDuration        time.Duration
	TotalRequests         int64
	ErrorRate             float64
	MemoryGrowthPercent   float64
	MemoryLeakDetected    bool
	GoroutineLeakDetected bool
	PeakGoroutines        int
	AvgResponseTime       time.Duration
	P99ResponseTime       time.Duration
	PerformanceStable     bool
	CheckpointMetrics     []CheckpointMetric
}

// CheckpointMetric captures metrics at a checkpoint
type CheckpointMetric struct {
	Timestamp  time.Time
	MemoryMB   float64
	Goroutines int
	AvgLatency time.Duration
	ErrorRate  float64
	Throughput float64
}

// ResourceExhaustionResult captures resource exhaustion test results
type ResourceExhaustionResult struct {
	Survived         bool
	RecoveryTime     time.Duration
	DegradedServices []string
	Errors           []error
}

// runEnduranceTest executes a long-running stability test
func runEnduranceTest(ctx context.Context, suite *performance.BenchmarkSuite, config struct {
	Name          string
	Duration      time.Duration
	WorkloadRate  int
	CheckInterval time.Duration
	MaxMemGrowth  float64
	MaxGoroutines int
	MaxErrorRate  float64
}) EnduranceTestResult {

	result := EnduranceTestResult{
		CheckpointMetrics: []CheckpointMetric{},
	}

	// Capture initial state
	runtime.GC()
	var initialMem runtime.MemStats
	runtime.ReadMemStats(&initialMem)
	initialGoroutines := runtime.NumGoroutine()

	// Metrics tracking
	var totalRequests int64
	var errorCount int64
	var totalLatency int64
	var peakGoroutines int32

	// Start time
	startTime := time.Now()
	endTime := startTime.Add(config.Duration)

	// Create worker pool
	var wg sync.WaitGroup
	stopChan := make(chan struct{})
	latencyChan := make(chan time.Duration, 10000)

	// Worker function
	worker := func(id int) {
		defer wg.Done()
		ticker := time.NewTicker(time.Second / time.Duration(config.WorkloadRate))
		defer ticker.Stop()

		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				start := time.Now()
				err := simulateWorkload(ctx)
				latency := time.Since(start)

				atomic.AddInt64(&totalRequests, 1)
				atomic.AddInt64(&totalLatency, int64(latency))

				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				}

				select {
				case latencyChan <- latency:
				default:
				}

				// Update peak goroutines
				current := int32(runtime.NumGoroutine())
				for {
					peak := atomic.LoadInt32(&peakGoroutines)
					if current <= peak || atomic.CompareAndSwapInt32(&peakGoroutines, peak, current) {
						break
					}
				}
			}
		}
	}

	// Start workers
	workerCount := 10
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go worker(i)
	}

	// Checkpoint monitoring
	checkpointTicker := time.NewTicker(config.CheckInterval)
	defer checkpointTicker.Stop()

	// Performance tracking
	var previousLatency time.Duration
	performanceStable := true

	// Monitor loop
monitorLoop:
	for {
		select {
		case <-ctx.Done():
			break monitorLoop
		case <-checkpointTicker.C:
			// Capture checkpoint metrics
			runtime.GC()
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)

			currentRequests := atomic.LoadInt64(&totalRequests)
			currentErrors := atomic.LoadInt64(&errorCount)
			currentLatency := atomic.LoadInt64(&totalLatency)

			checkpoint := CheckpointMetric{
				Timestamp:  time.Now(),
				MemoryMB:   float64(memStats.Alloc) / (1024 * 1024),
				Goroutines: runtime.NumGoroutine(),
				ErrorRate:  float64(currentErrors) / float64(currentRequests),
				Throughput: float64(currentRequests) / time.Since(startTime).Seconds(),
			}

			if currentRequests > 0 {
				checkpoint.AvgLatency = time.Duration(currentLatency / currentRequests)
			}

			result.CheckpointMetrics = append(result.CheckpointMetrics, checkpoint)

			// Check for performance degradation
			if previousLatency > 0 && checkpoint.AvgLatency > time.Duration(float64(previousLatency)*1.5) {
				performanceStable = false
				klog.Warningf("Performance degradation detected: latency increased from %v to %v",
					previousLatency, checkpoint.AvgLatency)
			}
			previousLatency = checkpoint.AvgLatency

			// Log checkpoint
			klog.Infof("Checkpoint at %v: Mem=%.2fMB, Goroutines=%d, ErrorRate=%.2f%%, AvgLatency=%v",
				time.Since(startTime), checkpoint.MemoryMB, checkpoint.Goroutines,
				checkpoint.ErrorRate*100, checkpoint.AvgLatency)

		default:
			if time.Now().After(endTime) {
				break monitorLoop
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Stop workers
	close(stopChan)
	wg.Wait()
	close(latencyChan)

	// Final metrics
	runtime.GC()
	var finalMem runtime.MemStats
	runtime.ReadMemStats(&finalMem)

	result.ActualDuration = time.Since(startTime)
	result.TotalRequests = atomic.LoadInt64(&totalRequests)
	result.ErrorRate = float64(atomic.LoadInt64(&errorCount)) / float64(result.TotalRequests)
	result.PeakGoroutines = int(atomic.LoadInt32(&peakGoroutines))
	result.PerformanceStable = performanceStable

	// Calculate memory growth
	memoryGrowth := float64(finalMem.Alloc-initialMem.Alloc) / float64(initialMem.Alloc) * 100
	result.MemoryGrowthPercent = memoryGrowth
	result.MemoryLeakDetected = memoryGrowth > config.MaxMemGrowth

	// Check for goroutine leak
	finalGoroutines := runtime.NumGoroutine()
	goroutineGrowth := finalGoroutines - initialGoroutines
	result.GoroutineLeakDetected = goroutineGrowth > 50 || finalGoroutines > config.MaxGoroutines

	// Calculate latency percentiles
	latencies := []time.Duration{}
	for latency := range latencyChan {
		latencies = append(latencies, latency)
	}

	if len(latencies) > 0 {
		analyzer := performance.NewMetricsAnalyzer()
		result.AvgResponseTime = analyzer.CalculateAverage(latencies)
		result.P99ResponseTime = analyzer.CalculatePercentile(latencies, 99)
	}

	return result
}

// testResourceExhaustion tests behavior under resource exhaustion
func testResourceExhaustion(ctx context.Context, scenario struct {
	Name         string
	ResourceType string
	ExhaustFunc  func(context.Context) error
	RecoverFunc  func() error
	MaxRecovery  time.Duration
}) ResourceExhaustionResult {

	result := ResourceExhaustionResult{
		Survived:         true,
		DegradedServices: []string{},
		Errors:           []error{},
	}

	// Start monitoring
	stopMonitor := make(chan struct{})
	var monitorWg sync.WaitGroup

	monitorWg.Add(1)
	go func() {
		defer monitorWg.Done()
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stopMonitor:
				return
			case <-ticker.C:
				// Check system health
				if err := checkSystemHealth(); err != nil {
					result.Errors = append(result.Errors, err)
					result.DegradedServices = append(result.DegradedServices, "health_check")
				}
			}
		}
	}()

	// Exhaust resource
	klog.Infof("Exhausting %s", scenario.ResourceType)
	// exhaustStart := time.Now() // Currently unused, commented out to avoid lint warning

	if err := scenario.ExhaustFunc(ctx); err != nil {
		result.Errors = append(result.Errors, err)
		result.Survived = false
	}

	// Try to recover
	klog.Infof("Recovering from %s exhaustion", scenario.ResourceType)
	recoverStart := time.Now()

	if err := scenario.RecoverFunc(); err != nil {
		result.Errors = append(result.Errors, err)
	}

	result.RecoveryTime = time.Since(recoverStart)

	// Stop monitoring
	close(stopMonitor)
	monitorWg.Wait()

	// Final health check
	if err := checkSystemHealth(); err != nil {
		result.Survived = false
	}

	return result
}

// Helper functions for resource exhaustion

func exhaustCPU(ctx context.Context) error {
	// Spawn CPU-intensive goroutines
	numCPU := runtime.NumCPU()
	stopChan := make(chan struct{})

	for i := 0; i < numCPU*2; i++ {
		go func() {
			for {
				select {
				case <-stopChan:
					return
				default:
					// CPU-intensive work
					for j := 0; j < 1000000; j++ {
						_ = j * j
					}
				}
			}
		}()
	}

	// Run for a period
	time.Sleep(10 * time.Second)
	close(stopChan)
	return nil
}

func recoverCPU() error {
	// Allow CPU to recover
	runtime.GC()
	time.Sleep(5 * time.Second)
	return nil
}

func exhaustMemory(ctx context.Context) error {
	// Allocate large amounts of memory
	allocations := [][]byte{}

	for i := 0; i < 100; i++ {
		// Allocate 10MB chunks
		data := make([]byte, 10*1024*1024)
		allocations = append(allocations, data)

		// Fill with data to ensure allocation
		for j := range data {
			data[j] = byte(j % 256)
		}

		time.Sleep(100 * time.Millisecond)
	}

	// Hold for a moment
	time.Sleep(5 * time.Second)

	// Release
	allocations = nil
	return nil
}

func recoverMemory() error {
	// Force garbage collection
	runtime.GC()
	runtime.GC() // Run twice to ensure cleanup
	time.Sleep(5 * time.Second)
	return nil
}

func exhaustGoroutines(ctx context.Context) error {
	// Spawn many goroutines
	var wg sync.WaitGroup
	stopChan := make(chan struct{})

	for i := 0; i < 10000; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			select {
			case <-stopChan:
			case <-time.After(30 * time.Second):
			}
		}(i)
	}

	// Let them run
	time.Sleep(5 * time.Second)

	// Clean up
	close(stopChan)
	wg.Wait()
	return nil
}

func recoverGoroutines() error {
	// Wait for goroutines to terminate
	time.Sleep(5 * time.Second)
	runtime.GC()
	return nil
}

func exhaustFileDescriptors(ctx context.Context) error {
	// Note: This is a simplified simulation
	// In real tests, would open actual files/sockets
	klog.Info("Simulating file descriptor exhaustion")
	time.Sleep(5 * time.Second)
	return nil
}

func recoverFileDescriptors() error {
	klog.Info("Recovering from file descriptor exhaustion")
	time.Sleep(3 * time.Second)
	return nil
}

func simulateWorkload(ctx context.Context) error {
	// Simulate typical workload
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(50+time.Now().UnixNano()%50) * time.Millisecond):
		// Variable latency between 50-100ms
		return nil
	}
}

func checkSystemHealth() error {
	// Simple health check
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Check memory pressure
	if memStats.Sys > 10*1024*1024*1024 { // 10GB threshold
		return fmt.Errorf("high memory usage: %d MB", memStats.Sys/(1024*1024))
	}

	// Check goroutine count
	if runtime.NumGoroutine() > 10000 {
		return fmt.Errorf("high goroutine count: %d", runtime.NumGoroutine())
	}

	return nil
}
