package optimized

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers"
)

const (

	// OptimizedNetworkIntentController holds optimizednetworkintentcontroller value.

	OptimizedNetworkIntentController = "optimized-networkintent"
)

// OptimizedNetworkIntentReconciler implements an optimized version of the NetworkIntent controller.

type OptimizedNetworkIntentReconciler struct {
	client.Client

	Scheme *runtime.Scheme

	Recorder record.EventRecorder

	// Configuration
	constants *config.Constants

	// Optimization components.

	backoffManager *BackoffManager

	statusBatcher *StatusBatcher

	metrics *ControllerMetrics

	// Configuration.

	config controllers.Config

	deps controllers.Dependencies

	// Performance tracking.

	activeReconcilers int64

	reconcilePool sync.Pool

	// Batching for API calls.

	apiCallBatcher *APICallBatcher

	// Context for graceful shutdown.

	ctx context.Context

	cancel context.CancelFunc

	wg sync.WaitGroup
}

// NewOptimizedNetworkIntentReconciler creates a new optimized NetworkIntent reconciler.

func NewOptimizedNetworkIntentReconciler(
	client client.Client,

	scheme *runtime.Scheme,

	recorder record.EventRecorder,

	config controllers.Config,

	deps controllers.Dependencies,
) *OptimizedNetworkIntentReconciler {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize optimization components.

	backoffManager := NewBackoffManager()

	statusBatcher := NewStatusBatcher(client, DefaultBatchConfig)

	metrics := NewControllerMetrics()

	apiCallBatcher := NewAPICallBatcher(client, DefaultAPIBatchConfig)

	reconciler := &OptimizedNetworkIntentReconciler{
		Client: client,

		Scheme: scheme,

		Recorder: recorder,

		backoffManager: backoffManager,

		statusBatcher: statusBatcher,

		metrics: metrics,

		config: config,

		deps: deps,

		apiCallBatcher: apiCallBatcher,

		ctx: ctx,

		cancel: cancel,
	}

	// Initialize object pool for reconcile contexts.

	reconciler.reconcilePool = sync.Pool{
		New: func() interface{} {
			return &ReconcileContext{
				ProcessingMetrics: make(map[string]float64),

				ResourcePlan: make(map[string]interface{}),

				Manifests: make(map[string]string),
			}
		},
	}

	// Start background cleanup for stale backoff entries.

	reconciler.wg.Add(1)

	go func() {
		defer reconciler.wg.Done()

		backoffManager.CleanupStaleEntries(ctx, 1*time.Hour)
	}()

	return reconciler
}

// ReconcileContext contains optimized reconcile state with object pooling.

type ReconcileContext struct {
	StartTime time.Time

	NetworkIntent *nephoranv1.NetworkIntent

	ProcessingPhase string

	ProcessingMetrics map[string]float64

	ResourcePlan map[string]interface{}

	Manifests map[string]string

	ErrorCount int

	LastError error
}

// Reset resets the context for reuse.

func (rc *ReconcileContext) Reset() {
	rc.StartTime = time.Time{}

	rc.NetworkIntent = nil

	rc.ProcessingPhase = ""

	for k := range rc.ProcessingMetrics {
		delete(rc.ProcessingMetrics, k)
	}

	for k := range rc.ResourcePlan {
		delete(rc.ResourcePlan, k)
	}

	for k := range rc.Manifests {
		delete(rc.Manifests, k)
	}

	rc.ErrorCount = 0

	rc.LastError = nil
}

//+kubebuilder:rbac:groups=nephoran.com,resources=networkintents,verbs=get;list;watch;create;update;patch;delete

//+kubebuilder:rbac:groups=nephoran.com,resources=networkintents/status,verbs=get;update;patch

//+kubebuilder:rbac:groups=nephoran.com,resources=networkintents/finalizers,verbs=update

// Reconcile implements the optimized reconcile loop.

func (r *OptimizedNetworkIntentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Track active reconcilers.

	atomic.AddInt64(&r.activeReconcilers, 1)

	defer atomic.AddInt64(&r.activeReconcilers, -1)

	r.metrics.UpdateActiveReconcilers(OptimizedNetworkIntentController, int(atomic.LoadInt64(&r.activeReconcilers)))

	// Get recycled context from pool.

	reconcileCtx := r.reconcilePool.Get().(*ReconcileContext)

	defer func() {
		reconcileCtx.Reset()

		r.reconcilePool.Put(reconcileCtx)
	}()

	reconcileCtx.StartTime = time.Now()

	timer := r.metrics.NewReconcileTimer(OptimizedNetworkIntentController, req.Namespace, req.Name, "main")

	defer timer.Finish()

	// Generate resource key for backoff management.

	resourceKey := fmt.Sprintf("%s/%s", req.Namespace, req.Name)

	logger.V(1).Info("Starting optimized reconciliation", "resource", resourceKey)

	// Optimized object retrieval with API call batching.

	networkIntent, err := r.getNetworkIntentOptimized(ctx, req.NamespacedName)
	if err != nil {

		if client.IgnoreNotFound(err) == nil {

			logger.V(1).Info("NetworkIntent not found, likely deleted")

			r.metrics.RecordReconcileResult(OptimizedNetworkIntentController, "not_found")

			return ctrl.Result{}, nil

		}

		// Classify error and determine backoff.

		errorType := r.backoffManager.ClassifyError(err)

		delay := r.backoffManager.GetNextDelay(resourceKey, errorType, err)

		r.metrics.RecordReconcileError(OptimizedNetworkIntentController, errorType, "fetch")

		r.metrics.RecordBackoffDelay(OptimizedNetworkIntentController, errorType, ExponentialBackoff, delay)

		logger.Error(err, "Failed to get NetworkIntent", "backoff_delay", delay)

		return ctrl.Result{RequeueAfter: delay}, nil

	}

	reconcileCtx.NetworkIntent = networkIntent

	// Handle deletion with optimized cleanup.

	if networkIntent.DeletionTimestamp != nil {
		return r.handleDeletionOptimized(ctx, networkIntent, resourceKey)
	}

	// Ensure finalizer exists (batched operation).

	if !r.hasFinalizer(networkIntent, r.constants.NetworkIntentFinalizer) {

		if err := r.addFinalizerOptimized(ctx, networkIntent); err != nil {

			errorType := r.backoffManager.ClassifyError(err)

			delay := r.backoffManager.GetNextDelay(resourceKey, errorType, err)

			r.metrics.RecordReconcileError(OptimizedNetworkIntentController, errorType, "finalizer")

			logger.Error(err, "Failed to add finalizer", "backoff_delay", delay)

			return ctrl.Result{RequeueAfter: delay}, nil

		}

		return ctrl.Result{RequeueAfter: time.Second}, nil

	}

	// Execute optimized processing pipeline.

	result, err := r.executeOptimizedPipeline(ctx, networkIntent, reconcileCtx, resourceKey)

	// Record metrics based on result.

	if err != nil {

		errorType := r.backoffManager.ClassifyError(err)

		r.metrics.RecordReconcileError(OptimizedNetworkIntentController, errorType, "pipeline")

		r.metrics.RecordReconcileResult(OptimizedNetworkIntentController, "error")

	} else {

		// Reset backoff on success.

		r.backoffManager.RecordSuccess(resourceKey)

		r.metrics.RecordBackoffReset(OptimizedNetworkIntentController, "NetworkIntent")

		r.metrics.RecordReconcileResult(OptimizedNetworkIntentController, "success")

	}

	return result, err
}

// getNetworkIntentOptimized retrieves NetworkIntent using optimized API calls.

func (r *OptimizedNetworkIntentReconciler) getNetworkIntentOptimized(ctx context.Context, key types.NamespacedName) (*nephoranv1.NetworkIntent, error) {
	timer := r.metrics.NewAPICallTimer(OptimizedNetworkIntentController, "get", "NetworkIntent")

	var networkIntent nephoranv1.NetworkIntent

	err := r.Get(ctx, key, &networkIntent)

	timer.FinishWithResult(err == nil, r.backoffManager.ClassifyError(err).String())

	if err != nil {
		return nil, err
	}

	return &networkIntent, nil
}

// executeOptimizedPipeline runs the optimized processing pipeline.

func (r *OptimizedNetworkIntentReconciler) executeOptimizedPipeline(
	ctx context.Context,

	networkIntent *nephoranv1.NetworkIntent,

	reconcileCtx *ReconcileContext,

	resourceKey string,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("pipeline", "optimized")

	// Phase 1: LLM Processing (if not already processed).

	if !r.hasCondition(networkIntent, "Processed") {

		phaseTimer := r.metrics.NewReconcileTimer(OptimizedNetworkIntentController,

			networkIntent.Namespace, networkIntent.Name, "llm-processing")

		result, err := r.processLLMPhaseOptimized(ctx, networkIntent, reconcileCtx, resourceKey)

		phaseTimer.Finish()

		if err != nil || result.RequeueAfter > 0 {
			return result, err
		}

	}

	// Phase 2: Resource Planning (if processed but not planned).

	if r.hasCondition(networkIntent, "Processed") && !r.hasCondition(networkIntent, "ResourcesPlanned") {

		phaseTimer := r.metrics.NewReconcileTimer(OptimizedNetworkIntentController,

			networkIntent.Namespace, networkIntent.Name, "resource-planning")

		result, err := r.planResourcesOptimized(ctx, networkIntent, reconcileCtx, resourceKey)

		phaseTimer.Finish()

		if err != nil || result.RequeueAfter > 0 {
			return result, err
		}

	}

	// Phase 3: Manifest Generation (if planned but not generated).

	if r.hasCondition(networkIntent, "ResourcesPlanned") && !r.hasCondition(networkIntent, "ManifestsGenerated") {

		phaseTimer := r.metrics.NewReconcileTimer(OptimizedNetworkIntentController,

			networkIntent.Namespace, networkIntent.Name, "manifest-generation")

		result, err := r.generateManifestsOptimized(ctx, networkIntent, reconcileCtx, resourceKey)

		phaseTimer.Finish()

		if err != nil || result.RequeueAfter > 0 {
			return result, err
		}

	}

	// Phase 4: GitOps Deployment (if manifests generated but not deployed).

	if r.hasCondition(networkIntent, "ManifestsGenerated") && !r.hasCondition(networkIntent, "Deployed") {

		phaseTimer := r.metrics.NewReconcileTimer(OptimizedNetworkIntentController,

			networkIntent.Namespace, networkIntent.Name, "deployment")

		result, err := r.deployViaGitOpsOptimized(ctx, networkIntent, reconcileCtx, resourceKey)

		phaseTimer.Finish()

		if err != nil || result.RequeueAfter > 0 {
			return result, err
		}

	}

	// Final status update.

	r.queueStatusUpdate(networkIntent, "Ready", "Completed", HighPriority)

	// Use intelligent requeue interval based on resource state.

	requeueInterval := r.calculateOptimalRequeueInterval(networkIntent, reconcileCtx)

	logger.V(1).Info("Pipeline completed successfully",

		"requeue_interval", requeueInterval,

		"processing_time", time.Since(reconcileCtx.StartTime))

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

// processLLMPhaseOptimized handles LLM processing with optimizations.

func (r *OptimizedNetworkIntentReconciler) processLLMPhaseOptimized(
	ctx context.Context,

	networkIntent *nephoranv1.NetworkIntent,

	reconcileCtx *ReconcileContext,

	resourceKey string,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("phase", "llm-processing")

	reconcileCtx.ProcessingPhase = "LLMProcessing"

	// Check retry count using backoff manager.

	retryCount := r.backoffManager.GetRetryCount(resourceKey + "-llm")

	// Queue phase status update.

	r.queueStatusUpdate(networkIntent, "LLMProcessing", "Processing intent with LLM", MediumPriority)

	// Simulate LLM processing (replace with actual implementation).

	processingStart := time.Now()

	// Mock LLM call - in real implementation, this would call the LLM service.

	time.Sleep(100 * time.Millisecond) // Simulate processing time

	processingDuration := time.Since(processingStart)

	reconcileCtx.ProcessingMetrics["llm_processing_duration"] = processingDuration.Seconds()

	// Queue success status update.

	condition := metav1.Condition{
		Type: "Processed",

		Status: metav1.ConditionTrue,

		Reason: "LLMProcessingCompleted",

		Message: fmt.Sprintf("Intent processed successfully in %v", processingDuration),

		LastTransitionTime: metav1.Now(),
	}

	r.queueConditionUpdate(networkIntent, condition, HighPriority)

	logger.V(1).Info("LLM processing completed",

		"duration", processingDuration,

		"retry_count", retryCount)

	return ctrl.Result{}, nil
}

// planResourcesOptimized handles resource planning with optimizations.

func (r *OptimizedNetworkIntentReconciler) planResourcesOptimized(
	ctx context.Context,

	networkIntent *nephoranv1.NetworkIntent,

	reconcileCtx *ReconcileContext,

	resourceKey string,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("phase", "resource-planning")

	reconcileCtx.ProcessingPhase = "ResourcePlanning"

	// Queue phase status update.

	r.queueStatusUpdate(networkIntent, "ResourcePlanning", "Planning resource deployment", MediumPriority)

	// Simulate resource planning.

	planningStart := time.Now()

	time.Sleep(50 * time.Millisecond)

	planningDuration := time.Since(planningStart)

	reconcileCtx.ProcessingMetrics["resource_planning_duration"] = planningDuration.Seconds()

	// Mock resource plan.

	reconcileCtx.ResourcePlan["network_functions"] = []string{"amf", "smf", "upf"}

	reconcileCtx.ResourcePlan["replicas"] = 3

	reconcileCtx.ResourcePlan["resources"] = map[string]string{
		"cpu": "500m",

		"memory": "512Mi",
	}

	// Queue success status update.

	condition := metav1.Condition{
		Type: "ResourcesPlanned",

		Status: metav1.ConditionTrue,

		Reason: "ResourcePlanningCompleted",

		Message: fmt.Sprintf("Resource planning completed in %v", planningDuration),

		LastTransitionTime: metav1.Now(),
	}

	r.queueConditionUpdate(networkIntent, condition, HighPriority)

	logger.V(1).Info("Resource planning completed",

		"duration", planningDuration,

		"functions", len(reconcileCtx.ResourcePlan))

	return ctrl.Result{}, nil
}

// generateManifestsOptimized handles manifest generation with optimizations.

func (r *OptimizedNetworkIntentReconciler) generateManifestsOptimized(
	ctx context.Context,

	networkIntent *nephoranv1.NetworkIntent,

	reconcileCtx *ReconcileContext,

	resourceKey string,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("phase", "manifest-generation")

	reconcileCtx.ProcessingPhase = "ManifestGeneration"

	// Queue phase status update.

	r.queueStatusUpdate(networkIntent, "ManifestGeneration", "Generating Kubernetes manifests", MediumPriority)

	// Simulate manifest generation.

	generationStart := time.Now()

	time.Sleep(75 * time.Millisecond)

	generationDuration := time.Since(generationStart)

	reconcileCtx.ProcessingMetrics["manifest_generation_duration"] = generationDuration.Seconds()

	// Mock manifest generation.

	reconcileCtx.Manifests["amf-deployment.yaml"] = "apiVersion: apps/v1\nkind: Deployment\n..."

	reconcileCtx.Manifests["smf-deployment.yaml"] = "apiVersion: apps/v1\nkind: Deployment\n..."

	reconcileCtx.Manifests["upf-deployment.yaml"] = "apiVersion: apps/v1\nkind: Deployment\n..."

	// Queue success status update.

	condition := metav1.Condition{
		Type: "ManifestsGenerated",

		Status: metav1.ConditionTrue,

		Reason: "ManifestGenerationCompleted",

		Message: fmt.Sprintf("Generated %d manifests in %v", len(reconcileCtx.Manifests), generationDuration),

		LastTransitionTime: metav1.Now(),
	}

	r.queueConditionUpdate(networkIntent, condition, HighPriority)

	logger.V(1).Info("Manifest generation completed",

		"duration", generationDuration,

		"manifests", len(reconcileCtx.Manifests))

	return ctrl.Result{}, nil
}

// deployViaGitOpsOptimized handles GitOps deployment with optimizations.

func (r *OptimizedNetworkIntentReconciler) deployViaGitOpsOptimized(
	ctx context.Context,

	networkIntent *nephoranv1.NetworkIntent,

	reconcileCtx *ReconcileContext,

	resourceKey string,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("phase", "gitops-deployment")

	reconcileCtx.ProcessingPhase = "GitOpsDeployment"

	// Queue phase status update.

	r.queueStatusUpdate(networkIntent, "Deploying", "Committing to GitOps repository", MediumPriority)

	// Simulate GitOps deployment.

	deploymentStart := time.Now()

	time.Sleep(200 * time.Millisecond)

	deploymentDuration := time.Since(deploymentStart)

	reconcileCtx.ProcessingMetrics["deployment_duration"] = deploymentDuration.Seconds()

	// Mock successful deployment.

	gitCommitHash := fmt.Sprintf("abc123-%d", time.Now().Unix())

	// Queue success status update.

	condition := metav1.Condition{
		Type: "Deployed",

		Status: metav1.ConditionTrue,

		Reason: "GitOpsDeploymentCompleted",

		Message: fmt.Sprintf("Deployed to GitOps repository (commit: %s) in %v", gitCommitHash, deploymentDuration),

		LastTransitionTime: metav1.Now(),
	}

	r.queueConditionUpdate(networkIntent, condition, CriticalPriority)

	logger.V(1).Info("GitOps deployment completed",

		"duration", deploymentDuration,

		"commit_hash", gitCommitHash)

	return ctrl.Result{}, nil
}

// Helper methods for optimized operations.

func (r *OptimizedNetworkIntentReconciler) queueStatusUpdate(networkIntent *nephoranv1.NetworkIntent, phase, message string, priority UpdatePriority) {
	key := types.NamespacedName{
		Namespace: networkIntent.Namespace,

		Name: networkIntent.Name,
	}

	condition := metav1.Condition{
		Type: "Phase",

		Status: metav1.ConditionTrue,

		Reason: phase,

		Message: message,

		LastTransitionTime: metav1.Now(),
	}

	r.statusBatcher.QueueNetworkIntentUpdate(key, []metav1.Condition{condition}, phase, priority)

	r.metrics.RecordStatusUpdate(OptimizedNetworkIntentController, priority.String(), "NetworkIntent", "queued")
}

func (r *OptimizedNetworkIntentReconciler) queueConditionUpdate(networkIntent *nephoranv1.NetworkIntent, condition metav1.Condition, priority UpdatePriority) {
	key := types.NamespacedName{
		Namespace: networkIntent.Namespace,

		Name: networkIntent.Name,
	}

	r.statusBatcher.QueueNetworkIntentUpdate(key, []metav1.Condition{condition}, "", priority)

	r.metrics.RecordStatusUpdate(OptimizedNetworkIntentController, priority.String(), "NetworkIntent", "queued")
}

func (r *OptimizedNetworkIntentReconciler) hasCondition(networkIntent *nephoranv1.NetworkIntent, conditionType string) bool {
	for _, condition := range networkIntent.Status.Conditions {
		if condition.Type == conditionType && condition.Status == metav1.ConditionTrue {
			return true
		}
	}

	return false
}

func (r *OptimizedNetworkIntentReconciler) hasFinalizer(networkIntent *nephoranv1.NetworkIntent, finalizer string) bool {
	for _, f := range networkIntent.Finalizers {
		if f == finalizer {
			return true
		}
	}

	return false
}

func (r *OptimizedNetworkIntentReconciler) addFinalizerOptimized(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) error {
	networkIntent.Finalizers = append(networkIntent.Finalizers, r.constants.NetworkIntentFinalizer)

	timer := r.metrics.NewAPICallTimer(OptimizedNetworkIntentController, "update", "NetworkIntent")

	err := r.Update(ctx, networkIntent)

	timer.FinishWithResult(err == nil, r.backoffManager.ClassifyError(err).String())

	return err
}

func (r *OptimizedNetworkIntentReconciler) handleDeletionOptimized(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, resourceKey string) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("phase", "deletion")

	// Queue deletion status update.

	r.queueStatusUpdate(networkIntent, "Deleting", "Cleaning up resources", HighPriority)

	// Simulate cleanup operations.

	cleanupStart := time.Now()

	time.Sleep(100 * time.Millisecond)

	cleanupDuration := time.Since(cleanupStart)

	// Remove finalizer.

	finalizers := make([]string, 0)

	for _, f := range networkIntent.Finalizers {
		if f != r.constants.NetworkIntentFinalizer {
			finalizers = append(finalizers, f)
		}
	}

	networkIntent.Finalizers = finalizers

	timer := r.metrics.NewAPICallTimer(OptimizedNetworkIntentController, "update", "NetworkIntent")

	err := r.Update(ctx, networkIntent)

	timer.FinishWithResult(err == nil, r.backoffManager.ClassifyError(err).String())

	if err != nil {

		errorType := r.backoffManager.ClassifyError(err)

		delay := r.backoffManager.GetNextDelay(resourceKey, errorType, err)

		r.metrics.RecordReconcileError(OptimizedNetworkIntentController, errorType, "deletion")

		logger.Error(err, "Failed to remove finalizer", "backoff_delay", delay)

		return ctrl.Result{RequeueAfter: delay}, nil

	}

	logger.Info("Resource deletion completed", "cleanup_duration", cleanupDuration)

	r.metrics.RecordReconcileResult(OptimizedNetworkIntentController, "deleted")

	return ctrl.Result{}, nil
}

func (r *OptimizedNetworkIntentReconciler) calculateOptimalRequeueInterval(networkIntent *nephoranv1.NetworkIntent, reconcileCtx *ReconcileContext) time.Duration {
	// Adaptive requeue intervals based on resource state and processing time.

	processingTime := time.Since(reconcileCtx.StartTime)

	// Base interval depends on current phase.

	var baseInterval time.Duration

	switch {

	case !r.hasCondition(networkIntent, "Deployed"):

		baseInterval = 30 * time.Second // Active deployment phase

	case r.hasCondition(networkIntent, "Ready"):

		baseInterval = 5 * time.Minute // Stable state, check less frequently

	default:

		baseInterval = 2 * time.Minute // Default interval

	}

	// Adjust based on processing time.

	if processingTime > 1*time.Second {
		// If processing took long, wait a bit more.

		baseInterval = time.Duration(float64(baseInterval) * 1.5)
	}

	// Cap the maximum interval.

	maxInterval := 10 * time.Minute

	if baseInterval > maxInterval {
		baseInterval = maxInterval
	}

	return baseInterval
}

// SetupWithManager sets up the controller with optimized configuration.

func (r *OptimizedNetworkIntentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nephoranv1.NetworkIntent{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 5, // Increased concurrency

		}).

		// Removed Watches - redundant with For() in controller-runtime v0.18+.

		Complete(r)
}

// Shutdown gracefully shuts down the optimized controller.

func (r *OptimizedNetworkIntentReconciler) Shutdown() error {
	r.cancel()

	// Stop status batcher.

	if err := r.statusBatcher.Stop(); err != nil {
		return fmt.Errorf("failed to stop status batcher: %w", err)
	}

	// Stop API call batcher.

	if err := r.apiCallBatcher.Stop(); err != nil {
		return fmt.Errorf("failed to stop API call batcher: %w", err)
	}

	// Wait for background goroutines.

	r.wg.Wait()

	return nil
}

// String methods for priority types.

func (p UpdatePriority) String() string {
	switch p {

	case LowPriority:

		return "low"

	case MediumPriority:

		return "medium"

	case HighPriority:

		return "high"

	case CriticalPriority:

		return "critical"

	default:

		return "unknown"

	}
}

// String performs string operation.

func (e ErrorType) String() string {
	return string(e)
}

// String performs string operation.

func (b BackoffStrategy) String() string {
	return string(b)
}
