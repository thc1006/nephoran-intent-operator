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




package shared



import (

	"container/heap"

	"fmt"

	"sync"

	"time"



	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/interfaces"



	"k8s.io/apimachinery/pkg/types"

)



// ExecutionTask represents a task to be executed.

type ExecutionTask struct {

	IntentName   types.NamespacedName       `json:"intentName"`

	Phase        interfaces.ProcessingPhase `json:"phase"`

	Priority     int                        `json:"priority"`

	Timestamp    time.Time                  `json:"timestamp"`

	Context      map[string]interface{}     `json:"context"`

	RetryCount   int                        `json:"retryCount"`

	MaxRetries   int                        `json:"maxRetries"`

	Dependencies []string                   `json:"dependencies,omitempty"`



	// Internal fields.

	index int // For heap implementation

}



// ExecutionQueue manages task execution with priority and ordering.

type ExecutionQueue struct {

	mutex    sync.RWMutex

	heap     *TaskHeap

	channel  chan *ExecutionTask

	capacity int



	// Task tracking.

	activeTasks    map[string]*ExecutionTask

	pendingTasks   map[string]*ExecutionTask

	completedTasks map[string]*ExecutionTask



	// Statistics.

	enqueuedCount int64

	dequeuedCount int64

	droppedCount  int64



	// Configuration.

	maxPendingTasks int

	taskTimeout     time.Duration

}



// TaskHeap implements a priority queue for execution tasks.

type TaskHeap []*ExecutionTask



// Len performs len operation.

func (h TaskHeap) Len() int { return len(h) }



// Less performs less operation.

func (h TaskHeap) Less(i, j int) bool {

	// Higher priority first, then older tasks first.

	if h[i].Priority != h[j].Priority {

		return h[i].Priority > h[j].Priority

	}

	return h[i].Timestamp.Before(h[j].Timestamp)

}



// Swap performs swap operation.

func (h TaskHeap) Swap(i, j int) {

	h[i], h[j] = h[j], h[i]

	h[i].index = i

	h[j].index = j

}



// Push performs push operation.

func (h *TaskHeap) Push(x interface{}) {

	n := len(*h)

	task := x.(*ExecutionTask)

	task.index = n

	*h = append(*h, task)

}



// Pop performs pop operation.

func (h *TaskHeap) Pop() interface{} {

	old := *h

	n := len(old)

	task := old[n-1]

	old[n-1] = nil  // avoid memory leak

	task.index = -1 // for safety

	*h = old[0 : n-1]

	return task

}



// NewExecutionQueue creates a new execution queue.

func NewExecutionQueue(capacity int) *ExecutionQueue {

	eq := &ExecutionQueue{

		heap:            &TaskHeap{},

		channel:         make(chan *ExecutionTask, capacity),

		capacity:        capacity,

		activeTasks:     make(map[string]*ExecutionTask),

		pendingTasks:    make(map[string]*ExecutionTask),

		completedTasks:  make(map[string]*ExecutionTask),

		maxPendingTasks: capacity * 2, // Allow more pending than active

		taskTimeout:     30 * time.Minute,

	}



	heap.Init(eq.heap)



	// Start background processor.

	go eq.processor()



	return eq

}



// Enqueue adds a task to the execution queue.

func (eq *ExecutionQueue) Enqueue(task *ExecutionTask) error {

	eq.mutex.Lock()

	defer eq.mutex.Unlock()



	taskKey := eq.getTaskKey(task)



	// Check if task already exists.

	if _, exists := eq.activeTasks[taskKey]; exists {

		return fmt.Errorf("task already active: %s", taskKey)

	}



	if _, exists := eq.pendingTasks[taskKey]; exists {

		return fmt.Errorf("task already pending: %s", taskKey)

	}



	// Check capacity.

	if len(eq.pendingTasks) >= eq.maxPendingTasks {

		eq.droppedCount++

		return fmt.Errorf("queue at capacity")

	}



	// Add to pending tasks.

	eq.pendingTasks[taskKey] = task



	// Add to priority heap.

	heap.Push(eq.heap, task)



	eq.enqueuedCount++



	return nil

}



// Dequeue removes and returns the highest priority task.

func (eq *ExecutionQueue) Dequeue() (*ExecutionTask, error) {

	eq.mutex.Lock()

	defer eq.mutex.Unlock()



	if eq.heap.Len() == 0 {

		return nil, fmt.Errorf("queue is empty")

	}



	// Get highest priority task.

	task := heap.Pop(eq.heap).(*ExecutionTask)

	taskKey := eq.getTaskKey(task)



	// Move from pending to active.

	delete(eq.pendingTasks, taskKey)

	eq.activeTasks[taskKey] = task



	eq.dequeuedCount++



	return task, nil

}



// CompleteTask marks a task as completed.

func (eq *ExecutionQueue) CompleteTask(task *ExecutionTask) {

	eq.mutex.Lock()

	defer eq.mutex.Unlock()



	taskKey := eq.getTaskKey(task)



	// Move from active to completed.

	if _, exists := eq.activeTasks[taskKey]; exists {

		delete(eq.activeTasks, taskKey)

		eq.completedTasks[taskKey] = task



		// Limit completed tasks history.

		if len(eq.completedTasks) > 1000 {

			// Remove oldest 100 tasks.

			count := 0

			for key := range eq.completedTasks {

				delete(eq.completedTasks, key)

				count++

				if count >= 100 {

					break

				}

			}

		}

	}

}



// FailTask marks a task as failed and potentially retries it.

func (eq *ExecutionQueue) FailTask(task *ExecutionTask, err error) error {

	eq.mutex.Lock()

	defer eq.mutex.Unlock()



	taskKey := eq.getTaskKey(task)



	// Remove from active tasks.

	delete(eq.activeTasks, taskKey)



	// Check if retry is needed.

	if task.RetryCount < task.MaxRetries {

		task.RetryCount++

		task.Priority += 10 // Increase priority for retries

		task.Timestamp = time.Now()



		// Re-enqueue for retry.

		eq.pendingTasks[taskKey] = task

		heap.Push(eq.heap, task)



		return nil

	}



	// Mark as completed with failure.

	eq.completedTasks[taskKey] = task



	return fmt.Errorf("task failed after %d retries: %w", task.MaxRetries, err)

}



// GetPendingCount returns the number of pending tasks.

func (eq *ExecutionQueue) GetPendingCount() int {

	eq.mutex.RLock()

	defer eq.mutex.RUnlock()



	return len(eq.pendingTasks)

}



// GetActiveCount returns the number of active tasks.

func (eq *ExecutionQueue) GetActiveCount() int {

	eq.mutex.RLock()

	defer eq.mutex.RUnlock()



	return len(eq.activeTasks)

}



// GetStats returns queue statistics.

func (eq *ExecutionQueue) GetStats() *QueueStats {

	eq.mutex.RLock()

	defer eq.mutex.RUnlock()



	return &QueueStats{

		PendingTasks:    int64(len(eq.pendingTasks)),

		ActiveTasks:     int64(len(eq.activeTasks)),

		CompletedTasks:  int64(len(eq.completedTasks)),

		EnqueuedCount:   eq.enqueuedCount,

		DequeuedCount:   eq.dequeuedCount,

		DroppedCount:    eq.droppedCount,

		Capacity:        int64(eq.capacity),

		MaxPendingTasks: int64(eq.maxPendingTasks),

	}

}



// ListActiveTasks returns all active tasks.

func (eq *ExecutionQueue) ListActiveTasks() []*ExecutionTask {

	eq.mutex.RLock()

	defer eq.mutex.RUnlock()



	tasks := make([]*ExecutionTask, 0, len(eq.activeTasks))

	for _, task := range eq.activeTasks {

		tasks = append(tasks, task)

	}



	return tasks

}



// ListPendingTasks returns all pending tasks.

func (eq *ExecutionQueue) ListPendingTasks() []*ExecutionTask {

	eq.mutex.RLock()

	defer eq.mutex.RUnlock()



	tasks := make([]*ExecutionTask, 0, len(eq.pendingTasks))

	for _, task := range eq.pendingTasks {

		tasks = append(tasks, task)

	}



	return tasks

}



// CancelTask cancels a pending or active task.

func (eq *ExecutionQueue) CancelTask(intentName types.NamespacedName, phase interfaces.ProcessingPhase) error {

	eq.mutex.Lock()

	defer eq.mutex.Unlock()



	taskKey := fmt.Sprintf("%s/%s:%s", intentName.Namespace, intentName.Name, phase)



	// Check pending tasks.

	if task, exists := eq.pendingTasks[taskKey]; exists {

		// Remove from heap.

		eq.removeFromHeap(task)

		delete(eq.pendingTasks, taskKey)

		return nil

	}



	// Check active tasks (mark for cancellation).

	if task, exists := eq.activeTasks[taskKey]; exists {

		// Add cancellation flag to context.

		if task.Context == nil {

			task.Context = make(map[string]interface{})

		}

		task.Context["cancelled"] = true

		return nil

	}



	return fmt.Errorf("task not found: %s", taskKey)

}



// CleanupExpiredTasks removes expired tasks.

func (eq *ExecutionQueue) CleanupExpiredTasks() int {

	eq.mutex.Lock()

	defer eq.mutex.Unlock()



	cutoff := time.Now().Add(-eq.taskTimeout)

	expiredCount := 0



	// Check active tasks for expiration.

	for taskKey, task := range eq.activeTasks {

		if task.Timestamp.Before(cutoff) {

			delete(eq.activeTasks, taskKey)

			expiredCount++

		}

	}



	// Check pending tasks for expiration.

	expiredPending := make([]*ExecutionTask, 0)

	for taskKey, task := range eq.pendingTasks {

		if task.Timestamp.Before(cutoff) {

			delete(eq.pendingTasks, taskKey)

			expiredPending = append(expiredPending, task)

			expiredCount++

		}

	}



	// Remove expired pending tasks from heap.

	for _, task := range expiredPending {

		eq.removeFromHeap(task)

	}



	return expiredCount

}



// Internal methods.



func (eq *ExecutionQueue) processor() {

	ticker := time.NewTicker(100 * time.Millisecond) // Process queue 10 times per second

	defer ticker.Stop()



	for range ticker.C {

		eq.processQueue()

	}

}



func (eq *ExecutionQueue) processQueue() {

	// Check if we can process more tasks.

	if len(eq.channel) >= eq.capacity {

		return // Channel is full

	}



	// Try to dequeue a task.

	task, err := eq.Dequeue()

	if err != nil {

		return // No tasks available

	}



	// Check dependencies.

	if !eq.checkTaskDependencies(task) {

		// Re-enqueue task if dependencies not met.

		eq.mutex.Lock()

		taskKey := eq.getTaskKey(task)

		delete(eq.activeTasks, taskKey)

		eq.pendingTasks[taskKey] = task

		heap.Push(eq.heap, task)

		eq.mutex.Unlock()

		return

	}



	// Send to execution channel.

	select {

	case eq.channel <- task:

		// Task sent successfully.

	default:

		// Channel is full, re-enqueue.

		eq.mutex.Lock()

		taskKey := eq.getTaskKey(task)

		delete(eq.activeTasks, taskKey)

		eq.pendingTasks[taskKey] = task

		heap.Push(eq.heap, task)

		eq.mutex.Unlock()

	}

}



func (eq *ExecutionQueue) getTaskKey(task *ExecutionTask) string {

	return fmt.Sprintf("%s/%s:%s", task.IntentName.Namespace, task.IntentName.Name, task.Phase)

}



func (eq *ExecutionQueue) removeFromHeap(target *ExecutionTask) {

	if target.index < 0 || target.index >= eq.heap.Len() {

		return // Task not in heap

	}



	// Move target to end and remove.

	heap.Remove(eq.heap, target.index)

}



func (eq *ExecutionQueue) checkTaskDependencies(task *ExecutionTask) bool {

	if len(task.Dependencies) == 0 {

		return true // No dependencies

	}



	// Check if all dependencies are completed.

	for _, depKey := range task.Dependencies {

		if _, exists := eq.completedTasks[depKey]; !exists {

			return false // Dependency not completed

		}

	}



	return true

}



// QueueStats provides statistics about the execution queue.

type QueueStats struct {

	PendingTasks    int64 `json:"pendingTasks"`

	ActiveTasks     int64 `json:"activeTasks"`

	CompletedTasks  int64 `json:"completedTasks"`

	EnqueuedCount   int64 `json:"enqueuedCount"`

	DequeuedCount   int64 `json:"dequeuedCount"`

	DroppedCount    int64 `json:"droppedCount"`

	Capacity        int64 `json:"capacity"`

	MaxPendingTasks int64 `json:"maxPendingTasks"`

}



// TaskFilter allows filtering tasks based on criteria.

type TaskFilter struct {

	IntentNamespace string                     `json:"intentNamespace,omitempty"`

	IntentName      string                     `json:"intentName,omitempty"`

	Phase           interfaces.ProcessingPhase `json:"phase,omitempty"`

	MinPriority     *int                       `json:"minPriority,omitempty"`

	MaxPriority     *int                       `json:"maxPriority,omitempty"`

	CreatedAfter    *time.Time                 `json:"createdAfter,omitempty"`

	CreatedBefore   *time.Time                 `json:"createdBefore,omitempty"`

}



// FilterTasks filters tasks based on criteria.

func (eq *ExecutionQueue) FilterTasks(filter *TaskFilter, includeActive, includePending, includeCompleted bool) []*ExecutionTask {

	eq.mutex.RLock()

	defer eq.mutex.RUnlock()



	var allTasks []*ExecutionTask



	if includeActive {

		for _, task := range eq.activeTasks {

			allTasks = append(allTasks, task)

		}

	}



	if includePending {

		for _, task := range eq.pendingTasks {

			allTasks = append(allTasks, task)

		}

	}



	if includeCompleted {

		for _, task := range eq.completedTasks {

			allTasks = append(allTasks, task)

		}

	}



	// Apply filter.

	var filtered []*ExecutionTask

	for _, task := range allTasks {

		if eq.matchesFilter(task, filter) {

			filtered = append(filtered, task)

		}

	}



	return filtered

}



func (eq *ExecutionQueue) matchesFilter(task *ExecutionTask, filter *TaskFilter) bool {

	if filter.IntentNamespace != "" && task.IntentName.Namespace != filter.IntentNamespace {

		return false

	}



	if filter.IntentName != "" && task.IntentName.Name != filter.IntentName {

		return false

	}



	if filter.Phase != "" && task.Phase != filter.Phase {

		return false

	}



	if filter.MinPriority != nil && task.Priority < *filter.MinPriority {

		return false

	}



	if filter.MaxPriority != nil && task.Priority > *filter.MaxPriority {

		return false

	}



	if filter.CreatedAfter != nil && task.Timestamp.Before(*filter.CreatedAfter) {

		return false

	}



	if filter.CreatedBefore != nil && task.Timestamp.After(*filter.CreatedBefore) {

		return false

	}



	return true

}

