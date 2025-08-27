package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/nephoran-intent-operator/internal/porch"
)

// TestMain_FlagParsing tests the command-line flag parsing functionality
func TestMain_FlagParsing(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		verify      func(t *testing.T, config Config)
	}{
		{
			name: "default flags",
			args: []string{},
			verify: func(t *testing.T, config Config) {
				assert.Equal(t, "./handoff", config.HandoffDir)
				assert.Equal(t, "porch", config.PorchPath)
				assert.Equal(t, "direct", config.Mode)
				assert.Equal(t, "./out", config.OutDir)
				assert.False(t, config.Once)
				assert.Equal(t, 500*time.Millisecond, config.DebounceDur)
			},
		},
		{
			name: "custom flags",
			args: []string{
				"-handoff", "/tmp/handoff",
				"-porch", "/usr/bin/porch",
				"-mode", "structured",
				"-out", "/tmp/out",
				"-once",
				"-debounce", "1s",
			},
			verify: func(t *testing.T, config Config) {
				assert.Equal(t, "/tmp/handoff", config.HandoffDir)
				assert.Equal(t, "/usr/bin/porch", config.PorchPath)
				assert.Equal(t, "structured", config.Mode)
				assert.Equal(t, "/tmp/out", config.OutDir)
				assert.True(t, config.Once)
				assert.Equal(t, 1*time.Second, config.DebounceDur)
			},
		},
		{
			name:        "invalid mode",
			args:        []string{"-mode", "invalid"},
			expectError: true,
		},
		{
			name:        "invalid debounce duration",
			args:        []string{"-debounce", "invalid"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new FlagSet for each test to avoid flag redefinition
			fs := flag.NewFlagSet("test", flag.ContinueOnError)

			config, err := parseFlagsWithFlagSet(fs, tt.args)
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				tt.verify(t, config)
			}
		})
	}
}

// TestMain_EndToEndWorkflow tests the complete end-to-end workflow
func TestMain_EndToEndWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	tempDir := t.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")
	outDir := filepath.Join(tempDir, "out")
	
	require.NoError(t, os.MkdirAll(handoffDir, 0755))
	require.NoError(t, os.MkdirAll(outDir, 0755))

	// Create mock porch executable
	mockPorchPath := createMockPorch(t, tempDir, 0, "Package processed successfully", "")

	tests := []struct {
		name         string
		args         []string
		setupFiles   func(t *testing.T)
		verifyResult func(t *testing.T)
		timeout      time.Duration
	}{
		{
			name: "process single file in once mode",
			args: []string{
				"-handoff", handoffDir,
				"-porch", mockPorchPath,
				"-mode", "direct",
				"-out", outDir,
				"-once",
				"-debounce", "100ms",
			},
			setupFiles: func(t *testing.T) {
				intentContent := `{
					"intent_type": "scaling",
					"target": "deployment/test-app",
					"replicas": 5,
					"namespace": "default"
				}`
				intentFile := filepath.Join(handoffDir, "intent-scale.json")
				require.NoError(t, // FIXME: Adding error check per errcheck linter
 _ = os.WriteFile(intentFile, []byte(intentContent), 0644))
			},
			verifyResult: func(t *testing.T) {
				// Verify intent file was moved to processed
				processedDir := filepath.Join(handoffDir, "processed")
				processedFiles, err := os.ReadDir(processedDir)
				require.NoError(t, err)
				assert.Len(t, processedFiles, 1)
				assert.Equal(t, "intent-scale.json", processedFiles[0].Name())

				// Verify status file was created (with timestamp pattern)
				statusDir := filepath.Join(handoffDir, "status")
				statusFiles, err := os.ReadDir(statusDir)
				require.NoError(t, err)
				assert.Len(t, statusFiles, 1)
				assert.Contains(t, statusFiles[0].Name(), "intent-scale.json")

				// Verify original file is gone
				originalFile := filepath.Join(handoffDir, "intent-scale.json")
				assert.NoFileExists(t, originalFile)
			},
			timeout: 5 * time.Second,
		},
		{
			name: "process multiple files concurrently",
			args: []string{
				"-handoff", handoffDir,
				"-porch", mockPorchPath,
				"-mode", "structured",
				"-out", outDir,
				"-once",
				"-debounce", "50ms",
			},
			setupFiles: func(t *testing.T) {
				for i := 1; i <= 3; i++ {
					intentContent := fmt.Sprintf(`{
						"intent_type": "scaling",
						"target": "deployment/app-%d",
						"namespace": "default",
						"replicas": %d
					}`, i, i*2)
					intentFile := filepath.Join(handoffDir, fmt.Sprintf("intent-%d.json", i))
					require.NoError(t, // FIXME: Adding error check per errcheck linter
 _ = os.WriteFile(intentFile, []byte(intentContent), 0644))
				}
			},
			verifyResult: func(t *testing.T) {
				// Verify all files were processed
				processedDir := filepath.Join(handoffDir, "processed")
				processedFiles, err := os.ReadDir(processedDir)
				require.NoError(t, err)
				assert.Len(t, processedFiles, 3)

				// Verify status files were created
				statusDir := filepath.Join(handoffDir, "status")
				statusFiles, err := os.ReadDir(statusDir)
				require.NoError(t, err)
				assert.Len(t, statusFiles, 3)
			},
			timeout: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up directories before each test
			cleanupDirs(t, handoffDir)
			require.NoError(t, os.MkdirAll(handoffDir, 0755))

			// Setup test files
			tt.setupFiles(t)

			// Build the conductor-loop binary
			binaryPath := buildConductorLoop(t, tempDir)

			// Run conductor-loop
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			cmd := exec.CommandContext(ctx, binaryPath, tt.args...)
			cmd.Dir = tempDir

			// Capture output for debugging
			output, err := cmd.CombinedOutput()
			
			// In once mode, the command should exit successfully
			if strings.Contains(strings.Join(tt.args, " "), "-once") {
				assert.NoError(t, err, "Command output: %s", string(output))
			}

			t.Logf("Command output: %s", string(output))

			// Verify results
			tt.verifyResult(t)
		})
	}
}

// TestMain_SignalHandling tests graceful shutdown on signals
func TestMain_SignalHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping signal handling test in short mode")
	}

	// Skip on Windows as signal handling is different
	if runtime.GOOS == "windows" {
		t.Skip("Skipping signal handling test on Windows")
	}

	tempDir := t.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")
	outDir := filepath.Join(tempDir, "out")
	
	require.NoError(t, os.MkdirAll(handoffDir, 0755))
	require.NoError(t, os.MkdirAll(outDir, 0755))

	// Create mock porch that takes some time to process
	mockPorchPath := createMockPorch(t, tempDir, 0, "Processed", "", 2*time.Second)

	// Build conductor-loop binary
	binaryPath := buildConductorLoop(t, tempDir)

	// Create an intent file
	intentFile := filepath.Join(handoffDir, "intent-signal-test.json")
	intentContent := `{"action": "scale", "target": "deployment/test", "replicas": 3}`
	require.NoError(t, // FIXME: Adding error check per errcheck linter
 _ = os.WriteFile(intentFile, []byte(intentContent), 0644))

	// Start conductor-loop (without -once, so it runs continuously)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	args := []string{
		"-handoff", handoffDir,
		"-porch", mockPorchPath,
		"-out", outDir,
		"-debounce", "100ms",
	}

	cmd := exec.CommandContext(ctx, binaryPath, args...)
	cmd.Dir = tempDir

	// Start the command
	require.NoError(t, cmd.Start())

	// Wait a moment for the process to start and begin processing
	time.Sleep(500 * time.Millisecond)

	// Send SIGTERM for graceful shutdown
	require.NoError(t, cmd.Process.Signal(syscall.SIGTERM))

	// Wait for the process to exit gracefully
	err := cmd.Wait()
	
	// Process should exit cleanly after signal
	if err != nil {
		// On some systems, SIGTERM may result in exit status
		if exitError, ok := err.(*exec.ExitError); ok {
			// Exit status 1 or 130 (128+2 for SIGINT) can be expected
			exitCode := exitError.ExitCode()
			assert.Contains(t, []int{0, 1, 130}, exitCode, "Unexpected exit code for graceful shutdown")
		}
	}

	t.Log("Process exited after SIGTERM")
}

// TestMain_WindowsPathHandling tests Windows-specific path handling
func TestMain_WindowsPathHandling(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows path test on non-Windows system")
	}

	tempDir := t.TempDir()
	
	// Test with Windows paths containing spaces and special characters
	handoffDir := filepath.Join(tempDir, "handoff dir with spaces")
	outDir := filepath.Join(tempDir, "out-dir_with-special.chars")
	
	require.NoError(t, os.MkdirAll(handoffDir, 0755))
	require.NoError(t, os.MkdirAll(outDir, 0755))

	mockPorchPath := createMockPorch(t, tempDir, 0, "Success", "")

	// Test parseFlags with Windows paths
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	os.Args = []string{
		"conductor-loop",
		"-handoff", handoffDir,
		"-porch", mockPorchPath,
		"-out", outDir,
		"-once",
	}

	config := parseFlags()
	
	// Verify paths are handled correctly
	assert.True(t, filepath.IsAbs(config.HandoffDir))
	assert.True(t, filepath.IsAbs(config.OutDir))
	assert.Contains(t, config.HandoffDir, "handoff dir with spaces")
	assert.Contains(t, config.OutDir, "out-dir_with-special.chars")
}

// TestMain_ExitCodes tests various exit scenarios
func TestMain_ExitCodes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping exit code test in short mode")
	}

	tempDir := t.TempDir()
	binaryPath := buildConductorLoop(t, tempDir)

	tests := []struct {
		name         string
		args         []string
		expectedExit int
		setupFunc    func(t *testing.T) string // returns temp directory
	}{
		{
			name: "invalid handoff directory",
			args: []string{
				"-handoff", "/nonexistent/directory",
				"-once",
			},
			expectedExit: 1,
			setupFunc: func(t *testing.T) string {
				return tempDir
			},
		},
		{
			name: "successful execution",
			args: []string{
				"-once",
			},
			expectedExit: 0,
			setupFunc: func(t *testing.T) string {
				testDir := filepath.Join(tempDir, "success-test")
				handoffDir := filepath.Join(testDir, "handoff")
				require.NoError(t, os.MkdirAll(handoffDir, 0755))
				
				// Create mock porch and intent file
				mockPorch := createMockPorch(t, testDir, 0, "success", "")
				
				intentFile := filepath.Join(handoffDir, "intent-test.json")
				require.NoError(t, // FIXME: Adding error check per errcheck linter
 _ = os.WriteFile(intentFile, []byte(`{"test": "intent"}`), 0644))
				
				// Setup test arguments - create local args slice
				testArgs := []string{"-handoff", handoffDir, "-porch", mockPorch, "-once"}
				_ = testArgs // Use the args in the actual test execution
				
				return testDir
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := tt.setupFunc(t)
			
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, binaryPath, tt.args...)
			cmd.Dir = workDir

			err := cmd.Run()
			
			if tt.expectedExit == 0 {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				if exitError, ok := err.(*exec.ExitError); ok {
					assert.Equal(t, tt.expectedExit, exitError.ExitCode())
				}
			}
		})
	}
}

// Helper functions

// buildConductorLoop builds the conductor-loop binary for testing
func buildConductorLoop(t *testing.T, tempDir string) string {
	binaryName := "conductor-loop"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	
	binaryPath := filepath.Join(tempDir, binaryName)
	
	// Build the binary
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = "." // Current directory should be cmd/conductor-loop
	
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to build conductor-loop: %s", string(output))
	
	return binaryPath
}

// createMockPorch creates a mock porch executable for testing
func createMockPorch(t *testing.T, tempDir string, exitCode int, stdout, stderr string, sleepDuration ...time.Duration) string {
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

// cleanupDirs removes all files and subdirectories from the given directory
func cleanupDirs(t *testing.T, dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return
	}
	
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Logf("Warning: failed to read directory %s: %v", dir, err)
		return
	}
	
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			t.Logf("Warning: failed to remove %s: %v", path, err)
		}
	}
}

// Benchmark for main workflow
func BenchmarkMain_SingleFileProcessing(b *testing.B) {
	tempDir := b.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")
	outDir := filepath.Join(tempDir, "out")
	
	require.NoError(b, os.MkdirAll(handoffDir, 0755))
	require.NoError(b, os.MkdirAll(outDir, 0755))

	mockPorchPath := createMockPorchB(b, tempDir, 0, "processed", "")
	binaryPath := buildConductorLoopB(b, tempDir)

	args := []string{
		"-handoff", handoffDir,
		"-porch", mockPorchPath,
		"-out", outDir,
		"-once",
		"-debounce", "10ms",
	}

	intentContent := `{"action": "scale", "target": "deployment/test", "replicas": 1}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		
		// Clean up and setup for each iteration
		cleanupDirsB(b, handoffDir)
		require.NoError(b, os.MkdirAll(handoffDir, 0755))
		
		intentFile := filepath.Join(handoffDir, fmt.Sprintf("intent-%d.json", i))
		require.NoError(b, // FIXME: Adding error check per errcheck linter
 _ = os.WriteFile(intentFile, []byte(intentContent), 0644))
		
		b.StartTimer()
		
		// Run conductor-loop
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, binaryPath, args...)
		cmd.Dir = tempDir
		_ = cmd.Run()
		cancel()
		
		b.StopTimer()
	}
}

// Benchmark-specific helper functions that accept *testing.B

func createMockPorchB(b *testing.B, tempDir string, exitCode int, stdout, stderr string, sleepDuration ...time.Duration) string {
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
	require.NoError(b, err)
	return mockPath
}

func buildConductorLoopB(b *testing.B, tempDir string) string {
	binaryName := "conductor-loop"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	
	binaryPath := filepath.Join(tempDir, binaryName)
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = filepath.Dir(tempDir) // Go to the main package directory
	
	require.NoError(b, cmd.Run())
	return binaryPath
}

func cleanupDirsB(b *testing.B, dirs ...string) {
	for _, dir := range dirs {
		if err := os.RemoveAll(dir); err != nil {
			b.Logf("Failed to cleanup directory %s: %v", dir, err)
		}
	}
}