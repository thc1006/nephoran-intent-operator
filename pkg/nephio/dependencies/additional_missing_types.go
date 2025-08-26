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

// Additional missing types found during compilation

// AffectedPackage represents a package affected by a vulnerability
type AffectedPackage struct {
	Package        *PackageReference `json:"package"`
	VersionRange   string            `json:"versionRange"`
	FixedInVersion string            `json:"fixedInVersion,omitempty"`
	Severity       string            `json:"severity"`
}

// ComplexityAnalysis represents analysis of dependency complexity
type ComplexityAnalysis struct {
	OverallComplexity  float64            `json:"overallComplexity"`
	CyclomaticComplexity float64          `json:"cyclomaticComplexity,omitempty"`
	DependencyDepth    int                `json:"dependencyDepth"`
	GraphComplexity    float64            `json:"graphComplexity"`
	PackageComplexities map[string]float64 `json:"packageComplexities,omitempty"`
	ComplexityTrends   []*ComplexityTrend `json:"complexityTrends,omitempty"`
}

// ComplexityTrend represents complexity trend over time
type ComplexityTrend struct {
	Timestamp  time.Time `json:"timestamp"`
	Complexity float64   `json:"complexity"`
	Change     float64   `json:"change"`
}

// CycleImpact represents the impact of a dependency cycle
type CycleImpact string

const (
	CycleImpactLow      CycleImpact = "low"
	CycleImpactMedium   CycleImpact = "medium"
	CycleImpactHigh     CycleImpact = "high"
	CycleImpactCritical CycleImpact = "critical"
)

// CycleBreakingOption represents an option for breaking dependency cycles
type CycleBreakingOption struct {
	ID          string      `json:"id"`
	Type        string      `json:"type"` // "remove", "replace", "refactor"
	Description string      `json:"description"`
	Impact      CycleImpact `json:"impact"`
	Effort      string      `json:"effort"` // "low", "medium", "high"
	Risk        string      `json:"risk"`   // "low", "medium", "high"
	Steps       []string    `json:"steps"`
	Benefits    []string    `json:"benefits,omitempty"`
}

// ResourceLimits represents resource usage limits
type ResourceLimits struct {
	MaxMemoryMB int     `json:"maxMemoryMB,omitempty"`
	MaxCPU      float64 `json:"maxCPU,omitempty"`
	MaxStorage  int     `json:"maxStorage,omitempty"`
	MaxNetwork  int     `json:"maxNetwork,omitempty"`
}

// PlatformConstraints represents platform-specific constraints
type PlatformConstraints struct {
	OperatingSystems  []string          `json:"operatingSystems,omitempty"`
	Architectures     []string          `json:"architectures,omitempty"`
	KubernetesVersion string            `json:"kubernetesVersion,omitempty"`
	RuntimeVersions   map[string]string `json:"runtimeVersions,omitempty"`
	ResourceLimits    *ResourceLimits   `json:"resourceLimits,omitempty"`
	NetworkPolicies   []string          `json:"networkPolicies,omitempty"`
}

// SecurityConstraints represents security-related constraints
type SecurityConstraints struct {
	RequiredScans      []string          `json:"requiredScans,omitempty"`
	MaxVulnerabilities int               `json:"maxVulnerabilities,omitempty"`
	AllowedLicenses    []string          `json:"allowedLicenses,omitempty"`
	BannedPackages     []string          `json:"bannedPackages,omitempty"`
	SecurityPolicies   map[string]string `json:"securityPolicies,omitempty"`
	ComplianceStandards []string         `json:"complianceStandards,omitempty"`
}

// PolicyConstraints represents policy-related constraints
type PolicyConstraints struct {
	OrganizationalPolicies []string          `json:"organizationalPolicies,omitempty"`
	LicensePolicies        []string          `json:"licensePolicies,omitempty"`
	VersionPolicies        map[string]string `json:"versionPolicies,omitempty"`
	UpdatePolicies         []string          `json:"updatePolicies,omitempty"`
	ApprovalRequired       bool              `json:"approvalRequired,omitempty"`
	ReviewRequired         bool              `json:"reviewRequired,omitempty"`
}

// TransitiveOptions represents options for transitive dependency resolution
type TransitiveOptions struct {
	IncludeTransitive   bool              `json:"includeTransitive"`
	MaxDepth           int               `json:"maxDepth,omitempty"`
	ExcludeScopes      []DependencyScope `json:"excludeScopes,omitempty"`
	FollowOptional     bool              `json:"followOptional,omitempty"`
	ResolutionStrategy string            `json:"resolutionStrategy,omitempty"`
}

// DependencyTree represents a tree structure of dependencies
type DependencyTree struct {
	Root         *PackageReference `json:"root"`
	Dependencies []*DependencyNode `json:"dependencies"`
	Depth        int               `json:"depth"`
	TotalNodes   int               `json:"totalNodes"`
	BuildTime    time.Duration     `json:"buildTime"`
}

// DependencyNode represents a node in the dependency tree
type DependencyNode struct {
	Package      *PackageReference `json:"package"`
	Dependencies []*DependencyNode `json:"dependencies,omitempty"`
	Depth        int               `json:"depth"`
	Optional     bool              `json:"optional,omitempty"`
	Scope        DependencyScope   `json:"scope,omitempty"`
}

// ResolutionWarning represents warnings during dependency resolution
type ResolutionWarning struct {
	Code        string                 `json:"code"`
	Type        string                 `json:"type"`
	Message     string                 `json:"message"`
	Package     *PackageReference      `json:"package,omitempty"`
	Severity    string                 `json:"severity"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Suggestion  string                 `json:"suggestion,omitempty"`
}

// ResolutionStatistics represents statistics about dependency resolution
type ResolutionStatistics struct {
	TotalPackages       int           `json:"totalPackages"`
	ResolvedPackages    int           `json:"resolvedPackages"`
	FailedPackages      int           `json:"failedPackages"`
	ConflictsResolved   int           `json:"conflictsResolved"`
	CacheHits           int           `json:"cacheHits"`
	ResolutionTime      time.Duration `json:"resolutionTime"`
	NetworkRequests     int           `json:"networkRequests,omitempty"`
	BytesDownloaded     int64         `json:"bytesDownloaded,omitempty"`
}

// ResolvedDependency represents a resolved dependency with metadata
type ResolvedDependency struct {
	Package           *PackageReference `json:"package"`
	ResolvedVersion   string            `json:"resolvedVersion"`
	ResolutionMethod  string            `json:"resolutionMethod"`
	Dependencies      []*PackageReference `json:"dependencies,omitempty"`
	Conflicts         []string          `json:"conflicts,omitempty"`
	ResolutionTime    time.Duration     `json:"resolutionTime"`
	ResolvedAt        time.Time         `json:"resolvedAt"`
	ResolvedBy        string            `json:"resolvedBy,omitempty"`
}

// SecurityInfo represents security information for a package
type SecurityInfo struct {
	SecurityScore     float64           `json:"securityScore"`
	VulnerabilityCount int              `json:"vulnerabilityCount"`
	Vulnerabilities   []*Vulnerability  `json:"vulnerabilities,omitempty"`
	SecurityGaps      []*SecurityGap    `json:"securityGaps,omitempty"`
	LastSecurityScan  time.Time         `json:"lastSecurityScan"`
	SecurityGrade     string            `json:"securityGrade"`
	ThreatLevel       string            `json:"threatLevel"`
}

// PerformanceInfo represents performance information for a package
type PerformanceInfo struct {
	PerformanceScore float64                `json:"performanceScore"`
	Benchmarks       map[string]float64     `json:"benchmarks,omitempty"`
	ResourceUsage    *ResourceUsageStats    `json:"resourceUsage,omitempty"`
	LoadTime         time.Duration          `json:"loadTime,omitempty"`
	PerformanceGrade string                 `json:"performanceGrade"`
}

// ResourceUsageStats represents resource usage statistics
type ResourceUsageStats struct {
	MemoryUsageMB    float64 `json:"memoryUsageMB"`
	CPUUsagePercent  float64 `json:"cpuUsagePercent"`
	DiskUsageMB      float64 `json:"diskUsageMB"`
	NetworkUsageKB   float64 `json:"networkUsageKB"`
	StartupTimeMS    float64 `json:"startupTimeMS"`
}

// ComplianceInfo represents compliance information for a package
type ComplianceInfo struct {
	ComplianceScore   float64                 `json:"complianceScore"`
	ComplianceChecks  map[string]bool         `json:"complianceChecks"`
	Violations        []*ComplianceViolation  `json:"violations,omitempty"`
	CertificationLevel string                 `json:"certificationLevel,omitempty"`
	ComplianceGrade   string                  `json:"complianceGrade"`
	LastAudit         time.Time               `json:"lastAudit"`
}

// ConflictImpact represents the impact of a dependency conflict
type ConflictImpact string

const (
	ConflictImpactLow      ConflictImpact = "low"
	ConflictImpactMedium   ConflictImpact = "medium" 
	ConflictImpactHigh     ConflictImpact = "high"
	ConflictImpactCritical ConflictImpact = "critical"
)

// ConflictResolutionStrategy represents a strategy for resolving conflicts
type ConflictResolutionStrategy struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        string   `json:"type"` // "automatic", "manual", "interactive"
	Effort      string   `json:"effort"`
	Risk        string   `json:"risk"`
	Steps       []string `json:"steps"`
	Applicable  bool     `json:"applicable"`
	Priority    int      `json:"priority"`
}