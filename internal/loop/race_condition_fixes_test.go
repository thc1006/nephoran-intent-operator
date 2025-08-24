package loop

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFixedRaceConditions demonstrates the fixes for the main race condition issues
func TestFixedRaceConditions(t *testing.T) {
	t.Run("FixedFilenamePattern", func(t *testing.T) {
		// BEFORE: Tests created files like 'concurrent-N.json'
		// AFTER: Ensure files follow 'intent-*.json' pattern
		
		syncHelper := NewTestSyncHelper(t)
		defer syncHelper.Cleanup()
		
		// Test that various filenames get converted to intent pattern
		testCases := []struct {
			input    string
			expected string
		}{
			{"concurrent-1.json", "intent-concurrent-1.json"},
			{"test.json", "intent-test.json"},
			{"intent-valid.json", "intent-valid.json"}, // Already correct
		}
		
		testContent := `{"intent_type": "scaling", "target": "test-deployment", "namespace": "default", "replicas": 3}`
		
		for _, tc := range testCases {
			path := syncHelper.CreateIntentFile(tc.input, testContent)
			actualName := filepath.Base(path)
			assert.Equal(t, tc.expected, actualName, "Filename should be converted to intent pattern")
			assert.True(t, IsIntentFile(actualName), "File should match intent pattern")
		}
	})
	
	t.Run("FixedFileExistsBeforeWatcher", func(t *testing.T) {
		// BEFORE: Watcher started, found no files, exited before files were created
		// AFTER: Ensure files exist BEFORE starting watcher
		
		syncHelper := NewTestSyncHelper(t)
		defer syncHelper.Cleanup()
		
		// Step 1: Create files FIRST with different content to avoid duplicate detection
		expectedFiles := []struct {
			name string
			content string
		}{
			{"intent-race-fix-1.json", `{"intent_type": "scaling", "target": "test-deployment-1", "namespace": "default", "replicas": 3}`},
			{"intent-race-fix-2.json", `{"intent_type": "scaling", "target": "test-deployment-2", "namespace": "default", "replicas": 5}`},
		}
		
		var createdPaths []string
		for _, file := range expectedFiles {
			path := syncHelper.CreateIntentFile(file.name, file.content)
			createdPaths = append(createdPaths, path)
		}
		
		// Step 2: Ensure files are visible to filesystem
		syncGuard := syncHelper.NewFileSystemSyncGuard()
		require.NoError(t, syncGuard.EnsureFilesVisible(createdPaths))
		
		// Step 3: Create mock porch
		mockConfig := MockPorchConfig{
			ExitCode: 0,
			Stdout:   "Race condition fix success",
			ProcessDelay: 50 * time.Millisecond,
		}
		mockPorchPath, _ := syncHelper.CreateMockPorch(mockConfig)
		
		// Step 4: NOW create and start watcher (files already exist)
		config := Config{
			PorchPath:   mockPorchPath,
			Mode:        "direct",
			OutDir:      syncHelper.GetOutDir(),
			Once:        true,
			DebounceDur: 25 * time.Millisecond,
			MaxWorkers:  1,
		}
		
		// Step 5: Use enhanced once watcher for proper completion tracking
		enhancedWatcher, err := syncHelper.NewEnhancedOnceWatcher(config, 2)
		require.NoError(t, err)
		defer enhancedWatcher.Close()
		
		// Step 6: Start with completion tracking
		require.NoError(t, enhancedWatcher.StartWithTracking())
		
		// Step 7: Wait for processing to complete
		err = enhancedWatcher.WaitForProcessingComplete(10 * time.Second)
		require.NoError(t, err)
		
		// Step 8: Verify results
		err = syncHelper.VerifyProcessingResults(2, 0)
		require.NoError(t, err)
		
		t.Log("Fixed race condition: files exist before watcher starts")
	})
	
	t.Run("FixedOnceModeTiming", func(t *testing.T) {
		// BEFORE: Once mode exited immediately without waiting for processing
		// AFTER: Wait for actual processing completion
		
		syncHelper := NewTestSyncHelper(t)
		defer syncHelper.Cleanup()
		
		// Create test files with different content to avoid duplicate detection
		file1 := syncHelper.CreateIntentFile("intent-once-timing-1.json", `{"intent_type": "scaling", "target": "test-timing-1", "namespace": "default", "replicas": 2}`)
		file2 := syncHelper.CreateIntentFile("intent-once-timing-2.json", `{"intent_type": "scaling", "target": "test-timing-2", "namespace": "default", "replicas": 4}`)
		
		// Ensure files are visible
		syncGuard := syncHelper.NewFileSystemSyncGuard()
		require.NoError(t, syncGuard.EnsureFilesVisible([]string{file1, file2}))
		
		// Create mock with processing delay to simulate real work
		mockConfig := MockPorchConfig{
			ExitCode:     0,
			Stdout:       "Once mode timing fix",
			ProcessDelay: 200 * time.Millisecond, // Simulate processing time
		}
		mockPorchPath, _ := syncHelper.CreateMockPorch(mockConfig)
		
		// Configure once mode
		config := Config{
			PorchPath:   mockPorchPath,
			Mode:        "direct",
			OutDir:      syncHelper.GetOutDir(),
			Once:        true, // This is the critical once mode
			DebounceDur: 25 * time.Millisecond,
			MaxWorkers:  1,
		}
		
		// Use once mode synchronizer for proper completion tracking
		watcher, err := syncHelper.StartWatcherWithSync(config)
		require.NoError(t, err)
		defer watcher.Close()
		
		synchronizer := NewOnceModeSynchronizer(watcher, 2)
		
		// This will wait for actual processing completion, not just watcher exit
		startTime := time.Now()
		err = synchronizer.StartWithCompletion(15 * time.Second)
		require.NoError(t, err)
		processingDuration := time.Since(startTime)
		
		// Verify processing took reasonable time (not immediate exit)
		assert.Greater(t, processingDuration, 100*time.Millisecond, 
			"Processing should take time, not exit immediately")
		
		// Verify results
		err = syncHelper.VerifyProcessingResults(2, 0)
		require.NoError(t, err)
		
		t.Logf("Fixed once mode timing: processing took %v", processingDuration)
	})
	
	t.Run("FixedCrossPlatformTiming", func(t *testing.T) {
		// BEFORE: Hardcoded timing that failed on different platforms
		// AFTER: Platform-aware timing and synchronization
		
		syncHelper := NewTestSyncHelper(t)
		defer syncHelper.Cleanup()
		
		// Note: syncHelper automatically adjusts timing based on platform
		t.Logf("Using platform-aware timing: baseDelay=%v, fileSettle=%v, processWait=%v",
			syncHelper.baseDelay, syncHelper.fileSettle, syncHelper.processWait)
		
		testContent := `{"intent_type": "scaling", "target": "test-deployment", "namespace": "default", "replicas": 3}`
		
		// Create files using platform-aware helper
		file := syncHelper.CreateIntentFile("intent-cross-platform.json", testContent)
		
		// Platform-specific filesystem flush
		syncGuard := syncHelper.NewFileSystemSyncGuard()
		syncGuard.FlushFileSystem() // Does platform-specific operations
		require.NoError(t, syncGuard.EnsureFilesVisible([]string{file}))
		
		// Create mock with platform-aware timing
		mockConfig := MockPorchConfig{
			ExitCode:     0,
			Stdout:       "Cross-platform timing success",
			ProcessDelay: syncHelper.baseDelay, // Platform-appropriate delay
		}
		mockPorchPath, _ := syncHelper.CreateMockPorch(mockConfig)
		
		// Configure with platform-aware settings
		config := Config{
			PorchPath:   mockPorchPath,
			Mode:        "direct",
			OutDir:      syncHelper.GetOutDir(),
			Once:        true,
			DebounceDur: syncHelper.baseDelay,
			MaxWorkers:  1,
		}
		
		// Use enhanced watcher with platform-aware timeouts
		enhancedWatcher, err := syncHelper.NewEnhancedOnceWatcher(config, 1)
		require.NoError(t, err)
		defer enhancedWatcher.Close()
		
		require.NoError(t, enhancedWatcher.StartWithTracking())
		
		// Use platform-aware timeout
		err = enhancedWatcher.WaitForProcessingComplete(syncHelper.processWait)
		require.NoError(t, err)
		
		// Verify results
		err = syncHelper.VerifyProcessingResults(1, 0)
		require.NoError(t, err)
		
		t.Log("Fixed cross-platform timing issues")
	})
}

// TestSynchronizationPrimitives tests the individual synchronization components
func TestSynchronizationPrimitives(t *testing.T) {
	t.Run("FileCreationSynchronizer", func(t *testing.T) {
		expectedFiles := []string{"file1.json", "file2.json"}
		fcs := NewFileCreationSynchronizer("/tmp", expectedFiles, 1*time.Second)
		
		// Simulate file creation
		go func() {
			time.Sleep(100 * time.Millisecond)
			fcs.NotifyFileCreated("file1.json")
			time.Sleep(100 * time.Millisecond)
			fcs.NotifyFileCreated("file2.json")
		}()
		
		// Should complete without timeout
		err := fcs.WaitForAllFiles()
		require.NoError(t, err)
	})
	
	t.Run("CrossPlatformSyncBarrier", func(t *testing.T) {
		barrier := NewCrossPlatformSyncBarrier(3)
		completed := make(chan int, 3)
		
		// Start 3 participants
		for i := 0; i < 3; i++ {
			go func(id int) {
				time.Sleep(time.Duration(id*50) * time.Millisecond) // Stagger arrival
				barrier.Wait()
				completed <- id
			}(i)
		}
		
		// All should complete around the same time
		for i := 0; i < 3; i++ {
			select {
			case id := <-completed:
				t.Logf("Participant %d completed", id)
			case <-time.After(1 * time.Second):
				t.Fatal("Timeout waiting for participants")
			}
		}
	})
	
	t.Run("EnhancedOnceState", func(t *testing.T) {
		state := NewEnhancedOnceState()
		
		// Simulate processing lifecycle
		state.MarkScanComplete(3)
		state.MarkProcessingStarted()
		
		state.IncrementProcessed()
		state.IncrementProcessed()
		state.IncrementFailed()
		
		state.MarkProcessingDone()
		
		// Check completion
		assert.True(t, state.IsComplete())
		
		// Get stats
		stats := state.GetStats()
		assert.Equal(t, int64(3), stats["files_scanned"])
		assert.Equal(t, int64(2), stats["files_processed"])
		assert.Equal(t, int64(1), stats["files_failed"])
		assert.True(t, stats["processing_done"].(bool))
	})
}

// TestErrorHandling tests error scenarios
func TestErrorHandling(t *testing.T) {
	t.Run("SynchronizationTimeout", func(t *testing.T) {
		expectedFiles := []string{"file1.json", "file2.json"}
		fcs := NewFileCreationSynchronizer("/tmp", expectedFiles, 100*time.Millisecond)
		
		// Only create one file - should timeout
		go func() {
			time.Sleep(50 * time.Millisecond)
			fcs.NotifyFileCreated("file1.json")
		}()
		
		err := fcs.WaitForAllFiles()
		require.Error(t, err)
		
		// Should be a synchronization timeout error
		syncErr, ok := err.(*SynchronizationTimeoutError)
		require.True(t, ok)
		assert.Equal(t, "file creation", syncErr.Operation)
		assert.Equal(t, 2, syncErr.Expected)
		assert.Equal(t, 1, syncErr.Actual)
	})
}

// BenchmarkSynchronizationOverhead benchmarks the overhead of synchronization
func BenchmarkSynchronizationOverhead(b *testing.B) {
	b.Run("WithSynchronization", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			syncHelper := NewTestSyncHelper(b)
			
			testContent := `{"intent_type": "scaling", "target": "test-deployment", "namespace": "default", "replicas": 3}`
			syncHelper.CreateIntentFile("intent-benchmark.json", testContent)
			
			syncHelper.Cleanup()
		}
	})
	
	b.Run("WithoutSynchronization", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Direct file creation without synchronization helpers
			tempDir := b.TempDir()
			file := filepath.Join(tempDir, "intent-benchmark.json")
			testContent := `{"intent_type": "scaling", "target": "test-deployment", "namespace": "default", "replicas": 3}`
			
			_ = file
			_ = testContent
			// Would write file directly here
		}
	})
}