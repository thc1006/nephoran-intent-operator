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

package webui

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/auth"
)

// Cache provides thread-safe in-memory caching with TTL support.

type Cache struct {
	items map[string]*cacheItem

	mutex sync.RWMutex

	maxSize int

	defaultTTL time.Duration

	stats *cacheStatsInternal
}

// cacheItem represents a cached item with expiration.

type cacheItem struct {
	value interface{}

	expiration time.Time

	hits int64

	lastAccess time.Time

	size int64 // approximate size in bytes

}

// CacheStats provides cache performance metrics.

type CacheStats struct {
	Hits int64

	Misses int64

	Sets int64

	Evictions int64

	ExpiredItems int64

	TotalItems int64

	TotalSize int64

	AvgItemSize float64
}

// cacheStatsInternal contains CacheStats with an internal mutex for thread safety.

type cacheStatsInternal struct {
	CacheStats

	mutex sync.RWMutex
}

// CacheConfig holds cache configuration.

type CacheConfig struct {
	MaxSize int

	DefaultTTL time.Duration

	CleanupInterval time.Duration

	MaxItemSize int64
}

// NewCache creates a new cache instance.

func NewCache(maxSize int, defaultTTL time.Duration) *Cache {

	cache := &Cache{

		items: make(map[string]*cacheItem),

		maxSize: maxSize,

		defaultTTL: defaultTTL,

		stats: &cacheStatsInternal{},
	}

	// Start cleanup goroutine.

	go cache.cleanupExpiredItems()

	return cache

}

// Get retrieves a value from cache.

func (c *Cache) Get(key string) (interface{}, bool) {

	c.mutex.RLock()

	defer c.mutex.RUnlock()

	item, exists := c.items[key]

	if !exists {

		c.updateStats(func(s *CacheStats) {

			s.Misses++

		})

		return nil, false

	}

	// Check expiration.

	if time.Now().After(item.expiration) {

		// Item expired, remove it.

		c.mutex.RUnlock()

		c.mutex.Lock()

		delete(c.items, key)

		c.mutex.Unlock()

		c.mutex.RLock()

		c.updateStats(func(s *CacheStats) {

			s.Misses++

			s.ExpiredItems++

			s.TotalItems--

			s.TotalSize -= item.size

		})

		return nil, false

	}

	// Update access statistics.

	item.hits++

	item.lastAccess = time.Now()

	c.updateStats(func(s *CacheStats) {

		s.Hits++

	})

	return item.value, true

}

// Set stores a value in cache with default TTL.

func (c *Cache) Set(key string, value interface{}) {

	c.SetWithTTL(key, value, c.defaultTTL)

}

// SetWithTTL stores a value in cache with custom TTL.

func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {

	c.mutex.Lock()

	defer c.mutex.Unlock()

	// Calculate approximate size.

	itemSize := c.estimateItemSize(key, value)

	// Check if we need to evict items.

	c.evictIfNecessary(itemSize)

	expiration := time.Now().Add(ttl)

	// Check if item already exists.

	if existingItem, exists := c.items[key]; exists {

		// Update existing item.

		c.updateStats(func(s *CacheStats) {

			s.TotalSize -= existingItem.size

			s.TotalSize += itemSize

		})

	} else {

		// New item.

		c.updateStats(func(s *CacheStats) {

			s.TotalItems++

			s.TotalSize += itemSize

		})

	}

	c.items[key] = &cacheItem{

		value: value,

		expiration: expiration,

		hits: 0,

		lastAccess: time.Now(),

		size: itemSize,
	}

	c.updateStats(func(s *CacheStats) {

		s.Sets++

		c.calculateAvgItemSize()

	})

}

// Delete removes an item from cache.

func (c *Cache) Delete(key string) bool {

	c.mutex.Lock()

	defer c.mutex.Unlock()

	if item, exists := c.items[key]; exists {

		delete(c.items, key)

		c.updateStats(func(s *CacheStats) {

			s.TotalItems--

			s.TotalSize -= item.size

			c.calculateAvgItemSize()

		})

		return true

	}

	return false

}

// Invalidate removes all items matching a key prefix.

func (c *Cache) Invalidate(keyPrefix string) int {

	c.mutex.Lock()

	defer c.mutex.Unlock()

	var removedCount int

	var removedSize int64

	for key, item := range c.items {

		if strings.HasPrefix(key, keyPrefix) {

			delete(c.items, key)

			removedCount++

			removedSize += item.size

		}

	}

	if removedCount > 0 {

		c.updateStats(func(s *CacheStats) {

			s.TotalItems -= int64(removedCount)

			s.TotalSize -= removedSize

			s.Evictions += int64(removedCount)

			c.calculateAvgItemSize()

		})

	}

	return removedCount

}

// Clear removes all items from cache.

func (c *Cache) Clear() {

	c.mutex.Lock()

	defer c.mutex.Unlock()

	itemCount := len(c.items)

	c.items = make(map[string]*cacheItem)

	c.updateStats(func(s *CacheStats) {

		s.TotalItems = 0

		s.TotalSize = 0

		s.AvgItemSize = 0

		s.Evictions += int64(itemCount)

	})

}

// Size returns the number of items in cache.

func (c *Cache) Size() int {

	c.mutex.RLock()

	defer c.mutex.RUnlock()

	return len(c.items)

}

// Stats returns cache statistics.

func (c *Cache) Stats() *CacheStats {

	c.stats.mutex.RLock()

	defer c.stats.mutex.RUnlock()

	// Return a copy of the stats without the mutex.

	return &CacheStats{

		Hits: c.stats.CacheStats.Hits,

		Misses: c.stats.CacheStats.Misses,

		Sets: c.stats.CacheStats.Sets,

		Evictions: c.stats.CacheStats.Evictions,

		ExpiredItems: c.stats.CacheStats.ExpiredItems,

		TotalItems: c.stats.CacheStats.TotalItems,

		TotalSize: c.stats.CacheStats.TotalSize,

		AvgItemSize: c.stats.CacheStats.AvgItemSize,
	}

}

// HitRate returns the cache hit rate as a percentage.

func (c *Cache) HitRate() float64 {

	stats := c.Stats()

	total := stats.Hits + stats.Misses

	if total == 0 {

		return 0

	}

	return float64(stats.Hits) / float64(total) * 100

}

// Keys returns all cache keys (for debugging).

func (c *Cache) Keys() []string {

	c.mutex.RLock()

	defer c.mutex.RUnlock()

	keys := make([]string, 0, len(c.items))

	for key := range c.items {

		keys = append(keys, key)

	}

	return keys

}

// Internal methods.

func (c *Cache) evictIfNecessary(newItemSize int64) {

	// Check if we need to evict based on count.

	for len(c.items) >= c.maxSize {

		c.evictLeastRecentlyUsed()

	}

	// Could also implement size-based eviction here if needed.

}

func (c *Cache) evictLeastRecentlyUsed() {

	var oldestKey string

	oldestTime := time.Now()

	// Find the least recently used item.

	for key, item := range c.items {

		if item.lastAccess.Before(oldestTime) {

			oldestTime = item.lastAccess

			oldestKey = key

		}

	}

	// Remove the oldest item.

	if oldestKey != "" {

		if item, exists := c.items[oldestKey]; exists {

			delete(c.items, oldestKey)

			c.updateStats(func(s *CacheStats) {

				s.Evictions++

				s.TotalItems--

				s.TotalSize -= item.size

				c.calculateAvgItemSize()

			})

		}

	}

}

func (c *Cache) cleanupExpiredItems() {

	ticker := time.NewTicker(1 * time.Minute)

	defer ticker.Stop()

	for range ticker.C {

		c.mutex.Lock()

		now := time.Now()

		var expiredCount int

		var expiredSize int64

		for key, item := range c.items {

			if now.After(item.expiration) {

				delete(c.items, key)

				expiredCount++

				expiredSize += item.size

			}

		}

		if expiredCount > 0 {

			c.updateStats(func(s *CacheStats) {

				s.ExpiredItems += int64(expiredCount)

				s.TotalItems -= int64(expiredCount)

				s.TotalSize -= expiredSize

				c.calculateAvgItemSize()

			})

		}

		c.mutex.Unlock()

	}

}

func (c *Cache) estimateItemSize(key string, value interface{}) int64 {

	// Rough estimation of item size in bytes.

	size := int64(len(key))

	switch v := value.(type) {

	case string:

		size += int64(len(v))

	case []byte:

		size += int64(len(v))

	case int, int32, int64, float32, float64:

		size += 8

	case bool:

		size += 1

	default:

		// For complex objects, assume 1KB average size.

		size += 1024

	}

	return size

}

func (c *Cache) updateStats(updateFn func(*CacheStats)) {

	c.stats.mutex.Lock()

	defer c.stats.mutex.Unlock()

	updateFn(&c.stats.CacheStats)

}

func (c *Cache) calculateAvgItemSize() {

	if c.stats.CacheStats.TotalItems > 0 {

		c.stats.CacheStats.AvgItemSize = float64(c.stats.CacheStats.TotalSize) / float64(c.stats.CacheStats.TotalItems)

	} else {

		c.stats.CacheStats.AvgItemSize = 0

	}

}

// CacheMiddleware provides HTTP middleware for caching responses.

func (s *NephoranAPIServer) cacheMiddleware(ttl time.Duration) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// Only cache GET requests.

			if r.Method != "GET" {

				next.ServeHTTP(w, r)

				return

			}

			// Skip caching for real-time endpoints.

			if strings.HasPrefix(r.URL.Path, "/api/v1/realtime/") {

				next.ServeHTTP(w, r)

				return

			}

			// Generate cache key.

			cacheKey := s.generateCacheKey(r)

			// Check cache first.

			if s.cache != nil {

				if cached, found := s.cache.Get(cacheKey); found {

					s.metrics.CacheHits.Inc()

					w.Header().Set("X-Cache", "HIT")

					s.writeJSONResponse(w, http.StatusOK, cached)

					return

				}

				s.metrics.CacheMisses.Inc()

			}

			// Wrap response writer to capture response.

			wrapper := &cacheResponseWrapper{

				ResponseWriter: w,

				body: make([]byte, 0),

				statusCode: http.StatusOK,
			}

			next.ServeHTTP(wrapper, r)

			// Cache successful responses.

			if wrapper.statusCode == http.StatusOK && s.cache != nil {

				s.cache.SetWithTTL(cacheKey, wrapper.body, ttl)

				w.Header().Set("X-Cache", "MISS")

			}

		})

	}

}

func (s *NephoranAPIServer) generateCacheKey(r *http.Request) string {

	// Include user ID for user-specific caching.

	userID := auth.GetUserID(r.Context())

	return fmt.Sprintf("%s:%s:%s:%s", r.Method, r.URL.Path, r.URL.RawQuery, userID)

}

// cacheResponseWrapper wraps http.ResponseWriter to capture response body.

type cacheResponseWrapper struct {
	http.ResponseWriter

	body []byte

	statusCode int
}

// WriteHeader performs writeheader operation.

func (w *cacheResponseWrapper) WriteHeader(code int) {

	w.statusCode = code

	w.ResponseWriter.WriteHeader(code)

}

// Write performs write operation.

func (w *cacheResponseWrapper) Write(data []byte) (int, error) {

	w.body = append(w.body, data...)

	return w.ResponseWriter.Write(data)

}
