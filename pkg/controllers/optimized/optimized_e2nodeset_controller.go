package optimized

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

const (
	OptimizedE2NodeSetController = "optimized-e2nodeset"
	E2NodeSetFinalizer           = "nephoran.com/e2nodeset-finalizer"

	// ConfigMap labels
	E2NodeSetLabelKey   = "nephoran.com/e2-nodeset"
	E2NodeAppLabelKey   = "app"
	E2NodeAppLabelValue = "e2-node-simulator"
	E2NodeIDLabelKey    = "nephoran.com/node-id"
	E2NodeIndexLabelKey = "nephoran.com/node-index"
)

// OptimizedE2NodeSetReconciler implements an optimized version of the E2NodeSet controller
type OptimizedE2NodeSetReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	// Optimization components
	backoffManager *BackoffManager
	statusBatcher  *StatusBatcher
	metrics        *ControllerMetrics

	// Performance tracking
	activeReconcilers int64
	reconcilePool     sync.Pool
	configMapCache    sync.Map // Cache for ConfigMap operations

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// E2NodeSetReconcileContext contains optimized reconcile state
type E2NodeSetReconcileContext struct {
	StartTime          time.Time
	E2NodeSet          *nephoranv1.E2NodeSet
	ExistingConfigMaps []*corev1.ConfigMap
	ProcessingMetrics  map[string]float64
	NodesCreated       int
	NodesUpdated       int
	NodesDeleted       int
	ErrorCount         int
}

// Reset resets the context for reuse
func (ctx *E2NodeSetReconcileContext) Reset() {
	ctx.StartTime = time.Time{}
	ctx.E2NodeSet = nil
	ctx.ExistingConfigMaps = ctx.ExistingConfigMaps[:0]
	for k := range ctx.ProcessingMetrics {
		delete(ctx.ProcessingMetrics, k)
	}
	ctx.NodesCreated = 0
	ctx.NodesUpdated = 0
	ctx.NodesDeleted = 0
	ctx.ErrorCount = 0
}

// NewOptimizedE2NodeSetReconciler creates a new optimized E2NodeSet reconciler
func NewOptimizedE2NodeSetReconciler(
	client client.Client,
	scheme *runtime.Scheme,
	recorder record.EventRecorder,
) *OptimizedE2NodeSetReconciler {

	ctx, cancel := context.WithCancel(context.Background())

	// Initialize optimization components
	backoffManager := NewBackoffManager()
	statusBatcher := NewStatusBatcher(client, DefaultBatchConfig)
	metrics := NewControllerMetrics()

	reconciler := &OptimizedE2NodeSetReconciler{
		Client:         client,
		Scheme:         scheme,
		Recorder:       recorder,
		backoffManager: backoffManager,
		statusBatcher:  statusBatcher,
		metrics:        metrics,
		ctx:            ctx,
		cancel:         cancel,
	}

	// Initialize object pool
	reconciler.reconcilePool = sync.Pool{
		New: func() interface{} {
			return &E2NodeSetReconcileContext{
				ProcessingMetrics:  make(map[string]float64),
				ExistingConfigMaps: make([]*corev1.ConfigMap, 0, 10),
			}
		},
	}

	// Start cache cleanup
	reconciler.wg.Add(1)
	go reconciler.cleanupCache()

	// Start stale backoff cleanup
	reconciler.wg.Add(1)
	go func() {
		defer reconciler.wg.Done()
		backoffManager.CleanupStaleEntries(ctx, 30*time.Minute)
	}()

	return reconciler
}

//+kubebuilder:rbac:groups=nephoran.com,resources=e2nodesets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nephoran.com,resources=e2nodesets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nephoran.com,resources=e2nodesets/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile implements the optimized reconcile loop
func (r *OptimizedE2NodeSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Track active reconcilers
	atomic.AddInt64(&r.activeReconcilers, 1)
	defer atomic.AddInt64(&r.activeReconcilers, -1)
	r.metrics.UpdateActiveReconcilers(OptimizedE2NodeSetController, int(atomic.LoadInt64(&r.activeReconcilers)))

	// Get recycled context from pool
	reconcileCtx := r.reconcilePool.Get().(*E2NodeSetReconcileContext)
	defer func() {
		reconcileCtx.Reset()
		r.reconcilePool.Put(reconcileCtx)
	}()

	reconcileCtx.StartTime = time.Now()
	timer := r.metrics.NewReconcileTimer(OptimizedE2NodeSetController, req.Namespace, req.Name, "main")
	defer timer.Finish()

	// Generate resource key for backoff management
	resourceKey := fmt.Sprintf("%s/%s", req.Namespace, req.Name)

	logger.V(1).Info("Starting optimized E2NodeSet reconciliation", "resource", resourceKey)

	// Optimized object retrieval
	e2nodeSet, err := r.getE2NodeSetOptimized(ctx, req.NamespacedName)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.V(1).Info("E2NodeSet not found, likely deleted")
			r.metrics.RecordReconcileResult(OptimizedE2NodeSetController, "not_found")
			return ctrl.Result{}, nil
		}

		// Classify error and determine backoff
		errorType := r.backoffManager.ClassifyError(err)
		delay := r.backoffManager.GetNextDelay(resourceKey, errorType, err)

		r.metrics.RecordReconcileError(OptimizedE2NodeSetController, errorType, "fetch")
		r.metrics.RecordBackoffDelay(OptimizedE2NodeSetController, errorType, ExponentialBackoff, delay)

		logger.Error(err, "Failed to get E2NodeSet", "backoff_delay", delay)
		return ctrl.Result{RequeueAfter: delay}, nil
	}

	reconcileCtx.E2NodeSet = e2nodeSet

	// Handle deletion
	if e2nodeSet.DeletionTimestamp != nil {
		return r.handleDeletionOptimized(ctx, e2nodeSet, reconcileCtx, resourceKey)
	}

	// Ensure finalizer exists
	if !r.hasFinalizer(e2nodeSet, E2NodeSetFinalizer) {
		if err := r.addFinalizerOptimized(ctx, e2nodeSet); err != nil {
			errorType := r.backoffManager.ClassifyError(err)
			delay := r.backoffManager.GetNextDelay(resourceKey, errorType, err)

			r.metrics.RecordReconcileError(OptimizedE2NodeSetController, errorType, "finalizer")
			logger.Error(err, "Failed to add finalizer", "backoff_delay", delay)
			return ctrl.Result{RequeueAfter: delay}, nil
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Execute optimized reconciliation
	result, err := r.reconcileNodesOptimized(ctx, e2nodeSet, reconcileCtx, resourceKey)

	// Record metrics based on result
	if err != nil {
		errorType := r.backoffManager.ClassifyError(err)
		r.metrics.RecordReconcileError(OptimizedE2NodeSetController, errorType, "reconcile")
		r.metrics.RecordReconcileResult(OptimizedE2NodeSetController, "error")
	} else {
		// Reset backoff on success
		r.backoffManager.RecordSuccess(resourceKey)
		r.metrics.RecordBackoffReset(OptimizedE2NodeSetController, "E2NodeSet")
		r.metrics.RecordReconcileResult(OptimizedE2NodeSetController, "success")
	}

	return result, err
}

// getE2NodeSetOptimized retrieves E2NodeSet using optimized API calls
func (r *OptimizedE2NodeSetReconciler) getE2NodeSetOptimized(ctx context.Context, key types.NamespacedName) (*nephoranv1.E2NodeSet, error) {
	timer := r.metrics.NewApiCallTimer(OptimizedE2NodeSetController, "get", "E2NodeSet")

	var e2nodeSet nephoranv1.E2NodeSet
	err := r.Get(ctx, key, &e2nodeSet)

	timer.FinishWithResult(err == nil, r.backoffManager.ClassifyError(err).String())

	if err != nil {
		return nil, err
	}

	return &e2nodeSet, nil
}

// reconcileNodesOptimized performs optimized node reconciliation
func (r *OptimizedE2NodeSetReconciler) reconcileNodesOptimized(
	ctx context.Context,
	e2nodeSet *nephoranv1.E2NodeSet,
	reconcileCtx *E2NodeSetReconcileContext,
	resourceKey string,
) (ctrl.Result, error) {

	logger := log.FromContext(ctx).WithValues("phase", "node-reconciliation")

	// Queue initial status update
	r.queueE2NodeSetStatusUpdate(e2nodeSet, 0, e2nodeSet.Spec.Replicas, "Reconciling", MediumPriority)

	// Get existing ConfigMaps efficiently
	existingConfigMaps, err := r.getExistingConfigMapsOptimized(ctx, e2nodeSet)
	if err != nil {
		errorType := r.backoffManager.ClassifyError(err)
		delay := r.backoffManager.GetNextDelay(resourceKey, errorType, err)

		logger.Error(err, "Failed to get existing ConfigMaps", "backoff_delay", delay)
		return ctrl.Result{RequeueAfter: delay}, err
	}

	reconcileCtx.ExistingConfigMaps = existingConfigMaps

	// Calculate desired vs actual state
	desiredReplicas := int(e2nodeSet.Spec.Replicas)
	existingCount := len(existingConfigMaps)

	reconcileStart := time.Now()

	var result ctrl.Result
	var reconcileErr error

	if existingCount < desiredReplicas {
		// Need to create more nodes
		result, reconcileErr = r.scaleUpOptimized(ctx, e2nodeSet, reconcileCtx, desiredReplicas-existingCount)
	} else if existingCount > desiredReplicas {
		// Need to delete excess nodes
		result, reconcileErr = r.scaleDownOptimized(ctx, e2nodeSet, reconcileCtx, existingCount-desiredReplicas)
	} else {
		// Update existing nodes if needed
		result, reconcileErr = r.updateExistingNodesOptimized(ctx, e2nodeSet, reconcileCtx)
	}

	reconcileDuration := time.Since(reconcileStart)
	reconcileCtx.ProcessingMetrics["node_reconciliation_duration"] = reconcileDuration.Seconds()

	if reconcileErr != nil {
		return result, reconcileErr
	}

	// Update final status with batched operation
	readyReplicas := int32(len(reconcileCtx.ExistingConfigMaps) - reconcileCtx.ErrorCount)
	conditions := []metav1.Condition{
		{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			Reason:             "NodesReconciled",
			Message:            fmt.Sprintf("Successfully reconciled %d nodes", readyReplicas),
			LastTransitionTime: metav1.Now(),
		},
	}

	r.statusBatcher.QueueE2NodeSetUpdate(
		types.NamespacedName{Namespace: e2nodeSet.Namespace, Name: e2nodeSet.Name},
		readyReplicas,
		e2nodeSet.Spec.Replicas,
		conditions,
		HighPriority,
	)

	// Calculate optimal requeue interval
	requeueInterval := r.calculateOptimalRequeueInterval(e2nodeSet, reconcileCtx)

	logger.V(1).Info("Node reconciliation completed",
		"created", reconcileCtx.NodesCreated,
		"updated", reconcileCtx.NodesUpdated,
		"deleted", reconcileCtx.NodesDeleted,
		"errors", reconcileCtx.ErrorCount,
		"duration", reconcileDuration,
		"requeue_interval", requeueInterval)

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

// getExistingConfigMapsOptimized efficiently retrieves existing ConfigMaps
func (r *OptimizedE2NodeSetReconciler) getExistingConfigMapsOptimized(ctx context.Context, e2nodeSet *nephoranv1.E2NodeSet) ([]*corev1.ConfigMap, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s/%s", e2nodeSet.Namespace, e2nodeSet.Name)
	if cached, ok := r.configMapCache.Load(cacheKey); ok {
		if cacheEntry, ok := cached.(configMapCacheEntry); ok {
			if time.Since(cacheEntry.timestamp) < 30*time.Second {
				return cacheEntry.configMaps, nil
			}
		}
	}

	timer := r.metrics.NewApiCallTimer(OptimizedE2NodeSetController, "list", "ConfigMap")

	var configMapList corev1.ConfigMapList
	err := r.List(ctx, &configMapList,
		client.InNamespace(e2nodeSet.Namespace),
		client.MatchingLabels{
			E2NodeSetLabelKey: e2nodeSet.Name,
			E2NodeAppLabelKey: E2NodeAppLabelValue,
		})

	timer.FinishWithResult(err == nil, r.backoffManager.ClassifyError(err).String())

	if err != nil {
		return nil, err
	}

	// Convert to slice of pointers
	configMaps := make([]*corev1.ConfigMap, len(configMapList.Items))
	for i := range configMapList.Items {
		configMaps[i] = &configMapList.Items[i]
	}

	// Update cache
	r.configMapCache.Store(cacheKey, configMapCacheEntry{
		configMaps: configMaps,
		timestamp:  time.Now(),
	})

	return configMaps, nil
}

// scaleUpOptimized creates new E2 nodes efficiently
func (r *OptimizedE2NodeSetReconciler) scaleUpOptimized(
	ctx context.Context,
	e2nodeSet *nephoranv1.E2NodeSet,
	reconcileCtx *E2NodeSetReconcileContext,
	nodesToCreate int,
) (ctrl.Result, error) {

	logger := log.FromContext(ctx).WithValues("phase", "scale-up", "nodes_to_create", nodesToCreate)

	// Update status to show scaling in progress
	r.queueE2NodeSetStatusUpdate(e2nodeSet, 0, e2nodeSet.Spec.Replicas, "ScalingUp", HighPriority)

	// Create nodes in batches to avoid overwhelming the API server
	batchSize := 3
	successCount := 0
	errorCount := 0

	for i := 0; i < nodesToCreate; i += batchSize {
		end := i + batchSize
		if end > nodesToCreate {
			end = nodesToCreate
		}

		// Create batch of nodes
		for j := i; j < end; j++ {
			nodeIndex := len(reconcileCtx.ExistingConfigMaps) + j
			if err := r.createE2NodeOptimized(ctx, e2nodeSet, nodeIndex); err != nil {
				logger.Error(err, "Failed to create E2 node", "index", nodeIndex)
				errorCount++
				reconcileCtx.ErrorCount++
			} else {
				successCount++
				reconcileCtx.NodesCreated++
			}
		}

		// Short pause between batches to prevent API rate limiting
		time.Sleep(100 * time.Millisecond)
	}

	// Update metrics
	r.metrics.RecordStatusUpdate(OptimizedE2NodeSetController, "high", "E2NodeSet", "scaled_up")

	logger.Info("Scale up completed",
		"successful", successCount,
		"failed", errorCount)

	if errorCount > 0 {
		// Some failures occurred, use backoff
		resourceKey := fmt.Sprintf("%s/%s", e2nodeSet.Namespace, e2nodeSet.Name)
		delay := r.backoffManager.GetNextDelay(resourceKey, ResourceError, fmt.Errorf("failed to create %d nodes", errorCount))

		return ctrl.Result{RequeueAfter: delay}, fmt.Errorf("failed to create %d out of %d nodes", errorCount, nodesToCreate)
	}

	return ctrl.Result{}, nil
}

// scaleDownOptimized removes excess E2 nodes efficiently
func (r *OptimizedE2NodeSetReconciler) scaleDownOptimized(
	ctx context.Context,
	e2nodeSet *nephoranv1.E2NodeSet,
	reconcileCtx *E2NodeSetReconcileContext,
	nodesToDelete int,
) (ctrl.Result, error) {

	logger := log.FromContext(ctx).WithValues("phase", "scale-down", "nodes_to_delete", nodesToDelete)

	// Update status to show scaling in progress
	r.queueE2NodeSetStatusUpdate(e2nodeSet, 0, e2nodeSet.Spec.Replicas, "ScalingDown", HighPriority)

	// Delete the highest-indexed nodes first (LIFO)
	existingConfigMaps := reconcileCtx.ExistingConfigMaps
	nodesToDeleteSlice := existingConfigMaps[len(existingConfigMaps)-nodesToDelete:]

	successCount := 0
	errorCount := 0

	// Delete in batches
	batchSize := 3
	for i := 0; i < len(nodesToDeleteSlice); i += batchSize {
		end := i + batchSize
		if end > len(nodesToDeleteSlice) {
			end = len(nodesToDeleteSlice)
		}

		// Delete batch
		for j := i; j < end; j++ {
			configMap := nodesToDeleteSlice[j]
			if err := r.deleteE2NodeOptimized(ctx, configMap); err != nil {
				logger.Error(err, "Failed to delete E2 node", "name", configMap.Name)
				errorCount++
				reconcileCtx.ErrorCount++
			} else {
				successCount++
				reconcileCtx.NodesDeleted++
			}
		}

		// Short pause between batches
		time.Sleep(100 * time.Millisecond)
	}

	logger.Info("Scale down completed",
		"successful", successCount,
		"failed", errorCount)

	if errorCount > 0 {
		resourceKey := fmt.Sprintf("%s/%s", e2nodeSet.Namespace, e2nodeSet.Name)
		delay := r.backoffManager.GetNextDelay(resourceKey, ResourceError, fmt.Errorf("failed to delete %d nodes", errorCount))

		return ctrl.Result{RequeueAfter: delay}, fmt.Errorf("failed to delete %d out of %d nodes", errorCount, nodesToDelete)
	}

	return ctrl.Result{}, nil
}

// updateExistingNodesOptimized updates existing nodes if needed
func (r *OptimizedE2NodeSetReconciler) updateExistingNodesOptimized(
	ctx context.Context,
	e2nodeSet *nephoranv1.E2NodeSet,
	reconcileCtx *E2NodeSetReconcileContext,
) (ctrl.Result, error) {

	logger := log.FromContext(ctx).WithValues("phase", "update-nodes")

	// For now, this is a placeholder - in a real implementation,
	// you would check if any configuration updates are needed
	// and apply them batch-wise

	logger.V(1).Info("Node update phase completed", "nodes_checked", len(reconcileCtx.ExistingConfigMaps))

	return ctrl.Result{}, nil
}

// createE2NodeOptimized creates a single E2 node ConfigMap
func (r *OptimizedE2NodeSetReconciler) createE2NodeOptimized(ctx context.Context, e2nodeSet *nephoranv1.E2NodeSet, nodeIndex int) error {
	timer := r.metrics.NewApiCallTimer(OptimizedE2NodeSetController, "create", "ConfigMap")

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("e2node-%s-%d", e2nodeSet.Name, nodeIndex),
			Namespace: e2nodeSet.Namespace,
			Labels: map[string]string{
				E2NodeSetLabelKey:   e2nodeSet.Name,
				E2NodeAppLabelKey:   E2NodeAppLabelValue,
				E2NodeIDLabelKey:    fmt.Sprintf("e2node-%d", nodeIndex),
				E2NodeIndexLabelKey: fmt.Sprintf("%d", nodeIndex),
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: e2nodeSet.APIVersion,
					Kind:       e2nodeSet.Kind,
					Name:       e2nodeSet.Name,
					UID:        e2nodeSet.UID,
					Controller: boolPtr(true),
				},
			},
		},
		Data: map[string]string{
			"node-config.json": fmt.Sprintf(`{
				"nodeId": "e2node-%d",
				"ricEndpoint": "%s",
				"e2InterfaceVersion": "v3.0",
				"createdAt": "%s"
			}`, nodeIndex, e2nodeSet.Spec.RicEndpoint, time.Now().Format(time.RFC3339)),
		},
	}

	err := r.Create(ctx, configMap)
	timer.FinishWithResult(err == nil, r.backoffManager.ClassifyError(err).String())

	return err
}

// deleteE2NodeOptimized deletes a single E2 node ConfigMap
func (r *OptimizedE2NodeSetReconciler) deleteE2NodeOptimized(ctx context.Context, configMap *corev1.ConfigMap) error {
	timer := r.metrics.NewApiCallTimer(OptimizedE2NodeSetController, "delete", "ConfigMap")

	err := r.Delete(ctx, configMap)
	timer.FinishWithResult(err == nil, r.backoffManager.ClassifyError(err).String())

	return err
}

// Helper methods

func (r *OptimizedE2NodeSetReconciler) queueE2NodeSetStatusUpdate(e2nodeSet *nephoranv1.E2NodeSet, readyReplicas, totalReplicas int32, phase string, priority UpdatePriority) {
	key := types.NamespacedName{
		Namespace: e2nodeSet.Namespace,
		Name:      e2nodeSet.Name,
	}

	condition := metav1.Condition{
		Type:               "Phase",
		Status:             metav1.ConditionTrue,
		Reason:             phase,
		Message:            fmt.Sprintf("E2NodeSet is in %s phase", phase),
		LastTransitionTime: metav1.Now(),
	}

	r.statusBatcher.QueueE2NodeSetUpdate(key, readyReplicas, totalReplicas, []metav1.Condition{condition}, priority)
	r.metrics.RecordStatusUpdate(OptimizedE2NodeSetController, priority.String(), "E2NodeSet", "queued")
}

func (r *OptimizedE2NodeSetReconciler) hasFinalizer(e2nodeSet *nephoranv1.E2NodeSet, finalizer string) bool {
	for _, f := range e2nodeSet.Finalizers {
		if f == finalizer {
			return true
		}
	}
	return false
}

func (r *OptimizedE2NodeSetReconciler) addFinalizerOptimized(ctx context.Context, e2nodeSet *nephoranv1.E2NodeSet) error {
	e2nodeSet.Finalizers = append(e2nodeSet.Finalizers, E2NodeSetFinalizer)

	timer := r.metrics.NewApiCallTimer(OptimizedE2NodeSetController, "update", "E2NodeSet")
	err := r.Update(ctx, e2nodeSet)
	timer.FinishWithResult(err == nil, r.backoffManager.ClassifyError(err).String())

	return err
}

func (r *OptimizedE2NodeSetReconciler) handleDeletionOptimized(
	ctx context.Context,
	e2nodeSet *nephoranv1.E2NodeSet,
	reconcileCtx *E2NodeSetReconcileContext,
	resourceKey string,
) (ctrl.Result, error) {

	logger := log.FromContext(ctx).WithValues("phase", "deletion")

	// Queue deletion status update
	r.queueE2NodeSetStatusUpdate(e2nodeSet, 0, 0, "Deleting", CriticalPriority)

	// Get existing ConfigMaps for cleanup
	existingConfigMaps, err := r.getExistingConfigMapsOptimized(ctx, e2nodeSet)
	if err != nil {
		errorType := r.backoffManager.ClassifyError(err)
		delay := r.backoffManager.GetNextDelay(resourceKey, errorType, err)

		logger.Error(err, "Failed to get ConfigMaps for cleanup", "backoff_delay", delay)
		return ctrl.Result{RequeueAfter: delay}, err
	}

	// Delete all associated ConfigMaps
	successCount := 0
	errorCount := 0

	for _, configMap := range existingConfigMaps {
		if err := r.deleteE2NodeOptimized(ctx, configMap); err != nil {
			logger.Error(err, "Failed to delete ConfigMap during cleanup", "name", configMap.Name)
			errorCount++
		} else {
			successCount++
		}
	}

	// If cleanup failed, requeue with backoff
	if errorCount > 0 {
		errorType := ResourceError
		delay := r.backoffManager.GetNextDelay(resourceKey, errorType, fmt.Errorf("failed to cleanup %d ConfigMaps", errorCount))

		logger.Error(err, "Cleanup failed for some ConfigMaps", "failed_count", errorCount, "backoff_delay", delay)
		return ctrl.Result{RequeueAfter: delay}, fmt.Errorf("cleanup failed for %d ConfigMaps", errorCount)
	}

	// Remove finalizer
	finalizers := make([]string, 0)
	for _, f := range e2nodeSet.Finalizers {
		if f != E2NodeSetFinalizer {
			finalizers = append(finalizers, f)
		}
	}
	e2nodeSet.Finalizers = finalizers

	timer := r.metrics.NewApiCallTimer(OptimizedE2NodeSetController, "update", "E2NodeSet")
	err = r.Update(ctx, e2nodeSet)
	timer.FinishWithResult(err == nil, r.backoffManager.ClassifyError(err).String())

	if err != nil {
		errorType := r.backoffManager.ClassifyError(err)
		delay := r.backoffManager.GetNextDelay(resourceKey, errorType, err)

		logger.Error(err, "Failed to remove finalizer", "backoff_delay", delay)
		return ctrl.Result{RequeueAfter: delay}, err
	}

	logger.Info("E2NodeSet deletion completed",
		"cleaned_configmaps", successCount,
		"cleanup_duration", time.Since(reconcileCtx.StartTime))

	r.metrics.RecordReconcileResult(OptimizedE2NodeSetController, "deleted")

	return ctrl.Result{}, nil
}

func (r *OptimizedE2NodeSetReconciler) calculateOptimalRequeueInterval(e2nodeSet *nephoranv1.E2NodeSet, reconcileCtx *E2NodeSetReconcileContext) time.Duration {
	// Base interval depends on current state
	var baseInterval time.Duration

	switch {
	case reconcileCtx.ErrorCount > 0:
		baseInterval = 1 * time.Minute // Errors occurred, check sooner
	case reconcileCtx.NodesCreated > 0 || reconcileCtx.NodesDeleted > 0:
		baseInterval = 2 * time.Minute // State changed, check sooner
	default:
		baseInterval = 5 * time.Minute // Stable state
	}

	// Adjust based on processing time
	processingTime := time.Since(reconcileCtx.StartTime)
	if processingTime > 5*time.Second {
		baseInterval = time.Duration(float64(baseInterval) * 1.2)
	}

	// Cap the interval
	maxInterval := 10 * time.Minute
	if baseInterval > maxInterval {
		baseInterval = maxInterval
	}

	return baseInterval
}

func (r *OptimizedE2NodeSetReconciler) cleanupCache() {
	defer r.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			// Clean up stale cache entries
			r.configMapCache.Range(func(key, value interface{}) bool {
				if cacheEntry, ok := value.(configMapCacheEntry); ok {
					if time.Since(cacheEntry.timestamp) > 10*time.Minute {
						r.configMapCache.Delete(key)
					}
				}
				return true
			})
		}
	}
}

// SetupWithManager sets up the controller with optimized configuration
func (r *OptimizedE2NodeSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nephoranv1.E2NodeSet{}).
		Owns(&corev1.ConfigMap{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 3, // Moderate concurrency for E2NodeSet
		}).
		// Removed Watches - redundant with For() in controller-runtime v0.18+
		Complete(r)
}

// Shutdown gracefully shuts down the optimized controller
func (r *OptimizedE2NodeSetReconciler) Shutdown() error {
	r.cancel()

	// Stop status batcher
	if err := r.statusBatcher.Stop(); err != nil {
		return fmt.Errorf("failed to stop status batcher: %w", err)
	}

	// Wait for background goroutines
	r.wg.Wait()

	return nil
}

// Cache entry structure
type configMapCacheEntry struct {
	configMaps []*corev1.ConfigMap
	timestamp  time.Time
}

// Helper function
func boolPtr(b bool) *bool {
	return &b
}
