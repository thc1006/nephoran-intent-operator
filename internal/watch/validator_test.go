package watch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestValidator(t *testing.T) {
	// Create temporary schema file
	tempDir := t.TempDir()
	schemaPath := filepath.Join(tempDir, "test-schema.json")
	
	schemaContent := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["intent_type", "target", "namespace", "replicas"],
		"properties": {
			"intent_type": {"const": "scaling"},
			"target": {"type": "string", "minLength": 1},
			"namespace": {"type": "string", "minLength": 1},
			"replicas": {"type": "integer", "minimum": 1, "maximum": 100},
			"reason": {"type": "string"},
			"source": {"type": "string", "enum": ["user", "planner", "test"]},
			"correlation_id": {"type": "string"}
		},
		"additionalProperties": false
	}`
	
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatalf("Failed to write test schema: %v", err)
	}

	// Create validator
	validator, err := NewValidator(schemaPath)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	tests := []struct {
		name      string
		json      string
		wantError bool
	}{
		{
			name: "valid intent",
			json: `{
				"intent_type": "scaling",
				"target": "my-deployment",
				"namespace": "default",
				"replicas": 5,
				"source": "test"
			}`,
			wantError: false,
		},
		{
			name: "missing required field",
			json: `{
				"intent_type": "scaling",
				"target": "my-deployment",
				"replicas": 5
			}`,
			wantError: true,
		},
		{
			name: "wrong intent type",
			json: `{
				"intent_type": "update",
				"target": "my-deployment",
				"namespace": "default",
				"replicas": 5
			}`,
			wantError: true,
		},
		{
			name: "replicas too high",
			json: `{
				"intent_type": "scaling",
				"target": "my-deployment",
				"namespace": "default",
				"replicas": 200
			}`,
			wantError: true,
		},
		{
			name: "invalid source enum",
			json: `{
				"intent_type": "scaling",
				"target": "my-deployment",
				"namespace": "default",
				"replicas": 5,
				"source": "invalid"
			}`,
			wantError: true,
		},
		{
			name: "additional properties not allowed",
			json: `{
				"intent_type": "scaling",
				"target": "my-deployment",
				"namespace": "default",
				"replicas": 5,
				"extra_field": "not allowed"
			}`,
			wantError: true,
		},
		{
			name:      "invalid JSON",
			json:      `{invalid json}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate([]byte(tt.json))
			hasError := err != nil
			if hasError != tt.wantError {
				t.Errorf("Validate() error = %v, wantError = %v", err, tt.wantError)
			}
		})
	}
}

func TestValidatorReload(t *testing.T) {
	// Create temporary schema file
	tempDir := t.TempDir()
	schemaPath := filepath.Join(tempDir, "reload-schema.json")
	
	// Initial schema
	initialSchema := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["name"],
		"properties": {
			"name": {"type": "string"}
		}
	}`
	
	if err := os.WriteFile(schemaPath, []byte(initialSchema), 0644); err != nil {
		t.Fatalf("Failed to write initial schema: %v", err)
	}

	// Create validator
	validator, err := NewValidator(schemaPath)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Test with initial schema
	validJSON := `{"name": "test"}`
	if err := validator.Validate([]byte(validJSON)); err != nil {
		t.Errorf("Initial validation failed: %v", err)
	}

	// Update schema
	updatedSchema := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["name", "age"],
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		}
	}`
	
	if err := os.WriteFile(schemaPath, []byte(updatedSchema), 0644); err != nil {
		t.Fatalf("Failed to write updated schema: %v", err)
	}

	// Reload schema
	if err := validator.ReloadSchema(); err != nil {
		t.Fatalf("Failed to reload schema: %v", err)
	}

	// Previous valid JSON should now be invalid
	if err := validator.Validate([]byte(validJSON)); err == nil {
		t.Error("Expected validation to fail after schema reload")
	}

	// New valid JSON
	newValidJSON := `{"name": "test", "age": 25}`
	if err := validator.Validate([]byte(newValidJSON)); err != nil {
		t.Errorf("Validation failed after reload: %v", err)
	}
}

func TestValidatorConcurrency(t *testing.T) {
	// Create temporary schema file
	tempDir := t.TempDir()
	schemaPath := filepath.Join(tempDir, "concurrent-schema.json")
	
	schemaContent := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["value"],
		"properties": {
			"value": {"type": "integer"}
		}
	}`
	
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatalf("Failed to write schema: %v", err)
	}

	validator, err := NewValidator(schemaPath)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Run concurrent validations
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			jsonData, _ := json.Marshal(map[string]int{"value": id})
			if err := validator.Validate(jsonData); err != nil {
				t.Errorf("Concurrent validation %d failed: %v", id, err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}