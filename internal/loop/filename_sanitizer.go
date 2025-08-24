package loop

import (
	"regexp"
	"strings"
)

// Windows reserved names that cannot be used as filenames
var windowsReservedNames = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true,
	"COM5": true, "COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true,
	"LPT5": true, "LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// Regular expression for valid filename characters
var validFilenameChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// Regular expression for collapsing duplicate separators
var duplicateSeparators = regexp.MustCompile(`[-._]{2,}`)

// sanitizeStatusFilename sanitizes a filename for use as a status file on all platforms,
// with special handling for Windows constraints.
//
// Rules applied:
// - Allow only [A-Za-z0-9._-] characters; replace others with '-'
// - Collapse duplicate separators (-, ., _)
// - Handle Windows reserved names (CON, PRN, AUX, NUL, COM1-9, LPT1-9) by adding suffix
// - Remove trailing dots and spaces (Windows constraint)
// - Trim leading/trailing separators
// - Cap filename length at 128 characters (conservative for cross-platform compatibility)
func sanitizeStatusFilename(filename string) string {
	if filename == "" {
		return "unnamed"
	}

	// Step 1: Replace invalid characters with '-'
	sanitized := validFilenameChars.ReplaceAllString(filename, "-")

	// Step 2: Collapse duplicate separators
	sanitized = duplicateSeparators.ReplaceAllString(sanitized, "-")

	// Step 3: Trim leading and trailing separators
	sanitized = strings.Trim(sanitized, "-._")

	// Step 4: Handle Windows reserved names
	// Check the base name without extension
	parts := strings.Split(sanitized, ".")
	if len(parts) > 0 {
		baseName := strings.ToUpper(parts[0])
		if windowsReservedNames[baseName] {
			// Add a safe suffix to avoid reserved name
			parts[0] = parts[0] + "-file"
			sanitized = strings.Join(parts, ".")
		}
	}

	// Step 5: Remove trailing dots and spaces (Windows doesn't allow these)
	sanitized = strings.TrimRight(sanitized, ". ")

	// Step 6: Ensure filename is not empty after sanitization
	if sanitized == "" || sanitized == "." || sanitized == ".." {
		sanitized = "unnamed"
	}

	// Step 7: Cap length at 128 characters (leaving room for timestamp and .status extension)
	// We need to reserve about 20 chars for "-YYYYMMDD-HHMMSS.status"
	maxBaseLength := 100
	if len(sanitized) > maxBaseLength {
		// Try to preserve extension if present
		ext := ""
		if lastDot := strings.LastIndex(sanitized, "."); lastDot > 0 && lastDot < len(sanitized)-1 {
			ext = sanitized[lastDot:]
			if len(ext) <= 10 { // reasonable extension length
				sanitized = sanitized[:lastDot]
			}
		}
		
		// Truncate and add extension back
		if len(sanitized) > maxBaseLength-len(ext) {
			sanitized = sanitized[:maxBaseLength-len(ext)]
		}
		sanitized = sanitized + ext
		
		// Final trim of any trailing separators from truncation
		sanitized = strings.TrimRight(sanitized, "-._")
	}

	return sanitized
}