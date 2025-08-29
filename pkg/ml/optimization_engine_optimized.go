//go:build ml && !test

// Package ml - Optimized version of critical functions from optimization_engine.go.
// This file demonstrates performance improvements without modifying the original code.
package ml

import (
	"context"
	"fmt"
	"sync"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// OptimizedDataGatherer provides concurrent and efficient data gathering.
type OptimizedDataGatherer struct {
	prometheusClient v1.API
	cache            *DataCache
	pool             *DataPointPool
}

// DataCache provides caching for Prometheus query results.
type DataCache struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry
	ttl     time.Duration
}

// CacheEntry represents a cached query result.
type CacheEntry struct {
	data      []DataPoint
	timestamp time.Time
}

// DataPointPool provides object pooling for DataPoint structs.
type DataPointPool struct {
	pool sync.Pool
}

// NewDataPointPool creates a new DataPoint object pool.
func NewDataPointPool() *DataPointPool {
	return &DataPointPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &DataPoint{
					Features: make(map[string]float64, 3),
					Metadata: make(map[string]string, 2),
				}
			},
		},
	}
}

// Get retrieves a DataPoint from the pool.
func (p *DataPointPool) Get() *DataPoint {
	return p.pool.Get().(*DataPoint)
}

// Put returns a DataPoint to the pool.
func (p *DataPointPool) Put(dp *DataPoint) {
	// Clear the maps instead of creating new ones.
	for k := range dp.Features {
		delete(dp.Features, k)
	}
	for k := range dp.Metadata {
		delete(dp.Metadata, k)
	}
	p.pool.Put(dp)
}

// NewDataCache creates a new cache with specified TTL.
func NewDataCache(ttl time.Duration) *DataCache {
	cache := &DataCache{
		entries: make(map[string]*CacheEntry),
		ttl:     ttl,
	}

	// Start cleanup goroutine.
	go cache.cleanupExpired()

	return cache
}

// Get retrieves data from cache if available and not expired.
func (c *DataCache) Get(key string) ([]DataPoint, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return nil, false
	}

	if time.Since(entry.timestamp) > c.ttl {
		return nil, false
	}

	return entry.data, true
}

// Set stores data in cache.
func (c *DataCache) Set(key string, data []DataPoint) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &CacheEntry{
		data:      data,
		timestamp: time.Now(),
	}
}

// cleanupExpired removes expired entries periodically.
func (c *DataCache) cleanupExpired() {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.entries {
			if now.Sub(entry.timestamp) > c.ttl {
				delete(c.entries, key)
			}
		}
		c.mu.Unlock()
	}
}

// GatherHistoricalDataOptimized demonstrates optimized concurrent data gathering.
func GatherHistoricalDataOptimized(ctx context.Context, client v1.API, intent *NetworkIntent, cache *DataCache, pool *DataPointPool) ([]DataPoint, error) {
	// Check cache first.
	cacheKey := fmt.Sprintf("historical:%s", intent.ID)
	if cached, found := cache.Get(cacheKey); found {
		return cached, nil
	}

	// Define time range with adaptive window.
	endTime := time.Now()
	startTime := endTime.Add(-7 * 24 * time.Hour) // Start with 7 days instead of 30

	// Adaptive step size based on time range.
	duration := endTime.Sub(startTime)
	var step time.Duration
	switch {
	case duration > 24*time.Hour*7:
		step = 4 * time.Hour // 4-hour steps for weekly data
	case duration > 24*time.Hour:
		step = time.Hour // Hourly for daily data
	default:
		step = 15 * time.Minute // 15-min for intraday
	}

	// Prepare queries.
	queries := []struct {
		name  string
		query string
	}{
		{"cpu", `avg_over_time(cpu_usage_rate[1h])`},
		{"memory", `avg_over_time(memory_usage_rate[1h])`},
		{"traffic", `rate(http_requests_total[1h])`},
	}

	// Channel for results.
	type queryResult struct {
		name   string
		result model.Value
		err    error
	}
	resultChan := make(chan queryResult, len(queries))

	// Execute queries concurrently.
	var wg sync.WaitGroup
	for _, q := range queries {
		wg.Add(1)
		go func(name, query string) {
			defer wg.Done()

			result, _, err := client.QueryRange(ctx, query, v1.Range{
				Start: startTime,
				End:   endTime,
				Step:  step,
			})

			resultChan <- queryResult{
				name:   name,
				result: result,
				err:    err,
			}
		}(q.name, q.query)
	}

	// Close channel when all queries complete.
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results.
	results := make(map[string]model.Value)
	var firstErr error
	for res := range resultChan {
		if res.err != nil && firstErr == nil {
			firstErr = res.err
		}
		if res.result != nil {
			results[res.name] = res.result
		}
	}

	if firstErr != nil {
		return nil, firstErr
	}

	// Process results efficiently using streaming.
	dataPoints := processResultsStreaming(results, pool)

	// Cache the results.
	cache.Set(cacheKey, dataPoints)

	return dataPoints, nil
}

// processResultsStreaming processes Prometheus results without loading all into memory.
func processResultsStreaming(results map[string]model.Value, pool *DataPointPool) []DataPoint {
	// Pre-allocate slice with estimated capacity.
	estimatedSize := 0
	for _, result := range results {
		if matrix, ok := result.(model.Matrix); ok && len(matrix) > 0 {
			estimatedSize = len(matrix[0].Values)
			break
		}
	}

	dataPoints := make([]DataPoint, 0, estimatedSize)

	// Create a map for efficient timestamp lookup.
	timestampMap := make(map[model.Time]*DataPoint)

	// Process CPU data first to establish timestamps.
	if cpuResult, exists := results["cpu"]; exists {
		if matrix, ok := cpuResult.(model.Matrix); ok && len(matrix) > 0 {
			for _, value := range matrix[0].Values {
				dp := pool.Get()
				// Safe timestamp conversion with overflow protection
				if value.Timestamp >= 0 && value.Timestamp <= float64(1<<62)*1000 {
					dp.Timestamp = time.Unix(int64(value.Timestamp)/1000, 0)
				} else {
					dp.Timestamp = time.Now() // fallback to current time
				}
				dp.Features["cpu_utilization"] = float64(value.Value)
				dp.Metadata["intent_id"] = "optimized"
				dp.Metadata["source"] = "prometheus"

				timestampMap[value.Timestamp] = dp
				dataPoints = append(dataPoints, *dp)
			}
		}
	}

	// Add memory data to existing data points.
	if memResult, exists := results["memory"]; exists {
		if matrix, ok := memResult.(model.Matrix); ok && len(matrix) > 0 {
			for _, value := range matrix[0].Values {
				if dp, found := timestampMap[value.Timestamp]; found {
					dp.Features["memory_utilization"] = float64(value.Value)
				}
			}
		}
	}

	// Add traffic data to existing data points.
	if trafficResult, exists := results["traffic"]; exists {
		if matrix, ok := trafficResult.(model.Matrix); ok && len(matrix) > 0 {
			for _, value := range matrix[0].Values {
				if dp, found := timestampMap[value.Timestamp]; found {
					dp.Features["request_rate"] = float64(value.Value)
				}
			}
		}
	}

	// Return pooled objects.
	for _, dp := range timestampMap {
		pool.Put(dp)
	}

	return dataPoints
}

// StreamingDataProcessor processes data points in chunks to reduce memory usage.
type StreamingDataProcessor struct {
	chunkSize int
	processor func(chunk []DataPoint) error
}

// NewStreamingDataProcessor creates a new streaming processor.
func NewStreamingDataProcessor(chunkSize int, processor func(chunk []DataPoint) error) *StreamingDataProcessor {
	return &StreamingDataProcessor{
		chunkSize: chunkSize,
		processor: processor,
	}
}

// Process handles data points in chunks.
func (s *StreamingDataProcessor) Process(dataPoints []DataPoint) error {
	for i := 0; i < len(dataPoints); i += s.chunkSize {
		end := i + s.chunkSize
		if end > len(dataPoints) {
			end = len(dataPoints)
		}

		chunk := dataPoints[i:end]
		if err := s.processor(chunk); err != nil {
			return err
		}
	}
	return nil
}

// OptimizedScalingRecommendation demonstrates efficient recommendation generation.
func GenerateScalingRecommendationsOptimized(ctx context.Context, data []DataPoint) (*ScalingRecommendation, error) {
	// Use streaming processor for large datasets.
	var (
		sumCPU, sumMemory float64
		maxCPU, maxMemory float64
		count             int
	)

	processor := NewStreamingDataProcessor(100, func(chunk []DataPoint) error {
		for _, point := range chunk {
			if cpu, exists := point.Features["cpu_utilization"]; exists {
				sumCPU += cpu
				if cpu > maxCPU {
					maxCPU = cpu
				}
				count++
			}
			if memory, exists := point.Features["memory_utilization"]; exists {
				sumMemory += memory
				if memory > maxMemory {
					maxMemory = memory
				}
			}
		}
		return nil
	})

	if err := processor.Process(data); err != nil {
		return nil, err
	}

	avgCPU := sumCPU / float64(count)
	avgMemory := sumMemory / float64(count)

	// Generate recommendations based on analysis.
	recommendation := &ScalingRecommendation{
		PredictiveScaling: true,
	}

	// Optimized decision logic.
	switch {
	case maxCPU > 80 || maxMemory > 80:
		recommendation.MinReplicas = 3
		recommendation.MaxReplicas = 20
		recommendation.TargetCPUUtilization = 60
		recommendation.TargetMemoryUtilization = 70
		recommendation.ScaleUpPolicy = "aggressive"
		recommendation.ScaleDownPolicy = "conservative"
	case avgCPU < 30 && avgMemory < 30:
		recommendation.MinReplicas = 1
		recommendation.MaxReplicas = 10
		recommendation.TargetCPUUtilization = 70
		recommendation.TargetMemoryUtilization = 80
		recommendation.ScaleUpPolicy = "moderate"
		recommendation.ScaleDownPolicy = "moderate"
	default:
		recommendation.MinReplicas = 2
		recommendation.MaxReplicas = 15
		recommendation.TargetCPUUtilization = 65
		recommendation.TargetMemoryUtilization = 75
		recommendation.ScaleUpPolicy = "moderate"
		recommendation.ScaleDownPolicy = "moderate"
	}

	return recommendation, nil
}

// CircuitBreaker provides circuit breaker pattern for Prometheus queries.
type CircuitBreaker struct {
	mu           sync.Mutex
	failures     int
	lastFailTime time.Time
	state        string // "closed", "open", "half-open"
	threshold    int
	timeout      time.Duration
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:     "closed",
		threshold: threshold,
		timeout:   timeout,
	}
}

// Call executes the function with circuit breaker protection.
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Check circuit state.
	switch cb.state {
	case "open":
		if time.Since(cb.lastFailTime) > cb.timeout {
			cb.state = "half-open"
			cb.failures = 0
		} else {
			return fmt.Errorf("circuit breaker is open")
		}
	}

	// Execute function.
	err := fn()
	if err != nil {
		cb.failures++
		cb.lastFailTime = time.Now()

		if cb.failures >= cb.threshold {
			cb.state = "open"
			return fmt.Errorf("circuit breaker opened: %w", err)
		}
		return err
	}

	// Success - reset failures.
	if cb.state == "half-open" {
		cb.state = "closed"
	}
	cb.failures = 0

	return nil
}
