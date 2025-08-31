package performance_tests

import (
	"context"

	"github.com/nephio-project/nephoran-intent-operator/pkg/rag"
	"github.com/nephio-project/nephoran-intent-operator/pkg/types"
)

// RAGServiceStub provides stub methods for testing
type RAGServiceStub struct {
	service *rag.RAGService
}

// NewRAGServiceStub creates a new RAG service stub
func NewRAGServiceStub(service *rag.RAGService) *RAGServiceStub {
	return &RAGServiceStub{service: service}
}

// ProcessQuery processes a query and returns a response
func (r *RAGServiceStub) ProcessQuery(ctx context.Context, request *rag.RAGRequest) (*rag.RAGResponse, error) {
	// Convert QueryRequest to RAGRequest if needed
	return &rag.RAGResponse{
		Answer:          "stub answer",
		SourceDocuments: make([]*types.SearchResult, 0),
		Documents:       make([]*types.SearchResult, 0),
		Query:           request.Query,
		Context:         make(map[string]interface{}),
		Metadata:        make(map[string]interface{}),
	}, nil
}

// GenerateEmbedding generates embedding for a text
func (r *RAGServiceStub) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Stub implementation
	return make([]float32, 384), nil
}