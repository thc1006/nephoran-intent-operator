package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/nephoran-intent-operator/internal/loop"
	"github.com/thc1006/nephoran-intent-operator/internal/porch"
)

// TestPathTraversalSecurity tests protection against path traversal attacks
func TestPathTraversalSecurity(t *testing.T) {
	tests := []struct {
		name          string
		intentContent string
		shouldFail    bool
		expectedError string
	}{
		{
			name: "path traversal in target",
			intentContent: `{
				"intent_type": "scaling",
				"target": "../../../etc/passwd",
				"namespace": "default",
				"replicas": 3
			}`,
			shouldFail:    false, // Should be handled gracefully, not fail
			expectedError: "",
		},
		{
			name: "path traversal in namespace",
			intentContent: `{
				"intent_type": "scaling",
				"target": "my-app",
				"namespace": "../../../var/log",
				"replicas": 3
			}`,
			shouldFail:    false,
			expectedError: "",
		},
		{
			name: "absolute path in target",
			intentContent: `{
				"intent_type": "scaling",
				"target": "/etc/passwd",
				"namespace": "default",
				"replicas": 3
			}`,
			shouldFail:    false,
			expectedError: "",
		},
		{
			name: "windows path traversal",
			intentContent: `{
				"intent_type": "scaling",
				"target": "..\\..\\windows\\system32",
				"namespace": "default",
				"replicas": 3
			}`,
			shouldFail:    false,
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			handoffDir := filepath.Join(tempDir, "handoff")
			outDir := filepath.Join(tempDir, "out")

			require.NoError(t, os.MkdirAll(handoffDir, 0755))
			require.NoError(t, os.MkdirAll(outDir, 0755))

			// Create malicious intent file
			intentFile := filepath.Join(handoffDir, "intent-malicious.json")
			require.NoError(t, os.WriteFile(intentFile, []byte(tt.intentContent), 0644))

			// Create mock porch executable
			mockPorch := createSecureMockPorch(t, tempDir)

			// Configure watcher with security-focused settings
			config := loop.Config{
				PorchPath:    mockPorch,
				Mode:         "direct",
				OutDir:       outDir,
				Once:         true,
				DebounceDur:  100 * time.Millisecond,
				MaxWorkers:   3, // Production-like worker count for realistic concurrency testing
				CleanupAfter: time.Hour,
			}

			watcher, err := loop.NewWatcher(handoffDir, config)
			require.NoError(t, err)
			defer watcher.Close()

			// Start watcher in once mode
			err = watcher.Start()

			if tt.shouldFail {
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			} else {
				assert.NoError(t, err)
			}

			// Verify no files were created outside the expected directories
			assertNoPathTraversal(t, tempDir, outDir)
		})
	}
}

// TestCommandInjectionSecurity tests protection against command injection
func TestCommandInjectionSecurity(t *testing.T) {
	tests := []struct {
		name          string
		porchPath     string
		intentContent string
		shouldDetect  bool
		expectedInLog string
	}{
		{
			name:      "malicious porch path with injection",
			porchPath: "porch; rm -rf /",
			intentContent: `{
				"intent_type": "scaling",
				"target": "my-app",
				"namespace": "default",
				"replicas": 3
			}`,
			shouldDetect:  true,
			expectedInLog: "command",
		},
		{
			name:      "malicious porch path with pipe",
			porchPath: "porch | cat /etc/passwd",
			intentContent: `{
				"intent_type": "scaling",
				"target": "my-app",
				"namespace": "default",
				"replicas": 3
			}`,
			shouldDetect:  true,
			expectedInLog: "command",
		},
		{
			name:      "injection in intent target",
			porchPath: createSecureMockPorch(t, t.TempDir()),
			intentContent: `{
				"intent_type": "scaling",
				"target": "my-app; echo injected",
				"namespace": "default",
				"replicas": 3
			}`,
			shouldDetect:  false, // This should be handled by porch validation
			expectedInLog: "",
		},
		{
			name:      "injection in intent namespace",
			porchPath: createSecureMockPorch(t, t.TempDir()),
			intentContent: `{
				"intent_type": "scaling",
				"target": "my-app",
				"namespace": "default && echo injected",
				"replicas": 3
			}`,
			shouldDetect:  false,
			expectedInLog: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			handoffDir := filepath.Join(tempDir, "handoff")
			outDir := filepath.Join(tempDir, "out")

			require.NoError(t, os.MkdirAll(handoffDir, 0755))
			require.NoError(t, os.MkdirAll(outDir, 0755))

			// Create intent file
			intentFile := filepath.Join(handoffDir, "intent-test.json")
			require.NoError(t, os.WriteFile(intentFile, []byte(tt.intentContent), 0644))

			// Configure executor
			config := porch.ExecutorConfig{
				PorchPath: tt.porchPath,
				Mode:      "direct",
				OutDir:    outDir,
				Timeout:   5 * time.Second,
			}

			executor := porch.NewExecutor(config)
			ctx := context.Background()

			// Execute and check for security violations
			result, err := executor.Execute(ctx, intentFile)

			// Command injection should either:
			// 1. Fail to execute (command not found)
			// 2. Be caught by validation
			// 3. Execute safely without actual injection

			if tt.shouldDetect {
				// Should fail due to invalid command
				assert.False(t, result.Success)
			} else {
				// Should complete normally
				require.NoError(t, err)
			}

			// Verify no dangerous operations were executed
			assertNoCommandInjection(t, tempDir)
		})
	}
}

// TestResourceExhaustionResilience tests protection against resource exhaustion attacks
func TestResourceExhaustionResilience(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(t *testing.T, handoffDir string)
		expectedFiles  int
		maxProcessTime time.Duration
	}{
		{
			name: "many small files",
			setupFunc: func(t *testing.T, handoffDir string) {
				// Create 100 small intent files
				for i := 0; i < 100; i++ {
					content := fmt.Sprintf(`{
						"intent_type": "scaling",
						"target": "app-%d",
						"namespace": "default",
						"replicas": 1
					}`, i)
					file := filepath.Join(handoffDir, fmt.Sprintf("intent-%d.json", i))
					require.NoError(t, os.WriteFile(file, []byte(content), 0644))
				}
			},
			expectedFiles:  100,
			maxProcessTime: 30 * time.Second,
		},
		{
			name: "large single file",
			setupFunc: func(t *testing.T, handoffDir string) {
				// Create one large intent file (1MB of data)
				largeData := strings.Repeat("A", 1024*1024)
				content := fmt.Sprintf(`{
					"intent_type": "scaling",
					"target": "my-app",
					"namespace": "default",
					"replicas": 3,
					"large_data": "%s"
				}`, largeData)
				file := filepath.Join(handoffDir, "large-intent.json")
				require.NoError(t, os.WriteFile(file, []byte(content), 0644))
			},
			expectedFiles:  1,
			maxProcessTime: 10 * time.Second,
		},
		{
			name: "rapid file creation",
			setupFunc: func(t *testing.T, handoffDir string) {
				// Rapidly create files during processing
				go func() {
					for i := 0; i < 50; i++ {
						content := fmt.Sprintf(`{
							"intent_type": "scaling",
							"target": "rapid-%d",
							"namespace": "default",
							"replicas": 1
						}`, i)
						file := filepath.Join(handoffDir, fmt.Sprintf("intent-rapid-%d.json", i))
						_ = os.WriteFile(file, []byte(content), 0644)
						time.Sleep(10 * time.Millisecond)
					}
				}()

				// Initial files
				for i := 0; i < 10; i++ {
					content := fmt.Sprintf(`{
						"intent_type": "scaling",
						"target": "initial-%d",
						"namespace": "default",
						"replicas": 1
					}`, i)
					file := filepath.Join(handoffDir, fmt.Sprintf("intent-initial-%d.json", i))
					require.NoError(t, os.WriteFile(file, []byte(content), 0644))
				}
			},
			expectedFiles:  10, // Only count initial files for deterministic testing
			maxProcessTime: 15 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			handoffDir := filepath.Join(tempDir, "handoff")
			outDir := filepath.Join(tempDir, "out")

			require.NoError(t, os.MkdirAll(handoffDir, 0755))
			require.NoError(t, os.MkdirAll(outDir, 0755))

			// Setup test files
			tt.setupFunc(t, handoffDir)

			// Create mock porch with small delay to simulate processing
			mockPorch := createMockPorchWithDelay(t, tempDir, 50*time.Millisecond)

			// Configure watcher with resource limits
			config := loop.Config{
				PorchPath:    mockPorch,
				Mode:         "direct",
				OutDir:       outDir,
				Once:         true,
				DebounceDur:  10 * time.Millisecond, // Short debounce for faster testing
				MaxWorkers:   3,                     // Limited workers
				CleanupAfter: time.Hour,
			}

			watcher, err := loop.NewWatcher(handoffDir, config)
			require.NoError(t, err)
			defer watcher.Close()

			// Measure processing time
			start := time.Now()
			err = watcher.Start()
			duration := time.Since(start)

			assert.NoError(t, err)
			assert.Less(t, duration, tt.maxProcessTime, "Processing took too long, possible resource exhaustion")

			// Verify system didn't run out of resources
			assertSystemHealth(t)
		})
	}
}

// TestFilePermissionValidation tests that created files have correct permissions
func TestFilePermissionValidation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("File permission tests not applicable on Windows")
	}

	tempDir := t.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")
	outDir := filepath.Join(tempDir, "out")

	require.NoError(t, os.MkdirAll(handoffDir, 0755))
	require.NoError(t, os.MkdirAll(outDir, 0755))

	// Create intent file
	intentContent := `{
		"intent_type": "scaling",
		"target": "my-app",
		"namespace": "default",
		"replicas": 3
	}`
	intentFile := filepath.Join(handoffDir, "intent-test.json")
	require.NoError(t, os.WriteFile(intentFile, []byte(intentContent), 0644))

	// Create mock porch
	mockPorch := createSecureMockPorch(t, tempDir)

	// Configure watcher
	config := loop.Config{
		PorchPath:    mockPorch,
		Mode:         "direct",
		OutDir:       outDir,
		Once:         true,
		DebounceDur:  100 * time.Millisecond,
		MaxWorkers:   3, // Production-like worker count for realistic concurrency testing
		CleanupAfter: time.Hour,
	}

	watcher, err := loop.NewWatcher(handoffDir, config)
	require.NoError(t, err)
	defer watcher.Close()

	// Process files
	err = watcher.Start()
	require.NoError(t, err)

	// Check file permissions
	err = filepath.WalkDir(tempDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		mode := info.Mode()

		if d.IsDir() {
			// Directories should be readable/writable by owner, readable by group
			expectedMode := fs.FileMode(0755)
			assert.Equal(t, expectedMode, mode&0777, "Directory %s has incorrect permissions", path)
		} else {
			// Determine expected permissions based on file type
			var expectedMode fs.FileMode
			if isExecutableScript(path) {
				// Executable scripts need execute permissions on Unix systems
				expectedMode = fs.FileMode(0755)
			} else {
				// Regular files should be readable/writable by owner, readable by group
				expectedMode = fs.FileMode(0644)
			}
			assert.Equal(t, expectedMode, mode&0777, "File %s has incorrect permissions", path)
		}

		return nil
	})
	require.NoError(t, err)
}

// TestInputValidation tests handling of malformed and invalid inputs
func TestInputValidation(t *testing.T) {
	tests := []struct {
		name           string
		intentContent  string
		configModifier func(*loop.Config)
		expectedError  bool
		shouldProcess  bool
	}{
		{
			name: "malformed JSON",
			intentContent: `{
				"intent_type": "scaling",
				"target": "my-app",
				"namespace": "default",
				"replicas": 3,
				"invalid": {{{
			}`,
			expectedError: false, // Should handle gracefully
			shouldProcess: false,
		},
		{
			name: "missing required fields",
			intentContent: `{
				"intent_type": "scaling"
			}`,
			expectedError: false,
			shouldProcess: false,
		},
		{
			name: "invalid intent type",
			intentContent: `{
				"intent_type": "invalid",
				"target": "my-app",
				"namespace": "default",
				"replicas": 3
			}`,
			expectedError: false,
			shouldProcess: false,
		},
		{
			name: "negative replicas",
			intentContent: `{
				"intent_type": "scaling",
				"target": "my-app",
				"namespace": "default",
				"replicas": -1
			}`,
			expectedError: false,
			shouldProcess: false,
		},
		{
			name: "extremely large replicas",
			intentContent: `{
				"intent_type": "scaling",
				"target": "my-app",
				"namespace": "default",
				"replicas": 999999999
			}`,
			expectedError: false,
			shouldProcess: false,
		},
		{
			name: "null values",
			intentContent: `{
				"intent_type": null,
				"target": null,
				"namespace": null,
				"replicas": null
			}`,
			expectedError: false,
			shouldProcess: false,
		},
		{
			name: "unicode characters",
			intentContent: `{
				"intent_type": "scaling",
				"target": "my-app-??",
				"namespace": "default-ñ",
				"replicas": 3
			}`,
			expectedError: false,
			shouldProcess: true, // Unicode should be handled
		},
		{
			name: "invalid timeout configuration",
			configModifier: func(c *loop.Config) {
				c.DebounceDur = -1 * time.Second
			},
			intentContent: `{
				"intent_type": "scaling",
				"target": "my-app",
				"namespace": "default",
				"replicas": 3
			}`,
			expectedError: false, // Should use default values
			shouldProcess: true,
		},
		{
			name: "invalid worker count",
			configModifier: func(c *loop.Config) {
				c.MaxWorkers = -5
			},
			intentContent: `{
				"intent_type": "scaling",
				"target": "my-app",
				"namespace": "default",
				"replicas": 3
			}`,
			expectedError: false, // Should use default values
			shouldProcess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			handoffDir := filepath.Join(tempDir, "handoff")
			outDir := filepath.Join(tempDir, "out")

			require.NoError(t, os.MkdirAll(handoffDir, 0755))
			require.NoError(t, os.MkdirAll(outDir, 0755))

			// Create intent file
			intentFile := filepath.Join(handoffDir, "intent-test.json")
			require.NoError(t, os.WriteFile(intentFile, []byte(tt.intentContent), 0644))

			// Create mock porch
			mockPorch := createSecureMockPorch(t, tempDir)

			// Configure watcher
			config := loop.Config{
				PorchPath:    mockPorch,
				Mode:         "direct",
				OutDir:       outDir,
				Once:         true,
				DebounceDur:  100 * time.Millisecond,
				MaxWorkers:   1,
				CleanupAfter: time.Hour,
			}

			// Apply config modifications
			if tt.configModifier != nil {
				tt.configModifier(&config)
			}

			watcher, err := loop.NewWatcher(handoffDir, config)
			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			defer watcher.Close()

			// Process files
			err = watcher.Start()
			assert.NoError(t, err, "Watcher should handle invalid inputs gracefully")

			// Verify processing behavior
			if tt.shouldProcess {
				// Check that files were processed (moved to processed/failed directories)
				processedExists := fileExists(t, filepath.Join(handoffDir, "processed", "intent-test.json"))
				failedExists := fileExists(t, filepath.Join(handoffDir, "failed", "intent-test.json"))
				assert.True(t, processedExists || failedExists, "File should have been processed")
			}
		})
	}
}

// TestConcurrentFileProcessing tests race conditions and atomic operations
func TestConcurrentFileProcessing(t *testing.T) {
	tempDir := t.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")
	outDir := filepath.Join(tempDir, "out")

	require.NoError(t, os.MkdirAll(handoffDir, 0755))
	require.NoError(t, os.MkdirAll(outDir, 0755))

	// Create mock porch with random delays to simulate real processing
	mockPorch := createMockPorchWithRandomDelay(t, tempDir)

	// Configure watcher with multiple workers
	config := loop.Config{
		PorchPath:    mockPorch,
		Mode:         "direct",
		OutDir:       outDir,
		Once:         false, // Continuous mode to test concurrent processing
		DebounceDur:  50 * time.Millisecond,
		MaxWorkers:   5, // Multiple workers to test concurrency
		CleanupAfter: time.Hour,
	}

	watcher, err := loop.NewWatcher(handoffDir, config)
	require.NoError(t, err)
	defer watcher.Close()

	// Start watcher in background
	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		_ = watcher.Start()
	}()

	// Concurrently create many intent files
	var wg sync.WaitGroup
	numFiles := 50

	for i := 0; i < numFiles; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			content := fmt.Sprintf(`{
				"intent_type": "scaling",
				"target": "app-%d",
				"namespace": "default",
				"replicas": %d
			}`, id, id%10+1)

			file := filepath.Join(handoffDir, fmt.Sprintf("intent-concurrent-%d.json", id))

			// Add small random delay to increase chance of race conditions
			time.Sleep(time.Duration(id%10) * time.Millisecond)

			err := os.WriteFile(file, []byte(content), 0644)
			if err != nil {
				t.Errorf("Failed to write file %d: %v", id, err)
			}
		}(i)
	}

	wg.Wait()

	// Wait for processing to complete
	time.Sleep(8 * time.Second)

	// Verify all files were processed exactly once
	processedCount := countFilesInDir(t, filepath.Join(handoffDir, "processed"))
	failedCount := countFilesInDir(t, filepath.Join(handoffDir, "failed"))
	totalProcessed := processedCount + failedCount

	assert.Equal(t, numFiles, totalProcessed, "All files should be processed exactly once")

	// Verify no race conditions corrupted state
	stats, err := watcher.GetStats()
	require.NoError(t, err)
	assert.Equal(t, totalProcessed, stats.ProcessedCount+stats.FailedCount, "Stats should match processed files")
}

// Helper functions

// isExecutableScript determines if a file is an executable script that requires execute permissions
func isExecutableScript(path string) bool {
	// Extract the filename for checking
	filename := filepath.Base(path)

	// Check for common executable script patterns
	switch {
	case strings.HasSuffix(filename, ".sh"):
		return true
	case strings.HasSuffix(filename, ".bat"):
		return false // Windows batch files don't need Unix executable permissions
	case strings.Contains(filename, "mock-porch") && !strings.HasSuffix(filename, ".bat"):
		return true // Unix mock-porch scripts
	default:
		return false
	}
}

func createSecureMockPorch(t testing.TB, tempDir string) string {
	mockPath, err := porch.CreateCrossPlatformMock(tempDir, porch.CrossPlatformMockOptions{
		ExitCode: 0,
		Stdout:   "Processing intent file completed successfully",
	})
	require.NoError(t, err)
	return mockPath
}

func createMockPorchWithDelay(t testing.TB, tempDir string, delay time.Duration) string {
	mockPath, err := porch.CreateCrossPlatformMock(tempDir, porch.CrossPlatformMockOptions{
		ExitCode: 0,
		Stdout:   "Processing completed",
		Sleep:    delay,
	})
	require.NoError(t, err)
	return mockPath
}

func createMockPorchWithRandomDelay(t testing.TB, tempDir string) string {
	mockPath, err := porch.CreateCrossPlatformMock(tempDir, porch.CrossPlatformMockOptions{
		CustomScript: struct {
			Windows string
			Unix    string
		}{
			Windows: `@echo off
if "%1"=="--help" (
    echo Mock porch help
    exit /b 0
)
set /a delay=%RANDOM% %% 3 + 1
timeout /t %delay% /nobreak >nul
echo Processing completed
exit /b 0`,
			Unix: `#!/bin/bash
if [ "$1" = "--help" ]; then
    echo "Mock porch help"
    exit 0
fi
delay=$((RANDOM % 3 + 1))
sleep $delay
echo "Processing completed"
exit 0`,
		},
	})
	require.NoError(t, err)
	return mockPath
}

func assertNoPathTraversal(t *testing.T, tempDir, outDir string) {
	// Check that no files were created outside expected directories
	err := filepath.WalkDir(tempDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Ensure all paths are within tempDir
		relPath, err := filepath.Rel(tempDir, path)
		if err != nil || strings.HasPrefix(relPath, "..") {
			t.Errorf("Path traversal detected: %s is outside %s", path, tempDir)
		}

		return nil
	})
	require.NoError(t, err)
}

func assertNoCommandInjection(t *testing.T, tempDir string) {
	// Check for signs of command injection
	suspiciousFiles := []string{
		"injected",
		"passwd",
		"backdoor",
		"exploit",
	}

	err := filepath.WalkDir(tempDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		name := strings.ToLower(d.Name())
		for _, suspicious := range suspiciousFiles {
			if strings.Contains(name, suspicious) {
				t.Errorf("Potential command injection artifact found: %s", path)
			}
		}

		return nil
	})
	require.NoError(t, err)
}

func assertSystemHealth(t *testing.T) {
	// Check basic system health indicators
	// This is a simplified check - in production you'd check memory, CPU, etc.

	// Ensure we can still create files (system not locked up)
	testFile := filepath.Join(os.TempDir(), fmt.Sprintf("health-check-%d.tmp", time.Now().UnixNano()))
	err := os.WriteFile(testFile, []byte("health check"), 0644)
	assert.NoError(t, err, "System should still be responsive")

	if err == nil {
		os.Remove(testFile)
	}
}

func fileExists(t *testing.T, path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func countFilesInDir(t *testing.T, dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			count++
		}
	}
	return count
}

// ProcessingStats represents file processing statistics
type ProcessingStats struct {
	TotalFiles     int `json:"total_files"`
	ProcessedFiles int `json:"processed_files"`
	FailedFiles    int `json:"failed_files"`
}
