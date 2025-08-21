// Package pathutil provides Windows-specific path utilities.
package pathutil

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"regexp"
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
	if len(p) > WindowsMaxPath && !strings.HasPrefix(p, `\\?\`) {
		return fmt.Errorf("path too long (%d chars, max %d without \\\\?\\ prefix): %q", len(p), WindowsMaxPath, p)
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

	// Perform Windows-specific validation first
	if err := ValidateWindowsPath(p); err != nil {
		return "", fmt.Errorf("Windows path validation failed: %w", err)
	}

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

	// Clean the path to remove . and .. elements
	cleaned := filepath.Clean(p)

	// Check for path traversal attempts after cleaning
	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("path traversal detected: %q", p)
	}

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
		if err := ValidateWindowsPath(abs); err != nil {
			return "", fmt.Errorf("normalized path validation failed: %w", err)
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