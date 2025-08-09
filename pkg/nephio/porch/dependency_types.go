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

package porch

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Supporting types for dependency management

// DependencyResolverConfig configures the dependency resolver
type DependencyResolverConfig struct {
	WorkerCount            int
	QueueSize              int
	EnableCaching          bool
	CacheTimeout           time.Duration
	MaxGraphDepth          int
	MaxResolutionTime      time.Duration
	VersionSolverConfig    *VersionSolverConfig
	ConflictResolverConfig *ConflictResolverConfig
	GraphBuilderConfig     *GraphBuilderConfig
	HealthCheckerConfig    *HealthCheckerConfig
	CacheConfig            *CacheConfig
}

// VersionSolverConfig configures the version solver
type VersionSolverConfig struct {
	SATSolverConfig *SATSolverConfig
	MaxIterations   int
	Timeout         time.Duration
}

// ConflictResolverConfig configures the conflict resolver
type ConflictResolverConfig struct {
	MaxRetries      int
	BackoffStrategy string
	Timeout         time.Duration
}

// GraphBuilderConfig configures the graph builder
type GraphBuilderConfig struct {
	MaxDepth           int
	ParallelProcessing bool
	BatchSize          int
}

// HealthCheckerConfig configures the health checker
type HealthCheckerConfig struct {
	CheckInterval    time.Duration
	MetricsRetention time.Duration
}

// CacheConfig configures caching
type CacheConfig struct {
	MaxSize        int
	TTL            time.Duration
	EvictionPolicy string
}

// DependencyResolverMetrics tracks resolver metrics
type DependencyResolverMetrics struct {
	resolutionsTotal        *prometheus.CounterVec
	resolutionTime          *prometheus.Histogram
	resolutionCacheHits     *prometheus.Counter
	resolutionCacheMisses   *prometheus.Counter
	graphsBuilt             *prometheus.Counter
	graphBuildTime          *prometheus.Histogram
	graphNodeCount          *prometheus.Histogram
	graphEdgeCount          *prometheus.Histogram
	graphCacheHits          *prometheus.Counter
	graphCacheMisses        *prometheus.Counter
	versionSolveTime        *prometheus.Histogram
	versionSolvesTotal      *prometheus.CounterVec
	updatePropagationsTotal *prometheus.Counter
	updatePropagationTime   *prometheus.Histogram
	propagatedUpdates       *prometheus.Counter
}

// Version Solver supporting types

type VersionSolver struct {
	satSolver *SATSolver
	config    *VersionSolverConfig
	logger    logr.Logger
}

func NewVersionSolver(config *VersionSolverConfig) *VersionSolver {
	return &VersionSolver{
		satSolver: NewSATSolver(config.SATSolverConfig),
		config:    config,
		logger:    log.Log.WithName("version-solver"),
	}
}

func (vs *VersionSolver) Solve(ctx context.Context, requirements []*VersionRequirement) (*VersionSolution, error) {
	return vs.satSolver.Solve(ctx, requirements)
}

func (vs *VersionSolver) Close() {
	vs.satSolver.Close()
}

// Conflict Resolver supporting types

type DependencyConflictResolver struct {
	config *ConflictResolverConfig
	logger logr.Logger
}

func NewDependencyConflictResolver(config *ConflictResolverConfig) *DependencyConflictResolver {
	return &DependencyConflictResolver{
		config: config,
		logger: log.Log.WithName("conflict-resolver"),
	}
}

func (dcr *DependencyConflictResolver) Close() {
	// Cleanup if needed
}

// Graph Builder supporting types

type DependencyGraphBuilder struct {
	config *GraphBuilderConfig
	logger logr.Logger
}

func NewDependencyGraphBuilder(config *GraphBuilderConfig) *DependencyGraphBuilder {
	return &DependencyGraphBuilder{
		config: config,
		logger: log.Log.WithName("graph-builder"),
	}
}

func (dgb *DependencyGraphBuilder) BuildGraph(ctx context.Context, graph *DependencyGraph, opts *GraphBuildOptions) error {
	// Implementation would build the graph based on options
	return nil
}

// Health Checker supporting types

type DependencyHealthChecker struct {
	config *HealthCheckerConfig
	logger logr.Logger
}

func NewDependencyHealthChecker(config *HealthCheckerConfig) *DependencyHealthChecker {
	return &DependencyHealthChecker{
		config: config,
		logger: log.Log.WithName("health-checker"),
	}
}

// Cache supporting types

type ResolutionCache interface {
	Get(ctx context.Context, key string) (*ResolutionResult, error)
	Set(ctx context.Context, key string, result *ResolutionResult) error
	Close()
}

type GraphCache interface {
	Get(ctx context.Context, key string) (*DependencyGraph, error)
	Set(ctx context.Context, key string, graph *DependencyGraph) error
	Close()
}

type VersionCache interface {
	Get(ctx context.Context, key string) ([]string, error)
	Set(ctx context.Context, key string, versions []string) error
	Close()
}

func NewResolutionCache(config *CacheConfig) ResolutionCache {
	// Implementation would create actual cache
	return &resolutionCacheImpl{config: config}
}

func NewGraphCache(config *CacheConfig) GraphCache {
	// Implementation would create actual cache
	return &graphCacheImpl{config: config}
}

func NewVersionCache(config *CacheConfig) VersionCache {
	// Implementation would create actual cache
	return &versionCacheImpl{config: config}
}

type resolutionCacheImpl struct {
	config *CacheConfig
	cache  map[string]*ResolutionResult
	mu     sync.RWMutex
}

func (rc *resolutionCacheImpl) Get(ctx context.Context, key string) (*ResolutionResult, error) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	if result, ok := rc.cache[key]; ok {
		return result, nil
	}
	return nil, fmt.Errorf("cache miss")
}

func (rc *resolutionCacheImpl) Set(ctx context.Context, key string, result *ResolutionResult) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if rc.cache == nil {
		rc.cache = make(map[string]*ResolutionResult)
	}
	rc.cache[key] = result
	return nil
}

func (rc *resolutionCacheImpl) Close() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.cache = nil
}

type graphCacheImpl struct {
	config *CacheConfig
	cache  map[string]*DependencyGraph
	mu     sync.RWMutex
}

func (gc *graphCacheImpl) Get(ctx context.Context, key string) (*DependencyGraph, error) {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	if graph, ok := gc.cache[key]; ok {
		return graph, nil
	}
	return nil, fmt.Errorf("cache miss")
}

func (gc *graphCacheImpl) Set(ctx context.Context, key string, graph *DependencyGraph) error {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	if gc.cache == nil {
		gc.cache = make(map[string]*DependencyGraph)
	}
	gc.cache[key] = graph
	return nil
}

func (gc *graphCacheImpl) Close() {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	gc.cache = nil
}

type versionCacheImpl struct {
	config *CacheConfig
	cache  map[string][]string
	mu     sync.RWMutex
}

func (vc *versionCacheImpl) Get(ctx context.Context, key string) ([]string, error) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	if versions, ok := vc.cache[key]; ok {
		return versions, nil
	}
	return nil, fmt.Errorf("cache miss")
}

func (vc *versionCacheImpl) Set(ctx context.Context, key string, versions []string) error {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	if vc.cache == nil {
		vc.cache = make(map[string][]string)
	}
	vc.cache[key] = versions
	return nil
}

func (vc *versionCacheImpl) Close() {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	vc.cache = nil
}

// Resolver Pool supporting types

type ResolverPool struct {
	workers   int
	queueSize int
	queue     chan *resolutionTask
	wg        sync.WaitGroup
	shutdown  chan struct{}
}

type resolutionTask struct {
	ctx         context.Context
	ref         *PackageReference
	constraints *DependencyConstraints
	result      chan *ResolutionResult
	error       chan error
}

func NewResolverPool(workers, queueSize int) *ResolverPool {
	pool := &ResolverPool{
		workers:   workers,
		queueSize: queueSize,
		queue:     make(chan *resolutionTask, queueSize),
		shutdown:  make(chan struct{}),
	}

	// Start workers
	for i := 0; i < workers; i++ {
		pool.wg.Add(1)
		go pool.worker()
	}

	return pool
}

func (rp *ResolverPool) worker() {
	defer rp.wg.Done()
	for {
		select {
		case task := <-rp.queue:
			// Process resolution task
			// Implementation would handle the actual resolution
		case <-rp.shutdown:
			return
		}
	}
}

func (rp *ResolverPool) Close() {
	close(rp.shutdown)
	rp.wg.Wait()
	close(rp.queue)
}

// Dependency Provider interface

type DependencyProvider interface {
	GetDependencies(ctx context.Context, ref *PackageReference) ([]*PackageReference, error)
	GetVersions(ctx context.Context, ref *PackageReference) ([]string, error)
	GetPackageInfo(ctx context.Context, ref *PackageReference, version string) (*PackageInfo, error)
}

// Resolution Strategy interface

type ResolutionStrategy interface {
	Resolve(ctx context.Context, graph *DependencyGraph, constraints *DependencyConstraints) (*ResolutionResult, error)
	GetName() string
	GetPriority() int
}

// Additional supporting types

type GraphStatistics struct {
	NodeCount            int
	EdgeCount            int
	MaxDepth             int
	AverageOutDegree     float64
	AverageInDegree      float64
	ConnectedComponents  int
	LargestComponentSize int
}

type CircularBreakingSuggestion struct {
	Cycle       *DependencyCycle
	BreakPoint  *DependencyEdge
	Alternative *PackageReference
	Impact      string
	Effort      EffortLevel
}

type CircularImpactAnalysis struct {
	AffectedPackages  int
	DeploymentImpact  string
	RiskLevel         RiskLevel
	MitigationOptions []string
}

type CycleBreakingOption struct {
	EdgeToRemove *DependencyEdge
	Impact       string
	Feasibility  float64
}

type CycleImpact struct {
	BlockedPackages  int
	DeploymentDelay  time.Duration
	RollbackRequired bool
}

type PropagationPlan struct {
	Steps            []*PropagationStep
	EstimatedTime    time.Duration
	AffectedPackages int
	RiskAssessment   *RiskAssessment
}

type PropagationStep struct {
	Package    *PackageReference
	Action     string
	Dependents []*PackageReference
	Priority   int
}

type PropagationImpact struct {
	DirectImpact    int
	IndirectImpact  int
	BreakingChanges []*BreakingChange
	RiskLevel       RiskLevel
}

type BreakingChange struct {
	Package     *PackageReference
	ChangeType  string
	Description string
	Mitigation  string
}

type UpdateImpact struct {
	Compatibility     bool
	BreakingChanges   []*BreakingChange
	PerformanceImpact string
	SecurityImpact    string
}

type UpdatePrerequisite struct {
	Package   *PackageReference
	Version   string
	Reason    string
	Mandatory bool
}

type UpdateRollback struct {
	Steps        []*RollbackStep
	DataBackup   bool
	TimeEstimate time.Duration
}

type RollbackStep struct {
	Action      string
	Package     *PackageReference
	OldVersion  string
	Description string
}

type ConflictResolutionOption struct {
	Strategy    string
	Description string
	Steps       []string
	RiskLevel   RiskLevel
	Success     float64
}

type ConflictImpact struct {
	BlockedPackages  int
	DelayedPackages  int
	DeploymentImpact string
	RiskLevel        RiskLevel
}

type ExcludedDependency struct {
	PackageRef  *PackageReference
	Reason      string
	Impact      string
	Alternative *PackageReference
}

type SubstitutionCondition struct {
	Type        string
	Expression  string
	Description string
}

type OutdatedDependency struct {
	PackageRef     *PackageReference
	CurrentVersion string
	LatestVersion  string
	VersionsBehind int
	SecurityIssues []string
}

type VulnerableDependency struct {
	PackageRef      *PackageReference
	Version         string
	Vulnerabilities []Vulnerability
	RiskScore       float64
}

type Vulnerability struct {
	CVE         string
	Severity    string
	Description string
	FixVersion  string
}

type ConflictingDependency struct {
	PackageRef *PackageReference
	Conflicts  []*DependencyConflict
	Impact     string
}

type UnusedDependency struct {
	PackageRef   *PackageReference
	LastUsed     time.Time
	Reason       string
	CanBeRemoved bool
}

type HealthRecommendation struct {
	Type        string
	Priority    int
	Description string
	Action      string
	Benefit     string
}

type ConflictedPackage struct {
	PackageRef    *PackageReference
	ConflictCount int
	ConflictTypes []ConflictType
}

type DependencyTree struct {
	Root      *TreeNode
	Depth     int
	NodeCount int
}

type TreeNode struct {
	PackageRef *PackageReference
	Version    string
	Children   []*TreeNode
	Level      int
	Optional   bool
}

type Dependent struct {
	PackageRef     *PackageReference
	DependencyType DependencyType
	Direct         bool
	Distance       int
}

type DependencyPath struct {
	From     *PackageReference
	To       *PackageReference
	Path     []*PackageReference
	Distance int
	Weight   int
}

type CompatibilityResult struct {
	Compatible  bool
	Reason      string
	Conflicts   []string
	Suggestions []string
}

type VersionHistory struct {
	PackageRef *PackageReference
	Versions   []*VersionInfo
}

type VersionInfo struct {
	Version     string
	ReleaseDate time.Time
	Deprecated  bool
	Supported   bool
	Changelog   string
}

type VersionDistance struct {
	Major int
	Minor int
	Patch int
	Total int
}

type TimeRange struct {
	Start time.Time
	End   time.Time
}

type ResolutionStatistics struct {
	TotalResolutions int
	SuccessRate      float64
	AverageTime      time.Duration
	ConflictRate     float64
}

type ResolverHealth struct {
	Status      string
	ActiveTasks int
	QueuedTasks int
	CacheSize   int
	ErrorRate   float64
	LastError   string
	Uptime      time.Duration
}

type CacheCleanupResult struct {
	EntriesRemoved int
	SpaceFreed     int64
	Duration       time.Duration
}

type SolutionStatistics struct {
	Variables      int
	Clauses        int
	Decisions      int
	Conflicts      int
	Propagations   int
	Backtracks     int
	LearnedClauses int
	SolveTime      time.Duration
}

type PackageInfo struct {
	Name         string
	Version      string
	Dependencies []*PackageReference
	Description  string
	Author       string
	License      string
	Repository   string
}

type RiskAssessment struct {
	OverallRisk    RiskLevel
	RiskFactors    []string
	Mitigations    []string
	Recommendation string
}

type MaintenanceWindow struct {
	Start     time.Time
	End       time.Time
	Recurring bool
	Timezone  string
}

type DeploymentContext struct {
	TargetClusters []*WorkloadCluster
	Environment    string
	TelcoProfile   *TelcoProfile
	ResourceLimits *ResourceLimits
	Policies       []string
}

type ResourceLimits struct {
	MaxCPU     int64
	MaxMemory  int64
	MaxStorage int64
	MaxNodes   int
}

type WorkloadCluster struct {
	Name       string
	Region     string
	Type       ClusterType
	Capacities *ClusterCapabilities
}

type ContextSelectionOptions struct {
	OptimizeForLatency bool
	OptimizeForCost    bool
	PreferEdge         bool
	RequireHA          bool
}

type AffinityRules struct {
	NodeAffinity string
	PodAffinity  string
	AntiAffinity string
}

type DependencyPolicyEngine struct {
	policies []DependencyPolicy
	logger   logr.Logger
}

type DependencyPolicy struct {
	Name        string
	Description string
	Rules       []PolicyRule
	Priority    int
}

type PolicyRule struct {
	Condition  string
	Action     string
	Parameters map[string]interface{}
}

func NewDependencyPolicyEngine() *DependencyPolicyEngine {
	return &DependencyPolicyEngine{
		policies: []DependencyPolicy{},
		logger:   log.Log.WithName("policy-engine"),
	}
}

type ContextMetricsCollector struct {
	metrics map[string]interface{}
	mu      sync.RWMutex
}

func NewContextMetricsCollector() *ContextMetricsCollector {
	return &ContextMetricsCollector{
		metrics: make(map[string]interface{}),
	}
}

type GraphVisualizer struct {
	logger logr.Logger
}

func NewGraphVisualizer() *GraphVisualizer {
	return &GraphVisualizer{
		logger: log.Log.WithName("graph-visualizer"),
	}
}

type GraphMetricsCollector struct {
	metrics map[string]interface{}
	mu      sync.RWMutex
}

func NewGraphMetricsCollector() *GraphMetricsCollector {
	return &GraphMetricsCollector{
		metrics: make(map[string]interface{}),
	}
}

type GraphAnalysisCache struct {
	cache map[string]*GraphAnalysisResult
	mu    sync.RWMutex
}

func NewGraphAnalysisCache() *GraphAnalysisCache {
	return &GraphAnalysisCache{
		cache: make(map[string]*GraphAnalysisResult),
	}
}

func (gac *GraphAnalysisCache) Get(graphID string) *GraphAnalysisResult {
	gac.mu.RLock()
	defer gac.mu.RUnlock()
	return gac.cache[graphID]
}

func (gac *GraphAnalysisCache) Set(graphID string, result *GraphAnalysisResult) {
	gac.mu.Lock()
	defer gac.mu.Unlock()
	gac.cache[graphID] = result
}

// Enums

type ClusterType string

const (
	ClusterTypeCloud ClusterType = "cloud"
	ClusterTypeEdge  ClusterType = "edge"
	ClusterTypeCore  ClusterType = "core"
)

type ComplianceLevel string

const (
	ComplianceLevelNone     ComplianceLevel = "none"
	ComplianceLevelBasic    ComplianceLevel = "basic"
	ComplianceLevelStandard ComplianceLevel = "standard"
	ComplianceLevelStrict   ComplianceLevel = "strict"
)

type LatencyClass string

const (
	LatencyClassUltraLow LatencyClass = "ultra-low"
	LatencyClassLow      LatencyClass = "low"
	LatencyClassMedium   LatencyClass = "medium"
	LatencyClassHigh     LatencyClass = "high"
)

type BandwidthClass string

const (
	BandwidthClassUltraHigh BandwidthClass = "ultra-high"
	BandwidthClassHigh      BandwidthClass = "high"
	BandwidthClassMedium    BandwidthClass = "medium"
	BandwidthClassLow       BandwidthClass = "low"
)

type TelcoDeploymentType string

const (
	TelcoDeploymentType5GCore TelcoDeploymentType = "5g-core"
	TelcoDeploymentTypeORAN   TelcoDeploymentType = "o-ran"
	TelcoDeploymentTypeEdge   TelcoDeploymentType = "edge"
	TelcoDeploymentTypeSlice  TelcoDeploymentType = "slice"
)

type RANType string

const (
	RANTypeORAN        RANType = "o-ran"
	RANTypeVRAN        RANType = "vran"
	RANTypeTraditional RANType = "traditional"
)

type CoreType string

const (
	CoreType5GSA  CoreType = "5g-sa"
	CoreType5GNSA CoreType = "5g-nsa"
	CoreTypeEPC   CoreType = "epc"
)

type SliceType string

const (
	SliceTypeEMBB  SliceType = "embb"
	SliceTypeURLLC SliceType = "urllc"
	SliceTypeMMTC  SliceType = "mmtc"
)

type RegulatoryRequirements struct {
	DataLocality        bool
	Encryption          string
	ComplianceStandards []string
}

type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

type UpdateType string

const (
	UpdateTypeMajor UpdateType = "major"
	UpdateTypeMinor UpdateType = "minor"
	UpdateTypePatch UpdateType = "patch"
)

type UpdateReason string

const (
	UpdateReasonDependency UpdateReason = "dependency"
	UpdateReasonSecurity   UpdateReason = "security"
	UpdateReasonBugFix     UpdateReason = "bugfix"
	UpdateReasonFeature    UpdateReason = "feature"
)

type SelectionReason string

const (
	SelectionReasonConstraintSatisfaction SelectionReason = "constraint_satisfaction"
	SelectionReasonOptimization           SelectionReason = "optimization"
	SelectionReasonPolicy                 SelectionReason = "policy"
	SelectionReasonDefault                SelectionReason = "default"
)

type RequirementPriority string

const (
	RequirementPriorityMandatory RequirementPriority = "mandatory"
	RequirementPriorityHigh      RequirementPriority = "high"
	RequirementPriorityMedium    RequirementPriority = "medium"
	RequirementPriorityLow       RequirementPriority = "low"
)

type RequirementScope string

const (
	RequirementScopeGlobal  RequirementScope = "global"
	RequirementScopeCluster RequirementScope = "cluster"
	RequirementScopeLocal   RequirementScope = "local"
)

type VersionSelectionStrategy string

const (
	VersionSelectionStrategyLatest     VersionSelectionStrategy = "latest"
	VersionSelectionStrategyStable     VersionSelectionStrategy = "stable"
	VersionSelectionStrategyCompatible VersionSelectionStrategy = "compatible"
	VersionSelectionStrategyNone       VersionSelectionStrategy = "none"
)

type CircularResolutionStrategy string

const (
	CircularResolutionStrategyBreak    CircularResolutionStrategy = "break"
	CircularResolutionStrategyRefactor CircularResolutionStrategy = "refactor"
	CircularResolutionStrategyIgnore   CircularResolutionStrategy = "ignore"
)

type PropagationStrategy string

const (
	PropagationStrategyEager     PropagationStrategy = "eager"
	PropagationStrategyLazy      PropagationStrategy = "lazy"
	PropagationStrategySelective PropagationStrategy = "selective"
)

type StepType string

const (
	StepTypeInstall   StepType = "install"
	StepTypeUpdate    StepType = "update"
	StepTypeConfigure StepType = "configure"
	StepTypeValidate  StepType = "validate"
)

type StepAction string

const (
	StepActionDeploy   StepAction = "deploy"
	StepActionUpgrade  StepAction = "upgrade"
	StepActionRollback StepAction = "rollback"
	StepActionRemove   StepAction = "remove"
)

type ResolutionWarning struct {
	Code       string
	Message    string
	Severity   string
	PackageRef *PackageReference
	Suggestion string
}

type Prerequisite struct {
	Condition   string
	Description string
	Mandatory   bool
}

type ValidationRule struct {
	Name       string
	Expression string
	Message    string
	Severity   string
}

type RollbackPlan struct {
	Steps        []*RollbackStep
	Checkpoints  []string
	TimeEstimate time.Duration
}

type ScopeConstraint struct {
	Scope   DependencyScope
	Allowed bool
	Reason  string
}

type SecurityConstraint struct {
	MinSecurityLevel string
	CVEExclusions    []string
	LicenseTypes     []string
}

type PolicyConstraint struct {
	PolicyName  string
	Enforcement string
	Parameters  map[string]interface{}
}

type ExclusionRule struct {
	Pattern    string
	Reason     string
	Exceptions []string
}

type InclusionRule struct {
	Pattern   string
	Reason    string
	Mandatory bool
}

type VersionConflict struct {
	PackageRef        *PackageReference
	RequestedVersions []string
	ConflictReason    string
	Resolution        string
}

type TreeSortOption string

const (
	TreeSortOptionAlphabetical TreeSortOption = "alphabetical"
	TreeSortOptionDependency   TreeSortOption = "dependency"
	TreeSortOptionPriority     TreeSortOption = "priority"
)

type CompatibilityMatrix struct {
	Packages      []*PackageReference
	Compatibility map[string]map[string]bool
	Conflicts     map[string]map[string]string
}

type ResourceProfile struct {
	CPURequests     int64
	MemoryRequests  int64
	CPULimits       int64
	MemoryLimits    int64
	StorageRequests int64
}

type SecurityProfile struct {
	Encryption      bool
	MTLS            bool
	NetworkPolicies bool
	PodSecurity     string
	SELinux         bool
}
