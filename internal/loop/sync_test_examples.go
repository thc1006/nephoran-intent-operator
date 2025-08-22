package loop

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ExampleFixedOnceMode demonstrates how to fix race conditions in once mode tests
func ExampleFixedOnceMode(t *testing.T) {
	// Create test synchronization helper
	syncHelper := NewTestSyncHelper(t)
	defer syncHelper.Cleanup()
	
	// Step 1: Create intent files with proper naming BEFORE starting watcher
	testContent := `{"apiVersion": "v1", "kind": "NetworkIntent", "spec": {"action": "scale", "target": "deployment", "count": 3}}`
	
	expectedFiles := []string{
		"intent-test-1.json",
		"intent-test-2.json", 
		"intent-test-3.json",
	}
	
	// Create files synchronously to ensure they exist
	var createdPaths []string
	for _, filename := range expectedFiles {
		path := syncHelper.CreateIntentFile(filename, testContent)
		createdPaths = append(createdPaths, path)
	}
	
	// Step 2: Ensure files are visible to filesystem
	syncGuard := syncHelper.NewFileSystemSyncGuard()
	require.NoError(t, syncGuard.EnsureFilesVisible(createdPaths))
	
	// Step 3: Create mock porch with tracking
	mockConfig := MockPorchConfig{
		ExitCode:     0,
		Stdout:       "Processing completed successfully",
		ProcessDelay: 100 * time.Millisecond,
	}
	mockPorchPath, _ := syncHelper.CreateMockPorch(mockConfig)
	
	// Step 4: Configure watcher with once mode
	config := Config{
		PorchPath:   mockPorchPath,
		Mode:        "direct",
		OutDir:      syncHelper.GetOutDir(),
		Once:        true,
		DebounceDur: 50 * time.Millisecond,
		MaxWorkers:  2,
	}
	
	// Step 5: Create enhanced once watcher with completion tracking
	enhancedWatcher, err := syncHelper.NewEnhancedOnceWatcher(config, len(expectedFiles))
	require.NoError(t, err)
	defer enhancedWatcher.Close()
	
	// Step 6: Start watcher with completion tracking
	require.NoError(t, enhancedWatcher.StartWithTracking())
	
	// Step 7: Wait for processing to complete with timeout
	err = enhancedWatcher.WaitForProcessingComplete(10 * time.Second)
	require.NoError(t, err)
	
	// Step 8: Verify results
	err = syncHelper.VerifyProcessingResults(len(expectedFiles), 0)
	require.NoError(t, err)
	
	t.Logf("Successfully processed %d files in once mode", len(expectedFiles))
}

// ExampleConcurrentFileProcessing demonstrates race condition-free concurrent testing
func ExampleConcurrentFileProcessing(t *testing.T) {
	syncHelper := NewTestSyncHelper(t)
	defer syncHelper.Cleanup()
	
	// Create multiple files concurrently with staggered timing
	numFiles := 10
	contentTemplate := `{"apiVersion": "v1", "kind": "NetworkIntent", "spec": {"action": "scale", "target": "deployment-%d", "count": %d}}`
	
	createdFiles := syncHelper.CreateMultipleIntentFiles(numFiles, contentTemplate)
	
	// Wait for all files to be created
	require.True(t, syncHelper.WaitForFilesCreated(5*time.Second), "Files should be created within timeout")
	
	// Ensure filesystem visibility
	syncGuard := syncHelper.NewFileSystemSyncGuard()
	require.NoError(t, syncGuard.EnsureFilesVisible(createdFiles))
	
	// Create mock porch with processing delay
	mockConfig := MockPorchConfig{
		ExitCode:     0,
		Stdout:       "Processed successfully",
		ProcessDelay: 50 * time.Millisecond, // Small delay to simulate real processing
	}
	mockPorchPath, _ := syncHelper.CreateMockPorch(mockConfig)
	
	// Configure watcher
	config := Config{
		PorchPath:   mockPorchPath,
		Mode:        "direct", 
		OutDir:      syncHelper.GetOutDir(),
		Once:        true,
		DebounceDur: 25 * time.Millisecond,
		MaxWorkers:  4, // Multiple workers for concurrency
	}
	
	// Create synchronizer for once mode
	watcher, err := syncHelper.StartWatcherWithSync(config)
	require.NoError(t, err)
	defer watcher.Close()
	
	// Create completion waiter
	completionWaiter := NewProcessingCompletionWaiter(watcher, numFiles)
	
	// Start watcher
	startTime := time.Now()
	go func() {
		err := watcher.Start()
		assert.NoError(t, err)
	}()
	
	// Wait for completion
	err = completionWaiter.WaitForCompletion()
	require.NoError(t, err)
	
	processingDuration := time.Since(startTime)
	t.Logf("Processed %d files concurrently in %v", numFiles, processingDuration)
	
	// Verify results
	err = syncHelper.VerifyProcessingResults(numFiles, 0)
	require.NoError(t, err)
}

// ExampleFailureHandling demonstrates proper failure handling synchronization
func ExampleFailureHandling(t *testing.T) {
	syncHelper := NewTestSyncHelper(t)
	defer syncHelper.Cleanup()
	
	// Create mix of valid and invalid files
	validContent := `{"apiVersion": "v1", "kind": "NetworkIntent", "spec": {"action": "scale"}}`
	invalidContent := `{invalid json content`
	
	validFile := syncHelper.CreateIntentFile("intent-valid.json", validContent)
	invalidFile := syncHelper.CreateIntentFile("intent-invalid.json", invalidContent)
	
	// Ensure files are visible
	syncGuard := syncHelper.NewFileSystemSyncGuard()
	require.NoError(t, syncGuard.EnsureFilesVisible([]string{validFile, invalidFile}))
	
	// Create mock porch that will succeed for valid files
	mockConfig := MockPorchConfig{
		ExitCode:     0,
		Stdout:       "Success",
		ProcessDelay: 50 * time.Millisecond,
	}
	mockPorchPath, _ := syncHelper.CreateMockPorch(mockConfig)
	
	// Configure watcher
	config := Config{
		PorchPath:   mockPorchPath,
		Mode:        "direct",
		OutDir:      syncHelper.GetOutDir(),
		Once:        true,
		DebounceDur: 25 * time.Millisecond,
		MaxWorkers:  3, // Use production-like worker count
	}
	
	// Create enhanced watcher
	enhancedWatcher, err := syncHelper.NewEnhancedOnceWatcher(config, 2)
	require.NoError(t, err)
	defer enhancedWatcher.Close()
	
	// Start with tracking
	require.NoError(t, enhancedWatcher.StartWithTracking())
	
	// Wait for completion
	err = enhancedWatcher.WaitForProcessingComplete(10 * time.Second)
	require.NoError(t, err)
	
	// Verify results: 1 processed (valid), 1 failed (invalid JSON)
	err = syncHelper.VerifyProcessingResults(1, 1)
	require.NoError(t, err)
	
	t.Log("Successfully handled mixed valid/invalid files")
}

// ExampleCrossPlatformTiming demonstrates cross-platform timing considerations
func ExampleCrossPlatformTiming(t *testing.T) {
	syncHelper := NewTestSyncHelper(t)
	defer syncHelper.Cleanup()
	
	// Create files with platform-appropriate timing
	testContent := `{"apiVersion": "v1", "kind": "NetworkIntent", "spec": {"action": "scale"}}`
	
	// Create files using cross-platform aware helper
	file1 := syncHelper.CreateIntentFile("intent-timing-1.json", testContent)
	file2 := syncHelper.CreateIntentFile("intent-timing-2.json", testContent)
	
	// Use cross-platform sync barrier for coordination
	_ = NewCrossPlatformSyncBarrier(2)
	
	// Ensure files are properly synced
	syncGuard := syncHelper.NewFileSystemSyncGuard()
	syncGuard.FlushFileSystem() // Platform-specific flush
	require.NoError(t, syncGuard.EnsureFilesVisible([]string{file1, file2}))
	
	// Create mock porch
	mockConfig := MockPorchConfig{
		ExitCode:     0,
		Stdout:       "Cross-platform success",
		ProcessDelay: 75 * time.Millisecond,
	}
	mockPorchPath, _ := syncHelper.CreateMockPorch(mockConfig)
	
	// Configure with platform-appropriate timeouts
	config := Config{
		PorchPath:   mockPorchPath,
		Mode:        "direct",
		OutDir:      syncHelper.GetOutDir(),
		Once:        true,
		DebounceDur: syncHelper.baseDelay,    // Platform-aware base delay
		MaxWorkers:  3, // Use production-like worker count
	}
	
	// Test with enhanced once mode synchronization
	enhancedWatcher, err := syncHelper.NewEnhancedOnceWatcher(config, 2)
	require.NoError(t, err)
	defer enhancedWatcher.Close()
	
	require.NoError(t, enhancedWatcher.StartWithTracking())
	
	// Use platform-aware wait time
	err = enhancedWatcher.WaitForProcessingComplete(syncHelper.processWait)
	require.NoError(t, err)
	
	// Verify results
	err = syncHelper.VerifyProcessingResults(2, 0)
	require.NoError(t, err)
	
	t.Log("Cross-platform timing test completed successfully")
}

// ExampleFilePatternValidation demonstrates fixing filename pattern issues
func ExampleFilePatternValidation(t *testing.T) {
	syncHelper := NewTestSyncHelper(t)
	defer syncHelper.Cleanup()
	
	// Test various filename patterns
	testContent := `{"apiVersion": "v1", "kind": "NetworkIntent", "spec": {"action": "scale"}}`
	
	// These will be automatically converted to intent-*.json pattern
	testFiles := []string{
		"concurrent-1.json",        // Will become intent-concurrent-1.json
		"test-file.json",          // Will become intent-test-file.json
		"intent-valid.json",       // Already correct pattern
		"simple.json",             // Will become intent-simple.json
	}
	
	var createdPaths []string
	for _, filename := range testFiles {
		// CreateIntentFile automatically ensures proper naming
		path := syncHelper.CreateIntentFile(filename, testContent)
		createdPaths = append(createdPaths, path)
		
		// Verify the created file follows intent-*.json pattern
		baseName := filepath.Base(path)
		assert.True(t, IsIntentFile(baseName), "File %s should match intent-*.json pattern", baseName)
	}
	
	// Ensure filesystem visibility
	syncGuard := syncHelper.NewFileSystemSyncGuard()
	require.NoError(t, syncGuard.EnsureFilesVisible(createdPaths))
	
	// Create mock porch
	mockConfig := MockPorchConfig{
		ExitCode: 0,
		Stdout:   "Pattern validation success",
	}
	mockPorchPath, _ := syncHelper.CreateMockPorch(mockConfig)
	
	// Configure watcher
	config := Config{
		PorchPath:   mockPorchPath,
		Mode:        "direct",
		OutDir:      syncHelper.GetOutDir(),
		Once:        true,
		DebounceDur: 50 * time.Millisecond,
		MaxWorkers:  2,
	}
	
	// Test with enhanced watcher
	enhancedWatcher, err := syncHelper.NewEnhancedOnceWatcher(config, len(testFiles))
	require.NoError(t, err)
	defer enhancedWatcher.Close()
	
	require.NoError(t, enhancedWatcher.StartWithTracking())
	
	err = enhancedWatcher.WaitForProcessingComplete(10 * time.Second)
	require.NoError(t, err)
	
	// Verify all files were processed
	err = syncHelper.VerifyProcessingResults(len(testFiles), 0)
	require.NoError(t, err)
	
	t.Log("Filename pattern validation test completed successfully")
}

// ExampleDebugTracking demonstrates comprehensive debug tracking
func ExampleDebugTracking(t *testing.T) {
	syncHelper := NewTestSyncHelper(t)
	defer syncHelper.Cleanup()
	
	// Create test files
	testContent := `{"apiVersion": "v1", "kind": "NetworkIntent", "spec": {"action": "scale"}}`
	
	file1 := syncHelper.CreateIntentFile("intent-debug-1.json", testContent)
	file2 := syncHelper.CreateIntentFile("intent-debug-2.json", testContent)
	
	// Create enhanced state tracking
	state := NewEnhancedOnceState()
	
	// Ensure files are visible
	syncGuard := syncHelper.NewFileSystemSyncGuard()
	require.NoError(t, syncGuard.EnsureFilesVisible([]string{file1, file2}))
	
	// Mark scan complete
	state.MarkScanComplete(2)
	
	// Create mock porch with tracking
	mockConfig := MockPorchConfig{
		ExitCode: 0,
		Stdout:   "Debug tracking success",
		ProcessDelay: 100 * time.Millisecond,
	}
	mockPorchPath, _ := syncHelper.CreateMockPorch(mockConfig)
	
	// Configure watcher
	config := Config{
		PorchPath:   mockPorchPath,
		Mode:        "direct",
		OutDir:      syncHelper.GetOutDir(),
		Once:        true,
		DebounceDur: 50 * time.Millisecond,
		MaxWorkers:  3, // Use production-like worker count
	}
	
	// Start with synchronizer
	watcher, err := syncHelper.StartWatcherWithSync(config)
	require.NoError(t, err)
	defer watcher.Close()
	
	synchronizer := NewOnceModeSynchronizer(watcher, 2)
	
	// Mark processing started
	state.MarkProcessingStarted()
	
	// Start processing with completion tracking
	err = synchronizer.StartWithCompletion(15 * time.Second)
	require.NoError(t, err)
	
	// Mark processing done
	state.MarkProcessingDone()
	
	// Get final statistics
	stats := state.GetStats()
	t.Logf("Debug tracking stats: %+v", stats)
	
	// Verify completion
	assert.True(t, state.IsComplete(), "Processing should be complete")
	
	// Verify results
	err = syncHelper.VerifyProcessingResults(2, 0)
	require.NoError(t, err)
	
	t.Log("Debug tracking test completed successfully")
}

// TestSynchronizationTimeoutError tests the custom timeout error
func TestSynchronizationTimeoutError(t *testing.T) {
	err := &SynchronizationTimeoutError{
		Operation: "test operation",
		Expected:  5,
		Actual:    3,
		Timeout:   2 * time.Second,
	}
	
	expectedMsg := "synchronization timeout for test operation: expected 5, got 3 after 2s"
	assert.Equal(t, expectedMsg, err.Error())
}

// BenchmarkSynchronizedProcessing benchmarks the synchronized processing approach
func BenchmarkSynchronizedProcessing(b *testing.B) {
	syncHelper := NewTestSyncHelper(b)
	defer syncHelper.Cleanup()
	
	testContent := `{"apiVersion": "v1", "kind": "NetworkIntent", "spec": {"action": "scale"}}`
	
	// Create mock porch
	mockConfig := MockPorchConfig{
		ExitCode: 0,
		Stdout:   "Benchmark success",
		ProcessDelay: 10 * time.Millisecond, // Fast processing for benchmark
	}
	mockPorchPath, _ := syncHelper.CreateMockPorch(mockConfig)
	
	config := Config{
		PorchPath:   mockPorchPath,
		Mode:        "direct",
		OutDir:      syncHelper.GetOutDir(),
		Once:        true,
		DebounceDur: 10 * time.Millisecond,
		MaxWorkers:  2,
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		
		// Create test file
		filename := fmt.Sprintf("intent-bench-%d.json", i)
		syncHelper.CreateIntentFile(filename, testContent)
		
		// Create watcher
		enhancedWatcher, err := syncHelper.NewEnhancedOnceWatcher(config, 1)
		require.NoError(b, err)
		
		b.StartTimer()
		
		// Measure synchronized processing time
		require.NoError(b, enhancedWatcher.StartWithTracking())
		err = enhancedWatcher.WaitForProcessingComplete(5 * time.Second)
		require.NoError(b, err)
		
		b.StopTimer()
		enhancedWatcher.Close()
	}
}