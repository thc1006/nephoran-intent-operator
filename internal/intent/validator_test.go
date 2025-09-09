package intent

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestNewValidator(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	contractsDir := filepath.Join(tempDir, "docs", "contracts")
<<<<<<< HEAD
	if err := os.MkdirAll(contractsDir, 0o755); err != nil {
=======
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		t.Fatalf("Failed to create contracts directory: %v", err)
	}

	// Test valid schema
	t.Run("ValidSchema", func(t *testing.T) {
		validSchema := `{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"$id": "https://example.com/schemas/intent.schema.json",
			"title": "ScalingIntent",
			"type": "object",
			"additionalProperties": false,
			"required": ["intent_type", "target", "namespace", "replicas"],
			"properties": {
				"intent_type": {
<<<<<<< HEAD
					"const": "scaling",
=======
					"type": "string",
					"enum": ["scaling", "deployment", "configuration"],
>>>>>>> 6835433495e87288b95961af7173d866977175ff
					"description": "MVP supports only scaling"
				},
				"target": {
					"type": "string",
					"minLength": 1,
					"description": "Kubernetes Deployment name (CNF simulator)"
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
					"type": "string",
					"description": "Optional id for tracing"
				}
			}
		}`

		schemaPath := filepath.Join(contractsDir, "intent.schema.json")
<<<<<<< HEAD
		if err := os.WriteFile(schemaPath, []byte(validSchema), 0o644); err != nil {
=======
		if err := os.WriteFile(schemaPath, []byte(validSchema), 0644); err != nil {
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			t.Fatalf("Failed to write schema file: %v", err)
		}

		validator, err := NewValidator(tempDir)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if validator == nil {
			t.Fatal("Expected validator to be non-nil")
		}

		if validator.schemaLoader == nil {
			t.Fatal("Expected compiled schema to be non-nil")
		}

		if !validator.IsHealthy() {
			t.Fatal("Expected validator to be healthy")
		}

		// Check metrics initialization
		metrics := validator.GetMetrics()
		if metrics.TotalValidations != 0 {
			t.Errorf("Expected 0 total validations, got %d", metrics.TotalValidations)
		}

		// Check schema info
		info := validator.GetSchemaInfo()
		if info["has_schema"] != true {
			t.Error("Expected has_schema to be true")
		}
	})

	// Test missing schema file (hard error)
	t.Run("MissingSchemaFile", func(t *testing.T) {
		missingDir := filepath.Join(tempDir, "missing")
		validator, err := NewValidator(missingDir)
		if err == nil {
			t.Fatal("Expected error for missing schema file")
		}
		if validator != nil {
			t.Fatal("Expected validator to be nil on error")
		}
	})

	// Test invalid JSON schema (hard error)
	t.Run("InvalidJSON", func(t *testing.T) {
		invalidJSON := `{"invalid": json}`
		schemaPath := filepath.Join(contractsDir, "intent.schema.json")
<<<<<<< HEAD
		if err := os.WriteFile(schemaPath, []byte(invalidJSON), 0o644); err != nil {
=======
		if err := os.WriteFile(schemaPath, []byte(invalidJSON), 0644); err != nil {
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			t.Fatalf("Failed to write invalid schema: %v", err)
		}

		validator, err := NewValidator(tempDir)
		if err == nil {
			t.Fatal("Expected error for invalid JSON")
		}
		if validator != nil {
			t.Fatal("Expected validator to be nil on error")
		}
	})

	// Test invalid schema compilation (hard error)
	t.Run("InvalidSchemaCompilation", func(t *testing.T) {
		invalidSchema := `{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type": "invalid-type"
		}`
		schemaPath := filepath.Join(contractsDir, "intent.schema.json")
<<<<<<< HEAD
		if err := os.WriteFile(schemaPath, []byte(invalidSchema), 0o644); err != nil {
=======
		if err := os.WriteFile(schemaPath, []byte(invalidSchema), 0644); err != nil {
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			t.Fatalf("Failed to write invalid schema: %v", err)
		}

		validator, err := NewValidator(tempDir)
		if err == nil {
			t.Fatal("Expected error for schema compilation failure")
		}
		if validator != nil {
			t.Fatal("Expected validator to be nil on error")
		}
	})
}

func TestValidatorValidateIntent(t *testing.T) {
	// Setup validator with valid schema
	tempDir := t.TempDir()
	contractsDir := filepath.Join(tempDir, "docs", "contracts")
<<<<<<< HEAD
	if err := os.MkdirAll(contractsDir, 0o755); err != nil {
=======
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		t.Fatalf("Failed to create contracts directory: %v", err)
	}

	validSchema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"$id": "https://example.com/schemas/intent.schema.json",
		"title": "ScalingIntent",
		"type": "object",
		"additionalProperties": false,
		"required": ["intent_type", "target", "namespace", "replicas"],
		"properties": {
			"intent_type": {
<<<<<<< HEAD
				"const": "scaling",
				"description": "MVP supports only scaling"
=======
				"type": "string",
				"enum": ["scaling", "deployment", "configuration"],
				"description": "MVP supports scaling, deployment, and configuration"
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			},
			"target": {
				"type": "string",
				"minLength": 1,
				"description": "Kubernetes Deployment name (CNF simulator)"
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
				"type": "string",
				"description": "Optional id for tracing"
			}
		}
	}`

	schemaPath := filepath.Join(contractsDir, "intent.schema.json")
<<<<<<< HEAD
	if err := os.WriteFile(schemaPath, []byte(validSchema), 0o644); err != nil {
=======
	if err := os.WriteFile(schemaPath, []byte(validSchema), 0644); err != nil {
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		t.Fatalf("Failed to write schema file: %v", err)
	}

	validator, err := NewValidator(tempDir)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Test valid intent
	t.Run("ValidIntent", func(t *testing.T) {
		intent := &ScalingIntent{
			IntentType:    "scaling",
			Target:        "test-deployment",
			Namespace:     "test-namespace",
			Replicas:      5,
			Reason:        "Load increased",
			Source:        "user",
			CorrelationID: "test-123",
		}

		errors := validator.ValidateIntent(intent)
		if len(errors) > 0 {
			t.Errorf("Expected no validation errors, got: %v", errors)
		}

		// Check metrics
		metrics := validator.GetMetrics()
		if metrics.ValidationSuccesses == 0 {
			t.Error("Expected validation success to be recorded")
		}
		if metrics.TotalValidations == 0 {
			t.Error("Expected total validations to be recorded")
		}
	})

	// Test invalid intent type
	t.Run("InvalidIntentType", func(t *testing.T) {
		intent := &ScalingIntent{
			IntentType: "invalid",
			Target:     "test-deployment",
			Namespace:  "test-namespace",
			Replicas:   5,
		}

		errors := validator.ValidateIntent(intent)
		if len(errors) == 0 {
			t.Error("Expected validation errors for invalid intent type")
		}

		// Check metrics
		metrics := validator.GetMetrics()
		if metrics.ValidationErrors == 0 {
			t.Error("Expected validation error to be recorded")
		}
	})

	// Test missing required fields
	t.Run("MissingRequiredFields", func(t *testing.T) {
		intent := &ScalingIntent{
			IntentType: "scaling",
			// Missing target, namespace, replicas
		}

		errors := validator.ValidateIntent(intent)
		if len(errors) == 0 {
			t.Error("Expected validation errors for missing required fields")
		}

		// Should have errors for target, namespace, replicas
		fieldErrors := make(map[string]bool)
		for _, err := range errors {
			fieldErrors[err.Field] = true
			t.Logf("Got validation error: field=%s, message=%s", err.Field, err.Message)
		}

		expectedFields := []string{"target", "namespace", "replicas"}
		for _, field := range expectedFields {
			if !fieldErrors[field] {
				t.Errorf("Expected validation error for field %s", field)
			}
		}
	})

	// Test replicas out of range
	t.Run("ReplicasOutOfRange", func(t *testing.T) {
		intent := &ScalingIntent{
			IntentType: "scaling",
			Target:     "test-deployment",
			Namespace:  "test-namespace",
			Replicas:   101, // Exceeds maximum
		}

		errors := validator.ValidateIntent(intent)
		if len(errors) == 0 {
			t.Error("Expected validation errors for replicas out of range")
		}
	})

	// Test invalid source enum
	t.Run("InvalidSource", func(t *testing.T) {
		intent := &ScalingIntent{
			IntentType: "scaling",
			Target:     "test-deployment",
			Namespace:  "test-namespace",
			Replicas:   5,
			Source:     "invalid-source",
		}

		errors := validator.ValidateIntent(intent)
		if len(errors) == 0 {
			t.Error("Expected validation errors for invalid source")
		}
	})

	// Test reason too long
	t.Run("ReasonTooLong", func(t *testing.T) {
		longReason := make([]byte, 513) // Exceeds 512 character limit
		for i := range longReason {
			longReason[i] = 'a'
		}

		intent := &ScalingIntent{
			IntentType: "scaling",
			Target:     "test-deployment",
			Namespace:  "test-namespace",
			Replicas:   5,
			Reason:     string(longReason),
		}

		errors := validator.ValidateIntent(intent)
		if len(errors) == 0 {
			t.Error("Expected validation errors for reason too long")
		}
	})
}

func TestValidatorValidateJSON(t *testing.T) {
	// Setup validator
	tempDir := t.TempDir()
	contractsDir := filepath.Join(tempDir, "docs", "contracts")
<<<<<<< HEAD
	if err := os.MkdirAll(contractsDir, 0o755); err != nil {
=======
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		t.Fatalf("Failed to create contracts directory: %v", err)
	}

	validSchema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"$id": "https://example.com/schemas/intent.schema.json",
		"title": "ScalingIntent",
		"type": "object",
		"additionalProperties": false,
		"required": ["intent_type", "target", "namespace", "replicas"],
		"properties": {
			"intent_type": {
<<<<<<< HEAD
				"const": "scaling",
				"description": "MVP supports only scaling"
=======
				"type": "string",
				"enum": ["scaling", "deployment", "configuration"],
				"description": "MVP supports scaling, deployment, and configuration"
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			},
			"target": {
				"type": "string",
				"minLength": 1,
				"description": "Kubernetes Deployment name"
			},
			"namespace": {
				"type": "string",
				"minLength": 1
			},
			"replicas": {
				"type": "integer",
				"minimum": 1,
				"maximum": 100
			}
		}
	}`

	schemaPath := filepath.Join(contractsDir, "intent.schema.json")
<<<<<<< HEAD
	if err := os.WriteFile(schemaPath, []byte(validSchema), 0o644); err != nil {
=======
	if err := os.WriteFile(schemaPath, []byte(validSchema), 0644); err != nil {
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		t.Fatalf("Failed to write schema file: %v", err)
	}

	validator, err := NewValidator(tempDir)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Test valid JSON
	t.Run("ValidJSON", func(t *testing.T) {
		validJSON := `{
			"intent_type": "scaling",
			"target": "test-deployment",
			"namespace": "test-namespace",
			"replicas": 5
		}`

		errors := validator.ValidateJSON([]byte(validJSON))
		if len(errors) > 0 {
			t.Errorf("Expected no validation errors, got: %v", errors)
		}
	})

	// Test invalid JSON syntax
	t.Run("InvalidJSONSyntax", func(t *testing.T) {
		invalidJSON := `{invalid json}`

		errors := validator.ValidateJSON([]byte(invalidJSON))
		if len(errors) == 0 {
			t.Error("Expected validation errors for invalid JSON syntax")
		}

		if errors[0].Field != "json" {
			t.Errorf("Expected field to be 'json', got '%s'", errors[0].Field)
		}
	})

	// Test JSON that doesn't match schema
	t.Run("JSONNotMatchingSchema", func(t *testing.T) {
		invalidJSON := `{
			"intent_type": "invalid",
			"target": "test-deployment",
			"namespace": "test-namespace",
			"replicas": 5
		}`

		errors := validator.ValidateJSON([]byte(invalidJSON))
		if len(errors) == 0 {
			t.Error("Expected validation errors for JSON not matching schema")
		}
	})
}

func TestValidatorMetrics(t *testing.T) {
	// Setup validator
	tempDir := t.TempDir()
	contractsDir := filepath.Join(tempDir, "docs", "contracts")
<<<<<<< HEAD
	if err := os.MkdirAll(contractsDir, 0o755); err != nil {
=======
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		t.Fatalf("Failed to create contracts directory: %v", err)
	}

	validSchema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"$id": "https://example.com/schemas/intent.schema.json",
		"title": "ScalingIntent",
		"type": "object",
		"additionalProperties": false,
		"required": ["intent_type", "target", "namespace", "replicas"],
		"properties": {
			"intent_type": {
<<<<<<< HEAD
				"const": "scaling",
				"description": "MVP supports only scaling"
=======
				"type": "string",
				"enum": ["scaling", "deployment", "configuration"],
				"description": "MVP supports scaling, deployment, and configuration"
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			},
			"target": {
				"type": "string",
				"minLength": 1,
				"description": "Kubernetes Deployment name"
			},
			"namespace": {
				"type": "string",
				"minLength": 1
			},
			"replicas": {
				"type": "integer",
				"minimum": 1,
				"maximum": 100
			}
		}
	}`

	schemaPath := filepath.Join(contractsDir, "intent.schema.json")
<<<<<<< HEAD
	if err := os.WriteFile(schemaPath, []byte(validSchema), 0o644); err != nil {
=======
	if err := os.WriteFile(schemaPath, []byte(validSchema), 0644); err != nil {
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		t.Fatalf("Failed to write schema file: %v", err)
	}

	validator, err := NewValidator(tempDir)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Test metrics tracking
	validIntent := &ScalingIntent{
		IntentType: "scaling",
		Target:     "test-deployment",
		Namespace:  "test-namespace",
		Replicas:   5,
	}

	invalidIntent := &ScalingIntent{
		IntentType: "invalid",
		Target:     "test-deployment",
		Namespace:  "test-namespace",
		Replicas:   5,
	}

	// Perform some validations
	validator.ValidateIntent(validIntent)   // Should succeed
	validator.ValidateIntent(invalidIntent) // Should fail

	metrics := validator.GetMetrics()

	if metrics.TotalValidations != 2 {
		t.Errorf("Expected 2 total validations, got %d", metrics.TotalValidations)
	}

	if metrics.ValidationSuccesses != 1 {
		t.Errorf("Expected 1 validation success, got %d", metrics.ValidationSuccesses)
	}

	if metrics.ValidationErrors != 1 {
		t.Errorf("Expected 1 validation error, got %d", metrics.ValidationErrors)
	}
}

func TestValidatorLogging(t *testing.T) {
	// Use a custom logger to capture logs
	var logOutput []string
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// This test is more about ensuring logging doesn't crash
	// Full log capture would require a custom handler
	_ = logger

	// Test that logging doesn't cause panics
	tempDir := t.TempDir()
	contractsDir := filepath.Join(tempDir, "docs", "contracts")
<<<<<<< HEAD
	if err := os.MkdirAll(contractsDir, 0o755); err != nil {
=======
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		t.Fatalf("Failed to create contracts directory: %v", err)
	}

	validSchema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"$id": "https://example.com/schemas/intent.schema.json",
		"type": "object",
		"required": ["intent_type"],
		"properties": {
<<<<<<< HEAD
			"intent_type": {"const": "scaling"}
=======
			"intent_type": {
				"type": "string",
				"enum": ["scaling", "deployment", "configuration"]
			}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		}
	}`

	schemaPath := filepath.Join(contractsDir, "intent.schema.json")
<<<<<<< HEAD
	if err := os.WriteFile(schemaPath, []byte(validSchema), 0o644); err != nil {
=======
	if err := os.WriteFile(schemaPath, []byte(validSchema), 0644); err != nil {
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		t.Fatalf("Failed to write schema file: %v", err)
	}

	validator, err := NewValidator(tempDir)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// These should not panic
	validator.ValidateJSON([]byte(`{"intent_type": "scaling"}`))
	validator.ValidateJSON([]byte(`{"intent_type": "invalid"}`))
	validator.ValidateJSON([]byte(`{invalid json}`))

	_ = logOutput // Prevent unused variable error
}

// TestSchemaValidationErrorHandling tests comprehensive error scenarios for schema validation
func TestSchemaValidationErrorHandling(t *testing.T) {
	// Setup validator with valid schema
	tempDir := t.TempDir()
	contractsDir := filepath.Join(tempDir, "docs", "contracts")
<<<<<<< HEAD
	if err := os.MkdirAll(contractsDir, 0o755); err != nil {
=======
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		t.Fatalf("Failed to create contracts directory: %v", err)
	}

	validSchema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"$id": "https://example.com/schemas/intent.schema.json",
		"title": "ScalingIntent",
		"type": "object",
		"additionalProperties": false,
		"required": ["intent_type", "target", "namespace", "replicas"],
		"properties": {
			"intent_type": {
<<<<<<< HEAD
				"const": "scaling",
				"description": "MVP supports only scaling"
=======
				"type": "string",
				"enum": ["scaling", "deployment", "configuration"],
				"description": "MVP supports scaling, deployment, and configuration"
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			},
			"target": {
				"type": "string",
				"minLength": 1,
				"maxLength": 63,
				"pattern": "^[a-z0-9]([a-z0-9-]*[a-z0-9])?$",
				"description": "Kubernetes Deployment name (CNF simulator)"
			},
			"namespace": {
				"type": "string",
				"minLength": 1,
				"maxLength": 63,
				"pattern": "^[a-z0-9]([a-z0-9-]*[a-z0-9])?$"
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
				"type": "string",
				"maxLength": 63,
				"description": "Optional id for tracing"
			}
		}
	}`

	schemaPath := filepath.Join(contractsDir, "intent.schema.json")
<<<<<<< HEAD
	if err := os.WriteFile(schemaPath, []byte(validSchema), 0o644); err != nil {
=======
	if err := os.WriteFile(schemaPath, []byte(validSchema), 0644); err != nil {
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		t.Fatalf("Failed to write schema file: %v", err)
	}

	validator, err := NewValidator(tempDir)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	tests := []struct {
		name           string
		input          interface{}
		jsonInput      string
		expectedErrors []string // Expected field names to have errors
		isJSONTest     bool     // Whether to test JSON input directly
	}{
		{
			name: "NullIntentType",
			input: &ScalingIntent{
				IntentType: "", // Empty string, but required
				Target:     "test-deployment",
				Namespace:  "test-namespace",
				Replicas:   5,
			},
			expectedErrors: []string{"intent_type"},
		},
		{
<<<<<<< HEAD
			name:  "MissingRequiredFields",
=======
			name: "MissingRequiredFields",
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			input: &ScalingIntent{
				// All required fields missing
			},
			expectedErrors: []string{"intent_type", "target", "namespace", "replicas"},
		},
		{
			name: "InvalidFieldTypes",
			jsonInput: `{
				"intent_type": "scaling",
				"target": 123,
				"namespace": true,
				"replicas": "not-a-number"
			}`,
			expectedErrors: []string{"target", "namespace", "replicas"},
			isJSONTest:     true,
		},
		{
			name: "EmptyStrings",
			input: &ScalingIntent{
				IntentType: "scaling",
				Target:     "", // Empty string violates minLength
				Namespace:  "", // Empty string violates minLength
				Replicas:   5,
			},
			expectedErrors: []string{"target", "namespace"},
		},
		{
			name: "NegativeReplicas",
			input: &ScalingIntent{
				IntentType: "scaling",
				Target:     "test-deployment",
				Namespace:  "test-namespace",
				Replicas:   -1, // Below minimum
			},
			expectedErrors: []string{"replicas"},
		},
		{
			name: "ZeroReplicas",
			input: &ScalingIntent{
				IntentType: "scaling",
				Target:     "test-deployment",
				Namespace:  "test-namespace",
				Replicas:   0, // Below minimum
			},
			expectedErrors: []string{"replicas"},
		},
		{
			name: "ExcessiveReplicas",
			input: &ScalingIntent{
				IntentType: "scaling",
				Target:     "test-deployment",
				Namespace:  "test-namespace",
				Replicas:   101, // Above maximum
			},
			expectedErrors: []string{"replicas"},
		},
		{
			name: "ExtremelyLargeReplicas",
			input: &ScalingIntent{
				IntentType: "scaling",
				Target:     "test-deployment",
				Namespace:  "test-namespace",
				Replicas:   999999, // Extremely large value
			},
			expectedErrors: []string{"replicas"},
		},
		{
			name: "InvalidTargetPattern",
			input: &ScalingIntent{
				IntentType: "scaling",
				Target:     "Invalid-Target_Name!", // Violates Kubernetes naming
				Namespace:  "test-namespace",
				Replicas:   5,
			},
			expectedErrors: []string{"target"},
		},
		{
			name: "InvalidNamespacePattern",
			input: &ScalingIntent{
				IntentType: "scaling",
				Target:     "test-deployment",
				Namespace:  "Invalid_Namespace!", // Violates Kubernetes naming
				Replicas:   5,
			},
			expectedErrors: []string{"namespace"},
		},
		{
			name: "TooLongTarget",
			input: &ScalingIntent{
				IntentType: "scaling",
				Target:     "this-is-a-very-long-target-name-that-exceeds-kubernetes-limits-x", // 64 chars, > 63 limit
				Namespace:  "test-namespace",
				Replicas:   5,
			},
			expectedErrors: []string{"target"},
		},
		{
			name: "TooLongReason",
			input: &ScalingIntent{
				IntentType: "scaling",
				Target:     "test-deployment",
				Namespace:  "test-namespace",
				Replicas:   5,
				Reason:     string(make([]byte, 513)), // > 512 chars
			},
			expectedErrors: []string{"reason"},
		},
		{
			name: "InvalidSourceEnum",
			input: &ScalingIntent{
				IntentType: "scaling",
				Target:     "test-deployment",
				Namespace:  "test-namespace",
				Replicas:   5,
				Source:     "invalid-source", // Not in enum
			},
			expectedErrors: []string{"source"},
		},
		{
			name: "AdditionalPropertiesNotAllowed",
			jsonInput: `{
				"intent_type": "scaling",
				"target": "test-deployment",
				"namespace": "test-namespace",
				"replicas": 5,
				"extra_field": "not-allowed"
			}`,
			expectedErrors: []string{"/"},
			isJSONTest:     true,
		},
		{
			name: "NullValues",
			jsonInput: `{
				"intent_type": null,
				"target": null,
				"namespace": null,
				"replicas": null
			}`,
			expectedErrors: []string{"intent_type", "target", "namespace", "replicas"},
			isJSONTest:     true,
		},
		{
			name: "MixedValidAndInvalidFields",
			input: &ScalingIntent{
<<<<<<< HEAD
				IntentType: "scaling",         // Valid
				Target:     "test-deployment", // Valid
				Namespace:  "",                // Invalid - empty
				Replicas:   101,               // Invalid - too large
				Source:     "invalid",         // Invalid - not in enum
=======
				IntentType: "scaling", // Valid
				Target:     "test-deployment", // Valid
				Namespace:  "", // Invalid - empty
				Replicas:   101, // Invalid - too large
				Source:     "invalid", // Invalid - not in enum
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			},
			expectedErrors: []string{"namespace", "replicas", "source"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var errors []ValidationError

			if tt.isJSONTest {
				errors = validator.ValidateJSON([]byte(tt.jsonInput))
			} else {
				if intent, ok := tt.input.(*ScalingIntent); ok {
					// Fill in long reason if needed
					if len(tt.expectedErrors) > 0 && tt.name == "TooLongReason" {
						longReason := make([]byte, 513)
						for i := range longReason {
							longReason[i] = 'a'
						}
						intent.Reason = string(longReason)
					}
					errors = validator.ValidateIntent(intent)
				}
			}

			if len(errors) == 0 {
				t.Errorf("Expected validation errors for %s, but got none", tt.name)
				return
			}

			// Check that expected error fields are present
			errorFields := make(map[string]bool)
			for _, err := range errors {
				errorFields[err.Field] = true
				t.Logf("Got validation error: field=%s, message=%s, value=%v", err.Field, err.Message, err.Value)
			}

			for _, expectedField := range tt.expectedErrors {
				if !errorFields[expectedField] {
					t.Errorf("Expected validation error for field '%s' but didn't find it. Got errors for: %v", expectedField, getKeys(errorFields))
				}
			}

			// Ensure metrics were updated
			metrics := validator.GetMetrics()
			if metrics.ValidationErrors == 0 {
				t.Error("Expected validation error metrics to be updated")
			}
		})
	}
}

// TestMalformedJSONInputs tests various malformed JSON scenarios
func TestMalformedJSONInputs(t *testing.T) {
	// Setup validator
	tempDir := t.TempDir()
	contractsDir := filepath.Join(tempDir, "docs", "contracts")
<<<<<<< HEAD
	if err := os.MkdirAll(contractsDir, 0o755); err != nil {
=======
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		t.Fatalf("Failed to create contracts directory: %v", err)
	}

	validSchema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["intent_type"],
		"properties": {
<<<<<<< HEAD
			"intent_type": {"const": "scaling"}
=======
			"intent_type": {
				"type": "string",
				"enum": ["scaling", "deployment", "configuration"]
			}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		}
	}`

	schemaPath := filepath.Join(contractsDir, "intent.schema.json")
<<<<<<< HEAD
	if err := os.WriteFile(schemaPath, []byte(validSchema), 0o644); err != nil {
=======
	if err := os.WriteFile(schemaPath, []byte(validSchema), 0644); err != nil {
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		t.Fatalf("Failed to write schema file: %v", err)
	}

	validator, err := NewValidator(tempDir)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	tests := []struct {
		name      string
		jsonInput string
	}{
		{"UnclosedBrace", `{"intent_type": "scaling"`},
		{"UnclosedQuote", `{"intent_type: "scaling"}`},
		{"TrailingComma", `{"intent_type": "scaling",}`},
		{"InvalidEscape", `{"intent_type": "scal\ing"}`},
		{"UnquotedKey", `{intent_type: "scaling"}`},
		{"SingleQuotes", `{'intent_type': 'scaling'}`},
		{"EmptyString", ``},
		{"OnlyWhitespace", `   \t\n  `},
		{"InvalidUnicode", `{"intent_type": "\uZZZZ"}`},
		{"ControlCharacters", `{"intent_type": "scaling\x00"}`},
		{"DoubleComma", `{"intent_type": "scaling",, "target": "test"}`},
		{"MissingColon", `{"intent_type" "scaling"}`},
		{"ExtraClosingBrace", `{"intent_type": "scaling"}}`},
		{"ArrayInsteadOfObject", `["intent_type", "scaling"]`},
		{"StringInsteadOfObject", `"not an object"`},
		{"NumberInsteadOfObject", `42`},
		{"BooleanInsteadOfObject", `true`},
		{"NullInsteadOfObject", `null`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validator.ValidateJSON([]byte(tt.jsonInput))
			if len(errors) == 0 {
				t.Errorf("Expected validation errors for malformed JSON %s, but got none", tt.name)
				return
			}

<<<<<<< HEAD
			// Should have at least one error - either JSON parsing error (field "json")
=======
			// Should have at least one error - either JSON parsing error (field "json") 
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			// or schema validation error (field "/" for root-level type mismatch)
			found := false
			for _, err := range errors {
				if err.Field == "json" || err.Field == "/" {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected validation error for input %s", tt.name)
			}

			// Log the error for debugging
			for _, err := range errors {
				t.Logf("Validation error for %s: field=%s, message=%s", tt.name, err.Field, err.Message)
			}
		})
	}
}

// TestSchemaFileCorruption tests various schema file corruption scenarios
func TestSchemaFileCorruption(t *testing.T) {
	tests := []struct {
<<<<<<< HEAD
		name          string
		schemaContent string
		shouldFail    bool
=======
		name         string
		schemaContent string
		shouldFail   bool
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	}{
		{
			name: "MissingSchemaVersion",
			schemaContent: `{
				"type": "object",
<<<<<<< HEAD
				"properties": {"intent_type": {"const": "scaling"}}
=======
				"properties": {
					"intent_type": {
						"type": "string", 
						"enum": ["scaling", "deployment", "configuration"]
					}
				}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			}`,
			shouldFail: false, // This might still be valid
		},
		{
			name: "InvalidSchemaType",
			schemaContent: `{
				"$schema": "http://json-schema.org/draft-07/schema#",
				"type": "invalid-type"
			}`,
			shouldFail: true,
		},
		{
			name: "CircularReference",
			schemaContent: `{
				"$schema": "http://json-schema.org/draft-07/schema#",
				"type": "object",
				"properties": {
					"self": {"$ref": "#"}
				}
			}`,
			shouldFail: false, // Circular refs might be valid in JSON Schema
		},
		{
			name: "InvalidPropertyType",
			schemaContent: `{
				"$schema": "http://json-schema.org/draft-07/schema#",
				"type": "object",
				"properties": {
					"intent_type": {
						"type": "invalid-property-type"
					}
				}
			}`,
			shouldFail: true,
		},
		{
			name: "InvalidMinimum",
			schemaContent: `{
				"$schema": "http://json-schema.org/draft-07/schema#",
				"type": "object",
				"properties": {
					"replicas": {
						"type": "integer",
						"minimum": "not-a-number"
					}
				}
			}`,
			shouldFail: true,
		},
		{
<<<<<<< HEAD
			name:          "EmptySchema",
			schemaContent: `{}`,
			shouldFail:    false, // Empty schema might be valid (allows anything)
=======
			name: "EmptySchema",
			schemaContent: `{}`,
			shouldFail: false, // Empty schema might be valid (allows anything)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			contractsDir := filepath.Join(tempDir, "docs", "contracts")
<<<<<<< HEAD
			if err := os.MkdirAll(contractsDir, 0o755); err != nil {
=======
			if err := os.MkdirAll(contractsDir, 0755); err != nil {
>>>>>>> 6835433495e87288b95961af7173d866977175ff
				t.Fatalf("Failed to create contracts directory: %v", err)
			}

			schemaPath := filepath.Join(contractsDir, "intent.schema.json")
<<<<<<< HEAD
			if err := os.WriteFile(schemaPath, []byte(tt.schemaContent), 0o644); err != nil {
=======
			if err := os.WriteFile(schemaPath, []byte(tt.schemaContent), 0644); err != nil {
>>>>>>> 6835433495e87288b95961af7173d866977175ff
				t.Fatalf("Failed to write schema file: %v", err)
			}

			validator, err := NewValidator(tempDir)

			if tt.shouldFail {
				if err == nil {
					t.Errorf("Expected NewValidator to fail for %s, but it succeeded", tt.name)
				}
				if validator != nil {
					t.Errorf("Expected validator to be nil for %s", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("Expected NewValidator to succeed for %s, but got error: %v", tt.name, err)
				}
				if validator == nil {
					t.Errorf("Expected validator to be non-nil for %s", tt.name)
				}
			}
		})
	}
}

// TestEdgeCaseValidation tests edge cases in validation logic
func TestEdgeCaseValidation(t *testing.T) {
	// Setup validator
	tempDir := t.TempDir()
	contractsDir := filepath.Join(tempDir, "docs", "contracts")
<<<<<<< HEAD
	if err := os.MkdirAll(contractsDir, 0o755); err != nil {
=======
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		t.Fatalf("Failed to create contracts directory: %v", err)
	}

	validSchema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["intent_type", "target", "namespace", "replicas"],
		"properties": {
<<<<<<< HEAD
			"intent_type": {"const": "scaling"},
=======
			"intent_type": {
				"type": "string",
				"enum": ["scaling", "deployment", "configuration"]
			},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			"target": {"type": "string", "minLength": 1},
			"namespace": {"type": "string", "minLength": 1},
			"replicas": {"type": "integer", "minimum": 1, "maximum": 100}
		}
	}`

	schemaPath := filepath.Join(contractsDir, "intent.schema.json")
<<<<<<< HEAD
	if err := os.WriteFile(schemaPath, []byte(validSchema), 0o644); err != nil {
=======
	if err := os.WriteFile(schemaPath, []byte(validSchema), 0644); err != nil {
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		t.Fatalf("Failed to write schema file: %v", err)
	}

	validator, err := NewValidator(tempDir)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	t.Run("NilIntent", func(t *testing.T) {
		// This should cause a panic or marshal error
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Expected panic when validating nil intent: %v", r)
			}
		}()

		errors := validator.ValidateIntent(nil)
		if len(errors) == 0 {
			t.Error("Expected validation errors for nil intent")
		}
	})

	t.Run("EmptyJSONBytes", func(t *testing.T) {
		errors := validator.ValidateJSON([]byte{})
		if len(errors) == 0 {
			t.Error("Expected validation errors for empty JSON bytes")
		}
	})

	t.Run("NilJSONBytes", func(t *testing.T) {
		errors := validator.ValidateJSON(nil)
		if len(errors) == 0 {
			t.Error("Expected validation errors for nil JSON bytes")
		}
	})

	t.Run("VeryLargeJSON", func(t *testing.T) {
		// Create a very large JSON (but still valid)
		longString := make([]byte, 10000) // 10KB string
		for i := range longString {
			longString[i] = 'a'
		}

		largeJSON := fmt.Sprintf(`{
			"intent_type": "scaling",
			"target": "test-deployment",
			"namespace": "test-namespace",
			"replicas": 5,
			"reason": "%s"
		}`, string(longString))

		errors := validator.ValidateJSON([]byte(largeJSON))
		// This should fail due to reason field length restriction if enforced by schema
		t.Logf("Large JSON validation resulted in %d errors", len(errors))
		for _, err := range errors {
			t.Logf("Large JSON error: %s - %s", err.Field, err.Message)
		}
	})
}

// Helper function to get keys from a map
func getKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Benchmark tests
func BenchmarkValidateIntent(b *testing.B) {
	// Setup
	tempDir := b.TempDir()
	contractsDir := filepath.Join(tempDir, "docs", "contracts")
<<<<<<< HEAD
	if err := os.MkdirAll(contractsDir, 0o755); err != nil {
=======
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		b.Fatalf("Failed to create contracts directory: %v", err)
	}

	validSchema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"$id": "https://example.com/schemas/intent.schema.json",
		"title": "ScalingIntent",
		"type": "object",
		"additionalProperties": false,
		"required": ["intent_type", "target", "namespace", "replicas"],
		"properties": {
			"intent_type": {
<<<<<<< HEAD
				"const": "scaling",
				"description": "MVP supports only scaling"
=======
				"type": "string",
				"enum": ["scaling", "deployment", "configuration"],
				"description": "MVP supports scaling, deployment, and configuration"
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			},
			"target": {
				"type": "string",
				"minLength": 1,
				"description": "Kubernetes Deployment name"
			},
			"namespace": {
				"type": "string",
				"minLength": 1
			},
			"replicas": {
				"type": "integer",
				"minimum": 1,
				"maximum": 100
			}
		}
	}`

	schemaPath := filepath.Join(contractsDir, "intent.schema.json")
<<<<<<< HEAD
	if err := os.WriteFile(schemaPath, []byte(validSchema), 0o644); err != nil {
=======
	if err := os.WriteFile(schemaPath, []byte(validSchema), 0644); err != nil {
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		b.Fatalf("Failed to write schema file: %v", err)
	}

	validator, err := NewValidator(tempDir)
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	intent := &ScalingIntent{
		IntentType: "scaling",
		Target:     "test-deployment",
		Namespace:  "test-namespace",
		Replicas:   5,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateIntent(intent)
	}
<<<<<<< HEAD
}
=======
}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
