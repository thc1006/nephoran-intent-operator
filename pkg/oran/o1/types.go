package o1

import (
	"context"
	"time"
)

// Interface represents the O-RAN O1 Management Interface
type Interface interface {
	// Configuration Management
	GetConfig(ctx context.Context, path string) (*ConfigResponse, error)
	SetConfig(ctx context.Context, config *ConfigRequest) (*ConfigResponse, error)
	
	// Performance Management
	GetPerformanceData(ctx context.Context, request *PerformanceRequest) (*PerformanceResponse, error)
	SubscribePerformanceData(ctx context.Context, subscription *PerformanceSubscription) (<-chan *PerformanceData, error)
	
	// Fault Management
	GetAlarms(ctx context.Context, filter *AlarmFilter) (*AlarmResponse, error)
	SubscribeAlarms(ctx context.Context, subscription *AlarmSubscription) (<-chan *Alarm, error)
	AcknowledgeAlarm(ctx context.Context, alarmID string) error
	
	// File Management
	UploadFile(ctx context.Context, file *FileUploadRequest) (*FileUploadResponse, error)
	DownloadFile(ctx context.Context, fileID string) (*FileDownloadResponse, error)
	
	// Heartbeat
	SendHeartbeat(ctx context.Context) (*HeartbeatResponse, error)
}

// ConfigRequest represents a configuration request
type ConfigRequest struct {
	Path      string                 `json:"path"`
	Operation string                 `json:"operation"` // "get", "set", "delete"
	Data      map[string]interface{} `json:"data,omitempty"`
	Format    string                 `json:"format"` // "json", "xml"
}

// ConfigResponse represents a configuration response
type ConfigResponse struct {
	Path      string                 `json:"path"`
	Data      map[string]interface{} `json:"data"`
	Status    string                 `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// PerformanceRequest represents a performance data request
type PerformanceRequest struct {
	MeasurementTypes []string  `json:"measurement_types"`
	ObjectInstances  []string  `json:"object_instances,omitempty"`
	StartTime        time.Time `json:"start_time"`
	EndTime          time.Time `json:"end_time"`
	Granularity      int       `json:"granularity"` // in seconds
}

// PerformanceResponse represents a performance data response
type PerformanceResponse struct {
	RequestID string            `json:"request_id"`
	Data      []PerformanceData `json:"data"`
	Status    string            `json:"status"`
	Message   string            `json:"message,omitempty"`
}

// PerformanceData represents performance measurement data
type PerformanceData struct {
	ObjectInstance   string                 `json:"object_instance"`
	MeasurementType  string                 `json:"measurement_type"`
	Value            float64                `json:"value"`
	Unit             string                 `json:"unit"`
	Timestamp        time.Time              `json:"timestamp"`
	AdditionalFields map[string]interface{} `json:"additional_fields,omitempty"`
}

// PerformanceSubscription represents a performance data subscription
type PerformanceSubscription struct {
	SubscriptionID   string   `json:"subscription_id"`
	MeasurementTypes []string `json:"measurement_types"`
	ObjectInstances  []string `json:"object_instances,omitempty"`
	ReportingPeriod  int      `json:"reporting_period"` // in seconds
	NotificationURI  string   `json:"notification_uri"`
}

// AlarmFilter represents filtering criteria for alarms
type AlarmFilter struct {
	AlarmTypes    []string  `json:"alarm_types,omitempty"`
	Severities    []string  `json:"severities,omitempty"`
	ObjectTypes   []string  `json:"object_types,omitempty"`
	StartTime     time.Time `json:"start_time,omitempty"`
	EndTime       time.Time `json:"end_time,omitempty"`
	Acknowledged  *bool     `json:"acknowledged,omitempty"`
	Limit         int       `json:"limit,omitempty"`
	Offset        int       `json:"offset,omitempty"`
}

// AlarmResponse represents an alarm response
type AlarmResponse struct {
	Alarms     []Alarm `json:"alarms"`
	TotalCount int     `json:"total_count"`
	Status     string  `json:"status"`
	Message    string  `json:"message,omitempty"`
}

// Alarm represents an O-RAN alarm
type Alarm struct {
	AlarmID         string                 `json:"alarm_id"`
	AlarmType       string                 `json:"alarm_type"`
	ObjectType      string                 `json:"object_type"`
	ObjectInstance  string                 `json:"object_instance"`
	Severity        string                 `json:"severity"` // "critical", "major", "minor", "warning"
	ProbableCause   string                 `json:"probable_cause"`
	SpecificProblem string                 `json:"specific_problem"`
	AlarmText       string                 `json:"alarm_text"`
	EventTime       time.Time              `json:"event_time"`
	Acknowledged    bool                   `json:"acknowledged"`
	ClearedTime     *time.Time             `json:"cleared_time,omitempty"`
	AdditionalInfo  map[string]interface{} `json:"additional_info,omitempty"`
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
	FileType  string            `json:"file_type"`
	FileSize  int64             `json:"file_size"`
	Content   []byte            `json:"content"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Status    string            `json:"status"`
	Message   string            `json:"message,omitempty"`
}

// HeartbeatResponse represents a heartbeat response
type HeartbeatResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message,omitempty"`
}

// NotificationEvent represents a notification event from O1
type NotificationEvent struct {
	EventType   string                 `json:"event_type"` // "alarm", "performance", "config_change"
	EventID     string                 `json:"event_id"`
	ObjectType  string                 `json:"object_type"`
	ObjectID    string                 `json:"object_id"`
	EventTime   time.Time              `json:"event_time"`
	EventData   map[string]interface{} `json:"event_data"`
	Severity    string                 `json:"severity,omitempty"`
	Description string                 `json:"description,omitempty"`
}

// Client represents an O1 interface client
type Client struct {
	endpoint   string
	httpClient interface{} // HTTP client for REST API calls
	timeout    time.Duration
	headers    map[string]string
}

// ClientConfig represents O1 client configuration
type ClientConfig struct {
	Endpoint        string            `json:"endpoint"`
	Username        string            `json:"username,omitempty"`
	Password        string            `json:"password,omitempty"`
	CertFile        string            `json:"cert_file,omitempty"`
	KeyFile         string            `json:"key_file,omitempty"`
	CAFile          string            `json:"ca_file,omitempty"`
	Timeout         time.Duration     `json:"timeout"`
	Headers         map[string]string `json:"headers,omitempty"`
	InsecureSkipTLS bool              `json:"insecure_skip_tls"`
}

// NewClient creates a new O1 interface client
func NewClient(config *ClientConfig) Interface {
	if config == nil {
		config = &ClientConfig{
			Endpoint: "http://localhost:8080",
			Timeout:  30 * time.Second,
			Headers:  make(map[string]string),
		}
	}

	return &Client{
		endpoint: config.Endpoint,
		timeout:  config.Timeout,
		headers:  config.Headers,
	}
}

// Common error types for O1 interface
type ErrorType string

const (
	ErrorTypeInvalidRequest   ErrorType = "invalid_request"
	ErrorTypeUnauthorized     ErrorType = "unauthorized"
	ErrorTypeNotFound         ErrorType = "not_found"
	ErrorTypeInternalError    ErrorType = "internal_error"
	ErrorTypeTimeout          ErrorType = "timeout"
	ErrorTypeUnsupported      ErrorType = "unsupported_operation"
)

// O1Error represents an O1 interface error
type O1Error struct {
	Type    ErrorType `json:"error_type"`
	Message string    `json:"message"`
	Code    int       `json:"code,omitempty"`
	Details string    `json:"details,omitempty"`
}

func (e *O1Error) Error() string {
	return e.Message
}

// Constants for O-RAN O1 interface
const (
	// Configuration paths
	ConfigPathRAN         = "/ran"
	ConfigPathCellular    = "/cellular"
	ConfigPathInterfaces  = "/interfaces"
	ConfigPathPerformance = "/performance"
	
	// Alarm severities
	AlarmSeverityCritical = "critical"
	AlarmSeverityMajor    = "major"
	AlarmSeverityMinor    = "minor"
	AlarmSeverityWarning  = "warning"
	
	// Performance measurement types
	PerfTypeRRCConnections  = "rrc_connections"
	PerfTypeThroughput      = "throughput"
	PerfTypeLatency         = "latency"
	PerfTypePacketLoss      = "packet_loss"
	PerfTypeCPUUtilization  = "cpu_utilization"
	PerfTypeMemoryUtilization = "memory_utilization"
	
	// Object types
	ObjectTypeNearRTRIC    = "near_rt_ric"
	ObjectTypeODU          = "o_du"
	ObjectTypeOCU          = "o_cu"
	ObjectTypeCell         = "cell"
	ObjectTypeUE           = "ue"
)