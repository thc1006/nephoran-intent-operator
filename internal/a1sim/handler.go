// Package a1sim provides A1 interface simulation handlers for O-RAN policy management.


package a1sim



import (

	"encoding/json"

	"net/http"

	"os"

	"path/filepath"

	"time"

)



// A1Policy matches the a1.policy.schema.json contract.

type A1Policy struct {

	PolicyTypeID string      `json:"policyTypeId"` // "oran.sim.scaling.v1"

	Scope        PolicyScope `json:"scope"`

	Rules        PolicyRules `json:"rules"`

	Notes        string      `json:"notes,omitempty"`

}



// PolicyScope defines the target scope for an A1 policy.

type PolicyScope struct {

	Namespace string `json:"namespace"`

	Target    string `json:"target"`

}



// PolicyRules defines the scaling rules and thresholds for an A1 policy.

type PolicyRules struct {

	Metric            string  `json:"metric"` // e.g., "kpm.p95_latency_ms"

	ScaleOutThreshold float64 `json:"scale_out_threshold"`

	ScaleInThreshold  float64 `json:"scale_in_threshold"`

	CooldownSeconds   int     `json:"cooldown_seconds"`

	MinReplicas       int     `json:"min_replicas"`

	MaxReplicas       int     `json:"max_replicas"`

}



// SavePolicyHandler creates an HTTP handler that saves A1 policies to the specified directory.

func SavePolicyHandler(dir string) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {

			w.WriteHeader(http.StatusMethodNotAllowed)

			return

		}

		if err := os.MkdirAll(dir, 0o755); err != nil {

			http.Error(w, err.Error(), http.StatusInternalServerError)

			return

		}

		var p A1Policy

		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {

			http.Error(w, err.Error(), http.StatusBadRequest)

			return

		}

		name := time.Now().UTC().Format("20060102T150405Z") + ".json"

		path := filepath.Join(dir, name)

		b, err := json.MarshalIndent(p, "", "  ")

		if err != nil {

			http.Error(w, "failed to marshal policy: "+err.Error(), http.StatusInternalServerError)

			return

		}

		if err := os.WriteFile(path, b, 0o640); err != nil {

			http.Error(w, err.Error(), http.StatusInternalServerError)

			return

		}

		w.Header().Set("Content-Type", "application/json")

		w.WriteHeader(http.StatusAccepted)

		if _, err := w.Write([]byte(`{"status":"accepted","saved":"` + path + `"}`)); err != nil {

			// Log error but can't change status code at this point.

			http.Error(w, "Failed to write response", http.StatusInternalServerError)

			return

		}

	}

}

