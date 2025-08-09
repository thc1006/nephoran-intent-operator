//go:build !rag

package rag

import (
	"context"
	"fmt"
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

// ProcessIntent returns an error indicating RAG is not enabled
func (c *noOpRAGClient) ProcessIntent(ctx context.Context, intent string) (string, error) {
	return "", fmt.Errorf("RAG support is not enabled. Build with -tags=rag to enable Weaviate integration")
}

// Search returns an error indicating RAG is not enabled
func (c *noOpRAGClient) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	return nil, fmt.Errorf("RAG support is not enabled. Build with -tags=rag to enable Weaviate integration")
}

// Initialize does nothing for no-op implementation
func (c *noOpRAGClient) Initialize(ctx context.Context) error {
	// No-op: nothing to initialize
	return nil
}

// Shutdown does nothing for no-op implementation
func (c *noOpRAGClient) Shutdown(ctx context.Context) error {
	// No-op: nothing to shutdown
	return nil
}

// IsHealthy always returns true for no-op implementation
func (c *noOpRAGClient) IsHealthy() bool {
	// No-op is always "healthy" since it does nothing
	return true
}