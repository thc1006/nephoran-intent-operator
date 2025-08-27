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

	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// ComponentType is an alias for shared.ComponentType for backward compatibility
// and cleaner imports within the optimization package
type ComponentType = shared.ComponentType

// OptimizationCategory represents different categories of optimizations
type OptimizationCategory string

const (
	CategoryPerformance        OptimizationCategory = "performance"
	CategoryResource           OptimizationCategory = "resource"
	CategoryCost               OptimizationCategory = "cost"
	CategoryReliability        OptimizationCategory = "reliability"
	CategorySecurity           OptimizationCategory = "security"
	CategoryCompliance         OptimizationCategory = "compliance"
	CategoryMaintenance        OptimizationCategory = "maintenance"
	CategoryTelecommunications OptimizationCategory = "telecommunications"
)

// OptimizationPriority represents priority levels for optimizations
type OptimizationPriority string

const (
	PriorityCritical OptimizationPriority = "critical"
	PriorityHigh     OptimizationPriority = "high"
	PriorityMedium   OptimizationPriority = "medium"
	PriorityLow      OptimizationPriority = "low"
)

// SeverityLevel represents severity levels
type SeverityLevel string

const (
	SeverityCritical SeverityLevel = "critical"
	SeverityHigh     SeverityLevel = "high"
	SeverityMedium   SeverityLevel = "medium"
	SeverityLow      SeverityLevel = "low"
	SeverityInfo     SeverityLevel = "info"
)

// AutomationLevel defines levels of automation for implementation steps
type AutomationLevel string

const (
	AutomationFull     AutomationLevel = "full"
	AutomationPartial  AutomationLevel = "partial"
	AutomationManual   AutomationLevel = "manual"
	AutomationAssisted AutomationLevel = "assisted"
)

// ExpectedBenefits represents expected benefits from optimization
type ExpectedBenefits struct {
	LatencyReduction       float64 `json:"latencyReduction"`
	ThroughputIncrease     float64 `json:"throughputIncrease"`
	ResourceSavings        float64 `json:"resourceSavings"`
	CostSavings            float64 `json:"costSavings"`
	EfficiencyGain         float64 `json:"efficiencyGain,omitempty"`
	ErrorRateReduction     float64 `json:"errorRateReduction,omitempty"`
	ReliabilityImprovement float64 `json:"reliabilityImprovement"`
	EnergyEfficiencyGain   float64 `json:"energyEfficiencyGain"`

	// Telecom-specific benefits
	SignalingEfficiencyGain float64 `json:"signalingEfficiencyGain"`
	SpectrumEfficiencyGain  float64 `json:"spectrumEfficiencyGain"`
	InteropImprovements     float64 `json:"interopImprovements"`
}

// ImplementationStep represents a single implementation step
type ImplementationStep struct {
	Order           int             `json:"order"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	EstimatedTime   time.Duration   `json:"estimatedTime"`
	AutomationLevel AutomationLevel `json:"automationLevel"`
	RequiredSkills  []string        `json:"requiredSkills,omitempty"`
	ValidationPoint bool            `json:"validationPoint,omitempty"`
	RollbackAction  string          `json:"rollbackAction,omitempty"`
}

// ConfidenceInterval represents a confidence interval for statistical data
type ConfidenceInterval struct {
	Timestamp       time.Time `json:"timestamp,omitempty"`
	LowerBound      float64   `json:"lowerBound"`
	UpperBound      float64   `json:"upperBound"`
	ConfidenceLevel float64   `json:"confidenceLevel"`
}

// ExpectedImpact represents the expected impact of optimization changes
type ExpectedImpact struct {
	LatencyReduction   float64 `json:"latencyReduction"`
	ThroughputIncrease float64 `json:"throughputIncrease"`
	ResourceSavings    float64 `json:"resourceSavings"`
	CostSavings        float64 `json:"costSavings"`
	EfficiencyGain     float64 `json:"efficiencyGain"`
}

// ComparisonOperator defines comparison operators for conditions
type ComparisonOperator string

const (
	OperatorGreaterThan  ComparisonOperator = "gt"
	OperatorLessThan     ComparisonOperator = "lt"
	OperatorEqual        ComparisonOperator = "eq"
	OperatorGreaterEqual ComparisonOperator = "gte"
	OperatorLessEqual    ComparisonOperator = "lte"
	OperatorBetween      ComparisonOperator = "between"
)

// ScenarioCondition defines when a strategy is applicable
type ScenarioCondition struct {
	MetricName    string             `json:"metricName"`
	Operator      ComparisonOperator `json:"operator"`
	Threshold     float64            `json:"threshold"`
	ComponentType ComponentType      `json:"componentType"`
	TimeWindow    time.Duration      `json:"timeWindow"`
}

// OptimizationKnowledgeBase contains optimization knowledge and best practices
type OptimizationKnowledgeBase struct {
	// Best practices indexed by component type
	BestPractices map[shared.ComponentType][]*OptimizationBestPractice `json:"bestPractices"`

	// Historical optimization results
	OptimizationHistory []*OptimizationHistoryEntry `json:"optimizationHistory"`

	// Common bottlenecks and their solutions
	KnownBottlenecks map[string]*BottleneckSolution `json:"knownBottlenecks"`

	// Telecom-specific optimization rules
	TelecomRules []*TelecomOptimizationRule `json:"telecomRules"`

	// ML model metadata
	ModelMetadata map[string]*MLModelMetadata `json:"modelMetadata"`

	// Compliance requirements
	ComplianceRequirements []*ComplianceRequirement `json:"complianceRequirements"`

	// Last updated timestamp
	LastUpdated time.Time `json:"lastUpdated"`
}

// OptimizationBestPractice represents an optimization best practice
type OptimizationBestPractice struct {
	ID              string               `json:"id"`
	Name            string               `json:"name"`
	Description     string               `json:"description"`
	ComponentType   shared.ComponentType `json:"componentType"`
	Category        OptimizationCategory `json:"category"`
	Priority        int                  `json:"priority"`
	ApplicableWhen  []string             `json:"applicableWhen"`
	Implementation  string               `json:"implementation"`
	ExpectedBenefit float64              `json:"expectedBenefit"`
	RiskLevel       string               `json:"riskLevel"`
	ValidationSteps []string             `json:"validationSteps"`
}

// OptimizationHistoryEntry tracks past optimization results
type OptimizationHistoryEntry struct {
	Timestamp        time.Time            `json:"timestamp"`
	ComponentType    shared.ComponentType `json:"componentType"`
	OptimizationType string               `json:"optimizationType"`
	BeforeMetrics    map[string]float64   `json:"beforeMetrics"`
	AfterMetrics     map[string]float64   `json:"afterMetrics"`
	ImprovementPct   float64              `json:"improvementPct"`
	Success          bool                 `json:"success"`
	FailureReason    string               `json:"failureReason,omitempty"`
	Duration         time.Duration        `json:"duration"`
}

// BottleneckSolution contains solutions for known bottlenecks
type BottleneckSolution struct {
	BottleneckType     string                  `json:"bottleneckType"`
	Description        string                  `json:"description"`
	ComponentTypes     []shared.ComponentType  `json:"componentTypes"`
	Solutions          []*OptimizationSolution `json:"solutions"`
	PreventiveMeasures []string                `json:"preventiveMeasures"`
	DetectionMethod    string                  `json:"detectionMethod"`
}

// OptimizationSolution represents a specific solution to a bottleneck
type OptimizationSolution struct {
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	Steps         []string      `json:"steps"`
	EstimatedTime time.Duration `json:"estimatedTime"`
	Complexity    string        `json:"complexity"` // "low", "medium", "high"
	SuccessRate   float64       `json:"successRate"`
	Prerequisites []string      `json:"prerequisites"`
	RollbackSteps []string      `json:"rollbackSteps"`
}

// TelecomOptimizationRule represents telecom-specific optimization rules
type TelecomOptimizationRule struct {
	ID              string                      `json:"id"`
	Name            string                      `json:"name"`
	Description     string                      `json:"description"`
	ComponentTypes  []shared.ComponentType      `json:"componentTypes"`
	Conditions      []OptimizationRuleCondition `json:"conditions"`
	Actions         []OptimizationRuleAction    `json:"actions"`
	Priority        int                         `json:"priority"`
	SLAImpact       string                      `json:"slaImpact"`
	ComplianceLevel string                      `json:"complianceLevel"`
	Enabled         bool                        `json:"enabled"`
}

// OptimizationRuleCondition defines a condition in optimization rules
type OptimizationRuleCondition struct {
	MetricName     string        `json:"metricName"`
	Operator       string        `json:"operator"`
	ThresholdValue float64       `json:"thresholdValue"`
	TimeWindow     time.Duration `json:"timeWindow"`
	Severity       string        `json:"severity"`
}

// OptimizationRuleAction defines an action to take when rule conditions are met
type OptimizationRuleAction struct {
	ActionType        string            `json:"actionType"`
	Parameters        map[string]string `json:"parameters"`
	AutoExecute       bool              `json:"autoExecute"`
	RequireApproval   bool              `json:"requireApproval"`
	NotificationLevel string            `json:"notificationLevel"`
}

// MLModelMetadata contains metadata about ML models used in optimization
type MLModelMetadata struct {
	ModelID        string                 `json:"modelId"`
	Name           string                 `json:"name"`
	Type           string                 `json:"type"`
	Version        string                 `json:"version"`
	ComponentTypes []shared.ComponentType `json:"componentTypes"`
	Accuracy       float64                `json:"accuracy"`
	TrainedOn      time.Time              `json:"trainedOn"`
	LastValidated  time.Time              `json:"lastValidated"`
	Parameters     map[string]interface{} `json:"parameters"`
	InputFeatures  []string               `json:"inputFeatures"`
	OutputFormat   string                 `json:"outputFormat"`
}

// ComplianceRequirement represents compliance requirements for optimizations
type ComplianceRequirement struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Standard        string                 `json:"standard"` // e.g., "O-RAN", "3GPP", "ITU"
	ComponentTypes  []shared.ComponentType `json:"componentTypes"`
	Requirements    []string               `json:"requirements"`
	ValidationRules []string               `json:"validationRules"`
	Mandatory       bool                   `json:"mandatory"`
	Deadline        time.Time              `json:"deadline,omitempty"`
}

// PerformanceBottleneck represents a detected performance bottleneck
type PerformanceBottleneck struct {
	ID                 string                 `json:"id"`
	Name               string                 `json:"name"`
	Description        string                 `json:"description"`
	ComponentType      shared.ComponentType   `json:"componentType"`
	Severity           SeverityLevel          `json:"severity"` // Using existing SeverityLevel type
	AffectedMetrics    []string               `json:"affectedMetrics"`
	RootCause          string                 `json:"rootCause"`
	ImpactScore        float64                `json:"impactScore"`
	DetectedAt         time.Time              `json:"detectedAt"`
	Duration           time.Duration          `json:"duration"`
	AffectedComponents []shared.ComponentType `json:"affectedComponents"`
	RecommendedActions []string               `json:"recommendedActions"`
	PredictedGrowth    float64                `json:"predictedGrowth"`
	BusinessImpact     string                 `json:"businessImpact"`
}

// PerformanceIssue represents a detected performance issue
type PerformanceIssue struct {
	ID                string               `json:"id"`
	Name              string               `json:"name"`
	Description       string               `json:"description"`
	ComponentType     shared.ComponentType `json:"componentType"`
	Category          IssueCategory        `json:"category"`
	Severity          SeverityLevel        `json:"severity"` // Using existing SeverityLevel type
	MetricName        string               `json:"metricName"`
	CurrentValue      float64              `json:"currentValue"`
	ExpectedValue     float64              `json:"expectedValue"`
	Deviation         float64              `json:"deviation"`
	FirstDetected     time.Time            `json:"firstDetected"`
	LastSeen          time.Time            `json:"lastSeen"`
	Frequency         int                  `json:"frequency"`
	TrendDirection    string               `json:"trendDirection"`
	RootCauseAnalysis *RootCauseAnalysis   `json:"rootCauseAnalysis,omitempty"`
	Resolution        *IssueResolution     `json:"resolution,omitempty"`
}

// IssueCategory represents the category of a performance issue
type IssueCategory string

const (
	IssueCategoryLatency      IssueCategory = "latency"
	IssueCategoryThroughput   IssueCategory = "throughput"
	IssueCategoryMemory       IssueCategory = "memory"
	IssueCategoryCPU          IssueCategory = "cpu"
	IssueCategoryNetwork      IssueCategory = "network"
	IssueCategoryStorage      IssueCategory = "storage"
	IssueCategoryAvailability IssueCategory = "availability"
)

// RootCauseAnalysis contains root cause analysis information
type RootCauseAnalysis struct {
	PossibleCauses     []string  `json:"possibleCauses"`
	MostLikelyCause    string    `json:"mostLikelyCause"`
	ConfidenceScore    float64   `json:"confidenceScore"`
	AnalysisMethod     string    `json:"analysisMethod"`
	SupportingEvidence []string  `json:"supportingEvidence"`
	CorrelatedMetrics  []string  `json:"correlatedMetrics"`
	AnalysisTimestamp  time.Time `json:"analysisTimestamp"`
}

// IssueResolution contains information about issue resolution
type IssueResolution struct {
	ResolutionID       string           `json:"resolutionId"`
	Status             ResolutionStatus `json:"status"`
	Actions            []string         `json:"actions"`
	ResolvedAt         time.Time        `json:"resolvedAt,omitempty"`
	ResolvedBy         string           `json:"resolvedBy"`
	VerificationSteps  []string         `json:"verificationSteps"`
	PreventionMeasures []string         `json:"preventionMeasures"`
}

// ResolutionStatus represents the status of issue resolution
type ResolutionStatus string

const (
	ResolutionStatusPending    ResolutionStatus = "pending"
	ResolutionStatusInProgress ResolutionStatus = "in_progress"
	ResolutionStatusResolved   ResolutionStatus = "resolved"
	ResolutionStatusFailed     ResolutionStatus = "failed"
)

// ResourceConstraint represents a resource constraint affecting performance
type ResourceConstraint struct {
	ID               string               `json:"id"`
	Name             string               `json:"name"`
	ComponentType    shared.ComponentType `json:"componentType"`
	ResourceType     ResourceType         `json:"resourceType"`   // Using existing ResourceType
	ConstraintType   string               `json:"constraintType"` // Using string to avoid conflicts
	CurrentUsage     float64              `json:"currentUsage"`
	MaxCapacity      float64              `json:"maxCapacity"`
	UtilizationPct   float64              `json:"utilizationPct"`
	Threshold        float64              `json:"threshold"`
	ProjectedGrowth  float64              `json:"projectedGrowth"`
	TimeToExhaustion time.Duration        `json:"timeToExhaustion,omitempty"`
	Impact           ConstraintImpact     `json:"impact"`
	Recommendations  []string             `json:"recommendations"`
	DetectedAt       time.Time            `json:"detectedAt"`
	BusinessContext  string               `json:"businessContext"`
}

// Additional constraint type constants for ResourceConstraint
const (
	ConstraintTypeCapacity    = "capacity"
	ConstraintTypePerformance = "performance"
	ConstraintTypeCost        = "cost"
	ConstraintTypeCompliance  = "compliance"
	ConstraintTypeSLA         = "sla"
)

// ConstraintImpact represents the impact of a resource constraint
type ConstraintImpact struct {
	PerformanceDegradation float64  `json:"performanceDegradation"`
	AffectedServices       []string `json:"affectedServices"`
	BusinessImpact         string   `json:"businessImpact"`
	SLAViolationRisk       float64  `json:"slaViolationRisk"`
	CostImplication        float64  `json:"costImplication"`
}

// OptimizationOpportunity represents an identified optimization opportunity
type OptimizationOpportunity struct {
	ID                   string                     `json:"id"`
	Name                 string                     `json:"name"`
	Description          string                     `json:"description"`
	ComponentType        shared.ComponentType       `json:"componentType"`
	Category             OptimizationCategory       `json:"category"`
	Priority             OpportunityPriority        `json:"priority"`
	EstimatedImprovement *EstimatedImprovement      `json:"estimatedImprovement"`
	RiskAssessment       *OpportunityRiskAssessment `json:"riskAssessment"`
	ROIAnalysis          *ROIAnalysis               `json:"roiAnalysis"`
	Dependencies         []string                   `json:"dependencies"`
	Prerequisites        []string                   `json:"prerequisites"`
	ValidationCriteria   []string                   `json:"validationCriteria"`
	DetectedAt           time.Time                  `json:"detectedAt"`
	EstimatedCompletion  time.Duration              `json:"estimatedCompletion"`
}

// OpportunityPriority represents the priority of an optimization opportunity
type OpportunityPriority string

const (
	OpportunityPriorityLow      OpportunityPriority = "low"
	OpportunityPriorityMedium   OpportunityPriority = "medium"
	OpportunityPriorityHigh     OpportunityPriority = "high"
	OpportunityPriorityCritical OpportunityPriority = "critical"
)

// EstimatedImprovement represents expected improvements from an optimization
type EstimatedImprovement struct {
	PerformanceGain    float64            `json:"performanceGain"`
	CostSaving         float64            `json:"costSaving"`
	ResourceReduction  float64            `json:"resourceReduction"`
	LatencyImprovement time.Duration      `json:"latencyImprovement"`
	ThroughputIncrease float64            `json:"throughputIncrease"`
	EfficiencyGain     float64            `json:"efficiencyGain"`
	MetricImpacts      map[string]float64 `json:"metricImpacts"`
	SLACompliance      float64            `json:"slaCompliance"`
}

// OpportunityRiskAssessment represents risk assessment for an optimization opportunity
type OpportunityRiskAssessment struct {
	OverallRisk          RiskLevel               `json:"overallRisk"` // Using existing RiskLevel type
	RiskFactors          []OpportunityRiskFactor `json:"riskFactors"`
	MitigationStrategies []string                `json:"mitigationStrategies"`
	ImpactAnalysis       *RiskImpactAnalysis     `json:"impactAnalysis"`
	ApprovalRequired     bool                    `json:"approvalRequired"`
	BackupPlan           string                  `json:"backupPlan"`
}

// OpportunityRiskFactor represents a specific risk factor for optimization opportunities
type OpportunityRiskFactor struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Probability float64 `json:"probability"`
	Impact      float64 `json:"impact"`
	Category    string  `json:"category"`
	Mitigation  string  `json:"mitigation"`
	Contingency string  `json:"contingency"`
}

// RiskImpactAnalysis contains analysis of potential risks
type RiskImpactAnalysis struct {
	PerformanceRisk  float64       `json:"performanceRisk"`
	AvailabilityRisk float64       `json:"availabilityRisk"`
	SecurityRisk     float64       `json:"securityRisk"`
	ComplianceRisk   float64       `json:"complianceRisk"`
	BusinessImpact   string        `json:"businessImpact"`
	RecoveryTime     time.Duration `json:"recoveryTime"`
	AffectedSystems  []string      `json:"affectedSystems"`
}

// ROIAnalysis contains return on investment analysis
type ROIAnalysis struct {
	InitialInvestment   float64            `json:"initialInvestment"`
	ExpectedSavings     float64            `json:"expectedSavings"`
	ROIPercentage       float64            `json:"roiPercentage"`
	PaybackPeriod       time.Duration      `json:"paybackPeriod"`
	NetPresentValue     float64            `json:"netPresentValue"`
	SavingsBreakdown    map[string]float64 `json:"savingsBreakdown"`
	CostBreakdown       map[string]float64 `json:"costBreakdown"`
	AssumptionsUsed     []string           `json:"assumptionsUsed"`
	SensitivityAnalysis map[string]float64 `json:"sensitivityAnalysis"`
}

// ResourceUtilization represents resource utilization metrics
type ResourceUtilization struct {
	ComponentType         shared.ComponentType       `json:"componentType"`
	TimePeriod            TimePeriod                 `json:"timePeriod"`
	CPUUtilization        *ResourceMetric            `json:"cpuUtilization"`
	MemoryUtilization     *ResourceMetric            `json:"memoryUtilization"`
	StorageUtilization    *ResourceMetric            `json:"storageUtilization"`
	NetworkUtilization    *ResourceMetric            `json:"networkUtilization"`
	CustomMetrics         map[string]*ResourceMetric `json:"customMetrics"`
	EfficiencyScore       float64                    `json:"efficiencyScore"`
	WasteFactors          []*WasteFactor             `json:"wasteFactors"`
	OptimizationPotential float64                    `json:"optimizationPotential"`
	BenchmarkComparison   *BenchmarkComparison       `json:"benchmarkComparison"`
	TrendAnalysis         *UtilizationTrend          `json:"trendAnalysis"`
}

// TimePeriod represents a time period for metrics
type TimePeriod struct {
	Start       time.Time     `json:"start"`
	End         time.Time     `json:"end"`
	Duration    time.Duration `json:"duration"`
	Granularity string        `json:"granularity"`
}

// ResourceMetric represents metrics for a specific resource type
type ResourceMetric struct {
	Current     float64               `json:"current"`
	Average     float64               `json:"average"`
	Peak        float64               `json:"peak"`
	Minimum     float64               `json:"minimum"`
	Percentiles map[string]float64    `json:"percentiles"`
	Unit        string                `json:"unit"`
	Threshold   *ResourceThreshold    `json:"threshold"`
	Anomalies   []*UtilizationAnomaly `json:"anomalies"`
}

// ResourceThreshold represents thresholds for resource metrics
type ResourceThreshold struct {
	Warning  float64 `json:"warning"`
	Critical float64 `json:"critical"`
	Target   float64 `json:"target"`
	Optimal  float64 `json:"optimal"`
}

// WasteFactor represents factors contributing to resource waste
type WasteFactor struct {
	Factor      string  `json:"factor"`
	Impact      float64 `json:"impact"`
	Description string  `json:"description"`
	Remedy      string  `json:"remedy"`
}

// BenchmarkComparison compares utilization against benchmarks
type BenchmarkComparison struct {
	Industry     float64 `json:"industry"`
	Internal     float64 `json:"internal"`
	BestPractice float64 `json:"bestPractice"`
	Deviation    float64 `json:"deviation"`
	Ranking      string  `json:"ranking"`
}

// UtilizationTrend represents trends in resource utilization
type UtilizationTrend struct {
	Direction       string        `json:"direction"`
	Rate            float64       `json:"rate"`
	Seasonality     bool          `json:"seasonality"`
	GrowthPattern   string        `json:"growthPattern"`
	ForecastPeriod  time.Duration `json:"forecastPeriod"`
	PredictedValues []float64     `json:"predictedValues"`
}

// UtilizationAnomaly represents an anomaly in resource utilization
type UtilizationAnomaly struct {
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Value       float64   `json:"value"`
	Expected    float64   `json:"expected"`
	Deviation   float64   `json:"deviation"`
	Description string    `json:"description"`
}

// ComponentPerformanceMetrics represents performance metrics for a component
type ComponentPerformanceMetrics struct {
	ComponentType     shared.ComponentType          `json:"componentType"`
	MeasurementPeriod TimePeriod                    `json:"measurementPeriod"`
	ResponseTime      *PerformanceMetric            `json:"responseTime"`
	Throughput        *PerformanceMetric            `json:"throughput"`
	ErrorRate         *PerformanceMetric            `json:"errorRate"`
	Availability      *AvailabilityMetric           `json:"availability"`
	Reliability       *ReliabilityMetric            `json:"reliability"`
	Scalability       *ScalabilityMetric            `json:"scalability"`
	Efficiency        *EfficiencyMetric             `json:"efficiency"`
	CustomMetrics     map[string]*PerformanceMetric `json:"customMetrics"`
	SLACompliance     *SLAComplianceMetric          `json:"slaCompliance"`
	QualityScore      float64                       `json:"qualityScore"`
	PerformanceTrends []*MetricTrend                `json:"performanceTrends"`
	BenchmarkResults  *PerformanceBenchmark         `json:"benchmarkResults"`
}

// PerformanceMetric represents a general performance metric
type PerformanceMetric struct {
	Value         float64           `json:"value"`
	Unit          string            `json:"unit"`
	Target        float64           `json:"target"`
	Baseline      float64           `json:"baseline"`
	Improvement   float64           `json:"improvement"`
	Trend         string            `json:"trend"`
	Statistics    *MetricStatistics `json:"statistics"` // Using existing type
	SLABoundaries *SLABoundaries    `json:"slaBoundaries"`
	QualityGates  []*QualityGate    `json:"qualityGates"`
}

// AvailabilityMetric represents availability metrics
type AvailabilityMetric struct {
	*PerformanceMetric
	Uptime        time.Duration `json:"uptime"`
	Downtime      time.Duration `json:"downtime"`
	MTBF          time.Duration `json:"mtbf"` // Mean Time Between Failures
	MTTR          time.Duration `json:"mttr"` // Mean Time To Recovery
	IncidentCount int           `json:"incidentCount"`
}

// ReliabilityMetric represents reliability metrics
type ReliabilityMetric struct {
	*PerformanceMetric
	SuccessRate      float64       `json:"successRate"`
	FailureRate      float64       `json:"failureRate"`
	RetryRate        float64       `json:"retryRate"`
	RecoveryTime     time.Duration `json:"recoveryTime"`
	DataIntegrity    float64       `json:"dataIntegrity"`
	ConsistencyScore float64       `json:"consistencyScore"`
}

// ScalabilityMetric represents scalability metrics
type ScalabilityMetric struct {
	*PerformanceMetric
	MaxCapacity        float64       `json:"maxCapacity"`
	CurrentLoad        float64       `json:"currentLoad"`
	LoadFactor         float64       `json:"loadFactor"`
	ElasticityScore    float64       `json:"elasticityScore"`
	ScalingLatency     time.Duration `json:"scalingLatency"`
	ResourceEfficiency float64       `json:"resourceEfficiency"`
}

// EfficiencyMetric represents efficiency metrics
type EfficiencyMetric struct {
	*PerformanceMetric
	ResourceEfficiency float64 `json:"resourceEfficiency"`
	CostEfficiency     float64 `json:"costEfficiency"`
	EnergyEfficiency   float64 `json:"energyEfficiency"`
	TimeEfficiency     float64 `json:"timeEfficiency"`
	WasteReduction     float64 `json:"wasteReduction"`
	OptimizationGain   float64 `json:"optimizationGain"`
}

// SLAComplianceMetric represents SLA compliance metrics
type SLAComplianceMetric struct {
	OverallCompliance  float64                   `json:"overallCompliance"`
	SLAViolations      int                       `json:"slaViolations"`
	ComplianceByMetric map[string]float64        `json:"complianceByMetric"`
	PenaltyCost        float64                   `json:"penaltyCost"`
	RiskScore          float64                   `json:"riskScore"`
	ComplianceHistory  []*ComplianceHistoryEntry `json:"complianceHistory"`
}

// SLABoundaries represents SLA boundaries for a metric
type SLABoundaries struct {
	Target    float64 `json:"target"`
	Threshold float64 `json:"threshold"`
	Critical  float64 `json:"critical"`
	Optimal   float64 `json:"optimal"`
}

// QualityGate represents a quality gate for performance metrics
type QualityGate struct {
	Name        string  `json:"name"`
	Threshold   float64 `json:"threshold"`
	Operator    string  `json:"operator"`
	Passed      bool    `json:"passed"`
	ActualValue float64 `json:"actualValue"`
	Impact      string  `json:"impact"`
}

// MetricTrend represents trends in performance metrics
type MetricTrend struct {
	MetricName       string        `json:"metricName"`
	TrendType        string        `json:"trendType"`
	Direction        string        `json:"direction"`
	ChangeRate       float64       `json:"changeRate"`
	Confidence       float64       `json:"confidence"`
	TimeHorizon      time.Duration `json:"timeHorizon"`
	Forecast         []float64     `json:"forecast"`
	InflectionPoints []time.Time   `json:"inflectionPoints"`
}

// ComplianceHistoryEntry represents historical compliance data
type ComplianceHistoryEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Compliance  float64   `json:"compliance"`
	Violations  int       `json:"violations"`
	MetricName  string    `json:"metricName"`
	ActionTaken string    `json:"actionTaken"`
}

// PerformanceTrend represents trends in system performance
type PerformanceTrend struct {
	ComponentType       shared.ComponentType `json:"componentType"`
	MetricName          string               `json:"metricName"`
	TrendType           TrendType            `json:"trendType"`
	Direction           TrendDirection       `json:"direction"`
	Slope               float64              `json:"slope"`
	Confidence          float64              `json:"confidence"`
	StartTime           time.Time            `json:"startTime"`
	EndTime             time.Time            `json:"endTime"`
	DataPoints          int                  `json:"dataPoints"`
	AverageValue        float64              `json:"averageValue"`
	VarianceValue       float64              `json:"varianceValue"`
	SeasonalityDetected bool                 `json:"seasonalityDetected"`
	SeasonalPeriod      time.Duration        `json:"seasonalPeriod,omitempty"`
	ChangePoints        []time.Time          `json:"changePoints"`
	Forecast            *ForecastData        `json:"forecast,omitempty"`
}

// TrendType represents the type of trend analysis
type TrendType string

const (
	TrendTypeLinear      TrendType = "linear"
	TrendTypeExponential TrendType = "exponential"
	TrendTypeSeasonal    TrendType = "seasonal"
	TrendTypeCyclical    TrendType = "cyclical"
	TrendTypeNoTrend     TrendType = "no_trend"
)

// Note: TrendDirection is already defined in performance_analysis_engine.go

// ForecastData represents forecast information for performance trends
type ForecastData struct {
	PredictedValues     []float64            `json:"predictedValues"`
	ConfidenceIntervals []ConfidenceInterval `json:"confidenceIntervals"` // Using existing type
	ForecastHorizon     time.Duration        `json:"forecastHorizon"`
	ModelAccuracy       float64              `json:"modelAccuracy"`
	ModelType           string               `json:"modelType"`
}

// Note: ConfidenceInterval is already defined in performance_analysis_engine.go

// PerformanceBenchmark represents benchmark results
type PerformanceBenchmark struct {
	BenchmarkSuite     string                `json:"benchmarkSuite"`
	Version            string                `json:"version"`
	ExecutionTime      time.Time             `json:"executionTime"`
	Duration           time.Duration         `json:"duration"`
	Results            map[string]float64    `json:"results"`
	Percentile         float64               `json:"percentile"`
	Ranking            int                   `json:"ranking"`
	BaselineComparison map[string]float64    `json:"baselineComparison"`
	Environment        *BenchmarkEnvironment `json:"environment"`
}

// BenchmarkEnvironment represents the environment where benchmarks were run
type BenchmarkEnvironment struct {
	Platform       string            `json:"platform"`
	Version        string            `json:"version"`
	Configuration  map[string]string `json:"configuration"`
	ResourceLimits map[string]string `json:"resourceLimits"`
	Load           string            `json:"load"`
	Concurrency    int               `json:"concurrency"`
}
