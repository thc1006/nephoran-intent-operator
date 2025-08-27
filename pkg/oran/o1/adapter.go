package o1

import (
	"context"
	"time"

	"github.com/go-logr/logr"
)

// O1Adapter implements the O-RAN O1 interface for management and orchestration
// This adapter provides comprehensive management capabilities including configuration,
// fault management, performance management, and software management following
// O-RAN.WG10.O1-Interface specifications
type O1Adapter struct {
	logger logr.Logger

	// Configuration management
	configManager   ConfigurationManager
	softwareManager SoftwareManager

	// Operational management
	faultManager       FaultManager
	performanceManager PerformanceManager

	// Network function management
	nfManager NetworkFunctionManager

	// Subscription management for notifications
	subscriptions map[string]*Subscription

	// Connection state
	connectionState ConnectionState
	lastHeartbeat   time.Time

	// Metrics and monitoring
	metrics         *O1Metrics
	notificationHub *NotificationHub
}

// ConnectionState represents the current state of O1 connections
type ConnectionState struct {
	State          string                 `json:"state"`
	ConnectedNFs   []string               `json:"connectedNFs"`
	ActiveSessions int                    `json:"activeSessions"`
	LastConnected  time.Time              `json:"lastConnected"`
	HealthStatus   string                 `json:"healthStatus"`
	Capabilities   []string               `json:"capabilities"`
	Configuration  map[string]interface{} `json:"configuration"`
}

// ConfigurationManager handles configuration management operations
type ConfigurationManager interface {
	// Configuration retrieval and updates
	GetConfiguration(ctx context.Context, target string) (map[string]interface{}, error)
	UpdateConfiguration(ctx context.Context, target string, config map[string]interface{}) error
	ValidateConfiguration(ctx context.Context, config map[string]interface{}) error

	// Schema and capability management
	GetConfigurationSchema(ctx context.Context, target string) ([]byte, error)
	GetSupportedCapabilities(ctx context.Context) ([]string, error)

	// Backup and restore
	BackupConfiguration(ctx context.Context, target string) ([]byte, error)
	RestoreConfiguration(ctx context.Context, target string, backup []byte) error
}

// SoftwareManager handles software management operations
type SoftwareManager interface {
	// Software inventory
	GetSoftwareInventory(ctx context.Context, target string) (*SoftwareInventory, error)

	// Software operations
	InstallSoftware(ctx context.Context, req *SoftwareInstallRequest) (*SoftwareOperation, error)
	UpdateSoftware(ctx context.Context, req *SoftwareUpdateRequest) (*SoftwareOperation, error)
	RemoveSoftware(ctx context.Context, req *SoftwareRemoveRequest) (*SoftwareOperation, error)

	// Operation monitoring
	GetSoftwareOperation(ctx context.Context, operationID string) (*SoftwareOperation, error)
	CancelSoftwareOperation(ctx context.Context, operationID string) error
}

// FaultManager handles fault management operations
type FaultManager interface {
	// Fault retrieval and acknowledgment
	GetAlarms(ctx context.Context, filter *AlarmFilter) ([]*Alarm, error)
	GetAlarm(ctx context.Context, alarmID string) (*Alarm, error)
	AcknowledgeAlarm(ctx context.Context, alarmID string, ackInfo *AlarmAcknowledgment) error
	ClearAlarm(ctx context.Context, alarmID string, clearInfo *AlarmClearance) error

	// Fault configuration
	SetFaultThresholds(ctx context.Context, thresholds map[string]*FaultThreshold) error
	GetFaultThresholds(ctx context.Context) (map[string]*FaultThreshold, error)

	// Event subscription
	SubscribeToAlarms(ctx context.Context, subscription *AlarmSubscription) (string, error)
	UnsubscribeFromAlarms(ctx context.Context, subscriptionID string) error
}

// PerformanceManager handles performance management operations
type PerformanceManager interface {
	// Performance measurement collection
	GetPerformanceMetrics(ctx context.Context, filter *PerformanceFilter) (*PerformanceData, error)
	GetHistoricalMetrics(ctx context.Context, filter *HistoricalFilter) (*HistoricalData, error)

	// Performance measurement job management
	CreateMeasurementJob(ctx context.Context, job *MeasurementJobRequest) (*MeasurementJob, error)
	GetMeasurementJob(ctx context.Context, jobID string) (*MeasurementJob, error)
	UpdateMeasurementJob(ctx context.Context, jobID string, updates *MeasurementJobUpdate) error
	DeleteMeasurementJob(ctx context.Context, jobID string) error
	ListMeasurementJobs(ctx context.Context, filter *MeasurementJobFilter) ([]*MeasurementJob, error)

	// Performance thresholds
	SetPerformanceThresholds(ctx context.Context, thresholds map[string]*PerformanceThreshold) error
	GetPerformanceThresholds(ctx context.Context) (map[string]*PerformanceThreshold, error)
}

// NetworkFunctionManager handles network function lifecycle
type NetworkFunctionManager interface {
	// Manager lifecycle
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	GetFunctionCount() int

	// NF lifecycle management
	RegisterNetworkFunction(ctx context.Context, nf *NetworkFunction) error
	DeregisterNetworkFunction(ctx context.Context, nfID string) error
	UpdateNetworkFunction(ctx context.Context, nfID string, updates *NetworkFunctionUpdate) error

	// NF discovery and status
	DiscoverNetworkFunctions(ctx context.Context, criteria *DiscoveryCriteria) ([]*NetworkFunction, error)
	GetNetworkFunctionStatus(ctx context.Context, nfID string) (*NetworkFunctionStatus, error)

	// NF configuration management
	ConfigureNetworkFunction(ctx context.Context, nfID string, config *NetworkFunctionConfig) error
	GetNetworkFunctionConfiguration(ctx context.Context, nfID string) (*NetworkFunctionConfig, error)
}

// Data structures for O1 operations

// SoftwareInventory represents software inventory information
type SoftwareInventory struct {
	TargetID      string          `json:"targetId"`
	SoftwareSlots []*SoftwareSlot `json:"softwareSlots"`
	ActiveSlot    string          `json:"activeSlot"`
	RunningSlot   string          `json:"runningSlot"`
	DownloadSlot  string          `json:"downloadSlot"`
	LastUpdated   time.Time       `json:"lastUpdated"`
	Capabilities  []string        `json:"capabilities"`
	SupportedOps  []string        `json:"supportedOperations"`
}

// SoftwareSlot represents a software slot
type SoftwareSlot struct {
	Name            string            `json:"name"`
	State           string            `json:"state"`
	Active          bool              `json:"active"`
	Running         bool              `json:"running"`
	Access          string            `json:"access"`
	Product         string            `json:"product"`
	ProductRevision string            `json:"productRevision"`
	Version         string            `json:"version"`
	BuildID         string            `json:"buildId"`
	BuildName       string            `json:"buildName"`
	BuildVersion    string            `json:"buildVersion"`
	Files           []*SoftwareFile   `json:"files"`
	AdditionalInfo  map[string]string `json:"additionalInfo"`
}

// SoftwareFile represents a file in a software slot
type SoftwareFile struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	LocalPath    string            `json:"localPath"`
	Size         int64             `json:"size"`
	Checksum     string            `json:"checksum"`
	ChecksumType string            `json:"checksumType"`
	InstallTime  time.Time         `json:"installTime"`
	Permissions  string            `json:"permissions"`
	Attributes   map[string]string `json:"attributes"`
}

// SoftwareOperation represents a software management operation
type SoftwareOperation struct {
	OperationID   string                 `json:"operationId"`
	Type          string                 `json:"type"`
	State         string                 `json:"state"`
	TargetID      string                 `json:"targetId"`
	RequestedBy   string                 `json:"requestedBy"`
	StartTime     time.Time              `json:"startTime"`
	EndTime       *time.Time             `json:"endTime,omitempty"`
	Progress      int                    `json:"progress"`
	StatusMessage string                 `json:"statusMessage"`
	ErrorDetails  *OperationError        `json:"errorDetails,omitempty"`
	Parameters    map[string]interface{} `json:"parameters"`
	Results       map[string]interface{} `json:"results,omitempty"`
}

// OperationError represents an operation error
type OperationError struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Details   string            `json:"details"`
	Timestamp time.Time         `json:"timestamp"`
	Context   map[string]string `json:"context"`
}

// Request structures for software management

// SoftwareInstallRequest represents a software installation request
type SoftwareInstallRequest struct {
	TargetID           string                 `json:"targetId"`
	SlotName           string                 `json:"slotName"`
	PackageURL         string                 `json:"packageUrl"`
	PackageCredentials *Credentials           `json:"packageCredentials,omitempty"`
	InstallOptions     map[string]interface{} `json:"installOptions,omitempty"`
	ValidateOnly       bool                   `json:"validateOnly"`
	ActivateAfter      bool                   `json:"activateAfter"`
	Timeout            time.Duration          `json:"timeout"`
}

// SoftwareUpdateRequest represents a software update request
type SoftwareUpdateRequest struct {
	TargetID          string                 `json:"targetId"`
	CurrentSlot       string                 `json:"currentSlot"`
	UpdatePackage     string                 `json:"updatePackage"`
	UpdateCredentials *Credentials           `json:"updateCredentials,omitempty"`
	UpdateOptions     map[string]interface{} `json:"updateOptions,omitempty"`
	BackupBefore      bool                   `json:"backupBefore"`
	ValidateOnly      bool                   `json:"validateOnly"`
	Timeout           time.Duration          `json:"timeout"`
}

// SoftwareRemoveRequest represents a software removal request
type SoftwareRemoveRequest struct {
	TargetID      string                 `json:"targetId"`
	SlotName      string                 `json:"slotName"`
	ForceRemove   bool                   `json:"forceRemove"`
	CleanupData   bool                   `json:"cleanupData"`
	RemoveOptions map[string]interface{} `json:"removeOptions,omitempty"`
	Timeout       time.Duration          `json:"timeout"`
}

// Credentials represents authentication credentials
type Credentials struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Token    string `json:"token,omitempty"`
	CertPath string `json:"certPath,omitempty"`
	KeyPath  string `json:"keyPath,omitempty"`
	CAPath   string `json:"caPath,omitempty"`
}

// Alarm management structures

// Alarm represents an O1 alarm
type Alarm struct {
	AlarmID           string                 `json:"alarmId"`
	AlarmType         string                 `json:"alarmType"`
	Type              string                 `json:"type"`                  // Missing field added
	Severity          string                 `json:"severity"`              // Missing field added
	SpecificProblem   string                 `json:"specificProblem"`       // Missing field added
	ManagedObjectID   string                 `json:"managedObjectId"`
	EventTime         time.Time              `json:"eventTime"`
	NotificationTime  time.Time              `json:"notificationTime"`
	ProbableCause     string                 `json:"probableCause"`
	PerceivedSeverity string                 `json:"perceivedSeverity"`
	AdditionalText    string                 `json:"additionalText"`
	AdditionalInfo    map[string]interface{} `json:"additionalInfo"`
	AlarmState        string                 `json:"alarmState"`
	AcknowledgedBy    string                 `json:"acknowledgedBy,omitempty"`
	AckTime           *time.Time             `json:"ackTime,omitempty"`
	ClearedBy         string                 `json:"clearedBy,omitempty"`
	ClearTime         *time.Time             `json:"clearTime,omitempty"`
	RootCauseAlarms   []string               `json:"rootCauseAlarms,omitempty"`
	CorrelatedAlarms  []string               `json:"correlatedAlarms,omitempty"`
	RepairActions     []string               `json:"repairActions,omitempty"`
}

// AlarmFilter represents alarm query filters
type AlarmFilter struct {
	AlarmTypes          []string   `json:"alarmTypes,omitempty"`
	ManagedObjectIDs    []string   `json:"managedObjectIds,omitempty"`
	PerceivedSeverities []string   `json:"perceivedSeverities,omitempty"`
	AlarmStates         []string   `json:"alarmStates,omitempty"`
	FromTime            *time.Time `json:"fromTime,omitempty"`
	ToTime              *time.Time `json:"toTime,omitempty"`
	AcknowledgedState   *bool      `json:"acknowledgedState,omitempty"`
	Limit               int        `json:"limit,omitempty"`
	Offset              int        `json:"offset,omitempty"`
}

// AlarmAcknowledgment represents alarm acknowledgment information
type AlarmAcknowledgment struct {
	AcknowledgedBy string                 `json:"acknowledgedBy"`
	AckComment     string                 `json:"ackComment,omitempty"`
	AckTime        time.Time              `json:"ackTime"`
	AdditionalInfo map[string]interface{} `json:"additionalInfo,omitempty"`
}

// AlarmClearance represents alarm clearance information
type AlarmClearance struct {
	ClearedBy      string                 `json:"clearedBy"`
	ClearComment   string                 `json:"clearComment,omitempty"`
	ClearTime      time.Time              `json:"clearTime"`
	AdditionalInfo map[string]interface{} `json:"additionalInfo,omitempty"`
}

// FaultThreshold represents fault threshold configuration
type FaultThreshold struct {
	MetricName        string             `json:"metricName"`
	ThresholdValue    float64            `json:"thresholdValue"`
	Condition         string             `json:"condition"`
	Severity          string             `json:"severity"`
	HysteresisValue   *float64           `json:"hysteresisValue,omitempty"`
	MonitoringPeriod  time.Duration      `json:"monitoringPeriod"`
	ClearingCondition *ClearingCondition `json:"clearingCondition,omitempty"`
	Enabled           bool               `json:"enabled"`
}

// ClearingCondition defines when an alarm should be cleared
type ClearingCondition struct {
	Condition      string        `json:"condition"`
	ThresholdValue float64       `json:"thresholdValue"`
	ClearingDelay  time.Duration `json:"clearingDelay"`
}

// AlarmSubscription represents an alarm subscription
type AlarmSubscription struct {
	SubscriptionID     string       `json:"subscriptionId"`
	Filter             *AlarmFilter `json:"filter"`
	CallbackURL        string       `json:"callbackUrl"`
	NotificationFormat string       `json:"notificationFormat"`
	DeliveryMethod     string       `json:"deliveryMethod"`
	Credentials        *Credentials `json:"credentials,omitempty"`
	RetryPolicy        *RetryPolicy `json:"retryPolicy,omitempty"`
	ExpirationTime     *time.Time   `json:"expirationTime,omitempty"`
}

// RetryPolicy defines retry behavior for notifications
type RetryPolicy struct {
	MaxRetries       int           `json:"maxRetries"`
	RetryInterval    time.Duration `json:"retryInterval"`
	BackoffFactor    float64       `json:"backoffFactor"`
	MaxRetryInterval time.Duration `json:"maxRetryInterval"`
}

// Performance management structures

// PerformanceData represents performance measurement data
type PerformanceData struct {
	MeasurementTime   time.Time                 `json:"measurementTime"`
	MeasurementPeriod time.Duration             `json:"measurementPeriod"`
	ManagedObjectID   string                    `json:"managedObjectId"`
	Measurements      []*PerformanceMeasurement `json:"measurements"`
	JobID             string                    `json:"jobId,omitempty"`
	GranularityPeriod time.Duration             `json:"granularityPeriod"`
	ReportingPeriod   time.Duration             `json:"reportingPeriod"`
	AdditionalInfo    map[string]interface{}    `json:"additionalInfo,omitempty"`
}

// PerformanceMeasurement represents a single performance measurement
type PerformanceMeasurement struct {
	MeasurementType string       `json:"measurementType"`
	Value           float64      `json:"value"`
	Unit            string       `json:"unit"`
	Validity        string       `json:"validity"`
	Timestamp       time.Time    `json:"timestamp"`
	Quality         *DataQuality `json:"quality,omitempty"`
}

// DataQuality represents measurement data quality information
type DataQuality struct {
	Reliability  float64 `json:"reliability"`
	Completeness float64 `json:"completeness"`
	Accuracy     float64 `json:"accuracy"`
	Timeliness   float64 `json:"timeliness"`
}

// PerformanceFilter represents performance data query filters
type PerformanceFilter struct {
	ManagedObjectIDs  []string       `json:"managedObjectIds,omitempty"`
	MeasurementTypes  []string       `json:"measurementTypes,omitempty"`
	FromTime          *time.Time     `json:"fromTime,omitempty"`
	ToTime            *time.Time     `json:"toTime,omitempty"`
	GranularityPeriod *time.Duration `json:"granularityPeriod,omitempty"`
	Limit             int            `json:"limit,omitempty"`
	Offset            int            `json:"offset,omitempty"`
}

// HistoricalFilter represents historical data query filters
type HistoricalFilter struct {
	PerformanceFilter
	AggregationMethod string        `json:"aggregationMethod,omitempty"`
	BinSize           time.Duration `json:"binSize,omitempty"`
}

// HistoricalData represents historical performance data
type HistoricalData struct {
	StartTime         time.Time           `json:"startTime"`
	EndTime           time.Time           `json:"endTime"`
	BinSize           time.Duration       `json:"binSize"`
	AggregationMethod string              `json:"aggregationMethod"`
	DataSeries        []*HistoricalSeries `json:"dataSeries"`
	Statistics        *DataStatistics     `json:"statistics,omitempty"`
}

// HistoricalSeries represents a time series of historical data
type HistoricalSeries struct {
	ManagedObjectID string       `json:"managedObjectId"`
	MeasurementType string       `json:"measurementType"`
	Unit            string       `json:"unit"`
	DataPoints      []*DataPoint `json:"dataPoints"`
}

// DataPoint represents a single data point in a time series
type DataPoint struct {
	Timestamp time.Time    `json:"timestamp"`
	Value     float64      `json:"value"`
	Quality   *DataQuality `json:"quality,omitempty"`
}

// DataStatistics represents statistical information about data
type DataStatistics struct {
	Count        int64   `json:"count"`
	Mean         float64 `json:"mean"`
	Min          float64 `json:"min"`
	Max          float64 `json:"max"`
	StdDev       float64 `json:"stdDev"`
	Percentile95 float64 `json:"percentile95"`
	Percentile99 float64 `json:"percentile99"`
}

// Measurement job management

// MeasurementJobRequest represents a request to create a measurement job
type MeasurementJobRequest struct {
	JobName           string                 `json:"jobName"`
	ManagedObjectIDs  []string               `json:"managedObjectIds"`
	MeasurementTypes  []string               `json:"measurementTypes"`
	GranularityPeriod time.Duration          `json:"granularityPeriod"`
	ReportingPeriod   time.Duration          `json:"reportingPeriod"`
	StartTime         *time.Time             `json:"startTime,omitempty"`
	StopTime          *time.Time             `json:"stopTime,omitempty"`
	FileBasedGP       bool                   `json:"fileBasedGP"`
	FileLocation      string                 `json:"fileLocation,omitempty"`
	CompressionType   string                 `json:"compressionType,omitempty"`
	JobParameters     map[string]interface{} `json:"jobParameters,omitempty"`
}

// MeasurementJob represents a performance measurement job
type MeasurementJob struct {
	JobID               string                 `json:"jobId"`
	JobName             string                 `json:"jobName"`
	AdministrativeState string                 `json:"administrativeState"`
	OperationalState    string                 `json:"operationalState"`
	ManagedObjectIDs    []string               `json:"managedObjectIds"`
	MeasurementTypes    []string               `json:"measurementTypes"`
	GranularityPeriod   time.Duration          `json:"granularityPeriod"`
	ReportingPeriod     time.Duration          `json:"reportingPeriod"`
	StartTime           time.Time              `json:"startTime"`
	StopTime            *time.Time             `json:"stopTime,omitempty"`
	FileBasedGP         bool                   `json:"fileBasedGP"`
	FileLocation        string                 `json:"fileLocation,omitempty"`
	CompressionType     string                 `json:"compressionType,omitempty"`
	JobParameters       map[string]interface{} `json:"jobParameters,omitempty"`
	CreatedTime         time.Time              `json:"createdTime"`
	LastModified        time.Time              `json:"lastModified"`
	CreatedBy           string                 `json:"createdBy"`
	JobState            string                 `json:"jobState"`
	StateChangeTime     time.Time              `json:"stateChangeTime"`
	ProcessingState     string                 `json:"processingState,omitempty"`
	ResultState         string                 `json:"resultState,omitempty"`
	ProgressInfo        *JobProgressInfo       `json:"progressInfo,omitempty"`
}

// JobProgressInfo represents job progress information
type JobProgressInfo struct {
	CompletionPercentage float64                `json:"completionPercentage"`
	ProcessedObjects     int                    `json:"processedObjects"`
	TotalObjects         int                    `json:"totalObjects"`
	EstimatedCompletion  *time.Time             `json:"estimatedCompletion,omitempty"`
	CurrentPhase         string                 `json:"currentPhase"`
	DetailedProgress     map[string]interface{} `json:"detailedProgress,omitempty"`
}

// MeasurementJobUpdate represents updates to a measurement job
type MeasurementJobUpdate struct {
	AdministrativeState *string                `json:"administrativeState,omitempty"`
	StopTime            *time.Time             `json:"stopTime,omitempty"`
	ReportingPeriod     *time.Duration         `json:"reportingPeriod,omitempty"`
	JobParameters       map[string]interface{} `json:"jobParameters,omitempty"`
}

// MeasurementJobFilter represents measurement job query filters
type MeasurementJobFilter struct {
	JobNames             []string   `json:"jobNames,omitempty"`
	JobStates            []string   `json:"jobStates,omitempty"`
	AdministrativeStates []string   `json:"administrativeStates,omitempty"`
	OperationalStates    []string   `json:"operationalStates,omitempty"`
	CreatedAfter         *time.Time `json:"createdAfter,omitempty"`
	CreatedBefore        *time.Time `json:"createdBefore,omitempty"`
	Limit                int        `json:"limit,omitempty"`
	Offset               int        `json:"offset,omitempty"`
}

// PerformanceThreshold represents performance threshold configuration
type PerformanceThreshold struct {
	ID                  string        `json:"id"`                     // Missing field added
	ObjectID            string        `json:"objectId"`               // Missing field added
	MeasurementType     string        `json:"measurementType"`
	ThresholdValue      float64       `json:"thresholdValue"`
	Direction           string        `json:"direction"`
	HysteresisValue     *float64      `json:"hysteresisValue,omitempty"`
	MonitoringPeriod    time.Duration `json:"monitoringPeriod"`
	ThresholdSeverity   string        `json:"thresholdSeverity"`
	ClearingThreshold   *float64      `json:"clearingThreshold,omitempty"`
	NotificationEnabled bool          `json:"notificationEnabled"`
	Enabled             bool          `json:"enabled"`
}

// Network function management structures

// NetworkFunction represents a network function managed by O1
type NetworkFunction struct {
	NFID                             string                             `json:"nfId"`
	NFType                           string                             `json:"nfType"`
	NFStatus                         string                             `json:"nfStatus"`
	HeartBeatTimer                   time.Duration                      `json:"heartBeatTimer"`
	PLMNList                         []*PLMN                            `json:"plmnList"`
	SNSSAIs                          []*SNSSAI                          `json:"snssais,omitempty"`
	PerPLMNSNSSAIList                []*PerPLMNSNSSAI                   `json:"perPlmnSnssaiList,omitempty"`
	NFServices                       []*NFService                       `json:"nfServices,omitempty"`
	DefaultNotificationSubscriptions []*DefaultNotificationSubscription `json:"defaultNotificationSubscriptions,omitempty"`
	NFProfile                        *NFProfile                         `json:"nfProfile,omitempty"`
	ManagementEndpoint               string                             `json:"managementEndpoint"`
	O1Capabilities                   []string                           `json:"o1Capabilities"`
	SupportedFeatures                []string                           `json:"supportedFeatures"`
	AdditionalInfo                   map[string]interface{}             `json:"additionalInfo,omitempty"`
}

// PLMN represents a Public Land Mobile Network identifier
type PLMN struct {
	MCC string `json:"mcc"`
	MNC string `json:"mnc"`
}

// SNSSAI represents a Single Network Slice Selection Assistance Information
type SNSSAI struct {
	SST string `json:"sst"`
	SD  string `json:"sd,omitempty"`
}

// PerPLMNSNSSAI represents SNSSAI information per PLMN
type PerPLMNSNSSAI struct {
	PLMNID  *PLMN     `json:"plmnId"`
	SNSSAIs []*SNSSAI `json:"snssais"`
}

// NFService represents a service provided by a network function
type NFService struct {
	ServiceInstanceID                string                             `json:"serviceInstanceId"`
	ServiceName                      string                             `json:"serviceName"`
	Versions                         []*NFServiceVersion                `json:"versions"`
	Scheme                           string                             `json:"scheme"`
	NFServiceStatus                  string                             `json:"nfServiceStatus"`
	FQDN                             string                             `json:"fqdn,omitempty"`
	InterPLMNFQDN                    string                             `json:"interPlmnFqdn,omitempty"`
	IPEndPoints                      []*IPEndPoint                      `json:"ipEndPoints,omitempty"`
	APIPrefix                        string                             `json:"apiPrefix,omitempty"`
	DefaultNotificationSubscriptions []*DefaultNotificationSubscription `json:"defaultNotificationSubscriptions,omitempty"`
	AllowedPLMNs                     []*PLMN                            `json:"allowedPlmns,omitempty"`
	AllowedNFTypes                   []string                           `json:"allowedNfTypes,omitempty"`
	AllowedNFDomains                 []string                           `json:"allowedNfDomains,omitempty"`
	AllowedNSSAIs                    []*SNSSAI                          `json:"allowedNssais,omitempty"`
	Priority                         int                                `json:"priority,omitempty"`
	Capacity                         int                                `json:"capacity,omitempty"`
	Load                             int                                `json:"load,omitempty"`
	LoadTimeStamp                    *time.Time                         `json:"loadTimeStamp,omitempty"`
	RecoveryTime                     *time.Time                         `json:"recoveryTime,omitempty"`
	ChfServiceList                   []string                           `json:"chfServiceList,omitempty"`
	SupportedFeatures                string                             `json:"supportedFeatures,omitempty"`
}

// NFServiceVersion represents a version of an NF service
type NFServiceVersion struct {
	APIVersionInURI string     `json:"apiVersionInUri"`
	APIFullVersion  string     `json:"apiFullVersion"`
	Expiry          *time.Time `json:"expiry,omitempty"`
}

// IPEndPoint represents an IP endpoint
type IPEndPoint struct {
	IPV4Address string `json:"ipv4Address,omitempty"`
	IPV6Address string `json:"ipv6Address,omitempty"`
	Transport   string `json:"transport,omitempty"`
	Port        int    `json:"port,omitempty"`
}

// DefaultNotificationSubscription represents a default notification subscription
type DefaultNotificationSubscription struct {
	NotificationType   string              `json:"notificationType"`
	CallbackURI        string              `json:"callbackUri"`
	N1MessageClass     string              `json:"n1MessageClass,omitempty"`
	N2InformationClass string              `json:"n2InformationClass,omitempty"`
	Versions           []*NFServiceVersion `json:"versions,omitempty"`
}

// NFProfile represents detailed NF profile information
type NFProfile struct {
	NFInstanceID                     string                             `json:"nfInstanceId"`
	NFType                           string                             `json:"nfType"`
	NFStatus                         string                             `json:"nfStatus"`
	HeartBeatTimer                   time.Duration                      `json:"heartBeatTimer"`
	PLMNList                         []*PLMN                            `json:"plmnList"`
	SNSSAIs                          []*SNSSAI                          `json:"snssais,omitempty"`
	PerPLMNSNSSAIList                []*PerPLMNSNSSAI                   `json:"perPlmnSnssaiList,omitempty"`
	NFServices                       []*NFService                       `json:"nfServices,omitempty"`
	NFServicePersistence             bool                               `json:"nfServicePersistence,omitempty"`
	NFProfileChangesSupportInd       bool                               `json:"nfProfileChangesSupportInd,omitempty"`
	NFProfileChangesInd              bool                               `json:"nfProfileChangesInd,omitempty"`
	DefaultNotificationSubscriptions []*DefaultNotificationSubscription `json:"defaultNotificationSubscriptions,omitempty"`
	FQDN                             string                             `json:"fqdn,omitempty"`
	InterPLMNFQDN                    string                             `json:"interPlmnFqdn,omitempty"`
	IPV4Addresses                    []string                           `json:"ipv4Addresses,omitempty"`
	IPV6Addresses                    []string                           `json:"ipv6Addresses,omitempty"`
	AllowedPLMNs                     []*PLMN                            `json:"allowedPlmns,omitempty"`
	AllowedNFTypes                   []string                           `json:"allowedNfTypes,omitempty"`
	AllowedNFDomains                 []string                           `json:"allowedNfDomains,omitempty"`
	AllowedNSSAIs                    []*SNSSAI                          `json:"allowedNssais,omitempty"`
	Priority                         int                                `json:"priority,omitempty"`
	Capacity                         int                                `json:"capacity,omitempty"`
	Load                             int                                `json:"load,omitempty"`
	LoadTimeStamp                    *time.Time                         `json:"loadTimeStamp,omitempty"`
	Locality                         string                             `json:"locality,omitempty"`
	UDRInfo                          map[string]interface{}             `json:"udrInfo,omitempty"`
	UDMInfo                          map[string]interface{}             `json:"udmInfo,omitempty"`
	AUSFInfo                         map[string]interface{}             `json:"ausfInfo,omitempty"`
	AMFInfo                          map[string]interface{}             `json:"amfInfo,omitempty"`
	SMFInfo                          map[string]interface{}             `json:"smfInfo,omitempty"`
	UPFInfo                          map[string]interface{}             `json:"upfInfo,omitempty"`
	PCFInfo                          map[string]interface{}             `json:"pcfInfo,omitempty"`
	BSFInfo                          map[string]interface{}             `json:"bsfInfo,omitempty"`
	CHFInfo                          map[string]interface{}             `json:"chfInfo,omitempty"`
	NRFInfo                          map[string]interface{}             `json:"nrfInfo,omitempty"`
	CustomInfo                       map[string]interface{}             `json:"customInfo,omitempty"`
	RecoveryTime                     *time.Time                         `json:"recoveryTime,omitempty"`
	NFServicePersistenceList         []string                           `json:"nfServicePersistenceList,omitempty"`
	SupportedFeatures                string                             `json:"supportedFeatures,omitempty"`
}

// NetworkFunctionUpdate represents updates to a network function
type NetworkFunctionUpdate struct {
	NFStatus          *string                `json:"nfStatus,omitempty"`
	HeartBeatTimer    *time.Duration         `json:"heartBeatTimer,omitempty"`
	NFServices        []*NFService           `json:"nfServices,omitempty"`
	Priority          *int                   `json:"priority,omitempty"`
	Capacity          *int                   `json:"capacity,omitempty"`
	Load              *int                   `json:"load,omitempty"`
	LoadTimeStamp     *time.Time             `json:"loadTimeStamp,omitempty"`
	SupportedFeatures []string               `json:"supportedFeatures,omitempty"`
	AdditionalInfo    map[string]interface{} `json:"additionalInfo,omitempty"`
}

// NetworkFunctionStatus represents current status of a network function
type NetworkFunctionStatus struct {
	NFID                 string                 `json:"nfId"`
	NFType               string                 `json:"nfType"`
	NFStatus             string                 `json:"nfStatus"`
	OperationalState     string                 `json:"operationalState"`
	AdministrativeState  string                 `json:"administrativeState"`
	AvailabilityState    string                 `json:"availabilityState"`
	HealthStatus         string                 `json:"healthStatus"`
	LastHeartbeat        time.Time              `json:"lastHeartbeat"`
	Uptime               time.Duration          `json:"uptime"`
	Load                 int                    `json:"load"`
	LoadTimeStamp        time.Time              `json:"loadTimeStamp"`
	ActiveConnections    int                    `json:"activeConnections"`
	ErrorRate            float64                `json:"errorRate"`
	PerformanceMetrics   map[string]float64     `json:"performanceMetrics"`
	AlarmCount           int                    `json:"alarmCount"`
	CriticalAlarms       int                    `json:"criticalAlarms"`
	MajorAlarms          int                    `json:"majorAlarms"`
	MinorAlarms          int                    `json:"minorAlarms"`
	WarningAlarms        int                    `json:"warningAlarms"`
	ServiceStatus        map[string]string      `json:"serviceStatus"`
	ConfigurationVersion string                 `json:"configurationVersion"`
	SoftwareVersion      string                 `json:"softwareVersion"`
	AdditionalInfo       map[string]interface{} `json:"additionalInfo,omitempty"`
}

// NetworkFunctionConfig represents network function configuration
type NetworkFunctionConfig struct {
	ConfigID         string                 `json:"configId"`
	Version          string                 `json:"version"`
	ConfigName       string                 `json:"configName"`
	Description      string                 `json:"description"`
	ConfigData       map[string]interface{} `json:"configData"`
	Schema           string                 `json:"schema,omitempty"`
	ValidatedAt      time.Time              `json:"validatedAt"`
	AppliedAt        *time.Time             `json:"appliedAt,omitempty"`
	CreatedBy        string                 `json:"createdBy"`
	ModifiedBy       string                 `json:"modifiedBy,omitempty"`
	CreatedAt        time.Time              `json:"createdAt"`
	ModifiedAt       *time.Time             `json:"modifiedAt,omitempty"`
	Status           string                 `json:"status"`
	ValidationErrors []string               `json:"validationErrors,omitempty"`
	Dependencies     []string               `json:"dependencies,omitempty"`
	Tags             map[string]string      `json:"tags,omitempty"`
}

// DiscoveryCriteria represents criteria for network function discovery
type DiscoveryCriteria struct {
	NFType            string    `json:"nfType,omitempty"`
	PLMNs             []*PLMN   `json:"plmns,omitempty"`
	SNSSAIs           []*SNSSAI `json:"snssais,omitempty"`
	RequiredNFTypes   []string  `json:"requiredNfTypes,omitempty"`
	PreferredLocality string    `json:"preferredLocality,omitempty"`
	Limit             int       `json:"limit,omitempty"`
	ServiceNames      []string  `json:"serviceNames,omitempty"`
}

// Subscription represents a generic subscription
type Subscription struct {
	SubscriptionID     string                 `json:"subscriptionId"`
	EventType          string                 `json:"eventType"`
	EventFilter        map[string]interface{} `json:"eventFilter,omitempty"`
	CallbackReference  string                 `json:"callbackReference"`
	Links              map[string]interface{} `json:"_links,omitempty"`
	ExpiryDeadline     *time.Time             `json:"expiryDeadline,omitempty"`
	SupportedFeatures  string                 `json:"supportedFeatures,omitempty"`
	SubscriptionStatus string                 `json:"subscriptionStatus"`
	CreatedAt          time.Time              `json:"createdAt"`
	LastNotified       *time.Time             `json:"lastNotified,omitempty"`
	NotificationCount  int64                  `json:"notificationCount"`
	FailureCount       int64                  `json:"failureCount"`
}

// O1Metrics represents O1 adapter metrics
type O1Metrics struct {
	ConfigurationOperations   int64                  `json:"configurationOperations"`
	SoftwareOperations        int64                  `json:"softwareOperations"`
	AlarmNotifications        int64                  `json:"alarmNotifications"`
	PerformanceMeasurements   int64                  `json:"performanceMeasurements"`
	ActiveSubscriptions       int64                  `json:"activeSubscriptions"`
	ConnectedNetworkFunctions int64                  `json:"connectedNetworkFunctions"`
	TotalMeasurementJobs      int64                  `json:"totalMeasurementJobs"`
	ActiveMeasurementJobs     int64                  `json:"activeMeasurementJobs"`
	SuccessfulOperations      int64                  `json:"successfulOperations"`
	FailedOperations          int64                  `json:"failedOperations"`
	AverageResponseTime       time.Duration          `json:"averageResponseTime"`
	ErrorRate                 float64                `json:"errorRate"`
	LastOperationTime         time.Time              `json:"lastOperationTime"`
	AdditionalMetrics         map[string]interface{} `json:"additionalMetrics,omitempty"`
}

// NotificationHub handles O1 notifications
type NotificationHub struct {
	activeSubscriptions map[string]*Subscription
	notificationQueue   chan *Notification
	deliveryWorkers     int
	retryPolicy         *RetryPolicy
	logger              logr.Logger
}

// Notification represents an O1 notification
type Notification struct {
	NotificationID   string                 `json:"notificationId"`
	NotificationType string                 `json:"notificationType"`
	EventTime        time.Time              `json:"eventTime"`
	SystemDN         string                 `json:"systemDN"`
	NotificationData map[string]interface{} `json:"notificationData"`
	CorrelationID    string                 `json:"correlationId,omitempty"`
	AdditionalInfo   map[string]interface{} `json:"additionalInfo,omitempty"`
}

// NewO1Adapter creates a new O1 adapter instance
func NewO1Adapter(logger logr.Logger) *O1Adapter {
	return &O1Adapter{
		logger:        logger.WithName("o1-adapter"),
		subscriptions: make(map[string]*Subscription),
		connectionState: ConnectionState{
			State:        "disconnected",
			ConnectedNFs: []string{},
			HealthStatus: "unknown",
			Capabilities: []string{
				"configuration-management",
				"software-management",
				"fault-management",
				"performance-management",
				"notification-management",
			},
			Configuration: make(map[string]interface{}),
		},
		metrics: &O1Metrics{
			AdditionalMetrics: make(map[string]interface{}),
		},
		notificationHub: &NotificationHub{
			activeSubscriptions: make(map[string]*Subscription),
			notificationQueue:   make(chan *Notification, 1000),
			deliveryWorkers:     5,
			retryPolicy: &RetryPolicy{
				MaxRetries:       3,
				RetryInterval:    30 * time.Second,
				BackoffFactor:    2.0,
				MaxRetryInterval: 300 * time.Second,
			},
			logger: logger.WithName("notification-hub"),
		},
	}
}

// Initialize initializes the O1 adapter
func (adapter *O1Adapter) Initialize(ctx context.Context) error {
	adapter.logger.Info("Initializing O1 adapter")

	// Initialize connection state
	adapter.connectionState.State = "connecting"
	adapter.connectionState.LastConnected = time.Now()

	// Start heartbeat mechanism
	go adapter.startHeartbeat(ctx)

	// Start notification hub
	go adapter.notificationHub.start(ctx)

	// Set state to connected
	adapter.connectionState.State = "connected"
	adapter.connectionState.HealthStatus = "healthy"

	adapter.logger.Info("O1 adapter initialized successfully")
	return nil
}

// Shutdown gracefully shuts down the O1 adapter
func (adapter *O1Adapter) Shutdown(ctx context.Context) error {
	adapter.logger.Info("Shutting down O1 adapter")

	adapter.connectionState.State = "disconnecting"

	// Close notification hub
	if adapter.notificationHub != nil {
		close(adapter.notificationHub.notificationQueue)
	}

	adapter.connectionState.State = "disconnected"
	adapter.logger.Info("O1 adapter shutdown completed")

	return nil
}

// GetConnectionState returns the current connection state
func (adapter *O1Adapter) GetConnectionState() *ConnectionState {
	return &adapter.connectionState
}

// GetMetrics returns current O1 metrics
func (adapter *O1Adapter) GetMetrics() *O1Metrics {
	return adapter.metrics
}

// Private helper methods

func (adapter *O1Adapter) startHeartbeat(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			adapter.lastHeartbeat = time.Now()
			adapter.connectionState.HealthStatus = "healthy"
		}
	}
}

func (hub *NotificationHub) start(ctx context.Context) {
	// Start notification delivery workers
	for i := 0; i < hub.deliveryWorkers; i++ {
		go hub.deliveryWorker(ctx, i)
	}

	hub.logger.Info("Notification hub started", "workers", hub.deliveryWorkers)
}

func (hub *NotificationHub) deliveryWorker(ctx context.Context, workerID int) {
	logger := hub.logger.WithValues("worker", workerID)
	logger.Info("Notification delivery worker started")

	for {
		select {
		case <-ctx.Done():
			logger.Info("Notification delivery worker stopping")
			return
		case notification := <-hub.notificationQueue:
			if notification != nil {
				hub.deliverNotification(ctx, notification)
			}
		}
	}
}

func (hub *NotificationHub) deliverNotification(ctx context.Context, notification *Notification) {
	// Implementation for notification delivery would go here
	// This would handle retry logic, different delivery methods, etc.
	hub.logger.Info("Delivering notification",
		"notificationId", notification.NotificationID,
		"type", notification.NotificationType,
	)
}
