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

// Root Cause Analysis and Alarm Masking

type AlarmMaskingManager interface {
	CreateAlarmMask(ctx context.Context, mask *AlarmMask) error
	RemoveAlarmMask(ctx context.Context, maskID string) error
	ListAlarmMasks(ctx context.Context) ([]*AlarmMask, error)
	GetAlarmMask(ctx context.Context, maskID string) (*AlarmMask, error)
	UpdateAlarmMask(ctx context.Context, maskID string, mask *AlarmMask) error
}

type AlarmMask struct {
	ID              string                 `json:"id"`
	MaskType        string                 `json:"maskType"` // "temporary", "permanent"
	ProbableCause   string                 `json:"probableCause"`
	Severity        string                 `json:"severity"`
	Scope           string                 `json:"scope"` // "global", "specific"
	ObjectClass     string                 `json:"objectClass,omitempty"`
	ObjectInstance  string                 `json:"objectInstance,omitempty"`
	StartTime       time.Time              `json:"startTime"`
	EndTime         *time.Time             `json:"endTime,omitempty"`
	Reason          string                 `json:"reason"`
	CreatedBy       string                 `json:"createdBy"`
	CreatedAt       time.Time              `json:"createdAt"`
	LastModified    time.Time              `json:"lastModified"`
	AdditionalInfo  map[string]interface{} `json:"additionalInfo,omitempty"`
}

type RootCauseAnalyzer interface {
	AnalyzeRootCause(ctx context.Context, alarmID string) (*RootCauseAnalysis, error)
	GenerateCausalGraph(ctx context.Context, events []O1Notification) (*CausalGraph, error)
	RecommendRemediation(ctx context.Context, analysis *RootCauseAnalysis) ([]RemediationAction, error)
}

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