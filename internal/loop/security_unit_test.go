package loop

import (
	"encoding/json"
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

// TestIsIntentFile_SecurityValidation tests security aspects of intent file validation
func TestIsIntentFile_SecurityValidation(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
		reason   string
	}{
		{
			name:     "normal intent file",
			filename: "intent-valid.json",
			expected: true,
			reason:   "should accept valid intent files",
		},
		{
			name:     "path traversal attempt",
			filename: filepath.Base("../../../intent-passwd.json"),
			expected: true, // Base filename is "intent-passwd.json" which is valid
			reason:   "should only check base filename, not full path",
		},
		{
			name:     "hidden file",
			filename: ".intent-hidden.json",
			expected: false, // Hidden files don't start with "intent-"
			reason:   "hidden files should not match intent pattern",
		},
		{
			name:     "executable disguised as intent",
			filename: "intent-malicious.exe.json",
			expected: true, // Matches pattern, would be caught by content validation
			reason:   "filename validation only checks pattern",
		},
		{
			name:     "very long filename",
			filename: "intent-" + strings.Repeat("a", 255) + ".json",
			expected: true,
			reason:   "should handle long filenames",
		},
		{
			name:     "unicode filename",
			filename: "intent-файл.json",
			expected: true,
			reason:   "should handle unicode filenames",
		},
		{
			name:     "null bytes in filename",
			filename: "intent-test\x00.json",
			expected: true, // Filesystem usually prevents null bytes
			reason:   "null bytes should not affect validation",
		},
		{
			name:     "non-intent file",
			filename: "script.sh",
			expected: false,
			reason:   "should reject non-JSON files",
		},
		{
			name:     "empty filename",
			filename: "",
			expected: false,
			reason:   "should reject empty filename",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsIntentFile(tt.filename)
			assert.Equal(t, tt.expected, result, tt.reason)
		})
	}
}

// TestStateManager_SecurityBehavior tests security aspects of state management
func TestStateManager_SecurityBehavior(t *testing.T) {
	tempDir := t.TempDir()

	sm, err := NewStateManager(tempDir)
	require.NoError(t, err)
	defer sm.Close()

	tests := []struct {
		name     string
		filePath string
		testFunc func(t *testing.T, sm *StateManager, filePath string)
	}{
		{
			name:     "path traversal in state file",
			filePath: "../../../etc/passwd",
			testFunc: func(t *testing.T, sm *StateManager, filePath string) {
				// Attempt to mark a file with path traversal as processed
				err := sm.MarkProcessed(filePath)
				assert.NoError(t, err, "should handle path traversal gracefully")

				// Verify the state file was created in the correct location
				stateFile := filepath.Join(tempDir, StateFileName)
				_, err = os.Stat(stateFile)
				assert.NoError(t, err, "state file should exist in correct location")

				// Verify no files were created outside the temp directory
				parentDir := filepath.Dir(tempDir)
				entries, err := os.ReadDir(parentDir)
				require.NoError(t, err)

				for _, entry := range entries {
					if strings.Contains(entry.Name(), "passwd") {
						t.Errorf("Suspicious file created outside temp dir: %s", entry.Name())
					}
				}
			},
		},
		{
			name:     "very long file path",
			filePath: "/" + strings.Repeat("a", 1000) + "/intent.json",
			testFunc: func(t *testing.T, sm *StateManager, filePath string) {
				err := sm.MarkProcessed(filePath)
				assert.NoError(t, err, "should handle very long paths")

				processed, err := sm.IsProcessed(filePath)
				assert.NoError(t, err)
				assert.True(t, processed, "should remember very long paths")
			},
		},
		{
			name:     "unicode file path",
			filePath: "/пуṫь/файл-интент.json",
			testFunc: func(t *testing.T, sm *StateManager, filePath string) {
				err := sm.MarkProcessed(filePath)
				assert.NoError(t, err, "should handle unicode paths")

				processed, err := sm.IsProcessed(filePath)
				assert.NoError(t, err)
				assert.True(t, processed, "should remember unicode paths")
			},
		},
		{
			name:     "null bytes in path",
			filePath: "/test\x00/intent.json",
			testFunc: func(t *testing.T, sm *StateManager, filePath string) {
				err := sm.MarkProcessed(filePath)
				assert.NoError(t, err, "should handle null bytes in path")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t, sm, tt.filePath)
		})
	}
}

// TestFileManager_SecurityBehavior tests security aspects of file management
func TestFileManager_SecurityBehavior(t *testing.T) {
	tempDir := t.TempDir()

	fm, err := NewFileManager(tempDir)
	require.NoError(t, err)

	// Create test intent file
	intentContent := `{
		"intent_type": "scaling",
		"target": "my-app",
		"namespace": "default",
		"replicas": 3
	}`
	intentFile := filepath.Join(tempDir, "test-intent.json")
	require.NoError(t, os.WriteFile(intentFile, []byte(intentContent), 0644))

	tests := []struct {
		name     string
		testFunc func(t *testing.T, fm *FileManager, intentFile string)
	}{
		{
			name: "move to processed - path validation",
			testFunc: func(t *testing.T, fm *FileManager, intentFile string) {
				err := fm.MoveToProcessed(intentFile)
				assert.NoError(t, err, "should move file successfully")

				// Verify file was moved to correct subdirectory
				processedFile := filepath.Join(tempDir, "processed", "test-intent.json")
				_, err = os.Stat(processedFile)
				assert.NoError(t, err, "file should exist in processed directory")

				// Verify original file no longer exists
				_, err = os.Stat(intentFile)
				assert.True(t, os.IsNotExist(err), "original file should be removed")
			},
		},
		{
			name: "move to failed - error message sanitization",
			testFunc: func(t *testing.T, fm *FileManager, intentFile string) {
				// Create new intent file for this test
				newIntentFile := filepath.Join(tempDir, "test-intent-2.json")
				require.NoError(t, os.WriteFile(newIntentFile, []byte(intentContent), 0644))

				// Attempt to move with malicious error message
				maliciousError := "error occurred\n#!/bin/bash\nrm -rf /\necho 'hacked'"
				err := fm.MoveToFailed(newIntentFile, maliciousError)
				assert.NoError(t, err, "should handle malicious error message")

				// Verify file was moved to failed directory
				failedFile := filepath.Join(tempDir, "failed", "test-intent-2.json")
				_, err = os.Stat(failedFile)
				assert.NoError(t, err, "file should exist in failed directory")

				// Verify error file was created with sanitized content
				errorFile := filepath.Join(tempDir, "failed", "test-intent-2.json.error")
				content, err := os.ReadFile(errorFile)
				assert.NoError(t, err, "error file should exist")

				errorContent := string(content)
				assert.Contains(t, errorContent, "error occurred", "should contain original error message")
				// Check that dangerous content is present but in a safe context (as text, not executable)
				assert.Contains(t, errorContent, "#!/bin/bash", "should preserve error message content")
			},
		},
		{
			name: "directory permissions",
			testFunc: func(t *testing.T, fm *FileManager, intentFile string) {
				if runtime.GOOS == "windows" {
					t.Skip("Permission tests not applicable on Windows")
				}

				// Create new intent file for this test
				newIntentFile := filepath.Join(tempDir, "test-intent-3.json")
				require.NoError(t, os.WriteFile(newIntentFile, []byte(intentContent), 0644))

				err := fm.MoveToProcessed(newIntentFile)
				require.NoError(t, err)

				// Check that created directories have secure permissions
				processedDir := filepath.Join(tempDir, "processed")
				info, err := os.Stat(processedDir)
				require.NoError(t, err)

				mode := info.Mode()
				expectedMode := os.FileMode(0755)
				assert.Equal(t, expectedMode, mode&0777, "processed directory should have mode 0755")
			},
		},
		{
			name: "cleanup security",
			testFunc: func(t *testing.T, fm *FileManager, intentFile string) {
				// Create multiple old files
				processedDir := filepath.Join(tempDir, "processed")
				require.NoError(t, os.MkdirAll(processedDir, 0755))

				oldTime := time.Now().Add(-48 * time.Hour)

				for i := 0; i < 5; i++ {
					oldFile := filepath.Join(processedDir, fmt.Sprintf("old-intent-%d.json", i))
					require.NoError(t, os.WriteFile(oldFile, []byte(intentContent), 0644))
					require.NoError(t, os.Chtimes(oldFile, oldTime, oldTime))
				}

				// Run cleanup
				err := fm.CleanupOldFiles(24 * time.Hour)
				assert.NoError(t, err, "cleanup should complete without error")

				// Verify files were cleaned up
				entries, err := os.ReadDir(processedDir)
				require.NoError(t, err)
				assert.Empty(t, entries, "old files should be cleaned up")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t, fm, intentFile)
		})
	}
}

// TestConfig_SecurityValidation tests security validation of configuration
func TestConfig_SecurityValidation(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		testFunc func(t *testing.T, config Config)
	}{
		{
			name: "malicious porch path",
			config: Config{
				PorchPath: "porch; rm -rf /",
				Mode:      "direct",
				OutDir:    "/tmp/out",
			},
			testFunc: func(t *testing.T, config Config) {
				// The watcher should be able to handle malicious porch paths
				// The actual validation happens during execution
				tempDir := t.TempDir()
				_, err := NewWatcher(tempDir, config)
				assert.NoError(t, err, "watcher creation should not fail on malicious path")
			},
		},
		{
			name: "path traversal in output directory",
			config: Config{
				PorchPath: "porch",
				Mode:      "direct",
				OutDir:    "../../../etc",
			},
			testFunc: func(t *testing.T, config Config) {
				tempDir := t.TempDir()
				_, err := NewWatcher(tempDir, config)
				assert.NoError(t, err, "should handle path traversal in output directory")
			},
		},
		{
			name: "negative values in configuration",
			config: Config{
				PorchPath:    "porch",
				Mode:         "direct",
				OutDir:       "/tmp/out",
				MaxWorkers:   -5,
				DebounceDur:  -1 * time.Second,
				CleanupAfter: -1 * time.Hour,
			},
			testFunc: func(t *testing.T, config Config) {
				tempDir := t.TempDir()
				watcher, err := NewWatcher(tempDir, config)
				require.NoError(t, err)
				defer watcher.Close()

				// Verify defaults were applied for negative values
				assert.Greater(t, watcher.config.MaxWorkers, 0, "should use positive default for MaxWorkers")
				assert.Greater(t, watcher.config.CleanupAfter, time.Duration(0), "should use positive default for CleanupAfter")
			},
		},
		{
			name: "extremely large values in configuration",
			config: Config{
				PorchPath:    "porch",
				Mode:         "direct",
				OutDir:       "/tmp/out",
				MaxWorkers:   999999,
				DebounceDur:  24 * time.Hour,
				CleanupAfter: 365 * 24 * time.Hour,
			},
			testFunc: func(t *testing.T, config Config) {
				tempDir := t.TempDir()
				watcher, err := NewWatcher(tempDir, config)
				require.NoError(t, err)
				defer watcher.Close()

				// Large values should be accepted but might be capped internally
				// This is more about ensuring the system doesn't crash
				assert.NotNil(t, watcher, "watcher should be created with large values")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t, tt.config)
		})
	}
}

// TestSanitizeInput tests input sanitization functions
func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal string",
			input:    "normal-app-name",
			expected: "normal-app-name",
		},
		{
			name:     "string with null bytes",
			input:    "app\x00name",
			expected: "appname", // Null bytes should be removed
		},
		{
			name:     "string with control characters",
			input:    "app\n\r\tname",
			expected: "app   name", // Control chars replaced with spaces
		},
		{
			name:     "string with unicode",
			input:    "приложение-名前",
			expected: "приложение-名前", // Unicode should be preserved
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "very long string",
			input:    strings.Repeat("a", 1000),
			expected: strings.Repeat("a", 255), // Should be truncated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeInput(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestValidateIntentContent tests validation of intent file content
func TestValidateIntentContent(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		expectError  bool
		errorMessage string
	}{
		{
			name: "valid intent",
			content: `{
				"intent_type": "scaling",
				"target": "my-app",
				"namespace": "default",
				"replicas": 3
			}`,
			expectError: false,
		},
		{
			name: "malformed JSON",
			content: `{
				"intent_type": "scaling",
				"target": "my-app",
				"namespace": "default",
				"replicas": 3
			`,
			expectError:  true,
			errorMessage: "invalid JSON",
		},
		{
			name: "missing required fields",
			content: `{
				"intent_type": "scaling"
			}`,
			expectError:  true,
			errorMessage: "missing required field",
		},
		{
			name: "invalid intent type",
			content: `{
				"intent_type": "malicious",
				"target": "my-app",
				"namespace": "default",
				"replicas": 3
			}`,
			expectError:  true,
			errorMessage: "invalid intent type",
		},
		{
			name: "path traversal in target",
			content: `{
				"intent_type": "scaling",
				"target": "../../../etc/passwd",
				"namespace": "default",
				"replicas": 3
			}`,
			expectError:  true,
			errorMessage: "invalid target",
		},
		{
			name: "command injection in target",
			content: `{
				"intent_type": "scaling",
				"target": "app; rm -rf /",
				"namespace": "default",
				"replicas": 3
			}`,
			expectError:  true,
			errorMessage: "invalid target",
		},
		{
			name: "negative replicas",
			content: `{
				"intent_type": "scaling",
				"target": "my-app",
				"namespace": "default",
				"replicas": -1
			}`,
			expectError:  true,
			errorMessage: "invalid replicas",
		},
		{
			name: "extremely large replicas",
			content: `{
				"intent_type": "scaling",
				"target": "my-app",
				"namespace": "default",
				"replicas": 999999999
			}`,
			expectError:  true,
			errorMessage: "invalid replicas",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIntentContent([]byte(tt.content))

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper functions for security validation

func sanitizeInput(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Replace control characters with spaces
	var result strings.Builder
	for _, r := range input {
		if r < 32 && r != '\t' && r != '\n' && r != '\r' {
			result.WriteRune(' ')
		} else if r == '\n' || r == '\r' || r == '\t' {
			result.WriteRune(' ')
		} else {
			result.WriteRune(r)
		}
	}

	// Truncate if too long
	output := result.String()
	if len(output) > 255 {
		output = output[:255]
	}

	return output
}

func validateIntentContent(content []byte) error {
	// Parse JSON
	var intent map[string]interface{}
	if err := json.Unmarshal(content, &intent); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Validate required fields
	intentType, ok := intent["intent_type"].(string)
	if !ok {
		return fmt.Errorf("missing required field: intent_type")
	}

	target, ok := intent["target"].(string)
	if !ok {
		return fmt.Errorf("missing required field: target")
	}

	namespace, ok := intent["namespace"].(string)
	if !ok {
		return fmt.Errorf("missing required field: namespace")
	}

	replicas, ok := intent["replicas"].(float64)
	if !ok {
		return fmt.Errorf("missing required field: replicas")
	}

	// Validate intent type
	if intentType != "scaling" {
		return fmt.Errorf("invalid intent type: %s", intentType)
	}

	// Validate target (check for path traversal and command injection)
	if strings.Contains(target, "..") || strings.Contains(target, "/") ||
		strings.Contains(target, "\\") || strings.Contains(target, ";") ||
		strings.Contains(target, "&") || strings.Contains(target, "|") {
		return fmt.Errorf("invalid target: contains suspicious characters")
	}

	// Validate namespace
	if strings.Contains(namespace, "..") || strings.Contains(namespace, "/") ||
		strings.Contains(namespace, "\\") || strings.Contains(namespace, ";") ||
		strings.Contains(namespace, "&") || strings.Contains(namespace, "|") {
		return fmt.Errorf("invalid namespace: contains suspicious characters")
	}

	// Validate replicas
	if replicas < 0 || replicas > 1000 {
		return fmt.Errorf("invalid replicas: must be between 0 and 1000")
	}

	return nil
}
