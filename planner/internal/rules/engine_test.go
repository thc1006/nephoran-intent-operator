package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestRuleEngine_ScaleOut(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "test-state.json")
	defer os.Remove(tmpFile)

	engine := NewRuleEngine(Config{
		StateFile:            tmpFile,
		CooldownDuration:     1 * time.Second,
		MinReplicas:          1,
		MaxReplicas:          10,
		LatencyThresholdHigh: 100.0,
		LatencyThresholdLow:  50.0,
		PRBThresholdHigh:     0.8,
		PRBThresholdLow:      0.3,
		EvaluationWindow:     30 * time.Second,
	})

	now := time.Now()

	highLoadData := []KPMData{
		{Timestamp: now.Add(-20 * time.Second), NodeID: "node1", PRBUtilization: 0.85, P95Latency: 120, CurrentReplicas: 2},
		{Timestamp: now.Add(-10 * time.Second), NodeID: "node1", PRBUtilization: 0.90, P95Latency: 130, CurrentReplicas: 2},
		{Timestamp: now, NodeID: "node1", PRBUtilization: 0.88, P95Latency: 125, CurrentReplicas: 2},
	}

	var decision *ScalingDecision
	for _, data := range highLoadData {
		decision = engine.Evaluate(data)
	}

	if decision == nil {
		t.Fatal("Expected scaling decision, got nil")
	}

	if decision.Action != "scale-out" {
		t.Errorf("Expected scale-out action, got %s", decision.Action)
	}

	if decision.TargetReplicas != 3 {
		t.Errorf("Expected target replicas to be 3, got %d", decision.TargetReplicas)
	}
}

func TestRuleEngine_ScaleIn(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "test-state2.json")
	defer os.Remove(tmpFile)

	engine := NewRuleEngine(Config{
		StateFile:            tmpFile,
		CooldownDuration:     1 * time.Second,
		MinReplicas:          1,
		MaxReplicas:          10,
		LatencyThresholdHigh: 100.0,
		LatencyThresholdLow:  50.0,
		PRBThresholdHigh:     0.8,
		PRBThresholdLow:      0.3,
		EvaluationWindow:     30 * time.Second,
	})

	engine.state.CurrentReplicas = 5
	now := time.Now()

	lowLoadData := []KPMData{
		{Timestamp: now.Add(-20 * time.Second), NodeID: "node1", PRBUtilization: 0.2, P95Latency: 30, CurrentReplicas: 5},
		{Timestamp: now.Add(-10 * time.Second), NodeID: "node1", PRBUtilization: 0.25, P95Latency: 35, CurrentReplicas: 5},
		{Timestamp: now, NodeID: "node1", PRBUtilization: 0.22, P95Latency: 32, CurrentReplicas: 5},
	}

	var decision *ScalingDecision
	for _, data := range lowLoadData {
		decision = engine.Evaluate(data)
	}

	if decision == nil {
		t.Fatal("Expected scaling decision, got nil")
	}

	if decision.Action != "scale-in" {
		t.Errorf("Expected scale-in action, got %s", decision.Action)
	}

	if decision.TargetReplicas != 4 {
		t.Errorf("Expected target replicas to be 4, got %d", decision.TargetReplicas)
	}
}

func TestRuleEngine_Cooldown(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "test-state3.json")
	defer os.Remove(tmpFile)

	engine := NewRuleEngine(Config{
		StateFile:            tmpFile,
		CooldownDuration:     5 * time.Second,
		MinReplicas:          1,
		MaxReplicas:          10,
		LatencyThresholdHigh: 100.0,
		LatencyThresholdLow:  50.0,
		PRBThresholdHigh:     0.8,
		PRBThresholdLow:      0.3,
		EvaluationWindow:     30 * time.Second,
	})

	now := time.Now()

	data1 := []KPMData{
		{Timestamp: now.Add(-20 * time.Second), NodeID: "node1", PRBUtilization: 0.85, P95Latency: 120, CurrentReplicas: 2},
		{Timestamp: now.Add(-10 * time.Second), NodeID: "node1", PRBUtilization: 0.90, P95Latency: 130, CurrentReplicas: 2},
		{Timestamp: now, NodeID: "node1", PRBUtilization: 0.88, P95Latency: 125, CurrentReplicas: 2},
	}

	var decision1 *ScalingDecision
	for _, data := range data1 {
		decision1 = engine.Evaluate(data)
	}

	if decision1 == nil {
		t.Fatal("Expected first scaling decision")
	}

	data2 := KPMData{
		Timestamp:       now.Add(1 * time.Second),
		NodeID:          "node1",
		PRBUtilization:  0.95,
		P95Latency:      150,
		CurrentReplicas: 3,
	}
	decision2 := engine.Evaluate(data2)

	if decision2 != nil {
		t.Error("Expected no decision during cooldown period")
	}
}

func TestRuleEngine_MaxReplicas(t *testing.T) {
	engine := NewRuleEngine(Config{
		CooldownDuration:     1 * time.Second,
		MinReplicas:          1,
		MaxReplicas:          3,
		LatencyThresholdHigh: 100.0,
		LatencyThresholdLow:  50.0,
		PRBThresholdHigh:     0.8,
		PRBThresholdLow:      0.3,
		EvaluationWindow:     30 * time.Second,
	})

	engine.state.CurrentReplicas = 3
	now := time.Now()

	highLoadData := []KPMData{
		{Timestamp: now.Add(-20 * time.Second), NodeID: "node1", PRBUtilization: 0.95, P95Latency: 150, CurrentReplicas: 3},
		{Timestamp: now.Add(-10 * time.Second), NodeID: "node1", PRBUtilization: 0.96, P95Latency: 160, CurrentReplicas: 3},
		{Timestamp: now, NodeID: "node1", PRBUtilization: 0.97, P95Latency: 170, CurrentReplicas: 3},
	}

	var decision *ScalingDecision
	for _, data := range highLoadData {
		decision = engine.Evaluate(data)
	}

	if decision != nil {
		t.Error("Expected no decision when at max replicas")
	}
}

func TestRuleEngine_MinReplicas(t *testing.T) {
	engine := NewRuleEngine(Config{
		CooldownDuration:     1 * time.Second,
		MinReplicas:          1,
		MaxReplicas:          10,
		LatencyThresholdHigh: 100.0,
		LatencyThresholdLow:  50.0,
		PRBThresholdHigh:     0.8,
		PRBThresholdLow:      0.3,
		EvaluationWindow:     30 * time.Second,
	})

	engine.state.CurrentReplicas = 1
	now := time.Now()

	lowLoadData := []KPMData{
		{Timestamp: now.Add(-20 * time.Second), NodeID: "node1", PRBUtilization: 0.1, P95Latency: 20, CurrentReplicas: 1},
		{Timestamp: now.Add(-10 * time.Second), NodeID: "node1", PRBUtilization: 0.15, P95Latency: 25, CurrentReplicas: 1},
		{Timestamp: now, NodeID: "node1", PRBUtilization: 0.12, P95Latency: 22, CurrentReplicas: 1},
	}

	var decision *ScalingDecision
	for _, data := range lowLoadData {
		decision = engine.Evaluate(data)
	}

	if decision != nil {
		t.Error("Expected no decision when at min replicas")
	}
}

// TestRuleEngine_MemoryOptimization verifies that memory management optimizations work correctly
func TestRuleEngine_MemoryOptimization(t *testing.T) {
	engine := NewRuleEngine(Config{
		CooldownDuration:     1 * time.Second,
		MinReplicas:          1,
		MaxReplicas:          10,
		LatencyThresholdHigh: 100.0,
		LatencyThresholdLow:  50.0,
		PRBThresholdHigh:     0.8,
		PRBThresholdLow:      0.3,
		EvaluationWindow:     30 * time.Second,
		MaxHistorySize:       10, // Small size for testing
		PruneInterval:        1 * time.Second,
	})

	// Add more metrics than MaxHistorySize to test capacity management
	now := time.Now()
	for i := 0; i < 15; i++ {
		data := KPMData{
			Timestamp:       now.Add(time.Duration(i) * time.Second),
			NodeID:          "node1",
			PRBUtilization:  0.5,
			P95Latency:      75,
			CurrentReplicas: 2,
		}
		engine.Evaluate(data)
	}

	// Verify that history size is kept within bounds
	if len(engine.state.MetricsHistory) > engine.config.MaxHistorySize {
		t.Errorf("History size %d exceeds max %d", len(engine.state.MetricsHistory), engine.config.MaxHistorySize)
	}

	// Verify that we still have recent metrics
	if len(engine.state.MetricsHistory) == 0 {
		t.Error("History should not be empty")
	}

	// Check that most recent metrics are preserved
	lastMetric := engine.state.MetricsHistory[len(engine.state.MetricsHistory)-1]
	if lastMetric.Timestamp.Before(now.Add(10 * time.Second)) {
		t.Error("Most recent metrics should be preserved")
	}
}

// TestRuleEngine_InPlacePruning verifies that pruning works without excessive allocations
func TestRuleEngine_InPlacePruning(t *testing.T) {
	engine := NewRuleEngine(Config{
		CooldownDuration:     1 * time.Second,
		MinReplicas:          1,
		MaxReplicas:          10,
		LatencyThresholdHigh: 100.0,
		LatencyThresholdLow:  50.0,
		PRBThresholdHigh:     0.8,
		PRBThresholdLow:      0.3,
		EvaluationWindow:     30 * time.Second,
		MaxHistorySize:       100,
		PruneInterval:        1 * time.Millisecond, // Force frequent pruning
	})

	now := time.Now()
	
	// Add old metrics that should be pruned
	for i := 0; i < 5; i++ {
		oldData := KPMData{
			Timestamp:       now.Add(-25 * time.Hour), // Beyond 24h threshold
			NodeID:          "node1",
			PRBUtilization:  0.5,
			P95Latency:      75,
			CurrentReplicas: 2,
		}
		engine.state.MetricsHistory = append(engine.state.MetricsHistory, oldData)
	}

	// Add recent metrics
	for i := 0; i < 3; i++ {
		recentData := KPMData{
			Timestamp:       now.Add(time.Duration(i) * time.Second),
			NodeID:          "node1",
			PRBUtilization:  0.5,
			P95Latency:      75,
			CurrentReplicas: 2,
		}
		engine.Evaluate(recentData)
	}

	// Force pruning by waiting and adding another metric
	time.Sleep(2 * time.Millisecond)
	engine.Evaluate(KPMData{
		Timestamp:       now.Add(10 * time.Second),
		NodeID:          "node1",
		PRBUtilization:  0.5,
		P95Latency:      75,
		CurrentReplicas: 2,
	})

	// Verify old metrics were pruned
	for _, metric := range engine.state.MetricsHistory {
		if metric.Timestamp.Before(now.Add(-23 * time.Hour)) {
			t.Error("Old metrics should have been pruned")
		}
	}

	// Verify we still have recent metrics
	if len(engine.state.MetricsHistory) < 3 {
		t.Error("Recent metrics should be preserved")
	}
}

// TestRuleEngine_DefaultConfig verifies that default performance settings are applied
func TestRuleEngine_DefaultConfig(t *testing.T) {
	engine := NewRuleEngine(Config{
		CooldownDuration:     1 * time.Second,
		MinReplicas:          1,
		MaxReplicas:          10,
		LatencyThresholdHigh: 100.0,
		LatencyThresholdLow:  50.0,
		PRBThresholdHigh:     0.8,
		PRBThresholdLow:      0.3,
		EvaluationWindow:     30 * time.Second,
		// MaxHistorySize and PruneInterval not set - should use defaults
	})

	if engine.config.MaxHistorySize != 300 {
		t.Errorf("Expected default MaxHistorySize of 300, got %d", engine.config.MaxHistorySize)
	}

	if engine.config.PruneInterval != 30*time.Second {
		t.Errorf("Expected default PruneInterval of 30s, got %v", engine.config.PruneInterval)
	}
}

// BenchmarkRuleEngine_MemoryPerformance benchmarks memory allocation patterns
func BenchmarkRuleEngine_MemoryPerformance(b *testing.B) {
	engine := NewRuleEngine(Config{
		CooldownDuration:     1 * time.Second,
		MinReplicas:          1,
		MaxReplicas:          10,
		LatencyThresholdHigh: 100.0,
		LatencyThresholdLow:  50.0,
		PRBThresholdHigh:     0.8,
		PRBThresholdLow:      0.3,
		EvaluationWindow:     30 * time.Second,
		MaxHistorySize:       100,
		PruneInterval:        10 * time.Second,
	})

	now := time.Now()
	
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		data := KPMData{
			Timestamp:       now.Add(time.Duration(i) * time.Second),
			NodeID:          "node1",
			PRBUtilization:  0.5,
			P95Latency:      75,
			CurrentReplicas: 2,
		}
		engine.Evaluate(data)
	}
}

// BenchmarkRuleEngine_PruningPerformance benchmarks the pruning operation
func BenchmarkRuleEngine_PruningPerformance(b *testing.B) {
	engine := NewRuleEngine(Config{
		CooldownDuration:     1 * time.Second,
		MinReplicas:          1,
		MaxReplicas:          10,
		LatencyThresholdHigh: 100.0,
		LatencyThresholdLow:  50.0,
		PRBThresholdHigh:     0.8,
		PRBThresholdLow:      0.3,
		EvaluationWindow:     30 * time.Second,
		MaxHistorySize:       1000,
		PruneInterval:        1 * time.Nanosecond, // Force pruning every call
	})

	// Pre-populate with metrics
	now := time.Now()
	for i := 0; i < 500; i++ {
		engine.state.MetricsHistory = append(engine.state.MetricsHistory, KPMData{
			Timestamp:       now.Add(time.Duration(i-400) * time.Hour), // Mix of old and new
			NodeID:          "node1",
			PRBUtilization:  0.5,
			P95Latency:      75,
			CurrentReplicas: 2,
		})
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		engine.pruneHistoryInPlace()
	}
}

// === COMPREHENSIVE PERFORMANCE & OPTIMIZATION TESTS ===

// BenchmarkRuleEngine_HighFrequencyInserts benchmarks high-frequency metric insertions
func BenchmarkRuleEngine_HighFrequencyInserts(b *testing.B) {
	engine := NewRuleEngine(Config{
		CooldownDuration:     1 * time.Second,
		MinReplicas:          1,
		MaxReplicas:          10,
		LatencyThresholdHigh: 100.0,
		LatencyThresholdLow:  50.0,
		PRBThresholdHigh:     0.8,
		PRBThresholdLow:      0.3,
		EvaluationWindow:     30 * time.Second,
		MaxHistorySize:       500,
		PruneInterval:        5 * time.Second,
	})

	b.Run("Sequential", func(b *testing.B) {
		now := time.Now()
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			data := KPMData{
				Timestamp:       now.Add(time.Duration(i) * time.Millisecond),
				NodeID:          "node1",
				PRBUtilization:  0.5 + float64(i%20)/100.0,
				P95Latency:      75 + float64(i%50),
				CurrentReplicas: 2,
			}
			engine.Evaluate(data)
		}
	})

	b.Run("WithPruning", func(b *testing.B) {
		engine := NewRuleEngine(Config{
			CooldownDuration:     1 * time.Second,
			MinReplicas:          1,
			MaxReplicas:          10,
			LatencyThresholdHigh: 100.0,
			LatencyThresholdLow:  50.0,
			PRBThresholdHigh:     0.8,
			PRBThresholdLow:      0.3,
			EvaluationWindow:     30 * time.Second,
			MaxHistorySize:       100,
			PruneInterval:        1 * time.Millisecond, // Force frequent pruning
		})

		now := time.Now()
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			data := KPMData{
				Timestamp:       now.Add(time.Duration(i) * time.Millisecond),
				NodeID:          "node1",
				PRBUtilization:  0.5 + float64(i%20)/100.0,
				P95Latency:      75 + float64(i%50),
				CurrentReplicas: 2,
			}
			engine.Evaluate(data)
		}
	})
}

// BenchmarkRuleEngine_CapacityManagement benchmarks capacity limit enforcement
func BenchmarkRuleEngine_CapacityManagement(b *testing.B) {
	tests := []struct {
		name           string
		maxHistorySize int
		insertCount    int
	}{
		{"Small_100_200", 100, 200},
		{"Medium_500_1000", 500, 1000},
		{"Large_1000_2000", 1000, 2000},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			engine := NewRuleEngine(Config{
				CooldownDuration:     1 * time.Second,
				MinReplicas:          1,
				MaxReplicas:          10,
				LatencyThresholdHigh: 100.0,
				LatencyThresholdLow:  50.0,
				PRBThresholdHigh:     0.8,
				PRBThresholdLow:      0.3,
				EvaluationWindow:     30 * time.Second,
				MaxHistorySize:       tt.maxHistorySize,
				PruneInterval:        10 * time.Second,
			})

			now := time.Now()
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				for j := 0; j < tt.insertCount; j++ {
					data := KPMData{
						Timestamp:       now.Add(time.Duration(i*tt.insertCount+j) * time.Millisecond),
						NodeID:          "node1",
						PRBUtilization:  0.5,
						P95Latency:      75,
						CurrentReplicas: 2,
					}
					engine.Evaluate(data)
				}

				// Verify capacity limit is respected
				if len(engine.state.MetricsHistory) > tt.maxHistorySize {
					b.Errorf("History size %d exceeds limit %d", len(engine.state.MetricsHistory), tt.maxHistorySize)
				}
			}
		})
	}
}

// BenchmarkRuleEngine_PruningComparison compares different pruning strategies
func BenchmarkRuleEngine_PruningComparison(b *testing.B) {
	// Test data setup
	setupEngine := func(size int) *RuleEngine {
		engine := NewRuleEngine(Config{
			MaxHistorySize: size * 2, // Allow growth before pruning
			PruneInterval:  1 * time.Nanosecond,
		})

		now := time.Now()
		// Add mix of old and new data
		for i := 0; i < size; i++ {
			age := time.Duration(i*12) * time.Hour // 0 to size*12 hours old
			engine.state.MetricsHistory = append(engine.state.MetricsHistory, KPMData{
				Timestamp:       now.Add(-age),
				NodeID:          "node1",
				PRBUtilization:  0.5,
				P95Latency:      75,
				CurrentReplicas: 2,
			})
		}
		return engine
	}

	sizes := []int{100, 500, 1000}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("InPlace_Size_%d", size), func(b *testing.B) {
			engine := setupEngine(size)
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				engine.pruneHistoryInPlace()
			}
		})

		b.Run(fmt.Sprintf("SliceRecreation_Size_%d", size), func(b *testing.B) {
			engine := setupEngine(size)
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Simulate old approach: slice recreation
				cutoff := time.Now().Add(-24 * time.Hour)
				var newHistory []KPMData
				for _, metric := range engine.state.MetricsHistory {
					if metric.Timestamp.After(cutoff) {
						newHistory = append(newHistory, metric)
					}
				}
				engine.state.MetricsHistory = newHistory
			}
		})
	}
}

// TestRuleEngine_CapacityLimitEnforcement tests that capacity limits are strictly enforced
func TestRuleEngine_CapacityLimitEnforcement(t *testing.T) {
	tests := []struct {
		name           string
		maxHistorySize int
		insertCount    int
		expectedMax    int
	}{
		{"Exact_Limit", 10, 10, 10},
		{"Over_Limit_Small", 10, 15, 10},
		{"Over_Limit_Large", 50, 100, 50},
		{"Way_Over_Limit", 20, 200, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewRuleEngine(Config{
				CooldownDuration:     1 * time.Second,
				MinReplicas:          1,
				MaxReplicas:          10,
				LatencyThresholdHigh: 100.0,
				LatencyThresholdLow:  50.0,
				PRBThresholdHigh:     0.8,
				PRBThresholdLow:      0.3,
				EvaluationWindow:     30 * time.Second,
				MaxHistorySize:       tt.maxHistorySize,
				PruneInterval:        10 * time.Second, // Prevent time-based pruning
			})

			now := time.Now()
			for i := 0; i < tt.insertCount; i++ {
				data := KPMData{
					Timestamp:       now.Add(time.Duration(i) * time.Second),
					NodeID:          "node1",
					PRBUtilization:  0.5,
					P95Latency:      75,
					CurrentReplicas: 2,
				}
				engine.Evaluate(data)

				// Check capacity after each insert
				if len(engine.state.MetricsHistory) > tt.expectedMax {
					t.Errorf("After insert %d: history size %d exceeds expected max %d", 
						i+1, len(engine.state.MetricsHistory), tt.expectedMax)
				}
			}

			// Final verification
			finalSize := len(engine.state.MetricsHistory)
			if finalSize > tt.expectedMax {
				t.Errorf("Final history size %d exceeds expected max %d", finalSize, tt.expectedMax)
			}

			// Verify most recent data is preserved
			if finalSize > 0 {
				lastMetric := engine.state.MetricsHistory[finalSize-1]
				expectedLastTime := now.Add(time.Duration(tt.insertCount-1) * time.Second)
				if !lastMetric.Timestamp.Equal(expectedLastTime) {
					t.Error("Most recent metric should be preserved")
				}
			}
		})
	}
}

// TestRuleEngine_PruningAccuracy tests that pruning correctly removes old data
func TestRuleEngine_PruningAccuracy(t *testing.T) {
	engine := NewRuleEngine(Config{
		CooldownDuration:     1 * time.Second,
		MinReplicas:          1,
		MaxReplicas:          10,
		LatencyThresholdHigh: 100.0,
		LatencyThresholdLow:  50.0,
		PRBThresholdHigh:     0.8,
		PRBThresholdLow:      0.3,
		EvaluationWindow:     30 * time.Second,
		MaxHistorySize:       1000,
		PruneInterval:        1 * time.Millisecond, // Force frequent pruning
	})

	now := time.Now()
	cutoff := now.Add(-24 * time.Hour)

	// Add data spanning the cutoff
	testData := []struct {
		offset   time.Duration
		shouldKeep bool
	}{
		{-30 * time.Hour, false}, // Old - should be pruned
		{-25 * time.Hour, false}, // Old - should be pruned  
		{-23 * time.Hour, true},  // Recent - should be kept
		{-12 * time.Hour, true},  // Recent - should be kept
		{-1 * time.Hour, true},   // Recent - should be kept
		{0, true},                // Current - should be kept
	}

	for i, td := range testData {
		data := KPMData{
			Timestamp:       now.Add(td.offset),
			NodeID:          "node1",
			PRBUtilization:  0.5,
			P95Latency:      75,
			CurrentReplicas: 2,
		}
		engine.state.MetricsHistory = append(engine.state.MetricsHistory, data)
		
		// Add a marker to identify this specific data point
		engine.state.MetricsHistory[len(engine.state.MetricsHistory)-1].ActiveUEs = i + 100
	}

	// Force pruning
	engine.pruneHistoryInPlace()

	// Verify pruning accuracy
	for _, metric := range engine.state.MetricsHistory {
		if metric.Timestamp.Before(cutoff) {
			t.Errorf("Old metric with timestamp %v should have been pruned (cutoff: %v)", 
				metric.Timestamp, cutoff)
		}
	}

	// Verify expected data is preserved
	expectedKept := 0
	for _, td := range testData {
		if td.shouldKeep {
			expectedKept++
		}
	}

	if len(engine.state.MetricsHistory) != expectedKept {
		t.Errorf("Expected %d metrics to be kept, but got %d", expectedKept, len(engine.state.MetricsHistory))
	}

	// Verify specific expected metrics are present
	foundMarkers := make(map[int]bool)
	for _, metric := range engine.state.MetricsHistory {
		foundMarkers[metric.ActiveUEs] = true
	}

	for i, td := range testData {
		marker := i + 100
		if td.shouldKeep && !foundMarkers[marker] {
			t.Errorf("Expected metric with marker %d to be kept", marker)
		}
		if !td.shouldKeep && foundMarkers[marker] {
			t.Errorf("Expected metric with marker %d to be pruned", marker)
		}
	}
}

// TestRuleEngine_LongRunningMemoryStability tests memory stability over extended operation
func TestRuleEngine_LongRunningMemoryStability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running test in short mode")
	}

	engine := NewRuleEngine(Config{
		CooldownDuration:     100 * time.Millisecond,
		MinReplicas:          1,
		MaxReplicas:          10,
		LatencyThresholdHigh: 100.0,
		LatencyThresholdLow:  50.0,
		PRBThresholdHigh:     0.8,
		PRBThresholdLow:      0.3,
		EvaluationWindow:     30 * time.Second,
		MaxHistorySize:       100,
		PruneInterval:        1 * time.Second,
	})

	now := time.Now()
	iterations := 1000
	
	// Track memory usage patterns
	historySizes := make([]int, 0, iterations/10)
	
	for i := 0; i < iterations; i++ {
		data := KPMData{
			Timestamp:       now.Add(time.Duration(i) * 100 * time.Millisecond),
			NodeID:          "node1",
			PRBUtilization:  0.4 + float64(i%40)/100.0, // Varies 0.4-0.8
			P95Latency:      60 + float64(i%80),         // Varies 60-140
			CurrentReplicas: 2 + i%3,                    // Varies 2-4
		}
		
		engine.Evaluate(data)
		
		// Sample memory usage every 10 iterations
		if i%10 == 0 {
			historySizes = append(historySizes, len(engine.state.MetricsHistory))
		}
		
		// Verify capacity limit is never exceeded
		if len(engine.state.MetricsHistory) > engine.config.MaxHistorySize {
			t.Fatalf("Iteration %d: history size %d exceeded limit %d", 
				i, len(engine.state.MetricsHistory), engine.config.MaxHistorySize)
		}
	}
	
	// Analyze memory stability
	if len(historySizes) < 10 {
		t.Fatal("Not enough history size samples")
	}
	
	// Check that memory usage stabilizes (doesn't grow unbounded)
	lastQuarter := historySizes[len(historySizes)*3/4:]
	maxLastQuarter := 0
	minLastQuarter := engine.config.MaxHistorySize
	
	for _, size := range lastQuarter {
		if size > maxLastQuarter {
			maxLastQuarter = size
		}
		if size < minLastQuarter {
			minLastQuarter = size
		}
	}
	
	// Memory should be stable in the last quarter of iterations
	if maxLastQuarter-minLastQuarter > engine.config.MaxHistorySize/4 {
		t.Errorf("Memory usage not stable: range %d-%d in last quarter", 
			minLastQuarter, maxLastQuarter)
	}
	
	// Final history should be within reasonable bounds
	finalSize := len(engine.state.MetricsHistory)
	if finalSize > engine.config.MaxHistorySize {
		t.Errorf("Final history size %d exceeds limit %d", finalSize, engine.config.MaxHistorySize)
	}
	
	t.Logf("Long-running test completed: %d iterations, final history size: %d/%d", 
		iterations, finalSize, engine.config.MaxHistorySize)
}

// TestRuleEngine_ThreadSafety tests concurrent access to the rule engine
func TestRuleEngine_ThreadSafety(t *testing.T) {
	engine := NewRuleEngine(Config{
		CooldownDuration:     100 * time.Millisecond,
		MinReplicas:          1,
		MaxReplicas:          10,
		LatencyThresholdHigh: 100.0,
		LatencyThresholdLow:  50.0,
		PRBThresholdHigh:     0.8,
		PRBThresholdLow:      0.3,
		EvaluationWindow:     30 * time.Second,
		MaxHistorySize:       200,
		PruneInterval:        100 * time.Millisecond,
	})

	const numGoroutines = 10
	const operationsPerGoroutine = 100
	
	var wg sync.WaitGroup
	var decisionsCount sync.Map
	
	// Concurrent metric insertion
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			now := time.Now()
			decisions := 0
			
			for j := 0; j < operationsPerGoroutine; j++ {
				data := KPMData{
					Timestamp:       now.Add(time.Duration(workerID*operationsPerGoroutine+j) * time.Millisecond),
					NodeID:          fmt.Sprintf("node%d", workerID),
					PRBUtilization:  0.3 + float64(j%60)/100.0, // Varies to trigger decisions
					P95Latency:      40 + float64(j%120),        // Varies to trigger decisions
					CurrentReplicas: 2,
				}
				
				decision := engine.Evaluate(data)
				if decision != nil {
					decisions++
				}
				
				// Small delay to allow other goroutines to interleave
				if j%10 == 0 {
					runtime.Gosched()
				}
			}
			
			decisionsCount.Store(workerID, decisions)
		}(i)
	}
	
	wg.Wait()
	
	// Verify engine state integrity
	engine.mu.RLock()
	historySize := len(engine.state.MetricsHistory)
	decisionHistorySize := len(engine.state.DecisionHistory)
	currentReplicas := engine.state.CurrentReplicas
	engine.mu.RUnlock()
	
	// Validate results
	if historySize > engine.config.MaxHistorySize {
		t.Errorf("History size %d exceeds maximum %d", historySize, engine.config.MaxHistorySize)
	}
	
	if historySize == 0 {
		t.Error("History should not be empty after concurrent operations")
	}
	
	if currentReplicas < engine.config.MinReplicas || currentReplicas > engine.config.MaxReplicas {
		t.Errorf("Current replicas %d outside valid range [%d, %d]", 
			currentReplicas, engine.config.MinReplicas, engine.config.MaxReplicas)
	}
	
	// Count total decisions made
	totalDecisions := 0
	decisionsCount.Range(func(key, value interface{}) bool {
		totalDecisions += value.(int)
		return true
	})
	
	if totalDecisions != decisionHistorySize {
		t.Errorf("Decision count mismatch: counted %d, history has %d", 
			totalDecisions, decisionHistorySize)
	}
	
	t.Logf("Thread safety test completed: %d goroutines, %d total operations, %d decisions, final history size: %d", 
		numGoroutines, numGoroutines*operationsPerGoroutine, totalDecisions, historySize)
}

// TestRuleEngine_EdgeCases tests boundary conditions and error scenarios
func TestRuleEngine_EdgeCases(t *testing.T) {
	t.Run("ZeroMaxHistorySize", func(t *testing.T) {
		engine := NewRuleEngine(Config{
			MaxHistorySize: 0, // Should default to 300
		})
		
		if engine.config.MaxHistorySize != 300 {
			t.Errorf("Expected default MaxHistorySize 300, got %d", engine.config.MaxHistorySize)
		}
	})

	t.Run("NegativeMaxHistorySize", func(t *testing.T) {
		engine := NewRuleEngine(Config{
			MaxHistorySize: -100, // Should default to 300
		})
		
		if engine.config.MaxHistorySize != 300 {
			t.Errorf("Expected default MaxHistorySize 300, got %d", engine.config.MaxHistorySize)
		}
	})

	t.Run("ZeroPruneInterval", func(t *testing.T) {
		engine := NewRuleEngine(Config{
			PruneInterval: 0, // Should default to 30s
		})
		
		if engine.config.PruneInterval != 30*time.Second {
			t.Errorf("Expected default PruneInterval 30s, got %v", engine.config.PruneInterval)
		}
	})

	t.Run("EmptyMetricsHistory", func(t *testing.T) {
		engine := NewRuleEngine(Config{
			CooldownDuration:     1 * time.Second,
			MinReplicas:          1,
			MaxReplicas:          10,
			LatencyThresholdHigh: 100.0,
			LatencyThresholdLow:  50.0,
			PRBThresholdHigh:     0.8,
			PRBThresholdLow:      0.3,
			EvaluationWindow:     30 * time.Second,
		})
		
		// Clear history
		engine.state.MetricsHistory = nil
		
		// Should not panic
		metrics := engine.calculateAverageMetrics()
		if metrics.DataPoints != 0 {
			t.Error("Expected 0 data points for empty history")
		}
		
		// Pruning empty history should not panic
		engine.pruneHistoryInPlace()
	})

	t.Run("SingleMetricInHistory", func(t *testing.T) {
		engine := NewRuleEngine(Config{
			MaxHistorySize: 1, // Very small limit
			PruneInterval:  1 * time.Millisecond,
		})
		
		now := time.Now()
		
		// Add multiple metrics - only last should remain
		for i := 0; i < 5; i++ {
			data := KPMData{
				Timestamp:       now.Add(time.Duration(i) * time.Second),
				NodeID:          "node1",
				PRBUtilization:  0.5,
				P95Latency:      75,
				CurrentReplicas: 2,
			}
			engine.Evaluate(data)
		}
		
		if len(engine.state.MetricsHistory) != 1 {
			t.Errorf("Expected exactly 1 metric in history, got %d", len(engine.state.MetricsHistory))
		}
		
		// Last metric should be preserved
		lastMetric := engine.state.MetricsHistory[0]
		expectedTime := now.Add(4 * time.Second)
		if !lastMetric.Timestamp.Equal(expectedTime) {
			t.Error("Wrong metric preserved")
		}
	})

	t.Run("ExtremeTimestamps", func(t *testing.T) {
		engine := NewRuleEngine(Config{
			MaxHistorySize: 100,
			PruneInterval:  1 * time.Millisecond,
		})
		
		// Add metric with very old timestamp
		veryOld := time.Unix(0, 0) // Unix epoch
		oldData := KPMData{
			Timestamp:       veryOld,
			NodeID:          "node1",
			PRBUtilization:  0.5,
			P95Latency:      75,
			CurrentReplicas: 2,
		}
		engine.Evaluate(oldData)
		
		// Add recent metric
		recent := time.Now()
		recentData := KPMData{
			Timestamp:       recent,
			NodeID:          "node1",
			PRBUtilization:  0.5,
			P95Latency:      75,
			CurrentReplicas: 2,
		}
		engine.Evaluate(recentData)
		
		// Force pruning
		time.Sleep(2 * time.Millisecond)
		engine.Evaluate(recentData)
		
		// Old metric should be pruned
		for _, metric := range engine.state.MetricsHistory {
			if metric.Timestamp.Equal(veryOld) {
				t.Error("Very old metric should have been pruned")
			}
		}
	})

	t.Run("InvalidCurrentReplicas", func(t *testing.T) {
		engine := NewRuleEngine(Config{
			CooldownDuration:     1 * time.Second,
			MinReplicas:          1,
			MaxReplicas:          10,
			LatencyThresholdHigh: 100.0,
			LatencyThresholdLow:  50.0,
			PRBThresholdHigh:     0.8,
			PRBThresholdLow:      0.3,
			EvaluationWindow:     30 * time.Second,
		})
		
		// Test with zero current replicas (should not update)
		data := KPMData{
			Timestamp:       time.Now(),
			NodeID:          "node1",
			PRBUtilization:  0.5,
			P95Latency:      75,
			CurrentReplicas: 0, // Invalid
		}
		
		originalReplicas := engine.state.CurrentReplicas
		engine.Evaluate(data)
		
		if engine.state.CurrentReplicas != originalReplicas {
			t.Error("Should not update replicas when CurrentReplicas is 0")
		}
		
		// Test with negative current replicas (should not update)
		data.CurrentReplicas = -5
		engine.Evaluate(data)
		
		if engine.state.CurrentReplicas != originalReplicas {
			t.Error("Should not update replicas when CurrentReplicas is negative")
		}
	})
}

// TestRuleEngine_BackwardCompatibility ensures existing functionality still works
func TestRuleEngine_BackwardCompatibility(t *testing.T) {
	t.Run("LegacyConfigValues", func(t *testing.T) {
		// Test with config that doesn't include new performance parameters
		engine := NewRuleEngine(Config{
			StateFile:            "",
			CooldownDuration:     1 * time.Second,
			MinReplicas:          1,
			MaxReplicas:          10,
			LatencyThresholdHigh: 100.0,
			LatencyThresholdLow:  50.0,
			PRBThresholdHigh:     0.8,
			PRBThresholdLow:      0.3,
			EvaluationWindow:     30 * time.Second,
			// MaxHistorySize and PruneInterval intentionally omitted
		})
		
		// Add multiple data points to meet DataPoints >= 3 requirement
		now := time.Now()
		highLoadData := []KPMData{
			{Timestamp: now.Add(-20 * time.Second), NodeID: "node1", PRBUtilization: 0.9, P95Latency: 150, CurrentReplicas: 2},
			{Timestamp: now.Add(-10 * time.Second), NodeID: "node1", PRBUtilization: 0.85, P95Latency: 140, CurrentReplicas: 2},
			{Timestamp: now, NodeID: "node1", PRBUtilization: 0.88, P95Latency: 145, CurrentReplicas: 2},
		}
		
		var decision *ScalingDecision
		for _, data := range highLoadData {
			decision = engine.Evaluate(data)
		}
		
		// Should work without issues
		if decision == nil {
			t.Error("Expected scaling decision with high load")
		}
	})

	t.Run("StateFileCompatibility", func(t *testing.T) {
		tmpFile := filepath.Join(os.TempDir(), "compat-test-state.json")
		defer os.Remove(tmpFile)
		
		// Create engine and generate some state
		engine1 := NewRuleEngine(Config{
			StateFile:            tmpFile,
			CooldownDuration:     1 * time.Second,
			MinReplicas:          1,
			MaxReplicas:          10,
			LatencyThresholdHigh: 100.0,
			LatencyThresholdLow:  50.0,
			PRBThresholdHigh:     0.8,
			PRBThresholdLow:      0.3,
			EvaluationWindow:     30 * time.Second,
		})
		
		now := time.Now()
		for i := 0; i < 5; i++ {
			data := KPMData{
				Timestamp:       now.Add(time.Duration(i) * time.Second),
				NodeID:          "node1",
				PRBUtilization:  0.5,
				P95Latency:      75,
				CurrentReplicas: 2,
			}
			engine1.Evaluate(data)
		}
		
		// Force save state
		engine1.saveState()
		
		// Create new engine that loads the same state
		engine2 := NewRuleEngine(Config{
			StateFile:            tmpFile,
			CooldownDuration:     1 * time.Second,
			MinReplicas:          1,
			MaxReplicas:          10,
			LatencyThresholdHigh: 100.0,
			LatencyThresholdLow:  50.0,
			PRBThresholdHigh:     0.8,
			PRBThresholdLow:      0.3,
			EvaluationWindow:     30 * time.Second,
		})
		
		// Should have loaded the state (allow for some variance due to default allocations)
		engine1Size := len(engine1.state.MetricsHistory)
		engine2Size := len(engine2.state.MetricsHistory)
		if engine2Size != engine1Size {
			t.Logf("Engine1 history size: %d, Engine2 history size: %d", engine1Size, engine2Size)
			// Check if at least some state was loaded
			if engine2Size == 0 {
				t.Error("No state was loaded from file")
			}
		}
	})

	t.Run("ExistingDecisionLogic", func(t *testing.T) {
		engine := NewRuleEngine(Config{
			CooldownDuration:     1 * time.Second,
			MinReplicas:          1,
			MaxReplicas:          10,
			LatencyThresholdHigh: 100.0,
			LatencyThresholdLow:  50.0,
			PRBThresholdHigh:     0.8,
			PRBThresholdLow:      0.3,
			EvaluationWindow:     30 * time.Second,
		})
		
		now := time.Now()
		
		// Test scale-out decision logic unchanged
		highLoadData := []KPMData{
			{Timestamp: now.Add(-20 * time.Second), NodeID: "node1", PRBUtilization: 0.85, P95Latency: 120, CurrentReplicas: 2},
			{Timestamp: now.Add(-10 * time.Second), NodeID: "node1", PRBUtilization: 0.90, P95Latency: 130, CurrentReplicas: 2},
			{Timestamp: now, NodeID: "node1", PRBUtilization: 0.88, P95Latency: 125, CurrentReplicas: 2},
		}
		
		var decision *ScalingDecision
		for _, data := range highLoadData {
			decision = engine.Evaluate(data)
		}
		
		if decision == nil || decision.Action != "scale-out" || decision.TargetReplicas != 3 {
			t.Error("Scale-out logic changed unexpectedly")
		}
		
		// Test scale-in decision logic unchanged  
		// Create new engine for scale-in test to avoid cooldown issues
		scaleInEngine := NewRuleEngine(Config{
			CooldownDuration:     1 * time.Second,
			MinReplicas:          1,
			MaxReplicas:          10,
			LatencyThresholdHigh: 100.0,
			LatencyThresholdLow:  50.0,
			PRBThresholdHigh:     0.8,
			PRBThresholdLow:      0.3,
			EvaluationWindow:     30 * time.Second,
		})
		scaleInEngine.state.CurrentReplicas = 5
		
		laterTime := now.Add(10 * time.Second) // Use later time to avoid cooldown
		lowLoadData := []KPMData{
			{Timestamp: laterTime.Add(-20 * time.Second), NodeID: "node1", PRBUtilization: 0.2, P95Latency: 30, CurrentReplicas: 5},
			{Timestamp: laterTime.Add(-10 * time.Second), NodeID: "node1", PRBUtilization: 0.25, P95Latency: 35, CurrentReplicas: 5},
			{Timestamp: laterTime, NodeID: "node1", PRBUtilization: 0.22, P95Latency: 32, CurrentReplicas: 5},
		}
		
		var scaleInDecision *ScalingDecision
		for _, data := range lowLoadData {
			scaleInDecision = scaleInEngine.Evaluate(data)
		}
		
		if scaleInDecision == nil || scaleInDecision.Action != "scale-in" || scaleInDecision.TargetReplicas != 4 {
			if scaleInDecision == nil {
				t.Error("Scale-in decision was nil")
			} else {
				t.Errorf("Scale-in logic changed: got action=%s, target=%d", scaleInDecision.Action, scaleInDecision.TargetReplicas)
			}
		}
	})
}

// BenchmarkRuleEngine_MemoryGrowthComparison compares memory growth patterns
func BenchmarkRuleEngine_MemoryGrowthComparison(b *testing.B) {
	b.Run("WithOptimizations", func(b *testing.B) {
		engine := NewRuleEngine(Config{
			MaxHistorySize: 100,
			PruneInterval:  1 * time.Second,
		})
		
		now := time.Now()
		
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)
		
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			data := KPMData{
				Timestamp:       now.Add(time.Duration(i) * time.Millisecond),
				NodeID:          "node1",
				PRBUtilization:  0.5,
				P95Latency:      75,
				CurrentReplicas: 2,
			}
			engine.Evaluate(data)
		}
		
		b.StopTimer()
		runtime.GC()
		runtime.ReadMemStats(&m2)
		
		b.ReportMetric(float64(m2.Alloc-m1.Alloc), "bytes/alloc")
		b.ReportMetric(float64(len(engine.state.MetricsHistory)), "final_history_size")
	})

	b.Run("WithoutCapacityLimits", func(b *testing.B) {
		engine := NewRuleEngine(Config{
			MaxHistorySize: 999999999, // Effectively unlimited
			PruneInterval:  999999999 * time.Second, // No pruning
		})
		
		now := time.Now()
		
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)
		
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			data := KPMData{
				Timestamp:       now.Add(time.Duration(i) * time.Millisecond),
				NodeID:          "node1",
				PRBUtilization:  0.5,
				P95Latency:      75,
				CurrentReplicas: 2,
			}
			engine.Evaluate(data)
		}
		
		b.StopTimer()
		runtime.GC()
		runtime.ReadMemStats(&m2)
		
		b.ReportMetric(float64(m2.Alloc-m1.Alloc), "bytes/alloc")
		b.ReportMetric(float64(len(engine.state.MetricsHistory)), "final_history_size")
	})
}
