//go:build !disable_rag && !test

package rag

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
	"unsafe"

	"container/heap"
)

// MemoryCache provides a high-performance in-memory cache with LRU eviction
type MemoryCache struct {
	config    *MemoryCacheConfig
	logger    *slog.Logger
	metrics   *MemoryCacheMetrics
	cache     map[string]*CacheItem
	lruList   *LRUList
	mutex     sync.RWMutex
	stopChan  chan struct{}
	startTime time.Time
}

// MemoryCacheConfig holds configuration for the in-memory cache
type MemoryCacheConfig struct {
	// Size limits
	MaxItems         int   `json:"max_items"`           // Maximum number of items
	MaxSizeBytes     int64 `json:"max_size_bytes"`      // Maximum total size in bytes
	MaxItemSizeBytes int64 `json:"max_item_size_bytes"` // Maximum individual item size

	// TTL settings
	DefaultTTL      time.Duration `json:"default_ttl"`      // Default time-to-live
	MaxTTL          time.Duration `json:"max_ttl"`          // Maximum allowed TTL
	CleanupInterval time.Duration `json:"cleanup_interval"` // How often to clean expired items

	// Performance settings
	EnableMetrics        bool `json:"enable_metrics"`        // Enable detailed metrics collection
	EnableCompression    bool `json:"enable_compression"`    // Enable compression for large items
	CompressionThreshold int  `json:"compression_threshold"` // Size threshold for compression

	// Eviction policy
	EvictionPolicy         string  `json:"eviction_policy"`           // "lru", "lfu", "ttl"
	EvictionRatio          float64 `json:"eviction_ratio"`            // Ratio of items to evict when full
	PreventHotSpotEviction bool    `json:"prevent_hot_spot_eviction"` // Don't evict frequently accessed items

	// Categories with different settings
	CategoryConfigs map[string]*CategoryConfig `json:"category_configs"`
}

// CategoryConfig holds category-specific cache settings
type CategoryConfig struct {
	MaxItems     int           `json:"max_items"`
	MaxSizeBytes int64         `json:"max_size_bytes"`
	DefaultTTL   time.Duration `json:"default_ttl"`
	Priority     int           `json:"priority"` // Higher priority = less likely to be evicted
}

// MemoryCacheMetrics tracks cache performance
type MemoryCacheMetrics struct {
	// Basic metrics
	TotalRequests int64 `json:"total_requests"`
	Hits          int64 `json:"hits"`
	Misses        int64 `json:"misses"`
	Sets          int64 `json:"sets"`
	Deletes       int64 `json:"deletes"`
	Evictions     int64 `json:"evictions"`

	// Performance metrics
	HitRate        float64       `json:"hit_rate"`
	AverageGetTime time.Duration `json:"average_get_time"`
	AverageSetTime time.Duration `json:"average_set_time"`

	// Size metrics
	CurrentItems     int64 `json:"current_items"`
	CurrentSizeBytes int64 `json:"current_size_bytes"`
	MaxItems         int64 `json:"max_items"`
	MaxSizeBytes     int64 `json:"max_size_bytes"`

	// Category metrics
	CategoryStats map[string]*CategoryStats `json:"category_stats"`

	// Eviction metrics
	LRUEvictions  int64 `json:"lru_evictions"`
	TTLEvictions  int64 `json:"ttl_evictions"`
	SizeEvictions int64 `json:"size_evictions"`

	// Timing metrics
	LastCleanup  time.Time `json:"last_cleanup"`
	LastEviction time.Time `json:"last_eviction"`
	LastUpdated  time.Time `json:"last_updated"`

	mutex sync.RWMutex
}

// CategoryStats holds statistics for a specific category
type CategoryStats struct {
	Items     int64   `json:"items"`
	SizeBytes int64   `json:"size_bytes"`
	Hits      int64   `json:"hits"`
	Misses    int64   `json:"misses"`
	Sets      int64   `json:"sets"`
	Evictions int64   `json:"evictions"`
	HitRate   float64 `json:"hit_rate"`
}

// CacheItem represents an item stored in the cache
type CacheItem struct {
	Key          string                 `json:"key"`
	Value        interface{}            `json:"value"`
	Category     string                 `json:"category"`
	SizeBytes    int64                  `json:"size_bytes"`
	CreatedAt    time.Time              `json:"created_at"`
	LastAccessed time.Time              `json:"last_accessed"`
	ExpiresAt    time.Time              `json:"expires_at"`
	AccessCount  int64                  `json:"access_count"`
	Priority     int                    `json:"priority"`
	Metadata     map[string]interface{} `json:"metadata"`

	// LRU list pointers
	prev *CacheItem
	next *CacheItem
}

// LRUList implements a doubly-linked list for LRU tracking
type LRUList struct {
	head *CacheItem
	tail *CacheItem
	size int
}

// CacheEntry represents a cache entry for external APIs
type CacheEntry struct {
	Key      string                 `json:"key"`
	Value    interface{}            `json:"value"`
	Category string                 `json:"category"`
	TTL      time.Duration          `json:"ttl"`
	Metadata map[string]interface{} `json:"metadata"`
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache(config *MemoryCacheConfig) *MemoryCache {
	if config == nil {
		config = getDefaultMemoryCacheConfig()
	}

	mc := &MemoryCache{
		config:    config,
		logger:    slog.Default().With("component", "memory-cache"),
		cache:     make(map[string]*CacheItem),
		lruList:   newLRUList(),
		stopChan:  make(chan struct{}),
		startTime: time.Now(),
		metrics: &MemoryCacheMetrics{
			CategoryStats: make(map[string]*CategoryStats),
			LastUpdated:   time.Now(),
		},
	}

	// Initialize category stats
	for category := range config.CategoryConfigs {
		mc.metrics.CategoryStats[category] = &CategoryStats{}
	}

	// Start background cleanup
	go mc.startCleanupRoutine()

	mc.logger.Info("Memory cache initialized",
		"max_items", config.MaxItems,
		"max_size_bytes", config.MaxSizeBytes,
		"eviction_policy", config.EvictionPolicy,
	)

	return mc
}

// Get retrieves an item from cache
func (mc *MemoryCache) Get(key string) (interface{}, bool) {
	startTime := time.Now()
	defer func() {
		mc.updateMetrics(func(m *MemoryCacheMetrics) {
			m.TotalRequests++
			getTime := time.Since(startTime)
			if m.TotalRequests > 0 {
				m.AverageGetTime = (m.AverageGetTime*time.Duration(m.TotalRequests-1) + getTime) / time.Duration(m.TotalRequests)
			} else {
				m.AverageGetTime = getTime
			}
		})
	}()

	mc.mutex.RLock()
	item, exists := mc.cache[key]
	mc.mutex.RUnlock()

	if !exists {
		mc.updateMetrics(func(m *MemoryCacheMetrics) {
			m.Misses++
			m.HitRate = float64(m.Hits) / float64(m.TotalRequests)
		})
		return nil, false
	}

	// Check if item has expired
	if mc.isExpired(item) {
		mc.mutex.Lock()
		mc.removeItem(key, "ttl")
		mc.mutex.Unlock()

		mc.updateMetrics(func(m *MemoryCacheMetrics) {
			m.Misses++
			m.TTLEvictions++
			m.HitRate = float64(m.Hits) / float64(m.TotalRequests)
		})
		return nil, false
	}

	// Update access information
	mc.mutex.Lock()
	item.LastAccessed = time.Now()
	item.AccessCount++
	mc.lruList.moveToFront(item)
	mc.mutex.Unlock()

	// Update metrics
	mc.updateMetrics(func(m *MemoryCacheMetrics) {
		m.Hits++
		m.HitRate = float64(m.Hits) / float64(m.TotalRequests)
		if stats, exists := m.CategoryStats[item.Category]; exists {
			stats.Hits++
			stats.HitRate = float64(stats.Hits) / float64(stats.Hits+stats.Misses)
		}
	})

	return item.Value, true
}

// GetWithCategory retrieves an item from cache with category information
func (mc *MemoryCache) GetWithCategory(key string) (interface{}, string, bool) {
	value, exists := mc.Get(key)
	if !exists {
		return nil, "", false
	}

	mc.mutex.RLock()
	item := mc.cache[key]
	category := item.Category
	mc.mutex.RUnlock()

	return value, category, true
}

// Set stores an item in cache
func (mc *MemoryCache) Set(key string, value interface{}, ttl time.Duration) error {
	return mc.SetWithCategory(key, value, "default", ttl, nil)
}

// SetWithCategory stores an item in cache with category and metadata
func (mc *MemoryCache) SetWithCategory(key string, value interface{}, category string, ttl time.Duration, metadata map[string]interface{}) error {
	startTime := time.Now()
	defer func() {
		mc.updateMetrics(func(m *MemoryCacheMetrics) {
			setTime := time.Since(startTime)
			if m.Sets > 0 {
				m.AverageSetTime = (m.AverageSetTime*time.Duration(m.Sets) + setTime) / time.Duration(m.Sets+1)
			} else {
				m.AverageSetTime = setTime
			}
			m.Sets++
		})
	}()

	// Calculate item size
	sizeBytes := mc.calculateSize(value)

	// Check item size limits
	if sizeBytes > mc.config.MaxItemSizeBytes {
		return fmt.Errorf("item size %d bytes exceeds maximum %d bytes", sizeBytes, mc.config.MaxItemSizeBytes)
	}

	// Apply category-specific limits
	categoryConfig := mc.getCategoryConfig(category)
	if categoryConfig != nil && sizeBytes > categoryConfig.MaxSizeBytes {
		return fmt.Errorf("item size %d bytes exceeds category limit %d bytes", sizeBytes, categoryConfig.MaxSizeBytes)
	}

	// Set TTL
	if ttl == 0 {
		if categoryConfig != nil && categoryConfig.DefaultTTL > 0 {
			ttl = categoryConfig.DefaultTTL
		} else {
			ttl = mc.config.DefaultTTL
		}
	}
	if ttl > mc.config.MaxTTL {
		ttl = mc.config.MaxTTL
	}

	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	// Check if we need to make space
	if mc.needsEviction(sizeBytes, category) {
		if err := mc.performEviction(sizeBytes, category); err != nil {
			return fmt.Errorf("failed to perform eviction: %w", err)
		}
	}

	// Remove existing item if present
	if existingItem, exists := mc.cache[key]; exists {
		mc.removeItemUnsafe(key, "update")
		sizeBytes -= existingItem.SizeBytes // Adjust for size difference
	}

	// Create new item
	item := &CacheItem{
		Key:          key,
		Value:        value,
		Category:     category,
		SizeBytes:    sizeBytes,
		CreatedAt:    time.Now(),
		LastAccessed: time.Now(),
		ExpiresAt:    time.Now().Add(ttl),
		AccessCount:  1,
		Priority:     mc.getPriority(category),
		Metadata:     metadata,
	}

	// Add to cache and LRU list
	mc.cache[key] = item
	mc.lruList.addToFront(item)

	// Update metrics
	mc.updateMetricsUnsafe(func(m *MemoryCacheMetrics) {
		m.CurrentItems = int64(len(mc.cache))
		m.CurrentSizeBytes += sizeBytes

		if stats, exists := m.CategoryStats[category]; exists {
			stats.Items++
			stats.SizeBytes += sizeBytes
			stats.Sets++
		} else {
			m.CategoryStats[category] = &CategoryStats{
				Items:     1,
				SizeBytes: sizeBytes,
				Sets:      1,
			}
		}
	})

	return nil
}

// Delete removes an item from cache
func (mc *MemoryCache) Delete(key string) bool {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if item, exists := mc.cache[key]; exists {
		mc.removeItemUnsafe(key, "delete")
		mc.updateMetricsUnsafe(func(m *MemoryCacheMetrics) {
			m.Deletes++
			if stats, exists := m.CategoryStats[item.Category]; exists {
				stats.Items--
				stats.SizeBytes -= item.SizeBytes
			}
		})
		return true
	}
	return false
}

// Clear removes all items from cache
func (mc *MemoryCache) Clear() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.cache = make(map[string]*CacheItem)
	mc.lruList = newLRUList()

	mc.updateMetricsUnsafe(func(m *MemoryCacheMetrics) {
		m.CurrentItems = 0
		m.CurrentSizeBytes = 0
		for _, stats := range m.CategoryStats {
			stats.Items = 0
			stats.SizeBytes = 0
		}
	})

	mc.logger.Info("Cache cleared")
}

// ClearCategory removes all items in a specific category
func (mc *MemoryCache) ClearCategory(category string) int {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	keysToDelete := make([]string, 0)
	for key, item := range mc.cache {
		if item.Category == category {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		mc.removeItemUnsafe(key, "category_clear")
	}

	mc.updateMetricsUnsafe(func(m *MemoryCacheMetrics) {
		if stats, exists := m.CategoryStats[category]; exists {
			stats.Items = 0
			stats.SizeBytes = 0
		}
	})

	mc.logger.Info("Category cleared", "category", category, "items_removed", len(keysToDelete))
	return len(keysToDelete)
}

// GetStats returns current cache statistics
func (mc *MemoryCache) GetStats() *MemoryCacheMetrics {
	mc.metrics.mutex.RLock()
	defer mc.metrics.mutex.RUnlock()

	// Create a deep copy
	stats := &MemoryCacheMetrics{
		TotalRequests:    mc.metrics.TotalRequests,
		Hits:             mc.metrics.Hits,
		Misses:           mc.metrics.Misses,
		Sets:             mc.metrics.Sets,
		Deletes:          mc.metrics.Deletes,
		Evictions:        mc.metrics.Evictions,
		HitRate:          mc.metrics.HitRate,
		AverageGetTime:   mc.metrics.AverageGetTime,
		AverageSetTime:   mc.metrics.AverageSetTime,
		CurrentItems:     mc.metrics.CurrentItems,
		CurrentSizeBytes: mc.metrics.CurrentSizeBytes,
		MaxItems:         int64(mc.config.MaxItems),
		MaxSizeBytes:     mc.config.MaxSizeBytes,
		LRUEvictions:     mc.metrics.LRUEvictions,
		TTLEvictions:     mc.metrics.TTLEvictions,
		SizeEvictions:    mc.metrics.SizeEvictions,
		LastCleanup:      mc.metrics.LastCleanup,
		LastEviction:     mc.metrics.LastEviction,
		LastUpdated:      mc.metrics.LastUpdated,
		CategoryStats:    make(map[string]*CategoryStats),
	}

	// Deep copy category stats
	for category, categoryStats := range mc.metrics.CategoryStats {
		stats.CategoryStats[category] = &CategoryStats{
			Items:     categoryStats.Items,
			SizeBytes: categoryStats.SizeBytes,
			Hits:      categoryStats.Hits,
			Misses:    categoryStats.Misses,
			Sets:      categoryStats.Sets,
			Evictions: categoryStats.Evictions,
			HitRate:   categoryStats.HitRate,
		}
	}

	return stats
}

// GetInfo returns detailed cache information
func (mc *MemoryCache) GetInfo() map[string]interface{} {
	stats := mc.GetStats()

	return map[string]interface{}{
		"status":         "healthy",
		"uptime_seconds": int64(time.Since(mc.startTime).Seconds()),
		"config": map[string]interface{}{
			"max_items":       mc.config.MaxItems,
			"max_size_bytes":  mc.config.MaxSizeBytes,
			"eviction_policy": mc.config.EvictionPolicy,
			"default_ttl":     mc.config.DefaultTTL.String(),
		},
		"stats": stats,
		"memory_efficiency": map[string]interface{}{
			"utilization_percent":      float64(stats.CurrentItems) / float64(stats.MaxItems) * 100,
			"size_utilization_percent": float64(stats.CurrentSizeBytes) / float64(stats.MaxSizeBytes) * 100,
			"average_item_size": func() float64 {
				if stats.CurrentItems > 0 {
					return float64(stats.CurrentSizeBytes) / float64(stats.CurrentItems)
				}
				return 0
			}(),
		},
	}
}

// Cleanup removes expired items
func (mc *MemoryCache) Cleanup() int {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	now := time.Now()
	expiredKeys := make([]string, 0)

	for key, item := range mc.cache {
		if now.After(item.ExpiresAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		mc.removeItemUnsafe(key, "cleanup")
	}

	mc.updateMetricsUnsafe(func(m *MemoryCacheMetrics) {
		m.TTLEvictions += int64(len(expiredKeys))
		m.LastCleanup = now
	})

	if len(expiredKeys) > 0 {
		mc.logger.Debug("Cleanup completed", "expired_items", len(expiredKeys))
	}

	return len(expiredKeys)
}

// Close closes the cache and stops background routines
func (mc *MemoryCache) Close() error {
	close(mc.stopChan)
	mc.Clear()
	mc.logger.Info("Memory cache closed")
	return nil
}

// Private methods

func (mc *MemoryCache) startCleanupRoutine() {
	if mc.config.CleanupInterval <= 0 {
		return
	}

	ticker := time.NewTicker(mc.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mc.Cleanup()
		case <-mc.stopChan:
			return
		}
	}
}

func (mc *MemoryCache) isExpired(item *CacheItem) bool {
	return time.Now().After(item.ExpiresAt)
}

func (mc *MemoryCache) needsEviction(newItemSize int64, category string) bool {
	// Check global limits
	if len(mc.cache) >= mc.config.MaxItems {
		return true
	}

	totalSizeAfterAdd := mc.metrics.CurrentSizeBytes + newItemSize
	if totalSizeAfterAdd > mc.config.MaxSizeBytes {
		return true
	}

	// Check category limits
	categoryConfig := mc.getCategoryConfig(category)
	if categoryConfig != nil {
		if stats, exists := mc.metrics.CategoryStats[category]; exists {
			if stats.Items >= int64(categoryConfig.MaxItems) {
				return true
			}
			if stats.SizeBytes+newItemSize > categoryConfig.MaxSizeBytes {
				return true
			}
		}
	}

	return false
}

func (mc *MemoryCache) performEviction(newItemSize int64, category string) error {
	switch mc.config.EvictionPolicy {
	case "lru":
		return mc.evictLRU(newItemSize, category)
	case "lfu":
		return mc.evictLFU(newItemSize, category)
	case "ttl":
		return mc.evictByTTL(newItemSize, category)
	default:
		return mc.evictLRU(newItemSize, category)
	}
}

func (mc *MemoryCache) evictLRU(newItemSize int64, category string) error {
	evictionTarget := int(float64(len(mc.cache)) * mc.config.EvictionRatio)
	if evictionTarget < 1 {
		evictionTarget = 1
	}

	evicted := 0
	current := mc.lruList.tail

	for current != nil && evicted < evictionTarget {
		prev := current.prev

		// Skip high-priority items if configured
		if mc.config.PreventHotSpotEviction && current.AccessCount > 10 {
			current = prev
			continue
		}

		// Skip items from same high-priority category if space allows
		if current.Priority > mc.getPriority(category) && mc.hasSpaceAfterEviction(newItemSize, evicted+1) {
			current = prev
			continue
		}

		key := current.Key
		mc.removeItemUnsafe(key, "lru_eviction")
		evicted++

		current = prev
	}

	mc.updateMetricsUnsafe(func(m *MemoryCacheMetrics) {
		m.LRUEvictions += int64(evicted)
		m.Evictions += int64(evicted)
		m.LastEviction = time.Now()
	})

	if evicted > 0 {
		mc.logger.Debug("LRU eviction completed", "evicted_items", evicted)
	}

	return nil
}

func (mc *MemoryCache) evictLFU(newItemSize int64, category string) error {
	// Create a priority queue of items sorted by access count
	items := make([]*CacheItem, 0, len(mc.cache))
	for _, item := range mc.cache {
		items = append(items, item)
	}

	// Sort by access count and last accessed time
	lfuHeap := &LFUHeap{items: items}
	heap.Init(lfuHeap)

	evictionTarget := int(float64(len(mc.cache)) * mc.config.EvictionRatio)
	if evictionTarget < 1 {
		evictionTarget = 1
	}

	evicted := 0
	for evicted < evictionTarget && lfuHeap.Len() > 0 {
		item := heap.Pop(lfuHeap).(*CacheItem)

		// Skip high-priority items if configured
		if mc.config.PreventHotSpotEviction && item.Priority > mc.getPriority(category) {
			continue
		}

		mc.removeItemUnsafe(item.Key, "lfu_eviction")
		evicted++
	}

	mc.updateMetricsUnsafe(func(m *MemoryCacheMetrics) {
		m.Evictions += int64(evicted)
		m.LastEviction = time.Now()
	})

	mc.logger.Debug("LFU eviction completed", "evicted_items", evicted)
	return nil
}

func (mc *MemoryCache) evictByTTL(newItemSize int64, category string) error {
	// First try to evict expired items
	expiredCount := 0
	now := time.Now()

	expiredKeys := make([]string, 0)
	for key, item := range mc.cache {
		if now.After(item.ExpiresAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		mc.removeItemUnsafe(key, "ttl_eviction")
		expiredCount++
	}

	// If we still need space, evict items closest to expiration
	if mc.needsEviction(newItemSize, category) {
		return mc.evictLRU(newItemSize, category) // Fall back to LRU
	}

	mc.updateMetricsUnsafe(func(m *MemoryCacheMetrics) {
		m.TTLEvictions += int64(expiredCount)
		m.Evictions += int64(expiredCount)
		m.LastEviction = time.Now()
	})

	return nil
}

func (mc *MemoryCache) hasSpaceAfterEviction(newItemSize int64, evictionCount int) bool {
	projectedItems := len(mc.cache) - evictionCount + 1
	projectedSize := mc.metrics.CurrentSizeBytes + newItemSize

	return projectedItems <= mc.config.MaxItems && projectedSize <= mc.config.MaxSizeBytes
}

func (mc *MemoryCache) removeItem(key, reason string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.removeItemUnsafe(key, reason)
}

func (mc *MemoryCache) removeItemUnsafe(key, reason string) {
	if item, exists := mc.cache[key]; exists {
		delete(mc.cache, key)
		mc.lruList.remove(item)

		mc.updateMetricsUnsafe(func(m *MemoryCacheMetrics) {
			m.CurrentItems = int64(len(mc.cache))
			m.CurrentSizeBytes -= item.SizeBytes
		})
	}
}

func (mc *MemoryCache) calculateSize(value interface{}) int64 {
	// This is a simplified size calculation
	// In a production system, you might want a more accurate calculation
	switch v := value.(type) {
	case string:
		return int64(len(v))
	case []byte:
		return int64(len(v))
	case []float32:
		return int64(len(v) * 4) // 4 bytes per float32
	case []float64:
		return int64(len(v) * 8) // 8 bytes per float64
	default:
		// Use unsafe.Sizeof for other types (approximation)
		return int64(unsafe.Sizeof(v))
	}
}

func (mc *MemoryCache) getCategoryConfig(category string) *CategoryConfig {
	if config, exists := mc.config.CategoryConfigs[category]; exists {
		return config
	}
	return nil
}

func (mc *MemoryCache) getPriority(category string) int {
	if config := mc.getCategoryConfig(category); config != nil {
		return config.Priority
	}
	return 1 // Default priority
}

func (mc *MemoryCache) updateMetrics(updater func(*MemoryCacheMetrics)) {
	mc.metrics.mutex.Lock()
	defer mc.metrics.mutex.Unlock()
	updater(mc.metrics)
	mc.metrics.LastUpdated = time.Now()
}

func (mc *MemoryCache) updateMetricsUnsafe(updater func(*MemoryCacheMetrics)) {
	updater(mc.metrics)
	mc.metrics.LastUpdated = time.Now()
}

// LRU List implementation

func newLRUList() *LRUList {
	return &LRUList{}
}

func (l *LRUList) addToFront(item *CacheItem) {
	if l.head == nil {
		l.head = item
		l.tail = item
		item.prev = nil
		item.next = nil
	} else {
		item.next = l.head
		item.prev = nil
		l.head.prev = item
		l.head = item
	}
	l.size++
}

func (l *LRUList) remove(item *CacheItem) {
	if item.prev != nil {
		item.prev.next = item.next
	} else {
		l.head = item.next
	}

	if item.next != nil {
		item.next.prev = item.prev
	} else {
		l.tail = item.prev
	}

	item.prev = nil
	item.next = nil
	l.size--
}

func (l *LRUList) moveToFront(item *CacheItem) {
	l.remove(item)
	l.addToFront(item)
}

// LFU Heap for eviction

type LFUHeap struct {
	items []*CacheItem
}

func (h LFUHeap) Len() int { return len(h.items) }

func (h LFUHeap) Less(i, j int) bool {
	// Less frequently used items first
	if h.items[i].AccessCount != h.items[j].AccessCount {
		return h.items[i].AccessCount < h.items[j].AccessCount
	}
	// If same access count, older items first
	return h.items[i].LastAccessed.Before(h.items[j].LastAccessed)
}

func (h LFUHeap) Swap(i, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
}

func (h *LFUHeap) Push(x interface{}) {
	h.items = append(h.items, x.(*CacheItem))
}

func (h *LFUHeap) Pop() interface{} {
	old := h.items
	n := len(old)
	item := old[n-1]
	h.items = old[0 : n-1]
	return item
}

// Configuration defaults

func getDefaultMemoryCacheConfig() *MemoryCacheConfig {
	return &MemoryCacheConfig{
		MaxItems:               10000,
		MaxSizeBytes:           100 * 1024 * 1024, // 100MB
		MaxItemSizeBytes:       10 * 1024 * 1024,  // 10MB
		DefaultTTL:             1 * time.Hour,
		MaxTTL:                 24 * time.Hour,
		CleanupInterval:        5 * time.Minute,
		EnableMetrics:          true,
		EnableCompression:      false,
		CompressionThreshold:   1024, // 1KB
		EvictionPolicy:         "lru",
		EvictionRatio:          0.1, // Evict 10% when full
		PreventHotSpotEviction: true,
		CategoryConfigs: map[string]*CategoryConfig{
			"embedding": {
				MaxItems:     5000,
				MaxSizeBytes: 50 * 1024 * 1024, // 50MB
				DefaultTTL:   2 * time.Hour,
				Priority:     2,
			},
			"document": {
				MaxItems:     1000,
				MaxSizeBytes: 30 * 1024 * 1024, // 30MB
				DefaultTTL:   4 * time.Hour,
				Priority:     1,
			},
			"query_result": {
				MaxItems:     2000,
				MaxSizeBytes: 15 * 1024 * 1024, // 15MB
				DefaultTTL:   30 * time.Minute,
				Priority:     3,
			},
			"context": {
				MaxItems:     1000,
				MaxSizeBytes: 5 * 1024 * 1024, // 5MB
				DefaultTTL:   15 * time.Minute,
				Priority:     2,
			},
		},
	}
}

// Multi-level cache interface compatibility

// MemoryCacheAdapter adapts MemoryCache to work with the multi-level cache system
type MemoryCacheAdapter struct {
	cache *MemoryCache
}

// NewMemoryCacheAdapter creates a new adapter
func NewMemoryCacheAdapter(config *MemoryCacheConfig) *MemoryCacheAdapter {
	return &MemoryCacheAdapter{
		cache: NewMemoryCache(config),
	}
}

// Get retrieves an embedding from memory cache
func (a *MemoryCacheAdapter) Get(key string) ([]float32, bool, error) {
	value, exists := a.cache.Get(key)
	if !exists {
		return nil, false, nil
	}

	if embedding, ok := value.([]float32); ok {
		return embedding, true, nil
	}

	return nil, false, fmt.Errorf("cached value is not []float32")
}

// Set stores an embedding in memory cache
func (a *MemoryCacheAdapter) Set(key string, embedding []float32, ttl time.Duration) error {
	return a.cache.SetWithCategory(key, embedding, "embedding", ttl, nil)
}

// Delete removes an embedding from memory cache
func (a *MemoryCacheAdapter) Delete(key string) error {
	a.cache.Delete(key)
	return nil
}

// Clear clears the embedding category
func (a *MemoryCacheAdapter) Clear() error {
	a.cache.ClearCategory("embedding")
	return nil
}

// Stats returns cache statistics
func (a *MemoryCacheAdapter) Stats() CacheStats {
	stats := a.cache.GetStats()
	embeddingStats := stats.CategoryStats["embedding"]

	if embeddingStats == nil {
		return CacheStats{}
	}

	return CacheStats{
		Size:    embeddingStats.Items,
		Hits:    embeddingStats.Hits,
		Misses:  embeddingStats.Misses,
		HitRate: embeddingStats.HitRate,
	}
}

// Close closes the cache
func (a *MemoryCacheAdapter) Close() error {
	return a.cache.Close()
}
