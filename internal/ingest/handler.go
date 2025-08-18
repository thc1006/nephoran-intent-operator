package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Handler struct {
	v        *Validator
	outDir   string
	provider IntentProvider
}

func NewHandler(v *Validator, outDir string, provider IntentProvider) *Handler {
	_ = os.MkdirAll(outDir, 0755)
	return &Handler{v: v, outDir: outDir, provider: provider}
}

// HandleIntent supports two input types:
// 1) JSON (Content-Type: application/json) - direct validation
// 2) Plain text (e.g., "scale nf-sim to 5 in ns ran-a") - parse → convert to JSON → validate
func (h *Handler) HandleIntent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ct := r.Header.Get("Content-Type")
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	var payload []byte
	if strings.HasPrefix(ct, "application/json") || strings.HasPrefix(ct, "text/json") {
		payload = body
	} else {
		// Use provider to parse natural language text
		ctx := context.Background()
		intent, err := h.provider.ParseIntent(ctx, string(body))
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to parse intent: %v", err), http.StatusBadRequest)
			return
		}
		
		// Convert intent map to JSON
		jsonData, err := json.Marshal(intent)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to marshal intent: %v", err), http.StatusInternalServerError)
			return
		}
		payload = jsonData
	}

	intent, err := h.v.ValidateBytes(payload)
	if err != nil {
		http.Error(w, "schema validation failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	ts := time.Now().UTC().Format("20060102T150405Z")
	fileName := fmt.Sprintf("intent-%s.json", ts)
	outFile := filepath.Join(h.outDir, fileName)
	if err := os.WriteFile(outFile, payload, 0644); err != nil {
		http.Error(w, "write intent failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "accepted",
		"saved":   outFile,
		"preview": intent,
	})
}
