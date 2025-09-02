//go:build !windows
// +build !windows

package loop

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/thc1006/nephoran-intent-operator/internal/pathutil"
)

// Unix-specific file sync operations (simpler than Windows).

const (

	// Retry parameters for Unix file operations (less aggressive than Windows).

	maxFileRetries = 3

	baseRetryDelay = 10 * time.Millisecond

	maxRetryDelay = 100 * time.Millisecond

	fileSyncTimeout = 1 * time.Second
)

// atomicWriteFile writes data to a file atomically on Unix with proper syncing.

func atomicWriteFile(filename string, data []byte, perm os.FileMode) error {
	if filename == "" {
		return fmt.Errorf("empty filename")
	}

	// On Unix, use normal path cleaning (NormalizeWindowsPath handles non-Windows gracefully).

	normalizedPath, err := pathutil.NormalizeWindowsPath(filename)
	if err != nil {
		return fmt.Errorf("failed to normalize path %q: %w", filename, err)
	}

	// Ensure parent directory exists.

	if err := pathutil.EnsureParentDirectory(normalizedPath); err != nil {
		return fmt.Errorf("failed to create parent directory for %q: %w", normalizedPath, err)
	}

	// Write to temporary file first.

	tempFile := normalizedPath + ".tmp"

	// Write with sync.

	if err := writeFileWithSync(tempFile, data, perm); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename (truly atomic on Unix).

	if err := os.Rename(tempFile, normalizedPath); err != nil {

		os.Remove(tempFile) // Clean up on failure

		return fmt.Errorf("failed to rename file: %w", err)

	}

	return nil
}

// writeFileWithSync writes data to a file and ensures it's synced to disk.

func writeFileWithSync(filename string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}

	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return err
	}

	// Sync file contents to disk.

	if err := f.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	// Also sync the directory to ensure the file entry is persisted.

	dirFd, err := os.Open(filepath.Dir(filename))

	if err == nil {

		if err := dirFd.Sync(); err != nil {
			log.Printf("Warning: Failed to sync directory %s: %v", filepath.Dir(filename), err)
		}

		dirFd.Close()

	}

	return nil
}

// renameFileWithRetry attempts to rename a file with limited retry on Unix.

func renameFileWithRetry(oldpath, newpath string) error {
	// On Unix, rename is atomic, so we don't need aggressive retry.

	err := os.Rename(oldpath, newpath)

	if err == nil {
		return nil
	}

	// Only retry for transient errors.

	if !os.IsNotExist(err) && !os.IsPermission(err) {

		time.Sleep(baseRetryDelay)

		if err := os.Rename(oldpath, newpath); err == nil {
			return nil
		}

	}

	return err
}

// readFileWithRetry reads a file with minimal retry logic for Unix.

func readFileWithRetry(filename string) ([]byte, error) {
	data, err := os.ReadFile(filename)

	if err == nil {
		return data, nil
	}

	// On Unix, file operations are more reliable, so minimal retry.

	if !os.IsNotExist(err) {

		time.Sleep(baseRetryDelay)

		data, err = os.ReadFile(filename)

		if err == nil {
			return data, nil
		}

	}

	if os.IsNotExist(err) {
		return nil, ErrFileGone
	}

	return nil, err
}

// openFileWithRetry opens a file with minimal retry logic for Unix.

func openFileWithRetry(filename string) (*os.File, error) {
	file, err := os.Open(filename)

	if err == nil {
		return file, nil
	}

	// Minimal retry on Unix.

	if !os.IsNotExist(err) {

		time.Sleep(baseRetryDelay)

		file, err = os.Open(filename)

		if err == nil {
			return file, nil
		}

	}

	if os.IsNotExist(err) {
		return nil, ErrFileGone
	}

	return nil, err
}

// copyFileWithSync copies a file with proper syncing on Unix.

func copyFileWithSync(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}

	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}

	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy: %w", err)
	}

	// Sync destination to disk.

	if err := dstFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync destination: %w", err)
	}

	return nil
}

// moveFileAtomic performs an atomic file move on Unix.

func moveFileAtomic(src, dst string) error {
	// Ensure destination directory exists.

	dstDir := filepath.Dir(dst)

	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Rename is atomic on Unix within same filesystem.

	if err := os.Rename(src, dst); err != nil {

		// Fall back to copy+delete for cross-filesystem moves.

		if err := copyFileWithSync(src, dst); err != nil {
			return fmt.Errorf("failed to copy file: %w", err)
		}

		return os.Remove(src)

	}

	return nil
}

// removeFileWithRetry removes a file with minimal retry logic.

func removeFileWithRetry(filename string) error {
	err := os.Remove(filename)

	if err == nil || os.IsNotExist(err) {
		return nil
	}

	// Single retry on Unix.

	time.Sleep(baseRetryDelay)

	err = os.Remove(filename)

	if err == nil || os.IsNotExist(err) {
		return nil
	}

	return err
}

// isRetryableError checks if an error is retryable on Unix.

func isRetryableError(err error) bool {
	// On Unix, most errors are not retryable.

	// Only retry for EINTR or EAGAIN.

	return false
}

// isUnsupportedError checks if an error is due to unsupported operation.

func isUnsupportedError(err error) bool {
	// Unix systems generally support sync operations.

	return false
}

// waitForFileStable waits for a file to become stable (not being written).

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

		// Check if file is stable.

		if size == lastSize && mod.Equal(lastMod) {
			return nil
		}

		lastSize = size

		lastMod = mod

		time.Sleep(50 * time.Millisecond)

	}

	return fmt.Errorf("timeout waiting for file to become stable")
}
