//go:build !disable_rag && !test

package rag

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// PerformanceIntegration provides a unified interface for all performance optimizations.

type PerformanceIntegration struct {

	// Core services.

	optimizedRAG *OptimizedRAGService

	weaviatePool *WeaviateConnectionPool

	memoryCache *MemoryCache

	redisCache *RedisCache

	errorHandler *ErrorHandler

	// Monitoring and metrics.

	prometheusMetrics *PrometheusMetrics

	metricsCollector *MetricsCollector

	monitor *RAGMonitor

	// Configuration.

	config *PerformanceConfig

	logger *slog.Logger

	// State management.

	isHealthy bool

	startTime time.Time
}

// PerformanceConfig aggregates all performance-related configurations.

type PerformanceConfig struct {

	// Service configurations.

	OptimizedRAGConfig *OptimizedRAGConfig `json:"optimized_rag_config"`

	WeaviatePoolConfig *PoolConfig `json:"weaviate_pool_config"`

	MemoryCacheConfig *MemoryCacheConfig `json:"memory_cache_config"`

	RedisCacheConfig *RedisCacheConfig `json:"redis_cache_config"`

	ErrorHandlingConfig *ErrorHandlingConfig `json:"error_handling_config"`

	MonitoringConfig *MonitoringConfig `json:"monitoring_config"`

	// Integration settings.

	EnableAutoOptimization bool `json:"enable_auto_optimization"`

	OptimizationInterval time.Duration `json:"optimization_interval"`

	PerformanceTargets *PerformanceTargets `json:"performance_targets"`

	// Feature flags.

	EnableDistributedTracing bool `json:"enable_distributed_tracing"`

	EnableChaosEngineering bool `json:"enable_chaos_engineering"`

	EnableLoadBalancing bool `json:"enable_load_balancing"`
}

// PerformanceTargets defines SLA targets for the system.

type PerformanceTargets struct {
	MaxQueryLatencyMs int `json:"max_query_latency_ms"`

	MinCacheHitRate float64 `json:"min_cache_hit_rate"`

	MaxErrorRate float64 `json:"max_error_rate"`

	MinThroughputQPS int `json:"min_throughput_qps"`

	MaxResourceUtilization float64 `json:"max_resource_utilization"`
}

// PerformanceReport provides comprehensive performance analytics.

type PerformanceReport struct {
	GeneratedAt time.Time `json:"generated_at"`

	SystemUptime time.Duration `json:"system_uptime"`

	OverallHealth string `json:"overall_health"`

	// Performance metrics.

	QueryPerformance *QueryPerformanceMetrics `json:"query_performance"`

	CachePerformance *CachePerformanceMetrics `json:"cache_performance"`

	ConnectionPerformance *ConnectionPerformanceMetrics `json:"connection_performance"`

	ErrorAnalysis *ErrorAnalysisMetrics `json:"error_analysis"`

	// Resource utilization.

	ResourceUtilization *ResourceUtilizationMetrics `json:"resource_utilization"`

	// Optimization recommendations.

	Recommendations []PerformanceRecommendation `json:"recommendations"`

	// SLA compliance.

	SLACompliance *SLAComplianceReport `json:"sla_compliance"`
}

// Individual metric structures.

type QueryPerformanceMetrics struct {
	TotalQueries int64 `json:"total_queries"`

	AverageLatency time.Duration `json:"average_latency"`

	P95Latency time.Duration `json:"p95_latency"`

	P99Latency time.Duration `json:"p99_latency"`

	ThroughputQPS float64 `json:"throughput_qps"`

	SuccessRate float64 `json:"success_rate"`

	QualityScore float64 `json:"quality_score"`
}

// CachePerformanceMetrics represents a cacheperformancemetrics.

type CachePerformanceMetrics struct {
	MemoryCacheHitRate float64 `json:"memory_cache_hit_rate"`

	RedisCacheHitRate float64 `json:"redis_cache_hit_rate"`

	OverallHitRate float64 `json:"overall_hit_rate"`

	CacheEfficiency float64 `json:"cache_efficiency"`

	EvictionRate float64 `json:"eviction_rate"`

	CacheLatency time.Duration `json:"cache_latency"`
}

// ConnectionPerformanceMetrics represents a connectionperformancemetrics.

type ConnectionPerformanceMetrics struct {
	PoolUtilization float64 `json:"pool_utilization"`

	AvgConnectionTime time.Duration `json:"avg_connection_time"`

	ConnectionFailures int64 `json:"connection_failures"`

	ConnectionRecoveries int64 `json:"connection_recoveries"`

	HealthyConnections int `json:"healthy_connections"`
}

// ErrorAnalysisMetrics represents a erroranalysismetrics.

type ErrorAnalysisMetrics struct {
	TotalErrors int64 `json:"total_errors"`

	ErrorRate float64 `json:"error_rate"`

	ErrorsByType map[string]int64 `json:"errors_by_type"`

	ErrorsByComponent map[string]int64 `json:"errors_by_component"`

	CircuitBreakerTrips int64 `json:"circuit_breaker_trips"`

	RecoveryEvents int64 `json:"recovery_events"`
}

// ResourceUtilizationMetrics represents a resourceutilizationmetrics.

type ResourceUtilizationMetrics struct {
	CPUUsage float64 `json:"cpu_usage"`

	MemoryUsage float64 `json:"memory_usage"`

	DiskUsage float64 `json:"disk_usage"`

	NetworkUtilization float64 `json:"network_utilization"`

	GoroutineCount int `json:"goroutine_count"`

	HeapSize int64 `json:"heap_size"`
}

// PerformanceRecommendation represents a performancerecommendation.

type PerformanceRecommendation struct {
	Category string `json:"category"`

	Priority string `json:"priority"`

	Title string `json:"title"`

	Description string `json:"description"`

	Impact string `json:"impact"`

	Effort string `json:"effort"`

	Actions []string `json:"actions"`

	Metadata map[string]interface{} `json:"metadata"`
}

// SLAComplianceReport represents a slacompliancereport.

type SLAComplianceReport struct {
	OverallCompliance float64 `json:"overall_compliance"`

	TargetCompliance map[string]bool `json:"target_compliance"`

	Violations []SLAViolation `json:"violations"`

	ComplianceTrends map[string][]CompliancePoint `json:"compliance_trends"`
}

// SLAViolation represents a slaviolation.

type SLAViolation struct {
	Target string `json:"target"`

	ActualValue float64 `json:"actual_value"`

	TargetValue float64 `json:"target_value"`

	Severity string `json:"severity"`

	Timestamp time.Time `json:"timestamp"`

	Duration time.Duration `json:"duration"`
}

// CompliancePoint represents a compliancepoint.

type CompliancePoint struct {
	Timestamp time.Time `json:"timestamp"`

	Value float64 `json:"value"`

	Compliant bool `json:"compliant"`
}

// NewPerformanceIntegration creates a comprehensive performance-optimized RAG system.

func NewPerformanceIntegration(config *PerformanceConfig, llmClient shared.ClientInterface) (*PerformanceIntegration, error) {

	if config == nil {

		config = getDefaultPerformanceConfig()

	}

	logger := slog.Default().With("component", "performance-integration")

	// Initialize Weaviate connection pool.

	weaviatePool := NewWeaviateConnectionPool(config.WeaviatePoolConfig)

	if err := weaviatePool.Start(); err != nil {

		return nil, fmt.Errorf("failed to start Weaviate pool: %w", err)

	}

	// Initialize memory cache.

	memoryCache := NewMemoryCache(config.MemoryCacheConfig)

	// Initialize Redis cache.

	redisCache, err := NewRedisCache(config.RedisCacheConfig)

	if err != nil {

		logger.Warn("Redis cache initialization failed, using memory-only caching", "error", err)

		redisCache = nil

	}

	// Initialize error handler.

	errorHandler := NewErrorHandler(config.ErrorHandlingConfig)

	// Initialize monitoring.

	monitor := NewRAGMonitor(config.MonitoringConfig)

	prometheusMetrics := NewPrometheusMetrics()

	metricsCollector := NewMetricsCollector(memoryCache, redisCache, weaviatePool)

	// Initialize optimized RAG service.

	optimizedRAG, err := NewOptimizedRAGService(weaviatePool, llmClient, config.OptimizedRAGConfig)

	if err != nil {

		return nil, fmt.Errorf("failed to create optimized RAG service: %w", err)

	}

	integration := &PerformanceIntegration{

		optimizedRAG: optimizedRAG,

		weaviatePool: weaviatePool,

		memoryCache: memoryCache,

		redisCache: redisCache,

		errorHandler: errorHandler,

		prometheusMetrics: prometheusMetrics,

		metricsCollector: metricsCollector,

		monitor: monitor,

		config: config,

		logger: logger,

		isHealthy: true,

		startTime: time.Now(),
	}

	// Register health checkers.

	ragHealthChecker := &RAGServiceHealthChecker{}

	if ragHealthChecker != nil {

		monitor.RegisterHealthChecker(ragHealthChecker)

	}

	if redisCache != nil {

		redisHealthChecker := &RedisHealthChecker{redisCache: redisCache}

		monitor.RegisterHealthChecker(redisHealthChecker)

	}

	// Start background optimization if enabled.

	if config.EnableAutoOptimization {

		go integration.startAutoOptimization()

	}

	// Start metrics collection.

	metricsCollector.Start()

	logger.Info("Performance integration initialized successfully",

		"weaviate_pool_enabled", true,

		"memory_cache_enabled", true,

		"redis_cache_enabled", redisCache != nil,

		"auto_optimization_enabled", config.EnableAutoOptimization,
	)

	return integration, nil

}

// ProcessQuery provides the main interface for processing queries with full optimization.

func (pi *PerformanceIntegration) ProcessQuery(ctx context.Context, request *RAGRequest) (*RAGResponse, error) {

	startTime := time.Now()

	// Add performance monitoring context.

	ctx = pi.addPerformanceContext(ctx, request)

	// Pre-process request for optimization.

	optimizedRequest := pi.preprocessRequest(request)

	// Execute query with full optimization stack.

	response, err := pi.optimizedRAG.ProcessQuery(ctx, optimizedRequest)

	// Record performance metrics.

	duration := time.Since(startTime)

	pi.recordQueryMetrics(request, response, err, duration)

	// Post-process response for quality enhancement.

	if response != nil && err == nil {

		response = pi.postprocessResponse(response, request)

	}

	return response, err

}

// Performance analysis and reporting.

// GeneratePerformanceReport performs generateperformancereport operation.

func (pi *PerformanceIntegration) GeneratePerformanceReport() *PerformanceReport {

	report := &PerformanceReport{

		GeneratedAt: time.Now(),

		SystemUptime: time.Since(pi.startTime),

		OverallHealth: pi.getOverallHealth(),
	}

	// Collect performance metrics.

	report.QueryPerformance = pi.collectQueryPerformanceMetrics()

	report.CachePerformance = pi.collectCachePerformanceMetrics()

	report.ConnectionPerformance = pi.collectConnectionPerformanceMetrics()

	report.ErrorAnalysis = pi.collectErrorAnalysisMetrics()

	report.ResourceUtilization = pi.collectResourceUtilizationMetrics()

	// Generate recommendations.

	report.Recommendations = pi.generatePerformanceRecommendations(report)

	// Check SLA compliance.

	report.SLACompliance = pi.checkSLACompliance(report)

	return report

}

func (pi *PerformanceIntegration) collectQueryPerformanceMetrics() *QueryPerformanceMetrics {

	ragMetrics := pi.optimizedRAG.GetOptimizedMetrics()

	return &QueryPerformanceMetrics{

		TotalQueries: ragMetrics.TotalQueries,

		AverageLatency: ragMetrics.AverageLatency,

		P95Latency: ragMetrics.P95Latency,

		P99Latency: ragMetrics.P99Latency,

		ThroughputQPS: pi.calculateThroughput(ragMetrics),

		SuccessRate: pi.calculateSuccessRate(ragMetrics),

		QualityScore: ragMetrics.AverageResponseQuality,
	}

}

func (pi *PerformanceIntegration) collectCachePerformanceMetrics() *CachePerformanceMetrics {

	memStats := pi.memoryCache.GetStats()

	var redisHitRate float64 = 0

	if pi.redisCache != nil {

		redisStats := pi.redisCache.GetMetrics()

		if redisStats.TotalRequests > 0 {

			redisHitRate = float64(redisStats.Hits) / float64(redisStats.TotalRequests)

		}

	}

	memHitRate := memStats.HitRate

	overallHitRate := (memHitRate + redisHitRate) / 2

	return &CachePerformanceMetrics{

		MemoryCacheHitRate: memHitRate,

		RedisCacheHitRate: redisHitRate,

		OverallHitRate: overallHitRate,

		CacheEfficiency: pi.calculateCacheEfficiency(memStats, redisHitRate),

		EvictionRate: pi.calculateEvictionRate(memStats),

		CacheLatency: memStats.AverageGetTime,
	}

}

func (pi *PerformanceIntegration) collectConnectionPerformanceMetrics() *ConnectionPerformanceMetrics {

	poolMetrics := pi.weaviatePool.GetMetrics()

	utilization := float64(0)

	if poolMetrics.TotalConnections > 0 {

		utilization = float64(poolMetrics.ActiveConnections) / float64(poolMetrics.TotalConnections)

	}

	return &ConnectionPerformanceMetrics{

		PoolUtilization: utilization,

		AvgConnectionTime: poolMetrics.AverageLatency,

		ConnectionFailures: poolMetrics.ConnectionFailures,

		ConnectionRecoveries: poolMetrics.ConnectionsCreated - poolMetrics.ConnectionsDestroyed,

		HealthyConnections: int(poolMetrics.ActiveConnections),
	}

}

func (pi *PerformanceIntegration) collectErrorAnalysisMetrics() *ErrorAnalysisMetrics {

	ragMetrics := pi.optimizedRAG.GetOptimizedMetrics()

	totalRequests := ragMetrics.TotalQueries

	errorRate := float64(0)

	if totalRequests > 0 {

		errorRate = float64(ragMetrics.FailedQueries) / float64(totalRequests)

	}

	return &ErrorAnalysisMetrics{

		TotalErrors: ragMetrics.FailedQueries,

		ErrorRate: errorRate,

		ErrorsByType: make(map[string]int64), // Would be populated from error handler

		ErrorsByComponent: make(map[string]int64), // Would be populated from error handler

		CircuitBreakerTrips: ragMetrics.CircuitBreakerTrips,

		RecoveryEvents: ragMetrics.RecoveryEvents,
	}

}

func (pi *PerformanceIntegration) collectResourceUtilizationMetrics() *ResourceUtilizationMetrics {

	// This would collect actual system metrics.

	return &ResourceUtilizationMetrics{

		CPUUsage: 50.0, // Placeholder

		MemoryUsage: 60.0, // Placeholder

		DiskUsage: 70.0, // Placeholder

		NetworkUtilization: 30.0, // Placeholder

		GoroutineCount: 100, // Placeholder

		HeapSize: 1024 * 1024 * 128, // Placeholder

	}

}

// Performance optimization methods.

func (pi *PerformanceIntegration) startAutoOptimization() {

	ticker := time.NewTicker(pi.config.OptimizationInterval)

	defer ticker.Stop()

	for range ticker.C {

		pi.performAutoOptimization()

	}

}

func (pi *PerformanceIntegration) performAutoOptimization() {

	pi.logger.Info("Starting automatic performance optimization")

	report := pi.GeneratePerformanceReport()

	// Optimize cache settings.

	pi.optimizeCacheSettings(report.CachePerformance)

	// Optimize connection pool.

	pi.optimizeConnectionPool(report.ConnectionPerformance)

	// Adjust error handling.

	pi.optimizeErrorHandling(report.ErrorAnalysis)

	pi.logger.Info("Automatic performance optimization completed")

}

func (pi *PerformanceIntegration) optimizeCacheSettings(cacheMetrics *CachePerformanceMetrics) {

	// Optimize memory cache size.

	if cacheMetrics.MemoryCacheHitRate < 0.5 {

		pi.logger.Info("Memory cache hit rate low, considering size increase",

			"current_hit_rate", cacheMetrics.MemoryCacheHitRate)

		// Logic to adjust cache sizes.

	}

	// Optimize TTL settings.

	if cacheMetrics.EvictionRate > 0.3 {

		pi.logger.Info("High eviction rate detected, considering TTL adjustment",

			"eviction_rate", cacheMetrics.EvictionRate)

		// Logic to adjust TTL settings.

	}

}

func (pi *PerformanceIntegration) optimizeConnectionPool(connMetrics *ConnectionPerformanceMetrics) {

	// Adjust pool size based on utilization.

	if connMetrics.PoolUtilization > 0.8 {

		pi.logger.Info("High pool utilization, considering pool expansion",

			"utilization", connMetrics.PoolUtilization)

		// Logic to expand connection pool.

	} else if connMetrics.PoolUtilization < 0.3 {

		pi.logger.Info("Low pool utilization, considering pool reduction",

			"utilization", connMetrics.PoolUtilization)

		// Logic to reduce connection pool.

	}

}

func (pi *PerformanceIntegration) optimizeErrorHandling(errorMetrics *ErrorAnalysisMetrics) {

	// Adjust circuit breaker thresholds.

	if errorMetrics.ErrorRate > 0.1 {

		pi.logger.Warn("High error rate detected, tightening circuit breaker",

			"error_rate", errorMetrics.ErrorRate)

		// Logic to adjust circuit breaker settings.

	}

}

// Recommendation generation.

func (pi *PerformanceIntegration) generatePerformanceRecommendations(report *PerformanceReport) []PerformanceRecommendation {

	var recommendations []PerformanceRecommendation

	// Cache performance recommendations.

	if report.CachePerformance.OverallHitRate < 0.6 {

		recommendations = append(recommendations, PerformanceRecommendation{

			Category: "caching",

			Priority: "high",

			Title: "Improve Cache Hit Rate",

			Description: "Cache hit rate is below optimal threshold. Consider increasing cache sizes or adjusting TTL settings.",

			Impact: "20-30% latency reduction",

			Effort: "low",

			Actions: []string{

				"Increase memory cache size",

				"Optimize cache TTL settings",

				"Implement intelligent prefetching",
			},

			Metadata: map[string]interface{}{

				"current_hit_rate": report.CachePerformance.OverallHitRate,

				"target_hit_rate": 0.8,
			},
		})

	}

	// Query performance recommendations.

	if report.QueryPerformance.P95Latency > time.Duration(pi.config.PerformanceTargets.MaxQueryLatencyMs)*time.Millisecond {

		recommendations = append(recommendations, PerformanceRecommendation{

			Category: "query_performance",

			Priority: "high",

			Title: "Reduce Query Latency",

			Description: "Query latency exceeds SLA targets. Consider query optimization and connection pool tuning.",

			Impact: "40-50% latency reduction",

			Effort: "medium",

			Actions: []string{

				"Enable query optimization",

				"Increase connection pool size",

				"Implement parallel processing",
			},

			Metadata: map[string]interface{}{

				"current_p95_latency": report.QueryPerformance.P95Latency.Milliseconds(),

				"target_latency": pi.config.PerformanceTargets.MaxQueryLatencyMs,
			},
		})

	}

	// Connection pool recommendations.

	if report.ConnectionPerformance.PoolUtilization > 0.9 {

		recommendations = append(recommendations, PerformanceRecommendation{

			Category: "connection_pool",

			Priority: "medium",

			Title: "Expand Connection Pool",

			Description: "Connection pool utilization is very high. Consider increasing pool size to prevent bottlenecks.",

			Impact: "15-25% throughput increase",

			Effort: "low",

			Actions: []string{

				"Increase max connections",

				"Optimize connection reuse",

				"Implement connection health checks",
			},

			Metadata: map[string]interface{}{

				"current_utilization": report.ConnectionPerformance.PoolUtilization,

				"recommended_increase": 0.3,
			},
		})

	}

	// Error handling recommendations.

	if report.ErrorAnalysis.ErrorRate > pi.config.PerformanceTargets.MaxErrorRate {

		recommendations = append(recommendations, PerformanceRecommendation{

			Category: "error_handling",

			Priority: "high",

			Title: "Improve Error Handling",

			Description: "Error rate exceeds acceptable thresholds. Enhance error handling and circuit breaker configuration.",

			Impact: "Improved reliability and user experience",

			Effort: "medium",

			Actions: []string{

				"Tighten circuit breaker thresholds",

				"Implement better retry policies",

				"Add fallback mechanisms",
			},

			Metadata: map[string]interface{}{

				"current_error_rate": report.ErrorAnalysis.ErrorRate,

				"target_error_rate": pi.config.PerformanceTargets.MaxErrorRate,
			},
		})

	}

	return recommendations

}

// SLA compliance checking.

func (pi *PerformanceIntegration) checkSLACompliance(report *PerformanceReport) *SLAComplianceReport {

	compliance := &SLAComplianceReport{

		TargetCompliance: make(map[string]bool),

		Violations: []SLAViolation{},

		ComplianceTrends: make(map[string][]CompliancePoint),
	}

	targets := pi.config.PerformanceTargets

	// Check latency SLA.

	latencyCompliant := report.QueryPerformance.P95Latency.Milliseconds() <= int64(targets.MaxQueryLatencyMs)

	compliance.TargetCompliance["latency"] = latencyCompliant

	if !latencyCompliant {

		compliance.Violations = append(compliance.Violations, SLAViolation{

			Target: "latency",

			ActualValue: float64(report.QueryPerformance.P95Latency.Milliseconds()),

			TargetValue: float64(targets.MaxQueryLatencyMs),

			Severity: "high",

			Timestamp: time.Now(),
		})

	}

	// Check cache hit rate SLA.

	cacheCompliant := report.CachePerformance.OverallHitRate >= targets.MinCacheHitRate

	compliance.TargetCompliance["cache_hit_rate"] = cacheCompliant

	if !cacheCompliant {

		compliance.Violations = append(compliance.Violations, SLAViolation{

			Target: "cache_hit_rate",

			ActualValue: report.CachePerformance.OverallHitRate,

			TargetValue: targets.MinCacheHitRate,

			Severity: "medium",

			Timestamp: time.Now(),
		})

	}

	// Check error rate SLA.

	errorCompliant := report.ErrorAnalysis.ErrorRate <= targets.MaxErrorRate

	compliance.TargetCompliance["error_rate"] = errorCompliant

	if !errorCompliant {

		compliance.Violations = append(compliance.Violations, SLAViolation{

			Target: "error_rate",

			ActualValue: report.ErrorAnalysis.ErrorRate,

			TargetValue: targets.MaxErrorRate,

			Severity: "critical",

			Timestamp: time.Now(),
		})

	}

	// Calculate overall compliance.

	totalTargets := len(compliance.TargetCompliance)

	compliantTargets := 0

	for _, compliant := range compliance.TargetCompliance {

		if compliant {

			compliantTargets++

		}

	}

	compliance.OverallCompliance = float64(compliantTargets) / float64(totalTargets)

	return compliance

}

// Helper methods.

func (pi *PerformanceIntegration) addPerformanceContext(ctx context.Context, request *RAGRequest) context.Context {

	// Add performance monitoring context.

	return context.WithValue(ctx, "performance_monitoring", true)

}

func (pi *PerformanceIntegration) preprocessRequest(request *RAGRequest) *RAGRequest {

	// Apply preprocessing optimizations.

	optimized := *request

	// Set optimal defaults based on configuration.

	if optimized.MaxResults == 0 {

		optimized.MaxResults = 10

	}

	return &optimized

}

func (pi *PerformanceIntegration) postprocessResponse(response *RAGResponse, request *RAGRequest) *RAGResponse {

	// Apply post-processing enhancements.

	enhanced := *response

	// Add performance metadata.

	if enhanced.Metadata == nil {

		enhanced.Metadata = make(map[string]interface{})

	}

	enhanced.Metadata["performance_optimized"] = true

	enhanced.Metadata["optimization_level"] = "full"

	return &enhanced

}

func (pi *PerformanceIntegration) recordQueryMetrics(request *RAGRequest, response *RAGResponse, err error, duration time.Duration) {

	status := "success"

	if err != nil {

		status = "error"

	}

	pi.prometheusMetrics.RecordQueryLatency(

		request.IntentType,

		"optimized",

		fmt.Sprintf("%v", response != nil && response.UsedCache),

		duration,
	)

	pi.prometheusMetrics.RecordQueryProcessed(request.IntentType, status)

	if response != nil {

		if quality, ok := response.Metadata["response_quality"].(float64); ok {

			pi.prometheusMetrics.RecordResponseQuality(request.IntentType, "optimized", quality)

		}

	}

}

func (pi *PerformanceIntegration) getOverallHealth() string {

	if !pi.isHealthy {

		return "unhealthy"

	}

	// Check component health.

	health := pi.optimizedRAG.GetHealth()

	if status, ok := health["status"].(string); ok && status != "healthy" {

		return "degraded"

	}

	return "healthy"

}

func (pi *PerformanceIntegration) calculateThroughput(metrics *OptimizedRAGMetrics) float64 {

	if metrics.TotalQueries == 0 {

		return 0

	}

	uptime := time.Since(pi.startTime)

	return float64(metrics.TotalQueries) / uptime.Seconds()

}

func (pi *PerformanceIntegration) calculateSuccessRate(metrics *OptimizedRAGMetrics) float64 {

	if metrics.TotalQueries == 0 {

		return 1.0

	}

	return float64(metrics.SuccessfulQueries) / float64(metrics.TotalQueries)

}

func (pi *PerformanceIntegration) calculateCacheEfficiency(memStats *MemoryCacheMetrics, redisHitRate float64) float64 {

	// Combined efficiency metric considering hit rates and latencies.

	memEfficiency := memStats.HitRate * 1.0 // Memory cache is faster

	redisEfficiency := redisHitRate * 0.8 // Redis cache is slower but still good

	return (memEfficiency + redisEfficiency) / 2

}

func (pi *PerformanceIntegration) calculateEvictionRate(memStats *MemoryCacheMetrics) float64 {

	if memStats.Sets == 0 {

		return 0

	}

	return float64(memStats.Evictions) / float64(memStats.Sets)

}

// Shutdown gracefully shuts down all components.

func (pi *PerformanceIntegration) Shutdown(ctx context.Context) error {

	pi.logger.Info("Shutting down performance integration")

	// Shutdown in reverse order of initialization.

	if pi.metricsCollector != nil {

		pi.metricsCollector.Stop()

	}

	if pi.monitor != nil {

		pi.monitor.Shutdown(ctx)

	}

	if pi.optimizedRAG != nil {

		pi.optimizedRAG.Shutdown(ctx)

	}

	if pi.weaviatePool != nil {

		pi.weaviatePool.Stop()

	}

	if pi.memoryCache != nil {

		pi.memoryCache.Close()

	}

	if pi.redisCache != nil {

		pi.redisCache.Close()

	}

	pi.logger.Info("Performance integration shutdown completed")

	return nil

}

// Configuration defaults.

func getDefaultPerformanceConfig() *PerformanceConfig {

	return &PerformanceConfig{

		OptimizedRAGConfig: getDefaultOptimizedRAGConfig(),

		WeaviatePoolConfig: DefaultPoolConfig(),

		MemoryCacheConfig: getDefaultMemoryCacheConfig(),

		RedisCacheConfig: getDefaultRedisCacheConfig(),

		ErrorHandlingConfig: getDefaultErrorHandlingConfig(),

		MonitoringConfig: getDefaultMonitoringConfig(),

		EnableAutoOptimization: true,

		OptimizationInterval: 15 * time.Minute,

		PerformanceTargets: &PerformanceTargets{

			MaxQueryLatencyMs: 2000, // 2 seconds

			MinCacheHitRate: 0.75, // 75%

			MaxErrorRate: 0.05, // 5%

			MinThroughputQPS: 10, // 10 queries per second

			MaxResourceUtilization: 0.8, // 80%

		},

		EnableDistributedTracing: false,

		EnableChaosEngineering: false,

		EnableLoadBalancing: false,
	}

}
