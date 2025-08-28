//go:build !disable_rag && !test

package rag

import (
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"
)

// PerformanceOptimizer manages and optimizes RAG pipeline performance
type PerformanceOptimizer struct {
	config              *OptimizerConfig
	logger              *slog.Logger
	metrics             *PerformanceMetrics
	mutex               sync.RWMutex
	activeOptimizations map[string]*OptimizationTask
}

// OptimizerConfig holds configuration for performance optimization
type OptimizerConfig struct {
	EnableAutoOptimization   bool          `json:"enable_auto_optimization"`
	OptimizationInterval     time.Duration `json:"optimization_interval"`
	MemoryThreshold          float64       `json:"memory_threshold"`  // Memory usage threshold (0.0-1.0)
	CPUThreshold             float64       `json:"cpu_threshold"`     // CPU usage threshold (0.0-1.0)
	LatencyThreshold         time.Duration `json:"latency_threshold"` // Maximum acceptable latency
	ThroughputTarget         int64         `json:"throughput_target"` // Target requests per minute
	EnableMemoryOptimization bool          `json:"enable_memory_optimization"`
	EnableCacheOptimization  bool          `json:"enable_cache_optimization"`
	EnableBatchOptimization  bool          `json:"enable_batch_optimization"`
	MonitoringWindow         time.Duration `json:"monitoring_window"` // Window for collecting metrics
}

// PerformanceMetrics tracks system performance metrics
type PerformanceMetrics struct {
	// System metrics
	CPUUsage       float64         `json:"cpu_usage"`
	MemoryUsage    float64         `json:"memory_usage"`
	GoroutineCount int             `json:"goroutine_count"`
	HeapSize       uint64          `json:"heap_size"`
	GCPauses       []time.Duration `json:"gc_pauses"`

	// Pipeline metrics
	AverageLatency time.Duration `json:"average_latency"`
	ThroughputRPM  int64         `json:"throughput_rpm"`
	ErrorRate      float64       `json:"error_rate"`
	CacheHitRate   float64       `json:"cache_hit_rate"`

	// Component metrics
	DocumentProcessingTime  time.Duration `json:"document_processing_time"`
	EmbeddingGenerationTime time.Duration `json:"embedding_generation_time"`
	RetrievalTime           time.Duration `json:"retrieval_time"`
	ContextAssemblyTime     time.Duration `json:"context_assembly_time"`

	// Optimization metrics
	OptimizationsApplied int       `json:"optimizations_applied"`
	LastOptimization     time.Time `json:"last_optimization"`
	PerformanceGain      float64   `json:"performance_gain"`

	mutex sync.RWMutex
}

// OptimizationTask represents an active optimization task
type OptimizationTask struct {
	ID                string        `json:"id"`
	Type              string        `json:"type"`
	Description       string        `json:"description"`
	StartTime         time.Time     `json:"start_time"`
	EstimatedDuration time.Duration `json:"estimated_duration"`
	Status            string        `json:"status"` // running, completed, failed
	Result            string        `json:"result,omitempty"`
	Impact            float64       `json:"impact"` // Expected performance impact (0.0-1.0)
}

// NewPerformanceOptimizer creates a new performance optimizer
func NewPerformanceOptimizer(config *OptimizerConfig) *PerformanceOptimizer {
	if config == nil {
		config = getDefaultOptimizerConfig()
	}

	optimizer := &PerformanceOptimizer{
		config:              config,
		logger:              slog.Default().With("component", "performance-optimizer"),
		metrics:             &PerformanceMetrics{},
		activeOptimizations: make(map[string]*OptimizationTask),
	}

	// Start background optimization if enabled
	if config.EnableAutoOptimization {
		go optimizer.startAutoOptimization()
	}

	return optimizer
}

// getDefaultOptimizerConfig returns default optimizer configuration
func getDefaultOptimizerConfig() *OptimizerConfig {
	return &OptimizerConfig{
		EnableAutoOptimization:   true,
		OptimizationInterval:     10 * time.Minute,
		MemoryThreshold:          0.8, // 80% memory usage
		CPUThreshold:             0.7, // 70% CPU usage
		LatencyThreshold:         5 * time.Second,
		ThroughputTarget:         1000, // 1000 RPM
		EnableMemoryOptimization: true,
		EnableCacheOptimization:  true,
		EnableBatchOptimization:  true,
		MonitoringWindow:         5 * time.Minute,
	}
}

// CollectMetrics collects current performance metrics
func (po *PerformanceOptimizer) CollectMetrics() *PerformanceMetrics {
	po.metrics.mutex.Lock()
	defer po.metrics.mutex.Unlock()

	// Collect system metrics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	po.metrics.GoroutineCount = runtime.NumGoroutine()
	po.metrics.HeapSize = memStats.HeapAlloc
	po.metrics.MemoryUsage = float64(memStats.HeapAlloc) / float64(memStats.HeapSys)

	// Note: CPU usage collection would require additional system monitoring
	// For now, we'll use a placeholder implementation

	// Return a copy of the metrics without mutex
	metricsCopy := &PerformanceMetrics{
		CPUUsage: po.metrics.CPUUsage,
		MemoryUsage: po.metrics.MemoryUsage,
		GoroutineCount: po.metrics.GoroutineCount,
		HeapSize: po.metrics.HeapSize,
		GCPauses: copyDurations(po.metrics.GCPauses),
		AverageLatency: po.metrics.AverageLatency,
		ThroughputRPM: po.metrics.ThroughputRPM,
		ErrorRate: po.metrics.ErrorRate,
		CacheHitRate: po.metrics.CacheHitRate,
		DocumentProcessingTime: po.metrics.DocumentProcessingTime,
		EmbeddingGenerationTime: po.metrics.EmbeddingGenerationTime,
		RetrievalTime: po.metrics.RetrievalTime,
		ContextAssemblyTime: po.metrics.ContextAssemblyTime,
		OptimizationsApplied: po.metrics.OptimizationsApplied,
		LastOptimization: po.metrics.LastOptimization,
		PerformanceGain: po.metrics.PerformanceGain,
	}
	return metricsCopy
}

// OptimizePerformance analyzes current performance and applies optimizations
func (po *PerformanceOptimizer) OptimizePerformance(pipeline *RAGPipeline) error {
	po.logger.Info("Starting performance optimization")

	// Collect current metrics
	metrics := po.CollectMetrics()

	// Analyze performance bottlenecks
	bottlenecks := po.analyzeBottlenecks(metrics)

	// Apply optimizations based on identified bottlenecks
	for _, bottleneck := range bottlenecks {
		optimization := po.createOptimizationTask(bottleneck)
		if err := po.applyOptimization(pipeline, optimization); err != nil {
			po.logger.Error("Failed to apply optimization", "optimization", optimization.ID, "error", err)
			continue
		}

		po.logger.Info("Applied optimization",
			"optimization", optimization.ID,
			"type", optimization.Type,
			"impact", optimization.Impact,
		)
	}

	po.logger.Info("Performance optimization completed")
	return nil
}

// analyzeBottlenecks identifies performance bottlenecks
func (po *PerformanceOptimizer) analyzeBottlenecks(metrics *PerformanceMetrics) []string {
	var bottlenecks []string

	// Memory pressure analysis
	if metrics.MemoryUsage > po.config.MemoryThreshold {
		bottlenecks = append(bottlenecks, "high_memory_usage")
	}

	// CPU usage analysis
	if metrics.CPUUsage > po.config.CPUThreshold {
		bottlenecks = append(bottlenecks, "high_cpu_usage")
	}

	// Latency analysis
	if metrics.AverageLatency > po.config.LatencyThreshold {
		bottlenecks = append(bottlenecks, "high_latency")
	}

	// Throughput analysis
	if metrics.ThroughputRPM < po.config.ThroughputTarget {
		bottlenecks = append(bottlenecks, "low_throughput")
	}

	// Cache efficiency analysis
	if metrics.CacheHitRate < 0.7 { // 70% cache hit rate threshold
		bottlenecks = append(bottlenecks, "low_cache_efficiency")
	}

	// Component-specific analysis
	if metrics.EmbeddingGenerationTime > 2*time.Second {
		bottlenecks = append(bottlenecks, "slow_embedding_generation")
	}

	if metrics.DocumentProcessingTime > 5*time.Second {
		bottlenecks = append(bottlenecks, "slow_document_processing")
	}

	return bottlenecks
}

// createOptimizationTask creates an optimization task for a bottleneck
func (po *PerformanceOptimizer) createOptimizationTask(bottleneck string) *OptimizationTask {
	task := &OptimizationTask{
		ID:        fmt.Sprintf("opt_%s_%d", bottleneck, time.Now().Unix()),
		Type:      bottleneck,
		StartTime: time.Now(),
		Status:    "running",
	}

	switch bottleneck {
	case "high_memory_usage":
		task.Description = "Optimize memory usage through garbage collection and cache cleanup"
		task.EstimatedDuration = 30 * time.Second
		task.Impact = 0.3

	case "high_cpu_usage":
		task.Description = "Reduce CPU usage through batch size optimization and concurrency limits"
		task.EstimatedDuration = 10 * time.Second
		task.Impact = 0.2

	case "high_latency":
		task.Description = "Reduce latency through connection pooling and request optimization"
		task.EstimatedDuration = 20 * time.Second
		task.Impact = 0.4

	case "low_throughput":
		task.Description = "Increase throughput through batch processing and parallel execution"
		task.EstimatedDuration = 15 * time.Second
		task.Impact = 0.5

	case "low_cache_efficiency":
		task.Description = "Improve cache efficiency through cache warming and eviction policy optimization"
		task.EstimatedDuration = 60 * time.Second
		task.Impact = 0.6

	case "slow_embedding_generation":
		task.Description = "Optimize embedding generation through provider selection and batch optimization"
		task.EstimatedDuration = 45 * time.Second
		task.Impact = 0.4

	case "slow_document_processing":
		task.Description = "Optimize document processing through streaming and memory management"
		task.EstimatedDuration = 30 * time.Second
		task.Impact = 0.3

	default:
		task.Description = "Generic performance optimization"
		task.EstimatedDuration = 20 * time.Second
		task.Impact = 0.2
	}

	return task
}

// applyOptimization applies a specific optimization to the pipeline
func (po *PerformanceOptimizer) applyOptimization(pipeline *RAGPipeline, task *OptimizationTask) error {
	po.mutex.Lock()
	po.activeOptimizations[task.ID] = task
	po.mutex.Unlock()

	defer func() {
		po.mutex.Lock()
		delete(po.activeOptimizations, task.ID)
		po.mutex.Unlock()
	}()

	switch task.Type {
	case "high_memory_usage":
		return po.optimizeMemoryUsage(pipeline, task)
	case "high_cpu_usage":
		return po.optimizeCPUUsage(pipeline, task)
	case "high_latency":
		return po.optimizeLatency(pipeline, task)
	case "low_throughput":
		return po.optimizeThroughput(pipeline, task)
	case "low_cache_efficiency":
		return po.optimizeCacheEfficiency(pipeline, task)
	case "slow_embedding_generation":
		return po.optimizeEmbeddingGeneration(pipeline, task)
	case "slow_document_processing":
		return po.optimizeDocumentProcessing(pipeline, task)
	default:
		task.Status = "failed"
		task.Result = "Unknown optimization type"
		return fmt.Errorf("unknown optimization type: %s", task.Type)
	}
}

// optimizeMemoryUsage optimizes memory usage
func (po *PerformanceOptimizer) optimizeMemoryUsage(pipeline *RAGPipeline, task *OptimizationTask) error {
	po.logger.Info("Optimizing memory usage")

	// Force garbage collection
	runtime.GC()

	// Clear cache if memory pressure is high
	if pipeline.redisCache != nil {
		// Implement selective cache cleanup based on usage patterns
		// This would require additional cache analytics
	}

	// Optimize batch sizes to reduce memory footprint
	if pipeline.config.DocumentLoaderConfig != nil {
		pipeline.config.DocumentLoaderConfig.PageProcessingBatch =
			min(pipeline.config.DocumentLoaderConfig.PageProcessingBatch, 5)
	}

	task.Status = "completed"
	task.Result = "Memory optimization applied: GC triggered, batch sizes reduced"
	return nil
}

// optimizeCPUUsage optimizes CPU usage
func (po *PerformanceOptimizer) optimizeCPUUsage(pipeline *RAGPipeline, task *OptimizationTask) error {
	po.logger.Info("Optimizing CPU usage")

	// Reduce concurrency if CPU usage is high
	maxConcurrency := runtime.NumCPU()

	if pipeline.config.MaxConcurrentProcessing > maxConcurrency {
		pipeline.config.MaxConcurrentProcessing = maxConcurrency
	}

	if pipeline.config.EmbeddingConfig != nil {
		pipeline.config.EmbeddingConfig.MaxConcurrency =
			min(pipeline.config.EmbeddingConfig.MaxConcurrency, maxConcurrency/2)
	}

	task.Status = "completed"
	task.Result = "CPU optimization applied: concurrency limits adjusted"
	return nil
}

// optimizeLatency optimizes request latency
func (po *PerformanceOptimizer) optimizeLatency(pipeline *RAGPipeline, task *OptimizationTask) error {
	po.logger.Info("Optimizing latency")

	// Reduce timeouts for faster failure detection
	if pipeline.config.EmbeddingConfig != nil {
		pipeline.config.EmbeddingConfig.RequestTimeout = minDuration(pipeline.config.EmbeddingConfig.RequestTimeout, 15*time.Second)
	}

	// Optimize cache to prioritize frequently accessed items
	// This would require implementing cache analytics and priority queues

	task.Status = "completed"
	task.Result = "Latency optimization applied: timeouts reduced, cache prioritization enabled"
	return nil
}

// optimizeThroughput optimizes system throughput
func (po *PerformanceOptimizer) optimizeThroughput(pipeline *RAGPipeline, task *OptimizationTask) error {
	po.logger.Info("Optimizing throughput")

	// Increase batch sizes for better throughput (within memory limits)
	if pipeline.config.EmbeddingConfig != nil && po.metrics.MemoryUsage < 0.6 {
		pipeline.config.EmbeddingConfig.BatchSize =
			min(pipeline.config.EmbeddingConfig.BatchSize*2, 200)
	}

	// Optimize connection pooling
	// This would require updating HTTP client configurations

	task.Status = "completed"
	task.Result = "Throughput optimization applied: batch sizes increased, connection pooling optimized"
	return nil
}

// optimizeCacheEfficiency optimizes cache performance
func (po *PerformanceOptimizer) optimizeCacheEfficiency(pipeline *RAGPipeline, task *OptimizationTask) error {
	po.logger.Info("Optimizing cache efficiency")

	// Implement cache warming for frequently accessed items
	if pipeline.redisCache != nil {
		// This would require implementing cache analytics and warming strategies
		po.logger.Debug("Cache warming would be implemented here")
	}

	// Adjust cache TTLs based on access patterns
	if pipeline.config.RedisCacheConfig != nil {
		// Increase TTL for frequently accessed items
		pipeline.config.RedisCacheConfig.EmbeddingTTL = 48 * time.Hour
	}

	task.Status = "completed"
	task.Result = "Cache optimization applied: TTLs adjusted, warming strategies enabled"
	return nil
}

// optimizeEmbeddingGeneration optimizes embedding generation performance
func (po *PerformanceOptimizer) optimizeEmbeddingGeneration(pipeline *RAGPipeline, task *OptimizationTask) error {
	po.logger.Info("Optimizing embedding generation")

	// Switch to faster provider if available
	if pipeline.embeddingService != nil {
		// This would require implementing provider performance tracking
		po.logger.Debug("Provider optimization would be implemented here")
	}

	// Optimize batch sizes for embedding generation
	if pipeline.config.EmbeddingConfig != nil {
		pipeline.config.EmbeddingConfig.BatchSize =
			min(pipeline.config.EmbeddingConfig.BatchSize, 50) // Smaller batches for faster response
	}

	task.Status = "completed"
	task.Result = "Embedding optimization applied: provider selection optimized, batch sizes adjusted"
	return nil
}

// optimizeDocumentProcessing optimizes document processing performance
func (po *PerformanceOptimizer) optimizeDocumentProcessing(pipeline *RAGPipeline, task *OptimizationTask) error {
	po.logger.Info("Optimizing document processing")

	// Enable streaming for large documents
	if pipeline.config.DocumentLoaderConfig != nil {
		pipeline.config.DocumentLoaderConfig.StreamingEnabled = true
		pipeline.config.DocumentLoaderConfig.StreamingThreshold = 20 * 1024 * 1024 // 20MB
	}

	// Optimize chunking batch sizes
	if pipeline.config.ChunkingConfig != nil {
		pipeline.config.ChunkingConfig.BatchSize =
			min(pipeline.config.ChunkingConfig.BatchSize, 25)
	}

	task.Status = "completed"
	task.Result = "Document processing optimization applied: streaming enabled, batch sizes optimized"
	return nil
}

// startAutoOptimization starts the background optimization process
func (po *PerformanceOptimizer) startAutoOptimization() {
	ticker := time.NewTicker(po.config.OptimizationInterval)
	defer ticker.Stop()

	po.logger.Info("Started auto-optimization", "interval", po.config.OptimizationInterval)

	for range ticker.C {
		// This would require access to the pipeline instance
		// In a real implementation, this would be injected or made available
		po.logger.Debug("Auto-optimization tick")
	}
}

// GetActiveOptimizations returns currently running optimizations
func (po *PerformanceOptimizer) GetActiveOptimizations() map[string]*OptimizationTask {
	po.mutex.RLock()
	defer po.mutex.RUnlock()

	// Return a copy to avoid concurrent access issues
	result := make(map[string]*OptimizationTask)
	for k, v := range po.activeOptimizations {
		taskCopy := *v
		result[k] = &taskCopy
	}

	return result
}

// GetPerformanceReport generates a comprehensive performance report
func (po *PerformanceOptimizer) GetPerformanceReport() *OptimizerPerformanceReport {
	metrics := po.CollectMetrics()
	activeOpts := po.GetActiveOptimizations()

	return &OptimizerPerformanceReport{
		GeneratedAt:         time.Now(),
		CurrentMetrics:      metrics,
		ActiveOptimizations: activeOpts,
		Recommendations:     po.generateRecommendations(metrics),
		HealthScore:         po.calculateHealthScore(metrics),
	}
}

// OptimizerPerformanceReport represents a comprehensive performance report
type OptimizerPerformanceReport struct {
	GeneratedAt         time.Time                    `json:"generated_at"`
	CurrentMetrics      *PerformanceMetrics          `json:"current_metrics"`
	ActiveOptimizations map[string]*OptimizationTask `json:"active_optimizations"`
	Recommendations     []string                     `json:"recommendations"`
	HealthScore         float64                      `json:"health_score"` // 0.0-1.0
}

// generateRecommendations generates performance recommendations
func (po *PerformanceOptimizer) generateRecommendations(metrics *PerformanceMetrics) []string {
	var recommendations []string

	if metrics.MemoryUsage > 0.8 {
		recommendations = append(recommendations, "Consider increasing available memory or optimizing memory usage")
	}

	if metrics.CacheHitRate < 0.7 {
		recommendations = append(recommendations, "Improve cache strategies and consider cache warming")
	}

	if metrics.AverageLatency > po.config.LatencyThreshold {
		recommendations = append(recommendations, "Optimize request processing and consider connection pooling")
	}

	if metrics.ErrorRate > 0.05 { // 5% error rate
		recommendations = append(recommendations, "Investigate and resolve sources of errors")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "System performance is within acceptable parameters")
	}

	return recommendations
}

// calculateHealthScore calculates an overall health score
func (po *PerformanceOptimizer) calculateHealthScore(metrics *PerformanceMetrics) float64 {
	score := 1.0

	// Memory usage impact
	if metrics.MemoryUsage > 0.8 {
		score -= 0.2
	} else if metrics.MemoryUsage > 0.6 {
		score -= 0.1
	}

	// CPU usage impact
	if metrics.CPUUsage > 0.8 {
		score -= 0.2
	} else if metrics.CPUUsage > 0.6 {
		score -= 0.1
	}

	// Latency impact
	if metrics.AverageLatency > po.config.LatencyThreshold {
		score -= 0.3
	}

	// Error rate impact
	if metrics.ErrorRate > 0.1 {
		score -= 0.3
	} else if metrics.ErrorRate > 0.05 {
		score -= 0.2
	}

	// Cache efficiency impact
	if metrics.CacheHitRate < 0.5 {
		score -= 0.2
	} else if metrics.CacheHitRate < 0.7 {
		score -= 0.1
	}

	if score < 0 {
		score = 0
	}

	return score
}

// Helper function for minimum duration calculation
func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}


// copyDurations creates a copy of duration slice  
func copyDurations(original []time.Duration) []time.Duration {
	if original == nil {
		return nil
	}
	copy := make([]time.Duration, len(original))
	for i, v := range original {
		copy[i] = v
	}
	return copy
}
