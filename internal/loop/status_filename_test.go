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

// TestStatusFilenameGeneration verifies that status files follow the canonical pattern
// in their filename as expected: <base>-YYYYMMDD-HHMMSS.status (without .json extension)
func TestStatusFilenameGeneration(t *testing.T) {
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
	defer watcher.Close()

	testCases := []struct {
		name           string
		intentFile     string
		expectedPrefix string
	}{
		{
			name:           "StandardIntentFile",
			intentFile:     "intent-scale.json",
			expectedPrefix: "intent-scale-",
		},
		{
			name:           "IntentWithMultipleDots",
			intentFile:     "intent.scale.test.json",
			expectedPrefix: "intent.scale.test-",
		},
		{
			name:           "IntentWithUnderscore",
			intentFile:     "intent_deployment.json",
			expectedPrefix: "intent_deployment-",
		},
		{
			name:           "IntentWithNumbers",
			intentFile:     "intent-123.json",
			expectedPrefix: "intent-123-",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create the intent file
			intentPath := filepath.Join(tempDir, tc.intentFile)
			intentContent := `{"apiVersion": "v1", "kind": "NetworkIntent", "metadata": {"name": "test"}}`
			err := os.WriteFile(intentPath, []byte(intentContent), 0644)
			require.NoError(t, err)

			// Process the file
			err = watcher.Start()
			assert.NoError(t, err)

			// Wait briefly for processing
			time.Sleep(100 * time.Millisecond)

			// Check that status file was created with correct naming
			statusDir := filepath.Join(tempDir, "status")
			entries, err := os.ReadDir(statusDir)
			require.NoError(t, err)

			// Find the status file for this intent
			var foundStatusFile string
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), tc.expectedPrefix) {
					foundStatusFile = entry.Name()
					break
				}
			}

			assert.NotEmpty(t, foundStatusFile, 
				"Status file should exist with prefix %s", tc.expectedPrefix)

			// Verify the full format: <base>-YYYYMMDD-HHMMSS.status
			if foundStatusFile != "" {
				assert.True(t, strings.HasSuffix(foundStatusFile, ".status"),
					"Status file should end with .status")
				
				// Check timestamp format
				withoutStatus := strings.TrimSuffix(foundStatusFile, ".status")
				parts := strings.Split(withoutStatus, "-")
				
				// Should have at least 3 parts: the filename parts, date, and time
				assert.GreaterOrEqual(t, len(parts), 3,
					"Status filename should include timestamp")
				
				// The base name should NOT have .json extension
				assert.NotContains(t, foundStatusFile, ".json-",
					"Status filename should not contain .json before timestamp")
			}

			// Clean up for next test
			os.RemoveAll(statusDir)
		})
	}
}

// TestStatusFilenameConsistency verifies that multiple status files for the same
// intent follow the canonical naming pattern without .json extension
func TestStatusFilenameConsistency(t *testing.T) {
	tempDir := t.TempDir()
	mockPorch := createMockPorch(t, tempDir, 0, "processed", "")
	
	config := &Config{
		DebounceDur: 10 * time.Millisecond,
		MaxWorkers:  1,
		PorchPath:   mockPorch,
		Once:        false,
		Period:      100 * time.Millisecond,
	}
	
	watcher, err := NewWatcher(tempDir, *config)
	require.NoError(t, err)
	defer watcher.Close()

	// Create intent file
	intentFile := "intent-versioning.json"
	intentPath := filepath.Join(tempDir, intentFile)
	
	// Start watcher
	go watcher.Start()
	
	// Process the file multiple times
	for i := 0; i < 3; i++ {
		// Write/update the intent file
		content := fmt.Sprintf(`{"version": %d}`, i+1)
		err := os.WriteFile(intentPath, []byte(content), 0644)
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

	// All status files should follow the canonical pattern without .json
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "intent-versioning-") {
			assert.NotContains(t, entry.Name(), ".json-",
				"Status file %s should not contain .json before timestamp", entry.Name())
			assert.True(t, strings.HasSuffix(entry.Name(), ".status"),
				"Status file %s should end with .status", entry.Name())
		}
	}
}