// Package planner implements rule-based closed-loop planning for O-RAN network optimization.
package planner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Intent matches the intent.schema.json contract
type Intent struct {
	IntentType    string `json:"intent_type"`              // "scaling"
	Target        string `json:"target"`                   // deployment name
	Namespace     string `json:"namespace"`                // k8s namespace
	Replicas      int    `json:"replicas"`                 // desired replicas (1-100)
	Reason        string `json:"reason,omitempty"`         // optional reason
	Source        string `json:"source,omitempty"`         // "user", "planner", or "test"
	CorrelationID string `json:"correlation_id,omitempty"` // optional trace id
}

// httpClient is a reusable HTTP client with timeout
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// PostIntent sends a scaling intent to the specified URL endpoint.
func PostIntent(url string, in Intent) error {
	b, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("failed to marshal intent: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("post intent status=%d", resp.StatusCode)
	}
	return nil
}
