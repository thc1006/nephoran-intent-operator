//go:build !disable_rag && !test

package rag

import (
	"context"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// Type aliases for compatibility with shared types
type TelecomDocument = shared.TelecomDocument
// SearchResult is already defined in enhanced_rag_integration.go
type SearchQuery = shared.SearchQuery

// Additional types not available in shared package

// WeaviateClient minimal definition for compatibility
type WeaviateClient struct {
	enabled bool
}

// Search method stub for WeaviateClient
func (wc *WeaviateClient) Search(ctx context.Context, query *SearchQuery) (*SearchResponse, error) {
	return &SearchResponse{
		Results:     []*SearchResult{},
		Total:       0,
		Took:        0,
		Query:       query.Query,
		ProcessedAt: time.Now(),
		Metadata:    make(map[string]interface{}),
	}, nil
}

// GetHealthStatus method stub for WeaviateClient  
func (wc *WeaviateClient) GetHealthStatus() *WeaviateHealthStatus {
	return &WeaviateHealthStatus{
		IsHealthy: true,
		LastCheck: time.Now(),
		Version:   "stub",
		Details:   "stub implementation",
	}
}

// WeaviateHealthStatus minimal definition
type WeaviateHealthStatus struct {
	IsHealthy bool      `json:"is_healthy"`
	LastCheck time.Time `json:"last_check"`
	Version   string    `json:"version"`
	Details   string    `json:"details"`
}

// WeaviateConfig minimal definition
type WeaviateConfig struct {
	Host   string `json:"host"`
	Scheme string `json:"scheme"`
	APIKey string `json:"api_key"`
}

// SearchResponse minimal definition  
type SearchResponse struct {
	Results     []*SearchResult        `json:"results"`
	Total       int                    `json:"total"`
	Took        time.Duration          `json:"took"`
	Query       string                 `json:"query"`
	ProcessedAt time.Time              `json:"processed_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}