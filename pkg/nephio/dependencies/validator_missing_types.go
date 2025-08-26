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

// Validator missing types without duplicates

// CompatibilityResult represents the result of compatibility validation
type CompatibilityResult struct {
	Packages            []*PackageReference         `json:"packages"`
	Compatible          bool                        `json:"compatible"`
	CompatibilityMatrix map[string]map[string]bool  `json:"compatibilityMatrix"`
	IncompatiblePairs   []*IncompatiblePair         `json:"incompatiblePairs,omitempty"`
	PlatformSupport     map[string]bool             `json:"platformSupport,omitempty"`
	VersionConflicts    []*VersionConflict          `json:"versionConflicts,omitempty"`
	ValidationTime      time.Duration               `json:"validationTime"`
	ValidatedAt         time.Time                   `json:"validatedAt"`
}

// IncompatiblePair represents a pair of incompatible packages
type IncompatiblePair struct {
	Package1 *PackageReference `json:"package1"`
	Package2 *PackageReference `json:"package2"`
	Reason   string            `json:"reason"`
	Severity string            `json:"severity"`
	Impact   string            `json:"impact,omitempty"`
}

// LicenseValidation represents the result of license validation
type LicenseValidation struct {
	Valid              bool                        `json:"valid"`
	PackageLicenses    []*PackageLicenseValidation `json:"packageLicenses"`
	ConflictingLicenses []*LicenseConflict         `json:"conflictingLicenses,omitempty"`
	ComplianceStatus   map[string]string           `json:"complianceStatus,omitempty"`
	LegalRisk          string                      `json:"legalRisk"`
	RecommendedActions []string                    `json:"recommendedActions,omitempty"`
	ValidationTime     time.Duration               `json:"validationTime"`
	ValidatedAt        time.Time                   `json:"validatedAt"`
}

// PackageLicenseValidation represents license validation for a single package
type PackageLicenseValidation struct {
	Package           *PackageReference `json:"package"`
	License           string            `json:"license"`
	LicenseType       string            `json:"licenseType"`   // "permissive", "copyleft", "commercial", "proprietary"
	OSIApproved       bool              `json:"osiApproved"`
	Compatible        bool              `json:"compatible"`
	LegalRisk         string            `json:"legalRisk"`     // "none", "low", "medium", "high"
	ComplianceChecks  map[string]bool   `json:"complianceChecks,omitempty"`
	Restrictions      []string          `json:"restrictions,omitempty"`
	Requirements      []string          `json:"requirements,omitempty"`
	ValidationErrors  []string          `json:"validationErrors,omitempty"`
}

// ComplianceValidation represents the result of compliance validation
type ComplianceValidation struct {
	Compliant          bool                      `json:"compliant"`
	ComplianceScore    float64                   `json:"complianceScore"`
	ComplianceChecks   []*ComplianceCheck        `json:"complianceChecks"`
	Violations         []*ComplianceViolation    `json:"violations,omitempty"`
	Requirements       []*ComplianceRequirement  `json:"requirements"`
	Frameworks         []string                  `json:"frameworks,omitempty"`
	CertificationLevel string                    `json:"certificationLevel,omitempty"`
	ValidationTime     time.Duration             `json:"validationTime"`
	ValidatedAt        time.Time                 `json:"validatedAt"`
}

// ComplianceCheck represents a single compliance check
type ComplianceCheck struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Framework   string    `json:"framework"`
	Category    string    `json:"category"`
	Status      string    `json:"status"`      // "passed", "failed", "warning", "not_applicable"
	Score       float64   `json:"score"`
	Evidence    []string  `json:"evidence,omitempty"`
	Issues      []string  `json:"issues,omitempty"`
	CheckedAt   time.Time `json:"checkedAt"`
}

// ComplianceRequirement represents a compliance requirement
type ComplianceRequirement struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Framework   string    `json:"framework"`
	Description string    `json:"description"`
	Required    bool      `json:"required"`
	Status      string    `json:"status"`
	Evidence    []string  `json:"evidence,omitempty"`
	Gaps        []string  `json:"gaps,omitempty"`
}

// PerformanceValidation represents the result of performance validation
type PerformanceValidation struct {
	Valid              bool                         `json:"valid"`
	OverallScore       float64                      `json:"overallScore"`
	PackagePerformance []*PackagePerformanceResult  `json:"packagePerformance"`
	PerformanceIssues  []*PerformanceIssue          `json:"performanceIssues,omitempty"`
	BenchmarkResults   []*BenchmarkResult           `json:"benchmarkResults,omitempty"`
	ResourceUsage      *ResourceUsageAnalysis       `json:"resourceUsage,omitempty"`
	ValidationTime     time.Duration                `json:"validationTime"`
	ValidatedAt        time.Time                    `json:"validatedAt"`
}

// PackagePerformanceResult represents performance results for a package
type PackagePerformanceResult struct {
	Package          *PackageReference      `json:"package"`
	PerformanceScore float64                `json:"performanceScore"`
	Metrics          map[string]float64     `json:"metrics"`
	Benchmarks       []*BenchmarkResult     `json:"benchmarks,omitempty"`
	ResourceUsage    *ResourceUsageStats    `json:"resourceUsage,omitempty"`
	Issues           []string               `json:"issues,omitempty"`
}

// PerformanceIssue represents a performance issue
type PerformanceIssue struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`        // "memory", "cpu", "io", "network", "startup"
	Severity    string    `json:"severity"`    // "low", "medium", "high", "critical"
	Description string    `json:"description"`
	Package     string    `json:"package,omitempty"`
	Metric      string    `json:"metric,omitempty"`
	Value       float64   `json:"value,omitempty"`
	Threshold   float64   `json:"threshold,omitempty"`
	Impact      string    `json:"impact,omitempty"`
	Suggestion  string    `json:"suggestion,omitempty"`
	DetectedAt  time.Time `json:"detectedAt"`
}

// QualityValidation represents quality validation results
type QualityValidation struct {
	Valid           bool                    `json:"valid"`
	QualityScore    float64                 `json:"qualityScore"`
	Grade           string                  `json:"grade"`
	QualityChecks   []*QualityCheck         `json:"qualityChecks"`
	CodeQuality     *CodeQualityMetrics     `json:"codeQuality,omitempty"`
	Documentation   *DocumentationMetrics   `json:"documentation,omitempty"`
	TestCoverage    *TestCoverageMetrics    `json:"testCoverage,omitempty"`
	MaintenanceScore float64                `json:"maintenanceScore,omitempty"`
	ValidationTime  time.Duration           `json:"validationTime"`
	ValidatedAt     time.Time               `json:"validatedAt"`
}

// QualityCheck represents a single quality check
type QualityCheck struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`    // "code", "documentation", "testing", "maintenance"
	Status      string    `json:"status"`      // "passed", "failed", "warning"
	Score       float64   `json:"score"`
	Details     string    `json:"details,omitempty"`
	Issues      []string  `json:"issues,omitempty"`
	CheckedAt   time.Time `json:"checkedAt"`
}

// CodeQualityMetrics represents code quality metrics
type CodeQualityMetrics struct {
	Complexity      float64 `json:"complexity"`
	Maintainability float64 `json:"maintainability"`
	Readability     float64 `json:"readability"`
	TechnicalDebt   float64 `json:"technicalDebt"`
	CodeSmells      int     `json:"codeSmells"`
	Duplications    float64 `json:"duplications"`
	BugPotential    int     `json:"bugPotential"`
}

// DocumentationMetrics represents documentation metrics
type DocumentationMetrics struct {
	Coverage        float64 `json:"coverage"`
	Quality         float64 `json:"quality"`
	Completeness    float64 `json:"completeness"`
	Accuracy        float64 `json:"accuracy"`
	MissingDocs     []string `json:"missingDocs,omitempty"`
	OutdatedDocs    []string `json:"outdatedDocs,omitempty"`
}

// TestCoverageMetrics represents test coverage metrics
type TestCoverageMetrics struct {
	LineCoverage     float64 `json:"lineCoverage"`
	BranchCoverage   float64 `json:"branchCoverage"`
	FunctionCoverage float64 `json:"functionCoverage"`
	TestCount        int     `json:"testCount"`
	PassingTests     int     `json:"passingTests"`
	FailingTests     int     `json:"failingTests"`
	SkippedTests     int     `json:"skippedTests"`
}

// PolicyViolation represents a policy violation
type PolicyViolation struct {
	ID          string    `json:"id"`
	PolicyID    string    `json:"policyId"`
	PolicyName  string    `json:"policyName"`
	ViolationType string  `json:"violationType"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Package     string    `json:"package,omitempty"`
	Component   string    `json:"component,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Remediation string    `json:"remediation,omitempty"`
	DetectedAt  time.Time `json:"detectedAt"`
}

// ErrorSeverity represents error severity levels
type ErrorSeverity string

const (
	ErrorSeverityLow      ErrorSeverity = "low"
	ErrorSeverityMedium   ErrorSeverity = "medium"
	ErrorSeverityHigh     ErrorSeverity = "high"
	ErrorSeverityCritical ErrorSeverity = "critical"
)

// WarningType represents types of warnings
type WarningType string

const (
	WarningTypeValidation    WarningType = "validation"
	WarningTypeCompatibility WarningType = "compatibility"
	WarningTypePerformance   WarningType = "performance"
	WarningTypeSecurity      WarningType = "security"
	WarningTypeLicense       WarningType = "license"
	WarningTypePolicy        WarningType = "policy"
	WarningTypeCompliance    WarningType = "compliance"
	WarningTypeConfiguration WarningType = "configuration"
	WarningTypeMaintenance   WarningType = "maintenance"
)

// WarningImpact represents the impact level of warnings
type WarningImpact string

const (
	WarningImpactLow      WarningImpact = "low"
	WarningImpactMedium   WarningImpact = "medium"
	WarningImpactHigh     WarningImpact = "high"
	WarningImpactCritical WarningImpact = "critical"
)