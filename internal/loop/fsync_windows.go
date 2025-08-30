//go:build windows
// +build windows

package loop

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/thc1006/nephoran-intent-operator/internal/pathutil"
)

// Windows-specific file sync operations with retry logic for race conditions

const (
	// Retry parameters for Windows file operations
	maxFileRetries  = 10
	baseRetryDelay  = 5 * time.Millisecond
	maxRetryDelay   = 500 * time.Millisecond
	fileSyncTimeout = 2 * time.Second
)

// atomicWriteFile writes data to a file atomically on Windows with proper syncing
func atomicWriteFile(filename string, data []byte, perm os.FileMode) error {
	if filename == "" {
		return fmt.Errorf("empty filename")
	}

	// Normalize path for Windows (handles long paths, drive-relative, etc.)
	normalizedPath, err := pathutil.NormalizeWindowsPath(filename)
	if err != nil {
		return fmt.Errorf("failed to normalize path %q: %w", filename, err)
	}

	// Ensure parent directory exists using robust utility
	if err := pathutil.EnsureParentDirectory(normalizedPath); err != nil {
		return fmt.Errorf("failed to create parent directory for %q: %w", normalizedPath, err)
	}

	// Write to temporary file first
	tempFile := normalizedPath + ".tmp"

	// Write with sync
	if err := writeFileWithSync(tempFile, data, perm); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename with retry for Windows
	if err := renameFileWithRetry(tempFile, normalizedPath); err != nil {
		os.Remove(tempFile) // Clean up on failure
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// writeFileWithSync writes data to a file and ensures it's synced to disk
func writeFileWithSync(filename string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return err
	}

	// Sync file contents to disk
	if err := f.Sync(); err != nil {
		// On Windows, some file systems don't support Sync
		// Log warning but don't fail
		if !isUnsupportedError(err) {
			return fmt.Errorf("failed to sync file: %w", err)
		}
	}

	return nil
}

// renameFileWithRetry attempts to rename a file with exponential backoff retry
func renameFileWithRetry(oldpath, newpath string) error {
	var lastErr error
	delay := baseRetryDelay

	for i := 0; i < maxFileRetries; i++ {
		err := os.Rename(oldpath, newpath)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			return err
		}

		// Wait before retry with exponential backoff
		time.Sleep(delay)
		delay *= 2
		if delay > maxRetryDelay {
			delay = maxRetryDelay
		}
	}

	return fmt.Errorf("rename failed after %d retries: %w", maxFileRetries, lastErr)
}

// readFileWithRetry reads a file with retry logic for Windows race conditions
func readFileWithRetry(filename string) ([]byte, error) {
	var lastErr error
	delay := baseRetryDelay

	for i := 0; i < maxFileRetries; i++ {
		data, err := os.ReadFile(filename)
		if err == nil {
			return data, nil
		}

		lastErr = err

		// If file doesn't exist, don't retry
		if os.IsNotExist(err) {
			// On final attempt, return specific error
			if i == maxFileRetries-1 {
				return nil, ErrFileGone
			}
		}

		// Check if error is retryable
		if !isRetryableError(err) {
			return nil, err
		}

		// Wait before retry
		time.Sleep(delay)
		delay *= 2
		if delay > maxRetryDelay {
			delay = maxRetryDelay
		}
	}

	// If we exhausted retries due to file not existing, return ErrFileGone
	if os.IsNotExist(lastErr) {
		return nil, ErrFileGone
	}

	return nil, fmt.Errorf("read failed after %d retries: %w", maxFileRetries, lastErr)
}

// openFileWithRetry opens a file with retry logic for Windows race conditions
func openFileWithRetry(filename string) (*os.File, error) {
	var lastErr error
	delay := baseRetryDelay

	// Use fewer retries for open operations since they're more frequent
	const openMaxRetries = 3

	for i := 0; i < openMaxRetries; i++ {
		file, err := os.Open(filename)
		if err == nil {
			return file, nil
		}

		lastErr = err

		// If file doesn't exist, don't retry - return immediately
		if os.IsNotExist(err) {
			return nil, err
		}

		// Check if error is retryable
		if !isRetryableError(err) {
			return nil, err
		}

		// Wait before retry with shorter delays
		if i < openMaxRetries-1 {
			time.Sleep(delay)
			delay *= 2
			if delay > 50*time.Millisecond {
				delay = 50 * time.Millisecond
			}
		}
	}

	return nil, fmt.Errorf("open failed after %d retries: %w", openMaxRetries, lastErr)
}

// copyFileWithSync copies a file with proper syncing on Windows
func copyFileWithSync(src, dst string) error {
	// Open source with retry
	srcFile, err := openFileWithRetry(src)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer srcFile.Close()

	// Create destination
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}
	defer dstFile.Close()

	// Copy contents
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy: %w", err)
	}

	// Sync destination to disk
	if err := dstFile.Sync(); err != nil {
		// Log warning but don't fail on unsupported sync
		if !isUnsupportedError(err) {
			return fmt.Errorf("failed to sync destination: %w", err)
		}
	}

	return nil
}

// moveFileAtomic performs an atomic file move with Windows-specific handling
func moveFileAtomic(src, dst string) error {
	// Ensure destination directory exists
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Try rename first (atomic on same volume)
	if err := renameFileWithRetry(src, dst); err == nil {
		return nil
	}

	// Fall back to copy+delete for cross-volume moves
	if err := copyFileWithSync(src, dst); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Delete source file with retry
	return removeFileWithRetry(src)
}

// removeFileWithRetry removes a file with retry logic
func removeFileWithRetry(filename string) error {
	var lastErr error
	delay := baseRetryDelay

	for i := 0; i < maxFileRetries; i++ {
		err := os.Remove(filename)
		if err == nil {
			return nil
		}

		// If file doesn't exist, consider it success
		if os.IsNotExist(err) {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			return err
		}

		// Wait before retry
		time.Sleep(delay)
		delay *= 2
		if delay > maxRetryDelay {
			delay = maxRetryDelay
		}
	}

	return fmt.Errorf("remove failed after %d retries: %w", maxFileRetries, lastErr)
}

// isRetryableError checks if an error is retryable on Windows
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific Windows errors that are retryable
	// Windows error codes from winerror.h
	const (
		ERROR_SHARING_VIOLATION = 32
		ERROR_LOCK_VIOLATION    = 33
		ERROR_ACCESS_DENIED     = 5
		ERROR_FILE_NOT_FOUND    = 2
		ERROR_PATH_NOT_FOUND    = 3
	)

	if pathErr, ok := err.(*os.PathError); ok {
		if errno, ok := pathErr.Err.(syscall.Errno); ok {
			switch errno {
			case ERROR_SHARING_VIOLATION, // File is in use
				ERROR_LOCK_VIOLATION, // File is locked
				ERROR_ACCESS_DENIED,  // Access denied (might be temporary)
				ERROR_FILE_NOT_FOUND, // File not found (might appear soon)
				ERROR_PATH_NOT_FOUND: // Path not found (might appear soon)
				return true
			}
		}
	}

	// Check error messages for known retryable patterns
	errStr := strings.ToLower(err.Error())
	retryablePatterns := []string{
		"sharing violation",
		"being used by another process",
		"access is denied",
		"the system cannot find",
		"permission denied",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// isUnsupportedError checks if an error is due to unsupported operation
func isUnsupportedError(err error) bool {
	if err == nil {
		return false
	}

	// Check for ENOTSUP or similar
	const ERROR_NOT_SUPPORTED = 50 // Windows error code for not supported

	if pathErr, ok := err.(*os.PathError); ok {
		if errno, ok := pathErr.Err.(syscall.Errno); ok {
			// Windows doesn't have ENOTSUP, but ERROR_NOT_SUPPORTED is similar
			if errno == ERROR_NOT_SUPPORTED {
				return true
			}
		}
	}

	// Check error message
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "not supported") ||
		strings.Contains(errStr, "invalid device")
}

// waitForFileStable waits for a file to become stable (not being written)
func waitForFileStable(filename string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastSize int64
	var lastMod time.Time

	for time.Now().Before(deadline) {
		stat, err := os.Stat(filename)
		if err != nil {
			if os.IsNotExist(err) {
				time.Sleep(baseRetryDelay)
				continue
			}
			return err
		}

		size := stat.Size()
		mod := stat.ModTime()

		// Check if file is stable
		if size == lastSize && mod.Equal(lastMod) {
			return nil
		}

		lastSize = size
		lastMod = mod
		time.Sleep(50 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for file to become stable")
}
