//go:build !disable_rag
// +build !disable_rag

package llm

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// BatchProcessorImpl handles batch processing of LLM requests for improved efficiency.
type BatchProcessorImpl struct {
	config BatchConfig
	logger *slog.Logger
	tracer trace.Tracer

	// Batch management.
	batches      map[string]*Batch
	batchesMutex sync.RWMutex

	// Request queuing.
	requestQueue  chan *BatchRequest
	priorityQueue *PriorityQueue

	// Worker management.
	workers       []*BatchWorker
	workerWg      sync.WaitGroup
	shutdown      chan bool
	isShutdown    bool
	shutdownMutex sync.RWMutex

	// Metrics.
	processedBatches   int64
	totalRequests      int64
	averageBatchSize   float64
	averageProcessTime time.Duration
	metricsMutex       sync.RWMutex
}

// BatchRequest represents a request to be processed in a batch.
type BatchRequest struct {
	ID         string
	Intent     string
	IntentType string
	ModelName  string
	Priority   Priority
	Context    context.Context
	ResultChan chan *BatchResult
	Metadata   map[string]interface{}
	SubmitTime time.Time
	Timeout    time.Duration
}

// BatchResult represents the result of a batch request.
type BatchResult struct {
	RequestID   string
	Response    string
	Error       error
	ProcessTime time.Duration
	BatchID     string
	BatchSize   int
	QueueTime   time.Duration
}

// Batch represents a collection of requests to be processed together.
type Batch struct {
	ID          string
	Requests    []*BatchRequest
	ModelName   string
	CreatedAt   time.Time
	ProcessedAt time.Time
	Status      BatchStatus
	mutex       sync.RWMutex
}

// BatchStatus represents the status of a batch.
type BatchStatus int

const (
	// BatchStatusPending holds batchstatuspending value.
	BatchStatusPending BatchStatus = iota
	// BatchStatusProcessing holds batchstatusprocessing value.
	BatchStatusProcessing
	// BatchStatusCompleted holds batchstatuscompleted value.
	BatchStatusCompleted
	// BatchStatusFailed holds batchstatusfailed value.
	BatchStatusFailed
)

// Priority levels for request prioritization.
type Priority int

const (
	// PriorityLow holds prioritylow value.
	PriorityLow Priority = iota
	// PriorityNormal holds prioritynormal value.
	PriorityNormal
	// PriorityHigh holds priorityhigh value.
	PriorityHigh
	// PriorityUrgent holds priorityurgent value.
	PriorityUrgent // Same as critical
	// PriorityCritical holds prioritycritical value.
	PriorityCritical = PriorityUrgent // Alias for backward compatibility
)

// BatchWorker processes batches.
type BatchWorker struct {
	id        int
	processor *BatchProcessorImpl
	client    *Client
	logger    *slog.Logger
	stopChan  chan bool
}

// PriorityQueue implements a priority queue for batch requests.
type PriorityQueue struct {
	items []*BatchRequest
	mutex sync.RWMutex
}

// NewBatchProcessor creates a new batch processor.
func NewBatchProcessor(config BatchConfig) *BatchProcessorImpl {
	bp := &BatchProcessorImpl{
		config:        config,
		logger:        slog.Default().With("component", "batch-processor"),
		tracer:        otel.Tracer("nephoran-intent-operator/batch-processor"),
		batches:       make(map[string]*Batch),
		requestQueue:  make(chan *BatchRequest, 1000),
		priorityQueue: NewPriorityQueue(),
		workers:       make([]*BatchWorker, config.ConcurrentBatches),
		shutdown:      make(chan bool, 1),
	}

	// Start workers.
	bp.startWorkers()

	// Start batch formation routine.
	go bp.batchFormationRoutine()

	return bp
}

// NewPriorityQueue creates a new priority queue.
func NewPriorityQueue() *PriorityQueue {
	return &PriorityQueue{
		items: make([]*BatchRequest, 0),
	}
}

// Push adds a request to the priority queue.
func (pq *PriorityQueue) Push(request *BatchRequest) {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	pq.items = append(pq.items, request)

	// Sort by priority (higher priority first).
	sort.Slice(pq.items, func(i, j int) bool {
		if pq.items[i].Priority != pq.items[j].Priority {
			return pq.items[i].Priority > pq.items[j].Priority
		}
		// If same priority, sort by submit time (FIFO).
		return pq.items[i].SubmitTime.Before(pq.items[j].SubmitTime)
	})
}

// Pop removes and returns the highest priority request.
func (pq *PriorityQueue) Pop() *BatchRequest {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	if len(pq.items) == 0 {
		return nil
	}

	item := pq.items[0]
	pq.items = pq.items[1:]
	return item
}

// Len returns the number of items in the queue.
func (pq *PriorityQueue) Len() int {
	pq.mutex.RLock()
	defer pq.mutex.RUnlock()
	return len(pq.items)
}

// ProcessRequest processes a single request through batching.
func (bp *BatchProcessorImpl) ProcessRequest(ctx context.Context, intent, intentType, modelName string, priority Priority) (*BatchResult, error) {
	bp.shutdownMutex.RLock()
	if bp.isShutdown {
		bp.shutdownMutex.RUnlock()
		return nil, fmt.Errorf("batch processor is shutdown")
	}
	bp.shutdownMutex.RUnlock()

	// Create batch request.
	request := &BatchRequest{
		ID:         generateRequestID(),
		Intent:     intent,
		IntentType: intentType,
		ModelName:  modelName,
		Priority:   priority,
		Context:    ctx,
		ResultChan: make(chan *BatchResult, 1),
		Metadata:   make(map[string]interface{}),
		SubmitTime: time.Now(),
		Timeout:    30 * time.Second, // Default timeout
	}

	// Add to queue.
	if bp.config.EnablePrioritization {
		bp.priorityQueue.Push(request)
	} else {
		select {
		case bp.requestQueue <- request:
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			return nil, fmt.Errorf("request queue is full")
		}
	}

	// Wait for result.
	select {
	case result := <-request.ResultChan:
		return result, result.Error
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(request.Timeout):
		return nil, fmt.Errorf("batch processing timeout")
	}
}

// startWorkers starts the batch processing workers.
func (bp *BatchProcessorImpl) startWorkers() {
	for i := 0; i < bp.config.ConcurrentBatches; i++ {
		worker := &BatchWorker{
			id:        i,
			processor: bp,
			logger:    bp.logger.With("worker_id", i),
			stopChan:  make(chan bool, 1),
		}
		bp.workers[i] = worker

		bp.workerWg.Add(1)
		go worker.run()
	}
}

// batchFormationRoutine forms batches from incoming requests.
func (bp *BatchProcessorImpl) batchFormationRoutine() {
	ticker := time.NewTicker(bp.config.BatchTimeout)
	defer ticker.Stop()

	pendingRequests := make(map[string][]*BatchRequest) // Group by model

	for {
		select {
		case <-bp.shutdown:
			return

		case request := <-bp.requestQueue:
			// Group requests by model.
			if pendingRequests[request.ModelName] == nil {
				pendingRequests[request.ModelName] = make([]*BatchRequest, 0)
			}
			pendingRequests[request.ModelName] = append(pendingRequests[request.ModelName], request)

			// Check if we should form a batch.
			if len(pendingRequests[request.ModelName]) >= bp.config.MaxBatchSize {
				bp.formBatch(request.ModelName, pendingRequests[request.ModelName])
				delete(pendingRequests, request.ModelName)
			}

		case <-ticker.C:
			// Process priority queue if enabled.
			if bp.config.EnablePrioritization {
				bp.processPriorityQueue(pendingRequests)
			}

			// Form batches from pending requests.
			for modelName, requests := range pendingRequests {
				if len(requests) > 0 {
					bp.formBatch(modelName, requests)
					delete(pendingRequests, modelName)
				}
			}
		}
	}
}

// processPriorityQueue processes requests from the priority queue.
func (bp *BatchProcessorImpl) processPriorityQueue(pendingRequests map[string][]*BatchRequest) {
	// Move high priority requests from priority queue to processing.
	for bp.priorityQueue.Len() > 0 {
		request := bp.priorityQueue.Pop()
		if request == nil {
			break
		}

		// Group by model.
		if pendingRequests[request.ModelName] == nil {
			pendingRequests[request.ModelName] = make([]*BatchRequest, 0)
		}
		pendingRequests[request.ModelName] = append(pendingRequests[request.ModelName], request)

		// Form batch if we have enough requests.
		if len(pendingRequests[request.ModelName]) >= bp.config.MaxBatchSize {
			bp.formBatch(request.ModelName, pendingRequests[request.ModelName])
			delete(pendingRequests, request.ModelName)
			break
		}
	}
}

// formBatch creates a batch from pending requests.
func (bp *BatchProcessorImpl) formBatch(modelName string, requests []*BatchRequest) {
	if len(requests) == 0 {
		return
	}

	batch := &Batch{
		ID:        generateBatchID(),
		Requests:  requests,
		ModelName: modelName,
		CreatedAt: time.Now(),
		Status:    BatchStatusPending,
	}

	bp.batchesMutex.Lock()
	bp.batches[batch.ID] = batch
	bp.batchesMutex.Unlock()

	// Send batch to available worker.
	go bp.assignBatchToWorker(batch)

	bp.logger.Debug("Batch formed",
		"batch_id", batch.ID,
		"model_name", modelName,
		"batch_size", len(requests),
	)
}

// assignBatchToWorker assigns a batch to an available worker.
func (bp *BatchProcessorImpl) assignBatchToWorker(batch *Batch) {
	// Simple round-robin assignment.
	for _, worker := range bp.workers {
		select {
		case <-worker.stopChan:
			continue
		default:
			worker.processBatch(batch)
			return
		}
	}

	// If no worker is immediately available, wait and retry.
	time.Sleep(10 * time.Millisecond)
	bp.assignBatchToWorker(batch)
}

// run starts the worker processing loop.
func (worker *BatchWorker) run() {
	defer worker.processor.workerWg.Done()

	worker.logger.Info("Batch worker started")

	for {
		select {
		case <-worker.stopChan:
			worker.logger.Info("Batch worker stopped")
			return
		default:
			// Worker will process batches as they are assigned.
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// processBatch processes a batch of requests.
func (worker *BatchWorker) processBatch(batch *Batch) {
	ctx, span := worker.processor.tracer.Start(context.Background(), "batch_processor.process_batch")
	defer span.End()

	span.SetAttributes(
		attribute.String("batch.id", batch.ID),
		attribute.String("batch.model_name", batch.ModelName),
		attribute.Int("batch.size", len(batch.Requests)),
	)

	worker.logger.Info("Processing batch",
		"batch_id", batch.ID,
		"batch_size", len(batch.Requests),
		"model_name", batch.ModelName,
	)

	batch.mutex.Lock()
	batch.Status = BatchStatusProcessing
	batch.ProcessedAt = time.Now()
	batch.mutex.Unlock()

	start := time.Now()

	// Process requests in parallel within the batch.
	var wg sync.WaitGroup
	results := make([]*BatchResult, len(batch.Requests))

	for i, request := range batch.Requests {
		wg.Add(1)
		go func(idx int, req *BatchRequest) {
			defer wg.Done()

			result := worker.processRequest(ctx, req, batch)
			results[idx] = result

			// Send result back to requester.
			select {
			case req.ResultChan <- result:
			case <-time.After(time.Second):
				worker.logger.Warn("Failed to send result to requester", "request_id", req.ID)
			}
		}(i, request)
	}

	wg.Wait()

	processingTime := time.Since(start)

	// Update batch status.
	batch.mutex.Lock()
	batch.Status = BatchStatusCompleted
	batch.mutex.Unlock()

	// Update metrics.
	worker.processor.updateMetrics(len(batch.Requests), processingTime)

	span.SetAttributes(
		attribute.Int64("batch.processing_time_ms", processingTime.Milliseconds()),
		attribute.Bool("batch.success", true),
	)

	worker.logger.Info("Batch processed",
		"batch_id", batch.ID,
		"batch_size", len(batch.Requests),
		"processing_time", processingTime,
	)
}

// processRequest processes a single request within a batch.
func (worker *BatchWorker) processRequest(ctx context.Context, request *BatchRequest, batch *Batch) *BatchResult {
	start := time.Now()
	queueTime := start.Sub(request.SubmitTime)

	// Use a simple client for now - in production, this would use the configured LLM client.
	response := fmt.Sprintf(`{
		"type": "%s",
		"name": "processed-%s",
		"namespace": "default",
		"spec": {
			"intent": "%s",
			"processed_at": "%s",
			"batch_id": "%s"
		}
	}`, request.IntentType, request.ID, request.Intent, time.Now().Format(time.RFC3339), batch.ID)

	processingTime := time.Since(start)

	return &BatchResult{
		RequestID:   request.ID,
		Response:    response,
		Error:       nil,
		ProcessTime: processingTime,
		BatchID:     batch.ID,
		BatchSize:   len(batch.Requests),
		QueueTime:   queueTime,
	}
}

// updateMetrics updates batch processing metrics.
func (bp *BatchProcessorImpl) updateMetrics(batchSize int, processingTime time.Duration) {
	bp.metricsMutex.Lock()
	defer bp.metricsMutex.Unlock()

	bp.processedBatches++
	bp.totalRequests += int64(batchSize)

	// Update average batch size.
	bp.averageBatchSize = (bp.averageBatchSize*float64(bp.processedBatches-1) + float64(batchSize)) / float64(bp.processedBatches)

	// Update average processing time.
	totalTime := bp.averageProcessTime*time.Duration(bp.processedBatches-1) + processingTime
	bp.averageProcessTime = totalTime / time.Duration(bp.processedBatches)
}

// GetStats returns current batch processor statistics.
func (bp *BatchProcessorImpl) GetStats() BatchProcessorStats {
	bp.metricsMutex.RLock()
	defer bp.metricsMutex.RUnlock()

	bp.batchesMutex.RLock()
	activeBatches := len(bp.batches)
	bp.batchesMutex.RUnlock()

	return BatchProcessorStats{
		ProcessedBatches:    bp.processedBatches,
		TotalRequests:       bp.totalRequests,
		ActiveBatches:       int64(activeBatches),
		AverageBatchSize:    bp.averageBatchSize,
		AverageProcessTime:  bp.averageProcessTime,
		QueueLength:         int64(len(bp.requestQueue)),
		PriorityQueueLength: int64(bp.priorityQueue.Len()),
		WorkerCount:         int64(len(bp.workers)),
	}
}

// BatchProcessorStats holds batch processor statistics.
type BatchProcessorStats struct {
	ProcessedBatches    int64         `json:"processed_batches"`
	TotalRequests       int64         `json:"total_requests"`
	ActiveBatches       int64         `json:"active_batches"`
	AverageBatchSize    float64       `json:"average_batch_size"`
	AverageProcessTime  time.Duration `json:"average_process_time"`
	QueueLength         int64         `json:"queue_length"`
	PriorityQueueLength int64         `json:"priority_queue_length"`
	WorkerCount         int64         `json:"worker_count"`
}

// GetBatch returns information about a specific batch.
func (bp *BatchProcessorImpl) GetBatch(batchID string) (*Batch, bool) {
	bp.batchesMutex.RLock()
	defer bp.batchesMutex.RUnlock()

	batch, exists := bp.batches[batchID]
	return batch, exists
}

// Close gracefully shuts down the batch processor.
func (bp *BatchProcessorImpl) Close() error {
	bp.shutdownMutex.Lock()
	bp.isShutdown = true
	bp.shutdownMutex.Unlock()

	bp.logger.Info("Shutting down batch processor")

	// Signal shutdown.
	close(bp.shutdown)

	// Stop all workers.
	for _, worker := range bp.workers {
		close(worker.stopChan)
	}

	// Wait for workers to finish.
	bp.workerWg.Wait()

	bp.logger.Info("Batch processor shutdown complete")
	return nil
}

// Helper functions.
func generateBatchID() string {
	return fmt.Sprintf("batch_%d", time.Now().UnixNano())
}

func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}
