// Package pathutil provides Windows-specific path utilities.
package pathutil

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// WindowsMaxPath is the maximum path length without long path support
const WindowsMaxPath = 248 // Leave room for filename

// Windows path validation patterns
var (
	// windowsDrivePattern matches Windows drive letters (C:, D:, etc.)
	windowsDrivePattern = regexp.MustCompile(`^[A-Za-z]:$`)
	// windowsAbsPathPattern matches absolute Windows paths (C:\path, D:\path)
	windowsAbsPathPattern = regexp.MustCompile(`^[A-Za-z]:[/\\]`)
	// windowsUNCPattern matches UNC paths (\\server\share) - not device paths
	windowsUNCPattern = regexp.MustCompile(`^\\\\[^\\?.]([^\\])*\\[^\\\s][^\\]*`)
	// windowsDevicePattern matches Windows device paths (\\?\ or \\.\)
	windowsDevicePattern = regexp.MustCompile(`^\\\\[?.]\\`)
)

// ValidateWindowsPath validates Windows-specific path edge cases
// It handles drive letters, UNC paths, mixed separators, and various edge cases.
func ValidateWindowsPath(p string) error {
	if runtime.GOOS != "windows" {
		return nil // Skip Windows validation on non-Windows systems
	}

	if p == "" {
		return fmt.Errorf("empty path")
	}

	// Check for drive letter only (C:, D:) - these are relative paths
	if windowsDrivePattern.MatchString(p) {
		return fmt.Errorf("drive letter without path (relative to current directory on drive): %q", p)
	}

	// Skip character validation for device paths as they have special rules
	if windowsDevicePattern.MatchString(p) {
		// Device paths are pre-validated by Windows, just check length
		if len(p) > 32767 { // Max path length for device paths
			return fmt.Errorf("device path too long (%d chars, max 32767): %q", len(p), p)
		}
		return nil
	}

	// Check for malformed UNC-like paths that start with \\ but are not valid UNC or device paths
	if strings.HasPrefix(p, "\\\\") && !windowsUNCPattern.MatchString(p) && !windowsDevicePattern.MatchString(p) {
		return fmt.Errorf("malformed UNC path: %q", p)
	}

	// Check for invalid characters in Windows paths
	invalidChars := []string{"<", ">", ":", "\"", "|", "?", "*"}
	for _, char := range invalidChars {
		// Allow colon only in drive letter position
		if char == ":" {
			// Count colons and check positions
			colonCount := strings.Count(p, ":")
			if colonCount > 1 {
				return fmt.Errorf("multiple colons not allowed in path: %q", p)
			}
			if colonCount == 1 {
				colonIndex := strings.Index(p, ":")
				// Colon must be at position 1 for drive letter (C:)
				if colonIndex != 1 || !regexp.MustCompile(`^[A-Za-z]:`).MatchString(p[:2]) {
					return fmt.Errorf("colon in invalid position: %q", p)
				}
			}
			continue
		}
		if strings.Contains(p, char) {
			return fmt.Errorf("invalid character %q in path: %q", char, p)
		}
	}

	// Check for reserved names (CON, PRN, AUX, etc.)
	reservedNames := []string{"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"}
	baseName := strings.ToUpper(filepath.Base(p))
	// Remove extension for checking
	if dotIndex := strings.LastIndex(baseName, "."); dotIndex != -1 {
		baseName = baseName[:dotIndex]
	}
	for _, reserved := range reservedNames {
		if baseName == reserved {
			return fmt.Errorf("reserved filename: %q", p)
		}
	}

	// Check path length (Windows has a 260 character limit by default, we use 248 to be conservative)
	// Allow some tolerance for paths near the limit as they can be fixed with \\?\ prefix
	maxLength := WindowsMaxPath + 10 // Allow some buffer for edge cases
	if len(p) > maxLength && !strings.HasPrefix(p, `\\?\`) {
		return fmt.Errorf("path too long (%d chars, max %d without \\\\?\\ prefix): %q", len(p), maxLength, p)
	}

	return nil
}

// NormalizeUserPath normalizes and validates a user-provided path.
// On Windows, it adds \\?\ prefix for long paths and handles various edge cases.
// It rejects paths with traversal attempts.
func NormalizeUserPath(p string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("empty path")
	}

	// Skip Windows validation initially - we'll validate after normalization
	// This allows paths that exceed the limit to be processed and potentially fixed by adding \\?\ prefix

	// Normalize mixed path separators on Windows
	if runtime.GOOS == "windows" {
		// Convert forward slashes to backslashes for consistency
		p = strings.ReplaceAll(p, "/", "\\")

		// Handle drive letter with relative path (C:temp -> C:\temp)
		if len(p) >= 2 && p[1] == ':' && len(p) > 2 && p[2] != '\\' {
			// This is a relative path on a specific drive (e.g., C:temp)
			// Convert to absolute path by getting current directory on that drive
			drive := p[:2]
			rest := p[2:]
			// Get current working directory for this drive
			cwd, err := os.Getwd()
			if err == nil && len(cwd) >= 2 && strings.ToUpper(cwd[:2]) == strings.ToUpper(drive) {
				// Same drive, use current directory as base
				p = filepath.Join(cwd, rest)
			} else {
				// Different drive, assume root of the drive
				p = drive + "\\" + rest
			}
		}
	}

	// Security check: detect path traversal attempts before cleaning
	// This is important because filepath.Clean() will resolve legitimate .. segments
	if detectPathTraversal(p) {
		return "", fmt.Errorf("path traversal attempt detected: %q", p)
	}

	// Clean the path to remove . and .. elements
	cleaned := filepath.Clean(p)

	// Convert to absolute path
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// On Windows, handle long paths and UNC paths
	if runtime.GOOS == "windows" {
		// Check if the path is already a long path or UNC path
		if !windowsDevicePattern.MatchString(abs) && !windowsUNCPattern.MatchString(abs) {
			// If path is too long, add the \\?\ prefix
			if len(abs) >= WindowsMaxPath {
				abs = `\\?\` + abs
			}
		}

		// Final validation of the normalized path
		// For long paths with \\?\ prefix, skip some validations that don't apply
		if !windowsDevicePattern.MatchString(abs) {
			if err := ValidateWindowsPath(abs); err != nil {
				return "", fmt.Errorf("Windows path validation failed: %w", err)
			}
		}
	}

	return abs, nil
}

// EnsureParentDir ensures the parent directory of a path exists.
// This function is resilient to concurrent directory creation using os.MkdirAll.
func EnsureParentDir(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}

	dir := filepath.Dir(path)
	if dir == "" || dir == "." || dir == "/" {
		return nil // Current directory or root, assume it exists
	}

	// Handle Windows drive root paths
	if runtime.GOOS == "windows" && len(dir) == 3 && dir[1] == ':' && (dir[2] == '\\' || dir[2] == '/') {
		return nil // Drive root like C:\ or C:/, assume it exists
	}

	// Check if directory already exists
	if info, err := os.Stat(dir); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("parent path exists but is not a directory: %q", dir)
		}
		return nil // Directory already exists
	}

	// Create the directory with all parents using os.MkdirAll
	// os.MkdirAll is safe for concurrent use and idempotent
	if err := os.MkdirAll(dir, 0755); err != nil {
		// Handle race condition where directory was created concurrently
		if os.IsExist(err) {
			// Verify it's actually a directory
			if info, statErr := os.Stat(dir); statErr == nil && info.IsDir() {
				return nil // Directory exists, created concurrently
			}
		}
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

// IsUNCPath checks if a path is a UNC path (\\server\share)
func IsUNCPath(path string) bool {
	return windowsUNCPattern.MatchString(path)
}

// IsWindowsDevicePath checks if a path is a Windows device path (\\?\ or \\.\)
func IsWindowsDevicePath(path string) bool {
	return windowsDevicePattern.MatchString(path)
}

// IsAbsoluteWindowsPath checks if a path is absolute on Windows
// This includes drive letters (C:\), UNC paths (\\server\share), and device paths
func IsAbsoluteWindowsPath(path string) bool {
	if runtime.GOOS != "windows" {
		return filepath.IsAbs(path)
	}
	return windowsAbsPathPattern.MatchString(path) || IsUNCPath(path) || IsWindowsDevicePath(path)
}

// NormalizePathSeparators normalizes path separators for the current OS
func NormalizePathSeparators(path string) string {
	if runtime.GOOS == "windows" {
		return strings.ReplaceAll(path, "/", "\\")
	}
	return strings.ReplaceAll(path, "\\", "/")
}

// NormalizeCRLF converts Windows CRLF line endings to Unix LF
func NormalizeCRLF(data []byte) []byte {
	return []byte(strings.ReplaceAll(string(data), "\r\n", "\n"))
}

// detectPathTraversal checks for path traversal patterns that should be rejected
// It looks for patterns that attempt to access files outside the intended directory
func detectPathTraversal(p string) bool {
	// Normalize separators for consistent checking
	normalized := strings.ReplaceAll(p, "\\", "/")

	// Check for common path traversal patterns
	traversalPatterns := []string{
		"../",      // Standard path traversal
		"..\\",     // Windows path traversal
		"/..",      // Path traversal at end or in middle
		"\\..",     // Windows path traversal at end
		"..%2f",    // URL encoded forward slash
		"..%5c",    // URL encoded backslash
		"%2e%2e/",  // URL encoded dots with slash
		"%2e%2e\\", // URL encoded dots with backslash
		"...//",    // Triple dots (some systems)
		"....//",   // Quad dots variation
	}

	// Convert to lowercase for case-insensitive matching
	lowerPath := strings.ToLower(normalized)

	// Check each pattern
	for _, pattern := range traversalPatterns {
		if strings.Contains(lowerPath, strings.ToLower(pattern)) {
			return true
		}
	}

	// Check if path starts with ../ or ..\\ (immediate parent directory access)
	if strings.HasPrefix(normalized, "../") || strings.HasPrefix(p, "..\\") {
		return true
	}

	// Check for excessive .. sequences (more than 2 in a row is suspicious)
	if strings.Contains(normalized, "../../..") {
		return true
	}

	return false
}
