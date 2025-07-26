package controllers

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/a1"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o1"
)

const (
	typeReadyManagedElement  = "Ready"
	O1ConfiguredCondition    = "O1Configured"
	A1PolicyAppliedCondition = "A1PolicyApplied"
)

// OranAdaptorReconciler reconciles a ManagedElement object
type OranAdaptorReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	O1Adaptor *o1.O1Adaptor
	A1Adaptor *a1.A1Adaptor
}

//+kubebuilder:rbac:groups=nephoran.nephoran.io,resources=managedelements,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nephoran.nephoran.io,resources=managedelements/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch

func (r *OranAdaptorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	me := &nephoranv1alpha1.ManagedElement{}
	if err := r.Get(ctx, req.NamespacedName, me); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Reconciling ManagedElement", "name", me.Name)

	// O2 Logic: Check the status of the associated Deployment
	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: me.Spec.DeploymentName, Namespace: me.Namespace}, deployment)
	if err != nil {
		logger.Error(err, "Failed to get associated Deployment", "DeploymentName", me.Spec.DeploymentName)
		meta.SetStatusCondition(&me.Status.Conditions, metav1.Condition{Type: typeReadyManagedElement, Status: metav1.ConditionFalse, Reason: "DeploymentNotFound", Message: err.Error()})
		return r.updateStatus(ctx, me)
	}

	isReady := deployment.Status.AvailableReplicas == *deployment.Spec.Replicas
	if !isReady {
		meta.SetStatusCondition(&me.Status.Conditions, metav1.Condition{Type: typeReadyManagedElement, Status: metav1.ConditionFalse, Reason: "Progressing", Message: "Deployment is not yet fully available."})
		return r.updateStatus(ctx, me)
	}
	meta.SetStatusCondition(&me.Status.Conditions, metav1.Condition{Type: typeReadyManagedElement, Status: metav1.ConditionTrue, Reason: "Ready", Message: "Deployment is fully available."})

	// O1 Logic: If ready, apply intent-driven O1 configuration
	if me.Spec.O1Config != "" {
		if err := r.O1Adaptor.ApplyConfiguration(ctx, me); err != nil {
			logger.Error(err, "O1 configuration failed")
			meta.SetStatusCondition(&me.Status.Conditions, metav1.Condition{Type: O1ConfiguredCondition, Status: metav1.ConditionFalse, Reason: "Failed", Message: err.Error()})
		} else {
			logger.Info("O1 configuration applied successfully", "ManagedElement", me.Name)
			meta.SetStatusCondition(&me.Status.Conditions, metav1.Condition{Type: O1ConfiguredCondition, Status: metav1.ConditionTrue, Reason: "Success", Message: "O1 configuration applied."})
		}
	}

	// A1 Logic: If ready, apply intent-driven A1 policy
	if me.Spec.A1Policy.Raw != nil {
		if err := r.A1Adaptor.ApplyPolicy(ctx, me); err != nil {
			logger.Error(err, "A1 policy application failed")
			meta.SetStatusCondition(&me.Status.Conditions, metav1.Condition{Type: A1PolicyAppliedCondition, Status: metav1.ConditionFalse, Reason: "Failed", Message: err.Error()})
		} else {
			logger.Info("A1 policy applied successfully", "ManagedElement", me.Name)
			meta.SetStatusCondition(&me.Status.Conditions, metav1.Condition{Type: A1PolicyAppliedCondition, Status: metav1.ConditionTrue, Reason: "Success", Message: "A1 policy applied."})
		}
	}

	return r.updateStatus(ctx, me)
}

func (r *OranAdaptorReconciler) updateStatus(ctx context.Context, me *nephoranv1alpha1.ManagedElement) (ctrl.Result, error) {
	if err := r.Status().Update(ctx, me); err != nil {
		log.FromContext(ctx).Error(err, "Failed to update ManagedElement status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *OranAdaptorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.O1Adaptor = o1.NewO1Adaptor()
	r.A1Adaptor = a1.NewA1Adaptor()
	return ctrl.NewControllerManagedBy(mgr).
		For(&nephoranv1alpha1.ManagedElement{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
