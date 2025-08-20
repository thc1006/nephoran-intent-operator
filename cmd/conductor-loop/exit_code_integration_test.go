package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/internal/ingest"
	"github.com/thc1006/nephoran-intent-operator/internal/loop"
)

func TestExitCodeLogic(t *testing.T) {
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

	// Create mock porch that fails immediately
	var fastFailPorchPath string
	if runtime.GOOS == "windows" {
		fastFailPorchPath = filepath.Join(tempDir, "failing-porch.bat")
		mockScript := "@echo off\necho Mock porch failing...\nexit /b 1\n"
		err = os.WriteFile(fastFailPorchPath, []byte(mockScript), 0755)
	} else {
		fastFailPorchPath = filepath.Join(tempDir, "failing-porch.sh")
		mockScript := "#!/bin/bash\necho 'Mock porch failing...'\nexit 1\n"
		err = os.WriteFile(fastFailPorchPath, []byte(mockScript), 0755)
	}
	if err != nil {
		t.Fatalf("Failed to create failing mock porch: %v", err)
	}

	// Create slow mock porch that takes time (for shutdown test)
	var slowPorchPath string
	if runtime.GOOS == "windows" {
		slowPorchPath = filepath.Join(tempDir, "slow-porch.bat")
		// Use powershell for more reliable delay
		slowScript := "@echo off\necho Mock porch processing slowly...\npowershell -Command \"Start-Sleep -Seconds 3\"\necho Mock porch failing after delay...\nexit /b 1\n"
		err = os.WriteFile(slowPorchPath, []byte(slowScript), 0755)
	} else {
		slowPorchPath = filepath.Join(tempDir, "slow-porch.sh")
		slowScript := "#!/bin/bash\necho 'Mock porch processing slowly...'\nsleep 3\necho 'Mock porch failing after delay...'\nexit 1\n"
		err = os.WriteFile(slowPorchPath, []byte(slowScript), 0755)
	}
	if err != nil {
		t.Fatalf("Failed to create slow mock porch: %v", err)
	}

	tests := []struct {
		name                string
		useProcessor        bool
		gracefulShutdown    bool
		expectedExitCode    int
		expectedMessage     string
	}{
		{
			name:             "real failure without shutdown",
			useProcessor:     false,
			gracefulShutdown: false,
			expectedExitCode: 8,
			expectedMessage:  "real failures",
		},
		{
			name:             "shutdown failure should give exit code 0",
			useProcessor:     false,
			gracefulShutdown: true,
			expectedExitCode: 0,
			expectedMessage:  "shutdown failures",
		},
		// TODO: Add processor approach tests
		// The processor approach needs "once" mode implementation to test properly
		// Currently it runs continuously and doesn't exit after processing files
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create fresh directories for each test
			testHandoffDir := filepath.Join(tempDir, tc.name, "handoff")
			testOutDir := filepath.Join(tempDir, tc.name, "out")
			
			err := os.MkdirAll(testHandoffDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create test handoff directory: %v", err)
			}

			err = os.MkdirAll(testOutDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create test output directory: %v", err)
			}

			// Create intent file that will fail
			intentFile := filepath.Join(testHandoffDir, "intent-test.json")
			intentContent := `{
	"intent_type": "scaling",
	"target": "test-deployment",
	"namespace": "default",
	"replicas": 3,
	"source": "test"
}`
			err = os.WriteFile(intentFile, []byte(intentContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create intent file: %v", err)
			}

			// Choose appropriate porch path based on test scenario
			var porchPath string
			if tc.gracefulShutdown {
				porchPath = slowPorchPath // Use slow porch for shutdown testing
			} else {
				porchPath = fastFailPorchPath // Use fast failing porch for regular testing
			}

			var watcher *loop.Watcher
			if tc.useProcessor {
				// New processor approach
				errorDir := filepath.Join(testOutDir, "errors")
				
				// Create validator (mock for test)
				validator := &mockValidator{}
				
				// Create processor with mock porch function that fails
				processorConfig := &loop.ProcessorConfig{
					HandoffDir:    testHandoffDir,
					ErrorDir:      errorDir,
					PorchMode:     "direct",
					BatchSize:     1,
					BatchInterval: 100 * time.Millisecond,
					MaxRetries:    1,
				}
				
				// Create mock porch function that always fails
				mockPorchFunc := func(ctx context.Context, intent *ingest.Intent, mode string) error {
					if tc.gracefulShutdown {
						time.Sleep(3 * time.Second) // Simulate slow processing
					}
					return fmt.Errorf("mock porch failure")
				}
				
				processor, err := loop.NewProcessor(processorConfig, validator, mockPorchFunc)
				if err != nil {
					t.Fatalf("Failed to create processor: %v", err)
				}
				
				processor.StartBatchProcessor()
				defer processor.Stop()
				
				watcher, err = loop.NewWatcherWithProcessor(testHandoffDir, processor)
				if err != nil {
					t.Fatalf("Failed to create watcher with processor: %v", err)
				}
			} else {
				// Legacy approach
				config := loop.Config{
					PorchPath:   porchPath,
					Mode:        "direct",
					OutDir:      testOutDir,
					Once:        true,
					MaxWorkers:  1,
					DebounceDur: 50 * time.Millisecond,
				}

				watcher, err = loop.NewWatcher(testHandoffDir, config)
				if err != nil {
					t.Fatalf("Failed to create watcher: %v", err)
				}
			}

			// If testing graceful shutdown, trigger it during processing
			if tc.gracefulShutdown {
				go func() {
					time.Sleep(500 * time.Millisecond) // Let processing start (wait for slow command)
					t.Logf("Triggering graceful shutdown...")
					watcher.Close() // Trigger graceful shutdown
				}()
			}

			// Start processing
			err = watcher.Start()
			if err != nil {
				t.Logf("Watcher returned error (may be expected): %v", err)
			}

			// Get stats and verify exit code logic
			stats, err := watcher.GetStats()
			if err != nil {
				t.Fatalf("Failed to get stats: %v", err)
			}

			t.Logf("Test %s - Stats: processed=%d, failed=%d, real_failed=%d, shutdown_failed=%d",
				tc.name, stats.ProcessedCount, stats.FailedCount, stats.RealFailedCount, stats.ShutdownFailedCount)

			// Simulate the exit code logic from main.go
			var actualExitCode int
			if stats.RealFailedCount > 0 {
				actualExitCode = 8
				t.Logf("Would exit with code 8 due to %d real failures", stats.RealFailedCount)
			} else if stats.ShutdownFailedCount > 0 {
				actualExitCode = 0
				t.Logf("Would exit with code 0 despite %d shutdown failures", stats.ShutdownFailedCount)
			} else {
				actualExitCode = 0
				t.Logf("Would exit with code 0 - all files processed successfully")
			}

			// Verify the expected exit code
			if actualExitCode != tc.expectedExitCode {
				t.Errorf("Expected exit code %d, got %d", tc.expectedExitCode, actualExitCode)
			}

			// Verify we have failures of the expected type
			if tc.gracefulShutdown && stats.ShutdownFailedCount == 0 {
				t.Errorf("Expected shutdown failures during graceful shutdown, got none")
			} else if !tc.gracefulShutdown && stats.RealFailedCount == 0 {
				t.Errorf("Expected real failures without graceful shutdown, got none")
			}

			watcher.Close()
		})
	}
}

// mockValidator is a simple mock validator for testing
type mockValidator struct{}

func (v *mockValidator) ValidateBytes(data []byte) (*ingest.Intent, error) {
	// Simple mock validation - always returns a basic intent
	return &ingest.Intent{
		IntentType: "scaling",
		Target:     "test-deployment",
		Namespace:  "default",
		Replicas:   3,
		Source:     "test",
	}, nil
}