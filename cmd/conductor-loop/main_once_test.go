package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/nephoran-intent-operator/internal/loop"
)

func TestOnceMode_ExitCodes(t *testing.T) {
	tests := []struct {
		name           string
		setupFiles     func(t *testing.T, handoffDir string)
		expectedFailed int
		expectedExit   int // 0=all ok, 8=some failed
	}{
		{
			name: "all_files_processed_successfully",
			setupFiles: func(t *testing.T, handoffDir string) {
				// Create a valid intent file
				content := `{"intent_type": "scaling", "target": "test", "replicas": 3}`
				require.NoError(t, os.WriteFile(
					filepath.Join(handoffDir, "intent-test1.json"),
					[]byte(content), 0644))
			},
			expectedFailed: 0,
			expectedExit:   0,
		},
		{
			name: "some_files_failed",
			setupFiles: func(t *testing.T, handoffDir string) {
				// Create a valid intent file
				valid := `{"intent_type": "scaling", "target": "test", "replicas": 3}`
				require.NoError(t, os.WriteFile(
					filepath.Join(handoffDir, "intent-valid.json"),
					[]byte(valid), 0644))
				
				// Create an invalid intent file (will fail processing)
				invalid := `{invalid json`
				require.NoError(t, os.WriteFile(
					filepath.Join(handoffDir, "intent-invalid.json"),
					[]byte(invalid), 0644))
			},
			expectedFailed: 1,
			expectedExit:   8,
		},
		{
			name: "no_files_to_process",
			setupFiles: func(t *testing.T, handoffDir string) {
				// No files, directory is empty
			},
			expectedFailed: 0,
			expectedExit:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tempDir := t.TempDir()
			handoffDir := filepath.Join(tempDir, "handoff")
			outDir := filepath.Join(tempDir, "out")
			
			require.NoError(t, os.MkdirAll(handoffDir, 0755))
			require.NoError(t, os.MkdirAll(outDir, 0755))
			
			// Setup test files
			tt.setupFiles(t, handoffDir)
			
			// Create mock porch executable that fails for invalid files
			mockPorch := filepath.Join(tempDir, "mock-porch.bat")
			// This mock will fail if the intent file contains "invalid"
			mockContent := `@echo off
findstr /C:"invalid" "%2" >nul
if %errorlevel% equ 0 (
    echo Error: Invalid intent file
    exit 1
)
exit 0`
			require.NoError(t, os.WriteFile(mockPorch, []byte(mockContent), 0755))
			
			// Create watcher with once mode
			config := loop.Config{
				PorchPath:   mockPorch,
				Mode:        "direct",
				OutDir:      outDir,
				Once:        true,
				DebounceDur: 0, // No debounce for testing
			}
			
			watcher, err := loop.NewWatcher(handoffDir, config)
			require.NoError(t, err)
			defer watcher.Close()
			
			// Run the watcher (it should process and exit immediately in once mode)
			err = watcher.Start()
			assert.NoError(t, err)
			
			// Give a bit of time for processing
			time.Sleep(100 * time.Millisecond)
			
			// Check the stats
			stats, err := watcher.GetStats()
			require.NoError(t, err)
			
			// Verify failed count matches expectation
			assert.Equal(t, tt.expectedFailed, stats.FailedCount,
				"Failed count mismatch. Failed files: %v", stats.FailedFiles)
			
			// Simulate what main.go would do
			var exitCode int
			if stats.FailedCount > 0 {
				exitCode = 8
			} else {
				exitCode = 0
			}
			
			assert.Equal(t, tt.expectedExit, exitCode,
				"Exit code mismatch for test case: %s", tt.name)
		})
	}
}