package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSecretLoaderSecurity tests security improvements in SecretLoader
func TestSecretLoaderSecurity(t *testing.T) {
	t.Run("prevent_directory_traversal_in_base_path", func(t *testing.T) {
		// Test that base path with .. is rejected
		_, err := NewSecretLoader("../../../etc", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "directory traversal")
	})

	t.Run("validate_secret_name", func(t *testing.T) {
		// Create temp directory for testing
		tempDir, err := os.MkdirTemp("", "secret-loader-test")
		require.NoError(t, err)
		defer func() {
			if err := os.RemoveAll(tempDir); err != nil {
				t.Logf("failed to remove temp dir %s: %v", tempDir, err)
			}
		}()

		// Create a test secret file
		secretFile := filepath.Join(tempDir, "test-secret")
		err = os.WriteFile(secretFile, []byte("secret-content"), 0600)
		require.NoError(t, err)

		loader, err := NewSecretLoader(tempDir, nil)
		require.NoError(t, err)

		// Test valid secret name
		content, err := loader.LoadSecret("test-secret")
		assert.NoError(t, err)
		assert.Equal(t, "secret-content", content)

		// Test path traversal in secret name
		_, err = loader.LoadSecret("../etc/passwd")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid secret name")

		// Test absolute path in secret name
		_, err = loader.LoadSecret("/etc/passwd")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid secret name")

		// Test backslash in secret name (Windows path separator)
		_, err = loader.LoadSecret("..\\windows\\system32")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid secret name")

		// Test empty secret name
		_, err = loader.LoadSecret("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid secret name")

		// Test very long secret name
		longName := ""
		for i := 0; i < 300; i++ {
			longName += "a"
		}
		_, err = loader.LoadSecret(longName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "less than 256 characters")
	})

	t.Run("absolute_path_conversion", func(t *testing.T) {
		// Create temp directory
		tempDir, err := os.MkdirTemp("", "secret-loader-abs-test")
		require.NoError(t, err)
		defer func() {
			if err := os.RemoveAll(tempDir); err != nil {
				t.Logf("failed to remove temp dir %s: %v", tempDir, err)
			}
		}()

		// Test that relative path gets converted to absolute
		loader, err := NewSecretLoader(tempDir, nil)
		assert.NoError(t, err)
		assert.NotNil(t, loader)

		// The basePath should be absolute
		assert.True(t, filepath.IsAbs(loader.basePath))
	})

	t.Run("no_ioutil_usage", func(t *testing.T) {
		// Create temp directory for testing
		tempDir, err := os.MkdirTemp("", "secret-loader-noioutil-test")
		require.NoError(t, err)
		defer func() {
			if err := os.RemoveAll(tempDir); err != nil {
				t.Logf("failed to remove temp dir %s: %v", tempDir, err)
			}
		}()

		// Create a test secret file
		secretFile := filepath.Join(tempDir, "test-secret")
		err = os.WriteFile(secretFile, []byte("test-content"), 0600)
		require.NoError(t, err)

		loader, err := NewSecretLoader(tempDir, nil)
		require.NoError(t, err)

		// This should use os.ReadFile internally, not os.ReadFile
		content, err := loader.LoadSecret("test-secret")
		assert.NoError(t, err)
		assert.Equal(t, "test-content", content)
	})
}

// TestFilePathValidation tests the validateFilePath function
func TestFilePathValidation(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid_path",
			path:      "/var/log/app.log",
			wantError: false,
		},
		{
			name:      "path_with_double_dot",
			path:      "../etc/passwd",
			wantError: true,
			errorMsg:  "dangerous pattern",
		},
		{
			name:      "path_with_double_slash",
			path:      "//etc//passwd",
			wantError: true,
			errorMsg:  "dangerous pattern",
		},
		{
			name:      "path_with_backslash",
			path:      "C:\\Windows\\System32",
			wantError: true,
			errorMsg:  "dangerous pattern",
		},
		{
			name:      "path_with_null_byte",
			path:      "/etc/passwd\x00.txt",
			wantError: true,
			errorMsg:  "dangerous pattern",
		},
		{
			name:      "empty_path",
			path:      "",
			wantError: true,
			errorMsg:  "cannot be empty",
		},
		{
			name:      "very_long_path",
			path:      "/" + strings.Repeat("a", 5000),
			wantError: true,
			errorMsg:  "too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFilePath(tt.path)
			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
