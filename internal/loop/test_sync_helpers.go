package loop

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/thc1006/nephoran-intent-operator/internal/porch"
)

// TestSyncHelper provides comprehensive test synchronization primitives.

type TestSyncHelper struct {
	t testing.TB

	tempDir string

	handoffDir string

	outDir string

	// Synchronization primitives.

	filesCreated sync.WaitGroup

	filesProcessed sync.WaitGroup

	// State tracking.

	createdFiles map[string]bool

	processedFiles map[string]bool

	mu sync.RWMutex

	// Cross-platform timing.

	baseDelay time.Duration

	fileSettle time.Duration

	processWait time.Duration
}

// NewTestSyncHelper creates a new test synchronization helper.

func NewTestSyncHelper(t testing.TB) *TestSyncHelper {
	tempDir := t.TempDir()

	handoffDir := filepath.Join(tempDir, "handoff")

	outDir := filepath.Join(tempDir, "out")

	require.NoError(t, os.MkdirAll(handoffDir, 0o755))

	require.NoError(t, os.MkdirAll(outDir, 0o755))

	// Cross-platform timing adjustments.

	baseDelay := 50 * time.Millisecond

	fileSettle := 100 * time.Millisecond

	processWait := 2 * time.Second

	if runtime.GOOS == "windows" {

		// Windows needs more time for file operations.

		baseDelay = 100 * time.Millisecond

		fileSettle = 200 * time.Millisecond

		processWait = 3 * time.Second

	}

	return &TestSyncHelper{
		t: t,

		tempDir: tempDir,

		handoffDir: handoffDir,

		outDir: outDir,

		createdFiles: make(map[string]bool),

		processedFiles: make(map[string]bool),

		baseDelay: baseDelay,

		fileSettle: fileSettle,

		processWait: processWait,
	}
}

// CreateIntentFile creates an intent file with proper naming and waits for file system to settle.

func (h *TestSyncHelper) CreateIntentFile(filename, content string) string {
	// Ensure filename follows intent-*.json pattern.

	if !IsIntentFile(filename) {

		// Transform filename to follow pattern.

		base := filepath.Base(filename)

		if ext := filepath.Ext(base); ext != ".json" {
			base = base + ".json"
		}

		if !IsIntentFile(base) {
			base = "intent-" + base
		}

		filename = base

	}

	filePath := filepath.Join(h.handoffDir, filename)

	h.mu.Lock()

	h.createdFiles[filename] = true

	h.mu.Unlock()

	// Create file atomically.

	tempPath := filePath + ".tmp"

	require.NoError(h.t, os.WriteFile(tempPath, []byte(content), 0o644))

	require.NoError(h.t, os.Rename(tempPath, filePath))

	// Wait for file system to settle.

	time.Sleep(h.fileSettle)

	// Verify file exists and is readable.

	_, err := os.ReadFile(filePath)

	require.NoError(h.t, err, "Created file should be readable")

	return filePath
}

// CreateMultipleIntentFiles creates multiple intent files with proper synchronization.

func (h *TestSyncHelper) CreateMultipleIntentFiles(count int, contentTemplate string) []string {
	var files []string

	// Pre-increment WaitGroup.

	h.filesCreated.Add(count)

	for i := range count {

		filename := fmt.Sprintf("intent-test-%d.json", i)

		content := fmt.Sprintf(contentTemplate, i, i+1) // Allows template substitution

		go func(idx int, fname, cont string) {
			defer h.filesCreated.Done()

			time.Sleep(time.Duration(idx) * h.baseDelay) // Stagger creation

			h.CreateIntentFile(fname, cont)
		}(i, filename, content)

		files = append(files, filepath.Join(h.handoffDir, filename))

	}

	return files
}

// WaitForFilesCreated waits for all files to be created.

func (h *TestSyncHelper) WaitForFilesCreated(timeout time.Duration) bool {
	done := make(chan struct{})

	go func() {
		h.filesCreated.Wait()

		close(done)
	}()

	select {

	case <-done:

		return true

	case <-time.After(timeout):

		return false

	}
}

// StartWatcherWithSync starts a watcher with proper synchronization for testing.

func (h *TestSyncHelper) StartWatcherWithSync(config Config) (*Watcher, error) {
	// Ensure files exist BEFORE creating watcher.

	if !h.WaitForFilesCreated(5 * time.Second) {
		return nil, fmt.Errorf("timeout waiting for files to be created")
	}

	// Additional settling time.

	time.Sleep(h.fileSettle)

	// Create watcher.

	watcher, err := NewWatcher(h.handoffDir, config)
	if err != nil {
		return nil, err
	}

	return watcher, nil
}

// ProcessingTracker tracks file processing completion.

type ProcessingTracker struct {
	expectedFiles int

	processedCount int64

	completeChan chan struct{}

	timeout time.Duration

	mu sync.Mutex
}

// NewProcessingTracker creates a new processing tracker.

func (h *TestSyncHelper) NewProcessingTracker(expectedFiles int) *ProcessingTracker {
	return &ProcessingTracker{
		expectedFiles: expectedFiles,

		completeChan: make(chan struct{}),

		timeout: h.processWait,
	}
}

// MarkProcessed marks a file as processed.

func (pt *ProcessingTracker) MarkProcessed() {
	count := atomic.AddInt64(&pt.processedCount, 1)

	if int(count) >= pt.expectedFiles {

		pt.mu.Lock()

		select {

		case <-pt.completeChan:

			// Already closed.

		default:

			close(pt.completeChan)

		}

		pt.mu.Unlock()

	}
}

// WaitForCompletion waits for all files to be processed.

func (pt *ProcessingTracker) WaitForCompletion() bool {
	select {

	case <-pt.completeChan:

		return true

	case <-time.After(pt.timeout):

		return false

	}
}

// GetProcessedCount returns the current processed count.

func (pt *ProcessingTracker) GetProcessedCount() int {
	return int(atomic.LoadInt64(&pt.processedCount))
}

// EnhancedOnceWatcher wraps watcher with completion tracking for once mode.

type EnhancedOnceWatcher struct {
	*Watcher

	tracker *ProcessingTracker

	done chan error

	started chan struct{}

	processing chan struct{}
}

// NewEnhancedOnceWatcher creates a watcher with completion tracking.

func (h *TestSyncHelper) NewEnhancedOnceWatcher(config Config, expectedFiles int) (*EnhancedOnceWatcher, error) {
	watcher, err := h.StartWatcherWithSync(config)
	if err != nil {
		return nil, err
	}

	tracker := h.NewProcessingTracker(expectedFiles)

	enhanced := &EnhancedOnceWatcher{
		Watcher: watcher,

		tracker: tracker,

		done: make(chan error, 1),

		started: make(chan struct{}),

		processing: make(chan struct{}),
	}

	return enhanced, nil
}

// StartWithTracking starts the watcher with completion tracking.

func (ew *EnhancedOnceWatcher) StartWithTracking() error {
	go func() {
		close(ew.started)

		// Signal that processing has begun.

		close(ew.processing)

		// Start the actual watcher.

		err := ew.Watcher.Start()

		ew.done <- err
	}()

	// Wait for watcher to start.

	<-ew.started

	// Brief delay to ensure watcher is fully initialized.

	time.Sleep(50 * time.Millisecond)

	return nil
}

// WaitForProcessingComplete waits for processing to complete in once mode.

func (ew *EnhancedOnceWatcher) WaitForProcessingComplete(timeout time.Duration) error {
	// Wait for processing to start.

	<-ew.processing

	// Wait for all files to be processed OR watcher to complete.

	processingDone := make(chan bool, 1)

	go func() {
		processingDone <- ew.tracker.WaitForCompletion()
	}()

	watcherDone := make(chan error, 1)

	go func() {
		select {

		case err := <-ew.done:

			watcherDone <- err

		case <-time.After(timeout):

			watcherDone <- fmt.Errorf("watcher timeout")

		}
	}()

	// Wait for either processing to complete or watcher to finish.

	select {

	case completed := <-processingDone:

		if !completed {
			return fmt.Errorf("processing timeout: expected %d files, processed %d",

				ew.tracker.expectedFiles, ew.tracker.GetProcessedCount())
		}

		// Give a bit more time for watcher to finish cleanup.

		select {

		case err := <-ew.done:

			return err

		case <-time.After(500 * time.Millisecond):

			return nil // Processing completed, watcher may still be finishing

		}

	case err := <-watcherDone:

		return err

	}
}

// FileSystemSyncGuard ensures files are properly synced before testing.

type FileSystemSyncGuard struct {
	dir string

	t testing.TB
}

// NewFileSystemSyncGuard creates a new sync guard.

func (h *TestSyncHelper) NewFileSystemSyncGuard() *FileSystemSyncGuard {
	return &FileSystemSyncGuard{
		dir: h.handoffDir,

		t: h.t,
	}
}

// EnsureFilesVisible ensures all files are visible to the file system watcher.

func (fsg *FileSystemSyncGuard) EnsureFilesVisible(expectedFiles []string) error {
	maxRetries := 10

	retryDelay := 50 * time.Millisecond

	for range maxRetries {

		allVisible := true

		for _, expectedFile := range expectedFiles {
			if _, err := os.Stat(expectedFile); os.IsNotExist(err) {

				allVisible = false

				break

			}
		}

		if allVisible {

			// Additional sync for directory listing.

			entries, err := os.ReadDir(fsg.dir)
			if err != nil {
				return fmt.Errorf("failed to read directory: %w", err)
			}

			visibleCount := 0

			for _, entry := range entries {
				if IsIntentFile(entry.Name()) {
					visibleCount++
				}
			}

			if visibleCount >= len(expectedFiles) {

				// All files are visible.

				time.Sleep(retryDelay) // Final settling time

				return nil

			}

		}

		time.Sleep(retryDelay)

		retryDelay = time.Duration(float64(retryDelay) * 1.2) // Exponential backoff

	}

	return fmt.Errorf("timeout: not all files became visible")
}

// FlushFileSystem flushes file system caches (platform-specific).

func (fsg *FileSystemSyncGuard) FlushFileSystem() {
	// Perform a directory sync operation.

	if f, err := os.Open(fsg.dir); err == nil {

		f.Sync()

		f.Close()

	}

	// Platform-specific optimizations could go here.

	if runtime.GOOS == "windows" {
		time.Sleep(100 * time.Millisecond) // Windows FS cache flush
	}
}

// MockPorchConfig provides configuration for creating mock porch executables.

type MockPorchConfig struct {
	ExitCode int

	Stdout string

	Stderr string

	ProcessDelay time.Duration

	FailPattern string // Pattern in filename that causes failure
}

// CreateMockPorch creates a cross-platform mock porch executable with tracking.

func (h *TestSyncHelper) CreateMockPorch(config MockPorchConfig) (string, *ProcessingTracker) {
	tracker := h.NewProcessingTracker(1) // Will be updated by caller

	options := porch.CrossPlatformMockOptions{
		ExitCode: config.ExitCode,

		Stdout: config.Stdout,

		Stderr: config.Stderr,

		Sleep: config.ProcessDelay,
	}

	if config.FailPattern != "" {
		options.FailOnPattern = config.FailPattern
	}

	mockPath, err := porch.CreateCrossPlatformMock(h.tempDir, options)

	require.NoError(h.t, err)

	return mockPath, tracker
}

// VerifyProcessingResults verifies that files were processed correctly.

func (h *TestSyncHelper) VerifyProcessingResults(expectedProcessed, expectedFailed int) error {
	// Check processed directory.

	processedDir := filepath.Join(h.handoffDir, "processed")

	processedFiles := 0

	if entries, err := os.ReadDir(processedDir); err == nil {
		// Count only intent files, not any auxiliary files
		for _, entry := range entries {
			if !entry.IsDir() && IsIntentFile(entry.Name()) {
				processedFiles++
			}
		}
	}

	// Check failed directory.

	failedDir := filepath.Join(h.handoffDir, "failed")

	failedFiles := 0

	if entries, err := os.ReadDir(failedDir); err == nil {
		// Count only intent files, not error logs
		for _, entry := range entries {
			if !entry.IsDir() && IsIntentFile(entry.Name()) {
				failedFiles++
			}
		}
	}

	// Check status directory.

	statusDir := filepath.Join(h.handoffDir, "status")

	statusFiles := 0

	if entries, err := os.ReadDir(statusDir); err == nil {
		statusFiles = len(entries)
	}

	// Verify counts.

	if processedFiles != expectedProcessed {
		return fmt.Errorf("expected %d processed files, got %d", expectedProcessed, processedFiles)
	}

	if failedFiles != expectedFailed {
		return fmt.Errorf("expected %d failed files, got %d", expectedFailed, failedFiles)
	}

	expectedStatus := expectedProcessed + expectedFailed

	if statusFiles != expectedStatus {
		return fmt.Errorf("expected %d status files, got %d", expectedStatus, statusFiles)
	}

	return nil
}

// WaitWithTimeout waits for a condition with timeout.

func (h *TestSyncHelper) WaitWithTimeout(condition func() bool, timeout time.Duration, description string) error {
	deadline := time.Now().Add(timeout)

	checkInterval := timeout / 20 // Check 20 times during timeout period

	if checkInterval < 10*time.Millisecond {
		checkInterval = 10 * time.Millisecond
	}

	for time.Now().Before(deadline) {

		if condition() {
			return nil
		}

		time.Sleep(checkInterval)

	}

	return fmt.Errorf("timeout waiting for %s", description)
}

// GetTempDir returns the temporary directory path.

func (h *TestSyncHelper) GetTempDir() string {
	return h.tempDir
}

// GetHandoffDir returns the handoff directory path.

func (h *TestSyncHelper) GetHandoffDir() string {
	return h.handoffDir
}

// GetOutDir returns the output directory path.

func (h *TestSyncHelper) GetOutDir() string {
	return h.outDir
}

// Cleanup performs cleanup operations.

func (h *TestSyncHelper) Cleanup() {
	// The temporary directory will be cleaned up automatically by testing.T.TempDir().

	// But we can perform any additional cleanup here if needed.
}
