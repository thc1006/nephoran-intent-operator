
package llm



import (

	"context"

	"crypto/sha256"

	"encoding/hex"

	"encoding/json"

	"fmt"

	"log/slog"

	"sync"

	"time"

)



// MultiLevelCache implements a multi-tier caching system with Redis and in-memory layers.

type MultiLevelCache struct {

	l1Cache *InMemoryCache

	l2Cache RedisCache

	config  *MultiLevelCacheConfig

	logger  *slog.Logger

	metrics *MultiLevelCacheMetrics

	mutex   sync.RWMutex

}



// MultiLevelCacheConfig holds configuration for the multi-level cache.

type MultiLevelCacheConfig struct {

	// L1 Cache (In-Memory).

	L1MaxSize        int           `json:"l1_max_size"`

	L1TTL            time.Duration `json:"l1_ttl"`

	L1EvictionPolicy string        `json:"l1_eviction_policy"` // LRU, LFU, FIFO



	// L2 Cache (Redis).

	RedisEnabled  bool          `json:"redis_enabled"`

	RedisAddr     string        `json:"redis_addr"`

	RedisPassword string        `json:"redis_password"`

	RedisDB       int           `json:"redis_db"`

	L2TTL         time.Duration `json:"l2_ttl"`

	RedisTimeout  time.Duration `json:"redis_timeout"`



	// General settings.

	CompressionEnabled   bool   `json:"compression_enabled"`

	CompressionThreshold int    `json:"compression_threshold"`

	SerializationFormat  string `json:"serialization_format"` // json, msgpack, protobuf



	// Performance settings.

	MaxKeySize       int   `json:"max_key_size"`

	MaxValueSize     int64 `json:"max_value_size"`

	EnableStatistics bool  `json:"enable_statistics"`



	// Prefixes for different cache types.

	RAGPrefix     string `json:"rag_prefix"`

	LLMPrefix     string `json:"llm_prefix"`

	ContextPrefix string `json:"context_prefix"`

	PromptPrefix  string `json:"prompt_prefix"`

}



// MultiLevelCacheMetrics tracks cache performance across levels.

type MultiLevelCacheMetrics struct {

	// L1 Metrics.

	L1Hits      int64 `json:"l1_hits"`

	L1Misses    int64 `json:"l1_misses"`

	L1Sets      int64 `json:"l1_sets"`

	L1Evictions int64 `json:"l1_evictions"`

	L1Size      int64 `json:"l1_size"`



	// L2 Metrics.

	L2Hits   int64 `json:"l2_hits"`

	L2Misses int64 `json:"l2_misses"`

	L2Sets   int64 `json:"l2_sets"`

	L2Errors int64 `json:"l2_errors"`



	// Combined metrics.

	TotalHits      int64         `json:"total_hits"`

	TotalMisses    int64         `json:"total_misses"`

	OverallHitRate float64       `json:"overall_hit_rate"`

	AverageGetTime time.Duration `json:"average_get_time"`

	AverageSetTime time.Duration `json:"average_set_time"`



	// Size metrics.

	TotalMemoryUsage int64     `json:"total_memory_usage"`

	LastUpdated      time.Time `json:"last_updated"`



	mutex sync.RWMutex

}



// MultiLevelCacheEntry represents a cached item with metadata.

type MultiLevelCacheEntry struct {

	Key          string                 `json:"key"`

	Value        interface{}            `json:"value"`

	CreatedAt    time.Time              `json:"created_at"`

	ExpiresAt    time.Time              `json:"expires_at"`

	AccessCount  int64                  `json:"access_count"`

	LastAccessed time.Time              `json:"last_accessed"`

	Size         int64                  `json:"size"`

	Compressed   bool                   `json:"compressed"`

	Metadata     map[string]interface{} `json:"metadata"`

}



// InMemoryCache implements L1 in-memory cache.

type InMemoryCache struct {

	data      map[string]*MultiLevelCacheEntry

	capacity  int

	evictList *EvictionList

	mutex     sync.RWMutex

}



// RedisCache interface for L2 Redis cache operations.

type RedisCache interface {

	Get(ctx context.Context, key string) ([]byte, error)

	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	Del(ctx context.Context, key string) error

	Exists(ctx context.Context, key string) (bool, error)

	TTL(ctx context.Context, key string) (time.Duration, error)

	Close() error

}



// EvictionList implements LRU eviction policy.

type EvictionList struct {

	head *EvictionNode

	tail *EvictionNode

	size int

}



// EvictionNode represents a node in the eviction list.

type EvictionNode struct {

	key  string

	prev *EvictionNode

	next *EvictionNode

}



// NewMultiLevelCache creates a new multi-level cache.

func NewMultiLevelCache(config *MultiLevelCacheConfig, redisCache RedisCache) *MultiLevelCache {

	if config == nil {

		config = getDefaultMultiLevelCacheConfig()

	}



	return &MultiLevelCache{

		l1Cache: NewInMemoryCache(config.L1MaxSize),

		l2Cache: redisCache,

		config:  config,

		logger:  slog.Default().With("component", "multi-level-cache"),

		metrics: &MultiLevelCacheMetrics{LastUpdated: time.Now()},

	}

}



// getDefaultMultiLevelCacheConfig returns default cache configuration.

func getDefaultMultiLevelCacheConfig() *MultiLevelCacheConfig {

	return &MultiLevelCacheConfig{

		L1MaxSize:            1000,

		L1TTL:                15 * time.Minute,

		L1EvictionPolicy:     "LRU",

		RedisEnabled:         false,

		RedisAddr:            "localhost:6379",

		RedisPassword:        "",

		RedisDB:              0,

		L2TTL:                time.Hour,

		RedisTimeout:         5 * time.Second,

		CompressionEnabled:   true,

		CompressionThreshold: 1024,

		SerializationFormat:  "json",

		MaxKeySize:           256,

		MaxValueSize:         10 * 1024 * 1024, // 10MB

		EnableStatistics:     true,

		RAGPrefix:            "rag:",

		LLMPrefix:            "llm:",

		ContextPrefix:        "ctx:",

		PromptPrefix:         "prompt:",

	}

}



// Get retrieves a value from the cache (tries L1 first, then L2).

func (mlc *MultiLevelCache) Get(ctx context.Context, key string) (interface{}, bool, error) {

	startTime := time.Now()

	defer func() {

		mlc.updateMetrics(func(m *MultiLevelCacheMetrics) {

			m.AverageGetTime = (m.AverageGetTime*time.Duration(m.TotalHits+m.TotalMisses-1) + time.Since(startTime)) / time.Duration(m.TotalHits+m.TotalMisses)

		})

	}()



	// Validate key.

	if err := mlc.validateKey(key); err != nil {

		return nil, false, err

	}



	// Try L1 cache first.

	if value, found := mlc.l1Cache.Get(key); found {

		mlc.updateMetrics(func(m *MultiLevelCacheMetrics) {

			m.L1Hits++

			m.TotalHits++

		})



		mlc.logger.Debug("L1 cache hit", "key", key)

		return value, true, nil

	}



	mlc.updateMetrics(func(m *MultiLevelCacheMetrics) {

		m.L1Misses++

	})



	// Try L2 cache (Redis) if enabled.

	if mlc.config.RedisEnabled && mlc.l2Cache != nil {

		data, err := mlc.l2Cache.Get(ctx, key)

		if err == nil && data != nil {

			// Deserialize the data.

			var entry MultiLevelCacheEntry

			if err := json.Unmarshal(data, &entry); err != nil {

				mlc.logger.Warn("Failed to deserialize L2 cache entry", "key", key, "error", err)

			} else {

				// Check if entry is expired.

				if time.Now().Before(entry.ExpiresAt) {

					// Store in L1 cache for faster future access.

					mlc.l1Cache.Set(key, &entry, mlc.config.L1TTL)



					mlc.updateMetrics(func(m *MultiLevelCacheMetrics) {

						m.L2Hits++

						m.TotalHits++

					})



					mlc.logger.Debug("L2 cache hit", "key", key)

					return entry.Value, true, nil

				}

			}

		} else if err != nil {

			mlc.updateMetrics(func(m *MultiLevelCacheMetrics) {

				m.L2Errors++

			})

			mlc.logger.Warn("L2 cache error", "key", key, "error", err)

		}



		mlc.updateMetrics(func(m *MultiLevelCacheMetrics) {

			m.L2Misses++

		})

	}



	mlc.updateMetrics(func(m *MultiLevelCacheMetrics) {

		m.TotalMisses++

	})



	mlc.logger.Debug("Cache miss", "key", key)

	return nil, false, nil

}



// Set stores a value in both L1 and L2 caches.

func (mlc *MultiLevelCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {

	startTime := time.Now()

	defer func() {

		mlc.updateMetrics(func(m *MultiLevelCacheMetrics) {

			setCount := m.L1Sets

			if mlc.config.RedisEnabled {

				setCount = m.L1Sets + m.L2Sets

			}

			m.AverageSetTime = (m.AverageSetTime*time.Duration(setCount-1) + time.Since(startTime)) / time.Duration(setCount)

		})

	}()



	// Validate inputs.

	if err := mlc.validateKey(key); err != nil {

		return err

	}

	if err := mlc.validateValue(value); err != nil {

		return err

	}



	// Calculate size.

	size := mlc.calculateSize(value)



	// Create cache entry.

	entry := &MultiLevelCacheEntry{

		Key:          key,

		Value:        value,

		CreatedAt:    time.Now(),

		ExpiresAt:    time.Now().Add(ttl),

		AccessCount:  0,

		LastAccessed: time.Now(),

		Size:         size,

		Compressed:   false,

		Metadata:     make(map[string]interface{}),

	}



	// Store in L1 cache.

	mlc.l1Cache.Set(key, entry, ttl)

	mlc.updateMetrics(func(m *MultiLevelCacheMetrics) {

		m.L1Sets++

	})



	// Store in L2 cache (Redis) if enabled.

	if mlc.config.RedisEnabled && mlc.l2Cache != nil {

		// Serialize the entry.

		data, err := json.Marshal(entry)

		if err != nil {

			mlc.logger.Warn("Failed to serialize cache entry for L2", "key", key, "error", err)

		} else {

			// Compress if enabled and beneficial.

			if mlc.config.CompressionEnabled && len(data) > mlc.config.CompressionThreshold {

				// In a real implementation, you would compress the data here.

				entry.Compressed = true

			}



			// Store in Redis with L2 TTL.

			l2TTL := mlc.config.L2TTL

			if ttl < l2TTL {

				l2TTL = ttl

			}



			if err := mlc.l2Cache.Set(ctx, key, data, l2TTL); err != nil {

				mlc.updateMetrics(func(m *MultiLevelCacheMetrics) {

					m.L2Errors++

				})

				mlc.logger.Warn("Failed to store in L2 cache", "key", key, "error", err)

			} else {

				mlc.updateMetrics(func(m *MultiLevelCacheMetrics) {

					m.L2Sets++

				})

			}

		}

	}



	mlc.logger.Debug("Cache set", "key", key, "size", size, "ttl", ttl)

	return nil

}



// Delete removes a key from both cache levels.

func (mlc *MultiLevelCache) Delete(ctx context.Context, key string) error {

	if err := mlc.validateKey(key); err != nil {

		return err

	}



	// Delete from L1 cache.

	mlc.l1Cache.Delete(key)



	// Delete from L2 cache if enabled.

	if mlc.config.RedisEnabled && mlc.l2Cache != nil {

		if err := mlc.l2Cache.Del(ctx, key); err != nil {

			mlc.updateMetrics(func(m *MultiLevelCacheMetrics) {

				m.L2Errors++

			})

			mlc.logger.Warn("Failed to delete from L2 cache", "key", key, "error", err)

		}

	}



	mlc.logger.Debug("Cache delete", "key", key)

	return nil

}



// GetRAG retrieves RAG-specific cached data.

func (mlc *MultiLevelCache) GetRAG(ctx context.Context, query string) (interface{}, bool, error) {

	key := mlc.buildKey(mlc.config.RAGPrefix, query)

	return mlc.Get(ctx, key)

}



// SetRAG stores RAG-specific data.

func (mlc *MultiLevelCache) SetRAG(ctx context.Context, query string, value interface{}, ttl time.Duration) error {

	key := mlc.buildKey(mlc.config.RAGPrefix, query)

	return mlc.Set(ctx, key, value, ttl)

}



// GetLLMResponse retrieves cached LLM response.

func (mlc *MultiLevelCache) GetLLMResponse(ctx context.Context, prompt, modelName string) (interface{}, bool, error) {

	key := mlc.buildKey(mlc.config.LLMPrefix, prompt, modelName)

	return mlc.Get(ctx, key)

}



// SetLLMResponse stores LLM response.

func (mlc *MultiLevelCache) SetLLMResponse(ctx context.Context, prompt, modelName string, response interface{}, ttl time.Duration) error {

	key := mlc.buildKey(mlc.config.LLMPrefix, prompt, modelName)

	return mlc.Set(ctx, key, response, ttl)

}



// GetContext retrieves cached context.

func (mlc *MultiLevelCache) GetContext(ctx context.Context, contextID string) (interface{}, bool, error) {

	key := mlc.buildKey(mlc.config.ContextPrefix, contextID)

	return mlc.Get(ctx, key)

}



// SetContext stores context.

func (mlc *MultiLevelCache) SetContext(ctx context.Context, contextID string, context interface{}, ttl time.Duration) error {

	key := mlc.buildKey(mlc.config.ContextPrefix, contextID)

	return mlc.Set(ctx, key, context, ttl)

}



// GetPrompt retrieves cached prompt.

func (mlc *MultiLevelCache) GetPrompt(ctx context.Context, promptHash string) (interface{}, bool, error) {

	key := mlc.buildKey(mlc.config.PromptPrefix, promptHash)

	return mlc.Get(ctx, key)

}



// SetPrompt stores prompt.

func (mlc *MultiLevelCache) SetPrompt(ctx context.Context, promptHash string, prompt interface{}, ttl time.Duration) error {

	key := mlc.buildKey(mlc.config.PromptPrefix, promptHash)

	return mlc.Set(ctx, key, prompt, ttl)

}



// buildKey creates a cache key from components.

func (mlc *MultiLevelCache) buildKey(prefix string, components ...string) string {

	// Create a hash of the components for consistent key generation.

	hasher := sha256.New()

	for _, component := range components {

		hasher.Write([]byte(component))

	}

	hash := hex.EncodeToString(hasher.Sum(nil))[:16] // Use first 16 chars



	return fmt.Sprintf("%s%s", prefix, hash)

}



// validateKey validates cache key.

func (mlc *MultiLevelCache) validateKey(key string) error {

	if key == "" {

		return fmt.Errorf("cache key cannot be empty")

	}

	if len(key) > mlc.config.MaxKeySize {

		return fmt.Errorf("cache key too long: %d > %d", len(key), mlc.config.MaxKeySize)

	}

	return nil

}



// validateValue validates cache value.

func (mlc *MultiLevelCache) validateValue(value interface{}) error {

	if value == nil {

		return fmt.Errorf("cache value cannot be nil")

	}



	size := mlc.calculateSize(value)

	if size > mlc.config.MaxValueSize {

		return fmt.Errorf("cache value too large: %d > %d", size, mlc.config.MaxValueSize)

	}



	return nil

}



// calculateSize estimates the size of a value.

func (mlc *MultiLevelCache) calculateSize(value interface{}) int64 {

	// Simple size estimation based on JSON serialization.

	data, err := json.Marshal(value)

	if err != nil {

		return 0

	}

	return int64(len(data))

}



// updateMetrics safely updates cache metrics.

func (mlc *MultiLevelCache) updateMetrics(updater func(*MultiLevelCacheMetrics)) {

	mlc.metrics.mutex.Lock()

	defer mlc.metrics.mutex.Unlock()



	updater(mlc.metrics)



	// Update derived metrics.

	mlc.metrics.OverallHitRate = float64(mlc.metrics.TotalHits) / float64(mlc.metrics.TotalHits+mlc.metrics.TotalMisses)

	mlc.metrics.LastUpdated = time.Now()

}



// GetMetrics returns current cache metrics.

func (mlc *MultiLevelCache) GetMetrics() *MultiLevelCacheMetrics {

	mlc.metrics.mutex.RLock()

	defer mlc.metrics.mutex.RUnlock()



	// Update L1 size.

	mlc.metrics.L1Size = int64(mlc.l1Cache.Size())



	// Create a copy without the mutex.

	metrics := &MultiLevelCacheMetrics{

		L1Hits:           mlc.metrics.L1Hits,

		L1Misses:         mlc.metrics.L1Misses,

		L1Sets:           mlc.metrics.L1Sets,

		L1Evictions:      mlc.metrics.L1Evictions,

		L1Size:           mlc.metrics.L1Size,

		L2Hits:           mlc.metrics.L2Hits,

		L2Misses:         mlc.metrics.L2Misses,

		L2Sets:           mlc.metrics.L2Sets,

		L2Errors:         mlc.metrics.L2Errors,

		TotalHits:        mlc.metrics.TotalHits,

		TotalMisses:      mlc.metrics.TotalMisses,

		OverallHitRate:   mlc.metrics.OverallHitRate,

		AverageGetTime:   mlc.metrics.AverageGetTime,

		AverageSetTime:   mlc.metrics.AverageSetTime,

		TotalMemoryUsage: mlc.metrics.TotalMemoryUsage,

		LastUpdated:      mlc.metrics.LastUpdated,

	}

	return metrics

}



// GetStats returns detailed cache statistics.

func (mlc *MultiLevelCache) GetStats() map[string]interface{} {

	metrics := mlc.GetMetrics()



	return map[string]interface{}{

		"l1_stats": map[string]interface{}{

			"hits":      metrics.L1Hits,

			"misses":    metrics.L1Misses,

			"sets":      metrics.L1Sets,

			"evictions": metrics.L1Evictions,

			"size":      metrics.L1Size,

			"hit_rate":  float64(metrics.L1Hits) / float64(metrics.L1Hits+metrics.L1Misses),

		},

		"l2_stats": map[string]interface{}{

			"hits":    metrics.L2Hits,

			"misses":  metrics.L2Misses,

			"sets":    metrics.L2Sets,

			"errors":  metrics.L2Errors,

			"enabled": mlc.config.RedisEnabled,

			"hit_rate": func() float64 {

				if metrics.L2Hits+metrics.L2Misses == 0 {

					return 0

				}

				return float64(metrics.L2Hits) / float64(metrics.L2Hits+metrics.L2Misses)

			}(),

		},

		"overall_stats": map[string]interface{}{

			"total_hits":       metrics.TotalHits,

			"total_misses":     metrics.TotalMisses,

			"overall_hit_rate": metrics.OverallHitRate,

			"average_get_time": metrics.AverageGetTime.String(),

			"average_set_time": metrics.AverageSetTime.String(),

			"memory_usage":     metrics.TotalMemoryUsage,

			"last_updated":     metrics.LastUpdated,

		},

	}

}



// Clear clears all cache levels.

func (mlc *MultiLevelCache) Clear(ctx context.Context) error {

	// Clear L1 cache.

	mlc.l1Cache.Clear()



	// Clear L2 cache (this would need to be implemented per Redis client).

	// For now, we'll just log that L2 should be cleared manually.

	if mlc.config.RedisEnabled {

		mlc.logger.Warn("L2 cache clear not implemented - manual Redis cleanup required")

	}



	// Reset metrics.

	mlc.updateMetrics(func(m *MultiLevelCacheMetrics) {

		*m = MultiLevelCacheMetrics{LastUpdated: time.Now()}

	})



	mlc.logger.Info("Cache cleared")

	return nil

}



// Close closes the cache and cleans up resources.

func (mlc *MultiLevelCache) Close() error {

	if mlc.l2Cache != nil {

		if err := mlc.l2Cache.Close(); err != nil {

			mlc.logger.Error("Failed to close L2 cache", "error", err)

		}

	}



	mlc.logger.Info("Multi-level cache closed")

	return nil

}



// NewInMemoryCache creates a new in-memory cache.

func NewInMemoryCache(capacity int) *InMemoryCache {

	return &InMemoryCache{

		data:      make(map[string]*MultiLevelCacheEntry),

		capacity:  capacity,

		evictList: NewEvictionList(),

	}

}



// Get retrieves an item from in-memory cache.

func (imc *InMemoryCache) Get(key string) (interface{}, bool) {

	imc.mutex.RLock()

	defer imc.mutex.RUnlock()



	entry, exists := imc.data[key]

	if !exists {

		return nil, false

	}



	// Check expiration.

	if time.Now().After(entry.ExpiresAt) {

		// Entry expired, remove it.

		delete(imc.data, key)

		imc.evictList.Remove(key)

		return nil, false

	}



	// Update access information.

	entry.AccessCount++

	entry.LastAccessed = time.Now()



	// Move to front of eviction list (LRU).

	imc.evictList.MoveToFront(key)



	return entry.Value, true

}



// Set stores an item in in-memory cache.

func (imc *InMemoryCache) Set(key string, value interface{}, ttl time.Duration) {

	imc.mutex.Lock()

	defer imc.mutex.Unlock()



	// Check if we need to evict.

	if len(imc.data) >= imc.capacity {

		// Evict least recently used.

		evictKey := imc.evictList.RemoveTail()

		if evictKey != "" {

			delete(imc.data, evictKey)

		}

	}



	// Create new entry.

	entry := &MultiLevelCacheEntry{

		Key:          key,

		Value:        value,

		CreatedAt:    time.Now(),

		ExpiresAt:    time.Now().Add(ttl),

		AccessCount:  0,

		LastAccessed: time.Now(),

		Size:         int64(len(fmt.Sprintf("%v", value))), // Simple size estimation

	}



	imc.data[key] = entry

	imc.evictList.AddToFront(key)

}



// Delete removes an item from in-memory cache.

func (imc *InMemoryCache) Delete(key string) {

	imc.mutex.Lock()

	defer imc.mutex.Unlock()



	delete(imc.data, key)

	imc.evictList.Remove(key)

}



// Size returns the current size of the cache.

func (imc *InMemoryCache) Size() int {

	imc.mutex.RLock()

	defer imc.mutex.RUnlock()

	return len(imc.data)

}



// Clear removes all items from the cache.

func (imc *InMemoryCache) Clear() {

	imc.mutex.Lock()

	defer imc.mutex.Unlock()



	imc.data = make(map[string]*MultiLevelCacheEntry)

	imc.evictList = NewEvictionList()

}



// NewEvictionList creates a new eviction list.

func NewEvictionList() *EvictionList {

	head := &EvictionNode{}

	tail := &EvictionNode{}

	head.next = tail

	tail.prev = head



	return &EvictionList{

		head: head,

		tail: tail,

		size: 0,

	}

}



// AddToFront adds a key to the front of the eviction list.

func (el *EvictionList) AddToFront(key string) {

	node := &EvictionNode{key: key}

	node.next = el.head.next

	node.prev = el.head

	el.head.next.prev = node

	el.head.next = node

	el.size++

}



// Remove removes a key from the eviction list.

func (el *EvictionList) Remove(key string) {

	// Find and remove the node.

	current := el.head.next

	for current != el.tail {

		if current.key == key {

			current.prev.next = current.next

			current.next.prev = current.prev

			el.size--

			break

		}

		current = current.next

	}

}



// RemoveTail removes and returns the key at the tail (least recently used).

func (el *EvictionList) RemoveTail() string {

	if el.size == 0 {

		return ""

	}



	node := el.tail.prev

	key := node.key

	node.prev.next = el.tail

	el.tail.prev = node.prev

	el.size--



	return key

}



// MoveToFront moves a key to the front of the eviction list.

func (el *EvictionList) MoveToFront(key string) {

	el.Remove(key)

	el.AddToFront(key)

}

