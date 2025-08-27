package intent

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSecuritySchemaValidationHardening verifies that schema validation failures
// are treated as critical security errors and not silently ignored
func TestSecuritySchemaValidationHardening(t *testing.T) {
	tempDir := t.TempDir()
	contractsDir := filepath.Join(tempDir, "docs", "contracts")
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		t.Fatalf("Failed to create contracts directory: %v", err)
	}

	t.Run("SECURITY_CRITICAL_SchemaCompilationFailure", func(t *testing.T) {
		// This tests the specific vulnerability that was fixed
		// Schema compilation errors MUST cause hard failures
		invalidSchema := `{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type": "invalid-type-that-will-fail-compilation"
		}`

		schemaPath := filepath.Join(contractsDir, "intent.schema.json")
		if err := os.WriteFile(schemaPath, []byte(invalidSchema), 0644); err != nil {
			t.Fatalf("Failed to write schema file: %v", err)
		}

		validator, err := NewValidator(tempDir)

		// SECURITY ASSERTION: Schema compilation failure MUST return an error
		if err == nil {
			t.Fatal("SECURITY VULNERABILITY: Schema compilation failure did not return error - validation could be bypassed!")
		}

		// SECURITY ASSERTION: Validator MUST be nil when schema compilation fails
		if validator != nil {
			t.Fatal("SECURITY VULNERABILITY: Validator returned despite schema compilation failure - could lead to bypass!")
		}

		// Verify error message indicates security impact
		if err.Error() == "" || err.Error() == "nil" {
			t.Fatal("SECURITY: Error message should clearly indicate validation failure")
		}
	})

	t.Run("SECURITY_MalformedSchemaRejection", func(t *testing.T) {
		// Malformed JSON in schema file should be rejected
		malformedSchema := `{this is not valid json}`

		schemaPath := filepath.Join(contractsDir, "intent.schema.json")
		if err := os.WriteFile(schemaPath, []byte(malformedSchema), 0644); err != nil {
			t.Fatalf("Failed to write schema file: %v", err)
		}

		validator, err := NewValidator(tempDir)

		if err == nil {
			t.Fatal("SECURITY: Malformed schema JSON should cause initialization failure")
		}

		if validator != nil {
			t.Fatal("SECURITY: No validator should be created with malformed schema")
		}
	})

	t.Run("SECURITY_MissingSchemaFileRejection", func(t *testing.T) {
		// Missing schema file should cause hard failure
		nonExistentDir := filepath.Join(tempDir, "nonexistent")

		validator, err := NewValidator(nonExistentDir)

		if err == nil {
			t.Fatal("SECURITY: Missing schema file should cause initialization failure")
		}

		if validator != nil {
			t.Fatal("SECURITY: No validator should be created when schema file is missing")
		}
	})

	t.Run("SECURITY_ValidSchemaEnforcesConstraints", func(t *testing.T) {
		// With a valid schema, all constraints must be enforced
		validSchema := `{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type": "object",
			"additionalProperties": false,
			"required": ["intent_type", "target", "namespace", "replicas"],
			"properties": {
				"intent_type": {"const": "scaling"},
				"target": {"type": "string", "minLength": 1, "maxLength": 63},
				"namespace": {"type": "string", "minLength": 1, "maxLength": 63},
				"replicas": {"type": "integer", "minimum": 1, "maximum": 100}
			}
		}`

		schemaPath := filepath.Join(contractsDir, "intent.schema.json")
		if err := os.WriteFile(schemaPath, []byte(validSchema), 0644); err != nil {
			t.Fatalf("Failed to write schema file: %v", err)
		}

		validator, err := NewValidator(tempDir)
		if err != nil {
			t.Fatalf("Valid schema should initialize successfully: %v", err)
		}

		if validator == nil {
			t.Fatal("Validator should be created with valid schema")
		}

		// Test that validator is healthy
		if !validator.IsHealthy() {
			t.Fatal("SECURITY: Validator should be healthy with valid schema")
		}

		// Test rejection of malicious input - SQL injection attempt
		maliciousIntent := &ScalingIntent{
			IntentType: "scaling'; DROP TABLE users; --",
			Target:     "test",
			Namespace:  "test",
			Replicas:   5,
		}

		errors := validator.ValidateIntent(maliciousIntent)
		if len(errors) == 0 {
			t.Fatal("SECURITY: Malicious intent_type should be rejected")
		}

		// Test rejection of excessive replicas (DoS prevention)
		excessiveIntent := &ScalingIntent{
			IntentType: "scaling",
			Target:     "test",
			Namespace:  "test",
			Replicas:   9999, // Way over limit
		}

		errors = validator.ValidateIntent(excessiveIntent)
		if len(errors) == 0 {
			t.Fatal("SECURITY: Excessive replicas should be rejected to prevent DoS")
		}

		// Test rejection of additional properties (prevent data injection)
		maliciousJSON := `{
			"intent_type": "scaling",
			"target": "test",
			"namespace": "test",
			"replicas": 5,
			"malicious_field": "evil_payload"
		}`

		errors = validator.ValidateJSON([]byte(maliciousJSON))
		if len(errors) == 0 {
			t.Fatal("SECURITY: Additional properties should be rejected to prevent injection")
		}
	})

	t.Run("SECURITY_MetricsTracking", func(t *testing.T) {
		// Security metrics must be tracked for monitoring
		validSchema := `{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type": "object",
			"required": ["intent_type"],
			"properties": {
				"intent_type": {"const": "scaling"}
			}
		}`

		schemaPath := filepath.Join(contractsDir, "intent.schema.json")
		if err := os.WriteFile(schemaPath, []byte(validSchema), 0644); err != nil {
			t.Fatalf("Failed to write schema file: %v", err)
		}

		validator, err := NewValidator(tempDir)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		// Perform some validations
		validator.ValidateJSON([]byte(`{"intent_type": "scaling"}`)) // Valid
		validator.ValidateJSON([]byte(`{"intent_type": "invalid"}`)) // Invalid
		validator.ValidateJSON([]byte(`{invalid json}`))             // Malformed

		metrics := validator.GetMetrics()

		// Security monitoring requirements
		if metrics.TotalValidations < 3 {
			t.Error("SECURITY: All validation attempts must be tracked")
		}

		if metrics.ValidationErrors < 2 {
			t.Error("SECURITY: Failed validations must be tracked for anomaly detection")
		}

		if metrics.ValidationSuccesses < 1 {
			t.Error("SECURITY: Successful validations must be tracked")
		}

		if metrics.LastValidationTime.IsZero() {
			t.Error("SECURITY: Validation timestamps required for audit trail")
		}
	})
}

// TestSecurityValidationBypass attempts various bypass techniques
func TestSecurityValidationBypass(t *testing.T) {
	tempDir := t.TempDir()
	contractsDir := filepath.Join(tempDir, "docs", "contracts")
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		t.Fatalf("Failed to create contracts directory: %v", err)
	}

	validSchema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"additionalProperties": false,
		"required": ["intent_type", "target", "namespace", "replicas"],
		"properties": {
			"intent_type": {"const": "scaling"},
			"target": {"type": "string", "pattern": "^[a-z0-9-]+$", "maxLength": 63},
			"namespace": {"type": "string", "pattern": "^[a-z0-9-]+$", "maxLength": 63},
			"replicas": {"type": "integer", "minimum": 1, "maximum": 100}
		}
	}`

	schemaPath := filepath.Join(contractsDir, "intent.schema.json")
	if err := os.WriteFile(schemaPath, []byte(validSchema), 0644); err != nil {
		t.Fatalf("Failed to write schema file: %v", err)
	}

	validator, err := NewValidator(tempDir)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Bypass attempt 1: Unicode characters
	unicodeBypass := `{
		"intent_type": "scaling",
		"target": "test\u0000deployment",
		"namespace": "test",
		"replicas": 5
	}`

	errors := validator.ValidateJSON([]byte(unicodeBypass))
	if len(errors) == 0 {
		t.Error("SECURITY: Null bytes in strings should be rejected")
	}

	// Bypass attempt 2: Integer overflow
	overflowBypass := `{
		"intent_type": "scaling",
		"target": "test",
		"namespace": "test",
		"replicas": 999999999999999999999
	}`

	errors = validator.ValidateJSON([]byte(overflowBypass))
	if len(errors) == 0 {
		t.Error("SECURITY: Integer overflow attempts should be rejected")
	}

	// Bypass attempt 3: Type confusion
	typeConfusion := `{
		"intent_type": "scaling",
		"target": "test",
		"namespace": "test",
		"replicas": "5"
	}`

	errors = validator.ValidateJSON([]byte(typeConfusion))
	if len(errors) == 0 {
		t.Error("SECURITY: Type confusion (string for integer) should be rejected")
	}

	// Bypass attempt 4: Case sensitivity exploit
	caseBypass := `{
		"Intent_Type": "scaling",
		"TARGET": "test",
		"namespace": "test",
		"replicas": 5
	}`

	errors = validator.ValidateJSON([]byte(caseBypass))
	if len(errors) == 0 {
		t.Error("SECURITY: Case variations in field names should be rejected")
	}

	// Bypass attempt 5: Nested object injection
	nestedBypass := `{
		"intent_type": "scaling",
		"target": {"$ref": "file:///etc/passwd"},
		"namespace": "test",
		"replicas": 5
	}`

	errors = validator.ValidateJSON([]byte(nestedBypass))
	if len(errors) == 0 {
		t.Error("SECURITY: Object injection in string fields should be rejected")
	}
}

// BenchmarkSecurityValidation measures performance to detect DoS vulnerabilities
func BenchmarkSecurityValidation(b *testing.B) {
	tempDir := b.TempDir()
	contractsDir := filepath.Join(tempDir, "docs", "contracts")
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		b.Fatalf("Failed to create contracts directory: %v", err)
	}

	validSchema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["intent_type"],
		"properties": {
			"intent_type": {"const": "scaling"}
		}
	}`

	schemaPath := filepath.Join(contractsDir, "intent.schema.json")
	if err := os.WriteFile(schemaPath, []byte(validSchema), 0644); err != nil {
		b.Fatalf("Failed to write schema file: %v", err)
	}

	validator, err := NewValidator(tempDir)
	if err != nil {
		b.Fatalf("Failed to create validator: %v", err)
	}

	validJSON := []byte(`{"intent_type": "scaling"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateJSON(validJSON)
	}

	// Performance should be reasonable to prevent DoS
	// If validation takes > 1ms per operation, investigate
	nsPerOp := b.Elapsed().Nanoseconds() / int64(b.N)
	if nsPerOp > 1000000 { // 1ms in nanoseconds
		b.Errorf("SECURITY: Validation too slow (%d ns/op), may be vulnerable to DoS", nsPerOp)
	}
}
