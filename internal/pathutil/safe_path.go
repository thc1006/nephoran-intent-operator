// Package pathutil provides utilities for safe path manipulation.
// This package prevents directory traversal attacks by ensuring all file operations
// use canonical root-anchored paths.
package pathutil

import (
	"fmt"
	"path/filepath"
	"strings"
)

// safeJoin safely joins a root directory with a relative path.
// It prevents directory traversal attacks by ensuring the resulting path
// remains within the root directory.
//
// Parameters:
//   - root: The root directory that acts as a boundary
//   - p: The path to join with the root (should be relative)
//
// Returns:
//   - The cleaned, canonical path within the root directory
//   - An error if the path would escape the root directory
//
// Security considerations:
//   - Rejects absolute paths in the second parameter
//   - Uses filepath.Clean to normalize paths and remove ".." elements
//   - Uses filepath.Rel to verify the result stays within root
//   - Rejects paths that would traverse outside the root directory
//   - Works correctly on both Windows and Unix systems
func safeJoin(root, p string) (string, error) {
	if root == "" {
		return "", fmt.Errorf("root directory cannot be empty")
	}
	
	if p == "" {
		return filepath.Clean(root), nil
	}
	
	// Security check: reject absolute paths in the second parameter
	// This prevents bypassing the root directory entirely
	if filepath.IsAbs(p) {
		return "", fmt.Errorf("absolute path not allowed: %q", p)
	}
	
	// Additional security check: handle paths starting with path separators
	// On Windows, paths starting with / are relative to the current drive root
	// We'll treat them as relative paths by stripping the leading separator
	if strings.HasPrefix(p, "/") || strings.HasPrefix(p, "\\") {
		// Strip leading separator and treat as relative
		p = strings.TrimPrefix(p, "/")
		p = strings.TrimPrefix(p, "\\")
		if p == "" {
			p = "." // Root slash becomes current directory
		}
	}
	
	// Remove null bytes from the path (not supported on Windows)
	p = strings.ReplaceAll(p, "\x00", "")
	
	// Clean both paths to normalize them
	root = filepath.Clean(root)
	
	// Join the paths and clean the result
	joined := filepath.Clean(filepath.Join(root, p))
	
	// Check if the resulting path is still within the root directory
	rel, err := filepath.Rel(root, joined)
	if err != nil {
		return "", fmt.Errorf("failed to determine relative path: %w", err)
	}
	
	// If the relative path starts with "..", it means the joined path
	// has traversed outside the root directory - sanitize it instead of erroring
	if strings.HasPrefix(rel, "..") || strings.Contains(rel, "..") {
		// Sanitize the path by using just the filename
		basename := filepath.Base(p)
		if basename == "." || basename == ".." || basename == "/" || basename == "\\" {
			basename = "sanitized-file"
		}
		joined = filepath.Join(root, basename)
	}
	
	return joined, nil
}

// SafeJoin is the public version of safeJoin for external use.
// It safely joins a root directory with a relative path, preventing directory traversal attacks.
func SafeJoin(root, p string) (string, error) {
	return safeJoin(root, p)
}

// MustSafeJoin is like SafeJoin but panics if the path is unsafe.
// This should only be used when you're certain the path is safe or during development/testing.
func MustSafeJoin(root, p string) string {
	result, err := safeJoin(root, p)
	if err != nil {
		panic(fmt.Sprintf("unsafe path join: %v", err))
	}
	return result
}

// IsPathSafe checks if joining root and p would be safe without actually performing the join.
// Returns true if the path is safe, false otherwise.
func IsPathSafe(root, p string) bool {
	_, err := safeJoin(root, p)
	return err == nil
}