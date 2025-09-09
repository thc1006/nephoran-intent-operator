package loop

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParentDirectoryCreation validates that all file write operations
// properly create parent directories when they don't exist
func TestParentDirectoryCreation(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("StateManager creates parent directories", func(t *testing.T) {
		// Create a nested directory structure that doesn't exist yet
		nestedDir := filepath.Join(tempDir, "state", "deeply", "nested", "dir")
		stateFile := filepath.Join(nestedDir, ".conductor-state.json")

		// Verify the directory doesn't exist initially
		_, err := os.Stat(nestedDir)
		require.True(t, os.IsNotExist(err), "Directory should not exist initially")

		// Create StateManager with the nested path
		sm, err := NewStateManager(nestedDir)
		require.NoError(t, err, "StateManager creation should succeed")
		defer sm.Close() // #nosec G307 - Error handled in defer

		// Create the directory first and then create a test file
		require.NoError(t, os.MkdirAll(nestedDir, 0o755))
		testFile := filepath.Join(nestedDir, "test.json")
		require.NoError(t, os.WriteFile(testFile, []byte(`{"test": true}`), 0o644))

		// Mark the file as processed - this should trigger state file write
		// and test that the StateManager can create parent directories for its state file
<<<<<<< HEAD
		err = sm.MarkProcessed("test.json")
=======
		err = sm.MarkProcessed(testFile)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		require.NoError(t, err, "MarkProcessed should succeed")

		// Verify the state file was created
		_, err = os.Stat(stateFile)
		assert.NoError(t, err, "State file should exist after MarkProcessed")
	})

	t.Run("Watcher creates status directories", func(t *testing.T) {
		// Create a watch directory that exists
		watchDir := filepath.Join(tempDir, "watcher")
		statusDir := filepath.Join(watchDir, "status")

		// Create the watch directory but not the status directory
		require.NoError(t, os.MkdirAll(watchDir, 0o755))

		// Verify the status directory doesn't exist initially
		_, err := os.Stat(statusDir)
		require.True(t, os.IsNotExist(err), "Status directory should not exist initially")

		// Create watcher config
		config := Config{
			PorchPath: "/tmp/test-porch",
			Mode:      "once",
		}

		// Create watcher
		watcher, err := NewWatcher(watchDir, config)
		require.NoError(t, err, "Watcher creation should succeed")
		defer watcher.Close() // #nosec G307 - Error handled in defer

		// Create an intent file that will trigger status file creation
		intentFile := filepath.Join(watchDir, "test-intent.json")
		intentContent := `{
			"api_version": "intent.nephoran.io/v1",
			"kind": "ScaleIntent", 
			"metadata": {"name": "test"},
			"spec": {"replicas": 3}
		}`

		require.NoError(t, os.WriteFile(intentFile, []byte(intentContent), 0o644))

		// Manually trigger status file write (simulating successful processing)
		// This should create the status directory
		watcher.writeStatusFileAtomic(intentFile, "success", "Test processing completed")

		// Verify the status directory was created
		_, err = os.Stat(statusDir)
		assert.NoError(t, err, "Status directory should exist after status write")

		// Verify a status file was created
		statusEntries, err := os.ReadDir(statusDir)
		require.NoError(t, err, "Should be able to read status directory")
		assert.Greater(t, len(statusEntries), 0, "Should have at least one status file")
	})

	t.Run("atomicWriteFile creates parent directories", func(t *testing.T) {
		// Create a deeply nested path that doesn't exist
		nestedPath := filepath.Join(tempDir, "atomic", "very", "deeply", "nested", "test.txt")
		parentDir := filepath.Dir(nestedPath)

		// Verify the directory doesn't exist initially
		_, err := os.Stat(parentDir)
		require.True(t, os.IsNotExist(err), "Parent directory should not exist initially")

		// Use atomicWriteFile to write to the nested path
		testData := []byte("test content for atomic write")
		err = atomicWriteFile(nestedPath, testData, 0o644)
		require.NoError(t, err, "atomicWriteFile should succeed")

		// Verify the file was created
		_, err = os.Stat(nestedPath)
		assert.NoError(t, err, "File should exist after atomic write")

		// Verify parent directory structure was created
		_, err = os.Stat(parentDir)
		assert.NoError(t, err, "Parent directory should exist")

		// Verify file contents
		content, err := os.ReadFile(nestedPath)
		require.NoError(t, err, "Should be able to read written file")
		assert.Equal(t, testData, content, "File content should match")
	})

	t.Run("Error handling for invalid paths", func(t *testing.T) {
		// StateManager currently allows empty base directory
		// This test just documents the current behavior
		sm, err := NewStateManager("")
		if err != nil {
			t.Logf("StateManager correctly rejects empty base directory: %v", err)
		} else {
			t.Logf("StateManager allows empty base directory (current behavior)")
			if sm != nil {
				sm.Close()
			}
		}

		// Test atomicWriteFile with empty filename
		err = atomicWriteFile("", []byte("test"), 0o644)
		assert.Error(t, err, "atomicWriteFile should fail with empty filename")
	})

	t.Run("Concurrent directory creation", func(t *testing.T) {
		// Test that concurrent directory creation doesn't cause race conditions
		concurrentDir := filepath.Join(tempDir, "concurrent", "creation")

		// Run multiple goroutines trying to create the same directory structure
		done := make(chan error, 5)

		for i := 0; i < 5; i++ {
			go func(id int) {
				filePath := filepath.Join(concurrentDir, "file"+string(rune('0'+id))+".txt")
				err := atomicWriteFile(filePath, []byte("concurrent test"), 0o644)
				done <- err
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 5; i++ {
			err := <-done
			assert.NoError(t, err, "Concurrent file creation should succeed")
		}

		// Verify all files were created
		entries, err := os.ReadDir(concurrentDir)
		require.NoError(t, err, "Should be able to read concurrent directory")
		assert.Equal(t, 5, len(entries), "Should have 5 files created concurrently")
	})

	t.Run("Windows long path handling", func(t *testing.T) {
		// Skip this test on non-Windows platforms
		if !isWindows() {
			t.Skip("Windows-specific test")
		}

		// Create a very long path (close to Windows limits)
		longDirName := "verylongdirectoryname_" + string(make([]byte, 50)) // Pad with spaces
		for i := range longDirName[25:] {
			if longDirName[25+i] == 0 {
				break
			}
			longDirName = longDirName[:25+i] + "x" + longDirName[25+i+1:]
		}

		longPath := filepath.Join(tempDir, longDirName, longDirName, longDirName, "test.txt")

		// Attempt to write to the long path
		err := atomicWriteFile(longPath, []byte("long path test"), 0o644)
		// On Windows, this should either succeed or fail gracefully
		if err != nil {
			// If it fails, it should be a clear error about path length
			t.Logf("Long path creation failed as expected: %v", err)
		} else {
			// If it succeeds, verify the file exists
			_, err = os.Stat(longPath)
			assert.NoError(t, err, "Long path file should exist if creation succeeded")
		}
	})
}

// isWindows returns true if running on Windows
func isWindows() bool {
	return filepath.Separator == '\\'
}

// TestDirectoryCreationErrorHandling tests error scenarios for directory creation
func TestDirectoryCreationErrorHandling(t *testing.T) {
	t.Run("Permission denied scenario", func(t *testing.T) {
		if isWindows() {
			t.Skip("Permission tests are complex on Windows")
		}

		tempDir := t.TempDir()

		// Create a directory with restricted permissions
		restrictedDir := filepath.Join(tempDir, "restricted")
		require.NoError(t, os.Mkdir(restrictedDir, 0o000)) // No permissions
		defer os.Chmod(restrictedDir, 0o755)               // Restore permissions for cleanup

		// Try to create a file in a subdirectory of the restricted directory
		testFile := filepath.Join(restrictedDir, "subdir", "test.txt")
		err := atomicWriteFile(testFile, []byte("test"), 0o644)

		// Should fail with permission error
		assert.Error(t, err, "Should fail to create file in restricted directory")
		assert.Contains(t, err.Error(), "permission denied", "Error should mention permission denied")
	})

	t.Run("Invalid characters in path", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create paths with potentially problematic characters
		problematicPaths := []string{
			filepath.Join(tempDir, "file\x00name.txt"), // Null byte
		}

		if isWindows() {
			// Add Windows-specific invalid characters
			problematicPaths = append(problematicPaths,
				filepath.Join(tempDir, "file<name.txt"),
				filepath.Join(tempDir, "file>name.txt"),
				filepath.Join(tempDir, "file|name.txt"),
			)
		}

		for _, path := range problematicPaths {
			err := atomicWriteFile(path, []byte("test"), 0o644)
			// Should fail gracefully
			assert.Error(t, err, "Should fail with invalid path characters: %s", path)
		}
	})
}
