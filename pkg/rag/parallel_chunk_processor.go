package rag

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// ParallelChunkProcessor handles parallel processing of document chunks
type ParallelChunkProcessor struct {
	config           *ParallelChunkConfig
	logger           *slog.Logger
	chunkingService  *ChunkingService
	embeddingService *EmbeddingService
	workerPool       *WorkerPool
	taskQueue        chan *ChunkTask
	resultQueue      chan *ChunkResult
	metrics          *ParallelProcessingMetrics
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
}

// ParallelChunkConfig holds configuration for parallel chunk processing
type ParallelChunkConfig struct {
	// Worker pool configuration
	NumWorkers          int           `json:"num_workers"`           // Number of parallel workers
	WorkerQueueSize     int           `json:"worker_queue_size"`     // Size of task queue per worker
	MaxTasksPerWorker   int           `json:"max_tasks_per_worker"`  // Max tasks a worker can handle
	
	// Batching configuration
	BatchSize           int           `json:"batch_size"`            // Chunks to process in batch
	BatchTimeout        time.Duration `json:"batch_timeout"`         // Timeout for batch collection
	EnableDynamicBatching bool        `json:"enable_dynamic_batching"` // Adjust batch size based on load
	
	// Performance tuning
	EnableCPUAffinity   bool          `json:"enable_cpu_affinity"`   // Pin workers to CPU cores
	EnableMemoryPooling bool          `json:"enable_memory_pooling"` // Use memory pools for chunks
	MaxMemoryPerWorker  int64         `json:"max_memory_per_worker"` // Memory limit per worker
	
	// Load balancing
	LoadBalancingStrategy string      `json:"load_balancing_strategy"` // "round_robin", "least_loaded", "hash_based"
	EnableWorkStealing    bool        `json:"enable_work_stealing"`    // Allow idle workers to steal work
	
	// Error handling
	MaxRetries          int           `json:"max_retries"`           // Max retries per chunk
	RetryBackoff        time.Duration `json:"retry_backoff"`         // Backoff between retries
	ErrorThreshold      float64       `json:"error_threshold"`       // Error rate to trigger circuit breaker
}

// WorkerPool manages a pool of chunk processing workers
type WorkerPool struct {
	workers         []*ChunkWorker
	taskDistributor *TaskDistributor
	config          *ParallelChunkConfig
	logger          *slog.Logger
	mu              sync.RWMutex
}

// ChunkWorker represents a worker that processes chunks
type ChunkWorker struct {
	id              int
	taskQueue       chan *ChunkTask
	resultQueue     chan *ChunkResult
	processor       *ChunkProcessor
	metrics         *WorkerMetrics
	cpuAffinity     int
	memoryPool      *MemoryPool
	lastActiveTime  time.Time
	isActive        atomic.Bool
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

// ChunkTask represents a chunk processing task
type ChunkTask struct {
	ID              string
	Document        *LoadedDocument
	ChunkData       string
	ChunkIndex      int
	StartOffset     int
	EndOffset       int
	Priority        int
	RetryCount      int
	SubmittedAt     time.Time
	Metadata        map[string]interface{}
}

// ChunkResult represents the result of chunk processing
type ChunkResult struct {
	TaskID          string
	Chunk           *DocumentChunk
	Embedding       []float32
	ProcessingTime  time.Duration
	WorkerID        int
	Error           error
	RetryCount      int
}

// ChunkProcessor handles the actual chunk processing logic
type ChunkProcessor struct {
	chunkingService  *ChunkingService
	embeddingService *EmbeddingService
	config           *ParallelChunkConfig
	logger           *slog.Logger
}

// TaskDistributor distributes tasks among workers
type TaskDistributor struct {
	strategy        string
	workers         []*ChunkWorker
	roundRobinIndex atomic.Int32
	hashSeed        uint32
	mu              sync.RWMutex
}

// WorkerMetrics tracks metrics for individual workers
type WorkerMetrics struct {
	TasksProcessed   atomic.Int64
	TasksFailed      atomic.Int64
	TotalProcessingTime atomic.Int64
	AverageLatency   time.Duration
	CurrentLoad      atomic.Int32
	LastError        error
	LastErrorTime    time.Time
	mu               sync.RWMutex
}

// ParallelProcessingMetrics tracks overall parallel processing metrics
type ParallelProcessingMetrics struct {
	TotalTasksSubmitted  atomic.Int64
	TotalTasksProcessed  atomic.Int64
	TotalTasksFailed     atomic.Int64
	TotalRetries         atomic.Int64
	AverageQueueDepth    float64
	AverageProcessingTime time.Duration
	WorkerUtilization    map[int]float64
	Throughput           float64 // Tasks per second
	ErrorRate            float64
	LastUpdated          time.Time
	mu                   sync.RWMutex
}

// MemoryPool manages memory allocation for chunk processing
type MemoryPool struct {
	buffers         chan []byte
	bufferSize      int
	maxBuffers      int
	allocatedCount  atomic.Int32
}

// NewParallelChunkProcessor creates a new parallel chunk processor
func NewParallelChunkProcessor(config *ParallelChunkConfig, chunkingService *ChunkingService, embeddingService *EmbeddingService) *ParallelChunkProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Set default number of workers based on CPU cores
	if config.NumWorkers == 0 {
		config.NumWorkers = runtime.NumCPU()
	}
	
	processor := &ParallelChunkProcessor{
		config:           config,
		logger:           slog.Default().With("component", "parallel-chunk-processor"),
		chunkingService:  chunkingService,
		embeddingService: embeddingService,
		taskQueue:        make(chan *ChunkTask, config.NumWorkers*config.WorkerQueueSize),
		resultQueue:      make(chan *ChunkResult, config.NumWorkers*config.WorkerQueueSize),
		metrics:          &ParallelProcessingMetrics{
			WorkerUtilization: make(map[int]float64),
			LastUpdated:       time.Now(),
		},
		ctx:              ctx,
		cancel:           cancel,
	}
	
	// Initialize worker pool
	processor.workerPool = NewWorkerPool(config, chunkingService, embeddingService)
	
	// Start workers
	processor.startWorkers()
	
	// Start metrics collector
	processor.wg.Add(1)
	go processor.metricsCollector()
	
	// Start result processor
	processor.wg.Add(1)
	go processor.resultProcessor()
	
	return processor
}

// ProcessDocumentChunks processes document chunks in parallel
func (pcp *ParallelChunkProcessor) ProcessDocumentChunks(ctx context.Context, doc *LoadedDocument) ([]*DocumentChunk, error) {
	startTime := time.Now()
	
	pcp.logger.Info("Starting parallel chunk processing",
		"doc_id", doc.ID,
		"content_length", len(doc.Content),
		"workers", pcp.config.NumWorkers)
	
	// Analyze document structure first
	structure, err := pcp.chunkingService.analyzeDocumentStructure(doc.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze document structure: %w", err)
	}
	
	// Create chunk tasks based on structure
	tasks := pcp.createChunkTasks(doc, structure)
	
	// Submit tasks to workers
	resultChan := make(chan *ChunkResult, len(tasks))
	pcp.submitTasks(ctx, tasks, resultChan)
	
	// Collect results
	chunks := pcp.collectResults(ctx, len(tasks), resultChan)
	
	// Post-process chunks
	processedChunks := pcp.postProcessChunks(chunks, doc, structure)
	
	processingTime := time.Since(startTime)
	pcp.logger.Info("Parallel chunk processing completed",
		"doc_id", doc.ID,
		"chunks_created", len(processedChunks),
		"processing_time", processingTime,
		"throughput", float64(len(processedChunks))/processingTime.Seconds())
	
	return processedChunks, nil
}

// createChunkTasks creates processing tasks from document structure
func (pcp *ParallelChunkProcessor) createChunkTasks(doc *LoadedDocument, structure *DocumentStructure) []*ChunkTask {
	var tasks []*ChunkTask
	
	// Create tasks for sections
	for i, section := range structure.Sections {
		task := &ChunkTask{
			ID:          fmt.Sprintf("%s_section_%d", doc.ID, i),
			Document:    doc,
			ChunkData:   section.Content,
			ChunkIndex:  i,
			StartOffset: section.StartOffset,
			EndOffset:   section.EndOffset,
			Priority:    pcp.calculatePriority(section),
			SubmittedAt: time.Now(),
			Metadata: map[string]interface{}{
				"section_title": section.Title,
				"section_level": section.Level,
			},
		}
		tasks = append(tasks, task)
	}
	
	// If no sections, create tasks based on size
	if len(tasks) == 0 {
		tasks = pcp.createSizeBasedTasks(doc)
	}
	
	return tasks
}

// createSizeBasedTasks creates tasks based on content size
func (pcp *ParallelChunkProcessor) createSizeBasedTasks(doc *LoadedDocument) []*ChunkTask {
	var tasks []*ChunkTask
	
	chunkSize := pcp.chunkingService.config.ChunkSize
	content := doc.Content
	
	for i := 0; i < len(content); i += chunkSize {
		end := i + chunkSize
		if end > len(content) {
			end = len(content)
		}
		
		task := &ChunkTask{
			ID:          fmt.Sprintf("%s_chunk_%d", doc.ID, i/chunkSize),
			Document:    doc,
			ChunkData:   content[i:end],
			ChunkIndex:  i / chunkSize,
			StartOffset: i,
			EndOffset:   end,
			Priority:    1,
			SubmittedAt: time.Now(),
		}
		tasks = append(tasks, task)
	}
	
	return tasks
}

// calculatePriority calculates task priority based on section importance
func (pcp *ParallelChunkProcessor) calculatePriority(section Section) int {
	priority := 1
	
	// Higher priority for top-level sections
	if section.Level == 1 {
		priority = 3
	} else if section.Level == 2 {
		priority = 2
	}
	
	// Higher priority for sections with specific keywords
	importantKeywords := []string{"abstract", "introduction", "summary", "conclusion", "requirements"}
	lowerTitle := strings.ToLower(section.Title)
	for _, keyword := range importantKeywords {
		if strings.Contains(lowerTitle, keyword) {
			priority += 2
			break
		}
	}
	
	return priority
}

// submitTasks submits tasks to the worker pool
func (pcp *ParallelChunkProcessor) submitTasks(ctx context.Context, tasks []*ChunkTask, resultChan chan<- *ChunkResult) {
	// Sort tasks by priority (higher priority first)
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Priority > tasks[j].Priority
	})
	
	// Submit tasks
	for _, task := range tasks {
		select {
		case pcp.taskQueue <- task:
			pcp.metrics.TotalTasksSubmitted.Add(1)
		case <-ctx.Done():
			pcp.logger.Warn("Context cancelled while submitting tasks")
			return
		}
	}
}

// collectResults collects processing results
func (pcp *ParallelChunkProcessor) collectResults(ctx context.Context, expectedResults int, resultChan <-chan *ChunkResult) []*DocumentChunk {
	var chunks []*DocumentChunk
	resultsReceived := 0
	
	timeout := time.NewTimer(pcp.config.BatchTimeout * time.Duration(expectedResults))
	defer timeout.Stop()
	
	for resultsReceived < expectedResults {
		select {
		case result := <-resultChan:
			resultsReceived++
			
			if result.Error != nil {
				pcp.logger.Error("Chunk processing failed",
					"task_id", result.TaskID,
					"error", result.Error,
					"retry_count", result.RetryCount)
				pcp.metrics.TotalTasksFailed.Add(1)
				
				// Optionally retry
				if result.RetryCount < pcp.config.MaxRetries {
					// Re-submit task with increased retry count
					// Implementation depends on retry strategy
				}
			} else {
				chunks = append(chunks, result.Chunk)
				pcp.metrics.TotalTasksProcessed.Add(1)
			}
			
		case <-timeout.C:
			pcp.logger.Warn("Timeout waiting for chunk results",
				"expected", expectedResults,
				"received", resultsReceived)
			return chunks
			
		case <-ctx.Done():
			return chunks
		}
	}
	
	return chunks
}

// postProcessChunks performs post-processing on chunks
func (pcp *ParallelChunkProcessor) postProcessChunks(chunks []*DocumentChunk, doc *LoadedDocument, structure *DocumentStructure) []*DocumentChunk {
	// Sort chunks by index to maintain order
	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].ChunkIndex < chunks[j].ChunkIndex
	})
	
	// Apply post-processing from chunking service
	return pcp.chunkingService.postProcessChunks(chunks, doc, structure)
}

// startWorkers starts the worker pool
func (pcp *ParallelChunkProcessor) startWorkers() {
	pcp.workerPool.Start(pcp.taskQueue, pcp.resultQueue)
}

// metricsCollector collects and updates metrics
func (pcp *ParallelChunkProcessor) metricsCollector() {
	defer pcp.wg.Done()
	
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			pcp.updateMetrics()
		case <-pcp.ctx.Done():
			return
		}
	}
}

// updateMetrics updates processing metrics
func (pcp *ParallelChunkProcessor) updateMetrics() {
	pcp.metrics.mu.Lock()
	defer pcp.metrics.mu.Unlock()
	
	// Calculate worker utilization
	for i, worker := range pcp.workerPool.workers {
		utilization := worker.GetUtilization()
		pcp.metrics.WorkerUtilization[i] = utilization
	}
	
	// Calculate average queue depth
	queueDepth := float64(len(pcp.taskQueue))
	pcp.metrics.AverageQueueDepth = (pcp.metrics.AverageQueueDepth + queueDepth) / 2
	
	// Calculate throughput
	processed := pcp.metrics.TotalTasksProcessed.Load()
	elapsed := time.Since(pcp.metrics.LastUpdated).Seconds()
	if elapsed > 0 {
		pcp.metrics.Throughput = float64(processed) / elapsed
	}
	
	// Calculate error rate
	total := pcp.metrics.TotalTasksProcessed.Load() + pcp.metrics.TotalTasksFailed.Load()
	if total > 0 {
		pcp.metrics.ErrorRate = float64(pcp.metrics.TotalTasksFailed.Load()) / float64(total)
	}
	
	pcp.metrics.LastUpdated = time.Now()
}

// resultProcessor processes results from workers
func (pcp *ParallelChunkProcessor) resultProcessor() {
	defer pcp.wg.Done()
	
	for {
		select {
		case result := <-pcp.resultQueue:
			// Process result (e.g., store, forward, etc.)
			if result.Error == nil {
				pcp.logger.Debug("Chunk processed successfully",
					"task_id", result.TaskID,
					"worker_id", result.WorkerID,
					"processing_time", result.ProcessingTime)
			}
		case <-pcp.ctx.Done():
			return
		}
	}
}

// GetMetrics returns current processing metrics
func (pcp *ParallelChunkProcessor) GetMetrics() ParallelProcessingMetrics {
	pcp.metrics.mu.RLock()
	defer pcp.metrics.mu.RUnlock()
	return *pcp.metrics
}

// Shutdown gracefully shuts down the processor
func (pcp *ParallelChunkProcessor) Shutdown(timeout time.Duration) error {
	pcp.logger.Info("Shutting down parallel chunk processor")
	
	// Signal shutdown
	pcp.cancel()
	
	// Stop accepting new tasks
	close(pcp.taskQueue)
	
	// Shutdown worker pool
	if err := pcp.workerPool.Shutdown(timeout); err != nil {
		return fmt.Errorf("failed to shutdown worker pool: %w", err)
	}
	
	// Wait for all goroutines
	done := make(chan struct{})
	go func() {
		pcp.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		pcp.logger.Info("Parallel chunk processor shut down successfully")
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("shutdown timeout exceeded")
	}
}

// WorkerPool implementation

func NewWorkerPool(config *ParallelChunkConfig, chunkingService *ChunkingService, embeddingService *EmbeddingService) *WorkerPool {
	pool := &WorkerPool{
		workers:         make([]*ChunkWorker, config.NumWorkers),
		taskDistributor: NewTaskDistributor(config.LoadBalancingStrategy),
		config:          config,
		logger:          slog.Default().With("component", "worker-pool"),
	}
	
	// Create workers
	for i := 0; i < config.NumWorkers; i++ {
		worker := NewChunkWorker(i, config, chunkingService, embeddingService)
		pool.workers[i] = worker
	}
	
	pool.taskDistributor.workers = pool.workers
	
	return pool
}

func (wp *WorkerPool) Start(taskQueue <-chan *ChunkTask, resultQueue chan<- *ChunkResult) {
	wp.logger.Info("Starting worker pool", "workers", len(wp.workers))
	
	for _, worker := range wp.workers {
		worker.Start(taskQueue, resultQueue)
	}
	
	// Start work stealing if enabled
	if wp.config.EnableWorkStealing {
		go wp.workStealingCoordinator()
	}
}

func (wp *WorkerPool) Shutdown(timeout time.Duration) error {
	wp.logger.Info("Shutting down worker pool")
	
	// Shutdown all workers
	var wg sync.WaitGroup
	for _, worker := range wp.workers {
		wg.Add(1)
		go func(w *ChunkWorker) {
			defer wg.Done()
			w.Shutdown(timeout)
		}(worker)
	}
	
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("worker pool shutdown timeout")
	}
}

func (wp *WorkerPool) workStealingCoordinator() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for range ticker.C {
		// Find idle and busy workers
		var idleWorkers, busyWorkers []*ChunkWorker
		
		for _, worker := range wp.workers {
			if worker.GetQueueDepth() == 0 && !worker.isActive.Load() {
				idleWorkers = append(idleWorkers, worker)
			} else if worker.GetQueueDepth() > wp.config.WorkerQueueSize/2 {
				busyWorkers = append(busyWorkers, worker)
			}
		}
		
		// Steal work from busy to idle workers
		for _, idle := range idleWorkers {
			for _, busy := range busyWorkers {
				if task := busy.StealTask(); task != nil {
					idle.AssignTask(task)
					wp.logger.Debug("Work stolen",
						"from_worker", busy.id,
						"to_worker", idle.id)
					break
				}
			}
		}
	}
}

// ChunkWorker implementation

func NewChunkWorker(id int, config *ParallelChunkConfig, chunkingService *ChunkingService, embeddingService *EmbeddingService) *ChunkWorker {
	ctx, cancel := context.WithCancel(context.Background())
	
	worker := &ChunkWorker{
		id:          id,
		taskQueue:   make(chan *ChunkTask, config.WorkerQueueSize),
		processor:   NewChunkProcessor(config, chunkingService, embeddingService),
		metrics:     &WorkerMetrics{},
		ctx:         ctx,
		cancel:      cancel,
	}
	
	// Set CPU affinity if enabled
	if config.EnableCPUAffinity {
		worker.cpuAffinity = id % runtime.NumCPU()
	}
	
	// Create memory pool if enabled
	if config.EnableMemoryPooling {
		worker.memoryPool = NewMemoryPool(1024*1024, 10) // 1MB buffers, max 10
	}
	
	return worker
}

func (w *ChunkWorker) Start(globalTaskQueue <-chan *ChunkTask, resultQueue chan<- *ChunkResult) {
	w.wg.Add(1)
	go w.processLoop(globalTaskQueue, resultQueue)
}

func (w *ChunkWorker) processLoop(globalTaskQueue <-chan *ChunkTask, resultQueue chan<- *ChunkResult) {
	defer w.wg.Done()
	
	// Set CPU affinity if configured
	if w.cpuAffinity >= 0 {
		// Platform-specific CPU affinity setting would go here
		// This is a simplified version
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}
	
	for {
		select {
		// Check local queue first
		case task := <-w.taskQueue:
			w.processTask(task, resultQueue)
			
		// Then check global queue
		case task := <-globalTaskQueue:
			w.processTask(task, resultQueue)
			
		case <-w.ctx.Done():
			return
		}
	}
}

func (w *ChunkWorker) processTask(task *ChunkTask, resultQueue chan<- *ChunkResult) {
	startTime := time.Now()
	w.isActive.Store(true)
	defer w.isActive.Store(false)
	
	w.metrics.CurrentLoad.Add(1)
	defer w.metrics.CurrentLoad.Add(-1)
	
	// Process the chunk
	chunk, embedding, err := w.processor.ProcessChunk(w.ctx, task)
	
	processingTime := time.Since(startTime)
	
	// Create result
	result := &ChunkResult{
		TaskID:         task.ID,
		Chunk:          chunk,
		Embedding:      embedding,
		ProcessingTime: processingTime,
		WorkerID:       w.id,
		Error:          err,
		RetryCount:     task.RetryCount,
	}
	
	// Update metrics
	if err != nil {
		w.metrics.TasksFailed.Add(1)
		w.metrics.mu.Lock()
		w.metrics.LastError = err
		w.metrics.LastErrorTime = time.Now()
		w.metrics.mu.Unlock()
	} else {
		w.metrics.TasksProcessed.Add(1)
	}
	
	w.metrics.TotalProcessingTime.Add(int64(processingTime))
	w.lastActiveTime = time.Now()
	
	// Send result
	select {
	case resultQueue <- result:
	case <-w.ctx.Done():
		return
	}
}

func (w *ChunkWorker) GetUtilization() float64 {
	// Calculate based on queue depth and activity
	queueUtilization := float64(len(w.taskQueue)) / float64(cap(w.taskQueue))
	
	// Check if actively processing
	activeBonus := 0.0
	if w.isActive.Load() {
		activeBonus = 0.2
	}
	
	// Consider recent activity
	timeSinceActive := time.Since(w.lastActiveTime)
	if timeSinceActive < 1*time.Second {
		activeBonus += 0.3
	} else if timeSinceActive < 5*time.Second {
		activeBonus += 0.1
	}
	
	utilization := queueUtilization + activeBonus
	if utilization > 1.0 {
		utilization = 1.0
	}
	
	return utilization
}

func (w *ChunkWorker) GetQueueDepth() int {
	return len(w.taskQueue)
}

func (w *ChunkWorker) StealTask() *ChunkTask {
	select {
	case task := <-w.taskQueue:
		return task
	default:
		return nil
	}
}

func (w *ChunkWorker) AssignTask(task *ChunkTask) bool {
	select {
	case w.taskQueue <- task:
		return true
	default:
		return false
	}
}

func (w *ChunkWorker) Shutdown(timeout time.Duration) {
	w.cancel()
	
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
	case <-time.After(timeout):
	}
}

// ChunkProcessor implementation

func NewChunkProcessor(config *ParallelChunkConfig, chunkingService *ChunkingService, embeddingService *EmbeddingService) *ChunkProcessor {
	return &ChunkProcessor{
		chunkingService:  chunkingService,
		embeddingService: embeddingService,
		config:           config,
		logger:           slog.Default().With("component", "chunk-processor"),
	}
}

func (cp *ChunkProcessor) ProcessChunk(ctx context.Context, task *ChunkTask) (*DocumentChunk, []float32, error) {
	// Create chunk using chunking service
	chunk := cp.chunkingService.createChunk(
		task.ChunkData,
		task.Document.ID,
		task.StartOffset,
		task.EndOffset-task.StartOffset,
		[]string{}, // hierarchy path
		"", // section title
		0,  // level
	)
	
	// Apply metadata from task
	if sectionTitle, ok := task.Metadata["section_title"].(string); ok {
		chunk.SectionTitle = sectionTitle
	}
	if sectionLevel, ok := task.Metadata["section_level"].(int); ok {
		chunk.HierarchyLevel = sectionLevel
	}
	
	// Analyze chunk content
	cp.chunkingService.analyzeChunkTechnicalContent(chunk)
	
	// Generate embedding if embedding service is available
	var embedding []float32
	if cp.embeddingService != nil {
		request := &EmbeddingRequest{
			Texts:     []string{chunk.CleanContent},
			ChunkIDs:  []string{chunk.ID},
			UseCache:  true,
			RequestID: fmt.Sprintf("chunk_%s", chunk.ID),
		}
		
		response, err := cp.embeddingService.GenerateEmbeddings(ctx, request)
		if err != nil {
			return chunk, nil, fmt.Errorf("failed to generate embedding: %w", err)
		}
		
		if len(response.Embeddings) > 0 {
			embedding = response.Embeddings[0]
		}
	}
	
	return chunk, embedding, nil
}

// TaskDistributor implementation

func NewTaskDistributor(strategy string) *TaskDistributor {
	return &TaskDistributor{
		strategy: strategy,
		hashSeed: uint32(time.Now().UnixNano()),
	}
}

func (td *TaskDistributor) SelectWorker(task *ChunkTask) *ChunkWorker {
	td.mu.RLock()
	defer td.mu.RUnlock()
	
	if len(td.workers) == 0 {
		return nil
	}
	
	switch td.strategy {
	case "round_robin":
		index := td.roundRobinIndex.Add(1) % int32(len(td.workers))
		return td.workers[index]
		
	case "least_loaded":
		var selected *ChunkWorker
		minLoad := int32(^uint32(0) >> 1) // Max int32
		
		for _, worker := range td.workers {
			load := worker.metrics.CurrentLoad.Load()
			if load < minLoad {
				minLoad = load
				selected = worker
			}
		}
		return selected
		
	case "hash_based":
		// Hash based on task ID for consistent assignment
		hash := fnvHash(task.ID, td.hashSeed)
		index := hash % uint32(len(td.workers))
		return td.workers[index]
		
	default:
		// Default to round robin
		index := td.roundRobinIndex.Add(1) % int32(len(td.workers))
		return td.workers[index]
	}
}

// MemoryPool implementation

func NewMemoryPool(bufferSize, maxBuffers int) *MemoryPool {
	return &MemoryPool{
		buffers:    make(chan []byte, maxBuffers),
		bufferSize: bufferSize,
		maxBuffers: maxBuffers,
	}
}

func (mp *MemoryPool) Get() []byte {
	select {
	case buf := <-mp.buffers:
		return buf
	default:
		// Allocate new buffer if under limit
		if mp.allocatedCount.Load() < int32(mp.maxBuffers) {
			mp.allocatedCount.Add(1)
			return make([]byte, mp.bufferSize)
		}
		// Block waiting for available buffer
		return <-mp.buffers
	}
}

func (mp *MemoryPool) Put(buf []byte) {
	if cap(buf) >= mp.bufferSize {
		// Clear buffer before returning to pool
		buf = buf[:0]
		select {
		case mp.buffers <- buf:
		default:
			// Pool is full, let GC handle it
		}
	}
}

// Helper functions

func fnvHash(s string, seed uint32) uint32 {
	h := seed
	for i := 0; i < len(s); i++ {
		h = h*16777619 ^ uint32(s[i])
	}
	return h
}

// Import sort package
import "sort"

// Import strings package is already included