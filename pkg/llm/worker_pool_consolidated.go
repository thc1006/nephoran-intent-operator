package llm

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// WorkerPool provides unified worker management with retry and performance optimization
type WorkerPool struct {
	// Configuration
	config *WorkerPoolConfig
	logger *slog.Logger

	// Worker management
	workers     []*Worker
	workQueue   chan Task
	resultQueue chan *TaskResult

	// Retry engine integration
	retryEngine *RetryEngine

	// Performance optimization
	optimizer *PerformanceOptimizer

	// State management
	isRunning      atomic.Bool
	activeWorkers  atomic.Int64
	processedTasks atomic.Int64
	failedTasks    atomic.Int64

	// Synchronization
	mutex  sync.RWMutex
	wg     sync.WaitGroup
	stopCh chan struct{}

	// Metrics and monitoring
	metrics       *WorkerPoolMetrics
	healthChecker *HealthChecker
}

// WorkerPoolConfig holds configuration for the worker pool
type WorkerPoolConfig struct {
	// Basic configuration
	MinWorkers  int           `json:"min_workers"`
	MaxWorkers  int           `json:"max_workers"`
	QueueSize   int           `json:"queue_size"`
	TaskTimeout time.Duration `json:"task_timeout"`

	// Scaling configuration
	ScalingEnabled     bool          `json:"scaling_enabled"`
	ScaleUpThreshold   float64       `json:"scale_up_threshold"`
	ScaleDownThreshold float64       `json:"scale_down_threshold"`
	ScaleCheckInterval time.Duration `json:"scale_check_interval"`

	// Performance optimization
	EnableOptimization bool `json:"enable_optimization"`
	OptimizationLevel  int  `json:"optimization_level"`

	// Health checking
	HealthCheckEnabled  bool          `json:"health_check_enabled"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`

	// Retry configuration
	RetryConfig *RetryEngineConfig `json:"retry_config"`
}

// Worker represents an individual worker
type Worker struct {
	ID             int
	workQueue      chan Task
	resultQueue    chan *TaskResult
	logger         *slog.Logger
	isHealthy      atomic.Bool
	lastTaskTime   time.Time
	processedCount atomic.Int64
	errorCount     atomic.Int64
	stopCh         chan struct{}
	mutex          sync.RWMutex
}

// Task represents a unit of work
type Task struct {
	ID         string                 `json:"id"`
	Type       TaskType               `json:"type"`
	Intent     string                 `json:"intent"`
	Parameters map[string]interface{} `json:"parameters"`
	Priority   Priority               `json:"priority"`
	Context    context.Context        `json:"-"`
	SubmitTime time.Time              `json:"submit_time"`
	RetryCount int                    `json:"retry_count"`
	MaxRetries int                    `json:"max_retries"`
	Callback   TaskCallback           `json:"-"`
}

// TaskType defines the type of task
type TaskType int

const (
	TaskTypeLLMProcessing TaskType = iota
	TaskTypeRAGProcessing
	TaskTypeBatchProcessing
	TaskTypeStreamProcessing
)

// Priority defines task priority levels
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// TaskCallback defines the callback function for task completion
type TaskCallback func(*TaskResult)

// TaskResult represents the result of task execution
type TaskResult struct {
	TaskID         string                 `json:"task_id"`
	Success        bool                   `json:"success"`
	Result         interface{}            `json:"result"`
	Error          error                  `json:"error,omitempty"`
	ProcessingTime time.Duration          `json:"processing_time"`
	WorkerID       int                    `json:"worker_id"`
	RetryCount     int                    `json:"retry_count"`
	Metadata       map[string]interface{} `json:"metadata"`
	Timestamp      time.Time              `json:"timestamp"`
}

// WorkerPoolMetrics tracks pool performance
type WorkerPoolMetrics struct {
	ActiveWorkers         int64         `json:"active_workers"`
	QueueSize             int64         `json:"queue_size"`
	ProcessedTasks        int64         `json:"processed_tasks"`
	FailedTasks           int64         `json:"failed_tasks"`
	AverageProcessingTime time.Duration `json:"average_processing_time"`
	ThroughputPerSecond   float64       `json:"throughput_per_second"`
	SuccessRate           float64       `json:"success_rate"`
	LastScaleEvent        time.Time     `json:"last_scale_event"`
	mutex                 sync.RWMutex
}

// RetryEngine handles retry logic with exponential backoff
type RetryEngine struct {
	config  *RetryEngineConfig
	logger  *slog.Logger
	metrics *RetryMetrics
}

// RetryEngineConfig defines retry behavior
type RetryEngineConfig struct {
	MaxRetries      int           `json:"max_retries"`
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffFactor   float64       `json:"backoff_factor"`
	JitterEnabled   bool          `json:"jitter_enabled"`
	RetryConditions []string      `json:"retry_conditions"`
}

// RetryMetrics tracks retry performance
type RetryMetrics struct {
	TotalRetries      int64   `json:"total_retries"`
	SuccessfulRetries int64   `json:"successful_retries"`
	FailedRetries     int64   `json:"failed_retries"`
	RetrySuccessRate  float64 `json:"retry_success_rate"`
	mutex             sync.RWMutex
}

// PerformanceOptimizer handles performance optimization
type PerformanceOptimizer struct {
	config        *OptimizerConfig
	logger        *slog.Logger
	metrics       *OptimizerMetrics
	optimizations []Optimization
}

// OptimizerConfig defines optimization settings
type OptimizerConfig struct {
	Enabled              bool          `json:"enabled"`
	OptimizationInterval time.Duration `json:"optimization_interval"`
	PerformanceThreshold float64       `json:"performance_threshold"`
	AutoTuningEnabled    bool          `json:"auto_tuning_enabled"`
}

// OptimizerMetrics tracks optimization performance
type OptimizerMetrics struct {
	OptimizationsApplied   int64   `json:"optimizations_applied"`
	PerformanceImprovement float64 `json:"performance_improvement"`
	mutex                  sync.RWMutex
}

// Optimization represents a performance optimization
type Optimization struct {
	Name        string
	Description string
	ApplyFunc   func() error
	RevertFunc  func() error
}

// HealthChecker monitors worker pool health
type HealthChecker struct {
	config  *HealthCheckConfig
	logger  *slog.Logger
	metrics *HealthMetrics
	stopCh  chan struct{}
}

// HealthCheckConfig defines health check settings
type HealthCheckConfig struct {
	Enabled            bool          `json:"enabled"`
	CheckInterval      time.Duration `json:"check_interval"`
	UnhealthyThreshold int           `json:"unhealthy_threshold"`
	RecoveryTimeout    time.Duration `json:"recovery_timeout"`
}

// HealthMetrics tracks health status
type HealthMetrics struct {
	HealthyWorkers   int64     `json:"healthy_workers"`
	UnhealthyWorkers int64     `json:"unhealthy_workers"`
	LastHealthCheck  time.Time `json:"last_health_check"`
	mutex            sync.RWMutex
}

// NewWorkerPool creates a new worker pool with integrated retry and optimization
func NewWorkerPool(config *WorkerPoolConfig) (*WorkerPool, error) {
	if config == nil {
		config = getDefaultWorkerPoolConfig()
	}

	// Validate configuration
	if err := validateWorkerPoolConfig(config); err != nil {
		return nil, fmt.Errorf("invalid worker pool configuration: %w", err)
	}

	pool := &WorkerPool{
		config:      config,
		logger:      slog.Default().With("component", "worker-pool"),
		workQueue:   make(chan Task, config.QueueSize),
		resultQueue: make(chan *TaskResult, config.QueueSize),
		stopCh:      make(chan struct{}),
		metrics:     &WorkerPoolMetrics{},
	}

	// Initialize retry engine
	if config.RetryConfig != nil {
		pool.retryEngine = &RetryEngine{
			config:  config.RetryConfig,
			logger:  pool.logger.With("component", "retry-engine"),
			metrics: &RetryMetrics{},
		}
	}

	// Initialize performance optimizer
	if config.EnableOptimization {
		pool.optimizer = &PerformanceOptimizer{
			config: &OptimizerConfig{
				Enabled:              true,
				OptimizationInterval: 5 * time.Minute,
				PerformanceThreshold: 0.8,
				AutoTuningEnabled:    true,
			},
			logger:  pool.logger.With("component", "performance-optimizer"),
			metrics: &OptimizerMetrics{},
		}
	}

	// Initialize health checker
	if config.HealthCheckEnabled {
		pool.healthChecker = &HealthChecker{
			config: &HealthCheckConfig{
				Enabled:            true,
				CheckInterval:      config.HealthCheckInterval,
				UnhealthyThreshold: 3,
				RecoveryTimeout:    30 * time.Second,
			},
			logger:  pool.logger.With("component", "health-checker"),
			metrics: &HealthMetrics{},
			stopCh:  make(chan struct{}),
		}
	}

	return pool, nil
}

// getDefaultWorkerPoolConfig returns default configuration
func getDefaultWorkerPoolConfig() *WorkerPoolConfig {
	return &WorkerPoolConfig{
		MinWorkers:          2,
		MaxWorkers:          10,
		QueueSize:           1000,
		TaskTimeout:         30 * time.Second,
		ScalingEnabled:      true,
		ScaleUpThreshold:    0.8,
		ScaleDownThreshold:  0.2,
		ScaleCheckInterval:  30 * time.Second,
		EnableOptimization:  true,
		OptimizationLevel:   2,
		HealthCheckEnabled:  true,
		HealthCheckInterval: 10 * time.Second,
		RetryConfig: &RetryEngineConfig{
			MaxRetries:      3,
			InitialDelay:    time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   2.0,
			JitterEnabled:   true,
			RetryConditions: []string{"timeout", "connection_error", "service_unavailable"},
		},
	}
}

// validateWorkerPoolConfig validates the worker pool configuration
func validateWorkerPoolConfig(config *WorkerPoolConfig) error {
	if config.MinWorkers < 1 {
		return fmt.Errorf("minimum workers must be at least 1")
	}
	if config.MaxWorkers < config.MinWorkers {
		return fmt.Errorf("maximum workers must be greater than or equal to minimum workers")
	}
	if config.QueueSize < 1 {
		return fmt.Errorf("queue size must be at least 1")
	}
	if config.TaskTimeout <= 0 {
		return fmt.Errorf("task timeout must be positive")
	}
	return nil
}

// Start initializes and starts the worker pool
func (wp *WorkerPool) Start(ctx context.Context) error {
	if wp.isRunning.Load() {
		return fmt.Errorf("worker pool is already running")
	}

	wp.logger.Info("Starting worker pool",
		slog.Int("min_workers", wp.config.MinWorkers),
		slog.Int("max_workers", wp.config.MaxWorkers),
		slog.Int("queue_size", wp.config.QueueSize),
	)

	// Create initial workers
	for i := 0; i < wp.config.MinWorkers; i++ {
		worker := wp.createWorker(i)
		wp.workers = append(wp.workers, worker)
		go worker.start(ctx)
	}

	wp.activeWorkers.Store(int64(wp.config.MinWorkers))
	wp.isRunning.Store(true)

	// Start background processes
	if wp.config.ScalingEnabled {
		go wp.autoScaler(ctx)
	}

	if wp.optimizer != nil {
		go wp.optimizer.start(ctx)
	}

	if wp.healthChecker != nil {
		go wp.healthChecker.start(ctx, wp.workers)
	}

	go wp.metricsCollector(ctx)

	wp.logger.Info("Worker pool started successfully")
	return nil
}

// Submit adds a task to the worker pool
func (wp *WorkerPool) Submit(task Task) error {
	if !wp.isRunning.Load() {
		return fmt.Errorf("worker pool is not running")
	}

	task.SubmitTime = time.Now()

	select {
	case wp.workQueue <- task:
		return nil
	default:
		return fmt.Errorf("work queue is full")
	}
}

// createWorker creates a new worker
func (wp *WorkerPool) createWorker(id int) *Worker {
	return &Worker{
		ID:          id,
		workQueue:   wp.workQueue,
		resultQueue: wp.resultQueue,
		logger:      wp.logger.With("worker_id", id),
		stopCh:      make(chan struct{}),
	}
}

// start runs the worker processing loop
func (w *Worker) start(ctx context.Context) {
	w.isHealthy.Store(true)
	w.logger.Debug("Worker started")

	for {
		select {
		case <-ctx.Done():
			w.logger.Debug("Worker stopping due to context cancellation")
			return
		case <-w.stopCh:
			w.logger.Debug("Worker stopping due to stop signal")
			return
		case task := <-w.workQueue:
			w.processTask(ctx, task)
		}
	}
}

// processTask processes a single task
func (w *Worker) processTask(ctx context.Context, task Task) {
	start := time.Now()
	w.lastTaskTime = start

	w.logger.Debug("Processing task", slog.String("task_id", task.ID), slog.String("type", fmt.Sprintf("%d", task.Type)))

	// Create task context with timeout
	taskCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Process the task based on its type
	result := &TaskResult{
		TaskID:     task.ID,
		WorkerID:   w.ID,
		Timestamp:  start,
		RetryCount: task.RetryCount,
		Metadata:   make(map[string]interface{}),
	}

	// Execute the task
	var err error
	result.Result, err = w.executeTask(taskCtx, task)
	result.ProcessingTime = time.Since(start)

	if err != nil {
		result.Success = false
		result.Error = err
		w.errorCount.Add(1)
		w.logger.Error("Task failed", slog.String("task_id", task.ID), slog.String("error", err.Error()))
	} else {
		result.Success = true
		w.processedCount.Add(1)
		w.logger.Debug("Task completed successfully", slog.String("task_id", task.ID))
	}

	// Send result
	select {
	case w.resultQueue <- result:
	default:
		w.logger.Warn("Result queue full, dropping result", slog.String("task_id", task.ID))
	}

	// Execute callback if provided
	if task.Callback != nil {
		go task.Callback(result)
	}
}

// executeTask executes the actual task logic
func (w *Worker) executeTask(ctx context.Context, task Task) (interface{}, error) {
	switch task.Type {
	case TaskTypeLLMProcessing:
		return w.processLLMTask(ctx, task)
	case TaskTypeRAGProcessing:
		return w.processRAGTask(ctx, task)
	case TaskTypeBatchProcessing:
		return w.processBatchTask(ctx, task)
	case TaskTypeStreamProcessing:
		return w.processStreamTask(ctx, task)
	default:
		return nil, fmt.Errorf("unknown task type: %d", task.Type)
	}
}

// Task processing methods for different types
func (w *Worker) processLLMTask(ctx context.Context, task Task) (interface{}, error) {
	time.Sleep(100 * time.Millisecond) // Simulate processing
	return map[string]interface{}{
		"response": "Processed LLM task: " + task.Intent,
		"tokens":   150,
	}, nil
}

func (w *Worker) processRAGTask(ctx context.Context, task Task) (interface{}, error) {
	time.Sleep(200 * time.Millisecond) // Simulate processing
	return map[string]interface{}{
		"response":   "Processed RAG task: " + task.Intent,
		"confidence": 0.92,
		"sources":    3,
	}, nil
}

func (w *Worker) processBatchTask(ctx context.Context, task Task) (interface{}, error) {
	time.Sleep(50 * time.Millisecond) // Simulate processing
	return map[string]interface{}{
		"batch_results": []string{"result1", "result2", "result3"},
		"batch_size":    3,
	}, nil
}

func (w *Worker) processStreamTask(ctx context.Context, task Task) (interface{}, error) {
	time.Sleep(75 * time.Millisecond) // Simulate processing
	return map[string]interface{}{
		"stream_id": "stream_123",
		"chunks":    5,
	}, nil
}

// Auto-scaling methods
func (wp *WorkerPool) autoScaler(ctx context.Context) {
	ticker := time.NewTicker(wp.config.ScaleCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-wp.stopCh:
			return
		case <-ticker.C:
			wp.checkAndScale()
		}
	}
}

func (wp *WorkerPool) checkAndScale() {
	queueSize := len(wp.workQueue)
	queueUtilization := float64(queueSize) / float64(wp.config.QueueSize)
	currentWorkers := int(wp.activeWorkers.Load())

	// Scale up if queue utilization is high
	if queueUtilization > wp.config.ScaleUpThreshold && currentWorkers < wp.config.MaxWorkers {
		wp.scaleUp()
	}

	// Scale down if queue utilization is low
	if queueUtilization < wp.config.ScaleDownThreshold && currentWorkers > wp.config.MinWorkers {
		wp.scaleDown()
	}
}

func (wp *WorkerPool) scaleUp() {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()

	newWorkerID := len(wp.workers)
	worker := wp.createWorker(newWorkerID)
	wp.workers = append(wp.workers, worker)

	go worker.start(context.Background())

	wp.activeWorkers.Add(1)
	wp.logger.Info("Scaled up worker pool", slog.Int("new_worker_id", newWorkerID))
}

func (wp *WorkerPool) scaleDown() {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()

	if len(wp.workers) <= wp.config.MinWorkers {
		return
	}

	// Stop the last worker
	lastWorker := wp.workers[len(wp.workers)-1]
	close(lastWorker.stopCh)
	wp.workers = wp.workers[:len(wp.workers)-1]

	wp.activeWorkers.Add(-1)
	wp.logger.Info("Scaled down worker pool", slog.Int("removed_worker_id", lastWorker.ID))
}

// Metrics collection
func (wp *WorkerPool) metricsCollector(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-wp.stopCh:
			return
		case <-ticker.C:
			wp.updatePoolMetrics()
		}
	}
}

func (wp *WorkerPool) updatePoolMetrics() {
	wp.updateMetrics(func(m *WorkerPoolMetrics) {
		m.ActiveWorkers = wp.activeWorkers.Load()
		m.QueueSize = int64(len(wp.workQueue))
		m.ProcessedTasks = wp.processedTasks.Load()
		m.FailedTasks = wp.failedTasks.Load()

		totalTasks := m.ProcessedTasks + m.FailedTasks
		if totalTasks > 0 {
			m.SuccessRate = float64(m.ProcessedTasks) / float64(totalTasks)
		}
	})
}

func (wp *WorkerPool) updateMetrics(updater func(*WorkerPoolMetrics)) {
	wp.metrics.mutex.Lock()
	defer wp.metrics.mutex.Unlock()
	updater(wp.metrics)
}

// GetMetrics returns current worker pool metrics
func (wp *WorkerPool) GetMetrics() *WorkerPoolMetrics {
	wp.metrics.mutex.RLock()
	defer wp.metrics.mutex.RUnlock()

	metrics := *wp.metrics
	return &metrics
}

// GetStatus returns the current status of the worker pool
func (wp *WorkerPool) GetStatus() map[string]interface{} {
	metrics := wp.GetMetrics()

	return map[string]interface{}{
		"running":         wp.isRunning.Load(),
		"active_workers":  metrics.ActiveWorkers,
		"queue_size":      metrics.QueueSize,
		"processed_tasks": metrics.ProcessedTasks,
		"failed_tasks":    metrics.FailedTasks,
		"success_rate":    metrics.SuccessRate,
		"queue_capacity":  wp.config.QueueSize,
		"min_workers":     wp.config.MinWorkers,
		"max_workers":     wp.config.MaxWorkers,
	}
}

// Shutdown gracefully stops the worker pool
func (wp *WorkerPool) Shutdown(ctx context.Context) error {
	if !wp.isRunning.Load() {
		return fmt.Errorf("worker pool is not running")
	}

	wp.logger.Info("Shutting down worker pool")
	wp.isRunning.Store(false)

	// Stop all background processes
	close(wp.stopCh)

	// Stop health checker
	if wp.healthChecker != nil {
		close(wp.healthChecker.stopCh)
	}

	// Stop all workers
	wp.mutex.Lock()
	for _, worker := range wp.workers {
		close(worker.stopCh)
	}
	wp.mutex.Unlock()

	wp.logger.Info("Worker pool shutdown completed")
	return nil
}

// Performance optimizer methods
func (po *PerformanceOptimizer) start(ctx context.Context) {
	ticker := time.NewTicker(po.config.OptimizationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			po.optimizePerformance()
		}
	}
}

func (po *PerformanceOptimizer) optimizePerformance() {
	po.logger.Debug("Running performance optimization")

	// Apply optimizations
	for _, optimization := range po.optimizations {
		if err := optimization.ApplyFunc(); err != nil {
			po.logger.Error("Failed to apply optimization",
				slog.String("optimization", optimization.Name),
				slog.String("error", err.Error()))
		} else {
			po.updateMetrics(func(m *OptimizerMetrics) {
				m.OptimizationsApplied++
			})
		}
	}
}

func (po *PerformanceOptimizer) updateMetrics(updater func(*OptimizerMetrics)) {
	po.metrics.mutex.Lock()
	defer po.metrics.mutex.Unlock()
	updater(po.metrics)
}

// Health checker methods
func (hc *HealthChecker) start(ctx context.Context, workers []*Worker) {
	ticker := time.NewTicker(hc.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-hc.stopCh:
			return
		case <-ticker.C:
			hc.checkWorkerHealth(workers)
		}
	}
}

func (hc *HealthChecker) checkWorkerHealth(workers []*Worker) {
	healthyCount := int64(0)
	unhealthyCount := int64(0)

	for _, worker := range workers {
		if worker.isHealthy.Load() {
			healthyCount++
		} else {
			unhealthyCount++
			hc.logger.Warn("Unhealthy worker detected", slog.Int("worker_id", worker.ID))
		}
	}

	hc.updateMetrics(func(m *HealthMetrics) {
		m.HealthyWorkers = healthyCount
		m.UnhealthyWorkers = unhealthyCount
		m.LastHealthCheck = time.Now()
	})
}

func (hc *HealthChecker) updateMetrics(updater func(*HealthMetrics)) {
	hc.metrics.mutex.Lock()
	defer hc.metrics.mutex.Unlock()
	updater(hc.metrics)
}
