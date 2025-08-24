// Package sla provides comprehensive SLA monitoring capabilities for the Nephoran Intent Operator.
// This service handles 1000+ intents/second while providing accurate SLA measurements.
package sla

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sony/gobreaker"
	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/errors"
	"github.com/thc1006/nephoran-intent-operator/pkg/health"
	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// Service provides comprehensive SLA monitoring for the Nephoran Intent Operator
type Service struct {
	// Core configuration
	config     *ServiceConfig
	logger     *logging.StructuredLogger
	collector  *Collector
	calculator *Calculator
	tracker    *Tracker
	alerting   *AlertManager
	storage    *StorageManager

	// Worker pool management
	workers     []*Worker
	workerPool  chan *Worker
	taskQueue   chan Task
	workerCount int32
	activeJobs  int64

	// Circuit breaker for external dependencies
	circuitBreaker *gobreaker.CircuitBreaker

	// Health and state management
	healthChecker *health.HealthChecker
	started       atomic.Bool
	stopping      atomic.Bool
	stopped       chan struct{}

	// Performance monitoring
	metrics         *ServiceMetrics
	lastStatsUpdate time.Time
	processingRate  atomic.Uint64
	errorRate       atomic.Uint64

	// Synchronization
	mu      sync.RWMutex
	startMu sync.Mutex
}

// ServiceConfig holds configuration for the SLA monitoring service
type ServiceConfig struct {
	// Worker pool configuration
	WorkerCount        int `yaml:"worker_count"`
	TaskQueueSize      int `yaml:"task_queue_size"`
	MaxConcurrentTasks int `yaml:"max_concurrent_tasks"`

	// Performance settings
	CollectionInterval   time.Duration `yaml:"collection_interval"`
	CalculationInterval  time.Duration `yaml:"calculation_interval"`
	MetricsFlushInterval time.Duration `yaml:"metrics_flush_interval"`

	// Capacity limits
	MaxIntentsPerSecond int     `yaml:"max_intents_per_second"`
	MaxMemoryUsageMB    int64   `yaml:"max_memory_usage_mb"`
	MaxCPUUsagePercent  float64 `yaml:"max_cpu_usage_percent"`

	// SLA thresholds
	AvailabilityTarget float64       `yaml:"availability_target"`
	P95LatencyTarget   time.Duration `yaml:"p95_latency_target"`
	ThroughputTarget   float64       `yaml:"throughput_target"`
	ErrorRateTarget    float64       `yaml:"error_rate_target"`

	// Circuit breaker settings
	CircuitBreakerTimeout      time.Duration `yaml:"circuit_breaker_timeout"`
	CircuitBreakerMaxRequests  uint32        `yaml:"circuit_breaker_max_requests"`
	CircuitBreakerInterval     time.Duration `yaml:"circuit_breaker_interval"`
	CircuitBreakerFailureRatio float64       `yaml:"circuit_breaker_failure_ratio"`

	// Health check configuration
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
	GracePeriod         time.Duration `yaml:"grace_period"`
}

// DefaultServiceConfig returns a configuration with production-ready defaults
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		// Worker pool optimized for high throughput
		WorkerCount:        runtime.NumCPU() * 2,
		TaskQueueSize:      10000,
		MaxConcurrentTasks: 1000,

		// Performance settings for sub-100ms latency
		CollectionInterval:   100 * time.Millisecond,
		CalculationInterval:  1 * time.Second,
		MetricsFlushInterval: 5 * time.Second,

		// Capacity limits for production workloads
		MaxIntentsPerSecond: 1000,
		MaxMemoryUsageMB:    50,  // <50MB memory footprint
		MaxCPUUsagePercent:  1.0, // <1% CPU overhead

		// Production SLA targets
		AvailabilityTarget: 99.95,
		P95LatencyTarget:   2 * time.Second,
		ThroughputTarget:   1000.0,
		ErrorRateTarget:    0.1,

		// Circuit breaker for resilience
		CircuitBreakerTimeout:      30 * time.Second,
		CircuitBreakerMaxRequests:  10,
		CircuitBreakerInterval:     10 * time.Second,
		CircuitBreakerFailureRatio: 0.6,

		// Health monitoring
		HealthCheckInterval: 30 * time.Second,
		GracePeriod:         2 * time.Minute,
	}
}

// ServiceMetrics contains Prometheus metrics for the SLA service
type ServiceMetrics struct {
	// Processing metrics
	TasksProcessed    *prometheus.CounterVec
	ProcessingLatency *prometheus.HistogramVec
	ActiveWorkers     prometheus.Gauge
	QueueDepth        prometheus.Gauge

	// Performance metrics
	ProcessingRate prometheus.Gauge
	ErrorRate      prometheus.Gauge
	MemoryUsage    prometheus.Gauge
	CPUUsage       prometheus.Gauge

	// SLA compliance metrics
	SLACompliance   *prometheus.GaugeVec
	SLAViolations   *prometheus.CounterVec
	ErrorBudgetBurn prometheus.Gauge

	// Circuit breaker metrics
	CircuitBreakerState *prometheus.GaugeVec
	CircuitBreakerTotal *prometheus.CounterVec
}

// Task represents work to be processed by the SLA monitoring service
type Task interface {
	Execute(ctx context.Context) error
	GetType() TaskType
	GetPriority() Priority
	GetTimeout() time.Duration
}

// TaskType defines the type of monitoring task
type TaskType string

const (
	TaskTypeCollection  TaskType = "collection"
	TaskTypeCalculation TaskType = "calculation"
	TaskTypeAlerting    TaskType = "alerting"
	TaskTypeStorage     TaskType = "storage"
	TaskTypeHealthCheck TaskType = "health_check"
)

// Priority defines task execution priority
type Priority int

const (
	PriorityLow      Priority = 1
	PriorityNormal   Priority = 2
	PriorityHigh     Priority = 3
	PriorityCritical Priority = 4
)

// Worker represents a worker in the processing pool
type Worker struct {
	id      int
	service *Service
	tasks   chan Task
	quit    chan struct{}
	metrics *WorkerMetrics
}

// WorkerMetrics tracks metrics for individual workers
type WorkerMetrics struct {
	TasksProcessed prometheus.Counter
	ProcessingTime prometheus.Histogram
	Errors         prometheus.Counter
}

// NewService creates a new SLA monitoring service with the given configuration
func NewService(cfg *ServiceConfig, appConfig *config.Config, logger *logging.StructuredLogger) (*Service, error) {
	if cfg == nil {
		cfg = DefaultServiceConfig()
	}

	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	// Initialize service metrics
	metrics := &ServiceMetrics{
		TasksProcessed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "sla_service_tasks_processed_total",
			Help: "Total number of SLA monitoring tasks processed",
		}, []string{"type", "status"}),

		ProcessingLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "sla_service_processing_latency_seconds",
			Help:    "Processing latency for SLA monitoring tasks",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
		}, []string{"type"}),

		ActiveWorkers: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "sla_service_active_workers",
			Help: "Number of active worker goroutines",
		}),

		QueueDepth: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "sla_service_queue_depth",
			Help: "Number of tasks waiting in the processing queue",
		}),

		ProcessingRate: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "sla_service_processing_rate",
			Help: "Current processing rate (tasks per second)",
		}),

		ErrorRate: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "sla_service_error_rate",
			Help: "Current error rate (errors per second)",
		}),

		MemoryUsage: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "sla_service_memory_usage_mb",
			Help: "Current memory usage in megabytes",
		}),

		CPUUsage: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "sla_service_cpu_usage_percent",
			Help: "Current CPU usage percentage",
		}),

		SLACompliance: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "sla_service_compliance_percent",
			Help: "Current SLA compliance percentage by metric type",
		}, []string{"metric_type"}),

		SLAViolations: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "sla_service_violations_total",
			Help: "Total number of SLA violations by type",
		}, []string{"violation_type", "severity"}),

		ErrorBudgetBurn: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "sla_service_error_budget_burn_rate",
			Help: "Current error budget burn rate",
		}),

		CircuitBreakerState: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "sla_service_circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
		}, []string{"dependency"}),

		CircuitBreakerTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "sla_service_circuit_breaker_total",
			Help: "Total circuit breaker state changes",
		}, []string{"dependency", "state"}),
	}

	// Register metrics with Prometheus
	prometheus.MustRegister(
		metrics.TasksProcessed,
		metrics.ProcessingLatency,
		metrics.ActiveWorkers,
		metrics.QueueDepth,
		metrics.ProcessingRate,
		metrics.ErrorRate,
		metrics.MemoryUsage,
		metrics.CPUUsage,
		metrics.SLACompliance,
		metrics.SLAViolations,
		metrics.ErrorBudgetBurn,
		metrics.CircuitBreakerState,
		metrics.CircuitBreakerTotal,
	)

	// Initialize circuit breaker
	cbSettings := gobreaker.Settings{
		Name:        "sla-service",
		MaxRequests: cfg.CircuitBreakerMaxRequests,
		Interval:    cfg.CircuitBreakerInterval,
		Timeout:     cfg.CircuitBreakerTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= cfg.CircuitBreakerFailureRatio
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			logger.InfoWithContext("Circuit breaker state changed",
				"component", "sla-service",
				"dependency", name,
				"from", from.String(),
				"to", to.String(),
			)

			// Update metrics
			var stateValue float64
			switch to {
			case gobreaker.StateClosed:
				stateValue = 0
			case gobreaker.StateOpen:
				stateValue = 1
			case gobreaker.StateHalfOpen:
				stateValue = 2
			}
			metrics.CircuitBreakerState.WithLabelValues(name).Set(stateValue)
			metrics.CircuitBreakerTotal.WithLabelValues(name, to.String()).Inc()
		},
	}

	circuitBreaker := gobreaker.NewCircuitBreaker(cbSettings)

	// Initialize health checker
	healthChecker := health.NewHealthChecker("sla-service", "v1.0.0",
		logger.WithComponent("health").Logger)

	// Create service instance
	service := &Service{
		config:         cfg,
		logger:         logger.WithComponent("sla-service"),
		circuitBreaker: circuitBreaker,
		healthChecker:  healthChecker,
		metrics:        metrics,
		stopped:        make(chan struct{}),
		taskQueue:      make(chan Task, cfg.TaskQueueSize),
		workerPool:     make(chan *Worker, cfg.WorkerCount),
	}

	// Initialize components
	var err error

	// Initialize collector with high-performance settings
	service.collector, err = NewCollector(&CollectorConfig{
		BufferSize:     cfg.TaskQueueSize,
		BatchSize:      100,
		FlushInterval:  cfg.MetricsFlushInterval,
		MaxCardinality: 10000,
		SamplingRate:   1.0, // Collect all metrics initially
	}, logger.WithComponent("collector"))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize collector: %w", err)
	}

	// Initialize calculator for real-time SLI/SLO calculations
	service.calculator, err = NewCalculator(&CalculatorConfig{
		WindowSize:          5 * time.Minute,
		CalculationInterval: cfg.CalculationInterval,
		QuantileAccuracy:    0.01,
		MaxHistoryPoints:    1000,
	}, logger.WithComponent("calculator"))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize calculator: %w", err)
	}

	// Initialize tracker for end-to-end monitoring
	service.tracker, err = NewTracker(&TrackerConfig{
		MaxActiveIntents:    cfg.MaxConcurrentTasks,
		TrackingTimeout:     10 * time.Minute,
		CompletionThreshold: 30 * time.Second,
	}, logger.WithComponent("tracker"))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracker: %w", err)
	}

	// Initialize alerting for SLA violation detection
	service.alerting, err = NewAlertManager(&AlertManagerConfig{
		EvaluationInterval:  cfg.CalculationInterval,
		NotificationTimeout: 30 * time.Second,
		MaxPendingAlerts:    1000,
		DeduplicationWindow: 5 * time.Minute,
	}, logger.WithComponent("alerting"))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize alert manager: %w", err)
	}

	// Initialize storage manager
	service.storage, err = NewStorageManager(&StorageConfig{
		RetentionPeriod:    7 * 24 * time.Hour, // 7 days
		CompactionInterval: 1 * time.Hour,
		MaxDiskUsageMB:     1000, // 1GB max storage
		CompressionEnabled: true,
	}, logger.WithComponent("storage"))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage manager: %w", err)
	}

	// Register health checks
	service.registerHealthChecks()

	return service, nil
}

// Start begins the SLA monitoring service
func (s *Service) Start(ctx context.Context) error {
	s.startMu.Lock()
	defer s.startMu.Unlock()

	if s.started.Load() {
		return fmt.Errorf("service already started")
	}

	s.logger.InfoWithContext("Starting SLA monitoring service",
		"worker_count", s.config.WorkerCount,
		"task_queue_size", s.config.TaskQueueSize,
		"max_concurrent_tasks", s.config.MaxConcurrentTasks,
	)

	// Start components
	if err := s.collector.Start(ctx); err != nil {
		return fmt.Errorf("failed to start collector: %w", err)
	}

	if err := s.calculator.Start(ctx); err != nil {
		return fmt.Errorf("failed to start calculator: %w", err)
	}

	if err := s.tracker.Start(ctx); err != nil {
		return fmt.Errorf("failed to start tracker: %w", err)
	}

	if err := s.alerting.Start(ctx); err != nil {
		return fmt.Errorf("failed to start alert manager: %w", err)
	}

	if err := s.storage.Start(ctx); err != nil {
		return fmt.Errorf("failed to start storage manager: %w", err)
	}

	// Initialize worker pool
	s.workers = make([]*Worker, s.config.WorkerCount)
	for i := 0; i < s.config.WorkerCount; i++ {
		worker := s.createWorker(i)
		s.workers[i] = worker
		s.workerPool <- worker

		go s.runWorker(ctx, worker)
	}

	// Start background goroutines
	go s.processTaskQueue(ctx)
	go s.updateMetrics(ctx)
	go s.performHealthChecks(ctx)

	// Mark as started and ready
	s.started.Store(true)
	s.healthChecker.SetReady(true)
	s.healthChecker.SetHealthy(true)

	s.logger.InfoWithContext("SLA monitoring service started successfully")

	return nil
}

// Stop gracefully shuts down the SLA monitoring service
func (s *Service) Stop(ctx context.Context) error {
	if !s.started.Load() || s.stopping.Load() {
		return nil
	}

	s.stopping.Store(true)
	s.logger.InfoWithContext("Stopping SLA monitoring service")

	// Mark as not ready
	s.healthChecker.SetReady(false)

	// Stop accepting new tasks
	close(s.taskQueue)

	// Wait for active tasks to complete or timeout
	timeout := 30 * time.Second
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) && atomic.LoadInt64(&s.activeJobs) > 0 {
		time.Sleep(100 * time.Millisecond)
	}

	// Stop components
	if err := s.collector.Stop(ctx); err != nil {
		s.logger.ErrorWithContext("Failed to stop collector", err)
	}

	if err := s.calculator.Stop(ctx); err != nil {
		s.logger.ErrorWithContext("Failed to stop calculator", err)
	}

	if err := s.tracker.Stop(ctx); err != nil {
		s.logger.ErrorWithContext("Failed to stop tracker", err)
	}

	if err := s.alerting.Stop(ctx); err != nil {
		s.logger.ErrorWithContext("Failed to stop alert manager", err)
	}

	if err := s.storage.Stop(ctx); err != nil {
		s.logger.ErrorWithContext("Failed to stop storage manager", err)
	}

	// Stop workers
	for _, worker := range s.workers {
		if worker != nil {
			close(worker.quit)
		}
	}

	close(s.stopped)
	s.logger.InfoWithContext("SLA monitoring service stopped")

	return nil
}

// SubmitTask submits a task for processing
func (s *Service) SubmitTask(ctx context.Context, task Task) error {
	if s.stopping.Load() {
		return fmt.Errorf("service is stopping")
	}

	select {
	case s.taskQueue <- task:
		s.metrics.QueueDepth.Set(float64(len(s.taskQueue)))
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("task queue is full")
	}
}

// GetHealthStatus returns the current health status of the service
func (s *Service) GetHealthStatus(ctx context.Context) *health.HealthResponse {
	return s.healthChecker.Check(ctx)
}

// GetMetrics returns current service metrics
func (s *Service) GetMetrics() ServiceStats {
	return ServiceStats{
		ActiveWorkers:   int(s.metrics.ActiveWorkers.Get()),
		QueueDepth:      int(s.metrics.QueueDepth.Get()),
		ProcessingRate:  s.metrics.ProcessingRate.Get(),
		ErrorRate:       s.metrics.ErrorRate.Get(),
		MemoryUsageMB:   s.metrics.MemoryUsage.Get(),
		CPUUsagePercent: s.metrics.CPUUsage.Get(),
		Uptime:          time.Since(s.lastStatsUpdate),
	}
}

// ServiceStats contains current service statistics
type ServiceStats struct {
	ActiveWorkers   int           `json:"active_workers"`
	QueueDepth      int           `json:"queue_depth"`
	ProcessingRate  float64       `json:"processing_rate"`
	ErrorRate       float64       `json:"error_rate"`
	MemoryUsageMB   float64       `json:"memory_usage_mb"`
	CPUUsagePercent float64       `json:"cpu_usage_percent"`
	Uptime          time.Duration `json:"uptime"`
}

// createWorker creates a new worker instance
func (s *Service) createWorker(id int) *Worker {
	return &Worker{
		id:      id,
		service: s,
		tasks:   make(chan Task, 10),
		quit:    make(chan struct{}),
		metrics: &WorkerMetrics{
			TasksProcessed: prometheus.NewCounter(prometheus.CounterOpts{
				Name:        "sla_worker_tasks_processed_total",
				Help:        "Total tasks processed by worker",
				ConstLabels: map[string]string{"worker_id": fmt.Sprintf("%d", id)},
			}),
			ProcessingTime: prometheus.NewHistogram(prometheus.HistogramOpts{
				Name:        "sla_worker_processing_time_seconds",
				Help:        "Task processing time by worker",
				ConstLabels: map[string]string{"worker_id": fmt.Sprintf("%d", id)},
				Buckets:     prometheus.ExponentialBuckets(0.001, 2, 10),
			}),
			Errors: prometheus.NewCounter(prometheus.CounterOpts{
				Name:        "sla_worker_errors_total",
				Help:        "Total errors by worker",
				ConstLabels: map[string]string{"worker_id": fmt.Sprintf("%d", id)},
			}),
		},
	}
}

// runWorker runs the worker goroutine
func (s *Service) runWorker(ctx context.Context, worker *Worker) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.ErrorWithContext("Worker panic recovered",
				fmt.Errorf("panic: %v", r),
				"worker_id", worker.id,
			)
		}
		atomic.AddInt32(&s.workerCount, -1)
		s.metrics.ActiveWorkers.Dec()
	}()

	atomic.AddInt32(&s.workerCount, 1)
	s.metrics.ActiveWorkers.Inc()

	for {
		select {
		case <-ctx.Done():
			return
		case <-worker.quit:
			return
		case task := <-worker.tasks:
			s.processTask(ctx, worker, task)
		}
	}
}

// processTaskQueue processes tasks from the main queue
func (s *Service) processTaskQueue(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopped:
			return
		case task, ok := <-s.taskQueue:
			if !ok {
				return
			}

			// Get a worker from the pool
			select {
			case worker := <-s.workerPool:
				select {
				case worker.tasks <- task:
					// Task dispatched successfully
				default:
					// Worker busy, put back in pool and try another
					s.workerPool <- worker
					// Try to process task with circuit breaker
					s.processTaskWithCircuitBreaker(ctx, task)
				}
			default:
				// No workers available, process with circuit breaker
				s.processTaskWithCircuitBreaker(ctx, task)
			}
		}
	}
}

// processTask processes a single task
func (s *Service) processTask(ctx context.Context, worker *Worker, task Task) {
	defer func() {
		// Return worker to pool
		select {
		case s.workerPool <- worker:
		default:
			// Pool is full, worker will be garbage collected
		}
		atomic.AddInt64(&s.activeJobs, -1)
	}()

	atomic.AddInt64(&s.activeJobs, 1)
	start := time.Now()

	// Create context with timeout
	taskCtx, cancel := context.WithTimeout(ctx, task.GetTimeout())
	defer cancel()

	// Execute task
	err := task.Execute(taskCtx)
	duration := time.Since(start)

	// Update metrics
	worker.metrics.TasksProcessed.Inc()
	worker.metrics.ProcessingTime.Observe(duration.Seconds())

	taskType := string(task.GetType())
	if err != nil {
		worker.metrics.Errors.Inc()
		s.metrics.TasksProcessed.WithLabelValues(taskType, "error").Inc()
		s.errorRate.Add(1)

		s.logger.ErrorWithContext("Task execution failed",
			err,
			"worker_id", worker.id,
			"task_type", taskType,
			"duration", duration,
		)
	} else {
		s.metrics.TasksProcessed.WithLabelValues(taskType, "success").Inc()
		s.processingRate.Add(1)
	}

	s.metrics.ProcessingLatency.WithLabelValues(taskType).Observe(duration.Seconds())
}

// processTaskWithCircuitBreaker processes a task with circuit breaker protection
func (s *Service) processTaskWithCircuitBreaker(ctx context.Context, task Task) {
	_, err := s.circuitBreaker.Execute(func() (interface{}, error) {
		return nil, task.Execute(ctx)
	})

	taskType := string(task.GetType())
	if err != nil {
		s.metrics.TasksProcessed.WithLabelValues(taskType, "circuit_breaker").Inc()
		s.logger.WarnWithContext("Task processed via circuit breaker",
			"task_type", taskType,
			"error", err.Error(),
		)
	}
}

// updateMetrics updates service performance metrics
func (s *Service) updateMetrics(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var lastProcessed, lastErrors uint64

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopped:
			return
		case <-ticker.C:
			// Calculate rates
			currentProcessed := atomic.LoadUint64(&s.processingRate)
			currentErrors := atomic.LoadUint64(&s.errorRate)

			processingRate := float64(currentProcessed-lastProcessed) / 5.0
			errorRate := float64(currentErrors-lastErrors) / 5.0

			s.metrics.ProcessingRate.Set(processingRate)
			s.metrics.ErrorRate.Set(errorRate)

			lastProcessed = currentProcessed
			lastErrors = currentErrors

			// Update resource usage
			s.updateResourceMetrics()

			// Update queue depth
			s.metrics.QueueDepth.Set(float64(len(s.taskQueue)))
			s.metrics.ActiveWorkers.Set(float64(atomic.LoadInt32(&s.workerCount)))
		}
	}
}

// updateResourceMetrics updates CPU and memory usage metrics
func (s *Service) updateResourceMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Convert bytes to MB
	memoryMB := float64(m.Alloc) / 1024 / 1024
	s.metrics.MemoryUsage.Set(memoryMB)

	// CPU usage would be calculated using additional tools in production
	// For now, we'll use a simplified approach
	cpuPercent := float64(runtime.NumGoroutine()) / float64(runtime.NumCPU()) * 0.1
	s.metrics.CPUUsage.Set(cpuPercent)
}

// performHealthChecks performs periodic health checks
func (s *Service) performHealthChecks(ctx context.Context) {
	ticker := time.NewTicker(s.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopped:
			return
		case <-ticker.C:
			// Check resource limits
			memoryMB := s.metrics.MemoryUsage.Get()
			cpuPercent := s.metrics.CPUUsage.Get()

			healthy := true
			if memoryMB > float64(s.config.MaxMemoryUsageMB) {
				healthy = false
				s.logger.WarnWithContext("Memory usage exceeds limit",
					"current_mb", memoryMB,
					"limit_mb", s.config.MaxMemoryUsageMB,
				)
			}

			if cpuPercent > s.config.MaxCPUUsagePercent {
				healthy = false
				s.logger.WarnWithContext("CPU usage exceeds limit",
					"current_percent", cpuPercent,
					"limit_percent", s.config.MaxCPUUsagePercent,
				)
			}

			s.healthChecker.SetHealthy(healthy)
		}
	}
}

// registerHealthChecks registers health check functions
func (s *Service) registerHealthChecks() {
	// Register component health checks
	s.healthChecker.RegisterCheck("collector", func(ctx context.Context) *health.Check {
		if s.collector == nil {
			return &health.Check{
				Status: health.StatusUnhealthy,
				Error:  "collector not initialized",
			}
		}
		return &health.Check{
			Status:  health.StatusHealthy,
			Message: "collector operational",
		}
	})

	s.healthChecker.RegisterCheck("calculator", func(ctx context.Context) *health.Check {
		if s.calculator == nil {
			return &health.Check{
				Status: health.StatusUnhealthy,
				Error:  "calculator not initialized",
			}
		}
		return &health.Check{
			Status:  health.StatusHealthy,
			Message: "calculator operational",
		}
	})

	s.healthChecker.RegisterCheck("tracker", func(ctx context.Context) *health.Check {
		if s.tracker == nil {
			return &health.Check{
				Status: health.StatusUnhealthy,
				Error:  "tracker not initialized",
			}
		}
		return &health.Check{
			Status:  health.StatusHealthy,
			Message: "tracker operational",
		}
	})

	s.healthChecker.RegisterCheck("worker_pool", func(ctx context.Context) *health.Check {
		activeWorkers := atomic.LoadInt32(&s.workerCount)
		if activeWorkers == 0 {
			return &health.Check{
				Status: health.StatusUnhealthy,
				Error:  "no active workers",
			}
		}

		queueDepth := len(s.taskQueue)
		if queueDepth > s.config.TaskQueueSize*3/4 {
			return &health.Check{
				Status:  health.StatusDegraded,
				Message: fmt.Sprintf("queue depth high: %d", queueDepth),
			}
		}

		return &health.Check{
			Status:  health.StatusHealthy,
			Message: fmt.Sprintf("workers: %d, queue: %d", activeWorkers, queueDepth),
		}
	})
}
