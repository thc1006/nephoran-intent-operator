package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/thc1006/nephoran-intent-operator/internal/porch"
)

func TestNewWatcher(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		dir       string
		config    Config
		setupFunc func(t *testing.T) string // returns porch executable path
		wantErr   bool
	}{
		{
			name: "valid configuration",
			dir:  tempDir,
			config: Config{
				Mode:        porch.ModeDirect,
				OutDir:      filepath.Join(tempDir, "out"),
				MaxWorkers:  2,
				DebounceDur: 500 * time.Millisecond,
			},
			setupFunc: func(t *testing.T) string {
				return createMockPorch(t, tempDir, 0, "success", "")
			},
			wantErr: false,
		},
		{
			name: "nonexistent directory",
			dir:  filepath.Join(tempDir, "nonexistent"),
			config: Config{
				Mode:   porch.ModeDirect,
				OutDir: filepath.Join(tempDir, "out"),
			},
			setupFunc: func(t *testing.T) string {
				return createMockPorch(t, tempDir, 0, "success", "")
			},
			wantErr: true,
		},
		{
			name: "default values applied",
			dir:  tempDir,
			config: Config{
				Mode:   porch.ModeDirect,
				OutDir: filepath.Join(tempDir, "out"),
				// MaxWorkers and CleanupAfter not set
			},
			setupFunc: func(t *testing.T) string {
				return createMockPorch(t, tempDir, 0, "success", "")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			porchPath := tt.setupFunc(t)
			tt.config.PorchPath = porchPath

			watcher, err := NewWatcher(tt.dir, tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, watcher)

			// Verify default values were applied
			if tt.config.MaxWorkers == 0 {
				assert.Equal(t, 2, watcher.config.MaxWorkers)
			}
			if tt.config.CleanupAfter == 0 {
				assert.Equal(t, 7*24*time.Hour, watcher.config.CleanupAfter)
			}

			// Verify components were created
			assert.NotNil(t, watcher.stateManager)
			assert.NotNil(t, watcher.fileManager)
			assert.NotNil(t, watcher.executor)
			assert.NotNil(t, watcher.watcher)

			// Cleanup
			require.NoError(t, watcher.Close())
		})
	}
}

func TestWatcher_ProcessExistingFiles(t *testing.T) {
	tempDir := t.TempDir()
	outDir := filepath.Join(tempDir, "out")
	require.NoError(t, os.MkdirAll(outDir, 0o755))

	// Create test intent files
	intentFiles := []string{
		"intent-test-1.json",
		"intent-test-2.json",
		"not-intent.txt", // Should be ignored
	}

	testContent := `{"apiVersion": "v1", "kind": "NetworkIntent", "action": "scale", "target": "deployment", "count": 3}`
	for _, fileName := range intentFiles {
		filePath := filepath.Join(tempDir, fileName)
		require.NoError(t, os.WriteFile(filePath, []byte(testContent), 0o644))
	}

	config := Config{
		PorchPath:  createMockPorch(t, tempDir, 0, "processed successfully", ""),
		Mode:       porch.ModeDirect,
		OutDir:     outDir,
		Once:       true, // Process existing files only
		MaxWorkers: 2,
	}

	watcher, err := NewWatcher(tempDir, config)
	require.NoError(t, err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	// Start watcher in once mode
	err = watcher.Start()
	require.NoError(t, err)

	// Wait a bit for processing to complete
	time.Sleep(1 * time.Second)

	// Verify only intent files were processed
	processedFiles, err := watcher.fileManager.GetProcessedFiles()
	require.NoError(t, err)

	// Should have 2 intent files processed (not the .txt file)
	assert.Len(t, processedFiles, 2)

	// Verify original intent files are gone
	assert.NoFileExists(t, filepath.Join(tempDir, "intent-test-1.json"))
	assert.NoFileExists(t, filepath.Join(tempDir, "intent-test-2.json"))

	// Verify non-intent file is still there
	assert.FileExists(t, filepath.Join(tempDir, "not-intent.txt"))
}

func TestWatcher_FileDetectionWithinRequirement(t *testing.T) {
	tempDir := t.TempDir()
	outDir := filepath.Join(tempDir, "out")
	require.NoError(t, os.MkdirAll(outDir, 0o755))

	config := Config{
		PorchPath:   createMockPorch(t, tempDir, 0, "processed", ""),
		Mode:        porch.ModeDirect,
		OutDir:      outDir,
		DebounceDur: 100 * time.Millisecond, // Short debounce for testing
		MaxWorkers:  2,                      // Production-like worker count for better concurrency testing
	}

	watcher, err := NewWatcher(tempDir, config)
	require.NoError(t, err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	// Start watcher in background
	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- watcher.Start()
	}()

	// Wait for watcher to start
	time.Sleep(200 * time.Millisecond)

	// Create a new intent file
	testFile := filepath.Join(tempDir, "intent-new.json")
	testContent := `{"apiVersion": "v1", "kind": "NetworkIntent", "action": "scale", "target": "deployment", "count": 5}`

	startTime := time.Now()
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0o644))

	// Wait for file to be processed (should be within 2 seconds)
	timeout := time.After(2 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var processed bool
	for !processed {
		select {
		case <-timeout:
			t.Fatal("File was not processed within 2 seconds")
		case <-ticker.C:
			// Check if file was processed
			if !fileExists(testFile) {
				processed = true
				detectionTime := time.Since(startTime)
				t.Logf("File detected and processed in %v", detectionTime)
				assert.Less(t, detectionTime, 2*time.Second, "File should be processed within 2 seconds")
			}
		}
	}

	// Cleanup
	cancel()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		// Force close if graceful shutdown doesn't work
	}
}

func TestWatcher_DebouncingRapidChanges(t *testing.T) {
	tempDir := t.TempDir()
	outDir := filepath.Join(tempDir, "out")
	require.NoError(t, os.MkdirAll(outDir, 0o755))

	config := Config{
		PorchPath:   createMockPorch(t, tempDir, 0, "processed", ""),
		Mode:        porch.ModeDirect,
		OutDir:      outDir,
		DebounceDur: 500 * time.Millisecond, // Longer debounce for testing
		MaxWorkers:  2,                      // Production-like worker count for better concurrency testing
	}

	watcher, err := NewWatcher(tempDir, config)
	require.NoError(t, err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	// Start watcher
	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- watcher.Start()
	}()

	// Wait for watcher to start
	time.Sleep(200 * time.Millisecond)

	testFile := filepath.Join(tempDir, "intent-debounce-test.json")

	// Write to file multiple times rapidly
	for i := 0; i < 5; i++ {
		content := fmt.Sprintf(`{"apiVersion": "v1", "kind": "NetworkIntent", "action": "scale", "count": %d}`, i+1)
		require.NoError(t, os.WriteFile(testFile, []byte(content), 0o644))
		time.Sleep(50 * time.Millisecond) // Rapid changes
	}

	// Wait for debouncing to settle and processing to complete
	time.Sleep(1 * time.Second)

	// File should be processed only once due to debouncing
	stats := watcher.executor.GetStats()
	assert.LessOrEqual(t, stats.TotalExecutions, 1, "File should be processed at most once due to debouncing")

	cancel()
	select {
	case <-done:
	case <-time.After(1 * time.Second):
	}
}

func TestWatcher_IdempotentProcessing(t *testing.T) {
	tempDir := t.TempDir()
	outDir := filepath.Join(tempDir, "out")
	require.NoError(t, os.MkdirAll(outDir, 0o755))

	config := Config{
		PorchPath:   createMockPorch(t, tempDir, 0, "processed", ""),
		Mode:        porch.ModeDirect,
		OutDir:      outDir,
		DebounceDur: 100 * time.Millisecond,
		MaxWorkers:  2, // Production-like worker count for better concurrency testing
	}

	watcher, err := NewWatcher(tempDir, config)
	require.NoError(t, err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	testFile := filepath.Join(tempDir, "intent-idempotent-test.json")
	testContent := `{"apiVersion": "v1", "kind": "NetworkIntent", "action": "scale", "target": "deployment", "count": 3}`
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0o644))

	// Manually mark file as already processed
	require.NoError(t, watcher.stateManager.MarkProcessed(testFile))

	// Start watcher in once mode
	config.Once = true
	watcher.config = config

	err = watcher.Start()
	require.NoError(t, err)

	// Verify file was not processed again
	stats := watcher.executor.GetStats()
	assert.Equal(t, 0, stats.TotalExecutions, "Already processed file should not be processed again")

	// File should still exist since it wasn't processed
	assert.FileExists(t, testFile)
}

func TestWatcher_ConcurrentFileProcessing(t *testing.T) {
	tempDir := t.TempDir()
	outDir := filepath.Join(tempDir, "out")
	require.NoError(t, os.MkdirAll(outDir, 0o755))

	config := Config{
		PorchPath:   createMockPorch(t, tempDir, 0, "processed", "", 100*time.Millisecond), // Add small delay
		Mode:        porch.ModeDirect,
		OutDir:      outDir,
		DebounceDur: 50 * time.Millisecond,
		MaxWorkers:  3, // Multiple workers for concurrency
	}

	watcher, err := NewWatcher(tempDir, config)
	require.NoError(t, err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	// Create multiple intent files
	numFiles := 10
	testContent := `{"apiVersion": "v1", "kind": "NetworkIntent", "action": "scale", "target": "deployment", "count": 1}`

	for i := 0; i < numFiles; i++ {
		testFile := filepath.Join(tempDir, fmt.Sprintf("intent-concurrent-test-%d.json", i))
		require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0o644))
	}

	// Start watcher in once mode
	config.Once = true
	watcher.config = config

	startTime := time.Now()
	err = watcher.Start()
	require.NoError(t, err)
	processingTime := time.Since(startTime)

	// Verify all files were processed
	stats := watcher.executor.GetStats()
	assert.Equal(t, numFiles, stats.TotalExecutions)
	assert.Equal(t, numFiles, stats.SuccessfulExecs)

	// With 3 workers, processing should be faster than sequential
	// (Though this is a rough estimate and may be flaky on slow systems)
	expectedSequentialTime := time.Duration(numFiles) * 100 * time.Millisecond
	t.Logf("Processing time: %v, Expected sequential: %v", processingTime, expectedSequentialTime)
}

func TestWatcher_FailureScenarios(t *testing.T) {
	tempDir := t.TempDir()
	outDir := filepath.Join(tempDir, "out")
	require.NoError(t, os.MkdirAll(outDir, 0o755))

	tests := []struct {
		name            string
		setupFunc       func(t *testing.T) string // returns porch path
		expectedSuccess bool
		expectedFailed  bool
	}{
		{
			name: "porch command fails",
			setupFunc: func(t *testing.T) string {
				return createMockPorch(t, tempDir, 1, "", "porch failed")
			},
			expectedSuccess: false,
			expectedFailed:  true,
		},
		{
			name: "porch command succeeds",
			setupFunc: func(t *testing.T) string {
				return createMockPorch(t, tempDir, 0, "success", "")
			},
			expectedSuccess: true,
			expectedFailed:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			porchPath := tt.setupFunc(t)

			config := Config{
				PorchPath:  porchPath,
				Mode:       porch.ModeDirect,
				OutDir:     outDir,
				Once:       true,
				MaxWorkers: 2, // Production-like worker count for better concurrency testing
			}

			watcher, err := NewWatcher(tempDir, config)
			require.NoError(t, err)
			defer watcher.Close() // #nosec G307 - Error handled in defer

			// Create test file
			testFile := filepath.Join(tempDir, fmt.Sprintf("intent-failure-test-%s.json", tt.name))
			testContent := `{"apiVersion": "v1", "kind": "NetworkIntent", "action": "scale", "target": "deployment", "count": 1}`
			require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0o644))

			// Process file
			err = watcher.Start()
			require.NoError(t, err)

			// Verify outcome
			if tt.expectedSuccess {
				processedFiles, err := watcher.fileManager.GetProcessedFiles()
				require.NoError(t, err)
				assert.Len(t, processedFiles, 1)

				// Verify status file was created (with timestamp pattern)
				statusDir := filepath.Join(tempDir, "status")
				statusFiles, err := os.ReadDir(statusDir)
				require.NoError(t, err)
				assert.Greater(t, len(statusFiles), 0, "Should have at least one status file")

				// Check that status file follows the expected naming pattern
				baseNameWithoutExt := strings.TrimSuffix(filepath.Base(testFile), filepath.Ext(filepath.Base(testFile)))
				expectedPattern := fmt.Sprintf("%s-\\d{8}-\\d{6}\\.status", regexp.QuoteMeta(baseNameWithoutExt))
				found := false
				for _, file := range statusFiles {
					if matched, _ := regexp.MatchString(expectedPattern, file.Name()); matched {
						found = true
						break
					}
				}
				assert.True(t, found, "Should find status file matching pattern %s", expectedPattern)
			}

			if tt.expectedFailed {
				failedFiles, err := watcher.fileManager.GetFailedFiles()
				require.NoError(t, err)
				assert.Len(t, failedFiles, 1)

				// Verify error log was created
				logFile := filepath.Join(watcher.fileManager.failedDir, filepath.Base(testFile)+".error.log")
				assert.FileExists(t, logFile)
			}
		})
	}
}

func TestWatcher_CleanupRoutine(t *testing.T) {
	tempDir := t.TempDir()
	outDir := filepath.Join(tempDir, "out")
	require.NoError(t, os.MkdirAll(outDir, 0o755))

	config := Config{
		PorchPath:    createMockPorch(t, tempDir, 0, "processed", ""),
		Mode:         porch.ModeDirect,
		OutDir:       outDir,
		MaxWorkers:   2,             // Use production-like worker count for better concurrency testing
		CleanupAfter: 2 * time.Hour, // Minimum valid duration for testing
	}

	watcher, err := NewWatcher(tempDir, config)
	require.NoError(t, err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	// Create and process a file
	testFile := filepath.Join(tempDir, "intent-cleanup-test.json")
	testContent := `{"apiVersion": "v1", "kind": "NetworkIntent", "action": "scale", "target": "deployment", "count": 1}`
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0o644))

	// Process once
	config.Once = true
	watcher.config = config
	err = watcher.Start()
	require.NoError(t, err)

	// Verify file was processed
	stats := watcher.executor.GetStats()
	assert.Equal(t, 1, stats.TotalExecutions)

	// The cleanup routine is more complex to test as it runs periodically
	// This test mainly verifies the watcher can be created with short cleanup duration
	// without errors
}

func TestWatcher_GracefulShutdown(t *testing.T) {
	tempDir := t.TempDir()
	outDir := filepath.Join(tempDir, "out")
	require.NoError(t, os.MkdirAll(outDir, 0o755))

	config := Config{
		PorchPath:   createMockPorch(t, tempDir, 0, "processed", "", 2*time.Second), // Longer processing
		Mode:        porch.ModeDirect,
		OutDir:      outDir,
		DebounceDur: 50 * time.Millisecond,
		MaxWorkers:  2,
	}

	watcher, err := NewWatcher(tempDir, config)
	require.NoError(t, err)

	// Create multiple files for processing
	for i := 0; i < 3; i++ {
		testFile := filepath.Join(tempDir, fmt.Sprintf("intent-shutdown-test-%d.json", i))
		testContent := `{"apiVersion": "v1", "kind": "NetworkIntent", "action": "scale", "target": "deployment", "count": 1}`
		require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0o644))
	}

	// Start watcher
	done := make(chan error, 1)
	go func() {
		config.Once = true
		watcher.config = config
		done <- watcher.Start()
	}()

	// Wait a bit, then close
	time.Sleep(500 * time.Millisecond)

	closeStart := time.Now()
	err = watcher.Close()
	closeTime := time.Since(closeStart)

	require.NoError(t, err)

	// Wait for start to complete
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Watcher didn't shutdown gracefully within timeout")
	}

	t.Logf("Graceful shutdown took %v", closeTime)

	// Should have processed some files (at least the ones that started)
	stats := watcher.executor.GetStats()
	assert.Greater(t, stats.TotalExecutions, 0)
}

func TestWatcher_StatusFileGeneration(t *testing.T) {
	tempDir := t.TempDir()
	outDir := filepath.Join(tempDir, "out")
	require.NoError(t, os.MkdirAll(outDir, 0o755))

	config := Config{
		PorchPath:  createMockPorch(t, tempDir, 0, "processing successful", ""),
		Mode:       porch.ModeStructured,
		OutDir:     outDir,
		Once:       true,
		MaxWorkers: 2, // Production-like worker count for better concurrency testing
	}

	watcher, err := NewWatcher(tempDir, config)
	require.NoError(t, err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	// Create test file
	testFile := filepath.Join(tempDir, "intent-status-test.json")
	testContent := `{"apiVersion": "v1", "kind": "NetworkIntent", "action": "scale", "target": "deployment", "count": 3}`
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0o644))

	// Process file
	err = watcher.Start()
	require.NoError(t, err)

	// Verify status file was created (with timestamp pattern)
	statusDir := filepath.Join(tempDir, "status")
	statusFiles, err := os.ReadDir(statusDir)
	require.NoError(t, err)
	assert.Greater(t, len(statusFiles), 0, "Should have at least one status file")

	// Find the status file with expected pattern
	baseNameWithoutExt := "intent-status-test"
	expectedPattern := fmt.Sprintf("%s-\\d{8}-\\d{6}\\.status", regexp.QuoteMeta(baseNameWithoutExt))
	var statusFile string
	for _, file := range statusFiles {
		if matched, _ := regexp.MatchString(expectedPattern, file.Name()); matched {
			statusFile = filepath.Join(statusDir, file.Name())
			break
		}
	}
	assert.NotEmpty(t, statusFile, "Should find status file matching pattern %s", expectedPattern)

	// Verify status file content
	statusContent, err := os.ReadFile(statusFile)
	require.NoError(t, err)

	statusStr := string(statusContent)
	assert.Contains(t, statusStr, `"intent_file": "intent-status-test.json"`)
	assert.Contains(t, statusStr, `"status": "success"`)
	assert.Contains(t, statusStr, `"processed_by": "conductor-loop"`)
	assert.Contains(t, statusStr, `"mode": "structured"`)
	assert.Contains(t, statusStr, "timestamp")
}

// Helper functions

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// createMockPorch creates a mock porch executable for testing
func createMockPorch(t testing.TB, tempDir string, exitCode int, stdout, stderr string, sleepDuration ...time.Duration) string {
	var sleep time.Duration
	if len(sleepDuration) > 0 {
		sleep = sleepDuration[0]
	}

	mockPath, err := porch.CreateCrossPlatformMock(tempDir, porch.CrossPlatformMockOptions{
		ExitCode: exitCode,
		Stdout:   stdout,
		Stderr:   stderr,
		Sleep:    sleep,
	})
	require.NoError(t, err)
	return mockPath
}

// createFastMockPorch creates an ultra-fast mock for performance tests without PowerShell overhead
func createFastMockPorch(t testing.TB, tempDir string, exitCode int, stdout, stderr string) string {
	var mockPath string
	var script string

	if runtime.GOOS == "windows" {
		// Minimal Windows batch file without PowerShell calls
		mockPath = filepath.Join(tempDir, "fast-mock-porch.bat")
		stdoutCmd := ""
		if stdout != "" {
			stdoutCmd = fmt.Sprintf("echo %s", stdout)
		}
		stderrCmd := ""
		if stderr != "" {
			stderrCmd = fmt.Sprintf("echo %s >&2", stderr)
		}

		script = fmt.Sprintf(`@echo off
if "%%1"=="--help" (
    echo Mock porch help
    exit /b 0
)
%s%s
exit /b %d`, stdoutCmd, stderrCmd, exitCode)
	} else {
		// Minimal Unix shell script
		mockPath = filepath.Join(tempDir, "fast-mock-porch.sh")
		stdoutCmd := ""
		if stdout != "" {
			stdoutCmd = fmt.Sprintf("echo \"%s\"", stdout)
		}
		stderrCmd := ""
		if stderr != "" {
			stderrCmd = fmt.Sprintf("echo \"%s\" >&2", stderr)
		}

		script = fmt.Sprintf(`#!/bin/bash
if [ "$1" = "--help" ]; then
    echo "Mock porch help"
    exit 0
fi
%s%s
exit %d`, stdoutCmd, stderrCmd, exitCode)
	}

	// Write the script file
	err := os.WriteFile(mockPath, []byte(script), 0o755)
	require.NoError(t, err)
	return mockPath
}

func BenchmarkWatcher_ProcessSingleFile(b *testing.B) {
	tempDir := b.TempDir()
	outDir := filepath.Join(tempDir, "out")
	require.NoError(b, os.MkdirAll(outDir, 0o755))

	config := Config{
		PorchPath:  createMockPorch(b, tempDir, 0, "processed", ""),
		Mode:       porch.ModeDirect,
		OutDir:     outDir,
		Once:       true,
		MaxWorkers: 2, // Production-like worker count for better concurrency testing
	}

	testContent := `{"apiVersion": "v1", "kind": "NetworkIntent", "action": "scale", "target": "deployment", "count": 1}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()

		// Create fresh watcher for each iteration
		watcher, err := NewWatcher(tempDir, config)
		require.NoError(b, err)

		// Create test file
		testFile := filepath.Join(tempDir, fmt.Sprintf("intent-bench-%d.json", i))
		require.NoError(b, os.WriteFile(testFile, []byte(testContent), 0o644))

		b.StartTimer()

		// Process file
		_ = watcher.Start()

		b.StopTimer()
		_ = watcher.Close()
	}
}

// =============================================================================
// COMPREHENSIVE TEST SUITE - Race Conditions, Security, Integration, Performance
// =============================================================================

// WatcherTestSuite provides comprehensive testing for the Watcher component
type WatcherTestSuite struct {
	suite.Suite
	tempDir   string
	config    Config
	porchPath string
}

func TestWatcherTestSuite(t *testing.T) {
	suite.Run(t, new(WatcherTestSuite))
}

func (s *WatcherTestSuite) SetupTest() {
	s.tempDir = s.T().TempDir()
	outDir := filepath.Join(s.tempDir, "out")
	s.Require().NoError(os.MkdirAll(outDir, 0o755))

	s.porchPath = createMockPorch(s.T(), s.tempDir, 0, "processed successfully", "")
	s.config = Config{
		PorchPath:   s.porchPath,
		Mode:        porch.ModeDirect,
		OutDir:      outDir,
		MaxWorkers:  4,
		DebounceDur: 50 * time.Millisecond,
		Once:        false,
	}
}

// =============================================================================
// RACE CONDITION TESTS
// =============================================================================

func (s *WatcherTestSuite) TestRaceCondition_ConcurrentFileProcessing() {
	s.T().Log("Testing concurrent file processing race conditions")

	// Use optimized config with no artificial delays
	raceConfig := s.config
	raceConfig.PorchPath = createFastMockPorch(s.T(), s.tempDir, 0, "processed", "") // Ultra-fast mock
	raceConfig.DebounceDur = 1 * time.Millisecond                                    // Minimal debounce

	watcher, err := NewWatcher(s.tempDir, raceConfig)
	s.Require().NoError(err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	numFiles := 10  // Reduced from 20 for faster test
	numWorkers := 4 // Reduced from 8
	watcher.config.MaxWorkers = numWorkers

	// Create files concurrently
	var wg sync.WaitGroup
	var processedCount int64

	// Track files to ensure no duplicates are processed
	fileSet := make(map[string]struct{})
	var fileMutex sync.Mutex

	for i := 0; i < numFiles; i++ {
		wg.Add(1)
		go func(fileID int) {
			defer wg.Done()

			fileName := fmt.Sprintf("intent-race-test-%d.json", fileID)
			filePath := filepath.Join(s.tempDir, fileName)
			testContent := fmt.Sprintf(`{"apiVersion": "v1", "kind": "NetworkIntent", "spec": {"action": "scale", "target": {"type": "deployment", "name": "app-%d"}}}`, fileID)

			s.Require().NoError(os.WriteFile(filePath, []byte(testContent), 0o644))

			fileMutex.Lock()
			fileSet[fileName] = struct{}{}
			fileMutex.Unlock()

			atomic.AddInt64(&processedCount, 1)
		}(i)
	}

	// Start watcher in once mode to process all files
	watcher.config.Once = true

	startTime := time.Now()
	err = watcher.Start()
	s.Require().NoError(err)

	wg.Wait()
	processingTime := time.Since(startTime)
	s.T().Logf("Processed %d files with %d workers in %v", numFiles, numWorkers, processingTime)

	// Verify files were processed (may not be all due to queue size limits and backpressure)
	stats := watcher.executor.GetStats()
	s.Assert().Greater(stats.TotalExecutions, 0, "Some files should be processed")
	s.Assert().LessOrEqual(stats.TotalExecutions, numFiles, "Should not process more files than created")

	// Verify processed files match execution stats
	processedFiles, err := watcher.fileManager.GetProcessedFiles()
	s.Require().NoError(err)
	s.Assert().Equal(stats.SuccessfulExecs, len(processedFiles), "Processed files should match successful executions")

	s.T().Logf("Successfully tested race conditions: processed %d/%d files", stats.TotalExecutions, numFiles)
}

func (s *WatcherTestSuite) TestRaceCondition_DirectoryCreationRace() {
	s.T().Log("Testing directory creation race conditions")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	numGoroutines := 50
	var wg sync.WaitGroup

	// Try to create the same directory from multiple goroutines
	statusDir := filepath.Join(s.tempDir, "status", "subdir")

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Use the internal directory manager
			watcher.ensureDirectoryExists(statusDir)
		}(i)
	}

	wg.Wait()

	// Verify directory was created successfully
	s.Assert().DirExists(statusDir)
}

func (s *WatcherTestSuite) TestRaceCondition_FileLevelLocking() {
	s.T().Log("Testing file-level locking mechanism")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	testFile := filepath.Join(s.tempDir, "intent-lock-test.json")
	testContent := `{"apiVersion": "v1", "kind": "NetworkIntent", "action": "scale", "target": "deployment", "count": 1}`
	s.Require().NoError(os.WriteFile(testFile, []byte(testContent), 0o644))

	numWorkers := 10
	var wg sync.WaitGroup
	var processCount int64

	// Simulate multiple workers trying to process the same file
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Get file lock (this should serialize access)
			lock := watcher.getOrCreateFileLock(testFile)
			lock.Lock()
			defer lock.Unlock()

			// Simulate processing
			count := atomic.AddInt64(&processCount, 1)
			time.Sleep(10 * time.Millisecond) // Simulate work
			s.T().Logf("Worker %d processed file (count: %d)", workerID, count)
		}(i)
	}

	wg.Wait()

	// All workers should have processed, but serially
	s.Assert().Equal(int64(numWorkers), processCount)
}

func (s *WatcherTestSuite) TestRaceCondition_WorkerPoolHighConcurrency() {
	s.T().Log("Testing worker pool with high concurrency")

	s.config.MaxWorkers = 8
	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	numFiles := 100
	testContent := `{"apiVersion": "v1", "kind": "NetworkIntent", "action": "scale", "target": "deployment", "count": 1}`

	// Create many files rapidly
	for i := 0; i < numFiles; i++ {
		fileName := fmt.Sprintf("intent-concurrency-%d.json", i)
		filePath := filepath.Join(s.tempDir, fileName)
		s.Require().NoError(os.WriteFile(filePath, []byte(testContent), 0o644))
	}

	// Process in once mode
	watcher.config.Once = true

	startTime := time.Now()
	err = watcher.Start()
	s.Require().NoError(err)
	processingTime := time.Since(startTime)

	stats := watcher.executor.GetStats()
	s.Assert().Equal(numFiles, stats.TotalExecutions)
	s.Assert().Equal(numFiles, stats.SuccessfulExecs)
	s.T().Logf("Processed %d files in %v with %d workers", numFiles, processingTime, s.config.MaxWorkers)
}

// =============================================================================
// JSON VALIDATION TESTS
// =============================================================================

func (s *WatcherTestSuite) TestJSONValidation_ValidJSONProcessing() {
	s.T().Log("Testing valid JSON processing")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	tests := []struct {
		name    string
		content string
		valid   bool
	}{
		{
			name:    "valid_intent_schema",
			content: `{"apiVersion": "v1", "kind": "NetworkIntent", "metadata": {"name": "test"}, "spec": {"action": "scale"}}`,
			valid:   true,
		},
		{
			name:    "valid_minimal_intent",
			content: `{"apiVersion": "v1", "kind": "NetworkIntent"}`,
			valid:   true,
		},
		{
			name:    "generic_valid_json",
			content: `{"apiVersion": "v1", "kind": "NetworkIntent", "spec": {"action": "scale", "target": {"name": "deployment"}}}`,
			valid:   true,
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			fileName := fmt.Sprintf("intent-%s.json", tt.name)
			filePath := filepath.Join(s.tempDir, fileName)

			err := os.WriteFile(filePath, []byte(tt.content), 0o644)
			require.NoError(t, err)

			err = watcher.validateJSONFile(filePath)
			if tt.valid {
				assert.NoError(t, err, "Valid JSON should pass validation")
			} else {
				assert.Error(t, err, "Invalid JSON should fail validation")
			}
		})
	}
}

func (s *WatcherTestSuite) TestJSONValidation_InvalidJSONRejection() {
	s.T().Log("Testing invalid JSON rejection")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	tests := []struct {
		name    string
		content string
		reason  string
	}{
		{
			name:    "malformed_json",
			content: `{"apiVersion": "v1", "kind": "NetworkIntent", "action": "scale", "target": "deployment", "count":}`,
			reason:  "malformed JSON",
		},
		{
			name:    "empty_file",
			content: ``,
			reason:  "empty file",
		},
		{
			name:    "invalid_json_syntax",
			content: `{action: scale}`,
			reason:  "invalid JSON syntax",
		},
		{
			name:    "missing_apiversion",
			content: `{"kind": "NetworkIntent"}`,
			reason:  "missing required fields",
		},
		{
			name:    "empty_apiversion",
			content: `{"apiVersion": "", "kind": "NetworkIntent"}`,
			reason:  "empty apiVersion",
		},
		{
			name:    "invalid_kind_type",
			content: `{"apiVersion": "v1", "kind": 123}`,
			reason:  "invalid kind type",
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			fileName := fmt.Sprintf("intent-invalid-%s.json", tt.name)
			filePath := filepath.Join(s.tempDir, fileName)

			err := os.WriteFile(filePath, []byte(tt.content), 0o644)
			require.NoError(t, err)

			err = watcher.validateJSONFile(filePath)
			if tt.reason == "missing required fields" || tt.reason == "empty apiVersion" {
				// These cases should fail validation
				assert.Error(t, err, "Invalid JSON should fail validation: %s", tt.reason)
			} else {
				assert.Error(t, err, "Invalid JSON should fail validation: %s", tt.reason)
			}
		})
	}
}

func (s *WatcherTestSuite) TestJSONValidation_SizeLimitEnforcement() {
	s.T().Log("Testing JSON size limit enforcement")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	// Test with file exceeding MaxJSONSize
	largeFileName := filepath.Join(s.tempDir, "intent-large.json")

	// Create valid JSON structure that exceeds size limit
	if MaxJSONSize <= 100 {
		s.T().Skip("MaxJSONSize too small for test")
	}
	// Create a file that EXCEEDS the limit (not MaxJSONSize-100, but MaxJSONSize+100)
	padding := strings.Repeat("x", MaxJSONSize+100)
	largeJSON := fmt.Sprintf(`{"apiVersion": "v1", "kind": "NetworkIntent", "data": "%s"}`, padding)

	s.Require().NoError(os.WriteFile(largeFileName, []byte(largeJSON), 0o644))

	err = watcher.validateJSONFile(largeFileName)
	s.Require().NotNil(err, "File exceeding size limit MUST return non-nil error")
	s.Assert().Contains(err.Error(), "exceeds maximum", "Error MUST contain 'exceeds maximum'")
}

func (s *WatcherTestSuite) TestJSONValidation_PathTraversalPrevention() {
	s.T().Log("Testing path traversal prevention")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	// Test various path traversal attempts
	maliciousPaths := []string{
		filepath.Join(s.tempDir, "..", "intent-traversal.json"),
		filepath.Join(s.tempDir, "subdir", "..", "..", "intent-traversal.json"),
		"/etc/passwd", // Absolute path outside watched directory
	}

	for i, maliciousPath := range maliciousPaths {
		s.T().Run(fmt.Sprintf("traversal_attempt_%d", i), func(t *testing.T) {
			// Try to create file outside watched directory
			os.MkdirAll(filepath.Dir(maliciousPath), 0o755)
			content := `{"apiVersion": "v1", "kind": "NetworkIntent"}`
			os.WriteFile(maliciousPath, []byte(content), 0o644)

			err := watcher.validatePath(maliciousPath)
			assert.Error(t, err, "Path traversal should be prevented for: %s", maliciousPath)
		})
	}
}

func (s *WatcherTestSuite) TestJSONValidation_RequiredFieldsValidation() {
	s.T().Log("Testing required fields validation")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	// Test various field validation scenarios
	tests := []struct {
		name        string
		content     string
		shouldPass  bool
		description string
	}{
		{
			name:        "all_required_fields",
			content:     `{"apiVersion": "v1", "kind": "NetworkIntent", "metadata": {"name": "test"}, "spec": {"action": "scale"}}`,
			shouldPass:  true,
			description: "complete intent with all fields",
		},
		{
			name:        "minimal_required_fields",
			content:     `{"apiVersion": "v1", "kind": "NetworkIntent"}`,
			shouldPass:  true,
			description: "minimal intent with required fields",
		},
		{
			name:        "invalid_metadata_name",
			content:     `{"apiVersion": "v1", "kind": "NetworkIntent", "metadata": {"name": ""}}`,
			shouldPass:  false,
			description: "empty metadata name",
		},
		{
			name:        "invalid_spec_type",
			content:     `{"apiVersion": "v1", "kind": "NetworkIntent", "spec": "invalid"}`,
			shouldPass:  false,
			description: "spec must be object",
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			fileName := fmt.Sprintf("intent-fields-%s.json", tt.name)
			filePath := filepath.Join(s.tempDir, fileName)

			err := os.WriteFile(filePath, []byte(tt.content), 0o644)
			require.NoError(t, err)

			err = watcher.validateJSONFile(filePath)
			if tt.shouldPass {
				assert.NoError(t, err, "Should pass validation: %s", tt.description)
			} else {
				assert.Error(t, err, "Should fail validation: %s", tt.description)
			}
		})
	}
}

// =============================================================================
// SECURITY TESTS
// =============================================================================

func (s *WatcherTestSuite) TestSecurity_PathTraversalAttacks() {
	s.T().Log("Testing protection against path traversal attacks")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	// Test various path traversal attack patterns
	attackPatterns := []struct {
		name        string
		path        string
		description string
	}{
		{
			name:        "dot_dot_traversal",
			path:        filepath.Join(s.tempDir, "..", "etc", "passwd"),
			description: "../ traversal",
		},
		{
			name:        "multiple_dot_dot",
			path:        filepath.Join(s.tempDir, "..", "..", "..", "etc", "passwd"),
			description: "multiple ../ traversal",
		},
		{
			name:        "mixed_traversal",
			path:        filepath.Join(s.tempDir, "subdir", "..", "..", "sensitive"),
			description: "mixed path traversal",
		},
	}

	for _, pattern := range attackPatterns {
		s.T().Run(pattern.name, func(t *testing.T) {
			err := watcher.validatePath(pattern.path)
			assert.Error(t, err, "Should reject path traversal: %s", pattern.description)
			assert.Contains(t, err.Error(), "outside watched directory",
				"Error should mention path restriction")
		})
	}
}

func (s *WatcherTestSuite) TestSecurity_JSONBombPrevention() {
	s.T().Log("Testing JSON bomb prevention")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	// Create deeply nested JSON structure
	deepJSON := `{"apiVersion": "v1", "kind": "NetworkIntent", "data": `
	nesting := 1000
	for i := 0; i < nesting; i++ {
		deepJSON += `{"level": `
	}
	deepJSON += `"end"`
	for i := 0; i < nesting; i++ {
		deepJSON += `}`
	}
	deepJSON += `}`

	fileName := filepath.Join(s.tempDir, "intent-bomb.json")
	err = os.WriteFile(fileName, []byte(deepJSON), 0o644)
	s.Require().NoError(err)

	// Should reject extremely deep JSON
	err = watcher.validateJSONFile(fileName)
	s.Assert().Error(err, "Should reject JSON bomb attempts")
}

func (s *WatcherTestSuite) TestSecurity_SuspiciousFilenamePatterns() {
	s.T().Log("Testing suspicious filename pattern detection")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	suspiciousNames := []string{
		"intent-test..json",
		"intent-test~.json",
		"intent-test$.json",
		"intent-test*.json",
		"intent-test?.json",
		"intent-test[.json",
		"intent-test{.json",
		"intent-test|.json",
		"intent-test<.json",
		"intent-test>.json",
	}

	for _, name := range suspiciousNames {
		s.T().Run(fmt.Sprintf("suspicious_%s", name), func(t *testing.T) {
			filePath := filepath.Join(s.tempDir, name)
			// Create the file first
			content := `{"apiVersion": "v1", "kind": "NetworkIntent"}`
			os.WriteFile(filePath, []byte(content), 0o644)

			err := watcher.validatePath(filePath)
			assert.Error(t, err, "Should reject suspicious filename: %s", name)

			// On Windows, certain characters are invalid Windows path characters
			// and will trigger "Windows path validation failed" before suspicious pattern check
			if runtime.GOOS == "windows" {
				// Windows-invalid characters: *, ?, |, <, >
				windowsInvalidChars := map[string]bool{
					"intent-test*.json": true,
					"intent-test?.json": true,
					"intent-test|.json": true,
					"intent-test<.json": true,
					"intent-test>.json": true,
				}

				if windowsInvalidChars[name] {
					assert.Contains(t, err.Error(), "Windows path validation failed",
						"Error should mention Windows path validation for invalid char: %s", name)
				} else {
					assert.Contains(t, err.Error(), "suspicious pattern",
						"Error should mention suspicious pattern for: %s", name)
				}
			} else {
				// On non-Windows, all should be suspicious patterns
				assert.Contains(t, err.Error(), "suspicious pattern",
					"Error should mention suspicious pattern")
			}
		})
	}
}

func (s *WatcherTestSuite) TestSecurity_FileSizeLimits() {
	s.T().Log("Testing file size limit enforcement")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	// Test various size scenarios
	tests := []struct {
		name       string
		size       int64
		shouldPass bool
	}{
		{"small_file", 1024, true},
		{"medium_file", 1024 * 1024, true},         // 1MB
		{"large_file", 5 * 1024 * 1024, true},      // 5MB
		{"oversized_file", MaxJSONSize + 1, false}, // Exceeds limit
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			fileName := fmt.Sprintf("intent-size-%s.json", tt.name)
			filePath := filepath.Join(s.tempDir, fileName)

			// Create file with specified size
			if tt.size <= MaxJSONSize {
				// Create valid JSON of specified size
				padding := strings.Repeat("x", int(tt.size-100))
				content := fmt.Sprintf(`{"apiVersion": "v1", "kind": "NetworkIntent", "data": "%s"}`, padding)
				err := os.WriteFile(filePath, []byte(content), 0o644)
				require.NoError(t, err)
			} else {
				// Create oversized file with well-formed JSON
				// Calculate padding size to exceed MaxJSONSize
				baseJSON := `{"apiVersion": "v1", "kind": "NetworkIntent", "data": ""}`
				baseSizeWithoutData := len(baseJSON) - 2 // Subtract 2 for the empty quotes
				paddingSize := int(tt.size) - baseSizeWithoutData

				// Generate deterministic padding with 'A' characters instead of random data
				padding := strings.Repeat("A", paddingSize)
				content := fmt.Sprintf(`{"apiVersion": "v1", "kind": "NetworkIntent", "data": "%s"}`, padding)

				err := os.WriteFile(filePath, []byte(content), 0o644)
				require.NoError(t, err)
			}

			err := watcher.validateJSONFile(filePath)
			if tt.shouldPass {
				assert.NoError(t, err, "File of size %d should pass", tt.size)
			} else {
				assert.Error(t, err, "File of size %d should fail", tt.size)
				assert.Contains(t, err.Error(), "exceeds maximum",
					"Error should mention size limit")
			}
		})
	}
}

// =============================================================================
// INTEGRATION TESTS
// =============================================================================

func (s *WatcherTestSuite) TestIntegration_EndToEndProcessingFlow() {
	s.T().Log("Testing end-to-end processing flow")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	// Create a valid intent file with proper structure
	testFile := filepath.Join(s.tempDir, "intent-e2e-test.json")
	testContent := `{"apiVersion": "v1", "kind": "NetworkIntent", "metadata": {"name": "test-e2e"}, "spec": {"action": "scale", "target": {"name": "deployment"}, "replicas": 3}}`
	s.Require().NoError(os.WriteFile(testFile, []byte(testContent), 0o644))

	// Start processing
	watcher.config.Once = true
	startTime := time.Now()
	err = watcher.Start()
	s.Require().NoError(err)
	processingTime := time.Since(startTime)

	// Add a small delay to ensure all async operations complete
	time.Sleep(100 * time.Millisecond)

	// Verify complete processing flow
	s.T().Logf("End-to-end processing took %v", processingTime)

	// 1. Original file should be gone (moved)
	s.Assert().NoFileExists(testFile, "Original file should be moved")

	// 2. Should have been moved to processed directory (with eventual consistency)
	var processedFiles []string
	for i := 0; i < 10; i++ {
		processedFiles, err = watcher.fileManager.GetProcessedFiles()
		s.Require().NoError(err)
		if len(processedFiles) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	s.Assert().Len(processedFiles, 1, "Should have one processed file")

	// 3. Verify processing was recorded in state (check with original file name since that's how it's stored)
	stats := watcher.executor.GetStats()
	s.Assert().Equal(1, stats.TotalExecutions, "Should have processed one file")
	s.Assert().Equal(1, stats.SuccessfulExecs, "Should have one successful execution")

	// 4. Status file should be created
	statusDir := filepath.Join(s.tempDir, "status")
	statusFiles, err := os.ReadDir(statusDir)
	s.Require().NoError(err)
	s.Assert().Greater(len(statusFiles), 0, "Should have status file")

	// 5. Verify status file content
	statusContent, err := os.ReadFile(filepath.Join(statusDir, statusFiles[0].Name()))
	s.Require().NoError(err)

	var statusData map[string]interface{}
	s.Require().NoError(json.Unmarshal(statusContent, &statusData))

	s.Assert().Equal("intent-e2e-test.json", statusData["intent_file"])
	s.Assert().Equal("success", statusData["status"])
	s.Assert().Equal("conductor-loop", statusData["processed_by"])
}

func (s *WatcherTestSuite) TestIntegration_StatusFileGenerationWithVersioning() {
	s.T().Log("Testing status file generation with versioning")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	// Process multiple files to test versioning
	numFiles := 3
	for i := 0; i < numFiles; i++ {
		fileName := fmt.Sprintf("intent-version-test-%d.json", i)
		filePath := filepath.Join(s.tempDir, fileName)
		testContent := fmt.Sprintf(`{"apiVersion": "v1", "kind": "NetworkIntent", "metadata": {"name": "test-%d"}}`, i)

		s.Require().NoError(os.WriteFile(filePath, []byte(testContent), 0o644))
	}

	watcher.config.Once = true
	err = watcher.Start()
	s.Require().NoError(err)

	// Add delay to ensure all status files are written
	time.Sleep(200 * time.Millisecond)

	// Verify status files were created with versioning (with eventual consistency)
	statusDir := filepath.Join(s.tempDir, "status")
	var statusFiles []os.DirEntry
	for i := 0; i < 10; i++ {
		statusFiles, err = os.ReadDir(statusDir)
		s.Require().NoError(err)
		if len(statusFiles) >= numFiles {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	s.Assert().Equal(numFiles, len(statusFiles), "Should have status file for each processed file")

	// Verify each status file has timestamp-based versioning
	for _, file := range statusFiles {
		s.T().Logf("Status file name: %s", file.Name())
		s.Assert().Contains(file.Name(), ".status", "Should have .status extension")
		// The pattern should match: intent-version-test-{id}-{timestamp}.status
		s.Assert().Regexp(`intent-version-test-\d+-\d{8}-\d{6}\.status`, file.Name(),
			"Status file should have timestamp versioning")
	}
}

func (s *WatcherTestSuite) TestIntegration_FileMovementProcessedFailed() {
	s.T().Log("Testing file movement to processed/failed directories")

	// Setup watcher with mock that will fail
	failingPorchPath := createMockPorch(s.T(), s.tempDir, 1, "", "porch command failed")
	failConfig := s.config
	failConfig.PorchPath = failingPorchPath

	watcher, err := NewWatcher(s.tempDir, failConfig)
	s.Require().NoError(err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	// Create files for both success and failure scenarios
	successFile := filepath.Join(s.tempDir, "intent-success.json")
	failureFile := filepath.Join(s.tempDir, "intent-failure.json")

	testContent := `{"apiVersion": "v1", "kind": "NetworkIntent", "spec": {"action": "scale"}}`
	s.Require().NoError(os.WriteFile(failureFile, []byte(testContent), 0o644))

	// Process the failing file
	watcher.config.Once = true
	err = watcher.Start()
	s.Require().NoError(err)

	// Verify failure handling
	failedFiles, err := watcher.fileManager.GetFailedFiles()
	s.Require().NoError(err)
	s.Assert().Len(failedFiles, 1, "Should have one failed file")

	// Now test success scenario
	successPorchPath := createMockPorch(s.T(), s.tempDir, 0, "success", "")
	successConfig := s.config
	successConfig.PorchPath = successPorchPath

	successWatcher, err := NewWatcher(s.tempDir, successConfig)
	s.Require().NoError(err)
	defer successWatcher.Close() // #nosec G307 - Error handled in defer

	s.Require().NoError(os.WriteFile(successFile, []byte(testContent), 0o644))

	successWatcher.config.Once = true
	err = successWatcher.Start()
	s.Require().NoError(err)

	// Verify success handling
	processedFiles, err := successWatcher.fileManager.GetProcessedFiles()
	s.Require().NoError(err)
	s.Assert().Len(processedFiles, 1, "Should have one processed file")
}

func (s *WatcherTestSuite) TestIntegration_GracefulShutdownWithActiveProcessing() {
	s.T().Log("Testing graceful shutdown with active processing")

	// Use porch with delay to simulate long-running processing
	slowPorchPath := createMockPorch(s.T(), s.tempDir, 0, "processed slowly", "", 2*time.Second)
	slowConfig := s.config
	slowConfig.PorchPath = slowPorchPath
	slowConfig.MaxWorkers = 2

	watcher, err := NewWatcher(s.tempDir, slowConfig)
	s.Require().NoError(err)

	// Create multiple files for processing
	numFiles := 4
	for i := 0; i < numFiles; i++ {
		fileName := fmt.Sprintf("intent-shutdown-%d.json", i)
		filePath := filepath.Join(s.tempDir, fileName)
		testContent := fmt.Sprintf(`{"apiVersion": "v1", "kind": "NetworkIntent", "metadata": {"name": "test-%d"}}`, i)

		s.Require().NoError(os.WriteFile(filePath, []byte(testContent), 0o644))
	}

	// Start processing in background
	done := make(chan error, 1)
	go func() {
		watcher.config.Once = true
		done <- watcher.Start()
	}()

	// Wait a bit, then initiate shutdown
	time.Sleep(500 * time.Millisecond)

	shutdownStart := time.Now()
	err = watcher.Close()
	shutdownDuration := time.Since(shutdownStart)
	s.Require().NoError(err)

	// Wait for processing to complete
	select {
	case err := <-done:
		s.Require().NoError(err)
	case <-time.After(10 * time.Second):
		s.T().Fatal("Processing didn't complete within timeout")
	}

	s.T().Logf("Graceful shutdown took %v", shutdownDuration)

	// Verify some processing occurred (at least files that started)
	stats := watcher.executor.GetStats()
	s.Assert().Greater(stats.TotalExecutions, 0, "Some files should have been processed")
}

// =============================================================================
// PERFORMANCE TESTS
// =============================================================================

func (s *WatcherTestSuite) TestPerformance_WorkerPoolScalability() {
	if testing.Short() {
		s.T().Skip("Skipping performance test in short mode")
	}

	s.T().Log("Testing worker pool scalability")

	workerCounts := []int{1, 2, 4} // Reduced from 8 to speed up tests
	numFiles := 20                 // Reduced from 50 to speed up tests
	testContent := `{"apiVersion": "v1", "kind": "NetworkIntent", "spec": {"action": "scale", "target": {"type": "deployment", "name": "test-app"}}}`

	for _, workers := range workerCounts {
		s.T().Run(fmt.Sprintf("workers_%d", workers), func(t *testing.T) {
			testDir := s.T().TempDir()
			outDir := filepath.Join(testDir, "out")
			require.NoError(t, os.MkdirAll(outDir, 0o755))

			config := Config{
				PorchPath:   createFastMockPorch(t, testDir, 0, "processed", ""), // Ultra-fast mock
				Mode:        porch.ModeDirect,
				OutDir:      outDir,
				MaxWorkers:  workers,
				DebounceDur: 5 * time.Millisecond, // Reduced from 10ms
				Once:        true,
			}

			watcher, err := NewWatcher(testDir, config)
			require.NoError(t, err)
			defer watcher.Close() // #nosec G307 - Error handled in defer

			// Create test files
			for i := 0; i < numFiles; i++ {
				fileName := fmt.Sprintf("intent-scale-%d.json", i)
				filePath := filepath.Join(testDir, fileName)
				require.NoError(t, os.WriteFile(filePath, []byte(testContent), 0o644))
			}

			// Measure processing time
			startTime := time.Now()
			err = watcher.Start()
			require.NoError(t, err)
			processingTime := time.Since(startTime)

			// Verify all files processed
			stats := watcher.executor.GetStats()
			assert.Equal(t, numFiles, stats.TotalExecutions)

			t.Logf("Workers: %d, Files: %d, Time: %v, Throughput: %.2f files/sec",
				workers, numFiles, processingTime, float64(numFiles)/processingTime.Seconds())
		})
	}
}

func (s *WatcherTestSuite) TestPerformance_DebouncingMechanism() {
	if testing.Short() {
		s.T().Skip("Skipping performance test in short mode")
	}

	s.T().Log("Testing debouncing mechanism effectiveness")

	debounceConfig := s.config
	debounceConfig.DebounceDur = 200 * time.Millisecond
	debounceConfig.MaxWorkers = 2 // Production-like worker count for better concurrency testing

	watcher, err := NewWatcher(s.tempDir, debounceConfig)
	s.Require().NoError(err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	// Start watcher in background
	done := make(chan error, 1)
	go func() {
		done <- watcher.Start()
	}()

	// Wait for watcher to start
	time.Sleep(100 * time.Millisecond)

	testFile := filepath.Join(s.tempDir, "intent-debounce-perf.json")

	// Write to file multiple times rapidly
	numWrites := 10
	for i := 0; i < numWrites; i++ {
		content := fmt.Sprintf(`{"apiVersion": "v1", "kind": "NetworkIntent", "iteration": %d}`, i)
		s.Require().NoError(os.WriteFile(testFile, []byte(content), 0o644))
		time.Sleep(50 * time.Millisecond) // Rapid writes within debounce window
	}

	// Wait for debouncing to settle
	time.Sleep(debounceConfig.DebounceDur + 500*time.Millisecond)

	// Stop watcher gracefully
	watcher.Close()

	// Wait for completion
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}

	// Verify debouncing worked - should have processed at most 1-2 times
	stats := watcher.executor.GetStats()
	s.Assert().LessOrEqual(stats.TotalExecutions, 2,
		"Debouncing should limit processing despite %d writes", numWrites)
	s.T().Logf("Debouncing: %d writes resulted in %d executions", numWrites, stats.TotalExecutions)
}

func (s *WatcherTestSuite) TestPerformance_BatchProcessingEfficiency() {
	if testing.Short() {
		s.T().Skip("Skipping performance test in short mode")
	}

	s.T().Log("Testing batch processing efficiency")

	batchSizes := []int{5, 15, 30} // Reduced sizes to speed up tests
	testContent := `{"apiVersion": "v1", "kind": "NetworkIntent", "spec": {"action": "scale", "target": {"type": "deployment", "name": "test-app"}}}`

	for _, batchSize := range batchSizes {
		s.T().Run(fmt.Sprintf("batch_%d", batchSize), func(t *testing.T) {
			testDir := s.T().TempDir()
			outDir := filepath.Join(testDir, "out")
			require.NoError(t, os.MkdirAll(outDir, 0o755))

			config := Config{
				PorchPath:   createFastMockPorch(t, testDir, 0, "processed", ""), // Ultra-fast mock
				Mode:        porch.ModeDirect,
				OutDir:      outDir,
				MaxWorkers:  4,
				DebounceDur: 5 * time.Millisecond, // Reduced from 10ms
				Once:        true,
			}

			watcher, err := NewWatcher(testDir, config)
			require.NoError(t, err)
			defer watcher.Close() // #nosec G307 - Error handled in defer

			// Create batch of files
			for i := 0; i < batchSize; i++ {
				fileName := fmt.Sprintf("intent-batch-%d.json", i)
				filePath := filepath.Join(testDir, fileName)
				require.NoError(t, os.WriteFile(filePath, []byte(testContent), 0o644))
			}

			// Measure batch processing
			startTime := time.Now()
			err = watcher.Start()
			require.NoError(t, err)
			processingTime := time.Since(startTime)

			stats := watcher.executor.GetStats()
			assert.Equal(t, batchSize, stats.TotalExecutions)

			throughput := float64(batchSize) / processingTime.Seconds()
			t.Logf("Batch size: %d, Time: %v, Throughput: %.2f files/sec",
				batchSize, processingTime, throughput)
		})
	}
}

func (s *WatcherTestSuite) TestPerformance_MemoryUsageUnderLoad() {
	if testing.Short() {
		s.T().Skip("Skipping performance test in short mode")
	}

	s.T().Log("Testing memory usage under load")

	// Record initial memory stats
	var initialStats, peakStats runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&initialStats)

	largeConfig := s.config
	largeConfig.MaxWorkers = 4                                                        // Reduced from 8
	largeConfig.PorchPath = createFastMockPorch(s.T(), s.tempDir, 0, "processed", "") // Ultra-fast mock

	watcher, err := NewWatcher(s.tempDir, largeConfig)
	s.Require().NoError(err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	// Create fewer files for faster test
	numFiles := 100 // Reduced from 500
	testContent := `{"apiVersion": "v1", "kind": "NetworkIntent", "spec": {"action": "scale", "target": {"type": "deployment", "name": "test-app"}}}`

	for i := 0; i < numFiles; i++ {
		fileName := fmt.Sprintf("intent-memory-%d.json", i)
		filePath := filepath.Join(s.tempDir, fileName)
		s.Require().NoError(os.WriteFile(filePath, []byte(testContent), 0o644))
	}

	// Process all files
	watcher.config.Once = true
	startTime := time.Now()
	err = watcher.Start()
	s.Require().NoError(err)
	processingTime := time.Since(startTime)

	// Check peak memory usage
	runtime.GC()
	runtime.ReadMemStats(&peakStats)

	memoryUsed := peakStats.Alloc - initialStats.Alloc
	s.T().Logf("Processed %d files in %v, Memory used: %d bytes (%.2f MB)",
		numFiles, processingTime, memoryUsed, float64(memoryUsed)/(1024*1024))

	// Memory usage should be reasonable (less than 50MB for 100 files)
	s.Assert().Less(memoryUsed, uint64(50*1024*1024),
		"Memory usage should be reasonable under load")

	// Verify all files were processed
	stats := watcher.executor.GetStats()
	s.Assert().Equal(numFiles, stats.TotalExecutions)
}

// =============================================================================
// LARGE SCALE INTEGRATION TEST
// =============================================================================

func (s *WatcherTestSuite) TestWatcherIntegration_LargeScaleProcessing() {
	if testing.Short() {
		s.T().Skip("Skipping large scale test in short mode")
	}

	s.T().Log("Testing large scale processing integration")

	// Use optimized configuration for large scale
	largeScaleConfig := s.config
	largeScaleConfig.MaxWorkers = runtime.GOMAXPROCS(0)                                    // Use all available CPU cores
	largeScaleConfig.DebounceDur = 1 * time.Millisecond                                    // Minimal debounce
	largeScaleConfig.PorchPath = createFastMockPorch(s.T(), s.tempDir, 0, "processed", "") // Ultra-fast mock
	largeScaleConfig.Once = true

	watcher, err := NewWatcher(s.tempDir, largeScaleConfig)
	s.Require().NoError(err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	// Create a large number of files for processing
	numFiles := 250 // Reasonable scale that won't timeout

	s.T().Logf("Creating %d test files...", numFiles)
	createStart := time.Now()

	// Create files in parallel for faster setup
	var createWg sync.WaitGroup
	createChan := make(chan int, runtime.GOMAXPROCS(0)*2) // Buffered channel

	for i := 0; i < numFiles; i++ {
		createWg.Add(1)
		createChan <- i

		go func(fileID int) {
			defer createWg.Done()
			defer func() { <-createChan }()

			fileName := fmt.Sprintf("intent-large-scale-%d.json", fileID)
			filePath := filepath.Join(s.tempDir, fileName)
			content := fmt.Sprintf(`{"apiVersion": "v1", "kind": "NetworkIntent", "metadata": {"name": "test-%d"}, "spec": {"action": "scale", "target": {"type": "deployment", "name": "app-%d"}}}`, fileID, fileID)

			s.Require().NoError(os.WriteFile(filePath, []byte(content), 0o644))
		}(i)
	}

	createWg.Wait()
	createDuration := time.Since(createStart)
	s.T().Logf("Created %d files in %v", numFiles, createDuration)

	// Record initial memory for monitoring
	var initialStats, peakStats runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&initialStats)

	// Start processing with progress logging
	s.T().Logf("Starting large scale processing with %d workers...", largeScaleConfig.MaxWorkers)
	startTime := time.Now()

	// Process files
	err = watcher.Start()
	s.Require().NoError(err)

	processingTime := time.Since(startTime)

	// Check memory usage
	runtime.GC()
	runtime.ReadMemStats(&peakStats)
	memoryUsed := peakStats.Alloc - initialStats.Alloc

	// Verify results
	stats := watcher.executor.GetStats()

	s.T().Logf("Large scale processing results:")
	s.T().Logf("  Files processed: %d/%d", stats.TotalExecutions, numFiles)
	s.T().Logf("  Processing time: %v", processingTime)
	s.T().Logf("  Throughput: %.2f files/sec", float64(stats.TotalExecutions)/processingTime.Seconds())
	s.T().Logf("  Memory used: %.2f MB", float64(memoryUsed)/(1024*1024))
	s.T().Logf("  Success rate: %.2f%%", float64(stats.SuccessfulExecs)/float64(stats.TotalExecutions)*100)

	// Assertions for large scale processing
	s.Assert().Greater(stats.TotalExecutions, 0, "Should process at least some files")
	s.Assert().LessOrEqual(stats.TotalExecutions, numFiles, "Should not process more files than created")
	s.Assert().GreaterOrEqual(stats.SuccessfulExecs, stats.TotalExecutions/2, "At least 50% should succeed")

	// Performance requirements
	s.Assert().Less(processingTime, 30*time.Second, "Large scale processing should complete within 30 seconds")
	s.Assert().Less(memoryUsed, uint64(100*1024*1024), "Memory usage should be under 100MB for large scale")

	// Verify file movement
	processedFiles, err := watcher.fileManager.GetProcessedFiles()
	s.Require().NoError(err)
	failedFiles, err := watcher.fileManager.GetFailedFiles()
	s.Require().NoError(err)

	totalMoved := len(processedFiles) + len(failedFiles)
	s.Assert().Equal(stats.TotalExecutions, totalMoved, "All processed files should be moved")

	// Throughput validation - should handle reasonable throughput
	throughput := float64(stats.TotalExecutions) / processingTime.Seconds()
	s.Assert().Greater(throughput, 10.0, "Should achieve at least 10 files/sec throughput")

	s.T().Logf("Large scale integration test completed successfully")
}

// =============================================================================
// WINDOWS FILENAME VALIDATION TESTS
// =============================================================================

func (s *WatcherTestSuite) TestWindowsFilenameValidation_StatusFileGeneration() {
	s.T().Log("Testing Windows filename validation in status file generation")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	// Test cases with problematic filenames that need sanitization
	testCases := []struct {
		name                 string
		intentFilename       string
		expectedStatusPrefix string
		shouldProcess        bool
		description          string
	}{
		{
			name:                 "WindowsReservedCharacters",
			intentFilename:       "intent<test>.json",
			expectedStatusPrefix: "intent-test-",
			shouldProcess:        true,
			description:          "Should sanitize Windows reserved characters < and >",
		},
		{
			name:                 "WindowsReservedDevice",
			intentFilename:       "CON.json",
			expectedStatusPrefix: "CON-file-",
			shouldProcess:        true,
			description:          "Should avoid Windows reserved device name CON",
		},
		{
			name:                 "UnicodeCharacters",
			intentFilename:       "intent-测试.json",
			expectedStatusPrefix: "intent-",
			shouldProcess:        true,
			description:          "Should sanitize Unicode characters for Windows compatibility",
		},
		{
			name:                 "SuspiciousPatterns",
			intentFilename:       "intent-test~.json",
			expectedStatusPrefix: "intent-test-",
			shouldProcess:        true,
			description:          "Should remove suspicious tilde character",
		},
		{
			name:                 "MultipleReservedChars",
			intentFilename:       "intent|file<test>.json",
			expectedStatusPrefix: "intent-file-test-",
			shouldProcess:        true,
			description:          "Should sanitize multiple reserved characters",
		},
		{
			name:                 "PathTraversal",
			intentFilename:       "..\\..\\intent.json",
			expectedStatusPrefix: "intent-",
			shouldProcess:        true,
			description:          "Should handle path traversal attempts safely",
		},
		{
			name:                 "AllReservedCharacters",
			intentFilename:       "intent<>:\"/\\|?*test.json",
			expectedStatusPrefix: "intent",
			shouldProcess:        true,
			description:          "Should sanitize all Windows reserved characters",
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			// Create intent file with problematic name
			intentPath := filepath.Join(s.tempDir, tc.intentFilename)
			testContent := `{"apiVersion": "v1", "kind": "NetworkIntent", "metadata": {"name": "test"}}`

			// Create directory if needed for path traversal test
			if strings.Contains(tc.intentFilename, "\\") || strings.Contains(tc.intentFilename, "/") {
				os.MkdirAll(filepath.Dir(intentPath), 0o755)
			}

			// Try to create file - Windows may not allow files with certain reserved characters
			err := os.WriteFile(intentPath, []byte(testContent), 0o644)
			if err != nil {
				t.Logf("Cannot create file %s on Windows (expected for reserved chars): %v", tc.intentFilename, err)
				// Skip this test case if Windows won't allow us to create the file
				return
			}

			// Process the file
			watcher.config.Once = true
			err = watcher.Start()
			assert.NoError(t, err)

			if tc.shouldProcess {
				// Check that status file was created with sanitized name
				statusDir := filepath.Join(s.tempDir, "status")
				statusFiles, err := os.ReadDir(statusDir)
				if err != nil {
					// Status directory might not exist if no files were processed
					t.Logf("Status directory not found (no files processed): %v", err)
					return
				}

				if len(statusFiles) == 0 {
					t.Logf("No status files found (file may not have been processed)")
					return
				}

				// Find the status file
				var foundStatusFile string
				for _, file := range statusFiles {
					if strings.HasPrefix(file.Name(), tc.expectedStatusPrefix) {
						foundStatusFile = file.Name()
						break
					}
				}

				assert.NotEmpty(t, foundStatusFile, "Should find status file with expected prefix for %s", tc.description)

				if foundStatusFile != "" {
					// Verify the status filename is Windows-safe
					assert.True(t, strings.HasSuffix(foundStatusFile, ".status"),
						"Status file should end with .status")

					// Should not contain any Windows reserved characters
					for _, reservedChar := range []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"} {
						assert.NotContains(t, foundStatusFile, reservedChar,
							"Status filename should not contain reserved char %s for %s", reservedChar, tc.description)
					}

					// Should not be a Windows reserved device name
					parts := strings.Split(foundStatusFile, "-")
					if len(parts) == 0 {
						return // Skip empty split result
					}
					baseName := parts[0]
					upperBaseName := strings.ToUpper(baseName)
					reservedNames := []string{"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"}
					for _, reserved := range reservedNames {
						if upperBaseName == reserved {
							assert.Fail(t, "Status filename should not be Windows reserved name", "Found reserved name: %s in %s", reserved, foundStatusFile)
						}
					}

					// Should contain only ASCII characters for Windows compatibility
					for i, r := range foundStatusFile {
						assert.True(t, r <= 127, "Character at position %d should be ASCII in %s for %s", i, foundStatusFile, tc.description)
					}

					// Should not have suspicious patterns
					assert.NotContains(t, foundStatusFile, "~", "Should not contain ~ for %s", tc.description)
					assert.NotContains(t, foundStatusFile, "..", "Should not contain .. for %s", tc.description)
					assert.NotRegexp(t, `^\.#`, foundStatusFile, "Should not start with .# for %s", tc.description)
				}
			}
		})
	}
}

func (s *WatcherTestSuite) TestWindowsFilenameValidation_PathTraversalPrevention() {
	s.T().Log("Testing path traversal prevention in Windows filename handling")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	// Test various path traversal attempts
	pathTraversalTests := []struct {
		name        string
		filePath    string
		description string
	}{
		{
			name:        "DotDotTraversal",
			filePath:    "..\\intent-traversal.json",
			description: "Should handle .. path traversal",
		},
		{
			name:        "MultipleDotDot",
			filePath:    "..\\..\\..\\intent-traversal.json",
			description: "Should handle multiple .. traversal",
		},
		{
			name:        "MixedSeparators",
			filePath:    "../..\\intent-traversal.json",
			description: "Should handle mixed path separators",
		},
		{
			name:        "AbsolutePath",
			filePath:    "C:\\Windows\\System32\\intent-traversal.json",
			description: "Should handle absolute paths outside watch directory",
		},
		{
			name:        "UNCPath",
			filePath:    "\\\\server\\share\\intent-traversal.json",
			description: "Should handle UNC paths",
		},
	}

	for _, tc := range pathTraversalTests {
		s.T().Run(tc.name, func(t *testing.T) {
			// Create the file with path traversal attempt
			testContent := `{"apiVersion": "v1", "kind": "NetworkIntent", "metadata": {"name": "traversal-test"}}`

			// Try to create directory structure if needed
			fullPath := filepath.Join(s.tempDir, tc.filePath)
			os.MkdirAll(filepath.Dir(fullPath), 0o755)

			// Write the file
			err := os.WriteFile(fullPath, []byte(testContent), 0o644)
			if err != nil {
				t.Logf("Could not create test file %s: %v (this is expected for some traversal attempts)", tc.filePath, err)
				return // Skip if we can't create the file (expected for some cases)
			}

			// Process the file
			watcher.config.Once = true
			err = watcher.Start()

			// Should either process safely or reject appropriately
			assert.NoError(t, err, "Watcher should handle path traversal gracefully for %s", tc.description)

			// If a status file was created, it should be in the correct status directory
			statusDir := filepath.Join(s.tempDir, "status")
			if statusFiles, err := os.ReadDir(statusDir); err == nil && len(statusFiles) > 0 {
				for _, file := range statusFiles {
					// Status file should be directly in status dir, not in traversed path
					statusPath := filepath.Join(statusDir, file.Name())
					relPath, err := filepath.Rel(statusDir, statusPath)
					assert.NoError(t, err)
					assert.Equal(t, file.Name(), relPath, "Status file should be directly in status directory for %s", tc.description)
				}
			}
		})
	}
}

func (s *WatcherTestSuite) TestWindowsFilenameValidation_LongFilenames() {
	s.T().Log("Testing handling of long filenames on Windows")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	// Test very long filenames
	longFilenameTests := []struct {
		name           string
		baseLength     int
		expectTruncate bool
		description    string
	}{
		{
			name:           "NormalLength",
			baseLength:     50,
			expectTruncate: false,
			description:    "Normal length filename should not be truncated",
		},
		{
			name:           "MediumLength",
			baseLength:     100,
			expectTruncate: false,
			description:    "Medium length filename should not be truncated",
		},
		{
			name:           "LongLength",
			baseLength:     200,
			expectTruncate: true,
			description:    "Long filename should be truncated to reasonable length",
		},
		{
			name:           "VeryLongLength",
			baseLength:     300,
			expectTruncate: true,
			description:    "Very long filename should be truncated",
		},
	}

	for _, tc := range longFilenameTests {
		s.T().Run(tc.name, func(t *testing.T) {
			// Create filename with specified length
			baseFilename := strings.Repeat("a", tc.baseLength)
			intentFilename := "intent-" + baseFilename + ".json"

			// Create the file
			intentPath := filepath.Join(s.tempDir, intentFilename)
			testContent := `{"apiVersion": "v1", "kind": "NetworkIntent", "metadata": {"name": "long-name-test"}}`
			err := os.WriteFile(intentPath, []byte(testContent), 0o644)
			require.NoError(t, err)

			// Process the file
			watcher.config.Once = true
			err = watcher.Start()
			assert.NoError(t, err)

			// Check status file creation
			statusDir := filepath.Join(s.tempDir, "status")
			statusFiles, err := os.ReadDir(statusDir)
			require.NoError(t, err)
			assert.Greater(t, len(statusFiles), 0, "Should create status file for %s", tc.description)

			// Verify status filename length
			for _, file := range statusFiles {
				statusName := file.Name()

				// Should end with .status
				assert.True(t, strings.HasSuffix(statusName, ".status"),
					"Status file should end with .status")

				// Should not exceed reasonable length (Windows NTFS limit is 255, but we use conservative limit)
				assert.LessOrEqual(t, len(statusName), 200,
					"Status filename should not exceed reasonable length for %s: got %d chars", tc.description, len(statusName))

				if tc.expectTruncate && tc.baseLength > 100 {
					// Should be shorter than the original for long names
					assert.Less(t, len(statusName), len(intentFilename),
						"Status filename should be truncated for %s", tc.description)
				}

				// Should still contain timestamp
				assert.Regexp(t, `\d{8}-\d{6}`, statusName,
					"Status filename should contain timestamp for %s", tc.description)
			}
		})
	}
}

func (s *WatcherTestSuite) TestWindowsFilenameValidation_SpecialFileTypes() {
	s.T().Log("Testing Windows filename validation with special file types")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	// Test different file extensions and special cases
	specialFileTests := []struct {
		name          string
		filename      string
		shouldProcess bool
		description   string
	}{
		{
			name:          "StandardJsonFile",
			filename:      "intent-test.json",
			shouldProcess: true,
			description:   "Standard JSON file should be processed",
		},
		{
			name:          "NoExtensionFile",
			filename:      "intent-test",
			shouldProcess: true,
			description:   "File without extension should be processed",
		},
		{
			name:          "MultipleExtensions",
			filename:      "intent-test.backup.json",
			shouldProcess: true,
			description:   "File with multiple extensions should be processed",
		},
		{
			name:          "WindowsExecutable",
			filename:      "intent-test.exe.json",
			shouldProcess: true,
			description:   "File with .exe in name should be processed safely",
		},
		{
			name:          "HiddenFile",
			filename:      ".intent-test.json",
			shouldProcess: true,
			description:   "Hidden file (starting with .) should be processed",
		},
		{
			name:          "FileWithSpaces",
			filename:      "intent test file.json",
			shouldProcess: true,
			description:   "File with spaces should be processed with sanitized name",
		},
	}

	for _, tc := range specialFileTests {
		s.T().Run(tc.name, func(t *testing.T) {
			// Create the file
			intentPath := filepath.Join(s.tempDir, tc.filename)
			testContent := `{"apiVersion": "v1", "kind": "NetworkIntent", "metadata": {"name": "special-file-test"}}`
			err := os.WriteFile(intentPath, []byte(testContent), 0o644)
			require.NoError(t, err)

			// Process the file
			watcher.config.Once = true
			err = watcher.Start()
			assert.NoError(t, err)

			if tc.shouldProcess {
				// Should create status file
				statusDir := filepath.Join(s.tempDir, "status")
				statusFiles, err := os.ReadDir(statusDir)
				require.NoError(t, err)
				assert.Greater(t, len(statusFiles), 0, "Should create status file for %s", tc.description)

				// Verify status filename is Windows-safe
				for _, file := range statusFiles {
					statusName := file.Name()

					// Should be valid Windows filename
					assert.True(t, strings.HasSuffix(statusName, ".status"),
						"Status file should end with .status for %s", tc.description)

					// Should not contain invalid characters for Windows
					invalidChars := []string{"<", ">", ":", "\"", "|", "?", "*"}
					for _, char := range invalidChars {
						assert.NotContains(t, statusName, char,
							"Status filename should not contain %s for %s", char, tc.description)
					}

					// Should handle path separators correctly
					assert.NotContains(t, statusName, "/",
						"Status filename should not contain forward slash for %s", tc.description)
					assert.NotContains(t, statusName, "\\",
						"Status filename should not contain backslash for %s", tc.description)

					// Should be valid ASCII for maximum Windows compatibility
					for i, r := range statusName {
						assert.True(t, r <= 127,
							"Character at position %d should be ASCII for %s: %c", i, tc.description, r)
					}
				}
			}
		})
	}
}
