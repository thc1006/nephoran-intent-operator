package o2providers

import (
	"time"
)

// EventType represents the type of provider event
type EventType string

const (
	// Resource lifecycle events
	EventTypeResourceCreated EventType = "resource.created"
	EventTypeResourceUpdated EventType = "resource.updated"
	EventTypeResourceDeleted EventType = "resource.deleted"
	EventTypeResourceFailed  EventType = "resource.failed"

	// Provider events
	EventTypeProviderStarted EventType = "provider.started"
	EventTypeProviderStopped EventType = "provider.stopped"
	EventTypeProviderError   EventType = "provider.error"

	// Operational events
	EventTypeScaleUp        EventType = "scale.up"
	EventTypeScaleDown      EventType = "scale.down"
	EventTypeHealthCheck    EventType = "health.check"
	EventTypeAlertTriggered EventType = "alert.triggered"
)

// EventSeverity represents the severity level of an event
type EventSeverity string

const (
	SeverityInfo     EventSeverity = "info"
	SeverityWarning  EventSeverity = "warning"
	SeverityError    EventSeverity = "error"
	SeverityCritical EventSeverity = "critical"
)

// ResourceEvent represents an event related to a specific resource
type ResourceEvent struct {
	// Event metadata
	ID        string    `json:"id"`
	Type      EventType `json:"type"`
	Timestamp time.Time `json:"timestamp"`

	// Resource information
	ResourceID   string       `json:"resourceId"`
	ResourceType ResourceType `json:"resourceType"`
	ResourceName string       `json:"resourceName"`

	// Event details
	Severity    EventSeverity `json:"severity"`
	Message     string        `json:"message"`
	Description string        `json:"description,omitempty"`

	// Context and metadata
	Context map[string]interface{} `json:"context,omitempty"`
	Labels  map[string]string      `json:"labels,omitempty"`
	Source  string                 `json:"source"` // provider name

	// State change information (if applicable)
	OldState map[string]interface{} `json:"oldState,omitempty"`
	NewState map[string]interface{} `json:"newState,omitempty"`
}

// ProviderEvent represents an event at the provider level
type ProviderEvent struct {
	// Event metadata
	ID        string    `json:"id"`
	Type      EventType `json:"type"`
	Timestamp time.Time `json:"timestamp"`

	// Provider information
	ProviderName string `json:"providerName"`
	ProviderType string `json:"providerType"`

	// Event details
	Severity    EventSeverity `json:"severity"`
	Message     string        `json:"message"`
	Description string        `json:"description,omitempty"`

	// Context and metadata
	Context map[string]interface{} `json:"context,omitempty"`
	Labels  map[string]string      `json:"labels,omitempty"`

	// Error information (if applicable)
	Error *ProviderError `json:"error,omitempty"`
}

// ProviderError represents detailed error information
type ProviderError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Details   string `json:"details,omitempty"`
	Retryable bool   `json:"retryable"`
	Cause     string `json:"cause,omitempty"`
}

// EventSubscription represents a subscription to provider events
type EventSubscription struct {
	ID        string         `json:"id"`
	Name      string         `json:"name,omitempty"`
	Filter    EventFilter    `json:"filter"`
	Webhook   *WebhookConfig `json:"webhook,omitempty"`
	Queue     *QueueConfig   `json:"queue,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	IsActive  bool           `json:"isActive"`
}

// EventFilter defines criteria for filtering events
type EventFilter struct {
	Types         []EventType       `json:"types,omitempty"`
	Severities    []EventSeverity   `json:"severities,omitempty"`
	ResourceTypes []ResourceType    `json:"resourceTypes,omitempty"`
	ResourceIDs   []string          `json:"resourceIds,omitempty"`
	Providers     []string          `json:"providers,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	TimeRange     *TimeRange        `json:"timeRange,omitempty"`
}

// TimeRange represents a time range filter
type TimeRange struct {
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

// WebhookConfig defines webhook configuration for event delivery
type WebhookConfig struct {
	URL         string            `json:"url"`
	Method      string            `json:"method,omitempty"` // defaults to POST
	Headers     map[string]string `json:"headers,omitempty"`
	Timeout     time.Duration     `json:"timeout,omitempty"`
	RetryPolicy *RetryPolicy      `json:"retryPolicy,omitempty"`

	// Security
	SecretToken string `json:"secretToken,omitempty"`
	TLSVerify   bool   `json:"tlsVerify"`
}

// QueueConfig defines message queue configuration for event delivery
type QueueConfig struct {
	Type      string                 `json:"type"` // redis, kafka, sqs, etc.
	Topic     string                 `json:"topic"`
	Config    map[string]interface{} `json:"config"`
	BatchSize int                    `json:"batchSize,omitempty"`
	Timeout   time.Duration          `json:"timeout,omitempty"`
}

// RetryPolicy defines retry behavior for failed event deliveries
type RetryPolicy struct {
	MaxRetries   int           `json:"maxRetries"`
	InitialDelay time.Duration `json:"initialDelay"`
	MaxDelay     time.Duration `json:"maxDelay"`
	Multiplier   float64       `json:"multiplier"`
	RandomJitter bool          `json:"randomJitter"`
}

// EventDeliveryStatus represents the status of event delivery
type EventDeliveryStatus struct {
	EventID      string            `json:"eventId"`
	Subscription string            `json:"subscription"`
	Status       DeliveryStatus    `json:"status"`
	Attempts     int               `json:"attempts"`
	LastAttempt  time.Time         `json:"lastAttempt"`
	NextRetry    *time.Time        `json:"nextRetry,omitempty"`
	Error        string            `json:"error,omitempty"`
	Response     *DeliveryResponse `json:"response,omitempty"`
}

// DeliveryStatus represents the status of event delivery
type DeliveryStatus string

const (
	DeliveryStatusPending   DeliveryStatus = "pending"
	DeliveryStatusDelivered DeliveryStatus = "delivered"
	DeliveryStatusFailed    DeliveryStatus = "failed"
	DeliveryStatusRetrying  DeliveryStatus = "retrying"
	DeliveryStatusAbandoned DeliveryStatus = "abandoned"
)

// DeliveryResponse represents the response from event delivery
type DeliveryResponse struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       string            `json:"body,omitempty"`
	Duration   time.Duration     `json:"duration"`
}
