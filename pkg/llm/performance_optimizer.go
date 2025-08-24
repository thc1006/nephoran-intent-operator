package llm

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// PerformanceOptimizer provides comprehensive performance monitoring and optimization
type PerformanceOptimizer struct {
	logger *slog.Logger
	tracer trace.Tracer
	meter  metric.Meter

	// Latency tracking
	latencyBuffer []LatencyDataPoint
	bufferMutex   sync.RWMutex
	maxBufferSize int

	// Performance metrics
	metrics *PerformanceMetrics

	// Circuit breaker integration
	circuitBreaker *AdvancedCircuitBreaker

	// Batch processor
	batchProcessor *BatchProcessor

	// Configuration
	config *PerformanceConfig
}

// LatencyDataPoint represents a single latency measurement
type LatencyDataPoint struct {
	Timestamp    time.Time
	Duration     time.Duration
	IntentType   string
	ModelName    string
	TokenCount   int
	Success      bool
	ErrorType    string
	RequestSize  int
	ResponseSize int
	CacheHit     bool
	CircuitState string
	RetryCount   int
}

// PerformanceMetrics holds all performance-related metrics
type PerformanceMetrics struct {
	// Prometheus metrics
	requestLatencyHistogram  prometheus.HistogramVec
	tokenUsageCounter        prometheus.CounterVec
	tokenCostGauge           prometheus.GaugeVec
	circuitBreakerStateGauge prometheus.GaugeVec
	batchProcessingHistogram prometheus.HistogramVec
	requestThroughputCounter prometheus.CounterVec
	errorRateCounter         prometheus.CounterVec
	cacheHitRateGauge        prometheus.GaugeVec

	// OpenTelemetry metrics
	requestLatencyHistogramOTel metric.Float64Histogram
	tokenUsageCounterOTel       metric.Int64Counter
	circuitBreakerCounterOTel   metric.Int64Counter
	batchSizeHistogramOTel      metric.Int64Histogram
}

// PerformanceConfig holds configuration for performance optimization
type PerformanceConfig struct {
	LatencyBufferSize     int                  `json:"latency_buffer_size"`
	OptimizationInterval  time.Duration        `json:"optimization_interval"`
	CircuitBreakerConfig  PerformanceCircuitBreakerConfig `json:"circuit_breaker"`
	BatchProcessingConfig BatchConfig          `json:"batch_processing"`
	MetricsExportInterval time.Duration        `json:"metrics_export_interval"`
	EnableTracing         bool                 `json:"enable_tracing"`
	TraceSamplingRatio    float64              `json:"trace_sampling_ratio"`
}

// PerformanceCircuitBreakerConfig holds advanced circuit breaker configuration for performance optimizer
type PerformanceCircuitBreakerConfig struct {
	FailureThreshold      int           `json:"failure_threshold"`
	SuccessThreshold      int           `json:"success_threshold"`
	Timeout               time.Duration `json:"timeout"`
	MaxConcurrentRequests int           `json:"max_concurrent_requests"`
	EnableAdaptiveTimeout bool          `json:"enable_adaptive_timeout"`
}

// BatchConfig holds batch processing configuration
type BatchConfig struct {
	MaxBatchSize         int           `json:"max_batch_size"`
	BatchTimeout         time.Duration `json:"batch_timeout"`
	ConcurrentBatches    int           `json:"concurrent_batches"`
	EnablePrioritization bool          `json:"enable_prioritization"`
}

// NewPerformanceOptimizer creates a new performance optimizer
func NewPerformanceOptimizer(config *PerformanceConfig) *PerformanceOptimizer {
	if config == nil {
		config = getDefaultPerformanceConfig()
	}

	logger := slog.Default().With("component", "performance-optimizer")
	tracer := otel.Tracer("nephoran-intent-operator/llm")
	meter := otel.Meter("nephoran-intent-operator/llm")

	po := &PerformanceOptimizer{
		logger:        logger,
		tracer:        tracer,
		meter:         meter,
		latencyBuffer: make([]LatencyDataPoint, 0, config.LatencyBufferSize),
		maxBufferSize: config.LatencyBufferSize,
		config:        config,
	}

	// Initialize metrics
	po.initializeMetrics()

	// Initialize circuit breaker
	// Convert to regular CircuitBreakerConfig for the circuit breaker
	cbConfig := &CircuitBreakerConfig{
		FailureThreshold:    config.CircuitBreakerConfig.FailureThreshold,
		SuccessThreshold:    config.CircuitBreakerConfig.SuccessThreshold,
		Timeout:             config.CircuitBreakerConfig.Timeout,
		MaxRequests:         config.CircuitBreakerConfig.MaxConcurrentRequests,
	}
	po.circuitBreaker = NewAdvancedCircuitBreaker(cbConfig)

	// Initialize batch processor
	po.batchProcessor = NewBatchProcessor(config.BatchProcessingConfig)

	// Start background optimization routine
	go po.optimizationRoutine()

	return po
}

// getDefaultPerformanceConfig returns default performance configuration
func getDefaultPerformanceConfig() *PerformanceConfig {
	return &PerformanceConfig{
		LatencyBufferSize:     10000,
		OptimizationInterval:  time.Minute,
		MetricsExportInterval: 30 * time.Second,
		EnableTracing:         true,
		TraceSamplingRatio:    0.1,
		CircuitBreakerConfig: PerformanceCircuitBreakerConfig{
			FailureThreshold:      5,
			SuccessThreshold:      3,
			Timeout:               30 * time.Second,
			MaxConcurrentRequests: 100,
			EnableAdaptiveTimeout: true,
		},
		BatchProcessingConfig: BatchConfig{
			MaxBatchSize:         10,
			BatchTimeout:         100 * time.Millisecond,
			ConcurrentBatches:    5,
			EnablePrioritization: true,
		},
	}
}

// initializeMetrics sets up Prometheus and OpenTelemetry metrics
func (po *PerformanceOptimizer) initializeMetrics() {
	// Prometheus metrics
	po.metrics = &PerformanceMetrics{
		requestLatencyHistogram: *promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "llm_request_duration_seconds",
			Help:    "Duration of LLM requests in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		}, []string{"intent_type", "model_name", "success", "cache_hit"}),

		tokenUsageCounter: *promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "llm_tokens_total",
			Help: "Total number of tokens processed",
		}, []string{"model_name", "token_type", "intent_type"}),

		tokenCostGauge: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "llm_token_cost_usd",
			Help: "Cost of token usage in USD",
		}, []string{"model_name", "token_type"}),

		circuitBreakerStateGauge: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "llm_circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
		}, []string{"backend"}),

		batchProcessingHistogram: *promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "llm_batch_processing_duration_seconds",
			Help:    "Duration of batch processing operations",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		}, []string{"batch_size_range"}),

		requestThroughputCounter: *promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "llm_requests_per_second_total",
			Help: "Request throughput per second",
		}, []string{"intent_type", "model_name"}),

		errorRateCounter: *promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "llm_errors_total",
			Help: "Total number of LLM request errors",
		}, []string{"error_type", "model_name"}),

		cacheHitRateGauge: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "llm_cache_hit_rate",
			Help: "Cache hit rate percentage",
		}, []string{"cache_type"}),
	}

	// OpenTelemetry metrics
	var err error
	po.metrics.requestLatencyHistogramOTel, err = po.meter.Float64Histogram(
		"llm.request.duration",
		metric.WithDescription("Duration of LLM requests"),
		metric.WithUnit("s"),
	)
	if err != nil {
		po.logger.Error("Failed to create OpenTelemetry latency histogram", "error", err)
	}

	po.metrics.tokenUsageCounterOTel, err = po.meter.Int64Counter(
		"llm.tokens.total",
		metric.WithDescription("Total number of tokens processed"),
	)
	if err != nil {
		po.logger.Error("Failed to create OpenTelemetry token counter", "error", err)
	}

	po.metrics.circuitBreakerCounterOTel, err = po.meter.Int64Counter(
		"llm.circuit_breaker.state_changes",
		metric.WithDescription("Circuit breaker state changes"),
	)
	if err != nil {
		po.logger.Error("Failed to create OpenTelemetry circuit breaker counter", "error", err)
	}

	po.metrics.batchSizeHistogramOTel, err = po.meter.Int64Histogram(
		"llm.batch.size",
		metric.WithDescription("Batch size distribution"),
	)
	if err != nil {
		po.logger.Error("Failed to create OpenTelemetry batch size histogram", "error", err)
	}
}

// RecordLatency records a latency data point
func (po *PerformanceOptimizer) RecordLatency(dataPoint LatencyDataPoint) {
	po.bufferMutex.Lock()
	defer po.bufferMutex.Unlock()

	// Add to buffer
	po.latencyBuffer = append(po.latencyBuffer, dataPoint)

	// Maintain buffer size
	if len(po.latencyBuffer) > po.maxBufferSize {
		// Remove oldest entries
		copy(po.latencyBuffer, po.latencyBuffer[len(po.latencyBuffer)-po.maxBufferSize:])
		po.latencyBuffer = po.latencyBuffer[:po.maxBufferSize]
	}

	// Update metrics
	po.updateMetrics(dataPoint)
}

// updateMetrics updates Prometheus and OpenTelemetry metrics
func (po *PerformanceOptimizer) updateMetrics(dataPoint LatencyDataPoint) {
	// Prometheus metrics
	labels := prometheus.Labels{
		"intent_type": dataPoint.IntentType,
		"model_name":  dataPoint.ModelName,
		"success":     fmt.Sprintf("%t", dataPoint.Success),
		"cache_hit":   fmt.Sprintf("%t", dataPoint.CacheHit),
	}
	po.metrics.requestLatencyHistogram.With(labels).Observe(dataPoint.Duration.Seconds())

	// Token usage
	if dataPoint.TokenCount > 0 {
		tokenLabels := prometheus.Labels{
			"model_name":  dataPoint.ModelName,
			"token_type":  "total",
			"intent_type": dataPoint.IntentType,
		}
		po.metrics.tokenUsageCounter.With(tokenLabels).Add(float64(dataPoint.TokenCount))
	}

	// Error tracking
	if !dataPoint.Success && dataPoint.ErrorType != "" {
		errorLabels := prometheus.Labels{
			"error_type": dataPoint.ErrorType,
			"model_name": dataPoint.ModelName,
		}
		po.metrics.errorRateCounter.With(errorLabels).Inc()
	}

	// OpenTelemetry metrics
	otelLabels := []attribute.KeyValue{
		attribute.String("intent_type", dataPoint.IntentType),
		attribute.String("model_name", dataPoint.ModelName),
		attribute.Bool("success", dataPoint.Success),
		attribute.Bool("cache_hit", dataPoint.CacheHit),
	}

	po.metrics.requestLatencyHistogramOTel.Record(context.Background(), dataPoint.Duration.Seconds(), metric.WithAttributes(otelLabels...))

	if dataPoint.TokenCount > 0 {
		po.metrics.tokenUsageCounterOTel.Add(context.Background(), int64(dataPoint.TokenCount), metric.WithAttributes(otelLabels...))
	}
}

// GetLatencyProfile returns a comprehensive latency profile
func (po *PerformanceOptimizer) GetLatencyProfile() *LatencyProfile {
	po.bufferMutex.RLock()
	defer po.bufferMutex.RUnlock()

	if len(po.latencyBuffer) == 0 {
		return &LatencyProfile{}
	}

	// Create a copy for analysis
	data := make([]LatencyDataPoint, len(po.latencyBuffer))
	copy(data, po.latencyBuffer)

	// Sort by duration
	sort.Slice(data, func(i, j int) bool {
		return data[i].Duration < data[j].Duration
	})

	profile := &LatencyProfile{
		TotalRequests: len(data),
		Percentiles:   calculatePercentiles(data),
		ByIntentType:  make(map[string]*IntentTypeProfile),
		ByModelName:   make(map[string]*ModelProfile),
		TimeRange: TimeRange{
			Start: data[0].Timestamp,
			End:   data[len(data)-1].Timestamp,
		},
	}

	// Calculate success rate
	successCount := 0
	for _, dp := range data {
		if dp.Success {
			successCount++
		}
	}
	profile.SuccessRate = float64(successCount) / float64(len(data))

	// Group by intent type
	intentGroups := make(map[string][]LatencyDataPoint)
	for _, dp := range data {
		intentGroups[dp.IntentType] = append(intentGroups[dp.IntentType], dp)
	}

	for intentType, group := range intentGroups {
		profile.ByIntentType[intentType] = &IntentTypeProfile{
			Count:       len(group),
			Percentiles: calculatePercentiles(group),
			SuccessRate: calculateSuccessRate(group),
		}
	}

	// Group by model
	modelGroups := make(map[string][]LatencyDataPoint)
	for _, dp := range data {
		modelGroups[dp.ModelName] = append(modelGroups[dp.ModelName], dp)
	}

	for modelName, group := range modelGroups {
		profile.ByModelName[modelName] = &ModelProfile{
			Count:         len(group),
			Percentiles:   calculatePercentiles(group),
			SuccessRate:   calculateSuccessRate(group),
			AvgTokenCount: calculateAvgTokenCount(group),
		}
	}

	return profile
}

// LatencyProfile represents a comprehensive latency analysis
type LatencyProfile struct {
	TotalRequests int                           `json:"total_requests"`
	SuccessRate   float64                       `json:"success_rate"`
	Percentiles   LatencyPercentiles            `json:"percentiles"`
	ByIntentType  map[string]*IntentTypeProfile `json:"by_intent_type"`
	ByModelName   map[string]*ModelProfile      `json:"by_model_name"`
	TimeRange     TimeRange                     `json:"time_range"`
}

// LatencyPercentiles holds percentile data
type LatencyPercentiles struct {
	P50 time.Duration `json:"p50"`
	P75 time.Duration `json:"p75"`
	P90 time.Duration `json:"p90"`
	P95 time.Duration `json:"p95"`
	P99 time.Duration `json:"p99"`
}

// IntentTypeProfile holds metrics for a specific intent type
type IntentTypeProfile struct {
	Count       int                `json:"count"`
	Percentiles LatencyPercentiles `json:"percentiles"`
	SuccessRate float64            `json:"success_rate"`
}

// ModelProfile holds metrics for a specific model
type ModelProfile struct {
	Count         int                `json:"count"`
	Percentiles   LatencyPercentiles `json:"percentiles"`
	SuccessRate   float64            `json:"success_rate"`
	AvgTokenCount float64            `json:"avg_token_count"`
}

// TimeRange represents a time range
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// optimizationRoutine runs background optimization tasks
func (po *PerformanceOptimizer) optimizationRoutine() {
	ticker := time.NewTicker(po.config.OptimizationInterval)
	defer ticker.Stop()

	for range ticker.C {
		po.performOptimization()
	}
}

// performOptimization analyzes performance data and makes optimizations
func (po *PerformanceOptimizer) performOptimization() {
	profile := po.GetLatencyProfile()

	// Log performance insights
	po.logger.Info("Performance optimization analysis",
		"total_requests", profile.TotalRequests,
		"success_rate", profile.SuccessRate,
		"p50_latency", profile.Percentiles.P50,
		"p95_latency", profile.Percentiles.P95,
		"p99_latency", profile.Percentiles.P99,
	)

	// Adaptive circuit breaker tuning
	if po.config.CircuitBreakerConfig.EnableAdaptiveTimeout {
		po.adaptCircuitBreakerTimeout(profile)
	}

	// Update cache hit rate metrics
	po.updateCacheMetrics()
}

// adaptCircuitBreakerTimeout adjusts circuit breaker timeout based on latency patterns
func (po *PerformanceOptimizer) adaptCircuitBreakerTimeout(profile *LatencyProfile) {
	// Use P95 latency + buffer as the new timeout
	adaptiveTimeout := profile.Percentiles.P95 + (profile.Percentiles.P95 / 2)

	// Ensure reasonable bounds
	minTimeout := 5 * time.Second
	maxTimeout := 60 * time.Second

	if adaptiveTimeout < minTimeout {
		adaptiveTimeout = minTimeout
	} else if adaptiveTimeout > maxTimeout {
		adaptiveTimeout = maxTimeout
	}

	// Update circuit breaker configuration
	po.circuitBreaker.UpdateTimeout(adaptiveTimeout)

	po.logger.Debug("Adapted circuit breaker timeout",
		"new_timeout", adaptiveTimeout,
		"p95_latency", profile.Percentiles.P95,
	)
}

// updateCacheMetrics updates cache-related metrics
func (po *PerformanceOptimizer) updateCacheMetrics() {
	po.bufferMutex.RLock()
	defer po.bufferMutex.RUnlock()

	if len(po.latencyBuffer) == 0 {
		return
	}

	cacheHits := 0
	for _, dp := range po.latencyBuffer {
		if dp.CacheHit {
			cacheHits++
		}
	}

	hitRate := float64(cacheHits) / float64(len(po.latencyBuffer)) * 100
	po.metrics.cacheHitRateGauge.WithLabelValues("response").Set(hitRate)
}

// Helper functions
func calculatePercentiles(data []LatencyDataPoint) LatencyPercentiles {
	if len(data) == 0 {
		return LatencyPercentiles{}
	}

	return LatencyPercentiles{
		P50: data[int(float64(len(data))*0.50)].Duration,
		P75: data[int(float64(len(data))*0.75)].Duration,
		P90: data[int(float64(len(data))*0.90)].Duration,
		P95: data[int(float64(len(data))*0.95)].Duration,
		P99: data[int(float64(len(data))*0.99)].Duration,
	}
}

func calculateSuccessRate(data []LatencyDataPoint) float64 {
	if len(data) == 0 {
		return 0
	}

	successCount := 0
	for _, dp := range data {
		if dp.Success {
			successCount++
		}
	}

	return float64(successCount) / float64(len(data))
}

func calculateAvgTokenCount(data []LatencyDataPoint) float64 {
	if len(data) == 0 {
		return 0
	}

	totalTokens := 0
	for _, dp := range data {
		totalTokens += dp.TokenCount
	}

	return float64(totalTokens) / float64(len(data))
}

// GetMetrics returns the current metrics
func (po *PerformanceOptimizer) GetMetrics() *PerformanceMetrics {
	return po.metrics
}

// GetCircuitBreaker returns the circuit breaker instance
func (po *PerformanceOptimizer) GetCircuitBreaker() *AdvancedCircuitBreaker {
	return po.circuitBreaker
}

// GetBatchProcessor returns the batch processor instance
func (po *PerformanceOptimizer) GetBatchProcessor() *BatchProcessor {
	return po.batchProcessor
}

// Close gracefully shuts down the performance optimizer
func (po *PerformanceOptimizer) Close() error {
	po.logger.Info("Shutting down performance optimizer")

	if po.batchProcessor != nil {
		po.batchProcessor.Close()
	}

	return nil
}
