package optimized

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// APICallType represents different types of API operations
type APICallType string

const (
	GetCall    APICallType = "get"
	ListCall   APICallType = "list"
	CreateCall APICallType = "create"
	UpdateCall APICallType = "update"
	DeleteCall APICallType = "delete"
	PatchCall  APICallType = "patch"
)

// APICall represents a batched API call
type APICall struct {
	Type          APICallType
	Object        client.Object
	Key           types.NamespacedName
	Options       []client.GetOption
	ListOptions   []client.ListOption
	PatchOptions  []client.PatchOption
	CreateOptions []client.CreateOption
	UpdateOptions []client.UpdateOption
	DeleteOptions []client.DeleteOption
	Patch         client.Patch
	ResultChannel chan APICallResult
	Timestamp     time.Time
	Priority      UpdatePriority
}

// APICallResult represents the result of an API call
type APICallResult struct {
	Object client.Object
	Error  error
}

// APIBatchConfig configures the API call batcher
type APIBatchConfig struct {
	MaxBatchSize    int           // Maximum number of calls per batch
	BatchTimeout    time.Duration // Maximum time to wait for a batch
	FlushInterval   time.Duration // Periodic flush interval
	MaxQueueSize    int           // Maximum queue size
	EnablePriority  bool          // Whether to prioritize calls
	ParallelBatches int           // Number of parallel batch processors
}

// DefaultAPIBatchConfig provides sensible defaults
var DefaultAPIBatchConfig = APIBatchConfig{
	MaxBatchSize:    5, // Smaller batch size for API calls
	BatchTimeout:    500 * time.Millisecond,
	FlushInterval:   1 * time.Second,
	MaxQueueSize:    500,
	EnablePriority:  true,
	ParallelBatches: 3,
}

// APICallBatcher batches Kubernetes API calls to reduce server load
type APICallBatcher struct {
	client     client.Client
	config     APIBatchConfig
	mu         sync.RWMutex
	queue      []*APICall
	flushTimer *time.Timer
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup

	// Metrics
	batchesProcessed int64
	callsProcessed   int64
	callsDropped     int64
	callsFailed      int64
}

// NewAPICallBatcher creates a new API call batcher
func NewAPICallBatcher(client client.Client, config APIBatchConfig) *APICallBatcher {
	ctx, cancel := context.WithCancel(context.Background())

	batcher := &APICallBatcher{
		client: client,
		config: config,
		queue:  make([]*APICall, 0, config.MaxBatchSize),
		ctx:    ctx,
		cancel: cancel,
	}

	// Start multiple batch processors
	for i := 0; i < config.ParallelBatches; i++ {
		batcher.wg.Add(1)
		go batcher.processCalls()
	}

	return batcher
}

// Get queues a Get API call
func (ab *APICallBatcher) Get(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
	resultChan := make(chan APICallResult, 1)

	call := &APICall{
		Type:          GetCall,
		Key:           key,
		Object:        obj,
		Options:       opts,
		ResultChannel: resultChan,
		Timestamp:     time.Now(),
		Priority:      MediumPriority,
	}

	if err := ab.queueCall(call); err != nil {
		return err
	}

	// Wait for result
	select {
	case result := <-resultChan:
		if result.Error != nil {
			return result.Error
		}
		// Copy result back to original object using reflection
		if result.Object != nil {
			objVal := reflect.ValueOf(obj)
			resultVal := reflect.ValueOf(result.Object)
			
			if objVal.Kind() == reflect.Ptr && resultVal.Kind() == reflect.Ptr {
				if objVal.Elem().Type() == resultVal.Elem().Type() {
					objVal.Elem().Set(resultVal.Elem())
				}
			}
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(10 * time.Second): // Timeout
		return fmt.Errorf("API call timeout")
	}
}

// Update queues an Update API call
func (ab *APICallBatcher) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	resultChan := make(chan APICallResult, 1)

	call := &APICall{
		Type:          UpdateCall,
		Object:        obj.DeepCopyObject().(client.Object),
		UpdateOptions: opts,
		ResultChannel: resultChan,
		Timestamp:     time.Now(),
		Priority:      HighPriority, // Updates are usually high priority
	}

	if err := ab.queueCall(call); err != nil {
		return err
	}

	// Wait for result
	select {
	case result := <-resultChan:
		return result.Error
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(10 * time.Second):
		return fmt.Errorf("API call timeout")
	}
}

// Create queues a Create API call
func (ab *APICallBatcher) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	resultChan := make(chan APICallResult, 1)

	call := &APICall{
		Type:          CreateCall,
		Object:        obj.DeepCopyObject().(client.Object),
		CreateOptions: opts,
		ResultChannel: resultChan,
		Timestamp:     time.Now(),
		Priority:      HighPriority,
	}

	if err := ab.queueCall(call); err != nil {
		return err
	}

	// Wait for result
	select {
	case result := <-resultChan:
		return result.Error
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(10 * time.Second):
		return fmt.Errorf("API call timeout")
	}
}

// Delete queues a Delete API call
func (ab *APICallBatcher) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	resultChan := make(chan APICallResult, 1)

	call := &APICall{
		Type:          DeleteCall,
		Object:        obj.DeepCopyObject().(client.Object),
		DeleteOptions: opts,
		ResultChannel: resultChan,
		Timestamp:     time.Now(),
		Priority:      HighPriority,
	}

	if err := ab.queueCall(call); err != nil {
		return err
	}

	// Wait for result
	select {
	case result := <-resultChan:
		return result.Error
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(10 * time.Second):
		return fmt.Errorf("API call timeout")
	}
}

// Patch queues a Patch API call
func (ab *APICallBatcher) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	resultChan := make(chan APICallResult, 1)

	call := &APICall{
		Type:          PatchCall,
		Object:        obj.DeepCopyObject().(client.Object),
		Patch:         patch,
		PatchOptions:  opts,
		ResultChannel: resultChan,
		Timestamp:     time.Now(),
		Priority:      HighPriority,
	}

	if err := ab.queueCall(call); err != nil {
		return err
	}

	// Wait for result
	select {
	case result := <-resultChan:
		return result.Error
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(10 * time.Second):
		return fmt.Errorf("API call timeout")
	}
}

// queueCall adds an API call to the queue
func (ab *APICallBatcher) queueCall(call *APICall) error {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	// Check queue size limit
	if len(ab.queue) >= ab.config.MaxQueueSize {
		ab.callsDropped++
		close(call.ResultChannel)
		return fmt.Errorf("API call queue is full")
	}

	ab.queue = append(ab.queue, call)

	// Trigger immediate flush for high priority calls
	if call.Priority >= HighPriority {
		ab.triggerFlush()
	} else if len(ab.queue) >= ab.config.MaxBatchSize {
		ab.triggerFlush()
	} else if ab.flushTimer == nil {
		ab.flushTimer = time.AfterFunc(ab.config.BatchTimeout, ab.triggerFlush)
	}

	return nil
}

// Flush immediately processes all queued calls
func (ab *APICallBatcher) Flush() error {
	ab.mu.Lock()
	if ab.flushTimer != nil {
		ab.flushTimer.Stop()
		ab.flushTimer = nil
	}
	ab.mu.Unlock()

	return ab.processBatch()
}

// Stop gracefully shuts down the API call batcher
func (ab *APICallBatcher) Stop() error {
	ab.cancel()

	// Process any remaining calls
	if err := ab.Flush(); err != nil {
		log.Log.Error(err, "Failed to flush final API batch during shutdown")
	}

	// Wait for background processors to finish
	ab.wg.Wait()

	return nil
}

// processCalls runs in background goroutines to process API calls
func (ab *APICallBatcher) processCalls() {
	defer ab.wg.Done()

	ticker := time.NewTicker(ab.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ab.ctx.Done():
			return
		case <-ticker.C:
			if err := ab.processBatch(); err != nil {
				log.Log.Error(err, "Failed to process API batch during periodic flush")
			}
		}
	}
}

// triggerFlush triggers an immediate flush
func (ab *APICallBatcher) triggerFlush() {
	go func() {
		if err := ab.processBatch(); err != nil {
			log.Log.Error(err, "Failed to process API batch during triggered flush")
		}
	}()
}

// processBatch processes a batch of API calls
func (ab *APICallBatcher) processBatch() error {
	ab.mu.Lock()

	if len(ab.queue) == 0 {
		ab.mu.Unlock()
		return nil
	}

	// Extract batch for processing
	batchSize := len(ab.queue)
	if batchSize > ab.config.MaxBatchSize {
		batchSize = ab.config.MaxBatchSize
	}

	batch := make([]*APICall, batchSize)
	copy(batch, ab.queue[:batchSize])

	// Remove processed items from queue
	remaining := ab.queue[batchSize:]
	ab.queue = make([]*APICall, 0, cap(ab.queue))
	ab.queue = append(ab.queue, remaining...)

	// Reset flush timer
	if ab.flushTimer != nil {
		ab.flushTimer.Stop()
		ab.flushTimer = nil
	}

	ab.mu.Unlock()

	// Sort by priority if enabled
	if ab.config.EnablePriority {
		ab.sortByPriority(batch)
	}

	// Process batch outside of lock
	successCount := 0
	for _, call := range batch {
		result := ab.executeCall(call)

		// Send result back to caller
		select {
		case call.ResultChannel <- result:
		default:
			// Channel closed, caller timed out
		}
		close(call.ResultChannel)

		if result.Error == nil {
			successCount++
		} else {
			ab.callsFailed++
		}
	}

	// Update metrics
	ab.batchesProcessed++
	ab.callsProcessed += int64(successCount)

	return nil
}

// executeCall executes a single API call
func (ab *APICallBatcher) executeCall(call *APICall) APICallResult {
	var err error
	var obj client.Object

	switch call.Type {
	case GetCall:
		obj = call.Object.DeepCopyObject().(client.Object)
		err = ab.client.Get(ab.ctx, call.Key, obj, call.Options...)

	case UpdateCall:
		err = ab.client.Update(ab.ctx, call.Object, call.UpdateOptions...)

	case CreateCall:
		err = ab.client.Create(ab.ctx, call.Object, call.CreateOptions...)

	case DeleteCall:
		err = ab.client.Delete(ab.ctx, call.Object, call.DeleteOptions...)

	case PatchCall:
		err = ab.client.Patch(ab.ctx, call.Object, call.Patch, call.PatchOptions...)

	default:
		err = fmt.Errorf("unsupported API call type: %s", call.Type)
	}

	return APICallResult{
		Object: obj,
		Error:  err,
	}
}

// sortByPriority sorts calls by priority (highest first)
func (ab *APICallBatcher) sortByPriority(calls []*APICall) {
	for i := 0; i < len(calls)-1; i++ {
		for j := i + 1; j < len(calls); j++ {
			if calls[j].Priority > calls[i].Priority {
				calls[i], calls[j] = calls[j], calls[i]
			}
		}
	}
}

// GetStats returns statistics about the API batcher
func (ab *APICallBatcher) GetStats() APIBatcherStats {
	ab.mu.RLock()
	defer ab.mu.RUnlock()

	return APIBatcherStats{
		QueueSize:        len(ab.queue),
		BatchesProcessed: ab.batchesProcessed,
		CallsProcessed:   ab.callsProcessed,
		CallsDropped:     ab.callsDropped,
		CallsFailed:      ab.callsFailed,
	}
}

// APIBatcherStats contains statistics about the API batcher
type APIBatcherStats struct {
	QueueSize        int   `json:"queue_size"`
	BatchesProcessed int64 `json:"batches_processed"`
	CallsProcessed   int64 `json:"calls_processed"`
	CallsDropped     int64 `json:"calls_dropped"`
	CallsFailed      int64 `json:"calls_failed"`
}
