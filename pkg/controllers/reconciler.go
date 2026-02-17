package controllers

import (
	"context"
	"fmt"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// Reconciler handles the main reconciliation logic for NetworkIntents.

type Reconciler struct {
	*NetworkIntentReconciler

	llmProcessor *LLMProcessor

	resourcePlanner *ResourcePlanner

	gitopsHandler *GitOpsHandler
}

// Ensure Reconciler implements the required interfaces.

var _ ReconcilerInterface = (*Reconciler)(nil)

// NewReconciler creates a new Reconciler with all required components.

func NewReconciler(r *NetworkIntentReconciler) *Reconciler {
	return &Reconciler{
		NetworkIntentReconciler: r,

		llmProcessor: NewLLMProcessor(r),

		resourcePlanner: NewResourcePlanner(r),

		gitopsHandler: NewGitOpsHandler(r),
	}
}

// Reconcile is the main reconciliation entry point.

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Add request ID for better tracking.

	reqID := fmt.Sprintf("req-%d", time.Now().UnixNano())

	logger = logger.WithValues("request_id", reqID)

	// Check for context cancellation early.

	select {

	case <-ctx.Done():

		logger.Info("Reconciliation cancelled due to context cancellation")

		return ctrl.Result{}, ctx.Err()

	default:

	}

	logger.V(1).Info("Starting reconciliation", "namespace", req.Namespace, "name", req.Name)

	// Fetch the NetworkIntent instance with timeout.

	var networkIntent nephoranv1.NetworkIntent

	if err := r.safeGet(ctx, req.NamespacedName, &networkIntent); err != nil {

		if client.IgnoreNotFound(err) == nil {

			logger.V(1).Info("NetworkIntent not found, likely deleted")

			return ctrl.Result{}, nil

		}

		logger.Error(err, "failed to fetch NetworkIntent")

		return ctrl.Result{RequeueAfter: time.Minute}, fmt.Errorf("failed to fetch NetworkIntent: %w", err)

	}

	// Handle deletion with proper context checking.

	if networkIntent.DeletionTimestamp != nil {

		logger.Info("NetworkIntent is being deleted, handling cleanup")

		return r.handleDeletion(ctx, &networkIntent)

	}

	// Add finalizer if it doesn't exist, with context checking.

	if !containsFinalizer(networkIntent.Finalizers, r.constants.NetworkIntentFinalizer) {

		logger.V(1).Info("Adding finalizer to NetworkIntent")

		networkIntent.Finalizers = append(networkIntent.Finalizers, r.constants.NetworkIntentFinalizer)

		if err := r.safeUpdate(ctx, &networkIntent); err != nil {

			logger.Error(err, "failed to add finalizer")

			return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to add finalizer: %w", err)

		}

		return ctrl.Result{RequeueAfter: time.Second}, nil

	}

	// Initialize processing context.

	processingCtx := &ProcessingContext{
		StartTime: time.Now(),

		ExtractedEntities: make(map[string]interface{}),

		TelecomContext: make(map[string]interface{}),

		DeploymentStatus: make(map[string]interface{}),

		Metrics: make(map[string]float64),
	}

	// Update observed generation.

	networkIntent.Status.ObservedGeneration = networkIntent.Generation

	// Initialize metrics collector.

	metricsCollector := r.deps.GetMetricsCollector()

	if metricsCollector != nil {
		metricsCollector.UpdateNetworkIntentStatus(networkIntent.Name, networkIntent.Namespace,

			r.extractIntentType(networkIntent.Spec.Intent), "processing")
	}

	// Set status to processing in controller-runtime metrics.

	r.metrics.SetStatus(networkIntent.Namespace, networkIntent.Name, "processing", StatusProcessing)

	// Check if already completed to avoid unnecessary work.

	if r.isProcessingComplete(&networkIntent) {

		logger.V(1).Info("NetworkIntent already fully processed, skipping")

		return ctrl.Result{}, nil

	}

	// Execute multi-phase processing pipeline.

	result, err := r.executeProcessingPipeline(ctx, &networkIntent, processingCtx)
	if err != nil {

		logger.Error(err, "processing pipeline failed")

		r.recordFailureEvent(&networkIntent, "ProcessingPipelineFailed", err.Error())

		if metricsCollector != nil {
			metricsCollector.UpdateNetworkIntentStatus(networkIntent.Name, networkIntent.Namespace,

				r.extractIntentType(networkIntent.Spec.Intent), "failed")
		}

		// Set status to failed in controller-runtime metrics.

		r.metrics.SetStatus(networkIntent.Namespace, networkIntent.Name, "failed", StatusFailed)

		// Ensure Phase="Error" is persisted. Fetch latest to avoid ResourceVersion conflicts.
		latest := &nephoranv1.NetworkIntent{}
		if fetchErr := r.safeGet(ctx, req.NamespacedName, latest); fetchErr == nil {
			latest.Status.Phase = nephoranv1.NetworkIntentPhase("Error")
			if statusErr := r.safeStatusUpdate(ctx, latest); statusErr != nil {
				logger.Error(statusErr, "failed to update error phase after pipeline failure")
			}
		}

		return result, err

	}

	if result.RequeueAfter > 0 {

		logger.V(1).Info("Processing pipeline requires requeue", "requeue_after", result.RequeueAfter)

		// Persist any phase changes that the pipeline set (e.g., "Error" on LLM failure).
		// Status updates inside the pipeline may fail due to ResourceVersion conflicts,
		// so we do a reliable fetch-and-update here to ensure the phase is persisted.
		if networkIntent.Status.Phase == nephoranv1.NetworkIntentPhase("Error") {
			latest := &nephoranv1.NetworkIntent{}
			if fetchErr := r.safeGet(ctx, req.NamespacedName, latest); fetchErr == nil {
				latest.Status.Phase = nephoranv1.NetworkIntentPhase("Error")
				if statusErr := r.safeStatusUpdate(ctx, latest); statusErr != nil {
					logger.Error(statusErr, "failed to persist Error phase during requeue")
				}
			}
		}

		return result, nil

	}

	// If the pipeline already set the phase to a terminal state ("Error" or "Processed"),
	// do not override it with "Completed". Return without requeue.
	currentPhase := networkIntent.Status.Phase
	if currentPhase == nephoranv1.NetworkIntentPhase("Error") || currentPhase == nephoranv1.NetworkIntentPhase("Processed") {
		logger.Info("Pipeline finished with terminal phase, not overriding", "phase", string(currentPhase))
		return ctrl.Result{}, nil
	}

	// Final status update.

	if err := r.updatePhase(ctx, &networkIntent, "Completed"); err != nil {

		logger.Error(err, "failed to update completion phase")

		return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update completion phase: %w", err)

	}

	// Set Ready condition to True indicating successful completion of entire pipeline.

	if err := r.setReadyCondition(ctx, &networkIntent, metav1.ConditionTrue, "AllPhasesCompleted", "All processing phases completed successfully and NetworkIntent is ready"); err != nil {
		logger.Error(err, "failed to set ready condition on completion")

		// Don't fail the reconciliation for this.
	}

	// Record success metrics.

	if metricsCollector != nil {

		processingDuration := time.Since(processingCtx.StartTime)

		metricsCollector.RecordNetworkIntentProcessed(

			r.extractIntentType(networkIntent.Spec.Intent), "completed", processingDuration)

		metricsCollector.UpdateNetworkIntentStatus(networkIntent.Name, networkIntent.Namespace,

			r.extractIntentType(networkIntent.Spec.Intent), "completed")

	}

	// Set status to ready in controller-runtime metrics.

	r.metrics.SetStatus(networkIntent.Namespace, networkIntent.Name, "ready", StatusReady)

	r.recordEvent(&networkIntent, "Normal", "ReconciliationCompleted",

		"NetworkIntent successfully processed through all phases and deployed")

	logger.Info("NetworkIntent reconciliation completed successfully",

		"intent", networkIntent.Spec.Intent,

		"phase", networkIntent.Status.Phase,

		"processing_time", time.Since(processingCtx.StartTime),

		"request_id", reqID)

	return ctrl.Result{}, nil
}

// executeProcessingPipeline executes the multi-phase processing pipeline.

func (r *Reconciler) executeProcessingPipeline(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, processingCtx *ProcessingContext) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("pipeline", "multi-phase")

	// Phase 1: LLM Processing with RAG context retrieval.
	// If ENABLE_LLM_INTENT=false, skip LLM and mark as directly processed.

	// When ENABLE_LLM_INTENT=false, skip the entire processing pipeline and
	// directly transition the NetworkIntent to the Processed phase.
	if os.Getenv("ENABLE_LLM_INTENT") == "false" {
		logger.Info("LLM intent processing disabled via ENABLE_LLM_INTENT=false, bypassing all phases")
		bypassCondition := metav1.Condition{
			Type:               "Processed",
			Status:             metav1.ConditionTrue,
			Reason:             "LLMBypassed",
			Message:            "LLM processing disabled; intent accepted directly without LLM enrichment",
			LastTransitionTime: metav1.Now(),
		}
		updateCondition(&networkIntent.Status.Conditions, bypassCondition)
		networkIntent.Status.Phase = nephoranv1.NetworkIntentPhase("Processed")
		if statusErr := r.safeStatusUpdate(ctx, networkIntent); statusErr != nil {
			logger.Error(statusErr, "failed to update status when bypassing LLM")
		}
		return ctrl.Result{}, nil
	}

	if !isConditionTrue(networkIntent.Status.Conditions, "Processed") {

		logger.Info("Starting Phase 1: LLM Processing")

		phaseStartTime := time.Now()

		if err := r.updatePhase(ctx, networkIntent, "LLMProcessing"); err != nil {
			return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update LLM processing phase: %w", err)
		}

		result, err := r.llmProcessor.ProcessLLMPhase(ctx, networkIntent, processingCtx)

		// Record phase duration.

		r.metrics.RecordProcessingDuration(networkIntent.Namespace, networkIntent.Name, "llm_processing", time.Since(phaseStartTime).Seconds())

		if err != nil || result.RequeueAfter > 0 {
			return result, err
		}

	}

	// Phase 2: Resource planning with telecom knowledge.

	if isConditionTrue(networkIntent.Status.Conditions, "Processed") && !isConditionTrue(networkIntent.Status.Conditions, "ResourcesPlanned") {

		logger.Info("Starting Phase 2: Resource Planning")

		phaseStartTime := time.Now()

		if err := r.updatePhase(ctx, networkIntent, "ResourcePlanning"); err != nil {
			return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update resource planning phase: %w", err)
		}

		result, err := r.resourcePlanner.PlanResources(ctx, networkIntent, processingCtx)

		// Record phase duration.

		r.metrics.RecordProcessingDuration(networkIntent.Namespace, networkIntent.Name, "resource_planning", time.Since(phaseStartTime).Seconds())

		if err != nil || result.RequeueAfter > 0 {
			return result, err
		}

	}

	// Phase 3: Deployment manifest generation.

	if isConditionTrue(networkIntent.Status.Conditions, "ResourcesPlanned") && !isConditionTrue(networkIntent.Status.Conditions, "ManifestsGenerated") {

		logger.Info("Starting Phase 3: Manifest Generation")

		phaseStartTime := time.Now()

		if err := r.updatePhase(ctx, networkIntent, "ManifestGeneration"); err != nil {
			return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update manifest generation phase: %w", err)
		}

		result, err := r.resourcePlanner.GenerateManifests(ctx, networkIntent, processingCtx)

		// Record phase duration.

		r.metrics.RecordProcessingDuration(networkIntent.Namespace, networkIntent.Name, "manifest_generation", time.Since(phaseStartTime).Seconds())

		if err != nil || result.RequeueAfter > 0 {
			return result, err
		}

	}

	// Phase 4: GitOps commit and validation.

	if isConditionTrue(networkIntent.Status.Conditions, "ManifestsGenerated") && !isConditionTrue(networkIntent.Status.Conditions, "GitOpsCommitted") {

		logger.Info("Starting Phase 4: GitOps Commit")

		phaseStartTime := time.Now()

		if err := r.updatePhase(ctx, networkIntent, "GitOpsCommit"); err != nil {
			return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update GitOps commit phase: %w", err)
		}

		result, err := r.gitopsHandler.CommitToGitOps(ctx, networkIntent, processingCtx)

		// Record phase duration.

		r.metrics.RecordProcessingDuration(networkIntent.Namespace, networkIntent.Name, "gitops_commit", time.Since(phaseStartTime).Seconds())

		if err != nil || result.RequeueAfter > 0 {
			return result, err
		}

	}

	// Phase 5: Deployment verification.

	if isConditionTrue(networkIntent.Status.Conditions, "GitOpsCommitted") && !isConditionTrue(networkIntent.Status.Conditions, "DeploymentVerified") {

		logger.Info("Starting Phase 5: Deployment Verification")

		phaseStartTime := time.Now()

		if err := r.updatePhase(ctx, networkIntent, "DeploymentVerification"); err != nil {
			return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update deployment verification phase: %w", err)
		}

		result, err := r.gitopsHandler.VerifyDeployment(ctx, networkIntent, processingCtx)

		// Record phase duration.

		r.metrics.RecordProcessingDuration(networkIntent.Namespace, networkIntent.Name, "deployment_verification", time.Since(phaseStartTime).Seconds())

		if err != nil || result.RequeueAfter > 0 {
			return result, err
		}

	}

	logger.Info("All phases completed successfully", "total_time", time.Since(processingCtx.StartTime))

	return ctrl.Result{}, nil
}

// isProcessingComplete checks if all processing phases are complete.

func (r *Reconciler) isProcessingComplete(networkIntent *nephoranv1.NetworkIntent) bool {
	return isConditionTrue(networkIntent.Status.Conditions, "Processed") &&

		isConditionTrue(networkIntent.Status.Conditions, "ResourcesPlanned") &&

		isConditionTrue(networkIntent.Status.Conditions, "ManifestsGenerated") &&

		isConditionTrue(networkIntent.Status.Conditions, "GitOpsCommitted") &&

		isConditionTrue(networkIntent.Status.Conditions, "DeploymentVerified") &&

		networkIntent.Status.Phase == "Completed"
}

// Interface implementation methods.

// ExecuteProcessingPipeline implements ReconcilerInterface (expose existing method).

func (r *Reconciler) ExecuteProcessingPipeline(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, processingCtx *ProcessingContext) (ctrl.Result, error) {
	return r.executeProcessingPipeline(ctx, networkIntent, processingCtx)
}

// IsProcessingComplete implements ReconcilerInterface (expose existing method).

func (r *Reconciler) IsProcessingComplete(networkIntent *nephoranv1.NetworkIntent) bool {
	return r.isProcessingComplete(networkIntent)
}
