package ingest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// IntentSchemaValidator validates intents against the JSON schema.

type IntentSchemaValidator struct {
	schemaPath string

	schema map[string]interface{}
}

// NewIntentSchemaValidator creates a new schema validator.

func NewIntentSchemaValidator(schemaPath string) (*IntentSchemaValidator, error) {
	// Default to the standard schema location.

	if schemaPath == "" {
		schemaPath = filepath.Join("docs", "contracts", "intent.schema.json")
	}

	// Load and parse the schema.

	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	var schema map[string]interface{}

	if err := json.Unmarshal(schemaData, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}

	return &IntentSchemaValidator{
		schemaPath: schemaPath,

		schema: schema,
	}, nil
}

// ValidateIntent validates an intent against the JSON schema.

// This is a simplified validation that checks the main requirements.

func (v *IntentSchemaValidator) ValidateIntent(intent map[string]interface{}) error {
	// Get schema properties.

	properties, ok := v.schema["properties"].(map[string]interface{})

	if !ok {
		return fmt.Errorf("invalid schema format: missing properties")
	}

	// Get required fields.

	requiredFields, ok := v.schema["required"].([]interface{})

	if !ok {
		return fmt.Errorf("invalid schema format: missing required fields")
	}

	// Check required fields.

	for _, field := range requiredFields {

		fieldName, ok := field.(string)

		if !ok {
			continue
		}

		if _, exists := intent[fieldName]; !exists {
			return fmt.Errorf("missing required field: %s", fieldName)
		}

	}

	// Validate each field.

	for fieldName, value := range intent {

		// Check if field is allowed (additionalProperties: false).

		if additionalProps, ok := v.schema["additionalProperties"].(bool); ok && !additionalProps {
			if _, exists := properties[fieldName]; !exists {
				return fmt.Errorf("additional property not allowed: %s", fieldName)
			}
		}

		// Get field schema.

		fieldSchema, ok := properties[fieldName].(map[string]interface{})

		if !ok {
			continue // Field not in schema, skip if additional properties allowed
		}

		// Validate field based on its schema.

		if err := v.validateField(fieldName, value, fieldSchema); err != nil {
			return err
		}

	}

	return nil
}

// validateField validates a single field against its schema.

func (v *IntentSchemaValidator) validateField(fieldName string, value interface{}, fieldSchema map[string]interface{}) error {
	// Check const value.

	if constValue, ok := fieldSchema["const"]; ok {

		if value != constValue {
			return fmt.Errorf("field %s must be '%v', got '%v'", fieldName, constValue, value)
		}

		return nil

	}

	// Check type.

	if fieldType, ok := fieldSchema["type"].(string); ok {
		switch fieldType {

		case "string":

			strVal, ok := value.(string)

			if !ok {
				return fmt.Errorf("field %s must be a string, got %T", fieldName, value)
			}

			// Check minLength.

			if minLen, ok := fieldSchema["minLength"].(float64); ok {
				if len(strVal) < int(minLen) {
					return fmt.Errorf("field %s must have minimum length %d, got %d", fieldName, int(minLen), len(strVal))
				}
			}

			// Check maxLength.

			if maxLen, ok := fieldSchema["maxLength"].(float64); ok {
				if len(strVal) > int(maxLen) {
					return fmt.Errorf("field %s must have maximum length %d, got %d", fieldName, int(maxLen), len(strVal))
				}
			}

			// Check enum values.

			if enum, ok := fieldSchema["enum"].([]interface{}); ok {

				valid := false

				for _, enumVal := range enum {
					if strVal == enumVal {

						valid = true

						break

					}
				}

				if !valid {
					return fmt.Errorf("field %s must be one of %v, got '%s'", fieldName, enum, strVal)
				}

			}

		case "integer":

			// JSON numbers come as float64, need to check if it's an integer.

			numVal, ok := value.(float64)

			if !ok {
				// Try to handle actual int from Go code.

				if intVal, ok := value.(int); ok {
					numVal = float64(intVal)
				} else {
					return fmt.Errorf("field %s must be an integer, got %T", fieldName, value)
				}
			}

			// Check if it's actually an integer.

			if numVal != float64(int(numVal)) {
				return fmt.Errorf("field %s must be an integer, got %v", fieldName, numVal)
			}

			intVal := int(numVal)

			// Check minimum.

			if minVal, ok := fieldSchema["minimum"].(float64); ok {
				if intVal < int(minVal) {
					return fmt.Errorf("field %s must be at least %d, got %d", fieldName, int(minVal), intVal)
				}
			}

			// Check maximum.

			if maxVal, ok := fieldSchema["maximum"].(float64); ok {
				if intVal > int(maxVal) {
					return fmt.Errorf("field %s must be at most %d, got %d", fieldName, int(maxVal), intVal)
				}
			}

		case "object":

			// Handle nested objects

			objVal, ok := value.(map[string]interface{})

			if !ok {
				return fmt.Errorf("field %s must be an object, got %T", fieldName, value)
			}

			// Get nested properties schema

			nestedProperties, ok := fieldSchema["properties"].(map[string]interface{})

			if !ok {
				return nil // No properties defined, accept any object
			}

			// Get required fields for nested object

			requiredFields, _ := fieldSchema["required"].([]interface{})

			// Check required fields in nested object

			for _, reqField := range requiredFields {

				reqFieldName, ok := reqField.(string)

				if !ok {
					continue
				}

				if _, exists := objVal[reqFieldName]; !exists {
					return fmt.Errorf("field %s missing required nested field: %s", fieldName, reqFieldName)
				}

			}

			// Validate each nested field

			for nestedFieldName, nestedValue := range objVal {

				// Check if field is allowed (additionalProperties: false)

				if additionalProps, ok := fieldSchema["additionalProperties"].(bool); ok && !additionalProps {
					if _, exists := nestedProperties[nestedFieldName]; !exists {
						return fmt.Errorf("field %s additional property not allowed: %s", fieldName, nestedFieldName)
					}
				}

				// Get nested field schema

				nestedFieldSchema, ok := nestedProperties[nestedFieldName].(map[string]interface{})

				if !ok {
					continue // Field not in schema, skip if additional properties allowed
				}

				// Recursively validate nested field

				if err := v.validateField(fieldName+"."+nestedFieldName, nestedValue, nestedFieldSchema); err != nil {
					return err
				}

			}

		}
	}

	return nil
}

// Validate validates an intent against the JSON schema (alias for ValidateIntent).
func (v *IntentSchemaValidator) Validate(intent map[string]interface{}) error {
	return v.ValidateIntent(intent)
}

// ValidateJSON validates a JSON string against the schema.
func (v *IntentSchemaValidator) ValidateJSON(jsonStr string) error {
	if jsonStr == "" {
		return fmt.Errorf("failed to parse JSON: empty string")
	}

	// Parse JSON string into map.
	var intent map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &intent); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate the parsed intent.
	if err := v.ValidateIntent(intent); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}

// GetSchema returns the loaded JSON schema.
func (v *IntentSchemaValidator) GetSchema() map[string]interface{} {
	return v.schema
}

// UpdateSchema reloads the schema from the file system.
func (v *IntentSchemaValidator) UpdateSchema() error {
	// Load and parse the schema from file.
	schemaData, err := os.ReadFile(v.schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(schemaData, &schema); err != nil {
		return fmt.Errorf("failed to parse schema: %w", err)
	}

	// Update the schema.
	v.schema = schema
	return nil
}

// ValidateIntentWithSchema validates an intent using the JSON schema.

// This is a convenience function that creates a validator and validates in one step.

func ValidateIntentWithSchema(intent map[string]interface{}, schemaPath string) error {
	validator, err := NewIntentSchemaValidator(schemaPath)
	if err != nil {
		// If schema file doesn't exist, fall back to basic validation.

		return ValidateIntent(intent)
	}

	return validator.ValidateIntent(intent)
}
