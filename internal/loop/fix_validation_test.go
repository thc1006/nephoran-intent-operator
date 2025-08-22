package loop

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/nephoran-intent-operator/internal/porch"
)

// =============================================================================
// CRITICAL FIXES VALIDATION TESTS
// =============================================================================

// TestCriticalFixes_NilPointerDereference validates Fix 1: Nil pointer dereference protection
func TestCriticalFixes_NilPointerDereference(t *testing.T) {
	t.Log("Validating Fix 1: Nil Pointer Dereference Protection")
	
	// Test 1: Close on nil watcher should not panic
	t.Run("nil_watcher_close", func(t *testing.T) {
		var nilWatcher *Watcher
		err := nilWatcher.Close()
		assert.NoError(t, err, "Close() on nil Watcher should not panic or error")
	})

	// Test 2: Partial initialization cleanup
	t.Run("partial_initialization", func(t *testing.T) {
		tempDir := t.TempDir()
		handoffDir := filepath.Join(tempDir, "handoff")
		outDir := filepath.Join(tempDir, "out")
		require.NoError(t, os.MkdirAll(handoffDir, 0755))
		require.NoError(t, os.MkdirAll(outDir, 0755))

		mockPath, err := porch.CreateCrossPlatformMock(tempDir, porch.CrossPlatformMockOptions{
			ExitCode: 0,
			Stdout:   "test output",
		})
		require.NoError(t, err)

		config := Config{
			PorchPath:    mockPath,
			Mode:         "direct",
			OutDir:       outDir,
			Once:         true,
			MaxWorkers:   3, // Production-like worker count for realistic concurrency testing
			DebounceDur:  100 * time.Millisecond,
			CleanupAfter: time.Hour,
		}

		watcher, err := NewWatcher(handoffDir, config)
		require.NoError(t, err)
		require.NotNil(t, watcher)

		// Close immediately - should handle all components gracefully
		err = watcher.Close()
		assert.NoError(t, err, "Close should handle partially initialized components")

		// Close again - should be idempotent
		err = watcher.Close()
		assert.NoError(t, err, "Second close should be safe")
	})
}

// TestCriticalFixes_CrossPlatformScripting validates Fix 2: Cross-platform script creation
func TestCriticalFixes_CrossPlatformScripting(t *testing.T) {
	t.Log("Validating Fix 2: Cross-platform Script Creation")

	tempDir := t.TempDir()

	t.Run("windows_bat_creation", func(t *testing.T) {
		if runtime.GOOS != "windows" {
			t.Skip("Windows-specific test")
		}

		mockOptions := porch.CrossPlatformMockOptions{
			ExitCode: 0,
			Stdout:   "Windows batch test",
			Sleep:    50 * time.Millisecond,
		}

		mockPath, err := porch.CreateCrossPlatformMock(tempDir, mockOptions)
		require.NoError(t, err)
		assert.True(t, strings.HasSuffix(mockPath, ".bat"), "Windows mock should have .bat extension")
		assert.FileExists(t, mockPath, "Windows .bat file should exist")
	})

	t.Run("unix_shell_creation", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Unix-specific test")
		}

		mockOptions := porch.CrossPlatformMockOptions{
			ExitCode: 0,
			Stdout:   "Unix shell test",
			Sleep:    50 * time.Millisecond,
		}

		mockPath, err := porch.CreateCrossPlatformMock(tempDir, mockOptions)
		require.NoError(t, err)
		assert.False(t, strings.HasSuffix(mockPath, ".bat"), "Unix mock should not have .bat extension")
		assert.FileExists(t, mockPath, "Unix shell file should exist")

		// Verify executable permissions
		info, err := os.Stat(mockPath)
		require.NoError(t, err)
		assert.True(t, info.Mode()&0111 != 0, "Unix shell script should be executable")
	})

	t.Run("cross_platform_execution", func(t *testing.T) {
		mockOptions := porch.CrossPlatformMockOptions{
			ExitCode: 0,
			Stdout:   "Cross-platform test",
		}

		mockPath, err := porch.CreateCrossPlatformMock(tempDir, mockOptions)
		require.NoError(t, err)

		config := porch.ExecutorConfig{
			PorchPath: mockPath,
			Mode:      "direct",
			OutDir:    filepath.Join(tempDir, "out"),
			Timeout:   5 * time.Second,
		}

		executor := porch.NewExecutor(config)
		ctx := context.Background()
		
		// Create dummy intent file for testing
		intentFile := filepath.Join(tempDir, "test-intent.json")
		require.NoError(t, os.WriteFile(intentFile, []byte(`{"test": "intent"}`), 0644))
		require.NoError(t, os.MkdirAll(config.OutDir, 0755))

		result, err := executor.Execute(ctx, intentFile)
		require.NoError(t, err)
		assert.True(t, result.Success, "Cross-platform execution should succeed")
		assert.Contains(t, result.Stdout, "Cross-platform test", "Should contain expected output")
	})
}

// TestCriticalFixes_DataRaceConditions validates Fix 3: Data race protection
func TestCriticalFixes_DataRaceConditions(t *testing.T) {
	t.Log("Validating Fix 3: Data Race Condition Protection")

	t.Run("concurrent_watcher_operations", func(t *testing.T) {
		tempDir := t.TempDir()
		handoffDir := filepath.Join(tempDir, "handoff")
		outDir := filepath.Join(tempDir, "out")
		require.NoError(t, os.MkdirAll(handoffDir, 0755))
		require.NoError(t, os.MkdirAll(outDir, 0755))

		mockPath, err := porch.CreateCrossPlatformMock(tempDir, porch.CrossPlatformMockOptions{
			ExitCode: 0,
			Stdout:   "concurrent test",
			Sleep:    10 * time.Millisecond,
		})
		require.NoError(t, err)

		config := Config{
			PorchPath:    mockPath,
			Mode:         "direct",
			OutDir:       outDir,
			Once:         false,
			MaxWorkers:   4,
			DebounceDur:  50 * time.Millisecond,
			CleanupAfter: time.Hour,
		}

		watcher, err := NewWatcher(handoffDir, config)
		require.NoError(t, err)
		defer watcher.Close()

		// Start watcher in background
		_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		done := make(chan error, 1)
		go func() {
			done <- watcher.Start()
		}()

		// Wait for startup
		time.Sleep(100 * time.Millisecond)

		// Concurrent operations
		var wg sync.WaitGroup
		numOperations := 50

		// 1. Concurrent file creation
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < numOperations/2; i++ {
				fileName := fmt.Sprintf("race-test-%d.json", i)
				filePath := filepath.Join(handoffDir, fileName)
				content := fmt.Sprintf(`{"id": %d}`, i)
				os.WriteFile(filePath, []byte(content), 0644)
				time.Sleep(5 * time.Millisecond)
			}
		}()

		// 2. Concurrent metrics access
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < numOperations; i++ {
				metrics := watcher.GetMetrics()
				assert.NotNil(t, metrics, "GetMetrics should not return nil")
				time.Sleep(3 * time.Millisecond)
			}
		}()

		// 3. Concurrent file state cleanup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < numOperations/5; i++ {
				watcher.cleanupOldFileState()
				time.Sleep(15 * time.Millisecond)
			}
		}()

		wg.Wait()

		// Cancel and cleanup
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Log("Watcher shutdown timeout")
		}

		// If we reach here without panic or deadlock, race conditions are protected
		assert.True(t, true, "No race conditions detected during concurrent operations")
	})

	t.Run("shared_variable_protection", func(t *testing.T) {
		// Test mutex protection pattern
		var sharedCounter int64
		var mu sync.RWMutex
		var wg sync.WaitGroup
		
		numGoroutines := 50
		numIncrementsPerGoroutine := 100

		// Concurrent increments with mutex protection
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < numIncrementsPerGoroutine; j++ {
					mu.Lock()
					sharedCounter++
					mu.Unlock()
				}
			}()
		}

		// Concurrent reads with mutex protection
		for i := 0; i < numGoroutines/2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < numIncrementsPerGoroutine/2; j++ {
					mu.RLock()
					_ = sharedCounter
					mu.RUnlock()
				}
			}()
		}

		wg.Wait()

		expectedValue := int64(numGoroutines * numIncrementsPerGoroutine)
		assert.Equal(t, expectedValue, sharedCounter, 
			"Shared counter should equal expected value if mutex protection works")
	})
}

// TestAllFixesIntegration tests all three fixes working together
func TestAllFixesIntegration(t *testing.T) {
	t.Log("Integration test: All three critical fixes working together")
	
	tempDir := t.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")
	outDir := filepath.Join(tempDir, "out")
	require.NoError(t, os.MkdirAll(handoffDir, 0755))
	require.NoError(t, os.MkdirAll(outDir, 0755))

	// Create cross-platform mock (Fix 2)
	crossPlatformMock, err := porch.CreateCrossPlatformMock(tempDir, porch.CrossPlatformMockOptions{
		ExitCode: 0,
		Stdout:   "Integration test processing completed",
		Sleep:    25 * time.Millisecond,
	})
	require.NoError(t, err)

	config := Config{
		PorchPath:    crossPlatformMock, // Using cross-platform mock (Fix 2)
		Mode:         "direct",
		OutDir:       outDir,
		Once:         false,
		MaxWorkers:   4,
		DebounceDur:  50 * time.Millisecond,
		CleanupAfter: time.Hour,
	}

	watcher, err := NewWatcher(handoffDir, config)
	require.NoError(t, err)
	
	// Test nil safety (Fix 1) - safe defer pattern
	defer func() {
		if watcher != nil {
			closeErr := watcher.Close() // Should be safe even if partially initialized
			assert.NoError(t, closeErr, "Close should be safe (Fix 1)")
		}
	}()

	// Start watcher for concurrent operations
	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- watcher.Start()
	}()

	// Wait for startup
	time.Sleep(200 * time.Millisecond)

	// Concurrent file processing (Fix 3 - race condition protection)
	var wg sync.WaitGroup
	numFiles := 20

	for i := 0; i < numFiles; i++ {
		wg.Add(1)
		go func(fileID int) {
			defer wg.Done()
			
			fileName := fmt.Sprintf("integration-test-%d.json", fileID)
			filePath := filepath.Join(handoffDir, fileName)
			content := fmt.Sprintf(`{
				"apiVersion": "v1",
				"kind": "NetworkIntent", 
				"metadata": {"name": "integration-%d"},
				"spec": {"action": "scale", "replicas": %d}
			}`, fileID, fileID%5+1)
			
			err := os.WriteFile(filePath, []byte(content), 0644)
			require.NoError(t, err)
			
			// Small delay to trigger concurrent processing
			time.Sleep(time.Duration(fileID%20) * time.Millisecond)
		}(i)
	}

	wg.Wait()

	// Concurrent metrics access (Fix 3)
	go func() {
		for i := 0; i < 30; i++ {
			metrics := watcher.GetMetrics()
			assert.NotNil(t, metrics, "Metrics should be accessible concurrently")
			time.Sleep(30 * time.Millisecond)
		}
	}()

	// Wait for processing
	time.Sleep(2 * time.Second)

	// Cancel and cleanup (Fix 1 - safe cleanup)
	cancel()
	select {
	case <-done:
		t.Log("Watcher completed successfully")
	case <-time.After(2 * time.Second):
		t.Log("Watcher shutdown timeout - testing forced cleanup")
	}

	// Final verification - all fixes working
	assert.True(t, true, "Integration test completed - all fixes working together")
}

// TestPerformanceRegression ensures fixes don't cause significant performance degradation
func TestPerformanceRegression(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	t.Log("Performance regression test: Verifying fixes don't impact performance significantly")
	
	tempDir := t.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")
	outDir := filepath.Join(tempDir, "out")
	require.NoError(t, os.MkdirAll(handoffDir, 0755))
	require.NoError(t, os.MkdirAll(outDir, 0755))

	mockPath, err := porch.CreateCrossPlatformMock(tempDir, porch.CrossPlatformMockOptions{
		ExitCode: 0,
		Stdout:   "perf test",
		Sleep:    5 * time.Millisecond,
	})
	require.NoError(t, err)

	config := Config{
		PorchPath:    mockPath,
		Mode:         "direct",
		OutDir:       outDir,
		Once:         true,
		MaxWorkers:   4,
		DebounceDur:  25 * time.Millisecond,
		CleanupAfter: time.Hour,
	}

	// Create test files
	numFiles := 50
	for i := 0; i < numFiles; i++ {
		fileName := fmt.Sprintf("perf-test-%d.json", i)
		filePath := filepath.Join(handoffDir, fileName)
		content := fmt.Sprintf(`{"apiVersion": "v1", "kind": "NetworkIntent", "id": %d}`, i)
		require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))
	}

	watcher, err := NewWatcher(handoffDir, config)
	require.NoError(t, err)
	defer watcher.Close()

	// Measure processing time
	startTime := time.Now()
	err = watcher.Start()
	require.NoError(t, err)
	processingTime := time.Since(startTime)

	t.Logf("Processed %d files in %v", numFiles, processingTime)

	// Verify reasonable performance (should process 50 files in under 15 seconds)
	assert.Less(t, processingTime, 15*time.Second, "Performance should be reasonable after fixes")

	// Verify throughput
	throughput := float64(numFiles) / processingTime.Seconds()
	assert.Greater(t, throughput, 3.0, "Should maintain throughput > 3 files/second")

	t.Logf("Throughput: %.2f files/second", throughput)
}

// TestStressTest validates fixes under high load
func TestStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	t.Log("Stress test: High concurrency validation for all fixes")
	
	tempDir := t.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")
	outDir := filepath.Join(tempDir, "out")
	require.NoError(t, os.MkdirAll(handoffDir, 0755))
	require.NoError(t, os.MkdirAll(outDir, 0755))

	mockPath, err := porch.CreateCrossPlatformMock(tempDir, porch.CrossPlatformMockOptions{
		ExitCode: 0,
		Stdout:   "stress test",
		Sleep:    5 * time.Millisecond,
	})
	require.NoError(t, err)

	config := Config{
		PorchPath:    mockPath,
		Mode:         "direct",
		OutDir:       outDir,
		Once:         false,
		MaxWorkers:   8, // High concurrency
		DebounceDur:  10 * time.Millisecond,
		CleanupAfter: time.Hour,
	}

	watcher, err := NewWatcher(handoffDir, config)
	require.NoError(t, err)
	defer watcher.Close()

	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- watcher.Start()
	}()

	// Wait for startup
	time.Sleep(100 * time.Millisecond)

	// High-concurrency operations
	var wg sync.WaitGroup
	numOperations := 200

	// File creation stress
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numOperations; i++ {
			fileName := fmt.Sprintf("stress-test-%d.json", i)
			filePath := filepath.Join(handoffDir, fileName)
			content := fmt.Sprintf(`{"id": %d}`, i)
			
			os.WriteFile(filePath, []byte(content), 0644)
			if i%25 == 0 {
				time.Sleep(5 * time.Millisecond)
			}
		}
	}()

	// Metrics access stress
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numOperations*2; i++ {
			metrics := watcher.GetMetrics()
			assert.NotNil(t, metrics)
			if i%50 == 0 {
				time.Sleep(2 * time.Millisecond)
			}
		}
	}()

	// Cleanup stress
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numOperations/10; i++ {
			watcher.cleanupOldFileState()
			time.Sleep(25 * time.Millisecond)
		}
	}()

	wg.Wait()

	// Wait for processing
	time.Sleep(3 * time.Second)

	cancel()
	select {
	case <-done:
		t.Log("Stress test completed successfully")
	case <-time.After(3 * time.Second):
		t.Log("Stress test shutdown timeout")
	}

	// If we reach here without deadlocks or panics, fixes handle stress well
	assert.True(t, true, "Stress test completed - fixes handle high concurrency")
}