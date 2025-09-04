//go:build !disable_rag
// +build !disable_rag

package llm

import (
	"fmt"
	"net/http"
)

// HandleStreamingRequest handles streaming requests - stub implementation.

func (sp *StreamingProcessor) HandleStreamingRequest(w http.ResponseWriter, r *http.Request, req *StreamingRequest) error {
	// Stub implementation - just return an error indicating this is not implemented.

	w.WriteHeader(http.StatusNotImplemented)

	_, err := w.Write([]byte("Streaming processor is in stub mode"))
	if err != nil {
		return fmt.Errorf("failed to write stub response: %w", err)
	}

	return nil
}
