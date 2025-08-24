//go:build windows
// +build windows

package pathutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWindowsPathLimitsComprehensive provides comprehensive coverage for Windows path
// validation, limits, and edge cases that were causing CI failures.
func TestWindowsPathLimitsComprehensive(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows path tests only apply to Windows")
	}

	t.Run("Windows_Path_Length_Limits", func(t *testing.T) {
		testCases := []struct {
			name        string
			pathLength  int
			shouldPass  bool
			description string
		}{
			{
				name:        "short_path",
				pathLength:  50,
				shouldPass:  true,
				description: "Short paths should always work",
			},
			{
				name:        "near_limit_path",
				pathLength:  240,
				shouldPass:  true,
				description: "Paths near the 248 limit should work",
			},
			{
				name:        "at_limit_path",
				pathLength:  WindowsMaxPath,
				shouldPass:  true,
				description: "Paths at the Windows limit should work",
			},
			{
				name:        "over_limit_path",
				pathLength:  260,
				shouldPass:  false,
				description: "Paths over the limit should fail without \\\\?\\ prefix",
			},
			{
				name:        "very_long_path",
				pathLength:  400,
				shouldPass:  false,
				description: "Very long paths should fail without \\\\?\\ prefix",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Create a path of the specified length
				baseDir := "C:\\"
				remaining := tc.pathLength - len(baseDir) - 4 // Leave room for ".txt"
				
				if remaining > 0 {
					// Build path with segments to reach desired length
					var pathSegments []string
					for remaining > 0 {
						segmentLen := 20 // Use 20-char segments
						if remaining < segmentLen {
							segmentLen = remaining
						}
						segment := strings.Repeat("a", segmentLen)
						pathSegments = append(pathSegments, segment)
						remaining -= segmentLen + 1 // +1 for separator
					}
					
					testPath := filepath.Join(baseDir, filepath.Join(pathSegments...), "test.txt")
					
					// Adjust length to be more precise
					if len(testPath) < tc.pathLength-5 {
						// Add padding to the last segment
						padding := tc.pathLength - len(testPath) - 5
						if padding > 0 {
							pathSegments[len(pathSegments)-1] += strings.Repeat("x", padding)
							testPath = filepath.Join(baseDir, filepath.Join(pathSegments...), "test.txt")
						}
					}

					t.Logf("Testing path length %d (actual: %d): %s", tc.pathLength, len(testPath), testPath)

					err := ValidateWindowsPath(testPath)
					
					if tc.shouldPass {
						assert.NoError(t, err, "Path should be valid: %s", tc.description)
					} else {
						assert.Error(t, err, "Path should be invalid: %s", tc.description)
						if err != nil {
							assert.Contains(t, err.Error(), "path too long", "Error should mention path length")
						}
					}
				}
			})
		}
	})

	t.Run("Windows_Reserved_Names", func(t *testing.T) {
		reservedNames := []string{
			"CON", "PRN", "AUX", "NUL",
			"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
			"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
		}

		for _, reservedName := range reservedNames {
			t.Run("reserved_"+strings.ToLower(reservedName), func(t *testing.T) {
				testCases := []struct {
					path        string
					shouldFail  bool
					description string
				}{
					{
						path:        "C:\\" + reservedName,
						shouldFail:  true,
						description: "Bare reserved name should fail",
					},
					{
						path:        "C:\\" + reservedName + ".txt",
						shouldFail:  true,
						description: "Reserved name with extension should fail",
					},
					{
						path:        "C:\\dir\\" + reservedName,
						shouldFail:  true,
						description: "Reserved name in subdirectory should fail",
					},
					{
						path:        "C:\\dir\\" + reservedName + ".json",
						shouldFail:  true,
						description: "Reserved name with extension in subdirectory should fail",
					},
					{
						path:        "C:\\" + strings.ToLower(reservedName),
						shouldFail:  true,
						description: "Lowercase reserved name should fail",
					},
					{
						path:        "C:\\valid" + reservedName + "name.txt",
						shouldFail:  false,
						description: "Reserved name as substring should be OK",
					},
				}

				for _, tc := range testCases {
					t.Run(tc.description, func(t *testing.T) {
						err := ValidateWindowsPath(tc.path)
						
						if tc.shouldFail {
							assert.Error(t, err, "Should fail for %s: %s", reservedName, tc.description)
							if err != nil {
								assert.Contains(t, strings.ToLower(err.Error()), "reserved", "Error should mention reserved filename")
							}
						} else {
							assert.NoError(t, err, "Should pass for %s: %s", reservedName, tc.description)
						}
					})
				}
			})
		}
	})

	t.Run("Windows_Invalid_Characters", func(t *testing.T) {
		invalidChars := []struct {
			char        string
			description string
			allowedIn   []string // Contexts where this char might be allowed
		}{
			{char: "<", description: "less than"},
			{char: ">", description: "greater than"},
			{char: "\"", description: "quote"},
			{char: "|", description: "pipe"},
			{char: "?", description: "question mark"},
			{char: "*", description: "asterisk"},
			{char: ":", description: "colon", allowedIn: []string{"drive"}},
		}

		for _, invalid := range invalidChars {
			t.Run("invalid_char_"+invalid.description, func(t *testing.T) {
				testPaths := []struct {
					path        string
					shouldFail  bool
					description string
				}{
					{
						path:        "C:\\dir\\file" + invalid.char + "name.txt",
						shouldFail:  true,
						description: "Invalid char in filename",
					},
					{
						path:        "C:\\dir" + invalid.char + "name\\file.txt",
						shouldFail:  true,
						description: "Invalid char in directory name",
					},
				}

				// Special handling for colon
				if invalid.char == ":" {
					testPaths = append(testPaths, []struct {
						path        string
						shouldFail  bool
						description string
					}{
						{
							path:        "C:\\valid\\path.txt",
							shouldFail:  false,
							description: "Valid drive letter colon should pass",
						},
						{
							path:        "C:\\dir\\file:name.txt",
							shouldFail:  true,
							description: "Colon not in drive position should fail",
						},
						{
							path:        "CC:\\invalid.txt",
							shouldFail:  true,
							description: "Multiple colons should fail",
						},
					}...)
				}

				for _, tc := range testPaths {
					t.Run(tc.description, func(t *testing.T) {
						err := ValidateWindowsPath(tc.path)
						
						if tc.shouldFail {
							assert.Error(t, err, "Should fail for invalid char %s: %s", invalid.char, tc.description)
						} else {
							assert.NoError(t, err, "Should pass for valid use of %s: %s", invalid.char, tc.description)
						}
					})
				}
			})
		}
	})

	t.Run("Windows_Path_Normalization", func(t *testing.T) {
		testCases := []struct {
			name        string
			input       string
			expected    string
			shouldError bool
			description string
		}{
			{
				name:        "mixed_separators",
				input:       "C:/Users/test\\Documents/file.txt",
				expected:    "C:\\Users\\test\\Documents\\file.txt",
				shouldError: false,
				description: "Mixed separators should be normalized to backslashes",
			},
			{
				name:        "drive_relative_path",
				input:       "C:temp\\file.txt",
				expected:    "", // Will be resolved based on current directory
				shouldError: false,
				description: "Drive-relative paths should be resolved",
			},
			{
				name:        "trailing_backslash",
				input:       "C:\\Users\\test\\",
				expected:    "", // Will be resolved to absolute path
				shouldError: false,
				description: "Trailing backslashes should be handled",
			},
			{
				name:        "path_traversal_attempt",
				input:       "C:\\Users\\..\\..\\Windows\\System32",
				expected:    "",
				shouldError: true,
				description: "Path traversal attempts should be rejected",
			},
			{
				name:        "empty_path",
				input:       "",
				expected:    "",
				shouldError: true,
				description: "Empty paths should be rejected",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				normalized, err := NormalizeUserPath(tc.input)
				
				if tc.shouldError {
					assert.Error(t, err, "Should error for %s: %s", tc.input, tc.description)
				} else {
					assert.NoError(t, err, "Should not error for %s: %s", tc.input, tc.description)
					
					if err == nil {
						t.Logf("Normalized %s to %s", tc.input, normalized)
						
						// Verify the normalized path is absolute
						assert.True(t, filepath.IsAbs(normalized) || IsAbsoluteWindowsPath(normalized),
							"Normalized path should be absolute")
						
						// Verify it doesn't contain traversal elements
						assert.NotContains(t, normalized, "..", "Normalized path should not contain ..")
						
						// Verify it's a valid Windows path
						err := ValidateWindowsPath(normalized)
						assert.NoError(t, err, "Normalized path should be valid")
					}
				}
			})
		}
	})

	t.Run("Windows_Long_Path_Support", func(t *testing.T) {
		// Test \\?\ prefix for long paths
		testCases := []struct {
			name         string
			basePath     string
			shouldPrefix bool
			description  string
		}{
			{
				name:         "short_path_no_prefix",
				basePath:     "C:\\Users\\test\\short.txt",
				shouldPrefix: false,
				description:  "Short paths should not get \\\\?\\ prefix",
			},
			{
				name:         "long_path_gets_prefix",
				basePath:     "C:\\" + strings.Repeat("verylongdirectoryname", 10) + "\\file.txt",
				shouldPrefix: true,
				description:  "Long paths should get \\\\?\\ prefix",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				if len(tc.basePath) < WindowsMaxPath && tc.shouldPrefix {
					// Make the path longer to trigger prefixing
					padding := WindowsMaxPath - len(tc.basePath) + 10
					dir := filepath.Dir(tc.basePath)
					base := filepath.Base(tc.basePath)
					paddedDir := dir + strings.Repeat("x", padding)
					tc.basePath = filepath.Join(paddedDir, base)
				}

				normalized, err := NormalizeUserPath(tc.basePath)
				assert.NoError(t, err, "Path normalization should succeed")

				if tc.shouldPrefix && len(tc.basePath) >= WindowsMaxPath {
					assert.True(t, strings.HasPrefix(normalized, `\\?\`),
						"Long paths should have \\\\?\\ prefix: %s", normalized)
				} else {
					assert.False(t, strings.HasPrefix(normalized, `\\?\`),
						"Short paths should not have \\\\?\\ prefix: %s", normalized)
				}
			})
		}
	})

	t.Run("Windows_UNC_Path_Handling", func(t *testing.T) {
		testCases := []struct {
			name        string
			path        string
			isUNC       bool
			isValid     bool
			description string
		}{
			{
				name:        "valid_unc_path",
				path:        "\\\\server\\share\\file.txt",
				isUNC:       true,
				isValid:     true,
				description: "Valid UNC path should be recognized",
			},
			{
				name:        "unc_path_with_subdirs",
				path:        "\\\\server\\share\\subdir\\file.txt",
				isUNC:       true,
				isValid:     true,
				description: "UNC path with subdirectories should be valid",
			},
			{
				name:        "device_path_not_unc",
				path:        "\\\\?\\C:\\file.txt",
				isUNC:       false,
				isValid:     true,
				description: "Device paths should not be considered UNC",
			},
			{
				name:        "invalid_unc_path",
				path:        "\\\\\\invalid",
				isUNC:       false,
				isValid:     false,
				description: "Invalid UNC format should be rejected",
			},
			{
				name:        "regular_path_not_unc",
				path:        "C:\\regular\\path.txt",
				isUNC:       false,
				isValid:     true,
				description: "Regular paths should not be considered UNC",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				isUNC := IsUNCPath(tc.path)
				assert.Equal(t, tc.isUNC, isUNC, "UNC detection should match expected for %s", tc.description)

				err := ValidateWindowsPath(tc.path)
				if tc.isValid {
					assert.NoError(t, err, "Path should be valid: %s", tc.description)
				} else {
					assert.Error(t, err, "Path should be invalid: %s", tc.description)
				}
			})
		}
	})

	t.Run("Windows_Device_Path_Handling", func(t *testing.T) {
		testCases := []struct {
			name        string
			path        string
			isDevice    bool
			isValid     bool
			description string
		}{
			{
				name:        "long_path_device",
				path:        "\\\\?\\C:\\very\\long\\path\\file.txt",
				isDevice:    true,
				isValid:     true,
				description: "Long path device format should be valid",
			},
			{
				name:        "dot_device_path",
				path:        "\\\\.\\COM1",
				isDevice:    true,
				isValid:     true,
				description: "Dot device path should be valid",
			},
			{
				name:        "regular_path_not_device",
				path:        "C:\\regular\\path.txt",
				isDevice:    false,
				isValid:     true,
				description: "Regular paths should not be considered device paths",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				isDevice := IsWindowsDevicePath(tc.path)
				assert.Equal(t, tc.isDevice, isDevice, "Device path detection should match expected for %s", tc.description)

				err := ValidateWindowsPath(tc.path)
				if tc.isValid {
					assert.NoError(t, err, "Path should be valid: %s", tc.description)
				} else {
					assert.Error(t, err, "Path should be invalid: %s", tc.description)
				}
			})
		}
	})

	t.Run("Parent_Directory_Creation_Edge_Cases", func(t *testing.T) {
		tempDir := t.TempDir()

		testCases := []struct {
			name        string
			path        string
			shouldWork  bool
			description string
		}{
			{
				name:        "simple_parent",
				path:        "simple\\parent\\file.txt",
				shouldWork:  true,
				description: "Simple parent directory creation",
			},
			{
				name:        "deep_nested_parent",
				path:        "very\\deeply\\nested\\directory\\structure\\file.txt",
				shouldWork:  true,
				description: "Deep nested directory creation",
			},
			{
				name:        "path_with_spaces",
				path:        "path with spaces\\and more spaces\\file.txt",
				shouldWork:  true,
				description: "Paths with spaces should work",
			},
			{
				name:        "mixed_separators",
				path:        "mixed/forward\\and/back\\separators\\file.txt",
				shouldWork:  true,
				description: "Mixed path separators should be handled",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				fullPath := filepath.Join(tempDir, tc.name, tc.path)
				
				err := EnsureParentDir(fullPath)
				
				if tc.shouldWork {
					assert.NoError(t, err, "Should succeed: %s", tc.description)
					
					if err == nil {
						// Verify parent directory was created
						parentDir := filepath.Dir(fullPath)
						_, statErr := os.Stat(parentDir)
						assert.NoError(t, statErr, "Parent directory should exist: %s", parentDir)
					}
				} else {
					assert.Error(t, err, "Should fail: %s", tc.description)
				}
			})
		}
	})

	t.Run("Concurrent_Path_Operations", func(t *testing.T) {
		// Test concurrent path validation and normalization
		numWorkers := 10
		var wg sync.WaitGroup
		errors := make(chan error, numWorkers)

		testPaths := []string{
			"C:\\Users\\test\\file1.txt",
			"C:\\Users\\test\\file2.txt",
			"C:\\Users\\test\\subdir\\file3.txt",
			"C:/Users/test/mixed/separators.txt",
			"C:\\Users\\test\\spaces in path\\file.txt",
		}

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				testPath := testPaths[id%len(testPaths)]
				
				// Test validation
				err := ValidateWindowsPath(testPath)
				if err != nil {
					errors <- err
					return
				}

				// Test normalization
				_, err = NormalizeUserPath(testPath)
				if err != nil {
					errors <- err
					return
				}

				// Test path utility functions
				_ = IsAbsoluteWindowsPath(testPath)
				_ = IsUNCPath(testPath)
				_ = IsWindowsDevicePath(testPath)
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for any errors
		var errorCount int
		for err := range errors {
			t.Errorf("Concurrent path operation error: %v", err)
			errorCount++
		}

		assert.Equal(t, 0, errorCount, "No concurrent path operation errors expected")
	})
}

// TestWindowsPathRegressionPrevention ensures specific path-related issues
// that caused Windows CI failures cannot reoccur.
func TestWindowsPathRegressionPrevention(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows path regression tests only apply to Windows")
	}

	t.Run("Status_File_Path_Length_Regression", func(t *testing.T) {
		// Test the scenario where status file paths might exceed Windows limits
		baseDir := t.TempDir()
		
		// Create a watch directory with a long name
		longDirName := strings.Repeat("verylongdirectoryname", 5) // 125 chars
		watchDir := filepath.Join(baseDir, longDirName)
		statusDir := filepath.Join(watchDir, "status")

		// Verify the paths are valid
		err := ValidateWindowsPath(watchDir)
		assert.NoError(t, err, "Watch directory path should be valid")

		err = ValidateWindowsPath(statusDir)
		assert.NoError(t, err, "Status directory path should be valid")

		// Test creating the directory structure
		err = EnsureParentDir(filepath.Join(statusDir, "test.txt"))
		assert.NoError(t, err, "Should be able to create long path directories")

		// Verify directories exist
		_, err = os.Stat(statusDir)
		assert.NoError(t, err, "Status directory should exist")
	})

	t.Run("Intent_File_Name_Validation", func(t *testing.T) {
		// Test various intent file names that might cause path issues
		testNames := []struct {
			name       string
			valid      bool
			description string
		}{
			{
				name:        "simple-intent.json",
				valid:       true,
				description: "Simple intent file name",
			},
			{
				name:        "intent with spaces.json",
				valid:       true,
				description: "Intent file with spaces",
			},
			{
				name:        strings.Repeat("long", 50) + ".json", // 200+ chars
				valid:       false,
				description: "Very long intent file name",
			},
			{
				name:        "intent<invalid>.json",
				valid:       false,
				description: "Intent file with invalid characters",
			},
			{
				name:        "CON.json",
				valid:       false,
				description: "Reserved name as intent file",
			},
		}

		baseDir := t.TempDir()
		watchDir := filepath.Join(baseDir, "watch")
		require.NoError(t, os.MkdirAll(watchDir, 0755))

		for _, tc := range testNames {
			t.Run("name_"+tc.name, func(t *testing.T) {
				intentPath := filepath.Join(watchDir, tc.name)
				
				err := ValidateWindowsPath(intentPath)
				
				if tc.valid {
					assert.NoError(t, err, "Should be valid: %s", tc.description)
				} else {
					assert.Error(t, err, "Should be invalid: %s", tc.description)
				}
			})
		}
	})

	t.Run("Path_Separator_Consistency", func(t *testing.T) {
		// Ensure path separator consistency doesn't break operations
		testPaths := []string{
			"C:\\Users\\test\\file.txt",      // Windows style
			"C:/Users/test/file.txt",         // Unix style
			"C:\\Users/test\\mixed/file.txt", // Mixed style
		}

		for _, testPath := range testPaths {
			t.Run("separators_"+strings.ReplaceAll(testPath, "\\", "_"), func(t *testing.T) {
				// Normalize the path
				normalized, err := NormalizeUserPath(testPath)
				assert.NoError(t, err, "Path normalization should succeed")
				
				// Verify it uses Windows separators
				assert.Contains(t, normalized, "\\", "Normalized path should use Windows separators")
				assert.NotContains(t, normalized, "/", "Normalized path should not contain forward slashes")
				
				// Verify it's valid
				err = ValidateWindowsPath(normalized)
				assert.NoError(t, err, "Normalized path should be valid")
			})
		}
	})

	t.Run("Long_Path_Normalization_Regression", func(t *testing.T) {
		// Test that long path normalization doesn't break
		baseDir := t.TempDir()
		
		// Create a path that's just under the limit
		longPath := baseDir
		for len(longPath) < WindowsMaxPath-50 {
			longPath = filepath.Join(longPath, "directory")
		}
		longPath = filepath.Join(longPath, "file.txt")

		t.Logf("Testing long path of length %d: %s", len(longPath), longPath)

		// Should be valid without prefix
		err := ValidateWindowsPath(longPath)
		assert.NoError(t, err, "Path just under limit should be valid")

		// Normalization should work
		normalized, err := NormalizeUserPath(longPath)
		assert.NoError(t, err, "Long path normalization should succeed")
		
		t.Logf("Normalized to length %d: %s", len(normalized), normalized)

		// Very long path should get prefix
		veryLongPath := longPath + strings.Repeat("x", 50)
		normalized, err = NormalizeUserPath(veryLongPath)
		if err == nil {
			// If normalization succeeded, it should have the prefix
			assert.True(t, strings.HasPrefix(normalized, `\\?\`) || len(normalized) < WindowsMaxPath,
				"Very long path should get \\\\?\\ prefix or be shortened")
		}
	})
}