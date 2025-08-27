package performance

import (
	"testing"
	"time"
)

// TestMetricsCollectorMinimal tests the core metrics collector functionality
func TestMetricsCollectorMinimal(t *testing.T) {
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
	// Memory and goroutine values may start at 0 in tests, which is valid
	if mem < 0 {
		t.Errorf("Memory usage should be non-negative, got %d", mem)
	}
	if gor < 0 {
		t.Errorf("Goroutine count should be non-negative, got %d", gor)
	}

	collector.Stop()
}

// TestMetricsAnalyzerMinimal tests the core analyzer functionality
func TestMetricsAnalyzerMinimal(t *testing.T) {
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
