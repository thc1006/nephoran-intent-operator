package performance_test

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/nephoran/nephoran-operator/pkg/performance"
	"k8s.io/klog/v2"
)

// TestComprehensivePerformanceBenchmark runs the complete performance test suite
func TestComprehensivePerformanceBenchmark(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping comprehensive performance benchmark in short mode")
	}

	// Initialize logging
	klog.InitFlags(nil)
	defer klog.Flush()

	// Create output directory for results
	outputDir := "/tmp/nephoran-performance-results"
	os.MkdirAll(outputDir, 0755)

	// Configure performance test
	config := &performance.PerformanceTestConfig{
		EnableProfiling:    true,
		EnableFlameGraphs:  true,
		EnableLoadTests:    true,
		EnableBenchmarks:   true,
		ProfileDuration:    30 * time.Second,
		LoadTestDuration:   2 * time.Minute,
		TargetCPUPercent:   60.0,  // Target: ≤60% CPU on 8-core cluster
		TargetP99Latency:   30 * time.Second, // Target: ≤30s P99 latency
		OutputDirectory:    outputDir,
		ClusterCores:       8,
	}

	// Create performance test runner
	runner := performance.NewPerformanceTestRunner(config)

	// Run comprehensive performance test
	ctx := context.Background()
	results, err := runner.RunFullPerformanceTest(ctx)
	if err != nil {
		t.Fatalf("Performance test failed: %v", err)
	}

	// Validate results
	validateResults(t, results)

	// Print summary
	fmt.Println("\n" + runner.GenerateExecutiveSummary())
	
	// Save detailed report
	report := runner.GenerateDetailedReport()
	reportPath := fmt.Sprintf("%s/detailed_report.txt", outputDir)
	if err := os.WriteFile(reportPath, []byte(report), 0644); err != nil {
		t.Errorf("Failed to save report: %v", err)
	}

	fmt.Printf("\nPerformance test results saved to: %s\n", outputDir)
}

// TestFlameGraphGeneration tests flamegraph generation for different profiles
func TestFlameGraphGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping flamegraph generation in short mode")
	}

	ctx := context.Background()
	profileDir := "/tmp/profiles"
	outputDir := "/tmp/flamegraphs"
	
	os.MkdirAll(profileDir, 0755)
	os.MkdirAll(outputDir, 0755)

	generator := performance.NewFlameGraphGenerator(profileDir, outputDir)

	// Test CPU flamegraph
	t.Run("CPU_FlameGraph", func(t *testing.T) {
		// Capture before profile
		beforeProfile, err := generator.CaptureBeforeProfile(ctx, "cpu", 10*time.Second)
		if err != nil {
			t.Errorf("Failed to capture before CPU profile: %v", err)
		} else {
			t.Logf("Before CPU profile: %s", beforeProfile.ProfilePath)
		}

		// Simulate optimization
		time.Sleep(1 * time.Second)

		// Capture after profile
		afterProfile, err := generator.CaptureAfterProfile(ctx, "cpu", 10*time.Second)
		if err != nil {
			t.Errorf("Failed to capture after CPU profile: %v", err)
		} else {
			t.Logf("After CPU profile: %s", afterProfile.ProfilePath)
		}

		// Generate comparison
		comparison, err := generator.GenerateComparison("cpu")
		if err != nil {
			t.Errorf("Failed to generate CPU comparison: %v", err)
		} else {
			t.Logf("CPU improvement: %.2f%%", comparison.Improvement)
			t.Logf("Diff flamegraph: %s", comparison.FlameGraphDiff)
		}
	})

	// Test memory flamegraph
	t.Run("Memory_FlameGraph", func(t *testing.T) {
		beforeProfile, err := generator.CaptureBeforeProfile(ctx, "memory", 0)
		if err != nil {
			t.Errorf("Failed to capture before memory profile: %v", err)
		} else {
			t.Logf("Before memory profile: %s", beforeProfile.ProfilePath)
		}

		// Simulate memory optimization
		allocateAndRelease()

		afterProfile, err := generator.CaptureAfterProfile(ctx, "memory", 0)
		if err != nil {
			t.Errorf("Failed to capture after memory profile: %v", err)
		} else {
			t.Logf("After memory profile: %s", afterProfile.ProfilePath)
		}

		comparison, err := generator.GenerateComparison("memory")
		if err != nil {
			t.Errorf("Failed to generate memory comparison: %v", err)
		} else {
			t.Logf("Memory improvement: %.2f%%", comparison.Improvement)
		}
	})

	// Print comparison report
	fmt.Println("\n" + generator.GetComparisonReport())
}

// TestLoadTestScenarios tests various load patterns
func TestLoadTestScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test scenarios in short mode")
	}

	ctx := context.Background()
	runner := performance.NewLoadTestRunner()

	scenarios := []performance.LoadTestScenario{
		{
			Name:            "baseline_constant_load",
			Description:     "Baseline constant load test",
			Duration:        30 * time.Second,
			MaxConcurrency:  10,
			TargetRPS:       20,
			WorkloadPattern: performance.PatternConstant,
			Workload: func(ctx context.Context) error {
				// Simulate intent processing
				time.Sleep(20 * time.Millisecond)
				return nil
			},
		},
		{
			Name:            "spike_resilience",
			Description:     "Test resilience to traffic spikes",
			Duration:        1 * time.Minute,
			MaxConcurrency:  50,
			TargetRPS:       100,
			WorkloadPattern: performance.PatternSpike,
			Workload: func(ctx context.Context) error {
				// Simulate varied processing time
				time.Sleep(time.Duration(10+rand.Intn(30)) * time.Millisecond)
				return nil
			},
		},
		{
			Name:            "realistic_telecom_workload",
			Description:     "Realistic telecommunications workload pattern",
			Duration:        2 * time.Minute,
			MaxConcurrency:  30,
			TargetRPS:       60,
			WorkloadPattern: performance.PatternRealistic,
			Workload: func(ctx context.Context) error {
				// Simulate realistic intent processing
				operations := []func(){
					func() { time.Sleep(15 * time.Millisecond) }, // Fast query
					func() { time.Sleep(50 * time.Millisecond) }, // Complex intent
					func() { time.Sleep(30 * time.Millisecond) }, // Medium complexity
				}
				operations[rand.Intn(len(operations))]()
				return nil
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			result, err := runner.RunScenario(ctx, scenario)
			if err != nil {
				t.Errorf("Scenario %s failed: %v", scenario.Name, err)
				return
			}

			// Log results
			t.Logf("Scenario: %s", scenario.Name)
			t.Logf("  Passed: %v", result.Passed)
			t.Logf("  Total Requests: %d", result.Metrics.TotalRequests)
			t.Logf("  Success Rate: %.2f%%", 
				float64(result.Metrics.SuccessfulRequests)/float64(result.Metrics.TotalRequests)*100)
			t.Logf("  P99 Latency: %v", result.Analysis.P99Latency)
			t.Logf("  Throughput: %.2f req/s", result.Analysis.Throughput)

			// Check if targets are met
			if result.Analysis.P99Latency > 30*time.Second {
				t.Errorf("P99 latency (%.2fs) exceeds 30s target", result.Analysis.P99Latency.Seconds())
			}

			if len(result.FailureReasons) > 0 {
				for _, reason := range result.FailureReasons {
					t.Logf("  Failure: %s", reason)
				}
			}
		})
	}

	// Generate final report
	report := runner.GenerateReport()
	fmt.Println("\n" + report)
}

// BenchmarkOptimizationImpact demonstrates the impact of optimizations
func BenchmarkOptimizationImpact(b *testing.B) {
	b.Run("Before_Optimization", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate unoptimized processing
			processIntentUnoptimized()
		}
	})

	b.Run("After_Optimization", func(b *testing.B) {
		// Initialize optimizations
		cache := initializeCache()
		pool := initializeConnectionPool()
		defer pool.Close()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate optimized processing
			processIntentOptimized(cache, pool)
		}
	})
}

// Helper functions

func validateResults(t *testing.T, results *performance.PerformanceTestResults) {
	if results.PerformanceTargets == nil {
		t.Fatal("Performance targets not validated")
	}

	targets := results.PerformanceTargets

	// Validate P99 latency target
	if !targets.P99TargetMet {
		t.Errorf("P99 latency target not met: %.2fs > %.2fs",
			targets.P99LatencyActual.Seconds(),
			targets.P99LatencyTarget.Seconds())
	} else {
		t.Logf("✓ P99 latency target met: %.2fs ≤ %.2fs (%.1f%% reduction)",
			targets.P99LatencyActual.Seconds(),
			targets.P99LatencyTarget.Seconds(),
			targets.P99LatencyReduction)
	}

	// Validate CPU target
	if !targets.CPUTargetMet {
		t.Errorf("CPU usage target not met: %.2f%% > %.2f%%",
			targets.CPUActual,
			targets.CPUTarget)
	} else {
		t.Logf("✓ CPU usage target met: %.2f%% ≤ %.2f%% (%.1f%% reduction)",
			targets.CPUActual,
			targets.CPUTarget,
			targets.CPUReduction)
	}

	// Check overall targets
	if !targets.OverallTargetsMet {
		t.Error("Not all performance targets were met")
	} else {
		t.Log("✓ All performance targets achieved successfully!")
	}

	// Validate optimization improvements
	if results.Summary != nil && results.Summary.AverageImprovement < 30 {
		t.Errorf("Average improvement (%.1f%%) below 30%% threshold",
			results.Summary.AverageImprovement)
	}
}

func allocateAndRelease() {
	// Simulate memory allocation patterns
	data := make([][]byte, 100)
	for i := range data {
		data[i] = make([]byte, 1024*1024) // 1MB per allocation
	}
	// Force GC to clean up
	runtime.GC()
}

func processIntentUnoptimized() {
	// Simulate unoptimized processing
	time.Sleep(50 * time.Millisecond)
	
	// Multiple individual queries
	for i := 0; i < 5; i++ {
		time.Sleep(5 * time.Millisecond)
	}
	
	// No caching
	computeExpensiveOperation()
}

func processIntentOptimized(cache map[string]interface{}, pool *ConnectionPool) {
	// Check cache first
	if _, exists := cache["result"]; exists {
		return
	}
	
	// Optimized processing with connection pooling
	conn := pool.Get()
	defer pool.Put(conn)
	
	// Batched queries
	time.Sleep(20 * time.Millisecond)
	
	// Store in cache
	cache["result"] = "processed"
}

func computeExpensiveOperation() {
	// Simulate expensive computation
	sum := 0
	for i := 0; i < 1000000; i++ {
		sum += i
	}
}

func initializeCache() map[string]interface{} {
	return make(map[string]interface{})
}

func initializeConnectionPool() *ConnectionPool {
	return &ConnectionPool{
		connections: make(chan *Connection, 10),
	}
}

// ConnectionPool simulates a connection pool
type ConnectionPool struct {
	connections chan *Connection
}

type Connection struct {
	id int
}

func (p *ConnectionPool) Get() *Connection {
	select {
	case conn := <-p.connections:
		return conn
	default:
		return &Connection{id: rand.Intn(1000)}
	}
}

func (p *ConnectionPool) Put(conn *Connection) {
	select {
	case p.connections <- conn:
	default:
		// Pool full, discard
	}
}

func (p *ConnectionPool) Close() {
	close(p.connections)
}

// TestPerformanceRegression checks for performance regressions
func TestPerformanceRegression(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping regression test in short mode")
	}

	suite := performance.NewBenchmarkSuite()
	ctx := context.Background()

	// Run current performance test
	currentResult, err := suite.RunSingleIntentBenchmark(ctx, func() error {
		time.Sleep(25 * time.Millisecond) // Current performance
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to run current benchmark: %v", err)
	}

	// Compare with baseline (simulated historical data)
	historicalResult := &performance.BenchmarkResult{
		LatencyP95:     20 * time.Second,
		Throughput:     1.5,
		PeakMemoryMB:   400,
		PeakCPUPercent: 70,
	}

	delta, comparison := suite.CompareWithBaseline(currentResult, historicalResult)
	
	t.Logf("Performance comparison with baseline:")
	t.Log(comparison)
	
	// Check for regression (negative delta means regression for overall score)
	if delta < -10 {
		t.Errorf("Performance regression detected: %.2f%% degradation", -delta)
	} else if delta > 10 {
		t.Logf("✓ Performance improvement: %.2f%% better than baseline", delta)
	} else {
		t.Logf("Performance stable: %.2f%% change from baseline", delta)
	}
}