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
	"fmt"
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
type AnalysisCacheConfig struct{}
type DataStoreConfig struct{}

// Validate validates the analyzer configuration
func (c *AnalysisConfig) Validate() error {
	// Add validation logic here
	return nil
}

// DefaultAnalyzerConfig returns a default configuration
func DefaultAnalyzerConfig() *AnalysisConfig {
	return &AnalysisConfig{
		Version:              "1.0.0",
		Currency:            "USD",
		EnableMLAnalysis:    false,
		EnableMLOptimization: false,
		EnableCaching:       true,
		EnableConcurrency:   true,
		EnableTrendAnalysis: true,
		EnableCostProjection: true,
		WorkerCount:         4,
		QueueSize:          100,
	}
}

// Metrics and analysis component types (simplified definitions)
type AnalyzerMetrics struct{}
type UsageAnalyzer struct{}
type CostAnalyzer struct{}
type HealthAnalyzer struct{}
type RiskAnalyzer struct{}
type PerformanceAnalyzer struct{}
type OptimizationEngine struct{}
type MLOptimizer struct{}
type UsageDataCollector struct{}
type MetricsCollector struct{}
type EventProcessor struct{}
type PredictionModel struct{}
type RecommendationModel struct{}
type AnomalyDetector struct{}
type AnalysisCache struct{}
type AnalysisDataStore struct{}
type CostProvider interface{}
type AnalysisWorkerPool struct{}

// Constructor functions (placeholder implementations)
func NewAnalyzerMetrics() *AnalyzerMetrics { return &AnalyzerMetrics{} }
func NewUsageAnalyzer(config *UsageAnalyzerConfig) (*UsageAnalyzer, error) { return &UsageAnalyzer{}, nil }
func NewCostAnalyzer(config *CostAnalyzerConfig) (*CostAnalyzer, error) { return &CostAnalyzer{}, nil }
func NewHealthAnalyzer(config *HealthAnalyzerConfig) (*HealthAnalyzer, error) { return &HealthAnalyzer{}, nil }
func NewRiskAnalyzer(config *RiskAnalyzerConfig) (*RiskAnalyzer, error) { return &RiskAnalyzer{}, nil }
func NewPerformanceAnalyzer(config *PerformanceAnalyzerConfig) (*PerformanceAnalyzer, error) { return &PerformanceAnalyzer{}, nil }
func NewOptimizationEngine(config *OptimizationEngineConfig) (*OptimizationEngine, error) { return &OptimizationEngine{}, nil }
func NewMLOptimizer(config *MLOptimizerConfig) (*MLOptimizer, error) { return &MLOptimizer{}, nil }
func NewUsageDataCollector(config *UsageCollectorConfig) *UsageDataCollector { return &UsageDataCollector{} }
func NewMetricsCollector(config *MetricsCollectorConfig) *MetricsCollector { return &MetricsCollector{} }
func NewEventProcessor(config *EventProcessorConfig) *EventProcessor { return &EventProcessor{} }
func NewPredictionModel(config *PredictionModelConfig) (*PredictionModel, error) { return &PredictionModel{}, nil }
func NewRecommendationModel(config *RecommendationModelConfig) (*RecommendationModel, error) { return &RecommendationModel{}, nil }
func NewAnomalyDetector(config *AnomalyDetectorConfig) (*AnomalyDetector, error) { return &AnomalyDetector{}, nil }
func NewAnalysisCache(config *AnalysisCacheConfig) *AnalysisCache { return &AnalysisCache{} }
func NewAnalysisDataStore(config *DataStoreConfig) (*AnalysisDataStore, error) { return &AnalysisDataStore{}, nil }
func NewAnalysisWorkerPool(workerCount, queueSize int) *AnalysisWorkerPool { return &AnalysisWorkerPool{} }

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

// HealthSnapshot represents a health snapshot at a point in time
type HealthSnapshot struct {
	Timestamp    time.Time `json:"timestamp"`
	HealthScore  float64   `json:"healthScore"`
	Grade        string    `json:"grade"`
	IssuesCount  int       `json:"issuesCount"`
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