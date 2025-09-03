package loop

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDirectoryErrorHandlingScenarios tests comprehensive error handling
// for directory creation failures in various scenarios
func TestDirectoryErrorHandlingScenarios(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("StateManager handles directory creation errors gracefully", func(t *testing.T) {
		// Test StateManager with various invalid directory scenarios
		testCases := []struct {
			name        string
			baseDir     string
			expectError bool
			errorSubstr string
		}{
			{
				name:        "empty base directory",
				baseDir:     "",
				expectError: false, // Current behavior allows empty dir
				errorSubstr: "",
			},
			{
				name:        "null byte in path",
				baseDir:     tempDir + "\x00invalid",
				expectError: true,
				errorSubstr: "invalid argument",
			},
			{
				name:        "very long path",
				baseDir:     tempDir + "/" + strings.Repeat("verylongdirectoryname", 50),
				expectError: true, // Should fail on Windows due to path length
				errorSubstr: "",   // Error message varies by platform
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				sm, err := NewStateManager(tc.baseDir)

				if tc.expectError {
					if err != nil {
						t.Logf("StateManager correctly rejected invalid base directory %q: %v", tc.baseDir, err)
						if tc.errorSubstr != "" {
							assert.Contains(t, err.Error(), tc.errorSubstr)
						}
					} else {
						// If StateManager was created, try to trigger a state save to see if it fails
						if sm != nil {
							// Create a test file first
							testFile := filepath.Join(tc.baseDir, "test.json")
							if err := os.WriteFile(testFile, []byte(`{"test": true}`), 0o644); err == nil {
								err = sm.MarkProcessed("test.json")
								assert.Error(t, err, "MarkProcessed should fail with invalid directory")
								if tc.errorSubstr != "" {
									assert.Contains(t, err.Error(), tc.errorSubstr)
								}
							}
							sm.Close()
						}
					}
				} else {
					assert.NoError(t, err, "StateManager should handle %q gracefully", tc.baseDir)
					if sm != nil {
						sm.Close()
					}
				}
			})
		}
	})

	t.Run("Watcher handles status directory creation errors", func(t *testing.T) {
		// Test watcher status file creation with problematic directories
		testCases := []struct {
			name        string
			setupDir    func() string
			expectError bool
			errorSubstr string
		}{
			{
				name: "normal nested directory",
				setupDir: func() string {
					dir := filepath.Join(tempDir, "normal", "nested")
					os.MkdirAll(dir, 0o755)
					return dir
				},
				expectError: false,
			},
			{
				name: "directory with invalid characters",
				setupDir: func() string {
					// Create a directory with special characters that might cause issues
					dir := filepath.Join(tempDir, "special-chars!@#$%")
					os.MkdirAll(dir, 0o755)
					return dir
				},
				expectError: false, // Should be handled gracefully
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				watchDir := tc.setupDir()

				config := Config{
					PorchPath: "/tmp/test-porch",
					Mode:      "once",
				}

				watcher, err := NewWatcher(watchDir, config)
				if tc.expectError {
					assert.Error(t, err, "Watcher creation should fail")
					if tc.errorSubstr != "" {
						assert.Contains(t, err.Error(), tc.errorSubstr)
					}
				} else {
					require.NoError(t, err, "Watcher creation should succeed")
					defer watcher.Close() // #nosec G307 - Error handled in defer

					// Try to write a status file
					intentFile := filepath.Join(watchDir, "test-intent.json")
					intentContent := `{"api_version": "intent.nephoran.io/v1", "kind": "ScaleIntent", "metadata": {"name": "test"}, "spec": {"replicas": 3}}`
					require.NoError(t, os.WriteFile(intentFile, []byte(intentContent), 0o644))

					// This should not panic or fail silently
					watcher.writeStatusFileAtomic(intentFile, "success", "Test processing completed")

					// Verify status directory was created
					statusDir := filepath.Join(watchDir, "status")
					_, err = os.Stat(statusDir)
					assert.NoError(t, err, "Status directory should be created")
				}
			})
		}
	})

	t.Run("atomicWriteFile handles edge cases", func(t *testing.T) {
		testCases := []struct {
			name        string
			filePath    string
			data        []byte
			expectError bool
			errorSubstr string
		}{
			{
				name:        "empty filename",
				filePath:    "",
				data:        []byte("test"),
				expectError: true,
				errorSubstr: "",
			},
			{
				name:        "null byte in filename",
				filePath:    tempDir + "/test\x00file.txt",
				data:        []byte("test"),
				expectError: true,
				errorSubstr: "invalid argument",
			},
			{
				name:        "very deep nesting",
				filePath:    filepath.Join(tempDir, strings.Repeat("nested/", 100), "test.txt"),
				data:        []byte("test"),
				expectError: false, // Should create all necessary directories
			},
			{
				name:        "large data",
				filePath:    filepath.Join(tempDir, "large-file.txt"),
				data:        make([]byte, 10*1024*1024), // 10MB
				expectError: false,                      // Should handle large files
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := atomicWriteFile(tc.filePath, tc.data, 0o644)

				if tc.expectError {
					assert.Error(t, err, "atomicWriteFile should fail for %q", tc.filePath)
					if tc.errorSubstr != "" {
						assert.Contains(t, err.Error(), tc.errorSubstr)
					}
				} else {
					assert.NoError(t, err, "atomicWriteFile should succeed for %q", tc.filePath)

					// Verify file was created and has correct content
					if err == nil {
						content, readErr := os.ReadFile(tc.filePath)
						assert.NoError(t, readErr, "Should be able to read written file")
						assert.Equal(t, tc.data, content, "File content should match")
					}
				}
			})
		}
	})

	t.Run("Error messages are informative", func(t *testing.T) {
		// Test that error messages provide sufficient information for debugging

		// Try to create a state manager with an invalid path and check error message quality
		invalidPath := tempDir + string([]byte{0, 1, 2}) + "/invalid"
		sm, err := NewStateManager(invalidPath)

		if err != nil {
			t.Logf("Error message: %v", err)
			// Error messages should be descriptive
			errorStr := err.Error()
			assert.True(t, len(errorStr) > 10, "Error message should be descriptive")
			assert.False(t, strings.Contains(errorStr, "unknown error"), "Error should not be generic")
		} else if sm != nil {
			// If creation succeeded, try an operation that should fail
			err = sm.MarkProcessed("nonexistent.json")
			if err != nil {
				t.Logf("MarkProcessed error message: %v", err)
				errorStr := err.Error()
				assert.True(t, len(errorStr) > 10, "Error message should be descriptive")
			}
			sm.Close()
		}

		// Test atomicWriteFile error messages
		err = atomicWriteFile("/dev/null/impossible/path.txt", []byte("test"), 0o644)
		if err != nil {
			t.Logf("atomicWriteFile error message: %v", err)
			errorStr := err.Error()
			assert.True(t, len(errorStr) > 10, "Error message should be descriptive")
			assert.True(t, strings.Contains(errorStr, "directory") ||
				strings.Contains(errorStr, "path") ||
				strings.Contains(errorStr, "permission"),
				"Error should mention the type of problem")
		}
	})

	t.Run("Cleanup on failure", func(t *testing.T) {
		// Test that temporary files are cleaned up when operations fail
		tempPath := filepath.Join(tempDir, "cleanup-test.txt")

		// Create a scenario where the rename might fail
		// First create the target file as read-only (if possible)
		if err := os.WriteFile(tempPath, []byte("existing"), 0o444); err == nil {
			// Change to read-only
			if runtime.GOOS != "windows" {
				os.Chmod(tempPath, 0o444)
			}

			// Try to atomic write - this might fail on the rename step
			err = atomicWriteFile(tempPath, []byte("new content"), 0o644)

			// Check that no .tmp files are left behind
			tmpPath := tempPath + ".tmp"
			_, tmpExists := os.Stat(tmpPath)
			assert.True(t, os.IsNotExist(tmpExists), "Temporary file should be cleaned up on failure")

			// Restore permissions for cleanup
			if runtime.GOOS != "windows" {
				os.Chmod(tempPath, 0o644)
			}
		}
	})
}

// TestRobustDirectoryCreation tests that directory creation is robust under various conditions
func TestRobustDirectoryCreation(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("Concurrent directory creation is safe", func(t *testing.T) {
		// Test that multiple goroutines can safely create the same directory structure
		sharedDir := filepath.Join(tempDir, "shared", "deeply", "nested")

		done := make(chan error, 10)

		// Launch multiple goroutines trying to create files in the same directory
		for i := 0; i < 10; i++ {
			go func(id int) {
				filePath := filepath.Join(sharedDir, fmt.Sprintf("file-%d.txt", id))
				err := atomicWriteFile(filePath, []byte(fmt.Sprintf("content-%d", id)), 0o644)
				done <- err
			}(i)
		}

		// Wait for all to complete
		for i := 0; i < 10; i++ {
			err := <-done
			assert.NoError(t, err, "Concurrent directory creation should not fail")
		}

		// Verify all files were created
		entries, err := os.ReadDir(sharedDir)
		require.NoError(t, err)
		assert.Equal(t, 10, len(entries), "All files should be created")
	})

	t.Run("Directory creation with existing file", func(t *testing.T) {
		// Test what happens when we try to create a directory where a file exists
		conflictPath := filepath.Join(tempDir, "conflict")

		// Create a file at the path where we want a directory
		require.NoError(t, os.WriteFile(conflictPath, []byte("I'm a file"), 0o644))

		// Try to create a file in a subdirectory - this should fail
		subFile := filepath.Join(conflictPath, "subfile.txt")
		err := atomicWriteFile(subFile, []byte("test"), 0o644)
		assert.Error(t, err, "Should fail when directory path conflicts with existing file")
	})

	t.Run("Deep directory nesting", func(t *testing.T) {
		// Test creating very deeply nested directories
		deepPath := tempDir
		for i := 0; i < 50; i++ {
			deepPath = filepath.Join(deepPath, fmt.Sprintf("level-%d", i))
		}
		deepFile := filepath.Join(deepPath, "deep-file.txt")

		err := atomicWriteFile(deepFile, []byte("deep content"), 0o644)
		if err != nil {
			// On some systems, very deep paths might fail - that's OK
			t.Logf("Deep path creation failed (expected on some systems): %v", err)
		} else {
			// If it succeeded, verify the file exists
			content, err := os.ReadFile(deepFile)
			assert.NoError(t, err)
			assert.Equal(t, []byte("deep content"), content)
		}
	})
}
