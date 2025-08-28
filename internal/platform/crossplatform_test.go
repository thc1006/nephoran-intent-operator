package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetScriptExtension(t *testing.T) {
	ext := GetScriptExtension()
	
	if runtime.GOOS == "windows" {
		assert.Equal(t, ".bat", ext)
	} else {
		assert.Equal(t, ".sh", ext)
	}
}

func TestGetScriptPath(t *testing.T) {
	tempDir := t.TempDir()
	
	path := GetScriptPath(tempDir, "test-script")
	
	expectedExt := GetScriptExtension()
	assert.True(t, strings.HasSuffix(path, "test-script"+expectedExt))
	assert.True(t, strings.HasPrefix(path, tempDir))
}

func TestGetExecutablePath(t *testing.T) {
	tempDir := t.TempDir()
	
	tests := []struct {
		name     string
		baseName string
		setup    func(string) // setup function to create test files
		expected string
	}{
		{
			name:     "basic executable",
			baseName: "test-app",
			setup: func(dir string) {
				if runtime.GOOS == "windows" {
					// Create a .exe file
					f, _ := os.Create(filepath.Join(dir, "test-app.exe"))
					f.Close()
				} else {
					// Create a regular executable
					f, _ := os.Create(filepath.Join(dir, "test-app"))
					f.Close()
				}
			},
			expected: func() string {
				if runtime.GOOS == "windows" {
					return "test-app.exe"
				}
				return "test-app"
			}(),
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test files
			if tt.setup != nil {
				tt.setup(tempDir)
			}
			
			path := GetExecutablePath(tempDir, tt.baseName)
			assert.True(t, strings.HasSuffix(path, tt.expected))
		})
	}
}

func TestCreateCrossPlatformScript_MockPorch(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := GetScriptPath(tempDir, "test-mock-porch")
	
	opts := ScriptOptions{
		ExitCode: 0,
		Stdout:   "Test output",
		Stderr:   "Test error",
		Sleep:    100 * time.Millisecond,
	}
	
	err := CreateCrossPlatformScript(scriptPath, MockPorchScript, opts)
	require.NoError(t, err)
	
	// Verify file was created
	assert.FileExists(t, scriptPath)
	
	// Verify file content
	content, err := os.ReadFile(scriptPath)
	require.NoError(t, err)
	
	contentStr := string(content)
	
	if runtime.GOOS == "windows" {
		assert.Contains(t, contentStr, "@echo off")
		assert.Contains(t, contentStr, "echo Test output")
		assert.Contains(t, contentStr, "exit /b 0")
	} else {
		assert.Contains(t, contentStr, "#!/bin/bash")
		assert.Contains(t, contentStr, `echo "Test output"`)
		assert.Contains(t, contentStr, "exit 0")
	}
}

func TestCreateCrossPlatformScript_GenericScript(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := GetScriptPath(tempDir, "test-generic")
	
	opts := ScriptOptions{
		ExitCode: 42,
		Stdout:   "Generic output",
	}
	
	err := CreateCrossPlatformScript(scriptPath, GenericScript, opts)
	require.NoError(t, err)
	
	// Verify file was created
	assert.FileExists(t, scriptPath)
	
	// Verify file permissions on Unix
	if runtime.GOOS != "windows" {
		info, err := os.Stat(scriptPath)
		require.NoError(t, err)
		assert.True(t, info.Mode()&0111 != 0, "Script should be executable")
	}
}

func TestCreateCrossPlatformScript_WithFailOnPattern(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := GetScriptPath(tempDir, "test-pattern-fail")
	
	opts := ScriptOptions{
		ExitCode:      1,
		FailOnPattern: "ERROR",
	}
	
	err := CreateCrossPlatformScript(scriptPath, MockPorchScript, opts)
	require.NoError(t, err)
	
	content, err := os.ReadFile(scriptPath)
	require.NoError(t, err)
	
	contentStr := string(content)
	
	if runtime.GOOS == "windows" {
		assert.Contains(t, contentStr, "findstr")
		assert.Contains(t, contentStr, "ERROR")
	} else {
		assert.Contains(t, contentStr, "grep")
		assert.Contains(t, contentStr, "ERROR")
	}
}

func TestCreateCrossPlatformScript_WithCustomCommands(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := GetScriptPath(tempDir, "test-custom")
	
	opts := ScriptOptions{
		ExitCode: 0,
		CustomCommands: struct {
			Windows []string
			Unix    []string
		}{
			Windows: []string{"echo Windows custom command"},
			Unix:    []string{"echo 'Unix custom command'"},
		},
	}
	
	err := CreateCrossPlatformScript(scriptPath, GenericScript, opts)
	require.NoError(t, err)
	
	content, err := os.ReadFile(scriptPath)
	require.NoError(t, err)
	
	contentStr := string(content)
	
	if runtime.GOOS == "windows" {
		assert.Contains(t, contentStr, "Windows custom command")
	} else {
		assert.Contains(t, contentStr, "Unix custom command")
	}
}

func TestIsExecutable(t *testing.T) {
	tempDir := t.TempDir()
	
	tests := []struct {
		name       string
		filename   string
		mode       os.FileMode
		expected   bool
		windowsExt bool // whether this test relies on Windows extensions
	}{
		{
			name:     "executable file on Unix",
			filename: "test-exec",
			mode:     0755,
			expected: runtime.GOOS != "windows", // true on Unix, false on Windows
		},
		{
			name:     "non-executable file on Unix",
			filename: "test-noexec",
			mode:     0644,
			expected: false,
		},
		{
			name:       "Windows .exe file",
			filename:   "test.exe",
			mode:       0644,
			expected:   runtime.GOOS == "windows",
			windowsExt: true,
		},
		{
			name:       "Windows .bat file",
			filename:   "test.bat",
			mode:       0644,
			expected:   runtime.GOOS == "windows",
			windowsExt: true,
		},
		{
			name:       "Windows .cmd file",
			filename:   "test.cmd",
			mode:       0644,
			expected:   runtime.GOOS == "windows",
			windowsExt: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip Windows extension tests on Unix and vice versa
			if tt.windowsExt && runtime.GOOS != "windows" {
				t.Skip("Skipping Windows extension test on Unix")
			}
			if !tt.windowsExt && runtime.GOOS == "windows" && strings.Contains(tt.name, "Unix") {
				t.Skip("Skipping Unix permission test on Windows")
			}
			
			filePath := filepath.Join(tempDir, tt.filename)
			
			// Create file
			file, err := os.Create(filePath)
			require.NoError(t, err)
			file.Close()
			
			// Set permissions
			err = os.Chmod(filePath, tt.mode)
			require.NoError(t, err)
			
			result := IsExecutable(filePath)
			assert.Equal(t, tt.expected, result, "IsExecutable result mismatch for %s", tt.filename)
		})
	}
}

func TestMakeExecutable(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test-file")
	
	// Create a non-executable file
	file, err := os.Create(filePath)
	require.NoError(t, err)
	file.Close()
	
	// Make it non-executable
	err = os.Chmod(filePath, 0644)
	require.NoError(t, err)
	
	// Test MakeExecutable
	err = MakeExecutable(filePath)
	require.NoError(t, err)
	
	// Verify it's now executable (on Unix) or at least readable (on Windows)
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filePath)
		require.NoError(t, err)
		assert.True(t, info.Mode()&0111 != 0, "File should be executable after MakeExecutable")
	} else {
		// On Windows, just verify the file is still accessible
		_, err := os.Stat(filePath)
		assert.NoError(t, err)
	}
}

func TestMockPorchScriptBehavior(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := GetScriptPath(tempDir, "behavior-test")
	
	tests := []struct {
		name string
		opts ScriptOptions
	}{
		{
			name: "success script",
			opts: ScriptOptions{
				ExitCode: 0,
				Stdout:   "Success message",
			},
		},
		{
			name: "failure script",
			opts: ScriptOptions{
				ExitCode: 1,
				Stderr:   "Error message",
			},
		},
		{
			name: "script with sleep",
			opts: ScriptOptions{
				ExitCode: 0,
				Sleep:    50 * time.Millisecond,
				Stdout:   "Delayed success",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create script
			err := CreateCrossPlatformScript(scriptPath, MockPorchScript, tt.opts)
			require.NoError(t, err)
			
			// Verify script content contains expected elements
			content, err := os.ReadFile(scriptPath)
			require.NoError(t, err)
			
			contentStr := string(content)
			
			// Verify platform-specific elements
			if runtime.GOOS == "windows" {
				assert.Contains(t, contentStr, "@echo off")
				assert.Contains(t, contentStr, fmt.Sprintf("exit /b %d", tt.opts.ExitCode))
			} else {
				assert.Contains(t, contentStr, "#!/bin/bash")
				assert.Contains(t, contentStr, fmt.Sprintf("exit %d", tt.opts.ExitCode))
			}
			
			// Clean up for next iteration
			os.Remove(scriptPath)
		})
	}
}

// Integration test that verifies the complete workflow
func TestCrossPlatformWorkflow(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create a mock porch script
	scriptPath := GetScriptPath(tempDir, "workflow-test")
	opts := ScriptOptions{
		ExitCode: 0,
		Stdout:   "Workflow test completed",
	}
	
	err := CreateCrossPlatformScript(scriptPath, MockPorchScript, opts)
	require.NoError(t, err)
	
	// Verify the script exists and is executable
	assert.FileExists(t, scriptPath)
	assert.True(t, IsExecutable(scriptPath))
	
	// Verify we can get the correct path
	retrievedPath := GetScriptPath(tempDir, "workflow-test")
	assert.Equal(t, scriptPath, retrievedPath)
	
	t.Logf("Cross-platform workflow test completed successfully on %s", runtime.GOOS)
}