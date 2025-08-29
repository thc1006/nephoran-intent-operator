//go:build !disable_rag




package rag



import (

	"context"

)



// NoopRAGClient implementation for when RAG is enabled but not fully implemented.

type NoopRAGClient struct{}



// Query performs query operation.

func (c *NoopRAGClient) Query(ctx context.Context, query string) ([]*Doc, error) {

	return []*Doc{}, nil

}



// Close performs close operation.

func (c *NoopRAGClient) Close() error {

	return nil

}



// Retrieve performs retrieve operation.

func (c *NoopRAGClient) Retrieve(ctx context.Context, query string) ([]Doc, error) {

	return []Doc{}, nil

}



// Initialize performs initialize operation.

func (c *NoopRAGClient) Initialize(ctx context.Context) error {

	return nil

}



// ProcessIntent performs processintent operation.

func (c *NoopRAGClient) ProcessIntent(ctx context.Context, intent string) (string, error) {

	return "RAG enabled but not fully implemented", nil

}



// IsHealthy performs ishealthy operation.

func (c *NoopRAGClient) IsHealthy() bool {

	return true

}



// Shutdown performs shutdown operation.

func (c *NoopRAGClient) Shutdown(ctx context.Context) error {

	return nil

}



// newRAGClientImpl creates a RAG client when RAG is enabled.

// This is a basic implementation that should be replaced with full RAG functionality.

func newRAGClientImpl(config *RAGClientConfig) RAGClient {

	return &NoopRAGClient{}

}

