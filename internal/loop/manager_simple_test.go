package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileManager_BasicOperations(t *testing.T) {
	dir := t.TempDir()
	fm, err := NewFileManager(dir)
	require.NoError(t, err)

	// Test successful file move to processed
	t.Run("move to processed", func(t *testing.T) {
		testFile := filepath.Join(dir, "test-intent.json")
		testContent := `{"action": "scale", "target": "deployment", "count": 3}`
		require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

		err := fm.MoveToProcessed(testFile)
		require.NoError(t, err)

		// Original file should not exist
		assert.NoFileExists(t, testFile)

		// File should exist in processed directory
		targetPath := filepath.Join(fm.processedDir, "test-intent.json")
		assert.FileExists(t, targetPath)
	})

	// Test successful file move to failed
	t.Run("move to failed", func(t *testing.T) {
		testFile := filepath.Join(dir, "failed-intent.json")
		testContent := `{"action": "scale", "target": "deployment", "count": 3}`
		require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

		errorMsg := "porch command failed: exit code 1"
		err := fm.MoveToFailed(testFile, errorMsg)
		require.NoError(t, err)

		// Original file should not exist
		assert.NoFileExists(t, testFile)

		// File should exist in failed directory
		targetPath := filepath.Join(fm.failedDir, "failed-intent.json")
		assert.FileExists(t, targetPath)

		// Error log should exist
		logPath := filepath.Join(fm.failedDir, "failed-intent.json.error.log")
		assert.FileExists(t, logPath)
	})

	// Test stats
	t.Run("get stats", func(t *testing.T) {
		stats, err := fm.GetStats()
		require.NoError(t, err)
		assert.Equal(t, 1, stats.ProcessedCount)
		assert.Equal(t, 1, stats.FailedCount)
	})
}

func TestFileManager_ErrorScenarios(t *testing.T) {
	dir := t.TempDir()
	fm, err := NewFileManager(dir)
	require.NoError(t, err)

	// Test moving nonexistent file
	t.Run("move nonexistent file", func(t *testing.T) {
		nonexistentFile := filepath.Join(dir, "nonexistent.json")
		err := fm.MoveToProcessed(nonexistentFile)
		assert.Error(t, err)
	})
}

func TestSanitizeErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal message",
			input:    "porch command failed: exit code 1",
			expected: "porch command failed: exit code 1",
		},
		{
			name:     "empty message",
			input:    "",
			expected: "[error message sanitized]",
		},
		{
			name:     "whitespace only",
			input:    "   \n  \t  ",
			expected: "[error message sanitized]",
		},
		{
			name:     "message with null bytes",
			input:    "error\x00message\x00here",
			expected: "error[?]message[?]here",
		},
		{
			name:     "message with path traversal",
			input:    "failed to read ../../../etc/passwd",
			expected: "failed to read etc/passwd",
		},
		{
			name:     "message with windows path traversal",
			input:    "error accessing ..\\..\\windows\\system32",
			expected: "error accessing windows\\system32",
		},
		{
			name:     "message with control characters",
			input:    "error\x01\x02\x03message\x1f",
			expected: "error[?][?][?]message[?]",
		},
		{
			name:     "message with newlines and tabs (preserved)",
			input:    "error line 1\nline 2\tcolumn",
			expected: "error line 1\nline 2\tcolumn",
		},
		{
			name:     "message with excessive newlines",
			input:    "error\n\n\n\n\nmessage",
			expected: "error\n\nmessage",
		},
		{
			name:     "overly long message",
			input:    string(make([]byte, 1100)) + "end",
			expected: strings.Repeat("[?]", 341) + "[...[truncated]", // 1100 null bytes + "end" -> 1103*3 chars -> truncated at 1024: 341 complete "[?]" + 1 "[" = 1024 chars
		},
		{
			name:     "complex malicious message",
			input:    "error\x00accessing\x01../../../etc/passwd\x02with\x03control\nchars\n\n\n\nand\ttabs",
			expected: "error[?]accessing[?]etc/passwd[?]with[?]control\nchars\n\nand\ttabs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeErrorMessage(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFileManager_ErrorSanitization(t *testing.T) {
	dir := t.TempDir()
	fm, err := NewFileManager(dir)
	require.NoError(t, err)

	// Create a test file
	testFile := filepath.Join(dir, "test-intent.json")
	testContent := `{"action": "scale", "target": "deployment", "count": 3}`
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

	// Test with malicious error message
	maliciousError := "access denied\x00for\x01../../../etc/passwd\x02\n\n\n\nwith control chars"
	err = fm.MoveToFailed(testFile, maliciousError)
	require.NoError(t, err)

	// Read the error log to verify sanitization
	logPath := filepath.Join(fm.failedDir, "test-intent.json.error.log")
	logContent, err := os.ReadFile(logPath)
	require.NoError(t, err)

	logStr := string(logContent)
	// Should not contain null bytes
	assert.NotContains(t, logStr, "\x00")
	// Should not contain other control characters (except \n, \t, \r)
	assert.NotContains(t, logStr, "\x01")
	assert.NotContains(t, logStr, "\x02")
	// Should not contain path traversal
	assert.NotContains(t, logStr, "../../../")
	// Should contain sanitized message
	assert.Contains(t, logStr, "access denied[?]for[?]etc/passwd[?]")
}
