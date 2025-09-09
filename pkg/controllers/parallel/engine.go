/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package parallel

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	v1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/contracts"
	resiliencecontroller "github.com/thc1006/nephoran-intent-operator/pkg/controllers/resilience"
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"
)

// ParallelProcessingConfig defines configuration for parallel processing
type ParallelProcessingConfig struct {
	// MaxConcurrentIntents limits concurrent intent processing
	MaxConcurrentIntents int

	// Worker pool sizes for different stages
	IntentPoolSize     int
	LLMPoolSize        int
	RAGPoolSize        int
	ResourcePoolSize   int
	ManifestPoolSize   int
	GitOpsPoolSize     int
	DeploymentPoolSize int

	// Task queue size
	TaskQueueSize int

	// Health check interval
	HealthCheckInterval time.Duration

	// Processing timeouts
	IntentTimeout     time.Duration
	LLMTimeout        time.Duration
	ManifestTimeout   time.Duration
	DeploymentTimeout time.Duration

	// Retry configurations
	MaxRetries      int
	RetryBackoff    time.Duration
	RetryMultiplier float64
}

// TaskType represents different types of processing tasks
type TaskType string

const (
	TaskTypeIntentParsing      TaskType = "intent_parsing"
	TaskTypeIntentProcessing   TaskType = "intent_processing"
	TaskTypeLLMProcessing      TaskType = "llm_processing"
	TaskTypeRAGQuery           TaskType = "rag_query"
	TaskTypeRAGRetrieval       TaskType = "rag_retrieval"
	TaskTypeResourcePlanning   TaskType = "resource_planning"
	TaskTypeManifestGeneration TaskType = "manifest_generation"
	TaskTypeGitOpsCommit       TaskType = "gitops_commit"
	TaskTypeDeployment         TaskType = "deployment"
	TaskTypeDeploymentVerify   TaskType = "deployment_verify"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
	TaskStatusRetrying  TaskStatus = "retrying"
)

// Task represents a unit of work in the parallel processing system
// Using Go 1.24+ struct evolution patterns for backward compatibility
type Task struct {
	// ID is a unique identifier for the task
	ID string `json:"id"`

	// IntentID links the task to a NetworkIntent
	IntentID string `json:"intent_id"`

	// CorrelationID for tracing and error correlation (Go 1.24+ migration)
	// Added for backward compatibility with existing tests
	CorrelationID string `json:"correlation_id,omitempty"`

	// Type specifies the type of task
	Type TaskType `json:"type"`

	// Priority determines processing priority
	Priority int `json:"priority"`

	// Status tracks the current task status
	Status TaskStatus `json:"status"`

	// InputData contains input parameters for the task
	InputData json.RawMessage `json:"input_data,omitempty"`

	// OutputData contains results from task execution
	OutputData json.RawMessage `json:"output_data,omitempty"`

	// Error contains error information if task failed
	Error error `json:"-"`

	// Context for the task execution
	Context context.Context `json:"-"`

	// Cancel function for task cancellation
	Cancel context.CancelFunc `json:"-"`

	// Timeout for task execution
	Timeout time.Duration `json:"timeout"`

	// RetryConfig for advanced retry configuration (Go 1.24+ evolution)
	RetryConfig *TaskRetryConfig `json:"retry_config,omitempty"`

	// RetryCount tracks number of retry attempts (maintained for compatibility)
	RetryCount int `json:"retry_count"`

	// CreatedAt timestamp
	CreatedAt time.Time `json:"created_at"`

	// StartedAt timestamp
	StartedAt *time.Time `json:"started_at,omitempty"`

	// CompletedAt timestamp
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// ScheduledAt timestamp when task was scheduled
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`

	// Dependencies are tasks that must complete before this task
	Dependencies []string `json:"dependencies,omitempty"`

	// Metadata for additional task information
	Metadata map[string]string `json:"metadata,omitempty"`

	// Intent reference for context
	Intent *v1.NetworkIntent `json:"-"`

	// Callback functions
	OnSuccess func(*TaskResult) `json:"-"`
	OnFailure func(*TaskResult) `json:"-"`

	// Version field for struct evolution tracking
	Version int `json:"version,omitempty"`
}

// GetID implements TaskInterface
func (t *Task) GetID() string {
	return t.ID
}

// GetPriority implements TaskInterface
func (t *Task) GetPriority() int {
	return t.Priority
}

// GetTimeout implements TaskInterface
func (t *Task) GetTimeout() time.Duration {
	return t.Timeout
}

// Execute implements TaskInterface - this is a placeholder
// Actual execution logic should be implemented by task processors
func (t *Task) Execute(ctx context.Context) error {
	// This is a placeholder implementation
	// Real execution would be handled by the processing engine
	return fmt.Errorf("task execution not implemented for type %s", t.Type)
}

// WorkflowResult contains the result of processing a NetworkIntent workflow
type WorkflowResult struct {
	// IntentID is the ID of the processed intent
	IntentID string

	// Success indicates if the workflow completed successfully
	Success bool

	// Error contains error information if workflow failed
	Error error

	// Tasks contains information about all tasks in the workflow
	Tasks []*Task

	// StartTime when workflow processing began
	StartTime time.Time

	// EndTime when workflow processing completed
	EndTime time.Time

	// Duration of workflow processing
	Duration time.Duration

	// Results contains structured output from the workflow
	Results map[string]interface{}
}

// TaskResult contains the result of task processing
type TaskResult struct {
	// Task identification
	TaskID   string
	WorkerID int
	PoolName string

	// Result status
	Success      bool
	Error        error
	ErrorMessage string

	// Timing information
	Duration       time.Duration
	QueueTime      time.Duration
	ProcessingTime time.Duration
	CompletedAt    time.Time

	// Resource usage
	MemoryUsed int64
	CPUUsed    float64

	// Output data
	OutputData       map[string]interface{}
	ProcessingResult *contracts.ProcessingResult
}

// ProcessingMetrics contains metrics about parallel processing performance
type ProcessingMetrics struct {
	// Total number of tasks processed
	TotalTasks int64

	// Number of successful tasks
	SuccessfulTasks int64

	// Number of failed tasks
	FailedTasks int64

	// Current number of running tasks
	RunningTasks int64

	// Current number of pending tasks
	PendingTasks int64

	// Average task processing latency
	AverageLatency time.Duration

	// Success rate (0.0 to 1.0)
	SuccessRate float64

	// Tasks per second throughput
	Throughput float64

	// Worker pool utilization
	WorkerUtilization map[TaskType]float64

	// Queue depths for different task types
	QueueDepths map[TaskType]int

	// Last updated timestamp
	LastUpdated time.Time
}

// TaskScheduler manages task scheduling and dependency resolution
type TaskScheduler struct {
	engine          *ParallelProcessingEngine
	schedulingQueue chan *Task
	dependencyGraph *DependencyGraph
	readyTasks      chan *Task
	strategies      map[string]SchedulingStrategy
	currentStrategy string
	logger          logr.Logger
	stopChan        chan struct{}
	mutex           sync.RWMutex
}

// SchedulingStrategy defines the interface for task scheduling strategies
type SchedulingStrategy interface {
	ScheduleTask(task *Task, availableWorkers map[string]int) (string, error)
	GetStrategyName() string
	GetMetrics() map[string]float64
}

// LoadBalancingStrategy defines the interface for load balancing strategies
type LoadBalancingStrategy interface {
	SelectPool(pools map[string]*WorkerPool, task *Task) (string, error)
	GetStrategyName() string
	UpdateMetrics(poolName string, metrics *PoolMetrics)
}

// DependencyGraph manages task dependencies and execution order
type DependencyGraph struct {
	nodes map[string]*DependencyNode
	mutex sync.RWMutex
}

// DependencyNode represents a single node in the dependency graph
type DependencyNode struct {
	TaskID       string
	Dependencies []*DependencyNode
	Dependents   []*DependencyNode
	Completed    bool
	Failed       bool
	mutex        sync.RWMutex
}

// LoadBalancer manages load balancing across worker pools
type LoadBalancer struct {
	strategies  map[string]LoadBalancingStrategy
	current     string
	poolMetrics map[string]*PoolMetrics
	logger      logr.Logger
	mutex       sync.RWMutex
}

// WorkerPool represents a pool of workers for processing tasks
type WorkerPool struct {
	name           string
	workerCount    int
	workers        []*Worker
	taskQueue      chan *Task
	resultQueue    chan *TaskResult
	activeWorkers  int32
	processedTasks int64
	failedTasks    int64
	averageLatency time.Duration
	memoryUsage    int64
	cpuUsage       float64
	logger         logr.Logger
	stopChan       chan struct{}
	mutex          sync.RWMutex
}

// PoolMetrics contains metrics for a worker pool
type PoolMetrics struct {
	ActiveWorkers  int32
	ProcessedTasks int64
	FailedTasks    int64
	AverageLatency time.Duration
	MemoryUsage    int64
	CPUUsage       float64
	QueueLength    int
	QueueCapacity  int
	Throughput     float64
	SuccessRate    float64
	LastUpdated    time.Time
}

// TaskProcessor defines the interface for processing tasks
type TaskProcessor interface {
	ProcessTask(ctx context.Context, task *Task) (*TaskResult, error)
	GetProcessorType() TaskType
	HealthCheck(ctx context.Context) error
	GetMetrics() map[string]interface{}
}

// PriorityQueue implements a priority-based task queue
type PriorityQueue struct {
	tasks       []*Task
	maxSize     int
	currentSize int
	mutex       sync.RWMutex
	notEmpty    *sync.Cond
}

// ParallelProcessingEngine orchestrates parallel processing of NetworkIntents
type ParallelProcessingEngine struct {
	config            *ParallelProcessingConfig
	resilienceManager *resiliencecontroller.ResilienceManager
	errorTracker      *monitoring.ErrorTracker
	logger            logr.Logger

	// Processing state
	mu      sync.RWMutex
	running bool
	ctx     context.Context
	cancel  context.CancelFunc

	// Task management
	tasks      map[string]*Task
	taskQueue  chan *Task
	resultChan chan *Task

	// Worker pools for different processing stages
	intentWorkers     []*Worker
	llmWorkers        []*Worker
	ragWorkers        []*Worker
	resourceWorkers   []*Worker
	manifestWorkers   []*Worker
	gitopsWorkers     []*Worker
	deploymentWorkers []*Worker

	// New structured worker pools
	intentPool     *WorkerPool
	llmPool        *WorkerPool
	ragPool        *WorkerPool
	resourcePool   *WorkerPool
	manifestPool   *WorkerPool
	gitopsPool     *WorkerPool
	deploymentPool *WorkerPool

	// Task scheduling and dependency management
	taskScheduler   *TaskScheduler
	dependencyGraph *DependencyGraph
	loadBalancer    *LoadBalancer

	// Metrics
	metrics     *ProcessingMetrics
	metricsLock sync.RWMutex

	// Health monitoring
	healthTicker *time.Ticker
}

// Worker represents a worker goroutine for processing tasks
type Worker struct {
	// Basic identification
	ID      int
	Type    TaskType
	Queue   chan *Task
	Results chan *Task
	Context context.Context
	Cancel  context.CancelFunc
	Engine  *ParallelProcessingEngine

	// Enhanced worker fields for structured pools
	id           int
	poolName     string
	taskQueue    chan *Task
	resultQueue  chan *TaskResult
	processor    TaskProcessor
	lastActivity time.Time
	logger       logr.Logger
	stopChan     chan struct{}

	// Worker state and metrics
	currentTask    *Task
	totalProcessed int64
	totalErrors    int64
	averageLatency time.Duration
	memoryUsage    int64
	cpuUsage       float64
	mutex          sync.RWMutex
}

// NewParallelProcessingEngine creates a new parallel processing engine
func NewParallelProcessingEngine(
	config *ParallelProcessingConfig,
	resilienceManager *resiliencecontroller.ResilienceManager,
	errorTracker *monitoring.ErrorTracker,
	logger logr.Logger,
) (*ParallelProcessingEngine, error) {
	if config == nil {
		config = &ParallelProcessingConfig{
			MaxConcurrentIntents: 50,
			IntentPoolSize:       10,
			LLMPoolSize:          5,
			RAGPoolSize:          5,
			ResourcePoolSize:     8,
			ManifestPoolSize:     8,
			GitOpsPoolSize:       4,
			DeploymentPoolSize:   6,
			TaskQueueSize:        1000,
			HealthCheckInterval:  30 * time.Second,
			IntentTimeout:        5 * time.Minute,
			LLMTimeout:           2 * time.Minute,
			ManifestTimeout:      1 * time.Minute,
			DeploymentTimeout:    10 * time.Minute,
			MaxRetries:           3,
			RetryBackoff:         5 * time.Second,
			RetryMultiplier:      2.0,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	engine := &ParallelProcessingEngine{
		config:            config,
		resilienceManager: resilienceManager,
		errorTracker:      errorTracker,
		logger:            logger,
		ctx:               ctx,
		cancel:            cancel,
		tasks:             make(map[string]*Task),
		taskQueue:         make(chan *Task, config.TaskQueueSize),
		resultChan:        make(chan *Task, config.TaskQueueSize),
		metrics: &ProcessingMetrics{
			WorkerUtilization: make(map[TaskType]float64),
			QueueDepths:       make(map[TaskType]int),
			LastUpdated:       time.Now(),
		},
	}

	return engine, nil
}

// Start initializes and starts the parallel processing engine
func (e *ParallelProcessingEngine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return fmt.Errorf("parallel processing engine is already running")
	}

	// Initialize worker pools
	e.initializeWorkerPools()

	// Start result processor
	go e.processResults()

	// Start metrics updater
	go e.updateMetricsLoop()

	// Start health checker
	if e.config.HealthCheckInterval > 0 {
		e.healthTicker = time.NewTicker(e.config.HealthCheckInterval)
		go e.healthCheckLoop()
	}

	e.running = true
	e.logger.Info("Parallel processing engine started", "config", e.config)

	return nil
}

// Stop gracefully shuts down the parallel processing engine
func (e *ParallelProcessingEngine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return
	}

	e.running = false
	
	// Stop health ticker first
	if e.healthTicker != nil {
		e.healthTicker.Stop()
	}

	// Cancel context to signal all goroutines to stop
	e.cancel()
	
	// Give goroutines a moment to clean up
	time.Sleep(100 * time.Millisecond)

	// Cancel all workers
	e.cancelAllWorkers()

	// ITERATION #9 fix: Close channels safely
	// Close taskQueue after workers have stopped to prevent "send on closed channel"
	select {
	case <-e.taskQueue:
		// Drain any remaining tasks
	default:
	}
	close(e.taskQueue)

	e.logger.Info("Parallel processing engine stopped")
}

// ProcessIntentWorkflow processes a NetworkIntent through the complete workflow
func (e *ParallelProcessingEngine) ProcessIntentWorkflow(ctx context.Context, intent *v1.NetworkIntent) (*WorkflowResult, error) {
	if !e.isRunning() {
		return nil, fmt.Errorf("parallel processing engine is not running")
	}

	workflowID := fmt.Sprintf("workflow-%s-%d", intent.Name, time.Now().UnixNano())

	result := &WorkflowResult{
		IntentID:  intent.Name,
		StartTime: time.Now(),
		Results:   make(map[string]interface{}),
	}

	e.logger.Info("Starting intent workflow", "intent", intent.Name, "workflowID", workflowID)

	// Create workflow tasks
	tasks, err := e.createWorkflowTasks(ctx, intent, workflowID)
	if err != nil {
		result.Error = fmt.Errorf("failed to create workflow tasks: %w", err)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result, err
	}

	result.Tasks = tasks

	// Submit tasks for processing
	for _, task := range tasks {
		if err := e.SubmitTask(task); err != nil {
			result.Error = fmt.Errorf("failed to submit task %s: %w", task.ID, err)
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result, err
		}
	}

	// Wait for all tasks to complete
	completionCtx, completionCancel := context.WithTimeout(ctx, e.config.IntentTimeout)
	defer completionCancel()

	completed, err := e.waitForWorkflowCompletion(completionCtx, workflowID, tasks)
	result.Success = completed && err == nil
	result.Error = err
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Collect results from completed tasks
	for _, task := range tasks {
		if task.OutputData != nil {
			var outputMap map[string]interface{}
			if err := json.Unmarshal(task.OutputData, &outputMap); err == nil {
				for k, v := range outputMap {
					result.Results[fmt.Sprintf("%s.%s", task.Type, k)] = v
				}
			}
		}
	}

	e.logger.Info("Intent workflow completed",
		"intent", intent.Name,
		"workflowID", workflowID,
		"success", result.Success,
		"duration", result.Duration,
		"tasksCount", len(tasks))

	return result, nil
}

// SubmitTask submits a task for processing
func (e *ParallelProcessingEngine) SubmitTask(task *Task) error {
	if !e.isRunning() {
		return fmt.Errorf("parallel processing engine is not running")
	}

	// Store task
	e.mu.Lock()
	e.tasks[task.ID] = task
	e.mu.Unlock()

	// Submit to queue
	select {
	case e.taskQueue <- task:
		task.Status = TaskStatusPending
		e.logger.V(1).Info("Task submitted", "taskID", task.ID, "type", task.Type)
		return nil
	default:
		return fmt.Errorf("task queue is full")
	}
}

// GetTask retrieves a task by ID
func (e *ParallelProcessingEngine) GetTask(taskID string) (*Task, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	task, exists := e.tasks[taskID]
	return task, exists
}

// GetMetrics returns current processing metrics
func (e *ParallelProcessingEngine) GetMetrics() *ProcessingMetrics {
	e.metricsLock.RLock()
	defer e.metricsLock.RUnlock()

	// Create a safe copy without copying mutex to avoid lock copying violation
	metrics := &ProcessingMetrics{
		TotalTasks:        e.metrics.TotalTasks,
		SuccessfulTasks:   e.metrics.SuccessfulTasks,
		FailedTasks:       e.metrics.FailedTasks,
		RunningTasks:      e.metrics.RunningTasks,
		PendingTasks:      e.metrics.PendingTasks,
		AverageLatency:    e.metrics.AverageLatency,
		SuccessRate:       e.metrics.SuccessRate,
		WorkerUtilization: make(map[TaskType]float64),
		QueueDepths:       make(map[TaskType]int),
	}

	// Copy maps safely
	for k, v := range e.metrics.WorkerUtilization {
		metrics.WorkerUtilization[k] = v
	}

	for k, v := range e.metrics.QueueDepths {
		metrics.QueueDepths[k] = v
	}

	return metrics
}

// HealthCheck returns the health status of the processing engine
func (e *ParallelProcessingEngine) HealthCheck() error {
	if !e.isRunning() {
		return fmt.Errorf("parallel processing engine is not running")
	}

	// Check if task queue is severely backed up
	if len(e.taskQueue) > int(float64(e.config.TaskQueueSize)*0.9) {
		return fmt.Errorf("task queue is critically full: %d/%d", len(e.taskQueue), e.config.TaskQueueSize)
	}

	// Check if too many tasks are stuck in running state
	runningTasks := e.countTasksByStatus(TaskStatusRunning)
	if runningTasks > int64(e.config.MaxConcurrentIntents*2) {
		return fmt.Errorf("too many running tasks: %d", runningTasks)
	}

	return nil
}

// Helper methods

func (e *ParallelProcessingEngine) isRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.running
}

func (e *ParallelProcessingEngine) initializeWorkerPools() {
	// Initialize different worker pools
	e.intentWorkers = e.createWorkers(TaskTypeIntentParsing, e.config.IntentPoolSize)
	e.llmWorkers = e.createWorkers(TaskTypeLLMProcessing, e.config.LLMPoolSize)
	e.ragWorkers = e.createWorkers(TaskTypeRAGQuery, e.config.RAGPoolSize)
	e.resourceWorkers = e.createWorkers(TaskTypeResourcePlanning, e.config.ResourcePoolSize)
	e.manifestWorkers = e.createWorkers(TaskTypeManifestGeneration, e.config.ManifestPoolSize)
	e.gitopsWorkers = e.createWorkers(TaskTypeGitOpsCommit, e.config.GitOpsPoolSize)
	e.deploymentWorkers = e.createWorkers(TaskTypeDeployment, e.config.DeploymentPoolSize)
}

func (e *ParallelProcessingEngine) createWorkers(taskType TaskType, poolSize int) []*Worker {
	workers := make([]*Worker, poolSize)

	for i := 0; i < poolSize; i++ {
		ctx, cancel := context.WithCancel(e.ctx)
		worker := &Worker{
			ID:      i,
			Type:    taskType,
			Queue:   make(chan *Task, 10),
			Results: e.resultChan,
			Context: ctx,
			Cancel:  cancel,
			Engine:  e,
		}

		workers[i] = worker
		go worker.Run()
	}

	// Start dispatcher for this worker pool
	go e.dispatchTasks(taskType, workers)

	return workers
}

func (e *ParallelProcessingEngine) dispatchTasks(taskType TaskType, workers []*Worker) {
	workerIndex := 0

	for {
		select {
		case task := <-e.taskQueue:
			if task.Type == taskType {
				// Round-robin assignment to workers
				worker := workers[workerIndex]
				workerIndex = (workerIndex + 1) % len(workers)

				select {
				case worker.Queue <- task:
					e.logger.V(1).Info("Task dispatched to worker",
						"taskID", task.ID,
						"workerType", taskType,
						"workerID", worker.ID)
				case <-e.ctx.Done():
					return
				}
			} else {
				// Put back in main queue for other dispatchers
				// ITERATION #9 fix: Add defensive check for closed taskQueue
				if e.isRunning() {
					select {
					case e.taskQueue <- task:
					case <-e.ctx.Done():
						return
					default:
						// Channel is full or closed, skip this task
						e.logger.V(2).Info("Task queue full/closed, dropping task", "taskID", task.ID)
					}
				} else {
					// Engine is shutting down, don't try to requeue
					return
				}
			}
		case <-e.ctx.Done():
			return
		}
	}
}

func (e *ParallelProcessingEngine) processResults() {
	for {
		select {
		case task := <-e.resultChan:
			e.handleTaskResult(task)
		case <-e.ctx.Done():
			return
		}
	}
}

func (e *ParallelProcessingEngine) handleTaskResult(task *Task) {
	e.mu.Lock()
	e.tasks[task.ID] = task
	e.mu.Unlock()

	// Update metrics
	e.updateTaskMetrics(task)

	// Log task completion
	if task.Error != nil {
		e.logger.Error(task.Error, "Task completed with error",
			"taskID", task.ID,
			"type", task.Type,
			"retryCount", task.RetryCount)

		// Track error
		if e.errorTracker != nil {
			e.errorTracker.TrackError(
				task.Context,
				task.Error,
				string(task.Type),
				"task_execution",
				map[string]interface{}{})
		}
	} else {
		e.logger.V(1).Info("Task completed successfully",
			"taskID", task.ID,
			"type", task.Type,
			"duration", task.CompletedAt.Sub(*task.StartedAt))
	}
}

func (e *ParallelProcessingEngine) updateTaskMetrics(task *Task) {
	e.metricsLock.Lock()
	defer e.metricsLock.Unlock()

	e.metrics.TotalTasks++

	if task.Error != nil {
		e.metrics.FailedTasks++
	} else {
		e.metrics.SuccessfulTasks++
	}

	// Update success rate
	if e.metrics.TotalTasks > 0 {
		e.metrics.SuccessRate = float64(e.metrics.SuccessfulTasks) / float64(e.metrics.TotalTasks)
	}

	// Update average latency
	if task.StartedAt != nil && task.CompletedAt != nil {
		latency := task.CompletedAt.Sub(*task.StartedAt)
		totalTasks := float64(e.metrics.TotalTasks)
		currentAvg := float64(e.metrics.AverageLatency)
		e.metrics.AverageLatency = time.Duration((currentAvg*(totalTasks-1) + float64(latency)) / totalTasks)
	}

	e.metrics.LastUpdated = time.Now()
}

func (e *ParallelProcessingEngine) updateMetricsLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.updateRuntimeMetrics()
		case <-e.ctx.Done():
			return
		}
	}
}

func (e *ParallelProcessingEngine) updateRuntimeMetrics() {
	e.metricsLock.Lock()
	defer e.metricsLock.Unlock()

	// Count tasks by status
	e.metrics.RunningTasks = e.countTasksByStatus(TaskStatusRunning)
	e.metrics.PendingTasks = e.countTasksByStatus(TaskStatusPending)

	// Update queue depths
	e.metrics.QueueDepths[TaskTypeIntentParsing] = len(e.taskQueue)

	// Update worker utilization (simplified)
	for taskType := range e.metrics.WorkerUtilization {
		e.metrics.WorkerUtilization[taskType] = 0.5 // Placeholder calculation
	}

	// Calculate throughput (tasks per second over last interval)
	e.metrics.Throughput = float64(e.metrics.TotalTasks) / time.Since(e.metrics.LastUpdated).Seconds()
}

func (e *ParallelProcessingEngine) countTasksByStatus(status TaskStatus) int64 {
	count := int64(0)
	for _, task := range e.tasks {
		if task.Status == status {
			count++
		}
	}
	return count
}

func (e *ParallelProcessingEngine) healthCheckLoop() {
	for {
		select {
		case <-e.healthTicker.C:
			if err := e.HealthCheck(); err != nil {
				e.logger.Error(err, "Health check failed")
			}
		case <-e.ctx.Done():
			return
		}
	}
}

func (e *ParallelProcessingEngine) cancelAllWorkers() {
	workerPools := [][]*Worker{
		e.intentWorkers,
		e.llmWorkers,
		e.ragWorkers,
		e.resourceWorkers,
		e.manifestWorkers,
		e.gitopsWorkers,
		e.deploymentWorkers,
	}

	for _, pool := range workerPools {
		for _, worker := range pool {
			worker.Cancel()
		}
	}
}

func (e *ParallelProcessingEngine) createWorkflowTasks(ctx context.Context, intent *v1.NetworkIntent, workflowID string) ([]*Task, error) {
	tasks := make([]*Task, 0)

	// Create a simple workflow for demonstration
	// In a real implementation, this would be more sophisticated based on the intent type

	// 1. Intent parsing task
	intentTask := &Task{
		ID:       fmt.Sprintf("%s-intent-parsing", workflowID),
		IntentID: intent.Name,
		Type:     TaskTypeIntentParsing,
		Priority: priorityToInt(intent.Spec.Priority),
		Status:   TaskStatusPending,
		InputData: json.RawMessage(`{}`),
		Timeout:   e.config.IntentTimeout,
		CreatedAt: time.Now(),
		Metadata: map[string]string{
			"workflow": workflowID,
			"stage":    "parsing",
		},
	}
	intentTask.Context, intentTask.Cancel = context.WithCancel(ctx)
	tasks = append(tasks, intentTask)

	// 2. LLM processing task (depends on intent parsing)
	llmTask := &Task{
		ID:           fmt.Sprintf("%s-llm-processing", workflowID),
		IntentID:     intent.Name,
		Type:         TaskTypeLLMProcessing,
		Priority:     priorityToInt(intent.Spec.Priority),
		Status:       TaskStatusPending,
		Dependencies: []string{intentTask.ID},
		InputData: json.RawMessage(`{}`),
		Timeout:   e.config.LLMTimeout,
		CreatedAt: time.Now(),
		Metadata: map[string]string{
			"workflow": workflowID,
			"stage":    "llm",
		},
	}
	llmTask.Context, llmTask.Cancel = context.WithCancel(ctx)
	tasks = append(tasks, llmTask)

	return tasks, nil
}

func (e *ParallelProcessingEngine) waitForWorkflowCompletion(ctx context.Context, workflowID string, tasks []*Task) (bool, error) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			allCompleted := true
			anyFailed := false

			for _, task := range tasks {
				if task.Status == TaskStatusPending || task.Status == TaskStatusRunning {
					allCompleted = false
				}
				if task.Status == TaskStatusFailed {
					anyFailed = true
				}
			}

			if allCompleted {
				return !anyFailed, nil
			}

		case <-ctx.Done():
			// Cancel all remaining tasks
			for _, task := range tasks {
				if task.Status == TaskStatusPending || task.Status == TaskStatusRunning {
					task.Cancel()
					task.Status = TaskStatusCancelled
				}
			}
			return false, ctx.Err()
		}
	}
}

// Run executes the worker's main processing loop
func (w *Worker) Run() {
	w.Engine.logger.Info("Worker started", "workerID", w.ID, "type", w.Type)

	for {
		select {
		case task := <-w.Queue:
			w.processTask(task)
		case <-w.Context.Done():
			w.Engine.logger.Info("Worker stopped", "workerID", w.ID, "type", w.Type)
			return
		}
	}
}

func (w *Worker) processTask(task *Task) {
	startTime := time.Now()
	task.StartedAt = &startTime
	task.Status = TaskStatusRunning

	w.Engine.logger.V(1).Info("Worker processing task",
		"workerID", w.ID,
		"taskID", task.ID,
		"type", task.Type)

	// Create timeout context with defensive nil check (ITERATION #9 fix)
	// Prevent "cannot create context from nil parent" panics
	taskCtx := task.Context
	if taskCtx == nil {
		w.Engine.logger.V(1).Info("Task context is nil, using background context",
			"taskID", task.ID, 
			"type", task.Type)
		taskCtx = context.Background()
	}
	
	ctx, cancel := context.WithTimeout(taskCtx, task.Timeout)
	defer cancel()

	// Execute task based on type
	var err error
	switch task.Type {
	case TaskTypeIntentParsing:
		err = w.processIntentParsing(ctx, task)
	case TaskTypeLLMProcessing:
		err = w.processLLMTask(ctx, task)
	case TaskTypeRAGQuery:
		err = w.processRAGQuery(ctx, task)
	case TaskTypeResourcePlanning:
		err = w.processResourcePlanning(ctx, task)
	case TaskTypeManifestGeneration:
		err = w.processManifestGeneration(ctx, task)
	case TaskTypeGitOpsCommit:
		err = w.processGitOpsCommit(ctx, task)
	case TaskTypeDeployment:
		err = w.processDeployment(ctx, task)
	default:
		err = fmt.Errorf("unknown task type: %s", task.Type)
	}

	// Update task status
	completedTime := time.Now()
	task.CompletedAt = &completedTime
	task.Error = err

	if err != nil {
		task.Status = TaskStatusFailed
	} else {
		task.Status = TaskStatusCompleted
	}

	// Send result
	select {
	case w.Results <- task:
	case <-w.Context.Done():
		return
	}
}

// Placeholder task processing methods
func (w *Worker) processIntentParsing(ctx context.Context, task *Task) error {
	// Simulate intent parsing work
	select {
	case <-time.After(100 * time.Millisecond):
		task.OutputData = json.RawMessage(`{}`)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Worker) processLLMTask(ctx context.Context, task *Task) error {
	// Simulate LLM processing work
	select {
	case <-time.After(500 * time.Millisecond):
		task.OutputData = json.RawMessage(`{}`)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Worker) processRAGQuery(ctx context.Context, task *Task) error {
	// Simulate RAG query work
	select {
	case <-time.After(200 * time.Millisecond):
		task.OutputData = json.RawMessage(`{}`)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Worker) processResourcePlanning(ctx context.Context, task *Task) error {
	// Simulate resource planning work
	select {
	case <-time.After(300 * time.Millisecond):
		task.OutputData = json.RawMessage(`{}`)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Worker) processManifestGeneration(ctx context.Context, task *Task) error {
	// Simulate manifest generation work
	select {
	case <-time.After(400 * time.Millisecond):
		task.OutputData = json.RawMessage(`{}`)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Worker) processGitOpsCommit(ctx context.Context, task *Task) error {
	// Simulate GitOps commit work
	select {
	case <-time.After(1000 * time.Millisecond):
		task.OutputData = json.RawMessage(`{}`)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Worker) processDeployment(ctx context.Context, task *Task) error {
	// Simulate deployment work
	select {
	case <-time.After(2000 * time.Millisecond):
		task.OutputData = json.RawMessage(`{}`)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// GetTaskStatus returns the status of a specific task
func (e *ParallelProcessingEngine) GetTaskStatus(taskID string) *Task {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if task, exists := e.tasks[taskID]; exists {
		return task
	}
	return nil
}

// GetDependencyMetrics returns dependency-related metrics
func (e *ParallelProcessingEngine) GetDependencyMetrics() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	metrics := make(map[string]interface{})
	metrics["total_tasks"] = len(e.tasks)

	// Count tasks by status
	statusCounts := make(map[string]int)
	for _, task := range e.tasks {
		statusCounts[string(task.Status)]++
	}
	metrics["tasks_by_status"] = statusCounts

	// Count tasks by type
	typeCounts := make(map[string]int)
	for _, task := range e.tasks {
		typeCounts[string(task.Type)]++
	}
	metrics["tasks_by_type"] = typeCounts

	return metrics
}

// priorityToInt converts NetworkPriority to integer value
func priorityToInt(priority v1.NetworkPriority) int {
	switch priority {
	case v1.NetworkPriorityLow:
		return 1
	case v1.NetworkPriorityNormal:
		return 2
	case v1.NetworkPriorityHigh:
		return 3
	case v1.NetworkPriorityCritical:
		return 4
	default:
		return 2 // Default to normal priority
	}
}

