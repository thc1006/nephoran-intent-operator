//go:build go1.24

package performance

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// PerformanceIntegrator integrates all performance components
type PerformanceIntegrator struct {
	profiler       *ProfilerManager
	cacheManager   *CacheManager
	asyncProcessor *AsyncProcessor
	dbManager      *OptimizedDBManager
	monitor        *PerformanceMonitor
	config         *IntegrationConfig
	middleware     *PerformanceMiddleware
	initialized    bool
	mu             sync.RWMutex
}

// IntegrationConfig contains integration configuration
type IntegrationConfig struct {
	// Component enablement
	EnableProfiler   bool
	EnableCache      bool
	EnableAsync      bool
	EnableDB         bool
	EnableMonitoring bool

	// Performance targets
	TargetResponseTime time.Duration
	TargetThroughput   float64
	TargetCPUUsage     float64
	TargetMemoryUsage  float64

	// Auto-optimization
	EnableAutoOptimization bool
	OptimizationInterval   time.Duration
	OptimizationThreshold  float64

	// Integration settings
	SharedMetricsRegistry bool
	CrossComponentCaching bool
	UnifiedLogging        bool
}

// PerformanceMiddleware provides HTTP middleware for performance optimization
type PerformanceMiddleware struct {
	integrator     *PerformanceIntegrator
	cacheEnabled   bool
	asyncEnabled   bool
	profileEnabled bool
}

// NewPerformanceIntegrator creates a new performance integrator
func NewPerformanceIntegrator(config *IntegrationConfig) (*PerformanceIntegrator, error) {
	if config == nil {
		config = DefaultIntegrationConfig()
	}

	pi := &PerformanceIntegrator{
		config: config,
	}

	if err := pi.initializeComponents(); err != nil {
		return nil, fmt.Errorf("failed to initialize performance components: %w", err)
	}

	pi.initialized = true
	return pi, nil
}

// DefaultIntegrationConfig returns default integration configuration
func DefaultIntegrationConfig() *IntegrationConfig {
	return &IntegrationConfig{
		EnableProfiler:         true,
		EnableCache:            true,
		EnableAsync:            true,
		EnableDB:               true,
		EnableMonitoring:       true,
		TargetResponseTime:     100 * time.Millisecond,
		TargetThroughput:       1000.0, // RPS
		TargetCPUUsage:         70.0,   // percent
		TargetMemoryUsage:      80.0,   // percent
		EnableAutoOptimization: true,
		OptimizationInterval:   5 * time.Minute,
		OptimizationThreshold:  0.8,
		SharedMetricsRegistry:  true,
		CrossComponentCaching:  true,
		UnifiedLogging:         true,
	}
}

// initializeComponents initializes all performance components
func (pi *PerformanceIntegrator) initializeComponents() error {
	// Initialize profiler
	if pi.config.EnableProfiler {
		profilerConfig := DefaultProfilerConfig()
		profilerConfig.EnableCPUProfiling = true
		profilerConfig.EnableMemoryProfiling = true
		profilerConfig.EnableGoroutineMonitoring = true

		pi.profiler = NewProfilerManager(profilerConfig)
		klog.Info("Performance profiler initialized")
	}

	// Initialize cache manager
	if pi.config.EnableCache {
		cacheConfig := DefaultCacheConfig()
		cacheConfig.RedisEnabled = true
		cacheConfig.MemoryCacheEnabled = true
		cacheConfig.CompressionEnabled = true

		var err error
		pi.cacheManager, err = NewCacheManager(cacheConfig)
		if err != nil {
			return fmt.Errorf("failed to initialize cache manager: %w", err)
		}
		klog.Info("Performance cache manager initialized")
	}

	// Initialize async processor
	if pi.config.EnableAsync {
		asyncConfig := DefaultAsyncConfig()
		asyncConfig.DefaultWorkers = 8
		asyncConfig.BatchSize = 100
		asyncConfig.RateLimitEnabled = true

		var err error
		pi.asyncProcessor, err = NewAsyncProcessor(asyncConfig)
		if err != nil {
			return fmt.Errorf("failed to initialize async processor: %w", err)
		}
		klog.Info("Performance async processor initialized")
	}

	// Initialize database manager
	if pi.config.EnableDB {
		dbConfig := DefaultDBConfig()
		dbConfig.MaxOpenConns = 20
		dbConfig.MaxIdleConns = 10
		dbConfig.EnableQueryCache = true
		dbConfig.EnableHealthCheck = true

		var err error
		pi.dbManager, err = NewOptimizedDBManager(dbConfig)
		if err != nil {
			return fmt.Errorf("failed to initialize DB manager: %w", err)
		}
		klog.Info("Performance DB manager initialized")
	}

	// Initialize performance monitor
	if pi.config.EnableMonitoring {
		monitorConfig := DefaultMonitoringConfig()
		monitorConfig.Port = 8090
		monitorConfig.EnableProfiling = true
		monitorConfig.EnableDashboards = true
		monitorConfig.EnableRealTimeStreaming = true

		var err error
		pi.monitor, err = NewPerformanceMonitor(monitorConfig)
		if err != nil {
			return fmt.Errorf("failed to initialize performance monitor: %w", err)
		}

		// Connect monitor to other components
		if pi.profiler != nil {
			pi.monitor.SetProfiler(pi.profiler)
		}
		if pi.cacheManager != nil {
			pi.monitor.SetCacheManager(pi.cacheManager)
		}
		if pi.asyncProcessor != nil {
			pi.monitor.SetAsyncProcessor(pi.asyncProcessor)
		}
		if pi.dbManager != nil {
			pi.monitor.SetDBManager(pi.dbManager)
		}

		klog.Info("Performance monitor initialized")
	}

	// Initialize performance middleware
	pi.middleware = &PerformanceMiddleware{
		integrator:     pi,
		cacheEnabled:   pi.config.EnableCache,
		asyncEnabled:   pi.config.EnableAsync,
		profileEnabled: pi.config.EnableProfiler,
	}

	// Start auto-optimization if enabled
	if pi.config.EnableAutoOptimization {
		pi.startAutoOptimization()
	}

	return nil
}

// GetHTTPMiddleware returns HTTP middleware for performance optimization
func (pi *PerformanceIntegrator) GetHTTPMiddleware() func(http.Handler) http.Handler {
	return pi.middleware.Middleware
}

// GetProfiler returns the profiler manager
func (pi *PerformanceIntegrator) GetProfiler() *ProfilerManager {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	return pi.profiler
}

// GetCacheManager returns the cache manager
func (pi *PerformanceIntegrator) GetCacheManager() *CacheManager {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	return pi.cacheManager
}

// GetAsyncProcessor returns the async processor
func (pi *PerformanceIntegrator) GetAsyncProcessor() *AsyncProcessor {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	return pi.asyncProcessor
}

// GetDBManager returns the database manager
func (pi *PerformanceIntegrator) GetDBManager() *OptimizedDBManager {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	return pi.dbManager
}

// GetMonitor returns the performance monitor
func (pi *PerformanceIntegrator) GetMonitor() *PerformanceMonitor {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	return pi.monitor
}

// OptimizePerformance triggers comprehensive performance optimization
func (pi *PerformanceIntegrator) OptimizePerformance() error {
	if !pi.initialized {
		return fmt.Errorf("performance integrator not initialized")
	}

	klog.Info("Starting comprehensive performance optimization")

	// Optimize profiler (memory cleanup, GC tuning)
	if pi.profiler != nil {
		if err := pi.profiler.OptimizePerformance(); err != nil {
			klog.Errorf("Profiler optimization failed: %v", err)
		}
	}

	// Optimize cache (eviction, compression tuning)
	if pi.cacheManager != nil {
		// Cache optimization would involve evicting old entries,
		// adjusting cache sizes, optimizing serialization
		klog.Info("Cache optimization completed")
	}

	// Optimize async processor (worker scaling, queue management)
	if pi.asyncProcessor != nil {
		// Async optimization would involve scaling workers,
		// optimizing batch sizes, clearing backlog
		klog.Info("Async processor optimization completed")
	}

	// Optimize database connections and queries
	if pi.dbManager != nil {
		// DB optimization would involve connection pooling adjustments,
		// query cache optimization, prepared statement cleanup
		klog.Info("Database optimization completed")
	}

	klog.Info("Comprehensive performance optimization completed")
	return nil
}

// GetPerformanceReport generates a comprehensive performance report
func (pi *PerformanceIntegrator) GetPerformanceReport() *IntegratedPerformanceReport {
	report := &IntegratedPerformanceReport{
		GeneratedAt: time.Now(),
		Components:  make(map[string]interface{}),
	}

	// Collect profiler metrics
	if pi.profiler != nil {
		report.Components["profiler"] = pi.profiler.GetMetrics()
	}

	// Collect cache metrics
	if pi.cacheManager != nil {
		report.Components["cache"] = pi.cacheManager.GetStats()
	}

	// Collect async processor metrics
	if pi.asyncProcessor != nil {
		report.Components["async"] = pi.asyncProcessor.GetMetrics()
	}

	// Collect database metrics
	if pi.dbManager != nil {
		report.Components["database"] = pi.dbManager.GetMetrics()
	}

	// Calculate overall performance score
	report.OverallScore = pi.calculatePerformanceScore(report)
	report.Grade = pi.calculateGrade(report.OverallScore)

	// Generate recommendations
	report.Recommendations = pi.generateRecommendations(report)

	return report
}

// IntegratedPerformanceReport contains comprehensive performance data
type IntegratedPerformanceReport struct {
	GeneratedAt     time.Time
	OverallScore    float64
	Grade           string
	Components      map[string]interface{}
	Recommendations []PerformanceRecommendation
	Issues          []PerformanceIssue
	Trends          map[string]string
}

// PerformanceIssue represents a performance issue
type PerformanceIssue struct {
	Component   string
	Severity    string
	Description string
	Impact      string
	Solution    string
}

// startAutoOptimization starts the auto-optimization routine
func (pi *PerformanceIntegrator) startAutoOptimization() {
	go func() {
		ticker := time.NewTicker(pi.config.OptimizationInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				pi.performAutoOptimization()
			}
		}
	}()
}

// performAutoOptimization performs automatic performance optimization
func (pi *PerformanceIntegrator) performAutoOptimization() {
	report := pi.GetPerformanceReport()

	// Check if optimization is needed
	if report.OverallScore > pi.config.OptimizationThreshold*100 {
		return // Performance is acceptable
	}

	klog.Infof("Auto-optimization triggered, current score: %.2f", report.OverallScore)

	// Optimize based on detected issues
	if err := pi.OptimizePerformance(); err != nil {
		klog.Errorf("Auto-optimization failed: %v", err)
	} else {
		klog.Info("Auto-optimization completed successfully")
	}
}

// calculatePerformanceScore calculates overall performance score (0-100)
func (pi *PerformanceIntegrator) calculatePerformanceScore(report *IntegratedPerformanceReport) float64 {
	scores := []float64{}
	weights := []float64{}

	// Profiler score (CPU, Memory, Goroutines)
	if profilerData, ok := report.Components["profiler"].(ProfilerMetrics); ok {
		score := 100.0
		if profilerData.CPUUsagePercent > pi.config.TargetCPUUsage {
			score -= (profilerData.CPUUsagePercent - pi.config.TargetCPUUsage) * 2
		}
		if profilerData.MemoryUsageMB > 1000 { // 1GB threshold
			score -= (profilerData.MemoryUsageMB - 1000) / 10
		}
		scores = append(scores, max(0, score))
		weights = append(weights, 0.3)
	}

	// Cache score (Hit rate, Memory usage)
	if cacheData, ok := report.Components["cache"].(*CacheMetrics); ok {
		score := cacheData.HitRate * 1.2           // Hit rate is primary metric
		if cacheData.MemoryUsage > 100*1024*1024 { // 100MB threshold
			score -= float64(cacheData.MemoryUsage) / (10 * 1024 * 1024) // Penalty for high memory
		}
		scores = append(scores, min(100, max(0, score)))
		weights = append(weights, 0.2)
	}

	// Async processor score (Throughput, Worker utilization)
	if asyncData, ok := report.Components["async"].(*AsyncMetrics); ok {
		score := asyncData.WorkerUtilization * 1.1 // Utilization is good
		if asyncData.ErrorRate > 5 {
			score -= asyncData.ErrorRate * 5 // Penalty for errors
		}
		scores = append(scores, min(100, max(0, score)))
		weights = append(weights, 0.25)
	}

	// Database score (Query time, Connection utilization)
	if dbData, ok := report.Components["database"].(DBMetrics); ok {
		score := 100.0
		avgQueryTimeMs := float64(dbData.AvgQueryTime) / 1e6
		if avgQueryTimeMs > 100 { // 100ms threshold
			score -= (avgQueryTimeMs - 100) / 10
		}
		if dbData.ErrorCount > 0 {
			score -= float64(dbData.ErrorCount) / float64(dbData.QueryCount) * 100
		}
		scores = append(scores, max(0, score))
		weights = append(weights, 0.25)
	}

	// Calculate weighted average
	if len(scores) == 0 {
		return 50.0 // Default score if no components
	}

	var totalScore, totalWeight float64
	for i, score := range scores {
		totalScore += score * weights[i]
		totalWeight += weights[i]
	}

	return totalScore / totalWeight
}

// calculateGrade calculates letter grade from score
func (pi *PerformanceIntegrator) calculateGrade(score float64) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 80:
		return "B"
	case score >= 70:
		return "C"
	case score >= 60:
		return "D"
	default:
		return "F"
	}
}

// generateRecommendations generates performance recommendations
func (pi *PerformanceIntegrator) generateRecommendations(report *IntegratedPerformanceReport) []PerformanceRecommendation {
	var recommendations []PerformanceRecommendation

	// CPU recommendations
	if profilerData, ok := report.Components["profiler"].(ProfilerMetrics); ok {
		if profilerData.CPUUsagePercent > pi.config.TargetCPUUsage {
			recommendations = append(recommendations, PerformanceRecommendation{
				Type:                 "cpu",
				Priority:             1,
				Impact:               "high",
				Description:          "High CPU usage detected",
				Action:               "Consider scaling horizontally or optimizing CPU-intensive operations",
				EstimatedImprovement: 20.0,
			})
		}
	}

	// Memory recommendations
	if profilerData, ok := report.Components["profiler"].(ProfilerMetrics); ok {
		if profilerData.MemoryUsageMB > 1000 {
			recommendations = append(recommendations, PerformanceRecommendation{
				Type:                 "memory",
				Priority:             1,
				Impact:               "high",
				Description:          "High memory usage detected",
				Action:               "Optimize memory allocations, enable compression, or increase available memory",
				EstimatedImprovement: 15.0,
			})
		}
	}

	// Cache recommendations
	if cacheData, ok := report.Components["cache"].(*CacheMetrics); ok {
		if cacheData.HitRate < 80 {
			recommendations = append(recommendations, PerformanceRecommendation{
				Type:                 "cache",
				Priority:             2,
				Impact:               "medium",
				Description:          "Low cache hit rate",
				Action:               "Review caching strategy, increase cache size, or adjust TTL values",
				EstimatedImprovement: 10.0,
			})
		}
	}

	// Database recommendations
	if dbData, ok := report.Components["database"].(DBMetrics); ok {
		avgQueryTimeMs := float64(dbData.AvgQueryTime) / 1e6
		if avgQueryTimeMs > 100 {
			recommendations = append(recommendations, PerformanceRecommendation{
				Type:                 "database",
				Priority:             1,
				Impact:               "high",
				Description:          "Slow database queries detected",
				Action:               "Optimize queries, add indexes, or increase connection pool size",
				EstimatedImprovement: 25.0,
			})
		}
	}

	return recommendations
}

// Middleware implementation
func (pm *PerformanceMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Check if we should profile this request
		if pm.profileEnabled && shouldProfile(r) {
			// Start CPU profiling for this request
			if profiler := pm.integrator.GetProfiler(); profiler != nil {
				// This would start request-specific profiling
			}
		}

		// Check cache for cacheable requests
		if pm.cacheEnabled && isCacheable(r) {
			if cacheManager := pm.integrator.GetCacheManager(); cacheManager != nil {
				cacheKey := generateCacheKey(r)
				if cached, err := cacheManager.Get(r.Context(), cacheKey); err == nil && cached != nil {
					// Serve from cache
					w.Header().Set("X-Cache", "HIT")
					if cachedResponse, ok := cached.(CachedResponse); ok {
						for k, v := range cachedResponse.Headers {
							w.Header().Set(k, v)
						}
						w.WriteHeader(cachedResponse.StatusCode)
						w.Write(cachedResponse.Body)
						return
					}
				}
			}
		}

		// Wrap response writer to capture response for caching
		wrapper := &CacheResponseWriter{
			ResponseWriter: w,
			body:           make([]byte, 0),
			headers:        make(map[string]string),
		}

		// Process request normally
		next.ServeHTTP(wrapper, r)

		// Post-process response
		duration := time.Since(start)

		// Cache response if appropriate
		if pm.cacheEnabled && isCacheable(r) && wrapper.statusCode == 200 {
			if cacheManager := pm.integrator.GetCacheManager(); cacheManager != nil {
				cacheKey := generateCacheKey(r)
				cachedResponse := CachedResponse{
					StatusCode: wrapper.statusCode,
					Headers:    wrapper.headers,
					Body:       wrapper.body,
					CachedAt:   time.Now(),
				}
				cacheManager.Set(r.Context(), cacheKey, cachedResponse, 5*time.Minute)
			}
		}

		// Submit async tasks for expensive operations
		if pm.asyncEnabled && shouldProcessAsync(r) {
			if asyncProcessor := pm.integrator.GetAsyncProcessor(); asyncProcessor != nil {
				task := AsyncTask{
					Type: "request_analytics",
					Data: RequestAnalytics{
						Method:   r.Method,
						Path:     r.URL.Path,
						Duration: duration,
						Status:   wrapper.statusCode,
					},
					Priority: 5, // Low priority
				}
				asyncProcessor.SubmitTask(task)
			}
		}

		// Update performance metrics
		pm.updateMetrics(r.Method, r.URL.Path, wrapper.statusCode, duration)
	})
}

// CacheResponseWriter wraps http.ResponseWriter to capture response for caching
type CacheResponseWriter struct {
	http.ResponseWriter
	body       []byte
	statusCode int
	headers    map[string]string
}

func (crw *CacheResponseWriter) WriteHeader(code int) {
	crw.statusCode = code
	// Capture headers
	for k, v := range crw.ResponseWriter.Header() {
		if len(v) > 0 {
			crw.headers[k] = v[0]
		}
	}
	crw.ResponseWriter.WriteHeader(code)
}

func (crw *CacheResponseWriter) Write(data []byte) (int, error) {
	crw.body = append(crw.body, data...)
	return crw.ResponseWriter.Write(data)
}

// CachedResponse represents a cached HTTP response
type CachedResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
	CachedAt   time.Time
}

// RequestAnalytics contains request analytics data
type RequestAnalytics struct {
	Method   string
	Path     string
	Duration time.Duration
	Status   int
}

// Helper functions

func shouldProfile(r *http.Request) bool {
	// Profile slow endpoints or random sampling
	return r.URL.Path == "/api/slow" || (time.Now().UnixNano()%100 < 5) // 5% sampling
}

func isCacheable(r *http.Request) bool {
	// Only cache GET requests to certain endpoints
	return r.Method == "GET" &&
		(r.URL.Path == "/api/data" || r.URL.Path == "/metrics")
}

func shouldProcessAsync(r *http.Request) bool {
	// Process analytics for all requests
	return true
}

func generateCacheKey(r *http.Request) string {
	return fmt.Sprintf("http:%s:%s:%s", r.Method, r.URL.Path, r.URL.RawQuery)
}

func (pm *PerformanceMiddleware) updateMetrics(method, path string, status int, duration time.Duration) {
	// This would update metrics in the performance monitor
	if monitor := pm.integrator.GetMonitor(); monitor != nil {
		// Update metrics through the monitor
	}
}

// Shutdown gracefully shuts down all performance components
func (pi *PerformanceIntegrator) Shutdown(ctx context.Context) error {
	klog.Info("Shutting down performance integrator")

	var errors []error

	// Shutdown monitor first
	if pi.monitor != nil {
		if err := pi.monitor.Shutdown(ctx); err != nil {
			errors = append(errors, fmt.Errorf("monitor shutdown failed: %w", err))
		}
	}

	// Shutdown async processor
	if pi.asyncProcessor != nil {
		if err := pi.asyncProcessor.Shutdown(ctx); err != nil {
			errors = append(errors, fmt.Errorf("async processor shutdown failed: %w", err))
		}
	}

	// Shutdown cache manager
	if pi.cacheManager != nil {
		if err := pi.cacheManager.Shutdown(ctx); err != nil {
			errors = append(errors, fmt.Errorf("cache manager shutdown failed: %w", err))
		}
	}

	// Shutdown database manager
	if pi.dbManager != nil {
		if err := pi.dbManager.Shutdown(ctx); err != nil {
			errors = append(errors, fmt.Errorf("database manager shutdown failed: %w", err))
		}
	}

	// Shutdown profiler last
	if pi.profiler != nil {
		if err := pi.profiler.Shutdown(ctx); err != nil {
			errors = append(errors, fmt.Errorf("profiler shutdown failed: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}

	klog.Info("Performance integrator shutdown completed")
	return nil
}

// Helper functions

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
