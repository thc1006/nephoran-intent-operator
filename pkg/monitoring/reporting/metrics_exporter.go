// Copyright 2024 Nephoran Intent Operator Authors.

//

// Licensed under the Apache License, Version 2.0 (the "License");.

// you may not use this file except in compliance with the License.

package reporting

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
)

// ExporterConfig contains configuration for metrics export.

type ExporterConfig struct {
	Prometheus PrometheusConfig `yaml:"prometheus"`

	InfluxDB InfluxDBConfig `yaml:"influxdb"`

	DataDog DataDogConfig `yaml:"datadog"`

	NewRelic NewRelicConfig `yaml:"newrelic"`

	Custom []CustomConfig `yaml:"custom"`

	Streaming StreamingConfig `yaml:"streaming"`

	Batch BatchConfig `yaml:"batch"`

	Transform TransformConfig `yaml:"transform"`

	Compression CompressionConfig `yaml:"compression"`
}

// PrometheusConfig contains Prometheus-specific configuration.

type PrometheusConfig struct {
	Enabled bool `yaml:"enabled"`

	URL string `yaml:"url"`

	RemoteWrite string `yaml:"remote_write"`

	Format string `yaml:"format"` // openmetrics, prometheus
}

// InfluxDBConfig contains InfluxDB-specific configuration.

type InfluxDBConfig struct {
	Enabled bool `yaml:"enabled"`

	URL string `yaml:"url"`

	Database string `yaml:"database"`

	Measurement string `yaml:"measurement"`

	Precision string `yaml:"precision"`

	RetentionPolicy string `yaml:"retention_policy"`
}

// DataDogConfig contains DataDog-specific configuration.

type DataDogConfig struct {
	Enabled bool `yaml:"enabled"`

	APIKey string `yaml:"api_key"`

	AppKey string `yaml:"app_key"`

	Site string `yaml:"site"`

	Tags []string `yaml:"tags"`
}

// NewRelicConfig contains New Relic-specific configuration.

type NewRelicConfig struct {
	Enabled bool `yaml:"enabled"`

	LicenseKey string `yaml:"license_key"`

	AccountID string `yaml:"account_id"`

	InsightsKey string `yaml:"insights_key"`
}

// CustomConfig contains configuration for custom exporters.

type CustomConfig struct {
	Name string `yaml:"name"`

	URL string `yaml:"url"`

	Method string `yaml:"method"`

	Headers map[string]string `yaml:"headers"`

	Format string `yaml:"format"`

	Template string `yaml:"template"`
}

// StreamingConfig contains streaming configuration.

type StreamingConfig struct {
	Enabled bool `yaml:"enabled"`

	BatchSize int `yaml:"batch_size"`

	FlushInterval time.Duration `yaml:"flush_interval"`

	BufferSize int `yaml:"buffer_size"`

	Workers int `yaml:"workers"`
}

// BatchConfig contains batch export configuration.

type BatchConfig struct {
	Enabled bool `yaml:"enabled"`

	Interval time.Duration `yaml:"interval"`

	BatchSize int `yaml:"batch_size"`

	Retention time.Duration `yaml:"retention"`
}

// TransformConfig contains data transformation configuration.

type TransformConfig struct {
	Aggregations []AggregationConfig `yaml:"aggregations"`

	Filters []FilterConfig `yaml:"filters"`

	Mappings []MappingConfig `yaml:"mappings"`
}

// AggregationConfig contains aggregation configuration.

type AggregationConfig struct {
	Name string `yaml:"name"`

	Metrics []string `yaml:"metrics"`

	Function string `yaml:"function"` // sum, avg, max, min, count

	Window time.Duration `yaml:"window"`

	GroupBy []string `yaml:"group_by"`
}

// FilterConfig contains filter configuration.

type FilterConfig struct {
	Name string `yaml:"name"`

	Metrics []string `yaml:"metrics"`

	Labels map[string]string `yaml:"labels"`

	Condition string `yaml:"condition"` // include, exclude
}

// MappingConfig contains metric mapping configuration.

type MappingConfig struct {
	From string `yaml:"from"`

	To string `yaml:"to"`

	Labels map[string]string `yaml:"labels"`
}

// CompressionConfig contains compression configuration.

type CompressionConfig struct {
	Enabled bool `yaml:"enabled"`

	Algorithm string `yaml:"algorithm"` // gzip, snappy, lz4

	Level int `yaml:"level"`

	MinSize int `yaml:"min_size"`
}

// MetricPoint represents a single metric data point.

type MetricPoint struct {
	Name string `json:"name"`

	Value float64 `json:"value"`

	Timestamp time.Time `json:"timestamp"`

	Labels map[string]string `json:"labels"`

	Type string `json:"type"` // gauge, counter, histogram

	Unit string `json:"unit,omitempty"`

	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// MetricBatch represents a batch of metrics.

type MetricBatch struct {
	ID string `json:"id"`

	Timestamp time.Time `json:"timestamp"`

	Metrics []MetricPoint `json:"metrics"`

	Source string `json:"source"`

	Version string `json:"version"`
}

// ExportResult represents the result of an export operation.

type ExportResult struct {
	Exporter string `json:"exporter"`

	Success bool `json:"success"`

	MetricCount int `json:"metric_count"`

	BytesExported int64 `json:"bytes_exported"`

	Duration time.Duration `json:"duration"`

	Error string `json:"error,omitempty"`

	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// StreamMetric represents a streaming metric.

type StreamMetric struct {
	Point MetricPoint `json:"point"`

	Targets []string `json:"targets"`

	Priority int `json:"priority"`
}

// MetricsExporter manages metrics export to various systems.

type MetricsExporter struct {
	config ExporterConfig

	promClient v1.API

	httpClient *http.Client

	streamCh chan StreamMetric

	batchBuffer []MetricPoint

	exporters map[string]Exporter

	aggregations map[string]*Aggregator

	logger *logrus.Logger

	mu sync.RWMutex

	stopCh chan struct{}

	wg sync.WaitGroup
}

// Exporter interface for different export backends.

type Exporter interface {
	Export(ctx context.Context, batch MetricBatch) (*ExportResult, error)

	Name() string

	HealthCheck(ctx context.Context) error
}

// Aggregator handles metric aggregation.

type Aggregator struct {
	config AggregationConfig

	buffer []MetricPoint

	lastFlush time.Time

	mu sync.Mutex
}

// NewMetricsExporter creates a new metrics exporter.

func NewMetricsExporter(config ExporterConfig, promClient v1.API, logger *logrus.Logger) *MetricsExporter {
	me := &MetricsExporter{
		config: config,

		promClient: promClient,

		httpClient: &http.Client{Timeout: 30 * time.Second},

		streamCh: make(chan StreamMetric, config.Streaming.BufferSize),

		batchBuffer: make([]MetricPoint, 0, config.Batch.BatchSize),

		exporters: make(map[string]Exporter),

		aggregations: make(map[string]*Aggregator),

		logger: logger,

		stopCh: make(chan struct{}),
	}

	// Initialize exporters.

	me.initializeExporters()

	// Initialize aggregators.

	me.initializeAggregators()

	return me
}

// Start starts the metrics exporter.

func (me *MetricsExporter) Start(ctx context.Context) error {
	me.logger.Info("Starting Metrics Exporter")

	// Start streaming workers.

	if me.config.Streaming.Enabled {
		for i := range me.config.Streaming.Workers {

			me.wg.Add(1)

			go me.streamingWorker(ctx, i)

		}
	}

	// Start batch export loop.

	if me.config.Batch.Enabled {

		me.wg.Add(1)

		go me.batchExportLoop(ctx)

	}

	// Start aggregation loops.

	for name, aggregator := range me.aggregations {

		me.wg.Add(1)

		go me.aggregationLoop(ctx, name, aggregator)

	}

	return nil
}

// Stop stops the metrics exporter.

func (me *MetricsExporter) Stop() {
	me.logger.Info("Stopping Metrics Exporter")

	close(me.stopCh)

	me.wg.Wait()
}

// StreamMetric streams a single metric to configured exporters.

func (me *MetricsExporter) StreamMetric(point MetricPoint, targets []string) error {
	if !me.config.Streaming.Enabled {
		return fmt.Errorf("streaming is not enabled")
	}

	metric := StreamMetric{
		Point: point,

		Targets: targets,

		Priority: 1,
	}

	select {

	case me.streamCh <- metric:

		return nil

	default:

		return fmt.Errorf("stream buffer full")

	}
}

// ExportBatch exports a batch of metrics.

func (me *MetricsExporter) ExportBatch(ctx context.Context, metrics []MetricPoint, targets []string) ([]*ExportResult, error) {
	batch := MetricBatch{
		ID: fmt.Sprintf("batch-%d", time.Now().Unix()),

		Timestamp: time.Now(),

		Metrics: metrics,

		Source: "nephoran-intent-operator",

		Version: "1.0.0",
	}

	results := make([]*ExportResult, 0)

	for _, target := range targets {

		exporter, exists := me.exporters[target]

		if !exists {

			me.logger.WithField("target", target).Warn("Export target not found")

			continue

		}

		result, err := exporter.Export(ctx, batch)
		if err != nil {

			me.logger.WithError(err).WithField("target", target).Error("Export failed")

			result = &ExportResult{
				Exporter: target,

				Success: false,

				Error: err.Error(),
			}

		}

		results = append(results, result)

	}

	return results, nil
}

// QueryAndExport queries Prometheus and exports the results.

func (me *MetricsExporter) QueryAndExport(ctx context.Context, query string, exportTargets []string) error {
	result, warnings, err := me.promClient.Query(ctx, query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to query Prometheus: %w", err)
	}

	if len(warnings) > 0 {
		me.logger.WithField("warnings", warnings).Warn("Query returned warnings")
	}

	metrics := me.convertPrometheusResult(result)

	if len(metrics) == 0 {
		return fmt.Errorf("no metrics returned from query")
	}

	_, err = me.ExportBatch(ctx, metrics, exportTargets)

	return err
}

// GetExporterStatus returns the status of all exporters.

func (me *MetricsExporter) GetExporterStatus(ctx context.Context) map[string]bool {
	me.mu.RLock()

	defer me.mu.RUnlock()

	status := make(map[string]bool)

	for name, exporter := range me.exporters {

		err := exporter.HealthCheck(ctx)

		status[name] = err == nil

	}

	return status
}

// initializeExporters initializes all configured exporters.

func (me *MetricsExporter) initializeExporters() {
	// Prometheus exporter.

	if me.config.Prometheus.Enabled {
		me.exporters["prometheus"] = &PrometheusExporter{
			config: me.config.Prometheus,

			client: me.httpClient,

			logger: me.logger,
		}
	}

	// InfluxDB exporter.

	if me.config.InfluxDB.Enabled {
		me.exporters["influxdb"] = &InfluxDBExporter{
			config: me.config.InfluxDB,

			client: me.httpClient,

			logger: me.logger,
		}
	}

	// DataDog exporter.

	if me.config.DataDog.Enabled {
		me.exporters["datadog"] = &DataDogExporter{
			config: me.config.DataDog,

			client: me.httpClient,

			logger: me.logger,
		}
	}

	// New Relic exporter.

	if me.config.NewRelic.Enabled {
		me.exporters["newrelic"] = &NewRelicExporter{
			config: me.config.NewRelic,

			client: me.httpClient,

			logger: me.logger,
		}
	}

	// Custom exporters.

	for _, customConfig := range me.config.Custom {
		me.exporters[customConfig.Name] = &CustomExporter{
			config: customConfig,

			client: me.httpClient,

			logger: me.logger,
		}
	}
}

// initializeAggregators initializes metric aggregators.

func (me *MetricsExporter) initializeAggregators() {
	for _, aggConfig := range me.config.Transform.Aggregations {
		me.aggregations[aggConfig.Name] = &Aggregator{
			config: aggConfig,

			buffer: make([]MetricPoint, 0),

			lastFlush: time.Now(),
		}
	}
}

// streamingWorker processes streaming metrics.

func (me *MetricsExporter) streamingWorker(ctx context.Context, workerID int) {
	defer me.wg.Done()

	logger := me.logger.WithField("worker_id", workerID)

	logger.Info("Starting streaming worker")

	batch := make([]StreamMetric, 0, me.config.Streaming.BatchSize)

	flushTimer := time.NewTimer(me.config.Streaming.FlushInterval)

	defer flushTimer.Stop()

	for {
		select {

		case <-ctx.Done():

			return

		case <-me.stopCh:

			return

		case metric := <-me.streamCh:

			batch = append(batch, metric)

			if len(batch) >= me.config.Streaming.BatchSize {

				me.flushStreamBatch(ctx, batch)

				batch = batch[:0]

				flushTimer.Reset(me.config.Streaming.FlushInterval)

			}

		case <-flushTimer.C:

			if len(batch) > 0 {

				me.flushStreamBatch(ctx, batch)

				batch = batch[:0]

			}

			flushTimer.Reset(me.config.Streaming.FlushInterval)

		}
	}
}

// flushStreamBatch flushes a batch of streaming metrics.

func (me *MetricsExporter) flushStreamBatch(ctx context.Context, batch []StreamMetric) {
	if len(batch) == 0 {
		return
	}

	// Group metrics by target.

	targetGroups := make(map[string][]MetricPoint)

	for _, metric := range batch {
		for _, target := range metric.Targets {

			if _, exists := targetGroups[target]; !exists {
				targetGroups[target] = make([]MetricPoint, 0)
			}

			targetGroups[target] = append(targetGroups[target], metric.Point)

		}
	}

	// Export to each target.

	for target, metrics := range targetGroups {
		if exporter, exists := me.exporters[target]; exists {

			metricBatch := MetricBatch{
				ID: fmt.Sprintf("stream-%d", time.Now().UnixNano()),

				Timestamp: time.Now(),

				Metrics: metrics,

				Source: "nephoran-intent-operator",

				Version: "1.0.0",
			}

			_, err := exporter.Export(ctx, metricBatch)
			if err != nil {
				me.logger.WithError(err).WithField("target", target).Error("Failed to export streaming batch")
			}

		}
	}
}

// batchExportLoop handles periodic batch exports.

func (me *MetricsExporter) batchExportLoop(ctx context.Context) {
	defer me.wg.Done()

	ticker := time.NewTicker(me.config.Batch.Interval)

	defer ticker.Stop()

	me.logger.Info("Starting batch export loop")

	for {
		select {

		case <-ctx.Done():

			return

		case <-me.stopCh:

			return

		case <-ticker.C:

			me.performBatchExport(ctx)

		}
	}
}

// performBatchExport performs a batch export.

func (me *MetricsExporter) performBatchExport(ctx context.Context) {
	me.mu.Lock()

	if len(me.batchBuffer) == 0 {

		me.mu.Unlock()

		return

	}

	// Copy buffer and reset.

	metrics := make([]MetricPoint, len(me.batchBuffer))

	copy(metrics, me.batchBuffer)

	me.batchBuffer = me.batchBuffer[:0]

	me.mu.Unlock()

	// Export to all enabled targets.

	targets := make([]string, 0, len(me.exporters))

	for name := range me.exporters {
		targets = append(targets, name)
	}

	results, err := me.ExportBatch(ctx, metrics, targets)
	if err != nil {

		me.logger.WithError(err).Error("Batch export failed")

		return

	}

	// Log results.

	for _, result := range results {
		if result.Success {
			me.logger.WithFields(logrus.Fields{
				"exporter": result.Exporter,

				"metric_count": result.MetricCount,

				"bytes_exported": result.BytesExported,

				"duration": result.Duration,
			}).Info("Batch export successful")
		} else {
			me.logger.WithFields(logrus.Fields{
				"exporter": result.Exporter,

				"error": result.Error,
			}).Error("Batch export failed")
		}
	}
}

// aggregationLoop handles metric aggregation.

func (me *MetricsExporter) aggregationLoop(ctx context.Context, name string, aggregator *Aggregator) {
	defer me.wg.Done()

	ticker := time.NewTicker(aggregator.config.Window)

	defer ticker.Stop()

	me.logger.WithField("aggregator", name).Info("Starting aggregation loop")

	for {
		select {

		case <-ctx.Done():

			return

		case <-me.stopCh:

			return

		case <-ticker.C:

			me.performAggregation(ctx, name, aggregator)

		}
	}
}

// performAggregation performs metric aggregation.

func (me *MetricsExporter) performAggregation(ctx context.Context, name string, aggregator *Aggregator) {
	aggregator.mu.Lock()

	defer aggregator.mu.Unlock()

	if len(aggregator.buffer) == 0 {
		return
	}

	// Group metrics by labels.

	groups := make(map[string][]MetricPoint)

	for _, point := range aggregator.buffer {

		key := me.generateGroupKey(point.Labels, aggregator.config.GroupBy)

		if _, exists := groups[key]; !exists {
			groups[key] = make([]MetricPoint, 0)
		}

		groups[key] = append(groups[key], point)

	}

	// Perform aggregation for each group.

	aggregatedMetrics := make([]MetricPoint, 0)

	for _, points := range groups {

		aggregated := me.aggregatePoints(points, aggregator.config.Function)

		if aggregated != nil {

			aggregated.Name = fmt.Sprintf("%s_%s", name, aggregator.config.Function)

			aggregated.Timestamp = time.Now()

			aggregatedMetrics = append(aggregatedMetrics, *aggregated)

		}

	}

	// Clear buffer.

	aggregator.buffer = aggregator.buffer[:0]

	aggregator.lastFlush = time.Now()

	// Export aggregated metrics.

	if len(aggregatedMetrics) > 0 {

		me.mu.Lock()

		me.batchBuffer = append(me.batchBuffer, aggregatedMetrics...)

		me.mu.Unlock()

	}
}

// generateGroupKey generates a key for grouping metrics.

func (me *MetricsExporter) generateGroupKey(labels map[string]string, groupBy []string) string {
	if len(groupBy) == 0 {
		return "all"
	}

	parts := make([]string, len(groupBy))

	for i, key := range groupBy {
		if value, exists := labels[key]; exists {
			parts[i] = fmt.Sprintf("%s=%s", key, value)
		} else {
			parts[i] = fmt.Sprintf("%s=", key)
		}
	}

	return strings.Join(parts, ",")
}

// aggregatePoints aggregates multiple points using the specified function.

func (me *MetricsExporter) aggregatePoints(points []MetricPoint, function string) *MetricPoint {
	if len(points) == 0 {
		return nil
	}

	var result float64

	switch function {

	case "sum":

		for _, point := range points {
			result += point.Value
		}

	case "avg":

		sum := 0.0

		for _, point := range points {
			sum += point.Value
		}

		result = sum / float64(len(points))

	case "max":

		result = math.Inf(-1)

		for _, point := range points {
			if point.Value > result {
				result = point.Value
			}
		}

	case "min":

		result = math.Inf(1)

		for _, point := range points {
			if point.Value < result {
				result = point.Value
			}
		}

	case "count":

		result = float64(len(points))

	default:

		return nil

	}

	// Use labels from first point as base.

	basePoint := points[0]

	return &MetricPoint{
		Name: basePoint.Name,

		Value: result,

		Timestamp: time.Now(),

		Labels: basePoint.Labels,

		Type: basePoint.Type,

		Unit: basePoint.Unit,

		Metadata: basePoint.Metadata,
	}
}

// convertPrometheusResult converts Prometheus query result to MetricPoints.

func (me *MetricsExporter) convertPrometheusResult(result model.Value) []MetricPoint {
	metrics := make([]MetricPoint, 0)

	switch result.Type() {

	case model.ValVector:

		vector := result.(model.Vector)

		for _, sample := range vector {

			labels := make(map[string]string)

			for k, v := range sample.Metric {
				if k != "__name__" {
					labels[string(k)] = string(v)
				}
			}

			metric := MetricPoint{
				Name: string(sample.Metric["__name__"]),

				Value: float64(sample.Value),

				Timestamp: sample.Timestamp.Time(),

				Labels: labels,

				Type: "gauge",
			}

			metrics = append(metrics, metric)

		}

	case model.ValScalar:

		scalar := result.(*model.Scalar)

		metric := MetricPoint{
			Name: "scalar_value",

			Value: float64(scalar.Value),

			Timestamp: scalar.Timestamp.Time(),

			Labels: make(map[string]string),

			Type: "gauge",
		}

		metrics = append(metrics, metric)

	case model.ValMatrix:

		matrix := result.(model.Matrix)

		for _, sampleStream := range matrix {

			labels := make(map[string]string)

			for k, v := range sampleStream.Metric {
				if k != "__name__" {
					labels[string(k)] = string(v)
				}
			}

			for _, value := range sampleStream.Values {

				metric := MetricPoint{
					Name: string(sampleStream.Metric["__name__"]),

					Value: float64(value.Value),

					Timestamp: value.Timestamp.Time(),

					Labels: labels,

					Type: "gauge",
				}

				metrics = append(metrics, metric)

			}

		}

	}

	return metrics
}

// PrometheusExporter exports metrics in Prometheus format.

type PrometheusExporter struct {
	config PrometheusConfig

	client *http.Client

	logger *logrus.Logger
}

// Export exports metrics to Prometheus.

func (pe *PrometheusExporter) Export(ctx context.Context, batch MetricBatch) (*ExportResult, error) {
	startTime := time.Now()

	// Convert to Prometheus format.

	data, err := pe.convertToPrometheusFormat(batch)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to Prometheus format: %w", err)
	}

	// Compress if enabled.

	var body io.Reader = bytes.NewReader(data)

	contentEncoding := ""

	if len(data) > 1000 { // Compress if data is larger than 1KB

		var buf bytes.Buffer

		gz := gzip.NewWriter(&buf)

		if _, err := gz.Write(data); err != nil {
			pe.logger.Error("Failed to write data to gzip writer", "error", err)
			// Fall back to uncompressed data
			body = bytes.NewReader(data)
		} else {
			if err := gz.Close(); err != nil {
				pe.logger.Error("Failed to close gzip writer", "error", err)
				// Fall back to uncompressed data
				body = bytes.NewReader(data)
			} else {
				body = &buf
				contentEncoding = "gzip"
			}
		}

	}

	// Send to remote write endpoint.

	req, err := http.NewRequestWithContext(ctx, "POST", pe.config.RemoteWrite, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-protobuf")

	if contentEncoding != "" {
		req.Header.Set("Content-Encoding", contentEncoding)
	}

	resp, err := pe.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	defer resp.Body.Close() // #nosec G307 - Error handled in defer

	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	return &ExportResult{
		Exporter: "prometheus",

		Success: success,

		MetricCount: len(batch.Metrics),

		BytesExported: int64(len(data)),

		Duration: time.Since(startTime),
	}, nil
}

// Name returns the exporter name.

func (pe *PrometheusExporter) Name() string {
	return "prometheus"
}

// HealthCheck checks if the Prometheus endpoint is healthy.

func (pe *PrometheusExporter) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", pe.config.URL+"/-/healthy", http.NoBody)
	if err != nil {
		return err
	}

	resp, err := pe.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close() // #nosec G307 - Error handled in defer

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unhealthy status: %d", resp.StatusCode)
	}

	return nil
}

// convertToPrometheusFormat converts metrics to Prometheus format.

func (pe *PrometheusExporter) convertToPrometheusFormat(batch MetricBatch) ([]byte, error) {
	var buf bytes.Buffer

	for _, metric := range batch.Metrics {

		// Write metric in Prometheus text format.

		line := metric.Name

		if len(metric.Labels) > 0 {

			labels := make([]string, 0, len(metric.Labels))

			for k, v := range metric.Labels {
				labels = append(labels, fmt.Sprintf(`%s="%s"`, k, v))
			}

			sort.Strings(labels)

			line += "{" + strings.Join(labels, ",") + "}"

		}

		line += fmt.Sprintf(" %g %d\n", metric.Value, metric.Timestamp.UnixMilli())

		buf.WriteString(line)

	}

	return buf.Bytes(), nil
}

// InfluxDBExporter exports metrics to InfluxDB.

type InfluxDBExporter struct {
	config InfluxDBConfig

	client *http.Client

	logger *logrus.Logger
}

// Export exports metrics to InfluxDB.

func (ie *InfluxDBExporter) Export(ctx context.Context, batch MetricBatch) (*ExportResult, error) {
	startTime := time.Now()

	// Convert to InfluxDB line protocol.

	data := ie.convertToInfluxFormat(batch)

	// Build URL with query parameters.

	url := fmt.Sprintf("%s/write?db=%s", ie.config.URL, ie.config.Database)

	if ie.config.Precision != "" {
		url += "&precision=" + ie.config.Precision
	}

	if ie.config.RetentionPolicy != "" {
		url += "&rp=" + ie.config.RetentionPolicy
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "text/plain")

	resp, err := ie.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close() // #nosec G307 - Error handled in defer

	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	return &ExportResult{
		Exporter: "influxdb",

		Success: success,

		MetricCount: len(batch.Metrics),

		BytesExported: int64(len(data)),

		Duration: time.Since(startTime),
	}, nil
}

// Name returns the exporter name.

func (ie *InfluxDBExporter) Name() string {
	return "influxdb"
}

// HealthCheck checks if InfluxDB is healthy.

func (ie *InfluxDBExporter) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", ie.config.URL+"/ping", http.NoBody)
	if err != nil {
		return err
	}

	resp, err := ie.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close() // #nosec G307 - Error handled in defer

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unhealthy status: %d", resp.StatusCode)
	}

	return nil
}

// convertToInfluxFormat converts metrics to InfluxDB line protocol.

func (ie *InfluxDBExporter) convertToInfluxFormat(batch MetricBatch) []byte {
	var buf bytes.Buffer

	for _, metric := range batch.Metrics {

		line := ie.config.Measurement

		if line == "" {
			line = metric.Name
		}

		// Add tags.

		if len(metric.Labels) > 0 {

			tags := make([]string, 0, len(metric.Labels))

			for k, v := range metric.Labels {
				tags = append(tags, fmt.Sprintf("%s=%s", k, v))
			}

			sort.Strings(tags)

			line += "," + strings.Join(tags, ",")

		}

		// Add fields.

		line += fmt.Sprintf(" value=%g", metric.Value)

		// Add timestamp.

		line += fmt.Sprintf(" %d\n", metric.Timestamp.UnixNano())

		buf.WriteString(line)

	}

	return buf.Bytes()
}

// DataDogExporter exports metrics to DataDog.

type DataDogExporter struct {
	config DataDogConfig

	client *http.Client

	logger *logrus.Logger
}

// Export exports metrics to DataDog.

func (de *DataDogExporter) Export(ctx context.Context, batch MetricBatch) (*ExportResult, error) {
	startTime := time.Now()

	// Convert to DataDog format.

	payload := de.convertToDataDogFormat(batch)

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://api.%s/api/v1/series", de.config.Site)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("DD-API-KEY", de.config.APIKey)

	if de.config.AppKey != "" {
		req.Header.Set("DD-APPLICATION-KEY", de.config.AppKey)
	}

	resp, err := de.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close() // #nosec G307 - Error handled in defer

	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	return &ExportResult{
		Exporter: "datadog",

		Success: success,

		MetricCount: len(batch.Metrics),

		BytesExported: int64(len(data)),

		Duration: time.Since(startTime),
	}, nil
}

// Name returns the exporter name.

func (de *DataDogExporter) Name() string {
	return "datadog"
}

// HealthCheck checks if DataDog API is accessible.

func (de *DataDogExporter) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("https://api.%s/api/v1/validate", de.config.Site)

	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return err
	}

	req.Header.Set("DD-API-KEY", de.config.APIKey)

	resp, err := de.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close() // #nosec G307 - Error handled in defer

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unhealthy status: %d", resp.StatusCode)
	}

	return nil
}

// convertToDataDogFormat converts metrics to DataDog format.

func (de *DataDogExporter) convertToDataDogFormat(batch MetricBatch) map[string]interface{} {
	series := make([]map[string]interface{}, len(batch.Metrics))

	for i, metric := range batch.Metrics {

		tags := make([]string, 0, len(metric.Labels)+len(de.config.Tags))

		// Add configured tags.

		tags = append(tags, de.config.Tags...)

		// Add metric labels as tags.

		for k, v := range metric.Labels {
			tags = append(tags, fmt.Sprintf("%s:%s", k, v))
		}

		points := [][]float64{{float64(metric.Timestamp.Unix()), metric.Value}}

		series[i] = map[string]interface{}{
			"metric": metric.Name,
			"points": points,
			"tags":   tags,
		}
	}

	return map[string]interface{}{
		"series": series,
	}
}

// NewRelicExporter exports metrics to New Relic.

type NewRelicExporter struct {
	config NewRelicConfig

	client *http.Client

	logger *logrus.Logger
}

// Export exports metrics to New Relic.

func (nr *NewRelicExporter) Export(ctx context.Context, batch MetricBatch) (*ExportResult, error) {
	startTime := time.Now()

	// Convert to New Relic format.

	payload := nr.convertToNewRelicFormat(batch)

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := "https://insights-collector.newrelic.com/v1/accounts/" + nr.config.AccountID + "/events"

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("X-Insert-Key", nr.config.InsightsKey)

	resp, err := nr.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close() // #nosec G307 - Error handled in defer

	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	return &ExportResult{
		Exporter: "newrelic",

		Success: success,

		MetricCount: len(batch.Metrics),

		BytesExported: int64(len(data)),

		Duration: time.Since(startTime),
	}, nil
}

// Name returns the exporter name.

func (nr *NewRelicExporter) Name() string {
	return "newrelic"
}

// HealthCheck checks if New Relic API is accessible.

func (nr *NewRelicExporter) HealthCheck(ctx context.Context) error {
	// New Relic doesn't have a dedicated health endpoint.

	// We'll just validate the configuration.

	if nr.config.AccountID == "" || nr.config.InsightsKey == "" {
		return fmt.Errorf("invalid New Relic configuration")
	}

	return nil
}

// convertToNewRelicFormat converts metrics to New Relic format.

func (nr *NewRelicExporter) convertToNewRelicFormat(batch MetricBatch) []map[string]interface{} {
	events := make([]map[string]interface{}, len(batch.Metrics))

	for i, metric := range batch.Metrics {

		event := map[string]interface{}{
			"eventType": "NephoranMetric",
			"timestamp": metric.Timestamp.Unix(),
			"metric":    metric.Name,
			"value":     metric.Value,
		}

		// Add labels as attributes.
		for k, v := range metric.Labels {
			event[k] = v
		}

		events[i] = event

	}

	return events
}

// CustomExporter exports metrics to custom endpoints.

type CustomExporter struct {
	config CustomConfig

	client *http.Client

	logger *logrus.Logger
}

// Export exports metrics to custom endpoint.

func (ce *CustomExporter) Export(ctx context.Context, batch MetricBatch) (*ExportResult, error) {
	startTime := time.Now()

	var data []byte

	var err error

	switch ce.config.Format {

	case "json":

		data, err = json.Marshal(batch)

	case "csv":

		data, err = ce.convertToCSV(batch)

	default:

		data, err = json.Marshal(batch)

	}

	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, ce.config.Method, ce.config.URL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	// Add custom headers.

	for k, v := range ce.config.Headers {
		req.Header.Set(k, v)
	}

	resp, err := ce.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close() // #nosec G307 - Error handled in defer

	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	return &ExportResult{
		Exporter: ce.config.Name,

		Success: success,

		MetricCount: len(batch.Metrics),

		BytesExported: int64(len(data)),

		Duration: time.Since(startTime),
	}, nil
}

// Name returns the exporter name.

func (ce *CustomExporter) Name() string {
	return ce.config.Name
}

// HealthCheck checks if the custom endpoint is accessible.

func (ce *CustomExporter) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "HEAD", ce.config.URL, http.NoBody)
	if err != nil {
		return err
	}

	resp, err := ce.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close() // #nosec G307 - Error handled in defer

	if resp.StatusCode >= 400 {
		return fmt.Errorf("unhealthy status: %d", resp.StatusCode)
	}

	return nil
}

// convertToCSV converts metrics to CSV format.

func (ce *CustomExporter) convertToCSV(batch MetricBatch) ([]byte, error) {
	var buf bytes.Buffer

	writer := csv.NewWriter(&buf)

	defer writer.Flush()

	// Write header.

	header := []string{"name", "value", "timestamp", "type", "unit"}

	if err := writer.Write(header); err != nil {
		return nil, err
	}

	// Write data.

	for _, metric := range batch.Metrics {

		record := []string{
			metric.Name,

			strconv.FormatFloat(metric.Value, 'f', -1, 64),

			metric.Timestamp.Format(time.RFC3339),

			metric.Type,

			metric.Unit,
		}

		// Add label columns.

		for k, v := range metric.Labels {
			record = append(record, fmt.Sprintf("%s=%s", k, v))
		}

		if err := writer.Write(record); err != nil {
			return nil, err
		}

	}

	return buf.Bytes(), nil
}
