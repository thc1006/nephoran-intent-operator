package pathutil

import (
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateWindowsPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	tests := []struct {
		name        string
		input       string
		wantErr     bool
		errContains string
	}{
		// Valid paths
		{
			name:    "valid absolute path with drive",
			input:   "C:\\temp\\file.txt",
			wantErr: false,
		},
		{
			name:    "valid UNC path",
			input:   "\\\\server\\share\\file.txt",
			wantErr: false,
		},
		{
			name:    "valid long path with prefix",
			input:   "\\\\?\\C:\\temp\\file.txt",
			wantErr: false,
		},
		{
			name:    "valid relative path",
			input:   "temp\\file.txt",
			wantErr: false,
		},
		// Edge cases that should fail
		{
			name:        "drive letter only (relative)",
			input:       "C:",
			wantErr:     true,
			errContains: "drive letter without path",
		},
		{
			name:        "empty path",
			input:       "",
			wantErr:     true,
			errContains: "empty path",
		},
		{
			name:        "invalid character <",
			input:       "C:\\temp\\<file>.txt",
			wantErr:     true,
			errContains: "invalid character",
		},
		{
			name:        "invalid character >",
			input:       "C:\\temp\\>file.txt",
			wantErr:     true,
			errContains: "invalid character",
		},
		{
			name:        "invalid character |",
			input:       "C:\\temp\\file|.txt",
			wantErr:     true,
			errContains: "invalid character",
		},
		{
			name:        "multiple colons",
			input:       "C:\\temp:colon\\file.txt",
			wantErr:     true,
			errContains: "multiple colons",
		},
		{
			name:        "colon in wrong position",
			input:       "temp:C\\file.txt",
			wantErr:     true,
			errContains: "colon in invalid position",
		},
		{
			name:        "reserved name CON",
			input:       "C:\\temp\\CON.txt",
			wantErr:     true,
			errContains: "reserved filename",
		},
		{
			name:        "reserved name PRN",
			input:       "C:\\temp\\PRN",
			wantErr:     true,
			errContains: "reserved filename",
		},
		{
			name:        "reserved name COM1",
			input:       "C:\\temp\\com1.log",
			wantErr:     true,
			errContains: "reserved filename",
		},
		{
			name:        "path too long without prefix",
			input:       "C:\\" + strings.Repeat("a", 256),
			wantErr:     true,
			errContains: "path too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWindowsPath(tt.input)
			
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNormalizeUserPath(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErr     bool
		errContains string
		expected    string // Expected result on Windows
	}{
		{
			name:    "valid relative path",
			input:   "test/file.json",
			wantErr: false,
		},
		{
			name:    "valid absolute path Unix style",
			input:   "/tmp/test/file.json",
			wantErr: false,
		},
		{
			name:    "valid Windows absolute path",
			input:   "C:\\temp\\file.json",
			wantErr: false,
		},
		{
			name:    "mixed separators",
			input:   "C:/temp\\file.json",
			wantErr: false,
		},
		{
			name:        "drive with relative path",
			input:       "C:temp\\file.json",
			wantErr:     false, // Should be converted to absolute
		},
		{
			name:    "UNC path",
			input:   "\\\\server\\share\\file.json",
			wantErr: false,
		},
		{
			name:        "empty path",
			input:       "",
			wantErr:     true,
			errContains: "empty path",
		},
		{
			name:        "path traversal with ..",
			input:       "../etc/passwd",
			wantErr:     true,
			errContains: "path traversal",
		},
		{
			name:        "path traversal in middle",
			input:       "test/../../../etc/passwd",
			wantErr:     true,
			errContains: "path traversal",
		},
		{
			name:        "Windows invalid characters",
			input:       "C:\\temp\\file<test>.txt",
			wantErr:     true,
			errContains: "Windows path validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizeUserPath(tt.input)
			
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result)
				// Result should be absolute
				if runtime.GOOS == "windows" {
					assert.True(t, IsAbsoluteWindowsPath(result),
						"Path should be absolute on Windows: %s", result)
				} else {
					assert.True(t, strings.HasPrefix(result, "/"),
						"Path should be absolute on Unix: %s", result)
				}
			}
		})
	}
}

func TestNormalizeUserPath_WindowsLongPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	// Create a very long path
	longPath := "C:\\" + strings.Repeat("a", 250) + "\\file.json"
	
	result, err := NormalizeUserPath(longPath)
	require.NoError(t, err)
	
	// Should have UNC prefix for long paths
	assert.True(t, strings.HasPrefix(result, `\\?\`), "Long path should have UNC prefix: %s", result)
}

func TestNormalizeUserPath_WindowsEdgeCases(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	tests := []struct {
		name     string
		input    string
		wantErr  bool
		contains string // What the result should contain
	}{
		{
			name:     "drive with relative path conversion",
			input:    "C:temp\\file.txt",
			wantErr:  false,
			contains: "C:\\", // Should be converted to absolute
		},
		{
			name:    "mixed separators normalization",
			input:   "C:/temp\\file.txt",
			wantErr: false,
		},
		{
			name:    "UNC path preservation",
			input:   "\\\\server\\share\\file.txt",
			wantErr: false,
		},
		{
			name:    "device path preservation",
			input:   "\\\\?\\C:\\temp\\file.txt",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizeUserPath(tt.input)
			
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result)
				if tt.contains != "" {
					assert.Contains(t, result, tt.contains)
				}
				// Result should be absolute
				assert.True(t, IsAbsoluteWindowsPath(result),
					"Path should be absolute: %s", result)
			}
		})
	}
}

func TestWindowsPathHelpers(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		isUNC    bool
		isDevice bool
		isAbs    bool
	}{
		{
			name:     "regular absolute path",
			path:     "C:\\temp\\file.txt",
			isUNC:    false,
			isDevice: false,
			isAbs:    true,
		},
		{
			name:     "UNC path",
			path:     "\\\\server\\share\\file.txt",
			isUNC:    true,
			isDevice: false,
			isAbs:    true,
		},
		{
			name:     "device path \\\\?\\",
			path:     "\\\\?\\C:\\temp\\file.txt",
			isUNC:    false,
			isDevice: true,
			isAbs:    true,
		},
		{
			name:     "device path \\\\.\\  ",
			path:     "\\\\.\\C:\\temp\\file.txt",
			isUNC:    false,
			isDevice: true,
			isAbs:    true,
		},
		{
			name:     "relative path",
			path:     "temp\\file.txt",
			isUNC:    false,
			isDevice: false,
			isAbs:    false,
		},
		{
			name:     "drive relative",
			path:     "C:temp",
			isUNC:    false,
			isDevice: false,
			isAbs:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isUNC, IsUNCPath(tt.path), "IsUNCPath mismatch")
			assert.Equal(t, tt.isDevice, IsWindowsDevicePath(tt.path), "IsWindowsDevicePath mismatch")
			
			if runtime.GOOS == "windows" {
				assert.Equal(t, tt.isAbs, IsAbsoluteWindowsPath(tt.path), "IsAbsoluteWindowsPath mismatch")
			}
		})
	}
}

func TestNormalizePathSeparators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "mixed separators on Windows",
			input:    "C:/temp\\file.txt",
			expected: func() string {
				if runtime.GOOS == "windows" {
					return "C:\\temp\\file.txt"
				}
				return "C:/temp/file.txt"
			}(),
		},
		{
			name:     "Unix path on Windows",
			input:    "/usr/local/bin",
			expected: func() string {
				if runtime.GOOS == "windows" {
					return "\\usr\\local\\bin"
				}
				return "/usr/local/bin"
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizePathSeparators(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnsureParentDir(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "create new directory",
			path:    tempDir + "/new/sub/file.txt",
			wantErr: false,
		},
		{
			name:    "existing directory",
			path:    tempDir + "/file.txt",
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "current directory",
			path:    "file.txt",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EnsureParentDir(tt.path)
			
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIsWindowsBatchFile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"batch file .bat", "test.bat", runtime.GOOS == "windows"},
		{"batch file .BAT", "test.BAT", runtime.GOOS == "windows"},
		{"command file .cmd", "test.cmd", runtime.GOOS == "windows"},
		{"command file .CMD", "test.CMD", runtime.GOOS == "windows"},
		{"shell script", "test.sh", false},
		{"executable", "test.exe", false},
		{"no extension", "test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsWindowsBatchFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeCRLF(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Windows line endings",
			input:    "line1\r\nline2\r\nline3",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "Unix line endings",
			input:    "line1\nline2\nline3",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "Mixed line endings",
			input:    "line1\r\nline2\nline3\r\n",
			expected: "line1\nline2\nline3\n",
		},
		{
			name:     "No line endings",
			input:    "single line",
			expected: "single line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeCRLF([]byte(tt.input))
			assert.Equal(t, tt.expected, string(result))
		})
	}
}