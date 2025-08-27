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
	"time"
)

// Additional types to fix compilation issues

// RecommendationRiskFactor identifies potential risks of implementing an optimization
type RecommendationRiskFactor struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Probability float64      `json:"probability"`
	Impact      ImpactLevel  `json:"impact"`
	Mitigation  string       `json:"mitigation"`
	Category    RiskCategory `json:"category"`
}

// RiskCategory represents different categories of risk
type RiskCategory string

const (
	RiskCategoryPerformance  RiskCategory = "performance"
	RiskCategoryAvailability RiskCategory = "availability"
	RiskCategorySecurity     RiskCategory = "security"
	RiskCategoryCompliance   RiskCategory = "compliance"
	RiskCategoryOperational  RiskCategory = "operational"
	RiskCategoryFinancial    RiskCategory = "financial"
)

// ImpactLevel represents the level of impact
type ImpactLevel string

const (
	ImpactCritical ImpactLevel = "critical"
	ImpactHigh     ImpactLevel = "high"
	ImpactMedium   ImpactLevel = "medium"
	ImpactLow      ImpactLevel = "low"
	ImpactMinimal  ImpactLevel = "minimal"
)

// RecommendationMitigationStrategy defines a strategy to mitigate risks
type RecommendationMitigationStrategy struct {
	Name            string        `json:"name"`
	Description     string        `json:"description"`
	Effectiveness   float64       `json:"effectiveness"`
	Cost            float64       `json:"cost"`
	TimeToImplement time.Duration `json:"timeToImplement"`
}

// OptimizationRiskAssessment contains comprehensive risk analysis for optimizations
type OptimizationRiskAssessment struct {
	OverallRiskLevel     RiskLevel                          `json:"overallRiskLevel"`
	ImplementationRisk   float64                            `json:"implementationRisk"`
	PerformanceRisk      float64                            `json:"performanceRisk"`
	AvailabilityRisk     float64                            `json:"availabilityRisk"`
	SecurityRisk         float64                            `json:"securityRisk"`
	ComplianceRisk       float64                            `json:"complianceRisk"`
	IdentifiedRisks      []RecommendationRiskFactor         `json:"identifiedRisks"`
	MitigationStrategies []RecommendationMitigationStrategy `json:"mitigationStrategies"`
	RiskScoreBreakdown   map[string]float64                 `json:"riskScoreBreakdown"`
}

// Add helper methods to convert predictors' types to recommendation engine types

// ConvertRiskAssessment converts the RiskAssessment from predictors to OptimizationRiskAssessment
func ConvertRiskAssessment(ra *RiskAssessment) *OptimizationRiskAssessment {
	if ra == nil {
		return &OptimizationRiskAssessment{
			OverallRiskLevel: RiskLevelMedium,
		}
	}

	return &OptimizationRiskAssessment{
		OverallRiskLevel:   scoreToRiskLevel(ra.OverallRiskScore),
		ImplementationRisk: ra.OverallRiskScore,
		PerformanceRisk:    ra.OverallRiskScore * 0.7,
		AvailabilityRisk:   ra.OverallRiskScore * 0.6,
		SecurityRisk:       ra.OverallRiskScore * 0.3,
		ComplianceRisk:     ra.OverallRiskScore * 0.3,
	}
}

func scoreToRiskLevel(score float64) RiskLevel {
	if score < 0.2 {
		return RiskLevelVeryLow
	} else if score < 0.4 {
		return RiskLevelLow
	} else if score < 0.6 {
		return RiskLevelMedium
	} else if score < 0.8 {
		return RiskLevelHigh
	}
	return RiskLevelVeryHigh
}
