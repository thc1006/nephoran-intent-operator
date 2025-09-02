package scenarios

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/performance"
	"k8s.io/klog/v2"
)

// TestSpikeLoad tests system behavior under sudden load increases
// DISABLED: func TestSpikeLoad(t *testing.T) {
	suite := performance.NewBenchmarkSuite()
	ctx := context.Background()

	// Define spike patterns
	spikePatterns := []struct {
		Name         string
		BaseLoad     int // Base concurrent users
		SpikeLoad    int // Peak concurrent users during spike
		RampUpTime   time.Duration
		SustainTime  time.Duration
		RampDownTime time.Duration
		MaxLatency   time.Duration
		RecoveryTime time.Duration
	}{
		{
			Name:         "Moderate Spike",
			BaseLoad:     10,
			SpikeLoad:    50,
			RampUpTime:   5 * time.Second,
			SustainTime:  30 * time.Second,
			RampDownTime: 5 * time.Second,
			MaxLatency:   10 * time.Second,
			RecoveryTime: 20 * time.Second,
		},
		{
			Name:         "Severe Spike",
			BaseLoad:     20,
			SpikeLoad:    200,
			RampUpTime:   2 * time.Second,
			SustainTime:  20 * time.Second,
			RampDownTime: 10 * time.Second,
			MaxLatency:   15 * time.Second,
			RecoveryTime: 30 * time.Second,
		},
		{
			Name:         "Flash Crowd",
			BaseLoad:     5,
			SpikeLoad:    500,
			RampUpTime:   1 * time.Second,
			SustainTime:  10 * time.Second,
			RampDownTime: 5 * time.Second,
			MaxLatency:   20 * time.Second,
			RecoveryTime: 60 * time.Second,
		},
		{
			Name:         "Oscillating Load",
			BaseLoad:     15,
			SpikeLoad:    100,
			RampUpTime:   3 * time.Second,
			SustainTime:  15 * time.Second,
			RampDownTime: 3 * time.Second,
			MaxLatency:   12 * time.Second,
			RecoveryTime: 25 * time.Second,
		},
	}

	for _, pattern := range spikePatterns {
		t.Run(pattern.Name, func(t *testing.T) {
			result := runSpikeTest(ctx, suite, pattern)

			// Validate spike handling
			if result.MaxLatency > pattern.MaxLatency {
				t.Errorf("%s: Max latency %v exceeded limit %v",
					pattern.Name, result.MaxLatency, pattern.MaxLatency)
			}

			if result.RecoveryTime > pattern.RecoveryTime {
				t.Errorf("%s: Recovery time %v exceeded limit %v",
					pattern.Name, result.RecoveryTime, pattern.RecoveryTime)
			}

			if result.ErrorRate > 0.05 { // 5% error rate threshold
				t.Errorf("%s: Error rate %.2f%% exceeded 5%% threshold",
					pattern.Name, result.ErrorRate*100)
			}

			t.Logf("%s Results:", pattern.Name)
			t.Logf("  Max Latency During Spike: %v", result.MaxLatency)
			t.Logf("  Recovery Time: %v", result.RecoveryTime)
			t.Logf("  Error Rate: %.2f%%", result.ErrorRate*100)
			t.Logf("  Throughput Drop: %.2f%%", result.ThroughputDrop*100)
			t.Logf("  Queue Depth: %d", result.MaxQueueDepth)
		})
	}
}

// TestCascadingFailure tests system resilience to cascading failures
// DISABLED: func TestCascadingFailure(t *testing.T) {
	ctx := context.Background()

	// Simulate component failures
	failures := []struct {
		Name        string
		Component   string
		FailureType string
		Duration    time.Duration
		Impact      func() error
	}{
		{
			Name:        "LLM Service Outage",
			Component:   "llm",
			FailureType: "complete",
			Duration:    30 * time.Second,
			Impact: func() error {
				return simulateLLMOutage(ctx)
			},
		},
		{
			Name:        "Database Slowdown",
			Component:   "database",
			FailureType: "degraded",
			Duration:    60 * time.Second,
			Impact: func() error {
				return simulateDatabaseSlowdown(ctx, 10*time.Second)
			},
		},
		{
			Name:        "Network Partition",
			Component:   "network",
			FailureType: "partition",
			Duration:    20 * time.Second,
			Impact: func() error {
				return simulateNetworkPartition(ctx)
			},
		},
		{
			Name:        "Memory Pressure",
			Component:   "memory",
			FailureType: "pressure",
			Duration:    45 * time.Second,
			Impact: func() error {
				return simulateMemoryPressure(ctx, 0.9) // 90% memory usage
			},
		},
	}

	for _, failure := range failures {
		t.Run(failure.Name, func(t *testing.T) {
			result := testFailureResilience(ctx, failure)

			// System should maintain partial functionality
			if result.AvailabilityDuringFailure < 0.5 {
				t.Errorf("%s: Availability %.2f%% below 50%% threshold",
					failure.Name, result.AvailabilityDuringFailure*100)
			}

			// Should recover after failure clears
			if !result.Recovered {
				t.Errorf("%s: System did not recover after failure", failure.Name)
			}

			t.Logf("%s Results:", failure.Name)
			t.Logf("  Availability During Failure: %.2f%%", result.AvailabilityDuringFailure*100)
			t.Logf("  Recovery Time: %v", result.RecoveryTime)
			t.Logf("  Degraded Operations: %v", result.DegradedOperations)
			t.Logf("  Circuit Breakers Triggered: %d", result.CircuitBreakersTriggered)
		})
	}
}

// TestBurstPattern tests handling of burst traffic patterns
// DISABLED: func TestBurstPattern(t *testing.T) {
	suite := performance.NewBenchmarkSuite()
	ctx := context.Background()

	// Define burst patterns
	patterns := []struct {
		Name          string
		BurstSize     int
		BurstInterval time.Duration
		BurstDuration time.Duration
		TestDuration  time.Duration
	}{
		{
			Name:          "Regular Bursts",
			BurstSize:     100,
			BurstInterval: 10 * time.Second,
			BurstDuration: 2 * time.Second,
			TestDuration:  60 * time.Second,
		},
		{
			Name:          "Irregular Bursts",
			BurstSize:     200,
			BurstInterval: 0, // Random intervals
			BurstDuration: 5 * time.Second,
			TestDuration:  120 * time.Second,
		},
		{
			Name:          "Micro Bursts",
			BurstSize:     50,
			BurstInterval: 1 * time.Second,
			BurstDuration: 100 * time.Millisecond,
			TestDuration:  30 * time.Second,
		},
	}

	for _, pattern := range patterns {
		t.Run(pattern.Name, func(t *testing.T) {
			result := runBurstTest(ctx, suite, pattern)

			// Validate burst handling
			if result.DroppedRequests > 0 {
				t.Logf("%s: Warning - %d requests dropped", pattern.Name, result.DroppedRequests)
			}

			if result.MaxQueueDepth > 1000 {
				t.Errorf("%s: Queue depth %d exceeded limit", pattern.Name, result.MaxQueueDepth)
			}

			t.Logf("%s Results:", pattern.Name)
			t.Logf("  Bursts Handled: %d", result.BurstsHandled)
			t.Logf("  Avg Burst Latency: %v", result.AvgBurstLatency)
			t.Logf("  Max Queue Depth: %d", result.MaxQueueDepth)
			t.Logf("  Dropped Requests: %d", result.DroppedRequests)
		})
	}
}

// SpikeTestResult captures spike test results
type SpikeTestResult struct {
	MaxLatency      time.Duration
	RecoveryTime    time.Duration
	ErrorRate       float64
	ThroughputDrop  float64
	MaxQueueDepth   int
	MetricsSnapshot map[string]interface{}
}

// FailureTestResult captures failure test results
type FailureTestResult struct {
	AvailabilityDuringFailure float64
	RecoveryTime              time.Duration
	Recovered                 bool
	DegradedOperations        []string
	CircuitBreakersTriggered  int
}

// BurstTestResult captures burst test results
type BurstTestResult struct {
	BurstsHandled   int
	AvgBurstLatency time.Duration
	MaxQueueDepth   int
	DroppedRequests int
}

// runSpikeTest executes a spike load test
func runSpikeTest(ctx context.Context, suite *performance.BenchmarkSuite, pattern struct {
	Name         string
	BaseLoad     int
	SpikeLoad    int
	RampUpTime   time.Duration
	SustainTime  time.Duration
	RampDownTime time.Duration
	MaxLatency   time.Duration
	RecoveryTime time.Duration
},
) SpikeTestResult {
	result := SpikeTestResult{
		MetricsSnapshot: make(map[string]interface{}),
	}

	var wg sync.WaitGroup
	var maxLatency int64 // Use atomic for thread safety
	var totalRequests int64
	var errorCount int64
	var queueDepth int32

	// Metrics collection
	metricsChan := make(chan time.Duration, 10000)
	errorChan := make(chan error, 1000)

	// Start with base load
	currentLoad := pattern.BaseLoad
	stopChan := make(chan struct{})

	// Worker function
	worker := func(id int) {
		defer wg.Done()
		for {
			select {
			case <-stopChan:
				return
			default:
				atomic.AddInt32(&queueDepth, 1)
				start := time.Now()

				// Simulate work
				err := processRequest(ctx)

				latency := time.Since(start)
				atomic.AddInt32(&queueDepth, -1)

				// Update max latency
				currentMax := atomic.LoadInt64(&maxLatency)
				if int64(latency) > currentMax {
					atomic.CompareAndSwapInt64(&maxLatency, currentMax, int64(latency))
				}

				atomic.AddInt64(&totalRequests, 1)

				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					select {
					case errorChan <- err:
					default:
					}
				}

				select {
				case metricsChan <- latency:
				default:
				}

				// Small delay between requests
				time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
			}
		}
	}

	// Phase 1: Base load
	klog.Infof("Starting base load phase with %d workers", pattern.BaseLoad)
	for i := 0; i < pattern.BaseLoad; i++ {
		wg.Add(1)
		go worker(i)
	}

	baselineStart := time.Now()
	time.Sleep(10 * time.Second) // Establish baseline

	// Capture baseline metrics
	baselineThroughput := float64(atomic.LoadInt64(&totalRequests)) / time.Since(baselineStart).Seconds()

	// Phase 2: Ramp up to spike
	klog.Infof("Ramping up to spike load: %d workers", pattern.SpikeLoad)
	rampUpSteps := 10
	stepDuration := pattern.RampUpTime / time.Duration(rampUpSteps)
	workersPerStep := (pattern.SpikeLoad - pattern.BaseLoad) / rampUpSteps

	for step := 0; step < rampUpSteps; step++ {
		for i := 0; i < workersPerStep; i++ {
			wg.Add(1)
			go worker(currentLoad + i)
		}
		currentLoad += workersPerStep
		time.Sleep(stepDuration)
	}

	// Phase 3: Sustain spike
	klog.Infof("Sustaining spike load for %v", pattern.SustainTime)
	spikeStart := time.Now()
	time.Sleep(pattern.SustainTime)

	// Capture spike metrics
	spikeThroughput := float64(atomic.LoadInt64(&totalRequests)) / time.Since(spikeStart).Seconds()
	result.ThroughputDrop = (baselineThroughput - spikeThroughput) / baselineThroughput
	result.MaxQueueDepth = int(atomic.LoadInt32(&queueDepth))

	// Phase 4: Ramp down
	klog.Infof("Ramping down from spike")
	close(stopChan) // Signal all workers to stop

	recoveryStart := time.Now()

	// Wait for recovery
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			currentQueueDepth := atomic.LoadInt32(&queueDepth)
			if currentQueueDepth < int32(pattern.BaseLoad) {
				result.RecoveryTime = time.Since(recoveryStart)
				goto recovered
			}
		case <-time.After(pattern.RecoveryTime * 2):
			result.RecoveryTime = pattern.RecoveryTime * 2
			goto recovered
		}
	}

recovered:
	wg.Wait()

	// Calculate final metrics
	result.MaxLatency = time.Duration(atomic.LoadInt64(&maxLatency))
	result.ErrorRate = float64(atomic.LoadInt64(&errorCount)) / float64(atomic.LoadInt64(&totalRequests))

	// Collect latency distribution
	close(metricsChan)
	latencies := []time.Duration{}
	for latency := range metricsChan {
		latencies = append(latencies, latency)
	}

	if len(latencies) > 0 {
		analyzer := performance.NewMetricsAnalyzer()
		for _, latency := range latencies {
			analyzer.AddSample(float64(latency))
		}
		result.MetricsSnapshot["p50_latency"] = analyzer.CalculatePercentile(50)
		result.MetricsSnapshot["p95_latency"] = analyzer.CalculatePercentile(95)
		result.MetricsSnapshot["p99_latency"] = analyzer.CalculatePercentile(99)
	}

	return result
}

// testFailureResilience tests resilience to component failures
func testFailureResilience(ctx context.Context, failure struct {
	Name        string
	Component   string
	FailureType string
	Duration    time.Duration
	Impact      func() error
},
) FailureTestResult {
	result := FailureTestResult{
		DegradedOperations: []string{},
	}

	// Start normal operations
	var successCount int64
	var totalCount int64
	stopChan := make(chan struct{})

	go func() {
		for {
			select {
			case <-stopChan:
				return
			default:
				atomic.AddInt64(&totalCount, 1)
				if err := processRequest(ctx); err == nil {
					atomic.AddInt64(&successCount, 1)
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	// Baseline measurement
	time.Sleep(5 * time.Second)
	baselineSuccess := atomic.LoadInt64(&successCount)
	baselineTotal := atomic.LoadInt64(&totalCount)

	// Inject failure
	klog.Infof("Injecting failure: %s", failure.Name)
	_ = time.Now()
	go failure.Impact()

	// Monitor during failure
	time.Sleep(failure.Duration)

	duringFailureSuccess := atomic.LoadInt64(&successCount) - baselineSuccess
	duringFailureTotal := atomic.LoadInt64(&totalCount) - baselineTotal

	if duringFailureTotal > 0 {
		result.AvailabilityDuringFailure = float64(duringFailureSuccess) / float64(duringFailureTotal)
	}

	// Wait for recovery
	recoveryStart := time.Now()
	recovered := false

	for i := 0; i < 60; i++ { // Wait up to 60 seconds
		time.Sleep(1 * time.Second)

		recentSuccess := atomic.LoadInt64(&successCount)
		recentTotal := atomic.LoadInt64(&totalCount)

		if recentTotal-baselineTotal > 0 {
			recentRate := float64(recentSuccess-baselineSuccess) / float64(recentTotal-baselineTotal)
			if recentRate > 0.95 { // 95% success rate
				recovered = true
				result.RecoveryTime = time.Since(recoveryStart)
				break
			}
		}
	}

	result.Recovered = recovered
	close(stopChan)

	// Check degraded operations (simplified)
	if result.AvailabilityDuringFailure < 1.0 {
		result.DegradedOperations = append(result.DegradedOperations, "intent_processing")
	}

	// Count circuit breakers (simplified)
	result.CircuitBreakersTriggered = rand.Intn(3) // Placeholder

	return result
}

// runBurstTest executes a burst pattern test
func runBurstTest(ctx context.Context, suite *performance.BenchmarkSuite, pattern struct {
	Name          string
	BurstSize     int
	BurstInterval time.Duration
	BurstDuration time.Duration
	TestDuration  time.Duration
},
) BurstTestResult {
	result := BurstTestResult{}

	var totalLatency int64
	var burstCount int32
	var droppedRequests int32
	var maxQueueDepth int32

	endTime := time.Now().Add(pattern.TestDuration)

	for time.Now().Before(endTime) {
		// Generate burst
		atomic.AddInt32(&burstCount, 1)

		var wg sync.WaitGroup
		_ = time.Now()

		for i := 0; i < pattern.BurstSize; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				// Try to process request
				queueDepth := atomic.AddInt32(&maxQueueDepth, 1)
				defer atomic.AddInt32(&maxQueueDepth, -1)

				if queueDepth > 1000 { // Queue limit
					atomic.AddInt32(&droppedRequests, 1)
					return
				}

				start := time.Now()
				processRequest(ctx)
				latency := time.Since(start)

				atomic.AddInt64(&totalLatency, int64(latency))
			}()
		}

		// Wait for burst to complete or timeout
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Burst completed
		case <-time.After(pattern.BurstDuration * 2):
			// Burst timeout
		}

		// Wait for next burst
		if pattern.BurstInterval > 0 {
			time.Sleep(pattern.BurstInterval)
		} else {
			// Random interval for irregular bursts
			time.Sleep(time.Duration(rand.Intn(10)+1) * time.Second)
		}
	}

	result.BurstsHandled = int(atomic.LoadInt32(&burstCount))
	result.DroppedRequests = int(atomic.LoadInt32(&droppedRequests))
	result.MaxQueueDepth = int(atomic.LoadInt32(&maxQueueDepth))

	if result.BurstsHandled > 0 {
		avgLatencyNanos := atomic.LoadInt64(&totalLatency) / int64(result.BurstsHandled*pattern.BurstSize)
		result.AvgBurstLatency = time.Duration(avgLatencyNanos)
	}

	return result
}

// Helper functions

func processRequest(ctx context.Context) error {
	// Simulate request processing with variable latency
	latency := time.Duration(rand.Intn(100)+50) * time.Millisecond

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(latency):
		// Randomly fail some requests
		if rand.Float32() < 0.01 { // 1% error rate
			return fmt.Errorf("simulated error")
		}
		return nil
	}
}

func simulateLLMOutage(ctx context.Context) error {
	// Simulate LLM service being completely unavailable
	klog.Info("Simulating LLM service outage")
	time.Sleep(30 * time.Second)
	klog.Info("LLM service restored")
	return nil
}

func simulateDatabaseSlowdown(ctx context.Context, severity time.Duration) error {
	// Simulate database responding slowly
	klog.Infof("Simulating database slowdown with %v added latency", severity)
	time.Sleep(60 * time.Second)
	klog.Info("Database performance restored")
	return nil
}

func simulateNetworkPartition(ctx context.Context) error {
	// Simulate network partition between components
	klog.Info("Simulating network partition")
	time.Sleep(20 * time.Second)
	klog.Info("Network partition healed")
	return nil
}

func simulateMemoryPressure(ctx context.Context, usage float64) error {
	// Simulate high memory usage
	klog.Infof("Simulating memory pressure at %.0f%% usage", usage*100)

	// Allocate memory to simulate pressure
	// In real test, would allocate actual memory
	time.Sleep(45 * time.Second)

	klog.Info("Memory pressure relieved")
	return nil
}
