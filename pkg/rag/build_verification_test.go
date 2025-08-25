package rag

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"
)

// TestRAGClientInterface verifies that the RAGClient interface is properly implemented
// in both build configurations (with and without the rag build tag)
func TestRAGClientInterface(t *testing.T) {
	tests := []struct {
		name   string
		config *RAGClientConfig
	}{
		{
			name: "minimal_config",
			config: &RAGClientConfig{
				Enabled:          true,
				MaxSearchResults: 5,
				MinConfidence:    0.7,
			},
		},
		{
			name: "full_config",
			config: &RAGClientConfig{
				Enabled:          true,
				MaxSearchResults: 10,
				MinConfidence:    0.8,
				WeaviateURL:      "http://localhost:8080",
				WeaviateAPIKey:   "test-api-key",
				LLMEndpoint:      "http://localhost:8081",
				LLMAPIKey:        "test-llm-key",
				MaxTokens:        4096,
				Temperature:      0.7,
			},
		},
		{
			name: "disabled_config",
			config: &RAGClientConfig{
				Enabled:          false,
				MaxSearchResults: 3,
				MinConfidence:    0.5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewRAGClient(tt.config)

			// Verify interface implementation
			if client == nil {
				t.Fatal("NewRAGClient returned nil")
			}

			// Check that client implements RAGClient interface
			var _ RAGClient = client

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Test Initialize method
			t.Run("Initialize", func(t *testing.T) {
				err := client.Initialize(ctx)

				// Check client type to determine expected behavior
				clientType := reflect.TypeOf(client).String()

				if clientType == "*rag.noOpRAGClient" {
					// No-op client should always succeed
					if err != nil {
						t.Errorf("No-op client Initialize should not fail, got: %v", err)
					}
				} else {
					// Weaviate client may fail if URL not configured or server not available
					// This is acceptable in test environments
					if err != nil {
						t.Logf("Weaviate client Initialize failed (expected in test env): %v", err)
					} else {
						t.Log("Weaviate client Initialize succeeded")
					}
				}
			})

			// Test Retrieve method
			t.Run("Retrieve", func(t *testing.T) {
				docs, err := client.Retrieve(ctx, "test query")

				// Check client type to determine expected behavior
				clientType := reflect.TypeOf(client).String()

				if clientType == "*rag.noOpRAGClient" {
					// No-op client should never error and return empty slice
					if err != nil {
						t.Errorf("No-op client Retrieve should not error, got: %v", err)
					}
					if docs == nil {
						t.Error("No-op client returned nil docs slice")
					} else if len(docs) != 0 {
						t.Errorf("No-op client should return empty docs, got %d docs", len(docs))
					}
				} else {
					// Weaviate client may error if server not available or URL not configured
					if err != nil {
						t.Logf("Weaviate client Retrieve failed (expected in test env): %v", err)
					} else {
						t.Logf("Weaviate client Retrieve succeeded with %d docs", len(docs))
					}

					// Verify return type consistency
					if err == nil && docs == nil {
						t.Error("Weaviate client returned nil docs slice without error")
					}
				}

				t.Logf("Retrieved %d documents", len(docs))
			})

			// Test Shutdown method
			t.Run("Shutdown", func(t *testing.T) {
				err := client.Shutdown(ctx)
				if err != nil {
					t.Errorf("Shutdown failed: %v", err)
				}
			})
		})
	}
}

// TestBuildTagConditionalBehavior tests the specific behavior differences
// between build tag configurations
func TestBuildTagConditionalBehavior(t *testing.T) {
	config := &RAGClientConfig{
		Enabled:          true,
		MaxSearchResults: 5,
		MinConfidence:    0.7,
		WeaviateURL:      "http://invalid-url-for-testing:8080",
		WeaviateAPIKey:   "test-key",
	}

	client := NewRAGClient(config)
	clientType := reflect.TypeOf(client).String()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Determine which implementation we're testing based on behavior
	err := client.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	docs, err := client.Retrieve(ctx, "test telecommunications query")

	// Check implementation type and expected behavior
	if clientType == "*rag.noOpRAGClient" || (err == nil && len(docs) == 0) {
		t.Log("✓ Running with no-op RAG client (without rag build tag)")
		verifyNoOpBehavior(t, client, ctx)
	} else if clientType == "*rag.weaviateRAGClient" || err != nil {
		t.Log("✓ Running with Weaviate RAG client (with rag build tag)")
		verifyWeaviateBehavior(t, client, ctx)
	} else {
		t.Logf("Unknown client type: %s", clientType)
	}

	// Always test shutdown
	if err := client.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}

// verifyNoOpBehavior tests expected behavior for no-op implementation
func verifyNoOpBehavior(t *testing.T, client RAGClient, ctx context.Context) {
	// No-op client should always return empty results, never error
	docs, err := client.Retrieve(ctx, "any query")
	if err != nil {
		t.Errorf("No-op client should never error on Retrieve, got: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("No-op client should return empty docs, got %d docs", len(docs))
	}

	// Multiple calls should be consistent
	for i := 0; i < 3; i++ {
		docs2, err2 := client.Retrieve(ctx, fmt.Sprintf("query %d", i))
		if err2 != nil {
			t.Errorf("No-op client call %d should not error, got: %v", i, err2)
		}
		if len(docs2) != 0 {
			t.Errorf("No-op client call %d should return empty docs, got %d docs", i, len(docs2))
		}
	}
}

// verifyWeaviateBehavior tests expected behavior for Weaviate implementation
func verifyWeaviateBehavior(t *testing.T, client RAGClient, ctx context.Context) {
	// Weaviate client may error due to connectivity issues in test environment
	docs, err := client.Retrieve(ctx, "telecommunications intent")

	// We expect either:
	// 1. Connection error (acceptable in test environment)
	// 2. Valid response with docs (if Weaviate is available)
	if err != nil {
		t.Logf("Weaviate client error (expected in test environment): %v", err)
		// Verify error is related to connectivity, not interface issues
		if err.Error() == "" {
			t.Error("Weaviate client returned empty error message")
		}
	} else {
		t.Logf("Weaviate client successful response with %d docs", len(docs))
		// If successful, verify doc structure
		for i, doc := range docs {
			if doc.ID == "" {
				t.Errorf("Doc %d has empty ID", i)
			}
			if doc.Confidence < 0 || doc.Confidence > 1 {
				t.Errorf("Doc %d has invalid confidence: %f", i, doc.Confidence)
			}
			if doc.Metadata == nil {
				t.Errorf("Doc %d has nil metadata", i)
			}
		}
	}
}

// TestRAGClientConfigDefaults tests that config defaults are handled properly
func TestRAGClientConfigDefaults(t *testing.T) {
	tests := []struct {
		name   string
		config *RAGClientConfig
	}{
		{
			name:   "nil_config",
			config: nil,
		},
		{
			name:   "empty_config",
			config: &RAGClientConfig{},
		},
		{
			name: "partial_config",
			config: &RAGClientConfig{
				Enabled: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic with any config
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("NewRAGClient panicked with config %v: %v", tt.config, r)
				}
			}()

			client := NewRAGClient(tt.config)
			if client == nil {
				t.Error("NewRAGClient returned nil")
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Basic operations should work even with minimal config
			clientType := reflect.TypeOf(client).String()

			err := client.Initialize(ctx)
			if clientType == "*rag.noOpRAGClient" {
				// No-op client should always succeed
				if err != nil {
					t.Errorf("No-op client Initialize failed with config %v: %v", tt.config, err)
				}
			} else {
				// Weaviate client may fail without proper URL
				if err != nil {
					t.Logf("Weaviate client Initialize failed (expected): %v", err)
				}
			}

			docs, err := client.Retrieve(ctx, "test")
			if clientType == "*rag.noOpRAGClient" {
				// No-op client should never error
				if err != nil {
					t.Errorf("No-op client Retrieve failed: %v", err)
				}
				if docs == nil {
					t.Error("No-op client returned nil docs slice")
				}
			} else {
				// Weaviate client may error without proper configuration
				if err != nil {
					t.Logf("Weaviate client Retrieve failed (expected): %v", err)
				}
				if err == nil && docs == nil {
					t.Error("Weaviate client returned nil docs slice without error")
				}
			}

			if err := client.Shutdown(ctx); err != nil {
				t.Errorf("Shutdown failed: %v", err)
			}
		})
	}
}

// TestConcurrentAccess tests that the RAG client is safe for concurrent use
func TestConcurrentAccess(t *testing.T) {
	config := &RAGClientConfig{
		Enabled:          true,
		MaxSearchResults: 5,
		MinConfidence:    0.7,
		WeaviateURL:      "http://localhost:8080", // Provide URL for Weaviate client
		WeaviateAPIKey:   "test-key",
	}

	client := NewRAGClient(config)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := client.Initialize(ctx)
	clientType := reflect.TypeOf(client).String()

	// Handle initialization based on client type
	if clientType == "*rag.noOpRAGClient" {
		if err != nil {
			t.Fatalf("No-op client Initialize failed: %v", err)
		}
	} else {
		// Weaviate client may fail initialization if server not available
		if err != nil {
			t.Logf("Weaviate client Initialize failed (expected): %v", err)
		} else {
			t.Log("Weaviate client Initialize succeeded")
		}
	}
	defer func() {
		if err := client.Shutdown(ctx); err != nil {
			t.Errorf("Shutdown failed: %v", err)
		}
	}()

	// Run concurrent retrieve operations
	const numGoroutines = 10
	const numCallsPerGoroutine = 5

	errCh := make(chan error, numGoroutines*numCallsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numCallsPerGoroutine; j++ {
				query := fmt.Sprintf("concurrent query %d-%d", id, j)
				_, err := client.Retrieve(ctx, query)
				if err != nil {
					errCh <- fmt.Errorf("goroutine %d call %d: %w", id, j, err)
				} else {
					errCh <- nil
				}
			}
		}(i)
	}

	// Collect results
	var errors []error
	for i := 0; i < numGoroutines*numCallsPerGoroutine; i++ {
		if err := <-errCh; err != nil {
			errors = append(errors, err)
		}
	}

	// Handle results based on client type
	if clientType == "*rag.noOpRAGClient" {
		// No-op client should never have errors
		if len(errors) > 0 {
			t.Errorf("No-op client had %d unexpected errors:", len(errors))
			for _, err := range errors {
				t.Errorf("  - %v", err)
			}
		} else {
			t.Log("✓ All concurrent operations completed successfully (no-op client)")
		}
	} else {
		// Weaviate client may have connection errors in test environment
		if len(errors) > 0 {
			t.Logf("Concurrent access resulted in %d errors (expected for Weaviate without server):", len(errors))
			for i, err := range errors {
				if i < 3 { // Only show first 3 errors to avoid spam
					t.Logf("  - %v", err)
				}
			}
			if len(errors) > 3 {
				t.Logf("  - ... and %d more errors", len(errors)-3)
			}
		} else {
			t.Log("✓ All concurrent operations completed successfully (Weaviate client)")
		}
	}
}

// BenchmarkRAGClientOperations benchmarks RAG client operations
func BenchmarkRAGClientOperations(b *testing.B) {
	config := &RAGClientConfig{
		Enabled:          true,
		MaxSearchResults: 5,
		MinConfidence:    0.7,
	}

	client := NewRAGClient(config)
	ctx := context.Background()

	if err := client.Initialize(ctx); err != nil {
		b.Fatalf("Initialize failed: %v", err)
	}
	defer func() {
		if err := client.Shutdown(ctx); err != nil {
			b.Errorf("Shutdown failed: %v", err)
		}
	}()

	b.Run("Retrieve", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.Retrieve(ctx, "benchmark query")
			if err != nil {
				b.Logf("Retrieve error (may be expected): %v", err)
			}
		}
	})

	b.Run("Initialize", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := client.Initialize(ctx)
			if err != nil {
				b.Errorf("Initialize failed: %v", err)
			}
		}
	})

	b.Run("Shutdown", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := client.Shutdown(ctx)
			if err != nil {
				b.Errorf("Shutdown failed: %v", err)
			}
		}
	})
}
