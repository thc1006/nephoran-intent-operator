//go:build !disable_rag
// +build !disable_rag

package rag

import (
	"context"
)

// NoopRAGClient implementation for when RAG is enabled but not fully implemented
type NoopRAGClient struct{}

func (c *NoopRAGClient) Query(ctx context.Context, query string) ([]*Doc, error) {
	return []*Doc{}, nil
}

func (c *NoopRAGClient) Close() error {
	return nil
}

func (c *NoopRAGClient) Retrieve(ctx context.Context, query string) ([]Doc, error) {
	return []Doc{}, nil
}

func (c *NoopRAGClient) Initialize(ctx context.Context) error {
	return nil
}

func (c *NoopRAGClient) ProcessIntent(ctx context.Context, intent string) (string, error) {
	return "RAG enabled but not fully implemented", nil
}

func (c *NoopRAGClient) IsHealthy() bool {
	return true
}

func (c *NoopRAGClient) Shutdown(ctx context.Context) error {
	return nil
}

// newRAGClientImpl creates a RAG client when RAG is enabled
// This is a basic implementation that should be replaced with full RAG functionality
func newRAGClientImpl(config *RAGClientConfig) RAGClient {
	return &NoopRAGClient{}
}