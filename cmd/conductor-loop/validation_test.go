package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateHandoffDir(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T) string // Returns the path to test
		wantError bool
		errorContains string
	}{
		{
			name: "empty path",
			setupFunc: func(t *testing.T) string {
				return ""
			},
			wantError: true,
			errorContains: "path cannot be empty",
		},
		{
			name: "existing valid directory",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				testDir := filepath.Join(tempDir, "valid-dir")
				require.NoError(t, os.MkdirAll(testDir, 0755))
				return testDir
			},
			wantError: false,
		},
		{
			name: "existing file (not directory)",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				testFile := filepath.Join(tempDir, "test-file.txt")
				require.NoError(t, // FIXME: Adding error check per errcheck linter
 _ = os.WriteFile(testFile, []byte("test"), 0644))
				return testFile
			},
			wantError: true,
			errorContains: "exists but is not a directory",
		},
		{
			name: "non-existing path with valid parent",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				return filepath.Join(tempDir, "new-dir")
			},
			wantError: false,
		},
		{
			name: "deeply nested non-existing path with valid parent",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				return filepath.Join(tempDir, "level1", "level2", "level3")
			},
			wantError: false,
		},
		{
			name: "path with spaces",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				testDir := filepath.Join(tempDir, "dir with spaces")
				require.NoError(t, os.MkdirAll(testDir, 0755))
				return testDir
			},
			wantError: false,
		},
		{
			name: "path with special characters",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				testDir := filepath.Join(tempDir, "dir-with_special.chars")
				require.NoError(t, os.MkdirAll(testDir, 0755))
				return testDir
			},
			wantError: false,
		},
		{
			name: "relative path that exists",
			setupFunc: func(t *testing.T) string {
				// Use current directory as a known existing directory
				return "."
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setupFunc(t)
			
			err := validateHandoffDir(path)
			
			if tt.wantError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateHandoffDir_PlatformSpecific(t *testing.T) {
	switch runtime.GOOS {
	case "windows":
		t.Run("windows_invalid_drive", func(t *testing.T) {
			// Test invalid drive letter on Windows
			err := validateHandoffDir("Z:\\nonexistent\\deeply\\nested\\invalid\\directory")
			assert.Error(t, err)
			// The error should indicate invalid path or parent directory
			errorMsg := err.Error()
			assert.True(t, 
				strings.Contains(errorMsg, "invalid parent directory") || 
				strings.Contains(errorMsg, "cannot access path") ||
				strings.Contains(errorMsg, "invalid path"),
				"Error message should indicate invalid path or parent directory, got: %s", errorMsg)
		})
		
		t.Run("windows_unc_path", func(t *testing.T) {
			// Test UNC path handling (should work if accessible)
			err := validateHandoffDir("\\\\localhost\\c$")
			// This might work or fail depending on system setup, but shouldn't panic
			// We just want to ensure it doesn't crash
			t.Logf("UNC path validation result: %v", err)
		})
		
		t.Run("windows_reserved_names", func(t *testing.T) {
			tempDir := t.TempDir()
			// Windows reserved names should be handled gracefully
			reservedPath := filepath.Join(tempDir, "CON")
			err := validateHandoffDir(reservedPath)
			// Exact behavior may vary, but it shouldn't panic
			t.Logf("Reserved name validation result: %v", err)
		})
		
	case "linux":
		t.Run("linux_permission_denied", func(t *testing.T) {
			// Create a directory without read permissions
			tempDir := t.TempDir()
			restrictedDir := filepath.Join(tempDir, "restricted")
			require.NoError(t, os.MkdirAll(restrictedDir, 0000))
			
			// Restore permissions for cleanup
			defer func() {
				// FIXME: Adding error check per errcheck linter

				_ = os.Chmod(restrictedDir, 0755)
			}()
			
			err := validateHandoffDir(restrictedDir)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "not readable")
		})
		
		t.Run("linux_root_path", func(t *testing.T) {
			// Test root directory validation
			err := validateHandoffDir("/")
			assert.NoError(t, err) // Root should be accessible
		})
		
	case "darwin":
		t.Run("darwin_case_sensitivity", func(t *testing.T) {
			tempDir := t.TempDir()
			testDir := filepath.Join(tempDir, "TestDir")
			require.NoError(t, os.MkdirAll(testDir, 0755))
			
			// Test with different case (macOS is typically case-insensitive)
			err := validateHandoffDir(filepath.Join(tempDir, "testdir"))
			// May or may not error depending on filesystem
			t.Logf("Case sensitivity test result: %v", err)
		})
	}
}

func TestValidateHandoffDir_EdgeCases(t *testing.T) {
	t.Run("symlink_to_directory", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Symlink test skipped on Windows due to permission requirements")
		}
		
		tempDir := t.TempDir()
		realDir := filepath.Join(tempDir, "real-dir")
		symlinkDir := filepath.Join(tempDir, "symlink-dir")
		
		require.NoError(t, os.MkdirAll(realDir, 0755))
		require.NoError(t, os.Symlink(realDir, symlinkDir))
		
		err := validateHandoffDir(symlinkDir)
		assert.NoError(t, err) // Symlinks to directories should be valid
	})
	
	t.Run("symlink_to_file", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Symlink test skipped on Windows due to permission requirements")
		}
		
		tempDir := t.TempDir()
		realFile := filepath.Join(tempDir, "real-file.txt")
		symlinkFile := filepath.Join(tempDir, "symlink-file")
		
		require.NoError(t, // FIXME: Adding error check per errcheck linter
 _ = os.WriteFile(realFile, []byte("test"), 0644))
		require.NoError(t, os.Symlink(realFile, symlinkFile))
		
		err := validateHandoffDir(symlinkFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exists but is not a directory")
	})
	
	t.Run("very_long_path", func(t *testing.T) {
		tempDir := t.TempDir()
		
		// Create a very long path
		var pathParts []string
		pathParts = append(pathParts, tempDir)
		
		// Add many directory levels
		for i := 0; i < 20; i++ {
			pathParts = append(pathParts, "very-long-directory-name-that-tests-path-length-limits")
		}
		
		longPath := filepath.Join(pathParts...)
		
		// This might fail on some systems due to path length limits
		err := validateHandoffDir(longPath)
		
		// We don't assert success/failure as it depends on the filesystem
		// but we ensure it doesn't panic
		t.Logf("Long path validation result: %v", err)
	})
	
	t.Run("path_with_dots", func(t *testing.T) {
		tempDir := t.TempDir()
		
		// Test paths with . and .. components
		testPaths := []string{
			filepath.Join(tempDir, ".", "test-dir"),
			filepath.Join(tempDir, "test-dir", "..", "test-dir"),
			filepath.Join(tempDir, "..", filepath.Base(tempDir), "test-dir"),
		}
		
		for _, path := range testPaths {
			t.Run("path_"+strings.ReplaceAll(path, string(filepath.Separator), "_"), func(t *testing.T) {
				err := validateHandoffDir(path)
				// Should work since it resolves to valid locations
				t.Logf("Path with dots (%s) validation result: %v", path, err)
			})
		}
	})
}

func TestValidateHandoffDir_Integration(t *testing.T) {
	t.Run("real_workflow_simulation", func(t *testing.T) {
		tempDir := t.TempDir()
		
		// Simulate the real workflow: handoff directory and error directory
		handoffDir := filepath.Join(tempDir, "handoff")
		errorDir := filepath.Join(tempDir, "handoff", "errors")
		
		// Validate handoff directory (should work - parent exists)
		err := validateHandoffDir(handoffDir)
		assert.NoError(t, err)
		
		// Create the handoff directory
		require.NoError(t, os.MkdirAll(handoffDir, 0755))
		
		// Validate error directory (should work - parent now exists)
		err = validateHandoffDir(errorDir)
		assert.NoError(t, err)
		
		// Create the error directory
		require.NoError(t, os.MkdirAll(errorDir, 0755))
		
		// Both directories should now be accessible
		err = validateHandoffDir(handoffDir)
		assert.NoError(t, err)
		
		err = validateHandoffDir(errorDir)
		assert.NoError(t, err)
	})
	
	t.Run("invalid_then_valid_creation", func(t *testing.T) {
		// Test invalid path first
		invalidPath := ""
		if runtime.GOOS == "windows" {
			invalidPath = "Z:\\nonexistent\\path"
		} else {
			invalidPath = "/nonexistent/deeply/nested/path"
		}
		
		err := validateHandoffDir(invalidPath)
		assert.Error(t, err)
		
		// Then test valid path creation
		tempDir := t.TempDir()
		validPath := filepath.Join(tempDir, "valid", "path")
		
		err = validateHandoffDir(validPath)
		assert.NoError(t, err)
		
		// Should be able to create it
		require.NoError(t, os.MkdirAll(validPath, 0755))
		
		// And validate again
		err = validateHandoffDir(validPath)
		assert.NoError(t, err)
	})
}

// Benchmark the validation function
func BenchmarkValidateHandoffDir(b *testing.B) {
	tempDir := b.TempDir()
	testDir := filepath.Join(tempDir, "benchmark-dir")
	require.NoError(b, os.MkdirAll(testDir, 0755))
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validateHandoffDir(testDir)
	}
}

func BenchmarkValidateHandoffDir_NonExisting(b *testing.B) {
	tempDir := b.TempDir()
	testDir := filepath.Join(tempDir, "non-existing-dir")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validateHandoffDir(testDir)
	}
}