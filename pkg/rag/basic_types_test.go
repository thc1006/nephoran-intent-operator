package rag

import (
	"context"
	"testing"

	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// Basic test to verify our fixed types compile correctly
func TestBasicTypesCompilation(t *testing.T) {
	// Test Document type alias compatibility
	var telecomDoc shared.TelecomDocument
	telecomDoc.ID = "test-id"
	telecomDoc.Content = "test content"

	// Should be assignable to shared.TelecomDocument
	if telecomDoc.ID != "test-id" {
		t.Errorf("Expected ID to be test-id, got %s", telecomDoc.ID)
	}

	// Test WeaviateClient stub methods
	config := &WeaviateConfig{
		Host:   "localhost",
		Scheme: "http",
	}
	client, err := NewWeaviateClient(config)
	if err != nil {
		t.Errorf("Expected no error creating client, got %v", err)
	}
	ctx := context.Background()

	// AddDocument should not panic
	err = client.AddDocument(ctx, &telecomDoc)
	if err != nil {
		t.Errorf("AddDocument should not error for stub, got %v", err)
	}

	// Test RAG client creation
	ragConfig := &RAGClientConfig{
		Enabled: false,
	}
	ragClient := NewRAGClient(ragConfig)
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
