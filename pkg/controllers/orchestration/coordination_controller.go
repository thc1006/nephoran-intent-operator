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
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/interfaces"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// CoordinationController coordinates the overall intent processing pipeline.

type CoordinationController struct {
	client.Client

	Scheme *runtime.Scheme

	Recorder record.EventRecorder

	Logger logr.Logger

	// Orchestration components.

	IntentOrchestrator *IntentOrchestrator

	EventBus *EventBus

	WorkQueueManager *WorkQueueManager

	StateMachine *StateMachine

	LockManager *IntentLockManager

	// Phase controllers.

	IntentProcessingController *IntentProcessingController

	ResourcePlanningController *ResourcePlanningController

	ManifestGenerationController *ManifestGenerationController

	GitOpsDeploymentController *GitOpsDeploymentController

	// Coordination state.

	activeIntents sync.Map // map[string]*CoordinationContext

	phaseTracker *PhaseTracker

	conflictResolver *ConflictResolver

	recoveryManager *RecoveryManager

	// Configuration.

	Config *CoordinationConfig

	// Metrics.

	MetricsCollector *MetricsCollector
}

// CoordinationConfig contains configuration for coordination.

type CoordinationConfig struct {
	MaxConcurrentIntents int `json:"maxConcurrentIntents"`

	PhaseTimeout time.Duration `json:"phaseTimeout"`

	MaxRetries int `json:"maxRetries"`

	RetryBackoff time.Duration `json:"retryBackoff"`

	EnableParallelProcessing bool `json:"enableParallelProcessing"`

	EnableConflictResolution bool `json:"enableConflictResolution"`

	EnableAutoRecovery bool `json:"enableAutoRecovery"`

	HealthCheckInterval time.Duration `json:"healthCheckInterval"`

	RecoveryTimeout time.Duration `json:"recoveryTimeout"`
}

// CoordinationContext tracks the context for an intent's coordination.

type CoordinationContext struct {
	IntentID string

	CorrelationID string

	StartTime time.Time

	CurrentPhase interfaces.ProcessingPhase

	PhaseHistory []PhaseExecution

	Locks []string // Changed to []string for compatibility with event_driven_coordinator

	Conflicts []Conflict

	RetryCount int

	LastActivity time.Time

	mutex sync.RWMutex

	// Additional fields for event-driven coordinator compatibility.

	LastUpdateTime time.Time `json:"lastUpdateTime"`

	Metadata json.RawMessage `json:"metadata"`

	CompletedPhases []interfaces.ProcessingPhase `json:"completedPhases"`

	ErrorHistory []string `json:"errorHistory"`
}

// PhaseExecution tracks the execution of a phase.

type PhaseExecution struct {
	Phase interfaces.ProcessingPhase

	StartTime time.Time

	CompletionTime *time.Time

	Status string // Pending, InProgress, Completed, Failed

	Error error

	RetryCount int

	ResourceRefs []ResourceReference
}

// ResourceLock represents a lock on a resource.

type ResourceLock struct {
	ResourceID string

	ResourceType string

	LockType string // Shared, Exclusive

	AcquiredAt time.Time

	Owner string
}

// Conflict represents a resource conflict.

type Conflict struct {
	ID string

	Type string // ResourceConflict, DependencyConflict, etc.

	Description string

	InvolvedIntents []string

	Severity string // Low, Medium, High, Critical

	DetectedAt time.Time

	ResolvedAt *time.Time

	Resolution string
}

// ResourceReference represents a reference to a created resource.

type ResourceReference struct {
	APIVersion string

	Kind string

	Name string

	Namespace string

	UID string

	Phase interfaces.ProcessingPhase
}

// NewCoordinationController creates a new coordination controller.

func NewCoordinationController(
	client client.Client,

	scheme *runtime.Scheme,

	recorder record.EventRecorder,

	orchestrator *IntentOrchestrator,

	config *CoordinationConfig,
) *CoordinationController {
	logger := log.Log.WithName("coordination-controller")

	return &CoordinationController{
		Client: client,

		Scheme: scheme,

		Recorder: recorder,

		Logger: logger,

		IntentOrchestrator: orchestrator,

		EventBus: orchestrator.EventBus,

		WorkQueueManager: orchestrator.WorkQueueManager,

		StateMachine: orchestrator.StateMachine,

		LockManager: orchestrator.LockManager,

		Config: config,

		phaseTracker: NewPhaseTracker(),

		conflictResolver: NewConflictResolver(client, logger),

		recoveryManager: NewRecoveryManager(client, logger),

		MetricsCollector: NewMetricsCollector(),
	}
}

// Reconcile coordinates the overall intent processing.

func (r *CoordinationController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Logger.WithValues("networkintent", req.NamespacedName)

	// Fetch the NetworkIntent instance.

	networkIntent := &nephoranv1.NetworkIntent{}

	if err := r.Get(ctx, req.NamespacedName, networkIntent); err != nil {

		if apierrors.IsNotFound(err) {
			// Handle cleanup if intent was deleted.

			return r.handleIntentDeletion(ctx, req.NamespacedName)
		}

		log.Error(err, "Failed to get NetworkIntent")

		return ctrl.Result{}, err

	}

	// Handle deletion.

	if networkIntent.DeletionTimestamp != nil {
		return r.handleIntentDeletion(ctx, req.NamespacedName)
	}

	// Add finalizer if not present.

	if !controllerutil.ContainsFinalizer(networkIntent, "coordination.nephoran.com/finalizer") {

		controllerutil.AddFinalizer(networkIntent, "coordination.nephoran.com/finalizer")

		return ctrl.Result{}, r.Update(ctx, networkIntent)

	}

	// Coordinate the intent processing.

	return r.coordinateIntent(ctx, networkIntent)
}

// coordinateIntent coordinates the processing of a NetworkIntent.

func (r *CoordinationController) coordinateIntent(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) (ctrl.Result, error) {
	log := r.Logger.WithValues("intent", networkIntent.Name, "namespace", networkIntent.Namespace)

	// Get or create coordination context.

	coordCtx := r.getOrCreateCoordinationContext(networkIntent)

	// Update activity timestamp.

	coordCtx.mutex.Lock()

	coordCtx.LastActivity = time.Now()

	coordCtx.mutex.Unlock()

	// Acquire intent lock to prevent concurrent processing.

	intentLock, err := r.LockManager.AcquireIntentLock(ctx, coordCtx.IntentID)
	if err != nil {

		log.Error(err, "Failed to acquire intent lock")

		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil

	}

	defer intentLock.Release()

	// Check for conflicts.

	if r.Config.EnableConflictResolution {

		conflicts, err := r.detectConflicts(ctx, networkIntent, coordCtx)
		if err != nil {

			log.Error(err, "Failed to detect conflicts")

			return ctrl.Result{}, err

		}

		if len(conflicts) > 0 {
			return r.handleConflicts(ctx, networkIntent, coordCtx, conflicts)
		}

	}

	// Determine current phase and coordinate next steps.

	currentPhase := r.StateMachine.GetCurrentPhase(networkIntent)

	log.Info("Coordinating intent processing", "currentPhase", currentPhase)

	// Update coordination context.

	coordCtx.mutex.Lock()

	coordCtx.CurrentPhase = currentPhase

	coordCtx.mutex.Unlock()

	// Execute phase coordination.

	result, err := r.coordinatePhase(ctx, networkIntent, currentPhase, coordCtx)
	if err != nil {
		return r.handleCoordinationError(ctx, networkIntent, currentPhase, err, coordCtx)
	}

	// Update phase tracking.

	r.phaseTracker.UpdatePhaseStatus(coordCtx.IntentID, currentPhase, "Completed")

	// Handle phase completion.

	return r.handlePhaseCompletion(ctx, networkIntent, currentPhase, result, coordCtx)
}

// coordinatePhase coordinates the execution of a specific phase.

func (r *CoordinationController) coordinatePhase(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, phase interfaces.ProcessingPhase, coordCtx *CoordinationContext) (interfaces.ProcessingResult, error) {
	// Record phase execution start.

	phaseExec := PhaseExecution{
		Phase: phase,

		StartTime: time.Now(),

		Status: "InProgress",

		RetryCount: 0,
	}

	coordCtx.mutex.Lock()

	coordCtx.PhaseHistory = append(coordCtx.PhaseHistory, phaseExec)

	phaseIndex := len(coordCtx.PhaseHistory) - 1

	coordCtx.mutex.Unlock()

	// Create or manage phase-specific resources.

	var result interfaces.ProcessingResult

	var err error

	switch phase {

	case interfaces.PhaseLLMProcessing:

		result, err = r.coordinateLLMProcessing(ctx, networkIntent, coordCtx)

	case interfaces.PhaseResourcePlanning:

		result, err = r.coordinateResourcePlanning(ctx, networkIntent, coordCtx)

	case interfaces.PhaseManifestGeneration:

		result, err = r.coordinateManifestGeneration(ctx, networkIntent, coordCtx)

	case interfaces.PhaseGitOpsCommit:

		result, err = r.coordinateGitOpsDeployment(ctx, networkIntent, coordCtx)

	case interfaces.PhaseDeploymentVerification:

		result, err = r.coordinateDeploymentVerification(ctx, networkIntent, coordCtx)

	default:

		return interfaces.ProcessingResult{}, fmt.Errorf("unknown phase: %s", phase)

	}

	// Update phase execution status.

	coordCtx.mutex.Lock()

	if phaseIndex < len(coordCtx.PhaseHistory) {

		now := time.Now()

		coordCtx.PhaseHistory[phaseIndex].CompletionTime = &now

		if err != nil {

			coordCtx.PhaseHistory[phaseIndex].Status = "Failed"

			coordCtx.PhaseHistory[phaseIndex].Error = err

		} else {
			coordCtx.PhaseHistory[phaseIndex].Status = "Completed"
		}

	}

	coordCtx.mutex.Unlock()

	return result, err
}

// coordinateLLMProcessing coordinates LLM processing phase.

func (r *CoordinationController) coordinateLLMProcessing(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, coordCtx *CoordinationContext) (interfaces.ProcessingResult, error) {
	// Check if IntentProcessing resource already exists.

	intentProcessing := &nephoranv1.IntentProcessing{}

	intentProcessingKey := types.NamespacedName{
		Namespace: networkIntent.Namespace,

		Name: fmt.Sprintf("%s-processing", networkIntent.Name),
	}

	err := r.Get(ctx, intentProcessingKey, intentProcessing)

	if err != nil && !apierrors.IsNotFound(err) {
		return interfaces.ProcessingResult{}, fmt.Errorf("failed to get IntentProcessing: %w", err)
	}

	// Create IntentProcessing resource if it doesn't exist.

	if apierrors.IsNotFound(err) {

		intentProcessing = &nephoranv1.IntentProcessing{
			ObjectMeta: metav1.ObjectMeta{
				Name: intentProcessingKey.Name,

				Namespace: intentProcessingKey.Namespace,

				Labels: map[string]string{
					"nephoran.com/intent-name": networkIntent.Name,

					"nephoran.com/phase": string(interfaces.PhaseLLMProcessing),
				},
			},

			Spec: nephoranv1.IntentProcessingSpec{
				ParentIntentRef: &nephoranv1.ObjectReference{
					Kind: "NetworkIntent",

					Name: networkIntent.Name,

					Namespace: networkIntent.Namespace,

					UID: string(networkIntent.UID),
				},

				OriginalIntent: networkIntent.Spec.Intent,

				Priority: nephoranv1.ConvertNetworkPriorityToPriority(networkIntent.Spec.Priority),

				// Use default values for timeout and retries as these are not part of NetworkIntent spec.

				// TimeoutSeconds: defaults to 120 seconds per IntentProcessing defaults.

				// MaxRetries: defaults to 3 per IntentProcessing defaults.

			},
		}

		if err := r.Create(ctx, intentProcessing); err != nil {
			return interfaces.ProcessingResult{}, fmt.Errorf("failed to create IntentProcessing: %w", err)
		}

		r.Logger.Info("Created IntentProcessing resource", "name", intentProcessing.Name)

	}

	// Wait for processing completion.

	result := interfaces.ProcessingResult{Success: false}

	if intentProcessing.IsProcessingComplete() {

		result.Success = true

		result.NextPhase = interfaces.PhaseResourcePlanning

		result.Data = json.RawMessage("{}")

		// Add resource reference.

		coordCtx.mutex.Lock()

		for i := range coordCtx.PhaseHistory {
			if coordCtx.PhaseHistory[i].Phase == interfaces.PhaseLLMProcessing && coordCtx.PhaseHistory[i].Status == "InProgress" {

				coordCtx.PhaseHistory[i].ResourceRefs = append(coordCtx.PhaseHistory[i].ResourceRefs, ResourceReference{
					APIVersion: "nephoran.com/v1",

					Kind: "IntentProcessing",

					Name: intentProcessing.Name,

					Namespace: intentProcessing.Namespace,

					UID: string(intentProcessing.UID),

					Phase: interfaces.PhaseLLMProcessing,
				})

				break

			}
		}

		coordCtx.mutex.Unlock()

	} else if intentProcessing.IsProcessingFailed() {

		result.Success = false

		result.ErrorMessage = "LLM processing failed"

		result.ErrorCode = "LLM_PROCESSING_FAILED"

	} else {
		// Still processing - requeue.

		return interfaces.ProcessingResult{
			Success: false,

			RetryAfter: func() *time.Duration { d := 10 * time.Second; return &d }(),
		}, nil
	}

	return result, nil
}

// coordinateResourcePlanning coordinates resource planning phase.

func (r *CoordinationController) coordinateResourcePlanning(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, coordCtx *CoordinationContext) (interfaces.ProcessingResult, error) {
	// Get the LLM processing results first.

	intentProcessing := &nephoranv1.IntentProcessing{}

	intentProcessingKey := types.NamespacedName{
		Namespace: networkIntent.Namespace,

		Name: fmt.Sprintf("%s-processing", networkIntent.Name),
	}

	if err := r.Get(ctx, intentProcessingKey, intentProcessing); err != nil {
		return interfaces.ProcessingResult{}, fmt.Errorf("failed to get IntentProcessing: %w", err)
	}

	// Check if ResourcePlan resource already exists.

	resourcePlan := &nephoranv1.ResourcePlan{}

	resourcePlanKey := types.NamespacedName{
		Namespace: networkIntent.Namespace,

		Name: fmt.Sprintf("%s-plan", networkIntent.Name),
	}

	err := r.Get(ctx, resourcePlanKey, resourcePlan)

	if err != nil && !apierrors.IsNotFound(err) {
		return interfaces.ProcessingResult{}, fmt.Errorf("failed to get ResourcePlan: %w", err)
	}

	// Create ResourcePlan resource if it doesn't exist.

	if apierrors.IsNotFound(err) {

		resourcePlan = &nephoranv1.ResourcePlan{
			ObjectMeta: metav1.ObjectMeta{
				Name: resourcePlanKey.Name,

				Namespace: resourcePlanKey.Namespace,

				Labels: map[string]string{
					"nephoran.com/intent-name": networkIntent.Name,

					"nephoran.com/phase": string(interfaces.PhaseResourcePlanning),
				},
			},

			Spec: nephoranv1.ResourcePlanSpec{
				ParentIntentRef: nephoranv1.ObjectReference{
					Kind: "NetworkIntent",

					Name: networkIntent.Name,

					Namespace: networkIntent.Namespace,

					UID: string(networkIntent.UID),
				},

				IntentProcessingRef: &nephoranv1.ObjectReference{
					Kind: "IntentProcessing",

					Name: intentProcessing.Name,

					Namespace: intentProcessing.Namespace,

					UID: string(intentProcessing.UID),
				},

				RequirementsInput: func() runtime.RawExtension {
					if intentProcessing.Status.LLMResponse != nil {
						return *intentProcessing.Status.LLMResponse
					}
					return runtime.RawExtension{}
				}(),

				TargetComponents: r.convertNetworkTargetComponentsToTargetComponents(networkIntent.Spec.TargetComponents),

				ResourceConstraints: networkIntent.Spec.ResourceConstraints,

				Priority: nephoranv1.ConvertNetworkPriorityToPriority(networkIntent.Spec.Priority),
			},
		}

		if err := r.Create(ctx, resourcePlan); err != nil {
			return interfaces.ProcessingResult{}, fmt.Errorf("failed to create ResourcePlan: %w", err)
		}

		r.Logger.Info("Created ResourcePlan resource", "name", resourcePlan.Name)

	}

	// Wait for planning completion.

	result := interfaces.ProcessingResult{Success: false}

	if resourcePlan.IsPlanningComplete() {

		result.Success = true

		result.NextPhase = interfaces.PhaseManifestGeneration

		result.Data = json.RawMessage("{}")

	} else if resourcePlan.IsPlanningFailed() {

		result.Success = false

		result.ErrorMessage = "Resource planning failed"

		result.ErrorCode = "RESOURCE_PLANNING_FAILED"

	} else {
		// Still planning - requeue.

		return interfaces.ProcessingResult{
			Success: false,

			RetryAfter: func() *time.Duration { d := 15 * time.Second; return &d }(),
		}, nil
	}

	return result, nil
}

// coordinateManifestGeneration coordinates manifest generation phase.

func (r *CoordinationController) coordinateManifestGeneration(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, coordCtx *CoordinationContext) (interfaces.ProcessingResult, error) {
	// For this implementation, we'll simulate manifest generation.

	// In practice, you would create and manage ManifestGeneration resources.

	result := interfaces.ProcessingResult{
		Success: true,

		NextPhase: interfaces.PhaseGitOpsCommit,

		Data: json.RawMessage("{}"),
		},
	}

	return result, nil
}

// coordinateGitOpsDeployment coordinates GitOps deployment phase.

func (r *CoordinationController) coordinateGitOpsDeployment(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, coordCtx *CoordinationContext) (interfaces.ProcessingResult, error) {
	// For this implementation, we'll simulate GitOps deployment.

	// In practice, you would create and manage GitOpsDeployment resources.

	result := interfaces.ProcessingResult{
		Success: true,

		NextPhase: interfaces.PhaseDeploymentVerification,

		Data: json.RawMessage("{}"),
	}

	return result, nil
}

// coordinateDeploymentVerification coordinates deployment verification phase.

func (r *CoordinationController) coordinateDeploymentVerification(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, coordCtx *CoordinationContext) (interfaces.ProcessingResult, error) {
	// For this implementation, we'll simulate deployment verification.

	// In practice, you would verify the actual deployment status.

	result := interfaces.ProcessingResult{
		Success: true,

		NextPhase: interfaces.PhaseCompleted,

		Data: json.RawMessage("{}"),
		},
	}

	return result, nil
}

// detectConflicts detects conflicts with other intents.

func (r *CoordinationController) detectConflicts(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, coordCtx *CoordinationContext) ([]Conflict, error) {
	var conflicts []Conflict

	// Check for resource conflicts with other active intents.

	r.activeIntents.Range(func(key, value interface{}) bool {
		otherIntentID := key.(string)

		_ = value.(*CoordinationContext) // otherCoordCtx not used in this stub

		if otherIntentID == coordCtx.IntentID {
			return true // Skip self
		}

		// Check for namespace conflicts.

		if r.hasNamespaceConflict(ctx, networkIntent, otherIntentID) {

			conflict := Conflict{
				ID: fmt.Sprintf("namespace-conflict-%s-%s", coordCtx.IntentID, otherIntentID),

				Type: "NamespaceConflict",

				Description: fmt.Sprintf("Namespace conflict detected between intents %s and %s", coordCtx.IntentID, otherIntentID),

				InvolvedIntents: []string{coordCtx.IntentID, otherIntentID},

				Severity: "Medium",

				DetectedAt: time.Now(),
			}

			conflicts = append(conflicts, conflict)

		}

		return true
	})

	return conflicts, nil
}

// hasNamespaceConflict checks for namespace conflicts.

func (r *CoordinationController) hasNamespaceConflict(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, otherIntentID string) bool {
	// Simplified conflict detection - check if targeting same namespace with same component.

	// In practice, this would be more sophisticated.

	return networkIntent.Spec.TargetNamespace != "" &&

		len(networkIntent.Spec.TargetComponents) > 0
}

// handleConflicts handles detected conflicts.

func (r *CoordinationController) handleConflicts(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, coordCtx *CoordinationContext, conflicts []Conflict) (ctrl.Result, error) {
	log := r.Logger.WithValues("intent", networkIntent.Name, "conflicts", len(conflicts))

	// Update coordination context with conflicts.

	coordCtx.mutex.Lock()

	coordCtx.Conflicts = append(coordCtx.Conflicts, conflicts...)

	coordCtx.mutex.Unlock()

	// Publish conflict detection event.

	if err := r.EventBus.PublishPhaseEvent(ctx, interfaces.PhaseIntentReceived, EventConflictDetected,

		coordCtx.IntentID, false, json.RawMessage("{}")); err != nil {
		log.Error(err, "Failed to publish conflict detection event")
	}

	// Attempt to resolve conflicts.

	for _, conflict := range conflicts {

		resolved, err := r.conflictResolver.ResolveConflict(ctx, conflict)
		if err != nil {

			log.Error(err, "Failed to resolve conflict", "conflictId", conflict.ID)

			continue

		}

		if resolved {

			log.Info("Conflict resolved", "conflictId", conflict.ID)

			// Update conflict status.

			now := time.Now()

			conflict.ResolvedAt = &now

			conflict.Resolution = "Automatically resolved"

			// Publish resolution event.

			if err := r.EventBus.PublishPhaseEvent(ctx, interfaces.PhaseIntentReceived, EventConflictResolved,

				coordCtx.IntentID, true, json.RawMessage("{}")); err != nil {
				log.Error(err, "Failed to publish conflict resolution event")
			}

		} else {

			r.Logger.Info("Conflict could not be resolved automatically", "conflictId", conflict.ID)

			// Manual intervention required.

			r.Recorder.Event(networkIntent, "Warning", "ConflictDetected",

				fmt.Sprintf("Conflict detected and requires manual intervention: %s", conflict.Description))

		}

	}

	// Check if all conflicts are resolved.

	unresolvedConflicts := 0

	for _, conflict := range conflicts {
		if conflict.ResolvedAt == nil {
			unresolvedConflicts++
		}
	}

	if unresolvedConflicts > 0 {
		// Requeue to check again later.

		return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
	}

	// All conflicts resolved, continue processing.

	log.Info("All conflicts resolved, continuing processing")

	return ctrl.Result{RequeueAfter: time.Second}, nil
}

// extractConflictTypes extracts conflict types from conflicts.

func (r *CoordinationController) extractConflictTypes(conflicts []Conflict) []string {
	types := make([]string, len(conflicts))

	for i, conflict := range conflicts {
		types[i] = conflict.Type
	}

	return types
}

// handlePhaseCompletion handles the completion of a phase.

func (r *CoordinationController) handlePhaseCompletion(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, phase interfaces.ProcessingPhase, result interfaces.ProcessingResult, coordCtx *CoordinationContext) (ctrl.Result, error) {
	log := r.Logger.WithValues("intent", networkIntent.Name, "phase", phase, "success", result.Success)

	if result.Success {

		// Update NetworkIntent status.

		if err := r.updateNetworkIntentStatus(ctx, networkIntent, phase, result); err != nil {

			log.Error(err, "Failed to update NetworkIntent status")

			return ctrl.Result{}, err

		}

		// Check if processing is complete.

		if result.NextPhase == interfaces.PhaseCompleted {
			return r.handleIntentCompletion(ctx, networkIntent, coordCtx)
		}

		// Determine retry behavior.

		if result.RetryAfter != nil {
			return ctrl.Result{RequeueAfter: *result.RetryAfter}, nil
		}

		// Continue to next phase.

		return ctrl.Result{RequeueAfter: time.Second}, nil

	}

	// Handle phase failure.

	return r.handlePhaseFailure(ctx, networkIntent, phase, result, coordCtx)
}

// handleIntentCompletion handles successful intent completion.

func (r *CoordinationController) handleIntentCompletion(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, coordCtx *CoordinationContext) (ctrl.Result, error) {
	log := r.Logger.WithValues("intent", networkIntent.Name)

	log.Info("Intent processing completed successfully")

	// Update NetworkIntent to completed status.

	networkIntent.Status.Phase = shared.ProcessingPhaseToNetworkIntentPhase(interfaces.PhaseCompleted)

	networkIntent.Status.LastUpdateTime = metav1.Now()

	networkIntent.Status.LastMessage = "Intent processing completed successfully"

	if err := r.Status().Update(ctx, networkIntent); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update NetworkIntent status: %w", err)
	}

	// Record completion event.

	r.Recorder.Event(networkIntent, "Normal", "IntentCompleted", "Network intent processing completed successfully")

	// Publish completion event.

	if err := r.EventBus.PublishPhaseEvent(ctx, interfaces.PhaseCompleted, EventIntentCompleted,

		coordCtx.IntentID, true, json.RawMessage("{}")); err != nil {
		log.Error(err, "Failed to publish completion event")
	}

	// Clean up coordination context.

	r.activeIntents.Delete(coordCtx.IntentID)

	// Record metrics.

	r.MetricsCollector.RecordIntentCompletion(coordCtx.IntentID, true, time.Since(coordCtx.StartTime))

	return ctrl.Result{}, nil
}

// handlePhaseFailure handles phase failures.

func (r *CoordinationController) handlePhaseFailure(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, phase interfaces.ProcessingPhase, result interfaces.ProcessingResult, coordCtx *CoordinationContext) (ctrl.Result, error) {
	log := r.Logger.WithValues("intent", networkIntent.Name, "phase", phase)

	log.Error(errors.New(result.ErrorMessage), "Phase failed", "errorCode", result.ErrorCode)

	// Increment retry count.

	coordCtx.mutex.Lock()

	coordCtx.RetryCount++

	retryCount := coordCtx.RetryCount

	coordCtx.mutex.Unlock()

	// Check if auto-recovery is enabled and should be attempted.

	if r.Config.EnableAutoRecovery && r.shouldAttemptRecovery(phase, retryCount) {
		return r.attemptRecovery(ctx, networkIntent, phase, result, coordCtx)
	}

	// Check if we should retry.

	if retryCount < r.Config.MaxRetries {

		backoffDuration := r.calculateBackoff(retryCount)

		log.Info("Scheduling retry", "attempt", retryCount+1, "backoff", backoffDuration)

		r.Recorder.Event(networkIntent, "Warning", "PhaseRetry",

			fmt.Sprintf("Retrying phase %s (attempt %d/%d): %s", phase, retryCount, r.Config.MaxRetries, result.ErrorMessage))

		return ctrl.Result{RequeueAfter: backoffDuration}, nil

	}

	// Max retries exceeded - mark as permanently failed.

	return r.handleIntentFailure(ctx, networkIntent, phase, result, coordCtx)
}

// shouldAttemptRecovery determines if recovery should be attempted.

func (r *CoordinationController) shouldAttemptRecovery(phase interfaces.ProcessingPhase, retryCount int) bool {
	// Only attempt recovery for certain phases and within retry limits.

	recoverablePhases := map[interfaces.ProcessingPhase]bool{
		interfaces.PhaseLLMProcessing: true,

		interfaces.PhaseResourcePlanning: true,

		interfaces.PhaseManifestGeneration: true,

		interfaces.PhaseGitOpsCommit: false, // Git operations are not easily recoverable

		interfaces.PhaseDeploymentVerification: true,
	}

	return recoverablePhases[phase] && retryCount < r.Config.MaxRetries/2
}

// attemptRecovery attempts to recover from a phase failure.

func (r *CoordinationController) attemptRecovery(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, phase interfaces.ProcessingPhase, result interfaces.ProcessingResult, coordCtx *CoordinationContext) (ctrl.Result, error) {
	log := r.Logger.WithValues("intent", networkIntent.Name, "phase", phase)

	log.Info("Attempting automatic recovery")

	// Publish recovery initiation event.

	if err := r.EventBus.PublishPhaseEvent(ctx, phase, EventRecoveryInitiated,

		coordCtx.IntentID, false, json.RawMessage("{}")); err != nil {
		log.Error(err, "Failed to publish recovery initiation event")
	}

	// Attempt recovery using the recovery manager.

	recovered, recoveryActions, err := r.recoveryManager.AttemptRecovery(ctx, networkIntent, phase, result.ErrorCode, coordCtx)
	if err != nil {

		log.Error(err, "Recovery attempt failed")

		return r.handleRecoveryFailure(ctx, networkIntent, phase, result, coordCtx)

	}

	if recovered {

		log.Info("Recovery successful", "actions", recoveryActions)

		r.Recorder.Event(networkIntent, "Normal", "RecoverySuccessful",

			fmt.Sprintf("Successfully recovered from %s failure: %s", phase, strings.Join(recoveryActions, ", ")))

		// Publish recovery completion event.

		if err := r.EventBus.PublishPhaseEvent(ctx, phase, EventRecoveryCompleted,

			coordCtx.IntentID, true, json.RawMessage("{}")); err != nil {
			log.Error(err, "Failed to publish recovery completion event")
		}

		// Retry the phase immediately.

		return ctrl.Result{RequeueAfter: time.Second}, nil

	}

	// Recovery was not successful.

	return r.handleRecoveryFailure(ctx, networkIntent, phase, result, coordCtx)
}

// handleRecoveryFailure handles recovery failure.

func (r *CoordinationController) handleRecoveryFailure(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, phase interfaces.ProcessingPhase, result interfaces.ProcessingResult, coordCtx *CoordinationContext) (ctrl.Result, error) {
	log := r.Logger.WithValues("intent", networkIntent.Name, "phase", phase)

	log.V(1).Info("Recovery failed, falling back to normal retry logic")

	r.Recorder.Event(networkIntent, "Warning", "RecoveryFailed",

		fmt.Sprintf("Failed to recover from %s failure, falling back to retry", phase))

	// Calculate backoff and retry.

	backoffDuration := r.calculateBackoff(coordCtx.RetryCount)

	return ctrl.Result{RequeueAfter: backoffDuration}, nil
}

// handleIntentFailure handles permanent intent failure.

func (r *CoordinationController) handleIntentFailure(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, phase interfaces.ProcessingPhase, result interfaces.ProcessingResult, coordCtx *CoordinationContext) (ctrl.Result, error) {
	log := r.Logger.WithValues("intent", networkIntent.Name, "phase", phase)

	log.Error(errors.New(result.ErrorMessage), "Intent processing failed permanently")

	// Update NetworkIntent to failed status.

	networkIntent.Status.Phase = shared.ProcessingPhaseToNetworkIntentPhase(interfaces.PhaseFailed)

	networkIntent.Status.LastUpdateTime = metav1.Now()

	networkIntent.Status.LastMessage = fmt.Sprintf("Processing failed at phase %s after %d attempts: %s", phase, coordCtx.RetryCount, result.ErrorMessage)

	if err := r.Status().Update(ctx, networkIntent); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update NetworkIntent status: %w", err)
	}

	// Record failure event.

	r.Recorder.Event(networkIntent, "Warning", "IntentFailed",

		fmt.Sprintf("Network intent processing failed at phase %s: %s", phase, result.ErrorMessage))

	// Publish failure event.

	if err := r.EventBus.PublishPhaseEvent(ctx, phase, EventIntentFailed,

		coordCtx.IntentID, false, json.RawMessage("{}")); err != nil {
		log.Error(err, "Failed to publish failure event")
	}

	// Clean up coordination context.

	r.activeIntents.Delete(coordCtx.IntentID)

	// Record metrics.

	r.MetricsCollector.RecordIntentCompletion(coordCtx.IntentID, false, time.Since(coordCtx.StartTime))

	return ctrl.Result{}, nil
}

// handleCoordinationError handles coordination errors.

func (r *CoordinationController) handleCoordinationError(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, phase interfaces.ProcessingPhase, err error, coordCtx *CoordinationContext) (ctrl.Result, error) {
	log := r.Logger.WithValues("intent", networkIntent.Name, "phase", phase)

	log.Error(err, "Coordination error occurred")

	// Create error result.

	result := interfaces.ProcessingResult{
		Success: false,

		ErrorMessage: err.Error(),

		ErrorCode: "COORDINATION_ERROR",
	}

	return r.handlePhaseFailure(ctx, networkIntent, phase, result, coordCtx)
}

// handleIntentDeletion handles intent deletion.

func (r *CoordinationController) handleIntentDeletion(ctx context.Context, namespacedName types.NamespacedName) (ctrl.Result, error) {
	log := r.Logger.WithValues("intent", namespacedName)

	log.Info("Handling intent deletion")

	// Clean up coordination contexts.

	r.activeIntents.Range(func(key, value interface{}) bool {
		intentID := key.(string)

		_ = value.(*CoordinationContext)

		// This is a simplified matching - in practice you'd match by namespace/name.

		r.activeIntents.Delete(intentID)

		log.Info("Cleaned up coordination context", "intentId", intentID)

		return true
	})

	return ctrl.Result{}, nil
}

// updateNetworkIntentStatus updates the NetworkIntent status.

func (r *CoordinationController) updateNetworkIntentStatus(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, phase interfaces.ProcessingPhase, result interfaces.ProcessingResult) error {
	// Update phase and basic status fields.

	networkIntent.Status.Phase = shared.ProcessingPhaseToNetworkIntentPhase(phase)

	networkIntent.Status.LastUpdateTime = metav1.Now()

	networkIntent.Status.LastMessage = fmt.Sprintf("Phase %s completed successfully", phase)

	return r.Status().Update(ctx, networkIntent)
}

// getOrCreateCoordinationContext gets or creates a coordination context.

func (r *CoordinationController) getOrCreateCoordinationContext(networkIntent *nephoranv1.NetworkIntent) *CoordinationContext {
	intentID := string(networkIntent.UID)

	if ctx, exists := r.activeIntents.Load(intentID); exists {
		return ctx.(*CoordinationContext)
	}

	coordCtx := &CoordinationContext{
		IntentID: intentID,

		CorrelationID: fmt.Sprintf("coord-%s-%d", networkIntent.Name, time.Now().Unix()),

		StartTime: time.Now(),

		CurrentPhase: interfaces.PhaseIntentReceived,

		PhaseHistory: make([]PhaseExecution, 0),

		Locks: make([]string, 0),

		Conflicts: make([]Conflict, 0),

		RetryCount: 0,

		LastActivity: time.Now(),

		LastUpdateTime: time.Now(),

		Metadata: make(map[string]interface{}),

		CompletedPhases: make([]interfaces.ProcessingPhase, 0),

		ErrorHistory: make([]string, 0),
	}

	r.activeIntents.Store(intentID, coordCtx)

	return coordCtx
}

// calculateBackoff calculates backoff duration for retries.

func (r *CoordinationController) calculateBackoff(retryCount int) time.Duration {
	backoff := r.Config.RetryBackoff

	for i := 1; i < retryCount; i++ {

		backoff *= 2

		if backoff > 5*time.Minute {

			backoff = 5 * time.Minute

			break

		}

	}

	return backoff
}

// SetupWithManager sets up the controller with the Manager.

func (r *CoordinationController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nephoranv1.NetworkIntent{}).
		Named("coordination").
		Complete(r)
}

// Start starts the coordination controller and all its components.

func (r *CoordinationController) Start(ctx context.Context) error {
	r.Logger.Info("Starting Coordination Controller")

	// Start health check routine.

	go r.healthCheckRoutine(ctx)

	// Start coordination context cleanup routine.

	go r.cleanupRoutine(ctx)

	r.Logger.Info("Coordination Controller started successfully")

	return nil
}

// healthCheckRoutine performs periodic health checks.

func (r *CoordinationController) healthCheckRoutine(ctx context.Context) {
	ticker := time.NewTicker(r.Config.HealthCheckInterval)

	defer ticker.Stop()

	for {
		select {

		case <-ticker.C:

			r.performHealthChecks(ctx)

		case <-ctx.Done():

			return

		}
	}
}

// performHealthChecks performs health checks on active intents.

func (r *CoordinationController) performHealthChecks(ctx context.Context) {
	r.activeIntents.Range(func(key, value interface{}) bool {
		intentID := key.(string)

		coordCtx := value.(*CoordinationContext)

		coordCtx.mutex.RLock()

		lastActivity := coordCtx.LastActivity

		currentPhase := coordCtx.CurrentPhase

		coordCtx.mutex.RUnlock()

		// Check for stale intents (no activity for too long).

		if time.Since(lastActivity) > r.Config.PhaseTimeout*2 {
			r.Logger.V(1).Info("Detected stale intent", "intentId", intentID, "lastActivity", lastActivity, "currentPhase", currentPhase)

			// In a production system, you might want to trigger recovery or cleanup.
		}

		return true
	})
}

// cleanupRoutine performs periodic cleanup of completed coordination contexts.

func (r *CoordinationController) cleanupRoutine(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute) // Cleanup every 5 minutes

	defer ticker.Stop()

	for {
		select {

		case <-ticker.C:

			r.performCleanup(ctx)

		case <-ctx.Done():

			return

		}
	}
}

// performCleanup cleans up old coordination contexts.

func (r *CoordinationController) performCleanup(ctx context.Context) {
	cutoff := time.Now().Add(-1 * time.Hour) // Clean up contexts older than 1 hour

	r.activeIntents.Range(func(key, value interface{}) bool {
		intentID := key.(string)

		coordCtx := value.(*CoordinationContext)

		if coordCtx.StartTime.Before(cutoff) {

			r.Logger.Info("Cleaning up old coordination context", "intentId", intentID, "age", time.Since(coordCtx.StartTime))

			r.activeIntents.Delete(intentID)

		}

		return true
	})
}

// Default configuration.

func DefaultCoordinationConfig() *CoordinationConfig {
	return &CoordinationConfig{
		MaxConcurrentIntents: 10,

		PhaseTimeout: 300 * time.Second,

		MaxRetries: 3,

		RetryBackoff: 30 * time.Second,

		EnableParallelProcessing: true,

		EnableConflictResolution: true,

		EnableAutoRecovery: true,

		HealthCheckInterval: 60 * time.Second,

		RecoveryTimeout: 120 * time.Second,
	}
}

// convertNetworkTargetComponentsToTargetComponents converts NetworkTargetComponent slice to TargetComponent slice.
func (r *CoordinationController) convertNetworkTargetComponentsToTargetComponents(networkComponents []nephoranv1.NetworkTargetComponent) []nephoranv1.TargetComponent {
	targetComponents := make([]nephoranv1.TargetComponent, len(networkComponents))
	for i, networkComp := range networkComponents {
		targetComponents[i] = nephoranv1.TargetComponent{
			Name: string(networkComp),
			Type: "deployment", // Default type for network components
		}
	}
	return targetComponents
}

// convertORANComponentsToTargetComponents converts ORANComponent slice to TargetComponent slice.

func (r *CoordinationController) convertORANComponentsToTargetComponents(oranComponents []nephoranv1.ORANComponent) []nephoranv1.TargetComponent {
	targetComponents := make([]nephoranv1.TargetComponent, len(oranComponents))

	for i, oranComp := range oranComponents {
		targetComponents[i] = nephoranv1.TargetComponent{
			Name: string(oranComp),

			Type: "deployment", // Default type for O-RAN components

		}
	}

	return targetComponents
}

// ManifestGenerationController handles manifest generation (stub type for compilation).

type ManifestGenerationController struct {
	client.Client

	Logger logr.Logger
}

// GitOpsDeploymentController handles GitOps deployments (stub type for compilation).

type GitOpsDeploymentController struct {
	client.Client

	Logger logr.Logger
}
