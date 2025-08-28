package o1

import (
	"time"
)

// Performance measurement type constants
const (
	PerfTypeRRCConnections = "RRC_CONNECTIONS"
	PerfTypeThroughput    = "THROUGHPUT"
)

// Object type constants
const (
	ObjectTypeCell = "CELL"
	ObjectTypeODU  = "ODU"
)

// Alarm severity constants
const (
	AlarmSeverityMajor   = "major"
	AlarmSeverityMinor   = "minor"
	AlarmSeverityWarning = "warning"
)

// Alarm represents an O-RAN alarm with all required fields
type Alarm struct {
	AlarmID           string                 `json:"alarm_id"`
	AlarmType         string                 `json:"alarm_type"`
	ObjectType        string                 `json:"object_type"`
	ObjectInstance    string                 `json:"object_instance"`
	ManagedObjectID   string                 `json:"managed_object_id"`    // Added missing field
	Severity          string                 `json:"severity"`             // "critical", "major", "minor", "warning"
	PerceivedSeverity string                 `json:"perceived_severity"`   // Added missing field - O-RAN specific severity
	ProbableCause     string                 `json:"probable_cause"`
	SpecificProblem   string                 `json:"specific_problem"`
	AlarmText         string                 `json:"alarm_text"`
	AdditionalText    string                 `json:"additional_text"`      // Added missing field
	EventTime         time.Time              `json:"event_time"`
	Acknowledged      bool                   `json:"acknowledged"`
	ClearedTime       *time.Time             `json:"cleared_time,omitempty"`
	AdditionalInfo    map[string]interface{} `json:"additional_info,omitempty"`
}

// AlarmSubscription represents an alarm subscription
type AlarmSubscription struct {
	SubscriptionID  string   `json:"subscription_id"`
	AlarmTypes      []string `json:"alarm_types,omitempty"`
	Severities      []string `json:"severities,omitempty"`
	ObjectTypes     []string `json:"object_types,omitempty"`
	NotificationURI string   `json:"notification_uri"`
}

// FileUploadRequest represents a file upload request
type FileUploadRequest struct {
	FileName    string            `json:"file_name"`
	FileType    string            `json:"file_type"`
	FileSize    int64             `json:"file_size"`
	Content     []byte            `json:"content"`
	Destination string            `json:"destination"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// FileUploadResponse represents a file upload response
type FileUploadResponse struct {
	FileID    string    `json:"file_id"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// FileDownloadResponse represents a file download response
type FileDownloadResponse struct {
	FileID    string            `json:"file_id"`
	FileName  string            `json:"file_name"`
	FileType  string            `json:"file_type"`    // Added missing field
	FileSize  int64             `json:"file_size"`
	Content   []byte            `json:"content"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Status    string            `json:"status"`
	Message   string            `json:"message,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// FileTransferRequest represents a file transfer request for O1
type FileTransferRequest struct {
	RequestID   string `json:"request_id"`
	Operation   string `json:"operation"` // "upload" or "download"
	FileName    string `json:"file_name"`
	FileType    string `json:"file_type"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Priority    int    `json:"priority"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// FileTransferResponse represents the response to a file transfer request
type FileTransferResponse struct {
	RequestID string    `json:"request_id"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// NotificationEvent represents a generic notification event
type NotificationEvent struct {
	EventID     string                 `json:"event_id"`
	EventType   string                 `json:"event_type"`
	Source      string                 `json:"source"`
	Target      string                 `json:"target"`
	Timestamp   time.Time              `json:"timestamp"`
	Severity    string                 `json:"severity"`
	Message     string                 `json:"message"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

// Client represents an O1 client
type Client struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Endpoint    string                 `json:"endpoint"`
	Credentials map[string]interface{} `json:"credentials,omitempty"`
	Connected   bool                   `json:"connected"`
	LastSeen    time.Time              `json:"last_seen"`
}

// ConfigRequest represents a configuration request
type ConfigRequest struct {
	RequestID   string                 `json:"request_id"`
	ClientID    string                 `json:"client_id"`
	Operation   string                 `json:"operation"` // GET, SET, CREATE, DELETE
	Path        string                 `json:"path"`      // Added missing field
	Target      string                 `json:"target"`
	Format      string                 `json:"format,omitempty"`    // Added missing field
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Data        interface{}            `json:"data,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// ConfigResponse represents a configuration response
type ConfigResponse struct {
	RequestID string      `json:"request_id"`
	Path      string      `json:"path"`    // Added missing field
	Status    string      `json:"status"`  // SUCCESS, ERROR
	Message   string      `json:"message"` // Added missing field
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// PerformanceRequest represents a performance data request
type PerformanceRequest struct {
	RequestID        string            `json:"request_id"`
	ClientID         string            `json:"client_id"`
	MetricType       string            `json:"metric_type"`
	MeasurementTypes []string          `json:"measurement_types,omitempty"` // Added missing field
	StartTime        time.Time         `json:"start_time"`
	EndTime          time.Time         `json:"end_time"`
	Granularity      string            `json:"granularity,omitempty"`       // Added missing field - changed to string
	Filters          map[string]string `json:"filters,omitempty"`
	Timestamp        time.Time         `json:"timestamp"`
}

// PerformanceResponse represents a performance data response
type PerformanceResponse struct {
	RequestID string                   `json:"request_id"`
	Status    string                   `json:"status"`
	Data      []PerformanceData        `json:"data,omitempty"`  // Changed to use PerformanceData
	Error     string                   `json:"error,omitempty"`
	Timestamp time.Time                `json:"timestamp"`
}

// PerformanceDataPoint represents a single performance data point
type PerformanceDataPoint struct {
	MetricName  string                 `json:"metric_name"`
	Value       float64                `json:"value"`
	Unit        string                 `json:"unit"`
	Timestamp   time.Time              `json:"timestamp"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// PerformanceData represents performance measurement data with all required fields
type PerformanceData struct {
	ObjectInstance   string                 `json:"object_instance"`   // Added missing field
	MeasurementType  string                 `json:"measurement_type"`  // Added missing field  
	Value            float64                `json:"value"`             // Added missing field
	Unit             string                 `json:"unit"`              // Added missing field
	Timestamp        time.Time              `json:"timestamp"`
	Source           string                 `json:"source,omitempty"`
	Type             string                 `json:"type,omitempty"`
	DataPoints       []PerformanceDataPoint `json:"data_points,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// PerformanceSubscription represents a performance data subscription
type PerformanceSubscription struct {
	SubscriptionID   string            `json:"subscription_id"`
	MetricTypes      []string          `json:"metric_types"`
	Interval         time.Duration     `json:"interval"`
	ReportingPeriod  time.Duration     `json:"reporting_period"`  // Added missing field
	Filters          map[string]string `json:"filters,omitempty"`
	NotificationURI  string            `json:"notification_uri"`
	Active           bool              `json:"active"`
}

// AlarmResponse represents a response to alarm operations
type AlarmResponse struct {
	RequestID  string    `json:"request_id"`
	AlarmID    string    `json:"alarm_id,omitempty"`
	Status     string    `json:"status"`
	Message    string    `json:"message,omitempty"`
	Error      string    `json:"error,omitempty"`
	Alarms     []Alarm   `json:"alarms,omitempty"`     // Changed to []Alarm to match usage
	TotalCount int       `json:"total_count,omitempty"` // Added missing field
	Timestamp  time.Time `json:"timestamp"`
}

// HeartbeatResponse represents a response to heartbeat operations
type HeartbeatResponse struct {
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Uptime    time.Duration `json:"uptime"`
}