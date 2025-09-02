//go:build !disable_rag && !test

package rag

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// ParallelChunkProcessor provides high-performance parallel chunk processing.

type ParallelChunkProcessor struct {
	logger *zap.Logger

	workerPool *ChunkWorkerPool

	loadBalancer ParallelLoadBalancer

	memoryPool *sync.Pool

	metrics *parallelMetrics

	maxRetries int

	retryDelay time.Duration
}

// ChunkWorkerPool manages a pool of chunk processing workers.

type ChunkWorkerPool struct {
	workers []*ChunkWorker

	taskQueue chan *ChunkTask

	resultQueue chan *ChunkResult

	maxWorkers int

	activeWorkers int32

	mu sync.RWMutex

	wg sync.WaitGroup
}

// ChunkWorker represents a single worker in the pool.

type ChunkWorker struct {
	id int

	pool *ChunkWorkerPool

	taskChannel chan *ChunkTask

	quit chan bool

	metrics *workerMetrics
}

// ChunkTask represents a chunk processing task.

type ChunkTask struct {
	ID string

	Chunk ProcessedChunk

	Processor func(ProcessedChunk) error

	Priority int

	Deadline time.Time

	Retries int
}

// ChunkResult represents the result of chunk processing.

type ChunkResult struct {
	TaskID string

	Error error

	Duration time.Duration

	WorkerID int
}

// ParallelLoadBalancer interface for different load balancing strategies.

type ParallelLoadBalancer interface {
	SelectWorker(workers []*ChunkWorker, task *ChunkTask) *ChunkWorker

	UpdateStats(workerID int, duration time.Duration, success bool)
}

// RoundRobinBalancer implements round-robin load balancing.

type RoundRobinBalancer struct {
	current uint64
}

// LeastLoadedBalancer implements least-loaded load balancing.

type LeastLoadedBalancer struct {
	workerLoads map[int]*atomic.Int64

	mu sync.RWMutex
}

// WorkStealingBalancer implements work stealing for better utilization.

type WorkStealingBalancer struct {
	queues map[int]chan *ChunkTask

	mu sync.RWMutex
}

// parallelMetrics tracks performance metrics for parallel processing.

type parallelMetrics struct {
	tasksProcessed prometheus.Counter

	tasksFailed prometheus.Counter

	processingDuration prometheus.Histogram

	queueDepth prometheus.Gauge

	activeWorkers prometheus.Gauge

	workStealing prometheus.Counter
}

// workerMetrics tracks individual worker performance.

type workerMetrics struct {
	tasksProcessed atomic.Int64

	tasksFailed atomic.Int64

	totalDuration atomic.Int64 // nanoseconds

	lastActive atomic.Int64 // unix timestamp
}

// ParallelConfig holds configuration for parallel processing.

type ParallelConfig struct {
	MaxWorkers int

	QueueSize int

	LoadBalancer string // "round-robin", "least-loaded", "work-stealing"

	MaxRetries int

	RetryDelay time.Duration

	EnableWorkStealing bool
}

// NewParallelChunkProcessor creates a new parallel chunk processor.

func NewParallelChunkProcessor(
	logger *zap.Logger,

	config ParallelConfig,
) *ParallelChunkProcessor {
	if config.MaxWorkers == 0 {
		config.MaxWorkers = 8
	}

	if config.QueueSize == 0 {
		config.QueueSize = config.MaxWorkers * 10
	}

	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	if config.RetryDelay == 0 {
		config.RetryDelay = time.Second
	}

	metrics := &parallelMetrics{
		tasksProcessed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rag_parallel_tasks_processed_total",

			Help: "Total number of chunk tasks processed",
		}),

		tasksFailed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rag_parallel_tasks_failed_total",

			Help: "Total number of chunk tasks failed",
		}),

		processingDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "rag_parallel_processing_duration_seconds",

			Help: "Duration of chunk processing operations",

			Buckets: prometheus.DefBuckets,
		}),

		queueDepth: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "rag_parallel_queue_depth",

			Help: "Current depth of the task queue",
		}),

		activeWorkers: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "rag_parallel_active_workers",

			Help: "Number of active workers",
		}),

		workStealing: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rag_parallel_work_stealing_total",

			Help: "Total number of work stealing events",
		}),
	}

	// Register metrics.

	prometheus.MustRegister(

		metrics.tasksProcessed,

		metrics.tasksFailed,

		metrics.processingDuration,

		metrics.queueDepth,

		metrics.activeWorkers,

		metrics.workStealing,
	)

	pool := &ChunkWorkerPool{
		workers: make([]*ChunkWorker, 0, config.MaxWorkers),

		taskQueue: make(chan *ChunkTask, config.QueueSize),

		resultQueue: make(chan *ChunkResult, config.QueueSize),

		maxWorkers: config.MaxWorkers,
	}

	// Create load balancer.

	var balancer ParallelLoadBalancer

	switch config.LoadBalancer {

	case "least-loaded":

		balancer = NewLeastLoadedBalancer(config.MaxWorkers)

	case "work-stealing":

		balancer = NewWorkStealingBalancer(config.MaxWorkers)

	default:

		balancer = &RoundRobinBalancer{}

	}

	processor := &ParallelChunkProcessor{
		logger: logger,

		workerPool: pool,

		loadBalancer: balancer,

		memoryPool: &sync.Pool{
			New: func() interface{} {
				return &ChunkTask{}
			},
		},

		metrics: metrics,

		maxRetries: config.MaxRetries,

		retryDelay: config.RetryDelay,
	}

	// Start workers.

	processor.startWorkers()

	// Start monitoring.

	go processor.monitorQueues()

	return processor
}

// ProcessChunks processes multiple chunks in parallel.

func (p *ParallelChunkProcessor) ProcessChunks(
	ctx context.Context,

	chunks []ProcessedChunk,

	processor func(ProcessedChunk) error,
) error {
	tasks := make([]*ChunkTask, len(chunks))

	for i, chunk := range chunks {

		task := p.memoryPool.Get().(*ChunkTask)

		task.ID = fmt.Sprintf("chunk-%d", chunk.ChunkIndex)

		task.Chunk = chunk

		task.Processor = processor

		task.Priority = 0

		task.Deadline = time.Now().Add(5 * time.Minute)

		task.Retries = 0

		tasks[i] = task

	}

	return p.ProcessTasks(ctx, tasks)
}

// ProcessTasks processes a batch of tasks.

func (p *ParallelChunkProcessor) ProcessTasks(
	ctx context.Context,

	tasks []*ChunkTask,
) error {
	resultChan := make(chan *ChunkResult, len(tasks))

	var wg sync.WaitGroup

	// Submit tasks.

	for _, task := range tasks {

		wg.Add(1)

		go func(t *ChunkTask) {
			defer wg.Done()

			result := p.processTask(ctx, t)

			resultChan <- result
		}(task)

	}

	// Wait for completion.

	go func() {
		wg.Wait()

		close(resultChan)
	}()

	// Collect results.

	var errors []error

	for result := range resultChan {

		if result.Error != nil {
			errors = append(errors, fmt.Errorf("task %s failed: %w", result.TaskID, result.Error))
		}

		// Return task to pool.

		task := &ChunkTask{ID: result.TaskID}

		p.memoryPool.Put(task)

	}

	if len(errors) > 0 {
		return fmt.Errorf("parallel processing failed with %d errors: %v", len(errors), errors)
	}

	return nil
}

// processTask processes a single task with retry logic.

func (p *ParallelChunkProcessor) processTask(ctx context.Context, task *ChunkTask) *ChunkResult {
	result := &ChunkResult{
		TaskID: task.ID,
	}

	// Select worker.

	worker := p.loadBalancer.SelectWorker(p.workerPool.workers, task)

	if worker == nil {

		// Fallback to queue.

		select {

		case p.workerPool.taskQueue <- task:

		case <-ctx.Done():

			result.Error = ctx.Err()

			return result

		}

		// Wait for result.

		select {

		case res := <-p.workerPool.resultQueue:

			return res

		case <-ctx.Done():

			result.Error = ctx.Err()

			return result

		}

	}

	// Direct dispatch to worker.

	select {

	case worker.taskChannel <- task:

		// Wait for result.

		select {

		case res := <-p.workerPool.resultQueue:

			return res

		case <-ctx.Done():

			result.Error = ctx.Err()

			return result

		}

	case <-ctx.Done():

		result.Error = ctx.Err()

		return result

	}
}

// startWorkers initializes and starts worker goroutines.

func (p *ParallelChunkProcessor) startWorkers() {
	for i := 0; i < p.workerPool.maxWorkers; i++ {

		worker := &ChunkWorker{
			id: i,

			pool: p.workerPool,

			taskChannel: make(chan *ChunkTask, 10),

			quit: make(chan bool),

			metrics: &workerMetrics{},
		}

		p.workerPool.workers = append(p.workerPool.workers, worker)

		go worker.start(p)

	}
}

// start runs the worker loop.

func (w *ChunkWorker) start(processor *ParallelChunkProcessor) {
	w.pool.wg.Add(1)

	defer w.pool.wg.Done()

	atomic.AddInt32(&w.pool.activeWorkers, 1)

	defer atomic.AddInt32(&w.pool.activeWorkers, -1)

	processor.metrics.activeWorkers.Inc()

	defer processor.metrics.activeWorkers.Dec()

	for {
		select {

		case task := <-w.taskChannel:

			w.processTask(processor, task)

		case task := <-w.pool.taskQueue:

			w.processTask(processor, task)

		case <-w.quit:

			return

		}
	}
}

// processTask handles a single task.

func (w *ChunkWorker) processTask(processor *ParallelChunkProcessor, task *ChunkTask) {
	start := time.Now()

	w.metrics.lastActive.Store(time.Now().Unix())

	result := &ChunkResult{
		TaskID: task.ID,

		WorkerID: w.id,
	}

	// Execute with retries.

	var err error

	for attempt := 0; attempt <= task.Retries; attempt++ {

		if attempt > 0 {
			time.Sleep(processor.retryDelay * time.Duration(attempt))
		}

		err = task.Processor(task.Chunk)

		if err == nil {
			break
		}

		if attempt < task.Retries {
			processor.logger.Warn("Chunk processing failed, retrying",

				zap.String("task_id", task.ID),

				zap.Int("attempt", attempt+1),

				zap.Error(err))
		}

	}

	result.Error = err

	result.Duration = time.Since(start)

	// Update metrics.

	if err == nil {

		w.metrics.tasksProcessed.Add(1)

		processor.metrics.tasksProcessed.Inc()

	} else {

		w.metrics.tasksFailed.Add(1)

		processor.metrics.tasksFailed.Inc()

	}

	w.metrics.totalDuration.Add(int64(result.Duration))

	processor.metrics.processingDuration.Observe(result.Duration.Seconds())

	// Update load balancer stats.

	processor.loadBalancer.UpdateStats(w.id, result.Duration, err == nil)

	// Send result.

	select {

	case w.pool.resultQueue <- result:

	default:

		processor.logger.Warn("Result queue full, dropping result",

			zap.String("task_id", task.ID))

	}
}

// monitorQueues monitors queue depths and performance.

func (p *ParallelChunkProcessor) monitorQueues() {
	ticker := time.NewTicker(5 * time.Second)

	defer ticker.Stop()

	for range ticker.C {

		p.metrics.queueDepth.Set(float64(len(p.workerPool.taskQueue)))

		p.metrics.activeWorkers.Set(float64(atomic.LoadInt32(&p.workerPool.activeWorkers)))

		// Log if queue is getting full.

		queueUtilization := float64(len(p.workerPool.taskQueue)) / float64(cap(p.workerPool.taskQueue))

		if queueUtilization > 0.8 {
			p.logger.Warn("Task queue high utilization",

				zap.Float64("utilization", queueUtilization),

				zap.Int("queue_depth", len(p.workerPool.taskQueue)))
		}

	}
}

// Shutdown gracefully shuts down the processor.

func (p *ParallelChunkProcessor) Shutdown() {
	// Stop all workers.

	for _, worker := range p.workerPool.workers {
		close(worker.quit)
	}

	p.workerPool.wg.Wait()

	close(p.workerPool.taskQueue)

	close(p.workerPool.resultQueue)
}

// Load balancer implementations.

// SelectWorker performs selectworker operation.

func (b *RoundRobinBalancer) SelectWorker(workers []*ChunkWorker, task *ChunkTask) *ChunkWorker {
	if len(workers) == 0 {
		return nil
	}

	index := atomic.AddUint64(&b.current, 1) % uint64(len(workers))

	return workers[index]
}

// UpdateStats performs updatestats operation.

func (b *RoundRobinBalancer) UpdateStats(workerID int, duration time.Duration, success bool) {
	// Round-robin doesn't need stats.
}

// NewLeastLoadedBalancer performs newleastloadedbalancer operation.

func NewLeastLoadedBalancer(maxWorkers int) *LeastLoadedBalancer {
	loads := make(map[int]*atomic.Int64)

	for i := 0; i < maxWorkers; i++ {
		loads[i] = &atomic.Int64{}
	}

	return &LeastLoadedBalancer{
		workerLoads: loads,
	}
}

// SelectWorker performs selectworker operation.

func (b *LeastLoadedBalancer) SelectWorker(workers []*ChunkWorker, task *ChunkTask) *ChunkWorker {
	if len(workers) == 0 {
		return nil
	}

	b.mu.RLock()

	defer b.mu.RUnlock()

	var selected *ChunkWorker

	minLoad := int64(^uint64(0) >> 1) // Max int64

	for _, worker := range workers {

		load := b.workerLoads[worker.id].Load()

		if load < minLoad {

			minLoad = load

			selected = worker

		}

	}

	if selected != nil {
		b.workerLoads[selected.id].Add(1)
	}

	return selected
}

// UpdateStats performs updatestats operation.

func (b *LeastLoadedBalancer) UpdateStats(workerID int, duration time.Duration, success bool) {
	b.mu.RLock()

	defer b.mu.RUnlock()

	if load, ok := b.workerLoads[workerID]; ok {
		load.Add(-1)
	}
}

// NewWorkStealingBalancer performs newworkstealingbalancer operation.

func NewWorkStealingBalancer(maxWorkers int) *WorkStealingBalancer {
	queues := make(map[int]chan *ChunkTask)

	for i := 0; i < maxWorkers; i++ {
		queues[i] = make(chan *ChunkTask, 10)
	}

	return &WorkStealingBalancer{
		queues: queues,
	}
}

// SelectWorker performs selectworker operation.

func (b *WorkStealingBalancer) SelectWorker(workers []*ChunkWorker, task *ChunkTask) *ChunkWorker {
	// Try to find worker with smallest queue.

	b.mu.RLock()

	defer b.mu.RUnlock()

	var selected *ChunkWorker

	minQueueSize := int(^uint(0) >> 1)

	for _, worker := range workers {
		if queue, ok := b.queues[worker.id]; ok {

			queueSize := len(queue)

			if queueSize < minQueueSize {

				minQueueSize = queueSize

				selected = worker

			}

		}
	}

	return selected
}

// UpdateStats performs updatestats operation.

func (b *WorkStealingBalancer) UpdateStats(workerID int, duration time.Duration, success bool) {
	// Work stealing uses queue sizes, not duration stats.
}

// ProcessDocumentChunks processes chunks for a loaded document.

func (pcp *ParallelChunkProcessor) ProcessDocumentChunks(ctx context.Context, doc *LoadedDocument) ([]*DocumentChunk, error) {
	// Convert document to chunks (simplified implementation).

	chunkSize := 1000 // Default chunk size

	content := doc.Content

	var chunks []*DocumentChunk

	for i := 0; i < len(content); i += chunkSize {

		end := i + chunkSize

		if end > len(content) {
			end = len(content)
		}

		chunk := &DocumentChunk{
			ID: fmt.Sprintf("%s-chunk-%d", doc.ID, i/chunkSize),

			DocumentID: doc.ID,

			Content: content[i:end],

			ChunkIndex: i / chunkSize,

			DocumentMetadata: doc.Metadata,
		}

		chunks = append(chunks, chunk)

	}

	return chunks, nil
}

// GetMetrics returns processor metrics.

func (pcp *ParallelChunkProcessor) GetMetrics() interface{} {
	return map[string]interface{}{
		"processed_tasks": pcp.metrics.tasksProcessed,

		"failed_tasks": pcp.metrics.tasksFailed,

		"active_workers": pcp.metrics.activeWorkers,

		"queue_depth": pcp.metrics.queueDepth,
	}
}
