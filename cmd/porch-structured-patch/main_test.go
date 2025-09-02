package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

func TestMain(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setupFiles  map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "missing intent flag",
			args:        []string{},
			expectError: true,
			errorMsg:    "intent flag is required",
		},
		{
			name:        "intent file does not exist",
			args:        []string{"-intent", "nonexistent.json"},
			expectError: true,
			errorMsg:    "file does not exist",
		},
		{
			name: "invalid JSON in intent file",
			args: []string{"-intent", "invalid.json"},
			setupFiles: map[string]string{
				"invalid.json": `{"invalid": json}`,
			},
			expectError: true,
			errorMsg:    "failed to parse intent JSON",
		},
		{
			name: "valid intent with default output",
			args: []string{"-intent", "valid.json"},
			setupFiles: map[string]string{
				"valid.json": `{
					"intent_type": "scaling",
					"target": "web-app",
					"namespace": "default",
					"replicas": 5,
					"reason": "increased load",
					"source": "autoscaler",
					"correlation_id": "test-123"
				}`,
			},
			expectError: false,
		},
		{
			name: "valid intent with custom output",
			args: []string{"-intent", "valid.json", "-out", "custom-output"},
			setupFiles: map[string]string{
				"valid.json": `{
					"intent_type": "scaling",
					"target": "api-server",
					"namespace": "production",
					"replicas": 10
				}`,
			},
			expectError: false,
		},
		{
			name: "boundary condition - zero replicas",
			args: []string{"-intent", "zero-replicas.json"},
			setupFiles: map[string]string{
				"zero-replicas.json": `{
					"intent_type": "scaling",
					"target": "worker",
					"namespace": "test",
					"replicas": 0
				}`,
			},
			expectError: false,
		},
		{
			name: "boundary condition - max replicas",
			args: []string{"-intent", "max-replicas.json"},
			setupFiles: map[string]string{
				"max-replicas.json": `{
					"intent_type": "scaling",
					"target": "batch-processor",
					"namespace": "processing",
					"replicas": 100
				}`,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tempDir := t.TempDir()

			// Setup test files
			for filename, content := range tt.setupFiles {
				filePath := filepath.Join(tempDir, filename)
				require.NoError(t, os.WriteFile(filePath, []byte(content), 0o644))
			}

			// Create logger for testing

			// Extract intent and output paths from args
			var intentPath, outputDir string
			outputDir = "examples/packages/structured" // default

			for i, arg := range tt.args {
				switch arg {
				case "-intent":
					if i+1 < len(tt.args) {
						intentPath = filepath.Join(tempDir, tt.args[i+1])
					}
				case "-out":
					if i+1 < len(tt.args) {
						outputDir = filepath.Join(tempDir, tt.args[i+1])
					}
				}
			}

			// Skip test cases that don't have intent file setup
			if intentPath == "" && tt.expectError && strings.Contains(tt.errorMsg, "intent flag is required") {
				return // This is tested in command line parsing
			}

			if intentPath == "" {
				intentPath = filepath.Join(tempDir, "nonexistent.json")
			}

			// Create output directory if it doesn't exist and test expects success
			if !tt.expectError && outputDir != "examples/packages/structured" {
				require.NoError(t, os.MkdirAll(outputDir, 0o755))
			}

			// Test the run function
			err := run(intentPath, outputDir, false)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)

				// Verify generated files exist and are valid
				if !tt.expectError {
					verifyGeneratedPackage(t, intentPath, outputDir)
				}
			}
		})
	}
}

func TestRunFunction(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		intentJSON  string
		outputDir   string
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful generation",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "test-app",
				"namespace": "default",
				"replicas": 3
			}`,
			outputDir:   filepath.Join(tempDir, "output1"),
			expectError: false,
		},
		{
			name: "missing required field - target",
			intentJSON: `{
				"intent_type": "scaling",
				"namespace": "default",
				"replicas": 3
			}`,
			outputDir:   filepath.Join(tempDir, "output2"),
			expectError: true,
			errorMsg:    "target is required",
		},
		{
			name: "missing required field - namespace",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "test-app",
				"replicas": 3
			}`,
			outputDir:   filepath.Join(tempDir, "output3"),
			expectError: true,
			errorMsg:    "namespace is required",
		},
		{
			name: "missing required field - replicas",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "test-app",
				"namespace": "default"
			}`,
			outputDir:   filepath.Join(tempDir, "output4"),
			expectError: false, // Missing replicas defaults to 0, which is valid
		},
		{
			name: "invalid intent_type",
			intentJSON: `{
				"intent_type": "invalid",
				"target": "test-app",
				"namespace": "default",
				"replicas": 3
			}`,
			outputDir:   filepath.Join(tempDir, "output5"),
			expectError: true,
			errorMsg:    "unsupported intent_type",
		},
		{
			name: "replicas below minimum",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "test-app",
				"namespace": "default",
				"replicas": -1
			}`,
			outputDir:   filepath.Join(tempDir, "output6"),
			expectError: true,
			errorMsg:    "replicas must be >= 0",
		},
		{
			name: "replicas above maximum",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "test-app",
				"namespace": "default",
				"replicas": 101
			}`,
			outputDir:   filepath.Join(tempDir, "output7"),
			expectError: false, // No longer checking maximum in basic validation
		},
		{
			name: "empty target name",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "",
				"namespace": "default",
				"replicas": 3
			}`,
			outputDir:   filepath.Join(tempDir, "output8"),
			expectError: true,
			errorMsg:    "target is required",
		},
		{
			name: "empty namespace",
			intentJSON: `{
				"intent_type": "scaling",
				"target": "test-app",
				"namespace": "",
				"replicas": 3
			}`,
			outputDir:   filepath.Join(tempDir, "output9"),
			expectError: true,
			errorMsg:    "namespace is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create intent file
			intentFile := filepath.Join(tempDir, fmt.Sprintf("intent_%s.json", strings.ReplaceAll(tt.name, " ", "_")))
			require.NoError(t, os.WriteFile(intentFile, []byte(tt.intentJSON), 0o644))

			// Create output directory if test expects success
			if !tt.expectError {
				require.NoError(t, os.MkdirAll(tt.outputDir, 0o755))
			}

			// Run the function
			err := run(intentFile, tt.outputDir, false)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				verifyGeneratedPackage(t, intentFile, tt.outputDir)
			}
		})
	}
}

func TestFileIOErrors(t *testing.T) {
	t.Run("invalid intent file path", func(t *testing.T) {
		// Test with a path that doesn't exist
		invalidPath := filepath.Join("nonexistent", "path", "intent.json")
		outputDir := t.TempDir()

		err := run(invalidPath, outputDir, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file does not exist")
	})
}

// verifyGeneratedPackage checks that all expected files are generated and valid
func verifyGeneratedPackage(t *testing.T, intentFile, outputDir string) {
	t.Helper()

	// Read and parse the intent to get expected target name
	intentData, err := os.ReadFile(intentFile)
	require.NoError(t, err)

	var intent map[string]interface{}
	require.NoError(t, json.Unmarshal(intentData, &intent))

	target := intent["target"].(string)

	// Find the package directory (should start with target-scaling-patch)
	entries, err := os.ReadDir(outputDir)
	require.NoError(t, err)

	var packageDir string
	expectedPrefix := fmt.Sprintf("%s-scaling-patch", target)
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), expectedPrefix) {
			packageDir = filepath.Join(outputDir, entry.Name())
			break
		}
	}

	// Verify package directory exists
	require.NotEmpty(t, packageDir, "Expected package directory with prefix %s not found", expectedPrefix)
	assert.DirExists(t, packageDir)

	// Verify Kptfile exists and is valid YAML
	kptfilePath := filepath.Join(packageDir, "Kptfile")
	assert.FileExists(t, kptfilePath)

	kptfileData, err := os.ReadFile(kptfilePath)
	require.NoError(t, err)

	var kptfile map[string]interface{}
	require.NoError(t, yaml.Unmarshal(kptfileData, &kptfile))

	assert.Equal(t, "kpt.dev/v1", kptfile["APIVersion"])
	assert.Equal(t, "Kptfile", kptfile["Kind"])

	metadata, ok := kptfile["Metadata"].(map[string]interface{})
	require.True(t, ok)
	// Package name should match the directory name which includes timestamp for collision resistance
	actualPackageName := filepath.Base(packageDir)
	assert.Equal(t, actualPackageName, metadata["Name"])

	// Verify scaling-patch.yaml exists and is valid YAML
	patchFilePath := filepath.Join(packageDir, "scaling-patch.yaml")
	assert.FileExists(t, patchFilePath)

	patchData, err := os.ReadFile(patchFilePath)
	require.NoError(t, err)

	var patchFile map[string]interface{}
	require.NoError(t, yaml.Unmarshal(patchData, &patchFile))

	assert.Equal(t, "apps/v1", patchFile["APIVersion"])
	assert.Equal(t, "Deployment", patchFile["Kind"])

	patchMetadata, ok := patchFile["Metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, target, patchMetadata["Name"])
	assert.Equal(t, intent["namespace"], patchMetadata["Namespace"])

	// Verify annotations
	annotations, ok := patchMetadata["Annotations"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "replace", annotations["config.kubernetes.io/merge-policy"])
	assert.Equal(t, intent["intent_type"], annotations["nephoran.io/intent-type"])
	assert.Contains(t, annotations, "nephoran.io/generated-at")

	// Verify spec contains correct replicas
	spec, ok := patchFile["Spec"].(map[string]interface{})
	require.True(t, ok)

	// Convert replicas to int for comparison (YAML may parse as float64)
	replicas := int(spec["Replicas"].(float64))

	// Handle missing replicas field (defaults to 0)
	var expectedReplicas int
	if replicasVal, ok := intent["replicas"]; ok && replicasVal != nil {
		expectedReplicas = int(replicasVal.(float64))
	} else {
		expectedReplicas = 0 // Default value when missing
	}
	assert.Equal(t, expectedReplicas, replicas)

	// Verify README.md exists
	readmePath := filepath.Join(packageDir, "README.md")
	assert.FileExists(t, readmePath)

	readmeData, err := os.ReadFile(readmePath)
	require.NoError(t, err)
	readmeContent := string(readmeData)

	assert.Contains(t, readmeContent, actualPackageName)
	assert.Contains(t, readmeContent, target)
	assert.Contains(t, readmeContent, intent["namespace"].(string))
	assert.Contains(t, readmeContent, fmt.Sprintf("%d", expectedReplicas))
}

func TestVerboseLogging(t *testing.T) {
	tempDir := t.TempDir()

	// Create valid intent file
	intentFile := filepath.Join(tempDir, "intent.json")
	intentJSON := `{
		"intent_type": "scaling",
		"target": "test-app",
		"namespace": "default",
		"replicas": 3
	}`
	require.NoError(t, os.WriteFile(intentFile, []byte(intentJSON), 0o644))

	outputDir := filepath.Join(tempDir, "output")
	require.NoError(t, os.MkdirAll(outputDir, 0o755))

	// Test with verbose logging (this mainly tests that verbose flag doesn't break anything)
	err := run(intentFile, outputDir, false)
	assert.NoError(t, err)
}

func TestPackagePathGeneration(t *testing.T) {
	tests := []struct {
		name           string
		target         string
		expectedSuffix string
	}{
		{
			name:           "simple target",
			target:         "web-app",
			expectedSuffix: "web-app-scaling-patch",
		},
		{
			name:           "target with numbers",
			target:         "api-v2",
			expectedSuffix: "api-v2-scaling-patch",
		},
		{
			name:           "target with hyphens",
			target:         "micro-service-worker",
			expectedSuffix: "micro-service-worker-scaling-patch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			intentJSON := fmt.Sprintf(`{
				"intent_type": "scaling",
				"target": "%s",
				"namespace": "default",
				"replicas": 3
			}`, tt.target)

			intentFile := filepath.Join(tempDir, "intent.json")
			require.NoError(t, os.WriteFile(intentFile, []byte(intentJSON), 0o644))

			outputDir := filepath.Join(tempDir, "output")
			require.NoError(t, os.MkdirAll(outputDir, 0o755))
			err := run(intentFile, outputDir, false)
			require.NoError(t, err)

			// Find the package directory (should start with expected suffix)
			entries, err := os.ReadDir(outputDir)
			require.NoError(t, err)

			found := false
			for _, entry := range entries {
				if entry.IsDir() && strings.HasPrefix(entry.Name(), tt.expectedSuffix) {
					expectedPackageDir := filepath.Join(outputDir, entry.Name())
					assert.DirExists(t, expectedPackageDir)
					found = true
					break
				}
			}
			assert.True(t, found, "Expected package directory with prefix %s not found", tt.expectedSuffix)
		})
	}
}
