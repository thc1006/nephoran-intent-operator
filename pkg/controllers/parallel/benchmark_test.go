/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package parallel

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/thc1006/nephoran-intent-operator/api/intent/v1alpha1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/resilience"
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"
)

// Benchmark configuration constants
const (
	benchmarkTimeout    = 5 * time.Minute
	warmupIntents       = 10
	measurementDuration = 30 * time.Second
)

// setupBenchmarkEngine creates a high-performance engine for benchmarking
func setupBenchmarkEngine() (*ParallelProcessingEngine, func(), error) {
	zapLogger, _ := zap.NewDevelopment()
	logger := zapr.NewLogger(zapLogger)

	ctx, cancel := context.WithCancel(context.Background())

	// High-performance resilience configuration
	resilienceConfig := &resilience.ResilienceConfig{
		DefaultTimeout:          30 * time.Second,
		MaxConcurrentOperations: 500,
		HealthCheckInterval:     30 * time.Second,
		TimeoutEnabled:          true,
		BulkheadEnabled:         true,
		CircuitBreakerEnabled:   false, // Disable for pure performance testing
		RateLimitingEnabled:     false, // Disable for pure performance testing
		RetryEnabled:            false, // Disable for consistent timing
		HealthCheckEnabled:      true,
	}

	resilienceMgr := resilience.NewResilienceManager(resilienceConfig, logger)
	if err := resilienceMgr.Start(ctx); err != nil {
		cancel()
		return nil, nil, err
	}

	// Minimal error tracking for performance
	errorConfig := &monitoring.ErrorTrackingConfig{
		EnablePrometheus:    false, // Disable for performance
		EnableOpenTelemetry: false, // Disable for performance
		AlertingEnabled:     false,
		DashboardEnabled:    false,
		ReportsEnabled:      false,
	}

	// Remove unused errorTracker

	// High-capacity engine configuration
	engineConfig := &ProcessingEngineConfig{
		MaxConcurrentIntents: 200,
		IntentWorkers:       25,
		LLMWorkers:          15,
		RAGWorkers:          15,
		ResourceWorkers:     20,
		ManifestWorkers:     20,
		GitOpsWorkers:       10,
		DeploymentWorkers:   10,
		MaxQueueSize:        1000,
		HealthCheckInterval:  30 * time.Second,
	}

	engine := NewParallelProcessingEngine(
		engineConfig,
		logger,
	)

	if err := engine.Start(ctx); err != nil {
		resilienceMgr.Stop()
		cancel()
		return nil, nil, err
	}

	cleanup := func() {
		engine.Stop()
		resilienceMgr.Stop()
		cancel()
	}

	return engine, cleanup, nil
}

// BenchmarkSingleIntentProcessing benchmarks processing a single intent
func BenchmarkSingleIntentProcessing(b *testing.B) {
	engine, cleanup, err := setupBenchmarkEngine()
	if err != nil {
		b.Fatalf("Failed to setup benchmark engine: %v", err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), benchmarkTimeout)
	defer cancel()

	// Warmup
	for i := 0; i < warmupIntents; i++ {
		intent := &v1alpha1.NetworkIntent{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("warmup-%d", i), Namespace: "benchmark"},
			Spec: v1alpha1.NetworkIntentSpec{
				IntentType: "simple_deployment",
				Intent:     "Deploy simple network service",
				Priority:   "medium",
			},
		}
		engine.ProcessIntentWorkflow(ctx, intent)
	}

	// Reset timer after warmup
	b.ResetTimer()

	// Benchmark single intent processing
	for i := 0; i < b.N; i++ {
		intent := &v1alpha1.NetworkIntent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("bench-intent-%d", i),
				Namespace: "benchmark",
			},
			Spec: v1alpha1.NetworkIntentSpec{
				IntentType: "benchmark_deployment",
				Intent:     fmt.Sprintf("Benchmark intent %d", i),
				Priority:   "medium",
			},
		}

		result, err := engine.ProcessIntentWorkflow(ctx, intent)
		if err != nil {
			b.Logf("Intent %d failed: %v", i, err)
		}
		if result != nil && !result.Success {
			b.Logf("Intent %d unsuccessful", i)
		}
	}

	// Report final metrics
	metrics := engine.GetMetrics()
	b.Logf("Final metrics: TotalTasks=%d, SuccessRate=%.2f%%, AvgLatency=%v",
		metrics.TotalTasks, metrics.SuccessRate*100, metrics.AverageLatency)
}

// BenchmarkConcurrentIntentProcessing benchmarks concurrent intent processing
func BenchmarkConcurrentIntentProcessing(b *testing.B) {
	engine, cleanup, err := setupBenchmarkEngine()
	if err != nil {
		b.Fatalf("Failed to setup benchmark engine: %v", err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), benchmarkTimeout)
	defer cancel()

	// Test different concurrency levels
	concurrencyLevels := []int{1, 5, 10, 25, 50, 100}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency-%d", concurrency), func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				startTime := time.Now()

				for j := 0; j < concurrency; j++ {
					wg.Add(1)
					go func(intentNum int) {
						defer wg.Done()

						intent := &v1alpha1.NetworkIntent{
							ObjectMeta: metav1.ObjectMeta{
								Name:      fmt.Sprintf("concurrent-%d-%d", i, intentNum),
								Namespace: "benchmark",
							},
							Spec: v1alpha1.NetworkIntentSpec{
								IntentType: "concurrent_test",
								Intent:     fmt.Sprintf("Concurrent intent %d-%d", i, intentNum),
								Priority:   "medium",
							},
						}

						engine.ProcessIntentWorkflow(ctx, intent)
					}(j)
				}

				wg.Wait()
				duration := time.Since(startTime)

				b.Logf("Batch %d with %d concurrent intents completed in %v", i, concurrency, duration)
			}
		})
	}
}

// BenchmarkTaskSubmission benchmarks raw task submission performance
func BenchmarkTaskSubmission(b *testing.B) {
	engine, cleanup, err := setupBenchmarkEngine()
	if err != nil {
		b.Fatalf("Failed to setup benchmark engine: %v", err)
	}
	defer cleanup()

	b.ResetTimer()

	// Benchmark task submission rate
	for i := 0; i < b.N; i++ {
		task := &Task{
			ID:        fmt.Sprintf("bench-task-%d", i),
			IntentID:  "benchmark-intent",
			Type:      TaskTypeLLMProcessing,
			Priority:  1,
			Status:    TaskStatusPending,
			InputData: map[string]interface{}{"intent": "benchmark task"},
			Timeout:   10 * time.Second,
		}

		err := engine.SubmitTask(task)
		if err != nil {
			b.Logf("Task submission failed: %v", err)
		}
	}

	// Wait for some processing to complete
	time.Sleep(2 * time.Second)

	metrics := engine.GetMetrics()
	b.Logf("Task submission benchmark: %d tasks submitted, %d processed",
		b.N, metrics.TotalTasks)
}

// BenchmarkMemoryAllocation benchmarks memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	engine, cleanup, err := setupBenchmarkEngine()
	if err != nil {
		b.Fatalf("Failed to setup benchmark engine: %v", err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), benchmarkTimeout)
	defer cancel()

	// Capture initial memory stats
	var m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	b.ResetTimer()

	// Benchmark with memory allocation tracking
	for i := 0; i < b.N; i++ {
		intent := &v1alpha1.NetworkIntent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("memory-bench-%d", i),
				Namespace: "benchmark",
			},
			Spec: v1alpha1.NetworkIntentSpec{
				IntentType: "memory_test",
				Intent:     fmt.Sprintf("Memory allocation test intent %d", i),
				Priority:   "medium",
			},
		}

		engine.ProcessIntentWorkflow(ctx, intent)

		// Periodic GC to measure actual allocation
		if i%100 == 0 {
			runtime.GC()
		}
	}

	// Capture final memory stats
	var m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m2)

	b.StopTimer()

	// Report memory allocation statistics
	allocatedMB := float64(m2.TotalAlloc-m1.TotalAlloc) / 1024 / 1024
	allocsPerOp := float64(m2.Mallocs-m1.Mallocs) / float64(b.N)

	b.Logf("Memory allocated: %.2f MB", allocatedMB)
	b.Logf("Allocations per operation: %.2f", allocsPerOp)
	b.Logf("Peak heap size: %.2f MB", float64(m2.HeapSys)/1024/1024)
}

// BenchmarkThroughput measures sustained throughput over time
func BenchmarkThroughput(b *testing.B) {
	engine, cleanup, err := setupBenchmarkEngine()
	if err != nil {
		b.Fatalf("Failed to setup benchmark engine: %v", err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), benchmarkTimeout)
	defer cancel()

	// Run sustained load test
	duration := measurementDuration
	startTime := time.Now()
	endTime := startTime.Add(duration)

	var intentCount int64
	var wg sync.WaitGroup

	b.ResetTimer()

	// Continuous intent submission
	for time.Now().Before(endTime) {
		wg.Add(1)
		go func(num int64) {
			defer wg.Done()

			intent := &v1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("throughput-%d", num),
					Namespace: "benchmark",
				},
				Spec: v1alpha1.NetworkIntentSpec{
					IntentType: "throughput_test",
					Intent:     fmt.Sprintf("Throughput test intent %d", num),
					Priority:   "medium",
				},
			}

			engine.ProcessIntentWorkflow(ctx, intent)
		}(intentCount)

		intentCount++

		// Rate limiting to prevent overwhelming the system
		time.Sleep(50 * time.Millisecond)
	}

	wg.Wait()
	actualDuration := time.Since(startTime)

	b.StopTimer()

	// Calculate throughput metrics
	throughput := float64(intentCount) / actualDuration.Seconds()

	metrics := engine.GetMetrics()
	b.Logf("Throughput test results:")
	b.Logf("  Duration: %v", actualDuration)
	b.Logf("  Intents submitted: %d", intentCount)
	b.Logf("  Throughput: %.2f intents/second", throughput)
	b.Logf("  Tasks processed: %d", metrics.TotalTasks)
	b.Logf("  Success rate: %.2f%%", metrics.SuccessRate*100)
	b.Logf("  Average latency: %v", metrics.AverageLatency)
}

// BenchmarkWorkerPoolScaling benchmarks worker pool scaling behavior
func BenchmarkWorkerPoolScaling(b *testing.B) {
	// Test different worker pool configurations
	poolConfigs := []struct {
		name         string
		intentPool   int
		llmPool      int
		resourcePool int
	}{
		{"Small", 5, 3, 4},
		{"Medium", 10, 6, 8},
		{"Large", 20, 12, 16},
		{"XLarge", 30, 18, 24},
	}

	for _, config := range poolConfigs {
		b.Run(config.name, func(b *testing.B) {
			// Create engine with specific pool configuration
			zapLogger, _ := zap.NewDevelopment()
			logger := zapr.NewLogger(zapLogger)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			resilienceConfig := &resilience.ResilienceConfig{
				DefaultTimeout:          30 * time.Second,
				MaxConcurrentOperations: 200,
				TimeoutEnabled:          false,
				BulkheadEnabled:         true,
				CircuitBreakerEnabled:   false,
				RateLimitingEnabled:     false,
				RetryEnabled:            false,
			}

			resilienceMgr := resilience.NewResilienceManager(resilienceConfig, logger)
			resilienceMgr.Start(ctx)
			defer resilienceMgr.Stop()

			errorConfig := &monitoring.ErrorTrackingConfig{
				EnablePrometheus:    false,
				EnableOpenTelemetry: false,
			}
			errorTracker, _ := monitoring.NewErrorTracker(errorConfig, logger)

			engineConfig := &ProcessingEngineConfig{
				MaxConcurrentIntents: 100,
				IntentWorkers:       config.intentPool,
				LLMWorkers:          config.llmPool,
				RAGWorkers:          config.llmPool,
				ResourceWorkers:     config.resourcePool,
				ManifestWorkers:     config.resourcePool,
				GitOpsWorkers:       config.resourcePool / 2,
				DeploymentWorkers:   config.resourcePool / 2,
				MaxQueueSize:        500,
			}

			engine, err := NewParallelProcessingEngine(engineConfig, resilienceMgr, errorTracker, logger)
			if err != nil {
				b.Fatalf("Failed to create engine: %v", err)
			}

			engine.Start(ctx)
			defer engine.Stop()

			b.ResetTimer()

			// Benchmark with this configuration
			var wg sync.WaitGroup
			startTime := time.Now()

			for i := 0; i < b.N; i++ {
				wg.Add(1)
				go func(intentNum int) {
					defer wg.Done()

					intent := &v1alpha1.NetworkIntent{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("scaling-%s-%d", config.name, intentNum),
							Namespace: "benchmark",
						},
						Spec: v1alpha1.NetworkIntentSpec{
							IntentType: "scaling_test",
							Intent:     fmt.Sprintf("Pool scaling test %d", intentNum),
							Priority:   "medium",
						},
					}

					engine.ProcessIntentWorkflow(ctx, intent)
				}(i)
			}

			wg.Wait()
			duration := time.Since(startTime)

			metrics := engine.GetMetrics()
			throughput := float64(b.N) / duration.Seconds()

			b.Logf("%s config: %.2f intents/sec, %d tasks, %.2f%% success",
				config.name, throughput, metrics.TotalTasks, metrics.SuccessRate*100)
		})
	}
}

// BenchmarkLatencyDistribution measures latency distribution
func BenchmarkLatencyDistribution(b *testing.B) {
	engine, cleanup, err := setupBenchmarkEngine()
	if err != nil {
		b.Fatalf("Failed to setup benchmark engine: %v", err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), benchmarkTimeout)
	defer cancel()

	// Collect latency measurements
	latencies := make([]time.Duration, 0, b.N)
	var latencyMutex sync.Mutex

	b.ResetTimer()

	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func(intentNum int) {
			defer wg.Done()

			intent := &v1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("latency-%d", intentNum),
					Namespace: "benchmark",
				},
				Spec: v1alpha1.NetworkIntentSpec{
					IntentType: "latency_test",
					Intent:     fmt.Sprintf("Latency measurement intent %d", intentNum),
					Priority:   "medium",
				},
			}

			startTime := time.Now()
			engine.ProcessIntentWorkflow(ctx, intent)
			latency := time.Since(startTime)

			latencyMutex.Lock()
			latencies = append(latencies, latency)
			latencyMutex.Unlock()
		}(i)
	}

	wg.Wait()
	b.StopTimer()

	// Calculate latency percentiles
	if len(latencies) > 0 {
		// Simple percentile calculation (in production, use a proper implementation)
		// This is simplified for the benchmark
		var total time.Duration
		min := latencies[0]
		max := latencies[0]

		for _, latency := range latencies {
			total += latency
			if latency < min {
				min = latency
			}
			if latency > max {
				max = latency
			}
		}

		avg := total / time.Duration(len(latencies))

		b.Logf("Latency distribution:")
		b.Logf("  Min: %v", min)
		b.Logf("  Max: %v", max)
		b.Logf("  Avg: %v", avg)
		b.Logf("  Samples: %d", len(latencies))
	}
}
