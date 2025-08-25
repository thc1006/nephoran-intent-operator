package planner

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPostIntent_OK(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in Intent
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			t.Fatalf("bad json: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer s.Close()

	err := PostIntent(s.URL, Intent{
		IntentType: "scaling",
		Target:     "nf-sim",
		Namespace:  "ran-a",
		Replicas:   3,
		Source:     "planner",
		Reason:     "test scaling intent",
	})
	if err != nil {
		t.Fatalf("PostIntent: %v", err)
	}
}
