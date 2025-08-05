//go:build !disable_rag && !test

package rag

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// TracingManager handles distributed tracing for RAG components
type TracingManager struct {
	tracer        trace.Tracer
	tracerProvider *sdktrace.TracerProvider
	serviceName   string
	version       string
}

// TracingConfig holds tracing configuration
type TracingConfig struct {
	ServiceName     string  `json:"service_name"`
	ServiceVersion  string  `json:"service_version"`
	JaegerEndpoint  string  `json:"jaeger_endpoint"`
	SamplingRate    float64 `json:"sampling_rate"`
	Environment     string  `json:"environment"`
	EnableTracing   bool    `json:"enable_tracing"`
}

// NewTracingManager creates a new tracing manager
func NewTracingManager(config *TracingConfig) (*TracingManager, error) {
	if !config.EnableTracing {
		return &TracingManager{
			tracer:      otel.Tracer("noop"),
			serviceName: config.ServiceName,
			version:     config.ServiceVersion,
		}, nil
	}

	// Create Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(config.JaegerEndpoint)))
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	// Create resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(config.Environment),
			attribute.String("nephoran.component", "rag-pipeline"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(config.SamplingRate)),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer := tp.Tracer(config.ServiceName)

	return &TracingManager{
		tracer:         tracer,
		tracerProvider: tp,
		serviceName:    config.ServiceName,
		version:        config.ServiceVersion,
	}, nil
}

// Shutdown gracefully shuts down the tracing manager
func (tm *TracingManager) Shutdown(ctx context.Context) error {
	if tm.tracerProvider != nil {
		return tm.tracerProvider.Shutdown(ctx)
	}
	return nil
}

// StartSpan starts a new trace span with comprehensive attributes
func (tm *TracingManager) StartSpan(ctx context.Context, operationName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	// Add default attributes
	defaultOpts := []trace.SpanStartOption{
		trace.WithAttributes(
			attribute.String("nephoran.service", tm.serviceName),
			attribute.String("nephoran.version", tm.version),
			attribute.String("nephoran.operation", operationName),
		),
	}
	
	// Merge with provided options
	allOpts := append(defaultOpts, opts...)
	
	return tm.tracer.Start(ctx, operationName, allOpts...)
}

// InstrumentDocumentProcessing adds tracing to document processing
func (tm *TracingManager) InstrumentDocumentProcessing(ctx context.Context, docID, docType string, fn func(context.Context) error) error {
	ctx, span := tm.StartSpan(ctx, "document_processing",
		trace.WithAttributes(
			attribute.String("document.id", docID),
			attribute.String("document.type", docType),
			attribute.String("nephoran.component", "document-processor"),
		),
	)
	defer span.End()

	startTime := time.Now()
	
	err := fn(ctx)
	
	duration := time.Since(startTime)
	span.SetAttributes(
		attribute.Int64("document.processing_duration_ms", duration.Milliseconds()),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.String("error.type", "processing_error"))
	} else {
		span.SetStatus(codes.Ok, "document processed successfully")
	}

	return err
}

// InstrumentChunking adds tracing to document chunking
func (tm *TracingManager) InstrumentChunking(ctx context.Context, docID string, strategy string, fn func(context.Context) ([]DocumentChunk, error)) ([]DocumentChunk, error) {
	ctx, span := tm.StartSpan(ctx, "document_chunking",
		trace.WithAttributes(
			attribute.String("document.id", docID),
			attribute.String("chunking.strategy", strategy),
			attribute.String("nephoran.component", "chunking-service"),
		),
	)
	defer span.End()

	startTime := time.Now()
	
	chunks, err := fn(ctx)
	
	duration := time.Since(startTime)
	span.SetAttributes(
		attribute.Int64("chunking.duration_ms", duration.Milliseconds()),
		attribute.Int("chunking.chunks_created", len(chunks)),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.String("error.type", "chunking_error"))
	} else {
		span.SetStatus(codes.Ok, "document chunked successfully")
		
		// Add chunk statistics
		if len(chunks) > 0 {
			totalSize := 0
			for _, chunk := range chunks {
				totalSize += len(chunk.Content)
			}
			avgChunkSize := totalSize / len(chunks)
			
			span.SetAttributes(
				attribute.Int("chunking.total_content_size", totalSize),
				attribute.Int("chunking.avg_chunk_size", avgChunkSize),
			)
		}
	}

	return chunks, err
}

// InstrumentEmbeddingGeneration adds tracing to embedding generation
func (tm *TracingManager) InstrumentEmbeddingGeneration(ctx context.Context, modelName string, inputTexts []string, fn func(context.Context) ([][]float32, error)) ([][]float32, error) {
	ctx, span := tm.StartSpan(ctx, "embedding_generation",
		trace.WithAttributes(
			attribute.String("embedding.model", modelName),
			attribute.Int("embedding.input_count", len(inputTexts)),
			attribute.String("nephoran.component", "embedding-service"),
		),
	)
	defer span.End()

	startTime := time.Now()
	
	embeddings, err := fn(ctx)
	
	duration := time.Since(startTime)
	span.SetAttributes(
		attribute.Int64("embedding.generation_duration_ms", duration.Milliseconds()),
		attribute.Int("embedding.output_count", len(embeddings)),
	)

	if len(inputTexts) > 0 {
		totalInputLength := 0
		for _, text := range inputTexts {
			totalInputLength += len(text)
		}
		avgInputLength := totalInputLength / len(inputTexts)
		
		span.SetAttributes(
			attribute.Int("embedding.total_input_length", totalInputLength),
			attribute.Int("embedding.avg_input_length", avgInputLength),
		)
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.String("error.type", "embedding_error"))
	} else {
		span.SetStatus(codes.Ok, "embeddings generated successfully")
		
		if len(embeddings) > 0 && len(embeddings[0]) > 0 {
			span.SetAttributes(
				attribute.Int("embedding.vector_dimension", len(embeddings[0])),
			)
		}
	}

	return embeddings, err
}

// InstrumentVectorStore adds tracing to vector store operations
func (tm *TracingManager) InstrumentVectorStore(ctx context.Context, operation string, objectCount int, fn func(context.Context) error) error {
	ctx, span := tm.StartSpan(ctx, "vector_store_operation",
		trace.WithAttributes(
			attribute.String("vector_store.operation", operation),
			attribute.Int("vector_store.object_count", objectCount),
			attribute.String("nephoran.component", "weaviate-client"),
		),
	)
	defer span.End()

	startTime := time.Now()
	
	err := fn(ctx)
	
	duration := time.Since(startTime)
	span.SetAttributes(
		attribute.Int64("vector_store.operation_duration_ms", duration.Milliseconds()),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.String("error.type", "vector_store_error"))
	} else {
		span.SetStatus(codes.Ok, "vector store operation completed successfully")
	}

	return err
}

// InstrumentRetrieval adds tracing to document retrieval
func (tm *TracingManager) InstrumentRetrieval(ctx context.Context, query string, searchType string, fn func(context.Context) ([]RetrievedDocument, error)) ([]RetrievedDocument, error) {
	ctx, span := tm.StartSpan(ctx, "document_retrieval",
		trace.WithAttributes(
			attribute.String("retrieval.query", query),
			attribute.String("retrieval.search_type", searchType),
			attribute.Int("retrieval.query_length", len(query)),
			attribute.String("nephoran.component", "retrieval-service"),
		),
	)
	defer span.End()

	startTime := time.Now()
	
	documents, err := fn(ctx)
	
	duration := time.Since(startTime)
	span.SetAttributes(
		attribute.Int64("retrieval.duration_ms", duration.Milliseconds()),
		attribute.Int("retrieval.documents_found", len(documents)),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.String("error.type", "retrieval_error"))
	} else {
		span.SetStatus(codes.Ok, "documents retrieved successfully")
		
		// Add retrieval quality metrics
		if len(documents) > 0 {
			totalScore := 0.0
			for _, doc := range documents {
				totalScore += float64(doc.Score)
			}
			avgScore := totalScore / float64(len(documents))
			
			span.SetAttributes(
				attribute.Float64("retrieval.avg_score", avgScore),
				attribute.Float64("retrieval.top_score", float64(documents[0].Score)),
			)
		}
	}

	return documents, err
}

// InstrumentContextAssembly adds tracing to context assembly
func (tm *TracingManager) InstrumentContextAssembly(ctx context.Context, documents []RetrievedDocument, strategy string, fn func(context.Context) (string, error)) (string, error) {
	ctx, span := tm.StartSpan(ctx, "context_assembly",
		trace.WithAttributes(
			attribute.String("context.assembly_strategy", strategy),
			attribute.Int("context.input_documents", len(documents)),
			attribute.String("nephoran.component", "context-assembler"),
		),
	)
	defer span.End()

	startTime := time.Now()
	
	context_str, err := fn(ctx)
	
	duration := time.Since(startTime)
	span.SetAttributes(
		attribute.Int64("context.assembly_duration_ms", duration.Milliseconds()),
		attribute.Int("context.output_length", len(context_str)),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.String("error.type", "context_assembly_error"))
	} else {
		span.SetStatus(codes.Ok, "context assembled successfully")
		
		// Calculate compression ratio
		if len(documents) > 0 {
			totalInputLength := 0
			for _, doc := range documents {
				totalInputLength += len(doc.Content)
			}
			
			compressionRatio := float64(len(context_str)) / float64(totalInputLength)
			span.SetAttributes(
				attribute.Int("context.total_input_length", totalInputLength),
				attribute.Float64("context.compression_ratio", compressionRatio),
			)
		}
	}

	return context_str, err
}

// InstrumentRAGQuery adds end-to-end tracing for RAG queries
func (tm *TracingManager) InstrumentRAGQuery(ctx context.Context, query string, intentType string, fn func(context.Context) (*RAGResponse, error)) (*RAGResponse, error) {
	ctx, span := tm.StartSpan(ctx, "rag_query",
		trace.WithAttributes(
			attribute.String("rag.query", query),
			attribute.String("rag.intent_type", intentType),
			attribute.Int("rag.query_length", len(query)),
			attribute.String("nephoran.component", "rag-service"),
		),
	)
	defer span.End()

	startTime := time.Now()
	
	response, err := fn(ctx)
	
	duration := time.Since(startTime)
	span.SetAttributes(
		attribute.Int64("rag.total_duration_ms", duration.Milliseconds()),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.String("error.type", "rag_query_error"))
	} else {
		span.SetStatus(codes.Ok, "RAG query completed successfully")
		
		if response != nil {
			span.SetAttributes(
				attribute.Int("rag.response_length", len(response.Answer)),
				attribute.Int("rag.sources_used", len(response.SourceDocuments)),
				attribute.Float64("rag.confidence_score", float64(response.Confidence)),
			)
			
			// Add source quality metrics
			if len(response.SourceDocuments) > 0 {
				totalScore := 0.0
				for _, source := range response.SourceDocuments {
					totalScore += float64(source.Score)
				}
				avgRelevance := totalScore / float64(len(response.SourceDocuments))
				
				span.SetAttributes(
					attribute.Float64("rag.avg_source_relevance", avgRelevance),
				)
			}
		}
	}

	return response, err
}

// InstrumentCacheOperation adds tracing to cache operations
func (tm *TracingManager) InstrumentCacheOperation(ctx context.Context, operation string, cacheKey string, fn func(context.Context) (interface{}, bool, error)) (interface{}, bool, error) {
	ctx, span := tm.StartSpan(ctx, "cache_operation",
		trace.WithAttributes(
			attribute.String("cache.operation", operation),
			attribute.String("cache.key", cacheKey),
			attribute.String("nephoran.component", "redis-cache"),
		),
	)
	defer span.End()

	startTime := time.Now()
	
	value, hit, err := fn(ctx)
	
	duration := time.Since(startTime)
	span.SetAttributes(
		attribute.Int64("cache.operation_duration_ms", duration.Milliseconds()),
		attribute.Bool("cache.hit", hit),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.String("error.type", "cache_error"))
	} else {
		span.SetStatus(codes.Ok, "cache operation completed successfully")
		
		if hit {
			span.SetAttributes(attribute.String("cache.status", "hit"))
		} else {
			span.SetAttributes(attribute.String("cache.status", "miss"))
		}
	}

	return value, hit, err
}

// PropagateTraceContext extracts trace context for cross-service calls
func (tm *TracingManager) PropagateTraceContext(ctx context.Context) map[string]string {
	headers := make(map[string]string)
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(headers))
	return headers
}

// ExtractTraceContext injects trace context from incoming requests
func (tm *TracingManager) ExtractTraceContext(ctx context.Context, headers map[string]string) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(headers))
}

// AddCustomAttributes adds custom attributes to the current span
func (tm *TracingManager) AddCustomAttributes(ctx context.Context, attributes ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.SetAttributes(attributes...)
	}
}

// RecordException records an exception in the current span
func (tm *TracingManager) RecordException(ctx context.Context, err error, description string) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.RecordError(err)
		span.SetAttributes(
			attribute.String("exception.type", fmt.Sprintf("%T", err)),
			attribute.String("exception.message", err.Error()),
			attribute.String("exception.description", description),
		)
	}
}

// CreateChildSpan creates a child span from the current context
func (tm *TracingManager) CreateChildSpan(ctx context.Context, operationName string, attributes ...attribute.KeyValue) (context.Context, trace.Span) {
	return tm.StartSpan(ctx, operationName, trace.WithAttributes(attributes...))
}

// GetDefaultTracingConfig returns default tracing configuration
func GetDefaultTracingConfig() *TracingConfig {
	return &TracingConfig{
		ServiceName:     "nephoran-rag-service",
		ServiceVersion:  "1.0.0",
		JaegerEndpoint:  "http://jaeger-collector:14268/api/traces",
		SamplingRate:    0.1,
		Environment:     "production",
		EnableTracing:   true,
	}
}