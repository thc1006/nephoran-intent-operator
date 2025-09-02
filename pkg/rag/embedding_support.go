//go:build !disable_rag && !test

package rag

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// LoadBalancer manages provider selection and load distribution.

type LoadBalancer struct {
	strategy string

	providers []string

	weights map[string]float64

	currentIdx int

	mutex sync.Mutex
}

// NewLoadBalancer performs newloadbalancer operation.

func NewLoadBalancer(strategy string, providers []string) *LoadBalancer {
	return &LoadBalancer{
		strategy: strategy,

		providers: providers,

		weights: make(map[string]float64),
	}
}

// SelectProvider selects the best provider based on strategy.

func (lb *LoadBalancer) SelectProvider(availableProviders []string, request *EmbeddingRequest) string {
	lb.mutex.Lock()

	defer lb.mutex.Unlock()

	if len(availableProviders) == 0 {
		return ""
	}

	if len(availableProviders) == 1 {
		return availableProviders[0]
	}

	switch lb.strategy {

	case "round_robin":

		return lb.roundRobin(availableProviders)

	case "least_cost":

		return lb.leastCost(availableProviders)

	case "fastest":

		return lb.fastest(availableProviders)

	case "quality":

		return lb.bestQuality(availableProviders)

	default:

		return lb.roundRobin(availableProviders)

	}
}

func (lb *LoadBalancer) roundRobin(providers []string) string {
	if lb.currentIdx >= len(providers) {
		lb.currentIdx = 0
	}

	selected := providers[lb.currentIdx]

	lb.currentIdx++

	return selected
}

func (lb *LoadBalancer) leastCost(providers []string) string {
	// For now, return the first provider.

	// In a real implementation, you would calculate costs.

	return providers[0]
}

func (lb *LoadBalancer) fastest(providers []string) string {
	// For now, return the first provider.

	// In a real implementation, you would check latency metrics.

	return providers[0]
}

func (lb *LoadBalancer) bestQuality(providers []string) string {
	// For now, return the first provider.

	// In a real implementation, you would check quality scores.

	return providers[0]
}

// CostManager tracks and manages embedding costs.

type CostManager struct {
	dailySpend map[string]float64

	monthlySpend map[string]float64

	limits CostLimits

	alerts []CostAlert

	mutex sync.RWMutex
}

// QualityManager assesses and ensures embedding quality.

type QualityManager struct {
	minScore float64

	sampleSize int

	testHistory []QualityResult

	qualityTests []QualityTest

	mutex sync.RWMutex
}

// QualityResult represents quality assessment result.

type QualityResult struct {
	Timestamp time.Time `json:"timestamp"`

	Provider string `json:"provider"`

	Model string `json:"model"`

	Score float64 `json:"score"`

	TestScores map[string]float64 `json:"test_scores"`

	SampleSize int `json:"sample_size"`
}

// QualityTest represents a quality test function.

type QualityTest struct {
	Name string

	TestFunc func([]float32) float64

	Weight float64

	Threshold float64
}

// EmbeddingCacheManager manages multi-level caching.

type EmbeddingCacheManager struct {
	l1Enabled bool

	l2Enabled bool

	l1Cache *LRUCache

	l2Cache RedisEmbeddingCache

	metrics *CacheMetrics

	mutex sync.RWMutex
}

// CacheMetrics tracks cache performance.

type CacheMetrics struct {
	// Basic cache metrics.

	Hits int64 `json:"hits"`

	Misses int64 `json:"misses"`

	TotalItems int64 `json:"total_items"`

	Evictions int64 `json:"evictions"`

	mutex sync.RWMutex `json:"-"`

	// L1/L2 cache specific metrics.

	L1Hits int64 `json:"l1_hits"`

	L1Misses int64 `json:"l1_misses"`

	L2Hits int64 `json:"l2_hits"`

	L2Misses int64 `json:"l2_misses"`

	L1HitRate float64 `json:"l1_hit_rate"`

	L2HitRate float64 `json:"l2_hit_rate"`

	TotalHitRate float64 `json:"total_hit_rate"`
}

// LRUCache implements an LRU cache for embeddings.

type LRUCache struct {
	capacity int64

	size int64

	items map[string]*CacheNode

	head *CacheNode

	tail *CacheNode

	mutex sync.Mutex
}

// CacheNode represents a node in the LRU cache.

type CacheNode struct {
	key string

	value []float32

	size int64

	expiry time.Time

	prev *CacheNode

	next *CacheNode
}

// ProviderHealthMonitor monitors provider health.

type ProviderHealthMonitor struct {
	healthChecks map[string]*HealthStatus

	checkInterval time.Duration

	stopChan chan struct{}

	mutex sync.RWMutex
}

// Document represents a document for processing.

type Document struct {
	ID string `json:"id"`

	Content string `json:"content"`

	Size int64 `json:"size"` // Size in bytes for streaming threshold checks

	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewCostManager performs newcostmanager operation.

func NewCostManager(limits CostLimits) *CostManager {
	return &CostManager{
		dailySpend: make(map[string]float64),

		monthlySpend: make(map[string]float64),

		limits: limits,

		alerts: []CostAlert{},
	}
}

// CanAfford checks if the request can be afforded within limits.

func (cm *CostManager) CanAfford(estimatedCost float64) bool {
	cm.mutex.RLock()

	defer cm.mutex.RUnlock()

	today := time.Now().Format("2006-01-02")

	thisMonth := time.Now().Format("2006-01")

	dailySpent := cm.dailySpend[today]

	monthlySpent := cm.monthlySpend[thisMonth]

	if cm.limits.DailyLimit > 0 && dailySpent+estimatedCost > cm.limits.DailyLimit {
		return false
	}

	if cm.limits.MonthlyLimit > 0 && monthlySpent+estimatedCost > cm.limits.MonthlyLimit {
		return false
	}

	return true
}

// RecordSpending records actual spending.

func (cm *CostManager) RecordSpending(cost float64) {
	cm.mutex.Lock()

	defer cm.mutex.Unlock()

	today := time.Now().Format("2006-01-02")

	thisMonth := time.Now().Format("2006-01")

	cm.dailySpend[today] += cost

	cm.monthlySpend[thisMonth] += cost

	// Check for alerts.

	cm.checkAlerts(today, thisMonth)
}

func (cm *CostManager) checkAlerts(today, thisMonth string) {
	dailySpent := cm.dailySpend[today]

	monthlySpent := cm.monthlySpend[thisMonth]

	// Daily alerts.

	if cm.limits.DailyLimit > 0 {

		dailyPercent := dailySpent / cm.limits.DailyLimit

		if dailyPercent >= cm.limits.AlertThreshold {

			alert := CostAlert{
				Timestamp: time.Now(),

				Type: "daily",

				Message: fmt.Sprintf("Daily spending has reached %.1f%% of limit", dailyPercent*100),

				Amount: dailySpent,

				Limit: cm.limits.DailyLimit,
			}

			if dailyPercent >= 0.95 {
				alert.Type = "critical"
			}

			cm.alerts = append(cm.alerts, alert)

		}

	}

	// Monthly alerts.

	if cm.limits.MonthlyLimit > 0 {

		monthlyPercent := monthlySpent / cm.limits.MonthlyLimit

		if monthlyPercent >= cm.limits.AlertThreshold {

			alert := CostAlert{
				Timestamp: time.Now(),

				Type: "monthly",

				Message: fmt.Sprintf("Monthly spending has reached %.1f%% of limit", monthlyPercent*100),

				Amount: monthlySpent,

				Limit: cm.limits.MonthlyLimit,
			}

			if monthlyPercent >= 0.95 {
				alert.Type = "critical"
			}

			cm.alerts = append(cm.alerts, alert)

		}

	}
}

// CostSummary holds cost tracking summary.

type CostSummary struct {
	DailySpending float64 `json:"daily_spending"`

	MonthlySpending float64 `json:"monthly_spending"`

	DailyLimit float64 `json:"daily_limit"`

	MonthlyLimit float64 `json:"monthly_limit"`

	Alerts []CostAlert `json:"alerts"`
}

// GetSummary returns cost summary.

func (cm *CostManager) GetSummary() *CostSummary {
	cm.mutex.RLock()

	defer cm.mutex.RUnlock()

	today := time.Now().Format("2006-01-02")

	thisMonth := time.Now().Format("2006-01")

	return &CostSummary{
		DailySpending: cm.dailySpend[today],

		MonthlySpending: cm.monthlySpend[thisMonth],

		DailyLimit: cm.limits.DailyLimit,

		MonthlyLimit: cm.limits.MonthlyLimit,

		Alerts: cm.alerts,
	}
}

// QualityManager assesses and ensures embedding quality.

func NewQualityManager(minScore float64, sampleSize int) *QualityManager {
	qm := &QualityManager{
		minScore: minScore,

		sampleSize: sampleSize,

		testHistory: []QualityResult{},
	}

	// Initialize quality tests.

	qm.qualityTests = []QualityTest{
		{
			Name: "magnitude_check",

			TestFunc: qm.magnitudeTest,

			Weight: 0.3,

			Threshold: 0.5,
		},

		{
			Name: "dimensionality_check",

			TestFunc: qm.dimensionalityTest,

			Weight: 0.2,

			Threshold: 0.8,
		},

		{
			Name: "distribution_check",

			TestFunc: qm.distributionTest,

			Weight: 0.3,

			Threshold: 0.6,
		},

		{
			Name: "consistency_check",

			TestFunc: qm.consistencyTest,

			Weight: 0.2,

			Threshold: 0.7,
		},
	}

	return qm
}

// AssessQuality assesses the quality of embeddings.

func (qm *QualityManager) AssessQuality(embedding []float32, provider, model string) float64 {
	qm.mutex.Lock()

	defer qm.mutex.Unlock()

	testScores := make(map[string]float64)

	totalScore := 0.0

	totalWeight := 0.0

	for _, test := range qm.qualityTests {

		score := test.TestFunc(embedding)

		testScores[test.Name] = score

		totalScore += score * test.Weight

		totalWeight += test.Weight

	}

	finalScore := totalScore / totalWeight

	// Record result.

	result := QualityResult{
		Timestamp: time.Now(),

		Provider: provider,

		Model: model,

		Score: finalScore,

		TestScores: testScores,

		SampleSize: 1,
	}

	qm.testHistory = append(qm.testHistory, result)

	// Keep only recent history.

	if len(qm.testHistory) > 1000 {
		qm.testHistory = qm.testHistory[len(qm.testHistory)-1000:]
	}

	return finalScore
}

// Quality test functions.

func (qm *QualityManager) magnitudeTest(embedding []float32) float64 {
	// Check if embedding has reasonable magnitude.

	var magnitude float64

	for _, val := range embedding {
		magnitude += float64(val * val)
	}

	magnitude = math.Sqrt(magnitude)

	// Good embeddings typically have magnitude between 0.1 and 10.

	if magnitude >= 0.1 && magnitude <= 10.0 {
		return 1.0
	} else if magnitude >= 0.01 && magnitude <= 100.0 {
		return 0.7
	} else {
		return 0.3
	}
}

func (qm *QualityManager) dimensionalityTest(embedding []float32) float64 {
	// Check if embedding has expected dimensionality (non-zero values).

	nonZeroCount := 0

	for _, val := range embedding {
		if val != 0 {
			nonZeroCount++
		}
	}

	ratio := float64(nonZeroCount) / float64(len(embedding))

	if ratio >= 0.5 {
		return 1.0
	} else if ratio >= 0.2 {
		return 0.7
	} else {
		return 0.3
	}
}

func (qm *QualityManager) distributionTest(embedding []float32) float64 {
	// Check if embedding values have good distribution.

	var mean, variance float64

	// Calculate mean.

	for _, val := range embedding {
		mean += float64(val)
	}

	mean /= float64(len(embedding))

	// Calculate variance.

	for _, val := range embedding {

		diff := float64(val) - mean

		variance += diff * diff

	}

	variance /= float64(len(embedding))

	stddev := math.Sqrt(variance)

	// Good embeddings have moderate standard deviation.

	if stddev >= 0.1 && stddev <= 2.0 {
		return 1.0
	} else if stddev >= 0.01 && stddev <= 5.0 {
		return 0.7
	} else {
		return 0.3
	}
}

func (qm *QualityManager) consistencyTest(embedding []float32) float64 {
	// Check for NaN or infinite values.

	for _, val := range embedding {
		if math.IsNaN(float64(val)) || math.IsInf(float64(val), 0) {
			return 0.0
		}
	}

	return 1.0
}

// EmbeddingCacheManager manages multi-level caching.

func NewEmbeddingCacheManager(config *EmbeddingConfig) (*EmbeddingCacheManager, error) {
	manager := &EmbeddingCacheManager{
		l1Enabled: config.EnableCaching,

		l2Enabled: config.L2CacheEnabled,

		metrics: &CacheMetrics{},
	}

	// Initialize L1 cache (in-memory).

	if manager.l1Enabled {
		manager.l1Cache = NewLRUCache(config.L1CacheSize)
	}

	// Initialize L2 cache (Redis).

	if manager.l2Enabled && config.EnableRedisCache {

		redisClient := redis.NewClient(&redis.Options{
			Addr: config.RedisAddr,

			Password: config.RedisPassword,

			DB: config.RedisDB,
		})

		manager.l2Cache = NewRedisEmbeddingCache(config.RedisAddr, config.RedisPassword, config.RedisDB)

		// Test Redis connection.

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		defer cancel()

		if err := redisClient.Ping(ctx).Err(); err != nil {
			return nil, fmt.Errorf("failed to connect to Redis: %w", err)
		}

	}

	return manager, nil
}

// Get retrieves embedding from cache.

func (ecm *EmbeddingCacheManager) Get(key string) ([]float32, bool) {
	ecm.mutex.RLock()

	defer ecm.mutex.RUnlock()

	// Try L1 cache first.

	if ecm.l1Enabled && ecm.l1Cache != nil {

		if embedding, found := ecm.l1Cache.Get(key); found {

			ecm.metrics.L1Hits++

			ecm.updateHitRates()

			return embedding, true

		}

		ecm.metrics.L1Misses++

	}

	// Try L2 cache.

	if ecm.l2Enabled && ecm.l2Cache != nil {

		if embedding, found, err := ecm.l2Cache.Get(key); err == nil && found {

			ecm.metrics.L2Hits++

			// Promote to L1 cache.

			if ecm.l1Enabled && ecm.l1Cache != nil {
				ecm.l1Cache.Set(key, embedding, time.Hour) // Use shorter TTL for L1
			}

			ecm.updateHitRates()

			return embedding, true

		}

		ecm.metrics.L2Misses++

	}

	ecm.updateHitRates()

	return nil, false
}

// Set stores embedding in cache.

func (ecm *EmbeddingCacheManager) Set(key string, embedding []float32, ttl time.Duration) {
	ecm.mutex.Lock()

	defer ecm.mutex.Unlock()

	// Store in L1 cache.

	if ecm.l1Enabled && ecm.l1Cache != nil {
		ecm.l1Cache.Set(key, embedding, ttl)
	}

	// Store in L2 cache.

	if ecm.l2Enabled && ecm.l2Cache != nil {
		ecm.l2Cache.Set(key, embedding, ttl)
	}
}

func (ecm *EmbeddingCacheManager) updateHitRates() {
	totalL1 := ecm.metrics.L1Hits + ecm.metrics.L1Misses

	if totalL1 > 0 {
		ecm.metrics.L1HitRate = float64(ecm.metrics.L1Hits) / float64(totalL1)
	}

	totalL2 := ecm.metrics.L2Hits + ecm.metrics.L2Misses

	if totalL2 > 0 {
		ecm.metrics.L2HitRate = float64(ecm.metrics.L2Hits) / float64(totalL2)
	}

	totalHits := ecm.metrics.L1Hits + ecm.metrics.L2Hits

	totalRequests := totalL1 + totalL2

	if totalRequests > 0 {
		ecm.metrics.TotalHitRate = float64(totalHits) / float64(totalRequests)
	}
}

// Close closes the cache manager.

func (ecm *EmbeddingCacheManager) Close() error {
	// For now, just return nil - interface does not expose client directly.

	return nil
}

// LRUCache implements an LRU cache for embeddings.

func NewLRUCache(capacity int64) *LRUCache {
	cache := &LRUCache{
		capacity: capacity,

		items: make(map[string]*CacheNode),
	}

	// Initialize dummy head and tail nodes.

	cache.head = &CacheNode{}

	cache.tail = &CacheNode{}

	cache.head.next = cache.tail

	cache.tail.prev = cache.head

	return cache
}

// Get retrieves an embedding from the LRU cache.

func (cache *LRUCache) Get(key string) ([]float32, bool) {
	cache.mutex.Lock()

	defer cache.mutex.Unlock()

	node, exists := cache.items[key]

	if !exists {
		return nil, false
	}

	// Check if expired.

	if time.Now().After(node.expiry) {

		cache.removeNode(node)

		delete(cache.items, key)

		return nil, false

	}

	// Move to front (most recently used).

	cache.moveToFront(node)

	return node.value, true
}

// Set stores an embedding in the LRU cache.

func (cache *LRUCache) Set(key string, value []float32, ttl time.Duration) {
	cache.mutex.Lock()

	defer cache.mutex.Unlock()

	// Calculate size (rough estimate).

	nodeSize := int64(len(value)*4 + len(key) + 64) // 4 bytes per float32 + key + overhead

	if existingNode, exists := cache.items[key]; exists {

		// Update existing node.

		cache.size -= existingNode.size

		existingNode.value = value

		existingNode.size = nodeSize

		existingNode.expiry = time.Now().Add(ttl)

		cache.size += nodeSize

		cache.moveToFront(existingNode)

	} else {

		// Create new node.

		newNode := &CacheNode{
			key: key,

			value: value,

			size: nodeSize,

			expiry: time.Now().Add(ttl),
		}

		cache.items[key] = newNode

		cache.addToFront(newNode)

		cache.size += nodeSize

	}

	// Evict if over capacity.

	for cache.size > cache.capacity {
		cache.evictLRU()
	}
}

func (cache *LRUCache) addToFront(node *CacheNode) {
	node.prev = cache.head

	node.next = cache.head.next

	cache.head.next.prev = node

	cache.head.next = node
}

func (cache *LRUCache) removeNode(node *CacheNode) {
	node.prev.next = node.next

	node.next.prev = node.prev

	cache.size -= node.size
}

func (cache *LRUCache) moveToFront(node *CacheNode) {
	cache.removeNode(node)

	cache.addToFront(node)
}

func (cache *LRUCache) evictLRU() {
	lru := cache.tail.prev

	if lru != cache.head {

		delete(cache.items, lru.key)

		cache.removeNode(lru)

	}
}

// ProviderHealthMonitor monitors provider health.

func NewProviderHealthMonitor(checkInterval time.Duration) *ProviderHealthMonitor {
	return &ProviderHealthMonitor{
		healthChecks: make(map[string]*HealthStatus),

		checkInterval: checkInterval,

		stopChan: make(chan struct{}),
	}
}

// StartMonitoring starts health monitoring for providers.

func (phm *ProviderHealthMonitor) StartMonitoring(providers map[string]EmbeddingProvider) {
	ticker := time.NewTicker(phm.checkInterval)

	defer ticker.Stop()

	for {
		select {

		case <-phm.stopChan:

			return

		case <-ticker.C:

			phm.checkProviderHealth(providers)

		}
	}
}

func (phm *ProviderHealthMonitor) checkProviderHealth(providers map[string]EmbeddingProvider) {
	phm.mutex.Lock()

	defer phm.mutex.Unlock()

	for name, provider := range providers {

		status, exists := phm.healthChecks[name]

		if !exists {

			status = &HealthStatus{}

			phm.healthChecks[name] = status

		}

		// Update health status.

		status.IsHealthy = provider.IsHealthy()

		status.LastCheck = time.Now()

		status.AverageLatency = provider.GetLatency()

		// Simple health check - you could implement more sophisticated checks.

		if status.IsHealthy {
			status.ConsecutiveFailures = 0
		} else {
			status.ConsecutiveFailures++
		}

	}
}

// GetStatus returns health status for all providers.

func (phm *ProviderHealthMonitor) GetStatus() map[string]*HealthStatus {
	phm.mutex.RLock()

	defer phm.mutex.RUnlock()

	// Return a copy.

	status := make(map[string]*HealthStatus)

	for k, v := range phm.healthChecks {

		statusCopy := *v

		status[k] = &statusCopy

	}

	return status
}

// Stop stops health monitoring.

func (phm *ProviderHealthMonitor) Stop() {
	close(phm.stopChan)
}
