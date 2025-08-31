// Package o1 implements O-RAN O1 interface specification
package o1

import (
	"context"
	"time"
)

// Configuration Management Types
type ConfigurationManager interface {
	GetConfiguration(ctx context.Context, objectClass string, objectInstance string) (*ConfigResponse, error)
	SetConfiguration(ctx context.Context, request *ConfigRequest) (*ConfigResponse, error)
	CreateManagedObject(ctx context.Context, request *ConfigRequest) (*ConfigResponse, error)
	DeleteManagedObject(ctx context.Context, objectClass string, objectInstance string) error
}

type ConfigRequest struct {
	ObjectClass    string                 `json:"objectClass"`
	ObjectInstance string                 `json:"objectInstance"`
	Attributes     map[string]interface{} `json:"attributes"`
	Operation      string                 `json:"operation"` // "create", "update", "delete"
	RequestID      string                 `json:"requestId"`
	Timestamp      time.Time              `json:"timestamp"`
}

type ConfigResponse struct {
	ObjectClass     string                 `json:"objectClass"`
	ObjectInstance  string                 `json:"objectInstance"`
	Attributes      map[string]interface{} `json:"attributes"`
	Status          string                 `json:"status"`
	ErrorMessage    string                 `json:"errorMessage,omitempty"`
	RequestID       string                 `json:"requestId"`
	Timestamp       time.Time              `json:"timestamp"`
}

// Performance Management Types
type PerformanceManager interface {
	GetPerformanceData(ctx context.Context, request *PerformanceRequest) (*PerformanceResponse, error)
	SubscribeToPerformanceData(ctx context.Context, subscription *PerformanceSubscription) error
	UnsubscribeFromPerformanceData(ctx context.Context, subscriptionID string) error
}

type PerformanceRequest struct {
	ObjectClass      string            `json:"objectClass"`
	ObjectInstances  []string          `json:"objectInstances"`
	PerformanceTypes []string          `json:"performanceTypes"`
	StartTime        time.Time         `json:"startTime"`
	EndTime          time.Time         `json:"endTime"`
	GranularityPeriod string           `json:"granularityPeriod"`
	ReportingPeriod  string            `json:"reportingPeriod"`
	RequestID        string            `json:"requestId"`
}

type PerformanceResponse struct {
	ObjectClass         string                       `json:"objectClass"`
	PerformanceData     []PerformanceMeasurement     `json:"performanceData"`
	Status              string                       `json:"status"`
	ErrorMessage        string                       `json:"errorMessage,omitempty"`
	RequestID           string                       `json:"requestId"`
	Timestamp           time.Time                    `json:"timestamp"`
}

type PerformanceMeasurement struct {
	ObjectInstance    string                 `json:"objectInstance"`
	MeasurementType   string                 `json:"measurementType"`
	Value             interface{}            `json:"value"`
	Unit              string                 `json:"unit"`
	Timestamp         time.Time              `json:"timestamp"`
	Attributes        map[string]interface{} `json:"attributes,omitempty"`
}

type PerformanceSubscription struct {
	SubscriptionID    string   `json:"subscriptionId"`
	ObjectClass       string   `json:"objectClass"`
	ObjectInstances   []string `json:"objectInstances"`
	PerformanceTypes  []string `json:"performanceTypes"`
	GranularityPeriod string   `json:"granularityPeriod"`
	ReportingPeriod   string   `json:"reportingPeriod"`
	CallbackURL       string   `json:"callbackUrl"`
	NotificationTypes []string `json:"notificationTypes"`
	CreatedAt         time.Time `json:"createdAt"`
	ExpiresAt         *time.Time `json:"expiresAt,omitempty"`
}

// NetworkFunctionConfig represents configuration for a network function (Original definition)
type NetworkFunctionConfig struct {
	ID            string                 `json:"id"`
	Parameters    map[string]interface{} `json:"parameters"`
	Policies      map[string]interface{} `json:"policies,omitempty"`
	ResourceLimits ResourceRequirements  `json:"resource_limits,omitempty"`
	ConfigVersion string                 `json:"config_version"`
	AppliedAt     time.Time              `json:"applied_at"`
}

// ResourceRequirements represents resource requirements
type ResourceRequirements struct {
	CPU          string                 `json:"cpu"`
	Memory       string                 `json:"memory"`
	Storage      string                 `json:"storage"`
	NetworkBW    string                 `json:"networkBw,omitempty"`
	GPUs         int                    `json:"gpus,omitempty"`
	Constraints  map[string]interface{} `json:"constraints,omitempty"`
}

// Alarm types
type Alarm struct {
	AlarmID           string                 `json:"alarmId"`
	ObjectClass       string                 `json:"objectClass"`
	ObjectInstance    string                 `json:"objectInstance"`
	EventType         string                 `json:"eventType"`
	ProbableCause     string                 `json:"probableCause"`
	SpecificProblem   string                 `json:"specificProblem"`
	PerceivedSeverity string                 `json:"perceivedSeverity"`
	AlarmRaisedTime   time.Time              `json:"alarmRaisedTime"`
	AlarmClearedTime  *time.Time             `json:"alarmClearedTime,omitempty"`
	AlarmChangedTime  *time.Time             `json:"alarmChangedTime,omitempty"`
	AckTime           *time.Time             `json:"ackTime,omitempty"`
	AckUserID         string                 `json:"ackUserId,omitempty"`
	AckSystemID       string                 `json:"ackSystemId,omitempty"`
	AckState          string                 `json:"ackState"`
	AlarmState        string                 `json:"alarmState"`
	AdditionalText    string                 `json:"additionalText,omitempty"`
	AdditionalInfo    map[string]interface{} `json:"additionalInfo,omitempty"`
}

// AlarmRecord represents an alarm record (alias for Alarm for compatibility)
type AlarmRecord = Alarm

// CorrelationRule represents a rule for alarm correlation
type CorrelationRule struct {
	RuleID          string                 `json:"ruleId"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description,omitempty"`
	SourceAlarms    []string               `json:"sourceAlarms"`
	TargetAlarm     string                 `json:"targetAlarm"`
	CorrelationType string                 `json:"correlationType"` // "cause-effect", "symptom-root", "temporal"
	TimeWindow      time.Duration          `json:"timeWindow"`
	Conditions      []CorrelationCondition `json:"conditions"`
	Action          string                 `json:"action"` // "suppress", "escalate", "merge"
	Priority        int                    `json:"priority"`
	Enabled         bool                   `json:"enabled"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// CorrelationCondition represents a condition for alarm correlation
type CorrelationCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // "equals", "contains", "matches"
	Value    interface{} `json:"value"`
	Weight   float64     `json:"weight"`
}

// Network Function Management Types
type NetworkFunction struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Type             string                 `json:"type"` // "CU", "DU", "RU", "AMF", "SMF", etc.
	Version          string                 `json:"version"`
	Vendor           string                 `json:"vendor"`
	State            string                 `json:"state"` // "active", "inactive", "failed"
	Endpoint         string                 `json:"endpoint"`
	Profile          string                 `json:"profile"`
	Capacity         ResourceRequirements   `json:"capacity"`
	LoadProfile      map[string]interface{} `json:"loadProfile,omitempty"`
	SupportedServices []string              `json:"supportedServices"`
	Dependencies     []string               `json:"dependencies,omitempty"`
	Configuration    *NetworkFunctionConfig `json:"configuration,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt        time.Time              `json:"createdAt"`
	UpdatedAt        time.Time              `json:"updatedAt"`
	LastHeartbeat    time.Time              `json:"lastHeartbeat"`
}

// NetworkFunctionUpdate represents updates to a network function
type NetworkFunctionUpdate struct {
	Name             *string                `json:"name,omitempty"`
	State            *string                `json:"state,omitempty"`
	Endpoint         *string                `json:"endpoint,omitempty"`
	Configuration    *NetworkFunctionConfig `json:"configuration,omitempty"`
	LoadProfile      map[string]interface{} `json:"loadProfile,omitempty"`
	SupportedServices []string              `json:"supportedServices,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// DiscoveryCriteria represents criteria for network function discovery
type DiscoveryCriteria struct {
	Type             string            `json:"type,omitempty"`
	Vendor           string            `json:"vendor,omitempty"`
	State            string            `json:"state,omitempty"`
	Location         string            `json:"location,omitempty"`
	Tags             map[string]string `json:"tags,omitempty"`
	MinCapacity      *ResourceRequirements `json:"minCapacity,omitempty"`
	SupportedServices []string         `json:"supportedServices,omitempty"`
}

// NetworkFunctionStatus represents the status of a network function
type NetworkFunctionStatus struct {
	ID               string                 `json:"id"`
	State            string                 `json:"state"`
	Health           string                 `json:"health"` // "healthy", "degraded", "unhealthy"
	LastUpdated      time.Time              `json:"lastUpdated"`
	Performance      map[string]interface{} `json:"performance,omitempty"`
	ResourceUsage    map[string]interface{} `json:"resourceUsage,omitempty"`
	ActiveConnections int                   `json:"activeConnections"`
	Uptime           time.Duration          `json:"uptime"`
	ErrorRate        float64                `json:"errorRate"`
	Throughput       map[string]float64     `json:"throughput,omitempty"`
	Latency          map[string]float64     `json:"latency,omitempty"`
	Alarms           []string               `json:"alarms,omitempty"`
	Details          map[string]interface{} `json:"details,omitempty"`
}

// Root Cause Analysis and Alarm Masking types are defined in fault_manager.go to avoid duplication

type RootCauseAnalysis struct {
	AlarmID            string                 `json:"alarmId"`
	RootCause          string                 `json:"rootCause"`
	ImpactedComponents []string               `json:"impactedComponents"`
	Severity           string                 `json:"severity"`
	Confidence         float64                `json:"confidence"`
	AnalysisTime       time.Time              `json:"analysisTime"`
	Details            map[string]interface{} `json:"details,omitempty"`
}

type CausalGraph struct {
	Nodes []CausalNode `json:"nodes"`
	Edges []CausalEdge `json:"edges"`
}

type CausalNode struct {
	ID              string                 `json:"id"`
	Type            string                 `json:"type"` // "alarm", "network-function", "infrastructure"
	Label           string                 `json:"label"`
	Attributes      map[string]interface{} `json:"attributes,omitempty"`
}

type CausalEdge struct {
	Source        string `json:"source"`
	Destination   string `json:"destination"`
	RelationType  string `json:"relationType"` // "causes", "impacts", "depends"
	Confidence    float64 `json:"confidence"`
}

type RemediationAction struct {
	ID          string                 `json:"id"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"` // "restart", "reconfigure", "migrate"
	Severity    string                 `json:"severity"`
	Impact      string                 `json:"impact"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Recommended bool                   `json:"recommended"`
}

// Additional missing types for O1 interface

// AlarmSubscription represents an alarm subscription
type AlarmSubscription struct {
	SubscriptionID string        `json:"subscriptionId"`
	ClientID       string        `json:"clientId"`
	NotifyURL      string        `json:"notifyUrl"`
	Filter         AlarmFilter   `json:"filter,omitempty"`
	CreatedAt      time.Time     `json:"createdAt"`
	ExpiresAt      *time.Time    `json:"expiresAt,omitempty"`
	Active         bool          `json:"active"`
}

// AlarmFilter represents alarm filtering criteria
type AlarmFilter struct {
	ObjectClass      []string `json:"objectClass,omitempty"`
	EventType        []string `json:"eventType,omitempty"`
	ProbableCause    []string `json:"probableCause,omitempty"`
	PerceivedSeverity []string `json:"perceivedSeverity,omitempty"`
}

// NotificationTemplate represents a notification template
type NotificationTemplate struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // "alarm", "performance", "config"
	Subject     string                 `json:"subject"`
	Body        string                 `json:"body"`
	Format      string                 `json:"format"` // "json", "xml", "plain"
	Variables   map[string]string      `json:"variables,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

// PerformanceThreshold represents performance thresholds
type PerformanceThreshold struct {
	ID               string                 `json:"id"`
	MetricName       string                 `json:"metricName"`
	ObjectClass      string                 `json:"objectClass"`
	ObjectInstance   string                 `json:"objectInstance,omitempty"`
	ThresholdType    string                 `json:"thresholdType"` // "upper", "lower"
	ThresholdValue   float64                `json:"thresholdValue"`
	Hysteresis       float64                `json:"hysteresis,omitempty"`
	Severity         string                 `json:"severity"`
	MonitoringState  string                 `json:"monitoringState"`
	ObservationPeriod time.Duration         `json:"observationPeriod"`
	NotificationTypes []string              `json:"notificationTypes"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// EscalationRule represents escalation rules for security incidents
type EscalationRule struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	TriggerType     string                 `json:"triggerType"` // "severity", "duration", "frequency"
	TriggerValue    interface{}            `json:"triggerValue"`
	EscalationLevel int                    `json:"escalationLevel"`
	Actions         []EscalationAction     `json:"actions"`
	Conditions      []EscalationCondition  `json:"conditions,omitempty"`
	Enabled         bool                   `json:"enabled"`
	Priority        int                    `json:"priority"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// EscalationAction represents an action in escalation
type EscalationAction struct {
	Type       string                 `json:"type"` // "notify", "execute", "escalate"
	Target     string                 `json:"target"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Delay      time.Duration          `json:"delay,omitempty"`
}

// EscalationCondition represents a condition for escalation
type EscalationCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// SMOIntegrationLayer represents the SMO integration layer
type SMOIntegrationLayer struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Endpoint      string                 `json:"endpoint"`
	Status        string                 `json:"status"`
	Version       string                 `json:"version"`
	Capabilities  []string               `json:"capabilities"`
	Configuration map[string]interface{} `json:"configuration"`
	LastHeartbeat time.Time              `json:"lastHeartbeat"`
	CreatedAt     time.Time              `json:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt"`
}

// Additional missing types for O1 client and SMO integration

// FunctionType represents network function type
type FunctionType string

const (
	FunctionTypeCU  FunctionType = "CU"
	FunctionTypeDU  FunctionType = "DU"
	FunctionTypeRU  FunctionType = "RU"
	FunctionTypeAMF FunctionType = "AMF"
	FunctionTypeSMF FunctionType = "SMF"
	FunctionTypeUPF FunctionType = "UPF"
)

// DeploymentTemplate represents deployment template
type DeploymentTemplate struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         FunctionType           `json:"type"`
	Version      string                 `json:"version"`
	Template     map[string]interface{} `json:"template"`
	Parameters   []TemplateParameter    `json:"parameters,omitempty"`
	Requirements ResourceRequirements   `json:"requirements"`
	CreatedAt    time.Time              `json:"createdAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
}

// TemplateParameter represents a template parameter
type TemplateParameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Description string      `json:"description,omitempty"`
}

// Client represents O1 client interface
type Client interface {
	Connect(ctx context.Context) error
	Disconnect() error
	IsConnected() bool
	GetConfiguration(ctx context.Context, req *ConfigRequest) (*ConfigResponse, error)
	SetConfiguration(ctx context.Context, req *ConfigRequest) (*ConfigResponse, error)
	GetPerformanceData(ctx context.Context, req *PerformanceRequest) (*PerformanceResponse, error)
	SubscribeToAlarms(ctx context.Context, subscription *AlarmSubscription) error
	UnsubscribeFromAlarms(ctx context.Context, subscriptionID string) error
	GetAlarms(ctx context.Context, filter *AlarmFilter) (*AlarmResponse, error)
}

// PerformanceData represents performance data
type PerformanceData struct {
	ObjectInstance string                 `json:"objectInstance"`
	Measurements   []PerformanceMeasurement `json:"measurements"`
	Timestamp      time.Time              `json:"timestamp"`
	CollectionID   string                 `json:"collectionId,omitempty"`
}

// AlarmResponse represents alarm response
type AlarmResponse struct {
	Alarms    []*Alarm `json:"alarms"`
	Total     int      `json:"total"`
	RequestID string   `json:"requestId"`
	Timestamp time.Time `json:"timestamp"`
}

// Additional missing types for O1 client implementation

// FileUploadRequest represents file upload request
type FileUploadRequest struct {
	FileName    string                 `json:"fileName"`
	FileType    string                 `json:"fileType"`
	Content     []byte                 `json:"content"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	RequestID   string                 `json:"requestId"`
	Timestamp   time.Time              `json:"timestamp"`
}

// FileUploadResponse represents file upload response
type FileUploadResponse struct {
	FileID      string    `json:"fileId"`
	Status      string    `json:"status"`
	Message     string    `json:"message,omitempty"`
	RequestID   string    `json:"requestId"`
	Timestamp   time.Time `json:"timestamp"`
}

// FileDownloadResponse represents file download response
type FileDownloadResponse struct {
	FileID      string                 `json:"fileId"`
	FileName    string                 `json:"fileName"`
	Content     []byte                 `json:"content"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Status      string                 `json:"status"`
	RequestID   string                 `json:"requestId"`
	Timestamp   time.Time              `json:"timestamp"`
}

// HeartbeatResponse represents heartbeat response
type HeartbeatResponse struct {
	Status      string    `json:"status"`
	Timestamp   time.Time `json:"timestamp"`
	RequestID   string    `json:"requestId"`
	ServerTime  time.Time `json:"serverTime"`
}

// AlarmHistoryRequest represents alarm history request
type AlarmHistoryRequest struct {
	AlarmID     string     `json:"alarmId,omitempty"`
	StartTime   time.Time  `json:"startTime"`
	EndTime     time.Time  `json:"endTime"`
	Filter      *AlarmFilter `json:"filter,omitempty"`
	Limit       int        `json:"limit,omitempty"`
	Offset      int        `json:"offset,omitempty"`
	RequestID   string     `json:"requestId"`
}