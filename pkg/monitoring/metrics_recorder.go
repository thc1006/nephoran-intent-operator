// Package monitoring - Metrics recording and storage implementation
package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// MetricsRecorder records and stores metrics data with time-series capabilities
type MetricsRecorder struct {
	mu            sync.RWMutex
	storage       MetricsStorage
	buffer        []MetricsRecord
	bufferSize    int
	flushInterval time.Duration
	registry      prometheus.Registerer
	stopCh        chan struct{}

	// Metrics
	recordsWritten   prometheus.Counter
	writeErrors      prometheus.Counter
	writeDuration    prometheus.Histogram
	bufferSize_gauge prometheus.Gauge
}

// MetricsStorage defines interface for metrics storage backends
type MetricsStorage interface {
	// WriteMetrics writes metrics to storage
	WriteMetrics(ctx context.Context, records []MetricsRecord) error

	// ReadMetrics reads metrics from storage
	ReadMetrics(ctx context.Context, query MetricsQuery) ([]MetricsRecord, error)

	// DeleteMetrics deletes metrics from storage
	DeleteMetrics(ctx context.Context, filter MetricsFilter) error

	// GetMetricNames returns all available metric names
	GetMetricNames(ctx context.Context) ([]string, error)

	// Close closes the storage connection
	Close() error
}

// MetricsRecord represents a single metrics record with metadata
type MetricsRecord struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Namespace string                 `json:"namespace"`
	Metric    string                 `json:"metric"`
	Value     float64                `json:"value"`
	Labels    map[string]string      `json:"labels"`
	Tags      []string               `json:"tags,omitempty"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}

// MetricsQuery represents a query for retrieving metrics
type MetricsQuery struct {
	Sources   []string          `json:"sources,omitempty"`
	Metrics   []string          `json:"metrics,omitempty"`
	StartTime time.Time         `json:"startTime"`
	EndTime   time.Time         `json:"endTime"`
	Labels    map[string]string `json:"labels,omitempty"`
	Limit     int               `json:"limit,omitempty"`
	Offset    int               `json:"offset,omitempty"`
	OrderBy   string            `json:"orderBy,omitempty"` // timestamp, value, source
	Ascending bool              `json:"ascending"`
}

// MetricsFilter represents a filter for deleting metrics
type MetricsFilter struct {
	Sources   []string          `json:"sources,omitempty"`
	Metrics   []string          `json:"metrics,omitempty"`
	OlderThan time.Time         `json:"olderThan,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// NewMetricsRecorder creates a new metrics recorder
func NewMetricsRecorder(storage MetricsStorage, bufferSize int, flushInterval time.Duration, registry prometheus.Registerer) *MetricsRecorder {
	mr := &MetricsRecorder{
		storage:       storage,
		buffer:        make([]MetricsRecord, 0, bufferSize),
		bufferSize:    bufferSize,
		flushInterval: flushInterval,
		registry:      registry,
		stopCh:        make(chan struct{}),
	}

	mr.initMetrics()
	return mr
}

// initMetrics initializes Prometheus metrics for the recorder
func (mr *MetricsRecorder) initMetrics() {
	mr.recordsWritten = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "oran_metrics_records_written_total",
		Help: "Total number of metrics records written to storage",
	})

	mr.writeErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "oran_metrics_write_errors_total",
		Help: "Total number of metrics write errors",
	})

	mr.writeDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "oran_metrics_write_duration_seconds",
		Help:    "Duration of metrics write operations in seconds",
		Buckets: prometheus.DefBuckets,
	})

	mr.bufferSize_gauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "oran_metrics_buffer_size",
		Help: "Current size of metrics buffer",
	})

	if mr.registry != nil {
		mr.registry.MustRegister(mr.recordsWritten)
		mr.registry.MustRegister(mr.writeErrors)
		mr.registry.MustRegister(mr.writeDuration)
		mr.registry.MustRegister(mr.bufferSize_gauge)
	}
}

// Start begins metrics recording with periodic flushing
func (mr *MetricsRecorder) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("Starting metrics recorder", "bufferSize", mr.bufferSize, "flushInterval", mr.flushInterval)

	// Start flush loop
	go mr.flushLoop(ctx)

	return nil
}

// Stop gracefully stops metrics recording
func (mr *MetricsRecorder) Stop(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("Stopping metrics recorder")

	// Signal stop
	close(mr.stopCh)

	// Flush remaining buffer
	if err := mr.flushBuffer(ctx); err != nil {
		logger.Error(err, "Failed to flush final buffer")
		return err
	}

	// Close storage
	if err := mr.storage.Close(); err != nil {
		logger.Error(err, "Failed to close storage")
		return err
	}

	return nil
}

// RecordMetrics records metrics data
func (mr *MetricsRecorder) RecordMetrics(ctx context.Context, data *MetricsData) error {
	if data == nil {
		return fmt.Errorf("metrics data cannot be nil")
	}

	logger := log.FromContext(ctx)
	logger.V(2).Info("Recording metrics", "source", data.Source, "metricsCount", len(data.Metrics))

	mr.mu.Lock()
	defer mr.mu.Unlock()

	// Convert MetricsData to MetricsRecords
	records := mr.convertMetricsData(data)

	// Add to buffer
	for _, record := range records {
		mr.buffer = append(mr.buffer, record)
		mr.bufferSize_gauge.Set(float64(len(mr.buffer)))

		// Check if buffer is full
		if len(mr.buffer) >= mr.bufferSize {
			// Flush buffer asynchronously
			go func(bufferedRecords []MetricsRecord) {
				if err := mr.writeToStorage(ctx, bufferedRecords); err != nil {
					logger.Error(err, "Failed to flush full buffer")
				}
			}(append([]MetricsRecord(nil), mr.buffer...)) // Copy buffer

			// Clear buffer
			mr.buffer = mr.buffer[:0]
			mr.bufferSize_gauge.Set(0)
		}
	}

	return nil
}

// QueryMetrics queries metrics from storage
func (mr *MetricsRecorder) QueryMetrics(ctx context.Context, query MetricsQuery) ([]MetricsRecord, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("Querying metrics", "query", fmt.Sprintf("%+v", query))

	return mr.storage.ReadMetrics(ctx, query)
}

// GetMetricNames returns all available metric names
func (mr *MetricsRecorder) GetMetricNames(ctx context.Context) ([]string, error) {
	return mr.storage.GetMetricNames(ctx)
}

// DeleteOldMetrics deletes metrics older than the specified time
func (mr *MetricsRecorder) DeleteOldMetrics(ctx context.Context, olderThan time.Time) error {
	logger := log.FromContext(ctx)
	logger.Info("Deleting old metrics", "olderThan", olderThan)

	filter := MetricsFilter{
		OlderThan: olderThan,
	}

	return mr.storage.DeleteMetrics(ctx, filter)
}

// GetBufferStatus returns current buffer status
func (mr *MetricsRecorder) GetBufferStatus() (int, int) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()
	return len(mr.buffer), mr.bufferSize
}

// FlushBuffer manually flushes the buffer
func (mr *MetricsRecorder) FlushBuffer(ctx context.Context) error {
	return mr.flushBuffer(ctx)
}

// flushLoop runs the periodic buffer flush
func (mr *MetricsRecorder) flushLoop(ctx context.Context) {
	logger := log.FromContext(ctx)
	logger.Info("Starting metrics flush loop", "interval", mr.flushInterval)

	ticker := time.NewTicker(mr.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Flush loop stopping due to context cancellation")
			return
		case <-mr.stopCh:
			logger.Info("Flush loop stopping")
			return
		case <-ticker.C:
			if err := mr.flushBuffer(ctx); err != nil {
				logger.Error(err, "Failed to flush buffer periodically")
			}
		}
	}
}

// flushBuffer flushes the current buffer to storage
func (mr *MetricsRecorder) flushBuffer(ctx context.Context) error {
	mr.mu.Lock()

	if len(mr.buffer) == 0 {
		mr.mu.Unlock()
		return nil
	}

	// Copy buffer for writing
	records := make([]MetricsRecord, len(mr.buffer))
	copy(records, mr.buffer)

	// Clear buffer
	mr.buffer = mr.buffer[:0]
	mr.bufferSize_gauge.Set(0)
	mr.mu.Unlock()

	// Write to storage
	return mr.writeToStorage(ctx, records)
}

// writeToStorage writes records to storage
func (mr *MetricsRecorder) writeToStorage(ctx context.Context, records []MetricsRecord) error {
	startTime := time.Now()
	defer func() {
		mr.writeDuration.Observe(time.Since(startTime).Seconds())
	}()

	logger := log.FromContext(ctx)
	logger.V(1).Info("Writing metrics to storage", "recordCount", len(records))

	err := mr.storage.WriteMetrics(ctx, records)
	if err != nil {
		mr.writeErrors.Inc()
		return fmt.Errorf("failed to write metrics to storage: %w", err)
	}

	mr.recordsWritten.Add(float64(len(records)))
	return nil
}

// convertMetricsData converts MetricsData to MetricsRecords
func (mr *MetricsRecorder) convertMetricsData(data *MetricsData) []MetricsRecord {
	records := make([]MetricsRecord, 0, len(data.Metrics))

	for metricName, value := range data.Metrics {
		// Convert string value to float64
		floatValue := 0.0
		if parsedValue, err := strconv.ParseFloat(value, 64); err == nil {
			floatValue = parsedValue
		}

		record := MetricsRecord{
			ID:        fmt.Sprintf("%s-%s-%d", data.Source, metricName, data.Timestamp.UnixNano()),
			Timestamp: data.Timestamp,
			Source:    data.Source,
			Namespace: data.Namespace,
			Metric:    metricName,
			Value:     floatValue,
			Labels:    make(map[string]string),
			Metadata:  data.Metadata, // Use existing RawMessage
		}

		// Copy labels
		for k, v := range data.Labels {
			record.Labels[k] = v
		}

		// Add pod and container info if available
		if data.Pod != "" {
			record.Labels["pod"] = data.Pod
		}
		if data.Container != "" {
			record.Labels["container"] = data.Container
		}

		records = append(records, record)
	}

	return records
}

// InMemoryMetricsStorage implements MetricsStorage using in-memory storage
type InMemoryMetricsStorage struct {
	mu      sync.RWMutex
	records []MetricsRecord
	indices map[string][]int // Metric name to record indices
}

// NewInMemoryMetricsStorage creates a new in-memory metrics storage
func NewInMemoryMetricsStorage() *InMemoryMetricsStorage {
	return &InMemoryMetricsStorage{
		records: make([]MetricsRecord, 0),
		indices: make(map[string][]int),
	}
}

// WriteMetrics writes metrics to in-memory storage
func (ims *InMemoryMetricsStorage) WriteMetrics(ctx context.Context, records []MetricsRecord) error {
	ims.mu.Lock()
	defer ims.mu.Unlock()

	for _, record := range records {
		// Add record
		index := len(ims.records)
		ims.records = append(ims.records, record)

		// Update indices
		if indices, exists := ims.indices[record.Metric]; exists {
			ims.indices[record.Metric] = append(indices, index)
		} else {
			ims.indices[record.Metric] = []int{index}
		}
	}

	return nil
}

// ReadMetrics reads metrics from in-memory storage
func (ims *InMemoryMetricsStorage) ReadMetrics(ctx context.Context, query MetricsQuery) ([]MetricsRecord, error) {
	ims.mu.RLock()
	defer ims.mu.RUnlock()

	var results []MetricsRecord

	// If specific metrics are requested, use indices
	if len(query.Metrics) > 0 {
		for _, metricName := range query.Metrics {
			if indices, exists := ims.indices[metricName]; exists {
				for _, index := range indices {
					if index < len(ims.records) {
						record := ims.records[index]
						if ims.matchesQuery(record, query) {
							results = append(results, record)
						}
					}
				}
			}
		}
	} else {
		// Scan all records
		for _, record := range ims.records {
			if ims.matchesQuery(record, query) {
				results = append(results, record)
			}
		}
	}

	// Apply limit and offset
	if query.Limit > 0 {
		start := query.Offset
		end := start + query.Limit

		if start < len(results) {
			if end > len(results) {
				end = len(results)
			}
			results = results[start:end]
		} else {
			results = nil
		}
	}

	return results, nil
}

// DeleteMetrics deletes metrics from in-memory storage
func (ims *InMemoryMetricsStorage) DeleteMetrics(ctx context.Context, filter MetricsFilter) error {
	ims.mu.Lock()
	defer ims.mu.Unlock()

	// Create new slice without matching records
	var newRecords []MetricsRecord
	newIndices := make(map[string][]int)

	for i, record := range ims.records {
		if !ims.matchesFilter(record, filter) {
			// Keep this record
			newIndex := len(newRecords)
			newRecords = append(newRecords, record)

			// Update indices
			if indices, exists := newIndices[record.Metric]; exists {
				newIndices[record.Metric] = append(indices, newIndex)
			} else {
				newIndices[record.Metric] = []int{newIndex}
			}
		} else {
			// Delete this record (don't add to new slice)
			_ = i // Record being deleted
		}
	}

	ims.records = newRecords
	ims.indices = newIndices

	return nil
}

// GetMetricNames returns all available metric names
func (ims *InMemoryMetricsStorage) GetMetricNames(ctx context.Context) ([]string, error) {
	ims.mu.RLock()
	defer ims.mu.RUnlock()

	names := make([]string, 0, len(ims.indices))
	for name := range ims.indices {
		names = append(names, name)
	}

	return names, nil
}

// Close closes the in-memory storage (no-op)
func (ims *InMemoryMetricsStorage) Close() error {
	return nil
}

// matchesQuery checks if a record matches the query criteria
func (ims *InMemoryMetricsStorage) matchesQuery(record MetricsRecord, query MetricsQuery) bool {
	// Check time range
	if !query.StartTime.IsZero() && record.Timestamp.Before(query.StartTime) {
		return false
	}
	if !query.EndTime.IsZero() && record.Timestamp.After(query.EndTime) {
		return false
	}

	// Check sources
	if len(query.Sources) > 0 {
		found := false
		for _, source := range query.Sources {
			if record.Source == source {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check labels
	for key, value := range query.Labels {
		if recordValue, exists := record.Labels[key]; !exists || recordValue != value {
			return false
		}
	}

	return true
}

// matchesFilter checks if a record matches the filter criteria for deletion
func (ims *InMemoryMetricsStorage) matchesFilter(record MetricsRecord, filter MetricsFilter) bool {
	// Check older than
	if !filter.OlderThan.IsZero() && record.Timestamp.After(filter.OlderThan) {
		return false
	}

	// Check sources
	if len(filter.Sources) > 0 {
		found := false
		for _, source := range filter.Sources {
			if record.Source == source {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check metrics
	if len(filter.Metrics) > 0 {
		found := false
		for _, metric := range filter.Metrics {
			if record.Metric == metric {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check labels
	for key, value := range filter.Labels {
		if recordValue, exists := record.Labels[key]; !exists || recordValue != value {
			return false
		}
	}

	return true
}

// GetRecordCount returns the total number of records in storage
func (ims *InMemoryMetricsStorage) GetRecordCount() int {
	ims.mu.RLock()
	defer ims.mu.RUnlock()
	return len(ims.records)
}

// GetStorageStats returns storage statistics
func (ims *InMemoryMetricsStorage) GetStorageStats() map[string]interface{} {
	ims.mu.RLock()
	defer ims.mu.RUnlock()

	stats := make(map[string]interface{})

	// Calculate memory usage estimation
	var memoryUsage int
	for _, record := range ims.records {
		// Rough estimation of memory usage per record
		memoryUsage += len(record.ID) + len(record.Source) + len(record.Namespace) + len(record.Metric)
		for k, v := range record.Labels {
			memoryUsage += len(k) + len(v)
		}

		// Add metadata size (rough estimation)
		if data, err := json.Marshal(record.Metadata); err == nil {
			memoryUsage += len(data)
		}

		memoryUsage += 64 // Estimated overhead per record
	}

	stats["estimated_memory_bytes"] = memoryUsage
	stats["record_count"] = len(ims.records)
	stats["metric_count"] = len(ims.indices)

	return stats
}

