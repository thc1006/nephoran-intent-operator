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
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/nephio-project/nephoran-intent-operator/pkg/controllers/interfaces"

	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
)

// CoordinationManager orchestrates the specialized controllers.

type CoordinationManager struct {
	logger logr.Logger

	stateManager *StateManager

	eventBus EventBus

	// Controller coordination.

	controllers map[ComponentType]ControllerInterface

	dependencies map[interfaces.ProcessingPhase][]interfaces.ProcessingPhase

	transitions map[interfaces.ProcessingPhase][]interfaces.ProcessingPhase

	// Phase execution.

	phaseExecutors map[interfaces.ProcessingPhase]PhaseExecutor

	executionQueue *ExecutionQueue

	// Parallel processing.

	parallelGroups map[string][]interfaces.ProcessingPhase

	parallelLimits map[interfaces.ProcessingPhase]int

	// Timeout management.

	phaseTimeouts map[interfaces.ProcessingPhase]time.Duration

	activeTimeouts map[string]*time.Timer

	// Error handling and recovery.

	errorStrategies map[interfaces.ProcessingPhase]ErrorStrategy

	recoveryManager *RecoveryManager

	// Configuration.

	config *CoordinationConfig

	// Internal state.

	mutex sync.RWMutex

	started bool

	stopChan chan bool

	workerWG sync.WaitGroup
}

// CoordinationConfig provides configuration for coordination.

type CoordinationConfig struct {

	// Execution configuration.

	MaxConcurrentIntents int `json:"maxConcurrentIntents"`

	DefaultPhaseTimeout time.Duration `json:"defaultPhaseTimeout"`

	PhaseTransitionTimeout time.Duration `json:"phaseTransitionTimeout"`

	// Parallel processing.

	EnableParallelProcessing bool `json:"enableParallelProcessing"`

	ParallelLimits map[interfaces.ProcessingPhase]int `json:"parallelLimits"`

	ParallelGroups map[string][]interfaces.ProcessingPhase `json:"parallelGroups"`

	// Error handling.

	MaxRetryAttempts int `json:"maxRetryAttempts"`

	RetryBackoff time.Duration `json:"retryBackoff"`

	ErrorStrategies map[interfaces.ProcessingPhase]ErrorStrategy `json:"errorStrategies"`

	// Health monitoring.

	HealthCheckInterval time.Duration `json:"healthCheckInterval"`

	UnhealthyThreshold int `json:"unhealthyThreshold"`

	// Performance tuning.

	WorkerPoolSize int `json:"workerPoolSize"`

	QueueCapacity int `json:"queueCapacity"`

	BatchProcessing bool `json:"batchProcessing"`

	BatchSize int `json:"batchSize"`
}

// DefaultCoordinationConfig returns default configuration.

func DefaultCoordinationConfig() *CoordinationConfig {

	return &CoordinationConfig{

		MaxConcurrentIntents: 100,

		DefaultPhaseTimeout: 5 * time.Minute,

		PhaseTransitionTimeout: 30 * time.Second,

		EnableParallelProcessing: true,

		ParallelLimits: map[interfaces.ProcessingPhase]int{

			interfaces.PhaseLLMProcessing: 10,

			interfaces.PhaseResourcePlanning: 20,

			interfaces.PhaseManifestGeneration: 15,

			interfaces.PhaseGitOpsCommit: 5,

			interfaces.PhaseDeploymentVerification: 10,
		},

		ParallelGroups: map[string][]interfaces.ProcessingPhase{

			"planning": {

				interfaces.PhaseResourcePlanning,

				interfaces.PhaseManifestGeneration,
			},
		},

		MaxRetryAttempts: 3,

		RetryBackoff: 2 * time.Second,

		HealthCheckInterval: 30 * time.Second,

		UnhealthyThreshold: 3,

		WorkerPoolSize: 10,

		QueueCapacity: 1000,

		BatchProcessing: false,

		BatchSize: 10,
	}

}

// ControllerInterface defines the interface for specialized controllers.

type ControllerInterface interface {
	GetComponentType() ComponentType

	ProcessPhase(ctx context.Context, intent types.NamespacedName, phase interfaces.ProcessingPhase, data interface{}) error

	IsHealthy() bool

	GetMetrics() map[string]interface{}

	ValidatePhaseData(phase interfaces.ProcessingPhase, data interface{}) error
}

// PhaseExecutor executes a specific processing phase.

type PhaseExecutor interface {
	Execute(ctx context.Context, intent types.NamespacedName, phase interfaces.ProcessingPhase, data interface{}) error

	Validate(phase interfaces.ProcessingPhase, data interface{}) error

	GetEstimatedDuration(phase interfaces.ProcessingPhase) time.Duration

	CanExecuteParallel() bool
}

// ErrorStrategy defines how to handle errors in a phase.

type ErrorStrategy struct {
	Type string `json:"type"` // "retry", "skip", "fail", "rollback"

	MaxRetries int `json:"maxRetries"`

	RetryBackoff time.Duration `json:"retryBackoff"`

	FallbackPhase string `json:"fallbackPhase,omitempty"`

	NotifyOnError bool `json:"notifyOnError"`
}

// NewCoordinationManager creates a new coordination manager.

func NewCoordinationManager(stateManager *StateManager, eventBus EventBus, config *CoordinationConfig) *CoordinationManager {

	if config == nil {

		config = DefaultCoordinationConfig()

	}

	cm := &CoordinationManager{

		logger: ctrl.Log.WithName("coordination-manager"),

		stateManager: stateManager,

		eventBus: eventBus,

		controllers: make(map[ComponentType]ControllerInterface),

		dependencies: make(map[interfaces.ProcessingPhase][]interfaces.ProcessingPhase),

		transitions: make(map[interfaces.ProcessingPhase][]interfaces.ProcessingPhase),

		phaseExecutors: make(map[interfaces.ProcessingPhase]PhaseExecutor),

		parallelGroups: config.ParallelGroups,

		parallelLimits: config.ParallelLimits,

		phaseTimeouts: make(map[interfaces.ProcessingPhase]time.Duration),

		activeTimeouts: make(map[string]*time.Timer),

		errorStrategies: config.ErrorStrategies,

		config: config,

		stopChan: make(chan bool),
	}

	// Initialize components.

	cm.initializeDependencies()

	cm.initializeTransitions()

	cm.initializeTimeouts()

	cm.initializeErrorStrategies()

	// Create execution queue.

	cm.executionQueue = NewExecutionQueue(config.QueueCapacity)

	// Create recovery manager.

	cm.recoveryManager = NewRecoveryManager(cm.stateManager, cm.eventBus)

	return cm

}

// RegisterController registers a specialized controller.

func (cm *CoordinationManager) RegisterController(controller ControllerInterface) error {

	cm.mutex.Lock()

	defer cm.mutex.Unlock()

	componentType := controller.GetComponentType()

	cm.controllers[componentType] = controller

	cm.logger.Info("Registered controller", "componentType", componentType)

	return nil

}

// RegisterPhaseExecutor registers a phase executor.

func (cm *CoordinationManager) RegisterPhaseExecutor(phase interfaces.ProcessingPhase, executor PhaseExecutor) {

	cm.mutex.Lock()

	defer cm.mutex.Unlock()

	cm.phaseExecutors[phase] = executor

	cm.logger.Info("Registered phase executor", "phase", phase)

}

// Start starts the coordination manager.

func (cm *CoordinationManager) Start(ctx context.Context) error {

	cm.mutex.Lock()

	defer cm.mutex.Unlock()

	if cm.started {

		return fmt.Errorf("coordination manager already started")

	}

	// Subscribe to relevant events.

	if err := cm.subscribeToEvents(); err != nil {

		return fmt.Errorf("failed to subscribe to events: %w", err)

	}

	// Start worker pool.

	for i := range cm.config.WorkerPoolSize {

		cm.workerWG.Add(1)

		go cm.worker(ctx, i)

	}

	// Start health monitoring.

	cm.workerWG.Add(1)

	go cm.healthMonitor(ctx)

	// Start recovery manager.

	if err := cm.recoveryManager.Start(ctx); err != nil {

		return fmt.Errorf("failed to start recovery manager: %w", err)

	}

	cm.started = true

	cm.logger.Info("Coordination manager started", "workers", cm.config.WorkerPoolSize)

	return nil

}

// Stop stops the coordination manager.

func (cm *CoordinationManager) Stop(ctx context.Context) error {

	cm.mutex.Lock()

	defer cm.mutex.Unlock()

	if !cm.started {

		return nil

	}

	cm.logger.Info("Stopping coordination manager")

	// Signal stop.

	close(cm.stopChan)

	// Stop recovery manager.

	cm.recoveryManager.Stop(ctx)

	// Wait for workers to finish.

	done := make(chan bool)

	go func() {

		cm.workerWG.Wait()

		done <- true

	}()

	select {

	case <-done:

		cm.logger.Info("All coordination workers stopped")

	case <-time.After(30 * time.Second):

		cm.logger.Info("Coordination worker shutdown timeout")

	}

	cm.started = false

	return nil

}

// ProcessIntent processes a network intent through the coordination pipeline.

func (cm *CoordinationManager) ProcessIntent(ctx context.Context, intentName types.NamespacedName) error {

	// Get current intent state.

	state, err := cm.stateManager.GetIntentState(ctx, intentName)

	if err != nil {

		return fmt.Errorf("failed to get intent state: %w", err)

	}

	// Determine next phase.

	nextPhase, err := cm.determineNextPhase(state)

	if err != nil {

		return fmt.Errorf("failed to determine next phase: %w", err)

	}

	if nextPhase == "" {

		// Intent is complete or failed.

		return nil

	}

	// Create execution task.

	task := &ExecutionTask{

		IntentName: intentName,

		Phase: nextPhase,

		Priority: cm.calculatePriority(state),

		Timestamp: time.Now(),

		Context: make(map[string]interface{}),
	}

	// Add to execution queue.

	if err := cm.executionQueue.Enqueue(task); err != nil {

		return fmt.Errorf("failed to enqueue execution task: %w", err)

	}

	cm.logger.V(1).Info("Intent processing queued", "intent", intentName, "phase", nextPhase)

	return nil

}

// TransitionPhase transitions an intent to the next phase.

func (cm *CoordinationManager) TransitionPhase(ctx context.Context, intentName types.NamespacedName, fromPhase, toPhase interfaces.ProcessingPhase, data interface{}) error {

	// Validate transition.

	if !cm.isValidTransition(fromPhase, toPhase) {

		return fmt.Errorf("invalid phase transition from %s to %s", fromPhase, toPhase)

	}

	// Check dependencies.

	if err := cm.checkPhaseDependencies(ctx, intentName, toPhase); err != nil {

		return fmt.Errorf("phase dependencies not met: %w", err)

	}

	// Execute the transition.

	return cm.executePhaseTransition(ctx, intentName, fromPhase, toPhase, data)

}

// Internal methods.

func (cm *CoordinationManager) initializeDependencies() {

	// Standard pipeline dependencies.

	cm.dependencies[interfaces.PhaseLLMProcessing] = []interfaces.ProcessingPhase{}

	cm.dependencies[interfaces.PhaseResourcePlanning] = []interfaces.ProcessingPhase{interfaces.PhaseLLMProcessing}

	cm.dependencies[interfaces.PhaseManifestGeneration] = []interfaces.ProcessingPhase{interfaces.PhaseResourcePlanning}

	cm.dependencies[interfaces.PhaseGitOpsCommit] = []interfaces.ProcessingPhase{interfaces.PhaseManifestGeneration}

	cm.dependencies[interfaces.PhaseDeploymentVerification] = []interfaces.ProcessingPhase{interfaces.PhaseGitOpsCommit}

}

func (cm *CoordinationManager) initializeTransitions() {

	// Valid phase transitions.

	cm.transitions[interfaces.PhaseIntentReceived] = []interfaces.ProcessingPhase{interfaces.PhaseLLMProcessing}

	cm.transitions[interfaces.PhaseLLMProcessing] = []interfaces.ProcessingPhase{interfaces.PhaseResourcePlanning, interfaces.PhaseFailed}

	cm.transitions[interfaces.PhaseResourcePlanning] = []interfaces.ProcessingPhase{interfaces.PhaseManifestGeneration, interfaces.PhaseFailed}

	cm.transitions[interfaces.PhaseManifestGeneration] = []interfaces.ProcessingPhase{interfaces.PhaseGitOpsCommit, interfaces.PhaseFailed}

	cm.transitions[interfaces.PhaseGitOpsCommit] = []interfaces.ProcessingPhase{interfaces.PhaseDeploymentVerification, interfaces.PhaseFailed}

	cm.transitions[interfaces.PhaseDeploymentVerification] = []interfaces.ProcessingPhase{interfaces.PhaseCompleted, interfaces.PhaseFailed}

	cm.transitions[interfaces.PhaseCompleted] = []interfaces.ProcessingPhase{}

	cm.transitions[interfaces.PhaseFailed] = []interfaces.ProcessingPhase{}

}

func (cm *CoordinationManager) initializeTimeouts() {

	// Default timeouts for each phase.

	cm.phaseTimeouts[interfaces.PhaseLLMProcessing] = 2 * time.Minute

	cm.phaseTimeouts[interfaces.PhaseResourcePlanning] = 1 * time.Minute

	cm.phaseTimeouts[interfaces.PhaseManifestGeneration] = 30 * time.Second

	cm.phaseTimeouts[interfaces.PhaseGitOpsCommit] = 1 * time.Minute

	cm.phaseTimeouts[interfaces.PhaseDeploymentVerification] = 5 * time.Minute

}

func (cm *CoordinationManager) initializeErrorStrategies() {

	if cm.errorStrategies == nil {

		cm.errorStrategies = make(map[interfaces.ProcessingPhase]ErrorStrategy)

	}

	// Default error strategies.

	defaultStrategy := ErrorStrategy{

		Type: "retry",

		MaxRetries: 3,

		RetryBackoff: 2 * time.Second,

		NotifyOnError: true,
	}

	phases := []interfaces.ProcessingPhase{

		interfaces.PhaseLLMProcessing,

		interfaces.PhaseResourcePlanning,

		interfaces.PhaseManifestGeneration,

		interfaces.PhaseGitOpsCommit,

		interfaces.PhaseDeploymentVerification,
	}

	for _, phase := range phases {

		if _, exists := cm.errorStrategies[phase]; !exists {

			cm.errorStrategies[phase] = defaultStrategy

		}

	}

}

func (cm *CoordinationManager) subscribeToEvents() error {

	// Subscribe to state change events.

	if err := cm.eventBus.Subscribe("state.changed", cm.handleStateChangeEvent); err != nil {

		return err

	}

	// Subscribe to phase completion events.

	phaseEvents := []string{

		"llm.processing.completed",

		"resource.planning.completed",

		"manifest.generation.completed",

		"gitops.commit.completed",

		"deployment.verification.completed",
	}

	for _, eventType := range phaseEvents {

		if err := cm.eventBus.Subscribe(eventType, cm.handlePhaseCompletionEvent); err != nil {

			return err

		}

	}

	// Subscribe to error events.

	errorEvents := []string{

		"llm.processing.failed",

		"resource.planning.failed",

		"manifest.generation.failed",

		"gitops.commit.failed",

		"deployment.verification.failed",
	}

	for _, eventType := range errorEvents {

		if err := cm.eventBus.Subscribe(eventType, cm.handlePhaseErrorEvent); err != nil {

			return err

		}

	}

	return nil

}

func (cm *CoordinationManager) handleStateChangeEvent(ctx context.Context, event ProcessingEvent) error {

	// Parse intent name.

	intentName, err := cm.parseIntentName(event.IntentID)

	if err != nil {

		return err

	}

	// Queue for processing if needed.

	return cm.ProcessIntent(ctx, intentName)

}

func (cm *CoordinationManager) handlePhaseCompletionEvent(ctx context.Context, event ProcessingEvent) error {

	intentName, err := cm.parseIntentName(event.IntentID)

	if err != nil {

		return err

	}

	// Cancel timeout if active.

	cm.cancelPhaseTimeout(event.IntentID)

	// Process next phase.

	return cm.ProcessIntent(ctx, intentName)

}

func (cm *CoordinationManager) handlePhaseErrorEvent(ctx context.Context, event ProcessingEvent) error {

	intentName, err := cm.parseIntentName(event.IntentID)

	if err != nil {

		return err

	}

	// Cancel timeout if active.

	cm.cancelPhaseTimeout(event.IntentID)

	// Handle error based on strategy.

	phase := interfaces.ProcessingPhase(event.Phase)

	strategy := cm.errorStrategies[phase]

	return cm.handlePhaseError(ctx, intentName, phase, fmt.Errorf("phase failed"), strategy)

}

func (cm *CoordinationManager) worker(ctx context.Context, workerID int) {

	defer cm.workerWG.Done()

	cm.logger.V(1).Info("Coordination worker started", "workerID", workerID)

	for {

		select {

		case task := <-cm.executionQueue.channel:

			cm.executeTask(ctx, task)

		case <-cm.stopChan:

			cm.logger.V(1).Info("Coordination worker stopping", "workerID", workerID)

			return

		case <-ctx.Done():

			cm.logger.V(1).Info("Coordination worker cancelled", "workerID", workerID)

			return

		}

	}

}

func (cm *CoordinationManager) executeTask(ctx context.Context, task *ExecutionTask) {

	taskCtx, cancel := context.WithTimeout(ctx, cm.getPhaseTimeout(task.Phase))

	defer cancel()

	cm.logger.Info("Executing task", "intent", task.IntentName, "phase", task.Phase)

	// Set up phase timeout.

	cm.setupPhaseTimeout(task.IntentName.String(), task.Phase, taskCtx)

	// Execute the phase.

	err := cm.executePhase(taskCtx, task.IntentName, task.Phase)

	// Cancel timeout.

	cm.cancelPhaseTimeout(task.IntentName.String())

	// Handle result.

	if err != nil {

		cm.logger.Error(err, "Task execution failed", "intent", task.IntentName, "phase", task.Phase)

		strategy := cm.errorStrategies[task.Phase]

		cm.handlePhaseError(taskCtx, task.IntentName, task.Phase, err, strategy)

	} else {

		cm.logger.Info("Task executed successfully", "intent", task.IntentName, "phase", task.Phase)

	}

}

func (cm *CoordinationManager) executePhase(ctx context.Context, intentName types.NamespacedName, phase interfaces.ProcessingPhase) error {

	// Get phase executor.

	executor, exists := cm.phaseExecutors[phase]

	if !exists {

		return fmt.Errorf("no executor registered for phase %s", phase)

	}

	// Get phase data from state.

	data, err := cm.stateManager.GetPhaseData(ctx, intentName, phase)

	if err != nil {

		return fmt.Errorf("failed to get phase data: %w", err)

	}

	// Validate phase data.

	if err := executor.Validate(phase, data); err != nil {

		return fmt.Errorf("phase data validation failed: %w", err)

	}

	// Execute phase.

	if err := executor.Execute(ctx, intentName, phase, data); err != nil {

		return fmt.Errorf("phase execution failed: %w", err)

	}

	// Update state to next phase.

	nextPhase := cm.getNextPhase(phase)

	if nextPhase != "" {

		metadata := map[string]interface{}{

			"completedPhase": phase,

			"duration": time.Since(time.Now()), // This would be properly calculated

		}

		return cm.stateManager.TransitionPhase(ctx, intentName, nextPhase, metadata)

	}

	return nil

}

func (cm *CoordinationManager) determineNextPhase(state *IntentState) (interfaces.ProcessingPhase, error) {

	currentPhase := state.CurrentPhase

	// Check if in terminal state.

	if currentPhase == interfaces.PhaseCompleted || currentPhase == interfaces.PhaseFailed {

		return "", nil

	}

	// Get possible transitions.

	transitions, exists := cm.transitions[currentPhase]

	if !exists || len(transitions) == 0 {

		return interfaces.PhaseFailed, nil

	}

	// Return the primary transition (first one).

	return transitions[0], nil

}

func (cm *CoordinationManager) isValidTransition(from, to interfaces.ProcessingPhase) bool {

	transitions, exists := cm.transitions[from]

	if !exists {

		return false

	}

	for _, validTransition := range transitions {

		if validTransition == to {

			return true

		}

	}

	return false

}

func (cm *CoordinationManager) checkPhaseDependencies(ctx context.Context, intentName types.NamespacedName, phase interfaces.ProcessingPhase) error {

	dependencies := cm.dependencies[phase]

	if len(dependencies) == 0 {

		return nil // No dependencies

	}

	state, err := cm.stateManager.GetIntentState(ctx, intentName)

	if err != nil {

		return err

	}

	// Check each dependency.

	for _, depPhase := range dependencies {

		if !cm.isPhaseSatisfied(state, depPhase) {

			return fmt.Errorf("dependency phase %s not satisfied", depPhase)

		}

	}

	return nil

}

func (cm *CoordinationManager) isPhaseSatisfied(state *IntentState, phase interfaces.ProcessingPhase) bool {

	// Check if phase is completed in conditions.

	for _, condition := range state.Conditions {

		if condition.Type == string(phase) && condition.Status == "True" {

			return true

		}

	}

	return false

}

func (cm *CoordinationManager) executePhaseTransition(ctx context.Context, intentName types.NamespacedName, fromPhase, toPhase interfaces.ProcessingPhase, data interface{}) error {

	// Update state.

	metadata := map[string]interface{}{

		"fromPhase": fromPhase,

		"toPhase": toPhase,

		"data": data,
	}

	return cm.stateManager.TransitionPhase(ctx, intentName, toPhase, metadata)

}

func (cm *CoordinationManager) calculatePriority(state *IntentState) int {

	// Simple priority calculation based on age and retry count.

	age := time.Since(state.CreationTime)

	priority := int(age.Minutes()) + state.RetryCount*10

	return priority

}

func (cm *CoordinationManager) getPhaseTimeout(phase interfaces.ProcessingPhase) time.Duration {

	if timeout, exists := cm.phaseTimeouts[phase]; exists {

		return timeout

	}

	return cm.config.DefaultPhaseTimeout

}

func (cm *CoordinationManager) setupPhaseTimeout(intentID string, phase interfaces.ProcessingPhase, ctx context.Context) {

	timeout := cm.getPhaseTimeout(phase)

	timer := time.AfterFunc(timeout, func() {

		cm.handlePhaseTimeout(ctx, intentID, phase)

	})

	cm.mutex.Lock()

	cm.activeTimeouts[intentID] = timer

	cm.mutex.Unlock()

}

func (cm *CoordinationManager) cancelPhaseTimeout(intentID string) {

	cm.mutex.Lock()

	defer cm.mutex.Unlock()

	if timer, exists := cm.activeTimeouts[intentID]; exists {

		timer.Stop()

		delete(cm.activeTimeouts, intentID)

	}

}

func (cm *CoordinationManager) handlePhaseTimeout(ctx context.Context, intentID string, phase interfaces.ProcessingPhase) {

	cm.logger.Error(fmt.Errorf("phase timeout"), "Phase timed out", "intentID", intentID, "phase", phase)

	// Parse intent name and handle as error.

	intentName, err := cm.parseIntentName(intentID)

	if err != nil {

		cm.logger.Error(err, "Failed to parse intent name", "intentID", intentID)

		return

	}

	strategy := cm.errorStrategies[phase]

	cm.handlePhaseError(ctx, intentName, phase, fmt.Errorf("phase timeout"), strategy)

}

func (cm *CoordinationManager) handlePhaseError(ctx context.Context, intentName types.NamespacedName, phase interfaces.ProcessingPhase, err error, strategy ErrorStrategy) error {

	cm.logger.Error(err, "Handling phase error", "intent", intentName, "phase", phase, "strategy", strategy.Type)

	switch strategy.Type {

	case "retry":

		return cm.retryPhase(ctx, intentName, phase, strategy)

	case "skip":

		return cm.skipPhase(ctx, intentName, phase)

	case "fail":

		return cm.failIntent(ctx, intentName, err)

	case "rollback":

		return cm.rollbackIntent(ctx, intentName, phase)

	default:

		return cm.failIntent(ctx, intentName, err)

	}

}

func (cm *CoordinationManager) retryPhase(ctx context.Context, intentName types.NamespacedName, phase interfaces.ProcessingPhase, strategy ErrorStrategy) error {

	// Get current state to check retry count.

	state, err := cm.stateManager.GetIntentState(ctx, intentName)

	if err != nil {

		return err

	}

	if state.RetryCount >= strategy.MaxRetries {

		return cm.failIntent(ctx, intentName, fmt.Errorf("max retries exceeded"))

	}

	// Increment retry count and schedule retry.

	return cm.stateManager.UpdateIntentState(ctx, intentName, func(state *IntentState) error {

		state.RetryCount++

		return nil

	})

}

func (cm *CoordinationManager) skipPhase(ctx context.Context, intentName types.NamespacedName, phase interfaces.ProcessingPhase) error {

	// Move to next phase.

	nextPhase := cm.getNextPhase(phase)

	if nextPhase == "" {

		return cm.failIntent(ctx, intentName, fmt.Errorf("no next phase after skip"))

	}

	metadata := map[string]interface{}{

		"skippedPhase": phase,

		"reason": "error_strategy_skip",
	}

	return cm.stateManager.TransitionPhase(ctx, intentName, nextPhase, metadata)

}

func (cm *CoordinationManager) failIntent(ctx context.Context, intentName types.NamespacedName, err error) error {

	metadata := map[string]interface{}{

		"error": err.Error(),
	}

	return cm.stateManager.TransitionPhase(ctx, intentName, interfaces.PhaseFailed, metadata)

}

func (cm *CoordinationManager) rollbackIntent(ctx context.Context, intentName types.NamespacedName, phase interfaces.ProcessingPhase) error {

	// Implement rollback logic based on current phase.

	// This is a simplified implementation.

	metadata := map[string]interface{}{

		"rollbackFromPhase": phase,

		"reason": "error_strategy_rollback",
	}

	return cm.stateManager.TransitionPhase(ctx, intentName, interfaces.PhaseFailed, metadata)

}

func (cm *CoordinationManager) getNextPhase(currentPhase interfaces.ProcessingPhase) interfaces.ProcessingPhase {

	transitions := cm.transitions[currentPhase]

	if len(transitions) > 0 {

		return transitions[0] // Return primary transition

	}

	return ""

}

func (cm *CoordinationManager) parseIntentName(intentID string) (types.NamespacedName, error) {

	// Simple parser - in practice, this would be more robust.

	return types.NamespacedName{

		Namespace: "default",

		Name: intentID,
	}, nil

}

func (cm *CoordinationManager) healthMonitor(ctx context.Context) {

	defer cm.workerWG.Done()

	ticker := time.NewTicker(cm.config.HealthCheckInterval)

	defer ticker.Stop()

	for {

		select {

		case <-ticker.C:

			cm.performHealthCheck()

		case <-cm.stopChan:

			return

		case <-ctx.Done():

			return

		}

	}

}

func (cm *CoordinationManager) performHealthCheck() {

	cm.mutex.RLock()

	controllers := make([]ControllerInterface, 0, len(cm.controllers))

	for _, controller := range cm.controllers {

		controllers = append(controllers, controller)

	}

	cm.mutex.RUnlock()

	unhealthyCount := 0

	for _, controller := range controllers {

		if !controller.IsHealthy() {

			unhealthyCount++

			cm.logger.Error(fmt.Errorf("controller unhealthy"), "Controller health check failed",

				"componentType", controller.GetComponentType())

		}

	}

	if unhealthyCount > cm.config.UnhealthyThreshold {

		cm.logger.Error(fmt.Errorf("system unhealthy"), "Too many unhealthy controllers",

			"unhealthyCount", unhealthyCount, "threshold", cm.config.UnhealthyThreshold)

	}

}
