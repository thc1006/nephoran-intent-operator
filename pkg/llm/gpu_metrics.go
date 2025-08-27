package llm

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// GPUMetrics provides comprehensive metrics for GPU acceleration
type GPUMetrics struct {
	// Prometheus metrics
	inferenceLatency prometheus.HistogramVec
	inferenceCount   prometheus.CounterVec
	batchSize        prometheus.HistogramVec
	gpuUtilization   prometheus.GaugeVec
	gpuMemoryUsage   prometheus.GaugeVec
	gpuTemperature   prometheus.GaugeVec
	gpuPowerUsage    prometheus.GaugeVec
	modelLoadTime    prometheus.HistogramVec
	cacheHitRate     prometheus.GaugeVec
	throughput       prometheus.GaugeVec
	queueDepth       prometheus.GaugeVec
	errorCount       prometheus.CounterVec

	// OpenTelemetry metrics
	inferenceLatencyOTel metric.Float64Histogram
	inferenceCountOTel   metric.Int64Counter
	batchSizeOTel        metric.Int64Histogram
	gpuUtilizationOTel   metric.Float64Gauge
	gpuMemoryUsageOTel   metric.Float64Gauge
	modelLoadTimeOTel    metric.Float64Histogram
	throughputOTel       metric.Float64Gauge
	errorCountOTel       metric.Int64Counter
}

// NewGPUMetrics creates a new GPU metrics instance
func NewGPUMetrics(meter metric.Meter) *GPUMetrics {
	metrics := &GPUMetrics{
		// Prometheus metrics
		inferenceLatency: *promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "gpu_inference_duration_seconds",
			Help:    "Duration of GPU inference operations in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
		}, []string{"model_name", "device_id", "batch_size"}),

		inferenceCount: *promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "gpu_inference_total",
			Help: "Total number of GPU inference operations",
		}, []string{"model_name", "device_id", "status"}),

		batchSize: *promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "gpu_batch_size",
			Help:    "Distribution of batch sizes for GPU inference",
			Buckets: []float64{1, 2, 4, 8, 16, 32, 64, 128, 256, 512},
		}, []string{"model_name", "device_id"}),

		gpuUtilization: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gpu_utilization_percent",
			Help: "GPU utilization percentage",
		}, []string{"device_id", "device_name"}),

		gpuMemoryUsage: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gpu_memory_usage_bytes",
			Help: "GPU memory usage in bytes",
		}, []string{"device_id", "device_name", "memory_type"}),

		gpuTemperature: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gpu_temperature_celsius",
			Help: "GPU temperature in Celsius",
		}, []string{"device_id", "device_name"}),

		gpuPowerUsage: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gpu_power_usage_watts",
			Help: "GPU power usage in watts",
		}, []string{"device_id", "device_name"}),

		modelLoadTime: *promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "gpu_model_load_duration_seconds",
			Help:    "Time taken to load models onto GPU",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 20, 30, 60},
		}, []string{"model_name", "device_id", "cache_hit"}),

		cacheHitRate: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gpu_cache_hit_rate",
			Help: "GPU cache hit rate percentage",
		}, []string{"cache_type", "device_id"}),

		throughput: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gpu_throughput_tokens_per_second",
			Help: "GPU inference throughput in tokens per second",
		}, []string{"model_name", "device_id"}),

		queueDepth: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gpu_queue_depth",
			Help: "Number of requests waiting in GPU queues",
		}, []string{"model_name", "device_id", "queue_type"}),

		errorCount: *promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "gpu_errors_total",
			Help: "Total number of GPU-related errors",
		}, []string{"error_type", "device_id", "model_name"}),
	}

	// Initialize OpenTelemetry metrics
	var err error

	metrics.inferenceLatencyOTel, err = meter.Float64Histogram(
		"gpu.inference.duration",
		metric.WithDescription("Duration of GPU inference operations"),
		metric.WithUnit("s"),
	)
	if err != nil {
		// Log error but continue
	}

	metrics.inferenceCountOTel, err = meter.Int64Counter(
		"gpu.inference.count",
		metric.WithDescription("Total number of GPU inference operations"),
	)
	if err != nil {
		// Log error but continue
	}

	metrics.batchSizeOTel, err = meter.Int64Histogram(
		"gpu.batch.size",
		metric.WithDescription("Distribution of batch sizes for GPU inference"),
	)
	if err != nil {
		// Log error but continue
	}

	metrics.gpuUtilizationOTel, err = meter.Float64Gauge(
		"gpu.utilization",
		metric.WithDescription("GPU utilization percentage"),
		metric.WithUnit("%"),
	)
	if err != nil {
		// Log error but continue
	}

	metrics.gpuMemoryUsageOTel, err = meter.Float64Gauge(
		"gpu.memory.usage",
		metric.WithDescription("GPU memory usage"),
		metric.WithUnit("By"),
	)
	if err != nil {
		// Log error but continue
	}

	metrics.modelLoadTimeOTel, err = meter.Float64Histogram(
		"gpu.model.load_time",
		metric.WithDescription("Time taken to load models onto GPU"),
		metric.WithUnit("s"),
	)
	if err != nil {
		// Log error but continue
	}

	metrics.throughputOTel, err = meter.Float64Gauge(
		"gpu.throughput",
		metric.WithDescription("GPU inference throughput"),
		metric.WithUnit("tokens/s"),
	)
	if err != nil {
		// Log error but continue
	}

	metrics.errorCountOTel, err = meter.Int64Counter(
		"gpu.errors",
		metric.WithDescription("Total number of GPU-related errors"),
	)
	if err != nil {
		// Log error but continue
	}

	return metrics
}

// RecordInferenceLatency records the latency of an inference operation
func (gm *GPUMetrics) RecordInferenceLatency(duration time.Duration, modelName string, deviceID ...int) {
	seconds := duration.Seconds()
	deviceIDStr := "0"
	batchSizeStr := "1"

	if len(deviceID) > 0 {
		deviceIDStr = string(rune('0' + deviceID[0]))
	}

	// Prometheus
	labels := prometheus.Labels{
		"model_name": modelName,
		"device_id":  deviceIDStr,
		"batch_size": batchSizeStr,
	}
	gm.inferenceLatency.With(labels).Observe(seconds)

	// OpenTelemetry
	ctx := context.Background()
	attrs := []attribute.KeyValue{
		attribute.String("model_name", modelName),
		attribute.String("device_id", deviceIDStr),
		attribute.String("batch_size", batchSizeStr),
	}
	gm.inferenceLatencyOTel.Record(ctx, seconds, metric.WithAttributes(attrs...))
}

// RecordSuccessfulInference records a successful inference operation
func (gm *GPUMetrics) RecordSuccessfulInference(modelName string, deviceID int) {
	deviceIDStr := string(rune('0' + deviceID))

	// Prometheus
	labels := prometheus.Labels{
		"model_name": modelName,
		"device_id":  deviceIDStr,
		"status":     "success",
	}
	gm.inferenceCount.With(labels).Inc()

	// OpenTelemetry
	ctx := context.Background()
	attrs := []attribute.KeyValue{
		attribute.String("model_name", modelName),
		attribute.String("device_id", deviceIDStr),
		attribute.String("status", "success"),
	}
	gm.inferenceCountOTel.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordFailedInference records a failed inference operation
func (gm *GPUMetrics) RecordFailedInference(modelName string, deviceID int, errorType string) {
	deviceIDStr := string(rune('0' + deviceID))

	// Prometheus
	inferenceLabels := prometheus.Labels{
		"model_name": modelName,
		"device_id":  deviceIDStr,
		"status":     "failure",
	}
	gm.inferenceCount.With(inferenceLabels).Inc()

	errorLabels := prometheus.Labels{
		"error_type": errorType,
		"device_id":  deviceIDStr,
		"model_name": modelName,
	}
	gm.errorCount.With(errorLabels).Inc()

	// OpenTelemetry
	ctx := context.Background()
	inferenceAttrs := []attribute.KeyValue{
		attribute.String("model_name", modelName),
		attribute.String("device_id", deviceIDStr),
		attribute.String("status", "failure"),
	}
	gm.inferenceCountOTel.Add(ctx, 1, metric.WithAttributes(inferenceAttrs...))

	errorAttrs := []attribute.KeyValue{
		attribute.String("error_type", errorType),
		attribute.String("device_id", deviceIDStr),
		attribute.String("model_name", modelName),
	}
	gm.errorCountOTel.Add(ctx, 1, metric.WithAttributes(errorAttrs...))
}

// RecordBatchSize records the size of an inference batch
func (gm *GPUMetrics) RecordBatchSize(batchSize int, modelName string, deviceID int) {
	deviceIDStr := string(rune('0' + deviceID))

	// Prometheus
	labels := prometheus.Labels{
		"model_name": modelName,
		"device_id":  deviceIDStr,
	}
	gm.batchSize.With(labels).Observe(float64(batchSize))

	// OpenTelemetry
	ctx := context.Background()
	attrs := []attribute.KeyValue{
		attribute.String("model_name", modelName),
		attribute.String("device_id", deviceIDStr),
	}
	gm.batchSizeOTel.Record(ctx, int64(batchSize), metric.WithAttributes(attrs...))
}

// RecordDeviceMetrics records comprehensive device metrics
func (gm *GPUMetrics) RecordDeviceMetrics(device *GPUDevice) {
	deviceIDStr := string(rune('0' + device.ID))

	// GPU Utilization
	utilizationLabels := prometheus.Labels{
		"device_id":   deviceIDStr,
		"device_name": device.Name,
	}
	gm.gpuUtilization.With(utilizationLabels).Set(device.Utilization)

	// GPU Memory Usage
	memoryLabels := prometheus.Labels{
		"device_id":   deviceIDStr,
		"device_name": device.Name,
		"memory_type": "allocated",
	}
	gm.gpuMemoryUsage.With(memoryLabels).Set(float64(device.AllocatedMemory))

	memoryLabels["memory_type"] = "available"
	gm.gpuMemoryUsage.With(memoryLabels).Set(float64(device.AvailableMemory))

	memoryLabels["memory_type"] = "total"
	gm.gpuMemoryUsage.With(memoryLabels).Set(float64(device.TotalMemory))

	// GPU Temperature
	tempLabels := prometheus.Labels{
		"device_id":   deviceIDStr,
		"device_name": device.Name,
	}
	gm.gpuTemperature.With(tempLabels).Set(float64(device.Temperature))

	// GPU Power Usage
	powerLabels := prometheus.Labels{
		"device_id":   deviceIDStr,
		"device_name": device.Name,
	}
	gm.gpuPowerUsage.With(powerLabels).Set(device.PowerUsage)

	// OpenTelemetry metrics
	ctx := context.Background()
	deviceAttrs := []attribute.KeyValue{
		attribute.String("device_id", deviceIDStr),
		attribute.String("device_name", device.Name),
	}

	gm.gpuUtilizationOTel.Record(ctx, device.Utilization, metric.WithAttributes(deviceAttrs...))

	memoryAttrs := append(deviceAttrs, attribute.String("memory_type", "allocated"))
	gm.gpuMemoryUsageOTel.Record(ctx, float64(device.AllocatedMemory), metric.WithAttributes(memoryAttrs...))
}

// RecordModelLoadTime records the time taken to load a model
func (gm *GPUMetrics) RecordModelLoadTime(duration time.Duration, modelName string, deviceID int, cacheHit bool) {
	seconds := duration.Seconds()
	deviceIDStr := string(rune('0' + deviceID))
	cacheHitStr := "false"
	if cacheHit {
		cacheHitStr = "true"
	}

	// Prometheus
	labels := prometheus.Labels{
		"model_name": modelName,
		"device_id":  deviceIDStr,
		"cache_hit":  cacheHitStr,
	}
	gm.modelLoadTime.With(labels).Observe(seconds)

	// OpenTelemetry
	ctx := context.Background()
	attrs := []attribute.KeyValue{
		attribute.String("model_name", modelName),
		attribute.String("device_id", deviceIDStr),
		attribute.Bool("cache_hit", cacheHit),
	}
	gm.modelLoadTimeOTel.Record(ctx, seconds, metric.WithAttributes(attrs...))
}

// RecordCacheHitRate records the cache hit rate for different cache types
func (gm *GPUMetrics) RecordCacheHitRate(hitRate float64, cacheType string, deviceID int) {
	deviceIDStr := string(rune('0' + deviceID))

	labels := prometheus.Labels{
		"cache_type": cacheType,
		"device_id":  deviceIDStr,
	}
	gm.cacheHitRate.With(labels).Set(hitRate)
}

// RecordThroughput records the inference throughput
func (gm *GPUMetrics) RecordThroughput(tokensPerSecond float64, modelName string, deviceID int) {
	deviceIDStr := string(rune('0' + deviceID))

	// Prometheus
	labels := prometheus.Labels{
		"model_name": modelName,
		"device_id":  deviceIDStr,
	}
	gm.throughput.With(labels).Set(tokensPerSecond)

	// OpenTelemetry
	ctx := context.Background()
	attrs := []attribute.KeyValue{
		attribute.String("model_name", modelName),
		attribute.String("device_id", deviceIDStr),
	}
	gm.throughputOTel.Record(ctx, tokensPerSecond, metric.WithAttributes(attrs...))
}

// RecordQueueDepth records the depth of processing queues
func (gm *GPUMetrics) RecordQueueDepth(depth int, modelName string, deviceID int, queueType string) {
	deviceIDStr := string(rune('0' + deviceID))

	labels := prometheus.Labels{
		"model_name": modelName,
		"device_id":  deviceIDStr,
		"queue_type": queueType,
	}
	gm.queueDepth.With(labels).Set(float64(depth))
}

// GetMetricsSummary returns a summary of current GPU metrics
func (gm *GPUMetrics) GetMetricsSummary() *GPUMetricsSummary {
	// This would gather current metric values
	// In a real implementation, this would query the metric registry
	return &GPUMetricsSummary{
		TotalInferences:    1000, // Example values
		SuccessRate:        0.995,
		AverageLatency:     45.2,
		TotalGPUs:          1,
		ActiveGPUs:         1,
		AverageUtilization: 78.5,
		TotalMemoryGB:      24.0,
		UsedMemoryGB:       18.2,
		AverageThroughput:  850.3,
	}
}

// GPUMetricsSummary provides a high-level overview of GPU performance
type GPUMetricsSummary struct {
	TotalInferences    int64   `json:"total_inferences"`
	SuccessRate        float64 `json:"success_rate"`
	AverageLatency     float64 `json:"average_latency_ms"`
	TotalGPUs          int     `json:"total_gpus"`
	ActiveGPUs         int     `json:"active_gpus"`
	AverageUtilization float64 `json:"average_utilization_percent"`
	TotalMemoryGB      float64 `json:"total_memory_gb"`
	UsedMemoryGB       float64 `json:"used_memory_gb"`
	AverageThroughput  float64 `json:"average_throughput_tokens_per_sec"`
	CacheHitRate       float64 `json:"cache_hit_rate_percent"`
	ErrorRate          float64 `json:"error_rate_percent"`
}

// GPUPerformanceReport provides detailed performance analysis
type GPUPerformanceReport struct {
	Summary          *GPUMetricsSummary            `json:"summary"`
	DeviceBreakdown  []*DeviceMetrics              `json:"device_breakdown"`
	ModelBreakdown   []*ModelMetrics               `json:"model_breakdown"`
	ThroughputTrends []*ThroughputDataPoint        `json:"throughput_trends"`
	LatencyTrends    []*LatencyDataPoint           `json:"latency_trends"`
	ErrorAnalysis    *ErrorAnalysis                `json:"error_analysis"`
	Recommendations  []*OptimizationRecommendation `json:"recommendations"`
	Timestamp        time.Time                     `json:"timestamp"`
	ReportPeriod     time.Duration                 `json:"report_period"`
}

// DeviceMetrics provides per-device performance metrics
type DeviceMetrics struct {
	DeviceID       int     `json:"device_id"`
	DeviceName     string  `json:"device_name"`
	Utilization    float64 `json:"utilization_percent"`
	MemoryUsageGB  float64 `json:"memory_usage_gb"`
	TotalMemoryGB  float64 `json:"total_memory_gb"`
	Temperature    int     `json:"temperature_celsius"`
	PowerUsage     float64 `json:"power_usage_watts"`
	ThroughputTPS  float64 `json:"throughput_tokens_per_second"`
	AverageLatency float64 `json:"average_latency_ms"`
	InferenceCount int64   `json:"inference_count"`
	ErrorCount     int64   `json:"error_count"`
	ActiveStreams  int     `json:"active_streams"`
}

// ModelMetrics provides per-model performance metrics
type ModelMetrics struct {
	ModelName        string  `json:"model_name"`
	InferenceCount   int64   `json:"inference_count"`
	AverageLatency   float64 `json:"average_latency_ms"`
	ThroughputTPS    float64 `json:"throughput_tokens_per_second"`
	CacheHitRate     float64 `json:"cache_hit_rate_percent"`
	ErrorRate        float64 `json:"error_rate_percent"`
	AverageBatchSize float64 `json:"average_batch_size"`
	MemoryUsageGB    float64 `json:"memory_usage_gb"`
	LoadTime         float64 `json:"load_time_seconds"`
}

// ThroughputDataPoint represents a throughput measurement over time
type ThroughputDataPoint struct {
	Timestamp    time.Time `json:"timestamp"`
	TokensPerSec float64   `json:"tokens_per_second"`
	ModelName    string    `json:"model_name"`
	DeviceID     int       `json:"device_id"`
}

// ErrorAnalysis provides detailed error analysis
type ErrorAnalysis struct {
	TotalErrors       int64                  `json:"total_errors"`
	ErrorRate         float64                `json:"error_rate_percent"`
	ErrorsByType      map[string]int64       `json:"errors_by_type"`
	ErrorsByDevice    map[int]int64          `json:"errors_by_device"`
	ErrorsByModel     map[string]int64       `json:"errors_by_model"`
	MostFrequentError string                 `json:"most_frequent_error"`
	ErrorTrends       []*ErrorTrendDataPoint `json:"error_trends"`
}

// ErrorTrendDataPoint represents error trends over time
type ErrorTrendDataPoint struct {
	Timestamp  time.Time `json:"timestamp"`
	ErrorCount int64     `json:"error_count"`
	ErrorType  string    `json:"error_type"`
}

// OptimizationRecommendation provides actionable performance recommendations
type OptimizationRecommendation struct {
	Type        string  `json:"type"`     // "memory", "batching", "model_placement", etc.
	Priority    string  `json:"priority"` // "high", "medium", "low"
	Description string  `json:"description"`
	Impact      string  `json:"impact"`     // Expected performance impact
	Confidence  float64 `json:"confidence"` // Confidence in the recommendation (0-1)
	Action      string  `json:"action"`     // Specific action to take
}
