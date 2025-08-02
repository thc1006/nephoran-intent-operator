package performance

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"

	"github.com/thc1006/nephoran-intent-operator/pkg/rag"
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"
)

// BenchmarkFramework provides comprehensive performance benchmarking
type BenchmarkFramework struct {
	config           *BenchmarkConfig
	ragService       *rag.RAGService
	metricsCollector *monitoring.MetricsCollector
	prometheusClient v1.API
	results          *BenchmarkResults
	baselines        *BaselineMetrics
	mu               sync.RWMutex
}

// BenchmarkConfig holds benchmarking configuration
type BenchmarkConfig struct {
	// Test configuration
	TestDuration        time.Duration `json:"test_duration"`
	ConcurrencyLevels   []int         `json:"concurrency_levels"`
	RequestRates        []int         `json:"request_rates"` // requests per second
	DatasetSizes        []int         `json:"dataset_sizes"` // number of documents
	
	// Query configuration
	TestQueries         []TestQuery   `json:"test_queries"`
	QueryTypes          []string      `json:"query_types"`
	DocumentTypes       []string      `json:"document_types"`
	
	// Performance targets
	LatencyTargets      LatencyTargets `json:"latency_targets"`
	ThroughputTargets   map[string]int `json:"throughput_targets"`
	ResourceTargets     ResourceTargets `json:"resource_targets"`
	
	// Output configuration
	OutputFormat        string        `json:"output_format"`
	OutputPath          string        `json:"output_path"`
	EnableContinuous    bool          `json:"enable_continuous"`
	ReportInterval      time.Duration `json:"report_interval"`
	
	// Regression detection
	EnableRegression    bool          `json:"enable_regression"`
	RegressionThreshold float64       `json:"regression_threshold"`
	BaselinePath        string        `json:"baseline_path"`
}

// LatencyTargets defines latency performance targets
type LatencyTargets struct {
	P50Target time.Duration `json:"p50_target"`
	P95Target time.Duration `json:"p95_target"`
	P99Target time.Duration `json:"p99_target"`
	MaxTarget time.Duration `json:"max_target"`
}

// ResourceTargets defines resource utilization targets
type ResourceTargets struct {
	CPUTarget    float64 `json:"cpu_target"`
	MemoryTarget float64 `json:"memory_target"`
	DiskTarget   float64 `json:"disk_target"`
}

// TestQuery represents a benchmark query
type TestQuery struct {
	ID          string                 `json:"id"`
	Query       string                 `json:"query"`
	IntentType  string                 `json:"intent_type"`
	Context     map[string]interface{} `json:"context"`
	Weight      float64                `json:"weight"`
	Expected    ExpectedResult         `json:"expected"`
}

// ExpectedResult defines expected benchmark results
type ExpectedResult struct {
	MinLatency      time.Duration `json:"min_latency"`
	MaxLatency      time.Duration `json:"max_latency"`
	MinConfidence   float64       `json:"min_confidence"`
	MinSourceCount  int           `json:"min_source_count"`
}

// BenchmarkResults stores comprehensive benchmark results
type BenchmarkResults struct {
	Timestamp         time.Time              `json:"timestamp"`
	Configuration     *BenchmarkConfig       `json:"configuration"`
	TestDuration      time.Duration          `json:"test_duration"`
	TotalRequests     int                    `json:"total_requests"`
	SuccessfulRequests int                   `json:"successful_requests"`
	FailedRequests    int                    `json:"failed_requests"`
	
	// Latency metrics
	LatencyMetrics    LatencyMetrics         `json:"latency_metrics"`
	
	// Throughput metrics
	ThroughputMetrics ThroughputMetrics      `json:"throughput_metrics"`
	
	// Resource utilization
	ResourceMetrics   ResourceMetrics        `json:"resource_metrics"`
	
	// Component-specific metrics
	ComponentMetrics  map[string]ComponentMetrics `json:"component_metrics"`
	
	// Error analysis
	ErrorAnalysis     ErrorAnalysis          `json:"error_analysis"`
	
	// Regression analysis
	RegressionAnalysis *RegressionAnalysis   `json:"regression_analysis,omitempty"`
	
	// Test environment
	Environment       EnvironmentInfo        `json:"environment"`
}

// LatencyMetrics contains latency performance data
type LatencyMetrics struct {
	Mean         time.Duration            `json:"mean"`
	Median       time.Duration            `json:"median"`
	P95          time.Duration            `json:"p95"`
	P99          time.Duration            `json:"p99"`
	Min          time.Duration            `json:"min"`
	Max          time.Duration            `json:"max"`
	StdDev       time.Duration            `json:"std_dev"`
	Distribution map[string]int           `json:"distribution"`
	ByQuery      map[string]time.Duration `json:"by_query"`
	ByComponent  map[string]time.Duration `json:"by_component"`
}

// ThroughputMetrics contains throughput performance data
type ThroughputMetrics struct {
	RequestsPerSecond     float64            `json:"requests_per_second"`
	PeakThroughput        float64            `json:"peak_throughput"`
	SustainedThroughput   float64            `json:"sustained_throughput"`
	ThroughputVariance    float64            `json:"throughput_variance"`
	ByQuery               map[string]float64 `json:"by_query"`
	ByComponent           map[string]float64 `json:"by_component"`
}

// ResourceMetrics contains resource utilization data
type ResourceMetrics struct {
	CPU     ResourceUtilization `json:"cpu"`
	Memory  ResourceUtilization `json:"memory"`
	Disk    ResourceUtilization `json:"disk"`
	Network NetworkUtilization  `json:"network"`
}

// ResourceUtilization represents resource usage statistics
type ResourceUtilization struct {
	Average    float64 `json:"average"`
	Peak       float64 `json:"peak"`
	P95        float64 `json:"p95"`
	Variance   float64 `json:"variance"`
	Timeline   []TimePoint `json:"timeline"`
}

// NetworkUtilization represents network usage statistics
type NetworkUtilization struct {
	BytesIn     ResourceUtilization `json:"bytes_in"`
	BytesOut    ResourceUtilization `json:"bytes_out"`
	PacketsIn   ResourceUtilization `json:"packets_in"`
	PacketsOut  ResourceUtilization `json:"packets_out"`
}

// ComponentMetrics contains component-specific performance data
type ComponentMetrics struct {
	Name              string        `json:"name"`
	RequestCount      int           `json:"request_count"`
	AverageLatency    time.Duration `json:"average_latency"`
	ErrorRate         float64       `json:"error_rate"`
	CacheHitRate      float64       `json:"cache_hit_rate"`
	ResourceUsage     ResourceMetrics `json:"resource_usage"`
}

// ErrorAnalysis contains error analysis data
type ErrorAnalysis struct {
	TotalErrors       int                    `json:"total_errors"`
	ErrorRate         float64                `json:"error_rate"`
	ErrorsByType      map[string]int         `json:"errors_by_type"`
	ErrorsByComponent map[string]int         `json:"errors_by_component"`
	ErrorDistribution map[string]ErrorStats  `json:"error_distribution"`
}

// ErrorStats contains statistics for specific error types
type ErrorStats struct {
	Count      int           `json:"count"`
	Rate       float64       `json:"rate"`
	FirstSeen  time.Time     `json:"first_seen"`
	LastSeen   time.Time     `json:"last_seen"`
	Duration   time.Duration `json:"duration"`
}

// RegressionAnalysis contains regression detection results
type RegressionAnalysis struct {
	HasRegression     bool                   `json:"has_regression"`
	RegressionScore   float64                `json:"regression_score"`
	Baseline          *BaselineMetrics       `json:"baseline"`
	Changes           map[string]float64     `json:"changes"`
	Recommendations   []string               `json:"recommendations"`
}

// BaselineMetrics contains baseline performance metrics
type BaselineMetrics struct {
	Timestamp         time.Time              `json:"timestamp"`
	Version           string                 `json:"version"`
	LatencyBaseline   LatencyMetrics         `json:"latency_baseline"`
	ThroughputBaseline ThroughputMetrics     `json:"throughput_baseline"`
	ResourceBaseline  ResourceMetrics        `json:"resource_baseline"`
	ComponentBaselines map[string]ComponentMetrics `json:"component_baselines"`
}

// EnvironmentInfo contains test environment information
type EnvironmentInfo struct {
	OS               string                 `json:"os"`
	Architecture     string                 `json:"architecture"`
	CPUCores         int                    `json:"cpu_cores"`
	Memory           int64                  `json:"memory_mb"`
	GoVersion        string                 `json:"go_version"`
	KubernetesVersion string                `json:"kubernetes_version"`
	ClusterNodes     int                    `json:"cluster_nodes"`
	Configuration    map[string]interface{} `json:"configuration"`
}

// TimePoint represents a data point in time series
type TimePoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// NewBenchmarkFramework creates a new benchmark framework
func NewBenchmarkFramework(config *BenchmarkConfig, ragService *rag.RAGService, metricsCollector *monitoring.MetricsCollector) (*BenchmarkFramework, error) {
	// Initialize Prometheus client
	client, err := api.NewClient(api.Config{
		Address: "http://prometheus:9090",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	// Load baseline metrics if available
	var baselines *BaselineMetrics
	if config.BaselinePath != "" {
		baselines, _ = loadBaselineMetrics(config.BaselinePath)
	}

	return &BenchmarkFramework{
		config:           config,
		ragService:       ragService,
		metricsCollector: metricsCollector,
		prometheusClient: v1.NewAPI(client),
		baselines:        baselines,
		results:          &BenchmarkResults{},
	}, nil
}

// RunBenchmark executes the complete benchmark suite
func (bf *BenchmarkFramework) RunBenchmark(ctx context.Context) (*BenchmarkResults, error) {
	log.Printf("Starting benchmark suite with %d concurrency levels and %d request rates",
		len(bf.config.ConcurrencyLevels), len(bf.config.RequestRates))

	// Initialize results
	bf.results = &BenchmarkResults{
		Timestamp:         time.Now(),
		Configuration:     bf.config,
		ComponentMetrics:  make(map[string]ComponentMetrics),
		Environment:       bf.collectEnvironmentInfo(),
	}

	// Run benchmark scenarios
	for _, concurrency := range bf.config.ConcurrencyLevels {
		for _, rate := range bf.config.RequestRates {
			log.Printf("Running benchmark: concurrency=%d, rate=%d rps", concurrency, rate)
			
			scenarioResults, err := bf.runScenario(ctx, concurrency, rate)
			if err != nil {
				log.Printf("Scenario failed: %v", err)
				continue
			}
			
			bf.aggregateResults(scenarioResults)
		}
	}

	// Collect final metrics
	bf.collectFinalMetrics(ctx)

	// Perform regression analysis
	if bf.config.EnableRegression && bf.baselines != nil {
		bf.performRegressionAnalysis()
	}

	// Generate recommendations
	bf.generateRecommendations()

	// Save results
	if err := bf.saveResults(); err != nil {
		log.Printf("Failed to save results: %v", err)
	}

	return bf.results, nil
}

// runScenario executes a single benchmark scenario
func (bf *BenchmarkFramework) runScenario(ctx context.Context, concurrency int, requestRate int) (*ScenarioResults, error) {
	scenario := &ScenarioResults{
		Concurrency:   concurrency,
		RequestRate:   requestRate,
		StartTime:     time.Now(),
		Latencies:     make([]time.Duration, 0),
		Errors:        make([]error, 0),
		ComponentData: make(map[string][]ComponentDataPoint),
	}

	// Rate limiter
	rateLimiter := time.NewTicker(time.Second / time.Duration(requestRate))
	defer rateLimiter.Stop()

	// Worker pool
	requestChan := make(chan TestQuery, concurrency*2)
	resultsChan := make(chan RequestResult, concurrency*2)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bf.benchmarkWorker(ctx, requestChan, resultsChan)
		}()
	}

	// Results collector
	go func() {
		for result := range resultsChan {
			bf.mu.Lock()
			scenario.TotalRequests++
			if result.Error != nil {
				scenario.FailedRequests++
				scenario.Errors = append(scenario.Errors, result.Error)
			} else {
				scenario.SuccessfulRequests++
				scenario.Latencies = append(scenario.Latencies, result.Latency)
			}
			bf.mu.Unlock()
		}
	}()

	// Send requests
	testCtx, cancel := context.WithTimeout(ctx, bf.config.TestDuration)
	defer cancel()

	queryIndex := 0
	for {
		select {
		case <-testCtx.Done():
			close(requestChan)
			wg.Wait()
			close(resultsChan)
			
			scenario.EndTime = time.Now()
			scenario.Duration = scenario.EndTime.Sub(scenario.StartTime)
			return scenario, nil
			
		case <-rateLimiter.C:
			if queryIndex >= len(bf.config.TestQueries) {
				queryIndex = 0
			}
			
			select {
			case requestChan <- bf.config.TestQueries[queryIndex]:
				queryIndex++
			default:
				// Channel full, skip this iteration
			}
		}
	}
}

// benchmarkWorker processes benchmark requests
func (bf *BenchmarkFramework) benchmarkWorker(ctx context.Context, requests <-chan TestQuery, results chan<- RequestResult) {
	for query := range requests {
		start := time.Now()
		
		response, err := bf.ragService.ProcessQuery(ctx, &rag.QueryRequest{
			Query:      query.Query,
			IntentType: query.IntentType,
			Context:    query.Context,
		})
		
		latency := time.Since(start)
		
		results <- RequestResult{
			Query:     query,
			Response:  response,
			Latency:   latency,
			Error:     err,
			Timestamp: start,
		}
	}
}

// collectFinalMetrics collects comprehensive performance metrics
func (bf *BenchmarkFramework) collectFinalMetrics(ctx context.Context) {
	// Collect latency metrics from Prometheus
	bf.collectLatencyMetrics(ctx)
	
	// Collect throughput metrics
	bf.collectThroughputMetrics(ctx)
	
	// Collect resource metrics
	bf.collectResourceMetrics(ctx)
	
	// Collect component metrics
	bf.collectComponentMetrics(ctx)
	
	// Collect error metrics
	bf.collectErrorMetrics(ctx)
}

// collectLatencyMetrics collects latency performance data
func (bf *BenchmarkFramework) collectLatencyMetrics(ctx context.Context) {
	// Query Prometheus for latency data
	queries := map[string]string{
		"p50": `histogram_quantile(0.50, rate(rag_query_latency_seconds_bucket[5m]))`,
		"p95": `histogram_quantile(0.95, rate(rag_query_latency_seconds_bucket[5m]))`,
		"p99": `histogram_quantile(0.99, rate(rag_query_latency_seconds_bucket[5m]))`,
		"mean": `rate(rag_query_latency_seconds_sum[5m]) / rate(rag_query_latency_seconds_count[5m])`,
	}

	latencyMetrics := LatencyMetrics{
		ByQuery:     make(map[string]time.Duration),
		ByComponent: make(map[string]time.Duration),
		Distribution: make(map[string]int),
	}

	for name, query := range queries {
		result, warnings, err := bf.prometheusClient.Query(ctx, query, time.Now())
		if err != nil {
			log.Printf("Failed to query %s latency: %v", name, err)
			continue
		}
		if len(warnings) > 0 {
			log.Printf("Warnings for %s query: %v", name, warnings)
		}

		// Process results and update latencyMetrics
		// Implementation depends on result format
		_ = result // Placeholder
	}

	bf.results.LatencyMetrics = latencyMetrics
}

// collectThroughputMetrics collects throughput performance data
func (bf *BenchmarkFramework) collectThroughputMetrics(ctx context.Context) {
	query := `rate(rag_queries_total[5m])`
	
	result, _, err := bf.prometheusClient.Query(ctx, query, time.Now())
	if err != nil {
		log.Printf("Failed to query throughput metrics: %v", err)
		return
	}

	throughputMetrics := ThroughputMetrics{
		ByQuery:     make(map[string]float64),
		ByComponent: make(map[string]float64),
	}

	// Process results
	_ = result // Placeholder

	bf.results.ThroughputMetrics = throughputMetrics
}

// collectResourceMetrics collects resource utilization data
func (bf *BenchmarkFramework) collectResourceMetrics(ctx context.Context) {
	resourceQueries := map[string]string{
		"cpu_usage":    `rate(process_cpu_seconds_total[5m]) * 100`,
		"memory_usage": `process_resident_memory_bytes / 1024 / 1024`,
		"disk_usage":   `rate(process_disk_reads_total[5m]) + rate(process_disk_writes_total[5m])`,
	}

	resourceMetrics := ResourceMetrics{}

	for resource, query := range resourceQueries {
		result, _, err := bf.prometheusClient.Query(ctx, query, time.Now())
		if err != nil {
			log.Printf("Failed to query %s metrics: %v", resource, err)
			continue
		}

		// Process results
		_ = result // Placeholder
	}

	bf.results.ResourceMetrics = resourceMetrics
}

// collectComponentMetrics collects component-specific metrics
func (bf *BenchmarkFramework) collectComponentMetrics(ctx context.Context) {
	components := []string{"rag-api", "weaviate", "llm-processor", "document-processor"}

	for _, component := range components {
		metrics := ComponentMetrics{
			Name: component,
		}

		// Collect component-specific metrics
		// Implementation would query Prometheus for component metrics

		bf.results.ComponentMetrics[component] = metrics
	}
}

// collectErrorMetrics collects error analysis data
func (bf *BenchmarkFramework) collectErrorMetrics(ctx context.Context) {
	errorQuery := `rate(rag_errors_total[5m])`
	
	result, _, err := bf.prometheusClient.Query(ctx, errorQuery, time.Now())
	if err != nil {
		log.Printf("Failed to query error metrics: %v", err)
		return
	}

	errorAnalysis := ErrorAnalysis{
		ErrorsByType:      make(map[string]int),
		ErrorsByComponent: make(map[string]int),
		ErrorDistribution: make(map[string]ErrorStats),
	}

	// Process error results
	_ = result // Placeholder

	bf.results.ErrorAnalysis = errorAnalysis
}

// performRegressionAnalysis performs regression detection
func (bf *BenchmarkFramework) performRegressionAnalysis() {
	if bf.baselines == nil {
		return
	}

	analysis := &RegressionAnalysis{
		Baseline:        bf.baselines,
		Changes:         make(map[string]float64),
		Recommendations: make([]string, 0),
	}

	// Compare current results with baseline
	currentP95 := bf.results.LatencyMetrics.P95
	baselineP95 := bf.baselines.LatencyBaseline.P95
	
	if baselineP95 > 0 {
		latencyChange := float64(currentP95-baselineP95) / float64(baselineP95)
		analysis.Changes["latency_p95"] = latencyChange
		
		if latencyChange > bf.config.RegressionThreshold {
			analysis.HasRegression = true
			analysis.RegressionScore += latencyChange
			analysis.Recommendations = append(analysis.Recommendations,
				fmt.Sprintf("P95 latency increased by %.2f%% (threshold: %.2f%%)",
					latencyChange*100, bf.config.RegressionThreshold*100))
		}
	}

	// Compare throughput
	currentThroughput := bf.results.ThroughputMetrics.RequestsPerSecond
	baselineThroughput := bf.baselines.ThroughputBaseline.RequestsPerSecond
	
	if baselineThroughput > 0 {
		throughputChange := (currentThroughput - baselineThroughput) / baselineThroughput
		analysis.Changes["throughput"] = throughputChange
		
		if throughputChange < -bf.config.RegressionThreshold {
			analysis.HasRegression = true
			analysis.RegressionScore += -throughputChange
			analysis.Recommendations = append(analysis.Recommendations,
				fmt.Sprintf("Throughput decreased by %.2f%% (threshold: %.2f%%)",
					-throughputChange*100, bf.config.RegressionThreshold*100))
		}
	}

	bf.results.RegressionAnalysis = analysis
}

// generateRecommendations generates performance improvement recommendations
func (bf *BenchmarkFramework) generateRecommendations() {
	recommendations := make([]string, 0)

	// Latency recommendations
	if bf.results.LatencyMetrics.P95 > bf.config.LatencyTargets.P95Target {
		recommendations = append(recommendations,
			"Consider enabling caching or optimizing database queries to reduce P95 latency")
	}

	// Throughput recommendations
	if bf.results.ThroughputMetrics.RequestsPerSecond < float64(bf.config.ThroughputTargets["target"]) {
		recommendations = append(recommendations,
			"Consider horizontal scaling or connection pooling to improve throughput")
	}

	// Resource recommendations
	if bf.results.ResourceMetrics.CPU.Peak > bf.config.ResourceTargets.CPUTarget {
		recommendations = append(recommendations,
			"High CPU usage detected - consider optimizing algorithms or scaling up")
	}

	// Error rate recommendations
	if bf.results.ErrorAnalysis.ErrorRate > 0.01 { // 1% error rate
		recommendations = append(recommendations,
			"Error rate exceeds acceptable threshold - investigate error causes")
	}

	if bf.results.RegressionAnalysis != nil {
		bf.results.RegressionAnalysis.Recommendations = append(
			bf.results.RegressionAnalysis.Recommendations, recommendations...)
	}
}

// saveResults saves benchmark results to file
func (bf *BenchmarkFramework) saveResults() error {
	filename := fmt.Sprintf("%s/benchmark_results_%s.json",
		bf.config.OutputPath, time.Now().Format("20060102_150405"))

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create results file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	
	if err := encoder.Encode(bf.results); err != nil {
		return fmt.Errorf("failed to encode results: %w", err)
	}

	log.Printf("Benchmark results saved to: %s", filename)
	return nil
}

// collectEnvironmentInfo collects test environment information
func (bf *BenchmarkFramework) collectEnvironmentInfo() EnvironmentInfo {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return EnvironmentInfo{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		CPUCores:     runtime.NumCPU(),
		Memory:       int64(memStats.Sys / 1024 / 1024), // MB
		GoVersion:    runtime.Version(),
		Configuration: map[string]interface{}{
			"max_procs": runtime.GOMAXPROCS(0),
		},
	}
}

// aggregateResults aggregates scenario results into overall results
func (bf *BenchmarkFramework) aggregateResults(scenario *ScenarioResults) {
	bf.mu.Lock()
	defer bf.mu.Unlock()

	bf.results.TotalRequests += scenario.TotalRequests
	bf.results.SuccessfulRequests += scenario.SuccessfulRequests
	bf.results.FailedRequests += scenario.FailedRequests
}

// Supporting types

// ScenarioResults contains results from a single benchmark scenario
type ScenarioResults struct {
	Concurrency        int                             `json:"concurrency"`
	RequestRate        int                             `json:"request_rate"`
	StartTime          time.Time                       `json:"start_time"`
	EndTime            time.Time                       `json:"end_time"`
	Duration           time.Duration                   `json:"duration"`
	TotalRequests      int                             `json:"total_requests"`
	SuccessfulRequests int                             `json:"successful_requests"`
	FailedRequests     int                             `json:"failed_requests"`
	Latencies          []time.Duration                 `json:"latencies"`
	Errors             []error                         `json:"errors"`
	ComponentData      map[string][]ComponentDataPoint `json:"component_data"`
}

// RequestResult contains the result of a single request
type RequestResult struct {
	Query     TestQuery         `json:"query"`
	Response  *rag.RAGResponse  `json:"response"`
	Latency   time.Duration     `json:"latency"`
	Error     error             `json:"error"`
	Timestamp time.Time         `json:"timestamp"`
}

// ComponentDataPoint represents a data point for component metrics
type ComponentDataPoint struct {
	Timestamp time.Time              `json:"timestamp"`
	Metrics   map[string]interface{} `json:"metrics"`
}

// loadBaselineMetrics loads baseline metrics from file
func loadBaselineMetrics(path string) (*BaselineMetrics, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var baseline BaselineMetrics
	if err := json.NewDecoder(file).Decode(&baseline); err != nil {
		return nil, err
	}

	return &baseline, nil
}

// GetDefaultBenchmarkConfig returns default benchmark configuration
func GetDefaultBenchmarkConfig() *BenchmarkConfig {
	return &BenchmarkConfig{
		TestDuration:      5 * time.Minute,
		ConcurrencyLevels: []int{1, 5, 10, 20},
		RequestRates:      []int{10, 50, 100, 200},
		DatasetSizes:      []int{100, 500, 1000},
		LatencyTargets: LatencyTargets{
			P50Target: 500 * time.Millisecond,
			P95Target: 2 * time.Second,
			P99Target: 5 * time.Second,
			MaxTarget: 10 * time.Second,
		},
		ThroughputTargets: map[string]int{
			"target": 100, // requests per second
		},
		ResourceTargets: ResourceTargets{
			CPUTarget:    80.0, // 80%
			MemoryTarget: 85.0, // 85%
			DiskTarget:   70.0, // 70%
		},
		OutputFormat:        "json",
		OutputPath:          "./benchmark_results",
		EnableContinuous:    false,
		ReportInterval:      1 * time.Minute,
		EnableRegression:    true,
		RegressionThreshold: 0.1, // 10% threshold
		TestQueries: []TestQuery{
			{
				ID:         "5g_config_query",
				Query:      "How do I configure 5G network slicing?",
				IntentType: "configuration_request",
				Weight:     1.0,
				Expected: ExpectedResult{
					MinLatency:     100 * time.Millisecond,
					MaxLatency:     5 * time.Second,
					MinConfidence:  0.7,
					MinSourceCount: 2,
				},
			},
			{
				ID:         "oran_knowledge_query",
				Query:      "What is the O-RAN E2 interface?",
				IntentType: "knowledge_request",
				Weight:     1.0,
				Expected: ExpectedResult{
					MinLatency:     50 * time.Millisecond,
					MaxLatency:     3 * time.Second,
					MinConfidence:  0.8,
					MinSourceCount: 1,
				},
			},
		},
	}
}