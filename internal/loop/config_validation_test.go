package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigValidation tests the Config.Validate() method, particularly OutDir validation
func TestConfigValidation(t *testing.T) {
	t.Run("Valid output directory", func(t *testing.T) {
		tempDir := t.TempDir()
		config := Config{
			PorchPath:    "/tmp/porch",
			Mode:         "direct",
			OutDir:       tempDir,
			MaxWorkers:   2,
			DebounceDur:  100 * time.Millisecond,
			CleanupAfter: 7 * 24 * time.Hour,
			GracePeriod:  5 * time.Second,
		}

		err := config.Validate()
		assert.NoError(t, err, "Should validate successfully with valid directory")
	})

	t.Run("Non-existent output directory with writable parent", func(t *testing.T) {
		tempDir := t.TempDir()
		nonExistentDir := filepath.Join(tempDir, "subdir", "nested")
		
		config := Config{
			PorchPath:    "/tmp/porch",
			Mode:         "direct",
			OutDir:       nonExistentDir,
			MaxWorkers:   2,
			DebounceDur:  100 * time.Millisecond,
			CleanupAfter: 7 * 24 * time.Hour,
			GracePeriod:  5 * time.Second,
		}

		err := config.Validate()
		assert.NoError(t, err, "Should validate successfully when parent is writable")
	})

	t.Run("Read-only output directory", func(t *testing.T) {
		if filepath.Separator == '\\' {
			t.Skip("Read-only tests are complex on Windows")
		}
		
		tempDir := t.TempDir()
		readOnlyDir := filepath.Join(tempDir, "readonly")
		
		// Create directory
		require.NoError(t, os.MkdirAll(readOnlyDir, 0755))
		
		// Make it read-only
		require.NoError(t, os.Chmod(readOnlyDir, 0444))
		defer os.Chmod(readOnlyDir, 0755) // Restore for cleanup
		
		config := Config{
			PorchPath:    "/tmp/porch",
			Mode:         "direct",
			OutDir:       readOnlyDir,
			MaxWorkers:   2,
			DebounceDur:  100 * time.Millisecond,
			CleanupAfter: 7 * 24 * time.Hour,
			GracePeriod:  5 * time.Second,
		}

		err := config.Validate()
		assert.Error(t, err, "Should fail validation for read-only directory")
		if err != nil {
			assert.Contains(t, err.Error(), "permission denied", "Error should mention permission denied")
		}
	})

	t.Run("Output directory that is a file", func(t *testing.T) {
		tempDir := t.TempDir()
		fileName := filepath.Join(tempDir, "notadir")
		
		// Create a file instead of directory
		require.NoError(t, os.WriteFile(fileName, []byte("test"), 0644))
		
		config := Config{
			PorchPath:    "/tmp/porch",
			Mode:         "direct",
			OutDir:       fileName,
			MaxWorkers:   2,
			DebounceDur:  100 * time.Millisecond,
			CleanupAfter: 7 * 24 * time.Hour,
			GracePeriod:  5 * time.Second,
		}

		err := config.Validate()
		assert.Error(t, err, "Should fail validation when OutDir is not a directory")
		assert.Contains(t, err.Error(), "not a directory", "Error should mention it's not a directory")
	})

	t.Run("Empty output directory should be allowed", func(t *testing.T) {
		config := Config{
			PorchPath:    "/tmp/porch",
			Mode:         "direct",
			OutDir:       "", // Empty means no validation needed
			MaxWorkers:   2,
			DebounceDur:  100 * time.Millisecond,
			CleanupAfter: 7 * 24 * time.Hour,
			GracePeriod:  5 * time.Second,
		}

		err := config.Validate()
		assert.NoError(t, err, "Should allow empty OutDir")
	})

	t.Run("Completely inaccessible path", func(t *testing.T) {
		// Use a path that doesn't exist and has no writable parent
		invalidPath := filepath.Join("Z:", "nonexistent", "path", "that", "should", "fail")
		if filepath.Separator == '/' {
			// Unix path
			invalidPath = "/root/nonexistent/path/that/should/fail"
		}
		
		config := Config{
			PorchPath:    "/tmp/porch",
			Mode:         "direct",
			OutDir:       invalidPath,
			MaxWorkers:   2,
			DebounceDur:  100 * time.Millisecond,
			CleanupAfter: 7 * 24 * time.Hour,
			GracePeriod:  5 * time.Second,
		}

		err := config.Validate()
		assert.Error(t, err, "Should fail validation for completely inaccessible path")
		// The error should mention either permission denied or parent directory issues
		errorStr := err.Error()
		assert.True(t, 
			strings.Contains(errorStr, "permission denied") ||
			strings.Contains(errorStr, "cannot access") ||
			strings.Contains(errorStr, "does not exist"),
			"Error should mention access issues, got: %s", errorStr)
	})
}