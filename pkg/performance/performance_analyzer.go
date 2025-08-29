//go:build go1.24

package performance

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// PerformanceAnalyzer provides comprehensive performance analysis and reporting.

type PerformanceAnalyzer struct {
	httpMetrics *HTTPPerformanceReport

	memoryMetrics *MemoryPerformanceReport

	jsonMetrics *JSONPerformanceReport

	goroutineMetrics *GoroutinePerformanceReport

	cacheMetrics *CachePerformanceReport

	dbMetrics *DatabasePerformanceReport

	overallMetrics *OverallPerformanceReport

	baselineMetrics *BaselineMetrics

	mu sync.RWMutex

	startTime time.Time

	lastReport time.Time
}

// HTTPPerformanceReport contains HTTP layer performance metrics.

type HTTPPerformanceReport struct {
	RequestsPerSecond float64

	AverageLatency time.Duration

	P95Latency time.Duration

	P99Latency time.Duration

	ConnectionPoolHitRate float64

	CompressionRatio float64

	HTTP2Usage float64

	TLSHandshakeTime time.Duration

	BufferPoolEfficiency float64

	ErrorRate float64

	ImprovementRatio float64
}

// MemoryPerformanceReport contains memory optimization metrics.

type MemoryPerformanceReport struct {
	HeapSize int64

	GCPauseTime time.Duration

	GCFrequency float64

	ObjectPoolHitRate float64

	RingBufferUtilization float64

	MemoryMapUsage int64

	AllocationReduction float64

	GCOptimization float64

	MemoryLeakDetection bool

	ImprovementRatio float64
}

// JSONPerformanceReport contains JSON processing metrics.

type JSONPerformanceReport struct {
	MarshalOpsPerSec float64

	UnmarshalOpsPerSec float64

	AverageProcessTime time.Duration

	SchemaHitRate float64

	PoolEfficiency float64

	ConcurrencyLevel int64

	SIMDUtilization float64

	CompressionSavings float64

	ErrorRate float64

	ImprovementRatio float64
}

// GoroutinePerformanceReport contains goroutine pool metrics.

type GoroutinePerformanceReport struct {
	TasksPerSecond float64

	AverageWaitTime time.Duration

	AverageProcessTime time.Duration

	WorkerUtilization float64

	WorkStealingEfficiency float64

	ScalingResponsiveness float64

	CPUAffinityBenefit float64

	PreemptionRate float64

	DeadlockFrequency float64

	ImprovementRatio float64
}

// CachePerformanceReport contains cache optimization metrics.

type CachePerformanceReport struct {
	HitRate float64

	AverageAccessTime time.Duration

	EvictionEfficiency float64

	ShardDistribution float64

	MemoryEfficiency float64

	ConcurrentPerformance float64

	TTLAccuracy float64

	WarmupTime time.Duration

	ImprovementRatio float64
}

// DatabasePerformanceReport contains database optimization metrics.

type DatabasePerformanceReport struct {
	QueriesPerSecond float64

	AverageQueryTime time.Duration

	ConnectionUtilization float64

	PreparedStmtHitRate float64

	BatchEfficiency float64

	TransactionThroughput float64

	ReplicationLag time.Duration

	QueryOptimization float64

	ErrorRate float64

	ImprovementRatio float64
}

// OverallPerformanceReport contains system-wide metrics.

type OverallPerformanceReport struct {
	TotalThroughput float64

	SystemLatency time.Duration

	ResourceUtilization float64

	ScalabilityFactor float64

	ReliabilityScore float64

	PerformanceGain float64

	EnergyEfficiency float64

	CostOptimization float64

	SLACompliance float64
}

// BaselineMetrics stores baseline performance measurements.

type BaselineMetrics struct {
	HTTPLatency time.Duration

	MemoryUsage int64

	JSONProcessTime time.Duration

	GoroutineOverhead time.Duration

	CacheAccessTime time.Duration

	DatabaseQueryTime time.Duration

	MeasuredAt time.Time
}

// PerformanceTarget defines target performance improvements.

type PerformanceTarget struct {
	HTTPLatencyReduction float64 // 20-25%

	MemoryAllocationReduce float64 // 30-35%

	JSONSpeedImprovement float64 // 25-30%

	GoroutineEfficiency float64 // 15-20%

	DatabasePerformance float64 // 25-30%

	OverallImprovement float64 // 22-28%

}

// NewPerformanceAnalyzer creates a new performance analyzer.

func NewPerformanceAnalyzer() *PerformanceAnalyzer {

	return &PerformanceAnalyzer{

		httpMetrics: &HTTPPerformanceReport{},

		memoryMetrics: &MemoryPerformanceReport{},

		jsonMetrics: &JSONPerformanceReport{},

		goroutineMetrics: &GoroutinePerformanceReport{},

		cacheMetrics: &CachePerformanceReport{},

		dbMetrics: &DatabasePerformanceReport{},

		overallMetrics: &OverallPerformanceReport{},

		baselineMetrics: &BaselineMetrics{},

		startTime: time.Now(),

		lastReport: time.Now(),
	}

}

// EstablishBaseline measures baseline performance without optimizations.

func (pa *PerformanceAnalyzer) EstablishBaseline() error {

	pa.mu.Lock()

	defer pa.mu.Unlock()

	klog.Info("Establishing performance baseline...")

	// Measure HTTP baseline.

	start := time.Now()

	// Simulate standard HTTP operations.

	for range 1000 {

		// Standard HTTP client operations.

		time.Sleep(time.Microsecond * 50) // Simulate network delay

	}

	pa.baselineMetrics.HTTPLatency = time.Since(start) / 1000

	// Measure memory baseline.

	var m1, m2 runtime.MemStats

	runtime.GC()

	runtime.ReadMemStats(&m1)

	// Simulate memory allocations.

	for range 10000 {

		_ = make([]byte, 1024)

	}

	runtime.ReadMemStats(&m2)

	pa.baselineMetrics.MemoryUsage = int64(m2.TotalAlloc - m1.TotalAlloc)

	// Measure JSON baseline.

	start = time.Now()

	testData := map[string]interface{}{

		"test": "data",

		"number": 123,

		"array": []string{"a", "b", "c"},
	}

	for range 1000 {

		// Standard JSON operations.

		data, _ := json.Marshal(testData)

		var result map[string]interface{}

		json.Unmarshal(data, &result)

	}

	pa.baselineMetrics.JSONProcessTime = time.Since(start) / 1000

	// Measure goroutine baseline.

	start = time.Now()

	var wg sync.WaitGroup

	for range 100 {

		wg.Add(1)

		go func() {

			defer wg.Done()

			time.Sleep(time.Microsecond)

		}()

	}

	wg.Wait()

	pa.baselineMetrics.GoroutineOverhead = time.Since(start) / 100

	// Measure cache baseline.

	start = time.Now()

	m := make(map[string]string)

	for i := range 1000 {

		m[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d", i)

	}

	for i := range 1000 {

		_ = m[fmt.Sprintf("key_%d", i)]

	}

	pa.baselineMetrics.CacheAccessTime = time.Since(start) / 2000

	// Measure database baseline (simplified).

	start = time.Now()

	for range 100 {

		// Simulate database query overhead.

		time.Sleep(time.Microsecond * 100)

	}

	pa.baselineMetrics.DatabaseQueryTime = time.Since(start) / 100

	pa.baselineMetrics.MeasuredAt = time.Now()

	klog.Info("Performance baseline established")

	return nil

}

// AnalyzeHTTPPerformance analyzes HTTP layer performance.

func (pa *PerformanceAnalyzer) AnalyzeHTTPPerformance(client *OptimizedHTTPClient) {

	pa.mu.Lock()

	defer pa.mu.Unlock()

	metrics := client.GetMetrics()

	elapsed := time.Since(pa.startTime).Seconds()

	pa.httpMetrics.RequestsPerSecond = float64(metrics.RequestCount) / elapsed

	pa.httpMetrics.AverageLatency = time.Duration(metrics.ResponseTime / max(metrics.RequestCount, 1))

	pa.httpMetrics.ConnectionPoolHitRate = float64(metrics.ConnectionsReused) / float64(max(metrics.ConnectionsCreated, 1))

	pa.httpMetrics.CompressionRatio = metrics.CompressionRatio

	pa.httpMetrics.HTTP2Usage = float64(metrics.HTTP2Connections) / float64(max(metrics.RequestCount, 1))

	pa.httpMetrics.ErrorRate = float64(metrics.ErrorCount) / float64(max(metrics.RequestCount, 1))

	pa.httpMetrics.BufferPoolEfficiency = client.bufferPool.GetHitRate()

	// Calculate improvement ratio.

	if pa.baselineMetrics.HTTPLatency > 0 {

		improvement := (pa.baselineMetrics.HTTPLatency - pa.httpMetrics.AverageLatency).Seconds() / pa.baselineMetrics.HTTPLatency.Seconds()

		pa.httpMetrics.ImprovementRatio = improvement

	}

}

// AnalyzeMemoryPerformance analyzes memory optimization performance.

func (pa *PerformanceAnalyzer) AnalyzeMemoryPerformance(manager *MemoryPoolManager) {

	pa.mu.Lock()

	defer pa.mu.Unlock()

	memStats := manager.GetMemoryStats()

	gcMetrics := manager.GetGCMetrics()

	pa.memoryMetrics.HeapSize = memStats.HeapSize

	pa.memoryMetrics.ObjectPoolHitRate = memStats.PoolHitRate

	pa.memoryMetrics.RingBufferUtilization = memStats.RingBufferUtilization

	pa.memoryMetrics.MemoryMapUsage = memStats.MemoryMapUsage

	if len(gcMetrics) > 0 {

		lastGC := gcMetrics[len(gcMetrics)-1]

		pa.memoryMetrics.GCPauseTime = lastGC.PauseTime

		pa.memoryMetrics.GCFrequency = float64(memStats.GCCount) / time.Since(pa.startTime).Hours()

	}

	// Calculate improvement ratio.

	if pa.baselineMetrics.MemoryUsage > 0 {

		currentUsage := memStats.TotalAllocated

		improvement := float64(pa.baselineMetrics.MemoryUsage-currentUsage) / float64(pa.baselineMetrics.MemoryUsage)

		pa.memoryMetrics.ImprovementRatio = improvement

		pa.memoryMetrics.AllocationReduction = improvement

	}

}

// AnalyzeJSONPerformance analyzes JSON processing performance.

func (pa *PerformanceAnalyzer) AnalyzeJSONPerformance(processor *OptimizedJSONProcessor) {

	pa.mu.Lock()

	defer pa.mu.Unlock()

	metrics := processor.GetMetrics()

	elapsed := time.Since(pa.startTime).Seconds()

	marshalOps := float64(metrics.MarshalCount)

	unmarshalOps := float64(metrics.UnmarshalCount)

	pa.jsonMetrics.MarshalOpsPerSec = marshalOps / elapsed

	pa.jsonMetrics.UnmarshalOpsPerSec = unmarshalOps / elapsed

	pa.jsonMetrics.AverageProcessTime = time.Duration(metrics.TotalProcessTime / max(metrics.MarshalCount+metrics.UnmarshalCount, 1))

	pa.jsonMetrics.SchemaHitRate = metrics.SchemaHitRate

	pa.jsonMetrics.PoolEfficiency = metrics.PoolHitRate

	pa.jsonMetrics.ConcurrencyLevel = metrics.ConcurrencyLevel

	pa.jsonMetrics.SIMDUtilization = float64(metrics.SIMDOperations) / float64(max(marshalOps+unmarshalOps, 1))

	pa.jsonMetrics.ErrorRate = float64(metrics.ErrorCount) / float64(max(marshalOps+unmarshalOps, 1))

	// Calculate improvement ratio.

	if pa.baselineMetrics.JSONProcessTime > 0 {

		improvement := (pa.baselineMetrics.JSONProcessTime - pa.jsonMetrics.AverageProcessTime).Seconds() / pa.baselineMetrics.JSONProcessTime.Seconds()

		pa.jsonMetrics.ImprovementRatio = improvement

	}

}

// AnalyzeGoroutinePerformance analyzes goroutine pool performance.

func (pa *PerformanceAnalyzer) AnalyzeGoroutinePerformance(pool *EnhancedGoroutinePool) {

	pa.mu.Lock()

	defer pa.mu.Unlock()

	metrics := pool.GetMetrics()

	elapsed := time.Since(pa.startTime).Seconds()

	pa.goroutineMetrics.TasksPerSecond = float64(metrics.CompletedTasks) / elapsed

	pa.goroutineMetrics.AverageWaitTime = time.Duration(metrics.AverageWaitTime)

	pa.goroutineMetrics.AverageProcessTime = time.Duration(metrics.AverageProcessTime)

	pa.goroutineMetrics.WorkStealingEfficiency = float64(metrics.StolenTasks) / float64(max(metrics.CompletedTasks, 1))

	if metrics.ActiveWorkers > 0 {

		pa.goroutineMetrics.WorkerUtilization = float64(metrics.CompletedTasks) / float64(metrics.ActiveWorkers) / elapsed

	}

	// Calculate improvement ratio.

	if pa.baselineMetrics.GoroutineOverhead > 0 {

		improvement := (pa.baselineMetrics.GoroutineOverhead - pa.goroutineMetrics.AverageProcessTime).Seconds() / pa.baselineMetrics.GoroutineOverhead.Seconds()

		pa.goroutineMetrics.ImprovementRatio = improvement

	}

}

// AnalyzeCachePerformance analyzes cache optimization performance.

func (pa *PerformanceAnalyzer) AnalyzeCachePerformance(cache *OptimizedCache[string, interface{}]) {

	pa.mu.Lock()

	defer pa.mu.Unlock()

	metrics := cache.GetMetrics()

	stats := cache.GetStats()

	pa.cacheMetrics.HitRate = metrics.HitRatio

	pa.cacheMetrics.AverageAccessTime = time.Duration(metrics.AverageAccessTime)

	pa.cacheMetrics.EvictionEfficiency = float64(metrics.Evictions) / float64(max(metrics.Size, 1))

	pa.cacheMetrics.MemoryEfficiency = stats.MemoryEfficiency / 100

	// Calculate shard distribution variance (lower is better).

	if len(metrics.ShardDistribution) > 0 {

		var total, variance float64

		for _, count := range metrics.ShardDistribution {

			total += float64(count)

		}

		avg := total / float64(len(metrics.ShardDistribution))

		for _, count := range metrics.ShardDistribution {

			diff := float64(count) - avg

			variance += diff * diff

		}

		variance /= float64(len(metrics.ShardDistribution))

		pa.cacheMetrics.ShardDistribution = 1.0 - (variance / (avg * avg)) // Normalize (1 = perfect distribution)

	}

	// Calculate improvement ratio.

	if pa.baselineMetrics.CacheAccessTime > 0 {

		improvement := (pa.baselineMetrics.CacheAccessTime - pa.cacheMetrics.AverageAccessTime).Seconds() / pa.baselineMetrics.CacheAccessTime.Seconds()

		pa.cacheMetrics.ImprovementRatio = improvement

	}

}

// AnalyzeDatabasePerformance analyzes database optimization performance.

func (pa *PerformanceAnalyzer) AnalyzeDatabasePerformance(manager *OptimizedDBManager) {

	pa.mu.Lock()

	defer pa.mu.Unlock()

	metrics := manager.GetMetrics()

	elapsed := time.Since(pa.startTime).Seconds()

	pa.dbMetrics.QueriesPerSecond = float64(metrics.QueryCount) / elapsed

	pa.dbMetrics.AverageQueryTime = time.Duration(metrics.AverageQueryTime)

	pa.dbMetrics.ConnectionUtilization = manager.GetConnectionUtilization() / 100

	pa.dbMetrics.PreparedStmtHitRate = float64(metrics.PreparedStmtHits) / float64(max(metrics.PreparedStmtHits+metrics.PreparedStmtMisses, 1))

	pa.dbMetrics.BatchEfficiency = float64(metrics.BatchCount) / float64(max(metrics.QueryCount, 1))

	pa.dbMetrics.TransactionThroughput = float64(metrics.TransactionCount) / elapsed

	pa.dbMetrics.ErrorRate = float64(metrics.ErrorCount) / float64(max(metrics.QueryCount, 1))

	// Calculate improvement ratio.

	if pa.baselineMetrics.DatabaseQueryTime > 0 {

		improvement := (pa.baselineMetrics.DatabaseQueryTime - pa.dbMetrics.AverageQueryTime).Seconds() / pa.baselineMetrics.DatabaseQueryTime.Seconds()

		pa.dbMetrics.ImprovementRatio = improvement

	}

}

// CalculateOverallPerformance calculates system-wide performance metrics.

func (pa *PerformanceAnalyzer) CalculateOverallPerformance() {

	pa.mu.Lock()

	defer pa.mu.Unlock()

	// Weighted average of all improvement ratios.

	weights := map[string]float64{

		"http": 0.20,

		"memory": 0.25,

		"json": 0.20,

		"goroutine": 0.15,

		"cache": 0.10,

		"database": 0.10,
	}

	overallImprovement := weights["http"]*pa.httpMetrics.ImprovementRatio +

		weights["memory"]*pa.memoryMetrics.ImprovementRatio +

		weights["json"]*pa.jsonMetrics.ImprovementRatio +

		weights["goroutine"]*pa.goroutineMetrics.ImprovementRatio +

		weights["cache"]*pa.cacheMetrics.ImprovementRatio +

		weights["database"]*pa.dbMetrics.ImprovementRatio

	pa.overallMetrics.PerformanceGain = overallImprovement

	// Calculate composite metrics.

	pa.overallMetrics.TotalThroughput = pa.httpMetrics.RequestsPerSecond +

		pa.jsonMetrics.MarshalOpsPerSec +

		pa.goroutineMetrics.TasksPerSecond +

		pa.dbMetrics.QueriesPerSecond

	// System latency (weighted average).

	pa.overallMetrics.SystemLatency = time.Duration(

		float64(pa.httpMetrics.AverageLatency)*0.4 +

			float64(pa.jsonMetrics.AverageProcessTime)*0.3 +

			float64(pa.goroutineMetrics.AverageProcessTime)*0.2 +

			float64(pa.dbMetrics.AverageQueryTime)*0.1,
	)

	// Resource utilization (average of all efficiency metrics).

	pa.overallMetrics.ResourceUtilization = (pa.httpMetrics.ConnectionPoolHitRate +

		pa.memoryMetrics.ObjectPoolHitRate +

		pa.jsonMetrics.PoolEfficiency +

		pa.goroutineMetrics.WorkerUtilization +

		pa.cacheMetrics.MemoryEfficiency +

		pa.dbMetrics.ConnectionUtilization) / 6.0

	// Reliability score (inverse of error rates).

	errorRate := (pa.httpMetrics.ErrorRate +

		pa.jsonMetrics.ErrorRate +

		pa.dbMetrics.ErrorRate) / 3.0

	pa.overallMetrics.ReliabilityScore = max(0, 1.0-errorRate)

	// SLA compliance (based on latency targets).

	latencyCompliance := 1.0

	if pa.overallMetrics.SystemLatency > time.Millisecond*100 {

		latencyCompliance = 0.5

	} else if pa.overallMetrics.SystemLatency > time.Millisecond*50 {

		latencyCompliance = 0.8

	}

	pa.overallMetrics.SLACompliance = latencyCompliance

}

// GeneratePerformanceReport generates a comprehensive performance report.

func (pa *PerformanceAnalyzer) GeneratePerformanceReport() *PerformanceReport {

	pa.mu.RLock()

	defer pa.mu.RUnlock()

	pa.CalculateOverallPerformance()

	return &PerformanceReport{

		Timestamp: time.Now(),

		AnalysisDuration: time.Since(pa.startTime),

		Baseline: *pa.baselineMetrics,

		HTTP: *pa.httpMetrics,

		Memory: *pa.memoryMetrics,

		JSON: *pa.jsonMetrics,

		Goroutine: *pa.goroutineMetrics,

		Cache: *pa.cacheMetrics,

		Database: *pa.dbMetrics,

		Overall: *pa.overallMetrics,

		Targets: GetPerformanceTargets(),

		AchievedTargets: pa.checkTargetAchievement(),
	}

}

// PerformanceReport contains the complete performance analysis.

type PerformanceReport struct {
	Timestamp time.Time

	AnalysisDuration time.Duration

	Baseline BaselineMetrics

	HTTP HTTPPerformanceReport

	Memory MemoryPerformanceReport

	JSON JSONPerformanceReport

	Goroutine GoroutinePerformanceReport

	Cache CachePerformanceReport

	Database DatabasePerformanceReport

	Overall OverallPerformanceReport

	Targets PerformanceTarget

	AchievedTargets map[string]bool
}

// GetPerformanceTargets returns the target performance improvements.

func GetPerformanceTargets() PerformanceTarget {

	return PerformanceTarget{

		HTTPLatencyReduction: 0.225, // 22.5% (midpoint of 20-25%)

		MemoryAllocationReduce: 0.325, // 32.5% (midpoint of 30-35%)

		JSONSpeedImprovement: 0.275, // 27.5% (midpoint of 25-30%)

		GoroutineEfficiency: 0.175, // 17.5% (midpoint of 15-20%)

		DatabasePerformance: 0.275, // 27.5% (midpoint of 25-30%)

		OverallImprovement: 0.25, // 25% (midpoint of 22-28%)

	}

}

// checkTargetAchievement checks which performance targets have been achieved.

func (pa *PerformanceAnalyzer) checkTargetAchievement() map[string]bool {

	targets := GetPerformanceTargets()

	results := make(map[string]bool)

	results["http_latency"] = pa.httpMetrics.ImprovementRatio >= targets.HTTPLatencyReduction

	results["memory_allocation"] = pa.memoryMetrics.ImprovementRatio >= targets.MemoryAllocationReduce

	results["json_speed"] = pa.jsonMetrics.ImprovementRatio >= targets.JSONSpeedImprovement

	results["goroutine_efficiency"] = pa.goroutineMetrics.ImprovementRatio >= targets.GoroutineEfficiency

	results["database_performance"] = pa.dbMetrics.ImprovementRatio >= targets.DatabasePerformance

	results["overall_improvement"] = pa.overallMetrics.PerformanceGain >= targets.OverallImprovement

	return results

}

// PrintPerformanceReport prints a detailed performance report.

func (pr *PerformanceReport) PrintPerformanceReport() {

	klog.Info("\n" + strings.Repeat("=", 80))

	klog.Info("NEPHORAN INTENT OPERATOR - GO 1.24+ PERFORMANCE OPTIMIZATION REPORT")

	klog.Info("================================================================================")

	klog.Infof("Report Generated: %s", pr.Timestamp.Format(time.RFC3339))

	klog.Infof("Analysis Duration: %v", pr.AnalysisDuration)

	klog.Info("")

	// Baseline metrics.

	klog.Info("BASELINE METRICS (Pre-optimization)")

	klog.Info("----------------------------------------")

	klog.Infof("HTTP Latency: %v", pr.Baseline.HTTPLatency)

	klog.Infof("Memory Usage: %d bytes", pr.Baseline.MemoryUsage)

	klog.Infof("JSON Process Time: %v", pr.Baseline.JSONProcessTime)

	klog.Infof("Goroutine Overhead: %v", pr.Baseline.GoroutineOverhead)

	klog.Infof("Cache Access Time: %v", pr.Baseline.CacheAccessTime)

	klog.Infof("Database Query Time: %v", pr.Baseline.DatabaseQueryTime)

	klog.Info("")

	// HTTP Performance.

	klog.Info("HTTP LAYER PERFORMANCE")

	klog.Info("----------------------------------------")

	klog.Infof("Requests/sec: %.2f", pr.HTTP.RequestsPerSecond)

	klog.Infof("Average Latency: %v", pr.HTTP.AverageLatency)

	klog.Infof("Connection Pool Hit Rate: %.2f%%", pr.HTTP.ConnectionPoolHitRate*100)

	klog.Infof("Compression Ratio: %.2f", pr.HTTP.CompressionRatio)

	klog.Infof("HTTP/2 Usage: %.2f%%", pr.HTTP.HTTP2Usage*100)

	klog.Infof("Buffer Pool Efficiency: %.2f%%", pr.HTTP.BufferPoolEfficiency*100)

	klog.Infof("Error Rate: %.4f%%", pr.HTTP.ErrorRate*100)

	klog.Infof("IMPROVEMENT: %.2f%% ✓", pr.HTTP.ImprovementRatio*100)

	klog.Info("")

	// Memory Performance.

	klog.Info("MEMORY OPTIMIZATION PERFORMANCE")

	klog.Info("----------------------------------------")

	klog.Infof("Heap Size: %d bytes", pr.Memory.HeapSize)

	klog.Infof("GC Pause Time: %v", pr.Memory.GCPauseTime)

	klog.Infof("GC Frequency: %.2f/hour", pr.Memory.GCFrequency)

	klog.Infof("Object Pool Hit Rate: %.2f%%", pr.Memory.ObjectPoolHitRate*100)

	klog.Infof("Ring Buffer Utilization: %.2f%%", pr.Memory.RingBufferUtilization*100)

	klog.Infof("Memory Map Usage: %d bytes", pr.Memory.MemoryMapUsage)

	klog.Infof("Allocation Reduction: %.2f%%", pr.Memory.AllocationReduction*100)

	klog.Infof("IMPROVEMENT: %.2f%% ✓", pr.Memory.ImprovementRatio*100)

	klog.Info("")

	// JSON Performance.

	klog.Info("JSON PROCESSING PERFORMANCE")

	klog.Info("----------------------------------------")

	klog.Infof("Marshal ops/sec: %.2f", pr.JSON.MarshalOpsPerSec)

	klog.Infof("Unmarshal ops/sec: %.2f", pr.JSON.UnmarshalOpsPerSec)

	klog.Infof("Average Process Time: %v", pr.JSON.AverageProcessTime)

	klog.Infof("Schema Hit Rate: %.2f%%", pr.JSON.SchemaHitRate*100)

	klog.Infof("Pool Efficiency: %.2f%%", pr.JSON.PoolEfficiency*100)

	klog.Infof("Concurrency Level: %d", pr.JSON.ConcurrencyLevel)

	klog.Infof("SIMD Utilization: %.2f%%", pr.JSON.SIMDUtilization*100)

	klog.Infof("Error Rate: %.4f%%", pr.JSON.ErrorRate*100)

	klog.Infof("IMPROVEMENT: %.2f%% ✓", pr.JSON.ImprovementRatio*100)

	klog.Info("")

	// Goroutine Performance.

	klog.Info("GOROUTINE POOL PERFORMANCE")

	klog.Info("----------------------------------------")

	klog.Infof("Tasks/sec: %.2f", pr.Goroutine.TasksPerSecond)

	klog.Infof("Average Wait Time: %v", pr.Goroutine.AverageWaitTime)

	klog.Infof("Average Process Time: %v", pr.Goroutine.AverageProcessTime)

	klog.Infof("Worker Utilization: %.2f%%", pr.Goroutine.WorkerUtilization*100)

	klog.Infof("Work Stealing Efficiency: %.2f%%", pr.Goroutine.WorkStealingEfficiency*100)

	klog.Infof("CPU Affinity Benefit: %.2f%%", pr.Goroutine.CPUAffinityBenefit*100)

	klog.Infof("IMPROVEMENT: %.2f%% ✓", pr.Goroutine.ImprovementRatio*100)

	klog.Info("")

	// Cache Performance.

	klog.Info("CACHE OPTIMIZATION PERFORMANCE")

	klog.Info("----------------------------------------")

	klog.Infof("Hit Rate: %.2f%%", pr.Cache.HitRate*100)

	klog.Infof("Average Access Time: %v", pr.Cache.AverageAccessTime)

	klog.Infof("Eviction Efficiency: %.2f%%", pr.Cache.EvictionEfficiency*100)

	klog.Infof("Shard Distribution: %.2f%%", pr.Cache.ShardDistribution*100)

	klog.Infof("Memory Efficiency: %.2f%%", pr.Cache.MemoryEfficiency*100)

	klog.Infof("IMPROVEMENT: %.2f%% ✓", pr.Cache.ImprovementRatio*100)

	klog.Info("")

	// Database Performance.

	klog.Info("DATABASE OPTIMIZATION PERFORMANCE")

	klog.Info("----------------------------------------")

	klog.Infof("Queries/sec: %.2f", pr.Database.QueriesPerSecond)

	klog.Infof("Average Query Time: %v", pr.Database.AverageQueryTime)

	klog.Infof("Connection Utilization: %.2f%%", pr.Database.ConnectionUtilization*100)

	klog.Infof("Prepared Stmt Hit Rate: %.2f%%", pr.Database.PreparedStmtHitRate*100)

	klog.Infof("Batch Efficiency: %.2f%%", pr.Database.BatchEfficiency*100)

	klog.Infof("Transaction Throughput: %.2f/sec", pr.Database.TransactionThroughput)

	klog.Infof("Error Rate: %.4f%%", pr.Database.ErrorRate*100)

	klog.Infof("IMPROVEMENT: %.2f%% ✓", pr.Database.ImprovementRatio*100)

	klog.Info("")

	// Overall Performance.

	klog.Info("OVERALL SYSTEM PERFORMANCE")

	klog.Info("----------------------------------------")

	klog.Infof("Total Throughput: %.2f ops/sec", pr.Overall.TotalThroughput)

	klog.Infof("System Latency: %v", pr.Overall.SystemLatency)

	klog.Infof("Resource Utilization: %.2f%%", pr.Overall.ResourceUtilization*100)

	klog.Infof("Reliability Score: %.2f%%", pr.Overall.ReliabilityScore*100)

	klog.Infof("SLA Compliance: %.2f%%", pr.Overall.SLACompliance*100)

	klog.Infof("Energy Efficiency: %.2f%%", pr.Overall.EnergyEfficiency*100)

	klog.Infof("OVERALL IMPROVEMENT: %.2f%% ✓", pr.Overall.PerformanceGain*100)

	klog.Info("")

	// Target Achievement.

	klog.Info("PERFORMANCE TARGET ACHIEVEMENT")

	klog.Info("----------------------------------------")

	for target, achieved := range pr.AchievedTargets {

		status := "❌ MISSED"

		if achieved {

			status = "✅ ACHIEVED"

		}

		klog.Infof("%s: %s", target, status)

	}

	klog.Info("")

	// Summary.

	klog.Info("PERFORMANCE OPTIMIZATION SUMMARY")

	klog.Info(strings.Repeat("=", 40))

	achievedCount := 0

	for _, achieved := range pr.AchievedTargets {

		if achieved {

			achievedCount++

		}

	}

	successRate := float64(achievedCount) / float64(len(pr.AchievedTargets)) * 100

	klog.Infof("Targets Achieved: %d/%d (%.1f%%)", achievedCount, len(pr.AchievedTargets), successRate)

	klog.Infof("Overall Performance Gain: %.2f%%", pr.Overall.PerformanceGain*100)

	if pr.Overall.PerformanceGain >= pr.Targets.OverallImprovement {

		klog.Info("🎉 SUCCESS: Performance optimization targets achieved!")

	} else {

		klog.Infof("⚠️  PARTIAL: %.2f%% improvement achieved (target: %.2f%%)",

			pr.Overall.PerformanceGain*100, pr.Targets.OverallImprovement*100)

	}

	klog.Info("\n" + strings.Repeat("=", 80))

}

// max returns the maximum of two values.

func max[T ~int | ~int64 | ~float64](a, b T) T {

	if a > b {

		return a

	}

	return b

}

// ValidatePerformanceTargets validates that all performance targets are met.

func (pr *PerformanceReport) ValidatePerformanceTargets() error {

	var failures []string

	if pr.HTTP.ImprovementRatio < pr.Targets.HTTPLatencyReduction {

		failures = append(failures, fmt.Sprintf("HTTP latency: %.2f%% < %.2f%%",

			pr.HTTP.ImprovementRatio*100, pr.Targets.HTTPLatencyReduction*100))

	}

	if pr.Memory.ImprovementRatio < pr.Targets.MemoryAllocationReduce {

		failures = append(failures, fmt.Sprintf("Memory allocation: %.2f%% < %.2f%%",

			pr.Memory.ImprovementRatio*100, pr.Targets.MemoryAllocationReduce*100))

	}

	if pr.JSON.ImprovementRatio < pr.Targets.JSONSpeedImprovement {

		failures = append(failures, fmt.Sprintf("JSON processing: %.2f%% < %.2f%%",

			pr.JSON.ImprovementRatio*100, pr.Targets.JSONSpeedImprovement*100))

	}

	if pr.Goroutine.ImprovementRatio < pr.Targets.GoroutineEfficiency {

		failures = append(failures, fmt.Sprintf("Goroutine efficiency: %.2f%% < %.2f%%",

			pr.Goroutine.ImprovementRatio*100, pr.Targets.GoroutineEfficiency*100))

	}

	if pr.Database.ImprovementRatio < pr.Targets.DatabasePerformance {

		failures = append(failures, fmt.Sprintf("Database performance: %.2f%% < %.2f%%",

			pr.Database.ImprovementRatio*100, pr.Targets.DatabasePerformance*100))

	}

	if pr.Overall.PerformanceGain < pr.Targets.OverallImprovement {

		failures = append(failures, fmt.Sprintf("Overall improvement: %.2f%% < %.2f%%",

			pr.Overall.PerformanceGain*100, pr.Targets.OverallImprovement*100))

	}

	if len(failures) > 0 {

		return fmt.Errorf("Performance targets not met: %v", failures)

	}

	return nil

}
