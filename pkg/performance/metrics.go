package performance

import (
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds all performance-related Prometheus metrics
type Metrics struct {
	// Request metrics
	RequestDuration *prometheus.HistogramVec
	RequestCounter  *prometheus.CounterVec
	RequestInFlight prometheus.Gauge

	// JSON processing metrics
	JSONMarshalDuration   prometheus.Histogram
	JSONUnmarshalDuration prometheus.Histogram
	JSONProcessingErrors  prometheus.Counter

	// Memory metrics
	HeapAlloc     prometheus.Gauge
	HeapSys       prometheus.Gauge
	HeapIdle      prometheus.Gauge
	HeapInuse     prometheus.Gauge
	HeapReleased  prometheus.Gauge
	StackInuse    prometheus.Gauge
	StackSys      prometheus.Gauge
	MSpanInuse    prometheus.Gauge
	MSpanSys      prometheus.Gauge
	MCacheInuse   prometheus.Gauge
	MCacheSys     prometheus.Gauge
	GCPauseTotal  prometheus.Gauge
	GCCount       prometheus.Gauge
	Goroutines    prometheus.Gauge

	// HTTP/2 metrics
	HTTP2StreamsActive    prometheus.Gauge
	HTTP2StreamsTotal     prometheus.Counter
	HTTP2FramesReceived   *prometheus.CounterVec
	HTTP2FramesSent       *prometheus.CounterVec
	HTTP2ConnectionErrors prometheus.Counter

	// Cache metrics
	CacheHits   *prometheus.CounterVec
	CacheMisses *prometheus.CounterVec
	CacheEvictions *prometheus.CounterVec
	CacheSize   *prometheus.GaugeVec

	// Database metrics
	DBConnectionsActive prometheus.Gauge
	DBConnectionsIdle   prometheus.Gauge
	DBQueryDuration     *prometheus.HistogramVec
	DBQueryErrors       *prometheus.CounterVec

	// Business metrics
	IntentsProcessed   *prometheus.CounterVec
	IntentLatency      *prometheus.HistogramVec
	ScalingOperations  *prometheus.CounterVec
	ScalingDuration    *prometheus.HistogramVec
}

// NewMetrics creates and registers all performance metrics
func NewMetrics(registry *prometheus.Registry) *Metrics {
	m := &Metrics{
		// Request metrics
		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "nephoran_http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to ~32s
			},
			[]string{"method", "endpoint", "status"},
		),
		RequestCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephoran_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		RequestInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "nephoran_http_requests_in_flight",
				Help: "Number of HTTP requests currently being processed",
			},
		),

		// JSON metrics
		JSONMarshalDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "nephoran_json_marshal_duration_seconds",
				Help:    "JSON marshaling duration in seconds",
				Buckets: prometheus.ExponentialBuckets(0.000001, 2, 20), // 1µs to ~1s
			},
		),
		JSONUnmarshalDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "nephoran_json_unmarshal_duration_seconds",
				Help:    "JSON unmarshaling duration in seconds",
				Buckets: prometheus.ExponentialBuckets(0.000001, 2, 20), // 1µs to ~1s
			},
		),
		JSONProcessingErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "nephoran_json_processing_errors_total",
				Help: "Total number of JSON processing errors",
			},
		),

		// Memory metrics
		HeapAlloc: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "nephoran_memory_heap_alloc_bytes",
				Help: "Number of heap bytes allocated and still in use",
			},
		),
		HeapSys: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "nephoran_memory_heap_sys_bytes",
				Help: "Number of heap bytes obtained from system",
			},
		),
		HeapIdle: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "nephoran_memory_heap_idle_bytes",
				Help: "Number of heap bytes waiting to be used",
			},
		),
		HeapInuse: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "nephoran_memory_heap_inuse_bytes",
				Help: "Number of heap bytes that are in use",
			},
		),
		HeapReleased: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "nephoran_memory_heap_released_bytes",
				Help: "Number of heap bytes released to OS",
			},
		),
		StackInuse: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "nephoran_memory_stack_inuse_bytes",
				Help: "Number of bytes in stack spans",
			},
		),
		StackSys: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "nephoran_memory_stack_sys_bytes",
				Help: "Number of bytes obtained from system for stack allocator",
			},
		),
		MSpanInuse: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "nephoran_memory_mspan_inuse_bytes",
				Help: "Number of bytes in mspan structures",
			},
		),
		MSpanSys: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "nephoran_memory_mspan_sys_bytes",
				Help: "Number of bytes obtained from system for mspan structures",
			},
		),
		MCacheInuse: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "nephoran_memory_mcache_inuse_bytes",
				Help: "Number of bytes in mcache structures",
			},
		),
		MCacheSys: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "nephoran_memory_mcache_sys_bytes",
				Help: "Number of bytes obtained from system for mcache structures",
			},
		),
		GCPauseTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "nephoran_gc_pause_total_seconds",
				Help: "Total GC pause duration in seconds",
			},
		),
		GCCount: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "nephoran_gc_count_total",
				Help: "Total number of GC cycles",
			},
		),
		Goroutines: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "nephoran_goroutines",
				Help: "Number of goroutines",
			},
		),

		// HTTP/2 metrics
		HTTP2StreamsActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "nephoran_http2_streams_active",
				Help: "Number of active HTTP/2 streams",
			},
		),
		HTTP2StreamsTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "nephoran_http2_streams_total",
				Help: "Total number of HTTP/2 streams created",
			},
		),
		HTTP2FramesReceived: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephoran_http2_frames_received_total",
				Help: "Total number of HTTP/2 frames received by type",
			},
			[]string{"frame_type"},
		),
		HTTP2FramesSent: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephoran_http2_frames_sent_total",
				Help: "Total number of HTTP/2 frames sent by type",
			},
			[]string{"frame_type"},
		),
		HTTP2ConnectionErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "nephoran_http2_connection_errors_total",
				Help: "Total number of HTTP/2 connection errors",
			},
		),

		// Cache metrics
		CacheHits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephoran_cache_hits_total",
				Help: "Total number of cache hits",
			},
			[]string{"cache_name"},
		),
		CacheMisses: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephoran_cache_misses_total",
				Help: "Total number of cache misses",
			},
			[]string{"cache_name"},
		),
		CacheEvictions: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephoran_cache_evictions_total",
				Help: "Total number of cache evictions",
			},
			[]string{"cache_name"},
		),
		CacheSize: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "nephoran_cache_size_bytes",
				Help: "Current cache size in bytes",
			},
			[]string{"cache_name"},
		),

		// Database metrics
		DBConnectionsActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "nephoran_db_connections_active",
				Help: "Number of active database connections",
			},
		),
		DBConnectionsIdle: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "nephoran_db_connections_idle",
				Help: "Number of idle database connections",
			},
		),
		DBQueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "nephoran_db_query_duration_seconds",
				Help:    "Database query duration in seconds",
				Buckets: prometheus.ExponentialBuckets(0.0001, 2, 15), // 100µs to ~3.2s
			},
			[]string{"query_type", "table"},
		),
		DBQueryErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephoran_db_query_errors_total",
				Help: "Total number of database query errors",
			},
			[]string{"query_type", "error_type"},
		),

		// Business metrics
		IntentsProcessed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephoran_intents_processed_total",
				Help: "Total number of intents processed",
			},
			[]string{"intent_type", "status"},
		),
		IntentLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "nephoran_intent_processing_duration_seconds",
				Help:    "Intent processing duration in seconds",
				Buckets: prometheus.ExponentialBuckets(0.01, 2, 15), // 10ms to ~327s
			},
			[]string{"intent_type"},
		),
		ScalingOperations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephoran_scaling_operations_total",
				Help: "Total number of scaling operations",
			},
			[]string{"direction", "resource_type", "status"},
		),
		ScalingDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "nephoran_scaling_duration_seconds",
				Help:    "Scaling operation duration in seconds",
				Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1s to ~512s
			},
			[]string{"direction", "resource_type"},
		),
	}

	// Register all metrics
	registry.MustRegister(
		m.RequestDuration,
		m.RequestCounter,
		m.RequestInFlight,
		m.JSONMarshalDuration,
		m.JSONUnmarshalDuration,
		m.JSONProcessingErrors,
		m.HeapAlloc,
		m.HeapSys,
		m.HeapIdle,
		m.HeapInuse,
		m.HeapReleased,
		m.StackInuse,
		m.StackSys,
		m.MSpanInuse,
		m.MSpanSys,
		m.MCacheInuse,
		m.MCacheSys,
		m.GCPauseTotal,
		m.GCCount,
		m.Goroutines,
		m.HTTP2StreamsActive,
		m.HTTP2StreamsTotal,
		m.HTTP2FramesReceived,
		m.HTTP2FramesSent,
		m.HTTP2ConnectionErrors,
		m.CacheHits,
		m.CacheMisses,
		m.CacheEvictions,
		m.CacheSize,
		m.DBConnectionsActive,
		m.DBConnectionsIdle,
		m.DBQueryDuration,
		m.DBQueryErrors,
		m.IntentsProcessed,
		m.IntentLatency,
		m.ScalingOperations,
		m.ScalingDuration,
	)

	// Start memory stats collector
	go m.collectMemoryStats()

	return m
}

// collectMemoryStats periodically collects memory statistics
func (m *Metrics) collectMemoryStats() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)

		m.HeapAlloc.Set(float64(ms.HeapAlloc))
		m.HeapSys.Set(float64(ms.HeapSys))
		m.HeapIdle.Set(float64(ms.HeapIdle))
		m.HeapInuse.Set(float64(ms.HeapInuse))
		m.HeapReleased.Set(float64(ms.HeapReleased))
		m.StackInuse.Set(float64(ms.StackInuse))
		m.StackSys.Set(float64(ms.StackSys))
		m.MSpanInuse.Set(float64(ms.MSpanInuse))
		m.MSpanSys.Set(float64(ms.MSpanSys))
		m.MCacheInuse.Set(float64(ms.MCacheInuse))
		m.MCacheSys.Set(float64(ms.MCacheSys))
		m.GCPauseTotal.Set(float64(ms.PauseTotalNs) / 1e9)
		m.GCCount.Set(float64(ms.NumGC))
		m.Goroutines.Set(float64(runtime.NumGoroutine()))
	}
}

// RecordRequestDuration records the duration of an HTTP request
func (m *Metrics) RecordRequestDuration(method, endpoint, status string, duration time.Duration) {
	m.RequestDuration.WithLabelValues(method, endpoint, status).Observe(duration.Seconds())
	m.RequestCounter.WithLabelValues(method, endpoint, status).Inc()
}

// RecordJSONMarshal records JSON marshaling performance
func (m *Metrics) RecordJSONMarshal(duration time.Duration, err error) {
	m.JSONMarshalDuration.Observe(duration.Seconds())
	if err != nil {
		m.JSONProcessingErrors.Inc()
	}
}

// RecordJSONUnmarshal records JSON unmarshaling performance
func (m *Metrics) RecordJSONUnmarshal(duration time.Duration, err error) {
	m.JSONUnmarshalDuration.Observe(duration.Seconds())
	if err != nil {
		m.JSONProcessingErrors.Inc()
	}
}

// RecordCacheHit records a cache hit
func (m *Metrics) RecordCacheHit(cacheName string) {
	m.CacheHits.WithLabelValues(cacheName).Inc()
}

// RecordCacheMiss records a cache miss
func (m *Metrics) RecordCacheMiss(cacheName string) {
	m.CacheMisses.WithLabelValues(cacheName).Inc()
}

// RecordCacheEviction records a cache eviction
func (m *Metrics) RecordCacheEviction(cacheName string) {
	m.CacheEvictions.WithLabelValues(cacheName).Inc()
}

// UpdateCacheSize updates the current cache size
func (m *Metrics) UpdateCacheSize(cacheName string, sizeBytes float64) {
	m.CacheSize.WithLabelValues(cacheName).Set(sizeBytes)
}

// RecordDBQuery records database query metrics
func (m *Metrics) RecordDBQuery(queryType, table string, duration time.Duration, err error) {
	m.DBQueryDuration.WithLabelValues(queryType, table).Observe(duration.Seconds())
	if err != nil {
		errorType := "unknown"
		if err.Error() != "" {
			// Simple error categorization
			switch {
			case contains(err.Error(), "timeout"):
				errorType = "timeout"
			case contains(err.Error(), "connection"):
				errorType = "connection"
			case contains(err.Error(), "syntax"):
				errorType = "syntax"
			default:
				errorType = "other"
			}
		}
		m.DBQueryErrors.WithLabelValues(queryType, errorType).Inc()
	}
}

// RecordIntentProcessing records intent processing metrics
func (m *Metrics) RecordIntentProcessing(intentType, status string, duration time.Duration) {
	m.IntentsProcessed.WithLabelValues(intentType, status).Inc()
	m.IntentLatency.WithLabelValues(intentType).Observe(duration.Seconds())
}

// RecordScalingOperation records scaling operation metrics
func (m *Metrics) RecordScalingOperation(direction, resourceType, status string, duration time.Duration) {
	m.ScalingOperations.WithLabelValues(direction, resourceType, status).Inc()
	m.ScalingDuration.WithLabelValues(direction, resourceType).Observe(duration.Seconds())
}

// Helper function for string contains
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || 
		   len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
		   findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}