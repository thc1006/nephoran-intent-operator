package chaos

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/stretchr/testify/suite"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring/sla"
)

// Basic types needed by all chaos components

// ExperimentType defines types of chaos experiments
type ExperimentType string

const (
	ExperimentTypeComponentFailure   ExperimentType = "component_failure"
	ExperimentTypeNetworkPartition   ExperimentType = "network_partition"
	ExperimentTypeResourceExhaustion ExperimentType = "resource_exhaustion"
	ExperimentTypeLatencyInjection   ExperimentType = "latency_injection"
	ExperimentTypeDataCorruption     ExperimentType = "data_corruption"
	ExperimentTypeCascadingFailure   ExperimentType = "cascading_failure"
)

// ExperimentTarget defines experiment targets
type ExperimentTarget struct {
	Component  string            `json:"component"`
	Instance   string            `json:"instance"`
	Percentage float64           `json:"percentage"`
	Selector   map[string]string `json:"selector"`
}

// ExpectedBehavior defines expected system behavior during chaos
type ExpectedBehavior struct {
	SLAImpact        *SLAImpactExpectation `json:"sla_impact"`
	RecoveryTime     time.Duration         `json:"recovery_time"`
	AlertsExpected   []string              `json:"alerts_expected"`
	NoAlertsExpected []string              `json:"no_alerts_expected"`
	MetricChanges    []*MetricExpectation  `json:"metric_changes"`
}

// SLAImpactExpectation defines expected SLA impact
type SLAImpactExpectation struct {
	AvailabilityDrop float64 `json:"availability_drop"` // Expected drop in availability
	LatencyIncrease  float64 `json:"latency_increase"`  // Expected increase in latency
	ThroughputDrop   float64 `json:"throughput_drop"`   // Expected drop in throughput
}

// MetricExpectation defines expected metric behavior
type MetricExpectation struct {
	MetricName      string  `json:"metric_name"`
	ExpectedChange  string  `json:"expected_change"` // "increase", "decrease", "unchanged"
	ChangeThreshold float64 `json:"change_threshold"`
}

// SafetyCheck defines safety checks to prevent excessive damage
type SafetyCheck struct {
	Name       string                 `json:"name"`
	Type       SafetyCheckType        `json:"type"`
	Threshold  float64                `json:"threshold"`
	Action     SafetyAction           `json:"action"`
	Parameters map[string]interface{} `json:"parameters"`
}

// SafetyCheckType defines types of safety checks
type SafetyCheckType string

const (
	SafetyCheckAvailability SafetyCheckType = "availability"
	SafetyCheckLatency      SafetyCheckType = "latency"
	SafetyCheckErrorRate    SafetyCheckType = "error_rate"
	SafetyCheckMemory       SafetyCheckType = "memory"
	SafetyCheckCPU          SafetyCheckType = "cpu"
)

// SafetyAction defines actions to take when safety check fails
type SafetyAction string

const (
	SafetyActionAbort   SafetyAction = "abort"
	SafetyActionPause   SafetyAction = "pause"
	SafetyActionRecover SafetyAction = "recover"
	SafetyActionAlert   SafetyAction = "alert"
)

// ChaosExperiment defines a chaos experiment
type ChaosExperiment struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Type             ExperimentType         `json:"type"`
	Target           ExperimentTarget       `json:"target"`
	Parameters       map[string]interface{} `json:"parameters"`
	Duration         time.Duration          `json:"duration"`
	ExpectedBehavior *ExpectedBehavior      `json:"expected_behavior"`
	SafetyChecks     []*SafetyCheck         `json:"safety_checks"`
}

// ExperimentStatus represents experiment status
type ExperimentStatus string

const (
	ExperimentStatusRunning   ExperimentStatus = "running"
	ExperimentStatusCompleted ExperimentStatus = "completed"
	ExperimentStatusAborted   ExperimentStatus = "aborted"
	ExperimentStatusFailed    ExperimentStatus = "failed"
)

// RecoveryStage represents a stage in the recovery process
type RecoveryStage struct {
	Name      string             `json:"name"`
	StartTime time.Time          `json:"start_time"`
	EndTime   time.Time          `json:"end_time"`
	Duration  time.Duration      `json:"duration"`
	Success   bool               `json:"success"`
	Metrics   map[string]float64 `json:"metrics"`
}

// RecoveryTracker tracks system recovery events and metrics
type RecoveryTracker struct {
	recoveryEvents map[string]*RecoveryEvent
	mutex          sync.RWMutex
}

// RecoveryEvent represents a recovery event
type RecoveryEvent struct {
	ID             string             `json:"id"`
	ComponentID    string             `json:"component_id"`
	FailureType    ExperimentType     `json:"failure_type"`
	StartTime      time.Time          `json:"start_time"`
	EndTime        time.Time          `json:"end_time"`
	Duration       time.Duration      `json:"duration"`
	Success        bool               `json:"success"`
	RecoveryStages []*RecoveryStage   `json:"recovery_stages"`
	Metrics        map[string]float64 `json:"metrics"`
}

// SLATracker tracks SLA metrics during chaos experiments
type SLATracker struct {
	prometheusClient v1.API
	metrics          *SLAMetrics
	violations       []*SLAViolation
	baseline         *SLABaseline
	mutex            sync.RWMutex
}

// SLAMetrics tracks SLA-related metrics
type SLAMetrics struct {
	Availability    float64           `json:"availability"`
	LatencyP95      time.Duration     `json:"latency_p95"`
	Throughput      float64           `json:"throughput"`
	ErrorRate       float64           `json:"error_rate"`
	LastUpdated     time.Time         `json:"last_updated"`
	TrendIndicators map[string]string `json:"trend_indicators"`
}

// SLABaseline represents baseline SLA metrics
type SLABaseline struct {
	Availability    float64       `json:"availability"`
	LatencyP95      time.Duration `json:"latency_p95"`
	Throughput      float64       `json:"throughput"`
	ErrorRate       float64       `json:"error_rate"`
	MeasurementTime time.Time     `json:"measurement_time"`
	Duration        time.Duration `json:"duration"`
}

// ExperimentScheduler schedules and manages chaos experiments
type ExperimentScheduler struct {
	activeExperiments map[string]*ScheduledExperiment
	queue             []*ScheduledExperiment
	maxConcurrent     int
	mutex             sync.RWMutex
}

// ScheduledExperiment represents a scheduled experiment
type ScheduledExperiment struct {
	Experiment  *ChaosExperiment `json:"experiment"`
	ScheduledAt time.Time        `json:"scheduled_at"`
	StartedAt   *time.Time       `json:"started_at,omitempty"`
	Status      string           `json:"status"`
	Priority    int              `json:"priority"`
}

// SafetyViolation represents a safety check violation
type SafetyViolation struct {
	ID          string                 `json:"id"`
	CheckName   string                 `json:"check_name"`
	CheckType   SafetyCheckType        `json:"check_type"`
	Threshold   float64                `json:"threshold"`
	ActualValue float64                `json:"actual_value"`
	Severity    ViolationSeverity      `json:"severity"`
	Timestamp   time.Time              `json:"timestamp"`
	Action      SafetyAction           `json:"action"`
	Context     map[string]interface{} `json:"context"`
	Resolved    bool                   `json:"resolved"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
}

// ViolationSeverity defines severity levels for violations
type ViolationSeverity string

const (
	ViolationSeverityLow      ViolationSeverity = "low"
	ViolationSeverityMedium   ViolationSeverity = "medium"
	ViolationSeverityHigh     ViolationSeverity = "high"
	ViolationSeverityCritical ViolationSeverity = "critical"
)

// SLAMonitor monitors SLA compliance
type SLAMonitor struct {
	prometheusClient v1.API
	slaTargets       *SLATargets
	violations       []*SLAViolation
	lastCheck        time.Time
	mutex            sync.RWMutex
}

// SLATargets defines SLA target thresholds
type SLATargets struct {
	Availability float64       `json:"availability"`
	LatencyP95   time.Duration `json:"latency_p95"`
	Throughput   float64       `json:"throughput"`
	ErrorRate    float64       `json:"error_rate"`
}

// AlertValidator validates alerting system behavior
type AlertValidator struct {
	prometheusClient  v1.API
	expectedAlerts    map[string]*ExpectedAlert
	actualAlerts      []*ActualAlert
	falsePositives    []*AlertEvent
	falseNegatives    []*AlertEvent
	validationResults *AlertValidationResults
	mutex             sync.RWMutex
}

// ExpectedAlert defines an expected alert
type ExpectedAlert struct {
	Name        string            `json:"name"`
	Severity    string            `json:"severity"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Timestamp   time.Time         `json:"timestamp"`
	Duration    time.Duration     `json:"duration"`
}

// ActualAlert represents an actual fired alert
type ActualAlert struct {
	Name        string            `json:"name"`
	Severity    string            `json:"severity"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	FiredAt     time.Time         `json:"fired_at"`
	ResolvedAt  *time.Time        `json:"resolved_at,omitempty"`
	State       string            `json:"state"`
}

// AlertValidationResults contains alert validation metrics
type AlertValidationResults struct {
	TotalExpected  int     `json:"total_expected"`
	TotalActual    int     `json:"total_actual"`
	TruePositives  int     `json:"true_positives"`
	FalsePositives int     `json:"false_positives"`
	FalseNegatives int     `json:"false_negatives"`
	Accuracy       float64 `json:"accuracy"`
	Precision      float64 `json:"precision"`
	Recall         float64 `json:"recall"`
	F1Score        float64 `json:"f1_score"`
}

// DataConsistencyValidator validates data consistency
type DataConsistencyValidator struct {
	prometheusClient  v1.API
	consistencyChecks []*ConsistencyCheck
	inconsistencies   []*DataInconsistency
	overallScore      float64
	lastValidation    time.Time
	mutex             sync.RWMutex
}

// ConsistencyCheck defines a data consistency check
type ConsistencyCheck struct {
	Name       string                 `json:"name"`
	Query      string                 `json:"query"`
	Expected   interface{}            `json:"expected"`
	Tolerance  float64                `json:"tolerance"`
	CheckType  ConsistencyCheckType   `json:"check_type"`
	Parameters map[string]interface{} `json:"parameters"`
}

// ConsistencyCheckType defines types of consistency checks
type ConsistencyCheckType string

const (
	ConsistencyCheckTypeValue        ConsistencyCheckType = "value"
	ConsistencyCheckTypeTrend        ConsistencyCheckType = "trend"
	ConsistencyCheckTypeCorrelation  ConsistencyCheckType = "correlation"
	ConsistencyCheckTypeCompleteness ConsistencyCheckType = "completeness"
)

// DataInconsistency represents a data consistency violation
type DataInconsistency struct {
	CheckName string                 `json:"check_name"`
	Expected  interface{}            `json:"expected"`
	Actual    interface{}            `json:"actual"`
	Deviation float64                `json:"deviation"`
	Timestamp time.Time              `json:"timestamp"`
	Severity  InconsistencySeverity  `json:"severity"`
	Context   map[string]interface{} `json:"context"`
	Resolved  bool                   `json:"resolved"`
}

// InconsistencySeverity defines inconsistency severity levels
type InconsistencySeverity string

const (
	InconsistencySeverityMinor    InconsistencySeverity = "minor"
	InconsistencySeverityModerate InconsistencySeverity = "moderate"
	InconsistencySeverityMajor    InconsistencySeverity = "major"
	InconsistencySeverityCritical InconsistencySeverity = "critical"
)

// SLAViolation represents an SLA violation event
type SLAViolation struct {
	ID              string                 `json:"id"`
	MetricName      string                 `json:"metric_name"`
	Threshold       float64                `json:"threshold"`
	ActualValue     float64                `json:"actual_value"`
	Deviation       float64                `json:"deviation"`
	Severity        ViolationSeverity      `json:"severity"`
	StartTime       time.Time              `json:"start_time"`
	EndTime         *time.Time             `json:"end_time,omitempty"`
	Duration        time.Duration          `json:"duration"`
	Impact          *ViolationImpact       `json:"impact"`
	Context         map[string]interface{} `json:"context"`
	Acknowledged    bool                   `json:"acknowledged"`
	Resolved        bool                   `json:"resolved"`
	ResolutionNotes string                 `json:"resolution_notes,omitempty"`
}

// ViolationImpact describes the impact of an SLA violation
type ViolationImpact struct {
	AffectedUsers     int                    `json:"affected_users"`
	BusinessImpact    string                 `json:"business_impact"`
	TechnicalImpact   string                 `json:"technical_impact"`
	DownstreamEffects []string               `json:"downstream_effects"`
	Metrics           map[string]interface{} `json:"metrics"`
}

// AlertEvent represents an alert event
type AlertEvent struct {
	ID           string                 `json:"id"`
	AlertName    string                 `json:"alert_name"`
	Severity     AlertSeverity          `json:"severity"`
	State        AlertState             `json:"state"`
	Labels       map[string]string      `json:"labels"`
	Annotations  map[string]string      `json:"annotations"`
	StartsAt     time.Time              `json:"starts_at"`
	EndsAt       *time.Time             `json:"ends_at,omitempty"`
	GeneratorURL string                 `json:"generator_url"`
	Context      map[string]interface{} `json:"context"`
	Source       string                 `json:"source"`
	Fingerprint  string                 `json:"fingerprint"`
}

// AlertSeverity defines alert severity levels
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityError    AlertSeverity = "error"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertState defines alert states
type AlertState string

const (
	AlertStateInactive AlertState = "inactive"
	AlertStatePending  AlertState = "pending"
	AlertStateFiring   AlertState = "firing"
	AlertStateResolved AlertState = "resolved"
)

// NewSLATracker creates a new SLA tracker
func NewSLATracker(prometheusClient v1.API) *SLATracker {
	return &SLATracker{
		prometheusClient: prometheusClient,
		metrics:          &SLAMetrics{},
		violations:       make([]*SLAViolation, 0),
		baseline:         &SLABaseline{},
	}
}

// NewRecoveryTracker creates a new recovery tracker
func NewRecoveryTracker() *RecoveryTracker {
	return &RecoveryTracker{
		recoveryEvents: make(map[string]*RecoveryEvent),
	}
}

// Constructor functions for monitoring components
func NewAlertMonitor(prometheusClient v1.API) *AlertMonitor {
	return &AlertMonitor{
		prometheusClient:  prometheusClient,
		alerts:            make([]*AlertEvent, 0),
		validationResults: &AlertValidationResults{},
	}
}

// AlertMonitor monitors alerts during chaos experiments
type AlertMonitor struct {
	prometheusClient  v1.API
	alerts            []*AlertEvent
	validationResults *AlertValidationResults
	mutex             sync.RWMutex
}

// collectAlerts simulates alert collection from Prometheus
func (am *AlertMonitor) collectAlerts() {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	// In real implementation, this would query Prometheus for active alerts
	// For testing purposes, we'll just maintain the structure
}

func NewDataConsistencyMonitor() *DataConsistencyMonitor {
	return &DataConsistencyMonitor{
		consistencyScore: 1.0,
		checks:           make([]*ConsistencyCheck, 0),
		violations:       make([]*DataInconsistency, 0),
	}
}

// DataConsistencyMonitor monitors data consistency
type DataConsistencyMonitor struct {
	consistencyScore float64
	checks           []*ConsistencyCheck
	violations       []*DataInconsistency
	mutex            sync.RWMutex
}

// performConsistencyChecks performs consistency validation
func (dcm *DataConsistencyMonitor) performConsistencyChecks() {
	dcm.mutex.Lock()
	defer dcm.mutex.Unlock()

	// Simulate consistency checking
	violationCount := len(dcm.violations)
	totalChecks := len(dcm.checks)
	if totalChecks == 0 {
		totalChecks = 10 // Default number of checks
	}

	dcm.consistencyScore = float64(totalChecks-violationCount) / float64(totalChecks)
	if dcm.consistencyScore < 0 {
		dcm.consistencyScore = 0
	}
}

func NewFalsePositiveTracker() *FalsePositiveTracker {
	return &FalsePositiveTracker{
		totalAlerts:    0,
		falsePositives: 0,
		alertEvents:    make([]*AlertEvent, 0),
	}
}

// FalsePositiveTracker tracks false positive alerts
type FalsePositiveTracker struct {
	totalAlerts    int
	falsePositives int
	alertEvents    []*AlertEvent
	mutex          sync.RWMutex
}

// checkForAlerts simulates alert checking
func (fpt *FalsePositiveTracker) checkForAlerts() {
	fpt.mutex.Lock()
	defer fpt.mutex.Unlock()
	// Simulation - in real implementation would query Prometheus for alerts
	// For testing purposes, we'll just track the counts
}

// AlertExperimentResults contains results from alert validation experiments
type AlertExperimentResults struct {
	TotalAlerts       int                     `json:"total_alerts"`
	ExpectedAlerts    int                     `json:"expected_alerts"`
	ActualAlerts      int                     `json:"actual_alerts"`
	FalsePositives    int                     `json:"false_positives"`
	FalseNegatives    int                     `json:"false_negatives"`
	AlertAccuracy     float64                 `json:"alert_accuracy"`
	ResponseTimes     []time.Duration         `json:"response_times"`
	AlertEvents       []*AlertEvent           `json:"alert_events"`
	ValidationSummary *AlertValidationResults `json:"validation_summary"`
}

// Additional types that were in the main test file

// RunningExperiment represents an active chaos experiment
type RunningExperiment struct {
	Experiment       *ChaosExperiment
	StartTime        time.Time
	Status           ExperimentStatus
	Injectors        []*FailureInstance
	Observations     []*ChaosObservation
	SafetyViolations []*SafetyViolation
	mutex            sync.RWMutex
}

// FailureInjector injects various types of failures
type FailureInjector struct {
	injectors      map[ExperimentType]Injector
	activeFailures []*FailureInstance
	mutex          sync.RWMutex
}

// Injector interface for different failure types
type Injector interface {
	Inject(ctx context.Context, target ExperimentTarget, parameters map[string]interface{}) (*FailureInstance, error)
	Stop(ctx context.Context, instance *FailureInstance) error
	Validate(ctx context.Context, instance *FailureInstance) error
}

// FailureInstance represents an active failure injection
type FailureInstance struct {
	ID         string                 `json:"id"`
	Type       ExperimentType         `json:"type"`
	Target     ExperimentTarget       `json:"target"`
	Parameters map[string]interface{} `json:"parameters"`
	StartTime  time.Time              `json:"start_time"`
	Status     FailureStatus          `json:"status"`
	Impact     *FailureImpact         `json:"impact"`
}

// FailureStatus represents failure injection status
type FailureStatus string

const (
	FailureStatusActive  FailureStatus = "active"
	FailureStatusStopped FailureStatus = "stopped"
	FailureStatusFailed  FailureStatus = "failed"
)

// FailureImpact tracks the impact of failure injection
type FailureImpact struct {
	AffectedComponents []string          `json:"affected_components"`
	SLAMetrics         *SLAMetricsImpact `json:"sla_metrics"`
	RecoveryMetrics    *RecoveryMetrics  `json:"recovery_metrics"`
}

// SLAMetricsImpact tracks SLA metric changes during failure
type SLAMetricsImpact struct {
	AvailabilityBefore float64 `json:"availability_before"`
	AvailabilityAfter  float64 `json:"availability_after"`
	LatencyP95Before   float64 `json:"latency_p95_before"`
	LatencyP95After    float64 `json:"latency_p95_after"`
	ThroughputBefore   float64 `json:"throughput_before"`
	ThroughputAfter    float64 `json:"throughput_after"`
}

// RecoveryMetrics tracks recovery characteristics
type RecoveryMetrics struct {
	RecoveryStartTime    time.Time        `json:"recovery_start_time"`
	RecoveryCompleteTime time.Time        `json:"recovery_complete_time"`
	RecoveryDuration     time.Duration    `json:"recovery_duration"`
	RecoveryStages       []*RecoveryStage `json:"recovery_stages"`
}

// ResilienceValidator validates system resilience under chaos
type ResilienceValidator struct {
	slaMonitor     *SLAMonitor
	alertValidator *AlertValidator
	dataValidator  *DataConsistencyValidator
	config         *ResilienceConfig
}

// ResilienceConfig configures resilience validation
type ResilienceConfig struct {
	SLATolerances        map[string]float64
	AlertResponseTime    time.Duration
	DataConsistencyCheck time.Duration
	RecoveryValidation   time.Duration
}

// ChaosMonitor monitors system behavior during chaos experiments
type ChaosMonitor struct {
	metrics       *ChaosMetrics
	observations  []*ChaosObservation
	slaViolations []*SLAViolation
	alertEvents   []*AlertEvent
	mutex         sync.RWMutex
}

// ChaosMetrics tracks metrics during chaos experiments
type ChaosMetrics struct {
	AvailabilityTimeSeries []MetricPoint `json:"availability_time_series"`
	LatencyTimeSeries      []MetricPoint `json:"latency_time_series"`
	ThroughputTimeSeries   []MetricPoint `json:"throughput_time_series"`
	ErrorRateTimeSeries    []MetricPoint `json:"error_rate_time_series"`
	RecoveryTimeSeries     []MetricPoint `json:"recovery_time_series"`
}

// MetricPoint represents a metric measurement at a point in time
type MetricPoint struct {
	Timestamp time.Time         `json:"timestamp"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels"`
}

// ChaosObservation represents an observation during chaos
type ChaosObservation struct {
	Timestamp   time.Time              `json:"timestamp"`
	Type        ObservationType        `json:"type"`
	Component   string                 `json:"component"`
	Description string                 `json:"description"`
	Severity    ObservationSeverity    `json:"severity"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ObservationType defines types of chaos observations
type ObservationType string

const (
	ObservationTypeSLAViolation ObservationType = "sla_violation"
	ObservationTypeAlert        ObservationType = "alert"
	ObservationTypeRecovery     ObservationType = "recovery"
	ObservationTypeFailure      ObservationType = "failure"
	ObservationTypeAnomaly      ObservationType = "anomaly"
)

// ObservationSeverity defines observation severity
type ObservationSeverity string

const (
	SeverityInfo     ObservationSeverity = "info"
	SeverityWarning  ObservationSeverity = "warning"
	SeverityError    ObservationSeverity = "error"
	SeverityCritical ObservationSeverity = "critical"
)

// Additional types needed by test methods

// RecoveryScenario defines a recovery testing scenario
type RecoveryScenario struct {
	Name             string
	FailureType      ExperimentType
	ExpectedRecovery time.Duration
	Tolerance        time.Duration
}

// PartialFailureScenario defines a partial failure scenario
type PartialFailureScenario struct {
	Name          string
	AffectedNodes []string
	HealthyNodes  []string
	ExpectedSLA   map[string]float64
}

// Types that were originally in the main test file

// ChaosEngine orchestrates chaos experiments
type ChaosEngine struct {
	experiments       []*ChaosExperiment
	activeExperiments map[string]*RunningExperiment
	scheduler         *ExperimentScheduler
	config            *ChaosEngineConfig
	mutex             sync.RWMutex
	ctx               context.Context
}

// ChaosEngineConfig configures the chaos engine
type ChaosEngineConfig struct {
	MaxConcurrentExperiments int
	ExperimentTimeout        time.Duration
	SafetyChecks             bool
	RecoveryEnabled          bool
	MetricsCollection        bool
}

// ChaosTestConfig defines chaos testing configuration
type ChaosTestConfig struct {
	// SLA targets under chaos conditions
	AvailabilityTargetUnderChaos float64       `yaml:"availability_target_under_chaos"` // 99.90% (degraded but acceptable)
	LatencyP95TargetUnderChaos   time.Duration `yaml:"latency_p95_target_under_chaos"`  // 5 seconds (degraded)
	ThroughputTargetUnderChaos   float64       `yaml:"throughput_target_under_chaos"`   // 30 intents/minute (degraded)
	RecoveryTimeTarget           time.Duration `yaml:"recovery_time_target"`            // 5 minutes max recovery

	// Chaos experiment parameters
	ExperimentDuration      time.Duration `yaml:"experiment_duration"`       // 30 minutes per experiment
	RecoveryObservationTime time.Duration `yaml:"recovery_observation_time"` // 10 minutes after chaos ends
	ChaosIntensity          float64       `yaml:"chaos_intensity"`           // 0.3 (30% failure rate)
	ConcurrentExperiments   int           `yaml:"concurrent_experiments"`    // 3 experiments simultaneously

	// Failure scenarios
	ComponentFailureRate        float64 `yaml:"component_failure_rate"`        // 0.1 (10% components fail)
	NetworkPartitionProbability float64 `yaml:"network_partition_probability"` // 0.05 (5% chance)
	ResourceExhaustionRate      float64 `yaml:"resource_exhaustion_rate"`      // 0.1 (10% resource exhaustion)

	// Monitoring validation
	FalsePositiveThreshold   float64 `yaml:"false_positive_threshold"`   // <1% false positives
	AlertAccuracyThreshold   float64 `yaml:"alert_accuracy_threshold"`   // >95% alert accuracy
	DataConsistencyThreshold float64 `yaml:"data_consistency_threshold"` // >99% data consistency
}

// SLAChaosTestSuite provides chaos engineering validation for SLA monitoring under failure conditions
type SLAChaosTestSuite struct {
	suite.Suite

	// Test infrastructure
	ctx              context.Context
	cancel           context.CancelFunc
	slaService       *sla.Service
	prometheusClient v1.API
	logger           *logging.StructuredLogger

	// Chaos engineering components
	chaosEngine         *ChaosEngine
	failureInjector     *FailureInjector
	resilienceValidator *ResilienceValidator
	recoveryTracker     *RecoveryTracker

	// Configuration
	config *ChaosTestConfig

	// Monitoring during chaos
	chaosMonitor *ChaosMonitor
	slaTracker   *SLATracker

	// Test state
	chaosStartTime    time.Time
	activeExperiments atomic.Int64
	totalFailures     atomic.Int64
	recoveryEvents    atomic.Int64
}
