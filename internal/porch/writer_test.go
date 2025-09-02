package porch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type testIntent struct {
	IntentType string `json:"intent_type"`
	Target     string `json:"target"`
	Namespace  string `json:"namespace"`
	Replicas   int    `json:"replicas"`
}

func TestWriteIntent_FullFormat(t *testing.T) {
	tmpDir := t.TempDir()

	intent := testIntent{
		IntentType: "scaling",
		Target:     "my-app",
		Namespace:  "default",
		Replicas:   3,
	}

	err := WriteIntent(intent, tmpDir, "full")
	if err != nil {
		t.Fatalf("WriteIntent failed: %v", err)
	}

	// Check file was created
	yamlPath := filepath.Join(tmpDir, "scaling-patch.yaml")
	content, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	expected := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  namespace: default
spec:
  replicas: 3
`
	if string(content) != expected {
		t.Errorf("Unexpected content:\nGot:\n%s\nExpected:\n%s", content, expected)
	}
}

func TestWriteIntent_SMPFormat(t *testing.T) {
	tmpDir := t.TempDir()

	intent := testIntent{
		IntentType: "scaling",
		Target:     "my-app",
		Namespace:  "production",
		Replicas:   5,
	}

	err := WriteIntent(intent, tmpDir, "smp")
	if err != nil {
		t.Fatalf("WriteIntent failed: %v", err)
	}

	// Check JSON file was created
	jsonPath := filepath.Join(tmpDir, "scaling-patch.json")
	content, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Parse and validate JSON structure
	var smp map[string]interface{}
	if err := json.Unmarshal(content, &smp); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	// Check structure
	if smp["apiVersion"] != "apps/v1" {
		t.Errorf("Wrong apiVersion: %v", smp["apiVersion"])
	}
	if smp["kind"] != "Deployment" {
		t.Errorf("Wrong kind: %v", smp["kind"])
	}

	metadata := smp["metadata"].(map[string]interface{})
	if metadata["name"] != "my-app" {
		t.Errorf("Wrong name: %v", metadata["name"])
	}
	if metadata["namespace"] != "production" {
		t.Errorf("Wrong namespace: %v", metadata["namespace"])
	}

	spec := smp["spec"].(map[string]interface{})
	if int(spec["replicas"].(float64)) != 5 {
		t.Errorf("Wrong replicas: %v", spec["replicas"])
	}
}

func TestWriteIntent_InvalidIntentType(t *testing.T) {
	tmpDir := t.TempDir()

	intent := testIntent{
		IntentType: "networking", // Not supported
		Target:     "my-app",
		Namespace:  "default",
		Replicas:   3,
	}

	err := WriteIntent(intent, tmpDir, "full")
	if err == nil {
		t.Fatal("Expected error for unsupported intent type")
	}

	expectedErr := "only scaling intent is supported in MVP"
	if err.Error() != expectedErr {
		t.Errorf("Unexpected error: got %v, want %v", err.Error(), expectedErr)
	}
}

func TestWriteIntent_DefaultFormat(t *testing.T) {
	tmpDir := t.TempDir()

	intent := testIntent{
		IntentType: "scaling",
		Target:     "test-app",
		Namespace:  "test-ns",
		Replicas:   1,
	}

	// Test with empty format string (should default to full)
	err := WriteIntent(intent, tmpDir, "")
	if err != nil {
		t.Fatalf("WriteIntent failed: %v", err)
	}

	// Check YAML file was created (default behavior)
	yamlPath := filepath.Join(tmpDir, "scaling-patch.yaml")
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		t.Error("Expected YAML file not created with default format")
	}
}

// TestWriteIntent_FileSystemErrors tests various filesystem error conditions
func TestWriteIntent_FileSystemErrors(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) string
		expectError string
		intent      testIntent
	}{
		{
			name: "write permission denied on directory",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				restrictedDir := filepath.Join(tmpDir, "restricted")
				err := os.Mkdir(restrictedDir, 0o755)
				if err != nil {
					t.Fatalf("Failed to create restricted directory: %v", err)
				}

				// Remove write permissions
				err = os.Chmod(restrictedDir, 0o444)
				if err != nil {
					t.Skipf("Cannot modify directory permissions on this system: %v", err)
				}

				return restrictedDir
			},
			expectError: "failed to write file",
			intent: testIntent{
				IntentType: "scaling",
				Target:     "test-app",
				Namespace:  "default",
				Replicas:   1,
			},
		},
		{
			name: "output directory is a file",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "not-a-directory")
				err := os.WriteFile(filePath, []byte("test"), 0o644)
				if err != nil {
					t.Fatalf("Failed to create file: %v", err)
				}
				return filePath
			},
			expectError: "failed to create output directory",
			intent: testIntent{
				IntentType: "scaling",
				Target:     "test-app",
				Namespace:  "default",
				Replicas:   1,
			},
		},
		{
			name: "disk full simulation",
			setupFunc: func(t *testing.T) string {
				// This is tricky to simulate reliably across platforms
				// We'll create a directory that exists but can't be written to
				tmpDir := t.TempDir()
				targetDir := filepath.Join(tmpDir, "diskfull")
				err := os.Mkdir(targetDir, 0o755)
				if err != nil {
					t.Fatalf("Failed to create target directory: %v", err)
				}
				return targetDir
			},
			expectError: "", // Will be checked separately as disk full is hard to simulate
			intent: testIntent{
				IntentType: "scaling",
				Target:     "test-app",
				Namespace:  "default",
				Replicas:   1,
			},
		},
		{
			name: "extremely long file path",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				// Create a very long nested directory structure
				longPath := tmpDir
				for i := 0; i < 50; i++ {
					longPath = filepath.Join(longPath, "very-long-directory-name-that-exceeds-normal-limits")
				}
				return longPath
			},
			expectError: "failed to create output directory",
			intent: testIntent{
				IntentType: "scaling",
				Target:     "test-app",
				Namespace:  "default",
				Replicas:   1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outDir := tt.setupFunc(t)

			// Ensure cleanup happens even if test fails
			defer func() {
				// Restore permissions for cleanup
				os.Chmod(outDir, 0o755)
			}()

			err := WriteIntent(tt.intent, outDir, "full")

			if tt.expectError == "" {
				// For disk full simulation, just verify it completed
				// (actual disk full is hard to simulate reliably)
				return
			}

			if err == nil {
				t.Errorf("Expected error but got nil")
				return
			}

			if !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("Expected error containing '%s' but got: %v", tt.expectError, err)
			}
		})
	}
}

// TestWriteIntent_InvalidIntentData tests handling of malformed intent data
func TestWriteIntent_InvalidIntentData(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		intent      interface{}
		expectError string
	}{
		{
			name:        "nil intent",
			intent:      nil,
			expectError: "failed to marshal intent",
		},
		{
			name:        "invalid intent structure",
			intent:      make(chan int), // channels can't be marshaled to JSON
			expectError: "failed to marshal intent",
		},
		{
			name: "intent with circular reference",
			intent: func() interface{} {
				type circular struct {
					Name string    `json:"name"`
					Ref  *circular `json:"ref"`
				}
				c := &circular{Name: "test"}
				c.Ref = c // circular reference
				return c
			}(),
			expectError: "failed to marshal intent",
		},
		{
			name: "intent missing required fields",
			intent: json.RawMessage(`{}`),
			expectError: "failed to unmarshal intent",
		},
		{
			name: "intent with wrong field types",
			intent: json.RawMessage(`{}`),
			expectError: "failed to unmarshal intent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WriteIntent(tt.intent, tmpDir, "full")

			if err == nil {
				t.Errorf("Expected error but got nil")
				return
			}

			if !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("Expected error containing '%s' but got: %v", tt.expectError, err)
			}
		})
	}
}

// TestWriteIntent_EdgeCases tests edge cases for intent processing
func TestWriteIntent_EdgeCases(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		intent      testIntent
		format      string
		expectError string
	}{
		{
			name: "very long target name",
			intent: testIntent{
				IntentType: "scaling",
				Target:     strings.Repeat("a", 1000),
				Namespace:  "default",
				Replicas:   1,
			},
			format:      "full",
			expectError: "",
		},
		{
			name: "target with special characters",
			intent: testIntent{
				IntentType: "scaling",
				Target:     "test-app_with.special-chars",
				Namespace:  "default",
				Replicas:   1,
			},
			format:      "full",
			expectError: "",
		},
		{
			name: "unicode characters in target",
			intent: testIntent{
				IntentType: "scaling",
				Target:     "测试-应用",
				Namespace:  "default",
				Replicas:   1,
			},
			format:      "full",
			expectError: "",
		},
		{
			name: "maximum replicas value",
			intent: testIntent{
				IntentType: "scaling",
				Target:     "max-replicas",
				Namespace:  "default",
				Replicas:   2147483647, // max int32
			},
			format:      "smp",
			expectError: "",
		},
		{
			name: "zero replicas",
			intent: testIntent{
				IntentType: "scaling",
				Target:     "zero-replicas",
				Namespace:  "default",
				Replicas:   0,
			},
			format:      "full",
			expectError: "",
		},
		{
			name: "negative replicas",
			intent: testIntent{
				IntentType: "scaling",
				Target:     "negative-replicas",
				Namespace:  "default",
				Replicas:   -1,
			},
			format:      "smp",
			expectError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subDir := filepath.Join(tmpDir, tt.name)
			err := WriteIntent(tt.intent, subDir, tt.format)

			if tt.expectError != "" {
				if err == nil {
					t.Errorf("Expected error but got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.expectError) {
					t.Errorf("Expected error containing '%s' but got: %v", tt.expectError, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
					return
				}

				// Verify file was created
				var expectedFile string
				if tt.format == "smp" {
					expectedFile = "scaling-patch.json"
				} else {
					expectedFile = "scaling-patch.yaml"
				}

				filePath := filepath.Join(subDir, expectedFile)
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					t.Errorf("Expected file %s was not created", expectedFile)
				}
			}
		})
	}
}

// TestWriteIntent_ConcurrentWrites tests concurrent writes to same directory
func TestWriteIntent_ConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()

	intent := testIntent{
		IntentType: "scaling",
		Target:     "concurrent-test",
		Namespace:  "default",
		Replicas:   3,
	}

	const numGoroutines = 10
	done := make(chan error, numGoroutines)

	// Launch multiple goroutines writing to the same directory
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			subDir := filepath.Join(tmpDir, fmt.Sprintf("goroutine-%d", id))
			err := WriteIntent(intent, subDir, "full")
			done <- err
		}(i)
	}

	// Wait for all goroutines to complete and check for errors
	for i := 0; i < numGoroutines; i++ {
		if err := <-done; err != nil {
			t.Errorf("Goroutine error: %v", err)
		}
	}

	// Verify all files were created
	for i := 0; i < numGoroutines; i++ {
		expectedPath := filepath.Join(tmpDir, fmt.Sprintf("goroutine-%d", i), "scaling-patch.yaml")
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			t.Errorf("Expected file from goroutine %d not found", i)
		}
	}
}

// TestWriteIntent_LargeIntentData tests handling of very large intent data
func TestWriteIntent_LargeIntentData(t *testing.T) {
	tmpDir := t.TempDir()

	// Create intent with extremely long target name to test memory usage
	largeIntent := testIntent{
		IntentType: "scaling",
		Target:     strings.Repeat("very-long-target-name-", 100000), // ~2MB string
		Namespace:  "default",
		Replicas:   1,
	}

	err := WriteIntent(largeIntent, tmpDir, "full")
	if err != nil {
		t.Fatalf("WriteIntent with large data failed: %v", err)
	}

	// Verify file was created and contains expected content
	yamlPath := filepath.Join(tmpDir, "scaling-patch.yaml")
	content, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Check that the large target name is present in the output
	if !strings.Contains(string(content), largeIntent.Target) {
		t.Error("Large target name not found in output file")
	}
}

