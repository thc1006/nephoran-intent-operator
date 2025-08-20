// Package pathutil provides Windows-specific path utilities.
package pathutil

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// WindowsMaxPath is the maximum path length without long path support
const WindowsMaxPath = 248 // Leave room for filename

// NormalizeUserPath normalizes and validates a user-provided path.
// On Windows, it adds \\?\ prefix for long paths.
// It rejects paths with traversal attempts.
func NormalizeUserPath(p string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("empty path")
	}

	// Clean the path to remove . and .. elements
	cleaned := filepath.Clean(p)

	// Check for path traversal attempts
	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("path traversal detected: %q", p)
	}

	// Convert to absolute path
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// On Windows, handle long paths
	if runtime.GOOS == "windows" {
		// Check if the path is already a UNC path
		if !strings.HasPrefix(abs, `\\?\`) && !strings.HasPrefix(abs, `\\.\`) {
			// If path is too long, add the \\?\ prefix
			if len(abs) >= WindowsMaxPath {
				abs = `\\?\` + abs
			}
		}
	}

	return abs, nil
}

// EnsureParentDir ensures the parent directory of a path exists.
func EnsureParentDir(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}

	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return nil // Current directory, assume it exists
	}

	// Check if directory exists
	if _, err := os.Stat(dir); err == nil {
		return nil // Already exists
	}

	// Create the directory with all parents
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory %q: %w", dir, err)
	}

	return nil
}

// IsWindowsBatchFile checks if a file is a Windows batch file
func IsWindowsBatchFile(path string) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".bat" || ext == ".cmd"
}

// NormalizeCRLF converts Windows CRLF line endings to Unix LF
func NormalizeCRLF(data []byte) []byte {
	return []byte(strings.ReplaceAll(string(data), "\r\n", "\n"))
}