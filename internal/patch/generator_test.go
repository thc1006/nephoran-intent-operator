package patch

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	"sigs.k8s.io/yaml"
)

func TestNewGenerator(t *testing.T) {
	intent := &Intent{
		IntentType: "scaling",
		Target:     "test-deployment",
		Namespace:  "test-ns",
		Replicas:   3,
	}
	outputDir := "/tmp/test-output"

	gen := NewGenerator(intent, outputDir, logr.Discard())

	if gen.Intent != intent {
		t.Errorf("Expected intent to match, got different pointer")
	}
	if gen.OutputDir != outputDir {
		t.Errorf("Expected outputDir '%s', got '%s'", outputDir, gen.OutputDir)
	}
}

func TestGenerate(t *testing.T) {
	tempDir := t.TempDir()

	intent := &Intent{
		IntentType:    "scaling",
		Target:        "test-deployment",
		Namespace:     "test-ns",
		Replicas:      5,
		Reason:        "test scaling",
		Source:        "test",
		CorrelationID: "test-123",
	}

	gen := NewGenerator(intent, tempDir, logr.Discard())

	// Test successful generation
	err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Find the generated package directory
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read output directory: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 package directory, found %d", len(entries))
	}

	packageDir := filepath.Join(tempDir, entries[0].Name())

	// Verify all expected files exist
	expectedFiles := []string{
		"Kptfile",
		"deployment-patch.yaml",
		"setters.yaml",
		"README.md",
	}

	for _, filename := range expectedFiles {
		path := filepath.Join(packageDir, filename)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file '%s' does not exist", filename)
		}
	}

	// Verify Kptfile content
	kptfilePath := filepath.Join(packageDir, "Kptfile")
	kptfileData, err := os.ReadFile(kptfilePath)
	if err != nil {
		t.Fatalf("Failed to read Kptfile: %v", err)
	}

	var kptfile map[string]interface{}
	if err := yaml.Unmarshal(kptfileData, &kptfile); err != nil {
		t.Fatalf("Failed to parse Kptfile YAML: %v", err)
	}

	// Verify Kptfile structure
	if metadata, ok := kptfile["metadata"].(map[string]interface{}); ok {
		if name, ok := metadata["name"].(string); !ok || name == "" {
			t.Error("Kptfile metadata.name is missing or empty")
		}
		if annotations, ok := metadata["annotations"].(map[string]interface{}); ok {
			if localConfig, ok := annotations["config.kubernetes.io/local-config"].(string); !ok || localConfig != "true" {
				t.Error("Kptfile missing local-config annotation")
			}
		} else {
			t.Error("Kptfile metadata.annotations is missing")
		}
	} else {
		t.Error("Kptfile metadata is missing")
	}

	// Verify patch content
	patchPath := filepath.Join(packageDir, "deployment-patch.yaml")
	patchData, err := os.ReadFile(patchPath)
	if err != nil {
		t.Fatalf("Failed to read patch file: %v", err)
	}

	patchContent := string(patchData)
	if len(patchContent) == 0 {
		t.Error("Patch file is empty")
	}

	// Verify setters content
	settersPath := filepath.Join(packageDir, "setters.yaml")
	settersData, err := os.ReadFile(settersPath)
	if err != nil {
		t.Fatalf("Failed to read setters file: %v", err)
	}

	var setters map[string]interface{}
	if err := yaml.Unmarshal(settersData, &setters); err != nil {
		t.Fatalf("Failed to parse setters YAML: %v", err)
	}

	if data, ok := setters["data"].(map[string]interface{}); ok {
		if replicas, ok := data["replicas"].(string); !ok || replicas != "5" {
			t.Errorf("Expected replicas '5', got '%v'", data["replicas"])
		}
		if target, ok := data["target"].(string); !ok || target != "test-deployment" {
			t.Errorf("Expected target 'test-deployment', got '%v'", data["target"])
		}
		if namespace, ok := data["namespace"].(string); !ok || namespace != "test-ns" {
			t.Errorf("Expected namespace 'test-ns', got '%v'", data["namespace"])
		}
	} else {
		t.Error("Setters data section is missing or invalid")
	}

	// Verify README content
	readmePath := filepath.Join(packageDir, "README.md")
	readmeData, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("Failed to read README: %v", err)
	}

	readmeContent := string(readmeData)
	if len(readmeContent) == 0 {
		t.Error("README is empty")
	}
}

func TestGenerateWithInvalidOutputDir(t *testing.T) {
	intent := &Intent{
		IntentType: "scaling",
		Target:     "test-deployment",
		Namespace:  "test-ns",
		Replicas:   3,
	}

	// Use a path that would require root permissions
	gen := NewGenerator(intent, "/root/forbidden", logr.Discard())

	err := gen.Generate()
	if err == nil {
		t.Error("Expected error when writing to forbidden directory, got nil")
	}
}

func TestGenerateKptfile(t *testing.T) {
	tempDir := t.TempDir()
	packageDir := filepath.Join(tempDir, "test-package")

	if err := os.MkdirAll(packageDir, 0o755); err != nil {
		t.Fatalf("Failed to create package directory: %v", err)
	}

	intent := &Intent{
		IntentType: "scaling",
		Target:     "test-deployment",
		Namespace:  "test-ns",
		Replicas:   3,
	}

	gen := NewGenerator(intent, tempDir, logr.Discard())

	// Test Kptfile generation
	err := gen.generateKptfile(packageDir)
	if err != nil {
		t.Fatalf("generateKptfile() failed: %v", err)
	}

	// Verify file exists and is readable
	kptfilePath := filepath.Join(packageDir, "Kptfile")
	data, err := os.ReadFile(kptfilePath)
	if err != nil {
		t.Fatalf("Failed to read generated Kptfile: %v", err)
	}

	var kptfile map[string]interface{}
	if err := yaml.Unmarshal(data, &kptfile); err != nil {
		t.Fatalf("Generated Kptfile is not valid YAML: %v", err)
	}

	// Verify pipeline section exists
	if _, ok := kptfile["pipeline"]; !ok {
		t.Error("Kptfile missing pipeline section")
	}
}

func TestGeneratePatch(t *testing.T) {
	tempDir := t.TempDir()
	packageDir := filepath.Join(tempDir, "test-package")

	if err := os.MkdirAll(packageDir, 0o755); err != nil {
		t.Fatalf("Failed to create package directory: %v", err)
	}

	intent := &Intent{
		IntentType: "scaling",
		Target:     "my-app",
		Namespace:  "prod",
		Replicas:   10,
	}

	gen := NewGenerator(intent, tempDir, logr.Discard())

	// Test patch generation
	err := gen.generatePatch(packageDir)
	if err != nil {
		t.Fatalf("generatePatch() failed: %v", err)
	}

	// Verify file exists
	patchPath := filepath.Join(packageDir, "deployment-patch.yaml")
	data, err := os.ReadFile(patchPath)
	if err != nil {
		t.Fatalf("Failed to read generated patch: %v", err)
	}

	content := string(data)

	// Verify kpt-file comment is present
	if len(content) == 0 {
		t.Error("Patch file is empty")
	}

	// Should contain metadata with name and namespace
	if !contains(content, "my-app") {
		t.Error("Patch doesn't contain target deployment name")
	}
	if !contains(content, "prod") {
		t.Error("Patch doesn't contain namespace")
	}
}

func TestGenerateSetters(t *testing.T) {
	tempDir := t.TempDir()
	packageDir := filepath.Join(tempDir, "test-package")

	if err := os.MkdirAll(packageDir, 0o755); err != nil {
		t.Fatalf("Failed to create package directory: %v", err)
	}

	intent := &Intent{
		IntentType: "scaling",
		Target:     "api-server",
		Namespace:  "default",
		Replicas:   7,
	}

	gen := NewGenerator(intent, tempDir, logr.Discard())

	// Test setters generation
	err := gen.generateSetters(packageDir)
	if err != nil {
		t.Fatalf("generateSetters() failed: %v", err)
	}

	// Verify file exists and content
	settersPath := filepath.Join(packageDir, "setters.yaml")
	data, err := os.ReadFile(settersPath)
	if err != nil {
		t.Fatalf("Failed to read generated setters: %v", err)
	}

	var setters map[string]interface{}
	if err := yaml.Unmarshal(data, &setters); err != nil {
		t.Fatalf("Generated setters is not valid YAML: %v", err)
	}

	// Verify data section
	if data, ok := setters["data"].(map[string]interface{}); ok {
		expectedData := map[string]string{
			"replicas":  "7",
			"target":    "api-server",
			"namespace": "default",
		}

		for key, expectedValue := range expectedData {
			if actualValue, ok := data[key].(string); !ok || actualValue != expectedValue {
				t.Errorf("Expected setters.data.%s = '%s', got '%v'", key, expectedValue, data[key])
			}
		}
	} else {
		t.Error("Setters missing data section")
	}
}

func TestGenerateReadme(t *testing.T) {
	tempDir := t.TempDir()
	packageDir := filepath.Join(tempDir, "test-package")

	if err := os.MkdirAll(packageDir, 0o755); err != nil {
		t.Fatalf("Failed to create package directory: %v", err)
	}

	intent := &Intent{
		IntentType: "scaling",
		Target:     "web-server",
		Namespace:  "staging",
		Replicas:   15,
	}

	gen := NewGenerator(intent, tempDir, logr.Discard())

	// Test README generation
	err := gen.generateReadme(packageDir)
	if err != nil {
		t.Fatalf("generateReadme() failed: %v", err)
	}

	// Verify file exists and content
	readmePath := filepath.Join(packageDir, "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("Failed to read generated README: %v", err)
	}

	content := string(data)

	// Verify README contains intent details
	requiredStrings := []string{
		"web-server",
		"staging",
		"15",
		"KRM Scaling Patch",
		"Usage",
		"Files",
	}

	for _, required := range requiredStrings {
		if !contains(content, required) {
			t.Errorf("README missing expected content: '%s'", required)
		}
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
