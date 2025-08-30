package loop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nephio-project/nephoran-intent-operator/internal/ingest"
)

// MockValidator implements a test validator
type MockValidator struct {
	shouldFail bool
	failMsg    string
}

func (m *MockValidator) ValidateBytes(data []byte) (*ingest.Intent, error) {
	if m.shouldFail {
		return nil, errors.New(m.failMsg)
	}

	var intent ingest.Intent
	if err := json.Unmarshal(data, &intent); err != nil {
		return nil, err
	}

	// Basic validation
	if intent.IntentType == "" || intent.Target == "" {
		return nil, errors.New("missing required fields")
	}

	return &intent, nil
}

// TestProcessorBasic tests basic processor functionality
func TestProcessorBasic(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")
	errorDir := filepath.Join(tempDir, "errors")

	// Create directories
	os.MkdirAll(handoffDir, 0755)
	os.MkdirAll(errorDir, 0755)

	// Create test configuration
	config := &ProcessorConfig{
		HandoffDir:    handoffDir,
		ErrorDir:      errorDir,
		PorchMode:     "structured",
		BatchSize:     2,
		BatchInterval: 100 * time.Millisecond,
		MaxRetries:    1,
	}

	// Create mock validator
	validator := &MockValidator{shouldFail: false}

	// Track porch submissions with thread-safe access
	var submittedIntents []*ingest.Intent
	var submittedMu sync.Mutex
	mockPorchFunc := func(ctx context.Context, intent *ingest.Intent, mode string) error {
		submittedMu.Lock()
		submittedIntents = append(submittedIntents, intent)
		submittedMu.Unlock()
		return nil
	}

	// Create processor
	processor, err := NewProcessor(config, validator, mockPorchFunc)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Start batch processor
	processor.StartBatchProcessor()
	defer processor.Stop()

	// Test data
	validIntent := ingest.Intent{
		IntentType: "scaling",
		Target:     "test-deployment",
		Namespace:  "default",
		Replicas:   3,
	}

	// Write test file
	intentData, _ := json.Marshal(validIntent)
	testFile := filepath.Join(handoffDir, "intent-test1.json")
	if err := os.WriteFile(testFile, intentData, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Process file
	if err := processor.ProcessFile(testFile); err != nil {
		t.Errorf("ProcessFile failed: %v", err)
	}

	// Wait for batch processing
	time.Sleep(200 * time.Millisecond)

	// Verify submission with thread-safe access
	submittedMu.Lock()
	submittedCount := len(submittedIntents)
	submittedMu.Unlock()
	if submittedCount != 1 {
		t.Errorf("Expected 1 submitted intent, got %d", submittedCount)
	}

	// Verify idempotency - process same file again
	if err := processor.ProcessFile(testFile); err != nil {
		t.Errorf("Second ProcessFile failed: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Should still be 1 (idempotent) with thread-safe access
	submittedMu.Lock()
	submittedCount = len(submittedIntents)
	submittedMu.Unlock()
	if submittedCount != 1 {
		t.Errorf("Expected 1 submitted intent (idempotent), got %d", submittedCount)
	}
}

// TestProcessorValidationError tests validation error handling
func TestProcessorValidationError(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")
	errorDir := filepath.Join(tempDir, "errors")

	// Create directories
	os.MkdirAll(handoffDir, 0755)
	os.MkdirAll(errorDir, 0755)

	// Create test configuration
	config := &ProcessorConfig{
		HandoffDir:    handoffDir,
		ErrorDir:      errorDir,
		PorchMode:     "direct",
		BatchSize:     1,
		BatchInterval: 100 * time.Millisecond,
		MaxRetries:    1,
	}

	// Create failing validator
	validator := &MockValidator{
		shouldFail: true,
		failMsg:    "validation error",
	}

	// Mock porch function (should not be called) with thread-safe access
	var porchCalled bool
	var porchMu sync.Mutex
	mockPorchFunc := func(ctx context.Context, intent *ingest.Intent, mode string) error {
		porchMu.Lock()
		porchCalled = true
		porchMu.Unlock()
		return nil
	}

	// Create processor
	processor, err := NewProcessor(config, validator, mockPorchFunc)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Start batch processor
	processor.StartBatchProcessor()
	defer processor.Stop()

	// Write invalid test file
	testFile := filepath.Join(handoffDir, "intent-invalid.json")
	if err := os.WriteFile(testFile, []byte(`{"invalid": "data"}`), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Process file
	processor.ProcessFile(testFile)

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Verify porch was not called with thread-safe access
	porchMu.Lock()
	wasPortchCalled := porchCalled
	porchMu.Unlock()
	if wasPortchCalled {
		t.Error("Porch function should not have been called for invalid intent")
	}

	// Check error file was created
	entries, err := os.ReadDir(errorDir)
	if err != nil {
		t.Fatalf("Failed to read error dir: %v", err)
	}

	if len(entries) < 2 { // Should have .error and .json files
		t.Errorf("Expected error files to be created, found %d files", len(entries))
	}
}

// TestProcessorPorchError tests porch submission error handling
func TestProcessorPorchError(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")
	errorDir := filepath.Join(tempDir, "errors")

	// Create directories
	os.MkdirAll(handoffDir, 0755)
	os.MkdirAll(errorDir, 0755)

	// Create test configuration
	config := &ProcessorConfig{
		HandoffDir:    handoffDir,
		ErrorDir:      errorDir,
		PorchMode:     "structured",
		BatchSize:     1,
		BatchInterval: 100 * time.Millisecond,
		MaxRetries:    2,
	}

	// Create mock validator
	validator := &MockValidator{shouldFail: false}

	// Mock porch function that fails with thread-safe access
	var submitAttempts int
	var attemptsMu sync.Mutex
	mockPorchFunc := func(ctx context.Context, intent *ingest.Intent, mode string) error {
		attemptsMu.Lock()
		submitAttempts++
		attemptsMu.Unlock()
		return errors.New("porch submission failed")
	}

	// Create processor
	processor, err := NewProcessor(config, validator, mockPorchFunc)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Start batch processor
	processor.StartBatchProcessor()
	defer processor.Stop()

	// Test data
	validIntent := ingest.Intent{
		IntentType: "scaling",
		Target:     "test-deployment",
		Namespace:  "default",
		Replicas:   3,
	}

	// Write test file
	intentData, _ := json.Marshal(validIntent)
	testFile := filepath.Join(handoffDir, "intent-porch-fail.json")
	if err := os.WriteFile(testFile, intentData, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Process file
	processor.ProcessFile(testFile)

	// Wait for processing and retries
	time.Sleep(3 * time.Second)

	// Verify retries happened with thread-safe access
	attemptsMu.Lock()
	finalAttempts := submitAttempts
	attemptsMu.Unlock()
	if finalAttempts != config.MaxRetries {
		t.Errorf("Expected %d submit attempts, got %d", config.MaxRetries, finalAttempts)
	}

	// Check error file was created
	entries, err := os.ReadDir(errorDir)
	if err != nil {
		t.Fatalf("Failed to read error dir: %v", err)
	}

	if len(entries) < 2 { // Should have .error and .json files
		t.Errorf("Expected error files to be created, found %d files", len(entries))
	}
}

// TestProcessorBatching tests batch processing
func TestProcessorBatching(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")
	errorDir := filepath.Join(tempDir, "errors")

	// Create directories
	os.MkdirAll(handoffDir, 0755)
	os.MkdirAll(errorDir, 0755)

	// Create test configuration
	config := &ProcessorConfig{
		HandoffDir:    handoffDir,
		ErrorDir:      errorDir,
		PorchMode:     "structured",
		BatchSize:     3,
		BatchInterval: 500 * time.Millisecond,
		MaxRetries:    1,
	}

	// Create mock validator
	validator := &MockValidator{shouldFail: false}

	// Track batch submissions with atomic counters and proper synchronization
	var totalProcessed int64
	var firstBatchDone = make(chan struct{})
	var allDone = make(chan struct{})
	var firstBatchOnce sync.Once
	var allDoneOnce sync.Once

	mockPorchFunc := func(ctx context.Context, intent *ingest.Intent, mode string) error {
		count := atomic.AddInt64(&totalProcessed, 1)
		// Signal when first batch (3 files) is complete
		if count == 3 {
			firstBatchOnce.Do(func() {
				close(firstBatchDone)
			})
		}
		// Signal when all files (5) are complete
		if count == 5 {
			allDoneOnce.Do(func() {
				close(allDone)
			})
		}
		return nil
	}

	// Create processor
	processor, err := NewProcessor(config, validator, mockPorchFunc)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Start batch processor and wait for it to be ready
	processor.StartBatchProcessor()
	<-processor.coordReady // Wait for coordinator to start
	defer processor.Stop()

	// Create and submit 5 test files
	for i := 0; i < 5; i++ {
		intent := ingest.Intent{
			IntentType: "scaling",
			Target:     "test-deployment",
			Namespace:  "default",
			Replicas:   i + 1,
		}

		intentData, _ := json.Marshal(intent)
		testFile := filepath.Join(handoffDir, fmt.Sprintf("intent-%d.json", i))
		if err := os.WriteFile(testFile, intentData, 0644); err != nil {
			t.Fatalf("Failed to write test file %d: %v", i, err)
		}

		if err := processor.ProcessFile(testFile); err != nil {
			t.Errorf("Failed to process file %d: %v", i, err)
		}
	}

	// Wait for first batch (3 files) to complete
	select {
	case <-firstBatchDone:
		// First batch completed successfully
		count := atomic.LoadInt64(&totalProcessed)
		if count != 3 {
			t.Errorf("Expected first batch to process 3 files, got %d", count)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for first batch to complete")
	}

	// Wait for all remaining files (5 total) to complete using channel signaling
	// This eliminates the race condition by using proper channel synchronization
	select {
	case <-allDone:
		// All files processed successfully
		count := atomic.LoadInt64(&totalProcessed)
		if count != 5 {
			t.Errorf("Expected all 5 files to be processed, got %d", count)
		}
	case <-time.After(2 * time.Second):
		count := atomic.LoadInt64(&totalProcessed)
		t.Fatalf("Timeout waiting for all files to be processed. Got %d, expected 5", count)
	}
}
