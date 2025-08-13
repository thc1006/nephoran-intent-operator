package planner

import (
"bytes"
"encoding/json"
"fmt"
"net/http"
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

func PostIntent(url string, in Intent) error {
b, _ := json.Marshal(in)
req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
req.Header.Set("Content-Type", "application/json")
resp, err := http.DefaultClient.Do(req)
if err != nil {
return err
}
defer resp.Body.Close()
if resp.StatusCode >= 300 {
return fmt.Errorf("post intent status=%d", resp.StatusCode)
}
return nil
}
