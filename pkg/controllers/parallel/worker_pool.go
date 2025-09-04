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
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
)

// NewWorkerPool creates a new worker pool.

func NewWorkerPool(name string, workerCount int, processor TaskProcessor, logger logr.Logger) *WorkerPool {
	pool := &WorkerPool{
		name: name,

		workerCount: workerCount,

		workers: make([]*Worker, workerCount),

		taskQueue: make(chan *Task, workerCount*2), // Buffer for better throughput

		resultQueue: make(chan *TaskResult, workerCount*2),

		logger: logger,

		stopChan: make(chan struct{}),
	}

	// Create workers.

	for i := range workerCount {

		worker := &Worker{
			id: i,

			poolName: name,

			taskQueue: pool.taskQueue,

			resultQueue: pool.resultQueue,

			processor: processor,

			lastActivity: time.Now(),

			logger: logger.WithValues("worker", i, "pool", name),

			stopChan: make(chan struct{}),
		}

		pool.workers[i] = worker

	}

	return pool
}

// Start starts all workers in the pool.

func (wp *WorkerPool) Start(ctx context.Context) error {
	wp.mutex.Lock()

	defer wp.mutex.Unlock()

	wp.logger.Info("Starting worker pool", "workers", wp.workerCount)

	// Start all workers.

	for _, worker := range wp.workers {
		go worker.Start(ctx)
	}

	// Start result processor.

	go wp.processResults(ctx)

	return nil
}

// Stop stops all workers in the pool.

func (wp *WorkerPool) Stop() {
	wp.mutex.Lock()

	defer wp.mutex.Unlock()

	wp.logger.Info("Stopping worker pool")

	// Signal stop to all workers.

	close(wp.stopChan)

	// Stop individual workers.

	for _, worker := range wp.workers {
		worker.Stop()
	}

	// Close channels.

	close(wp.taskQueue)

	close(wp.resultQueue)
}

// SubmitTask submits a task to the worker pool.

func (wp *WorkerPool) SubmitTask(task *Task) error {
	select {

	case wp.taskQueue <- task:

		return nil

	default:

		return fmt.Errorf("worker pool %s task queue full", wp.name)

	}
}

// GetMetrics returns worker pool metrics.

func (wp *WorkerPool) GetMetrics() map[string]interface{} {
	wp.mutex.RLock()

	defer wp.mutex.RUnlock()

	return map[string]interface{}{}
}

// GetHealth returns health status of the worker pool.

func (wp *WorkerPool) GetHealth() map[string]interface{} {
	activeWorkers := atomic.LoadInt32(&wp.activeWorkers)

	queueLength := len(wp.taskQueue)

	queueCapacity := cap(wp.taskQueue)

	healthy := activeWorkers > 0 && queueLength < queueCapacity

	health := map[string]interface{}{}

	if !healthy {

		issues := []string{}

		if activeWorkers == 0 {
			issues = append(issues, "no active workers")
		}

		if queueLength >= queueCapacity {
			issues = append(issues, "queue full")
		}

		health["issues"] = issues

	}

	return health
}

// processResults processes task results from workers.

func (wp *WorkerPool) processResults(ctx context.Context) {
	for {
		select {

		case result := <-wp.resultQueue:

			wp.handleTaskResult(result)

		case <-wp.stopChan:

			return

		case <-ctx.Done():

			return

		}
	}
}

// handleTaskResult handles a completed task result.

func (wp *WorkerPool) handleTaskResult(result *TaskResult) {
	// Update pool metrics.

	atomic.AddInt64(&wp.processedTasks, 1)

	if !result.Success {
		atomic.AddInt64(&wp.failedTasks, 1)
	}

	// Update average latency.

	wp.updateAverageLatency(result.Duration)

	// Update resource usage.

	atomic.AddInt64(&wp.memoryUsage, result.MemoryUsed)

	wp.updateCPUUsage(result.CPUUsed)

	wp.logger.Info("Task completed",

		"taskId", result.TaskID,

		"success", result.Success,

		"duration", result.Duration,

		"workerId", result.WorkerID)
}

// updateAverageLatency updates the average latency calculation.

func (wp *WorkerPool) updateAverageLatency(duration time.Duration) {
	wp.mutex.Lock()

	defer wp.mutex.Unlock()

	processed := atomic.LoadInt64(&wp.processedTasks)

	if processed == 1 {
		wp.averageLatency = duration
	} else {

		// Exponential moving average.

		alpha := 0.1

		wp.averageLatency = time.Duration(float64(wp.averageLatency)*(1-alpha) + float64(duration)*alpha)

	}
}

// updateCPUUsage updates the CPU usage calculation.

func (wp *WorkerPool) updateCPUUsage(cpuUsed float64) {
	wp.mutex.Lock()

	defer wp.mutex.Unlock()

	// Simple moving average.

	wp.cpuUsage = (wp.cpuUsage + cpuUsed) / 2.0
}

// Worker implementation.

// Start starts the worker.

func (w *Worker) Start(ctx context.Context) {
	w.logger.Info("Worker starting")

	for {
		select {

		case task := <-w.taskQueue:

			w.processTask(task)

		case <-w.stopChan:

			w.logger.Info("Worker stopping")

			return

		case <-ctx.Done():

			w.logger.Info("Worker context cancelled")

			return

		}
	}
}

// Stop stops the worker.

func (w *Worker) Stop() {
	close(w.stopChan)
}

// processTask method removed - using the implementation from engine.go

// updateMetrics updates worker metrics.

func (w *Worker) updateMetrics(duration time.Duration) {
	w.mutex.Lock()

	defer w.mutex.Unlock()

	processed := atomic.LoadInt64(&w.totalProcessed)

	if processed == 1 {
		w.averageLatency = duration
	} else {

		// Exponential moving average.

		alpha := 0.1

		w.averageLatency = time.Duration(float64(w.averageLatency)*(1-alpha) + float64(duration)*alpha)

	}
}

// getCurrentMemoryUsage returns current memory usage for this worker.

func (w *Worker) getCurrentMemoryUsage() int64 {
	var m runtime.MemStats

	runtime.ReadMemStats(&m)

	return int64(m.Alloc)
}

// getCurrentCPUUsage returns current CPU usage for this worker.

func (w *Worker) getCurrentCPUUsage() float64 {
	// This would implement actual per-worker CPU usage calculation.

	// For now, returning a simulated value.

	return 0.1 + float64(w.id)*0.05 // Vary by worker ID
}

// GetMetrics returns worker metrics.

func (w *Worker) GetMetrics() map[string]interface{} {
	w.mutex.RLock()

	defer w.mutex.RUnlock()

	return map[string]interface{}{}
}

// IsHealthy returns whether the worker is healthy.

func (w *Worker) IsHealthy() bool {
	w.mutex.RLock()

	defer w.mutex.RUnlock()

	// Consider worker unhealthy if it hasn't been active recently.

	return time.Since(w.lastActivity) < 5*time.Minute
}

// Supporting queue implementation.

// NewPriorityQueue creates a new priority queue.

func NewPriorityQueue(maxSize int) *PriorityQueue {
	pq := &PriorityQueue{
		tasks: make([]*Task, 0, maxSize),

		maxSize: maxSize,

		currentSize: 0,
	}

	pq.notEmpty = sync.NewCond(&pq.mutex)

	return pq
}

// Enqueue adds a task to the priority queue.

func (pq *PriorityQueue) Enqueue(task *Task) error {
	pq.mutex.Lock()

	defer pq.mutex.Unlock()

	if pq.currentSize >= pq.maxSize {
		return fmt.Errorf("priority queue is full")
	}

	// Insert task in priority order (higher priority first).

	inserted := false

	for i, existingTask := range pq.tasks {
		if task.Priority > existingTask.Priority {

			// Insert at position i by making space and moving elements.

			pq.tasks = append(pq.tasks, nil)   // Grow the slice
			copy(pq.tasks[i+1:], pq.tasks[i:]) // Shift elements to the right
			pq.tasks[i] = task                 // Insert the new task

			inserted = true

			break

		}
	}

	if !inserted {
		// Append to end (lowest priority).

		pq.tasks = append(pq.tasks, task)
	}

	pq.currentSize++

	pq.notEmpty.Signal()

	return nil
}

// Dequeue removes and returns the highest priority task.

func (pq *PriorityQueue) Dequeue() *Task {
	pq.mutex.Lock()

	defer pq.mutex.Unlock()

	for pq.currentSize == 0 {
		pq.notEmpty.Wait()
	}

	if pq.currentSize == 0 {
		return nil
	}

	task := pq.tasks[0]

	pq.tasks = pq.tasks[1:]

	pq.currentSize--

	return task
}

// DequeueWithTimeout removes and returns the highest priority task with timeout.

func (pq *PriorityQueue) DequeueWithTimeout(timeout time.Duration) *Task {
	done := make(chan *Task, 1)

	go func() {
		task := pq.Dequeue()

		done <- task
	}()

	select {

	case task := <-done:

		return task

	case <-time.After(timeout):

		return nil

	}
}

// Peek returns the highest priority task without removing it.

func (pq *PriorityQueue) Peek() *Task {
	pq.mutex.RLock()

	defer pq.mutex.RUnlock()

	if pq.currentSize == 0 {
		return nil
	}

	return pq.tasks[0]
}

// Size returns the current size of the queue.

func (pq *PriorityQueue) Size() int {
	pq.mutex.RLock()

	defer pq.mutex.RUnlock()

	return pq.currentSize
}

// IsEmpty returns whether the queue is empty.

func (pq *PriorityQueue) IsEmpty() bool {
	return pq.Size() == 0
}

// IsFull returns whether the queue is full.

func (pq *PriorityQueue) IsFull() bool {
	pq.mutex.RLock()

	defer pq.mutex.RUnlock()

	return pq.currentSize >= pq.maxSize
}

// Clear removes all tasks from the queue.

func (pq *PriorityQueue) Clear() {
	pq.mutex.Lock()

	defer pq.mutex.Unlock()

	pq.tasks = pq.tasks[:0]

	pq.currentSize = 0
}

// GetTasks returns a copy of all tasks in the queue (for monitoring).

func (pq *PriorityQueue) GetTasks() []*Task {
	pq.mutex.RLock()

	defer pq.mutex.RUnlock()

	tasks := make([]*Task, len(pq.tasks))

	copy(tasks, pq.tasks)

	return tasks
}

