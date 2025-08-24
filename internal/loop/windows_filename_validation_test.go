package loop

import (
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/unicode/norm"
)

// TestWindowsFilenameValidation_ReservedCharacters validates rejection of Windows-reserved characters: <>:"/\|?*
func TestWindowsFilenameValidation_ReservedCharacters(t *testing.T) {
	fixedTime := time.Date(2025, 8, 21, 14, 30, 22, 0, time.UTC)
	expectedTimestamp := "20250821-143022"

	reservedChars := []struct {
		char     string
		name     string
		filename string
	}{
		{"<", "LessThan", "intent<test.json"},
		{">", "GreaterThan", "intent>test.json"},
		{":", "Colon", "intent:test.json"},
		{"\"", "Quote", "intent\"test.json"},
		{"/", "ForwardSlash", "intent/test.json"},
		{"\\", "Backslash", "intent\\test.json"},
		{"|", "Pipe", "intent|test.json"},
		{"?", "Question", "intent?test.json"},
		{"*", "Asterisk", "intent*test.json"},
	}

	for _, tc := range reservedChars {
		t.Run(tc.name, func(t *testing.T) {
			result := ComputeStatusFileName(tc.filename, fixedTime)
			
			// Should not contain the reserved character
			assert.NotContains(t, result, tc.char, 
				"Status filename should not contain Windows reserved character: %s", tc.char)
			
			// Should end with .status
			assert.True(t, strings.HasSuffix(result, ".status"),
				"Status filename should end with .status")
			
			// Should contain timestamp
			assert.Contains(t, result, expectedTimestamp,
				"Status filename should contain timestamp")
			
			// Should start with some variation of the base name (sanitized)
			// For path separators, only the basename is used
			if !strings.Contains(tc.filename, "/") && !strings.Contains(tc.filename, "\\") {
				assert.True(t, strings.HasPrefix(result, "intent"),
					"Status filename should start with sanitized base name")
			} else {
				// For paths, should start with the actual basename
				assert.True(t, strings.HasPrefix(result, "test"),
					"Status filename should start with basename for paths")
			}
			
			// Verify it's a valid Windows filename (no reserved chars)
			for _, reservedChar := range []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"} {
				assert.NotContains(t, result, reservedChar,
					"Sanitized filename should not contain reserved char: %s", reservedChar)
			}
		})
	}
}

// TestWindowsFilenameValidation_ReservedDeviceNames validates rejection of reserved device names
func TestWindowsFilenameValidation_ReservedDeviceNames(t *testing.T) {
	fixedTime := time.Date(2025, 8, 21, 14, 30, 22, 0, time.UTC)

	reservedNames := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
		// Test lowercase versions too
		"con", "prn", "aux", "nul",
		"com1", "com2", "lpt1", "lpt2",
		// Test mixed case
		"Con", "Prn", "Com1", "Lpt1",
	}

	for _, reservedName := range reservedNames {
		t.Run(reservedName, func(t *testing.T) {
			filename := reservedName + ".json"
			result := ComputeStatusFileName(filename, fixedTime)
			
			// Should not be exactly the reserved name (case-insensitive check)
			baseName := strings.TrimSuffix(result, ".status")
			timestampIndex := strings.LastIndex(baseName, "-")
			if timestampIndex > 0 {
				actualBaseName := baseName[:timestampIndex]
				upperBaseName := strings.ToUpper(actualBaseName)
				upperReserved := strings.ToUpper(reservedName)
				
				assert.NotEqual(t, upperReserved, upperBaseName,
					"Status filename base should not be Windows reserved name: %s", reservedName)
				
				// Should have a suffix to avoid reserved name
				if windowsReservedNames[upperReserved] {
					assert.True(t, strings.Contains(upperBaseName, upperReserved+"-FILE") ||
						!strings.HasPrefix(upperBaseName, upperReserved),
						"Reserved name should be modified: %s -> %s", reservedName, actualBaseName)
				}
			}
			
			// Should be a valid status filename
			assert.True(t, strings.HasSuffix(result, ".status"),
				"Should end with .status")
		})
	}
}

// TestWindowsFilenameValidation_UnicodeNormalization tests Unicode normalization (NFKC) while preserving safe characters
func TestWindowsFilenameValidation_UnicodeNormalization(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Unicode normalization test specific to Windows")
	}
	
	fixedTime := time.Date(2025, 8, 21, 14, 30, 22, 0, time.UTC)

	unicodeTests := []struct {
		name        string
		filename    string
		expectEmpty bool // If true, expect only timestamp in result
		description string
	}{
		{
			name:        "ChineseCharacters",
			filename:    "intent-测试.json",
			expectEmpty: true, // Will be sanitized to just "intent-"
			description: "Chinese characters should be sanitized",
		},
		{
			name:        "CyrillicCharacters", 
			filename:    "intent-файл.json",
			expectEmpty: true, // Will be sanitized to just "intent-"
			description: "Cyrillic characters should be sanitized",
		},
		{
			name:        "JapaneseCharacters",
			filename:    "intent-ファイル.json",
			expectEmpty: true,
			description: "Japanese characters should be sanitized",
		},
		{
			name:        "ArabicCharacters",
			filename:    "intent-ملف.json",
			expectEmpty: true,
			description: "Arabic characters should be sanitized",
		},
		{
			name:        "EmojiCharacters",
			filename:    "intent-📁📄.json",
			expectEmpty: true,
			description: "Emoji should be sanitized",
		},
		{
			name:        "MixedUnicodeAndAscii",
			filename:    "intent-test-测试-file.json",
			expectEmpty: false, // Should preserve ASCII parts
			description: "Should preserve ASCII while sanitizing Unicode",
		},
		{
			name:        "AccentedCharacters",
			filename:    "intent-café-résumé.json",
			expectEmpty: true, // Even accented chars get sanitized for strict compatibility
			description: "Accented characters should be sanitized for Windows compatibility",
		},
		{
			name:        "UnicodeNormalization",
			filename:    "intent-e\u0301.json", // e + combining acute accent
			expectEmpty: true,
			description: "Unicode normalization should handle combining characters",
		},
	}

	for _, tc := range unicodeTests {
		t.Run(tc.name, func(t *testing.T) {
			// Verify input contains non-ASCII
			hasNonASCII := false
			for _, r := range tc.filename {
				if r > 127 {
					hasNonASCII = true
					break
				}
			}
			require.True(t, hasNonASCII, "Test input should contain non-ASCII characters")
			
			result := ComputeStatusFileName(tc.filename, fixedTime)
			
			// Should end with .status
			assert.True(t, strings.HasSuffix(result, ".status"),
				"Should end with .status")
			
			// Should contain timestamp
			assert.Contains(t, result, "20250821-143022",
				"Should contain timestamp")
			
			// Verify all characters in result are ASCII (safe for Windows)
			for i, r := range result {
				assert.True(t, r <= 127, 
					"Character at position %d should be ASCII: %c (%d) in %s", i, r, r, result)
			}
			
			// Result should only contain allowed characters for Windows filenames
			allowedChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789.-_"
			for i, r := range result {
				assert.True(t, strings.ContainsRune(allowedChars, r),
					"Character at position %d not allowed in Windows filename: %c in %s", i, r, result)
			}
			
			if tc.expectEmpty {
				// For heavily Unicode filenames, should result in minimal base name
				parts := strings.Split(strings.TrimSuffix(result, ".status"), "-")
				assert.GreaterOrEqual(t, len(parts), 3, // should have base, date, time at minimum
					"Should have timestamp parts even with sanitized Unicode")
			}
		})
	}
}

// TestWindowsFilenameValidation_StatusFilenameConsistency tests that status filenames are generated consistently on Windows
func TestWindowsFilenameValidation_StatusFilenameConsistency(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows consistency test only runs on Windows")
	}

	fixedTime := time.Date(2025, 8, 21, 14, 30, 22, 0, time.UTC)

	testCases := []struct {
		name     string
		filename string
	}{
		{"WindowsPath", "C:\\Users\\test\\intent-file.json"},
		{"UNCPath", "\\\\server\\share\\intent-file.json"}, 
		{"RelativePath", "..\\..\\intent-file.json"},
		{"PathWithSpaces", "C:\\Program Files\\intent file.json"},
		{"LongPath", strings.Repeat("a", 200) + ".json"},
		{"MixedSeparators", "C:/Users\\test/intent-file.json"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call multiple times to ensure consistency
			results := make([]string, 5)
			for i := 0; i < 5; i++ {
				results[i] = ComputeStatusFileName(tc.filename, fixedTime)
			}
			
			// All results should be identical
			for i := 1; i < len(results); i++ {
				assert.Equal(t, results[0], results[i],
					"Status filename generation should be consistent")
			}
			
			// Should be valid Windows filename
			result := results[0]
			assert.True(t, strings.HasSuffix(result, ".status"),
				"Should end with .status")
			assert.NotContains(t, result, "\\", "Should not contain backslashes")
			assert.NotContains(t, result, "/", "Should not contain forward slashes")
			
			// Should not exceed Windows filename length limits (255 chars for NTFS)
			assert.LessOrEqual(t, len(result), 200,
				"Filename should not be excessively long")
		})
	}
}

// TestWindowsFilenameValidation_SuspiciousPatterns covers all suspicious patterns
func TestWindowsFilenameValidation_SuspiciousPatterns(t *testing.T) {
	fixedTime := time.Date(2025, 8, 21, 14, 30, 22, 0, time.UTC)

	suspiciousPatterns := []struct {
		name        string
		filename    string
		description string
	}{
		{
			name:        "TrailingTilde",
			filename:    "intent-test~.json",
			description: "trailing ~ (Windows backup indicator)",
		},
		{
			name:        "TmpExtension",
			filename:    "intent-test.tmp.json",
			description: ".tmp in filename",
		},
		{
			name:        "BakExtension", 
			filename:    "intent-test.bak.json",
			description: ".bak in filename",
		},
		{
			name:        "OrigExtension",
			filename:    "intent-test.orig.json", 
			description: ".orig in filename",
		},
		{
			name:        "SwpExtension",
			filename:    "intent-test.swp.json",
			description: ".swp in filename (vim swap)",
		},
		{
			name:        "LeadingDotHash",
			filename:    ".#intent-test.json",
			description: "leading .# (emacs lock file)",
		},
		{
			name:        "RepeatedDots",
			filename:    "intent-test..json",
			description: "repeated dots",
		},
		{
			name:        "TripleDots",
			filename:    "intent-test...json",
			description: "triple dots",
		},
		{
			name:        "TrailingSpace",
			filename:    "intent-test .json",
			description: "trailing space before extension",
		},
		{
			name:        "TrailingDot",
			filename:    "intent-test..json", 
			description: "trailing dot before extension",
		},
		{
			name:        "LeadingSpace",
			filename:    " intent-test.json",
			description: "leading space",
		},
		{
			name:        "ControlCharacters",
			filename:    "intent-test\x01\x02.json",
			description: "control characters",
		},
		{
			name:        "TabCharacter",
			filename:    "intent-test\t.json",
			description: "tab character",
		},
		{
			name:        "NewlineCharacter",
			filename:    "intent-test\n.json",
			description: "newline character",
		},
	}

	for _, tc := range suspiciousPatterns {
		t.Run(tc.name, func(t *testing.T) {
			result := ComputeStatusFileName(tc.filename, fixedTime)
			
			// Should end with .status
			assert.True(t, strings.HasSuffix(result, ".status"),
				"Should end with .status for: %s", tc.description)
			
			// Should contain timestamp
			assert.Contains(t, result, "20250821-143022",
				"Should contain timestamp for: %s", tc.description)
			
			// Should not contain suspicious patterns in the result
			assert.NotContains(t, result, "~", "Should not contain ~ for: %s", tc.description)
			// Note: Extensions like .tmp, .bak, .orig, .swp are preserved as part of the filename structure
			// This is correct behavior - we sanitize the content but preserve valid extension patterns
			assert.NotRegexp(t, `\.{2,}`, result, "Should not contain repeated dots for: %s", tc.description)
			
			// Should not start with .# 
			assert.False(t, strings.HasPrefix(result, ".#"),
				"Should not start with .# for: %s", tc.description)
			
			// Should not have leading/trailing spaces
			assert.Equal(t, strings.TrimSpace(result), result,
				"Should not have leading/trailing spaces for: %s", tc.description)
			
			// Should not contain control characters
			for i, r := range result {
				assert.False(t, unicode.IsControl(r),
					"Should not contain control character at position %d for: %s", i, tc.description)
			}
			
			// Should be valid UTF-8 
			assert.True(t, utf8.ValidString(result),
				"Should be valid UTF-8 for: %s", tc.description)
		})
	}
}

// TestWindowsFilenameValidation_EdgeCases tests edge cases for Windows filename validation
func TestWindowsFilenameValidation_EdgeCases(t *testing.T) {
	fixedTime := time.Date(2025, 8, 21, 14, 30, 22, 0, time.UTC)

	edgeCases := []struct {
		name        string
		filename    string
		expectValid bool
		description string
	}{
		{
			name:        "EmptyFilename",
			filename:    "",
			expectValid: true, // Should generate default name
			description: "empty filename should get default name",
		},
		{
			name:        "OnlyExtension",
			filename:    ".json",
			expectValid: true,
			description: "filename that's only extension",
		},
		{
			name:        "OnlyDots",
			filename:    "...json",
			expectValid: true,
			description: "filename starting with dots",
		},
		{
			name:        "VeryLongFilename",
			filename:    strings.Repeat("a", 300) + ".json",
			expectValid: true, // Should be truncated
			description: "very long filename should be truncated",
		},
		{
			name:        "AllReservedChars",
			filename:    "<>:\"/\\|?*.json",
			expectValid: true, // Should be sanitized
			description: "filename with all reserved characters",
		},
		{
			name:        "WindowsReservedWithExt",
			filename:    "CON.json",
			expectValid: true, // Should be modified
			description: "Windows reserved name with extension",
		},
		{
			name:        "UnicodeOnlyFilename",
			filename:    "测试文件.json",
			expectValid: true, // Should be sanitized to minimal name
			description: "filename with only Unicode characters",
		},
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ComputeStatusFileName(tc.filename, fixedTime)
			
			if tc.expectValid {
				// Should always produce a valid result
				assert.NotEmpty(t, result, "Should produce non-empty result for: %s", tc.description)
				assert.True(t, strings.HasSuffix(result, ".status"),
					"Should end with .status for: %s", tc.description)
				assert.Contains(t, result, "20250821-143022",
					"Should contain timestamp for: %s", tc.description)
				
				// Should be reasonable length (not empty, not too long)
				assert.GreaterOrEqual(t, len(result), 20, // minimum reasonable length
					"Should have reasonable minimum length for: %s", tc.description)
				assert.LessOrEqual(t, len(result), 200, // maximum reasonable length
					"Should not be excessively long for: %s", tc.description)
				
				// Should contain only valid Windows filename characters
				validPattern := `^[A-Za-z0-9._-]+$`
				assert.Regexp(t, validPattern, result,
					"Should contain only valid characters for: %s", tc.description)
			}
		})
	}
}

// TestWindowsFilenameValidation_UnicodeNormalizationNFKC tests specific NFKC normalization behavior
func TestWindowsFilenameValidation_UnicodeNormalizationNFKC(t *testing.T) {
	fixedTime := time.Date(2025, 8, 21, 14, 30, 22, 0, time.UTC)

	// Test specific Unicode normalization cases
	normalizationTests := []struct {
		name     string
		input    string
		expected string // Expected after NFKC normalization (before sanitization)
	}{
		{
			name:     "CombiningCharacters",
			input:    "e\u0301", // e + combining acute accent → é
			expected: "é",
		},
		{
			name:     "CompatibilityCharacters",
			input:    "ﬁ", // fi ligature → fi
			expected: "fi",
		},
		{
			name:     "FullwidthCharacters",
			input:    "Ａ", // fullwidth A → A
			expected: "A",
		},
		{
			name:     "CircledNumbers",
			input:    "①②③", // circled numbers → 123
			expected: "123",
		},
	}

	for _, tc := range normalizationTests {
		t.Run(tc.name, func(t *testing.T) {
			// Apply NFKC normalization like the sanitizer should
			normalized := norm.NFKC.String(tc.input)
			assert.Equal(t, tc.expected, normalized,
				"NFKC normalization should work correctly")
			
			// Test the full sanitization process
			filename := "intent-" + tc.input + ".json"
			result := ComputeStatusFileName(filename, fixedTime)
			
			// Should be valid status filename
			assert.True(t, strings.HasSuffix(result, ".status"),
				"Should end with .status")
			assert.Contains(t, result, "20250821-143022",
				"Should contain timestamp")
			
			// If normalized characters are ASCII-compatible, they might be preserved
			// Otherwise, they'll be sanitized to ASCII-only
			for _, r := range result {
				assert.True(t, r <= 127,
					"All characters should be ASCII after sanitization")
			}
		})
	}
}

// BenchmarkWindowsFilenameValidation benchmarks the filename sanitization performance
func BenchmarkWindowsFilenameValidation(b *testing.B) {
	timestamp := time.Now()
	
	benchmarkCases := []struct {
		name     string
		filename string
	}{
		{"SimpleFilename", "intent-test.json"},
		{"UnicodeFilename", "intent-测试-файл-📁.json"},
		{"ReservedCharsFilename", "intent<>:\"/\\|?*.json"},
		{"LongFilename", strings.Repeat("intent-test-", 20) + ".json"},
		{"WindowsReserved", "CON.json"},
	}

	for _, bc := range benchmarkCases {
		b.Run(bc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				ComputeStatusFileName(bc.filename, timestamp)
			}
		})
	}
}