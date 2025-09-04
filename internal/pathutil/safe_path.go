// Package pathutil provides utilities for safe path manipulation.

// This package prevents directory traversal attacks by ensuring all file operations.

// use canonical root-anchored paths.

package pathutil

import (
	"fmt"
	"path/filepath"
	"strings"
)

// safeJoin safely joins a root directory with a relative path.

// It prevents directory traversal attacks by ensuring the resulting path.

// remains within the root directory.

//

// Parameters:.

//   - root: The root directory that acts as a boundary.

//   - p: The path to join with the root (should be relative).

//

// Returns:.

//   - The cleaned, canonical path within the root directory.

//   - An error if the path would escape the root directory.

//

// Security considerations:.

//   - Rejects absolute paths in the second parameter.

//   - Uses filepath.Clean to normalize paths and remove ".." elements.

//   - Uses filepath.Rel to verify the result stays within root.

//   - Rejects paths that would traverse outside the root directory.

//   - Works correctly on both Windows and Unix systems.

func safeJoin(root, p string) (string, error) {
	if root == "" {
		return "", fmt.Errorf("root directory cannot be empty")
	}

	if p == "" {
		return filepath.Clean(root), nil
	}

	// Security check: reject absolute paths in the second parameter.

	// This prevents bypassing the root directory entirely.

	if filepath.IsAbs(p) {
		return "", fmt.Errorf("absolute path not allowed: %q", p)
	}

	// Additional security check: reject absolute paths that start with separators.

	// These bypass the root directory entirely and should be rejected.

	if strings.HasPrefix(p, "/") && len(p) > 1 {
		return "", fmt.Errorf("absolute path not allowed: %q", p)
	}

	if strings.HasPrefix(p, "\\") && len(p) > 1 {
		return "", fmt.Errorf("absolute path not allowed: %q", p)
	}

	// Handle single separator (treat as current directory).

	if p == "/" || p == "\\" {
		p = "."
	}

	// Early path traversal detection before processing.

	// Only check for obvious traversal patterns - let the final check handle legitimate cases.

	if detectObviousPathTraversal(p) {
		return "", fmt.Errorf("path traversal detected in path %q", p)
	}

	// Remove null bytes from the path (not supported on Windows).

	p = strings.ReplaceAll(p, "\x00", "")

	// Clean both paths to normalize them.

	root = filepath.Clean(root)

	// Join the paths and clean the result.

	joined := filepath.Clean(filepath.Join(root, p))

	// Check if the resulting path is still within the root directory.

	rel, err := filepath.Rel(root, joined)
	if err != nil {
		return "", fmt.Errorf("failed to determine relative path: %w", err)
	}

	// If the relative path starts with "..", it means the joined path.

	// has traversed outside the root directory - reject it as a security violation.

	if strings.HasPrefix(rel, "..") || strings.Contains(rel, "..") {
		return "", fmt.Errorf("path traversal attempt: path %q would escape root directory %q", p, root)
	}

	return joined, nil
}

// SafeJoin safely joins a root directory with a relative path, preventing directory traversal attacks.
// This function ensures that the resulting path always stays within the specified root directory.
//
// Security Features:
//   - Rejects absolute paths
//   - Removes null bytes from path
//   - Normalizes path using filepath.Clean
//   - Validates that the resulting path remains within root directory
//   - Compatible with both Windows and Unix-like systems
//
// Parameters:
//
//	root: The root directory that serves as a boundary for path resolution
//	p: The relative path to join with the root
//
// Returns:
//
//	A canonicalized, safe path within the root directory, or an error if the path is unsafe
func SafeJoin(root, p string) (string, error) {
	return safeJoin(root, p)
}

// MustSafeJoin is like SafeJoin but panics if the path is unsafe.
// Use this method only when you are absolutely certain the path is safe or during development/testing.
//
// Unlike SafeJoin, this method does not return an error. Instead, it will:
//   - Return the safe, canonicalized path if valid
//   - Panic with a descriptive error message if the path is unsafe
//
// Warning: This method should not be used in production code where error handling is crucial.
//
// Parameters:
//
//	root: The root directory that serves as a boundary for path resolution
//	p: The relative path to join with the root
//
// Returns:
//
//	A canonicalized, safe path within the root directory
//
// Panics:
//
//	If the path would escape the root directory or is otherwise unsafe
func MustSafeJoin(root, p string) string {
	result, err := safeJoin(root, p)
	if err != nil {
		panic(fmt.Sprintf("unsafe path join: %v", err))
	}

	return result
}

// IsPathSafe checks if joining a root directory with a path would be safe without actually performing the join.
// This method provides a lightweight way to validate a path before using SafeJoin.
//
// The method performs the same safety checks as SafeJoin, including:
//   - Rejecting absolute paths
//   - Detecting path traversal attempts
//   - Ensuring the path remains within the root directory
//
// Parameters:
//
//	root: The root directory that serves as a boundary for path resolution
//	p: The relative path to validate
//
// Returns:
//
//	true if the path is safe to join with the root directory, false otherwise
func IsPathSafe(root, p string) bool {
	_, err := safeJoin(root, p)

	return err == nil
}

// detectObviousPathTraversal checks for obvious path traversal attacks.

// This only catches patterns that are clearly malicious, allowing legitimate ../usage.

func detectObviousPathTraversal(p string) bool {
	// Normalize separators for consistent checking.

	normalized := strings.ReplaceAll(p, "\\", "/")

	lowerPath := strings.ToLower(normalized)

	// Check for URL-encoded traversal patterns (always malicious).

	encodedPatterns := []string{
		"..%2f", // URL encoded forward slash

		"..%5c", // URL encoded backslash

		"%2e%2e/", // URL encoded dots with slash

		"%2e%2e\\", // URL encoded dots with backslash

		"...//", // Triple dots (suspicious)

		"....//", // Quad dots (suspicious)

	}

	for _, pattern := range encodedPatterns {
		if strings.Contains(lowerPath, strings.ToLower(pattern)) {
			return true
		}
	}

	// Check if path starts with ../ (immediate parent directory escape).

	if strings.HasPrefix(normalized, "../") {
		return true
	}

	// Check for excessive .. sequences (3 or more levels up is suspicious).

	if strings.Contains(normalized, "../../../") {
		return true
	}

	return false
}
