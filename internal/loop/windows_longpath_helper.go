//go:build windows

package loop

import (
	"os"
	"path/filepath"
	"strings"
)

// IsLongPathSupportEnabled empirically detects if Windows long path support is enabled
// by attempting to create a directory with a path longer than MAX_PATH (260 chars).
// Returns true if the operation succeeds, indicating long path support is enabled.
func IsLongPathSupportEnabled() bool {
	// Create a temporary base directory
	tempDir, err := os.MkdirTemp("", "longpath-test")
	if err != nil {
		return false
	}
	defer os.RemoveAll(tempDir)

	// Create a path longer than 260 characters without \\?\ prefix
	// Windows MAX_PATH is 260, including null terminator
	longName := strings.Repeat("a", 250) // Ensure we exceed MAX_PATH
	longPath := filepath.Join(tempDir, longName, "test")

	// Ensure the path is actually long enough
	if len(longPath) <= 260 {
		// If we can't create a long enough path, assume no long path support
		return false
	}

	// Attempt to create the directory
	err = os.MkdirAll(longPath, 0o755)
	if err != nil {
		// Failed to create long path - MAX_PATH is enforced
		return false
	}

	// Successfully created long path - long path support is enabled
	// Clean up (best effort)
	os.RemoveAll(filepath.Join(tempDir, longName))
	return true
}

// NormalizeLongPath adds the \\?\ prefix to a path if needed for long path support.
// This is only needed on Windows for paths exceeding MAX_PATH.
func NormalizeLongPath(path string) string {
	// Only process absolute paths
	if !filepath.IsAbs(path) {
		return path
	}

	// Check if path already has extended-length prefix
	if strings.HasPrefix(path, `\\?\`) || strings.HasPrefix(path, `\\.\`) {
		return path
	}

	// Add prefix for long paths (over 260 chars)
	if len(path) > 260 {
		return `\\?\` + path
	}

	return path
}
