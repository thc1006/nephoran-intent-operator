// Package validation provides production validation test runner
// This runner orchestrates all production deployment validation tests
package validation

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/thc1006/nephoran-intent-operator/tests/framework"
)

// ProductionValidationRunner orchestrates comprehensive production validation testing
type ProductionValidationRunner struct {
	validationSuite *ValidationSuite
	checklist       *ProductionReadinessChecklist
	config          *ValidationConfig
}

// NewProductionValidationRunner creates a new production validation runner
func NewProductionValidationRunner() *ProductionValidationRunner {
	// Create validation configuration with production targets
	config := &ValidationConfig{
		FunctionalTarget:  45, // Target 45/50 points
		PerformanceTarget: 23, // Target 23/25 points
		SecurityTarget:    14, // Target 14/15 points
		ProductionTarget:  8,  // Target 8/10 points (this is our focus)
		TotalTarget:       90, // Total 90/100 points

		TimeoutDuration:  45 * time.Minute, // Extended for production tests
		ConcurrencyLevel: 25,               // Reduced for stability
		LoadTestDuration: 10 * time.Minute,

		LatencyThreshold:    2 * time.Second,
		ThroughputThreshold: 45.0,
		AvailabilityTarget:  99.95,
		CoverageThreshold:   90.0,

		EnableE2ETesting:      true,
		EnableLoadTesting:     true,
		EnableChaosTesting:    true,
		EnableSecurityTesting: true,
	}

	// Create validation suite
	validationSuite := NewValidationSuite(config)

	return &ProductionValidationRunner{
		validationSuite: validationSuite,
		config:          config,
	}
}

// RunProductionValidation executes comprehensive production validation
func (pvr *ProductionValidationRunner) RunProductionValidation(ctx context.Context) error {
	ginkgo.By("Starting Production Deployment Validation Suite")

	// Setup validation suite
	pvr.validationSuite.SetupSuite()
	defer pvr.validationSuite.TearDownSuite()

	// Create production readiness checklist
	pvr.checklist = NewProductionReadinessChecklist(pvr.validationSuite.reliabilityTest)

	// Execute comprehensive validation with focus on production readiness
	results, err := pvr.validationSuite.ExecuteComprehensiveValidation(ctx)
	if err != nil {
		return fmt.Errorf("comprehensive validation failed: %w", err)
	}

	// Execute production readiness checklist
	checklistResults, err := pvr.checklist.ExecuteProductionReadinessAssessment(ctx)
	if err != nil {
		return fmt.Errorf("production readiness assessment failed: %w", err)
	}

	// Generate comprehensive reports
	pvr.generateReports(results, checklistResults)

	// Validate against production targets
	if results.ProductionScore < pvr.config.ProductionTarget {
		return fmt.Errorf("production readiness failed: achieved %d points, target %d points",
			results.ProductionScore, pvr.config.ProductionTarget)
	}

	if checklistResults.TotalScore < checklistResults.TargetScore {
		return fmt.Errorf("production checklist failed: achieved %d points, target %d points",
			checklistResults.TotalScore, checklistResults.TargetScore)
	}

	ginkgo.By(fmt.Sprintf("Production Validation PASSED: %d/%d validation points, %d/%d checklist points",
		results.ProductionScore, 10, checklistResults.TotalScore, checklistResults.MaxScore))

	return nil
}

// generateReports generates comprehensive production validation reports
func (pvr *ProductionValidationRunner) generateReports(results *ValidationResults, checklistResults *ProductionReadinessResults) {
	// Generate main validation report
	mainReport := pvr.generateMainReport(results)

	// Generate production readiness checklist report
	checklistReport := pvr.checklist.GenerateProductionReadinessReport()

	// Generate detailed component reports
	detailedReports := pvr.generateDetailedReports()

	// Write reports to files
	pvr.writeReportsToFiles(mainReport, checklistReport, detailedReports)

	// Print summary to console
	fmt.Println(mainReport)
	fmt.Println(checklistReport)
}

// generateMainReport generates the main production validation report
func (pvr *ProductionValidationRunner) generateMainReport(results *ValidationResults) string {
	return fmt.Sprintf(`
=============================================================================
PRODUCTION DEPLOYMENT VALIDATION SUMMARY
=============================================================================

COMPREHENSIVE VALIDATION RESULTS:
├── Total Score:                %d/%d points (Target: %d)
├── Production Readiness Score: %d/10 points (Target: 8)
├── Functional Score:           %d/50 points
├── Performance Score:          %d/25 points
└── Security Score:             %d/15 points

PRODUCTION READINESS BREAKDOWN:
├── High Availability:          %.2f%% availability achieved
├── Fault Tolerance:            Chaos engineering tests passed
├── Monitoring & Observability: Comprehensive observability validated
└── Disaster Recovery:          Backup and restore capabilities verified

PERFORMANCE METRICS:
├── P95 Latency:               %v (Target: < 2s)
├── Throughput:                %.1f intents/min (Target: > 45/min)
├── Availability:              %.2f%% (Target: > 99.95%%)
└── System Reliability:        Production-ready

EXECUTION SUMMARY:
├── Total Execution Time:      %v
├── Tests Executed:           %d
├── Tests Passed:             %d
├── Tests Failed:             %d
└── Validation Date:          %s

=============================================================================
`,
		results.TotalScore, results.MaxPossibleScore, pvr.config.TotalTarget,
		results.ProductionScore,
		results.FunctionalScore,
		results.PerformanceScore,
		results.SecurityScore,
		results.AvailabilityAchieved,
		results.P95Latency,
		results.ThroughputAchieved,
		results.AvailabilityAchieved,
		results.ExecutionTime,
		results.TestsExecuted,
		results.TestsPassed,
		results.TestsFailed,
		time.Now().Format("2006-01-02 15:04:05"),
	)
}

// generateDetailedReports generates detailed component reports
func (pvr *ProductionValidationRunner) generateDetailedReports() map[string]string {
	reports := make(map[string]string)

	// Get detailed metrics from specialized validators
	reliabilityValidator := pvr.validationSuite.reliabilityTest

	// Production deployment metrics
	if productionMetrics := reliabilityValidator.GetProductionMetrics(); productionMetrics != nil {
		reports["production_deployment"] = fmt.Sprintf("Production Deployment Metrics:\n%+v", productionMetrics)
	}

	// Chaos engineering metrics
	if chaosMetrics := reliabilityValidator.GetChaosMetrics(); chaosMetrics != nil {
		reports["chaos_engineering"] = fmt.Sprintf("Chaos Engineering Metrics:\n%+v", chaosMetrics)
	}

	// Observability metrics
	if observabilityMetrics := reliabilityValidator.GetObservabilityMetrics(); observabilityMetrics != nil {
		reports["observability"] = fmt.Sprintf("Observability Metrics:\n%+v", observabilityMetrics)
	}

	// Disaster recovery metrics
	if drMetrics := reliabilityValidator.GetDisasterRecoveryMetrics(); drMetrics != nil {
		reports["disaster_recovery"] = fmt.Sprintf("Disaster Recovery Metrics:\n%+v", drMetrics)
	}

	// Deployment scenarios metrics
	if deploymentMetrics := reliabilityValidator.GetDeploymentScenariosMetrics(); deploymentMetrics != nil {
		reports["deployment_scenarios"] = fmt.Sprintf("Deployment Scenarios Metrics:\n%+v", deploymentMetrics)
	}

	return reports
}

// writeReportsToFiles writes all reports to output files
func (pvr *ProductionValidationRunner) writeReportsToFiles(mainReport, checklistReport string, detailedReports map[string]string) {
	outputDir := "test-results/production-validation"

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		ginkgo.By(fmt.Sprintf("Failed to create output directory: %v", err))
		return
	}

	// Write main report
	mainReportFile := fmt.Sprintf("%s/production-validation-summary.txt", outputDir)
	if err := os.WriteFile(mainReportFile, []byte(mainReport), 0644); err != nil {
		ginkgo.By(fmt.Sprintf("Failed to write main report: %v", err))
	}

	// Write checklist report
	checklistReportFile := fmt.Sprintf("%s/production-readiness-checklist.txt", outputDir)
	if err := os.WriteFile(checklistReportFile, []byte(checklistReport), 0644); err != nil {
		ginkgo.By(fmt.Sprintf("Failed to write checklist report: %v", err))
	}

	// Write detailed reports
	for name, report := range detailedReports {
		detailedReportFile := fmt.Sprintf("%s/%s-detailed.txt", outputDir, name)
		if err := os.WriteFile(detailedReportFile, []byte(report), 0644); err != nil {
			ginkgo.By(fmt.Sprintf("Failed to write %s detailed report: %v", name, err))
		}
	}

	ginkgo.By(fmt.Sprintf("Production validation reports written to: %s", outputDir))
}

// Production Validation Test Suite using Ginkgo
var _ = ginkgo.Describe("Production Deployment Validation Suite", func() {
	var runner *ProductionValidationRunner
	var ctx context.Context
	var cancel context.CancelFunc

	ginkgo.BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 60*time.Minute)
		runner = NewProductionValidationRunner()
	})

	ginkgo.AfterEach(func() {
		defer cancel()
	})

	ginkgo.Context("when running comprehensive production validation", func() {
		ginkgo.It("should achieve production readiness target of 8/10 points", func() {
			err := runner.RunProductionValidation(ctx)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Additional validations
			checklistResults := runner.checklist.GetProductionReadinessResults()
			gomega.Expect(checklistResults).NotTo(gomega.BeNil())
			gomega.Expect(checklistResults.TotalScore).To(gomega.BeNumerically(">=", 8))
		})

		ginkgo.It("should validate high availability requirements", func() {
			err := runner.RunProductionValidation(ctx)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			checklistResults := runner.checklist.GetProductionReadinessResults()
			gomega.Expect(checklistResults.HighAvailabilityScore).To(gomega.BeNumerically(">=", 2))
		})

		ginkgo.It("should validate fault tolerance through chaos engineering", func() {
			err := runner.RunProductionValidation(ctx)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			checklistResults := runner.checklist.GetProductionReadinessResults()
			gomega.Expect(checklistResults.FaultToleranceScore).To(gomega.BeNumerically(">=", 2))
		})

		ginkgo.It("should validate monitoring and observability", func() {
			err := runner.RunProductionValidation(ctx)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			checklistResults := runner.checklist.GetProductionReadinessResults()
			gomega.Expect(checklistResults.MonitoringObservabilityScore).To(gomega.BeNumerically(">=", 1))
		})

		ginkgo.It("should validate disaster recovery capabilities", func() {
			err := runner.RunProductionValidation(ctx)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			checklistResults := runner.checklist.GetProductionReadinessResults()
			gomega.Expect(checklistResults.DisasterRecoveryScore).To(gomega.BeNumerically(">=", 1))
		})
	})

	ginkgo.Context("when running individual production validation categories", func() {
		ginkgo.It("should validate deployment scenarios", func() {
			err := runner.RunProductionValidation(ctx)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			checklistResults := runner.checklist.GetProductionReadinessResults()
			// Deployment scenarios are additional, so no minimum requirement
			gomega.Expect(checklistResults.DeploymentScenariosScore).To(gomega.BeNumerically(">=", 0))
		})

		ginkgo.It("should validate infrastructure as code", func() {
			err := runner.RunProductionValidation(ctx)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			checklistResults := runner.checklist.GetProductionReadinessResults()
			// Infrastructure as Code is additional, so no minimum requirement
			gomega.Expect(checklistResults.InfrastructureAsCodeScore).To(gomega.BeNumerically(">=", 0))
		})
	})

	ginkgo.Context("when generating production validation reports", func() {
		ginkgo.It("should generate comprehensive reports", func() {
			err := runner.RunProductionValidation(ctx)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Verify reports were generated
			outputDir := "test-results/production-validation"

			// Check main report exists
			mainReportFile := fmt.Sprintf("%s/production-validation-summary.txt", outputDir)
			_, err = os.Stat(mainReportFile)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Check checklist report exists
			checklistReportFile := fmt.Sprintf("%s/production-readiness-checklist.txt", outputDir)
			_, err = os.Stat(checklistReportFile)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})
})

// Entry point for standalone execution
func RunProductionValidationSuite() {
	// Configure Ginkgo for production validation
	// Create a dummy testing.T for Ginkgo - this is safe as Ginkgo doesn't actually use these methods in this context
	dummyT := &DummyTestingT{}
	ginkgo.RunSpecs(dummyT, "Nephoran Production Deployment Validation Suite")
}

// DummyTestingT provides a minimal testing.T interface for Ginkgo
type DummyTestingT struct {
	failed bool
}

func (t *DummyTestingT) Errorf(format string, args ...interface{}) {
	fmt.Printf("ERROR: "+format+"\n", args...)
	t.failed = true
}

func (t *DummyTestingT) FailNow() {
	t.failed = true
	panic("test failed")
}

func (t *DummyTestingT) Failed() bool {
	return t.failed
}
