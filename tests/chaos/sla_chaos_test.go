//go:build integration

package chaos

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring/sla"
)

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

// ExperimentStatus represents experiment status
type ExperimentStatus string

const (
	ExperimentStatusRunning   ExperimentStatus = "running"
	ExperimentStatusCompleted ExperimentStatus = "completed"
	ExperimentStatusAborted   ExperimentStatus = "aborted"
	ExperimentStatusFailed    ExperimentStatus = "failed"
)

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

// RecoveryStage represents a stage in the recovery process
type RecoveryStage struct {
	Name      string             `json:"name"`
	StartTime time.Time          `json:"start_time"`
	EndTime   time.Time          `json:"end_time"`
	Duration  time.Duration      `json:"duration"`
	Success   bool               `json:"success"`
	Metrics   map[string]float64 `json:"metrics"`
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

// SetupTest initializes the chaos testing suite
func (s *SLAChaosTestSuite) SetupTest() {
	s.ctx, s.cancel = context.WithTimeout(context.Background(), 2*time.Hour)
	s.chaosStartTime = time.Now()

	// Initialize chaos test configuration
	s.config = &ChaosTestConfig{
		// SLA targets under chaos (degraded but acceptable)
		AvailabilityTargetUnderChaos: 99.90,           // Slight degradation allowed
		LatencyP95TargetUnderChaos:   5 * time.Second, // Higher latency acceptable
		ThroughputTargetUnderChaos:   30.0,            // Reduced throughput acceptable
		RecoveryTimeTarget:           5 * time.Minute, // Max 5 minutes recovery

		// Experiment parameters
		ExperimentDuration:      30 * time.Minute,
		RecoveryObservationTime: 10 * time.Minute,
		ChaosIntensity:          0.3, // 30% failure rate
		ConcurrentExperiments:   3,

		// Failure rates
		ComponentFailureRate:        0.1,  // 10% component failure
		NetworkPartitionProbability: 0.05, // 5% network issues
		ResourceExhaustionRate:      0.1,  // 10% resource issues

		// Quality thresholds
		FalsePositiveThreshold:   0.01, // <1% false positives
		AlertAccuracyThreshold:   0.95, // >95% alert accuracy
		DataConsistencyThreshold: 0.99, // >99% data consistency
	}

	// Initialize logger
	var err error
	s.logger, err = logging.NewStructuredLogger(&logging.Config{
		Level:      "info",
		Format:     "json",
		Component:  "sla-chaos-test",
		TraceLevel: "debug",
	})
	s.Require().NoError(err, "Failed to initialize logger")

	// Initialize Prometheus client
	client, err := api.NewClient(api.Config{
		Address: "http://localhost:9090",
	})
	s.Require().NoError(err, "Failed to create Prometheus client")
	s.prometheusClient = v1.NewAPI(client)

	// Initialize SLA service
	slaConfig := sla.DefaultServiceConfig()
	slaConfig.AvailabilityTarget = s.config.AvailabilityTargetUnderChaos
	slaConfig.P95LatencyTarget = s.config.LatencyP95TargetUnderChaos
	slaConfig.ThroughputTarget = s.config.ThroughputTargetUnderChaos

	appConfig := &config.Config{
		LogLevel: "info",
	}

	s.slaService, err = sla.NewService(slaConfig, appConfig, s.logger)
	s.Require().NoError(err, "Failed to initialize SLA service")

	// Start SLA service
	err = s.slaService.Start(s.ctx)
	s.Require().NoError(err, "Failed to start SLA service")

	// Initialize chaos engineering components
	s.chaosEngine = NewChaosEngine(&ChaosEngineConfig{
		MaxConcurrentExperiments: s.config.ConcurrentExperiments,
		ExperimentTimeout:        s.config.ExperimentDuration + 10*time.Minute,
		SafetyChecks:             true,
		RecoveryEnabled:          true,
		MetricsCollection:        true,
	}, s.ctx)

	s.failureInjector = NewFailureInjector()
	s.resilienceValidator = NewResilienceValidator(&ResilienceConfig{
		SLATolerances: map[string]float64{
			"availability": s.config.AvailabilityTargetUnderChaos,
			"latency":      s.config.LatencyP95TargetUnderChaos.Seconds(),
			"throughput":   s.config.ThroughputTargetUnderChaos,
		},
		AlertResponseTime:    30 * time.Second,
		DataConsistencyCheck: 1 * time.Minute,
		RecoveryValidation:   s.config.RecoveryTimeTarget,
	})

	s.recoveryTracker = NewRecoveryTracker()
	s.chaosMonitor = NewChaosMonitor()
	s.slaTracker = NewSLATracker(s.prometheusClient)

	// Configure chaos experiments
	s.configureChaosExperiments()

	// Wait for services to stabilize
	time.Sleep(10 * time.Second)
}

// TearDownTest cleans up after chaos tests
func (s *SLAChaosTestSuite) TearDownTest() {
	// Stop all active experiments
	if s.chaosEngine != nil {
		s.chaosEngine.StopAllExperiments(s.ctx)
	}

	// Stop SLA service
	if s.slaService != nil {
		err := s.slaService.Stop(s.ctx)
		s.Assert().NoError(err, "Failed to stop SLA service")
	}

	// Generate cleanup report
	s.generateChaosTestReport()

	if s.cancel != nil {
		s.cancel()
	}
}

// TestSLAMonitoringResilienceUnderChaos validates SLA monitoring resilience
func (s *SLAChaosTestSuite) TestSLAMonitoringResilienceUnderChaos() {
	s.T().Log("Testing SLA monitoring resilience under chaos conditions")

	ctx, cancel := context.WithTimeout(s.ctx, s.config.ExperimentDuration+s.config.RecoveryObservationTime)
	defer cancel()

	// Start baseline monitoring
	s.T().Log("Establishing baseline metrics")
	baseline := s.establishBaseline(ctx, 5*time.Minute)

	// Start chaos monitoring
	go s.chaosMonitor.StartMonitoring(ctx)

	// Run concurrent chaos experiments
	experiments := []string{"component_failure", "network_partition", "resource_exhaustion"}
	var wg sync.WaitGroup

	for _, experimentType := range experiments {
		wg.Add(1)
		go func(expType string) {
			defer wg.Done()
			s.runChaosExperiment(ctx, expType)
		}(experimentType)
	}

	// Monitor SLA compliance during chaos
	go s.monitorSLADuringChaos(ctx)

	// Wait for experiments to complete
	wg.Wait()

	// Recovery observation period
	s.T().Log("Observing recovery period")
	s.observeRecovery(ctx, s.config.RecoveryObservationTime)

	// Validate resilience
	s.validateChaosResilience(baseline)
}

// TestAlertingSystemResilienceDuringChaos validates alert system resilience
func (s *SLAChaosTestSuite) TestAlertingSystemResilienceDuringChaos() {
	s.T().Log("Testing alerting system resilience during chaos")

	ctx, cancel := context.WithTimeout(s.ctx, 45*time.Minute)
	defer cancel()

	// Start alert monitoring
	alertMonitor := NewAlertMonitor(s.prometheusClient)
	go alertMonitor.StartMonitoring(ctx)

	// Inject cascading failure
	s.T().Log("Injecting cascading failure scenario")
	cascadingExperiment := s.createCascadingFailureExperiment()

	// Run experiment with alert validation
	results := s.runExperimentWithAlertValidation(ctx, cascadingExperiment, alertMonitor)

	// Validate alert accuracy
	s.validateAlertAccuracy(results)
	s.validateFalsePositiveRate(results)
	s.validateAlertResponseTime(results)
}

// TestDataConsistencyUnderChaos validates data consistency during failures
func (s *SLAChaosTestSuite) TestDataConsistencyUnderChaos() {
	s.T().Log("Testing data consistency under chaos conditions")

	ctx, cancel := context.WithTimeout(s.ctx, 1*time.Hour)
	defer cancel()

	// Start data consistency monitoring
	consistencyMonitor := NewDataConsistencyMonitor()
	go consistencyMonitor.StartMonitoring(ctx)

	// Run data corruption experiments
	dataExperiments := []string{"prometheus_corruption", "metric_loss", "timestamp_skew"}

	for _, experiment := range dataExperiments {
		s.T().Run(experiment, func(t *testing.T) {
			s.runDataConsistencyExperiment(ctx, experiment, consistencyMonitor)
		})
	}

	// Validate overall data consistency
	consistency := consistencyMonitor.GetConsistencyScore()
	s.T().Logf("Overall data consistency score: %.2f%%", consistency*100)

	s.Assert().GreaterOrEqual(consistency, s.config.DataConsistencyThreshold,
		"Data consistency below threshold: %.2f%% < %.2f%%",
		consistency*100, s.config.DataConsistencyThreshold*100)
}

// TestRecoveryMeasurementAccuracy validates recovery time measurement accuracy
func (s *SLAChaosTestSuite) TestRecoveryMeasurementAccuracy() {
	s.T().Log("Testing recovery measurement accuracy")

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Minute)
	defer cancel()

	// Define controlled failure scenarios with known recovery times
	scenarios := []*RecoveryScenario{
		{
			Name:             "quick_recovery",
			FailureType:      ExperimentTypeComponentFailure,
			ExpectedRecovery: 30 * time.Second,
			Tolerance:        10 * time.Second,
		},
		{
			Name:             "medium_recovery",
			FailureType:      ExperimentTypeNetworkPartition,
			ExpectedRecovery: 2 * time.Minute,
			Tolerance:        30 * time.Second,
		},
		{
			Name:             "slow_recovery",
			FailureType:      ExperimentTypeResourceExhaustion,
			ExpectedRecovery: 5 * time.Minute,
			Tolerance:        1 * time.Minute,
		},
	}

	for _, scenario := range scenarios {
		s.T().Run(scenario.Name, func(t *testing.T) {
			s.validateRecoveryMeasurement(ctx, scenario)
		})
	}
}

// TestPartialFailureHandling tests monitoring during partial system failures
func (s *SLAChaosTestSuite) TestPartialFailureHandling() {
	s.T().Log("Testing monitoring under partial failure conditions")

	ctx, cancel := context.WithTimeout(s.ctx, 45*time.Minute)
	defer cancel()

	// Simulate partial failures (some components down, others up)
	partialFailureScenarios := []PartialFailureScenario{
		{
			Name:          "prometheus_partial",
			AffectedNodes: []string{"prometheus-0"},
			HealthyNodes:  []string{"prometheus-1", "prometheus-2"},
			ExpectedSLA:   map[string]float64{"availability": 99.5},
		},
		{
			Name:          "grafana_partial",
			AffectedNodes: []string{"grafana-dashboard"},
			HealthyNodes:  []string{"grafana-api"},
			ExpectedSLA:   map[string]float64{"availability": 99.8},
		},
		{
			Name:          "sla_service_partial",
			AffectedNodes: []string{"sla-collector"},
			HealthyNodes:  []string{"sla-calculator", "sla-alerting"},
			ExpectedSLA:   map[string]float64{"availability": 99.0},
		},
	}

	for _, scenario := range partialFailureScenarios {
		s.T().Run(scenario.Name, func(t *testing.T) {
			s.testPartialFailureScenario(ctx, scenario)
		})
	}
}

// TestFalsePositiveDetection validates false positive detection during chaos
func (s *SLAChaosTestSuite) TestFalsePositiveDetection() {
	s.T().Log("Testing false positive detection during chaos experiments")

	ctx, cancel := context.WithTimeout(s.ctx, 1*time.Hour)
	defer cancel()

	// Run chaos experiments with false positive tracking
	falsePositiveTracker := NewFalsePositiveTracker()

	// Generate benign load variations that shouldn't trigger alerts
	go s.generateBenignLoadVariations(ctx)

	// Monitor for false positive alerts
	go falsePositiveTracker.MonitorAlerts(ctx, s.prometheusClient)

	// Run actual chaos experiments
	chaosResults := s.runMixedChaosExperiments(ctx)

	// Analyze false positive rate
	falsePositiveRate := falsePositiveTracker.CalculateFalsePositiveRate()

	s.T().Logf("False positive analysis:")
	s.T().Logf("  Total alerts: %d", falsePositiveTracker.GetTotalAlerts())
	s.T().Logf("  False positives: %d", falsePositiveTracker.GetFalsePositives())
	s.T().Logf("  False positive rate: %.2f%%", falsePositiveRate*100)

	s.Assert().Less(falsePositiveRate, s.config.FalsePositiveThreshold,
		"False positive rate too high: %.2f%% >= %.2f%%",
		falsePositiveRate*100, s.config.FalsePositiveThreshold*100)
}

// Helper methods for chaos experiment implementation

// runChaosExperiment runs a specific chaos experiment
func (s *SLAChaosTestSuite) runChaosExperiment(ctx context.Context, experimentType string) {
	s.T().Logf("Running chaos experiment: %s", experimentType)

	experiment := s.getChaosExperiment(experimentType)
	if experiment == nil {
		s.T().Errorf("Unknown experiment type: %s", experimentType)
		return
	}

	// Start experiment
	runningExperiment, err := s.chaosEngine.StartExperiment(ctx, experiment)
	if err != nil {
		s.T().Errorf("Failed to start experiment %s: %v", experimentType, err)
		return
	}

	s.activeExperiments.Add(1)
	defer s.activeExperiments.Add(-1)

	// Monitor experiment
	s.monitorExperiment(ctx, runningExperiment)

	// Wait for completion or timeout
	select {
	case <-ctx.Done():
		s.T().Logf("Experiment %s interrupted by context", experimentType)
	case <-time.After(experiment.Duration):
		s.T().Logf("Experiment %s completed normally", experimentType)
	}

	// Stop experiment
	err = s.chaosEngine.StopExperiment(ctx, runningExperiment.Experiment.ID)
	if err != nil {
		s.T().Errorf("Failed to stop experiment %s: %v", experimentType, err)
	}

	// Record results
	s.recordExperimentResults(runningExperiment)
}

// configureChaosExperiments sets up the chaos experiments
func (s *SLAChaosTestSuite) configureChaosExperiments() {
	experiments := []*ChaosExperiment{
		{
			ID:          "component_failure_001",
			Name:        "Component Failure",
			Description: "Inject failures into key components",
			Type:        ExperimentTypeComponentFailure,
			Target: ExperimentTarget{
				Component:  "sla-service",
				Percentage: s.config.ComponentFailureRate,
			},
			Duration: s.config.ExperimentDuration,
			ExpectedBehavior: &ExpectedBehavior{
				SLAImpact: &SLAImpactExpectation{
					AvailabilityDrop: 0.05, // 0.05% drop expected
					LatencyIncrease:  0.10, // 10% latency increase
					ThroughputDrop:   0.05, // 5% throughput drop
				},
				RecoveryTime:   2 * time.Minute,
				AlertsExpected: []string{"ComponentDown", "SLAViolation"},
			},
			SafetyChecks: []*SafetyCheck{
				{
					Name:      "availability_floor",
					Type:      SafetyCheckAvailability,
					Threshold: 99.0,
					Action:    SafetyActionAbort,
				},
			},
		},
		{
			ID:          "network_partition_001",
			Name:        "Network Partition",
			Description: "Simulate network partitions",
			Type:        ExperimentTypeNetworkPartition,
			Target: ExperimentTarget{
				Component:  "prometheus",
				Percentage: s.config.NetworkPartitionProbability,
			},
			Duration: s.config.ExperimentDuration,
			ExpectedBehavior: &ExpectedBehavior{
				SLAImpact: &SLAImpactExpectation{
					AvailabilityDrop: 0.02,
					LatencyIncrease:  0.20,
					ThroughputDrop:   0.10,
				},
				RecoveryTime:   3 * time.Minute,
				AlertsExpected: []string{"NetworkPartition", "MetricsCollectionDown"},
			},
		},
		{
			ID:          "resource_exhaustion_001",
			Name:        "Resource Exhaustion",
			Description: "Exhaust system resources",
			Type:        ExperimentTypeResourceExhaustion,
			Target: ExperimentTarget{
				Component:  "monitoring-stack",
				Percentage: s.config.ResourceExhaustionRate,
			},
			Duration: s.config.ExperimentDuration,
			ExpectedBehavior: &ExpectedBehavior{
				SLAImpact: &SLAImpactExpectation{
					AvailabilityDrop: 0.03,
					LatencyIncrease:  0.50,
					ThroughputDrop:   0.20,
				},
				RecoveryTime:   5 * time.Minute,
				AlertsExpected: []string{"HighMemoryUsage", "CPUThrottling"},
			},
		},
	}

	for _, exp := range experiments {
		s.chaosEngine.RegisterExperiment(exp)
	}
}

// Additional helper classes and methods...

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

// Constructor functions for chaos components
func NewChaosEngine(config *ChaosEngineConfig, ctx context.Context) *ChaosEngine {
	return &ChaosEngine{
		experiments:       make([]*ChaosExperiment, 0),
		activeExperiments: make(map[string]*RunningExperiment),
		config:            config,
		ctx:               ctx,
	}
}

func NewFailureInjector() *FailureInjector {
	return &FailureInjector{
		injectors:      make(map[ExperimentType]Injector),
		activeFailures: make([]*FailureInstance, 0),
	}
}

func NewResilienceValidator(config *ResilienceConfig) *ResilienceValidator {
	return &ResilienceValidator{
		config: config,
	}
}

func NewRecoveryTracker() *RecoveryTracker {
	return &RecoveryTracker{
		recoveryEvents: make(map[string]*RecoveryEvent),
	}
}

func NewChaosMonitor() *ChaosMonitor {
	return &ChaosMonitor{
		metrics:       &ChaosMetrics{},
		observations:  make([]*ChaosObservation, 0),
		slaViolations: make([]*SLAViolation, 0),
		alertEvents:   make([]*AlertEvent, 0),
	}
}

// Additional method implementations would continue here...

// TestSuite runner function
func TestSLAChaosTestSuite(t *testing.T) {
	suite.Run(t, new(SLAChaosTestSuite))
}
