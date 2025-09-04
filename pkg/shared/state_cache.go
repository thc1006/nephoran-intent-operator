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

package shared

import (
	"container/list"
	"sync"
	"time"
)

// CacheEntry represents an entry in the state cache.

type CacheEntry struct {
	Key string

	Value interface{}

	ExpiresAt time.Time

	AccessCount int64

	LastAccess time.Time

	CreatedAt time.Time

	Size int64

	element *list.Element
}

// StateCache provides thread-safe caching for intent states.

type StateCache struct {
	mutex sync.RWMutex

	entries map[string]*CacheEntry

	evictionList *list.List

	// Configuration.

	maxSize int

	defaultTTL time.Duration

	// Statistics.

	hits int64

	misses int64

	evictions int64

	currentSize int64

	// Cleanup.

	cleanupInterval time.Duration

	stopCleanup chan bool

	cleanupRunning bool
}

// NewStateCache creates a new state cache.

func NewStateCache(maxSize int, defaultTTL time.Duration) *StateCache {
	cache := &StateCache{
		entries: make(map[string]*CacheEntry),

		evictionList: list.New(),

		maxSize: maxSize,

		defaultTTL: defaultTTL,

		cleanupInterval: 5 * time.Minute,

		stopCleanup: make(chan bool),
	}

	// Start background cleanup.

	go cache.runCleanup()

	return cache
}

// Get retrieves a value from the cache.

func (c *StateCache) Get(key string) interface{} {
	c.mutex.RLock()

	entry, exists := c.entries[key]

	c.mutex.RUnlock()

	if !exists {

		c.mutex.Lock()

		c.misses++

		c.mutex.Unlock()

		return nil

	}

	// Check if expired.

	if time.Now().After(entry.ExpiresAt) {

		c.mutex.Lock()

		c.removeEntry(key)

		c.misses++

		c.mutex.Unlock()

		return nil

	}

	// Update access information.

	c.mutex.Lock()

	entry.AccessCount++

	entry.LastAccess = time.Now()

	c.moveToFront(entry)

	c.hits++

	c.mutex.Unlock()

	return entry.Value
}

// Set stores a value in the cache.

func (c *StateCache) Set(key string, value interface{}, ttl time.Duration) {
	if ttl == 0 {
		ttl = c.defaultTTL
	}

	c.mutex.Lock()

	defer c.mutex.Unlock()

	now := time.Now()

	expiresAt := now.Add(ttl)

	// Check if key already exists.

	if entry, exists := c.entries[key]; exists {

		// Update existing entry.

		entry.Value = value

		entry.ExpiresAt = expiresAt

		entry.LastAccess = now

		entry.AccessCount++

		c.moveToFront(entry)

		return

	}

	// Create new entry.

	entry := &CacheEntry{
		Key: key,

		Value: value,

		ExpiresAt: expiresAt,

		AccessCount: 1,

		LastAccess: now,

		CreatedAt: now,

		Size: c.calculateSize(value),
	}

	// Add to front of eviction list.

	entry.element = c.evictionList.PushFront(entry)

	c.entries[key] = entry

	c.currentSize++

	// Check if we need to evict entries.

	c.evictIfNecessary()
}

// Delete removes a key from the cache.

func (c *StateCache) Delete(key string) bool {
	c.mutex.Lock()

	defer c.mutex.Unlock()

	if entry, exists := c.entries[key]; exists {

		c.removeEntry(key)

		c.evictionList.Remove(entry.element)

		return true

	}

	return false
}

// Clear removes all entries from the cache.

func (c *StateCache) Clear() {
	c.mutex.Lock()

	defer c.mutex.Unlock()

	c.entries = make(map[string]*CacheEntry)

	c.evictionList = list.New()

	c.currentSize = 0
}

// Size returns the current number of entries in the cache.

func (c *StateCache) Size() int {
	c.mutex.RLock()

	defer c.mutex.RUnlock()

	return len(c.entries)
}

// Stats returns cache statistics.

func (c *StateCache) Stats() *CacheStats {
	c.mutex.RLock()

	defer c.mutex.RUnlock()

	total := c.hits + c.misses

	hitRate := 0.0

	if total > 0 {
		hitRate = float64(c.hits) / float64(total)
	}

	return &CacheStats{
		Hits: c.hits,

		Misses: c.misses,

		HitRate: hitRate,

		Evictions: c.evictions,

		CurrentSize: c.currentSize,

		MaxSize: int64(c.maxSize),

		EntryCount: int64(len(c.entries)),
	}
}

// Cleanup removes expired entries.

func (c *StateCache) Cleanup() int {
	c.mutex.Lock()

	defer c.mutex.Unlock()

	now := time.Now()

	expired := make([]string, 0)

	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			expired = append(expired, key)
		}
	}

	for _, key := range expired {
		c.removeEntry(key)
	}

	return len(expired)
}

// Keys returns all keys in the cache.

func (c *StateCache) Keys() []string {
	c.mutex.RLock()

	defer c.mutex.RUnlock()

	keys := make([]string, 0, len(c.entries))

	for key := range c.entries {
		keys = append(keys, key)
	}

	return keys
}

// Contains checks if a key exists in the cache (without updating access time).

func (c *StateCache) Contains(key string) bool {
	c.mutex.RLock()

	defer c.mutex.RUnlock()

	entry, exists := c.entries[key]

	if !exists {
		return false
	}

	// Check if expired.

	return time.Now().Before(entry.ExpiresAt)
}

// GetMultiple retrieves multiple values at once.

func (c *StateCache) GetMultiple(keys []string) map[string]interface{} {
	result := make(map[string]interface{})

	for _, key := range keys {
		if value := c.Get(key); value != nil {
			result[key] = value
		}
	}

	return result
}

// SetMultiple stores multiple values at once.

func (c *StateCache) SetMultiple(entries map[string]interface{}, ttl time.Duration) {
	for key, value := range entries {
		c.Set(key, value, ttl)
	}
}

// GetOldest returns the oldest entry in the cache.

func (c *StateCache) GetOldest() (string, interface{}, bool) {
	c.mutex.RLock()

	defer c.mutex.RUnlock()

	if c.evictionList.Len() == 0 {
		return "", nil, false
	}

	back := c.evictionList.Back()

	if back == nil {
		return "", nil, false
	}

	entry := back.Value.(*CacheEntry)

	return entry.Key, entry.Value, true
}

// GetMostRecent returns the most recently accessed entry.

func (c *StateCache) GetMostRecent() (string, interface{}, bool) {
	c.mutex.RLock()

	defer c.mutex.RUnlock()

	if c.evictionList.Len() == 0 {
		return "", nil, false
	}

	front := c.evictionList.Front()

	if front == nil {
		return "", nil, false
	}

	entry := front.Value.(*CacheEntry)

	return entry.Key, entry.Value, true
}

// Stop stops the cache cleanup process.

func (c *StateCache) Stop() {
	if c.cleanupRunning {

		c.stopCleanup <- true

		c.cleanupRunning = false

	}
}

// Internal methods.

func (c *StateCache) removeEntry(key string) {
	if entry, exists := c.entries[key]; exists {

		delete(c.entries, key)

		if entry.element != nil {
			c.evictionList.Remove(entry.element)
		}

		c.currentSize--

	}
}

func (c *StateCache) moveToFront(entry *CacheEntry) {
	if entry.element != nil {
		c.evictionList.MoveToFront(entry.element)
	}
}

func (c *StateCache) evictIfNecessary() {
	for len(c.entries) > c.maxSize {
		c.evictOldest()
	}
}

func (c *StateCache) evictOldest() {
	back := c.evictionList.Back()

	if back == nil {
		return
	}

	entry := back.Value.(*CacheEntry)

	c.removeEntry(entry.Key)

	c.evictions++
}

func (c *StateCache) calculateSize(value interface{}) int64 {
	// Simplified size calculation.

	// In a real implementation, you might want more accurate size calculation.

	return 1
}

func (c *StateCache) runCleanup() {
	c.cleanupRunning = true

	ticker := time.NewTicker(c.cleanupInterval)

	defer ticker.Stop()

	for {
		select {

		case <-ticker.C:

			c.Cleanup()

		case <-c.stopCleanup:

			return

		}
	}
}

// CacheStats represents cache statistics.

type CacheStats struct {
	Hits int64 `json:"hits"`

	Misses int64 `json:"misses"`

	HitRate float64 `json:"hitRate"`

	Evictions int64 `json:"evictions"`

	CurrentSize int64 `json:"currentSize"`

	MaxSize int64 `json:"maxSize"`

	EntryCount int64 `json:"entryCount"`
}

// CacheConfig provides configuration for the cache.

type CacheConfig struct {
	MaxSize int `json:"maxSize"`

	DefaultTTL time.Duration `json:"defaultTTL"`

	CleanupInterval time.Duration `json:"cleanupInterval"`

	EvictionPolicy string `json:"evictionPolicy"` // "lru", "lfu", "ttl"

	MaxMemoryMB int64 `json:"maxMemoryMB"`

	EnableStatistics bool `json:"enableStatistics"`

	EnableMetrics bool `json:"enableMetrics"`
}

// DefaultCacheConfig returns default cache configuration.

func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		MaxSize: 10000,

		DefaultTTL: 30 * time.Minute,

		CleanupInterval: 5 * time.Minute,

		EvictionPolicy: "lru",

		MaxMemoryMB: 100,

		EnableStatistics: true,

		EnableMetrics: true,
	}
}

// Advanced cache operations.

// GetWithStats retrieves a value and returns access statistics.

func (c *StateCache) GetWithStats(key string) (interface{}, *CacheEntryStats) {
	c.mutex.RLock()

	entry, exists := c.entries[key]

	c.mutex.RUnlock()

	if !exists || time.Now().After(entry.ExpiresAt) {

		c.mutex.Lock()

		if exists {
			c.removeEntry(key)
		}

		c.misses++

		c.mutex.Unlock()

		return nil, nil

	}

	// Update access information.

	c.mutex.Lock()

	entry.AccessCount++

	entry.LastAccess = time.Now()

	c.moveToFront(entry)

	c.hits++

	stats := &CacheEntryStats{
		AccessCount: entry.AccessCount,

		LastAccess: entry.LastAccess,

		CreatedAt: entry.CreatedAt,

		Size: entry.Size,

		TimeToExpiry: time.Until(entry.ExpiresAt),
	}

	c.mutex.Unlock()

	return entry.Value, stats
}

// SetWithCallback stores a value with an expiration callback.

func (c *StateCache) SetWithCallback(key string, value interface{}, ttl time.Duration, callback func(string, interface{})) {
	c.Set(key, value, ttl)

	// Schedule callback for expiration.

	go func() {
		time.Sleep(ttl)

		if !c.Contains(key) && callback != nil {
			callback(key, value)
		}
	}()
}

// CacheEntryStats provides statistics for a cache entry.

type CacheEntryStats struct {
	AccessCount int64 `json:"accessCount"`

	LastAccess time.Time `json:"lastAccess"`

	CreatedAt time.Time `json:"createdAt"`

	Size int64 `json:"size"`

	TimeToExpiry time.Duration `json:"timeToExpiry"`
}
