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

// Final batch of missing types from updater.go and validator.go

// PerformanceImpact represents the performance impact of an update
type PerformanceImpact struct {
	Level                string              `json:"level"` // "none", "low", "medium", "high", "critical"
	PerformanceScore     float64             `json:"performanceScore"`
	ResponseTimeImpact   float64             `json:"responseTimeImpact"` // percentage change
	ThroughputImpact     float64             `json:"throughputImpact"`   // percentage change
	ResourceUsageImpact  float64             `json:"resourceUsageImpact"` // percentage change
	MemoryUsageChange    float64             `json:"memoryUsageChange"`  // MB
	CPUUsageChange       float64             `json:"cpuUsageChange"`     // percentage
	StartupTimeImpact    float64             `json:"startupTimeImpact"`  // milliseconds
	RecommendedAction    string              `json:"recommendedAction"`
	BenchmarkResults     map[string]float64  `json:"benchmarkResults,omitempty"`
}

// CompatibilityImpact represents the compatibility impact of an update
type CompatibilityImpact struct {
	Level                    string              `json:"level"` // "none", "low", "medium", "high", "critical"
	BackwardCompatibility    bool                `json:"backwardCompatibility"`
	ForwardCompatibility     bool                `json:"forwardCompatibility"`
	APIChanges               []*APIChange        `json:"apiChanges,omitempty"`
	DeprecatedFeatures       []string            `json:"deprecatedFeatures,omitempty"`
	RemovedFeatures          []string            `json:"removedFeatures,omitempty"`
	NewFeatures              []string            `json:"newFeatures,omitempty"`
	PlatformCompatibility    map[string]bool     `json:"platformCompatibility,omitempty"`
	DependencyChanges        []*DependencyChange `json:"dependencyChanges,omitempty"`
	MigrationRequired        bool                `json:"migrationRequired"`
	MigrationComplexity      string              `json:"migrationComplexity,omitempty"`
	RecommendedAction        string              `json:"recommendedAction"`
}

// APIChange represents a change to an API
type APIChange struct {
	Type        string `json:"type"`        // "added", "modified", "deprecated", "removed"
	Element     string `json:"element"`     // "function", "method", "class", "interface", "constant"
	Name        string `json:"name"`
	Description string `json:"description"`
	Impact      string `json:"impact"`      // "breaking", "non-breaking", "behavioral"
	Mitigation  string `json:"mitigation,omitempty"`
}

// DependencyChange represents a change in dependencies
type DependencyChange struct {
	Type           string `json:"type"`    // "added", "updated", "removed"
	PackageName    string `json:"packageName"`
	OldVersion     string `json:"oldVersion,omitempty"`
	NewVersion     string `json:"newVersion,omitempty"`
	Impact         string `json:"impact"`  // "breaking", "non-breaking", "security"
	Required       bool   `json:"required"`
	Optional       bool   `json:"optional,omitempty"`
	TransitiveOnly bool   `json:"transitiveOnly,omitempty"`
}

// BusinessImpact represents the business impact of an update
type BusinessImpact struct {
	Level                 string              `json:"level"` // "none", "low", "medium", "high", "critical"
	RevenueImpact         float64             `json:"revenueImpact,omitempty"`        // estimated revenue impact
	CostImpact            float64             `json:"costImpact,omitempty"`           // estimated cost impact
	UserImpact            string              `json:"userImpact"`                     // "none", "minimal", "moderate", "significant"
	ServiceAvailability   float64             `json:"serviceAvailability,omitempty"`  // expected availability percentage
	DowntimeRequired      time.Duration       `json:"downtimeRequired,omitempty"`
	RollbackTime          time.Duration       `json:"rollbackTime,omitempty"`
	ComplianceImpact      string              `json:"complianceImpact,omitempty"`
	RiskLevel             string              `json:"riskLevel"`
	RecommendedTiming     string              `json:"recommendedTiming,omitempty"`
	StakeholderNotification bool              `json:"stakeholderNotification"`
	BusinessJustification string              `json:"businessJustification,omitempty"`
}

// BreakingChange represents a breaking change
type BreakingChange struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`        // "api", "behavior", "data", "config"
	Category    string    `json:"category"`    // "major", "minor", "patch"
	Description string    `json:"description"`
	Component   string    `json:"component,omitempty"`
	Version     string    `json:"version"`
	Severity    string    `json:"severity"`    // "low", "medium", "high", "critical"
	Impact      string    `json:"impact"`
	Mitigation  []string  `json:"mitigation,omitempty"`
	Timeline    string    `json:"timeline,omitempty"`
	DetectedAt  time.Time `json:"detectedAt"`
}

// BreakingChangeImpact represents the impact of breaking changes
type BreakingChangeImpact struct {
	TotalChanges       int                `json:"totalChanges"`
	CriticalChanges    int                `json:"criticalChanges"`
	HighImpactChanges  int                `json:"highImpactChanges"`
	Changes            []*BreakingChange  `json:"changes"`
	OverallImpact      string             `json:"overallImpact"`
	MigrationRequired  bool               `json:"migrationRequired"`
	EstimatedEffort    string             `json:"estimatedEffort"`
	RecommendedActions []string           `json:"recommendedActions"`
	RiskAssessment     string             `json:"riskAssessment"`
}

// ImpactRecommendation represents a recommendation based on impact analysis
type ImpactRecommendation struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`        // "security", "performance", "compatibility", "business"
	Priority    string    `json:"priority"`    // "low", "medium", "high", "critical"
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Action      string    `json:"action"`
	Rationale   string    `json:"rationale"`
	Timeline    string    `json:"timeline,omitempty"`
	Resources   []string  `json:"resources,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
	RiskLevel   string    `json:"riskLevel"`
	Benefits    []string  `json:"benefits,omitempty"`
	Costs       []string  `json:"costs,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

// MitigationStrategy represents a strategy for mitigating risks
type MitigationStrategy struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        string    `json:"type"`        // "preventive", "detective", "corrective"
	Scope       string    `json:"scope"`       // "pre-update", "during-update", "post-update"
	Steps       []string  `json:"steps"`
	Resources   []string  `json:"resources,omitempty"`
	Timeline    string    `json:"timeline,omitempty"`
	Effectiveness float64 `json:"effectiveness"` // 0-1
	Cost        string    `json:"cost,omitempty"`
	Complexity  string    `json:"complexity"`    // "low", "medium", "high"
	Automated   bool      `json:"automated"`
	CreatedAt   time.Time `json:"createdAt"`
}

// TestingRecommendation represents a testing recommendation
type TestingRecommendation struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`        // "unit", "integration", "performance", "security", "compatibility"
	Priority    string    `json:"priority"`    // "required", "recommended", "optional"
	Description string    `json:"description"`
	TestCases   []string  `json:"testCases"`
	Environment string    `json:"environment,omitempty"`
	Tools       []string  `json:"tools,omitempty"`
	Duration    time.Duration `json:"duration,omitempty"`
	Resources   []string  `json:"resources,omitempty"`
	Automation  bool      `json:"automation"`
	Coverage    float64   `json:"coverage,omitempty"` // 0-100
	PassCriteria []string `json:"passCriteria,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

// VersionValidation represents validation results for a package version
type VersionValidation struct {
	Version        string              `json:"version"`
	Valid          bool                `json:"valid"`
	VersionFormat  string              `json:"versionFormat"`  // "semver", "date", "custom"
	Stability      string              `json:"stability"`      // "stable", "prerelease", "development"
	Compatibility  *CompatibilityCheck `json:"compatibility,omitempty"`
	Availability   bool                `json:"availability"`
	PublishedAt    *time.Time          `json:"publishedAt,omitempty"`
	DownloadCount  int64               `json:"downloadCount,omitempty"`
	SecurityStatus string              `json:"securityStatus,omitempty"`
	Errors         []string            `json:"errors,omitempty"`
	Warnings       []string            `json:"warnings,omitempty"`
	ValidatedAt    time.Time           `json:"validatedAt"`
}

// CompatibilityCheck represents a compatibility check result
type CompatibilityCheck struct {
	Compatible        bool      `json:"compatible"`
	RequiredVersion   string    `json:"requiredVersion,omitempty"`
	ConflictingWith   []string  `json:"conflictingWith,omitempty"`
	PlatformSupport   map[string]bool `json:"platformSupport,omitempty"`
	APICompatibility  bool      `json:"apiCompatibility"`
	ABI_Compatibility bool      `json:"abiCompatibility,omitempty"`
	Reason           string    `json:"reason,omitempty"`
}

// PackageSecurityValidation represents security validation for a package
type PackageSecurityValidation struct {
	SecurityScore      float64           `json:"securityScore"`
	Grade             string            `json:"grade"`            // "A", "B", "C", "D", "F"
	Vulnerabilities   []*Vulnerability  `json:"vulnerabilities,omitempty"`
	SecurityGaps      []*SecurityGap    `json:"securityGaps,omitempty"`
	LicenseCompliance bool              `json:"licenseCompliance"`
	SignatureValid    bool              `json:"signatureValid"`
	Checksum          string            `json:"checksum,omitempty"`
	ChecksumValid     bool              `json:"checksumValid"`
	TrustScore        float64           `json:"trustScore"`
	ScanDate          time.Time         `json:"scanDate"`
	Issues            []string          `json:"issues,omitempty"`
	Recommendations   []string          `json:"recommendations,omitempty"`
}