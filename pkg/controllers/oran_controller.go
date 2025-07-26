package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// OranAdaptorReconciler reconciles a OranAdaptor object
type OranAdaptorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=*,resources=*,verbs=*

func (r *OranAdaptorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// This controller will act as an orchestrator for the different O-RAN interface adaptors.
	// Its responsibilities will include:
	// 1. Watching for relevant KRM resources (e.g., NetworkIntent, ManagedElement, etc.).
	// 2. Determining which O-RAN interface should handle the resource.
	// 3. Delegating the handling of the resource to the appropriate interface-specific adaptor.

	return ctrl.Result{}, nil
}

func (r *OranAdaptorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// TODO: Specify which resources this controller should watch.
		Complete(r)
}