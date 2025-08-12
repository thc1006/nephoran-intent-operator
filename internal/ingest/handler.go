package ingest

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ValidatorInterface defines the contract for validation
type ValidatorInterface interface {
	ValidateBytes([]byte) (*Intent, error)
}

type Handler struct {
	v      ValidatorInterface
	outDir string
}

func NewHandler(v ValidatorInterface, outDir string) *Handler {
	_ = os.MkdirAll(outDir, 0o755)
	return &Handler{v: v, outDir: outDir}
}

var simple = regexp.MustCompile(`(?i)scale\s+([a-z0-9\-]+)\s+to\s+(\d+)\s+in\s+ns\s+([a-z0-9\-]+)`)

// 支援兩種輸入：
// 1) JSON（Content-Type: application/json）— 直接驗證
// 2) 純文字（例如 "scale nf-sim to 5 in ns ran-a"）— 解析→轉 JSON→驗證
func (h *Handler) HandleIntent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, fmt.Sprintf("Method %s not allowed. Only POST is supported for this endpoint.", r.Method), http.StatusMethodNotAllowed)
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
			if len(body) == 0 {
				http.Error(w, "Request body is empty. Expected JSON intent or plain text command like: scale <target> to <replicas> in ns <namespace>", http.StatusBadRequest)
			} else {
				http.Error(w, fmt.Sprintf("Invalid plain text format. Expected: 'scale <target> to <replicas> in ns <namespace>'. Received: %q", string(body)), http.StatusBadRequest)
			}
			return
		}
		payload = []byte(fmt.Sprintf(`{"intent_type":"scaling","target":"%s","namespace":"%s","replicas":%s,"source":"user"}`, m[1], m[3], m[2]))
	}

	intent, err := h.v.ValidateBytes(payload)
	if err != nil {
		errMsg := fmt.Sprintf("Intent validation failed: %s", err.Error())
		// Add hint for common errors
		if strings.Contains(err.Error(), "intent_type") {
			errMsg += ". Note: Currently only 'scaling' intent type is supported."
		} else if strings.Contains(err.Error(), "replicas") {
			errMsg += ". Note: Replicas must be between 1 and 100."
		} else if strings.Contains(err.Error(), "source") {
			errMsg += ". Note: Source must be one of: 'user', 'planner', or 'test'."
		}
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	ts := time.Now().UTC().Format("20060102T150405Z")
	outFile := filepath.Join(h.outDir, fmt.Sprintf("intent-%s.json", ts))
	if err := os.WriteFile(outFile, payload, 0o644); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save intent to handoff directory: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Log with correlation ID if present
	logMsg := fmt.Sprintf("Intent accepted and saved to %s", outFile)
	if intent.CorrelationID != "" {
		logMsg = fmt.Sprintf("[correlation_id: %s] %s", intent.CorrelationID, logMsg)
	}
	log.Println(logMsg)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "accepted",
		"saved":   outFile,
		"preview": intent,
	})
}
