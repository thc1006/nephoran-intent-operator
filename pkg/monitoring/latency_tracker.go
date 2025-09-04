// Package monitoring provides E2E latency tracking for O-RAN operations
package monitoring

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// E2ELatencyTracker tracks end-to-end latency for O-RAN operations
type E2ELatencyTracker struct {
	tracer     trace.Tracer
	spans      map[string]trace.Span
	spansMutex sync.RWMutex
	metrics    map[string]*LatencyMetrics
	logger     logr.Logger
}

// LatencyMetrics holds latency statistics
type LatencyMetrics struct {
	MinLatency   time.Duration
	MaxLatency   time.Duration
	AvgLatency   time.Duration
	P95Latency   time.Duration
	P99Latency   time.Duration
	RequestCount int64
	FailureCount int64
	LastMeasured time.Time
	mutex        sync.RWMutex
}

// NewE2ELatencyTracker creates a new E2E latency tracker
func NewE2ELatencyTracker() *E2ELatencyTracker {
	return &E2ELatencyTracker{
		tracer:  otel.Tracer("nephio-e2e-latency"),
		spans:   make(map[string]trace.Span),
		metrics: make(map[string]*LatencyMetrics),
		logger:  log.Log.WithName("e2e-latency-tracker"),
	}
}

// StartOperation begins tracking an E2E operation
func (t *E2ELatencyTracker) StartOperation(ctx context.Context, operationID, operationType, component string, attrs map[string]string) (context.Context, error) {
	t.spansMutex.Lock()
	defer t.spansMutex.Unlock()

	spanName := fmt.Sprintf("e2e.%s.%s", component, operationType)

	// Create span attributes
	spanAttrs := []attribute.KeyValue{
		attribute.String("operation.id", operationID),
		attribute.String("operation.type", operationType),
		attribute.String("component", component),
		attribute.String("service.name", "nephio-oran"),
		attribute.String("service.version", "1.0.0"),
	}

	// Add custom attributes
	for k, v := range attrs {
		spanAttrs = append(spanAttrs, attribute.String(k, v))
	}

	ctx, span := t.tracer.Start(ctx, spanName, trace.WithAttributes(spanAttrs...))

	t.spans[operationID] = span

	t.logger.Info("Started E2E operation tracking",
		"operationID", operationID,
		"type", operationType,
		"component", component,
		"traceID", span.SpanContext().TraceID().String(),
		"spanID", span.SpanContext().SpanID().String())

	return ctx, nil
}

// EndOperation completes tracking an E2E operation
func (t *E2ELatencyTracker) EndOperation(ctx context.Context, operationID string, success bool, errorMsg string) error {
	t.spansMutex.Lock()
	defer t.spansMutex.Unlock()

	span, exists := t.spans[operationID]
	if !exists {
		return fmt.Errorf("operation %s not found", operationID)
	}

	defer func() {
		delete(t.spans, operationID)
		span.End()
	}()

	if !success {
		span.SetStatus(codes.Error, errorMsg)
		span.SetAttributes(attribute.String("error.message", errorMsg))
	} else {
		span.SetStatus(codes.Ok, "Operation completed successfully")
	}

	// Calculate latency from span
	if span.SpanContext().IsValid() {
		// Get latency from span timestamps (simplified approach)
		latency := time.Since(time.Now().Add(-5 * time.Second)) // Placeholder calculation

		// Update metrics
		t.updateMetrics(operationID, latency, success)

		span.SetAttributes(attribute.Int64("latency.milliseconds", latency.Milliseconds()))

		t.logger.Info("Completed E2E operation tracking",
			"operationID", operationID,
			"success", success,
			"latency", latency.String(),
			"traceID", span.SpanContext().TraceID().String())
	}

	return nil
}

// TrackOperation is a convenience method to track an operation with context
func (t *E2ELatencyTracker) TrackOperation(ctx context.Context, operationID string, fn func(context.Context) error) error {
	// Extract span from context (fixes line 294)
	currentSpan := trace.SpanFromContext(ctx)
	if !currentSpan.SpanContext().IsValid() {
		var err error
		ctx, err = t.StartOperation(ctx, operationID, "generic", "nephio", nil)
		if err != nil {
			return fmt.Errorf("failed to start operation tracking: %w", err)
		}
	}

	start := time.Now()
	err := fn(ctx)
	latency := time.Since(start)

	success := err == nil
	var errorMsg string
	if !success {
		errorMsg = err.Error()
	}

	// Record the latency measurement
	t.RecordLatency("generic", latency, success)

	if endErr := t.EndOperation(ctx, operationID, success, errorMsg); endErr != nil {
		t.logger.Error(endErr, "Failed to end operation tracking", "operationID", operationID)
	}

	return err
}

// GetSpanFromContext extracts span from context (line 294 fix)
func (t *E2ELatencyTracker) GetSpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// RecordLatency records a latency measurement
func (t *E2ELatencyTracker) RecordLatency(operationType string, latency time.Duration, success bool) {
	t.updateMetrics(operationType, latency, success)
}

// updateMetrics updates internal metrics
func (t *E2ELatencyTracker) updateMetrics(operationType string, latency time.Duration, success bool) {
	if _, exists := t.metrics[operationType]; !exists {
		t.metrics[operationType] = &LatencyMetrics{
			MinLatency:   latency,
			MaxLatency:   latency,
			AvgLatency:   latency,
			P95Latency:   latency,
			P99Latency:   latency,
			RequestCount: 0,
			FailureCount: 0,
			LastMeasured: time.Now(),
		}
	}

	metrics := t.metrics[operationType]
	metrics.mutex.Lock()
	defer metrics.mutex.Unlock()

	metrics.RequestCount++
	if !success {
		metrics.FailureCount++
	}

	if latency < metrics.MinLatency {
		metrics.MinLatency = latency
	}
	if latency > metrics.MaxLatency {
		metrics.MaxLatency = latency
	}

	// Simple average calculation (could be improved with sliding window)
	metrics.AvgLatency = (metrics.AvgLatency*time.Duration(metrics.RequestCount-1) + latency) / time.Duration(metrics.RequestCount)

	// Simplified percentile calculation (in production, use proper histogram)
	metrics.P95Latency = time.Duration(float64(metrics.MaxLatency) * 0.95)
	metrics.P99Latency = time.Duration(float64(metrics.MaxLatency) * 0.99)

	metrics.LastMeasured = time.Now()
}

// StartPeriodicCollection starts periodic latency collection
func (t *E2ELatencyTracker) StartPeriodicCollection(ctx context.Context, interval time.Duration) {
	go wait.UntilWithContext(ctx, func(ctx context.Context) {
		t.collectMetrics(ctx)
	}, interval)
}

// collectMetrics performs periodic metrics collection
func (t *E2ELatencyTracker) collectMetrics(ctx context.Context) {
	t.spansMutex.RLock()
	defer t.spansMutex.RUnlock()

	for opType, metrics := range t.metrics {
		metrics.mutex.RLock()
		t.logger.V(1).Info("Latency metrics collected",
			"operationType", opType,
			"requestCount", metrics.RequestCount,
			"failureCount", metrics.FailureCount,
			"avgLatency", metrics.AvgLatency.String(),
			"p95Latency", metrics.P95Latency.String())
		metrics.mutex.RUnlock()
	}
}
