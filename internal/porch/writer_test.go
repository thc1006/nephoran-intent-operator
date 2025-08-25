package porch

import (
	"encoding/json"
	"os"
	"path/filepath"
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
