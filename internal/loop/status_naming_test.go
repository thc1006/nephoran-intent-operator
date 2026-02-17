package loop

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestComputeStatusFileName(t *testing.T) {
	// Fixed timestamp for consistent testing
	fixedTime := time.Date(2025, 8, 21, 14, 30, 22, 0, time.UTC)
	expectedTimestamp := "20250821-143022"

	testCases := []struct {
		name             string
		srcPath          string
		expectedFilename string
		description      string
	}{
		{
			name:             "SingleDigitIndex",
			srcPath:          "intent-test-0.json",
			expectedFilename: "intent-test-0-" + expectedTimestamp + ".status",
			description:      "Should handle single digit indices correctly",
		},
		{
			name:             "DoubleDigitIndex",
			srcPath:          "intent-test-10.json",
			expectedFilename: "intent-test-10-" + expectedTimestamp + ".status",
			description:      "Should handle double digit indices correctly",
		},
		{
			name:             "TripleDigitIndex",
			srcPath:          "intent-test-100.json",
			expectedFilename: "intent-test-100-" + expectedTimestamp + ".status",
			description:      "Should handle triple digit indices correctly",
		},
		{
			name:             "FileWithoutExtension",
			srcPath:          "intent-no-ext",
			expectedFilename: "intent-no-ext-" + expectedTimestamp + ".status",
			description:      "Should handle files without extensions",
		},
		{
			name:             "FileWithMultipleDots",
			srcPath:          "intent.scale.test.json",
			expectedFilename: "intent.scale.test-" + expectedTimestamp + ".status",
			description:      "Should handle files with multiple dots correctly",
		},
		{
			name:             "AbsolutePath",
			srcPath:          "/path/to/intent-example.json",
			expectedFilename: "intent-example-" + expectedTimestamp + ".status",
			description:      "Should extract basename from absolute paths",
		},
		{
			name:    "WindowsAbsolutePath",
			srcPath: "C:\\Users\\test\\intent-windows.json",
			// On Linux, filepath.Base("C:\\Users\\test\\intent-windows.json") returns the
			// whole string unchanged (backslash is not a path separator on Linux).
			// filepath.Ext then strips ".json", leaving "C:\\Users\\test\\intent-windows".
			// sanitizeStatusFilename replaces ':', '\', and other non-[A-Za-z0-9._-] chars
			// with '-', collapsing consecutive '-' → "C-Users-test-intent-windows".
			expectedFilename: "C-Users-test-intent-windows-" + expectedTimestamp + ".status",
			description:      "Should handle Windows absolute paths (sanitized on Linux)",
		},
		{
			name:             "RelativePath",
			srcPath:          "subdir/intent-relative.json",
			expectedFilename: "intent-relative-" + expectedTimestamp + ".status",
			description:      "Should extract basename from relative paths",
		},
		{
			name:             "FileWithUnderscores",
			srcPath:          "intent_deployment_config.json",
			expectedFilename: "intent_deployment_config-" + expectedTimestamp + ".status",
			description:      "Should handle underscores in filenames",
		},
		{
			name:             "FileWithHyphens",
			srcPath:          "intent-scale-up-down.json",
			expectedFilename: "intent-scale-up-down-" + expectedTimestamp + ".status",
			description:      "Should handle multiple hyphens in filenames",
		},
		{
			name:             "FileWithNumbers",
			srcPath:          "intent-123-abc-456.json",
			expectedFilename: "intent-123-abc-456-" + expectedTimestamp + ".status",
			description:      "Should handle mixed numbers and letters",
		},
		{
			name:             "EmptyExtension",
			srcPath:          "intent-file.",
			expectedFilename: "intent-file-" + expectedTimestamp + ".status",
			description:      "Should handle files ending with a dot",
		},
		{
			name:             "NonJsonExtension",
			srcPath:          "intent-config.yaml",
			expectedFilename: "intent-config-" + expectedTimestamp + ".status",
			description:      "Should handle non-JSON file extensions",
		},
		{
			name:             "SpecialCharacters",
			srcPath:          "intent-test@special#chars.json",
			expectedFilename: "intent-test-special-chars-" + expectedTimestamp + ".status",
			description:      "Should sanitize special characters in filenames for cross-platform compatibility",
		},
		{
			name:             "UnicodeCharacters",
			srcPath:          "intent-测试-файл.json",
			expectedFilename: "intent-" + expectedTimestamp + ".status",
			description:      "Should sanitize Unicode characters for ASCII-only cross-platform compatibility",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ComputeStatusFileName(tc.srcPath, fixedTime)
			assert.Equal(t, tc.expectedFilename, result, tc.description)

			// Verify that the result always ends with .status
			assert.True(t, len(result) > 7, "Status filename should be longer than '.status'")
			assert.True(t, result[len(result)-7:] == ".status", "Status filename should end with '.status'")

			// Verify timestamp format is present
			assert.Contains(t, result, expectedTimestamp, "Status filename should contain the formatted timestamp")
		})
	}
}

func TestComputeStatusFileNameTimestampFormat(t *testing.T) {
	testCases := []struct {
		name      string
		timestamp time.Time
		expected  string
	}{
		{
			name:      "NewYear",
			timestamp: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			expected:  "intent-test-20250101-000000.status",
		},
		{
			name:      "MidYear",
			timestamp: time.Date(2025, 6, 15, 12, 30, 45, 0, time.UTC),
			expected:  "intent-test-20250615-123045.status",
		},
		{
			name:      "YearEnd",
			timestamp: time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC),
			expected:  "intent-test-20251231-235959.status",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ComputeStatusFileName("intent-test.json", tc.timestamp)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestComputeStatusFileNameConsistency(t *testing.T) {
	// Test that the same input always produces the same output
	srcPath := "intent-consistency-test.json"
	timestamp := time.Date(2025, 8, 21, 14, 30, 22, 0, time.UTC)

	expected := "intent-consistency-test-20250821-143022.status"

	// Call the function multiple times
	for i := 0; i < 10; i++ {
		result := ComputeStatusFileName(srcPath, timestamp)
		assert.Equal(t, expected, result, "Function should be deterministic")
	}
}

func TestComputeStatusFileNameEdgeCases(t *testing.T) {
	fixedTime := time.Date(2025, 8, 21, 14, 30, 22, 0, time.UTC)

	testCases := []struct {
		name        string
		srcPath     string
		expectPanic bool
		description string
	}{
		{
			name:        "EmptyPath",
			srcPath:     "",
			expectPanic: false,
			description: "Should handle empty path gracefully",
		},
		{
			name:        "JustFilename",
			srcPath:     "intent.json",
			expectPanic: false,
			description: "Should handle just a filename",
		},
		{
			name:        "PathWithSpaces",
			srcPath:     "intent with spaces.json",
			expectPanic: false,
			description: "Should handle paths with spaces",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectPanic {
				assert.Panics(t, func() {
					ComputeStatusFileName(tc.srcPath, fixedTime)
				}, tc.description)
			} else {
				assert.NotPanics(t, func() {
					result := ComputeStatusFileName(tc.srcPath, fixedTime)
					// Basic validation that we got a reasonable result
					assert.Contains(t, result, "20250821-143022", "Should contain timestamp")
					assert.True(t, len(result) >= 20, "Should be reasonable length")
					assert.True(t, result[len(result)-7:] == ".status", "Should end with .status")
				}, tc.description)
			}
		})
	}
}

func TestComputeStatusFileNameExtensionHandling(t *testing.T) {
	fixedTime := time.Date(2025, 8, 21, 14, 30, 22, 0, time.UTC)

	testCases := []struct {
		name     string
		srcPath  string
		expected string
	}{
		{
			name:     "StandardJson",
			srcPath:  "file.json",
			expected: "file-20250821-143022.status",
		},
		{
			name:     "MultipleExtensions",
			srcPath:  "file.backup.json",
			expected: "file.backup-20250821-143022.status",
		},
		{
			name:     "NoExtension",
			srcPath:  "file",
			expected: "file-20250821-143022.status",
		},
		{
			name:     "DotOnly",
			srcPath:  "file.",
			expected: "file-20250821-143022.status",
		},
		{
			name:     "LongExtension",
			srcPath:  "file.configuration",
			expected: "file-20250821-143022.status",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ComputeStatusFileName(tc.srcPath, fixedTime)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkComputeStatusFileName(b *testing.B) {
	timestamp := time.Now()
	srcPath := "intent-benchmark-test.json"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ComputeStatusFileName(srcPath, timestamp)
	}
}

func BenchmarkComputeStatusFileNameLongPath(b *testing.B) {
	timestamp := time.Now()
	srcPath := filepath.Join("very", "long", "nested", "path", "structure", "with", "many", "directories", "intent-benchmark-test.json")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ComputeStatusFileName(srcPath, timestamp)
	}
}
