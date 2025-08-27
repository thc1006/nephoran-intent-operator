//go:build performance
// +build performance

package rules

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// StressTestRuleEngine_HighFrequencyInserts performs stress testing with extremely high insertion rates
func StressTestRuleEngine_HighFrequencyInserts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	tests := []struct {
		name             string
		insertionsPerSec int
		duration         time.Duration
		maxHistorySize   int
		pruneInterval    time.Duration
	}{
		{"Moderate_1000_10s", 1000, 10 * time.Second, 500, 1 * time.Second},
		{"High_5000_10s", 5000, 10 * time.Second, 1000, 500 * time.Millisecond},
		{"Extreme_10000_5s", 10000, 5 * time.Second, 2000, 100 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewRuleEngine(Config{
				CooldownDuration:     50 * time.Millisecond,
				MinReplicas:          1,
				MaxReplicas:          20,
				LatencyThresholdHigh: 100.0,
				LatencyThresholdLow:  50.0,
				PRBThresholdHigh:     0.8,
				PRBThresholdLow:      0.3,
				EvaluationWindow:     30 * time.Second,
				MaxHistorySize:       tt.maxHistorySize,
				PruneInterval:        tt.pruneInterval,
			})

			var totalInsertions int64
			var totalDecisions int64
			var maxHistoryObserved int64
			var errors int64

			interval := time.Second / time.Duration(tt.insertionsPerSec)
			ctx, cancel := context.WithTimeout(context.Background(), tt.duration)
			defer cancel()

			start := time.Now()
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			var wg sync.WaitGroup
			wg.Add(1)

			go func() {
				defer wg.Done()
				i := 0
				for {
					select {
					case <-ctx.Done():
						return
					case now := <-ticker.C:
						data := KPMData{
							Timestamp:       now,
							NodeID:          "stress-node",
							PRBUtilization:  0.3 + float64(i%70)/100.0, // Varies 0.3-1.0
							P95Latency:      30 + float64(i%150),       // Varies 30-180
							CurrentReplicas: 2 + i%8,                   // Varies 2-9
						}

						decision := engine.Evaluate(data)
						atomic.AddInt64(&totalInsertions, 1)

						if decision != nil {
							atomic.AddInt64(&totalDecisions, 1)
						}

						// Track maximum history size observed
						engine.mu.RLock()
						currentSize := int64(len(engine.state.MetricsHistory))
						engine.mu.RUnlock()

						for {
							current := atomic.LoadInt64(&maxHistoryObserved)
							if currentSize <= current || atomic.CompareAndSwapInt64(&maxHistoryObserved, current, currentSize) {
								break
							}
						}

						// Check for capacity violations
						if currentSize > int64(tt.maxHistorySize) {
							atomic.AddInt64(&errors, 1)
						}

						i++
					}
				}
			}()

			wg.Wait()
			elapsed := time.Since(start)

			finalInsertions := atomic.LoadInt64(&totalInsertions)
			finalDecisions := atomic.LoadInt64(&totalDecisions)
			finalMaxHistory := atomic.LoadInt64(&maxHistoryObserved)
			finalErrors := atomic.LoadInt64(&errors)

			actualRate := float64(finalInsertions) / elapsed.Seconds()

			t.Logf("Stress Test Results:")
			t.Logf("  Duration: %v", elapsed)
			t.Logf("  Total Insertions: %d", finalInsertions)
			t.Logf("  Actual Rate: %.1f insertions/sec", actualRate)
			t.Logf("  Total Decisions: %d", finalDecisions)
			t.Logf("  Max History Size Observed: %d/%d", finalMaxHistory, tt.maxHistorySize)
			t.Logf("  Capacity Violations: %d", finalErrors)

			// Validate results
			if finalErrors > 0 {
				t.Errorf("Capacity violations detected: %d", finalErrors)
			}

			if finalInsertions == 0 {
				t.Error("No insertions completed")
			}

			// Check if we achieved reasonable insertion rate (allow 10% variance)
			expectedInsertions := int64(float64(tt.insertionsPerSec) * elapsed.Seconds())
			variance := float64(finalInsertions-expectedInsertions) / float64(expectedInsertions)
			if variance < -0.3 { // Allow for system load variations
				t.Errorf("Insertion rate too low: expected ~%d, got %d (%.1f%% variance)",
					expectedInsertions, finalInsertions, variance*100)
			}

			// Final state check
			engine.mu.RLock()
			finalHistorySize := len(engine.state.MetricsHistory)
			engine.mu.RUnlock()

			if finalHistorySize > tt.maxHistorySize {
				t.Errorf("Final history size %d exceeds limit %d", finalHistorySize, tt.maxHistorySize)
			}
		})
	}
}

// BenchmarkRuleEngine_ExtremeConcurrency tests performance under high concurrency
func BenchmarkRuleEngine_ExtremeConcurrency(b *testing.B) {
	concurrencyLevels := []int{1, 4, 8, 16, 32}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(b *testing.B) {
			engine := NewRuleEngine(Config{
				CooldownDuration:     10 * time.Millisecond,
				MinReplicas:          1,
				MaxReplicas:          20,
				LatencyThresholdHigh: 100.0,
				LatencyThresholdLow:  50.0,
				PRBThresholdHigh:     0.8,
				PRBThresholdLow:      0.3,
				EvaluationWindow:     30 * time.Second,
				MaxHistorySize:       1000,
				PruneInterval:        100 * time.Millisecond,
			})

			b.SetParallelism(concurrency)
			b.ResetTimer()
			b.ReportAllocs()

			b.RunParallel(func(pb *testing.PB) {
				workerID := 0
				for pb.Next() {
					data := KPMData{
						Timestamp:       time.Now(),
						NodeID:          fmt.Sprintf("node-%d", workerID%10),
						PRBUtilization:  0.3 + float64(workerID%60)/100.0,
						P95Latency:      40 + float64(workerID%120),
						CurrentReplicas: 2 + workerID%8,
					}
					engine.Evaluate(data)
					workerID++
				}
			})

			b.StopTimer()

			// Report final state
			engine.mu.RLock()
			finalSize := len(engine.state.MetricsHistory)
			engine.mu.RUnlock()

			b.ReportMetric(float64(finalSize), "final_history_size")
			b.ReportMetric(float64(concurrency), "goroutines")
		})
	}
}

// BenchmarkRuleEngine_MemoryAllocationPatterns analyzes memory allocation patterns
func BenchmarkRuleEngine_MemoryAllocationPatterns(b *testing.B) {
	scenarios := []struct {
		name           string
		maxHistorySize int
		pruneInterval  time.Duration
		description    string
	}{
		{"OptimizedSmall", 50, 1 * time.Second, "Small history with regular pruning"},
		{"OptimizedMedium", 200, 2 * time.Second, "Medium history with moderate pruning"},
		{"OptimizedLarge", 1000, 5 * time.Second, "Large history with infrequent pruning"},
		{"UnoptimizedGrowth", 999999, 999 * time.Hour, "Unlimited growth (no optimization)"},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			engine := NewRuleEngine(Config{
				MaxHistorySize: scenario.maxHistorySize,
				PruneInterval:  scenario.pruneInterval,
			})

			var m1, m2 runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&m1)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				data := KPMData{
					Timestamp:       time.Now().Add(time.Duration(i) * time.Millisecond),
					NodeID:          "perf-node",
					PRBUtilization:  0.5 + float64(i%40)/100.0,
					P95Latency:      60 + float64(i%80),
					CurrentReplicas: 2 + i%6,
				}
				engine.Evaluate(data)

				// Periodically force GC to measure actual memory usage
				if i%1000 == 0 {
					runtime.GC()
				}
			}

			b.StopTimer()
			runtime.GC()
			runtime.ReadMemStats(&m2)

			// Report metrics
			allocBytes := m2.Alloc - m1.Alloc
			totalAllocs := m2.TotalAlloc - m1.TotalAlloc
			finalHistorySize := len(engine.state.MetricsHistory)

			b.ReportMetric(float64(allocBytes), "bytes/retained")
			b.ReportMetric(float64(totalAllocs), "bytes/total")
			b.ReportMetric(float64(finalHistorySize), "final_history_size")
			b.ReportMetric(float64(allocBytes)/float64(b.N), "bytes_per_operation")

			b.Logf("%s: %d operations, %d bytes retained, %d final history size",
				scenario.description, b.N, allocBytes, finalHistorySize)
		})
	}
}

// BenchmarkRuleEngine_PruningEfficiency measures pruning operation efficiency
func BenchmarkRuleEngine_PruningEfficiency(b *testing.B) {
	historySizes := []int{100, 500, 1000, 5000}
	pruneRatios := []float64{0.1, 0.3, 0.5, 0.7, 0.9} // Fraction of data to be pruned

	for _, historySize := range historySizes {
		for _, pruneRatio := range pruneRatios {
			pruneCount := int(float64(historySize) * pruneRatio)
			keepCount := historySize - pruneCount

			b.Run(fmt.Sprintf("Size_%d_Prune_%d%%", historySize, int(pruneRatio*100)), func(b *testing.B) {
				engine := NewRuleEngine(Config{
					MaxHistorySize: historySize * 2, // Prevent capacity-based pruning
					PruneInterval:  1 * time.Nanosecond,
				})

				// Setup test data
				now := time.Now()
				for i := 0; i < historySize; i++ {
					var timestamp time.Time
					if i < pruneCount {
						// Old data (will be pruned)
						timestamp = now.Add(-25 * time.Hour)
					} else {
						// Recent data (will be kept)
						timestamp = now.Add(-time.Duration(i-pruneCount) * time.Minute)
					}

					engine.state.MetricsHistory = append(engine.state.MetricsHistory, KPMData{
						Timestamp:       timestamp,
						NodeID:          "bench-node",
						PRBUtilization:  0.5,
						P95Latency:      75,
						CurrentReplicas: 2,
					})
				}

				b.ResetTimer()
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					// Reset data for each iteration
					if i > 0 {
						engine.state.MetricsHistory = engine.state.MetricsHistory[:historySize]
					}
					engine.pruneHistoryInPlace()
				}

				b.StopTimer()

				// Verify pruning worked correctly
				finalSize := len(engine.state.MetricsHistory)
				if finalSize != keepCount {
					b.Errorf("Expected %d items after pruning, got %d", keepCount, finalSize)
				}

				b.ReportMetric(float64(historySize), "initial_size")
				b.ReportMetric(float64(pruneCount), "pruned_count")
				b.ReportMetric(float64(finalSize), "final_size")
			})
		}
	}
}

// StressTestRuleEngine_MemoryLeaks performs long-running test to detect memory leaks
func StressTestRuleEngine_MemoryLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	engine := NewRuleEngine(Config{
		CooldownDuration:     10 * time.Millisecond,
		MinReplicas:          1,
		MaxReplicas:          10,
		LatencyThresholdHigh: 100.0,
		LatencyThresholdLow:  50.0,
		PRBThresholdHigh:     0.8,
		PRBThresholdLow:      0.3,
		EvaluationWindow:     30 * time.Second,
		MaxHistorySize:       200,
		PruneInterval:        1 * time.Second,
	})

	const testDuration = 30 * time.Second
	const samplingInterval = 2 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()

	var memSamples []uint64
	var historySizeSamples []int

	// Memory sampling goroutine
	var samplingWg sync.WaitGroup
	samplingWg.Add(1)
	go func() {
		defer samplingWg.Done()
		ticker := time.NewTicker(samplingInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				var m runtime.MemStats
				runtime.GC()
				runtime.ReadMemStats(&m)
				memSamples = append(memSamples, m.Alloc)

				engine.mu.RLock()
				historySizeSamples = append(historySizeSamples, len(engine.state.MetricsHistory))
				engine.mu.RUnlock()
			}
		}
	}()

	// Metric insertion loop
	var insertionWg sync.WaitGroup
	insertionWg.Add(1)
	go func() {
		defer insertionWg.Done()
		i := 0
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				data := KPMData{
					Timestamp:       now,
					NodeID:          "leak-test-node",
					PRBUtilization:  0.3 + float64(i%60)/100.0,
					P95Latency:      40 + float64(i%100),
					CurrentReplicas: 2 + i%6,
				}
				engine.Evaluate(data)
				i++
			}
		}
	}()

	insertionWg.Wait()
	samplingWg.Wait()

	// Analyze memory samples for leaks
	if len(memSamples) < 3 {
		t.Fatal("Not enough memory samples collected")
	}

	// Check for sustained memory growth
	firstHalf := memSamples[:len(memSamples)/2]
	secondHalf := memSamples[len(memSamples)/2:]

	var firstHalfAvg, secondHalfAvg uint64
	for _, sample := range firstHalf {
		firstHalfAvg += sample
	}
	firstHalfAvg /= uint64(len(firstHalf))

	for _, sample := range secondHalf {
		secondHalfAvg += sample
	}
	secondHalfAvg /= uint64(len(secondHalf))

	// Allow for some growth, but not excessive
	growthRatio := float64(secondHalfAvg) / float64(firstHalfAvg)

	t.Logf("Memory Leak Analysis:")
	t.Logf("  Test Duration: %v", testDuration)
	t.Logf("  Memory Samples: %d", len(memSamples))
	t.Logf("  First Half Avg: %d bytes", firstHalfAvg)
	t.Logf("  Second Half Avg: %d bytes", secondHalfAvg)
	t.Logf("  Growth Ratio: %.2f", growthRatio)

	// Check history size stability
	var historyAvg float64
	for _, size := range historySizeSamples {
		historyAvg += float64(size)
	}
	historyAvg /= float64(len(historySizeSamples))

	t.Logf("  History Size Samples: %d", len(historySizeSamples))
	t.Logf("  History Size Average: %.1f", historyAvg)

	// Validate no significant memory leak (allow up to 50% growth due to system variations)
	if growthRatio > 1.5 {
		t.Errorf("Potential memory leak detected: %.2f growth ratio", growthRatio)
	}

	// Validate history size is controlled
	if historyAvg > float64(engine.config.MaxHistorySize)*1.1 {
		t.Errorf("History size not properly controlled: %.1f avg vs %d limit",
			historyAvg, engine.config.MaxHistorySize)
	}
}

// BenchmarkRuleEngine_ConfigurationImpact measures impact of different configuration parameters
func BenchmarkRuleEngine_ConfigurationImpact(b *testing.B) {
	configs := []struct {
		name           string
		maxHistorySize int
		pruneInterval  time.Duration
	}{
		{"Aggressive_50_100ms", 50, 100 * time.Millisecond},
		{"Moderate_200_1s", 200, 1 * time.Second},
		{"Conservative_1000_10s", 1000, 10 * time.Second},
		{"Minimal_10_10ms", 10, 10 * time.Millisecond},
	}

	for _, config := range configs {
		b.Run(config.name, func(b *testing.B) {
			engine := NewRuleEngine(Config{
				CooldownDuration:     100 * time.Millisecond,
				MinReplicas:          1,
				MaxReplicas:          10,
				LatencyThresholdHigh: 100.0,
				LatencyThresholdLow:  50.0,
				PRBThresholdHigh:     0.8,
				PRBThresholdLow:      0.3,
				EvaluationWindow:     30 * time.Second,
				MaxHistorySize:       config.maxHistorySize,
				PruneInterval:        config.pruneInterval,
			})

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				data := KPMData{
					Timestamp:       time.Now().Add(time.Duration(i) * time.Millisecond),
					NodeID:          "config-test-node",
					PRBUtilization:  0.4 + float64(i%50)/100.0,
					P95Latency:      50 + float64(i%100),
					CurrentReplicas: 2 + i%6,
				}
				engine.Evaluate(data)
			}

			finalHistorySize := len(engine.state.MetricsHistory)
			b.ReportMetric(float64(finalHistorySize), "final_history_size")
			b.ReportMetric(float64(config.maxHistorySize), "max_history_limit")
		})
	}
}
