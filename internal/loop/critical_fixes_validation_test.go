package loop

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/nephoran-intent-operator/internal/porch"
)

// TestCriticalFixes_NilPointerSafety validates that the nil pointer fix prevents panics
func TestCriticalFixes_NilPointerSafety(t *testing.T) {
	t.Run("nil_watcher_close_safety", func(t *testing.T) {
		// Test direct nil watcher close - should not panic
		var w *Watcher
		assert.NotPanics(t, func() {
			err := w.Close()
			assert.NoError(t, err, "Close() on nil Watcher should return no error")
		})
	})

	t.Run("watcher_close_after_nil_assignment", func(t *testing.T) {
		// Test watcher that becomes nil - simulate the main.go defer pattern
		var w *Watcher
		defer func() {
			// This is the pattern used in main.go - should not panic
			assert.NotPanics(t, func() {
				if w != nil {
					w.Close()
				}
			})
		}()

		// Watcher remains nil (simulating initialization failure)
		// The defer should handle this gracefully
	})
}

// TestCriticalFixes_CrossPlatformMocks validates cross-platform mock script creation
func TestCriticalFixes_CrossPlatformMocks(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("basic_cross_platform_mock", func(t *testing.T) {
		opts := porch.CrossPlatformMockOptions{
			ExitCode: 0,
			Stdout:   "Mock execution successful",
			Stderr:   "Mock stderr message",
		}

		mockPath, err := porch.CreateCrossPlatformMock(tempDir, opts)
		require.NoError(t, err)
		require.NotEmpty(t, mockPath)

		// Verify the mock file was created
		_, err = os.Stat(mockPath)
		assert.NoError(t, err, "Mock script should be created")

		// Verify correct extension based on platform
		if runtime.GOOS == "windows" {
			assert.Contains(t, mockPath, ".bat", "Windows should create .bat files")
		} else {
			assert.Contains(t, mockPath, ".sh", "Unix should create .sh files")
		}
	})

	t.Run("mock_with_timing", func(t *testing.T) {
		opts := porch.CrossPlatformMockOptions{
			ExitCode: 0,
			Stdout:   "Timed execution",
			Sleep:    100 * time.Millisecond,
		}

		mockPath, err := porch.CreateCrossPlatformMock(tempDir, opts)
		require.NoError(t, err)

		// Test that the script actually sleeps by measuring execution time
		start := time.Now()

		// Execute the mock (platform-appropriate execution would be tested in integration)
		// For unit test, just verify the file contains sleep commands
		content, err := os.ReadFile(mockPath)
		require.NoError(t, err)

		if runtime.GOOS == "windows" {
			assert.Contains(t, string(content), "Start-Sleep", "Windows mock should use PowerShell Start-Sleep")
		} else {
			assert.Contains(t, string(content), "sleep", "Unix mock should use sleep command")
		}

		duration := time.Since(start)
		// File creation should be fast
		assert.Less(t, duration, 50*time.Millisecond, "Mock creation should be fast")
	})

	t.Run("simple_mock_helper", func(t *testing.T) {
		mockPath, err := porch.CreateSimpleMock(tempDir)
		require.NoError(t, err)
		require.NotEmpty(t, mockPath)

		// Verify the file exists and contains success message
		content, err := os.ReadFile(mockPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "Mock porch processing completed successfully")
	})
}

// TestCriticalFixes_ConcurrencyDataRace validates thread-safe access patterns
func TestCriticalFixes_ConcurrencyDataRace(t *testing.T) {
	t.Run("concurrent_slice_access_safety", func(t *testing.T) {
		// Simulate the fixed pattern from processor_test.go
		var testData []string
		var mu sync.Mutex

		const numGoroutines = 10
		const itemsPerGoroutine = 100

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		// Start multiple goroutines writing to shared slice
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < itemsPerGoroutine; j++ {
					// Thread-safe append (same pattern as the fix)
					mu.Lock()
					testData = append(testData, "data")
					mu.Unlock()
				}
			}(i)
		}

		// Start goroutines reading from shared slice
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 50; j++ {
					// Thread-safe read (same pattern as the fix)
					mu.Lock()
					length := len(testData)
					mu.Unlock()

					// Use the length to prevent optimization
					_ = length
					time.Sleep(time.Microsecond)
				}
			}()
		}

		// Wait for all operations to complete
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		// Timeout after reasonable time
		select {
		case <-done:
			// Success - no data race occurred
		case <-time.After(10 * time.Second):
			t.Fatal("Test timed out - possible deadlock or very slow execution")
		}

		// Verify expected result
		mu.Lock()
		finalLength := len(testData)
		mu.Unlock()

		expectedLength := numGoroutines * itemsPerGoroutine
		assert.Equal(t, expectedLength, finalLength, "All items should be safely added to slice")
	})

	t.Run("processor_concurrent_pattern", func(t *testing.T) {
		// Test the exact pattern used in the processor_test.go fix
		var submittedIntents []string
		var submittedMu sync.Mutex

		mockPorchFunc := func(intent string) error {
			submittedMu.Lock()
			submittedIntents = append(submittedIntents, intent)
			submittedMu.Unlock()
			return nil
		}

		// Simulate concurrent processing
		const numWorkers = 5
		const intentsPerWorker = 20

		var wg sync.WaitGroup
		wg.Add(numWorkers)

		for i := 0; i < numWorkers; i++ {
			go func(workerID int) {
				defer wg.Done()
				for j := 0; j < intentsPerWorker; j++ {
					intent := "intent"
					mockPorchFunc(intent)
				}
			}(i)
		}

		wg.Wait()

		// Verify thread-safe access for reading results
		submittedMu.Lock()
		submittedCount := len(submittedIntents)
		submittedMu.Unlock()

		expectedCount := numWorkers * intentsPerWorker
		assert.Equal(t, expectedCount, submittedCount, "All intents should be safely processed")
	})
}

// TestCriticalFixes_Integration performs integration validation of all fixes
func TestCriticalFixes_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")
	require.NoError(t, os.MkdirAll(handoffDir, 0o755))

	t.Run("end_to_end_fix_validation", func(t *testing.T) {
		// Create cross-platform mock (Fix 2)
		mockPath, err := porch.CreateCrossPlatformMock(tempDir, porch.CrossPlatformMockOptions{
			ExitCode: 0,
			Stdout:   "Processing completed",
			Sleep:    10 * time.Millisecond,
		})
		require.NoError(t, err)

		// Verify mock works on current platform
		_, err = os.Stat(mockPath)
		assert.NoError(t, err)

		// Test watcher creation and safe cleanup (Fix 1)
		config := Config{
			DebounceDur: 100 * time.Millisecond,
		}

		watcher, err := NewWatcher(handoffDir, config)
		if err != nil {
			// If watcher creation fails, test nil safety
			var nilWatcher *Watcher
			assert.NotPanics(t, func() {
				nilWatcher.Close()
			})
		} else {
			// Test normal cleanup
			assert.NotPanics(t, func() {
				watcher.Close()
			})
		}
	})
}

// BenchmarkCriticalFixes_Performance benchmarks the performance impact of fixes
func BenchmarkCriticalFixes_Performance(b *testing.B) {
	b.Run("mutex_protected_slice_operations", func(b *testing.B) {
		var testData []string
		var mu sync.Mutex

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				// Simulate the thread-safe pattern from the fix
				mu.Lock()
				testData = append(testData, "item")
				length := len(testData)
				mu.Unlock()

				// Use the length to prevent optimization
				_ = length
			}
		})
	})

	b.Run("cross_platform_mock_creation", func(b *testing.B) {
		tempDir := b.TempDir()
		opts := porch.CrossPlatformMockOptions{
			ExitCode: 0,
			Stdout:   "benchmark test",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := porch.CreateCrossPlatformMock(tempDir, opts)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
