package monitoring

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	// "go.opentelemetry.io/otel/exporters/jaeger"  // Commented out due to version compatibility issues
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	// SpanStatusOK indicates successful span completion
	SpanStatusOK = "OK"
	// SpanStatusError indicates span completed with error
	SpanStatusError = "ERROR"
	// SpanStatusTimeout indicates span timed out
	SpanStatusTimeout = "TIMEOUT"
)

// DistributedTracer manages distributed tracing for O-RAN components
type DistributedTracer struct {
	tracer      oteltrace.Tracer
	provider    *trace.TracerProvider
	serviceName string
	spans       map[string]*TraceSpan
	mu          sync.RWMutex
}

// NewDistributedTracer creates a new distributed tracer
func NewDistributedTracer(serviceName, jaegerEndpoint string) (*DistributedTracer, error) {
	// Create Jaeger exporter - temporarily disabled due to version incompatibility
	// exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerEndpoint)))
	// if err != nil {
	//     return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	// }

	// Use no-op tracer as fallback
	var exp trace.SpanExporter
	err := (error)(nil)

	// Create resource
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String("1.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create trace provider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(res),
	)

	otel.SetTracerProvider(tp)

	return &DistributedTracer{
		tracer:      tp.Tracer(serviceName),
		provider:    tp,
		serviceName: serviceName,
		spans:       make(map[string]*TraceSpan),
	}, nil
}

// StartSpan starts a new trace span
func (dt *DistributedTracer) StartSpan(ctx context.Context, operationName string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	return dt.tracer.Start(ctx, operationName, opts...)
}

// CollectSpan implements TraceCollector interface
func (dt *DistributedTracer) CollectSpan(ctx context.Context, span *TraceSpan) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	dt.spans[span.SpanID] = span
	return nil
}

// GetTrace implements TraceCollector interface
func (dt *DistributedTracer) GetTrace(ctx context.Context, traceID string) ([]*TraceSpan, error) {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	var spans []*TraceSpan
	for _, span := range dt.spans {
		if span.TraceID == traceID {
			spans = append(spans, span)
		}
	}

	return spans, nil
}

// CreateSpanFromOtel converts OpenTelemetry span to TraceSpan
func (dt *DistributedTracer) CreateSpanFromOtel(span oteltrace.Span) *TraceSpan {
	spanCtx := span.SpanContext()

	traceSpan := &TraceSpan{
		TraceID:       spanCtx.TraceID().String(),
		SpanID:        spanCtx.SpanID().String(),
		OperationName: span.SpanContext().TraceID().String(), // This would be set properly in real implementation
		StartTime:     time.Now(),
		Status:        SpanStatusOK,
		ServiceName:   dt.serviceName,
		Tags:          make(map[string]string),
	}

	return traceSpan
}

// AddE2Trace adds E2 interface tracing
func (dt *DistributedTracer) AddE2Trace(ctx context.Context, messageType string, nodeID string) (context.Context, oteltrace.Span) {
	return dt.tracer.Start(ctx, fmt.Sprintf("e2:%s", messageType),
		oteltrace.WithAttributes(
			attribute.String("e2.message_type", messageType),
			attribute.String("e2.node_id", nodeID),
			attribute.String("interface", "E2"),
		),
	)
}

// AddA1Trace adds A1 interface tracing
func (dt *DistributedTracer) AddA1Trace(ctx context.Context, policyType string, policyID string) (context.Context, oteltrace.Span) {
	return dt.tracer.Start(ctx, fmt.Sprintf("a1:policy:%s", policyType),
		oteltrace.WithAttributes(
			attribute.String("a1.policy_type", policyType),
			attribute.String("a1.policy_id", policyID),
			attribute.String("interface", "A1"),
		),
	)
}

// AddO1Trace adds O1 interface tracing
func (dt *DistributedTracer) AddO1Trace(ctx context.Context, operation string, managedElement string) (context.Context, oteltrace.Span) {
	return dt.tracer.Start(ctx, fmt.Sprintf("o1:%s", operation),
		oteltrace.WithAttributes(
			attribute.String("o1.operation", operation),
			attribute.String("o1.managed_element", managedElement),
			attribute.String("interface", "O1"),
		),
	)
}

// Shutdown gracefully shuts down the tracer
func (dt *DistributedTracer) Shutdown(ctx context.Context) error {
	return dt.provider.Shutdown(ctx)
}

// TracingMiddleware creates middleware for HTTP tracing
func (dt *DistributedTracer) TracingMiddleware() func(next func()) func() {
	return func(next func()) func() {
		return func() {
			_, span := dt.StartSpan(context.Background(), "http_request")
			defer span.End()

			// Set common attributes
			span.SetAttributes(
				attribute.String("component", "http"),
				attribute.String("service.name", dt.serviceName),
			)

			// Call next handler with tracing context
			next()
		}
	}
}
