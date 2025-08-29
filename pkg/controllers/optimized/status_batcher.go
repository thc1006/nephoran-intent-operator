package optimized

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// StatusUpdate represents a pending status update.
type StatusUpdate struct {
	NamespacedName types.NamespacedName
	UpdateFunc     func(obj client.Object) error
	Priority       UpdatePriority
	Timestamp      time.Time
	RetryCount     int
}

// UpdatePriority defines the priority of status updates.
type UpdatePriority int

const (
	// LowPriority holds lowpriority value.
	LowPriority UpdatePriority = iota
	// MediumPriority holds mediumpriority value.
	MediumPriority
	// HighPriority holds highpriority value.
	HighPriority
	// CriticalPriority holds criticalpriority value.
	CriticalPriority
)

// BatchConfig configures the status batcher behavior.
type BatchConfig struct {
	MaxBatchSize   int           // Maximum number of updates per batch
	BatchTimeout   time.Duration // Maximum time to wait for a batch
	FlushInterval  time.Duration // Periodic flush interval
	MaxRetries     int           // Maximum retries per update
	RetryDelay     time.Duration // Base delay between retries
	EnablePriority bool          // Whether to prioritize updates
	MaxQueueSize   int           // Maximum queue size before dropping updates
}

// DefaultBatchConfig provides sensible defaults.
var DefaultBatchConfig = BatchConfig{
	MaxBatchSize:   10,
	BatchTimeout:   2 * time.Second,
	FlushInterval:  5 * time.Second,
	MaxRetries:     3,
	RetryDelay:     1 * time.Second,
	EnablePriority: true,
	MaxQueueSize:   1000,
}

// StatusBatcher batches status updates to reduce API server load.
type StatusBatcher struct {
	client     client.Client
	config     BatchConfig
	mu         sync.RWMutex
	updates    map[types.NamespacedName]*StatusUpdate
	queue      []*StatusUpdate
	flushTimer *time.Timer
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup

	// Metrics.
	batchesProcessed int64
	updatesProcessed int64
	updatesDropped   int64
	updatesFailed    int64
	averageBatchSize float64
}

// NewStatusBatcher creates a new status batcher.
func NewStatusBatcher(client client.Client, config BatchConfig) *StatusBatcher {
	ctx, cancel := context.WithCancel(context.Background())

	batcher := &StatusBatcher{
		client:  client,
		config:  config,
		updates: make(map[types.NamespacedName]*StatusUpdate),
		queue:   make([]*StatusUpdate, 0, config.MaxBatchSize),
		ctx:     ctx,
		cancel:  cancel,
	}

	// Start background processor.
	batcher.wg.Add(1)
	go batcher.processUpdates()

	return batcher
}

// QueueUpdate queues a status update for batching.
func (sb *StatusBatcher) QueueUpdate(namespacedName types.NamespacedName, updateFunc func(obj client.Object) error, priority UpdatePriority) error {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	// Check queue size limit.
	if len(sb.updates)+len(sb.queue) >= sb.config.MaxQueueSize {
		sb.updatesDropped++
		return fmt.Errorf("status update queue is full, dropping update for %s", namespacedName)
	}

	update := &StatusUpdate{
		NamespacedName: namespacedName,
		UpdateFunc:     updateFunc,
		Priority:       priority,
		Timestamp:      time.Now(),
		RetryCount:     0,
	}

	// Replace existing update for the same resource (latest wins).
	if existing, exists := sb.updates[namespacedName]; exists {
		// Update priority to higher value if needed.
		if priority > existing.Priority {
			existing.Priority = priority
		}
		existing.UpdateFunc = updateFunc
		existing.Timestamp = time.Now()
	} else {
		sb.updates[namespacedName] = update
		sb.queue = append(sb.queue, update)
	}

	// Trigger immediate flush for critical updates.
	if priority == CriticalPriority {
		sb.triggerFlush()
	} else if len(sb.queue) >= sb.config.MaxBatchSize {
		sb.triggerFlush()
	} else if sb.flushTimer == nil {
		sb.flushTimer = time.AfterFunc(sb.config.BatchTimeout, sb.triggerFlush)
	}

	return nil
}

// QueueNetworkIntentUpdate queues a NetworkIntent status update.
func (sb *StatusBatcher) QueueNetworkIntentUpdate(namespacedName types.NamespacedName, conditionUpdates []metav1.Condition, phase string, priority UpdatePriority) error {
	updateFunc := func(obj client.Object) error {
		networkIntent, ok := obj.(*nephoranv1.NetworkIntent)
		if !ok {
			return fmt.Errorf("object is not a NetworkIntent")
		}

		// Apply condition updates.
		for _, condition := range conditionUpdates {
			updateCondition(&networkIntent.Status.Conditions, condition)
		}

		// Update phase if provided.
		if phase != "" {
			networkIntent.Status.Phase = nephoranv1.NetworkIntentPhase(phase)
		}

		return nil
	}

	return sb.QueueUpdate(namespacedName, updateFunc, priority)
}

// QueueE2NodeSetUpdate queues an E2NodeSet status update.
func (sb *StatusBatcher) QueueE2NodeSetUpdate(namespacedName types.NamespacedName, readyReplicas, totalReplicas int32, conditions []metav1.Condition, priority UpdatePriority) error {
	updateFunc := func(obj client.Object) error {
		e2nodeSet, ok := obj.(*nephoranv1.E2NodeSet)
		if !ok {
			return fmt.Errorf("object is not an E2NodeSet")
		}

		// Update replica status.
		e2nodeSet.Status.ReadyReplicas = readyReplicas
		e2nodeSet.Status.CurrentReplicas = totalReplicas

		// Apply condition updates.
		for _, condition := range conditions {
			updateE2NodeSetCondition(&e2nodeSet.Status.Conditions, condition)
		}

		return nil
	}

	return sb.QueueUpdate(namespacedName, updateFunc, priority)
}

// Flush immediately processes all queued updates.
func (sb *StatusBatcher) Flush() error {
	sb.mu.Lock()
	if sb.flushTimer != nil {
		sb.flushTimer.Stop()
		sb.flushTimer = nil
	}
	sb.mu.Unlock()

	return sb.processBatch()
}

// Stop gracefully shuts down the status batcher.
func (sb *StatusBatcher) Stop() error {
	// Cancel background processing.
	sb.cancel()

	// Process any remaining updates.
	if err := sb.Flush(); err != nil {
		log.Log.Error(err, "Failed to flush final batch during shutdown")
	}

	// Wait for background goroutine to finish.
	sb.wg.Wait()

	return nil
}

// GetStats returns statistics about the batcher.
func (sb *StatusBatcher) GetStats() StatusBatcherStats {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	return StatusBatcherStats{
		QueueSize:        len(sb.queue),
		BatchesProcessed: sb.batchesProcessed,
		UpdatesProcessed: sb.updatesProcessed,
		UpdatesDropped:   sb.updatesDropped,
		UpdatesFailed:    sb.updatesFailed,
		AverageBatchSize: sb.averageBatchSize,
	}
}

// StatusBatcherStats contains statistics about the batcher.
type StatusBatcherStats struct {
	QueueSize        int     `json:"queue_size"`
	BatchesProcessed int64   `json:"batches_processed"`
	UpdatesProcessed int64   `json:"updates_processed"`
	UpdatesDropped   int64   `json:"updates_dropped"`
	UpdatesFailed    int64   `json:"updates_failed"`
	AverageBatchSize float64 `json:"average_batch_size"`
}

// processUpdates runs in a background goroutine to periodically flush updates.
func (sb *StatusBatcher) processUpdates() {
	defer sb.wg.Done()

	ticker := time.NewTicker(sb.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sb.ctx.Done():
			return
		case <-ticker.C:
			if err := sb.processBatch(); err != nil {
				log.Log.Error(err, "Failed to process batch during periodic flush")
			}
		}
	}
}

// triggerFlush triggers an immediate flush.
func (sb *StatusBatcher) triggerFlush() {
	go func() {
		if err := sb.processBatch(); err != nil {
			log.Log.Error(err, "Failed to process batch during triggered flush")
		}
	}()
}

// processBatch processes a batch of status updates.
func (sb *StatusBatcher) processBatch() error {
	sb.mu.Lock()

	if len(sb.queue) == 0 {
		sb.mu.Unlock()
		return nil
	}

	// Extract batch for processing.
	batchSize := len(sb.queue)
	if batchSize > sb.config.MaxBatchSize {
		batchSize = sb.config.MaxBatchSize
	}

	batch := make([]*StatusUpdate, batchSize)
	copy(batch, sb.queue[:batchSize])

	// Sort by priority if enabled.
	if sb.config.EnablePriority {
		sb.sortByPriority(batch)
	}

	// Clear processed items from queue and map.
	remaining := sb.queue[batchSize:]
	sb.queue = make([]*StatusUpdate, 0, cap(sb.queue))
	sb.queue = append(sb.queue, remaining...)

	for _, update := range batch {
		delete(sb.updates, update.NamespacedName)
	}

	// Reset flush timer.
	if sb.flushTimer != nil {
		sb.flushTimer.Stop()
		sb.flushTimer = nil
	}

	sb.mu.Unlock()

	// Process batch outside of lock.
	successCount := 0
	for _, update := range batch {
		if err := sb.processUpdate(update); err != nil {
			log.Log.Error(err, "Failed to process status update", "resource", update.NamespacedName)

			// Retry logic for failed updates.
			if update.RetryCount < sb.config.MaxRetries {
				update.RetryCount++

				// Requeue with delay.
				go func(u *StatusUpdate) {
					time.Sleep(time.Duration(u.RetryCount) * sb.config.RetryDelay)
					sb.QueueUpdate(u.NamespacedName, u.UpdateFunc, u.Priority)
				}(update)
			} else {
				sb.updatesFailed++
			}
		} else {
			successCount++
		}
	}

	// Update metrics.
	sb.batchesProcessed++
	sb.updatesProcessed += int64(successCount)

	// Update average batch size (exponential moving average).
	if sb.averageBatchSize == 0 {
		sb.averageBatchSize = float64(batchSize)
	} else {
		alpha := 0.1 // Smoothing factor
		sb.averageBatchSize = alpha*float64(batchSize) + (1-alpha)*sb.averageBatchSize
	}

	return nil
}

// processUpdate processes a single status update.
func (sb *StatusBatcher) processUpdate(update *StatusUpdate) error {
	// Determine object type based on the resource.
	var obj client.Object

	// Create appropriate object type - this is a simplified approach.
	// In practice, you might want to maintain a registry of types.
	if update.NamespacedName.Namespace != "" {
		// Try NetworkIntent first.
		obj = &nephoranv1.NetworkIntent{}
		if err := sb.client.Get(sb.ctx, update.NamespacedName, obj); err != nil {
			// Try E2NodeSet.
			obj = &nephoranv1.E2NodeSet{}
			if err := sb.client.Get(sb.ctx, update.NamespacedName, obj); err != nil {
				return fmt.Errorf("failed to get object %s: %w", update.NamespacedName, err)
			}
		}
	} else {
		return fmt.Errorf("invalid namespaced name: %s", update.NamespacedName)
	}

	// Apply the update function.
	if err := update.UpdateFunc(obj); err != nil {
		return fmt.Errorf("failed to apply update function: %w", err)
	}

	// Update the object status.
	if err := sb.client.Status().Update(sb.ctx, obj); err != nil {
		return fmt.Errorf("failed to update object status: %w", err)
	}

	return nil
}

// sortByPriority sorts updates by priority (highest first).
func (sb *StatusBatcher) sortByPriority(updates []*StatusUpdate) {
	sort.Slice(updates, func(i, j int) bool {
		return updates[i].Priority > updates[j].Priority
	})
}

// updateCondition updates a condition in a condition slice.
func updateCondition(conditions *[]metav1.Condition, newCondition metav1.Condition) {
	if conditions == nil {
		*conditions = make([]metav1.Condition, 0, 1) // Preallocate for typical single condition
	}

	// Find existing condition.
	for i, condition := range *conditions {
		if condition.Type == newCondition.Type {
			// Update existing condition.
			(*conditions)[i] = newCondition
			return
		}
	}

	// Add new condition.
	*conditions = append(*conditions, newCondition)
}

// updateE2NodeSetCondition updates a condition in an E2NodeSet condition slice.
// This converts metav1.Condition to E2NodeSetCondition.
func updateE2NodeSetCondition(conditions *[]nephoranv1.E2NodeSetCondition, newCondition metav1.Condition) {
	if conditions == nil {
		*conditions = make([]nephoranv1.E2NodeSetCondition, 0, 1) // Preallocate for typical single condition
	}

	// Convert metav1.Condition to E2NodeSetCondition.
	e2Condition := nephoranv1.E2NodeSetCondition{
		Type:               nephoranv1.E2NodeSetConditionType(newCondition.Type),
		Status:             newCondition.Status,
		LastTransitionTime: newCondition.LastTransitionTime,
		Reason:             newCondition.Reason,
		Message:            newCondition.Message,
	}

	// Find existing condition.
	for i, condition := range *conditions {
		if condition.Type == e2Condition.Type {
			// Update existing condition.
			(*conditions)[i] = e2Condition
			return
		}
	}

	// Add new condition.
	*conditions = append(*conditions, e2Condition)
}
