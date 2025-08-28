package dependencies

import (
	"time"
)

// Basic types used across the dependency management system

// PackageReference represents a reference to a package
type PackageReference struct {
	Name       string            `json:"name"`
	Version    string            `json:"version"`
	Repository string            `json:"repository"`
	Namespace  string            `json:"namespace,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// DependencyScope defines the scope of a dependency
type DependencyScope string

const (
	ScopeRuntime     DependencyScope = "runtime"
	ScopeBuild       DependencyScope = "build"
	ScopeTest        DependencyScope = "test"
	ScopeDevelopment DependencyScope = "development"
)

// DependencyType defines the type of dependency
type DependencyType string

const (
	TypeDirect   DependencyType = "direct"
	TypeIndirect DependencyType = "indirect"
	TypeOptional DependencyType = "optional"
)

// GraphConfig configuration for dependency graph analysis
type GraphConfig struct {
	MaxDepth        int  `json:"maxDepth"`
	IncludeIndirect bool `json:"includeIndirect"`
	ResolveConflicts bool `json:"resolveConflicts"`
	EnableCache     bool `json:"enableCache"`
}

// ResolverConfig configuration for dependency resolver
type ResolverConfig struct {
	MaxDepth          int               `json:"maxDepth"`
	ConcurrentWorkers int               `json:"concurrentWorkers"`
	Timeout           time.Duration     `json:"timeout"`
	EnableCaching     bool              `json:"enableCaching"`
	RetryAttempts     int               `json:"retryAttempts"`
	DefaultStrategy   ResolutionStrategy `json:"defaultStrategy"`
}

// DependencyProvider represents a source for dependency resolution
type DependencyProvider struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	URL         string            `json:"url"`
	Priority    int               `json:"priority"`
	Enabled     bool              `json:"enabled"`
	Credentials map[string]string `json:"credentials,omitempty"`
}

// ConflictResolution represents the resolution of a dependency conflict
type ConflictResolution struct {
	ConflictID  string    `json:"conflictId"`
	ResolvedAt  time.Time `json:"resolvedAt"`
	Strategy    string    `json:"strategy"`
	Resolution  string    `json:"resolution"`
	AppliedBy   string    `json:"appliedBy,omitempty"`
}

// ValidationRule represents a rule for dependency validation
type ValidationRule struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Severity    string      `json:"severity"`
	Description string      `json:"description"`
	Config      interface{} `json:"config,omitempty"`
	Enabled     bool        `json:"enabled"`
}

// ValidationStats represents validation statistics
type ValidationStats struct {
	TotalPackages   int           `json:"totalPackages"`
	ValidatedAt     time.Time     `json:"validatedAt"`
	Duration        time.Duration `json:"duration"`
	ErrorCount      int           `json:"errorCount"`
	WarningCount    int           `json:"warningCount"`
	RulesExecuted   int           `json:"rulesExecuted"`
	PackagesSkipped int           `json:"packagesSkipped"`
}

// CostSavings represents potential cost savings
type CostSavings struct {
	BinarySize   int64   `json:"binarySize"`
	BuildTime    int64   `json:"buildTime"` // in milliseconds
	Dependencies int     `json:"dependencies"`
	Percentage   float64 `json:"percentage"`
}

// CostMetrics represents various cost metrics
type CostMetrics struct {
	BinarySize    int64         `json:"binarySize"`
	BuildTime     time.Duration `json:"buildTime"`
	Dependencies  int           `json:"dependencies"`
	Complexity    float64       `json:"complexity"`
	Maintenance   float64       `json:"maintenance"`
}

// DependencyCost represents the cost of a single dependency
type DependencyCost struct {
	Package     *PackageReference `json:"package"`
	DirectCost  *CostMetrics      `json:"directCost"`
	IndirectCost *CostMetrics     `json:"indirectCost,omitempty"`
	Usage       string            `json:"usage"`
	Critical    bool              `json:"critical"`
}

// HealthMetricData represents a health metric data point
type HealthMetricData struct {
	Timestamp time.Time `json:"timestamp"`
	Score     float64   `json:"score"`
	Issues    int       `json:"issues"`
	Warnings  int       `json:"warnings"`
}

// HealthPrediction represents a predicted health state
type HealthPrediction struct {
	Date        time.Time `json:"date"`
	Score       float64   `json:"score"`
	Confidence  float64   `json:"confidence"`
	Factors     []string  `json:"factors"`
}

// SecurityVulnerability represents a security vulnerability
type SecurityVulnerability struct {
	ID          string    `json:"id"`
	CVE         string    `json:"cve,omitempty"`
	Severity    string    `json:"severity"`
	Score       float64   `json:"score"`
	Description string    `json:"description"`
	FixVersion  string    `json:"fixVersion,omitempty"`
	PublishedAt time.Time `json:"publishedAt"`
}

// LicenseRisk represents a license-related risk
type LicenseRisk struct {
	License     string   `json:"license"`
	RiskLevel   string   `json:"riskLevel"`
	Issues      []string `json:"issues"`
	Compatibility string `json:"compatibility"`
}

// DependencyConstraints represents constraints for dependency analysis
type DependencyConstraints struct {
	MaxDepth         int                  `json:"maxDepth"`
	AllowedScopes    []DependencyScope    `json:"allowedScopes"`
	ExcludePackages  []*PackageReference  `json:"excludePackages,omitempty"`
	VersionPins      []*VersionPin        `json:"versionPins,omitempty"`
	SecurityPolicy   *SecurityPolicy      `json:"securityPolicy,omitempty"`
	LicensePolicy    *LicensePolicy       `json:"licensePolicy,omitempty"`
}

// PerformanceAnalysis represents performance analysis results
type PerformanceAnalysis struct {
	OperationType   string                  `json:"operationType"`
	Duration        time.Duration           `json:"duration"`
	MemoryUsage     int64                   `json:"memoryUsage"`
	Bottlenecks     []*PerformanceBottleneck `json:"bottlenecks,omitempty"`
	Recommendations []string                `json:"recommendations,omitempty"`
	AnalyzedAt      time.Time               `json:"analyzedAt"`
}

// PerformanceBottleneck represents a performance bottleneck
type PerformanceBottleneck struct {
	Component   string        `json:"component"`
	Type        string        `json:"type"`
	Impact      string        `json:"impact"`
	Duration    time.Duration `json:"duration"`
	Suggestion  string        `json:"suggestion"`
}

// EvolutionPrediction represents predictions about dependency evolution
type EvolutionPrediction struct {
	Package        *PackageReference    `json:"package"`
	PredictionType string               `json:"predictionType"`
	Probability    float64              `json:"probability"`
	Timeline       *TimeRange           `json:"timeline"`
	Factors        []string             `json:"factors"`
	Confidence     float64              `json:"confidence"`
	Impact         *PredictionImpact    `json:"impact,omitempty"`
	GeneratedAt    time.Time            `json:"generatedAt"`
}

// PredictionImpact represents the impact of a prediction
type PredictionImpact struct {
	Severity        string   `json:"severity"`
	AffectedPackages int     `json:"affectedPackages"`
	Description     string   `json:"description"`
	Mitigation      []string `json:"mitigation,omitempty"`
}

// GraphAnalysis represents comprehensive graph analysis results
type GraphAnalysis struct {
	TotalNodes      int                    `json:"totalNodes"`
	TotalEdges      int                    `json:"totalEdges"`
	CriticalPaths   []*CriticalPath        `json:"criticalPaths"`
	Cycles          []*DependencyCycle     `json:"cycles,omitempty"`
	Patterns        []*GraphPattern        `json:"patterns,omitempty"`
	Metrics         *GraphMetrics          `json:"metrics"`
	AnalyzedAt      time.Time              `json:"analyzedAt"`
}

// OptimizationObjectives represents objectives for dependency optimization
type OptimizationObjectives struct {
	MinimizeBinarySize  bool    `json:"minimizeBinarySize"`
	MinimizeBuildTime   bool    `json:"minimizeBuildTime"`
	MaximizeStability   bool    `json:"maximizeStability"`
	MinimizeComplexity  bool    `json:"minimizeComplexity"`
	SecurityWeight      float64 `json:"securityWeight"`
	PerformanceWeight   float64 `json:"performanceWeight"`
	MaintainabilityWeight float64 `json:"maintainabilityWeight"`
}

// OptimizationRecommendations represents optimization recommendations
type OptimizationRecommendations struct {
	PackageName     string                        `json:"packageName"`
	Recommendations []*OptimizationRecommendation `json:"recommendations"`
	ExpectedImpact  *OptimizationImpact           `json:"expectedImpact"`
	Priority        string                        `json:"priority"`
	GeneratedAt     time.Time                     `json:"generatedAt"`
}

// OptimizationRecommendation represents a single optimization recommendation
type OptimizationRecommendation struct {
	Type        string            `json:"type"`
	Action      string            `json:"action"`
	Package     *PackageReference `json:"package,omitempty"`
	Reason      string            `json:"reason"`
	Impact      *OptimizationImpact `json:"impact"`
	Effort      string            `json:"effort"`
	Risk        string            `json:"risk"`
}

// OptimizationImpact represents the impact of an optimization
type OptimizationImpact struct {
	BinarySizeReduction int64         `json:"binarySizeReduction"`
	BuildTimeReduction  time.Duration `json:"buildTimeReduction"`
	ComplexityReduction float64       `json:"complexityReduction"`
	SecurityImprovement float64       `json:"securityImprovement"`
}

// UpdateStep represents a single step in an update plan
type UpdateStep struct {
	StepID      string        `json:"stepId"`
	Name        string        `json:"name"`
	Type        string        `json:"type"`
	Order       int           `json:"order"`
	Required    bool          `json:"required"`
	Timeout     time.Duration `json:"timeout"`
	Config      interface{}   `json:"config,omitempty"`
	Status      string        `json:"status,omitempty"`
}

// RollbackPlan represents a plan for rolling back dependency changes
type RollbackPlan struct {
	PlanID          string                   `json:"planId"`
	CreatedAt       time.Time                `json:"createdAt"`
	RollbackSteps   []*RollbackStep          `json:"rollbackSteps"`
	TargetVersion   string                   `json:"targetVersion"`
	Reason          string                   `json:"reason"`
	EstimatedTime   time.Duration            `json:"estimatedTime"`
	RiskAssessment  *RollbackRiskAssessment  `json:"riskAssessment,omitempty"`
	Validation      *RollbackValidation      `json:"validation,omitempty"`
}

// RollbackStep represents a single step in a rollback plan
type RollbackStep struct {
	StepID       string            `json:"stepId"`
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	Order        int               `json:"order"`
	Package      *PackageReference `json:"package,omitempty"`
	FromVersion  string            `json:"fromVersion"`
	ToVersion    string            `json:"toVersion"`
	Required     bool              `json:"required"`
	Timeout      time.Duration     `json:"timeout"`
	Dependencies []*PackageReference `json:"dependencies,omitempty"`
	Config       interface{}       `json:"config,omitempty"`
	Status       string            `json:"status,omitempty"`
}

// RollbackRiskAssessment represents risk assessment for a rollback
type RollbackRiskAssessment struct {
	RiskLevel       string                  `json:"riskLevel"`
	RiskFactors     []*RollbackRiskFactor   `json:"riskFactors"`
	Mitigations     []*RiskMitigation       `json:"mitigations,omitempty"`
	ApprovalRequired bool                   `json:"approvalRequired"`
	AssessedAt      time.Time               `json:"assessedAt"`
}

// RollbackRiskFactor represents a factor contributing to rollback risk
type RollbackRiskFactor struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Severity    string  `json:"severity"`
	Impact      string  `json:"impact"`
	Probability float64 `json:"probability"`
}

// RiskMitigation represents a mitigation strategy for a risk
type RiskMitigation struct {
	RiskType    string `json:"riskType"`
	Strategy    string `json:"strategy"`
	Description string `json:"description"`
	Effectiveness float64 `json:"effectiveness"`
}

// RollbackValidation represents validation for a rollback plan
type RollbackValidation struct {
	ValidatedAt     time.Time                  `json:"validatedAt"`
	Valid           bool                       `json:"valid"`
	ValidationRules []*RollbackValidationRule  `json:"validationRules"`
	Issues          []*RollbackValidationIssue `json:"issues,omitempty"`
	Warnings        []*RollbackValidationWarning `json:"warnings,omitempty"`
}

// RollbackValidationRule represents a validation rule for rollbacks
type RollbackValidationRule struct {
	RuleID      string `json:"ruleId"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Passed      bool   `json:"passed"`
}

// RollbackValidationIssue represents a validation issue for rollbacks
type RollbackValidationIssue struct {
	IssueID     string            `json:"issueId"`
	Type        string            `json:"type"`
	Message     string            `json:"message"`
	Package     *PackageReference `json:"package,omitempty"`
	Severity    string            `json:"severity"`
	Rule        string            `json:"rule,omitempty"`
	Blocking    bool              `json:"blocking"`
}

// RollbackValidationWarning represents a validation warning for rollbacks
type RollbackValidationWarning struct {
	WarningID   string            `json:"warningId"`
	Type        string            `json:"type"`
	Message     string            `json:"message"`
	Package     *PackageReference `json:"package,omitempty"`
	Severity    string            `json:"severity"`
	Rule        string            `json:"rule,omitempty"`
	Suggestion  string            `json:"suggestion,omitempty"`
}

// Missing types from analyzer.go and graph.go
type Vulnerability struct {
	ID          string    `json:"id"`
	CVE         string    `json:"cve,omitempty"`
	Severity    string    `json:"severity"`
	Score       float64   `json:"score"`
	Description string    `json:"description"`
	FixVersion  string    `json:"fixVersion,omitempty"`
	PublishedAt time.Time `json:"publishedAt"`
}

type ComplianceStatus string

const (
	ComplianceStatusCompliant    ComplianceStatus = "compliant"
	ComplianceStatusNonCompliant ComplianceStatus = "non-compliant"
	ComplianceStatusUnknown      ComplianceStatus = "unknown"
)

type PackageInfo struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Repository  string            `json:"repository"`
	Homepage    string            `json:"homepage,omitempty"`
	Description string            `json:"description,omitempty"`
	License     string            `json:"license,omitempty"`
	Maintainers []string          `json:"maintainers,omitempty"`
	Keywords    []string          `json:"keywords,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type SecurityMetrics struct {
	VulnerabilityCount int     `json:"vulnerabilityCount"`
	HighSeverityCount  int     `json:"highSeverityCount"`
	TrustScore         float64 `json:"trustScore"`
	LastScanned        time.Time `json:"lastScanned"`
	ComplianceScore    float64 `json:"complianceScore"`
}

// Additional missing types from graph.go and updater.go
type EdgeType string

const (
	EdgeTypeDependency EdgeType = "dependency"
	EdgeTypeConflict   EdgeType = "conflict" 
	EdgeTypeOptional   EdgeType = "optional"
)

type DependencyEdge struct {
	Source      *DependencyNode   `json:"source"`
	Target      *DependencyNode   `json:"target"`
	Type        EdgeType          `json:"type"`
	Constraint  string            `json:"constraint"`
	Weight      float64           `json:"weight"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type ImpactAnalysis struct {
	Impact          string    `json:"impact"`
	AffectedNodes   int       `json:"affectedNodes"`
	CriticalityScore float64  `json:"criticalityScore"`
	Description     string    `json:"description"`
	AnalyzedAt      time.Time `json:"analyzedAt"`
}

type Repository struct {
	Name        string            `json:"name"`
	URL         string            `json:"url"`
	Type        string            `json:"type"`
	Credentials map[string]string `json:"credentials,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type UpdateStrategy string

const (
	UpdateStrategyConservative UpdateStrategy = "conservative"
	UpdateStrategyAggressive   UpdateStrategy = "aggressive"
	UpdateStrategyBalanced     UpdateStrategy = "balanced"
)

type UpdateConstraints struct {
	MaxVersionJump  int           `json:"maxVersionJump"`
	AllowBreaking   bool          `json:"allowBreaking"`
	TestingRequired bool          `json:"testingRequired"`
	ApprovalNeeded  bool          `json:"approvalNeeded"`
	Timeout         time.Duration `json:"timeout"`
}

type SecurityConstraints struct {
	MaxVulnerabilities int      `json:"maxVulnerabilities"`
	BlockedSeverities  []string `json:"blockedSeverities"`
	RequireScanning    bool     `json:"requireScanning"`
	TrustMinimum       float64  `json:"trustMinimum"`
}

type CompatibilityRules struct {
	RequireCompatibility bool              `json:"requireCompatibility"`
	ExcludeIncompatible  bool              `json:"excludeIncompatible"`
	PlatformConstraints  map[string]string `json:"platformConstraints,omitempty"`
}

type RolloutConfig struct {
	BatchSize       int           `json:"batchSize"`
	MaxConcurrency  int           `json:"maxConcurrency"`
	RolloutDelay    time.Duration `json:"rolloutDelay"`
	HealthChecks    bool          `json:"healthChecks"`
	RollbackOnError bool          `json:"rollbackOnError"`
	StagedRollout   bool          `json:"stagedRollout"` // Added for staged rollout support
}

// Final missing types from updater.go
type ValidationConfig struct {
	EnableValidation     bool          `json:"enableValidation"`
	ValidationRules      []string      `json:"validationRules"`
	StrictMode          bool          `json:"strictMode"`
	ContinueOnWarnings  bool          `json:"continueOnWarnings"`
	ValidationTimeout   time.Duration `json:"validationTimeout"`
}

type ApprovalPolicy struct {
	RequireApproval    bool     `json:"requireApproval"`
	ApprovalRoles      []string `json:"approvalRoles"`
	AutoApprovalRules  []string `json:"autoApprovalRules"`
	ApprovalTimeout    time.Duration `json:"approvalTimeout"`
}

type NotificationConfig struct {
	EnableNotifications bool     `json:"enableNotifications"`
	Channels           []string `json:"channels"`
	Templates          map[string]string `json:"templates,omitempty"`
	Recipients         []string `json:"recipients"`
}

type UpdatedPackage struct {
	Package     *PackageReference `json:"package"`
	OldVersion  string            `json:"oldVersion"`
	NewVersion  string            `json:"newVersion"`
	UpdatedAt   time.Time         `json:"updatedAt"`
	UpdateTime  time.Duration     `json:"updateTime"`
}

type FailedUpdate struct {
	Package   *PackageReference `json:"package"`
	Reason    string            `json:"reason"`
	Error     string            `json:"error"`
	FailedAt  time.Time         `json:"failedAt"`
	Retryable bool              `json:"retryable"`
}

type SkippedUpdate struct {
	Package  *PackageReference `json:"package"`
	Reason   string            `json:"reason"`
	SkippedAt time.Time        `json:"skippedAt"`
}

type RolloutExecution struct {
	ExecutionID   string        `json:"executionId"`
	StartedAt     time.Time     `json:"startedAt"`
	CompletedAt   time.Time     `json:"completedAt,omitempty"`
	Status        string        `json:"status"`
	TotalBatches  int           `json:"totalBatches"`
	CurrentBatch  int           `json:"currentBatch"`
	ExecutionTime time.Duration `json:"executionTime"`
}

type ApprovalRequest struct {
	RequestID     string            `json:"requestId"`
	UpdatePackage *PackageReference `json:"updatePackage"`
	Requester     string            `json:"requester"`
	RequestedAt   time.Time         `json:"requestedAt"`
	Status        string            `json:"status"`
	Approver      string            `json:"approver,omitempty"`
	ApprovedAt    time.Time         `json:"approvedAt,omitempty"`
	Comments      string            `json:"comments,omitempty"`
}

type UpdateError struct {
	Code        string    `json:"code"`
	Type        string    `json:"type"`
	Message     string    `json:"message"`
	Package     *PackageReference `json:"package,omitempty"`
	Severity    string    `json:"severity"`
	Retryable   bool      `json:"retryable"`
	OccurredAt  time.Time `json:"occurredAt"`
}

type UpdateWarning struct {
	Code        string    `json:"code"`
	Type        string    `json:"type"`
	Message     string    `json:"message"`
	Package     *PackageReference `json:"package,omitempty"`
	Severity    string    `json:"severity"`
	Suggestion  string    `json:"suggestion,omitempty"`
	OccurredAt  time.Time `json:"occurredAt"`
}

// ValidationError and ValidationWarning - defined elsewhere to avoid duplicates
type UpdateStatistics struct {
	TotalUpdates    int           `json:"totalUpdates"`
	SuccessfulUpdates int         `json:"successfulUpdates"`
	FailedUpdates   int           `json:"failedUpdates"`
	SkippedUpdates  int           `json:"skippedUpdates"`
	UpdateTime      time.Duration `json:"updateTime"`
	StartedAt       time.Time     `json:"startedAt"`
	CompletedAt     time.Time     `json:"completedAt,omitempty"`
}

type UpdateType string
const (
	UpdateTypeMinor    UpdateType = "minor"
	UpdateTypeMajor    UpdateType = "major"
	UpdateTypePatch    UpdateType = "patch"
	UpdateTypeSecurity UpdateType = "security"
)

type UpdateReason string
const (
	UpdateReasonSecurity     UpdateReason = "security"
	UpdateReasonBugfix       UpdateReason = "bugfix"
	UpdateReasonFeature      UpdateReason = "feature"
	UpdateReasonMaintenance  UpdateReason = "maintenance"
)

type UpdatePriority string
const (
	UpdatePriorityLow      UpdatePriority = "low"
	UpdatePriorityMedium   UpdatePriority = "medium"
	UpdatePriorityHigh     UpdatePriority = "high"
	UpdatePriorityCritical UpdatePriority = "critical"
)

type MaintenanceWindow struct {
	Start       time.Time     `json:"start"`
	End         time.Time     `json:"end"`
	Timezone    string        `json:"timezone"`
	Recurring   bool          `json:"recurring"`
	Duration    time.Duration `json:"duration"`
	Description string        `json:"description,omitempty"`
}

type PropagationFilter struct {
	Environments   []string `json:"environments"`
	ExcludePatterns []string `json:"excludePatterns,omitempty"`
	IncludePatterns []string `json:"includePatterns,omitempty"`
	RequiredApproval bool    `json:"requiredApproval"`
}

type PropagatedUpdate struct {
	Environment string            `json:"environment"`
	Package     *PackageReference `json:"package"`
	Status      string            `json:"status"`
	UpdatedAt   time.Time         `json:"updatedAt"`
	Duration    time.Duration     `json:"duration"`
}

type FailedPropagation struct {
	Environment string            `json:"environment"`
	Package     *PackageReference `json:"package"`
	Error       string            `json:"error"`
	FailedAt    time.Time         `json:"failedAt"`
	Retryable   bool              `json:"retryable"`
}

type SkippedPropagation struct {
	Environment string            `json:"environment"`
	Package     *PackageReference `json:"package"`
	Reason      string            `json:"reason"`
	SkippedAt   time.Time         `json:"skippedAt"`
}

type EnvironmentUpdateResult struct {
	Environment        string               `json:"environment"`
	PropagatedUpdates  []*PropagatedUpdate  `json:"propagatedUpdates"`
	FailedPropagations []*FailedPropagation `json:"failedPropagations"`
	SkippedPropagations []*SkippedPropagation `json:"skippedPropagations"`
	Statistics         *UpdateStatistics     `json:"statistics"`
	CompletedAt        time.Time            `json:"completedAt"`
}

type AutoUpdateConfig struct {
	Enabled          bool          `json:"enabled"`
	Schedule         string        `json:"schedule"` 
	SecurityOnly     bool          `json:"securityOnly"`
	MaxVersionJump   int           `json:"maxVersionJump"`
	TestingRequired  bool          `json:"testingRequired"`
	ApprovalRequired bool          `json:"approvalRequired"`
	MaintenanceWindow *MaintenanceWindow `json:"maintenanceWindow,omitempty"`
}

type UpdateSchedule struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	CronExpression string            `json:"cronExpression"`
	Enabled        bool              `json:"enabled"`
	Packages       []*PackageReference `json:"packages"`
	Config         *AutoUpdateConfig `json:"config"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
}

type ScheduledUpdate struct {
	ScheduleID     string            `json:"scheduleId"`
	Package        *PackageReference `json:"package"`
	TargetVersion  string            `json:"targetVersion"`
	ScheduledFor   time.Time         `json:"scheduledFor"`
	Status         string            `json:"status"`
	ExecutedAt     time.Time         `json:"executedAt,omitempty"`
	CompletedAt    time.Time         `json:"completedAt,omitempty"`
}

type UpdatePlan struct {
	PlanID          string              `json:"planId"`
	Name            string              `json:"name"`
	Description     string              `json:"description,omitempty"`
	Packages        []*PackageReference `json:"packages"`
	UpdateSteps     []*UpdateStep       `json:"updateSteps"`
	Dependencies    []*PackageReference `json:"dependencies,omitempty"`
	Prerequisites   []string            `json:"prerequisites,omitempty"`
	EstimatedTime   time.Duration       `json:"estimatedTime"`
	RiskAssessment  *RiskAssessment     `json:"riskAssessment,omitempty"`
	RollbackPlan    *RollbackPlan       `json:"rollbackPlan,omitempty"`
	CreatedAt       time.Time           `json:"createdAt"`
	CreatedBy       string              `json:"createdBy"`
	Status          string              `json:"status"`
}

type PlanValidation struct {
	PlanID      string                `json:"planId"`
	Valid       bool                  `json:"valid"`
	Issues      []*UpdateValidationError    `json:"issues,omitempty"`
	Warnings    []*UpdateValidationWarning  `json:"warnings,omitempty"`
	ValidatedAt time.Time             `json:"validatedAt"`
	ValidatedBy string                `json:"validatedBy"`
}

type RiskAssessment struct {
	RiskLevel   string        `json:"riskLevel"`
	RiskFactors []string      `json:"riskFactors"`
	Mitigations []string      `json:"mitigations,omitempty"`
	AssessedAt  time.Time     `json:"assessedAt"`
	AssessedBy  string        `json:"assessedBy"`
}

type RollbackResult struct {
	PlanID         string              `json:"planId"`
	Success        bool                `json:"success"`
	RolledBackPackages []*PackageReference `json:"rolledBackPackages"`
	FailedRollbacks    []*FailedRollback   `json:"failedRollbacks,omitempty"`
	CompletedAt        time.Time           `json:"completedAt"`
	RollbackTime       time.Duration       `json:"rollbackTime"`
}

type FailedRollback struct {
	Package   *PackageReference `json:"package"`
	Reason    string            `json:"reason"`
	Error     string            `json:"error"`
	FailedAt  time.Time         `json:"failedAt"`
}

type PropagationError struct {
	Code        string    `json:"code"`
	Type        string    `json:"type"`
	Message     string    `json:"message"`
	Environment string    `json:"environment"`
	Package     *PackageReference `json:"package,omitempty"`
	Severity    string    `json:"severity"`
	Retryable   bool      `json:"retryable"`
	OccurredAt  time.Time `json:"occurredAt"`
}

type PropagationWarning struct {
	Code        string    `json:"code"`
	Type        string    `json:"type"`
	Message     string    `json:"message"`
	Environment string    `json:"environment"`
	Package     *PackageReference `json:"package,omitempty"`
	Severity    string    `json:"severity"`
	Suggestion  string    `json:"suggestion,omitempty"`
	OccurredAt  time.Time `json:"occurredAt"`
}
