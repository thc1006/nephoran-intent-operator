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

// TestWindowsFileRaceConditions tests that our retry logic handles Windows file system race conditions
func TestWindowsFileRaceConditions(t *testing.T) {
	t.Parallel()
	
	tempDir := t.TempDir()
	
	t.Run("ReadFileWithRetry_HandlesTransientErrors", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test-read.json")
		content := []byte(`{"test": "data"}`)
		
		// Write file
		require.NoError(t, os.WriteFile(testFile, content, 0644))
		
		// Read should succeed
		data, err := readFileWithRetry(testFile)
		require.NoError(t, err)
		assert.Equal(t, content, data)
		
		// Simulate file being moved by another process
		movedFile := filepath.Join(tempDir, "moved.json")
		require.NoError(t, os.Rename(testFile, movedFile))
		
		// Read should return ErrFileGone after retries
		_, err = readFileWithRetry(testFile)
		assert.ErrorIs(t, err, ErrFileGone)
	})
	
	t.Run("AtomicWriteFile_EnsuresAtomicWrites", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test-atomic.json")
		content := []byte(`{"atomic": "write"}`)
		
		// Write atomically
		err := atomicWriteFile(testFile, content, 0644)
		require.NoError(t, err)
		
		// Read back
		data, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, content, data)
		
		// Ensure temp file doesn't exist
		_, err = os.Stat(testFile + ".tmp")
		assert.True(t, os.IsNotExist(err), "temp file should not exist")
	})
	
	t.Run("ConcurrentFileOperations", func(t *testing.T) {
		// Test that multiple goroutines can safely write status files
		statusDir := filepath.Join(tempDir, "status")
		require.NoError(t, os.MkdirAll(statusDir, 0755))
		
		var wg sync.WaitGroup
		numWorkers := 10
		
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				
				statusFile := filepath.Join(statusDir, fmt.Sprintf("status-%d.json", id))
				content := []byte(`{"status": "success"}`)
				
				// Each goroutine writes its own status file atomically
				err := atomicWriteFile(statusFile, content, 0644)
				assert.NoError(t, err)
			}(i)
		}
		
		wg.Wait()
		
		// Verify all files were written
		entries, err := os.ReadDir(statusDir)
		require.NoError(t, err)
		assert.Equal(t, numWorkers, len(entries))
	})
	
	t.Run("MoveFileAtomic_HandlesCrossDirectoryMoves", func(t *testing.T) {
		srcDir := filepath.Join(tempDir, "src")
		dstDir := filepath.Join(tempDir, "dst")
		require.NoError(t, os.MkdirAll(srcDir, 0755))
		require.NoError(t, os.MkdirAll(dstDir, 0755))
		
		srcFile := filepath.Join(srcDir, "test.json")
		dstFile := filepath.Join(dstDir, "test.json")
		content := []byte(`{"move": "test"}`)
		
		// Create source file
		require.NoError(t, os.WriteFile(srcFile, content, 0644))
		
		// Move atomically
		err := moveFileAtomic(srcFile, dstFile)
		require.NoError(t, err)
		
		// Source should not exist
		_, err = os.Stat(srcFile)
		assert.True(t, os.IsNotExist(err))
		
		// Destination should exist with correct content
		data, err := os.ReadFile(dstFile)
		require.NoError(t, err)
		assert.Equal(t, content, data)
	})
	
	t.Run("StatusFileWrittenBeforeMove", func(t *testing.T) {
		// Simulate the watcher workflow: write status then move file
		intentFile := filepath.Join(tempDir, "intent.json")
		statusDir := filepath.Join(tempDir, "status")
		processedDir := filepath.Join(tempDir, "processed")
		
		require.NoError(t, os.MkdirAll(statusDir, 0755))
		require.NoError(t, os.MkdirAll(processedDir, 0755))
		
		// Create intent file
		require.NoError(t, os.WriteFile(intentFile, []byte(`{"intent": "test"}`), 0644))
		
		// Write status file first
		statusFile := filepath.Join(statusDir, "intent.json-20250101-120000.status")
		err := atomicWriteFile(statusFile, []byte(`{"status": "success"}`), 0644)
		require.NoError(t, err)
		
		// Small delay to ensure status is written
		time.Sleep(10 * time.Millisecond)
		
		// Move intent file to processed
		processedFile := filepath.Join(processedDir, "intent.json")
		err = moveFileAtomic(intentFile, processedFile)
		require.NoError(t, err)
		
		// Both status and processed file should exist
		_, err = os.Stat(statusFile)
		require.NoError(t, err, "status file should exist")
		
		_, err = os.Stat(processedFile)
		require.NoError(t, err, "processed file should exist")
		
		// Original intent file should not exist
		_, err = os.Stat(intentFile)
		assert.True(t, os.IsNotExist(err), "original file should be moved")
	})
}