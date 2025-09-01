package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/performance"
)

func main() {
	fmt.Println("🚀 Performance Optimization Demo")
	fmt.Println("====================================")

	// Initialize performance integrator with default configuration
	config := performance.DefaultIntegrationConfig
	config.EnableProfiling = true
	config.EnableMetrics = true
	config.UseSONIC = true
	config.EnableHTTP2 = true

	integrator := performance.NewPerformanceIntegrator(config)
	if integrator == nil {
		log.Fatal("Failed to create performance integrator")
	}

	fmt.Println("✅ Performance integrator initialized")

	// Demonstrate profiler capabilities
	if profiler := integrator.GetProfiler(); profiler != nil {
		fmt.Println("\n📊 Performance Metrics:")
		metrics := profiler.GetMetrics()
		fmt.Printf("  • CPU Usage: %.2f%%\n", metrics.CPUUsagePercent)
		fmt.Printf("  • Memory Usage: %.2f MB\n", metrics.MemoryUsageMB)
		fmt.Printf("  • Heap Size: %.2f MB\n", metrics.HeapSizeMB)
		fmt.Printf("  • Goroutine Count: %d\n", metrics.GoroutineCount)
		fmt.Printf("  • GC Count: %d\n", metrics.GCCount)
	}

	// Demonstrate cache capabilities
	if cacheManager := integrator.GetCacheManager(); cacheManager != nil {
		fmt.Println("\n🗂️  Cache Performance Test:")
		ctx := context.Background()

		// Test cache operations
		testKey := "performance_test_key"
		testValue := map[string]interface{}{
			"timestamp": time.Now(),
			"data":      "Performance optimization test data",
			"metrics":   []int{1, 2, 3, 4, 5},
		}

		// Set value in cache
		start := time.Now()
		err := cacheManager.Set(ctx, testKey, testValue)
		setDuration := time.Since(start)
		if err != nil {
			fmt.Printf("  ❌ Cache SET failed: %v\n", err)
		} else {
			fmt.Printf("  ✅ Cache SET completed in %v\n", setDuration)
		}

		// Get value from cache
		start = time.Now()
		cachedValue, err := cacheManager.Get(ctx, testKey)
		getDuration := time.Since(start)
		if err != nil {
			fmt.Printf("  ❌ Cache GET failed: %v\n", err)
		} else {
			fmt.Printf("  ✅ Cache GET completed in %v\n", getDuration)
			if cachedValue != nil {
				fmt.Printf("  📦 Cache HIT - data retrieved successfully\n")
			}
		}

		// Get cache statistics
		stats := cacheManager.GetStats()
		fmt.Printf("  • Hit Rate: %.2f%%\n", stats.HitRate)
		fmt.Printf("  • Miss Rate: %.2f%%\n", stats.MissRate)
		fmt.Printf("  • Memory Usage: %d bytes\n", stats.MemoryUsage)
		fmt.Printf("  • Total Operations: %d\n", stats.Gets+stats.Sets)
	}

	// Demonstrate async processing capabilities
	if asyncProcessor := integrator.GetAsyncProcessor(); asyncProcessor != nil {
		fmt.Println("\n⚡ Async Processing Test:")

		// Submit test tasks
		for i := 0; i < 5; i++ {
			task := performance.AsyncTask{
				ID:       fmt.Sprintf("demo_task_%d", i+1),
				Type:     "demo_task",
				Priority: 1,
				Data:     fmt.Sprintf("Task data %d", i+1),
				Context:  context.Background(),
				Timeout:  30 * time.Second,
			}

			err := asyncProcessor.SubmitTask(task)
			if err != nil {
				fmt.Printf("  ❌ Failed to submit task %d: %v\n", i+1, err)
			} else {
				fmt.Printf("  ✅ Task %d submitted successfully\n", i+1)
			}
		}

		// Wait a moment for processing
		time.Sleep(100 * time.Millisecond)

		// Get async metrics
		asyncMetrics := asyncProcessor.GetMetrics()
		fmt.Printf("  • Tasks Submitted: %d\n", asyncMetrics.TasksSubmitted)
		fmt.Printf("  • Tasks Completed: %d\n", asyncMetrics.TasksCompleted)
		fmt.Printf("  • Worker Utilization: %.2f%%\n", asyncMetrics.WorkerUtilization)
		fmt.Printf("  • Queue Depth: %d\n", asyncMetrics.QueueDepth)
	}

	// Generate comprehensive performance report
	fmt.Println("\n📈 Performance Report:")
	report := integrator.GetPerformanceReport()
	fmt.Printf("  • Overall Score: %.1f/100\n", report.OverallScore)
	fmt.Printf("  • Grade: %s\n", report.Grade)
	fmt.Printf("  • Components Analyzed: %d\n", len(report.Components))
	fmt.Printf("  • Recommendations: %d\n", len(report.Recommendations))

	if len(report.Recommendations) > 0 {
		fmt.Println("\n💡 Top Recommendations:")
		for i, rec := range report.Recommendations {
			if i >= 3 {
				break // Show top 3 recommendations
			}
			fmt.Printf("  %d. %s (Impact: %s, Priority: %d)\n",
				i+1, rec.Description, rec.Impact, rec.Priority)
			fmt.Printf("     Action: %s\n", rec.Action)
		}
	}

	// Trigger performance optimization
	fmt.Println("\n🔧 Performance Optimization:")
	start := time.Now()
	err = integrator.OptimizePerformance()
	optimizationDuration := time.Since(start)
	if err != nil {
		fmt.Printf("  ❌ Optimization failed: %v\n", err)
	} else {
		fmt.Printf("  ✅ Optimization completed in %v\n", optimizationDuration)
	}

	// Show final metrics
	if profiler := integrator.GetProfiler(); profiler != nil {
		fmt.Println("\n📊 Post-Optimization Metrics:")
		finalMetrics := profiler.GetMetrics()
		fmt.Printf("  • CPU Usage: %.2f%%\n", finalMetrics.CPUUsagePercent)
		fmt.Printf("  • Memory Usage: %.2f MB\n", finalMetrics.MemoryUsageMB)
		fmt.Printf("  • Heap Size: %.2f MB\n", finalMetrics.HeapSizeMB)
		fmt.Printf("  • Goroutine Count: %d\n", finalMetrics.GoroutineCount)
	}

	// Performance monitoring endpoints
	if monitor := integrator.GetMonitor(); monitor != nil {
		fmt.Println("\n🖥️  Monitoring Endpoints Available:")
		fmt.Println("  • Dashboard: http://localhost:8090/dashboard")
		fmt.Println("  • Metrics: http://localhost:8090/metrics")
		fmt.Println("  • Health: http://localhost:8090/health")
		fmt.Println("  • Profiling: http://localhost:8090/debug/pprof/")
		fmt.Println("  • Real-time Stream: http://localhost:8092/stream")
		fmt.Println("  • Benchmarks: http://localhost:8090/benchmark")
	}

	fmt.Println("\n🎯 Performance Optimization Summary:")
	fmt.Println("  ✅ CPU profiling and goroutine leak detection")
	fmt.Println("  ✅ Multi-level caching with Redis and in-memory layers")
	fmt.Println("  ✅ Connection pooling and database optimization")
	fmt.Println("  ✅ Batch processing and async operations")
	fmt.Println("  ✅ Real-time performance monitoring and alerts")
	fmt.Println("  ✅ Comprehensive metrics and dashboards")
	fmt.Println("  ✅ Auto-optimization and recommendation engine")

	// Graceful shutdown
	fmt.Println("\n⏹️  Shutting down performance integrator...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := integrator.Shutdown(ctx); err != nil {
		fmt.Printf("  ❌ Shutdown error: %v\n", err)
	} else {
		fmt.Println("  ✅ Shutdown completed successfully")
	}

	fmt.Println("\n🎉 Performance optimization demo completed!")
}
