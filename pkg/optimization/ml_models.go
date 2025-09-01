/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package optimization

import (
	"github.com/go-logr/logr"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// ExpectedImpact represents the expected impact of implementing an optimization
type ExpectedImpact struct {
	// Performance improvements
	LatencyReduction   float64 `json:"latencyReduction"`
	ThroughputIncrease float64 `json:"throughputIncrease"`
	EfficiencyGain     float64 `json:"efficiencyGain"`

	// Resource improvements
	ResourceSavings  float64 `json:"resourceSavings"`
	CostSavings      float64 `json:"costSavings"`
	EnergyEfficiency float64 `json:"energyEfficiency"`

	// Reliability improvements
	ReliabilityImprovement float64 `json:"reliabilityImprovement"`
	AvailabilityGain       float64 `json:"availabilityGain"`

	// Telecom-specific impacts
	SignalingEfficiencyGain float64 `json:"signalingEfficiencyGain"`
	SpectrumEfficiencyGain  float64 `json:"spectrumEfficiencyGain"`
	InteropImprovements     float64 `json:"interopImprovements"`

	// Quality of service
	SLAComplianceImprovement float64 `json:"slaComplianceImprovement"`
	UserExperienceGain       float64 `json:"userExperienceGain"`

	// Confidence metrics
	ConfidenceLevel float64 `json:"confidenceLevel"`
	DataQuality     float64 `json:"dataQuality"`
}

// OptimizationPriority represents the priority level of optimizations
type OptimizationPriority string

const (
	PriorityCritical OptimizationPriority = "critical"
	PriorityHigh     OptimizationPriority = "high"
	PriorityMedium   OptimizationPriority = "medium"
	PriorityLow      OptimizationPriority = "low"
)

// ImpactPredictor uses machine learning to predict optimization impacts
type ImpactPredictor struct {
	logger logr.Logger
	models map[OptimizationCategory]interface{} // ML models by category
}

// NewImpactPredictor creates a new impact predictor
func NewImpactPredictor(logger logr.Logger) *ImpactPredictor {
	return &ImpactPredictor{
		logger: logger.WithName("impact-predictor"),
		models: make(map[OptimizationCategory]interface{}),
	}
}

// PredictImpact predicts the expected impact of implementing a strategy
func (ip *ImpactPredictor) PredictImpact(
	strategy *OptimizationStrategyConfig,
	analysis *ComponentAnalysis,
	overallResult *PerformanceAnalysisResult,
) *ExpectedImpact {
	// Simplified impact prediction based on strategy benefits and current metrics
	baseBenefits := strategy.ExpectedBenefits

	// Scale benefits based on current performance issues
	scalingFactor := ip.calculateScalingFactor(analysis)

	impact := &ExpectedImpact{
		LatencyReduction:        baseBenefits.LatencyReduction * scalingFactor,
		ThroughputIncrease:      baseBenefits.ThroughputIncrease * scalingFactor,
		EfficiencyGain:          baseBenefits.EnergyEfficiencyGain * scalingFactor,
		ResourceSavings:         baseBenefits.ResourceSavings * scalingFactor,
		CostSavings:             baseBenefits.CostSavings * scalingFactor,
		ReliabilityImprovement:  baseBenefits.ReliabilityImprovement * scalingFactor,
		SignalingEfficiencyGain: baseBenefits.SignalingEfficiencyGain * scalingFactor,
		SpectrumEfficiencyGain:  baseBenefits.SpectrumEfficiencyGain * scalingFactor,
		InteropImprovements:     baseBenefits.InteropImprovements * scalingFactor,
		ConfidenceLevel:         0.8,  // Default confidence
		DataQuality:             0.85, // Default data quality
	}

	// Adjust based on component health  
	if analysis.HealthStatus == HealthStatusCritical {
		impact.ConfidenceLevel *= 0.9 // Slightly less confident with critical health
	}

	ip.logger.V(1).Info("Predicted impact",
		"strategy", strategy.Name,
		"scalingFactor", scalingFactor,
		"latencyReduction", impact.LatencyReduction)

	return impact
}

// calculateScalingFactor calculates how much to scale the expected benefits
func (ip *ImpactPredictor) calculateScalingFactor(analysis *ComponentAnalysis) float64 {
	// Base scaling factor
	scalingFactor := 1.0

	// Adjust based on performance score (lower score = higher potential impact)
	if analysis.PerformanceScore < 50 {
		scalingFactor *= 1.5 // High potential for improvement
	} else if analysis.PerformanceScore < 70 {
		scalingFactor *= 1.2 // Moderate potential
	} else if analysis.PerformanceScore > 90 {
		scalingFactor *= 0.7 // Low potential (already performing well)
	}

	// Adjust based on health status
	switch analysis.HealthStatus {
	case HealthStatusCritical:
		scalingFactor *= 2.0 // Critical systems have high improvement potential
	case HealthStatusWarning:
		scalingFactor *= 1.5
	case HealthStatusHealthy:
		scalingFactor *= 0.9 // Healthy systems have less potential
	}

	return scalingFactor
}

// ROICalculator calculates return on investment for optimizations
type ROICalculator struct {
	logger logr.Logger
}

// NewROICalculator creates a new ROI calculator
func NewROICalculator(logger logr.Logger) *ROICalculator {
	return &ROICalculator{
		logger: logger.WithName("roi-calculator"),
	}
}

// CalculateROI calculates the ROI for implementing an optimization
func (rc *ROICalculator) CalculateROI(
	strategy *OptimizationStrategyConfig,
	expectedImpact *ExpectedImpact,
	riskAssessment *RiskAssessment,
) float64 {
	// Calculate total benefits (simplified monetary conversion)
	totalBenefits := expectedImpact.CostSavings +
		(expectedImpact.ResourceSavings * 100) + // Convert resource savings to monetary value
		(expectedImpact.EfficiencyGain * 50) // Convert efficiency to monetary value

	// Calculate implementation costs (simplified estimate)
	implementationCost := rc.estimateImplementationCost(strategy)

	// Adjust for risk
	riskAdjustment := rc.calculateRiskAdjustment(riskAssessment)
	adjustedBenefits := totalBenefits * riskAdjustment

	// Calculate ROI as percentage
	if implementationCost > 0 {
		roi := ((adjustedBenefits - implementationCost) / implementationCost) * 100
		rc.logger.V(1).Info("Calculated ROI",
			"strategy", strategy.Name,
			"benefits", adjustedBenefits,
			"cost", implementationCost,
			"roi", roi)
		return roi
	}

	return 0.0
}

// estimateImplementationCost estimates the cost of implementing a strategy
func (rc *ROICalculator) estimateImplementationCost(strategy *OptimizationStrategyConfig) float64 {
	baseCost := 1000.0 // Base cost in arbitrary units

	// Scale cost based on number of implementation steps
	stepCost := float64(len(strategy.ImplementationSteps)) * 200.0

	// Scale cost based on automation level
	automationDiscount := 0.0
	for _, step := range strategy.ImplementationSteps {
		switch step.AutomationLevel {
		case AutomationFull:
			automationDiscount += 0.8 // 80% discount for fully automated
		case AutomationPartial:
			automationDiscount += 0.5 // 50% discount
		case AutomationAssisted:
			automationDiscount += 0.3 // 30% discount
		case AutomationManual:
			// No discount for manual steps
		}
	}

	totalCost := baseCost + stepCost
	totalCost *= (1.0 - (automationDiscount / float64(len(strategy.ImplementationSteps))))

	return totalCost
}

// calculateRiskAdjustment calculates risk adjustment factor for ROI
func (rc *ROICalculator) calculateRiskAdjustment(riskAssessment *RiskAssessment) float64 {
	// Start with 100% (no adjustment)
	adjustment := 1.0

	// Adjust based on overall risk level
	switch riskAssessment.OverallRiskLevel {
	case RiskLevelVeryLow:
		adjustment = 0.95 // Small discount
	case RiskLevelLow:
		adjustment = 0.90
	case RiskLevelMedium:
		adjustment = 0.80
	case RiskLevelHigh:
		adjustment = 0.65
	case RiskLevelVeryHigh:
		adjustment = 0.50 // Significant discount for high risk
	}

	return adjustment
}

// RiskAssessor assesses risks associated with optimization implementations
type RiskAssessor struct {
	logger logr.Logger
}

// NewRiskAssessor creates a new risk assessor
func NewRiskAssessor(logger logr.Logger) *RiskAssessor {
	return &RiskAssessor{
		logger: logger.WithName("risk-assessor"),
	}
}

// AssessRisk assesses the risk of implementing an optimization strategy
func (ra *RiskAssessor) AssessRisk(
	strategy *OptimizationStrategyConfig,
	analysis *ComponentAnalysis,
	overallResult *PerformanceAnalysisResult,
) *RiskAssessment {
	// Base risk assessment
	assessment := &RiskAssessment{
		OverallRiskLevel:     RiskLevelMedium, // Default to medium risk
		ImplementationRisk:   0.3,             // 30% base implementation risk
		PerformanceRisk:      0.2,             // 20% performance risk
		AvailabilityRisk:     0.2,             // 20% availability risk
		SecurityRisk:         0.1,             // 10% security risk
		ComplianceRisk:       0.1,             // 10% compliance risk
		IdentifiedRisks:      strategy.RiskFactors,
		MitigationStrategies: []MitigationStrategy{},
		RiskScoreBreakdown:   make(map[string]float64),
	}

	// Adjust risk based on component health
	healthRiskMultiplier := ra.getHealthRiskMultiplier(analysis.HealthStatus)
	assessment.ImplementationRisk *= healthRiskMultiplier
	assessment.PerformanceRisk *= healthRiskMultiplier

	// Adjust risk based on strategy category
	categoryRiskMultiplier := ra.getCategoryRiskMultiplier(strategy.Category)
	assessment.ImplementationRisk *= categoryRiskMultiplier

	// Calculate overall risk level based on individual risk scores
	totalRisk := assessment.ImplementationRisk + assessment.PerformanceRisk +
		assessment.AvailabilityRisk + assessment.SecurityRisk + assessment.ComplianceRisk

	assessment.OverallRiskLevel = ra.determineOverallRiskLevel(totalRisk)

	// Populate risk score breakdown
	assessment.RiskScoreBreakdown["implementation"] = assessment.ImplementationRisk * 100
	assessment.RiskScoreBreakdown["performance"] = assessment.PerformanceRisk * 100
	assessment.RiskScoreBreakdown["availability"] = assessment.AvailabilityRisk * 100
	assessment.RiskScoreBreakdown["security"] = assessment.SecurityRisk * 100
	assessment.RiskScoreBreakdown["compliance"] = assessment.ComplianceRisk * 100

	// Add mitigation strategies
	assessment.MitigationStrategies = ra.generateMitigationStrategies(strategy, assessment)

	ra.logger.V(1).Info("Assessed risk",
		"strategy", strategy.Name,
		"overallRiskLevel", assessment.OverallRiskLevel,
		"totalRisk", totalRisk)

	return assessment
}

// getHealthRiskMultiplier returns risk multiplier based on health status
func (ra *RiskAssessor) getHealthRiskMultiplier(healthStatus SystemHealthStatus) float64 {
	switch healthStatus {
	case HealthStatusCritical:
		return 2.0 // High risk for critical systems
	case HealthStatusWarning:
		return 1.5 // Moderate additional risk  
	case HealthStatusHealthy:
		return 1.0 // No additional risk
	case HealthStatusUnknown:
		return 1.2 // Slightly higher risk for unknown systems
	default:
		return 1.0
	}
}

// getCategoryRiskMultiplier returns risk multiplier based on optimization category
func (ra *RiskAssessor) getCategoryRiskMultiplier(category OptimizationCategory) float64 {
	switch category {
	case CategorySecurity:
		return 1.5 // Higher risk for security changes
	case CategoryCompliance:
		return 1.3 // Moderate risk for compliance changes
	case CategoryPerformance:
		return 1.2 // Slight additional risk
	case CategoryReliability:
		return 1.4 // Higher risk for reliability changes
	case CategoryTelecommunications:
		return 1.3 // Telecom changes carry additional risk
	default:
		return 1.0
	}
}

// determineOverallRiskLevel determines overall risk level from total risk score
func (ra *RiskAssessor) determineOverallRiskLevel(totalRisk float64) RiskLevel {
	if totalRisk < 0.3 {
		return RiskLevelVeryLow
	} else if totalRisk < 0.5 {
		return RiskLevelLow
	} else if totalRisk < 0.8 {
		return RiskLevelMedium
	} else if totalRisk < 1.2 {
		return RiskLevelHigh
	} else {
		return RiskLevelVeryHigh
	}
}

// generateMitigationStrategies generates mitigation strategies for identified risks
func (ra *RiskAssessor) generateMitigationStrategies(
	strategy *OptimizationStrategyConfig,
	assessment *RiskAssessment,
) []MitigationStrategy {
	var strategies []MitigationStrategy

	// Generate mitigation for high implementation risk
	if assessment.ImplementationRisk > 0.5 {
		strategies = append(strategies, MitigationStrategy{
			Name:            "Phased Implementation",
			Description:     "Implement optimization in phases to reduce risk",
			Effectiveness:   0.7,
			Cost:            500.0,
			TimeToImplement: 0, // No additional time
		})
	}

	// Generate mitigation for high performance risk
	if assessment.PerformanceRisk > 0.4 {
		strategies = append(strategies, MitigationStrategy{
			Name:            "Performance Monitoring",
			Description:     "Implement comprehensive performance monitoring during rollout",
			Effectiveness:   0.8,
			Cost:            300.0,
			TimeToImplement: 0,
		})
	}

	// Generate mitigation for high availability risk
	if assessment.AvailabilityRisk > 0.4 {
		strategies = append(strategies, MitigationStrategy{
			Name:            "Rollback Plan",
			Description:     "Prepare detailed rollback procedures",
			Effectiveness:   0.9,
			Cost:            200.0,
			TimeToImplement: 0,
		})
	}

	return strategies
}

// NewOptimizationKnowledgeBase creates a new knowledge base
// Note: OptimizationKnowledgeBase type is defined in types.go
func NewOptimizationKnowledgeBase() *OptimizationKnowledgeBase {
	return &OptimizationKnowledgeBase{
		BestPractices:       make(map[shared.ComponentType][]*OptimizationBestPractice),
		OptimizationHistory: make([]*OptimizationHistoryEntry, 0),
		KnownBottlenecks:    make(map[string]*BottleneckSolution),
		TelecomRules:        make([]*TelecomOptimizationRule, 0),
		// PerformanceBaselines: make(map[shared.ComponentType]*PerformanceBaseline), // Field doesn't exist in OptimizationKnowledgeBase
	}
}
