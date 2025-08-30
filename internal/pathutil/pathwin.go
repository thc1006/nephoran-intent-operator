// Package pathutil provides Windows-specific path normalization utilities.

package pathutil

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// NormalizeWindowsPath normalizes a Windows path, resolving multiple platform-specific complexities.
// It performs the following transformations:
//  1. Converts forward slashes to backslashes
//  2. Resolves drive-relative paths (e.g., C:foo)
//  3. Cleans the path (removes redundant separators and navigation segments)
//  4. Converts to an absolute path
//  5. Adds \\?\ prefix for Windows long paths (> 248 characters)
//
// On non-Windows platforms, it simply cleans the path using filepath.Clean().
//
// Parameters:
//
//	p: The input path to normalize
//
// Returns:
//
//	A normalized, cleaned path string and any encountered error
func NormalizeWindowsPath(p string) (string, error) {

	if p == "" {

		return "", fmt.Errorf("empty path")

	}

	// On non-Windows, just clean the path and return.

	if runtime.GOOS != "windows" {

		return filepath.Clean(p), nil

	}

	// Step 1: Convert forward slashes to backslashes for Windows consistency.

	p = strings.ReplaceAll(p, "/", "\\")

	// Step 2: Handle drive-relative paths (e.g., C:foo\bar).

	// These are relative to the current directory on the specified drive.

	if len(p) >= 2 && p[1] == ':' && (len(p) == 2 || (len(p) > 2 && p[2] != '\\')) {

		// This is a drive-relative path.

		// Use filepath.Abs to resolve it properly.

		absPath, err := filepath.Abs(p)

		if err != nil {

			return "", fmt.Errorf("failed to resolve drive-relative path %q: %w", p, err)

		}

		p = absPath

	}

	// Step 3: Clean the path (removes redundant separators, . and .. elements).

	// filepath.Clean handles most normalization and is safe with .. segments.

	cleaned := filepath.Clean(p)

	// Step 4: Convert to absolute path if not already.

	if !filepath.IsAbs(cleaned) {

		absPath, err := filepath.Abs(cleaned)

		if err != nil {

			return "", fmt.Errorf("failed to get absolute path for %q: %w", cleaned, err)

		}

		cleaned = absPath

	}

	// Step 5: Handle long paths by adding \\?\ prefix when needed.

	// Don't add prefix if path is already using it or if it's a UNC path.

	if len(cleaned) >= 248 && !strings.HasPrefix(cleaned, `\\?\`) && !strings.HasPrefix(cleaned, `\\`) {

		// Add extended-length path prefix for long paths.

		cleaned = `\\?\` + cleaned

	}

	return cleaned, nil

}

// IsValidWindowsPath checks if a given path is valid according to Windows path conventions.
// This validation includes several checks:
//  1. Disallows empty paths
//  2. Prevents Windows-reserved characters (< > " | ? *)
//  3. Validates drive letter placement and format
//  4. Blocks paths with reserved device names (CON, PRN, AUX, etc.)
//
// The function is lenient with relative paths and dot navigation segments.
// On non-Windows platforms, it always returns true.
//
// Parameters:
//
//	p: The path string to validate
//
// Returns:
//
//	true if the path is valid on Windows, false otherwise
func IsValidWindowsPath(p string) bool {

	if runtime.GOOS != "windows" {

		return true // Skip validation on non-Windows

	}

	if p == "" {

		return false

	}

	// Allow paths with . and .. segments - filepath.Clean() will resolve them safely.

	// Examples of allowed patterns:.

	// - "./././tmp/test" -> "tmp/test" after cleaning.

	// - "../parent/file" -> "../parent/file" if legitimately outside current dir.

	// Check for truly invalid characters (but allow : in drive position).

	invalidChars := []string{"<", ">", "\"", "|", "?", "*"}

	for _, char := range invalidChars {

		if strings.Contains(p, char) {

			return false

		}

	}

	// Check for multiple colons (only one allowed for drive letter).

	colonCount := strings.Count(p, ":")

	if colonCount > 1 {

		return false

	}

	if colonCount == 1 {

		// Colon must be at position 1 (after drive letter).

		colonIndex := strings.Index(p, ":")

		if colonIndex != 1 {

			return false

		}

		// Check that it's preceded by a letter.

		if p[0] < 'A' || (p[0] > 'Z' && p[0] < 'a') || p[0] > 'z' {

			return false

		}

	}

	// Check for reserved names.

	reservedNames := []string{

		"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4",

		"COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4",

		"LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	}

	baseName := strings.ToUpper(filepath.Base(p))

	// Remove extension for checking.

	if dotIndex := strings.LastIndex(baseName, "."); dotIndex != -1 {

		baseName = baseName[:dotIndex]

	}

	for _, reserved := range reservedNames {

		if baseName == reserved {

			return false

		}

	}

	return true

}

// ResolveDriveRelativePath resolves a drive-relative path to an absolute path on Windows.
// Drive-relative paths (e.g., C:foo) are resolved relative to the current directory on the specified drive.
//
// On non-Windows platforms, the path is returned as-is without modification.
//
// Parameters:
//
//	p: The drive-relative path to resolve
//
// Returns:
//
//	The resolved absolute path and any encountered error
func ResolveDriveRelativePath(p string) (string, error) {

	if runtime.GOOS != "windows" {

		return p, nil

	}

	// Check if it's a drive-relative path.

	if len(p) >= 2 && p[1] == ':' && (len(p) == 2 || (len(p) > 2 && p[2] != '\\' && p[2] != '/')) {

		// Use filepath.Abs which properly handles drive-relative paths on Windows.

		return filepath.Abs(p)

	}

	return p, nil

}

// EnsureParentDirectory creates the parent directory for a given file path if it doesn't exist.
// This method is safe for concurrent use and handles various edge cases:
//  1. Normalizes the input path
//  2. Handles empty paths
//  3. Skips creation for root or drive root directories
//  4. Gracefully handles concurrent directory creation
//
// If the parent directory already exists, the function returns nil.
// If the parent path exists but is not a directory, an error is returned.
//
// Parameters:
//
//	path: The file path whose parent directory should be ensured
//
// Returns:
//
//	An error if directory creation fails, nil otherwise
func EnsureParentDirectory(path string) error {

	if path == "" {

		return fmt.Errorf("empty path")

	}

	// Normalize the path first for consistent handling.

	normalized, err := NormalizeWindowsPath(path)

	if err != nil {

		// If normalization fails, try with the original path as fallback.

		// This allows the function to work even with malformed paths.

		normalized = filepath.Clean(path)

	}

	dir := filepath.Dir(normalized)

	if dir == "" || dir == "." || dir == "/" {

		return nil // Current directory or root, assume it exists

	}

	// Handle absolute paths that resolve to drive root on Windows.

	if runtime.GOOS == "windows" && len(dir) == 3 && dir[1] == ':' && dir[2] == '\\' {

		return nil // Drive root like C:\, assume it exists

	}

	// Check if directory already exists (os.MkdirAll is idempotent but this avoids syscall).

	if info, err := os.Stat(dir); err == nil {

		if !info.IsDir() {

			return fmt.Errorf("parent path exists but is not a directory: %q", dir)

		}

		return nil // Directory already exists

	}

	// Create the directory with all parents using os.MkdirAll.

	// os.MkdirAll is safe for concurrent use and handles existing directories gracefully.

	if err := os.MkdirAll(dir, 0o755); err != nil {

		// Check if error is due to concurrent creation (directory exists now).

		if os.IsExist(err) {

			// Double-check that it's actually a directory.

			if info, statErr := os.Stat(dir); statErr == nil && info.IsDir() {

				return nil // Successfully created by another goroutine

			}

		}

		return fmt.Errorf("failed to create parent directory %q: %w", dir, err)

	}

	return nil

}

// IsExtendedLengthPath determines whether a path uses the \\?\ prefix for Windows extended-length paths.
// Extended-length paths allow for paths exceeding the standard 260-character limit.
//
// Parameters:
//
//	path: The file path to check
//
// Returns:
//
//	true if the path starts with \\?\, false otherwise
func IsExtendedLengthPath(path string) bool {

	return strings.HasPrefix(path, `\\?\`)

}
