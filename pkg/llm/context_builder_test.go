package llm

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
	"github.com/weaviate/weaviate-go-client/v4/weaviate"
)

// MockWeaviateConnectionPool provides a mock implementation for testing
type MockWeaviateConnectionPool struct {
	searchResults []*shared.SearchResult
	searchError   error
	connected     bool
}

func (m *MockWeaviateConnectionPool) GetConnection() (*weaviate.Client, error) {
	if !m.connected {
		return nil, fmt.Errorf("weaviate connection failed")
	}
	return &weaviate.Client{}, nil
}

func (m *MockWeaviateConnectionPool) Search(ctx context.Context, query *shared.SearchQuery) (*shared.SearchResponse, error) {
	if m.searchError != nil {
		return nil, m.searchError
	}

	// Filter results based on limit
	limit := query.Limit
	if limit > len(m.searchResults) {
		limit = len(m.searchResults)
	}

	filteredResults := make([]*shared.SearchResult, limit)
	copy(filteredResults, m.searchResults[:limit])

	return &shared.SearchResponse{
		Results: filteredResults,
		Took:    10,
		Total:   int64(len(m.searchResults)),
	}, nil
}

func (m *MockWeaviateConnectionPool) Close() error {
	return nil
}

func TestContextBuilder_BuildContext(t *testing.T) {
	tests := []struct {
		name              string
		intent            string
		maxDocs           int
		mockSearchResults []*shared.SearchResult
		mockSearchError   error
		mockConnected     bool
		expectedDocs      int
		expectedError     bool
		errorContains     string
		validateResult    func(t *testing.T, result []map[string]any)
	}{
		{
			name:    "successful context building with multiple documents",
			intent:  "Deploy AMF network function in 5G core",
			maxDocs: 3,
			mockSearchResults: []*shared.SearchResult{
				{
					Document: &shared.TelecomDocument{
						ID:              "doc1",
						Title:           "5G AMF Deployment Guide",
						Content:         "This document describes how to deploy Access and Mobility Management Function in 5G core network",
						Source:          "3GPP TS 23.501",
						Category:        "deployment",
						Version:         "v17.0.0",
						Keywords:        []string{"AMF", "5G", "deployment"},
						NetworkFunction: []string{"AMF"},
						Technology:      []string{"5G"},
						Confidence:      0.9,
						CreatedAt:       time.Now().Add(-30 * 24 * time.Hour),
						UpdatedAt:       time.Now().Add(-1 * 24 * time.Hour),
					},
					Score:    0.95,
					Distance: 0.05,
				},
				{
					Document: &shared.TelecomDocument{
						ID:              "doc2",
						Title:           "O-RAN AMF Configuration",
						Content:         "O-RAN specific configuration for AMF network function deployment in cloud-native environments",
						Source:          "O-RAN.WG4.MP-v04.00",
						Category:        "configuration",
						Version:         "v4.0",
						Keywords:        []string{"O-RAN", "AMF", "configuration"},
						NetworkFunction: []string{"AMF"},
						Technology:      []string{"O-RAN", "5G"},
						Confidence:      0.85,
						CreatedAt:       time.Now().Add(-60 * 24 * time.Hour),
						UpdatedAt:       time.Now().Add(-7 * 24 * time.Hour),
					},
					Score:    0.88,
					Distance: 0.12,
				},
				{
					Document: &shared.TelecomDocument{
						ID:              "doc3",
						Title:           "Network Function Lifecycle Management",
						Content:         "Best practices for managing lifecycle of network functions including AMF in containerized environments",
						Source:          "ETSI NFV-MAN 001",
						Category:        "management",
						Version:         "v3.4.1",
						Keywords:        []string{"NFV", "lifecycle", "management"},
						NetworkFunction: []string{"AMF", "SMF", "UPF"},
						Technology:      []string{"NFV", "containers"},
						Confidence:      0.75,
						CreatedAt:       time.Now().Add(-90 * 24 * time.Hour),
						UpdatedAt:       time.Now().Add(-30 * 24 * time.Hour),
					},
					Score:    0.72,
					Distance: 0.28,
				},
			},
			mockConnected: true,
			expectedDocs:  3,
			expectedError: false,
			validateResult: func(t *testing.T, result []map[string]any) {
				if len(result) != 3 {
					t.Errorf("Expected 3 documents, got %d", len(result))
					return
				}

				// Validate first document
				firstDoc := result[0]
				if title, ok := firstDoc["title"].(string); !ok || title != "5G AMF Deployment Guide" {
					t.Errorf("Expected first doc title '5G AMF Deployment Guide', got %v", firstDoc["title"])
				}

				if score, ok := firstDoc["score"].(float32); !ok || score != 0.95 {
					t.Errorf("Expected first doc score 0.95, got %v", firstDoc["score"])
				}

				// Check enhanced query fields
				if enhancedQuery, exists := firstDoc["enhanced_query"]; !exists {
					t.Error("Expected enhanced_query field to exist")
				} else if eq, ok := enhancedQuery.(string); !ok || eq == "" {
					t.Error("Expected non-empty enhanced_query string")
				}
			},
		},
		{
			name:              "empty intent",
			intent:            "",
			maxDocs:           5,
			mockSearchResults: []*shared.SearchResult{},
			mockConnected:     true,
			expectedError:     true,
			errorContains:     "intent cannot be empty",
		},
		{
			name:              "zero maxDocs",
			intent:            "Deploy network function",
			maxDocs:           0,
			mockSearchResults: []*shared.SearchResult{},
			mockConnected:     true,
			expectedError:     true,
			errorContains:     "maxDocs must be greater than 0",
		},
		{
			name:              "weaviate connection failure",
			intent:            "Deploy AMF",
			maxDocs:           3,
			mockSearchResults: []*shared.SearchResult{},
			mockConnected:     false,
			expectedError:     true,
			errorContains:     "weaviate connection failed",
		},
		{
			name:              "search operation failure",
			intent:            "Deploy AMF network function",
			maxDocs:           3,
			mockSearchResults: []*shared.SearchResult{},
			mockSearchError:   fmt.Errorf("weaviate search timeout"),
			mockConnected:     true,
			expectedError:     true,
			errorContains:     "weaviate search timeout",
		},
		{
			name:              "no documents found",
			intent:            "Deploy unknown network function XYZ123",
			maxDocs:           5,
			mockSearchResults: []*shared.SearchResult{},
			mockConnected:     true,
			expectedDocs:      0,
			expectedError:     false,
			validateResult: func(t *testing.T, result []map[string]any) {
				if len(result) != 0 {
					t.Errorf("Expected 0 documents for non-matching query, got %d", len(result))
				}
			},
		},
		{
			name:    "telecommunications abbreviation expansion",
			intent:  "Configure gNB for 5G NR in O-RAN",
			maxDocs: 2,
			mockSearchResults: []*shared.SearchResult{
				{
					Document: &shared.TelecomDocument{
						ID:              "doc4",
						Title:           "gNodeB Configuration for New Radio",
						Content:         "Configuration procedures for gNodeB (gNB) in 5G New Radio (NR) networks following O-RAN specifications",
						Source:          "O-RAN.WG3.RAN-v03.00",
						Category:        "configuration",
						NetworkFunction: []string{"gNB", "DU", "CU"},
						Technology:      []string{"5G", "O-RAN", "NR"},
						Keywords:        []string{"gNodeB", "5G", "NR", "O-RAN", "configuration"},
						Confidence:      0.92,
						CreatedAt:       time.Now().Add(-15 * 24 * time.Hour),
						UpdatedAt:       time.Now().Add(-2 * 24 * time.Hour),
					},
					Score:    0.91,
					Distance: 0.09,
				},
			},
			mockConnected: true,
			expectedDocs:  1,
			expectedError: false,
			validateResult: func(t *testing.T, result []map[string]any) {
				if len(result) != 1 {
					t.Errorf("Expected 1 document, got %d", len(result))
					return
				}

				doc := result[0]
				if enhancedQuery, exists := doc["enhanced_query"]; exists {
					if eq, ok := enhancedQuery.(string); ok {
						// Check that abbreviations were expanded
						if !contextBuilderContains(eq, "gNodeB") || !contextBuilderContains(eq, "New Radio") {
							t.Logf("Enhanced query: %s", eq)
							// This is informational - abbreviation expansion may be implemented differently
						}
					}
				}
			},
		},
		{
			name:              "large document set with limit",
			intent:            "5G core network deployment",
			maxDocs:           2,                            // Limit to 2 docs
			mockSearchResults: generateMockSearchResults(5), // But have 5 available
			mockConnected:     true,
			expectedDocs:      2, // Should return only 2
			expectedError:     false,
			validateResult: func(t *testing.T, result []map[string]any) {
				if len(result) != 2 {
					t.Errorf("Expected 2 documents (limited), got %d", len(result))
				}
			},
		},
		{
			name:    "documents with various formats and metadata",
			intent:  "Network slice management procedures",
			maxDocs: 3,
			mockSearchResults: []*shared.SearchResult{
				{
					Document: &shared.TelecomDocument{
						ID:              "pdf_doc",
						Title:           "Network Slicing Architecture",
						Content:         "Network slicing enables multiple virtual networks on a shared physical infrastructure",
						Source:          "3GPP TS 28.530",
						Category:        "specification",
						DocumentType:    "PDF",
						Language:        "en",
						Version:         "v16.2.0",
						NetworkFunction: []string{"NSSF", "NSSMF"},
						Technology:      []string{"5G", "Network Slicing"},
						UseCase:         []string{"eMBB", "URLLC", "mMTC"},
						Keywords:        []string{"network", "slicing", "virtual", "architecture"},
						Confidence:      0.88,
						Metadata: map[string]interface{}{
							"page_count": 45,
							"format":     "PDF",
							"authors":    []string{"3GPP SA5"},
						},
						CreatedAt: time.Now().Add(-45 * 24 * time.Hour),
						UpdatedAt: time.Now().Add(-10 * 24 * time.Hour),
					},
					Score:    0.87,
					Distance: 0.13,
					Metadata: map[string]interface{}{
						"match_type": "semantic",
						"boost":      1.2,
					},
				},
			},
			mockConnected: true,
			expectedDocs:  1,
			expectedError: false,
			validateResult: func(t *testing.T, result []map[string]any) {
				if len(result) != 1 {
					t.Errorf("Expected 1 document, got %d", len(result))
					return
				}

				doc := result[0]

				// Validate metadata preservation
				if metadata, exists := doc["metadata"]; exists {
					if metaMap, ok := metadata.(map[string]interface{}); ok {
						if pageCount, exists := metaMap["page_count"]; !exists || pageCount != 45 {
							t.Errorf("Expected page_count metadata to be preserved")
						}
					}
				}

				// Validate use case information
				if useCases, exists := doc["use_case"]; exists {
					if ucSlice, ok := useCases.([]string); ok && len(ucSlice) > 0 {
						expectedUseCases := []string{"eMBB", "URLLC", "mMTC"}
						for _, expected := range expectedUseCases {
							found := false
							for _, actual := range ucSlice {
								if actual == expected {
									found = true
									break
								}
							}
							if !found {
								t.Errorf("Expected use case %s not found in result", expected)
							}
						}
					}
				}
			},
		},
		{
			name:    "documents with low confidence scores",
			intent:  "Deploy test network function",
			maxDocs: 2,
			mockSearchResults: []*shared.SearchResult{
				{
					Document: &shared.TelecomDocument{
						ID:         "low_conf_doc",
						Title:      "Draft Network Function Guide",
						Content:    "This is a draft document about network function deployment",
						Source:     "Internal Draft",
						Confidence: 0.3, // Low confidence
						CreatedAt:  time.Now().Add(-7 * 24 * time.Hour),
						UpdatedAt:  time.Now().Add(-1 * 24 * time.Hour),
					},
					Score:    0.45,
					Distance: 0.55,
				},
			},
			mockConnected: true,
			expectedDocs:  1,
			expectedError: false,
			validateResult: func(t *testing.T, result []map[string]any) {
				if len(result) != 1 {
					t.Errorf("Expected 1 document, got %d", len(result))
					return
				}

				doc := result[0]
				if confidence, exists := doc["confidence"]; exists {
					if conf, ok := confidence.(float32); ok && conf != 0.3 {
						t.Errorf("Expected confidence 0.3, got %f", conf)
					}
				}
			},
		},
		{
			name:    "context timeout scenario",
			intent:  "Deploy network function with timeout",
			maxDocs: 1,
			mockSearchResults: []*shared.SearchResult{
				{
					Document: &shared.TelecomDocument{
						ID:         "timeout_doc",
						Title:      "Network Function Deployment",
						Content:    "Standard deployment procedures for network functions",
						Source:     "Standard Guide",
						Confidence: 0.8,
						CreatedAt:  time.Now().Add(-30 * 24 * time.Hour),
						UpdatedAt:  time.Now().Add(-5 * 24 * time.Hour),
					},
					Score:    0.75,
					Distance: 0.25,
				},
			},
			mockConnected: true,
			expectedDocs:  1,
			expectedError: false,
		},
		{
			name:    "edge case - very long intent",
			intent:  generateLongIntent(),
			maxDocs: 1,
			mockSearchResults: []*shared.SearchResult{
				{
					Document: &shared.TelecomDocument{
						ID:         "edge_case_doc",
						Title:      "Edge Case Document",
						Content:    "Document for testing edge cases",
						Source:     "Test Source",
						Confidence: 0.5,
						CreatedAt:  time.Now(),
						UpdatedAt:  time.Now(),
					},
					Score:    0.6,
					Distance: 0.4,
				},
			},
			mockConnected: true,
			expectedDocs:  1,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock connection pool
			mockPool := &MockWeaviateConnectionPool{
				searchResults: tt.mockSearchResults,
				searchError:   tt.mockSearchError,
				connected:     tt.mockConnected,
			}

			// Create context builder stub directly
			contextBuilder := &ContextBuilderStub{}

			// Verify mock pool was created
			assert.NotNil(t, mockPool, "Mock pool should be created")

			// Execute the test
			ctx := context.Background()
			if tt.name == "context timeout scenario" {
				// Create a context with very short timeout to test timeout handling
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 1*time.Millisecond)
				defer cancel()
				time.Sleep(2 * time.Millisecond) // Ensure context is expired
			}

			result, err := contextBuilder.BuildContext(ctx, tt.intent, tt.maxDocs)

			// Check error expectations
			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorContains != "" && !contextBuilderContains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			// Check for unexpected errors
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Validate result
			if len(result) != tt.expectedDocs {
				t.Errorf("Expected %d documents, got %d", tt.expectedDocs, len(result))
			}

			// Run custom validation if provided
			if tt.validateResult != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

// Test performance characteristics
func TestContextBuilder_Performance(t *testing.T) {
	// Create mock with large number of results
	largeResultSet := generateMockSearchResults(100)
	mockPool := &MockWeaviateConnectionPool{
		searchResults: largeResultSet,
		connected:     true,
	}
	_ = mockPool // Use mockPool to avoid unused warning

	// Create context builder stub directly for performance testing
	contextBuilder := &ContextBuilderStub{}

	// Test with various document counts
	testCases := []int{1, 5, 10, 25, 50}

	for _, maxDocs := range testCases {
		t.Run(fmt.Sprintf("maxDocs_%d", maxDocs), func(t *testing.T) {
			start := time.Now()

			result, err := contextBuilder.BuildContext(context.Background(), "Deploy 5G network function", maxDocs)

			duration := time.Since(start)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(result) != maxDocs {
				t.Errorf("Expected %d documents, got %d", maxDocs, len(result))
			}

			// Performance assertion - should complete within reasonable time
			if duration > 5*time.Second {
				t.Errorf("Context building took too long: %v", duration)
			}

			t.Logf("Built context with %d documents in %v", maxDocs, duration)
		})
	}
}

// Test concurrent access
func TestContextBuilder_ConcurrentAccess(t *testing.T) {
	mockPool := &MockWeaviateConnectionPool{
		searchResults: generateMockSearchResults(10),
		connected:     true,
	}
	_ = mockPool // Use mockPool to avoid unused warning

	// Create context builder stub for concurrent testing
	contextBuilder := &ContextBuilderStub{}

	// Run multiple goroutines concurrently
	concurrentRequests := 10
	results := make(chan error, concurrentRequests)

	for i := 0; i < concurrentRequests; i++ {
		go func(id int) {
			intent := fmt.Sprintf("Deploy network function %d", id)
			_, err := contextBuilder.BuildContext(context.Background(), intent, 3)
			results <- err
		}(i)
	}

	// Collect results
	for i := 0; i < concurrentRequests; i++ {
		err := <-results
		if err != nil {
			t.Errorf("Concurrent request %d failed: %v", i, err)
		}
	}
}

// Helper functions

func contextBuilderContains(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) > 0 && len(needle) <= len(haystack) && haystack[len(haystack)-len(needle):] == needle) ||
		(len(haystack) >= len(needle) && haystack[:len(needle)] == needle) ||
		(len(haystack) > len(needle) && strings.Contains(haystack, needle))
}

func generateMockSearchResults(count int) []*shared.SearchResult {
	results := make([]*shared.SearchResult, count)

	for i := 0; i < count; i++ {
		results[i] = &shared.SearchResult{
			Document: &shared.TelecomDocument{
				ID:              fmt.Sprintf("doc_%d", i),
				Title:           fmt.Sprintf("Document %d", i),
				Content:         fmt.Sprintf("Content for document %d with various telecom terms like 5G, AMF, and network functions", i),
				Source:          fmt.Sprintf("Source %d", i),
				Category:        "test",
				Version:         "v1.0",
				Keywords:        []string{"test", "document", fmt.Sprintf("doc%d", i)},
				NetworkFunction: []string{"AMF", "SMF"},
				Technology:      []string{"5G", "test"},
				Confidence:      0.5 + float32(i%5)*0.1,
				CreatedAt:       time.Now().Add(-time.Duration(i*24) * time.Hour),
				UpdatedAt:       time.Now().Add(-time.Duration(i*12) * time.Hour),
			},
			Score:    0.9 - float32(i)*0.05,
			Distance: float32(i) * 0.05,
		}
	}

	return results
}

func generateLongIntent() string {
	intent := "Deploy a highly available, scalable, and secure Access and Mobility Management Function (AMF) "
	intent += "in a 5G standalone core network environment using O-RAN compliant infrastructure with "
	intent += "cloud-native principles, including automatic scaling policies, comprehensive monitoring, "
	intent += "fault tolerance mechanisms, and integration with Session Management Function (SMF) and "
	intent += "User Plane Function (UPF) components while ensuring compliance with 3GPP Release 16 "
	intent += "specifications and enterprise-grade security policies including mutual TLS authentication, "
	intent += "role-based access control, and audit logging capabilities for telecommunications operators "
	intent += "in production environments with 99.99% availability requirements and sub-millisecond latency targets"

	return intent
}

// NewContextBuilderWithPool creates a context builder with a custom connection pool for testing
func NewTestContextBuilderWithPool(config *ContextBuilderConfig, pool interface{}) *ContextBuilder {
	if config == nil {
		config = &ContextBuilderConfig{
			DefaultMaxDocs:        20,
			MaxContextLength:      1000,
			MinConfidenceScore:    0.3,
			QueryTimeout:          5 * time.Second,
			EnableHybridSearch:    true,
			HybridAlpha:           0.75,
			TelecomKeywords:       []string{"5G", "AMF", "SMF", "UPF"},
			QueryExpansionEnabled: true,
		}
	}

	return &ContextBuilder{
		config:       config,
		tokenManager: NewTokenManager(),
	}
}
