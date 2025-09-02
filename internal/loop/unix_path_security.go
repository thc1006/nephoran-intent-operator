//go:build !windows
// +build !windows

package loop

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// isPathSafe checks if a path is safe to use within a base directory on Unix systems.
// It prevents path traversal attacks by ensuring the resolved path is within the base directory.
func isPathSafe(path, baseDir string) bool {
	// Clean and resolve both paths to absolute paths
	cleanPath := filepath.Clean(path)
	cleanBase := filepath.Clean(baseDir)

	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return false
	}

	absBase, err := filepath.Abs(cleanBase)
	if err != nil {
		return false
	}

	// Check if the path is within the base directory
	// Add trailing slash to base to prevent partial matches
	basePath := absBase + string(filepath.Separator)
	targetPath := absPath + string(filepath.Separator)

	return strings.HasPrefix(targetPath, basePath) || absPath == absBase
}

// validateUnixPath performs Unix-specific path validation
func validateUnixPath(path string) error {
	// Check for dangerous path traversal patterns
	if strings.Contains(path, "../") {
		return fmt.Errorf("path traversal detected: path contains '../' sequences")
	}

	// Check for null bytes which can be dangerous
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("invalid path: contains null byte")
	}

	// Resolve the path to check for symlink attacks
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		// If the path doesn't exist yet, that's okay, but we need to check parent directories
		parent := filepath.Dir(path)
		if parent != "." && parent != "/" {
			_, err := os.Stat(parent)
			if os.IsNotExist(err) {
				return fmt.Errorf("output directory parent does not exist: %s", parent)
			}
		}
		return nil // Path doesn't exist yet, which is acceptable
	}

	// Check if resolved path contains dangerous patterns
	if strings.Contains(resolved, "../") {
		return fmt.Errorf("path traversal detected after symlink resolution")
	}

	return nil
}

// checkUnixPermissions checks if we have proper permissions for the given path on Unix systems
func checkUnixPermissions(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Check parent directory permissions
			parent := filepath.Dir(path)
			return checkUnixPermissions(parent)
		}
		return fmt.Errorf("cannot access path %s: %v", path, err)
	}

	// Check if it's a directory and we have write permissions
	if info.IsDir() {
		// Try to create a temporary file to test write permissions
		tempFile := filepath.Join(path, ".temp_perm_test")
		file, err := os.Create(tempFile)
		if err != nil {
			if os.IsPermission(err) {
				return fmt.Errorf("permission denied: cannot write to output directory %q", path)
			}
			return fmt.Errorf("cannot write to output directory %q: %v", path, err)
		}
		file.Close()
		os.Remove(tempFile)
		return nil
	}

	return fmt.Errorf("path exists but is not a directory: %s", path)
}

// secureTempfilePrefix generates a secure prefix for temporary files on Unix systems
func secureTempfilePrefix() string {
	// Use process ID and user ID for uniqueness and security
	pid := os.Getpid()
	uid := os.Getuid()
	return fmt.Sprintf("nephoran_%d_%d_", pid, uid)
}

// validateUnixFilename checks if a filename is valid on Unix systems
func validateUnixFilename(filename string) error {
	// Check for null bytes
	if strings.Contains(filename, "\x00") {
		return fmt.Errorf("filename contains null byte")
	}

	// Check for excessively long filenames (most Unix systems limit to 255 bytes)
	if len(filename) > 255 {
		return fmt.Errorf("filename too long: %d bytes (max 255)", len(filename))
	}

	// Unix allows most characters, but check for suspicious patterns
	suspicious := []string{"/../", "/./", "//"}
	for _, pattern := range suspicious {
		if strings.Contains(filename, pattern) {
			return fmt.Errorf("filename contains suspicious pattern: %s", strings.TrimPrefix(pattern, "/"))
		}
	}

	return nil
}

// isUnixHidden checks if a file is hidden on Unix systems (starts with dot)
func isUnixHidden(filename string) bool {
	base := filepath.Base(filename)
	return strings.HasPrefix(base, ".") && base != "." && base != ".."
}

// getUnixFileMode returns the file mode for newly created files on Unix
func getUnixFileMode() os.FileMode {
	return 0644 // rw-r--r--
}

// getUnixDirMode returns the directory mode for newly created directories on Unix
func getUnixDirMode() os.FileMode {
	return 0755 // rwxr-xr-x
}

// unixStat is a wrapper around os.Stat that returns Unix-specific information
func unixStat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

// unixLstat is a wrapper around os.Lstat (lstat syscall) for Unix systems
func unixLstat(path string) (os.FileInfo, error) {
	return os.Lstat(path)
}

// isUnixSymlink checks if a path is a symbolic link on Unix systems
func isUnixSymlink(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	return info.Mode()&os.ModeSymlink != 0, nil
}

// resolveUnixSymlinks resolves all symbolic links in a path on Unix systems
func resolveUnixSymlinks(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}

// checkUnixDiskSpace checks if there's sufficient disk space on Unix systems
func checkUnixDiskSpace(path string, requiredBytes int64) error {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return fmt.Errorf("cannot check disk space: %v", err)
	}

	// Calculate available space
	availableBytes := int64(stat.Bavail) * int64(stat.Bsize)

	if availableBytes < requiredBytes {
		return fmt.Errorf("insufficient disk space: %d bytes available, %d bytes required",
			availableBytes, requiredBytes)
	}

	return nil
}
