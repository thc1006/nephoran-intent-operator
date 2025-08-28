package latency

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
)

// E2ELatencyTracker tracks end-to-end latency from intent submission to deployment completion
type E2ELatencyTracker struct {
	mu sync.RWMutex

	// Active intent tracking
	activeIntents map[string]*IntentTrace

	// Completed intent history
	completedIntents *CircularBuffer

	// Percentile calculators
	percentileCalc *StreamingPercentiles

	// Latency histograms
	latencyHistogram *LatencyHistogram

	// SLA compliance tracking
	slaTracker *SLAComplianceTracker

	// Anomaly detection
	anomalyDetector *LatencyAnomalyDetector

	// Latency budget tracking
	budgetTracker *LatencyBudgetTracker

	// User experience correlation
	uxCorrelator *UserExperienceCorrelator

	// Metrics
	metrics *E2EMetrics

	// Configuration
	config *E2ETrackerConfig

	// Statistics
	stats *E2EStatistics
}

// E2ETrackerConfig contains configuration for the E2E tracker
type E2ETrackerConfig struct {
	// SLA targets
	P50Target            time.Duration `json:"p50_target"`
	P95Target            time.Duration `json:"p95_target"`
	P99Target            time.Duration `json:"p99_target"`
	MaxAcceptableLatency time.Duration `json:"max_acceptable_latency"`

	// Histogram configuration
	HistogramBuckets   []float64 `json:"histogram_buckets"`
	HistogramPrecision int       `json:"histogram_precision"`

	// Anomaly detection
	AnomalyWindowSize time.Duration `json:"anomaly_window_size"`
	AnomalyThreshold  float64       `json:"anomaly_threshold"`

	// Budget allocation (percentages)
	ControllerBudget float64 `json:"controller_budget"`
	LLMBudget        float64 `json:"llm_budget"`
	RAGBudget        float64 `json:"rag_budget"`
	GitOpsBudget     float64 `json:"gitops_budget"`
	DeploymentBudget float64 `json:"deployment_budget"`

	// Retention
	HistorySize       int           `json:"history_size"`
	RetentionDuration time.Duration `json:"retention_duration"`
}

// IntentTrace represents a complete intent processing trace
type IntentTrace struct {
	ID               string                   `json:"id"`
	IntentID         string                   `json:"intent_id"`
	IntentType       string                   `json:"intent_type"`
	StartTime        time.Time                `json:"start_time"`
	EndTime          time.Time                `json:"end_time"`
	TotalLatency     time.Duration            `json:"total_latency"`
	Stages           map[string]*StageLatency `json:"stages"`
	UserPerceived    time.Duration            `json:"user_perceived_latency"`
	SLACompliant     bool                     `json:"sla_compliant"`
	TraceContext     trace.SpanContext        `json:"trace_context"`
	Metadata         map[string]interface{}   `json:"metadata"`
	BudgetViolations []BudgetViolation        `json:"budget_violations"`
	AnomalyScore     float64                  `json:"anomaly_score"`
}

// StageLatency represents latency for a specific processing stage
type StageLatency struct {
	Name       string         `json:"name"`
	StartTime  time.Time      `json:"start_time"`
	EndTime    time.Time      `json:"end_time"`
	Duration   time.Duration  `json:"duration"`
	BudgetUsed float64        `json:"budget_used"`
	SubStages  []StageLatency `json:"sub_stages,omitempty"`
}

// StreamingPercentiles calculates percentiles using streaming algorithms
type StreamingPercentiles struct {
	mu sync.RWMutex

	// T-Digest for accurate percentile estimation
	tdigest *TDigest

	// Running statistics
	count int64
	sum   int64
	sumSq int64
	min   int64
	max   int64

	// Percentile cache
	percentileCache map[float64]time.Duration
	cacheTime       time.Time
	cacheTTL        time.Duration
}

// LatencyHistogram provides fine-grained latency distribution tracking
type LatencyHistogram struct {
	mu        sync.RWMutex
	buckets   []HistogramBucket
	overflow  int64
	underflow int64
	total     int64
}

// HistogramBucket represents a single histogram bucket
type HistogramBucket struct {
	UpperBound time.Duration `json:"upper_bound"`
	Count      int64         `json:"count"`
}

// SLAComplianceTracker tracks SLA compliance over time
type SLAComplianceTracker struct {
	mu sync.RWMutex

	// Time windows for SLA tracking
	windows map[time.Duration]*ComplianceWindow

	// Real-time compliance status
	currentCompliance float64

	// Historical compliance
	history []ComplianceSnapshot
}

// ComplianceWindow represents an SLA compliance time window
type ComplianceWindow struct {
	Duration       time.Duration
	TotalRequests  int64
	CompliantCount int64
	ViolationCount int64
	LastUpdated    time.Time
}

// LatencyAnomalyDetector detects latency anomalies and regressions
type LatencyAnomalyDetector struct {
	mu sync.RWMutex

	// Moving statistics
	movingAvg    *ExponentialMovingAverage
	movingStdDev *ExponentialMovingStdDev

	// Anomaly history
	anomalies []LatencyAnomalyEvent

	// Configuration
	sensitivity float64
	windowSize  time.Duration
}

// LatencyBudgetTracker tracks latency budgets for components
type LatencyBudgetTracker struct {
	mu sync.RWMutex

	// Component budgets (in milliseconds)
	budgets map[string]time.Duration

	// Current usage
	usage map[string]*BudgetUsage

	// Violations
	violations []BudgetViolation
}

// BudgetUsage tracks budget usage for a component
type BudgetUsage struct {
	Component      string        `json:"component"`
	Allocated      time.Duration `json:"allocated"`
	Used           time.Duration `json:"used"`
	Percentage     float64       `json:"percentage"`
	ViolationCount int64         `json:"violation_count"`
}

// BudgetViolation represents a budget violation event
type BudgetViolation struct {
	Component string        `json:"component"`
	Budget    time.Duration `json:"budget"`
	Actual    time.Duration `json:"actual"`
	Overage   time.Duration `json:"overage"`
	Timestamp time.Time     `json:"timestamp"`
	IntentID  string        `json:"intent_id"`
}

// UserExperienceCorrelator correlates technical latency with user experience
type UserExperienceCorrelator struct {
	mu sync.RWMutex

	// User session tracking
	sessions map[string]*MonitoringUserSession

	// Experience metrics
	experienceScores map[string]float64

	// Impact analysis
	impactAnalyzer *UserImpactAnalyzer
}

// E2EMetrics contains Prometheus metrics for E2E tracking
type E2EMetrics struct {
	intentLatency     *prometheus.HistogramVec
	percentileGauge   *prometheus.GaugeVec
	slaCompliance     prometheus.Gauge
	anomaliesDetected prometheus.Counter
	budgetViolations  *prometheus.CounterVec
	userExperience    *prometheus.GaugeVec
}

// E2EStatistics contains running statistics
type E2EStatistics struct {
	TotalIntents      atomic.Int64
	CompletedIntents  atomic.Int64
	FailedIntents     atomic.Int64
	P50Latency        atomic.Int64
	P95Latency        atomic.Int64
	P99Latency        atomic.Int64
	SLACompliance     atomic.Uint64 // Store as fixed-point percentage
	AnomaliesDetected atomic.Int64
	BudgetViolations  atomic.Int64
}

// NewE2ELatencyTracker creates a new end-to-end latency tracker
func NewE2ELatencyTracker(config *E2ETrackerConfig) *E2ELatencyTracker {
	if config == nil {
		config = DefaultE2EConfig()
	}

	tracker := &E2ELatencyTracker{
		activeIntents:    make(map[string]*IntentTrace),
		completedIntents: NewCircularBuffer(config.HistorySize),
		config:           config,
		stats:            &E2EStatistics{},
	}

	// Initialize components
	tracker.percentileCalc = NewStreamingPercentiles()
	tracker.latencyHistogram = NewLatencyHistogram(config.HistogramBuckets)
	tracker.slaTracker = NewSLAComplianceTracker(config)
	tracker.anomalyDetector = NewLatencyAnomalyDetector(config)
	tracker.budgetTracker = NewLatencyBudgetTracker(config)
	tracker.uxCorrelator = NewUserExperienceCorrelator()

	// Initialize metrics
	tracker.initMetrics()

	// Start background tasks
	go tracker.runPeriodicTasks()

	return tracker
}

// StartIntent begins tracking a new intent
func (t *E2ELatencyTracker) StartIntent(ctx context.Context, intentID, intentType string) context.Context {
	intentTrace := &IntentTrace{
		ID:         fmt.Sprintf("trace-%s-%d", intentID, time.Now().UnixNano()),
		IntentID:   intentID,
		IntentType: intentType,
		StartTime:  time.Now(),
		Stages:     make(map[string]*StageLatency),
		Metadata:   make(map[string]interface{}),
	}

	// Extract trace context if available
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		intentTrace.TraceContext = span.SpanContext()
	}

	// Store active trace
	t.mu.Lock()
	t.activeIntents[intentID] = intentTrace
	t.mu.Unlock()

	// Update statistics
	t.stats.TotalIntents.Add(1)

	// Store trace ID in context
	ctx = context.WithValue(ctx, "e2e_trace_id", intentTrace.ID)

	return ctx
}

// EndIntent completes tracking for an intent
func (t *E2ELatencyTracker) EndIntent(ctx context.Context, intentID string, success bool) *IntentTrace {
	t.mu.Lock()
	trace, exists := t.activeIntents[intentID]
	if exists {
		delete(t.activeIntents, intentID)
	}
	t.mu.Unlock()

	if !exists || trace == nil {
		return nil
	}

	// Complete the trace
	trace.EndTime = time.Now()
	trace.TotalLatency = trace.EndTime.Sub(trace.StartTime)

	// Calculate user-perceived latency (might differ from total)
	trace.UserPerceived = t.calculateUserPerceivedLatency(trace)

	// Check SLA compliance
	trace.SLACompliant = t.checkSLACompliance(trace)

	// Check budget violations
	trace.BudgetViolations = t.budgetTracker.CheckViolations(trace)

	// Calculate anomaly score
	trace.AnomalyScore = t.anomalyDetector.CalculateScore(trace.TotalLatency)

	// Update percentiles
	t.percentileCalc.Add(trace.TotalLatency)

	// Update histogram
	t.latencyHistogram.Add(trace.TotalLatency)

	// Update SLA tracker
	t.slaTracker.RecordIntent(trace)

	// Check for anomalies
	if trace.AnomalyScore > t.config.AnomalyThreshold {
		t.recordAnomaly(trace)
	}

	// Store completed trace
	t.completedIntents.Add(trace)

	// Update statistics
	if success {
		t.stats.CompletedIntents.Add(1)
	} else {
		t.stats.FailedIntents.Add(1)
	}

	// Update metrics
	t.updateMetrics(trace)

	return trace
}

// RecordStage records latency for a specific processing stage
func (t *E2ELatencyTracker) RecordStage(ctx context.Context, intentID, stageName string, duration time.Duration) {
	t.mu.Lock()
	trace, exists := t.activeIntents[intentID]
	t.mu.Unlock()

	if !exists || trace == nil {
		return
	}

	stage := &StageLatency{
		Name:      stageName,
		StartTime: time.Now().Add(-duration),
		EndTime:   time.Now(),
		Duration:  duration,
	}

	// Calculate budget usage
	if budget := t.budgetTracker.GetBudget(stageName); budget > 0 {
		stage.BudgetUsed = float64(duration) / float64(budget)
		t.budgetTracker.RecordUsage(stageName, duration)
	}

	t.mu.Lock()
	trace.Stages[stageName] = stage
	t.mu.Unlock()
}

// GetPercentiles returns current percentile values
func (t *E2ELatencyTracker) GetPercentiles() *PercentileReport {
	p50 := t.percentileCalc.GetPercentile(50)
	p95 := t.percentileCalc.GetPercentile(95)
	p99 := t.percentileCalc.GetPercentile(99)

	// Update statistics
	t.stats.P50Latency.Store(int64(p50))
	t.stats.P95Latency.Store(int64(p95))
	t.stats.P99Latency.Store(int64(p99))

	return &PercentileReport{
		P50:          p50,
		P95:          p95,
		P99:          p99,
		P999:         t.percentileCalc.GetPercentile(99.9),
		Mean:         t.percentileCalc.GetMean(),
		StdDev:       t.percentileCalc.GetStdDev(),
		Min:          t.percentileCalc.GetMin(),
		Max:          t.percentileCalc.GetMax(),
		Count:        t.percentileCalc.GetCount(),
		SLACompliant: t.checkPercentileSLA(p50, p95, p99),
	}
}

// GetHistogram returns the latency histogram
func (t *E2ELatencyTracker) GetHistogram() *HistogramReport {
	return t.latencyHistogram.GetReport()
}

// GetSLACompliance returns current SLA compliance metrics
func (t *E2ELatencyTracker) GetSLACompliance() *SLAReport {
	compliance := t.slaTracker.GetCompliance()

	// Update statistics
	t.stats.SLACompliance.Store(uint64(compliance.CurrentCompliance * 10000)) // Store as basis points

	return &SLAReport{
		CurrentCompliance:  compliance.CurrentCompliance,
		HourlyCompliance:   compliance.HourlyCompliance,
		DailyCompliance:    compliance.DailyCompliance,
		WeeklyCompliance:   compliance.WeeklyCompliance,
		ViolationCount:     compliance.ViolationCount,
		ConsecutiveSuccess: compliance.ConsecutiveSuccess,
		LastViolation:      compliance.LastViolation,
	}
}

// GetAnomalies returns detected anomalies
func (t *E2ELatencyTracker) GetAnomalies(since time.Time) []LatencyAnomalyEvent {
	return t.anomalyDetector.GetAnomalies(since)
}

// GetBudgetReport returns latency budget usage report
func (t *E2ELatencyTracker) GetBudgetReport() *BudgetReport {
	return t.budgetTracker.GetReport()
}

// GetUserExperienceReport returns user experience correlation report
func (t *E2ELatencyTracker) GetUserExperienceReport() *UserExperienceReport {
	return t.uxCorrelator.GetReport()
}

// Helper methods

func (t *E2ELatencyTracker) calculateUserPerceivedLatency(trace *IntentTrace) time.Duration {
	// User-perceived latency might be different from total latency
	// For example, if some operations happen asynchronously after initial response

	// Check if there's a specific "user_response" stage
	if stage, exists := trace.Stages["user_response"]; exists {
		return stage.Duration
	}

	// Otherwise, calculate based on critical path to first user feedback
	criticalLatency := time.Duration(0)
	for name, stage := range trace.Stages {
		// Consider stages that affect user perception
		if t.isUserFacingStage(name) {
			criticalLatency += stage.Duration
		}
	}

	if criticalLatency > 0 {
		return criticalLatency
	}

	return trace.TotalLatency
}

func (t *E2ELatencyTracker) isUserFacingStage(stageName string) bool {
	userFacingStages := []string{
		"controller_processing",
		"llm_processing",
		"initial_validation",
		"response_generation",
	}

	for _, stage := range userFacingStages {
		if stageName == stage {
			return true
		}
	}

	return false
}

func (t *E2ELatencyTracker) checkSLACompliance(trace *IntentTrace) bool {
	// Check against P95 target (primary SLA)
	return trace.TotalLatency <= t.config.P95Target
}

func (t *E2ELatencyTracker) checkPercentileSLA(p50, p95, p99 time.Duration) bool {
	return p50 <= t.config.P50Target &&
		p95 <= t.config.P95Target &&
		p99 <= t.config.P99Target
}

func (t *E2ELatencyTracker) recordAnomaly(trace *IntentTrace) {
	anomaly := LatencyAnomalyEvent{
		Timestamp:    trace.EndTime,
		IntentID:     trace.IntentID,
		Latency:      trace.TotalLatency,
		AnomalyScore: trace.AnomalyScore,
		Type:         t.classifyAnomaly(trace),
		Description:  t.generateAnomalyDescription(trace),
	}

	t.anomalyDetector.RecordAnomaly(anomaly)
	t.stats.AnomaliesDetected.Add(1)
}

func (t *E2ELatencyTracker) classifyAnomaly(trace *IntentTrace) string {
	if trace.TotalLatency > t.config.MaxAcceptableLatency {
		return "CRITICAL_LATENCY"
	}
	if trace.AnomalyScore > 3.0 {
		return "STATISTICAL_OUTLIER"
	}
	if len(trace.BudgetViolations) > 0 {
		return "BUDGET_VIOLATION"
	}
	return "PERFORMANCE_REGRESSION"
}

func (t *E2ELatencyTracker) generateAnomalyDescription(trace *IntentTrace) string {
	p95 := time.Duration(t.stats.P95Latency.Load())
	deviation := float64(trace.TotalLatency) / float64(p95)

	return fmt.Sprintf("Intent %s experienced %.2fx normal latency (%.2fs vs P95 %.2fs)",
		trace.IntentID,
		deviation,
		trace.TotalLatency.Seconds(),
		p95.Seconds())
}

func (t *E2ELatencyTracker) runPeriodicTasks() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Update cached percentiles
		t.percentileCalc.RefreshCache()

		// Clean old data
		t.cleanOldData()

		// Update SLA windows
		t.slaTracker.UpdateWindows()

		// Check for sustained anomalies
		t.anomalyDetector.CheckSustainedAnomalies()
	}
}

func (t *E2ELatencyTracker) cleanOldData() {
	cutoff := time.Now().Add(-t.config.RetentionDuration)

	// Clean completed intents
	t.completedIntents.RemoveOlderThan(cutoff)

	// Clean anomaly history
	t.anomalyDetector.CleanOldAnomalies(cutoff)

	// Clean budget violations
	t.budgetTracker.CleanOldViolations(cutoff)
}

func (t *E2ELatencyTracker) initMetrics() {
	t.metrics = &E2EMetrics{
		intentLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "nephoran_e2e_intent_latency_seconds",
			Help:    "End-to-end intent processing latency",
			Buckets: t.config.HistogramBuckets,
		}, []string{"intent_type", "status"}),

		percentileGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "nephoran_e2e_latency_percentile_seconds",
			Help: "Latency percentiles",
		}, []string{"percentile"}),

		slaCompliance: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "nephoran_e2e_sla_compliance_ratio",
			Help: "SLA compliance ratio",
		}),

		anomaliesDetected: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "nephoran_e2e_anomalies_detected_total",
			Help: "Total number of latency anomalies detected",
		}),

		budgetViolations: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "nephoran_e2e_budget_violations_total",
			Help: "Total number of latency budget violations",
		}, []string{"component"}),

		userExperience: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "nephoran_e2e_user_experience_score",
			Help: "User experience score based on latency",
		}, []string{"category"}),
	}
}

func (t *E2ELatencyTracker) updateMetrics(trace *IntentTrace) {
	// Update latency histogram
	status := "success"
	if !trace.SLACompliant {
		status = "sla_violation"
	}
	t.metrics.intentLatency.WithLabelValues(trace.IntentType, status).Observe(trace.TotalLatency.Seconds())

	// Update percentiles
	percentiles := t.GetPercentiles()
	t.metrics.percentileGauge.WithLabelValues("p50").Set(percentiles.P50.Seconds())
	t.metrics.percentileGauge.WithLabelValues("p95").Set(percentiles.P95.Seconds())
	t.metrics.percentileGauge.WithLabelValues("p99").Set(percentiles.P99.Seconds())

	// Update SLA compliance
	compliance := t.slaTracker.GetCompliance()
	t.metrics.slaCompliance.Set(compliance.CurrentCompliance)

	// Update budget violations
	for _, violation := range trace.BudgetViolations {
		t.metrics.budgetViolations.WithLabelValues(violation.Component).Inc()
		t.stats.BudgetViolations.Add(1)
	}
}

// Supporting types and helper implementations

type PercentileReport struct {
	P50          time.Duration `json:"p50"`
	P95          time.Duration `json:"p95"`
	P99          time.Duration `json:"p99"`
	P999         time.Duration `json:"p999"`
	Mean         time.Duration `json:"mean"`
	StdDev       time.Duration `json:"std_dev"`
	Min          time.Duration `json:"min"`
	Max          time.Duration `json:"max"`
	Count        int64         `json:"count"`
	SLACompliant bool          `json:"sla_compliant"`
}

type HistogramReport struct {
	Buckets      []HistogramBucket  `json:"buckets"`
	TotalCount   int64              `json:"total_count"`
	Underflow    int64              `json:"underflow"`
	Overflow     int64              `json:"overflow"`
	Distribution map[string]float64 `json:"distribution"`
}

type SLAReport struct {
	CurrentCompliance  float64   `json:"current_compliance"`
	HourlyCompliance   float64   `json:"hourly_compliance"`
	DailyCompliance    float64   `json:"daily_compliance"`
	WeeklyCompliance   float64   `json:"weekly_compliance"`
	ViolationCount     int64     `json:"violation_count"`
	ConsecutiveSuccess int64     `json:"consecutive_success"`
	LastViolation      time.Time `json:"last_violation"`
}

type BudgetReport struct {
	ComponentUsage  map[string]*BudgetUsage `json:"component_usage"`
	TotalViolations int64                   `json:"total_violations"`
	WorstOffender   string                  `json:"worst_offender"`
	BudgetHealth    float64                 `json:"budget_health"`
}

type UserExperienceReport struct {
	AverageScore     float64            `json:"average_score"`
	CategoryScores   map[string]float64 `json:"category_scores"`
	ImpactedSessions int64              `json:"impacted_sessions"`
	Recommendations  []string           `json:"recommendations"`
}

type LatencyAnomalyEvent struct {
	Timestamp    time.Time     `json:"timestamp"`
	IntentID     string        `json:"intent_id"`
	Latency      time.Duration `json:"latency"`
	AnomalyScore float64       `json:"anomaly_score"`
	Type         string        `json:"type"`
	Description  string        `json:"description"`
}

type MonitoringUserSession struct {
	SessionID       string        `json:"session_id"`
	UserID          string        `json:"user_id"`
	StartTime       time.Time     `json:"start_time"`
	IntentCount     int           `json:"intent_count"`
	AverageLatency  time.Duration `json:"average_latency"`
	ExperienceScore float64       `json:"experience_score"`
}

type UserImpactAnalyzer struct {
	sessions map[string]*MonitoringUserSession
}

type ComplianceSnapshot struct {
	Timestamp  time.Time `json:"timestamp"`
	Compliance float64   `json:"compliance"`
	Window     string    `json:"window"`
}

// TDigest implementation for accurate percentile calculation
type TDigest struct {
	centroids   []Centroid
	count       int64
	compression float64
}

type Centroid struct {
	mean  float64
	count int64
}

func NewStreamingPercentiles() *StreamingPercentiles {
	return &StreamingPercentiles{
		tdigest:         NewTDigest(100),
		percentileCache: make(map[float64]time.Duration),
		cacheTTL:        10 * time.Second,
	}
}

func (s *StreamingPercentiles) Add(latency time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	value := int64(latency)
	s.count++
	s.sum += value
	s.sumSq += value * value

	if s.count == 1 || value < s.min {
		s.min = value
	}
	if s.count == 1 || value > s.max {
		s.max = value
	}

	s.tdigest.Add(float64(value))

	// Invalidate cache
	s.percentileCache = make(map[float64]time.Duration)
}

func (s *StreamingPercentiles) GetPercentile(p float64) time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check cache
	if cached, exists := s.percentileCache[p]; exists && time.Since(s.cacheTime) < s.cacheTTL {
		return cached
	}

	if s.count == 0 {
		return 0
	}

	value := s.tdigest.Quantile(p / 100.0)
	return time.Duration(value)
}

func (s *StreamingPercentiles) GetMean() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.count == 0 {
		return 0
	}

	return time.Duration(s.sum / s.count)
}

func (s *StreamingPercentiles) GetStdDev() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.count == 0 {
		return 0
	}

	mean := float64(s.sum) / float64(s.count)
	variance := (float64(s.sumSq) / float64(s.count)) - (mean * mean)

	if variance < 0 {
		variance = 0
	}

	return time.Duration(math.Sqrt(variance))
}

func (s *StreamingPercentiles) GetMin() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Duration(s.min)
}

func (s *StreamingPercentiles) GetMax() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Duration(s.max)
}

func (s *StreamingPercentiles) GetCount() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.count
}

func (s *StreamingPercentiles) RefreshCache() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Pre-calculate common percentiles
	percentiles := []float64{50, 75, 90, 95, 99, 99.9}
	for _, p := range percentiles {
		value := s.tdigest.Quantile(p / 100.0)
		s.percentileCache[p] = time.Duration(value)
	}
	s.cacheTime = time.Now()
}

// Simplified TDigest implementation
func NewTDigest(compression float64) *TDigest {
	return &TDigest{
		centroids:   make([]Centroid, 0, int(compression)*2),
		compression: compression,
	}
}

func (t *TDigest) Add(value float64) {
	// Simplified implementation - in production, use a proper TDigest library
	t.centroids = append(t.centroids, Centroid{mean: value, count: 1})
	t.count++

	// Compress if needed
	if len(t.centroids) > int(t.compression*2) {
		t.compress()
	}
}

func (t *TDigest) Quantile(q float64) float64 {
	if t.count == 0 {
		return 0
	}

	// Simplified quantile calculation
	// In production, use proper TDigest quantile algorithm
	targetCount := int64(float64(t.count) * q)
	var cumCount int64

	for _, c := range t.centroids {
		cumCount += c.count
		if cumCount >= targetCount {
			return c.mean
		}
	}

	return t.centroids[len(t.centroids)-1].mean
}

func (t *TDigest) compress() {
	// Simplified compression - merge adjacent centroids
	// In production, use proper TDigest compression algorithm
	if len(t.centroids) <= int(t.compression) {
		return
	}

	// Sort centroids by mean
	for i := 0; i < len(t.centroids); i++ {
		for j := i + 1; j < len(t.centroids); j++ {
			if t.centroids[j].mean < t.centroids[i].mean {
				t.centroids[i], t.centroids[j] = t.centroids[j], t.centroids[i]
			}
		}
	}

	// Merge adjacent centroids
	merged := make([]Centroid, 0, int(t.compression))
	for i := 0; i < len(t.centroids); i += 2 {
		if i+1 < len(t.centroids) {
			totalCount := t.centroids[i].count + t.centroids[i+1].count
			weightedMean := (t.centroids[i].mean*float64(t.centroids[i].count) +
				t.centroids[i+1].mean*float64(t.centroids[i+1].count)) / float64(totalCount)
			merged = append(merged, Centroid{mean: weightedMean, count: totalCount})
		} else {
			merged = append(merged, t.centroids[i])
		}
	}

	t.centroids = merged
}

// Helper implementations for other components

func NewLatencyHistogram(buckets []float64) *LatencyHistogram {
	if len(buckets) == 0 {
		buckets = DefaultHistogramBuckets()
	}

	hist := &LatencyHistogram{
		buckets: make([]HistogramBucket, len(buckets)),
	}

	for i, bound := range buckets {
		hist.buckets[i] = HistogramBucket{
			UpperBound: time.Duration(bound * float64(time.Second)),
			Count:      0,
		}
	}

	return hist
}

func (h *LatencyHistogram) Add(latency time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.total++

	// Find appropriate bucket
	for i := range h.buckets {
		if latency <= h.buckets[i].UpperBound {
			h.buckets[i].Count++
			return
		}
	}

	// Overflow
	h.overflow++
}

func (h *LatencyHistogram) GetReport() *HistogramReport {
	h.mu.RLock()
	defer h.mu.RUnlock()

	report := &HistogramReport{
		Buckets:      make([]HistogramBucket, len(h.buckets)),
		TotalCount:   h.total,
		Underflow:    h.underflow,
		Overflow:     h.overflow,
		Distribution: make(map[string]float64),
	}

	copy(report.Buckets, h.buckets)

	// Calculate distribution percentages
	if h.total > 0 {
		for _, bucket := range h.buckets {
			key := fmt.Sprintf("%.3fs", bucket.UpperBound.Seconds())
			report.Distribution[key] = float64(bucket.Count) / float64(h.total) * 100
		}
	}

	return report
}

func NewSLAComplianceTracker(config *E2ETrackerConfig) *SLAComplianceTracker {
	tracker := &SLAComplianceTracker{
		windows: make(map[time.Duration]*ComplianceWindow),
		history: make([]ComplianceSnapshot, 0, 1000),
	}

	// Initialize windows
	windows := []time.Duration{
		1 * time.Hour,
		24 * time.Hour,
		7 * 24 * time.Hour,
	}

	for _, duration := range windows {
		tracker.windows[duration] = &ComplianceWindow{
			Duration:    duration,
			LastUpdated: time.Now(),
		}
	}

	return tracker
}

func (s *SLAComplianceTracker) RecordIntent(trace *IntentTrace) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, window := range s.windows {
		window.TotalRequests++
		if trace.SLACompliant {
			window.CompliantCount++
		} else {
			window.ViolationCount++
		}
	}

	// Update current compliance
	if s.windows[1*time.Hour].TotalRequests > 0 {
		s.currentCompliance = float64(s.windows[1*time.Hour].CompliantCount) /
			float64(s.windows[1*time.Hour].TotalRequests)
	}

	// Add to history
	s.history = append(s.history, ComplianceSnapshot{
		Timestamp:  time.Now(),
		Compliance: s.currentCompliance,
		Window:     "1h",
	})

	// Trim history
	if len(s.history) > 1000 {
		s.history = s.history[len(s.history)-1000:]
	}
}

func (s *SLAComplianceTracker) GetCompliance() *SLACompliance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	compliance := &SLACompliance{
		CurrentCompliance: s.currentCompliance,
	}

	if window, exists := s.windows[1*time.Hour]; exists && window.TotalRequests > 0 {
		compliance.HourlyCompliance = float64(window.CompliantCount) / float64(window.TotalRequests)
	}

	if window, exists := s.windows[24*time.Hour]; exists && window.TotalRequests > 0 {
		compliance.DailyCompliance = float64(window.CompliantCount) / float64(window.TotalRequests)
	}

	if window, exists := s.windows[7*24*time.Hour]; exists && window.TotalRequests > 0 {
		compliance.WeeklyCompliance = float64(window.CompliantCount) / float64(window.TotalRequests)
	}

	// Count violations and consecutive success
	for _, window := range s.windows {
		compliance.ViolationCount += window.ViolationCount
	}

	// Find consecutive success
	for i := len(s.history) - 1; i >= 0; i-- {
		if s.history[i].Compliance < 1.0 {
			compliance.LastViolation = s.history[i].Timestamp
			break
		}
		compliance.ConsecutiveSuccess++
	}

	return compliance
}

func (s *SLAComplianceTracker) UpdateWindows() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	for duration, window := range s.windows {
		if now.Sub(window.LastUpdated) > duration {
			// Reset window
			window.TotalRequests = 0
			window.CompliantCount = 0
			window.ViolationCount = 0
			window.LastUpdated = now
		}
	}
}

type SLACompliance struct {
	CurrentCompliance  float64
	HourlyCompliance   float64
	DailyCompliance    float64
	WeeklyCompliance   float64
	ViolationCount     int64
	ConsecutiveSuccess int64
	LastViolation      time.Time
}

func NewLatencyAnomalyDetector(config *E2ETrackerConfig) *LatencyAnomalyDetector {
	return &LatencyAnomalyDetector{
		movingAvg:    NewExponentialMovingAverage(0.1),
		movingStdDev: NewExponentialMovingStdDev(0.1),
		anomalies:    make([]LatencyAnomalyEvent, 0, 100),
		sensitivity:  config.AnomalyThreshold,
		windowSize:   config.AnomalyWindowSize,
	}
}

func (a *LatencyAnomalyDetector) CalculateScore(latency time.Duration) float64 {
	a.mu.Lock()
	defer a.mu.Unlock()

	value := float64(latency)
	avg := a.movingAvg.Value()
	stdDev := a.movingStdDev.Value()

	// Update moving statistics
	a.movingAvg.Add(value)
	a.movingStdDev.Add(value)

	if stdDev == 0 {
		return 0
	}

	// Calculate z-score
	zScore := math.Abs((value - avg) / stdDev)

	return zScore
}

func (a *LatencyAnomalyDetector) RecordAnomaly(anomaly LatencyAnomalyEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.anomalies = append(a.anomalies, anomaly)

	// Keep only last 100 anomalies
	if len(a.anomalies) > 100 {
		a.anomalies = a.anomalies[len(a.anomalies)-100:]
	}
}

func (a *LatencyAnomalyDetector) GetAnomalies(since time.Time) []LatencyAnomalyEvent {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var result []LatencyAnomalyEvent
	for _, anomaly := range a.anomalies {
		if anomaly.Timestamp.After(since) {
			result = append(result, anomaly)
		}
	}

	return result
}

func (a *LatencyAnomalyDetector) CheckSustainedAnomalies() {
	// Check if we're seeing sustained high latencies
	// This would trigger alerts for persistent performance degradation
}

func (a *LatencyAnomalyDetector) CleanOldAnomalies(cutoff time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()

	var kept []LatencyAnomalyEvent
	for _, anomaly := range a.anomalies {
		if anomaly.Timestamp.After(cutoff) {
			kept = append(kept, anomaly)
		}
	}

	a.anomalies = kept
}

type ExponentialMovingAverage struct {
	alpha float64
	value float64
	count int64
}

func NewExponentialMovingAverage(alpha float64) *ExponentialMovingAverage {
	return &ExponentialMovingAverage{alpha: alpha}
}

func (e *ExponentialMovingAverage) Add(value float64) {
	if e.count == 0 {
		e.value = value
	} else {
		e.value = e.alpha*value + (1-e.alpha)*e.value
	}
	e.count++
}

func (e *ExponentialMovingAverage) Value() float64 {
	return e.value
}

type ExponentialMovingStdDev struct {
	alpha    float64
	mean     float64
	variance float64
	count    int64
}

func NewExponentialMovingStdDev(alpha float64) *ExponentialMovingStdDev {
	return &ExponentialMovingStdDev{alpha: alpha}
}

func (e *ExponentialMovingStdDev) Add(value float64) {
	if e.count == 0 {
		e.mean = value
		e.variance = 0
	} else {
		delta := value - e.mean
		e.mean = e.mean + e.alpha*delta
		e.variance = (1 - e.alpha) * (e.variance + e.alpha*delta*delta)
	}
	e.count++
}

func (e *ExponentialMovingStdDev) Value() float64 {
	return math.Sqrt(e.variance)
}

func NewLatencyBudgetTracker(config *E2ETrackerConfig) *LatencyBudgetTracker {
	tracker := &LatencyBudgetTracker{
		budgets:    make(map[string]time.Duration),
		usage:      make(map[string]*BudgetUsage),
		violations: make([]BudgetViolation, 0, 100),
	}

	// Set up budgets based on config (assuming 2-second total budget)
	totalBudget := 2 * time.Second

	tracker.budgets["controller"] = time.Duration(float64(totalBudget) * config.ControllerBudget)
	tracker.budgets["llm_processor"] = time.Duration(float64(totalBudget) * config.LLMBudget)
	tracker.budgets["rag_system"] = time.Duration(float64(totalBudget) * config.RAGBudget)
	tracker.budgets["gitops"] = time.Duration(float64(totalBudget) * config.GitOpsBudget)
	tracker.budgets["deployment"] = time.Duration(float64(totalBudget) * config.DeploymentBudget)

	// Initialize usage
	for component, budget := range tracker.budgets {
		tracker.usage[component] = &BudgetUsage{
			Component: component,
			Allocated: budget,
		}
	}

	return tracker
}

func (b *LatencyBudgetTracker) GetBudget(component string) time.Duration {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.budgets[component]
}

func (b *LatencyBudgetTracker) RecordUsage(component string, duration time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if usage, exists := b.usage[component]; exists {
		usage.Used = duration
		usage.Percentage = float64(duration) / float64(usage.Allocated) * 100

		if duration > usage.Allocated {
			usage.ViolationCount++
		}
	}
}

func (b *LatencyBudgetTracker) CheckViolations(trace *IntentTrace) []BudgetViolation {
	b.mu.Lock()
	defer b.mu.Unlock()

	var violations []BudgetViolation

	for component, stage := range trace.Stages {
		if budget, exists := b.budgets[component]; exists {
			if stage.Duration > budget {
				violation := BudgetViolation{
					Component: component,
					Budget:    budget,
					Actual:    stage.Duration,
					Overage:   stage.Duration - budget,
					Timestamp: trace.EndTime,
					IntentID:  trace.IntentID,
				}
				violations = append(violations, violation)
				b.violations = append(b.violations, violation)
			}
		}
	}

	// Trim violations history
	if len(b.violations) > 100 {
		b.violations = b.violations[len(b.violations)-100:]
	}

	return violations
}

func (b *LatencyBudgetTracker) GetReport() *BudgetReport {
	b.mu.RLock()
	defer b.mu.RUnlock()

	report := &BudgetReport{
		ComponentUsage:  make(map[string]*BudgetUsage),
		TotalViolations: int64(len(b.violations)),
	}

	// Copy usage data
	for component, usage := range b.usage {
		report.ComponentUsage[component] = &BudgetUsage{
			Component:      usage.Component,
			Allocated:      usage.Allocated,
			Used:           usage.Used,
			Percentage:     usage.Percentage,
			ViolationCount: usage.ViolationCount,
		}

		// Find worst offender
		if usage.ViolationCount > 0 && (report.WorstOffender == "" ||
			usage.ViolationCount > report.ComponentUsage[report.WorstOffender].ViolationCount) {
			report.WorstOffender = component
		}
	}

	// Calculate budget health (percentage of components within budget)
	componentsWithinBudget := 0
	for _, usage := range b.usage {
		if usage.Percentage <= 100 {
			componentsWithinBudget++
		}
	}

	if len(b.usage) > 0 {
		report.BudgetHealth = float64(componentsWithinBudget) / float64(len(b.usage))
	}

	return report
}

func (b *LatencyBudgetTracker) CleanOldViolations(cutoff time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()

	var kept []BudgetViolation
	for _, violation := range b.violations {
		if violation.Timestamp.After(cutoff) {
			kept = append(kept, violation)
		}
	}

	b.violations = kept
}

func NewUserExperienceCorrelator() *UserExperienceCorrelator {
	return &UserExperienceCorrelator{
		sessions:         make(map[string]*MonitoringUserSession),
		experienceScores: make(map[string]float64),
		impactAnalyzer:   &UserImpactAnalyzer{sessions: make(map[string]*MonitoringUserSession)},
	}
}

func (u *UserExperienceCorrelator) GetReport() *UserExperienceReport {
	u.mu.RLock()
	defer u.mu.RUnlock()

	report := &UserExperienceReport{
		CategoryScores:  make(map[string]float64),
		Recommendations: []string{},
	}

	// Calculate average experience score
	totalScore := 0.0
	count := 0
	for _, score := range u.experienceScores {
		totalScore += score
		count++
	}

	if count > 0 {
		report.AverageScore = totalScore / float64(count)
	}

	// Categorize experience
	report.CategoryScores["excellent"] = u.calculateCategoryScore(90, 100)
	report.CategoryScores["good"] = u.calculateCategoryScore(70, 90)
	report.CategoryScores["fair"] = u.calculateCategoryScore(50, 70)
	report.CategoryScores["poor"] = u.calculateCategoryScore(0, 50)

	// Count impacted sessions
	for _, session := range u.sessions {
		if session.ExperienceScore < 70 {
			report.ImpactedSessions++
		}
	}

	// Generate recommendations
	if report.AverageScore < 70 {
		report.Recommendations = append(report.Recommendations,
			"Consider implementing caching to reduce latency",
			"Optimize LLM processing with batching",
			"Review database query performance")
	}

	return report
}

func (u *UserExperienceCorrelator) calculateCategoryScore(min, max float64) float64 {
	count := 0
	for _, score := range u.experienceScores {
		if score >= min && score < max {
			count++
		}
	}

	if len(u.experienceScores) == 0 {
		return 0
	}

	return float64(count) / float64(len(u.experienceScores)) * 100
}

// CircularBuffer for storing completed intents
type CircularBuffer struct {
	mu       sync.RWMutex
	items    []*IntentTrace
	capacity int
	head     int
	tail     int
	size     int
}

func NewCircularBuffer(capacity int) *CircularBuffer {
	return &CircularBuffer{
		items:    make([]*IntentTrace, capacity),
		capacity: capacity,
	}
}

func (c *CircularBuffer) Add(trace *IntentTrace) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[c.tail] = trace
	c.tail = (c.tail + 1) % c.capacity

	if c.size < c.capacity {
		c.size++
	} else {
		c.head = (c.head + 1) % c.capacity
	}
}

func (c *CircularBuffer) RemoveOlderThan(cutoff time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// This is a simplified implementation
	// In production, would implement proper removal
}

// Default configuration
func DefaultE2EConfig() *E2ETrackerConfig {
	return &E2ETrackerConfig{
		P50Target:            500 * time.Millisecond,
		P95Target:            2 * time.Second,
		P99Target:            5 * time.Second,
		MaxAcceptableLatency: 10 * time.Second,
		HistogramBuckets:     DefaultHistogramBuckets(),
		HistogramPrecision:   3,
		AnomalyWindowSize:    5 * time.Minute,
		AnomalyThreshold:     2.5,
		ControllerBudget:     0.10, // 10% of 2 seconds = 200ms
		LLMBudget:            0.40, // 40% of 2 seconds = 800ms
		RAGBudget:            0.20, // 20% of 2 seconds = 400ms
		GitOpsBudget:         0.20, // 20% of 2 seconds = 400ms
		DeploymentBudget:     0.10, // 10% of 2 seconds = 200ms
		HistorySize:          10000,
		RetentionDuration:    24 * time.Hour,
	}
}

func DefaultHistogramBuckets() []float64 {
	return []float64{
		0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 0.75,
		1.0, 1.5, 2.0, 2.5, 3.0, 4.0, 5.0, 7.5, 10.0,
	}
}
