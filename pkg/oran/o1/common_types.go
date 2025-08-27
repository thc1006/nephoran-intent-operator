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
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	ReportType  string        `json:"report_type"`
	Schedule    string        `json:"schedule"` // CRON expression
	Template    string        `json:"template"`
	Recipients  []string      `json:"recipients"`
	Parameters  map[string]interface{} `json:"parameters"`
	Enabled     bool          `json:"enabled"`
	NextRun     time.Time     `json:"next_run"`
	LastRun     time.Time     `json:"last_run"`
	RunCount    int64         `json:"run_count"`
	Status      string        `json:"status"` // ACTIVE, INACTIVE, ERROR
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
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