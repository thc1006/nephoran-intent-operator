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

// Absolute final missing types without duplicates

// AutoUpdateConfig represents configuration for automatic updates
type AutoUpdateConfig struct {
	Enabled              bool              `json:"enabled"`
	Schedule             string            `json:"schedule"`            // cron expression
	UpdateTypes          []string          `json:"updateTypes"`         // "security", "patch", "minor", "major"
	AutoApprove          []string          `json:"autoApprove,omitempty"`
	RequireApproval      []string          `json:"requireApproval,omitempty"`
	Environments         []string          `json:"environments,omitempty"`
	MaintenanceWindows   []string          `json:"maintenanceWindows,omitempty"`
	RollbackOnFailure    bool              `json:"rollbackOnFailure"`
	MaxConcurrency       int               `json:"maxConcurrency,omitempty"`
	NotificationChannels []string          `json:"notificationChannels,omitempty"`
	DryRun              bool              `json:"dryRun,omitempty"`
}

// UpdateSchedule represents a scheduled update
type UpdateSchedule struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Schedule    string    `json:"schedule"`    // cron expression
	Enabled     bool      `json:"enabled"`
	NextRun     time.Time `json:"nextRun"`
	LastRun     *time.Time `json:"lastRun,omitempty"`
	Config      *AutoUpdateConfig `json:"config"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// ScheduledUpdate represents a scheduled update execution
type ScheduledUpdate struct {
	ID         string              `json:"id"`
	ScheduleID string              `json:"scheduleId"`
	Packages   []*PackageReference `json:"packages"`
	Status     string              `json:"status"`     // "pending", "running", "completed", "failed", "cancelled"
	StartedAt  *time.Time          `json:"startedAt,omitempty"`
	CompletedAt *time.Time         `json:"completedAt,omitempty"`
	Duration   time.Duration       `json:"duration,omitempty"`
	Results    *UpdateResult       `json:"results,omitempty"`
	Errors     []string            `json:"errors,omitempty"`
	CreatedAt  time.Time           `json:"createdAt"`
}

// UpdatePlan represents a plan for updating dependencies
type UpdatePlan struct {
	ID              string              `json:"id"`
	Name            string              `json:"name"`
	Description     string              `json:"description,omitempty"`
	Packages        []*PackageReference `json:"packages"`
	UpdateStrategy  string              `json:"updateStrategy"`  // "all", "security-only", "selective"
	RolloutConfig   *RolloutConfig      `json:"rolloutConfig,omitempty"`
	Constraints     *UpdateConstraints  `json:"constraints,omitempty"`
	EstimatedTime   time.Duration       `json:"estimatedTime,omitempty"`
	Impact          *UpdateImpact       `json:"impact,omitempty"`
	Prerequisites   []string            `json:"prerequisites,omitempty"`
	ValidationRules []*ValidationRule   `json:"validationRules,omitempty"`
	CreatedBy       string              `json:"createdBy"`
	CreatedAt       time.Time           `json:"createdAt"`
	Status          string              `json:"status"`          // "draft", "approved", "executing", "completed", "failed"
}

// ValidationRule represents a validation rule for updates
type ValidationRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`        // "security", "performance", "compatibility"
	Condition   string                 `json:"condition"`   // expression to evaluate
	Action      string                 `json:"action"`      // "block", "warn", "require-approval"
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Enabled     bool                   `json:"enabled"`
	Priority    int                    `json:"priority"`
}

// PropagatedUpdate represents an update that has been propagated
type PropagatedUpdate struct {
	ID              string              `json:"id"`
	SourcePackage   *PackageReference   `json:"sourcePackage"`
	TargetPackages  []*PackageReference `json:"targetPackages"`
	Environment     string              `json:"environment"`
	PropagationType string              `json:"propagationType"` // "cascade", "selective", "batch"
	Status          string              `json:"status"`
	StartedAt       time.Time           `json:"startedAt"`
	CompletedAt     *time.Time          `json:"completedAt,omitempty"`
	Results         []*UpdateResult     `json:"results,omitempty"`
	Errors          []string            `json:"errors,omitempty"`
}

// FailedPropagation represents a failed propagation
type FailedPropagation struct {
	ID            string              `json:"id"`
	Package       *PackageReference   `json:"package"`
	Environment   string              `json:"environment"`
	Error         string              `json:"error"`
	ErrorCode     string              `json:"errorCode,omitempty"`
	Recoverable   bool                `json:"recoverable"`
	RetryCount    int                 `json:"retryCount"`
	NextRetry     *time.Time          `json:"nextRetry,omitempty"`
	FailedAt      time.Time           `json:"failedAt"`
	StackTrace    string              `json:"stackTrace,omitempty"`
}

// SkippedPropagation represents a skipped propagation
type SkippedPropagation struct {
	ID          string              `json:"id"`
	Package     *PackageReference   `json:"package"`
	Environment string              `json:"environment"`
	Reason      string              `json:"reason"`
	ReasonCode  string              `json:"reasonCode,omitempty"`
	SkippedAt   time.Time           `json:"skippedAt"`
	NextAttempt *time.Time          `json:"nextAttempt,omitempty"`
}

// EnvironmentUpdateResult represents update results for an environment
type EnvironmentUpdateResult struct {
	Environment      string               `json:"environment"`
	Success          bool                 `json:"success"`
	UpdatedPackages  []*UpdatedPackage    `json:"updatedPackages"`
	FailedUpdates    []*FailedUpdate      `json:"failedUpdates"`
	SkippedUpdates   []*SkippedUpdate     `json:"skippedUpdates"`
	TotalTime        time.Duration        `json:"totalTime"`
	StartedAt        time.Time            `json:"startedAt"`
	CompletedAt      *time.Time           `json:"completedAt,omitempty"`
	RollbackRequired bool                 `json:"rollbackRequired,omitempty"`
	HealthStatus     string               `json:"healthStatus,omitempty"`
}

// PropagationError represents a propagation error
type PropagationError struct {
	Code        string                 `json:"code"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Message     string                 `json:"message"`
	Package     *PackageReference      `json:"package,omitempty"`
	Environment string                 `json:"environment,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Remediation string                 `json:"remediation,omitempty"`
	Retryable   bool                   `json:"retryable"`
	OccurredAt  time.Time              `json:"occurredAt"`
}

// PropagationWarning represents a propagation warning
type PropagationWarning struct {
	Code           string                 `json:"code"`
	Type           string                 `json:"type"`
	Message        string                 `json:"message"`
	Package        *PackageReference      `json:"package,omitempty"`
	Environment    string                 `json:"environment,omitempty"`
	Context        map[string]interface{} `json:"context,omitempty"`
	Recommendation string                 `json:"recommendation,omitempty"`
	Impact         string                 `json:"impact"`
	OccurredAt     time.Time              `json:"occurredAt"`
}