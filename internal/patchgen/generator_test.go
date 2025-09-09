package patchgen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

func TestNewPatchPackage(t *testing.T) {
	intent := &Intent{
		IntentType:    "scaling",
		Target:        "test-app",
		Namespace:     "default",
		Replicas:      5,
		Reason:        "Load increase",
		Source:        "autoscaler",
		CorrelationID: "test-123",
	}

	outputDir := "/tmp/test-output"

	patchPackage := NewPatchPackage(intent, outputDir)

	assert.NotNil(t, patchPackage)
	assert.Equal(t, intent, patchPackage.Intent)
	assert.Equal(t, outputDir, patchPackage.OutputDir)

	// Verify Kptfile structure
	assert.NotNil(t, patchPackage.Kptfile)
	assert.Equal(t, "kpt.dev/v1", patchPackage.Kptfile.APIVersion)
	assert.Equal(t, "Kptfile", patchPackage.Kptfile.Kind)
	// Package names now have dynamic timestamps, verify the prefix
	assert.True(t, strings.HasPrefix(patchPackage.Kptfile.Metadata.Name, "test-app-scaling-patch"), 
		"Expected package name to start with 'test-app-scaling-patch', got %s", patchPackage.Kptfile.Metadata.Name)
	assert.Contains(t, patchPackage.Kptfile.Info.Description, "test-app")
	assert.Contains(t, patchPackage.Kptfile.Info.Description, "5 replicas")

	// Verify pipeline configuration
	assert.Len(t, patchPackage.Kptfile.Pipeline.Mutators, 1)
	mutator := patchPackage.Kptfile.Pipeline.Mutators[0]
	assert.Equal(t, "gcr.io/kpt-fn/apply-replacements:v0.1.1", mutator.Image)
	assert.Equal(t, "true", mutator.ConfigMap["apply-replacements"])

	// Verify PatchFile structure
	assert.NotNil(t, patchPackage.PatchFile)
	assert.Equal(t, "apps/v1", patchPackage.PatchFile.APIVersion)
	assert.Equal(t, "Deployment", patchPackage.PatchFile.Kind)
	assert.Equal(t, "test-app", patchPackage.PatchFile.Metadata.Name)
	assert.Equal(t, "default", patchPackage.PatchFile.Metadata.Namespace)
	assert.Equal(t, 5, patchPackage.PatchFile.Spec.Replicas)

	// Verify annotations
	annotations := patchPackage.PatchFile.Metadata.Annotations
	assert.Equal(t, "replace", annotations["config.kubernetes.io/merge-policy"])
	assert.Equal(t, "scaling", annotations["nephoran.io/intent-type"])
	assert.Contains(t, annotations, "nephoran.io/generated-at")
}

func TestPatchPackageGenerate(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		intent    *Intent
		outputDir string
	}{
		{
			name: "basic generation",
			intent: &Intent{
				IntentType: "scaling",
				Target:     "web-app",
				Namespace:  "default",
				Replicas:   3,
			},
			outputDir: tempDir,
		},
		{
			name: "generation with all optional fields",
			intent: &Intent{
				IntentType:    "scaling",
				Target:        "api-server",
				Namespace:     "production",
				Replicas:      10,
				Reason:        "High CPU usage",
				Source:        "prometheus",
				CorrelationID: "scale-event-456",
			},
			outputDir: tempDir,
		},
		{
			name: "zero replicas",
			intent: &Intent{
				IntentType: "scaling",
				Target:     "worker",
				Namespace:  "test",
				Replicas:   0,
			},
			outputDir: tempDir,
		},
		{
			name: "max replicas",
			intent: &Intent{
				IntentType: "scaling",
				Target:     "batch-processor",
				Namespace:  "processing",
				Replicas:   100,
			},
			outputDir: tempDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputSubDir := filepath.Join(tt.outputDir, tt.name)
			patchPackage := NewPatchPackage(tt.intent, outputSubDir)

			err := patchPackage.Generate()
			assert.NoError(t, err)

			// Verify package structure
			// Package names now have dynamic timestamps, so we need to find the directory
			entries, err := os.ReadDir(outputSubDir)
			assert.NoError(t, err)
			
			var packageDir string
			for _, entry := range entries {
				if entry.IsDir() && strings.HasPrefix(entry.Name(), fmt.Sprintf("%s-scaling-patch", tt.intent.Target)) {
					packageDir = filepath.Join(outputSubDir, entry.Name())
					break
				}
			}
			assert.NotEmpty(t, packageDir, "Package directory not found")
			assert.DirExists(t, packageDir)

			// Verify generated files
			verifyKptfile(t, packageDir, tt.intent)
			verifyPatchFile(t, packageDir, tt.intent)
			verifyReadme(t, packageDir, tt.intent)
		})
	}
}

func TestGenerateKptfile(t *testing.T) {
	tempDir := t.TempDir()

	intent := &Intent{
		IntentType: "scaling",
		Target:     "test-service",
		Namespace:  "staging",
		Replicas:   7,
	}

	patchPackage := NewPatchPackage(intent, tempDir)
	// Use the actual package name from the generated package
	packageDir := filepath.Join(tempDir, patchPackage.Kptfile.Metadata.Name)
	require.NoError(t, os.MkdirAll(packageDir, 0o755))

	err := patchPackage.generateKptfile(packageDir)
	assert.NoError(t, err)

	// Verify file exists
	kptfilePath := filepath.Join(packageDir, "Kptfile")
	assert.FileExists(t, kptfilePath)

	// Verify file content
	data, err := os.ReadFile(kptfilePath)
	require.NoError(t, err)

	var kptfile Kptfile
	err = yaml.Unmarshal(data, &kptfile)
	require.NoError(t, err)

	assert.Equal(t, "kpt.dev/v1", kptfile.APIVersion)
	assert.Equal(t, "Kptfile", kptfile.Kind)
	// Package names now have dynamic timestamps, verify the prefix
	assert.True(t, strings.HasPrefix(kptfile.Metadata.Name, "test-service-scaling-patch"),
		"Expected package name to start with 'test-service-scaling-patch', got %s", kptfile.Metadata.Name)
	assert.Contains(t, kptfile.Info.Description, "test-service")
	assert.Contains(t, kptfile.Info.Description, "7 replicas")
}

func TestGeneratePatchFile(t *testing.T) {
	tempDir := t.TempDir()

	intent := &Intent{
		IntentType:    "scaling",
		Target:        "backend-api",
		Namespace:     "production",
		Replicas:      15,
		Reason:        "Peak traffic",
		Source:        "load-balancer",
		CorrelationID: "lb-scale-789",
	}

	patchPackage := NewPatchPackage(intent, tempDir)
	// Use the actual package name from the generated package
	packageDir := filepath.Join(tempDir, patchPackage.Kptfile.Metadata.Name)
	require.NoError(t, os.MkdirAll(packageDir, 0o755))

	err := patchPackage.generatePatchFile(packageDir)
	assert.NoError(t, err)

	// Verify file exists
	patchPath := filepath.Join(packageDir, "scaling-patch.yaml")
	assert.FileExists(t, patchPath)

	// Verify file content
	data, err := os.ReadFile(patchPath)
	require.NoError(t, err)

	var patchFile PatchFile
	err = yaml.Unmarshal(data, &patchFile)
	require.NoError(t, err)

	assert.Equal(t, "apps/v1", patchFile.APIVersion)
	assert.Equal(t, "Deployment", patchFile.Kind)
	assert.Equal(t, "backend-api", patchFile.Metadata.Name)
	assert.Equal(t, "production", patchFile.Metadata.Namespace)
	assert.Equal(t, 15, patchFile.Spec.Replicas)

	// Verify annotations
	annotations := patchFile.Metadata.Annotations
	assert.Equal(t, "replace", annotations["config.kubernetes.io/merge-policy"])
	assert.Equal(t, "scaling", annotations["nephoran.io/intent-type"])
	assert.Contains(t, annotations, "nephoran.io/generated-at")

	// Verify timestamp format
	generatedAt := annotations["nephoran.io/generated-at"]
	_, err = time.Parse(time.RFC3339, generatedAt)
	assert.NoError(t, err, "Generated timestamp should be in RFC3339 format")
}

func TestGenerateReadme(t *testing.T) {
	tempDir := t.TempDir()

	intent := &Intent{
		IntentType:    "scaling",
		Target:        "frontend",
		Namespace:     "web",
		Replicas:      8,
		Reason:        "Marketing campaign",
		Source:        "manual",
		CorrelationID: "campaign-scale-101",
	}

	patchPackage := NewPatchPackage(intent, tempDir)
	// Use the actual package name from the generated package
	packageDir := filepath.Join(tempDir, patchPackage.Kptfile.Metadata.Name)
	require.NoError(t, os.MkdirAll(packageDir, 0o755))

	err := patchPackage.generateReadme(packageDir)
	assert.NoError(t, err)

	// Verify file exists
	readmePath := filepath.Join(packageDir, "README.md")
	assert.FileExists(t, readmePath)

	// Verify file content
	data, err := os.ReadFile(readmePath)
	require.NoError(t, err)
	content := string(data)

	// Check required content - should contain the target name
	assert.Contains(t, content, "frontend")
	assert.Contains(t, content, "scaling-patch")
	assert.Contains(t, content, "frontend")
	assert.Contains(t, content, "web")
	assert.Contains(t, content, "8")
	assert.Contains(t, content, "scaling")
	assert.Contains(t, content, "Kptfile")
	assert.Contains(t, content, "scaling-patch.yaml")
	assert.Contains(t, content, "kpt fn eval")
	assert.Contains(t, content, "Generated at:")
}

func TestGetPackagePath(t *testing.T) {
	tests := []struct {
		name               string
		target             string
		outputDir          string
		expectedPathSuffix string
	}{
		{
			name:               "simple path",
			target:             "web-app",
			outputDir:          filepath.Join("tmp", "output"),
			expectedPathSuffix: "web-app-scaling-patch",
		},
		{
			name:               "complex target name",
			target:             "api-gateway-v2",
			outputDir:          filepath.Join("var", "packages"),
			expectedPathSuffix: "api-gateway-v2-scaling-patch",
		},
		{
			name:               "nested output directory",
			target:             "worker",
			outputDir:          filepath.Join("home", "user", "kpt", "packages"),
			expectedPathSuffix: "worker-scaling-patch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent := &Intent{
				IntentType: "scaling",
				Target:     tt.target,
				Namespace:  "default",
				Replicas:   1,
			}

			patchPackage := NewPatchPackage(intent, tt.outputDir)
			actualPath := patchPackage.GetPackagePath()
			// Package names now have dynamic timestamps, verify the path contains the expected prefix
			assert.True(t, strings.HasPrefix(filepath.Base(actualPath), tt.expectedPathSuffix),
				"Expected package path to contain prefix '%s', got '%s'", tt.expectedPathSuffix, actualPath)
			assert.Equal(t, tt.outputDir, filepath.Dir(actualPath))
		})
	}
}

func TestFileIOErrorHandling(t *testing.T) {
	intent := &Intent{
		IntentType: "scaling",
		Target:     "test-app",
		Namespace:  "default",
		Replicas:   3,
	}

	t.Run("invalid output directory", func(t *testing.T) {
		// Try to create package in a file (not directory)
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "notadir")
		require.NoError(t, os.WriteFile(filePath, []byte("test"), 0o644))

		patchPackage := NewPatchPackage(intent, filePath)
		err := patchPackage.Generate()
		assert.Error(t, err)
	})
}

func TestHelperFunctions(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("writeFile creates directories", func(t *testing.T) {
		nestedPath := filepath.Join(tempDir, "nested", "deep", "file.txt")
		content := []byte("test content")

		err := writeFile(nestedPath, content)
		assert.NoError(t, err)
		assert.FileExists(t, nestedPath)

		data, err := os.ReadFile(nestedPath)
		require.NoError(t, err)
		assert.Equal(t, content, data)
	})

	t.Run("os.ReadFile reads content", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "read-test.txt")
		expectedContent := []byte("read this content")
		require.NoError(t, os.WriteFile(filePath, expectedContent, 0o644))

		content, err := os.ReadFile(filePath)
		assert.NoError(t, err)
		assert.Equal(t, expectedContent, content)
	})

	t.Run("os.ReadFile handles non-existent file", func(t *testing.T) {
		nonExistentPath := filepath.Join(tempDir, "does-not-exist.txt")

		content, err := os.ReadFile(nonExistentPath)
		assert.Error(t, err)
		assert.Nil(t, content)
	})
}

func TestPackageNameGeneration(t *testing.T) {
	tests := []struct {
		name                string
		target              string
		expectedPackageName string
	}{
		{
			name:                "simple target",
			target:              "webapp",
			expectedPackageName: "webapp-scaling-patch",
		},
		{
			name:                "hyphenated target",
			target:              "web-app",
			expectedPackageName: "web-app-scaling-patch",
		},
		{
			name:                "target with numbers",
			target:              "api-v2",
			expectedPackageName: "api-v2-scaling-patch",
		},
		{
			name:                "complex target name",
			target:              "micro-service-worker-v1",
			expectedPackageName: "micro-service-worker-v1-scaling-patch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent := &Intent{
				IntentType: "scaling",
				Target:     tt.target,
				Namespace:  "default",
				Replicas:   1,
			}

			patchPackage := NewPatchPackage(intent, "/tmp")
			// Package names now have dynamic timestamps, verify the prefix
			assert.True(t, strings.HasPrefix(patchPackage.Kptfile.Metadata.Name, tt.expectedPackageName),
				"Expected package name to start with '%s', got '%s'", tt.expectedPackageName, patchPackage.Kptfile.Metadata.Name)
		})
	}
}

func TestYAMLMarshaling(t *testing.T) {
	intent := &Intent{
		IntentType: "scaling",
		Target:     "test-marshal",
		Namespace:  "default",
		Replicas:   5,
	}

	patchPackage := NewPatchPackage(intent, "/tmp")

	t.Run("Kptfile marshals to valid YAML", func(t *testing.T) {
		data, err := yaml.Marshal(patchPackage.Kptfile)
		assert.NoError(t, err)

		// Verify it can be unmarshaled back
		var unmarshaled Kptfile
		err = yaml.Unmarshal(data, &unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, *patchPackage.Kptfile, unmarshaled)
	})

	t.Run("PatchFile marshals to valid YAML", func(t *testing.T) {
		data, err := yaml.Marshal(patchPackage.PatchFile)
		assert.NoError(t, err)

		// Verify it can be unmarshaled back
		var unmarshaled PatchFile
		err = yaml.Unmarshal(data, &unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, *patchPackage.PatchFile, unmarshaled)
	})
}

func TestConcurrentGeneration(t *testing.T) {
	tempDir := t.TempDir()
	const numGoroutines = 5

	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			intent := &Intent{
				IntentType: "scaling",
				Target:     fmt.Sprintf("app-%d", id),
				Namespace:  "default",
				Replicas:   id + 1,
			}

			outputDir := filepath.Join(tempDir, fmt.Sprintf("output-%d", id))
			patchPackage := NewPatchPackage(intent, outputDir)
			results <- patchPackage.Generate()
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		err := <-results
		assert.NoError(t, err, "Concurrent generation should succeed")
	}

	// Verify all packages were created
	for i := 0; i < numGoroutines; i++ {
		outputDir := filepath.Join(tempDir, fmt.Sprintf("output-%d", i))
		entries, err := os.ReadDir(outputDir)
		assert.NoError(t, err)
		
		// Find the package directory with the expected prefix
		expectedPrefix := fmt.Sprintf("app-%d-scaling-patch", i)
		found := false
		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(entry.Name(), expectedPrefix) {
				packageDir := filepath.Join(outputDir, entry.Name())
				assert.DirExists(t, packageDir)
				found = true
				break
			}
		}
		assert.True(t, found, "Package directory with prefix '%s' not found in %s", expectedPrefix, outputDir)
	}
}

func TestTimestampGeneration(t *testing.T) {
	intent := &Intent{
		IntentType: "scaling",
		Target:     "timestamp-test",
		Namespace:  "default",
		Replicas:   1,
	}

	// Generate two packages with a small delay
	patchPackage1 := NewPatchPackage(intent, "/tmp")
	timestamp1 := patchPackage1.PatchFile.Metadata.Annotations["nephoran.io/generated-at"]

	time.Sleep(10 * time.Millisecond) // Increase sleep time for Windows timing resolution

	patchPackage2 := NewPatchPackage(intent, "/tmp")
	timestamp2 := patchPackage2.PatchFile.Metadata.Annotations["nephoran.io/generated-at"]

	// Both should be valid RFC3339 timestamps
	_, err1 := time.Parse(time.RFC3339, timestamp1)
	_, err2 := time.Parse(time.RFC3339, timestamp2)
	assert.NoError(t, err1)
	assert.NoError(t, err2)

	// Parse timestamps to compare
	time1, _ := time.Parse(time.RFC3339, timestamp1)
	time2, _ := time.Parse(time.RFC3339, timestamp2)

	// time2 should be after time1
	assert.True(t, time2.After(time1) || time2.Equal(time1), "Second timestamp should be after or equal to first timestamp")
}

// Helper functions for test verification

func verifyKptfile(t *testing.T, packageDir string, intent *Intent) {
	t.Helper()

	kptfilePath := filepath.Join(packageDir, "Kptfile")
	assert.FileExists(t, kptfilePath)

	data, err := os.ReadFile(kptfilePath)
	require.NoError(t, err)

	var kptfile map[string]interface{}
	err = yaml.Unmarshal(data, &kptfile)
	require.NoError(t, err)

	assert.Equal(t, "kpt.dev/v1", kptfile["APIVersion"])
	assert.Equal(t, "Kptfile", kptfile["Kind"])

	metadata, ok := kptfile["Metadata"].(map[string]interface{})
	require.True(t, ok)
	expectedPrefix := fmt.Sprintf("%s-scaling-patch", intent.Target)
	name, ok := metadata["Name"].(string)
	require.True(t, ok, "Name field should be a string")
	assert.True(t, strings.HasPrefix(name, expectedPrefix), 
		"Expected name to start with '%s', got '%s'", expectedPrefix, name)

	info, ok := kptfile["Info"].(map[string]interface{})
	require.True(t, ok)
	description := info["Description"].(string)
	assert.Contains(t, description, intent.Target)
	assert.Contains(t, description, fmt.Sprintf("%d replicas", intent.Replicas))
}

func verifyPatchFile(t *testing.T, packageDir string, intent *Intent) {
	t.Helper()

	patchPath := filepath.Join(packageDir, "scaling-patch.yaml")
	assert.FileExists(t, patchPath)

	data, err := os.ReadFile(patchPath)
	require.NoError(t, err)

	var patchFile map[string]interface{}
	err = yaml.Unmarshal(data, &patchFile)
	require.NoError(t, err)

	assert.Equal(t, "apps/v1", patchFile["APIVersion"])
	assert.Equal(t, "Deployment", patchFile["Kind"])

	metadata, ok := patchFile["Metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, intent.Target, metadata["Name"])
	assert.Equal(t, intent.Namespace, metadata["Namespace"])

	annotations, ok := metadata["Annotations"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "replace", annotations["config.kubernetes.io/merge-policy"])
	assert.Equal(t, intent.IntentType, annotations["nephoran.io/intent-type"])
	assert.Contains(t, annotations, "nephoran.io/generated-at")

	spec, ok := patchFile["Spec"].(map[string]interface{})
	require.True(t, ok)
	replicas := int(spec["Replicas"].(float64))
	assert.Equal(t, intent.Replicas, replicas)
}

func verifyReadme(t *testing.T, packageDir string, intent *Intent) {
	t.Helper()

	readmePath := filepath.Join(packageDir, "README.md")
	assert.FileExists(t, readmePath)

	data, err := os.ReadFile(readmePath)
	require.NoError(t, err)
	content := string(data)

	// Verify essential content is present
	// The README should contain the target name, even if the full package name has timestamps
	assert.Contains(t, content, intent.Target)
	assert.Contains(t, content, "scaling-patch")
	assert.Contains(t, content, intent.Target)
	assert.Contains(t, content, intent.Namespace)
	assert.Contains(t, content, fmt.Sprintf("%d", intent.Replicas))
	assert.Contains(t, content, intent.IntentType)

	// Verify structure - the heading should contain the target at least
	assert.Contains(t, content, "# ")
	assert.Contains(t, content, intent.Target)
	assert.Contains(t, content, "## Intent Details")
	assert.Contains(t, content, "## Files")
	assert.Contains(t, content, "## Usage")
	assert.Contains(t, content, "## Generated")

	// Verify it mentions the files
	assert.Contains(t, content, "Kptfile")
	assert.Contains(t, content, "scaling-patch.yaml")
}
