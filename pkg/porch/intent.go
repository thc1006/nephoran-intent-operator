package porch

import (
	"encoding/json"
	"fmt"
	"os"
)

// ScalingIntent represents the MVP scaling intent structure
type ScalingIntent struct {
	IntentType     string  `json:"intent_type"`
	Target         string  `json:"target"`
	Namespace      string  `json:"namespace"`
	Replicas       int     `json:"replicas"`
	Reason         *string `json:"reason,omitempty"`
	Source         *string `json:"source,omitempty"`
	CorrelationID  *string `json:"correlation_id,omitempty"`
}

// ValidateIntent validates the scaling intent
func ValidateIntent(intent *ScalingIntent) error {
	if intent.IntentType != "scaling" {
		return fmt.Errorf("invalid intent_type: %s, only 'scaling' is supported", intent.IntentType)
	}
	if intent.Target == "" {
		return fmt.Errorf("target cannot be empty")
	}
	if intent.Namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}
	if intent.Replicas < 1 {
		return fmt.Errorf("replicas must be at least 1, got %d", intent.Replicas)
	}
	if intent.Replicas > 100 {
		return fmt.Errorf("replicas cannot exceed 100, got %d", intent.Replicas)
	}
	return nil
}

// ParseIntentFromFile reads and parses an intent JSON file
func ParseIntentFromFile(path string) (*ScalingIntent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read intent file: %w", err)
	}
	
	var intent ScalingIntent
	if err := json.Unmarshal(data, &intent); err != nil {
		return nil, fmt.Errorf("failed to parse intent JSON: %w", err)
	}
	
	if err := ValidateIntent(&intent); err != nil {
		return nil, fmt.Errorf("intent validation failed: %w", err)
	}
	
	return &intent, nil
}