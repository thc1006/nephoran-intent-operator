package pathutil

import (
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeUserPath(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantErr   bool
		errContains string
	}{
		{
			name:    "valid relative path",
			input:   "test/file.json",
			wantErr: false,
		},
		{
			name:    "valid absolute path",
			input:   "/tmp/test/file.json",
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
				assert.True(t, strings.HasPrefix(result, "/") || 
					(runtime.GOOS == "windows" && (strings.Contains(result, ":") || strings.HasPrefix(result, `\\?\`))),
					"Path should be absolute: %s", result)
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