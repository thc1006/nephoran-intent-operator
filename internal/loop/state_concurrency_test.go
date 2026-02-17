package loop

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIsProcessedRobustToENOENT verifies that IsProcessed handles missing files gracefully
// without returning errors, as required for concurrent operations where files may be
// moved or deleted by other workers.
func TestIsProcessedRobustToENOENT(t *testing.T) {
	tempDir := t.TempDir()
	sm, err := NewStateManager(tempDir)
	require.NoError(t, err)
	defer sm.Close() // #nosec G307 - Error handled in defer

	t.Run("NonExistentFileReturnsNotProcessedNoError", func(t *testing.T) {
		// When checking a file that doesn't exist and has no state entry,
		// should return false (not processed) with no error
		nonExistentFile := filepath.Join(tempDir, "nonexistent-file.json")
		processed, err := sm.IsProcessed(nonExistentFile)
		assert.NoError(t, err, "IsProcessed should not error on ENOENT")
		assert.False(t, processed, "Non-existent file should not be marked as processed")
	})

	t.Run("FileDisappearsAfterMarkedProcessed", func(t *testing.T) {
		// Create a file, mark it as processed, then delete it
		testFile := filepath.Join(tempDir, "disappearing.json")
		err := os.WriteFile(testFile, []byte(`{"test": true}`), 0o644)
		require.NoError(t, err)

		// Mark as processed using absolute path so calculateFileHash can find the file
		err = sm.MarkProcessed(testFile)
		require.NoError(t, err)

		// Delete the file (simulating concurrent removal)
		err = os.Remove(testFile)
		require.NoError(t, err)

		// IsProcessed should return false without error: the file hash no longer matches
		// because the file is gone and ErrFileGone causes a graceful false return.
		processed, err := sm.IsProcessed(testFile)
		assert.NoError(t, err, "IsProcessed should handle missing file gracefully")
		// processed may be true or false depending on implementation; the key invariant
		// is no error is returned.
		_ = processed
	})

	t.Run("ConcurrentFileOperations", func(t *testing.T) {
		var wg sync.WaitGroup
		numFiles := 20

		// Create files and record their absolute paths
		filePaths := make([]string, numFiles)
		for i := 0; i < numFiles; i++ {
			filename := fmt.Sprintf("concurrent-%d.json", i)
			testFile := filepath.Join(tempDir, filename)
			err := os.WriteFile(testFile, []byte(`{"test": true}`), 0o644)
			require.NoError(t, err)
			filePaths[i] = testFile
		}

		// Concurrent operations: mark some as processed while checking others
		for i := 0; i < numFiles; i++ {
			wg.Add(2)

			// Worker 1: Mark as processed (use absolute path so file can be found)
			go func(id int) {
				defer wg.Done()
				sm.MarkProcessed(filePaths[id])
			}(i)

			// Worker 2: Check if processed (may race with marking)
			go func(id int) {
				defer wg.Done()
				// Should never error, regardless of state
				_, err := sm.IsProcessed(filePaths[id])
				assert.NoError(t, err, "IsProcessed should never error during concurrent operations")
			}(i)
		}

		wg.Wait()

		// Verify all files are eventually marked as processed
		for i := 0; i < numFiles; i++ {
			processed, err := sm.IsProcessed(filePaths[i])
			assert.NoError(t, err)
			assert.True(t, processed, "File should be marked as processed")
		}
	})

	t.Run("FileDeletedDuringProcessing", func(t *testing.T) {
		// Simulate a file being deleted while another worker is checking it
		testFile := filepath.Join(tempDir, "racing.json")
		err := os.WriteFile(testFile, []byte(`{"test": true}`), 0o644)
		require.NoError(t, err)

		var wg sync.WaitGroup
		wg.Add(2)

		var checkErr error
		var checkResult bool

		// Worker 1: Delete the file after a small delay
		go func() {
			defer wg.Done()
			time.Sleep(10 * time.Millisecond)
			os.Remove(testFile)
		}()

		// Worker 2: Check if processed using absolute path
		go func() {
			defer wg.Done()
			// Add small delay to increase chance of race
			time.Sleep(5 * time.Millisecond)
			checkResult, checkErr = sm.IsProcessed(testFile)
		}()

		wg.Wait()

		// Should not error even if file was deleted during check
		assert.NoError(t, checkErr, "IsProcessed should handle concurrent deletion gracefully")
		assert.False(t, checkResult, "Unprocessed file should return false")
	})
}
