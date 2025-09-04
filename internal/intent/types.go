package intent

import (
	"encoding/json"
	"fmt"
	"time"
)

// ScalingIntent represents a scaling intention based on the schema in docs/contracts/intent.schema.json.

type ScalingIntent struct {
	IntentType string `json:"intent_type" validate:"required,eq=scaling"`

	Target string `json:"target" validate:"required,min=1"`

	Namespace string `json:"namespace" validate:"required,min=1"`

	Replicas int `json:"replicas" validate:"required,min=1,max=100"`

	Reason string `json:"reason,omitempty" validate:"max=512"`

	Source string `json:"source,omitempty" validate:"oneof=user planner test ''"`

	CorrelationID string `json:"correlation_id,omitempty"`
}

// LoadResult contains the result of loading and validating an intent file.

type LoadResult struct {
	Intent *ScalingIntent

	Errors []ValidationError

	LoadedAt time.Time

	FilePath string

	IsValid bool
}

// ValidationError represents a validation error with context.

type ValidationError struct {
	Field string `json:"field"`

	Message string `json:"message"`

	Value any `json:"value,omitempty"`
}

// Error implements the error interface for ValidationError.

func (ve ValidationError) Error() string {
	return ve.Message
}

// ToJSON serializes the intent to JSON.

func (si *ScalingIntent) ToJSON() ([]byte, error) {
	return json.MarshalIndent(si, "", "  ")
}

// FromJSON deserializes the intent from JSON.

func (si *ScalingIntent) FromJSON(data []byte) error {
	return json.Unmarshal(data, si)
}

// String returns a human-readable representation of the intent.

func (si *ScalingIntent) String() string {
	return fmt.Sprintf("ScalingIntent{target=%s, namespace=%s, replicas=%d, source=%s}",

		si.Target, si.Namespace, si.Replicas, si.Source)
}

// NetworkIntent is an alias for ScalingIntent to maintain compatibility.

type NetworkIntent = ScalingIntent
