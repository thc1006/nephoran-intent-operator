package performance

import (
	"context"
	"testing"
	"time"
)

// TestPerformancePackageIntegration validates all core performance components work together
func TestPerformancePackageIntegration(t *testing.T) {
	// Test metrics collector
	collector := NewMetricsCollector()
	if collector == nil {
		t.Fatal("MetricsCollector creation failed")
	}
	defer collector.Stop()
	
	// Test metrics analyzer
	analyzer := NewMetricsAnalyzer()
	if analyzer == nil {
		t.Fatal("MetricsAnalyzer creation failed")
	}
	
	// Test optimization engine
	optimizer := NewOptimizationEngine()
	if optimizer == nil {
		t.Fatal("OptimizationEngine creation failed")
	}
	
	// Test profiler
	profiler := NewProfiler()
	if profiler == nil {
		t.Fatal("Profiler creation failed")
	}
	
	// Test benchmark suite
	suite := NewBenchmarkSuite()
	if suite == nil {
		t.Fatal("BenchmarkSuite creation failed")
	}
	
	// Test CNF function tracking
	collector.RecordCNFDeployment(CNFFunctionCUCP, 2*time.Second)
	collector.RecordCNFDeployment(CNFFunctionDU, 3*time.Second)
	
	metrics := collector.GetCNFDeploymentMetrics()
	if len(metrics) != 2 {
		t.Errorf("Expected 2 CNF metrics, got %d", len(metrics))
	}
	
	// Test analyzer latency tracking
	analyzer.RecordLatency("deployment", 100*time.Millisecond)
	analyzer.RecordLatency("deployment", 150*time.Millisecond)
	analyzer.RecordThroughput("deployment", 50.0)
	
	summary := analyzer.GetPerformanceSummary()
	if summary == nil {
		t.Error("Performance summary is nil")
	}
	
	// Test cache components (minimal)
	cache := NewMemoryCache(100, 5*time.Minute)
	if cache == nil {
		t.Error("MemoryCache creation failed")
	}
	
	// Test batch processor
	processor := NewBatchProcessor(10, time.Second)
	if processor == nil {
		t.Error("BatchProcessor creation failed")
	}
	defer processor.Stop()
	
	// Test goroutine pool
	pool := NewGoroutinePool(5)
	if pool == nil {
		t.Error("GoroutinePool creation failed")
	}
	defer pool.Shutdown(context.Background())
	
	t.Log("All performance package components validated successfully!")
}