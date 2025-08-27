package patch

import (
	"encoding/json"
	"fmt"
	"os"
)

// Intent represents the scaling intent structure
type Intent struct {
	IntentType    string `json:"intent_type"`
	Target        string `json:"target"`
	Namespace     string `json:"namespace"`
	Replicas      int    `json:"replicas"`
	Reason        string `json:"reason,omitempty"`
	Source        string `json:"source,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

// LoadIntent reads and parses an intent JSON file
func LoadIntent(path string) (*Intent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read intent file: %w", err)
	}

	var intent Intent
	if err := json.Unmarshal(data, &intent); err != nil {
		return nil, fmt.Errorf("failed to parse intent JSON: %w", err)
	}

	// Validate required fields
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
