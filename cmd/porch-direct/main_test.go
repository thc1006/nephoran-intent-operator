package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/thc1006/nephoran-intent-operator/internal/intent"
)

// TestData contains test intent files and expected results
type TestData struct {
	name           string
	intent         intent.ScalingIntent
	expectValid    bool
	expectReplicas int
	expectTarget   string
	expectNS       string
}

func TestRunWithSampleIntent(t *testing.T) {
	tests := []TestData{
		{
			name: "valid_sample_intent",
			intent: intent.ScalingIntent{
				IntentType: "scaling",
				Target:     "nf-sim",
				Namespace:  "ran-a",
				Replicas:   3,
				Source:     "test",
			},
			expectValid:    true,
			expectReplicas: 3,
			expectTarget:   "nf-sim",
			expectNS:       "ran-a",
		},
		{
			name: "valid_minimal_intent",
			intent: intent.ScalingIntent{
				IntentType: "scaling",
				Target:     "another-sim",
				Namespace:  "ran-b",
				Replicas:   5,
			},
			expectValid:    true,
			expectReplicas: 5,
			expectTarget:   "another-sim",
			expectNS:       "ran-b",
		},
		{
			name: "valid_with_reason_and_correlation",
			intent: intent.ScalingIntent{
				IntentType:    "scaling",
				Target:        "test-sim",
				Namespace:     "test-ns",
				Replicas:      2,
				Source:        "user",
				Reason:        "Load testing scenario",
				CorrelationID: "test-corr-123",
			},
			expectValid:    true,
			expectReplicas: 2,
			expectTarget:   "test-sim",
			expectNS:       "test-ns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directories for test
			tempDir := t.TempDir()
			intentFile := filepath.Join(tempDir, "intent.json")
			outDir := filepath.Join(tempDir, "output")

			// Write intent file
			intentData, err := json.MarshalIndent(tt.intent, "", "  ")
			if err != nil {
				t.Fatalf("Failed to marshal intent: %v", err)
			}

			err = os.WriteFile(intentFile, intentData, 0644)
			if err != nil {
				t.Fatalf("Failed to write intent file: %v", err)
			}

			// Run the command
			cfg := &Config{
				IntentPath: intentFile,
				OutDir:     outDir,
				DryRun:     false,
				Minimal:    false,
			}
			err = run(cfg)
			if !tt.expectValid && err == nil {
				t.Fatalf("Expected error for invalid intent, but got none")
			}
			if tt.expectValid && err != nil {
				t.Fatalf("Expected success but got error: %v", err)
			}

			if !tt.expectValid {
				return // Skip validation for invalid intents
			}

			// Validate generated files
			packageDir := filepath.Join(outDir, tt.expectTarget+"-scaling")
			validateGeneratedPackage(t, packageDir, tt)
		})
	}
}

func TestRunWithInvalidIntents(t *testing.T) {
	invalidTests := []struct {
		name   string
		intent map[string]interface{}
	}{
		{
			name: "negative_replicas",
			intent: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "nf-sim",
				"namespace":   "ran-a",
				"replicas":    -1,
			},
		},
		{
			name: "zero_replicas",
			intent: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "nf-sim",
				"namespace":   "ran-a",
				"replicas":    0,
			},
		},
		{
			name: "missing_target",
			intent: map[string]interface{}{
				"intent_type": "scaling",
				"namespace":   "ran-a",
				"replicas":    3,
			},
		},
		{
			name: "missing_namespace",
			intent: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "nf-sim",
				"replicas":    3,
			},
		},
		{
			name: "missing_intent_type",
			intent: map[string]interface{}{
				"target":    "nf-sim",
				"namespace": "ran-a",
				"replicas":  3,
			},
		},
		{
			name: "invalid_intent_type",
			intent: map[string]interface{}{
				"intent_type": "wrong",
				"target":      "nf-sim",
				"namespace":   "ran-a",
				"replicas":    3,
			},
		},
		{
			name: "too_many_replicas",
			intent: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "nf-sim",
				"namespace":   "ran-a",
				"replicas":    101,
			},
		},
		{
			name: "empty_target",
			intent: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "",
				"namespace":   "ran-a",
				"replicas":    3,
			},
		},
		{
			name: "empty_namespace",
			intent: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "nf-sim",
				"namespace":   "",
				"replicas":    3,
			},
		},
		{
			name: "invalid_source",
			intent: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "nf-sim",
				"namespace":   "ran-a",
				"replicas":    3,
				"source":      "invalid",
			},
		},
	}

	for _, tt := range invalidTests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directories for test
			tempDir := t.TempDir()
			intentFile := filepath.Join(tempDir, "intent.json")
			outDir := filepath.Join(tempDir, "output")

			// Write intent file
			intentData, err := json.MarshalIndent(tt.intent, "", "  ")
			if err != nil {
				t.Fatalf("Failed to marshal intent: %v", err)
			}

			err = os.WriteFile(intentFile, intentData, 0644)
			if err != nil {
				t.Fatalf("Failed to write intent file: %v", err)
			}

			// Run the command - should fail validation
			cfg := &Config{
				IntentPath: intentFile,
				OutDir:     outDir,
				DryRun:     false,
				Minimal:    false,
			}
			err = run(cfg)
			if err == nil {
				t.Fatalf("Expected validation error for %s, but got none", tt.name)
			}

			// Verify error message contains relevant information
			if !strings.Contains(err.Error(), "validation failed") {
				t.Errorf("Expected validation error message, got: %v", err)
			}
		})
	}
}

func TestIdempotency(t *testing.T) {
	// Create sample intent
	intent := intent.ScalingIntent{
		IntentType: "scaling",
		Target:     "idempotent-test",
		Namespace:  "test-ns",
		Replicas:   2,
		Source:     "test",
	}

	// Create temporary directories for test
	tempDir := t.TempDir()
	intentFile := filepath.Join(tempDir, "intent.json")
	outDir := filepath.Join(tempDir, "output")

	// Write intent file
	intentData, err := json.MarshalIndent(intent, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal intent: %v", err)
	}

	err = os.WriteFile(intentFile, intentData, 0644)
	if err != nil {
		t.Fatalf("Failed to write intent file: %v", err)
	}

	// Run the command first time
	cfg := &Config{
		IntentPath: intentFile,
		OutDir:     outDir,
		DryRun:     false,
		Minimal:    false,
	}
	err = run(cfg)
	if err != nil {
		t.Fatalf("First run failed: %v", err)
	}

	packageDir := filepath.Join(outDir, "idempotent-test-scaling")

	// Collect file modification times after first run
	firstRunModTimes := make(map[string]int64)
	entries, err := os.ReadDir(packageDir)
	if err != nil {
		t.Fatalf("Failed to read package directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			filePath := filepath.Join(packageDir, entry.Name())
			info, err := os.Stat(filePath)
			if err != nil {
				t.Fatalf("Failed to stat file %s: %v", filePath, err)
			}
			firstRunModTimes[entry.Name()] = info.ModTime().Unix()
		}
	}

	// Run the command second time - should be idempotent
	err = run(cfg)
	if err != nil {
		t.Fatalf("Second run failed: %v", err)
	}

	// Check that file modification times haven't changed
	for filename, firstModTime := range firstRunModTimes {
		filePath := filepath.Join(packageDir, filename)
		info, err := os.Stat(filePath)
		if err != nil {
			t.Fatalf("Failed to stat file %s after second run: %v", filePath, err)
		}

		secondModTime := info.ModTime().Unix()
		if secondModTime != firstModTime {
			t.Errorf("File %s was modified on second run (idempotency violated)", filename)
		}
	}
}

func TestMinimalPackageGeneration(t *testing.T) {
	intent := intent.ScalingIntent{
		IntentType: "scaling",
		Target:     "minimal-test",
		Namespace:  "test-ns",
		Replicas:   1,
		Source:     "test",
	}

	// Create temporary directories for test
	tempDir := t.TempDir()
	intentFile := filepath.Join(tempDir, "intent.json")
	outDir := filepath.Join(tempDir, "output")

	// Write intent file
	intentData, err := json.MarshalIndent(intent, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal intent: %v", err)
	}

	err = os.WriteFile(intentFile, intentData, 0644)
	if err != nil {
		t.Fatalf("Failed to write intent file: %v", err)
	}

	// Run with minimal flag
	cfg := &Config{
		IntentPath: intentFile,
		OutDir:     outDir,
		DryRun:     false,
		Minimal:    true,
	}
	err = run(cfg)
	if err != nil {
		t.Fatalf("Minimal package generation failed: %v", err)
	}

	packageDir := filepath.Join(outDir, "minimal-test-minimal")

	// Verify only minimal files are present
	expectedFiles := []string{"Kptfile", "deployment.yaml"}
	unexpectedFiles := []string{"service.yaml", "README.md"}

	for _, file := range expectedFiles {
		filePath := filepath.Join(packageDir, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Expected file %s not found in minimal package", file)
		}
	}

	for _, file := range unexpectedFiles {
		filePath := filepath.Join(packageDir, file)
		if _, err := os.Stat(filePath); err == nil {
			t.Errorf("Unexpected file %s found in minimal package", file)
		}
	}
}

func TestDryRun(t *testing.T) {
	intent := intent.ScalingIntent{
		IntentType: "scaling",
		Target:     "dry-run-test",
		Namespace:  "test-ns",
		Replicas:   1,
		Source:     "test",
	}

	// Create temporary directories for test
	tempDir := t.TempDir()
	intentFile := filepath.Join(tempDir, "intent.json")
	outDir := filepath.Join(tempDir, "output")

	// Write intent file
	intentData, err := json.MarshalIndent(intent, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal intent: %v", err)
	}

	err = os.WriteFile(intentFile, intentData, 0644)
	if err != nil {
		t.Fatalf("Failed to write intent file: %v", err)
	}

	// Run in dry-run mode
	cfg := &Config{
		IntentPath: intentFile,
		OutDir:     outDir,
		DryRun:     true,
		Minimal:    false,
	}
	err = run(cfg)
	if err != nil {
		t.Fatalf("Dry run failed: %v", err)
	}

	// Verify no files were actually written
	packageDir := filepath.Join(outDir, "dry-run-test-scaling")
	if _, err := os.Stat(packageDir); err == nil {
		t.Errorf("Package directory should not exist in dry-run mode")
	}
}

// validateGeneratedPackage validates the generated KRM package contents
func validateGeneratedPackage(t *testing.T, packageDir string, expected TestData) {
	t.Helper()

	// Check that package directory exists
	if _, err := os.Stat(packageDir); os.IsNotExist(err) {
		t.Fatalf("Package directory %s does not exist", packageDir)
	}

	// Validate Kptfile
	kptfilePath := filepath.Join(packageDir, "Kptfile")
	validateKptfile(t, kptfilePath, expected)

	// Validate Deployment
	deploymentPath := filepath.Join(packageDir, "deployment.yaml")
	validateDeployment(t, deploymentPath, expected)

	// Validate Service (if not minimal)
	servicePath := filepath.Join(packageDir, "service.yaml")
	if _, err := os.Stat(servicePath); err == nil {
		validateService(t, servicePath, expected)
	}

	// Validate README (if not minimal)
	readmePath := filepath.Join(packageDir, "README.md")
	if _, err := os.Stat(readmePath); err == nil {
		validateReadme(t, readmePath, expected)
	}
}

func validateKptfile(t *testing.T, kptfilePath string, expected TestData) {
	t.Helper()

	content, err := os.ReadFile(kptfilePath)
	if err != nil {
		t.Fatalf("Failed to read Kptfile: %v", err)
	}

	var kptfile map[string]interface{}
	err = yaml.Unmarshal(content, &kptfile)
	if err != nil {
		t.Fatalf("Failed to parse Kptfile YAML: %v", err)
	}

	// Check basic structure - Go YAML marshaling uses field names, not yaml tags
	apiVersion, ok := kptfile["APIVersion"]
	if !ok {
		t.Errorf("APIVersion field missing from Kptfile")
	} else if apiVersion != "kpt.dev/v1" {
		t.Errorf("Expected APIVersion kpt.dev/v1, got %v", apiVersion)
	}

	kind, ok := kptfile["Kind"]
	if !ok {
		t.Errorf("Kind field missing from Kptfile")
	} else if kind != "Kptfile" {
		t.Errorf("Expected Kind Kptfile, got %v", kind)
	}

	// Check metadata
	metadataInterface, ok := kptfile["Metadata"]
	if !ok {
		t.Fatalf("Kptfile Metadata is missing")
	}

	metadata, ok := metadataInterface.(map[string]interface{})
	if !ok {
		t.Fatalf("Kptfile metadata is not a map, got %T: %v", metadataInterface, metadataInterface)
	}

	nameInterface, ok := metadata["Name"]
	if !ok {
		t.Errorf("Name field missing from metadata")
	} else {
		name, ok := nameInterface.(string)
		if !ok {
			t.Errorf("Name is not a string, got %T: %v", nameInterface, nameInterface)
		} else if !strings.Contains(name, expected.expectTarget) {
			t.Errorf("Expected Kptfile Name to contain %s, got %v", expected.expectTarget, name)
		}
	}

	// Check info section
	infoInterface, ok := kptfile["Info"]
	if !ok {
		t.Fatalf("Kptfile Info is missing")
	}

	info, ok := infoInterface.(map[string]interface{})
	if !ok {
		t.Fatalf("Kptfile info is not a map, got %T: %v", infoInterface, infoInterface)
	}

	descriptionInterface, ok := info["Description"]
	if !ok {
		t.Errorf("Description field missing from info")
	} else {
		description, ok := descriptionInterface.(string)
		if !ok {
			t.Errorf("Description is not a string, got %T: %v", descriptionInterface, descriptionInterface)
		} else if !strings.Contains(description, expected.expectTarget) {
			t.Errorf("Expected Description to contain %s, got %v", expected.expectTarget, description)
		}
	}
}

func validateDeployment(t *testing.T, deploymentPath string, expected TestData) {
	t.Helper()

	content, err := os.ReadFile(deploymentPath)
	if err != nil {
		t.Fatalf("Failed to read deployment.yaml: %v", err)
	}

	var deployment map[string]interface{}
	err = yaml.Unmarshal(content, &deployment)
	if err != nil {
		t.Fatalf("Failed to parse deployment YAML: %v", err)
	}

	// Check basic structure
	if deployment["apiVersion"] != "apps/v1" {
		t.Errorf("Expected apiVersion apps/v1, got %v", deployment["apiVersion"])
	}

	if deployment["kind"] != "Deployment" {
		t.Errorf("Expected kind Deployment, got %v", deployment["kind"])
	}

	// Check metadata
	metadata, ok := deployment["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("Deployment metadata is not a map")
	}

	name, ok := metadata["name"].(string)
	if !ok || name != expected.expectTarget {
		t.Errorf("Expected deployment name %s, got %v", expected.expectTarget, name)
	}

	namespace, ok := metadata["namespace"].(string)
	if !ok || namespace != expected.expectNS {
		t.Errorf("Expected deployment namespace %s, got %v", expected.expectNS, namespace)
	}

	// Check spec
	spec, ok := deployment["spec"].(map[string]interface{})
	if !ok {
		t.Fatalf("Deployment spec is not a map")
	}

	replicas, ok := spec["replicas"].(int)
	if !ok || replicas != expected.expectReplicas {
		t.Errorf("Expected deployment replicas %d, got %v", expected.expectReplicas, replicas)
	}

	// Check selector
	selector, ok := spec["selector"].(map[string]interface{})
	if !ok {
		t.Fatalf("Deployment selector is not a map")
	}

	matchLabels, ok := selector["matchLabels"].(map[string]interface{})
	if !ok {
		t.Fatalf("Deployment selector matchLabels is not a map")
	}

	app, ok := matchLabels["app"].(string)
	if !ok || app != expected.expectTarget {
		t.Errorf("Expected selector app %s, got %v", expected.expectTarget, app)
	}
}

func validateService(t *testing.T, servicePath string, expected TestData) {
	t.Helper()

	content, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatalf("Failed to read service.yaml: %v", err)
	}

	var service map[string]interface{}
	err = yaml.Unmarshal(content, &service)
	if err != nil {
		t.Fatalf("Failed to parse service YAML: %v", err)
	}

	// Check basic structure
	if service["apiVersion"] != "v1" {
		t.Errorf("Expected apiVersion v1, got %v", service["apiVersion"])
	}

	if service["kind"] != "Service" {
		t.Errorf("Expected kind Service, got %v", service["kind"])
	}

	// Check metadata
	metadata, ok := service["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("Service metadata is not a map")
	}

	name, ok := metadata["name"].(string)
	if !ok || !strings.Contains(name, expected.expectTarget) {
		t.Errorf("Expected service name to contain %s, got %v", expected.expectTarget, name)
	}

	namespace, ok := metadata["namespace"].(string)
	if !ok || namespace != expected.expectNS {
		t.Errorf("Expected service namespace %s, got %v", expected.expectNS, namespace)
	}
}

func validateReadme(t *testing.T, readmePath string, expected TestData) {
	t.Helper()

	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("Failed to read README.md: %v", err)
	}

	readmeContent := string(content)

	// Check that README contains expected values
	if !strings.Contains(readmeContent, expected.expectTarget) {
		t.Errorf("README should contain target %s", expected.expectTarget)
	}

	if !strings.Contains(readmeContent, expected.expectNS) {
		t.Errorf("README should contain namespace %s", expected.expectNS)
	}

	replicasStr := string(rune(expected.expectReplicas + '0'))
	if !strings.Contains(readmeContent, replicasStr) {
		t.Errorf("README should contain replicas %d", expected.expectReplicas)
	}
}

func TestFindProjectRoot(t *testing.T) {
	// Test findProjectRoot function
	root, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	// Verify go.mod exists in the found root
	gomodPath := filepath.Join(root, "go.mod")
	if _, err := os.Stat(gomodPath); os.IsNotExist(err) {
		t.Errorf("go.mod not found in project root %s", root)
	}

	// Verify this is actually the nephoran project
	content, err := os.ReadFile(gomodPath)
	if err != nil {
		t.Fatalf("Failed to read go.mod: %v", err)
	}

	if !strings.Contains(string(content), "github.com/thc1006/nephoran-intent-operator") {
		t.Errorf("Project root does not appear to be the nephoran project")
	}
}

// Benchmark tests
func BenchmarkRunValidIntent(b *testing.B) {
	intent := intent.ScalingIntent{
		IntentType: "scaling",
		Target:     "benchmark-test",
		Namespace:  "bench-ns",
		Replicas:   2,
		Source:     "test",
	}

	// Create temporary directories for benchmark
	tempDir := b.TempDir()
	intentFile := filepath.Join(tempDir, "intent.json")

	// Write intent file
	intentData, _ := json.MarshalIndent(intent, "", "  ")
	os.WriteFile(intentFile, intentData, 0644)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		outDir := filepath.Join(tempDir, "output", fmt.Sprintf("run-%d", i))
		cfg := &Config{
			IntentPath: intentFile,
			OutDir:     outDir,
			DryRun:     true,
			Minimal:    false,
		}
		err := run(cfg) // Use dry-run for benchmark
		if err != nil {
			b.Fatalf("Benchmark run failed: %v", err)
		}
	}
}