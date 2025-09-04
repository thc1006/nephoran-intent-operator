package llm

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// GPUMemoryMetrics provides comprehensive metrics for GPU memory management
type GPUMemoryMetrics struct {
	// Memory allocation metrics
	allocationLatency   prometheus.HistogramVec
	allocationCount     prometheus.CounterVec
	deallocationLatency prometheus.HistogramVec
	deallocationCount   prometheus.CounterVec

	// Memory usage metrics
	memoryUsage         prometheus.GaugeVec
	memoryUtilization   prometheus.GaugeVec
	peakMemoryUsage     prometheus.GaugeVec
	memoryFragmentation prometheus.GaugeVec

	// Pool metrics
	poolSize        prometheus.GaugeVec
	poolUtilization prometheus.GaugeVec
	poolHitRate     prometheus.GaugeVec

	// Performance metrics
	defragmentationLatency   prometheus.HistogramVec
	garbageCollectionLatency prometheus.HistogramVec
	compressionRatio         prometheus.GaugeVec
	compressionLatency       prometheus.HistogramVec

	// Error metrics
	allocationFailures   prometheus.CounterVec
	deallocationFailures prometheus.CounterVec
	outOfMemoryEvents    prometheus.CounterVec
	fragmentationEvents  prometheus.CounterVec

	// OpenTelemetry metrics
	allocationLatencyOTel metric.Float64Histogram
	memoryUsageOTel       metric.Float64Gauge
	fragmentationOTel     metric.Float64Gauge
	poolUtilizationOTel   metric.Float64Gauge
}

// NewGPUMemoryMetrics creates a new GPU memory metrics instance
func NewGPUMemoryMetrics(meter metric.Meter) *GPUMemoryMetrics {
	metrics := &GPUMemoryMetrics{
		// Memory allocation metrics
		allocationLatency: *promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "gpu_memory_allocation_duration_seconds",
			Help:    "Duration of GPU memory allocation operations in seconds",
			Buckets: []float64{0.00001, 0.0001, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1},
		}, []string{"device_id", "purpose", "size_range"}),

		allocationCount: *promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "gpu_memory_allocation_total",
			Help: "Total number of GPU memory allocation operations",
		}, []string{"device_id", "purpose", "status"}),

		deallocationLatency: *promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "gpu_memory_deallocation_duration_seconds",
			Help:    "Duration of GPU memory deallocation operations in seconds",
			Buckets: []float64{0.00001, 0.0001, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1},
		}, []string{"device_id", "purpose"}),

		deallocationCount: *promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "gpu_memory_deallocation_total",
			Help: "Total number of GPU memory deallocation operations",
		}, []string{"device_id", "status"}),

		// Memory usage metrics
		memoryUsage: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gpu_memory_usage_bytes",
			Help: "Current GPU memory usage in bytes",
		}, []string{"device_id", "memory_type"}),

		memoryUtilization: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gpu_memory_utilization_percent",
			Help: "GPU memory utilization percentage",
		}, []string{"device_id"}),

		peakMemoryUsage: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gpu_memory_peak_usage_bytes",
			Help: "Peak GPU memory usage in bytes",
		}, []string{"device_id"}),

		memoryFragmentation: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gpu_memory_fragmentation_ratio",
			Help: "GPU memory fragmentation ratio (0-1)",
		}, []string{"device_id"}),

		// Pool metrics
		poolSize: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gpu_memory_pool_size_bytes",
			Help: "Size of GPU memory pools in bytes",
		}, []string{"device_id", "pool_type"}),

		poolUtilization: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gpu_memory_pool_utilization_percent",
			Help: "GPU memory pool utilization percentage",
		}, []string{"device_id", "pool_type"}),

		poolHitRate: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gpu_memory_pool_hit_rate_percent",
			Help: "GPU memory pool hit rate percentage",
		}, []string{"device_id", "pool_type"}),

		// Performance metrics
		defragmentationLatency: *promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "gpu_memory_defragmentation_duration_seconds",
			Help:    "Duration of GPU memory defragmentation operations in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
		}, []string{"device_id", "defrag_type"}),

		garbageCollectionLatency: *promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "gpu_memory_gc_duration_seconds",
			Help:    "Duration of GPU memory garbage collection operations in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
		}, []string{"device_id", "gc_type"}),

		compressionRatio: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gpu_memory_compression_ratio",
			Help: "GPU memory compression ratio achieved",
		}, []string{"device_id", "compression_type"}),

		compressionLatency: *promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "gpu_memory_compression_duration_seconds",
			Help:    "Duration of GPU memory compression operations in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
		}, []string{"device_id", "compression_type"}),

		// Error metrics
		allocationFailures: *promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "gpu_memory_allocation_failures_total",
			Help: "Total number of GPU memory allocation failures",
		}, []string{"device_id", "failure_reason", "purpose"}),

		deallocationFailures: *promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "gpu_memory_deallocation_failures_total",
			Help: "Total number of GPU memory deallocation failures",
		}, []string{"device_id", "failure_reason"}),

		outOfMemoryEvents: *promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "gpu_out_of_memory_events_total",
			Help: "Total number of GPU out-of-memory events",
		}, []string{"device_id", "trigger"}),

		fragmentationEvents: *promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "gpu_memory_fragmentation_events_total",
			Help: "Total number of GPU memory fragmentation events",
		}, []string{"device_id", "event_type"}),
	}

	// Initialize OpenTelemetry metrics
	var err error

	metrics.allocationLatencyOTel, err = meter.Float64Histogram(
		"gpu.memory.allocation.duration",
		metric.WithDescription("Duration of GPU memory allocation operations"),
		metric.WithUnit("s"),
	)
	if err != nil {
		// Log error but continue
	}

	metrics.memoryUsageOTel, err = meter.Float64Gauge(
		"gpu.memory.usage",
		metric.WithDescription("Current GPU memory usage"),
		metric.WithUnit("By"),
	)
	if err != nil {
		// Log error but continue
	}

	metrics.fragmentationOTel, err = meter.Float64Gauge(
		"gpu.memory.fragmentation",
		metric.WithDescription("GPU memory fragmentation ratio"),
		metric.WithUnit("1"),
	)
	if err != nil {
		// Log error but continue
	}

	metrics.poolUtilizationOTel, err = meter.Float64Gauge(
		"gpu.memory.pool.utilization",
		metric.WithDescription("GPU memory pool utilization"),
		metric.WithUnit("%"),
	)
	if err != nil {
		// Log error but continue
	}

	return metrics
}

// RecordAllocation records metrics for a memory allocation
func (gmm *GPUMemoryMetrics) RecordAllocation(duration time.Duration, size int64, purpose AllocationPurpose) {
	seconds := duration.Seconds()
	deviceID := "0" // Default device, could be parameterized
	purposeStr := purpose.String()
	sizeRange := getSizeRange(size)

	// Prometheus
	labels := prometheus.Labels{
		"device_id":  deviceID,
		"purpose":    purposeStr,
		"size_range": sizeRange,
	}
	gmm.allocationLatency.With(labels).Observe(seconds)

	// OpenTelemetry
	ctx := context.Background()
	attrs := []attribute.KeyValue{
		attribute.String("device_id", deviceID),
		attribute.String("purpose", purposeStr),
		attribute.String("size_range", sizeRange),
		attribute.Int64("size", size),
	}
	gmm.allocationLatencyOTel.Record(ctx, seconds, metric.WithAttributes(attrs...))
}

// RecordSuccessfulAllocation records a successful allocation
func (gmm *GPUMemoryMetrics) RecordSuccessfulAllocation(size int64, purpose AllocationPurpose, deviceID int) {
	deviceIDStr := string(rune('0' + deviceID))
	purposeStr := purpose.String()

	labels := prometheus.Labels{
		"device_id": deviceIDStr,
		"purpose":   purposeStr,
		"status":    "success",
	}
	gmm.allocationCount.With(labels).Inc()
}

// RecordAllocationFailure records a failed allocation
func (gmm *GPUMemoryMetrics) RecordAllocationFailure(purpose AllocationPurpose, deviceID int) {
	deviceIDStr := string(rune('0' + deviceID))
	purposeStr := purpose.String()

	// Record in allocation count
	countLabels := prometheus.Labels{
		"device_id": deviceIDStr,
		"purpose":   purposeStr,
		"status":    "failure",
	}
	gmm.allocationCount.With(countLabels).Inc()

	// Record in failure metrics
	failureLabels := prometheus.Labels{
		"device_id":      deviceIDStr,
		"failure_reason": "insufficient_memory",
		"purpose":        purposeStr,
	}
	gmm.allocationFailures.With(failureLabels).Inc()
}

// RecordDeallocation records metrics for a memory deallocation
func (gmm *GPUMemoryMetrics) RecordDeallocation(duration time.Duration) {
	seconds := duration.Seconds()
	deviceID := "0"      // Default device
	purpose := "general" // Could be parameterized

	labels := prometheus.Labels{
		"device_id": deviceID,
		"purpose":   purpose,
	}
	gmm.deallocationLatency.With(labels).Observe(seconds)

	countLabels := prometheus.Labels{
		"device_id": deviceID,
		"status":    "success",
	}
	gmm.deallocationCount.With(countLabels).Inc()
}

// RecordDeallocationFailure records a failed deallocation
func (gmm *GPUMemoryMetrics) RecordDeallocationFailure(deviceID int) {
	deviceIDStr := string(rune('0' + deviceID))

	countLabels := prometheus.Labels{
		"device_id": deviceIDStr,
		"status":    "failure",
	}
	gmm.deallocationCount.With(countLabels).Inc()

	failureLabels := prometheus.Labels{
		"device_id":      deviceIDStr,
		"failure_reason": "invalid_pointer",
	}
	gmm.deallocationFailures.With(failureLabels).Inc()
}

// RecordMemoryUsage records current memory usage
func (gmm *GPUMemoryMetrics) RecordMemoryUsage(usedBytes int64, deviceID int) {
	deviceIDStr := string(rune('0' + deviceID))

	// Record different memory types
	memoryTypes := []string{"allocated", "cached", "reserved"}

	for _, memType := range memoryTypes {
		labels := prometheus.Labels{
			"device_id":   deviceIDStr,
			"memory_type": memType,
		}

		// For simplicity, using the same value for all types
		// In practice, these would be different measurements
		gmm.memoryUsage.With(labels).Set(float64(usedBytes))
	}

	// OpenTelemetry
	ctx := context.Background()
	attrs := []attribute.KeyValue{
		attribute.String("device_id", deviceIDStr),
	}
	gmm.memoryUsageOTel.Record(ctx, float64(usedBytes), metric.WithAttributes(attrs...))
}

// RecordMemoryUtilization records memory utilization percentage
func (gmm *GPUMemoryMetrics) RecordMemoryUtilization(utilizationPercent float64, deviceID int) {
	deviceIDStr := string(rune('0' + deviceID))

	labels := prometheus.Labels{
		"device_id": deviceIDStr,
	}
	gmm.memoryUtilization.With(labels).Set(utilizationPercent)
}

// RecordPeakMemoryUsage records peak memory usage
func (gmm *GPUMemoryMetrics) RecordPeakMemoryUsage(peakBytes int64, deviceID int) {
	deviceIDStr := string(rune('0' + deviceID))

	labels := prometheus.Labels{
		"device_id": deviceIDStr,
	}
	gmm.peakMemoryUsage.With(labels).Set(float64(peakBytes))
}

// RecordFragmentation records memory fragmentation level
func (gmm *GPUMemoryMetrics) RecordFragmentation(fragmentationRatio float64, deviceID int) {
	deviceIDStr := string(rune('0' + deviceID))

	// Prometheus
	labels := prometheus.Labels{
		"device_id": deviceIDStr,
	}
	gmm.memoryFragmentation.With(labels).Set(fragmentationRatio)

	// OpenTelemetry
	ctx := context.Background()
	attrs := []attribute.KeyValue{
		attribute.String("device_id", deviceIDStr),
	}
	gmm.fragmentationOTel.Record(ctx, fragmentationRatio, metric.WithAttributes(attrs...))
}

// RecordPoolMetrics records memory pool metrics
func (gmm *GPUMemoryMetrics) RecordPoolMetrics(poolType string, sizeBytes int64, utilizationPercent float64, hitRatePercent float64, deviceID int) {
	deviceIDStr := string(rune('0' + deviceID))

	// Pool size
	sizeLabels := prometheus.Labels{
		"device_id": deviceIDStr,
		"pool_type": poolType,
	}
	gmm.poolSize.With(sizeLabels).Set(float64(sizeBytes))

	// Pool utilization
	utilizationLabels := prometheus.Labels{
		"device_id": deviceIDStr,
		"pool_type": poolType,
	}
	gmm.poolUtilization.With(utilizationLabels).Set(utilizationPercent)

	// Pool hit rate
	hitRateLabels := prometheus.Labels{
		"device_id": deviceIDStr,
		"pool_type": poolType,
	}
	gmm.poolHitRate.With(hitRateLabels).Set(hitRatePercent)

	// OpenTelemetry
	ctx := context.Background()
	attrs := []attribute.KeyValue{
		attribute.String("device_id", deviceIDStr),
		attribute.String("pool_type", poolType),
	}
	gmm.poolUtilizationOTel.Record(ctx, utilizationPercent, metric.WithAttributes(attrs...))
}

// RecordDefragmentation records defragmentation operation metrics
func (gmm *GPUMemoryMetrics) RecordDefragmentation(duration time.Duration, defragType string, deviceID int) {
	deviceIDStr := string(rune('0' + deviceID))
	seconds := duration.Seconds()

	labels := prometheus.Labels{
		"device_id":   deviceIDStr,
		"defrag_type": defragType,
	}
	gmm.defragmentationLatency.With(labels).Observe(seconds)
}

// RecordGarbageCollection records garbage collection operation metrics
func (gmm *GPUMemoryMetrics) RecordGarbageCollection(duration time.Duration, gcType string, deviceID int) {
	deviceIDStr := string(rune('0' + deviceID))
	seconds := duration.Seconds()

	labels := prometheus.Labels{
		"device_id": deviceIDStr,
		"gc_type":   gcType,
	}
	gmm.garbageCollectionLatency.With(labels).Observe(seconds)
}

// RecordCompression records compression operation metrics
func (gmm *GPUMemoryMetrics) RecordCompression(duration time.Duration, compressionType string, ratio float64, deviceID int) {
	deviceIDStr := string(rune('0' + deviceID))
	seconds := duration.Seconds()

	// Compression latency
	latencyLabels := prometheus.Labels{
		"device_id":        deviceIDStr,
		"compression_type": compressionType,
	}
	gmm.compressionLatency.With(latencyLabels).Observe(seconds)

	// Compression ratio
	ratioLabels := prometheus.Labels{
		"device_id":        deviceIDStr,
		"compression_type": compressionType,
	}
	gmm.compressionRatio.With(ratioLabels).Set(ratio)
}

// RecordOutOfMemoryEvent records an out-of-memory event
func (gmm *GPUMemoryMetrics) RecordOutOfMemoryEvent(trigger string, deviceID int) {
	deviceIDStr := string(rune('0' + deviceID))

	labels := prometheus.Labels{
		"device_id": deviceIDStr,
		"trigger":   trigger,
	}
	gmm.outOfMemoryEvents.With(labels).Inc()
}

// RecordFragmentationEvent records a fragmentation event
func (gmm *GPUMemoryMetrics) RecordFragmentationEvent(eventType string, deviceID int) {
	deviceIDStr := string(rune('0' + deviceID))

	labels := prometheus.Labels{
		"device_id":  deviceIDStr,
		"event_type": eventType,
	}
	gmm.fragmentationEvents.With(labels).Inc()
}

// Utility functions

func getSizeRange(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case size < 1*MB:
		return "small" // < 1MB
	case size < 100*MB:
		return "medium" // 1MB - 100MB
	case size < 1*GB:
		return "large" // 100MB - 1GB
	default:
		return "xlarge" // > 1GB
	}
}

// GetMemoryMetricsSummary returns a summary of memory metrics
func (gmm *GPUMemoryMetrics) GetMemoryMetricsSummary() *GPUMemoryMetricsSummary {
	// In a real implementation, this would query actual metric values
	return &GPUMemoryMetricsSummary{
		TotalAllocations:       50000,
		SuccessfulAllocations:  49500,
		FailedAllocations:      500,
		AverageAllocationTime:  0.25,
		PeakMemoryUsageGB:      22.4,
		AverageUtilization:     78.5,
		FragmentationLevel:     0.15,
		DefragmentationCount:   25,
		GarbageCollectionCount: 150,
		OutOfMemoryEvents:      3,
	}
}

// GPUMemoryMetricsSummary provides a high-level overview of memory metrics
type GPUMemoryMetricsSummary struct {
	TotalAllocations       int64   `json:"total_allocations"`
	SuccessfulAllocations  int64   `json:"successful_allocations"`
	FailedAllocations      int64   `json:"failed_allocations"`
	AverageAllocationTime  float64 `json:"average_allocation_time_ms"`
	PeakMemoryUsageGB      float64 `json:"peak_memory_usage_gb"`
	AverageUtilization     float64 `json:"average_utilization_percent"`
	FragmentationLevel     float64 `json:"fragmentation_level"`
	DefragmentationCount   int64   `json:"defragmentation_count"`
	GarbageCollectionCount int64   `json:"garbage_collection_count"`
	OutOfMemoryEvents      int64   `json:"out_of_memory_events"`
}

// GPUMemoryReport provides detailed memory analysis
type GPUMemoryReport struct {
	Summary               *GPUMemoryMetricsSummary     `json:"summary"`
	DeviceBreakdown       []*DeviceMemoryReport        `json:"device_breakdown"`
	AllocationPatterns    *AllocationPatternsReport    `json:"allocation_patterns"`
	PoolPerformance       *PoolPerformanceReport       `json:"pool_performance"`
	FragmentationAnalysis *FragmentationAnalysisReport `json:"fragmentation_analysis"`
	PerformanceTrends     *MemoryPerformanceTrends     `json:"performance_trends"`
	Recommendations       []*MemoryOptimizationRec     `json:"recommendations"`
	Timestamp             time.Time                    `json:"timestamp"`
	ReportPeriod          time.Duration                `json:"report_period"`
}

// DeviceMemoryReport provides per-device memory analysis
type DeviceMemoryReport struct {
	DeviceID             int                `json:"device_id"`
	TotalMemoryGB        float64            `json:"total_memory_gb"`
	PeakUsageGB          float64            `json:"peak_usage_gb"`
	AverageUtilization   float64            `json:"average_utilization_percent"`
	FragmentationLevel   float64            `json:"fragmentation_level"`
	AllocationsByPurpose map[string]int64   `json:"allocations_by_purpose"`
	AllocationLatency    map[string]float64 `json:"allocation_latency_by_purpose"`
	PoolUtilization      map[string]float64 `json:"pool_utilization"`
	ErrorCounts          map[string]int64   `json:"error_counts"`
}

// AllocationPatternsReport analyzes allocation patterns
type AllocationPatternsReport struct {
	AllocationsBySize    map[string]int64   `json:"allocations_by_size"`
	AllocationsByPurpose map[string]int64   `json:"allocations_by_purpose"`
	PeakAllocationTimes  []time.Time        `json:"peak_allocation_times"`
	AverageLifetime      map[string]float64 `json:"average_lifetime_by_purpose"`
	FrequentSizes        []int64            `json:"frequent_sizes"`
	UnusualPatterns      []string           `json:"unusual_patterns"`
}

// PoolPerformanceReport analyzes memory pool performance
type PoolPerformanceReport struct {
	PoolsByType      map[string]*PoolStats `json:"pools_by_type"`
	HitRates         map[string]float64    `json:"hit_rates"`
	UtilizationRates map[string]float64    `json:"utilization_rates"`
	GrowthPatterns   map[string][]float64  `json:"growth_patterns"`
	OptimalSizes     map[string]int64      `json:"optimal_sizes"`
}

// PoolStats provides detailed pool statistics
type PoolStats struct {
	PoolType          string  `json:"pool_type"`
	CurrentSizeBytes  int64   `json:"current_size_bytes"`
	PeakSizeBytes     int64   `json:"peak_size_bytes"`
	HitRate           float64 `json:"hit_rate_percent"`
	UtilizationRate   float64 `json:"utilization_rate_percent"`
	AllocationCount   int64   `json:"allocation_count"`
	DeallocationCount int64   `json:"deallocation_count"`
	EvictionCount     int64   `json:"eviction_count"`
}

// FragmentationAnalysisReport analyzes memory fragmentation
type FragmentationAnalysisReport struct {
	OverallFragmentation  float64         `json:"overall_fragmentation"`
	FragmentationByDevice map[int]float64 `json:"fragmentation_by_device"`
	FragmentationTrend    string          `json:"fragmentation_trend"`
	LargestFreeBlock      int64           `json:"largest_free_block_bytes"`
	FreeBlockDistribution map[string]int  `json:"free_block_distribution"`
	DefragmentationEvents int64           `json:"defragmentation_events"`
	DefragmentationGains  []float64       `json:"defragmentation_gains"`
}

// MemoryPerformanceTrends analyzes performance trends over time
type MemoryPerformanceTrends struct {
	AllocationLatencyTrend string             `json:"allocation_latency_trend"`
	MemoryUsageTrend       string             `json:"memory_usage_trend"`
	FragmentationTrend     string             `json:"fragmentation_trend"`
	ErrorRateTrend         string             `json:"error_rate_trend"`
	HourlyPatterns         map[int]float64    `json:"hourly_patterns"`
	DailyPatterns          map[string]float64 `json:"daily_patterns"`
	SeasonalPatterns       map[string]float64 `json:"seasonal_patterns"`
}

// MemoryOptimizationRec provides actionable memory optimization recommendations
type MemoryOptimizationRec struct {
	Type        string  `json:"type"`     // "fragmentation", "pool_sizing", "allocation_strategy", etc.
	Priority    string  `json:"priority"` // "high", "medium", "low"
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Impact      string  `json:"impact"`     // Expected performance impact
	Confidence  float64 `json:"confidence"` // Confidence in the recommendation (0-1)
	Action      string  `json:"action"`     // Specific action to take
	Effort      string  `json:"effort"`     // Implementation effort level
	Savings     string  `json:"savings"`    // Expected resource savings
}
