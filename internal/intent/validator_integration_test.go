package intent

import (
	"path/filepath"
	"testing"
)

func TestValidatorWithActualSchema(t *testing.T) {
	// Test with the actual project schema
	projectRoot := filepath.Join("..", "..")
	
	t.Run("CreateValidatorWithActualSchema", func(t *testing.T) {
		validator, err := NewValidator(projectRoot)
		if err != nil {
			t.Fatalf("Failed to create validator with actual schema: %v", err)
		}

		if validator == nil {
			t.Fatal("Expected validator to be non-nil")
		}

		if !validator.IsHealthy() {
			t.Fatal("Expected validator to be healthy")
		}

		// Test that schema info is available
		info := validator.GetSchemaInfo()
		if info["has_schema"] != true {
			t.Error("Expected has_schema to be true")
		}

		if info["schema_uri"] != "https://example.com/schemas/intent.schema.json" {
			t.Errorf("Expected correct schema URI, got %v", info["schema_uri"])
		}
	})

	t.Run("ValidateValidIntent", func(t *testing.T) {
		validator, err := NewValidator(projectRoot)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		validIntent := &ScalingIntent{
			IntentType:    "scaling",
			Target:        "test-deployment",
			Namespace:     "test-namespace",
			Replicas:      5,
			Reason:        "Load testing",
			Source:        "user",
			CorrelationID: "test-123",
		}

		errors := validator.ValidateIntent(validIntent)
		if len(errors) > 0 {
			t.Errorf("Expected no validation errors for valid intent, got: %v", errors)
		}

		// Check metrics
		metrics := validator.GetMetrics()
		if metrics.ValidationSuccesses == 0 {
			t.Error("Expected at least one validation success")
		}
		if metrics.TotalValidations == 0 {
			t.Error("Expected total validations to be tracked")
		}
	})

	t.Run("ValidateInvalidIntent", func(t *testing.T) {
		validator, err := NewValidator(projectRoot)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		invalidIntent := &ScalingIntent{
			IntentType: "invalid", // Wrong intent type
			Target:     "test-deployment",
			Namespace:  "test-namespace",
			Replicas:   5,
		}

		errors := validator.ValidateIntent(invalidIntent)
		if len(errors) == 0 {
			t.Error("Expected validation errors for invalid intent type")
		}

		// Check metrics
		metrics := validator.GetMetrics()
		if metrics.ValidationErrors == 0 {
			t.Error("Expected at least one validation error to be recorded")
		}
	})

	t.Run("ValidateJSON", func(t *testing.T) {
		validator, err := NewValidator(projectRoot)
		if err != nil {
			t.Fatalf("Failed to create validator: %v", err)
		}

		validJSON := `{
			"intent_type": "scaling",
			"target": "test-deployment", 
			"namespace": "test-namespace",
			"replicas": 5,
			"reason": "Performance testing",
			"source": "user"
		}`

		errors := validator.ValidateJSON([]byte(validJSON))
		if len(errors) > 0 {
			t.Errorf("Expected no validation errors for valid JSON, got: %v", errors)
		}

		invalidJSON := `{
			"intent_type": "invalid",
			"target": "test-deployment",
			"namespace": "test-namespace", 
			"replicas": 5
		}`

		errors = validator.ValidateJSON([]byte(invalidJSON))
		if len(errors) == 0 {
			t.Error("Expected validation errors for invalid JSON")
		}
	})

	t.Run("TestHardErrorBehavior", func(t *testing.T) {
		// Test with non-existent project root - should fail hard
		invalidRoot := "/does/not/exist"
		
		validator, err := NewValidator(invalidRoot)
		if err == nil {
			t.Fatal("Expected hard error for non-existent schema file")
		}
		if validator != nil {
			t.Fatal("Expected nil validator on error")
		}

		// Verify this is not just a warning anymore
		if err != nil && err.Error() != "" {
			t.Logf("Correctly received hard error: %v", err)
		}
	})
}