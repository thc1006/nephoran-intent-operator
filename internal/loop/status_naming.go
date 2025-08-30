package loop

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// ComputeStatusFileName generates a canonical status filename for the given source path and timestamp.
//
// The canonical naming pattern is: <sanitizedBaseNameWithoutExt>-<YYYYMMDD>-<HHMMSS>.status
//
// Examples:
//   - "intent-test-0.json" + timestamp -> "intent-test-0-20250821-143022.status"
//   - "intent-test-10.json" + timestamp -> "intent-test-10-20250821-143022.status"
//   - "intent-test-100.json" + timestamp -> "intent-test-100-20250821-143022.status"
//   - "intent.scale.test.json" + timestamp -> "intent.scale.test-20250821-143022.status"
//   - "intent-no-ext" + timestamp -> "intent-no-ext-20250821-143022.status"
//
// Parameters:
//   - srcPath: The source file path (can be absolute or relative)
//   - timestamp: The timestamp to include in the filename
//
// Returns:
//   - The status filename (not the full path, just the filename)
func ComputeStatusFileName(srcPath string, timestamp time.Time) string {
	// Extract base name from source path
	baseName := filepath.Base(srcPath)

	// Remove file extension from base name
	baseNameWithoutExt := strings.TrimSuffix(baseName, filepath.Ext(baseName))

	// Sanitize the filename for cross-platform compatibility
	sanitizedName := sanitizeStatusFilename(baseNameWithoutExt)

	// Format timestamp as YYYYMMDD-HHMMSS
	timestampStr := timestamp.Format("20060102-150405")

	// Return the complete status filename
	return fmt.Sprintf("%s-%s.status", sanitizedName, timestampStr)
}
