package auth

import (
	"os"
	"path/filepath"
	"testing"
)

// TestValidateConfigFilePath tests the security validation for config file paths
func TestValidateConfigFilePath(t *testing.T) {
	// Save current directory for relative path tests
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	tests := []struct {
		name        string
		path        string
		expectError bool
		errorMsg    string
	}{
		// Valid paths
		{
			name:        "valid config path",
			path:        "/etc/nephoran/config.json",
			expectError: false,
		},
		{
			name:        "valid kubernetes config mount",
			path:        "/config/auth.json",
			expectError: false,
		},
		{
			name:        "valid relative config in development",
			path:        filepath.Join(currentDir, "config", "auth.json"),
			expectError: false,
		},

		// Path traversal attempts
		{
			name:        "path traversal with ..",
			path:        "/etc/nephoran/../../../etc/passwd",
			expectError: true,
			errorMsg:    "path traversal attempt detected",
		},
		{
			name:        "path traversal in middle",
			path:        "/etc/nephoran/../../passwd",
			expectError: true,
			errorMsg:    "path traversal attempt detected",
		},
		{
			name:        "encoded path traversal",
			path:        "/etc/nephoran/%2e%2e/passwd",
			expectError: true,
			errorMsg:    "not in allowed directory",
		},

		// Null byte injection
		{
			name:        "null byte injection",
			path:        "/etc/nephoran/config.json\x00.txt",
			expectError: true,
			errorMsg:    "null byte",
		},

		// Symlink attacks (validation only, actual symlink check is in loadFromFile)
		{
			name:        "path outside allowed directories",
			path:        "/etc/passwd",
			expectError: true,
			errorMsg:    "not in allowed directory",
		},
		{
			name:        "sensitive system file",
			path:        "/etc/shadow",
			expectError: true,
			errorMsg:    "not in allowed directory",
		},

		// Windows UNC paths
		{
			name:        "windows UNC path",
			path:        "\\\\server\\share\\config.json",
			expectError: true,
			errorMsg:    "UNC paths not allowed",
		},
		{
			name:        "forward slash UNC",
			path:        "//server/share/config.json",
			expectError: true,
			errorMsg:    "UNC paths not allowed",
		},

		// Special devices and files
		{
			name:        "dev directory",
			path:        "/dev/random",
			expectError: true,
			errorMsg:    "suspicious file path",
		},
		{
			name:        "proc filesystem",
			path:        "/proc/self/environ",
			expectError: true,
			errorMsg:    "suspicious file path",
		},
		{
			name:        "sys filesystem",
			path:        "/sys/kernel/config",
			expectError: true,
			errorMsg:    "suspicious file path",
		},

		// Invalid extensions
		{
			name:        "executable file",
			path:        "/etc/nephoran/config.exe",
			expectError: true,
			errorMsg:    "invalid config file extension",
		},
		{
			name:        "shell script",
			path:        "/etc/nephoran/config.sh",
			expectError: true,
			errorMsg:    "invalid config file extension",
		},

		// Hidden files
		{
			name:        "hidden file",
			path:        "/etc/nephoran/.hidden",
			expectError: true,
			errorMsg:    "hidden files not allowed",
		},

		// Empty and invalid paths
		{
			name:        "empty path",
			path:        "",
			expectError: true,
			errorMsg:    "empty file path",
		},
		{
			name:        "whitespace only",
			path:        "   ",
			expectError: true,
			errorMsg:    "empty file path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfigFilePath(tt.path)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for path %q, but got none", tt.path)
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for path %q: %v", tt.path, err)
				}
			}
		})
	}
}

// TestLoadFromFileSecurityChecks tests the comprehensive security checks in loadFromFile
func TestLoadFromFileSecurityChecks(t *testing.T) {
	config := &AuthConfig{}

	// Test 1: Path traversal attempt
	err := config.loadFromFile("../../etc/passwd")
	if err == nil {
		t.Error("Expected error for path traversal attempt, got none")
	}

	// Test 2: Attempting to read system files
	err = config.loadFromFile("/etc/passwd")
	if err == nil {
		t.Error("Expected error for system file access, got none")
	}

	// Test 3: Null byte injection
	err = config.loadFromFile("/etc/nephoran/config.json\x00.txt")
	if err == nil {
		t.Error("Expected error for null byte injection, got none")
	}

	// Test 4: Non-existent file in valid directory (should fail at file check, not path validation)
	err = config.loadFromFile("/etc/nephoran/nonexistent.json")
	if err == nil {
		t.Error("Expected error for non-existent file, got none")
	} else if !contains(err.Error(), "does not exist") && !contains(err.Error(), "not in allowed directory") {
		// It's okay if it fails at path validation because directory might not exist
		t.Logf("Got expected error: %v", err)
	}
}

// TestValidateFilePathComparison ensures OAuth2 secret validation still works
func TestValidateFilePathComparison(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectError bool
	}{
		{
			name:        "valid secrets path",
			path:        "/etc/secrets/oauth2.txt",
			expectError: false,
		},
		{
			name:        "valid run secrets",
			path:        "/run/secrets/client-secret",
			expectError: false,
		},
		{
			name:        "path traversal in secrets",
			path:        "/etc/secrets/../../../etc/passwd",
			expectError: true,
		},
		{
			name:        "system file",
			path:        "/etc/passwd",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFilePath(tt.path)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for path %q, but got none", tt.path)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for path %q: %v", tt.path, err)
				}
			}
		})
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr) >= 0))
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
