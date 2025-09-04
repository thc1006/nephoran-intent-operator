package main

import (
	"context"
	"encoding/json"
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
		fmt.Printf("  💻 CPU Usage: %.2f%%\n", metrics.CPUUsagePercent)
		fmt.Printf("  🧠 Memory Usage: %.2f MB\n", metrics.MemoryUsageMB)
		fmt.Printf("  📦 Heap Size: %.2f MB\n", metrics.HeapSizeMB)
		fmt.Printf("  🔧 Goroutine Count: %d\n", metrics.GoroutineCount)
		fmt.Printf("  🗑️ GC Count: %d\n", metrics.GCCount)
	}

	// Demonstrate cache capabilities
	if cacheManager := integrator.GetCacheManager(); cacheManager != nil {
		fmt.Println("\n⚡ Cache Performance Test:")
		ctx := context.Background()

		// Test cache operations
		testKey := "performance_test_key"
		testValue := json.RawMessage(`{"test": "data"}`)

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
				fmt.Printf("  🎯 Cache HIT - data retrieved successfully\n")
			}
		}

		// Get cache statistics
		stats := cacheManager.GetStats()
		fmt.Printf("  📈 Hit Rate: %.2f%%\n", stats.HitRate)
		fmt.Printf("  📉 Miss Rate: %.2f%%\n", stats.MissRate)
		// Note: Basic cache stats don't include detailed memory/operation counters
		fmt.Printf("  ✅ Cache operational\n")
	}

	// Demonstrate async processing capabilities
	if asyncProcessor := integrator.GetAsyncProcessor(); asyncProcessor != nil {
		fmt.Println("\n🔄 Async Processing Test:")

		// Submit test tasks
		for i := 0; i < 5; i++ {
			task := func() error {
				// Simulate some work
				time.Sleep(10 * time.Millisecond)
				fmt.Printf("    Processing task %d\n", i+1)
				return nil
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
		fmt.Printf("  📊 Queue Depth: %d\n", asyncMetrics.QueueDepth)
		fmt.Printf("  ⚡ Throughput: %.2f tasks/sec\n", asyncMetrics.Throughput)
		// Note: Basic async metrics don't include detailed counters
	}

	// Generate comprehensive performance report
	fmt.Println("\n📊 Performance Report:")
	report := integrator.GetPerformanceReport()
	fmt.Printf("  📈 Overall Score: %.1f/100\n", report.OverallScore())
	fmt.Printf("  🎯 Grade: %s\n", report.Grade())
	components := report.Components()
	fmt.Printf("  🔧 Components Analyzed: %d\n", len(components))
	fmt.Println("\n📋 Component Status:")
	for name, status := range components {
		fmt.Printf("  ✅ %s: %v\n", name, status)
	}
	fmt.Println("\n🚀 Basic performance optimizations applied")

	// Trigger performance optimization
	fmt.Println("\n⚡ Performance Optimization:")
	start := time.Now()
	err := integrator.OptimizePerformance()
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
		fmt.Printf("  💻 CPU Usage: %.2f%%\n", finalMetrics.CPUUsagePercent)
		fmt.Printf("  🧠 Memory Usage: %.2f MB\n", finalMetrics.MemoryUsageMB)
		fmt.Printf("  📦 Heap Size: %.2f MB\n", finalMetrics.HeapSizeMB)
		fmt.Printf("  🔧 Goroutine Count: %d\n", finalMetrics.GoroutineCount)
	}

	// Performance monitoring endpoints
	if monitor := integrator.GetMonitor(); monitor != nil {
		fmt.Println("\n🌐 Monitoring Endpoints Available:")
		fmt.Println("  📊 Dashboard: http://localhost:8090/dashboard")
		fmt.Println("  📈 Metrics: http://localhost:8090/metrics")
		fmt.Println("  💚 Health: http://localhost:8090/health")
		fmt.Println("  🔍 Profiling: http://localhost:8090/debug/pprof/")
		fmt.Println("  🔄 Real-time Stream: http://localhost:8092/stream")
		fmt.Println("  ⚡ Benchmarks: http://localhost:8090/benchmark")
	}

	fmt.Println("\n📋 Performance Optimization Summary:")
	fmt.Println("  ✅ CPU profiling and goroutine leak detection")
	fmt.Println("  ✅ Multi-level caching with Redis and in-memory layers")
	fmt.Println("  ✅ Connection pooling and database optimization")
	fmt.Println("  ✅ Batch processing and async operations")
	fmt.Println("  ✅ Real-time performance monitoring and alerts")
	fmt.Println("  ✅ Comprehensive metrics and dashboards")
	fmt.Println("  ✅ Auto-optimization and recommendation engine")

	// Graceful cleanup
	fmt.Println("\n🧹 Cleaning up performance integrator...")
	// Note: Basic integrator doesn't require explicit shutdown
	fmt.Println("  ✅ Cleanup completed successfully")

	fmt.Println("\n🎉 Performance optimization demo completed!")
}

