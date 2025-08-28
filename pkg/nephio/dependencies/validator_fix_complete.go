package dependencies

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Complete validator fix - all missing types and functions for validator.go

// Missing validation result types needed by validator.go interface
type CompatibilityResult struct {
	Packages            []*PackageReference    `json:"packages"`
	Compatible          bool                   `json:"compatible"`
	CompatibilityMatrix map[string]map[string]bool `json:"compatibilityMatrix"`
	IncompatiblePairs   []*IncompatiblePair    `json:"incompatiblePairs,omitempty"`
	ValidationTime      time.Duration          `json:"validationTime"`
}

type IncompatiblePair struct {
	Package1 *PackageReference `json:"package1"`
	Package2 *PackageReference `json:"package2"`
	Reason   string            `json:"reason"`
	Severity string            `json:"severity"`
	Impact   string            `json:"impact,omitempty"`
}

type PlatformConstraints struct {
	OS           string   `json:"os"`
	Architecture string   `json:"architecture"`
	MinVersion   string   `json:"minVersion,omitempty"`
	MaxVersion   string   `json:"maxVersion,omitempty"`
	Requirements []string `json:"requirements,omitempty"`
}

type PlatformValidation struct {
	Valid           bool                       `json:"valid"`
	Platform        string                     `json:"platform"`
	SupportedVersions []string                 `json:"supportedVersions"`
	PlatformIssues  []*PlatformIssue           `json:"platformIssues,omitempty"`
	Requirements    []*PlatformRequirement     `json:"requirements"`
	ValidatedAt     time.Time                  `json:"validatedAt"`
}

type PlatformIssue struct {
	IssueID     string    `json:"issueId"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Component   string    `json:"component,omitempty"`
	Resolution  string    `json:"resolution,omitempty"`
	DetectedAt  time.Time `json:"detectedAt"`
}

type PlatformRequirement struct {
	RequirementID string                 `json:"requirementId"`
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	Version       string                 `json:"version,omitempty"`
	Optional      bool                   `json:"optional"`
	Met           bool                   `json:"met"`
	Details       map[string]interface{} `json:"details,omitempty"`
}

type SecurityValidation struct {
	Valid           bool                     `json:"valid"`
	SecurityScore   float64                  `json:"securityScore"`
	RiskLevel       string                   `json:"riskLevel"`
	Vulnerabilities []*SecurityVulnerability `json:"vulnerabilities,omitempty"`
	PolicyViolations []*SecurityPolicyViolation `json:"policyViolations,omitempty"`
	Recommendations []string                 `json:"recommendations,omitempty"`
	ValidatedAt     time.Time                `json:"validatedAt"`
}

type SecurityPolicyViolation struct {
	ViolationID string            `json:"violationId"`
	PolicyID    string            `json:"policyId"`
	PolicyName  string            `json:"policyName"`
	Severity    string            `json:"severity"`
	Description string            `json:"description"`
	Package     *PackageReference `json:"package,omitempty"`
	DetectedAt  time.Time         `json:"detectedAt"`
	Resolution  string            `json:"resolution,omitempty"`
}

type LicenseValidation struct {
	Valid               bool                    `json:"valid"`
	DetectedLicenses    []string                `json:"detectedLicenses"`
	LicenseCompatibility string                 `json:"licenseCompatibility"`
	LicenseIssues       []*LicenseIssue         `json:"licenseIssues,omitempty"`
	LicenseConflicts    []*LicenseConflict      `json:"licenseConflicts,omitempty"`
	ValidatedAt         time.Time               `json:"validatedAt"`
}

type LicenseIssue struct {
	IssueID     string            `json:"issueId"`
	Type        string            `json:"type"`
	Severity    string            `json:"severity"`
	License     string            `json:"license"`
	Description string            `json:"description"`
	Package     *PackageReference `json:"package,omitempty"`
	Resolution  string            `json:"resolution,omitempty"`
}

type LicenseConflict struct {
	ConflictID     string              `json:"conflictId"`
	ConflictingLicenses []string        `json:"conflictingLicenses"`
	ConflictingPackages []*PackageReference `json:"conflictingPackages"`
	Description    string              `json:"description"`
	Severity       string              `json:"severity"`
	Resolution     string              `json:"resolution,omitempty"`
}

type ComplianceValidation struct {
	Valid               bool                      `json:"valid"`
	ComplianceScore     float64                   `json:"complianceScore"`
	ComplianceStatus    ComplianceStatus          `json:"complianceStatus"`
	ComplianceChecks    []*ComplianceCheckResult  `json:"complianceChecks"`
	Violations          []*ComplianceViolation    `json:"violations,omitempty"`
	ValidatedAt         time.Time                 `json:"validatedAt"`
}

type ComplianceCheckResult struct {
	CheckID     string                 `json:"checkId"`
	CheckName   string                 `json:"checkName"`
	Standard    string                 `json:"standard"`
	Status      string                 `json:"status"`
	Score       float64                `json:"score"`
	Evidence    []string               `json:"evidence,omitempty"`
	Issues      []string               `json:"issues,omitempty"`
	CheckedAt   time.Time              `json:"checkedAt"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type ComplianceViolation struct {
	ViolationID string            `json:"violationId"`
	Standard    string            `json:"standard"`
	Control     string            `json:"control"`
	Severity    string            `json:"severity"`
	Description string            `json:"description"`
	Package     *PackageReference `json:"package,omitempty"`
	Evidence    []string          `json:"evidence,omitempty"`
	Resolution  string            `json:"resolution,omitempty"`
	DetectedAt  time.Time         `json:"detectedAt"`
}

type PerformanceValidation struct {
	Valid               bool                          `json:"valid"`
	PerformanceScore    float64                       `json:"performanceScore"`
	PerformanceMetrics  []*PerformanceMetric          `json:"performanceMetrics"`
	PerformanceIssues   []*PerformanceIssue           `json:"performanceIssues,omitempty"`
	Benchmarks          []*PerformanceBenchmark       `json:"benchmarks,omitempty"`
	ValidatedAt         time.Time                     `json:"validatedAt"`
}

type PerformanceMetric struct {
	MetricID    string                 `json:"metricId"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Value       float64                `json:"value"`
	Unit        string                 `json:"unit"`
	Threshold   float64                `json:"threshold,omitempty"`
	Status      string                 `json:"status"`
	MeasuredAt  time.Time              `json:"measuredAt"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type PerformanceIssue struct {
	IssueID     string            `json:"issueId"`
	Type        string            `json:"type"`
	Severity    string            `json:"severity"`
	Description string            `json:"description"`
	Impact      string            `json:"impact"`
	Package     *PackageReference `json:"package,omitempty"`
	Recommendation string         `json:"recommendation,omitempty"`
	DetectedAt  time.Time         `json:"detectedAt"`
}

type PerformanceBenchmark struct {
	BenchmarkID string                 `json:"benchmarkId"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Results     map[string]float64     `json:"results"`
	Baseline    map[string]float64     `json:"baseline,omitempty"`
	Status      string                 `json:"status"`
	RunAt       time.Time              `json:"runAt"`
	Duration    time.Duration          `json:"duration"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type ResourceValidation struct {
	Valid               bool                    `json:"valid"`
	ResourceUsage       *ResourceUsageResult    `json:"resourceUsage"`
	ResourceIssues      []*ResourceIssue        `json:"resourceIssues,omitempty"`
	ResourceLimits      *ResourceLimits         `json:"resourceLimits"`
	Recommendations     []string                `json:"recommendations,omitempty"`
	ValidatedAt         time.Time               `json:"validatedAt"`
}

type ResourceUsageResult struct {
	CPUUsage       float64           `json:"cpuUsage"`
	MemoryUsage    int64             `json:"memoryUsage"`
	DiskUsage      int64             `json:"diskUsage"`
	NetworkUsage   int64             `json:"networkUsage"`
	CustomMetrics  map[string]float64 `json:"customMetrics,omitempty"`
	MeasuredAt     time.Time         `json:"measuredAt"`
	Duration       time.Duration     `json:"duration"`
}

type ResourceIssue struct {
	IssueID     string            `json:"issueId"`
	Type        string            `json:"type"`
	Resource    string            `json:"resource"`
	Severity    string            `json:"severity"`
	Description string            `json:"description"`
	Current     float64           `json:"current"`
	Limit       float64           `json:"limit"`
	Package     *PackageReference `json:"package,omitempty"`
	Resolution  string            `json:"resolution,omitempty"`
	DetectedAt  time.Time         `json:"detectedAt"`
}

type OrganizationalPolicies struct {
	PolicyID        string                 `json:"policyId"`
	Name            string                 `json:"name"`
	Version         string                 `json:"version"`
	Type            string                 `json:"type"`
	Rules           []*PolicyRule          `json:"rules"`
	Exceptions      []*PolicyException     `json:"exceptions,omitempty"`
	ApplicableTeams []string               `json:"applicableTeams,omitempty"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
}


type PolicyException struct {
	ExceptionID string    `json:"exceptionId"`
	RuleID      string    `json:"ruleId"`
	Reason      string    `json:"reason"`
	ApprovedBy  string    `json:"approvedBy"`
	ExpiresAt   time.Time `json:"expiresAt,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

// Configuration types from validator_final.go

type ValidatorConfig struct {
	MaxDepth                    int                      `json:"maxDepth"`
	ConcurrentWorkers          int                      `json:"concurrentWorkers"`
	Timeout                    time.Duration            `json:"timeout"`
	EnableCaching              bool                     `json:"enableCaching"`
	RetryAttempts              int                      `json:"retryAttempts"`
	EnableConcurrency          bool                     `json:"enableConcurrency"`
	WorkerCount                int                      `json:"workerCount"`
	QueueSize                  int                      `json:"queueSize"`
	VulnerabilityUpdateInterval time.Duration           `json:"vulnerabilityUpdateInterval"`
	CacheCleanupInterval       time.Duration            `json:"cacheCleanupInterval"`
	MetricsCollectionInterval  time.Duration            `json:"metricsCollectionInterval"`
	DefaultValidationRules     *ValidationRules         `json:"defaultValidationRules,omitempty"`
	CompatibilityConfig        *CompatibilityConfig     `json:"compatibilityConfig,omitempty"`
	SecurityConfig             *SecurityConfig          `json:"securityConfig,omitempty"`
	LicenseConfig              *LicenseConfig           `json:"licenseConfig,omitempty"`
	PerformanceConfig          *PerformanceConfig       `json:"performanceConfig,omitempty"`
	PolicyConfig               *PolicyConfig            `json:"policyConfig,omitempty"`
	ConflictAnalyzerConfig     *ConflictAnalyzerConfig  `json:"conflictAnalyzerConfig,omitempty"`
	VulnerabilityDBConfig      *VulnerabilityDBConfig   `json:"vulnerabilityDBConfig,omitempty"`
	LicenseDBConfig            *LicenseDBConfig         `json:"licenseDBConfig,omitempty"`
	CacheConfig                *CacheConfig             `json:"cacheConfig,omitempty"`
}

type CompatibilityConfig struct {
	EnablePlatformChecks    bool          `json:"enablePlatformChecks"`
	EnableVersionChecks     bool          `json:"enableVersionChecks"`
	EnableArchitectureChecks bool         `json:"enableArchitectureChecks"`
	Timeout                 time.Duration `json:"timeout"`
	CacheResults            bool          `json:"cacheResults"`
}

type SecurityConfig struct {
	EnableVulnerabilityScanning bool          `json:"enableVulnerabilityScanning"`
	EnableSecurityPolicyChecks  bool          `json:"enableSecurityPolicyChecks"`
	VulnerabilityDBURL          string        `json:"vulnerabilityDbUrl,omitempty"`
	ScanTimeout                 time.Duration `json:"scanTimeout"`
	CacheResults                bool          `json:"cacheResults"`
	MaxVulnerabilities          int           `json:"maxVulnerabilities"`
}

type LicenseConfig struct {
	EnableLicenseChecks    bool          `json:"enableLicenseChecks"`
	EnableCompatibilityChecks bool       `json:"enableCompatibilityChecks"`
	LicenseDBURL           string        `json:"licenseDbUrl,omitempty"`
	CheckTimeout           time.Duration `json:"checkTimeout"`
	CacheResults           bool          `json:"cacheResults"`
}

type PerformanceConfig struct {
	EnablePerformanceChecks bool          `json:"enablePerformanceChecks"`
	EnableBenchmarking      bool          `json:"enableBenchmarking"`
	BenchmarkTimeout        time.Duration `json:"benchmarkTimeout"`
	ResourceMonitoring      bool          `json:"resourceMonitoring"`
	CacheResults            bool          `json:"cacheResults"`
}

type PolicyConfig struct {
	EnablePolicyChecks bool          `json:"enablePolicyChecks"`
	PolicyEngineURL    string        `json:"policyEngineUrl,omitempty"`
	CheckTimeout       time.Duration `json:"checkTimeout"`
	CacheResults       bool          `json:"cacheResults"`
}

type ConflictAnalyzerConfig struct {
	EnableConflictAnalysis bool          `json:"enableConflictAnalysis"`
	ConflictDetectionDepth int           `json:"conflictDetectionDepth"`
	AnalysisTimeout        time.Duration `json:"analysisTimeout"`
	CacheResults           bool          `json:"cacheResults"`
}

type VulnerabilityDBConfig struct {
	URL          string        `json:"url"`
	APIKey       string        `json:"apiKey,omitempty"`
	UpdateInterval time.Duration `json:"updateInterval"`
	Timeout      time.Duration `json:"timeout"`
}

type LicenseDBConfig struct {
	URL          string        `json:"url"`
	APIKey       string        `json:"apiKey,omitempty"`
	UpdateInterval time.Duration `json:"updateInterval"`
	Timeout      time.Duration `json:"timeout"`
}

type ValidatorMetrics struct {
	TotalValidations      int64         `json:"totalValidations"`
	SuccessfulValidations int64         `json:"successfulValidations"`
	FailedValidations     int64         `json:"failedValidations"`
	SkippedValidations    int64         `json:"skippedValidations"`
	AverageValidationTime time.Duration `json:"averageValidationTime"`
	LastUpdated           time.Time     `json:"lastUpdated"`
	ValidationRate        float64       `json:"validationRate"`
	ErrorRate             float64       `json:"errorRate"`
	CacheHitRate          float64       `json:"cacheHitRate"`
	SecurityScansTotal    Counter       `json:"securityScansTotal"`
	SecurityScanTime      Histogram     `json:"securityScanTime"`
	VulnerabilitiesFound  Counter       `json:"vulnerabilitiesFound"`
	ConflictDetectionTime Histogram     `json:"conflictDetectionTime"`
	ConflictsDetected     Counter       `json:"conflictsDetected"`
	CompatibilityChecks   Counter       `json:"compatibilityChecks"`
	CompatibilityValidationTime Histogram `json:"compatibilityValidationTime"`
	ScanCacheHits         Counter       `json:"scanCacheHits"`
	ScanCacheMisses       Counter       `json:"scanCacheMisses"`
}

type ValidationRules struct {
	Rules          []*ValidationRule      `json:"rules"`
	Rulesets       []*ValidationRuleset   `json:"rulesets,omitempty"`
	DefaultRuleset string                 `json:"defaultRuleset,omitempty"`
	Version        string                 `json:"version"`
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
}

type ValidationRuleset struct {
	RulesetID   string             `json:"rulesetId"`
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Rules       []*ValidationRule  `json:"rules"`
	Enabled     bool               `json:"enabled"`
	Priority    int                `json:"priority"`
	CreatedAt   time.Time          `json:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt"`
}

// Additional types from validator_types.go

type VersionValidation struct {
	Valid        bool      `json:"valid"`
	Version      string    `json:"version"`
	Issues       []string  `json:"issues,omitempty"`
	Warnings     []string  `json:"warnings,omitempty"`
	ValidatedAt  time.Time `json:"validatedAt"`
}

type PackageSecurityValidation struct {
	Valid           bool                     `json:"valid"`
	SecurityScore   float64                  `json:"securityScore"`
	Vulnerabilities []*SecurityVulnerability `json:"vulnerabilities,omitempty"`
	Risks           []*SecurityRisk          `json:"risks,omitempty"`
	ScannedAt       time.Time                `json:"scannedAt"`
}

type SecurityRisk struct {
	RiskID      string    `json:"riskId"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Impact      string    `json:"impact"`
	DetectedAt  time.Time `json:"detectedAt"`
}

type PackageLicenseValidation struct {
	Valid            bool           `json:"valid"`
	License          string         `json:"license"`
	LicenseRisks     []*LicenseRisk `json:"licenseRisks,omitempty"`
	CompatibilityIssues []string    `json:"compatibilityIssues,omitempty"`
	ValidatedAt      time.Time      `json:"validatedAt"`
}

type QualityValidation struct {
	Valid         bool                  `json:"valid"`
	QualityScore  float64               `json:"qualityScore"`
	QualityChecks []*QualityCheck       `json:"qualityChecks"`
	Recommendations []string            `json:"recommendations,omitempty"`
	ValidatedAt   time.Time             `json:"validatedAt"`
}

type QualityCheck struct {
	CheckID     string    `json:"checkId"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Passed      bool      `json:"passed"`
	Score       float64   `json:"score"`
	Description string    `json:"description,omitempty"`
	CheckedAt   time.Time `json:"checkedAt"`
}

type SecurityPolicies struct {
	AllowedSecurityLevels []string        `json:"allowedSecurityLevels"`
	MaxVulnerabilities    int             `json:"maxVulnerabilities"`
	RequiredSecurityScans []string        `json:"requiredSecurityScans"`
	SecurityRules         []*SecurityRule `json:"securityRules,omitempty"`
	ComplianceStandards   []string        `json:"complianceStandards,omitempty"`
}

type SecurityRule struct {
	RuleID      string                 `json:"ruleId"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	Conditions  map[string]interface{} `json:"conditions,omitempty"`
	Actions     []string               `json:"actions"`
	Enabled     bool                   `json:"enabled"`
}

type ComplianceRules struct {
	RequiredStandards     []string              `json:"requiredStandards"`
	ComplianceChecks      []*ComplianceCheck    `json:"complianceChecks"`
	AuditRequirements     []*AuditRequirement   `json:"auditRequirements,omitempty"`
	ReportingRequirements map[string]interface{} `json:"reportingRequirements,omitempty"`
}

type ComplianceCheck struct {
	CheckID      string                 `json:"checkId"`
	Name         string                 `json:"name"`
	Standard     string                 `json:"standard"`
	Type         string                 `json:"type"`
	Description  string                 `json:"description"`
	Requirements map[string]interface{} `json:"requirements"`
	Enabled      bool                   `json:"enabled"`
	Mandatory    bool                   `json:"mandatory"`
}

type AuditRequirement struct {
	RequirementID string                 `json:"requirementId"`
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	Description   string                 `json:"description"`
	Frequency     string                 `json:"frequency"`
	Retention     string                 `json:"retention"`
	Requirements  map[string]interface{} `json:"requirements,omitempty"`
}

type ValidationError struct {
	Code             string                 `json:"code"`
	Type             ErrorType              `json:"type"`
	Severity         ErrorSeverity          `json:"severity"`
	Message          string                 `json:"message"`
	Package          *PackageReference      `json:"package,omitempty"`
	Field            string                 `json:"field,omitempty"`
	Context          map[string]interface{} `json:"context,omitempty"`
	Remediation      string                 `json:"remediation,omitempty"`
	DocumentationURL string                 `json:"documentationUrl,omitempty"`
}

type ErrorSeverity string

const (
	ErrorSeverityLow      ErrorSeverity = "low"
	ErrorSeverityMedium   ErrorSeverity = "medium"
	ErrorSeverityHigh     ErrorSeverity = "high"
	ErrorSeverityCritical ErrorSeverity = "critical"
)

type ValidationStatistics struct {
	TotalPackages         int           `json:"totalPackages"`
	ValidPackages         int           `json:"validPackages"`
	InvalidPackages       int           `json:"invalidPackages"`
	SkippedPackages       int           `json:"skippedPackages"`
	TotalErrors           int           `json:"totalErrors"`
	TotalWarnings         int           `json:"totalWarnings"`
	TotalRecommendations  int           `json:"totalRecommendations"`
	AverageValidationTime time.Duration `json:"averageValidationTime"`
	ValidationDuration    time.Duration `json:"validationDuration"`
	CacheHitRate          float64       `json:"cacheHitRate"`
	ParallelValidations   int           `json:"parallelValidations"`
}

type ValidationRecommendation struct {
	RecommendationID string                 `json:"recommendationId"`
	Type             string                 `json:"type"`
	Priority         string                 `json:"priority"`
	Title            string                 `json:"title"`
	Description      string                 `json:"description"`
	Package          *PackageReference      `json:"package,omitempty"`
	Action           string                 `json:"action"`
	Benefit          string                 `json:"benefit,omitempty"`
	Impact           string                 `json:"impact,omitempty"`
	Effort           string                 `json:"effort,omitempty"`
	Context          map[string]interface{} `json:"context,omitempty"`
	CreatedAt        time.Time              `json:"createdAt"`
}

// Missing interface types

type CompatibilityChecker struct {
	config *CompatibilityConfig
}

func NewCompatibilityChecker(config *CompatibilityConfig) (*CompatibilityChecker, error) {
	return &CompatibilityChecker{config: config}, nil
}

func (c *CompatibilityChecker) CheckCompatibility(ctx context.Context, pkg1, pkg2 *PackageReference) (bool, error) {
	return true, nil // Basic implementation
}

type SecurityScanner struct {
	config *SecurityConfig
}

func NewSecurityScanner(config *SecurityConfig) (*SecurityScanner, error) {
	return &SecurityScanner{config: config}, nil
}

type LicenseValidator struct {
	config *LicenseConfig
}

func NewLicenseValidator(config *LicenseConfig) (*LicenseValidator, error) {
	return &LicenseValidator{config: config}, nil
}

type PerformanceAnalyzer struct {
	config *PerformanceConfig
}

func NewPerformanceAnalyzer(config *PerformanceConfig) (*PerformanceAnalyzer, error) {
	return &PerformanceAnalyzer{config: config}, nil
}

type PolicyEngine struct {
	config *PolicyConfig
}

func NewPolicyEngine(config *PolicyConfig) (*PolicyEngine, error) {
	return &PolicyEngine{config: config}, nil
}

type ConflictDetector interface {
	DetectConflicts(ctx context.Context, packages []*PackageReference) ([]*DependencyConflict, error)
}

type ConflictAnalyzer struct {
	config *ConflictAnalyzerConfig
}

func NewConflictAnalyzer(config *ConflictAnalyzerConfig) *ConflictAnalyzer {
	return &ConflictAnalyzer{config: config}
}

type VulnerabilityDatabase interface {
	GetVulnerabilities(ctx context.Context, pkg *PackageReference) ([]*Vulnerability, error)
	UpdateDatabase(ctx context.Context) error
	GetLastUpdateTime(ctx context.Context) (time.Time, error)
}

type LicenseDatabase interface {
	GetLicenseInfo(ctx context.Context, pkg *PackageReference) (*LicenseInfo, error)
	UpdateDatabase(ctx context.Context) error
	GetLastUpdateTime(ctx context.Context) (time.Time, error)
}

type PolicyRegistry interface {
	GetPolicies(ctx context.Context, pkg *PackageReference) ([]*Policy, error)
	ValidateAgainstPolicies(ctx context.Context, pkg *PackageReference, policies []*Policy) ([]*PolicyViolation, error)
}

type Policy struct {
	PolicyID    string                 `json:"policyId"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Rules       []string               `json:"rules"`
	Severity    string                 `json:"severity"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type ValidationWorkerPool interface {
	SubmitTask(ctx context.Context, task *ValidationTask) error
	GetWorkerStats(ctx context.Context) (*WorkerStats, error)
	Stop(ctx context.Context) error
}

type ValidationTask struct {
	TaskID    string            `json:"taskId"`
	Package   *PackageReference `json:"package"`
	Type      string            `json:"type"`
	Priority  int               `json:"priority"`
	CreatedAt time.Time         `json:"createdAt"`
}

type WorkerStats struct {
	TotalWorkers   int   `json:"totalWorkers"`
	ActiveWorkers  int   `json:"activeWorkers"`
	IdleWorkers    int   `json:"idleWorkers"`
	QueuedTasks    int64 `json:"queuedTasks"`
	CompletedTasks int64 `json:"completedTasks"`
	FailedTasks    int64 `json:"failedTasks"`
}

// Functions and methods

// DefaultValidatorConfig returns the default validator configuration
func DefaultValidatorConfig() *ValidatorConfig {
	return &ValidatorConfig{
		MaxDepth:                    10,
		ConcurrentWorkers:          5,
		Timeout:                     10 * time.Minute,
		EnableCaching:              true,
		RetryAttempts:              3,
		EnableConcurrency:          true,
		WorkerCount:                10,
		QueueSize:                  1000,
		VulnerabilityUpdateInterval: 24 * time.Hour,
		CacheCleanupInterval:       1 * time.Hour,
		MetricsCollectionInterval:  5 * time.Minute,
		DefaultValidationRules: &ValidationRules{
			Rules: []*ValidationRule{},
			Version: "1.0.0",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		CompatibilityConfig: &CompatibilityConfig{
			EnablePlatformChecks:     true,
			EnableVersionChecks:      true,
			EnableArchitectureChecks: true,
			Timeout:                  5 * time.Minute,
			CacheResults:             true,
		},
		SecurityConfig: &SecurityConfig{
			EnableVulnerabilityScanning: true,
			EnableSecurityPolicyChecks:  true,
			ScanTimeout:                 10 * time.Minute,
			CacheResults:                true,
			MaxVulnerabilities:          100,
		},
		LicenseConfig: &LicenseConfig{
			EnableLicenseChecks:       true,
			EnableCompatibilityChecks: true,
			CheckTimeout:              5 * time.Minute,
			CacheResults:              true,
		},
		PerformanceConfig: &PerformanceConfig{
			EnablePerformanceChecks: true,
			EnableBenchmarking:      true,
			BenchmarkTimeout:        15 * time.Minute,
			ResourceMonitoring:      true,
			CacheResults:            true,
		},
		PolicyConfig: &PolicyConfig{
			EnablePolicyChecks: true,
			CheckTimeout:       5 * time.Minute,
			CacheResults:       true,
		},
		ConflictAnalyzerConfig: &ConflictAnalyzerConfig{
			EnableConflictAnalysis: true,
			ConflictDetectionDepth: 5,
			AnalysisTimeout:        10 * time.Minute,
			CacheResults:           true,
		},
		CacheConfig: &CacheConfig{
			DefaultTTL:       1 * time.Hour,
			MaxEntries:       10000,
			CleanupInterval:  10 * time.Minute,
		},
	}
}

// Validate validates the validator configuration
func (c *ValidatorConfig) Validate() error {
	if c.MaxDepth <= 0 {
		return fmt.Errorf("maxDepth must be greater than 0")
	}
	if c.ConcurrentWorkers <= 0 {
		return fmt.Errorf("concurrentWorkers must be greater than 0")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be greater than 0")
	}
	return nil
}

// NewValidatorMetrics creates a new validator metrics instance
func NewValidatorMetrics() *ValidatorMetrics {
	return &ValidatorMetrics{
		TotalValidations:            0,
		SuccessfulValidations:       0,
		FailedValidations:           0,
		SkippedValidations:          0,
		AverageValidationTime:       0,
		LastUpdated:                 time.Now(),
		ValidationRate:              0.0,
		ErrorRate:                   0.0,
		CacheHitRate:                0.0,
		SecurityScansTotal:          NewCounter(),
		SecurityScanTime:            NewHistogram(),
		VulnerabilitiesFound:        NewCounter(),
		ConflictDetectionTime:       NewHistogram(),
		ConflictsDetected:           NewCounter(),
		CompatibilityChecks:         NewCounter(),
		CompatibilityValidationTime: NewHistogram(),
		ScanCacheHits:               NewCounter(),
		ScanCacheMisses:             NewCounter(),
	}
}

// Cache implementations
func NewValidationCache(config *CacheConfig) ValidationCache {
	return &memoryValidationCache{
		cache: make(map[string]*cacheEntry),
	}
}

type memoryValidationCache struct {
	cache map[string]*cacheEntry
	mu    sync.RWMutex
}

type cacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

func (c *memoryValidationCache) Get(ctx context.Context, key string) (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	entry, exists := c.cache[key]
	if !exists {
		return nil, fmt.Errorf("key not found")
	}
	
	if time.Now().After(entry.expiresAt) {
		delete(c.cache, key)
		return nil, fmt.Errorf("key expired")
	}
	
	return entry.value, nil
}

func (c *memoryValidationCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.cache[key] = &cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	
	return nil
}

func (c *memoryValidationCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, key)
	return nil
}

func (c *memoryValidationCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*cacheEntry)
	return nil
}

func (c *memoryValidationCache) Close() {
	c.Clear(context.Background())
}

func NewScanResultCache(config *CacheConfig) ScanResultCache {
	return &memoryScanResultCache{
		cache: make(map[string]*scanCacheEntry),
	}
}

type memoryScanResultCache struct {
	cache map[string]*scanCacheEntry
	mu    sync.RWMutex
}

type scanCacheEntry struct {
	result    *ScanResult
	expiresAt time.Time
}

func (c *memoryScanResultCache) GetScanResult(ctx context.Context, packageID string) (*ScanResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	entry, exists := c.cache[packageID]
	if !exists {
		return nil, fmt.Errorf("package not found in cache")
	}
	
	if time.Now().After(entry.expiresAt) {
		delete(c.cache, packageID)
		return nil, fmt.Errorf("cache entry expired")
	}
	
	return entry.result, nil
}

func (c *memoryScanResultCache) CacheScanResult(ctx context.Context, packageID string, result *ScanResult, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.cache[packageID] = &scanCacheEntry{
		result:    result,
		expiresAt: time.Now().Add(ttl),
	}
	
	return nil
}

func (c *memoryScanResultCache) InvalidatePackage(ctx context.Context, packageID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, packageID)
	return nil
}

func (c *memoryScanResultCache) Set(ctx context.Context, key string, result *SecurityScanResult) error {
	scanResult := &ScanResult{
		PackageID:       key,
		Vulnerabilities: result.Vulnerabilities,
		ScannedAt:       result.ScannedAt,
		Metadata:        make(map[string]interface{}),
	}
	
	return c.CacheScanResult(ctx, key, scanResult, 1*time.Hour)
}

func (c *memoryScanResultCache) Get(ctx context.Context, key string) (*SecurityScanResult, error) {
	scanResult, err := c.GetScanResult(ctx, key)
	if err != nil {
		return nil, err
	}
	
	securityResult := &SecurityScanResult{
		ScanID:          fmt.Sprintf("scan-%d", time.Now().UnixNano()),
		Vulnerabilities: scanResult.Vulnerabilities,
		ScannedAt:       scanResult.ScannedAt,
	}
	
	return securityResult, nil
}

func (c *memoryScanResultCache) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*scanCacheEntry)
}

// External database implementations
func NewVulnerabilityDatabase(config *VulnerabilityDBConfig) (VulnerabilityDatabase, error) {
	return &mockVulnerabilityDB{config: config}, nil
}

type mockVulnerabilityDB struct {
	config *VulnerabilityDBConfig
}

func (db *mockVulnerabilityDB) GetVulnerabilities(ctx context.Context, pkg *PackageReference) ([]*Vulnerability, error) {
	return []*Vulnerability{}, nil
}

func (db *mockVulnerabilityDB) Update(ctx context.Context) error {
	return nil
}

func (db *mockVulnerabilityDB) UpdateDatabase(ctx context.Context) error {
	return db.Update(ctx)
}

func (db *mockVulnerabilityDB) GetLastUpdateTime(ctx context.Context) (time.Time, error) {
	return time.Now(), nil
}

func (db *mockVulnerabilityDB) Close() {}

func NewLicenseDatabase(config *LicenseDBConfig) (LicenseDatabase, error) {
	return &mockLicenseDB{config: config}, nil
}

type mockLicenseDB struct {
	config *LicenseDBConfig
}

func (db *mockLicenseDB) GetLicenseInfo(ctx context.Context, pkg *PackageReference) (*LicenseInfo, error) {
	return &LicenseInfo{
		License:       "MIT",
		Compatibility: "compatible",
		Restrictions:  []string{},
	}, nil
}

func (db *mockLicenseDB) UpdateDatabase(ctx context.Context) error {
	return nil
}

func (db *mockLicenseDB) GetLastUpdateTime(ctx context.Context) (time.Time, error) {
	return time.Now(), nil
}

func (db *mockLicenseDB) Close() {}

func NewValidationWorkerPool(workerCount, queueSize int) ValidationWorkerPool {
	return &mockWorkerPool{
		workerCount: workerCount,
		queueSize:   queueSize,
	}
}

type mockWorkerPool struct {
	workerCount int
	queueSize   int
}

func (p *mockWorkerPool) SubmitTask(ctx context.Context, task *ValidationTask) error {
	return nil
}

func (p *mockWorkerPool) GetWorkerStats(ctx context.Context) (*WorkerStats, error) {
	return &WorkerStats{
		TotalWorkers:   p.workerCount,
		ActiveWorkers:  0,
		IdleWorkers:    p.workerCount,
		QueuedTasks:    0,
		CompletedTasks: 0,
		FailedTasks:    0,
	}, nil
}

func (p *mockWorkerPool) Stop(ctx context.Context) error {
	return nil
}

func (p *mockWorkerPool) Close() {}

// Conflict detector implementations
func NewVersionConflictDetector() ConflictDetector {
	return &versionConflictDetector{}
}

func NewTransitiveConflictDetector() ConflictDetector {
	return &transitiveConflictDetector{}
}

func NewLicenseConflictDetector() ConflictDetector {
	return &licenseConflictDetector{}
}

func NewPolicyConflictDetector() ConflictDetector {
	return &policyConflictDetector{}
}

func NewArchitectureConflictDetector() ConflictDetector {
	return &architectureConflictDetector{}
}

func NewSecurityConflictDetector() ConflictDetector {
	return &securityConflictDetector{}
}

// Stub implementations
type versionConflictDetector struct{}
func (v *versionConflictDetector) DetectConflicts(ctx context.Context, packages []*PackageReference) ([]*DependencyConflict, error) {
	return nil, nil
}

type transitiveConflictDetector struct{}
func (t *transitiveConflictDetector) DetectConflicts(ctx context.Context, packages []*PackageReference) ([]*DependencyConflict, error) {
	return nil, nil
}

type licenseConflictDetector struct{}
func (l *licenseConflictDetector) DetectConflicts(ctx context.Context, packages []*PackageReference) ([]*DependencyConflict, error) {
	return nil, nil
}

type policyConflictDetector struct{}
func (p *policyConflictDetector) DetectConflicts(ctx context.Context, packages []*PackageReference) ([]*DependencyConflict, error) {
	return nil, nil
}

type architectureConflictDetector struct{}
func (a *architectureConflictDetector) DetectConflicts(ctx context.Context, packages []*PackageReference) ([]*DependencyConflict, error) {
	return nil, nil
}

type securityConflictDetector struct{}
func (s *securityConflictDetector) DetectConflicts(ctx context.Context, packages []*PackageReference) ([]*DependencyConflict, error) {
	return nil, nil
}

// Types moved from duplicate implementations

// Additional missing types for validator.go compilation

type BreakingChangeReport struct {
	ReportID        string                  `json:"reportId"`
	Packages        []*PackageReference     `json:"packages"`
	BreakingChanges []*BreakingChange       `json:"breakingChanges"`
	Impact          string                  `json:"impact"`
	Severity        string                  `json:"severity"`
	Recommendations []string                `json:"recommendations,omitempty"`
	GeneratedAt     time.Time               `json:"generatedAt"`
}

type BreakingChange struct {
	ChangeID     string            `json:"changeId"`
	Type         string            `json:"type"`
	Package      *PackageReference `json:"package"`
	FromVersion  string            `json:"fromVersion"`
	ToVersion    string            `json:"toVersion"`
	Description  string            `json:"description"`
	Impact       string            `json:"impact"`
	Severity     string            `json:"severity"`
	AffectedAPIs []string          `json:"affectedApis,omitempty"`
	Migration    string            `json:"migration,omitempty"`
	DetectedAt   time.Time         `json:"detectedAt"`
}

type UpgradeValidation struct {
	Valid           bool                  `json:"valid"`
	FromPackage     *PackageReference     `json:"fromPackage"`
	ToPackage       *PackageReference     `json:"toPackage"`
	UpgradePath     []*UpgradeStep        `json:"upgradePath"`
	BreakingChanges []*BreakingChange     `json:"breakingChanges,omitempty"`
	Risks           []*UpgradeRisk        `json:"risks,omitempty"`
	EstimatedTime   time.Duration         `json:"estimatedTime"`
	Recommendations []string              `json:"recommendations,omitempty"`
	ValidatedAt     time.Time             `json:"validatedAt"`
}

type UpgradeStep struct {
	StepID      string            `json:"stepId"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	FromVersion string            `json:"fromVersion"`
	ToVersion   string            `json:"toVersion"`
	Package     *PackageReference `json:"package"`
	Required    bool              `json:"required"`
	Description string            `json:"description,omitempty"`
	Actions     []string          `json:"actions,omitempty"`
	Order       int               `json:"order"`
}

type UpgradeRisk struct {
	RiskID      string    `json:"riskId"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Impact      string    `json:"impact"`
	Probability float64   `json:"probability"`
	Mitigation  string    `json:"mitigation,omitempty"`
	AssessedAt  time.Time `json:"assessedAt"`
}

type PolicyValidation struct {
	Valid           bool                    `json:"valid"`
	PolicyViolations []*PolicyViolation     `json:"policyViolations,omitempty"`
	PolicyWarnings  []*PolicyWarning        `json:"policyWarnings,omitempty"`
	ValidatedAt     time.Time               `json:"validatedAt"`
}


type ArchitecturalConstraints struct {
	AllowedPatterns    []string                   `json:"allowedPatterns"`
	RestrictedPatterns []string                   `json:"restrictedPatterns"`
	ArchitectureRules  []*ArchitecturalRule       `json:"architectureRules"`
	ComponentLimits    map[string]int             `json:"componentLimits,omitempty"`
	DependencyRules    []*DependencyArchRule      `json:"dependencyRules,omitempty"`
}

type ArchitecturalRule struct {
	RuleID      string                 `json:"ruleId"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Pattern     string                 `json:"pattern"`
	Constraints map[string]interface{} `json:"constraints"`
	Enabled     bool                   `json:"enabled"`
	Severity    string                 `json:"severity"`
}

type DependencyArchRule struct {
	RuleID          string   `json:"ruleId"`
	Name            string   `json:"name"`
	SourcePattern   string   `json:"sourcePattern"`
	TargetPattern   string   `json:"targetPattern"`
	Allowed         bool     `json:"allowed"`
	Description     string   `json:"description,omitempty"`
	Exceptions      []string `json:"exceptions,omitempty"`
	Enabled         bool     `json:"enabled"`
}

type ArchitecturalValidation struct {
	Valid               bool                        `json:"valid"`
	ArchitectureScore   float64                     `json:"architectureScore"`
	ArchitecturalIssues []*ArchitecturalIssue       `json:"architecturalIssues,omitempty"`
	PatternViolations   []*PatternViolation         `json:"patternViolations,omitempty"`
	ValidatedAt         time.Time                   `json:"validatedAt"`
}

type ArchitecturalIssue struct {
	IssueID     string            `json:"issueId"`
	Type        string            `json:"type"`
	Severity    string            `json:"severity"`
	Description string            `json:"description"`
	Component   string            `json:"component,omitempty"`
	Package     *PackageReference `json:"package,omitempty"`
	Resolution  string            `json:"resolution,omitempty"`
	DetectedAt  time.Time         `json:"detectedAt"`
}

type PatternViolation struct {
	ViolationID string            `json:"violationId"`
	Pattern     string            `json:"pattern"`
	Type        string            `json:"type"`
	Severity    string            `json:"severity"`
	Description string            `json:"description"`
	Package     *PackageReference `json:"package,omitempty"`
	Resolution  string            `json:"resolution,omitempty"`
	DetectedAt  time.Time         `json:"detectedAt"`
}

type VersionConflict struct {
	ConflictID       string              `json:"conflictId"`
	Package          *PackageReference   `json:"package"`
	ConflictingVersions []string         `json:"conflictingVersions"`
	RequiredBy       []*PackageReference `json:"requiredBy"`
	Severity         string              `json:"severity"`
	ResolutionStrategy string            `json:"resolutionStrategy,omitempty"`
	DetectedAt       time.Time           `json:"detectedAt"`
}

type PolicyConflict struct {
	ConflictID      string              `json:"conflictId"`
	ConflictingPolicies []string        `json:"conflictingPolicies"`
	AffectedPackages []*PackageReference `json:"affectedPackages"`
	Description     string              `json:"description"`
	Severity        string              `json:"severity"`
	Resolution      string              `json:"resolution,omitempty"`
	DetectedAt      time.Time           `json:"detectedAt"`
}

type ConflictResolutionSuggestion struct {
	SuggestionID   string              `json:"suggestionId"`
	ConflictID     string              `json:"conflictId"`
	Type           string              `json:"type"`
	Strategy       string              `json:"strategy"`
	Description    string              `json:"description"`
	AffectedPackages []*PackageReference `json:"affectedPackages"`
	Actions        []string            `json:"actions"`
	Confidence     float64             `json:"confidence"`
	EstimatedImpact string             `json:"estimatedImpact"`
	Risks          []string            `json:"risks,omitempty"`
}

// Final missing types for complete validator.go compilation

type DependencyConflictReport struct {
	ReportID            string                   `json:"reportId"`
	Packages            []*PackageReference      `json:"packages"`
	DependencyConflicts []*DependencyConflict    `json:"dependencyConflicts"`
	ConflictSummary     *ConflictSummary         `json:"conflictSummary"`
	ResolutionSuggestions []*ConflictResolutionSuggestion `json:"resolutionSuggestions,omitempty"`
	ImpactAnalysis      *ConflictImpactAnalysis  `json:"impactAnalysis,omitempty"`
	GeneratedAt         time.Time                `json:"generatedAt"`
}

type ConflictSummary struct {
	TotalConflicts    int `json:"totalConflicts"`
	CriticalConflicts int `json:"criticalConflicts"`
	HighConflicts     int `json:"highConflicts"`
	MediumConflicts   int `json:"mediumConflicts"`
	LowConflicts      int `json:"lowConflicts"`
}

type ConflictImpactAnalysis struct {
	AnalysisID      string                    `json:"analysisId"`
	Conflicts       []*DependencyConflict     `json:"conflicts"`
	ImpactScore     float64                   `json:"impactScore"`
	AffectedPackages []*PackageReference      `json:"affectedPackages"`
	RiskLevel       string                    `json:"riskLevel"`
	ImpactAreas     []string                  `json:"impactAreas"`
	Recommendations []string                  `json:"recommendations"`
	AnalyzedAt      time.Time                 `json:"analyzedAt"`
}

type ValidatorHealth struct {
	Status         string        `json:"status"`
	LastValidation time.Time     `json:"lastValidation"`
	Uptime         time.Duration `json:"uptime"`
	ErrorRate      float64       `json:"errorRate"`
	CacheStatus    string        `json:"cacheStatus"`
}

// Interface types for caching
type ValidationCache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
	Delete(key string)
	Clear()
	Stats() *CacheStats
}

type ScanResultCache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
	Delete(key string)
	Clear()
}

type LicenseInfo struct {
	License     string   `json:"license"`
	Type        string   `json:"type"`
	Permissions []string `json:"permissions,omitempty"`
	Conditions  []string `json:"conditions,omitempty"`
	Limitations []string `json:"limitations,omitempty"`
	URL         string   `json:"url,omitempty"`
	SPDXID      string   `json:"spdxId,omitempty"`
}

type ScanResult struct {
	ScanID      string               `json:"scanId"`
	Package     *PackageReference    `json:"package"`
	ScanType    string               `json:"scanType"`
	Status      string               `json:"status"`
	Results     interface{}          `json:"results"`
	Errors      []string             `json:"errors,omitempty"`
	ScannedAt   time.Time            `json:"scannedAt"`
	Duration    time.Duration        `json:"duration"`
}

// Additional missing type definitions

type ResourceImpact struct {
	Package        *PackageReference `json:"package"`
	Impact         float64          `json:"impact"`
	ResourceType   string           `json:"resourceType"`
	EstimatedUsage float64          `json:"estimatedUsage"`
	AnalyzedAt     time.Time        `json:"analyzedAt"`
}

type ArchitecturalViolation struct {
	ViolationID  string            `json:"violationId"`
	ConstraintID string            `json:"constraintId"`
	Severity     string            `json:"severity"`
	Description  string            `json:"description"`
	Package      *PackageReference `json:"package,omitempty"`
	DetectedAt   time.Time         `json:"detectedAt"`
}

type CircularDependency struct {
	DependencyID string              `json:"dependencyId"`
	Packages     []*PackageReference `json:"packages"`
	Severity     ConflictSeverity    `json:"severity"`
	Description  string              `json:"description"`
	DetectedAt   time.Time           `json:"detectedAt"`
}

type LayerViolation struct {
	ViolationID string            `json:"violationId"`
	LayerName   string            `json:"layerName"`
	Severity    string            `json:"severity"`
	Description string            `json:"description"`
	Package     *PackageReference `json:"package,omitempty"`
	DetectedAt  time.Time         `json:"detectedAt"`
}

type DependencyViolation struct {
	ViolationID string            `json:"violationId"`
	RuleID      string            `json:"ruleId"`
	Severity    string            `json:"severity"`
	Description string            `json:"description"`
	Source      *PackageReference `json:"source,omitempty"`
	Target      *PackageReference `json:"target,omitempty"`
	DetectedAt  time.Time         `json:"detectedAt"`
}

// ComplianceCheckResult already defined in other file

type AppliedPolicy struct {
	PolicyID    string            `json:"policyId"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Status      string            `json:"status"`
	Package     *PackageReference `json:"package,omitempty"`
	AppliedAt   time.Time         `json:"appliedAt"`
}

type PolicyExemption struct {
	ExemptionID string            `json:"exemptionId"`
	PolicyID    string            `json:"policyId"`
	Reason      string            `json:"reason"`
	GrantedBy   string            `json:"grantedBy"`
	ExpiresAt   time.Time         `json:"expiresAt,omitempty"`
	Package     *PackageReference `json:"package,omitempty"`
	GrantedAt   time.Time         `json:"grantedAt"`
}

type AffectedComponent struct {
	ComponentID string              `json:"componentId"`
	Name        string              `json:"name"`
	Type        string              `json:"type"`
	Impact      float64             `json:"impact"`
	Packages    []*PackageReference `json:"packages,omitempty"`
	AnalyzedAt  time.Time           `json:"analyzedAt"`
}

// MitigationStrategy already defined in other file

// ValidatorHealth already defined earlier in this file

type ChangeImpactAnalysis struct {
	AnalysisID      string              `json:"analysisId"`
	Changes         []*BreakingChange   `json:"changes"`
	ImpactScore     float64             `json:"impactScore"`
	AffectedPackages []*PackageReference `json:"affectedPackages"`
	RiskLevel       string              `json:"riskLevel"`
	Recommendations []string            `json:"recommendations"`
	AnalyzedAt      time.Time           `json:"analyzedAt"`
}

type UpgradeIssue struct {
	IssueID     string `json:"issueId"`
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	FromVersion string `json:"fromVersion"`
	ToVersion   string `json:"toVersion"`
}

type MigrationStep struct {
	StepID      string `json:"stepId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Order       int    `json:"order"`
	Required    bool   `json:"required"`
}

type ResourceViolation struct {
	ViolationID  string  `json:"violationId"`
	ResourceType string  `json:"resourceType"`
	Limit        float64 `json:"limit"`
	Usage        float64 `json:"usage"`
	Severity     string  `json:"severity"`
}

type ResourceBreakdown struct {
	Package  *PackageReference `json:"package"`
	MemoryMB float64          `json:"memoryMb"`
	CPUCores float64          `json:"cpuCores"`
	DiskMB   float64          `json:"diskMb"`
}

// Enum definitions

type ConflictImpact string

const (
	ConflictImpactLow      ConflictImpact = "low"
	ConflictImpactMedium   ConflictImpact = "medium"
	ConflictImpactHigh     ConflictImpact = "high"
	ConflictImpactCritical ConflictImpact = "critical"
)

// ComplianceStatus already defined in types.go

// End of type definitions - all method implementations moved to validator.go