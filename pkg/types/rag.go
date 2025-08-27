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
	"time"
)

// BatchSearchRequest represents a batch of search requests
type BatchSearchRequest struct {
	Queries           []*SearchQuery         `json:"queries"`
	MaxConcurrency    int                    `json:"max_concurrency"`
	EnableAggregation bool                   `json:"enable_aggregation"`
	DeduplicationKey  string                 `json:"deduplication_key"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// BatchSearchResponse represents the response from batch search
type BatchSearchResponse struct {
	Results             []*SearchResponse      `json:"results"`
	AggregatedResults   []*SearchResult        `json:"aggregated_results"`
	TotalProcessingTime time.Duration          `json:"total_processing_time"`
	ParallelQueries     int                    `json:"parallel_queries"`
	CacheHits           int                    `json:"cache_hits"`
	Metadata            map[string]interface{} `json:"metadata"`
}

// EmbeddingCacheInterface defines the interface for embedding caches
type EmbeddingCacheInterface interface {
	Get(key string) ([]float32, bool)
	Set(key string, embedding []float32, ttl time.Duration) error
	Delete(key string) error
	Clear() error
	Stats() EmbeddingCacheStats
}

// EmbeddingCacheStats represents embedding cache statistics
type EmbeddingCacheStats struct {
	Size    int64   `json:"size"`
	Hits    int64   `json:"hits"`
	Misses  int64   `json:"misses"`
	HitRate float64 `json:"hit_rate"`
}

// EmbeddingCacheEntry represents a cached embedding entry
type EmbeddingCacheEntry struct {
	Text        string        `json:"text"`
	Vector      []float32     `json:"vector"`
	CreatedAt   time.Time     `json:"created_at"`
	AccessCount int64         `json:"access_count"`
	TTL         time.Duration `json:"ttl"`
}
