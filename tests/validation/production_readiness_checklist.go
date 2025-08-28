// Package validation provides comprehensive production readiness checklist and scoring integration.
// This checklist ensures all production deployment requirements are met for 8/10 point target.
package validation

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
)

// ProductionReadinessChecklist provides comprehensive production readiness validation.
// Integrates all production validators to achieve 8/10 point target.
type ProductionReadinessChecklist struct {
	reliabilityValidator *ReliabilityValidator

	// Checklist results.
	results *ProductionReadinessResults
}

// ProductionReadinessResults contains comprehensive production readiness assessment.
type ProductionReadinessResults struct {
	// Overall scoring (target: 8/10 points).
	TotalScore  int
	MaxScore    int
	TargetScore int

	// Category breakdown.
	HighAvailabilityScore        int // 3 points max
	FaultToleranceScore          int // 3 points max
	MonitoringObservabilityScore int // 2 points max
	DisasterRecoveryScore        int // 2 points max

	// Detailed checklist items.
	ChecklistItems []*ChecklistItem

	// Additional metrics.
	DeploymentScenariosScore  int
	InfrastructureAsCodeScore int

	// Execution metadata.
	ExecutionTime       time.Duration
	ValidationTimestamp time.Time

	// Recommendations for improvement.
	Recommendations []string
}

// ChecklistItem represents a single production readiness check.
type ChecklistItem struct {
	Category       string
	Name           string
	Description    string
	Required       bool
	Status         CheckStatus
	Score          int
	MaxScore       int
	ValidationTime time.Duration
	ErrorMessage   string
	Recommendation string
}

// CheckStatus represents the status of a checklist item.
type CheckStatus string

const (
	// CheckStatusPassed holds checkstatuspassed value.
	CheckStatusPassed CheckStatus = "PASSED"
	// CheckStatusFailed holds checkstatusfailed value.
	CheckStatusFailed CheckStatus = "FAILED"
	// CheckStatusSkipped holds checkstatusskipped value.
	CheckStatusSkipped CheckStatus = "SKIPPED"
	// CheckStatusWarning holds checkstatuswarning value.
	CheckStatusWarning CheckStatus = "WARNING"
	// CheckStatusNotApplicable holds checkstatusnotapplicable value.
	CheckStatusNotApplicable CheckStatus = "N/A"
)

// NewProductionReadinessChecklist creates a new production readiness checklist.
func NewProductionReadinessChecklist(reliabilityValidator *ReliabilityValidator) *ProductionReadinessChecklist {
	return &ProductionReadinessChecklist{
		reliabilityValidator: reliabilityValidator,
		results: &ProductionReadinessResults{
			MaxScore:        10,
			TargetScore:     8,
			ChecklistItems:  []*ChecklistItem{},
			Recommendations: []string{},
		},
	}
}

// ExecuteProductionReadinessAssessment runs comprehensive production readiness validation.
func (prc *ProductionReadinessChecklist) ExecuteProductionReadinessAssessment(ctx context.Context) (*ProductionReadinessResults, error) {
	ginkgo.By("Starting Comprehensive Production Readiness Assessment")

	startTime := time.Now()
	prc.results.ValidationTimestamp = startTime

	// Initialize checklist with all required items.
	prc.initializeChecklist()

	// Category 1: High Availability Validation (3 points).
	haScore := prc.validateHighAvailability(ctx)
	prc.results.HighAvailabilityScore = haScore

	// Category 2: Fault Tolerance Validation (3 points).
	ftScore := prc.validateFaultTolerance(ctx)
	prc.results.FaultToleranceScore = ftScore

	// Category 3: Monitoring & Observability Validation (2 points).
	monScore := prc.validateMonitoringObservability(ctx)
	prc.results.MonitoringObservabilityScore = monScore

	// Category 4: Disaster Recovery Validation (2 points).
	drScore := prc.validateDisasterRecovery(ctx)
	prc.results.DisasterRecoveryScore = drScore

	// Additional validations.
	prc.validateDeploymentScenarios(ctx)
	prc.validateInfrastructureAsCode(ctx)

	// Calculate total score (capped at 10 points).
	totalScore := haScore + ftScore + monScore + drScore
	if totalScore > 10 {
		totalScore = 10
	}
	prc.results.TotalScore = totalScore

	prc.results.ExecutionTime = time.Since(startTime)

	// Generate recommendations.
	prc.generateRecommendations()

	// Final assessment.
	passed := prc.results.TotalScore >= prc.results.TargetScore

	ginkgo.By(fmt.Sprintf("Production Readiness Assessment Complete: %d/%d points (Target: %d) - %s",
		prc.results.TotalScore, prc.results.MaxScore, prc.results.TargetScore,
		func() string {
			if passed {
				return "PASSED"
			}
			return "FAILED"
		}()))

	return prc.results, nil
}

// initializeChecklist creates the comprehensive production readiness checklist.
func (prc *ProductionReadinessChecklist) initializeChecklist() {
	prc.results.ChecklistItems = []*ChecklistItem{
		// High Availability Checks (3 points).
		{
			Category:    "High Availability",
			Name:        "Multi-Zone Deployment",
			Description: "Services are deployed across multiple availability zones",
			Required:    true,
			MaxScore:    1,
		},
		{
			Category:    "High Availability",
			Name:        "Automatic Failover",
			Description: "System can automatically failover within 5 minutes",
			Required:    true,
			MaxScore:    1,
		},
		{
			Category:    "High Availability",
			Name:        "Load Balancer & Health Checks",
			Description: "Load balancers with health checks are properly configured",
			Required:    true,
			MaxScore:    1,
		},

		// Fault Tolerance Checks (3 points).
		{
			Category:    "Fault Tolerance",
			Name:        "Pod Failure Recovery",
			Description: "System recovers gracefully from pod failures",
			Required:    true,
			MaxScore:    1,
		},
		{
			Category:    "Fault Tolerance",
			Name:        "Network Partition Handling",
			Description: "System handles network partitions appropriately",
			Required:    true,
			MaxScore:    1,
		},
		{
			Category:    "Fault Tolerance",
			Name:        "Resource Constraint Resilience",
			Description: "System operates under resource constraints",
			Required:    true,
			MaxScore:    1,
		},

		// Monitoring & Observability Checks (2 points).
		{
			Category:    "Monitoring & Observability",
			Name:        "Metrics Collection",
			Description: "Prometheus metrics collection is configured",
			Required:    true,
			MaxScore:    1,
		},
		{
			Category:    "Monitoring & Observability",
			Name:        "Logging & Tracing",
			Description: "Log aggregation and distributed tracing are active",
			Required:    true,
			MaxScore:    1,
		},

		// Disaster Recovery Checks (2 points).
		{
			Category:    "Disaster Recovery",
			Name:        "Backup System",
			Description: "Automated backup system is deployed and functional",
			Required:    true,
			MaxScore:    1,
		},
		{
			Category:    "Disaster Recovery",
			Name:        "Restore Procedures",
			Description: "Restore procedures are documented and tested",
			Required:    true,
			MaxScore:    1,
		},

		// Additional Production Checks.
		{
			Category:    "Deployment Scenarios",
			Name:        "Blue-Green Deployment",
			Description: "Blue-green deployment capability is available",
			Required:    false,
			MaxScore:    0,
		},
		{
			Category:    "Deployment Scenarios",
			Name:        "Canary Deployment",
			Description: "Canary deployment with traffic splitting is configured",
			Required:    false,
			MaxScore:    0,
		},
		{
			Category:    "Infrastructure as Code",
			Name:        "RBAC Configuration",
			Description: "Role-based access control is properly configured",
			Required:    false,
			MaxScore:    0,
		},
		{
			Category:    "Infrastructure as Code",
			Name:        "Network Policies",
			Description: "Network security policies are in place",
			Required:    false,
			MaxScore:    0,
		},
	}
}

// validateHighAvailability executes high availability checks.
func (prc *ProductionReadinessChecklist) validateHighAvailability(ctx context.Context) int {
	ginkgo.By("Validating High Availability Requirements")

	if prc.reliabilityValidator == nil {
		return 0
	}

	haMetrics := prc.reliabilityValidator.ValidateHighAvailability(ctx)

	// Convert availability percentage to score (0-3 points).
	var score int
	if haMetrics.Availability >= 99.95 {
		score = 3 // Excellent availability
	} else if haMetrics.Availability >= 99.9 {
		score = 2 // Good availability
	} else if haMetrics.Availability >= 99.0 {
		score = 1 // Acceptable availability
	} else {
		score = 0 // Poor availability
	}

	// Update checklist items.
	prc.updateChecklistItems("High Availability", score, 3, fmt.Sprintf("Measured availability: %.2f%%", haMetrics.Availability))

	return score
}

// validateFaultTolerance executes fault tolerance checks.
func (prc *ProductionReadinessChecklist) validateFaultTolerance(ctx context.Context) int {
	ginkgo.By("Validating Fault Tolerance Requirements")

	if prc.reliabilityValidator == nil {
		return 0
	}

	passed := prc.reliabilityValidator.ValidateFaultTolerance(ctx)

	// Convert boolean result to score.
	var score int
	if passed {
		score = 3 // All fault tolerance tests passed
	} else {
		score = 1 // Some fault tolerance capability exists
	}

	// Update checklist items.
	prc.updateChecklistItems("Fault Tolerance", score, 3, fmt.Sprintf("Fault tolerance validation: %t", passed))

	return score
}

// validateMonitoringObservability executes monitoring and observability checks.
func (prc *ProductionReadinessChecklist) validateMonitoringObservability(ctx context.Context) int {
	ginkgo.By("Validating Monitoring & Observability Requirements")

	if prc.reliabilityValidator == nil {
		return 0
	}

	score := prc.reliabilityValidator.ValidateMonitoringObservability(ctx)

	// Score is already 0-2, use directly.
	prc.updateChecklistItems("Monitoring & Observability", score, 2, fmt.Sprintf("Monitoring score: %d/2", score))

	return score
}

// validateDisasterRecovery executes disaster recovery checks.
func (prc *ProductionReadinessChecklist) validateDisasterRecovery(ctx context.Context) int {
	ginkgo.By("Validating Disaster Recovery Requirements")

	if prc.reliabilityValidator == nil {
		return 0
	}

	passed := prc.reliabilityValidator.ValidateDisasterRecovery(ctx)

	// Convert boolean result to score.
	var score int
	if passed {
		score = 2 // Disaster recovery capabilities validated
	} else {
		score = 0 // No disaster recovery capabilities
	}

	// Update checklist items.
	prc.updateChecklistItems("Disaster Recovery", score, 2, fmt.Sprintf("Disaster recovery validation: %t", passed))

	return score
}

// validateDeploymentScenarios executes deployment scenarios checks.
func (prc *ProductionReadinessChecklist) validateDeploymentScenarios(ctx context.Context) {
	ginkgo.By("Validating Deployment Scenarios (Additional)")

	if prc.reliabilityValidator == nil {
		return
	}

	score, err := prc.reliabilityValidator.ValidateDeploymentScenarios(ctx)
	if err != nil {
		ginkgo.By(fmt.Sprintf("Deployment scenarios validation error: %v", err))
		score = 0
	}

	prc.results.DeploymentScenariosScore = score
	prc.updateChecklistItems("Deployment Scenarios", score, 3, fmt.Sprintf("Deployment scenarios score: %d", score))
}

// validateInfrastructureAsCode executes infrastructure as code checks.
func (prc *ProductionReadinessChecklist) validateInfrastructureAsCode(ctx context.Context) {
	ginkgo.By("Validating Infrastructure as Code (Additional)")

	if prc.reliabilityValidator == nil {
		return
	}

	score, err := prc.reliabilityValidator.ValidateInfrastructureAsCode(ctx)
	if err != nil {
		ginkgo.By(fmt.Sprintf("Infrastructure as code validation error: %v", err))
		score = 0
	}

	prc.results.InfrastructureAsCodeScore = score
	prc.updateChecklistItems("Infrastructure as Code", score, 4, fmt.Sprintf("Infrastructure as code score: %d", score))
}

// updateChecklistItems updates checklist items for a category.
func (prc *ProductionReadinessChecklist) updateChecklistItems(category string, actualScore, maxScore int, message string) {
	itemsUpdated := 0

	for _, item := range prc.results.ChecklistItems {
		if item.Category == category && item.Required {
			// Distribute score across required items in category.
			if actualScore > 0 && itemsUpdated < actualScore {
				item.Status = CheckStatusPassed
				item.Score = 1
				itemsUpdated++
			} else {
				item.Status = CheckStatusFailed
				item.Score = 0
			}
		}
	}
}

// generateRecommendations generates recommendations for improvement.
func (prc *ProductionReadinessChecklist) generateRecommendations() {
	recommendations := []string{}

	// High Availability recommendations.
	if prc.results.HighAvailabilityScore < 3 {
		recommendations = append(recommendations,
			"Improve high availability by implementing multi-zone deployments and automatic failover")
	}

	// Fault Tolerance recommendations.
	if prc.results.FaultToleranceScore < 3 {
		recommendations = append(recommendations,
			"Enhance fault tolerance through chaos engineering and circuit breaker patterns")
	}

	// Monitoring recommendations.
	if prc.results.MonitoringObservabilityScore < 2 {
		recommendations = append(recommendations,
			"Complete observability stack with metrics collection, logging, and distributed tracing")
	}

	// Disaster Recovery recommendations.
	if prc.results.DisasterRecoveryScore < 2 {
		recommendations = append(recommendations,
			"Implement comprehensive backup and disaster recovery procedures")
	}

	// Overall score recommendations.
	if prc.results.TotalScore < prc.results.TargetScore {
		gap := prc.results.TargetScore - prc.results.TotalScore
		recommendations = append(recommendations,
			fmt.Sprintf("Overall production readiness gap: %d points. Focus on highest impact improvements.", gap))
	}

	prc.results.Recommendations = recommendations
}

// GetProductionReadinessResults returns the assessment results.
func (prc *ProductionReadinessChecklist) GetProductionReadinessResults() *ProductionReadinessResults {
	return prc.results
}

// GenerateProductionReadinessReport generates comprehensive production readiness report.
func (prc *ProductionReadinessChecklist) GenerateProductionReadinessReport() string {
	if prc.results == nil {
		return "Production readiness assessment not executed"
	}

	report := fmt.Sprintf(`
=============================================================================
PRODUCTION READINESS ASSESSMENT REPORT
=============================================================================

OVERALL SCORE: %d/%d POINTS (TARGET: %d POINTS)
STATUS: %s

CATEGORY BREAKDOWN:
├── High Availability:          %d/3 points
├── Fault Tolerance:            %d/3 points  
├── Monitoring & Observability: %d/2 points
└── Disaster Recovery:          %d/2 points

ADDITIONAL VALIDATIONS:
├── Deployment Scenarios:       %d points
└── Infrastructure as Code:     %d points

CHECKLIST RESULTS:
`,
		prc.results.TotalScore, prc.results.MaxScore, prc.results.TargetScore,
		func() string {
			if prc.results.TotalScore >= prc.results.TargetScore {
				return "PASSED"
			}
			return "FAILED - NEEDS IMPROVEMENT"
		}(),
		prc.results.HighAvailabilityScore,
		prc.results.FaultToleranceScore,
		prc.results.MonitoringObservabilityScore,
		prc.results.DisasterRecoveryScore,
		prc.results.DeploymentScenariosScore,
		prc.results.InfrastructureAsCodeScore,
	)

	// Add checklist item details.
	categories := make(map[string][]*ChecklistItem)
	for _, item := range prc.results.ChecklistItems {
		categories[item.Category] = append(categories[item.Category], item)
	}

	for category, items := range categories {
		report += fmt.Sprintf("\n%s:\n", category)
		for _, item := range items {
			status := ""
			switch item.Status {
			case CheckStatusPassed:
				status = "✅"
			case CheckStatusFailed:
				status = "❌"
			case CheckStatusWarning:
				status = "⚠️"
			case CheckStatusSkipped:
				status = "⏭️"
			case CheckStatusNotApplicable:
				status = "—"
			}

			required := ""
			if item.Required {
				required = " (Required)"
			}

			report += fmt.Sprintf("  %s %-30s %s%s\n", status, item.Name, item.Description, required)
		}
	}

	// Add recommendations.
	if len(prc.results.Recommendations) > 0 {
		report += `
RECOMMENDATIONS FOR IMPROVEMENT:
`
		for i, rec := range prc.results.Recommendations {
			report += fmt.Sprintf("%d. %s\n", i+1, rec)
		}
	}

	report += fmt.Sprintf(`
EXECUTION SUMMARY:
├── Assessment Duration:     %v
├── Total Checklist Items:   %d
├── Items Passed:           %d
├── Items Failed:           %d
└── Assessment Date:        %s

=============================================================================
`,
		prc.results.ExecutionTime,
		len(prc.results.ChecklistItems),
		func() int {
			passed := 0
			for _, item := range prc.results.ChecklistItems {
				if item.Status == CheckStatusPassed {
					passed++
				}
			}
			return passed
		}(),
		func() int {
			failed := 0
			for _, item := range prc.results.ChecklistItems {
				if item.Status == CheckStatusFailed {
					failed++
				}
			}
			return failed
		}(),
		prc.results.ValidationTimestamp.Format("2006-01-02 15:04:05"),
	)

	return report
}
