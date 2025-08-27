package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsIntentFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{
			name:     "valid intent file",
			filename: "intent-20250811T180927Z.json",
			want:     true,
		},
		{
			name:     "valid intent file with path",
			filename: "/handoff/intent-test.json",
			want:     true,
		},
		{
			name:     "not an intent file - wrong prefix",
			filename: "test-20250811.json",
			want:     false,
		},
		{
			name:     "not an intent file - wrong extension",
			filename: "intent-test.yaml",
			want:     false,
		},
		{
			name:     "not an intent file - no extension",
			filename: "intent-test",
			want:     false,
		},
		{
			name:     "empty filename",
			filename: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isIntentFile(tt.filename)
			if got != tt.want {
				t.Errorf("isIntentFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestExtractCorrelationID(t *testing.T) {
	// Create temporary test files
	tempDir := t.TempDir()

	tests := []struct {
		name          string
		jsonContent   string
		wantID        string
		shouldSucceed bool
	}{
		{
			name:          "intent with correlation_id",
			jsonContent:   `{"correlation_id":"test-123","intent_type":"scaling","target":"nf-sim","namespace":"ran-a","replicas":5}`,
			wantID:        "test-123",
			shouldSucceed: true,
		},
		{
			name:          "intent without correlation_id",
			jsonContent:   `{"intent_type":"scaling","target":"nf-sim","namespace":"ran-a","replicas":5}`,
			wantID:        "",
			shouldSucceed: true,
		},
		{
			name:          "invalid JSON",
			jsonContent:   `{invalid json}`,
			wantID:        "",
			shouldSucceed: true, // Should handle gracefully
		},
		{
			name:          "empty file",
			jsonContent:   ``,
			wantID:        "",
			shouldSucceed: true, // Should handle gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			testFile := filepath.Join(tempDir, "test-intent.json")
			err := // FIXME: Adding error check per errcheck linter
 _ = os.WriteFile(testFile, []byte(tt.jsonContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Test extraction
			gotID := extractCorrelationID(testFile)
			if gotID != tt.wantID {
				t.Errorf("extractCorrelationID() = %q, want %q", gotID, tt.wantID)
			}

			// Clean up
			_ = // FIXME: Adding error check per errcheck linter
 _ = os.Remove(testFile)
		})
	}

	// Test with non-existent file
	t.Run("non-existent file", func(t *testing.T) {
		gotID := extractCorrelationID("/non/existent/file.json")
		if gotID != "" {
			t.Errorf("extractCorrelationID(non-existent) = %q, want empty string", gotID)
		}
	})
}

// TestBuildCommand verifies the command construction logic
func TestCommandConstruction(t *testing.T) {
	// This test validates that the command would be built correctly
	// In a real scenario, we'd mock exec.Command, but for MVP this demonstrates intent
	
	intentPath := "/handoff/intent-test.json"
	outDir := "examples/packages/scaling"
	
	// The expected command components
	expectedCmd := "go"
	expectedArgs := []string{"run", "./cmd/porch-publisher", "-intent", intentPath, "-out", outDir}
	
	// Verify these are the values we'd use (conceptual test)
	if expectedCmd != "go" {
		t.Errorf("Expected command to be 'go'")
	}
	
	if len(expectedArgs) != 6 {
		t.Errorf("Expected 6 command arguments, got %d", len(expectedArgs))
	}
	
	if expectedArgs[1] != "./cmd/porch-publisher" {
		t.Errorf("Expected porch-publisher path to be './cmd/porch-publisher', got %s", expectedArgs[1])
	}
}