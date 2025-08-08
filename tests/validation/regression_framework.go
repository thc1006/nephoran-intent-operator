// Package validation provides regression testing framework for Nephoran Intent Operator
// This framework prevents quality degradation by detecting regressions across all validation categories
package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// RegressionFramework provides comprehensive regression detection capabilities
type RegressionFramework struct {
	// Core components
	validationSuite *ValidationSuite
	baselineManager *BaselineManager
	detectionEngine *RegressionDetectionEngine
	trendAnalyzer   *TrendAnalyzer
	alertSystem     *AlertSystem
	
	// Configuration
	config     *RegressionConfig
	
	// State management
	currentResults *ValidationResults
	lastBaseline   *BaselineSnapshot
	
	mu sync.RWMutex
}

// RegressionConfig holds configuration for regression testing
type RegressionConfig struct {
	// Baseline management
	BaselineStoragePath   string
	AutoBaselineUpdate    bool
	BaselineRetention     int // Number of baselines to keep
	
	// Regression thresholds
	PerformanceThreshold  float64 // > 10% degradation
	FunctionalThreshold   float64 // > 5% decrease in pass rates
	SecurityThreshold     float64 // Any new vulnerabilities
	ProductionThreshold   float64 // Any availability violations
	OverallScoreThreshold float64 // < 5% decrease in total score
	
	// Detection settings
	MinimumSamples       int     // Minimum samples for trend analysis
	StatisticalConfidence float64 // Confidence level for regression detection
	EnableTrendAnalysis  bool
	EnableAlerting       bool
	
	// Alert configuration
	AlertWebhookURL      string
	AlertSlackChannel    string
	AlertEmailRecipients []string
	
	// CI/CD integration
	FailOnRegression     bool
	GenerateJUnitReport  bool
	ExportMetricsFormat  string // "prometheus", "json", "csv"
}

// DefaultRegressionConfig returns production-ready regression configuration
func DefaultRegressionConfig() *RegressionConfig {
	return &RegressionConfig{
		BaselineStoragePath:   "./regression-baselines",
		AutoBaselineUpdate:    false,
		BaselineRetention:     10,
		
		PerformanceThreshold:  10.0, // 10% performance degradation
		FunctionalThreshold:   5.0,  // 5% functional degradation
		SecurityThreshold:     0.0,  // No new security issues
		ProductionThreshold:   0.0,  // No production readiness degradation
		OverallScoreThreshold: 5.0,  // 5% overall score degradation
		
		MinimumSamples:       3,
		StatisticalConfidence: 95.0,
		EnableTrendAnalysis:  true,
		EnableAlerting:       true,
		
		FailOnRegression:    true,
		GenerateJUnitReport: true,
		ExportMetricsFormat: "prometheus",
	}
}

// BaselineSnapshot represents a validation baseline at a specific point in time
type BaselineSnapshot struct {
	// Metadata
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Version     string    `json:"version"`
	CommitHash  string    `json:"commit_hash"`
	Environment string    `json:"environment"`
	
	// Validation results
	Results *ValidationResults `json:"results"`
	
	// Performance baselines
	PerformanceBaselines map[string]*PerformanceBaseline `json:"performance_baselines"`
	
	// Security baselines
	SecurityBaselines *SecurityBaseline `json:"security_baselines"`
	
	// Production baselines
	ProductionBaselines *ProductionBaseline `json:"production_baselines"`
	
	// Statistical metrics
	Statistics *BaselineStatistics `json:"statistics"`
}

// PerformanceBaseline holds performance baseline metrics
type PerformanceBaseline struct {
	MetricName       string        `json:"metric_name"`
	AverageValue     float64       `json:"average_value"`
	MedianValue      float64       `json:"median_value"`
	P95Value         float64       `json:"p95_value"`
	P99Value         float64       `json:"p99_value"`
	StandardDev      float64       `json:"standard_deviation"`
	AcceptableRange  [2]float64    `json:"acceptable_range"`
	Unit             string        `json:"unit"`
	Threshold        float64       `json:"threshold"`
}

// SecurityBaseline holds security baseline metrics
type SecurityBaseline struct {
	VulnerabilityCount     int                        `json:"vulnerability_count"`
	CriticalVulnerabilities int                        `json:"critical_vulnerabilities"`
	HighVulnerabilities    int                        `json:"high_vulnerabilities"`
	SecurityScore          int                        `json:"security_score"`
	ComplianceFindings     map[string]*SecurityFinding `json:"compliance_findings"`
	EncryptionCoverage     float64                     `json:"encryption_coverage"`
	AuthenticationScore    int                         `json:"authentication_score"`
}

// ProductionBaseline holds production readiness baseline metrics
type ProductionBaseline struct {
	AvailabilityTarget     float64       `json:"availability_target"`
	MTBF                  time.Duration `json:"mtbf"`
	MTTR                  time.Duration `json:"mttr"`
	ErrorRate             float64       `json:"error_rate"`
	FaultToleranceScore   int           `json:"fault_tolerance_score"`
	MonitoringCoverage    float64       `json:"monitoring_coverage"`
	DisasterRecoveryScore int           `json:"disaster_recovery_score"`
}

// BaselineStatistics holds statistical analysis of baseline metrics
type BaselineStatistics struct {
	SampleSize       int     `json:"sample_size"`
	ConfidenceLevel  float64 `json:"confidence_level"`
	VariabilityIndex float64 `json:"variability_index"`
	TrendDirection   string  `json:"trend_direction"` // "improving", "stable", "degrading"
	SeasonalPatterns bool    `json:"seasonal_patterns"`
}

// RegressionDetection holds the results of regression detection
type RegressionDetection struct {
	// Overall regression status
	HasRegression    bool                    `json:"has_regression"`
	RegressionSeverity string                `json:"regression_severity"` // "low", "medium", "high", "critical"
	
	// Category-specific regressions
	PerformanceRegressions []*PerformanceRegression `json:"performance_regressions"`
	FunctionalRegressions  []*FunctionalRegression  `json:"functional_regressions"`
	SecurityRegressions    []*SecurityRegression    `json:"security_regressions"`
	ProductionRegressions  []*ProductionRegression  `json:"production_regressions"`
	
	// Statistical analysis
	StatisticalSignificance float64 `json:"statistical_significance"`
	ConfidenceLevel        float64 `json:"confidence_level"`
	
	// Comparison metadata
	BaselineID       string    `json:"baseline_id"`
	ComparisonTime   time.Time `json:"comparison_time"`
	ExecutionContext string    `json:"execution_context"`
}

// PerformanceRegression represents a detected performance regression
type PerformanceRegression struct {
	MetricName        string    `json:"metric_name"`
	BaselineValue     float64   `json:"baseline_value"`
	CurrentValue      float64   `json:"current_value"`
	DegradationPercent float64   `json:"degradation_percent"`
	Severity          string    `json:"severity"`
	Impact            string    `json:"impact"`
	Recommendation    string    `json:"recommendation"`
	DetectionTime     time.Time `json:"detection_time"`
}

// FunctionalRegression represents a detected functional regression
type FunctionalRegression struct {
	TestCategory      string    `json:"test_category"`
	BaselinePassRate  float64   `json:"baseline_pass_rate"`
	CurrentPassRate   float64   `json:"current_pass_rate"`
	FailedTests       []string  `json:"failed_tests"`
	NewFailures       []string  `json:"new_failures"`
	Severity          string    `json:"severity"`
	Impact            string    `json:"impact"`
	Recommendation    string    `json:"recommendation"`
	DetectionTime     time.Time `json:"detection_time"`
}

// SecurityRegression represents a detected security regression
type SecurityRegression struct {
	Finding           *SecurityFinding `json:"finding"`
	IsNewVulnerability bool            `json:"is_new_vulnerability"`
	SeverityChange    string           `json:"severity_change"`
	Impact            string           `json:"impact"`
	Recommendation    string           `json:"recommendation"`
	DetectionTime     time.Time        `json:"detection_time"`
}

// ProductionRegression represents a detected production readiness regression
type ProductionRegression struct {
	Metric            string    `json:"metric"`
	BaselineValue     float64   `json:"baseline_value"`
	CurrentValue      float64   `json:"current_value"`
	Impact            string    `json:"impact"`
	Recommendation    string    `json:"recommendation"`
	DetectionTime     time.Time `json:"detection_time"`
}

// NewRegressionFramework creates a new regression testing framework
func NewRegressionFramework(config *RegressionConfig, validationConfig *ValidationConfig) *RegressionFramework {
	if config == nil {
		config = DefaultRegressionConfig()
	}
	if validationConfig == nil {
		validationConfig = DefaultValidationConfig()
	}
	
	rf := &RegressionFramework{
		config:          config,
		validationSuite: NewValidationSuite(validationConfig),
		baselineManager: NewBaselineManager(config),
		detectionEngine: NewRegressionDetectionEngine(config),
		trendAnalyzer:   NewTrendAnalyzer(config),
		alertSystem:     NewAlertSystem(config),
	}
	
	// Initialize storage directory
	if err := os.MkdirAll(config.BaselineStoragePath, 0755); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create baseline storage directory: %v", err))
	}
	
	return rf
}

// ExecuteRegressionTest runs comprehensive regression testing
func (rf *RegressionFramework) ExecuteRegressionTest(ctx context.Context) (*RegressionDetection, error) {
	ginkgo.By("Starting Comprehensive Regression Testing")
	
	startTime := time.Now()
	
	// Step 1: Load latest baseline
	ginkgo.By("Loading latest baseline for comparison")
	baseline, err := rf.baselineManager.LoadLatestBaseline()
	if err != nil {
		if os.IsNotExist(err) {
			ginkgo.By("No baseline found - establishing new baseline")
			return rf.EstablishNewBaseline(ctx)
		}
		return nil, fmt.Errorf("failed to load baseline: %w", err)
	}
	rf.lastBaseline = baseline
	
	// Step 2: Execute current validation
	ginkgo.By("Executing current validation suite")
	currentResults, err := rf.validationSuite.ExecuteComprehensiveValidation(ctx)
	if err != nil {
		return nil, fmt.Errorf("validation execution failed: %w", err)
	}
	rf.currentResults = currentResults
	
	// Step 3: Detect regressions
	ginkgo.By("Detecting regressions against baseline")
	regressionDetection, err := rf.detectionEngine.DetectRegressions(baseline, currentResults)
	if err != nil {
		return nil, fmt.Errorf("regression detection failed: %w", err)
	}
	
	// Step 4: Perform trend analysis
	if rf.config.EnableTrendAnalysis {
		ginkgo.By("Performing trend analysis")
		trendAnalysis := rf.trendAnalyzer.AnalyzeTrends(baseline, currentResults)
		rf.enrichRegressionWithTrends(regressionDetection, trendAnalysis)
	}
	
	// Step 5: Generate alerts if regressions detected
	if rf.config.EnableAlerting && regressionDetection.HasRegression {
		ginkgo.By("Generating regression alerts")
		rf.alertSystem.SendRegressionAlert(regressionDetection)
	}
	
	// Step 6: Update baseline if configured
	if rf.config.AutoBaselineUpdate && !regressionDetection.HasRegression {
		ginkgo.By("Updating baseline with current results")
		rf.baselineManager.CreateBaseline(currentResults, "auto-update")
	}
	
	// Step 7: Generate comprehensive report
	rf.generateRegressionReport(regressionDetection, time.Since(startTime))
	
	// Step 8: Handle CI/CD integration
	if rf.config.FailOnRegression && regressionDetection.HasRegression {
		return regressionDetection, fmt.Errorf("regression detected: %s", regressionDetection.RegressionSeverity)
	}
	
	ginkgo.By(fmt.Sprintf("Regression testing completed: %s", 
		func() string {
			if regressionDetection.HasRegression {
				return fmt.Sprintf("REGRESSION DETECTED (%s)", regressionDetection.RegressionSeverity)
			}
			return "NO REGRESSION"
		}()))
	
	return regressionDetection, nil
}

// EstablishNewBaseline creates a new baseline when none exists
func (rf *RegressionFramework) EstablishNewBaseline(ctx context.Context) (*RegressionDetection, error) {
	ginkgo.By("Establishing new baseline")
	
	// Execute validation to establish baseline
	results, err := rf.validationSuite.ExecuteComprehensiveValidation(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute validation for baseline: %w", err)
	}
	
	// Create baseline
	baseline, err := rf.baselineManager.CreateBaseline(results, "initial-baseline")
	if err != nil {
		return nil, fmt.Errorf("failed to create baseline: %w", err)
	}
	
	ginkgo.By(fmt.Sprintf("Baseline established with ID: %s", baseline.ID))
	
	// Return no regression detection for new baseline
	return &RegressionDetection{
		HasRegression:           false,
		RegressionSeverity:     "none",
		BaselineID:            baseline.ID,
		ComparisonTime:        time.Now(),
		ExecutionContext:      "baseline-establishment",
		PerformanceRegressions: []*PerformanceRegression{},
		FunctionalRegressions:  []*FunctionalRegression{},
		SecurityRegressions:    []*SecurityRegression{},
		ProductionRegressions:  []*ProductionRegression{},
	}, nil
}

// GetRegressionHistory returns historical regression data
func (rf *RegressionFramework) GetRegressionHistory(days int) ([]*RegressionDetection, error) {
	return rf.baselineManager.GetRegressionHistory(days)
}

// GenerateRegressionTrends creates trend analysis over time
func (rf *RegressionFramework) GenerateRegressionTrends() (*TrendAnalysis, error) {
	baselines, err := rf.baselineManager.LoadAllBaselines()
	if err != nil {
		return nil, err
	}
	
	return rf.trendAnalyzer.GenerateTrends(baselines), nil
}

// enrichRegressionWithTrends adds trend information to regression detection
func (rf *RegressionFramework) enrichRegressionWithTrends(detection *RegressionDetection, trends *TrendAnalysis) {
	// Add trend context to regression detection
	// This would enhance the regression report with trend information
	for _, perfRegression := range detection.PerformanceRegressions {
		if trend, exists := trends.PerformanceTrends[perfRegression.MetricName]; exists {
			perfRegression.Impact = fmt.Sprintf("%s (Trend: %s)", perfRegression.Impact, trend.Direction)
		}
	}
}

// generateRegressionReport creates comprehensive regression report
func (rf *RegressionFramework) generateRegressionReport(detection *RegressionDetection, executionTime time.Duration) {
	ginkgo.By("Generating comprehensive regression report")
	
	status := "PASSED"
	if detection.HasRegression {
		status = fmt.Sprintf("FAILED (%s)", detection.RegressionSeverity)
	}
	
	report := fmt.Sprintf(`
=============================================================================
NEPHORAN INTENT OPERATOR - REGRESSION TEST REPORT
=============================================================================

REGRESSION STATUS: %s
EXECUTION TIME: %v
BASELINE ID: %s
COMPARISON TIME: %s

REGRESSION SUMMARY:
├── Performance Regressions: %d detected
├── Functional Regressions:  %d detected  
├── Security Regressions:    %d detected
└── Production Regressions:  %d detected

STATISTICAL CONFIDENCE: %.1f%%
DETECTION THRESHOLDS:
├── Performance Degradation: >%.1f%%
├── Functional Degradation:  >%.1f%%
├── Security Issues:         >%.1f%%
└── Production Issues:       >%.1f%%

=============================================================================
`,
		status,
		executionTime,
		detection.BaselineID,
		detection.ComparisonTime.Format(time.RFC3339),
		len(detection.PerformanceRegressions),
		len(detection.FunctionalRegressions),
		len(detection.SecurityRegressions),
		len(detection.ProductionRegressions),
		detection.ConfidenceLevel,
		rf.config.PerformanceThreshold,
		rf.config.FunctionalThreshold,
		rf.config.SecurityThreshold,
		rf.config.ProductionThreshold,
	)
	
	// Add detailed regression information
	if len(detection.PerformanceRegressions) > 0 {
		report += "\nPERFORMANCE REGRESSIONS:\n"
		for i, reg := range detection.PerformanceRegressions {
			report += fmt.Sprintf("  %d. %s: %.2f%% degradation (%.2f → %.2f)\n", 
				i+1, reg.MetricName, reg.DegradationPercent, reg.BaselineValue, reg.CurrentValue)
			report += fmt.Sprintf("     Impact: %s\n", reg.Impact)
			report += fmt.Sprintf("     Recommendation: %s\n", reg.Recommendation)
		}
	}
	
	if len(detection.FunctionalRegressions) > 0 {
		report += "\nFUNCTIONAL REGRESSIONS:\n"
		for i, reg := range detection.FunctionalRegressions {
			report += fmt.Sprintf("  %d. %s: Pass rate dropped from %.1f%% to %.1f%%\n",
				i+1, reg.TestCategory, reg.BaselinePassRate, reg.CurrentPassRate)
			if len(reg.NewFailures) > 0 {
				report += fmt.Sprintf("     New failures: %v\n", reg.NewFailures)
			}
		}
	}
	
	if len(detection.SecurityRegressions) > 0 {
		report += "\nSECURITY REGRESSIONS:\n"
		for i, reg := range detection.SecurityRegressions {
			report += fmt.Sprintf("  %d. %s: %s (%s)\n",
				i+1, reg.Finding.Type, reg.Finding.Description, reg.Finding.Severity)
		}
	}
	
	if len(detection.ProductionRegressions) > 0 {
		report += "\nPRODUCTION REGRESSIONS:\n"
		for i, reg := range detection.ProductionRegressions {
			report += fmt.Sprintf("  %d. %s: %.2f → %.2f\n",
				i+1, reg.Metric, reg.BaselineValue, reg.CurrentValue)
		}
	}
	
	report += "\n=============================================================================\n"
	
	fmt.Print(report)
	
	// Write detailed JSON report
	rf.writeDetailedRegressionReport(detection)
	
	// Generate JUnit report if configured
	if rf.config.GenerateJUnitReport {
		rf.generateJUnitReport(detection)
	}
	
	// Export metrics in configured format
	rf.exportMetrics(detection)
}

// writeDetailedRegressionReport writes detailed JSON report
func (rf *RegressionFramework) writeDetailedRegressionReport(detection *RegressionDetection) {
	reportPath := filepath.Join(rf.config.BaselineStoragePath, fmt.Sprintf("regression-report-%s.json", 
		detection.ComparisonTime.Format("2006-01-02T15-04-05")))
	
	data, err := json.MarshalIndent(detection, "", "  ")
	if err != nil {
		ginkgo.By(fmt.Sprintf("Warning: Failed to marshal regression report: %v", err))
		return
	}
	
	if err := os.WriteFile(reportPath, data, 0644); err != nil {
		ginkgo.By(fmt.Sprintf("Warning: Failed to write regression report: %v", err))
	}
}

// generateJUnitReport generates JUnit XML report for CI/CD integration
func (rf *RegressionFramework) generateJUnitReport(detection *RegressionDetection) {
	// Implementation would generate JUnit XML format report
	// This enables integration with CI/CD systems like Jenkins, GitHub Actions, etc.
}

// exportMetrics exports metrics in configured format
func (rf *RegressionFramework) exportMetrics(detection *RegressionDetection) {
	switch rf.config.ExportMetricsFormat {
	case "prometheus":
		rf.exportPrometheusMetrics(detection)
	case "json":
		rf.exportJSONMetrics(detection)
	case "csv":
		rf.exportCSVMetrics(detection)
	}
}

// exportPrometheusMetrics exports metrics in Prometheus format
func (rf *RegressionFramework) exportPrometheusMetrics(detection *RegressionDetection) {
	// Implementation would export metrics in Prometheus format
}

// exportJSONMetrics exports metrics in JSON format
func (rf *RegressionFramework) exportJSONMetrics(detection *RegressionDetection) {
	// Implementation would export metrics in JSON format
}

// exportCSVMetrics exports metrics in CSV format
func (rf *RegressionFramework) exportCSVMetrics(detection *RegressionDetection) {
	// Implementation would export metrics in CSV format
}