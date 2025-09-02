package intent

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// TestLoader_MalformedJSONEdgeCases tests edge cases with malformed JSON data
func TestLoader_MalformedJSONEdgeCases(t *testing.T) {
	loader, cleanup := createTestLoader(t)
	defer cleanup()

	tests := []struct {
		name        string
		jsonData    []byte
		expectError bool
		description string
	}{
		{
			name:        "extremely large JSON object",
			jsonData:    createLargeJSON(t, 1000000), // 1MB of data
			expectError: true,
			description: "Large JSON should be handled gracefully",
		},
		{
			name:        "deeply nested JSON",
			jsonData:    createDeeplyNestedJSON(t, 1000),
			expectError: true,
			description: "Deeply nested JSON should be rejected",
		},
		{
			name:        "JSON with unicode characters",
			jsonData:    []byte(`{"intent_type": "scaling", "target": "测试-应用", "namespace": "命名空间", "replicas": 3}`),
			expectError: true,
			description: "Unicode characters should be handled properly",
		},
		{
			name:        "JSON with escaped characters",
			jsonData:    []byte(`{"intent_type": "scaling", "target": "test\napp", "namespace": "test\tns", "replicas": 3}`),
			expectError: true,
			description: "Escaped characters should be handled",
		},
		{
			name:        "JSON with null values",
			jsonData:    []byte(`{"intent_type": null, "target": "test", "namespace": "default", "replicas": 3}`),
			expectError: true,
			description: "Null values should be rejected for required fields",
		},
		{
			name:        "JSON with mixed types",
			jsonData:    []byte(`{"intent_type": "scaling", "target": 123, "namespace": ["array"], "replicas": "string"}`),
			expectError: true,
			description: "Type mismatches should be caught",
		},
		{
			name:        "JSON with floating point numbers",
			jsonData:    []byte(`{"intent_type": "scaling", "target": "test", "namespace": "default", "replicas": 3.14}`),
			expectError: true,
			description: "Floating point replicas should be rejected",
		},
		{
			name:        "JSON with scientific notation",
			jsonData:    []byte(`{"intent_type": "scaling", "target": "test", "namespace": "default", "replicas": 1e2}`),
			expectError: true,
			description: "Scientific notation should be handled",
		},
		{
			name:        "JSON with very large numbers",
			jsonData:    []byte(fmt.Sprintf(`{"intent_type": "scaling", "target": "test", "namespace": "default", "replicas": %d}`, math.MaxInt64)),
			expectError: true,
			description: "Very large numbers should be rejected",
		},
		{
			name:        "JSON with negative numbers",
			jsonData:    []byte(`{"intent_type": "scaling", "target": "test", "namespace": "default", "replicas": -1}`),
			expectError: true,
			description: "Negative replicas should be rejected",
		},
		{
			name:        "JSON with boolean values",
			jsonData:    []byte(`{"intent_type": "scaling", "target": "test", "namespace": "default", "replicas": true}`),
			expectError: true,
			description: "Boolean values should be rejected for numeric fields",
		},
		{
			name:        "JSON with array values",
			jsonData:    []byte(`{"intent_type": ["scaling"], "target": "test", "namespace": "default", "replicas": 3}`),
			expectError: true,
			description: "Array values should be rejected for string fields",
		},
		{
			name:        "JSON with object values",
			jsonData:    []byte(`{"intent_type": "scaling", "target": {"name": "test"}, "namespace": "default", "replicas": 3}`),
			expectError: true,
			description: "Object values should be rejected for string fields",
		},
		{
			name:        "JSON with binary data encoded as base64",
			jsonData:    []byte(`{"intent_type": "scaling", "target": "dGVzdA==", "namespace": "default", "replicas": 3}`),
			expectError: true,
			description: "Base64 data with special characters should be rejected for Kubernetes names",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := loader.LoadFromJSON(tt.jsonData, "test.json")

			if tt.expectError {
				if err == nil && result.IsValid {
					t.Errorf("Expected error but got valid result: %s", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v (%s)", err, tt.description)
				}
				if !result.IsValid {
					t.Errorf("Expected valid result but got errors: %v (%s)", result.Errors, tt.description)
				}
			}
		})
	}
}

// TestLoader_ExtremeValueEdgeCases tests extreme values that could cause issues
func TestLoader_ExtremeValueEdgeCases(t *testing.T) {
	loader, cleanup := createTestLoader(t)
	defer cleanup()

	tests := []struct {
		name        string
		intent      ScalingIntent
		expectError bool
		description string
	}{
		{
			name: "extremely long target name",
			intent: ScalingIntent{
				IntentType: "scaling",
				Target:     strings.Repeat("a", 10000),
				Namespace:  "default",
				Replicas:   3,
			},
			expectError: true,
			description: "Very long target names should be rejected",
		},
		{
			name: "target with all special characters",
			intent: ScalingIntent{
				IntentType: "scaling",
				Target:     "!@#$%^&*()_+",
				Namespace:  "default",
				Replicas:   3,
			},
			expectError: true,
			description: "Special characters should be rejected",
		},
		{
			name: "target with unicode control characters",
			intent: ScalingIntent{
				IntentType: "scaling",
				Target:     "test\x00\x01\x02",
				Namespace:  "default",
				Replicas:   3,
			},
			expectError: true,
			description: "Control characters should be rejected",
		},
		{
			name: "target with emoji",
			intent: ScalingIntent{
				IntentType: "scaling",
				Target:     "test-app-😀",
				Namespace:  "default",
				Replicas:   3,
			},
			expectError: true,
			description: "Emoji characters should be rejected",
		},
		{
			name: "namespace with uppercase letters",
			intent: ScalingIntent{
				IntentType: "scaling",
				Target:     "test-app",
				Namespace:  "UPPERCASE",
				Replicas:   3,
			},
			expectError: true,
			description: "Uppercase letters should be rejected",
		},
		{
			name: "namespace starting with number",
			intent: ScalingIntent{
				IntentType: "scaling",
				Target:     "test-app",
				Namespace:  "1namespace",
				Replicas:   3,
			},
			expectError: true,
			description: "Names starting with numbers should be rejected",
		},
		{
			name: "namespace ending with hyphen",
			intent: ScalingIntent{
				IntentType: "scaling",
				Target:     "test-app",
				Namespace:  "namespace-",
				Replicas:   3,
			},
			expectError: true,
			description: "Names ending with hyphens should be rejected",
		},
		{
			name: "reason with extremely long text",
			intent: ScalingIntent{
				IntentType: "scaling",
				Target:     "test-app",
				Namespace:  "default",
				Replicas:   3,
				Reason:     strings.Repeat("a", 1000),
			},
			expectError: true,
			description: "Very long reason should be rejected",
		},
		{
			name: "correlation_id with invalid characters",
			intent: ScalingIntent{
				IntentType:    "scaling",
				Target:        "test-app",
				Namespace:     "default",
				Replicas:      3,
				CorrelationID: "test\n\r\t",
			},
			expectError: false,
			description: "Correlation ID allows more flexible format",
		},
		{
			name: "zero replicas",
			intent: ScalingIntent{
				IntentType: "scaling",
				Target:     "test-app",
				Namespace:  "default",
				Replicas:   0,
			},
			expectError: true,
			description: "Zero replicas should be rejected",
		},
		{
			name: "business limit exceeded replicas",
			intent: ScalingIntent{
				IntentType: "scaling",
				Target:     "test-app",
				Namespace:  "default",
				Replicas:   51,
			},
			expectError: true,
			description: "Replicas exceeding business limit should be rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, err := json.Marshal(tt.intent)
			if err != nil {
				t.Fatalf("Failed to marshal intent: %v", err)
			}

			result, err := loader.LoadFromJSON(jsonData, "test.json")

			if tt.expectError {
				if err == nil && result.IsValid {
					t.Errorf("Expected error but got valid result: %s", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v (%s)", err, tt.description)
				}
				if !result.IsValid {
					t.Errorf("Expected valid result but got errors: %v (%s)", result.Errors, tt.description)
				}
			}
		})
	}
}

// TestLoader_FileSystemEdgeCases tests filesystem-related edge cases
func TestLoader_FileSystemEdgeCases(t *testing.T) {
	loader, cleanup := createTestLoader(t)
	defer cleanup()

	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) string
		expectError bool
		description string
	}{
		{
			name: "file with BOM (Byte Order Mark)",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				filePath := filepath.Join(tempDir, "intent-bom.json")

				// Add UTF-8 BOM to the beginning
				content := "\xEF\xBB\xBF" + `{"intent_type": "scaling", "target": "test", "namespace": "default", "replicas": 3}`
				err := os.WriteFile(filePath, []byte(content), 0o644)
				if err != nil {
					t.Fatalf("Failed to write file: %v", err)
				}
				return filePath
			},
			expectError: true,
			description: "Files with BOM should be handled",
		},
		{
			name: "file with different line endings",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				filePath := filepath.Join(tempDir, "intent-crlf.json")

				// Use Windows line endings
				content := "{\r\n  \"intent_type\": \"scaling\",\r\n  \"target\": \"test\",\r\n  \"namespace\": \"default\",\r\n  \"replicas\": 3\r\n}"
				err := os.WriteFile(filePath, []byte(content), 0o644)
				if err != nil {
					t.Fatalf("Failed to write file: %v", err)
				}
				return filePath
			},
			expectError: false,
			description: "Different line endings should be handled",
		},
		{
			name: "file with only whitespace",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				filePath := filepath.Join(tempDir, "whitespace.json")

				content := "   \t\n\r   "
				err := os.WriteFile(filePath, []byte(content), 0o644)
				if err != nil {
					t.Fatalf("Failed to write file: %v", err)
				}
				return filePath
			},
			expectError: true,
			description: "Whitespace-only files should be rejected",
		},
		{
			name: "file with mixed encoding",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				filePath := filepath.Join(tempDir, "mixed-encoding.json")

				// Mix of valid UTF-8 and invalid bytes
				validJSON := `{"intent_type": "scaling", "target": "test", "namespace": "default", "replicas": 3}`
				content := []byte(validJSON)
				// Insert invalid UTF-8 byte sequence
				content = append(content[:10], append([]byte{0xFF, 0xFE}, content[10:]...)...)

				err := os.WriteFile(filePath, content, 0o644)
				if err != nil {
					t.Fatalf("Failed to write file: %v", err)
				}
				return filePath
			},
			expectError: true,
			description: "Files with invalid UTF-8 should be rejected",
		},
		{
			name: "extremely large file",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				filePath := filepath.Join(tempDir, "large.json")

				// Create 10MB JSON file
				largeContent := `{"intent_type": "scaling", "target": "` + strings.Repeat("a", 10*1024*1024) + `", "namespace": "default", "replicas": 3}`
				err := os.WriteFile(filePath, []byte(largeContent), 0o644)
				if err != nil {
					t.Fatalf("Failed to write large file: %v", err)
				}
				return filePath
			},
			expectError: true,
			description: "Very large files should be handled gracefully",
		},
		{
			name: "file with symlink",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				originalFile := filepath.Join(tempDir, "original.json")
				symlinkFile := filepath.Join(tempDir, "symlink.json")

				content := `{"intent_type": "scaling", "target": "test", "namespace": "default", "replicas": 3}`
				err := os.WriteFile(originalFile, []byte(content), 0o644)
				if err != nil {
					t.Fatalf("Failed to write original file: %v", err)
				}

				err = os.Symlink(originalFile, symlinkFile)
				if err != nil {
					t.Skipf("Cannot create symlink on this system: %v", err)
				}

				return symlinkFile
			},
			expectError: false,
			description: "Symlinks should be followed and handled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := tt.setupFunc(t)

			result, err := loader.LoadFromFile(filePath)

			if tt.expectError {
				if err == nil && result.IsValid {
					t.Errorf("Expected error but got valid result: %s", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v (%s)", err, tt.description)
				}
				if !result.IsValid {
					t.Errorf("Expected valid result but got errors: %v (%s)", result.Errors, tt.description)
				}
			}
		})
	}
}

// TestLoader_ConcurrencyEdgeCases tests concurrent access scenarios
func TestLoader_ConcurrencyEdgeCases(t *testing.T) {
	loader, cleanup := createTestLoader(t)
	defer cleanup()

	validJSON := []byte(`{"intent_type": "scaling", "target": "concurrent-test", "namespace": "default", "replicas": 3}`)
	invalidJSON := []byte(`{"intent_type": "invalid", "target": "test", "namespace": "default", "replicas": 3}`)

	const numGoroutines = 100
	const numIterations = 10

	done := make(chan error, numGoroutines*numIterations)

	// Launch many concurrent operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numIterations; j++ {
				var data []byte
				var expectValid bool

				if (id+j)%2 == 0 {
					data = validJSON
					expectValid = true
				} else {
					data = invalidJSON
					expectValid = false
				}

				result, err := loader.LoadFromJSON(data, fmt.Sprintf("concurrent-%d-%d.json", id, j))

				if expectValid {
					if err != nil || !result.IsValid {
						done <- fmt.Errorf("goroutine %d iteration %d: expected valid result but got error: %v", id, j, err)
						return
					}
				} else {
					if err == nil && result.IsValid {
						done <- fmt.Errorf("goroutine %d iteration %d: expected invalid result but got valid", id, j)
						return
					}
				}

				done <- nil
			}
		}(i)
	}

	// Wait for all operations to complete
	errorCount := 0
	for i := 0; i < numGoroutines*numIterations; i++ {
		if err := <-done; err != nil {
			errorCount++
			if errorCount == 1 {
				t.Logf("First concurrent error: %v", err)
			}
		}
	}

	if errorCount > 0 {
		t.Errorf("Concurrent operations failed: %d/%d", errorCount, numGoroutines*numIterations)
	}
}

// TestLoader_MemoryExhaustionEdgeCases tests memory-related edge cases
func TestLoader_MemoryExhaustionEdgeCases(t *testing.T) {
	loader, cleanup := createTestLoader(t)
	defer cleanup()

	tests := []struct {
		name        string
		dataFunc    func() []byte
		expectError bool
		description string
	}{
		{
			name: "many repeated keys",
			dataFunc: func() []byte {
				json := `{"intent_type": "scaling", "target": "test", "namespace": "default", "replicas": 3`
				for i := 0; i < 10000; i++ {
					json += fmt.Sprintf(`, "extra_%d": "value"`, i)
				}
				json += `}`
				return []byte(json)
			},
			expectError: true,
			description: "JSON with many keys should be handled",
		},
		{
			name: "extremely long string values",
			dataFunc: func() []byte {
				longString := strings.Repeat("a", 1000000)
				return []byte(fmt.Sprintf(`{"intent_type": "scaling", "target": "%s", "namespace": "default", "replicas": 3}`, longString))
			},
			expectError: true,
			description: "Very long string values should be rejected",
		},
		{
			name: "many nested arrays",
			dataFunc: func() []byte {
				json := `{"intent_type": "scaling", "target": "test", "namespace": "default", "replicas": 3, "nested": `
				for i := 0; i < 1000; i++ {
					json += `[`
				}
				json += `"deep"`
				for i := 0; i < 1000; i++ {
					json += `]`
				}
				json += `}`
				return []byte(json)
			},
			expectError: true,
			description: "Deeply nested arrays should be rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.dataFunc()

			result, err := loader.LoadFromJSON(data, "memory-test.json")

			if tt.expectError {
				if err == nil && result.IsValid {
					t.Errorf("Expected error but got valid result: %s", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v (%s)", err, tt.description)
				}
			}
		})
	}
}

// Helper functions

// createTestLoader creates a test loader with proper cleanup
func createTestLoader(t *testing.T) (*Loader, func()) {
	t.Helper()

	tempDir := t.TempDir()
	schemaDir := filepath.Join(tempDir, "docs", "contracts")
	err := os.MkdirAll(schemaDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create schema directory: %v", err)
	}

	schemaPath := filepath.Join(schemaDir, "intent.schema.json")
	schema := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$id": "https://example.com/schemas/intent.schema.json",
		"title": "ScalingIntent",
		"type": "object",
		"additionalProperties": false,
		"required": ["intent_type", "target", "namespace", "replicas"],
		"properties": {
			"intent_type": {
				"const": "scaling"
			},
			"target": {
				"type": "string",
				"minLength": 1
			},
			"namespace": {
				"type": "string",
				"minLength": 1
			},
			"replicas": {
				"type": "integer",
				"minimum": 1,
				"maximum": 100
			},
			"reason": {
				"type": "string",
				"maxLength": 512
			},
			"source": {
				"type": "string",
				"enum": ["user", "planner", "test"]
			},
			"correlation_id": {
				"type": "string"
			}
		}
	}`

	err = os.WriteFile(schemaPath, []byte(schema), 0o644)
	if err != nil {
		t.Fatalf("Failed to write schema file: %v", err)
	}

	loader, err := NewLoader(tempDir)
	if err != nil {
		t.Fatalf("Failed to create loader: %v", err)
	}

	cleanup := func() {
		// Cleanup is handled by t.TempDir()
	}

	return loader, cleanup
}

// createLargeJSON creates a large JSON payload for testing
func createLargeJSON(t *testing.T, size int) []byte {
	t.Helper()

	// Create a JSON with a very large target name
	largeString := strings.Repeat("a", size)
	json := fmt.Sprintf(`{"intent_type": "scaling", "target": "%s", "namespace": "default", "replicas": 3}`, largeString)
	return []byte(json)
}

// createDeeplyNestedJSON creates deeply nested JSON for testing
func createDeeplyNestedJSON(t *testing.T, depth int) []byte {
	t.Helper()

	json := `{"intent_type": "scaling", "target": "test", "namespace": "default", "replicas": 3, "nested":`

	for i := 0; i < depth; i++ {
		json += `{"level": `
	}
	json += `"deep"`
	for i := 0; i < depth; i++ {
		json += `}`
	}
	json += `}`

	return []byte(json)
}

// TestLoader_UnicodeEdgeCases tests Unicode handling edge cases
func TestLoader_UnicodeEdgeCases(t *testing.T) {
	loader, cleanup := createTestLoader(t)
	defer cleanup()

	tests := []struct {
		name        string
		jsonData    string
		expectError bool
		description string
	}{
		{
			name:        "surrogate pairs",
			jsonData:    `{"intent_type": "scaling", "target": "test-\uD83D\uDE00", "namespace": "default", "replicas": 3}`,
			expectError: true,
			description: "Surrogate pairs should be handled",
		},
		{
			name:        "invalid unicode sequences",
			jsonData:    `{"intent_type": "scaling", "target": "test-\uD800", "namespace": "default", "replicas": 3}`,
			expectError: true,
			description: "Invalid Unicode should be rejected",
		},
		{
			name:        "zero width characters",
			jsonData:    `{"intent_type": "scaling", "target": "test\u200B", "namespace": "default", "replicas": 3}`,
			expectError: true,
			description: "Zero-width characters should be handled",
		},
		{
			name:        "right-to-left marks",
			jsonData:    `{"intent_type": "scaling", "target": "test\u200F", "namespace": "default", "replicas": 3}`,
			expectError: true,
			description: "RTL marks should be handled",
		},
		{
			name:        "combining characters",
			jsonData:    `{"intent_type": "scaling", "target": "test\u0300", "namespace": "default", "replicas": 3}`,
			expectError: true,
			description: "Combining characters should be handled",
		},
		{
			name:        "valid ascii",
			jsonData:    `{"intent_type": "scaling", "target": "test-app", "namespace": "default", "replicas": 3}`,
			expectError: false,
			description: "Valid ASCII should work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate the test JSON is valid UTF-8
			if !utf8.ValidString(tt.jsonData) {
				t.Logf("Test JSON contains invalid UTF-8: %s", tt.name)
			}

			result, err := loader.LoadFromJSON([]byte(tt.jsonData), "unicode-test.json")

			if tt.expectError {
				if err == nil && result.IsValid {
					t.Errorf("Expected error but got valid result: %s", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v (%s)", err, tt.description)
				}
				if !result.IsValid {
					t.Errorf("Expected valid result but got errors: %v (%s)", result.Errors, tt.description)
				}
			}
		})
	}
}
