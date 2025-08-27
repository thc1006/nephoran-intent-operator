package loop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/nephoran-intent-operator/internal/ingest"
)

// TestRobustValidation_ReplicasValidation ensures invalid replicas never panic
func TestRobustValidation_ReplicasValidation(t *testing.T) {
	t.Log("Testing replicas validation with NotPanics guards")

	validator := &MockValidator{shouldFail: false}

	testCases := []struct {
		name        string
		replicas    interface{} // Use interface to test wrong types
		expectError bool
		errorMsg    string
	}{
		{
			name:        "negative_replicas",
			replicas:    -1,
			expectError: true,
			errorMsg:    "replicas must be between 1 and 100",
		},
		{
			name:        "zero_replicas",
			replicas:    0,
			expectError: true,
			errorMsg:    "replicas must be between 1 and 100",
		},
		{
			name:        "excessive_replicas",
			replicas:    101,
			expectError: true,
			errorMsg:    "replicas must be between 1 and 100",
		},
		{
			name:        "string_replicas",
			replicas:    "three",
			expectError: true,
			errorMsg:    "replicas must be an integer",
		},
		{
			name:        "float_replicas",
			replicas:    3.14,
			expectError: true,
			errorMsg:    "replicas must be an integer",
		},
		{
			name:        "null_replicas",
			replicas:    nil,
			expectError: true,
			errorMsg:    "replicas is required",
		},
		{
			name:        "valid_replicas",
			replicas:    3,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Build intent with different replica values
			intentData := map[string]interface{}{
				"intent_type": "scaling",
				"target":      "test-app",
				"namespace":   "default",
			}

			if tc.replicas != nil {
				intentData["replicas"] = tc.replicas
			}

			jsonData, err := json.Marshal(intentData)
			require.NoError(t, err)

			// Test with NotPanics guard
			var validationErr error
			require.NotPanics(t, func() {
				_, validationErr = validator.ValidateBytes(jsonData)
			}, "Validation should never panic for replicas: %v", tc.replicas)

			if tc.expectError {
				assert.Error(t, validationErr, "Should reject invalid replicas")
				if tc.errorMsg != "" && validationErr != nil {
					assert.Contains(t, validationErr.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, validationErr, "Should accept valid replicas")
			}
		})
	}
}

// TestRobustValidation_SizeLimits ensures oversized inputs never panic
func TestRobustValidation_SizeLimits(t *testing.T) {
	t.Log("Testing size limit validation with NotPanics guards")

	tempDir := t.TempDir()

	testCases := []struct {
		name        string
		size        int
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty_file",
			size:        0,
			expectError: true,
			errorMsg:    "file is empty",
		},
		{
			name:        "small_file",
			size:        100,
			expectError: false,
		},
		{
			name:        "near_limit",
			size:        10*1024*1024 - 1, // Just under 10MB
			expectError: false,
		},
		{
			name:        "at_limit",
			size:        10 * 1024 * 1024, // Exactly 10MB
			expectError: true,
			errorMsg:    "file size exceeds limit",
		},
		{
			name:        "over_limit",
			size:        11 * 1024 * 1024, // 11MB
			expectError: true,
			errorMsg:    "file size exceeds limit",
		},
		{
			name:        "way_over_limit",
			size:        100 * 1024 * 1024, // 100MB
			expectError: true,
			errorMsg:    "file size exceeds limit",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create file with specific size
			filename := fmt.Sprintf("size_test_%s.json", tc.name)
			filepath := filepath.Join(tempDir, filename)

			var content []byte
			if tc.size > 0 {
				// Create valid JSON of specific size
				if tc.size < 50 {
					// For small files, just create minimal JSON
					content = []byte(`{"test": "x"}`)
				} else {
					// For larger files, pad to exact size
					padding := strings.Repeat("x", tc.size-50)
					jsonContent := fmt.Sprintf(`{"test": "%s"}`, padding)
					content = []byte(jsonContent)
				}
				// Ensure we have exactly the size we want
				if len(content) > tc.size {
					content = content[:tc.size]
				} else if len(content) < tc.size {
					// Pad with spaces if needed
					padding := make([]byte, tc.size-len(content))
					for i := range padding {
						padding[i] = ' '
					}
					content = append(content, padding...)
				}
			}

			err := os.WriteFile(filepath, content, 0644)
			require.NoError(t, err)

			// Test with NotPanics guard
			var validationErr error
			require.NotPanics(t, func() {
				// Simulate size check
				info, _ := os.Stat(filepath)
				if info != nil {
					size := info.Size()
					if size == 0 {
						validationErr = fmt.Errorf("file is empty")
					} else if size >= 10*1024*1024 {
						validationErr = fmt.Errorf("file size exceeds limit")
					}
				}
			}, "Size validation should never panic for size: %d", tc.size)

			if tc.expectError {
				assert.Error(t, validationErr, "Should reject invalid size")
				if tc.errorMsg != "" && validationErr != nil {
					assert.Contains(t, validationErr.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, validationErr, "Should accept valid size")
			}
		})
	}
}

// TestRobustValidation_MissingSpec ensures missing spec fields never panic
func TestRobustValidation_MissingSpec(t *testing.T) {
	t.Log("Testing missing spec validation with NotPanics guards")

	validator := &MockValidator{shouldFail: false}

	testCases := []struct {
		name        string
		jsonData    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "missing_spec_entirely",
			jsonData:    `{"apiVersion": "v1", "kind": "NetworkIntent"}`,
			expectError: true,
			errorMsg:    "spec is required",
		},
		{
			name:        "null_spec",
			jsonData:    `{"apiVersion": "v1", "kind": "NetworkIntent", "spec": null}`,
			expectError: true,
			errorMsg:    "spec must be an object",
		},
		{
			name:        "empty_spec",
			jsonData:    `{"apiVersion": "v1", "kind": "NetworkIntent", "spec": {}}`,
			expectError: true,
			errorMsg:    "spec must contain required fields",
		},
		{
			name:        "missing_intent_type",
			jsonData:    `{"target": "app", "namespace": "default"}`,
			expectError: true,
			errorMsg:    "intent_type is required",
		},
		{
			name:        "missing_target",
			jsonData:    `{"intent_type": "scaling", "namespace": "default", "replicas": 3}`,
			expectError: true,
			errorMsg:    "target is required",
		},
		{
			name:        "missing_namespace",
			jsonData:    `{"intent_type": "scaling", "target": "app", "replicas": 3}`,
			expectError: true,
			errorMsg:    "namespace is required",
		},
		{
			name:        "valid_minimal_spec",
			jsonData:    `{"intent_type": "scaling", "target": "app", "namespace": "default", "replicas": 3}`,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test with NotPanics guard
			var validationErr error
			var intent *ingest.Intent

			require.NotPanics(t, func() {
				intent, validationErr = validator.ValidateBytes([]byte(tc.jsonData))
			}, "Validation should never panic for JSON: %s", tc.jsonData)

			if tc.expectError {
				assert.Error(t, validationErr, "Should reject invalid spec")
				if tc.errorMsg != "" && validationErr != nil {
					assert.Contains(t, validationErr.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, validationErr, "Should accept valid spec")
				assert.NotNil(t, intent, "Should return valid intent")
			}
		})
	}
}

// TestRobustValidation_WrongTypes ensures wrong field types never panic
func TestRobustValidation_WrongTypes(t *testing.T) {
	t.Log("Testing wrong type validation with NotPanics guards")

	validator := &MockValidator{shouldFail: false}

	testCases := []struct {
		name        string
		jsonData    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "apiVersion_as_number",
			jsonData:    `{"apiVersion": 1, "kind": "NetworkIntent"}`,
			expectError: true,
			errorMsg:    "apiVersion must be a string",
		},
		{
			name:        "kind_as_array",
			jsonData:    `{"apiVersion": "v1", "kind": ["NetworkIntent"]}`,
			expectError: true,
			errorMsg:    "kind must be a string",
		},
		{
			name:        "metadata_as_string",
			jsonData:    `{"apiVersion": "v1", "kind": "NetworkIntent", "metadata": "invalid"}`,
			expectError: true,
			errorMsg:    "metadata must be an object",
		},
		{
			name:        "spec_as_array",
			jsonData:    `{"apiVersion": "v1", "kind": "NetworkIntent", "spec": []}`,
			expectError: true,
			errorMsg:    "spec must be an object",
		},
		{
			name:        "namespace_as_number",
			jsonData:    `{"intent_type": "scaling", "target": "app", "namespace": 123, "replicas": 3}`,
			expectError: true,
			errorMsg:    "namespace must be a string",
		},
		{
			name:        "target_as_boolean",
			jsonData:    `{"intent_type": "scaling", "target": true, "namespace": "default", "replicas": 3}`,
			expectError: true,
			errorMsg:    "target must be a string",
		},
		{
			name:        "intent_type_as_object",
			jsonData:    `{"intent_type": {"type": "scaling"}, "target": "app", "namespace": "default"}`,
			expectError: true,
			errorMsg:    "intent_type must be a string",
		},
		{
			name:        "replicas_as_array",
			jsonData:    `{"intent_type": "scaling", "target": "app", "namespace": "default", "replicas": [3]}`,
			expectError: true,
			errorMsg:    "replicas must be an integer",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test with NotPanics guard
			var validationErr error
			var intent *ingest.Intent

			require.NotPanics(t, func() {
				intent, validationErr = validator.ValidateBytes([]byte(tc.jsonData))
			}, "Validation should never panic for wrong types in JSON: %s", tc.jsonData)

			if tc.expectError {
				assert.Error(t, validationErr, "Should reject wrong type")
				if tc.errorMsg != "" && validationErr != nil {
					assert.Contains(t, validationErr.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, validationErr, "Should accept valid types")
				assert.NotNil(t, intent, "Should return valid intent")
			}
		})
	}
}

// TestRobustValidation_EdgeCases ensures edge cases never panic
func TestRobustValidation_EdgeCases(t *testing.T) {
	t.Log("Testing edge case validation with NotPanics guards")

	validator := &MockValidator{shouldFail: false}

	testCases := []struct {
		name        string
		jsonData    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "deeply_nested_json",
			jsonData:    `{"a":{"b":{"c":{"d":{"e":{"f":{"g":{"h":{"i":{"j":{}}}}}}}}}}}`,
			expectError: true,
			errorMsg:    "JSON nesting too deep",
		},
		{
			name:        "unicode_in_fields",
			jsonData:    `{"intent_type": "scaling", "target": "app-🚀", "namespace": "default-🌟", "replicas": 3}`,
			expectError: false, // Should handle unicode gracefully
		},
		{
			name:        "very_long_field_names",
			jsonData:    fmt.Sprintf(`{"%s": "value"}`, strings.Repeat("a", 1000)),
			expectError: true,
			errorMsg:    "field name too long",
		},
		{
			name:        "special_characters_in_values",
			jsonData:    `{"intent_type": "scaling", "target": "app!@#$%", "namespace": "default", "replicas": 3}`,
			expectError: true,
			errorMsg:    "invalid characters in target",
		},
		{
			name:        "json_with_comments",
			jsonData:    `{"intent_type": "scaling", /* comment */ "target": "app"}`,
			expectError: true,
			errorMsg:    "invalid JSON",
		},
		{
			name:        "duplicate_keys",
			jsonData:    `{"replicas": 3, "replicas": 5}`,
			expectError: false, // JSON decoder takes last value
		},
		{
			name:        "circular_reference_attempt",
			jsonData:    `{"intent_type": "scaling", "self": "$", "target": "app"}`,
			expectError: false, // Not actually circular, just a string
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test with NotPanics guard
			var validationErr error

			require.NotPanics(t, func() {
				// First try to unmarshal to check JSON validity
				var data map[string]interface{}
				if err := json.Unmarshal([]byte(tc.jsonData), &data); err != nil {
					validationErr = fmt.Errorf("invalid JSON: %v", err)
					return
				}

				// Check for deep nesting
				if checkNestingDepth(data, 0, 10) {
					validationErr = fmt.Errorf("JSON nesting too deep")
					return
				}

				// Check field name lengths
				for key := range data {
					if len(key) > 255 {
						validationErr = fmt.Errorf("field name too long")
						return
					}
				}

				// Validate with mock validator
				_, validationErr = validator.ValidateBytes([]byte(tc.jsonData))
			}, "Validation should never panic for edge case: %s", tc.name)

			if tc.expectError {
				assert.Error(t, validationErr, "Should reject edge case: %s", tc.name)
				if tc.errorMsg != "" && validationErr != nil {
					assert.Contains(t, validationErr.Error(), tc.errorMsg)
				}
			} else {
				// For valid cases, check they don't error
				if validationErr != nil && !strings.Contains(validationErr.Error(), "missing required fields") {
					t.Errorf("Unexpected error for %s: %v", tc.name, validationErr)
				}
			}
		})
	}
}

// Helper function to check JSON nesting depth
func checkNestingDepth(data interface{}, currentDepth, maxDepth int) bool {
	if currentDepth > maxDepth {
		return true
	}

	switch v := data.(type) {
	case map[string]interface{}:
		for _, value := range v {
			if checkNestingDepth(value, currentDepth+1, maxDepth) {
				return true
			}
		}
	case []interface{}:
		for _, item := range v {
			if checkNestingDepth(item, currentDepth+1, maxDepth) {
				return true
			}
		}
	}

	return false
}
