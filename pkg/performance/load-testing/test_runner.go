// Package loadtesting provides test execution and orchestration.


package loadtesting



import (

	"context"

	"encoding/json"

	"fmt"

	"os"

	"path/filepath"

	"time"



	"go.uber.org/zap"

	"gopkg.in/yaml.v3"

)



// TestRunner orchestrates load test execution.

type TestRunner struct {

	logger    *zap.Logger

	framework *LoadTestFramework

	scenarios []TestScenario

	results   map[string]*LoadTestResults

}



// TestScenario defines a specific test scenario.

type TestScenario struct {

	Name        string         `yaml:"name" json:"name"`

	Description string         `yaml:"description" json:"description"`

	Config      LoadTestConfig `yaml:"config" json:"config"`

	Enabled     bool           `yaml:"enabled" json:"enabled"`

	Priority    int            `yaml:"priority" json:"priority"`

}



// NewTestRunner creates a new test runner.

func NewTestRunner(logger *zap.Logger) *TestRunner {

	if logger == nil {

		logger = zap.NewNop()

	}



	return &TestRunner{

		logger:  logger,

		results: make(map[string]*LoadTestResults),

	}

}



// LoadScenarios loads test scenarios from file.

func (r *TestRunner) LoadScenarios(path string) error {

	data, err := os.ReadFile(path)

	if err != nil {

		return fmt.Errorf("failed to read scenarios file: %w", err)

	}



	ext := filepath.Ext(path)

	switch ext {

	case ".yaml", ".yml":

		if err := yaml.Unmarshal(data, &r.scenarios); err != nil {

			return fmt.Errorf("failed to parse YAML: %w", err)

		}

	case ".json":

		if err := json.Unmarshal(data, &r.scenarios); err != nil {

			return fmt.Errorf("failed to parse JSON: %w", err)

		}

	default:

		return fmt.Errorf("unsupported file format: %s", ext)

	}



	r.logger.Info("Loaded test scenarios",

		zap.Int("count", len(r.scenarios)))



	return nil

}



// RunAll executes all enabled test scenarios.

func (r *TestRunner) RunAll(ctx context.Context) error {

	r.logger.Info("Starting comprehensive load testing suite")



	startTime := time.Now()

	successCount := 0

	failureCount := 0



	for _, scenario := range r.scenarios {

		if !scenario.Enabled {

			r.logger.Info("Skipping disabled scenario",

				zap.String("name", scenario.Name))

			continue

		}



		r.logger.Info("Running test scenario",

			zap.String("name", scenario.Name),

			zap.String("description", scenario.Description))



		result, err := r.RunScenario(ctx, scenario)

		if err != nil {

			r.logger.Error("Scenario failed",

				zap.String("name", scenario.Name),

				zap.Error(err))

			failureCount++

			continue

		}



		r.results[scenario.Name] = result



		// Check if scenario passed validation.

		if result.ValidationReport != nil && result.ValidationReport.Passed {

			successCount++

			r.logger.Info("Scenario passed validation",

				zap.String("name", scenario.Name),

				zap.Float64("score", result.ValidationReport.Score))

		} else {

			failureCount++

			r.logger.Warn("Scenario failed validation",

				zap.String("name", scenario.Name))

		}

	}



	duration := time.Since(startTime)



	r.logger.Info("Load testing suite completed",

		zap.Duration("duration", duration),

		zap.Int("success", successCount),

		zap.Int("failure", failureCount))



	return nil

}



// RunScenario executes a single test scenario.

func (r *TestRunner) RunScenario(ctx context.Context, scenario TestScenario) (*LoadTestResults, error) {

	// Create framework for this scenario.

	framework, err := NewLoadTestFramework(&scenario.Config, r.logger)

	if err != nil {

		return nil, fmt.Errorf("failed to create framework: %w", err)

	}



	// Run the test.

	results, err := framework.Run(ctx)

	if err != nil {

		return nil, fmt.Errorf("test execution failed: %w", err)

	}



	return results, nil

}



// GenerateReport generates a comprehensive test report.

func (r *TestRunner) GenerateReport(outputPath string) error {

	report := ComprehensiveReport{

		GeneratedAt: time.Now(),

		Scenarios:   make([]ScenarioReport, 0),

		Summary:     ReportSummary{},

	}



	// Process each scenario result.

	for name, result := range r.results {

		scenarioReport := ScenarioReport{

			Name:      name,

			StartTime: result.StartTime,

			EndTime:   result.EndTime,

			Duration:  result.Duration,

			Metrics: PerformanceMetrics{

				TotalRequests:      result.TotalRequests,

				SuccessfulRequests: result.SuccessfulRequests,

				FailedRequests:     result.FailedRequests,

				Availability:       result.Availability,

				ThroughputAvg:      result.AverageThroughput,

				ThroughputPeak:     result.PeakThroughput,

				LatencyP50:         result.LatencyP50,

				LatencyP95:         result.LatencyP95,

				LatencyP99:         result.LatencyP99,

			},

		}



		if result.ValidationReport != nil {

			scenarioReport.ValidationPassed = result.ValidationReport.Passed

			scenarioReport.ValidationScore = result.ValidationReport.Score

			scenarioReport.FailedCriteria = result.ValidationReport.FailedCriteria

			scenarioReport.Warnings = result.ValidationReport.Warnings

		}



		report.Scenarios = append(report.Scenarios, scenarioReport)



		// Update summary.

		if scenarioReport.ValidationPassed {

			report.Summary.PassedScenarios++

		} else {

			report.Summary.FailedScenarios++

		}

	}



	report.Summary.TotalScenarios = len(report.Scenarios)



	// Check performance claims.

	report.Summary.PerformanceClaims = r.validatePerformanceClaims()



	// Marshal report.

	data, err := json.MarshalIndent(report, "", "  ")

	if err != nil {

		return fmt.Errorf("failed to marshal report: %w", err)

	}



	// Write to file.

	if err := os.WriteFile(outputPath, data, 0o640); err != nil {

		return fmt.Errorf("failed to write report: %w", err)

	}



	r.logger.Info("Test report generated",

		zap.String("path", outputPath))



	return nil

}



// validatePerformanceClaims checks if all performance claims are met.

func (r *TestRunner) validatePerformanceClaims() PerformanceClaimsValidation {

	validation := PerformanceClaimsValidation{

		ConcurrentIntents200Plus: false,

		Throughput45PerMinute:    false,

		LatencyP95Under2Seconds:  false,

		Availability9995Percent:  false,

		AllClaimsMet:             false,

	}



	// Check each claim across all scenarios.

	for _, result := range r.results {

		// Use calculations from validator.

		if result.ValidationReport != nil {

			details := result.ValidationReport.DetailedResults



			if concurrent, ok := details["concurrent_intents"].(int); ok && concurrent >= 200 {

				validation.ConcurrentIntents200Plus = true

			}



			if throughput, ok := details["throughput"].(map[string]float64); ok {

				if avg, ok := throughput["average"]; ok && avg >= 45 {

					validation.Throughput45PerMinute = true

				}

			}



			if latency, ok := details["latency"].(map[string]time.Duration); ok {

				if p95, ok := latency["p95"]; ok && p95 <= 2*time.Second {

					validation.LatencyP95Under2Seconds = true

				}

			}



			if availability, ok := details["availability"].(float64); ok && availability >= 0.9995 {

				validation.Availability9995Percent = true

			}

		}

	}



	validation.AllClaimsMet = validation.ConcurrentIntents200Plus &&

		validation.Throughput45PerMinute &&

		validation.LatencyP95Under2Seconds &&

		validation.Availability9995Percent



	return validation

}



// GetResults returns test results for a specific scenario.

func (r *TestRunner) GetResults(scenarioName string) (*LoadTestResults, bool) {

	result, ok := r.results[scenarioName]

	return result, ok

}



// GetAllResults returns all test results.

func (r *TestRunner) GetAllResults() map[string]*LoadTestResults {

	return r.results

}



// Report structures.



// ComprehensiveReport contains the complete test report.

type ComprehensiveReport struct {

	GeneratedAt time.Time        `json:"generatedAt"`

	Scenarios   []ScenarioReport `json:"scenarios"`

	Summary     ReportSummary    `json:"summary"`

}



// ScenarioReport contains results for a single scenario.

type ScenarioReport struct {

	Name             string             `json:"name"`

	StartTime        time.Time          `json:"startTime"`

	EndTime          time.Time          `json:"endTime"`

	Duration         time.Duration      `json:"duration"`

	Metrics          PerformanceMetrics `json:"metrics"`

	ValidationPassed bool               `json:"validationPassed"`

	ValidationScore  float64            `json:"validationScore"`

	FailedCriteria   []string           `json:"failedCriteria,omitempty"`

	Warnings         []string           `json:"warnings,omitempty"`

}



// PerformanceMetrics contains key performance metrics.

type PerformanceMetrics struct {

	TotalRequests      int64         `json:"totalRequests"`

	SuccessfulRequests int64         `json:"successfulRequests"`

	FailedRequests     int64         `json:"failedRequests"`

	Availability       float64       `json:"availability"`

	ThroughputAvg      float64       `json:"throughputAvg"`

	ThroughputPeak     float64       `json:"throughputPeak"`

	LatencyP50         time.Duration `json:"latencyP50"`

	LatencyP95         time.Duration `json:"latencyP95"`

	LatencyP99         time.Duration `json:"latencyP99"`

}



// ReportSummary contains summary statistics.

type ReportSummary struct {

	TotalScenarios    int                         `json:"totalScenarios"`

	PassedScenarios   int                         `json:"passedScenarios"`

	FailedScenarios   int                         `json:"failedScenarios"`

	PerformanceClaims PerformanceClaimsValidation `json:"performanceClaims"`

}



// PerformanceClaimsValidation tracks validation of performance claims.

type PerformanceClaimsValidation struct {

	ConcurrentIntents200Plus bool `json:"concurrentIntents200Plus"`

	Throughput45PerMinute    bool `json:"throughput45PerMinute"`

	LatencyP95Under2Seconds  bool `json:"latencyP95Under2Seconds"`

	Availability9995Percent  bool `json:"availability9995Percent"`

	AllClaimsMet             bool `json:"allClaimsMet"`

}



// PredefinedScenarios returns a set of predefined test scenarios.

func PredefinedScenarios() []TestScenario {

	return []TestScenario{

		{

			Name:        "baseline_performance",

			Description: "Validate baseline performance claims",

			Enabled:     true,

			Priority:    1,

			Config: LoadTestConfig{

				TestID:           "baseline-001",

				Description:      "Baseline performance validation",

				LoadPattern:      LoadPatternConstant,

				Duration:         10 * time.Minute,

				WarmupDuration:   2 * time.Minute,

				CooldownDuration: 1 * time.Minute,

				WorkloadMix: WorkloadMix{

					CoreNetworkDeployments:   0.3,

					ORANConfigurations:       0.2,

					NetworkSlicing:           0.2,

					MultiVendorOrchestration: 0.1,

					PolicyManagement:         0.1,

					PerformanceOptimization:  0.05,

					FaultManagement:          0.03,

					ScalingOperations:        0.02,

				},

				IntentComplexity: ComplexityProfile{

					Simple:   0.2,

					Moderate: 0.6,

					Complex:  0.2,

				},

				Regions:            []string{"us-east", "us-west", "eu-central"},

				Concurrency:        200,

				TargetThroughput:   45,

				TargetLatencyP95:   2 * time.Second,

				TargetAvailability: 0.9995,

				MaxCPU:             80,

				MaxMemoryGB:        16,

				MaxNetworkMbps:     1000,

				ValidationMode:     ValidationModeStrict,

				MetricsInterval:    10 * time.Second,

				SamplingRate:       0.1,

			},

		},

		{

			Name:        "stress_test",

			Description: "Find system breaking points",

			Enabled:     true,

			Priority:    2,

			Config: LoadTestConfig{

				TestID:         "stress-001",

				Description:    "Stress testing for breaking points",

				LoadPattern:    LoadPatternRamp,

				Duration:       15 * time.Minute,

				WarmupDuration: 1 * time.Minute,

				WorkloadMix: WorkloadMix{

					CoreNetworkDeployments: 0.4,

					ORANConfigurations:     0.3,

					NetworkSlicing:         0.2,

					ScalingOperations:      0.1,

				},

				IntentComplexity: ComplexityProfile{

					Simple:   0.1,

					Moderate: 0.4,

					Complex:  0.5, // More complex intents for stress

				},

				Regions:     []string{"us-east", "us-west", "eu-central", "ap-south"},

				Concurrency: 300, // Higher than target

				RampUpStrategy: RampUpStrategy{

					Type:     "exponential",

					Duration: 5 * time.Minute,

					Steps:    20,

				},

				TargetThroughput: 60, // Higher than baseline

				TargetLatencyP95: 3 * time.Second,

				MaxCPU:           90,

				MaxMemoryGB:      32,

				MaxNetworkMbps:   2000,

				ValidationMode:   ValidationModeStatistical,

				MetricsInterval:  5 * time.Second,

				SamplingRate:     0.2,

			},

		},

		{

			Name:        "burst_traffic",

			Description: "Test burst traffic handling",

			Enabled:     true,

			Priority:    3,

			Config: LoadTestConfig{

				TestID:      "burst-001",

				Description: "Burst traffic pattern testing",

				LoadPattern: LoadPatternBurst,

				Duration:    10 * time.Minute,

				WorkloadMix: WorkloadMix{

					CoreNetworkDeployments: 0.5,

					ScalingOperations:      0.3,

					PolicyManagement:       0.2,

				},

				IntentComplexity: ComplexityProfile{

					Simple:   0.4,

					Moderate: 0.5,

					Complex:  0.1,

				},

				Regions:            []string{"us-east"},

				Concurrency:        250,

				TargetThroughput:   50,

				TargetLatencyP95:   2500 * time.Millisecond,

				TargetAvailability: 0.999,

				MaxCPU:             85,

				MaxMemoryGB:        20,

				MaxNetworkMbps:     1500,

				ValidationMode:     ValidationModeStatistical,

				MetricsInterval:    5 * time.Second,

				SamplingRate:       0.15,

			},

		},

		{

			Name:        "sustained_load",

			Description: "Long-duration sustained load test",

			Enabled:     true,

			Priority:    4,

			Config: LoadTestConfig{

				TestID:           "sustained-001",

				Description:      "Sustained load for stability validation",

				LoadPattern:      LoadPatternConstant,

				Duration:         30 * time.Minute, // Longer duration

				WarmupDuration:   3 * time.Minute,

				CooldownDuration: 2 * time.Minute,

				WorkloadMix: WorkloadMix{

					CoreNetworkDeployments:   0.25,

					ORANConfigurations:       0.25,

					NetworkSlicing:           0.25,

					MultiVendorOrchestration: 0.15,

					PolicyManagement:         0.05,

					PerformanceOptimization:  0.03,

					FaultManagement:          0.02,

				},

				IntentComplexity: ComplexityProfile{

					Simple:   0.2,

					Moderate: 0.6,

					Complex:  0.2,

				},

				Regions:            []string{"us-east", "us-west"},

				Concurrency:        180,

				TargetThroughput:   45,

				TargetLatencyP95:   2 * time.Second,

				TargetAvailability: 0.9995,

				MaxCPU:             75,

				MaxMemoryGB:        16,

				MaxNetworkMbps:     800,

				ValidationMode:     ValidationModeStrict,

				MetricsInterval:    30 * time.Second,

				SamplingRate:       0.05,

			},

		},

		{

			Name:        "realistic_telecom",

			Description: "Realistic telecom traffic patterns",

			Enabled:     true,

			Priority:    5,

			Config: LoadTestConfig{

				TestID:         "realistic-001",

				Description:    "Realistic telecom workload patterns",

				LoadPattern:    LoadPatternRealistic,

				Duration:       20 * time.Minute,

				WarmupDuration: 2 * time.Minute,

				WorkloadMix: WorkloadMix{

					CoreNetworkDeployments:   0.3,

					ORANConfigurations:       0.2,

					NetworkSlicing:           0.15,

					MultiVendorOrchestration: 0.1,

					PolicyManagement:         0.1,

					PerformanceOptimization:  0.05,

					FaultManagement:          0.05,

					ScalingOperations:        0.05,

				},

				IntentComplexity: ComplexityProfile{

					Simple:   0.2,

					Moderate: 0.6,

					Complex:  0.2,

				},

				Regions:            []string{"us-east", "us-west", "eu-central", "ap-south"},

				Concurrency:        200,

				TargetThroughput:   45,

				TargetLatencyP95:   2 * time.Second,

				TargetAvailability: 0.9995,

				MaxCPU:             80,

				MaxMemoryGB:        16,

				MaxNetworkMbps:     1000,

				ValidationMode:     ValidationModeStrict,

				MetricsInterval:    10 * time.Second,

				SamplingRate:       0.1,

				DetailedTracing:    true,

			},

		},

	}

}

