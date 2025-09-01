package o1

import (
	"context"
	"time"
)

// Common interfaces and types shared across O1 managers to avoid duplicates

// ReportDistributor interface for distributing reports across different channels
type ReportDistributor interface {
	DistributeReport(ctx context.Context, report *Report) error
	GetSupportedChannels() []string
	AddDistributionChannel(channel DistributionChannel) error
	RemoveDistributionChannel(channelID string) error
}

// DistributionChannel interface for report distribution channels
type DistributionChannel interface {
	GetID() string
	GetType() string // EMAIL, API, FILE, etc.
	SendReport(ctx context.Context, report *Report) error
	IsEnabled() bool
	GetConfiguration() map[string]interface{}
}

// Report represents a generic report structure
type Report struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // PERFORMANCE, ACCOUNTING, SECURITY, FAULT
	Title       string                 `json:"title"`
	Content     interface{}            `json:"content"`
	Format      string                 `json:"format"` // JSON, XML, PDF, HTML
	GeneratedAt time.Time              `json:"generated_at"`
	ExpiresAt   time.Time              `json:"expires_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
	Size        int64                  `json:"size"`
}

// ReportSchedule represents a report generation schedule (defined here to avoid duplicates)
type ReportSchedule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	ReportType  string                 `json:"report_type"`
	Schedule    string                 `json:"schedule"` // CRON expression
	Template    string                 `json:"template"`
	Recipients  []string               `json:"recipients"`
	Parameters  map[string]interface{} `json:"parameters"`
	Enabled     bool                   `json:"enabled"`
	NextRun     time.Time              `json:"next_run"`
	LastRun     time.Time              `json:"last_run"`
	RunCount    int64                  `json:"run_count"`
	Status      string                 `json:"status"` // ACTIVE, INACTIVE, ERROR
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// Notification represents a generic notification message
type Notification struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // ALARM, CONFIG, PERFORMANCE
	Severity    string                 `json:"severity"` // CRITICAL, MAJOR, MINOR, WARNING, CLEAR
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Source      string                 `json:"source"`
	Target      string                 `json:"target,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	AckRequired bool                   `json:"ack_required"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NotificationChannel interface for sending notifications (used by multiple managers)
type NotificationChannel interface {
	GetID() string
	GetType() string // EMAIL, SMS, WEBHOOK, etc.
	SendNotification(ctx context.Context, notification *Notification) error
	IsEnabled() bool
	GetConfiguration() map[string]interface{}
}

// EventCallback is a generic callback function for events
type EventCallback func(event interface{})

// SecurityPolicy represents security policy configuration (avoid duplicate in o1_adaptor.go)
type SecurityPolicy struct {
	PolicyID    string                 `json:"policy_id"`
	PolicyType  string                 `json:"policy_type"`
	Rules       []SecurityRule         `json:"rules"`
	Enforcement string                 `json:"enforcement"` // STRICT, PERMISSIVE
	ValidFrom   time.Time              `json:"valid_from"`
	ValidUntil  time.Time              `json:"valid_until"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// SecurityRule represents a security rule (avoid duplicate)
type SecurityRule struct {
	RuleID      string                 `json:"rule_id"`
	Action      string                 `json:"action"` // ALLOW, DENY, LOG
	Conditions  map[string]interface{} `json:"conditions"`
	Priority    int                    `json:"priority"`
	Description string                 `json:"description"`
	Enabled     bool                   `json:"enabled"`
}

// Common report distributor implementation
type DefaultReportDistributor struct {
	channels map[string]DistributionChannel
	config   *DistributorConfig
}

type DistributorConfig struct {
	MaxRetries      int           `json:"max_retries"`
	RetryInterval   time.Duration `json:"retry_interval"`
	TimeoutDuration time.Duration `json:"timeout_duration"`
	BufferSize      int           `json:"buffer_size"`
}

func NewDefaultReportDistributor(config *DistributorConfig) *DefaultReportDistributor {
	if config == nil {
		config = &DistributorConfig{
			MaxRetries:      3,
			RetryInterval:   5 * time.Second,
			TimeoutDuration: 30 * time.Second,
			BufferSize:      1000,
		}
	}

	return &DefaultReportDistributor{
		channels: make(map[string]DistributionChannel),
		config:   config,
	}
}

func (d *DefaultReportDistributor) DistributeReport(ctx context.Context, report *Report) error {
	for _, channel := range d.channels {
		if !channel.IsEnabled() {
			continue
		}

		// Distribute to channel with retry logic
		go func(ch DistributionChannel) {
			for attempt := 1; attempt <= d.config.MaxRetries; attempt++ {
				if err := ch.SendReport(ctx, report); err == nil {
					return
				}
				if attempt < d.config.MaxRetries {
					time.Sleep(d.config.RetryInterval)
				}
			}
		}(channel)
	}
	return nil
}

func (d *DefaultReportDistributor) GetSupportedChannels() []string {
	channels := make([]string, 0, len(d.channels))
	for _, channel := range d.channels {
		channels = append(channels, channel.GetType())
	}
	return channels
}

func (d *DefaultReportDistributor) AddDistributionChannel(channel DistributionChannel) error {
	d.channels[channel.GetID()] = channel
	return nil
}

func (d *DefaultReportDistributor) RemoveDistributionChannel(channelID string) error {
	delete(d.channels, channelID)
	return nil
}

// Email distribution channel implementation
type EmailDistributionChannel struct {
	id      string
	config  map[string]interface{}
	enabled bool
}

func NewEmailDistributionChannel(id string, config map[string]interface{}) *EmailDistributionChannel {
	return &EmailDistributionChannel{
		id:      id,
		config:  config,
		enabled: true,
	}
}

func (e *EmailDistributionChannel) GetID() string {
	return e.id
}

func (e *EmailDistributionChannel) GetType() string {
	return "EMAIL"
}

func (e *EmailDistributionChannel) SendReport(ctx context.Context, report *Report) error {
	// Placeholder implementation - would integrate with email service
	return nil
}

func (e *EmailDistributionChannel) IsEnabled() bool {
	return e.enabled
}

func (e *EmailDistributionChannel) GetConfiguration() map[string]interface{} {
	return e.config
}

// API distribution channel implementation
type APIDistributionChannel struct {
	id       string
	endpoint string
	config   map[string]interface{}
	enabled  bool
}

func NewAPIDistributionChannel(id, endpoint string, config map[string]interface{}) *APIDistributionChannel {
	return &APIDistributionChannel{
		id:       id,
		endpoint: endpoint,
		config:   config,
		enabled:  true,
	}
}

func (a *APIDistributionChannel) GetID() string {
	return a.id
}

func (a *APIDistributionChannel) GetType() string {
	return "API"
}

func (a *APIDistributionChannel) SendReport(ctx context.Context, report *Report) error {
	// Placeholder implementation - would make HTTP POST to endpoint
	return nil
}

func (a *APIDistributionChannel) IsEnabled() bool {
	return a.enabled
}

func (a *APIDistributionChannel) GetConfiguration() map[string]interface{} {
	return a.config
}

// Default notification channel implementations
type DefaultNotificationChannel struct {
	id      string
	chType  string
	config  map[string]interface{}
	enabled bool
}

func NewDefaultNotificationChannel(id, chType string, config map[string]interface{}) *DefaultNotificationChannel {
	return &DefaultNotificationChannel{
		id:      id,
		chType:  chType,
		config:  config,
		enabled: true,
	}
}

func (n *DefaultNotificationChannel) GetID() string {
	return n.id
}

func (n *DefaultNotificationChannel) GetType() string {
	return n.chType
}

func (n *DefaultNotificationChannel) SendNotification(ctx context.Context, notification *Notification) error {
	// Placeholder implementation
	return nil
}

func (n *DefaultNotificationChannel) IsEnabled() bool {
	return n.enabled
}

func (n *DefaultNotificationChannel) GetConfiguration() map[string]interface{} {
	return n.config
}

// Network Function Types (needed by adapter.go)

// NetworkFunction represents a network function in the system
type NetworkFunction struct {
	ID              string                 `json:"id"`
	NFID            string                 `json:"nfid"`
	Name            string                 `json:"name"`
	Type            string                 `json:"type"`
	Version         string                 `json:"version"`
	Status          string                 `json:"status"`
	NFStatus        string                 `json:"nf_status"`
	Configuration   map[string]interface{} `json:"configuration"`
	NFServices      []interface{}          `json:"nf_services,omitempty"`
	HeartBeatTimer  int                    `json:"heartbeat_timer,omitempty"`
	LastUpdated     time.Time              `json:"last_updated"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// NetworkFunctionUpdate represents updates to a network function
type NetworkFunctionUpdate struct {
	Name            *string                `json:"name,omitempty"`
	Status          *string                `json:"status,omitempty"`
	NFStatus        *string                `json:"nf_status,omitempty"`
	HeartBeatTimer  *int                   `json:"heartbeat_timer,omitempty"`
	NFServices      []interface{}          `json:"nf_services,omitempty"`
	Configuration   map[string]interface{} `json:"configuration,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// DiscoveryCriteria represents criteria for discovering network functions
type DiscoveryCriteria struct {
	Type            string                 `json:"type,omitempty"`
	Status          string                 `json:"status,omitempty"`
	NamePattern     string                 `json:"name_pattern,omitempty"`
	Filters         map[string]interface{} `json:"filters,omitempty"`
}

// NetworkFunctionStatus represents the status of a network function
type NetworkFunctionStatus struct {
	ID              string                 `json:"id"`
	Status          string                 `json:"status"`
	Health          string                 `json:"health"`
	LastSeen        time.Time              `json:"last_seen"`
	Metrics         map[string]interface{} `json:"metrics,omitempty"`
	Errors          []string               `json:"errors,omitempty"`
}

// NetworkFunctionConfig represents configuration for a network function
type NetworkFunctionConfig struct {
	ID              string                 `json:"id"`
	Configuration   map[string]interface{} `json:"configuration"`
	Schema          map[string]interface{} `json:"schema,omitempty"`
	Version         string                 `json:"version"`
	AppliedAt       time.Time              `json:"applied_at"`
}

// NotificationTemplate represents a notification template
type NotificationTemplate struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Type            string                 `json:"type"`
	Template        string                 `json:"template"`
	Subject         string                 `json:"subject,omitempty"`
	Body            string                 `json:"body,omitempty"`
	Variables       map[string]interface{} `json:"variables,omitempty"`
}

// Alarm represents a basic alarm structure
type Alarm struct {
	ID                 string                 `json:"id"`
	Type               string                 `json:"type"`
	Severity           string                 `json:"severity"`
	Source             string                 `json:"source"`
	Description        string                 `json:"description"`
	Timestamp          time.Time              `json:"timestamp"`
	Acknowledged       bool                   `json:"acknowledged"`
	AlarmID            string                 `json:"alarm_id,omitempty"`
	ObjectClass        string                 `json:"object_class,omitempty"`
	ObjectInstance     string                 `json:"object_instance,omitempty"`
	EventType          string                 `json:"event_type,omitempty"`
	ProbableCause      string                 `json:"probable_cause,omitempty"`
	SpecificProblem    string                 `json:"specific_problem,omitempty"`
	PerceivedSeverity  string                 `json:"perceived_severity,omitempty"`
	AlarmRaisedTime    time.Time              `json:"alarm_raised_time,omitempty"`
	AdditionalText     string                 `json:"additional_text,omitempty"`
	AckState           string                 `json:"ack_state,omitempty"`
	AlarmState         string                 `json:"alarm_state,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// Additional missing types for O1 system

// PerformanceThreshold represents a performance threshold
type PerformanceThreshold struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Metric      string                 `json:"metric"`
	Operator    string                 `json:"operator"`
	Value       float64                `json:"value"`
	Enabled     bool                   `json:"enabled"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// EscalationRule represents an escalation rule
type EscalationRule struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Condition       string                 `json:"condition"`
	Action          string                 `json:"action"`
	TargetLevel     int                    `json:"target_level"`
	DelayMinutes    int                    `json:"delay_minutes"`
	Enabled         bool                   `json:"enabled"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// SMOIntegrationLayer represents the SMO integration layer
type SMOIntegrationLayer struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Endpoint        string                 `json:"endpoint"`
	Status          string                 `json:"status"`
	Version         string                 `json:"version"`
	Capabilities    []string               `json:"capabilities,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// FunctionType represents a function type
type FunctionType struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Category        string                 `json:"category"`
	Version         string                 `json:"version"`
	Specifications  map[string]interface{} `json:"specifications,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// DeploymentTemplate represents a deployment template
type DeploymentTemplate struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Type            string                 `json:"type"`
	Template        map[string]interface{} `json:"template"`
	Parameters      map[string]interface{} `json:"parameters,omitempty"`
	Version         string                 `json:"version"`
	CreatedAt       time.Time              `json:"created_at"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// ConfigRequest represents a configuration request
type ConfigRequest struct {
	ID              string                 `json:"id"`
	Type            string                 `json:"type"`
	Target          string                 `json:"target"`
	Configuration   map[string]interface{} `json:"configuration"`
	RequestedBy     string                 `json:"requested_by"`
	RequestedAt     time.Time              `json:"requested_at"`
	Priority        string                 `json:"priority,omitempty"`
	ObjectInstance  string                 `json:"object_instance,omitempty"`
	Attributes      map[string]interface{} `json:"attributes,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// ConfigResponse represents a configuration response
type ConfigResponse struct {
	ID              string                 `json:"id"`
	RequestID       string                 `json:"request_id"`
	Status          string                 `json:"status"`
	Result          map[string]interface{} `json:"result,omitempty"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	ProcessedAt     time.Time              `json:"processed_at"`
	ObjectInstance  string                 `json:"object_instance,omitempty"`
	Attributes      map[string]interface{} `json:"attributes,omitempty"`
	Timestamp       time.Time              `json:"timestamp"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// Additional client types

// PerformanceRequest represents a performance request
type PerformanceRequest struct {
	ID              string                 `json:"id"`
	MetricType      string                 `json:"metric_type"`
	Target          string                 `json:"target"`
	TimeRange       map[string]interface{} `json:"time_range,omitempty"`
	Filters         map[string]interface{} `json:"filters,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// PerformanceResponse represents a performance response
type PerformanceResponse struct {
	ID              string                 `json:"id"`
	RequestID       string                 `json:"request_id"`
	Metrics         map[string]interface{} `json:"metrics"`
	Status          string                 `json:"status"`
	Timestamp       time.Time              `json:"timestamp"`
	PerformanceData []PerformanceData      `json:"performance_data,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// PerformanceSubscription represents a performance subscription
type PerformanceSubscription struct {
	ID              string                 `json:"id"`
	MetricType      string                 `json:"metric_type"`
	Callback        string                 `json:"callback"`
	Filters         map[string]interface{} `json:"filters,omitempty"`
	Status          string                 `json:"status"`
	CreatedAt       time.Time              `json:"created_at"`
	ReportingPeriod time.Duration          `json:"reporting_period,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// PerformanceMeasurement represents a performance measurement
type PerformanceMeasurement struct {
	ID               string                 `json:"id"`
	SubscriptionID   string                 `json:"subscription_id"`
	MetricValue      float64                `json:"metric_value"`
	Timestamp        time.Time              `json:"timestamp"`
	Source           string                 `json:"source"`
	ObjectInstance   string                 `json:"object_instance,omitempty"`
	MeasurementType  string                 `json:"measurement_type,omitempty"`
	Value            float64                `json:"value,omitempty"`
	Unit             string                 `json:"unit,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// AlarmFilter represents an alarm filter
type AlarmFilter struct {
	Severity        string                 `json:"severity,omitempty"`
	Source          string                 `json:"source,omitempty"`
	TimeRange       map[string]interface{} `json:"time_range,omitempty"`
	Status          string                 `json:"status,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// AlarmResponse represents an alarm response
type AlarmResponse struct {
	Alarms          []Alarm                `json:"alarms"`
	Total           int                    `json:"total"`
	RequestID       string                 `json:"request_id,omitempty"`
	Status          string                 `json:"status"`
	Timestamp       time.Time              `json:"timestamp"` 
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// FileUploadRequest represents a file upload request
type FileUploadRequest struct {
	ID              string                 `json:"id"`
	Filename        string                 `json:"filename"`
	FileType        string                 `json:"file_type"`
	Size            int64                  `json:"size"`
	Checksum        string                 `json:"checksum,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// FileUploadResponse represents a file upload response
type FileUploadResponse struct {
	ID              string                 `json:"id"`
	RequestID       string                 `json:"request_id"`
	Status          string                 `json:"status"`
	UploadURL       string                 `json:"upload_url,omitempty"`
	FileID          string                 `json:"file_id,omitempty"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// FileDownloadResponse represents a file download response
type FileDownloadResponse struct {
	ID              string                 `json:"id"`
	FileID          string                 `json:"file_id"`
	DownloadURL     string                 `json:"download_url"`
	Status          string                 `json:"status"`
	ExpiresAt       time.Time              `json:"expires_at"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// HeartbeatResponse represents a heartbeat response
type HeartbeatResponse struct {
	Status          string                 `json:"status"`
	Timestamp       time.Time              `json:"timestamp"`
	Version         string                 `json:"version"`
	Uptime          time.Duration          `json:"uptime"`
	Health          map[string]interface{} `json:"health,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// PerformanceData represents performance data
type PerformanceData struct {
	ID              string                 `json:"id"`
	Timestamp       time.Time              `json:"timestamp"`
	Metrics         map[string]interface{} `json:"metrics"`
	Source          string                 `json:"source"`
	DataType        string                 `json:"data_type"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}
