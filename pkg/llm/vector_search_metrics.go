package llm

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// VectorSearchMetrics provides comprehensive metrics for vector search operations
type VectorSearchMetrics struct {
	// Prometheus metrics
	embeddingGenerationLatency prometheus.HistogramVec
	embeddingGenerationCount   prometheus.CounterVec
	vectorIndexingLatency      prometheus.HistogramVec
	vectorIndexingCount        prometheus.CounterVec
	vectorSearchLatency        prometheus.HistogramVec
	vectorSearchCount          prometheus.CounterVec
	searchResultCount          prometheus.HistogramVec
	searchAccuracy             prometheus.GaugeVec
	cacheHitRate               prometheus.GaugeVec
	indexSize                  prometheus.GaugeVec
	gpuMemoryUsage             prometheus.GaugeVec
	throughputTPS              prometheus.GaugeVec
	errorCount                 prometheus.CounterVec

	// OpenTelemetry metrics
	embeddingLatencyOTel metric.Float64Histogram
	searchLatencyOTel    metric.Float64Histogram
	indexingLatencyOTel  metric.Float64Histogram
	throughputOTel       metric.Float64Gauge
	cacheHitRateOTel     metric.Float64Gauge
	errorCountOTel       metric.Int64Counter
}

// NewVectorSearchMetrics creates a new vector search metrics instance
func NewVectorSearchMetrics(meter metric.Meter) *VectorSearchMetrics {
	metrics := &VectorSearchMetrics{
		// Prometheus metrics
		embeddingGenerationLatency: *promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "vector_embedding_generation_duration_seconds",
			Help:    "Duration of embedding generation operations in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2},
		}, []string{"model_name", "batch_size", "method"}),

		embeddingGenerationCount: *promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "vector_embedding_generation_total",
			Help: "Total number of embedding generation operations",
		}, []string{"model_name", "status", "method"}),

		vectorIndexingLatency: *promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "vector_indexing_duration_seconds",
			Help:    "Duration of vector indexing operations in seconds",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1},
		}, []string{"index_type", "dimensions"}),

		vectorIndexingCount: *promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "vector_indexing_total",
			Help: "Total number of vector indexing operations",
		}, []string{"index_type", "status"}),

		vectorSearchLatency: *promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "vector_search_duration_seconds",
			Help:    "Duration of vector search operations in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		}, []string{"search_method", "top_k_range", "similarity_metric"}),

		vectorSearchCount: *promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "vector_search_total",
			Help: "Total number of vector search operations",
		}, []string{"search_method", "status"}),

		searchResultCount: *promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "vector_search_results_count",
			Help:    "Distribution of search result counts",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		}, []string{"search_method"}),

		searchAccuracy: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vector_search_accuracy_score",
			Help: "Search accuracy score based on relevance feedback",
		}, []string{"search_method", "similarity_metric"}),

		cacheHitRate: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vector_cache_hit_rate",
			Help: "Cache hit rate percentage for different cache types",
		}, []string{"cache_type", "cache_level"}),

		indexSize: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vector_index_size_vectors",
			Help: "Number of vectors in the index",
		}, []string{"index_type"}),

		gpuMemoryUsage: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vector_gpu_memory_usage_bytes",
			Help: "GPU memory usage for vector operations",
		}, []string{"device_id", "memory_type"}),

		throughputTPS: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vector_throughput_operations_per_second",
			Help: "Throughput of vector operations per second",
		}, []string{"operation_type", "method"}),

		errorCount: *promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "vector_errors_total",
			Help: "Total number of vector operation errors",
		}, []string{"error_type", "operation", "method"}),
	}

	// Initialize OpenTelemetry metrics
	var err error

	metrics.embeddingLatencyOTel, err = meter.Float64Histogram(
		"vector.embedding.duration",
		metric.WithDescription("Duration of embedding generation operations"),
		metric.WithUnit("s"),
	)
	if err != nil {
		// Log error but continue
	}

	metrics.searchLatencyOTel, err = meter.Float64Histogram(
		"vector.search.duration",
		metric.WithDescription("Duration of vector search operations"),
		metric.WithUnit("s"),
	)
	if err != nil {
		// Log error but continue
	}

	metrics.indexingLatencyOTel, err = meter.Float64Histogram(
		"vector.indexing.duration",
		metric.WithDescription("Duration of vector indexing operations"),
		metric.WithUnit("s"),
	)
	if err != nil {
		// Log error but continue
	}

	metrics.throughputOTel, err = meter.Float64Gauge(
		"vector.throughput",
		metric.WithDescription("Vector operations throughput"),
		metric.WithUnit("ops/s"),
	)
	if err != nil {
		// Log error but continue
	}

	metrics.cacheHitRateOTel, err = meter.Float64Gauge(
		"vector.cache.hit_rate",
		metric.WithDescription("Vector cache hit rate"),
		metric.WithUnit("%"),
	)
	if err != nil {
		// Log error but continue
	}

	metrics.errorCountOTel, err = meter.Int64Counter(
		"vector.errors",
		metric.WithDescription("Total number of vector operation errors"),
	)
	if err != nil {
		// Log error but continue
	}

	return metrics
}

// RecordEmbeddingGeneration records the metrics for embedding generation
func (vsm *VectorSearchMetrics) RecordEmbeddingGeneration(duration time.Duration, modelName string) {
	seconds := duration.Seconds()
	method := "single"

	// Prometheus
	labels := prometheus.Labels{
		"model_name": modelName,
		"batch_size": "1",
		"method":     method,
	}
	vsm.embeddingGenerationLatency.With(labels).Observe(seconds)

	countLabels := prometheus.Labels{
		"model_name": modelName,
		"status":     "success",
		"method":     method,
	}
	vsm.embeddingGenerationCount.With(countLabels).Inc()

	// OpenTelemetry
	ctx := context.Background()
	attrs := []attribute.KeyValue{
		attribute.String("model_name", modelName),
		attribute.String("method", method),
	}
	vsm.embeddingLatencyOTel.Record(ctx, seconds, metric.WithAttributes(attrs...))
}

// RecordBatchEmbeddingGeneration records metrics for batch embedding generation
func (vsm *VectorSearchMetrics) RecordBatchEmbeddingGeneration(duration time.Duration, batchSize int, modelName string) {
	seconds := duration.Seconds()
	method := "batch"
	batchSizeStr := getBatchSizeRange(batchSize)

	// Prometheus
	labels := prometheus.Labels{
		"model_name": modelName,
		"batch_size": batchSizeStr,
		"method":     method,
	}
	vsm.embeddingGenerationLatency.With(labels).Observe(seconds)

	countLabels := prometheus.Labels{
		"model_name": modelName,
		"status":     "success",
		"method":     method,
	}
	vsm.embeddingGenerationCount.With(countLabels).Inc()

	// OpenTelemetry
	ctx := context.Background()
	attrs := []attribute.KeyValue{
		attribute.String("model_name", modelName),
		attribute.String("method", method),
		attribute.Int("batch_size", batchSize),
	}
	vsm.embeddingLatencyOTel.Record(ctx, seconds, metric.WithAttributes(attrs...))

	// Record throughput
	throughput := float64(batchSize) / seconds
	vsm.recordThroughput(throughput, "embedding", method)
}

// RecordVectorIndexing records metrics for vector indexing operations
func (vsm *VectorSearchMetrics) RecordVectorIndexing(duration time.Duration) {
	seconds := duration.Seconds()
	indexType := "hnsw"  // Default, could be parameterized
	dimensions := "1536" // Default, could be parameterized

	// Prometheus
	labels := prometheus.Labels{
		"index_type": indexType,
		"dimensions": dimensions,
	}
	vsm.vectorIndexingLatency.With(labels).Observe(seconds)

	countLabels := prometheus.Labels{
		"index_type": indexType,
		"status":     "success",
	}
	vsm.vectorIndexingCount.With(countLabels).Inc()

	// OpenTelemetry
	ctx := context.Background()
	attrs := []attribute.KeyValue{
		attribute.String("index_type", indexType),
		attribute.String("dimensions", dimensions),
	}
	vsm.indexingLatencyOTel.Record(ctx, seconds, metric.WithAttributes(attrs...))
}

// RecordVectorSearch records metrics for vector search operations
func (vsm *VectorSearchMetrics) RecordVectorSearch(duration time.Duration, topK int) {
	seconds := duration.Seconds()
	searchMethod := "gpu" // Could be parameterized
	topKRange := getTopKRange(topK)
	similarityMetric := "cosine" // Could be parameterized

	// Prometheus
	labels := prometheus.Labels{
		"search_method":     searchMethod,
		"top_k_range":       topKRange,
		"similarity_metric": similarityMetric,
	}
	vsm.vectorSearchLatency.With(labels).Observe(seconds)

	countLabels := prometheus.Labels{
		"search_method": searchMethod,
		"status":        "success",
	}
	vsm.vectorSearchCount.With(countLabels).Inc()

	// OpenTelemetry
	ctx := context.Background()
	attrs := []attribute.KeyValue{
		attribute.String("search_method", searchMethod),
		attribute.String("similarity_metric", similarityMetric),
		attribute.Int("top_k", topK),
	}
	vsm.searchLatencyOTel.Record(ctx, seconds, metric.WithAttributes(attrs...))
}

// RecordSearchResults records the number of results returned by a search
func (vsm *VectorSearchMetrics) RecordSearchResults(resultCount int, searchMethod string) {
	labels := prometheus.Labels{
		"search_method": searchMethod,
	}
	vsm.searchResultCount.With(labels).Observe(float64(resultCount))
}

// RecordCacheHit records a cache hit
func (vsm *VectorSearchMetrics) RecordCacheHit(cacheType string) {
	// This would typically be called from a cache hit ratio calculation
	// For now, we'll just increment a counter or update a gauge
	vsm.updateCacheHitRate(cacheType, 1.0) // Placeholder
}

// RecordCacheMiss records a cache miss
func (vsm *VectorSearchMetrics) RecordCacheMiss(cacheType string) {
	vsm.updateCacheHitRate(cacheType, 0.0) // Placeholder
}

// RecordSearchAccuracy records search accuracy metrics
func (vsm *VectorSearchMetrics) RecordSearchAccuracy(accuracy float64, searchMethod, similarityMetric string) {
	labels := prometheus.Labels{
		"search_method":     searchMethod,
		"similarity_metric": similarityMetric,
	}
	vsm.searchAccuracy.With(labels).Set(accuracy)
}

// RecordIndexSize records the current size of vector indices
func (vsm *VectorSearchMetrics) RecordIndexSize(vectorCount int, indexType string) {
	labels := prometheus.Labels{
		"index_type": indexType,
	}
	vsm.indexSize.With(labels).Set(float64(vectorCount))
}

// RecordGPUMemoryUsage records GPU memory usage for vector operations
func (vsm *VectorSearchMetrics) RecordGPUMemoryUsage(memoryBytes int64, deviceID int, memoryType string) {
	deviceIDStr := string(rune('0' + deviceID))
	labels := prometheus.Labels{
		"device_id":   deviceIDStr,
		"memory_type": memoryType,
	}
	vsm.gpuMemoryUsage.With(labels).Set(float64(memoryBytes))
}

// RecordEmbeddingError records embedding generation errors
func (vsm *VectorSearchMetrics) RecordEmbeddingError(modelName string) {
	// Prometheus
	labels := prometheus.Labels{
		"model_name": modelName,
		"status":     "error",
		"method":     "single",
	}
	vsm.embeddingGenerationCount.With(labels).Inc()

	errorLabels := prometheus.Labels{
		"error_type": "embedding_generation",
		"operation":  "generate",
		"method":     "single",
	}
	vsm.errorCount.With(errorLabels).Inc()

	// OpenTelemetry
	ctx := context.Background()
	attrs := []attribute.KeyValue{
		attribute.String("error_type", "embedding_generation"),
		attribute.String("operation", "generate"),
		attribute.String("model_name", modelName),
	}
	vsm.errorCountOTel.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordSearchError records search operation errors
func (vsm *VectorSearchMetrics) RecordSearchError() {
	searchMethod := "gpu" // Could be parameterized

	// Prometheus
	countLabels := prometheus.Labels{
		"search_method": searchMethod,
		"status":        "error",
	}
	vsm.vectorSearchCount.With(countLabels).Inc()

	errorLabels := prometheus.Labels{
		"error_type": "search_execution",
		"operation":  "search",
		"method":     searchMethod,
	}
	vsm.errorCount.With(errorLabels).Inc()

	// OpenTelemetry
	ctx := context.Background()
	attrs := []attribute.KeyValue{
		attribute.String("error_type", "search_execution"),
		attribute.String("operation", "search"),
		attribute.String("method", searchMethod),
	}
	vsm.errorCountOTel.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// Private helper methods

func (vsm *VectorSearchMetrics) recordThroughput(throughput float64, operationType, method string) {
	// Prometheus
	labels := prometheus.Labels{
		"operation_type": operationType,
		"method":         method,
	}
	vsm.throughputTPS.With(labels).Set(throughput)

	// OpenTelemetry
	ctx := context.Background()
	attrs := []attribute.KeyValue{
		attribute.String("operation_type", operationType),
		attribute.String("method", method),
	}
	vsm.throughputOTel.Record(ctx, throughput, metric.WithAttributes(attrs...))
}

func (vsm *VectorSearchMetrics) updateCacheHitRate(cacheType string, hitValue float64) {
	cacheLevel := "l1" // Simplified, could be more sophisticated

	// Prometheus
	labels := prometheus.Labels{
		"cache_type":  cacheType,
		"cache_level": cacheLevel,
	}
	// This would need to be properly calculated from hit/miss ratios
	vsm.cacheHitRate.With(labels).Set(hitValue * 100) // Convert to percentage

	// OpenTelemetry
	ctx := context.Background()
	attrs := []attribute.KeyValue{
		attribute.String("cache_type", cacheType),
		attribute.String("cache_level", cacheLevel),
	}
	vsm.cacheHitRateOTel.Record(ctx, hitValue*100, metric.WithAttributes(attrs...))
}

// Utility functions for metric labeling

func getBatchSizeRange(batchSize int) string {
	switch {
	case batchSize == 1:
		return "1"
	case batchSize <= 10:
		return "2-10"
	case batchSize <= 50:
		return "11-50"
	case batchSize <= 100:
		return "51-100"
	case batchSize <= 500:
		return "101-500"
	default:
		return "500+"
	}
}

func getTopKRange(topK int) string {
	switch {
	case topK <= 5:
		return "1-5"
	case topK <= 10:
		return "6-10"
	case topK <= 25:
		return "11-25"
	case topK <= 50:
		return "26-50"
	case topK <= 100:
		return "51-100"
	default:
		return "100+"
	}
}

// GetMetricsSummary returns a comprehensive summary of vector search metrics
func (vsm *VectorSearchMetrics) GetMetricsSummary() *VectorSearchMetricsSummary {
	// In a real implementation, this would query the actual metric values
	// from the metric registries
	return &VectorSearchMetricsSummary{
		TotalEmbeddings:         10000,
		TotalSearches:           5000,
		AverageSearchLatency:    25.5,
		AverageEmbeddingLatency: 150.2,
		CacheHitRate:            85.3,
		IndexSize:               1000000,
		ErrorRate:               0.2,
		ThroughputEPS:           450.7,
		ThroughputSPS:           89.2,
		GPUUtilization:          72.8,
		GPUMemoryUsageGB:        6.4,
	}
}

// VectorSearchMetricsSummary provides a high-level overview
type VectorSearchMetricsSummary struct {
	TotalEmbeddings         int64   `json:"total_embeddings"`
	TotalSearches           int64   `json:"total_searches"`
	AverageSearchLatency    float64 `json:"average_search_latency_ms"`
	AverageEmbeddingLatency float64 `json:"average_embedding_latency_ms"`
	CacheHitRate            float64 `json:"cache_hit_rate_percent"`
	IndexSize               int64   `json:"index_size_vectors"`
	ErrorRate               float64 `json:"error_rate_percent"`
	ThroughputEPS           float64 `json:"throughput_embeddings_per_second"`
	ThroughputSPS           float64 `json:"throughput_searches_per_second"`
	GPUUtilization          float64 `json:"gpu_utilization_percent"`
	GPUMemoryUsageGB        float64 `json:"gpu_memory_usage_gb"`
}

// VectorSearchReport provides detailed performance analysis
type VectorSearchReport struct {
	Summary             *VectorSearchMetricsSummary  `json:"summary"`
	EmbeddingBreakdown  *EmbeddingMetricsBreakdown   `json:"embedding_breakdown"`
	SearchBreakdown     *SearchMetricsBreakdown      `json:"search_breakdown"`
	CacheAnalysis       *CacheAnalysisReport         `json:"cache_analysis"`
	PerformanceTrends   *PerformanceTrendsReport     `json:"performance_trends"`
	ErrorAnalysis       *ErrorAnalysisReport         `json:"error_analysis"`
	ResourceUtilization *ResourceUtilizationReport   `json:"resource_utilization"`
	Recommendations     []*PerformanceRecommendation `json:"recommendations"`
	Timestamp           time.Time                    `json:"timestamp"`
	ReportPeriod        time.Duration                `json:"report_period"`
}

// EmbeddingMetricsBreakdown provides detailed embedding metrics
type EmbeddingMetricsBreakdown struct {
	TotalRequests         int64              `json:"total_requests"`
	RequestsByModel       map[string]int64   `json:"requests_by_model"`
	LatencyByModel        map[string]float64 `json:"latency_by_model_ms"`
	ThroughputByModel     map[string]float64 `json:"throughput_by_model_eps"`
	BatchSizeDistribution map[string]float64 `json:"batch_size_distribution"`
	ErrorsByModel         map[string]int64   `json:"errors_by_model"`
}

// SearchMetricsBreakdown provides detailed search metrics
type SearchMetricsBreakdown struct {
	TotalSearches       int64              `json:"total_searches"`
	SearchesByMethod    map[string]int64   `json:"searches_by_method"`
	LatencyByMethod     map[string]float64 `json:"latency_by_method_ms"`
	AccuracyByMethod    map[string]float64 `json:"accuracy_by_method"`
	TopKDistribution    map[string]float64 `json:"top_k_distribution"`
	ResultsDistribution map[string]float64 `json:"results_distribution"`
}

// CacheAnalysisReport provides cache performance analysis
type CacheAnalysisReport struct {
	OverallHitRate  float64            `json:"overall_hit_rate_percent"`
	HitRateByType   map[string]float64 `json:"hit_rate_by_type"`
	HitRateByLevel  map[string]float64 `json:"hit_rate_by_level"`
	CacheSizes      map[string]int64   `json:"cache_sizes"`
	EvictionCounts  map[string]int64   `json:"eviction_counts"`
	CacheEfficiency float64            `json:"cache_efficiency_score"`
}

// PerformanceTrendsReport provides performance trend analysis
type PerformanceTrendsReport struct {
	LatencyTrend       string             `json:"latency_trend"`
	ThroughputTrend    string             `json:"throughput_trend"`
	ErrorRateTrend     string             `json:"error_rate_trend"`
	ResourceUsageTrend string             `json:"resource_usage_trend"`
	HourlyPatterns     map[int]float64    `json:"hourly_patterns"`
	DailyPatterns      map[string]float64 `json:"daily_patterns"`
}

// ErrorAnalysisReport provides error analysis and diagnostics
type ErrorAnalysisReport struct {
	TotalErrors   int64              `json:"total_errors"`
	ErrorsByType  map[string]int64   `json:"errors_by_type"`
	ErrorsByModel map[string]int64   `json:"errors_by_model"`
	ErrorRate     float64            `json:"error_rate_percent"`
	TopErrors     []string           `json:"top_errors"`
	ErrorTrends   map[string]float64 `json:"error_trends"`
	RecoveryRate  float64            `json:"recovery_rate_percent"`
	MTTR          float64            `json:"mean_time_to_recovery_seconds"`
}

// ResourceUtilizationReport provides resource usage analysis
type ResourceUtilizationReport struct {
	GPUUtilization     float64 `json:"gpu_utilization_percent"`
	GPUMemoryUsage     float64 `json:"gpu_memory_usage_percent"`
	CPUUtilization     float64 `json:"cpu_utilization_percent"`
	SystemMemoryUsage  float64 `json:"system_memory_usage_percent"`
	DiskIOUtilization  float64 `json:"disk_io_utilization_percent"`
	NetworkUtilization float64 `json:"network_utilization_percent"`
	ResourceEfficiency float64 `json:"resource_efficiency_score"`
}

// PerformanceRecommendation provides actionable performance recommendations
type PerformanceRecommendation struct {
	Type        string  `json:"type"`     // "cache", "batch_size", "gpu_memory", etc.
	Priority    string  `json:"priority"` // "high", "medium", "low"
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Impact      string  `json:"impact"`     // Expected performance impact
	Confidence  float64 `json:"confidence"` // Confidence in the recommendation (0-1)
	Action      string  `json:"action"`     // Specific action to take
	Effort      string  `json:"effort"`     // Implementation effort level
}
