package performance

import (
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// MetricsCollector collects and manages performance metrics.

type MetricsCollector struct {
	analyzer *MetricsAnalyzer

	cpuUsage float64

	memoryUsage uint64

	goroutineCount int

	mu sync.RWMutex

	stopChan chan struct{}
}

// NewMetricsCollector creates a new metrics collector.

func NewMetricsCollector() *MetricsCollector {

	mc := &MetricsCollector{

		analyzer: NewMetricsAnalyzer(),

		stopChan: make(chan struct{}),
	}

	// Start background collection.

	go mc.collectMetrics()

	return mc

}

// collectMetrics continuously collects system metrics.

func (mc *MetricsCollector) collectMetrics() {

	ticker := time.NewTicker(100 * time.Millisecond)

	defer ticker.Stop()

	for {

		select {

		case <-mc.stopChan:

			return

		case <-ticker.C:

			mc.updateMetrics()

		}

	}

}

// updateMetrics updates current metrics.

func (mc *MetricsCollector) updateMetrics() {

	mc.mu.Lock()

	defer mc.mu.Unlock()

	// Update CPU usage (simplified - in production use proper CPU monitoring).

	mc.cpuUsage = mc.calculateCPUUsage()

	// Update memory usage.

	var memStats runtime.MemStats

	runtime.ReadMemStats(&memStats)

	mc.memoryUsage = memStats.Alloc

	// Update goroutine count.

	mc.goroutineCount = runtime.NumGoroutine()

	// Record in analyzer.

	mc.analyzer.RecordResourceSnapshot(ResourceSnapshot{

		Timestamp: time.Now(),

		CPUPercent: mc.cpuUsage,

		MemoryMB: float64(mc.memoryUsage) / (1024 * 1024),

		GoroutineCount: mc.goroutineCount,
	})

}

// calculateCPUUsage calculates current CPU usage.

func (mc *MetricsCollector) calculateCPUUsage() float64 {

	// Simplified CPU calculation.

	// In production, use proper CPU monitoring libraries.

	return float64(runtime.NumCPU()) * 10.0 // Placeholder

}

// GetCPUUsage returns current CPU usage.

func (mc *MetricsCollector) GetCPUUsage() float64 {

	mc.mu.RLock()

	defer mc.mu.RUnlock()

	return mc.cpuUsage

}

// GetMemoryUsage returns current memory usage.

func (mc *MetricsCollector) GetMemoryUsage() uint64 {

	mc.mu.RLock()

	defer mc.mu.RUnlock()

	return mc.memoryUsage

}

// GetGoroutineCount returns current goroutine count.

func (mc *MetricsCollector) GetGoroutineCount() int {

	mc.mu.RLock()

	defer mc.mu.RUnlock()

	return mc.goroutineCount

}

// Stop stops the metrics collector.

func (mc *MetricsCollector) Stop() {

	close(mc.stopChan)

}

// ExportMetrics exports metrics to Prometheus.

func (mc *MetricsCollector) ExportMetrics(registry *prometheus.Registry) {

	mc.mu.RLock()

	defer mc.mu.RUnlock()

	// CPU usage.

	cpuGauge := prometheus.NewGauge(prometheus.GaugeOpts{

		Name: "system_cpu_usage_percent",

		Help: "Current CPU usage percentage",
	})

	cpuGauge.Set(mc.cpuUsage)

	registry.MustRegister(cpuGauge)

	// Memory usage.

	memGauge := prometheus.NewGauge(prometheus.GaugeOpts{

		Name: "system_memory_usage_bytes",

		Help: "Current memory usage in bytes",
	})

	memGauge.Set(float64(mc.memoryUsage))

	registry.MustRegister(memGauge)

	// Goroutine count.

	goroutineGauge := prometheus.NewGauge(prometheus.GaugeOpts{

		Name: "system_goroutines_count",

		Help: "Current number of goroutines",
	})

	goroutineGauge.Set(float64(mc.goroutineCount))

	registry.MustRegister(goroutineGauge)

	// Export analyzer metrics.

	mc.analyzer.ExportToPrometheus(registry)

}

// Reset resets all collected metrics to their initial state
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.cpuUsage = 0
	mc.memoryUsage = 0
	mc.goroutineCount = 0
	
	// Reset analyzer if it has a reset method
	if mc.analyzer != nil {
		// Analyzer reset would be implemented if needed
	}
}
