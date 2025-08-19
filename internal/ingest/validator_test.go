package ingest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewValidator(t *testing.T) {
	// Create a temporary schema file for testing
	tempDir := t.TempDir()
	schemaDir := filepath.Join(tempDir, "docs", "contracts")
	err := os.MkdirAll(schemaDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp schema dir: %v", err)
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

	err = os.WriteFile(schemaPath, []byte(schema), 0644)
	if err != nil {
		t.Fatalf("Failed to write temp schema file: %v", err)
	}

	tests := []struct {
		name        string
		schemaPath  string
		expectError bool
	}{
		{
			name:        "valid schema file path",
			schemaPath:  schemaPath,
			expectError: false,
		},
		{
			name:        "valid schema directory path",
			schemaPath:  tempDir,
			expectError: false,
		},
		{
			name:        "non-existent schema file",
			schemaPath:  "/non/existent/path/schema.json",
			expectError: true,
		},
		{
			name:        "invalid schema json",
			schemaPath:  createInvalidSchemaFile(t, tempDir),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator, err := NewValidator(tt.schemaPath)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				if validator != nil {
					t.Errorf("Expected nil validator but got non-nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if validator == nil {
					t.Errorf("Expected non-nil validator but got nil")
				}
			}
		})
	}
}

func createInvalidSchemaFile(t *testing.T, dir string) string {
	invalidSchemaPath := filepath.Join(dir, "invalid.json")
	err := os.WriteFile(invalidSchemaPath, []byte(`{invalid json`), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid schema file: %v", err)
	}
	return invalidSchemaPath
}

func TestValidateBytes_ValidCases(t *testing.T) {
	validator := createTestValidator(t)

	tests := []struct {
		name     string
		jsonData string
		expected Intent
	}{
		{
			name: "minimal valid intent",
			jsonData: `{
				"intent_type": "scaling",
				"target": "my-deployment",
				"namespace": "default",
				"replicas": 3
			}`,
			expected: Intent{
				IntentType: "scaling",
				Target:     "my-deployment",
				Namespace:  "default",
				Replicas:   3,
			},
		},
		{
			name: "complete intent with all optional fields",
			jsonData: `{
				"intent_type": "scaling",
				"target": "web-server",
				"namespace": "production",
				"replicas": 10,
				"reason": "High traffic expected",
				"source": "user",
				"correlation_id": "req-12345"
			}`,
			expected: Intent{
				IntentType:    "scaling",
				Target:        "web-server",
				Namespace:     "production",
				Replicas:      10,
				Reason:        "High traffic expected",
				Source:        "user",
				CorrelationID: "req-12345",
			},
		},
		{
			name: "intent with planner source",
			jsonData: `{
				"intent_type": "scaling",
				"target": "api-service",
				"namespace": "staging",
				"replicas": 5,
				"source": "planner"
			}`,
			expected: Intent{
				IntentType: "scaling",
				Target:     "api-service",
				Namespace:  "staging",
				Replicas:   5,
				Source:     "planner",
			},
		},
		{
			name: "intent with test source",
			jsonData: `{
				"intent_type": "scaling",
				"target": "test-deployment",
				"namespace": "test",
				"replicas": 1,
				"source": "test",
				"correlation_id": "test-correlation-123"
			}`,
			expected: Intent{
				IntentType:    "scaling",
				Target:        "test-deployment",
				Namespace:     "test",
				Replicas:      1,
				Source:        "test",
				CorrelationID: "test-correlation-123",
			},
		},
		{
			name: "intent with maximum replicas",
			jsonData: `{
				"intent_type": "scaling",
				"target": "large-deployment",
				"namespace": "production",
				"replicas": 100
			}`,
			expected: Intent{
				IntentType: "scaling",
				Target:     "large-deployment",
				Namespace:  "production",
				Replicas:   100,
			},
		},
		{
			name: "intent with long reason",
			jsonData: `{
				"intent_type": "scaling",
				"target": "deployment",
				"namespace": "default",
				"replicas": 2,
				"reason": "` + strings.Repeat("a", 512) + `"
			}`,
			expected: Intent{
				IntentType: "scaling",
				Target:     "deployment",
				Namespace:  "default",
				Replicas:   2,
				Reason:     strings.Repeat("a", 512),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidateBytes([]byte(tt.jsonData))
			if err != nil {
				t.Errorf("Expected no error but got: %v", err)
				return
			}

			if result == nil {
				t.Errorf("Expected non-nil result")
				return
			}

			if *result != tt.expected {
				t.Errorf("Expected %+v, got %+v", tt.expected, *result)
			}
		})
	}
}

func TestValidateBytes_InvalidJSON(t *testing.T) {
	validator := createTestValidator(t)

	tests := []struct {
		name     string
		jsonData string
	}{
		{
			name:     "malformed json",
			jsonData: `{invalid json`,
		},
		{
			name:     "empty string",
			jsonData: "",
		},
		{
			name:     "not json object",
			jsonData: `"just a string"`,
		},
		{
			name:     "unclosed brace",
			jsonData: `{"intent_type": "scaling"`,
		},
		{
			name:     "trailing comma",
			jsonData: `{"intent_type": "scaling",}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidateBytes([]byte(tt.jsonData))
			if err == nil {
				t.Errorf("Expected error for invalid JSON but got nil")
			}
			if result != nil {
				t.Errorf("Expected nil result for invalid JSON but got: %+v", result)
			}
			// Check for either "invalid json" or schema validation error
			errMsg := err.Error()
			if !strings.Contains(errMsg, "invalid json") && !strings.Contains(errMsg, "jsonschema validation failed") {
				t.Errorf("Expected 'invalid json' or schema validation error but got: %v", err)
			}
		})
	}
}

func TestValidateBytes_SchemaViolations(t *testing.T) {
	validator := createTestValidator(t)

	tests := []struct {
		name     string
		jsonData string
		errorMsg string
	}{
		{
			name: "missing intent_type",
			jsonData: `{
				"target": "my-deployment",
				"namespace": "default",
				"replicas": 3
			}`,
			errorMsg: "intent_type",
		},
		{
			name: "missing target",
			jsonData: `{
				"intent_type": "scaling",
				"namespace": "default",
				"replicas": 3
			}`,
			errorMsg: "target",
		},
		{
			name: "missing namespace",
			jsonData: `{
				"intent_type": "scaling",
				"target": "my-deployment",
				"replicas": 3
			}`,
			errorMsg: "namespace",
		},
		{
			name: "missing replicas",
			jsonData: `{
				"intent_type": "scaling",
				"target": "my-deployment",
				"namespace": "default"
			}`,
			errorMsg: "replicas",
		},
		{
			name: "invalid intent_type",
			jsonData: `{
				"intent_type": "invalid",
				"target": "my-deployment",
				"namespace": "default",
				"replicas": 3
			}`,
			errorMsg: "intent_type",
		},
		{
			name: "empty target",
			jsonData: `{
				"intent_type": "scaling",
				"target": "",
				"namespace": "default",
				"replicas": 3
			}`,
			errorMsg: "target",
		},
		{
			name: "empty namespace",
			jsonData: `{
				"intent_type": "scaling",
				"target": "my-deployment",
				"namespace": "",
				"replicas": 3
			}`,
			errorMsg: "namespace",
		},
		{
			name: "replicas too small",
			jsonData: `{
				"intent_type": "scaling",
				"target": "my-deployment",
				"namespace": "default",
				"replicas": 0
			}`,
			errorMsg: "replicas",
		},
		{
			name: "replicas too large",
			jsonData: `{
				"intent_type": "scaling",
				"target": "my-deployment",
				"namespace": "default",
				"replicas": 101
			}`,
			errorMsg: "replicas",
		},
		{
			name: "negative replicas",
			jsonData: `{
				"intent_type": "scaling",
				"target": "my-deployment",
				"namespace": "default",
				"replicas": -5
			}`,
			errorMsg: "replicas",
		},
		{
			name: "invalid source enum",
			jsonData: `{
				"intent_type": "scaling",
				"target": "my-deployment",
				"namespace": "default",
				"replicas": 3,
				"source": "invalid"
			}`,
			errorMsg: "source",
		},
		{
			name: "reason too long",
			jsonData: `{
				"intent_type": "scaling",
				"target": "my-deployment",
				"namespace": "default",
				"replicas": 3,
				"reason": "` + strings.Repeat("a", 513) + `"
			}`,
			errorMsg: "reason",
		},
		{
			name: "additional properties not allowed",
			jsonData: `{
				"intent_type": "scaling",
				"target": "my-deployment",
				"namespace": "default",
				"replicas": 3,
				"extra_field": "not allowed"
			}`,
			errorMsg: "additional properties",
		},
		{
			name: "wrong type for replicas",
			jsonData: `{
				"intent_type": "scaling",
				"target": "my-deployment",
				"namespace": "default",
				"replicas": "3"
			}`,
			errorMsg: "replicas",
		},
		{
			name: "wrong type for target",
			jsonData: `{
				"intent_type": "scaling",
				"target": 123,
				"namespace": "default",
				"replicas": 3
			}`,
			errorMsg: "target",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidateBytes([]byte(tt.jsonData))
			if err == nil {
				t.Errorf("Expected validation error but got nil")
				return
			}
			if result != nil {
				t.Errorf("Expected nil result for invalid data but got: %+v", result)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorMsg)) {
				t.Errorf("Expected error message to contain '%s' but got: %v", tt.errorMsg, err)
			}
		})
	}
}

func TestValidateBytes_EdgeCases(t *testing.T) {
	validator := createTestValidator(t)

	tests := []struct {
		name        string
		jsonData    string
		expectError bool
		description string
	}{
		{
			name: "null values for optional fields",
			jsonData: `{
				"intent_type": "scaling",
				"target": "my-deployment",
				"namespace": "default",
				"replicas": 3,
				"reason": null,
				"source": null,
				"correlation_id": null
			}`,
			expectError: true,
			description: "null values should not be allowed for typed optional fields",
		},
		{
			name: "very long target name",
			jsonData: `{
				"intent_type": "scaling",
				"target": "` + strings.Repeat("a", 1000) + `",
				"namespace": "default",
				"replicas": 3
			}`,
			expectError: false,
			description: "very long target names should be allowed",
		},
		{
			name: "very long namespace name",
			jsonData: `{
				"intent_type": "scaling",
				"target": "my-deployment",
				"namespace": "` + strings.Repeat("n", 1000) + `",
				"replicas": 3
			}`,
			expectError: false,
			description: "very long namespace names should be allowed",
		},
		{
			name: "very long correlation_id",
			jsonData: `{
				"intent_type": "scaling",
				"target": "my-deployment",
				"namespace": "default",
				"replicas": 3,
				"correlation_id": "` + strings.Repeat("c", 10000) + `"
			}`,
			expectError: false,
			description: "very long correlation_id should be allowed",
		},
		{
			name: "minimum replicas",
			jsonData: `{
				"intent_type": "scaling",
				"target": "my-deployment",
				"namespace": "default",
				"replicas": 1
			}`,
			expectError: false,
			description: "minimum replicas value should be allowed",
		},
		{
			name: "empty reason string",
			jsonData: `{
				"intent_type": "scaling",
				"target": "my-deployment",
				"namespace": "default",
				"replicas": 3,
				"reason": ""
			}`,
			expectError: false,
			description: "empty reason string should be allowed",
		},
		{
			name: "empty correlation_id string",
			jsonData: `{
				"intent_type": "scaling",
				"target": "my-deployment",
				"namespace": "default",
				"replicas": 3,
				"correlation_id": ""
			}`,
			expectError: false,
			description: "empty correlation_id string should be allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidateBytes([]byte(tt.jsonData))
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil: %s", tt.description)
				}
				if result != nil {
					t.Errorf("Expected nil result but got non-nil: %s", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v (%s)", err, tt.description)
				}
				if result == nil {
					t.Errorf("Expected non-nil result but got nil: %s", tt.description)
				}
			}
		})
	}
}

// createTestValidator creates a validator with a test schema for use in tests
func createTestValidator(t *testing.T) *Validator {
	tempDir := t.TempDir()
	schemaDir := filepath.Join(tempDir, "docs", "contracts")
	err := os.MkdirAll(schemaDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp schema dir: %v", err)
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

	err = os.WriteFile(schemaPath, []byte(schema), 0644)
	if err != nil {
		t.Fatalf("Failed to write temp schema file: %v", err)
	}

	validator, err := NewValidator(schemaPath)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	return validator
}

// TestValidateBytes_RealSchema tests the validator using the actual schema file
func TestValidateBytes_RealSchema(t *testing.T) {
	// Try to find the real schema file
	// This test will be skipped if the schema file is not found
	schemaPath := "../../docs/contracts/intent.schema.json"
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		t.Skip("Real schema file not found, skipping real schema test")
	}

	validator, err := NewValidator(schemaPath)
	if err != nil {
		t.Fatalf("Failed to create validator with real schema: %v", err)
	}

	// Test with valid data
	validJSON := `{
		"intent_type": "scaling",
		"target": "test-deployment",
		"namespace": "default",
		"replicas": 5,
		"source": "test",
		"correlation_id": "test-123"
	}`

	result, err := validator.ValidateBytes([]byte(validJSON))
	if err != nil {
		t.Errorf("Expected no error with real schema but got: %v", err)
	}
	if result == nil {
		t.Errorf("Expected non-nil result with real schema")
	}

	// Test with invalid data
	invalidJSON := `{
		"intent_type": "invalid",
		"target": "test-deployment",
		"namespace": "default",
		"replicas": 5
	}`

	result, err = validator.ValidateBytes([]byte(invalidJSON))
	if err == nil {
		t.Errorf("Expected error with invalid data but got nil")
	}
	if result != nil {
		t.Errorf("Expected nil result with invalid data but got: %+v", result)
	}
}

// TestNewValidator_FileSystemErrors tests error handling for filesystem issues
func TestNewValidator_FileSystemErrors(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) string
		expectError string
	}{
		{
			name: "read permission denied",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				schemaFile := filepath.Join(tempDir, "schema.json")
				
				// Create file with valid content first
				content := `{"$schema": "https://json-schema.org/draft/2020-12/schema"}`
				err := os.WriteFile(schemaFile, []byte(content), 0644)
				if err != nil {
					t.Fatalf("Failed to create schema file: %v", err)
				}
				
				// Remove read permissions (simulation - actual effect depends on OS)
				err = os.Chmod(schemaFile, 0000)
				if err != nil {
					t.Skipf("Cannot modify file permissions on this system: %v", err)
				}
				
				return schemaFile
			},
			expectError: "open schema",
		},
		{
			name: "schema file is directory",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				dirPath := filepath.Join(tempDir, "schema.json")
				err := os.Mkdir(dirPath, 0755)
				if err != nil {
					t.Fatalf("Failed to create directory: %v", err)
				}
				return dirPath
			},
			expectError: "open schema",
		},
		{
			name: "corrupted schema file",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				schemaFile := filepath.Join(tempDir, "corrupt.json")
				
				// Write binary data that's not valid JSON
				corruptData := []byte{0xFF, 0xFE, 0xFD, 0xFC, 0xFB}
				err := os.WriteFile(schemaFile, corruptData, 0644)
				if err != nil {
					t.Fatalf("Failed to create corrupt file: %v", err)
				}
				
				return schemaFile
			},
			expectError: "parse schema json",
		},
		{
			name: "empty schema file",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				schemaFile := filepath.Join(tempDir, "empty.json")
				err := os.WriteFile(schemaFile, []byte{}, 0644)
				if err != nil {
					t.Fatalf("Failed to create empty file: %v", err)
				}
				return schemaFile
			},
			expectError: "parse schema json",
		},
		{
			name: "invalid json schema structure",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				schemaFile := filepath.Join(tempDir, "invalid.json")
				
				// Valid JSON but invalid schema
				invalidSchema := `{"not": "a valid schema structure"}`
				err := os.WriteFile(schemaFile, []byte(invalidSchema), 0644)
				if err != nil {
					t.Fatalf("Failed to create invalid schema file: %v", err)
				}
				
				return schemaFile
			},
			expectError: "compile schema",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schemaPath := tt.setupFunc(t)
			
			// Ensure cleanup happens even if test fails
			defer func() {
				// Restore permissions for cleanup
				os.Chmod(schemaPath, 0644)
			}()
			
			validator, err := NewValidator(schemaPath)
			
			if err == nil {
				t.Errorf("Expected error but got nil")
				return
			}
			
			if validator != nil {
				t.Errorf("Expected nil validator but got non-nil")
			}
			
			if !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("Expected error containing '%s' but got: %v", tt.expectError, err)
			}
		})
	}
}

// TestValidateBytes_ExtremeCases tests extreme input cases for validation
func TestValidateBytes_ExtremeCases(t *testing.T) {
	validator := createTestValidator(t)

	tests := []struct {
		name        string
		input       []byte
		expectError bool
		description string
	}{
		{
			name:        "extremely large json payload",
			input:       []byte(`{"intent_type": "scaling", "target": "` + strings.Repeat("a", 1000000) + `", "namespace": "default", "replicas": 3}`),
			expectError: false,
			description: "Large JSON payload should be handled",
		},
		{
			name:        "deeply nested json",
			input:       createDeeplyNestedJSON(t, 100),
			expectError: true,
			description: "Deeply nested JSON should be rejected",
		},
		{
			name:        "unicode in json",
			input:       []byte(`{"intent_type": "scaling", "target": "测试-deployment", "namespace": "default", "replicas": 3}`),
			expectError: false,
			description: "Unicode characters should be allowed",
		},
		{
			name:        "json with null bytes",
			input:       []byte("{\x00\"intent_type\": \"scaling\", \"target\": \"test\", \"namespace\": \"default\", \"replicas\": 3}"),
			expectError: true,
			description: "JSON with null bytes should be rejected",
		},
		{
			name:        "json with control characters",
			input:       []byte("{\"intent_type\": \"scaling\", \"target\": \"test\x01\x02\", \"namespace\": \"default\", \"replicas\": 3}"),
			expectError: true,
			description: "JSON with control characters should be rejected as invalid JSON",
		},
		{
			name:        "maximum integer value",
			input:       []byte(`{"intent_type": "scaling", "target": "test", "namespace": "default", "replicas": 9223372036854775807}`),
			expectError: true,
			description: "Maximum integer value should exceed schema limits",
		},
		{
			name:        "negative zero replicas",
			input:       []byte(`{"intent_type": "scaling", "target": "test", "namespace": "default", "replicas": -0}`),
			expectError: true,
			description: "Negative zero should be treated as zero and fail validation",
		},
		{
			name:        "floating point replicas",
			input:       []byte(`{"intent_type": "scaling", "target": "test", "namespace": "default", "replicas": 3.14}`),
			expectError: true,
			description: "Floating point replicas should be rejected",
		},
		{
			name:        "scientific notation replicas",
			input:       []byte(`{"intent_type": "scaling", "target": "test", "namespace": "default", "replicas": 1e2}`),
			expectError: true,
			description: "Scientific notation replicas should be rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.ValidateBytes(tt.input)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil: %s", tt.description)
				}
				if result != nil {
					t.Errorf("Expected nil result but got non-nil: %s", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v (%s)", err, tt.description)
				}
				if result == nil {
					t.Errorf("Expected non-nil result but got nil: %s", tt.description)
				}
			}
		})
	}
}

// TestValidateBytes_MemoryExhaustion tests memory handling with large inputs
func TestValidateBytes_MemoryExhaustion(t *testing.T) {
	validator := createTestValidator(t)

	// Test with extremely large JSON array
	largeArrayJSON := `{"intent_type": "scaling", "target": "test", "namespace": "default", "replicas": 3, "large_array": [`
	for i := 0; i < 10000; i++ {
		if i > 0 {
			largeArrayJSON += ","
		}
		largeArrayJSON += `"item` + fmt.Sprintf("%d", i) + `"`
	}
	largeArrayJSON += `]}`

	result, err := validator.ValidateBytes([]byte(largeArrayJSON))
	// This should fail schema validation due to additional properties
	if err == nil {
		t.Errorf("Expected schema validation error for large array JSON")
	}
	if result != nil {
		t.Errorf("Expected nil result for invalid schema")
	}
}

// TestValidateBytes_ConcurrentAccess tests thread safety of validator
func TestValidateBytes_ConcurrentAccess(t *testing.T) {
	validator := createTestValidator(t)
	
	validJSON := `{
		"intent_type": "scaling",
		"target": "concurrent-test",
		"namespace": "default",
		"replicas": 3
	}`
	
	invalidJSON := `{
		"intent_type": "invalid",
		"target": "concurrent-test",
		"namespace": "default",
		"replicas": 3
	}`
	
	const numGoroutines = 50
	const numIterations = 10
	
	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines*numIterations)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			for j := 0; j < numIterations; j++ {
				var input []byte
				var expectError bool
				
				if (id+j)%2 == 0 {
					input = []byte(validJSON)
					expectError = false
				} else {
					input = []byte(invalidJSON)
					expectError = true
				}
				
				result, err := validator.ValidateBytes(input)
				
				if expectError {
					if err == nil {
						errors <- fmt.Errorf("goroutine %d iteration %d: expected error but got nil", id, j)
					}
					if result != nil {
						errors <- fmt.Errorf("goroutine %d iteration %d: expected nil result but got non-nil", id, j)
					}
				} else {
					if err != nil {
						errors <- fmt.Errorf("goroutine %d iteration %d: expected no error but got: %v", id, j, err)
					}
					if result == nil {
						errors <- fmt.Errorf("goroutine %d iteration %d: expected non-nil result but got nil", id, j)
					}
				}
			}
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
	
	// Check for any errors
	close(errors)
	for err := range errors {
		t.Error(err)
	}
}

// Helper function to create deeply nested JSON for testing
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