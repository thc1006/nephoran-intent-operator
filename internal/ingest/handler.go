package ingest

import (
	"context"
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
	v        ValidatorInterface
	outDir   string
	provider IntentProvider
}

func NewHandler(v ValidatorInterface, outDir string, provider IntentProvider) *Handler {
	_ = os.MkdirAll(outDir, 0o755)
	return &Handler{v: v, outDir: outDir, provider: provider}
}

var simple = regexp.MustCompile(`(?i)scale\s+([a-z0-9\-]+)\s+to\s+(\d+)\s+in\s+ns\s+([a-z0-9\-]+)`)

// HandleIntent supports two input types:
// 1) JSON (Content-Type: application/json) - direct validation
// 2) Plain text (e.g., "scale nf-sim to 5 in ns ran-a") - parse → convert to JSON → validate
func (h *Handler) HandleIntent(w http.ResponseWriter, r *http.Request) {
	// SECURITY FIX: Add security headers to prevent XSS and other attacks
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Content-Security-Policy", "default-src 'self'")

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, fmt.Sprintf("Method %s not allowed. Only POST is supported for this endpoint.", r.Method), http.StatusMethodNotAllowed)
		return
	}

	// SECURITY FIX: Validate Content-Type header to prevent MIME confusion attacks
	ct := r.Header.Get("Content-Type")
	if ct != "" && !strings.HasPrefix(ct, "application/json") && !strings.HasPrefix(ct, "text/json") && !strings.HasPrefix(ct, "text/plain") {
		http.Error(w, "Invalid Content-Type. Only application/json, text/json, and text/plain are allowed", http.StatusUnsupportedMediaType)
		return
	}

	// SECURITY FIX: Limit request body size to prevent DoS attacks
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024) // 1MB limit

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Request body too large or malformed", http.StatusRequestEntityTooLarge)
		return
	}
	defer r.Body.Close()

	var payload []byte
	if strings.HasPrefix(ct, "application/json") || strings.HasPrefix(ct, "text/json") {
		payload = body
	} else {
		// Try provider first (LLM or other custom parser)
		if h.provider != nil {
			ctx := context.Background()
			intent, err := h.provider.ParseIntent(ctx, string(body))
			if err == nil {
				// Convert intent map to JSON
				jsonData, err := json.Marshal(intent)
				if err == nil {
					payload = jsonData
				}
			}
		}
		
		// Fallback to regex parsing if provider failed or not available
		if payload == nil {
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

	// SECURITY FIX: Sanitize filename and validate output directory
	ts := time.Now().UTC().Format("20060102T150405Z")
	if !isValidTimestamp(ts) {
		http.Error(w, "Invalid timestamp format", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("intent-%s.json", ts)
	if !isValidFilename(filename) {
		http.Error(w, "Generated filename contains invalid characters", http.StatusInternalServerError)
		return
	}

	// Ensure the output directory exists and is within expected bounds
	absOutDir, err := filepath.Abs(h.outDir)
	if err != nil {
		http.Error(w, "Invalid output directory configuration", http.StatusInternalServerError)
		return
	}

	outFile := filepath.Join(absOutDir, filename)

	// Verify the resulting file path is still within the intended directory (prevent path traversal)
	if !strings.HasPrefix(filepath.Clean(outFile), filepath.Clean(absOutDir)) {
		http.Error(w, "Invalid file path detected", http.StatusBadRequest)
		return
	}
	if err := os.WriteFile(outFile, payload, 0o644); err != nil {
		http.Error(w, "Failed to save intent to handoff directory", http.StatusInternalServerError)
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


// isValidTimestamp validates timestamp format to prevent injection
func isValidTimestamp(ts string) bool {
	if len(ts) != 15 { // 20060102T150405Z format
		return false
	}
	// Only allow alphanumeric and T/Z characters
	for _, r := range ts {
		if !((r >= '0' && r <= '9') || r == 'T' || r == 'Z') {
			return false
		}
	}
	return true
}

// isValidFilename validates filename to prevent path traversal and injection
func isValidFilename(filename string) bool {
	if len(filename) == 0 || len(filename) > 255 {
		return false
	}

	// Reject dangerous characters
	dangerousChars := []string{"..", "/", "\\", ":", "*", "?", "\"", "<", ">", "|", "\x00"}
	for _, char := range dangerousChars {
		if strings.Contains(filename, char) {
			return false
		}
	}

	// Only allow specific pattern: intent-YYYYMMDDTHHMMSSZ.json
	matched, _ := regexp.MatchString(`^intent-\d{8}T\d{6}Z\.json$`, filename)
	return matched
}
