//go:build !disable_rag
// +build !disable_rag

package llm

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// EnhancedPerformanceClient provides a high-performance LLM client with all optimizations
type EnhancedPerformanceClient struct {
	// Core components
	baseClient     *Client
	performanceOpt *PerformanceOptimizer
	retryEngine    *RetryEngine
	batchProcessor BatchProcessor
	circuitBreaker *AdvancedCircuitBreaker

	// Configuration
	config *EnhancedClientConfig

	// Observability
	logger *slog.Logger
	tracer trace.Tracer
	meter  metric.Meter

	// Metrics
	prometheusMetrics *EnhancedPrometheusMetrics
	otelMetrics       *EnhancedOTelMetrics

	// State management
	isHealthy        bool
	lastHealthCheck  time.Time
	healthCheckMutex sync.RWMutex

	// Token management and cost tracking
	tokenTracker   *TokenTracker
	costCalculator *CostCalculator

	// Request context management
	activeRequests map[string]*RequestContext
	requestsMutex  sync.RWMutex
}

// EnhancedClientConfig holds configuration for the enhanced client
type EnhancedClientConfig struct {
	// Base client config
	BaseConfig ClientConfig

	// Performance optimization config
	PerformanceConfig *PerformanceConfig

	// Retry configuration
	RetryConfig RetryEngineConfig

	// Batch processing config
	BatchConfig BatchConfig

	// Circuit breaker config
	CircuitBreakerConfig CircuitBreakerConfig

	// Observability config
	TracingConfig TracingConfig
	MetricsConfig MetricsConfig

	// Health check config
	HealthCheckConfig HealthCheckConfig

	// Token and cost tracking
	TokenConfig TokenConfig
}

// TracingConfig holds tracing configuration
type TracingConfig struct {
	Enabled        bool    `json:"enabled"`
	SamplingRatio  float64 `json:"sampling_ratio"`
	ServiceName    string  `json:"service_name"`
	ServiceVersion string  `json:"service_version"`
}

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	Enabled          bool          `json:"enabled"`
	ExportInterval   time.Duration `json:"export_interval"`
	HistogramBuckets []float64     `json:"histogram_buckets"`
}

// HealthCheckConfig holds health check configuration
type HealthCheckConfig struct {
	Enabled          bool          `json:"enabled"`
	Interval         time.Duration `json:"interval"`
	Timeout          time.Duration `json:"timeout"`
	FailureThreshold int           `json:"failure_threshold"`
}

// TokenConfig holds token tracking configuration
type TokenConfig struct {
	TrackUsage     bool               `json:"track_usage"`
	TrackCosts     bool               `json:"track_costs"`
	CostPerToken   map[string]float64 `json:"cost_per_token"`
	BudgetLimit    float64            `json:"budget_limit"`
	AlertThreshold float64            `json:"alert_threshold"`
}

// EnhancedPrometheusMetrics holds all Prometheus metrics
type EnhancedPrometheusMetrics struct {
	// Request metrics
	requestDuration  *prometheus.HistogramVec
	requestsTotal    *prometheus.CounterVec
	requestsInFlight *prometheus.GaugeVec

	// Token and cost metrics
	tokensUsed        *prometheus.CounterVec
	tokenCosts        *prometheus.CounterVec
	budgetUtilization *prometheus.GaugeVec

	// Performance metrics
	cacheHitRate    *prometheus.GaugeVec
	batchEfficiency *prometheus.HistogramVec
	retryRate       *prometheus.GaugeVec

	// Circuit breaker metrics
	circuitBreakerState *prometheus.GaugeVec
	circuitBreakerTrips *prometheus.CounterVec

	// Error metrics
	errorRate    *prometheus.GaugeVec
	errorsByType *prometheus.CounterVec
}

// EnhancedOTelMetrics holds all OpenTelemetry metrics
type EnhancedOTelMetrics struct {
	requestDuration     metric.Float64Histogram
	requestsTotal       metric.Int64Counter
	tokensUsed          metric.Int64Counter
	tokenCosts          metric.Float64Counter
	cacheHits           metric.Int64Counter
	circuitBreakerTrips metric.Int64Counter
	retryAttempts       metric.Int64Counter
}

// TokenTracker tracks token usage across requests
type TokenTracker struct {
	totalTokens    int64
	tokensByModel  map[string]int64
	tokensByIntent map[string]int64
	mutex          sync.RWMutex
}

// CostCalculator calculates and tracks costs
type CostCalculator struct {
	costPerToken   map[string]float64
	totalCost      float64
	costsByModel   map[string]float64
	budgetLimit    float64
	alertThreshold float64
	mutex          sync.RWMutex
}

// NewEnhancedPerformanceClient creates a new enhanced performance client
func NewEnhancedPerformanceClient(config *EnhancedClientConfig) (*EnhancedPerformanceClient, error) {
	if config == nil {
		config = getDefaultEnhancedConfig()
	}

	// Create base client
	baseClient := NewClientWithConfig("", config.BaseConfig)

	// Create circuit breaker
	circuitBreaker := NewAdvancedCircuitBreaker(config.CircuitBreakerConfig)

	// Create performance optimizer
	performanceOpt := NewPerformanceOptimizer(config.PerformanceConfig)

	// Create retry engine
	retryEngine := NewRetryEngine(config.RetryConfig, circuitBreaker)

	// Create batch processor
	batchProcessor := NewBatchProcessor(config.BatchConfig)

	// Create observability components
	logger := slog.Default().With("component", "enhanced-llm-client")
	tracer := otel.Tracer("nephoran-intent-operator/enhanced-llm")
	meter := otel.Meter("nephoran-intent-operator/enhanced-llm")

	client := &EnhancedPerformanceClient{
		baseClient:     baseClient,
		performanceOpt: performanceOpt,
		retryEngine:    retryEngine,
		batchProcessor: batchProcessor,
		circuitBreaker: circuitBreaker,
		config:         config,
		logger:         logger,
		tracer:         tracer,
		meter:          meter,
		isHealthy:      true,
		activeRequests: make(map[string]*RequestContext),
		tokenTracker:   NewTokenTracker(),
		costCalculator: NewCostCalculator(config.TokenConfig.CostPerToken, config.TokenConfig.BudgetLimit, config.TokenConfig.AlertThreshold),
	}

	// Initialize metrics
	if err := client.initializeMetrics(); err != nil {
		return nil, fmt.Errorf("failed to initialize metrics: %w", err)
	}

	// Start health checking if enabled
	if config.HealthCheckConfig.Enabled {
		go client.healthCheckRoutine()
	}

	// Register circuit breaker callbacks
	circuitBreaker.AddStateChangeCallback(client.onCircuitBreakerStateChange)

	logger.Info("Enhanced performance client initialized",
		"tracing_enabled", config.TracingConfig.Enabled,
		"metrics_enabled", config.MetricsConfig.Enabled,
		"health_check_enabled", config.HealthCheckConfig.Enabled,
	)

	return client, nil
}

// getDefaultEnhancedConfig returns default configuration
func getDefaultEnhancedConfig() *EnhancedClientConfig {
	return &EnhancedClientConfig{
		BaseConfig: ClientConfig{
			ModelName:   "gpt-4o-mini",
			MaxTokens:   2048,
			BackendType: "openai",
			Timeout:     60 * time.Second,
		},
		PerformanceConfig: getDefaultPerformanceConfig(),
		RetryConfig: RetryEngineConfig{
			DefaultStrategy:      "exponential",
			MaxRetries:           3,
			BaseDelay:            time.Second,
			MaxDelay:             30 * time.Second,
			JitterEnabled:        true,
			BackoffMultiplier:    2.0,
			EnableAdaptive:       true,
			EnableClassification: true,
		},
		BatchConfig: BatchConfig{
			MaxBatchSize:         10,
			BatchTimeout:         100 * time.Millisecond,
			ConcurrentBatches:    5,
			EnablePrioritization: true,
		},
		CircuitBreakerConfig: CircuitBreakerConfig{
			FailureThreshold:      5,
			SuccessThreshold:      3,
			Timeout:               30 * time.Second,
			MaxConcurrentRequests: 100,
			EnableAdaptiveTimeout: true,
		},
		TracingConfig: TracingConfig{
			Enabled:        true,
			SamplingRatio:  0.1,
			ServiceName:    "nephoran-llm-client",
			ServiceVersion: "1.0.0",
		},
		MetricsConfig: MetricsConfig{
			Enabled:          true,
			ExportInterval:   30 * time.Second,
			HistogramBuckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		HealthCheckConfig: HealthCheckConfig{
			Enabled:          true,
			Interval:         30 * time.Second,
			Timeout:          10 * time.Second,
			FailureThreshold: 3,
		},
		TokenConfig: TokenConfig{
			TrackUsage:     true,
			TrackCosts:     true,
			CostPerToken:   map[string]float64{"gpt-4o-mini": 0.00015, "gpt-4": 0.03},
			BudgetLimit:    100.0,
			AlertThreshold: 80.0,
		},
	}
}

// initializeMetrics sets up Prometheus and OpenTelemetry metrics
func (c *EnhancedPerformanceClient) initializeMetrics() error {
	if !c.config.MetricsConfig.Enabled {
		return nil
	}

	// Initialize Prometheus metrics
	c.prometheusMetrics = &EnhancedPrometheusMetrics{
		requestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "llm_request_duration_seconds",
			Help:    "Duration of LLM requests",
			Buckets: c.config.MetricsConfig.HistogramBuckets,
		}, []string{"model", "intent_type", "success", "cache_hit", "batch"}),

		requestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "llm_requests_total",
			Help: "Total number of LLM requests",
		}, []string{"model", "intent_type", "status"}),

		requestsInFlight: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "llm_requests_in_flight",
			Help: "Current number of in-flight LLM requests",
		}, []string{"model"}),

		tokensUsed: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "llm_tokens_used_total",
			Help: "Total number of tokens used",
		}, []string{"model", "token_type", "intent_type"}),

		tokenCosts: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "llm_token_costs_usd_total",
			Help: "Total cost of token usage in USD",
		}, []string{"model", "intent_type"}),

		budgetUtilization: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "llm_budget_utilization_percent",
			Help: "Current budget utilization percentage",
		}, []string{}),

		cacheHitRate: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "llm_cache_hit_rate_percent",
			Help: "Cache hit rate percentage",
		}, []string{"cache_type"}),

		batchEfficiency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "llm_batch_efficiency_ratio",
			Help:    "Batch processing efficiency ratio",
			Buckets: []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
		}, []string{"batch_size_range"}),

		retryRate: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "llm_retry_rate_percent",
			Help: "Retry rate percentage",
		}, []string{"strategy"}),

		circuitBreakerState: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "llm_circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
		}, []string{"backend"}),

		circuitBreakerTrips: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "llm_circuit_breaker_trips_total",
			Help: "Total number of circuit breaker trips",
		}, []string{"backend", "reason"}),

		errorRate: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "llm_error_rate_percent",
			Help: "Error rate percentage",
		}, []string{"model", "error_class"}),

		errorsByType: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "llm_errors_by_type_total",
			Help: "Total errors by type",
		}, []string{"error_type", "model"}),
	}

	// Initialize OpenTelemetry metrics
	var err error
	c.otelMetrics = &EnhancedOTelMetrics{}

	c.otelMetrics.requestDuration, err = c.meter.Float64Histogram(
		"llm.request.duration",
		metric.WithDescription("Duration of LLM requests"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return fmt.Errorf("failed to create request duration histogram: %w", err)
	}

	c.otelMetrics.requestsTotal, err = c.meter.Int64Counter(
		"llm.requests.total",
		metric.WithDescription("Total number of LLM requests"),
	)
	if err != nil {
		return fmt.Errorf("failed to create requests total counter: %w", err)
	}

	c.otelMetrics.tokensUsed, err = c.meter.Int64Counter(
		"llm.tokens.used",
		metric.WithDescription("Total tokens used"),
	)
	if err != nil {
		return fmt.Errorf("failed to create tokens used counter: %w", err)
	}

	c.otelMetrics.tokenCosts, err = c.meter.Float64Counter(
		"llm.token.costs",
		metric.WithDescription("Total token costs"),
		metric.WithUnit("USD"),
	)
	if err != nil {
		return fmt.Errorf("failed to create token costs counter: %w", err)
	}

	c.otelMetrics.cacheHits, err = c.meter.Int64Counter(
		"llm.cache.hits",
		metric.WithDescription("Cache hits"),
	)
	if err != nil {
		return fmt.Errorf("failed to create cache hits counter: %w", err)
	}

	c.otelMetrics.circuitBreakerTrips, err = c.meter.Int64Counter(
		"llm.circuit_breaker.trips",
		metric.WithDescription("Circuit breaker trips"),
	)
	if err != nil {
		return fmt.Errorf("failed to create circuit breaker trips counter: %w", err)
	}

	c.otelMetrics.retryAttempts, err = c.meter.Int64Counter(
		"llm.retry.attempts",
		metric.WithDescription("Retry attempts"),
	)
	if err != nil {
		return fmt.Errorf("failed to create retry attempts counter: %w", err)
	}

	return nil
}

// ProcessIntent processes an intent with all performance optimizations
func (c *EnhancedPerformanceClient) ProcessIntent(ctx context.Context, intent string) (string, error) {
	return c.ProcessIntentWithOptions(ctx, &IntentProcessingOptions{
		Intent:     intent,
		Priority:   PriorityNormal,
		UseBatch:   true,
		UseCache:   true,
		IntentType: "NetworkFunctionDeployment", // Default
		ModelName:  c.config.BaseConfig.ModelName,
	})
}

// IntentProcessingOptions holds options for intent processing
type IntentProcessingOptions struct {
	Intent     string
	IntentType string
	ModelName  string
	Priority   Priority
	UseBatch   bool
	UseCache   bool
	Timeout    time.Duration
	Metadata   map[string]interface{}
}

// ProcessIntentWithOptions processes an intent with specific options
func (c *EnhancedPerformanceClient) ProcessIntentWithOptions(ctx context.Context, options *IntentProcessingOptions) (string, error) {
	// Start tracing
	ctx, span := c.tracer.Start(ctx, "enhanced_client.process_intent")
	defer span.End()

	span.SetAttributes(
		attribute.String("intent.type", options.IntentType),
		attribute.String("model.name", options.ModelName),
		attribute.String("priority", fmt.Sprintf("%d", options.Priority)),
		attribute.Bool("use_batch", options.UseBatch),
		attribute.Bool("use_cache", options.UseCache),
	)

	// Create request context
	requestCtx := c.createRequestContext(options)
	defer c.completeRequestContext(requestCtx.ID)

	// Check health
	if !c.isHealthy {
		span.SetStatus(codes.Error, "client is unhealthy")
		return "", fmt.Errorf("client is currently unhealthy")
	}

	// Update in-flight metrics
	c.prometheusMetrics.requestsInFlight.WithLabelValues(options.ModelName).Inc()
	defer c.prometheusMetrics.requestsInFlight.WithLabelValues(options.ModelName).Dec()

	start := time.Now()
	var response string
	var err error
	var cacheHit bool

	// Try cache first if enabled
	if options.UseCache {
		if cachedResponse, found := c.getCachedResponse(options.Intent); found {
			response = cachedResponse
			cacheHit = true
			c.otelMetrics.cacheHits.Add(ctx, 1, metric.WithAttributes(
				attribute.String("model", options.ModelName),
				attribute.String("intent_type", options.IntentType),
			))
		}
	}

	// Process if not cached
	if response == "" {
		if options.UseBatch {
			// Use batch processing
			batchResult, batchErr := c.batchProcessor.ProcessRequest(
				ctx,
				options.Intent,
				options.IntentType,
				options.ModelName,
				options.Priority,
			)
			if batchErr != nil {
				err = batchErr
			} else {
				response = batchResult.Response
			}
		} else {
			// Use direct processing with retry logic
			retryResult, retryErr := c.retryEngine.ExecuteWithRetry(ctx, func() error {
				var processErr error
				response, processErr = c.baseClient.ProcessIntent(ctx, options.Intent)
				return processErr
			})

			if retryErr != nil {
				err = retryErr
			}

			// Record retry metrics
			if retryResult != nil {
				c.otelMetrics.retryAttempts.Add(ctx, int64(retryResult.TotalAttempts-1))
			}
		}
	}

	duration := time.Since(start)
	success := err == nil

	// Record latency data
	latencyData := LatencyDataPoint{
		Timestamp:    start,
		Duration:     duration,
		IntentType:   options.IntentType,
		ModelName:    options.ModelName,
		Success:      success,
		CacheHit:     cacheHit,
		RequestSize:  len(options.Intent),
		ResponseSize: len(response),
	}

	if err != nil {
		latencyData.ErrorType = fmt.Sprintf("%T", err)
	}

	c.performanceOpt.RecordLatency(latencyData)

	// Update metrics
	c.updateMetrics(options, duration, success, cacheHit)

	// Track tokens and costs
	if success && c.config.TokenConfig.TrackUsage {
		tokenCount := c.estimateTokenCount(options.Intent, response)
		c.trackTokenUsage(options.ModelName, options.IntentType, tokenCount)

		if c.config.TokenConfig.TrackCosts {
			c.trackTokenCosts(options.ModelName, options.IntentType, tokenCount)
		}
	}

	// Update span
	span.SetAttributes(
		attribute.Bool("success", success),
		attribute.Bool("cache_hit", cacheHit),
		attribute.Int64("duration_ms", duration.Milliseconds()),
		attribute.Int("response_size", len(response)),
	)

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	span.SetStatus(codes.Ok, "")
	return response, nil
}

// Helper methods

func (c *EnhancedPerformanceClient) createRequestContext(options *IntentProcessingOptions) *RequestContext {
	ctx := &RequestContext{
		ID:        generateRequestID(),
		Intent:    options.Intent,
		StartTime: time.Now(),
		Metadata:  options.Metadata,
	}

	c.requestsMutex.Lock()
	c.activeRequests[ctx.ID] = ctx
	c.requestsMutex.Unlock()

	return ctx
}


func (c *EnhancedPerformanceClient) completeRequestContext(id string) {
	c.requestsMutex.Lock()
	delete(c.activeRequests, id)
	c.requestsMutex.Unlock()
}

func (c *EnhancedPerformanceClient) getCachedResponse(intent string) (string, bool) {
	// Simple cache implementation - in production, use the existing cache
	return "", false
}

func (c *EnhancedPerformanceClient) updateMetrics(options *IntentProcessingOptions, duration time.Duration, success, cacheHit bool) {
	labels := prometheus.Labels{
		"model":       options.ModelName,
		"intent_type": options.IntentType,
		"success":     fmt.Sprintf("%t", success),
		"cache_hit":   fmt.Sprintf("%t", cacheHit),
		"batch":       fmt.Sprintf("%t", options.UseBatch),
	}

	c.prometheusMetrics.requestDuration.With(labels).Observe(duration.Seconds())

	status := "success"
	if !success {
		status = "error"
	}

	requestLabels := prometheus.Labels{
		"model":       options.ModelName,
		"intent_type": options.IntentType,
		"status":      status,
	}
	c.prometheusMetrics.requestsTotal.With(requestLabels).Inc()

	// OpenTelemetry metrics
	otelLabels := []attribute.KeyValue{
		attribute.String("model", options.ModelName),
		attribute.String("intent_type", options.IntentType),
		attribute.Bool("success", success),
		attribute.Bool("cache_hit", cacheHit),
	}

	c.otelMetrics.requestDuration.Record(context.Background(), duration.Seconds(), metric.WithAttributes(otelLabels...))
	c.otelMetrics.requestsTotal.Add(context.Background(), 1, metric.WithAttributes(otelLabels...))
}

func (c *EnhancedPerformanceClient) estimateTokenCount(input, output string) int {
	// Simple token estimation - in production, use proper tokenization
	return (len(input) + len(output)) / 4 // Rough approximation
}

func (c *EnhancedPerformanceClient) trackTokenUsage(modelName, intentType string, tokenCount int) {
	c.tokenTracker.mutex.Lock()
	c.tokenTracker.totalTokens += int64(tokenCount)
	c.tokenTracker.tokensByModel[modelName] += int64(tokenCount)
	c.tokenTracker.tokensByIntent[intentType] += int64(tokenCount)
	c.tokenTracker.mutex.Unlock()

	// Update metrics
	labels := prometheus.Labels{
		"model":       modelName,
		"token_type":  "total",
		"intent_type": intentType,
	}
	c.prometheusMetrics.tokensUsed.With(labels).Add(float64(tokenCount))

	c.otelMetrics.tokensUsed.Add(context.Background(), int64(tokenCount), metric.WithAttributes(
		attribute.String("model", modelName),
		attribute.String("intent_type", intentType),
	))
}

func (c *EnhancedPerformanceClient) trackTokenCosts(modelName, intentType string, tokenCount int) {
	cost := c.costCalculator.CalculateCost(modelName, tokenCount)

	c.costCalculator.mutex.Lock()
	c.costCalculator.totalCost += cost
	c.costCalculator.costsByModel[modelName] += cost
	c.costCalculator.mutex.Unlock()

	// Update metrics
	labels := prometheus.Labels{
		"model":       modelName,
		"intent_type": intentType,
	}
	c.prometheusMetrics.tokenCosts.With(labels).Add(cost)

	// Check budget
	utilization := (c.costCalculator.totalCost / c.costCalculator.budgetLimit) * 100
	c.prometheusMetrics.budgetUtilization.WithLabelValues().Set(utilization)

	c.otelMetrics.tokenCosts.Add(context.Background(), cost, metric.WithAttributes(
		attribute.String("model", modelName),
		attribute.String("intent_type", intentType),
	))

	// Alert if threshold exceeded
	if utilization > c.costCalculator.alertThreshold {
		c.logger.Warn("Budget alert threshold exceeded",
			"utilization", utilization,
			"threshold", c.costCalculator.alertThreshold,
			"total_cost", c.costCalculator.totalCost,
		)
	}
}

func (c *EnhancedPerformanceClient) healthCheckRoutine() {
	ticker := time.NewTicker(c.config.HealthCheckConfig.Interval)
	defer ticker.Stop()

	for range ticker.C {
		c.performHealthCheck()
	}
}

func (c *EnhancedPerformanceClient) performHealthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.HealthCheckConfig.Timeout)
	defer cancel()

	// Simple health check - try to process a minimal intent
	_, err := c.baseClient.ProcessIntent(ctx, "health check")

	c.healthCheckMutex.Lock()
	c.isHealthy = err == nil
	c.lastHealthCheck = time.Now()
	c.healthCheckMutex.Unlock()

	if err != nil {
		c.logger.Warn("Health check failed", "error", err)
	}
}

func (c *EnhancedPerformanceClient) onCircuitBreakerStateChange(oldState, newState CircuitState, reason string) {
	c.prometheusMetrics.circuitBreakerState.WithLabelValues(c.config.BaseConfig.BackendType).Set(float64(newState))
	c.prometheusMetrics.circuitBreakerTrips.WithLabelValues(c.config.BaseConfig.BackendType, reason).Inc()

	c.otelMetrics.circuitBreakerTrips.Add(context.Background(), 1, metric.WithAttributes(
		attribute.String("backend", c.config.BaseConfig.BackendType),
		attribute.String("reason", reason),
		attribute.Int("old_state", int(oldState)),
		attribute.Int("new_state", int(newState)),
	))

	c.logger.Info("Circuit breaker state changed",
		"old_state", oldState,
		"new_state", newState,
		"reason", reason,
	)
}

// Utility functions

func NewTokenTracker() *TokenTracker {
	return &TokenTracker{
		tokensByModel:  make(map[string]int64),
		tokensByIntent: make(map[string]int64),
	}
}

func NewCostCalculator(costPerToken map[string]float64, budgetLimit, alertThreshold float64) *CostCalculator {
	return &CostCalculator{
		costPerToken:   costPerToken,
		budgetLimit:    budgetLimit,
		alertThreshold: alertThreshold,
		costsByModel:   make(map[string]float64),
	}
}

func (cc *CostCalculator) CalculateCost(modelName string, tokenCount int) float64 {
	if cost, exists := cc.costPerToken[modelName]; exists {
		return cost * float64(tokenCount)
	}
	return 0.0 // Unknown model, no cost
}

// GetHealthStatus returns the current health status
func (c *EnhancedPerformanceClient) GetHealthStatus() map[string]interface{} {
	c.healthCheckMutex.RLock()
	defer c.healthCheckMutex.RUnlock()

	return map[string]interface{}{
		"healthy":           c.isHealthy,
		"last_check":        c.lastHealthCheck,
		"circuit_breaker":   c.circuitBreaker.HealthCheck(),
		"active_requests":   len(c.activeRequests),
		"performance_stats": c.performanceOpt.GetLatencyProfile(),
	}
}

// GetMetrics returns comprehensive metrics
func (c *EnhancedPerformanceClient) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"performance":     c.performanceOpt.GetLatencyProfile(),
		"retry_engine":    c.retryEngine.GetMetrics(),
		"batch_processor": c.batchProcessor.GetStats(),
		"circuit_breaker": c.circuitBreaker.GetStats(),
		"token_usage":     c.getTokenUsageStats(),
		"cost_tracking":   c.getCostTrackingStats(),
	}
}

func (c *EnhancedPerformanceClient) getTokenUsageStats() map[string]interface{} {
	c.tokenTracker.mutex.RLock()
	defer c.tokenTracker.mutex.RUnlock()

	return map[string]interface{}{
		"total_tokens":     c.tokenTracker.totalTokens,
		"tokens_by_model":  c.tokenTracker.tokensByModel,
		"tokens_by_intent": c.tokenTracker.tokensByIntent,
	}
}

func (c *EnhancedPerformanceClient) getCostTrackingStats() map[string]interface{} {
	c.costCalculator.mutex.RLock()
	defer c.costCalculator.mutex.RUnlock()

	utilization := (c.costCalculator.totalCost / c.costCalculator.budgetLimit) * 100

	return map[string]interface{}{
		"total_cost":         c.costCalculator.totalCost,
		"costs_by_model":     c.costCalculator.costsByModel,
		"budget_limit":       c.costCalculator.budgetLimit,
		"budget_utilization": utilization,
		"alert_threshold":    c.costCalculator.alertThreshold,
	}
}

// Close gracefully shuts down the client
func (c *EnhancedPerformanceClient) Close() error {
	c.logger.Info("Shutting down enhanced performance client")

	if c.performanceOpt != nil {
		c.performanceOpt.Close()
	}

	if c.batchProcessor != nil {
		c.batchProcessor.Close()
	}

	c.logger.Info("Enhanced performance client shutdown complete")
	return nil
}
