package pathutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// DISABLED: func TestNormalizeWindowsPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific tests on non-Windows platform")
	}

	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, got string)
	}{
		{
			name:    "empty path",
			input:   "",
			wantErr: true,
		},
		{
			name:  "simple absolute path",
			input: `C:\Users\test`,
			check: func(t *testing.T, got string) {
				if !strings.HasPrefix(got, `C:\`) {
					t.Errorf("expected path to start with C:\\, got %q", got)
				}
			},
		},
		{
			name:  "drive-relative path",
			input: `C:foo\bar`,
			check: func(t *testing.T, got string) {
				if !filepath.IsAbs(got) {
					t.Errorf("expected absolute path, got %q", got)
				}
				if !strings.Contains(got, "foo") || !strings.Contains(got, "bar") {
					t.Errorf("expected path to contain foo\\bar, got %q", got)
				}
			},
		},
		{
			name:  "path with forward slashes",
			input: `C:/Users/test/file.txt`,
			check: func(t *testing.T, got string) {
				if strings.Contains(got, "/") {
					t.Errorf("expected backslashes, got %q", got)
				}
				if !strings.Contains(got, "Users") {
					t.Errorf("expected path to contain Users, got %q", got)
				}
			},
		},
		{
			name:  "mixed separators",
			input: `C:\Users/test\file.txt`,
			check: func(t *testing.T, got string) {
				if strings.Contains(got, "/") {
					t.Errorf("expected only backslashes, got %q", got)
				}
			},
		},
		{
			name:  "path with dots",
			input: `C:\Users\.\test\..\file.txt`,
			check: func(t *testing.T, got string) {
				// Should be cleaned
				if strings.Contains(got, `\.\`) || strings.Contains(got, `\..\`) {
					t.Errorf("expected dots to be cleaned, got %q", got)
				}
			},
		},
		{
			name:  "relative path",
			input: `.\test\file.txt`,
			check: func(t *testing.T, got string) {
				if !filepath.IsAbs(got) {
					t.Errorf("expected absolute path, got %q", got)
				}
			},
		},
		{
			name:  "UNC path",
			input: `\\server\share\file.txt`,
			check: func(t *testing.T, got string) {
				if !strings.HasPrefix(got, `\\`) {
					t.Errorf("expected UNC path prefix, got %q", got)
				}
			},
		},
		{
			name:  "trailing backslash",
			input: `C:\Users\test\`,
			check: func(t *testing.T, got string) {
				// filepath.Clean removes trailing slashes
				if strings.HasSuffix(got, `\`) && !strings.HasPrefix(got, `\\?\`) {
					// Note: \\?\ paths may keep trailing slash
					t.Errorf("expected no trailing backslash, got %q", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeWindowsPath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeWindowsPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

// DISABLED: func TestNormalizeWindowsPath_LongPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific tests on non-Windows platform")
	}

	// Create a path that's exactly 248 characters
	longPath := `C:\` + strings.Repeat("a", 244) + `.txt`

	got, err := NormalizeWindowsPath(longPath)
	if err != nil {
		t.Fatalf("NormalizeWindowsPath() error = %v", err)
	}

	// Should have \\?\ prefix
	if !strings.HasPrefix(got, `\\?\`) {
		t.Errorf("expected \\?\\ prefix for long path, got %q", got)
	}
}

// DISABLED: func TestIsValidWindowsPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific tests on non-Windows platform")
	}

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty path", "", false},
		{"valid absolute path", `C:\Users\test`, true},
		{"valid relative path", `.\test\file.txt`, true},
		{"path with dots", `./././tmp/test`, true}, // Should be allowed
		{"path with invalid char <", `C:\test<file.txt`, false},
		{"path with invalid char >", `C:\test>file.txt`, false},
		{"path with pipe", `C:\test|file.txt`, false},
		{"path with asterisk", `C:\test*.txt`, false},
		{"path with question mark", `C:\test?.txt`, false},
		{"multiple colons", `C:\test:file:txt`, false},
		{"colon in wrong position", `test:file.txt`, false},
		{"reserved name CON", `C:\CON`, false},
		{"reserved name PRN", `C:\PRN.txt`, false},
		{"valid drive letter", `D:\folder`, true},
		{"UNC path", `\\server\share`, true},
		{"drive-relative", `C:folder`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidWindowsPath(tt.input); got != tt.want {
				t.Errorf("IsValidWindowsPath(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// DISABLED: func TestResolveDriveRelativePath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific tests on non-Windows platform")
	}

	tests := []struct {
		name  string
		input string
		check func(t *testing.T, got string)
	}{
		{
			name:  "drive-relative C:foo",
			input: `C:foo`,
			check: func(t *testing.T, got string) {
				if !filepath.IsAbs(got) {
					t.Errorf("expected absolute path, got %q", got)
				}
				if !strings.Contains(got, "foo") {
					t.Errorf("expected path to contain foo, got %q", got)
				}
			},
		},
		{
			name:  "already absolute",
			input: `C:\Users\test`,
			check: func(t *testing.T, got string) {
				if got != `C:\Users\test` {
					t.Errorf("expected unchanged path, got %q", got)
				}
			},
		},
		{
			name:  "drive letter only",
			input: `C:`,
			check: func(t *testing.T, got string) {
				if !filepath.IsAbs(got) {
					t.Errorf("expected absolute path, got %q", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveDriveRelativePath(tt.input)
			if err != nil {
				t.Errorf("ResolveDriveRelativePath() error = %v", err)
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

// DISABLED: func TestEnsureParentDirectory(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific tests on non-Windows platform")
	}

	tempDir := t.TempDir()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name: "simple path",
			path: filepath.Join(tempDir, "subdir", "file.txt"),
		},
		{
			name: "nested path",
			path: filepath.Join(tempDir, "a", "b", "c", "file.txt"),
		},
		{
			name: "path with dots",
			path: filepath.Join(tempDir, ".", "subdir", "file.txt"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EnsureParentDirectory(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("EnsureParentDirectory() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil && tt.path != "" {
				// Check that parent directory was created
				parentDir := filepath.Dir(tt.path)
				if parentDir != "." {
					if _, err := os.Stat(parentDir); os.IsNotExist(err) {
						t.Errorf("parent directory was not created: %q", parentDir)
					}
				}
			}
		})
	}
}

// DISABLED: func TestIsExtendedLengthPath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"extended path", `\\?\C:\very\long\path`, true},
		{"UNC path", `\\server\share`, false},
		{"device path", `\\.\device`, false},
		{"regular path", `C:\path`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsExtendedLengthPath(tt.input); got != tt.want {
				t.Errorf("IsExtendedLengthPath(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// DISABLED: func TestRelaxedTraversalChecks(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific tests on non-Windows platform")
	}

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "multiple dot segments",
			input: `./././tmp/test`,
			want:  true, // Should be allowed - will normalize to tmp/test
		},
		{
			name:  "mixed dots and normal path",
			input: `./path/../file.txt`,
			want:  true, // Should be allowed - will normalize safely
		},
		{
			name:  "current directory reference",
			input: `.\current\file.txt`,
			want:  true, // Should be allowed
		},
		{
			name:  "parent directory reference",
			input: `..\parent\file.txt`,
			want:  true, // Should be allowed if it resolves to valid location
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidWindowsPath(tt.input)
			if got != tt.want {
				t.Errorf("IsValidWindowsPath(%q) = %v, want %v", tt.input, got, tt.want)
			}

			// Also test that normalization works without error
			if tt.want {
				_, err := NormalizeWindowsPath(tt.input)
				if err != nil {
					t.Errorf("NormalizeWindowsPath(%q) should not error, got: %v", tt.input, err)
				}
			}
		})
	}
}
