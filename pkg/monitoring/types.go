package monitoring

import (
	"context"
	"time"
)

// Severity levels for alerts and events
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
	SeverityError    Severity = "error"
)

// Alert severity alias for compatibility
type AlertSeverity = Severity

const (
	AlertSeverityInfo     = SeverityInfo
	AlertSeverityWarning  = SeverityWarning
	AlertSeverityCritical = SeverityCritical
	AlertSeverityError    = SeverityError
)

// VES Event structures
type VESEvent struct {
	Event VESEventData `json:"event"`
}

type VESEventData struct {
	CommonEventHeader    VESCommonEventHeader `json:"commonEventHeader"`
	ThresholdCrossingAlert *VESThresholdAlert `json:"thresholdCrossingAlert,omitempty"`
	FaultFields         *VESFaultFields      `json:"faultFields,omitempty"`
	MeasurementFields   *VESMeasurementFields `json:"measurementFields,omitempty"`
}

type VESCommonEventHeader struct {
	Domain                 string    `json:"domain"`
	EventID                string    `json:"eventId"`
	EventName              string    `json:"eventName"`
	EventType              string    `json:"eventType"`
	InternalHeaderFields   map[string]string `json:"internalHeaderFields,omitempty"`
	LastEpochMicrosec      int64     `json:"lastEpochMicrosec"`
	NfNamingCode           string    `json:"nfNamingCode,omitempty"`
	NfVendorName           string    `json:"nfVendorName,omitempty"`
	Priority               string    `json:"priority"`
	ReportingEntityID      string    `json:"reportingEntityId,omitempty"`
	ReportingEntityName    string    `json:"reportingEntityName"`
	Sequence               int       `json:"sequence"`
	SourceID               string    `json:"sourceId,omitempty"`
	SourceName             string    `json:"sourceName"`
	StartEpochMicrosec     int64     `json:"startEpochMicrosec"`
	TimeZoneOffset         string    `json:"timeZoneOffset,omitempty"`
	Version                string    `json:"version"`
	VesEventListenerVersion string   `json:"vesEventListenerVersion"`
}

type VESThresholdAlert struct {
	AlertAction                string                   `json:"alertAction"`
	AlertDescription           string                   `json:"alertDescription"`
	AlertType                  string                   `json:"alertType"`
	CollectionTimestamp        string                   `json:"collectionTimestamp"`
	ElementType                string                   `json:"elementType"`
	EventSeverity              string                   `json:"eventSeverity"`
	EventStartTimestamp        string                   `json:"eventStartTimestamp"`
	NetworkService             string                   `json:"networkService"`
	PossibleRootCause          string                   `json:"possibleRootCause"`
	ThresholdCrossingFieldsVersion string                `json:"thresholdCrossingFieldsVersion"`
	AdditionalFields           []VESNameValuePair       `json:"additionalFields,omitempty"`
	AssociatedAlertIdList      []string                 `json:"associatedAlertIdList,omitempty"`
	AdditionalParameters       []VESCounter             `json:"additionalParameters,omitempty"`
}

type VESFaultFields struct {
	AlarmCondition          string             `json:"alarmCondition"`
	AlarmInterfaceA         string             `json:"alarmInterfaceA,omitempty"`
	EventCategory           string             `json:"eventCategory,omitempty"`
	EventSeverity           string             `json:"eventSeverity"`
	EventSourceType         string             `json:"eventSourceType"`
	FaultFieldsVersion      string             `json:"faultFieldsVersion"`
	SpecificProblem         string             `json:"specificProblem"`
	VfStatus                string             `json:"vfStatus"`
	AdditionalFields        []VESNameValuePair `json:"additionalFields,omitempty"`
}

type VESMeasurementFields struct {
	MeasurementFieldsVersion    string              `json:"measurementFieldsVersion"`
	MeasurementInterval         float64             `json:"measurementInterval"`
	AdditionalFields            []VESNameValuePair  `json:"additionalFields,omitempty"`
	AdditionalMeasurements      []VESNamedMeasurement `json:"additionalMeasurements,omitempty"`
	AdditionalObjects           []VESJSONObject     `json:"additionalObjects,omitempty"`
	CodecUsageArray             []VESCodecUsage     `json:"codecUsageArray,omitempty"`
	ConcurrentSessions          int                 `json:"concurrentSessions,omitempty"`
	ConfiguredEntities          int                 `json:"configuredEntities,omitempty"`
	CPUUsageArray               []VESCPUUsage       `json:"cpuUsageArray,omitempty"`
	DiskUsageArray              []VESDiskUsage      `json:"diskUsageArray,omitempty"`
	FeatureUsageArray           []VESFeatureUsage   `json:"featureUsageArray,omitempty"`
	FilesystemUsageArray        []VESFilesystemUsage `json:"filesystemUsageArray,omitempty"`
	LatencyDistribution         []VESLatencyBucket  `json:"latencyDistribution,omitempty"`
	MeanRequestLatency          float64             `json:"meanRequestLatency,omitempty"`
	MemoryUsageArray            []VESMemoryUsage    `json:"memoryUsageArray,omitempty"`
	NumberOfMediaPortsInUse     int                 `json:"numberOfMediaPortsInUse,omitempty"`
	RequestRate                 float64             `json:"requestRate,omitempty"`
	VNICPerformanceArray        []VESVNICPerformance `json:"vNicPerformanceArray,omitempty"`
}

type VESNameValuePair struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type VESCounter struct {
	CriticCount    int    `json:"criticCount"`
	MajorCount     int    `json:"majorCount"`
	MinorCount     int    `json:"minorCount"`
	WarningCount   int    `json:"warningCount"`
	IndeterminateCount int `json:"indeterminateCount"`
}

type VESNamedMeasurement struct {
	Name                  string             `json:"name"`
	ArrayOfFields         []VESNameValuePair `json:"arrayOfFields,omitempty"`
}

type VESJSONObject struct {
	ObjectName     string                 `json:"objectName"`
	ObjectKeys     []VESKey               `json:"objectKeys,omitempty"`
	ObjectInstance map[string]interface{} `json:"objectInstance,omitempty"`
}

type VESKey struct {
	KeyName  string `json:"keyName"`
	KeyOrder int    `json:"keyOrder,omitempty"`
	KeyValue string `json:"keyValue,omitempty"`
}

type VESCodecUsage struct {
	CodecIdentifier   string `json:"codecIdentifier"`
	NumberInUse       int    `json:"numberInUse"`
}

type VESCPUUsage struct {
	CPUIdentifier          string  `json:"cpuIdentifier"`
	CPUIdle                float64 `json:"cpuIdle,omitempty"`
	CPUUsageInterrupt      float64 `json:"cpuUsageInterrupt,omitempty"`
	CPUUsageNice           float64 `json:"cpuUsageNice,omitempty"`
	CPUUsageSoftirq        float64 `json:"cpuUsageSoftirq,omitempty"`
	CPUUsageSteal          float64 `json:"cpuUsageSteal,omitempty"`
	CPUUsageSystem         float64 `json:"cpuUsageSystem,omitempty"`
	CPUUsageUser           float64 `json:"cpuUsageUser,omitempty"`
	CPUWait                float64 `json:"cpuWait,omitempty"`
	PercentUsage           float64 `json:"percentUsage"`
}

type VESDiskUsage struct {
	DiskIdentifier         string  `json:"diskIdentifier"`
	DiskIoTimeAvg          float64 `json:"diskIoTimeAvg,omitempty"`
	DiskIoTimeLast         float64 `json:"diskIoTimeLast,omitempty"`
	DiskIoTimeMax          float64 `json:"diskIoTimeMax,omitempty"`
	DiskIoTimeMin          float64 `json:"diskIoTimeMin,omitempty"`
	DiskMergedReadAvg      float64 `json:"diskMergedReadAvg,omitempty"`
	DiskMergedReadLast     float64 `json:"diskMergedReadLast,omitempty"`
	DiskMergedReadMax      float64 `json:"diskMergedReadMax,omitempty"`
	DiskMergedReadMin      float64 `json:"diskMergedReadMin,omitempty"`
	DiskMergedWriteAvg     float64 `json:"diskMergedWriteAvg,omitempty"`
	DiskMergedWriteLast    float64 `json:"diskMergedWriteLast,omitempty"`
	DiskMergedWriteMax     float64 `json:"diskMergedWriteMax,omitempty"`
	DiskMergedWriteMin     float64 `json:"diskMergedWriteMin,omitempty"`
	DiskOctetsReadAvg      float64 `json:"diskOctetsReadAvg,omitempty"`
	DiskOctetsReadLast     float64 `json:"diskOctetsReadLast,omitempty"`
	DiskOctetsReadMax      float64 `json:"diskOctetsReadMax,omitempty"`
	DiskOctetsReadMin      float64 `json:"diskOctetsReadMin,omitempty"`
	DiskOctetsWriteAvg     float64 `json:"diskOctetsWriteAvg,omitempty"`
	DiskOctetsWriteLast    float64 `json:"diskOctetsWriteLast,omitempty"`
	DiskOctetsWriteMax     float64 `json:"diskOctetsWriteMax,omitempty"`
	DiskOctetsWriteMin     float64 `json:"diskOctetsWriteMin,omitempty"`
	DiskOpsReadAvg         float64 `json:"diskOpsReadAvg,omitempty"`
	DiskOpsReadLast        float64 `json:"diskOpsReadLast,omitempty"`
	DiskOpsReadMax         float64 `json:"diskOpsReadMax,omitempty"`
	DiskOpsReadMin         float64 `json:"diskOpsReadMin,omitempty"`
	DiskOpsWriteAvg        float64 `json:"diskOpsWriteAvg,omitempty"`
	DiskOpsWriteLast       float64 `json:"diskOpsWriteLast,omitempty"`
	DiskOpsWriteMax        float64 `json:"diskOpsWriteMax,omitempty"`
	DiskOpsWriteMin        float64 `json:"diskOpsWriteMin,omitempty"`
	DiskPendingOperationsAvg float64 `json:"diskPendingOperationsAvg,omitempty"`
	DiskPendingOperationsLast float64 `json:"diskPendingOperationsLast,omitempty"`
	DiskPendingOperationsMax float64 `json:"diskPendingOperationsMax,omitempty"`
	DiskPendingOperationsMin float64 `json:"diskPendingOperationsMin,omitempty"`
	DiskTimeReadAvg        float64 `json:"diskTimeReadAvg,omitempty"`
	DiskTimeReadLast       float64 `json:"diskTimeReadLast,omitempty"`
	DiskTimeReadMax        float64 `json:"diskTimeReadMax,omitempty"`
	DiskTimeReadMin        float64 `json:"diskTimeReadMin,omitempty"`
	DiskTimeWriteAvg       float64 `json:"diskTimeWriteAvg,omitempty"`
	DiskTimeWriteLast      float64 `json:"diskTimeWriteLast,omitempty"`
	DiskTimeWriteMax       float64 `json:"diskTimeWriteMax,omitempty"`
	DiskTimeWriteMin       float64 `json:"diskTimeWriteMin,omitempty"`
}

type VESFeatureUsage struct {
	FeatureIdentifier      string `json:"featureIdentifier"`
	FeatureUtilization     int    `json:"featureUtilization"`
}

type VESFilesystemUsage struct {
	BlockConfigured       float64 `json:"blockConfigured"`
	BlockInode            float64 `json:"blockInode"`
	BlockUsed             float64 `json:"blockUsed"`
	EphemeralConfigured   float64 `json:"ephemeralConfigured,omitempty"`
	EphemeralInode        float64 `json:"ephemeralInode,omitempty"`
	EphemeralUsed         float64 `json:"ephemeralUsed,omitempty"`
	FilesystemName        string  `json:"filesystemName"`
}

type VESLatencyBucket struct {
	CountsInTheBucket     int     `json:"countsInTheBucket"`
	HighEndOfLatencyBucket float64 `json:"highEndOfLatencyBucket,omitempty"`
	LowEndOfLatencyBucket  float64 `json:"lowEndOfLatencyBucket,omitempty"`
}

type VESMemoryUsage struct {
	MemoryBuffered         float64 `json:"memoryBuffered,omitempty"`
	MemoryCached           float64 `json:"memoryCached,omitempty"`
	MemoryConfigured       float64 `json:"memoryConfigured,omitempty"`
	MemoryFree             float64 `json:"memoryFree,omitempty"`
	MemorySlabRecl         float64 `json:"memorySlabRecl,omitempty"`
	MemorySlabUnrecl       float64 `json:"memorySlabUnrecl,omitempty"`
	MemoryUsed             float64 `json:"memoryUsed,omitempty"`
	VmIdentifier           string  `json:"vmIdentifier"`
}

type VESVNICPerformance struct {
	ReceivedBroadcastPacketsAccumulated   float64 `json:"receivedBroadcastPacketsAccumulated,omitempty"`
	ReceivedBroadcastPacketsDelta         float64 `json:"receivedBroadcastPacketsDelta,omitempty"`
	ReceivedDiscardedPacketsAccumulated   float64 `json:"receivedDiscardedPacketsAccumulated,omitempty"`
	ReceivedDiscardedPacketsDelta         float64 `json:"receivedDiscardedPacketsDelta,omitempty"`
	ReceivedErrorPacketsAccumulated       float64 `json:"receivedErrorPacketsAccumulated,omitempty"`
	ReceivedErrorPacketsDelta             float64 `json:"receivedErrorPacketsDelta,omitempty"`
	ReceivedMulticastPacketsAccumulated   float64 `json:"receivedMulticastPacketsAccumulated,omitempty"`
	ReceivedMulticastPacketsDelta         float64 `json:"receivedMulticastPacketsDelta,omitempty"`
	ReceivedOctetsAccumulated             float64 `json:"receivedOctetsAccumulated,omitempty"`
	ReceivedOctetsDelta                   float64 `json:"receivedOctetsDelta,omitempty"`
	ReceivedTotalPacketsAccumulated       float64 `json:"receivedTotalPacketsAccumulated,omitempty"`
	ReceivedTotalPacketsDelta             float64 `json:"receivedTotalPacketsDelta,omitempty"`
	ReceivedUnicastPacketsAccumulated     float64 `json:"receivedUnicastPacketsAccumulated,omitempty"`
	ReceivedUnicastPacketsDelta           float64 `json:"receivedUnicastPacketsDelta,omitempty"`
	TransmittedBroadcastPacketsAccumulated float64 `json:"transmittedBroadcastPacketsAccumulated,omitempty"`
	TransmittedBroadcastPacketsDelta      float64 `json:"transmittedBroadcastPacketsDelta,omitempty"`
	TransmittedDiscardedPacketsAccumulated float64 `json:"transmittedDiscardedPacketsAccumulated,omitempty"`
	TransmittedDiscardedPacketsDelta      float64 `json:"transmittedDiscardedPacketsDelta,omitempty"`
	TransmittedErrorPacketsAccumulated    float64 `json:"transmittedErrorPacketsAccumulated,omitempty"`
	TransmittedErrorPacketsDelta          float64 `json:"transmittedErrorPacketsDelta,omitempty"`
	TransmittedMulticastPacketsAccumulated float64 `json:"transmittedMulticastPacketsAccumulated,omitempty"`
	TransmittedMulticastPacketsDelta      float64 `json:"transmittedMulticastPacketsDelta,omitempty"`
	TransmittedOctetsAccumulated          float64 `json:"transmittedOctetsAccumulated,omitempty"`
	TransmittedOctetsDelta                float64 `json:"transmittedOctetsDelta,omitempty"`
	TransmittedTotalPacketsAccumulated    float64 `json:"transmittedTotalPacketsAccumulated,omitempty"`
	TransmittedTotalPacketsDelta          float64 `json:"transmittedTotalPacketsDelta,omitempty"`
	TransmittedUnicastPacketsAccumulated  float64 `json:"transmittedUnicastPacketsAccumulated,omitempty"`
	TransmittedUnicastPacketsDelta        float64 `json:"transmittedUnicastPacketsDelta,omitempty"`
	ValuesAreSuspect                      string  `json:"valuesAreSuspect,omitempty"`
	VNicIdentifier                        string  `json:"vNicIdentifier"`
}

// Alert represents an active alert in the system
type Alert struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Rule        string                 `json:"rule"`
	Component   string                 `json:"component"`
	Message     string                 `json:"message"`
	Severity    AlertSeverity          `json:"severity"`
	Status      AlertStatus            `json:"status"`
	Labels      map[string]string      `json:"labels"`
	Annotations map[string]string      `json:"annotations"`
	StartsAt    time.Time              `json:"startsAt"`
	EndsAt      *time.Time             `json:"endsAt,omitempty"`
	GeneratorURL string                `json:"generatorURL,omitempty"`
	Fingerprint  string                `json:"fingerprint"`
	Silenced    bool                   `json:"silenced,omitempty"`
	AckBy       string                 `json:"ackBy,omitempty"`
	AckAt       *time.Time             `json:"ackAt,omitempty"`
}

type AlertStatus string

const (
	AlertStatusFiring   AlertStatus = "firing"
	AlertStatusResolved AlertStatus = "resolved"
	AlertStatusPending  AlertStatus = "pending"
	AlertStatusSilenced AlertStatus = "silenced"
)

// AlertRule defines conditions for generating alerts
type AlertRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Expression  string                 `json:"expression"`  // Query expression (e.g., Prometheus query)
	Condition   string                 `json:"condition"`   // "gt", "lt", "eq", "ne"
	Threshold   float64                `json:"threshold"`
	Duration    time.Duration          `json:"duration"`    // How long condition must be true
	Cooldown    time.Duration          `json:"cooldown"`    // Cooldown period between alerts
	Component   string                 `json:"component"`   // Component this rule applies to
	Severity    AlertSeverity          `json:"severity"`
	Labels      map[string]string      `json:"labels"`
	Annotations map[string]string      `json:"annotations"`
	Enabled     bool                   `json:"enabled"`
	LastFired   time.Time              `json:"lastFired"`
	Channels    []string               `json:"channels"`    // Notification channels
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

// NotificationChannel represents a way to send alert notifications
type NotificationChannel struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // "email", "slack", "webhook", "pagerduty"
	Config      map[string]interface{} `json:"config"`
	Enabled     bool                   `json:"enabled"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

// AlertManagerInterface defines the interface for alert management
type AlertManagerInterface interface {
	// Rule management
	CreateRule(ctx context.Context, rule *AlertRule) error
	UpdateRule(ctx context.Context, rule *AlertRule) error
	DeleteRule(ctx context.Context, ruleID string) error
	GetRule(ctx context.Context, ruleID string) (*AlertRule, error)
	ListRules(ctx context.Context) ([]*AlertRule, error)

	// Alert management  
	CreateAlert(ctx context.Context, alert *Alert) error
	GetAlert(ctx context.Context, alertID string) (*Alert, error)
	ListAlerts(ctx context.Context, filters map[string]string) ([]*Alert, error)
	AcknowledgeAlert(ctx context.Context, alertID, ackBy string) error
	SilenceAlert(ctx context.Context, alertID string, duration time.Duration) error
	ResolveAlert(ctx context.Context, alertID string) error

	// Channel management
	CreateChannel(ctx context.Context, channel *NotificationChannel) error
	UpdateChannel(ctx context.Context, channel *NotificationChannel) error
	DeleteChannel(ctx context.Context, channelID string) error
	GetChannel(ctx context.Context, channelID string) (*NotificationChannel, error)
	ListChannels(ctx context.Context) ([]*NotificationChannel, error)

	// Evaluation and notification
	EvaluateRules(ctx context.Context) error
	SendNotification(ctx context.Context, alert *Alert, channels []string) error
}

// MetricValue represents a single metric measurement
type MetricValue struct {
	Name      string                 `json:"name"`
	Value     float64                `json:"value"`
	Labels    map[string]string      `json:"labels"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// MetricCollector defines the interface for collecting metrics
type MetricCollector interface {
	Collect(ctx context.Context) ([]*MetricValue, error)
	GetMetric(ctx context.Context, name string, labels map[string]string) (*MetricValue, error)
	ListMetrics(ctx context.Context) ([]string, error)
}

// HealthCheck represents the result of a health check
type HealthCheck struct {
	Component   string                 `json:"component"`
	Status      string                 `json:"status"`
	Message     string                 `json:"message,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Duration    time.Duration          `json:"duration"`
}

// HealthChecker defines the interface for health checking
type HealthChecker interface {
	CheckHealth(ctx context.Context) (*HealthCheck, error)
	GetComponentName() string
}

// DefaultHealthChecker is a default implementation of HealthChecker
type DefaultHealthChecker struct {
	name       string
	version    string
	kubeClient interface{}
	recorder   interface{}
}

// NewHealthChecker creates a new default health checker
func NewHealthChecker(version string, kubeClient interface{}, recorder interface{}) HealthChecker {
	return &DefaultHealthChecker{
		name:       "nephoran-controller",
		version:    version,
		kubeClient: kubeClient,
		recorder:   recorder,
	}
}

// CheckHealth implements HealthChecker
func (d *DefaultHealthChecker) CheckHealth(ctx context.Context) (*HealthCheck, error) {
	return &HealthCheck{
		Component: d.name,
		Status:    "healthy",
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"version": d.version,
		},
	}, nil
}

// GetComponentName implements HealthChecker
func (d *DefaultHealthChecker) GetComponentName() string {
	return d.name
}

// MetricsCollector interface for collecting metrics
type MetricsCollector interface {
	CollectMetrics(ctx context.Context) (*PerformanceMetrics, error)
	UpdateControllerHealth(controllerName, component string, healthy bool)
	RecordReconciliation(controllerName string, duration time.Duration, success bool)
	UpdateNetworkIntentStatus(name, namespace, intentType, status string)
	RecordIntentProcessingDuration(name, namespace string, duration time.Duration, success bool)
	RecordKubernetesAPILatency(duration time.Duration)
	GetName() string
}

// DefaultMetricsCollector is a default implementation of MetricsCollector
type DefaultMetricsCollector struct {
	name string
}

// NewDefaultMetricsCollector creates a new default metrics collector
func NewDefaultMetricsCollector(name string) MetricsCollector {
	return &DefaultMetricsCollector{name: name}
}

// CollectMetrics implements MetricsCollector
func (d *DefaultMetricsCollector) CollectMetrics(ctx context.Context) (*PerformanceMetrics, error) {
	return &PerformanceMetrics{
		Timestamp: time.Now(),
	}, nil
}

// UpdateControllerHealth implements MetricsCollector
func (d *DefaultMetricsCollector) UpdateControllerHealth(controllerName, component string, healthy bool) {
	// Default implementation - could log or emit metrics
}

// RecordReconciliation implements MetricsCollector
func (d *DefaultMetricsCollector) RecordReconciliation(controllerName string, duration time.Duration, success bool) {
	// Default implementation - could log or emit metrics
}

// UpdateNetworkIntentStatus implements MetricsCollector
func (d *DefaultMetricsCollector) UpdateNetworkIntentStatus(name, namespace, intentType, status string) {
	// Default implementation - could log or emit metrics
}

// RecordIntentProcessingDuration implements MetricsCollector
func (d *DefaultMetricsCollector) RecordIntentProcessingDuration(name, namespace string, duration time.Duration, success bool) {
	// Default implementation - could log or emit metrics
}

// RecordKubernetesAPILatency implements MetricsCollector
func (d *DefaultMetricsCollector) RecordKubernetesAPILatency(duration time.Duration) {
	// Default implementation - could log or emit metrics
}

// GetName implements MetricsCollector
func (d *DefaultMetricsCollector) GetName() string {
	return d.name
}

// ThresholdConfig defines configuration for threshold-based alerting
type ThresholdConfig struct {
	MetricName    string                 `json:"metric_name"`
	Operator      string                 `json:"operator"` // "gt", "gte", "lt", "lte", "eq", "ne"
	Value         float64                `json:"value"`
	Duration      time.Duration          `json:"duration"`
	Labels        map[string]string      `json:"labels,omitempty"`
	Annotations   map[string]string      `json:"annotations,omitempty"`
}

// O-RAN specific monitoring structures
type ORANPerformanceData struct {
	NodeType    string                 `json:"node_type"`    // "CU", "DU", "RU"
	NodeID      string                 `json:"node_id"`
	Metrics     []*MetricValue         `json:"metrics"`
	KPIs        map[string]float64     `json:"kpis"`
	Timestamp   time.Time              `json:"timestamp"`
	ReportingInterval time.Duration   `json:"reporting_interval"`
}

type ORANFaultData struct {
	NodeType      string            `json:"node_type"`
	NodeID        string            `json:"node_id"`
	AlarmID       string            `json:"alarm_id"`
	AlarmType     string            `json:"alarm_type"`
	Severity      Severity          `json:"severity"`
	ProbableCause string            `json:"probable_cause"`
	Description   string            `json:"description"`
	RaisedAt      time.Time         `json:"raised_at"`
	ClearedAt     *time.Time        `json:"cleared_at,omitempty"`
	AdditionalInfo map[string]string `json:"additional_info,omitempty"`
}

// 5G Core monitoring structures
type CoreNetworkFunction struct {
	Type        string            `json:"type"`      // "AMF", "SMF", "UPF", etc.
	Instance    string            `json:"instance"`
	Status      string            `json:"status"`
	Load        float64           `json:"load"`
	Capacity    float64           `json:"capacity"`
	Connections int               `json:"connections"`
	Throughput  float64           `json:"throughput"`
	Latency     float64           `json:"latency"`
	ErrorRate   float64           `json:"error_rate"`
	Timestamp   time.Time         `json:"timestamp"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type SlicePerformance struct {
	SliceID         string            `json:"slice_id"`
	ServiceType     string            `json:"service_type"`  // "eMBB", "URLLC", "mMTC"
	Throughput      float64           `json:"throughput"`
	Latency         float64           `json:"latency"`
	PacketLoss      float64           `json:"packet_loss"`
	Availability    float64           `json:"availability"`
	ActiveSessions  int               `json:"active_sessions"`
	QoSCompliance   float64           `json:"qos_compliance"`
	ResourceUsage   float64           `json:"resource_usage"`
	Timestamp       time.Time         `json:"timestamp"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// Event Processing interfaces
type EventProcessor interface {
	ProcessEvent(ctx context.Context, event interface{}) error
	GetSupportedEventTypes() []string
}

type EventRouter interface {
	RouteEvent(ctx context.Context, event interface{}) error
	RegisterProcessor(eventType string, processor EventProcessor) error
	UnregisterProcessor(eventType string) error
}

// Monitoring aggregation and analytics
type MetricAggregator interface {
	Aggregate(ctx context.Context, metrics []*MetricValue, window time.Duration) ([]*MetricValue, error)
	GetAggregatedMetrics(ctx context.Context, name string, window time.Duration) ([]*MetricValue, error)
}

type AlertAnalyzer interface {
	AnalyzeAlert(ctx context.Context, alert *Alert) (*AlertAnalysis, error)
	GetTrends(ctx context.Context, timeRange time.Duration) (*AlertTrends, error)
}

type AlertAnalysis struct {
	AlertID           string                 `json:"alert_id"`
	Severity          AlertSeverity          `json:"severity"`
	Impact            string                 `json:"impact"`
	RootCause         string                 `json:"root_cause,omitempty"`
	SuggestedActions  []string               `json:"suggested_actions"`
	RelatedAlerts     []string               `json:"related_alerts"`
	AffectedServices  []string               `json:"affected_services"`
	Confidence        float64                `json:"confidence"`
	AnalyzedAt        time.Time              `json:"analyzed_at"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

type AlertTrends struct {
	TimeRange         time.Duration          `json:"time_range"`
	TotalAlerts       int                    `json:"total_alerts"`
	AlertsBySeverity  map[AlertSeverity]int  `json:"alerts_by_severity"`
	AlertsByComponent map[string]int         `json:"alerts_by_component"`
	TopAlertRules     []string               `json:"top_alert_rules"`
	TrendDirection    string                 `json:"trend_direction"` // "increasing", "decreasing", "stable"
	AnalyzedAt        time.Time              `json:"analyzed_at"`
}

// TraceSpan represents a tracing span for distributed tracing
type TraceSpan struct {
	TraceID       string                 `json:"trace_id"`
	SpanID        string                 `json:"span_id"`
	ParentSpanID  string                 `json:"parent_span_id,omitempty"`
	OperationName string                 `json:"operation_name"`
	StartTime     time.Time              `json:"start_time"`
	EndTime       time.Time              `json:"end_time"`
	Duration      time.Duration          `json:"duration"`
	Tags          map[string]interface{} `json:"tags,omitempty"`
	Logs          []TraceLog             `json:"logs,omitempty"`
	Status        TraceStatus            `json:"status"`
}

// TraceLog represents a log entry within a trace span
type TraceLog struct {
	Timestamp time.Time              `json:"timestamp"`
	Fields    map[string]interface{} `json:"fields"`
}

// TraceStatus represents the status of a trace span
type TraceStatus struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
}

// ComponentHealth represents the health status of a component
type ComponentHealth struct {
	Component   string                 `json:"component"`
	Status      string                 `json:"status"`
	Message     string                 `json:"message,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Duration    time.Duration          `json:"duration"`
}

// PerformanceMetrics represents performance metrics for system components
type PerformanceMetrics struct {
	Timestamp        time.Time              `json:"timestamp"`
	CPUUsage         float64                `json:"cpu_usage,omitempty"`
	MemoryUsage      float64                `json:"memory_usage,omitempty"`
	DiskUsage        float64                `json:"disk_usage,omitempty"`
	NetworkThroughput float64               `json:"network_throughput,omitempty"`
	ResponseTime     time.Duration          `json:"response_time,omitempty"`
	ErrorRate        float64                `json:"error_rate,omitempty"`
	Availability     float64                `json:"availability,omitempty"`
	CustomMetrics    map[string]interface{} `json:"custom_metrics,omitempty"`
}

// Note: MetricsRecorder is defined in metrics_recorder.go

// MetricsData represents raw metrics data
type MetricsData struct {
	Name      string                 `json:"name"`
	Value     float64                `json:"value"`
	Labels    map[string]string      `json:"labels,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NWDAFAnalytics represents NWDAF analytics functionality
type NWDAFAnalytics interface {
	CollectAnalytics(ctx context.Context) (*AnalyticsResult, error)
	ProcessData(data []MetricsData) (*AnalyticsResult, error)
	GetAnalyticsType() NWDAFAnalyticsType
}

// NWDAFAnalyticsType represents the type of analytics
type NWDAFAnalyticsType string

const (
	NWDAFAnalyticsTypeLoad       NWDAFAnalyticsType = "load"
	NWDAFAnalyticsTypeQoS        NWDAFAnalyticsType = "qos"
	NWDAFAnalyticsTypeSlice      NWDAFAnalyticsType = "slice"
	NWDAFAnalyticsTypeUE         NWDAFAnalyticsType = "ue"
	NWDAFAnalyticsTypeUEMobility NWDAFAnalyticsType = "ue_mobility"
)

// AnalyticsResult represents the result of analytics processing
type AnalyticsResult struct {
	Type      NWDAFAnalyticsType     `json:"type"`
	Result    map[string]interface{} `json:"result"`
	Timestamp time.Time              `json:"timestamp"`
	Confidence float64               `json:"confidence"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// SeasonalityDetector provides seasonality detection for time series data
type SeasonalityDetector interface {
	DetectSeasonality(data []MetricsData) (*SeasonalityResult, error)
	AnalyzeTrend(data []MetricsData) (*TrendResult, error)
}

// SeasonalityResult represents the result of seasonality detection
type SeasonalityResult struct {
	HasSeasonality bool                   `json:"has_seasonality"`
	Period         time.Duration          `json:"period,omitempty"`
	Strength       float64                `json:"strength"`
	Components     map[string]interface{} `json:"components,omitempty"`
	Timestamp      time.Time              `json:"timestamp"`
}

// TrendResult represents the result of trend analysis
type TrendResult struct {
	Direction string    `json:"direction"` // "increasing", "decreasing", "stable"
	Slope     float64   `json:"slope"`
	RSquared  float64   `json:"r_squared"`
	Timestamp time.Time `json:"timestamp"`
}

// SystemHealth represents the overall system health
type SystemHealth struct {
	OverallStatus string             `json:"overall_status"`
	Components    []ComponentHealth  `json:"components"`
	LastChecked   time.Time         `json:"last_checked"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ReportConfig represents configuration for performance reports
type ReportConfig struct {
	OutputFormat     string        `json:"output_format"` // "json", "html", "pdf"
	TimeRange        TimeRange     `json:"time_range"`
	IncludeMetrics   []string      `json:"include_metrics,omitempty"`
	GroupBy          []string      `json:"group_by,omitempty"`
	Aggregation      string        `json:"aggregation,omitempty"` // "avg", "sum", "max", "min"
	IncludeAlerts    bool          `json:"include_alerts"`
	Filters          map[string]string `json:"filters,omitempty"`
}

// PerformanceReport represents a generated performance report
type PerformanceReport struct {
	ID           string                 `json:"id"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description,omitempty"`
	TimeRange    TimeRange              `json:"time_range"`
	GeneratedAt  time.Time              `json:"generated_at"`
	Metrics      []PerformanceMetrics   `json:"metrics"`
	Alerts       []AlertItem            `json:"alerts,omitempty"`
	Summary      map[string]interface{} `json:"summary"`
	Config       ReportConfig           `json:"config"`
}

// TimeRange represents a time range for queries and reports
type TimeRange struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// AlertItem represents an alert in reports and listings
type AlertItem struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Severity    AlertSeverity          `json:"severity"`
	Status      AlertStatus            `json:"status"`
	Component   string                 `json:"component"`
	Message     string                 `json:"message"`
	Timestamp   time.Time              `json:"timestamp"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}