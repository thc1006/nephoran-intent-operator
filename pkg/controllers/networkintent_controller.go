package controllers

import (
	"context"
	"encoding/json"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1alpha1 "nephoran-intent-operator/pkg/apis/nephoran/v1alpha1"
)

const (
	// typeAvailableNetworkIntent represents the status of the Deployment reconciliation.
	typeAvailableNetworkIntent = "Available"
	// typeDegradedNetworkIntent represents the status of the Deployment reconciliation.
	typeDegradedNetworkIntent = "Degraded"
)

// NetworkIntentReconciler reconciles a NetworkIntent object
type NetworkIntentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=nephoran.com,resources=networkintents,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nephoran.com,resources=networkintents/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nephoran.com,resources=networkintents/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

func (r *NetworkIntentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	intent := &nephoranv1alpha1.NetworkIntent{}
	if err := r.Get(ctx, req.NamespacedName, intent); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("NetworkIntent resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get NetworkIntent")
		return ctrl.Result{}, err
	}

	meta.SetStatusCondition(&intent.Status.Conditions, metav1.Condition{Type: typeAvailableNetworkIntent, Status: metav1.ConditionUnknown, Reason: "Reconciling", Message: "Starting reconciliation"})
	if err := r.Status().Update(ctx, intent); err != nil {
		logger.Error(err, "Failed to update NetworkIntent status")
		return ctrl.Result{}, err
	}

	var deploymentIntent NetworkFunctionDeploymentIntent
	if err := json.Unmarshal(intent.Spec.Parameters.Raw, &deploymentIntent); err != nil {
		logger.Error(err, "Failed to unmarshal intent parameters")
		meta.SetStatusCondition(&intent.Status.Conditions, metav1.Condition{Type: typeDegradedNetworkIntent, Status: metav1.ConditionTrue, Reason: "InvalidParameters", Message: err.Error()})
		if err := r.Status().Update(ctx, intent); err != nil {
			logger.Error(err, "Failed to update NetworkIntent status")
		}
		return ctrl.Result{}, err
	}

	if deploymentIntent.Type != "NetworkFunctionDeployment" {
		logger.Info("Intent type is not NetworkFunctionDeployment, ignoring", "type", deploymentIntent.Type)
		return ctrl.Result{}, nil
	}

	deployment := r.deploymentForIntent(&deploymentIntent)
	if err := ctrl.SetControllerReference(intent, deployment, r.Scheme); err != nil {
		logger.Error(err, "Failed to set controller reference on Deployment")
		return ctrl.Result{}, err
	}

	found := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		err = r.Create(ctx, deployment)
		if err != nil {
			logger.Error(err, "Failed to create new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
			meta.SetStatusCondition(&intent.Status.Conditions, metav1.Condition{Type: typeDegradedNetworkIntent, Status: metav1.ConditionTrue, Reason: "DeploymentFailed", Message: "Failed to create Deployment"})
			if err := r.Status().Update(ctx, intent); err != nil {
				logger.Error(err, "Failed to update NetworkIntent status")
			}
			return ctrl.Result{}, err
		}
		meta.SetStatusCondition(&intent.Status.Conditions, metav1.Condition{Type: typeAvailableNetworkIntent, Status: metav1.ConditionFalse, Reason: "Creating", Message: "Deployment created successfully"})
		if err := r.Status().Update(ctx, intent); err != nil {
			logger.Error(err, "Failed to update NetworkIntent status")
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		logger.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	if !reflect.DeepEqual(deployment.Spec, found.Spec) {
		found.Spec = deployment.Spec
		logger.Info("Updating Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
		err = r.Update(ctx, found)
		if err != nil {
			logger.Error(err, "Failed to update Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return ctrl.Result{}, err
		}
	}

	if found.Status.AvailableReplicas == *found.Spec.Replicas {
		meta.SetStatusCondition(&intent.Status.Conditions, metav1.Condition{Type: typeAvailableNetworkIntent, Status: metav1.ConditionTrue, Reason: "Ready", Message: "Deployment is ready"})
	} else {
		meta.SetStatusCondition(&intent.Status.Conditions, metav1.Condition{Type: typeAvailableNetworkIntent, Status: metav1.ConditionFalse, Reason: "Progressing", Message: "Deployment is not ready yet"})
	}
	meta.SetStatusCondition(&intent.Status.Conditions, metav1.Condition{Type: typeDegradedNetworkIntent, Status: metav1.ConditionFalse, Reason: "Reconciled", Message: "Deployment reconciled successfully"})

	if err := r.Status().Update(ctx, intent); err != nil {
		logger.Error(err, "Failed to update NetworkIntent status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *NetworkIntentReconciler) deploymentForIntent(intent *NetworkFunctionDeploymentIntent) *appsv1.Deployment {
	replicas := int32(intent.Spec.Replicas)
	labels := map[string]string{"app": intent.Name}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      intent.Name,
			Namespace: intent.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: intent.Spec.Image,
						Name:  intent.Name,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse(intent.Spec.Resources.Requests.CPU),
								corev1.ResourceMemory: resource.MustParse(intent.Spec.Resources.Requests.Memory),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse(intent.Spec.Resources.Limits.CPU),
								corev1.ResourceMemory: resource.MustParse(intent.Spec.Resources.Limits.Memory),
							},
						},
					}},
				},
			},
		},
	}
}

func (r *NetworkIntentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nephoranv1alpha1.NetworkIntent{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}