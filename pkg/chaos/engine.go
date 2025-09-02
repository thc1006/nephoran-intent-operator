// Package chaos provides comprehensive chaos engineering capabilities for validating.

// system resilience and SLA compliance under failure conditions.

package chaos

import (
	
	"encoding/json"
"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ExperimentStatus represents the current state of a chaos experiment.

type ExperimentStatus string

const (

	// StatusPending holds statuspending value.

	StatusPending ExperimentStatus = "Pending"

	// StatusRunning holds statusrunning value.

	StatusRunning ExperimentStatus = "Running"

	// StatusCompleted holds statuscompleted value.

	StatusCompleted ExperimentStatus = "Completed"

	// StatusFailed holds statusfailed value.

	StatusFailed ExperimentStatus = "Failed"

	// StatusRolledBack holds statusrolledback value.

	StatusRolledBack ExperimentStatus = "RolledBack"

	// StatusAborted holds statusaborted value.

	StatusAborted ExperimentStatus = "Aborted"
)

// SafetyLevel defines the safety constraints for experiments.

type SafetyLevel string

const (

	// SafetyLevelLow holds safetylevellow value.

	SafetyLevelLow SafetyLevel = "Low" // Minimal constraints

	// SafetyLevelMedium holds safetylevelmedium value.

	SafetyLevelMedium SafetyLevel = "Medium" // Standard production constraints

	// SafetyLevelHigh holds safetylevelhigh value.

	SafetyLevelHigh SafetyLevel = "High" // Maximum safety, minimal impact

)

// BlastRadius defines the scope of impact for experiments.

type BlastRadius struct {
	Namespaces []string // Affected namespaces

	Services []string // Affected services

	MaxPods int // Maximum number of pods to affect

	MaxNodes int // Maximum number of nodes to affect

	TrafficPercent float64 // Percentage of traffic to affect

	Duration time.Duration // Maximum duration of impact
}

// SLAThresholds defines acceptable SLA impact during experiments.

type SLAThresholds struct {
	MaxLatencyMS int64 // Maximum acceptable latency in milliseconds

	MinAvailability float64 // Minimum acceptable availability percentage

	MinThroughput float64 // Minimum acceptable throughput (intents/minute)

	MaxErrorRate float64 // Maximum acceptable error rate percentage

	AutoRollbackEnable bool // Enable automatic rollback on threshold breach
}

// Experiment represents a chaos experiment configuration.

type Experiment struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Description string `json:"description"`

	Type ExperimentType `json:"type"`

	Target ExperimentTarget `json:"target"`

	Parameters map[string]string `json:"parameters"`

	SafetyLevel SafetyLevel `json:"safetyLevel"`

	BlastRadius BlastRadius `json:"blastRadius"`

	SLAThresholds SLAThresholds `json:"slaThresholds"`

	Schedule *Schedule `json:"schedule,omitempty"`

	Duration time.Duration `json:"duration"`

	Status ExperimentStatus `json:"status"`

	StartTime *time.Time `json:"startTime,omitempty"`

	EndTime *time.Time `json:"endTime,omitempty"`

	Results *ExperimentResult `json:"results,omitempty"`

	RollbackOnFail bool `json:"rollbackOnFail"`

	DryRun bool `json:"dryRun"`
}

// ExperimentType defines the type of chaos experiment.

type ExperimentType string

const (

	// ExperimentTypeNetwork holds experimenttypenetwork value.

	ExperimentTypeNetwork ExperimentType = "Network"

	// ExperimentTypePod holds experimenttypepod value.

	ExperimentTypePod ExperimentType = "Pod"

	// ExperimentTypeResource holds experimenttyperesource value.

	ExperimentTypeResource ExperimentType = "Resource"

	// ExperimentTypeDatabase holds experimenttypedatabase value.

	ExperimentTypeDatabase ExperimentType = "Database"

	// ExperimentTypeExternal holds experimenttypeexternal value.

	ExperimentTypeExternal ExperimentType = "External"

	// ExperimentTypeLoad holds experimenttypeload value.

	ExperimentTypeLoad ExperimentType = "Load"

	// ExperimentTypeDependency holds experimenttypedependency value.

	ExperimentTypeDependency ExperimentType = "Dependency"

	// ExperimentTypeComposite holds experimenttypecomposite value.

	ExperimentTypeComposite ExperimentType = "Composite"
)

// ExperimentTarget specifies what to target in the experiment.

type ExperimentTarget struct {
	Namespace string `json:"namespace"`

	LabelSelector map[string]string `json:"labelSelector"`

	Services []string `json:"services"`

	Pods []string `json:"pods"`

	Nodes []string `json:"nodes"`
}

// Schedule defines when experiments should run.

type Schedule struct {
	Type ScheduleType `json:"type"`

	Expression string `json:"expression"` // Cron expression or interval

	TimeWindow TimeWindow `json:"timeWindow"`
}

// ScheduleType defines how experiments are scheduled.

type ScheduleType string

const (

	// ScheduleTypeOnce holds scheduletypeonce value.

	ScheduleTypeOnce ScheduleType = "Once"

	// ScheduleTypeCron holds scheduletypecron value.

	ScheduleTypeCron ScheduleType = "Cron"

	// ScheduleTypeInterval holds scheduletypeinterval value.

	ScheduleTypeInterval ScheduleType = "Interval"

	// ScheduleTypeContinuous holds scheduletypecontinuous value.

	ScheduleTypeContinuous ScheduleType = "Continuous"
)

// TimeWindow defines when experiments are allowed to run.

type TimeWindow struct {
	StartHour int `json:"startHour"`

	EndHour int `json:"endHour"`

	Weekdays []string `json:"weekdays"`

	Timezone string `json:"timezone"`
}

// ExperimentResult captures the outcome of an experiment.

type ExperimentResult struct {
	Success bool `json:"success"`

	SLAImpact SLAImpactMetrics `json:"slaImpact"`

	RecoveryMetrics RecoveryMetrics `json:"recoveryMetrics"`

	Observations []string `json:"observations"`

	Recommendations []string `json:"recommendations"`

	FailureDetails string `json:"failureDetails,omitempty"`

	RollbackRequired bool `json:"rollbackRequired"`

	RollbackSuccess bool `json:"rollbackSuccess"`

	Artifacts json.RawMessage `json:"artifacts"`
}

// SLAImpactMetrics tracks SLA impact during experiments.

type SLAImpactMetrics struct {
	AvailabilityImpact float64 `json:"availabilityImpact"`

	LatencyP50Impact time.Duration `json:"latencyP50Impact"`

	LatencyP95Impact time.Duration `json:"latencyP95Impact"`

	LatencyP99Impact time.Duration `json:"latencyP99Impact"`

	ThroughputImpact float64 `json:"throughputImpact"`

	ErrorRateImpact float64 `json:"errorRateImpact"`

	AffectedUsers int `json:"affectedUsers"`

	DataConsistencyCheck bool `json:"dataConsistencyCheck"`
}

// RecoveryMetrics tracks recovery behavior.

type RecoveryMetrics struct {
	MTTR time.Duration `json:"mttr"`

	AutoRecoveryTriggered bool `json:"autoRecoveryTriggered"`

	AutoRecoverySuccess bool `json:"autoRecoverySuccess"`

	ManualIntervention bool `json:"manualIntervention"`

	DataLoss bool `json:"dataLoss"`

	ServiceDegradation []string `json:"serviceDegradation"`
}

// ChaosEngine orchestrates chaos experiments with safety controls.

type ChaosEngine struct {
	client client.Client

	kubeClient kubernetes.Interface

	logger *zap.Logger

	experiments map[string]*Experiment

	activeExperiments sync.Map

	injector *FailureInjector

	validator *ResilienceValidator

	recoveryTester *RecoveryTester

	killSwitch *KillSwitch

	metrics *ChaosMetrics

	config *EngineConfig

	mu sync.RWMutex

	ctx context.Context

	cancel context.CancelFunc
}

// EngineConfig defines chaos engine configuration.

type EngineConfig struct {
	MaxConcurrentExperiments int `json:"maxConcurrentExperiments"`

	DefaultSafetyLevel SafetyLevel `json:"defaultSafetyLevel"`

	DefaultDuration time.Duration `json:"defaultDuration"`

	MaxDuration time.Duration `json:"maxDuration"`

	ProductionMode bool `json:"productionMode"`

	AutoRollback bool `json:"autoRollback"`

	MetricsInterval time.Duration `json:"metricsInterval"`

	AlertThreshold float64 `json:"alertThreshold"`

	DryRunMode bool `json:"dryRunMode"`
}

// KillSwitch provides emergency experiment termination.

type KillSwitch struct {
	enabled bool

	triggered bool

	reason string

	mu sync.RWMutex
}

// ChaosMetrics tracks chaos engineering metrics.

type ChaosMetrics struct {
	experimentsTotal *prometheus.CounterVec

	experimentsActive prometheus.Gauge

	experimentsDuration *prometheus.HistogramVec

	slaViolations *prometheus.CounterVec

	recoveryTime *prometheus.HistogramVec

	rollbacksTotal *prometheus.CounterVec
}

// NewChaosEngine creates a new chaos engineering orchestration engine.

func NewChaosEngine(
	client client.Client,

	kubeClient kubernetes.Interface,

	logger *zap.Logger,

	config *EngineConfig,
) *ChaosEngine {
	ctx, cancel := context.WithCancel(context.Background())

	if config == nil {
		config = &EngineConfig{
			MaxConcurrentExperiments: 3,

			DefaultSafetyLevel: SafetyLevelMedium,

			DefaultDuration: 5 * time.Minute,

			MaxDuration: 30 * time.Minute,

			ProductionMode: false,

			AutoRollback: true,

			MetricsInterval: 10 * time.Second,

			AlertThreshold: 0.95,

			DryRunMode: false,
		}
	}

	return &ChaosEngine{
		client: client,

		kubeClient: kubeClient,

		logger: logger,

		experiments: make(map[string]*Experiment),

		injector: NewFailureInjector(client, kubeClient, logger),

		validator: NewResilienceValidator(client, logger),

		recoveryTester: NewRecoveryTester(client, kubeClient, logger),

		killSwitch: &KillSwitch{enabled: true},

		metrics: initMetrics(),

		config: config,

		ctx: ctx,

		cancel: cancel,
	}
}

// initMetrics initializes Prometheus metrics.

func initMetrics() *ChaosMetrics {
	return &ChaosMetrics{
		experimentsTotal: prometheus.NewCounterVec(

			prometheus.CounterOpts{
				Name: "chaos_experiments_total",

				Help: "Total number of chaos experiments executed",
			},

			[]string{"type", "status", "safety_level"},
		),

		experimentsActive: prometheus.NewGauge(

			prometheus.GaugeOpts{
				Name: "chaos_experiments_active",

				Help: "Number of currently active chaos experiments",
			},
		),

		experimentsDuration: prometheus.NewHistogramVec(

			prometheus.HistogramOpts{
				Name: "chaos_experiment_duration_seconds",

				Help: "Duration of chaos experiments in seconds",

				Buckets: prometheus.ExponentialBuckets(1, 2, 10),
			},

			[]string{"type", "safety_level"},
		),

		slaViolations: prometheus.NewCounterVec(

			prometheus.CounterOpts{
				Name: "chaos_sla_violations_total",

				Help: "Total number of SLA violations during chaos experiments",
			},

			[]string{"type", "metric"},
		),

		recoveryTime: prometheus.NewHistogramVec(

			prometheus.HistogramOpts{
				Name: "chaos_recovery_time_seconds",

				Help: "Time to recover from injected failures",

				Buckets: prometheus.ExponentialBuckets(1, 2, 10),
			},

			[]string{"type", "auto_recovery"},
		),

		rollbacksTotal: prometheus.NewCounterVec(

			prometheus.CounterOpts{
				Name: "chaos_rollbacks_total",

				Help: "Total number of experiment rollbacks",
			},

			[]string{"type", "reason"},
		),
	}
}

// RunExperiment executes a chaos experiment with safety controls.

func (e *ChaosEngine) RunExperiment(ctx context.Context, experiment *Experiment) (*ExperimentResult, error) {
	e.logger.Info("Starting chaos experiment",

		zap.String("id", experiment.ID),

		zap.String("name", experiment.Name),

		zap.String("type", string(experiment.Type)))

	// Check if kill switch is triggered.

	if e.killSwitch.IsTriggered() {
		return nil, fmt.Errorf("kill switch triggered: %s", e.killSwitch.GetReason())
	}

	// Validate experiment configuration.

	if err := e.validateExperiment(experiment); err != nil {
		return nil, fmt.Errorf("experiment validation failed: %w", err)
	}

	// Check concurrent experiment limits.

	if !e.canRunExperiment() {
		return nil, fmt.Errorf("maximum concurrent experiments reached")
	}

	// Apply safety constraints based on environment.

	e.applySafetyConstraints(experiment)

	// Register experiment.

	e.registerExperiment(experiment)

	defer e.unregisterExperiment(experiment.ID)

	// Set up monitoring for the experiment.

	monitorCtx, monitorCancel := context.WithCancel(ctx)

	defer monitorCancel()

	monitoringChan := e.startMonitoring(monitorCtx, experiment)

	// Execute experiment with timeout.

	experimentCtx, experimentCancel := context.WithTimeout(ctx, experiment.Duration)

	defer experimentCancel()

	// Track experiment start.

	startTime := time.Now()

	experiment.StartTime = &startTime

	experiment.Status = StatusRunning

	e.metrics.experimentsActive.Inc()

	// Execute based on experiment type.

	var result *ExperimentResult

	var err error

	if experiment.DryRun {
		result = e.simulateExperiment(experimentCtx, experiment)
	} else {
		result = e.executeExperiment(experimentCtx, experiment, monitoringChan)
	}

	// Track experiment completion.

	endTime := time.Now()

	experiment.EndTime = &endTime

	duration := endTime.Sub(startTime)

	e.metrics.experimentsActive.Dec()

	e.metrics.experimentsDuration.WithLabelValues(

		string(experiment.Type),

		string(experiment.SafetyLevel),
	).Observe(duration.Seconds())

	// Update experiment status.

	if result.Success {
		experiment.Status = StatusCompleted
	} else if result.RollbackRequired {
		experiment.Status = StatusRolledBack
	} else {
		experiment.Status = StatusFailed
	}

	e.metrics.experimentsTotal.WithLabelValues(

		string(experiment.Type),

		string(experiment.Status),

		string(experiment.SafetyLevel),
	).Inc()

	// Generate recommendations based on results.

	e.generateRecommendations(result, experiment)

	experiment.Results = result

	return result, err
}

// validateExperiment validates experiment configuration.

func (e *ChaosEngine) validateExperiment(experiment *Experiment) error {
	// Validate basic configuration.

	if experiment.ID == "" {
		experiment.ID = generateExperimentID()
	}

	if experiment.Duration == 0 {
		experiment.Duration = e.config.DefaultDuration
	}

	if experiment.Duration > e.config.MaxDuration {
		return fmt.Errorf("experiment duration %v exceeds maximum %v",

			experiment.Duration, e.config.MaxDuration)
	}

	// Validate blast radius.

	if err := e.validateBlastRadius(&experiment.BlastRadius); err != nil {
		return fmt.Errorf("invalid blast radius: %w", err)
	}

	// Validate SLA thresholds.

	if err := e.validateSLAThresholds(&experiment.SLAThresholds); err != nil {
		return fmt.Errorf("invalid SLA thresholds: %w", err)
	}

	// Production mode validations.

	if e.config.ProductionMode {

		if experiment.SafetyLevel == SafetyLevelLow {
			return fmt.Errorf("low safety level not allowed in production mode")
		}

		if !experiment.RollbackOnFail {
			e.logger.Warn("Rollback disabled in production mode",

				zap.String("experiment", experiment.ID))
		}

	}

	return nil
}

// validateBlastRadius validates blast radius configuration.

func (e *ChaosEngine) validateBlastRadius(radius *BlastRadius) error {
	if radius.MaxPods < 0 || radius.MaxNodes < 0 {
		return fmt.Errorf("negative values not allowed in blast radius")
	}

	if radius.TrafficPercent < 0 || radius.TrafficPercent > 100 {
		return fmt.Errorf("traffic percent must be between 0 and 100")
	}

	if radius.Duration > e.config.MaxDuration {
		return fmt.Errorf("blast radius duration exceeds maximum")
	}

	return nil
}

// validateSLAThresholds validates SLA threshold configuration.

func (e *ChaosEngine) validateSLAThresholds(thresholds *SLAThresholds) error {
	if thresholds.MaxLatencyMS < 0 {
		return fmt.Errorf("negative latency threshold not allowed")
	}

	if thresholds.MinAvailability < 0 || thresholds.MinAvailability > 100 {
		return fmt.Errorf("availability must be between 0 and 100")
	}

	if thresholds.MinThroughput < 0 {
		return fmt.Errorf("negative throughput threshold not allowed")
	}

	if thresholds.MaxErrorRate < 0 || thresholds.MaxErrorRate > 100 {
		return fmt.Errorf("error rate must be between 0 and 100")
	}

	return nil
}

// canRunExperiment checks if a new experiment can be started.

func (e *ChaosEngine) canRunExperiment() bool {
	count := 0

	e.activeExperiments.Range(func(key, value interface{}) bool {
		count++

		return true
	})

	return count < e.config.MaxConcurrentExperiments
}

// applySafetyConstraints applies safety constraints based on environment.

func (e *ChaosEngine) applySafetyConstraints(experiment *Experiment) {
	switch experiment.SafetyLevel {

	case SafetyLevelHigh:

		// Maximum safety constraints.

		if experiment.BlastRadius.MaxPods > 1 {
			experiment.BlastRadius.MaxPods = 1
		}

		if experiment.BlastRadius.TrafficPercent > 10 {
			experiment.BlastRadius.TrafficPercent = 10
		}

		experiment.RollbackOnFail = true

		experiment.SLAThresholds.AutoRollbackEnable = true

	case SafetyLevelMedium:

		// Standard safety constraints.

		if experiment.BlastRadius.MaxPods > 5 {
			experiment.BlastRadius.MaxPods = 5
		}

		if experiment.BlastRadius.TrafficPercent > 25 {
			experiment.BlastRadius.TrafficPercent = 25
		}

	case SafetyLevelLow:

		// Minimal constraints (not allowed in production).

		if e.config.ProductionMode {

			experiment.SafetyLevel = SafetyLevelMedium

			e.applySafetyConstraints(experiment)

		}

	}
}

// registerExperiment registers an experiment as active.

func (e *ChaosEngine) registerExperiment(experiment *Experiment) {
	e.mu.Lock()

	defer e.mu.Unlock()

	e.experiments[experiment.ID] = experiment

	e.activeExperiments.Store(experiment.ID, experiment)
}

// unregisterExperiment removes an experiment from active list.

func (e *ChaosEngine) unregisterExperiment(id string) {
	e.activeExperiments.Delete(id)
}

// startMonitoring starts monitoring for SLA violations during experiment.

func (e *ChaosEngine) startMonitoring(ctx context.Context, experiment *Experiment) <-chan SLAViolation {
	violationChan := make(chan SLAViolation, 10)

	go func() {
		ticker := time.NewTicker(e.config.MetricsInterval)

		defer ticker.Stop()

		defer close(violationChan)

		for {
			select {

			case <-ctx.Done():

				return

			case <-ticker.C:

				violations := e.validator.CheckSLACompliance(ctx, experiment.SLAThresholds)

				for _, violation := range violations {
					select {

					case violationChan <- violation:

						e.handleSLAViolation(ctx, experiment, violation)

					case <-ctx.Done():

						return

					}
				}

			}
		}
	}()

	return violationChan
}

// handleSLAViolation handles SLA violations during experiments.

func (e *ChaosEngine) handleSLAViolation(ctx context.Context, experiment *Experiment, violation SLAViolation) {
	e.logger.Warn("SLA violation detected",

		zap.String("experiment", experiment.ID),

		zap.String("metric", violation.Metric),

		zap.Float64("threshold", violation.Threshold),

		zap.Float64("actual", violation.ActualValue))

	e.metrics.slaViolations.WithLabelValues(

		string(experiment.Type),

		violation.Metric,
	).Inc()

	// Auto-rollback if enabled and threshold breached.

	if experiment.SLAThresholds.AutoRollbackEnable && violation.Severity == "Critical" {

		e.logger.Info("Triggering auto-rollback due to SLA violation",

			zap.String("experiment", experiment.ID))

		if err := e.rollbackExperiment(ctx, experiment); err != nil {
			e.logger.Error("Failed to rollback experiment",

				zap.String("experiment", experiment.ID),

				zap.Error(err))
		}

	}
}

// executeExperiment executes the actual chaos experiment.

func (e *ChaosEngine) executeExperiment(
	ctx context.Context,

	experiment *Experiment,

	monitoringChan <-chan SLAViolation,
) *ExperimentResult {
	result := &ExperimentResult{
		Success: true,

		Artifacts: make(map[string]interface{}),
	}

	// Pre-experiment validation.

	preCheckResult := e.validator.PreExperimentValidation(ctx, experiment)

	if !preCheckResult.Ready {

		result.Success = false

		result.FailureDetails = "Pre-experiment validation failed: " + preCheckResult.Reason

		return result

	}

	// Inject failures based on experiment type.

	injectionResult, err := e.injector.InjectFailure(ctx, experiment)
	if err != nil {

		result.Success = false

		result.FailureDetails = fmt.Sprintf("Failure injection failed: %v", err)

		result.RollbackRequired = true

		return result

	}

	// Monitor experiment execution.

	experimentComplete := make(chan bool)

	go func() {
		select {

		case <-ctx.Done():

			experimentComplete <- false

		case <-time.After(experiment.Duration):

			experimentComplete <- true

		}
	}()

	// Collect metrics during experiment.

	metricsCollector := e.startMetricsCollection(ctx, experiment)

	// Wait for experiment completion or interruption.

	select {

	case completed := <-experimentComplete:

		if !completed {

			result.Success = false

			result.FailureDetails = "Experiment interrupted"

			result.RollbackRequired = true

		}

	case violation := <-monitoringChan:

		if violation.Severity == "Critical" {

			result.Success = false

			result.FailureDetails = fmt.Sprintf("Critical SLA violation: %s", violation.Metric)

			result.RollbackRequired = true

		}

	}

	// Stop failure injection.

	if err := e.injector.StopFailure(ctx, injectionResult.InjectionID); err != nil {
		e.logger.Error("Failed to stop failure injection",

			zap.String("experiment", experiment.ID),

			zap.Error(err))
	}

	// Collect final metrics.

	result.SLAImpact = metricsCollector.GetImpactMetrics()

	// Test recovery.

	recoveryStart := time.Now()

	recoveryResult := e.recoveryTester.TestRecovery(ctx, experiment, injectionResult)

	recoveryDuration := time.Since(recoveryStart)

	result.RecoveryMetrics = RecoveryMetrics{
		MTTR: recoveryDuration,

		AutoRecoveryTriggered: recoveryResult.AutoRecoveryTriggered,

		AutoRecoverySuccess: recoveryResult.Success,

		ManualIntervention: recoveryResult.ManualRequired,

		DataLoss: recoveryResult.DataLoss,

		ServiceDegradation: recoveryResult.DegradedServices,
	}

	e.metrics.recoveryTime.WithLabelValues(

		string(experiment.Type),

		fmt.Sprintf("%v", recoveryResult.AutoRecoveryTriggered),
	).Observe(recoveryDuration.Seconds())

	// Post-experiment validation.

	postCheckResult := e.validator.PostExperimentValidation(ctx, experiment, result)

	if !postCheckResult.Success {
		result.Observations = append(result.Observations,

			"Post-experiment validation issues: "+postCheckResult.Issues)
	}

	return result
}

// simulateExperiment simulates an experiment in dry-run mode.

func (e *ChaosEngine) simulateExperiment(ctx context.Context, experiment *Experiment) *ExperimentResult {
	e.logger.Info("Simulating experiment in dry-run mode",

		zap.String("id", experiment.ID))

	// Simulate expected impact based on experiment type and parameters.

	result := &ExperimentResult{
		Success: true,

		SLAImpact: SLAImpactMetrics{
			AvailabilityImpact: e.simulateAvailabilityImpact(experiment),

			LatencyP95Impact: e.simulateLatencyImpact(experiment),

			ThroughputImpact: e.simulateThroughputImpact(experiment),

			ErrorRateImpact: e.simulateErrorRateImpact(experiment),
		},

		RecoveryMetrics: RecoveryMetrics{
			MTTR: e.simulateMTTR(experiment),

			AutoRecoveryTriggered: true,

			AutoRecoverySuccess: true,
		},

		Observations: []string{
			"Dry-run simulation completed",

			fmt.Sprintf("Expected availability impact: %.2f%%", e.simulateAvailabilityImpact(experiment)),

			fmt.Sprintf("Expected latency impact: %v", e.simulateLatencyImpact(experiment)),
		},

		Artifacts: make(map[string]interface{}),
	}

	result.Artifacts["simulation"] = true

	result.Artifacts["parameters"] = experiment.Parameters

	return result
}

// Simulation helper methods.

func (e *ChaosEngine) simulateAvailabilityImpact(experiment *Experiment) float64 {
	baseImpact := 0.0

	switch experiment.Type {

	case ExperimentTypePod:

		baseImpact = 2.0

	case ExperimentTypeNetwork:

		baseImpact = 5.0

	case ExperimentTypeDatabase:

		baseImpact = 10.0

	default:

		baseImpact = 1.0

	}

	// Adjust based on blast radius.

	radiusMultiplier := float64(experiment.BlastRadius.MaxPods) * 0.5

	return baseImpact * (1 + radiusMultiplier)
}

func (e *ChaosEngine) simulateLatencyImpact(experiment *Experiment) time.Duration {
	baseLatency := 100 * time.Millisecond

	switch experiment.Type {

	case ExperimentTypeNetwork:

		baseLatency = 500 * time.Millisecond

	case ExperimentTypeDatabase:

		baseLatency = 200 * time.Millisecond

	case ExperimentTypeExternal:

		baseLatency = 1000 * time.Millisecond

	}

	return baseLatency
}

func (e *ChaosEngine) simulateThroughputImpact(experiment *Experiment) float64 {
	baseImpact := 5.0

	if experiment.Type == ExperimentTypeLoad {
		baseImpact = 20.0
	}

	return baseImpact
}

func (e *ChaosEngine) simulateErrorRateImpact(experiment *Experiment) float64 {
	baseRate := 0.1

	switch experiment.Type {

	case ExperimentTypeNetwork:

		baseRate = 0.5

	case ExperimentTypeDatabase:

		baseRate = 0.3

	case ExperimentTypeExternal:

		baseRate = 1.0

	}

	return baseRate
}

func (e *ChaosEngine) simulateMTTR(experiment *Experiment) time.Duration {
	baseTime := 30 * time.Second

	if experiment.Type == ExperimentTypeDatabase {
		baseTime = 60 * time.Second
	}

	return baseTime
}

// rollbackExperiment performs experiment rollback.

func (e *ChaosEngine) rollbackExperiment(ctx context.Context, experiment *Experiment) error {
	e.logger.Info("Rolling back experiment",

		zap.String("id", experiment.ID))

	e.metrics.rollbacksTotal.WithLabelValues(

		string(experiment.Type),

		"sla_violation",
	).Inc()

	// Stop all active injections for this experiment.

	if err := e.injector.StopAllFailures(context.Background(), experiment.ID); err != nil {
		return fmt.Errorf("failed to stop failures: %w", err)
	}

	// Trigger recovery procedures.

	if err := e.recoveryTester.TriggerRecovery(context.Background(), experiment); err != nil {
		return fmt.Errorf("failed to trigger recovery: %w", err)
	}

	experiment.Status = StatusRolledBack

	return nil
}

// generateRecommendations generates recommendations based on experiment results.

func (e *ChaosEngine) generateRecommendations(result *ExperimentResult, experiment *Experiment) {
	recommendations := []string{}

	// Availability recommendations.

	if result.SLAImpact.AvailabilityImpact > 0.1 {
		recommendations = append(recommendations,

			"Consider implementing additional redundancy to improve availability during failures")
	}

	// Latency recommendations.

	if result.SLAImpact.LatencyP95Impact > 2*time.Second {
		recommendations = append(recommendations,

			"Implement circuit breakers to prevent latency cascades")
	}

	// Recovery recommendations.

	if result.RecoveryMetrics.MTTR > 5*time.Minute {
		recommendations = append(recommendations,

			"Improve auto-recovery mechanisms to reduce MTTR")
	}

	if !result.RecoveryMetrics.AutoRecoveryTriggered {
		recommendations = append(recommendations,

			"Implement automatic recovery triggers for this failure type")
	}

	if result.RecoveryMetrics.ManualIntervention {
		recommendations = append(recommendations,

			"Automate manual recovery steps to improve recovery time")
	}

	result.Recommendations = recommendations
}

// startMetricsCollection starts metrics collection during experiment.

func (e *ChaosEngine) startMetricsCollection(ctx context.Context, experiment *Experiment) *MetricsCollector {
	collector := &MetricsCollector{
		experiment: experiment,

		validator: e.validator,

		startTime: time.Now(),

		samples: []MetricsSample{},
	}

	go collector.Collect(ctx, e.config.MetricsInterval)

	return collector
}

// TriggerKillSwitch triggers the emergency kill switch.

func (e *ChaosEngine) TriggerKillSwitch(reason string) error {
	e.killSwitch.Trigger(reason)

	// Create a context with timeout for emergency stop
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Stop all active experiments.

	var errors []error

	e.activeExperiments.Range(func(key, value interface{}) bool {
		if exp, ok := value.(*Experiment); ok {
			if err := e.rollbackExperiment(ctx, exp); err != nil {
				errors = append(errors, err)
			}
		}

		return true
	})

	if len(errors) > 0 {
		return fmt.Errorf("failed to stop some experiments: %v", errors)
	}

	return nil
}

// GetExperimentStatus returns the status of an experiment.

func (e *ChaosEngine) GetExperimentStatus(id string) (*Experiment, error) {
	e.mu.RLock()

	defer e.mu.RUnlock()

	if exp, exists := e.experiments[id]; exists {
		return exp, nil
	}

	return nil, fmt.Errorf("experiment %s not found", id)
}

// ListExperiments lists all experiments.

func (e *ChaosEngine) ListExperiments() []*Experiment {
	e.mu.RLock()

	defer e.mu.RUnlock()

	experiments := make([]*Experiment, 0, len(e.experiments))

	for _, exp := range e.experiments {
		experiments = append(experiments, exp)
	}

	return experiments
}

// Stop gracefully stops the chaos engine.

func (e *ChaosEngine) Stop() error {
	e.logger.Info("Stopping chaos engine")

	// Cancel context to stop all goroutines.

	e.cancel()

	// Create a context with timeout for cleanup operations
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop all active experiments.

	var errors []error

	e.activeExperiments.Range(func(key, value interface{}) bool {
		if exp, ok := value.(*Experiment); ok {
			if err := e.rollbackExperiment(ctx, exp); err != nil {
				errors = append(errors, err)
			}
		}

		return true
	})

	if len(errors) > 0 {
		return fmt.Errorf("failed to stop some experiments: %v", errors)
	}

	return nil
}

// KillSwitch methods.

func (k *KillSwitch) Trigger(reason string) {
	k.mu.Lock()

	defer k.mu.Unlock()

	k.triggered = true

	k.reason = reason
}

// IsTriggered performs istriggered operation.

func (k *KillSwitch) IsTriggered() bool {
	k.mu.RLock()

	defer k.mu.RUnlock()

	return k.triggered
}

// GetReason performs getreason operation.

func (k *KillSwitch) GetReason() string {
	k.mu.RLock()

	defer k.mu.RUnlock()

	return k.reason
}

// Reset performs reset operation.

func (k *KillSwitch) Reset() {
	k.mu.Lock()

	defer k.mu.Unlock()

	k.triggered = false

	k.reason = ""
}

// MetricsCollector collects metrics during experiments.

type MetricsCollector struct {
	experiment *Experiment

	validator *ResilienceValidator

	startTime time.Time

	samples []MetricsSample

	mu sync.RWMutex
}

// MetricsSample represents a single metrics sample.

type MetricsSample struct {
	Timestamp time.Time

	Availability float64

	LatencyP50 time.Duration

	LatencyP95 time.Duration

	LatencyP99 time.Duration

	Throughput float64

	ErrorRate float64
}

// Collect starts collecting metrics.

func (m *MetricsCollector) Collect(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)

	defer ticker.Stop()

	for {
		select {

		case <-ctx.Done():

			return

		case <-ticker.C:

			sample := m.validator.CollectMetrics(ctx)

			m.mu.Lock()

			m.samples = append(m.samples, sample)

			m.mu.Unlock()

		}
	}
}

// GetImpactMetrics calculates impact metrics from samples.

func (m *MetricsCollector) GetImpactMetrics() SLAImpactMetrics {
	m.mu.RLock()

	defer m.mu.RUnlock()

	if len(m.samples) == 0 {
		return SLAImpactMetrics{}
	}

	// Calculate impact by comparing baseline with experiment samples.

	// This is a simplified calculation - real implementation would compare with baseline.

	var totalAvailability, totalThroughput, totalErrorRate float64

	var maxLatencyP50, maxLatencyP95, maxLatencyP99 time.Duration

	for _, sample := range m.samples {

		totalAvailability += sample.Availability

		totalThroughput += sample.Throughput

		totalErrorRate += sample.ErrorRate

		if sample.LatencyP50 > maxLatencyP50 {
			maxLatencyP50 = sample.LatencyP50
		}

		if sample.LatencyP95 > maxLatencyP95 {
			maxLatencyP95 = sample.LatencyP95
		}

		if sample.LatencyP99 > maxLatencyP99 {
			maxLatencyP99 = sample.LatencyP99
		}

	}

	count := float64(len(m.samples))

	return SLAImpactMetrics{
		AvailabilityImpact: 100 - (totalAvailability / count),

		LatencyP50Impact: maxLatencyP50,

		LatencyP95Impact: maxLatencyP95,

		LatencyP99Impact: maxLatencyP99,

		ThroughputImpact: 45 - (totalThroughput / count), // Assuming 45 intents/min baseline

		ErrorRateImpact: totalErrorRate / count,
	}
}

// Helper function to generate experiment ID.

func generateExperimentID() string {
	return fmt.Sprintf("exp-%d", time.Now().UnixNano())
}
