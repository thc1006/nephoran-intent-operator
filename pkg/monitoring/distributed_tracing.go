package monitoring

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// DistributedTracer manages OpenTelemetry tracing for the entire system.

type DistributedTracer struct {
	tracer trace.Tracer

	provider *sdktrace.TracerProvider

	alertManager *TraceAlertManager

	spanAnalyzer *SpanAnalyzer

	config *TracingConfig

	logger *StructuredLogger

	metricsRecorder *MetricsRecorder

	mu sync.RWMutex

	activeSpans map[string]*ActiveSpan
}

// TracingConfig holds configuration for distributed tracing.

type TracingConfig struct {
	ServiceName string `json:"service_name"`

	ServiceVersion string `json:"service_version"`

	Environment string `json:"environment"`

	JaegerEndpoint string `json:"jaeger_endpoint"`

	SamplingRatio float64 `json:"sampling_ratio"`

	BatchTimeout time.Duration `json:"batch_timeout"`

	MaxBatchSize int `json:"max_batch_size"`

	MaxQueueSize int `json:"max_queue_size"`

	// Trace-based alerting configuration.

	AlertingEnabled bool `json:"alerting_enabled"`

	AlertThresholds *TraceAlertThresholds `json:"alert_thresholds"`

	AnomalyDetection *TraceAnomalyConfig `json:"anomaly_detection"`

	PerformanceTargets *PerformanceTargets `json:"performance_targets"`
}

// TraceAlertThresholds defines thresholds for trace-based alerts.

type TraceAlertThresholds struct {
	HighLatencyThreshold time.Duration `json:"high_latency_threshold"`

	CriticalLatencyThreshold time.Duration `json:"critical_latency_threshold"`

	ErrorRateThreshold float64 `json:"error_rate_threshold"`

	SpanCountThreshold int `json:"span_count_threshold"`

	DeepTraceThreshold int `json:"deep_trace_threshold"`

	SlowOperationThreshold time.Duration `json:"slow_operation_threshold"`
}

// TraceAnomalyConfig defines anomaly detection parameters.

type TraceAnomalyConfig struct {
	Enabled bool `json:"enabled"`

	WindowSize time.Duration `json:"window_size"`

	MinimumSamples int `json:"minimum_samples"`

	StandardDeviations float64 `json:"standard_deviations"`

	LatencyAnomalyEnabled bool `json:"latency_anomaly_enabled"`

	ThroughputAnomalyEnabled bool `json:"throughput_anomaly_enabled"`
}

// PerformanceTargets defines SLO targets for trace analysis.

type PerformanceTargets struct {
	P50LatencyTarget time.Duration `json:"p50_latency_target"`

	P95LatencyTarget time.Duration `json:"p95_latency_target"`

	P99LatencyTarget time.Duration `json:"p99_latency_target"`

	ErrorRateTarget float64 `json:"error_rate_target"`

	AvailabilityTarget float64 `json:"availability_target"`
}

// ActiveSpan represents an active span with metadata.

type ActiveSpan struct {
	SpanID string `json:"span_id"`

	TraceID string `json:"trace_id"`

	OperationName string `json:"operation_name"`

	StartTime time.Time `json:"start_time"`

	Tags map[string]string `json:"tags"`

	Component string `json:"component"`

	Duration time.Duration `json:"duration"`

	Status codes.Code `json:"status"`
}

// TraceAlert represents a trace-based alert.

type TraceAlert struct {
	ID string `json:"id"`

	AlertType TraceAlertType `json:"alert_type"`

	Severity AlertSeverity `json:"severity"`

	TraceID string `json:"trace_id"`

	SpanID string `json:"span_id"`

	Message string `json:"message"`

	Timestamp time.Time `json:"timestamp"`

	Metadata map[string]interface{} `json:"metadata"`

	Component string `json:"component"`

	Operation string `json:"operation"`

	Duration time.Duration `json:"duration,omitempty"`

	ErrorRate float64 `json:"error_rate,omitempty"`

	Resolved bool `json:"resolved"`

	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

// TraceAlertType defines types of trace-based alerts.

type TraceAlertType string

const (

	// AlertTypeHighLatency holds alerttypehighlatency value.

	AlertTypeHighLatency TraceAlertType = "high_latency"

	// AlertTypeCriticalLatency holds alerttypecriticallatency value.

	AlertTypeCriticalLatency TraceAlertType = "critical_latency"

	// AlertTypeHighErrorRate holds alerttypehigherrorrate value.

	AlertTypeHighErrorRate TraceAlertType = "high_error_rate"

	// AlertTypeDeepTrace holds alerttypedeeptrace value.

	AlertTypeDeepTrace TraceAlertType = "deep_trace"

	// AlertTypeSlowOperation holds alerttypeslowoperation value.

	AlertTypeSlowOperation TraceAlertType = "slow_operation"

	// AlertTypeLatencyAnomaly holds alerttypelatencyanomaly value.

	AlertTypeLatencyAnomaly TraceAlertType = "latency_anomaly"

	// AlertTypeThroughputAnomaly holds alerttypethroughputanomaly value.

	AlertTypeThroughputAnomaly TraceAlertType = "throughput_anomaly"

	// AlertTypeSpanFailure holds alerttypespanfailure value.

	AlertTypeSpanFailure TraceAlertType = "span_failure"

	// AlertTypeCircuitBreaker holds alerttypecircuitbreaker value.

	AlertTypeCircuitBreaker TraceAlertType = "circuit_breaker"
)

// AlertSeverity constants are imported from alerting.go.

// Using constants from alerting.go.

const (

	// SeverityHigh holds severityhigh value.

	SeverityHigh = AlertSeverityError

	// SeverityCritical holds severitycritical value.

	SeverityCritical = AlertSeverityCritical

	// SeverityMedium holds severitymedium value.

	SeverityMedium = AlertSeverityWarning
)

// TraceAlertManager manages trace-based alerting.

type TraceAlertManager struct {
	alerts map[string]*TraceAlert

	alertHandlers []TraceAlertHandler

	config *TracingConfig

	logger *StructuredLogger

	metricsRecorder *MetricsRecorder

	mu sync.RWMutex
}

// TraceAlertHandler interface for handling trace alerts.

type TraceAlertHandler interface {
	HandleAlert(ctx context.Context, alert *TraceAlert) error
}

// SpanAnalyzer provides advanced span analysis capabilities.

type SpanAnalyzer struct {
	config *TracingConfig

	logger *StructuredLogger

	metricsRecorder *MetricsRecorder

	mu sync.RWMutex

	spanHistory map[string][]*SpanMetrics

	latencyHistory map[string][]time.Duration
}

// SpanMetrics holds metrics for span analysis.

type SpanMetrics struct {
	SpanID string `json:"span_id"`

	TraceID string `json:"trace_id"`

	OperationName string `json:"operation_name"`

	Component string `json:"component"`

	Duration time.Duration `json:"duration"`

	StartTime time.Time `json:"start_time"`

	EndTime time.Time `json:"end_time"`

	Status codes.Code `json:"status"`

	Tags map[string]string `json:"tags"`

	ErrorMessage string `json:"error_message,omitempty"`
}

// getEnv gets environment variable with default value.

func getEnv(key, defaultValue string) string {

	if value := os.Getenv(key); value != "" {

		return value

	}

	return defaultValue

}

// DefaultTracingConfig returns default tracing configuration.

func DefaultTracingConfig() *TracingConfig {

	return &TracingConfig{

		ServiceName: "nephoran-intent-operator",

		ServiceVersion: getEnv("NEPHORAN_VERSION", "1.0.0"),

		Environment: getEnv("NEPHORAN_ENVIRONMENT", "production"),

		JaegerEndpoint: getEnv("JAEGER_ENDPOINT", "http://jaeger-collector:14268/api/traces"),

		SamplingRatio: 0.1, // 10% sampling for production

		BatchTimeout: 5 * time.Second,

		MaxBatchSize: 512,

		MaxQueueSize: 2048,

		AlertingEnabled: true,

		AlertThresholds: &TraceAlertThresholds{

			HighLatencyThreshold: 5 * time.Second,

			CriticalLatencyThreshold: 10 * time.Second,

			ErrorRateThreshold: 0.05, // 5%

			SpanCountThreshold: 100,

			DeepTraceThreshold: 20, // Max span depth

			SlowOperationThreshold: 2 * time.Second,
		},

		AnomalyDetection: &TraceAnomalyConfig{

			Enabled: true,

			WindowSize: 15 * time.Minute,

			MinimumSamples: 10,

			StandardDeviations: 2.5,

			LatencyAnomalyEnabled: true,

			ThroughputAnomalyEnabled: true,
		},

		PerformanceTargets: &PerformanceTargets{

			P50LatencyTarget: 1 * time.Second,

			P95LatencyTarget: 3 * time.Second,

			P99LatencyTarget: 5 * time.Second,

			ErrorRateTarget: 0.01, // 1%

			AvailabilityTarget: 0.999, // 99.9%

		},
	}

}

// NewDistributedTracer creates a new distributed tracer.

func NewDistributedTracer(config *TracingConfig, logger *StructuredLogger, metricsRecorder *MetricsRecorder) (*DistributedTracer, error) {

	if config == nil {

		config = DefaultTracingConfig()

	}

	// Create Jaeger exporter.

	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(config.JaegerEndpoint)))

	if err != nil {

		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)

	}

	// Create resource with service information.

	res, err := resource.Merge(

		resource.Default(),

		resource.NewWithAttributes(

			semconv.SchemaURL,

			semconv.ServiceNameKey.String(config.ServiceName),

			semconv.ServiceVersionKey.String(config.ServiceVersion),

			semconv.DeploymentEnvironmentKey.String(config.Environment),
		),
	)

	if err != nil {

		return nil, fmt.Errorf("failed to create resource: %w", err)

	}

	// Create trace provider.

	provider := sdktrace.NewTracerProvider(

		sdktrace.WithBatcher(exporter,

			sdktrace.WithBatchTimeout(config.BatchTimeout),

			sdktrace.WithMaxExportBatchSize(config.MaxBatchSize),

			sdktrace.WithMaxQueueSize(config.MaxQueueSize),
		),

		sdktrace.WithResource(res),

		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(config.SamplingRatio)),
	)

	// Set global tracer provider.

	otel.SetTracerProvider(provider)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(

		propagation.TraceContext{},

		propagation.Baggage{},
	))

	// Create tracer.

	tracer := provider.Tracer(config.ServiceName)

	// Create alert manager.

	alertManager := &TraceAlertManager{

		alerts: make(map[string]*TraceAlert),

		alertHandlers: make([]TraceAlertHandler, 0),

		config: config,

		logger: logger,

		metricsRecorder: metricsRecorder,
	}

	// Create span analyzer.

	spanAnalyzer := &SpanAnalyzer{

		config: config,

		logger: logger,

		metricsRecorder: metricsRecorder,

		spanHistory: make(map[string][]*SpanMetrics),

		latencyHistory: make(map[string][]time.Duration),
	}

	dt := &DistributedTracer{

		tracer: tracer,

		provider: provider,

		alertManager: alertManager,

		spanAnalyzer: spanAnalyzer,

		config: config,

		logger: logger,

		metricsRecorder: metricsRecorder,

		activeSpans: make(map[string]*ActiveSpan),
	}

	return dt, nil

}

// StartSpan starts a new trace span with comprehensive metadata.

func (dt *DistributedTracer) StartSpan(ctx context.Context, operationName, component string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {

	// Add component and operation name as attributes.

	attributes := []attribute.KeyValue{

		attribute.String("component", component),

		attribute.String("operation", operationName),

		attribute.String("service", dt.config.ServiceName),

		attribute.String("environment", dt.config.Environment),
	}

	// Add correlation ID if available.

	if correlationID := GetCorrelationID(ctx); correlationID != "" {

		attributes = append(attributes, attribute.String("correlation_id", correlationID))

	}

	// Create span start options.

	spanOpts := append(opts, trace.WithAttributes(attributes...))

	spanCtx, span := dt.tracer.Start(ctx, operationName, spanOpts...)

	// Record active span.

	spanContext := span.SpanContext()

	activeSpan := &ActiveSpan{

		SpanID: spanContext.SpanID().String(),

		TraceID: spanContext.TraceID().String(),

		OperationName: operationName,

		StartTime: time.Now(),

		Tags: make(map[string]string),

		Component: component,

		Status: codes.Unset,
	}

	dt.mu.Lock()

	dt.activeSpans[spanContext.SpanID().String()] = activeSpan

	dt.mu.Unlock()

	// Log span start.

	if dt.logger != nil {

		dt.logger.Debug(spanCtx, "Trace span started",

			slog.String("trace_id", activeSpan.TraceID),

			slog.String("span_id", activeSpan.SpanID),

			slog.String("operation", operationName),

			slog.String("component", component),
		)

	}

	return spanCtx, span

}

// FinishSpan finishes a span and performs analysis.

func (dt *DistributedTracer) FinishSpan(span trace.Span, status codes.Code, err error) {

	spanContext := span.SpanContext()

	spanID := spanContext.SpanID().String()

	// Get active span.

	dt.mu.Lock()

	activeSpan, exists := dt.activeSpans[spanID]

	if exists {

		activeSpan.Duration = time.Since(activeSpan.StartTime)

		activeSpan.Status = status

		delete(dt.activeSpans, spanID)

	}

	dt.mu.Unlock()

	// Set span status.

	if err != nil {

		span.SetStatus(codes.Error, err.Error())

		span.RecordError(err)

	} else {

		span.SetStatus(status, "")

	}

	// End the span.

	span.End()

	// Perform span analysis if span exists.

	if exists {

		dt.analyzeSpan(activeSpan, err)

	}

	// Log span completion.

	if dt.logger != nil && activeSpan != nil {

		level := slog.LevelInfo

		if status == codes.Error {

			level = slog.LevelError

		}

		attrs := []slog.Attr{

			slog.String("trace_id", activeSpan.TraceID),

			slog.String("span_id", activeSpan.SpanID),

			slog.String("operation", activeSpan.OperationName),

			slog.String("component", activeSpan.Component),

			slog.Duration("duration", activeSpan.Duration),

			slog.String("status", status.String()),
		}

		if err != nil {

			attrs = append(attrs, slog.String("error", err.Error()))

		}

		dt.logger.logger.LogAttrs(context.Background(), level, "Trace span completed", attrs...)

	}

}

// analyzeSpan performs comprehensive span analysis for alerting.

func (dt *DistributedTracer) analyzeSpan(activeSpan *ActiveSpan, err error) {

	if !dt.config.AlertingEnabled {

		return

	}

	spanMetrics := &SpanMetrics{

		SpanID: activeSpan.SpanID,

		TraceID: activeSpan.TraceID,

		OperationName: activeSpan.OperationName,

		Component: activeSpan.Component,

		Duration: activeSpan.Duration,

		StartTime: activeSpan.StartTime,

		EndTime: activeSpan.StartTime.Add(activeSpan.Duration),

		Status: activeSpan.Status,

		Tags: activeSpan.Tags,
	}

	if err != nil {

		spanMetrics.ErrorMessage = err.Error()

	}

	// Store span metrics for analysis.

	dt.spanAnalyzer.RecordSpan(spanMetrics)

	// Check for immediate alerts.

	dt.checkSpanAlerts(spanMetrics)

	// Record metrics.

	if dt.metricsRecorder != nil {

		// Record span duration via SLI latency metrics.

		dt.metricsRecorder.RecordSLILatency(spanMetrics.Component, spanMetrics.OperationName, "default",

			spanMetrics.Duration.Seconds(), spanMetrics.Duration.Seconds())

		if spanMetrics.Status == codes.Error {

			dt.metricsRecorder.RecordError(spanMetrics.Component, "span_error")

		}

	}

}

// checkSpanAlerts checks for various alert conditions.

func (dt *DistributedTracer) checkSpanAlerts(spanMetrics *SpanMetrics) {

	thresholds := dt.config.AlertThresholds

	// High latency alert.

	if spanMetrics.Duration > thresholds.HighLatencyThreshold {

		severity := SeverityHigh

		if spanMetrics.Duration > thresholds.CriticalLatencyThreshold {

			severity = SeverityCritical

		}

		alert := &TraceAlert{

			ID: fmt.Sprintf("latency-%s-%d", spanMetrics.SpanID, time.Now().Unix()),

			AlertType: AlertTypeHighLatency,

			Severity: severity,

			TraceID: spanMetrics.TraceID,

			SpanID: spanMetrics.SpanID,

			Message: fmt.Sprintf("High latency detected: %v (threshold: %v)", spanMetrics.Duration, thresholds.HighLatencyThreshold),

			Timestamp: time.Now(),

			Component: spanMetrics.Component,

			Operation: spanMetrics.OperationName,

			Duration: spanMetrics.Duration,

			Metadata: map[string]interface{}{

				"threshold_exceeded": spanMetrics.Duration > thresholds.HighLatencyThreshold,

				"critical_threshold": spanMetrics.Duration > thresholds.CriticalLatencyThreshold,

				"start_time": spanMetrics.StartTime,

				"end_time": spanMetrics.EndTime,
			},
		}

		dt.alertManager.FireAlert(context.Background(), alert)

	}

	// Slow operation alert.

	if spanMetrics.Duration > thresholds.SlowOperationThreshold {

		alert := &TraceAlert{

			ID: fmt.Sprintf("slow-op-%s-%d", spanMetrics.SpanID, time.Now().Unix()),

			AlertType: AlertTypeSlowOperation,

			Severity: SeverityMedium,

			TraceID: spanMetrics.TraceID,

			SpanID: spanMetrics.SpanID,

			Message: fmt.Sprintf("Slow operation detected: %s took %v", spanMetrics.OperationName, spanMetrics.Duration),

			Timestamp: time.Now(),

			Component: spanMetrics.Component,

			Operation: spanMetrics.OperationName,

			Duration: spanMetrics.Duration,

			Metadata: map[string]interface{}{

				"operation_type": spanMetrics.OperationName,

				"expected_max": thresholds.SlowOperationThreshold,

				"actual_duration": spanMetrics.Duration,
			},
		}

		dt.alertManager.FireAlert(context.Background(), alert)

	}

	// Span failure alert.

	if spanMetrics.Status == codes.Error {

		alert := &TraceAlert{

			ID: fmt.Sprintf("span-error-%s-%d", spanMetrics.SpanID, time.Now().Unix()),

			AlertType: AlertTypeSpanFailure,

			Severity: SeverityHigh,

			TraceID: spanMetrics.TraceID,

			SpanID: spanMetrics.SpanID,

			Message: fmt.Sprintf("Span failure detected: %s", spanMetrics.ErrorMessage),

			Timestamp: time.Now(),

			Component: spanMetrics.Component,

			Operation: spanMetrics.OperationName,

			Duration: spanMetrics.Duration,

			Metadata: map[string]interface{}{

				"error_message": spanMetrics.ErrorMessage,

				"operation": spanMetrics.OperationName,
			},
		}

		dt.alertManager.FireAlert(context.Background(), alert)

	}

}

// RecordSpan records span metrics for analysis.

func (sa *SpanAnalyzer) RecordSpan(spanMetrics *SpanMetrics) {

	sa.mu.Lock()

	defer sa.mu.Unlock()

	operationKey := fmt.Sprintf("%s.%s", spanMetrics.Component, spanMetrics.OperationName)

	// Store span history.

	if sa.spanHistory[operationKey] == nil {

		sa.spanHistory[operationKey] = make([]*SpanMetrics, 0)

	}

	sa.spanHistory[operationKey] = append(sa.spanHistory[operationKey], spanMetrics)

	// Store latency history.

	if sa.latencyHistory[operationKey] == nil {

		sa.latencyHistory[operationKey] = make([]time.Duration, 0)

	}

	sa.latencyHistory[operationKey] = append(sa.latencyHistory[operationKey], spanMetrics.Duration)

	// Limit history size (keep last 1000 entries).

	if len(sa.spanHistory[operationKey]) > 1000 {

		sa.spanHistory[operationKey] = sa.spanHistory[operationKey][len(sa.spanHistory[operationKey])-1000:]

	}

	if len(sa.latencyHistory[operationKey]) > 1000 {

		sa.latencyHistory[operationKey] = sa.latencyHistory[operationKey][len(sa.latencyHistory[operationKey])-1000:]

	}

	// Check for anomalies if enabled.

	if sa.config.AnomalyDetection.Enabled {

		sa.checkAnomalies(operationKey, spanMetrics)

	}

}

// checkAnomalies performs anomaly detection on span metrics.

func (sa *SpanAnalyzer) checkAnomalies(operationKey string, spanMetrics *SpanMetrics) {

	config := sa.config.AnomalyDetection

	// Only check if we have minimum samples.

	latencies := sa.latencyHistory[operationKey]

	if len(latencies) < config.MinimumSamples {

		return

	}

	// Calculate statistics for latency anomaly detection.

	if config.LatencyAnomalyEnabled {

		mean, stdDev := calculateStatistics(latencies)

		threshold := mean + time.Duration(float64(stdDev)*config.StandardDeviations)

		if spanMetrics.Duration > threshold {

			// This is a latency anomaly.

			if sa.logger != nil {

				sa.logger.Warn(context.Background(), "Latency anomaly detected",

					slog.String("operation", operationKey),

					slog.Duration("duration", spanMetrics.Duration),

					slog.Duration("mean", mean),

					slog.Duration("threshold", threshold),

					slog.Float64("std_dev_multiplier", config.StandardDeviations),
				)

			}

			// Record anomaly metric.

			if sa.metricsRecorder != nil {

				sa.metricsRecorder.RecordAnomalyDetection("latency", "high", operationKey, "statistical")

			}

		}

	}

}

// FireAlert fires a trace-based alert.

func (tam *TraceAlertManager) FireAlert(ctx context.Context, alert *TraceAlert) {

	tam.mu.Lock()

	defer tam.mu.Unlock()

	// Store alert.

	tam.alerts[alert.ID] = alert

	// Log alert.

	if tam.logger != nil {

		tam.logger.Warn(ctx, "Trace alert fired",

			slog.String("alert_id", alert.ID),

			slog.String("alert_type", string(alert.AlertType)),

			slog.String("severity", string(alert.Severity)),

			slog.String("trace_id", alert.TraceID),

			slog.String("component", alert.Component),

			slog.String("message", alert.Message),
		)

	}

	// Record alert metric.

	if tam.metricsRecorder != nil {

		// Record as an alert occurrence.

		tam.metricsRecorder.RecordAlert(string(alert.Severity), alert.Component)

	}

	// Call alert handlers.

	for _, handler := range tam.alertHandlers {

		go func(h TraceAlertHandler) {

			if err := h.HandleAlert(ctx, alert); err != nil && tam.logger != nil {

				tam.logger.Error(ctx, "Failed to handle trace alert", err,

					slog.String("alert_id", alert.ID),

					slog.String("handler_type", fmt.Sprintf("%T", h)),
				)

			}

		}(handler)

	}

}

// AddAlertHandler adds a trace alert handler.

func (tam *TraceAlertManager) AddAlertHandler(handler TraceAlertHandler) {

	tam.mu.Lock()

	defer tam.mu.Unlock()

	tam.alertHandlers = append(tam.alertHandlers, handler)

}

// GetActiveAlerts returns all active (unresolved) alerts.

func (tam *TraceAlertManager) GetActiveAlerts() []*TraceAlert {

	tam.mu.RLock()

	defer tam.mu.RUnlock()

	activeAlerts := make([]*TraceAlert, 0)

	for _, alert := range tam.alerts {

		if !alert.Resolved {

			activeAlerts = append(activeAlerts, alert)

		}

	}

	return activeAlerts

}

// ResolveAlert resolves an alert by ID.

func (tam *TraceAlertManager) ResolveAlert(ctx context.Context, alertID string) error {

	tam.mu.Lock()

	defer tam.mu.Unlock()

	alert, exists := tam.alerts[alertID]

	if !exists {

		return fmt.Errorf("alert not found: %s", alertID)

	}

	now := time.Now()

	alert.Resolved = true

	alert.ResolvedAt = &now

	if tam.logger != nil {

		tam.logger.Info(ctx, "Trace alert resolved",

			slog.String("alert_id", alertID),

			slog.String("alert_type", string(alert.AlertType)),

			slog.Time("resolved_at", now),
		)

	}

	return nil

}

// Shutdown gracefully shuts down the distributed tracer.

func (dt *DistributedTracer) Shutdown(ctx context.Context) error {

	if dt.provider != nil {

		return dt.provider.Shutdown(ctx)

	}

	return nil

}

// Helper functions.

// calculateStatistics calculates mean and standard deviation.

func calculateStatistics(durations []time.Duration) (mean, stdDev time.Duration) {

	if len(durations) == 0 {

		return 0, 0

	}

	// Calculate mean.

	var sum time.Duration

	for _, d := range durations {

		sum += d

	}

	mean = sum / time.Duration(len(durations))

	// Calculate standard deviation.

	var variance float64

	for _, d := range durations {

		diff := float64(d - mean)

		variance += diff * diff

	}

	variance /= float64(len(durations))

	stdDev = time.Duration(math.Sqrt(variance))

	return mean, stdDev

}

// TraceMiddleware provides HTTP middleware for tracing.

func (dt *DistributedTracer) TraceMiddleware(component string) func(next interface{}) interface{} {

	return func(next interface{}) interface{} {

		// This would be implemented based on your HTTP framework.

		// For now, returning a placeholder.

		return next

	}

}

// GetActiveSpans returns currently active spans.

func (dt *DistributedTracer) GetActiveSpans() map[string]*ActiveSpan {

	dt.mu.RLock()

	defer dt.mu.RUnlock()

	spans := make(map[string]*ActiveSpan)

	for id, span := range dt.activeSpans {

		spans[id] = span

	}

	return spans

}

// GetSpanAnalytics returns analytics for a specific operation.

func (sa *SpanAnalyzer) GetSpanAnalytics(component, operation string) (*SpanAnalytics, error) {

	sa.mu.RLock()

	defer sa.mu.RUnlock()

	operationKey := fmt.Sprintf("%s.%s", component, operation)

	spans, exists := sa.spanHistory[operationKey]

	if !exists || len(spans) == 0 {

		return nil, fmt.Errorf("no data found for operation: %s", operationKey)

	}

	latencies := sa.latencyHistory[operationKey]

	mean, stdDev := calculateStatistics(latencies)

	// Calculate percentiles.

	p50, p95, p99 := calculatePercentiles(latencies)

	// Calculate error rate.

	errorCount := 0

	for _, span := range spans {

		if span.Status == codes.Error {

			errorCount++

		}

	}

	errorRate := float64(errorCount) / float64(len(spans))

	analytics := &SpanAnalytics{

		Component: component,

		Operation: operation,

		TotalSpans: len(spans),

		ErrorCount: errorCount,

		ErrorRate: errorRate,

		MeanLatency: mean,

		StdDevLatency: stdDev,

		P50Latency: p50,

		P95Latency: p95,

		P99Latency: p99,

		LastUpdated: time.Now(),
	}

	return analytics, nil

}

// SpanAnalytics holds analytics data for spans.

type SpanAnalytics struct {
	Component string `json:"component"`

	Operation string `json:"operation"`

	TotalSpans int `json:"total_spans"`

	ErrorCount int `json:"error_count"`

	ErrorRate float64 `json:"error_rate"`

	MeanLatency time.Duration `json:"mean_latency"`

	StdDevLatency time.Duration `json:"std_dev_latency"`

	P50Latency time.Duration `json:"p50_latency"`

	P95Latency time.Duration `json:"p95_latency"`

	P99Latency time.Duration `json:"p99_latency"`

	LastUpdated time.Time `json:"last_updated"`
}

// calculatePercentiles calculates P50, P95, and P99 percentiles.

func calculatePercentiles(durations []time.Duration) (p50, p95, p99 time.Duration) {

	if len(durations) == 0 {

		return 0, 0, 0

	}

	// Sort durations.

	sorted := make([]time.Duration, len(durations))

	copy(sorted, durations)

	// Simple sort implementation.

	for i := 0; i < len(sorted); i++ {

		for j := i + 1; j < len(sorted); j++ {

			if sorted[i] > sorted[j] {

				sorted[i], sorted[j] = sorted[j], sorted[i]

			}

		}

	}

	// Calculate percentiles.

	p50Index := int(float64(len(sorted)) * 0.50)

	p95Index := int(float64(len(sorted)) * 0.95)

	p99Index := int(float64(len(sorted)) * 0.99)

	if p50Index >= len(sorted) {

		p50Index = len(sorted) - 1

	}

	if p95Index >= len(sorted) {

		p95Index = len(sorted) - 1

	}

	if p99Index >= len(sorted) {

		p99Index = len(sorted) - 1

	}

	return sorted[p50Index], sorted[p95Index], sorted[p99Index]

}

// DefaultSlackAlertHandler provides Slack integration for trace alerts.

type DefaultSlackAlertHandler struct {
	WebhookURL string

	Channel string

	logger *StructuredLogger
}

// NewDefaultSlackAlertHandler creates a new Slack alert handler.

func NewDefaultSlackAlertHandler(webhookURL, channel string, logger *StructuredLogger) *DefaultSlackAlertHandler {

	return &DefaultSlackAlertHandler{

		WebhookURL: webhookURL,

		Channel: channel,

		logger: logger,
	}

}

// HandleAlert handles trace alerts by sending to Slack.

func (h *DefaultSlackAlertHandler) HandleAlert(ctx context.Context, alert *TraceAlert) error {

	// Create Slack message.

	message := fmt.Sprintf("🚨 *Trace Alert: %s*\n", alert.AlertType)

	message += fmt.Sprintf("*Severity:* %s\n", alert.Severity)

	message += fmt.Sprintf("*Component:* %s\n", alert.Component)

	message += fmt.Sprintf("*Operation:* %s\n", alert.Operation)

	message += fmt.Sprintf("*Message:* %s\n", alert.Message)

	message += fmt.Sprintf("*Trace ID:* `%s`\n", alert.TraceID)

	if alert.Duration > 0 {

		message += fmt.Sprintf("*Duration:* %v\n", alert.Duration)

	}

	message += fmt.Sprintf("*Time:* %s", alert.Timestamp.Format(time.RFC3339))

	// This would integrate with your Slack webhook implementation.

	if h.logger != nil {

		h.logger.Info(ctx, "Sending trace alert to Slack",

			slog.String("alert_id", alert.ID),

			slog.String("channel", h.Channel),
			slog.String("message", message),
		)

	}

	return nil

}
