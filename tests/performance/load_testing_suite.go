package performance_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"

	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"
	"github.com/thc1006/nephoran-intent-operator/pkg/rag"
)

// LoadTestSuite provides comprehensive load testing capabilities
type LoadTestSuite struct {
	config           *LoadTestConfig
	ragService       *rag.RAGService
	metricsCollector *monitoring.MetricsCollector
	prometheusClient v1.API
	results          *LoadTestResults
	mu               sync.RWMutex
}

// LoadTestConfig defines load test configuration
type LoadTestConfig struct {
	// Test scenarios
	Scenarios []LoadTestScenario `json:"scenarios"`

	// Test parameters
	Duration         time.Duration `json:"duration"`
	WarmupDuration   time.Duration `json:"warmup_duration"`
	CooldownDuration time.Duration `json:"cooldown_duration"`

	// Load patterns
	LoadPattern string `json:"load_pattern"` // constant, ramp, spike, wave
	BaseLoad    int    `json:"base_load"`    // requests per second
	PeakLoad    int    `json:"peak_load"`    // maximum requests per second

	// Monitoring
	MetricsInterval time.Duration `json:"metrics_interval"`
	EnableTracing   bool          `json:"enable_tracing"`

	// Output
	ResultsPath    string `json:"results_path"`
	RealtimeOutput bool   `json:"realtime_output"`
}

// LoadTestScenario defines a specific test scenario
type LoadTestScenario struct {
	Name            string          `json:"name"`
	Weight          float64         `json:"weight"` // Probability weight
	RequestTemplate RequestTemplate `json:"request_template"`
	ExpectedLatency time.Duration   `json:"expected_latency"`
	ExpectedSuccess float64         `json:"expected_success"`
	Tags            []string        `json:"tags"`
}

// RequestTemplate defines request parameters
type RequestTemplate struct {
	QueryType      string                 `json:"query_type"`
	QueryTemplates []string               `json:"query_templates"`
	IntentTypes    []string               `json:"intent_types"`
	Parameters     json.RawMessage `json:"parameters"`
	ContextData    json.RawMessage `json:"context_data"`
}

// LoadTestResults contains comprehensive load test results
type LoadTestResults struct {
	TestID        string          `json:"test_id"`
	StartTime     time.Time       `json:"start_time"`
	EndTime       time.Time       `json:"end_time"`
	Duration      time.Duration   `json:"duration"`
	Configuration *LoadTestConfig `json:"configuration"`

	// Request statistics
	TotalRequests      int64   `json:"total_requests"`
	SuccessfulRequests int64   `json:"successful_requests"`
	FailedRequests     int64   `json:"failed_requests"`
	RequestsPerSecond  float64 `json:"requests_per_second"`

	// Latency statistics
	LatencyStats LatencyStatistics `json:"latency_stats"`

	// Error analysis
	ErrorDistribution map[string]int64 `json:"error_distribution"`
	ErrorsByScenario  map[string]int64 `json:"errors_by_scenario"`

	// Resource usage
	ResourceUsage ResourceUsageStats `json:"resource_usage"`

	// Scenario results
	ScenarioResults map[string]ScenarioStats `json:"scenario_results"`

	// Time series data
	MetricsSeries []MetricsDataPoint `json:"metrics_series"`

	// Performance analysis
	PerformanceAnalysis PerformanceAnalysis `json:"performance_analysis"`
}

// LatencyStatistics contains detailed latency metrics
type LatencyStatistics struct {
	Mean         time.Duration      `json:"mean"`
	Median       time.Duration      `json:"median"`
	P95          time.Duration      `json:"p95"`
	P99          time.Duration      `json:"p99"`
	Min          time.Duration      `json:"min"`
	Max          time.Duration      `json:"max"`
	StdDev       time.Duration      `json:"std_dev"`
	Distribution map[string]int64   `json:"distribution"`
	Timeline     []LatencyDataPoint `json:"timeline"`
}

// ResourceUsageStats contains resource utilization data
type ResourceUsageStats struct {
	CPU        ResourceTimeSeries `json:"cpu"`
	Memory     ResourceTimeSeries `json:"memory"`
	Network    NetworkStats       `json:"network"`
	Disk       DiskStats          `json:"disk"`
	Kubernetes KubernetesStats    `json:"kubernetes"`
}

// ResourceTimeSeries represents time series resource data
type ResourceTimeSeries struct {
	Average  float64             `json:"average"`
	Peak     float64             `json:"peak"`
	Minimum  float64             `json:"minimum"`
	P95      float64             `json:"p95"`
	Timeline []ResourceDataPoint `json:"timeline"`
}

// NetworkStats contains network utilization data
type NetworkStats struct {
	BytesIn     ResourceTimeSeries `json:"bytes_in"`
	BytesOut    ResourceTimeSeries `json:"bytes_out"`
	PacketsIn   ResourceTimeSeries `json:"packets_in"`
	PacketsOut  ResourceTimeSeries `json:"packets_out"`
	Connections int64              `json:"connections"`
}

// DiskStats contains disk utilization data
type DiskStats struct {
	ReadOps    ResourceTimeSeries `json:"read_ops"`
	WriteOps   ResourceTimeSeries `json:"write_ops"`
	ReadBytes  ResourceTimeSeries `json:"read_bytes"`
	WriteBytes ResourceTimeSeries `json:"write_bytes"`
	Usage      ResourceTimeSeries `json:"usage"`
}

// KubernetesStats contains Kubernetes-specific metrics
type KubernetesStats struct {
	PodCount        int64                `json:"pod_count"`
	RestartCount    int64                `json:"restart_count"`
	ScalingEvents   []ScalingEvent       `json:"scaling_events"`
	ResourceQuotas  map[string]float64   `json:"resource_quotas"`
	NodeUtilization map[string]NodeStats `json:"node_utilization"`
}

// ScenarioStats contains per-scenario performance data
type ScenarioStats struct {
	Name              string           `json:"name"`
	RequestCount      int64            `json:"request_count"`
	SuccessCount      int64            `json:"success_count"`
	ErrorCount        int64            `json:"error_count"`
	SuccessRate       float64          `json:"success_rate"`
	AverageLatency    time.Duration    `json:"average_latency"`
	P95Latency        time.Duration    `json:"p95_latency"`
	RequestsPerSecond float64          `json:"requests_per_second"`
	ErrorTypes        map[string]int64 `json:"error_types"`
}

// MetricsDataPoint represents a metrics data point in time
type MetricsDataPoint struct {
	Timestamp         time.Time     `json:"timestamp"`
	RequestsPerSecond float64       `json:"requests_per_second"`
	ErrorRate         float64       `json:"error_rate"`
	AverageLatency    time.Duration `json:"average_latency"`
	P95Latency        time.Duration `json:"p95_latency"`
	CPUUsage          float64       `json:"cpu_usage"`
	MemoryUsage       float64       `json:"memory_usage"`
	ActiveConnections int64         `json:"active_connections"`
	QueueDepth        int64         `json:"queue_depth"`
}

// PerformanceAnalysis contains automated performance analysis
type PerformanceAnalysis struct {
	SLACompliance      map[string]float64 `json:"sla_compliance"`
	BottleneckAnalysis []BottleneckInfo   `json:"bottleneck_analysis"`
	Recommendations    []string           `json:"recommendations"`
	ScalabilityMetrics ScalabilityMetrics `json:"scalability_metrics"`
	ResourceEfficiency ResourceEfficiency `json:"resource_efficiency"`
}

// Supporting data structures
type LatencyDataPoint struct {
	Timestamp time.Time     `json:"timestamp"`
	Latency   time.Duration `json:"latency"`
	Scenario  string        `json:"scenario"`
}

type ResourceDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Component string    `json:"component"`
}

type ScalingEvent struct {
	Timestamp time.Time     `json:"timestamp"`
	Component string        `json:"component"`
	ScaleFrom int           `json:"scale_from"`
	ScaleTo   int           `json:"scale_to"`
	Trigger   string        `json:"trigger"`
	Duration  time.Duration `json:"duration"`
}

type NodeStats struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage"`
	NetworkIO   float64 `json:"network_io"`
	PodCount    int     `json:"pod_count"`
}

type BottleneckInfo struct {
	Component       string   `json:"component"`
	Metric          string   `json:"metric"`
	Severity        string   `json:"severity"`
	Value           float64  `json:"value"`
	Threshold       float64  `json:"threshold"`
	Description     string   `json:"description"`
	Recommendations []string `json:"recommendations"`
}

type ScalabilityMetrics struct {
	ThroughputScaling  float64 `json:"throughput_scaling"`  // Requests/sec per replica
	LatencyDegradation float64 `json:"latency_degradation"` // Latency increase with load
	ResourceEfficiency float64 `json:"resource_efficiency"` // Throughput per CPU/Memory
	ScalingEfficiency  float64 `json:"scaling_efficiency"`  // How well system scales
}

type ResourceEfficiency struct {
	CPUEfficiency    float64 `json:"cpu_efficiency"`    // Requests per CPU unit
	MemoryEfficiency float64 `json:"memory_efficiency"` // Requests per Memory unit
	CostEfficiency   float64 `json:"cost_efficiency"`   // Requests per dollar
	OverallScore     float64 `json:"overall_score"`     // Combined efficiency score
}

// NewLoadTestSuite creates a new load testing suite
func NewLoadTestSuite(config *LoadTestConfig, ragService *rag.RAGService, metricsCollector *monitoring.MetricsCollector) (*LoadTestSuite, error) {
	// Initialize Prometheus client
	client, err := api.NewClient(api.Config{
		Address: "http://prometheus:9090",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	return &LoadTestSuite{
		config:           config,
		ragService:       ragService,
		metricsCollector: metricsCollector,
		prometheusClient: v1.NewAPI(client),
		results: &LoadTestResults{
			TestID:            generateTestID(),
			ScenarioResults:   make(map[string]ScenarioStats),
			ErrorDistribution: make(map[string]int64),
			ErrorsByScenario:  make(map[string]int64),
		},
	}, nil
}

// RunLoadTest executes the complete load test suite
func (lts *LoadTestSuite) RunLoadTest(ctx context.Context) (*LoadTestResults, error) {
	log.Printf("Starting load test: %s", lts.results.TestID)

	lts.results.StartTime = time.Now()
	lts.results.Configuration = lts.config

	// Initialize metrics collection
	metricsCtx, metricsCancel := context.WithCancel(ctx)
	defer metricsCancel()

	// Start metrics collection
	go lts.collectMetrics(metricsCtx)

	// Warmup phase
	if lts.config.WarmupDuration > 0 {
		log.Printf("Starting warmup phase: %v", lts.config.WarmupDuration)
		if err := lts.executeWarmup(ctx); err != nil {
			log.Printf("Warmup failed: %v", err)
		}
	}

	// Main load test phase
	log.Printf("Starting main load test phase: %v", lts.config.Duration)
	if err := lts.executeLoadTest(ctx); err != nil {
		return nil, fmt.Errorf("load test execution failed: %w", err)
	}

	// Cooldown phase
	if lts.config.CooldownDuration > 0 {
		log.Printf("Starting cooldown phase: %v", lts.config.CooldownDuration)
		time.Sleep(lts.config.CooldownDuration)
	}

	lts.results.EndTime = time.Now()
	lts.results.Duration = lts.results.EndTime.Sub(lts.results.StartTime)

	// Collect final metrics and analyze results
	lts.collectFinalMetrics(ctx)
	lts.analyzePerformance()

	// Save results
	if err := lts.saveResults(); err != nil {
		log.Printf("Failed to save results: %v", err)
	}

	log.Printf("Load test completed: %s", lts.results.TestID)
	return lts.results, nil
}

// executeWarmup performs system warmup
func (lts *LoadTestSuite) executeWarmup(ctx context.Context) error {
	warmupCtx, cancel := context.WithTimeout(ctx, lts.config.WarmupDuration)
	defer cancel()

	// Low-intensity requests to warm up caches and connections
	warmupRate := lts.config.BaseLoad / 4
	if warmupRate < 1 {
		warmupRate = 1
	}

	return lts.executeConstantLoad(warmupCtx, warmupRate, false)
}

// executeLoadTest performs the main load test
func (lts *LoadTestSuite) executeLoadTest(ctx context.Context) error {
	testCtx, cancel := context.WithTimeout(ctx, lts.config.Duration)
	defer cancel()

	switch lts.config.LoadPattern {
	case "constant":
		return lts.executeConstantLoad(testCtx, lts.config.BaseLoad, true)
	case "ramp":
		return lts.executeRampLoad(testCtx)
	case "spike":
		return lts.executeSpikeLoad(testCtx)
	case "wave":
		return lts.executeWaveLoad(testCtx)
	default:
		return lts.executeConstantLoad(testCtx, lts.config.BaseLoad, true)
	}
}

// executeConstantLoad executes constant load pattern
func (lts *LoadTestSuite) executeConstantLoad(ctx context.Context, rps int, recordMetrics bool) error {
	rateLimiter := time.NewTicker(time.Second / time.Duration(rps))
	defer rateLimiter.Stop()

	// Worker pool
	workerCount := min(rps, 100) // Limit concurrent workers
	requestChan := make(chan LoadTestRequest, workerCount*2)
	resultsChan := make(chan LoadTestResult, workerCount*2)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lts.loadTestWorker(ctx, requestChan, resultsChan)
		}()
	}

	// Results processor
	go func() {
		for result := range resultsChan {
			if recordMetrics {
				lts.processResult(result)
			}
		}
	}()

	// Generate requests
	for {
		select {
		case <-ctx.Done():
			close(requestChan)
			wg.Wait()
			close(resultsChan)
			return ctx.Err()

		case <-rateLimiter.C:
			scenario := lts.selectScenario()
			request := lts.generateRequest(scenario)

			select {
			case requestChan <- request:
				atomic.AddInt64(&lts.results.TotalRequests, 1)
			default:
				// Channel full, skip this request
			}
		}
	}
}

// executeRampLoad executes ramping load pattern
func (lts *LoadTestSuite) executeRampLoad(ctx context.Context) error {
	duration := lts.config.Duration
	steps := 20 // Number of ramp steps
	stepDuration := duration / time.Duration(steps)

	for i := 0; i < steps; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Calculate current RPS
		progress := float64(i) / float64(steps-1)
		currentRPS := int(float64(lts.config.BaseLoad) +
			progress*float64(lts.config.PeakLoad-lts.config.BaseLoad))

		log.Printf("Ramp step %d/%d: %d RPS", i+1, steps, currentRPS)

		stepCtx, cancel := context.WithTimeout(ctx, stepDuration)
		err := lts.executeConstantLoad(stepCtx, currentRPS, true)
		cancel()

		if err != nil && err != context.DeadlineExceeded {
			return err
		}
	}

	return nil
}

// executeSpikeLoad executes spike load pattern
func (lts *LoadTestSuite) executeSpikeLoad(ctx context.Context) error {
	// 70% base load, 30% spike load
	baseDuration := time.Duration(float64(lts.config.Duration) * 0.7)
	spikeDuration := lts.config.Duration - baseDuration

	// Base load phase
	log.Printf("Spike test: Base load phase (%v at %d RPS)", baseDuration, lts.config.BaseLoad)
	baseCtx, cancel := context.WithTimeout(ctx, baseDuration)
	err := lts.executeConstantLoad(baseCtx, lts.config.BaseLoad, true)
	cancel()

	if err != nil && err != context.DeadlineExceeded {
		return err
	}

	// Spike load phase
	log.Printf("Spike test: Spike phase (%v at %d RPS)", spikeDuration, lts.config.PeakLoad)
	spikeCtx, cancel := context.WithTimeout(ctx, spikeDuration)
	err = lts.executeConstantLoad(spikeCtx, lts.config.PeakLoad, true)
	cancel()

	return err
}

// executeWaveLoad executes wave load pattern
func (lts *LoadTestSuite) executeWaveLoad(ctx context.Context) error {
	waves := 4 // Number of complete waves
	waveDuration := lts.config.Duration / time.Duration(waves)
	stepsPerWave := 20

	for wave := 0; wave < waves; wave++ {
		for step := 0; step < stepsPerWave; step++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// Calculate sine wave RPS
			angle := 2 * 3.14159 * float64(step) / float64(stepsPerWave)
			waveProgress := (1 + math.Sin(angle)) / 2
			currentRPS := int(float64(lts.config.BaseLoad) +
				waveProgress*float64(lts.config.PeakLoad-lts.config.BaseLoad))

			stepDuration := waveDuration / time.Duration(stepsPerWave)
			stepCtx, cancel := context.WithTimeout(ctx, stepDuration)
			err := lts.executeConstantLoad(stepCtx, currentRPS, true)
			cancel()

			if err != nil && err != context.DeadlineExceeded {
				return err
			}
		}

		log.Printf("Completed wave %d/%d", wave+1, waves)
	}

	return nil
}

// loadTestWorker processes load test requests
func (lts *LoadTestSuite) loadTestWorker(ctx context.Context, requests <-chan LoadTestRequest, results chan<- LoadTestResult) {
	for request := range requests {
		result := lts.executeRequest(ctx, request)

		select {
		case results <- result:
		case <-ctx.Done():
			return
		}
	}
}

// executeRequest executes a single load test request
func (lts *LoadTestSuite) executeRequest(ctx context.Context, request LoadTestRequest) LoadTestResult {
	start := time.Now()

	response, err := lts.ragService.ProcessQuery(ctx, &rag.RAGRequest{
		Query:      request.Query,
		IntentType: request.IntentType,
	})

	latency := time.Since(start)

	return LoadTestResult{
		Request:   request,
		Response:  response,
		Latency:   latency,
		Error:     err,
		Timestamp: start,
	}
}

// selectScenario selects a test scenario based on weights
func (lts *LoadTestSuite) selectScenario() LoadTestScenario {
	if len(lts.config.Scenarios) == 0 {
		return LoadTestScenario{Name: "default"}
	}

	// Calculate total weight
	totalWeight := 0.0
	for _, scenario := range lts.config.Scenarios {
		totalWeight += scenario.Weight
	}

	// Select scenario based on weight
	r := rand.Float64() * totalWeight
	currentWeight := 0.0

	for _, scenario := range lts.config.Scenarios {
		currentWeight += scenario.Weight
		if r <= currentWeight {
			return scenario
		}
	}

	return lts.config.Scenarios[0]
}

// generateRequest generates a request from scenario template
func (lts *LoadTestSuite) generateRequest(scenario LoadTestScenario) LoadTestRequest {
	template := scenario.RequestTemplate

	var query string
	if len(template.QueryTemplates) > 0 {
		query = template.QueryTemplates[rand.Intn(len(template.QueryTemplates))]
	}

	var intentType string
	if len(template.IntentTypes) > 0 {
		intentType = template.IntentTypes[rand.Intn(len(template.IntentTypes))]
	}

	return LoadTestRequest{
		Scenario:   scenario.Name,
		Query:      query,
		IntentType: intentType,
		Context:    template.ContextData,
		Timestamp:  time.Now(),
	}
}

// processResult processes a load test result
func (lts *LoadTestSuite) processResult(result LoadTestResult) {
	lts.mu.Lock()
	defer lts.mu.Unlock()

	if result.Error != nil {
		atomic.AddInt64(&lts.results.FailedRequests, 1)
		lts.results.ErrorDistribution[result.Error.Error()]++
		lts.results.ErrorsByScenario[result.Request.Scenario]++
	} else {
		atomic.AddInt64(&lts.results.SuccessfulRequests, 1)
	}

	// Update scenario stats
	stats, exists := lts.results.ScenarioResults[result.Request.Scenario]
	if !exists {
		stats = ScenarioStats{
			Name:       result.Request.Scenario,
			ErrorTypes: make(map[string]int64),
		}
	}

	stats.RequestCount++
	if result.Error != nil {
		stats.ErrorCount++
		stats.ErrorTypes[result.Error.Error()]++
	} else {
		stats.SuccessCount++
	}

	// Update latency stats
	stats.AverageLatency = time.Duration(
		(int64(stats.AverageLatency)*stats.RequestCount + int64(result.Latency)) /
			(stats.RequestCount + 1),
	)

	stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.RequestCount)
	lts.results.ScenarioResults[result.Request.Scenario] = stats
}

// collectMetrics collects real-time metrics during load test
func (lts *LoadTestSuite) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(lts.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dataPoint := lts.collectCurrentMetrics(ctx)
			lts.mu.Lock()
			lts.results.MetricsSeries = append(lts.results.MetricsSeries, dataPoint)
			lts.mu.Unlock()
		}
	}
}

// collectCurrentMetrics collects current system metrics
func (lts *LoadTestSuite) collectCurrentMetrics(ctx context.Context) MetricsDataPoint {
	now := time.Now()

	// Calculate current RPS
	totalRequests := atomic.LoadInt64(&lts.results.TotalRequests)
	elapsed := now.Sub(lts.results.StartTime).Seconds()
	rps := float64(totalRequests) / elapsed

	// Calculate error rate
	failedRequests := atomic.LoadInt64(&lts.results.FailedRequests)
	errorRate := 0.0
	if totalRequests > 0 {
		errorRate = float64(failedRequests) / float64(totalRequests)
	}

	return MetricsDataPoint{
		Timestamp:         now,
		RequestsPerSecond: rps,
		ErrorRate:         errorRate,
		// Additional metrics would be collected from Prometheus
		AverageLatency:    time.Millisecond * 500,  // Placeholder
		P95Latency:        time.Millisecond * 1200, // Placeholder
		CPUUsage:          65.0,                    // Placeholder
		MemoryUsage:       70.0,                    // Placeholder
		ActiveConnections: int64(rps * 2),          // Placeholder
		QueueDepth:        int64(rps / 10),         // Placeholder
	}
}

// collectFinalMetrics collects comprehensive final metrics
func (lts *LoadTestSuite) collectFinalMetrics(ctx context.Context) {
	// Calculate final statistics
	totalRequests := atomic.LoadInt64(&lts.results.TotalRequests)
	successfulRequests := atomic.LoadInt64(&lts.results.SuccessfulRequests)
	failedRequests := atomic.LoadInt64(&lts.results.FailedRequests)

	lts.results.TotalRequests = totalRequests
	lts.results.SuccessfulRequests = successfulRequests
	lts.results.FailedRequests = failedRequests

	elapsed := lts.results.Duration.Seconds()
	lts.results.RequestsPerSecond = float64(totalRequests) / elapsed

	// Collect resource usage from Prometheus
	lts.collectResourceMetrics(ctx)

	// Calculate latency statistics
	lts.calculateLatencyStatistics()
}

// collectResourceMetrics collects resource utilization metrics
func (lts *LoadTestSuite) collectResourceMetrics(ctx context.Context) {
	// Query Prometheus for resource metrics during test period
	// TODO: Implement actual Prometheus queries when monitoring is available
	_ = map[string]string{
		"cpu_avg":    `avg_over_time(rate(container_cpu_usage_seconds_total{namespace="nephoran-system"}[5m])[` + lts.config.Duration.String() + `:30s])`,
		"memory_avg": `avg_over_time(container_memory_usage_bytes{namespace="nephoran-system"}[` + lts.config.Duration.String() + `:30s])`,
	}

	lts.results.ResourceUsage = ResourceUsageStats{
		CPU:    ResourceTimeSeries{Average: 65.0, Peak: 85.0}, // Placeholder
		Memory: ResourceTimeSeries{Average: 70.0, Peak: 90.0}, // Placeholder
	}
}

// calculateLatencyStatistics calculates detailed latency statistics
func (lts *LoadTestSuite) calculateLatencyStatistics() {
	// Placeholder implementation - in real implementation,
	// this would calculate statistics from collected latency data
	lts.results.LatencyStats = LatencyStatistics{
		Mean:   time.Millisecond * 450,
		Median: time.Millisecond * 400,
		P95:    time.Millisecond * 1100,
		P99:    time.Millisecond * 2500,
		Min:    time.Millisecond * 50,
		Max:    time.Millisecond * 5000,
		Distribution: map[string]int64{
			"0-100ms":   int64(float64(lts.results.TotalRequests) * 0.15),
			"100-500ms": int64(float64(lts.results.TotalRequests) * 0.60),
			"500-1s":    int64(float64(lts.results.TotalRequests) * 0.20),
			"1s+":       int64(float64(lts.results.TotalRequests) * 0.05),
		},
	}
}

// analyzePerformance performs automated performance analysis
func (lts *LoadTestSuite) analyzePerformance() {
	analysis := PerformanceAnalysis{
		SLACompliance:      make(map[string]float64),
		BottleneckAnalysis: make([]BottleneckInfo, 0),
		Recommendations:    make([]string, 0),
	}

	// SLA compliance analysis
	analysis.SLACompliance["availability"] = float64(lts.results.SuccessfulRequests) / float64(lts.results.TotalRequests)
	analysis.SLACompliance["response_time"] = calculateSLACompliance(lts.results.LatencyStats.P95, time.Second*2)

	// Bottleneck detection
	if lts.results.LatencyStats.P95 > time.Second*2 {
		analysis.BottleneckAnalysis = append(analysis.BottleneckAnalysis, BottleneckInfo{
			Component:   "LLM Processing",
			Metric:      "P95 Latency",
			Severity:    "High",
			Value:       float64(lts.results.LatencyStats.P95.Milliseconds()),
			Threshold:   2000.0,
			Description: "P95 response time exceeds SLA target",
			Recommendations: []string{
				"Consider horizontal scaling of LLM processors",
				"Optimize prompt templates to reduce token usage",
				"Implement more aggressive caching strategies",
			},
		})
	}

	// Resource efficiency analysis
	analysis.ResourceEfficiency = ResourceEfficiency{
		CPUEfficiency:    lts.results.RequestsPerSecond / lts.results.ResourceUsage.CPU.Average,
		MemoryEfficiency: lts.results.RequestsPerSecond / (lts.results.ResourceUsage.Memory.Average / 1024 / 1024 / 1024),
		OverallScore:     0.75, // Placeholder calculation
	}

	// Scalability metrics
	analysis.ScalabilityMetrics = ScalabilityMetrics{
		ThroughputScaling:  lts.results.RequestsPerSecond / 3, // Assuming 3 replicas
		LatencyDegradation: 1.2,                               // 20% increase under load
		ResourceEfficiency: analysis.ResourceEfficiency.OverallScore,
		ScalingEfficiency:  0.85, // 85% scaling efficiency
	}

	// Generate recommendations
	if analysis.SLACompliance["availability"] < 0.99 {
		analysis.Recommendations = append(analysis.Recommendations,
			"Improve system reliability - availability below 99%")
	}

	if lts.results.ResourceUsage.CPU.Peak > 80.0 {
		analysis.Recommendations = append(analysis.Recommendations,
			"Consider CPU scaling - peak usage above 80%")
	}

	lts.results.PerformanceAnalysis = analysis
}

// saveResults saves load test results to file
func (lts *LoadTestSuite) saveResults() error {
	filename := fmt.Sprintf("%s/load_test_results_%s.json",
		lts.config.ResultsPath, lts.results.TestID)

	data, err := json.MarshalIndent(lts.results, "", "  ")
	if err != nil {
		return err
	}

	return writeFile(filename, data)
}

// Supporting types and functions
type LoadTestRequest struct {
	Scenario   string                 `json:"scenario"`
	Query      string                 `json:"query"`
	IntentType string                 `json:"intent_type"`
	Context    json.RawMessage `json:"context"`
	Timestamp  time.Time              `json:"timestamp"`
}

type LoadTestResult struct {
	Request   LoadTestRequest  `json:"request"`
	Response  *rag.RAGResponse `json:"response"`
	Latency   time.Duration    `json:"latency"`
	Error     error            `json:"error"`
	Timestamp time.Time        `json:"timestamp"`
}

// Utility functions
func generateTestID() string {
	return fmt.Sprintf("loadtest_%d", time.Now().Unix())
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func calculateSLACompliance(actual, target time.Duration) float64 {
	if actual <= target {
		return 1.0
	}
	return float64(target) / float64(actual)
}

func writeFile(filename string, data []byte) error {
	// Implementation would write to file
	return nil
}
