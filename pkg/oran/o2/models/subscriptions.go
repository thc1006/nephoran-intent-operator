package models

import (
	"time"
	
	"k8s.io/apimachinery/pkg/runtime"
)

// Subscription and Notification Models following O-RAN.WG6.O2ims-Interface-v01.01

// Subscription represents a subscription to O2 IMS events and notifications
type Subscription struct {
	SubscriptionID         string                 `json:"subscriptionId"`
	ConsumerSubscriptionID string                 `json:"consumerSubscriptionId,omitempty"`
	Filter                 *SubscriptionFilter    `json:"filter,omitempty"`
	Callback               string                 `json:"callback"`
	Authentication         *AuthenticationInfo    `json:"authentication,omitempty"`
	Extensions             map[string]interface{} `json:"extensions,omitempty"`
	
	// Subscription configuration
	EventTypes             []string               `json:"eventTypes"`
	NotificationFormat     string                 `json:"notificationFormat"` // JSON, XML
	DeliveryMethod         string                 `json:"deliveryMethod"` // HTTP, HTTPS, WEBHOOK
	RetryPolicy            *NotificationRetryPolicy `json:"retryPolicy,omitempty"`
	BufferSize             int                    `json:"bufferSize,omitempty"`
	BatchSize              int                    `json:"batchSize,omitempty"`
	BatchTimeout           time.Duration          `json:"batchTimeout,omitempty"`
	
	// Subscription status
	Status                 *SubscriptionStatus    `json:"status"`
	
	// Lifecycle information
	CreatedAt              time.Time              `json:"createdAt"`
	UpdatedAt              time.Time              `json:"updatedAt"`
	ExpiresAt              *time.Time             `json:"expiresAt,omitempty"`
	CreatedBy              string                 `json:"createdBy,omitempty"`
	UpdatedBy              string                 `json:"updatedBy,omitempty"`
}

// SubscriptionFilter defines filters for subscription events
type SubscriptionFilter struct {
	// Resource filters
	ResourcePoolIDs      []string               `json:"resourcePoolIds,omitempty"`
	ResourceTypeIDs      []string               `json:"resourceTypeIds,omitempty"`
	ResourceIDs          []string               `json:"resourceIds,omitempty"`
	DeploymentIDs        []string               `json:"deploymentIds,omitempty"`
	
	// Event filters
	EventTypes           []string               `json:"eventTypes,omitempty"`
	Severities           []string               `json:"severities,omitempty"`
	Sources              []string               `json:"sources,omitempty"`
	
	// Label and metadata filters
	Labels               map[string]string      `json:"labels,omitempty"`
	Annotations          map[string]string      `json:"annotations,omitempty"`
	
	// Time-based filters
	StartTime            *time.Time             `json:"startTime,omitempty"`
	EndTime              *time.Time             `json:"endTime,omitempty"`
	
	// Custom filter expressions
	FilterExpressions    []*FilterExpression    `json:"filterExpressions,omitempty"`
}

// FilterExpression represents a custom filter expression
type FilterExpression struct {
	Field      string      `json:"field"`
	Operator   string      `json:"operator"` // EQUALS, NOT_EQUALS, IN, NOT_IN, CONTAINS, REGEX, GT, LT, GTE, LTE
	Values     []string    `json:"values,omitempty"`
	Pattern    string      `json:"pattern,omitempty"` // For REGEX operator
	CaseSensitive bool     `json:"caseSensitive,omitempty"`
}

// AuthenticationInfo represents authentication information for callbacks
type AuthenticationInfo struct {
	Type         string            `json:"type"` // NONE, BASIC, BEARER, OAUTH2, CERTIFICATE
	Credentials  map[string]string `json:"credentials,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	Certificate  *CertificateInfo  `json:"certificate,omitempty"`
	OAuth2       *OAuth2Info       `json:"oauth2,omitempty"`
}

// CertificateInfo represents certificate-based authentication
type CertificateInfo struct {
	CertData     string `json:"certData"`
	KeyData      string `json:"keyData"`
	CAData       string `json:"caData,omitempty"`
	ServerName   string `json:"serverName,omitempty"`
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`
}

// OAuth2Info represents OAuth2 authentication information
type OAuth2Info struct {
	TokenURL     string   `json:"tokenUrl"`
	ClientID     string   `json:"clientId"`
	ClientSecret string   `json:"clientSecret"`
	Scopes       []string `json:"scopes,omitempty"`
}

// NotificationRetryPolicy defines retry behavior for failed notifications
type NotificationRetryPolicy struct {
	MaxRetries     int           `json:"maxRetries"`
	RetryDelay     time.Duration `json:"retryDelay"`
	BackoffFactor  float64       `json:"backoffFactor"`
	MaxRetryDelay  time.Duration `json:"maxRetryDelay"`
	RetryOnErrors  []int         `json:"retryOnErrors,omitempty"` // HTTP status codes
	GiveUpAfter    time.Duration `json:"giveUpAfter,omitempty"`
}

// SubscriptionStatus represents the status of a subscription
type SubscriptionStatus struct {
	State              string                 `json:"state"` // ACTIVE, PAUSED, FAILED, EXPIRED, TERMINATED
	Health             string                 `json:"health"` // HEALTHY, DEGRADED, UNHEALTHY
	LastNotification   *time.Time             `json:"lastNotification,omitempty"`
	TotalNotifications int64                  `json:"totalNotifications"`
	FailedNotifications int64                 `json:"failedNotifications"`
	LastFailure        *NotificationFailure   `json:"lastFailure,omitempty"`
	Statistics         *SubscriptionStatistics `json:"statistics,omitempty"`
	Conditions         []SubscriptionCondition `json:"conditions,omitempty"`
	CreatedAt          time.Time              `json:"createdAt"`
	LastUpdated        time.Time              `json:"lastUpdated"`
}

// NotificationFailure represents a notification delivery failure
type NotificationFailure struct {
	Timestamp    time.Time `json:"timestamp"`
	ErrorCode    int       `json:"errorCode"`
	ErrorMessage string    `json:"errorMessage"`
	RetryCount   int       `json:"retryCount"`
	NextRetry    *time.Time `json:"nextRetry,omitempty"`
}

// SubscriptionStatistics represents statistics for a subscription
type SubscriptionStatistics struct {
	NotificationsPerMinute float64   `json:"notificationsPerMinute"`
	AverageLatency         time.Duration `json:"averageLatency"`
	SuccessRate            float64   `json:"successRate"`
	LastReset              time.Time `json:"lastReset"`
	PeakNotificationsPerMinute float64 `json:"peakNotificationsPerMinute"`
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

// NotificationEventType represents a type of event that can be subscribed to
type NotificationEventType struct {
	EventTypeID    string                 `json:"eventTypeId"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description,omitempty"`
	Category       string                 `json:"category"` // RESOURCE, DEPLOYMENT, ALARM, PERFORMANCE
	Severity       string                 `json:"severity"` // INFO, WARNING, ERROR, CRITICAL
	Source         string                 `json:"source"`
	Schema         *runtime.RawExtension  `json:"schema,omitempty"`
	Examples       []*NotificationExample `json:"examples,omitempty"`
	Extensions     map[string]interface{} `json:"extensions,omitempty"`
	Deprecated     bool                   `json:"deprecated,omitempty"`
	SupportedSince string                 `json:"supportedSince,omitempty"`
}

// NotificationExample represents an example notification
type NotificationExample struct {
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	Data        *runtime.RawExtension `json:"data"`
}

// Notification represents a notification sent to subscribers
type Notification struct {
	NotificationID         string                 `json:"notificationId"`
	SubscriptionID         string                 `json:"subscriptionId"`
	ConsumerSubscriptionID string                 `json:"consumerSubscriptionId,omitempty"`
	EventType              string                 `json:"eventType"`
	EventID                string                 `json:"eventId,omitempty"`
	EventTime              time.Time              `json:"eventTime"`
	NotificationTime       time.Time              `json:"notificationTime"`
	
	// Event source and context
	Source                 string                 `json:"source"`
	Subject                string                 `json:"subject"`
	CorrelationID          string                 `json:"correlationId,omitempty"`
	
	// Event data
	Data                   *runtime.RawExtension  `json:"data"`
	DataContentType        string                 `json:"dataContentType,omitempty"`
	
	// Notification metadata
	Severity               string                 `json:"severity,omitempty"`
	Category               string                 `json:"category,omitempty"`
	Extensions             map[string]interface{} `json:"extensions,omitempty"`
	
	// Delivery tracking
	DeliveryStatus         *DeliveryStatus        `json:"deliveryStatus,omitempty"`
}

// DeliveryStatus represents the delivery status of a notification
type DeliveryStatus struct {
	Status         string    `json:"status"` // PENDING, DELIVERED, FAILED, RETRYING
	Attempts       int       `json:"attempts"`
	LastAttempt    time.Time `json:"lastAttempt"`
	NextAttempt    *time.Time `json:"nextAttempt,omitempty"`
	ErrorMessage   string    `json:"errorMessage,omitempty"`
	ResponseCode   int       `json:"responseCode,omitempty"`
	ResponseTime   time.Duration `json:"responseTime,omitempty"`
}

// Alarm represents an alarm in the O2 system
type Alarm struct {
	AlarmID              string                 `json:"alarmId"`
	AlarmDefinitionID    string                 `json:"alarmDefinitionId"`
	ProbableCause        string                 `json:"probableCause"`
	SpecificProblem      string                 `json:"specificProblem,omitempty"`
	PerceivedSeverity    string                 `json:"perceivedSeverity"` // CRITICAL, MAJOR, MINOR, WARNING, INDETERMINATE, CLEARED
	AlarmRaisedTime      time.Time              `json:"alarmRaisedTime"`
	AlarmChangedTime     *time.Time             `json:"alarmChangedTime,omitempty"`
	AlarmClearedTime     *time.Time             `json:"alarmClearedTime,omitempty"`
	AlarmAcknowledgedTime *time.Time            `json:"alarmAcknowledgedTime,omitempty"`
	
	// Alarm context
	ManagedObjectID      string                 `json:"managedObjectId"`
	ResourceID           string                 `json:"resourceId,omitempty"`
	ResourcePoolID       string                 `json:"resourcePoolId,omitempty"`
	DeploymentID         string                 `json:"deploymentId,omitempty"`
	
	// Alarm details
	FaultType            string                 `json:"faultType,omitempty"`
	FaultDetails         string                 `json:"faultDetails,omitempty"`
	EventType            string                 `json:"eventType"` // EQUIPMENT_ALARM, COMMUNICATION_ALARM, QOS_ALARM, PROCESSING_ERROR_ALARM, ENVIRONMENTAL_ALARM
	ProposedRepairActions []string              `json:"proposedRepairActions,omitempty"`
	AdditionalText       string                 `json:"additionalText,omitempty"`
	AdditionalInformation map[string]interface{} `json:"additionalInformation,omitempty"`
	
	// Alarm state
	AckState             string                 `json:"ackState"` // ACKNOWLEDGED, UNACKNOWLEDGED
	ClearanceState       string                 `json:"clearanceState"` // CLEARED, NOT_CLEARED
	
	// Correlation and grouping
	CorrelatedAlarms     []string               `json:"correlatedAlarms,omitempty"`
	RootCause            string                 `json:"rootCause,omitempty"`
	AlarmGroup           string                 `json:"alarmGroup,omitempty"`
	
	// Extensions
	Extensions           map[string]interface{} `json:"extensions,omitempty"`
}

// Filter types for subscription and notification queries

// SubscriptionFilter for querying subscriptions (different from event filter)
type SubscriptionQueryFilter struct {
	SubscriptionIDs        []string          `json:"subscriptionIds,omitempty"`
	ConsumerSubscriptionIDs []string         `json:"consumerSubscriptionIds,omitempty"`
	EventTypes             []string          `json:"eventTypes,omitempty"`
	States                 []string          `json:"states,omitempty"`
	HealthStates           []string          `json:"healthStates,omitempty"`
	CreatedBy              []string          `json:"createdBy,omitempty"`
	Callbacks              []string          `json:"callbacks,omitempty"`
	Labels                 map[string]string `json:"labels,omitempty"`
	CreatedAfter           *time.Time        `json:"createdAfter,omitempty"`
	CreatedBefore          *time.Time        `json:"createdBefore,omitempty"`
	ExpiresAfter           *time.Time        `json:"expiresAfter,omitempty"`
	ExpiresBefore          *time.Time        `json:"expiresBefore,omitempty"`
	Limit                  int               `json:"limit,omitempty"`
	Offset                 int               `json:"offset,omitempty"`
	SortBy                 string            `json:"sortBy,omitempty"`
	SortOrder              string            `json:"sortOrder,omitempty"`
}

// AlarmFilter defines filters for querying alarms
type AlarmFilter struct {
	AlarmIDs             []string          `json:"alarmIds,omitempty"`
	AlarmDefinitionIDs   []string          `json:"alarmDefinitionIds,omitempty"`
	ManagedObjectIDs     []string          `json:"managedObjectIds,omitempty"`
	ResourceIDs          []string          `json:"resourceIds,omitempty"`
	ResourcePoolIDs      []string          `json:"resourcePoolIds,omitempty"`
	DeploymentIDs        []string          `json:"deploymentIds,omitempty"`
	PerceivedSeverities  []string          `json:"perceivedSeverities,omitempty"`
	EventTypes           []string          `json:"eventTypes,omitempty"`
	FaultTypes           []string          `json:"faultTypes,omitempty"`
	AckStates            []string          `json:"ackStates,omitempty"`
	ClearanceStates      []string          `json:"clearanceStates,omitempty"`
	AlarmGroups          []string          `json:"alarmGroups,omitempty"`
	RaisedAfter          *time.Time        `json:"raisedAfter,omitempty"`
	RaisedBefore         *time.Time        `json:"raisedBefore,omitempty"`
	ClearedAfter         *time.Time        `json:"clearedAfter,omitempty"`
	ClearedBefore        *time.Time        `json:"clearedBefore,omitempty"`
	Labels               map[string]string `json:"labels,omitempty"`
	Limit                int               `json:"limit,omitempty"`
	Offset               int               `json:"offset,omitempty"`
	SortBy               string            `json:"sortBy,omitempty"`
	SortOrder            string            `json:"sortOrder,omitempty"`
}

// Request types for subscription and notification management

// CreateSubscriptionRequest represents a request to create a subscription
type CreateSubscriptionRequest struct {
	ConsumerSubscriptionID string                 `json:"consumerSubscriptionId,omitempty"`
	Filter                 *SubscriptionFilter    `json:"filter,omitempty"`
	Callback               string                 `json:"callback"`
	Authentication         *AuthenticationInfo    `json:"authentication,omitempty"`
	EventTypes             []string               `json:"eventTypes"`
	NotificationFormat     string                 `json:"notificationFormat,omitempty"`
	DeliveryMethod         string                 `json:"deliveryMethod,omitempty"`
	RetryPolicy            *NotificationRetryPolicy `json:"retryPolicy,omitempty"`
	BufferSize             int                    `json:"bufferSize,omitempty"`
	BatchSize              int                    `json:"batchSize,omitempty"`
	BatchTimeout           time.Duration          `json:"batchTimeout,omitempty"`
	ExpirationTime         *time.Time             `json:"expirationTime,omitempty"`
	Extensions             map[string]interface{} `json:"extensions,omitempty"`
	Metadata               map[string]string      `json:"metadata,omitempty"`
}

// UpdateSubscriptionRequest represents a request to update a subscription
type UpdateSubscriptionRequest struct {
	Filter                 *SubscriptionFilter    `json:"filter,omitempty"`
	Callback               *string                `json:"callback,omitempty"`
	Authentication         *AuthenticationInfo    `json:"authentication,omitempty"`
	EventTypes             []string               `json:"eventTypes,omitempty"`
	RetryPolicy            *NotificationRetryPolicy `json:"retryPolicy,omitempty"`
	BufferSize             *int                   `json:"bufferSize,omitempty"`
	BatchSize              *int                   `json:"batchSize,omitempty"`
	BatchTimeout           *time.Duration         `json:"batchTimeout,omitempty"`
	ExpirationTime         *time.Time             `json:"expirationTime,omitempty"`
	Extensions             map[string]interface{} `json:"extensions,omitempty"`
	Metadata               map[string]string      `json:"metadata,omitempty"`
	
	// Subscription control
	State                  *string                `json:"state,omitempty"` // ACTIVE, PAUSED
}

// AlarmAcknowledgementRequest represents a request to acknowledge an alarm
type AlarmAcknowledgementRequest struct {
	AcknowledgedBy   string                 `json:"acknowledgedBy"`
	AcknowledgeTime  *time.Time             `json:"acknowledgeTime,omitempty"`
	Comment          string                 `json:"comment,omitempty"`
	Extensions       map[string]interface{} `json:"extensions,omitempty"`
}

// AlarmClearRequest represents a request to clear an alarm
type AlarmClearRequest struct {
	ClearedBy        string                 `json:"clearedBy"`
	ClearTime        *time.Time             `json:"clearTime,omitempty"`
	ClearReason      string                 `json:"clearReason,omitempty"`
	Comment          string                 `json:"comment,omitempty"`
	Extensions       map[string]interface{} `json:"extensions,omitempty"`
}

// Webhook represents a webhook endpoint for notifications
type Webhook struct {
	WebhookID       string                 `json:"webhookId"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description,omitempty"`
	URL             string                 `json:"url"`
	Method          string                 `json:"method"` // POST, PUT, PATCH
	Headers         map[string]string      `json:"headers,omitempty"`
	Authentication  *AuthenticationInfo    `json:"authentication,omitempty"`
	ContentType     string                 `json:"contentType"`
	Template        string                 `json:"template,omitempty"`
	Timeout         time.Duration          `json:"timeout"`
	RetryPolicy     *NotificationRetryPolicy `json:"retryPolicy,omitempty"`
	Status          *WebhookStatus         `json:"status"`
	Extensions      map[string]interface{} `json:"extensions,omitempty"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
}

// WebhookStatus represents the status of a webhook
type WebhookStatus struct {
	State              string    `json:"state"` // ACTIVE, INACTIVE, FAILED
	LastDelivery       *time.Time `json:"lastDelivery,omitempty"`
	LastSuccess        *time.Time `json:"lastSuccess,omitempty"`
	LastFailure        *time.Time `json:"lastFailure,omitempty"`
	SuccessCount       int64     `json:"successCount"`
	FailureCount       int64     `json:"failureCount"`
	ConsecutiveFailures int      `json:"consecutiveFailures"`
	LastErrorMessage   string    `json:"lastErrorMessage,omitempty"`
}

// Constants for subscription and notification management

const (
	// Subscription States
	SubscriptionStateActive     = "ACTIVE"
	SubscriptionStatePaused     = "PAUSED"
	SubscriptionStateFailed     = "FAILED"
	SubscriptionStateExpired    = "EXPIRED"
	SubscriptionStateTerminated = "TERMINATED"
	
	// Notification Format Types
	NotificationFormatJSON = "JSON"
	NotificationFormatXML  = "XML"
	
	// Delivery Methods
	DeliveryMethodHTTP    = "HTTP"
	DeliveryMethodHTTPS   = "HTTPS"
	DeliveryMethodWebhook = "WEBHOOK"
	
	// Authentication Types
	AuthTypeNone        = "NONE"
	AuthTypeBasic       = "BASIC"
	AuthTypeBearer      = "BEARER"
	AuthTypeOAuth2      = "OAUTH2"
	AuthTypeCertificate = "CERTIFICATE"
	
	// Filter Operators
	FilterOpEquals     = "EQUALS"
	FilterOpNotEquals  = "NOT_EQUALS"
	FilterOpIn         = "IN"
	FilterOpNotIn      = "NOT_IN"
	FilterOpContains   = "CONTAINS"
	FilterOpRegex      = "REGEX"
	FilterOpGT         = "GT"
	FilterOpLT         = "LT"
	FilterOpGTE        = "GTE"
	FilterOpLTE        = "LTE"
	
	// Event Categories
	EventCategoryResource    = "RESOURCE"
	EventCategoryDeployment  = "DEPLOYMENT"
	EventCategoryAlarm       = "ALARM"
	EventCategoryPerformance = "PERFORMANCE"
	
	// Event Types
	EventTypeResourceCreated    = "RESOURCE_CREATED"
	EventTypeResourceUpdated    = "RESOURCE_UPDATED"
	EventTypeResourceDeleted    = "RESOURCE_DELETED"
	EventTypeResourceHealthChanged = "RESOURCE_HEALTH_CHANGED"
	EventTypeDeploymentStarted  = "DEPLOYMENT_STARTED"
	EventTypeDeploymentCompleted = "DEPLOYMENT_COMPLETED"
	EventTypeDeploymentFailed   = "DEPLOYMENT_FAILED"
	EventTypeAlarmRaised        = "ALARM_RAISED"
	EventTypeAlarmCleared       = "ALARM_CLEARED"
	EventTypeAlarmChanged       = "ALARM_CHANGED"
	
	// Alarm Severities
	AlarmSeverityCritical      = "CRITICAL"
	AlarmSeverityMajor         = "MAJOR"
	AlarmSeverityMinor         = "MINOR"
	AlarmSeverityWarning       = "WARNING"
	AlarmSeverityIndeterminate = "INDETERMINATE"
	AlarmSeverityCleared       = "CLEARED"
	
	// Alarm States
	AlarmAckStateAcknowledged   = "ACKNOWLEDGED"
	AlarmAckStateUnacknowledged = "UNACKNOWLEDGED"
	AlarmClearanceStateCleared     = "CLEARED"
	AlarmClearanceStateNotCleared  = "NOT_CLEARED"
	
	// Event Types for Alarms
	AlarmEventTypeEquipment     = "EQUIPMENT_ALARM"
	AlarmEventTypeCommunication = "COMMUNICATION_ALARM"
	AlarmEventTypeQoS           = "QOS_ALARM"
	AlarmEventTypeProcessingError = "PROCESSING_ERROR_ALARM"
	AlarmEventTypeEnvironmental = "ENVIRONMENTAL_ALARM"
	
	// Delivery Status
	DeliveryStatusPending   = "PENDING"
	DeliveryStatusDelivered = "DELIVERED"
	DeliveryStatusFailed    = "FAILED"
	DeliveryStatusRetrying  = "RETRYING"
)