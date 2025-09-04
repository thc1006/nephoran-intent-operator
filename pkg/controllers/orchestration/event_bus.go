/*

Copyright 2025.



Licensed under the Apache License, Version 2.0 (the "License");

you may not use this file except in compliance with the License.

You may obtain a copy of the License at



    http://www.apache.org/licenses/LICENSE-2.0



Unless required by applicable law or agreed to in writing, software

distributed under the License is distributed on an "AS IS" BASIS,

WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.

See the License for the specific language governing permissions and

limitations under the License.

*/

package orchestration

import (
	
	"encoding/json"
"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/interfaces"
)

// ProcessingEvent represents an event during intent processing.

type ProcessingEvent struct {
	Type string `json:"type"`

	Source string `json:"source"`

	IntentID string `json:"intentId"`

	Phase interfaces.ProcessingPhase `json:"phase"`

	Success bool `json:"success"`

	Data json.RawMessage `json:"data"`

	Timestamp time.Time `json:"timestamp"`

	CorrelationID string `json:"correlationId"`

	Metadata map[string]string `json:"metadata,omitempty"`
}

// EventHandler defines the interface for event handling.

type EventHandler func(ctx context.Context, event ProcessingEvent) error

// EventBus provides decoupled communication between controllers.

type EventBus struct {
	client client.Client

	recorder record.EventRecorder

	logger logr.Logger

	// Event handling.

	subscribers map[string][]EventHandler

	mutex sync.RWMutex

	// Event persistence.

	eventStore *EventStore

	// Configuration.

	bufferSize int

	maxRetries int

	retryBackoff time.Duration

	// Channels for event processing.

	eventChan chan ProcessingEvent

	stopChan chan bool

	started bool
}

// Event types for phase transitions.

const (

	// EventIntentReceived holds eventintentreceived value.

	EventIntentReceived = "intent.received"

	// EventLLMProcessingStarted holds eventllmprocessingstarted value.

	EventLLMProcessingStarted = "llm.processing.started"

	// EventLLMProcessingCompleted holds eventllmprocessingcompleted value.

	EventLLMProcessingCompleted = "llm.processing.completed"

	// EventLLMProcessingFailed holds eventllmprocessingfailed value.

	EventLLMProcessingFailed = "llm.processing.failed"

	// EventResourcePlanningStarted holds eventresourceplanningstarted value.

	EventResourcePlanningStarted = "resource.planning.started"

	// EventResourcePlanningCompleted holds eventresourceplanningcompleted value.

	EventResourcePlanningCompleted = "resource.planning.completed"

	// EventResourcePlanningFailed holds eventresourceplanningfailed value.

	EventResourcePlanningFailed = "resource.planning.failed"

	// EventManifestGenerationStarted holds eventmanifestgenerationstarted value.

	EventManifestGenerationStarted = "manifest.generation.started"

	// EventManifestGenerationCompleted holds eventmanifestgenerationcompleted value.

	EventManifestGenerationCompleted = "manifest.generation.completed"

	// EventManifestGenerationFailed holds eventmanifestgenerationfailed value.

	EventManifestGenerationFailed = "manifest.generation.failed"

	// EventGitOpsCommitStarted holds eventgitopscommitstarted value.

	EventGitOpsCommitStarted = "gitops.commit.started"

	// EventGitOpsCommitCompleted holds eventgitopscommitcompleted value.

	EventGitOpsCommitCompleted = "gitops.commit.completed"

	// EventGitOpsCommitFailed holds eventgitopscommitfailed value.

	EventGitOpsCommitFailed = "gitops.commit.failed"

	// EventDeploymentVerificationStarted holds eventdeploymentverificationstarted value.

	EventDeploymentVerificationStarted = "deployment.verification.started"

	// EventDeploymentVerificationCompleted holds eventdeploymentverificationcompleted value.

	EventDeploymentVerificationCompleted = "deployment.verification.completed"

	// EventDeploymentVerificationFailed holds eventdeploymentverificationfailed value.

	EventDeploymentVerificationFailed = "deployment.verification.failed"

	// EventProcessingFailed holds eventprocessingfailed value.

	EventProcessingFailed = "processing.failed"

	// EventRetryRequired holds eventretryrequired value.

	EventRetryRequired = "retry.required"

	// EventIntentCompleted holds eventintentcompleted value.

	EventIntentCompleted = "intent.completed"

	// EventIntentFailed holds eventintentfailed value.

	EventIntentFailed = "intent.failed"

	// Cross-phase coordination events.

	EventDependencyMet = "dependency.met"

	// EventResourceAllocated holds eventresourceallocated value.

	EventResourceAllocated = "resource.allocated"

	// EventResourceDeallocated holds eventresourcedeallocated value.

	EventResourceDeallocated = "resource.deallocated"

	// EventErrorRecovery holds eventerrorrecovery value.

	EventErrorRecovery = "error.recovery"

	// EventParallelPhaseSync holds eventparallelphasesync value.

	EventParallelPhaseSync = "parallel.phase.sync"
)

// NewEventBus creates a new EventBus.

func NewEventBus(client client.Client, logger logr.Logger) *EventBus {
	return &EventBus{
		client: client,

		logger: logger.WithName("event-bus"),

		subscribers: make(map[string][]EventHandler),

		eventStore: NewEventStore(),

		bufferSize: 1000,

		maxRetries: 3,

		retryBackoff: 1 * time.Second,

		eventChan: make(chan ProcessingEvent, 1000),

		stopChan: make(chan bool),
	}
}

// Subscribe registers an event handler for a specific event type.

func (e *EventBus) Subscribe(eventType string, handler EventHandler) error {
	e.mutex.Lock()

	defer e.mutex.Unlock()

	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	e.subscribers[eventType] = append(e.subscribers[eventType], handler)

	e.logger.Info("Subscribed to event type", "eventType", eventType, "handlerCount", len(e.subscribers[eventType]))

	return nil
}

// Unsubscribe removes an event handler (note: removes all handlers for the event type).

func (e *EventBus) Unsubscribe(eventType string) {
	e.mutex.Lock()

	defer e.mutex.Unlock()

	delete(e.subscribers, eventType)

	e.logger.Info("Unsubscribed from event type", "eventType", eventType)
}

// Publish publishes an event to all subscribers.

func (e *EventBus) Publish(ctx context.Context, event ProcessingEvent) error {
	if !e.started {
		return fmt.Errorf("event bus not started")
	}

	// Add timestamp if not set.

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Store event for persistence.

	if err := e.eventStore.Store(event); err != nil {
		e.logger.Error(err, "Failed to store event", "eventType", event.Type, "intentId", event.IntentID)
	}

	// Send to processing channel (non-blocking).

	select {

	case e.eventChan <- event:

		return nil

	default:

		e.logger.Error(fmt.Errorf("event channel full"), "Dropping event", "eventType", event.Type, "intentId", event.IntentID)

		return fmt.Errorf("event bus buffer full")

	}
}

// PublishPhaseEvent publishes a phase-specific event.

func (e *EventBus) PublishPhaseEvent(ctx context.Context, phase interfaces.ProcessingPhase, eventType, intentID string, success bool, data map[string]interface{}) error {
	// Marshal data to JSON for json.RawMessage
	dataJSON, _ := json.Marshal(data)
	
	event := ProcessingEvent{
		Type: eventType,

		Source: string(phase),

		IntentID: intentID,

		Phase: phase,

		Success: success,

		Data: json.RawMessage(dataJSON),

		Timestamp: time.Now(),

		CorrelationID: fmt.Sprintf("%s-%d", intentID, time.Now().UnixNano()),
	}

	return e.Publish(ctx, event)
}

// Start starts the event bus processing.

func (e *EventBus) Start(ctx context.Context) error {
	if e.started {
		return fmt.Errorf("event bus already started")
	}

	e.started = true

	// Start event processing goroutine.

	go e.processEvents(ctx)

	e.logger.Info("Event bus started")

	return nil
}

// Stop stops the event bus.

func (e *EventBus) Stop(ctx context.Context) error {
	if !e.started {
		return nil
	}

	e.logger.Info("Stopping event bus")

	close(e.stopChan)

	e.started = false

	// Wait for processing to complete with timeout.

	select {

	case <-time.After(10 * time.Second):

		e.logger.Info("Event bus stopped with timeout")

	case <-ctx.Done():

		e.logger.Info("Event bus stopped due to context cancellation")

	}

	return nil
}

// processEvents processes events from the event channel.

func (e *EventBus) processEvents(ctx context.Context) {
	e.logger.Info("Started event processing")

	for {
		select {

		case event := <-e.eventChan:

			e.handleEvent(ctx, event)

		case <-e.stopChan:

			e.logger.Info("Event processing stopped")

			return

		case <-ctx.Done():

			e.logger.Info("Event processing cancelled")

			return

		}
	}
}

// handleEvent processes a single event.

func (e *EventBus) handleEvent(ctx context.Context, event ProcessingEvent) {
	log := e.logger.WithValues("eventType", event.Type, "intentId", event.IntentID, "phase", event.Phase)

	// Get subscribers for this event type.

	e.mutex.RLock()

	handlers := e.subscribers[event.Type]

	// Also get wildcard subscribers (if any).

	wildcardHandlers := e.subscribers["*"]

	allHandlers := append(handlers, wildcardHandlers...)

	e.mutex.RUnlock()

	if len(allHandlers) == 0 {

		log.V(1).Info("No subscribers for event type")

		return

	}

	log.Info("Processing event", "handlerCount", len(allHandlers))

	// Process handlers with retry logic.

	for i, handler := range allHandlers {
		if err := e.executeHandlerWithRetry(ctx, handler, event); err != nil {

			log.Error(err, "Handler failed permanently", "handlerIndex", i)

			// Record handler failure.

			e.recordHandlerFailure(event, i, err)

		}
	}

	log.V(1).Info("Event processed successfully")
}

// executeHandlerWithRetry executes a handler with retry logic.

func (e *EventBus) executeHandlerWithRetry(ctx context.Context, handler EventHandler, event ProcessingEvent) error {
	var lastErr error

	for attempt := range e.maxRetries {

		if attempt > 0 {
			// Wait before retry.

			select {

			case <-time.After(e.retryBackoff * time.Duration(attempt)):

			case <-ctx.Done():

				return ctx.Err()

			}
		}

		if err := handler(ctx, event); err != nil {

			lastErr = err

			e.logger.Error(err, "Handler execution failed", "attempt", attempt+1, "eventType", event.Type)

			continue

		}

		return nil // Success

	}

	return fmt.Errorf("handler failed after %d attempts: %w", e.maxRetries, lastErr)
}

// recordHandlerFailure records a handler failure for monitoring.

func (e *EventBus) recordHandlerFailure(event ProcessingEvent, handlerIndex int, err error) {
	// Create a Kubernetes event for the failure.

	if e.recorder != nil {

		_ = fmt.Sprintf("Event handler %d failed for event type %s: %v", handlerIndex, event.Type, err)

		// We would need an object reference here - in practice, this would be the NetworkIntent.

		// For now, we'll just log it.

		e.logger.Error(err, "Handler failure recorded", "eventType", event.Type, "handlerIndex", handlerIndex)

	}
}

// GetEventHistory retrieves event history for an intent.

func (e *EventBus) GetEventHistory(ctx context.Context, intentID string) ([]ProcessingEvent, error) {
	return e.eventStore.GetEventsByIntentID(intentID)
}

// GetEventsByType retrieves events by type.

func (e *EventBus) GetEventsByType(ctx context.Context, eventType string, limit int) ([]ProcessingEvent, error) {
	return e.eventStore.GetEventsByType(eventType, limit)
}

// EventStore provides event persistence.

type EventStore struct {
	events []ProcessingEvent

	mutex sync.RWMutex

	// Configuration.

	maxEvents int

	retentionTime time.Duration
}

// NewEventStore creates a new event store.

func NewEventStore() *EventStore {
	return &EventStore{
		events: make([]ProcessingEvent, 0),

		maxEvents: 10000, // Keep last 10k events

		retentionTime: 24 * time.Hour, // Keep events for 24 hours

	}
}

// Store stores an event.

func (es *EventStore) Store(event ProcessingEvent) error {
	es.mutex.Lock()

	defer es.mutex.Unlock()

	// Add event.

	es.events = append(es.events, event)

	// Cleanup old events if needed.

	if len(es.events) > es.maxEvents {

		// Remove oldest 10% of events.

		removeCount := es.maxEvents / 10

		es.events = es.events[removeCount:]

	}

	// Cleanup events older than retention time.

	cutoff := time.Now().Add(-es.retentionTime)

	for i, e := range es.events {
		if e.Timestamp.After(cutoff) {

			es.events = es.events[i:]

			break

		}
	}

	return nil
}

// GetEventsByIntentID retrieves events for a specific intent.

func (es *EventStore) GetEventsByIntentID(intentID string) ([]ProcessingEvent, error) {
	es.mutex.RLock()

	defer es.mutex.RUnlock()

	var result []ProcessingEvent

	for _, event := range es.events {
		if event.IntentID == intentID {
			result = append(result, event)
		}
	}

	return result, nil
}

// GetEventsByType retrieves events by type.

func (es *EventStore) GetEventsByType(eventType string, limit int) ([]ProcessingEvent, error) {
	es.mutex.RLock()

	defer es.mutex.RUnlock()

	var result []ProcessingEvent

	count := 0

	// Return most recent events first.

	for i := len(es.events) - 1; i >= 0 && count < limit; i-- {
		if es.events[i].Type == eventType {

			result = append(result, es.events[i])

			count++

		}
	}

	return result, nil
}

// EventBusMetrics provides metrics for the event bus.

type EventBusMetrics struct {
	TotalEventsPublished int64 `json:"totalEventsPublished"`

	TotalEventsProcessed int64 `json:"totalEventsProcessed"`

	EventsPerType map[string]int64 `json:"eventsPerType"`

	FailedHandlers int64 `json:"failedHandlers"`

	AverageProcessingTime time.Duration `json:"averageProcessingTime"`

	BufferUtilization float64 `json:"bufferUtilization"`

	mutex sync.RWMutex
}

// NewEventBusMetrics creates new metrics collector.

func NewEventBusMetrics() *EventBusMetrics {
	return &EventBusMetrics{
		EventsPerType: make(map[string]int64),
	}
}

// RecordEventPublished records an event publication.

func (m *EventBusMetrics) RecordEventPublished(eventType string) {
	m.mutex.Lock()

	defer m.mutex.Unlock()

	m.TotalEventsPublished++

	m.EventsPerType[eventType]++
}

// RecordEventProcessed records event processing completion.

func (m *EventBusMetrics) RecordEventProcessed(processingTime time.Duration) {
	m.mutex.Lock()

	defer m.mutex.Unlock()

	m.TotalEventsProcessed++

	// Update average processing time (simple moving average).

	if m.AverageProcessingTime == 0 {
		m.AverageProcessingTime = processingTime
	} else {
		m.AverageProcessingTime = (m.AverageProcessingTime + processingTime) / 2
	}
}

// RecordHandlerFailure records a handler failure.

func (m *EventBusMetrics) RecordHandlerFailure() {
	m.mutex.Lock()

	defer m.mutex.Unlock()

	m.FailedHandlers++
}

// GetMetrics returns current metrics.

func (m *EventBusMetrics) GetMetrics() map[string]interface{} {
	m.mutex.RLock()

	defer m.mutex.RUnlock()

	return make(map[string]interface{})
}

