package pathutil

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestSafeJoin(t *testing.T) {
	tests := []struct {
		name        string
		root        string
		path        string
		expectError bool
		description string
	}{
		// Valid cases
		{
			name:        "simple relative path",
			root:        "/tmp",
			path:        "file.txt",
			expectError: false,
			description: "Should allow simple relative paths",
		},
		{
			name:        "nested relative path",
			root:        "/tmp",
			path:        "subdir/file.txt",
			expectError: false,
			description: "Should allow nested relative paths",
		},
		{
			name:        "current directory",
			root:        "/tmp",
			path:        ".",
			expectError: false,
			description: "Should handle current directory reference",
		},
		{
			name:        "empty path",
			root:        "/tmp",
			path:        "",
			expectError: false,
			description: "Should handle empty path (returns cleaned root)",
		},
		{
			name:        "path with redundant separators",
			root:        "/tmp",
			path:        "subdir//file.txt",
			expectError: false,
			description: "Should clean paths with redundant separators",
		},

		// Path traversal attempts
		{
			name:        "simple parent directory",
			root:        "/tmp",
			path:        "../",
			expectError: true,
			description: "Should reject simple parent directory traversal",
		},
		{
			name:        "double parent directory",
			root:        "/tmp",
			path:        "../../",
			expectError: true,
			description: "Should reject double parent directory traversal",
		},
		{
			name:        "mixed traversal attempt",
			root:        "/tmp",
			path:        "subdir/../../../etc/passwd",
			expectError: true,
			description: "Should reject complex traversal attempts",
		},
		{
			name:        "traversal with file",
			root:        "/tmp",
			path:        "../etc/passwd",
			expectError: true,
			description: "Should reject traversal to specific files",
		},

		// Absolute paths (should be rejected as they ignore root)
		{
			name:        "absolute unix path",
			root:        "/tmp",
			path:        "/etc/passwd",
			expectError: true,
			description: "Should reject absolute Unix paths",
		},

		// Edge cases
		{
			name:        "empty root",
			root:        "",
			path:        "file.txt",
			expectError: true,
			description: "Should reject empty root directory",
		},
		{
			name:        "root with trailing separator",
			root:        "/tmp/",
			path:        "file.txt",
			expectError: false,
			description: "Should handle root with trailing separator",
		},
		{
			name:        "complex valid path",
			root:        "/var/www",
			path:        "html/assets/../css/style.css",
			expectError: false,
			description: "Should allow complex but valid paths",
		},
	}

	// Add Windows-specific tests when running on Windows
	if runtime.GOOS == "windows" {
		windowsTests := []struct {
			name        string
			root        string
			path        string
			expectError bool
			description string
		}{
			{
				name:        "windows absolute path",
				root:        "C:\\temp",
				path:        "C:\\Windows\\System32",
				expectError: true,
				description: "Should reject Windows absolute paths",
			},
			{
				name:        "windows valid relative",
				root:        "C:\\temp",
				path:        "subdir\\file.txt",
				expectError: false,
				description: "Should allow Windows relative paths",
			},
			{
				name:        "windows traversal",
				root:        "C:\\temp",
				path:        "..\\..\\Windows\\System32",
				expectError: true,
				description: "Should reject Windows path traversal",
			},
		}
		tests = append(tests, windowsTests...)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SafeJoin(tt.root, tt.path)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none. Result: %s", tt.description, result)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.description, err)
				} else {
					// Verify the result is still within the root
					if !isWithinRoot(tt.root, result) {
						t.Errorf("Result %s is not within root %s", result, tt.root)
					}
				}
			}
		})
	}
}

func TestMustSafeJoin(t *testing.T) {
	// Test normal case
	result := MustSafeJoin("/tmp", "file.txt")
	expected := filepath.Join("/tmp", "file.txt")
	if result != expected {
		t.Errorf("MustSafeJoin failed: expected %s, got %s", expected, result)
	}

	// Test panic case
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustSafeJoin should have panicked for unsafe path")
		}
	}()
	MustSafeJoin("/tmp", "../etc/passwd")
}

func TestIsPathSafe(t *testing.T) {
	tests := []struct {
		root     string
		path     string
		expected bool
	}{
		{"/tmp", "file.txt", true},
		{"/tmp", "../etc/passwd", false},
		{"/tmp", "subdir/file.txt", true},
		{"", "file.txt", false}, // empty root
	}

	for _, tt := range tests {
		result := IsPathSafe(tt.root, tt.path)
		if result != tt.expected {
			t.Errorf("IsPathSafe(%q, %q) = %v, expected %v", tt.root, tt.path, result, tt.expected)
		}
	}
}

func TestSafeJoinBenchmark(t *testing.T) {
	// This is a simple performance test to ensure the function isn't too slow
	root := "/tmp"
	path := "subdir/file.txt"

	for i := 0; i < 1000; i++ {
		_, err := SafeJoin(root, path)
		if err != nil {
			t.Fatalf("Unexpected error in benchmark: %v", err)
		}
	}
}

// Helper function to verify a path is within the root directory
func isWithinRoot(root, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)

	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}

	return !filepath.IsAbs(rel) && !filepath.HasPrefix(rel, "..")
}

// Table-driven test for edge cases specific to different operating systems
func TestSafeJoinCrossPlatform(t *testing.T) {
	tests := []struct {
		name        string
		root        string
		path        string
		expectError bool
	}{
		// These should work on all platforms
		{"unix-style-root-unix-path", "/tmp", "file.txt", false},
		{"unix-style-root-traversal", "/tmp", "../etc", true},
		{"relative-root", "temp", "file.txt", false},
		{"relative-root-traversal", "temp", "../file.txt", true},
	}

	// Add platform-specific tests
	if runtime.GOOS == "windows" {
		tests = append(tests, []struct {
			name        string
			root        string
			path        string
			expectError bool
		}{
			{"windows-drive-root", "C:\\temp", "file.txt", false},
			{"windows-drive-traversal", "C:\\temp", "..\\..\\Windows", true},
			{"windows-unc-root", "\\\\server\\share", "file.txt", false},
		}...)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SafeJoin(tt.root, tt.path)
			if (err != nil) != tt.expectError {
				t.Errorf("SafeJoin(%q, %q) error = %v, expectError = %v", tt.root, tt.path, err, tt.expectError)
			}
		})
	}
}

// Benchmark the SafeJoin function
func BenchmarkSafeJoin(b *testing.B) {
	root := "/tmp"
	path := "subdir/file.txt"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = SafeJoin(root, path)
	}
}

func BenchmarkSafeJoinTraversal(b *testing.B) {
	root := "/tmp"
	path := "../../../etc/passwd"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = SafeJoin(root, path)
	}
}

// Fuzz test to find edge cases
func FuzzSafeJoin(f *testing.F) {
	// Add seed inputs
	f.Add("/tmp", "file.txt")
	f.Add("/tmp", "../etc/passwd")
	f.Add("", "file.txt")
	f.Add("/tmp", "")

	f.Fuzz(func(t *testing.T, root, path string) {
		// The function should never panic, regardless of input
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("SafeJoin panicked with root=%q, path=%q: %v", root, path, r)
			}
		}()

		result, err := SafeJoin(root, path)

		// If no error, the result should be within the root (if root is not empty)
		if err == nil && root != "" {
			if !isWithinRoot(root, result) {
				t.Errorf("Result %q is not within root %q", result, root)
			}
		}
	})
}
