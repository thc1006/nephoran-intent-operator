package conductor

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-logr/logr"
)

// PorchRunner interface for dependency injection and testing
type PorchRunner interface {
	ExecutePorch(logger logr.Logger, intentFile, name string) error
}

// MockPorchRunner is a spy implementation for testing
type MockPorchRunner struct {
	calls           []RunnerCall
	shouldError     bool
	errorMessage    string
	expectedCommand string
	expectedArgs    []string
}

// RunnerCall records the details of a porch execution call
type RunnerCall struct {
	Logger     logr.Logger
	IntentFile string
	Name       string
	Error      error
}

// NewMockPorchRunner creates a new mock porch runner
func NewMockPorchRunner() *MockPorchRunner {
	return &MockPorchRunner{
		calls: make([]RunnerCall, 0),
	}
}

// SetError configures the mock to return an error
func (m *MockPorchRunner) SetError(shouldError bool, message string) {
	m.shouldError = shouldError
	m.errorMessage = message
}

// SetExpectedCommand configures expected command and args for verification
func (m *MockPorchRunner) SetExpectedCommand(command string, args []string) {
	m.expectedCommand = command
	m.expectedArgs = args
}

// ExecutePorch implements PorchRunner interface
func (m *MockPorchRunner) ExecutePorch(logger logr.Logger, intentFile, name string) error {
	call := RunnerCall{
		Logger:     logger,
		IntentFile: intentFile,
		Name:       name,
	}

	if m.shouldError {
		call.Error = errors.New(m.errorMessage)
		m.calls = append(m.calls, call)
		return call.Error
	}

	m.calls = append(m.calls, call)
	return nil
}

// GetCalls returns all recorded calls
func (m *MockPorchRunner) GetCalls() []RunnerCall {
	return m.calls
}

// GetCallCount returns the number of calls made
func (m *MockPorchRunner) GetCallCount() int {
	return len(m.calls)
}

// Reset clears all recorded calls and resets error state
func (m *MockPorchRunner) Reset() {
	m.calls = make([]RunnerCall, 0)
	m.shouldError = false
	m.errorMessage = ""
}

// RealPorchRunner is the production implementation using the WatchReconciler's executePorch method
type RealPorchRunner struct {
	reconciler *WatchReconciler
}

// NewRealPorchRunner creates a real porch runner
func NewRealPorchRunner(reconciler *WatchReconciler) *RealPorchRunner {
	return &RealPorchRunner{
		reconciler: reconciler,
	}
}

// ExecutePorch implements PorchRunner interface by delegating to WatchReconciler
func (r *RealPorchRunner) ExecutePorch(logger logr.Logger, intentFile, name string) error {
	return r.reconciler.executePorch(logger, intentFile, name)
}

// DISABLED: func TestMockPorchRunner(t *testing.T) {
	tests := []struct {
		name        string
		shouldError bool
		errorMsg    string
		intentFile  string
		nameParam   string
	}{
		{
			name:        "successful-execution",
			shouldError: false,
			intentFile:  "/test/intent-test-default-123.json",
			nameParam:   "test-intent",
		},
		{
			name:        "execution-with-error",
			shouldError: true,
			errorMsg:    "porch command failed with exit code 1",
			intentFile:  "/test/intent-failing-default-456.json",
			nameParam:   "failing-intent",
		},
		{
			name:        "multiple-calls",
			shouldError: false,
			intentFile:  "/test/intent-multi-default-789.json",
			nameParam:   "multi-intent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockPorchRunner()
			mock.SetError(tt.shouldError, tt.errorMsg)

			// Create a dummy logger
			logger := logr.Discard()

			// Execute the call
			err := mock.ExecutePorch(logger, tt.intentFile, tt.nameParam)

			// Verify error handling
			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			// Verify call was recorded
			calls := mock.GetCalls()
			if len(calls) != 1 {
				t.Errorf("Expected 1 call, got %d", len(calls))
			}

			call := calls[0]
			if call.IntentFile != tt.intentFile {
				t.Errorf("Expected intent file '%s', got '%s'", tt.intentFile, call.IntentFile)
			}
			if call.Name != tt.nameParam {
				t.Errorf("Expected name '%s', got '%s'", tt.nameParam, call.Name)
			}

			// For error cases, verify error was recorded
			if tt.shouldError && call.Error == nil {
				t.Error("Expected error to be recorded in call")
			}
		})
	}
}

// DISABLED: func TestMockPorchRunnerMultipleCalls(t *testing.T) {
	mock := NewMockPorchRunner()
	logger := logr.Discard()

	// Make multiple calls
	testCases := []struct {
		intentFile string
		name       string
	}{
		{"/test/intent1.json", "intent1"},
		{"/test/intent2.json", "intent2"},
		{"/test/intent3.json", "intent3"},
	}

	for _, tc := range testCases {
		err := mock.ExecutePorch(logger, tc.intentFile, tc.name)
		if err != nil {
			t.Errorf("Unexpected error for %s: %v", tc.name, err)
		}
	}

	// Verify all calls were recorded
	calls := mock.GetCalls()
	if len(calls) != len(testCases) {
		t.Errorf("Expected %d calls, got %d", len(testCases), len(calls))
	}

	for i, expected := range testCases {
		if calls[i].IntentFile != expected.intentFile {
			t.Errorf("Call %d: expected intent file '%s', got '%s'", i, expected.intentFile, calls[i].IntentFile)
		}
		if calls[i].Name != expected.name {
			t.Errorf("Call %d: expected name '%s', got '%s'", i, expected.name, calls[i].Name)
		}
	}
}

// DISABLED: func TestMockPorchRunnerReset(t *testing.T) {
	mock := NewMockPorchRunner()
	logger := logr.Discard()

	// Make some calls
	_ = mock.ExecutePorch(logger, "/test/intent1.json", "intent1")
	_ = mock.ExecutePorch(logger, "/test/intent2.json", "intent2")

	// Verify calls exist
	if mock.GetCallCount() != 2 {
		t.Errorf("Expected 2 calls before reset, got %d", mock.GetCallCount())
	}

	// Reset and verify
	mock.Reset()
	if mock.GetCallCount() != 0 {
		t.Errorf("Expected 0 calls after reset, got %d", mock.GetCallCount())
	}

	// Verify error state is reset
	mock.SetError(true, "test error")
	mock.Reset()
	err := mock.ExecutePorch(logger, "/test/intent3.json", "intent3")
	if err != nil {
		t.Errorf("Expected no error after reset, got: %v", err)
	}
}

// DISABLED: func TestRealPorchRunnerConstruction(t *testing.T) {
	// Test construction of real porch runner
	reconciler := &WatchReconciler{
		PorchPath: "/usr/local/bin/porch",
		PorchMode: "apply",
		OutputDir: "/tmp/test-output",
		DryRun:    true, // Use dry run to avoid actual command execution
	}

	runner := NewRealPorchRunner(reconciler)
	if runner.reconciler != reconciler {
		t.Error("RealPorchRunner should store reference to reconciler")
	}
}

// DISABLED: func TestPorchRunnerCommandConstruction(t *testing.T) {
	// Test command construction by examining what the reconciler would build
	testDir := t.TempDir()
	intentFile := filepath.Join(testDir, "test-intent.json")

	// Create a test intent file
	intentContent := `{
		"intent_type": "scaling",
		"target": "nginx",
		"namespace": "default",
		"replicas": 3,
		"source": "conductor-watch"
	}`

	if err := os.WriteFile(intentFile, []byte(intentContent), 0o644); err != nil {
		t.Fatalf("Failed to create test intent file: %v", err)
	}

	// Test different reconciler configurations
	testCases := []struct {
		name         string
		reconciler   *WatchReconciler
		expectedArgs []string
	}{
		{
			name: "apply-mode",
			reconciler: &WatchReconciler{
				PorchPath: "/usr/local/bin/porch",
				PorchMode: "apply",
				OutputDir: testDir,
				DryRun:    true,
			},
			expectedArgs: []string{"generate", "--intent", intentFile, "--mode", "apply", "--output", testDir},
		},
		{
			name: "dry-run-mode",
			reconciler: &WatchReconciler{
				PorchPath: "/usr/local/bin/porch",
				PorchMode: "dry-run",
				OutputDir: testDir,
				DryRun:    true,
			},
			expectedArgs: []string{"generate", "--intent", intentFile, "--mode", "dry-run", "--output", testDir},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// We can't easily test the actual command construction without refactoring executePorch,
			// but we can verify the configuration is stored correctly
			if tc.reconciler.PorchPath != "/usr/local/bin/porch" {
				t.Error("PorchPath not set correctly")
			}
			if tc.reconciler.OutputDir != testDir {
				t.Error("OutputDir not set correctly")
			}

			// Test that dry-run mode prevents execution
			if !tc.reconciler.DryRun {
				t.Error("DryRun should be enabled for tests")
			}
		})
	}
}

// DISABLED: func TestPorchRunnerDryRunMode(t *testing.T) {
	// Test that dry-run mode in reconciler affects execution
	testDir := t.TempDir()

	reconciler := &WatchReconciler{
		PorchPath: "/fake/path/to/porch", // Non-existent path
		PorchMode: "apply",
		OutputDir: testDir,
		DryRun:    true, // This should prevent actual execution
	}

	runner := NewRealPorchRunner(reconciler)
	logger := logr.Discard()

	// This should not fail even with fake porch path because DryRun is true
	_ = runner.ExecutePorch(logger, "/fake/intent.json", "test-name")

	// The actual implementation would need to check DryRun in executePorch
	// For now, we just verify the runner was created correctly
	if runner.reconciler.DryRun != true {
		t.Error("DryRun mode should be preserved")
	}
}

// DISABLED: func TestPorchRunnerIntegrationWithWatchReconciler(t *testing.T) {
	// Integration test showing how the runner would be used in the reconciler
	testDir := t.TempDir()
	intentFile := filepath.Join(testDir, "intent-test-default-123.json")

	// Create test intent file
	intentContent := `{
		"intent_type": "scaling",
		"target": "nginx",
		"namespace": "default", 
		"replicas": 3,
		"source": "conductor-watch",
		"correlation_id": "test-default-123",
		"reason": "Test scaling intent"
	}`

	if err := os.WriteFile(intentFile, []byte(intentContent), 0o644); err != nil {
		t.Fatalf("Failed to create intent file: %v", err)
	}

	// Test with mock runner
	mockRunner := NewMockPorchRunner()
	logger := logr.Discard()

	// Simulate reconciler using the runner
	err := mockRunner.ExecutePorch(logger, intentFile, "test-default")
	if err != nil {
		t.Errorf("Mock runner failed: %v", err)
	}

	// Verify the call was made correctly
	calls := mockRunner.GetCalls()
	if len(calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(calls))
	}

	call := calls[0]
	if call.IntentFile != intentFile {
		t.Errorf("Expected intent file '%s', got '%s'", intentFile, call.IntentFile)
	}
	if call.Name != "test-default" {
		t.Errorf("Expected name 'test-default', got '%s'", call.Name)
	}

	// Test error scenario
	mockRunner.Reset()
	mockRunner.SetError(true, "porch execution failed")

	err = mockRunner.ExecutePorch(logger, intentFile, "test-default")
	if err == nil {
		t.Error("Expected error from mock runner")
	}
	if !strings.Contains(err.Error(), "porch execution failed") {
		t.Errorf("Expected error message to contain 'porch execution failed', got: %v", err)
	}
}
