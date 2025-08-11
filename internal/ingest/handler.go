package ingest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Handler struct {
	v      *Validator
	outDir string
}

func NewHandler(v *Validator, outDir string) *Handler {
	_ = os.MkdirAll(outDir, 0o755)
	return &Handler{v: v, outDir: outDir}
}

var simple = regexp.MustCompile(`(?i)scale\s+([a-z0-9\-]+)\s+to\s+(\d+)\s+in\s+ns\s+([a-z0-9\-]+)`)

// 支援兩種輸入：
// 1) JSON（Content-Type: application/json）— 直接驗證
// 2) 純文字（例如 "scale nf-sim to 5 in ns ran-a"）— 解析→轉 JSON→驗證
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
		m := simple.FindStringSubmatch(string(body))
		if len(m) != 4 {
			http.Error(w, "bad plain text, expect: scale <target> to <replicas> in ns <namespace>", http.StatusBadRequest)
			return
		}
		payload = []byte(fmt.Sprintf(`{"intent_type":"scaling","target":"%s","namespace":"%s","replicas":%s,"source":"user"}`, m[1], m[3], m[2]))
	}

	intent, err := h.v.ValidateBytes(payload)
	if err != nil {
		http.Error(w, "schema validation failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	ts := time.Now().UTC().Format("20060102T150405Z")
	outFile := filepath.Join(h.outDir, fmt.Sprintf("intent-%s.json", ts))
	if err := os.WriteFile(outFile, payload, 0o644); err != nil {
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
