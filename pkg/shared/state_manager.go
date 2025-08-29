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




package shared



import (

	"context"

	"fmt"

	"strconv"

	"sync"

	"time"



	"github.com/go-logr/logr"



	nephoranv1 "github.com/nephio-project/nephoran-intent-operator/api/v1"

	"github.com/nephio-project/nephoran-intent-operator/pkg/controllers/interfaces"



	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/apimachinery/pkg/types"

	"k8s.io/client-go/tools/record"



	"sigs.k8s.io/controller-runtime/pkg/client"

)



// StateManager provides centralized state management for all controllers.

type StateManager struct {

	client   client.Client

	recorder record.EventRecorder

	logger   logr.Logger

	scheme   *runtime.Scheme



	// State caching and synchronization.

	stateCache *StateCache

	stateLocks *LockManager



	// Event notification.

	eventBus EventBus



	// Configuration.

	config *StateManagerConfig



	// Metrics.

	metrics *StateMetrics



	// Internal coordination.

	mutex   sync.RWMutex

	started bool

}



// StateManagerConfig provides configuration for the state manager.

type StateManagerConfig struct {

	// Cache configuration.

	CacheSize            int           `json:"cacheSize"`

	CacheTTL             time.Duration `json:"cacheTTL"`

	CacheCleanupInterval time.Duration `json:"cacheCleanupInterval"`



	// Lock configuration.

	LockTimeout       time.Duration `json:"lockTimeout"`

	LockRetryInterval time.Duration `json:"lockRetryInterval"`

	MaxLockRetries    int           `json:"maxLockRetries"`



	// State synchronization.

	SyncInterval           time.Duration `json:"syncInterval"`

	StateValidationEnabled bool          `json:"stateValidationEnabled"`

	ConflictResolutionMode string        `json:"conflictResolutionMode"` // "latest", "merge", "manual"



	// Performance.

	BatchStateUpdates bool          `json:"batchStateUpdates"`

	BatchSize         int           `json:"batchSize"`

	BatchTimeout      time.Duration `json:"batchTimeout"`

}



// DefaultStateManagerConfig returns default configuration.

func DefaultStateManagerConfig() *StateManagerConfig {

	return &StateManagerConfig{

		CacheSize:              10000,

		CacheTTL:               30 * time.Minute,

		CacheCleanupInterval:   5 * time.Minute,

		LockTimeout:            30 * time.Second,

		LockRetryInterval:      1 * time.Second,

		MaxLockRetries:         10,

		SyncInterval:           10 * time.Second,

		StateValidationEnabled: true,

		ConflictResolutionMode: "latest",

		BatchStateUpdates:      true,

		BatchSize:              50,

		BatchTimeout:           5 * time.Second,

	}

}



// NewStateManager creates a new state manager.

func NewStateManager(client client.Client, recorder record.EventRecorder, logger logr.Logger, scheme *runtime.Scheme, eventBus EventBus) *StateManager {

	config := DefaultStateManagerConfig()



	return &StateManager{

		client:     client,

		recorder:   recorder,

		logger:     logger.WithName("state-manager"),

		scheme:     scheme,

		stateCache: NewStateCache(config.CacheSize, config.CacheTTL),

		stateLocks: NewLockManager(config.LockTimeout, config.LockRetryInterval, config.MaxLockRetries),

		eventBus:   eventBus,

		config:     config,

		metrics:    NewStateMetrics(),

	}

}



// Start starts the state manager.

func (sm *StateManager) Start(ctx context.Context) error {

	sm.mutex.Lock()

	defer sm.mutex.Unlock()



	if sm.started {

		return fmt.Errorf("state manager already started")

	}



	// Start background processes.

	go sm.runCacheCleanup(ctx)

	go sm.runStateSync(ctx)

	go sm.runMetricsCollection(ctx)



	sm.started = true

	sm.logger.Info("State manager started")



	return nil

}



// Stop stops the state manager.

func (sm *StateManager) Stop(ctx context.Context) error {

	sm.mutex.Lock()

	defer sm.mutex.Unlock()



	if !sm.started {

		return nil

	}



	sm.started = false

	sm.logger.Info("State manager stopped")



	return nil

}



// GetIntentState retrieves the current state of a network intent.

func (sm *StateManager) GetIntentState(ctx context.Context, namespacedName types.NamespacedName) (*IntentState, error) {

	// Try cache first.

	if state := sm.stateCache.Get(namespacedName.String()); state != nil {

		sm.metrics.RecordCacheHit()

		return state.(*IntentState), nil

	}



	sm.metrics.RecordCacheMiss()



	// Fetch from Kubernetes API.

	intent := &nephoranv1.NetworkIntent{}

	if err := sm.client.Get(ctx, namespacedName, intent); err != nil {

		return nil, fmt.Errorf("failed to get network intent: %w", err)

	}



	// Convert to internal state representation.

	state := sm.convertToIntentState(intent)



	// Cache the state.

	sm.stateCache.Set(namespacedName.String(), state, sm.config.CacheTTL)



	return state, nil

}



// UpdateIntentState updates the state of a network intent.

func (sm *StateManager) UpdateIntentState(ctx context.Context, namespacedName types.NamespacedName, updateFn func(*IntentState) error) error {

	// Acquire lock for this intent.

	lockKey := namespacedName.String()

	if err := sm.stateLocks.AcquireLock(ctx, lockKey); err != nil {

		return fmt.Errorf("failed to acquire lock for intent %s: %w", lockKey, err)

	}

	defer sm.stateLocks.ReleaseLock(lockKey)



	// Get current state.

	state, err := sm.GetIntentState(ctx, namespacedName)

	if err != nil {

		return fmt.Errorf("failed to get intent state: %w", err)

	}



	// Apply update.

	oldVersion := state.Version

	if err := updateFn(state); err != nil {

		return fmt.Errorf("failed to apply state update: %w", err)

	}



	// Increment version (convert from string to int, increment, convert back).

	version := 1

	if state.Version != "" {

		if v, err := strconv.Atoi(state.Version); err == nil {

			version = v + 1

		}

	}

	state.Version = strconv.Itoa(version)

	state.LastModified = time.Now()



	// Validate state if enabled.

	if sm.config.StateValidationEnabled {

		if err := sm.validateState(state); err != nil {

			return fmt.Errorf("state validation failed: %w", err)

		}

	}



	// Update in Kubernetes.

	if err := sm.updateKubernetesState(ctx, namespacedName, state); err != nil {

		// Rollback version on failure.

		state.Version = oldVersion

		return fmt.Errorf("failed to update kubernetes state: %w", err)

	}



	// Update cache.

	sm.stateCache.Set(namespacedName.String(), state, sm.config.CacheTTL)



	// Publish state change event.

	sm.publishStateChangeEvent(ctx, namespacedName, state)



	sm.metrics.RecordStateUpdate()



	return nil

}



// TransitionPhase transitions an intent to a new phase.

func (sm *StateManager) TransitionPhase(ctx context.Context, namespacedName types.NamespacedName, newPhase interfaces.ProcessingPhase, metadata map[string]interface{}) error {

	return sm.UpdateIntentState(ctx, namespacedName, func(state *IntentState) error {

		// Record phase transition.

		transition := PhaseTransition{

			FromPhase: state.CurrentPhase,

			ToPhase:   newPhase,

			Timestamp: time.Now(),

			Metadata:  metadata,

		}



		state.PhaseTransitions = append(state.PhaseTransitions, transition)

		state.CurrentPhase = newPhase

		state.PhaseStartTime = time.Now()



		// Update phase-specific data.

		if state.PhaseData == nil {

			state.PhaseData = make(map[interfaces.ProcessingPhase]interface{})

		}



		if metadata != nil {

			state.PhaseData[newPhase] = metadata

		}



		sm.logger.Info("Phase transition", "intent", namespacedName, "fromPhase", transition.FromPhase, "toPhase", transition.ToPhase)



		return nil

	})

}



// SetPhaseError sets an error for the current phase.

func (sm *StateManager) SetPhaseError(ctx context.Context, namespacedName types.NamespacedName, phase interfaces.ProcessingPhase, err error) error {

	return sm.UpdateIntentState(ctx, namespacedName, func(state *IntentState) error {

		if state.PhaseErrors == nil {

			state.PhaseErrors = make(map[interfaces.ProcessingPhase][]string)

		}



		state.PhaseErrors[phase] = append(state.PhaseErrors[phase], err.Error())

		state.LastError = err.Error()

		state.LastErrorTime = time.Now()



		return nil

	})

}



// GetPhaseData retrieves phase-specific data.

func (sm *StateManager) GetPhaseData(ctx context.Context, namespacedName types.NamespacedName, phase interfaces.ProcessingPhase) (interface{}, error) {

	state, err := sm.GetIntentState(ctx, namespacedName)

	if err != nil {

		return nil, err

	}



	if state.PhaseData == nil {

		return nil, nil

	}



	return state.PhaseData[phase], nil

}



// SetPhaseData sets phase-specific data.

func (sm *StateManager) SetPhaseData(ctx context.Context, namespacedName types.NamespacedName, phase interfaces.ProcessingPhase, data interface{}) error {

	return sm.UpdateIntentState(ctx, namespacedName, func(state *IntentState) error {

		if state.PhaseData == nil {

			state.PhaseData = make(map[interfaces.ProcessingPhase]interface{})

		}



		state.PhaseData[phase] = data

		return nil

	})

}



// AddDependency adds a dependency between intents.

func (sm *StateManager) AddDependency(ctx context.Context, intentName, dependsOnIntent types.NamespacedName) error {

	return sm.UpdateIntentState(ctx, intentName, func(state *IntentState) error {

		for _, dep := range state.Dependencies {

			if dep.Intent == dependsOnIntent.String() {

				return nil // Already exists

			}

		}



		dependency := IntentDependency{

			Intent:    dependsOnIntent.String(),

			Phase:     interfaces.PhaseCompleted, // Default: wait for completion

			Timestamp: time.Now(),

		}



		state.Dependencies = append(state.Dependencies, dependency)

		return nil

	})

}



// CheckDependencies checks if all dependencies are satisfied.

func (sm *StateManager) CheckDependencies(ctx context.Context, namespacedName types.NamespacedName) (bool, []string, error) {

	state, err := sm.GetIntentState(ctx, namespacedName)

	if err != nil {

		return false, nil, err

	}



	var unsatisfied []string



	for _, dep := range state.Dependencies {

		nn, err := sm.parseNamespacedName(dep.Intent)

		if err != nil {

			unsatisfied = append(unsatisfied, fmt.Sprintf("invalid dependency: %s", dep.Intent))

			continue

		}



		depState, err := sm.GetIntentState(ctx, nn)

		if err != nil {

			unsatisfied = append(unsatisfied, fmt.Sprintf("dependency not found: %s", dep.Intent))

			continue

		}



		// Check if dependency phase is satisfied.

		if !sm.isPhaseSatisfied(depState, dep.Phase) {

			unsatisfied = append(unsatisfied, fmt.Sprintf("dependency %s not at required phase %s", dep.Intent, dep.Phase))

		}

	}



	return len(unsatisfied) == 0, unsatisfied, nil

}



// ListStates lists all intent states matching the given selector.

func (sm *StateManager) ListStates(ctx context.Context, namespace string, labelSelector map[string]string) ([]*IntentState, error) {

	intentList := &nephoranv1.NetworkIntentList{}

	listOpts := []client.ListOption{

		client.InNamespace(namespace),

	}



	if len(labelSelector) > 0 {

		listOpts = append(listOpts, client.MatchingLabels(labelSelector))

	}



	if err := sm.client.List(ctx, intentList, listOpts...); err != nil {

		return nil, fmt.Errorf("failed to list network intents: %w", err)

	}



	states := make([]*IntentState, len(intentList.Items))

	for i, intent := range intentList.Items {

		states[i] = sm.convertToIntentState(&intent)

	}



	return states, nil

}



// GetStatistics returns state management statistics.

func (sm *StateManager) GetStatistics() *StateStatistics {

	return &StateStatistics{

		TotalStates:           sm.stateCache.Size(),

		CacheHitRate:          sm.metrics.GetCacheHitRate(),

		ActiveLocks:           sm.stateLocks.ActiveLocks(),

		AverageUpdateTime:     sm.metrics.GetAverageUpdateTime(),

		StateValidationErrors: sm.metrics.GetValidationErrors(),

		LastSyncTime:          sm.metrics.GetLastSyncTime(),

	}

}



// Internal methods.



func (sm *StateManager) convertToIntentState(intent *nephoranv1.NetworkIntent) *IntentState {

	state := &IntentState{

		NamespacedName: types.NamespacedName{

			Namespace: intent.Namespace,

			Name:      intent.Name,

		},

		CurrentPhase:     interfaces.ProcessingPhase(intent.Status.Phase),

		CreationTime:     intent.CreationTimestamp.Time,

		LastModified:     time.Now(),

		Version:          intent.ResourceVersion,

		Conditions:       make([]StateCondition, 0, 5),

		PhaseTransitions: make([]PhaseTransition, 0, 10),

		PhaseData:        make(map[interfaces.ProcessingPhase]interface{}),

		PhaseErrors:      make(map[interfaces.ProcessingPhase][]string),

		Dependencies:     make([]IntentDependency, 0, 3),

		Metadata:         make(map[string]interface{}),

	}



	// Initialize basic condition based on phase.

	if intent.Status.Phase != "" {

		state.Conditions = append(state.Conditions, StateCondition{

			Type:               "Ready",

			Status:             "True",

			LastTransitionTime: intent.Status.LastUpdateTime.Time,

			Reason:             "PhaseSet",

			Message:            intent.Status.LastMessage,

		})

	}



	// Extract metadata from annotations.

	if intent.Annotations != nil {

		for key, value := range intent.Annotations {

			if key == "nephoran.io/dependencies" {

				// Parse dependencies from annotation.

				// This is a simplified implementation.

				state.Metadata["dependencies"] = value

			}

		}

	}



	// Set phase start time based on last update time if available.

	if !intent.Status.LastUpdateTime.IsZero() {

		state.PhaseStartTime = intent.Status.LastUpdateTime.Time

	}



	return state

}



func (sm *StateManager) updateKubernetesState(ctx context.Context, namespacedName types.NamespacedName, state *IntentState) error {

	intent := &nephoranv1.NetworkIntent{}

	if err := sm.client.Get(ctx, namespacedName, intent); err != nil {

		return err

	}



	// Update intent status based on state.

	intent.Status.Phase = ProcessingPhaseToNetworkIntentPhase(state.CurrentPhase)



	// Update last message from conditions if available.

	if len(state.Conditions) > 0 {

		intent.Status.LastMessage = state.Conditions[0].Message

	}



	// Update processing times (use LastUpdateTime as available field).

	if !state.PhaseStartTime.IsZero() {

		intent.Status.LastUpdateTime = metav1.Time{Time: state.PhaseStartTime}

	}



	// Update last error (using LastUpdateTime as available field).

	if state.LastError != "" && !state.LastErrorTime.IsZero() {

		intent.Status.LastUpdateTime = metav1.Time{Time: state.LastErrorTime}

	}



	// Update status.

	return sm.client.Status().Update(ctx, intent)

}



func (sm *StateManager) validateState(state *IntentState) error {

	// Basic validation rules.

	if state.CurrentPhase == "" {

		return fmt.Errorf("current phase cannot be empty")

	}



	if state.Version == "" {

		return fmt.Errorf("version cannot be empty")

	}



	// Validate phase transitions.

	if len(state.PhaseTransitions) > 0 {

		lastTransition := state.PhaseTransitions[len(state.PhaseTransitions)-1]

		if lastTransition.ToPhase != state.CurrentPhase {

			return fmt.Errorf("current phase does not match last transition")

		}

	}



	// Validate dependencies.

	for _, dep := range state.Dependencies {

		if dep.Intent == "" {

			return fmt.Errorf("dependency intent name cannot be empty")

		}

	}



	return nil

}



func (sm *StateManager) publishStateChangeEvent(ctx context.Context, namespacedName types.NamespacedName, state *IntentState) {

	event := StateChangeEvent{

		Type:       "state.changed",

		IntentName: namespacedName,

		NewPhase:   state.CurrentPhase,

		Version:    state.Version,

		Timestamp:  time.Now(),

		Metadata:   state.Metadata,

	}



	if sm.eventBus != nil {

		sm.eventBus.PublishStateChange(ctx, event)

	}

}



func (sm *StateManager) isPhaseSatisfied(state *IntentState, requiredPhase interfaces.ProcessingPhase) bool {

	// Check if the current phase satisfies the requirement.

	phaseOrder := []interfaces.ProcessingPhase{

		interfaces.PhaseIntentReceived,

		interfaces.PhaseLLMProcessing,

		interfaces.PhaseResourcePlanning,

		interfaces.PhaseManifestGeneration,

		interfaces.PhaseGitOpsCommit,

		interfaces.PhaseDeploymentVerification,

		interfaces.PhaseCompleted,

	}



	currentIndex := -1

	requiredIndex := -1



	for i, phase := range phaseOrder {

		if phase == state.CurrentPhase {

			currentIndex = i

		}

		if phase == requiredPhase {

			requiredIndex = i

		}

	}



	return currentIndex >= requiredIndex

}



func (sm *StateManager) parseNamespacedName(nameStr string) (types.NamespacedName, error) {

	// Simple parser for "namespace/name" format.

	parts := make([]string, 2)

	if len(parts) != 2 {

		return types.NamespacedName{}, fmt.Errorf("invalid namespaced name format: %s", nameStr)

	}



	return types.NamespacedName{

		Namespace: parts[0],

		Name:      parts[1],

	}, nil

}



// Background processes.



func (sm *StateManager) runCacheCleanup(ctx context.Context) {

	ticker := time.NewTicker(sm.config.CacheCleanupInterval)

	defer ticker.Stop()



	for {

		select {

		case <-ticker.C:

			sm.stateCache.Cleanup()

			sm.metrics.RecordCacheCleanup()

		case <-ctx.Done():

			return

		}

	}

}



func (sm *StateManager) runStateSync(ctx context.Context) {

	ticker := time.NewTicker(sm.config.SyncInterval)

	defer ticker.Stop()



	for {

		select {

		case <-ticker.C:

			sm.performStateSync(ctx)

		case <-ctx.Done():

			return

		}

	}

}



func (sm *StateManager) performStateSync(ctx context.Context) {

	// This is a placeholder for state synchronization logic.

	// In a real implementation, this would sync cached states with Kubernetes.

	sm.metrics.RecordStateSync()

}



func (sm *StateManager) runMetricsCollection(ctx context.Context) {

	ticker := time.NewTicker(1 * time.Minute)

	defer ticker.Stop()



	for {

		select {

		case <-ticker.C:

			sm.collectMetrics()

		case <-ctx.Done():

			return

		}

	}

}



func (sm *StateManager) collectMetrics() {

	// Collect various metrics.

	sm.metrics.RecordCacheSize(int64(sm.stateCache.Size()))

	sm.metrics.RecordActiveLocks(sm.stateLocks.ActiveLocks())

}

