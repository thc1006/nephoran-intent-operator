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

	"sync"

	"time"

)



// StateMetrics provides metrics for state management operations.

type StateMetrics struct {

	mutex sync.RWMutex



	// Cache metrics.

	cacheHits     int64

	cacheMisses   int64

	cacheSize     int64

	cacheCleanups int64



	// State operation metrics.

	stateUpdates        int64

	stateSyncs          int64

	validationErrors    int64

	conflictResolutions int64



	// Lock metrics.

	activeLocks      int64

	lockAcquisitions int64

	lockReleases     int64

	lockTimeouts     int64



	// Timing metrics.

	updateTimes     []time.Duration

	syncTimes       []time.Duration

	validationTimes []time.Duration

	lastSyncTime    time.Time



	// Error tracking.

	errors          []MetricError

	maxErrorHistory int



	// Performance tracking.

	throughput *ThroughputTracker

	latency    *LatencyTracker



	// Resource usage.

	memoryUsage int64

	cpuUsage    float64

	diskUsage   int64

}



// MetricError represents an error in metrics.

type MetricError struct {

	Timestamp time.Time `json:"timestamp"`

	Operation string    `json:"operation"`

	Error     string    `json:"error"`

	Context   string    `json:"context,omitempty"`

	Severity  string    `json:"severity"`

}



// ThroughputTracker tracks throughput metrics.

type ThroughputTracker struct {

	mutex             sync.RWMutex

	operations        []time.Time

	windowSize        time.Duration

	currentThroughput float64

}



// LatencyTracker tracks latency metrics.

type LatencyTracker struct {

	mutex      sync.RWMutex

	latencies  []time.Duration

	maxSamples int

	currentP50 time.Duration

	currentP95 time.Duration

	currentP99 time.Duration

}



// NewStateMetrics creates a new state metrics collector.

func NewStateMetrics() *StateMetrics {

	return &StateMetrics{

		updateTimes:     make([]time.Duration, 0, 1000),

		syncTimes:       make([]time.Duration, 0, 100),

		validationTimes: make([]time.Duration, 0, 1000),

		errors:          make([]MetricError, 0, 100),

		maxErrorHistory: 100,

		throughput:      NewThroughputTracker(1 * time.Minute),

		latency:         NewLatencyTracker(1000),

	}

}



// NewThroughputTracker creates a new throughput tracker.

func NewThroughputTracker(windowSize time.Duration) *ThroughputTracker {

	return &ThroughputTracker{

		operations: make([]time.Time, 0, 1000),

		windowSize: windowSize,

	}

}



// NewLatencyTracker creates a new latency tracker.

func NewLatencyTracker(maxSamples int) *LatencyTracker {

	return &LatencyTracker{

		latencies:  make([]time.Duration, 0, maxSamples),

		maxSamples: maxSamples,

	}

}



// Cache metrics methods.



// RecordCacheHit performs recordcachehit operation.

func (sm *StateMetrics) RecordCacheHit() {

	sm.mutex.Lock()

	defer sm.mutex.Unlock()



	sm.cacheHits++

	sm.throughput.RecordOperation()

}



// RecordCacheMiss performs recordcachemiss operation.

func (sm *StateMetrics) RecordCacheMiss() {

	sm.mutex.Lock()

	defer sm.mutex.Unlock()



	sm.cacheMisses++

}



// RecordCacheSize performs recordcachesize operation.

func (sm *StateMetrics) RecordCacheSize(size int64) {

	sm.mutex.Lock()

	defer sm.mutex.Unlock()



	sm.cacheSize = size

}



// RecordCacheCleanup performs recordcachecleanup operation.

func (sm *StateMetrics) RecordCacheCleanup() {

	sm.mutex.Lock()

	defer sm.mutex.Unlock()



	sm.cacheCleanups++

}



// GetCacheHitRate performs getcachehitrate operation.

func (sm *StateMetrics) GetCacheHitRate() float64 {

	sm.mutex.RLock()

	defer sm.mutex.RUnlock()



	total := sm.cacheHits + sm.cacheMisses

	if total == 0 {

		return 0.0

	}



	return float64(sm.cacheHits) / float64(total)

}



// State operation metrics methods.



// RecordStateUpdate performs recordstateupdate operation.

func (sm *StateMetrics) RecordStateUpdate() {

	sm.mutex.Lock()

	defer sm.mutex.Unlock()



	sm.stateUpdates++

	sm.throughput.RecordOperation()

}



// RecordStateUpdateTime performs recordstateupdatetime operation.

func (sm *StateMetrics) RecordStateUpdateTime(duration time.Duration) {

	sm.mutex.Lock()

	defer sm.mutex.Unlock()



	sm.updateTimes = append(sm.updateTimes, duration)

	sm.latency.RecordLatency(duration)



	// Keep only recent times.

	if len(sm.updateTimes) > 1000 {

		sm.updateTimes = sm.updateTimes[100:]

	}

}



// RecordStateSync performs recordstatesync operation.

func (sm *StateMetrics) RecordStateSync() {

	sm.mutex.Lock()

	defer sm.mutex.Unlock()



	sm.stateSyncs++

	sm.lastSyncTime = time.Now()

}



// RecordStateSyncTime performs recordstatesynctime operation.

func (sm *StateMetrics) RecordStateSyncTime(duration time.Duration) {

	sm.mutex.Lock()

	defer sm.mutex.Unlock()



	sm.syncTimes = append(sm.syncTimes, duration)



	// Keep only recent times.

	if len(sm.syncTimes) > 100 {

		sm.syncTimes = sm.syncTimes[10:]

	}

}



// RecordValidationError performs recordvalidationerror operation.

func (sm *StateMetrics) RecordValidationError() {

	sm.mutex.Lock()

	defer sm.mutex.Unlock()



	sm.validationErrors++

}



// RecordConflictResolution performs recordconflictresolution operation.

func (sm *StateMetrics) RecordConflictResolution() {

	sm.mutex.Lock()

	defer sm.mutex.Unlock()



	sm.conflictResolutions++

}



// GetAverageUpdateTime performs getaverageupdatetime operation.

func (sm *StateMetrics) GetAverageUpdateTime() time.Duration {

	sm.mutex.RLock()

	defer sm.mutex.RUnlock()



	if len(sm.updateTimes) == 0 {

		return 0

	}



	var total time.Duration

	for _, duration := range sm.updateTimes {

		total += duration

	}



	return total / time.Duration(len(sm.updateTimes))

}



// GetValidationErrors performs getvalidationerrors operation.

func (sm *StateMetrics) GetValidationErrors() int64 {

	sm.mutex.RLock()

	defer sm.mutex.RUnlock()



	return sm.validationErrors

}



// GetLastSyncTime performs getlastsynctime operation.

func (sm *StateMetrics) GetLastSyncTime() time.Time {

	sm.mutex.RLock()

	defer sm.mutex.RUnlock()



	return sm.lastSyncTime

}



// Lock metrics methods.



// RecordActiveLocks performs recordactivelocks operation.

func (sm *StateMetrics) RecordActiveLocks(count int) {

	sm.mutex.Lock()

	defer sm.mutex.Unlock()



	sm.activeLocks = int64(count)

}



// RecordLockAcquisition performs recordlockacquisition operation.

func (sm *StateMetrics) RecordLockAcquisition() {

	sm.mutex.Lock()

	defer sm.mutex.Unlock()



	sm.lockAcquisitions++

}



// RecordLockRelease performs recordlockrelease operation.

func (sm *StateMetrics) RecordLockRelease() {

	sm.mutex.Lock()

	defer sm.mutex.Unlock()



	sm.lockReleases++

}



// RecordLockTimeout performs recordlocktimeout operation.

func (sm *StateMetrics) RecordLockTimeout() {

	sm.mutex.Lock()

	defer sm.mutex.Unlock()



	sm.lockTimeouts++

}



// Error tracking methods.



// RecordError performs recorderror operation.

func (sm *StateMetrics) RecordError(operation, errorMsg, context, severity string) {

	sm.mutex.Lock()

	defer sm.mutex.Unlock()



	error := MetricError{

		Timestamp: time.Now(),

		Operation: operation,

		Error:     errorMsg,

		Context:   context,

		Severity:  severity,

	}



	sm.errors = append(sm.errors, error)



	// Keep only recent errors.

	if len(sm.errors) > sm.maxErrorHistory {

		sm.errors = sm.errors[sm.maxErrorHistory/10:]

	}

}



// GetRecentErrors performs getrecenterrors operation.

func (sm *StateMetrics) GetRecentErrors(count int) []MetricError {

	sm.mutex.RLock()

	defer sm.mutex.RUnlock()



	if count > len(sm.errors) {

		count = len(sm.errors)

	}



	if count == 0 {

		return []MetricError{}

	}



	// Return most recent errors.

	start := len(sm.errors) - count

	result := make([]MetricError, count)

	copy(result, sm.errors[start:])



	return result

}



// Resource usage methods.



// RecordMemoryUsage performs recordmemoryusage operation.

func (sm *StateMetrics) RecordMemoryUsage(bytes int64) {

	sm.mutex.Lock()

	defer sm.mutex.Unlock()



	sm.memoryUsage = bytes

}



// RecordCPUUsage performs recordcpuusage operation.

func (sm *StateMetrics) RecordCPUUsage(percent float64) {

	sm.mutex.Lock()

	defer sm.mutex.Unlock()



	sm.cpuUsage = percent

}



// RecordDiskUsage performs recorddiskusage operation.

func (sm *StateMetrics) RecordDiskUsage(bytes int64) {

	sm.mutex.Lock()

	defer sm.mutex.Unlock()



	sm.diskUsage = bytes

}



// Comprehensive metrics retrieval.



// GetMetrics performs getmetrics operation.

func (sm *StateMetrics) GetMetrics() *StateMetricsSnapshot {

	sm.mutex.RLock()

	defer sm.mutex.RUnlock()



	return &StateMetricsSnapshot{

		Timestamp: time.Now(),



		// Cache metrics.

		CacheHits:     sm.cacheHits,

		CacheMisses:   sm.cacheMisses,

		CacheHitRate:  sm.GetCacheHitRate(),

		CacheSize:     sm.cacheSize,

		CacheCleanups: sm.cacheCleanups,



		// State operation metrics.

		StateUpdates:        sm.stateUpdates,

		StateSyncs:          sm.stateSyncs,

		ValidationErrors:    sm.validationErrors,

		ConflictResolutions: sm.conflictResolutions,



		// Lock metrics.

		ActiveLocks:      sm.activeLocks,

		LockAcquisitions: sm.lockAcquisitions,

		LockReleases:     sm.lockReleases,

		LockTimeouts:     sm.lockTimeouts,



		// Timing metrics.

		AverageUpdateTime: sm.GetAverageUpdateTime(),

		LastSyncTime:      sm.lastSyncTime,



		// Performance metrics.

		Throughput: sm.throughput.GetCurrentThroughput(),

		LatencyP50: sm.latency.GetP50(),

		LatencyP95: sm.latency.GetP95(),

		LatencyP99: sm.latency.GetP99(),



		// Resource usage.

		MemoryUsage: sm.memoryUsage,

		CPUUsage:    sm.cpuUsage,

		DiskUsage:   sm.diskUsage,



		// Error count.

		RecentErrorCount: int64(len(sm.errors)),

	}

}



// StateMetricsSnapshot represents a point-in-time snapshot of metrics.

type StateMetricsSnapshot struct {

	Timestamp time.Time `json:"timestamp"`



	// Cache metrics.

	CacheHits     int64   `json:"cacheHits"`

	CacheMisses   int64   `json:"cacheMisses"`

	CacheHitRate  float64 `json:"cacheHitRate"`

	CacheSize     int64   `json:"cacheSize"`

	CacheCleanups int64   `json:"cacheCleanups"`



	// State operation metrics.

	StateUpdates        int64 `json:"stateUpdates"`

	StateSyncs          int64 `json:"stateSyncs"`

	ValidationErrors    int64 `json:"validationErrors"`

	ConflictResolutions int64 `json:"conflictResolutions"`



	// Lock metrics.

	ActiveLocks      int64 `json:"activeLocks"`

	LockAcquisitions int64 `json:"lockAcquisitions"`

	LockReleases     int64 `json:"lockReleases"`

	LockTimeouts     int64 `json:"lockTimeouts"`



	// Timing metrics.

	AverageUpdateTime time.Duration `json:"averageUpdateTime"`

	LastSyncTime      time.Time     `json:"lastSyncTime"`



	// Performance metrics.

	Throughput float64       `json:"throughput"` // Operations per second

	LatencyP50 time.Duration `json:"latencyP50"`

	LatencyP95 time.Duration `json:"latencyP95"`

	LatencyP99 time.Duration `json:"latencyP99"`



	// Resource usage.

	MemoryUsage int64   `json:"memoryUsage"`

	CPUUsage    float64 `json:"cpuUsage"`

	DiskUsage   int64   `json:"diskUsage"`



	// Error metrics.

	RecentErrorCount int64 `json:"recentErrorCount"`

}



// ThroughputTracker methods.



// RecordOperation performs recordoperation operation.

func (tt *ThroughputTracker) RecordOperation() {

	tt.mutex.Lock()

	defer tt.mutex.Unlock()



	now := time.Now()

	tt.operations = append(tt.operations, now)



	// Remove old operations outside the window.

	cutoff := now.Add(-tt.windowSize)

	for i, op := range tt.operations {

		if op.After(cutoff) {

			tt.operations = tt.operations[i:]

			break

		}

	}



	// Calculate current throughput.

	if len(tt.operations) > 1 {

		elapsed := tt.operations[len(tt.operations)-1].Sub(tt.operations[0])

		if elapsed > 0 {

			tt.currentThroughput = float64(len(tt.operations)) / elapsed.Seconds()

		}

	}

}



// GetCurrentThroughput performs getcurrentthroughput operation.

func (tt *ThroughputTracker) GetCurrentThroughput() float64 {

	tt.mutex.RLock()

	defer tt.mutex.RUnlock()



	return tt.currentThroughput

}



// LatencyTracker methods.



// RecordLatency performs recordlatency operation.

func (lt *LatencyTracker) RecordLatency(latency time.Duration) {

	lt.mutex.Lock()

	defer lt.mutex.Unlock()



	lt.latencies = append(lt.latencies, latency)



	// Keep only recent samples.

	if len(lt.latencies) > lt.maxSamples {

		lt.latencies = lt.latencies[lt.maxSamples/10:]

	}



	// Update percentiles.

	lt.updatePercentiles()

}



func (lt *LatencyTracker) updatePercentiles() {

	if len(lt.latencies) == 0 {

		return

	}



	// Simple percentile calculation (in production, use a more efficient algorithm).

	sorted := make([]time.Duration, len(lt.latencies))

	copy(sorted, lt.latencies)



	// Simple bubble sort (replace with quicksort for large datasets).

	for i := range len(sorted) - 1 {

		for j := range len(sorted) - i - 1 {

			if sorted[j] > sorted[j+1] {

				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]

			}

		}

	}



	n := len(sorted)

	lt.currentP50 = sorted[n*50/100]

	lt.currentP95 = sorted[n*95/100]

	lt.currentP99 = sorted[n*99/100]

}



// GetP50 performs getp50 operation.

func (lt *LatencyTracker) GetP50() time.Duration {

	lt.mutex.RLock()

	defer lt.mutex.RUnlock()



	return lt.currentP50

}



// GetP95 performs getp95 operation.

func (lt *LatencyTracker) GetP95() time.Duration {

	lt.mutex.RLock()

	defer lt.mutex.RUnlock()



	return lt.currentP95

}



// GetP99 performs getp99 operation.

func (lt *LatencyTracker) GetP99() time.Duration {

	lt.mutex.RLock()

	defer lt.mutex.RUnlock()



	return lt.currentP99

}

