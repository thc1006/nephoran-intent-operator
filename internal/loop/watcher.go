package loop

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/thc1006/nephoran-intent-operator/internal/porch"
)

// Config holds configuration for the watcher
type Config struct {
	PorchPath      string        `json:"porch_path"`
	PorchURL       string        `json:"porch_url"`
	Mode           string        `json:"mode"`
	OutDir         string        `json:"out_dir"`
	Once           bool          `json:"once"`
	DebounceDur    time.Duration `json:"debounce_duration"`
	Period         time.Duration `json:"period"`
	MaxWorkers     int           `json:"max_workers"`
	CleanupAfter   time.Duration `json:"cleanup_after"`
	MetricsPort    int           `json:"metrics_port"`    // HTTP port for metrics endpoint
	MetricsAddr    string        `json:"metrics_addr"`    // Bind address for metrics (default: localhost)
	MetricsAuth    bool          `json:"metrics_auth"`    // Enable basic auth for metrics
	MetricsUser    string        `json:"metrics_user"`    // Basic auth username
	MetricsPass    string        `json:"metrics_pass"`    // Basic auth password
}

// Validate checks if the configuration is valid and secure
func (c *Config) Validate() error {
	// Validate MaxWorkers - handle extreme values gracefully
	maxSafeWorkers := runtime.NumCPU() * 4
	if c.MaxWorkers < 1 {
		log.Printf("Warning: max_workers %d is too low, using default value 2", c.MaxWorkers)
		c.MaxWorkers = 2
	}
	if c.MaxWorkers > maxSafeWorkers {
		log.Printf("Warning: max_workers %d exceeds safe limit of %d (4x CPU cores), capping to safe limit", c.MaxWorkers, maxSafeWorkers)
		c.MaxWorkers = maxSafeWorkers
	}
	
	// Validate MetricsPort
	if c.MetricsPort != 0 { // 0 means disabled
		if c.MetricsPort < 1024 || c.MetricsPort > 65535 {
			return fmt.Errorf("metrics_port must be between 1024-65535 or 0 to disable")
		}
	}
	
	// Validate DebounceDur - handle negative values gracefully
	if c.DebounceDur != 0 {
		if c.DebounceDur < 0 {
			log.Printf("Warning: debounce_duration %v is negative, using default 100ms", c.DebounceDur)
			c.DebounceDur = 100 * time.Millisecond
		} else if c.DebounceDur < 10*time.Millisecond {
			log.Printf("Warning: debounce_duration %v is too low, setting to 10ms to prevent CPU thrashing", c.DebounceDur)
			c.DebounceDur = 10 * time.Millisecond
		} else if c.DebounceDur > 5*time.Second {
			log.Printf("Warning: debounce_duration %v is too high, capping to 5s to prevent processing delays", c.DebounceDur)
			c.DebounceDur = 5 * time.Second
		}
	}
	
	// Validate Period (for periodic mode)
	if c.Period != 0 {
		if c.Period < 100*time.Millisecond {
			return fmt.Errorf("period must be at least 100ms")
		}
		if c.Period > 1*time.Hour {
			return fmt.Errorf("period must not exceed 1 hour")
		}
	}
	
	// Validate CleanupAfter
	if c.CleanupAfter != 0 {
		if c.CleanupAfter < 1*time.Hour {
			return fmt.Errorf("cleanup_after must be at least 1 hour")
		}
		if c.CleanupAfter > 30*24*time.Hour {
			return fmt.Errorf("cleanup_after must not exceed 30 days")
		}
	}
	
	// Validate metrics authentication
	if c.MetricsAuth {
		if c.MetricsUser == "" || c.MetricsPass == "" {
			return fmt.Errorf("metrics_auth requires both metrics_user and metrics_pass")
		}
		if len(c.MetricsPass) < 8 {
			return fmt.Errorf("metrics_pass must be at least 8 characters for security")
		}
	}
	
	// Validate Mode
	validModes := map[string]bool{"watch": true, "once": true, "periodic": true, "direct": true, "structured": true}
	if c.Mode != "" && !validModes[c.Mode] {
		return fmt.Errorf("invalid mode %q, must be one of: watch, once, periodic, direct, structured", c.Mode)
	}
	
	return nil
}

// FileProcessingState tracks the state of file processing to prevent duplicates
type FileProcessingState struct {
	processing   map[string]*sync.Mutex  // file-level locks
	recentEvents map[string]time.Time    // recent CREATE/WRITE events
	mu           sync.RWMutex            // protects the maps above
}

// WorkerPool manages a pool of workers with backpressure control
type WorkerPool struct {
	workQueue     chan WorkItem
	workers       sync.WaitGroup
	stopSignal    chan struct{}
	maxWorkers    int
	activeWorkers int64 // atomic counter
}

// WorkItem represents a unit of work for the worker pool
type WorkItem struct {
	FilePath string
	Attempt  int
	Ctx      context.Context
}

// DirectoryManager handles directory creation with sync.Once pattern
type DirectoryManager struct {
	dirOnce map[string]*sync.Once
	mu      sync.RWMutex
}

// WatcherMetrics holds comprehensive metrics for monitoring and observability
type WatcherMetrics struct {
	// Performance Metrics (atomic counters for thread safety)
	FilesProcessedTotal       int64 // Total files processed successfully
	FilesFailedTotal          int64 // Total files that failed processing
	ProcessingDurationTotal   int64 // Total processing time in nanoseconds
	ValidationFailuresTotal   int64 // Total validation failures
	RetryAttemptsTotal        int64 // Total retry attempts
	QueueDepthCurrent         int64 // Current queue depth
	BackpressureEventsTotal   int64 // Total backpressure events
	
	// Latency Metrics (for percentile calculations)
	ProcessingLatencies       []int64 // Ring buffer for latency samples
	latencyIndex              int64   // Current index in ring buffer
	latencyMutex              sync.RWMutex
	
	// Error Metrics
	ValidationErrorsByType    map[string]int64 // Validation errors by type
	ProcessingErrorsByType    map[string]int64 // Processing errors by type
	TimeoutCount              int64            // Number of timeouts
	
	// Resource Metrics
	MemoryUsageBytes          int64 // Current memory usage
	GoroutineCount            int64 // Current number of goroutines
	FileDescriptorCount       int64 // Current file descriptor usage
	DirectorySizeBytes        int64 // Watched directory size
	
	// Business Metrics
	ThroughputFilesPerSecond  float64   // Files processed per second
	AverageProcessingTime     time.Duration // Average processing time
	WorkerUtilization         float64   // Worker pool utilization %
	StatusFileGenerationRate  float64   // Status files generated per second
	
	// Metrics collection metadata
	StartTime                 time.Time
	LastUpdateTime            time.Time
	MetricsEnabled            bool
	
	// Mutex for non-atomic fields
	mu                        sync.RWMutex
}

// MetricsSample holds a sample for latency calculations
type MetricsSample struct {
	Timestamp time.Time
	Value     time.Duration
}

// Security and validation constants
const (
	MaxJSONSize        = 5 * 1024 * 1024   // 5MB max JSON size for handling larger intent files
	MaxStatusSize      = 256 * 1024        // 256KB max status file size (reduced for efficiency)
	MaxMessageSize     = 64 * 1024         // 64KB max error message size
	MaxFileNameLength  = 255               // Max filename length
	MaxPathDepth       = 10                // Max directory depth for path traversal prevention
	MaxJSONDepth       = 100               // Max JSON nesting depth for JSON bomb prevention
	
	// Metrics constants
	LatencyRingBufferSize = 1000           // Size of latency ring buffer
	MetricsUpdateInterval = 10 * time.Second // Metrics update frequency
)

// IntentSchema defines the expected structure of an intent file
type IntentSchema struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   map[string]interface{} `json:"metadata"`
	Spec       map[string]interface{} `json:"spec"`
}

// Watcher monitors a directory for intent file changes
type Watcher struct {
	watcher         *fsnotify.Watcher
	dir             string
	config          Config
	stateManager    *StateManager
	fileManager     *FileManager
	executor        *porch.StatefulExecutor
	
	// Enhanced debouncing and file processing state
	fileState       *FileProcessingState
	workerPool      *WorkerPool
	dirManager      *DirectoryManager
	
	// Metrics and monitoring
	metrics         *WatcherMetrics
	metricsServer   *http.Server
	
	// Context for graceful shutdown
	ctx             context.Context
	cancel          context.CancelFunc
	shutdownComplete chan struct{}
	
	// IntentProcessor for new pattern support
	processor       *IntentProcessor
}

// NewWatcher creates a new file system watcher (backward compatibility with Config approach)
func NewWatcher(dir string, config Config) (*Watcher, error) {
	return NewWatcherWithConfig(dir, config, nil)
}

// NewWatcherWithProcessor creates a new file system watcher with processor (new approach)
func NewWatcherWithProcessor(dir string, processor *IntentProcessor) (*Watcher, error) {
	// Use default config when using processor approach
	defaultConfig := Config{
		MaxWorkers:   2,
		CleanupAfter: 7 * 24 * time.Hour,
		DebounceDur:  100 * time.Millisecond,
		MetricsAddr:  "127.0.0.1",
		MetricsPort:  8080,
		Mode:        "watch",
	}
	return NewWatcherWithConfig(dir, defaultConfig, processor)
}

// NewWatcherWithConfig creates a new file system watcher with both config and processor support
func NewWatcherWithConfig(dir string, config Config, processor *IntentProcessor) (*Watcher, error) {
	// Set defaults before validation
	if config.MaxWorkers <= 0 {
		config.MaxWorkers = 2
	}
	if config.CleanupAfter <= 0 {
		config.CleanupAfter = 7 * 24 * time.Hour // 7 days
	}
	if config.DebounceDur <= 0 {
		config.DebounceDur = 100 * time.Millisecond
	}
	if config.MetricsAddr == "" {
		config.MetricsAddr = "127.0.0.1" // Default to localhost only for security
	}
	if config.MetricsPort < 0 {
		config.MetricsPort = 8080 // Default metrics port
	}
	
	// Validate configuration for security
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	// Add the directory to watch
	if err := watcher.Add(dir); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to add directory %s to watcher: %w", dir, err)
	}
	
	// Create state manager (only if not using processor approach)
	var stateManager *StateManager
	var fileManager *FileManager
	var executor *porch.StatefulExecutor
	
	if processor == nil {
		stateManager, err = NewStateManager(dir)
		if err != nil {
			watcher.Close()
			return nil, fmt.Errorf("failed to create state manager: %w", err)
		}
		
		// Create file manager
		fileManager, err = NewFileManager(dir)
		if err != nil {
			watcher.Close()
			return nil, fmt.Errorf("failed to create file manager: %w", err)
		}
		
		// Create porch executor
		executor = porch.NewStatefulExecutor(porch.ExecutorConfig{
			PorchPath: config.PorchPath,
			Mode:     config.Mode,
			OutDir:   config.OutDir,
			Timeout:  30 * time.Second,
		})
		
		// Validate porch path
		if err := porch.ValidatePorchPath(config.PorchPath); err != nil {
			log.Printf("Warning: Porch validation failed: %v", err)
		}
	}
	
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize enhanced components
	fileState := &FileProcessingState{
		processing:   make(map[string]*sync.Mutex),
		recentEvents: make(map[string]time.Time),
	}
	
	workerPool := &WorkerPool{
		workQueue:  make(chan WorkItem, config.MaxWorkers*3), // increased buffer
		stopSignal: make(chan struct{}),
		maxWorkers: config.MaxWorkers,
	}
	
	dirManager := &DirectoryManager{
		dirOnce: make(map[string]*sync.Once),
	}
	
	// Initialize metrics
	metrics := &WatcherMetrics{
		ProcessingLatencies:    make([]int64, LatencyRingBufferSize),
		ValidationErrorsByType: make(map[string]int64),
		ProcessingErrorsByType: make(map[string]int64),
		StartTime:              time.Now(),
		LastUpdateTime:         time.Now(),
		MetricsEnabled:         true,
	}

	w := &Watcher{
		watcher:          watcher,
		dir:             dir,
		config:          config,
		stateManager:    stateManager,
		fileManager:     fileManager,
		executor:        executor,
		fileState:       fileState,
		workerPool:      workerPool,
		dirManager:      dirManager,
		metrics:         metrics,
		ctx:             ctx,
		cancel:          cancel,
		shutdownComplete: make(chan struct{}),
		processor:       processor, // Support for IntentProcessor pattern
	}
	
	// Start metrics collection
	w.startMetricsCollection()
	
	// Start metrics HTTP server
	if err := w.startMetricsServer(); err != nil {
		log.Printf("Warning: Failed to start metrics server: %v", err)
		}
	
	return w, nil
}

// startMetricsCollection starts background metrics collection
func (w *Watcher) startMetricsCollection() {
	go func() {
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

// updateMetrics updates system and runtime metrics
func (w *Watcher) updateMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	w.metrics.mu.Lock()
	defer w.metrics.mu.Unlock()
	
	// Update resource metrics
	atomic.StoreInt64(&w.metrics.MemoryUsageBytes, int64(m.Alloc))
	atomic.StoreInt64(&w.metrics.GoroutineCount, int64(runtime.NumGoroutine()))
	
	// Update business metrics with proper locking
	now := time.Now()
	duration := now.Sub(w.metrics.StartTime).Seconds()
	
	w.metrics.mu.Lock()
	if duration > 0 {
		totalProcessed := atomic.LoadInt64(&w.metrics.FilesProcessedTotal)
		w.metrics.ThroughputFilesPerSecond = float64(totalProcessed) / duration
	}
	
	// Update average processing time
	totalDuration := atomic.LoadInt64(&w.metrics.ProcessingDurationTotal)
	totalFiles := atomic.LoadInt64(&w.metrics.FilesProcessedTotal)
	if totalFiles > 0 {
		w.metrics.AverageProcessingTime = time.Duration(totalDuration / totalFiles)
	}
	
	// Update worker utilization
	activeWorkers := atomic.LoadInt64(&w.workerPool.activeWorkers)
	w.metrics.WorkerUtilization = float64(activeWorkers) / float64(w.workerPool.maxWorkers) * 100
	
	// Update last update time
	w.metrics.LastUpdateTime = now
	w.metrics.mu.Unlock()
	
	// Update queue depth
	atomic.StoreInt64(&w.metrics.QueueDepthCurrent, int64(len(w.workerPool.workQueue)))
	
	// Update directory size
	if dirSize, err := w.calculateDirectorySize(w.dir); err == nil {
		atomic.StoreInt64(&w.metrics.DirectorySizeBytes, dirSize)
	}
	
	w.metrics.LastUpdateTime = now
}

// calculateDirectorySize calculates the total size of files in a directory
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

// recordProcessingLatency records a processing latency sample
func (w *Watcher) recordProcessingLatency(duration time.Duration) {
	w.metrics.latencyMutex.Lock()
	defer w.metrics.latencyMutex.Unlock()
	
	// Add to ring buffer
	index := atomic.AddInt64(&w.metrics.latencyIndex, 1) % LatencyRingBufferSize
	w.metrics.ProcessingLatencies[index] = duration.Nanoseconds()
	
	// Update total processing duration
	atomic.AddInt64(&w.metrics.ProcessingDurationTotal, duration.Nanoseconds())
}

// recordValidationError records a validation error by type
func (w *Watcher) recordValidationError(errorType string) {
	atomic.AddInt64(&w.metrics.ValidationFailuresTotal, 1)
	
	w.metrics.mu.Lock()
	w.metrics.ValidationErrorsByType[errorType]++
	w.metrics.mu.Unlock()
}

// recordProcessingError records a processing error by type
func (w *Watcher) recordProcessingError(errorType string) {
	atomic.AddInt64(&w.metrics.FilesFailedTotal, 1)
	
	w.metrics.mu.Lock()
	w.metrics.ProcessingErrorsByType[errorType]++
	w.metrics.mu.Unlock()
}

// startMetricsServer starts the HTTP metrics server with security features
func (w *Watcher) startMetricsServer() error {
	// Skip if metrics are disabled
	if w.config.MetricsPort == 0 {
		log.Printf("Metrics server disabled (port=0)")
		return nil
	}
	
	mux := http.NewServeMux()
	
	// Wrap handlers with authentication if enabled
	if w.config.MetricsAuth {
		mux.HandleFunc("/metrics", w.basicAuth(w.handleMetrics))
		mux.HandleFunc("/metrics/prometheus", w.basicAuth(w.handlePrometheusMetrics))
		mux.HandleFunc("/health", w.handleHealth) // Health check doesn't require auth
	} else {
		mux.HandleFunc("/metrics", w.handleMetrics)
		mux.HandleFunc("/metrics/prometheus", w.handlePrometheusMetrics)
		mux.HandleFunc("/health", w.handleHealth)
	}
	
	// Bind to specific address (default: localhost only)
	addr := fmt.Sprintf("%s:%d", w.config.MetricsAddr, w.config.MetricsPort)
	
	w.metricsServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
	
	go func() {
		log.Printf("Starting metrics server on %s (auth=%v)", addr, w.config.MetricsAuth)
		if err := w.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics server error: %v", err)
		}
	}()
	
	return nil
}

// basicAuth wraps an HTTP handler with basic authentication
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

// handleMetrics handles JSON metrics endpoint
func (w *Watcher) handleMetrics(writer http.ResponseWriter, request *http.Request) {
	w.metrics.mu.RLock()
	defer w.metrics.mu.RUnlock()
	
	// Calculate latency percentiles
	latencies := w.getLatencyPercentiles()
	
	response := map[string]interface{}{
		"performance": map[string]interface{}{
			"files_processed_total":     atomic.LoadInt64(&w.metrics.FilesProcessedTotal),
			"files_failed_total":        atomic.LoadInt64(&w.metrics.FilesFailedTotal),
			"throughput_files_per_sec":  w.metrics.ThroughputFilesPerSecond,
			"average_processing_time":   w.metrics.AverageProcessingTime.String(),
			"validation_failures_total": atomic.LoadInt64(&w.metrics.ValidationFailuresTotal),
			"retry_attempts_total":      atomic.LoadInt64(&w.metrics.RetryAttemptsTotal),
			"latency_percentiles": latencies,
		},
		"resources": map[string]interface{}{
			"memory_usage_bytes":    atomic.LoadInt64(&w.metrics.MemoryUsageBytes),
			"goroutine_count":       atomic.LoadInt64(&w.metrics.GoroutineCount),
			"directory_size_bytes":  atomic.LoadInt64(&w.metrics.DirectorySizeBytes),
		},
		"workers": map[string]interface{}{
			"max_workers":         w.workerPool.maxWorkers,
			"active_workers":      atomic.LoadInt64(&w.workerPool.activeWorkers),
			"worker_utilization":  w.metrics.WorkerUtilization,
			"queue_depth":         atomic.LoadInt64(&w.metrics.QueueDepthCurrent),
			"backpressure_events": atomic.LoadInt64(&w.metrics.BackpressureEventsTotal),
		},
		"errors": map[string]interface{}{
			"timeout_count":               atomic.LoadInt64(&w.metrics.TimeoutCount),
			"validation_errors_by_type":   w.metrics.ValidationErrorsByType,
			"processing_errors_by_type":   w.metrics.ProcessingErrorsByType,
		},
		"metadata": map[string]interface{}{
			"start_time":       w.metrics.StartTime.Format(time.RFC3339),
			"last_update":      w.metrics.LastUpdateTime.Format(time.RFC3339),
			"uptime_seconds":   time.Since(w.metrics.StartTime).Seconds(),
			"metrics_enabled":  w.metrics.MetricsEnabled,
		},
	}
	
	writer.Header().Set("Content-Type", "application/json")
	json.NewEncoder(writer).Encode(response)
}

// handlePrometheusMetrics handles Prometheus-format metrics endpoint
func (w *Watcher) handlePrometheusMetrics(writer http.ResponseWriter, request *http.Request) {
	w.metrics.mu.RLock()
	defer w.metrics.mu.RUnlock()
	
	var buf bytes.Buffer
	
	// Write Prometheus-format metrics
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
	
	// Latency histogram
	latencies := w.getLatencyPercentiles()
	for percentile, value := range latencies {
		buf.WriteString(fmt.Sprintf("# HELP conductor_loop_processing_latency_seconds_p%s Processing latency %s percentile in seconds\n", percentile, percentile))
		buf.WriteString(fmt.Sprintf("# TYPE conductor_loop_processing_latency_seconds_p%s gauge\n", percentile))
		buf.WriteString(fmt.Sprintf("conductor_loop_processing_latency_seconds_p%s %f\n", percentile, value))
	}
	
	writer.Header().Set("Content-Type", "text/plain")
	writer.Write(buf.Bytes())
}

// handleHealth handles health check endpoint
func (w *Watcher) handleHealth(writer http.ResponseWriter, request *http.Request) {
	health := map[string]interface{}{
		"status":     "healthy",
		"timestamp":  time.Now().Format(time.RFC3339),
		"uptime":     time.Since(w.metrics.StartTime).String(),
		"components": map[string]string{
			"fsnotify_watcher": "healthy",
			"worker_pool":      "healthy",
			"state_manager":    "healthy",
			"file_manager":     "healthy",
			"metrics":          "healthy",
		},
	}
	
	writer.Header().Set("Content-Type", "application/json")
	json.NewEncoder(writer).Encode(health)
}

// getLatencyPercentiles calculates latency percentiles from the ring buffer
func (w *Watcher) getLatencyPercentiles() map[string]float64 {
	w.metrics.latencyMutex.RLock()
	defer w.metrics.latencyMutex.RUnlock()
	
	// Collect non-zero latencies
	var latencies []int64
	for _, latency := range w.metrics.ProcessingLatencies {
		if latency > 0 {
			latencies = append(latencies, latency)
		}
	}
	
	if len(latencies) == 0 {
		return map[string]float64{
			"50": 0.0,
			"95": 0.0,
			"99": 0.0,
		}
	}
	
	// Simple percentile calculation (for production, consider using a proper algorithm)
	return map[string]float64{
		"50": float64(percentile(latencies, 50)) / 1e9, // Convert to seconds
		"95": float64(percentile(latencies, 95)) / 1e9,
		"99": float64(percentile(latencies, 99)) / 1e9,
	}
}

// percentile calculates the nth percentile of a slice of int64 values
func percentile(values []int64, p int) int64 {
	if len(values) == 0 {
		return 0
	}
	
	// Simple percentile calculation - for production use a sorting algorithm
	index := (len(values) * p) / 100
	if index >= len(values) {
		index = len(values) - 1
	}
	
	return values[index]
}

// ProcessExistingFiles processes any existing intent files in the directory (for restart idempotency)
func (w *Watcher) ProcessExistingFiles() error {
	// If using IntentProcessor pattern, delegate to processor
	if w.processor != nil {
		entries, err := os.ReadDir(w.dir)
		if err != nil {
			return fmt.Errorf("failed to read directory: %w", err)
		}

		count := 0
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			
			if IsIntentFile(entry.Name()) {
				fullPath := filepath.Join(w.dir, entry.Name())
				if err := w.processor.ProcessFile(fullPath); err != nil {
					log.Printf("Error processing existing file %s: %v", fullPath, err)
					// Continue processing other files
				}
				count++
			}
		}

		if count > 0 {
			log.Printf("Processed %d existing intent files", count)
			// Flush any batched files
			if err := w.processor.FlushBatch(); err != nil {
				log.Printf("Error flushing batch after processing existing files: %v", err)
			}
		}

		return nil
	}
	
	// Fallback to legacy processing approach
	return w.processExistingFiles()
}

// Start begins watching for file events
func (w *Watcher) Start() error {
	log.Printf("LOOP:START - Starting watcher on directory: %s", w.dir)
	log.Printf("Configuration: workers=%d, debounce=%v, once=%t, period=%v, metrics_port=%d, processor=%t", 
		w.config.MaxWorkers, w.config.DebounceDur, w.config.Once, w.config.Period, w.config.MetricsPort, w.processor != nil)
	
	// Start enhanced worker pool (only if not using processor)
	if w.processor == nil {
		w.startWorkerPool()
		
		// Start cleanup routine
		go w.cleanupRoutine()
	}
	
	// Start file state cleanup routine
	go w.fileStateCleanupRoutine()
	
	// If in "once" mode, process existing files first
	if w.config.Once {
		if err := w.ProcessExistingFiles(); err != nil {
			log.Printf("Warning: failed to process existing files: %v", err)
		}
		
		// In once mode, wait for all work to complete before exiting
		if w.processor == nil {
			w.waitForWorkersToFinish()
		} else {
			// For processor mode, flush any pending batches
			if err := w.processor.FlushBatch(); err != nil {
				log.Printf("Warning: failed to flush batch: %v", err)
			}
		}
		return nil
	}
	
	// Use polling if period is set, otherwise use fsnotify
	if w.config.Period > 0 {
		return w.startPolling()
	}
	
	// Main event loop (fsnotify)
	for {
		select {
		case <-w.ctx.Done():
			log.Printf("Watcher context cancelled, shutting down...")
			if w.processor == nil {
				w.waitForWorkersToFinish()
			}
			return nil
			
		case event, ok := <-w.watcher.Events:
			if !ok {
				log.Printf("Watcher events channel closed")
				if w.processor == nil {
					w.waitForWorkersToFinish()
				}
				return nil
			}
			
			// Process only Create and Write events for intent files
			if event.Op&(fsnotify.Create|fsnotify.Write) != 0 {
				if IsIntentFile(filepath.Base(event.Name)) {
					// Use hybrid processing approach
					w.handleIntentFile(event)
				}
			}
			
		case err, ok := <-w.watcher.Errors:
			if !ok {
				log.Printf("Watcher errors channel closed")
				if w.processor == nil {
					w.waitForWorkersToFinish()
				}
				return nil
			}
			if err != nil {
				log.Printf("Watcher error: %v", err)
			}
		}
	}
}

// handleIntentFile processes detected intent files using hybrid approach
func (w *Watcher) handleIntentFile(event fsnotify.Event) {
	operation := "UNKNOWN"
	if event.Op&fsnotify.Create != 0 {
		operation = "CREATE"
	} else if event.Op&fsnotify.Write != 0 {
		operation = "WRITE"
	}
	
	log.Printf("LOOP:%s - Intent file detected: %s", operation, filepath.Base(event.Name))
	
	// If we have a processor, use it directly
	if w.processor != nil {
		if err := w.processor.ProcessFile(event.Name); err != nil {
			log.Printf("Error processing file %s: %v", event.Name, err)
		}
		return
	}
	
	// Fallback to enhanced legacy processing with debouncing
	w.handleIntentFileWithEnhancedDebounce(event.Name, event.Op)
}

// handleIntentFileWithEnhancedDebounce handles file events with enhanced debouncing to prevent duplicate processing
func (w *Watcher) handleIntentFileWithEnhancedDebounce(filePath string, eventOp fsnotify.Op) {
	w.fileState.mu.Lock()
	defer w.fileState.mu.Unlock()
	
	now := time.Now()
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		log.Printf("Warning: failed to get absolute path for %s: %v", filePath, err)
		absPath = filePath
	}
	
	// Check for recent events to prevent duplicate processing of CREATE+WRITE sequences
	if lastEvent, exists := w.fileState.recentEvents[absPath]; exists {
		if now.Sub(lastEvent) < w.config.DebounceDur {
			log.Printf("LOOP:DEBOUNCE - Skipping duplicate event for %s (last: %v ago)", 
				filepath.Base(filePath), now.Sub(lastEvent))
			return
		}
	}
	
	// Record this event
	w.fileState.recentEvents[absPath] = now
	
	// Debounce the processing with cross-platform file system timing
	go func() {
		// Use adaptive debouncing based on runtime OS
		debounceTime := w.config.DebounceDur
		if runtime.GOOS == "windows" {
			// Windows needs longer debounce due to file system latency
			if debounceTime < 200*time.Millisecond {
				debounceTime = 200 * time.Millisecond
			}
		}
		
		time.Sleep(debounceTime)
		
		// Additional file existence and stability check
		if !w.isFileStable(absPath) {
			log.Printf("LOOP:UNSTABLE - File %s not stable, skipping", filepath.Base(filePath))
			return
		}
		
		// Create work item with timeout context
		workCtx, cancel := context.WithTimeout(w.ctx, 60*time.Second)
		workItem := WorkItem{
			FilePath: absPath,
			Attempt:  1,
			Ctx:      workCtx,
		}
		
		// Queue work item with backpressure handling
		select {
		case w.workerPool.workQueue <- workItem:
			// Successfully queued
		case <-w.ctx.Done():
			cancel()
			return
		default:
			// Work queue is full, implement backpressure
			log.Printf("LOOP:BACKPRESSURE - Work queue full, applying backpressure for %s", 
				filepath.Base(filePath))
			
			// Record backpressure event
			atomic.AddInt64(&w.metrics.BackpressureEventsTotal, 1)
			
			// Try again with exponential backoff
			go w.retryWithBackoff(workItem, cancel)
		}
	}()
}

// processExistingFiles processes all existing intent files in the directory (for -once mode)
func (w *Watcher) processExistingFiles() error {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}
	
	filesQueued := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		filename := entry.Name()
		if IsIntentFile(filename) {
			filePath := filepath.Join(w.dir, filename)
			
			workCtx, cancel := context.WithTimeout(w.ctx, 60*time.Second)
			workItem := WorkItem{
				FilePath: filePath,
				Attempt:  1,
				Ctx:      workCtx,
			}
			
			select {
			case w.workerPool.workQueue <- workItem:
				filesQueued++
			case <-w.ctx.Done():
				cancel()
				return nil
			default:
				cancel()
				log.Printf("Warning: work queue full during startup, skipping %s", filename)
			}
		}
	}
	
	log.Printf("Queued %d existing intent files for processing", filesQueued)
	return nil
}

// startPolling starts the polling-based file monitoring
func (w *Watcher) startPolling() error {
	ticker := time.NewTicker(w.config.Period)
	defer ticker.Stop()
	
	log.Printf("LOOP:SCAN - Starting polling mode with period: %v", w.config.Period)
	
	// Track processed files to avoid duplicates
	processedFiles := make(map[string]string) // filename -> SHA256
	
	for {
		select {
		case <-w.ctx.Done():
			log.Printf("LOOP:DONE - Polling context cancelled, shutting down...")
			w.waitForWorkersToFinish()
			return nil
			
		case <-ticker.C:
			log.Printf("LOOP:SCAN - Scanning directory: %s", w.dir)
			
			entries, err := os.ReadDir(w.dir)
			if err != nil {
				log.Printf("LOOP:ERROR - Failed to read directory: %v", err)
				continue
			}
			
			filesFound := 0
			filesQueued := 0
			
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				
				filename := entry.Name()
				if !IsIntentFile(filename) {
					continue
				}
				
				filesFound++
				filePath := filepath.Join(w.dir, filename)
				
				// Check if file is already being processed
				sha256Hash, err := w.stateManager.CalculateFileSHA256(filePath)
				if err != nil {
					log.Printf("LOOP:ERROR - Failed to calculate SHA256 for %s: %v", filePath, err)
					continue
				}
				
				// Check if this exact file (by SHA) was already processed
				if prevSHA, exists := processedFiles[filename]; exists && prevSHA == sha256Hash {
					log.Printf("LOOP:SKIP_DUP - File %s unchanged (SHA: %s), skipping", filename, sha256Hash[:8])
					continue
				}
				
				// Check state manager for historical processing
				processed, err := w.stateManager.IsProcessedBySHA(sha256Hash)
				if err != nil {
					log.Printf("LOOP:ERROR - Failed to check state for %s: %v", filePath, err)
				} else if processed {
					log.Printf("LOOP:SKIP_DUP - File %s already processed (SHA: %s), skipping", filename, sha256Hash[:8])
					processedFiles[filename] = sha256Hash
					continue
				}
				
				// Queue for processing
				workCtx, cancel := context.WithTimeout(w.ctx, 60*time.Second)
				workItem := WorkItem{
					FilePath: filePath,
					Attempt:  1,
					Ctx:      workCtx,
				}
				
				select {
				case w.workerPool.workQueue <- workItem:
					filesQueued++
					processedFiles[filename] = sha256Hash
					log.Printf("LOOP:PROCESS - Queued file %s for processing", filename)
				default:
					cancel()
					log.Printf("LOOP:WARNING - Work queue full, skipping file %s", filename)
				}
			}
			
			if filesFound > 0 {
				log.Printf("LOOP:SCAN - Found %d intent files, queued %d for processing", filesFound, filesQueued)
			}
		}
	}
}

// startWorkerPool starts the enhanced worker pool
func (w *Watcher) startWorkerPool() {
	for i := 0; i < w.workerPool.maxWorkers; i++ {
		w.workerPool.workers.Add(1)
		go w.enhancedWorker(i)
	}
	log.Printf("Started %d enhanced workers", w.workerPool.maxWorkers)
}

// enhancedWorker processes work items from the enhanced work queue
func (w *Watcher) enhancedWorker(workerID int) {
	defer w.workerPool.workers.Done()
	atomic.AddInt64(&w.workerPool.activeWorkers, 1)
	defer atomic.AddInt64(&w.workerPool.activeWorkers, -1)
	
	log.Printf("Enhanced worker %d started", workerID)
	
	for {
		select {
		case <-w.ctx.Done():
			log.Printf("Enhanced worker %d cancelled", workerID)
			return
			
		case workItem, ok := <-w.workerPool.workQueue:
			if !ok {
				log.Printf("Enhanced worker %d: work queue closed, exiting", workerID)
				return
			}
			w.processWorkItemWithLocking(workerID, workItem)
		}
	}
}

// processWorkItemWithLocking processes a work item with file-level locking
func (w *Watcher) processWorkItemWithLocking(workerID int, workItem WorkItem) {
	defer func() {
		if workItem.Ctx != nil {
			workItem.Ctx.Done()
		}
	}()
	
	// Get or create file-level mutex
	fileLock := w.getOrCreateFileLock(workItem.FilePath)
	fileLock.Lock()
	defer fileLock.Unlock()
	
	w.processIntentFileWithContext(workerID, workItem)
}

// processIntentFileWithContext processes a single intent file with context and timeout
func (w *Watcher) processIntentFileWithContext(workerID int, workItem WorkItem) {
	startTime := time.Now()
	filePath := workItem.FilePath
	log.Printf("LOOP:PROCESS - Worker %d processing file: %s (attempt %d)", 
		workerID, filepath.Base(filePath), workItem.Attempt)
	
	// Validate JSON file before processing
	if err := w.validateJSONFile(filePath); err != nil {
		log.Printf("LOOP:ERROR - Worker %d: JSON validation failed for %s: %v", 
			workerID, filepath.Base(filePath), err)
		
		// Record validation error
		w.recordValidationError(err.Error())
		
		// Mark as failed and move to failed directory
		w.stateManager.MarkFailed(filePath)
		w.fileManager.MoveToFailed(filePath, fmt.Sprintf("Validation failed: %v", err))
		w.writeStatusFileAtomic(filePath, "failed", fmt.Sprintf("JSON validation failed: %v", err))
		return
	}
	
	// Check if already processed using state manager
	processed, err := w.stateManager.IsProcessed(filePath)
	if err != nil {
		log.Printf("LOOP:ERROR - Worker %d: Error checking file state for %s: %v", 
			workerID, filepath.Base(filePath), err)
	} else if processed {
		log.Printf("LOOP:SKIP_DUP - Worker %d: File %s already processed", 
			workerID, filepath.Base(filePath))
		return
	}
	
	// Execute porch command with context timeout
	result, err := w.executor.Execute(workItem.Ctx, filePath)
	
	// Record processing latency
	processingDuration := time.Since(startTime)
	w.recordProcessingLatency(processingDuration)
	
	if result.Success {
		// Mark as processed in state manager
		if err := w.stateManager.MarkProcessed(filePath); err != nil {
			log.Printf("LOOP:WARNING - Worker %d: Failed to mark file as processed: %v", workerID, err)
		}
		
		// Move to processed directory
		if err := w.fileManager.MoveToProcessed(filePath); err != nil {
			log.Printf("LOOP:WARNING - Worker %d: Failed to move file to processed: %v", workerID, err)
		}
		
		// Write success status
		w.writeStatusFileAtomic(filePath, "success", fmt.Sprintf("Processed by worker %d in %v", workerID, result.Duration))
		
		// Record successful processing
		atomic.AddInt64(&w.metrics.FilesProcessedTotal, 1)
		
		log.Printf("LOOP:DONE - Worker %d: Successfully processed %s", workerID, filepath.Base(filePath))
	} else {
		// Mark as failed in state manager
		if err := w.stateManager.MarkFailed(filePath); err != nil {
			log.Printf("LOOP:WARNING - Worker %d: Failed to mark file as failed: %v", workerID, err)
		}
		
		// Create error message
		errorMsg := "Unknown error"
		if result.Error != nil {
			errorMsg = result.Error.Error()
			
			// Check for timeout
			if strings.Contains(errorMsg, "timed out") {
				atomic.AddInt64(&w.metrics.TimeoutCount, 1)
			}
		}
		if result.Stderr != "" {
			errorMsg += " (stderr: " + result.Stderr + ")"
		}
		
		// Record processing error
		w.recordProcessingError(errorMsg)
		
		// Move to failed directory
		if err := w.fileManager.MoveToFailed(filePath, errorMsg); err != nil {
			log.Printf("LOOP:WARNING - Worker %d: Failed to move file to failed: %v", workerID, err)
		}
		
		// Write failure status
		w.writeStatusFileAtomic(filePath, "failed", errorMsg)
		
		log.Printf("LOOP:ERROR - Worker %d: Failed to process %s: %s", workerID, filepath.Base(filePath), errorMsg)
	}
}


// cleanupRoutine periodically cleans up old processed and failed files
func (w *Watcher) cleanupRoutine() {
	ticker := time.NewTicker(24 * time.Hour) // Run cleanup daily
	defer ticker.Stop()
	
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			log.Printf("Running periodic cleanup...")
			
			// Cleanup old state entries
			if err := w.stateManager.CleanupOldEntries(w.config.CleanupAfter); err != nil {
				log.Printf("Warning: cleanup state entries failed: %v", err)
			}
			
			// Cleanup old processed/failed files
			if err := w.fileManager.CleanupOldFiles(w.config.CleanupAfter); err != nil {
				log.Printf("Warning: cleanup old files failed: %v", err)
			}
			
			// Log statistics
			stats := w.executor.GetStats()
			log.Printf("Executor stats: total=%d, success=%d, failed=%d, avg_time=%v",
				stats.TotalExecutions, stats.SuccessfulExecs, stats.FailedExecs, stats.AverageExecTime)
		}
	}
}

// waitForWorkersToFinish waits for all workers to complete and cleans up
func (w *Watcher) waitForWorkersToFinish() {
	log.Printf("Signaling enhanced workers to stop...")
	
	// Wait for queue to drain first
	for {
		queueSize := len(w.workerPool.workQueue)
		if queueSize == 0 {
			break
		}
		log.Printf("Waiting for queue to drain: %d items remaining", queueSize)
		time.Sleep(100 * time.Millisecond)
	}
	
	// Give workers time to finish current processing
	time.Sleep(200 * time.Millisecond)
	
	// Close work queue to signal workers to finish processing and exit
	close(w.workerPool.workQueue)
	
	log.Printf("Waiting for enhanced workers to finish...")
	w.workerPool.workers.Wait()
	
	log.Printf("All enhanced workers stopped")
	close(w.shutdownComplete)
}

// Close stops the watcher and releases resources
func (w *Watcher) Close() error {
	// Defensive nil check to prevent panic
	if w == nil {
		log.Printf("Close() called on nil Watcher - ignoring")
		return nil
	}
	
	log.Printf("Closing watcher...")
	
	// Cancel context to signal shutdown
	if w.cancel != nil {
		w.cancel()
	}
	
	// Close the metrics server
	if w.metricsServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		w.metricsServer.Shutdown(ctx)
	}
	
	// Close the file system watcher
	if w.watcher != nil {
		w.watcher.Close()
	}
	
	// Stop processor if using IntentProcessor pattern
	if w.processor != nil {
		w.processor.Stop()
	}
	
	// Save state (only if using legacy approach)
	if w.stateManager != nil {
		if err := w.stateManager.Close(); err != nil {
			log.Printf("Warning: failed to save state: %v", err)
		}
	}
	
	// Clear file state tracking
	if w.fileState != nil {
		w.fileState.mu.Lock()
		w.fileState.recentEvents = make(map[string]time.Time)
		w.fileState.processing = make(map[string]*sync.Mutex)
		w.fileState.mu.Unlock()
	}
	
	log.Printf("Watcher closed")
	return nil
}

// validateJSONDepth checks if JSON has excessive nesting to prevent JSON bomb attacks
func (w *Watcher) validateJSONDepth(reader io.Reader, maxDepth int) error {
	decoder := json.NewDecoder(reader)
	depth := 0
	
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		
		switch token := token.(type) {
		case json.Delim:
			if token == '{' || token == '[' {
				depth++
				if depth > maxDepth {
					return fmt.Errorf("JSON nesting depth %d exceeds maximum %d", depth, maxDepth)
				}
			} else if token == '}' || token == ']' {
				depth--
			}
		}
	}
	
	return nil
}

// validateJSONFile validates that a JSON file is safe to parse and meets requirements
func (w *Watcher) validateJSONFile(filePath string) error {
	// Check file size
	stat, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	
	if stat.Size() > MaxJSONSize {
		return fmt.Errorf("file size %d exceeds maximum %d bytes", stat.Size(), MaxJSONSize)
	}
	
	if stat.Size() == 0 {
		return fmt.Errorf("file is empty")
	}
	
	// Validate path safety (prevent traversal)
	if err := w.validatePath(filePath); err != nil {
		return fmt.Errorf("path validation failed: %w", err)
	}
	
	// Read and validate JSON structure
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	
	// Use limited reader to prevent memory exhaustion
	limitedReader := io.LimitReader(file, MaxJSONSize)
	
	// Check for JSON bomb (excessive nesting) before full parsing
	if err := w.validateJSONDepth(limitedReader, MaxJSONDepth); err != nil {
		return fmt.Errorf("JSON bomb detected: %w", err)
	}
	
	// Reset reader position
	file.Seek(0, 0)
	limitedReader = io.LimitReader(file, MaxJSONSize)
	
	// Parse JSON with decoder (more memory efficient than ReadAll)
	decoder := json.NewDecoder(limitedReader)
	decoder.DisallowUnknownFields() // Strict validation
	
	var intent IntentSchema
	if err := decoder.Decode(&intent); err != nil {
		// Try as generic map if schema validation fails
		file.Seek(0, 0)
		limitedReader = io.LimitReader(file, MaxJSONSize)
		decoder = json.NewDecoder(limitedReader)
		
		var genericIntent map[string]interface{}
		if err := decoder.Decode(&genericIntent); err != nil {
			return fmt.Errorf("invalid JSON format: %w", err)
		}
		
		// Validate required fields exist
		if err := w.validateIntentFields(genericIntent); err != nil {
			return fmt.Errorf("intent validation failed: %w", err)
		}
	} else {
		// Convert struct back to map for validation
		intentMap := map[string]interface{}{
			"apiVersion": intent.APIVersion,
			"kind":       intent.Kind,
			"metadata":   intent.Metadata,
			"spec":       intent.Spec,
		}
		
		// Validate the structured intent fields
		if err := w.validateIntentFields(intentMap); err != nil {
			return fmt.Errorf("intent validation failed: %w", err)
		}
	}
	
	return nil
}

// validatePath ensures the file path is safe and within expected boundaries
func (w *Watcher) validatePath(filePath string) error {
	// Clean the path to remove any ../ or ./ sequences
	cleanPath := filepath.Clean(filePath)
	
	// Ensure the path is absolute
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}
	
	// Check if path is within the watched directory
	watchedDir, err := filepath.Abs(w.dir)
	if err != nil {
		return fmt.Errorf("failed to get watched directory absolute path: %w", err)
	}
	
	// Ensure the file is within the watched directory
	if !strings.HasPrefix(absPath, watchedDir) {
		return fmt.Errorf("file path %s is outside watched directory %s", absPath, watchedDir)
	}
	
	// Check path depth to prevent deep traversal attacks
	relPath, err := filepath.Rel(watchedDir, absPath)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}
	
	depth := len(strings.Split(relPath, string(filepath.Separator)))
	if depth > MaxPathDepth {
		return fmt.Errorf("path depth %d exceeds maximum %d", depth, MaxPathDepth)
	}
	
	// Check filename length
	filename := filepath.Base(absPath)
	if len(filename) > MaxFileNameLength {
		return fmt.Errorf("filename length %d exceeds maximum %d", len(filename), MaxFileNameLength)
	}
	
	// Check for suspicious patterns in filename
	suspiciousPatterns := []string{
		"..",
		"~",
		"$",
		"*",
		"?",
		"[",
		"]",
		"{",
		"}",
		"|",
		"<",
		">",
		"\x00", // null byte
	}
	
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(filename, pattern) {
			return fmt.Errorf("filename contains suspicious pattern: %s", pattern)
		}
	}
	
	return nil
}

// validateIntentFields checks that required fields exist in the intent
func (w *Watcher) validateIntentFields(intent map[string]interface{}) error {
	// Determine intent format and validate accordingly
	if w.isKubernetesStyleIntent(intent) {
		return w.validateKubernetesStyleIntent(intent)
	}
	
	// Fallback to simple scaling intent validation
	return w.validateScalingIntent(intent)
}

// isKubernetesStyleIntent determines if the intent follows Kubernetes resource format
func (w *Watcher) isKubernetesStyleIntent(intent map[string]interface{}) bool {
	_, hasAPIVersion := intent["apiVersion"]
	_, hasKind := intent["kind"]
	// Check if it has apiVersion and kind (metadata is optional)
	return hasAPIVersion && hasKind
}

// validateKubernetesStyleIntent validates Kubernetes-style intent structure
func (w *Watcher) validateKubernetesStyleIntent(intent map[string]interface{}) error {
	// Validate required top-level fields
	requiredFields := []string{"apiVersion", "kind"}
	for _, field := range requiredFields {
		if _, exists := intent[field]; !exists {
			return fmt.Errorf("missing required field: %s", field)
		}
	}
	
	// Validate apiVersion format
	if err := w.validateAPIVersion(intent["apiVersion"]); err != nil {
		return fmt.Errorf("apiVersion validation failed: %w", err)
	}
	
	// Validate kind
	if err := w.validateKind(intent["kind"]); err != nil {
		return fmt.Errorf("kind validation failed: %w", err)
	}
	
	// Validate metadata if present
	if metadata, exists := intent["metadata"]; exists {
		if err := w.validateMetadata(metadata); err != nil {
			return fmt.Errorf("metadata validation failed: %w", err)
		}
	}
	
	// Validate spec based on kind
	if spec, exists := intent["spec"]; exists {
		kind, _ := intent["kind"].(string)
		if err := w.validateSpecByKind(kind, spec); err != nil {
			return fmt.Errorf("spec validation failed: %w", err)
		}
	}
	
	return nil
}

// validateScalingIntent validates simple scaling intent structure (legacy format)
func (w *Watcher) validateScalingIntent(intent map[string]interface{}) error {
	// Required fields for scaling intent
	requiredFields := []string{"intent_type", "target", "namespace", "replicas"}
	for _, field := range requiredFields {
		value, exists := intent[field]
		if !exists || value == nil {
			return fmt.Errorf("missing or null required field: %s", field)
		}
	}
	
	// Validate intent_type
	if intentType, ok := intent["intent_type"].(string); !ok {
		return fmt.Errorf("intent_type must be a string")
	} else {
		validTypes := map[string]bool{"scaling": true, "deployment": true, "service": true}
		if !validTypes[intentType] {
			return fmt.Errorf("intent_type '%s' is not supported (supported: scaling, deployment, service)", intentType)
		}
	}
	
	// Validate target
	if targetVal := intent["target"]; targetVal == nil {
		return fmt.Errorf("target cannot be null")
	} else if target, ok := targetVal.(string); !ok || target == "" {
		return fmt.Errorf("target must be a non-empty string")
	}
	
	// Validate namespace
	if namespaceVal := intent["namespace"]; namespaceVal == nil {
		return fmt.Errorf("namespace cannot be null")
	} else if namespace, ok := namespaceVal.(string); !ok || namespace == "" {
		return fmt.Errorf("namespace must be a non-empty string")
	}
	
	// Validate replicas - handle both int and float64 from JSON
	if replicasVal, exists := intent["replicas"]; exists {
		if replicasVal == nil {
			return fmt.Errorf("replicas cannot be null")
		}
		
		var replicas float64
		var ok bool
		
		// JSON unmarshaling can produce either int or float64
		switch v := replicasVal.(type) {
		case float64:
			replicas = v
			ok = true
		case int:
			replicas = float64(v)
			ok = true
		case int64:
			replicas = float64(v)
			ok = true
		default:
			ok = false
		}
		
		if !ok || replicas < 0 || replicas > 1000 {
			return fmt.Errorf("replicas must be a number between 0 and 1000")
		}
	}
	
	// Validate optional fields
	if source, exists := intent["source"]; exists {
		if sourceStr, ok := source.(string); !ok {
			return fmt.Errorf("source must be a string")
		} else {
			validSources := map[string]bool{"user": true, "planner": true, "test": true}
			if !validSources[sourceStr] {
				return fmt.Errorf("source must be one of: user, planner, test")
			}
		}
	}
	
	return nil
}

// validateAPIVersion validates the apiVersion field format
func (w *Watcher) validateAPIVersion(apiVersion interface{}) error {
	apiVersionStr, ok := apiVersion.(string)
	if !ok || apiVersionStr == "" {
		return fmt.Errorf("apiVersion must be a non-empty string")
	}
	
	// Expected formats: "group/version" or "version" for core resources
	validPatterns := []*regexp.Regexp{
		regexp.MustCompile(`^[a-z0-9.-]+/v[0-9]+([a-z]+[0-9]*)?$`), // group/version format (e.g., "intent.nephio.org/v1alpha1")
		regexp.MustCompile(`^v[0-9]+([a-z]+[0-9]*)?$`),             // version only format (e.g., "v1")
		regexp.MustCompile(`^nephoran\.com/v[0-9]+([a-z]+[0-9]*)?$`), // nephoran-specific format
	}
	
	for _, pattern := range validPatterns {
		if pattern.MatchString(apiVersionStr) {
			return nil
		}
	}
	
	// Simple check for basic format requirements
	if len(apiVersionStr) < 2 || (!strings.Contains(apiVersionStr, "/") && !strings.HasPrefix(apiVersionStr, "v")) {
		return fmt.Errorf("apiVersion format invalid: expected 'group/version' or 'version' format")
	}
	
	return nil
}

// validateKind validates the kind field
func (w *Watcher) validateKind(kind interface{}) error {
	kindStr, ok := kind.(string)
	if !ok || kindStr == "" {
		return fmt.Errorf("kind must be a non-empty string")
	}
	
	// Validate known intent kinds
	validKinds := map[string]bool{
		"NetworkIntent":   true,
		"ResourceIntent":  true,
		"ScalingIntent":   true,
		"DeploymentIntent": true,
		"ServiceIntent":   true,
	}
	
	if !validKinds[kindStr] {
		return fmt.Errorf("unsupported kind: %s (supported: NetworkIntent, ResourceIntent, ScalingIntent, DeploymentIntent, ServiceIntent)", kindStr)
	}
	
	return nil
}

// validateMetadata validates the metadata field structure
func (w *Watcher) validateMetadata(metadata interface{}) error {
	metaMap, ok := metadata.(map[string]interface{})
	if !ok {
		return fmt.Errorf("metadata must be an object")
	}
	
	// Note: metadata.name is not strictly required for all intent types
	// but if present, it must be valid
	
	// Validate name if present
	if name, exists := metaMap["name"]; exists {
		if nameStr, ok := name.(string); !ok || nameStr == "" {
			return fmt.Errorf("metadata.name must be a non-empty string")
		} else {
			// Validate name format (Kubernetes naming conventions)
			if len(nameStr) > 253 {
				return fmt.Errorf("metadata.name too long (max 253 characters)")
			}
			if !isValidKubernetesName(nameStr) {
				return fmt.Errorf("metadata.name contains invalid characters (must be lowercase alphanumeric, hyphens, dots)")
			}
		}
	}
	
	// Validate namespace if present
	if namespace, exists := metaMap["namespace"]; exists {
		if namespaceStr, ok := namespace.(string); !ok || namespaceStr == "" {
			return fmt.Errorf("metadata.namespace must be a non-empty string")
		} else if !isValidKubernetesName(namespaceStr) {
			return fmt.Errorf("metadata.namespace contains invalid characters")
		}
	}
	
	// Validate labels if present
	if labels, exists := metaMap["labels"]; exists {
		if err := w.validateLabels(labels); err != nil {
			return fmt.Errorf("metadata.labels validation failed: %w", err)
		}
	}
	
	return nil
}

// validateSpecByKind validates the spec field based on the intent kind
func (w *Watcher) validateSpecByKind(kind string, spec interface{}) error {
	specMap, ok := spec.(map[string]interface{})
	if !ok {
		return fmt.Errorf("spec must be an object")
	}
	
	switch kind {
	case "NetworkIntent":
		return w.validateNetworkIntentSpec(specMap)
	case "ResourceIntent":
		return w.validateResourceIntentSpec(specMap)
	case "ScalingIntent":
		return w.validateScalingIntentSpec(specMap)
	case "DeploymentIntent":
		return w.validateDeploymentIntentSpec(specMap)
	case "ServiceIntent":
		return w.validateServiceIntentSpec(specMap)
	default:
		// For unknown kinds, perform basic validation
		return w.validateGenericSpec(specMap)
	}
}

// validateNetworkIntentSpec validates NetworkIntent spec fields
func (w *Watcher) validateNetworkIntentSpec(spec map[string]interface{}) error {
	// Validate action field
	if action, exists := spec["action"]; exists {
		if actionStr, ok := action.(string); !ok {
			return fmt.Errorf("spec.action must be a string")
		} else {
			validActions := map[string]bool{"deploy": true, "scale": true, "update": true, "delete": true}
			if !validActions[actionStr] {
				return fmt.Errorf("spec.action must be one of: deploy, scale, update, delete")
			}
		}
	}
	
	// Validate target configuration
	if target, exists := spec["target"]; exists {
		if err := w.validateTargetSpec(target); err != nil {
			return fmt.Errorf("spec.target validation failed: %w", err)
		}
	}
	
	// Validate parameters
	if parameters, exists := spec["parameters"]; exists {
		if err := w.validateNetworkParameters(parameters); err != nil {
			return fmt.Errorf("spec.parameters validation failed: %w", err)
		}
	}
	
	// Validate constraints
	if constraints, exists := spec["constraints"]; exists {
		if err := w.validateConstraints(constraints); err != nil {
			return fmt.Errorf("spec.constraints validation failed: %w", err)
		}
	}
	
	return nil
}

// validateTargetSpec validates target configuration
func (w *Watcher) validateTargetSpec(target interface{}) error {
	targetMap, ok := target.(map[string]interface{})
	if !ok {
		return fmt.Errorf("target must be an object")
	}
	
	// Validate required target fields
	if targetType, exists := targetMap["type"]; exists {
		if typeStr, ok := targetType.(string); !ok || typeStr == "" {
			return fmt.Errorf("target.type must be a non-empty string")
		}
	}
	
	if name, exists := targetMap["name"]; exists {
		if nameStr, ok := name.(string); !ok || nameStr == "" {
			return fmt.Errorf("target.name must be a non-empty string")
		}
	}
	
	return nil
}

// validateNetworkParameters validates network function parameters
func (w *Watcher) validateNetworkParameters(parameters interface{}) error {
	paramMap, ok := parameters.(map[string]interface{})
	if !ok {
		return fmt.Errorf("parameters must be an object")
	}
	
	// Validate network functions if present
	if nfs, exists := paramMap["networkFunctions"]; exists {
		if nfList, ok := nfs.([]interface{}); ok {
			for i, nf := range nfList {
				if err := w.validateNetworkFunction(nf); err != nil {
					return fmt.Errorf("networkFunctions[%d] validation failed: %w", i, err)
				}
			}
		} else {
			return fmt.Errorf("networkFunctions must be an array")
		}
	}
	
	// Validate connectivity if present
	if connectivity, exists := paramMap["connectivity"]; exists {
		if err := w.validateConnectivity(connectivity); err != nil {
			return fmt.Errorf("connectivity validation failed: %w", err)
		}
	}
	
	// Validate SLA if present
	if sla, exists := paramMap["sla"]; exists {
		if err := w.validateSLA(sla); err != nil {
			return fmt.Errorf("sla validation failed: %w", err)
		}
	}
	
	return nil
}

// validateNetworkFunction validates individual network function configuration
func (w *Watcher) validateNetworkFunction(nf interface{}) error {
	nfMap, ok := nf.(map[string]interface{})
	if !ok {
		return fmt.Errorf("network function must be an object")
	}
	
	// Validate required fields
	requiredFields := []string{"name", "type"}
	for _, field := range requiredFields {
		if value, exists := nfMap[field]; !exists {
			return fmt.Errorf("missing required field: %s", field)
		} else if str, ok := value.(string); !ok || str == "" {
			return fmt.Errorf("%s must be a non-empty string", field)
		}
	}
	
	// Validate resources if present
	if resources, exists := nfMap["resources"]; exists {
		if err := w.validateResourceRequirements(resources); err != nil {
			return fmt.Errorf("resources validation failed: %w", err)
		}
	}
	
	return nil
}

// validateResourceRequirements validates resource requirements
func (w *Watcher) validateResourceRequirements(resources interface{}) error {
	resMap, ok := resources.(map[string]interface{})
	if !ok {
		return fmt.Errorf("resources must be an object")
	}
	
	// Validate resource fields format
	resourceFields := []string{"cpu", "memory", "storage"}
	for _, field := range resourceFields {
		if value, exists := resMap[field]; exists {
			if str, ok := value.(string); !ok || str == "" {
				return fmt.Errorf("%s must be a non-empty string", field)
			}
			// Could add more specific validation for Kubernetes resource format here
		}
	}
	
	return nil
}

// validateConnectivity validates connectivity configuration
func (w *Watcher) validateConnectivity(connectivity interface{}) error {
	connMap, ok := connectivity.(map[string]interface{})
	if !ok {
		return fmt.Errorf("connectivity must be an object")
	}
	
	// Validate connectivity sections
	for section, config := range connMap {
		if configMap, ok := config.(map[string]interface{}); ok {
			// Validate interface field
			if iface, exists := configMap["interface"]; exists {
				if ifaceStr, ok := iface.(string); !ok || ifaceStr == "" {
					return fmt.Errorf("%s.interface must be a non-empty string", section)
				}
			}
		} else {
			return fmt.Errorf("%s must be an object", section)
		}
	}
	
	return nil
}

// validateSLA validates SLA configuration
func (w *Watcher) validateSLA(sla interface{}) error {
	slaMap, ok := sla.(map[string]interface{})
	if !ok {
		return fmt.Errorf("sla must be an object")
	}
	
	// Validate SLA fields are strings (could be more specific)
	slaFields := []string{"availability", "throughput", "latency", "jitter"}
	for _, field := range slaFields {
		if value, exists := slaMap[field]; exists {
			if str, ok := value.(string); !ok || str == "" {
				return fmt.Errorf("sla.%s must be a non-empty string", field)
			}
		}
	}
	
	return nil
}

// validateConstraints validates constraints configuration
func (w *Watcher) validateConstraints(constraints interface{}) error {
	constMap, ok := constraints.(map[string]interface{})
	if !ok {
		return fmt.Errorf("constraints must be an object")
	}
	
	// Validate placement constraints
	if placement, exists := constMap["placement"]; exists {
		if _, ok := placement.(map[string]interface{}); !ok {
			return fmt.Errorf("constraints.placement must be an object")
		}
	}
	
	// Validate security constraints
	if security, exists := constMap["security"]; exists {
		if _, ok := security.(map[string]interface{}); !ok {
			return fmt.Errorf("constraints.security must be an object")
		}
	}
	
	return nil
}

// validateLabels validates Kubernetes labels
func (w *Watcher) validateLabels(labels interface{}) error {
	labelMap, ok := labels.(map[string]interface{})
	if !ok {
		return fmt.Errorf("labels must be an object")
	}
	
	for key, value := range labelMap {
		// Validate label key
		if !isValidLabelKey(key) {
			return fmt.Errorf("invalid label key: %s", key)
		}
		
		// Validate label value
		if valueStr, ok := value.(string); !ok {
			return fmt.Errorf("label value must be a string")
		} else if !isValidLabelValue(valueStr) {
			return fmt.Errorf("invalid label value: %s", valueStr)
		}
	}
	
	return nil
}

// validateResourceIntentSpec validates ResourceIntent spec (placeholder)
func (w *Watcher) validateResourceIntentSpec(spec map[string]interface{}) error {
	// Basic validation for ResourceIntent
	if action, exists := spec["action"]; exists {
		if actionStr, ok := action.(string); !ok || actionStr == "" {
			return fmt.Errorf("spec.action must be a non-empty string")
		}
	}
	return nil
}

// validateScalingIntentSpec validates ScalingIntent spec (placeholder)
func (w *Watcher) validateScalingIntentSpec(spec map[string]interface{}) error {
	// Basic validation for ScalingIntent
	if replicas, exists := spec["replicas"]; exists {
		if replicasNum, ok := replicas.(float64); !ok || replicasNum < 1 {
			return fmt.Errorf("spec.replicas must be a positive number")
		}
	}
	return nil
}

// validateDeploymentIntentSpec validates DeploymentIntent spec (placeholder)
func (w *Watcher) validateDeploymentIntentSpec(spec map[string]interface{}) error {
	// Basic validation for DeploymentIntent
	if target, exists := spec["target"]; exists {
		if targetStr, ok := target.(string); !ok || targetStr == "" {
			return fmt.Errorf("spec.target must be a non-empty string")
		}
	}
	return nil
}

// validateServiceIntentSpec validates ServiceIntent spec (placeholder)
func (w *Watcher) validateServiceIntentSpec(spec map[string]interface{}) error {
	// Basic validation for ServiceIntent
	if ports, exists := spec["ports"]; exists {
		if _, ok := ports.([]interface{}); !ok {
			return fmt.Errorf("spec.ports must be an array")
		}
	}
	return nil
}

// validateGenericSpec performs basic validation for unknown spec types
func (w *Watcher) validateGenericSpec(spec map[string]interface{}) error {
	// Perform basic validation that can apply to any spec
	if len(spec) == 0 {
		return fmt.Errorf("spec cannot be empty")
	}
	return nil
}

// isValidKubernetesName validates Kubernetes resource names
func isValidKubernetesName(name string) bool {
	if len(name) == 0 || len(name) > 253 {
		return false
	}
	
	// Kubernetes names must be lowercase alphanumeric, hyphens, and dots
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' || char == '.') {
			return false
		}
	}
	
	// Cannot start or end with hyphen or dot
	return name[0] != '-' && name[0] != '.' && name[len(name)-1] != '-' && name[len(name)-1] != '.'
}

// isValidLabelKey validates Kubernetes label keys
func isValidLabelKey(key string) bool {
	if len(key) == 0 || len(key) > 253 {
		return false
	}
	
	// Split on '/' to check prefix and name separately
	parts := strings.Split(key, "/")
	if len(parts) > 2 {
		return false
	}
	
	// Validate each part
	for _, part := range parts {
		if !isValidKubernetesName(part) {
			return false
		}
	}
	
	return true
}

// isValidLabelValue validates Kubernetes label values
func isValidLabelValue(value string) bool {
	if len(value) > 63 {
		return false
	}
	
	// Empty values are allowed
	if len(value) == 0 {
		return true
	}
	
	// Must be alphanumeric, hyphens, underscores, dots
	for _, char := range value {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || 
			(char >= '0' && char <= '9') || char == '-' || char == '_' || char == '.') {
			return false
		}
	}
	
	// Cannot start or end with non-alphanumeric characters
	first := value[0]
	last := value[len(value)-1]
	return ((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || (first >= '0' && first <= '9')) &&
		((last >= 'a' && last <= 'z') || (last >= 'A' && last <= 'Z') || (last >= '0' && last <= '9'))
}

// safeMarshalJSON marshals JSON with size limits
func (w *Watcher) safeMarshalJSON(v interface{}, maxSize int) ([]byte, error) {
	// Use a buffer with size limit
	var buf bytes.Buffer
	buf.Grow(4096) // Pre-allocate reasonable size
	
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	
	if err := encoder.Encode(v); err != nil {
		return nil, fmt.Errorf("JSON encoding failed: %w", err)
	}
	
	data := buf.Bytes()
	if len(data) > maxSize {
		return nil, fmt.Errorf("data too large: %d bytes > %d limit", len(data), maxSize)
	}
	
	// Remove trailing newline added by encoder
	if len(data) > 0 && data[len(data)-1] == '\n' {
		data = data[:len(data)-1]
	}
	
	return data, nil
}

// GetStats returns processing statistics (only if using legacy approach)
func (w *Watcher) GetStats() (ProcessingStats, error) {
	if w.fileManager != nil {
		return w.fileManager.GetStats()
	}
	return ProcessingStats{}, fmt.Errorf("stats not available when using IntentProcessor approach")
}

// GetMetrics returns a copy of the current metrics
func (w *Watcher) GetMetrics() *WatcherMetrics {
	if w == nil {
		return nil
	}
	if w.metrics == nil {
		return &WatcherMetrics{}
	}
	
	w.metrics.mu.RLock()
	defer w.metrics.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	return &WatcherMetrics{
		FilesProcessedTotal:     atomic.LoadInt64(&w.metrics.FilesProcessedTotal),
		FilesFailedTotal:        atomic.LoadInt64(&w.metrics.FilesFailedTotal),
		ValidationFailuresTotal: atomic.LoadInt64(&w.metrics.ValidationFailuresTotal),
		RetryAttemptsTotal:      atomic.LoadInt64(&w.metrics.RetryAttemptsTotal),
		QueueDepthCurrent:       atomic.LoadInt64(&w.metrics.QueueDepthCurrent),
		BackpressureEventsTotal: atomic.LoadInt64(&w.metrics.BackpressureEventsTotal),
		TimeoutCount:            atomic.LoadInt64(&w.metrics.TimeoutCount),
		MemoryUsageBytes:        atomic.LoadInt64(&w.metrics.MemoryUsageBytes),
		GoroutineCount:          atomic.LoadInt64(&w.metrics.GoroutineCount),
		DirectorySizeBytes:      atomic.LoadInt64(&w.metrics.DirectorySizeBytes),
		ThroughputFilesPerSecond: w.metrics.ThroughputFilesPerSecond,
		AverageProcessingTime:    w.metrics.AverageProcessingTime,
		WorkerUtilization:        w.metrics.WorkerUtilization,
		StartTime:               w.metrics.StartTime,
		LastUpdateTime:          w.metrics.LastUpdateTime,
		MetricsEnabled:          w.metrics.MetricsEnabled,
	}
}

// getOrCreateFileLock gets or creates a file-level mutex for the given path
func (w *Watcher) getOrCreateFileLock(filePath string) *sync.Mutex {
	w.fileState.mu.Lock()
	defer w.fileState.mu.Unlock()
	
	if lock, exists := w.fileState.processing[filePath]; exists {
		return lock
	}
	
	// Create new mutex for this file
	lock := &sync.Mutex{}
	w.fileState.processing[filePath] = lock
	return lock
}

// retryWithBackoff implements exponential backoff for work item retry
func (w *Watcher) retryWithBackoff(workItem WorkItem, cancelFunc context.CancelFunc) {
	maxRetries := 3
	baseDelay := 100 * time.Millisecond
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Record retry attempt
		atomic.AddInt64(&w.metrics.RetryAttemptsTotal, 1)
		
		delay := time.Duration(attempt*attempt) * baseDelay
		timer := time.NewTimer(delay)
		
		select {
		case <-w.ctx.Done():
			timer.Stop()
			cancelFunc()
			return
		case <-timer.C:
			// Try to queue again
			select {
			case w.workerPool.workQueue <- workItem:
				log.Printf("LOOP:RETRY_SUCCESS - Queued %s after %d attempts", 
					filepath.Base(workItem.FilePath), attempt)
				return
			case <-w.ctx.Done():
				cancelFunc()
				return
			default:
				if attempt == maxRetries {
					log.Printf("LOOP:RETRY_FAILED - Failed to queue %s after %d attempts", 
						filepath.Base(workItem.FilePath), maxRetries)
					cancelFunc()
					return
				}
				// Continue to next retry
			}
		}
	}
}

// cleanupOldFileState performs a single cleanup of old file state entries (for testing)
func (w *Watcher) cleanupOldFileState() {
	w.fileState.mu.Lock()
	defer w.fileState.mu.Unlock()
	
	now := time.Now()
	cleanedEvents := 0
	cleanedLocks := 0
	
	// Clean up old events (older than 1 minute)
	for filePath, timestamp := range w.fileState.recentEvents {
		if now.Sub(timestamp) > time.Minute {
			delete(w.fileState.recentEvents, filePath)
			cleanedEvents++
		}
	}
	
	// Clean up old processing locks (older than 30 seconds)  
	for filePath, _ := range w.fileState.processing {
		// For simplicity, we remove locks that exist but we don't track their creation time
		// In a real scenario, you might want to add timestamps to track lock age
		delete(w.fileState.processing, filePath)
		cleanedLocks++
	}
}

// fileStateCleanupRoutine periodically cleans up old file state entries
func (w *Watcher) fileStateCleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.cleanupOldFileState()
		}
	}
}

// writeStatusFileAtomic atomically writes a status file with directory creation
func (w *Watcher) writeStatusFileAtomic(intentFile, status, message string) {
	// Truncate message if too large
	if len(message) > MaxMessageSize {
		message = message[:MaxMessageSize-3] + "..."
		log.Printf("Warning: Status message truncated for %s", filepath.Base(intentFile))
	}
	
	statusData := map[string]interface{}{
		"intent_file":  filepath.Base(intentFile),
		"status":       status,
		"message":      message,
		"timestamp":    time.Now().Format(time.RFC3339),
		"processed_by": "conductor-loop",
		"mode":         w.config.Mode,
		"porch_path":   w.config.PorchPath,
		"worker_id":    fmt.Sprintf("worker-%d", atomic.LoadInt64(&w.workerPool.activeWorkers)),
	}
	
	// Use safe JSON marshaling with size limits
	data, err := w.safeMarshalJSON(statusData, MaxStatusSize)
	if err != nil {
		log.Printf("Failed to marshal status data: %v", err)
		return
	}
	
	// Create versioned status filename based on intent filename
	baseName := filepath.Base(intentFile)
	timestamp := time.Now().Format("20060102-150405")
	statusFile := filepath.Join(w.dir, "status", fmt.Sprintf("%s-%s.status", baseName, timestamp))
	
	// Ensure status directory exists using enhanced directory manager
	statusDir := filepath.Dir(statusFile)
	w.ensureDirectoryExists(statusDir)
	
	// Write atomically using temp file + rename
	tempFile := statusFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		log.Printf("Failed to write temporary status file: %v", err)
		return
	}
	
	// Atomic rename
	if err := os.Rename(tempFile, statusFile); err != nil {
		os.Remove(tempFile) // Clean up on failure
		log.Printf("Failed to rename temporary status file: %v", err)
		return
	}
	
	log.Printf("Status written atomically to: %s", statusFile)
}

// ensureDirectoryExists ensures a directory exists using sync.Once pattern
func (w *Watcher) ensureDirectoryExists(dir string) {
	w.dirManager.mu.RLock()
	onceFunc, exists := w.dirManager.dirOnce[dir]
	w.dirManager.mu.RUnlock()
	
	if !exists {
		w.dirManager.mu.Lock()
		// Double-check locking pattern
		if onceFunc, exists = w.dirManager.dirOnce[dir]; !exists {
			onceFunc = &sync.Once{}
			w.dirManager.dirOnce[dir] = onceFunc
		}
		w.dirManager.mu.Unlock()
	}
	
	// Use sync.Once to ensure directory is created only once
	onceFunc.Do(func() {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("Failed to create directory %s: %v", dir, err)
		} else {
			log.Printf("Created directory: %s", dir)
		}
	})
}

// IsIntentFile checks if a filename matches the intent file pattern
func IsIntentFile(filename string) bool {
	// Check if file matches intent-*.json pattern
	return strings.HasPrefix(filename, "intent-") && strings.HasSuffix(filename, ".json")
}

// isFileStable checks if a file exists and is stable (not being written to)
func (w *Watcher) isFileStable(filePath string) bool {
	// Check if file exists
	stat1, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	
	// Wait a small amount of time and check again
	time.Sleep(50 * time.Millisecond)
	
	stat2, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	
	// File is stable if size and modification time haven't changed
	return stat1.Size() == stat2.Size() && stat1.ModTime().Equal(stat2.ModTime())
}