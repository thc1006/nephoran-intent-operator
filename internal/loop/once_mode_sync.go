package loop

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// OnceModeSynchronizer provides synchronization for "once" mode operations.
type OnceModeSynchronizer struct {
	watcher        *Watcher
	expectedFiles  int
	processedCount int64
	failedCount    int64

	// Synchronization channels.
	startedChan  chan struct{}
	completeChan chan struct{}

	// State tracking.
	started   bool
	completed bool
	mu        sync.RWMutex

	// Context for cancellation.
	ctx    context.Context
	cancel context.CancelFunc
}

// NewOnceModeSynchronizer creates a new synchronizer for once mode.
func NewOnceModeSynchronizer(watcher *Watcher, expectedFiles int) *OnceModeSynchronizer {
	ctx, cancel := context.WithCancel(context.Background())

	return &OnceModeSynchronizer{
		watcher:       watcher,
		expectedFiles: expectedFiles,
		startedChan:   make(chan struct{}),
		completeChan:  make(chan struct{}),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// StartWithCompletion starts the watcher and waits for completion.
func (oms *OnceModeSynchronizer) StartWithCompletion(timeout time.Duration) error {
	// Start monitoring before starting watcher.
	go oms.monitorProgress()

	// Start watcher in background.
	errChan := make(chan error, 1)
	go func() {
		defer close(oms.startedChan)
		errChan <- oms.watcher.Start()
	}()

	// Wait for watcher to start processing.
	select {
	case <-oms.startedChan:
		// Watcher started.
	case <-time.After(1 * time.Second):
		oms.cancel()
		return <-errChan // Return the error from Start()
	}

	// Wait for completion or timeout.
	select {
	case <-oms.completeChan:
		oms.cancel()     // Signal completion
		return <-errChan // Get final result
	case <-time.After(timeout):
		oms.cancel()
		return <-errChan // Return error or nil
	case err := <-errChan:
		// Watcher completed before we detected completion.
		oms.markCompleted()
		return err
	}
}

// monitorProgress monitors processing progress and signals completion.
func (oms *OnceModeSynchronizer) monitorProgress() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-oms.ctx.Done():
			return
		case <-ticker.C:
			if oms.checkCompletion() {
				oms.markCompleted()
				return
			}
		}
	}
}

// checkCompletion checks if processing is complete.
func (oms *OnceModeSynchronizer) checkCompletion() bool {
	// Check if we have processed expected number of files.
	processed := atomic.LoadInt64(&oms.processedCount)
	failed := atomic.LoadInt64(&oms.failedCount)
	total := processed + failed

	if int(total) >= oms.expectedFiles {
		return true
	}

	// Also check watcher's internal stats if available.
	if oms.watcher.executor != nil {
		stats := oms.watcher.executor.GetStats()
		if stats.TotalExecutions >= oms.expectedFiles {
			return true
		}
	}

	return false
}

// markCompleted marks processing as completed.
func (oms *OnceModeSynchronizer) markCompleted() {
	oms.mu.Lock()
	defer oms.mu.Unlock()

	if !oms.completed {
		oms.completed = true
		close(oms.completeChan)
	}
}

// NotifyProcessed notifies that a file was processed successfully.
func (oms *OnceModeSynchronizer) NotifyProcessed() {
	atomic.AddInt64(&oms.processedCount, 1)
}

// NotifyFailed notifies that a file processing failed.
func (oms *OnceModeSynchronizer) NotifyFailed() {
	atomic.AddInt64(&oms.failedCount, 1)
}

// GetStats returns current processing statistics.
func (oms *OnceModeSynchronizer) GetStats() (processed, failed int) {
	return int(atomic.LoadInt64(&oms.processedCount)), int(atomic.LoadInt64(&oms.failedCount))
}

// FileCreationSynchronizer ensures files exist before watcher starts.
type FileCreationSynchronizer struct {
	directory     string
	expectedFiles map[string]bool
	createdFiles  map[string]bool
	mu            sync.RWMutex
	allCreated    chan struct{}
	timeout       time.Duration
}

// NewFileCreationSynchronizer creates a new file creation synchronizer.
func NewFileCreationSynchronizer(directory string, expectedFiles []string, timeout time.Duration) *FileCreationSynchronizer {
	expected := make(map[string]bool)
	for _, file := range expectedFiles {
		expected[file] = true
	}

	return &FileCreationSynchronizer{
		directory:     directory,
		expectedFiles: expected,
		createdFiles:  make(map[string]bool),
		allCreated:    make(chan struct{}),
		timeout:       timeout,
	}
}

// NotifyFileCreated notifies that a file was created.
func (fcs *FileCreationSynchronizer) NotifyFileCreated(filename string) {
	fcs.mu.Lock()
	defer fcs.mu.Unlock()

	if fcs.expectedFiles[filename] {
		fcs.createdFiles[filename] = true

		// Check if all files are created.
		if len(fcs.createdFiles) == len(fcs.expectedFiles) {
			select {
			case <-fcs.allCreated:
				// Already closed.
			default:
				close(fcs.allCreated)
			}
		}
	}
}

// WaitForAllFiles waits for all expected files to be created.
func (fcs *FileCreationSynchronizer) WaitForAllFiles() error {
	select {
	case <-fcs.allCreated:
		return nil
	case <-time.After(fcs.timeout):
		fcs.mu.RLock()
		created := len(fcs.createdFiles)
		expected := len(fcs.expectedFiles)
		fcs.mu.RUnlock()

		return &SynchronizationTimeoutError{
			Operation: "file creation",
			Expected:  expected,
			Actual:    created,
			Timeout:   fcs.timeout,
		}
	}
}

// ProcessingCompletionWaiter waits for processing to complete in once mode.
type ProcessingCompletionWaiter struct {
	watcher       *Watcher
	expectedFiles int
	checkInterval time.Duration
	maxWaitTime   time.Duration
	startTime     time.Time
}

// NewProcessingCompletionWaiter creates a new completion waiter.
func NewProcessingCompletionWaiter(watcher *Watcher, expectedFiles int) *ProcessingCompletionWaiter {
	return &ProcessingCompletionWaiter{
		watcher:       watcher,
		expectedFiles: expectedFiles,
		checkInterval: 100 * time.Millisecond,
		maxWaitTime:   30 * time.Second,
		startTime:     time.Now(),
	}
}

// WaitForCompletion waits for all processing to complete.
func (pcw *ProcessingCompletionWaiter) WaitForCompletion() error {
	ticker := time.NewTicker(pcw.checkInterval)
	defer ticker.Stop()

	timeout := time.After(pcw.maxWaitTime)

	for {
		select {
		case <-timeout:
			return &SynchronizationTimeoutError{
				Operation: "processing completion",
				Expected:  pcw.expectedFiles,
				Actual:    pcw.getCurrentProcessedCount(),
				Timeout:   pcw.maxWaitTime,
			}

		case <-ticker.C:
			if pcw.isProcessingComplete() {
				// Give additional time for cleanup.
				time.Sleep(200 * time.Millisecond)
				return nil
			}
		}
	}
}

// isProcessingComplete checks if processing is complete.
func (pcw *ProcessingCompletionWaiter) isProcessingComplete() bool {
	if pcw.watcher.executor != nil {
		stats := pcw.watcher.executor.GetStats()
		return stats.TotalExecutions >= pcw.expectedFiles
	}
	return false
}

// getCurrentProcessedCount gets current processed count.
func (pcw *ProcessingCompletionWaiter) getCurrentProcessedCount() int {
	if pcw.watcher.executor != nil {
		stats := pcw.watcher.executor.GetStats()
		return stats.TotalExecutions
	}
	return 0
}

// SynchronizationTimeoutError represents a synchronization timeout.
type SynchronizationTimeoutError struct {
	Operation string
	Expected  int
	Actual    int
	Timeout   time.Duration
}

// Error implements the error interface.
func (e *SynchronizationTimeoutError) Error() string {
	return fmt.Sprintf("synchronization timeout for %s: expected %d, got %d after %v",
		e.Operation, e.Expected, e.Actual, e.Timeout)
}

// CrossPlatformSyncBarrier provides cross-platform synchronization.
type CrossPlatformSyncBarrier struct {
	participants int
	waiting      int
	generation   int
	mu           sync.Mutex
	cond         *sync.Cond
}

// NewCrossPlatformSyncBarrier creates a new sync barrier.
func NewCrossPlatformSyncBarrier(participants int) *CrossPlatformSyncBarrier {
	barrier := &CrossPlatformSyncBarrier{
		participants: participants,
	}
	barrier.cond = sync.NewCond(&barrier.mu)
	return barrier
}

// Wait waits for all participants to reach the barrier.
func (b *CrossPlatformSyncBarrier) Wait() {
	b.mu.Lock()
	defer b.mu.Unlock()

	generation := b.generation
	b.waiting++

	if b.waiting == b.participants {
		// Last participant - wake everyone up.
		b.waiting = 0
		b.generation++
		b.cond.Broadcast()
	} else {
		// Wait for other participants.
		for generation == b.generation {
			b.cond.Wait()
		}
	}
}

// Enhanced watcher state for once mode.
type EnhancedOnceState struct {
	filesScanned      atomic.Bool
	processingStarted atomic.Bool
	processingDone    atomic.Bool

	scannedFiles   int64
	processedFiles int64
	failedFiles    int64

	startTime           time.Time
	scanCompleteTime    time.Time
	processCompleteTime time.Time
}

// NewEnhancedOnceState creates new enhanced state tracking.
func NewEnhancedOnceState() *EnhancedOnceState {
	return &EnhancedOnceState{
		startTime: time.Now(),
	}
}

// MarkScanComplete marks file scanning as complete.
func (eos *EnhancedOnceState) MarkScanComplete(fileCount int) {
	eos.filesScanned.Store(true)
	atomic.StoreInt64(&eos.scannedFiles, int64(fileCount))
	eos.scanCompleteTime = time.Now()
}

// MarkProcessingStarted marks processing as started.
func (eos *EnhancedOnceState) MarkProcessingStarted() {
	eos.processingStarted.Store(true)
}

// MarkProcessingDone marks processing as complete.
func (eos *EnhancedOnceState) MarkProcessingDone() {
	eos.processingDone.Store(true)
	eos.processCompleteTime = time.Now()
}

// IncrementProcessed increments processed file count.
func (eos *EnhancedOnceState) IncrementProcessed() {
	atomic.AddInt64(&eos.processedFiles, 1)
}

// IncrementFailed increments failed file count.
func (eos *EnhancedOnceState) IncrementFailed() {
	atomic.AddInt64(&eos.failedFiles, 1)
}

// GetStats returns current state statistics.
func (eos *EnhancedOnceState) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"files_scanned":      atomic.LoadInt64(&eos.scannedFiles),
		"files_processed":    atomic.LoadInt64(&eos.processedFiles),
		"files_failed":       atomic.LoadInt64(&eos.failedFiles),
		"scan_complete":      eos.filesScanned.Load(),
		"processing_started": eos.processingStarted.Load(),
		"processing_done":    eos.processingDone.Load(),
		"scan_duration":      eos.scanCompleteTime.Sub(eos.startTime),
		"process_duration":   eos.processCompleteTime.Sub(eos.scanCompleteTime),
		"total_duration":     time.Since(eos.startTime),
	}
}

// IsComplete returns true if processing is complete.
func (eos *EnhancedOnceState) IsComplete() bool {
	if !eos.filesScanned.Load() || !eos.processingStarted.Load() {
		return false
	}

	scanned := atomic.LoadInt64(&eos.scannedFiles)
	processed := atomic.LoadInt64(&eos.processedFiles)
	failed := atomic.LoadInt64(&eos.failedFiles)

	return (processed + failed) >= scanned
}
