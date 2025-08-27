package rag

import (
	"context"
	"testing"

	"github.com/thc1006/nephoran-intent-operator/pkg/types"
)

// TestDocumentTypeIntegration tests the integration between pkg/rag and pkg/types
func TestDocumentTypeIntegration(t *testing.T) {
	// Create a TelecomDocument using the alias in pkg/rag
	ragDoc := &TelecomDocument{
		ID:              "integration-test-1",
		Title:           "Integration Test Document",
		Content:         "This tests the integration between pkg/rag and pkg/types",
		Source:          "test",
		Category:        "integration-test",
		Version:         "1.0",
		Keywords:        []string{"integration", "test"},
		Language:        "en",
		DocumentType:    "test-document",
		NetworkFunction: []string{"test-nf"},
		Technology:      []string{"test-tech"},
		UseCase:         []string{"testing"},
		Confidence:      0.95,
		Metadata: map[string]interface{}{
			"test_field": "test_value",
			"created_by": "integration_test",
		},
	}

	// Verify it's compatible with types.TelecomDocument
	var typesDoc *types.TelecomDocument = ragDoc
	if typesDoc.ID != "integration-test-1" {
		t.Errorf("Expected ID integration-test-1, got %s", typesDoc.ID)
	}
	if typesDoc.Title != "Integration Test Document" {
		t.Errorf("Expected title 'Integration Test Document', got %s", typesDoc.Title)
	}

	// Test WeaviateClient AddDocument with the document
	client := &WeaviateClient{}
	ctx := context.Background()

	err := client.AddDocument(ctx, ragDoc)
	if err != nil {
		t.Errorf("AddDocument should not return error for stub implementation, got %v", err)
	}

	// Test other WeaviateClient methods work with TelecomDocument references
	query := &SearchQuery{
		Query:         "integration test",
		Limit:         10,
		HybridSearch:  true,
		UseReranker:   false,
		MinConfidence: 0.7,
	}

	response, err := client.Search(ctx, query)
	if err != nil {
		t.Errorf("Search should not return error for stub implementation, got %v", err)
	}
	if response == nil {
		t.Error("Search should return non-nil response")
	}

	// Test health status includes version field
	health := client.GetHealthStatus()
	if health == nil {
		t.Error("GetHealthStatus should return non-nil status")
	}
	// Version field should be accessible (may be empty string for stub)
	_ = health.Version
}

// TestRAGClientIntegration tests the RAG client interface
func TestRAGClientIntegration(t *testing.T) {
	config := &RAGClientConfig{
		Enabled:          false, // Use no-op implementation
		MaxSearchResults: 50,
		MinConfidence:    0.8,
		WeaviateURL:      "http://localhost:8080",
		LLMEndpoint:      "http://localhost:8081",
		MaxTokens:        4096,
		Temperature:      0.3,
	}

	client := NewRAGClient(config)
	if client == nil {
		t.Fatal("NewRAGClient should not return nil")
	}

	ctx := context.Background()

	// Test Initialize
	err := client.Initialize(ctx)
	if err != nil {
		t.Errorf("Initialize should succeed for no-op client, got %v", err)
	}

	// Test Retrieve
	docs, err := client.Retrieve(ctx, "test query for integration")
	if err != nil {
		t.Errorf("Retrieve should succeed for no-op client, got %v", err)
	}
	// No-op client returns empty results, which is expected
	if docs == nil {
		t.Error("Retrieve should return non-nil slice (even if empty)")
	}

	// Test Shutdown
	err = client.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown should succeed for no-op client, got %v", err)
	}
}
