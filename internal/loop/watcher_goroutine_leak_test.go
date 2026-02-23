package loop

import (
	"runtime"
	"testing"
	"time"
)

// TestWatcher_NoGoroutineLeak verifies that background goroutines are properly cleaned up.
func TestWatcher_NoGoroutineLeak(t *testing.T) {
	// Force GC to clean up any existing goroutines
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	// Get baseline goroutine count
	before := runtime.NumGoroutine()
	t.Logf("Baseline goroutine count: %d", before)

	// Create and destroy watchers multiple times
	for i := 0; i < 5; i++ {
		t.Logf("Iteration %d: Creating watcher...", i+1)

		// Create temp directory
		tmpDir := t.TempDir()

		// Create watcher
		config := Config{
			MaxWorkers:   2,
			DebounceDur:  100 * time.Millisecond,
			Once:         true, // Run once to avoid infinite loop
			GracePeriod:  5 * time.Second,
			CleanupAfter: 24 * time.Hour,
		}

		watcher, err := NewWatcher(tmpDir, config)
		if err != nil {
			t.Fatalf("Failed to create watcher: %v", err)
		}

		// Start watcher (will process existing files and exit immediately in "once" mode)
		// This should NOT spawn background goroutines since we're in once mode,
		// but let's test the Close() path anyway
		err = watcher.Start()
		if err != nil {
			t.Fatalf("Failed to start watcher: %v", err)
		}

		// Close watcher
		t.Logf("Iteration %d: Closing watcher...", i+1)
		err = watcher.Close()
		if err != nil {
			t.Fatalf("Failed to close watcher: %v", err)
		}

		// Give goroutines time to exit
		time.Sleep(200 * time.Millisecond)

		// Check goroutine count
		current := runtime.NumGoroutine()
		t.Logf("Iteration %d: Current goroutine count: %d (baseline: %d)", i+1, current, before)
	}

	// Force GC again
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	// Get final goroutine count
	after := runtime.NumGoroutine()
	t.Logf("Final goroutine count: %d (baseline: %d)", after, before)

	// Allow for small variance (testing framework goroutines, etc.)
	// but fail if we're leaking goroutines
	if after > before+5 {
		t.Errorf("Goroutine leak detected: before=%d, after=%d, leaked=%d", before, after, after-before)
		t.Logf("Dumping goroutine stack traces...")
		buf := make([]byte, 1<<16)
		stackSize := runtime.Stack(buf, true)
		t.Logf("Goroutine stacks:\n%s", buf[:stackSize])
	} else {
		t.Logf("✅ No goroutine leak detected (diff: %d goroutines)", after-before)
	}
}

// TestWatcher_BackgroundGoroutinesWithContext verifies that background goroutines exit when context is cancelled.
func TestWatcher_BackgroundGoroutinesWithContext(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create watcher with continuous mode (not once)
	config := Config{
		MaxWorkers:   2,
		DebounceDur:  100 * time.Millisecond,
		Once:         false, // Continuous mode
		GracePeriod:  5 * time.Second,
		CleanupAfter: 24 * time.Hour,
	}

	watcher, err := NewWatcher(tmpDir, config)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	// Get baseline goroutine count before starting
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	beforeStart := runtime.NumGoroutine()
	t.Logf("Goroutines before Start(): %d", beforeStart)

	// Start watcher in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- watcher.Start()
	}()

	// Wait for background goroutines to start
	time.Sleep(500 * time.Millisecond)

	afterStart := runtime.NumGoroutine()
	t.Logf("Goroutines after Start(): %d (expected: +3-4 for worker pool + 2 cleanup routines)", afterStart)

	// Expected goroutines:
	// - 2 worker goroutines (MaxWorkers=2)
	// - 1 cleanupRoutine (started in Start() if processor == nil)
	// - 1 fileStateCleanupRoutine (always started)
	// - 1 Start() goroutine itself
	// Total: ~5 additional goroutines
	expectedMin := beforeStart + 3
	expectedMax := beforeStart + 10 // Allow some variance

	if afterStart < expectedMin || afterStart > expectedMax {
		t.Errorf("Unexpected goroutine count after Start(): got %d, expected %d-%d", afterStart, expectedMin, expectedMax)
	}

	// Close watcher
	t.Logf("Closing watcher...")
	err = watcher.Close()
	if err != nil {
		t.Fatalf("Failed to close watcher: %v", err)
	}

	// Wait for goroutines to exit
	time.Sleep(1 * time.Second)
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	afterClose := runtime.NumGoroutine()
	t.Logf("Goroutines after Close(): %d (baseline: %d)", afterClose, beforeStart)

	// Verify goroutines are cleaned up (allow for small variance)
	if afterClose > beforeStart+5 {
		t.Errorf("Goroutines not cleaned up after Close(): baseline=%d, after=%d, leaked=%d",
			beforeStart, afterClose, afterClose-beforeStart)

		// Dump stack traces
		buf := make([]byte, 1<<16)
		stackSize := runtime.Stack(buf, true)
		t.Logf("Goroutine stacks after Close():\n%s", buf[:stackSize])
	} else {
		t.Logf("✅ Background goroutines properly cleaned up (diff: %d goroutines)", afterClose-beforeStart)
	}
}
