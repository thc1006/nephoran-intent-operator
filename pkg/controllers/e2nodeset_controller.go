package controllers

import (
	"context"
	"fmt"
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
)

type E2NodeSetReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	GitClient git.ClientInterface
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

	// Get current simulated E2 nodes (ConfigMaps)
	currentNodes, err := r.getCurrentE2Nodes(ctx, &e2nodeSet)
	if err != nil {
		logger.Error(err, "Failed to get current E2 nodes")
		return ctrl.Result{}, err
	}

	currentReplicas := int32(len(currentNodes))
	desiredReplicas := e2nodeSet.Spec.Replicas

	logger.Info("E2NodeSet status", "current", currentReplicas, "desired", desiredReplicas)

	// Handle scaling operations
	if currentReplicas < desiredReplicas {
		// Scale up - create new E2 nodes
		for i := currentReplicas; i < desiredReplicas; i++ {
			if err := r.createE2Node(ctx, &e2nodeSet, i); err != nil {
				logger.Error(err, "Failed to create E2 node", "index", i)
				return ctrl.Result{RequeueAfter: time.Second * 30}, err
			}
			logger.Info("Created E2 node", "index", i)
		}
	} else if currentReplicas > desiredReplicas {
		// Scale down - delete excess E2 nodes
		for i := desiredReplicas; i < currentReplicas; i++ {
			if err := r.deleteE2Node(ctx, &e2nodeSet, i); err != nil {
				logger.Error(err, "Failed to delete E2 node", "index", i)
				return ctrl.Result{RequeueAfter: time.Second * 30}, err
			}
			logger.Info("Deleted E2 node", "index", i)
		}
	}

	// Update status with current ready replicas
	return r.updateStatus(ctx, &e2nodeSet, desiredReplicas)
}

func (r *E2NodeSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nephoranv1.E2NodeSet{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

// getCurrentE2Nodes returns all ConfigMaps that represent E2 nodes for this E2NodeSet
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

// createE2Node creates a new ConfigMap representing a simulated E2 node
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

// deleteE2Node deletes a ConfigMap representing a simulated E2 node
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

// getE2NodeLabels returns the labels used to identify E2 nodes belonging to this E2NodeSet
func (r *E2NodeSetReconciler) getE2NodeLabels(e2nodeSet *nephoranv1.E2NodeSet) map[string]string {
	return map[string]string{
		"app":                     "e2node",
		"e2nodeset":               e2nodeSet.Name,
		"nephoran.com/component":  "simulated-gnb",
		"nephoran.com/managed-by": "e2nodeset-controller",
	}
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
