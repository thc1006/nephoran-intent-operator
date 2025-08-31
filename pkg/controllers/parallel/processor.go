package parallel

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

// NewParallelProcessor creates a new parallel processor
func NewParallelProcessor(config *ProcessorConfig) *ParallelProcessor {
	if config == nil {
		config = NewDefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())
	logger := ctrl.Log.WithName("parallel-processor")

	processor := &ParallelProcessor{
		maxConcurrency: config.MaxConcurrency,
		timeout:        config.DefaultTimeout,
		workerPool:     make(chan struct{}, config.MaxConcurrency),
		taskQueue:      make(chan TaskInterface, config.QueueSize),
		results:        make(chan *ControllerTaskResult, config.QueueSize),
		ctx:            ctx,
		cancel:         cancel,
		logger:         logger,
		metrics:        NewProcessorMetrics(),
	}

	// Initialize worker pool
	for i := 0; i < config.MaxConcurrency; i++ {
		processor.workerPool <- struct{}{}
	}

	// Start workers
	processor.startWorkers()

	return processor
}

// NewProcessorMetrics creates new processor metrics
func NewProcessorMetrics() *ProcessorMetrics {
	return &ProcessorMetrics{
		MinTime: time.Duration(^uint64(0) >> 1), // Max duration value
	}
}

// SubmitTask submits a task for parallel execution
func (pp *ParallelProcessor) SubmitTask(task TaskInterface) error {
	select {
	case pp.taskQueue <- task:
		pp.updateMetrics(func(m *ProcessorMetrics) {
			m.TotalTasks++
		})
		return nil
	case <-pp.ctx.Done():
		return fmt.Errorf("processor is shutting down")
	default:
		return fmt.Errorf("task queue is full")
	}
}

// SubmitTasks submits multiple tasks for parallel execution
func (pp *ParallelProcessor) SubmitTasks(tasks []TaskInterface) error {
	for _, task := range tasks {
		if err := pp.SubmitTask(task); err != nil {
			return fmt.Errorf("failed to submit task %s: %w", task.GetID(), err)
		}
	}
	return nil
}

// GetResult retrieves a task result (non-blocking)
func (pp *ParallelProcessor) GetResult() *ControllerTaskResult {
	select {
	case result := <-pp.results:
		return result
	default:
		return nil
	}
}

// WaitForResult waits for a task result with timeout
func (pp *ParallelProcessor) WaitForResult(timeout time.Duration) *ControllerTaskResult {
	select {
	case result := <-pp.results:
		return result
	case <-time.After(timeout):
		return nil
	case <-pp.ctx.Done():
		return nil
	}
}

// WaitForAllTasks waits for all submitted tasks to complete
func (pp *ParallelProcessor) WaitForAllTasks() {
	pp.wg.Wait()
}

// WaitForAllTasksWithTimeout waits for all tasks with a timeout
func (pp *ParallelProcessor) WaitForAllTasksWithTimeout(timeout time.Duration) error {
	done := make(chan struct{})
	go func() {
		pp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for tasks to complete")
	case <-pp.ctx.Done():
		return fmt.Errorf("processor shutdown")
	}
}

// GetMetrics returns current processing metrics
func (pp *ParallelProcessor) GetMetrics() ProcessorMetrics {
	pp.metrics.mutex.RLock()
	defer pp.metrics.mutex.RUnlock()
	return *pp.metrics
}

// Shutdown gracefully shuts down the processor
func (pp *ParallelProcessor) Shutdown() error {
	pp.logger.Info("Shutting down parallel processor")
	
	// Close task queue to signal no more tasks
	close(pp.taskQueue)
	
	// Wait for all tasks to complete
	pp.wg.Wait()
	
	// Cancel context to stop workers
	pp.cancel()
	
	// Close results channel
	close(pp.results)
	
	pp.logger.Info("Parallel processor shutdown complete")
	return nil
}

// startWorkers starts the worker goroutines
func (pp *ParallelProcessor) startWorkers() {
	for i := 0; i < pp.maxConcurrency; i++ {
		go pp.worker(i)
	}
}

// worker processes tasks from the task queue
func (pp *ParallelProcessor) worker(workerID int) {
	logger := pp.logger.WithValues("worker_id", workerID)
	logger.Info("Worker started")
	
	for {
		select {
		case task, ok := <-pp.taskQueue:
			if !ok {
				logger.Info("Task queue closed, worker stopping")
				return
			}
			pp.processTask(task, logger)
		case <-pp.ctx.Done():
			logger.Info("Worker stopping due to context cancellation")
			return
		}
	}
}

// processTask processes a single task
func (pp *ParallelProcessor) processTask(task TaskInterface, logger logr.Logger) {
	pp.wg.Add(1)
	defer pp.wg.Done()

	// Get a slot from worker pool
	<-pp.workerPool
	defer func() {
		pp.workerPool <- struct{}{}
	}()

	startTime := time.Now()
	taskLogger := logger.WithValues("task_id", task.GetID())
	taskLogger.Info("Processing task")

	// Create task context with timeout
	taskTimeout := task.GetTimeout()
	if taskTimeout == 0 {
		taskTimeout = pp.timeout
	}
	
	ctx, cancel := context.WithTimeout(pp.ctx, taskTimeout)
	defer cancel()

	// Execute the task
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("task panicked: %v", r)
				taskLogger.Error(err, "Task execution panicked")
			}
		}()
		err = task.Execute(ctx)
	}()

	duration := time.Since(startTime)
	success := err == nil

	// Create result
	result := &ControllerTaskResult{
		TaskID:      task.GetID(),
		Success:     success,
		Error:       err,
		Duration:    duration,
		CompletedAt: time.Now(),
	}

	// Update metrics
	pp.updateMetrics(func(m *ProcessorMetrics) {
		m.CompletedTasks++
		if !success {
			m.FailedTasks++
		}
		
		// Update timing metrics
		if duration > m.MaxTime {
			m.MaxTime = duration
		}
		if duration < m.MinTime {
			m.MinTime = duration
		}
		
		// Calculate running average
		totalCompleted := m.CompletedTasks
		if totalCompleted > 0 {
			m.AverageTime = time.Duration((int64(m.AverageTime)*(totalCompleted-1) + int64(duration)) / totalCompleted)
		}
	})

	// Send result
	select {
	case pp.results <- result:
	case <-pp.ctx.Done():
		taskLogger.Info("Could not send result, processor shutting down")
	default:
		taskLogger.Info("Result channel full, dropping result")
	}

	if success {
		taskLogger.Info("Task completed successfully", "duration", duration)
	} else {
		taskLogger.Error(err, "Task failed", "duration", duration)
	}
}

// updateMetrics safely updates processor metrics
func (pp *ParallelProcessor) updateMetrics(updater func(*ProcessorMetrics)) {
	pp.metrics.mutex.Lock()
	defer pp.metrics.mutex.Unlock()
	updater(pp.metrics)
}

// NewParallelReconciler creates a new parallel reconciler
func NewParallelReconciler(processor *ParallelProcessor) *ParallelReconciler {
	return &ParallelReconciler{
		Processor: processor,
		Logger:    ctrl.Log.WithName("parallel-reconciler"),
	}
}

// ReconcileParallel executes reconciliation tasks in parallel
func (pr *ParallelReconciler) ReconcileParallel(ctx context.Context, tasks []TaskInterface) error {
	if len(tasks) == 0 {
		return nil
	}

	pr.Logger.Info("Starting parallel reconciliation", "task_count", len(tasks))

	// Submit all tasks
	if err := pr.Processor.SubmitTasks(tasks); err != nil {
		return fmt.Errorf("failed to submit tasks: %w", err)
	}

	// Wait for all tasks to complete with timeout
	timeout := 5 * time.Minute // Default timeout for reconciliation
	if err := pr.Processor.WaitForAllTasksWithTimeout(timeout); err != nil {
		return fmt.Errorf("timeout waiting for reconciliation tasks: %w", err)
	}

	// Collect results
	var errors []error
	for i := 0; i < len(tasks); i++ {
		result := pr.Processor.WaitForResult(1 * time.Second)
		if result != nil && !result.Success {
			errors = append(errors, result.Error)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("reconciliation failed with %d errors: %v", len(errors), errors)
	}

	pr.Logger.Info("Parallel reconciliation completed successfully")
	return nil
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(batchSize int, flushInterval time.Duration, processor func(context.Context, []TaskInterface) error) *BatchProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &BatchProcessor{
		batchSize:     batchSize,
		flushInterval: flushInterval,
		processor:     processor,
		taskBuffer:    make([]TaskInterface, 0, batchSize),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// ExecuteWithRetry executes a task with retry logic
func ExecuteWithRetry(ctx context.Context, task TaskInterface, maxRetries int, backoff time.Duration) error {
	var lastErr error
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(backoff * time.Duration(attempt)):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		
		err := task.Execute(ctx)
		if err == nil {
			return nil
		}
		
		lastErr = err
	}
	
	return fmt.Errorf("task failed after %d attempts: %w", maxRetries+1, lastErr)
}

// ExecuteTasksInBatches executes tasks in batches with specified batch size
func ExecuteTasksInBatches(ctx context.Context, tasks []Task, batchSize int) error {
	if batchSize <= 0 {
		batchSize = 1
	}
	
	for i := 0; i < len(tasks); i += batchSize {
		end := i + batchSize
		if end > len(tasks) {
			end = len(tasks)
		}
		
		batch := tasks[i:end]
		var wg sync.WaitGroup
		errors := make(chan error, len(batch))
		
		for _, task := range batch {
			wg.Add(1)
			go func(t Task) {
				defer wg.Done()
				if err := t.Execute(ctx); err != nil {
					errors <- err
				}
			}(task)
		}
		
		wg.Wait()
		close(errors)
		
		// Check for errors
		for err := range errors {
			if err != nil {
				return fmt.Errorf("batch execution failed: %w", err)
			}
		}
	}
	
	return nil
}