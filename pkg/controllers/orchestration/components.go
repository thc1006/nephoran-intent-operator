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
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nephio-project/nephoran-intent-operator/pkg/controllers/interfaces"
)

// WorkerPool manages a pool of workers for processing jobs.

type WorkerPool struct {
	id string

	workerCount int

	workers []*Worker

	workQueue workqueue.TypedRateLimitingInterface[string]

	processor JobProcessor

	logger logr.Logger

	// Control.

	stopChan chan bool

	started bool

	mutex sync.RWMutex
}

// JobProcessor defines the interface for processing jobs.

type JobProcessor func(ctx context.Context, jobID string) error

// Worker represents a single worker in the pool.

type Worker struct {
	id string

	workQueue workqueue.TypedRateLimitingInterface[string]

	processor JobProcessor

	logger logr.Logger

	stopChan chan bool

	started bool
}

// NewWorkerPool creates a new worker pool.

func NewWorkerPool(id string, workerCount int, workQueue workqueue.TypedRateLimitingInterface[string], processor JobProcessor, logger logr.Logger) *WorkerPool {

	return &WorkerPool{

		id: id,

		workerCount: workerCount,

		workQueue: workQueue,

		processor: processor,

		logger: logger.WithName("worker-pool").WithValues("poolId", id),

		workers: make([]*Worker, 0, workerCount),

		stopChan: make(chan bool),
	}

}

// Start starts the worker pool.

func (wp *WorkerPool) Start(ctx context.Context) error {

	wp.mutex.Lock()

	defer wp.mutex.Unlock()

	if wp.started {

		return fmt.Errorf("worker pool %s already started", wp.id)

	}

	// Create and start workers.

	for i := range wp.workerCount {

		worker := &Worker{

			id: fmt.Sprintf("%s-worker-%d", wp.id, i),

			workQueue: wp.workQueue,

			processor: wp.processor,

			logger: wp.logger.WithValues("workerId", fmt.Sprintf("%s-worker-%d", wp.id, i)),

			stopChan: make(chan bool),
		}

		wp.workers = append(wp.workers, worker)

		go worker.Start(ctx)

	}

	wp.started = true

	wp.logger.Info("Worker pool started", "workerCount", wp.workerCount)

	return nil

}

// Stop stops the worker pool.

func (wp *WorkerPool) Stop(ctx context.Context) error {

	wp.mutex.Lock()

	defer wp.mutex.Unlock()

	if !wp.started {

		return nil

	}

	wp.logger.Info("Stopping worker pool")

	// Stop all workers.

	for _, worker := range wp.workers {

		worker.Stop(ctx)

	}

	close(wp.stopChan)

	wp.started = false

	wp.logger.Info("Worker pool stopped")

	return nil

}

// GetStatus returns the status of the worker pool.

func (wp *WorkerPool) GetStatus() WorkerPoolStatus {

	wp.mutex.RLock()

	defer wp.mutex.RUnlock()

	status := WorkerPoolStatus{

		ID: wp.id,

		WorkerCount: wp.workerCount,

		Started: wp.started,

		QueueLength: wp.workQueue.Len(),

		Workers: make([]WorkerStatus, len(wp.workers)),
	}

	for i, worker := range wp.workers {

		status.Workers[i] = WorkerStatus{

			ID: worker.id,

			Started: worker.started,
		}

	}

	return status

}

// Start starts a worker.

func (w *Worker) Start(ctx context.Context) {

	w.started = true

	w.logger.Info("Worker started")

	// Main processing loop.

	for {

		select {

		case <-w.stopChan:

			w.logger.Info("Worker stopped")

			return

		case <-ctx.Done():

			w.logger.Info("Worker cancelled")

			return

		default:

			// Process next job.

			w.processNextJob(ctx)

		}

	}

}

// Stop stops a worker.

func (w *Worker) Stop(ctx context.Context) {

	if !w.started {

		return

	}

	close(w.stopChan)

	w.started = false

}

// processNextJob processes the next job from the queue.

func (w *Worker) processNextJob(ctx context.Context) {

	// Get next job from queue (blocks until item available).

	item, shutdown := w.workQueue.Get()

	if shutdown {

		return

	}

	defer w.workQueue.Done(item)

	jobID := item

	w.logger.V(1).Info("Processing job", "jobId", jobID)

	// Process the job.

	if err := w.processor(ctx, jobID); err != nil {

		w.logger.Error(err, "Job processing failed", "jobId", jobID)

		// Check if we should retry.

		if w.workQueue.NumRequeues(item) < 5 { // Max 5 retries

			w.workQueue.AddRateLimited(item)

		} else {

			w.logger.Error(err, "Job failed permanently after retries", "jobId", jobID)

			w.workQueue.Forget(item)

		}

		return

	}

	// Job succeeded.

	w.workQueue.Forget(item)

	w.logger.V(1).Info("Job completed successfully", "jobId", jobID)

}

// IntentLockManager handles distributed locking for intent processing.

type IntentLockManager struct {
	client client.Client

	logger logr.Logger

	holderIdentity string

	leaseDuration int32

	renewDeadline int32

	// Active leases.

	leases sync.Map // map[string]*coordinationv1.Lease

}

// NewIntentLockManager creates a new lock manager.

func NewIntentLockManager(client client.Client, logger logr.Logger) *IntentLockManager {

	return &IntentLockManager{

		client: client,

		logger: logger.WithName("lock-manager"),

		holderIdentity: fmt.Sprintf("intent-orchestrator-%d", time.Now().Unix()),

		leaseDuration: 60, // 60 seconds

		renewDeadline: 45, // 45 seconds

	}

}

// AcquireIntentLock acquires a distributed lock for intent processing.

func (l *IntentLockManager) AcquireIntentLock(ctx context.Context, intentID string) (*IntentLock, error) {

	lockName := fmt.Sprintf("intent-lock-%s", intentID)

	// Create lease resource.

	lease := &coordinationv1.Lease{

		ObjectMeta: metav1.ObjectMeta{

			Name: lockName,

			Namespace: "nephoran-system",
		},

		Spec: coordinationv1.LeaseSpec{

			HolderIdentity: &l.holderIdentity,

			LeaseDurationSeconds: &l.leaseDuration,

			RenewTime: &metav1.MicroTime{Time: time.Now()},
		},
	}

	// Try to create or update the lease.

	existingLease := &coordinationv1.Lease{}

	err := l.client.Get(ctx, client.ObjectKeyFromObject(lease), existingLease)

	if err == nil {

		// Lease exists, check if we can acquire it.

		if existingLease.Spec.HolderIdentity != nil && *existingLease.Spec.HolderIdentity != l.holderIdentity {

			// Check if lease has expired.

			if existingLease.Spec.RenewTime != nil {

				deadline := existingLease.Spec.RenewTime.Add(time.Duration(l.leaseDuration) * time.Second)

				if time.Now().Before(deadline) {

					return nil, fmt.Errorf("intent %s is locked by %s", intentID, *existingLease.Spec.HolderIdentity)

				}

			}

		}

		// Update existing lease.

		existingLease.Spec.HolderIdentity = &l.holderIdentity

		existingLease.Spec.RenewTime = &metav1.MicroTime{Time: time.Now()}

		if err := l.client.Update(ctx, existingLease); err != nil {

			return nil, fmt.Errorf("failed to update lease: %w", err)

		}

		lease = existingLease

	} else {

		// Create new lease.

		if err := l.client.Create(ctx, lease); err != nil {

			return nil, fmt.Errorf("failed to create lease: %w", err)

		}

	}

	// Store lease for renewal.

	l.leases.Store(intentID, lease)

	// Create lock object.

	lock := &IntentLock{

		intentID: intentID,

		lockManager: l,

		lease: lease,

		ctx: ctx,

		stopRenewal: make(chan bool),
	}

	// Start renewal goroutine.

	go lock.renewLease()

	l.logger.Info("Acquired intent lock", "intentId", intentID)

	return lock, nil

}

// IntentLock represents a distributed lock on an intent.

type IntentLock struct {
	intentID string

	lockManager *IntentLockManager

	lease *coordinationv1.Lease

	ctx context.Context

	stopRenewal chan bool
}

// Release releases the intent lock.

func (l *IntentLock) Release() error {

	l.lockManager.logger.Info("Releasing intent lock", "intentId", l.intentID)

	// Stop renewal.

	close(l.stopRenewal)

	// Delete lease.

	if err := l.lockManager.client.Delete(l.ctx, l.lease); err != nil {

		l.lockManager.logger.Error(err, "Failed to delete lease", "intentId", l.intentID)

	}

	// Remove from active leases.

	l.lockManager.leases.Delete(l.intentID)

	return nil

}

// renewLease periodically renews the lease.

func (l *IntentLock) renewLease() {

	ticker := time.NewTicker(time.Duration(l.lockManager.renewDeadline) * time.Second)

	defer ticker.Stop()

	for {

		select {

		case <-ticker.C:

			// Renew the lease.

			l.lease.Spec.RenewTime = &metav1.MicroTime{Time: time.Now()}

			if err := l.lockManager.client.Update(l.ctx, l.lease); err != nil {

				l.lockManager.logger.Error(err, "Failed to renew lease", "intentId", l.intentID)

			}

		case <-l.stopRenewal:

			return

		case <-l.ctx.Done():

			return

		}

	}

}

// MetricsCollector collects and manages metrics for the orchestrator.

type MetricsCollector struct {
	phaseMetrics map[interfaces.ProcessingPhase]*PhaseMetrics

	intentMetrics map[string]*IntentMetrics

	overallMetrics *OverallMetrics

	mutex sync.RWMutex
}

// PhaseMetrics tracks metrics for a specific processing phase.

type PhaseMetrics struct {
	Phase interfaces.ProcessingPhase `json:"phase"`

	TotalStarted int64 `json:"totalStarted"`

	TotalCompleted int64 `json:"totalCompleted"`

	TotalFailed int64 `json:"totalFailed"`

	AverageProcessingTime time.Duration `json:"averageProcessingTime"`

	SuccessRate float64 `json:"successRate"`

	// Current state.

	ActiveCount int `json:"activeCount"`

	LastStartTime time.Time `json:"lastStartTime"`

	LastCompletionTime time.Time `json:"lastCompletionTime"`
}

// IntentMetrics tracks metrics for a specific intent.

type IntentMetrics struct {
	IntentID string `json:"intentId"`

	StartTime time.Time `json:"startTime"`

	CompletionTime *time.Time `json:"completionTime,omitempty"`

	CurrentPhase interfaces.ProcessingPhase `json:"currentPhase"`

	CompletedPhases []interfaces.ProcessingPhase `json:"completedPhases"`

	TotalDuration time.Duration `json:"totalDuration"`

	Success bool `json:"success"`
}

// OverallMetrics tracks system-wide metrics.

type OverallMetrics struct {
	TotalIntents int64 `json:"totalIntents"`

	CompletedIntents int64 `json:"completedIntents"`

	FailedIntents int64 `json:"failedIntents"`

	ActiveIntents int64 `json:"activeIntents"`

	AverageProcessingTime time.Duration `json:"averageProcessingTime"`

	SystemThroughput float64 `json:"systemThroughput"` // intents per minute

	StartTime time.Time `json:"startTime"`

	LastUpdate time.Time `json:"lastUpdate"`
}

// NewMetricsCollector creates a new metrics collector.

func NewMetricsCollector() *MetricsCollector {

	return &MetricsCollector{

		phaseMetrics: make(map[interfaces.ProcessingPhase]*PhaseMetrics),

		intentMetrics: make(map[string]*IntentMetrics),

		overallMetrics: &OverallMetrics{

			StartTime: time.Now(),
		},
	}

}

// RecordPhaseStart records the start of a processing phase.

func (m *MetricsCollector) RecordPhaseStart(phase interfaces.ProcessingPhase, intentID string) {

	m.mutex.Lock()

	defer m.mutex.Unlock()

	// Initialize phase metrics if not exists.

	if _, exists := m.phaseMetrics[phase]; !exists {

		m.phaseMetrics[phase] = &PhaseMetrics{

			Phase: phase,
		}

	}

	phaseMetrics := m.phaseMetrics[phase]

	phaseMetrics.TotalStarted++

	phaseMetrics.ActiveCount++

	phaseMetrics.LastStartTime = time.Now()

	// Initialize intent metrics if not exists.

	if _, exists := m.intentMetrics[intentID]; !exists {

		m.intentMetrics[intentID] = &IntentMetrics{

			IntentID: intentID,

			StartTime: time.Now(),

			CompletedPhases: make([]interfaces.ProcessingPhase, 0),
		}

		m.overallMetrics.TotalIntents++

		m.overallMetrics.ActiveIntents++

	}

	intentMetrics := m.intentMetrics[intentID]

	intentMetrics.CurrentPhase = phase

}

// RecordPhaseCompletion records the completion of a processing phase.

func (m *MetricsCollector) RecordPhaseCompletion(phase interfaces.ProcessingPhase, intentID string, success bool) {

	m.mutex.Lock()

	defer m.mutex.Unlock()

	phaseMetrics := m.phaseMetrics[phase]

	if phaseMetrics == nil {

		return

	}

	phaseMetrics.ActiveCount--

	phaseMetrics.LastCompletionTime = time.Now()

	if success {

		phaseMetrics.TotalCompleted++

	} else {

		phaseMetrics.TotalFailed++

	}

	// Calculate success rate.

	if phaseMetrics.TotalStarted > 0 {

		phaseMetrics.SuccessRate = float64(phaseMetrics.TotalCompleted) / float64(phaseMetrics.TotalStarted)

	}

	// Update intent metrics.

	if intentMetrics, exists := m.intentMetrics[intentID]; exists {

		if success {

			intentMetrics.CompletedPhases = append(intentMetrics.CompletedPhases, phase)

		}

	}

}

// RecordIntentCompletion records the completion of an intent.

func (m *MetricsCollector) RecordIntentCompletion(intentID string, success bool, duration time.Duration) {

	m.mutex.Lock()

	defer m.mutex.Unlock()

	if intentMetrics, exists := m.intentMetrics[intentID]; exists {

		now := time.Now()

		intentMetrics.CompletionTime = &now

		intentMetrics.TotalDuration = duration

		intentMetrics.Success = success

		m.overallMetrics.ActiveIntents--

		if success {

			m.overallMetrics.CompletedIntents++

		} else {

			m.overallMetrics.FailedIntents++

		}

		// Update average processing time.

		if m.overallMetrics.CompletedIntents > 0 {

			totalDuration := time.Duration(m.overallMetrics.CompletedIntents) * m.overallMetrics.AverageProcessingTime

			totalDuration += duration

			m.overallMetrics.AverageProcessingTime = totalDuration / time.Duration(m.overallMetrics.CompletedIntents+1)

		} else {

			m.overallMetrics.AverageProcessingTime = duration

		}

		// Calculate throughput (intents per minute).

		elapsed := time.Since(m.overallMetrics.StartTime)

		if elapsed.Minutes() > 0 {

			m.overallMetrics.SystemThroughput = float64(m.overallMetrics.CompletedIntents) / elapsed.Minutes()

		}

		m.overallMetrics.LastUpdate = now

	}

}

// GetPhaseMetrics returns metrics for a specific phase.

func (m *MetricsCollector) GetPhaseMetrics(phase interfaces.ProcessingPhase) *PhaseMetrics {

	m.mutex.RLock()

	defer m.mutex.RUnlock()

	if metrics, exists := m.phaseMetrics[phase]; exists {

		// Return a copy to avoid race conditions.

		metricsCopy := *metrics

		return &metricsCopy

	}

	return nil

}

// GetIntentMetrics returns metrics for a specific intent.

func (m *MetricsCollector) GetIntentMetrics(intentID string) *IntentMetrics {

	m.mutex.RLock()

	defer m.mutex.RUnlock()

	if metrics, exists := m.intentMetrics[intentID]; exists {

		// Return a copy.

		metricsCopy := *metrics

		metricsCopy.CompletedPhases = make([]interfaces.ProcessingPhase, len(metrics.CompletedPhases))

		copy(metricsCopy.CompletedPhases, metrics.CompletedPhases)

		return &metricsCopy

	}

	return nil

}

// GetOverallMetrics returns overall system metrics.

func (m *MetricsCollector) GetOverallMetrics() *OverallMetrics {

	m.mutex.RLock()

	defer m.mutex.RUnlock()

	// Return a copy.

	metricsCopy := *m.overallMetrics

	return &metricsCopy

}

// GetAllMetrics returns all collected metrics.

func (m *MetricsCollector) GetAllMetrics() map[string]interface{} {

	m.mutex.RLock()

	defer m.mutex.RUnlock()

	return map[string]interface{}{

		"phases": m.phaseMetrics,

		"intents": m.intentMetrics,

		"overall": m.overallMetrics,
	}

}

// Supporting data structures.

// WorkerPoolStatus represents the status of a worker pool.

type WorkerPoolStatus struct {
	ID string `json:"id"`

	WorkerCount int `json:"workerCount"`

	Started bool `json:"started"`

	QueueLength int `json:"queueLength"`

	Workers []WorkerStatus `json:"workers"`
}

// WorkerStatus represents the status of a single worker.

type WorkerStatus struct {
	ID string `json:"id"`

	Started bool `json:"started"`
}
