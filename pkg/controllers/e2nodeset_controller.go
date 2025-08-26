package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/prometheus/client_golang/prometheus"
	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

const (
	E2NodeSetFinalizer = "nephoran.com/e2nodeset-finalizer"

	// Labels for ConfigMap selection and management
	E2NodeSetLabelKey   = "nephoran.com/e2-nodeset"
	E2NodeAppLabelKey   = "app"
	E2NodeAppLabelValue = "e2-node-simulator"
	E2NodeIDLabelKey    = "nephoran.com/node-id"
	E2NodeIndexLabelKey = "nephoran.com/node-index"

	// ConfigMap data keys
	E2NodeConfigKey = "e2node-config.json"
	E2NodeStatusKey = "e2node-status.json"

	// Default values
	DefaultHeartbeatInterval  = 10 * time.Second
	DefaultMetricsInterval    = "30s"
	DefaultE2InterfaceVersion = "v3.0"

	// Retry and backoff configuration
	DefaultMaxRetries = 3
	BaseBackoffDelay  = 1 * time.Second
	MaxBackoffDelay   = 5 * time.Minute
	JitterFactor      = 0.1 // 10% jitter
	BackoffMultiplier = 2.0
)

// E2NodeConfigData represents the configuration data stored in ConfigMap
type E2NodeConfigData struct {
	NodeID             string               `json:"nodeId"`
	E2InterfaceVersion string               `json:"e2InterfaceVersion"`
	RICEndpoint        string               `json:"ricEndpoint"`
	RANFunctions       []RANFunctionConfig  `json:"ranFunctions"`
	SimulationConfig   SimulationConfigData `json:"simulationConfig"`
	CreatedAt          time.Time            `json:"createdAt"`
	UpdatedAt          time.Time            `json:"updatedAt"`
}

// E2NodeStatusData represents the status data stored in ConfigMap
type E2NodeStatusData struct {
	NodeID              string     `json:"nodeId"`
	State               string     `json:"state"`
	LastHeartbeat       *time.Time `json:"lastHeartbeat,omitempty"`
	ConnectedSince      *time.Time `json:"connectedSince,omitempty"`
	ActiveSubscriptions int32      `json:"activeSubscriptions"`
	ErrorMessage        string     `json:"errorMessage,omitempty"`
	HeartbeatCount      int64      `json:"heartbeatCount"`
	StatusUpdatedAt     time.Time  `json:"statusUpdatedAt"`
}

// RANFunctionConfig represents RAN function configuration
type RANFunctionConfig struct {
	FunctionID  int32  `json:"functionId"`
	Revision    int32  `json:"revision"`
	Description string `json:"description"`
	OID         string `json:"oid"`
}

// SimulationConfigData represents simulation configuration
type SimulationConfigData struct {
	UECount           int32  `json:"ueCount"`
	TrafficGeneration bool   `json:"trafficGeneration"`
	MetricsInterval   string `json:"metricsInterval"`
	TrafficProfile    string `json:"trafficProfile"`
}

type E2NodeSetReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	// Metrics
	nodesTotal      prometheus.GaugeVec
	nodesReady      prometheus.GaugeVec
	reconcilesTotal prometheus.CounterVec
	reconcileErrors prometheus.CounterVec
	heartbeatsTotal prometheus.CounterVec
}

// RegisterMetrics registers Prometheus metrics
func (r *E2NodeSetReconciler) RegisterMetrics() {
	r.nodesTotal = *prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "e2nodeset_nodes_total",
			Help: "Total number of E2 nodes in E2NodeSet",
		},
		[]string{"namespace", "name"},
	)

	r.nodesReady = *prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "e2nodeset_nodes_ready",
			Help: "Number of ready E2 nodes in E2NodeSet",
		},
		[]string{"namespace", "name"},
	)

	r.reconcilesTotal = *prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "e2nodeset_reconciles_total",
			Help: "Total number of E2NodeSet reconciliations",
		},
		[]string{"namespace", "name", "result"},
	)

	r.reconcileErrors = *prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "e2nodeset_reconcile_errors_total",
			Help: "Total number of E2NodeSet reconciliation errors",
		},
		[]string{"namespace", "name", "error_type"},
	)

	r.heartbeatsTotal = *prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "e2nodeset_heartbeats_total",
			Help: "Total number of E2 node heartbeats",
		},
		[]string{"namespace", "name", "node_id"},
	)

	metrics.Registry.MustRegister(
		&r.nodesTotal,
		&r.nodesReady,
		&r.reconcilesTotal,
		&r.reconcileErrors,
		&r.heartbeatsTotal,
	)
}

//+kubebuilder:rbac:groups=nephoran.com,resources=e2nodesets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nephoran.com,resources=e2nodesets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nephoran.com,resources=e2nodesets/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *E2NodeSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	startTime := time.Now()
	result := "success"

	defer func() {
		r.reconcilesTotal.WithLabelValues(req.Namespace, req.Name, result).Inc()
		logger.Info("Reconciliation completed", "duration", time.Since(startTime), "result", result)
	}()

	// Fetch the E2NodeSet instance
	var e2nodeSet nephoranv1.E2NodeSet
	if err := r.Get(ctx, req.NamespacedName, &e2nodeSet); err != nil {
		if errors.IsNotFound(err) {
			// E2NodeSet was deleted, cleanup will be handled by finalizer
			logger.Info("E2NodeSet resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get E2NodeSet")
		result = "error"
		r.reconcileErrors.WithLabelValues(req.Namespace, req.Name, "fetch_error").Inc()
		return ctrl.Result{}, err
	}

	logger.Info("Reconciling E2NodeSet",
		"name", e2nodeSet.Name,
		"namespace", e2nodeSet.Namespace,
		"replicas", e2nodeSet.Spec.Replicas,
		"generation", e2nodeSet.Generation)

	// Handle deletion
	if e2nodeSet.DeletionTimestamp != nil {
		return r.handleDeletion(ctx, &e2nodeSet)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&e2nodeSet, E2NodeSetFinalizer) {
		controllerutil.AddFinalizer(&e2nodeSet, E2NodeSetFinalizer)
		if err := r.Update(ctx, &e2nodeSet); err != nil {
			logger.Error(err, "Failed to add finalizer")
			result = "error"
			r.reconcileErrors.WithLabelValues(req.Namespace, req.Name, "finalizer_error").Inc()
			return ctrl.Result{}, err
		}
		logger.Info("Added finalizer to E2NodeSet")
		return ctrl.Result{Requeue: true}, nil
	}

	// Reconcile E2 nodes
	if err := r.reconcileE2Nodes(ctx, &e2nodeSet); err != nil {
		logger.Error(err, "Failed to reconcile E2 nodes")
		result = "error"
		r.reconcileErrors.WithLabelValues(req.Namespace, req.Name, "reconcile_nodes_error").Inc()

		// Get retry count and calculate backoff
		retryCount := getRetryCount(&e2nodeSet, "e2-provisioning")
		if retryCount < DefaultMaxRetries {
			setRetryCount(&e2nodeSet, "e2-provisioning", retryCount+1)
			r.Update(ctx, &e2nodeSet)

			// Set Ready condition to False
			r.setReadyCondition(ctx, &e2nodeSet, metav1.ConditionFalse, "E2NodesReconciliationFailed",
				fmt.Sprintf("Failed to reconcile E2 nodes (attempt %d/%d): %v", retryCount+1, DefaultMaxRetries, err))

			// Record event for reconciliation failure
			r.Recorder.Eventf(&e2nodeSet, corev1.EventTypeWarning, "ReconciliationRetrying",
				"Failed to reconcile E2 nodes, retrying (attempt %d/%d): %v", retryCount+1, DefaultMaxRetries, err)

			backoffDelay := calculateExponentialBackoffForOperation(retryCount, "e2-provisioning")
			logger.V(1).Info("Scheduling E2 nodes reconciliation retry with exponential backoff",
				"delay", backoffDelay,
				"attempt", retryCount+1,
				"max_retries", DefaultMaxRetries)

			return ctrl.Result{RequeueAfter: backoffDelay}, nil
		} else {
			// Max retries exceeded
			r.setReadyCondition(ctx, &e2nodeSet, metav1.ConditionFalse, "E2NodesReconciliationFailedMaxRetries",
				fmt.Sprintf("Failed to reconcile E2 nodes after %d retries: %v", DefaultMaxRetries, err))

			r.Recorder.Eventf(&e2nodeSet, corev1.EventTypeWarning, "ReconciliationFailedMaxRetries",
				"Failed to reconcile E2 nodes after %d retries: %v", DefaultMaxRetries, err)

			return ctrl.Result{RequeueAfter: time.Hour}, err // Long delay before retry
		}
	}

	// Clear retry counts on successful reconciliation
	clearRetryCount(&e2nodeSet, "e2-provisioning")
	clearRetryCount(&e2nodeSet, "configmap-operations")
	r.Update(ctx, &e2nodeSet)

	// Set Ready condition to True
	r.setReadyCondition(ctx, &e2nodeSet, metav1.ConditionTrue, "E2NodesReady", "All E2 nodes are successfully reconciled")

	// Update metrics
	r.updateMetrics(&e2nodeSet)

	// Start heartbeat simulation for connected nodes
	go r.simulateHeartbeats(ctx, &e2nodeSet)

	return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
}

func (r *E2NodeSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Register metrics if not already registered
	if r.Recorder == nil {
		r.Recorder = mgr.GetEventRecorderFor("e2nodeset-controller")
	}

	// Setup controller to watch E2NodeSet resources and ConfigMaps
	return ctrl.NewControllerManagedBy(mgr).
		For(&nephoranv1.E2NodeSet{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

// reconcileE2Nodes manages E2 nodes by creating/deleting ConfigMaps
func (r *E2NodeSetReconciler) reconcileE2Nodes(ctx context.Context, e2nodeSet *nephoranv1.E2NodeSet) error {
	logger := log.FromContext(ctx)

	// Get current ConfigMaps representing E2 nodes
	currentConfigMaps, err := r.getCurrentE2NodeConfigMaps(ctx, e2nodeSet)
	if err != nil {
		return fmt.Errorf("failed to get current E2 node ConfigMaps: %w", err)
	}

	currentReplicas := int32(len(currentConfigMaps))
	desiredReplicas := e2nodeSet.Spec.Replicas

	logger.Info("E2NodeSet scaling status",
		"current", currentReplicas,
		"desired", desiredReplicas)

	if currentReplicas < desiredReplicas {
		// Scale up - create new E2 node ConfigMaps
		return r.scaleUpE2Nodes(ctx, e2nodeSet, currentConfigMaps, desiredReplicas)
	} else if currentReplicas > desiredReplicas {
		// Scale down - delete excess E2 node ConfigMaps
		return r.scaleDownE2Nodes(ctx, e2nodeSet, currentConfigMaps, desiredReplicas)
	}

	// Check if any nodes need updates
	return r.updateE2Nodes(ctx, e2nodeSet, currentConfigMaps)
}

// getCurrentE2NodeConfigMaps returns ConfigMaps representing E2 nodes for this E2NodeSet
func (r *E2NodeSetReconciler) getCurrentE2NodeConfigMaps(ctx context.Context, e2nodeSet *nephoranv1.E2NodeSet) ([]*corev1.ConfigMap, error) {
	// Create label selector for E2 node ConfigMaps
	selector := labels.NewSelector()

	// Add requirement for e2nodeset label
	req, err := labels.NewRequirement(E2NodeSetLabelKey, selection.Equals, []string{e2nodeSet.Name})
	if err != nil {
		return nil, fmt.Errorf("failed to create label requirement: %w", err)
	}
	selector = selector.Add(*req)

	// Add requirement for app label
	appReq, err := labels.NewRequirement(E2NodeAppLabelKey, selection.Equals, []string{E2NodeAppLabelValue})
	if err != nil {
		return nil, fmt.Errorf("failed to create app label requirement: %w", err)
	}
	selector = selector.Add(*appReq)

	// List ConfigMaps with selector
	var configMapList corev1.ConfigMapList
	listOpts := &client.ListOptions{
		Namespace:     e2nodeSet.Namespace,
		LabelSelector: selector,
	}

	if err := r.List(ctx, &configMapList, listOpts); err != nil {
		return nil, fmt.Errorf("failed to list E2 node ConfigMaps: %w", err)
	}

	// Convert to slice of pointers and sort by node index
	configMaps := make([]*corev1.ConfigMap, len(configMapList.Items))
	for i := range configMapList.Items {
		configMaps[i] = &configMapList.Items[i]
	}

	// Sort ConfigMaps by node index for consistent ordering
	sort.Slice(configMaps, func(i, j int) bool {
		indexI := r.getNodeIndexFromConfigMap(configMaps[i])
		indexJ := r.getNodeIndexFromConfigMap(configMaps[j])
		return indexI < indexJ
	})

	return configMaps, nil
}

// scaleUpE2Nodes creates new E2 node ConfigMaps
func (r *E2NodeSetReconciler) scaleUpE2Nodes(ctx context.Context, e2nodeSet *nephoranv1.E2NodeSet,
	currentConfigMaps []*corev1.ConfigMap, desiredReplicas int32) error {
	logger := log.FromContext(ctx)

	// Find next available index
	usedIndices := make(map[int32]bool)
	for _, cm := range currentConfigMaps {
		index := r.getNodeIndexFromConfigMap(cm)
		usedIndices[index] = true
	}

	// Create new ConfigMaps for missing indices
	var nextIndex int32 = 0
	for int32(len(currentConfigMaps)) < desiredReplicas {
		// Find next unused index
		for usedIndices[nextIndex] {
			nextIndex++
		}

		configMap, err := r.createE2NodeConfigMap(ctx, e2nodeSet, nextIndex)
		if err != nil {
			// Error handling is done inside createE2NodeConfigMap
			return fmt.Errorf("failed to create E2 node ConfigMap at index %d: %w", nextIndex, err)
		}

		logger.Info("Created E2 node ConfigMap",
			"name", configMap.Name,
			"nodeId", r.generateNodeID(e2nodeSet, nextIndex),
			"index", nextIndex)

		// Record event for node creation
		r.Recorder.Eventf(e2nodeSet, corev1.EventTypeNormal, "E2NodeCreated",
			"Created E2 node %s at index %d", configMap.Name, nextIndex)

		currentConfigMaps = append(currentConfigMaps, configMap)
		usedIndices[nextIndex] = true
		nextIndex++
	}

	return nil
}

// scaleDownE2Nodes deletes excess E2 node ConfigMaps
func (r *E2NodeSetReconciler) scaleDownE2Nodes(ctx context.Context, e2nodeSet *nephoranv1.E2NodeSet,
	currentConfigMaps []*corev1.ConfigMap, desiredReplicas int32) error {
	logger := log.FromContext(ctx)

	// Sort ConfigMaps by index (highest first for deletion)
	sort.Slice(currentConfigMaps, func(i, j int) bool {
		indexI := r.getNodeIndexFromConfigMap(currentConfigMaps[i])
		indexJ := r.getNodeIndexFromConfigMap(currentConfigMaps[j])
		return indexI > indexJ
	})

	// Delete excess ConfigMaps
	for i := int32(0); i < int32(len(currentConfigMaps))-desiredReplicas; i++ {
		configMap := currentConfigMaps[i]
		nodeID := r.getNodeIDFromConfigMap(configMap)

		logger.Info("Deleting E2 node ConfigMap",
			"name", configMap.Name,
			"nodeId", nodeID)

		if err := r.Delete(ctx, configMap); err != nil {
			if !errors.IsNotFound(err) {
				// Get retry count for deletion operations
				retryCount := getRetryCount(e2nodeSet, "configmap-operations")
				if retryCount < DefaultMaxRetries {
					setRetryCount(e2nodeSet, "configmap-operations", retryCount+1)
					r.Client.Update(ctx, e2nodeSet)

					// Set Ready condition to False
					r.setReadyCondition(ctx, e2nodeSet, metav1.ConditionFalse, "ConfigMapDeletionFailed",
						fmt.Sprintf("Failed to delete ConfigMap %s (attempt %d/%d): %v", configMap.Name, retryCount+1, DefaultMaxRetries, err))
				}
				return fmt.Errorf("failed to delete E2 node ConfigMap %s: %w", configMap.Name, err)
			}
		}

		// Record event for node deletion
		r.Recorder.Eventf(e2nodeSet, corev1.EventTypeNormal, "E2NodeDeleted",
			"Deleted E2 node %s", configMap.Name)
	}

	return nil
}

// updateE2Nodes checks if existing ConfigMaps need updates
func (r *E2NodeSetReconciler) updateE2Nodes(ctx context.Context, e2nodeSet *nephoranv1.E2NodeSet,
	currentConfigMaps []*corev1.ConfigMap) error {
	logger := log.FromContext(ctx)

	for _, configMap := range currentConfigMaps {
		// Check if ConfigMap needs update based on E2NodeSet template
		if r.configMapNeedsUpdate(configMap, e2nodeSet) {
			index := r.getNodeIndexFromConfigMap(configMap)
			nodeID := r.generateNodeID(e2nodeSet, index)

			logger.Info("Updating E2 node ConfigMap", "name", configMap.Name, "nodeId", nodeID)

			updatedConfigMap := r.buildE2NodeConfigMap(e2nodeSet, index)
			updatedConfigMap.Name = configMap.Name
			updatedConfigMap.Namespace = configMap.Namespace
			updatedConfigMap.ResourceVersion = configMap.ResourceVersion

			// Preserve existing status if available
			if statusData, exists := configMap.Data[E2NodeStatusKey]; exists {
				updatedConfigMap.Data[E2NodeStatusKey] = statusData
			}

			// Check retry count before attempting update
			retryCount := getRetryCount(e2nodeSet, "configmap-operations")
			if retryCount >= DefaultMaxRetries {
				return fmt.Errorf("max retries (%d) exceeded for ConfigMap update", DefaultMaxRetries)
			}

			if err := r.Update(ctx, updatedConfigMap); err != nil {
				// Increment retry count on failure
				setRetryCount(e2nodeSet, "configmap-operations", retryCount+1)
				r.Client.Update(ctx, e2nodeSet)

				// Set Ready condition to False
				r.setReadyCondition(ctx, e2nodeSet, metav1.ConditionFalse, "ConfigMapUpdateFailed",
					fmt.Sprintf("Failed to update ConfigMap %s (attempt %d/%d): %v", configMap.Name, retryCount+1, DefaultMaxRetries, err))

				return fmt.Errorf("failed to update E2 node ConfigMap %s: %w", configMap.Name, err)
			}

			// Clear retry count on successful update
			clearRetryCount(e2nodeSet, "configmap-operations")
			r.Client.Update(ctx, e2nodeSet)

			// Record event for node update
			r.Recorder.Eventf(e2nodeSet, corev1.EventTypeNormal, "E2NodeUpdated",
				"Updated E2 node %s configuration", configMap.Name)
		}
	}

	return nil
}

// Helper functions

// generateNodeID generates a unique node ID for an E2 node
func (r *E2NodeSetReconciler) generateNodeID(e2nodeSet *nephoranv1.E2NodeSet, index int32) string {
	return fmt.Sprintf("%s-%s-node-%d", e2nodeSet.Namespace, e2nodeSet.Name, index)
}

// createE2NodeConfigMap creates a new ConfigMap representing an E2 node with retry logic
func (r *E2NodeSetReconciler) createE2NodeConfigMap(ctx context.Context, e2nodeSet *nephoranv1.E2NodeSet, index int32) (*corev1.ConfigMap, error) {
	configMap := r.buildE2NodeConfigMap(e2nodeSet, index)

	if err := controllerutil.SetControllerReference(e2nodeSet, configMap, r.Scheme); err != nil {
		return nil, fmt.Errorf("failed to set controller reference: %w", err)
	}

	// Check retry count before attempting create
	retryCount := getRetryCount(e2nodeSet, "configmap-operations")
	if retryCount >= DefaultMaxRetries {
		return nil, fmt.Errorf("max retries (%d) exceeded for ConfigMap creation", DefaultMaxRetries)
	}

	if err := r.Create(ctx, configMap); err != nil {
		// Increment retry count on failure
		setRetryCount(e2nodeSet, "configmap-operations", retryCount+1)
		r.Client.Update(ctx, e2nodeSet)

		// Set Ready condition to False
		r.setReadyCondition(ctx, e2nodeSet, metav1.ConditionFalse, "ConfigMapCreationFailed",
			fmt.Sprintf("Failed to create ConfigMap %s (attempt %d/%d): %v", configMap.Name, retryCount+1, DefaultMaxRetries, err))

		return nil, fmt.Errorf("failed to create ConfigMap: %w", err)
	}

	// Clear retry count on successful creation
	clearRetryCount(e2nodeSet, "configmap-operations")
	r.Client.Update(ctx, e2nodeSet)

	return configMap, nil
}

// Exponential backoff helper functions

// calculateExponentialBackoff calculates the exponential backoff delay with jitter
// retryCount: current retry attempt (0-based)
// baseDelay: base delay duration (defaults to BaseBackoffDelay if zero)
// maxDelay: maximum delay duration (defaults to MaxBackoffDelay if zero)
func calculateExponentialBackoff(retryCount int, baseDelay, maxDelay time.Duration) time.Duration {
	if baseDelay <= 0 {
		baseDelay = BaseBackoffDelay
	}
	if maxDelay <= 0 {
		maxDelay = MaxBackoffDelay
	}

	// Calculate exponential backoff: baseDelay * multiplier^retryCount
	backoffSeconds := float64(baseDelay.Seconds()) * math.Pow(BackoffMultiplier, float64(retryCount))
	backoffDuration := time.Duration(backoffSeconds * float64(time.Second))

	// Cap at max delay
	if backoffDuration > maxDelay {
		backoffDuration = maxDelay
	}

	// Add jitter to prevent thundering herd
	jitterRange := float64(backoffDuration) * JitterFactor
	jitter := (rand.Float64()*2 - 1) * jitterRange // Random value between -jitterRange and +jitterRange
	finalDelay := time.Duration(float64(backoffDuration) + jitter)

	// Ensure final delay is positive and doesn't exceed max
	if finalDelay < 0 {
		finalDelay = baseDelay
	}
	if finalDelay > maxDelay {
		finalDelay = maxDelay
	}

	return finalDelay
}

// calculateExponentialBackoffForOperation calculates backoff with operation-specific configurations
func calculateExponentialBackoffForOperation(retryCount int, operation string) time.Duration {
	switch operation {
	case "configmap-operations":
		// ConfigMap operations: moderate backoff for Kubernetes API
		return calculateExponentialBackoff(retryCount, 2*time.Second, 2*time.Minute)
	case "e2-provisioning":
		// E2 node provisioning: longer backoff for complex operations
		return calculateExponentialBackoff(retryCount, 5*time.Second, 5*time.Minute)
	case "cleanup":
		// Cleanup operations: existing configuration
		return calculateExponentialBackoff(retryCount, 10*time.Second, 5*time.Minute)
	default:
		// Default configuration
		return calculateExponentialBackoff(retryCount, BaseBackoffDelay, MaxBackoffDelay)
	}
}

// Retry count management functions

// getRetryCount retrieves the retry count for a specific operation from annotations
func getRetryCount(e2nodeSet *nephoranv1.E2NodeSet, operation string) int {
	if e2nodeSet.Annotations == nil {
		return 0
	}

	key := fmt.Sprintf("nephoran.com/%s-retry-count", operation)
	if countStr, exists := e2nodeSet.Annotations[key]; exists {
		if count, err := fmt.Sscanf(countStr, "%d", new(int)); err == nil && count == 1 {
			var result int
			fmt.Sscanf(countStr, "%d", &result)
			return result
		}
	}
	return 0
}

// setRetryCount sets the retry count for a specific operation in annotations
func setRetryCount(e2nodeSet *nephoranv1.E2NodeSet, operation string, count int) {
	if e2nodeSet.Annotations == nil {
		e2nodeSet.Annotations = make(map[string]string)
	}

	key := fmt.Sprintf("nephoran.com/%s-retry-count", operation)
	e2nodeSet.Annotations[key] = fmt.Sprintf("%d", count)
}

// clearRetryCount removes the retry count for a specific operation from annotations
func clearRetryCount(e2nodeSet *nephoranv1.E2NodeSet, operation string) {
	if e2nodeSet.Annotations == nil {
		return
	}

	key := fmt.Sprintf("nephoran.com/%s-retry-count", operation)
	delete(e2nodeSet.Annotations, key)
}

// setReadyCondition sets the Ready condition with proper reason and message
func (r *E2NodeSetReconciler) setReadyCondition(ctx context.Context, e2nodeSet *nephoranv1.E2NodeSet, status metav1.ConditionStatus, reason, message string) error {
	condition := nephoranv1.E2NodeSetCondition{
		Type:               "Ready",
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	// Update the condition
	r.setE2NodeSetCondition(e2nodeSet, condition)

	// Update the status
	return r.Status().Update(ctx, e2nodeSet)
}

// buildE2NodeConfigMap builds a ConfigMap for an E2 node
func (r *E2NodeSetReconciler) buildE2NodeConfigMap(e2nodeSet *nephoranv1.E2NodeSet, index int32) *corev1.ConfigMap {
	nodeID := r.generateNodeID(e2nodeSet, index)
	configMapName := r.generateConfigMapName(e2nodeSet, index)
	ricEndpoint := r.getRICEndpoint(e2nodeSet)

	// Build configuration data
	config := E2NodeConfigData{
		NodeID:             nodeID,
		E2InterfaceVersion: r.getE2InterfaceVersion(e2nodeSet),
		RICEndpoint:        ricEndpoint,
		RANFunctions:       r.buildRANFunctions(e2nodeSet),
		SimulationConfig:   r.buildSimulationConfig(e2nodeSet),
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	// Build initial status data
	now := time.Now().UTC()
	status := E2NodeStatusData{
		NodeID:              nodeID,
		State:               string(nephoranv1.E2NodeLifecycleStatePending),
		ActiveSubscriptions: 0,
		HeartbeatCount:      0,
		StatusUpdatedAt:     now,
	}

	// Marshal to JSON
	configJSON, _ := json.Marshal(config)
	statusJSON, _ := json.Marshal(status)

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: e2nodeSet.Namespace,
			Labels: map[string]string{
				E2NodeSetLabelKey:   e2nodeSet.Name,
				E2NodeAppLabelKey:   E2NodeAppLabelValue,
				E2NodeIDLabelKey:    nodeID,
				E2NodeIndexLabelKey: strconv.Itoa(int(index)),
			},
			Annotations: map[string]string{
				"nephoran.com/e2-interface-version": config.E2InterfaceVersion,
				"nephoran.com/ric-endpoint":         ricEndpoint,
				"nephoran.com/created-at":           now.Format(time.RFC3339),
			},
		},
		Data: map[string]string{
			E2NodeConfigKey: string(configJSON),
			E2NodeStatusKey: string(statusJSON),
		},
	}

	return configMap
}

// buildRANFunctions creates RAN function configurations
func (r *E2NodeSetReconciler) buildRANFunctions(e2nodeSet *nephoranv1.E2NodeSet) []RANFunctionConfig {
	// Use template RAN functions if provided, otherwise use defaults
	if len(e2nodeSet.Spec.Template.Spec.SupportedRANFunctions) > 0 {
		functions := make([]RANFunctionConfig, len(e2nodeSet.Spec.Template.Spec.SupportedRANFunctions))
		for i, ranFunc := range e2nodeSet.Spec.Template.Spec.SupportedRANFunctions {
			functions[i] = RANFunctionConfig{
				FunctionID:  ranFunc.FunctionID,
				Revision:    ranFunc.Revision,
				Description: ranFunc.Description,
				OID:         ranFunc.OID,
			}
		}
		return functions
	}

	// Default RAN functions if none specified in template
	return []RANFunctionConfig{
		{
			FunctionID:  1,
			Revision:    1,
			Description: "Key Performance Measurement Service Model",
			OID:         "1.3.6.1.4.1.53148.1.1.2.2",
		},
		{
			FunctionID:  2,
			Revision:    1,
			Description: "RAN Control Service Model",
			OID:         "1.3.6.1.4.1.53148.1.1.2.3",
		},
		{
			FunctionID:  3,
			Revision:    1,
			Description: "Network Information Service Model",
			OID:         "1.3.6.1.4.1.53148.1.1.2.4",
		},
	}
}

// buildSimulationConfig creates simulation configuration
func (r *E2NodeSetReconciler) buildSimulationConfig(e2nodeSet *nephoranv1.E2NodeSet) SimulationConfigData {
	if e2nodeSet.Spec.SimulationConfig != nil {
		return SimulationConfigData{
			UECount:           e2nodeSet.Spec.SimulationConfig.UECount,
			TrafficGeneration: e2nodeSet.Spec.SimulationConfig.TrafficGeneration,
			MetricsInterval:   e2nodeSet.Spec.SimulationConfig.MetricsInterval,
			TrafficProfile:    string(e2nodeSet.Spec.SimulationConfig.TrafficProfile),
		}
	}

	// Default simulation config
	return SimulationConfigData{
		UECount:           100,
		TrafficGeneration: false,
		MetricsInterval:   DefaultMetricsInterval,
		TrafficProfile:    string(nephoranv1.TrafficProfileLow),
	}
}

// getE2InterfaceVersion returns E2 interface version from template or default
func (r *E2NodeSetReconciler) getE2InterfaceVersion(e2nodeSet *nephoranv1.E2NodeSet) string {
	if e2nodeSet.Spec.Template.Spec.E2InterfaceVersion != "" {
		return e2nodeSet.Spec.Template.Spec.E2InterfaceVersion
	}
	return DefaultE2InterfaceVersion
}

// getRICEndpoint returns the Near-RT RIC endpoint for E2 connections
func (r *E2NodeSetReconciler) getRICEndpoint(e2nodeSet *nephoranv1.E2NodeSet) string {
	// Priority order:
	// 1. Use ricConfiguration.ricEndpoint if provided
	// 2. Use deprecated spec.ricEndpoint for backward compatibility
	// 3. Fall back to annotation for legacy compatibility
	// 4. Use default endpoint

	if e2nodeSet.Spec.RICConfiguration != nil && e2nodeSet.Spec.RICConfiguration.RICEndpoint != "" {
		return e2nodeSet.Spec.RICConfiguration.RICEndpoint
	}

	if e2nodeSet.Spec.RicEndpoint != "" {
		return e2nodeSet.Spec.RicEndpoint
	}

	if endpoint, ok := e2nodeSet.Annotations["nephoran.com/near-rt-ric-endpoint"]; ok {
		return endpoint
	}

	return "http://near-rt-ric:38080" // Default Near-RT RIC endpoint
}

// generateConfigMapName generates a name for the E2 node ConfigMap
func (r *E2NodeSetReconciler) generateConfigMapName(e2nodeSet *nephoranv1.E2NodeSet, index int32) string {
	return fmt.Sprintf("%s-e2node-%d", e2nodeSet.Name, index)
}

// getNodeIndexFromConfigMap extracts the node index from ConfigMap labels
func (r *E2NodeSetReconciler) getNodeIndexFromConfigMap(configMap *corev1.ConfigMap) int32 {
	if indexStr, ok := configMap.Labels[E2NodeIndexLabelKey]; ok {
		if index, err := strconv.Atoi(indexStr); err == nil {
			return int32(index)
		}
	}
	return 0
}

// getNodeIDFromConfigMap extracts the node ID from ConfigMap labels
func (r *E2NodeSetReconciler) getNodeIDFromConfigMap(configMap *corev1.ConfigMap) string {
	if nodeID, ok := configMap.Labels[E2NodeIDLabelKey]; ok {
		return nodeID
	}
	return ""
}

// configMapNeedsUpdate checks if ConfigMap needs to be updated
func (r *E2NodeSetReconciler) configMapNeedsUpdate(configMap *corev1.ConfigMap, e2nodeSet *nephoranv1.E2NodeSet) bool {
	// Parse existing config
	configDataStr, ok := configMap.Data[E2NodeConfigKey]
	if !ok {
		return true // Missing config data
	}

	var existingConfig E2NodeConfigData
	if err := json.Unmarshal([]byte(configDataStr), &existingConfig); err != nil {
		return true // Invalid config data
	}

	// Check if key fields have changed
	if existingConfig.E2InterfaceVersion != r.getE2InterfaceVersion(e2nodeSet) {
		return true
	}

	if existingConfig.RICEndpoint != r.getRICEndpoint(e2nodeSet) {
		return true
	}

	// Check if RAN functions have changed
	expectedRANFunctions := r.buildRANFunctions(e2nodeSet)
	if len(existingConfig.RANFunctions) != len(expectedRANFunctions) {
		return true
	}

	for i, expected := range expectedRANFunctions {
		if i >= len(existingConfig.RANFunctions) ||
			existingConfig.RANFunctions[i].FunctionID != expected.FunctionID ||
			existingConfig.RANFunctions[i].Revision != expected.Revision {
			return true
		}
	}

	// Check simulation config changes
	expectedSimConfig := r.buildSimulationConfig(e2nodeSet)
	if existingConfig.SimulationConfig.UECount != expectedSimConfig.UECount ||
		existingConfig.SimulationConfig.TrafficGeneration != expectedSimConfig.TrafficGeneration ||
		existingConfig.SimulationConfig.TrafficProfile != expectedSimConfig.TrafficProfile {
		return true
	}

	return false
}

// updateMetrics updates Prometheus metrics for the E2NodeSet
func (r *E2NodeSetReconciler) updateMetrics(e2nodeSet *nephoranv1.E2NodeSet) {
	r.nodesTotal.WithLabelValues(e2nodeSet.Namespace, e2nodeSet.Name).Set(float64(e2nodeSet.Status.CurrentReplicas))
	r.nodesReady.WithLabelValues(e2nodeSet.Namespace, e2nodeSet.Name).Set(float64(e2nodeSet.Status.ReadyReplicas))
}

// simulateHeartbeats simulates heartbeat messages for connected E2 nodes
func (r *E2NodeSetReconciler) simulateHeartbeats(ctx context.Context, e2nodeSet *nephoranv1.E2NodeSet) {
	logger := log.FromContext(ctx)
	ticker := time.NewTicker(DefaultHeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Get current E2 node ConfigMaps
			configMaps, err := r.getCurrentE2NodeConfigMaps(ctx, e2nodeSet)
			if err != nil {
				logger.Error(err, "Failed to get ConfigMaps for heartbeat simulation")
				continue
			}

			// Update status for each node
			for _, configMap := range configMaps {
				if err := r.updateNodeHeartbeat(ctx, configMap); err != nil {
					logger.Error(err, "Failed to update node heartbeat", "configMap", configMap.Name)
				}
			}

			// Update E2NodeSet status
			if err := r.updateE2NodeSetStatus(ctx, e2nodeSet); err != nil {
				logger.Error(err, "Failed to update E2NodeSet status")
			}
		}
	}
}

// updateNodeHeartbeat updates heartbeat information for a single E2 node
func (r *E2NodeSetReconciler) updateNodeHeartbeat(ctx context.Context, configMap *corev1.ConfigMap) error {
	// Parse existing status
	statusDataStr, ok := configMap.Data[E2NodeStatusKey]
	if !ok {
		return fmt.Errorf("missing status data in ConfigMap %s", configMap.Name)
	}

	var status E2NodeStatusData
	if err := json.Unmarshal([]byte(statusDataStr), &status); err != nil {
		return fmt.Errorf("failed to unmarshal status data: %w", err)
	}

	// Update heartbeat information
	now := time.Now().UTC()
	status.LastHeartbeat = &now
	status.HeartbeatCount++
	status.StatusUpdatedAt = now

	// Simulate state transitions
	if status.State == string(nephoranv1.E2NodeLifecycleStatePending) {
		status.State = string(nephoranv1.E2NodeLifecycleStateInitializing)
	} else if status.State == string(nephoranv1.E2NodeLifecycleStateInitializing) {
		status.State = string(nephoranv1.E2NodeLifecycleStateConnected)
		status.ConnectedSince = &now
	}

	// Update ConfigMap with new status
	statusJSON, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal status data: %w", err)
	}

	updatedConfigMap := configMap.DeepCopy()
	updatedConfigMap.Data[E2NodeStatusKey] = string(statusJSON)

	if err := r.Update(ctx, updatedConfigMap); err != nil {
		// For heartbeat updates, we'll log the error but continue operation
		// since heartbeat failures shouldn't block the main reconciliation
		return fmt.Errorf("failed to update ConfigMap heartbeat: %w", err)
	}

	// Update heartbeat metrics
	nodeID := r.getNodeIDFromConfigMap(configMap)
	if nodeID != "" {
		namespaceName := strings.Split(nodeID, "-")
		if len(namespaceName) >= 2 {
			r.heartbeatsTotal.WithLabelValues(namespaceName[0], namespaceName[1], nodeID).Inc()
		}
	}

	return nil
}

// updateE2NodeSetStatus updates the E2NodeSet status based on current ConfigMaps
func (r *E2NodeSetReconciler) updateE2NodeSetStatus(ctx context.Context, e2nodeSet *nephoranv1.E2NodeSet) error {
	// Fetch current E2NodeSet to avoid conflicts
	var currentE2NodeSet nephoranv1.E2NodeSet
	if err := r.Get(ctx, types.NamespacedName{Name: e2nodeSet.Name, Namespace: e2nodeSet.Namespace}, &currentE2NodeSet); err != nil {
		return fmt.Errorf("failed to fetch current E2NodeSet: %w", err)
	}

	// Get current ConfigMaps
	configMaps, err := r.getCurrentE2NodeConfigMaps(ctx, &currentE2NodeSet)
	if err != nil {
		return fmt.Errorf("failed to get current ConfigMaps: %w", err)
	}

	// Count replicas by state
	var readyReplicas, currentReplicas, availableReplicas int32
	var e2NodeStatuses []nephoranv1.E2NodeStatus

	currentReplicas = int32(len(configMaps))

	for _, configMap := range configMaps {
		// Parse status from ConfigMap
		statusDataStr, ok := configMap.Data[E2NodeStatusKey]
		if !ok {
			continue
		}

		var statusData E2NodeStatusData
		if err := json.Unmarshal([]byte(statusDataStr), &statusData); err != nil {
			continue
		}

		// Convert to E2NodeStatus
		nodeStatus := nephoranv1.E2NodeStatus{
			NodeID:              statusData.NodeID,
			State:               nephoranv1.E2NodeLifecycleState(statusData.State),
			ActiveSubscriptions: statusData.ActiveSubscriptions,
			ErrorMessage:        statusData.ErrorMessage,
		}

		if statusData.LastHeartbeat != nil {
			nodeStatus.LastHeartbeat = &metav1.Time{Time: *statusData.LastHeartbeat}
		}

		if statusData.ConnectedSince != nil {
			nodeStatus.ConnectedSince = &metav1.Time{Time: *statusData.ConnectedSince}
		}

		e2NodeStatuses = append(e2NodeStatuses, nodeStatus)

		// Count ready and available replicas
		if statusData.State == string(nephoranv1.E2NodeLifecycleStateConnected) {
			readyReplicas++
			availableReplicas++
		} else if statusData.State == string(nephoranv1.E2NodeLifecycleStateInitializing) {
			availableReplicas++
		}
	}

	// Update status fields
	currentE2NodeSet.Status.CurrentReplicas = currentReplicas
	currentE2NodeSet.Status.ReadyReplicas = readyReplicas
	currentE2NodeSet.Status.AvailableReplicas = availableReplicas
	currentE2NodeSet.Status.UpdatedReplicas = currentReplicas // All replicas use current template
	currentE2NodeSet.Status.E2NodeStatuses = e2NodeStatuses
	currentE2NodeSet.Status.ObservedGeneration = currentE2NodeSet.Generation
	now := metav1.Now()
	currentE2NodeSet.Status.LastUpdateTime = &now

	// Update conditions
	r.updateE2NodeSetConditions(&currentE2NodeSet, readyReplicas, currentReplicas)

	// Update status
	return r.Status().Update(ctx, &currentE2NodeSet)
}

// updateE2NodeSetConditions updates the E2NodeSet conditions
func (r *E2NodeSetReconciler) updateE2NodeSetConditions(e2nodeSet *nephoranv1.E2NodeSet, readyReplicas, currentReplicas int32) {
	now := metav1.Now()

	// Available condition
	availableCondition := nephoranv1.E2NodeSetCondition{
		Type:               nephoranv1.E2NodeSetConditionAvailable,
		LastTransitionTime: now,
	}

	if readyReplicas >= e2nodeSet.Spec.Replicas {
		availableCondition.Status = metav1.ConditionTrue
		availableCondition.Reason = "MinimumReplicasAvailable"
		availableCondition.Message = "All desired replicas are ready"
	} else {
		availableCondition.Status = metav1.ConditionFalse
		availableCondition.Reason = "MinimumReplicasUnavailable"
		availableCondition.Message = fmt.Sprintf("Only %d of %d desired replicas are ready", readyReplicas, e2nodeSet.Spec.Replicas)
	}

	// Progressing condition
	progressingCondition := nephoranv1.E2NodeSetCondition{
		Type:               nephoranv1.E2NodeSetConditionProgressing,
		LastTransitionTime: now,
	}

	if currentReplicas != e2nodeSet.Spec.Replicas {
		progressingCondition.Status = metav1.ConditionTrue
		progressingCondition.Reason = "ScalingInProgress"
		progressingCondition.Message = fmt.Sprintf("Scaling from %d to %d replicas", currentReplicas, e2nodeSet.Spec.Replicas)
	} else if readyReplicas < currentReplicas {
		progressingCondition.Status = metav1.ConditionTrue
		progressingCondition.Reason = "ReplicasNotReady"
		progressingCondition.Message = fmt.Sprintf("Waiting for %d replicas to become ready", currentReplicas-readyReplicas)
	} else {
		progressingCondition.Status = metav1.ConditionFalse
		progressingCondition.Reason = "ReplicasUpToDate"
		progressingCondition.Message = "All replicas are up to date and ready"
	}

	// Update conditions
	r.setE2NodeSetCondition(e2nodeSet, availableCondition)
	r.setE2NodeSetCondition(e2nodeSet, progressingCondition)
}

// setE2NodeSetCondition sets a condition in the E2NodeSet status
func (r *E2NodeSetReconciler) setE2NodeSetCondition(e2nodeSet *nephoranv1.E2NodeSet, condition nephoranv1.E2NodeSetCondition) {
	// Find existing condition
	for i, existingCondition := range e2nodeSet.Status.Conditions {
		if existingCondition.Type == condition.Type {
			// Update existing condition if status changed
			if existingCondition.Status != condition.Status {
				condition.LastTransitionTime = metav1.Now()
			} else {
				condition.LastTransitionTime = existingCondition.LastTransitionTime
			}
			e2nodeSet.Status.Conditions[i] = condition
			return
		}
	}

	// Add new condition
	e2nodeSet.Status.Conditions = append(e2nodeSet.Status.Conditions, condition)
}

// handleDeletion handles the deletion of an E2NodeSet and cleanup of all associated ConfigMaps with exponential backoff
func (r *E2NodeSetReconciler) handleDeletion(ctx context.Context, e2nodeSet *nephoranv1.E2NodeSet) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Handling E2NodeSet deletion", "name", e2nodeSet.Name, "namespace", e2nodeSet.Namespace)

	// Get retry count for cleanup operations
	retryCount := getRetryCount(e2nodeSet, "cleanup")
	logger.V(1).Info("Cleanup retry count", "count", retryCount, "max", DefaultMaxRetries)

	// Get all ConfigMaps that belong to this E2NodeSet
	currentConfigMaps, err := r.getCurrentE2NodeConfigMaps(ctx, e2nodeSet)
	if err != nil {
		logger.Error(err, "Failed to get current E2 node ConfigMaps during deletion")
		// Continue with cleanup attempt even if we can't list ConfigMaps
	}

	// Delete all ConfigMaps belonging to this E2NodeSet
	cleanupErrors := []error{}
	for _, configMap := range currentConfigMaps {
		nodeID := r.getNodeIDFromConfigMap(configMap)
		logger.Info("Deleting E2 node ConfigMap during E2NodeSet deletion",
			"configMap", configMap.Name, "nodeID", nodeID)

		if err := r.Delete(ctx, configMap); err != nil {
			if !errors.IsNotFound(err) {
				logger.Error(err, "Failed to delete E2 node ConfigMap during deletion",
					"configMap", configMap.Name, "nodeID", nodeID)
				cleanupErrors = append(cleanupErrors, fmt.Errorf("failed to delete ConfigMap %s: %w", configMap.Name, err))
			}
		} else {
			logger.Info("Successfully deleted E2 node ConfigMap during deletion",
				"configMap", configMap.Name, "nodeID", nodeID)
		}
	}

	// If there were cleanup errors, handle retry logic
	if len(cleanupErrors) > 0 {
		if retryCount >= DefaultMaxRetries {
			// Max retries exceeded, remove finalizer to prevent stuck resources
			logger.Error(fmt.Errorf("max cleanup retries exceeded"),
				"Removing finalizer to prevent stuck resource after max retries",
				"errorCount", len(cleanupErrors), "maxRetries", DefaultMaxRetries)

			// Set Ready condition to indicate cleanup failure
			r.setReadyCondition(ctx, e2nodeSet, metav1.ConditionFalse, "CleanupFailedMaxRetries",
				fmt.Sprintf("Cleanup failed after %d retries, removing finalizer to prevent stuck resource", DefaultMaxRetries))

			// Remove finalizer even if cleanup failed to prevent stuck resources
			controllerutil.RemoveFinalizer(e2nodeSet, E2NodeSetFinalizer)
			if err := r.Update(ctx, e2nodeSet); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to remove finalizer after cleanup failure: %w", err)
			}

			r.Recorder.Eventf(e2nodeSet, corev1.EventTypeWarning, "CleanupFailedMaxRetries",
				"Cleanup failed after %d retries, finalizer removed to prevent stuck resource", DefaultMaxRetries)

			return ctrl.Result{}, nil
		}

		// Increment retry count and requeue with exponential backoff
		setRetryCount(e2nodeSet, "cleanup", retryCount+1)
		r.Update(ctx, e2nodeSet)

		// Set Ready condition to indicate cleanup retry
		r.setReadyCondition(ctx, e2nodeSet, metav1.ConditionFalse, "CleanupRetrying",
			fmt.Sprintf("Cleanup failed, retrying (attempt %d/%d): %d errors occurred", retryCount+1, DefaultMaxRetries, len(cleanupErrors)))

		r.Recorder.Eventf(e2nodeSet, corev1.EventTypeWarning, "CleanupRetry",
			"Cleanup failed, retrying (attempt %d/%d): %d ConfigMaps failed to delete", retryCount+1, DefaultMaxRetries, len(cleanupErrors))

		backoffDelay := calculateExponentialBackoffForOperation(retryCount, "cleanup")
		logger.V(1).Info("Scheduling cleanup retry with exponential backoff",
			"delay", backoffDelay,
			"attempt", retryCount+1,
			"max_retries", DefaultMaxRetries)

		return ctrl.Result{RequeueAfter: backoffDelay}, fmt.Errorf("cleanup errors occurred for %d ConfigMaps, retrying", len(cleanupErrors))
	}

	// All cleanup successful, clear retry count and remove finalizer
	clearRetryCount(e2nodeSet, "cleanup")
	controllerutil.RemoveFinalizer(e2nodeSet, E2NodeSetFinalizer)
	if err := r.Update(ctx, e2nodeSet); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to remove finalizer: %w", err)
	}

	logger.Info("Successfully completed E2NodeSet deletion cleanup",
		"name", e2nodeSet.Name, "cleanedConfigMaps", len(currentConfigMaps))

	// Record event for successful deletion
	r.Recorder.Eventf(e2nodeSet, corev1.EventTypeNormal, "E2NodeSetDeleted",
		"Successfully deleted E2NodeSet and %d E2 nodes", len(currentConfigMaps))

	return ctrl.Result{}, nil
}
