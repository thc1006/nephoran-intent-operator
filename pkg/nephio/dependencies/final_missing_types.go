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

// Final batch of missing types from resolver.go and validator.go

// TransitiveResult represents the result of transitive dependency resolution
type TransitiveResult struct {
	Success            bool                   `json:"success"`
	ResolvedDependencies []*ResolvedDependency `json:"resolvedDependencies"`
	Tree               *DependencyTree        `json:"tree"`
	Conflicts          []*DependencyConflict  `json:"conflicts,omitempty"`
	Errors             []error                `json:"errors,omitempty"`
	Statistics         *ResolutionStatistics  `json:"statistics"`
	ResolutionTime     time.Duration          `json:"resolutionTime"`
}

// ConstraintSolution represents a solution to dependency constraints
type ConstraintSolution struct {
	ID            string                `json:"id"`
	Name          string                `json:"name"`
	Description   string                `json:"description"`
	Packages      []*PackageReference   `json:"packages"`
	Constraints   []*Constraint         `json:"constraints"`
	Score         float64               `json:"score"`
	Feasible      bool                  `json:"feasible"`
	Optimal       bool                  `json:"optimal"`
	TradeOffs     []string              `json:"tradeOffs,omitempty"`
	GeneratedAt   time.Time             `json:"generatedAt"`
}

// Constraint represents a dependency constraint
type Constraint struct {
	ID          string      `json:"id"`
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Package     string      `json:"package,omitempty"`
	Version     string      `json:"version,omitempty"`
	Operator    string      `json:"operator,omitempty"`
	Value       interface{} `json:"value,omitempty"`
	Priority    int         `json:"priority,omitempty"`
	Required    bool        `json:"required"`
}

// ConstraintValidation represents validation results for constraints
type ConstraintValidation struct {
	Valid              bool                      `json:"valid"`
	ValidatedConstraints []*ConstraintResult     `json:"validatedConstraints"`
	Violations         []*ConstraintViolation   `json:"violations,omitempty"`
	Warnings           []*ConstraintWarning     `json:"warnings,omitempty"`
	OverallScore       float64                  `json:"overallScore"`
	ValidationTime     time.Duration            `json:"validationTime"`
	ValidatedAt        time.Time                `json:"validatedAt"`
}

// ConstraintResult represents the result of validating a single constraint
type ConstraintResult struct {
	Constraint *Constraint `json:"constraint"`
	Satisfied  bool        `json:"satisfied"`
	Score      float64     `json:"score"`
	Evidence   []string    `json:"evidence,omitempty"`
	Issues     []string    `json:"issues,omitempty"`
}

// ConstraintViolation represents a constraint violation
type ConstraintViolation struct {
	Constraint  *Constraint `json:"constraint"`
	Description string      `json:"description"`
	Severity    string      `json:"severity"`
	Impact      string      `json:"impact"`
	Remediation string      `json:"remediation,omitempty"`
	DetectedAt  time.Time   `json:"detectedAt"`
}

// ConstraintWarning represents a constraint warning
type ConstraintWarning struct {
	Constraint     *Constraint `json:"constraint"`
	Message        string      `json:"message"`
	Severity       string      `json:"severity"`
	Recommendation string      `json:"recommendation,omitempty"`
	DetectedAt     time.Time   `json:"detectedAt"`
}

// VersionRequirement represents a version requirement for a package
type VersionRequirement struct {
	Package    string   `json:"package"`
	Constraint string   `json:"constraint"` // e.g., ">=1.0.0,<2.0.0"
	Ranges     []string `json:"ranges,omitempty"`
	Source     string   `json:"source,omitempty"`
	Required   bool     `json:"required"`
	Optional   bool     `json:"optional,omitempty"`
}

// VersionResolution represents the resolution of version requirements
type VersionResolution struct {
	Package           string              `json:"package"`
	ResolvedVersion   string              `json:"resolvedVersion"`
	Requirements      []*VersionRequirement `json:"requirements"`
	Candidates        []*VersionCandidate `json:"candidates,omitempty"`
	ResolutionMethod  string              `json:"resolutionMethod"`
	ConflictsResolved []string            `json:"conflictsResolved,omitempty"`
	ResolutionTime    time.Duration       `json:"resolutionTime"`
	ResolvedAt        time.Time           `json:"resolvedAt"`
}

// VersionCandidate represents a candidate version for resolution
type VersionCandidate struct {
	Version      string    `json:"version"`
	Score        float64   `json:"score"`
	Compatible   bool      `json:"compatible"`
	Stable       bool      `json:"stable"`
	PreRelease   bool      `json:"preRelease,omitempty"`
	PublishedAt  time.Time `json:"publishedAt,omitempty"`
	DownloadCount int64    `json:"downloadCount,omitempty"`
	Reasons      []string  `json:"reasons,omitempty"`
}

// Conflict types for validator

// VersionConflict represents a version conflict between packages
type VersionConflict struct {
	ID                string              `json:"id"`
	PackageName       string              `json:"packageName"`
	ConflictingVersions []string          `json:"conflictingVersions"`
	RequiredBy        []*PackageReference `json:"requiredBy"`
	Severity          ConflictSeverity    `json:"severity"`
	Description       string              `json:"description"`
	ResolutionOptions []string            `json:"resolutionOptions,omitempty"`
	AutoResolvable    bool                `json:"autoResolvable"`
	DetectedAt        time.Time           `json:"detectedAt"`
}

// LicenseConflict represents a license conflict between packages
type LicenseConflict struct {
	ID                  string              `json:"id"`
	ConflictingPackages []*PackageReference `json:"conflictingPackages"`
	ConflictingLicenses []string            `json:"conflictingLicenses"`
	ConflictType        string              `json:"conflictType"` // "incompatible", "copyleft", "commercial"
	Severity            ConflictSeverity    `json:"severity"`
	Description         string              `json:"description"`
	LegalRisk           string              `json:"legalRisk"`
	ResolutionOptions   []string            `json:"resolutionOptions,omitempty"`
	DetectedAt          time.Time           `json:"detectedAt"`
}

// PolicyConflict represents a policy conflict
type PolicyConflict struct {
	ID                string              `json:"id"`
	PolicyName        string              `json:"policyName"`
	PolicyType        string              `json:"policyType"`
	ViolatingPackages []*PackageReference `json:"violatingPackages"`
	ViolationType     string              `json:"violationType"`
	Severity          ConflictSeverity    `json:"severity"`
	Description       string              `json:"description"`
	ComplianceRisk    string              `json:"complianceRisk"`
	RequiredAction    string              `json:"requiredAction"`
	ResolutionOptions []string            `json:"resolutionOptions,omitempty"`
	DetectedAt        time.Time           `json:"detectedAt"`
}

// ConflictResolutionSuggestion represents a suggestion for resolving conflicts
type ConflictResolutionSuggestion struct {
	ID          string   `json:"id"`
	ConflictID  string   `json:"conflictId"`
	Type        string   `json:"type"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Steps       []string `json:"steps"`
	Effort      string   `json:"effort"`
	Risk        string   `json:"risk"`
	Impact      string   `json:"impact"`
	Automated   bool     `json:"automated"`
	Priority    int      `json:"priority"`
	Confidence  float64  `json:"confidence"`
}

// ConflictImpactAnalysis represents analysis of conflict impact
type ConflictImpactAnalysis struct {
	AnalysisID      string                     `json:"analysisId"`
	Conflicts       []*DependencyConflict      `json:"conflicts"`
	OverallImpact   ConflictImpact             `json:"overallImpact"`
	ImpactScore     float64                    `json:"impactScore"`
	AffectedPackages []*PackageReference       `json:"affectedPackages"`
	BusinessImpact  string                     `json:"businessImpact,omitempty"`
	TechnicalImpact string                     `json:"technicalImpact,omitempty"`
	RiskAssessment  *ConflictRiskAssessment    `json:"riskAssessment,omitempty"`
	Recommendations []*ConflictResolutionSuggestion `json:"recommendations,omitempty"`
	AnalyzedAt      time.Time                  `json:"analyzedAt"`
}

// ConflictRiskAssessment represents risk assessment for conflicts
type ConflictRiskAssessment struct {
	RiskLevel        string    `json:"riskLevel"`
	RiskScore        float64   `json:"riskScore"`
	RiskFactors      []string  `json:"riskFactors"`
	MitigationSteps  []string  `json:"mitigationSteps,omitempty"`
	Probability      float64   `json:"probability"`      // 0-1
	ImpactSeverity   float64   `json:"impactSeverity"`   // 0-1
	TimeToResolution string    `json:"timeToResolution,omitempty"`
	AssessedAt       time.Time `json:"assessedAt"`
}