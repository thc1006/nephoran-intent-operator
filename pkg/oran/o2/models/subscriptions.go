package o2models

import (
	"time"

	"k8s.io/apimachinery/pkg/runtime"
)

// Subscription and Event Management Models following O-RAN.WG6.O2ims-Interface-v01.01

// Subscription represents an event subscription in O2 IMS
type Subscription struct {
	SubscriptionID string `json:"subscriptionId"`
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	CallbackUri    string `json:"callbackUri"`

	// Subscription filter and configuration
	Filter                 *SubscriptionFilter `json:"filter,omitempty"`
	ConsumerSubscriptionId string              `json:"consumerSubscriptionId,omitempty"`

	// Event types and configuration
	EventTypes  []string            `json:"eventTypes"`
	EventConfig *EventConfiguration `json:"eventConfig,omitempty"`

	// Authentication and security
	Authentication *SubscriptionAuth `json:"authentication,omitempty"`

	// Subscription lifecycle
	Status     *SubscriptionStatus    `json:"status"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
	CreatedAt  time.Time              `json:"createdAt"`
	UpdatedAt  time.Time              `json:"updatedAt"`
	CreatedBy  string                 `json:"createdBy,omitempty"`
	UpdatedBy  string                 `json:"updatedBy,omitempty"`
}

// SubscriptionFilter defines filters for event subscriptions
type SubscriptionFilter struct {
	// Resource filters
	ResourceTypes   []string `json:"resourceTypes,omitempty"`
	ResourceIDs     []string `json:"resourceIds,omitempty"`
	ResourcePoolIDs []string `json:"resourcePoolIds,omitempty"`

	// Event filters
	EventSeverities []string `json:"eventSeverities,omitempty"`
	EventSources    []string `json:"eventSources,omitempty"`

	// Geographic filters - using types from resource_types.go
	LocationFilter *LocationFilter `json:"locationFilter,omitempty"`

	// Custom filters
	CustomFilters map[string]interface{} `json:"customFilters,omitempty"`

	// Time filters
	TimeWindow *TimeWindow `json:"timeWindow,omitempty"`
}

// LocationFilter, GeographicArea, and BoundingBox are defined in resource_types.go
// to avoid duplicate declarations across the package
// TimeWindow defines a time-based filter
type TimeWindow struct {
	StartTime  *time.Time       `json:"startTime,omitempty"`
	EndTime    *time.Time       `json:"endTime,omitempty"`
	TimeOfDay  *TimeOfDayWindow `json:"timeOfDay,omitempty"`
	DaysOfWeek []string         `json:"daysOfWeek,omitempty"` // MON, TUE, WED, etc.
	TimeZone   string           `json:"timeZone,omitempty"`
}

// TimeOfDayWindow defines a daily time window
type TimeOfDayWindow struct {
	StartHour   int `json:"startHour"`   // 0-23
	StartMinute int `json:"startMinute"` // 0-59
	EndHour     int `json:"endHour"`     // 0-23
	EndMinute   int `json:"endMinute"`   // 0-59
}

// EventConfiguration defines event delivery configuration
type EventConfiguration struct {
	BatchSize       int           `json:"batchSize,omitempty"`       // Events per batch
	MaxBatchWait    time.Duration `json:"maxBatchWait,omitempty"`    // Maximum wait time for batching
	RetryAttempts   int           `json:"retryAttempts,omitempty"`   // Number of retry attempts
	RetryDelay      time.Duration `json:"retryDelay,omitempty"`      // Delay between retries
	MaxRetryDelay   time.Duration `json:"maxRetryDelay,omitempty"`   // Maximum retry delay
	RetryBackoff    float64       `json:"retryBackoff,omitempty"`    // Retry backoff multiplier
	DeadLetterQueue string        `json:"deadLetterQueue,omitempty"` // DLQ for failed events
	Timeout         time.Duration `json:"timeout,omitempty"`         // Event delivery timeout
}

// SubscriptionAuth defines authentication for subscriptions
type SubscriptionAuth struct {
	Type       string                 `json:"type"` // NONE, BASIC, BEARER, OAUTH2, MTLS
	Username   string                 `json:"username,omitempty"`
	Password   string                 `json:"password,omitempty"`
	Token      string                 `json:"token,omitempty"`
	OAuth2     *OAuth2Config          `json:"oauth2,omitempty"`
	TLS        *TLSConfig             `json:"tls,omitempty"`
	Headers    map[string]string      `json:"headers,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// OAuth2Config defines OAuth2 configuration
type OAuth2Config struct {
	ClientID     string            `json:"clientId"`
	ClientSecret string            `json:"clientSecret"`
	TokenURL     string            `json:"tokenUrl"`
	Scopes       []string          `json:"scopes,omitempty"`
	Parameters   map[string]string `json:"parameters,omitempty"`
}

// TLSConfig defines TLS configuration
type TLSConfig struct {
	CertFile           string `json:"certFile,omitempty"`
	KeyFile            string `json:"keyFile,omitempty"`
	CACertFile         string `json:"caCertFile,omitempty"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify"`
	ServerName         string `json:"serverName,omitempty"`
}

// SubscriptionStatus represents the status of a subscription
type SubscriptionStatus struct {
	State           string                     `json:"state"`           // ACTIVE, PAUSED, FAILED, TERMINATED
	Health          string                     `json:"health"`          // HEALTHY, DEGRADED, UNHEALTHY
	EventsDelivered int64                      `json:"eventsDelivered"` // Total events delivered
	EventsFailed    int64                      `json:"eventsFailed"`    // Failed delivery events
	LastEventTime   *time.Time                 `json:"lastEventTime,omitempty"`
	LastErrorTime   *time.Time                 `json:"lastErrorTime,omitempty"`
	LastError       string                     `json:"lastError,omitempty"`
	Statistics      *SubscriptionStatistics    `json:"statistics,omitempty"`
	Conditions      []SubscriptionCondition    `json:"conditions,omitempty"`
	Events          []*SubscriptionStatusEvent `json:"events,omitempty"`
}

// SubscriptionStatistics provides statistics for subscription performance
type SubscriptionStatistics struct {
	DeliveryRate         float64       `json:"deliveryRate"`         // Events per second
	AverageDeliveryTime  time.Duration `json:"averageDeliveryTime"`  // Average delivery latency
	SuccessRate          float64       `json:"successRate"`          // Success rate percentage
	RetryRate            float64       `json:"retryRate"`            // Retry rate percentage
	DeadLetterQueueCount int64         `json:"deadLetterQueueCount"` // Events in DLQ
	BacklogSize          int64         `json:"backlogSize"`          // Pending events
}

// SubscriptionCondition represents a condition of the subscription
type SubscriptionCondition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"` // True, False, Unknown
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
	LastTransitionTime time.Time `json:"lastTransitionTime"`
	LastUpdateTime     time.Time `json:"lastUpdateTime,omitempty"`
}

// SubscriptionStatusEvent represents an event in subscription lifecycle
type SubscriptionStatusEvent struct {
	ID             string                 `json:"id"`
	Type           string                 `json:"type"` // CREATED, UPDATED, PAUSED, RESUMED, FAILED, TERMINATED
	Reason         string                 `json:"reason"`
	Message        string                 `json:"message"`
	Timestamp      time.Time              `json:"timestamp"`
	AdditionalData map[string]interface{} `json:"additionalData,omitempty"`
}

// InfrastructureEvent represents an infrastructure event
type InfrastructureEvent struct {
	EventID            string `json:"eventId"`
	EventType          string `json:"eventType"`
	EventTime          string `json:"eventTime"`
	Domain             string `json:"domain"`
	EventName          string `json:"eventName"`
	SourceName         string `json:"sourceName"`
	SourceID           string `json:"sourceId"`
	ReportingID        string `json:"reportingEntityId"`
	Priority           string `json:"priority"`
	Version            string `json:"version"`
	VesVersion         string `json:"vesEventListenerVersion"`
	TimeZone           string `json:"timeZoneOffset"`
	LastEpochMicrosec  int64  `json:"lastEpochMicrosec"`
	StartEpochMicrosec int64  `json:"startEpochMicrosec"`
	Sequence           int64  `json:"sequence"`

	// Event data payload
	EventData *runtime.RawExtension `json:"eventData,omitempty"`

	// Additional metadata
	Tags       map[string]string      `json:"tags,omitempty"`
	Labels     map[string]string      `json:"labels,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`

	// Event routing and processing
	ResourceID     string `json:"resourceId,omitempty"`
	ResourcePoolID string `json:"resourcePoolId,omitempty"`
	Severity       string `json:"severity,omitempty"`
	Category       string `json:"category,omitempty"`
	Source         string `json:"source,omitempty"`

	// Lifecycle information
	ProcessedAt    *time.Time `json:"processedAt,omitempty"`
	DeliveredAt    *time.Time `json:"deliveredAt,omitempty"`
	AcknowledgedAt *time.Time `json:"acknowledgedAt,omitempty"`
}

// EventBatch represents a batch of events for delivery
type EventBatch struct {
	BatchID      string                 `json:"batchId"`
	Events       []*InfrastructureEvent `json:"events"`
	BatchSize    int                    `json:"batchSize"`
	CreatedAt    time.Time              `json:"createdAt"`
	ExpiresAt    *time.Time             `json:"expiresAt,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	DeliveryInfo *EventDeliveryInfo     `json:"deliveryInfo,omitempty"`
}

// EventDeliveryInfo tracks event delivery status
type EventDeliveryInfo struct {
	SubscriptionID string     `json:"subscriptionId"`
	AttemptCount   int        `json:"attemptCount"`
	FirstAttempt   time.Time  `json:"firstAttempt"`
	LastAttempt    *time.Time `json:"lastAttempt,omitempty"`
	DeliveryStatus string     `json:"deliveryStatus"` // PENDING, DELIVERED, FAILED, EXPIRED
	ErrorMessage   string     `json:"errorMessage,omitempty"`
	ResponseCode   int        `json:"responseCode,omitempty"`
	ResponseBody   string     `json:"responseBody,omitempty"`
}

// EventTemplate defines templates for event generation
type EventTemplate struct {
	TemplateID  string                 `json:"templateId"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	EventType   string                 `json:"eventType"`
	Version     string                 `json:"version"`
	Schema      *runtime.RawExtension  `json:"schema"`
	Template    *runtime.RawExtension  `json:"template"`
	Variables   map[string]interface{} `json:"variables,omitempty"`
	Metadata    map[string]string      `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

// Filter types for subscription queries

// SubscriptionQueryFilter defines filters for querying subscriptions
type SubscriptionQueryFilter struct {
	Names           []string          `json:"names,omitempty"`
	EventTypes      []string          `json:"eventTypes,omitempty"`
	States          []string          `json:"states,omitempty"`
	ResourceTypes   []string          `json:"resourceTypes,omitempty"`
	ResourcePoolIDs []string          `json:"resourcePoolIds,omitempty"`
	CreatedBy       []string          `json:"createdBy,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
	CreatedAfter    *time.Time        `json:"createdAfter,omitempty"`
	CreatedBefore   *time.Time        `json:"createdBefore,omitempty"`
	Limit           int               `json:"limit,omitempty"`
	Offset          int               `json:"offset,omitempty"`
	SortBy          string            `json:"sortBy,omitempty"`
	SortOrder       string            `json:"sortOrder,omitempty"`
}

// EventFilter defines filters for querying events
type EventFilter struct {
	EventTypes      []string   `json:"eventTypes,omitempty"`
	ResourceIDs     []string   `json:"resourceIds,omitempty"`
	ResourcePoolIDs []string   `json:"resourcePoolIds,omitempty"`
	Sources         []string   `json:"sources,omitempty"`
	Severities      []string   `json:"severities,omitempty"`
	Categories      []string   `json:"categories,omitempty"`
	StartTime       *time.Time `json:"startTime,omitempty"`
	EndTime         *time.Time `json:"endTime,omitempty"`
	Limit           int        `json:"limit,omitempty"`
	Offset          int        `json:"offset,omitempty"`
	SortBy          string     `json:"sortBy,omitempty"`
	SortOrder       string     `json:"sortOrder,omitempty"`
}

// Request types for subscription management operations

// CreateSubscriptionRequest represents a request to create a subscription
type CreateSubscriptionRequest struct {
	Name                   string                 `json:"name"`
	Description            string                 `json:"description,omitempty"`
	CallbackUri            string                 `json:"callbackUri"`
	ConsumerSubscriptionId string                 `json:"consumerSubscriptionId,omitempty"`
	EventTypes             []string               `json:"eventTypes"`
	Filter                 *SubscriptionFilter    `json:"filter,omitempty"`
	EventConfig            *EventConfiguration    `json:"eventConfig,omitempty"`
	Authentication         *SubscriptionAuth      `json:"authentication,omitempty"`
	Extensions             map[string]interface{} `json:"extensions,omitempty"`
	Metadata               map[string]string      `json:"metadata,omitempty"`
}

// UpdateSubscriptionRequest represents a request to update a subscription
type UpdateSubscriptionRequest struct {
	Name                   *string                `json:"name,omitempty"`
	Description            *string                `json:"description,omitempty"`
	CallbackUri            *string                `json:"callbackUri,omitempty"`
	ConsumerSubscriptionId *string                `json:"consumerSubscriptionId,omitempty"`
	EventTypes             []string               `json:"eventTypes,omitempty"`
	Filter                 *SubscriptionFilter    `json:"filter,omitempty"`
	EventConfig            *EventConfiguration    `json:"eventConfig,omitempty"`
	Authentication         *SubscriptionAuth      `json:"authentication,omitempty"`
	Extensions             map[string]interface{} `json:"extensions,omitempty"`
	Metadata               map[string]string      `json:"metadata,omitempty"`
}

// Constants for subscription management

const (
	// Subscription States
	SubscriptionStateActive     = "ACTIVE"
	SubscriptionStatePaused     = "PAUSED"
	SubscriptionStateFailed     = "FAILED"
	SubscriptionStateTerminated = "TERMINATED"

	// Subscription Health States
	SubscriptionHealthHealthy   = "HEALTHY"
	SubscriptionHealthDegraded  = "DEGRADED"
	SubscriptionHealthUnhealthy = "UNHEALTHY"

	// Authentication Types
	AuthTypeNone   = "NONE"
	AuthTypeBasic  = "BASIC"
	AuthTypeBearer = "BEARER"
	AuthTypeOAuth2 = "OAUTH2"
	AuthTypeMTLS   = "MTLS"

	// Event Delivery Status
	EventDeliveryStatusPending   = "PENDING"
	EventDeliveryStatusDelivered = "DELIVERED"
	EventDeliveryStatusFailed    = "FAILED"
	EventDeliveryStatusExpired   = "EXPIRED"

	// Event Types (O-RAN specific)
	EventTypeResourceCreated    = "ResourceCreated"
	EventTypeResourceUpdated    = "ResourceUpdated"
	EventTypeResourceDeleted    = "ResourceDeleted"
	EventTypeResourceFault      = "ResourceFault"
	EventTypeResourceAlarm      = "ResourceAlarm"
	EventTypePerformanceMetrics = "PerformanceMetrics"
	EventTypeConfigChanged      = "ConfigurationChanged"

	// Event Severities
	EventSeverityCritical = "CRITICAL"
	EventSeverityMajor    = "MAJOR"
	EventSeverityMinor    = "MINOR"
	EventSeverityWarning  = "WARNING"
	EventSeverityInfo     = "INFO"

	// Event Categories
	EventCategoryFault         = "FAULT"
	EventCategoryPerformance   = "PERFORMANCE"
	EventCategoryConfiguration = "CONFIGURATION"
	EventCategorySecurity      = "SECURITY"
	EventCategoryLifecycle     = "LIFECYCLE"

	// Time Zone Formats
	TimeZoneUTC   = "UTC"
	TimeZoneLocal = "LOCAL"

	// Days of week
	DayMonday    = "MON"
	DayTuesday   = "TUE"
	DayWednesday = "WED"
	DayThursday  = "THU"
	DayFriday    = "FRI"
	DaySaturday  = "SAT"
	DaySunday    = "SUN"
)

