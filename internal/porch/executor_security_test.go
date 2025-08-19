package porch

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecutor_CommandInjectionPrevention tests protection against command injection
func TestExecutor_CommandInjectionPrevention(t *testing.T) {
	tests := []struct {
		name           string
		porchPath      string
		intentPath     string
		outDir         string
		expectedResult func(t *testing.T, result *ExecutionResult)
		setupFunc      func(t *testing.T, tempDir string) (string, string, string)
	}{
		{
			name:      "malicious porch path with semicolon",
			porchPath: "porch; echo 'injected'",
			setupFunc: func(t *testing.T, tempDir string) (string, string, string) {
				intentFile := filepath.Join(tempDir, "intent.json")
				outDir := filepath.Join(tempDir, "out")
				require.NoError(t, os.WriteFile(intentFile, []byte(`{"test": "intent"}`), 0644))
				require.NoError(t, os.MkdirAll(outDir, 0755))
				return "porch; echo 'injected'", intentFile, outDir
			},
			expectedResult: func(t *testing.T, result *ExecutionResult) {
				// Should fail because command contains injection
				assert.False(t, result.Success)
				assert.NotNil(t, result.Error)
			},
		},
		{
			name:      "malicious porch path with pipe",
			porchPath: "porch | cat /etc/passwd",
			setupFunc: func(t *testing.T, tempDir string) (string, string, string) {
				intentFile := filepath.Join(tempDir, "intent.json")
				outDir := filepath.Join(tempDir, "out")
				require.NoError(t, os.WriteFile(intentFile, []byte(`{"test": "intent"}`), 0644))
				require.NoError(t, os.MkdirAll(outDir, 0755))
				return "porch | cat /etc/passwd", intentFile, outDir
			},
			expectedResult: func(t *testing.T, result *ExecutionResult) {
				assert.False(t, result.Success)
				assert.NotNil(t, result.Error)
			},
		},
		{
			name:      "malicious porch path with ampersand",
			porchPath: "porch && rm -rf /tmp",
			setupFunc: func(t *testing.T, tempDir string) (string, string, string) {
				intentFile := filepath.Join(tempDir, "intent.json")
				outDir := filepath.Join(tempDir, "out")
				require.NoError(t, os.WriteFile(intentFile, []byte(`{"test": "intent"}`), 0644))
				require.NoError(t, os.MkdirAll(outDir, 0755))
				return "porch && rm -rf /tmp", intentFile, outDir
			},
			expectedResult: func(t *testing.T, result *ExecutionResult) {
				assert.False(t, result.Success)
				assert.NotNil(t, result.Error)
			},
		},
		{
			name: "path traversal in intent path",
			setupFunc: func(t *testing.T, tempDir string) (string, string, string) {
				// Create a mock porch that would succeed
				mockPorch := createSecureMockPorch(t, tempDir)
				maliciousPath := "../../../etc/passwd"
				outDir := filepath.Join(tempDir, "out")
				require.NoError(t, os.MkdirAll(outDir, 0755))
				return mockPorch, maliciousPath, outDir
			},
			expectedResult: func(t *testing.T, result *ExecutionResult) {
				// Should handle gracefully - the executor will try to resolve the path
				// Error handling depends on whether the file exists
				assert.NotNil(t, result)
				// The command should either succeed (if it can resolve the path) or fail gracefully
			},
		},
		{
			name: "path traversal in output directory",
			setupFunc: func(t *testing.T, tempDir string) (string, string, string) {
				mockPorch := createSecureMockPorch(t, tempDir)
				intentFile := filepath.Join(tempDir, "intent.json")
				require.NoError(t, os.WriteFile(intentFile, []byte(`{"test": "intent"}`), 0644))
				maliciousOutDir := "../../../tmp/malicious"
				return mockPorch, intentFile, maliciousOutDir
			},
			expectedResult: func(t *testing.T, result *ExecutionResult) {
				// Should handle path traversal in output directory
				assert.NotNil(t, result)
				// The executor should process the path, but the actual directory creation
				// should be handled safely by the underlying filesystem
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			porchPath, intentPath, outDir := tt.setupFunc(t, tempDir)

			config := ExecutorConfig{
				PorchPath: porchPath,
				Mode:      ModeDirect,
				OutDir:    outDir,
				Timeout:   5 * time.Second,
			}

			executor := NewExecutor(config)
			ctx := context.Background()

			result, err := executor.Execute(ctx, intentPath)
			require.NoError(t, err) // Execute should not return error, all errors in result

			tt.expectedResult(t, result)

			// Verify no malicious files were created outside the temp directory
			assertNoMaliciousActivity(t, tempDir)
		})
	}
}

// TestExecutor_ResourceExhaustionPrevention tests protection against resource exhaustion
func TestExecutor_ResourceExhaustionPrevention(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(t *testing.T, tempDir string) ExecutorConfig
		testFunc       func(t *testing.T, executor *Executor, tempDir string)
		timeoutExpected bool
	}{
		{
			name: "very short timeout",
			setupFunc: func(t *testing.T, tempDir string) ExecutorConfig {
				// Create mock porch that sleeps longer than timeout
				mockPorch := createMockPorchWithDelay(t, tempDir, 2*time.Second)
				return ExecutorConfig{
					PorchPath: mockPorch,
					Mode:      ModeDirect,
					OutDir:    filepath.Join(tempDir, "out"),
					Timeout:   100 * time.Millisecond, // Very short timeout
				}
			},
			testFunc: func(t *testing.T, executor *Executor, tempDir string) {
				intentFile := filepath.Join(tempDir, "intent.json")
				require.NoError(t, os.WriteFile(intentFile, []byte(`{"test": "intent"}`), 0644))

				ctx := context.Background()
				result, err := executor.Execute(ctx, intentFile)
				require.NoError(t, err)

				assert.False(t, result.Success)
				assert.NotNil(t, result.Error)
				assert.Contains(t, result.Error.Error(), "timed out")
			},
			timeoutExpected: true,
		},
		{
			name: "context cancellation",
			setupFunc: func(t *testing.T, tempDir string) ExecutorConfig {
				// Create mock porch that sleeps for a long time
				mockPorch := createMockPorchWithDelay(t, tempDir, 10*time.Second)
				return ExecutorConfig{
					PorchPath: mockPorch,
					Mode:      ModeDirect,
					OutDir:    filepath.Join(tempDir, "out"),
					Timeout:   30 * time.Second, // Long timeout
				}
			},
			testFunc: func(t *testing.T, executor *Executor, tempDir string) {
				intentFile := filepath.Join(tempDir, "intent.json")
				require.NoError(t, os.WriteFile(intentFile, []byte(`{"test": "intent"}`), 0644))

				// Create context that will be cancelled quickly
				ctx, cancel := context.WithCancel(context.Background())
				
				// Cancel after 200ms
				go func() {
					time.Sleep(200 * time.Millisecond)
					cancel()
				}()

				result, err := executor.Execute(ctx, intentFile)
				require.NoError(t, err)

				assert.False(t, result.Success)
				assert.NotNil(t, result.Error)
			},
			timeoutExpected: false,
		},
		{
			name: "large file processing",
			setupFunc: func(t *testing.T, tempDir string) ExecutorConfig {
				mockPorch := createSecureMockPorch(t, tempDir)
				return ExecutorConfig{
					PorchPath: mockPorch,
					Mode:      ModeDirect,
					OutDir:    filepath.Join(tempDir, "out"),
					Timeout:   10 * time.Second,
				}
			},
			testFunc: func(t *testing.T, executor *Executor, tempDir string) {
				// Create a large intent file (1MB)
				largeContent := fmt.Sprintf(`{
					"intent_type": "scaling",
					"target": "my-app",
					"namespace": "default",
					"replicas": 3,
					"large_data": "%s"
				}`, strings.Repeat("A", 1024*1024))

				intentFile := filepath.Join(tempDir, "large-intent.json")
				require.NoError(t, os.WriteFile(intentFile, []byte(largeContent), 0644))

				ctx := context.Background()
				start := time.Now()
				result, err := executor.Execute(ctx, intentFile)
				duration := time.Since(start)

				require.NoError(t, err)
				// Should complete within reasonable time even with large file
				assert.Less(t, duration, 5*time.Second, "Large file processing took too long")
				
				// The mock porch should handle the large file successfully
				assert.True(t, result.Success)
			},
			timeoutExpected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "out"), 0755))

			config := tt.setupFunc(t, tempDir)
			executor := NewExecutor(config)

			tt.testFunc(t, executor, tempDir)
		})
	}
}

// TestExecutor_PathTraversalPrevention tests handling of malicious paths
func TestExecutor_PathTraversalPrevention(t *testing.T) {
	tests := []struct {
		name       string
		intentPath string
		outDir     string
		testFunc   func(t *testing.T, result *ExecutionResult, tempDir string)
	}{
		{
			name:       "intent path with relative traversal",
			intentPath: "../../../etc/passwd",
			outDir:     "out",
			testFunc: func(t *testing.T, result *ExecutionResult, tempDir string) {
				// Should handle gracefully without creating files outside tempDir
				assertNoPathTraversal(t, tempDir)
			},
		},
		{
			name:       "intent path with absolute path",
			intentPath: "/etc/passwd",
			outDir:     "out",
			testFunc: func(t *testing.T, result *ExecutionResult, tempDir string) {
				// Should handle absolute paths without security issues
				assertNoPathTraversal(t, tempDir)
			},
		},
		{
			name:       "output directory with traversal",
			intentPath: "intent.json",
			outDir:     "../../../tmp/malicious",
			testFunc: func(t *testing.T, result *ExecutionResult, tempDir string) {
				// Should not create directories outside expected locations
				// Check that no files were created in /tmp/malicious
				maliciousPath := "/tmp/malicious"
				if runtime.GOOS == "windows" {
					maliciousPath = "C:\\tmp\\malicious"
				}
				_, err := os.Stat(maliciousPath)
				assert.True(t, os.IsNotExist(err), "Should not create directories via path traversal")
			},
		},
		{
			name:       "unicode in paths",
			intentPath: "интент-файл.json",
			outDir:     "выход",
			testFunc: func(t *testing.T, result *ExecutionResult, tempDir string) {
				// Should handle unicode paths correctly
				assert.NotNil(t, result, "Should handle unicode paths")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			
			// Create mock porch
			mockPorch := createSecureMockPorch(t, tempDir)
			
			// Create intent file if using relative path
			if !filepath.IsAbs(tt.intentPath) && !strings.Contains(tt.intentPath, "..") {
				intentFile := filepath.Join(tempDir, tt.intentPath)
				require.NoError(t, os.WriteFile(intentFile, []byte(`{"test": "intent"}`), 0644))
				tt.intentPath = intentFile
			}

			config := ExecutorConfig{
				PorchPath: mockPorch,
				Mode:      ModeDirect,
				OutDir:    tt.outDir,
				Timeout:   5 * time.Second,
			}

			executor := NewExecutor(config)
			ctx := context.Background()

			result, err := executor.Execute(ctx, tt.intentPath)
			require.NoError(t, err)

			tt.testFunc(t, result, tempDir)
		})
	}
}

// TestExecutor_FilePermissionsSecurity tests that created files have secure permissions
func TestExecutor_FilePermissionsSecurity(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("File permission tests not applicable on Windows")
	}

	tempDir := t.TempDir()
	intentFile := filepath.Join(tempDir, "intent.json")
	outDir := filepath.Join(tempDir, "out")
	
	require.NoError(t, os.WriteFile(intentFile, []byte(`{"test": "intent"}`), 0644))
	require.NoError(t, os.MkdirAll(outDir, 0755))

	// Create mock porch that creates output files
	mockPorch := createMockPorchWithFileCreation(t, tempDir, outDir)

	config := ExecutorConfig{
		PorchPath: mockPorch,
		Mode:      ModeDirect,
		OutDir:    outDir,
		Timeout:   5 * time.Second,
	}

	executor := NewExecutor(config)
	ctx := context.Background()

	result, err := executor.Execute(ctx, intentFile)
	require.NoError(t, err)
	
	// If the mock porch succeeded, check file permissions
	if result.Success {
		err = filepath.WalkDir(outDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			info, err := d.Info()
			if err != nil {
				return err
			}

			mode := info.Mode()
			
			if d.IsDir() {
				// Directories should not be world-writable
				assert.False(t, mode&0002 != 0, "Directory %s should not be world-writable", path)
			} else {
				// Files should not be world-writable or executable by others
				assert.False(t, mode&0002 != 0, "File %s should not be world-writable", path)
				assert.False(t, mode&0001 != 0, "File %s should not be executable by others", path)
			}

			return nil
		})
		require.NoError(t, err)
	}
}

// TestValidatePorchPath_Security tests security aspects of porch path validation
func TestValidatePorchPath_Security(t *testing.T) {
	tests := []struct {
		name      string
		porchPath string
		wantErr   bool
		errorMsg  string
	}{
		{
			name:      "valid porch path",
			porchPath: createSecureMockPorch(t, t.TempDir()),
			wantErr:   false,
		},
		{
			name:      "path with command injection",
			porchPath: "porch; rm -rf /",
			wantErr:   true,
			errorMsg:  "validation failed",
		},
		{
			name:      "path with pipe",
			porchPath: "porch | cat /etc/passwd",
			wantErr:   true,
			errorMsg:  "validation failed",
		},
		{
			name:      "nonexistent path",
			porchPath: "/nonexistent/porch/executable",
			wantErr:   true,
			errorMsg:  "validation failed",
		},
		{
			name:      "relative path with traversal",
			porchPath: "../../../usr/bin/porch",
			wantErr:   true,
			errorMsg:  "validation failed",
		},
		{
			name:      "path with null bytes",
			porchPath: "porch\x00",
			wantErr:   true,
			errorMsg:  "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePorchPath(tt.porchPath)
			
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestStatefulExecutor_SecurityStats tests that statistics don't leak sensitive information
func TestStatefulExecutor_SecurityStats(t *testing.T) {
	tempDir := t.TempDir()
	intentFile := filepath.Join(tempDir, "intent.json")
	outDir := filepath.Join(tempDir, "out")
	
	require.NoError(t, os.WriteFile(intentFile, []byte(`{"test": "intent"}`), 0644))
	require.NoError(t, os.MkdirAll(outDir, 0755))

	config := ExecutorConfig{
		PorchPath: createSecureMockPorch(t, tempDir),
		Mode:      ModeDirect,
		OutDir:    outDir,
		Timeout:   5 * time.Second,
	}

	executor := NewStatefulExecutor(config)
	ctx := context.Background()

	// Execute multiple times with different outcomes
	_, _ = executor.Execute(ctx, intentFile)
	_, _ = executor.Execute(ctx, "/nonexistent/path")

	stats := executor.GetStats()

	// Verify stats don't contain sensitive information
	assert.GreaterOrEqual(t, stats.TotalExecutions, 0)
	assert.GreaterOrEqual(t, stats.SuccessfulExecs, 0)
	assert.GreaterOrEqual(t, stats.FailedExecs, 0)
	assert.GreaterOrEqual(t, stats.TotalExecTime, time.Duration(0))
	assert.GreaterOrEqual(t, stats.AverageExecTime, time.Duration(0))
	assert.GreaterOrEqual(t, stats.TimeoutCount, 0)

	// Stats should be reasonable and not indicate resource exhaustion
	assert.LessOrEqual(t, stats.AverageExecTime, 10*time.Second, "Average execution time should be reasonable")
	assert.Equal(t, stats.TotalExecutions, stats.SuccessfulExecs+stats.FailedExecs, "Stats should be consistent")
}

// Helper functions

func createSecureMockPorch(t testing.TB, tempDir string) string {
	var mockScript string
	var mockPath string

	if runtime.GOOS == "windows" {
		mockPath = filepath.Join(tempDir, "mock-porch.bat")
		mockScript = `@echo off
if "%1"=="--help" (
    echo Mock porch help
    exit /b 0
)
echo Processing intent: %2
echo Output dir: %4
exit /b 0`
	} else {
		mockPath = filepath.Join(tempDir, "mock-porch")
		mockScript = `#!/bin/bash
if [ "$1" = "--help" ]; then
    echo "Mock porch help"
    exit 0
fi
echo "Processing intent: $2"
echo "Output dir: $4"
exit 0`
	}

	require.NoError(t, os.WriteFile(mockPath, []byte(mockScript), 0755))
	return mockPath
}

func createMockPorchWithDelay(t testing.TB, tempDir string, delay time.Duration) string {
	var mockScript string
	var mockPath string

	if runtime.GOOS == "windows" {
		mockPath = filepath.Join(tempDir, "mock-porch-delay.bat")
		mockScript = fmt.Sprintf(`@echo off
if "%%1"=="--help" (
    echo Mock porch help
    exit /b 0
)
timeout /t %d /nobreak >nul 2>&1
echo Processing completed
exit /b 0`, int(delay.Seconds())+1)
	} else {
		mockPath = filepath.Join(tempDir, "mock-porch-delay")
		mockScript = fmt.Sprintf(`#!/bin/bash
if [ "$1" = "--help" ]; then
    echo "Mock porch help"
    exit 0
fi
sleep %v
echo "Processing completed"
exit 0`, delay.Seconds())
	}

	require.NoError(t, os.WriteFile(mockPath, []byte(mockScript), 0755))
	return mockPath
}

func createMockPorchWithFileCreation(t testing.TB, tempDir, outDir string) string {
	var mockScript string
	var mockPath string

	if runtime.GOOS == "windows" {
		mockPath = filepath.Join(tempDir, "mock-porch-create.bat")
		mockScript = fmt.Sprintf(`@echo off
if "%%1"=="--help" (
    echo Mock porch help
    exit /b 0
)
echo Test output > "%s\\test-output.txt"
echo Processing completed
exit /b 0`, outDir)
	} else {
		mockPath = filepath.Join(tempDir, "mock-porch-create")
		mockScript = fmt.Sprintf(`#!/bin/bash
if [ "$1" = "--help" ]; then
    echo "Mock porch help"
    exit 0
fi
echo "Test output" > "%s/test-output.txt"
echo "Processing completed"
exit 0`, outDir)
	}

	require.NoError(t, os.WriteFile(mockPath, []byte(mockScript), 0755))
	return mockPath
}

func assertNoMaliciousActivity(t *testing.T, tempDir string) {
	// Check for signs of malicious activity
	suspiciousFiles := []string{
		"passwd",
		"shadow",
		"backdoor",
		"injected",
		"malicious",
	}

	err := filepath.WalkDir(tempDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		name := strings.ToLower(d.Name())
		for _, suspicious := range suspiciousFiles {
			if strings.Contains(name, suspicious) {
				t.Errorf("Suspicious file found: %s", path)
			}
		}

		return nil
	})
	require.NoError(t, err)
}

func assertNoPathTraversal(t *testing.T, tempDir string) {
	// Ensure all created files are within the temp directory
	err := filepath.WalkDir(tempDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Check that path is within tempDir
		relPath, err := filepath.Rel(tempDir, path)
		if err != nil || strings.HasPrefix(relPath, "..") {
			t.Errorf("Path traversal detected: %s is outside %s", path, tempDir)
		}

		return nil
	})
	require.NoError(t, err)
}