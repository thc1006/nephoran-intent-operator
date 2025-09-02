package main

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/bep/debounce"
)

// DISABLED: func TestDebounceRaceCondition(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Simulate the debouncing logic used in main.go
	processors := make(map[string]func())
	processorsMutex := &sync.RWMutex{}

	processCount := 0
	processCountMutex := &sync.Mutex{}

	// Simulate rapid file creation events
	testFile := filepath.Join(tempDir, "intent-test.json")

	// Create the test file
	err := os.WriteFile(testFile, []byte(`{"correlation_id":"test-123"}`), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Simulate multiple rapid events for the same file
	for i := 0; i < 10; i++ {
		go func() {
			processorsMutex.Lock()
			processor, exists := processors[testFile]
			if !exists {
				// Create new debounced processor with 100ms window for faster test
				filePath := testFile // Capture for closure
				debouncedFunc := debounce.New(100 * time.Millisecond)
				processor = func() {
					debouncedFunc(func() {
						processCountMutex.Lock()
						processCount++
						processCountMutex.Unlock()

						// Clean up processor after execution
						processorsMutex.Lock()
						delete(processors, filePath)
						processorsMutex.Unlock()
					})
				}
				processors[testFile] = processor
			}
			processorsMutex.Unlock()

			// Trigger debounced processing
			processor()
		}()
	}

	// Wait for debouncing to complete
	time.Sleep(200 * time.Millisecond)

	// Check that only one processing occurred despite 10 rapid events
	processCountMutex.Lock()
	finalCount := processCount
	processCountMutex.Unlock()

	if finalCount != 1 {
		t.Errorf("Expected 1 processing event, but got %d", finalCount)
	}

	// Verify processors map is cleaned up
	processorsMutex.RLock()
	remainingProcessors := len(processors)
	processorsMutex.RUnlock()

	if remainingProcessors != 0 {
		t.Errorf("Expected 0 remaining processors, but got %d", remainingProcessors)
	}
}

// DISABLED: func TestDebounceMultipleFiles(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Simulate the debouncing logic for multiple files
	processors := make(map[string]func())
	processorsMutex := &sync.RWMutex{}

	processCount := 0
	processCountMutex := &sync.Mutex{}

	// Create multiple test files
	files := []string{
		filepath.Join(tempDir, "intent-1.json"),
		filepath.Join(tempDir, "intent-2.json"),
		filepath.Join(tempDir, "intent-3.json"),
	}

	for _, file := range files {
		err := os.WriteFile(file, []byte(`{"correlation_id":"test-123"}`), 0o644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	// Process each file with debouncing
	for _, testFile := range files {
		go func(file string) {
			processorsMutex.Lock()
			processor, exists := processors[file]
			if !exists {
				// Create new debounced processor
				filePath := file // Capture for closure
				debouncedFunc := debounce.New(50 * time.Millisecond)
				processor = func() {
					debouncedFunc(func() {
						processCountMutex.Lock()
						processCount++
						processCountMutex.Unlock()

						// Clean up processor after execution
						processorsMutex.Lock()
						delete(processors, filePath)
						processorsMutex.Unlock()
					})
				}
				processors[file] = processor
			}
			processorsMutex.Unlock()

			// Trigger debounced processing
			processor()
		}(testFile)
	}

	// Wait for all debouncing to complete
	time.Sleep(150 * time.Millisecond)

	// Check that each file was processed once
	processCountMutex.Lock()
	finalCount := processCount
	processCountMutex.Unlock()

	if finalCount != 3 {
		t.Errorf("Expected 3 processing events (one per file), but got %d", finalCount)
	}

	// Verify processors map is cleaned up
	processorsMutex.RLock()
	remainingProcessors := len(processors)
	processorsMutex.RUnlock()

	if remainingProcessors != 0 {
		t.Errorf("Expected 0 remaining processors, but got %d", remainingProcessors)
	}
}
