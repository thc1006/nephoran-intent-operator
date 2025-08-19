package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestMainIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// Create temp directory for test files
	tmpDir := t.TempDir()
	
	// Create test intent file
	intent := Intent{
		IntentType: "scaling",
		Target:     "test-app",
		Namespace:  "test-namespace",
		Replicas:   3,
	}
	
	intentJSON, err := json.Marshal(intent)
	if err != nil {
		t.Fatalf("Failed to marshal intent: %v", err)
	}
	
	intentFile := filepath.Join(tmpDir, "test-intent.json")
	if err := os.WriteFile(intentFile, intentJSON, 0644); err != nil {
		t.Fatalf("Failed to write intent file: %v", err)
	}
	
	outputDir := filepath.Join(tmpDir, "output")
	
	// Test default format (full)
	t.Run("DefaultFormat", func(t *testing.T) {
		cmd := exec.Command("go", "run", "main.go", "-intent", intentFile, "-out", outputDir)
		cmd.Dir = "."
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Command failed: %v\nOutput: %s", err, output)
		}
		
		// Check YAML file exists
		yamlPath := filepath.Join(outputDir, "scaling-patch.yaml")
		if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
			t.Error("Expected YAML file not created")
		}
		
		content, _ := os.ReadFile(yamlPath)
		expected := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
  namespace: test-namespace
spec:
  replicas: 3
`
		if string(content) != expected {
			t.Errorf("Unexpected YAML content:\nGot:\n%s\nExpected:\n%s", content, expected)
		}
	})
	
	// Test SMP format
	t.Run("SMPFormat", func(t *testing.T) {
		outputDirSMP := filepath.Join(tmpDir, "output-smp")
		cmd := exec.Command("go", "run", "main.go", "-intent", intentFile, "-out", outputDirSMP, "-format", "smp")
		cmd.Dir = "."
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Command failed: %v\nOutput: %s", err, output)
		}
		
		// Check JSON file exists
		jsonPath := filepath.Join(outputDirSMP, "scaling-patch.json")
		if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
			t.Error("Expected JSON file not created")
		}
		
		content, _ := os.ReadFile(jsonPath)
		var smp map[string]interface{}
		if err := json.Unmarshal(content, &smp); err != nil {
			t.Fatalf("Invalid JSON output: %v", err)
		}
		
		// Validate structure
		if smp["apiVersion"] != "apps/v1" {
			t.Errorf("Wrong apiVersion: %v", smp["apiVersion"])
		}
		if smp["kind"] != "Deployment" {
			t.Errorf("Wrong kind: %v", smp["kind"])
		}
	})
	
	// Test invalid intent type
	t.Run("InvalidIntentType", func(t *testing.T) {
		invalidIntent := Intent{
			IntentType: "networking",
			Target:     "test-app",
			Namespace:  "test-namespace",
			Replicas:   3,
		}
		
		invalidJSON, _ := json.Marshal(invalidIntent)
		invalidFile := filepath.Join(tmpDir, "invalid-intent.json")
		os.WriteFile(invalidFile, invalidJSON, 0644)
		
		cmd := exec.Command("go", "run", "main.go", "-intent", invalidFile, "-out", tmpDir)
		cmd.Dir = "."
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("Expected command to fail for invalid intent type")
		}
		
		if !contains(string(output), "only scaling intent is supported") {
			t.Errorf("Expected error message about scaling intent, got: %s", output)
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[0:len(substr)] == substr || contains(s[1:], substr)))
}