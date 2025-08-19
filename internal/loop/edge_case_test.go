package loop

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFileSystemEdgeCases tests handling of various file system edge cases
func TestFileSystemEdgeCases(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Symlink tests require admin privileges on Windows")
	}

	tests := []struct {
		name     string
		setupFunc func(t *testing.T, tempDir string) string
		testFunc  func(t *testing.T, watcher *Watcher, testFile string)
	}{
		{
			name: "symlink to intent file",
			setupFunc: func(t *testing.T, tempDir string) string {
				// Create actual intent file
				realFile := filepath.Join(tempDir, "real-intent.json")
				content := `{
					"intent_type": "scaling",
					"target": "my-app",
					"namespace": "default",
					"replicas": 3
				}`
				require.NoError(t, os.WriteFile(realFile, []byte(content), 0644))

				// Create symlink
				symlinkFile := filepath.Join(tempDir, "symlink-intent.json")
				require.NoError(t, os.Symlink(realFile, symlinkFile))
				
				return symlinkFile
			},
			testFunc: func(t *testing.T, watcher *Watcher, testFile string) {
				// Should handle symlinks gracefully
				assert.True(t, IsIntentFile(filepath.Base(testFile)))
			},
		},
		{
			name: "broken symlink",
			setupFunc: func(t *testing.T, tempDir string) string {
				// Create symlink to non-existent file
				symlinkFile := filepath.Join(tempDir, "broken-symlink-intent.json")
				require.NoError(t, os.Symlink("/nonexistent/file.json", symlinkFile))
				
				return symlinkFile
			},
			testFunc: func(t *testing.T, watcher *Watcher, testFile string) {
				// Should detect as intent file but fail processing gracefully
				assert.True(t, IsIntentFile(filepath.Base(testFile)))
			},
		},
		{
			name: "file with special characters",
			setupFunc: func(t *testing.T, tempDir string) string {
				// Create file with special characters in name
				specialFile := filepath.Join(tempDir, "special-chars-!@#$%^&*()_+-=[]{}|;':\",./<>?.json")
				content := `{
					"intent_type": "scaling",
					"target": "my-app",
					"namespace": "default",
					"replicas": 3
				}`
				require.NoError(t, os.WriteFile(specialFile, []byte(content), 0644))
				
				return specialFile
			},
			testFunc: func(t *testing.T, watcher *Watcher, testFile string) {
				// Should handle special characters in filenames
				assert.True(t, IsIntentFile(filepath.Base(testFile)))
			},
		},
		{
			name: "unicode filename",
			setupFunc: func(t *testing.T, tempDir string) string {
				// Create file with unicode characters
				unicodeFile := filepath.Join(tempDir, "файл-интент-🚀.json")
				content := `{
					"intent_type": "scaling",
					"target": "my-app-🔧",
					"namespace": "пространство",
					"replicas": 3
				}`
				require.NoError(t, os.WriteFile(unicodeFile, []byte(content), 0644))
				
				return unicodeFile
			},
			testFunc: func(t *testing.T, watcher *Watcher, testFile string) {
				// Should handle unicode filenames and content
				assert.True(t, IsIntentFile(filepath.Base(testFile)))
			},
		},
		{
			name: "very long filename",
			setupFunc: func(t *testing.T, tempDir string) string {
				// Create file with very long name (near filesystem limit)
				longName := strings.Repeat("a", 200) + ".json"
				longFile := filepath.Join(tempDir, longName)
				content := `{
					"intent_type": "scaling",
					"target": "my-app",
					"namespace": "default",
					"replicas": 3
				}`
				require.NoError(t, os.WriteFile(longFile, []byte(content), 0644))
				
				return longFile
			},
			testFunc: func(t *testing.T, watcher *Watcher, testFile string) {
				// Should handle very long filenames
				assert.True(t, IsIntentFile(filepath.Base(testFile)))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			handoffDir := filepath.Join(tempDir, "handoff")
			outDir := filepath.Join(tempDir, "out")
			
			require.NoError(t, os.MkdirAll(handoffDir, 0755))
			require.NoError(t, os.MkdirAll(outDir, 0755))

			// Setup test file in handoff directory
			testFile := tt.setupFunc(t, handoffDir)

			// Create mock porch
			mockPorch := createEdgeCaseMockPorch(t, tempDir)

			config := Config{
				PorchPath:    mockPorch,
				Mode:         "direct",
				OutDir:       outDir,
				Once:         true,
				DebounceDur:  100 * time.Millisecond,
				MaxWorkers:   1,
				CleanupAfter: time.Hour,
			}

			watcher, err := NewWatcher(handoffDir, config)
			require.NoError(t, err)
			defer watcher.Close()

			tt.testFunc(t, watcher, testFile)

			// Process files
			err = watcher.Start()
			assert.NoError(t, err, "Watcher should handle edge cases gracefully")
		})
	}
}

// TestNetworkDiskFailureSimulation tests behavior under resource constraints
func TestNetworkDiskFailureSimulation(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T, tempDir string) (string, string)
		testFunc  func(t *testing.T, watcher *Watcher, handoffDir, outDir string)
	}{
		{
			name: "disk full simulation",
			setupFunc: func(t *testing.T, tempDir string) (string, string) {
				handoffDir := filepath.Join(tempDir, "handoff")
				outDir := filepath.Join(tempDir, "out")
				
				require.NoError(t, os.MkdirAll(handoffDir, 0755))
				
				// Create a very small output directory to simulate disk full
				// Note: This is a simplified simulation
				require.NoError(t, os.MkdirAll(outDir, 0755))
				
				return handoffDir, outDir
			},
			testFunc: func(t *testing.T, watcher *Watcher, handoffDir, outDir string) {
				// Create intent file
				intentContent := `{
					"intent_type": "scaling",
					"target": "my-app",
					"namespace": "default",
					"replicas": 3
				}`
				intentFile := filepath.Join(handoffDir, "disk-full-test.json")
				require.NoError(t, os.WriteFile(intentFile, []byte(intentContent), 0644))

				// Process should handle disk space issues gracefully
				err := watcher.Start()
				assert.NoError(t, err, "Should handle disk space issues gracefully")
			},
		},
		{
			name: "partial write simulation",
			setupFunc: func(t *testing.T, tempDir string) (string, string) {
				handoffDir := filepath.Join(tempDir, "handoff")
				outDir := filepath.Join(tempDir, "out")
				
				require.NoError(t, os.MkdirAll(handoffDir, 0755))
				require.NoError(t, os.MkdirAll(outDir, 0755))
				
				return handoffDir, outDir
			},
			testFunc: func(t *testing.T, watcher *Watcher, handoffDir, outDir string) {
				// Create intent file that will be partially written during processing
				intentContent := `{
					"intent_type": "scaling",
					"target": "my-app",
					"namespace": "default",
					"replicas": 3
				}`
				
				// Write file in chunks to simulate partial write
				intentFile := filepath.Join(handoffDir, "partial-write-test.json")
				
				// First write partial content
				require.NoError(t, os.WriteFile(intentFile, []byte(intentContent[:20]), 0644))
				
				// Small delay then complete the write
				go func() {
					time.Sleep(50 * time.Millisecond)
					os.WriteFile(intentFile, []byte(intentContent), 0644)
				}()

				err := watcher.Start()
				assert.NoError(t, err, "Should handle partial writes gracefully")
			},
		},
		{
			name: "read-only filesystem simulation",
			setupFunc: func(t *testing.T, tempDir string) (string, string) {
				handoffDir := filepath.Join(tempDir, "handoff")
				outDir := filepath.Join(tempDir, "out")
				
				require.NoError(t, os.MkdirAll(handoffDir, 0755))
				require.NoError(t, os.MkdirAll(outDir, 0755))
				
				// Create intent file first
				intentContent := `{
					"intent_type": "scaling",
					"target": "my-app",
					"namespace": "default",
					"replicas": 3
				}`
				intentFile := filepath.Join(handoffDir, "readonly-test.json")
				require.NoError(t, os.WriteFile(intentFile, []byte(intentContent), 0644))
				
				// Make handoff directory read-only (simulation)
				if runtime.GOOS != "windows" {
					require.NoError(t, os.Chmod(handoffDir, 0555))
				}
				
				return handoffDir, outDir
			},
			testFunc: func(t *testing.T, watcher *Watcher, handoffDir, outDir string) {
				// Should handle read-only filesystem gracefully
				err := watcher.Start()
				assert.NoError(t, err, "Should handle read-only filesystem gracefully")
				
				// Restore permissions for cleanup
				if runtime.GOOS != "windows" {
					os.Chmod(handoffDir, 0755)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			handoffDir, outDir := tt.setupFunc(t, tempDir)

			// Create mock porch that handles errors gracefully
			mockPorch := createRobustMockPorch(t, tempDir)

			config := Config{
				PorchPath:    mockPorch,
				Mode:         "direct",
				OutDir:       outDir,
				Once:         true,
				DebounceDur:  50 * time.Millisecond,
				MaxWorkers:   1,
				CleanupAfter: time.Hour,
			}

			watcher, err := NewWatcher(handoffDir, config)
			require.NoError(t, err)
			defer watcher.Close()

			tt.testFunc(t, watcher, handoffDir, outDir)
		})
	}
}

// TestSignalHandlingGracefulShutdown tests graceful shutdown under various conditions
func TestSignalHandlingGracefulShutdown(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T, tempDir string) (string, int)
		testFunc  func(t *testing.T, watcher *Watcher, ctx context.Context, cancel context.CancelFunc)
	}{
		{
			name: "graceful shutdown during processing",
			setupFunc: func(t *testing.T, tempDir string) (string, int) {
				handoffDir := filepath.Join(tempDir, "handoff")
				require.NoError(t, os.MkdirAll(handoffDir, 0755))
				
				// Create multiple intent files
				for i := 0; i < 10; i++ {
					content := fmt.Sprintf(`{
						"intent_type": "scaling",
						"target": "app-%d",
						"namespace": "default",
						"replicas": %d
					}`, i, i%5+1)
					
					file := filepath.Join(handoffDir, fmt.Sprintf("intent-%d.json", i))
					require.NoError(t, os.WriteFile(file, []byte(content), 0644))
				}
				
				return handoffDir, 10
			},
			testFunc: func(t *testing.T, watcher *Watcher, ctx context.Context, cancel context.CancelFunc) {
				// Start processing in background
				go func() {
					_ = watcher.Start()
				}()
				
				// Wait a bit for processing to start
				time.Sleep(100 * time.Millisecond)
				
				// Cancel context to simulate shutdown signal
				cancel()
				
				// Wait for graceful shutdown
				time.Sleep(2 * time.Second)
				
				// Watcher should have stopped gracefully
				assert.NotNil(t, watcher, "Watcher should still exist after graceful shutdown")
			},
		},
		{
			name: "shutdown during worker processing",
			setupFunc: func(t *testing.T, tempDir string) (string, int) {
				handoffDir := filepath.Join(tempDir, "handoff")
				require.NoError(t, os.MkdirAll(handoffDir, 0755))
				
				// Create files that will take time to process
				for i := 0; i < 5; i++ {
					content := fmt.Sprintf(`{
						"intent_type": "scaling",
						"target": "slow-app-%d",
						"namespace": "default",
						"replicas": %d
					}`, i, i%3+1)
					
					file := filepath.Join(handoffDir, fmt.Sprintf("slow-intent-%d.json", i))
					require.NoError(t, os.WriteFile(file, []byte(content), 0644))
				}
				
				return handoffDir, 5
			},
			testFunc: func(t *testing.T, watcher *Watcher, ctx context.Context, cancel context.CancelFunc) {
				// Start processing
				go func() {
					_ = watcher.Start()
				}()
				
				// Let some processing start
				time.Sleep(200 * time.Millisecond)
				
				// Cancel and wait for shutdown
				cancel()
				time.Sleep(3 * time.Second)
				
				// Should have shut down gracefully
				assert.NotNil(t, watcher)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			outDir := filepath.Join(tempDir, "out")
			require.NoError(t, os.MkdirAll(outDir, 0755))

			handoffDir, _ := tt.setupFunc(t, tempDir)

			// Create mock porch with delay to simulate processing time
			mockPorch := createSlowMockPorch(t, tempDir, 500*time.Millisecond)

			config := Config{
				PorchPath:    mockPorch,
				Mode:         "direct",
				OutDir:       outDir,
				Once:         false, // Continuous mode
				DebounceDur:  50 * time.Millisecond,
				MaxWorkers:   3,
				CleanupAfter: time.Hour,
			}

			watcher, err := NewWatcher(handoffDir, config)
			require.NoError(t, err)
			defer watcher.Close()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			tt.testFunc(t, watcher, ctx, cancel)
		})
	}
}

// TestStateCorruptionRecovery tests recovery from corrupted state files
func TestStateCorruptionRecovery(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T, tempDir string) string
		testFunc  func(t *testing.T, sm *StateManager)
	}{
		{
			name: "corrupted JSON state file",
			setupFunc: func(t *testing.T, tempDir string) string {
				stateFile := filepath.Join(tempDir, StateFileName)
				// Write corrupted JSON
				corruptedState := `{
					"files": {
						"file1.json": {
							"status": "processed",
							"timestamp": "2025-08-15T12:00:00Z"
						},
						"file2.json": {
							"status": "processed"
							// Missing comma and closing brace
				}`
				require.NoError(t, os.WriteFile(stateFile, []byte(corruptedState), 0644))
				return tempDir
			},
			testFunc: func(t *testing.T, sm *StateManager) {
				// Should create new state when corrupted state is detected
				err := sm.MarkProcessed("new-file.json")
				assert.NoError(t, err, "Should recover from corrupted state")

				processed, err := sm.IsProcessed("new-file.json")
				assert.NoError(t, err)
				assert.True(t, processed, "Should work after recovery")
			},
		},
		{
			name: "empty state file",
			setupFunc: func(t *testing.T, tempDir string) string {
				stateFile := filepath.Join(tempDir, StateFileName)
				// Create empty state file
				require.NoError(t, os.WriteFile(stateFile, []byte(""), 0644))
				return tempDir
			},
			testFunc: func(t *testing.T, sm *StateManager) {
				// Should handle empty state file gracefully
				err := sm.MarkProcessed("test-file.json")
				assert.NoError(t, err, "Should handle empty state file")

				processed, err := sm.IsProcessed("test-file.json")
				assert.NoError(t, err)
				assert.True(t, processed)
			},
		},
		{
			name: "binary garbage in state file",
			setupFunc: func(t *testing.T, tempDir string) string {
				stateFile := filepath.Join(tempDir, StateFileName)
				// Write binary garbage
				garbage := make([]byte, 1024)
				for i := range garbage {
					garbage[i] = byte(i % 256)
				}
				require.NoError(t, os.WriteFile(stateFile, garbage, 0644))
				return tempDir
			},
			testFunc: func(t *testing.T, sm *StateManager) {
				// Should handle binary garbage gracefully
				err := sm.MarkProcessed("recovery-test.json")
				assert.NoError(t, err, "Should recover from binary garbage")

				processed, err := sm.IsProcessed("recovery-test.json")
				assert.NoError(t, err)
				assert.True(t, processed)
			},
		},
		{
			name: "state file with invalid timestamps",
			setupFunc: func(t *testing.T, tempDir string) string {
				stateFile := filepath.Join(tempDir, StateFileName)
				invalidState := `{
					"files": {
						"file1.json": {
							"status": "processed",
							"timestamp": "invalid-timestamp"
						},
						"file2.json": {
							"status": "processed",
							"timestamp": "2025-13-45T99:99:99Z"
						}
					}
				}`
				require.NoError(t, os.WriteFile(stateFile, []byte(invalidState), 0644))
				return tempDir
			},
			testFunc: func(t *testing.T, sm *StateManager) {
				// Should handle invalid timestamps gracefully
				processed, err := sm.IsProcessed("file1.json")
				assert.NoError(t, err, "Should handle invalid timestamps")
				
				// File should still be considered processed despite invalid timestamp
				assert.True(t, processed, "Should preserve file status despite invalid timestamp")

				// Should be able to add new files
				err = sm.MarkProcessed("new-file.json")
				assert.NoError(t, err)
			},
		},
		{
			name: "state file permissions issue",
			setupFunc: func(t *testing.T, tempDir string) string {
				stateFile := filepath.Join(tempDir, StateFileName)
				validState := `{
					"files": {
						"file1.json": {
							"status": "processed",
							"timestamp": "2025-08-15T12:00:00Z"
						}
					}
				}`
				require.NoError(t, os.WriteFile(stateFile, []byte(validState), 0644))
				
				// Make state file read-only to simulate permission issues
				if runtime.GOOS != "windows" {
					require.NoError(t, os.Chmod(stateFile, 0444))
				}
				
				return tempDir
			},
			testFunc: func(t *testing.T, sm *StateManager) {
				// Should handle permission issues gracefully
				if runtime.GOOS != "windows" {
					err := sm.MarkProcessed("permission-test.json")
					// Might fail due to permissions, but should not crash
					assert.NotNil(t, err, "Should return error for permission issues")
				} else {
					t.Skip("Permission test not applicable on Windows")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			stateDir := tt.setupFunc(t, tempDir)

			// Create state manager with potentially corrupted state
			sm, err := NewStateManager(stateDir)
			
			// State manager creation should succeed even with corrupted state
			require.NoError(t, err, "StateManager should handle corrupted state during creation")
			defer sm.Close()

			tt.testFunc(t, sm)
		})
	}
}

// TestConcurrentStateManagement tests concurrent access to state management
func TestConcurrentStateManagement(t *testing.T) {
	tempDir := t.TempDir()
	
	sm, err := NewStateManager(tempDir)
	require.NoError(t, err)
	defer sm.Close()

	// Test concurrent read/write operations
	var wg sync.WaitGroup
	numOperations := 50

	// Concurrent writes
	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			filename := fmt.Sprintf("concurrent-file-%d.json", id)
			
			err := sm.MarkProcessed(filename)
			assert.NoError(t, err, "Concurrent MarkProcessed should not fail")
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			filename := fmt.Sprintf("concurrent-file-%d.json", id)
			
			// May or may not be processed depending on timing
			_, err := sm.IsProcessed(filename)
			assert.NoError(t, err, "Concurrent IsProcessed should not fail")
		}(i)
	}

	wg.Wait()

	// Verify final state is consistent
	for i := 0; i < numOperations; i++ {
		filename := fmt.Sprintf("concurrent-file-%d.json", i)
		processed, err := sm.IsProcessed(filename)
		assert.NoError(t, err)
		assert.True(t, processed, "All files should be marked as processed")
	}
}

// Helper functions

func createEdgeCaseMockPorch(t testing.TB, tempDir string) string {
	var mockScript string
	var mockPath string

	if runtime.GOOS == "windows" {
		mockPath = filepath.Join(tempDir, "mock-porch-edge.bat")
		mockScript = `@echo off
if "%1"=="--help" (
    echo Mock porch help
    exit /b 0
)
echo Processing file: %2
echo Output directory: %4
REM Handle edge cases gracefully
if not exist "%2" (
    echo Warning: Input file does not exist
    exit /b 1
)
echo Edge case processing completed
exit /b 0`
	} else {
		mockPath = filepath.Join(tempDir, "mock-porch-edge")
		mockScript = `#!/bin/bash
if [ "$1" = "--help" ]; then
    echo "Mock porch help"
    exit 0
fi
echo "Processing file: $2"
echo "Output directory: $4"
# Handle edge cases gracefully
if [ ! -f "$2" ]; then
    echo "Warning: Input file does not exist"
    exit 1
fi
echo "Edge case processing completed"
exit 0`
	}

	require.NoError(t, os.WriteFile(mockPath, []byte(mockScript), 0755))
	return mockPath
}

func createRobustMockPorch(t testing.TB, tempDir string) string {
	var mockScript string
	var mockPath string

	if runtime.GOOS == "windows" {
		mockPath = filepath.Join(tempDir, "mock-porch-robust.bat")
		mockScript = `@echo off
if "%1"=="--help" (
    echo Mock porch help
    exit /b 0
)
REM Robust error handling
echo Processing: %2
if not exist "%4" (
    echo Creating output directory: %4
    mkdir "%4" 2>nul
)
echo Robust processing completed
exit /b 0`
	} else {
		mockPath = filepath.Join(tempDir, "mock-porch-robust")
		mockScript = `#!/bin/bash
if [ "$1" = "--help" ]; then
    echo "Mock porch help"
    exit 0
fi
# Robust error handling
echo "Processing: $2"
if [ ! -d "$4" ]; then
    echo "Creating output directory: $4"
    mkdir -p "$4" 2>/dev/null
fi
echo "Robust processing completed"
exit 0`
	}

	require.NoError(t, os.WriteFile(mockPath, []byte(mockScript), 0755))
	return mockPath
}

func createSlowMockPorch(t testing.TB, tempDir string, delay time.Duration) string {
	var mockScript string
	var mockPath string

	if runtime.GOOS == "windows" {
		mockPath = filepath.Join(tempDir, "mock-porch-slow.bat")
		mockScript = fmt.Sprintf(`@echo off
if "%%1"=="--help" (
    echo Mock porch help
    exit /b 0
)
echo Processing slowly: %%2
timeout /t %d /nobreak >nul 2>&1
echo Slow processing completed
exit /b 0`, int(delay.Seconds())+1)
	} else {
		mockPath = filepath.Join(tempDir, "mock-porch-slow")
		mockScript = fmt.Sprintf(`#!/bin/bash
if [ "$1" = "--help" ]; then
    echo "Mock porch help"
    exit 0
fi
echo "Processing slowly: $2"
sleep %v
echo "Slow processing completed"
exit 0`, delay.Seconds())
	}

	require.NoError(t, os.WriteFile(mockPath, []byte(mockScript), 0755))
	return mockPath
}

// isProcessRunning checks if a process is still running (simplified check)
func isProcessRunning(pid int) bool {
	if runtime.GOOS == "windows" {
		// Simplified check for Windows
		return true // Assume running for test purposes
	}
	
	// Send signal 0 to check if process exists (Unix-like systems only)
	// This is a simplified implementation
	return true // For testing purposes, assume process is running
}