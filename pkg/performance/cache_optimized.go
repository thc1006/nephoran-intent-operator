//go:build go1.24

package performance

import (
	"context"
	"fmt"
	"hash/fnv"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"k8s.io/klog/v2"
)

// OptimizedCache provides high-performance caching with Go 1.24+ optimizations.
type OptimizedCache[K comparable, V any] struct {
	shards         []*CacheShard[K, V]
	shardCount     int
	config         *CacheConfig
	metrics        *CacheMetrics
	evictionPolicy EvictionPolicy
	hasher         Hasher[K]
	shutdown       chan struct{}
	wg             sync.WaitGroup
}

// CacheConfig contains cache optimization configuration.
type CacheConfig struct {
	ShardCount       int
	MaxSize          int64
	TTL              time.Duration
	CleanupInterval  time.Duration
	EvictionPolicy   EvictionPolicy
	EnableMetrics    bool
	EnableWarmup     bool
	CompressionLevel int
	SerializerType   SerializerType
	MemoryLimit      int64
	Preallocation    bool
}

// CacheShard represents a cache shard with improved type safety.
type CacheShard[K comparable, V any] struct {
	entries        map[K]*CacheEntry[V]
	lruHead        *CacheEntry[V]
	lruTail        *CacheEntry[V]
	lfuHeap        *LFUHeap[K, V]
	size           int64
	maxSize        int64
	hitCount       int64
	missCount      int64
	evictions      int64
	mu             sync.RWMutex
	lastCleanup    time.Time
	ttl            time.Duration
	evictionPolicy EvictionPolicy
}

// CacheEntry represents a cache entry with metadata.
type CacheEntry[V any] struct {
	key         interface{}
	value       V
	size        int64
	expiration  time.Time
	accessCount int64
	lastAccess  time.Time
	createdAt   time.Time
	prev        *CacheEntry[V]
	next        *CacheEntry[V]
	frequency   int64
	compressed  bool
	version     uint64
}

// EvictionPolicy defines cache eviction strategies.
type EvictionPolicy int

const (
	// EvictionLRU holds evictionlru value.
	EvictionLRU EvictionPolicy = iota
	// EvictionLFU holds evictionlfu value.
	EvictionLFU
	// EvictionTTL holds evictionttl value.
	EvictionTTL
	// EvictionRandom holds evictionrandom value.
	EvictionRandom
	// EvictionAdaptive holds evictionadaptive value.
	EvictionAdaptive
)

// SerializerType defines serialization methods.
type SerializerType int

const (
	// SerializerNone holds serializernone value.
	SerializerNone SerializerType = iota
	// SerializerJSON holds serializerjson value.
	SerializerJSON
	// SerializerGob holds serializergob value.
	SerializerGob
	// SerializerProtobuf holds serializerprotobuf value.
	SerializerProtobuf
)

// Hasher provides generic hash function interface.
type Hasher[K comparable] interface {
	Hash(key K) uint64
}

// DefaultHasher provides default hashing implementation.
type DefaultHasher[K comparable] struct{}

// Hash implements the Hasher interface.
func (h DefaultHasher[K]) Hash(key K) uint64 {
	hasher := fnv.New64a()
	// Convert key to bytes - simplified for demonstration.
	// In real implementation, use proper type-specific hashing.
	hasher.Write([]byte(fmt.Sprintf("%v", key)))
	return hasher.Sum64()
}

// LFUHeap implements a min-heap for LFU eviction.
type LFUHeap[K comparable, V any] struct {
	entries  []*CacheEntry[V]
	indexMap map[K]int
	mu       sync.RWMutex
}

// CacheMetrics tracks cache performance metrics.
type CacheMetrics struct {
	Hits              int64
	Misses            int64
	Evictions         int64
	Size              int64
	MemoryUsage       int64
	CompressionRatio  float64
	AverageAccessTime int64 // nanoseconds
	HitRatio          float64
	Throughput        float64 // operations per second
	ShardDistribution []int64
	Collisions        int64
	ExpiredEntries    int64
}

// CacheStats provides detailed cache statistics.
type CacheStats struct {
	TotalEntries     int64
	TotalHits        int64
	TotalMisses      int64
	TotalEvictions   int64
	HitRatio         float64
	AverageEntrySize int64
	MemoryEfficiency float64
	ShardStats       []ShardStats
}

// ShardStats provides per-shard statistics.
type ShardStats struct {
	ShardID   int
	Entries   int64
	Hits      int64
	Misses    int64
	Evictions int64
	Size      int64
}

// NewOptimizedCache creates a new optimized cache with Go 1.24+ generics.
func NewOptimizedCache[K comparable, V any](config *CacheConfig) *OptimizedCache[K, V] {
	if config == nil {
		config = DefaultCacheConfig()
	}

	// Ensure shard count is power of 2 for efficient modulo.
	shardCount := config.ShardCount
	if shardCount <= 0 {
		shardCount = runtime.NumCPU()
	}

	// Round up to next power of 2.
	powerOf2 := 1
	for powerOf2 < shardCount {
		powerOf2 <<= 1
	}
	shardCount = powerOf2

	cache := &OptimizedCache[K, V]{
		shards:         make([]*CacheShard[K, V], shardCount),
		shardCount:     shardCount,
		config:         config,
		metrics:        &CacheMetrics{ShardDistribution: make([]int64, shardCount)},
		evictionPolicy: config.EvictionPolicy,
		hasher:         DefaultHasher[K]{},
		shutdown:       make(chan struct{}),
	}

	// Initialize shards.
	maxSizePerShard := config.MaxSize / int64(shardCount)
	for i := 0; i < shardCount; i++ {
		cache.shards[i] = NewCacheShard[K, V](maxSizePerShard, config)
	}

	// Start background tasks.
	cache.startBackgroundTasks()

	return cache
}

// DefaultCacheConfig returns default cache configuration.
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		ShardCount:       runtime.NumCPU(),
		MaxSize:          10000,
		TTL:              5 * time.Minute,
		CleanupInterval:  1 * time.Minute,
		EvictionPolicy:   EvictionLRU,
		EnableMetrics:    true,
		EnableWarmup:     false,
		CompressionLevel: 0,
		SerializerType:   SerializerNone,
		MemoryLimit:      100 * 1024 * 1024, // 100MB
		Preallocation:    true,
	}
}

// NewCacheShard creates a new cache shard.
func NewCacheShard[K comparable, V any](maxSize int64, config *CacheConfig) *CacheShard[K, V] {
	shard := &CacheShard[K, V]{
		maxSize:        maxSize,
		ttl:            config.TTL,
		evictionPolicy: config.EvictionPolicy,
		lastCleanup:    time.Now(),
	}

	if config.Preallocation {
		// Pre-allocate map with estimated size.
		shard.entries = make(map[K]*CacheEntry[V], maxSize/10)
	} else {
		shard.entries = make(map[K]*CacheEntry[V])
	}

	if config.EvictionPolicy == EvictionLFU {
		shard.lfuHeap = &LFUHeap[K, V]{
			entries:  make([]*CacheEntry[V], 0, maxSize/10),
			indexMap: make(map[K]int),
		}
	}

	return shard
}

// getShard returns the shard for a given key.
func (c *OptimizedCache[K, V]) getShard(key K) *CacheShard[K, V] {
	hash := c.hasher.Hash(key)
	shardIndex := hash & uint64(c.shardCount-1) // Fast modulo for power of 2
	return c.shards[shardIndex]
}

// Set stores a value in the cache.
func (c *OptimizedCache[K, V]) Set(key K, value V) error {
	return c.SetWithTTL(key, value, c.config.TTL)
}

// SetWithTTL stores a value in the cache with specific TTL.
func (c *OptimizedCache[K, V]) SetWithTTL(key K, value V, ttl time.Duration) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		c.updateAccessTime(duration)
	}()

	shard := c.getShard(key)
	return shard.Set(key, value, ttl)
}

// Get retrieves a value from the cache.
func (c *OptimizedCache[K, V]) Get(key K) (V, bool) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		c.updateAccessTime(duration)
	}()

	shard := c.getShard(key)
	return shard.Get(key)
}

// GetOrSet gets a value or sets it if not present (atomic operation).
func (c *OptimizedCache[K, V]) GetOrSet(key K, valueFunc func() V) (V, bool) {
	shard := c.getShard(key)
	return shard.GetOrSet(key, valueFunc, c.config.TTL)
}

// Delete removes a value from the cache.
func (c *OptimizedCache[K, V]) Delete(key K) bool {
	shard := c.getShard(key)
	return shard.Delete(key)
}

// Clear removes all entries from the cache.
func (c *OptimizedCache[K, V]) Clear() {
	for _, shard := range c.shards {
		shard.Clear()
	}
	c.resetMetrics()
}

// Size returns the current cache size.
func (c *OptimizedCache[K, V]) Size() int64 {
	var total int64
	for _, shard := range c.shards {
		total += shard.Size()
	}
	return total
}

// Set stores a value in the shard.
func (s *CacheShard[K, V]) Set(key K, value V, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	expiration := now.Add(ttl)

	// Check if entry already exists.
	if existing, exists := s.entries[key]; exists {
		// Update existing entry.
		existing.value = value
		existing.expiration = expiration
		existing.lastAccess = now
		existing.version++
		s.moveToFront(existing)
		return nil
	}

	// Create new entry.
	entry := &CacheEntry[V]{
		key:        key,
		value:      value,
		size:       s.calculateSize(value),
		expiration: expiration,
		lastAccess: now,
		createdAt:  now,
		version:    1,
	}

	// Check if we need to evict.
	for s.size+entry.size > s.maxSize && len(s.entries) > 0 {
		s.evictOne()
	}

	// Add new entry.
	s.entries[key] = entry
	s.size += entry.size
	s.addToFront(entry)

	if s.evictionPolicy == EvictionLFU {
		s.lfuHeap.Add(key, entry)
	}

	return nil
}

// Get retrieves a value from the shard.
func (s *CacheShard[K, V]) Get(key K) (V, bool) {
	s.mu.RLock()
	entry, exists := s.entries[key]
	s.mu.RUnlock()

	if !exists {
		atomic.AddInt64(&s.missCount, 1)
		var zero V
		return zero, false
	}

	// Check expiration.
	if time.Now().After(entry.expiration) {
		// Entry expired, remove it.
		s.mu.Lock()
		delete(s.entries, key)
		s.size -= entry.size
		s.removeFromList(entry)
		s.mu.Unlock()

		atomic.AddInt64(&s.missCount, 1)
		var zero V
		return zero, false
	}

	// Update access information.
	s.mu.Lock()
	entry.lastAccess = time.Now()
	atomic.AddInt64(&entry.accessCount, 1)
	atomic.AddInt64(&entry.frequency, 1)
	s.moveToFront(entry)

	if s.evictionPolicy == EvictionLFU {
		s.lfuHeap.Update(key, entry)
	}
	s.mu.Unlock()

	atomic.AddInt64(&s.hitCount, 1)
	return entry.value, true
}

// GetOrSet gets a value or sets it if not present.
func (s *CacheShard[K, V]) GetOrSet(key K, valueFunc func() V, ttl time.Duration) (V, bool) {
	// Try to get first (fast path).
	if value, exists := s.Get(key); exists {
		return value, true
	}

	// Doesn't exist, need to set (slow path).
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check after acquiring lock.
	if entry, exists := s.entries[key]; exists && time.Now().Before(entry.expiration) {
		return entry.value, true
	}

	// Generate value and set it.
	value := valueFunc()
	s.setUnsafe(key, value, ttl)
	return value, false
}

// setUnsafe sets a value without locking (caller must hold lock).
func (s *CacheShard[K, V]) setUnsafe(key K, value V, ttl time.Duration) {
	now := time.Now()
	expiration := now.Add(ttl)

	entry := &CacheEntry[V]{
		key:        key,
		value:      value,
		size:       s.calculateSize(value),
		expiration: expiration,
		lastAccess: now,
		createdAt:  now,
		version:    1,
	}

	// Evict if necessary.
	for s.size+entry.size > s.maxSize && len(s.entries) > 0 {
		s.evictOne()
	}

	s.entries[key] = entry
	s.size += entry.size
	s.addToFront(entry)
}

// Delete removes a value from the shard.
func (s *CacheShard[K, V]) Delete(key K) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.entries[key]
	if !exists {
		return false
	}

	delete(s.entries, key)
	s.size -= entry.size
	s.removeFromList(entry)

	if s.evictionPolicy == EvictionLFU {
		s.lfuHeap.Remove(key)
	}

	return true
}

// Clear removes all entries from the shard.
func (s *CacheShard[K, V]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = make(map[K]*CacheEntry[V])
	s.size = 0
	s.lruHead = nil
	s.lruTail = nil
	atomic.StoreInt64(&s.hitCount, 0)
	atomic.StoreInt64(&s.missCount, 0)
	atomic.StoreInt64(&s.evictions, 0)

	if s.lfuHeap != nil {
		s.lfuHeap.Clear()
	}
}

// Size returns the shard size.
func (s *CacheShard[K, V]) Size() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return int64(len(s.entries))
}

// evictOne evicts one entry based on the eviction policy.
func (s *CacheShard[K, V]) evictOne() {
	switch s.evictionPolicy {
	case EvictionLRU:
		s.evictLRU()
	case EvictionLFU:
		s.evictLFU()
	case EvictionTTL:
		s.evictExpired()
	case EvictionRandom:
		s.evictRandom()
	case EvictionAdaptive:
		s.evictAdaptive()
	default:
		s.evictLRU()
	}
}

// evictLRU evicts the least recently used entry.
func (s *CacheShard[K, V]) evictLRU() {
	if s.lruTail == nil {
		return
	}

	entry := s.lruTail
	delete(s.entries, entry.key.(K))
	s.size -= entry.size
	s.removeFromList(entry)
	atomic.AddInt64(&s.evictions, 1)
}

// evictLFU evicts the least frequently used entry.
func (s *CacheShard[K, V]) evictLFU() {
	if s.lfuHeap == nil || len(s.lfuHeap.entries) == 0 {
		s.evictLRU() // Fallback to LRU
		return
	}

	entry := s.lfuHeap.ExtractMin()
	if entry != nil {
		delete(s.entries, entry.key.(K))
		s.size -= entry.size
		s.removeFromList(entry)
		atomic.AddInt64(&s.evictions, 1)
	}
}

// evictExpired evicts expired entries.
func (s *CacheShard[K, V]) evictExpired() {
	now := time.Now()
	for key, entry := range s.entries {
		if now.After(entry.expiration) {
			delete(s.entries, key)
			s.size -= entry.size
			s.removeFromList(entry)
			atomic.AddInt64(&s.evictions, 1)
			return // Evict one at a time
		}
	}

	// No expired entries, fallback to LRU.
	s.evictLRU()
}

// evictRandom evicts a random entry.
func (s *CacheShard[K, V]) evictRandom() {
	if len(s.entries) == 0 {
		return
	}

	// Simple random eviction (not cryptographically secure).
	for key, entry := range s.entries {
		delete(s.entries, key)
		s.size -= entry.size
		s.removeFromList(entry)
		atomic.AddInt64(&s.evictions, 1)
		return // Evict first entry (pseudo-random due to map iteration)
	}
}

// evictAdaptive uses adaptive eviction based on access patterns.
func (s *CacheShard[K, V]) evictAdaptive() {
	// Simple adaptive strategy: use LFU if we have frequency data, otherwise LRU.
	if s.lfuHeap != nil && len(s.lfuHeap.entries) > 0 {
		s.evictLFU()
	} else {
		s.evictLRU()
	}
}

// LRU list management.
func (s *CacheShard[K, V]) addToFront(entry *CacheEntry[V]) {
	if s.lruHead == nil {
		s.lruHead = entry
		s.lruTail = entry
	} else {
		entry.next = s.lruHead
		s.lruHead.prev = entry
		s.lruHead = entry
	}
}

// moveToFront moves an entry to the front of the LRU list.
func (s *CacheShard[K, V]) moveToFront(entry *CacheEntry[V]) {
	if entry == s.lruHead {
		return
	}

	s.removeFromList(entry)
	s.addToFront(entry)
}

// removeFromList removes an entry from the LRU list.
func (s *CacheShard[K, V]) removeFromList(entry *CacheEntry[V]) {
	if entry.prev != nil {
		entry.prev.next = entry.next
	} else {
		s.lruHead = entry.next
	}

	if entry.next != nil {
		entry.next.prev = entry.prev
	} else {
		s.lruTail = entry.prev
	}

	entry.prev = nil
	entry.next = nil
}

// calculateSize estimates the size of a value (simplified).
func (s *CacheShard[K, V]) calculateSize(value V) int64 {
	// Simple size estimation - in real implementation, use proper size calculation.
	return int64(unsafe.Sizeof(value))
}

// LFU Heap implementation.
func (h *LFUHeap[K, V]) Add(key K, entry *CacheEntry[V]) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.indexMap[key]; exists {
		return // Already exists
	}

	h.entries = append(h.entries, entry)
	idx := len(h.entries) - 1
	h.indexMap[key] = idx
	h.heapifyUp(idx)
}

// Update updates an entry in the LFU heap.
func (h *LFUHeap[K, V]) Update(key K, entry *CacheEntry[V]) {
	h.mu.Lock()
	defer h.mu.Unlock()

	idx, exists := h.indexMap[key]
	if !exists {
		return
	}

	h.entries[idx] = entry
	h.heapifyUp(idx)
	h.heapifyDown(idx)
}

// Remove removes an entry from the LFU heap.
func (h *LFUHeap[K, V]) Remove(key K) {
	h.mu.Lock()
	defer h.mu.Unlock()

	idx, exists := h.indexMap[key]
	if !exists {
		return
	}

	lastIdx := len(h.entries) - 1
	h.entries[idx] = h.entries[lastIdx]
	h.entries = h.entries[:lastIdx]
	delete(h.indexMap, key)

	if idx < len(h.entries) {
		h.indexMap[h.entries[idx].key.(K)] = idx
		h.heapifyDown(idx)
	}
}

// ExtractMin extracts the minimum (least frequently used) entry.
func (h *LFUHeap[K, V]) ExtractMin() *CacheEntry[V] {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.entries) == 0 {
		return nil
	}

	min := h.entries[0]
	lastIdx := len(h.entries) - 1
	h.entries[0] = h.entries[lastIdx]
	h.entries = h.entries[:lastIdx]
	delete(h.indexMap, min.key.(K))

	if len(h.entries) > 0 {
		h.indexMap[h.entries[0].key.(K)] = 0
		h.heapifyDown(0)
	}

	return min
}

// Clear clears the LFU heap.
func (h *LFUHeap[K, V]) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.entries = h.entries[:0]
	h.indexMap = make(map[K]int)
}

// heapifyUp maintains heap property upward.
func (h *LFUHeap[K, V]) heapifyUp(idx int) {
	for idx > 0 {
		parent := (idx - 1) / 2
		if atomic.LoadInt64(&h.entries[idx].frequency) >= atomic.LoadInt64(&h.entries[parent].frequency) {
			break
		}
		h.swap(idx, parent)
		idx = parent
	}
}

// heapifyDown maintains heap property downward.
func (h *LFUHeap[K, V]) heapifyDown(idx int) {
	for {
		smallest := idx
		left := 2*idx + 1
		right := 2*idx + 2

		if left < len(h.entries) && atomic.LoadInt64(&h.entries[left].frequency) < atomic.LoadInt64(&h.entries[smallest].frequency) {
			smallest = left
		}
		if right < len(h.entries) && atomic.LoadInt64(&h.entries[right].frequency) < atomic.LoadInt64(&h.entries[smallest].frequency) {
			smallest = right
		}

		if smallest == idx {
			break
		}

		h.swap(idx, smallest)
		idx = smallest
	}
}

// swap swaps two entries in the heap.
func (h *LFUHeap[K, V]) swap(i, j int) {
	h.entries[i], h.entries[j] = h.entries[j], h.entries[i]
	h.indexMap[h.entries[i].key.(K)] = i
	h.indexMap[h.entries[j].key.(K)] = j
}

// startBackgroundTasks starts background maintenance tasks.
func (c *OptimizedCache[K, V]) startBackgroundTasks() {
	// Cleanup expired entries.
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		ticker := time.NewTicker(c.config.CleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.cleanup()
			case <-c.shutdown:
				return
			}
		}
	}()

	// Metrics collection.
	if c.config.EnableMetrics {
		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					c.updateMetrics()
				case <-c.shutdown:
					return
				}
			}
		}()
	}
}

// cleanup removes expired entries from all shards.
func (c *OptimizedCache[K, V]) cleanup() {
	now := time.Now()
	for i, shard := range c.shards {
		shard.cleanupExpired(now)
		atomic.StoreInt64(&c.metrics.ShardDistribution[i], shard.Size())
	}
}

// cleanupExpired removes expired entries from the shard.
func (s *CacheShard[K, V]) cleanupExpired(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for key, entry := range s.entries {
		if now.After(entry.expiration) {
			delete(s.entries, key)
			s.size -= entry.size
			s.removeFromList(entry)
			atomic.AddInt64(&s.evictions, 1)
		}
	}

	s.lastCleanup = now
}

// updateAccessTime updates average access time metric.
func (c *OptimizedCache[K, V]) updateAccessTime(duration time.Duration) {
	currentAvg := atomic.LoadInt64(&c.metrics.AverageAccessTime)
	newAvg := (currentAvg + duration.Nanoseconds()) / 2
	atomic.StoreInt64(&c.metrics.AverageAccessTime, newAvg)
}

// updateMetrics updates cache metrics.
func (c *OptimizedCache[K, V]) updateMetrics() {
	var totalHits, totalMisses, totalEvictions, totalSize, totalMemory int64

	for i, shard := range c.shards {
		hits := atomic.LoadInt64(&shard.hitCount)
		misses := atomic.LoadInt64(&shard.missCount)
		evictions := atomic.LoadInt64(&shard.evictions)
		size := shard.Size()

		totalHits += hits
		totalMisses += misses
		totalEvictions += evictions
		totalSize += size
		totalMemory += shard.size

		atomic.StoreInt64(&c.metrics.ShardDistribution[i], size)
	}

	atomic.StoreInt64(&c.metrics.Hits, totalHits)
	atomic.StoreInt64(&c.metrics.Misses, totalMisses)
	atomic.StoreInt64(&c.metrics.Evictions, totalEvictions)
	atomic.StoreInt64(&c.metrics.Size, totalSize)
	atomic.StoreInt64(&c.metrics.MemoryUsage, totalMemory)

	// Calculate hit ratio.
	total := totalHits + totalMisses
	if total > 0 {
		c.metrics.HitRatio = float64(totalHits) / float64(total)
	}
}

// resetMetrics resets all cache metrics.
func (c *OptimizedCache[K, V]) resetMetrics() {
	atomic.StoreInt64(&c.metrics.Hits, 0)
	atomic.StoreInt64(&c.metrics.Misses, 0)
	atomic.StoreInt64(&c.metrics.Evictions, 0)
	atomic.StoreInt64(&c.metrics.Size, 0)
	atomic.StoreInt64(&c.metrics.MemoryUsage, 0)
	atomic.StoreInt64(&c.metrics.AverageAccessTime, 0)
	c.metrics.HitRatio = 0
	c.metrics.Throughput = 0
	c.metrics.CompressionRatio = 0

	for i := range c.metrics.ShardDistribution {
		atomic.StoreInt64(&c.metrics.ShardDistribution[i], 0)
	}

	// Reset shard metrics.
	for _, shard := range c.shards {
		atomic.StoreInt64(&shard.hitCount, 0)
		atomic.StoreInt64(&shard.missCount, 0)
		atomic.StoreInt64(&shard.evictions, 0)
	}
}

// GetMetrics returns current cache metrics.
func (c *OptimizedCache[K, V]) GetMetrics() CacheMetrics {
	c.updateMetrics() // Ensure metrics are current

	return CacheMetrics{
		Hits:              atomic.LoadInt64(&c.metrics.Hits),
		Misses:            atomic.LoadInt64(&c.metrics.Misses),
		Evictions:         atomic.LoadInt64(&c.metrics.Evictions),
		Size:              atomic.LoadInt64(&c.metrics.Size),
		MemoryUsage:       atomic.LoadInt64(&c.metrics.MemoryUsage),
		CompressionRatio:  c.metrics.CompressionRatio,
		AverageAccessTime: atomic.LoadInt64(&c.metrics.AverageAccessTime),
		HitRatio:          c.metrics.HitRatio,
		Throughput:        c.metrics.Throughput,
		ShardDistribution: append([]int64(nil), c.metrics.ShardDistribution...),
		Collisions:        atomic.LoadInt64(&c.metrics.Collisions),
		ExpiredEntries:    atomic.LoadInt64(&c.metrics.ExpiredEntries),
	}
}

// GetStats returns detailed cache statistics.
func (c *OptimizedCache[K, V]) GetStats() CacheStats {
	metrics := c.GetMetrics()
	shardStats := make([]ShardStats, len(c.shards))

	for i, shard := range c.shards {
		shardStats[i] = ShardStats{
			ShardID:   i,
			Entries:   shard.Size(),
			Hits:      atomic.LoadInt64(&shard.hitCount),
			Misses:    atomic.LoadInt64(&shard.missCount),
			Evictions: atomic.LoadInt64(&shard.evictions),
			Size:      shard.size,
		}
	}

	var averageEntrySize int64
	if metrics.Size > 0 {
		averageEntrySize = metrics.MemoryUsage / metrics.Size
	}

	memoryEfficiency := float64(0)
	if c.config.MemoryLimit > 0 {
		memoryEfficiency = float64(metrics.MemoryUsage) / float64(c.config.MemoryLimit) * 100
	}

	return CacheStats{
		TotalEntries:     metrics.Size,
		TotalHits:        metrics.Hits,
		TotalMisses:      metrics.Misses,
		TotalEvictions:   metrics.Evictions,
		HitRatio:         metrics.HitRatio,
		AverageEntrySize: averageEntrySize,
		MemoryEfficiency: memoryEfficiency,
		ShardStats:       shardStats,
	}
}

// GetHitRatio returns the current hit ratio.
func (c *OptimizedCache[K, V]) GetHitRatio() float64 {
	return c.metrics.HitRatio
}

// GetAverageAccessTime returns the average access time in microseconds.
func (c *OptimizedCache[K, V]) GetAverageAccessTime() float64 {
	accessTime := atomic.LoadInt64(&c.metrics.AverageAccessTime)
	return float64(accessTime) / 1000 // Convert to microseconds
}

// Keys returns all keys in the cache (expensive operation).
func (c *OptimizedCache[K, V]) Keys() []K {
	var keys []K
	for _, shard := range c.shards {
		shardKeys := shard.Keys()
		keys = append(keys, shardKeys...)
	}
	return keys
}

// Keys returns all keys in the shard.
func (s *CacheShard[K, V]) Keys() []K {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]K, 0, len(s.entries))
	for key := range s.entries {
		keys = append(keys, key)
	}
	return keys
}

// Shutdown gracefully shuts down the cache.
func (c *OptimizedCache[K, V]) Shutdown(ctx context.Context) error {
	close(c.shutdown)

	// Wait for background tasks to finish.
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All tasks finished.
	case <-ctx.Done():
		return ctx.Err()
	}

	// Log final metrics.
	metrics := c.GetMetrics()
	klog.Infof("Cache shutdown - Size: %d, Hit ratio: %.2f%%, Memory usage: %d bytes",
		metrics.Size,
		metrics.HitRatio*100,
		metrics.MemoryUsage,
	)

	return nil
}

// ResetMetrics resets all cache metrics.
func (c *OptimizedCache[K, V]) ResetMetrics() {
	c.resetMetrics()
}
