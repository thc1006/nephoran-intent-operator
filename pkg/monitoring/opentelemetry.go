
package monitoring



import (

	"context"

	"fmt"

	"time"



	"go.opentelemetry.io/otel"

	"go.opentelemetry.io/otel/attribute"

	"go.opentelemetry.io/otel/codes"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"

	"go.opentelemetry.io/otel/exporters/prometheus"

	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/otel/propagation"

	"go.opentelemetry.io/otel/sdk/resource"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"

	"go.opentelemetry.io/otel/trace"



	"github.com/nephio-project/nephoran-intent-operator/pkg/config"

)



const (

	// ServiceName holds servicename value.

	ServiceName = "nephoran-intent-operator"

	// ServiceVersion holds serviceversion value.

	ServiceVersion = "1.0.0"

)



// OpenTelemetryConfig holds OpenTelemetry configuration.

type OpenTelemetryConfig struct {

	ServiceName    string

	ServiceVersion string

	Environment    string

	JaegerEndpoint string

	SamplingRate   float64

	EnableMetrics  bool

	EnableTracing  bool

	EnableLogging  bool

}



// OpenTelemetryProvider manages OpenTelemetry setup.

type OpenTelemetryProvider struct {

	config         *OpenTelemetryConfig

	tracerProvider *sdktrace.TracerProvider

	tracer         trace.Tracer

	meter          metric.Meter

	shutdownFuncs  []func(context.Context) error

}



// NewOpenTelemetryProvider creates a new OpenTelemetry provider.

func NewOpenTelemetryProvider(config *OpenTelemetryConfig) (*OpenTelemetryProvider, error) {

	if config == nil {

		config = DefaultOpenTelemetryConfig()

	}



	otp := &OpenTelemetryProvider{

		config: config,

	}



	var err error



	// Initialize resource.

	res, err := otp.initResource()

	if err != nil {

		return nil, fmt.Errorf("failed to initialize resource: %w", err)

	}



	// Initialize tracing.

	if config.EnableTracing {

		err = otp.initTracing(res)

		if err != nil {

			return nil, fmt.Errorf("failed to initialize tracing: %w", err)

		}

	}



	// Initialize metrics.

	if config.EnableMetrics {

		err = otp.initMetrics(res)

		if err != nil {

			return nil, fmt.Errorf("failed to initialize metrics: %w", err)

		}

	}



	// Set global propagator.

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(

		propagation.TraceContext{},

		propagation.Baggage{},

	))



	return otp, nil

}



// DefaultOpenTelemetryConfig returns default OpenTelemetry configuration.

func DefaultOpenTelemetryConfig() *OpenTelemetryConfig {

	return &OpenTelemetryConfig{

		ServiceName:    ServiceName,

		ServiceVersion: ServiceVersion,

		Environment:    config.GetEnvOrDefault("NEPHORAN_ENVIRONMENT", "production"),

		JaegerEndpoint: config.GetEnvOrDefault("JAEGER_ENDPOINT", "http://jaeger-collector:14268/api/traces"),

		SamplingRate:   0.1, // 10% sampling rate

		EnableMetrics:  true,

		EnableTracing:  true,

		EnableLogging:  true,

	}

}



// initResource initializes OpenTelemetry resource.

func (otp *OpenTelemetryProvider) initResource() (*resource.Resource, error) {

	return resource.Merge(

		resource.Default(),

		resource.NewWithAttributes(

			semconv.SchemaURL,

			semconv.ServiceName(otp.config.ServiceName),

			semconv.ServiceVersion(otp.config.ServiceVersion),

			semconv.DeploymentEnvironment(otp.config.Environment),

			attribute.String("service.type", "kubernetes-operator"),

			attribute.String("service.component", "nephoran-intent-operator"),

		),

	)

}



// initTracing initializes OpenTelemetry tracing.

func (otp *OpenTelemetryProvider) initTracing(res *resource.Resource) error {

	// Create OTLP exporter.

	exporter, err := otlptrace.New(

		context.Background(),

		otlptracehttp.NewClient(

			otlptracehttp.WithEndpoint(otp.config.JaegerEndpoint),

			otlptracehttp.WithInsecure(),

		),

	)

	if err != nil {

		return fmt.Errorf("failed to create OTLP exporter: %w", err)

	}



	// Create tracer provider.

	otp.tracerProvider = sdktrace.NewTracerProvider(

		sdktrace.WithBatcher(exporter),

		sdktrace.WithResource(res),

		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(otp.config.SamplingRate)),

	)



	// Set global tracer provider.

	otel.SetTracerProvider(otp.tracerProvider)



	// Get tracer.

	otp.tracer = otp.tracerProvider.Tracer(otp.config.ServiceName)



	// Add shutdown function.

	otp.shutdownFuncs = append(otp.shutdownFuncs, otp.tracerProvider.Shutdown)



	return nil

}



// initMetrics initializes OpenTelemetry metrics.

func (otp *OpenTelemetryProvider) initMetrics(res *resource.Resource) error {

	// Create Prometheus exporter.

	_, err := prometheus.New()

	if err != nil {

		return fmt.Errorf("failed to create Prometheus exporter: %w", err)

	}



	// Get meter.

	otp.meter = otel.Meter(otp.config.ServiceName)



	// Add shutdown function (Prometheus exporter doesn't need explicit shutdown).

	otp.shutdownFuncs = append(otp.shutdownFuncs, func(ctx context.Context) error {

		return nil

	})



	return nil

}



// GetTracer returns the tracer instance.

func (otp *OpenTelemetryProvider) GetTracer() trace.Tracer {

	return otp.tracer

}



// GetMeter returns the meter instance.

func (otp *OpenTelemetryProvider) GetMeter() metric.Meter {

	return otp.meter

}



// Shutdown gracefully shuts down the OpenTelemetry provider.

func (otp *OpenTelemetryProvider) Shutdown(ctx context.Context) error {

	var errors []error



	for _, shutdown := range otp.shutdownFuncs {

		if err := shutdown(ctx); err != nil {

			errors = append(errors, err)

		}

	}



	if len(errors) > 0 {

		return fmt.Errorf("shutdown errors: %v", errors)

	}



	return nil

}



// NetworkIntentTracer provides tracing for NetworkIntent operations.

type NetworkIntentTracer struct {

	tracer trace.Tracer

}



// NewNetworkIntentTracer creates a new NetworkIntent tracer.

func NewNetworkIntentTracer(tracer trace.Tracer) *NetworkIntentTracer {

	return &NetworkIntentTracer{tracer: tracer}

}



// StartReconciliation starts a reconciliation span.

func (nt *NetworkIntentTracer) StartReconciliation(ctx context.Context, name, namespace string) (context.Context, trace.Span) {

	return nt.tracer.Start(ctx, "networkintent.reconciliation",

		trace.WithAttributes(

			attribute.String("networkintent.name", name),

			attribute.String("networkintent.namespace", namespace),

			attribute.String("operation.type", "reconciliation"),

		),

	)

}



// StartLLMProcessing starts an LLM processing span.

func (nt *NetworkIntentTracer) StartLLMProcessing(ctx context.Context, model, intentType string) (context.Context, trace.Span) {

	return nt.tracer.Start(ctx, "networkintent.llm_processing",

		trace.WithAttributes(

			attribute.String("llm.model", model),

			attribute.String("intent.type", intentType),

			attribute.String("operation.type", "llm_processing"),

		),

	)

}



// StartRAGRetrieval starts a RAG retrieval span.

func (nt *NetworkIntentTracer) StartRAGRetrieval(ctx context.Context, query string) (context.Context, trace.Span) {

	return nt.tracer.Start(ctx, "networkintent.rag_retrieval",

		trace.WithAttributes(

			attribute.String("rag.query", query),

			attribute.String("operation.type", "rag_retrieval"),

		),

	)

}



// StartGitOpsGeneration starts a GitOps generation span.

func (nt *NetworkIntentTracer) StartGitOpsGeneration(ctx context.Context, packageType string) (context.Context, trace.Span) {

	return nt.tracer.Start(ctx, "networkintent.gitops_generation",

		trace.WithAttributes(

			attribute.String("gitops.package_type", packageType),

			attribute.String("operation.type", "gitops_generation"),

		),

	)

}



// AddSpanAttributes adds attributes to the current span.

func (nt *NetworkIntentTracer) AddSpanAttributes(span trace.Span, attrs ...attribute.KeyValue) {

	span.SetAttributes(attrs...)

}



// RecordError records an error in the span.

func (nt *NetworkIntentTracer) RecordError(span trace.Span, err error) {

	span.RecordError(err)

	span.SetStatus(codes.Error, err.Error())

}



// SetSpanStatus sets the span status.

func (nt *NetworkIntentTracer) SetSpanStatus(span trace.Span, code codes.Code, description string) {

	span.SetStatus(code, description)

}



// ORANInterfaceTracer provides tracing for O-RAN interface operations.

type ORANInterfaceTracer struct {

	tracer trace.Tracer

}



// NewORANInterfaceTracer creates a new O-RAN interface tracer.

func NewORANInterfaceTracer(tracer trace.Tracer) *ORANInterfaceTracer {

	return &ORANInterfaceTracer{tracer: tracer}

}



// StartInterfaceOperation starts an interface operation span.

func (ot *ORANInterfaceTracer) StartInterfaceOperation(ctx context.Context, interfaceType, operation, endpoint string) (context.Context, trace.Span) {

	return ot.tracer.Start(ctx, fmt.Sprintf("oran.%s.%s", interfaceType, operation),

		trace.WithAttributes(

			attribute.String("oran.interface", interfaceType),

			attribute.String("oran.operation", operation),

			attribute.String("oran.endpoint", endpoint),

			attribute.String("operation.type", "oran_interface"),

		),

	)

}



// StartPolicyOperation starts a policy operation span.

func (ot *ORANInterfaceTracer) StartPolicyOperation(ctx context.Context, policyType, operation string) (context.Context, trace.Span) {

	return ot.tracer.Start(ctx, fmt.Sprintf("oran.policy.%s", operation),

		trace.WithAttributes(

			attribute.String("oran.policy_type", policyType),

			attribute.String("oran.operation", operation),

			attribute.String("operation.type", "policy_operation"),

		),

	)

}



// E2NodeSetTracer provides tracing for E2NodeSet operations.

type E2NodeSetTracer struct {

	tracer trace.Tracer

}



// NewE2NodeSetTracer creates a new E2NodeSet tracer.

func NewE2NodeSetTracer(tracer trace.Tracer) *E2NodeSetTracer {

	return &E2NodeSetTracer{tracer: tracer}

}



// StartReconciliation starts a reconciliation span.

func (et *E2NodeSetTracer) StartReconciliation(ctx context.Context, name, namespace string) (context.Context, trace.Span) {

	return et.tracer.Start(ctx, "e2nodeset.reconciliation",

		trace.WithAttributes(

			attribute.String("e2nodeset.name", name),

			attribute.String("e2nodeset.namespace", namespace),

			attribute.String("operation.type", "reconciliation"),

		),

	)

}



// StartScalingOperation starts a scaling operation span.

func (et *E2NodeSetTracer) StartScalingOperation(ctx context.Context, name, namespace string, fromReplicas, toReplicas int) (context.Context, trace.Span) {

	return et.tracer.Start(ctx, "e2nodeset.scaling",

		trace.WithAttributes(

			attribute.String("e2nodeset.name", name),

			attribute.String("e2nodeset.namespace", namespace),

			attribute.Int("scaling.from_replicas", fromReplicas),

			attribute.Int("scaling.to_replicas", toReplicas),

			attribute.String("operation.type", "scaling"),

		),

	)

}



// MetricsInstrumentation provides OpenTelemetry metrics instrumentation.

type MetricsInstrumentation struct {

	meter                    metric.Meter

	intentProcessingDuration metric.Float64Histogram

	intentProcessingCounter  metric.Int64Counter

	llmRequestDuration       metric.Float64Histogram

	llmTokensUsed            metric.Int64Counter

	oranRequestDuration      metric.Float64Histogram

	oranRequestCounter       metric.Int64Counter

	systemResourceUsage      metric.Float64Gauge

}



// NewMetricsInstrumentation creates a new metrics instrumentation.

func NewMetricsInstrumentation(meter metric.Meter) (*MetricsInstrumentation, error) {

	mi := &MetricsInstrumentation{meter: meter}



	var err error



	// Intent processing metrics.

	mi.intentProcessingDuration, err = meter.Float64Histogram(

		"nephoran.intent.processing.duration",

		metric.WithDescription("Duration of intent processing operations"),

		metric.WithUnit("s"),

	)

	if err != nil {

		return nil, err

	}



	mi.intentProcessingCounter, err = meter.Int64Counter(

		"nephoran.intent.processing.total",

		metric.WithDescription("Total number of intent processing operations"),

	)

	if err != nil {

		return nil, err

	}



	// LLM metrics.

	mi.llmRequestDuration, err = meter.Float64Histogram(

		"nephoran.llm.request.duration",

		metric.WithDescription("Duration of LLM requests"),

		metric.WithUnit("s"),

	)

	if err != nil {

		return nil, err

	}



	mi.llmTokensUsed, err = meter.Int64Counter(

		"nephoran.llm.tokens.used",

		metric.WithDescription("Total number of LLM tokens used"),

	)

	if err != nil {

		return nil, err

	}



	// O-RAN metrics.

	mi.oranRequestDuration, err = meter.Float64Histogram(

		"nephoran.oran.request.duration",

		metric.WithDescription("Duration of O-RAN interface requests"),

		metric.WithUnit("s"),

	)

	if err != nil {

		return nil, err

	}



	mi.oranRequestCounter, err = meter.Int64Counter(

		"nephoran.oran.request.total",

		metric.WithDescription("Total number of O-RAN interface requests"),

	)

	if err != nil {

		return nil, err

	}



	// System metrics.

	mi.systemResourceUsage, err = meter.Float64Gauge(

		"nephoran.system.resource.usage",

		metric.WithDescription("System resource usage"),

	)

	if err != nil {

		return nil, err

	}



	return mi, nil

}



// RecordIntentProcessing records intent processing metrics.

func (mi *MetricsInstrumentation) RecordIntentProcessing(ctx context.Context, duration time.Duration, intentType, status string) {

	mi.intentProcessingDuration.Record(ctx, duration.Seconds(),

		metric.WithAttributes(

			attribute.String("intent_type", intentType),

			attribute.String("status", status),

		),

	)

	mi.intentProcessingCounter.Add(ctx, 1,

		metric.WithAttributes(

			attribute.String("intent_type", intentType),

			attribute.String("status", status),

		),

	)

}



// RecordLLMRequest records LLM request metrics.

func (mi *MetricsInstrumentation) RecordLLMRequest(ctx context.Context, duration time.Duration, model, status string, tokensUsed int64) {

	mi.llmRequestDuration.Record(ctx, duration.Seconds(),

		metric.WithAttributes(

			attribute.String("model", model),

			attribute.String("status", status),

		),

	)

	mi.llmTokensUsed.Add(ctx, tokensUsed,

		metric.WithAttributes(

			attribute.String("model", model),

		),

	)

}



// RecordORANRequest records O-RAN request metrics.

func (mi *MetricsInstrumentation) RecordORANRequest(ctx context.Context, duration time.Duration, interfaceType, operation, status string) {

	mi.oranRequestDuration.Record(ctx, duration.Seconds(),

		metric.WithAttributes(

			attribute.String("interface", interfaceType),

			attribute.String("operation", operation),

			attribute.String("status", status),

		),

	)

	mi.oranRequestCounter.Add(ctx, 1,

		metric.WithAttributes(

			attribute.String("interface", interfaceType),

			attribute.String("operation", operation),

			attribute.String("status", status),

		),

	)

}



// RecordResourceUsage records system resource usage.

func (mi *MetricsInstrumentation) RecordResourceUsage(ctx context.Context, resourceType string, usage float64) {

	mi.systemResourceUsage.Record(ctx, usage,

		metric.WithAttributes(

			attribute.String("resource_type", resourceType),

		),

	)

}



// Note: Utility functions have been moved to pkg/config/env_helpers.go.



// TraceWithSpan executes a function within a trace span.

func TraceWithSpan(ctx context.Context, tracer trace.Tracer, spanName string, fn func(context.Context, trace.Span) error, attrs ...attribute.KeyValue) error {

	ctx, span := tracer.Start(ctx, spanName, trace.WithAttributes(attrs...))

	defer span.End()



	if err := fn(ctx, span); err != nil {

		span.RecordError(err)

		span.SetStatus(codes.Error, err.Error())

		return err

	}



	span.SetStatus(codes.Ok, "")

	return nil

}

