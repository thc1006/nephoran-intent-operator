package watch

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Validator handles JSON schema validation with caching.
type Validator struct {
	mu       sync.RWMutex
	compiler *jsonschema.Compiler
	schema   *jsonschema.Schema
	path     string
}

// NewValidator creates a new validator with the given schema file.
func NewValidator(schemaPath string) (*Validator, error) {
	v := &Validator{
		path:     schemaPath,
		compiler: jsonschema.NewCompiler(),
	}

	if err := v.loadSchema(); err != nil {
		return nil, fmt.Errorf("failed to load schema: %w", err)
	}

	return v, nil
}

// loadSchema loads and compiles the JSON schema.
func (v *Validator) loadSchema() error {
	// Read schema file.
	schemaData, err := os.ReadFile(v.path)
	if err != nil {
		return fmt.Errorf("failed to read schema file %s: %w", v.path, err)
	}

	// Parse schema as JSON.
	var schemaJSON interface{}
	if err := json.Unmarshal(schemaData, &schemaJSON); err != nil {
		return fmt.Errorf("failed to parse schema JSON: %w", err)
	}

	// Add schema to compiler with draft 2020-12 support.
	// Note: jsonschema/v6 automatically detects the draft from $schema field.
	if err := v.compiler.AddResource(v.path, schemaJSON); err != nil {
		return fmt.Errorf("failed to add schema to compiler: %w", err)
	}

	// Compile the schema.
	schema, err := v.compiler.Compile(v.path)
	if err != nil {
		return fmt.Errorf("failed to compile schema: %w", err)
	}

	v.schema = schema
	return nil
}

// Validate validates JSON data against the schema.
func (v *Validator) Validate(data []byte) error {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Parse JSON for validation.
	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Validate against schema.
	if err := v.schema.Validate(jsonData); err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}

	return nil
}

// ReloadSchema reloads the schema file (useful for hot reload).
func (v *Validator) ReloadSchema() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.compiler = jsonschema.NewCompiler()
	return v.loadSchema()
}
