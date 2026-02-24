// Package loop provides file system watching and processing capabilities for the conductor.
// This file contains metrics collection, reporting, and HTTP server functionality.
package loop

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"
)

// Metrics-related methods for Watcher

func (w *Watcher) startMetricsCollection() {
	// Track metrics collection goroutine for proper shutdown
	w.backgroundWG.Add(1)
	go func() {
		defer w.backgroundWG.Done()
		ticker := time.NewTicker(MetricsUpdateInterval)

		defer ticker.Stop()

		for {
			select {
			case <-w.ctx.Done():

				return

			case <-ticker.C:

				w.updateMetrics()
			}
		}
	}()
}

// updateMetrics updates system and runtime metrics.


func (w *Watcher) updateMetrics() {
	var m runtime.MemStats

	runtime.ReadMemStats(&m)

	w.metrics.mu.Lock()

	defer w.metrics.mu.Unlock()

	// Update resource metrics.

	atomic.StoreInt64(&w.metrics.MemoryUsageBytes, int64(m.Alloc))

	atomic.StoreInt64(&w.metrics.GoroutineCount, int64(runtime.NumGoroutine()))

	// Update business metrics with proper locking.

	now := time.Now()

	duration := now.Sub(w.metrics.StartTime).Seconds()

	w.metrics.mu.Lock()

	if duration > 0 {
		totalProcessed := atomic.LoadInt64(&w.metrics.FilesProcessedTotal)

		w.metrics.ThroughputFilesPerSecond = float64(totalProcessed) / duration
	}

	// Update average processing time.

	totalDuration := atomic.LoadInt64(&w.metrics.ProcessingDurationTotal)

	totalFiles := atomic.LoadInt64(&w.metrics.FilesProcessedTotal)

	if totalFiles > 0 {
		w.metrics.AverageProcessingTime = time.Duration(totalDuration / totalFiles)
	}

	// Update worker utilization.

	activeWorkers := atomic.LoadInt64(&w.workerPool.activeWorkers)

	w.metrics.WorkerUtilization = float64(activeWorkers) / float64(w.workerPool.maxWorkers) * 100

	// Update last update time.

	w.metrics.LastUpdateTime = now

	w.metrics.mu.Unlock()

	// Update queue depth.

	atomic.StoreInt64(&w.metrics.QueueDepthCurrent, int64(len(w.workerPool.workQueue)))

	// Update directory size.

	if dirSize, err := w.calculateDirectorySize(w.dir); err == nil {
		atomic.StoreInt64(&w.metrics.DirectorySizeBytes, dirSize)
	}

	w.metrics.LastUpdateTime = now
}

// calculateDirectorySize calculates the total size of files in a directory.


func (w *Watcher) calculateDirectorySize(dir string) (int64, error) {
	var totalSize int64

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors, don't fail the entire calculation
		}

		if !info.IsDir() {
			totalSize += info.Size()
		}

		return nil
	})

	return totalSize, err
}

// recordProcessingLatency records a processing latency sample.


func (w *Watcher) recordProcessingLatency(duration time.Duration) {
	w.metrics.latencyMutex.Lock()

	defer w.metrics.latencyMutex.Unlock()

	// Add to ring buffer.

	index := atomic.AddInt64(&w.metrics.latencyIndex, 1) % LatencyRingBufferSize

	w.metrics.ProcessingLatencies[index] = duration.Nanoseconds()

	// Update total processing duration.

	atomic.AddInt64(&w.metrics.ProcessingDurationTotal, duration.Nanoseconds())
}

// recordValidationError records a validation error by type.


func (w *Watcher) recordValidationError(errorType string) {
	atomic.AddInt64(&w.metrics.ValidationFailuresTotal, 1)

	w.metrics.mu.Lock()

	w.metrics.ValidationErrorsByType[errorType]++

	w.metrics.mu.Unlock()
}

// recordProcessingError records a processing error by type.


func (w *Watcher) recordProcessingError(errorType string) {
	atomic.AddInt64(&w.metrics.FilesFailedTotal, 1)

	w.metrics.mu.Lock()

	w.metrics.ProcessingErrorsByType[errorType]++

	w.metrics.mu.Unlock()
}

// startMetricsServer starts the HTTP metrics server with security features.


func (w *Watcher) startMetricsServer() error {
	// Skip if metrics are disabled.

	if w.config.MetricsPort == 0 {
		w.logger.InfoEvent("Metrics server disabled", "port", 0)

		return nil
	}

	mux := http.NewServeMux()

	// Wrap handlers with authentication if enabled.

	if w.config.MetricsAuth {
		mux.HandleFunc("/metrics", w.basicAuth(w.handleMetrics))

		mux.HandleFunc("/metrics/prometheus", w.basicAuth(w.handlePrometheusMetrics))

		mux.HandleFunc("/health", w.handleHealth) // Health check doesn't require auth
	} else {
		mux.HandleFunc("/metrics", w.handleMetrics)

		mux.HandleFunc("/metrics/prometheus", w.handlePrometheusMetrics)

		mux.HandleFunc("/health", w.handleHealth)
	}

	// Bind to specific address (default: localhost only).

	addr := fmt.Sprintf("%s:%d", w.config.MetricsAddr, w.config.MetricsPort)

	w.metricsServer = &http.Server{
		Addr: addr,

		Handler: mux,

		ReadTimeout: 5 * time.Second,

		WriteTimeout: 10 * time.Second,

		IdleTimeout: 15 * time.Second,
	}

	// Track metrics server goroutine for proper shutdown
	w.backgroundWG.Add(1)
	go func() {
		defer w.backgroundWG.Done()
		w.logger.InfoEvent("Starting metrics server",
			"addr", addr,
			"auth", w.config.MetricsAuth)

		if err := w.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			w.logger.ErrorEvent(err, "Metrics server error")
		}
	}()

	return nil
}

// basicAuth wraps an HTTP handler with basic authentication.


func (w *Watcher) basicAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		user, pass, ok := req.BasicAuth()

		if !ok || user != w.config.MetricsUser || pass != w.config.MetricsPass {
			resp.Header().Set("WWW-Authenticate", `Basic realm="Metrics"`)

			http.Error(resp, "Unauthorized", http.StatusUnauthorized)

			atomic.AddInt64(&w.metrics.ValidationFailuresTotal, 1)

			return
		}

		handler(resp, req)
	}
}

// handleMetrics handles JSON metrics endpoint.


func (w *Watcher) handleMetrics(writer http.ResponseWriter, request *http.Request) {
	w.metrics.mu.RLock()

	defer w.metrics.mu.RUnlock()

	// Calculate latency percentiles.

	latencies := w.getLatencyPercentiles()

	response := map[string]interface{}{
		"metrics": map[string]interface{}{
			"files_processed_total":     atomic.LoadInt64(&w.metrics.FilesProcessedTotal),
			"files_failed_total":        atomic.LoadInt64(&w.metrics.FilesFailedTotal),
			"throughput_files_per_sec":  w.metrics.ThroughputFilesPerSecond,
			"average_processing_time":   w.metrics.AverageProcessingTime.String(),
			"validation_failures_total": atomic.LoadInt64(&w.metrics.ValidationFailuresTotal),
			"retry_attempts_total":      atomic.LoadInt64(&w.metrics.RetryAttemptsTotal),
			"latency_percentiles":       latencies,
		},
		"resources": map[string]interface{}{},
		"workers":   map[string]interface{}{},
		"errors":    map[string]interface{}{},
		"metadata":  map[string]interface{}{},
	}

	writer.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(writer).Encode(response); err != nil {
		w.logger.ErrorEvent(err, "Failed to encode metrics response")
		http.Error(writer, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// handlePrometheusMetrics handles Prometheus-format metrics endpoint.


func (w *Watcher) handlePrometheusMetrics(writer http.ResponseWriter, request *http.Request) {
	w.metrics.mu.RLock()

	defer w.metrics.mu.RUnlock()

	var buf bytes.Buffer

	// Write Prometheus-format metrics.

	buf.WriteString("# HELP conductor_loop_files_processed_total Total number of files processed successfully\n")

	buf.WriteString("# TYPE conductor_loop_files_processed_total counter\n")

	buf.WriteString(fmt.Sprintf("conductor_loop_files_processed_total %d\n", atomic.LoadInt64(&w.metrics.FilesProcessedTotal)))

	buf.WriteString("# HELP conductor_loop_files_failed_total Total number of files that failed processing\n")

	buf.WriteString("# TYPE conductor_loop_files_failed_total counter\n")

	buf.WriteString(fmt.Sprintf("conductor_loop_files_failed_total %d\n", atomic.LoadInt64(&w.metrics.FilesFailedTotal)))

	buf.WriteString("# HELP conductor_loop_validation_failures_total Total number of validation failures\n")

	buf.WriteString("# TYPE conductor_loop_validation_failures_total counter\n")

	buf.WriteString(fmt.Sprintf("conductor_loop_validation_failures_total %d\n", atomic.LoadInt64(&w.metrics.ValidationFailuresTotal)))

	buf.WriteString("# HELP conductor_loop_retry_attempts_total Total number of retry attempts\n")

	buf.WriteString("# TYPE conductor_loop_retry_attempts_total counter\n")

	buf.WriteString(fmt.Sprintf("conductor_loop_retry_attempts_total %d\n", atomic.LoadInt64(&w.metrics.RetryAttemptsTotal)))

	buf.WriteString("# HELP conductor_loop_throughput_files_per_second Files processed per second\n")

	buf.WriteString("# TYPE conductor_loop_throughput_files_per_second gauge\n")

	buf.WriteString(fmt.Sprintf("conductor_loop_throughput_files_per_second %f\n", w.metrics.ThroughputFilesPerSecond))

	buf.WriteString("# HELP conductor_loop_average_processing_time_seconds Average processing time in seconds\n")

	buf.WriteString("# TYPE conductor_loop_average_processing_time_seconds gauge\n")

	buf.WriteString(fmt.Sprintf("conductor_loop_average_processing_time_seconds %f\n", w.metrics.AverageProcessingTime.Seconds()))

	buf.WriteString("# HELP conductor_loop_memory_usage_bytes Current memory usage in bytes\n")

	buf.WriteString("# TYPE conductor_loop_memory_usage_bytes gauge\n")

	buf.WriteString(fmt.Sprintf("conductor_loop_memory_usage_bytes %d\n", atomic.LoadInt64(&w.metrics.MemoryUsageBytes)))

	buf.WriteString("# HELP conductor_loop_goroutine_count Current number of goroutines\n")

	buf.WriteString("# TYPE conductor_loop_goroutine_count gauge\n")

	buf.WriteString(fmt.Sprintf("conductor_loop_goroutine_count %d\n", atomic.LoadInt64(&w.metrics.GoroutineCount)))

	buf.WriteString("# HELP conductor_loop_worker_utilization_percent Worker pool utilization percentage\n")

	buf.WriteString("# TYPE conductor_loop_worker_utilization_percent gauge\n")

	buf.WriteString(fmt.Sprintf("conductor_loop_worker_utilization_percent %f\n", w.metrics.WorkerUtilization))

	buf.WriteString("# HELP conductor_loop_queue_depth Current queue depth\n")

	buf.WriteString("# TYPE conductor_loop_queue_depth gauge\n")

	buf.WriteString(fmt.Sprintf("conductor_loop_queue_depth %d\n", atomic.LoadInt64(&w.metrics.QueueDepthCurrent)))

	buf.WriteString("# HELP conductor_loop_backpressure_events_total Total backpressure events\n")

	buf.WriteString("# TYPE conductor_loop_backpressure_events_total counter\n")

	buf.WriteString(fmt.Sprintf("conductor_loop_backpressure_events_total %d\n", atomic.LoadInt64(&w.metrics.BackpressureEventsTotal)))

	buf.WriteString("# HELP conductor_loop_timeout_count_total Total number of timeouts\n")

	buf.WriteString("# TYPE conductor_loop_timeout_count_total counter\n")

	buf.WriteString(fmt.Sprintf("conductor_loop_timeout_count_total %d\n", atomic.LoadInt64(&w.metrics.TimeoutCount)))

	// Latency histogram.

	latencies := w.getLatencyPercentiles()

	for percentile, value := range latencies {
		buf.WriteString(fmt.Sprintf("# HELP conductor_loop_processing_latency_seconds_p%s Processing latency %s percentile in seconds\n", percentile, percentile))

		buf.WriteString(fmt.Sprintf("# TYPE conductor_loop_processing_latency_seconds_p%s gauge\n", percentile))

		buf.WriteString(fmt.Sprintf("conductor_loop_processing_latency_seconds_p%s %f\n", percentile, value))
	}

	writer.Header().Set("Content-Type", "text/plain")

	if _, err := writer.Write(buf.Bytes()); err != nil {
		w.logger.ErrorEvent(err, "Failed to write Prometheus metrics")
	}
}

// handleHealth handles health check endpoint.


func (w *Watcher) handleHealth(writer http.ResponseWriter, request *http.Request) {
	health := json.RawMessage(`{}`)

	writer.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(writer).Encode(health); err != nil {
		w.logger.ErrorEvent(err, "Failed to encode health response")
		http.Error(writer, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// getLatencyPercentiles calculates latency percentiles from the ring buffer.


func (w *Watcher) getLatencyPercentiles() map[string]float64 {
	w.metrics.latencyMutex.RLock()

	defer w.metrics.latencyMutex.RUnlock()

	// Collect non-zero latencies and convert to float64.

	var latencies []float64

	for _, latency := range w.metrics.ProcessingLatencies {
		if latency > 0 {
			latencies = append(latencies, float64(latency))
		}
	}

	if len(latencies) == 0 {
		return map[string]float64{
			"50": 0.0,

			"95": 0.0,

			"99": 0.0,
		}
	}

	// Simple percentile calculation (for production, consider using a proper algorithm).

	return map[string]float64{
		"50": percentile(latencies, 50) / 1e9, // Convert to seconds

		"95": percentile(latencies, 95) / 1e9,

		"99": percentile(latencies, 99) / 1e9,
	}
}

// Note: percentile function is defined in bounded_stats.go.

// ProcessExistingFiles processes any existing intent files in the directory (for restart idempotency).


func (w *Watcher) GetMetrics() *WatcherMetrics {
	if w == nil {
		return nil
	}

	if w.metrics == nil {
		return &WatcherMetrics{}
	}

	w.metrics.mu.RLock()

	defer w.metrics.mu.RUnlock()

	// Create a copy to avoid race conditions.

	return &WatcherMetrics{
		FilesProcessedTotal: atomic.LoadInt64(&w.metrics.FilesProcessedTotal),

		FilesFailedTotal: atomic.LoadInt64(&w.metrics.FilesFailedTotal),

		ValidationFailuresTotal: atomic.LoadInt64(&w.metrics.ValidationFailuresTotal),

		RetryAttemptsTotal: atomic.LoadInt64(&w.metrics.RetryAttemptsTotal),

		QueueDepthCurrent: atomic.LoadInt64(&w.metrics.QueueDepthCurrent),

		BackpressureEventsTotal: atomic.LoadInt64(&w.metrics.BackpressureEventsTotal),

		TimeoutCount: atomic.LoadInt64(&w.metrics.TimeoutCount),

		MemoryUsageBytes: atomic.LoadInt64(&w.metrics.MemoryUsageBytes),

		GoroutineCount: atomic.LoadInt64(&w.metrics.GoroutineCount),

		DirectorySizeBytes: atomic.LoadInt64(&w.metrics.DirectorySizeBytes),

		ThroughputFilesPerSecond: w.metrics.ThroughputFilesPerSecond,

		AverageProcessingTime: w.metrics.AverageProcessingTime,

		WorkerUtilization: w.metrics.WorkerUtilization,

		StartTime: w.metrics.StartTime,

		LastUpdateTime: w.metrics.LastUpdateTime,

		MetricsEnabled: w.metrics.MetricsEnabled,
	}
}

// getOrCreateFileLock gets or creates a file-level mutex for the given path.


