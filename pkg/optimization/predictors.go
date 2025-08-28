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
	"context"
	"math"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring/reporting"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// ImpactPredictor predicts the impact of optimization recommendations.
type ImpactPredictor struct {
	logger  logr.Logger
	models  map[string]*PredictionModel
	metrics *PredictorMetrics
	weights map[shared.ComponentType]float64
}

// PredictionModel represents a machine learning model for impact prediction.
type PredictionModel struct {
	Type        string                 `json:"type"`        // linear, polynomial, neural_network
	Features    []string               `json:"features"`    // input features
	Weights     []float64              `json:"weights"`     // model weights
	Bias        float64                `json:"bias"`        // model bias
	Accuracy    float64                `json:"accuracy"`    // model accuracy (0-1)
	LastTrained time.Time              `json:"lastTrained"` // last training time
	Metadata    map[string]interface{} `json:"metadata"`    // additional model metadata
}

// PredictorMetrics tracks performance metrics for predictors.
type PredictorMetrics struct {
	PredictionsCount int64     `json:"predictionsCount"`
	AccuracyScore    float64   `json:"accuracyScore"`
	AverageLatency   float64   `json:"averageLatency"` // milliseconds
	PredictionErrors int64     `json:"predictionErrors"`
	LastUpdateTime   time.Time `json:"lastUpdateTime"`
}

// ImpactPrediction represents a predicted impact of an optimization.
type ImpactPrediction struct {
	ComponentType        shared.ComponentType           `json:"componentType"`
	OptimizationType     string                         `json:"optimizationType"`
	PredictedImprovement float64                        `json:"predictedImprovement"` // percentage
	ConfidenceScore      float64                        `json:"confidenceScore"`      // 0-1
	TimeToEffect         time.Duration                  `json:"timeToEffect"`         // expected time to see effect
	RiskScore            float64                        `json:"riskScore"`            // 0-1 (higher = riskier)
	ResourceRequirements reporting.ResourceUsageMetrics `json:"resourceRequirements"`
	Reasoning            string                         `json:"reasoning"`
}

// Note: ComponentType is defined in performance_analysis_engine.go.

// NewImpactPredictor creates a new impact predictor.
func NewImpactPredictor(logger logr.Logger) *ImpactPredictor {
	return &ImpactPredictor{
		logger:  logger.WithName("impact-predictor"),
		models:  make(map[string]*PredictionModel),
		metrics: &PredictorMetrics{},
		weights: map[shared.ComponentType]float64{
			shared.ComponentTypeLLMProcessor: 0.4,
			shared.ComponentTypeRAGSystem:    0.3,
			shared.ComponentTypeCache:        0.2,
			shared.ComponentTypeNetworking:   0.05,
			shared.ComponentTypeDatabase:     0.03,
			shared.ComponentTypeKubernetes:   0.02,
		},
	}
}

// PredictImpact predicts the impact of a specific optimization.
func (ip *ImpactPredictor) PredictImpact(ctx context.Context, component shared.ComponentType, optimization string, metrics reporting.ResourceUsageMetrics) (*ImpactPrediction, error) {
	ip.metrics.PredictionsCount++

	// Simple heuristic-based prediction for now.
	// In production, this would use trained ML models.
	prediction := &ImpactPrediction{
		ComponentType:        component,
		OptimizationType:     optimization,
		ConfidenceScore:      0.75, // default confidence
		TimeToEffect:         5 * time.Minute,
		RiskScore:            0.3, // moderate risk
		ResourceRequirements: metrics,
	}

	// Calculate predicted improvement based on component type and optimization.
	improvement, reasoning := ip.calculateImprovement(component, optimization, metrics)
	prediction.PredictedImprovement = improvement
	prediction.Reasoning = reasoning

	// Adjust confidence based on historical accuracy.
	if model, exists := ip.models[string(component)]; exists {
		prediction.ConfidenceScore = model.Accuracy
	}

	return prediction, nil
}

// calculateImprovement calculates the expected improvement for an optimization.
func (ip *ImpactPredictor) calculateImprovement(component shared.ComponentType, optimization string, metrics reporting.ResourceUsageMetrics) (float64, string) {
	baseImprovement := 10.0 // 10% base improvement
	reasoning := "Based on historical patterns and system analysis"

	switch component {
	case shared.ComponentTypeLLMProcessor:
		if metrics.CPUUtilization > 80 {
			baseImprovement *= 1.5
			reasoning += ": High CPU utilization indicates significant optimization potential"
		}
		if optimization == "batch_processing" {
			baseImprovement *= 1.3
			reasoning += ": Batch processing typically provides 30% additional improvement"
		}
	case shared.ComponentTypeRAGSystem:
		if metrics.MemoryUtilization > 70 {
			baseImprovement *= 1.4
			reasoning += ": High memory usage suggests caching opportunities"
		}
	case shared.ComponentTypeCache:
		baseImprovement *= 2.0 // Cache optimizations typically have high impact
		reasoning += ": Cache optimizations typically provide substantial performance gains"
	}

	// Cap improvement at 80% to be realistic.
	return math.Min(baseImprovement, 80.0), reasoning
}

// ROICalculator calculates return on investment for optimizations.
type ROICalculator struct {
	logger     logr.Logger
	costModels map[shared.ComponentType]*CostModel
	metrics    *ROIMetrics
}

// CostModel represents cost modeling for different components.
type CostModel struct {
	FixedCosts      map[string]float64 `json:"fixedCosts"`      // Fixed costs per time period
	VariableCosts   map[string]float64 `json:"variableCosts"`   // Variable costs per unit
	Depreciation    float64            `json:"depreciation"`    // Depreciation rate
	MaintenanceCost float64            `json:"maintenanceCost"` // Maintenance cost percentage
}

// ROIMetrics tracks ROI calculation metrics.
type ROIMetrics struct {
	CalculationsCount   int64     `json:"calculationsCount"`
	AverageROI          float64   `json:"averageROI"`
	PositiveROICount    int64     `json:"positiveROICount"`
	NegativeROICount    int64     `json:"negativeROICount"`
	TotalCostSavings    float64   `json:"totalCostSavings"`
	LastCalculationTime time.Time `json:"lastCalculationTime"`
}

// ROICalculation represents a return on investment calculation.
type ROICalculation struct {
	OptimizationType  string        `json:"optimizationType"`
	InitialInvestment float64       `json:"initialInvestment"` // dollars
	ExpectedSavings   float64       `json:"expectedSavings"`   // dollars per month
	PaybackPeriod     time.Duration `json:"paybackPeriod"`     // time to break even
	ROIPercentage     float64       `json:"roiPercentage"`     // percentage ROI
	NPV               float64       `json:"npv"`               // Net Present Value
	IRR               float64       `json:"irr"`               // Internal Rate of Return
	BreakEvenPoint    time.Time     `json:"breakEvenPoint"`    // date when ROI becomes positive
	RiskAdjustedROI   float64       `json:"riskAdjustedROI"`   // ROI adjusted for risk
	ConfidenceLevel   float64       `json:"confidenceLevel"`   // 0-1
}

// NewROICalculator creates a new ROI calculator.
func NewROICalculator(logger logr.Logger) *ROICalculator {
	return &ROICalculator{
		logger:     logger.WithName("roi-calculator"),
		costModels: make(map[shared.ComponentType]*CostModel),
		metrics:    &ROIMetrics{},
	}
}

// CalculateROI calculates the return on investment for an optimization.
func (rc *ROICalculator) CalculateROI(ctx context.Context, component shared.ComponentType, optimization string, impact *ImpactPrediction) (*ROICalculation, error) {
	rc.metrics.CalculationsCount++
	rc.metrics.LastCalculationTime = time.Now()

	// Estimate initial investment based on optimization type.
	initialInvestment := rc.estimateInitialInvestment(component, optimization)

	// Estimate monthly savings based on predicted improvement.
	monthlySavings := rc.estimateMonthlySavings(component, impact.PredictedImprovement)

	// Calculate basic ROI metrics.
	calculation := &ROICalculation{
		OptimizationType:  optimization,
		InitialInvestment: initialInvestment,
		ExpectedSavings:   monthlySavings,
		ConfidenceLevel:   impact.ConfidenceScore,
	}

	// Calculate payback period.
	if monthlySavings > 0 {
		paybackMonths := initialInvestment / monthlySavings
		calculation.PaybackPeriod = time.Duration(paybackMonths*30*24) * time.Hour
		calculation.BreakEvenPoint = time.Now().Add(calculation.PaybackPeriod)
	}

	// Calculate ROI percentage (annual).
	if initialInvestment > 0 {
		annualSavings := monthlySavings * 12
		calculation.ROIPercentage = ((annualSavings - initialInvestment) / initialInvestment) * 100
	}

	// Risk-adjusted ROI.
	riskMultiplier := 1.0 - impact.RiskScore
	calculation.RiskAdjustedROI = calculation.ROIPercentage * riskMultiplier

	// Track metrics.
	if calculation.ROIPercentage > 0 {
		rc.metrics.PositiveROICount++
	} else {
		rc.metrics.NegativeROICount++
	}

	rc.metrics.TotalCostSavings += monthlySavings * 12

	return calculation, nil
}

// estimateInitialInvestment estimates the upfront cost of an optimization.
func (rc *ROICalculator) estimateInitialInvestment(component shared.ComponentType, optimization string) float64 {
	baseCost := 1000.0 // Base cost in dollars

	switch component {
	case shared.ComponentTypeLLMProcessor:
		baseCost = 5000.0 // Higher cost for LLM optimizations
	case shared.ComponentTypeRAGSystem:
		baseCost = 3000.0
	case shared.ComponentTypeCache:
		baseCost = 2000.0
	case shared.ComponentTypeNetworking:
		baseCost = 4000.0
	case shared.ComponentTypeDatabase:
		baseCost = 2500.0
	case shared.ComponentTypeKubernetes:
		baseCost = 1500.0
	}

	// Adjust based on optimization type.
	switch optimization {
	case "batch_processing":
		baseCost *= 1.5
	case "caching_improvement":
		baseCost *= 1.2
	case "resource_scaling":
		baseCost *= 0.8
	case "algorithm_optimization":
		baseCost *= 2.0
	}

	return baseCost
}

// estimateMonthlySavings estimates monthly cost savings from an optimization.
func (rc *ROICalculator) estimateMonthlySavings(component shared.ComponentType, improvementPercent float64) float64 {
	// Base monthly operational cost.
	baseMonthlyCost := 10000.0

	switch component {
	case shared.ComponentTypeLLMProcessor:
		baseMonthlyCost = 20000.0 // LLM API costs are high
	case shared.ComponentTypeRAGSystem:
		baseMonthlyCost = 8000.0
	case shared.ComponentTypeCache:
		baseMonthlyCost = 5000.0
	case shared.ComponentTypeNetworking:
		baseMonthlyCost = 3000.0
	case shared.ComponentTypeDatabase:
		baseMonthlyCost = 4000.0
	case shared.ComponentTypeKubernetes:
		baseMonthlyCost = 12000.0
	}

	// Calculate savings as percentage of base cost.
	return baseMonthlyCost * (improvementPercent / 100.0)
}

// RiskAssessor assesses risks associated with optimizations.
type RiskAssessor struct {
	logger     logr.Logger
	riskModels map[shared.ComponentType]*RiskModel
	metrics    *RiskMetrics
}

// RiskModel represents risk assessment model for components.
type RiskModel struct {
	ComponentType        shared.ComponentType `json:"componentType"`
	BaseRiskScore        float64              `json:"baseRiskScore"`        // 0-1
	RiskFactors          map[string]float64   `json:"riskFactors"`          // factor name -> weight
	MitigationStrategies map[string]string    `json:"mitigationStrategies"` // risk -> mitigation
	HistoricalFailures   int                  `json:"historicalFailures"`
	LastIncidentTime     time.Time            `json:"lastIncidentTime"`
}

// RiskMetrics tracks risk assessment metrics.
type RiskMetrics struct {
	AssessmentsCount   int64     `json:"assessmentsCount"`
	AverageRiskScore   float64   `json:"averageRiskScore"`
	HighRiskCount      int64     `json:"highRiskCount"`
	MediumRiskCount    int64     `json:"mediumRiskCount"`
	LowRiskCount       int64     `json:"lowRiskCount"`
	LastAssessmentTime time.Time `json:"lastAssessmentTime"`
}

// RiskAssessment represents a comprehensive risk assessment.
type RiskAssessment struct {
	OptimizationType      string               `json:"optimizationType"`
	ComponentType         shared.ComponentType `json:"componentType"`
	OverallRiskScore      float64              `json:"overallRiskScore"` // 0-1 (higher = riskier)
	RiskLevel             string               `json:"riskLevel"`        // Low, Medium, High, Critical
	RiskFactors           []RiskFactor         `json:"riskFactors"`
	MitigationStrategies  []MitigationStrategy `json:"mitigationStrategies"`
	RecommendedActions    []string             `json:"recommendedActions"`
	MaxAcceptableDowntime time.Duration        `json:"maxAcceptableDowntime"`
	EstimatedDowntime     time.Duration        `json:"estimatedDowntime"`
	BusinessImpact        string               `json:"businessImpact"`
	TechnicalComplexity   int                  `json:"technicalComplexity"` // 1-10
	ReversibilityScore    float64              `json:"reversibilityScore"`  // 0-1 (higher = easier to reverse)
}

// RiskFactor represents an individual risk factor.
type RiskFactor struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Probability float64 `json:"probability"` // 0-1
	Impact      float64 `json:"impact"`      // 0-1
	RiskScore   float64 `json:"riskScore"`   // probability * impact
	Category    string  `json:"category"`    // Technical, Business, Operational
}

// MitigationStrategy represents a risk mitigation strategy.
type MitigationStrategy struct {
	RiskFactor         string        `json:"riskFactor"`
	Strategy           string        `json:"strategy"`
	Effectiveness      float64       `json:"effectiveness"` // 0-1
	ImplementationTime time.Duration `json:"implementationTime"`
	Cost               float64       `json:"cost"`
	Priority           string        `json:"priority"` // Low, Medium, High
}

// NewRiskAssessor creates a new risk assessor.
func NewRiskAssessor(logger logr.Logger) *RiskAssessor {
	return &RiskAssessor{
		logger:     logger.WithName("risk-assessor"),
		riskModels: make(map[shared.ComponentType]*RiskModel),
		metrics:    &RiskMetrics{},
	}
}

// AssessRisk performs a comprehensive risk assessment for an optimization.
func (ra *RiskAssessor) AssessRisk(ctx context.Context, component shared.ComponentType, optimization string, impact *ImpactPrediction) (*RiskAssessment, error) {
	ra.metrics.AssessmentsCount++
	ra.metrics.LastAssessmentTime = time.Now()

	assessment := &RiskAssessment{
		OptimizationType:      optimization,
		ComponentType:         component,
		MaxAcceptableDowntime: 5 * time.Minute, // Default acceptable downtime
		EstimatedDowntime:     1 * time.Minute, // Default estimated downtime
		TechnicalComplexity:   5,               // Medium complexity
		ReversibilityScore:    0.7,             // Reasonably reversible
	}

	// Identify risk factors.
	assessment.RiskFactors = ra.identifyRiskFactors(component, optimization)

	// Calculate overall risk score.
	assessment.OverallRiskScore = ra.calculateOverallRisk(assessment.RiskFactors)

	// Determine risk level.
	assessment.RiskLevel = ra.determineRiskLevel(assessment.OverallRiskScore)

	// Generate mitigation strategies.
	assessment.MitigationStrategies = ra.generateMitigationStrategies(assessment.RiskFactors)

	// Generate recommended actions.
	assessment.RecommendedActions = ra.generateRecommendedActions(assessment)

	// Determine business impact.
	assessment.BusinessImpact = ra.determineBusinnessImpact(component, assessment.OverallRiskScore)

	// Update metrics.
	ra.updateRiskMetrics(assessment.RiskLevel)

	return assessment, nil
}

// identifyRiskFactors identifies potential risk factors for an optimization.
func (ra *RiskAssessor) identifyRiskFactors(component shared.ComponentType, optimization string) []RiskFactor {
	factors := []RiskFactor{
		{
			Name:        "System Downtime",
			Description: "Potential for system downtime during optimization implementation",
			Probability: 0.3,
			Impact:      0.8,
			Category:    "Operational",
		},
		{
			Name:        "Performance Regression",
			Description: "Risk of performance degradation instead of improvement",
			Probability: 0.2,
			Impact:      0.6,
			Category:    "Technical",
		},
		{
			Name:        "Configuration Errors",
			Description: "Risk of configuration mistakes during implementation",
			Probability: 0.4,
			Impact:      0.5,
			Category:    "Technical",
		},
	}

	// Add component-specific risks.
	switch component {
	case shared.ComponentTypeLLMProcessor:
		factors = append(factors, RiskFactor{
			Name:        "API Rate Limiting",
			Description: "Risk of hitting LLM API rate limits",
			Probability: 0.6,
			Impact:      0.7,
			Category:    "Technical",
		})
	case shared.ComponentTypeRAGSystem:
		factors = append(factors, RiskFactor{
			Name:        "Data Consistency",
			Description: "Risk of data inconsistency in vector database",
			Probability: 0.3,
			Impact:      0.8,
			Category:    "Technical",
		})
	case shared.ComponentTypeCache:
		factors = append(factors, RiskFactor{
			Name:        "Cache Invalidation",
			Description: "Risk of improper cache invalidation leading to stale data",
			Probability: 0.4,
			Impact:      0.6,
			Category:    "Technical",
		})
	}

	// Calculate risk scores.
	for i := range factors {
		factors[i].RiskScore = factors[i].Probability * factors[i].Impact
	}

	return factors
}

// calculateOverallRisk calculates the overall risk score from individual factors.
func (ra *RiskAssessor) calculateOverallRisk(factors []RiskFactor) float64 {
	if len(factors) == 0 {
		return 0.0
	}

	totalRisk := 0.0
	for _, factor := range factors {
		totalRisk += factor.RiskScore
	}

	// Normalize to 0-1 range.
	return math.Min(totalRisk/float64(len(factors)), 1.0)
}

// determineRiskLevel determines the risk level from the overall risk score.
func (ra *RiskAssessor) determineRiskLevel(score float64) string {
	switch {
	case score >= 0.8:
		return "Critical"
	case score >= 0.6:
		return "High"
	case score >= 0.4:
		return "Medium"
	case score >= 0.2:
		return "Low"
	default:
		return "Minimal"
	}
}

// generateMitigationStrategies generates mitigation strategies for identified risks.
func (ra *RiskAssessor) generateMitigationStrategies(factors []RiskFactor) []MitigationStrategy {
	strategies := []MitigationStrategy{}

	for _, factor := range factors {
		var strategy MitigationStrategy
		switch factor.Name {
		case "System Downtime":
			strategy = MitigationStrategy{
				RiskFactor:         factor.Name,
				Strategy:           "Implement blue-green deployment with rollback capability",
				Effectiveness:      0.8,
				ImplementationTime: 2 * time.Hour,
				Cost:               500.0,
				Priority:           "High",
			}
		case "Performance Regression":
			strategy = MitigationStrategy{
				RiskFactor:         factor.Name,
				Strategy:           "Implement comprehensive performance testing and monitoring",
				Effectiveness:      0.7,
				ImplementationTime: 4 * time.Hour,
				Cost:               800.0,
				Priority:           "High",
			}
		case "Configuration Errors":
			strategy = MitigationStrategy{
				RiskFactor:         factor.Name,
				Strategy:           "Use infrastructure as code with peer review",
				Effectiveness:      0.75,
				ImplementationTime: 1 * time.Hour,
				Cost:               200.0,
				Priority:           "Medium",
			}
		default:
			strategy = MitigationStrategy{
				RiskFactor:         factor.Name,
				Strategy:           "Implement standard monitoring and alerting",
				Effectiveness:      0.6,
				ImplementationTime: 30 * time.Minute,
				Cost:               100.0,
				Priority:           "Medium",
			}
		}
		strategies = append(strategies, strategy)
	}

	return strategies
}

// generateRecommendedActions generates recommended actions based on risk assessment.
func (ra *RiskAssessor) generateRecommendedActions(assessment *RiskAssessment) []string {
	actions := []string{
		"Create comprehensive backup and rollback plan",
		"Set up enhanced monitoring during optimization deployment",
		"Schedule optimization during low-traffic periods",
	}

	switch assessment.RiskLevel {
	case "Critical":
		actions = append(actions,
			"Require executive approval before proceeding",
			"Implement phased rollout with canary testing",
			"Have dedicated incident response team on standby",
		)
	case "High":
		actions = append(actions,
			"Require senior engineer approval",
			"Implement gradual rollout with monitoring checkpoints",
			"Prepare detailed rollback procedures",
		)
	case "Medium":
		actions = append(actions,
			"Implement standard change management process",
			"Test in staging environment first",
		)
	case "Low", "Minimal":
		actions = append(actions,
			"Follow standard deployment procedures",
			"Monitor key metrics for 24 hours post-deployment",
		)
	}

	return actions
}

// determineBusinnessImpact determines business impact based on component and risk.
func (ra *RiskAssessor) determineBusinnessImpact(component shared.ComponentType, riskScore float64) string {
	baseImpact := "Medium"

	// Component-specific impact.
	switch component {
	case shared.ComponentTypeLLMProcessor:
		baseImpact = "High" // Core functionality
	case shared.ComponentTypeRAGSystem:
		baseImpact = "High" // Core functionality
	case shared.ComponentTypeCache:
		baseImpact = "Medium" // Performance impact
	case shared.ComponentTypeNetworking:
		baseImpact = "High" // Connectivity critical
	case shared.ComponentTypeDatabase:
		baseImpact = "Medium" // Data availability
	case shared.ComponentTypeKubernetes:
		baseImpact = "Critical" // Infrastructure critical
	}

	// Adjust based on risk score.
	if riskScore >= 0.8 {
		switch baseImpact {
		case "Medium":
			return "High"
		case "High":
			return "Critical"
		}
	} else if riskScore <= 0.3 {
		switch baseImpact {
		case "High":
			return "Medium"
		case "Critical":
			return "High"
		}
	}

	return baseImpact
}

// updateRiskMetrics updates risk assessment metrics.
func (ra *RiskAssessor) updateRiskMetrics(riskLevel string) {
	switch riskLevel {
	case "Critical", "High":
		ra.metrics.HighRiskCount++
	case "Medium":
		ra.metrics.MediumRiskCount++
	case "Low", "Minimal":
		ra.metrics.LowRiskCount++
	}

	// Update average risk score (simplified).
	total := ra.metrics.HighRiskCount + ra.metrics.MediumRiskCount + ra.metrics.LowRiskCount
	if total > 0 {
		weightedSum := float64(ra.metrics.HighRiskCount)*0.8 + float64(ra.metrics.MediumRiskCount)*0.5 + float64(ra.metrics.LowRiskCount)*0.2
		ra.metrics.AverageRiskScore = float64(weightedSum) / float64(total)
	}
}

// Additional missing types referenced in performance_analysis_engine.go.

// MetricsStore manages storage and retrieval of performance metrics.
type MetricsStore struct {
	data       map[string][]float64
	timestamps map[string][]time.Time
	mu         sync.RWMutex
}

// HistoricalDataStore manages historical performance data for trend analysis.
type HistoricalDataStore struct {
	dataPoints map[string][]DataPoint
	retention  time.Duration
	mu         sync.RWMutex
}

// Note: DataPoint is defined in optimization_dashboard.go.

// PatternDetector detects performance patterns and anomalies.
type PatternDetector struct {
	patterns    []Pattern
	sensitivity float64
	windowSize  time.Duration
	mu          sync.RWMutex
}

// Pattern represents a detected performance pattern.
type Pattern struct {
	Type        string    `json:"type"`       // trend, spike, anomaly, cyclic
	Confidence  float64   `json:"confidence"` // 0-1
	StartTime   time.Time `json:"startTime"`
	EndTime     time.Time `json:"endTime"`
	Description string    `json:"description"`
	Metrics     []string  `json:"metrics"`
}

// BottleneckPredictor predicts potential system bottlenecks.
type BottleneckPredictor struct {
	models            map[string]*PredictionModel
	thresholds        map[string]float64
	predictionHorizon time.Duration
	mu                sync.RWMutex
}

// PerformanceForecaster provides performance forecasting capabilities.
type PerformanceForecaster struct {
	forecastModels map[string]*ForecastModel
	horizon        time.Duration
	accuracy       float64
	mu             sync.RWMutex
}

// ForecastModel represents a forecasting model.
type ForecastModel struct {
	Type        string             `json:"type"` // linear, arima, neural
	Parameters  map[string]float64 `json:"parameters"`
	Accuracy    float64            `json:"accuracy"`
	LastTrained time.Time          `json:"lastTrained"`
}

// OptimizationRanker ranks optimization opportunities by potential impact.
type OptimizationRanker struct {
	rankingCriteria []RankingCriterion
	weights         map[string]float64
	mu              sync.RWMutex
}

// RankingCriterion defines criteria for ranking optimizations.
type RankingCriterion struct {
	Name        string  `json:"name"`
	Weight      float64 `json:"weight"`
	Description string  `json:"description"`
}

// NewMetricsStore creates a new metrics store.
func NewMetricsStore() *MetricsStore {
	return &MetricsStore{
		data:       make(map[string][]float64),
		timestamps: make(map[string][]time.Time),
	}
}

// NewHistoricalDataStore creates a new historical data store.
func NewHistoricalDataStore(retention time.Duration) *HistoricalDataStore {
	return &HistoricalDataStore{
		dataPoints: make(map[string][]DataPoint),
		retention:  retention,
	}
}

// NewPatternDetector creates a new pattern detector.
func NewPatternDetector(sensitivity float64, windowSize time.Duration) *PatternDetector {
	return &PatternDetector{
		patterns:    make([]Pattern, 0),
		sensitivity: sensitivity,
		windowSize:  windowSize,
	}
}

// NewBottleneckPredictor creates a new bottleneck predictor.
func NewBottleneckPredictor(predictionHorizon time.Duration) *BottleneckPredictor {
	return &BottleneckPredictor{
		models:            make(map[string]*PredictionModel),
		thresholds:        make(map[string]float64),
		predictionHorizon: predictionHorizon,
	}
}

// NewPerformanceForecaster creates a new performance forecaster.
func NewPerformanceForecaster(horizon time.Duration) *PerformanceForecaster {
	return &PerformanceForecaster{
		forecastModels: make(map[string]*ForecastModel),
		horizon:        horizon,
		accuracy:       0.85, // Default accuracy target
	}
}

// NewOptimizationRanker creates a new optimization ranker.
func NewOptimizationRanker() *OptimizationRanker {
	return &OptimizationRanker{
		rankingCriteria: make([]RankingCriterion, 0),
		weights:         make(map[string]float64),
	}
}
