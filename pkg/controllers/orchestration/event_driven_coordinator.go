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
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/interfaces"
)

// EventDrivenCoordinator provides enhanced coordination with event persistence and replay.

type EventDrivenCoordinator struct {
	client client.Client

	logger logr.Logger

	eventBus *EventBus

	phaseTracker *PhaseTracker

	eventStore *PersistentEventStore

	replayManager *EventReplayManager

	// State management.

	coordinationContexts map[string]*CoordinationContext

	mutex sync.RWMutex

	// Configuration.

	enablePersistence bool

	enableReplay bool

	replayWindow time.Duration

	maxRetries int
}

// PersistentEventStore provides Kubernetes-based event persistence.

type PersistentEventStore struct {
	client client.Client

	logger logr.Logger

	namespace string

	configMapPrefix string

	// In-memory cache for performance.

	eventCache map[string][]ProcessingEvent

	cacheMutex sync.RWMutex
}

// EventReplayManager handles event replay for recovery scenarios.

type EventReplayManager struct {
	eventStore *PersistentEventStore

	eventBus *EventBus

	logger logr.Logger

	// Replay configuration.

	maxReplayEvents int

	replayBatchSize int

	replayDelay time.Duration
}

// CoordinationEvent represents a coordination-specific event.

type CoordinationEvent struct {
	ProcessingEvent

	// Coordination-specific fields.

	Coordination CoordinationEventData `json:"coordination"`
}

// CoordinationEventData contains coordination-specific data.

type CoordinationEventData struct {
	PreviousPhase interfaces.ProcessingPhase `json:"previousPhase,omitempty"`

	NextPhase interfaces.ProcessingPhase `json:"nextPhase,omitempty"`

	Dependencies []string `json:"dependencies,omitempty"`

	ConflictID string `json:"conflictId,omitempty"`

	RecoveryAction string `json:"recoveryAction,omitempty"`

	ResourceLocks []string `json:"resourceLocks,omitempty"`

	ParallelContext string `json:"parallelContext,omitempty"`
}

// Enhanced event types for coordination.

const (

	// EventPhaseTransition holds eventphasetransition value.

	EventPhaseTransition = "coordination.phase.transition"

	// EventDependencyResolved holds eventdependencyresolved value.

	EventDependencyResolved = "coordination.dependency.resolved"

	// EventConflictDetected holds eventconflictdetected value.

	EventConflictDetected = "coordination.conflict.detected"

	// EventConflictResolved holds eventconflictresolved value.

	EventConflictResolved = "coordination.conflict.resolved"

	// EventResourceLockAcquired holds eventresourcelockacquired value.

	EventResourceLockAcquired = "coordination.resource.lock.acquired"

	// EventResourceLockReleased holds eventresourcelockreleased value.

	EventResourceLockReleased = "coordination.resource.lock.released"

	// EventRecoveryInitiated holds eventrecoveryinitiated value.

	EventRecoveryInitiated = "coordination.recovery.initiated"

	// EventRecoveryCompleted holds eventrecoverycompleted value.

	EventRecoveryCompleted = "coordination.recovery.completed"

	// EventParallelPhaseStarted holds eventparallelphasestarted value.

	EventParallelPhaseStarted = "coordination.parallel.phase.started"

	// EventParallelPhaseCompleted holds eventparallelphasecompleted value.

	EventParallelPhaseCompleted = "coordination.parallel.phase.completed"

	// EventReplayInitiated holds eventreplayinitiated value.

	EventReplayInitiated = "coordination.replay.initiated"

	// EventReplayCompleted holds eventreplaycompleted value.

	EventReplayCompleted = "coordination.replay.completed"
)

// NewEventDrivenCoordinator creates a new event-driven coordinator.

func NewEventDrivenCoordinator(client client.Client, logger logr.Logger) *EventDrivenCoordinator {
	eventBus := NewEventBus(client, logger)

	phaseTracker := NewPhaseTracker()

	persistentStore := &PersistentEventStore{
		client: client,

		logger: logger.WithName("persistent-event-store"),

		namespace: "nephoran-system",

		configMapPrefix: "intent-events",

		eventCache: make(map[string][]ProcessingEvent),
	}

	replayManager := &EventReplayManager{
		eventStore: persistentStore,

		eventBus: eventBus,

		logger: logger.WithName("replay-manager"),

		maxReplayEvents: 1000,

		replayBatchSize: 10,

		replayDelay: 100 * time.Millisecond,
	}

	coordinator := &EventDrivenCoordinator{
		client: client,

		logger: logger.WithName("event-driven-coordinator"),

		eventBus: eventBus,

		phaseTracker: phaseTracker,

		eventStore: persistentStore,

		replayManager: replayManager,

		coordinationContexts: make(map[string]*CoordinationContext),

		enablePersistence: true,

		enableReplay: true,

		replayWindow: 24 * time.Hour,

		maxRetries: 3,
	}

	// Subscribe to coordination events.

	coordinator.setupEventSubscriptions()

	return coordinator
}

// Start initializes the event-driven coordinator.

func (edc *EventDrivenCoordinator) Start(ctx context.Context) error {
	edc.logger.Info("Starting event-driven coordinator")

	// Start the event bus.

	if err := edc.eventBus.Start(ctx); err != nil {
		return fmt.Errorf("failed to start event bus: %w", err)
	}

	// Start recovery monitoring.

	go edc.monitorForRecovery(ctx)

	edc.logger.Info("Event-driven coordinator started")

	return nil
}

// Stop shuts down the event-driven coordinator.

func (edc *EventDrivenCoordinator) Stop(ctx context.Context) error {
	edc.logger.Info("Stopping event-driven coordinator")

	// Stop the event bus.

	if err := edc.eventBus.Stop(ctx); err != nil {
		edc.logger.Error(err, "Error stopping event bus")
	}

	edc.logger.Info("Event-driven coordinator stopped")

	return nil
}

// setupEventSubscriptions sets up event handlers for coordination.

func (edc *EventDrivenCoordinator) setupEventSubscriptions() {
	// Subscribe to phase transition events.

	edc.eventBus.Subscribe(EventPhaseTransition, edc.handlePhaseTransition)

	// Subscribe to conflict events.

	edc.eventBus.Subscribe(EventConflictDetected, edc.handleConflictDetected)

	edc.eventBus.Subscribe(EventConflictResolved, edc.handleConflictResolved)

	// Subscribe to resource lock events.

	edc.eventBus.Subscribe(EventResourceLockAcquired, edc.handleResourceLockAcquired)

	edc.eventBus.Subscribe(EventResourceLockReleased, edc.handleResourceLockReleased)

	// Subscribe to recovery events.

	edc.eventBus.Subscribe(EventRecoveryInitiated, edc.handleRecoveryInitiated)

	edc.eventBus.Subscribe(EventRecoveryCompleted, edc.handleRecoveryCompleted)

	// Subscribe to all events for persistence (wildcard subscription).

	edc.eventBus.Subscribe("*", edc.handleEventPersistence)
}

// CoordinateIntentWithEvents coordinates an intent using event-driven approach.

func (edc *EventDrivenCoordinator) CoordinateIntentWithEvents(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) error {
	intentID := string(networkIntent.UID)

	// Create coordination context.
	metadataMap := make(map[string]interface{})
	metadataJSON, _ := json.Marshal(metadataMap)

	coordCtx := &CoordinationContext{
		IntentID: intentID,

		CurrentPhase: interfaces.PhaseReceived,

		StartTime: time.Now(),

		LastUpdateTime: time.Now(),

		Metadata: json.RawMessage(metadataJSON),
	}

	// Store coordination context.

	edc.mutex.Lock()

	edc.coordinationContexts[intentID] = coordCtx

	edc.mutex.Unlock()

	// Publish intent received event.

	if err := edc.publishCoordinationEvent(ctx, EventIntentReceived, intentID, interfaces.PhaseReceived, CoordinationEventData{}); err != nil {
		return fmt.Errorf("failed to publish intent received event: %w", err)
	}

	// Start phase processing.

	return edc.initiatePhaseProcessing(ctx, networkIntent, coordCtx)
}

// initiatePhaseProcessing starts the first phase of processing.

func (edc *EventDrivenCoordinator) initiatePhaseProcessing(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, coordCtx *CoordinationContext) error {
	intentID := coordCtx.IntentID

	// Check for existing events to replay.

	if edc.enableReplay {
		if err := edc.replayManager.ReplayEventsForIntent(ctx, intentID); err != nil {
			edc.logger.Error(err, "Failed to replay events", "intentId", intentID)
		}
	}

	// Start LLM processing phase.

	return edc.transitionToPhase(ctx, coordCtx, interfaces.PhaseLLMProcessing)
}

// transitionToPhase transitions to a new processing phase.

func (edc *EventDrivenCoordinator) transitionToPhase(ctx context.Context, coordCtx *CoordinationContext, nextPhase interfaces.ProcessingPhase) error {
	previousPhase := coordCtx.CurrentPhase

	coordCtx.CurrentPhase = nextPhase

	coordCtx.LastUpdateTime = time.Now()

	// Update phase tracker.

	edc.phaseTracker.UpdatePhaseStatus(coordCtx.IntentID, nextPhase, "InProgress")

	// Publish phase transition event.

	transitionData := CoordinationEventData{
		PreviousPhase: previousPhase,

		NextPhase: nextPhase,
	}

	return edc.publishCoordinationEvent(ctx, EventPhaseTransition, coordCtx.IntentID, nextPhase, transitionData)
}

// publishCoordinationEvent publishes a coordination-specific event.

func (edc *EventDrivenCoordinator) publishCoordinationEvent(ctx context.Context, eventType, intentID string, phase interfaces.ProcessingPhase, coordData CoordinationEventData) error {
	// Marshal empty data map to JSON for json.RawMessage
	dataMap := make(map[string]interface{})
	dataJSON, _ := json.Marshal(dataMap)
	
	event := CoordinationEvent{
		ProcessingEvent: ProcessingEvent{
			Type: eventType,

			Source: "coordination-controller",

			IntentID: intentID,

			Phase: phase,

			Success: true,

			Data: json.RawMessage(dataJSON),

			Timestamp: time.Now(),

			CorrelationID: fmt.Sprintf("%s-%d", intentID, time.Now().UnixNano()),
		},

		Coordination: coordData,
	}

	return edc.eventBus.Publish(ctx, event.ProcessingEvent)
}

// Event handlers.

// handlePhaseTransition handles phase transition events.

func (edc *EventDrivenCoordinator) handlePhaseTransition(ctx context.Context, event ProcessingEvent) error {
	edc.logger.Info("Handling phase transition", "intentId", event.IntentID, "phase", event.Phase)

	// Update coordination context.

	edc.mutex.Lock()

	if coordCtx, exists := edc.coordinationContexts[event.IntentID]; exists {

		coordCtx.LastUpdateTime = time.Now()

		// Add to completed phases when transitioning from it.

		if coordCtx.CurrentPhase != interfaces.PhaseReceived {
			coordCtx.CompletedPhases = append(coordCtx.CompletedPhases, coordCtx.CurrentPhase)
		}

		coordCtx.CurrentPhase = event.Phase

	}

	edc.mutex.Unlock()

	return nil
}

// handleConflictDetected handles conflict detection events.

func (edc *EventDrivenCoordinator) handleConflictDetected(ctx context.Context, event ProcessingEvent) error {
	edc.logger.Info("Handling conflict detection", "intentId", event.IntentID, "conflict", event.Data)

	// Extract conflict data from event.
	var conflictData map[string]interface{}
	if err := json.Unmarshal(event.Data, &conflictData); err != nil {
		return fmt.Errorf("failed to unmarshal conflict data: %w", err)
	}

	conflictID, ok := conflictData["conflictId"].(string)
	if !ok || conflictID == "" {
		return fmt.Errorf("missing conflict ID in conflict detected event")
	}

	// Update coordination context with conflict.

	edc.mutex.Lock()

	if coordCtx, exists := edc.coordinationContexts[event.IntentID]; exists {

		conflictType, _ := conflictData["type"].(string)
		conflict := Conflict{
			ID: conflictID,
			Type: conflictType,

			InvolvedIntents: []string{event.IntentID},

			DetectedAt: time.Now(),
		}

		coordCtx.Conflicts = append(coordCtx.Conflicts, conflict)

	}

	edc.mutex.Unlock()

	return nil
}

// handleConflictResolved handles conflict resolution events.

func (edc *EventDrivenCoordinator) handleConflictResolved(ctx context.Context, event ProcessingEvent) error {
	edc.logger.Info("Handling conflict resolution", "intentId", event.IntentID)

	var conflictData map[string]interface{}
	if err := json.Unmarshal(event.Data, &conflictData); err != nil {
		return fmt.Errorf("failed to unmarshal conflict data: %w", err)
	}
	conflictID, _ := conflictData["conflictId"].(string)

	// Remove resolved conflict from coordination context.

	edc.mutex.Lock()

	if coordCtx, exists := edc.coordinationContexts[event.IntentID]; exists {
		for i, conflict := range coordCtx.Conflicts {
			if conflict.ID == conflictID {

				coordCtx.Conflicts = append(coordCtx.Conflicts[:i], coordCtx.Conflicts[i+1:]...)

				break

			}
		}
	}

	edc.mutex.Unlock()

	return nil
}

// handleResourceLockAcquired handles resource lock acquisition events.

func (edc *EventDrivenCoordinator) handleResourceLockAcquired(ctx context.Context, event ProcessingEvent) error {
	edc.logger.Info("Handling resource lock acquired", "intentId", event.IntentID)

	var lockData map[string]interface{}
	if err := json.Unmarshal(event.Data, &lockData); err != nil {
		return fmt.Errorf("failed to unmarshal lock data: %w", err)
	}
	lockID, _ := lockData["lockId"].(string)

	// Add lock to coordination context.

	edc.mutex.Lock()

	if coordCtx, exists := edc.coordinationContexts[event.IntentID]; exists {
		coordCtx.Locks = append(coordCtx.Locks, lockID)
	}

	edc.mutex.Unlock()

	return nil
}

// handleResourceLockReleased handles resource lock release events.

func (edc *EventDrivenCoordinator) handleResourceLockReleased(ctx context.Context, event ProcessingEvent) error {
	edc.logger.Info("Handling resource lock released", "intentId", event.IntentID)

	var lockData map[string]interface{}
	if err := json.Unmarshal(event.Data, &lockData); err != nil {
		return fmt.Errorf("failed to unmarshal lock data: %w", err)
	}
	lockID, _ := lockData["lockId"].(string)

	// Remove lock from coordination context.

	edc.mutex.Lock()

	if coordCtx, exists := edc.coordinationContexts[event.IntentID]; exists {
		for i, lock := range coordCtx.Locks {
			if lock == lockID {

				coordCtx.Locks = append(coordCtx.Locks[:i], coordCtx.Locks[i+1:]...)

				break

			}
		}
	}

	edc.mutex.Unlock()

	return nil
}

// handleRecoveryInitiated handles recovery initiation events.

func (edc *EventDrivenCoordinator) handleRecoveryInitiated(ctx context.Context, event ProcessingEvent) error {
	edc.logger.Info("Handling recovery initiated", "intentId", event.IntentID)

	// Record recovery in coordination context.
	var recoveryData map[string]interface{}
	if err := json.Unmarshal(event.Data, &recoveryData); err != nil {
		return fmt.Errorf("failed to unmarshal recovery data: %w", err)
	}

	edc.mutex.Lock()

	if coordCtx, exists := edc.coordinationContexts[event.IntentID]; exists {

		coordCtx.RetryCount++

		recoveryAction, _ := recoveryData["action"].(string)

		coordCtx.ErrorHistory = append(coordCtx.ErrorHistory, fmt.Sprintf("Recovery initiated: %s", recoveryAction))

	}

	edc.mutex.Unlock()

	return nil
}

// handleRecoveryCompleted handles recovery completion events.

func (edc *EventDrivenCoordinator) handleRecoveryCompleted(ctx context.Context, event ProcessingEvent) error {
	edc.logger.Info("Handling recovery completed", "intentId", event.IntentID)

	// Update coordination context.
	var recoveryData map[string]interface{}
	if err := json.Unmarshal(event.Data, &recoveryData); err != nil {
		return fmt.Errorf("failed to unmarshal recovery data: %w", err)
	}

	edc.mutex.Lock()

	if coordCtx, exists := edc.coordinationContexts[event.IntentID]; exists {

		recoveryAction, _ := recoveryData["action"].(string)

		coordCtx.ErrorHistory = append(coordCtx.ErrorHistory, fmt.Sprintf("Recovery completed: %s", recoveryAction))

	}

	edc.mutex.Unlock()

	return nil
}

// handleEventPersistence handles persistent storage of all events.

func (edc *EventDrivenCoordinator) handleEventPersistence(ctx context.Context, event ProcessingEvent) error {
	if !edc.enablePersistence {
		return nil
	}

	return edc.eventStore.PersistEvent(ctx, event)
}

// monitorForRecovery monitors for failed intents and initiates recovery.

func (edc *EventDrivenCoordinator) monitorForRecovery(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)

	defer ticker.Stop()

	for {
		select {

		case <-ticker.C:

			edc.checkForRecoveryNeeded(ctx)

		case <-ctx.Done():

			return

		}
	}
}

// checkForRecoveryNeeded checks coordination contexts for recovery needs.

func (edc *EventDrivenCoordinator) checkForRecoveryNeeded(ctx context.Context) {
	edc.mutex.RLock()

	contexts := make([]*CoordinationContext, 0, len(edc.coordinationContexts))

	for _, coordCtx := range edc.coordinationContexts {
		contexts = append(contexts, coordCtx)
	}

	edc.mutex.RUnlock()

	for _, coordCtx := range contexts {
		// Check if intent is stuck (no update in 10 minutes).

		if time.Since(coordCtx.LastUpdateTime) > 10*time.Minute {

			edc.logger.Info("Intent appears stuck, initiating recovery", "intentId", coordCtx.IntentID, "currentPhase", coordCtx.CurrentPhase)

			// Publish recovery initiated event.

			recoveryData := CoordinationEventData{
				RecoveryAction: "timeout-recovery",
			}

			if err := edc.publishCoordinationEvent(ctx, EventRecoveryInitiated, coordCtx.IntentID, coordCtx.CurrentPhase, recoveryData); err != nil {
				edc.logger.Error(err, "Failed to publish recovery event", "intentId", coordCtx.IntentID)
			}

		}
	}
}

// GetCoordinationContext retrieves the coordination context for an intent.

func (edc *EventDrivenCoordinator) GetCoordinationContext(intentID string) (*CoordinationContext, bool) {
	edc.mutex.RLock()

	defer edc.mutex.RUnlock()

	coordCtx, exists := edc.coordinationContexts[intentID]

	if exists {

		// Return a field-by-field copy to avoid copying mutex.

		contextCopy := &CoordinationContext{
			IntentID: coordCtx.IntentID,

			CorrelationID: coordCtx.CorrelationID,

			StartTime: coordCtx.StartTime,

			CurrentPhase: coordCtx.CurrentPhase,

			RetryCount: coordCtx.RetryCount,

			LastActivity: coordCtx.LastActivity,

			LastUpdateTime: coordCtx.LastUpdateTime,
		}

		// Deep copy slices to avoid shared references.

		if coordCtx.PhaseHistory != nil {

			contextCopy.PhaseHistory = make([]PhaseExecution, len(coordCtx.PhaseHistory))

			copy(contextCopy.PhaseHistory, coordCtx.PhaseHistory)

		}

		if coordCtx.Locks != nil {

			contextCopy.Locks = make([]string, len(coordCtx.Locks))

			copy(contextCopy.Locks, coordCtx.Locks)

		}

		if coordCtx.Conflicts != nil {

			contextCopy.Conflicts = make([]Conflict, len(coordCtx.Conflicts))

			copy(contextCopy.Conflicts, coordCtx.Conflicts)

		}

		if coordCtx.CompletedPhases != nil {

			contextCopy.CompletedPhases = make([]interfaces.ProcessingPhase, len(coordCtx.CompletedPhases))

			copy(contextCopy.CompletedPhases, coordCtx.CompletedPhases)

		}

		if coordCtx.ErrorHistory != nil {

			contextCopy.ErrorHistory = make([]string, len(coordCtx.ErrorHistory))

			copy(contextCopy.ErrorHistory, coordCtx.ErrorHistory)

		}

		// Deep copy json.RawMessage to avoid shared references.

		if coordCtx.Metadata != nil {

			// Copy the raw JSON bytes
			metadataCopy := make(json.RawMessage, len(coordCtx.Metadata))
			copy(metadataCopy, coordCtx.Metadata)
			contextCopy.Metadata = metadataCopy

		}

		return contextCopy, true

	}

	return nil, false
}

// PersistEvent stores an event in Kubernetes ConfigMaps.

func (ps *PersistentEventStore) PersistEvent(ctx context.Context, event ProcessingEvent) error {
	// Use ConfigMap for event persistence.

	configMapName := fmt.Sprintf("%s-%s", ps.configMapPrefix, event.IntentID)

	// Get or create ConfigMap.

	configMap := &corev1.ConfigMap{}

	err := ps.client.Get(ctx, client.ObjectKey{
		Name: configMapName,

		Namespace: ps.namespace,
	}, configMap)
	if err != nil {
		// Create new ConfigMap.

		configMap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: configMapName,

				Namespace: ps.namespace,

				Labels: map[string]string{
					"nephoran.com/intent-id": event.IntentID,

					"nephoran.com/event-store": "true",
				},
			},

			Data: make(map[string]string),
		}
	}

	// Add event to ConfigMap.

	eventKey := fmt.Sprintf("event-%d", time.Now().UnixNano())

	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	configMap.Data[eventKey] = string(eventData)

	// Update ConfigMap.

	if err := ps.client.Update(ctx, configMap); err != nil {
		// If update fails, try create.

		if err := ps.client.Create(ctx, configMap); err != nil {
			return fmt.Errorf("failed to persist event: %w", err)
		}
	}

	// Update cache.

	ps.cacheMutex.Lock()

	ps.eventCache[event.IntentID] = append(ps.eventCache[event.IntentID], event)

	ps.cacheMutex.Unlock()

	return nil
}

// ReplayEventsForIntent replays events for a specific intent.

func (rm *EventReplayManager) ReplayEventsForIntent(ctx context.Context, intentID string) error {
	rm.logger.Info("Starting event replay", "intentId", intentID)

	// Get events from persistent store.

	events, err := rm.eventStore.GetEventsForIntent(ctx, intentID)
	if err != nil {
		return fmt.Errorf("failed to get events for replay: %w", err)
	}

	if len(events) == 0 {

		rm.logger.Info("No events found for replay", "intentId", intentID)

		return nil

	}

	// Replay events in batches.

	for i := 0; i < len(events); i += rm.replayBatchSize {

		end := i + rm.replayBatchSize

		if end > len(events) {
			end = len(events)
		}

		batch := events[i:end]

		for _, event := range batch {
			// Republish event.

			if err := rm.eventBus.Publish(ctx, event); err != nil {
				rm.logger.Error(err, "Failed to replay event", "eventType", event.Type, "intentId", intentID)
			}
		}

		// Delay between batches.

		select {

		case <-time.After(rm.replayDelay):

		case <-ctx.Done():

			return ctx.Err()

		}

	}

	rm.logger.Info("Event replay completed", "intentId", intentID, "eventsReplayed", len(events))

	return nil
}

// GetEventsForIntent retrieves events for an intent from persistent storage.

func (ps *PersistentEventStore) GetEventsForIntent(ctx context.Context, intentID string) ([]ProcessingEvent, error) {
	// Check cache first.

	ps.cacheMutex.RLock()

	if events, exists := ps.eventCache[intentID]; exists {

		ps.cacheMutex.RUnlock()

		return events, nil

	}

	ps.cacheMutex.RUnlock()

	// Load from ConfigMap.

	configMapName := fmt.Sprintf("%s-%s", ps.configMapPrefix, intentID)

	configMap := &corev1.ConfigMap{}

	err := ps.client.Get(ctx, client.ObjectKey{
		Name: configMapName,

		Namespace: ps.namespace,
	}, configMap)
	if err != nil {
<<<<<<< HEAD
=======
		// If ConfigMap doesn't exist, return empty events list (no events to replay)
		if client.IgnoreNotFound(err) == nil {
			return []ProcessingEvent{}, nil
		}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		return nil, fmt.Errorf("failed to get events ConfigMap: %w", err)
	}

	// Parse events.

	events := make([]ProcessingEvent, 0, len(configMap.Data))

	for _, eventData := range configMap.Data {

		var event ProcessingEvent

		if err := json.Unmarshal([]byte(eventData), &event); err != nil {

			ps.logger.Error(err, "Failed to unmarshal event")

			continue

		}

		events = append(events, event)

	}

	// Update cache.

	ps.cacheMutex.Lock()

	ps.eventCache[intentID] = events

	ps.cacheMutex.Unlock()

	return events, nil
}
