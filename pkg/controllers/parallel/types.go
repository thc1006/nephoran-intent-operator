package parallel

import (
	"context"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// TaskInterface represents a parallelizable task in controller processing
type TaskInterface interface {
	Execute(ctx context.Context) error
	GetID() string
	GetPriority() int
	GetTimeout() time.Duration
}

// ControllerTaskResult represents the result of a parallel task execution
type ControllerTaskResult struct {
	TaskID      string        `json:"task_id"`
	Success     bool          `json:"success"`
	Error       error         `json:"error,omitempty"`
	Duration    time.Duration `json:"duration"`
	CompletedAt time.Time     `json:"completed_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ParallelProcessor manages parallel execution of controller tasks
type ParallelProcessor struct {
	maxConcurrency int
	timeout        time.Duration
	workerPool     chan struct{}
	taskQueue      chan TaskInterface
	results        chan *ControllerTaskResult
	wg             sync.WaitGroup
	ctx            context.Context
	cancel         context.CancelFunc
	logger         logr.Logger
	metrics        *ProcessorMetrics
}

// ProcessorMetrics tracks parallel processing metrics
type ProcessorMetrics struct {
	TotalTasks     int64         `json:"total_tasks"`
	CompletedTasks int64         `json:"completed_tasks"`
	FailedTasks    int64         `json:"failed_tasks"`
	AverageTime    time.Duration `json:"average_time"`
	MaxTime        time.Duration `json:"max_time"`
	MinTime        time.Duration `json:"min_time"`
	mutex          sync.RWMutex
}

// ProcessorConfig contains configuration for parallel processing
type ProcessorConfig struct {
	MaxConcurrency   int           `json:"max_concurrency"`
	DefaultTimeout   time.Duration `json:"default_timeout"`
	QueueSize        int           `json:"queue_size"`
	EnableMetrics    bool          `json:"enable_metrics"`
	RetryAttempts    int           `json:"retry_attempts"`
	RetryBackoff     time.Duration `json:"retry_backoff"`
}

// ReconcileTask represents a Kubernetes reconciliation task
type ReconcileTask struct {
	ID       string
	Priority int
	Timeout  time.Duration
	Object   client.Object
	Req      ctrl.Request
	Client   client.Client
	Scheme   *runtime.Scheme
	Reconciler func(context.Context, ctrl.Request) (ctrl.Result, error)
}

// Execute implements the Task interface for ReconcileTask
func (rt *ReconcileTask) Execute(ctx context.Context) error {
	logger := log.FromContext(ctx).WithValues("task_id", rt.ID)
	logger.Info("Executing reconcile task")
	
	timeoutCtx, cancel := context.WithTimeout(ctx, rt.Timeout)
	defer cancel()
	
	result, err := rt.Reconciler(timeoutCtx, rt.Req)
	if err != nil {
		logger.Error(err, "Reconcile task failed")
		return err
	}
	
	logger.Info("Reconcile task completed", "result", result)
	return nil
}

// GetID implements the TaskInterface for ReconcileTask
func (rt *ReconcileTask) GetID() string {
	return rt.ID
}

// GetPriority implements the TaskInterface for ReconcileTask
func (rt *ReconcileTask) GetPriority() int {
	return rt.Priority
}

// GetTimeout implements the TaskInterface for ReconcileTask
func (rt *ReconcileTask) GetTimeout() time.Duration {
	return rt.Timeout
}

// ValidationTask represents a validation task that can run in parallel
type ValidationTask struct {
	ID       string
	Priority int
	Timeout  time.Duration
	Object   client.Object
	Rules    []ValidationRule
	Client   client.Client
}

// ValidationRule represents a validation rule
type ValidationRule struct {
	Name        string
	Description string
	Validator   func(context.Context, client.Object, client.Client) error
}

// Execute implements the TaskInterface for ValidationTask
func (vt *ValidationTask) Execute(ctx context.Context) error {
	logger := log.FromContext(ctx).WithValues("task_id", vt.ID)
	logger.Info("Executing validation task")
	
	timeoutCtx, cancel := context.WithTimeout(ctx, vt.Timeout)
	defer cancel()
	
	for _, rule := range vt.Rules {
		if err := rule.Validator(timeoutCtx, vt.Object, vt.Client); err != nil {
			logger.Error(err, "Validation rule failed", "rule", rule.Name)
			return err
		}
	}
	
	logger.Info("Validation task completed successfully")
	return nil
}

// GetID implements the TaskInterface for ValidationTask
func (vt *ValidationTask) GetID() string {
	return vt.ID
}

// GetPriority implements the TaskInterface for ValidationTask
func (vt *ValidationTask) GetPriority() int {
	return vt.Priority
}

// GetTimeout implements the TaskInterface for ValidationTask
func (vt *ValidationTask) GetTimeout() time.Duration {
	return vt.Timeout
}

// ProcessingTask represents a generic processing task
type ProcessingTask struct {
	ID       string
	Priority int
	Timeout  time.Duration
	Processor func(context.Context) error
	Metadata map[string]interface{}
}

// Execute implements the TaskInterface for ProcessingTask
func (pt *ProcessingTask) Execute(ctx context.Context) error {
	logger := log.FromContext(ctx).WithValues("task_id", pt.ID)
	logger.Info("Executing processing task")
	
	timeoutCtx, cancel := context.WithTimeout(ctx, pt.Timeout)
	defer cancel()
	
	err := pt.Processor(timeoutCtx)
	if err != nil {
		logger.Error(err, "Processing task failed")
		return err
	}
	
	logger.Info("Processing task completed successfully")
	return nil
}

// GetID implements the TaskInterface for ProcessingTask
func (pt *ProcessingTask) GetID() string {
	return pt.ID
}

// GetPriority implements the TaskInterface for ProcessingTask
func (pt *ProcessingTask) GetPriority() int {
	return pt.Priority
}

// GetTimeout implements the TaskInterface for ProcessingTask
func (pt *ProcessingTask) GetTimeout() time.Duration {
	return pt.Timeout
}

// ParallelReconciler provides parallel processing capabilities for controllers
type ParallelReconciler struct {
	Client    client.Client
	Scheme    *runtime.Scheme
	Processor *ParallelProcessor
	Logger    logr.Logger
}

// TaskPriorityQueue implements a priority queue for tasks
type TaskPriorityQueue struct {
	tasks []TaskInterface
	mutex sync.RWMutex
}

// Push adds a task to the priority queue
func (pq *TaskPriorityQueue) Push(task TaskInterface) {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()
	
	// Insert task maintaining priority order (higher priority first)
	inserted := false
	for i, existing := range pq.tasks {
		if task.GetPriority() > existing.GetPriority() {
			pq.tasks = append(pq.tasks[:i], append([]TaskInterface{task}, pq.tasks[i:]...)...)
			inserted = true
			break
		}
	}
	
	if !inserted {
		pq.tasks = append(pq.tasks, task)
	}
}

// Pop removes and returns the highest priority task
func (pq *TaskPriorityQueue) Pop() TaskInterface {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()
	
	if len(pq.tasks) == 0 {
		return nil
	}
	
	task := pq.tasks[0]
	pq.tasks = pq.tasks[1:]
	return task
}

// Len returns the number of tasks in the queue
func (pq *TaskPriorityQueue) Len() int {
	pq.mutex.RLock()
	defer pq.mutex.RUnlock()
	return len(pq.tasks)
}

// IsEmpty returns true if the queue is empty
func (pq *TaskPriorityQueue) IsEmpty() bool {
	return pq.Len() == 0
}

// BatchProcessor handles batch processing of related tasks
type BatchProcessor struct {
	batchSize      int
	flushInterval  time.Duration
	processor      func(context.Context, []TaskInterface) error
	taskBuffer     []TaskInterface
	mutex          sync.Mutex
	flushTimer     *time.Timer
	ctx            context.Context
	cancel         context.CancelFunc
}

// AddTask adds a task to the batch buffer
func (bp *BatchProcessor) AddTask(task TaskInterface) {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()
	
	bp.taskBuffer = append(bp.taskBuffer, task)
	
	if len(bp.taskBuffer) >= bp.batchSize {
		bp.flushBatch()
	} else if bp.flushTimer == nil {
		bp.flushTimer = time.AfterFunc(bp.flushInterval, func() {
			bp.mutex.Lock()
			defer bp.mutex.Unlock()
			bp.flushBatch()
		})
	}
}

// flushBatch processes the current batch of tasks
func (bp *BatchProcessor) flushBatch() {
	if len(bp.taskBuffer) == 0 {
		return
	}
	
	tasks := make([]TaskInterface, len(bp.taskBuffer))
	copy(tasks, bp.taskBuffer)
	bp.taskBuffer = bp.taskBuffer[:0]
	
	if bp.flushTimer != nil {
		bp.flushTimer.Stop()
		bp.flushTimer = nil
	}
	
	go func() {
		if err := bp.processor(bp.ctx, tasks); err != nil {
			// Log error - in a real implementation, this would use proper logging
			// logger.Error(err, "Batch processing failed")
		}
	}()
}

// Close gracefully shuts down the batch processor
func (bp *BatchProcessor) Close() error {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()
	
	bp.cancel()
	bp.flushBatch()
	
	if bp.flushTimer != nil {
		bp.flushTimer.Stop()
	}
	
	return nil
}

// Constants for parallel processing
const (
	DefaultMaxConcurrency = 10
	DefaultTimeout        = 30 * time.Second
	DefaultQueueSize      = 1000
	DefaultRetryAttempts  = 3
	DefaultRetryBackoff   = 1 * time.Second
	
	// Task priorities
	PriorityHigh   = 100
	PriorityNormal = 50
	PriorityLow    = 10
	
	// Task types
	TaskTypeReconcile   = "reconcile"
	TaskTypeValidation  = "validation"
	TaskTypeProcessing  = "processing"
	TaskTypeBatch       = "batch"
)

// Helper functions

// NewDefaultConfig returns a default parallel processor configuration
func NewDefaultConfig() *ProcessorConfig {
	return &ProcessorConfig{
		MaxConcurrency: DefaultMaxConcurrency,
		DefaultTimeout: DefaultTimeout,
		QueueSize:      DefaultQueueSize,
		EnableMetrics:  true,
		RetryAttempts:  DefaultRetryAttempts,
		RetryBackoff:   DefaultRetryBackoff,
	}
}

// NewTaskPriorityQueue creates a new priority queue for tasks
func NewTaskPriorityQueue() *TaskPriorityQueue {
	return &TaskPriorityQueue{
		tasks: make([]TaskInterface, 0),
	}
}

// Ensure we have the logr.Logger type available
var _ logr.Logger = ctrl.Log.WithName("parallel")