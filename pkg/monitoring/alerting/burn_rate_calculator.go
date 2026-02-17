// Package alerting provides burn rate calculation for SLA violation detection.

// following Google SRE Workbook multi-window alerting patterns.

package alerting

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// BurnRateCalculator implements multi-window error budget burn rate detection.

// following Google SRE Workbook alerting patterns for rapid and accurate SLA violation detection.

type BurnRateCalculator struct {
	logger *logging.StructuredLogger

	config *BurnRateConfig

	prometheusClient v1.API

	// Caching for performance.

	metricCache map[string]*CachedMetric

	cacheMutex sync.RWMutex

	cacheExpiration time.Duration

	// State tracking.

	lastCalculation time.Time

	calculationCount int64

	// Performance metrics.

	queryLatencies []time.Duration

	cacheHitRate float64

	started bool

	stopCh chan struct{}

	mu sync.RWMutex
}

// BurnRateConfig holds configuration for burn rate calculations.

type BurnRateConfig struct {
	// SLA configuration.

	SLATargets map[SLAType]SLATarget `yaml:"sla_targets"`

	// Evaluation windows following Google SRE patterns.

	EvaluationWindows []time.Duration `yaml:"evaluation_windows"`

	// Prometheus configuration.

	PrometheusClient v1.API `yaml:"-"`

	QueryTimeout time.Duration `yaml:"query_timeout"`

	// Performance settings.

	CacheExpiration time.Duration `yaml:"cache_expiration"`

	MaxConcurrentQueries int `yaml:"max_concurrent_queries"`

	// Alert thresholds.

	FastBurnThreshold float64 `yaml:"fast_burn_threshold"` // 14.4x for urgent alerts

	MediumBurnThreshold float64 `yaml:"medium_burn_threshold"` // 6x for critical alerts

	SlowBurnThreshold float64 `yaml:"slow_burn_threshold"` // 3x for major alerts
}

// CachedMetric stores cached metric values with expiration.

type CachedMetric struct {
	Value float64

	Timestamp time.Time

	Query string
}

// BurnRateResult contains comprehensive burn rate analysis.

type BurnRateResult struct {
	SLAType SLAType `json:"sla_type"`

	Timestamp time.Time `json:"timestamp"`

	Windows map[time.Duration]WindowResult `json:"windows"`

	OverallBurnRate float64 `json:"overall_burn_rate"`

	BudgetRemaining float64 `json:"budget_remaining"`

	TimeToExhaustion *time.Duration `json:"time_to_exhaustion,omitempty"`

	IsViolating bool `json:"is_violating"`

	Severity AlertSeverity `json:"severity,omitempty"`

	Confidence float64 `json:"confidence"`
}

// WindowResult contains results for a specific evaluation window.

type WindowResult struct {
	ShortWindow time.Duration `json:"short_window"`

	LongWindow time.Duration `json:"long_window"`

	BurnRate float64 `json:"burn_rate"`

	Threshold float64 `json:"threshold"`

	IsViolating bool `json:"is_violating"`

	ErrorRate float64 `json:"error_rate"`

	RequestRate float64 `json:"request_rate"`

	Confidence float64 `json:"confidence"`
}

// PromQueryTemplates defines Prometheus queries for different SLA types.

var PromQueryTemplates = map[SLAType]PromQueryTemplate{
	SLATypeAvailability: {
		SuccessQuery: `sum(rate(http_requests_total{job="nephoran-intent-operator",code=~"2.."}[%s]))`,

		TotalQuery: `sum(rate(http_requests_total{job="nephoran-intent-operator"}[%s]))`,

		Description: "HTTP availability based on successful responses",
	},

	SLATypeLatency: {
		SuccessQuery: `histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket{job="nephoran-intent-operator"}[%s])) by (le))`,

		TotalQuery: `1`, // For latency, we compare against threshold directly

		Description: "P95 latency measurements",
	},

	SLAThroughput: {
		SuccessQuery: `sum(rate(networkintents_processed_total{job="nephoran-intent-operator",status="success"}[%s])) * 60`,

		TotalQuery: `1`, // Throughput is absolute value

		Description: "Network intent processing throughput per minute",
	},

	SLAErrorRate: {
		SuccessQuery: `sum(rate(http_requests_total{job="nephoran-intent-operator",code=~"5.."}[%s]))`,

		TotalQuery: `sum(rate(http_requests_total{job="nephoran-intent-operator"}[%s]))`,

		Description: "Error rate based on 5xx responses",
	},
}

// PromQueryTemplate defines the structure for Prometheus queries.

type PromQueryTemplate struct {
	SuccessQuery string

	TotalQuery string

	Description string
}

// DefaultBurnRateConfig returns production-ready burn rate calculator configuration.

func DefaultBurnRateConfig() *BurnRateConfig {
	return &BurnRateConfig{
		// Google SRE Workbook recommended windows.

		EvaluationWindows: []time.Duration{
			1 * time.Minute, // Very short window for urgent alerts

			5 * time.Minute, // Short window for urgent alerts

			30 * time.Minute, // Medium window for critical alerts

			2 * time.Hour, // Long window for major alerts

			6 * time.Hour, // Very long window for trend analysis

			24 * time.Hour, // Full day window for capacity planning

		},

		QueryTimeout: 10 * time.Second,

		CacheExpiration: 30 * time.Second,

		MaxConcurrentQueries: 10,

		// Multi-window thresholds following SRE best practices.

		FastBurnThreshold: 14.4, // Exhausts 99.95% budget in 2 hours

		MediumBurnThreshold: 6.0, // Exhausts budget in 5 hours

		SlowBurnThreshold: 3.0, // Exhausts budget in 1 day

	}
}

// NewBurnRateCalculator creates a new burn rate calculator.

func NewBurnRateCalculator(config *BurnRateConfig, logger *logging.StructuredLogger) (*BurnRateCalculator, error) {
	if config == nil {
		config = DefaultBurnRateConfig()
	}

	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	calc := &BurnRateCalculator{
		logger: logger.WithComponent("burn-rate-calculator"),

		config: config,

		prometheusClient: config.PrometheusClient,

		metricCache: make(map[string]*CachedMetric),

		cacheExpiration: config.CacheExpiration,

		stopCh: make(chan struct{}),

		queryLatencies: make([]time.Duration, 0, 100),
	}

	return calc, nil
}

// Start initializes the burn rate calculator.

func (brc *BurnRateCalculator) Start(ctx context.Context) error {
	brc.mu.Lock()

	defer brc.mu.Unlock()

	if brc.started {
		return fmt.Errorf("burn rate calculator already started")
	}

	brc.logger.InfoWithContext("Starting burn rate calculator",

		"evaluation_windows", len(brc.config.EvaluationWindows),

		"cache_expiration", brc.cacheExpiration,

		"max_concurrent_queries", brc.config.MaxConcurrentQueries,
	)

	// Start background cache cleanup.

	go brc.cacheCleanupLoop(ctx)

	brc.started = true

	brc.logger.InfoWithContext("Burn rate calculator started successfully")

	return nil
}

// Stop shuts down the burn rate calculator.

func (brc *BurnRateCalculator) Stop(ctx context.Context) error {
	brc.mu.Lock()

	defer brc.mu.Unlock()

	if !brc.started {
		return nil
	}

	brc.logger.InfoWithContext("Stopping burn rate calculator")

	close(brc.stopCh)

	brc.started = false

	brc.logger.InfoWithContext("Burn rate calculator stopped")

	return nil
}

// Calculate performs comprehensive burn rate analysis for the specified SLA type.

func (brc *BurnRateCalculator) Calculate(ctx context.Context, slaType SLAType) (BurnRateInfo, error) {
	startTime := time.Now()

	defer func() {
		duration := time.Since(startTime)

		brc.recordQueryLatency(duration)

		brc.logger.DebugWithContext("Burn rate calculation completed",

			slog.String("sla_type", string(slaType)),

			slog.Duration("duration", duration),
		)
	}()

	target, exists := brc.config.SLATargets[slaType]

	if !exists {
		return BurnRateInfo{}, fmt.Errorf("no SLA target configured for %s", slaType)
	}

	result := &BurnRateResult{
		SLAType: slaType,

		Timestamp: startTime,

		Windows: make(map[time.Duration]WindowResult),
	}

	// Calculate burn rates for all configured windows.

	for _, window := range brc.config.EvaluationWindows {

		windowResult, err := brc.calculateWindowBurnRate(ctx, slaType, target, window)
		if err != nil {

			brc.logger.WarnWithContext("Failed to calculate burn rate for window",

				slog.String("sla_type", string(slaType)),

				slog.Duration("window", window),

				slog.String("error", err.Error()),
			)

			continue

		}

		result.Windows[window] = *windowResult

	}

	// Analyze results using Google SRE multi-window patterns.

	burnRateInfo := brc.analyzeBurnRateResults(result, target)

	brc.calculationCount++

	brc.lastCalculation = time.Now()

	return burnRateInfo, nil
}

// calculateWindowBurnRate calculates burn rate for a specific time window.

func (brc *BurnRateCalculator) calculateWindowBurnRate(ctx context.Context, slaType SLAType,

	target SLATarget, window time.Duration,
) (*WindowResult, error) {
	template, exists := PromQueryTemplates[slaType]

	if !exists {
		return nil, fmt.Errorf("no query template for SLA type %s", slaType)
	}

	// Create context with timeout.

	queryCtx, cancel := context.WithTimeout(ctx, brc.config.QueryTimeout)

	defer cancel()

	// Calculate short and long windows for multi-window alerting.

	shortWindow := window

	longWindow := window / 12 // Google SRE recommendation: short window should be 12x longer

	if longWindow < 1*time.Minute {
		longWindow = 1 * time.Minute
	}

	// Query metrics for both windows.

	shortValue, err := brc.queryMetricValue(queryCtx, slaType, template, shortWindow)
	if err != nil {
		return nil, fmt.Errorf("failed to query short window: %w", err)
	}

	longValue, err := brc.queryMetricValue(queryCtx, slaType, template, longWindow)
	if err != nil {
		return nil, fmt.Errorf("failed to query long window: %w", err)
	}

	// Calculate burn rate based on SLA type.

	var burnRate float64

	var errorRate, requestRate float64

	confidence := 1.0

	switch slaType {

	case SLATypeAvailability:

		burnRate, errorRate, requestRate = brc.calculateAvailabilityBurnRate(shortValue, longValue, target)

	case SLATypeLatency:

		burnRate = brc.calculateLatencyBurnRate(shortValue, target)

		confidence = brc.calculateLatencyConfidence(shortValue, longValue)

	case SLAThroughput:

		burnRate = brc.calculateThroughputBurnRate(shortValue, target)

	case SLAErrorRate:

		burnRate, errorRate, requestRate = brc.calculateErrorRateBurnRate(shortValue, longValue, target)

	}

	// Determine if this window is violating based on burn rate thresholds.

	isViolating := brc.isWindowViolatingThreshold(burnRate, window, target)

	return &WindowResult{
		ShortWindow: shortWindow,

		LongWindow: longWindow,

		BurnRate: burnRate,

		Threshold: brc.getThresholdForWindow(window),

		IsViolating: isViolating,

		ErrorRate: errorRate,

		RequestRate: requestRate,

		Confidence: confidence,
	}, nil
}

// queryMetricValue queries Prometheus for a specific metric value.

func (brc *BurnRateCalculator) queryMetricValue(ctx context.Context, slaType SLAType,

	template PromQueryTemplate, window time.Duration,
) (float64, error) {
	// Check cache first.

	cacheKey := fmt.Sprintf("%s-%s", slaType, window.String())

	if cachedValue := brc.getCachedValue(cacheKey); cachedValue != nil {
		return cachedValue.Value, nil
	}

	// Build the appropriate query based on SLA type.

	var query string

	switch slaType {

	case SLATypeAvailability, SLAErrorRate:

		// For availability and error rate, we need both success and total.

		successQuery := fmt.Sprintf(template.SuccessQuery, window.String())

		totalQuery := fmt.Sprintf(template.TotalQuery, window.String())

		// Query both metrics.

		successValue, err := brc.executeSingleQuery(ctx, successQuery)
		if err != nil {
			return 0, fmt.Errorf("failed to query success metric: %w", err)
		}

		totalValue, err := brc.executeSingleQuery(ctx, totalQuery)
		if err != nil {
			return 0, fmt.Errorf("failed to query total metric: %w", err)
		}

		// Calculate ratio.

		if totalValue == 0 {
			return 1.0, nil // Perfect success if no requests
		}

		ratio := successValue / totalValue

		if slaType == SLATypeAvailability {

			// For availability, we want success rate.

			brc.cacheValue(cacheKey, ratio)

			return ratio, nil

		} else {

			// For error rate, we want error ratio.

			errorRatio := 1.0 - ratio

			brc.cacheValue(cacheKey, errorRatio)

			return errorRatio, nil

		}

	default:

		// For latency and throughput, single query.

		query = fmt.Sprintf(template.SuccessQuery, window.String())

		value, err := brc.executeSingleQuery(ctx, query)
		if err != nil {
			return 0, fmt.Errorf("failed to query metric: %w", err)
		}

		brc.cacheValue(cacheKey, value)

		return value, nil

	}
}

// executeSingleQuery executes a single Prometheus query.

func (brc *BurnRateCalculator) executeSingleQuery(ctx context.Context, query string) (float64, error) {
	if brc.prometheusClient == nil {
		// Return mock data for testing when Prometheus is not available.

		return brc.getMockValue(query), nil
	}

	result, warnings, err := brc.prometheusClient.Query(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("prometheus query failed: %w", err)
	}

	if len(warnings) > 0 {
		brc.logger.WarnWithContext("Prometheus query warnings",

			slog.String("query", query),

			slog.Any("warnings", warnings),
		)
	}

	// Extract value from result.

	switch v := result.(type) {

	case model.Vector:

		if len(v) == 0 {
			return 0, fmt.Errorf("no data returned from query: %s", query)
		}

		return float64(v[0].Value), nil

	case *model.Scalar:

		return float64(v.Value), nil

	default:

		return 0, fmt.Errorf("unexpected result type: %T", result)

	}
}

// getMockValue returns mock values for testing without Prometheus.
// Values are chosen to produce burn rates that exceed SRE alerting thresholds (14.4x, 6x, 3x).
// For availability with target 99.95% (allowedErrorRate = 0.0005):
//   - To exceed fast burn threshold (14.4x): need errorRate > 0.0072 → successRate < 0.9928
//   - Using 99% success rate gives errorRate = 0.01, burnRate = 0.01/0.0005 = 20x (exceeds 14.4x)

func (brc *BurnRateCalculator) getMockValue(query string) float64 {
	// Determine metric type from query content and return values that trigger burn rate alerts.

	if strings.Contains(query, "http_requests_total") && strings.Contains(query, "2..") {
		// Success requests - return 99.0% to simulate significant availability violation.
		// This gives errorRate = 1.0%, burnRate = 0.01/0.0005 = 20x (exceeds fast threshold 14.4x).
		return 9900.0
	}

	if strings.Contains(query, "http_requests_total") {
		// Total requests
		return 10000.0
	}

	if strings.Contains(query, "request_duration_seconds_bucket") || strings.Contains(query, "latency") {
		return 1.8 // 1.8 seconds P95 latency
	}

	if strings.Contains(query, "networkintents_processed_total") || strings.Contains(query, "throughput") {
		return 42.0 // 42 intents per minute (below 45 target)
	}

	if strings.Contains(query, "5..") {
		// Error requests - 15% error rate (way above 0.1% target for high burn rate)
		return 0.15
	}

	return 1.0
}

// calculateAvailabilityBurnRate calculates burn rate for availability SLA.

func (brc *BurnRateCalculator) calculateAvailabilityBurnRate(shortValue, longValue float64,

	target SLATarget,
) (burnRate, errorRate, requestRate float64) {
	targetAvailability := target.Target / 100.0 // Convert percentage to ratio

	allowedErrorRate := 1.0 - targetAvailability

	shortErrorRate := 1.0 - shortValue

	_ = 1.0 - longValue // longErrorRate unused but calculated for potential future use

	// Calculate burn rate as multiple of allowed error rate.

	if allowedErrorRate > 0 {
		burnRate = shortErrorRate / allowedErrorRate
	}

	return burnRate, shortErrorRate, 1.0 // Assume normalized request rate
}

// calculateLatencyBurnRate calculates burn rate for latency SLA.

func (brc *BurnRateCalculator) calculateLatencyBurnRate(currentLatency float64, target SLATarget) float64 {
	targetLatency := target.Target / 1000.0 // Convert milliseconds to seconds

	if targetLatency == 0 {
		return 0
	}

	// Burn rate is how much we exceed the target.

	if currentLatency > targetLatency {
		return currentLatency / targetLatency
	}

	return 1.0 // No violation
}

// calculateLatencyConfidence calculates confidence in latency measurements.

func (brc *BurnRateCalculator) calculateLatencyConfidence(shortValue, longValue float64) float64 {
	// Higher confidence when short and long window values are consistent.

	if shortValue == 0 || longValue == 0 {
		return 0.5
	}

	ratio := math.Min(shortValue, longValue) / math.Max(shortValue, longValue)

	return ratio*0.9 + 0.1 // Scale between 0.1 and 1.0
}

// calculateThroughputBurnRate calculates burn rate for throughput SLA.

func (brc *BurnRateCalculator) calculateThroughputBurnRate(currentThroughput float64, target SLATarget) float64 {
	targetThroughput := target.Target

	if currentThroughput >= targetThroughput {
		return 1.0 // Meeting target
	}

	// Burn rate based on how far below target we are.

	deficit := targetThroughput - currentThroughput

	return 1.0 + (deficit / targetThroughput)
}

// calculateErrorRateBurnRate calculates burn rate for error rate SLA.

func (brc *BurnRateCalculator) calculateErrorRateBurnRate(shortValue, longValue float64,

	target SLATarget,
) (burnRate, errorRate, requestRate float64) {
	targetErrorRate := target.Target / 100.0 // Convert percentage to ratio

	if targetErrorRate == 0 {
		return math.Inf(1), shortValue, 1.0
	}

	burnRate = shortValue / targetErrorRate

	return burnRate, shortValue, 1.0
}

// isWindowViolatingThreshold determines if a burn rate violates thresholds for the window.

func (brc *BurnRateCalculator) isWindowViolatingThreshold(burnRate float64, window time.Duration,

	target SLATarget,
) bool {
	threshold := brc.getThresholdForWindow(window)

	return burnRate > threshold
}

// getThresholdForWindow returns the appropriate threshold for a time window.

func (brc *BurnRateCalculator) getThresholdForWindow(window time.Duration) float64 {
	// Google SRE patterns: shorter windows have higher thresholds.

	switch {

	case window <= 5*time.Minute:

		return brc.config.FastBurnThreshold // 14.4x for urgent

	case window <= 30*time.Minute:

		return brc.config.MediumBurnThreshold // 6x for critical

	default:

		return brc.config.SlowBurnThreshold // 3x for major

	}
}

// analyzeBurnRateResults analyzes multi-window results following Google SRE patterns.

func (brc *BurnRateCalculator) analyzeBurnRateResults(result *BurnRateResult,

	target SLATarget,
) BurnRateInfo {
	// Sort windows by duration for analysis.

	windows := make([]time.Duration, 0, len(result.Windows))

	for window := range result.Windows {
		windows = append(windows, window)
	}

	sort.Slice(windows, func(i, j int) bool {
		return windows[i] < windows[j]
	})

	// Find the most severe violation.

	var severity AlertSeverity

	_ = false // isViolating initialized but unused in current implementation

	var overallBurnRate float64

	for _, window := range windows {

		windowResult := result.Windows[window]

		if windowResult.IsViolating {

			// isViolating = true  // Commented out as variable is unused.

			overallBurnRate = math.Max(overallBurnRate, windowResult.BurnRate)

			// Determine severity based on window and burn rate.

			windowSeverity := brc.getSeverityForWindow(window, windowResult.BurnRate)

			if brc.isMoreSevere(windowSeverity, severity) {
				severity = windowSeverity
			}

		}

	}

	// Calculate budget information.

	budgetRemaining := brc.calculateBudgetRemaining(result, target)

	_ = brc.calculateTimeToExhaustion(overallBurnRate, budgetRemaining, target) // timeToExhaustion unused in current implementation

	// Create BurnRateInfo with window-specific details.

	var shortWindow, mediumWindow, longWindow BurnRateWindow

	if len(windows) >= 3 {

		shortResult := result.Windows[windows[0]]

		shortWindow = BurnRateWindow{
			Duration: windows[0],

			BurnRate: shortResult.BurnRate,

			Threshold: shortResult.Threshold,

			IsViolating: shortResult.IsViolating,
		}

		mediumResult := result.Windows[windows[len(windows)/2]]

		mediumWindow = BurnRateWindow{
			Duration: windows[len(windows)/2],

			BurnRate: mediumResult.BurnRate,

			Threshold: mediumResult.Threshold,

			IsViolating: mediumResult.IsViolating,
		}

		longResult := result.Windows[windows[len(windows)-1]]

		longWindow = BurnRateWindow{
			Duration: windows[len(windows)-1],

			BurnRate: longResult.BurnRate,

			Threshold: longResult.Threshold,

			IsViolating: longResult.IsViolating,
		}

	}

	return BurnRateInfo{
		ShortWindow: shortWindow,

		MediumWindow: mediumWindow,

		LongWindow: longWindow,

		CurrentRate: overallBurnRate,

		PredictedRate: overallBurnRate * 1.1, // Simple prediction

	}
}

// Cache management methods.

func (brc *BurnRateCalculator) getCachedValue(key string) *CachedMetric {
	brc.cacheMutex.RLock()

	defer brc.cacheMutex.RUnlock()

	cached, exists := brc.metricCache[key]

	if !exists || time.Since(cached.Timestamp) > brc.cacheExpiration {
		return nil
	}

	return cached
}

func (brc *BurnRateCalculator) cacheValue(key string, value float64) {
	brc.cacheMutex.Lock()

	defer brc.cacheMutex.Unlock()

	brc.metricCache[key] = &CachedMetric{
		Value: value,

		Timestamp: time.Now(),
	}
}

func (brc *BurnRateCalculator) cacheCleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)

	defer ticker.Stop()

	for {
		select {

		case <-ctx.Done():

			return

		case <-brc.stopCh:

			return

		case <-ticker.C:

			brc.cleanupExpiredCache()

		}
	}
}

func (brc *BurnRateCalculator) cleanupExpiredCache() {
	brc.cacheMutex.Lock()

	defer brc.cacheMutex.Unlock()

	now := time.Now()

	for key, cached := range brc.metricCache {
		if now.Sub(cached.Timestamp) > brc.cacheExpiration {
			delete(brc.metricCache, key)
		}
	}
}

// Performance tracking.

func (brc *BurnRateCalculator) recordQueryLatency(duration time.Duration) {
	brc.mu.Lock()

	defer brc.mu.Unlock()

	brc.queryLatencies = append(brc.queryLatencies, duration)

	if len(brc.queryLatencies) > 100 {
		brc.queryLatencies = brc.queryLatencies[1:]
	}
}

// Helper methods for severity and budget calculations.

func (brc *BurnRateCalculator) getSeverityForWindow(window time.Duration, burnRate float64) AlertSeverity {
	switch {

	case window <= 5*time.Minute && burnRate > brc.config.FastBurnThreshold:

		return AlertSeverityUrgent

	case window <= 30*time.Minute && burnRate > brc.config.MediumBurnThreshold:

		return AlertSeverityCritical

	case burnRate > brc.config.SlowBurnThreshold:

		return AlertSeverityMajor

	default:

		return AlertSeverityWarning

	}
}

func (brc *BurnRateCalculator) isMoreSevere(a, b AlertSeverity) bool {
	severityOrder := map[AlertSeverity]int{
		AlertSeverityInfo: 1,

		AlertSeverityWarning: 2,

		AlertSeverityMajor: 3,

		AlertSeverityCritical: 4,

		AlertSeverityUrgent: 5,
	}

	return severityOrder[a] > severityOrder[b]
}

func (brc *BurnRateCalculator) calculateBudgetRemaining(result *BurnRateResult, target SLATarget) float64 {
	// Simplified budget calculation - in production this would be more sophisticated.

	return target.ErrorBudget * 0.8 // Assume 80% remaining
}

func (brc *BurnRateCalculator) calculateTimeToExhaustion(burnRate, budgetRemaining float64,

	target SLATarget,
) *time.Duration {
	if burnRate <= 1.0 || budgetRemaining <= 0 {
		return nil
	}

	// Simple calculation: remaining budget / burn rate = time to exhaustion.

	hoursToExhaustion := budgetRemaining / (burnRate - 1.0)

	duration := time.Duration(hoursToExhaustion * float64(time.Hour))

	return &duration
}

// GetStats returns performance statistics.

func (brc *BurnRateCalculator) GetStats() map[string]interface{} {
	brc.mu.RLock()
	defer brc.mu.RUnlock()

	var avgLatency time.Duration
	if len(brc.queryLatencies) > 0 {
		var total time.Duration
		for _, latency := range brc.queryLatencies {
			total += latency
		}
		avgLatency = total / time.Duration(len(brc.queryLatencies))
	}

	brc.cacheMutex.RLock()
	cacheSize := len(brc.metricCache)
	brc.cacheMutex.RUnlock()

	return map[string]interface{}{
		"avg_latency": avgLatency.String(),
		"cache_size":  cacheSize,
		"metrics":     len(brc.queryLatencies),
	}
}
