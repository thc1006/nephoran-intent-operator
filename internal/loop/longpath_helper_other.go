//go:build !windows

package loop

// IsLongPathSupportEnabled always returns false on non-Windows platforms
// as MAX_PATH limitations are Windows-specific.
func IsLongPathSupportEnabled() bool {
	return false
}

// NormalizeLongPath is a no-op on non-Windows platforms
// as the \\?\ prefix is Windows-specific.
func NormalizeLongPath(path string) string {
	return path
}
