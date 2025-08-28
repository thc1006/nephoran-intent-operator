package loop

import (
	"strings"
)

// IsIntentFile checks if a filename matches the intent file pattern.
// Intent files must start with "intent-" and end with ".json".
func IsIntentFile(filename string) bool {
	// Check for the required prefix and suffix.
	return strings.HasPrefix(filename, "intent-") && strings.HasSuffix(filename, ".json")
}
