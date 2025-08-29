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
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/interfaces"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// IntentOrchestrator manages the overall processing pipeline for NetworkIntents.
type IntentOrchestrator struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	// State management.
	StateMachine       *StateMachine
	ProcessingContexts sync.Map // map[string]*interfaces.ProcessingContext

	// Phase controllers.
	Controllers map[interfaces.ProcessingPhase]interfaces.PhaseController

	// Coordination.
	EventBus         *EventBus
	WorkQueueManager *WorkQueueManager
	LockManager      *IntentLockManager

	// Configuration.
	Config *OrchestratorConfig

	// Metrics and observability.
	MetricsCollector *MetricsCollector
	Logger           logr.Logger
}

// OrchestratorConfig holds configuration for the orchestrator.
type OrchestratorConfig struct {
	MaxConcurrentIntents     int           `json:"maxConcurrentIntents"`
	MaxConcurrentPhases      int           `json:"maxConcurrentPhases"`
	PhaseTimeout             time.Duration `json:"phaseTimeout"`
	RetryBackoffBase         time.Duration `json:"retryBackoffBase"`
	RetryBackoffMax          time.Duration `json:"retryBackoffMax"`
	MaxRetries               int           `json:"maxRetries"`
	EnableParallelProcessing bool          `json:"enableParallelProcessing"`

	// Per-phase configurations.
	PhaseConfigs map[interfaces.ProcessingPhase]*PhaseConfig `json:"phaseConfigs"`
}

// PhaseConfig holds configuration for individual phases.
type PhaseConfig struct {
	Timeout        time.Duration `json:"timeout"`
	MaxRetries     int           `json:"maxRetries"`
	MaxConcurrency int           `json:"maxConcurrency"`
	Priority       int           `json:"priority"`

	// Dependencies and constraints.
	RequiredDependencies []interfaces.ProcessingPhase `json:"requiredDependencies"`
	OptionalDependencies []interfaces.ProcessingPhase `json:"optionalDependencies"`
	BlockedBy            []interfaces.ProcessingPhase `json:"blockedBy"`

	// Resource requirements.
	ResourceLimits map[string]interface{} `json:"resourceLimits"`
}

// NewIntentOrchestrator creates a new IntentOrchestrator.
func NewIntentOrchestrator(client client.Client, scheme *runtime.Scheme, recorder record.EventRecorder, config *OrchestratorConfig) (*IntentOrchestrator, error) {
	logger := log.Log.WithName("intent-orchestrator")

	orchestrator := &IntentOrchestrator{
		Client:           client,
		Scheme:           scheme,
		Recorder:         recorder,
		Config:           config,
		Logger:           logger,
		Controllers:      make(map[interfaces.ProcessingPhase]interfaces.PhaseController),
		MetricsCollector: NewMetricsCollector(),
	}

	// Initialize components.
	orchestrator.StateMachine = NewStateMachine(config)
	orchestrator.EventBus = NewEventBus(client, logger)
	orchestrator.WorkQueueManager = NewWorkQueueManager(config, logger)
	orchestrator.LockManager = NewIntentLockManager(client, logger)

	return orchestrator, nil
}

// RegisterController registers a phase controller with the orchestrator.
func (o *IntentOrchestrator) RegisterController(phase interfaces.ProcessingPhase, controller interfaces.PhaseController) error {
	if controller == nil {
		return fmt.Errorf("controller cannot be nil for phase %s", phase)
	}

	o.Controllers[phase] = controller
	o.Logger.Info("Registered phase controller", "phase", phase)

	// Subscribe to phase completion events.
	return o.EventBus.Subscribe(string(phase), o.handlePhaseEvent)
}

// Reconcile handles NetworkIntent resources.
func (o *IntentOrchestrator) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := o.Logger.WithValues("networkintent", req.NamespacedName)

	// Fetch the NetworkIntent instance.
	intent := &nephoranv1.NetworkIntent{}
	if err := o.Get(ctx, req.NamespacedName, intent); err != nil {
		if apierrors.IsNotFound(err) {
			// Object not found, handle cleanup.
			return o.handleIntentDeletion(ctx, req.NamespacedName)
		}
		log.Error(err, "Failed to fetch NetworkIntent")
		return ctrl.Result{}, err
	}

	// Handle deletion.
	if intent.DeletionTimestamp != nil {
		return o.handleIntentDeletion(ctx, req.NamespacedName)
	}

	// Add finalizer if not present.
	if !controllerutil.ContainsFinalizer(intent, "networkintent.nephoran.com/orchestrator") {
		controllerutil.AddFinalizer(intent, "networkintent.nephoran.com/orchestrator")
		return ctrl.Result{}, o.Update(ctx, intent)
	}

	// Start processing.
	return o.processIntent(ctx, intent)
}

// processIntent orchestrates the processing of a NetworkIntent.
func (o *IntentOrchestrator) processIntent(ctx context.Context, intent *nephoranv1.NetworkIntent) (ctrl.Result, error) {
	log := o.Logger.WithValues("intent", intent.Name, "namespace", intent.Namespace)

	// Get or create processing context.
	processingCtx, err := o.getOrCreateProcessingContext(ctx, intent)
	if err != nil {
		log.Error(err, "Failed to get processing context")
		return ctrl.Result{}, err
	}

	// Acquire intent lock.
	lock, err := o.LockManager.AcquireIntentLock(ctx, processingCtx.IntentID)
	if err != nil {
		log.Error(err, "Failed to acquire intent lock")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}
	defer lock.Release()

	// Determine current phase and next actions.
	currentPhase := o.StateMachine.GetCurrentPhase(intent)
	log.Info("Processing intent", "currentPhase", currentPhase)

	// Record phase start metrics.
	o.MetricsCollector.RecordPhaseStart(currentPhase, processingCtx.IntentID)

	// Execute phase processing.
	result, err := o.executePhase(ctx, intent, currentPhase, processingCtx)
	if err != nil {
		log.Error(err, "Phase execution failed", "phase", currentPhase)
		o.MetricsCollector.RecordPhaseCompletion(currentPhase, processingCtx.IntentID, false)
		return o.handlePhaseError(ctx, intent, currentPhase, err, processingCtx)
	}

	// Record phase completion.
	o.MetricsCollector.RecordPhaseCompletion(currentPhase, processingCtx.IntentID, result.Success)

	// Update processing context with results.
	o.updateProcessingContext(processingCtx, currentPhase, result)

	// Determine next phase and schedule if needed.
	return o.handlePhaseResult(ctx, intent, currentPhase, result, processingCtx)
}

// executePhase executes a specific processing phase.
func (o *IntentOrchestrator) executePhase(ctx context.Context, intent *nephoranv1.NetworkIntent, phase interfaces.ProcessingPhase, processingCtx *interfaces.ProcessingContext) (interfaces.ProcessingResult, error) {
	log := o.Logger.WithValues("intent", intent.Name, "phase", phase)

	// Get the controller for this phase.
	controller, exists := o.Controllers[phase]
	if !exists {
		return interfaces.ProcessingResult{}, fmt.Errorf("no controller found for phase %s", phase)
	}

	// Check dependencies.
	if err := o.checkPhaseDependencies(ctx, phase, processingCtx); err != nil {
		return interfaces.ProcessingResult{}, fmt.Errorf("phase dependencies not met: %w", err)
	}

	// Create phase-specific context with timeout.
	phaseConfig := o.Config.PhaseConfigs[phase]
	if phaseConfig == nil {
		phaseConfig = &PhaseConfig{
			Timeout:    o.Config.PhaseTimeout,
			MaxRetries: o.Config.MaxRetries,
		}
	}

	phaseCtx, cancel := context.WithTimeout(ctx, phaseConfig.Timeout)
	defer cancel()

	// Execute the phase.
	log.Info("Executing phase", "timeout", phaseConfig.Timeout)
	result, err := controller.ProcessPhase(phaseCtx, intent, phase)
	if err != nil {
		return result, fmt.Errorf("phase %s failed: %w", phase, err)
	}

	// Validate result.
	if err := o.validatePhaseResult(phase, result); err != nil {
		return result, fmt.Errorf("phase %s result validation failed: %w", phase, err)
	}

	log.Info("Phase completed successfully", "success", result.Success, "nextPhase", result.NextPhase)
	return result, nil
}

// handlePhaseResult handles the result of a phase execution.
func (o *IntentOrchestrator) handlePhaseResult(ctx context.Context, intent *nephoranv1.NetworkIntent, currentPhase interfaces.ProcessingPhase, result interfaces.ProcessingResult, processingCtx *interfaces.ProcessingContext) (ctrl.Result, error) {
	log := o.Logger.WithValues("intent", intent.Name, "phase", currentPhase, "success", result.Success)

	// Update intent status.
	if err := o.updateIntentStatus(ctx, intent, currentPhase, result); err != nil {
		log.Error(err, "Failed to update intent status")
		return ctrl.Result{}, err
	}

	// Publish phase completion event.
	event := ProcessingEvent{
		Type:          fmt.Sprintf("phase.%s.completed", currentPhase),
		IntentID:      processingCtx.IntentID,
		Phase:         currentPhase,
		Success:       result.Success,
		Data:          result.Data,
		Timestamp:     time.Now(),
		CorrelationID: processingCtx.CorrelationID,
	}

	if err := o.EventBus.Publish(ctx, event); err != nil {
		log.Error(err, "Failed to publish phase completion event")
	}

	// Handle success.
	if result.Success {
		return o.handlePhaseSuccess(ctx, intent, currentPhase, result, processingCtx)
	}

	// Handle failure with retry logic.
	return o.handlePhaseFailure(ctx, intent, currentPhase, result, processingCtx)
}

// handlePhaseSuccess handles successful phase completion.
func (o *IntentOrchestrator) handlePhaseSuccess(ctx context.Context, intent *nephoranv1.NetworkIntent, currentPhase interfaces.ProcessingPhase, result interfaces.ProcessingResult, processingCtx *interfaces.ProcessingContext) (ctrl.Result, error) {
	log := o.Logger.WithValues("intent", intent.Name, "phase", currentPhase)

	// Record success event.
	o.Recorder.Event(intent, "Normal", "PhaseCompleted", fmt.Sprintf("Phase %s completed successfully", currentPhase))

	// Determine next phase.
	nextPhase := result.NextPhase
	if nextPhase == "" {
		nextPhase = o.StateMachine.GetNextPhase(currentPhase)
	}

	// Check if processing is complete.
	if nextPhase == interfaces.PhaseCompleted {
		log.Info("Intent processing completed successfully")
		return o.handleIntentCompletion(ctx, intent, processingCtx)
	}

	// Schedule next phase.
	if o.Config.EnableParallelProcessing {
		return o.scheduleParallelProcessing(ctx, intent, nextPhase, processingCtx)
	}

	return o.scheduleNextPhase(ctx, intent, nextPhase, processingCtx)
}

// handlePhaseFailure handles phase failure with retry logic.
func (o *IntentOrchestrator) handlePhaseFailure(ctx context.Context, intent *nephoranv1.NetworkIntent, currentPhase interfaces.ProcessingPhase, result interfaces.ProcessingResult, processingCtx *interfaces.ProcessingContext) (ctrl.Result, error) {
	log := o.Logger.WithValues("intent", intent.Name, "phase", currentPhase)

	// Get phase status.
	phaseStatus := processingCtx.PhaseMetrics[currentPhase]
	retryCount := int(phaseStatus.ErrorCount)

	phaseConfig := o.Config.PhaseConfigs[currentPhase]
	maxRetries := o.Config.MaxRetries
	if phaseConfig != nil && phaseConfig.MaxRetries > 0 {
		maxRetries = phaseConfig.MaxRetries
	}

	// Check if we should retry.
	if retryCount < maxRetries {
		backoffDuration := o.calculateBackoff(retryCount)
		if result.RetryAfter != nil {
			backoffDuration = *result.RetryAfter
		}

		log.Info("Retrying phase", "retryCount", retryCount+1, "backoff", backoffDuration)
		o.Recorder.Event(intent, "Warning", "PhaseRetry", fmt.Sprintf("Retrying phase %s (attempt %d/%d)", currentPhase, retryCount+1, maxRetries))

		return ctrl.Result{RequeueAfter: backoffDuration}, nil
	}

	// Max retries exceeded, fail the intent.
	log.Error(fmt.Errorf("max retries exceeded"), "Phase failed permanently", "phase", currentPhase, "retries", retryCount)
	return o.handleIntentFailure(ctx, intent, currentPhase, fmt.Errorf("phase %s failed after %d retries: %s", currentPhase, retryCount, result.ErrorMessage), processingCtx)
}

// scheduleParallelProcessing schedules phases that can run in parallel.
func (o *IntentOrchestrator) scheduleParallelProcessing(ctx context.Context, intent *nephoranv1.NetworkIntent, nextPhase interfaces.ProcessingPhase, processingCtx *interfaces.ProcessingContext) (ctrl.Result, error) {
	log := o.Logger.WithValues("intent", intent.Name, "nextPhase", nextPhase)

	// Get phases that can run in parallel with the next phase.
	parallelPhases := o.StateMachine.GetParallelPhases(nextPhase)

	// Schedule all parallel phases.
	for _, phase := range parallelPhases {
		job := ProcessingJob{
			ID:         fmt.Sprintf("%s-%s-%d", processingCtx.IntentID, phase, time.Now().Unix()),
			IntentID:   processingCtx.IntentID,
			Phase:      phase,
			Priority:   o.getPhasePriority(phase),
			Data:       map[string]interface{}{"parallelExecution": true},
			Context:    processingCtx,
			MaxRetries: o.Config.MaxRetries,
			Timeout:    o.Config.PhaseTimeout,
		}

		if err := o.WorkQueueManager.EnqueueJob(ctx, phase, job); err != nil {
			log.Error(err, "Failed to enqueue parallel phase", "phase", phase)
		}
	}

	log.Info("Scheduled parallel phases", "phases", parallelPhases)
	return ctrl.Result{}, nil
}

// scheduleNextPhase schedules the next phase for sequential processing.
func (o *IntentOrchestrator) scheduleNextPhase(ctx context.Context, intent *nephoranv1.NetworkIntent, nextPhase interfaces.ProcessingPhase, processingCtx *interfaces.ProcessingContext) (ctrl.Result, error) {
	log := o.Logger.WithValues("intent", intent.Name, "nextPhase", nextPhase)

	// Update current phase.
	processingCtx.CurrentPhase = nextPhase

	// Immediate requeue for next phase.
	log.Info("Scheduling next phase")
	return ctrl.Result{Requeue: true}, nil
}

// Helper methods.

func (o *IntentOrchestrator) getOrCreateProcessingContext(ctx context.Context, intent *nephoranv1.NetworkIntent) (*interfaces.ProcessingContext, error) {
	intentID := string(intent.UID)

	if ctx, exists := o.ProcessingContexts.Load(intentID); exists {
		return ctx.(*interfaces.ProcessingContext), nil
	}

	processingCtx := &interfaces.ProcessingContext{
		IntentID:       intentID,
		CorrelationID:  fmt.Sprintf("intent-%s-%d", intent.Name, time.Now().Unix()),
		StartTime:      time.Now(),
		CurrentPhase:   interfaces.PhaseIntentReceived,
		IntentType:     string(intent.Spec.IntentType),
		OriginalIntent: intent.Spec.Intent,
		PhaseMetrics:   make(map[interfaces.ProcessingPhase]interfaces.PhaseMetrics),
		TotalMetrics:   make(map[string]float64),
	}

	o.ProcessingContexts.Store(intentID, processingCtx)
	return processingCtx, nil
}

func (o *IntentOrchestrator) updateProcessingContext(processingCtx *interfaces.ProcessingContext, phase interfaces.ProcessingPhase, result interfaces.ProcessingResult) {
	// Update phase metrics.
	phaseMetrics := processingCtx.PhaseMetrics[phase]
	if result.Success {
		phaseMetrics.Duration = time.Since(processingCtx.StartTime)
	} else {
		phaseMetrics.ErrorCount++
	}

	// Update phase-specific data.
	switch phase {
	case interfaces.PhaseLLMProcessing:
		processingCtx.LLMResponse = result.Data
	case interfaces.PhaseResourcePlanning:
		processingCtx.ResourcePlan = result.Data
	case interfaces.PhaseManifestGeneration:
		if manifests, ok := result.Data["manifests"].(map[string]string); ok {
			processingCtx.GeneratedManifests = manifests
		}
	case interfaces.PhaseGitOpsCommit:
		if commitHash, ok := result.Data["commitHash"].(string); ok {
			processingCtx.GitCommitHash = commitHash
		}
	case interfaces.PhaseDeploymentVerification:
		processingCtx.DeploymentStatus = result.Data
	}

	processingCtx.PhaseMetrics[phase] = phaseMetrics
}

func (o *IntentOrchestrator) checkPhaseDependencies(ctx context.Context, phase interfaces.ProcessingPhase, processingCtx *interfaces.ProcessingContext) error {
	phaseConfig := o.Config.PhaseConfigs[phase]
	if phaseConfig == nil {
		return nil
	}

	// Check required dependencies.
	for _, depPhase := range phaseConfig.RequiredDependencies {
		if metrics, exists := processingCtx.PhaseMetrics[depPhase]; !exists || metrics.Duration == 0 {
			return fmt.Errorf("required dependency %s not completed", depPhase)
		}
	}

	// Check blocking conditions.
	for _, blockingPhase := range phaseConfig.BlockedBy {
		if metrics, exists := processingCtx.PhaseMetrics[blockingPhase]; exists && metrics.ErrorCount > 0 {
			return fmt.Errorf("blocked by failed phase %s", blockingPhase)
		}
	}

	return nil
}

func (o *IntentOrchestrator) validatePhaseResult(phase interfaces.ProcessingPhase, result interfaces.ProcessingResult) error {
	// Basic validation.
	if !result.Success && result.ErrorMessage == "" {
		return fmt.Errorf("failed result must include error message")
	}

	// Phase-specific validation.
	switch phase {
	case interfaces.PhaseLLMProcessing:
		if result.Success && result.Data == nil {
			return fmt.Errorf("LLM processing must return data on success")
		}
	case interfaces.PhaseManifestGeneration:
		if result.Success {
			if manifests, ok := result.Data["manifests"]; !ok || manifests == nil {
				return fmt.Errorf("manifest generation must return manifests on success")
			}
		}
	case interfaces.PhaseGitOpsCommit:
		if result.Success {
			if commitHash, ok := result.Data["commitHash"]; !ok || commitHash == "" {
				return fmt.Errorf("GitOps commit must return commit hash on success")
			}
		}
	}

	return nil
}

func (o *IntentOrchestrator) updateIntentStatus(ctx context.Context, intent *nephoranv1.NetworkIntent, phase interfaces.ProcessingPhase, result interfaces.ProcessingResult) error {
	// Update phase in status.
	intent.Status.Phase = shared.ProcessingPhaseToNetworkIntentPhase(phase)

	// Update timestamps.
	now := metav1.Now()
	intent.Status.LastUpdateTime = now

	// Update completion time in extension if needed.
	// intent.Status.ProcessingCompletionTime field doesn't exist in v1 API.

	// Update conditions.
	condition := metav1.Condition{
		Type:               string(phase),
		Status:             metav1.ConditionFalse,
		ObservedGeneration: intent.Generation,
		Reason:             "Processing",
		Message:            fmt.Sprintf("Phase %s is processing", phase),
		LastTransitionTime: now,
	}

	if result.Success {
		condition.Status = metav1.ConditionTrue
		condition.Reason = "Completed"
		condition.Message = fmt.Sprintf("Phase %s completed successfully", phase)
	} else if result.ErrorMessage != "" {
		condition.Reason = "Failed"
		condition.Message = result.ErrorMessage
	}

	// Update conditions array.
	for i, existing := range intent.Status.Conditions {
		if existing.Type == condition.Type {
			intent.Status.Conditions[i] = condition
			return o.Status().Update(ctx, intent)
		}
	}

	intent.Status.Conditions = append(intent.Status.Conditions, condition)
	return o.Status().Update(ctx, intent)
}

func (o *IntentOrchestrator) calculateBackoff(retryCount int) time.Duration {
	backoff := o.Config.RetryBackoffBase
	for i := 0; i < retryCount; i++ {
		backoff *= 2
		if backoff > o.Config.RetryBackoffMax {
			backoff = o.Config.RetryBackoffMax
			break
		}
	}
	return backoff
}

func (o *IntentOrchestrator) getPhasePriority(phase interfaces.ProcessingPhase) int {
	if config, exists := o.Config.PhaseConfigs[phase]; exists {
		return config.Priority
	}
	return 0 // Default priority
}

func (o *IntentOrchestrator) handleIntentCompletion(ctx context.Context, intent *nephoranv1.NetworkIntent, processingCtx *interfaces.ProcessingContext) (ctrl.Result, error) {
	o.Logger.Info("Intent processing completed", "intent", intent.Name)

	// Clean up processing context.
	o.ProcessingContexts.Delete(processingCtx.IntentID)

	// Record completion metrics.
	o.MetricsCollector.RecordIntentCompletion(processingCtx.IntentID, true, time.Since(processingCtx.StartTime))

	// Update final status.
	intent.Status.Phase = shared.ProcessingPhaseToNetworkIntentPhase(interfaces.PhaseCompleted)
	now := metav1.Now()
	intent.Status.LastUpdateTime = now

	return ctrl.Result{}, o.Status().Update(ctx, intent)
}

func (o *IntentOrchestrator) handleIntentFailure(ctx context.Context, intent *nephoranv1.NetworkIntent, phase interfaces.ProcessingPhase, err error, processingCtx *interfaces.ProcessingContext) (ctrl.Result, error) {
	o.Logger.Error(err, "Intent processing failed", "intent", intent.Name, "phase", phase)

	// Clean up processing context.
	o.ProcessingContexts.Delete(processingCtx.IntentID)

	// Record failure metrics.
	o.MetricsCollector.RecordIntentCompletion(processingCtx.IntentID, false, time.Since(processingCtx.StartTime))

	// Update status.
	intent.Status.Phase = shared.ProcessingPhaseToNetworkIntentPhase(interfaces.PhaseFailed)

	// Record failure event.
	o.Recorder.Event(intent, "Warning", "ProcessingFailed", err.Error())

	return ctrl.Result{}, o.Status().Update(ctx, intent)
}

func (o *IntentOrchestrator) handleIntentDeletion(ctx context.Context, namespacedName types.NamespacedName) (ctrl.Result, error) {
	o.Logger.Info("Handling intent deletion", "intent", namespacedName)

	// Clean up any processing contexts.
	// This is a simplified cleanup - in production, you'd want more thorough cleanup.
	o.ProcessingContexts.Range(func(key, value interface{}) bool {
		if processingCtx, ok := value.(*interfaces.ProcessingContext); ok {
			// Check if this context belongs to the deleted intent.
			// This would need more sophisticated matching in practice.
			_ = processingCtx // Mark as used to avoid compiler warning
			o.ProcessingContexts.Delete(key)
		}
		return true
	})

	return ctrl.Result{}, nil
}

func (o *IntentOrchestrator) handlePhaseError(ctx context.Context, intent *nephoranv1.NetworkIntent, phase interfaces.ProcessingPhase, err error, processingCtx *interfaces.ProcessingContext) (ctrl.Result, error) {
	o.Logger.Error(err, "Phase error occurred", "intent", intent.Name, "phase", phase)

	// Record error in processing context.
	phaseMetrics := processingCtx.PhaseMetrics[phase]
	phaseMetrics.ErrorCount++
	processingCtx.PhaseMetrics[phase] = phaseMetrics

	// Create error result.
	result := interfaces.ProcessingResult{
		Success:      false,
		ErrorMessage: err.Error(),
		ErrorCode:    "PHASE_EXECUTION_ERROR",
	}

	return o.handlePhaseResult(ctx, intent, phase, result, processingCtx)
}

func (o *IntentOrchestrator) handlePhaseEvent(ctx context.Context, event ProcessingEvent) error {
	o.Logger.Info("Received phase event", "type", event.Type, "intentId", event.IntentID, "phase", event.Phase)

	// Handle cross-phase coordination here if needed.
	// For example, triggering dependent phases when prerequisites are met.

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (o *IntentOrchestrator) SetupWithManager(mgr ctrl.Manager) error {
	// Set up the main reconciler.
	return ctrl.NewControllerManagedBy(mgr).
		For(&nephoranv1.NetworkIntent{}).
		Named("intent-orchestrator").
		Complete(o)
}

// Start starts the orchestrator and all its components.
func (o *IntentOrchestrator) Start(ctx context.Context) error {
	o.Logger.Info("Starting Intent Orchestrator")

	// Start all components.
	if err := o.EventBus.Start(ctx); err != nil {
		return fmt.Errorf("failed to start event bus: %w", err)
	}

	if err := o.WorkQueueManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start work queue manager: %w", err)
	}

	// Start all registered controllers.
	for phase, controller := range o.Controllers {
		if err := controller.Start(ctx); err != nil {
			return fmt.Errorf("failed to start controller for phase %s: %w", phase, err)
		}
	}

	o.Logger.Info("Intent Orchestrator started successfully")
	return nil
}

// Stop stops the orchestrator and all its components.
func (o *IntentOrchestrator) Stop(ctx context.Context) error {
	o.Logger.Info("Stopping Intent Orchestrator")

	// Stop all registered controllers.
	for phase, controller := range o.Controllers {
		if err := controller.Stop(ctx); err != nil {
			o.Logger.Error(err, "Error stopping controller", "phase", phase)
		}
	}

	// Stop components.
	if err := o.WorkQueueManager.Stop(ctx); err != nil {
		o.Logger.Error(err, "Error stopping work queue manager")
	}

	if err := o.EventBus.Stop(ctx); err != nil {
		o.Logger.Error(err, "Error stopping event bus")
	}

	o.Logger.Info("Intent Orchestrator stopped")
	return nil
}
