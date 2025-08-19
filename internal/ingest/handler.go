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

// IntentProvider interface for different parsing modes
type IntentProvider interface {
	ParseIntent(ctx context.Context, text string) (map[string]interface{}, error)
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

	ts := time.Now().UTC().Format("20060102T150405Z")
	fileName := fmt.Sprintf("intent-%s.json", ts)
	outFile := filepath.Join(h.outDir, fileName)
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

// NewProvider creates an IntentProvider based on the mode
func NewProvider(mode string, providerName string) (IntentProvider, error) {
	switch mode {
	case "llm":
		// For now, return a mock provider for LLM mode
		return &MockLLMProvider{}, nil
	case "rules":
		// Rules mode doesn't need a provider (uses regex)
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown mode: %s", mode)
	}
}

// MockLLMProvider is a mock implementation for testing
type MockLLMProvider struct{}

func (m *MockLLMProvider) ParseIntent(ctx context.Context, text string) (map[string]interface{}, error) {
	// Simple mock that mimics the regex parser for testing
	matches := simple.FindStringSubmatch(text)
	if len(matches) != 4 {
		return nil, fmt.Errorf("could not parse intent from text")
	}
	
	return map[string]interface{}{
		"intent_type": "scaling",
		"target":      matches[1],
		"namespace":   matches[3],
		"replicas":    matches[2],
		"source":      "user",
	}, nil
}