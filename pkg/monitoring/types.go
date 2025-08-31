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
	CommonEventHeader CommonEventHeader `json:"commonEventHeader"`
	MeasurementFields *MeasurementFields `json:"measurementFields,omitempty"`
	FaultFields       *FaultFields       `json:"faultFields,omitempty"`
	ThresholdFields   *ThresholdFields   `json:"thresholdCrossingAlertFields,omitempty"`
}

type CommonEventHeader struct {
	Domain                 string    `json:"domain"`
	EventID                string    `json:"eventId"`
	EventName              string    `json:"eventName"`
	EventType              string    `json:"eventType"`
	LastEpochMicrosec      int64     `json:"lastEpochMicrosec"`
	Priority               string    `json:"priority"`
	ReportingEntityID      string    `json:"reportingEntityId"`
	ReportingEntityName    string    `json:"reportingEntityName"`
	Sequence               int       `json:"sequence"`
	SourceID               string    `json:"sourceId"`
	SourceName             string    `json:"sourceName"`
	StartEpochMicrosec     int64     `json:"startEpochMicrosec"`
	Version                string    `json:"version"`
	VesEventListenerVersion string   `json:"vesEventListenerVersion"`
	TimeZoneOffset         string    `json:"timeZoneOffset"`
	NfNamingCode          string    `json:"nfNamingCode,omitempty"`
	NfVendorName          string    `json:"nfVendorName,omitempty"`
}

type MeasurementFields struct {
	MeasurementInterval    int64                    `json:"measurementInterval"`
	MeasurementFieldsVersion string                `json:"measurementFieldsVersion"`
	AdditionalMeasurements []AdditionalMeasurement `json:"additionalMeasurements,omitempty"`
	CPUUsageArray         []CPUUsage               `json:"cpuUsageArray,omitempty"`
	MemoryUsageArray      []MemoryUsage            `json:"memoryUsageArray,omitempty"`
	NetworkUsageArray     []NetworkUsage           `json:"networkUsageArray,omitempty"`
}

type FaultFields struct {
	FaultFieldsVersion   string `json:"faultFieldsVersion"`
	AlarmCondition       string `json:"alarmCondition"`
	AlarmInterfaceA      string `json:"alarmInterfaceA,omitempty"`
	EventCategory        string `json:"eventCategory,omitempty"`
	EventSeverity        string `json:"eventSeverity"`
	EventSourceType      string `json:"eventSourceType"`
	SpecificProblem      string `json:"specificProblem"`
	VfStatus             string `json:"vfStatus"`
}

type ThresholdFields struct {
	ThresholdFieldsVersion    string                 `json:"thresholdCrossingFieldsVersion"`
	AlertAction               string                 `json:"alertAction"`
	AlertDescription          string                 `json:"alertDescription"`
	AlertType                 string                 `json:"alertType"`
	CollectionTimestamp       string                 `json:"collectionTimestamp"`
	EventSeverity             string                 `json:"eventSeverity"`
	EventStartTimestamp       string                 `json:"eventStartTimestamp"`
	AdditionalParameters      []AdditionalParameter  `json:"additionalParameters,omitempty"`
}

type AdditionalMeasurement struct {
	Name   string            `json:"name"`
	Values map[string]string `json:"hashMap"`
}

type AdditionalParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type CPUUsage struct {
	CPUID           string  `json:"cpuIdentifier"`
	PercentUsage    float64 `json:"percentUsage"`
	CPUCapacityMhz  float64 `json:"cpuCapacityMhz,omitempty"`
}

type MemoryUsage struct {
	MemoryFree      float64 `json:"memoryFree"`
	MemoryUsed      float64 `json:"memoryUsed"`
	MemoryBuffered  float64 `json:"memoryBuffered,omitempty"`
	MemoryCached    float64 `json:"memoryCached,omitempty"`
}

type NetworkUsage struct {
	NetworkInterface   string  `json:"networkInterface"`
	BytesIn           float64 `json:"bytesIn"`
	BytesOut          float64 `json:"bytesOut"`
	PacketsIn         float64 `json:"packetsIn"`
	PacketsOut        float64 `json:"packetsOut"`
}

// Health monitoring structures
type ComponentHealth struct {
	Name      string            `json:"name"`
	Status    HealthStatus      `json:"status"`
	Message   string            `json:"message"`
	Timestamp time.Time         `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

type SystemHealth struct {
	OverallStatus HealthStatus      `json:"overallStatus"`
	Components    []ComponentHealth `json:"components"`
	LastChecked   time.Time         `json:"lastChecked"`
}

// Performance metrics structures
type PerformanceMetrics struct {
	Timestamp   time.Time         `json:"timestamp"`
	CPU         CPUMetrics        `json:"cpu"`
	Memory      MemoryMetrics     `json:"memory"`
	Network     NetworkMetrics    `json:"network"`
	Application AppMetrics        `json:"application"`
	Custom      map[string]interface{} `json:"custom,omitempty"`
}

type CPUMetrics struct {
	UsagePercent float64 `json:"usagePercent"`
	LoadAverage  float64 `json:"loadAverage"`
	Cores        int     `json:"cores"`
}

type MemoryMetrics struct {
	UsedBytes      int64   `json:"usedBytes"`
	TotalBytes     int64   `json:"totalBytes"`
	UsagePercent   float64 `json:"usagePercent"`
	CachedBytes    int64   `json:"cachedBytes"`
	BufferedBytes  int64   `json:"bufferedBytes"`
}

type NetworkMetrics struct {
	BytesReceived    int64 `json:"bytesReceived"`
	BytesSent        int64 `json:"bytesSent"`
	PacketsReceived  int64 `json:"packetsReceived"`
	PacketsSent      int64 `json:"packetsSent"`
	ErrorsReceived   int64 `json:"errorsReceived"`
	ErrorsSent       int64 `json:"errorsSent"`
}

type AppMetrics struct {
	RequestsTotal    int64   `json:"requestsTotal"`
	RequestsPerSec   float64 `json:"requestsPerSecond"`
	ResponseTimeAvg  float64 `json:"responseTimeAvgMs"`
	ErrorRate        float64 `json:"errorRate"`
	ActiveConnections int64  `json:"activeConnections"`
}

// Notification structures
type NotificationChannel struct {
	Name     string                 `json:"name"`
	Type     NotificationType       `json:"type"`
	Config   map[string]interface{} `json:"config"`
	Enabled  bool                   `json:"enabled"`
	Filters  []NotificationFilter   `json:"filters,omitempty"`
}

type NotificationType string

const (
	NotificationTypeEmail    NotificationType = "email"
	NotificationTypeSlack    NotificationType = "slack"
	NotificationTypeWebhook  NotificationType = "webhook"
	NotificationTypeSMS      NotificationType = "sms"
)

type NotificationFilter struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// Interface definitions
type HealthChecker interface {
	CheckHealth(ctx context.Context) (*ComponentHealth, error)
	GetName() string
}

type MetricsCollector interface {
	CollectMetrics(ctx context.Context) (*PerformanceMetrics, error)
	GetName() string
}

type VESEventSender interface {
	SendEvent(ctx context.Context, event *VESEvent) error
	SendBatch(ctx context.Context, events []*VESEvent) error
}

type AlertManager interface {
	SendAlert(ctx context.Context, alert Alert) error
	GetAlerts(ctx context.Context, filters map[string]string) ([]Alert, error)
}

// Alert structure
type Alert struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Severity    AlertSeverity          `json:"severity"`
	Status      AlertStatus            `json:"status"`
	Labels      map[string]string      `json:"labels"`
	Annotations map[string]string      `json:"annotations"`
	StartsAt    time.Time              `json:"startsAt"`
	EndsAt      *time.Time             `json:"endsAt,omitempty"`
	GeneratorURL string                `json:"generatorURL,omitempty"`
	Fingerprint  string                `json:"fingerprint"`
}

type AlertStatus string

const (
	AlertStatusFiring   AlertStatus = "firing"
	AlertStatusResolved AlertStatus = "resolved"
	AlertStatusSuppressed AlertStatus = "suppressed"
)

// NWDAF Analytics structures
type NWDAFAnalytics struct {
	AnalyticsID   string                 `json:"analyticsId"`
	AnalyticsType NWDAFAnalyticsType     `json:"analyticsType"`
	Timestamp     time.Time              `json:"timestamp"`
	Data          map[string]interface{} `json:"data"`
	Confidence    float64                `json:"confidence"`
	Validity      time.Duration          `json:"validity"`
}

type NWDAFAnalyticsType string

const (
	NWDAFAnalyticsTypeLoadLevel     NWDAFAnalyticsType = "LOAD_LEVEL_INFORMATION"
	NWDAFAnalyticsTypeNetworkPerf   NWDAFAnalyticsType = "NETWORK_PERFORMANCE"
	NWDAFAnalyticsTypeNFLoad        NWDAFAnalyticsType = "NF_LOAD"
	NWDAFAnalyticsTypeServiceExp    NWDAFAnalyticsType = "SERVICE_EXPERIENCE"
	NWDAFAnalyticsTypeUEMobility    NWDAFAnalyticsType = "UE_MOBILITY"
	NWDAFAnalyticsTypeUEComm        NWDAFAnalyticsType = "UE_COMMUNICATION"
)

// Distributed tracing structures
type TraceSpan struct {
	TraceID      string            `json:"traceId"`
	SpanID       string            `json:"spanId"`
	ParentSpanID string            `json:"parentSpanId,omitempty"`
	OperationName string           `json:"operationName"`
	StartTime    time.Time         `json:"startTime"`
	FinishTime   *time.Time        `json:"finishTime,omitempty"`
	Duration     time.Duration     `json:"duration"`
	Tags         map[string]string `json:"tags,omitempty"`
	Status       SpanStatus        `json:"status"`
	ServiceName  string            `json:"serviceName"`
}

type SpanStatus string

const (
	SpanStatusOK    SpanStatus = "OK"
	SpanStatusError SpanStatus = "ERROR"
)

type TraceCollector interface {
	CollectSpan(ctx context.Context, span *TraceSpan) error
	GetTrace(ctx context.Context, traceID string) ([]*TraceSpan, error)
}