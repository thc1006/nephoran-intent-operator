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
			input:    "intent-测试-文件",
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

// TestSanitizeStatusFilename_WindowsReservedCharacters validates rejection of Windows-reserved characters: <>:"/\|?*
func TestSanitizeStatusFilename_WindowsReservedCharacters(t *testing.T) {
	reservedChars := []struct {
		char     string
		input    string
		expected string
	}{
		{"<", "intent<test", "intent-test"},
		{">", "intent>test", "intent-test"},
		{":", "intent:test", "intent-test"},
		{"\"", "intent\"test", "intent-test"},
		{"/", "intent/test", "intent-test"},
		{"\\", "intent\\test", "intent-test"},
		{"|", "intent|test", "intent-test"},
		{"?", "intent?test", "intent-test"},
		{"*", "intent*test", "intent-test"},
		// Multiple reserved chars
		{"<>", "intent<>test", "intent-test"},
		{"|?*", "intent|?*test", "intent-test"},
		{"all", "intent<>:\"/\\|?*test", "intent-test"},
	}

	for _, tc := range reservedChars {
		t.Run("ReservedChar_"+tc.char, func(t *testing.T) {
			result := sanitizeStatusFilename(tc.input)

			// Should not contain any reserved characters
			reservedSet := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
			for _, reserved := range reservedSet {
				assert.NotContains(t, result, reserved,
					"Result should not contain reserved character: %s", reserved)
			}

			assert.Equal(t, tc.expected, result,
				"Reserved character sanitization should match expected result")
		})
	}
}

// TestSanitizeStatusFilename_AllWindowsReservedDevices validates rejection of all Windows reserved device names
func TestSanitizeStatusFilename_AllWindowsReservedDevices(t *testing.T) {
	reservedDevices := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	}

	for _, device := range reservedDevices {
		t.Run("Device_"+device, func(t *testing.T) {
			// Test uppercase
			result := sanitizeStatusFilename(device)
			assert.Equal(t, device+"-file", result,
				"Uppercase reserved device should get -file suffix")

			// Test lowercase
			result = sanitizeStatusFilename(strings.ToLower(device))
			assert.Equal(t, strings.ToLower(device)+"-file", result,
				"Lowercase reserved device should get -file suffix")

			// Test mixed case
			mixed := strings.Title(strings.ToLower(device))
			result = sanitizeStatusFilename(mixed)
			assert.Equal(t, mixed+"-file", result,
				"Mixed case reserved device should get -file suffix")

			// Test with extension
			result = sanitizeStatusFilename(device + ".json")
			assert.Equal(t, device+"-file.json", result,
				"Reserved device with extension should get -file suffix before extension")
		})
	}
}

// TestSanitizeStatusFilename_UnicodeNormalizationNFKC tests Unicode normalization while preserving safe characters
func TestSanitizeStatusFilename_UnicodeNormalizationNFKC(t *testing.T) {
	unicodeTests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name:        "ChineseCharacters",
			input:       "intent-测�?-file",
			description: "Chinese characters should be sanitized to ASCII-safe",
		},
		{
			name:        "CyrillicCharacters",
			input:       "intent-?айл-test",
			description: "Cyrillic characters should be sanitized to ASCII-safe",
		},
		{
			name:        "JapaneseCharacters",
			input:       "intent-?�ァ?�ル-test",
			description: "Japanese characters should be sanitized to ASCII-safe",
		},
		{
			name:        "ArabicCharacters",
			input:       "intent-???-test",
			description: "Arabic characters should be sanitized to ASCII-safe",
		},
		{
			name:        "EmojiCharacters",
			input:       "intent-????-test",
			description: "Emoji should be sanitized to ASCII-safe",
		},
		{
			name:        "AccentedCharacters",
			input:       "intent-café-résumé",
			description: "Accented characters should be sanitized for strict Windows compatibility",
		},
		{
			name:        "MixedUnicodeAndASCII",
			input:       "intent-test-测�?-file",
			description: "Should preserve ASCII while sanitizing Unicode",
		},
		{
			name:        "UnicodeNormalizationCombining",
			input:       "intent-e\u0301-test", // e + combining acute accent
			description: "Unicode normalization should handle combining characters",
		},
		{
			name:        "OnlyUnicodeCharacters",
			input:       "测�??�件",
			description: "Filename with only Unicode should fall back to unnamed",
		},
	}

	for _, tc := range unicodeTests {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeStatusFilename(tc.input)

			// Should contain only ASCII characters
			for i, r := range result {
				assert.True(t, r <= 127,
					"Character at position %d should be ASCII: %c (%d) in result: %s", i, r, r, result)
			}

			// Should not be empty
			assert.NotEmpty(t, result, "Result should not be empty")

			// Should only contain allowed filename characters
			allowedPattern := "^[A-Za-z0-9._-]+$"
			assert.Regexp(t, allowedPattern, result,
				"Result should contain only allowed filename characters: %s", result)

			// Should not be just separators
			assert.NotRegexp(t, "^[-._]+$", result,
				"Result should not be only separators")
		})
	}
}

// TestSanitizeStatusFilename_StatusFilenameConsistencyWindows tests consistent generation on Windows
func TestSanitizeStatusFilename_StatusFilenameConsistencyWindows(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{"WindowsPath", "C:\\Users\\test\\intent-file"},
		{"UNCPath", "\\\\server\\share\\intent-file"},
		{"RelativePath", "..\\..\\intent-file"},
		{"PathWithSpaces", "intent file with spaces"},
		{"LongName", strings.Repeat("intent-", 20) + "file"},
		{"MixedSeparators", "intent/test\\file"},
		{"SpecialChars", "intent<>:\"|?*file"},
		{"UnicodeChars", "intent-测�?-?айл"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call multiple times to ensure consistency
			results := make([]string, 5)
			for i := 0; i < 5; i++ {
				results[i] = sanitizeStatusFilename(tc.input)
			}

			// All results should be identical
			for i := 1; i < len(results); i++ {
				assert.Equal(t, results[0], results[i],
					"Function should be deterministic for input: %s", tc.input)
			}

			result := results[0]

			// Should be valid Windows filename
			assert.NotContains(t, result, "\\", "Should not contain backslashes")
			assert.NotContains(t, result, "/", "Should not contain forward slashes")
			assert.NotContains(t, result, ":", "Should not contain colons")
			assert.NotContains(t, result, "*", "Should not contain asterisks")
			assert.NotContains(t, result, "?", "Should not contain question marks")
			assert.NotContains(t, result, "\"", "Should not contain quotes")
			assert.NotContains(t, result, "<", "Should not contain less than")
			assert.NotContains(t, result, ">", "Should not contain greater than")
			assert.NotContains(t, result, "|", "Should not contain pipe")

			// Should not exceed Windows filename length limits
			assert.LessOrEqual(t, len(result), 200,
				"Filename should not be excessively long")
		})
	}
}

// TestSanitizeStatusFilename_AllSuspiciousPatterns covers all suspicious patterns comprehensively
func TestSanitizeStatusFilename_AllSuspiciousPatterns(t *testing.T) {
	suspiciousPatterns := []struct {
		name        string
		input       string
		description string
	}{
		// Trailing patterns
		{"TrailingTilde", "intent-test~", "trailing ~ (Windows backup indicator)"},
		{"TrailingDollar", "intent-test$", "trailing $ character"},

		// File extensions that might be suspicious
		{"TmpInFilename", "intent-test.tmp", ".tmp extension"},
		{"BakInFilename", "intent-test.bak", ".bak extension"},
		{"OrigInFilename", "intent-test.orig", ".orig extension"},
		{"SwpInFilename", "intent-test.swp", ".swp extension (vim swap)"},

		// Leading patterns
		{"LeadingDotHash", ".#intent-test", "leading .# (emacs lock file)"},
		{"LeadingTilde", "~intent-test", "leading ~ character"},
		{"LeadingDot", ".intent-test", "leading dot (hidden file)"},

		// Repeated patterns
		{"RepeatedDots", "intent-test..ext", "repeated dots"},
		{"TripleDots", "intent-test...ext", "triple dots"},
		{"QuadrupleDots", "intent-test....ext", "quadruple dots"},
		{"RepeatedHyphens", "intent-test--ext", "repeated hyphens"},
		{"RepeatedUnderscores", "intent-test__ext", "repeated underscores"},

		// Whitespace issues
		{"TrailingSpace", "intent-test ", "trailing space"},
		{"LeadingSpace", " intent-test", "leading space"},
		{"MultipleSpaces", "intent  test  file", "multiple spaces"},

		// Control characters
		{"TabCharacter", "intent-test\t", "tab character"},
		{"NewlineCharacter", "intent-test\n", "newline character"},
		{"CarriageReturn", "intent-test\r", "carriage return"},
		{"VerticalTab", "intent-test\v", "vertical tab"},
		{"FormFeed", "intent-test\f", "form feed"},
		{"Backspace", "intent-test\b", "backspace"},
		{"Bell", "intent-test\a", "bell character"},

		// Mixed suspicious patterns
		{"MultiplePatterns", "~intent-test...bak$", "multiple suspicious patterns"},
		{"ComplexPattern", ".#intent..test~~.tmp.", "complex suspicious pattern"},
	}

	for _, tc := range suspiciousPatterns {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeStatusFilename(tc.input)

			// Should not contain suspicious patterns
			assert.NotContains(t, result, "~", "Should not contain ~ for: %s", tc.description)
			assert.NotRegexp(t, `\.{2,}`, result, "Should not contain repeated dots for: %s", tc.description)
			assert.False(t, strings.HasPrefix(result, ".#"), "Should not start with .# for: %s", tc.description)
			assert.False(t, strings.HasSuffix(result, "$"), "Should not end with $ for: %s", tc.description)

			// Should not have leading/trailing spaces or separators
			assert.Equal(t, strings.TrimSpace(result), result,
				"Should not have leading/trailing spaces for: %s", tc.description)
			assert.Equal(t, strings.Trim(result, "-._"), result,
				"Should not have leading/trailing separators for: %s", tc.description)

			// Should not contain control characters
			for i, r := range result {
				assert.False(t, r < 32 && r != '\t' && r != '\n' && r != '\r',
					"Should not contain control character at position %d for: %s", i, tc.description)
			}

			// Should not be empty
			assert.NotEmpty(t, result, "Should not be empty for: %s", tc.description)
		})
	}
}

// TestSanitizeStatusFilename_EdgeCasesComprehensive tests comprehensive edge cases
func TestSanitizeStatusFilename_EdgeCasesComprehensive(t *testing.T) {
	edgeCases := []struct {
		name        string
		input       string
		expectValid bool
		description string
	}{
		{"EmptyString", "", true, "empty string should get default name"},
		{"WhitespaceOnly", "   \t\n\r  ", true, "whitespace-only should get default name"},
		{"OnlyDots", "......", true, "only dots should get default name"},
		{"OnlySeparators", "---___...", true, "only separators should get default name"},
		{"OnlyReservedChars", "<>:\"/\\|?*", true, "only reserved chars should get default name"},
		{"OnlyUnicode", "测试文件", true, "only Unicode should get default name"},
		{"VeryLongFilename", strings.Repeat("a", 500), true, "very long should be truncated"},
		{"LongUnicode", strings.Repeat("测", 200), true, "long Unicode should be handled"},
		{"PathTraversal", "../../../etc/passwd", true, "path traversal should be sanitized"},
		{"WindowsPathLong", "C:\\Program Files\\Very Long Path Name\\With Spaces\\intent-file.json", true, "Windows path should be sanitized"},
		{"MixedComplexity", "C:\\测试\\<file|name>with测试*.tmp~", true, "complex mixed input should be handled"},
		{"NullBytes", "intent\x00test", true, "null bytes should be removed"},
		{"HighASCII", "intent\x7F\x80\x90test", true, "high ASCII should be sanitized"},
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeStatusFilename(tc.input)

			if tc.expectValid {
				// Should always produce a valid result
				assert.NotEmpty(t, result, "Should produce non-empty result for: %s", tc.description)

				// Should be reasonable length
				assert.LessOrEqual(t, len(result), 150,
					"Should not be excessively long for: %s", tc.description)

				// Should contain only valid characters
				for i, r := range result {
					assert.True(t, (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
						(r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_',
						"Character at position %d should be valid: %c (%d) for: %s", i, r, r, tc.description)
				}

				// Should not be problematic filenames
				assert.NotEqual(t, ".", result, "Should not be single dot")
				assert.NotEqual(t, "..", result, "Should not be double dot")

				// Should not start or end with separators
				assert.False(t, strings.HasPrefix(result, "-") || strings.HasPrefix(result, ".") || strings.HasPrefix(result, "_"),
					"Should not start with separator for: %s", tc.description)
				assert.False(t, strings.HasSuffix(result, "-") || strings.HasSuffix(result, ".") || strings.HasSuffix(result, "_"),
					"Should not end with separator for: %s", tc.description)
			}
		})
	}
}

// BenchmarkSanitizeStatusFilename_ComprehensivePerformance benchmarks various scenarios
func BenchmarkSanitizeStatusFilename_ComprehensivePerformance(b *testing.B) {
	benchmarkCases := []struct {
		name  string
		input string
	}{
		{"SimpleASCII", "intent-test-file.json"},
		{"WindowsReservedChars", "intent<>:\"/\\|?*test.json"},
		{"UnicodeHeavy", "intent-测�?-?айл-?�ァ?�ル.json"},
		{"WindowsReservedDevice", "CON.json"},
		{"VeryLongFilename", strings.Repeat("intent-test-", 50) + ".json"},
		{"ComplexMixed", "C:\\测�?\\<file|name>with??��?*.tmp~.json"},
		{"OnlyUnicode", "测�??�件?�称"},
		{"PathTraversal", "../../../etc/passwd"},
		{"ControlCharacters", "intent\t\n\r\b\a\f\vtest.json"},
		{"RepeatedPatterns", "intent...test---file___name.json"},
	}

	for _, bc := range benchmarkCases {
		b.Run(bc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				sanitizeStatusFilename(bc.input)
			}
		})
	}
}
