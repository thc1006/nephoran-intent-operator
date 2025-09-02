package a1sim

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// DISABLED: func TestSavePolicyHandler_OK(t *testing.T) {
	tmp := t.TempDir()
	h := SavePolicyHandler(tmp)

	p := A1Policy{
		PolicyTypeID: "oran.sim.scaling.v1",
		Scope: PolicyScope{
			Namespace: "ran-a",
			Target:    "nf-sim",
		},
		Rules: PolicyRules{
			Metric:            "kpm.p95_latency_ms",
			ScaleOutThreshold: 100.0,
			ScaleInThreshold:  50.0,
			CooldownSeconds:   60,
			MinReplicas:       1,
			MaxReplicas:       10,
		},
	}
	b, _ := json.Marshal(p)

	req := httptest.NewRequest(http.MethodPost, "/a1/policies", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("want 202, got %d, body=%s", rr.Code, rr.Body.String())
	}
	files, _ := os.ReadDir(tmp)
	if len(files) == 0 {
		t.Fatalf("expected a saved policy in %s", tmp)
	}
	if filepath.Ext(files[0].Name()) != ".json" {
		t.Fatalf("expected .json file, got %s", files[0].Name())
	}
}
