package o1

import (
	"time"
)

// Missing types for fault_manager.go compilation

// AlarmRecord represents a basic alarm record
type AlarmRecord struct {
	AlarmID               string                 `json:"alarm_id"`
	ObjectClass           string                 `json:"object_class"`
	ObjectInstance        string                 `json:"object_instance,omitempty"`
	NotificationID        string                 `json:"notification_id,omitempty"`
	EventType             string                 `json:"event_type"`
	ProbableCause         string                 `json:"probable_cause"`
	PerceivedSeverity     string                 `json:"perceived_severity"`
	SpecificProblem       string                 `json:"specific_problem,omitempty"`
	AdditionalText        string                 `json:"additional_text,omitempty"`
	AlarmType             string                 `json:"alarm_type,omitempty"`
	AlarmState            string                 `json:"alarm_state"`
	AckState              string                 `json:"ack_state,omitempty"`
	AlarmRaisedTime       *time.Time             `json:"alarm_raised_time"`
	AlarmClearedTime      *time.Time             `json:"alarm_cleared_time,omitempty"`
	AckTime               *time.Time             `json:"ack_time,omitempty"`
	AdditionalInformation map[string]interface{} `json:"additional_information,omitempty"`
}

// AlarmHistoryRequest represents a request for alarm history
type AlarmHistoryRequest struct {
	ObjectClass string     `json:"object_class,omitempty"`
	StartTime   *time.Time `json:"start_time,omitempty"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	Severity    []string   `json:"severity,omitempty"`
	Limit       int        `json:"limit,omitempty"`
	Offset      int        `json:"offset,omitempty"`
}

// AlarmSubscription represents an alarm subscription
type AlarmSubscription struct {
	SubscriptionID string                 `json:"subscription_id"`
	ObjectClass    []string               `json:"object_class,omitempty"`
	EventType      []string               `json:"event_type,omitempty"`
	Severity       []string               `json:"severity,omitempty"`
	Filters        map[string]interface{} `json:"filters,omitempty"`
	NotifyURI      string                 `json:"notify_uri,omitempty"`
}

// CorrelationRule represents a correlation rule
type CorrelationRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Pattern     map[string]interface{} `json:"pattern"`
	Action      string                 `json:"action"`
	TimeWindow  time.Duration          `json:"time_window"`
	Enabled     bool                   `json:"enabled"`
	Description string                 `json:"description,omitempty"`
}

// AlarmNotificationChannel represents a notification channel for alarms (renamed to avoid conflict)
type AlarmNotificationChannel interface {
	Send(alarm *EnhancedAlarm) error
	IsHealthy() bool
	GetChannelType() string
}

// WeaviatePooledConnection represents a pooled Weaviate connection
type WeaviatePooledConnection struct {
	ID         string    `json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
	UsageCount int64     `json:"usage_count"`
	IsHealthy  bool      `json:"is_healthy"`
}

// PoolMetrics represents connection pool metrics
type PoolMetrics struct {
	ActiveConnections int64         `json:"active_connections"`
	IdleConnections   int64         `json:"idle_connections"`
	TotalConnections  int64         `json:"total_connections"`
	AverageWaitTime   time.Duration `json:"average_wait_time"`
	ConnectionErrors  int64         `json:"connection_errors"`
}
