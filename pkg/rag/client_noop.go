//go:build disable_rag

package rag

import (
	"context"
)

// noOpRAGClient is a no-op implementation of RAGClient.
// Used when the rag build tag is not enabled.
type noOpRAGClient struct {
	config *RAGClientConfig
}

// newRAGClientImpl creates a no-op RAG client.
func newRAGClientImpl(config *RAGClientConfig) RAGClient {
	return &noOpRAGClient{config: config}
}

// Retrieve returns empty results for no-op implementation.
func (c *noOpRAGClient) Retrieve(ctx context.Context, query string) ([]Doc, error) {
	// Return empty results - no error, just no content.
	return []Doc{}, nil
}

// Initialize is a no-op for the no-op implementation.
func (c *noOpRAGClient) Initialize(ctx context.Context) error {
	// No-op client requires no initialization.
	return nil
}

// ProcessIntent returns a no-op response.
func (c *noOpRAGClient) ProcessIntent(ctx context.Context, intent string) (string, error) {
	return "RAG support is not enabled", nil
}

// IsHealthy always returns false for no-op client.
func (c *noOpRAGClient) IsHealthy() bool {
	return false
}

// Shutdown is a no-op for the no-op implementation.
func (c *noOpRAGClient) Shutdown(ctx context.Context) error {
	// No-op client requires no shutdown.
	return nil
}

// Close is a no-op for the no-op implementation.
func (c *noOpRAGClient) Close() error {
	// No-op client requires no close.
	return nil
}

// Query returns empty results for no-op implementation.
func (c *noOpRAGClient) Query(ctx context.Context, query string) ([]*Doc, error) {
	// Return empty results - no error, just no content.
	return []*Doc{}, nil
}
