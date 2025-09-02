//go:build !disable_rag
// +build !disable_rag

package llm

import (
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// ResponseCache provides unified multi-level caching with intelligent features.

type ResponseCache struct {
	// L1 Cache (in-memory, fastest).

	l1Entries map[string]*CacheEntry

	l1MaxSize int

	l1Mutex sync.RWMutex

	// L2 Cache (persistent, larger).

	l2Entries map[string]*CacheEntry

	l2MaxSize int

	l2Mutex sync.RWMutex

	// Configuration.

	ttl time.Duration

	maxSize int

	// Semantic similarity caching.

	similarityThreshold float64

	semanticIndex map[string][]string // keyword -> cache keys

	semanticMutex sync.RWMutex

	// Adaptive TTL.

	adaptiveTTL bool

	accessFrequency map[string]*AccessStats

	accessMutex sync.RWMutex

	// Cache warming.

	prewarmEnabled bool

	prewarmPatterns []string

	// Cleanup and lifecycle.

	stopCh chan struct{}

	stopOnce sync.Once

	stopped bool

	logger *slog.Logger

	// Metrics.

	metrics *CacheMetrics
}

// CacheEntry represents a cached response with metadata.

type CacheEntry struct {
	Response string `json:"response"`

	Timestamp time.Time `json:"timestamp"`

	HitCount int64 `json:"hit_count"`

	LastAccess time.Time `json:"last_access"`

	TTL time.Duration `json:"ttl"`

	Size int `json:"size"`

	Keywords []string `json:"keywords"`

	SimilarityScore float64 `json:"similarity_score"`

	Level int `json:"level"` // 1 for L1, 2 for L2
}

// AccessStats tracks access patterns for adaptive TTL.

type AccessStats struct {
	AccessCount int64 `json:"access_count"`

	LastAccess time.Time `json:"last_access"`

	AccessPattern []time.Time `json:"access_pattern"`

	AverageInterval time.Duration `json:"average_interval"`
}

// CacheMetrics tracks cache performance.

type CacheMetrics struct {
	L1Hits int64 `json:"l1_hits"`

	L2Hits int64 `json:"l2_hits"`

	Misses int64 `json:"misses"`

	Evictions int64 `json:"evictions"`

	HitRate float64 `json:"hit_rate"`

	L1HitRate float64 `json:"l1_hit_rate"`

	L2HitRate float64 `json:"l2_hit_rate"`

	SemanticHits int64 `json:"semantic_hits"`

	AdaptiveTTLAdjustments int64 `json:"adaptive_ttl_adjustments"`

	mutex sync.RWMutex
}

// CacheConfig holds configuration for the cache system.

type CacheConfig struct {
	TTL time.Duration `json:"ttl"`

	MaxSize int `json:"max_size"`

	L1MaxSize int `json:"l1_max_size"`

	L2MaxSize int `json:"l2_max_size"`

	SimilarityThreshold float64 `json:"similarity_threshold"`

	AdaptiveTTL bool `json:"adaptive_ttl"`

	PrewarmEnabled bool `json:"prewarm_enabled"`

	PrewarmPatterns []string `json:"prewarm_patterns"`
}

// NewResponseCache creates a new unified response cache.

func NewResponseCache(ttl time.Duration, maxSize int) *ResponseCache {
	// For small caches, use single-level (L1 only) to avoid complexity
	var l1MaxSize, l2MaxSize int
	if maxSize <= 4 {
		l1MaxSize = maxSize // All goes to L1
		l2MaxSize = 0       // No L2
	} else {
		l1MaxSize = maxSize / 4 // 25% for L1
		if l1MaxSize < 1 {
			l1MaxSize = 1
		}
		l2MaxSize = maxSize // Total for L2
	}

	config := &CacheConfig{
		TTL: ttl,

		MaxSize: maxSize,

		L1MaxSize: l1MaxSize,

		L2MaxSize: l2MaxSize,

		SimilarityThreshold: 0.85,

		AdaptiveTTL: true,

		PrewarmEnabled: false,
	}

	return NewResponseCacheWithConfig(config)
}

// NewResponseCacheWithConfig creates a cache with specific configuration.

func NewResponseCacheWithConfig(config *CacheConfig) *ResponseCache {
	cache := &ResponseCache{
		l1Entries: make(map[string]*CacheEntry),

		l2Entries: make(map[string]*CacheEntry),

		l1MaxSize: config.L1MaxSize,

		l2MaxSize: config.L2MaxSize,

		ttl: config.TTL,

		maxSize: config.MaxSize,

		similarityThreshold: config.SimilarityThreshold,

		semanticIndex: make(map[string][]string),

		adaptiveTTL: config.AdaptiveTTL,

		accessFrequency: make(map[string]*AccessStats),

		prewarmEnabled: config.PrewarmEnabled,

		prewarmPatterns: config.PrewarmPatterns,

		stopCh: make(chan struct{}),

		stopped: false,

		logger: slog.Default().With("component", "response-cache"),

		metrics: &CacheMetrics{},
	}

	// Start cleanup routine.

	go cache.cleanup()

	// Start cache warming if enabled.

	if cache.prewarmEnabled {
		go cache.prewarmCache()
	}

	return cache
}

// Get retrieves a response from cache with semantic similarity fallback.

func (c *ResponseCache) Get(key string) (string, bool) {
	if c.stopped {
		return "", false
	}

	// Try L1 cache first.

	if response, found := c.getFromL1(key); found {

		c.updateAccessStats(key)

		c.updateMetrics(func(m *CacheMetrics) {
			m.L1Hits++
		})

		return response, true

	}

	// Try L2 cache.

	if response, found := c.getFromL2(key); found {

		c.updateAccessStats(key)

		c.updateMetrics(func(m *CacheMetrics) {
			m.L2Hits++
		})

		// Promote to L1 for frequently accessed items.

		c.promoteToL1(key, response)

		return response, true

	}

	// Try semantic similarity search.

	if response, found := c.getSemanticSimilar(key); found {

		c.updateMetrics(func(m *CacheMetrics) {
			m.SemanticHits++
		})

		return response, true

	}

	// Cache miss.

	c.updateMetrics(func(m *CacheMetrics) {
		m.Misses++
	})

	return "", false
}

// Set stores a response in the appropriate cache level.

func (c *ResponseCache) Set(key, response string) {
	if c.stopped {
		return
	}

	// Extract keywords for semantic indexing.

	keywords := c.extractKeywords(key)

	// Calculate adaptive TTL if enabled.

	ttl := c.ttl

	if c.adaptiveTTL {
		ttl = c.calculateAdaptiveTTL(key)
	}

	entry := &CacheEntry{
		Response: response,

		Timestamp: time.Now(),

		LastAccess: time.Now(),

		HitCount: 0,

		TTL: ttl,

		Size: len(response),

		Keywords: keywords,

		Level: 1,
	}

	// Store in L1 first.

	c.setInL1(key, entry)

	// Update semantic index.

	c.updateSemanticIndex(key, keywords)
}

// getFromL1 retrieves from L1 cache.

func (c *ResponseCache) getFromL1(key string) (string, bool) {
	c.l1Mutex.RLock()

	defer c.l1Mutex.RUnlock()

	entry, exists := c.l1Entries[key]

	if !exists {
		return "", false
	}

	// Check TTL.

	if time.Since(entry.Timestamp) > entry.TTL {
		// Entry expired, will be cleaned up later.

		return "", false
	}

	entry.HitCount++

	entry.LastAccess = time.Now()

	return entry.Response, true
}

// getFromL2 retrieves from L2 cache.

func (c *ResponseCache) getFromL2(key string) (string, bool) {
	c.l2Mutex.RLock()

	defer c.l2Mutex.RUnlock()

	entry, exists := c.l2Entries[key]

	if !exists {
		return "", false
	}

	// Check TTL.

	if time.Since(entry.Timestamp) > entry.TTL {
		return "", false
	}

	entry.HitCount++

	entry.LastAccess = time.Now()

	return entry.Response, true
}

// setInL1 stores entry in L1 cache.

func (c *ResponseCache) setInL1(key string, entry *CacheEntry) {
	c.l1Mutex.Lock()
	defer c.l1Mutex.Unlock()

	// Check if L1 is full and evict if needed
	if len(c.l1Entries) >= c.l1MaxSize {
		c.evictFromL1WithLock()
	}

	entry.Level = 1
	c.l1Entries[key] = entry
}

// setInL2 stores entry in L2 cache.

func (c *ResponseCache) setInL2(key string, entry *CacheEntry) {
	if c.l2MaxSize == 0 {
		return // L2 disabled
	}

	c.l2Mutex.Lock()
	defer c.l2Mutex.Unlock()

	// Check if L2 is full and evict if needed
	if len(c.l2Entries) >= c.l2MaxSize {
		c.evictFromL2WithLock()
	}

	entry.Level = 2
	c.l2Entries[key] = entry
}

// promoteToL1 promotes a frequently accessed L2 entry to L1.

func (c *ResponseCache) promoteToL1(key, response string) {
	c.l2Mutex.RLock()

	entry, exists := c.l2Entries[key]

	c.l2Mutex.RUnlock()

	if !exists {
		return
	}

	// Check if entry is accessed frequently enough.

	if entry.HitCount > 3 {

		// Remove from L2.

		c.l2Mutex.Lock()

		delete(c.l2Entries, key)

		c.l2Mutex.Unlock()

		// Add to L1.

		c.setInL1(key, entry)

	}
}

// evictFromL1 removes least recently used entries from L1.

func (c *ResponseCache) evictFromL1() {
	if len(c.l1Entries) == 0 {
		return
	}

	// Find LRU entry.

	var oldestKey string

	oldestTime := time.Now()

	for key, entry := range c.l1Entries {
		if entry.Timestamp.Before(oldestTime) {

			oldestTime = entry.Timestamp

			oldestKey = key

		}
	}

	if oldestKey != "" {

		// Move to L2 before evicting from L1.

		if entry, exists := c.l1Entries[oldestKey]; exists {
			c.setInL2(oldestKey, entry)
		}

		delete(c.l1Entries, oldestKey)

		c.updateMetrics(func(m *CacheMetrics) {
			m.Evictions++
		})

	}
}

// evictFromL2 removes least recently used entries from L2.

func (c *ResponseCache) evictFromL2() {
	if len(c.l2Entries) == 0 {
		return
	}

	// Find LRU entry.

	var oldestKey string

	oldestTime := time.Now()

	for key, entry := range c.l2Entries {
		if entry.Timestamp.Before(oldestTime) {

			oldestTime = entry.Timestamp

			oldestKey = key

		}
	}

	if oldestKey != "" {

		delete(c.l2Entries, oldestKey)

		c.updateMetrics(func(m *CacheMetrics) {
			m.Evictions++
		})

	}
}

// evictFromL1WithLock removes least recently used entries from L1 (assumes L1 is locked).
func (c *ResponseCache) evictFromL1WithLock() {
	if len(c.l1Entries) == 0 {
		return
	}

	// Find LRU entry.
	var oldestKey string
	oldestTime := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC) // Start with far future time

	for key, entry := range c.l1Entries {
		if entry.Timestamp.Before(oldestTime) {
			oldestTime = entry.Timestamp
			oldestKey = key
		}
	}

	if oldestKey != "" {
		// Move to L2 before evicting from L1, only if L2 is enabled
		if c.l2MaxSize > 0 {
			if entry, exists := c.l1Entries[oldestKey]; exists {
				// Need to lock L2 to move entry
				c.l2Mutex.Lock()
				c.setInL2WithLock(oldestKey, entry)
				c.l2Mutex.Unlock()
			}
		}

		delete(c.l1Entries, oldestKey)

		c.updateMetrics(func(m *CacheMetrics) {
			m.Evictions++
		})
	}
}

// evictFromL2WithLock removes least recently used entries from L2 (assumes L2 is locked).
func (c *ResponseCache) evictFromL2WithLock() {
	if len(c.l2Entries) == 0 {
		return
	}

	// Find LRU entry.
	var oldestKey string
	oldestTime := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC) // Start with far future time

	for key, entry := range c.l2Entries {
		if entry.Timestamp.Before(oldestTime) {
			oldestTime = entry.Timestamp
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(c.l2Entries, oldestKey)

		c.updateMetrics(func(m *CacheMetrics) {
			m.Evictions++
		})
	}
}

// setInL2WithLock stores entry in L2 cache (assumes L2 is locked).
func (c *ResponseCache) setInL2WithLock(key string, entry *CacheEntry) {
	// Check if L2 is full.
	if len(c.l2Entries) >= c.l2MaxSize {
		c.evictFromL2WithLock()
	}

	entry.Level = 2
	c.l2Entries[key] = entry
}

// evictOverallWithLocks removes the least recently used entry from either cache level (assumes both locks held)
func (c *ResponseCache) evictOverallWithLocks() {
	var oldestKey string
	var oldestTime time.Time = time.Now()
	var fromL1 bool

	// Find the oldest entry across both cache levels
	for key, entry := range c.l1Entries {
		if entry.LastAccess.Before(oldestTime) {
			oldestTime = entry.LastAccess
			oldestKey = key
			fromL1 = true
		}
	}

	for key, entry := range c.l2Entries {
		if entry.LastAccess.Before(oldestTime) {
			oldestTime = entry.LastAccess
			oldestKey = key
			fromL1 = false
		}
	}

	// Remove the oldest entry
	if oldestKey != "" {
		if fromL1 {
			delete(c.l1Entries, oldestKey)
		} else {
			delete(c.l2Entries, oldestKey)
		}

		c.updateMetrics(func(m *CacheMetrics) {
			m.Evictions++
		})
	}
}

// evictOverall removes the least recently used entry from either cache level
func (c *ResponseCache) evictOverall() {
	var oldestKey string
	var oldestTime time.Time = time.Now()
	var fromL1 bool

	// Find the oldest entry across both cache levels
	for key, entry := range c.l1Entries {
		if entry.LastAccess.Before(oldestTime) {
			oldestTime = entry.LastAccess
			oldestKey = key
			fromL1 = true
		}
	}

	for key, entry := range c.l2Entries {
		if entry.LastAccess.Before(oldestTime) {
			oldestTime = entry.LastAccess
			oldestKey = key
			fromL1 = false
		}
	}

	// Remove the oldest entry
	if oldestKey != "" {
		if fromL1 {
			delete(c.l1Entries, oldestKey)
		} else {
			delete(c.l2Entries, oldestKey)
		}

		c.updateMetrics(func(m *CacheMetrics) {
			m.Evictions++
		})
	}
}

// getSemanticSimilar finds semantically similar cached entries.

func (c *ResponseCache) getSemanticSimilar(key string) (string, bool) {
	keywords := c.extractKeywords(key)

	if len(keywords) == 0 {
		return "", false
	}

	c.semanticMutex.RLock()

	defer c.semanticMutex.RUnlock()

	// Find candidate keys based on keyword overlap.

	candidates := make(map[string]float64)

	for _, keyword := range keywords {
		if cacheKeys, exists := c.semanticIndex[keyword]; exists {
			for _, cacheKey := range cacheKeys {
				candidates[cacheKey]++
			}
		}
	}

	// Calculate similarity scores and find best match.

	bestKey := ""

	bestScore := 0.0

	for candidateKey, score := range candidates {

		// Normalize score by total keywords.

		normalizedScore := score / float64(len(keywords))

		if normalizedScore > bestScore && normalizedScore >= c.similarityThreshold {

			bestScore = normalizedScore

			bestKey = candidateKey

		}

	}

	if bestKey != "" {

		// Try to get the response from cache.

		if response, found := c.getFromL1(bestKey); found {
			return response, true
		}

		if response, found := c.getFromL2(bestKey); found {
			return response, true
		}

	}

	return "", false
}

// extractKeywords extracts important keywords from cache key.

func (c *ResponseCache) extractKeywords(key string) []string {
	// Simple keyword extraction - split by common separators and filter.

	words := strings.FieldsFunc(strings.ToLower(key), func(c rune) bool {
		return c == ' ' || c == '-' || c == '_' || c == ':' || c == ',' || c == '.'
	})

	// Filter out common stop words and short words.

	stopWords := map[string]bool{
		"a": true, "an": true, "and": true, "are": true, "as": true, "at": true,

		"be": true, "by": true, "for": true, "from": true, "in": true, "is": true,

		"it": true, "of": true, "on": true, "or": true, "that": true, "the": true,

		"to": true, "will": true, "with": true,
	}

	var keywords []string

	for _, word := range words {
		if len(word) > 2 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// updateSemanticIndex updates the semantic index with new keywords.

func (c *ResponseCache) updateSemanticIndex(key string, keywords []string) {
	c.semanticMutex.Lock()

	defer c.semanticMutex.Unlock()

	for _, keyword := range keywords {

		if _, exists := c.semanticIndex[keyword]; !exists {
			c.semanticIndex[keyword] = make([]string, 0)
		}

		c.semanticIndex[keyword] = append(c.semanticIndex[keyword], key)

	}
}

// updateAccessStats tracks access patterns for adaptive TTL.

func (c *ResponseCache) updateAccessStats(key string) {
	if !c.adaptiveTTL {
		return
	}

	c.accessMutex.Lock()

	defer c.accessMutex.Unlock()

	stats, exists := c.accessFrequency[key]

	if !exists {

		stats = &AccessStats{
			AccessPattern: make([]time.Time, 0),
		}

		c.accessFrequency[key] = stats

	}

	now := time.Now()

	stats.AccessCount++

	stats.LastAccess = now

	stats.AccessPattern = append(stats.AccessPattern, now)

	// Keep only recent access pattern (last 10 accesses).

	if len(stats.AccessPattern) > 10 {
		stats.AccessPattern = stats.AccessPattern[len(stats.AccessPattern)-10:]
	}

	// Calculate average interval.

	if len(stats.AccessPattern) > 1 {

		totalInterval := time.Duration(0)

		for i := 1; i < len(stats.AccessPattern); i++ {
			totalInterval += stats.AccessPattern[i].Sub(stats.AccessPattern[i-1])
		}

		stats.AverageInterval = totalInterval / time.Duration(len(stats.AccessPattern)-1)

	}
}

// calculateAdaptiveTTL calculates TTL based on access patterns.

func (c *ResponseCache) calculateAdaptiveTTL(key string) time.Duration {
	c.accessMutex.RLock()

	stats, exists := c.accessFrequency[key]

	c.accessMutex.RUnlock()

	if !exists {
		return c.ttl // Default TTL for new entries
	}

	// Adjust TTL based on access frequency.

	if stats.AccessCount > 10 && stats.AverageInterval > 0 {
		// For frequently accessed items, extend TTL.

		if stats.AverageInterval < time.Hour {
			return c.ttl * 2 // Double TTL for frequently accessed items
		}
	}

	// For rarely accessed items, use default TTL.

	return c.ttl
}

// prewarmCache preloads cache with common patterns.

func (c *ResponseCache) prewarmCache() {
	// This is a placeholder for cache warming logic.

	// In a real implementation, this would load common queries.

	c.logger.Info("Cache prewarming started")

	for _, pattern := range c.prewarmPatterns {
		// Simulate prewarming with pattern.

		c.logger.Debug("Prewarming pattern", slog.String("pattern", pattern))

		// In practice, you'd execute common queries here.
	}
}

// cleanup removes expired entries.

func (c *ResponseCache) cleanup() {
	ticker := time.NewTicker(time.Minute)

	defer ticker.Stop()

	for {
		select {

		case <-c.stopCh:

			return

		case <-ticker.C:

			if c.stopped {
				return
			}

			c.performCleanup()

		}
	}
}

// performCleanup removes expired entries from both cache levels.

func (c *ResponseCache) performCleanup() {
	now := time.Now()

	// Cleanup L1.

	c.l1Mutex.Lock()

	for key, entry := range c.l1Entries {
		if now.Sub(entry.Timestamp) > entry.TTL {
			delete(c.l1Entries, key)
		}
	}

	c.l1Mutex.Unlock()

	// Cleanup L2.

	c.l2Mutex.Lock()

	for key, entry := range c.l2Entries {
		if now.Sub(entry.Timestamp) > entry.TTL {
			delete(c.l2Entries, key)
		}
	}

	c.l2Mutex.Unlock()

	// Cleanup semantic index.

	c.cleanupSemanticIndex()

	// Update hit rates.

	c.updateHitRates()
}

// cleanupSemanticIndex removes references to deleted cache entries.

func (c *ResponseCache) cleanupSemanticIndex() {
	c.semanticMutex.Lock()

	defer c.semanticMutex.Unlock()

	for keyword, cacheKeys := range c.semanticIndex {

		validKeys := make([]string, 0)

		for _, key := range cacheKeys {

			// Check if key still exists in either cache level.

			c.l1Mutex.RLock()

			_, existsL1 := c.l1Entries[key]

			c.l1Mutex.RUnlock()

			c.l2Mutex.RLock()

			_, existsL2 := c.l2Entries[key]

			c.l2Mutex.RUnlock()

			if existsL1 || existsL2 {
				validKeys = append(validKeys, key)
			}

		}

		if len(validKeys) == 0 {
			delete(c.semanticIndex, keyword)
		} else {
			c.semanticIndex[keyword] = validKeys
		}

	}
}

// updateHitRates calculates and updates cache hit rates.

func (c *ResponseCache) updateHitRates() {
	c.updateMetrics(func(m *CacheMetrics) {
		total := m.L1Hits + m.L2Hits + m.Misses

		if total > 0 {

			m.HitRate = float64(m.L1Hits+m.L2Hits) / float64(total)

			m.L1HitRate = float64(m.L1Hits) / float64(total)

			m.L2HitRate = float64(m.L2Hits) / float64(total)

		}
	})
}

// updateMetrics safely updates cache metrics.

func (c *ResponseCache) updateMetrics(updater func(*CacheMetrics)) {
	c.metrics.mutex.Lock()

	defer c.metrics.mutex.Unlock()

	updater(c.metrics)
}

// NewCacheMetrics creates a new CacheMetrics instance
func NewCacheMetrics() *CacheMetrics {
	return &CacheMetrics{}
}

// RecordModelLoad records a model loading operation
func (cm *CacheMetrics) RecordModelLoad(modelName string, deviceID int) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	// This is a basic implementation - could be extended to track per-model stats
	// For now, we'll just track that a model load occurred
}

// RecordCacheMiss records a cache miss
func (cm *CacheMetrics) RecordCacheMiss() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.Misses++
}

// RecordCacheHit records a cache hit at the specified level
func (cm *CacheMetrics) RecordCacheHit(level string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	switch level {
	case "l1_gpu", "l1":
		cm.L1Hits++
	case "l2_memory", "l2":
		cm.L2Hits++
	case "semantic":
		cm.SemanticHits++
	}
}

// RecordCacheOperation records a cache operation with its duration
func (cm *CacheMetrics) RecordCacheOperation(operation string, duration time.Duration) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	// This is a basic implementation - could be extended to track operation-specific metrics
	// For now, we'll just record that an operation occurred
}

// GetMetrics returns current cache metrics.

func (c *ResponseCache) GetMetrics() *CacheMetrics {
	c.metrics.mutex.RLock()

	defer c.metrics.mutex.RUnlock()

	// Create new metrics struct to avoid copying mutex.

	metrics := &CacheMetrics{
		L1Hits: c.metrics.L1Hits,

		L2Hits: c.metrics.L2Hits,

		Misses: c.metrics.Misses,

		Evictions: c.metrics.Evictions,

		HitRate: c.metrics.HitRate,

		L1HitRate: c.metrics.L1HitRate,

		L2HitRate: c.metrics.L2HitRate,

		SemanticHits: c.metrics.SemanticHits,

		AdaptiveTTLAdjustments: c.metrics.AdaptiveTTLAdjustments,
	}

	return metrics
}

// GetStats returns detailed cache statistics.

func (c *ResponseCache) GetStats() map[string]interface{} {
	c.l1Mutex.RLock()

	l1Size := len(c.l1Entries)

	c.l1Mutex.RUnlock()

	c.l2Mutex.RLock()

	l2Size := len(c.l2Entries)

	c.l2Mutex.RUnlock()

	c.semanticMutex.RLock()

	semanticIndexSize := len(c.semanticIndex)

	c.semanticMutex.RUnlock()

	metrics := c.GetMetrics()

	return map[string]interface{}{
		"l1_size": l1Size,

		"l2_size": l2Size,

		"l1_max_size": c.l1MaxSize,

		"l2_max_size": c.l2MaxSize,

		"semantic_index_size": semanticIndexSize,

		"ttl": c.ttl.String(),

		"similarity_threshold": c.similarityThreshold,

		"adaptive_ttl": c.adaptiveTTL,

		"prewarm_enabled": c.prewarmEnabled,

		"hit_rate": metrics.HitRate,

		"l1_hit_rate": metrics.L1HitRate,

		"l2_hit_rate": metrics.L2HitRate,

		"semantic_hits": metrics.SemanticHits,

		"evictions": metrics.Evictions,
	}
}

// Stop gracefully shuts down the cache.

func (c *ResponseCache) Stop() {
	c.stopOnce.Do(func() {
		c.l1Mutex.Lock()

		c.stopped = true

		c.l1Mutex.Unlock()

		close(c.stopCh)

		c.logger.Info("Response cache stopped")
	})
}

// IsStopped returns whether the cache has been stopped.

func (c *ResponseCache) IsStopped() bool {
	c.l1Mutex.RLock()

	defer c.l1Mutex.RUnlock()

	return c.stopped
}

// Clear removes all entries from the cache.

func (c *ResponseCache) Clear() {
	c.l1Mutex.Lock()

	c.l2Mutex.Lock()

	c.semanticMutex.Lock()

	c.accessMutex.Lock()

	defer c.accessMutex.Unlock()

	defer c.semanticMutex.Unlock()

	defer c.l2Mutex.Unlock()

	defer c.l1Mutex.Unlock()

	c.l1Entries = make(map[string]*CacheEntry)

	c.l2Entries = make(map[string]*CacheEntry)

	c.semanticIndex = make(map[string][]string)

	c.accessFrequency = make(map[string]*AccessStats)

	c.logger.Info("Cache cleared")
}

// GetSize returns the total number of cached entries.

func (c *ResponseCache) GetSize() int {
	c.l1Mutex.RLock()

	l1Size := len(c.l1Entries)

	c.l1Mutex.RUnlock()

	c.l2Mutex.RLock()

	l2Size := len(c.l2Entries)

	c.l2Mutex.RUnlock()

	return l1Size + l2Size
}

// generateCacheKey creates a consistent cache key from components.

func GenerateCacheKey(components ...string) string {
	combined := strings.Join(components, "|")

	hash := sha256.Sum256([]byte(combined))

	return hex.EncodeToString(hash[:])
}
