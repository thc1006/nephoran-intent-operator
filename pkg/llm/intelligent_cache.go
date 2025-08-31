//go:build !disable_rag
// +build !disable_rag

package llm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"
)

// IntelligentCache provides advanced caching with intelligent invalidation strategies.

type IntelligentCache struct {

	// Multi-level cache architecture.

	l1Cache *L1Cache // In-memory, ultra-fast

	l2Cache *L2Cache // Distributed cache (Redis-like)

	l3Cache *L3Cache // Persistent storage cache

	// Cache policies and strategies.

	policies *CachePolicies

	invalidator *CacheInvalidator

	warmer *CacheWarmer

	// Analytics and optimization.

	analyzer *CacheAnalyzer

	optimizer *CacheOptimizer

	// Configuration.

	config *IntelligentCacheConfig

	logger *slog.Logger

	// Metrics and monitoring.

	metrics *CacheMetrics

	tracer CacheTracer

	// State management.

	state CacheState

	stateMutex sync.RWMutex
}

// L1Cache represents ultra-fast in-memory cache.

type L1Cache struct {

	// Segmented cache for better concurrency.

	segments []*CacheSegment

	segmentCount int

	segmentMask uint64

	// Cache configuration.

	maxSize int64

	ttl time.Duration

	// Advanced features.

	lru *LRUManager

	bloomFilter *BloomFilter

	// Metrics.

	stats *L1CacheStats

	mutex sync.RWMutex
}

// CacheSegment provides lock-free cache operations for a subset of keys.

type CacheSegment struct {
	entries map[string]*IntelligentCacheEntry

	accessOrder *AccessOrderManager

	size int64

	lastAccess time.Time

	mutex sync.RWMutex

	// Lock-free optimization.

	readOptimized bool

	writeBuffer chan *CacheOperation
}

// IntelligentCacheEntry represents a cached item with comprehensive metadata.

type IntelligentCacheEntry struct {

	// Data.

	Key string

	Value []byte

	OriginalValue interface{} // Keep reference to avoid re-serialization

	// Metadata.

	CreatedAt time.Time

	LastAccessed time.Time

	LastModified time.Time

	ExpiresAt time.Time

	// Access patterns.

	AccessCount int64

	AccessFrequency float64

	AccessPattern CacheAccessPattern

	// Dependency tracking.

	Dependencies []string

	DependentKeys []string

	// Semantic information.

	IntentType string

	Parameters map[string]interface{}

	Similarity float64

	// Cache optimization.

	Size int64

	CompressionType CompressionType

	SerializationType SerializationType

	// Quality metrics.

	HitProbability float64

	CostBenefit float64

	mutex sync.RWMutex
}

// IntelligentCacheConfig holds configuration for the intelligent cache.

type IntelligentCacheConfig struct {

	// Cache levels configuration.

	L1Config L1CacheConfig `json:"l1_config"`

	L2Config L2CacheConfig `json:"l2_config"`

	L3Config L3CacheConfig `json:"l3_config"`

	// Cache policies.

	EvictionPolicy EvictionPolicy `json:"eviction_policy"`

	ReplicationPolicy ReplicationPolicy `json:"replication_policy"`

	ConsistencyLevel ConsistencyLevel `json:"consistency_level"`

	// Intelligence features.

	SemanticCaching bool `json:"semantic_caching"`

	PredictiveCaching bool `json:"predictive_caching"`

	AdaptiveTTL bool `json:"adaptive_ttl"`

	// Performance tuning.

	PrewarmEnabled bool `json:"prewarm_enabled"`

	PrewarmPatterns []string `json:"prewarm_patterns"`

	// Analytics.

	AnalyticsEnabled bool `json:"analytics_enabled"`

	OptimizationEnabled bool `json:"optimization_enabled"`

	// Compression and serialization.

	CompressionThreshold int `json:"compression_threshold"`

	DefaultCompression CompressionType `json:"default_compression"`

	DefaultSerialization SerializationType `json:"default_serialization"`
}

// L1CacheConfig configures the L1 (in-memory) cache.

type L1CacheConfig struct {
	MaxSize int64 `json:"max_size"`

	MaxEntries int `json:"max_entries"`

	TTL time.Duration `json:"ttl"`

	SegmentCount int `json:"segment_count"`

	BloomFilterSize int `json:"bloom_filter_size"`

	LRUEnabled bool `json:"lru_enabled"`
}

// CachePolicies defines intelligent caching strategies.

type CachePolicies struct {

	// Semantic-based policies.

	SemanticSimilarity *SemanticPolicy

	ParameterSimilarity *ParameterPolicy

	IntentClassification *ClassificationPolicy

	// Time-based policies.

	AdaptiveTTL *AdaptiveTTLPolicy

	TimeOfDayPolicy *TimePolicy

	// Usage-based policies.

	AccessFrequency *FrequencyPolicy

	CostBenefitPolicy *CostBenefitPolicy

	// Dependency policies.

	DependencyTracking *DependencyPolicy

	InvalidationCascade *CascadePolicy
}

// SemanticPolicy handles semantic similarity for cache hits.

type SemanticPolicy struct {
	Enabled bool `json:"enabled"`

	SimilarityThreshold float64 `json:"similarity_threshold"`

	EmbeddingModel string `json:"embedding_model"`

	VectorDimension int `json:"vector_dimension"`

	// Embedding cache.

	embeddings map[string][]float32

	embeddingsMutex sync.RWMutex
}

// CacheInvalidator manages intelligent cache invalidation.

type CacheInvalidator struct {

	// Invalidation strategies.

	strategies []InvalidationStrategy

	// Dependency graph.

	dependencyGraph *DependencyGraph

	// Pattern-based invalidation.

	patterns []InvalidationPattern

	// Predictive invalidation.

	predictor *InvalidationPredictor

	// Event-driven invalidation.

	eventSubscriber *EventSubscriber

	config *InvalidationConfig

	logger *slog.Logger
}

// CacheWarmer handles intelligent cache warming.

type CacheWarmer struct {

	// Warming strategies.

	strategies []WarmingStrategy

	// Pattern analysis.

	patternAnalyzer *PatternAnalyzer

	// Predictive warming.

	predictor *WarmingPredictor

	// Scheduler.

	scheduler *WarmingScheduler

	config *WarmingConfig

	logger *slog.Logger
}

// NewIntelligentCache creates a new intelligent cache.

func NewIntelligentCache(config *IntelligentCacheConfig) (*IntelligentCache, error) {

	if config == nil {

		config = getDefaultIntelligentCacheConfig()

	}

	logger := slog.Default().With("component", "intelligent-cache")

	// Create L1 cache with optimized segments.

	l1Cache, err := NewL1Cache(config.L1Config)

	if err != nil {

		return nil, fmt.Errorf("failed to create L1 cache: %w", err)

	}

	// Initialize cache policies.

	policies := &CachePolicies{

		SemanticSimilarity: &SemanticPolicy{

			Enabled: config.SemanticCaching,

			SimilarityThreshold: 0.85,

			EmbeddingModel: "text-embedding-3-small",

			VectorDimension: 1536,

			embeddings: make(map[string][]float32),
		},

		AdaptiveTTL: &AdaptiveTTLPolicy{

			Enabled: config.AdaptiveTTL,

			BaselineTTL: config.L1Config.TTL,

			MinTTL: time.Minute,

			MaxTTL: 24 * time.Hour,

			AccessThreshold: 10,
		},
	}

	// Create invalidator.

	invalidator := &CacheInvalidator{

		strategies: []InvalidationStrategy{},

		dependencyGraph: NewDependencyGraph(),

		patterns: []InvalidationPattern{},

		config: &InvalidationConfig{},

		logger: logger,
	}

	// Create warmer.

	warmer := &CacheWarmer{

		strategies: []WarmingStrategy{},

		patternAnalyzer: NewPatternAnalyzer(),

		config: &WarmingConfig{},

		logger: logger,
	}

	cache := &IntelligentCache{

		l1Cache: l1Cache,

		policies: policies,

		invalidator: invalidator,

		warmer: warmer,

		config: config,

		logger: logger,

		metrics: &CacheMetrics{},

		state: CacheStateActive,
	}

	// Start background processes.

	if config.PrewarmEnabled {

		go cache.warmingRoutine()

	}

	if config.AnalyticsEnabled {

		go cache.analyticsRoutine()

	}

	if config.OptimizationEnabled {

		go cache.optimizationRoutine()

	}

	logger.Info("Intelligent cache initialized",

		"l1_segments", config.L1Config.SegmentCount,

		"semantic_caching", config.SemanticCaching,

		"predictive_caching", config.PredictiveCaching,

		"adaptive_ttl", config.AdaptiveTTL,
	)

	return cache, nil

}

// Get retrieves a value from cache with intelligent matching.

func (ic *IntelligentCache) Get(ctx context.Context, key string) (interface{}, bool, error) {

	start := time.Now()

	defer func() {

		ic.metrics.RecordOperation("get", time.Since(start))

	}()

	// Try exact match first (L1 cache).

	if value, found := ic.l1Cache.Get(key); found {

		ic.metrics.RecordHit("l1")

		return value, true, nil

	}

	// Try semantic similarity matching if enabled.

	if ic.config.SemanticCaching {

		if value, similarity, found := ic.getSemanticMatch(ctx, key); found {

			ic.metrics.RecordHit("semantic")

			ic.logger.Debug("Semantic cache hit",

				"key", key,

				"similarity", similarity,
			)

			// Cache the result for exact future matches.

			ic.l1Cache.Set(key, value, ic.calculateAdaptiveTTL(key, similarity))

			return value, true, nil

		}

	}

	ic.metrics.RecordMiss()

	return nil, false, nil

}

// Set stores a value in cache with intelligent policies.

func (ic *IntelligentCache) Set(ctx context.Context, key string, value interface{}, options ...CacheOption) error {

	start := time.Now()

	defer func() {

		ic.metrics.RecordOperation("set", time.Since(start))

	}()

	// Apply cache options.

	opts := &CacheOptions{}

	for _, option := range options {

		option(opts)

	}

	// Create cache entry with comprehensive metadata.

	entry := &IntelligentCacheEntry{

		Key: key,

		OriginalValue: value,

		CreatedAt: time.Now(),

		LastAccessed: time.Now(),

		LastModified: time.Now(),

		AccessCount: 0,

		IntentType: opts.IntentType,

		Parameters: opts.Parameters,

		Dependencies: opts.Dependencies,
	}

	// Serialize value for storage.

	serializedValue, err := ic.serializeValue(value, opts.Serialization)

	if err != nil {

		return fmt.Errorf("serialization failed: %w", err)

	}

	// Compress if beneficial.

	if len(serializedValue) > ic.config.CompressionThreshold {

		compressedValue, err := ic.compressValue(serializedValue, opts.Compression)

		if err != nil {

			ic.logger.Warn("Compression failed, storing uncompressed", "error", err)

		} else {

			entry.Value = compressedValue

			entry.CompressionType = opts.Compression

		}

	} else {

		entry.Value = serializedValue

	}

	entry.Size = int64(len(entry.Value))

	entry.SerializationType = opts.Serialization

	// Calculate TTL with adaptive policy.

	ttl := ic.calculateAdaptiveTTL(key, 1.0)

	if opts.TTL > 0 {

		ttl = opts.TTL

	}

	entry.ExpiresAt = time.Now().Add(ttl)

	// Store in L1 cache.

	ic.l1Cache.SetEntry(entry)

	// Update semantic embeddings if enabled.

	if ic.config.SemanticCaching {

		go ic.updateSemanticEmbedding(key, value)

	}

	// Update dependency graph.

	if len(opts.Dependencies) > 0 {

		ic.invalidator.dependencyGraph.AddDependencies(key, opts.Dependencies)

	}

	ic.metrics.RecordSet()

	return nil

}

// getSemanticMatch finds semantically similar cached entries.

func (ic *IntelligentCache) getSemanticMatch(ctx context.Context, key string) (interface{}, float64, bool) {

	if !ic.policies.SemanticSimilarity.Enabled {

		return nil, 0, false

	}

	// Get embedding for the key.

	embedding, err := ic.getEmbedding(ctx, key)

	if err != nil {

		ic.logger.Debug("Failed to get embedding", "error", err)

		return nil, 0, false

	}

	bestSimilarity := 0.0

	var bestMatch *IntelligentCacheEntry

	// Compare with cached embeddings.

	ic.policies.SemanticSimilarity.embeddingsMutex.RLock()

	for cachedKey, cachedEmbedding := range ic.policies.SemanticSimilarity.embeddings {

		similarity := ic.calculateCosineSimilarity(embedding, cachedEmbedding)

		if similarity > bestSimilarity && similarity >= ic.policies.SemanticSimilarity.SimilarityThreshold {

			if entry, found := ic.l1Cache.GetEntry(cachedKey); found {

				bestSimilarity = similarity

				bestMatch = entry

			}

		}

	}

	ic.policies.SemanticSimilarity.embeddingsMutex.RUnlock()

	if bestMatch != nil {

		return bestMatch.OriginalValue, bestSimilarity, true

	}

	return nil, 0, false

}

// calculateAdaptiveTTL calculates adaptive TTL based on access patterns and similarity.

func (ic *IntelligentCache) calculateAdaptiveTTL(key string, similarity float64) time.Duration {

	if !ic.policies.AdaptiveTTL.Enabled {

		return ic.config.L1Config.TTL

	}

	policy := ic.policies.AdaptiveTTL

	baseTTL := policy.BaselineTTL

	// Adjust based on similarity (higher similarity = longer TTL).

	similarityFactor := similarity

	// Adjust based on access frequency.

	accessFactor := 1.0

	if entry, found := ic.l1Cache.GetEntry(key); found {

		if entry.AccessCount > int64(policy.AccessThreshold) {

			accessFactor = 1.5 // Increase TTL for frequently accessed items

		}

	}

	// Calculate adaptive TTL.

	adaptiveTTL := time.Duration(float64(baseTTL) * similarityFactor * accessFactor)

	// Apply bounds.

	if adaptiveTTL < policy.MinTTL {

		adaptiveTTL = policy.MinTTL

	}

	if adaptiveTTL > policy.MaxTTL {

		adaptiveTTL = policy.MaxTTL

	}

	return adaptiveTTL

}

// serializeValue serializes value using the specified method.

func (ic *IntelligentCache) serializeValue(value interface{}, method SerializationType) ([]byte, error) {

	switch method {

	case SerializationJSON:

		return ic.serializeJSON(value)

	case SerializationMsgPack:

		return ic.serializeMsgPack(value)

	case SerializationGob:

		return ic.serializeGob(value)

	default:

		return ic.serializeJSON(value) // Default fallback

	}

}

// compressValue compresses value using the specified method.

func (ic *IntelligentCache) compressValue(data []byte, method CompressionType) ([]byte, error) {

	switch method {

	case CompressionGzip:

		return ic.compressGzip(data)

	case CompressionLZ4:

		return ic.compressLZ4(data)

	case CompressionZstd:

		return ic.compressZstd(data)

	default:

		return data, nil // No compression

	}

}

// getEmbedding gets or creates embedding for a text.

func (ic *IntelligentCache) getEmbedding(ctx context.Context, text string) ([]float32, error) {

	// Check if embedding is already cached.

	ic.policies.SemanticSimilarity.embeddingsMutex.RLock()

	if embedding, exists := ic.policies.SemanticSimilarity.embeddings[text]; exists {

		ic.policies.SemanticSimilarity.embeddingsMutex.RUnlock()

		return embedding, nil

	}

	ic.policies.SemanticSimilarity.embeddingsMutex.RUnlock()

	// Generate new embedding (this would call an embedding service).

	embedding, err := ic.generateEmbedding(ctx, text)

	if err != nil {

		return nil, err

	}

	// Cache the embedding.

	ic.policies.SemanticSimilarity.embeddingsMutex.Lock()

	ic.policies.SemanticSimilarity.embeddings[text] = embedding

	ic.policies.SemanticSimilarity.embeddingsMutex.Unlock()

	return embedding, nil

}

// calculateCosineSimilarity calculates cosine similarity between two vectors.

func (ic *IntelligentCache) calculateCosineSimilarity(a, b []float32) float64 {

	if len(a) != len(b) {

		return 0.0

	}

	var dotProduct, normA, normB float64

	for i := range len(a) {

		dotProduct += float64(a[i] * b[i])

		normA += float64(a[i] * a[i])

		normB += float64(b[i] * b[i])

	}

	if normA == 0.0 || normB == 0.0 {

		return 0.0

	}

	return dotProduct / (normA * normB)

}

// generateCacheKey creates an optimized cache key.

func (ic *IntelligentCache) generateCacheKey(intent string, params map[string]interface{}) string {

	// Create deterministic key from intent and parameters.

	h := sha256.New()

	h.Write([]byte(intent))

	// Sort parameters for consistent key generation.

	if params != nil {

		keys := make([]string, 0, len(params))

		for k := range params {

			keys = append(keys, k)

		}

		sort.Strings(keys)

		for _, k := range keys {

			fmt.Fprintf(h, "%s:%v", k, params[k])

		}

	}

	return hex.EncodeToString(h.Sum(nil))[:32] // Use first 32 characters

}

// Background routines.

func (ic *IntelligentCache) warmingRoutine() {

	ticker := time.NewTicker(time.Minute * 5) // Run every 5 minutes

	defer ticker.Stop()

	for {

		select {

		case <-ticker.C:

			ic.performCacheWarming()

		}

	}

}

func (ic *IntelligentCache) analyticsRoutine() {

	ticker := time.NewTicker(time.Minute * 1) // Analyze every minute

	defer ticker.Stop()

	for {

		select {

		case <-ticker.C:

			ic.performAnalytics()

		}

	}

}

func (ic *IntelligentCache) optimizationRoutine() {

	ticker := time.NewTicker(time.Minute * 10) // Optimize every 10 minutes

	defer ticker.Stop()

	for {

		select {

		case <-ticker.C:

			ic.performOptimization()

		}

	}

}

// Placeholder implementations for compilation.

// NewL1Cache performs newl1cache operation.

func NewL1Cache(_ L1CacheConfig) (*L1Cache, error) { return nil, nil }

// Get performs get operation.

func (l1 *L1Cache) Get(_ string) (interface{}, bool) { return nil, false }

// Set performs set operation.

func (l1 *L1Cache) Set(_ string, _ interface{}, _ time.Duration) {}

// SetEntry performs setentry operation.

func (l1 *L1Cache) SetEntry(entry *IntelligentCacheEntry) {}

// GetEntry performs getentry operation.

func (l1 *L1Cache) GetEntry(_ string) (*IntelligentCacheEntry, bool) { return nil, false }

// NewDependencyGraph performs newdependencygraph operation.

func NewDependencyGraph() *DependencyGraph { return &DependencyGraph{} }

// AddDependencies performs adddependencies operation.

func (dg *DependencyGraph) AddDependencies(key string, deps []string) {}

// NewPatternAnalyzer performs newpatternanalyzer operation.

func NewPatternAnalyzer() *PatternAnalyzer { return &PatternAnalyzer{} }

// RecordOperation performs recordoperation operation.

func (cm *CacheMetrics) RecordOperation(op string, duration time.Duration) {}

// RecordHit performs recordhit operation.

func (cm *CacheMetrics) RecordHit(level string) {}

// RecordMiss performs recordmiss operation.

func (cm *CacheMetrics) RecordMiss() {}

// RecordSet performs recordset operation.

func (cm *CacheMetrics) RecordSet() {}

func (ic *IntelligentCache) serializeJSON(_ interface{}) ([]byte, error) { return nil, nil }

func (ic *IntelligentCache) serializeMsgPack(_ interface{}) ([]byte, error) { return nil, nil }

func (ic *IntelligentCache) serializeGob(_ interface{}) ([]byte, error) { return nil, nil }

func (ic *IntelligentCache) compressGzip(data []byte) ([]byte, error) { return data, nil }

func (ic *IntelligentCache) compressLZ4(data []byte) ([]byte, error) { return data, nil }

func (ic *IntelligentCache) compressZstd(data []byte) ([]byte, error) { return data, nil }

func (ic *IntelligentCache) generateEmbedding(ctx context.Context, text string) ([]float32, error) {

	return nil, nil

}

func (ic *IntelligentCache) updateSemanticEmbedding(key string, value interface{}) {}

func (ic *IntelligentCache) performCacheWarming() {}

func (ic *IntelligentCache) performAnalytics() {}

func (ic *IntelligentCache) performOptimization() {}

func getDefaultIntelligentCacheConfig() *IntelligentCacheConfig {

	return &IntelligentCacheConfig{

		L1Config: L1CacheConfig{

			MaxSize: 100 * 1024 * 1024, // 100MB

			MaxEntries: 10000,

			TTL: 30 * time.Minute,

			SegmentCount: 16,

			BloomFilterSize: 10000,

			LRUEnabled: true,
		},

		SemanticCaching: true,

		PredictiveCaching: true,

		AdaptiveTTL: true,

		PrewarmEnabled: true,

		AnalyticsEnabled: true,

		OptimizationEnabled: true,

		CompressionThreshold: 1024,

		DefaultCompression: CompressionLZ4,

		DefaultSerialization: SerializationMsgPack,
	}

}

// Supporting type definitions.

// L2Cache represents a l2cache.

type (
	L2Cache struct{}

	// L3Cache represents a l3cache.

	L3Cache struct{}

	// CacheAnalyzer represents a cacheanalyzer.

	CacheAnalyzer struct{}

	// CacheOptimizer represents a cacheoptimizer (type alias to avoid redeclaration).

	IntelligentCacheOptimizer struct{}

	// CacheTracer represents a cachetracer.

	CacheTracer interface{}

	// CacheState represents a cachestate.

	CacheState int

	// LRUManager represents a lrumanager.

	LRUManager struct{}

	// BloomFilter represents a bloomfilter.

	BloomFilter struct{}

	// L1CacheStats represents a l1cachestats.

	L1CacheStats struct{}

	// CacheOperation represents a cacheoperation.

	CacheOperation struct{}

	// CacheAccessPattern represents a accesspattern.

	CacheAccessPattern int

	// AccessOrderManager represents a accessordermanager.

	AccessOrderManager struct{}
)

// EvictionPolicy represents a evictionpolicy.

type (
	EvictionPolicy int

	// ReplicationPolicy represents a replicationpolicy.

	ReplicationPolicy int

	// ConsistencyLevel represents a consistencylevel.

	ConsistencyLevel int

	// CompressionType represents a compressiontype.

	CompressionType int

	// SerializationType represents a serializationtype.

	SerializationType int
)

// L2CacheConfig represents a l2cacheconfig.

type (
	L2CacheConfig struct{}

	// L3CacheConfig represents a l3cacheconfig.

	L3CacheConfig struct{}
)

// ParameterPolicy represents a parameterpolicy.

type (
	ParameterPolicy struct{}

	// ClassificationPolicy represents a classificationpolicy.

	ClassificationPolicy struct{}

	// AdaptiveTTLPolicy represents a adaptivettlpolicy.

	AdaptiveTTLPolicy struct {
		Enabled bool

		BaselineTTL time.Duration

		MinTTL time.Duration

		MaxTTL time.Duration

		AccessThreshold int
	}
)

// TimePolicy represents a timepolicy.

type (
	TimePolicy struct{}

	// FrequencyPolicy represents a frequencypolicy.

	FrequencyPolicy struct{}

	// CostBenefitPolicy represents a costbenefitpolicy.

	CostBenefitPolicy struct{}

	// DependencyPolicy represents a dependencypolicy.

	DependencyPolicy struct{}

	// CascadePolicy represents a cascadepolicy.

	CascadePolicy struct{}
)

// InvalidationStrategy represents a invalidationstrategy.

type (
	InvalidationStrategy interface{}

	// DependencyGraph represents a dependencygraph.

	DependencyGraph struct{}

	// InvalidationPattern represents a invalidationpattern.

	InvalidationPattern struct{}

	// InvalidationPredictor represents a invalidationpredictor.

	InvalidationPredictor struct{}

	// EventSubscriber represents a eventsubscriber.

	EventSubscriber struct{}

	// InvalidationConfig represents a invalidationconfig.

	InvalidationConfig struct{}
)

// WarmingStrategy represents a warmingstrategy.

type (
	WarmingStrategy interface{}

	// PatternAnalyzer represents a patternanalyzer.

	PatternAnalyzer struct{}

	// WarmingPredictor represents a warmingpredictor.

	WarmingPredictor struct{}

	// WarmingScheduler represents a warmingscheduler.

	WarmingScheduler struct{}

	// WarmingConfig represents a warmingconfig.

	WarmingConfig struct{}
)

// CacheOptions represents a cacheoptions.

type CacheOptions struct {
	TTL time.Duration

	IntentType string

	Parameters map[string]interface{}

	Dependencies []string

	Compression CompressionType

	Serialization SerializationType
}

// CacheOption represents a cacheoption.

type CacheOption func(*CacheOptions)

const (

	// CacheStateActive holds cachestateactive value.
	CacheStateActive CacheState = iota
	// CacheStateShutdown holds cachestateShutdown value.
	CacheStateShutdown

	// CompressionGzip holds compressiongzip value.

	CompressionGzip CompressionType = iota

	// CompressionLZ4 holds compressionlz4 value.

	CompressionLZ4

	// CompressionZstd holds compressionzstd value.

	CompressionZstd

	// SerializationJSON holds serializationjson value.

	SerializationJSON SerializationType = iota

	// SerializationMsgPack holds serializationmsgpack value.

	SerializationMsgPack

	// SerializationGob holds serializationgob value.

	SerializationGob
)
