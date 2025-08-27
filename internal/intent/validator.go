package intent

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// ValidatorMetrics tracks validation statistics for security monitoring
type ValidatorMetrics struct {
	TotalValidations    int64
	ValidationSuccesses int64
	ValidationErrors    int64
	SchemaLoadFailures  int64
	LastValidationTime  time.Time
}

// Validator handles JSON schema validation for scaling intents
type Validator struct {
	schema       *jsonschema.Schema
	schemaURI    string
	schemaPath   string
	schemaLoader *jsonschema.Schema // For compatibility with tests (points to same as schema)
	logger       *slog.Logger
	metrics      atomic.Pointer[ValidatorMetrics]
	initialized  atomic.Bool
}

// NewValidator creates a new validator using the schema from docs/contracts/intent.schema.json
func NewValidator(projectRoot string) (*Validator, error) {
	logger := slog.Default().With("component", "intent.validator")
	schemaPath := filepath.Join(projectRoot, "docs", "contracts", "intent.schema.json")

	// Log validation initialization for security audit
	logger.Info("initializing schema validator", "schema_path", schemaPath)

	// Read the schema file
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		// SECURITY: Log schema load failure for monitoring
		logger.Error("failed to read schema file",
			"path", schemaPath,
			"error", err)
		return nil, fmt.Errorf("failed to read schema file %s: %w", schemaPath, err)
	}

	// Parse the schema JSON to validate it's well-formed
	var schemaObj map[string]interface{}
	if err := json.Unmarshal(schemaData, &schemaObj); err != nil {
		// SECURITY: Log malformed schema for monitoring
		logger.Error("failed to parse schema JSON",
			"path", schemaPath,
			"error", err)
		return nil, fmt.Errorf("failed to parse schema JSON: %w", err)
	}

	// Try to compile the schema
	compiler := jsonschema.NewCompiler()
	schemaURI := "https://example.com/schemas/intent.schema.json"

	// Load the schema as a resource - parse JSON first
	var schemaInterface interface{}
	if err := json.Unmarshal(schemaData, &schemaInterface); err != nil {
		logger.Error("failed to unmarshal schema for compilation",
			"error", err)
		return nil, fmt.Errorf("failed to unmarshal schema for compilation: %w", err)
	}

	if err := compiler.AddResource(schemaURI, schemaInterface); err != nil {
		return nil, fmt.Errorf("failed to add schema resource: %w", err)
	}

	schema, err := compiler.Compile(schemaURI)
	if err != nil {
		// SECURITY: Schema compilation failure is a critical error
		// Never proceed with validation if schema is invalid
		logger.Error("CRITICAL: schema compilation failed",
			"schema_uri", schemaURI,
			"error", err,
			"security_impact", "validation disabled - rejecting all requests")
		return nil, fmt.Errorf("schema compilation failed - cannot proceed with validation: %w", err)
	}

	v := &Validator{
		schema:       schema,
		schemaURI:    schemaURI,
		schemaPath:   schemaPath,
		schemaLoader: schema, // For test compatibility
		logger:       logger,
	}

	// Initialize metrics
	v.metrics.Store(&ValidatorMetrics{})
	v.initialized.Store(true)

	logger.Info("schema validator initialized successfully",
		"schema_uri", schemaURI,
		"schema_path", schemaPath)

	return v, nil
}

// ValidateIntent validates a ScalingIntent against the JSON schema
func (v *Validator) ValidateIntent(intent *ScalingIntent) []ValidationError {
	// Update metrics
	metrics := v.metrics.Load()
	if metrics != nil {
		atomic.AddInt64(&metrics.TotalValidations, 1)
		metrics.LastValidationTime = time.Now()
	}

	// Schema should never be nil if validator was created successfully
	if v.schema == nil || !v.initialized.Load() {
		// This should never happen if NewValidator succeeded
		v.logger.Error("CRITICAL: validator not properly initialized",
			"has_schema", v.schema != nil,
			"initialized", v.initialized.Load())
		if metrics != nil {
			atomic.AddInt64(&metrics.ValidationErrors, 1)
		}
		return []ValidationError{{
			Field:   "validator",
			Message: "internal error: schema validator not initialized",
		}}
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
		if metrics != nil {
			atomic.AddInt64(&metrics.ValidationErrors, 1)
		}
		v.logger.Debug("validation failed",
			"error", err,
			"intent_type", intent.IntentType,
			"target", intent.Target)
		return v.convertValidationError(err)
	}

	if metrics != nil {
		atomic.AddInt64(&metrics.ValidationSuccesses, 1)
	}
	return nil
}

// ValidateJSON validates raw JSON data against the schema
func (v *Validator) ValidateJSON(data []byte) []ValidationError {
	// Update metrics
	metrics := v.metrics.Load()
	if metrics != nil {
		atomic.AddInt64(&metrics.TotalValidations, 1)
		metrics.LastValidationTime = time.Now()
	}

	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		if metrics != nil {
			atomic.AddInt64(&metrics.ValidationErrors, 1)
		}
		return []ValidationError{{
			Field:   "json",
			Message: fmt.Sprintf("invalid JSON: %v", err),
		}}
	}

	// Schema should never be nil if validator was created successfully
	if v.schema == nil || !v.initialized.Load() {
		// This should never happen if NewValidator succeeded
		v.logger.Error("CRITICAL: validator not properly initialized",
			"has_schema", v.schema != nil,
			"initialized", v.initialized.Load())
		if metrics != nil {
			atomic.AddInt64(&metrics.ValidationErrors, 1)
		}
		return []ValidationError{{
			Field:   "validator",
			Message: "internal error: schema validator not initialized",
		}}
	}

	if err := v.schema.Validate(obj); err != nil {
		if metrics != nil {
			atomic.AddInt64(&metrics.ValidationErrors, 1)
		}
		v.logger.Debug("JSON validation failed", "error", err)
		return v.convertValidationError(err)
	}

	if metrics != nil {
		atomic.AddInt64(&metrics.ValidationSuccesses, 1)
	}
	return nil
}

// convertValidationError converts jsonschema validation errors to our ValidationError type
func (v *Validator) convertValidationError(err error) []ValidationError {
	var errors []ValidationError

	if validationErr, ok := err.(*jsonschema.ValidationError); ok {
		// Convert instance location slice to string
		var fieldPath string
		if len(validationErr.InstanceLocation) > 0 {
			// Join without leading slash for compatibility with existing tests
			fieldPath = strings.Join(validationErr.InstanceLocation, "/")
		} else {
			fieldPath = "/"
		}

		errors = append(errors, ValidationError{
			Field:   fieldPath,
			Message: err.Error(), // Use the error string representation
		})

		// Add any nested validation errors
		for _, cause := range validationErr.Causes {
			childErrors := v.convertValidationError(cause)
			// Only add non-duplicate child errors
			for _, childErr := range childErrors {
				// Skip the root error if we've already added field-specific errors
				if childErr.Field != "/" {
					errors = append(errors, childErr)
				}
			}
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

// IsHealthy returns true if the validator is properly initialized and ready
func (v *Validator) IsHealthy() bool {
	return v != nil && v.schema != nil && v.initialized.Load()
}

// GetMetrics returns the current validation metrics for monitoring
func (v *Validator) GetMetrics() ValidatorMetrics {
	if v == nil || v.metrics.Load() == nil {
		return ValidatorMetrics{}
	}
	metrics := v.metrics.Load()
	return ValidatorMetrics{
		TotalValidations:    atomic.LoadInt64(&metrics.TotalValidations),
		ValidationSuccesses: atomic.LoadInt64(&metrics.ValidationSuccesses),
		ValidationErrors:    atomic.LoadInt64(&metrics.ValidationErrors),
		SchemaLoadFailures:  atomic.LoadInt64(&metrics.SchemaLoadFailures),
		LastValidationTime:  metrics.LastValidationTime,
	}
}

// GetSchemaInfo returns information about the loaded schema for diagnostics
func (v *Validator) GetSchemaInfo() map[string]interface{} {
	info := make(map[string]interface{})
	info["has_schema"] = v.schema != nil
	info["schema_uri"] = v.schemaURI
	info["schema_path"] = v.schemaPath
	info["is_healthy"] = v.IsHealthy()
	info["initialized"] = v.initialized.Load()
	return info
}
