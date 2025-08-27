package performance

import (
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// CNFFunction represents a CNF function type for performance tracking
type CNFFunction string

const (
	CNFFunctionCUCP CNFFunction = "cu-cp"
	CNFFunctionCUUP CNFFunction = "cu-up"
	CNFFunctionDU   CNFFunction = "du"
	CNFFunctionRIC  CNFFunction = "ric"
)

// MetricsCollector collects and manages performance metrics
type MetricsCollector struct {
	analyzer       *MetricsAnalyzer
	cpuUsage       float64
	memoryUsage    uint64
	goroutineCount int
	mu             sync.RWMutex
	stopChan       chan struct{}

	// CNF deployment metrics
	cnfDeployments map[string]time.Duration
	cnfMutex       sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	mc := &MetricsCollector{
		analyzer:       NewMetricsAnalyzer(),
		stopChan:       make(chan struct{}),
		cnfDeployments: make(map[string]time.Duration),
	}

	// Start background collection
	go mc.collectMetrics()

	return mc
}

// collectMetrics continuously collects system metrics
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

// updateMetrics updates current metrics
func (mc *MetricsCollector) updateMetrics() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Update CPU usage (simplified - in production use proper CPU monitoring)
	mc.cpuUsage = mc.calculateCPUUsage()

	// Update memory usage
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	mc.memoryUsage = memStats.Alloc

	// Update goroutine count
	mc.goroutineCount = runtime.NumGoroutine()

	// Record in analyzer
	mc.analyzer.RecordResourceSnapshot(ResourceSnapshot{
		Timestamp:      time.Now(),
		CPUPercent:     mc.cpuUsage,
		MemoryMB:       float64(mc.memoryUsage) / (1024 * 1024),
		GoroutineCount: mc.goroutineCount,
	})
}

// calculateCPUUsage calculates current CPU usage
func (mc *MetricsCollector) calculateCPUUsage() float64 {
	// Simplified CPU calculation
	// In production, use proper CPU monitoring libraries
	return float64(runtime.NumCPU()) * 10.0 // Placeholder
}

// RecordCNFDeployment records a CNF deployment metric
func (mc *MetricsCollector) RecordCNFDeployment(function CNFFunction, duration time.Duration) {
	mc.cnfMutex.Lock()
	defer mc.cnfMutex.Unlock()

	// Store the deployment duration for the CNF function
	mc.cnfDeployments[string(function)] = duration
}

// GetCNFDeploymentMetrics returns recorded CNF deployment metrics
func (mc *MetricsCollector) GetCNFDeploymentMetrics() map[string]time.Duration {
	mc.cnfMutex.RLock()
	defer mc.cnfMutex.RUnlock()

	result := make(map[string]time.Duration)
	for k, v := range mc.cnfDeployments {
		result[k] = v
	}
	return result
}

// GetCPUUsage returns current CPU usage
func (mc *MetricsCollector) GetCPUUsage() float64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.cpuUsage
}

// GetMemoryUsage returns current memory usage
func (mc *MetricsCollector) GetMemoryUsage() uint64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.memoryUsage
}

// GetGoroutineCount returns current goroutine count
func (mc *MetricsCollector) GetGoroutineCount() int {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.goroutineCount
}

// Stop stops the metrics collector
func (mc *MetricsCollector) Stop() {
	close(mc.stopChan)
}

// ExportMetrics exports metrics to Prometheus
func (mc *MetricsCollector) ExportMetrics(registry *prometheus.Registry) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// CPU usage
	cpuGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "system_cpu_usage_percent",
		Help: "Current CPU usage percentage",
	})
	cpuGauge.Set(mc.cpuUsage)
	registry.MustRegister(cpuGauge)

	// Memory usage
	memGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "system_memory_usage_bytes",
		Help: "Current memory usage in bytes",
	})
	memGauge.Set(float64(mc.memoryUsage))
	registry.MustRegister(memGauge)

	// Goroutine count
	goroutineGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "system_goroutines_count",
		Help: "Current number of goroutines",
	})
	goroutineGauge.Set(float64(mc.goroutineCount))
	registry.MustRegister(goroutineGauge)

	// CNF deployment metrics
	mc.cnfMutex.RLock()
	cnfGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cnf_deployment_duration_seconds",
		Help: "Duration of CNF deployments in seconds",
	}, []string{"function"})

	for function, duration := range mc.cnfDeployments {
		cnfGauge.WithLabelValues(function).Set(duration.Seconds())
	}
	mc.cnfMutex.RUnlock()

	registry.MustRegister(cnfGauge)

	// Export analyzer metrics
	mc.analyzer.ExportToPrometheus(registry)
}
