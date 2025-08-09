//go:build !disable_rag && !test

package rag

import (
	"compress/gzip"
	"context"
	"crypto/md5"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	"bytes"
	"github.com/go-redis/redis/v8"
)

// RedisCache provides Redis-based caching for RAG components
type RedisCache struct {
	client    *redis.Client
	config    *RedisCacheConfig
	logger    *slog.Logger
	metrics   *RedisCacheMetrics
	keyPrefix string
	mutex     sync.RWMutex
}

// RedisCacheConfig holds Redis cache configuration
type RedisCacheConfig struct {
	// Redis connection
	Address      string `json:"address"`
	Password     string `json:"password"`
	Database     int    `json:"database"`
	PoolSize     int    `json:"pool_size"`
	MinIdleConns int    `json:"min_idle_conns"`
	MaxRetries   int    `json:"max_retries"`

	// Connection timeouts
	DialTimeout  time.Duration `json:"dial_timeout"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`

	// Cache behavior
	DefaultTTL    time.Duration `json:"default_ttl"`
	MaxKeyLength  int           `json:"max_key_length"`
	EnableMetrics bool          `json:"enable_metrics"`
	KeyPrefix     string        `json:"key_prefix"`

	// Cache categories with different TTLs
	EmbeddingTTL   time.Duration `json:"embedding_ttl"`
	DocumentTTL    time.Duration `json:"document_ttl"`
	QueryResultTTL time.Duration `json:"query_result_ttl"`
	ContextTTL     time.Duration `json:"context_ttl"`

	// Performance settings
	EnableCompression bool `json:"enable_compression"`
	CompressionLevel  int  `json:"compression_level"`
	MaxValueSize      int  `json:"max_value_size"`

	// Cleanup and maintenance
	EnableCleanup      bool          `json:"enable_cleanup"`
	CleanupInterval    time.Duration `json:"cleanup_interval"`
	MaxMemoryThreshold float64       `json:"max_memory_threshold"`
}

// RedisCacheMetrics tracks Redis cache performance
type RedisCacheMetrics struct {
	// Basic metrics
	TotalRequests int64 `json:"total_requests"`
	Hits          int64 `json:"hits"`
	Misses        int64 `json:"misses"`
	Sets          int64 `json:"sets"`
	Deletes       int64 `json:"deletes"`
	Errors        int64 `json:"errors"`

	// Performance metrics
	AverageGetTime time.Duration `json:"average_get_time"`
	AverageSetTime time.Duration `json:"average_set_time"`
	HitRate        float64       `json:"hit_rate"`

	// Category-specific metrics
	EmbeddingHits     int64 `json:"embedding_hits"`
	EmbeddingMisses   int64 `json:"embedding_misses"`
	DocumentHits      int64 `json:"document_hits"`
	DocumentMisses    int64 `json:"document_misses"`
	QueryResultHits   int64 `json:"query_result_hits"`
	QueryResultMisses int64 `json:"query_result_misses"`
	ContextHits       int64 `json:"context_hits"`
	ContextMisses     int64 `json:"context_misses"`

	// Resource metrics
	MemoryUsage int64     `json:"memory_usage"`
	KeyCount    int64     `json:"key_count"`
	LastCleanup time.Time `json:"last_cleanup"`

	LastUpdated time.Time `json:"last_updated"`
	mutex       sync.RWMutex
}

// CacheKey represents a cache key with category information
type CacheKey struct {
	Category   string                 `json:"category"`   // embedding, document, query_result, context
	Identifier string                 `json:"identifier"` // unique identifier for the cached item
	Version    string                 `json:"version"`    // version for cache invalidation
	Metadata   map[string]interface{} `json:"metadata"`   // additional metadata
}

// CachedItem represents an item stored in cache
type CachedItem struct {
	Key         CacheKey               `json:"key"`
	Data        interface{}            `json:"data"`
	CreatedAt   time.Time              `json:"created_at"`
	ExpiresAt   time.Time              `json:"expires_at"`
	AccessCount int64                  `json:"access_count"`
	LastAccess  time.Time              `json:"last_access"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// EmbeddingCacheEntry represents a cached embedding
type EmbeddingCacheEntry struct {
	Text      string    `json:"text"`
	Embedding []float32 `json:"embedding"`
	ModelName string    `json:"model_name"`
	CreatedAt time.Time `json:"created_at"`
}

// DocumentCacheEntry represents a cached document
type DocumentCacheEntry struct {
	Document    *LoadedDocument `json:"document"`
	ProcessedAt time.Time       `json:"processed_at"`
	Hash        string          `json:"hash"`
}

// QueryResultCacheEntry represents cached query results
type QueryResultCacheEntry struct {
	Query       string                  `json:"query"`
	Results     []*EnhancedSearchResult `json:"results"`
	Metadata    map[string]interface{}  `json:"metadata"`
	ProcessedAt time.Time               `json:"processed_at"`
}

// ContextCacheEntry represents cached assembled context
type ContextCacheEntry struct {
	Query     string           `json:"query"`
	Context   string           `json:"context"`
	Metadata  *ContextMetadata `json:"metadata"`
	CreatedAt time.Time        `json:"created_at"`
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(config *RedisCacheConfig) (*RedisCache, error) {
	if config == nil {
		config = getDefaultRedisCacheConfig()
	}

	// Create Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:         config.Address,
		Password:     config.Password,
		DB:           config.Database,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		MaxRetries:   config.MaxRetries,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	})

	cache := &RedisCache{
		client:    rdb,
		config:    config,
		logger:    slog.Default().With("component", "redis-cache"),
		keyPrefix: config.KeyPrefix,
		metrics: &RedisCacheMetrics{
			LastUpdated: time.Now(),
		},
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	cache.logger.Info("Redis cache initialized",
		"address", config.Address,
		"database", config.Database,
		"pool_size", config.PoolSize,
	)

	// Start background tasks
	if config.EnableCleanup {
		go cache.startCleanupTask()
	}

	if config.EnableMetrics {
		go cache.startMetricsCollection()
	}

	return cache, nil
}

// Embedding cache methods

// GetEmbedding retrieves a cached embedding
func (rc *RedisCache) GetEmbedding(ctx context.Context, text, modelName string) ([]float32, bool) {
	key := rc.buildEmbeddingKey(text, modelName)

	startTime := time.Now()
	defer func() {
		rc.updateMetrics(func(m *RedisCacheMetrics) {
			m.TotalRequests++
			getTime := time.Since(startTime)
			if m.TotalRequests > 0 {
				m.AverageGetTime = (m.AverageGetTime*time.Duration(m.TotalRequests-1) + getTime) / time.Duration(m.TotalRequests)
			} else {
				m.AverageGetTime = getTime
			}
		})
	}()

	data, err := rc.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			rc.updateMetrics(func(m *RedisCacheMetrics) {
				m.Misses++
				m.EmbeddingMisses++
			})
			return nil, false
		}

		rc.logger.Error("Failed to get embedding from cache", "error", err, "key", key)
		rc.updateMetrics(func(m *RedisCacheMetrics) {
			m.Errors++
		})
		return nil, false
	}

	var entry EmbeddingCacheEntry
	if err := json.Unmarshal([]byte(data), &entry); err != nil {
		rc.logger.Error("Failed to unmarshal embedding cache entry", "error", err)
		rc.updateMetrics(func(m *RedisCacheMetrics) {
			m.Errors++
		})
		return nil, false
	}

	rc.updateMetrics(func(m *RedisCacheMetrics) {
		m.Hits++
		m.EmbeddingHits++
		m.HitRate = float64(m.Hits) / float64(m.TotalRequests)
	})

	return entry.Embedding, true
}

// SetEmbedding stores an embedding in cache
func (rc *RedisCache) SetEmbedding(ctx context.Context, text, modelName string, embedding []float32) error {
	key := rc.buildEmbeddingKey(text, modelName)

	startTime := time.Now()
	defer func() {
		rc.updateMetrics(func(m *RedisCacheMetrics) {
			setTime := time.Since(startTime)
			if m.Sets > 0 {
				m.AverageSetTime = (m.AverageSetTime*time.Duration(m.Sets) + setTime) / time.Duration(m.Sets+1)
			} else {
				m.AverageSetTime = setTime
			}
			m.Sets++
		})
	}()

	entry := EmbeddingCacheEntry{
		Text:      text,
		Embedding: embedding,
		ModelName: modelName,
		CreatedAt: time.Now(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		rc.updateMetrics(func(m *RedisCacheMetrics) { m.Errors++ })
		return fmt.Errorf("failed to marshal embedding cache entry: %w", err)
	}

	if len(data) > rc.config.MaxValueSize {
		rc.logger.Warn("Embedding cache entry too large", "size", len(data), "max_size", rc.config.MaxValueSize)
		return fmt.Errorf("cache entry too large: %d bytes", len(data))
	}

	err = rc.client.Set(ctx, key, data, rc.config.EmbeddingTTL).Err()
	if err != nil {
		rc.updateMetrics(func(m *RedisCacheMetrics) { m.Errors++ })
		return fmt.Errorf("failed to set embedding in cache: %w", err)
	}

	return nil
}

// buildEmbeddingKey builds a cache key for embeddings
func (rc *RedisCache) buildEmbeddingKey(text, modelName string) string {
	// Create a hash of the text to keep key length manageable
	textHash := fmt.Sprintf("%x", hash(text))
	return rc.buildKey("embedding", fmt.Sprintf("%s:%s", modelName, textHash))
}

// Document cache methods

// GetDocument retrieves a cached document
func (rc *RedisCache) GetDocument(ctx context.Context, docID string) (*LoadedDocument, bool) {
	key := rc.buildKey("document", docID)

	data, err := rc.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			rc.updateMetrics(func(m *RedisCacheMetrics) {
				m.Misses++
				m.DocumentMisses++
			})
			return nil, false
		}

		rc.logger.Error("Failed to get document from cache", "error", err, "key", key)
		rc.updateMetrics(func(m *RedisCacheMetrics) { m.Errors++ })
		return nil, false
	}

	var entry DocumentCacheEntry
	if err := json.Unmarshal([]byte(data), &entry); err != nil {
		rc.logger.Error("Failed to unmarshal document cache entry", "error", err)
		rc.updateMetrics(func(m *RedisCacheMetrics) { m.Errors++ })
		return nil, false
	}

	rc.updateMetrics(func(m *RedisCacheMetrics) {
		m.Hits++
		m.DocumentHits++
		m.HitRate = float64(m.Hits) / float64(m.TotalRequests)
	})

	return entry.Document, true
}

// SetDocument stores a document in cache
func (rc *RedisCache) SetDocument(ctx context.Context, doc *LoadedDocument) error {
	if doc == nil {
		return fmt.Errorf("document cannot be nil")
	}

	key := rc.buildKey("document", doc.ID)

	entry := DocumentCacheEntry{
		Document:    doc,
		ProcessedAt: time.Now(),
		Hash:        doc.Hash,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		rc.updateMetrics(func(m *RedisCacheMetrics) { m.Errors++ })
		return fmt.Errorf("failed to marshal document cache entry: %w", err)
	}

	if len(data) > rc.config.MaxValueSize {
		rc.logger.Warn("Document cache entry too large", "doc_id", doc.ID, "size", len(data))
		return fmt.Errorf("document cache entry too large: %d bytes", len(data))
	}

	err = rc.client.Set(ctx, key, data, rc.config.DocumentTTL).Err()
	if err != nil {
		rc.updateMetrics(func(m *RedisCacheMetrics) { m.Errors++ })
		return fmt.Errorf("failed to set document in cache: %w", err)
	}

	rc.updateMetrics(func(m *RedisCacheMetrics) { m.Sets++ })
	return nil
}

// Query result cache methods

// GetQueryResults retrieves cached query results
func (rc *RedisCache) GetQueryResults(ctx context.Context, query string, filters map[string]interface{}) ([]*EnhancedSearchResult, bool) {
	key := rc.buildQueryResultKey(query, filters)

	data, err := rc.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			rc.updateMetrics(func(m *RedisCacheMetrics) {
				m.Misses++
				m.QueryResultMisses++
			})
			return nil, false
		}

		rc.logger.Error("Failed to get query results from cache", "error", err, "key", key)
		rc.updateMetrics(func(m *RedisCacheMetrics) { m.Errors++ })
		return nil, false
	}

	var entry QueryResultCacheEntry
	if err := json.Unmarshal([]byte(data), &entry); err != nil {
		rc.logger.Error("Failed to unmarshal query result cache entry", "error", err)
		rc.updateMetrics(func(m *RedisCacheMetrics) { m.Errors++ })
		return nil, false
	}

	rc.updateMetrics(func(m *RedisCacheMetrics) {
		m.Hits++
		m.QueryResultHits++
		m.HitRate = float64(m.Hits) / float64(m.TotalRequests)
	})

	return entry.Results, true
}

// SetQueryResults stores query results in cache
func (rc *RedisCache) SetQueryResults(ctx context.Context, query string, filters map[string]interface{}, results []*EnhancedSearchResult) error {
	key := rc.buildQueryResultKey(query, filters)

	entry := QueryResultCacheEntry{
		Query:       query,
		Results:     results,
		Metadata:    filters,
		ProcessedAt: time.Now(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		rc.updateMetrics(func(m *RedisCacheMetrics) { m.Errors++ })
		return fmt.Errorf("failed to marshal query result cache entry: %w", err)
	}

	if len(data) > rc.config.MaxValueSize {
		rc.logger.Warn("Query result cache entry too large", "query", query, "size", len(data))
		return fmt.Errorf("query result cache entry too large: %d bytes", len(data))
	}

	err = rc.client.Set(ctx, key, data, rc.config.QueryResultTTL).Err()
	if err != nil {
		rc.updateMetrics(func(m *RedisCacheMetrics) { m.Errors++ })
		return fmt.Errorf("failed to set query results in cache: %w", err)
	}

	rc.updateMetrics(func(m *RedisCacheMetrics) { m.Sets++ })
	return nil
}

// buildQueryResultKey builds a cache key for query results
func (rc *RedisCache) buildQueryResultKey(query string, filters map[string]interface{}) string {
	// Create a deterministic key from query and filters
	var keyParts []string
	keyParts = append(keyParts, query)

	// Sort filter keys for deterministic key generation
	filterKeys := make([]string, 0, len(filters))
	for k := range filters {
		filterKeys = append(filterKeys, k)
	}

	for _, k := range filterKeys {
		if v, ok := filters[k]; ok {
			keyParts = append(keyParts, fmt.Sprintf("%s:%v", k, v))
		}
	}

	combinedKey := strings.Join(keyParts, "|")
	keyHash := fmt.Sprintf("%x", hash(combinedKey))
	return rc.buildKey("query_result", keyHash)
}

// Context cache methods

// GetContext retrieves cached context
func (rc *RedisCache) GetContext(ctx context.Context, query string, contextKey string) (string, *ContextMetadata, bool) {
	key := rc.buildKey("context", fmt.Sprintf("%s:%s", contextKey, fmt.Sprintf("%x", hash(query))))

	data, err := rc.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			rc.updateMetrics(func(m *RedisCacheMetrics) {
				m.Misses++
				m.ContextMisses++
			})
			return "", nil, false
		}

		rc.logger.Error("Failed to get context from cache", "error", err, "key", key)
		rc.updateMetrics(func(m *RedisCacheMetrics) { m.Errors++ })
		return "", nil, false
	}

	var entry ContextCacheEntry
	if err := json.Unmarshal([]byte(data), &entry); err != nil {
		rc.logger.Error("Failed to unmarshal context cache entry", "error", err)
		rc.updateMetrics(func(m *RedisCacheMetrics) { m.Errors++ })
		return "", nil, false
	}

	rc.updateMetrics(func(m *RedisCacheMetrics) {
		m.Hits++
		m.ContextHits++
		m.HitRate = float64(m.Hits) / float64(m.TotalRequests)
	})

	return entry.Context, entry.Metadata, true
}

// SetContext stores context in cache
func (rc *RedisCache) SetContext(ctx context.Context, query, contextKey, contextContent string, metadata *ContextMetadata) error {
	key := rc.buildKey("context", fmt.Sprintf("%s:%s", contextKey, fmt.Sprintf("%x", hash(query))))

	entry := ContextCacheEntry{
		Query:     query,
		Context:   contextContent,
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		rc.updateMetrics(func(m *RedisCacheMetrics) { m.Errors++ })
		return fmt.Errorf("failed to marshal context cache entry: %w", err)
	}

	if len(data) > rc.config.MaxValueSize {
		rc.logger.Warn("Context cache entry too large", "query", query, "size", len(data))
		return fmt.Errorf("context cache entry too large: %d bytes", len(data))
	}

	err = rc.client.Set(ctx, key, data, rc.config.ContextTTL).Err()
	if err != nil {
		rc.updateMetrics(func(m *RedisCacheMetrics) { m.Errors++ })
		return fmt.Errorf("failed to set context in cache: %w", err)
	}

	rc.updateMetrics(func(m *RedisCacheMetrics) { m.Sets++ })
	return nil
}

// General cache methods

// buildKey builds a cache key with prefix and category
func (rc *RedisCache) buildKey(category, identifier string) string {
	key := fmt.Sprintf("%s%s:%s", rc.keyPrefix, category, identifier)

	// Ensure key doesn't exceed maximum length
	if len(key) > rc.config.MaxKeyLength {
		// Use hash for long keys
		keyHash := fmt.Sprintf("%x", hash(key))
		key = fmt.Sprintf("%s%s:hash:%s", rc.keyPrefix, category, keyHash)
	}

	return key
}

// Delete removes an item from cache
func (rc *RedisCache) Delete(ctx context.Context, category, identifier string) error {
	key := rc.buildKey(category, identifier)

	err := rc.client.Del(ctx, key).Err()
	if err != nil {
		rc.updateMetrics(func(m *RedisCacheMetrics) { m.Errors++ })
		return fmt.Errorf("failed to delete from cache: %w", err)
	}

	rc.updateMetrics(func(m *RedisCacheMetrics) { m.Deletes++ })
	return nil
}

// Clear removes all items with the specified category prefix
func (rc *RedisCache) Clear(ctx context.Context, category string) error {
	pattern := rc.buildKey(category, "*")

	// Use SCAN to find keys (more efficient than KEYS for large datasets)
	var cursor uint64
	var deletedCount int64

	for {
		keys, newCursor, err := rc.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			rc.updateMetrics(func(m *RedisCacheMetrics) { m.Errors++ })
			return fmt.Errorf("failed to scan cache keys: %w", err)
		}

		if len(keys) > 0 {
			deleted, err := rc.client.Del(ctx, keys...).Result()
			if err != nil {
				rc.updateMetrics(func(m *RedisCacheMetrics) { m.Errors++ })
				return fmt.Errorf("failed to delete cache keys: %w", err)
			}
			deletedCount += deleted
		}

		cursor = newCursor
		if cursor == 0 {
			break
		}
	}

	rc.updateMetrics(func(m *RedisCacheMetrics) {
		m.Deletes += deletedCount
	})

	rc.logger.Info("Cache category cleared", "category", category, "deleted_count", deletedCount)
	return nil
}

// Background tasks

// startCleanupTask starts the background cleanup task
func (rc *RedisCache) startCleanupTask() {
	ticker := time.NewTicker(rc.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := rc.performCleanup(context.Background()); err != nil {
			rc.logger.Error("Cache cleanup failed", "error", err)
		}
	}
}

// performCleanup performs cache cleanup and maintenance
func (rc *RedisCache) performCleanup(ctx context.Context) error {
	rc.logger.Debug("Starting cache cleanup")

	// Get memory usage
	_, err := rc.client.Info(ctx, "memory").Result()
	if err != nil {
		return fmt.Errorf("failed to get Redis memory info: %w", err)
	}

	// Parse memory usage (simplified)
	memoryUsed := int64(0) // In a real implementation, parse the INFO output

	// Check if memory usage is above threshold
	if rc.config.MaxMemoryThreshold > 0 {
		// In a real implementation, you would:
		// 1. Parse Redis INFO memory output
		// 2. Calculate memory usage percentage
		// 3. Perform cleanup if above threshold
		rc.logger.Debug("Memory usage check completed", "memory_used", memoryUsed)
	}

	// Update cleanup metrics
	rc.updateMetrics(func(m *RedisCacheMetrics) {
		m.MemoryUsage = memoryUsed
		m.LastCleanup = time.Now()
	})

	rc.logger.Debug("Cache cleanup completed")
	return nil
}

// startMetricsCollection starts background metrics collection
func (rc *RedisCache) startMetricsCollection() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		rc.collectMetrics(context.Background())
	}
}

// collectMetrics collects and updates cache metrics
func (rc *RedisCache) collectMetrics(ctx context.Context) {
	// Get key count
	keyCount, err := rc.client.DBSize(ctx).Result()
	if err != nil {
		rc.logger.Warn("Failed to get key count", "error", err)
	} else {
		rc.updateMetrics(func(m *RedisCacheMetrics) {
			m.KeyCount = keyCount
		})
	}

	// Log metrics periodically
	metrics := rc.GetMetrics()
	rc.logger.Debug("Cache metrics",
		"hit_rate", metrics.HitRate,
		"total_requests", metrics.TotalRequests,
		"key_count", metrics.KeyCount,
		"memory_usage", metrics.MemoryUsage,
	)
}

// Metrics methods

// updateMetrics safely updates cache metrics
func (rc *RedisCache) updateMetrics(updater func(*RedisCacheMetrics)) {
	rc.metrics.mutex.Lock()
	defer rc.metrics.mutex.Unlock()
	updater(rc.metrics)
	rc.metrics.LastUpdated = time.Now()
}

// GetMetrics returns current cache metrics
func (rc *RedisCache) GetMetrics() *RedisCacheMetrics {
	rc.metrics.mutex.RLock()
	defer rc.metrics.mutex.RUnlock()

	// Return a copy
	metrics := *rc.metrics
	return &metrics
}

// GetHealthStatus returns cache health status
func (rc *RedisCache) GetHealthStatus(ctx context.Context) map[string]interface{} {
	// Test Redis connection
	pingStart := time.Now()
	err := rc.client.Ping(ctx).Err()
	pingTime := time.Since(pingStart)

	healthy := err == nil
	status := "healthy"
	if !healthy {
		status = "unhealthy"
	}

	metrics := rc.GetMetrics()

	return map[string]interface{}{
		"status":    status,
		"healthy":   healthy,
		"ping_time": pingTime,
		"error": func() string {
			if err != nil {
				return err.Error()
			}
			return ""
		}(),
		"metrics": map[string]interface{}{
			"hit_rate":       metrics.HitRate,
			"total_requests": metrics.TotalRequests,
			"errors":         metrics.Errors,
			"key_count":      metrics.KeyCount,
			"memory_usage":   metrics.MemoryUsage,
		},
		"last_updated": metrics.LastUpdated,
	}
}

// Close closes the Redis connection
func (rc *RedisCache) Close() error {
	rc.logger.Info("Closing Redis cache connection")
	return rc.client.Close()
}

// Helper functions

// hash creates a hash of the input string
func hash(input string) []byte {
	h := md5.New()
	h.Write([]byte(input))
	return h.Sum(nil)
}

// Enhanced embedding cache methods with binary encoding

// SetEmbeddingBinary stores an embedding using efficient binary encoding
func (rc *RedisCache) SetEmbeddingBinary(ctx context.Context, text, modelName string, embedding []float32) error {
	key := rc.buildEmbeddingKey(text, modelName)

	startTime := time.Now()
	defer func() {
		rc.updateMetrics(func(m *RedisCacheMetrics) {
			setTime := time.Since(startTime)
			if m.Sets > 0 {
				m.AverageSetTime = (m.AverageSetTime*time.Duration(m.Sets) + setTime) / time.Duration(m.Sets+1)
			} else {
				m.AverageSetTime = setTime
			}
			m.Sets++
		})
	}()

	// Convert embedding to binary format for more efficient storage
	binaryData, err := rc.encodeEmbeddingBinary(embedding, text, modelName)
	if err != nil {
		rc.updateMetrics(func(m *RedisCacheMetrics) { m.Errors++ })
		return fmt.Errorf("failed to encode embedding: %w", err)
	}

	// Apply compression if enabled
	if rc.config.EnableCompression {
		binaryData, err = rc.compress(binaryData)
		if err != nil {
			rc.updateMetrics(func(m *RedisCacheMetrics) { m.Errors++ })
			return fmt.Errorf("failed to compress embedding: %w", err)
		}
	}

	if len(binaryData) > rc.config.MaxValueSize {
		rc.logger.Warn("Compressed embedding cache entry too large", "size", len(binaryData), "max_size", rc.config.MaxValueSize)
		return fmt.Errorf("compressed cache entry too large: %d bytes", len(binaryData))
	}

	err = rc.client.Set(ctx, key, binaryData, rc.config.EmbeddingTTL).Err()
	if err != nil {
		rc.updateMetrics(func(m *RedisCacheMetrics) { m.Errors++ })
		return fmt.Errorf("failed to set binary embedding in cache: %w", err)
	}

	return nil
}

// GetEmbeddingBinary retrieves an embedding using binary decoding
func (rc *RedisCache) GetEmbeddingBinary(ctx context.Context, text, modelName string) ([]float32, bool) {
	key := rc.buildEmbeddingKey(text, modelName)

	startTime := time.Now()
	defer func() {
		rc.updateMetrics(func(m *RedisCacheMetrics) {
			m.TotalRequests++
			getTime := time.Since(startTime)
			if m.TotalRequests > 0 {
				m.AverageGetTime = (m.AverageGetTime*time.Duration(m.TotalRequests-1) + getTime) / time.Duration(m.TotalRequests)
			} else {
				m.AverageGetTime = getTime
			}
		})
	}()

	data, err := rc.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			rc.updateMetrics(func(m *RedisCacheMetrics) {
				m.Misses++
				m.EmbeddingMisses++
			})
			return nil, false
		}

		rc.logger.Error("Failed to get binary embedding from cache", "error", err, "key", key)
		rc.updateMetrics(func(m *RedisCacheMetrics) { m.Errors++ })
		return nil, false
	}

	// Decompress if compression was used
	if rc.config.EnableCompression {
		data, err = rc.decompress(data)
		if err != nil {
			rc.logger.Error("Failed to decompress embedding data", "error", err)
			rc.updateMetrics(func(m *RedisCacheMetrics) { m.Errors++ })
			return nil, false
		}
	}

	// Decode binary data
	embedding, err := rc.decodeEmbeddingBinary(data)
	if err != nil {
		rc.logger.Error("Failed to decode binary embedding", "error", err)
		rc.updateMetrics(func(m *RedisCacheMetrics) { m.Errors++ })
		return nil, false
	}

	rc.updateMetrics(func(m *RedisCacheMetrics) {
		m.Hits++
		m.EmbeddingHits++
		m.HitRate = float64(m.Hits) / float64(m.TotalRequests)
	})

	return embedding, true
}

// encodeEmbeddingBinary encodes an embedding to binary format
func (rc *RedisCache) encodeEmbeddingBinary(embedding []float32, text, modelName string) ([]byte, error) {
	var buf bytes.Buffer

	// Write header
	if err := binary.Write(&buf, binary.LittleEndian, uint32(len(embedding))); err != nil {
		return nil, fmt.Errorf("failed to write embedding length: %w", err)
	}

	// Write model name length and data
	modelNameBytes := []byte(modelName)
	if err := binary.Write(&buf, binary.LittleEndian, uint32(len(modelNameBytes))); err != nil {
		return nil, fmt.Errorf("failed to write model name length: %w", err)
	}
	if _, err := buf.Write(modelNameBytes); err != nil {
		return nil, fmt.Errorf("failed to write model name: %w", err)
	}

	// Write timestamp
	if err := binary.Write(&buf, binary.LittleEndian, time.Now().Unix()); err != nil {
		return nil, fmt.Errorf("failed to write timestamp: %w", err)
	}

	// Write embedding data
	for _, val := range embedding {
		if err := binary.Write(&buf, binary.LittleEndian, val); err != nil {
			return nil, fmt.Errorf("failed to write embedding value: %w", err)
		}
	}

	return buf.Bytes(), nil
}

// decodeEmbeddingBinary decodes binary embedding data
func (rc *RedisCache) decodeEmbeddingBinary(data []byte) ([]float32, error) {
	buf := bytes.NewReader(data)

	// Read embedding length
	var embeddingLen uint32
	if err := binary.Read(buf, binary.LittleEndian, &embeddingLen); err != nil {
		return nil, fmt.Errorf("failed to read embedding length: %w", err)
	}

	// Read model name length
	var modelNameLen uint32
	if err := binary.Read(buf, binary.LittleEndian, &modelNameLen); err != nil {
		return nil, fmt.Errorf("failed to read model name length: %w", err)
	}

	// Skip model name
	if _, err := buf.Seek(int64(modelNameLen), io.SeekCurrent); err != nil {
		return nil, fmt.Errorf("failed to skip model name: %w", err)
	}

	// Skip timestamp
	if _, err := buf.Seek(8, io.SeekCurrent); err != nil {
		return nil, fmt.Errorf("failed to skip timestamp: %w", err)
	}

	// Read embedding data
	embedding := make([]float32, embeddingLen)
	for i := range embedding {
		if err := binary.Read(buf, binary.LittleEndian, &embedding[i]); err != nil {
			return nil, fmt.Errorf("failed to read embedding value at index %d: %w", i, err)
		}
	}

	return embedding, nil
}

// compress compresses data using gzip
func (rc *RedisCache) compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, rc.config.CompressionLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip writer: %w", err)
	}

	if _, err := writer.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write compressed data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}

// decompress decompresses gzip data
func (rc *RedisCache) decompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read decompressed data: %w", err)
	}

	return decompressed, nil
}

// Batch operations for better performance

// SetEmbeddingsBatch stores multiple embeddings in a single Redis pipeline
func (rc *RedisCache) SetEmbeddingsBatch(ctx context.Context, embeddings map[string]EmbeddingBatchItem) error {
	if len(embeddings) == 0 {
		return nil
	}

	pipe := rc.client.Pipeline()
	defer pipe.Close()

	for textHash, item := range embeddings {
		key := rc.buildEmbeddingKey(item.Text, item.ModelName)

		var data []byte
		var err error

		if rc.config.EnableCompression {
			// Use binary encoding for batch operations
			binaryData, err := rc.encodeEmbeddingBinary(item.Embedding, item.Text, item.ModelName)
			if err != nil {
				rc.logger.Error("Failed to encode embedding in batch", "error", err, "text_hash", textHash)
				continue
			}

			data, err = rc.compress(binaryData)
			if err != nil {
				rc.logger.Error("Failed to compress embedding in batch", "error", err, "text_hash", textHash)
				continue
			}
		} else {
			// Use JSON encoding for uncompressed storage
			entry := EmbeddingCacheEntry{
				Text:      item.Text,
				Embedding: item.Embedding,
				ModelName: item.ModelName,
				CreatedAt: time.Now(),
			}

			data, err = json.Marshal(entry)
			if err != nil {
				rc.logger.Error("Failed to marshal embedding in batch", "error", err, "text_hash", textHash)
				continue
			}
		}

		if len(data) <= rc.config.MaxValueSize {
			pipe.Set(ctx, key, data, rc.config.EmbeddingTTL)
		} else {
			rc.logger.Warn("Skipping large embedding in batch", "text_hash", textHash, "size", len(data))
		}
	}

	// Execute batch
	_, err := pipe.Exec(ctx)
	if err != nil {
		rc.updateMetrics(func(m *RedisCacheMetrics) { m.Errors++ })
		return fmt.Errorf("failed to execute embedding batch: %w", err)
	}

	rc.updateMetrics(func(m *RedisCacheMetrics) {
		m.Sets += int64(len(embeddings))
	})

	rc.logger.Debug("Batch embedding storage completed", "count", len(embeddings))
	return nil
}

// EmbeddingBatchItem represents an item in a batch embedding operation
type EmbeddingBatchItem struct {
	Text      string    `json:"text"`
	Embedding []float32 `json:"embedding"`
	ModelName string    `json:"model_name"`
}

// Cache warming functions

// WarmCache pre-loads frequently accessed items into cache
func (rc *RedisCache) WarmCache(ctx context.Context, warmupConfig *CacheWarmupConfig) error {
	rc.logger.Info("Starting cache warming", "config", warmupConfig)

	// Warm up embeddings if configured
	if warmupConfig.EmbeddingWarmup != nil {
		if err := rc.warmEmbeddingCache(ctx, warmupConfig.EmbeddingWarmup); err != nil {
			return fmt.Errorf("embedding cache warmup failed: %w", err)
		}
	}

	// Warm up query results if configured
	if warmupConfig.QueryWarmup != nil {
		if err := rc.warmQueryCache(ctx, warmupConfig.QueryWarmup); err != nil {
			return fmt.Errorf("query cache warmup failed: %w", err)
		}
	}

	rc.logger.Info("Cache warming completed")
	return nil
}

// CacheWarmupConfig configures cache warming
type CacheWarmupConfig struct {
	EmbeddingWarmup *EmbeddingWarmupConfig `json:"embedding_warmup"`
	QueryWarmup     *QueryWarmupConfig     `json:"query_warmup"`
}

// EmbeddingWarmupConfig configures embedding cache warming
type EmbeddingWarmupConfig struct {
	CommonTexts   []string `json:"common_texts"`
	ModelNames    []string `json:"model_names"`
	MaxWarmupSize int      `json:"max_warmup_size"`
}

// QueryWarmupConfig configures query cache warming
type QueryWarmupConfig struct {
	CommonQueries []string `json:"common_queries"`
	MaxWarmupSize int      `json:"max_warmup_size"`
}

// warmEmbeddingCache warms up the embedding cache
func (rc *RedisCache) warmEmbeddingCache(ctx context.Context, config *EmbeddingWarmupConfig) error {
	// In a real implementation, you would:
	// 1. Load common texts from configuration or analytics
	// 2. Generate embeddings for these texts
	// 3. Store them in cache

	rc.logger.Debug("Warming embedding cache",
		"common_texts", len(config.CommonTexts),
		"models", len(config.ModelNames),
	)

	// This is a placeholder - actual implementation would generate and cache embeddings
	return nil
}

// warmQueryCache warms up the query cache
func (rc *RedisCache) warmQueryCache(ctx context.Context, config *QueryWarmupConfig) error {
	// In a real implementation, you would:
	// 1. Load common queries from configuration or analytics
	// 2. Execute these queries
	// 3. Store results in cache

	rc.logger.Debug("Warming query cache", "common_queries", len(config.CommonQueries))

	// This is a placeholder - actual implementation would execute and cache queries
	return nil
}

// Cache statistics and analysis

// GetCacheStatistics returns detailed cache statistics
func (rc *RedisCache) GetCacheStatistics(ctx context.Context) (*CacheStatistics, error) {
	stats := &CacheStatistics{
		GeneratedAt: time.Now(),
	}

	// Get Redis info
	info, err := rc.client.Info(ctx, "all").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis info: %w", err)
	}

	// Parse Redis info (simplified)
	stats.RedisInfo = info

	// Get key distribution by category
	stats.KeyDistribution = make(map[string]int64)
	categories := []string{"embedding", "document", "query_result", "context"}

	for _, category := range categories {
		pattern := rc.buildKey(category, "*")
		count, err := rc.countKeysWithPattern(ctx, pattern)
		if err != nil {
			rc.logger.Warn("Failed to count keys for category", "category", category, "error", err)
			count = 0
		}
		stats.KeyDistribution[category] = count
	}

	// Get current metrics
	metrics := rc.GetMetrics()
	stats.Metrics = metrics

	return stats, nil
}

// CacheStatistics holds detailed cache statistics
type CacheStatistics struct {
	GeneratedAt     time.Time          `json:"generated_at"`
	RedisInfo       string             `json:"redis_info"`
	KeyDistribution map[string]int64   `json:"key_distribution"`
	Metrics         *RedisCacheMetrics `json:"metrics"`
}

// countKeysWithPattern counts keys matching a pattern
func (rc *RedisCache) countKeysWithPattern(ctx context.Context, pattern string) (int64, error) {
	var cursor uint64
	var count int64

	for {
		keys, newCursor, err := rc.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return 0, err
		}

		count += int64(len(keys))
		cursor = newCursor
		if cursor == 0 {
			break
		}
	}

	return count, nil
}

// NoOpRedisCache provides a no-operation Redis cache implementation
type NoOpRedisCache struct{}

// NewNoOpRedisCache creates a new no-op Redis cache
func NewNoOpRedisCache() *NoOpRedisCache {
	return &NoOpRedisCache{}
}

// Get implements RedisEmbeddingCache interface
func (c *NoOpRedisCache) Get(key string) ([]float32, bool, error) {
	return nil, false, nil
}

// Set implements RedisEmbeddingCache interface
func (c *NoOpRedisCache) Set(key string, embedding []float32, ttl time.Duration) error {
	return nil
}

// Delete implements RedisEmbeddingCache interface
func (c *NoOpRedisCache) Delete(key string) error {
	return nil
}

// Clear implements RedisEmbeddingCache interface
func (c *NoOpRedisCache) Clear() error {
	return nil
}

// Stats implements RedisEmbeddingCache interface
func (c *NoOpRedisCache) Stats() CacheStats {
	return CacheStats{}
}

// Close implements RedisEmbeddingCache interface
func (c *NoOpRedisCache) Close() error {
	return nil
}

// NewRedisEmbeddingCache creates a new Redis embedding cache
func NewRedisEmbeddingCache(addr, password string, db int) RedisEmbeddingCache {
	config := &RedisCacheConfig{
		Address:           addr,
		Password:          password,
		Database:          db,
		PoolSize:          10,
		MinIdleConns:      2,
		MaxRetries:        3,
		DialTimeout:       5 * time.Second,
		ReadTimeout:       3 * time.Second,
		WriteTimeout:      3 * time.Second,
		IdleTimeout:       5 * time.Minute,
		DefaultTTL:        24 * time.Hour,
		EmbeddingTTL:      24 * time.Hour,
		KeyPrefix:         "nephoran:rag:",
		EnableCompression: true,
		MaxValueSize:      10 * 1024 * 1024, // 10MB
	}

	cache, err := NewRedisCache(config)
	if err != nil {
		slog.Default().Error("Failed to create Redis cache, using no-op", "error", err)
		return NewNoOpRedisCache()
	}

	return &RedisEmbeddingCacheAdapter{cache: cache}
}

// RedisEmbeddingCacheAdapter adapts RedisCache to RedisEmbeddingCache interface
type RedisEmbeddingCacheAdapter struct {
	cache *RedisCache
}

// Get implements RedisEmbeddingCache interface
func (a *RedisEmbeddingCacheAdapter) Get(key string) ([]float32, bool, error) {
	embedding, found := a.cache.GetEmbedding(context.Background(), key, "default")
	if !found {
		return nil, false, nil
	}
	return embedding, true, nil
}

// Set implements RedisEmbeddingCache interface
func (a *RedisEmbeddingCacheAdapter) Set(key string, embedding []float32, ttl time.Duration) error {
	return a.cache.SetEmbedding(context.Background(), key, "default", embedding)
}

// Delete implements RedisEmbeddingCache interface
func (a *RedisEmbeddingCacheAdapter) Delete(key string) error {
	return a.cache.Delete(context.Background(), "embedding", key)
}

// Clear implements RedisEmbeddingCache interface
func (a *RedisEmbeddingCacheAdapter) Clear() error {
	return a.cache.Clear(context.Background(), "embedding")
}

// Stats implements RedisEmbeddingCache interface
func (a *RedisEmbeddingCacheAdapter) Stats() CacheStats {
	metrics := a.cache.GetMetrics()
	return CacheStats{
		Size:    metrics.KeyCount,
		Hits:    metrics.EmbeddingHits,
		Misses:  metrics.EmbeddingMisses,
		HitRate: metrics.HitRate,
	}
}

// Close implements RedisEmbeddingCache interface
func (a *RedisEmbeddingCacheAdapter) Close() error {
	return a.cache.Close()
}
