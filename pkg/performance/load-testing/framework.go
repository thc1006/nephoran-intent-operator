// Package loadtesting provides distributed load testing infrastructure for Nephoran Intent Operator.


package loadtesting



import (

	"context"

	"fmt"

	"sync"

	"sync/atomic"

	"time"



	"github.com/prometheus/client_golang/prometheus"

	"go.uber.org/zap"

	"golang.org/x/sync/errgroup"

)



// LoadTestFramework orchestrates distributed load testing.

type LoadTestFramework struct {

	logger           *zap.Logger

	generators       []LoadGenerator

	monitors         []PerformanceMonitor

	validators       []PerformanceValidator

	config           *LoadTestConfig

	metrics          *LoadTestMetrics

	resultCollector  *ResultCollector

	coordinationChan chan CoordinationMessage

	stopChan         chan struct{}

	wg               sync.WaitGroup

	mu               sync.RWMutex

	state            LoadTestState

}



// LoadTestConfig defines test configuration.

type LoadTestConfig struct {

	// Test identification.

	TestID      string            `json:"testId"`

	Description string            `json:"description"`

	Tags        map[string]string `json:"tags"`



	// Load pattern configuration.

	LoadPattern      LoadPattern   `json:"loadPattern"`

	Duration         time.Duration `json:"duration"`

	WarmupDuration   time.Duration `json:"warmupDuration"`

	CooldownDuration time.Duration `json:"cooldownDuration"`



	// Workload configuration.

	WorkloadMix      WorkloadMix       `json:"workloadMix"`

	IntentComplexity ComplexityProfile `json:"intentComplexity"`



	// Distribution configuration.

	Regions        []string       `json:"regions"`

	Concurrency    int            `json:"concurrency"`

	RampUpStrategy RampUpStrategy `json:"rampUpStrategy"`



	// Performance targets.

	TargetThroughput   float64       `json:"targetThroughput"` // intents per minute

	TargetLatencyP95   time.Duration `json:"targetLatencyP95"`

	TargetAvailability float64       `json:"targetAvailability"`



	// Resource limits.

	MaxCPU         float64 `json:"maxCpu"`

	MaxMemoryGB    float64 `json:"maxMemoryGb"`

	MaxNetworkMbps float64 `json:"maxNetworkMbps"`



	// Validation configuration.

	ValidationMode   ValidationMode    `json:"validationMode"`

	StatisticalTests []StatisticalTest `json:"statisticalTests"`



	// Monitoring configuration.

	MetricsInterval time.Duration `json:"metricsInterval"`

	SamplingRate    float64       `json:"samplingRate"`

	DetailedTracing bool          `json:"detailedTracing"`

}



// LoadPattern defines the load generation pattern.

type LoadPattern string



const (

	// LoadPatternConstant holds loadpatternconstant value.

	LoadPatternConstant LoadPattern = "constant"

	// LoadPatternRamp holds loadpatternramp value.

	LoadPatternRamp LoadPattern = "ramp"

	// LoadPatternStep holds loadpatternstep value.

	LoadPatternStep LoadPattern = "step"

	// LoadPatternBurst holds loadpatternburst value.

	LoadPatternBurst LoadPattern = "burst"

	// LoadPatternWave holds loadpatternwave value.

	LoadPatternWave LoadPattern = "wave"

	// LoadPatternRealistic holds loadpatternrealistic value.

	LoadPatternRealistic LoadPattern = "realistic"

	// LoadPatternChaos holds loadpatternchaos value.

	LoadPatternChaos LoadPattern = "chaos"

)



// WorkloadMix defines the distribution of different workload types.

type WorkloadMix struct {

	CoreNetworkDeployments   float64 `json:"coreNetworkDeployments"`   // 5G Core

	ORANConfigurations       float64 `json:"oranConfigurations"`       // O-RAN RIC

	NetworkSlicing           float64 `json:"networkSlicing"`           // Slice management

	MultiVendorOrchestration float64 `json:"multiVendorOrchestration"` // Multi-vendor

	PolicyManagement         float64 `json:"policyManagement"`         // A1 policies

	PerformanceOptimization  float64 `json:"performanceOptimization"`  // KPI optimization

	FaultManagement          float64 `json:"faultManagement"`          // Fault recovery

	ScalingOperations        float64 `json:"scalingOperations"`        // Auto-scaling

}



// ComplexityProfile defines the distribution of intent complexity.

type ComplexityProfile struct {

	Simple   float64 `json:"simple"`   // Basic single-action intents

	Moderate float64 `json:"moderate"` // Multi-step with dependencies

	Complex  float64 `json:"complex"`  // Cross-domain orchestration

}



// RampUpStrategy defines how load is increased.

type RampUpStrategy struct {

	Type     string        `json:"type"` // linear, exponential, stepped

	Duration time.Duration `json:"duration"`

	Steps    int           `json:"steps"`

}



// ValidationMode defines how results are validated.

type ValidationMode string



const (

	// ValidationModeStrict holds validationmodestrict value.

	ValidationModeStrict ValidationMode = "strict" // All metrics must pass

	// ValidationModeStatistical holds validationmodestatistical value.

	ValidationModeStatistical ValidationMode = "statistical" // Statistical significance

	// ValidationModeTrending holds validationmodetrending value.

	ValidationModeTrending ValidationMode = "trending" // Trend analysis

	// ValidationModeComparative holds validationmodecomparative value.

	ValidationModeComparative ValidationMode = "comparative" // Compare with baseline

)



// StatisticalTest defines statistical validation tests.

type StatisticalTest struct {

	Name          string  `json:"name"`

	Type          string  `json:"type"` // t-test, mann-whitney, ks-test

	Significance  float64 `json:"significance"`

	MinSampleSize int     `json:"minSampleSize"`

}



// LoadTestState represents the current state of load testing.

type LoadTestState string



const (

	// LoadTestStateIdle holds loadteststateidle value.

	LoadTestStateIdle LoadTestState = "idle"

	// LoadTestStateInitializing holds loadteststateinitializing value.

	LoadTestStateInitializing LoadTestState = "initializing"

	// LoadTestStateWarmingUp holds loadteststatewarmingup value.

	LoadTestStateWarmingUp LoadTestState = "warming_up"

	// LoadTestStateRunning holds loadteststaterunning value.

	LoadTestStateRunning LoadTestState = "running"

	// LoadTestStateCoolingDown holds loadteststatecoolingdown value.

	LoadTestStateCoolingDown LoadTestState = "cooling_down"

	// LoadTestStateAnalyzing holds loadteststateanalyzing value.

	LoadTestStateAnalyzing LoadTestState = "analyzing"

	// LoadTestStateCompleted holds loadteststatecompleted value.

	LoadTestStateCompleted LoadTestState = "completed"

	// LoadTestStateFailed holds loadteststatefailed value.

	LoadTestStateFailed LoadTestState = "failed"

)



// CoordinationMessage for distributed coordination.

type CoordinationMessage struct {

	Type      string                 `json:"type"`

	Timestamp time.Time              `json:"timestamp"`

	Source    string                 `json:"source"`

	Data      map[string]interface{} `json:"data"`

}



// LoadTestMetrics tracks test metrics.

type LoadTestMetrics struct {

	// Counters.

	totalRequests      *atomic.Int64

	successfulRequests *atomic.Int64

	failedRequests     *atomic.Int64



	// Latency tracking.

	latencyHistogram *prometheus.HistogramVec



	// Throughput tracking.

	throughputGauge *prometheus.GaugeVec



	// Resource tracking.

	cpuUsageGauge     *prometheus.GaugeVec

	memoryUsageGauge  *prometheus.GaugeVec

	networkUsageGauge *prometheus.GaugeVec



	// Custom metrics.

	customMetrics map[string]prometheus.Collector

	mu            sync.RWMutex

}



// LoadGenerator interface for different load generation strategies.

type LoadGenerator interface {

	// Initialize prepares the generator.

	Initialize(config *LoadTestConfig) error



	// Generate starts generating load.

	Generate(ctx context.Context) error



	// GetMetrics returns current metrics.

	GetMetrics() GeneratorMetrics



	// Adjust modifies generation parameters.

	Adjust(params map[string]interface{}) error



	// Stop gracefully stops generation.

	Stop() error

}



// GeneratorMetrics contains generator-specific metrics.

type GeneratorMetrics struct {

	RequestsSent      int64         `json:"requestsSent"`

	RequestsSucceeded int64         `json:"requestsSucceeded"`

	RequestsFailed    int64         `json:"requestsFailed"`

	AverageLatency    time.Duration `json:"averageLatency"`

	CurrentRate       float64       `json:"currentRate"`

	ErrorRate         float64       `json:"errorRate"`

}



// PerformanceMonitor interface for monitoring during tests.

type PerformanceMonitor interface {

	// Start begins monitoring.

	Start(ctx context.Context) error



	// Collect gathers current metrics.

	Collect() (MonitoringData, error)



	// Analyze performs real-time analysis.

	Analyze(data MonitoringData) AnalysisResult



	// Alert triggers alerts for anomalies.

	Alert(result AnalysisResult) error



	// Stop stops monitoring.

	Stop() error

}



// MonitoringData contains collected monitoring data.

type MonitoringData struct {

	Timestamp          time.Time              `json:"timestamp"`

	ApplicationMetrics map[string]float64     `json:"applicationMetrics"`

	SystemMetrics      map[string]float64     `json:"systemMetrics"`

	NetworkMetrics     map[string]float64     `json:"networkMetrics"`

	CustomMetrics      map[string]interface{} `json:"customMetrics"`

}



// AnalysisResult contains analysis findings.

type AnalysisResult struct {

	Timestamp       time.Time             `json:"timestamp"`

	Anomalies       []Anomaly             `json:"anomalies"`

	Trends          map[string]Trend      `json:"trends"`

	Predictions     map[string]Prediction `json:"predictions"`

	Recommendations []string              `json:"recommendations"`

}



// Anomaly represents detected anomalies.

type Anomaly struct {

	Type        string    `json:"type"`

	Severity    string    `json:"severity"`

	Description string    `json:"description"`

	Metric      string    `json:"metric"`

	Value       float64   `json:"value"`

	Expected    float64   `json:"expected"`

	Deviation   float64   `json:"deviation"`

	Timestamp   time.Time `json:"timestamp"`

}



// Trend represents metric trends.

type Trend struct {

	Direction  string  `json:"direction"` // increasing, decreasing, stable

	Rate       float64 `json:"rate"`      // rate of change

	Confidence float64 `json:"confidence"`

	Prediction float64 `json:"prediction"` // predicted value

}



// Prediction contains predictive analysis.

type Prediction struct {

	Metric      string        `json:"metric"`

	Value       float64       `json:"value"`

	Confidence  float64       `json:"confidence"`

	TimeHorizon time.Duration `json:"timeHorizon"`

}



// PerformanceValidator validates performance claims.

type PerformanceValidator interface {

	// Validate checks if performance meets requirements.

	Validate(results *LoadTestResults) ValidationReport



	// ValidateRealtime performs real-time validation.

	ValidateRealtime(metrics GeneratorMetrics) bool



	// GetThresholds returns validation thresholds.

	GetThresholds() ValidationThresholds

}



// ValidationThresholds defines validation criteria.

type ValidationThresholds struct {

	MinThroughput    float64        `json:"minThroughput"`

	MaxLatencyP50    time.Duration  `json:"maxLatencyP50"`

	MaxLatencyP95    time.Duration  `json:"maxLatencyP95"`

	MaxLatencyP99    time.Duration  `json:"maxLatencyP99"`

	MinAvailability  float64        `json:"minAvailability"`

	MaxErrorRate     float64        `json:"maxErrorRate"`

	MaxResourceUsage ResourceLimits `json:"maxResourceUsage"`

}



// ResourceLimits defines resource usage limits.

type ResourceLimits struct {

	CPUPercent  float64 `json:"cpuPercent"`

	MemoryGB    float64 `json:"memoryGb"`

	NetworkMbps float64 `json:"networkMbps"`

	DiskIOPS    float64 `json:"diskIops"`

}



// ValidationReport contains validation results.

type ValidationReport struct {

	Passed          bool                   `json:"passed"`

	Score           float64                `json:"score"`

	FailedCriteria  []string               `json:"failedCriteria"`

	Warnings        []string               `json:"warnings"`

	DetailedResults map[string]interface{} `json:"detailedResults"`

	Recommendations []string               `json:"recommendations"`

	Evidence        []Evidence             `json:"evidence"`

}



// Evidence provides proof for validation results.

type Evidence struct {

	Type        string                 `json:"type"`

	Description string                 `json:"description"`

	Data        map[string]interface{} `json:"data"`

	Timestamp   time.Time              `json:"timestamp"`

}



// ResultCollector aggregates test results.

type ResultCollector struct {

	results       *LoadTestResults

	buffer        []DataPoint

	bufferSize    int

	flushInterval time.Duration

	aggregators   map[string]Aggregator

	storage       ResultStorage

	mu            sync.Mutex

}



// LoadTestResults contains complete test results.

type LoadTestResults struct {

	TestID    string          `json:"testId"`

	StartTime time.Time       `json:"startTime"`

	EndTime   time.Time       `json:"endTime"`

	Duration  time.Duration   `json:"duration"`

	Config    *LoadTestConfig `json:"config"`



	// Performance metrics.

	TotalRequests      int64 `json:"totalRequests"`

	SuccessfulRequests int64 `json:"successfulRequests"`

	FailedRequests     int64 `json:"failedRequests"`



	// Latency metrics.

	LatencyP50    time.Duration `json:"latencyP50"`

	LatencyP95    time.Duration `json:"latencyP95"`

	LatencyP99    time.Duration `json:"latencyP99"`

	LatencyMin    time.Duration `json:"latencyMin"`

	LatencyMax    time.Duration `json:"latencyMax"`

	LatencyMean   time.Duration `json:"latencyMean"`

	LatencyStdDev time.Duration `json:"latencyStdDev"`



	// Throughput metrics.

	AverageThroughput float64 `json:"averageThroughput"`

	PeakThroughput    float64 `json:"peakThroughput"`

	MinThroughput     float64 `json:"minThroughput"`



	// Availability metrics.

	Availability float64       `json:"availability"`

	Downtime     time.Duration `json:"downtime"`

	MTBF         time.Duration `json:"mtbf"` // Mean Time Between Failures

	MTTR         time.Duration `json:"mttr"` // Mean Time To Recovery



	// Resource metrics.

	PeakCPU         float64 `json:"peakCpu"`

	AverageCPU      float64 `json:"averageCpu"`

	PeakMemoryGB    float64 `json:"peakMemoryGb"`

	AverageMemoryGB float64 `json:"averageMemoryGb"`

	PeakNetworkMbps float64 `json:"peakNetworkMbps"`



	// Breaking point analysis.

	BreakingPoint *BreakingPointAnalysis `json:"breakingPoint"`



	// Statistical analysis.

	StatisticalAnalysis *StatisticalAnalysis `json:"statisticalAnalysis"`



	// Validation results.

	ValidationReport *ValidationReport `json:"validationReport"`



	// Raw data samples.

	DataPoints []DataPoint `json:"dataPoints,omitempty"`

}



// DataPoint represents a single measurement.

type DataPoint struct {

	Timestamp     time.Time              `json:"timestamp"`

	Latency       time.Duration          `json:"latency"`

	Success       bool                   `json:"success"`

	ErrorCode     string                 `json:"errorCode,omitempty"`

	IntentType    string                 `json:"intentType"`

	Complexity    string                 `json:"complexity"`

	ResourceUsage ResourceSnapshot       `json:"resourceUsage"`

	CustomMetrics map[string]interface{} `json:"customMetrics,omitempty"`

}



// ResourceSnapshot captures resource usage at a point in time.

type ResourceSnapshot struct {

	CPUPercent      float64 `json:"cpuPercent"`

	MemoryBytes     int64   `json:"memoryBytes"`

	NetworkBytesIn  int64   `json:"networkBytesIn"`

	NetworkBytesOut int64   `json:"networkBytesOut"`

	DiskReadBytes   int64   `json:"diskReadBytes"`

	DiskWriteBytes  int64   `json:"diskWriteBytes"`

}



// BreakingPointAnalysis identifies system limits.

type BreakingPointAnalysis struct {

	MaxConcurrentIntents int           `json:"maxConcurrentIntents"`

	MaxThroughput        float64       `json:"maxThroughput"`

	LatencyDegradation   []Degradation `json:"latencyDegradation"`

	ResourceBottleneck   string        `json:"resourceBottleneck"`

	FailureMode          string        `json:"failureMode"`

	RecoveryBehavior     string        `json:"recoveryBehavior"`

}



// Degradation tracks performance degradation.

type Degradation struct {

	Load       float64       `json:"load"`

	Latency    time.Duration `json:"latency"`

	ErrorRate  float64       `json:"errorRate"`

	Throughput float64       `json:"throughput"`

}



// StatisticalAnalysis provides statistical insights.

type StatisticalAnalysis struct {

	ConfidenceIntervals map[string]ConfidenceInterval `json:"confidenceIntervals"`

	Correlations        map[string]float64            `json:"correlations"`

	Distributions       map[string]Distribution       `json:"distributions"`

	TestResults         map[string]TestResult         `json:"testResults"`

}



// ConfidenceInterval represents statistical confidence interval.

type ConfidenceInterval struct {

	Mean       float64 `json:"mean"`

	Lower      float64 `json:"lower"`

	Upper      float64 `json:"upper"`

	Confidence float64 `json:"confidence"`

}



// Distribution describes data distribution.

type Distribution struct {

	Type          string             `json:"type"`

	Parameters    map[string]float64 `json:"parameters"`

	GoodnessOfFit float64            `json:"goodnessOfFit"`

}



// TestResult contains statistical test results.

type TestResult struct {

	TestName    string  `json:"testName"`

	Statistic   float64 `json:"statistic"`

	PValue      float64 `json:"pValue"`

	Significant bool    `json:"significant"`

	Conclusion  string  `json:"conclusion"`

}



// Aggregator interface for data aggregation strategies.

type Aggregator interface {

	Add(value float64)

	Reset()

	Result() AggregationResult

}



// AggregationResult contains aggregated metrics.

type AggregationResult struct {

	Count       int64               `json:"count"`

	Sum         float64             `json:"sum"`

	Mean        float64             `json:"mean"`

	Min         float64             `json:"min"`

	Max         float64             `json:"max"`

	StdDev      float64             `json:"stdDev"`

	Percentiles map[float64]float64 `json:"percentiles"`

}



// ResultStorage interface for persisting results.

type ResultStorage interface {

	Save(results *LoadTestResults) error

	Load(testID string) (*LoadTestResults, error)

	List(filter map[string]string) ([]*LoadTestResults, error)

	Delete(testID string) error

}



// NewLoadTestFramework creates a new load testing framework.

func NewLoadTestFramework(config *LoadTestConfig, logger *zap.Logger) (*LoadTestFramework, error) {

	if config == nil {

		return nil, fmt.Errorf("config is required")

	}



	if logger == nil {

		logger = zap.NewNop()

	}



	// Validate configuration.

	if err := validateConfig(config); err != nil {

		return nil, fmt.Errorf("invalid configuration: %w", err)

	}



	framework := &LoadTestFramework{

		logger:           logger,

		config:           config,

		metrics:          newLoadTestMetrics(),

		resultCollector:  newResultCollector(config),

		coordinationChan: make(chan CoordinationMessage, 100),

		stopChan:         make(chan struct{}),

		state:            LoadTestStateIdle,

		generators:       make([]LoadGenerator, 0),

		monitors:         make([]PerformanceMonitor, 0),

		validators:       make([]PerformanceValidator, 0),

	}



	// Initialize components based on configuration.

	if err := framework.initializeComponents(); err != nil {

		return nil, fmt.Errorf("failed to initialize components: %w", err)

	}



	return framework, nil

}



// Run executes the load test.

func (f *LoadTestFramework) Run(ctx context.Context) (*LoadTestResults, error) {

	f.mu.Lock()

	if f.state != LoadTestStateIdle {

		f.mu.Unlock()

		return nil, fmt.Errorf("test already running or completed")

	}

	f.state = LoadTestStateInitializing

	f.mu.Unlock()



	f.logger.Info("Starting load test",

		zap.String("testId", f.config.TestID),

		zap.String("pattern", string(f.config.LoadPattern)),

		zap.Duration("duration", f.config.Duration))



	// Create cancellable context.

	ctx, cancel := context.WithTimeout(ctx, f.config.Duration+f.config.WarmupDuration+f.config.CooldownDuration)

	defer cancel()



	// Start result collection.

	f.resultCollector.Start(ctx)



	// Execute test phases.

	if err := f.executeWarmup(ctx); err != nil {

		f.setState(LoadTestStateFailed)

		return nil, fmt.Errorf("warmup failed: %w", err)

	}



	if err := f.executeMainTest(ctx); err != nil {

		f.setState(LoadTestStateFailed)

		return nil, fmt.Errorf("main test failed: %w", err)

	}



	if err := f.executeCooldown(ctx); err != nil {

		f.setState(LoadTestStateFailed)

		return nil, fmt.Errorf("cooldown failed: %w", err)

	}



	// Analyze results.

	f.setState(LoadTestStateAnalyzing)

	results := f.analyzeResults()



	// Validate performance claims.

	if f.config.ValidationMode != "" {

		results.ValidationReport = f.validateResults(results)

	}



	f.setState(LoadTestStateCompleted)



	f.logger.Info("Load test completed",

		zap.String("testId", f.config.TestID),

		zap.Int64("totalRequests", results.TotalRequests),

		zap.Float64("availability", results.Availability),

		zap.Duration("p95Latency", results.LatencyP95))



	return results, nil

}



// Stop gracefully stops the load test.

func (f *LoadTestFramework) Stop() error {

	f.mu.Lock()

	defer f.mu.Unlock()



	if f.state == LoadTestStateIdle || f.state == LoadTestStateCompleted {

		return nil

	}



	f.logger.Info("Stopping load test", zap.String("testId", f.config.TestID))



	// Signal stop to all components.

	close(f.stopChan)



	// Stop generators.

	for _, gen := range f.generators {

		if err := gen.Stop(); err != nil {

			f.logger.Error("Failed to stop generator", zap.Error(err))

		}

	}



	// Stop monitors.

	for _, mon := range f.monitors {

		if err := mon.Stop(); err != nil {

			f.logger.Error("Failed to stop monitor", zap.Error(err))

		}

	}



	// Wait for graceful shutdown.

	done := make(chan struct{})

	go func() {

		f.wg.Wait()

		close(done)

	}()



	select {

	case <-done:

		f.logger.Info("Load test stopped gracefully")

	case <-time.After(30 * time.Second):

		f.logger.Warn("Load test stop timeout, forcing shutdown")

	}



	f.state = LoadTestStateCompleted

	return nil

}



// GetState returns the current test state.

func (f *LoadTestFramework) GetState() LoadTestState {

	f.mu.RLock()

	defer f.mu.RUnlock()

	return f.state

}



// GetMetrics returns current test metrics.

func (f *LoadTestFramework) GetMetrics() GeneratorMetrics {

	metrics := GeneratorMetrics{

		RequestsSent:      f.metrics.totalRequests.Load(),

		RequestsSucceeded: f.metrics.successfulRequests.Load(),

		RequestsFailed:    f.metrics.failedRequests.Load(),

	}



	if metrics.RequestsSent > 0 {

		metrics.ErrorRate = float64(metrics.RequestsFailed) / float64(metrics.RequestsSent)

	}



	return metrics

}



// Private methods.



func (f *LoadTestFramework) initializeComponents() error {

	// Initialize generators based on configuration.

	for i := range f.config.Concurrency {

		gen, err := f.createGenerator(i)

		if err != nil {

			return fmt.Errorf("failed to create generator %d: %w", i, err)

		}

		f.generators = append(f.generators, gen)

	}



	// Initialize monitors.

	monitor, err := f.createMonitor()

	if err != nil {

		return fmt.Errorf("failed to create monitor: %w", err)

	}

	f.monitors = append(f.monitors, monitor)



	// Initialize validators.

	validator := f.createValidator()

	f.validators = append(f.validators, validator)



	return nil

}



func (f *LoadTestFramework) createGenerator(id int) (LoadGenerator, error) {

	// Create appropriate generator based on workload type.

	switch f.config.LoadPattern {

	case LoadPatternRealistic:

		return NewRealisticTelecomGenerator(id, f.config, f.logger)

	case LoadPatternBurst:

		return NewBurstLoadGenerator(id, f.config, f.logger)

	case LoadPatternChaos:

		return NewChaosLoadGenerator(id, f.config, f.logger)

	default:

		return NewStandardLoadGenerator(id, f.config, f.logger)

	}

}



func (f *LoadTestFramework) createMonitor() (PerformanceMonitor, error) {

	return NewComprehensiveMonitor(f.config, f.logger)

}



func (f *LoadTestFramework) createValidator() PerformanceValidator {

	return NewPerformanceValidator(f.config, f.logger)

}



func (f *LoadTestFramework) setState(state LoadTestState) {

	f.mu.Lock()

	defer f.mu.Unlock()

	f.state = state

	f.logger.Debug("State transition",

		zap.String("from", string(f.state)),

		zap.String("to", string(state)))

}



func (f *LoadTestFramework) executeWarmup(ctx context.Context) error {

	if f.config.WarmupDuration == 0 {

		return nil

	}



	f.setState(LoadTestStateWarmingUp)

	f.logger.Info("Starting warmup phase", zap.Duration("duration", f.config.WarmupDuration))



	warmupCtx, cancel := context.WithTimeout(ctx, f.config.WarmupDuration)

	defer cancel()



	// Start generators with reduced load.

	g, ctx := errgroup.WithContext(warmupCtx)



	for _, gen := range f.generators {

		generator := gen

		g.Go(func() error {

			// Reduce load during warmup.

			params := map[string]interface{}{

				"rate_multiplier": 0.5,

			}

			if err := generator.Adjust(params); err != nil {

				return err

			}

			return generator.Generate(ctx)

		})

	}



	// Start monitors.

	for _, mon := range f.monitors {

		monitor := mon

		g.Go(func() error {

			return monitor.Start(ctx)

		})

	}



	return g.Wait()

}



func (f *LoadTestFramework) executeMainTest(ctx context.Context) error {

	f.setState(LoadTestStateRunning)

	f.logger.Info("Starting main test phase", zap.Duration("duration", f.config.Duration))



	testCtx, cancel := context.WithTimeout(ctx, f.config.Duration)

	defer cancel()



	// Apply ramp-up strategy.

	if err := f.applyRampUpStrategy(testCtx); err != nil {

		return fmt.Errorf("failed to apply ramp-up strategy: %w", err)

	}



	g, ctx := errgroup.WithContext(testCtx)



	// Run generators at full load.

	for _, gen := range f.generators {

		generator := gen

		g.Go(func() error {

			// Reset to full load.

			params := map[string]interface{}{

				"rate_multiplier": 1.0,

			}

			if err := generator.Adjust(params); err != nil {

				return err

			}

			return generator.Generate(ctx)

		})

	}



	// Continuous monitoring and validation.

	g.Go(func() error {

		return f.continuousValidation(ctx)

	})



	return g.Wait()

}



func (f *LoadTestFramework) executeCooldown(ctx context.Context) error {

	if f.config.CooldownDuration == 0 {

		return nil

	}



	f.setState(LoadTestStateCoolingDown)

	f.logger.Info("Starting cooldown phase", zap.Duration("duration", f.config.CooldownDuration))



	cooldownCtx, cancel := context.WithTimeout(ctx, f.config.CooldownDuration)

	defer cancel()



	// Gradually reduce load.

	ticker := time.NewTicker(f.config.CooldownDuration / 10)

	defer ticker.Stop()



	steps := 10

	for i := steps; i > 0; i-- {

		select {

		case <-ticker.C:

			multiplier := float64(i) / float64(steps)

			for _, gen := range f.generators {

				params := map[string]interface{}{

					"rate_multiplier": multiplier,

				}

				if err := gen.Adjust(params); err != nil {

					f.logger.Warn("Failed to adjust generator", zap.Error(err))

				}

			}

		case <-cooldownCtx.Done():

			return cooldownCtx.Err()

		}

	}



	return nil

}



func (f *LoadTestFramework) applyRampUpStrategy(ctx context.Context) error {

	strategy := f.config.RampUpStrategy

	if strategy.Duration == 0 {

		return nil // No ramp-up

	}



	steps := strategy.Steps

	if steps == 0 {

		steps = 10

	}



	stepDuration := strategy.Duration / time.Duration(steps)



	for i := 1; i <= steps; i++ {

		select {

		case <-ctx.Done():

			return ctx.Err()

		default:

			multiplier := float64(i) / float64(steps)



			for _, gen := range f.generators {

				params := map[string]interface{}{

					"rate_multiplier": multiplier,

				}

				if err := gen.Adjust(params); err != nil {

					return fmt.Errorf("failed to adjust generator: %w", err)

				}

			}



			time.Sleep(stepDuration)

		}

	}



	return nil

}



func (f *LoadTestFramework) continuousValidation(ctx context.Context) error {

	ticker := time.NewTicker(f.config.MetricsInterval)

	defer ticker.Stop()



	for {

		select {

		case <-ctx.Done():

			return ctx.Err()

		case <-ticker.C:

			// Collect current metrics.

			metrics := f.GetMetrics()



			// Real-time validation.

			for _, validator := range f.validators {

				if !validator.ValidateRealtime(metrics) {

					f.logger.Warn("Real-time validation failed",

						zap.Int64("requests", metrics.RequestsSent),

						zap.Float64("errorRate", metrics.ErrorRate))

				}

			}



			// Collect monitoring data.

			for _, monitor := range f.monitors {

				data, err := monitor.Collect()

				if err != nil {

					f.logger.Error("Failed to collect monitoring data", zap.Error(err))

					continue

				}



				// Analyze for anomalies.

				result := monitor.Analyze(data)

				if len(result.Anomalies) > 0 {

					f.logger.Warn("Anomalies detected",

						zap.Int("count", len(result.Anomalies)),

						zap.Any("anomalies", result.Anomalies))



					// Alert on critical anomalies.

					for _, anomaly := range result.Anomalies {

						if anomaly.Severity == "critical" {

							if err := monitor.Alert(result); err != nil {

								f.logger.Error("Failed to send alert", zap.Error(err))

							}

							break

						}

					}

				}

			}

		}

	}

}



func (f *LoadTestFramework) analyzeResults() *LoadTestResults {

	return f.resultCollector.Finalize()

}



func (f *LoadTestFramework) validateResults(results *LoadTestResults) *ValidationReport {

	var reports []*ValidationReport



	for _, validator := range f.validators {

		report := validator.Validate(results)

		reports = append(reports, &report)

	}



	// Combine validation reports.

	combined := &ValidationReport{

		Passed:          true,

		Score:           100.0,

		FailedCriteria:  []string{},

		Warnings:        []string{},

		DetailedResults: make(map[string]interface{}),

		Recommendations: []string{},

		Evidence:        []Evidence{},

	}



	for _, report := range reports {

		if !report.Passed {

			combined.Passed = false

		}

		combined.Score = (combined.Score + report.Score) / 2

		combined.FailedCriteria = append(combined.FailedCriteria, report.FailedCriteria...)

		combined.Warnings = append(combined.Warnings, report.Warnings...)

		combined.Recommendations = append(combined.Recommendations, report.Recommendations...)

		combined.Evidence = append(combined.Evidence, report.Evidence...)

	}



	return combined

}



func validateConfig(config *LoadTestConfig) error {

	if config.TestID == "" {

		config.TestID = fmt.Sprintf("load-test-%d", time.Now().Unix())

	}



	if config.Duration == 0 {

		return fmt.Errorf("test duration must be specified")

	}



	if config.Concurrency <= 0 {

		return fmt.Errorf("concurrency must be positive")

	}



	// Validate workload mix.

	total := config.WorkloadMix.CoreNetworkDeployments +

		config.WorkloadMix.ORANConfigurations +

		config.WorkloadMix.NetworkSlicing +

		config.WorkloadMix.MultiVendorOrchestration +

		config.WorkloadMix.PolicyManagement +

		config.WorkloadMix.PerformanceOptimization +

		config.WorkloadMix.FaultManagement +

		config.WorkloadMix.ScalingOperations



	if total > 0 && total != 1.0 {

		// Normalize.

		config.WorkloadMix.CoreNetworkDeployments /= total

		config.WorkloadMix.ORANConfigurations /= total

		config.WorkloadMix.NetworkSlicing /= total

		config.WorkloadMix.MultiVendorOrchestration /= total

		config.WorkloadMix.PolicyManagement /= total

		config.WorkloadMix.PerformanceOptimization /= total

		config.WorkloadMix.FaultManagement /= total

		config.WorkloadMix.ScalingOperations /= total

	}



	// Validate complexity profile.

	complexityTotal := config.IntentComplexity.Simple +

		config.IntentComplexity.Moderate +

		config.IntentComplexity.Complex



	if complexityTotal > 0 && complexityTotal != 1.0 {

		// Normalize.

		config.IntentComplexity.Simple /= complexityTotal

		config.IntentComplexity.Moderate /= complexityTotal

		config.IntentComplexity.Complex /= complexityTotal

	}



	// Set defaults if not specified.

	if config.MetricsInterval == 0 {

		config.MetricsInterval = 10 * time.Second

	}



	if config.SamplingRate == 0 {

		config.SamplingRate = 0.1 // 10% sampling

	}



	return nil

}



func newLoadTestMetrics() *LoadTestMetrics {

	return &LoadTestMetrics{

		totalRequests:      &atomic.Int64{},

		successfulRequests: &atomic.Int64{},

		failedRequests:     &atomic.Int64{},

		latencyHistogram: prometheus.NewHistogramVec(

			prometheus.HistogramOpts{

				Name:    "loadtest_request_latency_seconds",

				Help:    "Request latency in seconds",

				Buckets: prometheus.ExponentialBuckets(0.001, 2, 15),

			},

			[]string{"intent_type", "complexity"},

		),

		throughputGauge: prometheus.NewGaugeVec(

			prometheus.GaugeOpts{

				Name: "loadtest_throughput_rpm",

				Help: "Current throughput in requests per minute",

			},

			[]string{"generator"},

		),

		cpuUsageGauge: prometheus.NewGaugeVec(

			prometheus.GaugeOpts{

				Name: "loadtest_cpu_usage_percent",

				Help: "CPU usage percentage",

			},

			[]string{"component"},

		),

		memoryUsageGauge: prometheus.NewGaugeVec(

			prometheus.GaugeOpts{

				Name: "loadtest_memory_usage_bytes",

				Help: "Memory usage in bytes",

			},

			[]string{"component"},

		),

		networkUsageGauge: prometheus.NewGaugeVec(

			prometheus.GaugeOpts{

				Name: "loadtest_network_usage_mbps",

				Help: "Network usage in Mbps",

			},

			[]string{"direction"},

		),

		customMetrics: make(map[string]prometheus.Collector),

	}

}



func newResultCollector(config *LoadTestConfig) *ResultCollector {

	return &ResultCollector{

		results: &LoadTestResults{

			TestID:    config.TestID,

			Config:    config,

			StartTime: time.Now(),

		},

		buffer:        make([]DataPoint, 0, 10000),

		bufferSize:    10000,

		flushInterval: 30 * time.Second,

		aggregators:   make(map[string]Aggregator),

	}

}



// Start begins result collection.

func (rc *ResultCollector) Start(ctx context.Context) {

	go rc.collectLoop(ctx)

}



func (rc *ResultCollector) collectLoop(ctx context.Context) {

	ticker := time.NewTicker(rc.flushInterval)

	defer ticker.Stop()



	for {

		select {

		case <-ctx.Done():

			rc.flush()

			return

		case <-ticker.C:

			rc.flush()

		}

	}

}



func (rc *ResultCollector) flush() {

	rc.mu.Lock()

	defer rc.mu.Unlock()



	if len(rc.buffer) == 0 {

		return

	}



	// Process buffered data points.

	for _, dp := range rc.buffer {

		// Update aggregators.

		if agg, ok := rc.aggregators["latency"]; ok {

			agg.Add(float64(dp.Latency))

		}



		// Update counters.

		rc.results.TotalRequests++

		if dp.Success {

			rc.results.SuccessfulRequests++

		} else {

			rc.results.FailedRequests++

		}

	}



	// Clear buffer.

	rc.buffer = rc.buffer[:0]

}



// Finalize completes result collection and returns final results.

func (rc *ResultCollector) Finalize() *LoadTestResults {

	rc.flush()



	rc.mu.Lock()

	defer rc.mu.Unlock()



	rc.results.EndTime = time.Now()

	rc.results.Duration = rc.results.EndTime.Sub(rc.results.StartTime)



	// Calculate final metrics.

	if rc.results.TotalRequests > 0 {

		rc.results.Availability = float64(rc.results.SuccessfulRequests) / float64(rc.results.TotalRequests)

		rc.results.AverageThroughput = float64(rc.results.TotalRequests) / rc.results.Duration.Minutes()

	}



	// Calculate latency percentiles from aggregators.

	if agg, ok := rc.aggregators["latency"]; ok {

		result := agg.Result()

		if p50, ok := result.Percentiles[0.5]; ok {

			rc.results.LatencyP50 = time.Duration(p50)

		}

		if p95, ok := result.Percentiles[0.95]; ok {

			rc.results.LatencyP95 = time.Duration(p95)

		}

		if p99, ok := result.Percentiles[0.99]; ok {

			rc.results.LatencyP99 = time.Duration(p99)

		}

		rc.results.LatencyMean = time.Duration(result.Mean)

		rc.results.LatencyMin = time.Duration(result.Min)

		rc.results.LatencyMax = time.Duration(result.Max)

	}



	return rc.results

}

