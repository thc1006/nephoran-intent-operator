package models

import (
	"time"
	
	"k8s.io/apimachinery/pkg/runtime"
)

// Subscription and Event Management Models following O-RAN.WG6.O2ims-Interface-v01.01

// Subscription represents an event subscription in O2 IMS
type Subscription struct {
	SubscriptionID      string                 `json:"subscriptionId"`
	Name                string                 `json:"name"`
	Description         string                 `json:"description,omitempty"`
	CallbackUri         string                 `json:"callbackUri"`
	
	// Subscription filter and configuration
	Filter              *SubscriptionFilter    `json:"filter,omitempty"`
	ConsumerSubscriptionId string              `json:"consumerSubscriptionId,omitempty"`
	
	// Event types and configuration
	EventTypes          []string               `json:"eventTypes"`
	EventConfig         *EventConfiguration    `json:"eventConfig,omitempty"`
	
	// Authentication and security
	Authentication      *SubscriptionAuth      `json:"authentication,omitempty"`
	
	// Subscription lifecycle
	Status              *SubscriptionStatus    `json:"status"`
	Extensions          map[string]interface{} `json:"extensions,omitempty"`
	CreatedAt           time.Time              `json:"createdAt"`
	UpdatedAt           time.Time              `json:"updatedAt"`
	CreatedBy           string                 `json:"createdBy,omitempty"`
	UpdatedBy           string                 `json:"updatedBy,omitempty"`
}

// SubscriptionFilter defines filters for event subscriptions
type SubscriptionFilter struct {
	// Resource filters
	ResourceTypes       []string               `json:"resourceTypes,omitempty"`
	ResourceIDs         []string               `json:"resourceIds,omitempty"`
	ResourcePoolIDs     []string               `json:"resourcePoolIds,omitempty"`
	
	// Event filters
	EventSeverities     []string               `json:"eventSeverities,omitempty"`
	EventSources        []string               `json:"eventSources,omitempty"`
	
	// Geographic filters
	LocationFilter      *LocationFilter        `json:"locationFilter,omitempty"`
	
	// Custom filters
	CustomFilters       map[string]interface{} `json:"customFilters,omitempty"`
	
	// Time filters
	TimeWindow          *TimeWindow            `json:"timeWindow,omitempty"`
}

// LocationFilter defines geographic location-based filtering
type LocationFilter struct {
	Regions             []string               `json:"regions,omitempty"`
	Zones               []string               `json:"zones,omitempty"`
	DataCenters         []string               `json:"dataCenters,omitempty"`
	Coordinates         *GeographicArea        `json:"coordinates,omitempty"`
}

// GeographicArea defines a geographic area for filtering
type GeographicArea struct {
	CenterLatitude      float64                `json:"centerLatitude"`
	CenterLongitude     float64                `json:"centerLongitude"`
	Radius              float64                `json:"radius"` // in kilometers
	BoundingBox         *BoundingBox           `json:"boundingBox,omitempty"`
}

// BoundingBox defines a rectangular geographic boundary
type BoundingBox struct {
	NorthLatitude       float64                `json:"northLatitude"`
	SouthLatitude       float64                `json:"southLatitude"`
	EastLongitude       float64                `json:"eastLongitude"`
	WestLongitude       float64                `json:"westLongitude"`
}

// TimeWindow defines a time-based filter
type TimeWindow struct {
	StartTime           *time.Time             `json:"startTime,omitempty"`
	EndTime             *time.Time             `json:"endTime,omitempty"`
	TimeOfDay           *TimeOfDayWindow       `json:"timeOfDay,omitempty"`
	DaysOfWeek          []string               `json:"daysOfWeek,omitempty"` // MON, TUE, WED, etc.
	TimeZone            string                 `json:"timeZone,omitempty"`
}

// TimeOfDayWindow defines a daily time window
type TimeOfDayWindow struct {
	StartHour           int                    `json:"startHour"` // 0-23
	StartMinute         int                    `json:"startMinute"` // 0-59
	EndHour             int                    `json:"endHour"` // 0-23
	EndMinute           int                    `json:"endMinute"` // 0-59
}

// EventConfiguration defines event delivery configuration
type EventConfiguration struct {
	// Delivery options
	DeliveryMethod      string                 `json:"deliveryMethod"` // webhook, queue, stream
	BatchConfig         *BatchConfiguration    `json:"batchConfig,omitempty"`
	RetryConfig         *EventRetryConfig      `json:"retryConfig,omitempty"`
	
	// Event format and encoding
	EventFormat         string                 `json:"eventFormat"` // json, xml, cloudevents
	Encoding            string                 `json:"encoding"` // utf8, base64
	Compression         string                 `json:"compression,omitempty"` // gzip, deflate
	
	// Event enrichment
	IncludeResourceData bool                   `json:"includeResourceData"`
	IncludeMetadata     bool                   `json:"includeMetadata"`
	CustomHeaders       map[string]string      `json:"customHeaders,omitempty"`
	
	// Quality of service
	QoS                 *EventQoS              `json:"qos,omitempty"`
}

// BatchConfiguration defines event batching configuration
type BatchConfiguration struct {
	Enabled             bool                   `json:"enabled"`
	MaxBatchSize        int                    `json:"maxBatchSize"`
	MaxWaitTime         time.Duration          `json:"maxWaitTime"`
	FlushOnShutdown     bool                   `json:"flushOnShutdown"`
}

// EventRetryConfig defines retry configuration for event delivery
type EventRetryConfig struct {
	MaxRetries          int                    `json:"maxRetries"`
	InitialRetryDelay   time.Duration          `json:"initialRetryDelay"`
	MaxRetryDelay       time.Duration          `json:"maxRetryDelay"`
	BackoffMultiplier   float64                `json:"backoffMultiplier"`
	RetryableErrors     []string               `json:"retryableErrors,omitempty"`
	DeadLetterQueue     string                 `json:"deadLetterQueue,omitempty"`
}

// EventQoS defines quality of service parameters for events
type EventQoS struct {
	DeliveryGuarantee   string                 `json:"deliveryGuarantee"` // AT_LEAST_ONCE, AT_MOST_ONCE, EXACTLY_ONCE
	Priority            string                 `json:"priority"` // HIGH, MEDIUM, LOW
	MaxDeliveryTime     time.Duration          `json:"maxDeliveryTime,omitempty"`
	OrderingGuarantee   bool                   `json:"orderingGuarantee"`
}

// SubscriptionAuth defines authentication configuration for subscriptions
type SubscriptionAuth struct {
	Type                string                 `json:"type"` // none, basic, bearer, oauth2, certificate
	Credentials         map[string]string      `json:"credentials,omitempty"`
	OAuth2Config        *OAuth2Config          `json:"oauth2Config,omitempty"`
	CertificateConfig   *CertificateConfig     `json:"certificateConfig,omitempty"`
}

// OAuth2Config defines OAuth2 authentication configuration
type OAuth2Config struct {
	TokenUrl            string                 `json:"tokenUrl"`
	ClientId            string                 `json:"clientId"`
	ClientSecret        string                 `json:"clientSecret"`
	Scope               []string               `json:"scope,omitempty"`
	GrantType           string                 `json:"grantType"` // client_credentials, authorization_code
}

// CertificateConfig defines certificate-based authentication
type CertificateConfig struct {
	CertificatePath     string                 `json:"certificatePath"`
	PrivateKeyPath      string                 `json:"privateKeyPath"`
	CACertificatePath   string                 `json:"caCertificatePath,omitempty"`
	SkipTLSVerify       bool                   `json:"skipTlsVerify"`
}

// SubscriptionStatus represents the status of a subscription
type SubscriptionStatus struct {
	State               string                 `json:"state"` // ACTIVE, INACTIVE, SUSPENDED, ERROR
	Health              string                 `json:"health"` // HEALTHY, DEGRADED, UNHEALTHY
	LastEventDelivered  *time.Time             `json:"lastEventDelivered,omitempty"`
	EventsDelivered     int64                  `json:"eventsDelivered"`
	EventsFailures      int64                  `json:"eventsFailures"`
	LastError           string                 `json:"lastError,omitempty"`
	LastHealthCheck     time.Time              `json:"lastHealthCheck"`
	DeliveryStats       *DeliveryStatistics    `json:"deliveryStats,omitempty"`
	Conditions          []SubscriptionCondition `json:"conditions,omitempty"`
}

// DeliveryStatistics provides statistics about event delivery
type DeliveryStatistics struct {
	TotalEvents         int64                  `json:"totalEvents"`
	SuccessfulDeliveries int64                 `json:"successfulDeliveries"`
	FailedDeliveries    int64                  `json:"failedDeliveries"`
	RetryAttempts       int64                  `json:"retryAttempts"`
	AverageDeliveryTime time.Duration          `json:"averageDeliveryTime"`
	DeliveryRate        float64                `json:"deliveryRate"` // events per second
	ErrorRate           float64                `json:"errorRate"` // percentage
	LastResetTime       time.Time              `json:"lastResetTime"`
}

// SubscriptionCondition represents a condition of the subscription
type SubscriptionCondition struct {
	Type                string                 `json:"type"`
	Status              string                 `json:"status"` // True, False, Unknown
	Reason              string                 `json:"reason,omitempty"`
	Message             string                 `json:"message,omitempty"`
	LastTransitionTime  time.Time              `json:"lastTransitionTime"`
	LastUpdateTime      time.Time              `json:"lastUpdateTime,omitempty"`
}

// InfrastructureEvent represents an event that occurred in the infrastructure
type InfrastructureEvent struct {
	EventID             string                 `json:"eventId"`
	EventType           string                 `json:"eventType"`
	EventTime           time.Time              `json:"eventTime"`
	EventSource         string                 `json:"eventSource"`
	
	// Event classification
	Severity            string                 `json:"severity"` // CRITICAL, MAJOR, MINOR, WARNING, INFO
	Category            string                 `json:"category"` // ALARM, STATE_CHANGE, PERFORMANCE, SECURITY
	
	// Resource information
	ResourceID          string                 `json:"resourceId,omitempty"`
	ResourceType        string                 `json:"resourceType,omitempty"`
	ResourcePoolID      string                 `json:"resourcePoolId,omitempty"`
	
	// Event details
	Summary             string                 `json:"summary"`
	Description         string                 `json:"description,omitempty"`
	Reason              string                 `json:"reason,omitempty"`
	
	// Event data
	EventData           *runtime.RawExtension  `json:"eventData,omitempty"`
	PreviousState       *runtime.RawExtension  `json:"previousState,omitempty"`
	CurrentState        *runtime.RawExtension  `json:"currentState,omitempty"`
	
	// Context and metadata
	CorrelationID       string                 `json:"correlationId,omitempty"`
	Tags                map[string]string      `json:"tags,omitempty"`
	Labels              map[string]string      `json:"labels,omitempty"`
	Extensions          map[string]interface{} `json:"extensions,omitempty"`
	
	// Geographic and temporal context
	Location            *EventLocation         `json:"location,omitempty"`
	Duration            *time.Duration         `json:"duration,omitempty"`
	
	// Notification tracking
	NotificationsSent   []string               `json:"notificationsSent,omitempty"`
	AcknowledgedBy      []string               `json:"acknowledgedBy,omitempty"`
	ResolvedBy          string                 `json:"resolvedBy,omitempty"`
	ResolvedAt          *time.Time             `json:"resolvedAt,omitempty"`
}

// Constants for subscriptions and events

const (
	// Event types
	EventTypeResourceCreated      = "ResourceCreated"
	EventTypeResourceUpdated      = "ResourceUpdated"
	EventTypeResourceDeleted      = "ResourceDeleted"
	EventTypeResourceStateChanged = "ResourceStateChanged"
	EventTypeResourceHealthChanged = "ResourceHealthChanged"
	EventTypeAlarmRaised          = "AlarmRaised"
	EventTypeAlarmCleared         = "AlarmCleared"
	EventTypeAlarmChanged         = "AlarmChanged"
	EventTypeDeploymentCreated    = "DeploymentCreated"
	EventTypeDeploymentUpdated    = "DeploymentUpdated"
	EventTypeDeploymentCompleted  = "DeploymentCompleted"
	EventTypeDeploymentFailed     = "DeploymentFailed"
	EventTypeDeploymentDeleted    = "DeploymentDeleted"
	
	// Event severities
	EventSeverityCritical         = "CRITICAL"
	EventSeverityMajor            = "MAJOR"
	EventSeverityMinor            = "MINOR"
	EventSeverityWarning          = "WARNING"
	EventSeverityInfo             = "INFO"
	
	// Event categories
	EventCategoryAlarm            = "ALARM"
	EventCategoryStateChange      = "STATE_CHANGE"
	EventCategoryPerformance      = "PERFORMANCE"
	EventCategorySecurity         = "SECURITY"
	EventCategoryConfiguration    = "CONFIGURATION"
	
	// Subscription states
	SubscriptionStateActive       = "ACTIVE"
	SubscriptionStateInactive     = "INACTIVE"
	SubscriptionStateSuspended    = "SUSPENDED"
	SubscriptionStateError        = "ERROR"
	
	// Authentication types
	AuthTypeNone                  = "none"
	AuthTypeBasic                 = "basic"
	AuthTypeBearer                = "bearer"
	AuthTypeOAuth2                = "oauth2"
	AuthTypeCertificate           = "certificate"
	
	// Delivery methods
	DeliveryMethodWebhook         = "webhook"
	DeliveryMethodQueue           = "queue"
	DeliveryMethodStream          = "stream"
	
	// Event formats
	EventFormatJSON               = "json"
	EventFormatXML                = "xml"
	EventFormatCloudEvents        = "cloudevents"
	
	// QoS delivery guarantees
	QoSAtLeastOnce                = "AT_LEAST_ONCE"
	QoSAtMostOnce                 = "AT_MOST_ONCE"
	QoSExactlyOnce                = "EXACTLY_ONCE"
)