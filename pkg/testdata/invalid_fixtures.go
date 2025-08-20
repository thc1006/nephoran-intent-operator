package testdata

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// InvalidFixtureFactory provides methods to create invalid test intent data
type InvalidFixtureFactory struct {
	BaseDir string
	Counter int
}

// NewInvalidFixtureFactory creates a new invalid fixture factory
func NewInvalidFixtureFactory(baseDir string) *InvalidFixtureFactory {
	return &InvalidFixtureFactory{
		BaseDir: baseDir,
	}
}

// CreateIntentWithStringTarget creates an intent with invalid string target (legacy format)
func (f *InvalidFixtureFactory) CreateIntentWithStringTarget(name string) map[string]interface{} {
	f.Counter++
	return map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "NetworkIntent",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-string-target-%d", name, f.Counter),
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"action": "scale",
			"target": fmt.Sprintf("deployment/%s", name), // Invalid: string instead of object
		},
	}
}

// CreateIntentWithMissingTargetType creates an intent with target missing type field
func (f *InvalidFixtureFactory) CreateIntentWithMissingTargetType(name string) map[string]interface{} {
	f.Counter++
	return map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "NetworkIntent",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-missing-type-%d", name, f.Counter),
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"action": "scale",
			"target": map[string]interface{}{
				"name": name, // Missing "type" field
			},
		},
	}
}

// CreateIntentWithMissingTargetName creates an intent with target missing name field
func (f *InvalidFixtureFactory) CreateIntentWithMissingTargetName(name string) map[string]interface{} {
	f.Counter++
	return map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "NetworkIntent",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-missing-name-%d", name, f.Counter),
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"action": "scale",
			"target": map[string]interface{}{
				"type": "deployment", // Missing "name" field
			},
		},
	}
}

// CreateIntentWithEmptyTarget creates an intent with empty target object
func (f *InvalidFixtureFactory) CreateIntentWithEmptyTarget(name string) map[string]interface{} {
	f.Counter++
	return map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "NetworkIntent",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-empty-target-%d", name, f.Counter),
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"action": "scale",
			"target": map[string]interface{}{}, // Empty target object
		},
	}
}

// CreateIntentWithInvalidTargetType creates an intent with unsupported target type
func (f *InvalidFixtureFactory) CreateIntentWithInvalidTargetType(name string) map[string]interface{} {
	f.Counter++
	return map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "NetworkIntent",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-invalid-type-%d", name, f.Counter),
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"action": "scale",
			"target": map[string]interface{}{
				"type": "unsupported-resource", // Invalid resource type
				"name": name,
			},
		},
	}
}

// CreateIntentWithEmptyTargetType creates an intent with empty target type
func (f *InvalidFixtureFactory) CreateIntentWithEmptyTargetType(name string) map[string]interface{} {
	f.Counter++
	return map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "NetworkIntent",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-empty-type-%d", name, f.Counter),
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"action": "scale",
			"target": map[string]interface{}{
				"type": "", // Empty string type
				"name": name,
			},
		},
	}
}

// CreateIntentWithEmptyTargetName creates an intent with empty target name
func (f *InvalidFixtureFactory) CreateIntentWithEmptyTargetName(name string) map[string]interface{} {
	f.Counter++
	return map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "NetworkIntent",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-empty-name-%d", name, f.Counter),
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"action": "scale",
			"target": map[string]interface{}{
				"type": "deployment",
				"name": "", // Empty string name
			},
		},
	}
}

// CreateIntentWithNullTarget creates an intent with null target
func (f *InvalidFixtureFactory) CreateIntentWithNullTarget(name string) map[string]interface{} {
	f.Counter++
	return map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "NetworkIntent",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-null-target-%d", name, f.Counter),
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"action": "scale",
			"target": nil, // Null target
		},
	}
}

// CreateIntentWithNonStringTargetType creates an intent with non-string target type
func (f *InvalidFixtureFactory) CreateIntentWithNonStringTargetType(name string) map[string]interface{} {
	f.Counter++
	return map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "NetworkIntent",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-non-string-type-%d", name, f.Counter),
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"action": "scale",
			"target": map[string]interface{}{
				"type": 123, // Non-string type
				"name": name,
			},
		},
	}
}

// CreateIntentWithNonStringTargetName creates an intent with non-string target name
func (f *InvalidFixtureFactory) CreateIntentWithNonStringTargetName(name string) map[string]interface{} {
	f.Counter++
	return map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "NetworkIntent",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-non-string-name-%d", name, f.Counter),
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"action": "scale",
			"target": map[string]interface{}{
				"type": "deployment",
				"name": []string{name}, // Non-string name (array)
			},
		},
	}
}

// CreateIntentWithArrayTarget creates an intent with array target (invalid)
func (f *InvalidFixtureFactory) CreateIntentWithArrayTarget(name string) map[string]interface{} {
	f.Counter++
	return map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "NetworkIntent",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-array-target-%d", name, f.Counter),
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"action": "scale",
			"target": []string{"deployment", name}, // Array instead of object
		},
	}
}

// CreateIntentWithNestedTarget creates an intent with overly nested target
func (f *InvalidFixtureFactory) CreateIntentWithNestedTarget(name string) map[string]interface{} {
	f.Counter++
	return map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "NetworkIntent",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-nested-target-%d", name, f.Counter),
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"action": "scale",
			"target": map[string]interface{}{
				"type": "deployment",
				"name": name,
				"nested": map[string]interface{}{
					"deeper": map[string]interface{}{
						"invalid": "structure",
					},
				},
			},
		},
	}
}

// CreateIntentFileWithInvalidTarget creates a file with invalid target content
func (f *InvalidFixtureFactory) CreateIntentFileWithInvalidTarget(filename string, invalidType string) (string, error) {
	var intent map[string]interface{}
	
	switch invalidType {
	case "string_target":
		intent = f.CreateIntentWithStringTarget("test")
	case "missing_type":
		intent = f.CreateIntentWithMissingTargetType("test")
	case "missing_name":
		intent = f.CreateIntentWithMissingTargetName("test")
	case "empty_target":
		intent = f.CreateIntentWithEmptyTarget("test")
	case "invalid_type":
		intent = f.CreateIntentWithInvalidTargetType("test")
	case "empty_type":
		intent = f.CreateIntentWithEmptyTargetType("test")
	case "empty_name":
		intent = f.CreateIntentWithEmptyTargetName("test")
	case "null_target":
		intent = f.CreateIntentWithNullTarget("test")
	case "non_string_type":
		intent = f.CreateIntentWithNonStringTargetType("test")
	case "non_string_name":
		intent = f.CreateIntentWithNonStringTargetName("test")
	case "array_target":
		intent = f.CreateIntentWithArrayTarget("test")
	case "nested_target":
		intent = f.CreateIntentWithNestedTarget("test")
	default:
		intent = f.CreateIntentWithStringTarget("test")
	}
	
	data, err := json.MarshalIndent(intent, "", "  ")
	if err != nil {
		return "", err
	}
	
	filePath := fmt.Sprintf("%s/%s", f.BaseDir, filename)
	if err := createFileWithContent(filePath, data); err != nil {
		return "", err
	}
	
	return filePath, nil
}

// CreateBatchInvalidIntents creates multiple invalid intent files for batch testing
func (f *InvalidFixtureFactory) CreateBatchInvalidIntents(baseName string, invalidTypes []string) ([]string, error) {
	var filePaths []string
	
	for i, invalidType := range invalidTypes {
		filename := fmt.Sprintf("%s-invalid-%s-%d.json", baseName, invalidType, i)
		filePath, err := f.CreateIntentFileWithInvalidTarget(filename, invalidType)
		if err != nil {
			return nil, fmt.Errorf("failed to create invalid intent file %s: %w", filename, err)
		}
		filePaths = append(filePaths, filePath)
	}
	
	return filePaths, nil
}

// GetAllInvalidTypes returns all available invalid target types for testing
func (f *InvalidFixtureFactory) GetAllInvalidTypes() []string {
	return []string{
		"string_target",
		"missing_type", 
		"missing_name",
		"empty_target",
		"invalid_type",
		"empty_type",
		"empty_name",
		"null_target",
		"non_string_type",
		"non_string_name",
		"array_target",
		"nested_target",
	}
}

// CreateTestSuiteInvalidIntents creates a comprehensive test suite of invalid intents
func (f *InvalidFixtureFactory) CreateTestSuiteInvalidIntents(baseName string) (map[string]string, error) {
	invalidTypes := f.GetAllInvalidTypes()
	filePaths := make(map[string]string)
	
	for _, invalidType := range invalidTypes {
		filename := fmt.Sprintf("%s-%s.json", baseName, invalidType)
		filePath, err := f.CreateIntentFileWithInvalidTarget(filename, invalidType)
		if err != nil {
			return nil, fmt.Errorf("failed to create invalid intent file %s: %w", filename, err)
		}
		filePaths[invalidType] = filePath
	}
	
	return filePaths, nil
}

// GetCreatedFileCount returns the number of invalid files created by this factory
func (f *InvalidFixtureFactory) GetCreatedFileCount() int {
	return f.Counter
}

// Reset resets the factory counter
func (f *InvalidFixtureFactory) Reset() {
	f.Counter = 0
}

// Helper function to create files with content
func createFileWithContent(filePath string, content []byte) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}
	
	return os.WriteFile(filePath, content, 0644)
}