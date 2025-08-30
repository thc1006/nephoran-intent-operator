package models

import (
	"time"
)

// Resource State Constants
const (
	ResourceStatePending    = "PENDING"
	ResourceStateActive     = "ACTIVE"
	ResourceStateInactive   = "INACTIVE"
	ResourceStateDeleting   = "DELETING"
	ResourceStateDeleted    = "DELETED"
	ResourceStateError      = "ERROR"
)

// Operational State Constants
const (
	OpStateEnabled  = "ENABLED"
	OpStateDisabled = "DISABLED"
)

// Administrative State Constants
const (
	AdminStateLocked   = "LOCKED"
	AdminStateUnlocked = "UNLOCKED"
)

// Usage State Constants
const (
	UsageStateIdle   = "IDLE"
	UsageStateActive = "ACTIVE"
	UsageStateBusy   = "BUSY"
)

// Health State Constants
const (
	HealthStateHealthy   = "HEALTHY"
	HealthStateUnhealthy = "UNHEALTHY"
	HealthStateUnknown   = "UNKNOWN"
)

// Event type constants for notifications (moved from duplicated type)
const (
	NotificationEventAlarmNew       = "AlarmNew"
	NotificationEventAlarmUpdate    = "AlarmUpdate"
	NotificationEventAlarmClear     = "AlarmClear"
	NotificationEventResourceChange = "ResourceChange"
	NotificationEventHealthChange   = "HealthChange"
)

// O2-specific alarm structure (compatible with O-RAN specs)
type O2Alarm struct {
	AlarmID               string                 `json:"alarmId"`
	ResourceID            string                 `json:"resourceId"`
	AlarmType             string                 `json:"alarmType"`
	ProbableCause         string                 `json:"probableCause"`
	PerceivedSeverity     string                 `json:"perceivedSeverity"` // Critical, Major, Minor, Warning, Indeterminate, Cleared
	AlarmRaisedTime       time.Time              `json:"alarmRaisedTime"`
	AlarmChangedTime      time.Time              `json:"alarmChangedTime,omitempty"`
	AlarmClearedTime      *time.Time             `json:"alarmClearedTime,omitempty"`
	AlarmAcknowledged     bool                   `json:"alarmAcknowledged"`
	AlarmAckTime          *time.Time             `json:"alarmAckTime,omitempty"`
	AlarmAckUser          string                 `json:"alarmAckUser,omitempty"`
	ProposedRepairActions []string               `json:"proposedRepairActions,omitempty"`
	AdditionalText        string                 `json:"additionalText,omitempty"`
	AdditionalInfo        map[string]interface{} `json:"additionalInfo,omitempty"`
	RootCauseAlarmIDs     []string               `json:"rootCauseAlarmIds,omitempty"`
}

// O2AlarmFilter represents filters for querying O2 alarms
type O2AlarmFilter struct {
	ResourceID        string    `json:"resourceId,omitempty"`
	AlarmType         string    `json:"alarmType,omitempty"`
	PerceivedSeverity []string  `json:"perceivedSeverity,omitempty"`
	StartTime         time.Time `json:"startTime,omitempty"`
	EndTime           time.Time `json:"endTime,omitempty"`
	AcknowledgedState *bool     `json:"acknowledgedState,omitempty"`
}

// O2AlarmAcknowledgementRequest represents a request to acknowledge an O2 alarm
type O2AlarmAcknowledgementRequest struct {
	AlarmID string `json:"alarmId"`
	AckUser string `json:"ackUser"`
	AckNote string `json:"ackNote,omitempty"`
}

// O2AlarmClearRequest represents a request to clear an O2 alarm
type O2AlarmClearRequest struct {
	AlarmID   string `json:"alarmId"`
	ClearUser string `json:"clearUser"`
	ClearNote string `json:"clearNote,omitempty"`
}

// HealthStatus represents the health status of a service or resource
type HealthStatus struct {
	Status      string                 `json:"status"`
	Message     string                 `json:"message,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Component   string                 `json:"component,omitempty"`
	ChecksPassed int                   `json:"checksPassed,omitempty"`
	ChecksFailed int                   `json:"checksFailed,omitempty"`
}

// APIInfo represents API version and metadata information
type APIInfo struct {
	Version     string                 `json:"version"`
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description,omitempty"`
	Contact     *APIContact            `json:"contact,omitempty"`
	License     *APILicense            `json:"license,omitempty"`
	BuildInfo   *APIBuildInfo          `json:"buildInfo,omitempty"`
	Extensions  map[string]interface{} `json:"extensions,omitempty"`
}

// APIContact represents contact information for the API
type APIContact struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	URL   string `json:"url,omitempty"`
}

// APILicense represents license information for the API
type APILicense struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// APIBuildInfo represents build information for the API
type APIBuildInfo struct {
	Version   string    `json:"version"`
	Commit    string    `json:"commit,omitempty"`
	BuildTime time.Time `json:"buildTime,omitempty"`
	GoVersion string    `json:"goVersion,omitempty"`
}

// Request types for various API operations
type CreateResourceTypeRequest struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	Category     string                 `json:"category,omitempty"`
	Version      string                 `json:"version,omitempty"`
	Vendor       string                 `json:"vendor,omitempty"`
	Specification map[string]interface{} `json:"specification"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
	Tags         map[string]string      `json:"tags,omitempty"`
}

type UpdateResourceTypeRequest struct {
	Description   *string                `json:"description,omitempty"`
	Category      *string                `json:"category,omitempty"`
	Version       *string                `json:"version,omitempty"`
	Vendor        *string                `json:"vendor,omitempty"`
	Specification map[string]interface{} `json:"specification,omitempty"`
	Properties    map[string]interface{} `json:"properties,omitempty"`
	Tags          map[string]string      `json:"tags,omitempty"`
}

// MetricsFilter for filtering metrics queries
type MetricsFilter struct {
	ResourceID  string    `json:"resourceId,omitempty"`
	MetricNames []string  `json:"metricNames,omitempty"`
	StartTime   time.Time `json:"startTime,omitempty"`
	EndTime     time.Time `json:"endTime,omitempty"`
	Interval    string    `json:"interval,omitempty"`
	Aggregation string    `json:"aggregation,omitempty"`
}

// CloudProvider represents a cloud provider configuration
type CloudProvider struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Type           string                 `json:"type"` // AWS, Azure, GCP, OpenStack, VMware
	Region         string                 `json:"region,omitempty"`
	Endpoint       string                 `json:"endpoint,omitempty"`
	Credentials    map[string]string      `json:"credentials"`
	Configuration  map[string]interface{} `json:"configuration,omitempty"`
	Capabilities   []string               `json:"capabilities"`
	Status         string                 `json:"status"`
	LastSync       time.Time              `json:"lastSync,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}