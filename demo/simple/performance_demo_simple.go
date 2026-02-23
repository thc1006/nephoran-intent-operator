package main

import (
	"context"
	"fmt"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/performance"
)

func main() {
	fmt.Println("=== Nephoran Performance Demo ===")

	// Initialize default config
	config := performance.DefaultIntegrationConfig
	fmt.Printf("Using performance config with GOMAXPROCS: %d\n", config.GOMAXPROCS)

	// Initialize performance integrator
	integrator := performance.NewPerformanceIntegrator(config)
	fmt.Printf("Performance integrator initialized: %T\n", integrator)

	// Run a simple benchmark
	suite := performance.NewBenchmarkSuite()
	duration := suite.Run(context.Background(), "demo-test", func() {
		// Simulate some work
		time.Sleep(10 * time.Millisecond)
		fmt.Printf(".")
	})

	fmt.Printf("\nDemo benchmark completed in: %v\n", duration)

	// Create a profiler
	profiler := performance.NewProfiler()
	profiler.Start()

	// Simulate some work
	for i := 0; i < 5; i++ {
		time.Sleep(5 * time.Millisecond)
	}

	results := profiler.Stop()
	fmt.Printf("Profiling results: %+v\n", results)

	// Create metrics analyzer
	analyzer := performance.NewMetricsAnalyzer()
	analyzer.AddSample(float64(duration.Milliseconds()))
	analyzer.AddSample(25.0)
	analyzer.AddSample(30.0)

	avg := analyzer.GetAverage()
	fmt.Printf("Average performance: %.2fms\n", avg)

	fmt.Println("Demo completed successfully!")
}
