package performance_tests

import (
	"context"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/rag"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
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
	// Create mock documents for the response
	documents := []*shared.SearchResult{
		{
			ID:         "doc-1",
			Content:    "Mock search result 1 for query: " + request.Query,
			Score:      0.85,
			Confidence: 0.85,
			Metadata:   map[string]interface{}{"source": "test"},
		},
		{
			ID:         "doc-2",
			Content:    "Mock search result 2 for query: " + request.Query,
			Score:      0.75,
			Confidence: 0.75,
			Metadata:   map[string]interface{}{"source": "test"},
		},
	}

	// Convert QueryRequest to RAGResponse format as expected by performance tests
	return &rag.RAGResponse{
		Answer:          "stub answer",
		SourceDocuments: rag.ConvertSharedSearchResultsToRAG(documents),
		Query:           request.Query,
		ProcessingTime:  50 * time.Millisecond,
		RetrievalTime:   25 * time.Millisecond,
		Confidence:      0.8,
		GenerationTime:  25 * time.Millisecond,
		UsedCache:       false,
		IntentType:      "test",
		Metadata:        make(map[string]interface{}),
	}, nil
}

// GenerateEmbedding generates embedding for a text
func (r *RAGServiceStub) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Stub implementation
	return make([]float32, 384), nil
}
