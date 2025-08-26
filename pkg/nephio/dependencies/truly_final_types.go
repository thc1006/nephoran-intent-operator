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

// Truly final missing types from updater.go

// PlanValidation represents validation results for an update plan
type PlanValidation struct {
	Valid           bool                    `json:"valid"`
	ValidationScore float64                 `json:"validationScore"`
	Checks          []*PlanValidationCheck  `json:"checks"`
	Issues          []*PlanValidationIssue  `json:"issues,omitempty"`
	Warnings        []*PlanValidationWarning `json:"warnings,omitempty"`
	Recommendations []string                `json:"recommendations,omitempty"`
	EstimatedRisk   string                  `json:"estimatedRisk"`
	ValidatedAt     time.Time               `json:"validatedAt"`
	ValidatedBy     string                  `json:"validatedBy,omitempty"`
}

// PlanValidationCheck represents a single validation check
type PlanValidationCheck struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	Score       float64   `json:"score"`
	Details     string    `json:"details,omitempty"`
	CheckedAt   time.Time `json:"checkedAt"`
}

// PlanValidationIssue represents a validation issue
type PlanValidationIssue struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Package     string    `json:"package,omitempty"`
	Resolution  string    `json:"resolution,omitempty"`
	Blocking    bool      `json:"blocking"`
	DetectedAt  time.Time `json:"detectedAt"`
}

// PlanValidationWarning represents a validation warning
type PlanValidationWarning struct {
	ID             string    `json:"id"`
	Type           string    `json:"type"`
	Message        string    `json:"message"`
	Package        string    `json:"package,omitempty"`
	Recommendation string    `json:"recommendation,omitempty"`
	Impact         string    `json:"impact"`
	DetectedAt     time.Time `json:"detectedAt"`
}

// RollbackValidation represents validation for rollback operations
type RollbackValidation struct {
	Valid           bool                        `json:"valid"`
	RollbackPlan    *RollbackPlan              `json:"rollbackPlan"`
	Feasible        bool                        `json:"feasible"`
	EstimatedTime   time.Duration               `json:"estimatedTime"`
	DataLoss        bool                        `json:"dataLoss"`
	ServiceImpact   string                      `json:"serviceImpact"`
	Prerequisites   []string                    `json:"prerequisites,omitempty"`
	Risks           []string                    `json:"risks,omitempty"`
	Checks          []*RollbackValidationCheck  `json:"checks"`
	ValidatedAt     time.Time                   `json:"validatedAt"`
}

// RollbackValidationCheck represents a rollback validation check
type RollbackValidationCheck struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Details     string    `json:"details,omitempty"`
	CheckedAt   time.Time `json:"checkedAt"`
}

// UpdateHistoryFilter represents a filter for update history
type UpdateHistoryFilter struct {
	Packages    []string    `json:"packages,omitempty"`
	UpdateTypes []string    `json:"updateTypes,omitempty"`
	Status      []string    `json:"status,omitempty"`
	FromDate    *time.Time  `json:"fromDate,omitempty"`
	ToDate      *time.Time  `json:"toDate,omitempty"`
	Environment string      `json:"environment,omitempty"`
	Requester   string      `json:"requester,omitempty"`
	Limit       int         `json:"limit,omitempty"`
	Offset      int         `json:"offset,omitempty"`
}

// UpdateRecord represents a historical update record
type UpdateRecord struct {
	ID              string              `json:"id"`
	Type            string              `json:"type"`
	Package         *PackageReference   `json:"package,omitempty"`
	Packages        []*PackageReference `json:"packages,omitempty"`
	FromVersion     string              `json:"fromVersion,omitempty"`
	ToVersion       string              `json:"toVersion,omitempty"`
	Status          string              `json:"status"`
	Environment     string              `json:"environment,omitempty"`
	Requester       string              `json:"requester"`
	StartedAt       time.Time           `json:"startedAt"`
	CompletedAt     *time.Time          `json:"completedAt,omitempty"`
	Duration        time.Duration       `json:"duration,omitempty"`
	Result          *UpdateResult       `json:"result,omitempty"`
	RollbackRecord  *RollbackRecord     `json:"rollbackRecord,omitempty"`
	ApprovalProcess *ApprovalProcess    `json:"approvalProcess,omitempty"`
}

// RollbackRecord represents a rollback record
type RollbackRecord struct {
	ID          string        `json:"id"`
	Reason      string        `json:"reason"`
	TriggeredBy string        `json:"triggeredBy"`
	StartedAt   time.Time     `json:"startedAt"`
	CompletedAt *time.Time    `json:"completedAt,omitempty"`
	Duration    time.Duration `json:"duration,omitempty"`
	Success     bool          `json:"success"`
	Result      *RollbackResult `json:"result,omitempty"`
}

// ApprovalProcess represents an approval process
type ApprovalProcess struct {
	ID          string              `json:"id"`
	Status      string              `json:"status"`
	Approvers   []*ApprovalRecord   `json:"approvers"`
	StartedAt   time.Time           `json:"startedAt"`
	CompletedAt *time.Time          `json:"completedAt,omitempty"`
	Duration    time.Duration       `json:"duration,omitempty"`
}

// ApprovalRecord represents a single approval record
type ApprovalRecord struct {
	Approver    string     `json:"approver"`
	Status      string     `json:"status"`     // "pending", "approved", "rejected"
	Decision    string     `json:"decision,omitempty"`
	Comments    string     `json:"comments,omitempty"`
	DecisionAt  *time.Time `json:"decisionAt,omitempty"`
}

// ChangeTracker represents a change tracker for monitoring updates
type ChangeTracker interface {
	TrackChange(change *ChangeEvent) error
	GetChanges(filter *ChangeFilter) ([]*ChangeEvent, error)
	GetChangeReport(timeRange *TimeRange) (*ChangeReport, error)
}

// ChangeEvent represents a change event
type ChangeEvent struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`        // "update", "rollback", "approval"
	Package     *PackageReference      `json:"package,omitempty"`
	OldValue    interface{}            `json:"oldValue,omitempty"`
	NewValue    interface{}            `json:"newValue,omitempty"`
	Environment string                 `json:"environment,omitempty"`
	User        string                 `json:"user"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ChangeFilter represents a filter for changes
type ChangeFilter struct {
	Types       []string   `json:"types,omitempty"`
	Packages    []string   `json:"packages,omitempty"`
	Users       []string   `json:"users,omitempty"`
	Environment string     `json:"environment,omitempty"`
	FromDate    *time.Time `json:"fromDate,omitempty"`
	ToDate      *time.Time `json:"toDate,omitempty"`
	Limit       int        `json:"limit,omitempty"`
	Offset      int        `json:"offset,omitempty"`
}

// ChangeReport represents a change report
type ChangeReport struct {
	ReportID    string         `json:"reportId"`
	TimeRange   *TimeRange     `json:"timeRange"`
	TotalChanges int           `json:"totalChanges"`
	ChangesByType map[string]int `json:"changesByType"`
	ChangesByUser map[string]int `json:"changesByUser"`
	ChangesByPackage map[string]int `json:"changesByPackage"`
	Changes     []*ChangeEvent `json:"changes"`
	GeneratedAt time.Time      `json:"generatedAt"`
	GeneratedBy string         `json:"generatedBy,omitempty"`
}

// ApprovalFilter represents a filter for approvals
type ApprovalFilter struct {
	Status      []string   `json:"status,omitempty"`
	Types       []string   `json:"types,omitempty"`
	Approvers   []string   `json:"approvers,omitempty"`
	Requesters  []string   `json:"requesters,omitempty"`
	Priority    []string   `json:"priority,omitempty"`
	FromDate    *time.Time `json:"fromDate,omitempty"`
	ToDate      *time.Time `json:"toDate,omitempty"`
	Limit       int        `json:"limit,omitempty"`
	Offset      int        `json:"offset,omitempty"`
}

// UpdateNotification represents a notification about updates
type UpdateNotification struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`        // "update_started", "update_completed", "update_failed", "approval_required"
	Priority    string                 `json:"priority"`    // "low", "medium", "high", "critical"
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Recipients  []string               `json:"recipients"`
	Channels    []string               `json:"channels"`    // "email", "slack", "webhook"
	Package     *PackageReference      `json:"package,omitempty"`
	UpdateID    string                 `json:"updateId,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	SentAt      *time.Time             `json:"sentAt,omitempty"`
	Status      string                 `json:"status"`      // "pending", "sent", "failed"
	DeliveryStatus map[string]string   `json:"deliveryStatus,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
}

// UpdaterHealth represents the health status of the updater
type UpdaterHealth struct {
	Status           string                   `json:"status"`      // "healthy", "degraded", "unhealthy"
	Uptime           time.Duration            `json:"uptime"`
	LastHealthCheck  time.Time                `json:"lastHealthCheck"`
	ComponentHealth  map[string]string        `json:"componentHealth"`
	ActiveUpdates    int                      `json:"activeUpdates"`
	PendingApprovals int                      `json:"pendingApprovals"`
	FailedUpdates    int                      `json:"failedUpdates"`
	Issues           []string                 `json:"issues,omitempty"`
	Warnings         []string                 `json:"warnings,omitempty"`
	Metrics          *UpdaterMetrics          `json:"metrics,omitempty"`
	Version          string                   `json:"version,omitempty"`
}

// UpdaterMetrics represents metrics for the updater
type UpdaterMetrics struct {
	TotalUpdates        int64         `json:"totalUpdates"`
	SuccessfulUpdates   int64         `json:"successfulUpdates"`
	FailedUpdates       int64         `json:"failedUpdates"`
	AverageUpdateTime   time.Duration `json:"averageUpdateTime"`
	UpdatesPerHour      float64       `json:"updatesPerHour"`
	ErrorRate           float64       `json:"errorRate"`
	RollbackRate        float64       `json:"rollbackRate"`
	ApprovalPendingRate float64       `json:"approvalPendingRate"`
	LastUpdateTime      *time.Time    `json:"lastUpdateTime,omitempty"`
	MetricsCollectedAt  time.Time     `json:"metricsCollectedAt"`
}