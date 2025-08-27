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
	"time"
)

// ClientInterface defines the interface for LLM clients
// This interface is shared between packages to avoid circular dependencies
type ClientInterface interface {
	ProcessIntent(ctx context.Context, prompt string) (string, error)
	ProcessIntentStream(ctx context.Context, prompt string, chunks chan<- *StreamingChunk) error
	GetSupportedModels() []string
	GetModelCapabilities(modelName string) (*ModelCapabilities, error)
	ValidateModel(modelName string) error
	EstimateTokens(text string) int
	GetMaxTokens(modelName string) int
	Close() error
}

// StreamingChunk represents a chunk of streamed response
type StreamingChunk struct {
	Content   string
	IsLast    bool
	Metadata  map[string]interface{}
	Timestamp time.Time
}

// ModelCapabilities describes what a model can do
type ModelCapabilities struct {
	MaxTokens         int                    `json:"max_tokens"`
	SupportsChat      bool                   `json:"supports_chat"`
	SupportsFunction  bool                   `json:"supports_function"`
	SupportsStreaming bool                   `json:"supports_streaming"`
	CostPerToken      float64                `json:"cost_per_token"`
	Features          map[string]interface{} `json:"features"`
}

// TelecomDocument represents a document in the telecom knowledge base
type TelecomDocument struct {
	ID              string                 `json:"id"`
	Title           string                 `json:"title"`
	Content         string                 `json:"content"`
	Source          string                 `json:"source"`
	Category        string                 `json:"category"`
	Version         string                 `json:"version"`
	Keywords        []string               `json:"keywords"`
	Language        string                 `json:"language"`
	DocumentType    string                 `json:"document_type"`
	NetworkFunction []string               `json:"network_function"`
	Technology      []string               `json:"technology"`
	UseCase         []string               `json:"use_case"`
	Confidence      float32                `json:"confidence"`
	Metadata        map[string]interface{} `json:"metadata"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// SearchResult represents a search result from the vector database
type SearchResult struct {
	Document *TelecomDocument       `json:"document"`
	Score    float32                `json:"score"`
	Distance float32                `json:"distance"`
	Metadata map[string]interface{} `json:"metadata"`
}

// SearchQuery represents a search query to the vector database
type SearchQuery struct {
	Query         string                 `json:"query"`
	Limit         int                    `json:"limit"`
	Filters       map[string]interface{} `json:"filters,omitempty"`
	HybridSearch  bool                   `json:"hybrid_search"`
	HybridAlpha   float32                `json:"hybrid_alpha"`
	UseReranker   bool                   `json:"use_reranker"`
	MinConfidence float32                `json:"min_confidence"`
	ExpandQuery   bool                   `json:"expand_query"`
}

// SearchResponse represents the response from a search operation
type SearchResponse struct {
	Results []*SearchResult `json:"results"`
	Took    int64           `json:"took"`
	Total   int64           `json:"total"`
}
