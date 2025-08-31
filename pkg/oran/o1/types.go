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

// Fault Management Types
type FaultManager interface {
	GetCurrentAlarms(ctx context.Context, objectClass string) ([]*AlarmRecord, error)
	GetAlarmHistory(ctx context.Context, request *AlarmHistoryRequest) ([]*AlarmRecord, error)
	AcknowledgeAlarm(ctx context.Context, alarmID string) error
	ClearAlarm(ctx context.Context, alarmID string) error
	SubscribeToAlarms(ctx context.Context, subscription *AlarmSubscription) error
	UnsubscribeFromAlarms(ctx context.Context, subscriptionID string) error
}

type AlarmRecord struct {
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

type AlarmHistoryRequest struct {
	ObjectClass    string     `json:"objectClass"`
	ObjectInstance string     `json:"objectInstance,omitempty"`
	StartTime      time.Time  `json:"startTime"`
	EndTime        time.Time  `json:"endTime"`
	SeverityFilter []string   `json:"severityFilter,omitempty"`
	AlarmTypes     []string   `json:"alarmTypes,omitempty"`
	RequestID      string     `json:"requestId"`
}

type AlarmSubscription struct {
	SubscriptionID    string   `json:"subscriptionId"`
	ObjectClass       string   `json:"objectClass"`
	ObjectInstances   []string `json:"objectInstances,omitempty"`
	NotificationTypes []string `json:"notificationTypes"`
	CallbackURL       string   `json:"callbackUrl"`
	CreatedAt         time.Time `json:"createdAt"`
	ExpiresAt         *time.Time `json:"expiresAt,omitempty"`
}

// O1 Client interface
type Client interface {
	// Configuration management
	GetConfig(ctx context.Context, objectClass, objectInstance string) (*ConfigResponse, error)
	SetConfig(ctx context.Context, request *ConfigRequest) (*ConfigResponse, error)
	
	// Performance management
	GetPerformance(ctx context.Context, request *PerformanceRequest) (*PerformanceResponse, error)
	
	// Fault management
	GetAlarms(ctx context.Context, objectClass string) ([]*AlarmRecord, error)
	
	// Connection management
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	IsConnected() bool
	
	// Health check
	Health(ctx context.Context) error
}

// SMO Integration Layer types
type SMOIntegrationLayer interface {
	RegisterManagedElement(ctx context.Context, element *ManagedElement) error
	UnregisterManagedElement(ctx context.Context, elementID string) error
	GetManagedElements(ctx context.Context) ([]*ManagedElement, error)
	DeployNetworkFunction(ctx context.Context, deployment *FunctionDeployment) error
	ScaleNetworkFunction(ctx context.Context, scaling *FunctionScaling) error
}

type ManagedElement struct {
	ElementID           string                 `json:"elementId"`
	ElementType         string                 `json:"elementType"`
	VendorName          string                 `json:"vendorName"`
	UserLabel           string                 `json:"userLabel"`
	LocationName        string                 `json:"locationName"`
	SwVersion           string                 `json:"swVersion"`
	ManagedBy           string                 `json:"managedBy"`
	SupportedFunctions  []FunctionType         `json:"supportedFunctions"`
	ManagementInterface string                 `json:"managementInterface"`
	Status              string                 `json:"status"`
	Attributes          map[string]interface{} `json:"attributes"`
	CreatedAt           time.Time              `json:"createdAt"`
	UpdatedAt           time.Time              `json:"updatedAt"`
}

type FunctionType struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Vendor      string                 `json:"vendor"`
	Capabilities []string              `json:"capabilities"`
	Requirements map[string]interface{} `json:"requirements"`
}

type FunctionDeployment struct {
	DeploymentID    string                 `json:"deploymentId"`
	FunctionName    string                 `json:"functionName"`
	FunctionVersion string                 `json:"functionVersion"`
	TargetElements  []string               `json:"targetElements"`
	Template        DeploymentTemplate     `json:"template"`
	Parameters      map[string]interface{} `json:"parameters"`
	RequestedBy     string                 `json:"requestedBy"`
	Priority        int                    `json:"priority"`
	CreatedAt       time.Time              `json:"createdAt"`
}

type DeploymentTemplate struct {
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	Description  string                 `json:"description"`
	Vendor       string                 `json:"vendor"`
	Category     string                 `json:"category"`
	ChartURL     string                 `json:"chartUrl,omitempty"`
	ImageURL     string                 `json:"imageUrl,omitempty"`
	ConfigSchema map[string]interface{} `json:"configSchema"`
	Resources    ResourceRequirements   `json:"resources"`
	Dependencies []string               `json:"dependencies,omitempty"`
}

type ResourceRequirements struct {
	CPU          string                 `json:"cpu"`
	Memory       string                 `json:"memory"`
	Storage      string                 `json:"storage"`
	NetworkBW    string                 `json:"networkBw,omitempty"`
	GPUs         int                    `json:"gpus,omitempty"`
	Constraints  map[string]interface{} `json:"constraints,omitempty"`
}

type FunctionScaling struct {
	DeploymentID    string                 `json:"deploymentId"`
	ScalingType     string                 `json:"scalingType"` // "horizontal", "vertical"
	TargetReplicas  int                    `json:"targetReplicas,omitempty"`
	ResourceLimits  ResourceRequirements   `json:"resourceLimits,omitempty"`
	Triggers        []ScalingTrigger       `json:"triggers,omitempty"`
	RequestedBy     string                 `json:"requestedBy"`
	Priority        int                    `json:"priority"`
	CreatedAt       time.Time              `json:"createdAt"`
}

type ScalingTrigger struct {
	Type        string                 `json:"type"`       // "cpu", "memory", "custom"
	Threshold   float64                `json:"threshold"`
	Operator    string                 `json:"operator"`   // "gt", "lt", "avg"
	Duration    time.Duration          `json:"duration"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Network Slice Management
type NetworkSliceManager interface {
	CreateSlice(ctx context.Context, slice *NetworkSliceInstance) error
	ModifySlice(ctx context.Context, sliceID string, modifications *SliceModification) error
	DeleteSlice(ctx context.Context, sliceID string) error
	GetSlice(ctx context.Context, sliceID string) (*NetworkSliceInstance, error)
	ListSlices(ctx context.Context) ([]*NetworkSliceInstance, error)
	GetSliceStatus(ctx context.Context, sliceID string) (*SliceStatus, error)
}

type NetworkSliceInstance struct {
	SliceID              string                 `json:"sliceId"`
	SliceType            string                 `json:"sliceType"`          // "eMBB", "URLLC", "mMTC"
	ServiceProfile       ServiceProfile         `json:"serviceProfile"`
	SliceProfile         SliceProfile          `json:"sliceProfile"`
	Status               string                 `json:"status"`
	OperationalState     string                 `json:"operationalState"`
	AdministrativeState  string                 `json:"administrativeState"`
	NetworkFunctions     []string               `json:"networkFunctions"`
	CoverageArea         []CoverageAreaInfo     `json:"coverageArea"`
	CreatedAt            time.Time              `json:"createdAt"`
	ModifiedAt           time.Time              `json:"modifiedAt"`
	ExpiresAt            *time.Time             `json:"expiresAt,omitempty"`
	Attributes           map[string]interface{} `json:"attributes,omitempty"`
}

type ServiceProfile struct {
	ServiceProfileID   string  `json:"serviceProfileId"`
	MaxNumberOfUEs     int     `json:"maxNumberOfUEs"`
	DLLatency          int     `json:"dlLatency"`       // milliseconds
	ULLatency          int     `json:"ulLatency"`       // milliseconds
	DLThruputPerUE     int     `json:"dlThruputPerUE"`  // kbps
	ULThruputPerUE     int     `json:"ulThruputPerUE"`  // kbps
	DLThruputPerSlice  int     `json:"dlThruputPerSlice"` // kbps
	ULThruputPerSlice  int     `json:"ulThruputPerSlice"` // kbps
	MaxPktLossRate     float64 `json:"maxPktLossRate"`    // percentage
	Availability       float64 `json:"availability"`      // percentage
	Jitter             int     `json:"jitter"`            // milliseconds
	SurvivalTime       int     `json:"survivalTime"`      // milliseconds
	CSAvailability     float64 `json:"csAvailability"`    // percentage
	Reliability        float64 `json:"reliability"`       // percentage
	ExpDataRate        int     `json:"expDataRate"`       // kbps
	PayloadSize        int     `json:"payloadSize"`       // bytes
	TrafficDensity     int     `json:"trafficDensity"`    // per km²
	ConnDensity        int     `json:"connDensity"`       // per km²
}

type SliceProfile struct {
	SliceProfileID     string                 `json:"sliceProfileId"`
	PLMNInfoList       []PLMNInfo            `json:"plmnInfoList"`
	CNSliceSubnetProfile CNSliceSubnetProfile `json:"cnSliceSubnetProfile"`
	RANSliceSubnetProfile RANSliceSubnetProfile `json:"ranSliceSubnetProfile"`
}

type PLMNInfo struct {
	PLMNID   string   `json:"plmnId"`
	SNSSAIs  []SNSSAI `json:"snssais"`
}

type SNSSAI struct {
	SST int    `json:"sst"`
	SD  string `json:"sd,omitempty"`
}

type CNSliceSubnetProfile struct {
	CNSliceSubnetProfileID string                 `json:"cnSliceSubnetProfileId"`
	NetworkFunctions       []string               `json:"networkFunctions"`
	Attributes             map[string]interface{} `json:"attributes,omitempty"`
}

type RANSliceSubnetProfile struct {
	RANSliceSubnetProfileID string                 `json:"ranSliceSubnetProfileId"`
	CoverageAreaTAList      []int                  `json:"coverageAreaTAList"`
	RANNFRequirements       RANNFRequirements      `json:"ranNfRequirements"`
	Attributes              map[string]interface{} `json:"attributes,omitempty"`
}

type RANNFRequirements struct {
	AMFRegion     string `json:"amfRegion,omitempty"`
	AMFSet        string `json:"amfSet,omitempty"`
	AMFPointer    string `json:"amfPointer,omitempty"`
	SMFInstances  []string `json:"smfInstances,omitempty"`
	UPFInstances  []string `json:"upfInstances,omitempty"`
}

type CoverageAreaInfo struct {
	CoverageAreaID string                 `json:"coverageAreaId"`
	GeographicArea GeographicArea         `json:"geographicArea"`
	TACs           []int                  `json:"tacs"`
	Attributes     map[string]interface{} `json:"attributes,omitempty"`
}

type GeographicArea struct {
	Type         string    `json:"type"`         // "Point", "Polygon", "Ellipse"
	Coordinates  []float64 `json:"coordinates"`
	Radius       float64   `json:"radius,omitempty"`
	InnerRadius  float64   `json:"innerRadius,omitempty"`
	Attributes   map[string]interface{} `json:"attributes,omitempty"`
}

type SliceModification struct {
	ModificationType string                 `json:"modificationType"` // "add", "remove", "modify"
	ServiceProfile   *ServiceProfile        `json:"serviceProfile,omitempty"`
	SliceProfile     *SliceProfile         `json:"sliceProfile,omitempty"`
	CoverageArea     []CoverageAreaInfo     `json:"coverageArea,omitempty"`
	Attributes       map[string]interface{} `json:"attributes,omitempty"`
}

type SliceStatus struct {
	SliceID              string                 `json:"sliceId"`
	OperationalState     string                 `json:"operationalState"`
	AdministrativeState  string                 `json:"administrativeState"`
	ResourceUtilization  ResourceUtilization   `json:"resourceUtilization"`
	PerformanceMetrics   PerformanceMetrics    `json:"performanceMetrics"`
	ActiveSessions       int                    `json:"activeSessions"`
	ConnectedUEs         int                    `json:"connectedUes"`
	AlarmSummary         AlarmSummary          `json:"alarmSummary"`
	LastUpdated          time.Time              `json:"lastUpdated"`
}

type ResourceUtilization struct {
	CPUUtilization    float64 `json:"cpuUtilization"`    // percentage
	MemoryUtilization float64 `json:"memoryUtilization"` // percentage
	NetworkUtilization float64 `json:"networkUtilization"` // percentage
	StorageUtilization float64 `json:"storageUtilization"` // percentage
}

type PerformanceMetrics struct {
	Throughput    float64 `json:"throughput"`    // Mbps
	Latency       float64 `json:"latency"`       // milliseconds
	PacketLoss    float64 `json:"packetLoss"`    // percentage
	Availability  float64 `json:"availability"`  // percentage
	Reliability   float64 `json:"reliability"`   // percentage
	Jitter        float64 `json:"jitter"`        // milliseconds
}

type AlarmSummary struct {
	Critical int `json:"critical"`
	Major    int `json:"major"`
	Minor    int `json:"minor"`
	Warning  int `json:"warning"`
	Total    int `json:"total"`
}

// Event and Notification types for O1 interface
type NotificationManager interface {
	SendNotification(ctx context.Context, notification *O1Notification) error
	SubscribeToNotifications(ctx context.Context, subscription *NotificationSubscription) error
	UnsubscribeFromNotifications(ctx context.Context, subscriptionID string) error
	GetNotificationHistory(ctx context.Context, request *NotificationHistoryRequest) ([]*O1Notification, error)
}

type O1Notification struct {
	NotificationID   string                 `json:"notificationId"`
	NotificationType string                 `json:"notificationType"`
	ObjectClass      string                 `json:"objectClass"`
	ObjectInstance   string                 `json:"objectInstance"`
	EventTime        time.Time              `json:"eventTime"`
	SystemDN         string                 `json:"systemDN"`
	AttributeChanges []AttributeChange      `json:"attributeChanges,omitempty"`
	AlarmInformation *AlarmRecord           `json:"alarmInformation,omitempty"`
	StateChange      *StateChangeInfo       `json:"stateChange,omitempty"`
	AdditionalInfo   map[string]interface{} `json:"additionalInfo,omitempty"`
}

type AttributeChange struct {
	AttributeName string      `json:"attributeName"`
	OldValue      interface{} `json:"oldValue,omitempty"`
	NewValue      interface{} `json:"newValue"`
	ChangeType    string      `json:"changeType"` // "added", "removed", "modified"
}

type StateChangeInfo struct {
	StateType    string    `json:"stateType"`    // "operational", "administrative", "usage"
	OldState     string    `json:"oldState"`
	NewState     string    `json:"newState"`
	ChangeTime   time.Time `json:"changeTime"`
	ChangeReason string    `json:"changeReason,omitempty"`
}

type NotificationSubscription struct {
	SubscriptionID      string   `json:"subscriptionId"`
	NotificationTypes   []string `json:"notificationTypes"`
	ObjectClasses       []string `json:"objectClasses,omitempty"`
	ObjectInstances     []string `json:"objectInstances,omitempty"`
	AttributeNames      []string `json:"attributeNames,omitempty"`
	CallbackURL         string   `json:"callbackUrl"`
	AuthenticationInfo  string   `json:"authenticationInfo,omitempty"`
	CreatedAt           time.Time `json:"createdAt"`
	ExpiresAt           *time.Time `json:"expiresAt,omitempty"`
}

type NotificationHistoryRequest struct {
	StartTime         time.Time `json:"startTime"`
	EndTime           time.Time `json:"endTime"`
	NotificationTypes []string  `json:"notificationTypes,omitempty"`
	ObjectClasses     []string  `json:"objectClasses,omitempty"`
	ObjectInstances   []string  `json:"objectInstances,omitempty"`
	MaxResults        int       `json:"maxResults,omitempty"`
	RequestID         string    `json:"requestId"`
}

// AlarmResponse represents an alarm query response
type AlarmResponse struct {
	Alarms    []Alarm           `json:"alarms"`
	Status    string            `json:"status"`
	Message   string            `json:"message,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	Count     int               `json:"count"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// FileUploadRequest represents a file upload request
type FileUploadRequest struct {
	FileName    string            `json:"file_name"`
	FileType    string            `json:"file_type"`
	Content     []byte            `json:"content"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Description string            `json:"description,omitempty"`
}

// FileUploadResponse represents a file upload response
type FileUploadResponse struct {
	FileID    string            `json:"file_id"`
	FileName  string            `json:"file_name"`
	Status    string            `json:"status"`
	Message   string            `json:"message,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	Size      int64             `json:"size"`
	Checksum  string            `json:"checksum,omitempty"`
}

// FileDownloadResponse represents a file download response
type FileDownloadResponse struct {
	FileID    string            `json:"file_id"`
	FileName  string            `json:"file_name"`
	Content   []byte            `json:"content"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
}

// HeartbeatResponse represents a heartbeat response
type HeartbeatResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Message   string            `json:"message,omitempty"`
	Uptime    time.Duration     `json:"uptime,omitempty"`
	Version   string            `json:"version,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// CorrelationRule represents a rule for correlating events
type CorrelationRule struct {
	RuleID      string                 `json:"rule_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Conditions  []CorrelationCondition `json:"conditions"`
	Actions     []CorrelationAction    `json:"actions"`
	Enabled     bool                   `json:"enabled"`
	Priority    int                    `json:"priority"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// CorrelationCondition represents a condition in a correlation rule
type CorrelationCondition struct {
	Field     string      `json:"field"`
	Operator  string      `json:"operator"` // "eq", "ne", "gt", "lt", "contains", etc.
	Value     interface{} `json:"value"`
	Threshold int         `json:"threshold,omitempty"`
	TimeWindow string     `json:"time_window,omitempty"`
}

// CorrelationAction represents an action to take when rule matches
type CorrelationAction struct {
	Type       string                 `json:"type"` // "alert", "log", "callback", etc.
	Target     string                 `json:"target,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// NetworkFunction represents a network function in the O-RAN system
type NetworkFunction struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Type          string                 `json:"type"` // "CU", "DU", "RU", "AMF", "SMF", etc.
	Version       string                 `json:"version"`
	Vendor        string                 `json:"vendor"`
	Description   string                 `json:"description,omitempty"`
	Status        string                 `json:"status"` // "active", "inactive", "error"
	Capabilities  []string               `json:"capabilities"`
	Endpoints     []Endpoint             `json:"endpoints"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// NetworkFunctionUpdate represents updates to a network function
type NetworkFunctionUpdate struct {
	Name          *string                `json:"name,omitempty"`
	Version       *string                `json:"version,omitempty"`
	Status        *string                `json:"status,omitempty"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
	Capabilities  []string               `json:"capabilities,omitempty"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// DiscoveryCriteria represents criteria for discovering network functions
type DiscoveryCriteria struct {
	Type         string            `json:"type,omitempty"`
	Vendor       string            `json:"vendor,omitempty"`
	Status       string            `json:"status,omitempty"`
	Capabilities []string          `json:"capabilities,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	MaxResults   int               `json:"max_results,omitempty"`
}

// NetworkFunctionStatus represents the status of a network function
type NetworkFunctionStatus struct {
	ID               string                 `json:"id"`
	OperationalState string                 `json:"operational_state"` // "enabled", "disabled"
	AdminState       string                 `json:"admin_state"`       // "unlocked", "locked"
	UsageState       string                 `json:"usage_state"`       // "idle", "active", "busy"
	HealthStatus     string                 `json:"health_status"`     // "healthy", "unhealthy", "unknown"
	Alarms           []AlarmSummary         `json:"alarms,omitempty"`
	Metrics          map[string]interface{} `json:"metrics,omitempty"`
	LastUpdated      time.Time              `json:"last_updated"`
}

// NetworkFunctionConfig represents configuration for a network function
type NetworkFunctionConfig struct {
	ID            string                 `json:"id"`
	Parameters    map[string]interface{} `json:"parameters"`
	Policies      map[string]interface{} `json:"policies,omitempty"`
	ResourceLimits ResourceRequirements  `json:"resource_limits,omitempty"`
	ConfigVersion string                 `json:"config_version"`
	AppliedAt     time.Time              `json:"applied_at"`
}

// Endpoint represents a network endpoint
type Endpoint struct {
	Name     string            `json:"name"`
	Protocol string            `json:"protocol"` // "http", "grpc", "rest", etc.
	Address  string            `json:"address"`
	Port     int               `json:"port"`
	Path     string            `json:"path,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}