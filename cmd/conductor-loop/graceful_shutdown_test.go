package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/internal/loop"
)

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

	// Create mock porch executable (simple version for testing)
	var mockPorchPath string
	if runtime.GOOS == "windows" {
		mockPorchPath = filepath.Join(tempDir, "mock-porch.bat")
		mockScript := "@echo off\necho Mock porch processing...\ntimeout /t 1 >nul 2>&1\necho Mock porch completed\nexit /b 0\n"
		err = os.WriteFile(mockPorchPath, []byte(mockScript), 0755)
	} else {
		mockPorchPath = filepath.Join(tempDir, "mock-porch.sh")
		mockScript := "#!/bin/bash\necho 'Mock porch processing...'\nsleep 1\necho 'Mock porch completed'\nexit 0\n"
		err = os.WriteFile(mockPorchPath, []byte(mockScript), 0755)
	}
	if err != nil {
		t.Fatalf("Failed to create mock porch: %v", err)
	}

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
	go func() {
		done <- watcher.Start()
	}()

	// Let processing start, then simulate graceful shutdown
	time.Sleep(100 * time.Millisecond)
	
	// Trigger graceful shutdown
	watcher.Close()

	// Wait for processing to complete
	select {
	case err := <-done:
		if err != nil {
			t.Logf("Watcher returned error (expected during shutdown): %v", err)
		}
	case <-time.After(5 * time.Second):
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

	// Verify total failed count is the sum of real and shutdown failures
	expectedTotal := stats.RealFailedCount + stats.ShutdownFailedCount
	if stats.FailedCount != expectedTotal {
		t.Errorf("Failed count mismatch: total=%d, real+shutdown=%d", 
			stats.FailedCount, expectedTotal)
	}
}

func TestShutdownFailureDetection(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")
	
	err := os.MkdirAll(handoffDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create handoff directory: %v", err)
	}

	// Create watcher
	config := loop.Config{
		PorchPath:   "mock-porch",
		Mode:        "direct",
		OutDir:      tempDir,
		MaxWorkers:  1,
		DebounceDur: 100 * time.Millisecond,
	}

	watcher, err := loop.NewWatcher(handoffDir, config)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	// Test different error scenarios
	testCases := []struct {
		name              string
		err               error
		errorMsg          string
		gracefulShutdown  bool
		expectedShutdown  bool
	}{
		{
			name:             "context canceled during shutdown",
			err:              context.Canceled,
			errorMsg:         "context canceled",
			gracefulShutdown: true,
			expectedShutdown: true,
		},
		{
			name:             "signal killed during shutdown",
			err:              nil,
			errorMsg:         "signal: killed",
			gracefulShutdown: true,
			expectedShutdown: true,
		},
		{
			name:             "normal validation error",
			err:              nil,
			errorMsg:         "invalid JSON format",
			gracefulShutdown: false,
			expectedShutdown: false,
		},
		{
			name:             "context canceled without shutdown",
			err:              context.Canceled,
			errorMsg:         "context canceled",
			gracefulShutdown: false,
			expectedShutdown: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new watcher instance for clean testing
			newWatcher, err := loop.NewWatcher(handoffDir, config)
			if err != nil {
				t.Fatalf("Failed to create new watcher: %v", err)
			}
			defer newWatcher.Close()
			
			// Set graceful shutdown state if needed
			if tc.gracefulShutdown {
				newWatcher.Close() // This triggers graceful shutdown state
			}
			
			result := newWatcher.IsShutdownFailure(tc.err, tc.errorMsg)
			if result != tc.expectedShutdown {
				t.Errorf("Expected shutdown failure detection to be %v, got %v", 
					tc.expectedShutdown, result)
			}
		})
	}
}