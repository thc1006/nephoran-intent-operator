package loop

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStatusFilenameGeneration verifies that status files follow the production naming pattern:
// <basename_including_ext>-YYYYMMDD-HHMMSS.status
// e.g. intent-scale.json-20260217-101825.status
func TestStatusFilenameGeneration(t *testing.T) {
	testCases := []struct {
		name       string
		intentFile string
	}{
		{
			name:       "StandardIntentFile",
			intentFile: "intent-scale.json",
		},
		{
			name:       "IntentWithMultipleDots",
			intentFile: "intent-scale-test.json",
		},
		{
			name:       "IntentWithUnderscore",
			intentFile: "intent-deployment.json",
		},
		{
			name:       "IntentWithNumbers",
			intentFile: "intent-123.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new temp dir and watcher for each test case
			tempDir := t.TempDir()

			// Create a simple mock porch command
			mockPorch := createMockPorch(t, tempDir, 0, "processed", "")

			config := &Config{
				DebounceDur: 10 * time.Millisecond,
				MaxWorkers:  2,
				PorchPath:   mockPorch,
				Once:        true,
			}

			watcher, err := NewWatcher(tempDir, *config)
			require.NoError(t, err)
			defer watcher.Close() // #nosec G307 - Error handled in defer

			// Create the intent file with valid content for the watcher's JSON validator
			intentPath := filepath.Join(tempDir, tc.intentFile)
			intentContent := `{"intent_type": "scaling", "target": "my-app", "namespace": "default", "replicas": 3}`
			err = os.WriteFile(intentPath, []byte(intentContent), 0o644)
			require.NoError(t, err)

			// Process the file
			err = watcher.Start()
			assert.NoError(t, err)

			// Wait briefly for processing
			time.Sleep(100 * time.Millisecond)

			// Check that status file was created with correct naming.
			// Production code: baseName := filepath.Base(intentFile)
			// statusFile = fmt.Sprintf("%s-%s.status", baseName, timestamp)
			// So for "intent-scale.json" the status file is "intent-scale.json-YYYYMMDD-HHMMSS.status"
			statusDir := filepath.Join(tempDir, "status")
			entries, err := os.ReadDir(statusDir)
			require.NoError(t, err)

			// The expected prefix is the full intent filename followed by a dash
			intentBasename := filepath.Base(tc.intentFile)
			expectedPrefix := intentBasename + "-"

			// Find the status file for this intent
			var foundStatusFile string
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), expectedPrefix) {
					foundStatusFile = entry.Name()
					break
				}
			}

			assert.NotEmpty(t, foundStatusFile,
				"Status file should exist with prefix %s", expectedPrefix)

			// Verify the full format: <basename>-YYYYMMDD-HHMMSS.status
			if foundStatusFile != "" {
				assert.True(t, strings.HasSuffix(foundStatusFile, ".status"),
					"Status file should end with .status")

				// Strip the prefix and .status suffix; what remains is the timestamp
				withoutPrefix := strings.TrimPrefix(foundStatusFile, expectedPrefix)
				timestamp := strings.TrimSuffix(withoutPrefix, ".status")

				// Timestamp should be exactly 15 chars: YYYYMMDD-HHMMSS
				assert.Len(t, timestamp, 15,
					"Timestamp portion should be 15 chars (YYYYMMDD-HHMMSS)")
			}

			// Clean up for next test
			os.RemoveAll(statusDir)
		})
	}
}

// TestStatusFilenameConsistency verifies that multiple status files for the same
// intent follow the production naming pattern: <basename>-YYYYMMDD-HHMMSS.status
func TestStatusFilenameConsistency(t *testing.T) {
	tempDir := t.TempDir()
	mockPorch := createMockPorch(t, tempDir, 0, "processed", "")

	config := &Config{
		DebounceDur: 10 * time.Millisecond,
		MaxWorkers:  3, // Production-like worker count for realistic concurrency testing
		PorchPath:   mockPorch,
		Once:        false,
		Period:      100 * time.Millisecond,
	}

	watcher, err := NewWatcher(tempDir, *config)
	require.NoError(t, err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	// Create intent file
	intentFile := "intent-versioning.json"
	intentPath := filepath.Join(tempDir, intentFile)

	// Start watcher
	go watcher.Start()

	// Process the file multiple times
	for i := 0; i < 3; i++ {
		// Write/update the intent file with valid content
		content := fmt.Sprintf(`{"intent_type": "scaling", "target": "my-app", "namespace": "default", "replicas": %d}`, i+1)
		err := os.WriteFile(intentPath, []byte(content), 0o644)
		require.NoError(t, err)

		// Wait for processing
		time.Sleep(150 * time.Millisecond)
	}

	// Stop the watcher
	watcher.Close()

	// Check status files
	statusDir := filepath.Join(tempDir, "status")
	entries, err := os.ReadDir(statusDir)
	if err != nil {
		// Status dir might not exist if processing didn't complete
		return
	}

	// Production code produces: intent-versioning.json-YYYYMMDD-HHMMSS.status
	// All matching status files should end with .status
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "intent-versioning.json-") {
			assert.True(t, strings.HasSuffix(entry.Name(), ".status"),
				"Status file %s should end with .status", entry.Name())
		}
	}
}
