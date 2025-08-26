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

package dependencies

import (
	"time"
)

// Analysis result types that are referenced in analyzer.go but not defined

// PackageUsageMetrics contains usage metrics for a package
type PackageUsageMetrics struct {
	TotalUsage      int64     `json:"totalUsage"`
	DailyUsage      []int64   `json:"dailyUsage,omitempty"`
	PeakUsage       int64     `json:"peakUsage"`
	AverageUsage    float64   `json:"averageUsage"`
	UsageVariance   float64   `json:"usageVariance"`
	LastUsed        time.Time `json:"lastUsed"`
	UsageFrequency  string    `json:"usageFrequency"` // "high", "medium", "low", "rare"
	UsagePattern    string    `json:"usagePattern"`   // "steady", "bursty", "seasonal", "declining"
}

// QualityMetrics contains quality metrics for a package
type QualityMetrics struct {
	CodeQuality     float64   `json:"codeQuality"`
	TestCoverage    float64   `json:"testCoverage,omitempty"`
	Documentation   float64   `json:"documentation,omitempty"`
	Maintainability float64   `json:"maintainability,omitempty"`
	Reliability     float64   `json:"reliability,omitempty"`
	Security        float64   `json:"security,omitempty"`
	Performance     float64   `json:"performance,omitempty"`
	LastAssessed    time.Time `json:"lastAssessed"`
}

// PerformanceMetrics contains performance metrics for a package
type PerformanceMetrics struct {
	ResponseTime    float64   `json:"responseTime,omitempty"`    // in milliseconds
	Throughput      float64   `json:"throughput,omitempty"`      // requests per second
	ErrorRate       float64   `json:"errorRate,omitempty"`       // percentage
	MemoryUsage     int64     `json:"memoryUsage,omitempty"`     // in bytes
	CPUUsage        float64   `json:"cpuUsage,omitempty"`        // percentage
	DiskUsage       int64     `json:"diskUsage,omitempty"`       // in bytes
	NetworkIO       int64     `json:"networkIO,omitempty"`       // bytes per second
	LoadTime        float64   `json:"loadTime,omitempty"`        // in milliseconds
	LastMeasured    time.Time `json:"lastMeasured"`
}

// ResourceUsageMetrics contains resource usage metrics
type ResourceUsageMetrics struct {
	Memory    *ResourceMetric `json:"memory"`
	CPU       *ResourceMetric `json:"cpu"`
	Disk      *ResourceMetric `json:"disk"`
	Network   *ResourceMetric `json:"network"`
	GPU       *ResourceMetric `json:"gpu,omitempty"`
	Handles   *ResourceMetric `json:"handles,omitempty"`
}

// ResourceMetric represents a resource usage metric
type ResourceMetric struct {
	Current     float64   `json:"current"`
	Average     float64   `json:"average"`
	Peak        float64   `json:"peak"`
	Min         float64   `json:"min"`
	Trend       string    `json:"trend"` // "increasing", "decreasing", "stable"
	Unit        string    `json:"unit"`  // "bytes", "percent", "count", etc.
	LastUpdated time.Time `json:"lastUpdated"`
}

// CostMetrics contains cost metrics for a package
type CostMetrics struct {
	TotalCost       *Cost                `json:"totalCost"`
	CostPerUsage    *Cost                `json:"costPerUsage,omitempty"`
	CostBreakdown   map[string]*Cost     `json:"costBreakdown,omitempty"`
	CostTrend       CostTrend           `json:"costTrend"`
	CostEfficiency  float64             `json:"costEfficiency"`
	CostVariability float64             `json:"costVariability"`
	LastCalculated  time.Time           `json:"lastCalculated"`
}

// RiskFactor represents a risk factor for a package
type RiskFactor struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // "security", "maintenance", "performance", "compatibility"
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"` // "low", "medium", "high", "critical"
	Likelihood  float64   `json:"likelihood"` // 0-1
	Impact      float64   `json:"impact"`     // 0-1
	Score       float64   `json:"score"`      // calculated risk score
	Mitigation  string    `json:"mitigation,omitempty"`
	DetectedAt  time.Time `json:"detectedAt"`
}

// RecommendedAction represents a recommended action for a package
type RecommendedAction struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // "upgrade", "replace", "remove", "configure"
	Priority    string    `json:"priority"` // "low", "medium", "high", "critical"
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Package     string    `json:"package,omitempty"`
	Target      string    `json:"target,omitempty"` // target version, package, etc.
	Impact      *ActionImpact `json:"impact,omitempty"`
	Effort      string    `json:"effort"` // "low", "medium", "high"
	Deadline    *time.Time `json:"deadline,omitempty"`
	Steps       []string  `json:"steps,omitempty"`
	Reason      string    `json:"reason,omitempty"`
	Benefits    []string  `json:"benefits,omitempty"`
	Risks       []string  `json:"risks,omitempty"`
}

// ActionImpact represents the impact of a recommended action
type ActionImpact struct {
	Security     float64 `json:"security,omitempty"`     // impact on security score
	Performance  float64 `json:"performance,omitempty"`  // impact on performance score
	Reliability  float64 `json:"reliability,omitempty"`  // impact on reliability score
	Cost         *Cost   `json:"cost,omitempty"`         // cost impact
	Complexity   float64 `json:"complexity,omitempty"`   // impact on complexity
	Compatibility float64 `json:"compatibility,omitempty"` // impact on compatibility
}

// PredictedIssue represents a predicted issue for a package
type PredictedIssue struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // "security", "performance", "compatibility", "maintenance"
	Severity    string    `json:"severity"` // "low", "medium", "high", "critical"
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Package     string    `json:"package"`
	Probability float64   `json:"probability"` // 0-1
	TimeFrame   string    `json:"timeFrame"`   // "immediate", "short-term", "medium-term", "long-term"
	Impact      *IssueImpact `json:"impact,omitempty"`
	Prevention  []string  `json:"prevention,omitempty"`
	Mitigation  []string  `json:"mitigation,omitempty"`
	Confidence  float64   `json:"confidence"` // 0-1, confidence in prediction
	ModelVersion string   `json:"modelVersion,omitempty"`
	PredictedAt time.Time `json:"predictedAt"`
}

// IssueImpact represents the impact of a predicted issue
type IssueImpact struct {
	Availability float64 `json:"availability,omitempty"` // impact on availability
	Performance  float64 `json:"performance,omitempty"`  // impact on performance
	Security     float64 `json:"security,omitempty"`     // impact on security
	Data         float64 `json:"data,omitempty"`         // impact on data integrity
	Operations   float64 `json:"operations,omitempty"`   // impact on operations
	Users        int     `json:"users,omitempty"`        // number of users affected
	Revenue      *Cost   `json:"revenue,omitempty"`      // revenue impact
}

// Risk analysis types
type RiskAnalysis struct {
	AnalysisID           string                     `json:"analysisId"`
	OverallRiskScore     float64                    `json:"overallRiskScore"`
	RiskGrade           string                     `json:"riskGrade"` // "A", "B", "C", "D", "F"
	SecurityRisks       []*SecurityRisk            `json:"securityRisks,omitempty"`
	OperationalRisks    []*OperationalRisk         `json:"operationalRisks,omitempty"`
	ComplianceRisks     []*ComplianceRisk          `json:"complianceRisks,omitempty"`
	BusinessRisks       []*BusinessRisk            `json:"businessRisks,omitempty"`
	TechnicalRisks      []*TechnicalRisk           `json:"technicalRisks,omitempty"`
	RiskTrend           string                     `json:"riskTrend"` // "improving", "stable", "degrading"
	HighRiskPackages    []string                   `json:"highRiskPackages,omitempty"`
	RiskMitigations     []*RiskMitigation          `json:"riskMitigations,omitempty"`
	RiskMatrix          *RiskMatrix                `json:"riskMatrix,omitempty"`
	AnalyzedAt          time.Time                  `json:"analyzedAt"`
}

// SecurityRiskAssessment contains security risk assessment results
type SecurityRiskAssessment struct {
	AssessmentID    string              `json:"assessmentId"`
	OverallScore    float64            `json:"overallScore"`
	SecurityGrade   string             `json:"securityGrade"`
	Vulnerabilities []*Vulnerability   `json:"vulnerabilities"`
	ThreatLevel     string             `json:"threatLevel"` // "low", "medium", "high", "critical"
	ComplianceStatus map[string]string `json:"complianceStatus,omitempty"` // framework -> status
	SecurityGaps    []*SecurityGap     `json:"securityGaps,omitempty"`
	Recommendations []*SecurityRecommendation `json:"recommendations,omitempty"`
	LastScan        time.Time          `json:"lastScan"`
	NextScan        *time.Time         `json:"nextScan,omitempty"`
}

// ComplianceRiskAnalysis contains compliance risk analysis results
type ComplianceRiskAnalysis struct {
	AnalysisID      string                    `json:"analysisId"`
	ComplianceScore float64                  `json:"complianceScore"`
	Frameworks      map[string]*ComplianceStatus `json:"frameworks"`
	Violations      []*ComplianceViolation   `json:"violations,omitempty"`
	Risks           []*ComplianceRisk        `json:"risks,omitempty"`
	RemediationPlan []*RemediationAction     `json:"remediationPlan,omitempty"`
	NextAudit       *time.Time               `json:"nextAudit,omitempty"`
	AnalyzedAt      time.Time                `json:"analyzedAt"`
}

// Risk types
type SecurityRisk struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Package     string    `json:"package,omitempty"`
	Probability float64   `json:"probability"`
	Impact      float64   `json:"impact"`
	Score       float64   `json:"score"`
	CVSS        float64   `json:"cvss,omitempty"`
	CVE         string    `json:"cve,omitempty"`
	DetectedAt  time.Time `json:"detectedAt"`
}

type OperationalRisk struct {
	ID          string    `json:"id"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Probability float64   `json:"probability"`
	Impact      float64   `json:"impact"`
	Score       float64   `json:"score"`
	Services    []string  `json:"services,omitempty"`
	DetectedAt  time.Time `json:"detectedAt"`
}

type ComplianceRisk struct {
	ID          string    `json:"id"`
	Framework   string    `json:"framework"`
	Requirement string    `json:"requirement"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"`
	Probability float64   `json:"probability"`
	Impact      float64   `json:"impact"`
	Score       float64   `json:"score"`
	DetectedAt  time.Time `json:"detectedAt"`
}

type BusinessRisk struct {
	ID          string    `json:"id"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Probability float64   `json:"probability"`
	Impact      float64   `json:"impact"`
	Score       float64   `json:"score"`
	Revenue     *Cost     `json:"revenue,omitempty"`
	Customers   int       `json:"customers,omitempty"`
	DetectedAt  time.Time `json:"detectedAt"`
}

type TechnicalRisk struct {
	ID          string    `json:"id"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Probability float64   `json:"probability"`
	Impact      float64   `json:"impact"`
	Score       float64   `json:"score"`
	Systems     []string  `json:"systems,omitempty"`
	DetectedAt  time.Time `json:"detectedAt"`
}

type RiskMitigation struct {
	ID          string    `json:"id"`
	RiskID      string    `json:"riskId"`
	Strategy    string    `json:"strategy"`
	Description string    `json:"description"`
	Effectiveness float64 `json:"effectiveness"` // 0-1
	Cost        *Cost     `json:"cost,omitempty"`
	Timeline    string    `json:"timeline"`
	Owner       string    `json:"owner,omitempty"`
	Status      string    `json:"status"` // "planned", "in-progress", "completed"
}

type RiskMatrix struct {
	Dimensions map[string][]string           `json:"dimensions"`
	Cells      map[string]*RiskMatrixCell    `json:"cells"`
	Thresholds map[string]float64            `json:"thresholds"`
}

type RiskMatrixCell struct {
	Probability string  `json:"probability"`
	Impact      string  `json:"impact"`
	Level       string  `json:"level"` // "low", "medium", "high", "critical"
	Color       string  `json:"color,omitempty"`
	Count       int     `json:"count"`
}

// Security types
type SecurityGap struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"`
	Package     string    `json:"package,omitempty"`
	Remediation string    `json:"remediation,omitempty"`
	DetectedAt  time.Time `json:"detectedAt"`
}

type SecurityRecommendation struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Priority    string    `json:"priority"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Package     string    `json:"package,omitempty"`
	Action      string    `json:"action"`
	Impact      string    `json:"impact"`
	Effort      string    `json:"effort"`
	Timeline    string    `json:"timeline,omitempty"`
}

// Compliance types (removing ComplianceStatus as it's defined elsewhere)
type RequirementStatus struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Evidence    []string  `json:"evidence,omitempty"`
	Gaps        []string  `json:"gaps,omitempty"`
}

type ComplianceViolation struct {
	ID          string    `json:"id"`
	Framework   string    `json:"framework"`
	Requirement string    `json:"requirement"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"`
	Package     string    `json:"package,omitempty"`
	Evidence    string    `json:"evidence,omitempty"`
	DetectedAt  time.Time `json:"detectedAt"`
}

type RemediationAction struct {
	ID          string    `json:"id"`
	ViolationID string    `json:"violationId"`
	Action      string    `json:"action"`
	Description string    `json:"description"`
	Priority    string    `json:"priority"`
	Owner       string    `json:"owner,omitempty"`
	DueDate     *time.Time `json:"dueDate,omitempty"`
	Status      string    `json:"status"` // "planned", "in-progress", "completed"
	Cost        *Cost     `json:"cost,omitempty"`
}

// Performance analysis types
type PerformanceAnalysis struct {
	AnalysisID      string                        `json:"analysisId"`
	OverallScore    float64                      `json:"overallScore"`
	EfficiencyScore float64                      `json:"efficiencyScore"`
	PerformanceGrade string                      `json:"performanceGrade"`
	Benchmarks      []*BenchmarkResult           `json:"benchmarks,omitempty"`
	Bottlenecks     []*PerformanceBottleneck     `json:"bottlenecks,omitempty"`
	Optimizations   []*PerformanceOptimization   `json:"optimizations,omitempty"`
	ResourceUsage   *ResourceUsageAnalysis       `json:"resourceUsage,omitempty"`
	TrendAnalysis   *PerformanceTrendAnalysis    `json:"trendAnalysis,omitempty"`
	AnalyzedAt      time.Time                    `json:"analyzedAt"`
}

type BenchmarkResult struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Package     string                 `json:"package,omitempty"`
	Metrics     map[string]float64     `json:"metrics"`
	Baseline    map[string]float64     `json:"baseline,omitempty"`
	Comparison  map[string]float64     `json:"comparison,omitempty"` // percentage difference
	Status      string                 `json:"status"` // "pass", "fail", "warning"
	RunTime     time.Duration          `json:"runTime"`
	ExecutedAt  time.Time              `json:"executedAt"`
}

type PerformanceBottleneck struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // "cpu", "memory", "disk", "network"
	Description string    `json:"description"`
	Package     string    `json:"package,omitempty"`
	Severity    string    `json:"severity"`
	Impact      float64   `json:"impact"` // percentage impact on performance
	Resolution  string    `json:"resolution,omitempty"`
	DetectedAt  time.Time `json:"detectedAt"`
}

type ResourceUsageAnalysis struct {
	AnalysisID   string                         `json:"analysisId"`
	Usage        *ResourceUsageMetrics          `json:"usage"`
	Trends       map[string]*ResourceTrend      `json:"trends,omitempty"`
	Predictions  map[string]*ResourcePrediction `json:"predictions,omitempty"`
	Limits       map[string]*ResourceLimit      `json:"limits,omitempty"`
	Efficiency   *ResourceEfficiency            `json:"efficiency,omitempty"`
	AnalyzedAt   time.Time                      `json:"analyzedAt"`
}

type ResourceTrend struct {
	Resource   string    `json:"resource"`
	Trend      string    `json:"trend"` // "increasing", "decreasing", "stable"
	Slope      float64   `json:"slope"`
	Confidence float64   `json:"confidence"`
	Period     time.Duration `json:"period"`
}

type ResourcePrediction struct {
	Resource    string    `json:"resource"`
	Forecast    []float64 `json:"forecast"`
	Timeline    []time.Time `json:"timeline"`
	Confidence  float64   `json:"confidence"`
	Model       string    `json:"model,omitempty"`
}

type ResourceLimit struct {
	Resource    string  `json:"resource"`
	SoftLimit   float64 `json:"softLimit,omitempty"`
	HardLimit   float64 `json:"hardLimit,omitempty"`
	Current     float64 `json:"current"`
	Utilization float64 `json:"utilization"` // percentage
	Status      string  `json:"status"` // "normal", "warning", "critical"
}

type ResourceEfficiency struct {
	OverallScore    float64                `json:"overallScore"`
	ResourceScores  map[string]float64     `json:"resourceScores"`
	WastedResources map[string]float64     `json:"wastedResources"`
	Optimization    *ResourceOptimization  `json:"optimization,omitempty"`
}

type ResourceOptimization struct {
	PotentialSavings map[string]float64 `json:"potentialSavings"`
	Recommendations  []string          `json:"recommendations"`
	Impact          string            `json:"impact"` // "low", "medium", "high"
	Effort          string            `json:"effort"`
}

type PerformanceTrendAnalysis struct {
	OverallTrend    string                    `json:"overallTrend"`
	MetricTrends    map[string]*MetricTrend   `json:"metricTrends"`
	Anomalies       []*PerformanceAnomaly     `json:"anomalies,omitempty"`
	Seasonality     *PerformanceSeasonality   `json:"seasonality,omitempty"`
	Forecasts       []*PerformanceForecast    `json:"forecasts,omitempty"`
}

type MetricTrend struct {
	Metric     string    `json:"metric"`
	Trend      string    `json:"trend"`
	Change     float64   `json:"change"` // percentage change
	Period     time.Duration `json:"period"`
	Confidence float64   `json:"confidence"`
}

type PerformanceAnomaly struct {
	ID          string    `json:"id"`
	Metric      string    `json:"metric"`
	Timestamp   time.Time `json:"timestamp"`
	Value       float64   `json:"value"`
	Expected    float64   `json:"expected"`
	Deviation   float64   `json:"deviation"`
	Severity    string    `json:"severity"`
	Duration    time.Duration `json:"duration,omitempty"`
	Cause       string    `json:"cause,omitempty"`
}

type PerformanceSeasonality struct {
	Metric      string        `json:"metric"`
	Period      time.Duration `json:"period"`
	Amplitude   float64       `json:"amplitude"`
	Phase       float64       `json:"phase"`
	Strength    float64       `json:"strength"`
	Confidence  float64       `json:"confidence"`
}

type PerformanceForecast struct {
	Metric      string      `json:"metric"`
	Timeline    []time.Time `json:"timeline"`
	Values      []float64   `json:"values"`
	LowerBound  []float64   `json:"lowerBound,omitempty"`
	UpperBound  []float64   `json:"upperBound,omitempty"`
	Confidence  float64     `json:"confidence"`
	Model       string      `json:"model,omitempty"`
}

// Optimization types that are referenced but not defined
type OptimizationObjectives struct {
	CostOptimization        bool   `json:"costOptimization"`
	PerformanceOptimization bool   `json:"performanceOptimization"`
	SecurityOptimization    bool   `json:"securityOptimization"`
	QualityOptimization     bool   `json:"qualityOptimization"`
	EfficiencyOptimization  bool   `json:"efficiencyOptimization,omitempty"`
	Priorities              map[string]float64 `json:"priorities,omitempty"`
	Constraints             *OptimizationConstraints `json:"constraints,omitempty"`
}

type OptimizationConstraints struct {
	MaxCostIncrease     float64   `json:"maxCostIncrease,omitempty"`
	MinPerformanceGain  float64   `json:"minPerformanceGain,omitempty"`
	MaxRiskIncrease     float64   `json:"maxRiskIncrease,omitempty"`
	Budget              *Cost     `json:"budget,omitempty"`
	Timeline            time.Duration `json:"timeline,omitempty"`
	RequiredCompliance  []string  `json:"requiredCompliance,omitempty"`
}

// Optimization recommendation types
type VersionOptimization struct {
	ID              string   `json:"id"`
	Package         string   `json:"package"`
	CurrentVersion  string   `json:"currentVersion"`
	RecommendedVersion string `json:"recommendedVersion"`
	Reason          string   `json:"reason"`
	Benefits        []string `json:"benefits"`
	Risks           []string `json:"risks,omitempty"`
	Impact          *OptimizationImpact `json:"impact"`
	Effort          string   `json:"effort"`
	Priority        string   `json:"priority"`
}

type DependencyOptimization struct {
	ID              string   `json:"id"`
	Type            string   `json:"type"` // "remove", "replace", "consolidate"
	Description     string   `json:"description"`
	AffectedPackages []string `json:"affectedPackages"`
	Recommendation  string   `json:"recommendation"`
	Benefits        []string `json:"benefits"`
	Risks           []string `json:"risks,omitempty"`
	Impact          *OptimizationImpact `json:"impact"`
	Effort          string   `json:"effort"`
	Priority        string   `json:"priority"`
}

type SecurityOptimization struct {
	ID              string   `json:"id"`
	SecurityIssue   string   `json:"securityIssue"`
	Package         string   `json:"package,omitempty"`
	Recommendation  string   `json:"recommendation"`
	Action          string   `json:"action"`
	Benefits        []string `json:"benefits"`
	Impact          *OptimizationImpact `json:"impact"`
	Effort          string   `json:"effort"`
	Priority        string   `json:"priority"`
	Deadline        *time.Time `json:"deadline,omitempty"`
}

type PerformanceOptimization struct {
	ID              string   `json:"id"`
	PerformanceIssue string  `json:"performanceIssue"`
	Package         string   `json:"package,omitempty"`
	Recommendation  string   `json:"recommendation"`
	ExpectedGain    float64  `json:"expectedGain"` // percentage improvement
	Benefits        []string `json:"benefits"`
	Impact          *OptimizationImpact `json:"impact"`
	Effort          string   `json:"effort"`
	Priority        string   `json:"priority"`
}

type OptimizationAction struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Package     string    `json:"package,omitempty"`
	Action      string    `json:"action"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Benefits    []string  `json:"benefits"`
	Risks       []string  `json:"risks,omitempty"`
	Impact      *OptimizationImpact `json:"impact"`
	Effort      string    `json:"effort"`
	Timeline    string    `json:"timeline,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"` // other actions this depends on
	Status      string    `json:"status,omitempty"` // "pending", "in-progress", "completed", "cancelled"
}

type OptimizationImpact struct {
	Performance  float64 `json:"performance,omitempty"`  // percentage change in performance score
	Security     float64 `json:"security,omitempty"`     // percentage change in security score
	Cost         *Cost   `json:"cost,omitempty"`         // cost impact
	Quality      float64 `json:"quality,omitempty"`      // percentage change in quality score
	Reliability  float64 `json:"reliability,omitempty"`  // percentage change in reliability score
	Maintainability float64 `json:"maintainability,omitempty"` // percentage change in maintainability score
	Risk         float64 `json:"risk,omitempty"`         // percentage change in risk score
	Complexity   float64 `json:"complexity,omitempty"`   // percentage change in complexity
}

type OptimizationBenefits struct {
	CostSavings     *Cost   `json:"costSavings,omitempty"`
	PerformanceGain float64 `json:"performanceGain,omitempty"`
	SecurityGain    float64 `json:"securityGain,omitempty"`
	QualityGain     float64 `json:"qualityGain,omitempty"`
	RiskReduction   float64 `json:"riskReduction,omitempty"`
	ComplexityReduction float64 `json:"complexityReduction,omitempty"`
	MaintenanceReduction float64 `json:"maintenanceReduction,omitempty"`
}

// Machine learning types
type MLRecommendation struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Confidence  float64   `json:"confidence"`
	Model       string    `json:"model"`
	Features    map[string]float64 `json:"features,omitempty"`
	Impact      *OptimizationImpact `json:"impact,omitempty"`
	Reasoning   string    `json:"reasoning,omitempty"`
	GeneratedAt time.Time `json:"generatedAt"`
}

// Issue prediction types
type IssuePrediction struct {
	PredictionID    string            `json:"predictionId"`
	PredictedIssues []*PredictedIssue `json:"predictedIssues"`
	OverallRisk     float64           `json:"overallRisk"`
	Confidence      float64           `json:"confidence"`
	ModelVersion    string            `json:"modelVersion"`
	PredictedAt     time.Time         `json:"predictedAt"`
}

// Upgrade recommendation types
type UpgradeRecommendation struct {
	ID              string    `json:"id"`
	Package         string    `json:"package"`
	FromVersion     string    `json:"fromVersion"`
	ToVersion       string    `json:"toVersion"`
	UpgradeType     string    `json:"upgradeType"` // "major", "minor", "patch", "security"
	Reason          string    `json:"reason"`
	Priority        string    `json:"priority"`
	Benefits        []string  `json:"benefits"`
	Risks           []string  `json:"risks,omitempty"`
	BreakingChanges []string  `json:"breakingChanges,omitempty"`
	TestingRequired bool      `json:"testingRequired"`
	RollbackPlan    string    `json:"rollbackPlan,omitempty"`
	Impact          *OptimizationImpact `json:"impact"`
	Effort          string    `json:"effort"`
	Timeline        string    `json:"timeline,omitempty"`
	Confidence      float64   `json:"confidence"`
	RecommendedAt   time.Time `json:"recommendedAt"`
}

// Evolution analysis types
type EvolutionAnalysis struct {
	AnalysisID      string                    `json:"analysisId"`
	TimeRange       *TimeRange                `json:"timeRange"`
	PackageEvolution []*PackageEvolution      `json:"packageEvolution"`
	TrendAnalysis   *EvolutionTrendAnalysis   `json:"trendAnalysis"`
	Predictions     []*EvolutionPrediction    `json:"predictions,omitempty"`
	Patterns        []*EvolutionPattern       `json:"patterns,omitempty"`
	AnalyzedAt      time.Time                 `json:"analyzedAt"`
}

type PackageEvolution struct {
	Package         *PackageReference         `json:"package"`
	VersionHistory  []*VersionHistoryEntry    `json:"versionHistory"`
	UsageEvolution  []*UsageEvolutionEntry    `json:"usageEvolution,omitempty"`
	QualityEvolution []*QualityEvolutionEntry `json:"qualityEvolution,omitempty"`
	SecurityEvolution []*SecurityEvolutionEntry `json:"securityEvolution,omitempty"`
	EvolutionScore  float64                   `json:"evolutionScore"`
	Maturity        string                    `json:"maturity"` // "experimental", "stable", "mature", "legacy"
	Lifecycle       string                    `json:"lifecycle"` // "growing", "stable", "declining", "deprecated"
}

type VersionHistoryEntry struct {
	Version     string    `json:"version"`
	ReleasedAt  time.Time `json:"releasedAt"`
	ChangeType  string    `json:"changeType"` // "major", "minor", "patch", "security"
	Changes     []string  `json:"changes,omitempty"`
	Adoption    float64   `json:"adoption,omitempty"` // percentage of users who adopted this version
	Stability   float64   `json:"stability,omitempty"` // stability score
}

type UsageEvolutionEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Usage     int64     `json:"usage"`
	Users     int       `json:"users,omitempty"`
	Growth    float64   `json:"growth,omitempty"` // growth rate
}

type QualityEvolutionEntry struct {
	Timestamp     time.Time `json:"timestamp"`
	QualityScore  float64   `json:"qualityScore"`
	TestCoverage  float64   `json:"testCoverage,omitempty"`
	Documentation float64   `json:"documentation,omitempty"`
	BugCount      int       `json:"bugCount,omitempty"`
}

type SecurityEvolutionEntry struct {
	Timestamp         time.Time `json:"timestamp"`
	SecurityScore     float64   `json:"securityScore"`
	VulnerabilityCount int      `json:"vulnerabilityCount"`
	CriticalVulns     int       `json:"criticalVulns,omitempty"`
	HighVulns         int       `json:"highVulns,omitempty"`
}

type EvolutionTrendAnalysis struct {
	OverallTrend    string                   `json:"overallTrend"`
	UsageTrend      string                   `json:"usageTrend"`
	QualityTrend    string                   `json:"qualityTrend"`
	SecurityTrend   string                   `json:"securityTrend"`
	TrendStrength   float64                  `json:"trendStrength"`
	PackageTrends   map[string]*PackageTrend `json:"packageTrends,omitempty"`
}

type PackageTrend struct {
	Package       string  `json:"package"`
	UsageTrend    string  `json:"usageTrend"`
	QualityTrend  string  `json:"qualityTrend"`
	SecurityTrend string  `json:"securityTrend"`
	TrendScore    float64 `json:"trendScore"`
	Recommendation string `json:"recommendation"`
}

type EvolutionPrediction struct {
	Package         string    `json:"package"`
	Prediction      string    `json:"prediction"`
	Confidence      float64   `json:"confidence"`
	TimeFrame       string    `json:"timeFrame"`
	Rationale       string    `json:"rationale"`
	Impact          string    `json:"impact"`
	Recommendation  string    `json:"recommendation,omitempty"`
}

type EvolutionPattern struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"` // "growth", "decline", "plateau", "revival", "disruption"
	Description string   `json:"description"`
	Packages    []string `json:"packages"`
	Confidence  float64  `json:"confidence"`
	Duration    time.Duration `json:"duration"`
	Impact      string   `json:"impact"`
}

// Anomaly detection types
type AnomalyDetectionResult struct {
	DetectionID string            `json:"detectionId"`
	Anomalies   []*DetectedAnomaly `json:"anomalies"`
	Model       string            `json:"model"`
	Sensitivity float64           `json:"sensitivity"`
	Threshold   float64           `json:"threshold"`
	DetectedAt  time.Time         `json:"detectedAt"`
}

type DetectedAnomaly struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Package     string    `json:"package,omitempty"`
	Metric      string    `json:"metric"`
	Timestamp   time.Time `json:"timestamp"`
	Value       float64   `json:"value"`
	Expected    float64   `json:"expected"`
	Deviation   float64   `json:"deviation"`
	AnomalyScore float64  `json:"anomalyScore"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// Replacement suggestion types
type ReplacementCriteria struct {
	MaxCostIncrease     float64  `json:"maxCostIncrease,omitempty"`
	MinQualityScore     float64  `json:"minQualityScore,omitempty"`
	RequiredFeatures    []string `json:"requiredFeatures,omitempty"`
	LicenseCompatibility []string `json:"licenseCompatibility,omitempty"`
	MaxMigrationEffort  string   `json:"maxMigrationEffort,omitempty"`
	SecurityRequirements []string `json:"securityRequirements,omitempty"`
	PerformanceRequirements map[string]float64 `json:"performanceRequirements,omitempty"`
}

type ReplacementSuggestion struct {
	ID                 string               `json:"id"`
	OriginalPackage    *PackageReference    `json:"originalPackage"`
	ReplacementPackage *PackageReference    `json:"replacementPackage"`
	Reason             string               `json:"reason"`
	Benefits           []string             `json:"benefits"`
	Drawbacks          []string             `json:"drawbacks,omitempty"`
	MigrationPlan      *MigrationPlan       `json:"migrationPlan,omitempty"`
	Impact             *ReplacementImpact   `json:"impact"`
	Confidence         float64              `json:"confidence"`
	Effort             string               `json:"effort"`
	Timeline           string               `json:"timeline,omitempty"`
	SuggestedAt        time.Time            `json:"suggestedAt"`
}

type MigrationPlan struct {
	Steps              []string    `json:"steps"`
	EstimatedDuration  time.Duration `json:"estimatedDuration"`
	RequiredResources  []string    `json:"requiredResources,omitempty"`
	RiskMitigation     []string    `json:"riskMitigation,omitempty"`
	RollbackPlan       []string    `json:"rollbackPlan,omitempty"`
	TestingStrategy    string      `json:"testingStrategy,omitempty"`
}

type ReplacementImpact struct {
	Cost         *Cost   `json:"cost,omitempty"`
	Performance  float64 `json:"performance,omitempty"`
	Security     float64 `json:"security,omitempty"`
	Quality      float64 `json:"quality,omitempty"`
	Compatibility float64 `json:"compatibility,omitempty"`
	Effort       string  `json:"effort"`
	Risk         float64 `json:"risk,omitempty"`
}

// Optimized graph types
type OptimizedGraph struct {
	OriginalGraph *DependencyGraph       `json:"originalGraph"`
	OptimizedGraph *DependencyGraph      `json:"optimizedGraph"`
	Optimizations []*GraphOptimization   `json:"optimizations"`
	Improvements  *OptimizationMetrics   `json:"improvements"`
	OptimizedAt   time.Time              `json:"optimizedAt"`
}

type GraphOptimization struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Applied     bool     `json:"applied"`
	Impact      *OptimizationImpact `json:"impact"`
}

type OptimizationMetrics struct {
	ComplexityReduction float64 `json:"complexityReduction"`
	SizeReduction      float64 `json:"sizeReduction"`
	CycleReduction     int     `json:"cycleReduction"`
	PerformanceGain    float64 `json:"performanceGain"`
	CostSavings        *Cost   `json:"costSavings,omitempty"`
}

// Health types for analyzer
type AnalyzerHealth struct {
	Status       string                 `json:"status"` // "healthy", "warning", "critical"
	Uptime       time.Duration          `json:"uptime"`
	LastRestart  time.Time              `json:"lastRestart,omitempty"`
	Components   map[string]string      `json:"components"` // component -> status
	Metrics      *HealthMetrics         `json:"metrics"`
	Issues       []string               `json:"issues,omitempty"`
	CheckedAt    time.Time              `json:"checkedAt"`
}

type HealthMetrics struct {
	AnalysesCompleted  int64   `json:"analysesCompleted"`
	AnalysesPerSecond  float64 `json:"analysesPerSecond"`
	AverageResponseTime time.Duration `json:"averageResponseTime"`
	ErrorRate          float64 `json:"errorRate"`
	CacheHitRate       float64 `json:"cacheHitRate,omitempty"`
	MemoryUsage        int64   `json:"memoryUsage"`
	CPUUsage           float64 `json:"cpuUsage"`
}

// Distribution statistics
type DistributionStats struct {
	Mean     float64 `json:"mean"`
	Median   float64 `json:"median"`
	Mode     float64 `json:"mode,omitempty"`
	StdDev   float64 `json:"stdDev"`
	Variance float64 `json:"variance"`
	Skewness float64 `json:"skewness,omitempty"`
	Kurtosis float64 `json:"kurtosis,omitempty"`
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
	Range    float64 `json:"range"`
	Percentiles map[string]float64 `json:"percentiles,omitempty"` // "p25", "p50", "p75", "p90", "p95", "p99"
}