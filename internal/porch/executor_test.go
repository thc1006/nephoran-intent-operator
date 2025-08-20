package porch

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExecutor(t *testing.T) {
	tests := []struct {
		name           string
		config         ExecutorConfig
		expectedMode   string
		expectedTimeout time.Duration
	}{
		{
			name: "default configuration",
			config: ExecutorConfig{
				PorchPath: "porch",
				Mode:     "",
				OutDir:   "./out",
			},
			expectedMode:   ModeDirect,
			expectedTimeout: DefaultTimeout,
		},
		{
			name: "custom configuration",
			config: ExecutorConfig{
				PorchPath: "/usr/bin/porch",
				Mode:     ModeStructured,
				OutDir:   "/tmp/out",
				Timeout:  45 * time.Second,
			},
			expectedMode:   ModeStructured,
			expectedTimeout: 45 * time.Second,
		},
		{
			name: "invalid mode defaults to direct",
			config: ExecutorConfig{
				PorchPath: "porch",
				Mode:     "invalid-mode",
				OutDir:   "./out",
				Timeout:  10 * time.Second,
			},
			expectedMode:   ModeDirect,
			expectedTimeout: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewExecutor(tt.config)
			
			assert.NotNil(t, executor)
			assert.Equal(t, tt.expectedMode, executor.config.Mode)
			assert.Equal(t, tt.expectedTimeout, executor.config.Timeout)
			assert.Equal(t, tt.config.PorchPath, executor.config.PorchPath)
			assert.Equal(t, tt.config.OutDir, executor.config.OutDir)
		})
	}
}

func TestExecutor_BuildCommand(t *testing.T) {
	tempDir := t.TempDir()
	intentFile := filepath.Join(tempDir, "test-intent.json")
	outDir := filepath.Join(tempDir, "out")
	
	// Create test files/directories
	require.NoError(t, os.WriteFile(intentFile, []byte(`{"test": "intent"}`), 0644))
	require.NoError(t, os.MkdirAll(outDir, 0755))

	tests := []struct {
		name        string
		config      ExecutorConfig
		intentPath  string
		expectedCmd []string
		wantErr     bool
	}{
		{
			name: "direct mode",
			config: ExecutorConfig{
				PorchPath: "porch",
				Mode:     ModeDirect,
				OutDir:   outDir,
			},
			intentPath: intentFile,
			expectedCmd: []string{
				"porch",
				"-intent", filepath.Clean(intentFile),
				"-out", filepath.Clean(outDir),
			},
			wantErr: false,
		},
		{
			name: "structured mode",
			config: ExecutorConfig{
				PorchPath: "/usr/bin/porch",
				Mode:     ModeStructured,
				OutDir:   outDir,
			},
			intentPath: intentFile,
			expectedCmd: []string{
				"/usr/bin/porch",
				"-intent", filepath.Clean(intentFile),
				"-out", filepath.Clean(outDir),
				"-structured",
			},
			wantErr: false,
		},
		{
			name: "invalid mode",
			config: ExecutorConfig{
				PorchPath: "porch",
				Mode:     "invalid",
				OutDir:   outDir,
			},
			intentPath: intentFile,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewExecutor(tt.config)
			
			cmd, err := executor.buildCommand(tt.intentPath)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			
			// Normalize paths for comparison
			expectedCmd := make([]string, len(tt.expectedCmd))
			for i, arg := range tt.expectedCmd {
				if strings.Contains(arg, string(filepath.Separator)) {
					absPath, _ := filepath.Abs(arg)
					expectedCmd[i] = absPath
				} else {
					expectedCmd[i] = arg
				}
			}
			
			assert.Equal(t, expectedCmd, cmd)
		})
	}
}

func TestExecutor_Execute_MockCommand(t *testing.T) {
	tempDir := t.TempDir()
	intentFile := filepath.Join(tempDir, "test-intent.json")
	outDir := filepath.Join(tempDir, "out")
	
	require.NoError(t, os.WriteFile(intentFile, []byte(`{"test": "intent"}`), 0644))
	require.NoError(t, os.MkdirAll(outDir, 0755))

	tests := []struct {
		name           string
		mockCommand    func(t *testing.T) string // returns path to mock executable
		expectedResult func(t *testing.T, result *ExecutionResult)
		timeout        time.Duration
	}{
		{
			name: "successful execution",
			mockCommand: func(t *testing.T) string {
				return createMockPorch(t, tempDir, 0, "success output", "")
			},
			expectedResult: func(t *testing.T, result *ExecutionResult) {
				assert.True(t, result.Success)
				assert.Equal(t, 0, result.ExitCode)
				assert.Equal(t, "success output", result.Stdout)
				assert.Empty(t, result.Stderr)
				assert.Nil(t, result.Error)
			},
			timeout: 5 * time.Second,
		},
		{
			name: "failed execution",
			mockCommand: func(t *testing.T) string {
				return createMockPorch(t, tempDir, 1, "", "error occurred")
			},
			expectedResult: func(t *testing.T, result *ExecutionResult) {
				assert.False(t, result.Success)
				assert.Equal(t, 1, result.ExitCode)
				assert.Empty(t, result.Stdout)
				assert.Equal(t, "error occurred", result.Stderr)
				assert.NotNil(t, result.Error)
			},
			timeout: 5 * time.Second,
		},
		{
			name: "timeout execution",
			mockCommand: func(t *testing.T) string {
				return createMockPorch(t, tempDir, 0, "output", "", time.Second*3) // Sleep longer than timeout
			},
			expectedResult: func(t *testing.T, result *ExecutionResult) {
				assert.False(t, result.Success)
				assert.NotNil(t, result.Error)
				assert.Contains(t, result.Error.Error(), "timed out")
			},
			timeout: time.Second, // Short timeout
		},
		{
			name: "command not found",
			mockCommand: func(t *testing.T) string {
				return "/nonexistent/porch/command"
			},
			expectedResult: func(t *testing.T, result *ExecutionResult) {
				assert.False(t, result.Success)
				assert.Equal(t, -1, result.ExitCode)
				assert.NotNil(t, result.Error)
			},
			timeout: 5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPorchPath := tt.mockCommand(t)
			
			config := ExecutorConfig{
				PorchPath: mockPorchPath,
				Mode:     ModeDirect,
				OutDir:   outDir,
				Timeout:  tt.timeout,
			}
			
			executor := NewExecutor(config)
			ctx := context.Background()
			
			result, err := executor.Execute(ctx, intentFile)
			require.NoError(t, err) // Execute should not return error, all errors in result
			require.NotNil(t, result)
			
			tt.expectedResult(t, result)
			assert.NotZero(t, result.Duration)
			assert.NotEmpty(t, result.Command)
		})
	}
}

func TestExecutor_Execute_ContextCancellation(t *testing.T) {
	tempDir := t.TempDir()
	intentFile := filepath.Join(tempDir, "test-intent.json")
	outDir := filepath.Join(tempDir, "out")
	
	require.NoError(t, os.WriteFile(intentFile, []byte(`{"test": "intent"}`), 0644))
	require.NoError(t, os.MkdirAll(outDir, 0755))

	// Create mock porch that sleeps for a long time
	mockPorchPath := createMockPorch(t, tempDir, 0, "output", "", 10*time.Second)
	
	config := ExecutorConfig{
		PorchPath: mockPorchPath,
		Mode:     ModeDirect,
		OutDir:   outDir,
		Timeout:  30 * time.Second, // Long timeout
	}
	
	executor := NewExecutor(config)
	
	// Create context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	
	// Cancel context after 100ms
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()
	
	result, err := executor.Execute(ctx, intentFile)
	require.NoError(t, err)
	require.NotNil(t, result)
	
	// Should have failed due to context cancellation
	assert.False(t, result.Success)
	assert.NotNil(t, result.Error)
}

func TestStatefulExecutor(t *testing.T) {
	tempDir := t.TempDir()
	intentFile := filepath.Join(tempDir, "test-intent.json")
	outDir := filepath.Join(tempDir, "out")
	
	require.NoError(t, os.WriteFile(intentFile, []byte(`{"test": "intent"}`), 0644))
	require.NoError(t, os.MkdirAll(outDir, 0755))

	config := ExecutorConfig{
		PorchPath: createMockPorch(t, tempDir, 0, "success", ""),
		Mode:     ModeDirect,
		OutDir:   outDir,
		Timeout:  5 * time.Second,
	}
	
	executor := NewStatefulExecutor(config)
	ctx := context.Background()

	// Initial stats should be empty
	stats := executor.GetStats()
	assert.Equal(t, 0, stats.TotalExecutions)
	assert.Equal(t, 0, stats.SuccessfulExecs)
	assert.Equal(t, 0, stats.FailedExecs)
	assert.Equal(t, time.Duration(0), stats.AverageExecTime)

	// Execute successful command
	result, err := executor.Execute(ctx, intentFile)
	require.NoError(t, err)
	assert.True(t, result.Success)

	// Check updated stats
	stats = executor.GetStats()
	assert.Equal(t, 1, stats.TotalExecutions)
	assert.Equal(t, 1, stats.SuccessfulExecs)
	assert.Equal(t, 0, stats.FailedExecs)
	assert.NotZero(t, stats.AverageExecTime)
	assert.Equal(t, stats.AverageExecTime, stats.TotalExecTime)

	// Execute failed command
	executor.config.PorchPath = createMockPorch(t, tempDir, 1, "", "error")
	result, err = executor.Execute(ctx, intentFile)
	require.NoError(t, err)
	assert.False(t, result.Success)

	// Check updated stats
	stats = executor.GetStats()
	assert.Equal(t, 2, stats.TotalExecutions)
	assert.Equal(t, 1, stats.SuccessfulExecs)
	assert.Equal(t, 1, stats.FailedExecs)

	// Test reset stats
	executor.ResetStats()
	stats = executor.GetStats()
	assert.Equal(t, 0, stats.TotalExecutions)
	assert.Equal(t, 0, stats.SuccessfulExecs)
	assert.Equal(t, 0, stats.FailedExecs)
}

func TestStatefulExecutor_TimeoutTracking(t *testing.T) {
	tempDir := t.TempDir()
	intentFile := filepath.Join(tempDir, "test-intent.json")
	outDir := filepath.Join(tempDir, "out")
	
	require.NoError(t, os.WriteFile(intentFile, []byte(`{"test": "intent"}`), 0644))
	require.NoError(t, os.MkdirAll(outDir, 0755))

	config := ExecutorConfig{
		PorchPath: createMockPorch(t, tempDir, 0, "output", "", 2*time.Second), // Sleep longer than timeout
		Mode:     ModeDirect,
		OutDir:   outDir,
		Timeout:  500 * time.Millisecond, // Short timeout
	}
	
	executor := NewStatefulExecutor(config)
	ctx := context.Background()

	// Execute command that will timeout
	result, err := executor.Execute(ctx, intentFile)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error.Error(), "timed out")

	// Check timeout was tracked
	stats := executor.GetStats()
	assert.Equal(t, 1, stats.TotalExecutions)
	assert.Equal(t, 0, stats.SuccessfulExecs)
	assert.Equal(t, 1, stats.FailedExecs)
	assert.Equal(t, 1, stats.TimeoutCount)
}

func TestGetExitCode(t *testing.T) {
	tests := []struct {
		name         string
		setupCmd     func() (*exec.Cmd, error)
		expectedCode int
	}{
		{
			name: "successful command",
			setupCmd: func() (*exec.Cmd, error) {
				cmd := exec.Command("echo", "hello")
				err := cmd.Run()
				return cmd, err
			},
			expectedCode: 0,
		},
		{
			name: "failed command",
			setupCmd: func() (*exec.Cmd, error) {
				cmd := exec.Command("false") // Command that always returns 1 on Unix-like systems
				err := cmd.Run()
				return cmd, err
			},
			expectedCode: 1,
		},
		{
			name: "command not found",
			setupCmd: func() (*exec.Cmd, error) {
				cmd := exec.Command("nonexistent-command-12345")
				err := cmd.Run()
				return cmd, err
			},
			expectedCode: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip command not found test on Windows as behavior differs
			if tt.name == "command not found" && filepath.Separator == '\\' {
				t.Skip("Skipping command not found test on Windows")
			}
			
			// Skip false command test on Windows as it doesn't exist
			if tt.name == "failed command" && filepath.Separator == '\\' {
				t.Skip("Skipping false command test on Windows")
			}

			cmd, err := tt.setupCmd()
			exitCode := getExitCode(cmd, err)
			assert.Equal(t, tt.expectedCode, exitCode)
		})
	}
}

func TestValidatePorchPath(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		setupFunc func(t *testing.T) string // returns porch path
		wantErr   bool
	}{
		{
			name: "valid porch executable",
			setupFunc: func(t *testing.T) string {
				return createMockPorch(t, tempDir, 0, "Usage: porch [options]", "")
			},
			wantErr: false,
		},
		{
			name: "nonexistent executable",
			setupFunc: func(t *testing.T) string {
				return "/nonexistent/porch"
			},
			wantErr: true,
		},
		{
			name: "executable that fails",
			setupFunc: func(t *testing.T) string {
				return createMockPorch(t, tempDir, 1, "", "command failed")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			porchPath := tt.setupFunc(t)
			
			err := ValidatePorchPath(porchPath)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExecutor_ConcurrentExecutions(t *testing.T) {
	tempDir := t.TempDir()
	outDir := filepath.Join(tempDir, "out")
	require.NoError(t, os.MkdirAll(outDir, 0755))

	config := ExecutorConfig{
		PorchPath: createMockPorch(t, tempDir, 0, "success", ""),
		Mode:     ModeDirect,
		OutDir:   outDir,
		Timeout:  5 * time.Second,
	}
	
	executor := NewStatefulExecutor(config)
	ctx := context.Background()

	// Create multiple intent files
	intentFiles := make([]string, 10)
	for i := 0; i < 10; i++ {
		intentFile := filepath.Join(tempDir, fmt.Sprintf("intent-%d.json", i))
		require.NoError(t, os.WriteFile(intentFile, []byte(`{"test": "intent"}`), 0644))
		intentFiles[i] = intentFile
	}

	// Execute concurrently
	var wg sync.WaitGroup
	results := make(chan *ExecutionResult, 10)
	errors := make(chan error, 10)

	for _, intentFile := range intentFiles {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			result, err := executor.Execute(ctx, file)
			if err != nil {
				errors <- err
			} else {
				results <- result
			}
		}(intentFile)
	}

	wg.Wait()
	close(results)
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent execution error: %v", err)
	}

	// Verify all results
	successCount := 0
	for result := range results {
		if result.Success {
			successCount++
		}
	}
	assert.Equal(t, 10, successCount)

	// Verify stats
	stats := executor.GetStats()
	assert.Equal(t, 10, stats.TotalExecutions)
	assert.Equal(t, 10, stats.SuccessfulExecs)
	assert.Equal(t, 0, stats.FailedExecs)
}

// Helper function to create mock porch executable for testing
func createMockPorch(t testing.TB, tempDir string, exitCode int, stdout, stderr string, sleepDuration ...time.Duration) string {
	var sleep time.Duration
	if len(sleepDuration) > 0 {
		sleep = sleepDuration[0]
	}

	mockPath, err := CreateCrossPlatformMock(tempDir, CrossPlatformMockOptions{
		ExitCode: exitCode,
		Stdout:   stdout,
		Stderr:   stderr,
		Sleep:    sleep,
	})
	require.NoError(t, err)
	return mockPath
}

func BenchmarkExecutor_Execute(b *testing.B) {
	tempDir := b.TempDir()
	intentFile := filepath.Join(tempDir, "test-intent.json")
	outDir := filepath.Join(tempDir, "out")
	
	require.NoError(b, os.WriteFile(intentFile, []byte(`{"test": "intent"}`), 0644))
	require.NoError(b, os.MkdirAll(outDir, 0755))

	config := ExecutorConfig{
		PorchPath: createMockPorch(b, tempDir, 0, "success", ""),
		Mode:     ModeDirect,
		OutDir:   outDir,
		Timeout:  5 * time.Second,
	}
	
	executor := NewExecutor(config)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.Execute(ctx, intentFile)
	}
}

func BenchmarkStatefulExecutor_GetStats(b *testing.B) {
	config := ExecutorConfig{
		PorchPath: "mock-porch",
		Mode:     ModeDirect,
		OutDir:   "./out",
		Timeout:  5 * time.Second,
	}
	
	executor := NewStatefulExecutor(config)
	
	// Simulate some executions to have stats
	executor.stats.TotalExecutions = 1000
	executor.stats.SuccessfulExecs = 950
	executor.stats.FailedExecs = 50
	executor.stats.TotalExecTime = 10 * time.Minute
	executor.stats.AverageExecTime = 600 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = executor.GetStats()
	}
}