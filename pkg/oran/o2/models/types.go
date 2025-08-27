package models

import (
	"time"
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
