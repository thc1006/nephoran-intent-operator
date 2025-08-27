// Package security provides comprehensive security test execution
// for the Nephoran Intent Operator with orchestrated test execution.
package security

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/tests/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ComprehensiveSecurityTestSuite orchestrates all security testing components
type ComprehensiveSecurityTestSuite struct {
	client             client.Client
	k8sClient          kubernetes.Interface
	config             *rest.Config
	namespace          string
	penetrationTester  *PenetrationTestSuite
	securityValidator  *AutomatedSecurityValidator
	continuousMonitor  *ContinuousSecurityMonitor
	regressionPipeline *SecurityRegressionPipeline
	testResults        *ComprehensiveTestResults
	mutex              sync.RWMutex
}

// ComprehensiveTestResults aggregates all security test results
type ComprehensiveTestResults struct {
	TestSuiteID             string                      `json:"test_suite_id"`
	ExecutionTimestamp      time.Time                   `json:"execution_timestamp"`
	TotalDuration           time.Duration               `json:"total_duration"`
	OverallStatus           string                      `json:"overall_status"`
	SecurityScore           float64                     `json:"security_score"`
	ComplianceScore         float64                     `json:"compliance_score"`
	ThreatDetections        int                         `json:"threat_detections"`
	Vulnerabilities         int                         `json:"vulnerabilities"`
	CriticalIssues          int                         `json:"critical_issues"`
	TestCategories          map[string]CategoryResult   `json:"test_categories"`
	DetailedResults         *DetailedSecurityResults    `json:"detailed_results"`
	ExecutionSummary        *ExecutionSummary           `json:"execution_summary"`
	ComplianceFrameworks    map[string]ComplianceResult `json:"compliance_frameworks"`
	SecurityRecommendations []SecurityRecommendation    `json:"security_recommendations"`
	TrendAnalysis           *SecurityTrendAnalysis      `json:"trend_analysis"`
}

// CategoryResult represents results for a specific test category
type CategoryResult struct {
	Category     string        `json:"category"`
	Status       string        `json:"status"`
	TestsRun     int           `json:"tests_run"`
	TestsPassed  int           `json:"tests_passed"`
	TestsFailed  int           `json:"tests_failed"`
	TestsSkipped int           `json:"tests_skipped"`
	Duration     time.Duration `json:"duration"`
	Score        float64       `json:"score"`
	Issues       []string      `json:"issues"`
	Findings     int           `json:"findings"`
}

// DetailedSecurityResults contains detailed results from each component
type DetailedSecurityResults struct {
	PenetrationTestResults    *TestResults               `json:"penetration_test_results"`
	SecurityValidationResults *SecurityValidationResults `json:"security_validation_results"`
	ContinuousMonitoringData  *MonitoringData            `json:"continuous_monitoring_data"`
	RegressionPipelineResults *PipelineExecutionResults  `json:"regression_pipeline_results"`
}

// ExecutionSummary provides high-level execution information
type ExecutionSummary struct {
	StartTime          time.Time     `json:"start_time"`
	EndTime            time.Time     `json:"end_time"`
	TotalExecutionTime time.Duration `json:"total_execution_time"`
	ParallelExecution  bool          `json:"parallel_execution"`
	TestEnvironment    string        `json:"test_environment"`
	KubernetesVersion  string        `json:"kubernetes_version"`
	TestDataGenerated  int64         `json:"test_data_generated_bytes"`
	ResourcesUsed      ResourceUsage `json:"resources_used"`
}

// ResourceUsage tracks resource consumption during testing
type ResourceUsage struct {
	PeakCPUUsage    float64 `json:"peak_cpu_usage"`
	PeakMemoryUsage float64 `json:"peak_memory_usage"`
	NetworkTraffic  int64   `json:"network_traffic_bytes"`
	StorageUsed     int64   `json:"storage_used_bytes"`
	TestPods        int     `json:"test_pods_created"`
}

// SecurityTrendAnalysis provides trend analysis across test executions
type SecurityTrendAnalysis struct {
	HistoricalScores     []float64 `json:"historical_scores"`
	TrendDirection       string    `json:"trend_direction"`
	ScoreImprovement     float64   `json:"score_improvement"`
	RecentRegression     bool      `json:"recent_regression"`
	PredictedFutureScore float64   `json:"predicted_future_score"`
	RecommendedActions   []string  `json:"recommended_actions"`
}

// NewComprehensiveSecurityTestSuite creates a new comprehensive test suite
func NewComprehensiveSecurityTestSuite(client client.Client, k8sClient kubernetes.Interface, config *rest.Config, namespace string) *ComprehensiveSecurityTestSuite {
	suite := &ComprehensiveSecurityTestSuite{
		client:    client,
		k8sClient: k8sClient,
		config:    config,
		namespace: namespace,
		testResults: &ComprehensiveTestResults{
			TestSuiteID:             fmt.Sprintf("comprehensive-security-test-%d", time.Now().Unix()),
			ExecutionTimestamp:      time.Now(),
			TestCategories:          make(map[string]CategoryResult),
			ComplianceFrameworks:    make(map[string]ComplianceResult),
			SecurityRecommendations: make([]SecurityRecommendation, 0),
		},
	}

	// Initialize component test suites
	suite.penetrationTester = NewPenetrationTestSuite(client, k8sClient, config, namespace, "https://localhost:8080")
	suite.securityValidator = NewAutomatedSecurityValidator(client, k8sClient, config, namespace)
	suite.continuousMonitor = NewContinuousSecurityMonitor(client, k8sClient, config, namespace)
	suite.regressionPipeline = NewSecurityRegressionPipeline(client, k8sClient, config, namespace)

	return suite
}

// ExecuteComprehensiveSecurityTests runs all security test components
func (suite *ComprehensiveSecurityTestSuite) ExecuteComprehensiveSecurityTests(ctx context.Context) *ComprehensiveTestResults {
	fmt.Println("🔒 Starting Comprehensive Security Validation...")
	startTime := time.Now()

	// Initialize test environment
	suite.setupTestEnvironment(ctx)

	// Execute test categories in parallel
	var wg sync.WaitGroup
	errorChan := make(chan error, 4)

	// Category 1: Penetration Testing
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("🎯 Executing Penetration Testing Suite...")
		result := suite.executePenetrationTesting(ctx)
		suite.addCategoryResult("penetration_testing", result)
	}()

	// Category 2: Security Control Validation
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("✅ Executing Security Control Validation...")
		result := suite.executeSecurityControlValidation(ctx)
		suite.addCategoryResult("security_control_validation", result)
	}()

	// Category 3: Continuous Security Monitoring
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("📊 Executing Continuous Security Monitoring...")
		result := suite.executeContinuousMonitoring(ctx)
		suite.addCategoryResult("continuous_monitoring", result)
	}()

	// Category 4: Security Regression Testing
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("🔄 Executing Security Regression Pipeline...")
		result := suite.executeRegressionTesting(ctx)
		suite.addCategoryResult("regression_testing", result)
	}()

	// Wait for all test categories to complete
	wg.Wait()
	close(errorChan)

	// Process any errors
	var errors []error
	for err := range errorChan {
		if err != nil {
			errors = append(errors, err)
		}
	}

	// Calculate overall results
	suite.calculateOverallResults()

	// Generate trend analysis
	suite.performTrendAnalysis(ctx)

	// Generate security recommendations
	suite.generateSecurityRecommendations()

	// Complete execution summary
	suite.testResults.TotalDuration = time.Since(startTime)
	suite.testResults.ExecutionSummary = &ExecutionSummary{
		StartTime:          startTime,
		EndTime:            time.Now(),
		TotalExecutionTime: suite.testResults.TotalDuration,
		ParallelExecution:  true,
		TestEnvironment:    "kubernetes",
		KubernetesVersion:  suite.getKubernetesVersion(ctx),
	}

	// Generate comprehensive report
	suite.generateComprehensiveReport()

	fmt.Printf("🏁 Comprehensive Security Testing Completed!\n")
	fmt.Printf("   Overall Status: %s\n", suite.testResults.OverallStatus)
	fmt.Printf("   Security Score: %.2f/100\n", suite.testResults.SecurityScore)
	fmt.Printf("   Compliance Score: %.2f/100\n", suite.testResults.ComplianceScore)
	fmt.Printf("   Execution Time: %s\n", suite.testResults.TotalDuration.String())

	return suite.testResults
}

// Test Category Execution Methods

func (suite *ComprehensiveSecurityTestSuite) executePenetrationTesting(ctx context.Context) CategoryResult {
	start := time.Now()
	result := CategoryResult{
		Category: "Penetration Testing",
		Status:   "running",
		Issues:   make([]string, 0),
	}

	// Execute penetration tests
	penetrationResults := suite.runPenetrationTests(ctx)

	// Process results
	result.TestsRun = len(penetrationResults.PenetrationResults)
	result.TestsPassed = 0
	result.TestsFailed = 0
	result.Vulnerabilities = penetrationResults.VulnerabilityCount

	for _, penResult := range penetrationResults.PenetrationResults {
		if penResult.Status == "passed" {
			result.TestsPassed++
		} else if penResult.Status == "failed" {
			result.TestsFailed++
			for _, vuln := range penResult.Vulnerabilities {
				result.Issues = append(result.Issues, vuln.Title)
			}
		} else {
			result.TestsSkipped++
		}
	}

	result.Score = penetrationResults.SecurityScore
	result.Duration = time.Since(start)

	if result.TestsFailed == 0 && result.Vulnerabilities == 0 {
		result.Status = "passed"
	} else {
		result.Status = "failed"
	}

	return result
}

func (suite *ComprehensiveSecurityTestSuite) executeSecurityControlValidation(ctx context.Context) CategoryResult {
	start := time.Now()
	result := CategoryResult{
		Category: "Security Control Validation",
		Status:   "running",
		Issues:   make([]string, 0),
	}

	// Execute security control validation
	validationResults := suite.runSecurityControlValidation(ctx)

	// Process results
	result.TestsRun = len(validationResults.SecurityControls)
	result.TestsPassed = 0
	result.TestsFailed = 0

	for _, control := range validationResults.SecurityControls {
		if control.Status == "passed" {
			result.TestsPassed++
		} else if control.Status == "failed" {
			result.TestsFailed++
			result.Issues = append(result.Issues, control.ControlName)
		} else {
			result.TestsSkipped++
		}
	}

	result.Score = validationResults.ComplianceScore
	result.Duration = time.Since(start)

	if result.TestsFailed == 0 {
		result.Status = "passed"
	} else {
		result.Status = "failed"
	}

	return result
}

func (suite *ComprehensiveSecurityTestSuite) executeContinuousMonitoring(ctx context.Context) CategoryResult {
	start := time.Now()
	result := CategoryResult{
		Category: "Continuous Monitoring",
		Status:   "running",
		Issues:   make([]string, 0),
	}

	// Execute continuous monitoring for a short period
	monitoringData := suite.runContinuousMonitoring(ctx, 2*time.Minute)

	// Process monitoring results
	result.TestsRun = 1 // Monitoring is a single long-running test
	result.Findings = len(monitoringData.ThreatDetections)

	if len(monitoringData.SecurityAlerts) == 0 && len(monitoringData.ThreatDetections) == 0 {
		result.TestsPassed = 1
		result.Status = "passed"
		result.Score = 100.0
	} else {
		result.TestsFailed = 1
		result.Status = "failed"
		result.Score = 70.0 // Reduced score due to threats/alerts

		for _, alert := range monitoringData.SecurityAlerts {
			result.Issues = append(result.Issues, alert.Title)
		}
	}

	result.Duration = time.Since(start)
	return result
}

func (suite *ComprehensiveSecurityTestSuite) executeRegressionTesting(ctx context.Context) CategoryResult {
	start := time.Now()
	result := CategoryResult{
		Category: "Regression Testing",
		Status:   "running",
		Issues:   make([]string, 0),
	}

	// Execute regression testing pipeline
	pipelineResults := suite.runRegressionPipeline(ctx)

	// Process pipeline results
	result.TestsRun = len(pipelineResults.StageResults)
	result.TestsPassed = 0
	result.TestsFailed = 0

	for _, stage := range pipelineResults.StageResults {
		if stage.Status == "completed" {
			result.TestsPassed++
		} else if stage.Status == "failed" {
			result.TestsFailed++
			result.Issues = append(result.Issues, fmt.Sprintf("Stage failed: %s", stage.StageName))
		} else {
			result.TestsSkipped++
		}
	}

	// Calculate score based on compliance results
	totalCompliance := 0.0
	complianceCount := 0
	for _, score := range pipelineResults.ComplianceResults {
		totalCompliance += score
		complianceCount++
	}

	if complianceCount > 0 {
		result.Score = totalCompliance / float64(complianceCount)
	} else {
		result.Score = 100.0
	}

	result.Duration = time.Since(start)

	if result.TestsFailed == 0 {
		result.Status = "passed"
	} else {
		result.Status = "failed"
	}

	return result
}

// Component Test Runners

func (suite *ComprehensiveSecurityTestSuite) runPenetrationTests(ctx context.Context) *TestResults {
	// Simulate comprehensive penetration testing
	// In real implementation, this would run the full penetration testing suite

	return &TestResults{
		TestID:             fmt.Sprintf("pen-test-%d", time.Now().Unix()),
		Timestamp:          time.Now(),
		Duration:           2 * time.Minute,
		TotalTests:         15,
		PassedTests:        13,
		FailedTests:        2,
		SecurityScore:      87.5,
		VulnerabilityCount: 2,
		PenetrationResults: []PenetrationResult{
			{
				TestName:      "API Security Testing",
				TestCategory:  "API",
				Status:        "passed",
				ExecutionTime: 30 * time.Second,
			},
			{
				TestName:      "Container Security Testing",
				TestCategory:  "Container",
				Status:        "failed",
				ExecutionTime: 45 * time.Second,
				Vulnerabilities: []SecurityIssue{
					{
						ID:          "CONT-001",
						Severity:    "MEDIUM",
						Title:       "Container running as root",
						Description: "Container detected running with root privileges",
						CVSS:        6.5,
					},
				},
			},
		},
		CriticalIssues: []SecurityIssue{},
	}
}

func (suite *ComprehensiveSecurityTestSuite) runSecurityControlValidation(ctx context.Context) *SecurityValidationResults {
	// Simulate security control validation
	// In real implementation, this would run the full validation suite

	return &SecurityValidationResults{
		ValidationID:    fmt.Sprintf("validation-%d", time.Now().Unix()),
		Timestamp:       time.Now(),
		Duration:        1 * time.Minute,
		TotalControls:   20,
		PassedControls:  18,
		FailedControls:  2,
		ComplianceScore: 90.0,
		SecurityControls: []SecurityControlResult{
			{
				ControlID:   "CONT-001",
				ControlName: "Non-Root User Enforcement",
				Category:    "Container Security",
				Status:      "passed",
				Severity:    "HIGH",
			},
			{
				ControlID:   "NET-001",
				ControlName: "Default Deny Network Policies",
				Category:    "Network Security",
				Status:      "failed",
				Severity:    "HIGH",
			},
		},
		ComplianceFrameworks: map[string]ComplianceResult{
			"NIST-CSF": {
				Framework:      "NIST Cybersecurity Framework",
				Score:          90.0,
				PassedControls: 18,
				FailedControls: 2,
			},
		},
	}
}

func (suite *ComprehensiveSecurityTestSuite) runContinuousMonitoring(ctx context.Context, duration time.Duration) *MonitoringData {
	// Simulate continuous monitoring for specified duration
	// In real implementation, this would start the monitoring system

	return &MonitoringData{
		StartTime:        time.Now(),
		LastUpdate:       time.Now().Add(duration),
		SecurityAlerts:   []SecurityAlert{},
		ThreatDetections: []ThreatDetection{},
		ComplianceDrifts: []ComplianceDrift{},
		MetricsCollection: map[string]MetricValue{
			"security_score": {
				Name:      "security_score",
				Value:     95.0,
				Unit:      "percentage",
				Timestamp: time.Now(),
				Status:    "normal",
			},
		},
		HealthChecks: map[string]HealthStatus{
			"api_server": {
				Component: "api_server",
				Status:    "healthy",
				LastCheck: time.Now(),
			},
		},
	}
}

func (suite *ComprehensiveSecurityTestSuite) runRegressionPipeline(ctx context.Context) *PipelineExecutionResults {
	// Simulate regression pipeline execution
	// In real implementation, this would run the full pipeline

	return &PipelineExecutionResults{
		ExecutionID:  fmt.Sprintf("pipeline-%d", time.Now().Unix()),
		PipelineID:   "security-regression",
		StartTime:    time.Now(),
		EndTime:      time.Now().Add(3 * time.Minute),
		Duration:     3 * time.Minute,
		Status:       "success",
		TriggerEvent: "manual",
		StageResults: []StageExecutionResult{
			{
				StageName: "security-scan",
				Status:    "completed",
				Duration:  1 * time.Minute,
			},
			{
				StageName: "compliance-check",
				Status:    "completed",
				Duration:  1 * time.Minute,
			},
		},
		ComplianceResults: map[string]float64{
			"NIST-CSF": 92.0,
			"CIS-K8S":  88.0,
		},
	}
}

// Analysis and Reporting Methods

func (suite *ComprehensiveSecurityTestSuite) calculateOverallResults() {
	suite.mutex.Lock()
	defer suite.mutex.Unlock()

	totalTests := 0
	totalPassed := 0
	totalFailed := 0
	totalSkipped := 0
	totalScore := 0.0
	categoryCount := 0
	allPassed := true

	for _, category := range suite.testResults.TestCategories {
		totalTests += category.TestsRun
		totalPassed += category.TestsPassed
		totalFailed += category.TestsFailed
		totalSkipped += category.TestsSkipped
		totalScore += category.Score
		categoryCount++

		if category.Status != "passed" {
			allPassed = false
		}
	}

	// Calculate overall status
	if allPassed && totalFailed == 0 {
		suite.testResults.OverallStatus = "PASSED"
	} else {
		suite.testResults.OverallStatus = "FAILED"
	}

	// Calculate average scores
	if categoryCount > 0 {
		suite.testResults.SecurityScore = totalScore / float64(categoryCount)
		suite.testResults.ComplianceScore = totalScore / float64(categoryCount)
	} else {
		suite.testResults.SecurityScore = 0.0
		suite.testResults.ComplianceScore = 0.0
	}

	// Aggregate detailed results
	suite.testResults.DetailedResults = &DetailedSecurityResults{
		PenetrationTestResults:    suite.getPenetrationResults(),
		SecurityValidationResults: suite.getValidationResults(),
		ContinuousMonitoringData:  suite.getMonitoringResults(),
		RegressionPipelineResults: suite.getRegressionResults(),
	}
}

func (suite *ComprehensiveSecurityTestSuite) performTrendAnalysis(ctx context.Context) {
	// Load historical test data
	historicalScores := suite.loadHistoricalScores()

	// Analyze trends
	currentScore := suite.testResults.SecurityScore

	trendAnalysis := &SecurityTrendAnalysis{
		HistoricalScores: historicalScores,
	}

	// Determine trend direction
	if len(historicalScores) > 0 {
		lastScore := historicalScores[len(historicalScores)-1]
		if currentScore > lastScore {
			trendAnalysis.TrendDirection = "improving"
			trendAnalysis.ScoreImprovement = currentScore - lastScore
		} else if currentScore < lastScore {
			trendAnalysis.TrendDirection = "declining"
			trendAnalysis.ScoreImprovement = currentScore - lastScore
			trendAnalysis.RecentRegression = true
		} else {
			trendAnalysis.TrendDirection = "stable"
			trendAnalysis.ScoreImprovement = 0.0
		}
	}

	// Generate predictions (simplified)
	if len(historicalScores) >= 3 {
		recentAvg := (historicalScores[len(historicalScores)-1] + historicalScores[len(historicalScores)-2] + historicalScores[len(historicalScores)-3]) / 3
		trendAnalysis.PredictedFutureScore = (recentAvg + currentScore) / 2
	} else {
		trendAnalysis.PredictedFutureScore = currentScore
	}

	// Generate recommendations
	if trendAnalysis.RecentRegression {
		trendAnalysis.RecommendedActions = append(trendAnalysis.RecommendedActions, "Investigate recent security regression")
	}
	if currentScore < 90.0 {
		trendAnalysis.RecommendedActions = append(trendAnalysis.RecommendedActions, "Focus on improving security controls")
	}

	suite.testResults.TrendAnalysis = trendAnalysis
}

func (suite *ComprehensiveSecurityTestSuite) generateSecurityRecommendations() {
	recommendations := make([]SecurityRecommendation, 0)

	// Generate recommendations based on test results
	for category, result := range suite.testResults.TestCategories {
		if result.Status == "failed" || result.Score < 80.0 {
			recommendation := SecurityRecommendation{
				ID:          fmt.Sprintf("REC-%s-%d", category, time.Now().Unix()),
				Priority:    "HIGH",
				Category:    result.Category,
				Title:       fmt.Sprintf("Improve %s", result.Category),
				Description: fmt.Sprintf("Address failures in %s category to improve security posture", result.Category),
				Impact:      "Improves overall security score and reduces vulnerabilities",
				Effort:      "MEDIUM",
				Timeline:    "2-4 weeks",
			}
			recommendations = append(recommendations, recommendation)
		}
	}

	// Add general recommendations
	if suite.testResults.SecurityScore < 90.0 {
		recommendations = append(recommendations, SecurityRecommendation{
			ID:          fmt.Sprintf("REC-GENERAL-%d", time.Now().Unix()),
			Priority:    "MEDIUM",
			Category:    "General Security",
			Title:       "Enhance Overall Security Posture",
			Description: "Implement comprehensive security improvements across all categories",
			Impact:      "Significantly improves security posture and compliance",
			Effort:      "HIGH",
			Timeline:    "6-8 weeks",
		})
	}

	suite.testResults.SecurityRecommendations = recommendations
}

// Helper Methods

func (suite *ComprehensiveSecurityTestSuite) setupTestEnvironment(ctx context.Context) {
	// Create test namespace if it doesn't exist
	namespace := &corev1.Namespace{
		ObjectMeta: corev1.ObjectMeta{
			Name: suite.namespace,
		},
	}
	utils.CreateNamespace(suite.k8sClient, namespace)
}

func (suite *ComprehensiveSecurityTestSuite) getKubernetesVersion(ctx context.Context) string {
	version, err := suite.k8sClient.Discovery().ServerVersion()
	if err != nil {
		return "unknown"
	}
	return version.GitVersion
}

func (suite *ComprehensiveSecurityTestSuite) addCategoryResult(category string, result CategoryResult) {
	suite.mutex.Lock()
	defer suite.mutex.Unlock()
	suite.testResults.TestCategories[category] = result
}

func (suite *ComprehensiveSecurityTestSuite) getPenetrationResults() *TestResults {
	// Return aggregated penetration test results
	return suite.penetrationTester.testResults
}

func (suite *ComprehensiveSecurityTestSuite) getValidationResults() *SecurityValidationResults {
	// Return aggregated validation results
	return suite.securityValidator.results
}

func (suite *ComprehensiveSecurityTestSuite) getMonitoringResults() *MonitoringData {
	// Return aggregated monitoring results
	return suite.continuousMonitor.monitoringData
}

func (suite *ComprehensiveSecurityTestSuite) getRegressionResults() *PipelineExecutionResults {
	// Return aggregated regression results
	return suite.regressionPipeline.executionResults
}

func (suite *ComprehensiveSecurityTestSuite) loadHistoricalScores() []float64 {
	// Load historical security scores from storage
	// This is a placeholder implementation
	return []float64{85.0, 87.5, 89.0, 92.0}
}

func (suite *ComprehensiveSecurityTestSuite) generateComprehensiveReport() {
	// Create reports directory
	reportsDir := "test-results/security/comprehensive"
	os.MkdirAll(reportsDir, 0755)

	// Generate JSON report
	jsonReport, _ := json.MarshalIndent(suite.testResults, "", "  ")
	jsonFile := filepath.Join(reportsDir, fmt.Sprintf("comprehensive-security-report-%s.json", suite.testResults.TestSuiteID))
	os.WriteFile(jsonFile, jsonReport, 0644)

	// Generate HTML report
	suite.generateHTMLComprehensiveReport(reportsDir)

	// Generate summary report
	suite.generateSummaryReport(reportsDir)

	fmt.Printf("📊 Reports generated in: %s\n", reportsDir)
}

func (suite *ComprehensiveSecurityTestSuite) generateHTMLComprehensiveReport(reportsDir string) {
	htmlTemplate := `<!DOCTYPE html>
<html>
<head>
    <title>Comprehensive Security Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 30px; border-radius: 10px; text-align: center; }
        .summary { display: flex; gap: 20px; margin: 20px 0; }
        .metric { background: #f8f9fa; padding: 20px; border-radius: 8px; text-align: center; flex: 1; }
        .metric h3 { margin: 0 0 10px 0; color: #333; }
        .metric .value { font-size: 2em; font-weight: bold; }
        .passed { color: #28a745; }
        .failed { color: #dc3545; }
        .warning { color: #ffc107; }
        .categories { margin: 20px 0; }
        .category { background: white; border: 1px solid #dee2e6; border-radius: 8px; margin: 10px 0; padding: 20px; }
        .category-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 15px; }
        .category-title { font-size: 1.2em; font-weight: bold; }
        .category-status { padding: 5px 15px; border-radius: 20px; color: white; text-transform: uppercase; font-size: 0.8em; }
        .status-passed { background-color: #28a745; }
        .status-failed { background-color: #dc3545; }
        .recommendations { margin: 20px 0; }
        .recommendation { background: #e9ecef; border-left: 4px solid #007bff; padding: 15px; margin: 10px 0; }
        table { width: 100%%; border-collapse: collapse; margin: 20px 0; }
        th, td { border: 1px solid #dee2e6; padding: 12px; text-align: left; }
        th { background-color: #f8f9fa; font-weight: bold; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🔒 Comprehensive Security Test Report</h1>
        <p>Test Suite ID: %s</p>
        <p>Executed: %s</p>
        <p>Duration: %s</p>
    </div>
    
    <div class="summary">
        <div class="metric">
            <h3>Overall Status</h3>
            <div class="value %s">%s</div>
        </div>
        <div class="metric">
            <h3>Security Score</h3>
            <div class="value">%.1f/100</div>
        </div>
        <div class="metric">
            <h3>Compliance Score</h3>
            <div class="value">%.1f/100</div>
        </div>
        <div class="metric">
            <h3>Categories Tested</h3>
            <div class="value">%d</div>
        </div>
    </div>
    
    <div class="categories">
        <h2>Test Categories</h2>
        <!-- Categories will be populated here -->
    </div>
    
    <div class="recommendations">
        <h2>Security Recommendations</h2>
        <p>Based on the test results, here are the recommended security improvements:</p>
        <!-- Recommendations will be populated here -->
    </div>
    
    <footer style="margin-top: 50px; padding: 20px; background: #f8f9fa; text-align: center; border-radius: 8px;">
        <p>Generated by Nephoran Security Testing Suite</p>
        <p>For questions or support, contact the security team</p>
    </footer>
</body>
</html>`

	statusClass := "failed"
	if suite.testResults.OverallStatus == "PASSED" {
		statusClass = "passed"
	}

	htmlContent := fmt.Sprintf(htmlTemplate,
		suite.testResults.TestSuiteID,
		suite.testResults.ExecutionTimestamp.Format(time.RFC3339),
		suite.testResults.TotalDuration.String(),
		statusClass,
		suite.testResults.OverallStatus,
		suite.testResults.SecurityScore,
		suite.testResults.ComplianceScore,
		len(suite.testResults.TestCategories),
	)

	htmlFile := filepath.Join(reportsDir, fmt.Sprintf("comprehensive-security-report-%s.html", suite.testResults.TestSuiteID))
	os.WriteFile(htmlFile, []byte(htmlContent), 0644)
}

func (suite *ComprehensiveSecurityTestSuite) generateSummaryReport(reportsDir string) {
	summary := fmt.Sprintf(`Comprehensive Security Test Summary
==========================================

Test Suite ID: %s
Execution Time: %s
Total Duration: %s

Overall Results:
- Status: %s
- Security Score: %.2f/100
- Compliance Score: %.2f/100
- Threat Detections: %d
- Critical Issues: %d

Category Results:
`, suite.testResults.TestSuiteID,
		suite.testResults.ExecutionTimestamp.Format("2006-01-02 15:04:05"),
		suite.testResults.TotalDuration.String(),
		suite.testResults.OverallStatus,
		suite.testResults.SecurityScore,
		suite.testResults.ComplianceScore,
		suite.testResults.ThreatDetections,
		suite.testResults.CriticalIssues)

	for _, category := range suite.testResults.TestCategories {
		summary += fmt.Sprintf("- %s: %s (Score: %.1f, Tests: %d/%d passed)\n",
			category.Category,
			category.Status,
			category.Score,
			category.TestsPassed,
			category.TestsRun)
	}

	summary += fmt.Sprintf("\nSecurity Recommendations: %d\n", len(suite.testResults.SecurityRecommendations))
	for i, rec := range suite.testResults.SecurityRecommendations {
		if i < 5 { // Show top 5 recommendations
			summary += fmt.Sprintf("- [%s] %s\n", rec.Priority, rec.Title)
		}
	}

	summaryFile := filepath.Join(reportsDir, fmt.Sprintf("security-test-summary-%s.txt", suite.testResults.TestSuiteID))
	os.WriteFile(summaryFile, []byte(summary), 0644)
}

// RunComprehensiveSecurityTests is the main entry point for comprehensive testing
func RunComprehensiveSecurityTests(client client.Client, k8sClient kubernetes.Interface, config *rest.Config, namespace string) *ComprehensiveTestResults {
	suite := NewComprehensiveSecurityTestSuite(client, k8sClient, config, namespace)
	return suite.ExecuteComprehensiveSecurityTests(context.Background())
}
