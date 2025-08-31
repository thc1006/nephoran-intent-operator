package o1

import (
	"time"
)

// Performance measurement type constants
const (
	PerfTypeRRCConnections = "RRC_CONNECTIONS"
	PerfTypeThroughput     = "THROUGHPUT"
)

// Object type constants
const (
	ObjectTypeCell = "CELL"
	ObjectTypeODU  = "ODU"
)

// AlarmSeverity represents standardized alarm severity levels
type AlarmSeverity string

const (
	AlarmSeverityMajor    AlarmSeverity = "MAJOR"
	AlarmSeverityMinor    AlarmSeverity = "MINOR"
	AlarmSeverityWarning  AlarmSeverity = "WARNING"
	AlarmSeverityCritical AlarmSeverity = "CRITICAL"
)

// Alarm represents a comprehensive O-RAN alarm
type Alarm struct {
	ID                string                 `json:"id"`
	AlarmID           string                 `json:"alarm_id"`
	Type              string                 `json:"type"`
	AlarmType         string                 `json:"alarm_type"`
	Severity          AlarmSeverity          `json:"severity"`
	ObjectType        string                 `json:"object_type,omitempty"`
	ObjectInstance    string                 `json:"object_instance,omitempty"`
	ManagedObjectID   string                 `json:"managed_object_id"`
	ManagedElementID  string                 `json:"managed_element_id"`
	SpecificProblem   string                 `json:"specific_problem"`
	AlarmText         string                 `json:"alarm_text"`
	PerceivedSeverity string                 `json:"perceived_severity"`
	ProbableCause     string                 `json:"probable_cause"`
	EventTime         time.Time              `json:"event_time"`
	NotificationTime  time.Time              `json:"notification_time,omitempty"`
	AdditionalText    string                 `json:"additional_text,omitempty"`
	AdditionalInfo    map[string]interface{} `json:"additional_info,omitempty"`
	AlarmState        string                 `json:"alarm_state,omitempty"`
	Acknowledged      bool                   `json:"acknowledged,omitempty"`
	AcknowledgedBy    string                 `json:"acknowledged_by,omitempty"`
	AckTime           *time.Time             `json:"ack_time,omitempty"`
	ClearedBy         string                 `json:"cleared_by,omitempty"`
	ClearTime         *time.Time             `json:"clear_time,omitempty"`
	RootCauseAlarms   []string               `json:"root_cause_alarms,omitempty"`
	CorrelatedAlarms  []string               `json:"correlated_alarms,omitempty"`
	RepairActions     []string               `json:"repair_actions,omitempty"`
}

// PerformanceDataPoint represents a single performance measurement point
type PerformanceDataPoint struct {
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status,omitempty"`
}

// PerformanceData represents a standardized performance measurement
type PerformanceData struct {
	ObjectInstance    string                 `json:"object_instance"`
	MeasurementType   string                 `json:"measurement_type"`
	Value             float64                `json:"value"`
	Unit              string                 `json:"unit"`
	Timestamp         time.Time              `json:"timestamp"`
	Granularity       time.Duration          `json:"granularity,omitempty"`
	Source            string                 `json:"source,omitempty"`
	Type              string                 `json:"type,omitempty"`
	DataPoints        []PerformanceDataPoint `json:"data_points,omitempty"`
	AdditionalDetails map[string]interface{} `json:"additional_details,omitempty"`
}

// NetworkFunction represents a network function in the O-RAN architecture
type NetworkFunction struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Type             string                 `json:"type"`
	Vendor           string                 `json:"vendor"`
	Version          string                 `json:"version"`
	Status           string                 `json:"status"`
	Configuration    map[string]interface{} `json:"configuration,omitempty"`
	OperationalState string                 `json:"operational_state"`
	AdditionalInfo   map[string]interface{} `json:"additional_info,omitempty"`
}

// NetworkFunctionStatus provides detailed status of a network function
type NetworkFunctionStatus struct {
	ID               string                 `json:"id"`
	HealthStatus     string                 `json:"health_status"`
	Alarms           []Alarm                `json:"alarms,omitempty"`
	Performance      []PerformanceData      `json:"performance,omitempty"`
	LastUpdated      time.Time              `json:"last_updated"`
	AdditionalStatus map[string]interface{} `json:"additional_status,omitempty"`
}

// NetworkFunctionUpdate represents updates to a network function
type NetworkFunctionUpdate struct {
	Name             *string                `json:"name,omitempty"`
	Configuration    map[string]interface{} `json:"configuration,omitempty"`
	OperationalState *string                `json:"operational_state,omitempty"`
}

// DiscoveryCriteria defines parameters for network function discovery
type DiscoveryCriteria struct {
	Type             []string `json:"type,omitempty"`
	Vendor           []string `json:"vendor,omitempty"`
	Status           []string `json:"status,omitempty"`
	OperationalState []string `json:"operational_state,omitempty"`
}

// CorrelationRule defines a rule for correlating alarms or events
type CorrelationRule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Conditions  []string `json:"conditions"`
	Actions     []string `json:"actions"`
	Severity    string   `json:"severity,omitempty"`
}

// Notification represents a generic notification in the O-RAN system
type Notification struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
	Severity  string                 `json:"severity,omitempty"`
	Category  string                 `json:"category,omitempty"`
}

// FaultManagerConfig represents configuration for fault management
type FaultManagerConfig struct {
	MaxAlarmRetention    int           `json:"max_alarm_retention"`
	RetentionPeriod      time.Duration `json:"retention_period"`
	AlarmEscalationRules []string      `json:"alarm_escalation_rules,omitempty"`
	NotificationChannels []string      `json:"notification_channels,omitempty"`
}

// EnhancedAlarm extends the base Alarm with additional correlation and diagnostic information
type EnhancedAlarm struct {
	Alarm
	CorrelationID    string   `json:"correlation_id,omitempty"`
	DiagnosticHints  []string `json:"diagnostic_hints,omitempty"`
	RecommendedFixes []string `json:"recommended_fixes,omitempty"`
}

// AlarmHistoryEntry represents a historical record of an alarm
type AlarmHistoryEntry struct {
	AlarmID        string                 `json:"alarm_id"`
	Timestamp      time.Time              `json:"timestamp"`
	State          string                 `json:"state"`
	PreviousState  string                 `json:"previous_state,omitempty"`
	AdditionalInfo map[string]interface{} `json:"additional_info,omitempty"`
}

// AlarmCorrelationEngine provides advanced alarm correlation and analysis capabilities
type AlarmCorrelationEngine struct {
	Rules               []CorrelationRule `json:"rules"`
	ActiveCorrelations  []string          `json:"active_correlations"`
	LastProcessedAlarms []string          `json:"last_processed_alarms"`
}
