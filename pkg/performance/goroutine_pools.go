//go:build go1.24

package performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"golang.org/x/sync/errgroup"
	"k8s.io/klog/v2"
)

// EnhancedGoroutinePool provides advanced goroutine management with Go 1.24+ optimizations.

type EnhancedGoroutinePool struct {
	workStealingQueues []*WorkStealingQueue

	workers []*Worker

	scheduler *Scheduler

	affinityManager *CPUAffinityManager

	metrics *PoolMetrics

	config *PoolConfig

	shutdown chan struct{}

	wg sync.WaitGroup

	mu sync.RWMutex
}

// PoolConfig contains goroutine pool configuration.

type PoolConfig struct {
	MinWorkers int

	MaxWorkers int

	QueueSize int

	ScalingFactor float64

	IdleTimeout time.Duration

	TaskTimeout time.Duration

	EnableWorkStealing bool

	EnableCPUAffinity bool

	EnablePriorityQueue bool

	MetricsInterval time.Duration

	SpinCount int

	PreemptionEnabled bool
}

// WorkStealingQueue implements a lock-free work-stealing deque.

type WorkStealingQueue struct {
	tasks []unsafe.Pointer

	head int64

	tail int64

	mask int64

	steals int64

	victims int64

	mu sync.Mutex // Only for resize operations

}

// Worker represents a worker goroutine with CPU affinity.

type Worker struct {
	id int

	queue *WorkStealingQueue

	localTasks int64

	stolenTasks int64

	processedTasks int64

	failedTasks int64

	idleTime int64

	processingTime int64

	lastTaskTime time.Time

	cpuAffinity int

	context context.Context

	cancel context.CancelFunc

	priority Priority
}

// Scheduler manages task distribution and worker scaling.

type Scheduler struct {
	pendingTasks int64

	activeWorkers int64

	idleWorkers int64

	scalingDecisions int64

	loadAverage float64

	throughput float64

	lastScaleTime time.Time

	mu sync.RWMutex
}

// CPUAffinityManager manages CPU affinity for workers.

type CPUAffinityManager struct {
	cpuCount int

	affinityMap map[int]int // worker ID -> CPU core

	coreUtilization []float64

	numaNodes []NumaNode

	enabled bool

	mu sync.RWMutex
}

// NumaNode represents a NUMA node with CPU cores.

type NumaNode struct {
	ID int

	Cores []int

	Memory int64
}

// Task represents a task with priority and metadata.

type Task struct {
	ID uint64

	Function func() error

	Priority Priority

	Deadline time.Time

	Context context.Context

	Callback func(error)

	CreatedAt time.Time

	StartedAt time.Time

	CompletedAt time.Time

	Retries int

	MaxRetries int
}

// Priority defines task priority levels.

type Priority int

const (

	// PriorityLow holds prioritylow value.

	PriorityLow Priority = iota

	// PriorityNormal holds prioritynormal value.

	PriorityNormal

	// PriorityHigh holds priorityhigh value.

	PriorityHigh

	// PriorityCritical holds prioritycritical value.

	PriorityCritical
)

// PoolMetrics tracks advanced pool performance metrics.

type PoolMetrics struct {
	ActiveWorkers int64

	IdleWorkers int64

	TotalTasks int64

	CompletedTasks int64

	FailedTasks int64

	StolenTasks int64

	QueuedTasks int64

	AverageWaitTime int64 // nanoseconds

	AverageProcessTime int64 // nanoseconds

	Throughput float64 // tasks per second

	CPUUtilization float64

	MemoryUsage int64

	GoroutineCount int64

	ScalingEvents int64

	PreemptionEvents int64

	AffinityViolations int64
}

// NewEnhancedGoroutinePool creates a new enhanced goroutine pool.

func NewEnhancedGoroutinePool(config *PoolConfig) *EnhancedGoroutinePool {

	if config == nil {

		config = DefaultPoolConfig()

	}

	numCPU := runtime.NumCPU()

	if config.MaxWorkers > numCPU*4 {

		config.MaxWorkers = numCPU * 4 // Reasonable upper bound

	}

	pool := &EnhancedGoroutinePool{

		workStealingQueues: make([]*WorkStealingQueue, config.MaxWorkers),

		workers: make([]*Worker, 0, config.MaxWorkers),

		scheduler: NewScheduler(),

		metrics: &PoolMetrics{},

		config: config,

		shutdown: make(chan struct{}),
	}

	// Initialize work-stealing queues.

	for i := range pool.workStealingQueues {

		pool.workStealingQueues[i] = NewWorkStealingQueue(config.QueueSize)

	}

	// Initialize CPU affinity manager.

	if config.EnableCPUAffinity {

		pool.affinityManager = NewCPUAffinityManager(numCPU)

	}

	// Start with minimum workers.

	for range config.MinWorkers {

		pool.addWorker()

	}

	// Start background tasks.

	pool.startBackgroundTasks()

	return pool

}

// DefaultPoolConfig returns default pool configuration.

func DefaultPoolConfig() *PoolConfig {

	return &PoolConfig{

		MinWorkers: runtime.NumCPU(),

		MaxWorkers: runtime.NumCPU() * 2,

		QueueSize: 1024,

		ScalingFactor: 1.5,

		IdleTimeout: 5 * time.Minute,

		TaskTimeout: 30 * time.Second,

		EnableWorkStealing: true,

		EnableCPUAffinity: true,

		EnablePriorityQueue: true,

		MetricsInterval: 10 * time.Second,

		SpinCount: 100,

		PreemptionEnabled: true,
	}

}

// NewWorkStealingQueue creates a new work-stealing queue.

func NewWorkStealingQueue(size int) *WorkStealingQueue {

	// Ensure size is power of 2.

	powerOf2Size := 1

	for powerOf2Size < size {

		powerOf2Size <<= 1

	}

	return &WorkStealingQueue{

		tasks: make([]unsafe.Pointer, powerOf2Size),

		mask: int64(powerOf2Size - 1),
	}

}

// PushBottom adds a task to the bottom of the queue (owner only).

func (wsq *WorkStealingQueue) PushBottom(task *Task) bool {

	tail := atomic.LoadInt64(&wsq.tail)

	head := atomic.LoadInt64(&wsq.head)

	size := tail - head

	// Check if queue is full.

	if size >= int64(len(wsq.tasks)) {

		return false

	}

	// Store task.

	atomic.StorePointer(&wsq.tasks[tail&wsq.mask], unsafe.Pointer(task))

	atomic.StoreInt64(&wsq.tail, tail+1)

	return true

}

// PopBottom removes a task from the bottom of the queue (owner only).

func (wsq *WorkStealingQueue) PopBottom() *Task {

	tail := atomic.LoadInt64(&wsq.tail) - 1

	atomic.StoreInt64(&wsq.tail, tail)

	head := atomic.LoadInt64(&wsq.head)

	if tail < head {

		// Queue is empty.

		atomic.StoreInt64(&wsq.tail, head)

		return nil

	}

	taskPtr := atomic.LoadPointer(&wsq.tasks[tail&wsq.mask])

	if tail == head {

		// Last element, need to compete with stealers.

		if !atomic.CompareAndSwapInt64(&wsq.head, head, head+1) {

			// Stealer won.

			atomic.StoreInt64(&wsq.tail, head+1)

			return nil

		}

	}

	return (*Task)(taskPtr)

}

// PopTop removes a task from the top of the queue (stealers).

func (wsq *WorkStealingQueue) PopTop() *Task {

	head := atomic.LoadInt64(&wsq.head)

	tail := atomic.LoadInt64(&wsq.tail)

	if head >= tail {

		// Queue is empty.

		return nil

	}

	taskPtr := atomic.LoadPointer(&wsq.tasks[head&wsq.mask])

	if !atomic.CompareAndSwapInt64(&wsq.head, head, head+1) {

		// Failed to steal.

		return nil

	}

	atomic.AddInt64(&wsq.steals, 1)

	return (*Task)(taskPtr)

}

// Size returns the approximate queue size.

func (wsq *WorkStealingQueue) Size() int64 {

	tail := atomic.LoadInt64(&wsq.tail)

	head := atomic.LoadInt64(&wsq.head)

	size := tail - head

	if size < 0 {

		size = 0

	}

	return size

}

// NewScheduler creates a new task scheduler.

func NewScheduler() *Scheduler {

	return &Scheduler{

		lastScaleTime: time.Now(),
	}

}

// NewCPUAffinityManager creates a new CPU affinity manager.

func NewCPUAffinityManager(cpuCount int) *CPUAffinityManager {

	return &CPUAffinityManager{

		cpuCount: cpuCount,

		affinityMap: make(map[int]int),

		coreUtilization: make([]float64, cpuCount),

		enabled: true,
	}

}

// SubmitTask submits a task to the pool with priority.

func (pool *EnhancedGoroutinePool) SubmitTask(task *Task) error {

	if task == nil {

		return fmt.Errorf("task cannot be nil")

	}

	task.CreatedAt = time.Now()

	if task.Context == nil {

		task.Context = context.Background()

	}

	atomic.AddInt64(&pool.metrics.TotalTasks, 1)

	atomic.AddInt64(&pool.scheduler.pendingTasks, 1)

	// Try to submit to least loaded worker queue.

	bestWorkerIdx := pool.findBestWorker()

	if bestWorkerIdx >= 0 && bestWorkerIdx < len(pool.workStealingQueues) {

		if pool.workStealingQueues[bestWorkerIdx].PushBottom(task) {

			atomic.AddInt64(&pool.metrics.QueuedTasks, 1)

			return nil

		}

	}

	// All queues are full, try to scale up.

	if pool.shouldScaleUp() {

		if pool.addWorker() {

			// Retry with new worker.

			newWorkerIdx := len(pool.workers) - 1

			if newWorkerIdx >= 0 && newWorkerIdx < len(pool.workStealingQueues) {

				if pool.workStealingQueues[newWorkerIdx].PushBottom(task) {

					atomic.AddInt64(&pool.metrics.QueuedTasks, 1)

					return nil

				}

			}

		}

	}

	// Failed to submit task.

	atomic.AddInt64(&pool.metrics.FailedTasks, 1)

	return fmt.Errorf("failed to submit task: all queues full")

}

// SubmitWithTimeout submits a task with a timeout.

func (pool *EnhancedGoroutinePool) SubmitWithTimeout(task *Task, timeout time.Duration) error {

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	defer cancel()

	task.Context = ctx

	task.Deadline = time.Now().Add(timeout)

	return pool.SubmitTask(task)

}

// SubmitBatch submits multiple tasks as a batch.

func (pool *EnhancedGoroutinePool) SubmitBatch(tasks []*Task) error {

	if len(tasks) == 0 {

		return nil

	}

	// Use error group for batch submission.

	g, ctx := errgroup.WithContext(context.Background())

	for _, task := range tasks {

		taskCopy := task

		g.Go(func() error {

			taskCopy.Context = ctx

			return pool.SubmitTask(taskCopy)

		})

	}

	return g.Wait()

}

// findBestWorker finds the worker with the least load.

func (pool *EnhancedGoroutinePool) findBestWorker() int {

	pool.mu.RLock()

	defer pool.mu.RUnlock()

	if len(pool.workers) == 0 {

		return -1

	}

	bestIdx := 0

	minQueueSize := pool.workStealingQueues[0].Size()

	for i := 1; i < len(pool.workers); i++ {

		queueSize := pool.workStealingQueues[i].Size()

		if queueSize < minQueueSize {

			minQueueSize = queueSize

			bestIdx = i

		}

	}

	return bestIdx

}

// shouldScaleUp determines if the pool should add more workers.

func (pool *EnhancedGoroutinePool) shouldScaleUp() bool {

	pool.mu.RLock()

	defer pool.mu.RUnlock()

	if len(pool.workers) >= pool.config.MaxWorkers {

		return false

	}

	if time.Since(pool.scheduler.lastScaleTime) < 5*time.Second {

		return false

	}

	// Check if average queue size exceeds threshold.

	totalQueueSize := int64(0)

	for _, queue := range pool.workStealingQueues[:len(pool.workers)] {

		totalQueueSize += queue.Size()

	}

	if len(pool.workers) > 0 {

		avgQueueSize := float64(totalQueueSize) / float64(len(pool.workers))

		return avgQueueSize > 10 // Scale up if average queue size > 10

	}

	return true

}

// shouldScaleDown determines if the pool should remove workers.

func (pool *EnhancedGoroutinePool) shouldScaleDown() bool {

	pool.mu.RLock()

	defer pool.mu.RUnlock()

	if len(pool.workers) <= pool.config.MinWorkers {

		return false

	}

	if time.Since(pool.scheduler.lastScaleTime) < 30*time.Second {

		return false

	}

	// Check if workers are idle.

	idleWorkers := atomic.LoadInt64(&pool.scheduler.idleWorkers)

	activeWorkers := atomic.LoadInt64(&pool.scheduler.activeWorkers)

	totalWorkers := idleWorkers + activeWorkers

	if totalWorkers > 0 {

		idleRatio := float64(idleWorkers) / float64(totalWorkers)

		return idleRatio > 0.7 // Scale down if >70% workers are idle

	}

	return false

}

// addWorker adds a new worker to the pool.

func (pool *EnhancedGoroutinePool) addWorker() bool {

	pool.mu.Lock()

	defer pool.mu.Unlock()

	if len(pool.workers) >= pool.config.MaxWorkers {

		return false

	}

	workerID := len(pool.workers)

	ctx, cancel := context.WithCancel(context.Background())

	worker := &Worker{

		id: workerID,

		queue: pool.workStealingQueues[workerID],

		context: ctx,

		cancel: cancel,

		priority: PriorityNormal,
	}

	// Set CPU affinity if enabled.

	if pool.config.EnableCPUAffinity && pool.affinityManager != nil {

		worker.cpuAffinity = pool.affinityManager.AssignCPU(workerID)

	}

	pool.workers = append(pool.workers, worker)

	atomic.AddInt64(&pool.scheduler.activeWorkers, 1)

	atomic.AddInt64(&pool.metrics.ScalingEvents, 1)

	pool.scheduler.lastScaleTime = time.Now()

	pool.wg.Add(1)

	go pool.workerLoop(worker)

	klog.V(2).Infof("Added worker %d to pool (total: %d)", workerID, len(pool.workers))

	return true

}

// removeWorker removes a worker from the pool.

func (pool *EnhancedGoroutinePool) removeWorker() bool {

	pool.mu.Lock()

	defer pool.mu.Unlock()

	if len(pool.workers) <= pool.config.MinWorkers {

		return false

	}

	// Find the most idle worker.

	mostIdleIdx := pool.findMostIdleWorker()

	if mostIdleIdx == -1 {

		return false

	}

	worker := pool.workers[mostIdleIdx]

	worker.cancel() // Signal worker to stop

	// Remove from slice.

	pool.workers = append(pool.workers[:mostIdleIdx], pool.workers[mostIdleIdx+1:]...)

	atomic.AddInt64(&pool.scheduler.activeWorkers, -1)

	atomic.AddInt64(&pool.metrics.ScalingEvents, 1)

	pool.scheduler.lastScaleTime = time.Now()

	klog.V(2).Infof("Removed worker %d from pool (total: %d)", worker.id, len(pool.workers))

	return true

}

// findMostIdleWorker finds the worker that has been idle the longest.

func (pool *EnhancedGoroutinePool) findMostIdleWorker() int {

	mostIdleIdx := -1

	maxIdleTime := int64(0)

	for i, worker := range pool.workers {

		idleTime := atomic.LoadInt64(&worker.idleTime)

		if idleTime > maxIdleTime {

			maxIdleTime = idleTime

			mostIdleIdx = i

		}

	}

	return mostIdleIdx

}

// workerLoop is the main loop for a worker goroutine.

func (pool *EnhancedGoroutinePool) workerLoop(worker *Worker) {

	defer pool.wg.Done()

	defer func() {

		atomic.AddInt64(&pool.scheduler.activeWorkers, -1)

		if r := recover(); r != nil {

			klog.Errorf("Worker %d panic: %v", worker.id, r)

		}

	}()

	spinCount := 0

	idleStart := time.Now()

	for {

		select {

		case <-worker.context.Done():

			return

		case <-pool.shutdown:

			return

		default:

		}

		// Try to get task from own queue first.

		task := worker.queue.PopBottom()

		if task == nil && pool.config.EnableWorkStealing {

			// Try to steal from other workers.

			task = pool.stealTask(worker.id)

		}

		if task != nil {

			// Reset idle tracking.

			if spinCount > 0 {

				idleDuration := time.Since(idleStart)

				atomic.AddInt64(&worker.idleTime, idleDuration.Nanoseconds())

				atomic.AddInt64(&pool.scheduler.idleWorkers, -1)

				spinCount = 0

			}

			// Process task.

			pool.processTask(worker, task)

			idleStart = time.Now()

		} else {

			// No task available.

			spinCount++

			if spinCount == 1 {

				// First idle spin.

				atomic.AddInt64(&pool.scheduler.idleWorkers, 1)

			}

			if spinCount < pool.config.SpinCount {

				// Spin-wait for better latency.

				runtime.Gosched()

			} else {

				// Sleep to save CPU.

				time.Sleep(100 * time.Microsecond)

			}

			// Check if worker should be removed due to inactivity.

			if spinCount > 10000 && pool.shouldScaleDown() {

				return

			}

		}

	}

}

// stealTask attempts to steal a task from another worker's queue.

func (pool *EnhancedGoroutinePool) stealTask(workerID int) *Task {

	pool.mu.RLock()

	workerCount := len(pool.workers)

	pool.mu.RUnlock()

	// Try to steal from a random worker (avoid systematic bias).

	for attempts := range 3 {

		victimID := (workerID + 1 + attempts) % workerCount

		if victimID != workerID && victimID < len(pool.workStealingQueues) {

			if task := pool.workStealingQueues[victimID].PopTop(); task != nil {

				atomic.AddInt64(&pool.metrics.StolenTasks, 1)

				atomic.AddInt64(&pool.workers[workerID].stolenTasks, 1)

				atomic.AddInt64(&pool.workStealingQueues[victimID].victims, 1)

				return task

			}

		}

	}

	return nil

}

// processTask processes a single task.

func (pool *EnhancedGoroutinePool) processTask(worker *Worker, task *Task) {

	start := time.Now()

	task.StartedAt = start

	atomic.AddInt64(&pool.scheduler.pendingTasks, -1)

	atomic.AddInt64(&pool.metrics.QueuedTasks, -1)

	defer func() {

		task.CompletedAt = time.Now()

		processingTime := task.CompletedAt.Sub(task.StartedAt)

		waitTime := task.StartedAt.Sub(task.CreatedAt)

		atomic.AddInt64(&worker.processedTasks, 1)

		atomic.AddInt64(&worker.processingTime, processingTime.Nanoseconds())

		atomic.AddInt64(&pool.metrics.CompletedTasks, 1)

		// Update metrics.

		pool.updateTaskMetrics(waitTime, processingTime)

		if r := recover(); r != nil {

			atomic.AddInt64(&worker.failedTasks, 1)

			atomic.AddInt64(&pool.metrics.FailedTasks, 1)

			err := fmt.Errorf("task panic: %v", r)

			if task.Callback != nil {

				task.Callback(err)

			}

			klog.Errorf("Task %d panic: %v", task.ID, r)

		}

	}()

	// Check task timeout.

	if !task.Deadline.IsZero() && time.Now().After(task.Deadline) {

		err := fmt.Errorf("task %d timed out", task.ID)

		if task.Callback != nil {

			task.Callback(err)

		}

		return

	}

	// Execute task.

	var err error

	if task.Function != nil {

		err = task.Function()

	}

	// Handle result.

	if err != nil {

		atomic.AddInt64(&worker.failedTasks, 1)

		atomic.AddInt64(&pool.metrics.FailedTasks, 1)

		// Retry logic.

		if task.Retries < task.MaxRetries {

			task.Retries++

			// Re-submit with exponential backoff.

			go func() {

				backoff := time.Duration(task.Retries) * time.Second

				time.Sleep(backoff)

				pool.SubmitTask(task)

			}()

			return

		}

	}

	if task.Callback != nil {

		task.Callback(err)

	}

}

// updateTaskMetrics updates task processing metrics.

func (pool *EnhancedGoroutinePool) updateTaskMetrics(waitTime, processingTime time.Duration) {

	// Update average wait time.

	currentAvgWait := atomic.LoadInt64(&pool.metrics.AverageWaitTime)

	newAvgWait := (currentAvgWait + waitTime.Nanoseconds()) / 2

	atomic.StoreInt64(&pool.metrics.AverageWaitTime, newAvgWait)

	// Update average processing time.

	currentAvgProcess := atomic.LoadInt64(&pool.metrics.AverageProcessTime)

	newAvgProcess := (currentAvgProcess + processingTime.Nanoseconds()) / 2

	atomic.StoreInt64(&pool.metrics.AverageProcessTime, newAvgProcess)

}

// AssignCPU assigns a CPU core to a worker (simplified implementation).

func (cam *CPUAffinityManager) AssignCPU(workerID int) int {

	if !cam.enabled {

		return -1

	}

	cam.mu.Lock()

	defer cam.mu.Unlock()

	// Simple round-robin assignment.

	cpuCore := workerID % cam.cpuCount

	cam.affinityMap[workerID] = cpuCore

	return cpuCore

}

// startBackgroundTasks starts background maintenance tasks.

func (pool *EnhancedGoroutinePool) startBackgroundTasks() {

	// Metrics collection.

	go func() {

		ticker := time.NewTicker(pool.config.MetricsInterval)

		defer ticker.Stop()

		for {

			select {

			case <-ticker.C:

				pool.updateMetrics()

			case <-pool.shutdown:

				return

			}

		}

	}()

	// Auto-scaling.

	go func() {

		ticker := time.NewTicker(5 * time.Second)

		defer ticker.Stop()

		for {

			select {

			case <-ticker.C:

				if pool.shouldScaleUp() {

					pool.addWorker()

				} else if pool.shouldScaleDown() {

					pool.removeWorker()

				}

			case <-pool.shutdown:

				return

			}

		}

	}()

}

// updateMetrics updates pool metrics.

func (pool *EnhancedGoroutinePool) updateMetrics() {

	pool.mu.RLock()

	workerCount := len(pool.workers)

	pool.mu.RUnlock()

	atomic.StoreInt64(&pool.metrics.ActiveWorkers, int64(workerCount))

	atomic.StoreInt64(&pool.metrics.GoroutineCount, int64(runtime.NumGoroutine()))

	// Calculate throughput (tasks per second).

	completedTasks := atomic.LoadInt64(&pool.metrics.CompletedTasks)

	if pool.scheduler.lastScaleTime.Before(time.Now().Add(-1 * time.Second)) {

		elapsed := time.Since(pool.scheduler.lastScaleTime).Seconds()

		if elapsed > 0 {

			pool.metrics.Throughput = float64(completedTasks) / elapsed

		}

	}

	// Update CPU utilization (simplified).

	var memStats runtime.MemStats

	runtime.ReadMemStats(&memStats)

	atomic.StoreInt64(&pool.metrics.MemoryUsage, int64(memStats.HeapInuse))

}

// GetMetrics returns current pool metrics.

func (pool *EnhancedGoroutinePool) GetMetrics() PoolMetrics {

	return PoolMetrics{

		ActiveWorkers: atomic.LoadInt64(&pool.metrics.ActiveWorkers),

		IdleWorkers: atomic.LoadInt64(&pool.scheduler.idleWorkers),

		TotalTasks: atomic.LoadInt64(&pool.metrics.TotalTasks),

		CompletedTasks: atomic.LoadInt64(&pool.metrics.CompletedTasks),

		FailedTasks: atomic.LoadInt64(&pool.metrics.FailedTasks),

		StolenTasks: atomic.LoadInt64(&pool.metrics.StolenTasks),

		QueuedTasks: atomic.LoadInt64(&pool.metrics.QueuedTasks),

		AverageWaitTime: atomic.LoadInt64(&pool.metrics.AverageWaitTime),

		AverageProcessTime: atomic.LoadInt64(&pool.metrics.AverageProcessTime),

		Throughput: pool.metrics.Throughput,

		CPUUtilization: pool.metrics.CPUUtilization,

		MemoryUsage: atomic.LoadInt64(&pool.metrics.MemoryUsage),

		GoroutineCount: int64(atomic.LoadInt64(&pool.metrics.GoroutineCount)),

		ScalingEvents: atomic.LoadInt64(&pool.metrics.ScalingEvents),

		PreemptionEvents: atomic.LoadInt64(&pool.metrics.PreemptionEvents),

		AffinityViolations: atomic.LoadInt64(&pool.metrics.AffinityViolations),
	}

}

// GetWorkerStats returns statistics for all workers.

func (pool *EnhancedGoroutinePool) GetWorkerStats() []WorkerStats {

	pool.mu.RLock()

	defer pool.mu.RUnlock()

	stats := make([]WorkerStats, len(pool.workers))

	for i, worker := range pool.workers {

		stats[i] = WorkerStats{

			ID: worker.id,

			ProcessedTasks: atomic.LoadInt64(&worker.processedTasks),

			FailedTasks: atomic.LoadInt64(&worker.failedTasks),

			StolenTasks: atomic.LoadInt64(&worker.stolenTasks),

			IdleTime: atomic.LoadInt64(&worker.idleTime),

			ProcessingTime: atomic.LoadInt64(&worker.processingTime),

			CPUAffinity: worker.cpuAffinity,

			QueueSize: worker.queue.Size(),
		}

	}

	return stats

}

// WorkerStats contains statistics for a single worker.

type WorkerStats struct {
	ID int

	ProcessedTasks int64

	FailedTasks int64

	StolenTasks int64

	IdleTime int64

	ProcessingTime int64

	CPUAffinity int

	QueueSize int64
}

// GetAverageWaitTime returns the average task wait time in milliseconds.

func (pool *EnhancedGoroutinePool) GetAverageWaitTime() float64 {

	waitTime := atomic.LoadInt64(&pool.metrics.AverageWaitTime)

	return float64(waitTime) / 1e6 // Convert to milliseconds

}

// GetAverageProcessingTime returns the average task processing time in milliseconds.

func (pool *EnhancedGoroutinePool) GetAverageProcessingTime() float64 {

	processTime := atomic.LoadInt64(&pool.metrics.AverageProcessTime)

	return float64(processTime) / 1e6 // Convert to milliseconds

}

// GetThroughput returns the current throughput in tasks per second.

func (pool *EnhancedGoroutinePool) GetThroughput() float64 {

	return pool.metrics.Throughput

}

// Shutdown gracefully shuts down the goroutine pool.

func (pool *EnhancedGoroutinePool) Shutdown(ctx context.Context) error {

	close(pool.shutdown)

	// Cancel all workers.

	pool.mu.RLock()

	for _, worker := range pool.workers {

		worker.cancel()

	}

	pool.mu.RUnlock()

	// Wait for all workers to finish.

	done := make(chan struct{})

	go func() {

		pool.wg.Wait()

		close(done)

	}()

	// Wait with timeout.

	select {

	case <-done:

		// All workers finished.

	case <-ctx.Done():

		return ctx.Err()

	}

	// Log final metrics.

	metrics := pool.GetMetrics()

	klog.Infof("Goroutine pool shutdown - Total tasks: %d, Completed: %d, Failed: %d, Throughput: %.2f tasks/sec",

		metrics.TotalTasks,

		metrics.CompletedTasks,

		metrics.FailedTasks,

		metrics.Throughput,
	)

	return nil

}

// ResetMetrics resets all performance metrics.

func (pool *EnhancedGoroutinePool) ResetMetrics() {

	atomic.StoreInt64(&pool.metrics.TotalTasks, 0)

	atomic.StoreInt64(&pool.metrics.CompletedTasks, 0)

	atomic.StoreInt64(&pool.metrics.FailedTasks, 0)

	atomic.StoreInt64(&pool.metrics.StolenTasks, 0)

	atomic.StoreInt64(&pool.metrics.QueuedTasks, 0)

	atomic.StoreInt64(&pool.metrics.AverageWaitTime, 0)

	atomic.StoreInt64(&pool.metrics.AverageProcessTime, 0)

	atomic.StoreInt64(&pool.metrics.MemoryUsage, 0)

	atomic.StoreInt64(&pool.metrics.ScalingEvents, 0)

	atomic.StoreInt64(&pool.metrics.PreemptionEvents, 0)

	atomic.StoreInt64(&pool.metrics.AffinityViolations, 0)

	pool.metrics.Throughput = 0

	pool.metrics.CPUUtilization = 0

	// Reset scheduler metrics.

	atomic.StoreInt64(&pool.scheduler.pendingTasks, 0)

	atomic.StoreInt64(&pool.scheduler.idleWorkers, 0)

	// Reset worker metrics.

	pool.mu.RLock()

	for _, worker := range pool.workers {

		atomic.StoreInt64(&worker.processedTasks, 0)

		atomic.StoreInt64(&worker.failedTasks, 0)

		atomic.StoreInt64(&worker.stolenTasks, 0)

		atomic.StoreInt64(&worker.idleTime, 0)

		atomic.StoreInt64(&worker.processingTime, 0)

	}

	pool.mu.RUnlock()

}
