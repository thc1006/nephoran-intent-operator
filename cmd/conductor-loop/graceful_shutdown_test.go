package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"

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
	defer watcher.Close()

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
			err:               createMockExitError(syscall.SIGKILL),
			expectedShutdown:  true,
			description:       "SIGKILL during shutdown should be classified as shutdown failure",
		},
		{
			name:              "signal_killed_without_shutdown",
			shuttingDown:      false,
			err:               createMockExitError(syscall.SIGKILL),
			expectedShutdown:  false,
			description:       "SIGKILL without shutdown should NOT be classified as shutdown failure",
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

// createMockExitError creates a mock error that will be classified correctly by IsShutdownFailure
func createMockExitError(signal syscall.Signal) error {
	// For testing purposes, we'll use a real exit error from a terminated process
	// This is the most reliable way to get proper signal information
	return createRealExitError(signal)
}

// createRealExitError creates a real *exec.ExitError for testing signal scenarios
func createRealExitError(signal syscall.Signal) error {
	// Create a short-lived process that we can kill with the specified signal
	// Use a simple command that will run long enough for us to signal it
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// On Windows, use timeout command
		cmd = exec.Command("timeout", "10")
	} else {
		// On Unix, use sleep command
		cmd = exec.Command("sleep", "10")  
	}
	
	// Start the process
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start test process: %w", err)
	}
	
	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)
	
	// Kill it with the specified signal
	if runtime.GOOS == "windows" {
		// On Windows, just kill the process (no specific signal support)
		cmd.Process.Kill()
	} else {
		// On Unix, send the specific signal
		cmd.Process.Signal(signal)
	}
	
	// Wait for it to complete and capture the exit error
	waitErr := cmd.Wait()
	
	// Return the exit error which should contain the signal information
	return waitErr
}