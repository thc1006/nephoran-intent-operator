package loop

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileManager_BasicOperations(t *testing.T) {
	dir := t.TempDir()
	fm, err := NewFileManager(dir)
	require.NoError(t, err)

	// Test successful file move to processed
	t.Run("move to processed", func(t *testing.T) {
		testFile := filepath.Join(dir, "test-intent.json")
		testContent := `{"action": "scale", "target": "deployment", "count": 3}`
		require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

		err := fm.MoveToProcessed(testFile)
		require.NoError(t, err)

		// Original file should not exist
		assert.NoFileExists(t, testFile)

		// File should exist in processed directory
		targetPath := filepath.Join(fm.processedDir, "test-intent.json")
		assert.FileExists(t, targetPath)
	})

	// Test successful file move to failed
	t.Run("move to failed", func(t *testing.T) {
		testFile := filepath.Join(dir, "failed-intent.json")
		testContent := `{"action": "scale", "target": "deployment", "count": 3}`
		require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

		errorMsg := "porch command failed: exit code 1"
		err := fm.MoveToFailed(testFile, errorMsg)
		require.NoError(t, err)

		// Original file should not exist
		assert.NoFileExists(t, testFile)

		// File should exist in failed directory
		targetPath := filepath.Join(fm.failedDir, "failed-intent.json")
		assert.FileExists(t, targetPath)

		// Error log should exist
		logPath := filepath.Join(fm.failedDir, "failed-intent.json.error.log")
		assert.FileExists(t, logPath)
	})

	// Test stats
	t.Run("get stats", func(t *testing.T) {
		stats, err := fm.GetStats()
		require.NoError(t, err)
		assert.Equal(t, 1, stats.ProcessedCount)
		assert.Equal(t, 1, stats.FailedCount)
	})
}

func TestFileManager_ErrorScenarios(t *testing.T) {
	dir := t.TempDir()
	fm, err := NewFileManager(dir)
	require.NoError(t, err)

	// Test moving nonexistent file
	t.Run("move nonexistent file", func(t *testing.T) {
		nonexistentFile := filepath.Join(dir, "nonexistent.json")
		err := fm.MoveToProcessed(nonexistentFile)
		assert.Error(t, err)
	})
}
