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

// Basic types that are missing from other files

// UsageData represents usage data for packages.
type UsageData struct {
	Package   *PackageReference `json:"package"`
	Usage     int64             `json:"usage"`
	Timestamp time.Time         `json:"timestamp"`
}

// Additional basic types needed for compilation that are NOT already defined elsewhere
type GraphNode struct {
	PackageRef *PackageReference `json:"packageRef"`
}
type GraphMetrics struct{}
type DependencyGraph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}
type CostTrend string
type HealthGrade string
type OptimizationStrategy string
type HealthTrend string
type PackageRegistryConfig struct{}
type UpdateStrategy string
type UpdateType string
type GraphEdge struct{}
type RolloutStatus string
type UpdateStep struct{}
type DependencyUpdate struct{}

// TimeRange represents a time range for analysis
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ApprovalDecision represents a decision made during approval process
type ApprovalDecision struct {
	Approved bool      `json:"approved"`
	Reason   string    `json:"reason,omitempty"`
	Time     time.Time `json:"time"`
}

// VersionPin represents a version constraint
type VersionPin struct {
	Version string `json:"version"`
	Locked  bool   `json:"locked"`
}

// SecurityPolicy represents security configuration
type SecurityPolicy struct {
	Enabled bool     `json:"enabled"`
	Rules   []string `json:"rules,omitempty"`
}

// LicensePolicy represents license requirements
type LicensePolicy struct {
	AllowedLicenses []string `json:"allowedLicenses"`
	Restricted      []string `json:"restricted,omitempty"`
}

// CriticalPath represents a critical dependency path
type CriticalPath []*GraphNode

// Additional missing types needed for compilation
type DependencyCycle []*GraphNode
type GraphPattern struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type DependencyNode struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type UpdateResult struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

type PropagationResult struct {
	Affected int    `json:"affected"`
	Success  bool   `json:"success"`
	Message  string `json:"message,omitempty"`
}

type SecurityPolicies struct {
	Enabled bool     `json:"enabled"`
	Rules   []string `json:"rules,omitempty"`
}

type ComplianceRules struct {
	Required []string `json:"required,omitempty"`
	Excluded []string `json:"excluded,omitempty"`
}

// Additional validator-related types
type PlatformConstraints struct {
	OS           string   `json:"os,omitempty"`
	Architecture string   `json:"architecture,omitempty"`
	Versions     []string `json:"versions,omitempty"`
}

type ResourceLimits struct {
	MaxMemory string `json:"maxMemory,omitempty"`
	MaxCPU    string `json:"maxCPU,omitempty"`
}

type OrganizationalPolicies struct {
	AllowedSources []string `json:"allowedSources,omitempty"`
	BlockedSources []string `json:"blockedSources,omitempty"`
}

type VersionValidation struct {
	Required bool   `json:"required"`
	Pattern  string `json:"pattern,omitempty"`
}

type PackageSecurityValidation struct {
	Enabled      bool     `json:"enabled"`
	CheckSources []string `json:"checkSources,omitempty"`
}

type PackageLicenseValidation struct {
	Enabled         bool     `json:"enabled"`
	AllowedLicenses []string `json:"allowedLicenses,omitempty"`
}

type QualityValidation struct {
	Enabled         bool    `json:"enabled"`
	MinQualityScore float64 `json:"minQualityScore,omitempty"`
}

type ErrorSeverity string

const (
	ErrorSeverityLow      ErrorSeverity = "low"
	ErrorSeverityMedium   ErrorSeverity = "medium"
	ErrorSeverityHigh     ErrorSeverity = "high"
	ErrorSeverityCritical ErrorSeverity = "critical"
)

type WarningType string

const (
	WarningTypeDeprecated WarningType = "deprecated"
	WarningTypeOutdated   WarningType = "outdated"
	WarningTypeSecurity   WarningType = "security"
)

type WarningImpact string

const (
	WarningImpactLow    WarningImpact = "low"
	WarningImpactMedium WarningImpact = "medium"
	WarningImpactHigh   WarningImpact = "high"
)

// Additional validation types
type CompatibilityResult struct {
	Compatible bool     `json:"compatible"`
	Issues     []string `json:"issues,omitempty"`
	Score      float64  `json:"score"`
}

type PlatformValidation struct {
	Passed   bool     `json:"passed"`
	Warnings []string `json:"warnings,omitempty"`
	Errors   []string `json:"errors,omitempty"`
}

type LicenseValidation struct {
	Valid    bool     `json:"valid"`
	License  string   `json:"license,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

type ComplianceValidation struct {
	Compliant bool     `json:"compliant"`
	Violations []string `json:"violations,omitempty"`
}

type PerformanceValidation struct {
	Acceptable bool    `json:"acceptable"`
	Score      float64 `json:"score"`
	Metrics    map[string]interface{} `json:"metrics,omitempty"`
}

type PolicyValidation struct {
	Valid     bool     `json:"valid"`
	Violations []string `json:"violations,omitempty"`
}

type ValidationRecommendation struct {
	Action      string `json:"action"`
	Reason      string `json:"reason"`
	Severity    string `json:"severity"`
	Description string `json:"description,omitempty"`
}

type ValidationStatistics struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Warnings int `json:"warnings"`
}

type ComplianceStatus struct {
	Status     string    `json:"status"`
	LastCheck  time.Time `json:"lastCheck"`
	Score      float64   `json:"score"`
	Violations []string  `json:"violations,omitempty"`
}
