package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// MetricsServer provides HTTP endpoints for metrics.
type MetricsServer struct {
	watcher *OptimizedWatcher
	server  *http.Server
	addr    string
	port    int
}

// NewMetricsServer creates a new metrics server.
func NewMetricsServer(watcher *OptimizedWatcher, addr string, port int) *MetricsServer {
	if addr == "" {
		addr = "127.0.0.1"
	}
	if port <= 0 {
		port = 8080
	}

	return &MetricsServer{
		watcher: watcher,
		addr:    addr,
		port:    port,
	}
}

// Start starts the metrics server.
func (s *MetricsServer) Start() error {
	mux := http.NewServeMux()

	// Register endpoints.
	mux.HandleFunc("/metrics", s.handleMetrics)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/stats", s.handleStats)

	s.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.addr, s.port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("Metrics server starting on %s:%d", s.addr, s.port)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	return nil
}

// Stop stops the metrics server.
func (s *MetricsServer) Stop() error {
	if s.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}

// handleMetrics handles the /metrics endpoint.
func (s *MetricsServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := s.watcher.GetMetrics()

	// Convert to Prometheus format.
	fmt.Fprintf(w, "# HELP conductor_processed_total Total number of files processed\n")
	fmt.Fprintf(w, "# TYPE conductor_processed_total counter\n")
	fmt.Fprintf(w, "conductor_processed_total %d\n\n", metrics.FilesProcessedTotal)

	fmt.Fprintf(w, "# HELP conductor_failure_total Total number of failed processings\n")
	fmt.Fprintf(w, "# TYPE conductor_failure_total counter\n")
	fmt.Fprintf(w, "conductor_failure_total %d\n\n", metrics.FilesFailedTotal)

	fmt.Fprintf(w, "# HELP conductor_success_rate Success rate percentage\n")
	fmt.Fprintf(w, "# TYPE conductor_success_rate gauge\n")
	successRate := float64(metrics.FilesProcessedTotal) / float64(metrics.FilesProcessedTotal+metrics.FilesFailedTotal)
	if metrics.FilesProcessedTotal+metrics.FilesFailedTotal == 0 {
		successRate = 0
	}
	fmt.Fprintf(w, "conductor_success_rate %.2f\n\n", successRate*100)

	fmt.Fprintf(w, "# HELP conductor_throughput_per_second Current throughput in files per second\n")
	fmt.Fprintf(w, "# TYPE conductor_throughput_per_second gauge\n")
	fmt.Fprintf(w, "conductor_throughput_per_second %.2f\n\n", metrics.ThroughputFilesPerSecond)

	fmt.Fprintf(w, "# HELP conductor_processing_duration_total Total processing duration in nanoseconds\n")
	fmt.Fprintf(w, "# TYPE conductor_processing_duration_total counter\n")
	fmt.Fprintf(w, "conductor_processing_duration_total %d\n\n", metrics.ProcessingDurationTotal)

	fmt.Fprintf(w, "# HELP conductor_queue_depth Current queue depth\n")
	fmt.Fprintf(w, "# TYPE conductor_queue_depth gauge\n")
	fmt.Fprintf(w, "conductor_queue_depth %d\n\n", metrics.QueueDepthCurrent)

	fmt.Fprintf(w, "# HELP conductor_goroutines Number of goroutines\n")
	fmt.Fprintf(w, "# TYPE conductor_goroutines gauge\n")
	fmt.Fprintf(w, "conductor_goroutines %d\n\n", metrics.GoroutineCount)

	fmt.Fprintf(w, "# HELP conductor_backpressure_total Total backpressure events\n")
	fmt.Fprintf(w, "# TYPE conductor_backpressure_total counter\n")
	fmt.Fprintf(w, "conductor_backpressure_total %d\n\n", metrics.BackpressureEventsTotal)

	fmt.Fprintf(w, "# HELP conductor_memory_usage_bytes Current memory usage in bytes\n")
	fmt.Fprintf(w, "# TYPE conductor_memory_usage_bytes gauge\n")
	fmt.Fprintf(w, "conductor_memory_usage_bytes %d\n\n", metrics.MemoryUsageBytes)

	fmt.Fprintf(w, "# HELP conductor_file_descriptors Current file descriptor count\n")
	fmt.Fprintf(w, "# TYPE conductor_file_descriptors gauge\n")
	fmt.Fprintf(w, "conductor_file_descriptors %d\n\n", metrics.FileDescriptorCount)

	fmt.Fprintf(w, "# HELP conductor_directory_size_bytes Watched directory size in bytes\n")
	fmt.Fprintf(w, "# TYPE conductor_directory_size_bytes gauge\n")
	fmt.Fprintf(w, "conductor_directory_size_bytes %d\n\n", metrics.DirectorySizeBytes)
}

// handleHealth handles the /health endpoint.
func (s *MetricsServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := s.watcher.GetMetrics()

	health := struct {
		Status    string    `json:"status"`
		Timestamp time.Time `json:"timestamp"`
		Checks    struct {
			WorkerPool struct {
				Status        string `json:"status"`
				ActiveWorkers int64  `json:"active_workers"`
				QueueDepth    int    `json:"queue_depth"`
				Backpressured bool   `json:"backpressured"`
			} `json:"worker_pool"`
			Resources struct {
				Status     string  `json:"status"`
				MemoryMB   float64 `json:"memory_mb"`
				CPUPercent float64 `json:"cpu_percent"`
				Goroutines int     `json:"goroutines"`
			} `json:"resources"`
			Processing struct {
				Status      string  `json:"status"`
				SuccessRate float64 `json:"success_rate"`
				Throughput  float64 `json:"throughput"`
			} `json:"processing"`
		} `json:"checks"`
	}{
		Status:    "healthy",
		Timestamp: time.Now(),
	}

	// Check worker pool health from current queue depth.
	health.Checks.WorkerPool.QueueDepth = int(metrics.QueueDepthCurrent)
	health.Checks.WorkerPool.Backpressured = metrics.BackpressureEventsTotal > 0

	if metrics.BackpressureEventsTotal > 0 {
		health.Checks.WorkerPool.Status = "degraded"
		health.Status = "degraded"
	} else {
		health.Checks.WorkerPool.Status = "healthy"
	}

	// Check resource health.
	health.Checks.Resources.MemoryMB = float64(metrics.MemoryUsageBytes) / (1024 * 1024)
	health.Checks.Resources.Goroutines = int(metrics.GoroutineCount)

	if metrics.MemoryUsageBytes > 512*1024*1024 || metrics.GoroutineCount > 1000 {
		health.Checks.Resources.Status = "warning"
		if health.Status == "healthy" {
			health.Status = "degraded"
		}
	} else {
		health.Checks.Resources.Status = "healthy"
	}

	// Check processing health.
	total := metrics.FilesProcessedTotal + metrics.FilesFailedTotal
	successRate := float64(1.0)
	if total > 0 {
		successRate = float64(metrics.FilesProcessedTotal) / float64(total)
	}
	health.Checks.Processing.SuccessRate = successRate
	health.Checks.Processing.Throughput = metrics.ThroughputFilesPerSecond

	if successRate < 0.95 {
		health.Checks.Processing.Status = "degraded"
		health.Status = "degraded"
	} else {
		health.Checks.Processing.Status = "healthy"
	}

	// Set appropriate HTTP status code.
	statusCode := http.StatusOK
	if health.Status == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(health)
}

// handleStats handles the /stats endpoint with detailed statistics.
func (s *MetricsServer) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := s.watcher.GetMetrics()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}
