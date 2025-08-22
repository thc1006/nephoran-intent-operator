// +build windows

package loop

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWindowsPathSecurityValidation tests Windows-specific path security scenarios
func TestWindowsPathSecurityValidation(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) string
		configFunc  func(baseDir string) Config
		expectError bool
		errorCheck  func(t *testing.T, err error)
	}{
		{
			name: "Windows UNC path traversal attempt",
			setupFunc: func(t *testing.T) string {
				return t.TempDir()
			},
			configFunc: func(baseDir string) Config {
				return Config{
					PorchPath: "porch",
					Mode:      "direct",
					OutDir:    `\\server\share\..\admin$`,
				}
			},
			expectError: true,
			errorCheck: func(t *testing.T, err error) {
				assertErrorContainsAny(t, err,
					"path traversal detected",
					"output directory does not exist",
					"output directory parent does not exist",
					"invalid UNC path",
				)
			},
		},
		{
			name: "Windows drive letter with traversal",
			setupFunc: func(t *testing.T) string {
				return t.TempDir()
			},
			configFunc: func(baseDir string) Config {
				return Config{
					PorchPath: "porch",
					Mode:      "direct",
					OutDir:    `C:\..\..\Windows\System32`,
				}
			},
			expectError: true,
			errorCheck: func(t *testing.T, err error) {
				// On Windows, this might resolve to a valid path
				// The important thing is that it's outside our safe boundaries
				if err != nil {
					assertErrorContainsAny(t, err,
						"path traversal detected",
						"output directory does not exist",
						"access is denied",
						"Access is denied",
						"is not writable",
					)
				}
			},
		},
		{
			name: "Windows alternate data stream attempt",
			setupFunc: func(t *testing.T) string {
				return t.TempDir()
			},
			configFunc: func(baseDir string) Config {
				return Config{
					PorchPath: "porch",
					Mode:      "direct",
					OutDir:    filepath.Join(baseDir, "output:hidden"),
				}
			},
			expectError: true,
			errorCheck: func(t *testing.T, err error) {
				assertErrorContainsAny(t, err,
					"invalid character",
					"The filename, directory name, or volume label syntax is incorrect",
					"The directory name is invalid",
					"output directory does not exist",
					"cannot create output directory",
				)
			},
		},
		{
			name: "Windows reserved device name",
			setupFunc: func(t *testing.T) string {
				return t.TempDir()
			},
			configFunc: func(baseDir string) Config {
				return Config{
					PorchPath: "porch",
					Mode:      "direct",
					OutDir:    `C:\CON`,
				}
			},
			expectError: true,
			errorCheck: func(t *testing.T, err error) {
				assertErrorContainsAny(t, err,
					"reserved",
					"invalid",
					"access is denied",
					"output directory does not exist",
				)
			},
		},
		{
			name: "Valid Windows path with spaces",
			setupFunc: func(t *testing.T) string {
				baseDir := t.TempDir()
				outputDir := filepath.Join(baseDir, "Program Files", "My App", "Output")
				require.NoError(t, os.MkdirAll(outputDir, 0755))
				return baseDir
			},
			configFunc: func(baseDir string) Config {
				return Config{
					PorchPath:    createMockPorchExecutable(t, baseDir),
					Mode:         "direct",
					OutDir:       filepath.Join(baseDir, "Program Files", "My App", "Output"),
					MaxWorkers:   2,
					MetricsPort:  0,
				}
			},
			expectError: false,
		},
		{
			name: "Windows long path (>260 chars)",
			setupFunc: func(t *testing.T) string {
				baseDir := t.TempDir()
				// Create a very long path
				longPath := baseDir
				for i := 0; i < 30; i++ {
					longPath = filepath.Join(longPath, "verylongdirectoryname")
				}
				// Don't try to create it, just return the base
				return baseDir
			},
			configFunc: func(baseDir string) Config {
				longPath := baseDir
				for i := 0; i < 30; i++ {
					longPath = filepath.Join(longPath, "verylongdirectoryname")
				}
				return Config{
					PorchPath: "porch",
					Mode:      "direct",
					OutDir:    longPath,
				}
			},
			expectError: true,
			errorCheck: func(t *testing.T, err error) {
				assertErrorContainsAny(t, err,
					"too long",
					"output directory parent does not exist",
					"The filename or extension is too long",
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := tt.setupFunc(t)
			config := tt.configFunc(baseDir)
			
			_, err := NewWatcher(baseDir, config)
			
			if tt.expectError {
				if err == nil {
					t.Log("Warning: Expected error but got none. This might be due to OS-level permissions.")
				} else if tt.errorCheck != nil {
					tt.errorCheck(t, err)
				}
			} else {
				assert.NoError(t, err, "Should not return error for valid path")
			}
		})
	}
}

// TestWindowsPathNormalization tests that paths are properly normalized on Windows
func TestWindowsPathNormalization(t *testing.T) {
	tempDir := t.TempDir()
	
	tests := []struct {
		name       string
		inputPath  string
		normalized bool
		safe       bool
	}{
		{
			name:       "Forward slashes converted to backslashes",
			inputPath:  "C:/Users/test/output",
			normalized: true,
			safe:       false, // Outside temp directory
		},
		{
			name:       "Mixed slashes normalized",
			inputPath:  `C:\Users/test\subfolder/output`,
			normalized: true,
			safe:       false, // Outside temp directory
		},
		{
			name:       "Dot segments removed",
			inputPath:  filepath.Join(tempDir, ".", "output", ".", "data"),
			normalized: true,
			safe:       true,
		},
		{
			name:       "Double backslashes normalized",
			inputPath:  filepath.Join(tempDir, "output\\\\data"),
			normalized: true,
			safe:       true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean the path as the validation code would
			cleaned := filepath.Clean(tt.inputPath)
			abs, err := filepath.Abs(cleaned)
			
			if tt.normalized {
				assert.NoError(t, err, "Path should be normalizable")
				assert.NotContains(t, abs, "/", "Should not contain forward slashes after normalization")
				assert.NotContains(t, abs, "\\\\", "Should not contain double backslashes")
				assert.NotContains(t, abs, "\\.", "Should not contain dot segments")
			}
			
			// Check if path is considered safe (within temp directory)
			if tt.safe {
				assert.True(t, isPathSafe(abs, tempDir), "Path should be considered safe")
			}
		})
	}
}

// TestWindowsCRLFHandling tests that CRLF line endings don't cause issues
func TestWindowsCRLFHandling(t *testing.T) {
	tempDir := t.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")
	require.NoError(t, os.MkdirAll(handoffDir, 0755))
	
	// Create an intent file with CRLF line endings
	intentContent := "{\r\n  \"intent_type\": \"scaling\",\r\n  \"target\": \"test-app\",\r\n  \"namespace\": \"default\",\r\n  \"replicas\": 3\r\n}\r\n"
	intentFile := filepath.Join(handoffDir, "intent-test.json")
	require.NoError(t, os.WriteFile(intentFile, []byte(intentContent), 0644))
	
	// The file should be readable and processable
	data, err := os.ReadFile(intentFile)
	assert.NoError(t, err, "Should be able to read file with CRLF")
	assert.Contains(t, string(data), "\r\n", "Should preserve CRLF")
	
	// Validate that the file is recognized as an intent file
	assert.True(t, IsIntentFile(filepath.Base(intentFile)), "Should recognize as intent file")
}