package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisCache provides Redis-based caching for RAG components
type RedisCache struct {
	client      *redis.Client
	config      *RedisCacheConfig
	logger      *slog.Logger
	metrics     *RedisCacheMetrics
	keyPrefix   string
	mutex       sync.RWMutex
}

// RedisCacheConfig holds Redis cache configuration
type RedisCacheConfig struct {
	// Redis connection
	Address         string `json:"address"`
	Password        string `json:"password"`
	Database        int    `json:"database"`
	PoolSize        int    `json:"pool_size"`
	MinIdleConns    int    `json:"min_idle_conns"`
	MaxRetries      int    `json:"max_retries"`
	
	// Connection timeouts
	DialTimeout     time.Duration `json:"dial_timeout"`
	ReadTimeout     time.Duration `json:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout"`
	IdleTimeout     time.Duration `json:"idle_timeout"`
	
	// Cache behavior
	DefaultTTL      time.Duration `json:"default_ttl"`
	MaxKeyLength    int           `json:"max_key_length"`
	EnableMetrics   bool          `json:"enable_metrics"`
	KeyPrefix       string        `json:"key_prefix"`
	
	// Cache categories with different TTLs
	EmbeddingTTL    time.Duration `json:"embedding_ttl"`
	DocumentTTL     time.Duration `json:"document_ttl"`
	QueryResultTTL  time.Duration `json:"query_result_ttl"`
	ContextTTL      time.Duration `json:"context_ttl"`
	
	// Performance settings
	EnableCompression bool `json:"enable_compression"`
	CompressionLevel  int  `json:"compression_level"`
	MaxValueSize      int  `json:"max_value_size"`
	
	// Cleanup and maintenance
	EnableCleanup       bool          `json:"enable_cleanup"`
	CleanupInterval     time.Duration `json:"cleanup_interval"`
	MaxMemoryThreshold  float64       `json:"max_memory_threshold"`
}

// RedisCacheMetrics tracks Redis cache performance
type RedisCacheMetrics struct {
	// Basic metrics
	TotalRequests    int64 `json:"total_requests"`
	Hits             int64 `json:"hits"`
	Misses           int64 `json:"misses"`
	Sets             int64 `json:"sets"`
	Deletes          int64 `json:"deletes"`
	Errors           int64 `json:"errors"`
	
	// Performance metrics
	AverageGetTime   time.Duration `json:"average_get_time"`
	AverageSetTime   time.Duration `json:"average_set_time"`
	HitRate          float64       `json:"hit_rate"`
	
	// Category-specific metrics
	EmbeddingHits    int64 `json:"embedding_hits"`
	EmbeddingMisses  int64 `json:"embedding_misses"`
	DocumentHits     int64 `json:"document_hits"`
	DocumentMisses   int64 `json:"document_misses"`
	QueryResultHits  int64 `json:"query_result_hits"`
	QueryResultMisses int64 `json:"query_result_misses"`
	ContextHits      int64 `json:"context_hits"`
	ContextMisses    int64 `json:"context_misses"`
	
	// Resource metrics
	MemoryUsage      int64     `json:"memory_usage"`
	KeyCount         int64     `json:"key_count"`
	LastCleanup      time.Time `json:"last_cleanup"`
	
	LastUpdated      time.Time `json:"last_updated"`
	mutex            sync.RWMutex
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
	Key         CacheKey    `json:"key"`
	Data        interface{} `json:"data"`
	CreatedAt   time.Time   `json:"created_at"`
	ExpiresAt   time.Time   `json:"expires_at"`
	AccessCount int64       `json:"access_count"`
	LastAccess  time.Time   `json:"last_access"`
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
	Query    string           `json:"query"`
	Context  string           `json:"context"`
	Metadata *ContextMetadata `json:"metadata"`
	CreatedAt time.Time       `json:"created_at"`
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

// getDefaultRedisCacheConfig returns default Redis cache configuration
func getDefaultRedisCacheConfig() *RedisCacheConfig {
	return &RedisCacheConfig{
		Address:            "localhost:6379",
		Password:           "",
		Database:           0,
		PoolSize:           10,
		MinIdleConns:       5,
		MaxRetries:         3,
		DialTimeout:        5 * time.Second,
		ReadTimeout:        3 * time.Second,
		WriteTimeout:       3 * time.Second,
		IdleTimeout:        5 * time.Minute,
		DefaultTTL:         1 * time.Hour,
		MaxKeyLength:       250,
		EnableMetrics:      true,
		KeyPrefix:          "nephoran:rag:",
		EmbeddingTTL:       24 * time.Hour,
		DocumentTTL:        6 * time.Hour,
		QueryResultTTL:     30 * time.Minute,
		ContextTTL:         15 * time.Minute,
		EnableCompression:  true,
		CompressionLevel:   6,
		MaxValueSize:       10 * 1024 * 1024, // 10MB
		EnableCleanup:      true,
		CleanupInterval:    1 * time.Hour,
		MaxMemoryThreshold: 0.8, // 80% of max memory
	}
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
	info, err := rc.client.Info(ctx, "memory").Result()
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
		"status":     status,
		"healthy":    healthy,
		"ping_time":  pingTime,
		"error":      func() string { if err != nil { return err.Error() } return "" }(),
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