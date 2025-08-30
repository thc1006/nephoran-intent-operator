// Package generator provides utilities for sanitizing and validating patch generation inputs.

package generator

import (
	"path/filepath"
	"regexp"
	"strings"
)

// SanitizePath sanitizes and prevents directory traversal in path generation.
// It removes absolute path references, directory traversal attempts, and cleans the input path.
// This helps prevent potential security vulnerabilities when processing file paths.
//
// Parameters:
//   - path: The input file path to sanitize
//
// Returns a cleaned, safe path string with all traversal attempts removed.
func SanitizePath(path string) string {

	// Remove directory traversal attempts.

	path = filepath.Clean(path)

	// Prevent absolute paths.

	if filepath.IsAbs(path) {

		path = filepath.Base(path)

	}

	// Remove any remaining ".." or leading/trailing slashes.

	path = strings.TrimPrefix(path, "../")

	path = strings.TrimPrefix(path, "/")

	path = strings.ReplaceAll(path, "../", "")

	return path

}

// SanitizeCommand sanitizes a command string to prevent command injection attacks.
// It removes dangerous shell characters and special symbols that could be used for malicious purposes.
//
// Parameters:
//   - cmd: The input command string to sanitize
//
// Returns a sanitized command string with potentially harmful characters removed.
func SanitizeCommand(cmd string) string {

	// Regex for dangerous shell characters including backticks.

	// Matches: ; & | < > ( ) $ `.

	dangerousChars := regexp.MustCompile("[;&|<>()$`]")

	// Remove dangerous characters.

	sanitizedCmd := dangerousChars.ReplaceAllString(cmd, "")

	// Additional layer of sanitization for critical characters.

	sanitizedCmd = strings.ReplaceAll(sanitizedCmd, ";", "")

	sanitizedCmd = strings.ReplaceAll(sanitizedCmd, "|", "")

	sanitizedCmd = strings.ReplaceAll(sanitizedCmd, "&", "")

	return sanitizedCmd

}

// ValidateBinaryContent performs security validations on binary content.
// It checks:
//  1. Content size (max 5 MB)
//  2. Presence of non-printable characters
//  3. Basic YAML structure validation
//
// Parameters:
//   - content: The byte slice containing the binary data to validate
//
// Returns true if the content passes all security checks, false otherwise.
func ValidateBinaryContent(content []byte) bool {

	// Check content size (max 5 MB).

	if len(content) > 5*1024*1024 {

		return false

	}

	// Check for binary/non-printable characters.

	for _, b := range content {

		if b < 32 && b != 9 && b != 10 && b != 13 {

			return false

		}

	}

	// Basic YAML validation - checks for simple key:value multi-line format.

	// This regex validates basic YAML structure but does NOT guarantee full YAML compliance.

	// Pattern matches: key: value (with optional whitespace).

	// Examples:.

	//   key: value.

	//   another_key: 123.

	//   NAME-1: something.

	contentStr := string(content)

	yamlValidationRegex := regexp.MustCompile(`^(\s*[a-zA-Z0-9_-]+\s*:\s*[^\n]+\n)*$`)

	return yamlValidationRegex.MatchString(contentStr)

}
