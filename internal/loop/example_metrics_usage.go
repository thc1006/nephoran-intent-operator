package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// ExampleMetricsUsage demonstrates how to use the comprehensive metrics system.

func ExampleMetricsUsage() {
	// Create a watcher with metrics enabled.

	config := Config{
		PorchPath: "/usr/bin/porch",

		Mode: "direct",

		OutDir: "/tmp/output",

		Once: false,

		DebounceDur: 100 * time.Millisecond,

		Period: 0, // Use fsnotify

		MaxWorkers: 4,

		CleanupAfter: 7 * 24 * time.Hour,

		MetricsPort: 8080, // HTTP metrics server port

	}

	watcher, err := NewWatcher("/tmp/handoff", config)
	if err != nil {
		log.Fatalf("Failed to create watcher: %v", err)
	}

	defer watcher.Close() // #nosec G307 - Error handled in defer

	// Start the watcher in a goroutine.

	go func() {
		if err := watcher.Start(); err != nil {
			log.Printf("Watcher error: %v", err)
		}
	}()

	// Give it some time to start.

	time.Sleep(2 * time.Second)

	// Example 1: Get metrics programmatically.

	fmt.Println("=== Programmatic Metrics Access ===")

	metrics := watcher.GetMetrics()

	fmt.Printf("Files Processed: %d\n", metrics.FilesProcessedTotal)

	fmt.Printf("Files Failed: %d\n", metrics.FilesFailedTotal)

	fmt.Printf("Memory Usage: %d bytes\n", metrics.MemoryUsageBytes)

	fmt.Printf("Worker Utilization: %.2f%%\n", metrics.WorkerUtilization)

	fmt.Printf("Throughput: %.2f files/sec\n", metrics.ThroughputFilesPerSecond)

	// Example 2: Access JSON metrics via HTTP.

	fmt.Println("\n=== HTTP JSON Metrics ===")

	ctx := context.Background()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/metrics", http.NoBody)
	if err != nil {
		log.Printf("Failed to create request: %v", err)

		return
	}

	client := &http.Client{}

	resp, err := client.Do(req)

	if err != nil {
		log.Printf("Failed to get metrics: %v", err)
	} else {
		defer resp.Body.Close() // #nosec G307 - Error handled in defer

		var metricsResponse map[string]interface{}

		if err := json.NewDecoder(resp.Body).Decode(&metricsResponse); err != nil {
			log.Printf("Failed to decode metrics: %v", err)
		} else {
			// Pretty print some key metrics.

			if performance, ok := metricsResponse["performance"].(map[string]interface{}); ok {
				fmt.Printf("Total files processed: %.0f\n", performance["files_processed_total"])

				fmt.Printf("Average processing time: %s\n", performance["average_processing_time"])

				if latencies, ok := performance["latency_percentiles"].(map[string]interface{}); ok {
					fmt.Printf("P95 latency: %.4f seconds\n", latencies["95"])

					fmt.Printf("P99 latency: %.4f seconds\n", latencies["99"])
				}
			}
		}
	}

	// Example 3: Prometheus metrics format.

	fmt.Println("\n=== Prometheus Metrics ===")

	req, err = http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/metrics/prometheus", http.NoBody)
	if err != nil {
		log.Printf("Failed to create request: %v", err)

		return
	}

	resp, err = client.Do(req)

	if err != nil {
		log.Printf("Failed to get Prometheus metrics: %v", err)
	} else {
		defer resp.Body.Close() // #nosec G307 - Error handled in defer

		// Show first few lines of Prometheus metrics.

		buf := make([]byte, 500)

		n, err := resp.Body.Read(buf)

		if err != nil && err != io.EOF {
			log.Printf("Failed to read Prometheus metrics: %v", err)
		} else {
			fmt.Printf("%s...\n", string(buf[:n]))
		}
	}

	// Example 4: Health check.

	fmt.Println("\n=== Health Check ===")

	req, err = http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/health", http.NoBody)
	if err != nil {
		log.Printf("Failed to create request: %v", err)

		return
	}

	resp, err = client.Do(req)

	if err != nil {
		log.Printf("Failed to get health status: %v", err)
	} else {
		defer resp.Body.Close() // #nosec G307 - Error handled in defer

		var health map[string]interface{}

		if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
			log.Printf("Failed to decode health: %v", err)
		} else {
			fmt.Printf("Status: %s\n", health["status"])

			fmt.Printf("Uptime: %s\n", health["uptime"])
		}
	}
}

// MetricsCollectorExample shows how to collect and analyze metrics over time.

func MetricsCollectorExample() {
	// This would typically be run in a monitoring system.

	metricsHistory := make([]*WatcherMetrics, 0, 100)

	config := Config{
		PorchPath: "/usr/bin/porch",

		MetricsPort: 8080,

		MaxWorkers: 2,
	}

	watcher, err := NewWatcher("/tmp/handoff", config)
	if err != nil {
		log.Fatalf("Failed to create watcher: %v", err)
	}

	defer watcher.Close() // #nosec G307 - Error handled in defer

	// Collect metrics every 10 seconds for analysis.

	ticker := time.NewTicker(10 * time.Second)

	defer ticker.Stop()

	for i := range 5 { // Collect 5 samples
		<-ticker.C

		metrics := watcher.GetMetrics()

		metricsHistory = append(metricsHistory, metrics)

		fmt.Printf("Sample %d: Processed=%d, Failed=%d, Memory=%dMB, Workers=%.1f%%\n",

			i+1,

			metrics.FilesProcessedTotal,

			metrics.FilesFailedTotal,

			metrics.MemoryUsageBytes/(1024*1024),

			metrics.WorkerUtilization)
	}

	// Analyze trends.

	fmt.Println("\n=== Trend Analysis ===")

	if len(metricsHistory) >= 2 {
		latest := metricsHistory[len(metricsHistory)-1]

		previous := metricsHistory[len(metricsHistory)-2]

		processingRate := float64(latest.FilesProcessedTotal-previous.FilesProcessedTotal) / 10.0 // per second

		fmt.Printf("Current processing rate: %.2f files/second\n", processingRate)

		memoryGrowth := latest.MemoryUsageBytes - previous.MemoryUsageBytes

		fmt.Printf("Memory growth: %d bytes\n", memoryGrowth)

		if latest.FilesFailedTotal > previous.FilesFailedTotal {
			fmt.Printf("⚠️ New failures detected: %d\n", latest.FilesFailedTotal-previous.FilesFailedTotal)
		}
	}
}

// AlertingExample demonstrates how to set up alerting based on metrics.

func AlertingExample(watcher *Watcher) {
	// This would typically integrate with alerting systems like Prometheus AlertManager.

	checkAlerts := func() {
		metrics := watcher.GetMetrics()

		// Memory usage alert.

		if metrics.MemoryUsageBytes > 100*1024*1024 { // 100MB
			fmt.Printf("🚨 ALERT: High memory usage: %d MB\n", metrics.MemoryUsageBytes/(1024*1024))
		}

		// Worker utilization alert.

		if metrics.WorkerUtilization > 90.0 {
			fmt.Printf("🚨 ALERT: High worker utilization: %.1f%%\n", metrics.WorkerUtilization)
		}

		// Error rate alert.

		totalFiles := metrics.FilesProcessedTotal + metrics.FilesFailedTotal

		if totalFiles > 0 {
			errorRate := float64(metrics.FilesFailedTotal) / float64(totalFiles) * 100

			if errorRate > 10.0 {
				fmt.Printf("🚨 ALERT: High error rate: %.1f%%\n", errorRate)
			}
		}

		// Processing latency alert (would need actual latency calculation).

		if metrics.AverageProcessingTime > 30*time.Second {
			fmt.Printf("🚨 ALERT: High processing latency: %v\n", metrics.AverageProcessingTime)
		}

		// Backpressure alert.

		if metrics.BackpressureEventsTotal > 0 {
			fmt.Printf("⚠️ WARNING: Backpressure events detected: %d\n", metrics.BackpressureEventsTotal)
		}
	}

	// Check alerts every minute.

	ticker := time.NewTicker(1 * time.Minute)

	defer ticker.Stop()

	for range ticker.C {
		checkAlerts()
	}
}

// CustomMetricsExtension shows how to extend metrics with custom business logic.

type CustomMetrics struct {
	*WatcherMetrics

	BusinessKPIs map[string]float64
}

// CalculateBusinessMetrics performs calculatebusinessmetrics operation.

func (c *CustomMetrics) CalculateBusinessMetrics() {
	if c.BusinessKPIs == nil {
		c.BusinessKPIs = make(map[string]float64)
	}

	// Example business KPIs.

	totalFiles := c.FilesProcessedTotal + c.FilesFailedTotal

	if totalFiles > 0 {
		c.BusinessKPIs["success_rate"] = float64(c.FilesProcessedTotal) / float64(totalFiles) * 100

		c.BusinessKPIs["failure_rate"] = float64(c.FilesFailedTotal) / float64(totalFiles) * 100
	}

	// Processing efficiency (files per second per worker).

	if c.ThroughputFilesPerSecond > 0 {
		// Assuming we know max workers from config.

		c.BusinessKPIs["processing_efficiency"] = c.ThroughputFilesPerSecond / 4.0 // assuming 4 workers
	}

	// Memory efficiency (files processed per MB).

	if c.MemoryUsageBytes > 0 {
		memoryMB := float64(c.MemoryUsageBytes) / (1024 * 1024)

		c.BusinessKPIs["memory_efficiency"] = float64(c.FilesProcessedTotal) / memoryMB
	}
}
