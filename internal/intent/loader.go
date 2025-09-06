package intent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Loader handles loading and validating intent files
type Loader struct {
	validator   *Validator
	projectRoot string
}

// NewLoader creates a new loader with the given project root
func NewLoader(projectRoot string) (*Loader, error) {
	validator, err := NewValidator(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to create validator: %w", err)
	}

	return &Loader{
		validator:   validator,
		projectRoot: projectRoot,
	}, nil
}

// LoadFromFile loads and validates an intent from a JSON file
func (l *Loader) LoadFromFile(filePath string) (*LoadResult, error) {
	startTime := time.Now()

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return &LoadResult{
			Errors:   []ValidationError{{Field: "file", Message: fmt.Sprintf("failed to read file: %v", err)}},
			LoadedAt: startTime,
			FilePath: filePath,
			IsValid:  false,
		}, err
	}

	return l.LoadFromJSON(data, filePath)
}

// LoadFromJSON loads and validates an intent from JSON data
func (l *Loader) LoadFromJSON(data []byte, sourcePath string) (*LoadResult, error) {
	startTime := time.Now()
	result := &LoadResult{
		LoadedAt: startTime,
		FilePath: sourcePath,
		IsValid:  false,
	}

	// First validate against the schema
	schemaErrors := l.validator.ValidateJSON(data)
	if len(schemaErrors) > 0 {
		result.Errors = schemaErrors
		return result, nil
	}

	// Parse into our struct
	var intent ScalingIntent
	if err := json.Unmarshal(data, &intent); err != nil {
		result.Errors = []ValidationError{{
			Field:   "json",
			Message: fmt.Sprintf("failed to unmarshal intent: %v", err),
		}}
		return result, nil
	}

	// Additional business logic validation
	bizErrors := l.validateBusinessLogic(&intent)
	if len(bizErrors) > 0 {
		result.Errors = bizErrors
		return result, nil
	}

	// Success
	result.Intent = &intent
	result.IsValid = true
	return result, nil
}

// validateBusinessLogic performs additional validation beyond the schema
func (l *Loader) validateBusinessLogic(intent *ScalingIntent) []ValidationError {
	var errors []ValidationError

	// Validate target name format (Kubernetes resource naming)
	if !isValidKubernetesName(intent.Target) {
		errors = append(errors, ValidationError{
			Field:   "target",
			Message: "target must be a valid Kubernetes resource name (lowercase alphanumeric and hyphens)",
			Value:   intent.Target,
		})
	}

	// Validate namespace format
	if !isValidKubernetesName(intent.Namespace) {
		errors = append(errors, ValidationError{
			Field:   "namespace",
			Message: "namespace must be a valid Kubernetes namespace name (lowercase alphanumeric and hyphens)",
			Value:   intent.Namespace,
		})
	}

	// Validate replicas range (additional business constraints)
	if intent.Replicas < 1 {
		errors = append(errors, ValidationError{
			Field:   "replicas",
			Message: "replicas must be at least 1",
			Value:   intent.Replicas,
		})
	}
	if intent.Replicas > 50 { // Business limit lower than schema max
		errors = append(errors, ValidationError{
			Field:   "replicas",
			Message: "replicas must not exceed 50 for MVP",
			Value:   intent.Replicas,
		})
	}

	return errors
}

// isValidKubernetesName checks if a string is a valid Kubernetes resource name
func isValidKubernetesName(name string) bool {
	if len(name) == 0 || len(name) > 63 {
		return false
	}

	// Must start and end with alphanumeric (per Kubernetes RFC 1123)
	if !isAlphaNumeric(name[0]) || !isAlphaNumeric(name[len(name)-1]) {
		return false
	}

	// Check each character
	for _, char := range name {
		if !isAlphaNumeric(byte(char)) && char != '-' {
			return false
		}
	}

	return true
}

// isAlphaNumeric checks if a byte is alphanumeric lowercase
func isAlphaNumeric(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}

// GetProjectRoot returns the project root directory
func (l *Loader) GetProjectRoot() string {
	return l.projectRoot
}

// GetSchemaPath returns the path to the intent schema file
func (l *Loader) GetSchemaPath() string {
	return filepath.Join(l.projectRoot, "docs", "contracts", "intent.schema.json")
}