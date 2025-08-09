package rag

import (
	"context"
	"strings"
	"testing"
)

// TestBuildModes verifies that the RAG client works in both build modes
func TestBuildModes(t *testing.T) {
	config := &RAGClientConfig{
		Enabled:          true,
		MaxSearchResults: 5,
		MinConfidence:    0.7,
	}
	
	client := NewRAGClient(config)
	
	// Initialize should always work
	ctx := context.Background()
	err := client.Initialize(ctx)
	if err != nil {
		t.Errorf("Initialize failed: %v", err)
	}
	
	// ProcessIntent behavior depends on build tags
	result, err := client.ProcessIntent(ctx, "test intent")
	
	// Check if this is a no-op build (without rag tag)
	if err != nil && strings.Contains(err.Error(), "RAG support is not enabled") {
		// This is expected for no-op build
		t.Logf("Running with no-op RAG client (expected without rag build tag)")
	} else if err != nil {
		// Unexpected error
		t.Errorf("ProcessIntent failed unexpectedly: %v", err)
	} else {
		// RAG is enabled
		t.Logf("Running with Weaviate RAG client (build with rag tag): %s", result)
	}
	
	// IsHealthy should always return something
	healthy := client.IsHealthy()
	t.Logf("Client health status: %v", healthy)
	
	// Shutdown should always work
	err = client.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}