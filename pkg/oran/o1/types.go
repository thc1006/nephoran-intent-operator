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
	ID                 string                 `json:"id"`
	AlarmID            string                 `json:"alarm_id"`
	Type               string                 `json:"type"`
	AlarmType          string                 `json:"alarm_type"`
	Severity           AlarmSeverity          `json:"severity"`
	ObjectType         string                 `json:"object_type,omitempty"`
	ObjectInstance     string                 `json:"object_instance,omitempty"`
	ManagedObjectID    string                 `json:"managed_object_id"`
	ManagedElementID   string                 `json:"managed_element_id"`
	SpecificProblem    string                 `json:"specific_problem"`
	AlarmText          string                 `json:"alarm_text"`
	PerceivedSeverity  string                 `json:"perceived_severity"`
	ProbableCause      string                 `json:"probable_cause"`
	EventTime          time.Time              `json:"event_time"`
	NotificationTime   time.Time              `json:"notification_time,omitempty"`
	AdditionalText     string                 `json:"additional_text,omitempty"`
	AdditionalInfo     map[string]interface{} `json:"additional_info,omitempty"`
	AlarmState         string                 `json:"alarm_state,omitempty"`
	Acknowledged       bool                   `json:"acknowledged,omitempty"`
	AcknowledgedBy     string                 `json:"acknowledged_by,omitempty"`
	AckTime            *time.Time             `json:"ack_time,omitempty"`
	ClearedBy          string                 `json:"cleared_by,omitempty"`
	ClearTime          *time.Time             `json:"clear_time,omitempty"`
	RootCauseAlarms    []string               `json:"root_cause_alarms,omitempty"`
	CorrelatedAlarms   []string               `json:"correlated_alarms,omitempty"`
	RepairActions      []string               `json:"repair_actions,omitempty"`
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

// NetworkFunctionUpdate represents updates to a network function
type NetworkFunctionUpdate struct {
	Name             *string                `json:"name,omitempty"`
	Configuration    map[string]interface{} `json:"configuration,omitempty"`
	OperationalState *string                `json:"operational_state,omitempty"`
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

// DiscoveryCriteria defines parameters for network function discovery
type DiscoveryCriteria struct {
	Type             []string `json:"type,omitempty"`
	Vendor           []string `json:"vendor,omitempty"`
	Status           []string `json:"status,omitempty"`
	OperationalState []string `json:"operational_state,omitempty"`
}

// Include the rest of the existing type definitions from the original file