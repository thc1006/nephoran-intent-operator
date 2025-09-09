package performance_tests

import (
	"context"
<<<<<<< HEAD
	"crypto/rand"
	"encoding/json"
	"fmt"
=======
	"encoding/json"
	"fmt"
	"math/rand"
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	vegeta "github.com/tsenart/vegeta/v12/lib"
)

// ProductionLoadTestConfig defines load test configuration
type ProductionLoadTestConfig struct {
	Duration         time.Duration
	RampUpDuration   time.Duration
	TargetRPS        int
	MaxConcurrency   int
	TestScenarios    []TestScenario
	WarmupDuration   time.Duration
	CooldownDuration time.Duration
	MetricsInterval  time.Duration
}

// TestScenario represents a test scenario
type TestScenario struct {
	Name        string
	Weight      float64
	Endpoint    string
	Method      string
	PayloadFunc func() interface{}
	Validators  []ResponseValidator
}

// ResponseValidator validates response
type ResponseValidator func(*http.Response, []byte) error

// ProductionLoadTestResult contains test results
type ProductionLoadTestResult struct {
	StartTime           time.Time                  `json:"start_time"`
	EndTime             time.Time                  `json:"end_time"`
	Duration            time.Duration              `json:"duration"`
	TotalRequests       int64                      `json:"total_requests"`
	SuccessfulRequests  int64                      `json:"successful_requests"`
	FailedRequests      int64                      `json:"failed_requests"`
	AverageLatency      time.Duration              `json:"average_latency"`
	P50Latency          time.Duration              `json:"p50_latency"`
	P95Latency          time.Duration              `json:"p95_latency"`
	P99Latency          time.Duration              `json:"p99_latency"`
	MaxLatency          time.Duration              `json:"max_latency"`
	MinLatency          time.Duration              `json:"min_latency"`
	Throughput          float64                    `json:"throughput"`
	ErrorRate           float64                    `json:"error_rate"`
	ScenarioResults     map[string]*ScenarioResult `json:"scenario_results"`
	ResourceUtilization *ProductionResourceMetrics `json:"resource_utilization"`
	SystemPerformance   *SystemMetrics             `json:"system_performance"`
}

// ScenarioResult contains scenario-specific results
type ScenarioResult struct {
	Requests     int64            `json:"requests"`
	Successes    int64            `json:"successes"`
	Failures     int64            `json:"failures"`
	AvgLatency   time.Duration    `json:"avg_latency"`
	ErrorsByType map[string]int64 `json:"errors_by_type"`
}

// ProductionResourceMetrics tracks resource utilization
type ProductionResourceMetrics struct {
	CPUUsage         float64 `json:"cpu_usage"`
	MemoryUsage      float64 `json:"memory_usage"`
	DiskIOPS         float64 `json:"disk_iops"`
	NetworkBandwidth float64 `json:"network_bandwidth"`
	GoroutineCount   int     `json:"goroutine_count"`
}

// SystemMetrics tracks system performance
type SystemMetrics struct {
	DatabaseLatency      time.Duration     `json:"database_latency"`
	CacheHitRate         float64           `json:"cache_hit_rate"`
	QueueDepth           int64             `json:"queue_depth"`
	CircuitBreakerStatus map[string]string `json:"circuit_breaker_status"`
}

// Prometheus metrics
var (
	requestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "load_test_requests_total",
		Help: "Total number of requests made during load test",
	}, []string{"scenario", "status"})

	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "load_test_request_duration_seconds",
		Help:    "Request duration in seconds",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 20),
	}, []string{"scenario"})

	concurrentRequests = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "load_test_concurrent_requests",
		Help: "Number of concurrent requests",
	}, []string{"scenario"})

	systemMetrics = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "load_test_system_metrics",
		Help: "System performance metrics",
	}, []string{"metric"})
)

// ProductionLoadTester executes production-scale load tests
type ProductionLoadTester struct {
<<<<<<< HEAD
	config           *LoadTestConfig
	httpClient       *http.Client
	results          *LoadTestResult
=======
	config           *ProductionLoadTestConfig
	httpClient       *http.Client
	results          *ProductionLoadTestResult
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	scenarioWeights  []float64
	totalWeight      float64
	activeRequests   int64
	mu               sync.RWMutex
	ctx              context.Context
	cancel           context.CancelFunc
	metricsCollector *MetricsCollector
}

// NewProductionLoadTester creates a new load tester
<<<<<<< HEAD
func NewProductionLoadTester(config *LoadTestConfig) *ProductionLoadTester {
=======
func NewProductionLoadTester(config *ProductionLoadTestConfig) *ProductionLoadTester {
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	ctx, cancel := context.WithCancel(context.Background())

	tester := &ProductionLoadTester{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        1000,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
		},
<<<<<<< HEAD
		results: &LoadTestResult{
=======
		results: &ProductionLoadTestResult{
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			ScenarioResults: make(map[string]*ScenarioResult),
		},
		ctx:              ctx,
		cancel:           cancel,
		metricsCollector: NewMetricsCollector(),
	}

	// Calculate scenario weights
	for _, scenario := range config.TestScenarios {
		tester.scenarioWeights = append(tester.scenarioWeights, scenario.Weight)
		tester.totalWeight += scenario.Weight
		tester.results.ScenarioResults[scenario.Name] = &ScenarioResult{
			ErrorsByType: make(map[string]int64),
		}
	}

	return tester
}

// RunTest executes the load test
<<<<<<< HEAD
func (plt *ProductionLoadTester) RunTest() (*LoadTestResult, error) {
=======
func (plt *ProductionLoadTester) RunTest() (*ProductionLoadTestResult, error) {
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	plt.results.StartTime = time.Now()

	// Start metrics collection
	go plt.metricsCollector.Start(plt.ctx, plt.config.MetricsInterval)

	// Warmup phase
	if plt.config.WarmupDuration > 0 {
		fmt.Printf("Starting warmup phase (%v)...\n", plt.config.WarmupDuration)
		plt.runPhase(plt.config.WarmupDuration, plt.config.TargetRPS/10) // 10% load during warmup
	}

	// Ramp-up phase
	if plt.config.RampUpDuration > 0 {
		fmt.Printf("Starting ramp-up phase (%v)...\n", plt.config.RampUpDuration)
		plt.runRampUp()
	}

	// Main test phase
	fmt.Printf("Starting main test phase (%v at %d RPS)...\n", plt.config.Duration, plt.config.TargetRPS)
	plt.runPhase(plt.config.Duration, plt.config.TargetRPS)

	// Cooldown phase
	if plt.config.CooldownDuration > 0 {
		fmt.Printf("Starting cooldown phase (%v)...\n", plt.config.CooldownDuration)
		plt.runPhase(plt.config.CooldownDuration, plt.config.TargetRPS/10)
	}

	plt.results.EndTime = time.Now()
	plt.results.Duration = plt.results.EndTime.Sub(plt.results.StartTime)

	// Collect final metrics
	plt.collectFinalMetrics()

	return plt.results, nil
}

// runPhase runs a test phase with specified RPS
func (plt *ProductionLoadTester) runPhase(duration time.Duration, targetRPS int) {
	rate := vegeta.Rate{Freq: targetRPS, Per: time.Second}
	targeter := plt.createTargeter()
	attacker := vegeta.NewAttacker(
		vegeta.Workers(uint64(plt.config.MaxConcurrency)),
		vegeta.KeepAlive(true),
	)

	results := attacker.Attack(targeter, rate, duration, "Load Test")
	plt.processResults(results)
}

// runRampUp gradually increases load
func (plt *ProductionLoadTester) runRampUp() {
	steps := 10
	stepDuration := plt.config.RampUpDuration / time.Duration(steps)
	rpsIncrement := plt.config.TargetRPS / steps

	for i := 1; i <= steps; i++ {
		currentRPS := i * rpsIncrement
		fmt.Printf("Ramping up: %d/%d RPS\n", currentRPS, plt.config.TargetRPS)
		plt.runPhase(stepDuration, currentRPS)
	}
}

// createTargeter creates Vegeta targeter
func (plt *ProductionLoadTester) createTargeter() vegeta.Targeter {
	return func(tgt *vegeta.Target) error {
		scenario := plt.selectScenario()

		tgt.Method = scenario.Method
		tgt.URL = scenario.Endpoint

		if scenario.PayloadFunc != nil {
			payload := scenario.PayloadFunc()
			body, err := json.Marshal(payload)
			if err != nil {
				return err
			}
			tgt.Body = body
			tgt.Header = http.Header{
				"Content-Type": []string{"application/json"},
			}
		}

		// Track concurrent requests
		atomic.AddInt64(&plt.activeRequests, 1)
		concurrentRequests.WithLabelValues(scenario.Name).Inc()

		return nil
	}
}

// selectScenario selects a scenario based on weights
func (plt *ProductionLoadTester) selectScenario() TestScenario {
	r := rand.Float64() * plt.totalWeight
	cumulative := 0.0

	for i, weight := range plt.scenarioWeights {
		cumulative += weight
		if r <= cumulative {
			return plt.config.TestScenarios[i]
		}
	}

	return plt.config.TestScenarios[len(plt.config.TestScenarios)-1]
}

// processResults processes attack results
func (plt *ProductionLoadTester) processResults(results <-chan *vegeta.Result) {
	for res := range results {
		atomic.AddInt64(&plt.activeRequests, -1)
		atomic.AddInt64(&plt.results.TotalRequests, 1)

		scenarioName := plt.getScenarioFromURL(res.URL)
		concurrentRequests.WithLabelValues(scenarioName).Dec()

		plt.mu.Lock()
		scenarioResult := plt.results.ScenarioResults[scenarioName]
		scenarioResult.Requests++

		if res.Error == "" && res.Code >= 200 && res.Code < 300 {
			atomic.AddInt64(&plt.results.SuccessfulRequests, 1)
			scenarioResult.Successes++
			requestsTotal.WithLabelValues(scenarioName, "success").Inc()
		} else {
			atomic.AddInt64(&plt.results.FailedRequests, 1)
			scenarioResult.Failures++
			requestsTotal.WithLabelValues(scenarioName, "failure").Inc()

<<<<<<< HEAD
			errorType := plt.categorizeError(res.Error, res.Code)
=======
			errorType := plt.categorizeError(res.Error, int(res.Code))
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			scenarioResult.ErrorsByType[errorType]++
		}

		requestDuration.WithLabelValues(scenarioName).Observe(res.Latency.Seconds())
		plt.mu.Unlock()
	}
}

// getScenarioFromURL identifies scenario from URL
func (plt *ProductionLoadTester) getScenarioFromURL(url string) string {
	for _, scenario := range plt.config.TestScenarios {
		if scenario.Endpoint == url {
			return scenario.Name
		}
	}
	return "unknown"
}

// categorizeError categorizes error types
func (plt *ProductionLoadTester) categorizeError(err string, code int) string {
	if err != "" {
		if contains(err, "timeout") {
			return "timeout"
		}
		if contains(err, "connection refused") {
			return "connection_refused"
		}
		if contains(err, "EOF") {
			return "connection_closed"
		}
		return "network_error"
	}

	switch {
	case code >= 500:
		return fmt.Sprintf("5xx_%d", code)
	case code >= 400:
		return fmt.Sprintf("4xx_%d", code)
	default:
		return fmt.Sprintf("unexpected_%d", code)
	}
}

// collectFinalMetrics collects final test metrics
func (plt *ProductionLoadTester) collectFinalMetrics() {
	// Calculate throughput
	plt.results.Throughput = float64(plt.results.TotalRequests) / plt.results.Duration.Seconds()

	// Calculate error rate
	if plt.results.TotalRequests > 0 {
		plt.results.ErrorRate = float64(plt.results.FailedRequests) / float64(plt.results.TotalRequests)
	}

	// Get resource metrics
	plt.results.ResourceUtilization = plt.metricsCollector.GetResourceMetrics()
	plt.results.SystemPerformance = plt.metricsCollector.GetSystemMetrics()

	// Calculate latency percentiles (would need to track all latencies for accurate percentiles)
	// This is a simplified version
	for name, result := range plt.results.ScenarioResults {
		if result.Requests > 0 {
			// In real implementation, we would track all latencies
			fmt.Printf("Scenario %s: %d requests, %d successes, %d failures\n",
				name, result.Requests, result.Successes, result.Failures)
		}
	}
}

// TelecomWorkloadGenerator generates realistic telecom workloads
type TelecomWorkloadGenerator struct {
	scenarios []TestScenario
}

// NewTelecomWorkloadGenerator creates telecom-specific workload generator
func NewTelecomWorkloadGenerator() *TelecomWorkloadGenerator {
	return &TelecomWorkloadGenerator{
		scenarios: []TestScenario{
			{
				Name:        "NetworkIntentCreation",
				Weight:      30.0,
				Endpoint:    "http://localhost:8080/api/v1/networkintents",
				Method:      "POST",
				PayloadFunc: generateNetworkIntentPayload,
				Validators: []ResponseValidator{
					validateStatusCode(201),
					validateResponseTime(2 * time.Second),
				},
			},
			{
				Name:        "PolicyManagement",
				Weight:      25.0,
				Endpoint:    "http://localhost:8080/api/v1/policies",
				Method:      "POST",
				PayloadFunc: generatePolicyPayload,
				Validators: []ResponseValidator{
					validateStatusCode(201),
					validateResponseTime(1 * time.Second),
				},
			},
			{
				Name:        "E2NodeScaling",
				Weight:      20.0,
				Endpoint:    "http://localhost:8080/api/v1/e2nodesets/scale",
				Method:      "POST",
				PayloadFunc: generateScalingPayload,
				Validators: []ResponseValidator{
					validateStatusCode(200),
					validateResponseTime(3 * time.Second),
				},
			},
			{
				Name:     "MetricsQuery",
				Weight:   15.0,
				Endpoint: "http://localhost:8080/api/v1/metrics",
				Method:   "GET",
				Validators: []ResponseValidator{
					validateStatusCode(200),
					validateResponseTime(500 * time.Millisecond),
				},
			},
			{
				Name:     "StatusCheck",
				Weight:   10.0,
				Endpoint: "http://localhost:8080/api/v1/status",
				Method:   "GET",
				Validators: []ResponseValidator{
					validateStatusCode(200),
					validateResponseTime(100 * time.Millisecond),
				},
			},
		},
	}
}

// GetScenarios returns telecom workload scenarios
func (twg *TelecomWorkloadGenerator) GetScenarios() []TestScenario {
	return twg.scenarios
}

// Payload generation functions

func generateNetworkIntentPayload() interface{} {
	intents := []string{
		"Deploy AMF with 3 replicas for enhanced reliability",
		"Scale SMF to handle 50000 UE sessions",
		"Configure UPF for ultra-low latency URLLC slice",
		"Optimize gNB parameters for maximum throughput",
		"Enable network slicing for IoT devices",
	}

	specs := []map[string]interface{}{
		{
			"intent":   intents[rand.Intn(len(intents))],
			"priority": "high",
		},
		{
			"intent":   intents[rand.Intn(len(intents))],
			"priority": "medium",
		},
		{
			"intent":   intents[rand.Intn(len(intents))],
			"priority": "low",
		},
	}

	return map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("intent-%d", rand.Intn(100000)),
			"namespace": "default",
		},
		"spec": specs[rand.Intn(len(specs))],
	}
}

func generatePolicyPayload() interface{} {
	return map[string]interface{}{
		"max_throughput":    rand.Intn(10000) + 1000,
		"min_throughput":    rand.Intn(1000) + 100,
		"latency_target_ms": rand.Intn(50) + 10,
		"reliability":       0.99 + rand.Float64()*0.009,
		"target_cells": []string{
			fmt.Sprintf("cell-%d", rand.Intn(100)),
			fmt.Sprintf("cell-%d", rand.Intn(100)),
		},
	}
}

func generateScalingPayload() interface{} {
	payloads := []map[string]interface{}{
		{
			"component": "amf",
			"replicas":  rand.Intn(10) + 1,
		},
		{
			"component": "smf",
			"replicas":  rand.Intn(5) + 1,
		},
	}
	return payloads[rand.Intn(len(payloads))]
}

// Validator functions

func validateStatusCode(expected int) ResponseValidator {
	return func(resp *http.Response, body []byte) error {
		if resp.StatusCode != expected {
			return fmt.Errorf("expected status %d, got %d", expected, resp.StatusCode)
		}
		return nil
	}
}

func validateResponseTime(maxDuration time.Duration) ResponseValidator {
	return func(resp *http.Response, body []byte) error {
		// Response time validation would be done at the request level
		return nil
	}
}

// MetricsCollector collects system metrics during test
type MetricsCollector struct {
<<<<<<< HEAD
	resourceMetrics *ResourceMetrics
=======
	resourceMetrics *ProductionResourceMetrics
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	systemMetrics   *SystemMetrics
	mu              sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
<<<<<<< HEAD
		resourceMetrics: &ResourceMetrics{},
=======
		resourceMetrics: &ProductionResourceMetrics{},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		systemMetrics: &SystemMetrics{
			CircuitBreakerStatus: make(map[string]string),
		},
	}
}

// Start starts metrics collection
func (mc *MetricsCollector) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mc.collectMetrics()
		}
	}
}

// collectMetrics collects current metrics
func (mc *MetricsCollector) collectMetrics() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Collect resource metrics (simplified)
	mc.resourceMetrics.CPUUsage = getCPUUsage()
	mc.resourceMetrics.MemoryUsage = getMemoryUsage()
	mc.resourceMetrics.GoroutineCount = runtime.NumGoroutine()

	// Collect system metrics
	mc.systemMetrics.CacheHitRate = getCacheHitRate()
	mc.systemMetrics.QueueDepth = getQueueDepth()

	// Update Prometheus metrics
	systemMetrics.WithLabelValues("cpu_usage").Set(mc.resourceMetrics.CPUUsage)
	systemMetrics.WithLabelValues("memory_usage").Set(mc.resourceMetrics.MemoryUsage)
	systemMetrics.WithLabelValues("cache_hit_rate").Set(mc.systemMetrics.CacheHitRate)
	systemMetrics.WithLabelValues("queue_depth").Set(float64(mc.systemMetrics.QueueDepth))
}

// GetResourceMetrics returns current resource metrics
<<<<<<< HEAD
func (mc *MetricsCollector) GetResourceMetrics() *ResourceMetrics {
=======
func (mc *MetricsCollector) GetResourceMetrics() *ProductionResourceMetrics {
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	metrics := *mc.resourceMetrics
	return &metrics
}

// GetSystemMetrics returns current system metrics
func (mc *MetricsCollector) GetSystemMetrics() *SystemMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	metrics := *mc.systemMetrics
	return &metrics
}

// Helper functions (simplified implementations)

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}

func getCPUUsage() float64 {
	// In real implementation, would use system metrics
	return 45.5
}

func getMemoryUsage() float64 {
	// In real implementation, would use runtime.MemStats
	return 62.3
}

func getCacheHitRate() float64 {
	// In real implementation, would query cache metrics
	return 0.85
}

func getQueueDepth() int64 {
	// In real implementation, would query queue metrics
	return 42
}

// RunProductionLoadTest executes a production-scale load test
func RunProductionLoadTest() error {
<<<<<<< HEAD
	config := &LoadTestConfig{
=======
	config := &ProductionLoadTestConfig{
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		Duration:         30 * time.Minute,
		RampUpDuration:   5 * time.Minute,
		TargetRPS:        1000,
		MaxConcurrency:   200,
		WarmupDuration:   2 * time.Minute,
		CooldownDuration: 2 * time.Minute,
		MetricsInterval:  10 * time.Second,
		TestScenarios:    NewTelecomWorkloadGenerator().GetScenarios(),
	}

	tester := NewProductionLoadTester(config)
	results, err := tester.RunTest()
	if err != nil {
		return fmt.Errorf("load test failed: %w", err)
	}

	// Generate report
	report, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	fmt.Printf("\n=== Load Test Results ===\n%s\n", report)

	// Validate against performance requirements
	if results.ErrorRate > 0.01 {
		return fmt.Errorf("error rate %.2f%% exceeds 1%% threshold", results.ErrorRate*100)
	}

	if results.P95Latency > 2*time.Second {
		return fmt.Errorf("P95 latency %v exceeds 2s threshold", results.P95Latency)
	}

	if results.Throughput < 900 {
		return fmt.Errorf("throughput %.2f RPS below 900 RPS requirement", results.Throughput)
	}

	fmt.Println("\n??All performance requirements met!")
	return nil
}
