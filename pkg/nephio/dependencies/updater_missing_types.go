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

// Missing types from updater.go

// ValidationConfig represents configuration for validation
type ValidationConfig struct {
	EnableLinting       bool     `json:"enableLinting"`
	EnableSecurity      bool     `json:"enableSecurity"`
	EnableCompliance    bool     `json:"enableCompliance"`
	EnablePerformance   bool     `json:"enablePerformance"`
	RequiredChecks      []string `json:"requiredChecks"`
	OptionalChecks      []string `json:"optionalChecks,omitempty"`
	FailOnWarnings      bool     `json:"failOnWarnings"`
	Timeout             time.Duration `json:"timeout"`
	CustomValidators    []string `json:"customValidators,omitempty"`
}

// ApprovalPolicy represents policy for update approvals
type ApprovalPolicy struct {
	Required            bool     `json:"required"`
	RequiredApprovers   int      `json:"requiredApprovers"`
	ApproverGroups      []string `json:"approverGroups,omitempty"`
	AutoApprove         []string `json:"autoApprove,omitempty"`
	AlwaysRequire       []string `json:"alwaysRequire,omitempty"`
	Timeout             time.Duration `json:"timeout"`
	AllowSelfApproval   bool     `json:"allowSelfApproval"`
}

// NotificationConfig represents notification configuration
type NotificationConfig struct {
	Enabled     bool     `json:"enabled"`
	Channels    []string `json:"channels"`    // "email", "slack", "webhook", "sms"
	Recipients  []string `json:"recipients"`
	Events      []string `json:"events"`     // "start", "success", "failure", "conflict", "approval"
	Templates   map[string]string `json:"templates,omitempty"`
	Webhooks    []string `json:"webhooks,omitempty"`
	SlackConfig *SlackNotificationConfig `json:"slackConfig,omitempty"`
	EmailConfig *EmailNotificationConfig `json:"emailConfig,omitempty"`
}

// SlackNotificationConfig represents Slack notification configuration
type SlackNotificationConfig struct {
	Token       string `json:"token"`
	Channel     string `json:"channel"`
	Username    string `json:"username,omitempty"`
	IconEmoji   string `json:"iconEmoji,omitempty"`
	IconURL     string `json:"iconURL,omitempty"`
}

// EmailNotificationConfig represents email notification configuration
type EmailNotificationConfig struct {
	SMTPHost     string   `json:"smtpHost"`
	SMTPPort     int      `json:"smtpPort"`
	Username     string   `json:"username"`
	Password     string   `json:"password"`
	FromAddress  string   `json:"fromAddress"`
	ToAddresses  []string `json:"toAddresses"`
	Subject      string   `json:"subject,omitempty"`
	BodyTemplate string   `json:"bodyTemplate,omitempty"`
}

// UpdatedPackage represents a successfully updated package
type UpdatedPackage struct {
	Package       *PackageReference `json:"package"`
	PreviousVersion string          `json:"previousVersion"`
	NewVersion    string            `json:"newVersion"`
	UpdateType    string            `json:"updateType"` // "major", "minor", "patch", "security"
	UpdateTime    time.Duration     `json:"updateTime"`
	UpdatedAt     time.Time         `json:"updatedAt"`
	ChangesSummary []string         `json:"changesSummary,omitempty"`
	ReleaseNotes  string            `json:"releaseNotes,omitempty"`
}

// FailedUpdate represents a failed package update
type FailedUpdate struct {
	Package       *PackageReference `json:"package"`
	TargetVersion string            `json:"targetVersion"`
	Error         string            `json:"error"`
	ErrorCode     string            `json:"errorCode,omitempty"`
	Recoverable   bool              `json:"recoverable"`
	RetryCount    int               `json:"retryCount"`
	FailedAt      time.Time         `json:"failedAt"`
	Suggestions   []string          `json:"suggestions,omitempty"`
}

// SkippedUpdate represents a skipped package update
type SkippedUpdate struct {
	Package       *PackageReference `json:"package"`
	TargetVersion string            `json:"targetVersion"`
	Reason        string            `json:"reason"`
	ReasonCode    string            `json:"reasonCode,omitempty"`
	SkippedAt     time.Time         `json:"skippedAt"`
	NextAttempt   *time.Time        `json:"nextAttempt,omitempty"`
}

// RolloutExecution represents the execution of a rollout
type RolloutExecution struct {
	ID            string            `json:"id"`
	Status        string            `json:"status"` // "pending", "running", "completed", "failed", "cancelled"
	StartedAt     time.Time         `json:"startedAt"`
	CompletedAt   *time.Time        `json:"completedAt,omitempty"`
	Duration      time.Duration     `json:"duration"`
	Progress      float64           `json:"progress"` // 0-100
	CurrentBatch  int               `json:"currentBatch"`
	TotalBatches  int               `json:"totalBatches"`
	SuccessCount  int               `json:"successCount"`
	FailureCount  int               `json:"failureCount"`
	SkippedCount  int               `json:"skippedCount"`
	Errors        []string          `json:"errors,omitempty"`
	Logs          []string          `json:"logs,omitempty"`
}

// MaintenanceWindow represents a maintenance window for updates
type MaintenanceWindow struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	StartTime   time.Time `json:"startTime"`
	EndTime     time.Time `json:"endTime"`
	Timezone    string    `json:"timezone"`
	Recurrence  *RecurrenceConfig `json:"recurrence,omitempty"`
	Enabled     bool      `json:"enabled"`
	Priority    int       `json:"priority"`
	Exceptions  []time.Time `json:"exceptions,omitempty"`
}

// RecurrenceConfig represents recurrence configuration for maintenance windows
type RecurrenceConfig struct {
	Type       string `json:"type"`       // "daily", "weekly", "monthly", "yearly"
	Interval   int    `json:"interval"`   // every N periods
	DaysOfWeek []int  `json:"daysOfWeek,omitempty"` // 0=Sunday, 1=Monday, etc.
	DaysOfMonth []int `json:"daysOfMonth,omitempty"` // 1-31
	EndDate    *time.Time `json:"endDate,omitempty"`
	MaxOccurrences int `json:"maxOccurrences,omitempty"`
}

// SecurityImpact represents the security impact of an update
type SecurityImpact struct {
	Level                string              `json:"level"` // "none", "low", "medium", "high", "critical"
	VulnerabilitiesFixed []*Vulnerability    `json:"vulnerabilitiesFixed,omitempty"`
	CVEsAddressed        []string            `json:"cvesAddressed,omitempty"`
	SecurityScore        float64             `json:"securityScore"`
	RiskReduction        float64             `json:"riskReduction"`
	ComplianceImprovement map[string]string  `json:"complianceImprovement,omitempty"`
	RecommendedAction    string              `json:"recommendedAction"`
	UrgencyLevel         string              `json:"urgencyLevel"`
}