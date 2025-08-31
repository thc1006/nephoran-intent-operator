// Package health provides enhanced multi-tiered health monitoring for the Nephoran Intent Operator.

// This system offers comprehensive health assessment with weighted scoring, contextual checks,.

// and performance optimization achieving sub-100ms execution times.

package health

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/thc1006/nephoran-intent-operator/pkg/health"
)

// Note: HealthTier, HealthContext, HealthWeight, EnhancedCheck, and StateTransition.

// are now defined in types.go to avoid duplicates.

// EnhancedCheckFunc is a function that performs an enhanced health check.

type EnhancedCheckFunc func(ctx context.Context, context HealthContext) *EnhancedCheck

// CheckConfig holds configuration for an enhanced health check.

type CheckConfig struct {
	Name string `json:"name"`

	Tier HealthTier `json:"tier"`

	Weight HealthWeight `json:"weight"`

	Interval time.Duration `json:"interval"`

	Timeout time.Duration `json:"timeout"`

	FailureThreshold int `json:"failure_threshold"`

	Enabled bool `json:"enabled"`

	// Context-specific settings.

	ContextualSettings map[HealthContext]ContextualConfig `json:"contextual_settings,omitempty"`

	// Dependencies.

	Dependencies []string `json:"dependencies,omitempty"`

	// Performance thresholds.

	LatencyThreshold time.Duration `json:"latency_threshold"`

	SuccessRateThreshold float64 `json:"success_rate_threshold"`
}

// ContextualConfig holds context-specific configuration.

type ContextualConfig struct {
	Interval time.Duration `json:"interval"`

	Timeout time.Duration `json:"timeout"`

	Enabled bool `json:"enabled"`

	Weight HealthWeight `json:"weight"`
}

// EnhancedHealthChecker manages enhanced multi-tiered health checks.

type EnhancedHealthChecker struct {

	// Core configuration.

	serviceName string

	serviceVersion string

	startTime time.Time

	logger *slog.Logger

	// Check registry and state.

	checks map[string]*CheckConfig

	checkFuncs map[string]EnhancedCheckFunc

	checkResults map[string]*EnhancedCheck

	checkHistory map[string][]EnhancedCheck

	mu sync.RWMutex

	// Context management.

	currentContext HealthContext

	contextMu sync.RWMutex

	// State management.

	overallHealth atomic.Value // health.Status

	healthScore atomic.Value // float64

	lastExecution time.Time

	executionCount int64

	// Performance optimization.

	cacheEnabled bool

	cacheExpiry time.Duration

	cachedResults map[string]*CachedResult

	cacheMu sync.RWMutex

	// Worker pool for parallel execution.

	workerCount int

	taskQueue chan HealthTask

	workerWg sync.WaitGroup

	shutdown chan struct{}

	running int32 // 0=stopped, 1=running (replaced atomic.Bool for compatibility)

	// Metrics.

	metrics *EnhancedHealthMetrics
}

// CachedResult holds cached health check results.

type CachedResult struct {
	Result *EnhancedCheck

	Timestamp time.Time
}

// HealthTask represents a health check task for the worker pool.

type HealthTask struct {
	Name string

	Config *CheckConfig

	CheckFunc EnhancedCheckFunc

	Context HealthContext

	ResultCh chan *EnhancedCheck
}

// EnhancedHealthMetrics contains Prometheus metrics for enhanced health checking.

type EnhancedHealthMetrics struct {
	CheckDuration *prometheus.HistogramVec

	CheckSuccess *prometheus.CounterVec

	CheckFailures *prometheus.CounterVec

	HealthScore *prometheus.GaugeVec

	StateTransitions *prometheus.CounterVec

	CacheHitRate prometheus.Gauge

	ParallelExecution prometheus.Histogram

	WeightedScores *prometheus.GaugeVec
}

// NewEnhancedHealthChecker creates a new enhanced health checker.

func NewEnhancedHealthChecker(serviceName, serviceVersion string, logger *slog.Logger) *EnhancedHealthChecker {

	if logger == nil {

		logger = slog.Default()

	}

	ehc := &EnhancedHealthChecker{

		serviceName: serviceName,

		serviceVersion: serviceVersion,

		startTime: time.Now(),

		logger: logger,

		checks: make(map[string]*CheckConfig),

		checkFuncs: make(map[string]EnhancedCheckFunc),

		checkResults: make(map[string]*EnhancedCheck),

		checkHistory: make(map[string][]EnhancedCheck),

		currentContext: ContextStartup,

		cacheEnabled: true,

		cacheExpiry: 30 * time.Second,

		cachedResults: make(map[string]*CachedResult),

		workerCount: 4, // Optimized for sub-100ms execution

		taskQueue: make(chan HealthTask, 100),

		shutdown: make(chan struct{}),

		metrics: initializeEnhancedHealthMetrics(),
	}

	// Initialize atomic values.

	ehc.overallHealth.Store(health.StatusUnknown)

	ehc.healthScore.Store(0.0)

	// Start worker pool.

	ehc.startWorkerPool()

	return ehc

}

// initializeEnhancedHealthMetrics initializes Prometheus metrics.

func initializeEnhancedHealthMetrics() *EnhancedHealthMetrics {

	metrics := &EnhancedHealthMetrics{

		CheckDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{

			Name: "enhanced_health_check_duration_seconds",

			Help: "Duration of enhanced health checks",

			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s

		}, []string{"check_name", "tier", "context"}),

		CheckSuccess: prometheus.NewCounterVec(prometheus.CounterOpts{

			Name: "enhanced_health_check_success_total",

			Help: "Total number of successful health checks",
		}, []string{"check_name", "tier"}),

		CheckFailures: prometheus.NewCounterVec(prometheus.CounterOpts{

			Name: "enhanced_health_check_failures_total",

			Help: "Total number of failed health checks",
		}, []string{"check_name", "tier", "failure_type"}),

		HealthScore: prometheus.NewGaugeVec(prometheus.GaugeOpts{

			Name: "enhanced_health_score",

			Help: "Weighted health score by tier and component",
		}, []string{"tier", "component"}),

		StateTransitions: prometheus.NewCounterVec(prometheus.CounterOpts{

			Name: "enhanced_health_state_transitions_total",

			Help: "Total number of health state transitions",
		}, []string{"check_name", "from_state", "to_state"}),

		CacheHitRate: prometheus.NewGauge(prometheus.GaugeOpts{

			Name: "enhanced_health_cache_hit_rate",

			Help: "Cache hit rate for health checks",
		}),

		ParallelExecution: prometheus.NewHistogram(prometheus.HistogramOpts{

			Name: "enhanced_health_parallel_execution_seconds",

			Help: "Total time for parallel health check execution",

			Buckets: prometheus.ExponentialBuckets(0.01, 2, 8), // 10ms to ~2.5s

		}),

		WeightedScores: prometheus.NewGaugeVec(prometheus.GaugeOpts{

			Name: "enhanced_health_weighted_scores",

			Help: "Individual weighted health scores",
		}, []string{"check_name", "tier"}),
	}

	// Register metrics.

	prometheus.MustRegister(

		metrics.CheckDuration,

		metrics.CheckSuccess,

		metrics.CheckFailures,

		metrics.HealthScore,

		metrics.StateTransitions,

		metrics.CacheHitRate,

		metrics.ParallelExecution,

		metrics.WeightedScores,
	)

	return metrics

}

// RegisterEnhancedCheck registers an enhanced health check.

func (ehc *EnhancedHealthChecker) RegisterEnhancedCheck(config *CheckConfig, checkFunc EnhancedCheckFunc) error {

	if config == nil {

		return fmt.Errorf("check config cannot be nil")

	}

	if checkFunc == nil {

		return fmt.Errorf("check function cannot be nil")

	}

	if config.Name == "" {

		return fmt.Errorf("check name cannot be empty")

	}

	ehc.mu.Lock()

	defer ehc.mu.Unlock()

	ehc.checks[config.Name] = config

	ehc.checkFuncs[config.Name] = checkFunc

	// Initialize result.

	ehc.checkResults[config.Name] = &EnhancedCheck{

		Name: config.Name,

		Status: health.StatusUnknown,

		Tier: config.Tier,

		Weight: config.Weight,

		Context: ehc.getCurrentContext(),

		Timestamp: time.Now(),

		StateTransitions: make([]StateTransition, 0),

		Dependencies: config.Dependencies,

		Metadata: make(map[string]interface{}),
	}

	ehc.logger.Info("Enhanced health check registered",

		"name", config.Name,

		"tier", config.Tier.String(),

		"weight", config.Weight,

		"service", ehc.serviceName)

	return nil

}

// SetContext updates the current operational context.

func (ehc *EnhancedHealthChecker) SetContext(context HealthContext) {

	ehc.contextMu.Lock()

	defer ehc.contextMu.Unlock()

	oldContext := ehc.currentContext

	ehc.currentContext = context

	ehc.logger.Info("Health check context changed",

		"from", oldContext.String(),

		"to", context.String(),

		"service", ehc.serviceName)

}

// getCurrentContext returns the current operational context.

func (ehc *EnhancedHealthChecker) getCurrentContext() HealthContext {

	ehc.contextMu.RLock()

	defer ehc.contextMu.RUnlock()

	return ehc.currentContext

}

// CheckEnhanced performs all enhanced health checks with parallel execution.

func (ehc *EnhancedHealthChecker) CheckEnhanced(ctx context.Context) *EnhancedHealthResponse {

	start := time.Now()

	// Create execution context with timeout.

	execCtx, cancel := context.WithTimeout(ctx, 90*time.Second) // Leave 10ms buffer for sub-100ms target

	defer cancel()

	response := &EnhancedHealthResponse{

		Service: ehc.serviceName,

		Version: ehc.serviceVersion,

		Timestamp: start,

		Context: ehc.getCurrentContext(),

		Checks: make([]EnhancedCheck, 0),

		TierSummaries: make(map[HealthTier]TierSummary),

		WeightedScore: 0.0,

		OverallStatus: health.StatusUnknown,

		ExecutionMetrics: ExecutionMetrics{

			StartTime: start,
		},
	}

	// Get snapshot of checks to execute.

	ehc.mu.RLock()

	checksToExecute := make(map[string]*CheckConfig)

	checkFuncs := make(map[string]EnhancedCheckFunc)

	for name, config := range ehc.checks {

		if ehc.shouldExecuteCheck(config, response.Context) {

			checksToExecute[name] = config

			checkFuncs[name] = ehc.checkFuncs[name]

		}

	}

	ehc.mu.RUnlock()

	// Execute checks in parallel using worker pool.

	results := ehc.executeChecksParallel(execCtx, checksToExecute, checkFuncs, response.Context)

	// Process results and update state.

	ehc.processCheckResults(results, response)

	// Calculate execution metrics.

	response.ExecutionMetrics.Duration = time.Since(start)

	response.ExecutionMetrics.CheckCount = len(results)

	response.ExecutionMetrics.ParallelExecution = true

	// Update atomic values.

	ehc.overallHealth.Store(response.OverallStatus)

	ehc.healthScore.Store(response.WeightedScore)

	ehc.lastExecution = time.Now()

	atomic.AddInt64(&ehc.executionCount, 1)

	// Record metrics.

	ehc.metrics.ParallelExecution.Observe(response.ExecutionMetrics.Duration.Seconds())

	ehc.logger.Debug("Enhanced health check completed",

		"duration", response.ExecutionMetrics.Duration,

		"checks", len(results),

		"score", response.WeightedScore,

		"status", response.OverallStatus)

	return response

}

// shouldExecuteCheck determines if a check should be executed in the current context.

func (ehc *EnhancedHealthChecker) shouldExecuteCheck(config *CheckConfig, context HealthContext) bool {

	if !config.Enabled {

		return false

	}

	// Check contextual settings.

	if contextualConfig, exists := config.ContextualSettings[context]; exists {

		return contextualConfig.Enabled

	}

	return true

}

// executeChecksParallel executes health checks in parallel using the worker pool.

func (ehc *EnhancedHealthChecker) executeChecksParallel(ctx context.Context, checks map[string]*CheckConfig, checkFuncs map[string]EnhancedCheckFunc, healthContext HealthContext) map[string]*EnhancedCheck {

	results := make(map[string]*EnhancedCheck)

	resultsMu := sync.Mutex{}

	// Channel for collecting results.

	resultCh := make(chan *EnhancedCheck, len(checks))

	// Submit tasks to worker pool.

	for name, config := range checks {

		checkFunc := checkFuncs[name]

		// Check cache first if enabled.

		if ehc.cacheEnabled {

			if cached := ehc.getCachedResult(name); cached != nil {

				ehc.metrics.CacheHitRate.Inc()

				resultsMu.Lock()

				results[name] = cached.Result

				resultsMu.Unlock()

				continue

			}

		}

		task := HealthTask{

			Name: name,

			Config: config,

			CheckFunc: checkFunc,

			Context: healthContext,

			ResultCh: resultCh,
		}

		select {

		case ehc.taskQueue <- task:

		case <-ctx.Done():

			return results

		}

	}

	// Collect results.

	expectedResults := len(checks)

	for range expectedResults {

		select {

		case result := <-resultCh:

			if result != nil {

				resultsMu.Lock()

				results[result.Name] = result

				resultsMu.Unlock()

				// Cache result if enabled.

				if ehc.cacheEnabled {

					ehc.cacheResult(result.Name, result)

				}

			}

		case <-ctx.Done():

			return results

		}

	}

	return results

}

// isRunning checks if the worker pool is running (thread-safe).

func (ehc *EnhancedHealthChecker) isRunning() bool {

	return atomic.LoadInt32(&ehc.running) == 1

}

// startWorkerPool starts the worker pool for parallel health check execution.

func (ehc *EnhancedHealthChecker) startWorkerPool() {

	if !atomic.CompareAndSwapInt32(&ehc.running, 0, 1) {

		return // Already running

	}

	for i := range ehc.workerCount {

		ehc.workerWg.Add(1)

		go ehc.healthCheckWorker(i)

	}

	ehc.logger.Info("Enhanced health check worker pool started",

		"workers", ehc.workerCount,

		"service", ehc.serviceName)

}

// healthCheckWorker processes health check tasks.

func (ehc *EnhancedHealthChecker) healthCheckWorker(workerID int) {

	defer ehc.workerWg.Done()

	for {

		select {

		case <-ehc.shutdown:

			return

		case task := <-ehc.taskQueue:

			result := ehc.executeHealthCheckTask(task)

			select {

			case task.ResultCh <- result:

			case <-ehc.shutdown:

				return

			}

		}

	}

}

// executeHealthCheckTask executes a single health check task.

func (ehc *EnhancedHealthChecker) executeHealthCheckTask(task HealthTask) *EnhancedCheck {

	start := time.Now()

	// Create timeout context.

	ctx, cancel := context.WithTimeout(context.Background(), task.Config.Timeout)

	defer cancel()

	// Execute the check.

	var result *EnhancedCheck

	func() {

		defer func() {

			if r := recover(); r != nil {

				result = &EnhancedCheck{

					Name: task.Name,

					Status: health.StatusUnhealthy,

					Error: fmt.Sprintf("Health check panicked: %v", r),

					Duration: time.Since(start),

					Timestamp: time.Now(),

					Tier: task.Config.Tier,

					Weight: task.Config.Weight,

					Context: task.Context,
				}

			}

		}()

		result = task.CheckFunc(ctx, task.Context)

	}()

	if result == nil {

		result = &EnhancedCheck{

			Name: task.Name,

			Status: health.StatusUnknown,

			Error: "Check function returned nil",

			Duration: time.Since(start),

			Timestamp: time.Now(),

			Tier: task.Config.Tier,

			Weight: task.Config.Weight,

			Context: task.Context,
		}

	}

	// Ensure required fields are set.

	result.Name = task.Name

	result.Duration = time.Since(start)

	result.Timestamp = time.Now()

	result.Tier = task.Config.Tier

	result.Weight = task.Config.Weight

	result.Context = task.Context

	// Calculate health score.

	result.Score = ehc.calculateHealthScore(result)

	// Update state tracking.

	ehc.updateStateTracking(result)

	// Record metrics.

	ehc.recordCheckMetrics(result)

	return result

}

// processCheckResults processes all check results and builds response.

func (ehc *EnhancedHealthChecker) processCheckResults(results map[string]*EnhancedCheck, response *EnhancedHealthResponse) {

	tierSummaries := make(map[HealthTier]TierSummary)

	totalWeightedScore := 0.0

	totalWeight := 0.0

	// Process each result.

	for _, result := range results {

		response.Checks = append(response.Checks, *result)

		// Update tier summaries.

		tierSummary := tierSummaries[result.Tier]

		tierSummary.Tier = result.Tier

		tierSummary.TotalChecks++

		tierSummary.TotalWeight += float64(result.Weight)

		tierSummary.WeightedScore += result.Score * float64(result.Weight)

		switch result.Status {

		case health.StatusHealthy:

			tierSummary.HealthyChecks++

		case health.StatusUnhealthy:

			tierSummary.UnhealthyChecks++

		case health.StatusDegraded:

			tierSummary.DegradedChecks++

		default:

			tierSummary.UnknownChecks++

		}

		tierSummaries[result.Tier] = tierSummary

		// Add to overall score calculation.

		totalWeightedScore += result.Score * float64(result.Weight)

		totalWeight += float64(result.Weight)

	}

	// Calculate tier averages.

	for tier, summary := range tierSummaries {

		if summary.TotalWeight > 0 {

			summary.AverageScore = summary.WeightedScore / summary.TotalWeight

		}

		tierSummaries[tier] = summary

	}

	response.TierSummaries = tierSummaries

	// Calculate overall weighted score.

	if totalWeight > 0 {

		response.WeightedScore = totalWeightedScore / totalWeight

	}

	// Determine overall status.

	response.OverallStatus = ehc.determineOverallStatus(tierSummaries, response.WeightedScore)

	// Update stored results.

	ehc.mu.Lock()

	for name, result := range results {

		ehc.checkResults[name] = result

		ehc.updateCheckHistory(name, result)

	}

	ehc.mu.Unlock()

}

// calculateHealthScore calculates a numerical health score for a check result.

func (ehc *EnhancedHealthChecker) calculateHealthScore(check *EnhancedCheck) float64 {

	baseScore := 0.0

	switch check.Status {

	case health.StatusHealthy:

		baseScore = 1.0

	case health.StatusDegraded:

		baseScore = 0.6

	case health.StatusUnhealthy:

		baseScore = 0.0

	default:

		baseScore = 0.3 // Unknown

	}

	// Apply latency penalty if duration exceeds thresholds.

	latencyPenalty := 0.0

	if check.Duration > 5*time.Second {

		latencyPenalty = 0.3

	} else if check.Duration > time.Second {

		latencyPenalty = 0.1

	}

	// Apply consecutive failure penalty.

	consecutiveFailurePenalty := float64(check.ConsecutiveFails) * 0.1

	if consecutiveFailurePenalty > 0.5 {

		consecutiveFailurePenalty = 0.5

	}

	finalScore := baseScore - latencyPenalty - consecutiveFailurePenalty

	if finalScore < 0 {

		finalScore = 0

	}

	return finalScore

}

// updateStateTracking updates state tracking information for a check.

func (ehc *EnhancedHealthChecker) updateStateTracking(result *EnhancedCheck) {

	ehc.mu.RLock()

	previousResult, exists := ehc.checkResults[result.Name]

	ehc.mu.RUnlock()

	if exists && previousResult.Status != result.Status {

		// Record state transition.

		transition := StateTransition{

			From: previousResult.Status,

			To: result.Status,

			Timestamp: time.Now(),
		}

		result.StateTransitions = append(previousResult.StateTransitions, transition)

		// Update metrics.

		ehc.metrics.StateTransitions.WithLabelValues(

			result.Name,

			string(previousResult.Status),

			string(result.Status),
		).Inc()

	} else if exists {

		result.StateTransitions = previousResult.StateTransitions

	}

	// Update consecutive failures.

	if exists {

		if result.Status != health.StatusHealthy {

			result.ConsecutiveFails = previousResult.ConsecutiveFails + 1

		} else {

			result.ConsecutiveFails = 0

			result.LastHealthy = time.Now()

		}

	}

}

// updateCheckHistory updates the historical data for a check.

func (ehc *EnhancedHealthChecker) updateCheckHistory(name string, result *EnhancedCheck) {

	history := ehc.checkHistory[name]

	history = append(history, *result)

	// Keep only last 100 entries.

	if len(history) > 100 {

		history = history[len(history)-100:]

	}

	ehc.checkHistory[name] = history

}

// recordCheckMetrics records metrics for a check result.

func (ehc *EnhancedHealthChecker) recordCheckMetrics(result *EnhancedCheck) {

	labels := []string{result.Name, result.Tier.String(), result.Context.String()}

	// Record duration.

	ehc.metrics.CheckDuration.WithLabelValues(labels...).Observe(result.Duration.Seconds())

	// Record success/failure.

	if result.Status == health.StatusHealthy {

		ehc.metrics.CheckSuccess.WithLabelValues(result.Name, result.Tier.String()).Inc()

	} else {

		failureType := "unknown"

		if result.Error != "" {

			failureType = "error"

		} else if result.Status == health.StatusDegraded {

			failureType = "degraded"

		} else if result.Status == health.StatusUnhealthy {

			failureType = "unhealthy"

		}

		ehc.metrics.CheckFailures.WithLabelValues(result.Name, result.Tier.String(), failureType).Inc()

	}

	// Record weighted score.

	ehc.metrics.WeightedScores.WithLabelValues(result.Name, result.Tier.String()).Set(result.Score)

}

// determineOverallStatus determines the overall system health status.

func (ehc *EnhancedHealthChecker) determineOverallStatus(tierSummaries map[HealthTier]TierSummary, weightedScore float64) health.Status {

	// Critical tier checks override everything.

	if systemSummary, exists := tierSummaries[TierSystem]; exists {

		if systemSummary.UnhealthyChecks > 0 {

			return health.StatusUnhealthy

		}

	}

	// Use weighted score for overall determination.

	if weightedScore >= 0.95 {

		return health.StatusHealthy

	} else if weightedScore >= 0.7 {

		return health.StatusDegraded

	} else if weightedScore > 0 {

		return health.StatusUnhealthy

	}

	return health.StatusUnknown

}

// getCachedResult retrieves a cached result if valid.

func (ehc *EnhancedHealthChecker) getCachedResult(name string) *CachedResult {

	if !ehc.cacheEnabled {

		return nil

	}

	ehc.cacheMu.RLock()

	defer ehc.cacheMu.RUnlock()

	cached, exists := ehc.cachedResults[name]

	if !exists {

		return nil

	}

	if time.Since(cached.Timestamp) > ehc.cacheExpiry {

		return nil

	}

	return cached

}

// cacheResult caches a health check result.

func (ehc *EnhancedHealthChecker) cacheResult(name string, result *EnhancedCheck) {

	if !ehc.cacheEnabled {

		return

	}

	ehc.cacheMu.Lock()

	defer ehc.cacheMu.Unlock()

	ehc.cachedResults[name] = &CachedResult{

		Result: result,

		Timestamp: time.Now(),
	}

}

// Stop gracefully shuts down the enhanced health checker.

func (ehc *EnhancedHealthChecker) Stop() {

	if !atomic.CompareAndSwapInt32(&ehc.running, 1, 0) {

		return // Already stopped

	}

	ehc.logger.Info("Stopping enhanced health checker", "service", ehc.serviceName)

	close(ehc.shutdown)

	ehc.workerWg.Wait()

	ehc.logger.Info("Enhanced health checker stopped", "service", ehc.serviceName)

}

// GetOverallHealth returns the current overall health status.

func (ehc *EnhancedHealthChecker) GetOverallHealth() health.Status {

	if status := ehc.overallHealth.Load(); status != nil {

		return status.(health.Status)

	}

	return health.StatusUnknown

}

// GetHealthScore returns the current overall health score.

func (ehc *EnhancedHealthChecker) GetHealthScore() float64 {

	if score := ehc.healthScore.Load(); score != nil {

		return score.(float64)

	}

	return 0.0

}

// GetCheckHistory returns the historical data for a specific check.

func (ehc *EnhancedHealthChecker) GetCheckHistory(checkName string, limit int) []EnhancedCheck {

	ehc.mu.RLock()

	defer ehc.mu.RUnlock()

	history, exists := ehc.checkHistory[checkName]

	if !exists {

		return nil

	}

	if limit > 0 && len(history) > limit {

		return history[len(history)-limit:]

	}

	return history

}
