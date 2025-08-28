/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package types

import (
	"context"
	"sync"
	"time"

	"github.com/weaviate/weaviate-go-client/v4/weaviate"
)

// ClientInterface defines the interface for LLM clients
// This interface is shared between packages to avoid circular dependencies
type ClientInterface interface {
	ProcessIntent(ctx context.Context, prompt string) (string, error)
	ProcessIntentStream(ctx context.Context, prompt string, chunks chan<- *StreamingChunk) error
	GetSupportedModels() []string
	GetModelCapabilities(modelName string) (*ModelCapabilities, error)
	SupportsStreaming() bool
	GetMaxTokens() int
	Close() error
}

// StreamingChunk represents a chunk of streaming response
type StreamingChunk struct {
	Content  string    `json:"content"`
	Index    int       `json:"index"`
	IsLast   bool      `json:"is_last"`
	Metadata Metadata  `json:"metadata"`
	Time     time.Time `json:"time"`
}

// ModelCapabilities describes what a model can do
type ModelCapabilities struct {
	Name          string   `json:"name"`
	MaxTokens     int      `json:"max_tokens"`
	SupportsChat  bool     `json:"supports_chat"`
	SupportsTools bool     `json:"supports_tools"`
	InputTypes    []string `json:"input_types"`
	OutputTypes   []string `json:"output_types"`
	Description   string   `json:"description"`
}

// Metadata represents flexible metadata
type Metadata map[string]interface{}

// TelecomDocument represents a document in the telecom domain
type TelecomDocument struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Type        string    `json:"type"`
	Source      string    `json:"source"`
	Category    string    `json:"category"`
	Tags        []string  `json:"tags"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Version     string    `json:"version"`
	Metadata    Metadata  `json:"metadata"`
	Embedding   []float32 `json:"embedding,omitempty"`
	Confidence  float64   `json:"confidence,omitempty"`
}

// SearchResult represents a search result from RAG or other search operations
type SearchResult struct {
	Document *TelecomDocument       `json:"document"`
	Score    float32                `json:"score"`
	Distance float32                `json:"distance"`
	Metadata map[string]interface{} `json:"metadata"`
}

// SearchQuery represents a search query with various parameters
type SearchQuery struct {
	Query         string                 `json:"query"`
	Limit         int                    `json:"limit"`
	Filters       map[string]interface{} `json:"filters"`
	HybridSearch  bool                   `json:"hybrid_search"`
	UseReranker   bool                   `json:"use_reranker"`
	MinConfidence float32                `json:"min_confidence"`
	HybridAlpha   *float32               `json:"hybrid_alpha,omitempty"`
	ExpandQuery   bool                   `json:"expand_query"`
}

// SearchResponse represents a search response
type SearchResponse struct {
	Results []*SearchResult `json:"results"`
	Took    time.Duration   `json:"took"`
	Total   int64           `json:"total"`
	Query   string          `json:"query"`
}

// PooledConnection represents a connection in a connection pool
type PooledConnection struct {
	// Core connection fields
	ID          string                 `json:"id"`
	CreatedAt   time.Time              `json:"created_at"`
	LastUsed    time.Time              `json:"last_used"`
	IsHealthy   bool                   `json:"is_healthy"`
	UsageCount  int64                  `json:"usage_count"`
	InUse       bool                   `json:"in_use"`
	
	// Connection clients
	Client         *weaviate.Client       `json:"-"`
	HTTPClient     interface{}            `json:"http_client,omitempty"`
	FastHTTPClient interface{}            `json:"fast_http_client,omitempty"`
	Connection     interface{}            `json:"connection,omitempty"`
	
	// Pool management
	Metadata    map[string]interface{} `json:"metadata"`
	MaxUses     int64                  `json:"max_uses"`
	MaxIdleTime time.Duration          `json:"max_idle_time"`
	
	// Thread safety
	Mu sync.RWMutex `json:"-"`
}

// TokenUsage represents token usage information
type TokenUsage struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	EstimatedCost    float64 `json:"estimated_cost"`
}

// BenchmarkTestResult represents the result of a benchmark test
type BenchmarkTestResult struct {
	Name         string        `json:"name"`
	Duration     time.Duration `json:"duration"`
	Operations   int64         `json:"operations"`
	BytesPerOp   int64         `json:"bytes_per_op"`
	AllocsPerOp  int64         `json:"allocs_per_op"`
	MemoryUsed   int64         `json:"memory_used"`
	Success      bool          `json:"success"`
	Error        string        `json:"error,omitempty"`
	Metadata     Metadata      `json:"metadata"`
	StartedAt    time.Time     `json:"started_at"`
	CompletedAt  time.Time     `json:"completed_at"`
}