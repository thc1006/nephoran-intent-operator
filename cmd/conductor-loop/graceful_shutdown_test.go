package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/thc1006/nephoran-intent-operator/internal/loop"
)

// createMockPorchGraceful creates a controllable mock porch executable for graceful shutdown testing
func createMockPorchGraceful(t *testing.T, tempDir string, behavior string) string {
	t.Helper()
	
	var mockPorchPath string
	var mockScript string
	
	if runtime.GOOS == "windows" {
		mockPorchPath = filepath.Join(tempDir, "mock-porch.bat")
		switch behavior {
		case "success":
			mockScript = "@echo off\necho Mock porch processing...\ntimeout /t 1 >nul 2>&1\necho Mock porch completed\nexit /b 0\n"
		case "failure":
			mockScript = "@echo off\necho Mock porch processing...\ntimeout /t 1 >nul 2>&1\necho Mock porch failed\nexit /b 1\n"
		case "slow":
			mockScript = "@echo off\necho Mock porch processing...\ntimeout /t 5 >nul 2>&1\necho Mock porch completed\nexit /b 0\n"
		case "killed":
			mockScript = "@echo off\necho Mock porch processing...\ntaskkill /f /im timeout.exe >nul 2>&1\nexit /b 137\n"
		default:
			mockScript = "@echo off\necho Mock porch processing...\nexit /b 0\n"
		}
	} else {
		mockPorchPath = filepath.Join(tempDir, "mock-porch")
		switch behavior {
		case "success":
			mockScript = "#!/bin/bash\necho 'Mock porch processing...'\nsleep 1\necho 'Mock porch completed'\nexit 0\n"
		case "failure":
			mockScript = "#!/bin/bash\necho 'Mock porch processing...'\nsleep 1\necho 'Mock porch failed'\nexit 1\n"
		case "slow":
			mockScript = "#!/bin/bash\necho 'Mock porch processing...'\nsleep 5\necho 'Mock porch completed'\nexit 0\n"
		case "killed":
			mockScript = "#!/bin/bash\necho 'Mock porch processing...'\nkill -9 $$\n"
		default:
			mockScript = "#!/bin/bash\necho 'Mock porch processing...'\nexit 0\n"
		}
	}
	
	err := os.WriteFile(mockPorchPath, []byte(mockScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock porch: %v", err)
	}
	
	return mockPorchPath
}

func TestGracefulShutdownExitCode(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")
	outDir := filepath.Join(tempDir, "out")

	err := os.MkdirAll(handoffDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create handoff directory: %v", err)
	}

	err = os.MkdirAll(outDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	// Create mock porch executable with slow behavior to ensure graceful shutdown interruption
	mockPorchPath := createMockPorchGraceful(t, tempDir, "slow")

	// Create watcher with legacy config approach for testing
	config := loop.Config{
		PorchPath:   mockPorchPath,
		Mode:        "direct",
		OutDir:      outDir,
		Once:        true,
		MaxWorkers:  2,
		DebounceDur: 100 * time.Millisecond,
	}

	watcher, err := loop.NewWatcher(handoffDir, config)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer func() { _ = watcher.Close() }()

	// Create intent files that will be processed
	intentFiles := []string{
		"intent-test-1.json",
		"intent-test-2.json",
		"intent-test-3.json",
	}

	intentContent := `{
	"intent_type": "scaling",
	"target": "test-deployment",
	"namespace": "default",
	"replicas": 3,
	"source": "test"
}`

	for _, filename := range intentFiles {
		filePath := filepath.Join(handoffDir, filename)
		err := os.WriteFile(filePath, []byte(intentContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create intent file %s: %v", filename, err)
		}
	}

	// Start processing in background
	done := make(chan error, 1)
	started := make(chan struct{})
	go func() {
		close(started) // Signal that Start() has been called
		done <- watcher.Start()
	}()

	// Wait for Start() to be called
	<-started
	
	// Let processing begin, then simulate graceful shutdown
	// Give enough time for files to be queued but not necessarily completed
	time.Sleep(200 * time.Millisecond)
	
	// Trigger graceful shutdown
	watcher.Close()

	// Wait for processing to complete with appropriate timeout
	select {
	case err := <-done:
		if err != nil {
			t.Logf("Watcher returned error (expected during shutdown): %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout waiting for watcher to complete")
	}

	// Check processing stats
	stats, err := watcher.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	t.Logf("Processing stats:")
	t.Logf("  Total processed: %d", stats.ProcessedCount)
	t.Logf("  Total failed: %d", stats.FailedCount)
	t.Logf("  Real failures: %d", stats.RealFailedCount)
	t.Logf("  Shutdown failures: %d", stats.ShutdownFailedCount)

	// Verify the counting invariant: totalFailed == realFailed + shutdownFailed
	expectedTotal := stats.RealFailedCount + stats.ShutdownFailedCount
	if stats.FailedCount != expectedTotal {
		t.Errorf("❌ Counting invariant violation: totalFailed=%d, realFailed+shutdownFailed=%d", 
			stats.FailedCount, expectedTotal)
	} else {
		t.Logf("✅ Counting invariant satisfied: totalFailed=%d == realFailed=%d + shutdownFailed=%d", 
			stats.FailedCount, stats.RealFailedCount, stats.ShutdownFailedCount)
	}

	// Verify that we distinguish between shutdown failures and real failures
	if stats.ShutdownFailedCount > 0 {
		t.Logf("✅ Successfully detected %d shutdown failures", stats.ShutdownFailedCount)
	}

	// In a graceful shutdown scenario, we should expect:
	// 1. Some files may have completed processing
	// 2. Some files may have failed due to context cancellation (shutdown failures)
	// 3. Real failures should be 0 (unless there are actual validation/processing errors)
	
	// The key test: exit code should be 0 even if there are shutdown failures
	if stats.RealFailedCount == 0 {
		t.Logf("✅ No real failures detected - exit code should be 0")
	} else {
		t.Errorf("❌ Unexpected real failures: %d - exit code would be 8", stats.RealFailedCount)
	}
}

func TestShutdownFailureDetection(t *testing.T) {
	tests := []struct {
		name              string
		shuttingDown      bool
		err               error
		expectedShutdown  bool
		description       string
	}{
		{
			name:              "no_error",
			shuttingDown:      true,
			err:               nil,
			expectedShutdown:  false,
			description:       "No error should never be classified as shutdown failure",
		},
		{
			name:              "context_canceled_during_shutdown", 
			shuttingDown:      true,
			err:               context.Canceled,
			expectedShutdown:  true,
			description:       "Context cancellation during shutdown should be classified as shutdown failure",
		},
		{
			name:              "context_canceled_without_shutdown",
			shuttingDown:      false,
			err:               context.Canceled,
			expectedShutdown:  false,
			description:       "Context cancellation without shutdown should NOT be classified as shutdown failure",
		},
		{
			name:              "context_deadline_during_shutdown",
			shuttingDown:      true,
			err:               context.DeadlineExceeded,
			expectedShutdown:  true,
			description:       "Context timeout during shutdown should be classified as shutdown failure",
		},
		{
			name:              "context_deadline_without_shutdown",
			shuttingDown:      false,
			err:               context.DeadlineExceeded,
			expectedShutdown:  false,
			description:       "Context timeout without shutdown should NOT be classified as shutdown failure",
		},
		{
			name:              "signal_killed_during_shutdown",
			shuttingDown:      true,
			err:               createMockExitError(137), // SIGKILL exit code
			expectedShutdown:  true,
			description:       "SIGKILL during shutdown should be classified as shutdown failure",
		},
		{
			name:              "signal_killed_without_shutdown",
			shuttingDown:      false,
			err:               createMockExitError(137), // SIGKILL exit code
			expectedShutdown:  false,
			description:       "SIGKILL without shutdown should NOT be classified as shutdown failure",
		},
		{
			name:              "sigterm_exit_code_during_shutdown",
			shuttingDown:      true,
			err:               createMockExitError(143), // SIGTERM exit code
			expectedShutdown:  true,
			description:       "Exit code 143 (SIGTERM) during shutdown should be classified as shutdown failure",
		},
		{
			name:              "signal_message_during_shutdown",
			shuttingDown:      true,
			err:               fmt.Errorf("process terminated: signal: killed"),
			expectedShutdown:  true,
			description:       "Error message containing 'signal: killed' during shutdown should be classified as shutdown failure",
		},
		{
			name:              "real_failure_during_shutdown",
			shuttingDown:      true,
			err:               fmt.Errorf("validation failed: invalid schema"),
			expectedShutdown:  false,
			description:       "Real validation error during shutdown should NOT be classified as shutdown failure",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Test: %s", tc.description)
			
			// Test the centralized IsShutdownFailure function
			result := loop.IsShutdownFailure(tc.shuttingDown, tc.err)
			
			if result != tc.expectedShutdown {
				t.Errorf("Expected shutdown failure detection to be %v, got %v for error: %v (shuttingDown=%v)", 
					tc.expectedShutdown, result, tc.err, tc.shuttingDown)
			} else {
				t.Logf("✅ Correct classification: shuttingDown=%v, err=%v → isShutdownFailure=%v", 
					tc.shuttingDown, tc.err, result)
			}
		})
	}
}

// createMockExitError creates a mock *exec.ExitError with the specified exit code
func createMockExitError(exitCode int) error {
	// Create a short-lived process that will exit with the specified code
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// On Windows, use cmd /c exit with the specified code
		cmd = exec.Command("cmd", "/c", fmt.Sprintf("exit %d", exitCode))
	} else {
		// On Unix, use sh -c exit with the specified code
		cmd = exec.Command("sh", "-c", fmt.Sprintf("exit %d", exitCode))
	}
	
	// Run and wait for the process to complete with the exit code
	err := cmd.Run()
	
	// This should return an *exec.ExitError with the specified exit code
	return err
}

// TestWatcher_GracefulShutdown_DrainsWithoutHang tests that the watcher properly drains
// work during graceful shutdown without hanging or timing out
func TestWatcher_GracefulShutdown_DrainsWithoutHang(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")
	outDir := filepath.Join(tempDir, "out")
	
	err := os.MkdirAll(handoffDir, 0755)
	require.NoError(t, err)
	
	err = os.MkdirAll(outDir, 0755)
	require.NoError(t, err)
	
	// Create mock porch executable that processes files quickly
	mockPorchPath := createMockPorchGraceful(t, tempDir, "success")
	
	// Create N > maxWorkers intents to ensure queue draining
	numIntents := 12  // More than typical worker count
	maxWorkers := 3   // Smaller worker count to force queuing
	
	// Create watcher config with once=true for immediate shutdown test
	config := loop.Config{
		PorchPath:   mockPorchPath,
		Mode:        "direct",
		OutDir:      outDir, 
		Once:        true,  // Critical: this triggers immediate shutdown after processing
		MaxWorkers:  maxWorkers,
		DebounceDur: 50 * time.Millisecond,
	}
	
	watcher, err := loop.NewWatcher(handoffDir, config)
	require.NoError(t, err)
	defer func() { _ = watcher.Close() }()
	
	// Create multiple intent files
	intentContent := `{
	"intent_type": "scaling",
	"target": "test-deployment-drain",
	"namespace": "default",
	"replicas": 5,
	"source": "test"
}`
	
	// Preallocate slice with expected capacity for performance
	createdFiles := make([]string, 0, numIntents)
	for i := 0; i < numIntents; i++ {
		// Use proper intent file naming convention: "intent-" prefix required
		filename := fmt.Sprintf("intent-drain-%03d.json", i)
		filePath := filepath.Join(handoffDir, filename)
		err := os.WriteFile(filePath, []byte(intentContent), 0644)
		require.NoError(t, err)
		createdFiles = append(createdFiles, filename)
	}
	
	// Verify files were created
	entries, err := os.ReadDir(handoffDir)
	require.NoError(t, err)
	t.Logf("Created %d intent files, directory contains %d entries:", numIntents, len(entries))
	for _, entry := range entries {
		t.Logf("  - %s", entry.Name())
	}
	
	// Start watcher processing - this should immediately start processing files
	// and then drain without hanging because once=true
	startTime := time.Now()
	done := make(chan error, 1)
	
	go func() {
		done <- watcher.Start()
	}()
	
	// Watcher should complete processing and exit cleanly due to once=true
	// Allow reasonable time for N files to process, but not too long
	maxExpectedTime := 15 * time.Second
	
	select {
	case err := <-done:
		processingDuration := time.Since(startTime)
		t.Logf("✅ Watcher completed in %v (expected < %v)", processingDuration, maxExpectedTime)
		
		if err != nil {
			t.Logf("Watcher completed with error (may be expected): %v", err)
		}
		
		// Verify processing completed within reasonable time
		if processingDuration > maxExpectedTime {
			t.Errorf("❌ Processing took too long: %v > %v", processingDuration, maxExpectedTime)
		}
		
	case <-time.After(maxExpectedTime):
		t.Fatal("❌ Timeout: watcher failed to drain and complete within expected time")
	}
	
	// Verify final state
	stats, err := watcher.GetStats()
	require.NoError(t, err)
	
	t.Logf("Final processing statistics:")
	t.Logf("  Total processed: %d", stats.ProcessedCount)
	t.Logf("  Total failed: %d", stats.FailedCount)
	t.Logf("  Real failures: %d", stats.RealFailedCount)
	t.Logf("  Shutdown failures: %d", stats.ShutdownFailedCount)
	t.Logf("  Total queue size was: %d intents", numIntents)
	
	// Verify the queue drained properly
	totalProcessed := stats.ProcessedCount + stats.FailedCount
	if totalProcessed == 0 {
		t.Error("❌ No files were processed - queue may not have drained")
	}
	
	// In once mode with proper draining, most/all files should be processed or failed
	// Allow some tolerance since exact behavior depends on timing
	minExpectedProcessing := numIntents / 2  // At least half should be processed
	if totalProcessed < minExpectedProcessing {
		t.Errorf("❌ Too few files processed: got %d, expected at least %d", 
			totalProcessed, minExpectedProcessing)
	}
	
	// Verify the counting invariant
	expectedTotal := stats.RealFailedCount + stats.ShutdownFailedCount
	if stats.FailedCount != expectedTotal {
		t.Errorf("❌ Counting invariant violation: totalFailed=%d != realFailed=%d + shutdownFailed=%d", 
			stats.FailedCount, stats.RealFailedCount, stats.ShutdownFailedCount)
	} else {
		t.Logf("✅ Counting invariant satisfied: %d = %d + %d", 
			stats.FailedCount, stats.RealFailedCount, stats.ShutdownFailedCount)
	}
	
	// Test key assertions:
	// 1. No timeout while draining (verified above)
	// 2. Queue processes without hanging (verified by completion)
	// 3. Tasks complete or fail consistently (verified by counting invariant)
	
	t.Logf("✅ Graceful shutdown drain test completed successfully")
}