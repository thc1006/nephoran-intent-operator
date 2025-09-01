package o2

import (
	"context"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
)

// Additional missing types for resource lifecycle
type ResourceLifecycleEvent struct {
	EventID     string                 `json:"eventId"`
	ResourceID  string                 `json:"resourceId"`
	EventType   string                 `json:"eventType"`
	State       string                 `json:"state"`
	Phase       string                 `json:"phase"`
	Reason      string                 `json:"reason,omitempty"`
	Message     string                 `json:"message,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Source      string                 `json:"source,omitempty"`
	Actor       string                 `json:"actor,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type ResourceEventBus interface {
	Publish(ctx context.Context, event *ResourceLifecycleEvent) error
	Subscribe(ctx context.Context, filter *EventFilter, callback func(*ResourceLifecycleEvent) error) error
	Unsubscribe(ctx context.Context, subscriptionID string) error
	GetEventHistory(ctx context.Context, resourceID string, limit int) ([]*ResourceLifecycleEvent, error)
}


// Simple event bus implementation
type simpleEventBus struct {
	subscribers map[string]func(*ResourceLifecycleEvent) error
	events      []*ResourceLifecycleEvent
}

func NewResourceEventBus() ResourceEventBus {
	return &simpleEventBus{
		subscribers: make(map[string]func(*ResourceLifecycleEvent) error),
		events:      make([]*ResourceLifecycleEvent, 0),
	}
}

func (bus *simpleEventBus) Publish(ctx context.Context, event *ResourceLifecycleEvent) error {
	bus.events = append(bus.events, event)
	for _, callback := range bus.subscribers {
		if err := callback(event); err != nil {
			// Log error but don't fail the publish
			continue
		}
	}
	return nil
}

func (bus *simpleEventBus) Subscribe(ctx context.Context, filter *EventFilter, callback func(*ResourceLifecycleEvent) error) error {
	// Simple implementation - in production this would be more sophisticated
	subscriptionID := generateSubscriptionID()
	bus.subscribers[subscriptionID] = callback
	return nil
}

func (bus *simpleEventBus) Unsubscribe(ctx context.Context, subscriptionID string) error {
	delete(bus.subscribers, subscriptionID)
	return nil
}

func (bus *simpleEventBus) GetEventHistory(ctx context.Context, resourceID string, limit int) ([]*ResourceLifecycleEvent, error) {
	var result []*ResourceLifecycleEvent
	for _, event := range bus.events {
		if event.ResourceID == resourceID {
			result = append(result, event)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func generateSubscriptionID() string {
	return "sub-" + time.Now().Format("20060102-150405")
}

// EventSubscriber interface for handling resource lifecycle events
type EventSubscriber interface {
	OnResourceEvent(ctx context.Context, event *ResourceLifecycleEvent) error
	GetSubscriberID() string
	IsActive() bool
}

// ResourceLifecycleMetrics for tracking resource lifecycle performance
type ResourceLifecycleMetrics struct {
	TotalEvents         int64         `json:"totalEvents"`
	EventsPerSecond     float64       `json:"eventsPerSecond"`
	AverageProcessTime  time.Duration `json:"averageProcessTime"`
	ErrorRate           float64       `json:"errorRate"`
	LastEventTimestamp  time.Time     `json:"lastEventTimestamp"`
	SubscriberCount     int           `json:"subscriberCount"`
	EventTypeCounters   map[string]int64 `json:"eventTypeCounters"`
	StateTransitionTime map[string]time.Duration `json:"stateTransitionTime"`
}

// ResourcePolicies for defining resource lifecycle rules
type ResourcePolicies struct {
	ValidationPolicies    []ValidationPolicy    `json:"validationPolicies,omitempty"`
	TransitionPolicies    []TransitionPolicy    `json:"transitionPolicies,omitempty"`
	RetentionPolicies     []RetentionPolicy     `json:"retentionPolicies,omitempty"`
	NotificationPolicies  []NotificationPolicy  `json:"notificationPolicies,omitempty"`
	AutomationPolicies    []AutomationPolicy    `json:"automationPolicies,omitempty"`
}

type ValidationPolicy struct {
	Name        string                 `json:"name"`
	Enabled     bool                   `json:"enabled"`
	ResourceType string                `json:"resourceType,omitempty"`
	Conditions  []PolicyCondition      `json:"conditions"`
	Actions     []ValidationAction     `json:"actions"`
	Priority    int                    `json:"priority"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type TransitionPolicy struct {
	Name         string                 `json:"name"`
	Enabled      bool                   `json:"enabled"`
	FromState    string                 `json:"fromState"`
	ToState      string                 `json:"toState"`
	Conditions   []PolicyCondition      `json:"conditions"`
	Actions      []TransitionAction     `json:"actions"`
	Timeout      time.Duration          `json:"timeout,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type RetentionPolicy struct {
	Name         string                 `json:"name"`
	Enabled      bool                   `json:"enabled"`
	ResourceType string                 `json:"resourceType,omitempty"`
	MaxAge       time.Duration          `json:"maxAge"`
	MaxCount     int                    `json:"maxCount,omitempty"`
	Conditions   []PolicyCondition      `json:"conditions"`
	Actions      []RetentionAction      `json:"actions"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type NotificationPolicy struct {
	Name         string                 `json:"name"`
	Enabled      bool                   `json:"enabled"`
	EventTypes   []string               `json:"eventTypes"`
	Conditions   []PolicyCondition      `json:"conditions"`
	Recipients   []NotificationRecipient `json:"recipients"`
	Template     string                 `json:"template,omitempty"`
	Throttling   *ThrottlingConfig      `json:"throttling,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type AutomationPolicy struct {
	Name         string                 `json:"name"`
	Enabled      bool                   `json:"enabled"`
	Trigger      PolicyTrigger          `json:"trigger"`
	Conditions   []PolicyCondition      `json:"conditions"`
	Actions      []AutomationAction     `json:"actions"`
	Schedule     string                 `json:"schedule,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type PolicyCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
	LogicalOp string     `json:"logicalOp,omitempty"`
}

type ValidationAction struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

type TransitionAction struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Delay      time.Duration          `json:"delay,omitempty"`
}

type RetentionAction struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

type NotificationRecipient struct {
	Type       string            `json:"type"`
	Address    string            `json:"address"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

type ThrottlingConfig struct {
	MaxEventsPerMinute int           `json:"maxEventsPerMinute"`
	BurstSize          int           `json:"burstSize"`
	CooldownPeriod     time.Duration `json:"cooldownPeriod"`
}

type PolicyTrigger struct {
	Type       string                 `json:"type"`
	EventTypes []string               `json:"eventTypes,omitempty"`
	Schedule   string                 `json:"schedule,omitempty"`
	Conditions []PolicyCondition      `json:"conditions,omitempty"`
}

type AutomationAction struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Timeout    time.Duration          `json:"timeout,omitempty"`
}