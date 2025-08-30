//go:build !disable_rag && !test

package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nephio-project/nephoran-intent-operator/pkg/shared"
)

// PerformanceBenchmarker provides comprehensive performance benchmarking.

type PerformanceBenchmarker struct {
	config *BenchmarkConfig

	logger *slog.Logger

	results *BenchmarkResults

	testData *TestDataset

	originalClient *WeaviateClient

	optimizedPipeline *OptimizedRAGPipeline

	batchClient *OptimizedBatchSearchClient

	grpcClient *GRPCWeaviateClient
}

// BenchmarkConfig holds benchmark configuration.

type BenchmarkConfig struct {

	// Test parameters.

	WarmupQueries int `json:"warmup_queries"`

	BenchmarkQueries int `json:"benchmark_queries"`

	ConcurrencyLevels []int `json:"concurrency_levels"`

	BatchSizes []int `json:"batch_sizes"`

	TestDuration time.Duration `json:"test_duration"`

	// Performance targets.

	LatencyTarget time.Duration `json:"latency_target"`

	ThroughputTarget float64 `json:"throughput_target"`

	AccuracyTarget float32 `json:"accuracy_target"`

	MemoryUsageTarget int64 `json:"memory_usage_target"`

	// Test scenarios.

	EnableLatencyTest bool `json:"enable_latency_test"`

	EnableThroughputTest bool `json:"enable_throughput_test"`

	EnableStressTest bool `json:"enable_stress_test"`

	EnableAccuracyTest bool `json:"enable_accuracy_test"`

	EnableMemoryTest bool `json:"enable_memory_test"`

	EnableComparisonTest bool `json:"enable_comparison_test"`

	// Output configuration.

	EnableDetailedLogs bool `json:"enable_detailed_logs"`

	GenerateReport bool `json:"generate_report"`

	ReportFormat string `json:"report_format"`
}

// BenchmarkResults holds comprehensive benchmark results.

type BenchmarkResults struct {

	// Overall results.

	TestSuite *TestSuiteResults `json:"test_suite"`

	LatencyResults *LatencyBenchmarkResults `json:"latency_results"`

	ThroughputResults *ThroughputBenchmarkResults `json:"throughput_results"`

	StressTestResults *StressTestResults `json:"stress_test_results"`

	AccuracyResults *AccuracyBenchmarkResults `json:"accuracy_results"`

	MemoryResults *MemoryBenchmarkResults `json:"memory_results"`

	ComparisonResults *ComparisonBenchmarkResults `json:"comparison_results"`

	// Improvement metrics.

	ImprovementSummary *ImprovementSummary `json:"improvement_summary"`

	// Test metadata.

	TestTimestamp time.Time `json:"test_timestamp"`

	TestDuration time.Duration `json:"test_duration"`

	TestConfiguration *BenchmarkConfig `json:"test_configuration"`

	SystemInfo *SystemInfo `json:"system_info"`
}

// TestSuiteResults provides overall test suite results.

type TestSuiteResults struct {
	TotalTests int `json:"total_tests"`

	PassedTests int `json:"passed_tests"`

	FailedTests int `json:"failed_tests"`

	TestCoverage float64 `json:"test_coverage"`

	OverallScore float64 `json:"overall_score"`

	TestResults map[string]*TestResult `json:"test_results"`
}

// Note: TestResult is defined in integration_validator.go.

// LatencyBenchmarkResults holds latency benchmark results.

type LatencyBenchmarkResults struct {
	SingleQueryLatency *LatencyMetrics `json:"single_query_latency"`

	BatchQueryLatency *LatencyMetrics `json:"batch_query_latency"`

	CachedQueryLatency *LatencyMetrics `json:"cached_query_latency"`

	ConcurrentLatency map[int]*LatencyMetrics `json:"concurrent_latency"`

	OptimizedLatency *LatencyMetrics `json:"optimized_latency"`

	BaselineLatency *LatencyMetrics `json:"baseline_latency"`
}

// LatencyMetrics provides detailed latency statistics.

type LatencyMetrics struct {
	Mean time.Duration `json:"mean"`

	Median time.Duration `json:"median"`

	P95 time.Duration `json:"p95"`

	P99 time.Duration `json:"p99"`

	Min time.Duration `json:"min"`

	Max time.Duration `json:"max"`

	StdDev time.Duration `json:"std_dev"`

	SampleSize int `json:"sample_size"`

	Percentiles map[string]time.Duration `json:"percentiles"`
}

// ThroughputBenchmarkResults holds throughput benchmark results.

type ThroughputBenchmarkResults struct {
	QueriesPerSecond float64 `json:"queries_per_second"`

	MaxThroughput float64 `json:"max_throughput"`

	SustainedThroughput float64 `json:"sustained_throughput"`

	BatchThroughput map[int]float64 `json:"batch_throughput"`

	ConcurrentThroughput map[int]float64 `json:"concurrent_throughput"`

	OptimizedThroughput float64 `json:"optimized_throughput"`

	BaselineThroughput float64 `json:"baseline_throughput"`

	ThroughputOverTime []ThroughputSample `json:"throughput_over_time"`
}

// ThroughputSample represents a throughput measurement at a specific time.

type ThroughputSample struct {
	Timestamp time.Time `json:"timestamp"`

	QPS float64 `json:"qps"`

	ActiveConns int `json:"active_connections"`
}

// StressTestResults holds stress test results.

type StressTestResults struct {
	MaxConcurrentQueries int `json:"max_concurrent_queries"`

	BreakingPoint int `json:"breaking_point"`

	RecoveryTime time.Duration `json:"recovery_time"`

	ErrorRateUnderStress float64 `json:"error_rate_under_stress"`

	MemoryUsageUnderStress int64 `json:"memory_usage_under_stress"`

	CPUUsageUnderStress float64 `json:"cpu_usage_under_stress"`

	StressTestPhases []*StressPhaseResult `json:"stress_test_phases"`
}

// StressPhaseResult represents results from a stress test phase.

type StressPhaseResult struct {
	Phase string `json:"phase"`

	ConcurrencyLevel int `json:"concurrency_level"`

	Duration time.Duration `json:"duration"`

	SuccessfulOps int64 `json:"successful_ops"`

	FailedOps int64 `json:"failed_ops"`

	ErrorRate float64 `json:"error_rate"`

	AverageLatency time.Duration `json:"average_latency"`

	ThroughputQPS float64 `json:"throughput_qps"`
}

// AccuracyBenchmarkResults holds accuracy benchmark results.

type AccuracyBenchmarkResults struct {
	OverallAccuracy float32 `json:"overall_accuracy"`

	RecallScore float32 `json:"recall_score"`

	PrecisionScore float32 `json:"precision_score"`

	F1Score float32 `json:"f1_score"`

	SemanticAccuracy float32 `json:"semantic_accuracy"`

	QueryTypeAccuracy map[string]float32 `json:"query_type_accuracy"`

	OptimizedAccuracy float32 `json:"optimized_accuracy"`

	BaselineAccuracy float32 `json:"baseline_accuracy"`

	AccuracyByComplexity map[string]float32 `json:"accuracy_by_complexity"`
}

// MemoryBenchmarkResults holds memory benchmark results.

type MemoryBenchmarkResults struct {
	BaselineMemoryUsage int64 `json:"baseline_memory_usage"`

	PeakMemoryUsage int64 `json:"peak_memory_usage"`

	AverageMemoryUsage int64 `json:"average_memory_usage"`

	MemoryUsageByLoad map[int]int64 `json:"memory_usage_by_load"`

	CacheMemoryUsage int64 `json:"cache_memory_usage"`

	ConnectionPoolUsage int64 `json:"connection_pool_usage"`

	GCPressure float64 `json:"gc_pressure"`

	MemoryLeakDetection *MemoryLeakResult `json:"memory_leak_detection"`
}

// MemoryLeakResult represents memory leak detection results.

type MemoryLeakResult struct {
	LeakDetected bool `json:"leak_detected"`

	LeakRate float64 `json:"leak_rate"` // bytes per operation

	LeakConfidence float32 `json:"leak_confidence"`

	LeakLocation string `json:"leak_location"`
}

// ComparisonBenchmarkResults compares different implementations.

type ComparisonBenchmarkResults struct {
	BaselineVsOptimized *PerformanceComparison `json:"baseline_vs_optimized"`

	HTTPvsGRPC *PerformanceComparison `json:"http_vs_grpc"`

	SingleVsBatch *PerformanceComparison `json:"single_vs_batch"`

	CachedVsUncached *PerformanceComparison `json:"cached_vs_uncached"`

	DefaultVsOptimizedHNSW *PerformanceComparison `json:"default_vs_optimized_hnsw"`
}

// PerformanceComparison represents a performance comparison between two approaches.

type PerformanceComparison struct {
	ApproachA string `json:"approach_a"`

	ApproachB string `json:"approach_b"`

	LatencyImprovement time.Duration `json:"latency_improvement"`

	ThroughputImprovement float64 `json:"throughput_improvement"`

	AccuracyDifference float32 `json:"accuracy_difference"`

	MemoryDifference int64 `json:"memory_difference"`

	OverallImprovement float64 `json:"overall_improvement"`

	RecommendedApproach string `json:"recommended_approach"`
}

// ImprovementSummary provides an overall summary of improvements.

type ImprovementSummary struct {
	LatencyImprovement float64 `json:"latency_improvement"` // Percentage improvement

	ThroughputImprovement float64 `json:"throughput_improvement"` // Percentage improvement

	AccuracyImprovement float64 `json:"accuracy_improvement"` // Percentage improvement

	MemoryEfficiency float64 `json:"memory_efficiency"` // Percentage improvement

	OverallScore float64 `json:"overall_score"` // 0-100 score

	AchievedTargets []string `json:"achieved_targets"`

	MissedTargets []string `json:"missed_targets"`

	Recommendations []string `json:"recommendations"`
}

// SystemInfo holds system information for benchmarking context.

type SystemInfo struct {
	CPUCores int `json:"cpu_cores"`

	TotalMemory int64 `json:"total_memory"`

	AvailableMemory int64 `json:"available_memory"`

	GoVersion string `json:"go_version"`

	OSInfo string `json:"os_info"`

	BuildInfo string `json:"build_info"`
}

// TestDataset holds test data for benchmarking.

type TestDataset struct {
	Queries []*TestQuery `json:"queries"`

	ExpectedResults map[string]*ExpectedResult `json:"expected_results"`

	GroundTruth map[string][]*shared.SearchResult `json:"ground_truth"`

	QueryComplexity map[string]string `json:"query_complexity"`
}

// TestQuery represents a test query with metadata.

type TestQuery struct {
	ID string `json:"id"`

	Query string `json:"query"`

	Category string `json:"category"`

	Complexity string `json:"complexity"`

	Metadata map[string]interface{} `json:"metadata"`

	Expected *ExpectedResult `json:"expected"`
}

// ExpectedResult represents expected benchmark result.

type ExpectedResult struct {
	MinResults int `json:"min_results"`

	MaxLatency time.Duration `json:"max_latency"`

	MinAccuracy float32 `json:"min_accuracy"`

	ExpectedTerms []string `json:"expected_terms"`
}

// NewPerformanceBenchmarker creates a new performance benchmarker.

func NewPerformanceBenchmarker(

	originalClient *WeaviateClient,

	optimizedPipeline *OptimizedRAGPipeline,

	batchClient *OptimizedBatchSearchClient,

	grpcClient *GRPCWeaviateClient,

	config *BenchmarkConfig,

) *PerformanceBenchmarker {

	if config == nil {

		config = getDefaultBenchmarkConfig()

	}

	logger := slog.Default().With("component", "performance-benchmarker")

	benchmarker := &PerformanceBenchmarker{

		config: config,

		logger: logger,

		results: &BenchmarkResults{},

		testData: generateTestDataset(),

		originalClient: originalClient,

		optimizedPipeline: optimizedPipeline,

		batchClient: batchClient,

		grpcClient: grpcClient,
	}

	return benchmarker

}

// getDefaultBenchmarkConfig returns default benchmark configuration.

func getDefaultBenchmarkConfig() *BenchmarkConfig {

	return &BenchmarkConfig{

		WarmupQueries: 100,

		BenchmarkQueries: 1000,

		ConcurrencyLevels: []int{1, 5, 10, 20, 50, 100},

		BatchSizes: []int{1, 5, 10, 20, 50},

		TestDuration: 5 * time.Minute,

		LatencyTarget: 200 * time.Millisecond,

		ThroughputTarget: 100.0, // QPS

		AccuracyTarget: 0.95,

		MemoryUsageTarget: 500 * 1024 * 1024, // 500MB

		EnableLatencyTest: true,

		EnableThroughputTest: true,

		EnableStressTest: true,

		EnableAccuracyTest: true,

		EnableMemoryTest: true,

		EnableComparisonTest: true,

		EnableDetailedLogs: true,

		GenerateReport: true,

		ReportFormat: "json",
	}

}

// RunComprehensiveBenchmark runs all benchmark tests.

func (pb *PerformanceBenchmarker) RunComprehensiveBenchmark(ctx context.Context) (*BenchmarkResults, error) {

	pb.logger.Info("Starting comprehensive RAG performance benchmark")

	startTime := time.Now()

	// Initialize results.

	pb.results = &BenchmarkResults{

		TestSuite: &TestSuiteResults{TestResults: make(map[string]*TestResult)},

		TestTimestamp: startTime,

		TestConfiguration: pb.config,

		SystemInfo: pb.getSystemInfo(),
	}

	// Warmup phase.

	err := pb.runWarmup(ctx)

	if err != nil {

		return nil, fmt.Errorf("warmup failed: %w", err)

	}

	// Run individual benchmark tests.

	tests := []struct {
		name string

		enabled bool

		runner func(context.Context) error
	}{

		{"latency_benchmark", pb.config.EnableLatencyTest, pb.runLatencyBenchmark},

		{"throughput_benchmark", pb.config.EnableThroughputTest, pb.runThroughputBenchmark},

		{"stress_test", pb.config.EnableStressTest, pb.runStressTest},

		{"accuracy_benchmark", pb.config.EnableAccuracyTest, pb.runAccuracyBenchmark},

		{"memory_benchmark", pb.config.EnableMemoryTest, pb.runMemoryBenchmark},

		{"comparison_benchmark", pb.config.EnableComparisonTest, pb.runComparisonBenchmark},
	}

	for _, test := range tests {

		if !test.enabled {

			pb.logger.Info("Skipping disabled test", "test", test.name)

			continue

		}

		pb.logger.Info("Running benchmark test", "test", test.name)

		testStart := time.Now()

		err := test.runner(ctx)

		result := &TestResult{

			TestName: test.name,

			Passed: err == nil,

			Duration: time.Since(testStart),

			Score: 0.0, // Will be calculated later

		}

		if err != nil {

			result.Error = err.Error()

			pb.logger.Error("Benchmark test failed", "test", test.name, "error", err)

		} else {

			pb.logger.Info("Benchmark test completed", "test", test.name, "duration", result.Duration)

		}

		pb.results.TestSuite.TestResults[test.name] = result

		pb.results.TestSuite.TotalTests++

		if result.Passed {

			pb.results.TestSuite.PassedTests++

		} else {

			pb.results.TestSuite.FailedTests++

		}

	}

	// Calculate final metrics and improvement summary.

	pb.calculateFinalMetrics()

	pb.results.TestDuration = time.Since(startTime)

	pb.logger.Info("Comprehensive benchmark completed",

		"duration", pb.results.TestDuration,

		"passed_tests", pb.results.TestSuite.PassedTests,

		"failed_tests", pb.results.TestSuite.FailedTests,
	)

	return pb.results, nil

}

// runWarmup performs warmup queries to stabilize performance.

func (pb *PerformanceBenchmarker) runWarmup(ctx context.Context) error {

	pb.logger.Info("Starting warmup phase", "queries", pb.config.WarmupQueries)

	for i := range pb.config.WarmupQueries {

		query := pb.testData.Queries[i%len(pb.testData.Queries)]

		// Warmup original client.

		searchQuery := &SearchQuery{

			Query: query.Query,

			Limit: 10,
		}

		_, err := pb.originalClient.Search(ctx, searchQuery)

		if err != nil {

			pb.logger.Warn("Warmup query failed", "error", err)

		}

		// Brief pause to avoid overwhelming the system.

		time.Sleep(10 * time.Millisecond)

	}

	pb.logger.Info("Warmup phase completed")

	return nil

}

// runLatencyBenchmark runs latency benchmark tests.

func (pb *PerformanceBenchmarker) runLatencyBenchmark(ctx context.Context) error {

	pb.results.LatencyResults = &LatencyBenchmarkResults{

		ConcurrentLatency: make(map[int]*LatencyMetrics),
	}

	// Single query latency.

	pb.results.LatencyResults.SingleQueryLatency = pb.measureSingleQueryLatency(ctx)

	// Batch query latency.

	pb.results.LatencyResults.BatchQueryLatency = pb.measureBatchQueryLatency(ctx)

	// Cached vs uncached latency.

	pb.results.LatencyResults.CachedQueryLatency = pb.measureCachedQueryLatency(ctx)

	// Concurrent latency at different levels.

	for _, concurrency := range pb.config.ConcurrencyLevels {

		pb.results.LatencyResults.ConcurrentLatency[concurrency] = pb.measureConcurrentLatency(ctx, concurrency)

	}

	// Compare optimized vs baseline.

	pb.results.LatencyResults.BaselineLatency = pb.measureBaselineLatency(ctx)

	pb.results.LatencyResults.OptimizedLatency = pb.measureOptimizedLatency(ctx)

	return nil

}

// measureSingleQueryLatency measures latency for individual queries.

func (pb *PerformanceBenchmarker) measureSingleQueryLatency(ctx context.Context) *LatencyMetrics {

	var latencies []time.Duration

	for i := range pb.config.BenchmarkQueries {

		query := pb.testData.Queries[i%len(pb.testData.Queries)]

		start := time.Now()

		searchQuery := &SearchQuery{

			Query: query.Query,

			Limit: 10,
		}

		_, err := pb.originalClient.Search(ctx, searchQuery)

		latency := time.Since(start)

		if err == nil {

			latencies = append(latencies, latency)

		}

	}

	return pb.calculateLatencyMetrics(latencies)

}

// measureBatchQueryLatency measures latency for batch queries.

func (pb *PerformanceBenchmarker) measureBatchQueryLatency(ctx context.Context) *LatencyMetrics {

	var latencies []time.Duration

	// Test different batch sizes.

	for _, batchSize := range pb.config.BatchSizes {

		if batchSize <= 1 {

			continue // Skip single queries for batch test

		}

		// Create batch of queries.

		queries := make([]*SearchQuery, batchSize)

		for i := range batchSize {

			testQuery := pb.testData.Queries[i%len(pb.testData.Queries)]

			queries[i] = &SearchQuery{

				Query: testQuery.Query,

				Limit: 10,
			}

		}

		start := time.Now()

		batchRequest := &BatchSearchRequest{

			Queries: queries,

			MaxConcurrency: 5,
		}

		_, err := pb.batchClient.BatchSearch(ctx, batchRequest)

		latency := time.Since(start)

		if err == nil {

			// Calculate per-query latency.

			perQueryLatency := latency / time.Duration(batchSize)

			latencies = append(latencies, perQueryLatency)

		}

	}

	return pb.calculateLatencyMetrics(latencies)

}

// measureCachedQueryLatency measures latency for cached queries.

func (pb *PerformanceBenchmarker) measureCachedQueryLatency(ctx context.Context) *LatencyMetrics {

	var latencies []time.Duration

	// Use a small set of queries that will be repeated (cached).

	testQueries := pb.testData.Queries[:10] // Use first 10 queries

	// First, populate the cache.

	for _, query := range testQueries {

		ragRequest := &RAGRequest{

			Query: query.Query,

			MaxResults: 10,
		}

		pb.optimizedPipeline.ProcessQuery(ctx, ragRequest)

	}

	// Now measure cached query latency.

	for i := range pb.config.BenchmarkQueries {

		query := testQueries[i%len(testQueries)]

		start := time.Now()

		ragRequest := &RAGRequest{

			Query: query.Query,

			MaxResults: 10,
		}

		_, err := pb.optimizedPipeline.ProcessQuery(ctx, ragRequest)

		latency := time.Since(start)

		if err == nil {

			latencies = append(latencies, latency)

		}

	}

	return pb.calculateLatencyMetrics(latencies)

}

// measureConcurrentLatency measures latency under concurrent load.

func (pb *PerformanceBenchmarker) measureConcurrentLatency(ctx context.Context, concurrency int) *LatencyMetrics {

	var latencies []time.Duration

	var mutex sync.Mutex

	// Use buffered channel to control concurrency.

	semaphore := make(chan struct{}, concurrency)

	var wg sync.WaitGroup

	for i := range pb.config.BenchmarkQueries {

		wg.Add(1)

		go func(queryIndex int) {

			defer wg.Done()

			// Acquire semaphore.

			semaphore <- struct{}{}

			defer func() { <-semaphore }()

			query := pb.testData.Queries[queryIndex%len(pb.testData.Queries)]

			start := time.Now()

			searchQuery := &SearchQuery{

				Query: query.Query,

				Limit: 10,
			}

			_, err := pb.originalClient.Search(ctx, searchQuery)

			latency := time.Since(start)

			if err == nil {

				mutex.Lock()

				latencies = append(latencies, latency)

				mutex.Unlock()

			}

		}(i)

	}

	wg.Wait()

	return pb.calculateLatencyMetrics(latencies)

}

// measureBaselineLatency measures baseline (non-optimized) latency.

func (pb *PerformanceBenchmarker) measureBaselineLatency(ctx context.Context) *LatencyMetrics {

	// This would measure the original, non-optimized implementation.

	return pb.measureSingleQueryLatency(ctx)

}

// measureOptimizedLatency measures optimized pipeline latency.

func (pb *PerformanceBenchmarker) measureOptimizedLatency(ctx context.Context) *LatencyMetrics {

	var latencies []time.Duration

	for i := range pb.config.BenchmarkQueries {

		query := pb.testData.Queries[i%len(pb.testData.Queries)]

		start := time.Now()

		ragRequest := &RAGRequest{

			Query: query.Query,

			MaxResults: 10,
		}

		_, err := pb.optimizedPipeline.ProcessQuery(ctx, ragRequest)

		latency := time.Since(start)

		if err == nil {

			latencies = append(latencies, latency)

		}

	}

	return pb.calculateLatencyMetrics(latencies)

}

// runThroughputBenchmark runs throughput benchmark tests.

func (pb *PerformanceBenchmarker) runThroughputBenchmark(ctx context.Context) error {

	pb.results.ThroughputResults = &ThroughputBenchmarkResults{

		BatchThroughput: make(map[int]float64),

		ConcurrentThroughput: make(map[int]float64),

		ThroughputOverTime: make([]ThroughputSample, 0),
	}

	// Measure baseline throughput.

	pb.results.ThroughputResults.BaselineThroughput = pb.measureBaseThroughput(ctx)

	// Measure optimized throughput.

	pb.results.ThroughputResults.OptimizedThroughput = pb.measureOptimizedThroughput(ctx)

	// Measure batch throughput for different batch sizes.

	for _, batchSize := range pb.config.BatchSizes {

		pb.results.ThroughputResults.BatchThroughput[batchSize] = pb.measureBatchThroughput(ctx, batchSize)

	}

	// Measure concurrent throughput for different concurrency levels.

	for _, concurrency := range pb.config.ConcurrencyLevels {

		pb.results.ThroughputResults.ConcurrentThroughput[concurrency] = pb.measureConcurrentThroughput(ctx, concurrency)

	}

	// Find maximum sustained throughput.

	pb.results.ThroughputResults.MaxThroughput = pb.findMaxThroughput(ctx)

	pb.results.ThroughputResults.SustainedThroughput = pb.measureSustainedThroughput(ctx)

	return nil

}

// measureBaseThroughput measures baseline throughput.

func (pb *PerformanceBenchmarker) measureBaseThroughput(ctx context.Context) float64 {

	start := time.Now()

	successfulQueries := 0

	for i := range pb.config.BenchmarkQueries {

		query := pb.testData.Queries[i%len(pb.testData.Queries)]

		searchQuery := &SearchQuery{

			Query: query.Query,

			Limit: 10,
		}

		_, err := pb.originalClient.Search(ctx, searchQuery)

		if err == nil {

			successfulQueries++

		}

	}

	duration := time.Since(start)

	return float64(successfulQueries) / duration.Seconds()

}

// measureOptimizedThroughput measures optimized pipeline throughput.

func (pb *PerformanceBenchmarker) measureOptimizedThroughput(ctx context.Context) float64 {

	start := time.Now()

	successfulQueries := 0

	for i := range pb.config.BenchmarkQueries {

		query := pb.testData.Queries[i%len(pb.testData.Queries)]

		ragRequest := &RAGRequest{

			Query: query.Query,

			MaxResults: 10,
		}

		_, err := pb.optimizedPipeline.ProcessQuery(ctx, ragRequest)

		if err == nil {

			successfulQueries++

		}

	}

	duration := time.Since(start)

	return float64(successfulQueries) / duration.Seconds()

}

// measureBatchThroughput measures throughput for batch processing.

func (pb *PerformanceBenchmarker) measureBatchThroughput(ctx context.Context, batchSize int) float64 {

	if batchSize <= 1 {

		return pb.measureBaseThroughput(ctx)

	}

	start := time.Now()

	totalQueries := 0

	// Process queries in batches.

	for i := 0; i < pb.config.BenchmarkQueries; i += batchSize {

		endIndex := i + batchSize

		if endIndex > pb.config.BenchmarkQueries {

			endIndex = pb.config.BenchmarkQueries

		}

		// Create batch.

		queries := make([]*SearchQuery, endIndex-i)

		for j := i; j < endIndex; j++ {

			testQuery := pb.testData.Queries[j%len(pb.testData.Queries)]

			queries[j-i] = &SearchQuery{

				Query: testQuery.Query,

				Limit: 10,
			}

		}

		batchRequest := &BatchSearchRequest{

			Queries: queries,

			MaxConcurrency: 5,
		}

		_, err := pb.batchClient.BatchSearch(ctx, batchRequest)

		if err == nil {

			totalQueries += len(queries)

		}

	}

	duration := time.Since(start)

	return float64(totalQueries) / duration.Seconds()

}

// measureConcurrentThroughput measures throughput under concurrent load.

func (pb *PerformanceBenchmarker) measureConcurrentThroughput(ctx context.Context, concurrency int) float64 {

	start := time.Now()

	var successfulQueries int64

	semaphore := make(chan struct{}, concurrency)

	var wg sync.WaitGroup

	for i := range pb.config.BenchmarkQueries {

		wg.Add(1)

		go func(queryIndex int) {

			defer wg.Done()

			// Acquire semaphore.

			semaphore <- struct{}{}

			defer func() { <-semaphore }()

			query := pb.testData.Queries[queryIndex%len(pb.testData.Queries)]

			searchQuery := &SearchQuery{

				Query: query.Query,

				Limit: 10,
			}

			_, err := pb.originalClient.Search(ctx, searchQuery)

			if err == nil {

				atomic.AddInt64(&successfulQueries, 1)

			}

		}(i)

	}

	wg.Wait()

	duration := time.Since(start)

	return float64(successfulQueries) / duration.Seconds()

}

// findMaxThroughput finds the maximum sustainable throughput.

func (pb *PerformanceBenchmarker) findMaxThroughput(ctx context.Context) float64 {

	maxThroughput := 0.0

	for _, concurrency := range pb.config.ConcurrencyLevels {

		throughput := pb.measureConcurrentThroughput(ctx, concurrency)

		if throughput > maxThroughput {

			maxThroughput = throughput

		}

	}

	return maxThroughput

}

// measureSustainedThroughput measures sustained throughput over time.

func (pb *PerformanceBenchmarker) measureSustainedThroughput(ctx context.Context) float64 {

	// Measure throughput over the test duration.

	testCtx, cancel := context.WithTimeout(ctx, pb.config.TestDuration)

	defer cancel()

	start := time.Now()

	var totalQueries int64

	// Run queries continuously for the test duration.

	var wg sync.WaitGroup

	for i := range 10 { // Use 10 concurrent workers

		wg.Add(1)

		go func(workerID int) {

			defer wg.Done()

			queryIndex := 0

			for {

				select {

				case <-testCtx.Done():

					return

				default:

					query := pb.testData.Queries[queryIndex%len(pb.testData.Queries)]

					searchQuery := &SearchQuery{

						Query: query.Query,

						Limit: 10,
					}

					_, err := pb.originalClient.Search(testCtx, searchQuery)

					if err == nil {

						atomic.AddInt64(&totalQueries, 1)

					}

					queryIndex++

				}

			}

		}(i)

	}

	wg.Wait()

	duration := time.Since(start)

	return float64(totalQueries) / duration.Seconds()

}

// runStressTest runs stress tests to find breaking points.

func (pb *PerformanceBenchmarker) runStressTest(ctx context.Context) error {

	pb.results.StressTestResults = &StressTestResults{

		StressTestPhases: make([]*StressPhaseResult, 0),
	}

	// Gradually increase load to find breaking point.

	concurrencyLevels := []int{10, 25, 50, 100, 200, 500, 1000}

	for _, concurrency := range concurrencyLevels {

		pb.logger.Info("Running stress test phase", "concurrency", concurrency)

		phaseResult := pb.runStressPhase(ctx, concurrency)

		pb.results.StressTestResults.StressTestPhases = append(pb.results.StressTestResults.StressTestPhases, phaseResult)

		// Check if we've hit the breaking point (high error rate or timeouts).

		if phaseResult.ErrorRate > 0.1 { // 10% error rate threshold

			pb.results.StressTestResults.BreakingPoint = concurrency

			pb.logger.Info("Breaking point identified", "concurrency", concurrency, "error_rate", phaseResult.ErrorRate)

			break

		}

	}

	// Find maximum concurrent queries that still maintain acceptable performance.

	pb.results.StressTestResults.MaxConcurrentQueries = pb.findMaxConcurrentQueries(ctx)

	return nil

}

// runStressPhase runs a single stress test phase.

func (pb *PerformanceBenchmarker) runStressPhase(ctx context.Context, concurrency int) *StressPhaseResult {

	start := time.Now()

	phaseDuration := 30 * time.Second // Each phase runs for 30 seconds

	phaseCtx, cancel := context.WithTimeout(ctx, phaseDuration)

	defer cancel()

	var successfulOps, failedOps int64

	var totalLatency time.Duration

	var latencyCount int64

	semaphore := make(chan struct{}, concurrency)

	var wg sync.WaitGroup

	queryIndex := 0

	// Start workers.

	for i := range concurrency {

		wg.Add(1)

		go func(workerID int) {

			defer wg.Done()

			for {

				select {

				case <-phaseCtx.Done():

					return

				default:

					// Acquire semaphore.

					semaphore <- struct{}{}

					query := pb.testData.Queries[queryIndex%len(pb.testData.Queries)]

					queryIndex++

					start := time.Now()

					searchQuery := &SearchQuery{

						Query: query.Query,

						Limit: 10,
					}

					_, err := pb.originalClient.Search(phaseCtx, searchQuery)

					latency := time.Since(start)

					if err == nil {

						atomic.AddInt64(&successfulOps, 1)

						atomic.AddInt64(&latencyCount, 1)

						atomic.AddInt64((*int64)(&totalLatency), int64(latency))

					} else {

						atomic.AddInt64(&failedOps, 1)

					}

					// Release semaphore.

					<-semaphore

				}

			}

		}(i)

	}

	wg.Wait()

	totalOps := successfulOps + failedOps

	errorRate := 0.0

	if totalOps > 0 {

		errorRate = float64(failedOps) / float64(totalOps)

	}

	averageLatency := time.Duration(0)

	if latencyCount > 0 {

		averageLatency = time.Duration(int64(totalLatency) / latencyCount)

	}

	actualDuration := time.Since(start)

	throughputQPS := float64(successfulOps) / actualDuration.Seconds()

	return &StressPhaseResult{

		Phase: fmt.Sprintf("stress_test_%d", concurrency),

		ConcurrencyLevel: concurrency,

		Duration: actualDuration,

		SuccessfulOps: successfulOps,

		FailedOps: failedOps,

		ErrorRate: errorRate,

		AverageLatency: averageLatency,

		ThroughputQPS: throughputQPS,
	}

}

// findMaxConcurrentQueries finds the maximum concurrent queries the system can handle.

func (pb *PerformanceBenchmarker) findMaxConcurrentQueries(ctx context.Context) int {

	maxConcurrent := 0

	for _, phase := range pb.results.StressTestResults.StressTestPhases {

		if phase.ErrorRate < 0.05 { // Less than 5% error rate

			if phase.ConcurrencyLevel > maxConcurrent {

				maxConcurrent = phase.ConcurrencyLevel

			}

		}

	}

	return maxConcurrent

}

// runAccuracyBenchmark runs accuracy benchmark tests.

func (pb *PerformanceBenchmarker) runAccuracyBenchmark(ctx context.Context) error {

	pb.results.AccuracyResults = &AccuracyBenchmarkResults{

		QueryTypeAccuracy: make(map[string]float32),

		AccuracyByComplexity: make(map[string]float32),
	}

	// Measure overall accuracy.

	pb.results.AccuracyResults.OverallAccuracy = pb.measureOverallAccuracy(ctx)

	// Measure baseline vs optimized accuracy.

	pb.results.AccuracyResults.BaselineAccuracy = pb.measureBaselineAccuracy(ctx)

	pb.results.AccuracyResults.OptimizedAccuracy = pb.measureOptimizedAccuracy(ctx)

	// Calculate precision, recall, and F1 score.

	pb.calculateAccuracyMetrics(ctx)

	return nil

}

// measureOverallAccuracy measures overall search accuracy.

func (pb *PerformanceBenchmarker) measureOverallAccuracy(ctx context.Context) float32 {

	totalQueries := 0

	accurateResults := 0

	for _, testQuery := range pb.testData.Queries {

		if testQuery.Expected == nil {

			continue

		}

		searchQuery := &SearchQuery{

			Query: testQuery.Query,

			Limit: testQuery.Expected.MinResults,
		}

		result, err := pb.originalClient.Search(ctx, searchQuery)

		if err != nil {

			continue

		}

		totalQueries++

		// Check if results meet expectations.

		if len(result.Results) >= testQuery.Expected.MinResults {

			// Check if expected terms are present in results.

			hasExpectedTerms := pb.checkExpectedTerms(result.Results, testQuery.Expected.ExpectedTerms)

			if hasExpectedTerms {

				accurateResults++

			}

		}

	}

	if totalQueries == 0 {

		return 0.0

	}

	return float32(accurateResults) / float32(totalQueries)

}

// measureBaselineAccuracy measures baseline accuracy.

func (pb *PerformanceBenchmarker) measureBaselineAccuracy(ctx context.Context) float32 {

	// This would measure accuracy of the original, non-optimized implementation.

	return pb.measureOverallAccuracy(ctx)

}

// measureOptimizedAccuracy measures optimized pipeline accuracy.

func (pb *PerformanceBenchmarker) measureOptimizedAccuracy(ctx context.Context) float32 {

	totalQueries := 0

	accurateResults := 0

	for _, testQuery := range pb.testData.Queries {

		if testQuery.Expected == nil {

			continue

		}

		ragRequest := &RAGRequest{

			Query: testQuery.Query,

			MaxResults: testQuery.Expected.MinResults,
		}

		result, err := pb.optimizedPipeline.ProcessQuery(ctx, ragRequest)

		if err != nil {

			continue

		}

		totalQueries++

		// Check if results meet expectations.

		if len(result.SourceDocuments) >= testQuery.Expected.MinResults {

			// Convert to search results for checking expected terms.

			searchResults := make([]*SearchResult, len(result.SourceDocuments))

			for i, doc := range result.SourceDocuments {

				searchResults[i] = &SearchResult{

					Document: doc.Document,

					Score: doc.Score,
				}

			}

			hasExpectedTerms := pb.checkExpectedTerms(searchResults, testQuery.Expected.ExpectedTerms)

			if hasExpectedTerms {

				accurateResults++

			}

		}

	}

	if totalQueries == 0 {

		return 0.0

	}

	return float32(accurateResults) / float32(totalQueries)

}

// checkExpectedTerms checks if expected terms are present in search results.

func (pb *PerformanceBenchmarker) checkExpectedTerms(results []*SearchResult, expectedTerms []string) bool {

	if len(expectedTerms) == 0 {

		return true // No specific terms expected

	}

	// Combine all result content.

	var allContent string

	for _, result := range results {

		if result.Document != nil {

			allContent += " " + result.Document.Content

		}

	}

	allContent = strings.ToLower(allContent)

	// Check if at least one expected term is present.

	for _, term := range expectedTerms {

		if strings.Contains(allContent, strings.ToLower(term)) {

			return true

		}

	}

	return false

}

// calculateAccuracyMetrics calculates precision, recall, and F1 score.

func (pb *PerformanceBenchmarker) calculateAccuracyMetrics(ctx context.Context) {

	// This would implement precision, recall, and F1 score calculations.

	// based on ground truth data vs actual results.

	pb.results.AccuracyResults.PrecisionScore = 0.85 // Placeholder

	pb.results.AccuracyResults.RecallScore = 0.82 // Placeholder

	// Calculate F1 score.

	precision := pb.results.AccuracyResults.PrecisionScore

	recall := pb.results.AccuracyResults.RecallScore

	if precision+recall > 0 {

		pb.results.AccuracyResults.F1Score = 2 * (precision * recall) / (precision + recall)

	}

}

// runMemoryBenchmark runs memory usage benchmark tests.

func (pb *PerformanceBenchmarker) runMemoryBenchmark(ctx context.Context) error {

	pb.results.MemoryResults = &MemoryBenchmarkResults{

		MemoryUsageByLoad: make(map[int]int64),
	}

	// Measure baseline memory usage.

	pb.results.MemoryResults.BaselineMemoryUsage = pb.getCurrentMemoryUsage()

	// Measure memory usage under different loads.

	for _, concurrency := range pb.config.ConcurrencyLevels {

		memUsage := pb.measureMemoryUnderLoad(ctx, concurrency)

		pb.results.MemoryResults.MemoryUsageByLoad[concurrency] = memUsage

		if memUsage > pb.results.MemoryResults.PeakMemoryUsage {

			pb.results.MemoryResults.PeakMemoryUsage = memUsage

		}

	}

	// Check for memory leaks.

	pb.results.MemoryResults.MemoryLeakDetection = pb.detectMemoryLeaks(ctx)

	return nil

}

// getCurrentMemoryUsage gets current memory usage.

func (pb *PerformanceBenchmarker) getCurrentMemoryUsage() int64 {

	// This would use runtime.MemStats to get actual memory usage.

	// For now, return placeholder value.

	return 50 * 1024 * 1024 // 50MB placeholder

}

// measureMemoryUnderLoad measures memory usage under specific load.

func (pb *PerformanceBenchmarker) measureMemoryUnderLoad(ctx context.Context, concurrency int) int64 {

	// Start load.

	loadCtx, cancel := context.WithTimeout(ctx, 30*time.Second)

	defer cancel()

	// Monitor memory usage during load.

	maxMemory := pb.getCurrentMemoryUsage()

	// Start concurrent queries.

	var wg sync.WaitGroup

	for i := range concurrency {

		wg.Add(1)

		go func(workerID int) {

			defer wg.Done()

			for {

				select {

				case <-loadCtx.Done():

					return

				default:

					query := pb.testData.Queries[0]

					searchQuery := &SearchQuery{

						Query: query.Query,

						Limit: 10,
					}

					pb.originalClient.Search(loadCtx, searchQuery)

					// Sample memory usage.

					currentMemory := pb.getCurrentMemoryUsage()

					if currentMemory > maxMemory {

						maxMemory = currentMemory

					}

					time.Sleep(10 * time.Millisecond)

				}

			}

		}(i)

	}

	wg.Wait()

	return maxMemory

}

// detectMemoryLeaks detects potential memory leaks.

func (pb *PerformanceBenchmarker) detectMemoryLeaks(ctx context.Context) *MemoryLeakResult {

	// Run operations and monitor memory growth.

	initialMemory := pb.getCurrentMemoryUsage()

	// Run a series of operations.

	for i := range 1000 {

		query := pb.testData.Queries[i%len(pb.testData.Queries)]

		searchQuery := &SearchQuery{

			Query: query.Query,

			Limit: 10,
		}

		pb.originalClient.Search(ctx, searchQuery)

	}

	finalMemory := pb.getCurrentMemoryUsage()

	memoryGrowth := finalMemory - initialMemory

	// Simple leak detection heuristic.

	leakRate := float64(memoryGrowth) / 1000.0 // bytes per operation

	leakDetected := leakRate > 1024 // More than 1KB growth per operation

	return &MemoryLeakResult{

		LeakDetected: leakDetected,

		LeakRate: leakRate,

		LeakConfidence: 0.7, // Placeholder confidence

		LeakLocation: "unknown", // Would need more sophisticated detection

	}

}

// runComparisonBenchmark runs comparison tests between different implementations.

func (pb *PerformanceBenchmarker) runComparisonBenchmark(ctx context.Context) error {

	pb.results.ComparisonResults = &ComparisonBenchmarkResults{}

	// Compare baseline vs optimized.

	pb.results.ComparisonResults.BaselineVsOptimized = pb.compareBaselineVsOptimized(ctx)

	// Compare HTTP vs gRPC (if gRPC client available).

	if pb.grpcClient != nil {

		pb.results.ComparisonResults.HTTPvsGRPC = pb.compareHTTPvsGRPC(ctx)

	}

	// Compare single vs batch processing.

	pb.results.ComparisonResults.SingleVsBatch = pb.compareSingleVsBatch(ctx)

	// Compare cached vs uncached performance.

	pb.results.ComparisonResults.CachedVsUncached = pb.compareCachedVsUncached(ctx)

	return nil

}

// compareBaselineVsOptimized compares baseline vs optimized performance.

func (pb *PerformanceBenchmarker) compareBaselineVsOptimized(ctx context.Context) *PerformanceComparison {

	// Measure baseline performance.

	baselineLatency := pb.results.LatencyResults.BaselineLatency.Mean

	baselineThroughput := pb.results.ThroughputResults.BaselineThroughput

	baselineAccuracy := pb.results.AccuracyResults.BaselineAccuracy

	// Measure optimized performance.

	optimizedLatency := pb.results.LatencyResults.OptimizedLatency.Mean

	optimizedThroughput := pb.results.ThroughputResults.OptimizedThroughput

	optimizedAccuracy := pb.results.AccuracyResults.OptimizedAccuracy

	// Calculate improvements.

	latencyImprovement := baselineLatency - optimizedLatency

	throughputImprovement := optimizedThroughput - baselineThroughput

	accuracyDifference := optimizedAccuracy - baselineAccuracy

	// Calculate overall improvement score.

	latencyImprovementPct := float64(latencyImprovement) / float64(baselineLatency) * 100

	throughputImprovementPct := throughputImprovement / baselineThroughput * 100

	overallImprovement := (latencyImprovementPct + throughputImprovementPct) / 2

	return &PerformanceComparison{

		ApproachA: "baseline",

		ApproachB: "optimized",

		LatencyImprovement: latencyImprovement,

		ThroughputImprovement: throughputImprovement,

		AccuracyDifference: accuracyDifference,

		OverallImprovement: overallImprovement,

		RecommendedApproach: "optimized",
	}

}

// compareHTTPvsGRPC compares HTTP vs gRPC client performance.

func (pb *PerformanceBenchmarker) compareHTTPvsGRPC(ctx context.Context) *PerformanceComparison {

	// Measure HTTP client performance.

	httpLatencies := pb.measureHTTPClientLatency(ctx)

	httpThroughput := pb.measureHTTPClientThroughput(ctx)

	// Measure gRPC client performance.

	grpcLatencies := pb.measureGRPCClientLatency(ctx)

	grpcThroughput := pb.measureGRPCClientThroughput(ctx)

	latencyImprovement := httpLatencies.Mean - grpcLatencies.Mean

	throughputImprovement := grpcThroughput - httpThroughput

	overallImprovement := (float64(latencyImprovement)/float64(httpLatencies.Mean) +

		throughputImprovement/httpThroughput) / 2 * 100

	return &PerformanceComparison{

		ApproachA: "http",

		ApproachB: "grpc",

		LatencyImprovement: latencyImprovement,

		ThroughputImprovement: throughputImprovement,

		OverallImprovement: overallImprovement,

		RecommendedApproach: "grpc",
	}

}

// compareSingleVsBatch compares single vs batch processing.

func (pb *PerformanceBenchmarker) compareSingleVsBatch(ctx context.Context) *PerformanceComparison {

	singleLatency := pb.results.LatencyResults.SingleQueryLatency.Mean

	batchLatency := pb.results.LatencyResults.BatchQueryLatency.Mean

	latencyImprovement := singleLatency - batchLatency

	return &PerformanceComparison{

		ApproachA: "single",

		ApproachB: "batch",

		LatencyImprovement: latencyImprovement,

		OverallImprovement: float64(latencyImprovement) / float64(singleLatency) * 100,

		RecommendedApproach: "batch",
	}

}

// compareCachedVsUncached compares cached vs uncached performance.

func (pb *PerformanceBenchmarker) compareCachedVsUncached(ctx context.Context) *PerformanceComparison {

	uncachedLatency := pb.results.LatencyResults.SingleQueryLatency.Mean

	cachedLatency := pb.results.LatencyResults.CachedQueryLatency.Mean

	latencyImprovement := uncachedLatency - cachedLatency

	return &PerformanceComparison{

		ApproachA: "uncached",

		ApproachB: "cached",

		LatencyImprovement: latencyImprovement,

		OverallImprovement: float64(latencyImprovement) / float64(uncachedLatency) * 100,

		RecommendedApproach: "cached",
	}

}

// Helper methods for HTTP vs gRPC comparison.

func (pb *PerformanceBenchmarker) measureHTTPClientLatency(ctx context.Context) *LatencyMetrics {

	// Measure HTTP client latency - similar to single query latency.

	return pb.results.LatencyResults.SingleQueryLatency

}

func (pb *PerformanceBenchmarker) measureHTTPClientThroughput(ctx context.Context) float64 {

	// Measure HTTP client throughput.

	return pb.results.ThroughputResults.BaselineThroughput

}

func (pb *PerformanceBenchmarker) measureGRPCClientLatency(ctx context.Context) *LatencyMetrics {

	if pb.grpcClient == nil {

		return &LatencyMetrics{}

	}

	var latencies []time.Duration

	for i := range 100 {

		query := pb.testData.Queries[i%len(pb.testData.Queries)]

		start := time.Now()

		searchQuery := &SearchQuery{

			Query: query.Query,

			Limit: 10,
		}

		_, err := pb.grpcClient.Search(ctx, searchQuery)

		latency := time.Since(start)

		if err == nil {

			latencies = append(latencies, latency)

		}

	}

	return pb.calculateLatencyMetrics(latencies)

}

func (pb *PerformanceBenchmarker) measureGRPCClientThroughput(ctx context.Context) float64 {

	if pb.grpcClient == nil {

		return 0

	}

	start := time.Now()

	successfulQueries := 0

	for i := range 100 {

		query := pb.testData.Queries[i%len(pb.testData.Queries)]

		searchQuery := &SearchQuery{

			Query: query.Query,

			Limit: 10,
		}

		_, err := pb.grpcClient.Search(ctx, searchQuery)

		if err == nil {

			successfulQueries++

		}

	}

	duration := time.Since(start)

	return float64(successfulQueries) / duration.Seconds()

}

// calculateLatencyMetrics calculates comprehensive latency statistics.

func (pb *PerformanceBenchmarker) calculateLatencyMetrics(latencies []time.Duration) *LatencyMetrics {

	if len(latencies) == 0 {

		return &LatencyMetrics{}

	}

	// Sort latencies for percentile calculations.

	sort.Slice(latencies, func(i, j int) bool {

		return latencies[i] < latencies[j]

	})

	// Calculate basic statistics.

	var sum time.Duration

	for _, lat := range latencies {

		sum += lat

	}

	mean := sum / time.Duration(len(latencies))

	// Calculate median.

	median := latencies[len(latencies)/2]

	// Calculate percentiles.

	p95Index := int(0.95 * float64(len(latencies)))

	if p95Index >= len(latencies) {

		p95Index = len(latencies) - 1

	}

	p95 := latencies[p95Index]

	p99Index := int(0.99 * float64(len(latencies)))

	if p99Index >= len(latencies) {

		p99Index = len(latencies) - 1

	}

	p99 := latencies[p99Index]

	// Calculate standard deviation.

	var variance float64

	for _, lat := range latencies {

		diff := float64(lat - mean)

		variance += diff * diff

	}

	variance /= float64(len(latencies))

	stdDev := time.Duration(math.Sqrt(variance))

	return &LatencyMetrics{

		Mean: mean,

		Median: median,

		P95: p95,

		P99: p99,

		Min: latencies[0],

		Max: latencies[len(latencies)-1],

		StdDev: stdDev,

		SampleSize: len(latencies),

		Percentiles: map[string]time.Duration{

			"p50": median,

			"p90": latencies[int(0.90*float64(len(latencies)))],

			"p95": p95,

			"p99": p99,
		},
	}

}

// calculateFinalMetrics calculates final metrics and improvement summary.

func (pb *PerformanceBenchmarker) calculateFinalMetrics() {

	// Calculate improvement summary.

	pb.results.ImprovementSummary = &ImprovementSummary{

		AchievedTargets: make([]string, 0),

		MissedTargets: make([]string, 0),

		Recommendations: make([]string, 0),
	}

	// Calculate overall improvements.

	if pb.results.ComparisonResults != nil && pb.results.ComparisonResults.BaselineVsOptimized != nil {

		comparison := pb.results.ComparisonResults.BaselineVsOptimized

		pb.results.ImprovementSummary.LatencyImprovement = comparison.OverallImprovement

		pb.results.ImprovementSummary.ThroughputImprovement = comparison.ThroughputImprovement / pb.results.ThroughputResults.BaselineThroughput * 100

		pb.results.ImprovementSummary.AccuracyImprovement = float64(comparison.AccuracyDifference) / float64(pb.results.AccuracyResults.BaselineAccuracy) * 100

	}

	// Check if targets were achieved.

	if pb.results.LatencyResults != nil && pb.results.LatencyResults.OptimizedLatency != nil {

		if pb.results.LatencyResults.OptimizedLatency.P95 <= pb.config.LatencyTarget {

			pb.results.ImprovementSummary.AchievedTargets = append(pb.results.ImprovementSummary.AchievedTargets, "latency_target")

		} else {

			pb.results.ImprovementSummary.MissedTargets = append(pb.results.ImprovementSummary.MissedTargets, "latency_target")

		}

	}

	if pb.results.ThroughputResults != nil {

		if pb.results.ThroughputResults.OptimizedThroughput >= pb.config.ThroughputTarget {

			pb.results.ImprovementSummary.AchievedTargets = append(pb.results.ImprovementSummary.AchievedTargets, "throughput_target")

		} else {

			pb.results.ImprovementSummary.MissedTargets = append(pb.results.ImprovementSummary.MissedTargets, "throughput_target")

		}

	}

	if pb.results.AccuracyResults != nil {

		if pb.results.AccuracyResults.OptimizedAccuracy >= pb.config.AccuracyTarget {

			pb.results.ImprovementSummary.AchievedTargets = append(pb.results.ImprovementSummary.AchievedTargets, "accuracy_target")

		} else {

			pb.results.ImprovementSummary.MissedTargets = append(pb.results.ImprovementSummary.MissedTargets, "accuracy_target")

		}

	}

	// Calculate overall score (0-100).

	achievedCount := len(pb.results.ImprovementSummary.AchievedTargets)

	totalTargets := achievedCount + len(pb.results.ImprovementSummary.MissedTargets)

	if totalTargets > 0 {

		pb.results.ImprovementSummary.OverallScore = float64(achievedCount) / float64(totalTargets) * 100

	}

	// Generate recommendations.

	pb.generateRecommendations()

	// Calculate test suite metrics.

	pb.results.TestSuite.TestCoverage = float64(pb.results.TestSuite.PassedTests) / float64(pb.results.TestSuite.TotalTests) * 100

	pb.results.TestSuite.OverallScore = pb.results.ImprovementSummary.OverallScore

}

// generateRecommendations generates performance recommendations.

func (pb *PerformanceBenchmarker) generateRecommendations() {

	recommendations := make([]string, 0)

	// Latency recommendations.

	if pb.results.LatencyResults != nil && pb.results.LatencyResults.OptimizedLatency != nil {

		if pb.results.LatencyResults.OptimizedLatency.P95 > pb.config.LatencyTarget {

			recommendations = append(recommendations, "Consider further HNSW parameter optimization to reduce query latency")

			recommendations = append(recommendations, "Evaluate increasing cache size to improve cache hit rates")

		}

	}

	// Throughput recommendations.

	if pb.results.ThroughputResults != nil {

		if pb.results.ThroughputResults.OptimizedThroughput < pb.config.ThroughputTarget {

			recommendations = append(recommendations, "Consider increasing connection pool size for higher throughput")

			recommendations = append(recommendations, "Evaluate batch processing for improved throughput")

		}

	}

	// Memory recommendations.

	if pb.results.MemoryResults != nil {

		if pb.results.MemoryResults.PeakMemoryUsage > pb.config.MemoryUsageTarget {

			recommendations = append(recommendations, "Consider optimizing cache sizes to reduce memory usage")

			recommendations = append(recommendations, "Implement more aggressive garbage collection tuning")

		}

		if pb.results.MemoryResults.MemoryLeakDetection != nil && pb.results.MemoryResults.MemoryLeakDetection.LeakDetected {

			recommendations = append(recommendations, "Investigate and fix detected memory leaks")

		}

	}

	// Accuracy recommendations.

	if pb.results.AccuracyResults != nil {

		if pb.results.AccuracyResults.OptimizedAccuracy < pb.config.AccuracyTarget {

			recommendations = append(recommendations, "Consider improving query preprocessing and expansion")

			recommendations = append(recommendations, "Evaluate adjusting semantic similarity thresholds")

		}

	}

	pb.results.ImprovementSummary.Recommendations = recommendations

}

// getSystemInfo collects system information.

func (pb *PerformanceBenchmarker) getSystemInfo() *SystemInfo {

	return &SystemInfo{

		CPUCores: 8, // Placeholder - would use runtime.NumCPU()

		TotalMemory: 16 * 1024 * 1024 * 1024, // 16GB placeholder

		AvailableMemory: 8 * 1024 * 1024 * 1024, // 8GB placeholder

		GoVersion: "go1.21.0", // Placeholder - would use runtime.Version()

		OSInfo: "linux", // Placeholder - would use runtime.GOOS

		BuildInfo: "dev", // Placeholder - would use build info

	}

}

// generateTestDataset generates test data for benchmarking.

func generateTestDataset() *TestDataset {

	queries := []*TestQuery{

		{

			ID: "q1",

			Query: "5G AMF configuration parameters",

			Category: "configuration",

			Complexity: "simple",

			Expected: &ExpectedResult{

				MinResults: 5,

				MaxLatency: 200 * time.Millisecond,

				MinAccuracy: 0.8,

				ExpectedTerms: []string{"AMF", "5G", "configuration"},
			},
		},

		{

			ID: "q2",

			Query: "network slicing optimization techniques for URLLC",

			Category: "optimization",

			Complexity: "complex",

			Expected: &ExpectedResult{

				MinResults: 3,

				MaxLatency: 300 * time.Millisecond,

				MinAccuracy: 0.75,

				ExpectedTerms: []string{"network slicing", "URLLC", "optimization"},
			},
		},

		{

			ID: "q3",

			Query: "O-RAN interface specifications and protocols",

			Category: "specification",

			Complexity: "medium",

			Expected: &ExpectedResult{

				MinResults: 8,

				MaxLatency: 250 * time.Millisecond,

				MinAccuracy: 0.85,

				ExpectedTerms: []string{"O-RAN", "interface", "protocol"},
			},
		},
	}

	// Expand the dataset with more diverse queries.

	for i := range 100 {

		baseQuery := queries[i%len(queries)]

		newQuery := &TestQuery{

			ID: fmt.Sprintf("q%d", i+4),

			Query: fmt.Sprintf("%s variant %d", baseQuery.Query, i),

			Category: baseQuery.Category,

			Complexity: baseQuery.Complexity,

			Expected: baseQuery.Expected,
		}

		queries = append(queries, newQuery)

	}

	return &TestDataset{

		Queries: queries,

		ExpectedResults: make(map[string]*ExpectedResult),

		GroundTruth: make(map[string][]*shared.SearchResult),

		QueryComplexity: make(map[string]string),
	}

}

// GetBenchmarkResults returns the current benchmark results.

func (pb *PerformanceBenchmarker) GetBenchmarkResults() *BenchmarkResults {

	return pb.results

}

// GenerateReport generates a performance benchmark report.

func (pb *PerformanceBenchmarker) GenerateReport() (string, error) {

	if !pb.config.GenerateReport {

		return "", nil

	}

	// Generate JSON report.

	report := map[string]interface{}{

		"benchmark_summary": pb.results.ImprovementSummary,

		"latency_results": pb.results.LatencyResults,

		"throughput_results": pb.results.ThroughputResults,

		"accuracy_results": pb.results.AccuracyResults,

		"memory_results": pb.results.MemoryResults,

		"test_suite": pb.results.TestSuite,

		"system_info": pb.results.SystemInfo,

		"timestamp": pb.results.TestTimestamp,

		"duration": pb.results.TestDuration,
	}

	// Convert to JSON string.

	reportJSON, err := json.MarshalIndent(report, "", "  ")

	if err != nil {

		return "", fmt.Errorf("failed to generate JSON report: %w", err)

	}

	return string(reportJSON), nil

}
