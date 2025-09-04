//go:build go1.24

package runtime

import (
	"runtime"
	"testing"
)

func TestDefaultOptimizationConfig(t *testing.T) {
	config := DefaultOptimizationConfig()

	if config == nil {
		t.Fatal("DefaultOptimizationConfig returned nil")
	}

	if config.MaxProcs <= 0 {
		t.Errorf("Expected positive MaxProcs, got %d", config.MaxProcs)
	}

	if config.MaxProcs > runtime.NumCPU() {
		t.Errorf("MaxProcs %d exceeds available CPUs %d", config.MaxProcs, runtime.NumCPU())
	}

	if !config.SwissTablesEnabled {
		t.Error("Expected SwissTablesEnabled to be true")
	}

	if !config.PGOEnabled {
		t.Error("Expected PGOEnabled to be true")
	}
}

func TestApplyOptimizations(t *testing.T) {
	config := &OptimizationConfig{
		MaxProcs:           2,
		AutoTuneMaxProcs:   false,
		SwissTablesEnabled: true,
		GCPercent:          100,
		PGOEnabled:         true,
	}

	originalMaxProcs := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(originalMaxProcs)

	err := ApplyOptimizations(config)
	if err != nil {
		t.Fatalf("ApplyOptimizations failed: %v", err)
	}

	newMaxProcs := runtime.GOMAXPROCS(0)
	if newMaxProcs != 2 {
		t.Errorf("Expected GOMAXPROCS to be 2, got %d", newMaxProcs)
	}
}

func TestCalculateOptimalGOMAXPROCS(t *testing.T) {
	optimal := calculateOptimalGOMAXPROCS()

	if optimal <= 0 {
		t.Errorf("Expected positive optimal GOMAXPROCS, got %d", optimal)
	}

	if optimal > runtime.NumCPU() {
		t.Errorf("Optimal GOMAXPROCS %d exceeds available CPUs %d", optimal, runtime.NumCPU())
	}
}

func TestGetRuntimeStats(t *testing.T) {
	stats := GetRuntimeStats()

	expectedKeys := []string{
		"gomaxprocs", "num_cpu", "num_goroutine", "num_gc",
		"heap_alloc_mb", "heap_sys_mb", "heap_in_use_mb",
		"stack_in_use_mb", "next_gc_mb",
	}

	for _, key := range expectedKeys {
		if _, exists := stats[key]; !exists {
			t.Errorf("Expected key %s not found in runtime stats", key)
		}
	}

	// Verify types
	if gomaxprocs, ok := stats["gomaxprocs"].(int); !ok || gomaxprocs <= 0 {
		t.Errorf("Expected positive int for gomaxprocs, got %v", stats["gomaxprocs"])
	}

	if numCPU, ok := stats["num_cpu"].(int); !ok || numCPU <= 0 {
		t.Errorf("Expected positive int for num_cpu, got %v", stats["num_cpu"])
	}
}

func TestTuneForTelcoWorkloads(t *testing.T) {
	originalMaxProcs := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(originalMaxProcs)

	err := TuneForTelcoWorkloads()
	if err != nil {
		t.Fatalf("TuneForTelcoWorkloads failed: %v", err)
	}

	// Verify GOMAXPROCS was set appropriately
	newMaxProcs := runtime.GOMAXPROCS(0)
	if newMaxProcs <= 0 {
		t.Errorf("Expected positive GOMAXPROCS after telco tuning, got %d", newMaxProcs)
	}
}

func TestPerformanceMonitor(t *testing.T) {
	config := DefaultOptimizationConfig()
	monitor := NewPerformanceMonitor(config)

	if monitor == nil {
		t.Fatal("NewPerformanceMonitor returned nil")
	}

	if monitor.config != config {
		t.Error("Performance monitor config not set correctly")
	}

	// Test start and stop
	monitor.Start()
	monitor.TriggerAdjustment()
	monitor.Stop()
}

func BenchmarkApplyOptimizations(b *testing.B) {
	config := DefaultOptimizationConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := ApplyOptimizations(config)
		if err != nil {
			b.Fatalf("ApplyOptimizations failed: %v", err)
		}
	}
}

func BenchmarkGetRuntimeStats(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stats := GetRuntimeStats()
		if len(stats) == 0 {
			b.Fatal("GetRuntimeStats returned empty map")
		}
	}
}
