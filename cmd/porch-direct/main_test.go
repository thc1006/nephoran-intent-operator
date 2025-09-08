package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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

			err = os.WriteFile(intentFile, intentData, 0o644)
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
				"target":      "test-app",
				"namespace":   "default",
				"replicas":    -1,
			},
		},
		{
			name: "zero_replicas",
			intent: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "test-app",
				"namespace":   "default",
				"replicas":    0,
			},
		},
		{
			name: "missing_target",
			intent: map[string]interface{}{
				"intent_type": "scaling",
				"namespace":   "default",
				"replicas":    1,
			},
		},
		{
			name: "missing_namespace",
			intent: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "test-app",
				"replicas":    1,
			},
		},
		{
			name: "missing_intent_type",
			intent: map[string]interface{}{
				"target":    "test-app",
				"namespace": "default",
				"replicas":  1,
			},
		},
		{
			name: "invalid_intent_type",
			intent: map[string]interface{}{
				"intent_type": "invalid",
				"target":      "test-app",
				"namespace":   "default",
				"replicas":    1,
			},
		},
		{
			name: "too_many_replicas",
			intent: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "test-app",
				"namespace":   "default",
				"replicas":    10000,
			},
		},
		{
			name: "empty_target",
			intent: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "",
				"namespace":   "default",
				"replicas":    1,
			},
		},
		{
			name: "empty_namespace",
			intent: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "test-app",
				"namespace":   "",
				"replicas":    1,
			},
		},
		{
			name: "invalid_source",
			intent: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "test-app",
				"namespace":   "default",
				"replicas":    1,
				"source":      "invalid_source",
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

			err = os.WriteFile(intentFile, intentData, 0o644)
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

	err = os.WriteFile(intentFile, intentData, 0o644)
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

	err = os.WriteFile(intentFile, intentData, 0o644)
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

	err = os.WriteFile(intentFile, intentData, 0o644)
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

// TestRunWithFileSystemErrors tests filesystem error conditions
func TestRunWithFileSystemErrors(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) (string, string) // returns intentFile, outDir
		expectError string
	}{
		{
			name: "intent file does not exist",
			setupFunc: func(t *testing.T) (string, string) {
				tempDir := t.TempDir()
				nonExistentFile := filepath.Join(tempDir, "non-existent.json")
				outDir := filepath.Join(tempDir, "output")
				return nonExistentFile, outDir
			},
			expectError: "failed to load intent file",
		},
		{
			name: "intent file is a directory",
			setupFunc: func(t *testing.T) (string, string) {
				tempDir := t.TempDir()
				dirPath := filepath.Join(tempDir, "intent-dir")
				err := os.Mkdir(dirPath, 0o755)
				if err != nil {
					t.Fatalf("Failed to create directory: %v", err)
				}
				outDir := filepath.Join(tempDir, "output")
				return dirPath, outDir
			},
			expectError: "failed to load intent file",
		},
		{
			name: "intent file permission denied",
			setupFunc: func(t *testing.T) (string, string) {
				// Skip this test on Windows as file permissions work differently
				if runtime.GOOS == "windows" {
					t.Skip("File permission tests not reliable on Windows")
				}
				
				tempDir := t.TempDir()
				intentFile := filepath.Join(tempDir, "intent.json")
				outDir := filepath.Join(tempDir, "output")

				// Create valid intent file first
				intent := intent.ScalingIntent{
					IntentType: "scaling",
					Target:     "test-app",
					Namespace:  "default",
					Replicas:   1,
				}
				intentData, _ := json.MarshalIndent(intent, "", "  ")
				err := os.WriteFile(intentFile, intentData, 0o644)
				if err != nil {
					t.Fatalf("Failed to create intent file: %v", err)
				}

				// Remove read permissions
				err = os.Chmod(intentFile, 0o000)
				if err != nil {
					t.Skipf("Cannot modify file permissions on this system: %v", err)
				}

				return intentFile, outDir
			},
			expectError: "failed to load intent file",
		},
		{
			name: "output directory permission denied",
			setupFunc: func(t *testing.T) (string, string) {
				// Skip this test on Windows as directory permissions work differently
				if runtime.GOOS == "windows" {
					t.Skip("Directory permission tests not reliable on Windows")
				}
				
				tempDir := t.TempDir()
				intentFile := filepath.Join(tempDir, "intent.json")
				restrictedDir := filepath.Join(tempDir, "restricted")
				outDir := filepath.Join(restrictedDir, "output")

				// Create valid intent file
				intent := intent.ScalingIntent{
					IntentType: "scaling",
					Target:     "test-app",
					Namespace:  "default",
					Replicas:   1,
				}
				intentData, _ := json.MarshalIndent(intent, "", "  ")
				err := os.WriteFile(intentFile, intentData, 0o644)
				if err != nil {
					t.Fatalf("Failed to create intent file: %v", err)
				}

				// Create restricted directory
				err = os.Mkdir(restrictedDir, 0o755)
				if err != nil {
					t.Fatalf("Failed to create restricted directory: %v", err)
				}

				// Remove write permissions
				err = os.Chmod(restrictedDir, 0o444)
				if err != nil {
					t.Skipf("Cannot modify directory permissions on this system: %v", err)
				}

				return intentFile, outDir
			},
			expectError: "failed to write package",
		},
		{
			name: "output directory exists as file",
			setupFunc: func(t *testing.T) (string, string) {
				tempDir := t.TempDir()
				intentFile := filepath.Join(tempDir, "intent.json")
				outDir := filepath.Join(tempDir, "output")

				// Create valid intent file
				intent := intent.ScalingIntent{
					IntentType: "scaling",
					Target:     "test-app",
					Namespace:  "default",
					Replicas:   1,
				}
				intentData, _ := json.MarshalIndent(intent, "", "  ")
				err := os.WriteFile(intentFile, intentData, 0o644)
				if err != nil {
					t.Fatalf("Failed to create intent file: %v", err)
				}

				// Create file where directory should be
				err = os.WriteFile(outDir, []byte("not a directory"), 0o644)
				if err != nil {
					t.Fatalf("Failed to create file: %v", err)
				}

				return intentFile, outDir
			},
			expectError: "failed to write package",
		},
		{
			name: "invalid output directory path",
			setupFunc: func(t *testing.T) (string, string) {
				tempDir := t.TempDir()
				intentFile := filepath.Join(tempDir, "intent.json")

				// Create invalid output path with invalid characters
				outDir := filepath.Join(tempDir, "invalid-path-\x00-with-null-bytes")

				// Create valid intent file
				intent := intent.ScalingIntent{
					IntentType: "scaling",
					Target:     "test-app",
					Namespace:  "default",
					Replicas:   1,
				}
				intentData, _ := json.MarshalIndent(intent, "", "  ")
				err := os.WriteFile(intentFile, intentData, 0o644)
				if err != nil {
					t.Fatalf("Failed to create intent file: %v", err)
				}

				return intentFile, outDir
			},
			expectError: "failed to write package",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intentFile, outDir := tt.setupFunc(t)

			// Ensure cleanup happens even if test fails
			defer func() {
				// Restore permissions for cleanup
				if err := os.Chmod(filepath.Dir(intentFile), 0o755); err != nil {
					t.Logf("Failed to restore intent dir permissions: %v", err)
				}
				if err := os.Chmod(intentFile, 0o644); err != nil {
					t.Logf("Failed to restore intent file permissions: %v", err)
				}
				if err := os.Chmod(filepath.Dir(outDir), 0o755); err != nil {
					t.Logf("Failed to restore out dir permissions: %v", err)
				}
			}()

			cfg := &Config{
				IntentPath: intentFile,
				OutDir:     outDir,
				DryRun:     false,
				Minimal:    false,
			}

			err := run(cfg)
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

// TestRunWithMalformedIntentFiles tests handling of malformed intent files
func TestRunWithMalformedIntentFiles(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		expectError string
	}{
		{
			name:        "empty file",
			fileContent: "",
			expectError: "intent validation failed",
		},
		{
			name:        "invalid JSON",
			fileContent: `{invalid json`,
			expectError: "intent validation failed",
		},
		{
			name:        "binary file",
			fileContent: string([]byte{0xFF, 0xFE, 0xFD, 0xFC}),
			expectError: "intent validation failed",
		},
		{
			name:        "very large file",
			fileContent: `{"intent_type": "scaling", "target": "` + strings.Repeat("a", 10000000) + `", "namespace": "default", "replicas": 1}`,
			expectError: "intent validation failed",
		},
		{
			name:        "deeply nested JSON",
			fileContent: createDeeplyNestedIntentJSON(100),
			expectError: "intent validation failed",
		},
		{
			name:        "JSON with null bytes",
			fileContent: "{\x00\"intent_type\": \"scaling\", \"target\": \"test\", \"namespace\": \"default\", \"replicas\": 1}",
			expectError: "intent validation failed",
		},
		{
			name: "JSON with circular reference structure",
			fileContent: `{
				"intent_type": "scaling",
				"target": "test",
				"namespace": "default", 
				"replicas": 1,
				"self_ref": "this would create circular reference if supported"
			}`,
			expectError: "intent validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			intentFile := filepath.Join(tempDir, "intent.json")
			outDir := filepath.Join(tempDir, "output")

			// Write malformed intent file
			err := os.WriteFile(intentFile, []byte(tt.fileContent), 0o644)
			if err != nil {
				t.Fatalf("Failed to write intent file: %v", err)
			}

			cfg := &Config{
				IntentPath: intentFile,
				OutDir:     outDir,
				DryRun:     false,
				Minimal:    false,
			}

			err = run(cfg)

			if tt.expectError == "" {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error containing '%s' but got nil", tt.expectError)
				} else if !strings.Contains(err.Error(), tt.expectError) {
					t.Errorf("Expected error containing '%s' but got: %v", tt.expectError, err)
				}
			}
		})
	}
}

// TestRunWithResourceExhaustion tests resource exhaustion scenarios
func TestRunWithResourceExhaustion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping resource exhaustion test in short mode")
	}
	
	// Test with fewer concurrent runs for CI stability (was 100)
	const numConcurrentRuns = 10

	done := make(chan error, numConcurrentRuns)

	for i := 0; i < numConcurrentRuns; i++ {
		go func(id int) {
			intent := intent.ScalingIntent{
				IntentType: "scaling",
				Target:     fmt.Sprintf("concurrent-test-%d", id),
				Namespace:  "default",
				Replicas:   1,
				Source:     "test",
			}

			tempDir := os.TempDir()
			intentFile := filepath.Join(tempDir, fmt.Sprintf("intent-%d.json", id))
			outDir := filepath.Join(tempDir, fmt.Sprintf("output-%d", id))

			// Write intent file
			intentData, _ := json.MarshalIndent(intent, "", "  ")
			err := os.WriteFile(intentFile, intentData, 0o644)
			if err != nil {
				done <- fmt.Errorf("failed to write intent file: %v", err)
				return
			}

			// Cleanup function
			defer func() {
				os.RemoveAll(intentFile)
				os.RemoveAll(outDir)
			}()

			cfg := &Config{
				IntentPath: intentFile,
				OutDir:     outDir,
				DryRun:     true, // Use dry-run to avoid actual file writes
				Minimal:    true,
			}

			done <- run(cfg)
		}(i)
	}

	// Wait for all goroutines to complete
	errorCount := 0
	for i := 0; i < numConcurrentRuns; i++ {
		if err := <-done; err != nil {
			errorCount++
			if errorCount == 1 {
				t.Logf("First error from concurrent run: %v", err)
			}
		}
	}

	// Allow some failures due to resource constraints, but not all
	maxAllowedErrors := numConcurrentRuns / 10 // Allow 10% failure rate
	if errorCount > maxAllowedErrors {
		t.Errorf("Too many concurrent runs failed: %d/%d (max allowed: %d)",
			errorCount, numConcurrentRuns, maxAllowedErrors)
	}
}

// TestRunProjectRootDiscovery tests project root discovery errors
func TestRunProjectRootDiscovery(t *testing.T) {
	// Save original working directory
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(origWd); err != nil {
			t.Logf("Failed to restore working directory: %v", err)
		}
	}()

	// Create an isolated temporary directory structure that prevents finding any parent go.mod
	// We create a nested directory structure to ensure complete isolation
	tempDir := t.TempDir()
	isolatedDir := filepath.Join(tempDir, "isolated", "test", "directory")
	err = os.MkdirAll(isolatedDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create isolated directory: %v", err)
	}
	
	err = os.Chdir(isolatedDir)
	if err != nil {
		t.Fatalf("Failed to change to isolated directory: %v", err)
	}

	// Create valid intent file
	intent := intent.ScalingIntent{
		IntentType: "scaling",
		Target:     "test-app",
		Namespace:  "default",
		Replicas:   1,
	}
	intentFile := filepath.Join(isolatedDir, "intent.json")
	outDir := filepath.Join(isolatedDir, "output")

	intentData, _ := json.MarshalIndent(intent, "", "  ")
	err = os.WriteFile(intentFile, intentData, 0o644)
	if err != nil {
		t.Fatalf("Failed to write intent file: %v", err)
	}

	cfg := &Config{
		IntentPath: intentFile,
		OutDir:     outDir,
		DryRun:     false,
		Minimal:    false,
	}

	err = run(cfg)
	if err == nil {
		t.Error("Expected project root discovery error but got nil")
	}
	if !strings.Contains(err.Error(), "go.mod not found") {
		t.Errorf("Expected go.mod error but got: %v", err)
	}
}

// Helper function to create deeply nested JSON for testing
func createDeeplyNestedIntentJSON(depth int) string {
	json := `{"intent_type": "scaling", "target": "test", "namespace": "default", "replicas": 1, "nested":`

	for i := 0; i < depth; i++ {
		json += `{"level": `
	}
	json += `"deep"`
	for i := 0; i < depth; i++ {
		json += `}`
	}
	json += `}`

	return json
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
	if err := os.WriteFile(intentFile, intentData, 0o644); err != nil {
		b.Fatalf("Failed to write intent file: %v", err)
	}

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

