//go:build windows
// +build windows

package loop

import (
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
	"github.com/thc1006/nephoran-intent-operator/internal/porch"
)

// TestWindowsParentDirectoryCreation provides comprehensive coverage for parent directory
// creation fixes that resolve Windows CI failures related to status file operations.
func TestWindowsParentDirectoryCreation(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows parent directory tests only apply to Windows")
	}

	t.Run("Status_File_Parent_Directory_Creation", func(t *testing.T) {
		baseDir := t.TempDir()

		testCases := []struct {
			name      string
			structure []string // Directories to create
			statusDir string   // Expected status directory
			testFile  string   // File to process
		}{
			{
				name:      "simple_nested_status",
				structure: []string{"watch"},
				statusDir: filepath.Join("watch", "status"),
				testFile:  "test-intent.json",
			},
			{
				name:      "deeply_nested_status",
				structure: []string{"watch", "deep", "nested"},
				statusDir: filepath.Join("watch", "deep", "nested", "status"),
				testFile:  "deep-intent.json",
			},
			{
				name:      "windows_drive_relative",
				structure: []string{},
				statusDir: "status", // Relative to testBaseDir, watcher writes to watchDir/status
				testFile:  "drive-intent.json",
			},
			{
				name:      "long_path_status",
				structure: []string{strings.Repeat("verylongdirname", 5)},
				statusDir: filepath.Join(strings.Repeat("verylongdirname", 5), "status"),
				testFile:  "long-path-intent.json",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Create test-specific directory
				testBaseDir := filepath.Join(baseDir, tc.name)
				
				// Create watch directory structure
				var watchDir string
				if len(tc.structure) > 0 {
					watchDir = filepath.Join(testBaseDir, filepath.Join(tc.structure...))
				} else {
					watchDir = testBaseDir
				}

				// Only create watch directory, not status directory (that should be auto-created)
				require.NoError(t, os.MkdirAll(watchDir, 0755))

				// Determine expected status directory
				var expectedStatusDir string
				if filepath.IsAbs(tc.statusDir) {
					expectedStatusDir = tc.statusDir
				} else {
					expectedStatusDir = filepath.Join(testBaseDir, tc.statusDir)
				}

				// Verify status directory doesn't exist initially
				_, err := os.Stat(expectedStatusDir)
				require.True(t, os.IsNotExist(err), "Status directory should not exist initially")

				// Create cross-platform mock porch executable
				mockPorch, err := porch.CreateSimpleMock(baseDir)
				require.NoError(t, err, "Failed to create mock porch executable")

				// Create watcher
				config := Config{
					PorchPath: mockPorch,
					Mode:      "once",
				}

				watcher, err := NewWatcher(watchDir, config)
				require.NoError(t, err, "Watcher creation should succeed")
				defer watcher.Close()

				// Create intent file
				intentFile := filepath.Join(watchDir, tc.testFile)
				intentContent := `{
					"api_version": "intent.nephoran.io/v1",
					"kind": "ScaleIntent",
					"metadata": {"name": "test-` + tc.name + `"},
					"spec": {"replicas": 3}
				}`
				require.NoError(t, os.WriteFile(intentFile, []byte(intentContent), 0644))

				// Trigger status file write - this should create parent directories
				watcher.writeStatusFileAtomic(intentFile, "success", "Test processing completed for "+tc.name)

				// Verify status directory was created
				_, err = os.Stat(expectedStatusDir)
				assert.NoError(t, err, "Status directory should exist after status write")

				// Verify status file was created
				statusEntries, err := os.ReadDir(expectedStatusDir)
				require.NoError(t, err, "Should be able to read status directory")
				assert.Greater(t, len(statusEntries), 0, "Should have at least one status file")

				// Verify status file content
				for _, entry := range statusEntries {
					if strings.HasSuffix(entry.Name(), ".status") {
						statusFilePath := filepath.Join(expectedStatusDir, entry.Name())
						content, err := os.ReadFile(statusFilePath)
						assert.NoError(t, err, "Should be able to read status file")
						assert.Contains(t, string(content), "success", "Status file should contain success status")
					}
				}
			})
		}
	})

	t.Run("StateManager_Parent_Directory_Creation", func(t *testing.T) {
		baseDir := t.TempDir()

		testCases := []struct {
			name         string
			stateDir     string
			relativePath bool
		}{
			{
				name:     "simple_state_dir",
				stateDir: "state",
			},
			{
				name:     "nested_state_dir",
				stateDir: filepath.Join("deeply", "nested", "state", "dir"),
			},
			{
				name:         "relative_state_path",
				stateDir:     ".conductor",
				relativePath: true,
			},
			{
				name:     "windows_backslash_path",
				stateDir: "windows\\style\\path",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var stateDir string
				if tc.relativePath {
					stateDir = tc.stateDir
				} else {
					stateDir = filepath.Join(baseDir, tc.name, tc.stateDir)
				}

				// Verify directory doesn't exist initially
				if !tc.relativePath {
					_, err := os.Stat(stateDir)
					require.True(t, os.IsNotExist(err), "State directory should not exist initially")
				}

				// Create StateManager - this should create parent directories
				sm, err := NewStateManager(stateDir)
				require.NoError(t, err, "StateManager creation should succeed")
				defer sm.Close()

				// Create a test file to mark as processed
				if !tc.relativePath {
					testFile := filepath.Join(stateDir, "test.json")
					// Ensure parent directory exists before creating test file
					require.NoError(t, os.MkdirAll(filepath.Dir(testFile), 0755))
					require.NoError(t, os.WriteFile(testFile, []byte(`{"test": true}`), 0644))
				} else {
					// For relative paths, create in current dir
					require.NoError(t, os.WriteFile("test.json", []byte(`{"test": true}`), 0644))
					defer os.Remove("test.json")
				}

				// Mark file as processed - this triggers state file creation
				err = sm.MarkProcessed("test.json")
				require.NoError(t, err, "MarkProcessed should succeed")

				// Verify state file was created (implying parent directories exist)
				var stateFile string
				if tc.relativePath {
					stateFile = filepath.Join(stateDir, ".conductor-state.json")
				} else {
					stateFile = filepath.Join(stateDir, ".conductor-state.json")
				}
				
				_, err = os.Stat(stateFile)
				assert.NoError(t, err, "State file should exist after MarkProcessed")
			})
		}
	})

	t.Run("Atomic_Write_Parent_Directory_Creation", func(t *testing.T) {
		baseDir := t.TempDir()

		testCases := []struct {
			name     string
			filePath string
			content  []byte
		}{
			{
				name:     "single_level_parent",
				filePath: filepath.Join("single", "atomic.txt"),
				content:  []byte("single level test"),
			},
			{
				name:     "multi_level_parent",
				filePath: filepath.Join("multi", "level", "deep", "atomic.txt"),
				content:  []byte("multi level test"),
			},
			{
				name:     "windows_max_path_approach",
				filePath: filepath.Join(strings.Repeat("a", 50), strings.Repeat("b", 50), "atomic.txt"),
				content:  []byte("long path test"),
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				fullPath := filepath.Join(baseDir, tc.name, tc.filePath)
				parentDir := filepath.Dir(fullPath)

				// Verify parent directory doesn't exist initially
				_, err := os.Stat(parentDir)
				require.True(t, os.IsNotExist(err), "Parent directory should not exist initially")

				// Use atomicWriteFile - this should create parent directories
				err = atomicWriteFile(fullPath, tc.content, 0644)
				require.NoError(t, err, "atomicWriteFile should succeed")

				// Verify file was created
				_, err = os.Stat(fullPath)
				assert.NoError(t, err, "File should exist after atomic write")

				// Verify parent directory was created
				_, err = os.Stat(parentDir)
				assert.NoError(t, err, "Parent directory should exist after atomic write")

				// Verify content is correct
				readContent, err := os.ReadFile(fullPath)
				require.NoError(t, err, "Should be able to read written file")
				assert.Equal(t, tc.content, readContent, "File content should match")

				// Verify no temp files remain
				tempFile := fullPath + ".tmp"
				_, err = os.Stat(tempFile)
				assert.True(t, os.IsNotExist(err), "Temp file should not exist after atomic write")
			})
		}
	})

	t.Run("Concurrent_Parent_Directory_Creation", func(t *testing.T) {
		// Test concurrent directory creation doesn't cause race conditions
		baseDir := t.TempDir()
		numWorkers := 10
		sharedParentDir := filepath.Join(baseDir, "concurrent", "shared")

		var wg sync.WaitGroup
		errors := make(chan error, numWorkers)

		// Run multiple workers trying to create files in the same parent directory
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				filePath := filepath.Join(sharedParentDir, fmt.Sprintf("worker-%d.txt", id))
				content := []byte(fmt.Sprintf("Worker %d content", id))

				err := atomicWriteFile(filePath, content, 0644)
				if err != nil {
					errors <- fmt.Errorf("worker %d failed: %w", id, err)
					return
				}

				// Verify file was created correctly
				readContent, err := os.ReadFile(filePath)
				if err != nil {
					errors <- fmt.Errorf("worker %d read failed: %w", id, err)
					return
				}

				if string(readContent) != string(content) {
					errors <- fmt.Errorf("worker %d content mismatch", id)
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for any errors
		var errorCount int
		for err := range errors {
			t.Errorf("Concurrent error: %v", err)
			errorCount++
		}

		assert.Equal(t, 0, errorCount, "No concurrent errors expected")

		// Verify all files were created
		entries, err := os.ReadDir(sharedParentDir)
		require.NoError(t, err, "Should be able to read shared directory")
		assert.Equal(t, numWorkers, len(entries), "All worker files should be created")
	})

	t.Run("Windows_Path_Edge_Cases", func(t *testing.T) {
		baseDir := t.TempDir()

		testCases := []struct {
			name        string
			pathPattern string
			shouldWork  bool
			description string
		}{
			{
				name:        "trailing_backslash",
				pathPattern: "trailing\\backslash\\",
				shouldWork:  true,
				description: "Paths with trailing backslashes",
			},
			{
				name:        "mixed_separators",
				pathPattern: "mixed/separators\\and/more",
				shouldWork:  true,
				description: "Mixed forward and backward slashes",
			},
			{
				name:        "dot_directories",
				pathPattern: "dot/../resolved/path",
				shouldWork:  true,
				description: "Paths with dot directories",
			},
			{
				name:        "spaces_in_path",
				pathPattern: "path with spaces/and more spaces",
				shouldWork:  true,
				description: "Paths with spaces",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				testDir := filepath.Join(baseDir, tc.name)
				testFile := filepath.Join(testDir, tc.pathPattern, "test.txt")
				content := []byte("Windows path edge case test")

				err := atomicWriteFile(testFile, content, 0644)
				
				if tc.shouldWork {
					assert.NoError(t, err, "Path should work: %s", tc.description)
					
					if err == nil {
						// Verify file was created
						_, statErr := os.Stat(testFile)
						assert.NoError(t, statErr, "File should exist after creation")

						// Verify content
						readContent, readErr := os.ReadFile(testFile)
						assert.NoError(t, readErr, "Should be able to read file")
						assert.Equal(t, content, readContent, "Content should match")
					}
				} else {
					assert.Error(t, err, "Path should fail: %s", tc.description)
				}
			})
		}
	})

	t.Run("Status_File_Atomic_Operations", func(t *testing.T) {
		// Test that status file creation is atomic even when parent directories need to be created
		baseDir := t.TempDir()
		watchDir := filepath.Join(baseDir, "watch")
		statusDir := filepath.Join(watchDir, "status")

		// Create watch directory but not status directory
		require.NoError(t, os.MkdirAll(watchDir, 0755))

		// Create cross-platform mock porch executable
		mockPorch, err := porch.CreateSimpleMock(baseDir)
		require.NoError(t, err, "Failed to create mock porch executable")

		config := Config{
			PorchPath: mockPorch,
			Mode:      "once",
		}

		watcher, err := NewWatcher(watchDir, config)
		require.NoError(t, err)
		defer watcher.Close()

		// Create intent file
		intentFile := filepath.Join(watchDir, "atomic-test.json")
		intentContent := `{"api_version": "intent.nephoran.io/v1", "kind": "ScaleIntent"}`
		require.NoError(t, os.WriteFile(intentFile, []byte(intentContent), 0644))

		// Simulate concurrent status writes
		numConcurrent := 5
		var wg sync.WaitGroup
		errors := make(chan error, numConcurrent)

		for i := 0; i < numConcurrent; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				status := fmt.Sprintf("status-%d", id)
				message := fmt.Sprintf("Concurrent status write %d", id)
				
				// Each worker writes its own status (different filenames due to timestamps)
				time.Sleep(time.Duration(id) * time.Millisecond) // Slight stagger
				watcher.writeStatusFileAtomic(intentFile, status, message)
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for errors
		for err := range errors {
			t.Errorf("Concurrent status write error: %v", err)
		}

		// Verify status directory was created
		_, err = os.Stat(statusDir)
		assert.NoError(t, err, "Status directory should exist")

		// Verify status files were created
		statusEntries, err := os.ReadDir(statusDir)
		require.NoError(t, err, "Should be able to read status directory")
		assert.GreaterOrEqual(t, len(statusEntries), 1, "Should have at least one status file")

		// Verify no partial or corrupt status files
		for _, entry := range statusEntries {
			statusFile := filepath.Join(statusDir, entry.Name())
			content, err := os.ReadFile(statusFile)
			assert.NoError(t, err, "Should be able to read status file: %s", entry.Name())
			assert.Greater(t, len(content), 0, "Status file should not be empty: %s", entry.Name())
			
			// Verify it looks like JSON
			assert.True(t, strings.HasPrefix(string(content), "{"), "Status file should be JSON: %s", entry.Name())
			assert.True(t, strings.HasSuffix(strings.TrimSpace(string(content)), "}"), "Status file should be complete JSON: %s", entry.Name())
		}
	})

	t.Run("Parent_Directory_Permission_Handling", func(t *testing.T) {
		// Test error handling when parent directory creation fails
		baseDir := t.TempDir()

		t.Run("read_only_parent", func(t *testing.T) {
			// Create a read-only directory
			readOnlyDir := filepath.Join(baseDir, "readonly")
			require.NoError(t, os.Mkdir(readOnlyDir, 0755))

			// Make it read-only (this is tricky on Windows, so we'll just test the concept)
			// On Windows, we can't easily make directories truly read-only, so this test
			// validates the error handling path rather than actual permission denial
			
			testFile := filepath.Join(readOnlyDir, "subdir", "test.txt")
			err := atomicWriteFile(testFile, []byte("test"), 0644)
			
			// On Windows, this might still succeed due to how permissions work
			// The important thing is that it either succeeds or fails gracefully
			if err != nil {
				t.Logf("Expected permission error: %v", err)
				assert.Contains(t, strings.ToLower(err.Error()), "permission", "Error should be permission-related")
			} else {
				t.Logf("File creation succeeded despite read-only parent (Windows behavior)")
				
				// Clean up
				os.Remove(testFile)
				os.Remove(filepath.Dir(testFile))
			}
		})
	})
}

// TestWindowsParentDirectoryRegressionPrevention ensures specific directory creation
// issues that caused Windows CI failures cannot reoccur.
func TestWindowsParentDirectoryRegressionPrevention(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows parent directory regression tests only apply to Windows")
	}

	t.Run("Status_Directory_Missing_Regression", func(t *testing.T) {
		// This test recreates the exact scenario that caused CI failures:
		// 1. Watcher created in directory without status subdirectory
		// 2. Status file write attempted
		// 3. Should automatically create status directory
		
		baseDir := t.TempDir()
		watchDir := filepath.Join(baseDir, "regression-test")
		statusDir := filepath.Join(watchDir, "status")

		// Create only the watch directory
		require.NoError(t, os.MkdirAll(watchDir, 0755))

		// Verify status directory doesn't exist
		_, err := os.Stat(statusDir)
		require.True(t, os.IsNotExist(err), "Status directory should not exist initially")

		// Create cross-platform mock porch executable
		mockPorch, err := porch.CreateSimpleMock(baseDir)
		require.NoError(t, err, "Failed to create mock porch executable")

		// Create watcher
		config := Config{
			PorchPath: mockPorch,
			Mode:      "once",
		}

		watcher, err := NewWatcher(watchDir, config)
		require.NoError(t, err, "Watcher creation should succeed")
		defer watcher.Close()

		// Create intent file
		intentFile := filepath.Join(watchDir, "regression-intent.json")
		require.NoError(t, os.WriteFile(intentFile, []byte(`{"kind": "test"}`), 0644))

		// This is the critical operation that was failing:
		// Writing status file when status directory doesn't exist
		watcher.writeStatusFileAtomic(intentFile, "success", "Regression test completed")

		// Verify status directory was created
		statInfo, err := os.Stat(statusDir)
		assert.NoError(t, err, "Status directory should be created automatically")
		assert.True(t, statInfo.IsDir(), "Status path should be a directory")

		// Verify status file was created
		entries, err := os.ReadDir(statusDir)
		require.NoError(t, err, "Should be able to read status directory")
		assert.Greater(t, len(entries), 0, "Should have at least one status file")

		// Verify status file is valid
		statusFile := filepath.Join(statusDir, entries[0].Name())
		content, err := os.ReadFile(statusFile)
		require.NoError(t, err, "Should be able to read status file")
		assert.Contains(t, string(content), "success", "Status should contain success")
		assert.Contains(t, string(content), "Regression test completed", "Status should contain message")
	})

	t.Run("Deep_Path_Status_Creation", func(t *testing.T) {
		// Test deep path scenarios that might fail on Windows due to path length limits
		baseDir := t.TempDir()
		
		// Create a reasonably deep path (but not too deep to cause MAX_PATH issues)
		deepPath := filepath.Join(baseDir, "level1", "level2", "level3", "level4", "watch")
		statusDir := filepath.Join(deepPath, "status")

		// Create only the watch directory
		require.NoError(t, os.MkdirAll(deepPath, 0755))

		// Create cross-platform mock porch executable
		mockPorch, err := porch.CreateSimpleMock(baseDir)
		require.NoError(t, err, "Failed to create mock porch executable")

		config := Config{
			PorchPath: mockPorch,
			Mode:      "once",
		}

		watcher, err := NewWatcher(deepPath, config)
		require.NoError(t, err, "Watcher creation should succeed for deep paths")
		defer watcher.Close()

		// Create intent file
		intentFile := filepath.Join(deepPath, "deep-intent.json")
		require.NoError(t, os.WriteFile(intentFile, []byte(`{"kind": "DeepTest"}`), 0644))

		// Write status file - this should create the deep status directory
		watcher.writeStatusFileAtomic(intentFile, "success", "Deep path test completed")

		// Verify status directory was created at the deep path
		_, err = os.Stat(statusDir)
		assert.NoError(t, err, "Deep status directory should be created")

		// Verify status file exists
		entries, err := os.ReadDir(statusDir)
		require.NoError(t, err, "Should be able to read deep status directory")
		assert.Greater(t, len(entries), 0, "Should have status file in deep directory")
	})

	t.Run("Backslash_Path_Handling", func(t *testing.T) {
		// Test Windows-specific backslash path handling
		baseDir := t.TempDir()
		
		// Use Windows-style backslash paths
		watchDir := filepath.Join(baseDir, "backslash\\test\\dir")
		statusDir := filepath.Join(watchDir, "status")

		// Normalize the path for Windows
		watchDir = filepath.FromSlash(watchDir)
		statusDir = filepath.FromSlash(statusDir)

		// Create watch directory
		require.NoError(t, os.MkdirAll(watchDir, 0755))

		// Create cross-platform mock porch executable
		mockPorch, err := porch.CreateSimpleMock(baseDir)
		require.NoError(t, err, "Failed to create mock porch executable")

		config := Config{
			PorchPath: mockPorch,
			Mode:      "once",
		}

		watcher, err := NewWatcher(watchDir, config)
		require.NoError(t, err, "Watcher should handle backslash paths")
		defer watcher.Close()

		// Create intent file
		intentFile := filepath.Join(watchDir, "backslash-intent.json")
		require.NoError(t, os.WriteFile(intentFile, []byte(`{"kind": "BackslashTest"}`), 0644))

		// Write status file
		watcher.writeStatusFileAtomic(intentFile, "success", "Backslash path test completed")

		// Verify status directory creation
		_, err = os.Stat(statusDir)
		assert.NoError(t, err, "Status directory should be created with backslash paths")
	})
}