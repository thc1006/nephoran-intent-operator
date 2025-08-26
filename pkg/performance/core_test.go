package performance

import (
	"testing"
	"time"
)

// TestMetricsCollector tests the core metrics collector functionality
func TestMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector()
	
	if collector == nil {
		t.Fatal("NewMetricsCollector returned nil")
	}
	
	// Test CNF deployment recording
	collector.RecordCNFDeployment(CNFFunctionCUCP, 5*time.Second)
	
	metrics := collector.GetCNFDeploymentMetrics()
	if len(metrics) != 1 {
		t.Errorf("Expected 1 CNF metric, got %d", len(metrics))
	}
	
	if metrics[string(CNFFunctionCUCP)] != 5*time.Second {
		t.Errorf("Expected 5s deployment time, got %v", metrics[string(CNFFunctionCUCP)])
	}
	
	// Test basic metrics
	cpu := collector.GetCPUUsage()
	mem := collector.GetMemoryUsage()
	gor := collector.GetGoroutineCount()
	
	if cpu < 0 {
		t.Errorf("CPU usage should be non-negative, got %f", cpu)
	}
	if mem == 0 {
		t.Errorf("Memory usage should be positive, got %d", mem)
	}
	if gor == 0 {
		t.Errorf("Goroutine count should be positive, got %d", gor)
	}
	
	collector.Stop()
}

// TestMetricsAnalyzer tests the core analyzer functionality
func TestMetricsAnalyzer(t *testing.T) {
	analyzer := NewMetricsAnalyzer()
	
	if analyzer == nil {
		t.Fatal("NewMetricsAnalyzer returned nil")
	}
	
	// Test latency recording
	analyzer.RecordLatency("test_operation", 100*time.Millisecond)
	analyzer.RecordLatency("test_operation", 200*time.Millisecond)
	analyzer.RecordLatency("test_operation", 150*time.Millisecond)
	
	// Test throughput recording  
	analyzer.RecordThroughput("test_operation", 100.0)
	analyzer.RecordThroughput("test_operation", 150.0)
	
	// Test performance summary
	summary := analyzer.GetPerformanceSummary()
	if summary == nil {
		t.Fatal("GetPerformanceSummary returned nil")
	}
	
	latencies, ok := summary["latencies"].(map[string]interface{})
	if !ok {
		t.Fatal("Latencies not found in summary")
	}
	
	if _, exists := latencies["test_operation"]; !exists {
		t.Error("test_operation latencies not found in summary")
	}
}

// TestBenchmarkSuite tests the benchmark suite creation
func TestBenchmarkSuite(t *testing.T) {
	suite := NewBenchmarkSuite()
	
	if suite == nil {
		t.Fatal("NewBenchmarkSuite returned nil")
	}
	
	if suite.metrics == nil {
		t.Error("BenchmarkSuite metrics collector is nil")
	}
	
	if suite.optimizer == nil {
		t.Error("BenchmarkSuite optimizer is nil")
	}
	
	if suite.profiler == nil {
		t.Error("BenchmarkSuite profiler is nil")
	}
}

// TestOptimizationEngine tests the optimization engine creation
func TestOptimizationEngine(t *testing.T) {
	engine := NewOptimizationEngine()
	
	if engine == nil {
		t.Fatal("NewOptimizationEngine returned nil")
	}
	
	if engine.metrics == nil {
		t.Error("OptimizationEngine metrics collector is nil")
	}
}

// TestCNFFunctionConstants tests the CNF function constants
func TestCNFFunctionConstants(t *testing.T) {
	functions := []CNFFunction{
		CNFFunctionCUCP,
		CNFFunctionCUUP,
		CNFFunctionDU,
		CNFFunctionRIC,
	}
	
	expected := []string{"cu-cp", "cu-up", "du", "ric"}
	
	for i, fn := range functions {
		if string(fn) != expected[i] {
			t.Errorf("Expected %s, got %s", expected[i], string(fn))
		}
	}
}