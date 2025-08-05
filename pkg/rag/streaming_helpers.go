//go:build !disable_rag && !test

package rag

import (
	"runtime"
	"sync/atomic"
	"time"
)

// MemoryMonitor tracks memory usage for backpressure handling
type MemoryMonitor struct {
	maxMemory       int64
	currentMemory   atomic.Int64
}

// NewMemoryMonitor creates a new memory monitor
func NewMemoryMonitor(maxMemory int64) *MemoryMonitor {
	return &MemoryMonitor{
		maxMemory: maxMemory,
	}
}

// GetMemoryUsage returns current and max memory usage
func (mm *MemoryMonitor) GetMemoryUsage() (current, max int64) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	// Use actual memory stats
	current = int64(memStats.Alloc)
	
	// For testing, also consider our tracked memory
	tracked := mm.currentMemory.Load()
	if tracked > current {
		current = tracked
	}
	
	return current, mm.maxMemory
}

// AllocateMemory simulates memory allocation (for testing)
func (mm *MemoryMonitor) AllocateMemory(bytes int64) {
	mm.currentMemory.Add(bytes)
}

// FreeMemory simulates memory deallocation (for testing)
func (mm *MemoryMonitor) FreeMemory(bytes int64) {
	mm.currentMemory.Add(-bytes)
}

// ProcessingPool manages concurrent document processing
type ProcessingPool struct {
	maxConcurrent int
	semaphore     chan struct{}
}

// NewProcessingPool creates a new processing pool
func NewProcessingPool(maxConcurrent int) *ProcessingPool {
	return &ProcessingPool{
		maxConcurrent: maxConcurrent,
		semaphore:     make(chan struct{}, maxConcurrent),
	}
}

// Acquire acquires a processing slot
func (pp *ProcessingPool) Acquire() {
	pp.semaphore <- struct{}{}
}

// Release releases a processing slot
func (pp *ProcessingPool) Release() {
	<-pp.semaphore
}

// TryAcquire attempts to acquire a processing slot without blocking
func (pp *ProcessingPool) TryAcquire() bool {
	select {
	case pp.semaphore <- struct{}{}:
		return true
	default:
		return false
	}
}

// NoOpRedisCache is a no-op implementation of RedisEmbeddingCache
type NoOpRedisCache struct{}

func (n *NoOpRedisCache) Get(key string) ([]float32, bool, error) {
	return nil, false, nil
}

func (n *NoOpRedisCache) Set(key string, embedding []float32, ttl time.Duration) error {
	return nil
}

func (n *NoOpRedisCache) Delete(key string) error {
	return nil
}

func (n *NoOpRedisCache) Clear() error {
	return nil
}

func (n *NoOpRedisCache) Stats() CacheStats {
	return CacheStats{}
}

func (n *NoOpRedisCache) Close() error {
	return nil
}

// getDefaultChunkingConfig returns default chunking configuration
func getDefaultChunkingConfig() *ChunkingConfig {
	return &ChunkingConfig{
		ChunkSize:        1024,
		ChunkOverlap:     128,
		MinChunkSize:     100,
		MaxChunkSize:     4096,
		PreserveHierarchy: true,
		MaxHierarchyDepth: 5,
		IncludeParentContext: true,
		UseSemanticBoundaries: true,
		SentenceBoundaryWeight: 1.0,
		ParagraphBoundaryWeight: 2.0,
		SectionBoundaryWeight: 3.0,
		PreserveTechnicalTerms: true,
		TechnicalTermPatterns: []string{
			`\b[A-Z]{2,}(?:-[A-Z]{2,})*\b`,
			`\b\d+G\b`,
			`\b(?:Rel|Release)[-\s]*\d+\b`,
			`\b[vV]\d+\.\d+(?:\.\d+)?\b`,
		},
		PreserveTablesAndFigures: true,
		PreserveCodeBlocks: true,
		MinContentRatio: 0.3,
		MaxEmptyLines: 5,
		FilterNoiseContent: true,
		BatchSize: 10,
		MaxConcurrency: 4,
		AddSectionHeaders: true,
		AddDocumentMetadata: true,
		AddChunkMetadata: true,
	}
}

// getDefaultEmbeddingConfig returns default embedding configuration
func getDefaultEmbeddingConfig() *EmbeddingConfig {
	return &EmbeddingConfig{
		Provider:     "openai",
		ModelName:    "text-embedding-3-large",
		Dimensions:   3072,
		MaxTokens:    8191,
		BatchSize:    100,
		MaxConcurrency: 5,
		RequestTimeout: 30 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    1 * time.Second,
		RateLimit:     3000,
		TokenRateLimit: 1000000,
		MinTextLength: 10,
		MaxTextLength: 8000,
		NormalizeText: true,
		RemoveStopWords: false,
		EnableCaching: true,
		CacheTTL:      24 * time.Hour,
		EnableRedisCache: false,
		L1CacheSize:   1000,
		L2CacheEnabled: false,
		FallbackEnabled: true,
		LoadBalancing: "least_cost",
		HealthCheckInterval: 5 * time.Minute,
		EnableCostTracking: true,
		DailyCostLimit: 100.0,
		MonthlyCostLimit: 2000.0,
		CostAlertThreshold: 80.0,
		EnableQualityCheck: true,
		MinQualityScore: 0.7,
		QualityCheckSample: 10,
		TelecomPreprocessing: true,
		PreserveTechnicalTerms: true,
		TechnicalTermWeighting: 1.5,
		EnableMetrics: true,
		MetricsInterval: 5 * time.Minute,
	}
}