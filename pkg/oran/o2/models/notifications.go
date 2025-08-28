package o2models

import (
	"time"
)

// Notification and Event Type Models following O-RAN.WG6.O2ims-Interface-v01.01

// NotificationEventType represents a type of notification event
type NotificationEventType struct {
	EventTypeID string                 `json:"eventTypeId"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Schema      string                 `json:"schema,omitempty"`
	Category    string                 `json:"category,omitempty"`
	Severity    string                 `json:"severity,omitempty"`
	Extensions  map[string]interface{} `json:"extensions,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

// Alarm represents an alarm in the system
type Alarm struct {
	AlarmID        string                 `json:"alarmId"`
	ResourceID     string                 `json:"resourceId"`
	AlarmType      string                 `json:"alarmType"`
	Severity       string                 `json:"severity"`
	Status         string                 `json:"status"` // ACTIVE, CLEARED, ACKNOWLEDGED
	Message        string                 `json:"message"`
	Description    string                 `json:"description,omitempty"`
	Source         string                 `json:"source"`
	RaisedAt       time.Time              `json:"raisedAt"`
	ClearedAt      *time.Time             `json:"clearedAt,omitempty"`
	AcknowledgedAt *time.Time             `json:"acknowledgedAt,omitempty"`
	AcknowledgedBy string                 `json:"acknowledgedBy,omitempty"`
	AdditionalInfo map[string]interface{} `json:"additionalInfo,omitempty"`
}

// AlarmFilter defines filters for alarm queries
type AlarmFilter struct {
	AlarmIDs     []string          `json:"alarmIds,omitempty"`
	ResourceIDs  []string          `json:"resourceIds,omitempty"`
	AlarmTypes   []string          `json:"alarmTypes,omitempty"`
	Severities   []string          `json:"severities,omitempty"`
	Statuses     []string          `json:"statuses,omitempty"`
	Sources      []string          `json:"sources,omitempty"`
	RaisedAfter  *time.Time        `json:"raisedAfter,omitempty"`
	RaisedBefore *time.Time        `json:"raisedBefore,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	Limit        int               `json:"limit,omitempty"`
	Offset       int               `json:"offset,omitempty"`
	SortBy       string            `json:"sortBy,omitempty"`
	SortOrder    string            `json:"sortOrder,omitempty"`
}

// AlarmAcknowledgementRequest represents a request to acknowledge an alarm
type AlarmAcknowledgementRequest struct {
	AcknowledgedBy string                 `json:"acknowledgedBy"`
	Message        string                 `json:"message,omitempty"`
	Timestamp      time.Time              `json:"timestamp"`
	Extensions     map[string]interface{} `json:"extensions,omitempty"`
}

// AlarmClearRequest represents a request to clear an alarm
type AlarmClearRequest struct {
	ClearedBy  string                 `json:"clearedBy"`
	Message    string                 `json:"message,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// Constants for alarm management

const (
	// Alarm Statuses
	AlarmStatusActive       = "ACTIVE"
	AlarmStatusCleared      = "CLEARED"
	AlarmStatusAcknowledged = "ACKNOWLEDGED"

	// Alarm Severities
	AlarmSeverityCritical = "CRITICAL"
	AlarmSeverityMajor    = "MAJOR"
	AlarmSeverityMinor    = "MINOR"
	AlarmSeverityWarning  = "WARNING"
	AlarmSeverityInfo     = "INFO"

	// Alarm Types
	AlarmTypeEquipment     = "EQUIPMENT"
	AlarmTypeService       = "SERVICE"
	AlarmTypePerformance   = "PERFORMANCE"
	AlarmTypeSecurity      = "SECURITY"
	AlarmTypeConfiguration = "CONFIGURATION"
)
