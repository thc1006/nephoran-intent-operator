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

package orchestration

import (
	
	"encoding/json"
"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/client-go/util/workqueue"

	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/interfaces"
)

// ProcessingJob represents a job to be processed by a worker.

type ProcessingJob struct {
	ID string `json:"id"`

	IntentID string `json:"intentId"`

	Phase interfaces.ProcessingPhase `json:"phase"`

	Priority int `json:"priority"`

	Data json.RawMessage `json:"data"`

	Context *interfaces.ProcessingContext `json:"context"`

	RetryCount int `json:"retryCount"`

	MaxRetries int `json:"maxRetries"`

	Timeout time.Duration `json:"timeout"`

	CreatedAt time.Time `json:"createdAt"`

	ScheduledAt *time.Time `json:"scheduledAt,omitempty"`

	StartedAt *time.Time `json:"startedAt,omitempty"`

	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// WorkQueueManager manages work queues for different processing phases.

type WorkQueueManager struct {
	logger logr.Logger

	config *OrchestratorConfig

	// Work queues per phase.

	queues map[interfaces.ProcessingPhase]workqueue.TypedRateLimitingInterface[string]

	// Worker pools per phase.

	workerPools map[interfaces.ProcessingPhase]*WorkerPool

	// Priority queue for global job scheduling.

	priorityQueue *PriorityQueue

	// Job tracking.

	activeJobs sync.Map // map[string]*ProcessingJob

	completedJobs sync.Map // map[string]*JobResult

	// Control channels.

	stopChan chan bool

	started bool

	mutex sync.RWMutex

	// Metrics.

	metrics *WorkQueueMetrics
}

// NewWorkQueueManager creates a new work queue manager.

func NewWorkQueueManager(config *OrchestratorConfig, logger logr.Logger) *WorkQueueManager {
	return &WorkQueueManager{
		logger: logger.WithName("work-queue-manager"),

		config: config,

		queues: make(map[interfaces.ProcessingPhase]workqueue.TypedRateLimitingInterface[string]),

		workerPools: make(map[interfaces.ProcessingPhase]*WorkerPool),

		priorityQueue: NewPriorityQueue(),

		stopChan: make(chan bool),

		metrics: NewWorkQueueMetrics(),
	}
}

// Start starts the work queue manager.

func (wqm *WorkQueueManager) Start(ctx context.Context) error {
	wqm.mutex.Lock()

	defer wqm.mutex.Unlock()

	if wqm.started {
		return fmt.Errorf("work queue manager already started")
	}

	// Initialize queues and worker pools for each phase.

	phases := []interfaces.ProcessingPhase{
		interfaces.PhaseLLMProcessing,

		interfaces.PhaseResourcePlanning,

		interfaces.PhaseManifestGeneration,

		interfaces.PhaseGitOpsCommit,

		interfaces.PhaseDeploymentVerification,
	}

	for _, phase := range phases {

		// Create rate-limited work queue.

		queue := workqueue.NewTypedRateLimitingQueue(

			workqueue.DefaultTypedControllerRateLimiter[string](),
		)

		wqm.queues[phase] = queue

		// Get worker count for this phase.

		workerCount := wqm.getWorkerCount(phase)

		// Create worker pool.

		workerPool := NewWorkerPool(

			string(phase),

			workerCount,

			queue,

			wqm.processJob,

			wqm.logger.WithValues("phase", phase),
		)

		wqm.workerPools[phase] = workerPool

		// Start worker pool.

		if err := workerPool.Start(ctx); err != nil {
			return fmt.Errorf("failed to start worker pool for phase %s: %w", phase, err)
		}

	}

	// Start priority queue processor.

	go wqm.processPriorityQueue(ctx)

	// Start metrics collector.

	go wqm.collectMetrics(ctx)

	wqm.started = true

	wqm.logger.Info("Work queue manager started")

	return nil
}

// Stop stops the work queue manager.

func (wqm *WorkQueueManager) Stop(ctx context.Context) error {
	wqm.mutex.Lock()

	defer wqm.mutex.Unlock()

	if !wqm.started {
		return nil
	}

	wqm.logger.Info("Stopping work queue manager")

	// Stop all worker pools.

	for phase, workerPool := range wqm.workerPools {

		wqm.logger.Info("Stopping worker pool", "phase", phase)

		if err := workerPool.Stop(ctx); err != nil {
			wqm.logger.Error(err, "Error stopping worker pool", "phase", phase)
		}

	}

	// Shutdown all queues.

	for phase, queue := range wqm.queues {

		wqm.logger.Info("Shutting down queue", "phase", phase)

		queue.ShutDown()

	}

	// Signal stop to background goroutines.

	close(wqm.stopChan)

	wqm.started = false

	wqm.logger.Info("Work queue manager stopped")

	return nil
}

// EnqueueJob adds a job to the appropriate work queue.

func (wqm *WorkQueueManager) EnqueueJob(ctx context.Context, phase interfaces.ProcessingPhase, job ProcessingJob) error {
	if !wqm.started {
		return fmt.Errorf("work queue manager not started")
	}

	// Set creation time.

	job.CreatedAt = time.Now()

	// Validate job.

	if err := wqm.validateJob(job); err != nil {
		return fmt.Errorf("job validation failed: %w", err)
	}

	// Store job for tracking.

	wqm.activeJobs.Store(job.ID, &job)

	// Add to priority queue for global scheduling.

	wqm.priorityQueue.Push(job)

	// Record metrics.

	wqm.metrics.RecordJobEnqueued(phase)

	wqm.logger.Info("Job enqueued", "jobId", job.ID, "phase", phase, "priority", job.Priority)

	return nil
}

// processPriorityQueue processes jobs from the priority queue.

func (wqm *WorkQueueManager) processPriorityQueue(ctx context.Context) {
	wqm.logger.Info("Started priority queue processing")

	ticker := time.NewTicker(100 * time.Millisecond) // Process every 100ms

	defer ticker.Stop()

	for {
		select {

		case <-ticker.C:

			wqm.processNextPriorityJob(ctx)

		case <-wqm.stopChan:

			wqm.logger.Info("Priority queue processing stopped")

			return

		case <-ctx.Done():

			wqm.logger.Info("Priority queue processing cancelled")

			return

		}
	}
}

// processNextPriorityJob processes the next job from the priority queue.

func (wqm *WorkQueueManager) processNextPriorityJob(ctx context.Context) {
	job := wqm.priorityQueue.Pop()

	if job == nil {
		return
	}

	// Check if we can schedule this job (respect concurrency limits).

	if !wqm.canScheduleJob(*job) {

		// Put it back in the queue for later.

		wqm.priorityQueue.Push(*job)

		return

	}

	// Get the appropriate queue for this phase.

	queue, exists := wqm.queues[job.Phase]

	if !exists {

		wqm.logger.Error(fmt.Errorf("no queue found"), "Unknown phase", "phase", job.Phase)

		wqm.recordJobFailure(*job, fmt.Errorf("no queue found for phase %s", job.Phase))

		return

	}

	// Mark job as scheduled.

	now := time.Now()

	job.ScheduledAt = &now

	wqm.activeJobs.Store(job.ID, job)

	// Add to phase-specific queue.

	queue.Add(job.ID)

	wqm.logger.V(1).Info("Job scheduled to phase queue", "jobId", job.ID, "phase", job.Phase)
}

// canScheduleJob checks if a job can be scheduled based on concurrency limits.

func (wqm *WorkQueueManager) canScheduleJob(job ProcessingJob) bool {
	// Check global concurrency limit.

	activeJobCount := 0

	wqm.activeJobs.Range(func(key, value interface{}) bool {
		if j, ok := value.(*ProcessingJob); ok && j.StartedAt != nil && j.CompletedAt == nil {
			activeJobCount++
		}

		return true
	})

	if activeJobCount >= wqm.config.MaxConcurrentIntents {
		return false
	}

	// Check phase-specific concurrency limit.

	phaseConfig, exists := wqm.config.PhaseConfigs[job.Phase]

	if exists && phaseConfig.MaxConcurrency > 0 {

		phaseActiveCount := 0

		wqm.activeJobs.Range(func(key, value interface{}) bool {
			if j, ok := value.(*ProcessingJob); ok && j.Phase == job.Phase && j.StartedAt != nil && j.CompletedAt == nil {
				phaseActiveCount++
			}

			return true
		})

		if phaseActiveCount >= phaseConfig.MaxConcurrency {
			return false
		}

	}

	return true
}

// processJob processes a single job (called by worker pools).

func (wqm *WorkQueueManager) processJob(ctx context.Context, jobID string) error {
	// Retrieve job from active jobs.

	jobInterface, exists := wqm.activeJobs.Load(jobID)

	if !exists {
		return fmt.Errorf("job %s not found", jobID)
	}

	job, ok := jobInterface.(*ProcessingJob)

	if !ok {
		return fmt.Errorf("invalid job type for %s", jobID)
	}

	// Mark job as started.

	now := time.Now()

	job.StartedAt = &now

	wqm.activeJobs.Store(jobID, job)

	wqm.logger.Info("Processing job", "jobId", jobID, "phase", job.Phase, "attempt", job.RetryCount+1)

	// Record metrics.

	wqm.metrics.RecordJobStarted(job.Phase)

	// Create job-specific context with timeout.

	jobCtx, cancel := context.WithTimeout(ctx, job.Timeout)

	defer cancel()

	// Execute the job (this would call the appropriate controller).

	result, err := wqm.executeJob(jobCtx, job)

	// Mark job as completed.

	completedAt := time.Now()

	job.CompletedAt = &completedAt

	// Handle result.

	if err != nil {
		return wqm.handleJobError(job, err)
	}

	return wqm.handleJobSuccess(job, result)
}

// executeJob executes a job by delegating to the appropriate controller.

func (wqm *WorkQueueManager) executeJob(ctx context.Context, job *ProcessingJob) (*JobResult, error) {
	// This is a simplified implementation - in practice, this would.

	// delegate to the appropriate phase controller.

	wqm.logger.Info("Executing job", "jobId", job.ID, "phase", job.Phase)

	// Simulate job execution.

	result := &JobResult{
		JobID: job.ID,

		Phase: job.Phase,

		Success: true,

		Data: make(map[string]interface{}),

		StartTime: *job.StartedAt,

		EndTime: time.Now(),
	}

	result.Duration = result.EndTime.Sub(result.StartTime)

	result.Data["executionPhase"] = string(job.Phase)

	result.Data["processingContext"] = job.Context

	return result, nil
}

// handleJobSuccess handles successful job completion.

func (wqm *WorkQueueManager) handleJobSuccess(job *ProcessingJob, result *JobResult) error {
	wqm.logger.Info("Job completed successfully", "jobId", job.ID, "phase", job.Phase, "duration", result.Duration)

	// Store result.

	wqm.completedJobs.Store(job.ID, result)

	// Remove from active jobs.

	wqm.activeJobs.Delete(job.ID)

	// Record metrics.

	wqm.metrics.RecordJobCompleted(job.Phase, result.Duration, true)

	return nil
}

// handleJobError handles job execution errors.

func (wqm *WorkQueueManager) handleJobError(job *ProcessingJob, err error) error {
	wqm.logger.Error(err, "Job execution failed", "jobId", job.ID, "phase", job.Phase, "attempt", job.RetryCount+1)

	// Check if we should retry.

	if job.RetryCount < job.MaxRetries {

		job.RetryCount++

		job.StartedAt = nil

		job.CompletedAt = nil

		// Put back in priority queue for retry.

		wqm.priorityQueue.Push(*job)

		wqm.logger.Info("Job scheduled for retry", "jobId", job.ID, "attempt", job.RetryCount)

		return nil

	}

	// Max retries exceeded.

	return wqm.recordJobFailure(*job, err)
}

// recordJobFailure records a permanent job failure.

func (wqm *WorkQueueManager) recordJobFailure(job ProcessingJob, err error) error {
	wqm.logger.Error(err, "Job failed permanently", "jobId", job.ID, "phase", job.Phase, "attempts", job.RetryCount+1)

	// Create failure result.

	result := &JobResult{
		JobID: job.ID,

		Phase: job.Phase,

		Success: false,

		Error: err.Error(),

		StartTime: *job.StartedAt,

		EndTime: time.Now(),
	}

	result.Duration = result.EndTime.Sub(result.StartTime)

	// Store result.

	wqm.completedJobs.Store(job.ID, result)

	// Remove from active jobs.

	wqm.activeJobs.Delete(job.ID)

	// Record metrics.

	wqm.metrics.RecordJobCompleted(job.Phase, result.Duration, false)

	return err
}

// validateJob validates a processing job.

func (wqm *WorkQueueManager) validateJob(job ProcessingJob) error {
	if job.ID == "" {
		return fmt.Errorf("job ID is required")
	}

	if job.IntentID == "" {
		return fmt.Errorf("intent ID is required")
	}

	if job.Context == nil {
		return fmt.Errorf("processing context is required")
	}

	if job.Timeout <= 0 {
		job.Timeout = 5 * time.Minute // Default timeout
	}

	if job.MaxRetries < 0 {
		job.MaxRetries = 3 // Default retries
	}

	return nil
}

// getWorkerCount returns the number of workers for a phase.

func (wqm *WorkQueueManager) getWorkerCount(phase interfaces.ProcessingPhase) int {
	if phaseConfig, exists := wqm.config.PhaseConfigs[phase]; exists && phaseConfig.MaxConcurrency > 0 {
		return phaseConfig.MaxConcurrency
	}

	// Default worker counts per phase.

	switch phase {

	case interfaces.PhaseLLMProcessing:

		return 5 // Limited by external API rate limits

	case interfaces.PhaseResourcePlanning:

		return 10 // CPU-intensive but can parallelize

	case interfaces.PhaseManifestGeneration:

		return 15 // Fast, template-based generation

	case interfaces.PhaseGitOpsCommit:

		return 3 // Limited by git repository access

	case interfaces.PhaseDeploymentVerification:

		return 8 // I/O bound, can parallelize

	default:

		return 5 // Conservative default

	}
}

// collectMetrics collects metrics in the background.

func (wqm *WorkQueueManager) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)

	defer ticker.Stop()

	for {
		select {

		case <-ticker.C:

			wqm.updateQueueMetrics()

		case <-wqm.stopChan:

			return

		case <-ctx.Done():

			return

		}
	}
}

// updateQueueMetrics updates queue depth and processing metrics.

func (wqm *WorkQueueManager) updateQueueMetrics() {
	for phase, queue := range wqm.queues {

		depth := queue.Len()

		wqm.metrics.UpdateQueueDepth(phase, depth)

	}

	// Update active job count.

	activeCount := 0

	wqm.activeJobs.Range(func(key, value interface{}) bool {
		if job, ok := value.(*ProcessingJob); ok && job.StartedAt != nil && job.CompletedAt == nil {
			activeCount++
		}

		return true
	})

	wqm.metrics.UpdateActiveJobCount(activeCount)
}

// GetJobStatus returns the status of a job.

func (wqm *WorkQueueManager) GetJobStatus(jobID string) (*JobStatus, error) {
	// Check active jobs.

	if jobInterface, exists := wqm.activeJobs.Load(jobID); exists {

		job, ok := jobInterface.(*ProcessingJob)

		if !ok {
			return nil, fmt.Errorf("invalid job type")
		}

		status := &JobStatus{
			JobID: job.ID,

			Phase: job.Phase,

			Status: wqm.getJobStatusString(job),

			CreatedAt: job.CreatedAt,

			ScheduledAt: job.ScheduledAt,

			StartedAt: job.StartedAt,

			CompletedAt: job.CompletedAt,

			RetryCount: job.RetryCount,

			MaxRetries: job.MaxRetries,
		}

		if job.StartedAt != nil && job.CompletedAt == nil {
			status.Duration = time.Since(*job.StartedAt)
		}

		return status, nil

	}

	// Check completed jobs.

	if resultInterface, exists := wqm.completedJobs.Load(jobID); exists {

		result, ok := resultInterface.(*JobResult)

		if !ok {
			return nil, fmt.Errorf("invalid result type")
		}

		status := &JobStatus{
			JobID: result.JobID,

			Phase: result.Phase,

			Status: wqm.getResultStatusString(result),

			CreatedAt: result.StartTime, // Approximation

			StartedAt: &result.StartTime,

			CompletedAt: &result.EndTime,

			Duration: result.Duration,
		}

		return status, nil

	}

	return nil, fmt.Errorf("job %s not found", jobID)
}

// getJobStatusString returns a human-readable status string for a job.

func (wqm *WorkQueueManager) getJobStatusString(job *ProcessingJob) string {
	if job.CompletedAt != nil {
		return "completed"
	}

	if job.StartedAt != nil {
		return "processing"
	}

	if job.ScheduledAt != nil {
		return "scheduled"
	}

	return "queued"
}

// getResultStatusString returns a status string for a job result.

func (wqm *WorkQueueManager) getResultStatusString(result *JobResult) string {
	if result.Success {
		return "completed_success"
	}

	return "completed_failed"
}

// GetMetrics returns current work queue metrics.

func (wqm *WorkQueueManager) GetMetrics() map[string]interface{} {
	return wqm.metrics.GetMetrics()
}

// JobResult represents the result of a job execution.

type JobResult struct {
	JobID string `json:"jobId"`

	Phase interfaces.ProcessingPhase `json:"phase"`

	Success bool `json:"success"`

	Data json.RawMessage `json:"data,omitempty"`

	Error string `json:"error,omitempty"`

	StartTime time.Time `json:"startTime"`

	EndTime time.Time `json:"endTime"`

	Duration time.Duration `json:"duration"`
}

// JobStatus represents the current status of a job.

type JobStatus struct {
	JobID string `json:"jobId"`

	Phase interfaces.ProcessingPhase `json:"phase"`

	Status string `json:"status"`

	CreatedAt time.Time `json:"createdAt"`

	ScheduledAt *time.Time `json:"scheduledAt,omitempty"`

	StartedAt *time.Time `json:"startedAt,omitempty"`

	CompletedAt *time.Time `json:"completedAt,omitempty"`

	Duration time.Duration `json:"duration"`

	RetryCount int `json:"retryCount"`

	MaxRetries int `json:"maxRetries"`
}

// PriorityQueue implements a priority queue for processing jobs.

type PriorityQueue struct {
	jobs []ProcessingJob

	mutex sync.RWMutex
}

// NewPriorityQueue creates a new priority queue.

func NewPriorityQueue() *PriorityQueue {
	return &PriorityQueue{
		jobs: make([]ProcessingJob, 0),
	}
}

// Push adds a job to the priority queue.

func (pq *PriorityQueue) Push(job ProcessingJob) {
	pq.mutex.Lock()

	defer pq.mutex.Unlock()

	pq.jobs = append(pq.jobs, job)

	// Sort by priority (higher priority first).

	sort.Slice(pq.jobs, func(i, j int) bool {
		if pq.jobs[i].Priority != pq.jobs[j].Priority {
			return pq.jobs[i].Priority > pq.jobs[j].Priority
		}

		// If priorities are equal, sort by creation time (FIFO).

		return pq.jobs[i].CreatedAt.Before(pq.jobs[j].CreatedAt)
	})
}

// Pop removes and returns the highest priority job.

func (pq *PriorityQueue) Pop() *ProcessingJob {
	pq.mutex.Lock()

	defer pq.mutex.Unlock()

	if len(pq.jobs) == 0 {
		return nil
	}

	job := pq.jobs[0]

	pq.jobs = pq.jobs[1:]

	return &job
}

// Len returns the number of jobs in the queue.

func (pq *PriorityQueue) Len() int {
	pq.mutex.RLock()

	defer pq.mutex.RUnlock()

	return len(pq.jobs)
}

// WorkQueueMetrics tracks work queue performance metrics.

type WorkQueueMetrics struct {
	JobsEnqueued map[interfaces.ProcessingPhase]int64 `json:"jobsEnqueued"`

	JobsStarted map[interfaces.ProcessingPhase]int64 `json:"jobsStarted"`

	JobsCompleted map[interfaces.ProcessingPhase]int64 `json:"jobsCompleted"`

	JobsFailed map[interfaces.ProcessingPhase]int64 `json:"jobsFailed"`

	QueueDepths map[interfaces.ProcessingPhase]int `json:"queueDepths"`

	ProcessingTimes map[interfaces.ProcessingPhase]time.Duration `json:"processingTimes"`

	ActiveJobCount int `json:"activeJobCount"`

	mutex sync.RWMutex
}

// NewWorkQueueMetrics creates new work queue metrics.

func NewWorkQueueMetrics() *WorkQueueMetrics {
	return &WorkQueueMetrics{
		JobsEnqueued: make(map[interfaces.ProcessingPhase]int64),

		JobsStarted: make(map[interfaces.ProcessingPhase]int64),

		JobsCompleted: make(map[interfaces.ProcessingPhase]int64),

		JobsFailed: make(map[interfaces.ProcessingPhase]int64),

		QueueDepths: make(map[interfaces.ProcessingPhase]int),

		ProcessingTimes: make(map[interfaces.ProcessingPhase]time.Duration),
	}
}

// RecordJobEnqueued records a job being enqueued.

func (m *WorkQueueMetrics) RecordJobEnqueued(phase interfaces.ProcessingPhase) {
	m.mutex.Lock()

	defer m.mutex.Unlock()

	m.JobsEnqueued[phase]++
}

// RecordJobStarted records a job starting.

func (m *WorkQueueMetrics) RecordJobStarted(phase interfaces.ProcessingPhase) {
	m.mutex.Lock()

	defer m.mutex.Unlock()

	m.JobsStarted[phase]++
}

// RecordJobCompleted records a job completion.

func (m *WorkQueueMetrics) RecordJobCompleted(phase interfaces.ProcessingPhase, duration time.Duration, success bool) {
	m.mutex.Lock()

	defer m.mutex.Unlock()

	if success {
		m.JobsCompleted[phase]++
	} else {
		m.JobsFailed[phase]++
	}

	// Update average processing time.

	if currentAvg, exists := m.ProcessingTimes[phase]; exists {
		m.ProcessingTimes[phase] = (currentAvg + duration) / 2
	} else {
		m.ProcessingTimes[phase] = duration
	}
}

// UpdateQueueDepth updates the queue depth for a phase.

func (m *WorkQueueMetrics) UpdateQueueDepth(phase interfaces.ProcessingPhase, depth int) {
	m.mutex.Lock()

	defer m.mutex.Unlock()

	m.QueueDepths[phase] = depth
}

// UpdateActiveJobCount updates the active job count.

func (m *WorkQueueMetrics) UpdateActiveJobCount(count int) {
	m.mutex.Lock()

	defer m.mutex.Unlock()

	m.ActiveJobCount = count
}

// GetMetrics returns current metrics.

func (m *WorkQueueMetrics) GetMetrics() map[string]interface{} {
	m.mutex.RLock()

	defer m.mutex.RUnlock()

	return json.RawMessage("{}")
}
