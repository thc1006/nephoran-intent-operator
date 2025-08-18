package intent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Validator handles JSON schema validation for scaling intents
type Validator struct {
	schema    *jsonschema.Schema
	schemaURI string
}

// NewValidator creates a new validator using the schema from docs/contracts/intent.schema.json
func NewValidator(projectRoot string) (*Validator, error) {
	schemaPath := filepath.Join(projectRoot, "docs", "contracts", "intent.schema.json")
	
	// Read the schema file
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file %s: %w", schemaPath, err)
	}

	// For now, use a simplified validation approach
	// Parse the schema JSON to validate it's well-formed
	var schemaObj map[string]interface{}
	if err := json.Unmarshal(schemaData, &schemaObj); err != nil {
		return nil, fmt.Errorf("failed to parse schema JSON: %w", err)
	}

	// Try to compile the schema
	compiler := jsonschema.NewCompiler()
	schemaURI := "https://example.com/schemas/intent.schema.json"
	
	// Load the schema as a resource
	if err := compiler.AddResource(schemaURI, string(schemaData)); err != nil {
		return nil, fmt.Errorf("failed to add schema resource: %w", err)
	}

	schema, err := compiler.Compile(schemaURI)
	if err != nil {
		// Fallback: create a validator without schema compilation for MVP
		fmt.Printf("Warning: Schema compilation failed, using basic validation: %v\n", err)
		return &Validator{
			schema:    nil, // Will use basic validation
			schemaURI: schemaURI,
		}, nil
	}

	return &Validator{
		schema:    schema,
		schemaURI: schemaURI,
	}, nil
}

// ValidateIntent validates a ScalingIntent against the JSON schema
func (v *Validator) ValidateIntent(intent *ScalingIntent) []ValidationError {
	// If schema is nil, use basic validation
	if v.schema == nil {
		return v.basicValidation(intent)
	}

	// Convert intent to JSON for validation
	data, err := json.Marshal(intent)
	if err != nil {
		return []ValidationError{{
			Field:   "json",
			Message: fmt.Sprintf("failed to marshal intent: %v", err),
		}}
	}

	// Parse JSON
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return []ValidationError{{
			Field:   "json",
			Message: fmt.Sprintf("failed to unmarshal intent: %v", err),
		}}
	}

	// Validate against schema
	if err := v.schema.Validate(obj); err != nil {
		return v.convertValidationError(err)
	}

	return nil
}

// ValidateJSON validates raw JSON data against the schema
func (v *Validator) ValidateJSON(data []byte) []ValidationError {
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return []ValidationError{{
			Field:   "json",
			Message: fmt.Sprintf("invalid JSON: %v", err),
		}}
	}

	// If schema is nil, parse as ScalingIntent and use basic validation
	if v.schema == nil {
		var intent ScalingIntent
		if err := json.Unmarshal(data, &intent); err != nil {
			return []ValidationError{{
				Field:   "json",
				Message: fmt.Sprintf("failed to parse as ScalingIntent: %v", err),
			}}
		}
		return v.basicValidation(&intent)
	}

	if err := v.schema.Validate(obj); err != nil {
		return v.convertValidationError(err)
	}

	return nil
}

// basicValidation provides basic validation when schema compilation fails
func (v *Validator) basicValidation(intent *ScalingIntent) []ValidationError {
	var errors []ValidationError

	// Check required fields
	if intent.IntentType != "scaling" {
		errors = append(errors, ValidationError{
			Field:   "intent_type",
			Message: "must be 'scaling'",
			Value:   intent.IntentType,
		})
	}

	if intent.Target == "" {
		errors = append(errors, ValidationError{
			Field:   "target",
			Message: "is required",
		})
	}

	if intent.Namespace == "" {
		errors = append(errors, ValidationError{
			Field:   "namespace",
			Message: "is required",
		})
	}

	if intent.Replicas < 1 || intent.Replicas > 100 {
		errors = append(errors, ValidationError{
			Field:   "replicas",
			Message: "must be between 1 and 100",
			Value:   intent.Replicas,
		})
	}

	// Check optional field constraints
	if intent.Reason != "" && len(intent.Reason) > 512 {
		errors = append(errors, ValidationError{
			Field:   "reason",
			Message: "must be 512 characters or less",
			Value:   len(intent.Reason),
		})
	}

	if intent.Source != "" && intent.Source != "user" && intent.Source != "planner" && intent.Source != "test" {
		errors = append(errors, ValidationError{
			Field:   "source",
			Message: "must be one of: user, planner, test",
			Value:   intent.Source,
		})
	}

	return errors
}

// convertValidationError converts jsonschema validation errors to our ValidationError type
func (v *Validator) convertValidationError(err error) []ValidationError {
	var errors []ValidationError

	if validationErr, ok := err.(*jsonschema.ValidationError); ok {
		// Convert instance location slice to string
		var fieldPath string
		if len(validationErr.InstanceLocation) > 0 {
			fieldPath = fmt.Sprintf("/%s", strings.Join(validationErr.InstanceLocation, "/"))
		} else {
			fieldPath = "/"
		}

		errors = append(errors, ValidationError{
			Field:   fieldPath,
			Message: err.Error(), // Use the error string representation
		})

		// Add any nested validation errors
		for _, cause := range validationErr.Causes {
			errors = append(errors, v.convertValidationError(cause)...)
		}
	} else {
		// Fallback for other error types
		errors = append(errors, ValidationError{
			Field:   "unknown",
			Message: err.Error(),
		})
	}

	return errors
}

// GetSchemaURI returns the schema URI used by this validator
func (v *Validator) GetSchemaURI() string {
	return v.schemaURI
}