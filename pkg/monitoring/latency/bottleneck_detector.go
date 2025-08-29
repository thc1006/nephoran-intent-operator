package latency

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// BottleneckDetector automatically identifies and analyzes performance bottlenecks.

type BottleneckDetector struct {
	mu sync.RWMutex

	// Detection engines.

	criticalPathAnalyzer *CriticalPathAnalyzer

	resourceBottleneckFinder *ResourceBottleneckFinder

	contentionDetector *ContentionPointDetector

	cascadeAnalyzer *CascadeDelayAnalyzer

	rootCauseAnalyzer *RootCauseAnalyzer

	// Remediation system.

	autoRemediator *AutomaticRemediator

	// Detection results.

	detectedBottlenecks []DetectedBottleneck

	activeBottlenecks map[string]*ActiveBottleneck

	historicalBottlenecks *BottleneckHistory

	// Metrics and monitoring.

	detectionMetrics *DetectionMetrics

	performanceImpact *PerformanceImpactTracker

	// Configuration.

	config *BottleneckDetectorConfig
}

// BottleneckDetectorConfig contains configuration for bottleneck detection.

type BottleneckDetectorConfig struct {

	// Detection thresholds.

	LatencyThreshold time.Duration `json:"latency_threshold"`

	ResourceThreshold ResourceThresholds `json:"resource_threshold"`

	ContentionThreshold ContentionThresholds `json:"contention_threshold"`

	// Analysis settings.

	CriticalPathDepth int `json:"critical_path_depth"`

	AnalysisWindow time.Duration `json:"analysis_window"`

	MinSamplesForDetection int `json:"min_samples_for_detection"`

	// Auto-remediation.

	EnableAutoRemediation bool `json:"enable_auto_remediation"`

	RemediationPolicy RemediationPolicy `json:"remediation_policy"`

	// Alerting.

	AlertingEnabled bool `json:"alerting_enabled"`

	AlertThresholds AlertThresholds `json:"alert_thresholds"`
}

// ResourceThresholds defines thresholds for resource bottlenecks.

type ResourceThresholds struct {
	CPUUtilization float64 `json:"cpu_utilization"`

	MemoryUtilization float64 `json:"memory_utilization"`

	DiskIOUtilization float64 `json:"disk_io_utilization"`

	NetworkUtilization float64 `json:"network_utilization"`

	ConnectionPoolUsage float64 `json:"connection_pool_usage"`
}

// ContentionThresholds defines thresholds for contention detection.

type ContentionThresholds struct {
	LockWaitTime time.Duration `json:"lock_wait_time"`

	QueueDepth int `json:"queue_depth"`

	ThreadContention float64 `json:"thread_contention"`

	DatabaseLockTime time.Duration `json:"database_lock_time"`
}

// RemediationPolicy defines how automatic remediation should work.

type RemediationPolicy struct {
	MaxScaleOutInstances int `json:"max_scale_out_instances"`

	MinScaleInInstances int `json:"min_scale_in_instances"`

	ScaleStepSize int `json:"scale_step_size"`

	CooldownPeriod time.Duration `json:"cooldown_period"`

	AllowCacheInvalidation bool `json:"allow_cache_invalidation"`

	AllowCircuitBreaker bool `json:"allow_circuit_breaker"`
}

// AlertThresholds defines when to trigger alerts.

type AlertThresholds struct {
	CriticalBottleneckCount int `json:"critical_bottleneck_count"`

	SustainedDuration time.Duration `json:"sustained_duration"`

	ImpactThreshold float64 `json:"impact_threshold"`
}

// DetectedBottleneck represents a detected performance bottleneck.

type DetectedBottleneck struct {
	ID string `json:"id"`

	Type BottleneckType `json:"type"`

	Component string `json:"component"`

	Severity BottleneckSeverity `json:"severity"`

	DetectedAt time.Time `json:"detected_at"`

	Duration time.Duration `json:"duration"`

	Impact float64 `json:"impact"`

	Description string `json:"description"`

	Evidence []BottleneckEvidence `json:"evidence"`

	RootCause *RootCause `json:"root_cause"`

	Recommendations []RemediationAction `json:"recommendations"`

	AutoRemediation *AutoRemediationResult `json:"auto_remediation,omitempty"`
}

// BottleneckType categorizes the type of bottleneck.

type BottleneckType string

const (

	// CPUBottleneck holds cpubottleneck value.

	CPUBottleneck BottleneckType = "CPU"

	// MemoryBottleneck holds memorybottleneck value.

	MemoryBottleneck BottleneckType = "MEMORY"

	// IOBottleneck holds iobottleneck value.

	IOBottleneck BottleneckType = "IO"

	// NetworkBottleneck holds networkbottleneck value.

	NetworkBottleneck BottleneckType = "NETWORK"

	// DatabaseBottleneck holds databasebottleneck value.

	DatabaseBottleneck BottleneckType = "DATABASE"

	// ConcurrencyBottleneck holds concurrencybottleneck value.

	ConcurrencyBottleneck BottleneckType = "CONCURRENCY"

	// QueueBottleneck holds queuebottleneck value.

	QueueBottleneck BottleneckType = "QUEUE"

	// CacheBottleneck holds cachebottleneck value.

	CacheBottleneck BottleneckType = "CACHE"

	// AlgorithmicBottleneck holds algorithmicbottleneck value.

	AlgorithmicBottleneck BottleneckType = "ALGORITHMIC"
)

// BottleneckSeverity indicates the severity of a bottleneck.

type BottleneckSeverity string

const (

	// CriticalSeverity holds criticalseverity value.

	CriticalSeverity BottleneckSeverity = "CRITICAL"

	// HighSeverity holds highseverity value.

	HighSeverity BottleneckSeverity = "HIGH"

	// MediumSeverity holds mediumseverity value.

	MediumSeverity BottleneckSeverity = "MEDIUM"

	// LowSeverity holds lowseverity value.

	LowSeverity BottleneckSeverity = "LOW"
)

// BottleneckEvidence provides evidence for a detected bottleneck.

type BottleneckEvidence struct {
	Metric string `json:"metric"`

	Value interface{} `json:"value"`

	Threshold interface{} `json:"threshold"`

	Deviation float64 `json:"deviation"`

	Timestamp time.Time `json:"timestamp"`
}

// ActiveBottleneck represents a currently active bottleneck.

type ActiveBottleneck struct {
	Bottleneck DetectedBottleneck `json:"bottleneck"`

	StartTime time.Time `json:"start_time"`

	LastSeen time.Time `json:"last_seen"`

	OccurrenceCount int `json:"occurrence_count"`

	CumulativeImpact float64 `json:"cumulative_impact"`

	Status string `json:"status"` // "active", "mitigating", "resolved"

}

// CriticalPathAnalyzer identifies the critical path in processing.

type CriticalPathAnalyzer struct {
	mu sync.RWMutex

	// Dependency graph.

	dependencyGraph *DependencyGraph

	// Path analysis.

	criticalPaths map[string]*CriticalPath

	pathLatencies map[string]time.Duration

	// Statistics.

	pathStatistics *PathStatistics
}

// CriticalPath represents the longest path through the system.

type CriticalPath struct {
	ID string `json:"id"`

	Components []PathComponent `json:"components"`

	TotalLatency time.Duration `json:"total_latency"`

	BottleneckNode string `json:"bottleneck_node"`

	OptimizationPotential time.Duration `json:"optimization_potential"`
}

// PathComponent represents a component in the critical path.

type PathComponent struct {
	Name string `json:"name"`

	Latency time.Duration `json:"latency"`

	Percentage float64 `json:"percentage"`

	Dependencies []string `json:"dependencies"`

	Parallelizable bool `json:"parallelizable"`
}

// ResourceBottleneckFinder identifies resource-related bottlenecks.

type ResourceBottleneckFinder struct {
	mu sync.RWMutex

	// Resource monitors.

	cpuMonitor *CPUMonitor

	memoryMonitor *MemoryMonitor

	diskMonitor *DiskIOMonitor

	networkMonitor *NetworkMonitor

	// Resource bottlenecks.

	bottlenecks []ResourceBottleneck

	// Thresholds.

	thresholds ResourceThresholds
}

// ResourceBottleneck represents a resource-related bottleneck.

type ResourceBottleneck struct {
	Resource string `json:"resource"`

	Utilization float64 `json:"utilization"`

	Threshold float64 `json:"threshold"`

	Duration time.Duration `json:"duration"`

	Impact string `json:"impact"`

	AffectedOps []string `json:"affected_operations"`
}

// ContentionPointDetector identifies contention points in the system.

type ContentionPointDetector struct {
	mu sync.RWMutex

	// Contention monitors.

	lockMonitor *LockContentionMonitor

	poolMonitor *ConnectionPoolMonitor

	queueMonitor *QueueContentionMonitor

	// Detected contention.

	contentionPoints []ContentionPoint

	// Thresholds.

	thresholds ContentionThresholds
}

// ContentionPoint represents a point of contention in the system.

type ContentionPoint struct {
	Type string `json:"type"` // "lock", "pool", "queue"

	Location string `json:"location"`

	WaitTime time.Duration `json:"wait_time"`

	WaitingCount int `json:"waiting_count"`

	HoldTime time.Duration `json:"hold_time"`

	Impact float64 `json:"impact"`
}

// CascadeDelayAnalyzer analyzes how delays cascade through the system.

type CascadeDelayAnalyzer struct {
	mu sync.RWMutex

	// Cascade tracking.

	cascades map[string]*DelayCascade

	impactMap map[string][]string

	// Analysis results.

	cascadeEffects []CascadeEffect
}

// DelayCascade represents how a delay cascades through components.

type DelayCascade struct {
	OriginComponent string `json:"origin_component"`

	OriginDelay time.Duration `json:"origin_delay"`

	AffectedComponents []AffectedComponent `json:"affected_components"`

	TotalImpact time.Duration `json:"total_impact"`

	AmplificationFactor float64 `json:"amplification_factor"`
}

// AffectedComponent represents a component affected by cascade delay.

type AffectedComponent struct {
	Name string `json:"name"`

	DirectDelay time.Duration `json:"direct_delay"`

	IndirectDelay time.Duration `json:"indirect_delay"`

	PropagationPath []string `json:"propagation_path"`
}

// CascadeEffect represents the effect of a cascade.

type CascadeEffect struct {
	Source string `json:"source"`

	Targets []string `json:"targets"`

	DelayMultiplier float64 `json:"delay_multiplier"`

	CriticalPath bool `json:"critical_path"`
}

// RootCauseAnalyzer performs root cause analysis on bottlenecks.

type RootCauseAnalyzer struct {
	mu sync.RWMutex

	// Analysis engines.

	changeAnalyzer *ChangeCorrelationAnalyzer

	patternMatcher *PatternMatcher

	anomalyCorrelator *AnomalyCorrelator

	// Root causes.

	identifiedCauses map[string]*RootCause
}

// RootCause represents the root cause of a bottleneck.

type RootCause struct {
	ID string `json:"id"`

	Type string `json:"type"`

	Description string `json:"description"`

	Confidence float64 `json:"confidence"`

	Evidence []CausalEvidence `json:"evidence"`

	Timeline []TimelineEvent `json:"timeline"`

	RelatedChanges []SystemChange `json:"related_changes"`
}

// CausalEvidence provides evidence for a root cause.

type CausalEvidence struct {
	Type string `json:"type"`

	Description string `json:"description"`

	Correlation float64 `json:"correlation"`

	Timestamp time.Time `json:"timestamp"`
}

// TimelineEvent represents an event in the root cause timeline.

type TimelineEvent struct {
	Timestamp time.Time `json:"timestamp"`

	Event string `json:"event"`

	Component string `json:"component"`

	Impact string `json:"impact"`
}

// SystemChange represents a change in the system.

type SystemChange struct {
	Timestamp time.Time `json:"timestamp"`

	Type string `json:"type"` // "deployment", "config", "scale"

	Component string `json:"component"`

	Description string `json:"description"`

	CorrelationScore float64 `json:"correlation_score"`
}

// AutomaticRemediator handles automatic remediation of bottlenecks.

type AutomaticRemediator struct {
	mu sync.RWMutex

	// Remediation strategies.

	strategies map[BottleneckType][]RemediationStrategy

	// Remediation history.

	history []RemediationAction

	// Active remediations.

	activeRemediations map[string]*ActiveRemediation

	// Policy.

	policy RemediationPolicy
}

// RemediationStrategy defines a strategy for remediating a bottleneck.

type RemediationStrategy struct {
	Name string `json:"name"`

	Type string `json:"type"`

	Applicability func(*DetectedBottleneck) bool

	Execute func(context.Context, *DetectedBottleneck) error

	Rollback func(context.Context) error

	ExpectedImprovement float64 `json:"expected_improvement"`

	Risk string `json:"risk"` // "low", "medium", "high"

}

// RemediationAction represents a remediation action.

type RemediationAction struct {
	ID string `json:"id"`

	BottleneckID string `json:"bottleneck_id"`

	Strategy string `json:"strategy"`

	ExecutedAt time.Time `json:"executed_at"`

	Status string `json:"status"` // "pending", "executing", "success", "failed"

	Result string `json:"result"`

	ImprovementMeasured float64 `json:"improvement_measured"`
}

// ActiveRemediation represents an ongoing remediation.

type ActiveRemediation struct {
	Action RemediationAction `json:"action"`

	StartTime time.Time `json:"start_time"`

	Context context.Context

	CancelFunc context.CancelFunc
}

// AutoRemediationResult represents the result of automatic remediation.

type AutoRemediationResult struct {
	Success bool `json:"success"`

	ActionsTaken []string `json:"actions_taken"`

	ImprovementPct float64 `json:"improvement_pct"`

	Message string `json:"message"`
}

// BottleneckHistory maintains historical bottleneck data.

type BottleneckHistory struct {
	mu sync.RWMutex

	// Historical data.

	bottlenecks []HistoricalBottleneck

	// Patterns.

	recurringPatterns []RecurringPattern

	// Statistics.

	statistics *HistoricalStatistics
}

// HistoricalBottleneck represents a historical bottleneck.

type HistoricalBottleneck struct {
	Bottleneck DetectedBottleneck `json:"bottleneck"`

	Resolution string `json:"resolution"`

	ResolutionTime time.Duration `json:"resolution_time"`

	Recurrence int `json:"recurrence"`
}

// RecurringPattern represents a recurring bottleneck pattern.

type RecurringPattern struct {
	Pattern string `json:"pattern"`

	Frequency string `json:"frequency"` // "hourly", "daily", "weekly"

	Components []string `json:"components"`

	TriggerConditions []string `json:"trigger_conditions"`

	PreventiveAction string `json:"preventive_action"`
}

// DetectionMetrics tracks bottleneck detection metrics.

type DetectionMetrics struct {
	bottlenecksDetected atomic.Int64

	criticalBottlenecks atomic.Int64

	autoRemediations atomic.Int64

	successfulRemediations atomic.Int64

	falsePositives atomic.Int64

	detectionLatency atomic.Int64

	averageResolutionTime atomic.Int64
}

// PerformanceImpactTracker tracks the performance impact of bottlenecks.

type PerformanceImpactTracker struct {
	mu sync.RWMutex

	// Impact measurements.

	impactHistory []ImpactMeasurement

	// Current impact.

	currentImpact float64

	// Cumulative impact.

	cumulativeImpact float64
}

// ImpactMeasurement represents a performance impact measurement.

type ImpactMeasurement struct {
	Timestamp time.Time `json:"timestamp"`

	BottleneckID string `json:"bottleneck_id"`

	Component string `json:"component"`

	LatencyImpact time.Duration `json:"latency_impact"`

	ThroughputImpact float64 `json:"throughput_impact"`

	ErrorRateImpact float64 `json:"error_rate_impact"`

	UserImpact int `json:"user_impact"`
}

// NewBottleneckDetector creates a new bottleneck detector.

func NewBottleneckDetector(config *BottleneckDetectorConfig) *BottleneckDetector {

	if config == nil {

		config = DefaultBottleneckConfig()

	}

	detector := &BottleneckDetector{

		config: config,

		activeBottlenecks: make(map[string]*ActiveBottleneck),

		detectedBottlenecks: make([]DetectedBottleneck, 0),

		detectionMetrics: &DetectionMetrics{},
	}

	// Initialize analysis engines.

	detector.criticalPathAnalyzer = NewCriticalPathAnalyzer()

	detector.resourceBottleneckFinder = NewResourceBottleneckFinder(config.ResourceThreshold)

	detector.contentionDetector = NewContentionPointDetector(config.ContentionThreshold)

	detector.cascadeAnalyzer = NewCascadeDelayAnalyzer()

	detector.rootCauseAnalyzer = NewRootCauseAnalyzer()

	// Initialize remediation system.

	if config.EnableAutoRemediation {

		detector.autoRemediator = NewAutomaticRemediator(config.RemediationPolicy)

	}

	// Initialize tracking.

	detector.historicalBottlenecks = NewBottleneckHistory()

	detector.performanceImpact = NewPerformanceImpactTracker()

	// Start detection loop.

	go detector.runContinuousDetection()

	return detector

}

// DetectBottlenecks analyzes data to detect bottlenecks.

func (d *BottleneckDetector) DetectBottlenecks(ctx context.Context, profile *IntentProfile) []DetectedBottleneck {

	var bottlenecks []DetectedBottleneck

	// Analyze critical path.

	criticalPath := d.criticalPathAnalyzer.AnalyzePath(profile)

	if criticalPath != nil && criticalPath.BottleneckNode != "" {

		bottlenecks = append(bottlenecks, d.createBottleneckFromPath(criticalPath))

	}

	// Check resource bottlenecks.

	resourceBottlenecks := d.resourceBottleneckFinder.FindBottlenecks(profile)

	for _, rb := range resourceBottlenecks {

		bottlenecks = append(bottlenecks, d.createBottleneckFromResource(rb))

	}

	// Detect contention points.

	contentionPoints := d.contentionDetector.DetectContention(profile)

	for _, cp := range contentionPoints {

		bottlenecks = append(bottlenecks, d.createBottleneckFromContention(cp))

	}

	// Analyze cascade delays.

	cascades := d.cascadeAnalyzer.AnalyzeCascades(profile)

	for _, cascade := range cascades {

		if cascade.AmplificationFactor > 1.5 {

			bottlenecks = append(bottlenecks, d.createBottleneckFromCascade(cascade))

		}

	}

	// Perform root cause analysis.

	for i := range bottlenecks {

		bottlenecks[i].RootCause = d.rootCauseAnalyzer.AnalyzeRootCause(&bottlenecks[i])

	}

	// Generate recommendations.

	for i := range bottlenecks {

		bottlenecks[i].Recommendations = d.generateRecommendations(&bottlenecks[i])

	}

	// Store detected bottlenecks.

	d.storeBottlenecks(bottlenecks)

	// Auto-remediate if enabled.

	if d.config.EnableAutoRemediation {

		for i := range bottlenecks {

			if bottlenecks[i].Severity == CriticalSeverity {

				result := d.autoRemediator.Remediate(ctx, &bottlenecks[i])

				bottlenecks[i].AutoRemediation = result

			}

		}

	}

	// Update metrics.

	d.detectionMetrics.bottlenecksDetected.Add(int64(len(bottlenecks)))

	return bottlenecks

}

// GetActiveBottlenecks returns currently active bottlenecks.

func (d *BottleneckDetector) GetActiveBottlenecks() []ActiveBottleneck {

	d.mu.RLock()

	defer d.mu.RUnlock()

	var active []ActiveBottleneck

	for _, bottleneck := range d.activeBottlenecks {

		active = append(active, *bottleneck)

	}

	// Sort by impact.

	sort.Slice(active, func(i, j int) bool {

		return active[i].CumulativeImpact > active[j].CumulativeImpact

	})

	return active

}

// GetBottleneckHistory returns historical bottleneck data.

func (d *BottleneckDetector) GetBottleneckHistory(duration time.Duration) []HistoricalBottleneck {

	return d.historicalBottlenecks.GetHistory(duration)

}

// GetRecurringPatterns returns detected recurring patterns.

func (d *BottleneckDetector) GetRecurringPatterns() []RecurringPattern {

	return d.historicalBottlenecks.GetPatterns()

}

// GetImpactReport returns performance impact report.

func (d *BottleneckDetector) GetImpactReport() *ImpactReport {

	return &ImpactReport{

		CurrentImpact: d.performanceImpact.GetCurrentImpact(),

		CumulativeImpact: d.performanceImpact.GetCumulativeImpact(),

		ImpactHistory: d.performanceImpact.GetHistory(),

		TopImpactors: d.getTopImpactors(),
	}

}

// Helper methods.

func (d *BottleneckDetector) createBottleneckFromPath(path *CriticalPath) DetectedBottleneck {

	return DetectedBottleneck{

		ID: fmt.Sprintf("bottleneck-%d", time.Now().UnixNano()),

		Type: AlgorithmicBottleneck,

		Component: path.BottleneckNode,

		Severity: d.calculateSeverity(path.TotalLatency),

		DetectedAt: time.Now(),

		Duration: path.TotalLatency,

		Impact: d.calculateImpact(path),

		Description: fmt.Sprintf("Critical path bottleneck in %s", path.BottleneckNode),

		Evidence: d.gatherPathEvidence(path),
	}

}

func (d *BottleneckDetector) createBottleneckFromResource(rb ResourceBottleneck) DetectedBottleneck {

	bottleneckType := d.mapResourceToType(rb.Resource)

	return DetectedBottleneck{

		ID: fmt.Sprintf("bottleneck-%d", time.Now().UnixNano()),

		Type: bottleneckType,

		Component: rb.Resource,

		Severity: d.calculateResourceSeverity(rb.Utilization, rb.Threshold),

		DetectedAt: time.Now(),

		Duration: rb.Duration,

		Impact: rb.Utilization / 100,

		Description: fmt.Sprintf("%s utilization at %.1f%% (threshold: %.1f%%)",

			rb.Resource, rb.Utilization, rb.Threshold),

		Evidence: d.gatherResourceEvidence(rb),
	}

}

func (d *BottleneckDetector) createBottleneckFromContention(cp ContentionPoint) DetectedBottleneck {

	return DetectedBottleneck{

		ID: fmt.Sprintf("bottleneck-%d", time.Now().UnixNano()),

		Type: ConcurrencyBottleneck,

		Component: cp.Location,

		Severity: d.calculateContentionSeverity(cp),

		DetectedAt: time.Now(),

		Duration: cp.WaitTime,

		Impact: cp.Impact,

		Description: fmt.Sprintf("Contention at %s: %d waiting, %.2fs wait time",

			cp.Location, cp.WaitingCount, cp.WaitTime.Seconds()),

		Evidence: d.gatherContentionEvidence(cp),
	}

}

func (d *BottleneckDetector) createBottleneckFromCascade(cascade *DelayCascade) DetectedBottleneck {

	return DetectedBottleneck{

		ID: fmt.Sprintf("bottleneck-%d", time.Now().UnixNano()),

		Type: AlgorithmicBottleneck,

		Component: cascade.OriginComponent,

		Severity: d.calculateCascadeSeverity(cascade),

		DetectedAt: time.Now(),

		Duration: cascade.TotalImpact,

		Impact: cascade.AmplificationFactor,

		Description: fmt.Sprintf("Cascade delay from %s affecting %d components (%.1fx amplification)",

			cascade.OriginComponent, len(cascade.AffectedComponents), cascade.AmplificationFactor),

		Evidence: d.gatherCascadeEvidence(cascade),
	}

}

func (d *BottleneckDetector) calculateSeverity(latency time.Duration) BottleneckSeverity {

	if latency > d.config.LatencyThreshold*3 {

		return CriticalSeverity

	} else if latency > d.config.LatencyThreshold*2 {

		return HighSeverity

	} else if latency > d.config.LatencyThreshold {

		return MediumSeverity

	}

	return LowSeverity

}

func (d *BottleneckDetector) calculateResourceSeverity(utilization, threshold float64) BottleneckSeverity {

	ratio := utilization / threshold

	if ratio > 1.5 {

		return CriticalSeverity

	} else if ratio > 1.2 {

		return HighSeverity

	} else if ratio > 1.0 {

		return MediumSeverity

	}

	return LowSeverity

}

func (d *BottleneckDetector) calculateContentionSeverity(cp ContentionPoint) BottleneckSeverity {

	if cp.WaitTime > d.config.ContentionThreshold.LockWaitTime*3 {

		return CriticalSeverity

	} else if cp.WaitTime > d.config.ContentionThreshold.LockWaitTime*2 {

		return HighSeverity

	} else if cp.WaitTime > d.config.ContentionThreshold.LockWaitTime {

		return MediumSeverity

	}

	return LowSeverity

}

func (d *BottleneckDetector) calculateCascadeSeverity(cascade *DelayCascade) BottleneckSeverity {

	if cascade.AmplificationFactor > 3.0 {

		return CriticalSeverity

	} else if cascade.AmplificationFactor > 2.0 {

		return HighSeverity

	} else if cascade.AmplificationFactor > 1.5 {

		return MediumSeverity

	}

	return LowSeverity

}

func (d *BottleneckDetector) calculateImpact(path *CriticalPath) float64 {

	if path.OptimizationPotential == 0 {

		return 0

	}

	return float64(path.OptimizationPotential) / float64(path.TotalLatency)

}

func (d *BottleneckDetector) mapResourceToType(resource string) BottleneckType {

	switch resource {

	case "cpu":

		return CPUBottleneck

	case "memory":

		return MemoryBottleneck

	case "disk":

		return IOBottleneck

	case "network":

		return NetworkBottleneck

	default:

		return AlgorithmicBottleneck

	}

}

func (d *BottleneckDetector) gatherPathEvidence(path *CriticalPath) []BottleneckEvidence {

	var evidence []BottleneckEvidence

	for _, component := range path.Components {

		evidence = append(evidence, BottleneckEvidence{

			Metric: fmt.Sprintf("%s_latency", component.Name),

			Value: component.Latency,

			Threshold: d.config.LatencyThreshold,

			Deviation: float64(component.Latency) / float64(d.config.LatencyThreshold),

			Timestamp: time.Now(),
		})

	}

	return evidence

}

func (d *BottleneckDetector) gatherResourceEvidence(rb ResourceBottleneck) []BottleneckEvidence {

	return []BottleneckEvidence{

		{

			Metric: fmt.Sprintf("%s_utilization", rb.Resource),

			Value: rb.Utilization,

			Threshold: rb.Threshold,

			Deviation: rb.Utilization / rb.Threshold,

			Timestamp: time.Now(),
		},
	}

}

func (d *BottleneckDetector) gatherContentionEvidence(cp ContentionPoint) []BottleneckEvidence {

	return []BottleneckEvidence{

		{

			Metric: "wait_time",

			Value: cp.WaitTime,

			Threshold: d.config.ContentionThreshold.LockWaitTime,

			Deviation: float64(cp.WaitTime) / float64(d.config.ContentionThreshold.LockWaitTime),

			Timestamp: time.Now(),
		},

		{

			Metric: "waiting_count",

			Value: cp.WaitingCount,

			Threshold: 10, // Example threshold

			Deviation: float64(cp.WaitingCount) / 10,

			Timestamp: time.Now(),
		},
	}

}

func (d *BottleneckDetector) gatherCascadeEvidence(cascade *DelayCascade) []BottleneckEvidence {

	var evidence []BottleneckEvidence

	evidence = append(evidence, BottleneckEvidence{

		Metric: "amplification_factor",

		Value: cascade.AmplificationFactor,

		Threshold: 1.5,

		Deviation: cascade.AmplificationFactor / 1.5,

		Timestamp: time.Now(),
	})

	for _, affected := range cascade.AffectedComponents {

		evidence = append(evidence, BottleneckEvidence{

			Metric: fmt.Sprintf("%s_delay", affected.Name),

			Value: affected.DirectDelay + affected.IndirectDelay,

			Threshold: d.config.LatencyThreshold,

			Deviation: float64(affected.DirectDelay+affected.IndirectDelay) / float64(d.config.LatencyThreshold),

			Timestamp: time.Now(),
		})

	}

	return evidence

}

func (d *BottleneckDetector) generateRecommendations(bottleneck *DetectedBottleneck) []RemediationAction {

	var recommendations []RemediationAction

	switch bottleneck.Type {

	case CPUBottleneck:

		recommendations = append(recommendations, RemediationAction{

			ID: fmt.Sprintf("rec-%d", time.Now().UnixNano()),

			Strategy: "scale_out",

			Status: "recommended",

			Result: "Increase CPU allocation or scale out instances",
		})

	case MemoryBottleneck:

		recommendations = append(recommendations, RemediationAction{

			ID: fmt.Sprintf("rec-%d", time.Now().UnixNano()),

			Strategy: "optimize_memory",

			Status: "recommended",

			Result: "Optimize memory usage or increase memory allocation",
		})

	case DatabaseBottleneck:

		recommendations = append(recommendations, RemediationAction{

			ID: fmt.Sprintf("rec-%d", time.Now().UnixNano()),

			Strategy: "optimize_queries",

			Status: "recommended",

			Result: "Optimize database queries or add indexes",
		})

	case ConcurrencyBottleneck:

		recommendations = append(recommendations, RemediationAction{

			ID: fmt.Sprintf("rec-%d", time.Now().UnixNano()),

			Strategy: "increase_concurrency",

			Status: "recommended",

			Result: "Increase connection pool size or worker threads",
		})

	case CacheBottleneck:

		recommendations = append(recommendations, RemediationAction{

			ID: fmt.Sprintf("rec-%d", time.Now().UnixNano()),

			Strategy: "improve_caching",

			Status: "recommended",

			Result: "Implement or improve caching strategy",
		})

	}

	return recommendations

}

func (d *BottleneckDetector) storeBottlenecks(bottlenecks []DetectedBottleneck) {

	d.mu.Lock()

	defer d.mu.Unlock()

	for _, bottleneck := range bottlenecks {

		// Add to detected list.

		d.detectedBottlenecks = append(d.detectedBottlenecks, bottleneck)

		// Update active bottlenecks.

		if active, exists := d.activeBottlenecks[bottleneck.Component]; exists {

			active.LastSeen = time.Now()

			active.OccurrenceCount++

			active.CumulativeImpact += bottleneck.Impact

		} else {

			d.activeBottlenecks[bottleneck.Component] = &ActiveBottleneck{

				Bottleneck: bottleneck,

				StartTime: time.Now(),

				LastSeen: time.Now(),

				OccurrenceCount: 1,

				CumulativeImpact: bottleneck.Impact,

				Status: "active",
			}

		}

		// Update metrics.

		if bottleneck.Severity == CriticalSeverity {

			d.detectionMetrics.criticalBottlenecks.Add(1)

		}

	}

	// Trim old data.

	if len(d.detectedBottlenecks) > 10000 {

		d.detectedBottlenecks = d.detectedBottlenecks[len(d.detectedBottlenecks)-10000:]

	}

}

func (d *BottleneckDetector) getTopImpactors() []TopImpactor {

	d.mu.RLock()

	defer d.mu.RUnlock()

	var impactors []TopImpactor

	for component, active := range d.activeBottlenecks {

		impactors = append(impactors, TopImpactor{

			Component: component,

			CumulativeImpact: active.CumulativeImpact,

			OccurrenceCount: active.OccurrenceCount,

			Type: active.Bottleneck.Type,
		})

	}

	// Sort by impact.

	sort.Slice(impactors, func(i, j int) bool {

		return impactors[i].CumulativeImpact > impactors[j].CumulativeImpact

	})

	// Return top 10.

	if len(impactors) > 10 {

		impactors = impactors[:10]

	}

	return impactors

}

func (d *BottleneckDetector) runContinuousDetection() {

	ticker := time.NewTicker(d.config.AnalysisWindow)

	defer ticker.Stop()

	for range ticker.C {

		// Clean up resolved bottlenecks.

		d.cleanupResolvedBottlenecks()

		// Check for recurring patterns.

		d.historicalBottlenecks.AnalyzePatterns()

		// Update impact measurements.

		d.performanceImpact.UpdateMeasurements()

	}

}

func (d *BottleneckDetector) cleanupResolvedBottlenecks() {

	d.mu.Lock()

	defer d.mu.Unlock()

	cutoff := time.Now().Add(-5 * time.Minute)

	for component, active := range d.activeBottlenecks {

		if active.LastSeen.Before(cutoff) {

			active.Status = "resolved"

			// Move to history.

			d.historicalBottlenecks.Add(HistoricalBottleneck{

				Bottleneck: active.Bottleneck,

				Resolution: "timeout",

				ResolutionTime: time.Since(active.StartTime),

				Recurrence: active.OccurrenceCount,
			})

			delete(d.activeBottlenecks, component)

		}

	}

}

// Supporting type implementations.

// NewCriticalPathAnalyzer performs newcriticalpathanalyzer operation.

func NewCriticalPathAnalyzer() *CriticalPathAnalyzer {

	return &CriticalPathAnalyzer{

		dependencyGraph: &DependencyGraph{},

		criticalPaths: make(map[string]*CriticalPath),

		pathLatencies: make(map[string]time.Duration),

		pathStatistics: &PathStatistics{},
	}

}

// AnalyzePath performs analyzepath operation.

func (c *CriticalPathAnalyzer) AnalyzePath(profile *IntentProfile) *CriticalPath {

	// Simplified critical path analysis.

	path := &CriticalPath{

		ID: fmt.Sprintf("path-%d", time.Now().UnixNano()),

		Components: []PathComponent{},

		TotalLatency: profile.TotalDuration,
	}

	// Find the component with maximum latency.

	var maxComponent string

	var maxLatency time.Duration

	for component, latency := range profile.Components {

		if latency.Duration > maxLatency {

			maxLatency = latency.Duration

			maxComponent = component

		}

		path.Components = append(path.Components, PathComponent{

			Name: component,

			Latency: latency.Duration,

			Percentage: float64(latency.Duration) / float64(profile.TotalDuration) * 100,

			Parallelizable: false, // Simplified

		})

	}

	path.BottleneckNode = maxComponent

	path.OptimizationPotential = maxLatency / 2 // Assume 50% improvement potential

	return path

}

// NewResourceBottleneckFinder performs newresourcebottleneckfinder operation.

func NewResourceBottleneckFinder(thresholds ResourceThresholds) *ResourceBottleneckFinder {

	return &ResourceBottleneckFinder{

		thresholds: thresholds,

		cpuMonitor: &CPUMonitor{},

		memoryMonitor: &MemoryMonitor{},

		diskMonitor: &DiskIOMonitor{},

		networkMonitor: &NetworkMonitor{},

		bottlenecks: make([]ResourceBottleneck, 0),
	}

}

// FindBottlenecks performs findbottlenecks operation.

func (r *ResourceBottleneckFinder) FindBottlenecks(profile *IntentProfile) []ResourceBottleneck {

	var bottlenecks []ResourceBottleneck

	// Check CPU utilization.

	if cpuUtil := r.cpuMonitor.GetUtilization(); cpuUtil > r.thresholds.CPUUtilization {

		bottlenecks = append(bottlenecks, ResourceBottleneck{

			Resource: "cpu",

			Utilization: cpuUtil,

			Threshold: r.thresholds.CPUUtilization,

			Duration: profile.TotalDuration,

			Impact: "high",

			AffectedOps: []string{"llm_processing", "rag_retrieval"},
		})

	}

	// Check memory utilization.

	if memUtil := r.memoryMonitor.GetUtilization(); memUtil > r.thresholds.MemoryUtilization {

		bottlenecks = append(bottlenecks, ResourceBottleneck{

			Resource: "memory",

			Utilization: memUtil,

			Threshold: r.thresholds.MemoryUtilization,

			Duration: profile.TotalDuration,

			Impact: "medium",

			AffectedOps: []string{"cache", "rag_indexing"},
		})

	}

	return bottlenecks

}

// NewContentionPointDetector performs newcontentionpointdetector operation.

func NewContentionPointDetector(thresholds ContentionThresholds) *ContentionPointDetector {

	return &ContentionPointDetector{

		thresholds: thresholds,

		lockMonitor: &LockContentionMonitor{},

		poolMonitor: &ConnectionPoolMonitor{},

		queueMonitor: &QueueContentionMonitor{},

		contentionPoints: make([]ContentionPoint, 0),
	}

}

// DetectContention performs detectcontention operation.

func (c *ContentionPointDetector) DetectContention(profile *IntentProfile) []ContentionPoint {

	var points []ContentionPoint

	// Check for database lock contention.

	for _, latency := range profile.Components {

		if latency.DatabaseLatency != nil &&

			latency.DatabaseLatency.LockWaitTime > c.thresholds.DatabaseLockTime {

			points = append(points, ContentionPoint{

				Type: "lock",

				Location: "database",

				WaitTime: latency.DatabaseLatency.LockWaitTime,

				WaitingCount: 5, // Example

				HoldTime: latency.DatabaseLatency.TransactionTime,

				Impact: float64(latency.DatabaseLatency.LockWaitTime) / float64(latency.Duration),
			})

		}

	}

	return points

}

// NewCascadeDelayAnalyzer performs newcascadedelayanalyzer operation.

func NewCascadeDelayAnalyzer() *CascadeDelayAnalyzer {

	return &CascadeDelayAnalyzer{

		cascades: make(map[string]*DelayCascade),

		impactMap: make(map[string][]string),

		cascadeEffects: make([]CascadeEffect, 0),
	}

}

// AnalyzeCascades performs analyzecascades operation.

func (c *CascadeDelayAnalyzer) AnalyzeCascades(profile *IntentProfile) []*DelayCascade {

	var cascades []*DelayCascade

	// Simplified cascade analysis.

	for component, latency := range profile.Components {

		if latency.Duration > 500*time.Millisecond {

			cascade := &DelayCascade{

				OriginComponent: component,

				OriginDelay: latency.Duration,

				TotalImpact: latency.Duration * 2, // Assume 2x impact

				AmplificationFactor: 2.0,
			}

			// Find affected components (simplified).

			for otherComp := range profile.Components {

				if otherComp != component {

					cascade.AffectedComponents = append(cascade.AffectedComponents, AffectedComponent{

						Name: otherComp,

						DirectDelay: latency.Duration / 4,

						IndirectDelay: latency.Duration / 8,
					})

				}

			}

			if len(cascade.AffectedComponents) > 0 {

				cascades = append(cascades, cascade)

			}

		}

	}

	return cascades

}

// NewRootCauseAnalyzer performs newrootcauseanalyzer operation.

func NewRootCauseAnalyzer() *RootCauseAnalyzer {

	return &RootCauseAnalyzer{

		changeAnalyzer: &ChangeCorrelationAnalyzer{},

		patternMatcher: &PatternMatcher{},

		anomalyCorrelator: &AnomalyCorrelator{},

		identifiedCauses: make(map[string]*RootCause),
	}

}

// AnalyzeRootCause performs analyzerootcause operation.

func (r *RootCauseAnalyzer) AnalyzeRootCause(bottleneck *DetectedBottleneck) *RootCause {

	cause := &RootCause{

		ID: fmt.Sprintf("cause-%d", time.Now().UnixNano()),

		Type: string(bottleneck.Type),

		Description: fmt.Sprintf("Root cause for %s bottleneck", bottleneck.Component),

		Confidence: 0.75, // Example confidence

	}

	// Add evidence.

	cause.Evidence = []CausalEvidence{

		{

			Type: "metric",

			Description: "High latency detected",

			Correlation: 0.85,

			Timestamp: time.Now(),
		},
	}

	// Add timeline.

	cause.Timeline = []TimelineEvent{

		{

			Timestamp: bottleneck.DetectedAt,

			Event: "Bottleneck detected",

			Component: bottleneck.Component,

			Impact: string(bottleneck.Severity),
		},
	}

	return cause

}

// NewAutomaticRemediator performs newautomaticremediator operation.

func NewAutomaticRemediator(policy RemediationPolicy) *AutomaticRemediator {

	remediator := &AutomaticRemediator{

		strategies: make(map[BottleneckType][]RemediationStrategy),

		history: make([]RemediationAction, 0),

		activeRemediations: make(map[string]*ActiveRemediation),

		policy: policy,
	}

	// Initialize strategies.

	remediator.initializeStrategies()

	return remediator

}

func (a *AutomaticRemediator) initializeStrategies() {

	// CPU bottleneck strategies.

	a.strategies[CPUBottleneck] = []RemediationStrategy{

		{

			Name: "scale_out",

			Type: "scaling",

			Applicability: func(b *DetectedBottleneck) bool {

				return b.Severity == CriticalSeverity

			},

			Execute: func(ctx context.Context, b *DetectedBottleneck) error {

				// Scale out implementation.

				return nil

			},

			ExpectedImprovement: 0.4,

			Risk: "low",
		},
	}

	// Add more strategies for other bottleneck types.

}

// Remediate performs remediate operation.

func (a *AutomaticRemediator) Remediate(ctx context.Context, bottleneck *DetectedBottleneck) *AutoRemediationResult {

	result := &AutoRemediationResult{

		Success: false,

		Message: "No applicable remediation strategy",
	}

	// Find applicable strategies.

	strategies, exists := a.strategies[bottleneck.Type]

	if !exists {

		return result

	}

	for _, strategy := range strategies {

		if strategy.Applicability(bottleneck) {

			// Execute remediation.

			action := RemediationAction{

				ID: fmt.Sprintf("action-%d", time.Now().UnixNano()),

				BottleneckID: bottleneck.ID,

				Strategy: strategy.Name,

				ExecutedAt: time.Now(),

				Status: "executing",
			}

			// Create cancellable context.

			remCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)

			// Store active remediation.

			a.mu.Lock()

			a.activeRemediations[action.ID] = &ActiveRemediation{

				Action: action,

				StartTime: time.Now(),

				Context: remCtx,

				CancelFunc: cancel,
			}

			a.mu.Unlock()

			// Execute strategy.

			err := strategy.Execute(remCtx, bottleneck)

			// Clean up.

			a.mu.Lock()

			delete(a.activeRemediations, action.ID)

			a.mu.Unlock()

			cancel()

			if err == nil {

				result.Success = true

				result.ActionsTaken = append(result.ActionsTaken, strategy.Name)

				result.ImprovementPct = strategy.ExpectedImprovement

				result.Message = "Remediation successful"

				action.Status = "success"

				action.Result = "Completed successfully"

				action.ImprovementMeasured = strategy.ExpectedImprovement

			} else {

				action.Status = "failed"

				action.Result = err.Error()

			}

			// Store in history.

			a.mu.Lock()

			a.history = append(a.history, action)

			a.mu.Unlock()

			// Update metrics.

			a.updateMetrics(result.Success)

			break // Apply only one strategy

		}

	}

	return result

}

func (a *AutomaticRemediator) updateMetrics(success bool) {

	// Update remediation metrics.

}

// NewBottleneckHistory performs newbottleneckhistory operation.

func NewBottleneckHistory() *BottleneckHistory {

	return &BottleneckHistory{

		bottlenecks: make([]HistoricalBottleneck, 0),

		recurringPatterns: make([]RecurringPattern, 0),

		statistics: &HistoricalStatistics{},
	}

}

// Add performs add operation.

func (h *BottleneckHistory) Add(bottleneck HistoricalBottleneck) {

	h.mu.Lock()

	defer h.mu.Unlock()

	h.bottlenecks = append(h.bottlenecks, bottleneck)

	// Keep only last 1000 entries.

	if len(h.bottlenecks) > 1000 {

		h.bottlenecks = h.bottlenecks[len(h.bottlenecks)-1000:]

	}

}

// GetHistory performs gethistory operation.

func (h *BottleneckHistory) GetHistory(duration time.Duration) []HistoricalBottleneck {

	h.mu.RLock()

	defer h.mu.RUnlock()

	cutoff := time.Now().Add(-duration)

	var history []HistoricalBottleneck

	for _, b := range h.bottlenecks {

		if b.Bottleneck.DetectedAt.After(cutoff) {

			history = append(history, b)

		}

	}

	return history

}

// GetPatterns performs getpatterns operation.

func (h *BottleneckHistory) GetPatterns() []RecurringPattern {

	h.mu.RLock()

	defer h.mu.RUnlock()

	patterns := make([]RecurringPattern, len(h.recurringPatterns))

	copy(patterns, h.recurringPatterns)

	return patterns

}

// AnalyzePatterns performs analyzepatterns operation.

func (h *BottleneckHistory) AnalyzePatterns() {

	// Analyze historical data for patterns.

}

// NewPerformanceImpactTracker performs newperformanceimpacttracker operation.

func NewPerformanceImpactTracker() *PerformanceImpactTracker {

	return &PerformanceImpactTracker{

		impactHistory: make([]ImpactMeasurement, 0),
	}

}

// GetCurrentImpact performs getcurrentimpact operation.

func (p *PerformanceImpactTracker) GetCurrentImpact() float64 {

	p.mu.RLock()

	defer p.mu.RUnlock()

	return p.currentImpact

}

// GetCumulativeImpact performs getcumulativeimpact operation.

func (p *PerformanceImpactTracker) GetCumulativeImpact() float64 {

	p.mu.RLock()

	defer p.mu.RUnlock()

	return p.cumulativeImpact

}

// GetHistory performs gethistory operation.

func (p *PerformanceImpactTracker) GetHistory() []ImpactMeasurement {

	p.mu.RLock()

	defer p.mu.RUnlock()

	history := make([]ImpactMeasurement, len(p.impactHistory))

	copy(history, p.impactHistory)

	return history

}

// UpdateMeasurements performs updatemeasurements operation.

func (p *PerformanceImpactTracker) UpdateMeasurements() {

	// Update impact measurements.

}

// Stub implementations for monitors.

type CPUMonitor struct{}

// GetUtilization performs getutilization operation.

func (c *CPUMonitor) GetUtilization() float64 { return 75.0 } // Example

// MemoryMonitor represents a memorymonitor.

type MemoryMonitor struct{}

// GetUtilization performs getutilization operation.

func (m *MemoryMonitor) GetUtilization() float64 { return 65.0 } // Example

// DiskIOMonitor represents a diskiomonitor.

type (
	DiskIOMonitor struct{}

	// NetworkMonitor represents a networkmonitor.

	NetworkMonitor struct{}

	// LockContentionMonitor represents a lockcontentionmonitor.

	LockContentionMonitor struct{}

	// ConnectionPoolMonitor represents a connectionpoolmonitor.

	ConnectionPoolMonitor struct{}

	// QueueContentionMonitor represents a queuecontentionmonitor.

	QueueContentionMonitor struct{}

	// DependencyGraph represents a dependencygraph.

	DependencyGraph struct{}

	// PathStatistics represents a pathstatistics.

	PathStatistics struct{}

	// ChangeCorrelationAnalyzer represents a changecorrelationanalyzer.

	ChangeCorrelationAnalyzer struct{}

	// PatternMatcher represents a patternmatcher.

	PatternMatcher struct{}

	// AnomalyCorrelator represents a anomalycorrelator.

	AnomalyCorrelator struct{}

	// HistoricalStatistics represents a historicalstatistics.

	HistoricalStatistics struct{}
)

// Report types.

type ImpactReport struct {
	CurrentImpact float64 `json:"current_impact"`

	CumulativeImpact float64 `json:"cumulative_impact"`

	ImpactHistory []ImpactMeasurement `json:"impact_history"`

	TopImpactors []TopImpactor `json:"top_impactors"`
}

// TopImpactor represents a topimpactor.

type TopImpactor struct {
	Component string `json:"component"`

	CumulativeImpact float64 `json:"cumulative_impact"`

	OccurrenceCount int `json:"occurrence_count"`

	Type BottleneckType `json:"type"`
}

// DefaultBottleneckConfig returns default configuration.

func DefaultBottleneckConfig() *BottleneckDetectorConfig {

	return &BottleneckDetectorConfig{

		LatencyThreshold: 500 * time.Millisecond,

		ResourceThreshold: ResourceThresholds{

			CPUUtilization: 80.0,

			MemoryUtilization: 85.0,

			DiskIOUtilization: 75.0,

			NetworkUtilization: 70.0,

			ConnectionPoolUsage: 90.0,
		},

		ContentionThreshold: ContentionThresholds{

			LockWaitTime: 100 * time.Millisecond,

			QueueDepth: 100,

			ThreadContention: 50.0,

			DatabaseLockTime: 200 * time.Millisecond,
		},

		CriticalPathDepth: 5,

		AnalysisWindow: 1 * time.Minute,

		MinSamplesForDetection: 10,

		EnableAutoRemediation: false,

		RemediationPolicy: RemediationPolicy{

			MaxScaleOutInstances: 10,

			MinScaleInInstances: 1,

			ScaleStepSize: 2,

			CooldownPeriod: 5 * time.Minute,

			AllowCacheInvalidation: true,

			AllowCircuitBreaker: true,
		},

		AlertingEnabled: true,

		AlertThresholds: AlertThresholds{

			CriticalBottleneckCount: 3,

			SustainedDuration: 5 * time.Minute,

			ImpactThreshold: 0.5,
		},
	}

}
