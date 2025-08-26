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

// Final missing types without duplicates

// PolicyValidation represents policy validation results
type PolicyValidation struct {
	Valid              bool                   `json:"valid"`
	PolicyScore        float64                `json:"policyScore"`
	PolicyChecks       []*PolicyCheck         `json:"policyChecks"`
	Violations         []*PolicyViolation     `json:"violations,omitempty"`
	OrganizationalCompliance bool             `json:"organizationalCompliance"`
	RegulatoryCompliance     bool             `json:"regulatoryCompliance"`
	ValidationTime     time.Duration          `json:"validationTime"`
	ValidatedAt        time.Time              `json:"validatedAt"`
}

// PolicyCheck represents a single policy check
type PolicyCheck struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	Status      string    `json:"status"`
	Score       float64   `json:"score"`
	Evidence    []string  `json:"evidence,omitempty"`
	Issues      []string  `json:"issues,omitempty"`
	CheckedAt   time.Time `json:"checkedAt"`
}

// ValidationRecommendation represents a validation recommendation
type ValidationRecommendation struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Priority    string    `json:"priority"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Action      string    `json:"action"`
	Package     string    `json:"package,omitempty"`
	Impact      string    `json:"impact"`
	Effort      string    `json:"effort"`
	Benefits    []string  `json:"benefits,omitempty"`
	Risks       []string  `json:"risks,omitempty"`
	Timeline    string    `json:"timeline,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

// ValidationStatistics represents validation statistics
type ValidationStatistics struct {
	TotalPackages      int           `json:"totalPackages"`
	ValidPackages      int           `json:"validPackages"`
	InvalidPackages    int           `json:"invalidPackages"`
	TotalChecks        int           `json:"totalChecks"`
	PassedChecks       int           `json:"passedChecks"`
	FailedChecks       int           `json:"failedChecks"`
	WarningCount       int           `json:"warningCount"`
	ErrorCount         int           `json:"errorCount"`
	ValidationTime     time.Duration `json:"validationTime"`
	CacheHitRate       float64       `json:"cacheHitRate,omitempty"`
	AveragePackageTime time.Duration `json:"averagePackageTime"`
}

// ApprovalRequest represents a request for update approval
type ApprovalRequest struct {
	ID            string              `json:"id"`
	Type          string              `json:"type"`          // "package_update", "security_update", "breaking_change"
	Priority      string              `json:"priority"`      // "low", "medium", "high", "critical"
	Title         string              `json:"title"`
	Description   string              `json:"description"`
	Packages      []*PackageReference `json:"packages"`
	Changes       []string            `json:"changes"`
	Impact        *UpdateImpact       `json:"impact,omitempty"`
	Justification string              `json:"justification"`
	Requester     string              `json:"requester"`
	RequestedAt   time.Time           `json:"requestedAt"`
	ExpiresAt     *time.Time          `json:"expiresAt,omitempty"`
	Status        string              `json:"status"`        // "pending", "approved", "rejected", "expired"
	ApprovedBy    []string            `json:"approvedBy,omitempty"`
	RejectedBy    []string            `json:"rejectedBy,omitempty"`
	Comments      []string            `json:"comments,omitempty"`
	ApprovedAt    *time.Time          `json:"approvedAt,omitempty"`
	RejectedAt    *time.Time          `json:"rejectedAt,omitempty"`
}

// UpdateImpact represents the overall impact of an update
type UpdateImpact struct {
	OverallLevel         string                `json:"overallLevel"`
	SecurityImpact       *SecurityImpact       `json:"securityImpact,omitempty"`
	PerformanceImpact    *PerformanceImpact    `json:"performanceImpact,omitempty"`
	CompatibilityImpact  *CompatibilityImpact  `json:"compatibilityImpact,omitempty"`
	BusinessImpact       *BusinessImpact       `json:"businessImpact,omitempty"`
	BreakingChangeImpact *BreakingChangeImpact `json:"breakingChangeImpact,omitempty"`
	RiskAssessment       string                `json:"riskAssessment"`
	MitigationRequired   bool                  `json:"mitigationRequired"`
	TestingRequired      bool                  `json:"testingRequired"`
	RollbackPlan         string                `json:"rollbackPlan,omitempty"`
	ImpactScore          float64               `json:"impactScore"`
}

// UpdateError represents an update error
type UpdateError struct {
	Code        string                 `json:"code"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Message     string                 `json:"message"`
	Package     *PackageReference      `json:"package,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Remediation string                 `json:"remediation,omitempty"`
	Retryable   bool                   `json:"retryable"`
	OccurredAt  time.Time              `json:"occurredAt"`
	StackTrace  string                 `json:"stackTrace,omitempty"`
}

// UpdateWarning represents an update warning
type UpdateWarning struct {
	Code           string                 `json:"code"`
	Type           string                 `json:"type"`
	Message        string                 `json:"message"`
	Package        *PackageReference      `json:"package,omitempty"`
	Context        map[string]interface{} `json:"context,omitempty"`
	Recommendation string                 `json:"recommendation,omitempty"`
	Impact         string                 `json:"impact"`
	Dismissible    bool                   `json:"dismissible"`
	OccurredAt     time.Time              `json:"occurredAt"`
}

// UpdateStatistics represents update statistics
type UpdateStatistics struct {
	TotalPackages     int           `json:"totalPackages"`
	UpdatedPackages   int           `json:"updatedPackages"`
	FailedPackages    int           `json:"failedPackages"`
	SkippedPackages   int           `json:"skippedPackages"`
	SecurityUpdates   int           `json:"securityUpdates"`
	MajorUpdates      int           `json:"majorUpdates"`
	MinorUpdates      int           `json:"minorUpdates"`
	PatchUpdates      int           `json:"patchUpdates"`
	TotalUpdateTime   time.Duration `json:"totalUpdateTime"`
	AverageUpdateTime time.Duration `json:"averageUpdateTime"`
	ConflictsResolved int           `json:"conflictsResolved"`
	RollbacksRequired int           `json:"rollbacksRequired"`
	ApprovalsPending  int           `json:"approvalsPending,omitempty"`
}

// PropagationFilter represents a filter for update propagation
type PropagationFilter struct {
	IncludePackages   []string          `json:"includePackages,omitempty"`
	ExcludePackages   []string          `json:"excludePackages,omitempty"`
	IncludeScopes     []DependencyScope `json:"includeScopes,omitempty"`
	ExcludeScopes     []DependencyScope `json:"excludeScopes,omitempty"`
	MaxDepth          int               `json:"maxDepth,omitempty"`
	UpdateTypes       []string          `json:"updateTypes,omitempty"`     // "major", "minor", "patch", "security"
	Environments      []string          `json:"environments,omitempty"`
	RequireApproval   bool              `json:"requireApproval,omitempty"`
	SecurityOnly      bool              `json:"securityOnly,omitempty"`
	FollowTransitive  bool              `json:"followTransitive"`
	VersionConstraints map[string]string `json:"versionConstraints,omitempty"`
}