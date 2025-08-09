//go:build !disable_rag && !test

package rag

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusMetrics provides comprehensive Prometheus metrics for RAG system
type PrometheusMetrics struct {
	// Cache metrics - Redis
	redisCacheHits      *prometheus.CounterVec
	redisCacheMisses    *prometheus.CounterVec
	redisCacheLatency   *prometheus.HistogramVec
	redisCacheSize      *prometheus.GaugeVec
	redisCacheEvictions *prometheus.CounterVec

	// Cache metrics - Memory
	memoryCacheHits        *prometheus.CounterVec
	memoryCacheMisses      *prometheus.CounterVec
	memoryCacheLatency     *prometheus.HistogramVec
	memoryCacheSize        *prometheus.GaugeVec
	memoryCacheEvictions   *prometheus.CounterVec
	memoryCacheUtilization *prometheus.GaugeVec

	// Connection pool metrics - Weaviate
	weaviatePoolConnections      *prometheus.GaugeVec
	weaviatePoolConnectionsTotal *prometheus.CounterVec
	weaviatePoolLatency          *prometheus.HistogramVec
	weaviatePoolErrors           *prometheus.CounterVec

	// Query performance metrics
	queryLatency    *prometheus.HistogramVec
	queryThroughput *prometheus.CounterVec
	queryErrors     *prometheus.CounterVec
	queryComplexity *prometheus.HistogramVec

	// Embedding metrics
	embeddingLatency    *prometheus.HistogramVec
	embeddingThroughput *prometheus.CounterVec
	embeddingCost       *prometheus.CounterVec
	embeddingTokens     *prometheus.CounterVec

	// Document processing metrics
	documentProcessingLatency *prometheus.HistogramVec
	documentProcessingSize    *prometheus.HistogramVec
	chunkingLatency           *prometheus.HistogramVec
	chunkingThroughput        *prometheus.CounterVec

	// System resource metrics
	systemCPU        prometheus.Gauge
	systemMemory     prometheus.Gauge
	systemDisk       prometheus.Gauge
	systemGoroutines prometheus.Gauge
	systemHeapSize   prometheus.Gauge

	// Business metrics
	intentTypes     *prometheus.CounterVec
	userSessions    *prometheus.GaugeVec
	responseQuality *prometheus.HistogramVec
}

// NewPrometheusMetrics creates and registers all Prometheus metrics
func NewPrometheusMetrics() *PrometheusMetrics {
	pm := &PrometheusMetrics{}
	pm.initializeMetrics()
	pm.registerMetrics()
	return pm
}

func (pm *PrometheusMetrics) initializeMetrics() {
	// Redis cache metrics
	pm.redisCacheHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_redis_cache_hits_total",
			Help: "Total number of Redis cache hits by category",
		},
		[]string{"category"},
	)

	pm.redisCacheMisses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_redis_cache_misses_total",
			Help: "Total number of Redis cache misses by category",
		},
		[]string{"category"},
	)

	pm.redisCacheLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nephoran_redis_cache_latency_seconds",
			Help:    "Redis cache operation latency in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"operation", "category"},
	)

	pm.redisCacheSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nephoran_redis_cache_size_bytes",
			Help: "Current Redis cache size in bytes by category",
		},
		[]string{"category"},
	)

	pm.redisCacheEvictions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_redis_cache_evictions_total",
			Help: "Total number of Redis cache evictions by category",
		},
		[]string{"category", "reason"},
	)

	// Memory cache metrics
	pm.memoryCacheHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_memory_cache_hits_total",
			Help: "Total number of memory cache hits by category",
		},
		[]string{"category"},
	)

	pm.memoryCacheMisses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_memory_cache_misses_total",
			Help: "Total number of memory cache misses by category",
		},
		[]string{"category"},
	)

	pm.memoryCacheLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nephoran_memory_cache_latency_seconds",
			Help:    "Memory cache operation latency in seconds",
			Buckets: []float64{0.00001, 0.00005, 0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1},
		},
		[]string{"operation", "category"},
	)

	pm.memoryCacheSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nephoran_memory_cache_size_bytes",
			Help: "Current memory cache size in bytes by category",
		},
		[]string{"category"},
	)

	pm.memoryCacheEvictions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_memory_cache_evictions_total",
			Help: "Total number of memory cache evictions by category and reason",
		},
		[]string{"category", "reason"},
	)

	pm.memoryCacheUtilization = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nephoran_memory_cache_utilization_ratio",
			Help: "Memory cache utilization ratio (0-1) by type",
		},
		[]string{"type"}, // "items" or "size"
	)

	// Weaviate connection pool metrics
	pm.weaviatePoolConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nephoran_weaviate_pool_connections",
			Help: "Current number of Weaviate connections by state",
		},
		[]string{"state"}, // "active", "idle", "total"
	)

	pm.weaviatePoolConnectionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_weaviate_pool_connections_total",
			Help: "Total Weaviate pool connection events",
		},
		[]string{"event"}, // "created", "destroyed", "reused"
	)

	pm.weaviatePoolLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nephoran_weaviate_pool_latency_seconds",
			Help:    "Weaviate connection pool operation latency in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
		},
		[]string{"operation"}, // "get_connection", "execute_query", "health_check"
	)

	pm.weaviatePoolErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_weaviate_pool_errors_total",
			Help: "Total Weaviate pool errors by type",
		},
		[]string{"error_type"},
	)

	// Query performance metrics
	pm.queryLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nephoran_query_latency_seconds",
			Help:    "End-to-end query processing latency in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 20, 30, 60, 120},
		},
		[]string{"intent_type", "enhancement", "cache_used"},
	)

	pm.queryThroughput = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_queries_processed_total",
			Help: "Total number of processed queries",
		},
		[]string{"intent_type", "status"},
	)

	pm.queryErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_query_errors_total",
			Help: "Total number of query errors by type",
		},
		[]string{"error_type", "component"},
	)

	pm.queryComplexity = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nephoran_query_complexity_score",
			Help:    "Query complexity score distribution",
			Buckets: []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
		},
		[]string{"intent_type"},
	)

	// Embedding metrics
	pm.embeddingLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nephoran_embedding_latency_seconds",
			Help:    "Embedding generation latency in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 20, 30},
		},
		[]string{"provider", "model", "batch_size_range"},
	)

	pm.embeddingThroughput = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_embeddings_generated_total",
			Help: "Total number of embeddings generated",
		},
		[]string{"provider", "model", "status"},
	)

	pm.embeddingCost = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_embedding_cost_usd_total",
			Help: "Total embedding generation cost in USD",
		},
		[]string{"provider", "model"},
	)

	pm.embeddingTokens = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_embedding_tokens_total",
			Help: "Total tokens processed for embeddings",
		},
		[]string{"provider", "model"},
	)

	// Document processing metrics
	pm.documentProcessingLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nephoran_document_processing_latency_seconds",
			Help:    "Document processing latency in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600},
		},
		[]string{"document_type", "processing_stage"},
	)

	pm.documentProcessingSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nephoran_document_size_bytes",
			Help:    "Processed document sizes in bytes",
			Buckets: []float64{1024, 10240, 102400, 1048576, 10485760, 104857600, 1073741824}, // 1KB to 1GB
		},
		[]string{"document_type"},
	)

	pm.chunkingLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nephoran_chunking_latency_seconds",
			Help:    "Document chunking latency in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10},
		},
		[]string{"strategy", "document_type"},
	)

	pm.chunkingThroughput = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_chunks_created_total",
			Help: "Total number of document chunks created",
		},
		[]string{"strategy", "document_type"},
	)

	// System resource metrics
	pm.systemCPU = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "nephoran_system_cpu_usage_percent",
			Help: "Current CPU usage percentage",
		},
	)

	pm.systemMemory = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "nephoran_system_memory_usage_percent",
			Help: "Current memory usage percentage",
		},
	)

	pm.systemDisk = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "nephoran_system_disk_usage_percent",
			Help: "Current disk usage percentage",
		},
	)

	pm.systemGoroutines = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "nephoran_system_goroutines_count",
			Help: "Current number of goroutines",
		},
	)

	pm.systemHeapSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "nephoran_system_heap_size_bytes",
			Help: "Current heap size in bytes",
		},
	)

	// Business metrics
	pm.intentTypes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_intent_types_total",
			Help: "Total processed queries by intent type",
		},
		[]string{"intent_type", "domain"},
	)

	pm.userSessions = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nephoran_user_sessions_active",
			Help: "Currently active user sessions",
		},
		[]string{"session_type"},
	)

	pm.responseQuality = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nephoran_response_quality_score",
			Help:    "Response quality score distribution (0-1)",
			Buckets: []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
		},
		[]string{"intent_type", "enhancement_used"},
	)
}

func (pm *PrometheusMetrics) registerMetrics() {
	// Redis cache metrics
	prometheus.MustRegister(
		pm.redisCacheHits,
		pm.redisCacheMisses,
		pm.redisCacheLatency,
		pm.redisCacheSize,
		pm.redisCacheEvictions,
	)

	// Memory cache metrics
	prometheus.MustRegister(
		pm.memoryCacheHits,
		pm.memoryCacheMisses,
		pm.memoryCacheLatency,
		pm.memoryCacheSize,
		pm.memoryCacheEvictions,
		pm.memoryCacheUtilization,
	)

	// Weaviate pool metrics
	prometheus.MustRegister(
		pm.weaviatePoolConnections,
		pm.weaviatePoolConnectionsTotal,
		pm.weaviatePoolLatency,
		pm.weaviatePoolErrors,
	)

	// Query metrics
	prometheus.MustRegister(
		pm.queryLatency,
		pm.queryThroughput,
		pm.queryErrors,
		pm.queryComplexity,
	)

	// Embedding metrics
	prometheus.MustRegister(
		pm.embeddingLatency,
		pm.embeddingThroughput,
		pm.embeddingCost,
		pm.embeddingTokens,
	)

	// Document processing metrics
	prometheus.MustRegister(
		pm.documentProcessingLatency,
		pm.documentProcessingSize,
		pm.chunkingLatency,
		pm.chunkingThroughput,
	)

	// System metrics
	prometheus.MustRegister(
		pm.systemCPU,
		pm.systemMemory,
		pm.systemDisk,
		pm.systemGoroutines,
		pm.systemHeapSize,
	)

	// Business metrics
	prometheus.MustRegister(
		pm.intentTypes,
		pm.userSessions,
		pm.responseQuality,
	)
}

// Redis Cache Metrics Methods

func (pm *PrometheusMetrics) RecordRedisCacheHit(category string) {
	pm.redisCacheHits.WithLabelValues(category).Inc()
}

func (pm *PrometheusMetrics) RecordRedisCacheMiss(category string) {
	pm.redisCacheMisses.WithLabelValues(category).Inc()
}

func (pm *PrometheusMetrics) RecordRedisCacheLatency(operation, category string, duration time.Duration) {
	pm.redisCacheLatency.WithLabelValues(operation, category).Observe(duration.Seconds())
}

func (pm *PrometheusMetrics) UpdateRedisCacheSize(category string, sizeBytes int64) {
	pm.redisCacheSize.WithLabelValues(category).Set(float64(sizeBytes))
}

func (pm *PrometheusMetrics) RecordRedisCacheEviction(category, reason string) {
	pm.redisCacheEvictions.WithLabelValues(category, reason).Inc()
}

// Memory Cache Metrics Methods

func (pm *PrometheusMetrics) RecordMemoryCacheHit(category string) {
	pm.memoryCacheHits.WithLabelValues(category).Inc()
}

func (pm *PrometheusMetrics) RecordMemoryCacheMiss(category string) {
	pm.memoryCacheMisses.WithLabelValues(category).Inc()
}

func (pm *PrometheusMetrics) RecordMemoryCacheLatency(operation, category string, duration time.Duration) {
	pm.memoryCacheLatency.WithLabelValues(operation, category).Observe(duration.Seconds())
}

func (pm *PrometheusMetrics) UpdateMemoryCacheSize(category string, sizeBytes int64) {
	pm.memoryCacheSize.WithLabelValues(category).Set(float64(sizeBytes))
}

func (pm *PrometheusMetrics) RecordMemoryCacheEviction(category, reason string) {
	pm.memoryCacheEvictions.WithLabelValues(category, reason).Inc()
}

func (pm *PrometheusMetrics) UpdateMemoryCacheUtilization(utilizationType string, ratio float64) {
	pm.memoryCacheUtilization.WithLabelValues(utilizationType).Set(ratio)
}

// Weaviate Pool Metrics Methods

func (pm *PrometheusMetrics) UpdateWeaviatePoolConnections(state string, count int) {
	pm.weaviatePoolConnections.WithLabelValues(state).Set(float64(count))
}

func (pm *PrometheusMetrics) RecordWeaviatePoolConnectionEvent(event string) {
	pm.weaviatePoolConnectionsTotal.WithLabelValues(event).Inc()
}

func (pm *PrometheusMetrics) RecordWeaviatePoolLatency(operation string, duration time.Duration) {
	pm.weaviatePoolLatency.WithLabelValues(operation).Observe(duration.Seconds())
}

func (pm *PrometheusMetrics) RecordWeaviatePoolError(errorType string) {
	pm.weaviatePoolErrors.WithLabelValues(errorType).Inc()
}

// Query Metrics Methods

func (pm *PrometheusMetrics) RecordQueryLatency(intentType, enhancement, cacheUsed string, duration time.Duration) {
	pm.queryLatency.WithLabelValues(intentType, enhancement, cacheUsed).Observe(duration.Seconds())
}

func (pm *PrometheusMetrics) RecordQueryProcessed(intentType, status string) {
	pm.queryThroughput.WithLabelValues(intentType, status).Inc()
}

func (pm *PrometheusMetrics) RecordQueryError(errorType, component string) {
	pm.queryErrors.WithLabelValues(errorType, component).Inc()
}

func (pm *PrometheusMetrics) RecordQueryComplexity(intentType string, complexity float64) {
	pm.queryComplexity.WithLabelValues(intentType).Observe(complexity)
}

// Embedding Metrics Methods

func (pm *PrometheusMetrics) RecordEmbeddingLatency(provider, model, batchSizeRange string, duration time.Duration) {
	pm.embeddingLatency.WithLabelValues(provider, model, batchSizeRange).Observe(duration.Seconds())
}

func (pm *PrometheusMetrics) RecordEmbeddingGenerated(provider, model, status string) {
	pm.embeddingThroughput.WithLabelValues(provider, model, status).Inc()
}

func (pm *PrometheusMetrics) RecordEmbeddingCost(provider, model string, costUSD float64) {
	pm.embeddingCost.WithLabelValues(provider, model).Add(costUSD)
}

func (pm *PrometheusMetrics) RecordEmbeddingTokens(provider, model string, tokens int64) {
	pm.embeddingTokens.WithLabelValues(provider, model).Add(float64(tokens))
}

// Document Processing Metrics Methods

func (pm *PrometheusMetrics) RecordDocumentProcessingLatency(docType, stage string, duration time.Duration) {
	pm.documentProcessingLatency.WithLabelValues(docType, stage).Observe(duration.Seconds())
}

func (pm *PrometheusMetrics) RecordDocumentSize(docType string, sizeBytes int64) {
	pm.documentProcessingSize.WithLabelValues(docType).Observe(float64(sizeBytes))
}

func (pm *PrometheusMetrics) RecordChunkingLatency(strategy, docType string, duration time.Duration) {
	pm.chunkingLatency.WithLabelValues(strategy, docType).Observe(duration.Seconds())
}

func (pm *PrometheusMetrics) RecordChunkCreated(strategy, docType string) {
	pm.chunkingThroughput.WithLabelValues(strategy, docType).Inc()
}

// System Metrics Methods

func (pm *PrometheusMetrics) UpdateSystemCPU(percent float64) {
	pm.systemCPU.Set(percent)
}

func (pm *PrometheusMetrics) UpdateSystemMemory(percent float64) {
	pm.systemMemory.Set(percent)
}

func (pm *PrometheusMetrics) UpdateSystemDisk(percent float64) {
	pm.systemDisk.Set(percent)
}

func (pm *PrometheusMetrics) UpdateSystemGoroutines(count int) {
	pm.systemGoroutines.Set(float64(count))
}

func (pm *PrometheusMetrics) UpdateSystemHeapSize(bytes int64) {
	pm.systemHeapSize.Set(float64(bytes))
}

// Business Metrics Methods

func (pm *PrometheusMetrics) RecordIntentType(intentType, domain string) {
	pm.intentTypes.WithLabelValues(intentType, domain).Inc()
}

func (pm *PrometheusMetrics) UpdateActiveUserSessions(sessionType string, count int) {
	pm.userSessions.WithLabelValues(sessionType).Set(float64(count))
}

func (pm *PrometheusMetrics) RecordResponseQuality(intentType, enhancementUsed string, quality float64) {
	pm.responseQuality.WithLabelValues(intentType, enhancementUsed).Observe(quality)
}

// Batch update methods for efficiency

func (pm *PrometheusMetrics) UpdateRedisCacheMetrics(metrics *RedisCacheMetrics) {
	categories := []string{"embedding", "document", "query_result", "context"}

	for _, category := range categories {
		if stats, exists := metrics.CategoryStats[category]; exists {
			// Update hits/misses
			pm.UpdateRedisCacheSize(category, stats.SizeBytes)

			// Calculate and update hit rate
			if stats.Hits+stats.Misses > 0 {
				hitRate := float64(stats.Hits) / float64(stats.Hits+stats.Misses)
				pm.UpdateRedisCacheSize(fmt.Sprintf("%s_hit_rate", category), int64(hitRate*100))
			}
		}
	}
}

func (pm *PrometheusMetrics) UpdateMemoryCacheMetrics(metrics *MemoryCacheMetrics) {
	// Update utilization metrics
	if metrics.MaxItems > 0 {
		itemUtilization := float64(metrics.CurrentItems) / float64(metrics.MaxItems)
		pm.UpdateMemoryCacheUtilization("items", itemUtilization)
	}

	if metrics.MaxSizeBytes > 0 {
		sizeUtilization := float64(metrics.CurrentSizeBytes) / float64(metrics.MaxSizeBytes)
		pm.UpdateMemoryCacheUtilization("size", sizeUtilization)
	}

	// Update category-specific metrics
	for category, stats := range metrics.CategoryStats {
		pm.UpdateMemoryCacheSize(category, stats.SizeBytes)
	}
}

func (pm *PrometheusMetrics) UpdateWeaviatePoolMetrics(poolMetrics PoolMetrics) {
	pm.UpdateWeaviatePoolConnections("active", int(poolMetrics.ActiveConnections))
	pm.UpdateWeaviatePoolConnections("idle", int(poolMetrics.IdleConnections))
	pm.UpdateWeaviatePoolConnections("total", int(poolMetrics.TotalConnections))
}

// Helper functions

func (pm *PrometheusMetrics) getBatchSizeRange(batchSize int) string {
	switch {
	case batchSize <= 10:
		return "1-10"
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

func (pm *PrometheusMetrics) calculateComplexity(query string, intentType string, filtersCount int) float64 {
	// Simplified complexity calculation
	complexity := 0.0

	// Base complexity by intent type
	switch intentType {
	case "configuration":
		complexity += 0.3
	case "troubleshooting":
		complexity += 0.5
	case "optimization":
		complexity += 0.7
	case "monitoring":
		complexity += 0.4
	default:
		complexity += 0.2
	}

	// Add complexity based on query length
	queryLength := len(query)
	if queryLength > 500 {
		complexity += 0.3
	} else if queryLength > 200 {
		complexity += 0.2
	} else if queryLength > 100 {
		complexity += 0.1
	}

	// Add complexity based on filters
	complexity += float64(filtersCount) * 0.1

	// Cap at 1.0
	if complexity > 1.0 {
		complexity = 1.0
	}

	return complexity
}

// MetricsCollector provides a centralized way to collect and update all metrics
type MetricsCollector struct {
	prometheus         *PrometheusMetrics
	memoryCache        *MemoryCache
	redisCache         *RedisCache
	weaviatePool       *WeaviateConnectionPool
	collectionInterval time.Duration
	stopChan           chan struct{}
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(memoryCache *MemoryCache, redisCache *RedisCache, weaviatePool *WeaviateConnectionPool) *MetricsCollector {
	return &MetricsCollector{
		prometheus:         NewPrometheusMetrics(),
		memoryCache:        memoryCache,
		redisCache:         redisCache,
		weaviatePool:       weaviatePool,
		collectionInterval: 30 * time.Second,
		stopChan:           make(chan struct{}),
	}
}

// Start begins collecting metrics in the background
func (mc *MetricsCollector) Start() {
	go mc.collectMetricsLoop()
}

// Stop stops the metrics collection
func (mc *MetricsCollector) Stop() {
	close(mc.stopChan)
}

func (mc *MetricsCollector) collectMetricsLoop() {
	ticker := time.NewTicker(mc.collectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mc.collectAllMetrics()
		case <-mc.stopChan:
			return
		}
	}
}

func (mc *MetricsCollector) collectAllMetrics() {
	// Collect memory cache metrics
	if mc.memoryCache != nil {
		memoryStats := mc.memoryCache.GetStats()
		mc.prometheus.UpdateMemoryCacheMetrics(memoryStats)
	}

	// Collect Redis cache metrics
	if mc.redisCache != nil {
		redisStats := mc.redisCache.GetMetrics()
		mc.prometheus.UpdateRedisCacheMetrics(redisStats)
	}

	// Collect Weaviate pool metrics
	if mc.weaviatePool != nil {
		poolStats := mc.weaviatePool.GetMetrics()
		mc.prometheus.UpdateWeaviatePoolMetrics(poolStats)
	}
}

// GetPrometheusMetrics returns the Prometheus metrics instance
func (mc *MetricsCollector) GetPrometheusMetrics() *PrometheusMetrics {
	return mc.prometheus
}
