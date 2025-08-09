//go:build !rag

package rag

import (
	"context"
)

// noOpRAGClient is a no-op implementation of RAGClient
// Used when the rag build tag is not enabled
type noOpRAGClient struct {
	config *RAGClientConfig
}

// newRAGClientImpl creates a no-op RAG client
func newRAGClientImpl(config *RAGClientConfig) RAGClient {
	return &noOpRAGClient{config: config}
}

// Retrieve returns empty results for no-op implementation
func (c *noOpRAGClient) Retrieve(ctx context.Context, query string) ([]Doc, error) {
	// Return empty results - no error, just no content
	return []Doc{}, nil
}

// Initialize is a no-op for the no-op implementation
func (c *noOpRAGClient) Initialize(ctx context.Context) error {
	// No-op client requires no initialization
	return nil
}

// Shutdown is a no-op for the no-op implementation
func (c *noOpRAGClient) Shutdown(ctx context.Context) error {
	// No-op client requires no shutdown
	return nil
}