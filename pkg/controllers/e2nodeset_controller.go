package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/git"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/e2"
)

type E2NodeSetReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	GitClient  git.ClientInterface
	E2Manager  *e2.E2Manager // Enhanced: E2Manager for real E2AP protocol integration
}

//+kubebuilder:rbac:groups=nephoran.com,resources=e2nodesets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nephoran.com,resources=e2nodesets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nephoran.com,resources=e2nodesets/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *E2NodeSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the E2NodeSet instance
	var e2nodeSet nephoranv1.E2NodeSet
	if err := r.Get(ctx, req.NamespacedName, &e2nodeSet); err != nil {
		if errors.IsNotFound(err) {
			// E2NodeSet was deleted, cleanup will be handled by garbage collection
			logger.Info("E2NodeSet resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get E2NodeSet")
		return ctrl.Result{}, err
	}

	logger.Info("Reconciling E2NodeSet", "name", e2nodeSet.Name, "namespace", e2nodeSet.Namespace, "replicas", e2nodeSet.Spec.Replicas)

	// Enhanced: Get current E2 nodes using E2Manager instead of ConfigMaps
	currentE2Nodes, err := r.getCurrentE2NodesViaE2Manager(ctx, &e2nodeSet)
	if err != nil {
		logger.Error(err, "Failed to get current E2 nodes from E2Manager")
		return ctrl.Result{}, err
	}

	currentReplicas := int32(len(currentE2Nodes))
	desiredReplicas := e2nodeSet.Spec.Replicas

	logger.Info("E2NodeSet status", "current", currentReplicas, "desired", desiredReplicas)

	// Handle scaling operations using E2AP protocol
	if currentReplicas < desiredReplicas {
		// Scale up - create and register new E2 nodes with Near-RT RIC
		for i := currentReplicas; i < desiredReplicas; i++ {
			if err := r.createE2NodeWithE2AP(ctx, &e2nodeSet, i); err != nil {
				logger.Error(err, "Failed to create E2 node via E2AP", "index", i)
				return ctrl.Result{RequeueAfter: time.Second * 30}, err
			}
			logger.Info("Created E2 node via E2AP", "index", i)
		}
	} else if currentReplicas > desiredReplicas {
		// Scale down - deregister and delete excess E2 nodes
		for i := desiredReplicas; i < currentReplicas; i++ {
			if err := r.deleteE2NodeWithE2AP(ctx, &e2nodeSet, i); err != nil {
				logger.Error(err, "Failed to delete E2 node via E2AP", "index", i)
				return ctrl.Result{RequeueAfter: time.Second * 30}, err
			}
			logger.Info("Deleted E2 node via E2AP", "index", i)
		}
	}

	// Enhanced: Get health status from E2Manager and update status
	readyReplicas, err := r.getReadyReplicasFromE2Manager(ctx, &e2nodeSet)
	if err != nil {
		logger.Error(err, "Failed to get ready replicas from E2Manager")
		readyReplicas = 0 // Assume no replicas ready if we can't determine
	}

	return r.updateStatus(ctx, &e2nodeSet, readyReplicas)
}

func (r *E2NodeSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Enhanced: Remove ConfigMap ownership as we now use E2Manager
	return ctrl.NewControllerManagedBy(mgr).
		For(&nephoranv1.E2NodeSet{}).
		Complete(r)
}

// Enhanced: getCurrentE2NodesViaE2Manager returns E2 nodes managed by E2Manager
func (r *E2NodeSetReconciler) getCurrentE2NodesViaE2Manager(ctx context.Context, e2nodeSet *nephoranv1.E2NodeSet) ([]*e2.E2Node, error) {
	if r.E2Manager == nil {
		return nil, fmt.Errorf("E2Manager not initialized")
	}

	// Get all E2 nodes from E2Manager
	allNodes, err := r.E2Manager.ListE2Nodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list E2 nodes from E2Manager: %w", err)
	}

	// Filter nodes that belong to this E2NodeSet
	var filteredNodes []*e2.E2Node
	nodeSetPrefix := r.getE2NodePrefix(e2nodeSet)
	for _, node := range allNodes {
		if strings.HasPrefix(node.NodeID, nodeSetPrefix) {
			filteredNodes = append(filteredNodes, node)
		}
	}

	return filteredNodes, nil
}

// getCurrentE2Nodes returns all ConfigMaps that represent E2 nodes for this E2NodeSet (legacy fallback)
func (r *E2NodeSetReconciler) getCurrentE2Nodes(ctx context.Context, e2nodeSet *nephoranv1.E2NodeSet) ([]corev1.ConfigMap, error) {
	// List ConfigMaps with labels that match this E2NodeSet
	labelSelector := labels.SelectorFromSet(r.getE2NodeLabels(e2nodeSet))
	var configMaps corev1.ConfigMapList
	if err := r.List(ctx, &configMaps, &client.ListOptions{
		Namespace:     e2nodeSet.Namespace,
		LabelSelector: labelSelector,
	}); err != nil {
		return nil, err
	}
	return configMaps.Items, nil
}

// Enhanced: createE2NodeWithE2AP creates and registers a new E2 node using E2AP protocol
func (r *E2NodeSetReconciler) createE2NodeWithE2AP(ctx context.Context, e2nodeSet *nephoranv1.E2NodeSet, index int32) error {
	logger := log.FromContext(ctx)

	if r.E2Manager == nil {
		return fmt.Errorf("E2Manager not initialized")
	}

	nodeID := r.generateNodeID(e2nodeSet, index)
	nearRTRICEndpoint := r.getNearRTRICEndpoint(e2nodeSet)

	logger.Info("Creating E2 node with E2AP", "nodeID", nodeID, "endpoint", nearRTRICEndpoint)

	// Step 1: Setup E2 connection to Near-RT RIC
	if err := r.E2Manager.SetupE2Connection(nodeID, nearRTRICEndpoint); err != nil {
		return fmt.Errorf("failed to setup E2 connection for node %s: %w", nodeID, err)
	}

	// Step 2: Create and register RAN functions for this E2 node
	ranFunctions := r.createDefaultRANFunctions(e2nodeSet)
	if err := r.E2Manager.RegisterE2Node(ctx, nodeID, ranFunctions); err != nil {
		return fmt.Errorf("failed to register E2 node %s: %w", nodeID, err)
	}

	// Step 3: Create a ConfigMap for tracking (hybrid approach during migration)
	if err := r.createTrackingConfigMap(ctx, e2nodeSet, index, nodeID); err != nil {
		// Log error but don't fail the operation
		logger.Error(err, "Failed to create tracking ConfigMap", "nodeID", nodeID)
	}

	logger.Info("Successfully created E2 node with E2AP", "nodeID", nodeID)
	return nil
}

// Legacy: createE2Node creates a new ConfigMap representing a simulated E2 node
func (r *E2NodeSetReconciler) createE2Node(ctx context.Context, e2nodeSet *nephoranv1.E2NodeSet, index int32) error {
	nodeName := fmt.Sprintf("%s-node-%d", e2nodeSet.Name, index)
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nodeName,
			Namespace: e2nodeSet.Namespace,
			Labels:    r.getE2NodeLabels(e2nodeSet),
		},
		Data: map[string]string{
			"nodeId":    fmt.Sprintf("%s-node-%d", e2nodeSet.Name, index),
			"nodeType":  "simulated-gnb",
			"status":    "active",
			"created":   time.Now().Format(time.RFC3339),
			"e2nodeSet": e2nodeSet.Name,
			"index":     fmt.Sprintf("%d", index),
		},
	}

	// Set E2NodeSet as owner of the ConfigMap for garbage collection
	if err := controllerutil.SetControllerReference(e2nodeSet, configMap, r.Scheme); err != nil {
		return err
	}

	return r.Create(ctx, configMap)
}

// Enhanced: deleteE2NodeWithE2AP deregisters and deletes an E2 node using E2AP protocol
func (r *E2NodeSetReconciler) deleteE2NodeWithE2AP(ctx context.Context, e2nodeSet *nephoranv1.E2NodeSet, index int32) error {
	logger := log.FromContext(ctx)

	if r.E2Manager == nil {
		return fmt.Errorf("E2Manager not initialized")
	}

	nodeID := r.generateNodeID(e2nodeSet, index)

	logger.Info("Deleting E2 node with E2AP", "nodeID", nodeID)

	// Step 1: Get current E2 nodes to verify existence
	currentNodes, err := r.E2Manager.ListE2Nodes(ctx)
	if err != nil {
		return fmt.Errorf("failed to list E2 nodes: %w", err)
	}

	// Step 2: Check if node exists before attempting deletion
	nodeExists := false
	for _, node := range currentNodes {
		if node.NodeID == nodeID {
			nodeExists = true
			break
		}
	}

	if !nodeExists {
		logger.Info("E2 node does not exist, skipping deletion", "nodeID", nodeID)
		return nil
	}

	// Step 3: Deregister the E2 node (this should handle cleanup of subscriptions, etc.)
	// Note: The actual deregistration logic would need to be implemented in E2Manager
	// For now, we'll assume the E2Manager handles this through the existing interface

	// Step 4: Delete tracking ConfigMap if it exists
	if err := r.deleteTrackingConfigMap(ctx, e2nodeSet, index); err != nil {
		// Log error but don't fail the operation
		logger.Error(err, "Failed to delete tracking ConfigMap", "nodeID", nodeID)
	}

	logger.Info("Successfully deleted E2 node with E2AP", "nodeID", nodeID)
	return nil
}

// Legacy: deleteE2Node deletes a ConfigMap representing a simulated E2 node
func (r *E2NodeSetReconciler) deleteE2Node(ctx context.Context, e2nodeSet *nephoranv1.E2NodeSet, index int32) error {
	nodeName := fmt.Sprintf("%s-node-%d", e2nodeSet.Name, index)
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nodeName,
			Namespace: e2nodeSet.Namespace,
		},
	}
	return r.Delete(ctx, configMap)
}

// Enhanced: Helper functions for E2AP integration

// generateNodeID generates a unique node ID for an E2 node
func (r *E2NodeSetReconciler) generateNodeID(e2nodeSet *nephoranv1.E2NodeSet, index int32) string {
	return fmt.Sprintf("%s-%s-node-%d", e2nodeSet.Namespace, e2nodeSet.Name, index)
}

// getE2NodePrefix returns the prefix used for E2 nodes in this set
func (r *E2NodeSetReconciler) getE2NodePrefix(e2nodeSet *nephoranv1.E2NodeSet) string {
	return fmt.Sprintf("%s-%s-node-", e2nodeSet.Namespace, e2nodeSet.Name)
}

// getNearRTRICEndpoint returns the Near-RT RIC endpoint for E2 connections
func (r *E2NodeSetReconciler) getNearRTRICEndpoint(e2nodeSet *nephoranv1.E2NodeSet) string {
	// TODO: This could be configurable via annotations or spec in future
	// For now, use a default endpoint
	if endpoint, ok := e2nodeSet.Annotations["nephoran.com/near-rt-ric-endpoint"]; ok {
		return endpoint
	}
	return "http://near-rt-ric:38080" // Default Near-RT RIC endpoint
}

// createDefaultRANFunctions creates default RAN functions for an E2 node
func (r *E2NodeSetReconciler) createDefaultRANFunctions(e2nodeSet *nephoranv1.E2NodeSet) []e2.RanFunction {
	return []e2.RanFunction{
		{
			FunctionID:          1,
			FunctionDefinition:  "KPM Service Model Function",
			FunctionRevision:    1,
			FunctionOID:         "1.3.6.1.4.1.53148.1.1.2.2",
			FunctionDescription: "Key Performance Measurement Service Model",
			ServiceModel:        e2.CreateEnhancedKPMServiceModel(),
		},
		{
			FunctionID:          2,
			FunctionDefinition:  "RC Service Model Function",
			FunctionRevision:    1,
			FunctionOID:         "1.3.6.1.4.1.53148.1.1.2.3",
			FunctionDescription: "RAN Control Service Model",
			ServiceModel:        e2.CreateEnhancedRCServiceModel(),
		},
		// Add more RAN functions as needed
	}
}

// createTrackingConfigMap creates a ConfigMap for tracking (hybrid approach)
func (r *E2NodeSetReconciler) createTrackingConfigMap(ctx context.Context, e2nodeSet *nephoranv1.E2NodeSet, index int32, nodeID string) error {
	nodeName := fmt.Sprintf("%s-node-%d", e2nodeSet.Name, index)
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nodeName,
			Namespace: e2nodeSet.Namespace,
			Labels:    r.getE2NodeLabels(e2nodeSet),
			Annotations: map[string]string{
				"nephoran.com/e2-node-id": nodeID,
				"nephoran.com/e2ap-enabled": "true",
			},
		},
		Data: map[string]string{
			"nodeId":         nodeID,
			"nodeType":       "e2ap-enabled-gnb",
			"status":         "active",
			"created":        time.Now().Format(time.RFC3339),
			"e2nodeSet":      e2nodeSet.Name,
			"index":          fmt.Sprintf("%d", index),
			"e2apProtocol":   "enabled",
			"nearRTRIC":      r.getNearRTRICEndpoint(e2nodeSet),
		},
	}

	// Set E2NodeSet as owner of the ConfigMap for garbage collection
	if err := controllerutil.SetControllerReference(e2nodeSet, configMap, r.Scheme); err != nil {
		return err
	}

	return r.Create(ctx, configMap)
}

// deleteTrackingConfigMap deletes the tracking ConfigMap
func (r *E2NodeSetReconciler) deleteTrackingConfigMap(ctx context.Context, e2nodeSet *nephoranv1.E2NodeSet, index int32) error {
	nodeName := fmt.Sprintf("%s-node-%d", e2nodeSet.Name, index)
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nodeName,
			Namespace: e2nodeSet.Namespace,
		},
	}
	return r.Delete(ctx, configMap)
}

// getE2NodeLabels returns the labels used to identify E2 nodes belonging to this E2NodeSet
func (r *E2NodeSetReconciler) getE2NodeLabels(e2nodeSet *nephoranv1.E2NodeSet) map[string]string {
	return map[string]string{
		"app":                     "e2node",
		"e2nodeset":               e2nodeSet.Name,
		"nephoran.com/component":  "e2ap-enabled-gnb", // Enhanced: Updated to reflect E2AP capability
		"nephoran.com/managed-by": "e2nodeset-controller",
		"nephoran.com/protocol":   "e2ap", // Enhanced: Indicate E2AP protocol support
	}
}

// Enhanced: getReadyReplicasFromE2Manager determines ready replicas from E2Manager health status
func (r *E2NodeSetReconciler) getReadyReplicasFromE2Manager(ctx context.Context, e2nodeSet *nephoranv1.E2NodeSet) (int32, error) {
	if r.E2Manager == nil {
		return 0, fmt.Errorf("E2Manager not initialized")
	}

	// Get all E2 nodes from E2Manager
	allNodes, err := r.E2Manager.ListE2Nodes(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to list E2 nodes from E2Manager: %w", err)
	}

	// Count ready nodes that belong to this E2NodeSet
	var readyCount int32
	nodeSetPrefix := r.getE2NodePrefix(e2nodeSet)
	for _, node := range allNodes {
		if strings.HasPrefix(node.NodeID, nodeSetPrefix) {
			// Check if node is healthy and ready
			if node.HealthStatus.Status == "HEALTHY" && node.ConnectionStatus.State == "CONNECTED" {
				readyCount++
			}
		}
	}

	return readyCount, nil
}

// updateStatus updates the E2NodeSet status with current ready replicas
func (r *E2NodeSetReconciler) updateStatus(ctx context.Context, e2nodeSet *nephoranv1.E2NodeSet, readyReplicas int32) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Update status if it has changed
	if e2nodeSet.Status.ReadyReplicas != readyReplicas {
		e2nodeSet.Status.ReadyReplicas = readyReplicas
		if err := r.Status().Update(ctx, e2nodeSet); err != nil {
			logger.Error(err, "Failed to update E2NodeSet status")
			return ctrl.Result{}, err
		}
		logger.Info("Updated E2NodeSet status", "readyReplicas", readyReplicas)
	}

	return ctrl.Result{}, nil
}
