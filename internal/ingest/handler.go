// Package ingest provides HTTP handlers and validation mechanisms
// for processing network intent commands from various input sources.
// It supports both JSON and plain text input formats with comprehensive
// security validation and intent parsing capabilities.

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

// ValidatorInterface defines the contract for validation of network intents.

type ValidatorInterface interface {
	ValidateBytes([]byte) (*Intent, error)
}

// Handler represents an HTTP handler for processing network intent requests.

type Handler struct {
	v ValidatorInterface

	outDir string

	provider IntentProvider
}

// NewHandler creates a new Handler instance with the specified validator, output directory, and intent provider.

func NewHandler(v ValidatorInterface, outDir string, provider IntentProvider) *Handler {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Printf("Warning: Failed to create output directory %s: %v", outDir, err)
	}

	return &Handler{v: v, outDir: outDir, provider: provider}
}

var simple = regexp.MustCompile(`(?i)scale\s+([a-z0-9\-]+)\s+to\s+(\d+)\s+in\s+ns\s+([a-z0-9\-]+)`)

// HandleIntent supports two input types:

// 1) JSON (Content-Type: application/json) - direct validation.

// 2) Plain text (e.g., "scale nf-sim to 5 in ns ran-a") - parse → convert to JSON → validate.

func (h *Handler) HandleIntent(w http.ResponseWriter, r *http.Request) {
	// SECURITY: Set security headers to prevent various attacks.

	w.Header().Set("X-Content-Type-Options", "nosniff")

	w.Header().Set("X-Frame-Options", "DENY")

	w.Header().Set("X-XSS-Protection", "1; mode=block")

	if r.Method != http.MethodPost {

		w.Header().Set("Allow", "POST")

		http.Error(w, fmt.Sprintf("Method %s not allowed. Only POST is supported for this endpoint.", r.Method), http.StatusMethodNotAllowed)

		return

	}

	// SECURITY: Validate Content-Type header to prevent MIME confusion attacks.

	ct := r.Header.Get("Content-Type")

	if ct != "" && !strings.HasPrefix(ct, "application/json") && !strings.HasPrefix(ct, "text/json") && !strings.HasPrefix(ct, "text/plain") {

		http.Error(w, "Invalid Content-Type", http.StatusUnsupportedMediaType)

		return

	}

	// SECURITY: Limit request body size to prevent DoS attacks.

	const maxRequestSize = 1 << 20 // 1MB

	if r.ContentLength > maxRequestSize {

		http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)

		return

	}

	// Use LimitReader as an additional safeguard.

	limitedReader := io.LimitReader(r.Body, maxRequestSize)

	body, err := io.ReadAll(limitedReader)
	if err != nil {

		http.Error(w, "Failed to read request body", http.StatusBadRequest)

		return

	}

	defer r.Body.Close() // #nosec G307 - Error handled in defer

	var payload []byte

	if strings.HasPrefix(ct, "application/json") || strings.HasPrefix(ct, "text/json") {
		payload = body
	} else {

		// Try provider first (LLM or other custom parser).

		if h.provider != nil {

			ctx := r.Context()

			intent, err := h.provider.ParseIntent(ctx, string(body))

			if err == nil {

				// Convert intent map to JSON.

				jsonData, err := json.Marshal(intent)

				if err == nil {
					payload = jsonData
				}

			}

		}

		// Fallback to regex parsing if provider failed or not available.

		if payload == nil {

			m := simple.FindStringSubmatch(string(body))

			if len(m) != 4 {

				if len(body) == 0 {
					http.Error(w, "Request body is empty. Expected JSON intent or plain text command like: scale <target> to <replicas> in ns <namespace>", http.StatusBadRequest)
				} else {
					http.Error(w, "Invalid plain text format. Expected: 'scale <target> to <replicas> in ns <namespace>'. Received: "+string(body), http.StatusBadRequest)
				}

				return

			}

			payload = []byte(fmt.Sprintf(`{"intent_type":"scaling","target":%q,"namespace":%q,"replicas":%s,"source":"user","target_resources":["deployment/%s"],"status":"pending"}`, m[1], m[3], m[2], m[1]))

		}

	}

	intent, err := h.v.ValidateBytes(payload)
	if err != nil {

		errMsg := fmt.Sprintf("Intent validation failed: %s", err.Error())

		// Add hint for common errors.

		errStr := err.Error()

		switch {

		case strings.Contains(errStr, "intent_type"):

			errMsg += ". Note: Currently only 'scaling' intent type is supported."

		case strings.Contains(errStr, "replicas"):

			errMsg += ". Note: Replicas must be between 1 and 100."

		case strings.Contains(errStr, "source"):

			errMsg += ". Note: Source must be one of: 'user', 'planner', or 'test'."

		}

		http.Error(w, errMsg, http.StatusBadRequest)

		return

	}

	now := time.Now().UTC()
	
	ts := now.Format("20060102T150405Z")

	fileName := fmt.Sprintf("intent-%s.json", ts)

	outFile := filepath.Join(h.outDir, fileName)

	// Ensure default values for target_resources and status
	targetResources := intent.TargetResources
	if targetResources == nil || len(targetResources) == 0 {
		targetResources = []string{"deployment/" + intent.Target}
	}
	
	status := intent.Status
	if status == "" {
		status = "pending"
	}

	// Create the structured output for saving to file
	structuredOutput := map[string]interface{}{
		"id":               fmt.Sprintf("scale-%s-001", intent.Target),
		"type":             intent.IntentType,
		"description":      fmt.Sprintf("Scale %s to %d replicas", intent.Target, intent.Replicas),
		"target_resources": targetResources,
		"status":           status,
		"parameters":       intent.ToPreviewFormat()["parameters"],
	}

	// Marshal structured output to JSON
	structuredJSON, err := json.MarshalIndent(structuredOutput, "", "  ")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal structured output: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(outFile, structuredJSON, 0o640); err != nil {

		http.Error(w, fmt.Sprintf("Failed to save intent to handoff directory: %s", err.Error()), http.StatusInternalServerError)

		return

	}

	// Log with correlation ID if present.

	logMsg := fmt.Sprintf("Intent accepted and saved to %s", outFile)

	if intent.CorrelationID != "" {
		logMsg = fmt.Sprintf("[correlation_id: %s] %s", intent.CorrelationID, logMsg)
	}

	log.Println(logMsg)

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusAccepted)

	if err := json.NewEncoder(w).Encode(map[string]any{
		"status": "accepted",

		"saved": outFile,

		"preview": intent.ToPreviewFormat(),
	}); err != nil {
		log.Printf("Failed to encode response JSON: %v", err)
	}
}
