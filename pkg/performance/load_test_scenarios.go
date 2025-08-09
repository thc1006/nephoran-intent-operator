package performance

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"k8s.io/klog/v2"
)

// LoadTestScenario represents a load testing scenario
type LoadTestScenario struct {
	Name            string
	Description     string
	Duration        time.Duration
	RampUpTime      time.Duration
	MaxConcurrency  int
	TargetRPS       int
	WorkloadPattern WorkloadPattern
	Workload        func(ctx context.Context) error
	Metrics         *LoadTestMetrics
}

// WorkloadPattern defines the pattern of load generation
type WorkloadPattern string

const (
	PatternConstant   WorkloadPattern = "constant"
	PatternLinearRamp WorkloadPattern = "linear-ramp"
	PatternStepRamp   WorkloadPattern = "step-ramp"
	PatternSpike      WorkloadPattern = "spike"
	PatternWave       WorkloadPattern = "wave"
	PatternRealistic  WorkloadPattern = "realistic"
)

// LoadTestMetrics tracks load test metrics
type LoadTestMetrics struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	TotalLatency       int64 // in nanoseconds
	MinLatency         int64
	MaxLatency         int64
	Latencies          []time.Duration
	ErrorTypes         map[string]int64
	StartTime          time.Time
	EndTime            time.Time
	mu                 sync.RWMutex
}

// LoadTestRunner executes load test scenarios
type LoadTestRunner struct {
	scenarios       []LoadTestScenario
	profiler        *Profiler
	flameGraphGen   *FlameGraphGenerator
	metricsReporter *MetricsReporter
	results         map[string]*LoadTestResult
	mu              sync.RWMutex
}

// LoadTestResult contains the results of a load test
type LoadTestResult struct {
	Scenario       LoadTestScenario
	Metrics        *LoadTestMetrics
	ProfileData    map[string]*ProfileData
	FlameGraphs    map[string]string
	Analysis       *PerformanceAnalysis
	Passed         bool
	FailureReasons []string
}

// PerformanceAnalysis contains analysis of performance data
type PerformanceAnalysis struct {
	AverageLatency  time.Duration
	P50Latency      time.Duration
	P95Latency      time.Duration
	P99Latency      time.Duration
	Throughput      float64
	ErrorRate       float64
	CPUUtilization  float64
	MemoryUsage     float64
	GoroutineCount  int
	Bottlenecks     []Bottleneck
	Recommendations []string
}

// Bottleneck represents a performance bottleneck
type Bottleneck struct {
	Component   string
	Type        string // "CPU", "Memory", "I/O", "Lock Contention"
	Severity    string // "Critical", "High", "Medium", "Low"
	Impact      float64
	Description string
	Solution    string
}

// MetricsReporter reports metrics during load tests
type MetricsReporter struct {
	interval time.Duration
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewLoadTestRunner creates a new load test runner
func NewLoadTestRunner() *LoadTestRunner {
	return &LoadTestRunner{
		scenarios:       make([]LoadTestScenario, 0),
		profiler:        NewProfiler(),
		flameGraphGen:   NewFlameGraphGenerator("/tmp/profiles", "/tmp/flamegraphs"),
		metricsReporter: NewMetricsReporter(5 * time.Second),
		results:         make(map[string]*LoadTestResult),
	}
}

// NewMetricsReporter creates a new metrics reporter
func NewMetricsReporter(interval time.Duration) *MetricsReporter {
	return &MetricsReporter{
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// AddScenario adds a load test scenario
func (r *LoadTestRunner) AddScenario(scenario LoadTestScenario) {
	r.scenarios = append(r.scenarios, scenario)
}

// RunScenario executes a specific load test scenario
func (r *LoadTestRunner) RunScenario(ctx context.Context, scenario LoadTestScenario) (*LoadTestResult, error) {
	klog.Infof("Starting load test scenario: %s", scenario.Name)

	result := &LoadTestResult{
		Scenario:    scenario,
		Metrics:     NewLoadTestMetrics(),
		ProfileData: make(map[string]*ProfileData),
		FlameGraphs: make(map[string]string),
	}

	// Start profiling
	if err := r.profiler.StartCPUProfile(); err != nil {
		klog.Errorf("Failed to start CPU profiling: %v", err)
	}
	defer r.profiler.StopCPUProfile()

	// Start metrics reporting
	r.metricsReporter.Start(result.Metrics)
	defer r.metricsReporter.Stop()

	// Execute the load test
	result.Metrics.StartTime = time.Now()

	switch scenario.WorkloadPattern {
	case PatternConstant:
		r.runConstantLoad(ctx, scenario, result.Metrics)
	case PatternLinearRamp:
		r.runLinearRampLoad(ctx, scenario, result.Metrics)
	case PatternStepRamp:
		r.runStepRampLoad(ctx, scenario, result.Metrics)
	case PatternSpike:
		r.runSpikeLoad(ctx, scenario, result.Metrics)
	case PatternWave:
		r.runWaveLoad(ctx, scenario, result.Metrics)
	case PatternRealistic:
		r.runRealisticLoad(ctx, scenario, result.Metrics)
	default:
		r.runConstantLoad(ctx, scenario, result.Metrics)
	}

	result.Metrics.EndTime = time.Now()

	// Capture final profiles
	cpuProfile, _ := r.profiler.StopCPUProfile()
	memProfile, _ := r.profiler.CaptureMemoryProfile()
	goroutineProfile, _ := r.profiler.CaptureGoroutineProfile()

	result.ProfileData["cpu"] = &ProfileData{ProfilePath: cpuProfile}
	result.ProfileData["memory"] = &ProfileData{ProfilePath: memProfile}
	result.ProfileData["goroutine"] = &ProfileData{ProfilePath: goroutineProfile}

	// Generate flame graphs
	for profileType, data := range result.ProfileData {
		if flamegraph, err := r.flameGraphGen.generateFlameGraph(data.ProfilePath, fmt.Sprintf("%s_%s", scenario.Name, profileType)); err == nil {
			result.FlameGraphs[profileType] = flamegraph
		}
	}

	// Analyze results
	result.Analysis = r.analyzeResults(result.Metrics)

	// Determine pass/fail
	result.Passed, result.FailureReasons = r.evaluateResults(scenario, result.Analysis)

	// Store result
	r.mu.Lock()
	r.results[scenario.Name] = result
	r.mu.Unlock()

	klog.Infof("Load test scenario %s completed. Passed: %v", scenario.Name, result.Passed)

	return result, nil
}

// runConstantLoad runs a constant load pattern
func (r *LoadTestRunner) runConstantLoad(ctx context.Context, scenario LoadTestScenario, metrics *LoadTestMetrics) {
	var wg sync.WaitGroup
	stopCh := make(chan struct{})

	// Calculate delay between requests to achieve target RPS
	delay := time.Second / time.Duration(scenario.TargetRPS)

	// Start workers
	for i := 0; i < scenario.MaxConcurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			ticker := time.NewTicker(delay * time.Duration(scenario.MaxConcurrency))
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-stopCh:
					return
				case <-ticker.C:
					r.executeRequest(ctx, scenario, metrics)
				}
			}
		}(i)
	}

	// Run for specified duration
	time.Sleep(scenario.Duration)
	close(stopCh)
	wg.Wait()
}

// runLinearRampLoad runs a linear ramp-up load pattern
func (r *LoadTestRunner) runLinearRampLoad(ctx context.Context, scenario LoadTestScenario, metrics *LoadTestMetrics) {
	var wg sync.WaitGroup
	stopCh := make(chan struct{})

	// Calculate ramp-up rate
	rampUpSteps := 10
	stepDuration := scenario.RampUpTime / time.Duration(rampUpSteps)
	workersPerStep := scenario.MaxConcurrency / rampUpSteps

	currentWorkers := 0

	for step := 0; step < rampUpSteps; step++ {
		// Add workers for this step
		for i := 0; i < workersPerStep; i++ {
			currentWorkers++
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				delay := time.Second / time.Duration(scenario.TargetRPS/currentWorkers)
				ticker := time.NewTicker(delay)
				defer ticker.Stop()

				for {
					select {
					case <-ctx.Done():
						return
					case <-stopCh:
						return
					case <-ticker.C:
						r.executeRequest(ctx, scenario, metrics)
					}
				}
			}(currentWorkers)
		}

		time.Sleep(stepDuration)
	}

	// Run at full load for remaining duration
	time.Sleep(scenario.Duration - scenario.RampUpTime)
	close(stopCh)
	wg.Wait()
}

// runStepRampLoad runs a step ramp-up load pattern
func (r *LoadTestRunner) runStepRampLoad(ctx context.Context, scenario LoadTestScenario, metrics *LoadTestMetrics) {
	steps := []int{25, 50, 75, 100} // Percentage of max load
	stepDuration := scenario.Duration / time.Duration(len(steps))

	for _, percentage := range steps {
		workers := (scenario.MaxConcurrency * percentage) / 100
		targetRPS := (scenario.TargetRPS * percentage) / 100

		var wg sync.WaitGroup
		stopCh := make(chan struct{})

		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				delay := time.Second / time.Duration(targetRPS/workers)
				ticker := time.NewTicker(delay)
				defer ticker.Stop()

				for {
					select {
					case <-ctx.Done():
						return
					case <-stopCh:
						return
					case <-ticker.C:
						r.executeRequest(ctx, scenario, metrics)
					}
				}
			}()
		}

		time.Sleep(stepDuration)
		close(stopCh)
		wg.Wait()
	}
}

// runSpikeLoad runs a spike load pattern
func (r *LoadTestRunner) runSpikeLoad(ctx context.Context, scenario LoadTestScenario, metrics *LoadTestMetrics) {
	normalLoad := scenario.MaxConcurrency / 4
	spikeLoad := scenario.MaxConcurrency

	phases := []struct {
		workers  int
		duration time.Duration
	}{
		{normalLoad, scenario.Duration / 4}, // Normal
		{spikeLoad, scenario.Duration / 4},  // Spike
		{normalLoad, scenario.Duration / 4}, // Recovery
		{spikeLoad, scenario.Duration / 4},  // Second spike
	}

	for _, phase := range phases {
		var wg sync.WaitGroup
		stopCh := make(chan struct{})

		for i := 0; i < phase.workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				delay := time.Second / time.Duration(scenario.TargetRPS/phase.workers)
				ticker := time.NewTicker(delay)
				defer ticker.Stop()

				for {
					select {
					case <-ctx.Done():
						return
					case <-stopCh:
						return
					case <-ticker.C:
						r.executeRequest(ctx, scenario, metrics)
					}
				}
			}()
		}

		time.Sleep(phase.duration)
		close(stopCh)
		wg.Wait()
	}
}

// runWaveLoad runs a wave (sinusoidal) load pattern
func (r *LoadTestRunner) runWaveLoad(ctx context.Context, scenario LoadTestScenario, metrics *LoadTestMetrics) {
	var wg sync.WaitGroup
	stopCh := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()

		startTime := time.Now()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-stopCh:
				return
			case <-ticker.C:
				elapsed := time.Since(startTime).Seconds()
				// Sinusoidal pattern
				wavePosition := elapsed * 2 * 3.14159 / 60 // Complete wave every 60 seconds
				loadFactor := (1 + float64(scenario.MaxConcurrency)*0.5*math.Sin(wavePosition)) / 2
				currentWorkers := int(float64(scenario.MaxConcurrency) * loadFactor)

				// Adjust worker pool size
				for i := 0; i < currentWorkers; i++ {
					go func() {
						r.executeRequest(ctx, scenario, metrics)
					}()
				}
			}
		}
	}()

	time.Sleep(scenario.Duration)
	close(stopCh)
	wg.Wait()
}

// runRealisticLoad runs a realistic telecom workload pattern
func (r *LoadTestRunner) runRealisticLoad(ctx context.Context, scenario LoadTestScenario, metrics *LoadTestMetrics) {
	// Simulate realistic patterns:
	// - Morning ramp-up
	// - Lunch time spike
	// - Afternoon steady state
	// - Evening peak
	// - Night time low load

	hourlyPatterns := []struct {
		hour     int
		loadPct  int
		duration time.Duration
	}{
		{6, 20, scenario.Duration / 8},   // Early morning
		{8, 60, scenario.Duration / 8},   // Morning ramp
		{12, 80, scenario.Duration / 8},  // Lunch spike
		{14, 70, scenario.Duration / 8},  // Afternoon
		{18, 100, scenario.Duration / 8}, // Evening peak
		{20, 90, scenario.Duration / 8},  // Early evening
		{22, 50, scenario.Duration / 8},  // Late evening
		{0, 30, scenario.Duration / 8},   // Night
	}

	for _, pattern := range hourlyPatterns {
		workers := (scenario.MaxConcurrency * pattern.loadPct) / 100

		var wg sync.WaitGroup
		stopCh := make(chan struct{})

		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				// Add some randomness to simulate real traffic
				jitter := time.Duration(rand.Intn(100)) * time.Millisecond
				delay := time.Second/time.Duration(scenario.TargetRPS/workers) + jitter
				ticker := time.NewTicker(delay)
				defer ticker.Stop()

				for {
					select {
					case <-ctx.Done():
						return
					case <-stopCh:
						return
					case <-ticker.C:
						r.executeRequest(ctx, scenario, metrics)
					}
				}
			}()
		}

		time.Sleep(pattern.duration)
		close(stopCh)
		wg.Wait()
	}
}

// executeRequest executes a single request and records metrics
func (r *LoadTestRunner) executeRequest(ctx context.Context, scenario LoadTestScenario, metrics *LoadTestMetrics) {
	start := time.Now()

	err := scenario.Workload(ctx)

	latency := time.Since(start)

	// Update metrics
	atomic.AddInt64(&metrics.TotalRequests, 1)

	if err != nil {
		atomic.AddInt64(&metrics.FailedRequests, 1)

		metrics.mu.Lock()
		errorType := fmt.Sprintf("%T", err)
		if metrics.ErrorTypes == nil {
			metrics.ErrorTypes = make(map[string]int64)
		}
		metrics.ErrorTypes[errorType]++
		metrics.mu.Unlock()
	} else {
		atomic.AddInt64(&metrics.SuccessfulRequests, 1)
	}

	// Update latency metrics
	latencyNs := latency.Nanoseconds()
	atomic.AddInt64(&metrics.TotalLatency, latencyNs)

	// Update min/max latency
	for {
		oldMin := atomic.LoadInt64(&metrics.MinLatency)
		if oldMin == 0 || latencyNs < oldMin {
			if atomic.CompareAndSwapInt64(&metrics.MinLatency, oldMin, latencyNs) {
				break
			}
		} else {
			break
		}
	}

	for {
		oldMax := atomic.LoadInt64(&metrics.MaxLatency)
		if latencyNs > oldMax {
			if atomic.CompareAndSwapInt64(&metrics.MaxLatency, oldMax, latencyNs) {
				break
			}
		} else {
			break
		}
	}

	// Store latency for percentile calculation
	metrics.mu.Lock()
	metrics.Latencies = append(metrics.Latencies, latency)
	metrics.mu.Unlock()
}

// analyzeResults analyzes the load test results
func (r *LoadTestRunner) analyzeResults(metrics *LoadTestMetrics) *PerformanceAnalysis {
	analysis := &PerformanceAnalysis{
		Bottlenecks:     make([]Bottleneck, 0),
		Recommendations: make([]string, 0),
	}

	// Calculate basic metrics
	totalRequests := atomic.LoadInt64(&metrics.TotalRequests)
	successfulRequests := atomic.LoadInt64(&metrics.SuccessfulRequests)
	failedRequests := atomic.LoadInt64(&metrics.FailedRequests)

	if totalRequests > 0 {
		analysis.ErrorRate = float64(failedRequests) / float64(totalRequests) * 100
		analysis.AverageLatency = time.Duration(atomic.LoadInt64(&metrics.TotalLatency) / totalRequests)
	}

	// Calculate percentiles
	metrics.mu.RLock()
	if len(metrics.Latencies) > 0 {
		latencies := make([]time.Duration, len(metrics.Latencies))
		copy(latencies, metrics.Latencies)
		metrics.mu.RUnlock()

		analyzer := NewMetricsAnalyzer()
		analysis.P50Latency = analyzer.CalculatePercentile(latencies, 50)
		analysis.P95Latency = analyzer.CalculatePercentile(latencies, 95)
		analysis.P99Latency = analyzer.CalculatePercentile(latencies, 99)
	} else {
		metrics.mu.RUnlock()
	}

	// Calculate throughput
	duration := metrics.EndTime.Sub(metrics.StartTime).Seconds()
	if duration > 0 {
		analysis.Throughput = float64(successfulRequests) / duration
	}

	// Identify bottlenecks
	if analysis.P99Latency > 30*time.Second {
		analysis.Bottlenecks = append(analysis.Bottlenecks, Bottleneck{
			Component:   "Intent Processing",
			Type:        "Latency",
			Severity:    "Critical",
			Impact:      float64(analysis.P99Latency.Seconds()),
			Description: fmt.Sprintf("P99 latency exceeds 30s threshold: %.2fs", analysis.P99Latency.Seconds()),
			Solution:    "Enable response caching, optimize LLM queries, implement request batching",
		})
	}

	if analysis.ErrorRate > 5 {
		analysis.Bottlenecks = append(analysis.Bottlenecks, Bottleneck{
			Component:   "System Reliability",
			Type:        "Errors",
			Severity:    "High",
			Impact:      analysis.ErrorRate,
			Description: fmt.Sprintf("Error rate exceeds 5%% threshold: %.2f%%", analysis.ErrorRate),
			Solution:    "Implement circuit breakers, add retry logic with exponential backoff",
		})
	}

	// Generate recommendations
	if analysis.P99Latency > 20*time.Second {
		analysis.Recommendations = append(analysis.Recommendations,
			"Implement multi-level caching for LLM responses")
	}

	if analysis.Throughput < 10 {
		analysis.Recommendations = append(analysis.Recommendations,
			"Increase worker pool size and optimize request batching")
	}

	if len(metrics.ErrorTypes) > 3 {
		analysis.Recommendations = append(analysis.Recommendations,
			"Implement comprehensive error handling and recovery mechanisms")
	}

	return analysis
}

// evaluateResults evaluates if the test passed based on criteria
func (r *LoadTestRunner) evaluateResults(scenario LoadTestScenario, analysis *PerformanceAnalysis) (bool, []string) {
	passed := true
	reasons := make([]string, 0)

	// Check latency requirements
	if analysis.P99Latency > 30*time.Second {
		passed = false
		reasons = append(reasons, fmt.Sprintf("P99 latency (%.2fs) exceeds 30s threshold", analysis.P99Latency.Seconds()))
	}

	// Check error rate
	if analysis.ErrorRate > 5 {
		passed = false
		reasons = append(reasons, fmt.Sprintf("Error rate (%.2f%%) exceeds 5%% threshold", analysis.ErrorRate))
	}

	// Check throughput
	expectedThroughput := float64(scenario.TargetRPS) * 0.8 // Allow 20% deviation
	if analysis.Throughput < expectedThroughput {
		passed = false
		reasons = append(reasons, fmt.Sprintf("Throughput (%.2f rps) below expected %.2f rps", analysis.Throughput, expectedThroughput))
	}

	// Check for critical bottlenecks
	for _, bottleneck := range analysis.Bottlenecks {
		if bottleneck.Severity == "Critical" {
			passed = false
			reasons = append(reasons, fmt.Sprintf("Critical bottleneck: %s", bottleneck.Description))
		}
	}

	return passed, reasons
}

// NewLoadTestMetrics creates new load test metrics
func NewLoadTestMetrics() *LoadTestMetrics {
	return &LoadTestMetrics{
		Latencies:  make([]time.Duration, 0),
		ErrorTypes: make(map[string]int64),
	}
}

// Start starts the metrics reporter
func (mr *MetricsReporter) Start(metrics *LoadTestMetrics) {
	mr.wg.Add(1)
	go func() {
		defer mr.wg.Done()
		ticker := time.NewTicker(mr.interval)
		defer ticker.Stop()

		for {
			select {
			case <-mr.stopCh:
				return
			case <-ticker.C:
				mr.reportMetrics(metrics)
			}
		}
	}()
}

// Stop stops the metrics reporter
func (mr *MetricsReporter) Stop() {
	close(mr.stopCh)
	mr.wg.Wait()
}

// reportMetrics reports current metrics
func (mr *MetricsReporter) reportMetrics(metrics *LoadTestMetrics) {
	totalRequests := atomic.LoadInt64(&metrics.TotalRequests)
	successfulRequests := atomic.LoadInt64(&metrics.SuccessfulRequests)
	failedRequests := atomic.LoadInt64(&metrics.FailedRequests)

	var avgLatency time.Duration
	if totalRequests > 0 {
		avgLatency = time.Duration(atomic.LoadInt64(&metrics.TotalLatency) / totalRequests)
	}

	klog.Infof("Load Test Progress - Total: %d, Success: %d, Failed: %d, Avg Latency: %v",
		totalRequests, successfulRequests, failedRequests, avgLatency)
}

// GenerateReport generates a comprehensive load test report
func (r *LoadTestRunner) GenerateReport() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var report strings.Builder

	report.WriteString("=== NEPHORAN INTENT OPERATOR LOAD TEST REPORT ===\n\n")
	report.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC3339)))

	for name, result := range r.results {
		report.WriteString(fmt.Sprintf("Scenario: %s\n", name))
		report.WriteString(fmt.Sprintf("Status: %s\n", getPassFailStatus(result.Passed)))
		report.WriteString(fmt.Sprintf("Duration: %v\n", result.Metrics.EndTime.Sub(result.Metrics.StartTime)))
		report.WriteString(fmt.Sprintf("Pattern: %s\n", result.Scenario.WorkloadPattern))
		report.WriteString("\n")

		// Metrics summary
		report.WriteString("Performance Metrics:\n")
		report.WriteString(fmt.Sprintf("  Total Requests: %d\n", result.Metrics.TotalRequests))
		report.WriteString(fmt.Sprintf("  Successful: %d (%.2f%%)\n",
			result.Metrics.SuccessfulRequests,
			float64(result.Metrics.SuccessfulRequests)/float64(result.Metrics.TotalRequests)*100))
		report.WriteString(fmt.Sprintf("  Failed: %d (%.2f%%)\n",
			result.Metrics.FailedRequests,
			result.Analysis.ErrorRate))
		report.WriteString(fmt.Sprintf("  Throughput: %.2f req/s\n", result.Analysis.Throughput))
		report.WriteString("\n")

		// Latency metrics
		report.WriteString("Latency Analysis:\n")
		report.WriteString(fmt.Sprintf("  Average: %v\n", result.Analysis.AverageLatency))
		report.WriteString(fmt.Sprintf("  P50: %v\n", result.Analysis.P50Latency))
		report.WriteString(fmt.Sprintf("  P95: %v\n", result.Analysis.P95Latency))
		report.WriteString(fmt.Sprintf("  P99: %v\n", result.Analysis.P99Latency))
		report.WriteString(fmt.Sprintf("  Min: %v\n", time.Duration(result.Metrics.MinLatency)))
		report.WriteString(fmt.Sprintf("  Max: %v\n", time.Duration(result.Metrics.MaxLatency)))
		report.WriteString("\n")

		// Bottlenecks
		if len(result.Analysis.Bottlenecks) > 0 {
			report.WriteString("Identified Bottlenecks:\n")
			for _, bottleneck := range result.Analysis.Bottlenecks {
				report.WriteString(fmt.Sprintf("  - [%s] %s: %s\n",
					bottleneck.Severity, bottleneck.Component, bottleneck.Description))
				report.WriteString(fmt.Sprintf("    Solution: %s\n", bottleneck.Solution))
			}
			report.WriteString("\n")
		}

		// Recommendations
		if len(result.Analysis.Recommendations) > 0 {
			report.WriteString("Recommendations:\n")
			for _, rec := range result.Analysis.Recommendations {
				report.WriteString(fmt.Sprintf("  - %s\n", rec))
			}
			report.WriteString("\n")
		}

		// Failure reasons
		if !result.Passed && len(result.FailureReasons) > 0 {
			report.WriteString("Failure Reasons:\n")
			for _, reason := range result.FailureReasons {
				report.WriteString(fmt.Sprintf("  - %s\n", reason))
			}
			report.WriteString("\n")
		}

		// Flame graphs
		if len(result.FlameGraphs) > 0 {
			report.WriteString("Generated Flame Graphs:\n")
			for profileType, path := range result.FlameGraphs {
				report.WriteString(fmt.Sprintf("  - %s: %s\n", profileType, path))
			}
			report.WriteString("\n")
		}

		report.WriteString(strings.Repeat("-", 60) + "\n\n")
	}

	// Overall summary
	report.WriteString("=== OVERALL SUMMARY ===\n\n")

	passedCount := 0
	for _, result := range r.results {
		if result.Passed {
			passedCount++
		}
	}

	report.WriteString(fmt.Sprintf("Total Scenarios: %d\n", len(r.results)))
	report.WriteString(fmt.Sprintf("Passed: %d\n", passedCount))
	report.WriteString(fmt.Sprintf("Failed: %d\n", len(r.results)-passedCount))
	report.WriteString(fmt.Sprintf("Success Rate: %.2f%%\n", float64(passedCount)/float64(len(r.results))*100))

	return report.String()
}

func getPassFailStatus(passed bool) string {
	if passed {
		return "✓ PASSED"
	}
	return "✗ FAILED"
}
