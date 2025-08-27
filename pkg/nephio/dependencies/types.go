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
	"context"
	"fmt"
	"time"
	
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
)

// Core missing types for dependency analyzer

// TimeRange defines a time range for analysis
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ScopeFilter defines filtering criteria for analysis scope
type ScopeFilter struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Values     []string `json:"values"`
	Operator   string   `json:"operator,omitempty"` // "include", "exclude", "equals", "contains"
	CaseSensitive bool  `json:"caseSensitive,omitempty"`
}

// VersionRange defines a version range constraint
type VersionRange struct {
	Min         string `json:"min,omitempty"`
	Max         string `json:"max,omitempty"`
	MinInclusive bool   `json:"minInclusive,omitempty"`
	MaxInclusive bool   `json:"maxInclusive,omitempty"`
	Constraint   string `json:"constraint,omitempty"` // Semantic version constraint like ">=1.0.0,<2.0.0"
}

// GraphAnalysis contains graph analysis results
type GraphAnalysis struct {
	TotalNodes      int                    `json:"totalNodes"`
	TotalEdges      int                    `json:"totalEdges"`
	ConnectedComponents int                `json:"connectedComponents"`
	CyclicDependencies []*CyclicDependency `json:"cyclicDependencies"`
	CriticalPath    []*GraphNode          `json:"criticalPath"`
	Metrics         *GraphMetrics         `json:"metrics"`
}

// CyclicDependency represents a circular dependency
type CyclicDependency struct {
	Cycle       []*GraphNode  `json:"cycle"`
	Impact      string        `json:"impact"`
	Severity    string        `json:"severity"`
	Resolution  string        `json:"resolution,omitempty"`
}



// SecurityInfo contains security-related information
type SecurityInfo struct {
	Vulnerabilities  []*VulnerabilityInfo `json:"vulnerabilities,omitempty"`
	LastSecurityScan time.Time           `json:"lastSecurityScan,omitempty"`
	SecurityScore    float64             `json:"securityScore,omitempty"`
	TrustLevel       string              `json:"trustLevel,omitempty"`
}

// VulnerabilityInfo represents a security vulnerability
type VulnerabilityInfo struct {
	ID          string    `json:"id"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	CVSS        float64   `json:"cvss,omitempty"`
	Published   time.Time `json:"published"`
	FixedIn     string    `json:"fixedIn,omitempty"`
	References  []string  `json:"references,omitempty"`
}

// AnalysisReport represents a comprehensive dependency analysis report
type AnalysisReport struct {
	ID              string          `json:"id"`
	RequestID       string          `json:"requestId,omitempty"`
	Type            string          `json:"type"`
	Status          string          `json:"status"`
	Progress        float64         `json:"progress"`
	
	// Analysis scope
	ScopeFilters    []*ScopeFilter  `json:"scopeFilters,omitempty"`
	TimeRange       *TimeRange      `json:"timeRange,omitempty"`
	
	// Analysis results
	Summary         *AnalysisSummary `json:"summary"`
	GraphAnalysis   *GraphAnalysis   `json:"graphAnalysis,omitempty"`
	
	// Execution metadata
	StartedAt       time.Time       `json:"startedAt"`
	CompletedAt     *time.Time      `json:"completedAt,omitempty"`
	Duration        time.Duration   `json:"duration,omitempty"`
	ExecutedBy      string          `json:"executedBy,omitempty"`
	
	// Configuration
	Config          *AnalysisConfig  `json:"config,omitempty"`
	
	// Error information
	Errors          []string         `json:"errors,omitempty"`
	Warnings        []string         `json:"warnings,omitempty"`
}

// AnalysisSummary provides a high-level summary of analysis results
type AnalysisSummary struct {
	TotalPackages      int     `json:"totalPackages"`
	DirectDependencies int     `json:"directDependencies"`
	TransitiveDeps     int     `json:"transitiveDependencies"`
	OutdatedPackages   int     `json:"outdatedPackages"`
	VulnerablePackages int     `json:"vulnerablePackages"`
	LicenseIssues      int     `json:"licenseIssues"`
	CyclicDeps         int     `json:"cyclicDependencies"`
	RiskScore          float64 `json:"riskScore"`
	HealthScore        float64 `json:"healthScore"`
}

// AnalysisConfig contains configuration for dependency analysis
type AnalysisConfig struct {
	// Analysis scope
	Scope               []*ScopeFilter       `json:"scope,omitempty"`
	IncludeTransitive   bool                 `json:"includeTransitive"`
	MaxDepth            int                  `json:"maxDepth,omitempty"`
	
	// Analysis types
	EnableGraphAnalysis      bool            `json:"enableGraphAnalysis"`
	EnableSecurityAnalysis   bool            `json:"enableSecurityAnalysis"`
	EnableLicenseAnalysis    bool            `json:"enableLicenseAnalysis"`
	EnableVersionAnalysis    bool            `json:"enableVersionAnalysis"`
	EnableUsageAnalysis      bool            `json:"enableUsageAnalysis"`
	EnableCostAnalysis       bool            `json:"enableCostAnalysis"`
	EnableHealthAnalysis     bool            `json:"enableHealthAnalysis"`
	EnableRiskAnalysis       bool            `json:"enableRiskAnalysis"`
	EnablePerformanceAnalysis bool           `json:"enablePerformanceAnalysis"`
	EnableMLAnalysis         bool            `json:"enableMLAnalysis"`
	EnableMLOptimization     bool            `json:"enableMLOptimization"`
	
	// Output options
	OutputFormat            string           `json:"outputFormat"`          // "json", "yaml", "csv", "html"
	IncludeRecommendations  bool            `json:"includeRecommendations"`
	IncludeVisualizations   bool            `json:"includeVisualizations"`
	IncludeMetrics          bool            `json:"includeMetrics"`
	
	// Performance options
	EnableCaching           bool            `json:"enableCaching"`
	EnableConcurrency       bool            `json:"enableConcurrency"`
	WorkerCount             int             `json:"workerCount,omitempty"`
	QueueSize               int             `json:"queueSize,omitempty"`
	EnableTrendAnalysis     bool            `json:"enableTrendAnalysis"`
	EnableCostProjection    bool            `json:"enableCostProjection"`
	
	// Filtering and thresholds
	MinimumRiskScore        float64         `json:"minimumRiskScore,omitempty"`
	SecuritySeverityFilter  []string        `json:"securitySeverityFilter,omitempty"`
	LicenseFilter           []string        `json:"licenseFilter,omitempty"`
	
	// External integrations
	PackageRegistries       []string        `json:"packageRegistries,omitempty"`
	SecurityDatabases       []string        `json:"securityDatabases,omitempty"`
	LicenseDatabases        []string        `json:"licenseDatabases,omitempty"`
	
	// Component configurations (references to separate config types)
	UsageAnalyzerConfig     *UsageAnalyzerConfig     `json:"usageAnalyzerConfig,omitempty"`
	CostAnalyzerConfig      *CostAnalyzerConfig      `json:"costAnalyzerConfig,omitempty"`
	HealthAnalyzerConfig    *HealthAnalyzerConfig    `json:"healthAnalyzerConfig,omitempty"`
	RiskAnalyzerConfig      *RiskAnalyzerConfig      `json:"riskAnalyzerConfig,omitempty"`
	PerformanceAnalyzerConfig *PerformanceAnalyzerConfig `json:"performanceAnalyzerConfig,omitempty"`
	OptimizationEngineConfig *OptimizationEngineConfig `json:"optimizationEngineConfig,omitempty"`
	MLOptimizerConfig       *MLOptimizerConfig       `json:"mlOptimizerConfig,omitempty"`
	UsageCollectorConfig    *UsageCollectorConfig    `json:"usageCollectorConfig,omitempty"`
	MetricsCollectorConfig  *MetricsCollectorConfig  `json:"metricsCollectorConfig,omitempty"`
	EventProcessorConfig    *EventProcessorConfig    `json:"eventProcessorConfig,omitempty"`
	PredictionModelConfig   *PredictionModelConfig   `json:"predictionModelConfig,omitempty"`
	RecommendationModelConfig *RecommendationModelConfig `json:"recommendationModelConfig,omitempty"`
	AnomalyDetectorConfig   *AnomalyDetectorConfig   `json:"anomalyDetectorConfig,omitempty"`
	
	// System metadata
	Version              string `json:"version"`
	Currency             string `json:"currency,omitempty"`
}

// VersionStatistics contains statistics about version resolution
type VersionStatistics struct {
	TotalPackages      int           `json:"totalPackages"`
	ResolvedPackages   int           `json:"resolvedPackages"`
	ConflictedPackages int           `json:"conflictedPackages"`
	CacheHits          int           `json:"cacheHits"`
	CacheMisses        int           `json:"cacheMisses"`
	ResolutionTime     time.Duration `json:"resolutionTime"`
}

// VersionResolutionResult contains aggregate results for version resolution of multiple packages
type VersionResolutionResult struct {
	Success        bool                              `json:"success"`
	Resolutions    map[string]*VersionResolution     `json:"resolutions"`
	Conflicts      []*VersionConflict                `json:"conflicts"`
	Statistics     *VersionStatistics                `json:"statistics"`
	ResolutionTime time.Duration                     `json:"resolutionTime"`
}


// ConflictStatistics contains statistics about conflict detection
type ConflictStatistics struct {
	TotalConflicts     int           `json:"totalConflicts"`
	ResolvedConflicts  int           `json:"resolvedConflicts"`
	UnresolvedConflicts int          `json:"unresolvedConflicts"`
	DetectionTime      time.Duration `json:"detectionTime"`
}

// Component configuration types (placeholder implementations)
type UsageAnalyzerConfig struct{}
type CostAnalyzerConfig struct{}
type HealthAnalyzerConfig struct{}
type RiskAnalyzerConfig struct{}
type PerformanceAnalyzerConfig struct{}
type OptimizationEngineConfig struct{}
type MLOptimizerConfig struct{}
type UsageCollectorConfig struct{}
type MetricsCollectorConfig struct{}
type EventProcessorConfig struct{}
type PredictionModelConfig struct{}
type RecommendationModelConfig struct{}
type AnomalyDetectorConfig struct{}
type AnalysisCacheConfig struct {
	TTL               time.Duration `json:"ttl"`
	MaxEntries        int           `json:"maxEntries"`
	CleanupInterval   time.Duration `json:"cleanupInterval"`
}
type DataStoreConfig struct{}

// Validate validates the analyzer configuration
func (c *AnalysisConfig) Validate() error {
	// Add validation logic here
	return nil
}


// Metrics and analysis component types (simplified definitions)
type UsageAnalyzer struct{}
type CostAnalyzer struct{}
type HealthAnalyzer struct{}
type RiskAnalyzer struct{}
type PerformanceAnalyzer struct{}
type CostProvider interface{}

// Constructor functions (placeholder implementations)
func NewUsageAnalyzer(config *UsageAnalyzerConfig) (*UsageAnalyzer, error) { return &UsageAnalyzer{}, nil }
func NewCostAnalyzer(config *CostAnalyzerConfig) (*CostAnalyzer, error) { return &CostAnalyzer{}, nil }
func NewHealthAnalyzer(config *HealthAnalyzerConfig) (*HealthAnalyzer, error) { return &HealthAnalyzer{}, nil }
func NewRiskAnalyzer(config *RiskAnalyzerConfig) (*RiskAnalyzer, error) { return &RiskAnalyzer{}, nil }
func NewPerformanceAnalyzer(config *PerformanceAnalyzerConfig) (*PerformanceAnalyzer, error) { return &PerformanceAnalyzer{}, nil }

// Additional analysis result types that are referenced

// UsageAnalysisResult represents package usage analysis results
type UsageAnalysisResult struct {
	PackageName         string            `json:"packageName"`
	TotalUsage          int64             `json:"totalUsage"`
	UniqueUsers         int               `json:"uniquUsers"`
	UsageByVersion      map[string]int64  `json:"usageByVersion"`
	UsageByEnvironment  map[string]int64  `json:"usageByEnvironment"`
	UsageTrends         []*UsageTrend     `json:"usageTrends,omitempty"`
	PopularityScore     float64           `json:"popularityScore"`
	AdoptionRate        float64           `json:"adoptionRate"`
	UsageGrowthRate     float64           `json:"usageGrowthRate"`
	PeakUsagePeriods    []*TimePeriod     `json:"peakUsagePeriods,omitempty"`
	UnusedPeriods       []*TimePeriod     `json:"unusedPeriods,omitempty"`
	RecommendedAction   string            `json:"recommendedAction,omitempty"`
}

// UsageTrend represents usage trend over time
type UsageTrend struct {
	Period      *TimePeriod `json:"period"`
	Usage       int64       `json:"usage"`
	GrowthRate  float64     `json:"growthRate"`
	Anomalies   []string    `json:"anomalies,omitempty"`
}

// TimePeriod represents a time period
type TimePeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Label string    `json:"label,omitempty"`
}

// CostAnalysisResult represents cost analysis results
type CostAnalysisResult struct {
	PackageName         string                 `json:"packageName"`
	TotalCost           float64               `json:"totalCost"`
	CostBreakdown       *CostBreakdown        `json:"costBreakdown"`
	CostPerUser         float64               `json:"costPerUser,omitempty"`
	CostPerEnvironment  map[string]float64    `json:"costPerEnvironment,omitempty"`
	CostTrends          []*CostTrend          `json:"costTrends,omitempty"`
	CostOptimizations   []*CostOptimization   `json:"costOptimizations,omitempty"`
	ProjectedCost       *CostProjection       `json:"projectedCost,omitempty"`
	CostRisk           string                `json:"costRisk,omitempty"`
	RecommendedActions []string              `json:"recommendedActions,omitempty"`
}

// CostBreakdown represents detailed cost breakdown
type CostBreakdown struct {
	LicenseCost     float64            `json:"licenseCost"`
	SupportCost     float64            `json:"supportCost"`
	MaintenanceCost float64            `json:"maintenanceCost"`
	InfrastructureCost float64         `json:"infrastructureCost"`
	OperationalCost float64            `json:"operationalCost"`
	CustomBreakdown map[string]float64 `json:"customBreakdown,omitempty"`
}


// CostOptimization represents a cost optimization opportunity
type CostOptimization struct {
	ID              string    `json:"id"`
	Type            string    `json:"type"`
	Description     string    `json:"description"`
	PotentialSaving float64   `json:"potentialSaving"`
	Effort          string    `json:"effort"`      // "low", "medium", "high"
	Risk           string    `json:"risk"`        // "low", "medium", "high"
	Timeline        string    `json:"timeline"`    // Expected implementation time
	Priority        string    `json:"priority"`    // "low", "medium", "high", "critical"
}

// CostProjection represents cost projections
type CostProjection struct {
	ProjectionPeriod *TimePeriod `json:"projectionPeriod"`
	BaselineCost     float64     `json:"baselineCost"`
	ProjectedCost    float64     `json:"projectedCost"`
	ConfidenceLevel  float64     `json:"confidenceLevel"`
	Assumptions      []string    `json:"assumptions,omitempty"`
	Scenarios        []*CostScenario `json:"scenarios,omitempty"`
}

// CostScenario represents different cost scenarios
type CostScenario struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Cost        float64 `json:"cost"`
	Probability float64 `json:"probability"`
	Factors     []string `json:"factors,omitempty"`
}

// Usage analysis types

// UsagePattern represents usage patterns for packages
type UsagePattern struct {
	Pattern     string    `json:"pattern"`
	Frequency   int64     `json:"frequency"`
	TimeRange   *TimePeriod `json:"timeRange"`
	Description string    `json:"description,omitempty"`
}

// PeakUsageTime represents peak usage time periods
type PeakUsageTime struct {
	StartTime   time.Time `json:"startTime"`
	EndTime     time.Time `json:"endTime"`
	Usage       int64     `json:"usage"`
	Description string    `json:"description,omitempty"`
}

// SeasonalPattern represents seasonal usage patterns
type SeasonalPattern struct {
	Season      string    `json:"season"`
	Multiplier  float64   `json:"multiplier"`
	StartDate   time.Time `json:"startDate"`
	EndDate     time.Time `json:"endDate"`
	Description string    `json:"description,omitempty"`
}

// UsageRanking represents package usage rankings
type UsageRanking struct {
	PackageName string  `json:"packageName"`
	UsageCount  int64   `json:"usageCount"`
	Rank        int     `json:"rank"`
	Score       float64 `json:"score,omitempty"`
}

// TrendingPackage represents trending packages
type TrendingPackage struct {
	PackageName string  `json:"packageName"`
	GrowthRate  float64 `json:"growthRate"`
	UsageCount  int64   `json:"usageCount"`
	TrendScore  float64 `json:"trendScore"`
	Period      *TimePeriod `json:"period"`
}

// UnusedPackage represents packages that are not being used
type UnusedPackage struct {
	PackageName     string    `json:"packageName"`
	LastUsed        time.Time `json:"lastUsed"`
	DaysUnused      int       `json:"daysUnused"`
	RecommendAction string    `json:"recommendAction,omitempty"`
}

// UnderutilizedPackage represents packages with low utilization
type UnderutilizedPackage struct {
	PackageName        string  `json:"packageName"`
	UtilizationRate    float64 `json:"utilizationRate"`
	ExpectedUtilization float64 `json:"expectedUtilization"`
	RecommendAction    string  `json:"recommendAction,omitempty"`
}

// Additional analysis types

// UsageOptimization represents a usage optimization recommendation
type UsageOptimization struct {
	ID             string  `json:"id"`
	Type           string  `json:"type"`
	Description    string  `json:"description"`
	Priority       string  `json:"priority"` // high, medium, low
	PotentialSavings float64 `json:"potentialSavings,omitempty"`
}

// Cost represents monetary cost
type Cost struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Period   string  `json:"period,omitempty"` // monthly, yearly, etc.
}

// PackageCost represents cost associated with a specific package
type PackageCost struct {
	PackageName string  `json:"packageName"`
	Cost        *Cost   `json:"cost"`
	UsageCount  int64   `json:"usageCount"`
	CostPerUse  float64 `json:"costPerUse,omitempty"`
}

// CostOptimizationOpportunity represents a cost optimization opportunity
type CostOptimizationOpportunity struct {
	ID               string  `json:"id"`
	Type             string  `json:"type"`
	Description      string  `json:"description"`
	PotentialSavings *Cost   `json:"potentialSavings"`
	ImplementationEffort string `json:"implementationEffort"` // low, medium, high
	Priority         string  `json:"priority"`
}

// CostBenchmarkComparison represents comparison against benchmarks
type CostBenchmarkComparison struct {
	BenchmarkType string  `json:"benchmarkType"`
	YourCost      *Cost   `json:"yourCost"`
	BenchmarkCost *Cost   `json:"benchmarkCost"`
	Variance      float64 `json:"variance"` // percentage difference
	Rating        string  `json:"rating"`   // excellent, good, average, poor
}

// Health analysis types

// SecurityHealth represents security health metrics
type SecurityHealth struct {
	Score                float64                `json:"score"`
	VulnerabilityCount   int                    `json:"vulnerabilityCount"`
	CriticalVulnerabilities int                 `json:"criticalVulnerabilities"`
	LastSecurityScan     time.Time              `json:"lastSecurityScan"`
	SecurityIssues       []*SecurityIssue       `json:"securityIssues,omitempty"`
}

// MaintenanceHealth represents maintenance health metrics
type MaintenanceHealth struct {
	Score               float64   `json:"score"`
	OutdatedPackages    int       `json:"outdatedPackages"`
	DeprecatedPackages  int       `json:"deprecatedPackages"`
	LastMaintenance     time.Time `json:"lastMaintenance"`
	MaintenanceIssues   []string  `json:"maintenanceIssues,omitempty"`
}

// QualityHealth represents code quality health metrics
type QualityHealth struct {
	Score           float64  `json:"score"`
	CodeCoverage    float64  `json:"codeCoverage"`
	TestsCount      int      `json:"testsCount"`
	LintingIssues   int      `json:"lintingIssues"`
	QualityIssues   []string `json:"qualityIssues,omitempty"`
}

// PerformanceHealth represents performance health metrics
type PerformanceHealth struct {
	Score              float64  `json:"score"`
	ResponseTime       float64  `json:"responseTime"`
	ThroughputScore    float64  `json:"throughputScore"`
	ResourceUsage      float64  `json:"resourceUsage"`
	PerformanceIssues  []string `json:"performanceIssues,omitempty"`
}

// SecurityIssue represents a security issue
type SecurityIssue struct {
	ID          string    `json:"id"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Package     string    `json:"package"`
	FixAvailable bool     `json:"fixAvailable"`
	DetectedAt  time.Time `json:"detectedAt"`
}

// Additional analysis result types

// RiskAnalysis represents risk analysis results
type RiskAnalysis struct {
	AnalysisID      string         `json:"analysisId"`
	OverallRiskScore float64       `json:"overallRiskScore"`
	RiskGrade       string         `json:"riskGrade"`
	RiskFactors     []RiskFactor   `json:"riskFactors"`
	Recommendations []string       `json:"recommendations,omitempty"`
}

// RiskFactor represents an individual risk factor
type RiskFactor struct {
	Type        string  `json:"type"`
	Severity    string  `json:"severity"`
	Score       float64 `json:"score"`
	Description string  `json:"description"`
}

// PerformanceAnalysis represents performance analysis results
type PerformanceAnalysis struct {
	AnalysisID       string              `json:"analysisId"`
	OverallScore     float64             `json:"overallScore"`
	ResponseTime     *PerformanceMetric  `json:"responseTime,omitempty"`
	Throughput       *PerformanceMetric  `json:"throughput,omitempty"`
	ResourceUsage    *ResourceMetrics    `json:"resourceUsage,omitempty"`
	Bottlenecks      []string            `json:"bottlenecks,omitempty"`
}

// PerformanceMetric represents a performance metric
type PerformanceMetric struct {
	Current   float64 `json:"current"`
	Target    float64 `json:"target"`
	Unit      string  `json:"unit"`
	Status    string  `json:"status"` // good, warning, critical
}

// ResourceMetrics represents resource usage metrics
type ResourceMetrics struct {
	CPU    float64 `json:"cpu"`
	Memory float64 `json:"memory"`
	Disk   float64 `json:"disk"`
	Network float64 `json:"network"`
}


// CriticalIssue represents a critical issue
type CriticalIssue struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Impact      string    `json:"impact"`
	DetectedAt  time.Time `json:"detectedAt"`
	Resolution  string    `json:"resolution,omitempty"`
}

// HealthWarning represents a health warning
type HealthWarning struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Message     string    `json:"message"`
	Severity    string    `json:"severity"`
	DetectedAt  time.Time `json:"detectedAt"`
}

// HealthRecommendation represents a health improvement recommendation
type HealthRecommendation struct {
	ID           string  `json:"id"`
	Type         string  `json:"type"`
	Priority     string  `json:"priority"`
	Description  string  `json:"description"`
	ActionItems  []string `json:"actionItems"`
	Benefits     []string `json:"benefits,omitempty"`
	Actions      []string `json:"actions,omitempty"`
	EstimatedImpact float64 `json:"estimatedImpact"`
}

// VersionOptimization represents a version optimization recommendation
type VersionOptimization struct {
	PackageName     string `json:"packageName"`
	CurrentVersion  string `json:"currentVersion"`
	RecommendedVersion string `json:"recommendedVersion"`
	Reason          string `json:"reason"`
	Priority        string `json:"priority"`
	Impact          string `json:"impact,omitempty"`
}

// DependencyOptimization represents a dependency optimization recommendation
type DependencyOptimization struct {
	Type            string   `json:"type"` // remove, replace, upgrade
	Description     string   `json:"description"`
	AffectedPackages []string `json:"affectedPackages"`
	Benefit         string   `json:"benefit"`
	Effort          string   `json:"effort"`
}

// DistributionStats represents statistical distribution
type DistributionStats struct {
	Mean       float64 `json:"mean"`
	Median     float64 `json:"median"`
	Mode       float64 `json:"mode"`
	StdDev     float64 `json:"stdDev"`
	Min        float64 `json:"min"`
	Max        float64 `json:"max"`
	Percentiles map[string]float64 `json:"percentiles,omitempty"`
}

// More analysis types

// UpgradeRecommendation represents an upgrade recommendation
type UpgradeRecommendation struct {
	PackageName     string `json:"packageName"`
	CurrentVersion  string `json:"currentVersion"`
	RecommendedVersion string `json:"recommendedVersion"`
	Reason          string `json:"reason"`
	Priority        string `json:"priority"`
	Benefits        []string `json:"benefits,omitempty"`
	Risks           []string `json:"risks,omitempty"`
}

// ReplacementSuggestion represents a package replacement suggestion
type ReplacementSuggestion struct {
	OriginalPackage string `json:"originalPackage"`
	SuggestedPackage string `json:"suggestedPackage"`
	Reason          string `json:"reason"`
	Benefits        []string `json:"benefits,omitempty"`
	MigrationEffort string `json:"migrationEffort"`
}

// IssuePrediction represents predicted issues
type IssuePrediction struct {
	Type            string    `json:"type"`
	Probability     float64   `json:"probability"`
	Severity        string    `json:"severity"`
	Description     string    `json:"description"`
	PredictedDate   time.Time `json:"predictedDate,omitempty"`
	PreventionSteps []string  `json:"preventionSteps,omitempty"`
}

// SecurityOptimization represents security optimization recommendations
type SecurityOptimization struct {
	Type            string   `json:"type"`
	Description     string   `json:"description"`
	AffectedPackages []string `json:"affectedPackages"`
	Severity        string   `json:"severity"`
	Action          string   `json:"action"`
	Priority        string   `json:"priority"`
}

// PerformanceOptimization represents performance optimization recommendations
type PerformanceOptimization struct {
	Type            string   `json:"type"`
	Description     string   `json:"description"`
	AffectedPackages []string `json:"affectedPackages"`
	ExpectedImprovement string `json:"expectedImprovement"`
	ImplementationEffort string `json:"implementationEffort"`
}

// OptimizationAction represents an optimization action
type OptimizationAction struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Priority    string    `json:"priority"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	DueDate     time.Time `json:"dueDate,omitempty"`
}

// OptimizationBenefits represents expected benefits from optimizations
type OptimizationBenefits struct {
	CostSavings       *Cost   `json:"costSavings,omitempty"`
	PerformanceGains  float64 `json:"performanceGains,omitempty"`
	SecurityImprovements []string `json:"securityImprovements,omitempty"`
	MaintenanceReduction string `json:"maintenanceReduction,omitempty"`
}

// MLRecommendation represents ML-driven recommendations
type MLRecommendation struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Confidence  float64 `json:"confidence"`
	ModelVersion string `json:"modelVersion"`
	Evidence    []string `json:"evidence,omitempty"`
}

// String methods for debugging
func (tr *TimeRange) String() string {
	return fmt.Sprintf("TimeRange{Start: %v, End: %v}", tr.Start, tr.End)
}

func (ar *AnalysisReport) String() string {
	return fmt.Sprintf("AnalysisReport{ID: %s, Type: %s, Status: %s}", ar.ID, ar.Type, ar.Status)
}

// Missing types for analyzer.go

// TrendAnalysis represents trend analysis results over time
type TrendAnalysis struct {
	AnalysisID    string                 `json:"analysisId"`
	TimeRange     *TimeRange             `json:"timeRange"`
	Packages      []string               `json:"packages,omitempty"`
	Period        string                 `json:"period,omitempty"`
	Trends        []*Trend               `json:"trends"`
	OverallTrend  TrendDirection         `json:"overallTrend"` 
	Confidence    float64                `json:"confidence"`
	Predictions   []*TrendPrediction     `json:"predictions,omitempty"`
	Anomalies     []*TrendAnomaly        `json:"anomalies,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Trend represents a specific trend
type Trend struct {
	Type       TrendType      `json:"type"`
	Direction  TrendDirection `json:"direction"`
	Strength   float64        `json:"strength"`
	Slope      float64        `json:"slope"`
	DataPoints []*TrendPoint  `json:"dataPoints"`
}

// TrendPoint represents a data point in a trend
type TrendPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// TrendPrediction represents a future trend prediction
type TrendPrediction struct {
	TargetDate   time.Time `json:"targetDate"`
	PredictedValue float64 `json:"predictedValue"`
	Confidence   float64   `json:"confidence"`
	Range        *ValueRange `json:"range,omitempty"`
}

// TrendAnomaly represents an anomaly in trend data
type TrendAnomaly struct {
	Timestamp   time.Time `json:"timestamp"`
	ExpectedValue float64 `json:"expectedValue"`
	ActualValue float64   `json:"actualValue"`
	Deviation   float64   `json:"deviation"`
	Severity    string    `json:"severity"`
}

// ValueRange represents a range of values
type ValueRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// TrendDirection defines trend directions
type TrendDirection string

const (
	TrendDirectionUp     TrendDirection = "up"
	TrendDirectionDown   TrendDirection = "down"
	TrendDirectionStable TrendDirection = "stable"
	TrendDirectionVolatile TrendDirection = "volatile"
)

// TrendType defines types of trends
type TrendType string

const (
	TrendTypeUsage       TrendType = "usage"
	TrendTypeCost        TrendType = "cost"
	TrendTypePerformance TrendType = "performance"
	TrendTypeSecurity    TrendType = "security"
	TrendTypeHealth      TrendType = "health"
)

// AnomalyDetectionResult represents anomaly detection results
type AnomalyDetectionResult struct {
	DetectionID   string                 `json:"detectionId"`
	Anomalies     []*Anomaly             `json:"anomalies"`
	TotalAnomalies int                   `json:"totalAnomalies"`
	SeverityBreakdown map[string]int      `json:"severityBreakdown"`
	DetectionModel string                `json:"detectionModel"`
	ConfidenceThreshold float64          `json:"confidenceThreshold"`
	AnalysisTime  time.Duration          `json:"analysisTime"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Anomaly represents a detected anomaly
type Anomaly struct {
	ID           string    `json:"id"`
	Type         AnomalyType `json:"type"`
	Severity     string    `json:"severity"`
	Description  string    `json:"description"`
	Timestamp    time.Time `json:"timestamp"`
	Value        float64   `json:"value"`
	ExpectedValue float64  `json:"expectedValue"`
	Confidence   float64   `json:"confidence"`
	Package      *PackageReference `json:"package,omitempty"`
	Context      map[string]interface{} `json:"context,omitempty"`
}

// AnomalyType defines types of anomalies
type AnomalyType string

const (
	AnomalyTypeUsageSpike      AnomalyType = "usage_spike"
	AnomalyTypeUsageDrop       AnomalyType = "usage_drop"
	AnomalyTypeCostIncrease    AnomalyType = "cost_increase"
	AnomalyTypePerformanceDrop AnomalyType = "performance_drop"
	AnomalyTypeSecurityAlert   AnomalyType = "security_alert"
)

// AnalysisError represents an analysis error
type AnalysisError struct {
	Code        string                 `json:"code"`
	Type        AnalysisErrorType      `json:"type"`
	Severity    ErrorSeverity          `json:"severity"`
	Message     string                 `json:"message"`
	Package     *PackageReference      `json:"package,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	StackTrace  string                 `json:"stackTrace,omitempty"`
}

// AnalysisWarning represents an analysis warning
type AnalysisWarning struct {
	Code        string                 `json:"code"`
	Type        AnalysisWarningType    `json:"type"`
	Message     string                 `json:"message"`
	Package     *PackageReference      `json:"package,omitempty"`
	Recommendation string              `json:"recommendation,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// AnalysisErrorType defines types of analysis errors
type AnalysisErrorType string

const (
	AnalysisErrorTypeData         AnalysisErrorType = "data"
	AnalysisErrorTypeValidation   AnalysisErrorType = "validation"
	AnalysisErrorTypeProcessing   AnalysisErrorType = "processing"
	AnalysisErrorTypeIntegration  AnalysisErrorType = "integration"
	AnalysisErrorTypeConfiguration AnalysisErrorType = "configuration"
)

// AnalysisWarningType defines types of analysis warnings
type AnalysisWarningType string

const (
	AnalysisWarningTypeData        AnalysisWarningType = "data"
	AnalysisWarningTypePerformance AnalysisWarningType = "performance"
	AnalysisWarningTypeQuality     AnalysisWarningType = "quality"
	AnalysisWarningTypeSecurity    AnalysisWarningType = "security"
	AnalysisWarningTypeCompliance  AnalysisWarningType = "compliance"
)

// AnalysisStatistics contains comprehensive analysis statistics
type AnalysisStatistics struct {
	TotalPackagesAnalyzed    int           `json:"totalPackagesAnalyzed"`
	SuccessfulAnalyses       int           `json:"successfulAnalyses"`
	FailedAnalyses          int           `json:"failedAnalyses"`
	TotalExecutionTime      time.Duration `json:"totalExecutionTime"`
	AverageExecutionTime    time.Duration `json:"averageExecutionTime"`
	CacheHits               int           `json:"cacheHits"`
	CacheMisses             int           `json:"cacheMisses"`
	DataPointsProcessed     int64         `json:"dataPointsProcessed"`
	ModelInferences         int           `json:"modelInferences"`
	ResourceUtilization     *ResourceUtilizationStats `json:"resourceUtilization,omitempty"`
}

// ResourceUtilizationStats contains resource utilization statistics
type ResourceUtilizationStats struct {
	CPUUsage    float64 `json:"cpuUsage"`
	MemoryUsage float64 `json:"memoryUsage"`
	DiskUsage   float64 `json:"diskUsage"`
	NetworkIO   float64 `json:"networkIO"`
}

// PackageUsageMetrics contains detailed usage metrics for a package
type PackageUsageMetrics struct {
	PackageName         string            `json:"packageName"`
	TotalUsage          int64             `json:"totalUsage"`
	UniqueUsers         int               `json:"uniqueUsers"`
	UsageFrequency      float64           `json:"usageFrequency"`
	UsageByVersion      map[string]int64  `json:"usageByVersion"`
	UsageByEnvironment  map[string]int64  `json:"usageByEnvironment"`
	UsageByRegion       map[string]int64  `json:"usageByRegion,omitempty"`
	PeakUsage           int64             `json:"peakUsage"`
	AverageUsage        float64           `json:"averageUsage"`
	UsageGrowthRate     float64           `json:"usageGrowthRate"`
	SeasonalPatterns    []*SeasonalPattern `json:"seasonalPatterns,omitempty"`
	LastUsed            time.Time         `json:"lastUsed"`
	UsageScore          float64           `json:"usageScore"`
}

// HealthScore represents comprehensive health scoring for a package
type HealthScore struct {
	Package             *PackageReference   `json:"package"`
	OverallScore        float64             `json:"overallScore"`
	Grade               HealthGrade         `json:"grade"`
	SecurityScore       float64             `json:"securityScore"`
	MaintenanceScore    float64             `json:"maintenanceScore"`
	QualityScore        float64             `json:"qualityScore"`
	PerformanceScore    float64             `json:"performanceScore"`
	CommunityScore      float64             `json:"communityScore"`
	DocumentationScore  float64             `json:"documentationScore"`
	LicenseScore        float64             `json:"licenseScore"`
	CriticalIssues      []*CriticalIssue    `json:"criticalIssues,omitempty"`
	Warnings            []*HealthWarning    `json:"warnings,omitempty"`
	Recommendations     []*HealthRecommendation `json:"recommendations,omitempty"`
	Trend               HealthTrend         `json:"trend"`
	LastAssessed        time.Time           `json:"lastAssessed"`
	AssessmentVersion   string              `json:"assessmentVersion"`
}

// QualityMetrics contains quality-related metrics
type QualityMetrics struct {
	CodeQuality         float64           `json:"codeQuality"`
	TestCoverage        float64           `json:"testCoverage"`
	DocumentationCoverage float64         `json:"documentationCoverage"`
	BugReports          int               `json:"bugReports"`
	OpenIssues          int               `json:"openIssues"`
	Contributors        int               `json:"contributors"`
	CommitFrequency     float64           `json:"commitFrequency"`
	ReleaseFrequency    float64           `json:"releaseFrequency"`
	SecurityIssues      int               `json:"securityIssues"`
	LintingScore        float64           `json:"lintingScore"`
	ComplexityScore     float64           `json:"complexityScore"`
	MaintainabilityIndex float64          `json:"maintainabilityIndex"`
	TechnicalDebt       float64           `json:"technicalDebt"`
	BestPracticesScore  float64           `json:"bestPracticesScore"`
}

// PerformanceMetrics contains performance-related metrics
type PerformanceMetrics struct {
	ResponseTime        float64           `json:"responseTime"`
	Throughput          float64           `json:"throughput"`
	Latency             float64           `json:"latency"`
	MemoryUsage         float64           `json:"memoryUsage"`
	CPUUsage            float64           `json:"cpuUsage"`
	DiskIO              float64           `json:"diskIO"`
	NetworkIO           float64           `json:"networkIO"`
	StartupTime         float64           `json:"startupTime"`
	ResourceEfficiency  float64           `json:"resourceEfficiency"`
	ScalabilityScore    float64           `json:"scalabilityScore"`
	Bottlenecks         []string          `json:"bottlenecks,omitempty"`
	PerformanceGrade    string            `json:"performanceGrade"`
	BenchmarkResults    []*BenchmarkResult `json:"benchmarkResults,omitempty"`
}

// BenchmarkResult contains benchmark test results
type BenchmarkResult struct {
	Name            string        `json:"name"`
	Iterations      int64         `json:"iterations"`
	Duration        time.Duration `json:"duration"`
	NsPerOperation  int64         `json:"nsPerOperation"`
	BytesPerOperation int64       `json:"bytesPerOperation"`
	AllocsPerOperation int64      `json:"allocsPerOperation"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// AffectedPackage represents a package affected by a vulnerability or issue
type AffectedPackage struct {
	Package         *PackageReference `json:"package"`
	VersionRange    *VersionRange     `json:"versionRange"`
	Severity        string            `json:"severity"`
	FixedInVersion  string            `json:"fixedInVersion,omitempty"`
	Workaround      string            `json:"workaround,omitempty"`
	ImpactScore     float64           `json:"impactScore"`
}

// Missing types from validator.go

// ErrorSeverity defines error severity levels
type ErrorSeverity string

const (
	ErrorSeverityLow      ErrorSeverity = "low"
	ErrorSeverityMedium   ErrorSeverity = "medium"
	ErrorSeverityHigh     ErrorSeverity = "high"
	ErrorSeverityCritical ErrorSeverity = "critical"
)

// WarningType defines types of warnings
type WarningType string

const (
	WarningTypeCompatibility WarningType = "compatibility"
	WarningTypeSecurity      WarningType = "security"
	WarningTypeLicense       WarningType = "license"
	WarningTypePerformance   WarningType = "performance"
	WarningTypeQuality       WarningType = "quality"
	WarningTypeCompliance    WarningType = "compliance"
)

// WarningImpact defines warning impact levels
type WarningImpact string

const (
	WarningImpactLow      WarningImpact = "low"
	WarningImpactMedium   WarningImpact = "medium"
	WarningImpactHigh     WarningImpact = "high"
	WarningImpactCritical WarningImpact = "critical"
)

// ConflictImpact defines conflict impact levels
type ConflictImpact string

const (
	ConflictImpactLow      ConflictImpact = "low"
	ConflictImpactMedium   ConflictImpact = "medium"
	ConflictImpactHigh     ConflictImpact = "high"
	ConflictImpactCritical ConflictImpact = "critical"
)

// Additional missing types for analyzer.go and validator.go

// OptimizationObjectives defines optimization objectives
type OptimizationObjectives struct {
	CostOptimization        bool    `json:"costOptimization"`
	PerformanceOptimization bool    `json:"performanceOptimization"`
	SecurityOptimization    bool    `json:"securityOptimization"`
	QualityOptimization     bool    `json:"qualityOptimization"`
	EfficiencyOptimization  bool    `json:"efficiencyOptimization"`
	WeightedObjectives     map[string]float64 `json:"weightedObjectives,omitempty"`
}

// CostConstraints defines cost constraints for optimization
type CostConstraints struct {
	MaxTotalCost      float64           `json:"maxTotalCost,omitempty"`
	MaxCostPerPackage float64           `json:"maxCostPerPackage,omitempty"`
	CostCategories    map[string]float64 `json:"costCategories,omitempty"`
	Currency          string            `json:"currency"`
}

// CostReport contains comprehensive cost reporting
type CostReport struct {
	ReportID        string            `json:"reportId"`
	Scope           *CostReportScope  `json:"scope"`
	TotalCost       *Cost             `json:"totalCost"`
	CostBreakdown   *CostBreakdown    `json:"costBreakdown"`
	CostTrends      []*CostTrend      `json:"costTrends,omitempty"`
	Optimizations   []*CostOptimization `json:"optimizations,omitempty"`
	GeneratedAt     time.Time         `json:"generatedAt"`
}

// CostReportScope defines the scope for cost reporting
type CostReportScope struct {
	Packages      []*PackageReference `json:"packages"`
	TimeRange     *TimeRange          `json:"timeRange,omitempty"`
	Environment   string              `json:"environment,omitempty"`
	Categories    []string            `json:"categories,omitempty"`
}

// Additional analyzer types

// UnusedDependencyReport contains unused dependency analysis
type UnusedDependencyReport struct {
	ReportID            string              `json:"reportId"`
	AnalyzedPackages    []*PackageReference `json:"analyzedPackages"`
	UnusedDependencies  []*UnusedDependency `json:"unusedDependencies"`
	PotentialSavings    *Cost               `json:"potentialSavings,omitempty"`
	RemovalRecommendations []*RemovalRecommendation `json:"removalRecommendations,omitempty"`
	AnalysisTime        time.Duration       `json:"analysisTime"`
	GeneratedAt         time.Time           `json:"generatedAt"`
}

// UnusedDependency represents an unused dependency
type UnusedDependency struct {
	Package         *PackageReference `json:"package"`
	LastUsed        time.Time         `json:"lastUsed,omitempty"`
	DaysUnused      int               `json:"daysUnused"`
	SizeOnDisk      int64             `json:"sizeOnDisk,omitempty"`
	Cost            *Cost             `json:"cost,omitempty"`
	RemovalRisk     string            `json:"removalRisk"`
	Dependencies    []*PackageReference `json:"dependencies,omitempty"`
}

// RemovalRecommendation represents a recommendation to remove a dependency
type RemovalRecommendation struct {
	Package         *PackageReference `json:"package"`
	Reason          string            `json:"reason"`
	Priority        string            `json:"priority"`
	Risk            string            `json:"risk"`
	PotentialSavings *Cost            `json:"potentialSavings,omitempty"`
	Steps           []string          `json:"steps,omitempty"`
}

// SecurityRiskAssessment contains security risk assessment results
type SecurityRiskAssessment struct {
	AssessmentID    string              `json:"assessmentId"`
	Packages        []*PackageReference `json:"packages"`
	RiskLevel       RiskLevel           `json:"riskLevel"`
	RiskScore       float64             `json:"riskScore"`
	SecurityIssues  []*SecurityIssue    `json:"securityIssues"`
	RiskFactors     []*RiskFactor       `json:"riskFactors"`
	Recommendations []*SecurityRecommendation `json:"recommendations,omitempty"`
	AssessedAt      time.Time           `json:"assessedAt"`
}

// SecurityRecommendation represents a security recommendation
type SecurityRecommendation struct {
	ID              string    `json:"id"`
	Type            string    `json:"type"`
	Priority        string    `json:"priority"`
	Description     string    `json:"description"`
	AffectedPackages []*PackageReference `json:"affectedPackages"`
	Actions         []string  `json:"actions"`
	Timeline        string    `json:"timeline,omitempty"`
}

// ComplianceRiskAnalysis contains compliance risk analysis results
type ComplianceRiskAnalysis struct {
	AnalysisID        string              `json:"analysisId"`
	Packages          []*PackageReference `json:"packages"`
	ComplianceStatus  ComplianceStatus    `json:"complianceStatus"`
	RiskLevel         RiskLevel           `json:"riskLevel"`
	ComplianceIssues  []*ComplianceIssue  `json:"complianceIssues"`
	RequiredActions   []*ComplianceAction `json:"requiredActions,omitempty"`
	AnalyzedAt        time.Time           `json:"analyzedAt"`
}

// ComplianceIssue represents a compliance issue
type ComplianceIssue struct {
	ID              string            `json:"id"`
	Type            string            `json:"type"`
	Severity        string            `json:"severity"`
	Description     string            `json:"description"`
	Package         *PackageReference `json:"package,omitempty"`
	Regulation      string            `json:"regulation,omitempty"`
	RequiredAction  string            `json:"requiredAction"`
}

// ComplianceAction represents a required compliance action
type ComplianceAction struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Priority    string    `json:"priority"`
	Deadline    time.Time `json:"deadline,omitempty"`
	Status      string    `json:"status"`
}

// OptimizedGraph represents an optimized dependency graph
type OptimizedGraph struct {
	OriginalGraph   *DependencyGraph    `json:"originalGraph"`
	OptimizedGraph  *DependencyGraph    `json:"optimizedGraph"`
	Optimizations   []*GraphOptimization `json:"optimizations"`
	Metrics         *OptimizationMetrics `json:"metrics"`
	Strategy        OptimizationStrategy `json:"strategy"`
	OptimizedAt     time.Time           `json:"optimizedAt"`
}

// GraphOptimization represents a graph optimization
type GraphOptimization struct {
	Type            string            `json:"type"`
	Description     string            `json:"description"`
	AffectedNodes   []string          `json:"affectedNodes"`
	Benefit         string            `json:"benefit"`
	Impact          string            `json:"impact"`
}

// OptimizationMetrics contains optimization metrics
type OptimizationMetrics struct {
	NodesReduced       int     `json:"nodesReduced"`
	EdgesReduced       int     `json:"edgesReduced"`
	ComplexityReduced  float64 `json:"complexityReduced"`
	PerformanceGain    float64 `json:"performanceGain"`
	CostReduction      *Cost   `json:"costReduction,omitempty"`
}

// ReplacementCriteria defines criteria for package replacement suggestions
type ReplacementCriteria struct {
	MinPopularityScore  float64   `json:"minPopularityScore,omitempty"`
	MaxAge              time.Duration `json:"maxAge,omitempty"`
	RequiredLicenses    []string  `json:"requiredLicenses,omitempty"`
	ExcludedLicenses    []string  `json:"excludedLicenses,omitempty"`
	MinSecurityScore    float64   `json:"minSecurityScore,omitempty"`
	RequiredFeatures    []string  `json:"requiredFeatures,omitempty"`
	PerformanceRequirements map[string]float64 `json:"performanceRequirements,omitempty"`
}

// ResourceUsageAnalysis contains resource usage analysis results
type ResourceUsageAnalysis struct {
	AnalysisID      string              `json:"analysisId"`
	Packages        []*PackageReference `json:"packages"`
	TotalUsage      *ResourceUsage      `json:"totalUsage"`
	UsageByPackage  map[string]*ResourceUsage `json:"usageByPackage"`
	Trends          []*ResourceUsageTrend `json:"trends,omitempty"`
	Recommendations []*ResourceOptimization `json:"recommendations,omitempty"`
	AnalyzedAt      time.Time           `json:"analyzedAt"`
}

// ResourceUsage represents resource usage metrics
type ResourceUsage struct {
	CPU     float64 `json:"cpu"`
	Memory  int64   `json:"memory"`
	Disk    int64   `json:"disk"`
	Network int64   `json:"network"`
	GPU     float64 `json:"gpu,omitempty"`
}

// ResourceUsageTrend represents resource usage trends
type ResourceUsageTrend struct {
	ResourceType string        `json:"resourceType"`
	Trend        TrendDirection `json:"trend"`
	GrowthRate   float64       `json:"growthRate"`
	DataPoints   []*ResourceDataPoint `json:"dataPoints"`
}

// ResourceDataPoint represents a resource usage data point
type ResourceDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
}

// ResourceOptimization represents resource optimization recommendations
type ResourceOptimization struct {
	Type                 string    `json:"type"`
	Description          string    `json:"description"`
	AffectedPackages     []*PackageReference `json:"affectedPackages"`
	ExpectedSavings      *ResourceUsage `json:"expectedSavings"`
	ImplementationEffort string    `json:"implementationEffort"`
	Priority            string    `json:"priority"`
}

// EvolutionAnalysis contains package evolution analysis
type EvolutionAnalysis struct {
	AnalysisID        string              `json:"analysisId"`
	Package           *PackageReference   `json:"package"`
	VersionHistory    []*VersionInfo      `json:"versionHistory"`
	EvolutionTrends   []*EvolutionTrend   `json:"evolutionTrends"`
	BreakingChanges   []*BreakingChange   `json:"breakingChanges,omitempty"`
	Predictions       []*EvolutionPrediction `json:"predictions,omitempty"`
	AnalyzedAt        time.Time           `json:"analyzedAt"`
}

// VersionInfo contains information about a specific version
type VersionInfo struct {
	Version         string            `json:"version"`
	ReleaseDate     time.Time         `json:"releaseDate"`
	Changes         []string          `json:"changes,omitempty"`
	BreakingChanges []string          `json:"breakingChanges,omitempty"`
	SecurityFixes   []string          `json:"securityFixes,omitempty"`
	Stability       string            `json:"stability"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// EvolutionTrend represents evolution trends for a package
type EvolutionTrend struct {
	Aspect      string         `json:"aspect"`
	Direction   TrendDirection `json:"direction"`
	Confidence  float64        `json:"confidence"`
	Description string         `json:"description"`
}

// BreakingChange represents a breaking change in package evolution
type BreakingChange struct {
	Version     string    `json:"version"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Impact      string    `json:"impact"`
	Mitigation  string    `json:"mitigation,omitempty"`
	AffectedAPIs []string `json:"affectedAPIs,omitempty"`
}

// EvolutionPrediction represents predictions about package evolution
type EvolutionPrediction struct {
	Aspect          string    `json:"aspect"`
	PredictedValue  string    `json:"predictedValue"`
	Confidence      float64   `json:"confidence"`
	TimeHorizon     time.Duration `json:"timeHorizon"`
	BasedOnFactors  []string  `json:"basedOnFactors,omitempty"`
}

// AnalyzerHealth represents analyzer health status
type AnalyzerHealth struct {
	Status          string            `json:"status"`
	Uptime          time.Duration     `json:"uptime"`
	ComponentsHealthy int             `json:"componentsHealthy"`
	ComponentsTotal   int             `json:"componentsTotal"`
	Components      map[string]string `json:"components"`
	LastHealthCheck time.Time         `json:"lastHealthCheck"`
	Issues          []HealthIssue     `json:"issues,omitempty"`
	Metrics         map[string]float64 `json:"metrics,omitempty"`
}

// HealthIssue represents a health issue
type HealthIssue struct {
	Severity    string `json:"severity"`
	Component   string `json:"component"`
	Description string `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
}

// AnalysisModels contains machine learning models for analysis
type AnalysisModels struct {
	Version             string     `json:"version"`
	PredictionModel     *ModelInfo `json:"predictionModel,omitempty"`
	RecommendationModel *ModelInfo `json:"recommendationModel,omitempty"`
	AnomalyModel        *ModelInfo `json:"anomalyModel,omitempty"`
	OptimizationModel   *ModelInfo `json:"optimizationModel,omitempty"`
}

// ModelInfo contains information about a machine learning model
type ModelInfo struct {
	Name            string    `json:"name"`
	Version         string    `json:"version"`
	Type            string    `json:"type"`
	Accuracy        float64   `json:"accuracy,omitempty"`
	LastTrained     time.Time `json:"lastTrained"`
	TrainingDataSize int64    `json:"trainingDataSize"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// Missing interface types that are referenced

// Alert represents a monitoring alert
type Alert struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Message     string                 `json:"message"`
	Source      string                 `json:"source"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Additional missing types from analyzer.go

// ResourceUsageMetrics contains resource usage metrics for a package
type ResourceUsageMetrics struct {
	PackageName     string    `json:"packageName"`
	CPUUsage        float64   `json:"cpuUsage"`
	MemoryUsage     int64     `json:"memoryUsage"`
	DiskUsage       int64     `json:"diskUsage"`
	NetworkUsage    int64     `json:"networkUsage"`
	GPUUsage        float64   `json:"gpuUsage,omitempty"`
	MaxCPU          float64   `json:"maxCPU"`
	MaxMemory       int64     `json:"maxMemory"`
	AverageUsage    float64   `json:"averageUsage"`
	UsageEfficiency float64   `json:"usageEfficiency"`
	ResourceScore   float64   `json:"resourceScore"`
	Bottlenecks     []string  `json:"bottlenecks,omitempty"`
	Recommendations []string  `json:"recommendations,omitempty"`
}

// CostMetrics contains cost metrics for a package
type CostMetrics struct {
	PackageName         string            `json:"packageName"`
	TotalCost           float64           `json:"totalCost"`
	LicenseCost         float64           `json:"licenseCost"`
	SupportCost         float64           `json:"supportCost"`
	InfrastructureCost  float64           `json:"infrastructureCost"`
	OperationalCost     float64           `json:"operationalCost"`
	CostPerUser         float64           `json:"costPerUser,omitempty"`
	CostPerTransaction  float64           `json:"costPerTransaction,omitempty"`
	CostEfficiency      float64           `json:"costEfficiency"`
	Currency            string            `json:"currency"`
	BillingPeriod       string            `json:"billingPeriod,omitempty"`
	CostTrendDirection  string            `json:"costTrendDirection,omitempty"`
	ProjectedCost       float64           `json:"projectedCost,omitempty"`
}

// RecommendedAction represents a recommended action for a package
type RecommendedAction struct {
	ID          string            `json:"id"`
	Type        ActionType        `json:"type"`
	Priority    ActionPriority    `json:"priority"`
	Description string            `json:"description"`
	Package     *PackageReference `json:"package,omitempty"`
	Reason      string            `json:"reason"`
	Benefits    []string          `json:"benefits,omitempty"`
	Risks       []string          `json:"risks,omitempty"`
	Steps       []string          `json:"steps"`
	Effort      ActionEffort      `json:"effort"`
	Timeline    string            `json:"timeline,omitempty"`
	Impact      ActionImpact      `json:"impact"`
	Status      ActionStatus      `json:"status,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
	DueDate     time.Time         `json:"dueDate,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// PredictedIssue represents a predicted issue for a package
type PredictedIssue struct {
	ID          string            `json:"id"`
	Type        IssueType         `json:"type"`
	Severity    IssueSeverity     `json:"severity"`
	Description string            `json:"description"`
	Package     *PackageReference `json:"package,omitempty"`
	Probability float64           `json:"probability"`
	Confidence  float64           `json:"confidence"`
	TimeFrame   time.Duration     `json:"timeFrame"`
	Impact      IssueImpact       `json:"impact"`
	Mitigation  []string          `json:"mitigation,omitempty"`
	Indicators  []string          `json:"indicators,omitempty"`
	PredictedAt time.Time         `json:"predictedAt"`
	Model       string            `json:"model,omitempty"`
}

// Action-related enums

// ActionType defines types of recommended actions
type ActionType string

const (
	ActionTypeUpgrade    ActionType = "upgrade"
	ActionTypeDowngrade  ActionType = "downgrade"
	ActionTypeReplace    ActionType = "replace"
	ActionTypeRemove     ActionType = "remove"
	ActionTypeAdd        ActionType = "add"
	ActionTypeConfigure  ActionType = "configure"
	ActionTypeMonitor    ActionType = "monitor"
	ActionTypeReview     ActionType = "review"
)

// ActionPriority defines priority levels for actions
type ActionPriority string

const (
	ActionPriorityLow      ActionPriority = "low"
	ActionPriorityMedium   ActionPriority = "medium"
	ActionPriorityHigh     ActionPriority = "high"
	ActionPritorityCritical ActionPriority = "critical"
)

// ActionEffort defines effort levels for actions
type ActionEffort string

const (
	ActionEffortMinimal    ActionEffort = "minimal"
	ActionEffortLow        ActionEffort = "low"
	ActionEffortMedium     ActionEffort = "medium"
	ActionEffortHigh       ActionEffort = "high"
	ActionEffortSignificant ActionEffort = "significant"
)

// ActionImpact defines impact levels for actions
type ActionImpact string

const (
	ActionImpactMinimal ActionImpact = "minimal"
	ActionImpactLow     ActionImpact = "low"
	ActionImpactMedium  ActionImpact = "medium"
	ActionImpactHigh    ActionImpact = "high"
	ActionImpactMajor   ActionImpact = "major"
)

// ActionStatus defines status of actions
type ActionStatus string

const (
	ActionStatusPending     ActionStatus = "pending"
	ActionStatusInProgress  ActionStatus = "in_progress"
	ActionStatusCompleted   ActionStatus = "completed"
	ActionStatusFailed      ActionStatus = "failed"
	ActionStatusCancelled   ActionStatus = "cancelled"
	ActionStatusOnHold      ActionStatus = "on_hold"
)

// Issue-related enums

// IssueType defines types of predicted issues
type IssueType string

const (
	IssueTypeSecurity     IssueType = "security"
	IssueTypePerformance  IssueType = "performance"
	IssueTypeCompatibility IssueType = "compatibility"
	IssueTypeMaintenance  IssueType = "maintenance"
	IssueTypeLicense      IssueType = "license"
	IssueTypeCompliance   IssueType = "compliance"
	IssueTypeDependency   IssueType = "dependency"
	IssueTypeObsolescence IssueType = "obsolescence"
)

// IssueSeverity defines severity levels for issues
type IssueSeverity string

const (
	IssueSeverityInfo     IssueSeverity = "info"
	IssueSeverityLow      IssueSeverity = "low"
	IssueSeverityMedium   IssueSeverity = "medium"
	IssueSeverityHigh     IssueSeverity = "high"
	IssueSeverityCritical IssueSeverity = "critical"
)

// IssueImpact defines impact levels for issues
type IssueImpact string

const (
	IssueImpactMinimal IssueImpact = "minimal"
	IssueImpactLow     IssueImpact = "low"
	IssueImpactMedium  IssueImpact = "medium"
	IssueImpactHigh    IssueImpact = "high"
	IssueImpactSevere  IssueImpact = "severe"
)

// Additional validator types

// HealthTrendAnalysis contains health trend analysis results
type HealthTrendAnalysis struct {
	AnalysisID    string              `json:"analysisId"`
	Packages      []*PackageReference `json:"packages"`
	OverallTrend  HealthTrend         `json:"overallTrend"`
	TrendStrength float64             `json:"trendStrength"`
	Trends        []*PackageHealthTrend `json:"trends"`
	Predictions   []*HealthPrediction `json:"predictions,omitempty"`
	TrendConfidence  float64                   `json:"trendConfidence"`
	TrendPredictions []*HealthTrendPrediction  `json:"trendPredictions,omitempty"`
	HealthSnapshots  []*HealthSnapshot         `json:"healthSnapshots,omitempty"`
	Recommendations  []*HealthRecommendation   `json:"recommendations,omitempty"`
	AnalyzedAt    time.Time           `json:"analyzedAt"`
}

// PackageHealthTrend represents health trend for a specific package
type PackageHealthTrend struct {
	Package       *PackageReference `json:"package"`
	Trend         HealthTrend       `json:"trend"`
	CurrentScore  float64           `json:"currentScore"`
	PreviousScore float64           `json:"previousScore"`
	ChangeRate    float64           `json:"changeRate"`
	Confidence    float64           `json:"confidence"`
}

// HealthPrediction represents a prediction about package health
type HealthPrediction struct {
	Package         *PackageReference `json:"package"`
	PredictedScore  float64           `json:"predictedScore"`
	PredictedGrade  HealthGrade       `json:"predictedGrade"`
	TimeHorizon     time.Duration     `json:"timeHorizon"`
	Confidence      float64           `json:"confidence"`
	RiskFactors     []string          `json:"riskFactors,omitempty"`
}
// HealthSnapshot represents a snapshot of package health at a point in time
type HealthSnapshot struct {
	PackageName       string    `json:"packageName"`
	Timestamp         time.Time `json:"timestamp"`
	OverallScore      float64   `json:"overallScore"`
	SecurityHealth    float64   `json:"securityHealth"`
	PerformanceHealth float64   `json:"performanceHealth"`
	MaintenanceHealth float64   `json:"maintenanceHealth"`
	QualityHealth     float64   `json:"qualityHealth"`
	Grade             HealthGrade `json:"grade"`
}

// HealthTrendPrediction represents a prediction about health trends
type HealthTrendPrediction struct {
	PackageName      string        `json:"packageName"`
	PredictedTrend   HealthTrend   `json:"predictedTrend"`
	PredictedScore   float64       `json:"predictedScore"`
	TimeHorizon      time.Duration `json:"timeHorizon"`
	Confidence       float64       `json:"confidence"`
	Factors          []string      `json:"factors,omitempty"`
	RiskIndicators   []string      `json:"riskIndicators,omitempty"`
}

// Additional missing types from validator.go

// ValidationRecommendation represents a validation recommendation
type ValidationRecommendation struct {
	ID              string            `json:"id"`
	Type            string            `json:"type"`
	Priority        string            `json:"priority"`
	Description     string            `json:"description"`
	Package         *PackageReference `json:"package,omitempty"`
	Action          string            `json:"action"`
	Benefits        []string          `json:"benefits,omitempty"`
	Risks           []string          `json:"risks,omitempty"`
	EstimatedEffort string            `json:"estimatedEffort"`
	Timeline        string            `json:"timeline,omitempty"`
}

// ValidationStatistics contains validation statistics
type ValidationStatistics struct {
	TotalPackages       int           `json:"totalPackages"`
	ValidatedPackages   int           `json:"validatedPackages"`
	FailedValidations   int           `json:"failedValidations"`
	TotalErrors         int           `json:"totalErrors"`
	TotalWarnings       int           `json:"totalWarnings"`
	AverageScore        float64       `json:"averageScore"`
	ValidationTime      time.Duration `json:"validationTime"`
	CacheHitRate        float64       `json:"cacheHitRate"`
	ThroughputPPS       float64       `json:"throughputPPS"` // Packages per second
}

// Additional types needed for completeness

// AnalyzerConfig represents the main analyzer configuration
type AnalyzerConfig struct {
	Version                     string                      `json:"version"`
	EnableMLAnalysis           bool                        `json:"enableMLAnalysis"`
	EnableMLOptimization       bool                        `json:"enableMLOptimization"`
	EnableCaching              bool                        `json:"enableCaching"`
	EnableConcurrency          bool                        `json:"enableConcurrency"`
	EnableTrendAnalysis        bool                        `json:"enableTrendAnalysis"`
	EnableCostProjection       bool                        `json:"enableCostProjection"`
	WorkerCount                int                         `json:"workerCount"`
	QueueSize                  int                         `json:"queueSize"`
	Currency                   string                      `json:"currency"`
	UsageAnalyzerConfig        *UsageAnalyzerConfig        `json:"usageAnalyzerConfig,omitempty"`
	CostAnalyzerConfig         *CostAnalyzerConfig         `json:"costAnalyzerConfig,omitempty"`
	HealthAnalyzerConfig       *HealthAnalyzerConfig       `json:"healthAnalyzerConfig,omitempty"`
	RiskAnalyzerConfig         *RiskAnalyzerConfig         `json:"riskAnalyzerConfig,omitempty"`
	PerformanceAnalyzerConfig  *PerformanceAnalyzerConfig  `json:"performanceAnalyzerConfig,omitempty"`
	OptimizationEngineConfig   *OptimizationEngineConfig   `json:"optimizationEngineConfig,omitempty"`
	MLOptimizerConfig          *MLOptimizerConfig          `json:"mlOptimizerConfig,omitempty"`
	UsageCollectorConfig       *UsageCollectorConfig       `json:"usageCollectorConfig,omitempty"`
	MetricsCollectorConfig     *MetricsCollectorConfig     `json:"metricsCollectorConfig,omitempty"`
	EventProcessorConfig       *EventProcessorConfig       `json:"eventProcessorConfig,omitempty"`
	PredictionModelConfig      *PredictionModelConfig      `json:"predictionModelConfig,omitempty"`
	RecommendationModelConfig  *RecommendationModelConfig  `json:"recommendationModelConfig,omitempty"`
	AnomalyDetectorConfig      *AnomalyDetectorConfig      `json:"anomalyDetectorConfig,omitempty"`
	AnalysisCacheConfig        *AnalysisCacheConfig        `json:"analysisCacheConfig,omitempty"`
	DataStoreConfig            *DataStoreConfig            `json:"dataStoreConfig,omitempty"`
	PackageRegistryConfig      *PackageRegistryConfig      `json:"packageRegistryConfig,omitempty"`
}

// Additional missing types from various files

// CriticalPath represents a critical path in dependency graph
type CriticalPath struct {
	Path      []*GraphNode `json:"path"`
	Length    int          `json:"length"`
	Weight    float64      `json:"weight"`
	Bottleneck *GraphNode  `json:"bottleneck,omitempty"`
}

// GraphPattern represents patterns in dependency graphs
type GraphPattern struct {
	Type        PatternType   `json:"type"`
	Nodes       []*GraphNode  `json:"nodes"`
	Description string        `json:"description"`
	Frequency   int           `json:"frequency"`
	Impact      PatternImpact `json:"impact"`
}

// PatternType defines types of graph patterns
type PatternType string

const (
	PatternTypeCircular    PatternType = "circular"
	PatternTypeFanOut      PatternType = "fan_out"
	PatternTypeFanIn       PatternType = "fan_in"
	PatternTypeChain       PatternType = "chain"
	PatternTypeHub         PatternType = "hub"
	PatternTypeIsolated    PatternType = "isolated"
)

// PatternAnalysis represents the result of pattern detection analysis
type PatternAnalysis struct {
	AnalysisID         string         `json:"analysisId"`
	PatternsDetected   []*GraphPattern `json:"patternsDetected"`
	TotalPatterns      int            `json:"totalPatterns"`
	PatternsByType     map[PatternType]int `json:"patternsByType"`
	AnalyzedAt         time.Time      `json:"analyzedAt"`
	AnalysisTime       time.Duration  `json:"analysisTime"`
	Recommendations    []string       `json:"recommendations,omitempty"`
}

// PatternImpact defines impact levels of patterns
type PatternImpact string

const (
	PatternImpactLow     PatternImpact = "low"
	PatternImpactMedium  PatternImpact = "medium"
	PatternImpactHigh    PatternImpact = "high"
	PatternImpactCritical PatternImpact = "critical"
)

// CycleImpact defines impact levels of dependency cycles
type CycleImpact string

const (
	CycleImpactLow     CycleImpact = "low"
	CycleImpactMedium  CycleImpact = "medium"
	CycleImpactHigh    CycleImpact = "high"
	CycleImpactSevere  CycleImpact = "severe"
)

// CycleBreakingOption represents options for breaking dependency cycles
type CycleBreakingOption struct {
	Type            BreakingOptionType `json:"type"`
	Target          *GraphNode         `json:"target"`
	Description     string             `json:"description"`
	Cost            float64            `json:"cost"`
	Risk            BreakingRiskLevel  `json:"risk"`
	Impact          string             `json:"impact"`
	Recommendation  string             `json:"recommendation"`
}

// BreakingOptionType defines types of cycle breaking options
type BreakingOptionType string

const (
	BreakingOptionTypeRemove    BreakingOptionType = "remove"
	BreakingOptionTypeReplace   BreakingOptionType = "replace"
	BreakingOptionTypeDowngrade BreakingOptionType = "downgrade"
	BreakingOptionTypeExclude   BreakingOptionType = "exclude"
	BreakingOptionTypeRefactor  BreakingOptionType = "refactor"
)

// BreakingRiskLevel defines risk levels for breaking changes
type BreakingRiskLevel string

const (
	BreakingRiskLow      BreakingRiskLevel = "low"
	BreakingRiskMedium   BreakingRiskLevel = "medium"
	BreakingRiskHigh     BreakingRiskLevel = "high"
	BreakingRiskCritical BreakingRiskLevel = "critical"
)

// AutoUpdateConfig represents automatic update configuration
type AutoUpdateConfig struct {
	Enabled              bool              `json:"enabled"`
	UpdateStrategy       UpdateStrategy    `json:"updateStrategy"`
	MaintenanceWindows   []*MaintenanceWindow `json:"maintenanceWindows,omitempty"`
	ExcludedPackages     []string          `json:"excludedPackages,omitempty"`
	IncludedPackages     []string          `json:"includedPackages,omitempty"`
	UpdateTypes          []UpdateType      `json:"updateTypes"`
	ApprovalRequired     bool              `json:"approvalRequired"`
	RollbackEnabled      bool              `json:"rollbackEnabled"`
	TestingRequired      bool              `json:"testingRequired"`
	MaxConcurrentUpdates int               `json:"maxConcurrentUpdates"`
	UpdateInterval       time.Duration     `json:"updateInterval"`
	NotificationConfig   *NotificationConfig `json:"notificationConfig,omitempty"`
}


// MaintenanceWindow represents a maintenance window for updates
type MaintenanceWindow struct {
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	StartTime   string        `json:"startTime"` // Time in format "15:04"
	EndTime     string        `json:"endTime"`   // Time in format "15:04"
	Days        []Weekday     `json:"days"`      // Days of week
	TimeZone    string        `json:"timeZone"`
	Duration    time.Duration `json:"duration,omitempty"`
	Recurrence  Recurrence    `json:"recurrence"`
}

// Weekday represents days of the week
type Weekday string

const (
	Monday    Weekday = "monday"
	Tuesday   Weekday = "tuesday"
	Wednesday Weekday = "wednesday"
	Thursday  Weekday = "thursday"
	Friday    Weekday = "friday"
	Saturday  Weekday = "saturday"
	Sunday    Weekday = "sunday"
)

// Recurrence defines recurrence patterns
type Recurrence string

const (
	RecurrenceDaily   Recurrence = "daily"
	RecurrenceWeekly  Recurrence = "weekly"
	RecurrenceMonthly Recurrence = "monthly"
	RecurrenceCustom  Recurrence = "custom"
)

// NotificationConfig represents notification configuration
type NotificationConfig struct {
	Enabled     bool                    `json:"enabled"`
	Channels    []*NotificationChannel  `json:"channels"`
	Templates   map[string]string       `json:"templates,omitempty"`
	Events      []NotificationEvent     `json:"events"`
}

// NotificationChannel represents a notification channel
type NotificationChannel struct {
	Type     ChannelType       `json:"type"`
	Endpoint string            `json:"endpoint"`
	Config   map[string]interface{} `json:"config,omitempty"`
}

// ChannelType defines notification channel types
type ChannelType string

const (
	ChannelTypeEmail    ChannelType = "email"
	ChannelTypeSlack    ChannelType = "slack"
	ChannelTypeWebhook  ChannelType = "webhook"
	ChannelTypeSMS      ChannelType = "sms"
)

// NotificationEvent defines notification events
type NotificationEvent string

const (
	NotificationEventUpdateAvailable NotificationEvent = "update_available"
	NotificationEventUpdateStarted   NotificationEvent = "update_started"
	NotificationEventUpdateCompleted NotificationEvent = "update_completed"
	NotificationEventUpdateFailed    NotificationEvent = "update_failed"
	NotificationEventSecurityAlert   NotificationEvent = "security_alert"
	NotificationEventError           NotificationEvent = "error"
)

// Additional missing types from graph.go

// GraphUpdate represents an update to a dependency graph
type GraphUpdate struct {
	Type        UpdateType        `json:"type"`
	Node        *GraphNode        `json:"node,omitempty"`
	Edge        *GraphEdge        `json:"edge,omitempty"`
	Operation   UpdateOperation   `json:"operation"`
	Timestamp   time.Time         `json:"timestamp"`
	Reason      string            `json:"reason,omitempty"`
	Impact      UpdateImpact      `json:"impact"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateOperation defines graph update operations
type UpdateOperation string

const (
	UpdateOperationAdd    UpdateOperation = "add"
	UpdateOperationRemove UpdateOperation = "remove"
	UpdateOperationUpdate UpdateOperation = "update"
	UpdateOperationMove   UpdateOperation = "move"
)

// UpdateImpact defines impact levels of graph updates
type UpdateImpact string

const (
	UpdateImpactNone     UpdateImpact = "none"
	UpdateImpactMinimal  UpdateImpact = "minimal"
	UpdateImpactModerate UpdateImpact = "moderate"
	UpdateImpactHigh     UpdateImpact = "high"
	UpdateImpactCritical UpdateImpact = "critical"
)

// GraphVisitor represents a visitor for graph traversal
type GraphVisitor interface {
	VisitNode(node *GraphNode) error
	VisitEdge(edge *GraphEdge) error
	PreOrder() bool
	PostOrder() bool
}

// GraphPath represents a path through the dependency graph
type GraphPath struct {
	Nodes    []*GraphNode `json:"nodes"`
	Edges    []*GraphEdge `json:"edges"`
	Length   int          `json:"length"`
	Weight   float64      `json:"weight"`
	Cost     float64      `json:"cost,omitempty"`
	Valid    bool         `json:"valid"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ComplexityAnalysis represents complexity analysis results
type ComplexityAnalysis struct {
	AnalysisID         string            `json:"analysisId"`
	OverallComplexity  float64           `json:"overallComplexity"`
	CyclomaticComplexity float64         `json:"cyclomaticComplexity"`
	CognitiveComplexity float64          `json:"cognitiveComplexity"`
	DepthComplexity    float64           `json:"depthComplexity"`
	BreadthComplexity  float64           `json:"breadthComplexity"`
	ComplexityFactors  []ComplexityFactor `json:"complexityFactors"`
	Recommendations    []string          `json:"recommendations,omitempty"`
	AnalyzedAt         time.Time         `json:"analyzedAt"`
}

// ComplexityFactor represents a factor contributing to complexity
type ComplexityFactor struct {
	Type         ComplexityFactorType `json:"type"`
	Value        float64              `json:"value"`
	Weight       float64              `json:"weight"`
	Description  string               `json:"description"`
	Contribution float64              `json:"contribution"`
}

// ComplexityFactorType defines types of complexity factors
type ComplexityFactorType string

const (
	ComplexityFactorNodes       ComplexityFactorType = "nodes"
	ComplexityFactorEdges       ComplexityFactorType = "edges"
	ComplexityFactorCycles      ComplexityFactorType = "cycles"
	ComplexityFactorDepth       ComplexityFactorType = "depth"
	ComplexityFactorFanOut      ComplexityFactorType = "fan_out"
	ComplexityFactorCoupling    ComplexityFactorType = "coupling"
)

// GraphFilter represents filters for graph operations
type GraphFilter struct {
	IncludePackages []string            `json:"includePackages,omitempty"`
	ExcludePackages []string            `json:"excludePackages,omitempty"`
	IncludeScopes   []string            `json:"includeScopes,omitempty"`
	ExcludeScopes   []string            `json:"excludeScopes,omitempty"`
	MaxDepth        int                 `json:"maxDepth,omitempty"`
	MinWeight       float64             `json:"minWeight,omitempty"`
	NodeTypes       []string            `json:"nodeTypes,omitempty"`
	EdgeTypes       []string            `json:"edgeTypes,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// GraphMetadata represents metadata for graph operations
type GraphMetadata struct {
	Version     string                 `json:"version"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	Generator   string                 `json:"generator"`
	Environment string                 `json:"environment,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
}

// OptimizationOptions represents options for graph optimization
type OptimizationOptions struct {
	Strategy           OptimizationStrategy `json:"strategy"`
	MaxIterations      int                  `json:"maxIterations"`
	ConvergenceThreshold float64            `json:"convergenceThreshold"`
	PreserveStructure  bool                 `json:"preserveStructure"`
	MinimizeNodes      bool                 `json:"minimizeNodes"`
	MinimizeEdges      bool                 `json:"minimizeEdges"`
	OptimizePerformance bool                `json:"optimizePerformance"`
	OptimizeCost       bool                 `json:"optimizeCost"`
	Constraints        []OptimizationConstraint `json:"constraints,omitempty"`
	Objectives         []OptimizationObjective  `json:"objectives,omitempty"`
}

// OptimizationConstraint represents a constraint for optimization
type OptimizationConstraint struct {
	Type        string  `json:"type"`
	Target      string  `json:"target"`
	Operator    string  `json:"operator"` // "<=", ">=", "=", "!=", "<", ">"
	Value       float64 `json:"value"`
	Description string  `json:"description,omitempty"`
}


// OptimizationObjective represents an optimization objective
type OptimizationObjective struct {
	Type        ObjectiveType `json:"type"`
	Weight      float64       `json:"weight"`
	Target      string        `json:"target,omitempty"`
	Description string        `json:"description,omitempty"`
}

// ObjectiveType defines types of optimization objectives
type ObjectiveType string

const (
	ObjectiveTypeMinimizeCost        ObjectiveType = "minimize_cost"
	ObjectiveTypeMaximizePerformance ObjectiveType = "maximize_performance"
	ObjectiveTypeMinimizeComplexity  ObjectiveType = "minimize_complexity"
	ObjectiveTypeMinimizeRisk        ObjectiveType = "minimize_risk"
	ObjectiveTypeMaximizeReliability ObjectiveType = "maximize_reliability"
)

// VersionStrategy defines version selection strategies
type VersionStrategy string

const (
	VersionStrategyExact      VersionStrategy = "exact"
	VersionStrategyLatest     VersionStrategy = "latest"
	VersionStrategyStable     VersionStrategy = "stable"
	VersionStrategyPrerelease VersionStrategy = "prerelease"
	VersionStrategyRange      VersionStrategy = "range"
	VersionStrategyPinned     VersionStrategy = "pinned"
)

// PackageFilter represents filters for package operations
type PackageFilter struct {
	Names           []string        `json:"names,omitempty"`
	Namespaces      []string        `json:"namespaces,omitempty"`
	Versions        []string        `json:"versions,omitempty"`
	VersionRange    *VersionRange   `json:"versionRange,omitempty"`
	Licenses        []string        `json:"licenses,omitempty"`
	Categories      []string        `json:"categories,omitempty"`
	Tags            []string        `json:"tags,omitempty"`
	MinRating       float64         `json:"minRating,omitempty"`
	MaxAge          time.Duration   `json:"maxAge,omitempty"`
	IncludePrerelease bool          `json:"includePrerelease,omitempty"`
	ExcludeDeprecated bool          `json:"excludeDeprecated,omitempty"`
	SecurityFilter  *SecurityFilter `json:"securityFilter,omitempty"`
	CustomFilters   map[string]interface{} `json:"customFilters,omitempty"`
}

// SecurityFilter represents security-related package filters
type SecurityFilter struct {
	MinSecurityScore     float64 `json:"minSecurityScore,omitempty"`
	MaxVulnerabilities   int     `json:"maxVulnerabilities,omitempty"`
	ExcludedSeverities   []string `json:"excludedSeverities,omitempty"`
	RequireSignature     bool    `json:"requireSignature,omitempty"`
	RequireVerification  bool    `json:"requireVerification,omitempty"`
}

// Additional missing types from graph.go interfaces

// GraphTransformer interface for graph transformations
type GraphTransformer interface {
	Transform(graph *DependencyGraph) (*DependencyGraph, error)
	CanTransform(graph *DependencyGraph) bool
	TransformationType() string
}

// GraphValidation represents graph validation results
type GraphValidation struct {
	Valid           bool                   `json:"valid"`
	Errors          []*ValidationError     `json:"errors,omitempty"`
	Warnings        []*ValidationWarning   `json:"warnings,omitempty"`
	ValidationScore float64                `json:"validationScore"`
	ValidatedAt     time.Time              `json:"validatedAt"`
	Validator       string                 `json:"validator"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// GraphManagerHealth represents health status of graph manager
type GraphManagerHealth struct {
	Status          string            `json:"status"`
	Components      map[string]string `json:"components"`
	LastCheck       time.Time         `json:"lastCheck"`
	Issues          []string          `json:"issues,omitempty"`
	UpTime          time.Duration     `json:"upTime"`
	MemoryUsage     int64             `json:"memoryUsage"`
	GoroutineCount  int               `json:"goroutineCount"`
}

// GraphManagerMetrics represents metrics for graph manager
type GraphManagerMetrics struct {
	GraphsManaged       int64         `json:"graphsManaged"`
	NodesTotal          int64         `json:"nodesTotal"`
	EdgesTotal          int64         `json:"edgesTotal"`
	OperationsPerSecond float64       `json:"operationsPerSecond"`
	AverageResponseTime time.Duration `json:"averageResponseTime"`
	CacheHitRate        float64       `json:"cacheHitRate"`
	ErrorRate           float64       `json:"errorRate"`
	LastUpdated         time.Time     `json:"lastUpdated"`
	
	// Analysis metrics - these need to have Observe() method for compatibility
	CycleDetectionTime     MetricObserver `json:"-"`
	CyclesDetected         MetricCounter  `json:"-"`
	TopologicalSortTime    MetricObserver `json:"-"`
	MetricsCalculationTime MetricObserver `json:"-"`
}

// MetricObserver interface for metrics that observe values
type MetricObserver interface {
	Observe(value float64)
}

// MetricCounter interface for metrics that count events
type MetricCounter interface {
	Add(value float64)
}

// Simple stub implementations for metrics
type stubMetricObserver struct{}
func (s *stubMetricObserver) Observe(value float64) {}

type stubMetricCounter struct{}
func (s *stubMetricCounter) Add(value float64) {}

// DependencyInfo represents information about a dependency relationship
type DependencyInfo struct {
	Package     *PackageReference `json:"package"`
	Version     string            `json:"version,omitempty"`
	Scope       DependencyScope   `json:"scope"`
	Optional    bool              `json:"optional,omitempty"`
	Transitive  bool              `json:"transitive,omitempty"`
	Source      string            `json:"source,omitempty"`
	Reason      string            `json:"reason,omitempty"`
}

// GraphStorage interface for graph persistence
type GraphStorage interface {
	Store(graph *DependencyGraph) error
	Load(id string) (*DependencyGraph, error)
	Delete(id string) error
	List() ([]string, error)
	Exists(id string) bool
	Close() error
}

// GraphAnalyzer interface for graph analysis
type GraphAnalyzer interface {
	Analyze(graph *DependencyGraph) (*GraphAnalysis, error)
	AnalyzeComplexity(graph *DependencyGraph) (*ComplexityAnalysis, error)
	FindCycles(graph *DependencyGraph) ([]*CyclicDependency, error)
	FindCriticalPath(graph *DependencyGraph) (*CriticalPath, error)
	DetectPatterns(graph *DependencyGraph) ([]*GraphPattern, error)
}

// GraphVisualizer interface for graph visualization
type GraphVisualizer interface {
	GenerateVisualization(graph *DependencyGraph, format string) ([]byte, error)
	GenerateLayout(graph *DependencyGraph, layout LayoutType) (*GraphLayout, error)
	ExportToFormat(graph *DependencyGraph, format string) ([]byte, error)
}


// LayoutType defines graph layout types
type LayoutType string

const (
	LayoutTypeHierarchical LayoutType = "hierarchical"
	LayoutTypeCircular     LayoutType = "circular"
	LayoutTypeForceDirected LayoutType = "force_directed"
	LayoutTypeTree         LayoutType = "tree"
	LayoutTypeRadial       LayoutType = "radial"
)


// GraphLayout represents graph layout information
type GraphLayout struct {
	Type        LayoutType               `json:"type"`
	Nodes       map[string]*NodePosition `json:"nodes"`
	Bounds      *LayoutBounds            `json:"bounds"`
	Properties  map[string]interface{}   `json:"properties,omitempty"`
	GeneratedAt time.Time                `json:"generatedAt"`
}

// NodePosition represents position of a node in layout
type NodePosition struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width,omitempty"`
	Height float64 `json:"height,omitempty"`
}

// LayoutBounds represents layout boundaries
type LayoutBounds struct {
	MinX float64 `json:"minX"`
	MaxX float64 `json:"maxX"`
	MinY float64 `json:"minY"`
	MaxY float64 `json:"maxY"`
}

// GraphOptimizer interface for graph optimization
type GraphOptimizer interface {
	Optimize(graph *DependencyGraph, options *OptimizationOptions) (*OptimizedGraph, error)
	EstimateOptimization(graph *DependencyGraph, options *OptimizationOptions) (*OptimizationEstimate, error)
	ValidateOptimization(original, optimized *DependencyGraph) (*OptimizationValidation, error)
}

// OptimizationEstimate represents optimization estimate
type OptimizationEstimate struct {
	PotentialSavings    float64       `json:"potentialSavings"`
	EstimatedTime       time.Duration `json:"estimatedTime"`
	EstimatedComplexity float64       `json:"estimatedComplexity"`
	RiskAssessment      string        `json:"riskAssessment"`
	Recommendations     []string      `json:"recommendations,omitempty"`
}

// OptimizationValidation represents optimization validation results
type OptimizationValidation struct {
	Valid               bool     `json:"valid"`
	FunctionalEquivalence bool   `json:"functionalEquivalence"`
	PerformanceImprovement float64 `json:"performanceImprovement"`
	Issues              []string `json:"issues,omitempty"`
	ValidatedAt         time.Time `json:"validatedAt"`
}

// GraphAlgorithm interface for graph algorithms
type GraphAlgorithm interface {
	Execute(graph *DependencyGraph) (interface{}, error)
	Name() string
	Description() string
	RequiredCapabilities() []string
}

// Additional missing types from various files

// GraphWorkerPool represents a pool of workers for graph operations
type GraphWorkerPool interface {
	Submit(task GraphTask) error
	SubmitWithTimeout(task GraphTask, timeout time.Duration) error
	Workers() int
	QueueSize() int
	ActiveJobs() int
	Close() error
}

// GraphTask represents a task for graph workers
type GraphTask interface {
	Execute() error
	ID() string
	Priority() int
}

// GraphManagerConfig represents configuration for graph manager
type GraphManagerConfig struct {
	MaxConcurrency    int           `json:"maxConcurrency"`
	WorkerPoolSize    int           `json:"workerPoolSize"`
	QueueSize         int           `json:"queueSize"`
	CacheEnabled      bool          `json:"cacheEnabled"`
	CacheSize         int           `json:"cacheSize"`
	CacheTTL          time.Duration `json:"cacheTTL"`
	StorageEnabled    bool          `json:"storageEnabled"`
	StorageConfig     *StorageConfig `json:"storageConfig,omitempty"`
	MetricsEnabled    bool          `json:"metricsEnabled"`
	MetricsInterval   time.Duration `json:"metricsInterval"`
	ValidationEnabled bool          `json:"validationEnabled"`
	OptimizationLevel OptimizationLevel `json:"optimizationLevel"`
}

// StorageConfig represents storage configuration
type StorageConfig struct {
	Type     StorageType `json:"type"`
	URL      string      `json:"url"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

// StorageType defines storage types
type StorageType string

const (
	StorageTypeMemory     StorageType = "memory"
	StorageTypeFile       StorageType = "file"
	StorageTypeDatabase   StorageType = "database"
	StorageTypeRedis      StorageType = "redis"
	StorageTypeEtcd       StorageType = "etcd"
)

// OptimizationLevel defines optimization levels
type OptimizationLevel string

const (
	OptimizationLevelNone     OptimizationLevel = "none"
	OptimizationLevelBasic    OptimizationLevel = "basic"
	OptimizationLevelStandard OptimizationLevel = "standard"
	OptimizationLevelAdvanced OptimizationLevel = "advanced"
)

// PlatformConstraints represents platform-specific constraints
type PlatformConstraints struct {
	OS                []string          `json:"os,omitempty"`
	Architecture      []string          `json:"architecture,omitempty"`
	MinCPUCores       int               `json:"minCPUCores,omitempty"`
	MinMemory         int64             `json:"minMemory,omitempty"`
	MinDiskSpace      int64             `json:"minDiskSpace,omitempty"`
	RequiredFeatures  []string          `json:"requiredFeatures,omitempty"`
	ForbiddenFeatures []string          `json:"forbiddenFeatures,omitempty"`
	EnvironmentVars   map[string]string `json:"environmentVars,omitempty"`
	NetworkAccess     NetworkAccess     `json:"networkAccess,omitempty"`
	SecurityContext   *SecurityContext  `json:"securityContext,omitempty"`
}

// NetworkAccess defines network access requirements
type NetworkAccess string

const (
	NetworkAccessNone       NetworkAccess = "none"
	NetworkAccessRestricted NetworkAccess = "restricted"
	NetworkAccessInternet   NetworkAccess = "internet"
	NetworkAccessFull       NetworkAccess = "full"
)

// SecurityContext represents security context constraints
type SecurityContext struct {
	RunAsUser         *int64            `json:"runAsUser,omitempty"`
	RunAsGroup        *int64            `json:"runAsGroup,omitempty"`
	RunAsNonRoot      *bool             `json:"runAsNonRoot,omitempty"`
	ReadOnlyRootFilesystem *bool        `json:"readOnlyRootFilesystem,omitempty"`
	AllowPrivilegeEscalation *bool      `json:"allowPrivilegeEscalation,omitempty"`
	Capabilities      *SecurityCapabilities `json:"capabilities,omitempty"`
	SELinuxOptions    *SELinuxOptions   `json:"seLinuxOptions,omitempty"`
}

// SecurityCapabilities represents security capabilities
type SecurityCapabilities struct {
	Add  []string `json:"add,omitempty"`
	Drop []string `json:"drop,omitempty"`
}

// SELinuxOptions represents SELinux options
type SELinuxOptions struct {
	User  string `json:"user,omitempty"`
	Role  string `json:"role,omitempty"`
	Type  string `json:"type,omitempty"`
	Level string `json:"level,omitempty"`
}

// SecurityConstraints represents security-related constraints
type SecurityConstraints struct {
	MaxVulnerabilities      int              `json:"maxVulnerabilities,omitempty"`
	ForbiddenSeverities     []string         `json:"forbiddenSeverities,omitempty"`
	RequiredLicenses        []string         `json:"requiredLicenses,omitempty"`
	ForbiddenLicenses       []string         `json:"forbiddenLicenses,omitempty"`
	MinSecurityScore        float64          `json:"minSecurityScore,omitempty"`
	RequireSignature        bool             `json:"requireSignature,omitempty"`
	RequireVerification     bool             `json:"requireVerification,omitempty"`
	TrustedPublishers       []string         `json:"trustedPublishers,omitempty"`
	BlacklistedPublishers   []string         `json:"blacklistedPublishers,omitempty"`
	ComplianceRequirements  []ComplianceRequirement `json:"complianceRequirements,omitempty"`
	SecurityPolicies        []*SecurityPolicy `json:"securityPolicies,omitempty"`
}

// ComplianceRequirement represents a compliance requirement
type ComplianceRequirement struct {
	Standard    string   `json:"standard"`
	Version     string   `json:"version,omitempty"`
	Controls    []string `json:"controls,omitempty"`
	Mandatory   bool     `json:"mandatory"`
	Description string   `json:"description,omitempty"`
}

// SecurityPolicy represents a security policy
type SecurityPolicy struct {
	Name        string            `json:"name"`
	Version     string            `json:"version,omitempty"`
	Rules       []*SecurityRule   `json:"rules"`
	Enforcement PolicyEnforcement `json:"enforcement"`
	Description string            `json:"description,omitempty"`
}

// SecurityRule represents a security rule
type SecurityRule struct {
	ID          string            `json:"id"`
	Type        SecurityRuleType  `json:"type"`
	Condition   string            `json:"condition"`
	Action      SecurityAction    `json:"action"`
	Severity    SecuritySeverity  `json:"severity"`
	Message     string            `json:"message,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// SecurityRuleType defines types of security rules
type SecurityRuleType string

const (
	SecurityRuleTypeVulnerability SecurityRuleType = "vulnerability"
	SecurityRuleTypeLicense       SecurityRuleType = "license"
	SecurityRuleTypeCompliance    SecurityRuleType = "compliance"
	SecurityRuleTypePolicy        SecurityRuleType = "policy"
	SecurityRuleTypeBehavior      SecurityRuleType = "behavior"
)

// SecurityAction defines security actions
type SecurityAction string

const (
	SecurityActionAllow     SecurityAction = "allow"
	SecurityActionDeny      SecurityAction = "deny"
	SecurityActionWarn      SecurityAction = "warn"
	SecurityActionQuarantine SecurityAction = "quarantine"
	SecurityActionBlock     SecurityAction = "block"
)

// SecuritySeverity defines security severity levels
type SecuritySeverity string

const (
	SecuritySeverityInfo     SecuritySeverity = "info"
	SecuritySeverityLow      SecuritySeverity = "low"
	SecuritySeverityMedium   SecuritySeverity = "medium"
	SecuritySeverityHigh     SecuritySeverity = "high"
	SecuritySeverityCritical SecuritySeverity = "critical"
)

// PolicyEnforcement defines policy enforcement modes
type PolicyEnforcement string

const (
	PolicyEnforcementPermissive PolicyEnforcement = "permissive"
	PolicyEnforcementEnforcing  PolicyEnforcement = "enforcing"
	PolicyEnforcementDisabled   PolicyEnforcement = "disabled"
)

// Final batch of missing types

// TransitiveOptions represents options for transitive dependency resolution
type TransitiveOptions struct {
	MaxDepth        int           `json:"maxDepth,omitempty"`
	IncludeOptional bool          `json:"includeOptional,omitempty"`
	IncludeTest     bool          `json:"includeTest,omitempty"`
	ExcludedScopes  []string      `json:"excludedScopes,omitempty"`
	IncludedScopes  []string      `json:"includedScopes,omitempty"`
	Filter          *PackageFilter `json:"filter,omitempty"`
	Strategy        TransitiveStrategy `json:"strategy"`
	Timeout         time.Duration `json:"timeout,omitempty"`
}

// TransitiveStrategy defines strategies for transitive resolution
type TransitiveStrategy string

const (
	TransitiveStrategyBreadthFirst TransitiveStrategy = "breadth_first"
	TransitiveStrategyDepthFirst   TransitiveStrategy = "depth_first"
	TransitiveStrategyOptimal      TransitiveStrategy = "optimal"
)

// TransitiveResult represents the result of transitive dependency resolution
type TransitiveResult struct {
	Dependencies []*ResolvedDependency `json:"dependencies"`
	Tree         *DependencyTree       `json:"tree"`
	Statistics   *ResolutionStatistics `json:"statistics"`
	Warnings     []*ResolutionWarning  `json:"warnings,omitempty"`
	Errors       []*ResolutionError    `json:"errors,omitempty"`
	ResolvedAt   time.Time             `json:"resolvedAt"`
}

// PolicyConstraints represents policy-based constraints
type PolicyConstraints struct {
	AllowedLicenses       []string               `json:"allowedLicenses,omitempty"`
	ForbiddenLicenses     []string               `json:"forbiddenLicenses,omitempty"`
	MaxPackageAge         time.Duration          `json:"maxPackageAge,omitempty"`
	MinSecurityRating     float64                `json:"minSecurityRating,omitempty"`
	RequireActiveMaintenance bool                `json:"requireActiveMaintenance,omitempty"`
	TrustedPublishers     []string               `json:"trustedPublishers,omitempty"`
	BlacklistedPackages   []string               `json:"blacklistedPackages,omitempty"`
	CustomPolicies        []*CustomPolicy        `json:"customPolicies,omitempty"`
	ComplianceStandards   []ComplianceStandard   `json:"complianceStandards,omitempty"`
}

// CustomPolicy represents a custom policy constraint
type CustomPolicy struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Rules       []*PolicyRule     `json:"rules"`
	Severity    PolicySeverity    `json:"severity"`
	Action      PolicyAction      `json:"action"`
}

// PolicyRule represents a policy rule
type PolicyRule struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // "equals", "contains", "matches", "gt", "lt", etc.
	Value    interface{} `json:"value"`
	Negate   bool        `json:"negate,omitempty"`
}

// PolicySeverity defines policy severity levels
type PolicySeverity string

const (
	PolicySeverityInfo     PolicySeverity = "info"
	PolicySeverityWarning  PolicySeverity = "warning"
	PolicySeverityError    PolicySeverity = "error"
	PolicySeverityCritical PolicySeverity = "critical"
)

// PolicyAction defines policy actions
type PolicyAction string

const (
	PolicyActionLog     PolicyAction = "log"
	PolicyActionWarn    PolicyAction = "warn"
	PolicyActionReject  PolicyAction = "reject"
	PolicyActionBlock   PolicyAction = "block"
)

// ComplianceStandard represents a compliance standard
type ComplianceStandard struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Controls    []string `json:"controls,omitempty"`
	Mandatory   bool     `json:"mandatory"`
	Authority   string   `json:"authority,omitempty"`
}

// DependencyTree represents a dependency tree structure
type DependencyTree struct {
	Root     *TreeNode `json:"root"`
	Depth    int       `json:"depth"`
	NodeCount int      `json:"nodeCount"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// TreeNode represents a node in the dependency tree
type TreeNode struct {
	Package  *PackageReference `json:"package"`
	Children []*TreeNode       `json:"children,omitempty"`
	Parent   *TreeNode         `json:"parent,omitempty"`
	Level    int               `json:"level"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ResolutionWarning represents a warning during resolution
type ResolutionWarning struct {
	Code        string            `json:"code"`
	Type        WarningType       `json:"type"`
	Message     string            `json:"message"`
	Package     *PackageReference `json:"package,omitempty"`
	Suggestion  string            `json:"suggestion,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ResolutionError represents an error during resolution
type ResolutionError struct {
	Code        string            `json:"code"`
	Type        ErrorType         `json:"type"`
	Message     string            `json:"message"`
	Package     *PackageReference `json:"package,omitempty"`
	Cause       error             `json:"cause,omitempty"`
	Resolution  string            `json:"resolution,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ResolutionStatistics represents statistics from resolution
type ResolutionStatistics struct {
	TotalPackages       int           `json:"totalPackages"`
	ResolvedPackages    int           `json:"resolvedPackages"`
	UnresolvedPackages  int           `json:"unresolvedPackages"`
	ConflictsResolved   int           `json:"conflictsResolved"`
	ResolutionTime      time.Duration `json:"resolutionTime"`
	NetworkRequests     int           `json:"networkRequests"`
	CacheHits           int           `json:"cacheHits"`
	CacheMisses         int           `json:"cacheMisses"`
	DependencyDepth     int           `json:"dependencyDepth"`
}

// ResolvedDependency represents a resolved dependency
type ResolvedDependency struct {
	Package         *PackageReference   `json:"package"`
	RequestedBy     []*PackageReference `json:"requestedBy,omitempty"`
	ResolvedVersion string              `json:"resolvedVersion"`
	Source          ResolutionSource    `json:"source"`
	SecurityInfo    *SecurityInfo       `json:"securityInfo,omitempty"`
	LicenseInfo     *LicenseInfo        `json:"licenseInfo,omitempty"`
	PerformanceInfo *PerformanceInfo    `json:"performanceInfo,omitempty"`
	ComplianceInfo  *ComplianceInfo     `json:"complianceInfo,omitempty"`
	ResolvedAt      time.Time           `json:"resolvedAt"`
	ResolutionPath  []*PackageReference `json:"resolutionPath,omitempty"`
}

// ResolutionSource defines sources for dependency resolution
type ResolutionSource string

const (
	ResolutionSourceRegistry  ResolutionSource = "registry"
	ResolutionSourceCache     ResolutionSource = "cache"
	ResolutionSourceLocal     ResolutionSource = "local"
	ResolutionSourceMirror    ResolutionSource = "mirror"
	ResolutionSourceFallback  ResolutionSource = "fallback"
)

// LicenseInfo contains license information
type LicenseInfo struct {
	Name            string   `json:"name"`
	SPDXID          string   `json:"spdxId,omitempty"`
	URL             string   `json:"url,omitempty"`
	Text            string   `json:"text,omitempty"`
	Compatible      bool     `json:"compatible"`
	Category        string   `json:"category,omitempty"`
	Restrictions    []string `json:"restrictions,omitempty"`
	Permissions     []string `json:"permissions,omitempty"`
	Conditions      []string `json:"conditions,omitempty"`
}

// PerformanceInfo contains performance information
type PerformanceInfo struct {
	BundleSize      int64                  `json:"bundleSize,omitempty"`
	LoadTime        time.Duration          `json:"loadTime,omitempty"`
	MemoryUsage     int64                  `json:"memoryUsage,omitempty"`
	CPUUsage        float64                `json:"cpuUsage,omitempty"`
	StartupTime     time.Duration          `json:"startupTime,omitempty"`
	Benchmarks      []*PerformanceBenchmark `json:"benchmarks,omitempty"`
	OptimizationTips []string              `json:"optimizationTips,omitempty"`
}

// PerformanceBenchmark represents a performance benchmark
type PerformanceBenchmark struct {
	Name           string        `json:"name"`
	Value          float64       `json:"value"`
	Unit           string        `json:"unit"`
	BenchmarkedAt  time.Time     `json:"benchmarkedAt"`
	Environment    string        `json:"environment,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ComplianceInfo contains compliance information
type ComplianceInfo struct {
	Standards       []ComplianceStandard `json:"standards,omitempty"`
	Violations      []*ComplianceViolation `json:"violations,omitempty"`
	Score           float64              `json:"score"`
	LastAssessed    time.Time            `json:"lastAssessed"`
	CertificationLevel string            `json:"certificationLevel,omitempty"`
	Auditor         string               `json:"auditor,omitempty"`
}

// ComplianceViolation represents a compliance violation
type ComplianceViolation struct {
	Standard    string    `json:"standard"`
	Control     string    `json:"control"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Remediation string    `json:"remediation,omitempty"`
	DetectedAt  time.Time `json:"detectedAt"`
}

// ConflictResolutionStrategy represents a strategy for resolving conflicts
type ConflictResolutionStrategy struct {
	Type        ConflictResolutionType `json:"type"`
	Priority    int                    `json:"priority"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Conditions  []string               `json:"conditions,omitempty"`
}

// ConflictResolutionType defines types of conflict resolution strategies
type ConflictResolutionType string

const (
	ConflictResolutionTypeLatest      ConflictResolutionType = "latest"
	ConflictResolutionTypeOldest      ConflictResolutionType = "oldest"
	ConflictResolutionTypeNearest     ConflictResolutionType = "nearest"
	ConflictResolutionTypeMajority    ConflictResolutionType = "majority"
	ConflictResolutionTypeManual      ConflictResolutionType = "manual"
	ConflictResolutionTypeCustom      ConflictResolutionType = "custom"
)

// Final missing types from resolver.go and validator.go

// ConstraintSolution represents a solution for constraint resolution
type ConstraintSolution struct {
	ID          string                 `json:"id"`
	Type        SolutionType           `json:"type"`
	Description string                 `json:"description"`
	Packages    []*PackageReference    `json:"packages"`
	Changes     []*ConstraintChange    `json:"changes"`
	Cost        float64                `json:"cost"`
	Confidence  float64                `json:"confidence"`
	Valid       bool                   `json:"valid"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	
	// Additional fields needed by resolver.go
	Satisfiable bool                      `json:"satisfiable"`
	Assignments map[string]interface{}    `json:"assignments"`
	Conflicts   []*ConstraintConflict     `json:"conflicts"`
	Statistics  interface{}               `json:"statistics"`
	SolvingTime time.Duration             `json:"solvingTime"`
	Algorithm   string                    `json:"algorithm"`
}

// SolutionType defines types of constraint solutions
type SolutionType string

const (
	SolutionTypeUpgrade   SolutionType = "upgrade"
	SolutionTypeDowngrade SolutionType = "downgrade"
	SolutionTypeReplace   SolutionType = "replace"
	SolutionTypeRemove    SolutionType = "remove"
	SolutionTypeAdd       SolutionType = "add"
)

// ConstraintChange represents a change in constraints
type ConstraintChange struct {
	Type        ChangeType        `json:"type"`
	Package     *PackageReference `json:"package"`
	OldValue    string            `json:"oldValue,omitempty"`
	NewValue    string            `json:"newValue"`
	Reason      string            `json:"reason"`
	Impact      ChangeImpact      `json:"impact"`
}

// ChangeType defines types of constraint changes
type ChangeType string

const (
	ChangeTypeVersion     ChangeType = "version"
	ChangeTypeConstraint  ChangeType = "constraint"
	ChangeTypeDependency  ChangeType = "dependency"
	ChangeTypeExclusion   ChangeType = "exclusion"
)

// ChangeImpact defines impact levels of changes
type ChangeImpact string

const (
	ChangeImpactNone     ChangeImpact = "none"
	ChangeImpactMinor    ChangeImpact = "minor"
	ChangeImpactModerate ChangeImpact = "moderate"
	ChangeImpactMajor    ChangeImpact = "major"
	ChangeImpactBreaking ChangeImpact = "breaking"
)

// ConstraintValidation represents validation of constraints
type ConstraintValidation struct {
	Valid       bool                   `json:"valid"`
	Violations  []*ConstraintViolation `json:"violations,omitempty"`
	Warnings    []*ConstraintWarning   `json:"warnings,omitempty"`
	Score       float64                `json:"score"`
	ValidatedAt time.Time              `json:"validatedAt"`
	Validator   string                 `json:"validator"`
}

// ConstraintViolation represents a constraint violation
type ConstraintViolation struct {
	Type        ViolationType     `json:"type"`
	Constraint  string            `json:"constraint"`
	Package     *PackageReference `json:"package,omitempty"`
	Message     string            `json:"message"`
	Severity    ViolationSeverity `json:"severity"`
	Suggestion  string            `json:"suggestion,omitempty"`
}

// ViolationType defines types of constraint violations
type ViolationType string

const (
	ViolationTypeVersion      ViolationType = "version"
	ViolationTypeDependency   ViolationType = "dependency"
	ViolationTypeSecurity     ViolationType = "security"
	ViolationTypeLicense      ViolationType = "license"
	ViolationTypePolicy       ViolationType = "policy"
	ViolationTypeCompatibility ViolationType = "compatibility"
)

// ViolationSeverity defines severity levels of violations
type ViolationSeverity string

const (
	ViolationSeverityInfo     ViolationSeverity = "info"
	ViolationSeverityWarning  ViolationSeverity = "warning"
	ViolationSeverityError    ViolationSeverity = "error"
	ViolationSeverityCritical ViolationSeverity = "critical"
)

// ConstraintWarning represents a constraint warning
type ConstraintWarning struct {
	Type       WarningType       `json:"type"`
	Message    string            `json:"message"`
	Package    *PackageReference `json:"package,omitempty"`
	Suggestion string            `json:"suggestion,omitempty"`
}

// VersionRequirement represents a version requirement
type VersionRequirement struct {
	Package     *PackageReference `json:"package"`
	Constraint  string            `json:"constraint"`
	RequiredBy  *PackageReference `json:"requiredBy,omitempty"`
	Optional    bool              `json:"optional,omitempty"`
	Scope       string            `json:"scope,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// VersionResolution represents the result of version resolution
type VersionResolution struct {
	Package         *PackageReference   `json:"package"`
	ResolvedVersion string              `json:"resolvedVersion"`
	Candidates      []*VersionCandidate `json:"candidates"`
	Resolution      string              `json:"resolution"`
	Confidence      float64             `json:"confidence"`
	ResolvedAt      time.Time           `json:"resolvedAt"`
}

// VersionCandidate represents a version candidate
type VersionCandidate struct {
	Version     string            `json:"version"`
	Score       float64           `json:"score"`
	Available   bool              `json:"available"`
	Deprecated  bool              `json:"deprecated,omitempty"`
	Prerelease  bool              `json:"prerelease,omitempty"`
	Stability   string            `json:"stability,omitempty"`
	Source      string            `json:"source"`
	ReleaseDate time.Time         `json:"releaseDate,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}


// VersionConflict represents a version conflict
type VersionConflict struct {
	Package           *PackageReference    `json:"package"`
	ConflictingVersions []string           `json:"conflictingVersions"`
	RequiredBy        []*PackageReference  `json:"requiredBy"`
	Severity          ConflictSeverity     `json:"severity"`
	Impact            string               `json:"impact"`
	PossibleResolutions []*ConflictResolutionStrategy `json:"possibleResolutions,omitempty"`
}

// LicenseConflict represents a license conflict
type LicenseConflict struct {
	Package1        *PackageReference `json:"package1"`
	Package2        *PackageReference `json:"package2"`
	License1        string            `json:"license1"`
	License2        string            `json:"license2"`
	ConflictType    string            `json:"conflictType"`
	Severity        ConflictSeverity  `json:"severity"`
	Description     string            `json:"description"`
	Recommendations []string          `json:"recommendations,omitempty"`
}

// PolicyConflict represents a policy conflict
type PolicyConflict struct {
	Package         *PackageReference `json:"package"`
	Policy          string            `json:"policy"`
	ViolatedRule    string            `json:"violatedRule"`
	Severity        ConflictSeverity  `json:"severity"`
	Description     string            `json:"description"`
	RequiredAction  string            `json:"requiredAction"`
	Exemptions      []string          `json:"exemptions,omitempty"`
}

// ConflictResolutionSuggestion represents a suggestion for resolving conflicts
type ConflictResolutionSuggestion struct {
	ID              string                 `json:"id"`
	Type            SuggestionType         `json:"type"`
	Description     string                 `json:"description"`
	Actions         []*ResolutionAction    `json:"actions"`
	Priority        SuggestionPriority     `json:"priority"`
	Confidence      float64                `json:"confidence"`
	EstimatedEffort string                 `json:"estimatedEffort"`
	Risks           []string               `json:"risks,omitempty"`
	Benefits        []string               `json:"benefits,omitempty"`
}

// SuggestionType defines types of resolution suggestions
type SuggestionType string

const (
	SuggestionTypeAutomatic SuggestionType = "automatic"
	SuggestionTypeManual    SuggestionType = "manual"
	SuggestionTypeOptional  SuggestionType = "optional"
)

// SuggestionPriority defines priority levels for suggestions
type SuggestionPriority string

const (
	SuggestionPriorityLow      SuggestionPriority = "low"
	SuggestionPriorityMedium   SuggestionPriority = "medium"
	SuggestionPriorityHigh     SuggestionPriority = "high"
	SuggestionPriorityCritical SuggestionPriority = "critical"
)

// ResolutionAction represents an action to resolve a conflict
type ResolutionAction struct {
	Type        ActionType        `json:"type"`
	Package     *PackageReference `json:"package,omitempty"`
	OldValue    string            `json:"oldValue,omitempty"`
	NewValue    string            `json:"newValue"`
	Description string            `json:"description"`
	Reversible  bool              `json:"reversible"`
}

// ConflictImpactAnalysis represents analysis of conflict impact
type ConflictImpactAnalysis struct {
	AnalysisID        string                    `json:"analysisId"`
	Conflicts         []*DependencyConflict     `json:"conflicts"`
	OverallImpact     ConflictImpact            `json:"overallImpact"`
	ImpactScore       float64                   `json:"impactScore"`
	AffectedPackages  []*AffectedPackage        `json:"affectedPackages"`
	ImpactCategories  map[string]float64        `json:"impactCategories"`
	Recommendations   []*ImpactRecommendation   `json:"recommendations,omitempty"`
	AnalyzedAt        time.Time                 `json:"analyzedAt"`
}

// ImpactRecommendation represents a recommendation based on impact analysis
type ImpactRecommendation struct {
	Type            string    `json:"type"`
	Priority        string    `json:"priority"`
	Description     string    `json:"description"`
	AffectedPackages []*PackageReference `json:"affectedPackages"`
	Actions         []string  `json:"actions"`
	Timeline        string    `json:"timeline,omitempty"`
	Resources       []string  `json:"resources,omitempty"`
}

// Final final batch of missing types from resolver.go

// ConflictStrategy defines strategies for handling conflicts
type ConflictStrategy string

const (
	ConflictStrategyFail     ConflictStrategy = "fail"
	ConflictStrategyWarn     ConflictStrategy = "warn"
	ConflictStrategyResolve  ConflictStrategy = "resolve"
	ConflictStrategyIgnore   ConflictStrategy = "ignore"
)

// ConflictResolution represents the result of conflict resolution
type ConflictResolution struct {
	ConflictID      string                        `json:"conflictId"`
	Strategy        ConflictStrategy              `json:"strategy"`
	Resolution      []*ConflictResolutionStrategy `json:"resolution"`
	Success         bool                          `json:"success"`
	Changes         []*ConstraintChange           `json:"changes,omitempty"`
	ResolvedAt      time.Time                     `json:"resolvedAt"`
	Duration        time.Duration                 `json:"duration"`
}

// RollbackPlan represents a plan for rolling back changes
type RollbackPlan struct {
	PlanID      string            `json:"planId"`
	Description string            `json:"description"`
	Steps       []interface{}     `json:"steps"` // Using interface{} to avoid duplicate type conflict
	CreatedAt   time.Time         `json:"createdAt"`
	ValidUntil  time.Time         `json:"validUntil,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}


// CacheStats represents cache statistics
type CacheStats struct {
	Hits             int64   `json:"hits"`
	Misses           int64   `json:"misses"`
	HitRate          float64 `json:"hitRate"`
	Size             int64   `json:"size"`
	MaxSize          int64   `json:"maxSize"`
	Evictions        int64   `json:"evictions"`
	LastAccess       time.Time `json:"lastAccess,omitempty"`
	LastEviction     time.Time `json:"lastEviction,omitempty"`
}

// ResolverHealth represents resolver health status
type ResolverHealth struct {
	Status              string            `json:"status"`
	Components          map[string]string `json:"components"`
	LastCheck           time.Time         `json:"lastCheck"`
	Issues              []string          `json:"issues,omitempty"`
	UpTime              time.Duration     `json:"upTime"`
	ActiveResolutions   int               `json:"activeResolutions"`
	CacheHealth         *CacheStats       `json:"cacheHealth,omitempty"`
	RegistryConnectivity bool             `json:"registryConnectivity"`
}

// ResolverMetrics represents resolver performance metrics
type ResolverMetrics struct {
	ResolutionsTotal        int64         `json:"resolutionsTotal"`
	ResolutionsSuccessful   int64         `json:"resolutionsSuccessful"`
	ResolutionsFailed       int64         `json:"resolutionsFailed"`
	AverageResolutionTime   time.Duration `json:"averageResolutionTime"`
	CacheHitRate            float64       `json:"cacheHitRate"`
	ConflictsTotal          int64         `json:"conflictsTotal"`
	ConflictsResolved       int64         `json:"conflictsResolved"`
	NetworkRequestsTotal    int64         `json:"networkRequestsTotal"`
	NetworkRequestsFailed   int64         `json:"networkRequestsFailed"`
	LastUpdated             time.Time     `json:"lastUpdated"`
	
	// Prometheus counters and histograms referenced in resolver.go
	CacheHits                    prometheus.Counter
	CacheMisses                  prometheus.Counter
	ConstraintCacheHits          prometheus.Counter
	ConstraintCacheMisses        prometheus.Counter
	ConstraintSolvingTime        prometheus.Histogram
	ConstraintSolvingSuccess     prometheus.Counter
	ConstraintSolvingFailures    prometheus.Counter
	VersionResolutionTime        prometheus.Histogram
	VersionResolutionSuccess     prometheus.Counter
	VersionResolutionFailures    prometheus.Counter
	ConflictDetectionTime        prometheus.Histogram
	ConflictsDetected            prometheus.Counter
}

// Very final missing types from resolver.go

// ConstraintSolverMetrics represents metrics for constraint solver
type ConstraintSolverMetrics struct {
	SolveTime           time.Duration `json:"solveTime"`
	ConstraintsTotal    int           `json:"constraintsTotal"`
	ConstraintsSolved   int           `json:"constraintsSolved"`
	Iterations          int           `json:"iterations"`
	BacktrackingEvents  int           `json:"backtrackingEvents"`
	CacheHits           int           `json:"cacheHits"`
	CacheMisses         int           `json:"cacheMisses"`
}

// ConstraintCache represents a cache for constraint solving
type ConstraintCache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
	Delete(key string)
	Clear()
	Size() int
	Stats() *CacheStats
}

// SolverHeuristic represents heuristics for constraint solving
type SolverHeuristic interface {
	SelectVariable(variables []string) string
	SelectValue(variable string, domain []interface{}) interface{}
	Priority() int
	Name() string
}

// ConstraintOptimizer represents an optimizer for constraints
type ConstraintOptimizer interface {
	Optimize(constraints []*Constraint) ([]*Constraint, error)
	EstimateOptimization(constraints []*Constraint) (*OptimizationEstimate, error)
	CanOptimize(constraints []*Constraint) bool
}

// VersionSolverMetrics represents metrics for version solver
type VersionSolverMetrics struct {
	SolveTime        time.Duration `json:"solveTime"`
	VersionsEvaluated int          `json:"versionsEvaluated"`
	ConflictsFound   int           `json:"conflictsFound"`
	ResolutionSteps  int           `json:"resolutionSteps"`
	CacheHits        int           `json:"cacheHits"`
	CacheMisses      int           `json:"cacheMisses"`
}

// VersionCache represents a cache for version resolution
type VersionCache interface {
	Get(packageName, version string) (interface{}, bool)
	Set(packageName, version string, value interface{})
	Delete(packageName, version string)
	Clear()
	Size() int
	Stats() *CacheStats
}

// VersionComparator interface for comparing versions
type VersionComparator interface {
	Compare(version1, version2 string) int
	IsCompatible(version1, version2 string) bool
	ParseVersion(version string) (*ParsedVersion, error)
	ValidateVersion(version string) error
}

// ParsedVersion represents a parsed version
type ParsedVersion struct {
	Major      int    `json:"major"`
	Minor      int    `json:"minor"`
	Patch      int    `json:"patch"`
	Prerelease string `json:"prerelease,omitempty"`
	Build      string `json:"build,omitempty"`
	Original   string `json:"original"`
}

// VersionSelector interface for selecting versions
type VersionSelector interface {
	SelectVersion(candidates []*VersionCandidate, constraints []string) (*VersionCandidate, error)
	RankCandidates(candidates []*VersionCandidate) []*VersionCandidate
	FilterCandidates(candidates []*VersionCandidate, criteria *SelectionCriteria) []*VersionCandidate
}

// SelectionCriteria represents criteria for version selection
type SelectionCriteria struct {
	PreferStable     bool      `json:"preferStable"`
	AllowPrerelease  bool      `json:"allowPrerelease"`
	MaxAge           time.Duration `json:"maxAge,omitempty"`
	MinSecurityScore float64   `json:"minSecurityScore,omitempty"`
	RequiredFeatures []string  `json:"requiredFeatures,omitempty"`
	ExcludedVersions []string  `json:"excludedVersions,omitempty"`
}

// RollbackResult represents the result of a rollback operation
type RollbackResult struct {
	PlanID      string            `json:"planId"`
	Success     bool              `json:"success"`
	Steps       []interface{}     `json:"steps"`  // Using interface{} to avoid conflicts
	Errors      []string          `json:"errors,omitempty"`
	Duration    time.Duration     `json:"duration"`
	RolledBackAt time.Time        `json:"rolledBackAt"`
}

// ResolutionCache represents a cache for dependency resolution
type ResolutionCache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
	Delete(key string)
	Clear()
	Size() int
	Stats() *CacheStats
}

// PrereleaseStrategy defines strategies for handling prerelease versions
type PrereleaseStrategy string

const (
	PrereleaseStrategyAllow  PrereleaseStrategy = "allow"
	PrereleaseStrategyDeny   PrereleaseStrategy = "deny"
	PrereleaseStrategyLatest PrereleaseStrategy = "latest"
)

// BuildMetadataStrategy defines strategies for handling build metadata
type BuildMetadataStrategy string

const (
	BuildMetadataStrategyIgnore  BuildMetadataStrategy = "ignore"
	BuildMetadataStrategyInclude BuildMetadataStrategy = "include"
	BuildMetadataStrategyPrefer  BuildMetadataStrategy = "prefer"
)

// ConflictResolverMetrics represents metrics for conflict resolver
type ConflictResolverMetrics struct {
	ConflictsDetected   int64         `json:"conflictsDetected"`
	ConflictsResolved   int64         `json:"conflictsResolved"`
	AverageResolveTime  time.Duration `json:"averageResolveTime"`
	FailedResolutions   int64         `json:"failedResolutions"`
	AutomaticResolutions int64        `json:"automaticResolutions"`
	ManualResolutions   int64         `json:"manualResolutions"`
}

// ConflictDetector interface for detecting conflicts
type ConflictDetector interface {
	DetectConflicts(dependencies []*PackageReference) ([]*DependencyConflict, error)
	ValidateResolution(resolution *DependencyResolution) error
	AnalyzeImpact(conflicts []*DependencyConflict) (*ConflictImpactAnalysis, error)
}

// ConflictPredictor interface for predicting conflicts
type ConflictPredictor interface {
	PredictConflicts(dependencies []*PackageReference) ([]*PredictedConflict, error)
	EstimateResolutionEffort(conflicts []*DependencyConflict) (time.Duration, error)
	RecommendPrevention(predictions []*PredictedConflict) ([]*PreventionRecommendation, error)
}

// PredictedConflict represents a predicted conflict
type PredictedConflict struct {
	ID          string               `json:"id"`
	Type        ConflictType         `json:"type"`
	Probability float64              `json:"probability"`
	Packages    []*PackageReference  `json:"packages"`
	Description string               `json:"description"`
	Impact      ConflictImpact       `json:"impact"`
	Prevention  []string             `json:"prevention,omitempty"`
}

// PreventionRecommendation represents a recommendation to prevent conflicts
type PreventionRecommendation struct {
	Type        string               `json:"type"`
	Description string               `json:"description"`
	Actions     []string             `json:"actions"`
	Packages    []*PackageReference  `json:"packages,omitempty"`
	Priority    string               `json:"priority"`
}

// Constraint represents a generic constraint
type Constraint struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Package     *PackageReference `json:"package,omitempty"`
	Expression  string            `json:"expression"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// DependencyResolution represents the result of dependency resolution
type DependencyResolution struct {
	Dependencies []*ResolvedDependency `json:"dependencies"`
	Conflicts    []*DependencyConflict `json:"conflicts,omitempty"`
	Warnings     []*ResolutionWarning  `json:"warnings,omitempty"`
	Success      bool                  `json:"success"`
	Duration     time.Duration         `json:"duration"`
	ResolvedAt   time.Time             `json:"resolvedAt"`
}

// WorkerPool interface for managing worker pools
type WorkerPool interface {
	Submit(task func() error) error
	Workers() int
	ActiveJobs() int
	Close() error
}

// RateLimiter interface for rate limiting
type RateLimiter interface {
	Allow() bool
	Wait(ctx context.Context) error
	Limit() int
}

// ResolverConfig represents configuration for dependency resolver
type ResolverConfig struct {
	MaxConcurrency       int           `json:"maxConcurrency"`
	Timeout              time.Duration `json:"timeout"`
	RetryCount           int           `json:"retryCount"`
	CacheEnabled         bool          `json:"cacheEnabled"`
	CacheTTL             time.Duration `json:"cacheTTL"`
	RateLimitEnabled     bool          `json:"rateLimitEnabled"`
	RateLimit            int           `json:"rateLimit"`
	WorkerPoolSize       int           `json:"workerPoolSize"`
	EnableConflictResolution bool      `json:"enableConflictResolution"`
	
	// Strategy and solver configuration
	DefaultStrategy           ResolutionStrategy `json:"defaultStrategy"`
	MaxSolverIterations       int               `json:"maxSolverIterations"`
	MaxSolverBacktracks       int               `json:"maxSolverBacktracks"`
	EnableSolverHeuristics    bool              `json:"enableSolverHeuristics"`
	ParallelSolving           bool              `json:"parallelSolving"`
	EnableCaching             bool              `json:"enableCaching"`
	EnableConcurrency         bool              `json:"enableConcurrency"`
	WorkerCount               int               `json:"workerCount"`
	QueueSize                 int               `json:"queueSize"`
	
	// Version and prerelease handling
	PrereleaseStrategy        PrereleaseStrategy    `json:"prereleaseStrategy"`
	BuildMetadataStrategy     BuildMetadataStrategy `json:"buildMetadataStrategy"`
	StrictSemVer              bool                  `json:"strictSemVer"`
	
	// Machine learning and conflict prediction
	EnableMLConflictPrediction bool                           `json:"enableMLConflictPrediction"`
	ConflictStrategies         map[string]interface{}        `json:"conflictStrategies"`
	
	// Cache configuration
	CacheConfig               *CacheConfig          `json:"cacheConfig"`
	CacheCleanupInterval      time.Duration         `json:"cacheCleanupInterval"`
	MetricsCollectionInterval time.Duration         `json:"metricsCollectionInterval"`
	HealthCheckInterval       time.Duration         `json:"healthCheckInterval"`
	
	// Provider configurations
	GitConfig                 *GitConfig            `json:"gitConfig"`
	OCIConfig                 *OCIConfig            `json:"ociConfig"`
	HelmConfig                *HelmConfig           `json:"helmConfig"`
	LocalConfig               *LocalConfig          `json:"localConfig"`
}

// Configuration types for providers  
type CacheConfig struct {
	TTL             time.Duration `json:"ttl"`
	MaxEntries      int           `json:"maxEntries"`
	CleanupInterval time.Duration `json:"cleanupInterval"`
}

type GitConfig struct {
	DefaultBranch string `json:"defaultBranch"`
	Token         string `json:"token,omitempty"`
}

type OCIConfig struct {
	Registry string `json:"registry"`
	Token    string `json:"token,omitempty"`
}

type HelmConfig struct {
	Repository string `json:"repository"`
}

type LocalConfig struct {
	RootPath string `json:"rootPath"`
}

// Validate validates the resolver configuration
func (c *ResolverConfig) Validate() error {
	if c.MaxConcurrency <= 0 {
		return fmt.Errorf("max concurrency must be positive")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")  
	}
	if c.WorkerCount <= 0 {
		c.WorkerCount = 4 // Default
	}
	if c.QueueSize <= 0 {
		c.QueueSize = 100 // Default
	}
	return nil
}

// DefaultResolverConfig returns a default resolver configuration
func DefaultResolverConfig() *ResolverConfig {
	return &ResolverConfig{
		MaxConcurrency:             10,
		Timeout:                    30 * time.Second,
		RetryCount:                 3,
		CacheEnabled:               true,
		CacheTTL:                   1 * time.Hour,
		RateLimitEnabled:           true,
		RateLimit:                  100,
		WorkerPoolSize:             5,
		EnableConflictResolution:   true,
		DefaultStrategy:            ResolutionStrategy("stable"),
		MaxSolverIterations:        1000,
		MaxSolverBacktracks:        100,
		EnableSolverHeuristics:     true,
		ParallelSolving:            true,
		EnableCaching:              true,
		EnableConcurrency:          true,
		WorkerCount:                4,
		QueueSize:                  100,
		PrereleaseStrategy:         PrereleaseStrategy("ignore"),
		BuildMetadataStrategy:      BuildMetadataStrategy("ignore"),
		StrictSemVer:               true,
		EnableMLConflictPrediction: false,
		ConflictStrategies:         make(map[string]interface{}),
		CacheConfig: &CacheConfig{
			TTL:             1 * time.Hour,
			MaxEntries:      1000,
			CleanupInterval: 10 * time.Minute,
		},
		CacheCleanupInterval:      10 * time.Minute,
		MetricsCollectionInterval: 1 * time.Minute,
		HealthCheckInterval:       30 * time.Second,
		GitConfig: &GitConfig{
			DefaultBranch: "main",
		},
		OCIConfig: &OCIConfig{
			Registry: "ghcr.io",
		},
		HelmConfig: &HelmConfig{
			Repository: "https://charts.helm.sh/stable",
		},
		LocalConfig: &LocalConfig{
			RootPath: "./packages",
		},
	}
}

// NewResolverMetrics creates a new resolver metrics instance
func NewResolverMetrics() *ResolverMetrics {
	return &ResolverMetrics{
		ResolutionsTotal:        0,
		ResolutionsSuccessful:   0,
		ResolutionsFailed:       0,
		AverageResolutionTime:   0,
		CacheHitRate:            0.0,
		ConflictsTotal:          0,
		ConflictsResolved:       0,
		NetworkRequestsTotal:    0,
		NetworkRequestsFailed:   0,
		LastUpdated:             time.Now(),
		
		// Initialize prometheus metrics
		CacheHits:                    prometheus.NewCounter(prometheus.CounterOpts{Name: "cache_hits_total"}),
		CacheMisses:                  prometheus.NewCounter(prometheus.CounterOpts{Name: "cache_misses_total"}),
		ConstraintCacheHits:          prometheus.NewCounter(prometheus.CounterOpts{Name: "constraint_cache_hits_total"}),
		ConstraintCacheMisses:        prometheus.NewCounter(prometheus.CounterOpts{Name: "constraint_cache_misses_total"}),
		ConstraintSolvingTime:        prometheus.NewHistogram(prometheus.HistogramOpts{Name: "constraint_solving_duration_seconds"}),
		ConstraintSolvingSuccess:     prometheus.NewCounter(prometheus.CounterOpts{Name: "constraint_solving_success_total"}),
		ConstraintSolvingFailures:    prometheus.NewCounter(prometheus.CounterOpts{Name: "constraint_solving_failures_total"}),
		VersionResolutionTime:        prometheus.NewHistogram(prometheus.HistogramOpts{Name: "version_resolution_duration_seconds"}),
		VersionResolutionSuccess:     prometheus.NewCounter(prometheus.CounterOpts{Name: "version_resolution_success_total"}),
		VersionResolutionFailures:    prometheus.NewCounter(prometheus.CounterOpts{Name: "version_resolution_failures_total"}),
		ConflictDetectionTime:        prometheus.NewHistogram(prometheus.HistogramOpts{Name: "conflict_detection_duration_seconds"}),
		ConflictsDetected:            prometheus.NewCounter(prometheus.CounterOpts{Name: "conflicts_detected_total"}),
	}
}

// DependencyProvider interface for providing dependencies
type DependencyProvider interface {
	GetDependency(ctx context.Context, ref *PackageReference) (*PackageReference, error)
	ListVersions(ctx context.Context, name string) ([]string, error)
	GetMetadata(ctx context.Context, ref *PackageReference) (map[string]interface{}, error)
	Close() error
}

// UpdateConstraints represents constraints for updates
type UpdateConstraints struct {
	AllowMajorUpdates   bool      `json:"allowMajorUpdates"`
	AllowMinorUpdates   bool      `json:"allowMinorUpdates"`
	AllowPatchUpdates   bool      `json:"allowPatchUpdates"`
	AllowPrereleases    bool      `json:"allowPrereleases"`
	MaxVersionAge       time.Duration `json:"maxVersionAge,omitempty"`
	ExcludedVersions    []string  `json:"excludedVersions,omitempty"`
	RequiredTags        []string  `json:"requiredTags,omitempty"`
	SecurityConstraints *SecurityConstraints `json:"securityConstraints,omitempty"`
}

// CompatibilityRules represents rules for compatibility checking
type CompatibilityRules struct {
	Rules               []*CompatibilityRule `json:"rules"`
	StrictMode          bool                 `json:"strictMode"`
	BreakingChangeThreshold float64         `json:"breakingChangeThreshold"`
	CompatibilityMatrix map[string][]string  `json:"compatibilityMatrix,omitempty"`
}

// CompatibilityRule represents a single compatibility rule
type CompatibilityRule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Pattern     string   `json:"pattern"`
	Compatible  []string `json:"compatible"`
	Incompatible []string `json:"incompatible"`
	Priority    int      `json:"priority"`
}

// RolloutConfig represents configuration for rollout strategy
type RolloutConfig struct {
	Strategy            RolloutStrategy `json:"strategy"`
	BatchSize           int             `json:"batchSize"`
	BatchDelay          time.Duration   `json:"batchDelay"`
	MaxConcurrency      int             `json:"maxConcurrency"`
	FailureThreshold    float64         `json:"failureThreshold"`
	RollbackEnabled     bool            `json:"rollbackEnabled"`
	EnableStagedRollout bool            `json:"enableStagedRollout"`
	CanaryConfig        *CanaryConfig   `json:"canaryConfig,omitempty"`
}

// RolloutStrategy defines rollout strategies
type RolloutStrategy string

const (
	RolloutStrategyAll     RolloutStrategy = "all"
	RolloutStrategyBatched RolloutStrategy = "batched"
	RolloutStrategyCanary  RolloutStrategy = "canary"
	RolloutStrategyBlueGreen RolloutStrategy = "blue_green"
)

// CanaryConfig represents canary deployment configuration
type CanaryConfig struct {
	TrafficPercent   float64       `json:"trafficPercent"`
	Duration         time.Duration `json:"duration"`
	SuccessThreshold float64       `json:"successThreshold"`
	FailureThreshold float64       `json:"failureThreshold"`
	MetricsConfig    *MetricsConfig `json:"metricsConfig,omitempty"`
}

// MetricsConfig represents metrics configuration
type MetricsConfig struct {
	Enabled           bool          `json:"enabled"`
	CollectionInterval time.Duration `json:"collectionInterval"`
	Metrics           []string      `json:"metrics"`
	Thresholds        map[string]float64 `json:"thresholds,omitempty"`
}

// ValidationConfig represents validation configuration
type ValidationConfig struct {
	Enabled          bool          `json:"enabled"`
	Timeout          time.Duration `json:"timeout"`
	ValidationSteps  []*ValidationStep `json:"validationSteps"`
	FailFast         bool          `json:"failFast"`
	RetryOnFailure   bool          `json:"retryOnFailure"`
	RetryCount       int           `json:"retryCount"`
}

// ValidationStep represents a validation step
type ValidationStep struct {
	Name        string        `json:"name"`
	Type        string        `json:"type"`
	Command     string        `json:"command,omitempty"`
	Script      string        `json:"script,omitempty"`
	Timeout     time.Duration `json:"timeout"`
	Required    bool          `json:"required"`
	RetryCount  int           `json:"retryCount"`
}

// ApprovalPolicy represents approval policy for updates
type ApprovalPolicy struct {
	Required         bool              `json:"required"`
	Approvers        []string          `json:"approvers"`
	MinApprovals     int               `json:"minApprovals"`
	AutoApproveRules []*AutoApproveRule `json:"autoApproveRules,omitempty"`
	Timeout          time.Duration     `json:"timeout,omitempty"`
	EscalationPolicy *EscalationPolicy `json:"escalationPolicy,omitempty"`
}

// AutoApproveRule represents a rule for automatic approval
type AutoApproveRule struct {
	Name        string   `json:"name"`
	Conditions  []string `json:"conditions"`
	UpdateTypes []string `json:"updateTypes"`
	Packages    []string `json:"packages,omitempty"`
}

// EscalationPolicy represents escalation policy for approvals
type EscalationPolicy struct {
	Enabled      bool          `json:"enabled"`
	EscalateAfter time.Duration `json:"escalateAfter"`
	Escalators   []string      `json:"escalators"`
	MaxEscalations int          `json:"maxEscalations"`
}

// Very final missing types from updater.go

// UpdatedPackage represents a package that was successfully updated
type UpdatedPackage struct {
	Package     *PackageReference `json:"package"`
	OldVersion  string            `json:"oldVersion"`
	NewVersion  string            `json:"newVersion"`
	UpdatedAt   time.Time         `json:"updatedAt"`
	Duration    time.Duration     `json:"duration"`
	Changes     []string          `json:"changes,omitempty"`
	Impact      UpdateImpact      `json:"impact"`
}

// FailedUpdate represents a package that failed to update
type FailedUpdate struct {
	Package     *PackageReference `json:"package"`
	Error       string            `json:"error"`
	FailedAt    time.Time         `json:"failedAt"`
	Retryable   bool              `json:"retryable"`
	Attempts    int               `json:"attempts"`
	MaxAttempts int               `json:"maxAttempts"`
}

// SkippedUpdate represents a package that was skipped during update
type SkippedUpdate struct {
	Package   *PackageReference `json:"package"`
	Reason    string            `json:"reason"`
	SkippedAt time.Time         `json:"skippedAt"`
}

// RolloutExecution represents the execution of a rollout
type RolloutExecution struct {
	ID          string            `json:"id"`
	Status      RolloutStatus     `json:"status"`
	StartedAt   time.Time         `json:"startedAt"`
	CompletedAt *time.Time        `json:"completedAt,omitempty"`
	Duration    time.Duration     `json:"duration,omitempty"`
	Progress    RolloutProgress   `json:"progress"`
	Batches     []*BatchExecution `json:"batches"`
	Errors      []string          `json:"errors,omitempty"`
}

// Final missing impact types

// CompatibilityImpact represents compatibility impact of an update
type CompatibilityImpact struct {
	BreakingChanges     []string          `json:"breakingChanges,omitempty"`
	DeprecatedAPIs      []string          `json:"deprecatedAPIs,omitempty"`
	NewAPIs             []string          `json:"newAPIs,omitempty"`
	BackwardCompatible  bool              `json:"backwardCompatible"`
	ForwardCompatible   bool              `json:"forwardCompatible"`
	MigrationRequired   bool              `json:"migrationRequired"`
	MigrationSteps      []string          `json:"migrationSteps,omitempty"`
	CompatibilityScore  float64           `json:"compatibilityScore"`
}

// BusinessImpact represents business impact of an update
type BusinessImpact struct {
	CostImpact          float64   `json:"costImpact"`
	RevenueImpact       float64   `json:"revenueImpact"`
	UserExperienceImpact string   `json:"userExperienceImpact"`
	MaintenanceImpact   string    `json:"maintenanceImpact"`
	RiskLevel           string    `json:"riskLevel"`
	StakeholderImpact   []string  `json:"stakeholderImpact,omitempty"`
	BusinessCriticality string    `json:"businessCriticality"`
	RegulatoryImpact    []string  `json:"regulatoryImpact,omitempty"`
}

// BreakingChangeImpact represents impact of breaking changes
type BreakingChangeImpact struct {
	AffectedComponents  []string          `json:"affectedComponents"`
	SeverityLevel       string            `json:"severityLevel"`
	RequiredActions     []string          `json:"requiredActions"`
	AutomatableFixes    []string          `json:"automatableFixes,omitempty"`
	ManualInterventions []string          `json:"manualInterventions,omitempty"`
	EstimatedEffort     time.Duration     `json:"estimatedEffort"`
	ImpactScore         float64           `json:"impactScore"`
}

// MitigationStrategy represents strategy for mitigating update risks
type MitigationStrategy struct {
	Type                string            `json:"type"`
	Description         string            `json:"description"`
	Steps               []string          `json:"steps"`
	RiskReduction       float64           `json:"riskReduction"`
	ImplementationTime  time.Duration     `json:"implementationTime"`
	Prerequisites       []string          `json:"prerequisites,omitempty"`
	Monitoring          []string          `json:"monitoring,omitempty"`
	RollbackPlan        *RollbackPlan     `json:"rollbackPlan,omitempty"`
}

// Absolutely final missing types

// TestingRecommendation represents testing recommendations for an update
type TestingRecommendation struct {
	Type            string        `json:"type"`
	Description     string        `json:"description"`
	TestCategories  []string      `json:"testCategories"`
	RequiredTests   []string      `json:"requiredTests"`
	OptionalTests   []string      `json:"optionalTests,omitempty"`
	EstimatedTime   time.Duration `json:"estimatedTime"`
	Tools           []string      `json:"tools,omitempty"`
	Environment     string        `json:"environment,omitempty"`
}

// CompatibilityResult represents compatibility validation result
type CompatibilityResult struct {
	Compatible      bool                 `json:"compatible"`
	Score           float64              `json:"score"`
	Issues          []*CompatibilityIssue `json:"issues,omitempty"`
	Recommendations []string             `json:"recommendations,omitempty"`
	TestedWith      []string             `json:"testedWith,omitempty"`
	ValidatedAt     time.Time            `json:"validatedAt"`
}

// CompatibilityIssue represents a compatibility issue
type CompatibilityIssue struct {
	Type        string   `json:"type"`
	Severity    string   `json:"severity"`
	Description string   `json:"description"`
	Components  []string `json:"components,omitempty"`
	Resolution  string   `json:"resolution,omitempty"`
}

// LicenseValidation represents license validation result
type LicenseValidation struct {
	Valid           bool              `json:"valid"`
	License         *LicenseInfo      `json:"license"`
	Issues          []*LicenseIssue   `json:"issues,omitempty"`
	Recommendations []string          `json:"recommendations,omitempty"`
	ValidatedAt     time.Time         `json:"validatedAt"`
}

// LicenseIssue represents a license issue
type LicenseIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Resolution  string `json:"resolution,omitempty"`
}

// ComplianceValidation represents compliance validation result
type ComplianceValidation struct {
	Compliant       bool                      `json:"compliant"`
	Score           float64                   `json:"score"`
	Violations      []*ComplianceViolation    `json:"violations,omitempty"`
	Standards       []ComplianceStandard      `json:"standards"`
	Recommendations []string                  `json:"recommendations,omitempty"`
	ValidatedAt     time.Time                 `json:"validatedAt"`
}

// PerformanceValidation represents performance validation result
type PerformanceValidation struct {
	Valid           bool                      `json:"valid"`
	Score           float64                   `json:"score"`
	Metrics         *PerformanceMetrics       `json:"metrics"`
	Benchmarks      []*PerformanceBenchmark   `json:"benchmarks,omitempty"`
	Issues          []*PerformanceIssue       `json:"issues,omitempty"`
	Recommendations []string                  `json:"recommendations,omitempty"`
	ValidatedAt     time.Time                 `json:"validatedAt"`
}

// PerformanceIssue represents a performance issue
type PerformanceIssue struct {
	Type        string  `json:"type"`
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
	Metric      string  `json:"metric"`
	Threshold   float64 `json:"threshold"`
	ActualValue float64 `json:"actualValue"`
	Resolution  string  `json:"resolution,omitempty"`
}

// VersionValidation represents version validation result
type VersionValidation struct {
	Valid           bool                  `json:"valid"`
	Version         string                `json:"version"`
	Issues          []*VersionIssue       `json:"issues,omitempty"`
	Recommendations []string              `json:"recommendations,omitempty"`
	ValidatedAt     time.Time             `json:"validatedAt"`
}

// VersionIssue represents a version issue
type VersionIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Resolution  string `json:"resolution,omitempty"`
}

// PackageSecurityValidation represents package security validation
type PackageSecurityValidation struct {
	Secure          bool                    `json:"secure"`
	SecurityScore   float64                 `json:"securityScore"`
	Vulnerabilities []*SecurityVulnerability `json:"vulnerabilities,omitempty"`
	SecurityInfo    *SecurityInfo           `json:"securityInfo"`
	Recommendations []string                `json:"recommendations,omitempty"`
	ValidatedAt     time.Time               `json:"validatedAt"`
}

// SecurityVulnerability represents a security vulnerability
type SecurityVulnerability struct {
	ID          string    `json:"id"`
	CVE         string    `json:"cve,omitempty"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	FixedIn     string    `json:"fixedIn,omitempty"`
	References  []string  `json:"references,omitempty"`
	PublishedAt time.Time `json:"publishedAt,omitempty"`
}

// PackageLicenseValidation represents package license validation
type PackageLicenseValidation struct {
	Valid           bool           `json:"valid"`
	LicenseInfo     *LicenseInfo   `json:"licenseInfo"`
	Issues          []*LicenseIssue `json:"issues,omitempty"`
	Recommendations []string       `json:"recommendations,omitempty"`
	ValidatedAt     time.Time      `json:"validatedAt"`
}

// QualityValidation represents quality validation result
type QualityValidation struct {
	Valid           bool               `json:"valid"`
	QualityScore    float64            `json:"qualityScore"`
	Metrics         *QualityMetrics    `json:"metrics"`
	Issues          []*QualityIssue    `json:"issues,omitempty"`
	Recommendations []string           `json:"recommendations,omitempty"`
	ValidatedAt     time.Time          `json:"validatedAt"`
}

// QualityIssue represents a quality issue
type QualityIssue struct {
	Type        string  `json:"type"`
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
	Metric      string  `json:"metric,omitempty"`
	Threshold   float64 `json:"threshold,omitempty"`
	ActualValue float64 `json:"actualValue,omitempty"`
	Resolution  string  `json:"resolution,omitempty"`
}

// PolicyViolation represents a policy violation
type PolicyViolation struct {
	PolicyName    string                 `json:"policyName"`
	RuleName      string                 `json:"ruleName"`
	Severity      PolicySeverity         `json:"severity"`
	Description   string                 `json:"description"`
	Package       *PackageReference      `json:"package,omitempty"`
	ViolatedValue interface{}            `json:"violatedValue,omitempty"`
	ExpectedValue interface{}            `json:"expectedValue,omitempty"`
	Action        PolicyAction           `json:"action"`
	Exemptions    []string               `json:"exemptions,omitempty"`
	DetectedAt    time.Time              `json:"detectedAt"`
}

// Truly final batch of missing types

// PolicyValidation represents policy validation result
type PolicyValidation struct {
	Valid           bool                `json:"valid"`
	Score           float64             `json:"score"`
	Violations      []*PolicyViolation  `json:"violations,omitempty"`
	Policies        []*SecurityPolicy   `json:"policies"`
	Recommendations []string            `json:"recommendations,omitempty"`
	ValidatedAt     time.Time           `json:"validatedAt"`
}

// ApprovalRequest represents a request for approval
type ApprovalRequest struct {
	ID              string            `json:"id"`
	Type            string            `json:"type"`
	Requester       string            `json:"requester"`
	Package         *PackageReference `json:"package,omitempty"`
	Changes         []string          `json:"changes"`
	Justification   string            `json:"justification"`
	Impact          string            `json:"impact"`
	Priority        string            `json:"priority"`
	RequestedAt     time.Time         `json:"requestedAt"`
	Deadline        *time.Time        `json:"deadline,omitempty"`
	Status          ApprovalStatus    `json:"status"`
	Approvers       []string          `json:"approvers"`
	ApprovalHistory []*ApprovalAction `json:"approvalHistory,omitempty"`
}

// ApprovalStatus defines approval statuses
type ApprovalStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusRejected ApprovalStatus = "rejected"
	ApprovalStatusExpired  ApprovalStatus = "expired"
)

// ApprovalAction represents an approval action
type ApprovalAction struct {
	Approver  string         `json:"approver"`
	Action    ApprovalStatus `json:"action"`
	Comment   string         `json:"comment,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}

// UpdateError represents an update error
type UpdateError struct {
	Code        string            `json:"code"`
	Type        string            `json:"type"`
	Message     string            `json:"message"`
	Package     *PackageReference `json:"package,omitempty"`
	Component   string            `json:"component,omitempty"`
	Stack       string            `json:"stack,omitempty"`
	Recoverable bool              `json:"recoverable"`
	Timestamp   time.Time         `json:"timestamp"`
}

// UpdateWarning represents an update warning
type UpdateWarning struct {
	Code      string            `json:"code"`
	Type      string            `json:"type"`
	Message   string            `json:"message"`
	Package   *PackageReference `json:"package,omitempty"`
	Component string            `json:"component,omitempty"`
	Severity  string            `json:"severity"`
	Timestamp time.Time         `json:"timestamp"`
}

// UpdateStatistics represents update statistics
type UpdateStatistics struct {
	TotalPackages       int           `json:"totalPackages"`
	UpdatedPackages     int           `json:"updatedPackages"`
	FailedPackages      int           `json:"failedPackages"`
	SkippedPackages     int           `json:"skippedPackages"`
	TotalUpdateTime     time.Duration `json:"totalUpdateTime"`
	AverageUpdateTime   time.Duration `json:"averageUpdateTime"`
	SuccessRate         float64       `json:"successRate"`
	ConflictsResolved   int           `json:"conflictsResolved"`
	TestsExecuted       int           `json:"testsExecuted"`
	TestSuccessRate     float64       `json:"testSuccessRate"`
}

// PropagationFilter represents filters for update propagation
type PropagationFilter struct {
	IncludeEnvironments []string          `json:"includeEnvironments,omitempty"`
	ExcludeEnvironments []string          `json:"excludeEnvironments,omitempty"`
	PackageFilters      []*PackageFilter  `json:"packageFilters,omitempty"`
	UpdateTypes         []string          `json:"updateTypes,omitempty"`
	MinSuccessRate      float64           `json:"minSuccessRate,omitempty"`
	RequireApproval     bool              `json:"requireApproval"`
	CustomRules         []string          `json:"customRules,omitempty"`
}

// PropagatedUpdate represents a propagated update
type PropagatedUpdate struct {
	Package       *PackageReference `json:"package"`
	Environment   string            `json:"environment"`
	Status        PropagationStatus `json:"status"`
	StartedAt     time.Time         `json:"startedAt"`
	CompletedAt   *time.Time        `json:"completedAt,omitempty"`
	Duration      time.Duration     `json:"duration,omitempty"`
	Changes       []string          `json:"changes,omitempty"`
	TestResults   []string          `json:"testResults,omitempty"`
}

// PropagationStatus defines propagation statuses
type PropagationStatus string

const (
	PropagationStatusPending   PropagationStatus = "pending"
	PropagationStatusRunning   PropagationStatus = "running"
	PropagationStatusCompleted PropagationStatus = "completed"
	PropagationStatusFailed    PropagationStatus = "failed"
	PropagationStatusSkipped   PropagationStatus = "skipped"
)

// FailedPropagation represents a failed propagation
type FailedPropagation struct {
	Package     *PackageReference `json:"package"`
	Environment string            `json:"environment"`
	Error       string            `json:"error"`
	FailedAt    time.Time         `json:"failedAt"`
	Retryable   bool              `json:"retryable"`
	Attempts    int               `json:"attempts"`
}

// SkippedPropagation represents a skipped propagation
type SkippedPropagation struct {
	Package     *PackageReference `json:"package"`
	Environment string            `json:"environment"`
	Reason      string            `json:"reason"`
	SkippedAt   time.Time         `json:"skippedAt"`
}

// EnvironmentUpdateResult represents the result of updates in an environment
type EnvironmentUpdateResult struct {
	Environment     string               `json:"environment"`
	Status          EnvironmentStatus    `json:"status"`
	UpdatedPackages []*PropagatedUpdate  `json:"updatedPackages"`
	FailedUpdates   []*FailedPropagation `json:"failedUpdates,omitempty"`
	SkippedUpdates  []*SkippedPropagation `json:"skippedUpdates,omitempty"`
	StartedAt       time.Time            `json:"startedAt"`
	CompletedAt     *time.Time           `json:"completedAt,omitempty"`
	Duration        time.Duration        `json:"duration,omitempty"`
	SuccessRate     float64              `json:"successRate"`
}

// EnvironmentStatus defines environment statuses
type EnvironmentStatus string

const (
	EnvironmentStatusPending   EnvironmentStatus = "pending"
	EnvironmentStatusRunning   EnvironmentStatus = "running" 
	EnvironmentStatusCompleted EnvironmentStatus = "completed"
	EnvironmentStatusFailed    EnvironmentStatus = "failed"
)

// Last batch of missing types

// UpdateSchedule represents a schedule for updates
type UpdateSchedule interface {
	Next(now time.Time) time.Time
	ShouldUpdate(now time.Time) bool
	Description() string
}

// ScheduledUpdate represents a scheduled update
type ScheduledUpdate struct {
	ID            string              `json:"id"`
	Package       *PackageReference   `json:"package"`
	Schedule      string              `json:"schedule"`
	NextUpdate    time.Time           `json:"nextUpdate"`
	LastUpdate    *time.Time          `json:"lastUpdate,omitempty"`
	Status        ScheduleStatus      `json:"status"`
	Config        *UpdateConfig       `json:"config"`
	CreatedAt     time.Time           `json:"createdAt"`
	UpdatedAt     time.Time           `json:"updatedAt"`
}

// ScheduleStatus defines schedule statuses
type ScheduleStatus string

const (
	ScheduleStatusActive   ScheduleStatus = "active"
	ScheduleStatusPaused   ScheduleStatus = "paused"
	ScheduleStatusDisabled ScheduleStatus = "disabled"
)

// ValidatorConfig holds validation configuration
type ValidatorConfig struct {
	EnableCompatibilityCheck bool                     `json:"enableCompatibilityCheck"`
	EnableSecurityScan      bool                     `json:"enableSecurityScan"`
	EnableLicenseValidation bool                     `json:"enableLicenseValidation"`
	SecurityScanTimeout     time.Duration            `json:"securityScanTimeout"`
	ValidationTimeout       time.Duration            `json:"validationTimeout"`
	CacheEnabled           bool                     `json:"cacheEnabled"`
	CacheTTL               time.Duration            `json:"cacheTTL"`
	PolicyRules            []string                 `json:"policyRules,omitempty"`
}

// CompatibilityChecker provides compatibility checking functionality
type CompatibilityChecker struct {
	logger logr.Logger
	config *ValidatorConfig
}

// SecurityScanner provides security scanning functionality  
type SecurityScanner struct {
	logger logr.Logger
	config *ValidatorConfig
}

// LicenseValidator provides license validation functionality
type LicenseValidator struct {
	logger logr.Logger
	config *ValidatorConfig
}

// PolicyEngine provides policy validation functionality
type PolicyEngine struct {
	logger logr.Logger
	config *ValidatorConfig
}

// Additional missing validator types
type ConflictAnalyzer struct {
	logger logr.Logger
}

type ValidationCache struct {
	cache map[string]interface{}
}

type ScanResultCache struct {
	cache map[string]interface{}
}

type VulnerabilityDatabase interface {
	ScanPackage(packageName, version string) ([]Vulnerability, error)
}

type LicenseDatabase interface {
	GetLicense(packageName, version string) (*License, error)
}

type PolicyRegistry interface {
	GetPolicies() ([]Policy, error)
	ValidateAgainstPolicy(pkg Package, policy Policy) error
}

// Vulnerability type already defined in validator.go - removing duplicate

type License struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Permissions []string `json:"permissions"`
	Conditions  []string `json:"conditions"`
	Limitations []string `json:"limitations"`
}

type Policy struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Rules       []PolicyRule           `json:"rules"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// PolicyRule type already defined at line 2362 - removing duplicate

type Package struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description,omitempty"`
	License     string            `json:"license,omitempty"`
	Repository  string            `json:"repository,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// UpdateConfig represents update configuration
type UpdateConfig struct {
	Strategy            UpdateStrategy        `json:"strategy"`
	Constraints         *UpdateConstraints    `json:"constraints,omitempty"`
	ValidationConfig    *ValidationConfig     `json:"validationConfig,omitempty"`
	RolloutConfig       *RolloutConfig        `json:"rolloutConfig,omitempty"`
	ApprovalConfig      *ApprovalPolicy       `json:"approvalConfig,omitempty"`
	NotificationConfig  *NotificationConfig   `json:"notificationConfig,omitempty"`
}

// UpdatePlan represents a plan for updates
type UpdatePlan struct {
	ID                string               `json:"id"`
	Description       string               `json:"description"`
	Updates           []*PlannedUpdate     `json:"updates"`
	UpdateSteps       []*UpdateStep        `json:"updateSteps,omitempty"`
	Dependencies      []*UpdateDependency  `json:"dependencies,omitempty"`
	EstimatedDuration time.Duration        `json:"estimatedDuration"`
	RiskAssessment    *RiskAssessment      `json:"riskAssessment,omitempty"`
	RollbackPlan      *RollbackPlan        `json:"rollbackPlan,omitempty"`
	CreatedAt         time.Time            `json:"createdAt"`
	CreatedBy         string               `json:"createdBy"`
	Status            PlanStatus           `json:"status"`
}

// PlannedUpdate represents a planned update
type PlannedUpdate struct {
	Package         *PackageReference `json:"package"`
	FromVersion     string            `json:"fromVersion"`
	ToVersion       string            `json:"toVersion"`
	Priority        string            `json:"priority"`
	EstimatedTime   time.Duration     `json:"estimatedTime"`
	Dependencies    []string          `json:"dependencies,omitempty"`
	Risks           []string          `json:"risks,omitempty"`
}

// UpdateDependency represents a dependency between updates
type UpdateDependency struct {
	From        string            `json:"from"`
	To          string            `json:"to"`
	Type        string            `json:"type"`
	Description string            `json:"description,omitempty"`
}

// PlanStatus defines plan statuses
type PlanStatus string

const (
	PlanStatusDraft    PlanStatus = "draft"
	PlanStatusReview   PlanStatus = "review"
	PlanStatusApproved PlanStatus = "approved"
	PlanStatusExecuting PlanStatus = "executing"
	PlanStatusCompleted PlanStatus = "completed"
	PlanStatusFailed   PlanStatus = "failed"
	PlanStatusCancelled PlanStatus = "cancelled"
)

// RiskAssessment represents risk assessment for updates
type RiskAssessment struct {
	OverallRisk     RiskLevel       `json:"overallRisk"`
	RiskFactors     []*RiskFactor   `json:"riskFactors"`
	RiskScore       float64         `json:"riskScore"`
	MitigationSteps []string        `json:"mitigationSteps,omitempty"`
	AssessedAt      time.Time       `json:"assessedAt"`
	AssessedBy      string          `json:"assessedBy,omitempty"`
}

// PropagationError represents a propagation error
type PropagationError struct {
	Environment string            `json:"environment"`
	Package     *PackageReference `json:"package"`
	Error       string            `json:"error"`
	ErrorCode   string            `json:"errorCode"`
	Timestamp   time.Time         `json:"timestamp"`
	Stack       string            `json:"stack,omitempty"`
	Retryable   bool              `json:"retryable"`
}

// PropagationWarning represents a propagation warning
type PropagationWarning struct {
	Environment string            `json:"environment"`
	Package     *PackageReference `json:"package"`
	Warning     string            `json:"warning"`
	WarningCode string            `json:"warningCode"`
	Timestamp   time.Time         `json:"timestamp"`
	Severity    string            `json:"severity"`
}

// Absolutely final missing types

// PlanValidation represents validation of an update plan
type PlanValidation struct {
	Valid               bool                  `json:"valid"`
	ValidationScore     float64               `json:"validationScore"`
	Issues              []*PlanValidationIssue `json:"issues,omitempty"`
	Warnings            []*PlanValidationWarning `json:"warnings,omitempty"`
	Recommendations     []string              `json:"recommendations,omitempty"`
	EstimatedRisk       RiskLevel             `json:"estimatedRisk"`
	ValidatedAt         time.Time             `json:"validatedAt"`
	ValidatedBy         string                `json:"validatedBy,omitempty"`
}

// PlanValidationIssue represents an issue in plan validation
type PlanValidationIssue struct {
	Type        string   `json:"type"`
	Severity    string   `json:"severity"`
	Description string   `json:"description"`
	Updates     []string `json:"updates,omitempty"`
	Resolution  string   `json:"resolution,omitempty"`
}

// PlanValidationWarning represents a warning in plan validation
type PlanValidationWarning struct {
	Type        string   `json:"type"`
	Message     string   `json:"message"`
	Updates     []string `json:"updates,omitempty"`
	Suggestion  string   `json:"suggestion,omitempty"`
}

// RollbackValidation represents validation of a rollback operation
type RollbackValidation struct {
	Valid           bool                      `json:"valid"`
	ValidationScore float64                   `json:"validationScore"`
	Issues          []*RollbackValidationIssue `json:"issues,omitempty"`
	Warnings        []*RollbackValidationWarning `json:"warnings,omitempty"`
	EstimatedTime   time.Duration             `json:"estimatedTime"`
	RiskAssessment  *RiskAssessment           `json:"riskAssessment,omitempty"`
	ValidatedAt     time.Time                 `json:"validatedAt"`
}

// RollbackValidationIssue represents an issue in rollback validation
type RollbackValidationIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Step        string `json:"step,omitempty"`
	Resolution  string `json:"resolution,omitempty"`
}

// RollbackValidationWarning represents a warning in rollback validation
type RollbackValidationWarning struct {
	Type        string `json:"type"`
	Message     string `json:"message"`
	Step        string `json:"step,omitempty"`
	Suggestion  string `json:"suggestion,omitempty"`
}

// UpdateHistoryFilter represents filters for update history
type UpdateHistoryFilter struct {
	Packages        []string      `json:"packages,omitempty"`
	Environments    []string      `json:"environments,omitempty"`
	TimeRange       *TimeRange    `json:"timeRange,omitempty"`
	Status          []string      `json:"status,omitempty"`
	UpdateTypes     []string      `json:"updateTypes,omitempty"`
	Initiators      []string      `json:"initiators,omitempty"`
	MinDuration     time.Duration `json:"minDuration,omitempty"`
	MaxDuration     time.Duration `json:"maxDuration,omitempty"`
	IncludeRollbacks bool         `json:"includeRollbacks"`
}

// UpdateRecord represents a record of an update operation
type UpdateRecord struct {
	ID              string            `json:"id"`
	Package         *PackageReference `json:"package"`
	Environment     string            `json:"environment,omitempty"`
	FromVersion     string            `json:"fromVersion"`
	ToVersion       string            `json:"toVersion"`
	Status          UpdateStatus      `json:"status"`
	StartedAt       time.Time         `json:"startedAt"`
	CompletedAt     *time.Time        `json:"completedAt,omitempty"`
	Duration        time.Duration     `json:"duration,omitempty"`
	InitiatedBy     string            `json:"initiatedBy"`
	UpdateType      UpdateType        `json:"updateType"`
	Changes         []string          `json:"changes,omitempty"`
	TestResults     []string          `json:"testResults,omitempty"`
	RollbackRecord  *RollbackRecord   `json:"rollbackRecord,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateStatus defines update statuses
type UpdateStatus string

const (
	UpdateStatusPending   UpdateStatus = "pending"
	UpdateStatusRunning   UpdateStatus = "running"
	UpdateStatusCompleted UpdateStatus = "completed"
	UpdateStatusFailed    UpdateStatus = "failed"
	UpdateStatusRolledBack UpdateStatus = "rolled_back"
)

// RollbackRecord represents a record of a rollback operation
type RollbackRecord struct {
	ID          string        `json:"id"`
	Reason      string        `json:"reason"`
	StartedAt   time.Time     `json:"startedAt"`
	CompletedAt *time.Time    `json:"completedAt,omitempty"`
	Duration    time.Duration `json:"duration,omitempty"`
	Success     bool          `json:"success"`
	Steps       []string      `json:"steps,omitempty"`
	Errors      []string      `json:"errors,omitempty"`
}

// The truly absolutely final missing types

// ChangeReport represents a report of changes between versions
type ChangeReport struct {
	Package         *PackageReference `json:"package"`
	FromVersion     string            `json:"fromVersion"`
	ToVersion       string            `json:"toVersion"`
	Changes         []*Change         `json:"changes"`
	BreakingChanges []*BreakingChange `json:"breakingChanges,omitempty"`
	NewFeatures     []string          `json:"newFeatures,omitempty"`
	BugFixes        []string          `json:"bugFixes,omitempty"`
	SecurityFixes   []string          `json:"securityFixes,omitempty"`
	Dependencies    []*DependencyChange `json:"dependencies,omitempty"`
	GeneratedAt     time.Time         `json:"generatedAt"`
}

// Change represents a single change
type Change struct {
	Type        ChangeType `json:"type"`
	Description string     `json:"description"`
	Component   string     `json:"component,omitempty"`
	Impact      string     `json:"impact,omitempty"`
	Breaking    bool       `json:"breaking"`
}

// DependencyChange represents a change in dependencies
type DependencyChange struct {
	Package     *PackageReference `json:"package"`
	ChangeType  string            `json:"changeType"`
	OldVersion  string            `json:"oldVersion,omitempty"`
	NewVersion  string            `json:"newVersion,omitempty"`
	Description string            `json:"description,omitempty"`
}

// ApprovalFilter represents filters for approval requests
type ApprovalFilter struct {
	Status         []ApprovalStatus `json:"status,omitempty"`
	Requester      []string         `json:"requester,omitempty"`
	Approver       []string         `json:"approver,omitempty"`
	TimeRange      *TimeRange       `json:"timeRange,omitempty"`
	Packages       []string         `json:"packages,omitempty"`
	Priority       []string         `json:"priority,omitempty"`
	PendingOnly    bool             `json:"pendingOnly"`
	ExpiringSoon   bool             `json:"expiringSoon"`
	ExpiredOnly    bool             `json:"expiredOnly"`
}

// UpdateNotification represents a notification about an update
type UpdateNotification struct {
	ID          string                 `json:"id"`
	Type        NotificationType       `json:"type"`
	Subject     string                 `json:"subject"`
	Message     string                 `json:"message"`
	Package     *PackageReference      `json:"package,omitempty"`
	Recipient   string                 `json:"recipient"`
	Channel     string                 `json:"channel"`
	Status      NotificationStatus     `json:"status"`
	Priority    NotificationPriority   `json:"priority"`
	Timestamp   time.Time              `json:"timestamp"`
	SentAt      *time.Time             `json:"sentAt,omitempty"`
	ReadAt      *time.Time             `json:"readAt,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NotificationType defines notification types
type NotificationType string

const (
	NotificationTypeUpdateAvailable   NotificationType = "update_available"
	NotificationTypeUpdateStarted     NotificationType = "update_started"
	NotificationTypeUpdateCompleted   NotificationType = "update_completed"
	NotificationTypeUpdateFailed      NotificationType = "update_failed"
	NotificationTypeApprovalRequired  NotificationType = "approval_required"
	NotificationTypeSecurityAlert     NotificationType = "security_alert"
)

// NotificationStatus defines notification statuses
type NotificationStatus string

const (
	NotificationStatusPending   NotificationStatus = "pending"
	NotificationStatusSent      NotificationStatus = "sent"
	NotificationStatusRead      NotificationStatus = "read"
	NotificationStatusFailed    NotificationStatus = "failed"
	NotificationStatusCancelled NotificationStatus = "cancelled"
)

// NotificationPriority defines notification priorities
type NotificationPriority string

const (
	NotificationPriorityLow      NotificationPriority = "low"
	NotificationPriorityNormal   NotificationPriority = "normal"
	NotificationPriorityHigh     NotificationPriority = "high"
	NotificationPriorityCritical NotificationPriority = "critical"
)

// UpdaterHealth represents health status of the updater
type UpdaterHealth struct {
	Status              string            `json:"status"`
	Components          map[string]string `json:"components"`
	LastCheck           time.Time         `json:"lastCheck"`
	Issues              []string          `json:"issues,omitempty"`
	UpTime              time.Duration     `json:"upTime"`
	ActiveUpdates       int               `json:"activeUpdates"`
	QueuedUpdates       int               `json:"queuedUpdates"`
	ScheduledUpdates    int               `json:"scheduledUpdates"`
	LastSuccessfulUpdate time.Time        `json:"lastSuccessfulUpdate,omitempty"`
}

// UpdaterMetrics represents metrics for the updater
type UpdaterMetrics struct {
	TotalUpdates          int64         `json:"totalUpdates"`
	SuccessfulUpdates     int64         `json:"successfulUpdates"`
	FailedUpdates         int64         `json:"failedUpdates"`
	SkippedUpdates        int64         `json:"skippedUpdates"`
	AverageUpdateTime     time.Duration `json:"averageUpdateTime"`
	UpdatesPerHour        float64       `json:"updatesPerHour"`
	QueueSize             int           `json:"queueSize"`
	ActiveWorkers         int           `json:"activeWorkers"`
	ThroughputPPS         float64       `json:"throughputPPS"` // Packages per second
	ErrorRate             float64       `json:"errorRate"`
	LastUpdated           time.Time     `json:"lastUpdated"`
	
	// Prometheus metrics for impact analysis
	ImpactAnalysisTime    *prometheus.HistogramVec `json:"-"`
	ImpactAnalysisTotal   *prometheus.CounterVec   `json:"-"`
}

// Additional missing types for stub implementations

type CrossUpdateImpact struct {
	Updates      []*DependencyUpdate `json:"updates"`
	Conflicts    []*UpdateConflict   `json:"conflicts,omitempty"`
	Dependencies []*CrossDependency  `json:"dependencies,omitempty"`
	RiskLevel    string              `json:"riskLevel"`
}

type UpdateConflict struct {
	ID           string             `json:"id"`
	Update1      *DependencyUpdate  `json:"update1"`
	Update2      *DependencyUpdate  `json:"update2"`
	ConflictType string             `json:"conflictType"`
	Description  string             `json:"description"`
}

type CrossDependency struct {
	ID           string            `json:"id"`
	Source       *PackageReference `json:"source"`
	Target       *PackageReference `json:"target"`
	Relationship string            `json:"relationship"`
}

type RolloutStage struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Status      RolloutStatus `json:"status"`
	StartedAt   *time.Time    `json:"startedAt,omitempty"`
	CompletedAt *time.Time    `json:"completedAt,omitempty"`
}

// UpdateAnalysisResult represents the result of analyzing a single update
type UpdateAnalysisResult struct {
	UpdateID  string            `json:"updateId"`
	Package   *PackageReference `json:"package"`
	Impact    UpdateImpact      `json:"impact"`
	RiskLevel string            `json:"riskLevel"`
}


// The very last missing types from validator.go

// PlatformValidation represents platform validation result
type PlatformValidation struct {
	Valid           bool                     `json:"valid"`
	Score           float64                  `json:"score"`
	Issues          []*PlatformValidationIssue `json:"issues,omitempty"`
	Recommendations []string                 `json:"recommendations,omitempty"`
	SupportedPlatforms []string              `json:"supportedPlatforms"`
	ValidatedAt     time.Time                `json:"validatedAt"`
}

// PlatformValidationIssue represents a platform validation issue
type PlatformValidationIssue struct {
	Platform    string `json:"platform"`
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Resolution  string `json:"resolution,omitempty"`
}

// SecurityPolicies represents security policies
type SecurityPolicies struct {
	Policies        []*SecurityPolicy `json:"policies"`
	DefaultPolicy   *SecurityPolicy   `json:"defaultPolicy,omitempty"`
	EnforcementMode PolicyEnforcement `json:"enforcementMode"`
	UpdatedAt       time.Time         `json:"updatedAt"`
}

// ComplianceRules represents compliance rules
type ComplianceRules struct {
	Rules         []*ComplianceRule `json:"rules"`
	Standards     []string          `json:"standards"`
	Required      bool              `json:"required"`
	EnforcementMode string          `json:"enforcementMode"`
	UpdatedAt     time.Time         `json:"updatedAt"`
}

// ComplianceRule represents a compliance rule
type ComplianceRule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Standard    string   `json:"standard"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	Severity    string   `json:"severity"`
	Pattern     string   `json:"pattern,omitempty"`
	Required    bool     `json:"required"`
	Tags        []string `json:"tags,omitempty"`
}

// ResourceLimits represents resource limits
type ResourceLimits struct {
	CPU     *ResourceLimit `json:"cpu,omitempty"`
	Memory  *ResourceLimit `json:"memory,omitempty"`
	Disk    *ResourceLimit `json:"disk,omitempty"`
	Network *ResourceLimit `json:"network,omitempty"`
	GPU     *ResourceLimit `json:"gpu,omitempty"`
}

// ResourceLimit represents a limit on a resource
type ResourceLimit struct {
	Min     float64 `json:"min,omitempty"`
	Max     float64 `json:"max,omitempty"`
	Default float64 `json:"default,omitempty"`
	Unit    string  `json:"unit"`
}

// OrganizationalPolicies represents organizational policies
type OrganizationalPolicies struct {
	Policies         []*OrganizationalPolicy `json:"policies"`
	DefaultPolicies  []*OrganizationalPolicy `json:"defaultPolicies,omitempty"`
	InheritanceRules []string                `json:"inheritanceRules,omitempty"`
	OverrideRules    []string                `json:"overrideRules,omitempty"`
	UpdatedAt        time.Time               `json:"updatedAt"`
}

// OrganizationalPolicy represents an organizational policy
type OrganizationalPolicy struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	Scope       []string `json:"scope"`
	Rules       []string `json:"rules"`
	Required    bool     `json:"required"`
	Exceptions  []string `json:"exceptions,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// The absolutely final missing types from validator.go

// SecurityValidation represents security validation result
type SecurityValidation struct {
	Valid           bool                      `json:"valid"`
	SecurityScore   float64                   `json:"securityScore"`
	Vulnerabilities []*SecurityVulnerability  `json:"vulnerabilities,omitempty"`
	SecurityInfo    *SecurityInfo             `json:"securityInfo"`
	PolicyViolations []*SecurityPolicyViolation `json:"policyViolations,omitempty"`
	Recommendations []string                  `json:"recommendations,omitempty"`
	ValidatedAt     time.Time                 `json:"validatedAt"`
}

// SecurityPolicyViolation represents a security policy violation
type SecurityPolicyViolation struct {
	PolicyName  string `json:"policyName"`
	RuleName    string `json:"ruleName"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Resolution  string `json:"resolution,omitempty"`
}

// ResourceValidation represents resource validation result
type ResourceValidation struct {
	Valid           bool                     `json:"valid"`
	Score           float64                  `json:"score"`
	ResourceUsage   *ResourceUsage           `json:"resourceUsage"`
	Limits          *ResourceLimits          `json:"limits,omitempty"`
	Issues          []*ResourceValidationIssue `json:"issues,omitempty"`
	Recommendations []string                 `json:"recommendations,omitempty"`
	ValidatedAt     time.Time                `json:"validatedAt"`
}

// ResourceValidationIssue represents a resource validation issue
type ResourceValidationIssue struct {
	ResourceType string  `json:"resourceType"`
	Type         string  `json:"type"`
	Severity     string  `json:"severity"`
	Description  string  `json:"description"`
	CurrentValue float64 `json:"currentValue"`
	LimitValue   float64 `json:"limitValue,omitempty"`
	Resolution   string  `json:"resolution,omitempty"`
}

// BreakingChangeReport represents a breaking change report
type BreakingChangeReport struct {
	Package         *PackageReference  `json:"package"`
	FromVersion     string             `json:"fromVersion"`
	ToVersion       string             `json:"toVersion"`
	BreakingChanges []*BreakingChange  `json:"breakingChanges"`
	ImpactAnalysis  *BreakingChangeImpact `json:"impactAnalysis"`
	Mitigation      []string           `json:"mitigation,omitempty"`
	GeneratedAt     time.Time          `json:"generatedAt"`
}

// UpgradeValidation represents upgrade validation result
type UpgradeValidation struct {
	Valid               bool                      `json:"valid"`
	UpgradeScore        float64                   `json:"upgradeScore"`
	CompatibilityResult *CompatibilityResult      `json:"compatibilityResult"`
	BreakingChangeReport *BreakingChangeReport    `json:"breakingChangeReport,omitempty"`
	RiskAssessment      *RiskAssessment           `json:"riskAssessment,omitempty"`
	Issues              []*UpgradeValidationIssue `json:"issues,omitempty"`
	Recommendations     []string                  `json:"recommendations,omitempty"`
	ValidatedAt         time.Time                 `json:"validatedAt"`
}

// UpgradeValidationIssue represents an upgrade validation issue
type UpgradeValidationIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Component   string `json:"component,omitempty"`
	Resolution  string `json:"resolution,omitempty"`
}

// ArchitecturalConstraints represents architectural constraints
type ArchitecturalConstraints struct {
	PatternConstraints    []string          `json:"patternConstraints,omitempty"`
	LayerConstraints      []string          `json:"layerConstraints,omitempty"`
	DependencyRules       []string          `json:"dependencyRules,omitempty"`
	CouplingLimits        map[string]float64 `json:"couplingLimits,omitempty"`
	CohesionRequirements  map[string]float64 `json:"cohesionRequirements,omitempty"`
	ModularityRules       []string          `json:"modularityRules,omitempty"`
	InterfaceConstraints  []string          `json:"interfaceConstraints,omitempty"`
	ComponentConstraints  []string          `json:"componentConstraints,omitempty"`
}

// Final missing types to complete the dependencies package

// ArchitecturalValidation represents architectural validation result
type ArchitecturalValidation struct {
	Valid           bool                        `json:"valid"`
	Score           float64                     `json:"score"`
	Constraints     *ArchitecturalConstraints   `json:"constraints"`
	Issues          []*ArchitecturalValidationIssue `json:"issues,omitempty"`
	Recommendations []string                    `json:"recommendations,omitempty"`
	ValidatedAt     time.Time                   `json:"validatedAt"`
}

// ArchitecturalValidationIssue represents an architectural validation issue
type ArchitecturalValidationIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Component   string `json:"component,omitempty"`
	Rule        string `json:"rule,omitempty"`
	Resolution  string `json:"resolution,omitempty"`
}

// DependencyConflictReport represents a dependency conflict report
type DependencyConflictReport struct {
	ReportID    string                 `json:"reportId"`
	Conflicts   []*DependencyConflict  `json:"conflicts"`
	Severity    ConflictSeverity       `json:"severity"`
	Impact      *ConflictImpactAnalysis `json:"impact"`
	Resolutions []*ConflictResolutionSuggestion `json:"resolutions,omitempty"`
	GeneratedAt time.Time              `json:"generatedAt"`
}

// ValidatorHealth represents health status of the validator
type ValidatorHealth struct {
	Status              string            `json:"status"`
	Components          map[string]string `json:"components"`
	LastCheck           time.Time         `json:"lastCheck"`
	Issues              []string          `json:"issues,omitempty"`
	UpTime              time.Duration     `json:"upTime"`
	ActiveValidations   int               `json:"activeValidations"`
	QueuedValidations   int               `json:"queuedValidations"`
	LastValidation      time.Time         `json:"lastValidation,omitempty"`
}

// ValidatorMetrics represents metrics for the validator
type ValidatorMetrics struct {
	TotalValidations      int64         `json:"totalValidations"`
	SuccessfulValidations int64         `json:"successfulValidations"`
	FailedValidations     int64         `json:"failedValidations"`
	AverageValidationTime time.Duration `json:"averageValidationTime"`
	ValidationsPerHour    float64       `json:"validationsPerHour"`
	ErrorRate             float64       `json:"errorRate"`
	CacheHitRate          float64       `json:"cacheHitRate"`
	ThroughputPPS         float64       `json:"throughputPPS"` // Packages per second
	LastUpdated           time.Time     `json:"lastUpdated"`
}

// ValidationRules represents validation rules
type ValidationRules struct {
	Rules         []*ValidationRule `json:"rules"`
	RuleGroups    []*ValidationRuleGroup `json:"ruleGroups,omitempty"`
	DefaultRules  []*ValidationRule `json:"defaultRules,omitempty"`
	CustomRules   []*ValidationRule `json:"customRules,omitempty"`
	UpdatedAt     time.Time         `json:"updatedAt"`
}

// ValidationRule represents a validation rule
type ValidationRule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	Severity    string   `json:"severity"`
	Pattern     string   `json:"pattern,omitempty"`
	Script      string   `json:"script,omitempty"`
	Enabled     bool     `json:"enabled"`
	Tags        []string `json:"tags,omitempty"`
	Priority    int      `json:"priority"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// ValidationRuleGroup represents a group of validation rules
type ValidationRuleGroup struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Rules       []*ValidationRule  `json:"rules"`
	Enabled     bool               `json:"enabled"`
	Priority    int                `json:"priority"`
	CreatedAt   time.Time          `json:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt"`
}

// RolloutProgress represents rollout progress
type RolloutProgress struct {
	TotalBatches     int     `json:"totalBatches"`
	CompletedBatches int     `json:"completedBatches"`
	FailedBatches    int     `json:"failedBatches"`
	ProgressPercent  float64 `json:"progressPercent"`
}

// BatchExecution represents the execution of a batch
type BatchExecution struct {
	ID          string        `json:"id"`
	Status      BatchStatus   `json:"status"`
	Packages    []string      `json:"packages"`
	StartedAt   time.Time     `json:"startedAt"`
	CompletedAt *time.Time    `json:"completedAt,omitempty"`
	Duration    time.Duration `json:"duration,omitempty"`
	Errors      []string      `json:"errors,omitempty"`
}

// BatchStatus defines batch statuses
type BatchStatus string

const (
	BatchStatusPending   BatchStatus = "pending"
	BatchStatusRunning   BatchStatus = "running"
	BatchStatusCompleted BatchStatus = "completed"
	BatchStatusFailed    BatchStatus = "failed"
)

// SecurityImpact represents security impact of an update
type SecurityImpact struct {
	VulnerabilitiesFixed  int               `json:"vulnerabilitiesFixed"`
	VulnerabilitiesAdded  int               `json:"vulnerabilitiesAdded"`
	SecurityScoreChange   float64           `json:"securityScoreChange"`
	CriticalIssuesFixed   int               `json:"criticalIssuesFixed"`
	SecurityRecommendations []string        `json:"securityRecommendations,omitempty"`
	ComplianceImpact      *ComplianceImpact `json:"complianceImpact,omitempty"`
}

// ComplianceImpact represents compliance impact of an update  
type ComplianceImpact struct {
	ComplianceScoreChange float64  `json:"complianceScoreChange"`
	StandardsAffected     []string `json:"standardsAffected,omitempty"`
	ViolationsFixed       int      `json:"violationsFixed"`
	ViolationsAdded       int      `json:"violationsAdded"`
}

// PerformanceImpact represents performance impact of an update
type PerformanceImpact struct {
	BenchmarkResults    []*BenchmarkComparison `json:"benchmarkResults,omitempty"`
	MemoryUsageChange   float64                `json:"memoryUsageChange"`
	CPUUsageChange      float64                `json:"cpuUsageChange"`
	StartupTimeChange   time.Duration          `json:"startupTimeChange"`
	ThroughputChange    float64                `json:"throughputChange"`
	LatencyChange       time.Duration          `json:"latencyChange"`
	OverallImpactScore  float64                `json:"overallImpactScore"`
}

// BenchmarkComparison represents comparison of benchmark results
type BenchmarkComparison struct {
	BenchmarkName   string        `json:"benchmarkName"`
	OldResult       float64       `json:"oldResult"`
	NewResult       float64       `json:"newResult"`
	ChangePercent   float64       `json:"changePercent"`
	Improvement     bool          `json:"improvement"`
	Significance    string        `json:"significance"`
}

// Validate method for AnalyzerConfig
func (c *AnalyzerConfig) Validate() error {
	if c.Version == "" {
		c.Version = "1.0.0"
	}
	if c.Currency == "" {
		c.Currency = "USD"
	}
	if c.WorkerCount <= 0 {
		c.WorkerCount = 4
	}
	if c.QueueSize <= 0 {
		c.QueueSize = 100
	}
	return nil
}