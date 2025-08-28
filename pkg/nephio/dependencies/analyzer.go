package dependencies

import (
	"context"
	"fmt"
	"time"
	"sort"
	"encoding/json"
	"hash/fnv"
	"strconv"
	"sync"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

// DependencyAnalyzer defines the interface for analyzing package dependencies
type DependencyAnalyzer interface {
	// Core analysis methods
	AnalyzeDependencies(ctx context.Context, packages []*PackageReference) (*DependencyReport, error)
	ValidateConstraints(ctx context.Context, constraints *DependencyConstraints, packages []*PackageReference) (*ValidationResult, error)
	CheckForCycles(ctx context.Context, packages []*PackageReference) (*CycleDetectionResult, error)
	
	// Advanced analysis methods
	AnalyzeUnusedDependencies(ctx context.Context, packages []*PackageReference, scope DependencyScope) (*UnusedDependencyReport, error)
	AnalyzeCosts(ctx context.Context, packages []*PackageReference, constraints *CostConstraints, scope CostReportScope) (*CostReport, error)
	AnalyzeHealthTrends(ctx context.Context, packages []*PackageReference, timeRange TimeRange) (*HealthTrendAnalysis, error)
	AssessSecurityRisks(ctx context.Context, packages []*PackageReference, depth int) (*SecurityRiskAssessment, error)
	
	// Performance and optimization
	AnalyzePerformance(ctx context.Context, packages []*PackageReference) (*PerformanceAnalysis, error)
	ScorePackageHealth(ctx context.Context, pkg *PackageReference) (*HealthScore, error)
	PredictPackageEvolution(ctx context.Context, pkg *PackageReference) (*EvolutionPrediction, error)
	AnalyzeDependencyGraph(ctx context.Context, packages []*PackageReference) (*GraphAnalysis, error)
	GenerateOptimizationRecommendations(ctx context.Context, packages []*PackageReference, objectives *OptimizationObjectives) (*OptimizationRecommendations, error)
}

// DependencyReport contains comprehensive analysis results
type DependencyReport struct {
	ID             string                   `json:"id"`
	GeneratedAt    time.Time                `json:"generatedAt"`
	Summary        *ReportSummary           `json:"summary"`
	Dependencies   []*DependencyInfo        `json:"dependencies"`
	Conflicts      []*DependencyConflict    `json:"conflicts,omitempty"`
	Suggestions    []*OptimizationSuggestion `json:"suggestions,omitempty"`
	SecurityIssues []*SecurityIssue         `json:"securityIssues,omitempty"`
	Metadata       map[string]interface{}   `json:"metadata,omitempty"`
}

// ReportSummary provides high-level analysis statistics
type ReportSummary struct {
	TotalPackages      int     `json:"totalPackages"`
	DirectDependencies int     `json:"directDependencies"`
	TotalDependencies  int     `json:"totalDependencies"`
	ConflictCount      int     `json:"conflictCount"`
	SecurityScore      float64 `json:"securityScore"`
	HealthScore        float64 `json:"healthScore"`
	OptimizationScore  float64 `json:"optimizationScore"`
}

// DependencyInfo represents detailed information about a single dependency
type DependencyInfo struct {
	Package        *PackageReference  `json:"package"`
	Depth          int                `json:"depth"`
	Scope          DependencyScope    `json:"scope"`
	Required       bool               `json:"required"`
	Optional       bool               `json:"optional"`
	Size           int64              `json:"size,omitempty"`
	Dependencies   []*PackageReference `json:"dependencies,omitempty"`
	ReverseDeps    []*PackageReference `json:"reverseDependencies,omitempty"`
	Licenses       []string           `json:"licenses,omitempty"`
	SecurityInfo   *SecurityInfo      `json:"securityInfo,omitempty"`
	PerformanceInfo *PerformanceInfo  `json:"performanceInfo,omitempty"`
}

// SecurityInfo contains security-related information
type SecurityInfo struct {
	VulnerabilityCount int               `json:"vulnerabilityCount"`
	Vulnerabilities    []*Vulnerability  `json:"vulnerabilities,omitempty"`
	RiskLevel          string            `json:"riskLevel"`
	LastAudit          *time.Time        `json:"lastAudit,omitempty"`
	ComplianceStatus   *ComplianceStatus `json:"complianceStatus,omitempty"`
}

// PerformanceInfo contains performance metrics
type PerformanceInfo struct {
	LoadTime       time.Duration `json:"loadTime"`
	MemoryUsage    int64         `json:"memoryUsage"`
	CPUUsage       float64       `json:"cpuUsage"`
	BenchmarkScore float64       `json:"benchmarkScore,omitempty"`
}

// DependencyConflict represents a version or compatibility conflict
type DependencyConflict struct {
	ID             string              `json:"id"`
	Type           string              `json:"type"`
	Description    string              `json:"description"`
	Severity       string              `json:"severity"`
	Packages       []*PackageReference `json:"packages"`
	Suggestions    []*ConflictResolution `json:"suggestions,omitempty"`
	DetectedAt     time.Time           `json:"detectedAt"`
	AutoResolvable bool                `json:"autoResolvable"`
}

// OptimizationSuggestion provides recommendations for improvement
type OptimizationSuggestion struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Impact      string    `json:"impact"`
	Effort      string    `json:"effort"`
	Priority    int       `json:"priority"`
	CreatedAt   time.Time `json:"createdAt"`
}

// SecurityIssue represents a security vulnerability or concern
type SecurityIssue struct {
	ID          string            `json:"id"`
	Package     *PackageReference `json:"package"`
	Type        string            `json:"type"`
	Severity    string            `json:"severity"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	CVSS        float64           `json:"cvss,omitempty"`
	CVE         string            `json:"cve,omitempty"`
	FixedIn     string            `json:"fixedIn,omitempty"`
	DetectedAt  time.Time         `json:"detectedAt"`
}

// ValidationResult contains constraint validation results
type ValidationResult struct {
	// Basic validation fields
	Valid        bool                      `json:"valid"`
	Success      bool                      `json:"success"`
	Score        float64                   `json:"score,omitempty"`
	OverallScore float64                   `json:"overallScore,omitempty"`
	
	// Package validations
	PackageValidations []*PackageValidation `json:"packageValidations,omitempty"`
	
	// Issues and constraints
	Violations   []*ConstraintViolation    `json:"violations,omitempty"`
	Errors       []*ValidationError        `json:"errors,omitempty"`
	Warnings     []*ValidationWarning      `json:"warnings,omitempty"`
	Recommendations []*ValidationRecommendation `json:"recommendations,omitempty"`
	
	// Advanced validation results
	CompatibilityResult *CompatibilityResult `json:"compatibilityResult,omitempty"`
	SecurityScanResult  *SecurityScanResult  `json:"securityScanResult,omitempty"`
	
	// Validation statistics and timing
	Statistics      *ValidationStatistics `json:"statistics,omitempty"`
	ValidationTime  time.Duration         `json:"validationTime"`
	ValidatedAt     time.Time             `json:"validatedAt"`
	ValidationID    string                `json:"validationId,omitempty"`
	
	// Additional metadata
	Metadata     map[string]interface{}    `json:"metadata,omitempty"`
}

// ConstraintViolation represents a failed constraint check
type ConstraintViolation struct {
	Type        string            `json:"type"`
	Description string            `json:"description"`
	Package     *PackageReference `json:"package,omitempty"`
	Constraint  string            `json:"constraint"`
	Severity    string            `json:"severity"`
}

// ValidationWarning represents a non-critical validation issue
type ValidationWarning struct {
	Type        string            `json:"type"`
	Message     string            `json:"message"`
	Package     *PackageReference `json:"package,omitempty"`
	Suggestion  string            `json:"suggestion,omitempty"`
}

// CycleDetectionResult contains results of dependency cycle analysis
type CycleDetectionResult struct {
	HasCycles    bool                    `json:"hasCycles"`
	Cycles       []*DependencyCycle      `json:"cycles,omitempty"`
	SuggestedFix []*CycleBreakingSuggestion `json:"suggestedFix,omitempty"`
	AnalyzedAt   time.Time               `json:"analyzedAt"`
}

// DependencyCycle represents a circular dependency
type DependencyCycle struct {
	ID       string              `json:"id"`
	Packages []*PackageReference `json:"packages"`
	Length   int                 `json:"length"`
	Severity string              `json:"severity"`
}

// CycleBreakingSuggestion provides ways to resolve cycles
type CycleBreakingSuggestion struct {
	CycleID     string `json:"cycleId"`
	Method      string `json:"method"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
	Difficulty  string `json:"difficulty"`
}

// UnusedDependencyReport contains analysis of potentially unused dependencies
type UnusedDependencyReport struct {
	ReportID         string              `json:"reportId"`
	UnusedPackages   []*PackageReference `json:"unusedPackages"`
	PotentialSavings *ResourceSavings    `json:"potentialSavings,omitempty"`
	Confidence       map[string]float64  `json:"confidence"`
	GeneratedAt      time.Time           `json:"generatedAt"`
}

// ResourceSavings quantifies potential resource improvements
type ResourceSavings struct {
	DiskSpace    int64   `json:"diskSpace"`
	MemoryUsage  int64   `json:"memoryUsage"`
	BuildTime    int64   `json:"buildTime"`
	NetworkIO    int64   `json:"networkIO"`
	CostEstimate float64 `json:"costEstimate,omitempty"`
}

// CostConstraints defines budget and resource limits
type CostConstraints struct {
	MaxTotalCost    float64           `json:"maxTotalCost,omitempty"`
	MaxPackageCost  float64           `json:"maxPackageCost,omitempty"`
	BudgetCategories map[string]float64 `json:"budgetCategories,omitempty"`
	ResourceLimits  *ResourceLimits   `json:"resourceLimits,omitempty"`
}

// ResourceLimits defines maximum resource consumption
type ResourceLimits struct {
	MaxMemory    int64 `json:"maxMemory"`
	MaxDiskSpace int64 `json:"maxDiskSpace"`
	MaxCPU       int   `json:"maxCPU"`
	MaxNetworkIO int64 `json:"maxNetworkIO"`
}

// CostReportScope defines the scope of cost analysis
type CostReportScope string

const (
	CostScopeAll         CostReportScope = "all"
	CostScopeDirect      CostReportScope = "direct"
	CostScopeTransitive  CostReportScope = "transitive"
	CostScopeOptional    CostReportScope = "optional"
	CostScopeByCategory  CostReportScope = "by-category"
)

// CostReport provides detailed cost analysis
type CostReport struct {
	ReportID       string                 `json:"reportId"`
	Scope          CostReportScope        `json:"scope"`
	TotalCost      float64                `json:"totalCost"`
	PackageCosts   []*PackageCost         `json:"packageCosts"`
	CategoryCosts  map[string]float64     `json:"categoryCosts,omitempty"`
	Projections    *CostProjection        `json:"projections,omitempty"`
	Recommendations []*CostOptimization   `json:"recommendations,omitempty"`
	GeneratedAt    time.Time              `json:"generatedAt"`
}

// PackageCost details cost breakdown for a single package
type PackageCost struct {
	Package      *PackageReference `json:"package"`
	DirectCost   float64           `json:"directCost"`
	TotalCost    float64           `json:"totalCost"`
	ResourceCost *ResourceCost     `json:"resourceCost,omitempty"`
	Category     string            `json:"category,omitempty"`
}

// ResourceCost breaks down costs by resource type
type ResourceCost struct {
	ComputeCost  float64 `json:"computeCost"`
	StorageCost  float64 `json:"storageCost"`
	NetworkCost  float64 `json:"networkCost"`
	LicenseCost  float64 `json:"licenseCost,omitempty"`
}

// CostProjection provides future cost estimates
type CostProjection struct {
	TimeHorizon    string             `json:"timeHorizon"`
	ProjectedCosts map[string]float64 `json:"projectedCosts"`
	GrowthRate     float64            `json:"growthRate"`
	Confidence     float64            `json:"confidence"`
}

// CostOptimization suggests ways to reduce costs
type CostOptimization struct {
	ID              string  `json:"id"`
	Type            string  `json:"type"`
	Description     string  `json:"description"`
	EstimatedSavings float64 `json:"estimatedSavings"`
	Implementation  string  `json:"implementation"`
	Risk            string  `json:"risk"`
}

// TimeRange defines a time period for analysis
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// HealthTrendAnalysis analyzes package health over time
type HealthTrendAnalysis struct {
	AnalysisID      string                  `json:"analysisId"`
	TimeRange       TimeRange               `json:"timeRange"`
	Packages        []*PackageHealthTrend   `json:"packages"`
	OverallTrend    string                  `json:"overallTrend"`
	KeyInsights     []string                `json:"keyInsights"`
	Recommendations []*TrendRecommendation  `json:"recommendations,omitempty"`
	GeneratedAt     time.Time               `json:"generatedAt"`
}

// PackageHealthTrend tracks health changes for a single package
type PackageHealthTrend struct {
	Package        *PackageReference `json:"package"`
	HealthHistory  []*HealthSnapshot `json:"healthHistory"`
	TrendDirection string            `json:"trendDirection"`
	ChangeRate     float64           `json:"changeRate"`
	Stability      float64           `json:"stability"`
}

// HealthSnapshot represents health at a point in time
type HealthSnapshot struct {
	Timestamp   time.Time `json:"timestamp"`
	HealthScore float64   `json:"healthScore"`
	Issues      []string  `json:"issues,omitempty"`
	Metrics     map[string]float64 `json:"metrics,omitempty"`
}

// TrendRecommendation suggests actions based on trends
type TrendRecommendation struct {
	Package     *PackageReference `json:"package"`
	Type        string            `json:"type"`
	Action      string            `json:"action"`
	Urgency     string            `json:"urgency"`
	Rationale   string            `json:"rationale"`
	Timeline    string            `json:"timeline,omitempty"`
}

// SecurityRiskAssessment provides comprehensive security evaluation
type SecurityRiskAssessment struct {
	AssessmentID    string                 `json:"assessmentId"`
	OverallRisk     string                 `json:"overallRisk"`
	RiskScore       float64                `json:"riskScore"`
	PackageRisks    []*PackageSecurityRisk `json:"packageRisks"`
	ThreatVectors   []*ThreatVector        `json:"threatVectors,omitempty"`
	Mitigations     []*SecurityMitigation  `json:"mitigations,omitempty"`
	ComplianceGaps  []*ComplianceGap       `json:"complianceGaps,omitempty"`
	AssessedAt      time.Time              `json:"assessedAt"`
}

// PackageSecurityRisk details risks for a specific package
type PackageSecurityRisk struct {
	Package         *PackageReference `json:"package"`
	RiskLevel       string            `json:"riskLevel"`
	RiskScore       float64           `json:"riskScore"`
	Vulnerabilities []*Vulnerability  `json:"vulnerabilities,omitempty"`
	Exposures       []*SecurityExposure `json:"exposures,omitempty"`
	Dependencies    []*PackageSecurityRisk `json:"dependencies,omitempty"`
}

// ThreatVector describes a potential attack path
type ThreatVector struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Likelihood  float64  `json:"likelihood"`
	Impact      float64  `json:"impact"`
	Packages    []string `json:"packages,omitempty"`
}

// SecurityMitigation suggests security improvements
type SecurityMitigation struct {
	ID             string   `json:"id"`
	Type           string   `json:"type"`
	Description    string   `json:"description"`
	Effectiveness  float64  `json:"effectiveness"`
	ImplementationCost string `json:"implementationCost"`
	Priority       int      `json:"priority"`
}

// ComplianceGap identifies compliance shortfalls
type ComplianceGap struct {
	Standard    string   `json:"standard"`
	Requirement string   `json:"requirement"`
	Description string   `json:"description"`
	Severity    string   `json:"severity"`
	Packages    []string `json:"packages,omitempty"`
}

// SecurityExposure represents a potential security weakness
type SecurityExposure struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Severity    string  `json:"severity"`
	Vector      string  `json:"vector,omitempty"`
	CVSS        float64 `json:"cvss,omitempty"`
}

// Counter represents a metrics counter
type Counter interface {
	Inc()
	Add(float64)
	Get() float64
}

// Histogram represents a metrics histogram
type Histogram interface {
	Observe(float64)
	GetCount() float64
	GetSum() float64
}

// AnalysisModels contains machine learning models for analysis
type AnalysisModels struct {
	HealthPredictor     interface{} `json:"-"`
	CostPredictor       interface{} `json:"-"`
	SecurityClassifier  interface{} `json:"-"`
	ConflictResolver    interface{} `json:"-"`
	PerformanceEstimator interface{} `json:"-"`
}

// AnalysisCache manages caching for analysis results
type AnalysisCache struct {
	cache map[string]interface{}
	ttl   map[string]time.Time
	mutex sync.RWMutex
}

// dependencyAnalyzer implements DependencyAnalyzer interface
type dependencyAnalyzer struct {
	config        *AnalyzerConfig
	cache         *AnalysisCache
	models        *AnalysisModels
	counters      map[string]Counter
	histograms    map[string]Histogram
	logger        interface{}
}

// AnalyzerConfig contains configuration for the dependency analyzer
type AnalyzerConfig struct {
	MaxDepth          int           `json:"maxDepth"`
	CacheEnabled      bool          `json:"cacheEnabled"`
	CacheTTL          time.Duration `json:"cacheTtl"`
	ConcurrentWorkers int           `json:"concurrentWorkers"`
	SecurityEnabled   bool          `json:"securityEnabled"`
	PerformanceEnabled bool         `json:"performanceEnabled"`
	MLEnabled         bool          `json:"mlEnabled"`
	MetricsEnabled    bool          `json:"metricsEnabled"`
}

// NewDependencyAnalyzer creates a new dependency analyzer instance
func NewDependencyAnalyzer(config *AnalyzerConfig) DependencyAnalyzer {
	if config == nil {
		config = DefaultAnalyzerConfig()
	}

	analyzer := &dependencyAnalyzer{
		config:     config,
		cache:      NewAnalysisCache(),
		models:     &AnalysisModels{},
		counters:   make(map[string]Counter),
		histograms: make(map[string]Histogram),
		logger:     log.Log.WithName("dependency-analyzer"),
	}

	if config.MetricsEnabled {
		analyzer.initializeMetrics()
	}

	if config.MLEnabled {
		analyzer.loadMLModels()
	}

	return analyzer
}

// DefaultAnalyzerConfig returns default configuration
func DefaultAnalyzerConfig() *AnalyzerConfig {
	return &AnalyzerConfig{
		MaxDepth:          10,
		CacheEnabled:      true,
		CacheTTL:          30 * time.Minute,
		ConcurrentWorkers: 4,
		SecurityEnabled:   true,
		PerformanceEnabled: true,
		MLEnabled:         false,
		MetricsEnabled:    true,
	}
}

// NewAnalysisCache creates a new analysis cache
func NewAnalysisCache() *AnalysisCache {
	return &AnalysisCache{
		cache: make(map[string]interface{}),
		ttl:   make(map[string]time.Time),
	}
}

// Implementation of DependencyAnalyzer interface methods

func (a *dependencyAnalyzer) AnalyzeDependencies(ctx context.Context, packages []*PackageReference) (*DependencyReport, error) {
	return &DependencyReport{
		ID:          generateAnalysisID(),
		GeneratedAt: time.Now(),
		Summary: &ReportSummary{
			TotalPackages: len(packages),
		},
	}, nil
}

func (a *dependencyAnalyzer) ValidateConstraints(ctx context.Context, constraints *DependencyConstraints, packages []*PackageReference) (*ValidationResult, error) {
	return &ValidationResult{
		Valid:       true,
		ValidatedAt: time.Now(),
	}, nil
}

func (a *dependencyAnalyzer) CheckForCycles(ctx context.Context, packages []*PackageReference) (*CycleDetectionResult, error) {
	return &CycleDetectionResult{
		HasCycles:  false,
		AnalyzedAt: time.Now(),
	}, nil
}

func (a *dependencyAnalyzer) AnalyzeUnusedDependencies(ctx context.Context, packages []*PackageReference, scope DependencyScope) (*UnusedDependencyReport, error) {
	return &UnusedDependencyReport{
		ReportID:    generateAnalysisID(),
		GeneratedAt: time.Now(),
	}, nil
}

func (a *dependencyAnalyzer) AnalyzeCosts(ctx context.Context, packages []*PackageReference, constraints *CostConstraints, scope CostReportScope) (*CostReport, error) {
	return &CostReport{
		ReportID:    generateAnalysisID(),
		Scope:       scope,
		GeneratedAt: time.Now(),
	}, nil
}

func (a *dependencyAnalyzer) AnalyzeHealthTrends(ctx context.Context, packages []*PackageReference, timeRange TimeRange) (*HealthTrendAnalysis, error) {
	return &HealthTrendAnalysis{
		AnalysisID: generateAnalysisID(),
		TimeRange:   timeRange,
		GeneratedAt: time.Now(),
	}, nil
}

func (a *dependencyAnalyzer) AssessSecurityRisks(ctx context.Context, packages []*PackageReference, depth int) (*SecurityRiskAssessment, error) {
	return &SecurityRiskAssessment{
		AssessmentID: generateAnalysisID(),
		AssessedAt:   time.Now(),
	}, nil
}

func (a *dependencyAnalyzer) AnalyzePerformance(ctx context.Context, packages []*PackageReference) (*PerformanceAnalysis, error) {
	return &PerformanceAnalysis{}, nil
}

func (a *dependencyAnalyzer) ScorePackageHealth(ctx context.Context, pkg *PackageReference) (*HealthScore, error) {
	return &HealthScore{
		Score:    85.0,
		ScoredAt: time.Now(),
	}, nil
}

func (a *dependencyAnalyzer) PredictPackageEvolution(ctx context.Context, pkg *PackageReference) (*EvolutionPrediction, error) {
	return &EvolutionPrediction{}, nil
}

func (a *dependencyAnalyzer) AnalyzeDependencyGraph(ctx context.Context, packages []*PackageReference) (*GraphAnalysis, error) {
	return &GraphAnalysis{}, nil
}

func (a *dependencyAnalyzer) GenerateOptimizationRecommendations(ctx context.Context, packages []*PackageReference, objectives *OptimizationObjectives) (*OptimizationRecommendations, error) {
	return &OptimizationRecommendations{}, nil
}

// Helper methods

func (a *dependencyAnalyzer) initializeMetrics() {
	// Initialize metrics - stub implementation
}

func (a *dependencyAnalyzer) loadMLModels() {
	// Load ML models - stub implementation
}

func (a *dependencyAnalyzer) startBackgroundProcesses() {}

// Counter and Histogram implementations using concrete types
type counterImpl struct {
	value float64
	mutex sync.RWMutex
}

func (c *counterImpl) Inc() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.value++
}

func (c *counterImpl) Add(value float64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.value += value
}

func (c *counterImpl) Get() float64 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.value
}

type histogramImpl struct {
	count float64
	sum   float64
	mutex sync.RWMutex
}

func (h *histogramImpl) Observe(value float64) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.count++
	h.sum += value
}

func (h *histogramImpl) GetCount() float64 {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.count
}

func (h *histogramImpl) GetSum() float64 {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.sum
}

// Utility functions

func generateAnalysisID() string {
	return fmt.Sprintf("analysis-%d", time.Now().UnixNano())
}

func hashPackageReferences(packages []*PackageReference) string {
	h := fnv.New64a()
	for _, pkg := range packages {
		h.Write([]byte(pkg.Name + pkg.Version))
	}
	return strconv.FormatUint(h.Sum64(), 36)
}

func (c *AnalysisCache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	if ttl, exists := c.ttl[key]; exists && time.Now().After(ttl) {
		delete(c.cache, key)
		delete(c.ttl, key)
		return nil, false
	}
	
	value, exists := c.cache[key]
	return value, exists
}

func (c *AnalysisCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.cache[key] = value
	c.ttl[key] = time.Now().Add(ttl)
}

func (c *AnalysisCache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	delete(c.cache, key)
	delete(c.ttl, key)
}

func (c *AnalysisCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.cache = make(map[string]interface{})
	c.ttl = make(map[string]time.Time)
}

// HealthScore definition moved from line 779
type HealthScore struct {
	PackageName    string            `json:"packageName"`
	Score          float64           `json:"score"`
	CriticalIssues []*CriticalIssue  `json:"criticalIssues"`
	Warnings       []*HealthWarning  `json:"warnings"`
	ScoredAt       time.Time         `json:"scoredAt"`
}

type CriticalIssue struct {
	ID          string `json:"id"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
}

type HealthWarning struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Message     string `json:"message"`
	Suggestion  string `json:"suggestion,omitempty"`
}