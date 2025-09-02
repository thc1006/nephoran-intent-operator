// Copyright 2024 Nephio
// SPDX-License-Identifier: Apache-2.0

package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// MetricsRecorder defines the interface for recording metrics
type MetricsRecorder interface {
	RecordEvent(event MetricEvent) error
	RecordBatch(events []MetricEvent) error
	GetMetrics(filter MetricFilter) ([]MetricEvent, error)
	Reset() error
}

// MetricEvent represents a single metric event
type MetricEvent struct {
	Name        string                 `json:"name"`
	Type        MetricType             `json:"type"`
	Value       float64                `json:"value"`
	Labels      map[string]string      `json:"labels"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// MetricType represents the type of metric
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

// MetricFilter defines filters for querying metrics
type MetricFilter struct {
	Names     []string          `json:"names,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
	StartTime *time.Time        `json:"start_time,omitempty"`
	EndTime   *time.Time        `json:"end_time,omitempty"`
	Limit     int               `json:"limit,omitempty"`
}

// InMemoryMetricsRecorder implements MetricsRecorder using in-memory storage
type InMemoryMetricsRecorder struct {
	events []MetricEvent
	mu     sync.RWMutex
}

// NewInMemoryMetricsRecorder creates a new in-memory metrics recorder
func NewInMemoryMetricsRecorder() *InMemoryMetricsRecorder {
	return &InMemoryMetricsRecorder{
		events: make([]MetricEvent, 0),
	}
}

// RecordEvent records a single metric event
func (r *InMemoryMetricsRecorder) RecordEvent(event MetricEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	r.events = append(r.events, event)

	// Keep only last 10000 events to prevent memory bloat
	if len(r.events) > 10000 {
		r.events = r.events[len(r.events)-10000:]
	}

	return nil
}

// RecordBatch records multiple metric events
func (r *InMemoryMetricsRecorder) RecordBatch(events []MetricEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for i := range events {
		if events[i].Timestamp.IsZero() {
			events[i].Timestamp = now
		}
	}

	r.events = append(r.events, events...)

	// Keep only last 10000 events
	if len(r.events) > 10000 {
		r.events = r.events[len(r.events)-10000:]
	}

	return nil
}

// GetMetrics retrieves metrics based on filter
func (r *InMemoryMetricsRecorder) GetMetrics(filter MetricFilter) ([]MetricEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []MetricEvent

	for _, event := range r.events {
		if r.matchesFilter(event, filter) {
			result = append(result, event)
		}
	}

	// Apply limit if specified
	if filter.Limit > 0 && len(result) > filter.Limit {
		result = result[len(result)-filter.Limit:]
	}

	return result, nil
}

// matchesFilter checks if an event matches the given filter
func (r *InMemoryMetricsRecorder) matchesFilter(event MetricEvent, filter MetricFilter) bool {
	// Check name filter
	if len(filter.Names) > 0 {
		found := false
		for _, name := range filter.Names {
			if event.Name == name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check label filters
	for key, value := range filter.Labels {
		if eventValue, exists := event.Labels[key]; !exists || eventValue != value {
			return false
		}
	}

	// Check time range
	if filter.StartTime != nil && event.Timestamp.Before(*filter.StartTime) {
		return false
	}
	if filter.EndTime != nil && event.Timestamp.After(*filter.EndTime) {
		return false
	}

	return true
}

// Reset clears all recorded events
func (r *InMemoryMetricsRecorder) Reset() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.events = make([]MetricEvent, 0)
	return nil
}

// PrometheusMetricsRecorder implements MetricsRecorder using Prometheus
type PrometheusMetricsRecorder struct {
	counters   map[string]*prometheus.CounterVec
	gauges     map[string]*prometheus.GaugeVec
	histograms map[string]*prometheus.HistogramVec
	summaries  map[string]*prometheus.SummaryVec
	registry   prometheus.Registerer
	mu         sync.RWMutex
}

// NewPrometheusMetricsRecorder creates a new Prometheus metrics recorder
func NewPrometheusMetricsRecorder(registry prometheus.Registerer) *PrometheusMetricsRecorder {
	if registry == nil {
		registry = prometheus.DefaultRegisterer
	}

	return &PrometheusMetricsRecorder{
		counters:   make(map[string]*prometheus.CounterVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
		histograms: make(map[string]*prometheus.HistogramVec),
		summaries:  make(map[string]*prometheus.SummaryVec),
		registry:   registry,
	}
}

// RecordEvent records a single metric event
func (r *PrometheusMetricsRecorder) RecordEvent(event MetricEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch event.Type {
	case MetricTypeCounter:
		return r.recordCounter(event)
	case MetricTypeGauge:
		return r.recordGauge(event)
	case MetricTypeHistogram:
		return r.recordHistogram(event)
	case MetricTypeSummary:
		return r.recordSummary(event)
	default:
		return fmt.Errorf("unsupported metric type: %s", event.Type)
	}
}

// recordCounter records a counter metric
func (r *PrometheusMetricsRecorder) recordCounter(event MetricEvent) error {
	counter, exists := r.counters[event.Name]
	if !exists {
		labelNames := make([]string, 0, len(event.Labels))
		for key := range event.Labels {
			labelNames = append(labelNames, key)
		}

		counter = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: event.Name,
				Help: fmt.Sprintf("Counter metric for %s", event.Name),
			},
			labelNames,
		)

		if err := r.registry.Register(counter); err != nil {
			return fmt.Errorf("failed to register counter %s: %w", event.Name, err)
		}

		r.counters[event.Name] = counter
	}

	// Convert map[string]string to prometheus.Labels
	promLabels := prometheus.Labels(event.Labels)
	counter.With(promLabels).Add(event.Value)
	return nil
}

// recordGauge records a gauge metric
func (r *PrometheusMetricsRecorder) recordGauge(event MetricEvent) error {
	gauge, exists := r.gauges[event.Name]
	if !exists {
		labelNames := make([]string, 0, len(event.Labels))
		for key := range event.Labels {
			labelNames = append(labelNames, key)
		}

		gauge = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: event.Name,
				Help: fmt.Sprintf("Gauge metric for %s", event.Name),
			},
			labelNames,
		)

		if err := r.registry.Register(gauge); err != nil {
			return fmt.Errorf("failed to register gauge %s: %w", event.Name, err)
		}

		r.gauges[event.Name] = gauge
	}

	// Convert map[string]string to prometheus.Labels
	promLabels := prometheus.Labels(event.Labels)
	gauge.With(promLabels).Set(event.Value)
	return nil
}

// recordHistogram records a histogram metric
func (r *PrometheusMetricsRecorder) recordHistogram(event MetricEvent) error {
	histogram, exists := r.histograms[event.Name]
	if !exists {
		labelNames := make([]string, 0, len(event.Labels))
		for key := range event.Labels {
			labelNames = append(labelNames, key)
		}

		histogram = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    event.Name,
				Help:    fmt.Sprintf("Histogram metric for %s", event.Name),
				Buckets: prometheus.DefBuckets,
			},
			labelNames,
		)

		if err := r.registry.Register(histogram); err != nil {
			return fmt.Errorf("failed to register histogram %s: %w", event.Name, err)
		}

		r.histograms[event.Name] = histogram
	}

	// Convert map[string]string to prometheus.Labels
	promLabels := prometheus.Labels(event.Labels)
	histogram.With(promLabels).Observe(event.Value)
	return nil
}

// recordSummary records a summary metric
func (r *PrometheusMetricsRecorder) recordSummary(event MetricEvent) error {
	summary, exists := r.summaries[event.Name]
	if !exists {
		labelNames := make([]string, 0, len(event.Labels))
		for key := range event.Labels {
			labelNames = append(labelNames, key)
		}

		summary = prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name: event.Name,
				Help: fmt.Sprintf("Summary metric for %s", event.Name),
			},
			labelNames,
		)

		if err := r.registry.Register(summary); err != nil {
			return fmt.Errorf("failed to register summary %s: %w", event.Name, err)
		}

		r.summaries[event.Name] = summary
	}

	// Convert map[string]string to prometheus.Labels
	promLabels := prometheus.Labels(event.Labels)
	summary.With(promLabels).Observe(event.Value)
	return nil
}

// RecordBatch records multiple metric events
func (r *PrometheusMetricsRecorder) RecordBatch(events []MetricEvent) error {
	for _, event := range events {
		if err := r.RecordEvent(event); err != nil {
			return fmt.Errorf("failed to record event %s: %w", event.Name, err)
		}
	}
	return nil
}

// GetMetrics retrieves metrics (not implemented for Prometheus)
func (r *PrometheusMetricsRecorder) GetMetrics(filter MetricFilter) ([]MetricEvent, error) {
	return nil, fmt.Errorf("GetMetrics not supported for Prometheus recorder")
}

// Reset clears all registered metrics
func (r *PrometheusMetricsRecorder) Reset() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.counters = make(map[string]*prometheus.CounterVec)
	r.gauges = make(map[string]*prometheus.GaugeVec)
	r.histograms = make(map[string]*prometheus.HistogramVec)
	r.summaries = make(map[string]*prometheus.SummaryVec)

	return nil
}

// OpenTelemetryMetricsRecorder implements MetricsRecorder using OpenTelemetry
type OpenTelemetryMetricsRecorder struct {
	meter      metric.Meter
	counters   map[string]metric.Int64Counter
	gauges     map[string]metric.Float64Gauge
	histograms map[string]metric.Float64Histogram
	mu         sync.RWMutex
}

// NewOpenTelemetryMetricsRecorder creates a new OpenTelemetry metrics recorder
func NewOpenTelemetryMetricsRecorder(meter metric.Meter) *OpenTelemetryMetricsRecorder {
	return &OpenTelemetryMetricsRecorder{
		meter:      meter,
		counters:   make(map[string]metric.Int64Counter),
		gauges:     make(map[string]metric.Float64Gauge),
		histograms: make(map[string]metric.Float64Histogram),
	}
}

// RecordEvent records a single metric event
func (r *OpenTelemetryMetricsRecorder) RecordEvent(event MetricEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ctx := context.Background()
	attrs := make([]attribute.KeyValue, 0, len(event.Labels))
	for key, value := range event.Labels {
		attrs = append(attrs, attribute.String(key, value))
	}

	switch event.Type {
	case MetricTypeCounter:
		return r.recordOTelCounter(ctx, event, attrs)
	case MetricTypeGauge:
		return r.recordOTelGauge(ctx, event, attrs)
	case MetricTypeHistogram:
		return r.recordOTelHistogram(ctx, event, attrs)
	default:
		return fmt.Errorf("unsupported metric type for OpenTelemetry: %s", event.Type)
	}
}

// recordOTelCounter records an OpenTelemetry counter metric
func (r *OpenTelemetryMetricsRecorder) recordOTelCounter(ctx context.Context, event MetricEvent, attrs []attribute.KeyValue) error {
	counter, exists := r.counters[event.Name]
	if !exists {
		var err error
		counter, err = r.meter.Int64Counter(event.Name)
		if err != nil {
			return fmt.Errorf("failed to create counter %s: %w", event.Name, err)
		}
		r.counters[event.Name] = counter
	}

	counter.Add(ctx, int64(event.Value), metric.WithAttributes(attrs...))
	return nil
}

// recordOTelGauge records an OpenTelemetry gauge metric
func (r *OpenTelemetryMetricsRecorder) recordOTelGauge(ctx context.Context, event MetricEvent, attrs []attribute.KeyValue) error {
	gauge, exists := r.gauges[event.Name]
	if !exists {
		var err error
		gauge, err = r.meter.Float64Gauge(event.Name)
		if err != nil {
			return fmt.Errorf("failed to create gauge %s: %w", event.Name, err)
		}
		r.gauges[event.Name] = gauge
	}

	gauge.Record(ctx, event.Value, metric.WithAttributes(attrs...))
	return nil
}

// recordOTelHistogram records an OpenTelemetry histogram metric
func (r *OpenTelemetryMetricsRecorder) recordOTelHistogram(ctx context.Context, event MetricEvent, attrs []attribute.KeyValue) error {
	histogram, exists := r.histograms[event.Name]
	if !exists {
		var err error
		histogram, err = r.meter.Float64Histogram(event.Name)
		if err != nil {
			return fmt.Errorf("failed to create histogram %s: %w", event.Name, err)
		}
		r.histograms[event.Name] = histogram
	}

	histogram.Record(ctx, event.Value, metric.WithAttributes(attrs...))
	return nil
}

// RecordBatch records multiple metric events
func (r *OpenTelemetryMetricsRecorder) RecordBatch(events []MetricEvent) error {
	for _, event := range events {
		if err := r.RecordEvent(event); err != nil {
			return fmt.Errorf("failed to record event %s: %w", event.Name, err)
		}
	}
	return nil
}

// GetMetrics retrieves metrics (not implemented for OpenTelemetry)
func (r *OpenTelemetryMetricsRecorder) GetMetrics(filter MetricFilter) ([]MetricEvent, error) {
	return nil, fmt.Errorf("GetMetrics not supported for OpenTelemetry recorder")
}

// Reset clears all registered metrics
func (r *OpenTelemetryMetricsRecorder) Reset() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.counters = make(map[string]metric.Int64Counter)
	r.gauges = make(map[string]metric.Float64Gauge)
	r.histograms = make(map[string]metric.Float64Histogram)

	return nil
}

// CompositeMetricsRecorder combines multiple recorders
type CompositeMetricsRecorder struct {
	recorders []MetricsRecorder
	mu        sync.RWMutex
}

// NewCompositeMetricsRecorder creates a new composite metrics recorder
func NewCompositeMetricsRecorder(recorders ...MetricsRecorder) *CompositeMetricsRecorder {
	return &CompositeMetricsRecorder{
		recorders: recorders,
	}
}

// AddRecorder adds a metrics recorder
func (r *CompositeMetricsRecorder) AddRecorder(recorder MetricsRecorder) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recorders = append(r.recorders, recorder)
}

// RecordEvent records a single metric event to all recorders
func (r *CompositeMetricsRecorder) RecordEvent(event MetricEvent) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var lastError error
	for _, recorder := range r.recorders {
		if err := recorder.RecordEvent(event); err != nil {
			log.Log.Error(err, "failed to record event in recorder", "event", event.Name)
			lastError = err
		}
	}

	return lastError
}

// RecordBatch records multiple metric events to all recorders
func (r *CompositeMetricsRecorder) RecordBatch(events []MetricEvent) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var lastError error
	for _, recorder := range r.recorders {
		if err := recorder.RecordBatch(events); err != nil {
			log.Log.Error(err, "failed to record batch in recorder")
			lastError = err
		}
	}

	return lastError
}

// GetMetrics retrieves metrics from the first recorder that supports it
func (r *CompositeMetricsRecorder) GetMetrics(filter MetricFilter) ([]MetricEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, recorder := range r.recorders {
		if metrics, err := recorder.GetMetrics(filter); err == nil {
			return metrics, nil
		}
	}

	return nil, fmt.Errorf("no recorder supports GetMetrics")
}

// Reset resets all recorders
func (r *CompositeMetricsRecorder) Reset() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var lastError error
	for _, recorder := range r.recorders {
		if err := recorder.Reset(); err != nil {
			log.Log.Error(err, "failed to reset recorder")
			lastError = err
		}
	}

	return lastError
}

// MetricsAggregator aggregates metrics over time windows
type MetricsAggregator struct {
	recorder      MetricsRecorder
	windowSize    time.Duration
	aggregateFunc func([]MetricEvent) MetricEvent
	mu            sync.RWMutex
}

// NewMetricsAggregator creates a new metrics aggregator
func NewMetricsAggregator(recorder MetricsRecorder, windowSize time.Duration) *MetricsAggregator {
	return &MetricsAggregator{
		recorder:   recorder,
		windowSize: windowSize,
		aggregateFunc: func(events []MetricEvent) MetricEvent {
			if len(events) == 0 {
				return MetricEvent{}
			}

			// Default aggregation: sum for counters, average for others
			sum := 0.0
			for _, event := range events {
				sum += event.Value
			}

			result := events[0] // Copy first event as template
			if result.Type == MetricTypeCounter {
				result.Value = sum
			} else {
				result.Value = sum / float64(len(events))
			}
			result.Timestamp = time.Now()

			return result
		},
	}
}

// SetAggregateFunction sets a custom aggregation function
func (a *MetricsAggregator) SetAggregateFunction(fn func([]MetricEvent) MetricEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.aggregateFunc = fn
}

// AggregateAndRecord aggregates metrics within the time window and records the result
func (a *MetricsAggregator) AggregateAndRecord(name string, labels map[string]string) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	now := time.Now()
	startTime := now.Add(-a.windowSize)

	filter := MetricFilter{
		Names:     []string{name},
		Labels:    labels,
		StartTime: &startTime,
		EndTime:   &now,
	}

	events, err := a.recorder.GetMetrics(filter)
	if err != nil {
		return fmt.Errorf("failed to get metrics for aggregation: %w", err)
	}

	if len(events) == 0 {
		return nil // No events to aggregate
	}

	aggregated := a.aggregateFunc(events)
	return a.recorder.RecordEvent(aggregated)
}

// MetricsBuffer buffers metrics before recording
type MetricsBuffer struct {
	recorder   MetricsRecorder
	buffer     []MetricEvent
	bufferSize int
	flushTimer *time.Timer
	flushAfter time.Duration
	mu         sync.Mutex
}

// NewMetricsBuffer creates a new metrics buffer
func NewMetricsBuffer(recorder MetricsRecorder, bufferSize int, flushAfter time.Duration) *MetricsBuffer {
	mb := &MetricsBuffer{
		recorder:   recorder,
		buffer:     make([]MetricEvent, 0, bufferSize),
		bufferSize: bufferSize,
		flushAfter: flushAfter,
	}

	if flushAfter > 0 {
		mb.resetFlushTimer()
	}

	return mb
}

// Record adds a metric event to the buffer
func (b *MetricsBuffer) Record(event MetricEvent) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.buffer = append(b.buffer, event)

	if len(b.buffer) >= b.bufferSize {
		return b.flush()
	}

	return nil
}

// Flush writes all buffered events to the recorder
func (b *MetricsBuffer) Flush() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.flush()
}

// flush internal method (assumes lock is held)
func (b *MetricsBuffer) flush() error {
	if len(b.buffer) == 0 {
		return nil
	}

	if err := b.recorder.RecordBatch(b.buffer); err != nil {
		return fmt.Errorf("failed to flush buffer: %w", err)
	}

	b.buffer = b.buffer[:0] // Reset buffer
	b.resetFlushTimer()

	return nil
}

// resetFlushTimer resets the flush timer
func (b *MetricsBuffer) resetFlushTimer() {
	if b.flushTimer != nil {
		b.flushTimer.Stop()
	}

	if b.flushAfter > 0 {
		b.flushTimer = time.AfterFunc(b.flushAfter, func() {
			if err := b.Flush(); err != nil {
				log.Log.Error(err, "failed to flush metrics buffer on timer")
			}
		})
	}
}

// Close flushes any remaining events and stops the timer
func (b *MetricsBuffer) Close() error {
	if b.flushTimer != nil {
		b.flushTimer.Stop()
	}

	return b.Flush()
}

// MetricsSerializer handles serialization of metrics
type MetricsSerializer struct {
	format string
}

// NewMetricsSerializer creates a new metrics serializer
func NewMetricsSerializer(format string) *MetricsSerializer {
	return &MetricsSerializer{
		format: format,
	}
}

// Serialize converts metrics to the specified format
func (s *MetricsSerializer) Serialize(events []MetricEvent) ([]byte, error) {
	switch s.format {
	case "json":
		return json.Marshal(events)
	default:
		return nil, fmt.Errorf("unsupported format: %s", s.format)
	}
}

// Deserialize converts data from the specified format to metrics
func (s *MetricsSerializer) Deserialize(data []byte) ([]MetricEvent, error) {
	switch s.format {
	case "json":
		var events []MetricEvent
		if err := json.Unmarshal(data, &events); err != nil {
			return nil, fmt.Errorf("failed to deserialize JSON: %w", err)
		}
		return events, nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", s.format)
	}
}

// MetricsValidator validates metric events
type MetricsValidator struct {
	rules []ValidationRule
}

// ValidationRule defines a validation rule for metrics
type ValidationRule struct {
	Name        string
	Description string
	Validator   func(MetricEvent) error
}

// NewMetricsValidator creates a new metrics validator
func NewMetricsValidator() *MetricsValidator {
	return &MetricsValidator{
		rules: []ValidationRule{
			{
				Name:        "required_fields",
				Description: "Validates that required fields are present",
				Validator: func(event MetricEvent) error {
					if event.Name == "" {
						return fmt.Errorf("metric name is required")
					}
					if event.Type == "" {
						return fmt.Errorf("metric type is required")
					}
					return nil
				},
			},
			{
				Name:        "valid_type",
				Description: "Validates that metric type is supported",
				Validator: func(event MetricEvent) error {
					switch event.Type {
					case MetricTypeCounter, MetricTypeGauge, MetricTypeHistogram, MetricTypeSummary:
						return nil
					default:
						return fmt.Errorf("unsupported metric type: %s", event.Type)
					}
				},
			},
			{
				Name:        "positive_values",
				Description: "Validates that counter values are non-negative",
				Validator: func(event MetricEvent) error {
					if event.Type == MetricTypeCounter && event.Value < 0 {
						return fmt.Errorf("counter values must be non-negative, got: %f", event.Value)
					}
					return nil
				},
			},
		},
	}
}

// AddRule adds a custom validation rule
func (v *MetricsValidator) AddRule(rule ValidationRule) {
	v.rules = append(v.rules, rule)
}

// Validate validates a metric event against all rules
func (v *MetricsValidator) Validate(event MetricEvent) error {
	for _, rule := range v.rules {
		if err := rule.Validator(event); err != nil {
			return fmt.Errorf("validation rule '%s' failed: %w", rule.Name, err)
		}
	}
	return nil
}

// ValidateBatch validates a batch of metric events
func (v *MetricsValidator) ValidateBatch(events []MetricEvent) error {
	for i, event := range events {
		if err := v.Validate(event); err != nil {
			return fmt.Errorf("validation failed for event %d: %w", i, err)
		}
	}
	return nil
}

// MetricsTransformer transforms metric events
type MetricsTransformer struct {
	transformers []TransformFunc
}

// TransformFunc defines a function that transforms a metric event
type TransformFunc func(MetricEvent) MetricEvent

// NewMetricsTransformer creates a new metrics transformer
func NewMetricsTransformer() *MetricsTransformer {
	return &MetricsTransformer{
		transformers: make([]TransformFunc, 0),
	}
}

// AddTransformer adds a transform function
func (t *MetricsTransformer) AddTransformer(transformer TransformFunc) {
	t.transformers = append(t.transformers, transformer)
}

// Transform applies all transformers to a metric event
func (t *MetricsTransformer) Transform(event MetricEvent) MetricEvent {
	result := event
	for _, transformer := range t.transformers {
		result = transformer(result)
	}
	return result
}

// TransformBatch applies transformations to a batch of events
func (t *MetricsTransformer) TransformBatch(events []MetricEvent) []MetricEvent {
	result := make([]MetricEvent, len(events))
	for i, event := range events {
		result[i] = t.Transform(event)
	}
	return result
}

// Common transformer functions

// AddLabelTransformer adds a label to all events
func AddLabelTransformer(key, value string) TransformFunc {
	return func(event MetricEvent) MetricEvent {
		if event.Labels == nil {
			event.Labels = make(map[string]string)
		}
		event.Labels[key] = value
		return event
	}
}

// RenameMetricTransformer renames a metric
func RenameMetricTransformer(oldName, newName string) TransformFunc {
	return func(event MetricEvent) MetricEvent {
		if event.Name == oldName {
			event.Name = newName
		}
		return event
	}
}

// ScaleValueTransformer scales metric values by a factor
func ScaleValueTransformer(factor float64) TransformFunc {
	return func(event MetricEvent) MetricEvent {
		event.Value *= factor
		return event
	}
}

// UnitConversionTransformer converts units with a conversion factor
func UnitConversionTransformer(fromUnit, toUnit string, factor float64) TransformFunc {
	return func(event MetricEvent) MetricEvent {
		if unit, exists := event.Labels["unit"]; exists && unit == fromUnit {
			event.Value *= factor
			event.Labels["unit"] = toUnit
		}
		return event
	}
}

// TimestampTransformer sets the timestamp to current time
func TimestampTransformer() TransformFunc {
	return func(event MetricEvent) MetricEvent {
		event.Timestamp = time.Now()
		return event
	}
}

// ConditionalTransformer applies a transformer only if a condition is met
func ConditionalTransformer(condition func(MetricEvent) bool, transformer TransformFunc) TransformFunc {
	return func(event MetricEvent) MetricEvent {
		if condition(event) {
			return transformer(event)
		}
		return event
	}
}

// LabelFilterTransformer filters events based on label values
func LabelFilterTransformer(key, value string, include bool) TransformFunc {
	return func(event MetricEvent) MetricEvent {
		if eventValue, exists := event.Labels[key]; exists {
			matches := eventValue == value
			if (include && matches) || (!include && !matches) {
				return event
			}
		}
		// Return empty event to indicate filtering
		return MetricEvent{}
	}
}