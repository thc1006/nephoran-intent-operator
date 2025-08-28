// Package dependencies provides advanced dependency graph analysis for O-RAN and Nephio packages
package dependencies

import (
	"time"
)

// DependencyGraph represents a graph of package dependencies
type DependencyGraph interface {
	// Core graph operations
	AddNode(node *DependencyNode) error
	RemoveNode(nodeID string) error
	AddEdge(from, to string, edgeType EdgeType) error
	RemoveEdge(from, to string) error

	// Graph queries
	GetNode(nodeID string) (*DependencyNode, error)
	GetNodes() []*DependencyNode
	GetEdges() []*DependencyEdge
	GetNeighbors(nodeID string) ([]*DependencyNode, error)
	GetDependencies(nodeID string) ([]*DependencyNode, error)
	GetDependents(nodeID string) ([]*DependencyNode, error)

	// Graph analysis
	FindCycles() ([]*Cycle, error)
	GetTopologicalSort() ([]string, error)
	ComputeCentrality() (map[string]*CentralityMetrics, error)
	FindCriticalPaths() ([]*CriticalPath, error)
	DetectPatterns() ([]*GraphPattern, error)

	// Graph modification and optimization
	ResolveCycles(options *CycleBreakingOption) (*DependencyGraph, error)
	OptimizeGraph(constraints *OptimizationConstraints) (*DependencyGraph, error)
	MergeGraphs(other DependencyGraph) error
	SubGraph(nodeIDs []string) (DependencyGraph, error)

	// Serialization and persistence
	Serialize() ([]byte, error)
	Deserialize(data []byte) error
	Save(path string) error
	Load(path string) error

	// Statistics and metrics
	GetStatistics() *GraphStatistics
	ValidateIntegrity() error
	Clone() DependencyGraph

	// Advanced analysis methods
	ComputeMetrics() (*GraphMetrics, error)
	AnalyzeImpact(changes []*GraphUpdate) (*ImpactAnalysis, error)
	TraverseGraph(visitor GraphVisitor) error
	FindPaths(from, to string, maxDepth int) ([]*GraphPath, error)
	ComputeDistribution() (*DistributionStats, error)
	EstimateComplexity() (*ComplexityMetrics, error)

	// Machine learning and prediction
	PredictDependencies(context *PredictionContext) ([]*DependencyPrediction, error)
	TrainModel(data *TrainingData) error
	GetRecommendations(nodeID string) ([]*DependencyRecommendation, error)

	// Versioning and evolution
	GetVersionHistory() ([]*GraphVersion, error)
	ApplyChangeset(changeset *GraphChangeset) error
	GetDiff(other DependencyGraph) (*GraphDiff, error)
	Rollback(versionID string) error

	// Performance and optimization
	EnableCaching(config *CacheConfig) error
	GetCacheStatistics() *CacheStatistics
	CompressGraph() error
	DecompressGraph() error

	// Monitoring and observability
	GetHealthScore() (*HealthScore, error)
	GetPerformanceMetrics() (*PerformanceMetrics, error)
	SetMetricsCollector(collector MetricsCollector) error

	// Concurrent operations
	Lock() error
	Unlock() error
	IsLocked() bool

	// Export and import
	ExportToDOT() (string, error)
	ExportToJSON() ([]byte, error)
	ExportToGraphML() ([]byte, error)
	ImportFromDOT(dot string) error

	// Validation and verification
	ValidateGraph() (*ValidationResult, error)
	CheckConsistency() error
	VerifyConstraints(constraints *GraphConstraints) (*ConstraintViolation, error)
}

// DependencyNode represents a node in the dependency graph
type DependencyNode struct {
	// Basic identification
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Type        NodeType  `json:"type"`
	Status      NodeStatus `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`

	// Package information
	PackageInfo *PackageInfo `json:"packageInfo,omitempty"`
	Repository  *Repository  `json:"repository,omitempty"`
	License     *License     `json:"license,omitempty"`

	// Dependency metadata
	Dependencies   []string                 `json:"dependencies"`
	DevDependencies []string                `json:"devDependencies,omitempty"`
	OptionalDeps   []string                 `json:"optionalDependencies,omitempty"`
	PeerDeps       []string                 `json:"peerDependencies,omitempty"`
	BundledDeps    []string                 `json:"bundledDependencies,omitempty"`
	Metadata       map[string]interface{}   `json:"metadata,omitempty"`
	Tags           []string                 `json:"tags,omitempty"`
	Labels         map[string]string        `json:"labels,omitempty"`

	// Analysis data
	Centrality      *CentralityMetrics `json:"centrality,omitempty"`
	Metrics         *NodeMetrics       `json:"metrics,omitempty"`
	SecurityInfo    *SecurityInfo      `json:"securityInfo,omitempty"`
	PerformanceInfo *PerformanceInfo   `json:"performanceInfo,omitempty"`

	// Graph position
	Depth           int     `json:"depth"`
	Distance        int     `json:"distance,omitempty"`
	PathCount       int     `json:"pathCount,omitempty"`
	Weight          float64 `json:"weight,omitempty"`

	// State and lifecycle
	IsRoot          bool   `json:"isRoot"`
	IsLeaf          bool   `json:"isLeaf"`
	IsCritical      bool   `json:"isCritical"`
	IsDeprecated    bool   `json:"isDeprecated"`
	IsOptional      bool   `json:"isOptional"`
	DeprecationInfo *DeprecationInfo `json:"deprecationInfo,omitempty"`

	// Caching and performance
	CacheKey        string    `json:"cacheKey,omitempty"`
	LastAccessed    time.Time `json:"lastAccessed,omitempty"`
	AccessCount     int64     `json:"accessCount,omitempty"`

	// Validation and integrity
	Checksum        string `json:"checksum,omitempty"`
	Signature       string `json:"signature,omitempty"`
	ValidatedAt     time.Time `json:"validatedAt,omitempty"`
}

// GraphStatistics contains statistical information about the graph
type GraphStatistics struct {
	// Basic counts
	NodeCount        int     `json:"nodeCount"`
	EdgeCount        int     `json:"edgeCount"`
	CycleCount       int     `json:"cycleCount"`
	ComponentCount   int     `json:"componentCount"`
	
	// Graph properties
	Density          float64 `json:"density"`
	Diameter         int     `json:"diameter"`
	AverageDistance  float64 `json:"averageDistance"`
	ClusteringCoeff  float64 `json:"clusteringCoefficient"`
	
	// Connectivity
	IsConnected      bool    `json:"isConnected"`
	IsAcyclic        bool    `json:"isAcyclic"`
	HasSelfLoops     bool    `json:"hasSelfLoops"`
	HasMultipleEdges bool    `json:"hasMultipleEdges"`
	
	// Distribution metrics
	DegreeDistribution map[int]int `json:"degreeDistribution"`
	InDegreeStats      *StatsSummary `json:"inDegreeStats"`
	OutDegreeStats     *StatsSummary `json:"outDegreeStats"`
	
	// Performance metrics
	ComputeTime      time.Duration `json:"computeTime"`
	MemoryUsage      int64         `json:"memoryUsage"`
	LastUpdated      time.Time     `json:"lastUpdated"`
}

// GraphBuildOptions contains options for building dependency graphs
type GraphBuildOptions struct {
	// Source configuration
	RootPackages      []*PackageReference `json:"rootPackages"`
	MaxDepth          int                 `json:"maxDepth,omitempty"`
	IncludeOptional   bool                `json:"includeOptional,omitempty"`
	IncludeTest       bool                `json:"includeTest,omitempty"`
	IncludeTransitive bool                `json:"includeTransitive,omitempty"`

	// Filtering and constraints
	VersionStrategy VersionStrategy   `json:"versionStrategy,omitempty"`
	ScopeFilter     []DependencyScope `json:"scopeFilter,omitempty"`
	PackageFilter   *PackageFilter    `json:"packageFilter,omitempty"`

	// Analysis options
	ComputeMetrics    bool `json:"computeMetrics,omitempty"`
	DetectCycles      bool `json:"detectCycles,omitempty"`
	ComputeCentrality bool `json:"computeCentrality,omitempty"`

	// Performance options
	UseParallel  bool `json:"useParallel,omitempty"`
	CacheResults bool `json:"cacheResults,omitempty"`

	// Metadata
	Name     string                 `json:"name,omitempty"`
	Tags     map[string]string      `json:"tags,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NodeType defines the type of node in the graph
type NodeType string

const (
	NodeTypeRoot         NodeType = "root"
	NodeTypeLeaf         NodeType = "leaf"
	NodeTypeIntermediate NodeType = "intermediate"
	NodeTypeOrphan       NodeType = "orphan"
)

// NodeStatus defines the status of a node
type NodeStatus string

const (
	NodeStatusResolved   NodeStatus = "resolved"
	NodeStatusUnresolved NodeStatus = "unresolved"
	NodeStatusConflicted NodeStatus = "conflicted"
	NodeStatusMissing    NodeStatus = "missing"
	NodeStatusDeprecated NodeStatus = "deprecated"
)

// Missing type definitions that were causing compilation errors

// CriticalPath represents a critical path in the dependency graph
type CriticalPath struct {
	ID          string   `json:"id"`
	Nodes       []string `json:"nodes"`
	Edges       []string `json:"edges"`
	Length      int      `json:"length"`
	TotalWeight float64  `json:"totalWeight"`
	MaxDelay    float64  `json:"maxDelay,omitempty"`
	Priority    int      `json:"priority,omitempty"`
	RiskLevel   string   `json:"riskLevel,omitempty"`
	Description string   `json:"description,omitempty"`
	Impact      string   `json:"impact,omitempty"`
}

// GraphPattern represents a detected pattern in the dependency graph
type GraphPattern struct {
	PatternID   string `json:"patternId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        GraphPatternType `json:"type"`
	Occurrences []*PatternOccurrence `json:"occurrences"`
	Impact      GraphImpact `json:"impact"`
	Suggestions []string `json:"suggestions,omitempty"`
	Confidence  float64  `json:"confidence,omitempty"`
	Severity    string   `json:"severity,omitempty"`
}

// CycleBreakingOption represents options for breaking dependency cycles
type CycleBreakingOption struct {
	OptionID    string   `json:"optionId"`
	Description string   `json:"description"`
	EdgesToBreak []string `json:"edgesToBreak"`
	CostEstimate float64  `json:"costEstimate"`
	RiskLevel   string    `json:"riskLevel"`
	Feasibility float64  `json:"feasibility"`
	Impact      *CycleImpact `json:"impact,omitempty"`
	Alternative []string `json:"alternative,omitempty"`
}

// GraphUpdate represents an update to the dependency graph
type GraphUpdate struct {
	UpdateID    string    `json:"updateId"`
	Type        string    `json:"type"` // "add_node", "remove_node", "add_edge", "remove_edge", "update_node", "update_edge"
	Timestamp   time.Time `json:"timestamp"`
	NodeID      string    `json:"nodeId,omitempty"`
	EdgeID      string    `json:"edgeId,omitempty"`
	FromNodeID  string    `json:"fromNodeId,omitempty"`
	ToNodeID    string    `json:"toNodeId,omitempty"`
	OldValue    interface{} `json:"oldValue,omitempty"`
	NewValue    interface{} `json:"newValue,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// GraphVisitor interface for graph traversal
type GraphVisitor interface {
	VisitNode(node *DependencyNode) error
	VisitEdge(edge *DependencyEdge) error
	EnterSubgraph(subgraphName string) error
	ExitSubgraph(subgraphName string) error
}

// GraphPath represents a path between nodes in the graph
type GraphPath struct {
	PathID      string   `json:"pathId"`
	StartNode   string   `json:"startNode"`
	EndNode     string   `json:"endNode"`
	Nodes       []string `json:"nodes"`
	Edges       []string `json:"edges"`
	Length      int      `json:"length"`
	TotalWeight float64  `json:"totalWeight"`
	PathType    string   `json:"pathType"` // "shortest", "longest", "critical", "alternative"
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// CycleImpact represents the impact of breaking a cycle
type CycleImpact struct {
	AffectedNodes    []string `json:"affectedNodes"`
	AffectedEdges    []string `json:"affectedEdges"`
	PerformanceGain  float64  `json:"performanceGain"`
	StabilityRisk    float64  `json:"stabilityRisk"`
	MaintenanceCost  float64  `json:"maintenanceCost"`
	BusinessImpact   string   `json:"businessImpact,omitempty"`
	TechnicalImpact  string   `json:"technicalImpact,omitempty"`
	AlternativePaths []string `json:"alternativePaths,omitempty"`
}

// DistributionStats represents statistical distribution of graph properties
type DistributionStats struct {
	NodeDegreeDistribution map[int]int `json:"nodeDegreeDistribution"`
	EdgeWeightDistribution map[string]int `json:"edgeWeightDistribution"`
	DepthDistribution      map[int]int `json:"depthDistribution"`
	ComponentSizeDistribution map[int]int `json:"componentSizeDistribution"`
	PathLengthDistribution map[int]int `json:"pathLengthDistribution"`
	CentralityDistribution map[string]float64 `json:"centralityDistribution"`
	Mean          float64 `json:"mean"`
	Median        float64 `json:"median"`
	StandardDev   float64 `json:"standardDeviation"`
	Skewness      float64 `json:"skewness,omitempty"`
	Kurtosis      float64 `json:"kurtosis,omitempty"`
}

// VersionStrategy defines the strategy for version resolution
type VersionStrategy struct {
	Strategy    string `json:"strategy"` // "latest", "stable", "exact", "range", "semantic"
	Preference  string `json:"preference,omitempty"` // "major", "minor", "patch"
	AllowPrereleases bool `json:"allowPrereleases,omitempty"`
	AllowDowngrades  bool `json:"allowDowngrades,omitempty"`
	ConflictResolution string `json:"conflictResolution,omitempty"` // "newest", "oldest", "manual"
	Constraints []string `json:"constraints,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// PackageFilter defines filtering criteria for packages
type PackageFilter struct {
	IncludePatterns []string `json:"includePatterns,omitempty"`
	ExcludePatterns []string `json:"excludePatterns,omitempty"`
	Scopes          []DependencyScope `json:"scopes,omitempty"`
	Types           []string `json:"types,omitempty"`
	Licenses        []string `json:"licenses,omitempty"`
	Registries      []string `json:"registries,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	MinVersion      string   `json:"minVersion,omitempty"`
	MaxVersion      string   `json:"maxVersion,omitempty"`
	SecurityLevel   string   `json:"securityLevel,omitempty"`
	Deprecated      *bool    `json:"deprecated,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// License contains license information
type License struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
	Text string `json:"text,omitempty"`
}

// CentralityMetrics contains centrality measures for a node
type CentralityMetrics struct {
	Betweenness    float64 `json:"betweenness"`
	Closeness      float64 `json:"closeness"`
	Degree         int     `json:"degree"`
	InDegree       int     `json:"inDegree"`
	OutDegree      int     `json:"outDegree"`
	Eigenvector    float64 `json:"eigenvector,omitempty"`
	PageRank       float64 `json:"pageRank,omitempty"`
	HubScore       float64 `json:"hubScore,omitempty"`
	AuthorityScore float64 `json:"authorityScore,omitempty"`
}

// NodeMetrics contains metrics for a node
type NodeMetrics struct {
	Size           int64     `json:"size,omitempty"`
	Complexity     float64   `json:"complexity,omitempty"`
	LastUsed       time.Time `json:"lastUsed,omitempty"`
	UsageCount     int64     `json:"usageCount,omitempty"`
	BuildTime      time.Duration `json:"buildTime,omitempty"`
	TestCoverage   float64   `json:"testCoverage,omitempty"`
	Maintainability float64  `json:"maintainability,omitempty"`
}



// DeprecationInfo contains deprecation information
type DeprecationInfo struct {
	Since       string    `json:"since"`
	Until       string    `json:"until,omitempty"`
	Reason      string    `json:"reason,omitempty"`
	Alternative string    `json:"alternative,omitempty"`
	Migration   string    `json:"migration,omitempty"`
	Severity    string    `json:"severity,omitempty"`
	AnnouncedAt time.Time `json:"announcedAt,omitempty"`
}

// StatsSummary contains statistical summary
type StatsSummary struct {
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Mean   float64 `json:"mean"`
	Median float64 `json:"median"`
	StdDev float64 `json:"stddev"`
	Count  int     `json:"count"`
}

// GraphPatternType defines types of graph patterns
type GraphPatternType string

const (
	GraphPatternAntiPattern  GraphPatternType = "anti_pattern"
	GraphPatternBestPractice GraphPatternType = "best_practice"
	GraphPatternCircularDep  GraphPatternType = "circular_dependency"
	GraphPatternDuplication  GraphPatternType = "duplication"
	GraphPatternUnused       GraphPatternType = "unused"
)

// PatternOccurrence represents an occurrence of a pattern
type PatternOccurrence struct {
	Location    []string `json:"location"`
	Severity    string   `json:"severity"`
	Context     string   `json:"context,omitempty"`
	Suggestion  string   `json:"suggestion,omitempty"`
}

// GraphImpact represents the impact of a pattern or change
type GraphImpact struct {
	Performance  float64 `json:"performance"`
	Maintainability float64 `json:"maintainability"`
	Security     float64 `json:"security"`
	Stability    float64 `json:"stability"`
	Overall      float64 `json:"overall"`
	Description  string  `json:"description,omitempty"`
}

// GraphMetrics contains comprehensive graph metrics
type GraphMetrics struct {
	BasicStats      *GraphStatistics `json:"basicStats"`
	Centrality      map[string]*CentralityMetrics `json:"centrality,omitempty"`
	Performance     *PerformanceMetrics `json:"performance,omitempty"`
	Quality         *QualityMetrics `json:"quality,omitempty"`
	Security        *SecurityMetrics `json:"security,omitempty"`
	Complexity      *ComplexityMetrics `json:"complexity,omitempty"`
}

// QualityMetrics placeholder types
type QualityMetrics struct {
	OverallScore    float64 `json:"overallScore"`
	Maintainability float64 `json:"maintainability"`
	Reliability     float64 `json:"reliability"`
	Testability     float64 `json:"testability"`
}

type ComplexityMetrics struct {
	CyclomaticComplexity float64 `json:"cyclomaticComplexity"`
	CognitiveComplexity  float64 `json:"cognitiveComplexity"`
	Maintainability      float64 `json:"maintainability"`
}

type PerformanceMetrics struct {
	MemoryUsage     int64         `json:"memoryUsage"`
	CPUUsage        float64       `json:"cpuUsage"`
	ResponseTime    time.Duration `json:"responseTime"`
	Throughput      float64       `json:"throughput"`
}

// Placeholder types for interface completeness

type PredictionContext struct {
	Context string `json:"context"`
}

type DependencyPrediction struct {
	Prediction string `json:"prediction"`
}

type TrainingData struct {
	Data string `json:"data"`
}

type DependencyRecommendation struct {
	Recommendation string `json:"recommendation"`
}

type GraphVersion struct {
	Version string `json:"version"`
}

type GraphChangeset struct {
	Changes string `json:"changes"`
}

type GraphDiff struct {
	Diff string `json:"diff"`
}

type CacheConfig struct {
	Config string `json:"config"`
}

type CacheStatistics struct {
	Stats string `json:"stats"`
}

type MetricsCollector interface {
	Collect() error
}


type GraphConstraints struct {
	Constraints string `json:"constraints"`
}


type OptimizationConstraints struct {
	Constraints string `json:"constraints"`
}

type Cycle struct {
	Nodes []string `json:"nodes"`
	Edges []string `json:"edges"`
}

