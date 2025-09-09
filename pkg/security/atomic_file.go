package security

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// AtomicFileWriter provides secure atomic file creation with unique names
type AtomicFileWriter struct {
	dir string
}

// NewAtomicFileWriter creates a new atomic file writer for the specified directory
func NewAtomicFileWriter(dir string) (*AtomicFileWriter, error) {
	// Validate directory exists and is writable
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("directory validation failed: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", dir)
	}
	
	// Test write permission
	testFile := filepath.Join(dir, fmt.Sprintf(".test-%d", time.Now().UnixNano()))
	f, err := os.OpenFile(testFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("directory not writable: %w", err)
	}
	f.Close()
	os.Remove(testFile)
	
	return &AtomicFileWriter{dir: dir}, nil
}

// CreateIntentFile creates a unique intent file with atomic guarantees
// Returns the file handle and full path
func (w *AtomicFileWriter) CreateIntentFile() (*os.File, string, error) {
	// Generate cryptographically secure random ID
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, "", fmt.Errorf("failed to generate random ID: %w", err)
	}
	randomID := hex.EncodeToString(randomBytes)
	
	// Create filename with high-resolution timestamp and random ID
	// This ensures uniqueness even with concurrent operations
	fname := fmt.Sprintf("intent-%d-%s.json", time.Now().UnixNano(), randomID)
	fpath := filepath.Join(w.dir, fname)
	
	// Validate filename doesn't contain path traversal
	if err := validateFilename(fname); err != nil {
		return nil, "", fmt.Errorf("invalid filename: %w", err)
	}
	
	// Open with O_EXCL to ensure atomic creation (fails if file exists)
	f, err := os.OpenFile(fpath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create file atomically: %w", err)
	}
	
	return f, fpath, nil
}

// WriteIntent atomically writes intent data to a new file
func (w *AtomicFileWriter) WriteIntent(data []byte) (string, error) {
	f, fpath, err := w.CreateIntentFile()
	if err != nil {
		return "", err
	}
	defer f.Close()
	
	// Write data
	if _, err := f.Write(data); err != nil {
		// Clean up on write failure
		os.Remove(fpath)
		return "", fmt.Errorf("failed to write data: %w", err)
	}
	
	// Sync to ensure data is persisted
	if err := f.Sync(); err != nil {
		os.Remove(fpath)
		return "", fmt.Errorf("failed to sync data: %w", err)
	}
	
	return fpath, nil
}

// WriteIntentAtomic writes to a temporary file and atomically renames it
// This provides even stronger guarantees against partial writes
func (w *AtomicFileWriter) WriteIntentAtomic(data []byte) (string, error) {
	// Create temporary file
	tempFile, err := os.CreateTemp(w.dir, ".tmp-intent-*.json")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	
	// Ensure cleanup on any error
	success := false
	defer func() {
		if !success {
			tempFile.Close()
			os.Remove(tempPath)
		}
	}()
	
	// Write data
	if _, err := tempFile.Write(data); err != nil {
		return "", fmt.Errorf("failed to write data: %w", err)
	}
	
	// Sync before rename
	if err := tempFile.Sync(); err != nil {
		return "", fmt.Errorf("failed to sync data: %w", err)
	}
	
	// Close before rename (required on Windows)
	if err := tempFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}
	
	// Generate final filename
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random ID: %w", err)
	}
	fname := fmt.Sprintf("intent-%d-%s.json", time.Now().UnixNano(), hex.EncodeToString(randomBytes))
	finalPath := filepath.Join(w.dir, fname)
	
	// Atomic rename
	if err := os.Rename(tempPath, finalPath); err != nil {
		return "", fmt.Errorf("failed to rename file atomically: %w", err)
	}
	
	success = true
	return finalPath, nil
}

// validateFilename checks for path traversal attempts
func validateFilename(fname string) error {
	// Check for path traversal
	if filepath.Base(fname) != fname {
		return fmt.Errorf("filename contains path separator")
	}
	
	// Check for special characters that could cause issues
	for _, c := range fname {
		if c == 0 || c < 32 || c > 126 {
			return fmt.Errorf("filename contains invalid characters")
		}
	}
	
	// Check for relative path components
	if fname == "." || fname == ".." {
		return fmt.Errorf("invalid filename")
	}
	
	return nil
}

// SecureCopy performs a secure copy with validation
func SecureCopy(dst io.Writer, src io.Reader, maxSize int64) (int64, error) {
	// Limit the amount of data to prevent resource exhaustion
	limitedReader := io.LimitReader(src, maxSize)
	
	// Use io.CopyN for additional safety
	n, err := io.Copy(dst, limitedReader)
	if err != nil {
		return n, fmt.Errorf("copy failed: %w", err)
	}
	
	// Check if we hit the size limit
	if n == maxSize {
		// Try to read one more byte to see if there's more data
		buf := make([]byte, 1)
		if _, err := src.Read(buf); err != io.EOF {
			return n, fmt.Errorf("input exceeds maximum size of %d bytes", maxSize)
		}
	}
	
	return n, nil
}