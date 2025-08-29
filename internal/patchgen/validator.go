package patchgen

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// IntentSchema defines the JSON Schema 2020-12 for Intent validation.

const IntentSchema = `{

  "$schema": "https://json-schema.org/draft/2020-12/schema",

  "$id": "https://nephoran.io/schemas/intent.json",

  "title": "Intent Schema",

  "description": "Schema for validating Intent JSON structures",

  "type": "object",

  "properties": {

    "intent_type": {

      "type": "string",

      "enum": ["scaling"],

      "description": "Type of intent operation"

    },

    "target": {

      "type": "string",

      "minLength": 1,

      "description": "Name of the target deployment"

    },

    "namespace": {

      "type": "string",

      "minLength": 1,

      "description": "Kubernetes namespace"

    },

    "replicas": {

      "type": "integer",

      "minimum": 0,

      "maximum": 100,

      "description": "Desired number of replicas"

    },

    "reason": {

      "type": "string",

      "description": "Optional reason for the scaling operation"

    },

    "source": {

      "type": "string",

      "description": "Source system that generated the intent"

    },

    "correlation_id": {

      "type": "string",

      "description": "Correlation ID for tracking"

    }

  },

  "required": ["intent_type", "target", "namespace", "replicas"],

  "additionalProperties": false

}`

// Validator handles JSON Schema validation using JSON Schema 2020-12.

type Validator struct {
	schema *jsonschema.Schema

	logger logr.Logger
}

// NewValidator creates a new validator instance with the Intent schema.

func NewValidator(logger logr.Logger) (*Validator, error) {

	compiler := jsonschema.NewCompiler()

	compiler.DefaultDraft(jsonschema.Draft2020)

	// Parse the schema as JSON.

	var schemaDoc interface{}

	if err := json.Unmarshal([]byte(IntentSchema), &schemaDoc); err != nil {

		return nil, fmt.Errorf("failed to parse schema JSON: %w", err)

	}

	// Add the schema to the compiler.

	if err := compiler.AddResource("https://nephoran.io/schemas/intent.json", schemaDoc); err != nil {

		return nil, fmt.Errorf("failed to add schema resource: %w", err)

	}

	schema, err := compiler.Compile("https://nephoran.io/schemas/intent.json")

	if err != nil {

		return nil, fmt.Errorf("failed to compile schema: %w", err)

	}

	return &Validator{schema: schema, logger: logger.WithName("validator")}, nil

}

// ValidateIntent validates the intent JSON against the schema and returns the parsed Intent.

func (v *Validator) ValidateIntent(intentData []byte) (*Intent, error) {

	// First validate against schema.

	var rawIntent interface{}

	if err := json.Unmarshal(intentData, &rawIntent); err != nil {

		return nil, fmt.Errorf("invalid JSON: %w", err)

	}

	if err := v.schema.Validate(rawIntent); err != nil {

		return nil, fmt.Errorf("schema validation failed: %w", err)

	}

	// Parse into struct.

	var intent Intent

	if err := json.Unmarshal(intentData, &intent); err != nil {

		return nil, fmt.Errorf("failed to parse intent: %w", err)

	}

	return &intent, nil

}

// ValidateIntentFile reads and validates an intent file.

func (v *Validator) ValidateIntentFile(filePath string) (*Intent, error) {

	data, err := os.ReadFile(filePath)

	if err != nil {

		return nil, fmt.Errorf("failed to read intent file %s: %w", filePath, err)

	}

	return v.ValidateIntent(data)

}

// ValidateIntentMap validates an intent provided as a map.

func (v *Validator) ValidateIntentMap(intent map[string]interface{}) error {

	v.logger.V(1).Info("Validating intent", "intent", intent)

	if err := v.schema.Validate(intent); err != nil {

		return fmt.Errorf("schema validation failed: %w", err)

	}

	v.logger.Info("Intent validation successful")

	return nil

}

// LoadIntent reads and parses an intent JSON file with validation.

func LoadIntent(path string) (*Intent, error) {

	data, err := os.ReadFile(path)

	if err != nil {

		return nil, fmt.Errorf("failed to read intent file: %w", err)

	}

	var intent Intent

	if err := json.Unmarshal(data, &intent); err != nil {

		return nil, fmt.Errorf("failed to parse intent JSON: %w", err)

	}

	// Validate required fields.

	if intent.IntentType != "scaling" {

		return nil, fmt.Errorf("unsupported intent_type: %s (expected 'scaling')", intent.IntentType)

	}

	if intent.Target == "" {

		return nil, fmt.Errorf("target is required")

	}

	if intent.Namespace == "" {

		return nil, fmt.Errorf("namespace is required")

	}

	if intent.Replicas < 0 {

		return nil, fmt.Errorf("replicas must be >= 0")

	}

	return &intent, nil

}
