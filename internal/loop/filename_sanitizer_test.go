package loop

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeStatusFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		desc     string
	}{
		// Basic sanitization
		{
			name:     "AlphanumericOnly",
			input:    "intent123",
			expected: "intent123",
			desc:     "Alphanumeric characters should pass through unchanged",
		},
		{
			name:     "WithValidSeparators",
			input:    "intent-test_file.json",
			expected: "intent-test_file.json",
			desc:     "Valid separators should be preserved",
		},
		{
			name:     "InvalidCharacters",
			input:    "intent@#$%test!file",
			expected: "intent-test-file",
			desc:     "Invalid characters should be replaced with dashes",
		},
		{
			name:     "SpacesAndTabs",
			input:    "intent   test\tfile",
			expected: "intent-test-file",
			desc:     "Whitespace should be replaced with dashes",
		},
		
		// Duplicate separator handling
		{
			name:     "DuplicateDashes",
			input:    "intent---test",
			expected: "intent-test",
			desc:     "Duplicate dashes should be collapsed",
		},
		{
			name:     "DuplicateDots",
			input:    "intent...test",
			expected: "intent-test",
			desc:     "Duplicate dots should be collapsed",
		},
		{
			name:     "MixedDuplicates",
			input:    "intent.-._test",
			expected: "intent-test",
			desc:     "Mixed duplicate separators should be collapsed",
		},
		
		// Edge trimming
		{
			name:     "LeadingTrailingSeparators",
			input:    "---intent-test---",
			expected: "intent-test",
			desc:     "Leading and trailing separators should be trimmed",
		},
		{
			name:     "TrailingDots",
			input:    "intent-test...",
			expected: "intent-test",
			desc:     "Trailing dots should be removed (Windows constraint)",
		},
		{
			name:     "TrailingSpaces",
			input:    "intent-test   ",
			expected: "intent-test",
			desc:     "Trailing spaces should be removed",
		},
		
		// Windows reserved names
		{
			name:     "WindowsReservedCON",
			input:    "CON",
			expected: "CON-file",
			desc:     "Windows reserved name CON should get suffix",
		},
		{
			name:     "WindowsReservedCOM1",
			input:    "com1.json",
			expected: "com1-file.json",
			desc:     "Windows reserved name COM1 should get suffix",
		},
		{
			name:     "WindowsReservedLPT1",
			input:    "LPT1",
			expected: "LPT1-file",
			desc:     "Windows reserved name LPT1 should get suffix",
		},
		{
			name:     "WindowsReservedAUX",
			input:    "aux.txt",
			expected: "aux-file.txt",
			desc:     "Windows reserved name AUX should get suffix",
		},
		{
			name:     "WindowsReservedNUL",
			input:    "nul",
			expected: "nul-file",
			desc:     "Windows reserved name NUL should get suffix",
		},
		{
			name:     "NotReservedIfNotBaseName",
			input:    "mycon.json",
			expected: "mycon.json",
			desc:     "CON as part of name should not be affected",
		},
		
		// Empty and special cases
		{
			name:     "EmptyString",
			input:    "",
			expected: "unnamed",
			desc:     "Empty string should become 'unnamed'",
		},
		{
			name:     "OnlyInvalidChars",
			input:    "@#$%^&*()",
			expected: "unnamed",
			desc:     "String with only invalid chars should become 'unnamed'",
		},
		{
			name:     "SingleDot",
			input:    ".",
			expected: "unnamed",
			desc:     "Single dot should become 'unnamed'",
		},
		{
			name:     "DoubleDot",
			input:    "..",
			expected: "unnamed",
			desc:     "Double dot should become 'unnamed'",
		},
		
		// Length capping
		{
			name:     "VeryLongFilename",
			input:    strings.Repeat("a", 150),
			expected: strings.Repeat("a", 100),
			desc:     "Very long filenames should be truncated to 100 chars",
		},
		{
			name:     "LongWithExtension",
			input:    strings.Repeat("a", 150) + ".json",
			expected: strings.Repeat("a", 95) + ".json",
			desc:     "Long filenames with extension should preserve extension",
		},
		
		// Special characters that might appear in tests
		{
			name:     "UnicodeCharacters",
			input:    "intent-测试-файл",
			expected: "intent",
			desc:     "Unicode characters should be replaced",
		},
		{
			name:     "PathSeparators",
			input:    "intent/test\\file",
			expected: "intent-test-file",
			desc:     "Path separators should be replaced",
		},
		{
			name:     "Parentheses",
			input:    "intent(test)file",
			expected: "intent-test-file",
			desc:     "Parentheses should be replaced",
		},
		{
			name:     "SquareBrackets",
			input:    "intent[test]file",
			expected: "intent-test-file",
			desc:     "Square brackets should be replaced",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeStatusFilename(tt.input)
			assert.Equal(t, tt.expected, result, tt.desc)
			
			// Additional validations
			assert.NotContains(t, result, "//", "Should not contain double slashes")
			assert.NotContains(t, result, "\\\\", "Should not contain double backslashes")
			assert.False(t, strings.HasSuffix(result, "."), "Should not end with dot")
			assert.False(t, strings.HasSuffix(result, " "), "Should not end with space")
			assert.True(t, len(result) <= 100, "Should not exceed 100 characters")
		})
	}
}

func TestSanitizeStatusFilename_WindowsEdgeCases(t *testing.T) {
	// Test all Windows reserved names
	reservedNames := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	}

	for _, reserved := range reservedNames {
		t.Run("Reserved_"+reserved, func(t *testing.T) {
			// Test lowercase
			result := sanitizeStatusFilename(strings.ToLower(reserved))
			assert.Equal(t, strings.ToLower(reserved)+"-file", result)
			
			// Test with extension
			result = sanitizeStatusFilename(strings.ToLower(reserved) + ".json")
			assert.Equal(t, strings.ToLower(reserved)+"-file.json", result)
		})
	}
}