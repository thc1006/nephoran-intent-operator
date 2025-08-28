package loop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/internal/ingest"
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
	var submittedMu sync.Mutex
	var submittedIntents []*ingest.Intent
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
	if !WaitFor(t, func() bool {
		submittedMu.Lock()
		defer submittedMu.Unlock()
		return len(submittedIntents) == 1
	}, 2*time.Second, "first intent submission") {
		submittedMu.Lock()
		t.Errorf("Expected 1 submitted intent, got %d", len(submittedIntents))
		submittedMu.Unlock()
	}

	// Verify idempotency - process same file again
	if err := processor.ProcessFile(testFile); err != nil {
		t.Errorf("Second ProcessFile failed: %v", err)
	}

	// Give it a moment to ensure no duplicate processing
	time.Sleep(300 * time.Millisecond)

	// Should still be 1 (idempotent)
	submittedMu.Lock()
	if len(submittedIntents) != 1 {
		t.Errorf("Expected 1 submitted intent (idempotent), got %d", len(submittedIntents))
	}
	submittedMu.Unlock()
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

	// Mock porch function (should not be called)
	porchCalled := false
	mockPorchFunc := func(ctx context.Context, intent *ingest.Intent, mode string) error {
		porchCalled = true
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
	if err := processor.ProcessFile(testFile); err != nil {
		t.Errorf("ProcessFile failed: %v", err)
	}

	// Wait for processing
	time.Sleep(300 * time.Millisecond)

	// Verify porch was not called
	if porchCalled {
		t.Error("Porch function should not have been called for invalid intent")
	}

	// Check error file was created
	if !WaitFor(t, func() bool {
		entries, _ := os.ReadDir(errorDir)
		return len(entries) >= 2
	}, 2*time.Second, "error files creation") {
		entries, _ := os.ReadDir(errorDir)
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

	// Mock porch function that fails with thread-safe counter
	var attemptsMu sync.Mutex
	submitAttempts := 0
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
	if err := processor.ProcessFile(testFile); err != nil {
		t.Errorf("ProcessFile failed: %v", err)
	}

	// Wait for processing and retries
	if !WaitFor(t, func() bool {
		attemptsMu.Lock()
		defer attemptsMu.Unlock()
		return submitAttempts >= config.MaxRetries
	}, 4*time.Second, "retry attempts") {
		attemptsMu.Lock()
		t.Errorf("Expected %d submit attempts, got %d", config.MaxRetries, submitAttempts)
		attemptsMu.Unlock()
	}

	// Check error file was created
	if !WaitFor(t, func() bool {
		entries, _ := os.ReadDir(errorDir)
		return len(entries) >= 2
	}, 2*time.Second, "error files creation") {
		entries, _ := os.ReadDir(errorDir)
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

	// Track batch submissions with thread-safe counters
	var batchMu sync.Mutex
	var batchSizes []int
	currentBatch := 0
	mockPorchFunc := func(ctx context.Context, intent *ingest.Intent, mode string) error {
		batchMu.Lock()
		currentBatch++
		batchMu.Unlock()
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

	// Create multiple test files
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
			t.Errorf("ProcessFile failed for file %d: %v", i, err)
		}
		
		// After 3 files, batch should flush
		if i == 2 {
			if !WaitFor(t, func() bool {
				batchMu.Lock()
				defer batchMu.Unlock()
				return currentBatch == 3
			}, 1*time.Second, "batch flush after 3 files") {
				batchMu.Lock()
				t.Errorf("Expected batch to flush after 3 files, got %d", currentBatch)
				batchMu.Unlock()
			}
			batchMu.Lock()
			batchSizes = append(batchSizes, currentBatch)
			currentBatch = 0
			batchMu.Unlock()
		}
	}

	// Wait for interval flush of remaining files
	if !WaitFor(t, func() bool {
		batchMu.Lock()
		defer batchMu.Unlock()
		return currentBatch == 2
	}, 2*time.Second, "interval flush of remaining files") {
		batchMu.Lock()
		t.Errorf("Expected remaining 2 files to be processed, got %d", currentBatch)
		batchMu.Unlock()
	}
}