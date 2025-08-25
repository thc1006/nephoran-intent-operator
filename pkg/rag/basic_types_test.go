package rag

import (
	"context"
	"testing"

	"github.com/thc1006/nephoran-intent-operator/pkg/types"
)

// Basic test to verify our fixed types compile correctly
func TestBasicTypesCompilation(t *testing.T) {
	// Test Document type alias compatibility
	var telecomDoc TelecomDocument
	telecomDoc.ID = "test-id"
	telecomDoc.Content = "test content"

	// Should be assignable to types.TelecomDocument
	var typesDoc types.TelecomDocument = telecomDoc
	if typesDoc.ID != "test-id" {
		t.Errorf("Expected ID to be test-id, got %s", typesDoc.ID)
	}

	// Test WeaviateClient stub methods
	client := &WeaviateClient{}
	ctx := context.Background()

	// AddDocument should not panic
	err := client.AddDocument(ctx, &telecomDoc)
	if err != nil {
		t.Errorf("AddDocument should not error for stub, got %v", err)
	}

	// Test RAG client creation
	config := &RAGClientConfig{
		Enabled: false,
	}
	ragClient := NewRAGClient(config)
	if ragClient == nil {
		t.Error("NewRAGClient should not return nil")
	}

	// Should be able to call interface methods
	_, err = ragClient.Retrieve(ctx, "test query")
	if err != nil {
		t.Errorf("Retrieve should not error for no-op client, got %v", err)
	}

	err = ragClient.Initialize(ctx)
	if err != nil {
		t.Errorf("Initialize should not error for no-op client, got %v", err)
	}

	err = ragClient.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown should not error for no-op client, got %v", err)
	}
}

// Test basic search types
func TestSearchTypesBasic(t *testing.T) {
	// Test SearchQuery
	query := SearchQuery{
		Query:         "test query",
		Limit:         10,
		HybridSearch:  true,
		UseReranker:   true,
		MinConfidence: 0.5,
	}

	if query.Query != "test query" {
		t.Errorf("Expected query='test query', got %s", query.Query)
	}
	if query.Limit != 10 {
		t.Errorf("Expected limit=10, got %d", query.Limit)
	}

	// Test SearchResponse
	result := SearchResult{
		ID:         "result-1",
		Content:    "result content",
		Confidence: 0.8,
	}

	response := SearchResponse{
		Results: []*SearchResult{&result},
		Total:   1,
		Query:   "test",
	}

	if len(response.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(response.Results))
	}
}