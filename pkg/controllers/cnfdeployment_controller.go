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

package controllers

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/cnf"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"
	"github.com/thc1006/nephoran-intent-operator/pkg/performance"
	"github.com/thc1006/nephoran-intent-operator/pkg/rag"
)

const (
	CNFDeploymentControllerName = "cnfdeployment-controller"
	CNFReconcileInterval        = 30 * time.Second
	CNFStatusUpdateInterval     = 10 * time.Second
)

// CNFDeploymentReconciler reconciles a CNFDeployment object
type CNFDeploymentReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	Recorder         record.EventRecorder
	CNFOrchestrator  *cnf.CNFOrchestrator
	LLMProcessor     *llm.Processor
	RAGService       *rag.Service
	MetricsCollector *performance.MetricsCollector
	MonitoringSystem *monitoring.SystemInstrumentation
	Config           *CNFControllerConfig
}

// CNFControllerConfig holds configuration for the CNF controller
type CNFControllerConfig struct {
	// DefaultHelmRepository is the default Helm repository URL
	DefaultHelmRepository string

	// DefaultHelmTimeout is the default timeout for Helm operations
	DefaultHelmTimeout time.Duration

	// EnableAutoScaling enables auto-scaling features
	EnableAutoScaling bool

	// ResourceQuota defines default resource quotas
	ResourceQuota ResourceQuotaConfig
}

// ResourceQuotaConfig defines resource quota limits
type ResourceQuotaConfig struct {
	// MaxCPU is the maximum CPU per CNF deployment
	MaxCPU string

	// MaxMemory is the maximum memory per CNF deployment
	MaxMemory string

	// MaxReplicas is the maximum number of replicas
	MaxReplicas int32
}

// DeploymentResult contains the result of a deployment operation
type DeploymentResult struct {
	Success   bool
	Error     error
	Duration  time.Duration
	Resources []string
}

//+kubebuilder:rbac:groups=nephoran.nephio.org,resources=cnfdeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nephoran.nephio.org,resources=cnfdeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nephoran.nephio.org,resources=cnfdeployments/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *CNFDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("cnfdeployment", req.NamespacedName)

	// Fetch the CNFDeployment instance
	var cnfDeployment nephoranv1.CNFDeployment
	if err := r.Get(ctx, req.NamespacedName, &cnfDeployment); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("CNFDeployment resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get CNFDeployment")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !cnfDeployment.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &cnfDeployment)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&cnfDeployment, "cnfdeployment.nephoran.nephio.org/finalizer") {
		controllerutil.AddFinalizer(&cnfDeployment, "cnfdeployment.nephoran.nephio.org/finalizer")
		if err := r.Update(ctx, &cnfDeployment); err != nil {
			logger.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Validate CNF deployment
	if err := cnfDeployment.ValidateCNFDeployment(); err != nil {
		logger.Error(err, "CNFDeployment validation failed")
		r.updateCNFDeploymentStatus(ctx, &cnfDeployment, "Failed", fmt.Sprintf("Validation failed: %v", err))
		return ctrl.Result{RequeueAfter: CNFReconcileInterval}, nil
	}

	// Check if deployment is ready
	isReady, err := r.checkDeploymentReadiness(ctx, &cnfDeployment)
	if err != nil {
		logger.Error(err, "Failed to check deployment readiness")
		r.updateCNFDeploymentStatus(ctx, &cnfDeployment, "Failed", fmt.Sprintf("Readiness check failed: %v", err))
		return ctrl.Result{RequeueAfter: CNFReconcileInterval}, err
	}

	if isReady {
		logger.Info("CNF deployment is already ready")
		r.updateCNFDeploymentStatus(ctx, &cnfDeployment, "Running", "CNF deployment is running")
		return ctrl.Result{RequeueAfter: CNFReconcileInterval}, nil
	}

	// Update status to indicate deployment is starting
	r.updateCNFDeploymentStatus(ctx, &cnfDeployment, "Deploying", "Starting CNF deployment")

	// Deploy the CNF
	result, err := r.deployCNF(ctx, &cnfDeployment)
	if err != nil {
		logger.Error(err, "Failed to deploy CNF")
		r.updateCNFDeploymentStatus(ctx, &cnfDeployment, "Failed", fmt.Sprintf("Deployment failed: %v", err))
		r.Recorder.Event(&cnfDeployment, "Warning", "DeploymentFailed", fmt.Sprintf("CNF deployment failed: %v", err))
		return ctrl.Result{RequeueAfter: CNFReconcileInterval}, err
	}

	logger.Info("CNF deployment completed successfully", "duration", result.Duration)

	// Update status to running
	r.updateCNFDeploymentStatus(ctx, &cnfDeployment, "Running", "CNF deployment completed successfully")
	r.Recorder.Event(&cnfDeployment, "Normal", "DeploymentCompleted", "CNF deployment completed successfully")

	// Record metrics
	if r.MetricsCollector != nil {
		r.MetricsCollector.RecordCNFDeployment(string(cnfDeployment.Spec.Function), result.Duration)
	}

	// Record metrics in monitoring system if available
	if r.MonitoringSystem != nil {
		r.MonitoringSystem.RecordCNFDeployment(string(cnfDeployment.Spec.Function), result.Duration)
	}

	return ctrl.Result{RequeueAfter: CNFReconcileInterval}, nil
}

// handleDeletion handles the cleanup when a CNFDeployment is being deleted
func (r *CNFDeploymentReconciler) handleDeletion(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("cnfdeployment", cnfDeployment.Name)

	// Perform cleanup operations
	if err := r.cleanupCNFResources(ctx, cnfDeployment); err != nil {
		logger.Error(err, "Failed to cleanup CNF resources")
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	// Remove the finalizer
	controllerutil.RemoveFinalizer(cnfDeployment, "cnfdeployment.nephoran.nephio.org/finalizer")
	if err := r.Update(ctx, cnfDeployment); err != nil {
		logger.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully cleaned up CNFDeployment")
	return ctrl.Result{}, nil
}

// cleanupCNFResources removes all resources associated with the CNF deployment
func (r *CNFDeploymentReconciler) cleanupCNFResources(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) error {
	logger := log.FromContext(ctx).WithValues("cnfdeployment", cnfDeployment.Name)

	// Clean up based on deployment strategy
	switch cnfDeployment.Spec.DeploymentStrategy {
	case nephoranv1.DeploymentStrategyHelm:
		return r.cleanupHelmDeployment(ctx, cnfDeployment)
	case nephoranv1.DeploymentStrategyOperator:
		return r.cleanupOperatorDeployment(ctx, cnfDeployment)
	case nephoranv1.DeploymentStrategyDirect:
		return r.cleanupDirectDeployment(ctx, cnfDeployment)
	case nephoranv1.DeploymentStrategyGitOps:
		return r.cleanupGitOpsDeployment(ctx, cnfDeployment)
	default:
		logger.Info("Unknown deployment strategy, skipping cleanup", "strategy", cnfDeployment.Spec.DeploymentStrategy)
		return nil
	}
}

// deployCNF deploys the CNF based on the specified strategy
func (r *CNFDeploymentReconciler) deployCNF(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) (*DeploymentResult, error) {
	startTime := time.Now()
	result := &DeploymentResult{}

	// Deploy based on strategy
	switch cnfDeployment.Spec.DeploymentStrategy {
	case nephoranv1.DeploymentStrategyHelm:
		err := r.deployWithHelm(ctx, cnfDeployment)
		result.Success = err == nil
		result.Error = err
	case nephoranv1.DeploymentStrategyOperator:
		err := r.deployWithOperator(ctx, cnfDeployment)
		result.Success = err == nil
		result.Error = err
	case nephoranv1.DeploymentStrategyDirect:
		err := r.deployDirect(ctx, cnfDeployment)
		result.Success = err == nil
		result.Error = err
	case nephoranv1.DeploymentStrategyGitOps:
		err := r.deployWithGitOps(ctx, cnfDeployment)
		result.Success = err == nil
		result.Error = err
	default:
		result.Error = fmt.Errorf("unsupported deployment strategy: %s", cnfDeployment.Spec.DeploymentStrategy)
		result.Success = false
	}

	result.Duration = time.Since(startTime)
	return result, result.Error
}

// checkDeploymentReadiness checks if the CNF deployment is ready
func (r *CNFDeploymentReconciler) checkDeploymentReadiness(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) (bool, error) {
	// Check deployment readiness based on deployment strategy
	switch cnfDeployment.Spec.DeploymentStrategy {
	case nephoranv1.DeploymentStrategyHelm:
		return r.checkHelmDeploymentReadiness(ctx, cnfDeployment)
	case nephoranv1.DeploymentStrategyOperator:
		return r.checkOperatorDeploymentReadiness(ctx, cnfDeployment)
	case nephoranv1.DeploymentStrategyDirect:
		return r.checkDirectDeploymentReadiness(ctx, cnfDeployment)
	case nephoranv1.DeploymentStrategyGitOps:
		return r.checkGitOpsDeploymentReadiness(ctx, cnfDeployment)
	default:
		return false, fmt.Errorf("unsupported deployment strategy: %s", cnfDeployment.Spec.DeploymentStrategy)
	}
}

// checkHelmDeploymentReadiness checks readiness for Helm deployments
func (r *CNFDeploymentReconciler) checkHelmDeploymentReadiness(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) (bool, error) {
	// Check if Helm release exists and is deployed
	// This is a placeholder - actual implementation would use Helm client
	return false, nil
}

// checkOperatorDeploymentReadiness checks readiness for Operator deployments
func (r *CNFDeploymentReconciler) checkOperatorDeploymentReadiness(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) (bool, error) {
	// Check if the custom resource managed by the operator is ready
	// This is a placeholder - actual implementation would check the operator's CRD status
	return false, nil
}

// checkDirectDeploymentReadiness checks readiness for direct Kubernetes deployments
func (r *CNFDeploymentReconciler) checkDirectDeploymentReadiness(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) (bool, error) {
	// Check if Kubernetes resources (Deployment, StatefulSet, etc.) are ready
	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      cnfDeployment.Name,
		Namespace: cnfDeployment.Namespace,
	}, deployment)

	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	// Check if deployment is ready
	return deployment.Status.ReadyReplicas == cnfDeployment.Spec.Replicas, nil
}

// checkGitOpsDeploymentReadiness checks readiness for GitOps deployments
func (r *CNFDeploymentReconciler) checkGitOpsDeploymentReadiness(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) (bool, error) {
	// Check if GitOps system has deployed the resources
	// This is a placeholder - actual implementation would check ArgoCD/Flux status
	return false, nil
}

// deployWithHelm deploys the CNF using Helm
func (r *CNFDeploymentReconciler) deployWithHelm(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) error {
	logger := log.FromContext(ctx).WithValues("cnfdeployment", cnfDeployment.Name, "strategy", "helm")

	if cnfDeployment.Spec.Helm == nil {
		return fmt.Errorf("helm configuration is required for Helm deployment strategy")
	}

	logger.Info("Deploying CNF with Helm", "chart", cnfDeployment.Spec.Helm.ChartName, "version", cnfDeployment.Spec.Helm.ChartVersion)

	// Use CNF Orchestrator to deploy with Helm
	if r.CNFOrchestrator != nil {
		return r.CNFOrchestrator.DeployWithHelm(ctx, cnfDeployment)
	}

	// Fallback implementation - this is a placeholder
	// In a real implementation, this would use the Helm Go client
	logger.Info("CNF Orchestrator not available, using fallback Helm deployment")
	return fmt.Errorf("helm deployment not yet implemented")
}

// deployWithOperator deploys the CNF using an operator
func (r *CNFDeploymentReconciler) deployWithOperator(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) error {
	logger := log.FromContext(ctx).WithValues("cnfdeployment", cnfDeployment.Name, "strategy", "operator")

	if cnfDeployment.Spec.Operator == nil {
		return fmt.Errorf("operator configuration is required for Operator deployment strategy")
	}

	logger.Info("Deploying CNF with Operator", "operator", cnfDeployment.Spec.Operator.Name)

	// Use CNF Orchestrator to deploy with operator
	if r.CNFOrchestrator != nil {
		return r.CNFOrchestrator.DeployWithOperator(ctx, cnfDeployment)
	}

	// Fallback implementation - this is a placeholder
	logger.Info("CNF Orchestrator not available, using fallback operator deployment")
	return fmt.Errorf("operator deployment not yet implemented")
}

// deployDirect deploys the CNF using direct Kubernetes resources
func (r *CNFDeploymentReconciler) deployDirect(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) error {
	logger := log.FromContext(ctx).WithValues("cnfdeployment", cnfDeployment.Name, "strategy", "direct")

	// Create Kubernetes Deployment
	deployment := r.createKubernetesDeployment(cnfDeployment)
	if err := r.Create(ctx, deployment); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create deployment: %w", err)
		}
	}

	// Create Service if needed
	service := r.createKubernetesService(cnfDeployment)
	if err := r.Create(ctx, service); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create service: %w", err)
		}
	}

	// Create HPA if auto-scaling is enabled
	if cnfDeployment.Spec.AutoScaling != nil && cnfDeployment.Spec.AutoScaling.Enabled {
		hpa := r.createHorizontalPodAutoscaler(cnfDeployment)
		if err := r.Create(ctx, hpa); err != nil {
			if !errors.IsAlreadyExists(err) {
				logger.Error(err, "Failed to create HPA, continuing without auto-scaling")
			}
		}
	}

	logger.Info("Direct deployment completed successfully")
	return nil
}

// deployWithGitOps deploys the CNF using GitOps
func (r *CNFDeploymentReconciler) deployWithGitOps(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) error {
	logger := log.FromContext(ctx).WithValues("cnfdeployment", cnfDeployment.Name, "strategy", "gitops")

	// Use CNF Orchestrator to deploy with GitOps
	if r.CNFOrchestrator != nil {
		return r.CNFOrchestrator.DeployWithGitOps(ctx, cnfDeployment)
	}

	logger.Info("CNF Orchestrator not available, using fallback GitOps deployment")
	return fmt.Errorf("gitops deployment not yet implemented")
}

// updateCNFDeploymentStatus updates the status of a CNFDeployment
func (r *CNFDeploymentReconciler) updateCNFDeploymentStatus(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment, phase, message string) {
	cnfDeployment.Status.Phase = phase
	cnfDeployment.Status.LastUpdatedTime = &metav1.Time{Time: time.Now()}

	// Update condition
	condition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "DeploymentReady",
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	if phase == "Failed" {
		condition.Status = metav1.ConditionFalse
		condition.Reason = "DeploymentFailed"
	}

	// Update or add condition
	found := false
	for i, existingCondition := range cnfDeployment.Status.Conditions {
		if existingCondition.Type == condition.Type {
			cnfDeployment.Status.Conditions[i] = condition
			found = true
			break
		}
	}
	if !found {
		cnfDeployment.Status.Conditions = append(cnfDeployment.Status.Conditions, condition)
	}

	// Update health check status
	r.updateHealthStatus(ctx, cnfDeployment)

	// Update the status
	if err := r.Status().Update(ctx, cnfDeployment); err != nil {
		log.FromContext(ctx).Error(err, "Failed to update CNFDeployment status")
	}
}

// updateHealthStatus updates the health status of the CNF deployment
func (r *CNFDeploymentReconciler) updateHealthStatus(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) {
	// Implement health checks based on CNF function type
	// This is a placeholder for actual health check implementation

	// Update health status
	if cnfDeployment.Status.Health == nil {
		cnfDeployment.Status.Health = &nephoranv1.CNFHealthStatus{}
	}

	cnfDeployment.Status.Health.Status = "Healthy"
	cnfDeployment.Status.Health.LastCheckTime = &metav1.Time{Time: time.Now()}

	// Add health check details
	if cnfDeployment.Status.Health.Details == nil {
		cnfDeployment.Status.Health.Details = make(map[string]string)
	}
	cnfDeployment.Status.Health.Details["lastCheck"] = time.Now().Format(time.RFC3339)
	cnfDeployment.Status.Health.Details["checkType"] = "basic"
}

// createKubernetesDeployment creates a Kubernetes Deployment for the CNF
func (r *CNFDeploymentReconciler) createKubernetesDeployment(cnfDeployment *nephoranv1.CNFDeployment) *appsv1.Deployment {
	labels := map[string]string{
		"app":                         cnfDeployment.Name,
		"cnf-type":                    string(cnfDeployment.Spec.CNFType),
		"cnf-function":                string(cnfDeployment.Spec.Function),
		"nephoran.nephio.org/managed": "true",
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cnfDeployment.Name,
			Namespace: cnfDeployment.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &cnfDeployment.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  string(cnfDeployment.Spec.Function),
							Image: r.getImageForCNFFunction(cnfDeployment.Spec.Function),
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    cnfDeployment.Spec.Resources.CPU,
									corev1.ResourceMemory: cnfDeployment.Spec.Resources.Memory,
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    cnfDeployment.Spec.Resources.CPU,
									corev1.ResourceMemory: cnfDeployment.Spec.Resources.Memory,
								},
							},
						},
					},
				},
			},
		},
	}

	// Set resource limits if specified
	if cnfDeployment.Spec.Resources.MaxCPU != nil {
		deployment.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU] = *cnfDeployment.Spec.Resources.MaxCPU
	}
	if cnfDeployment.Spec.Resources.MaxMemory != nil {
		deployment.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceMemory] = *cnfDeployment.Spec.Resources.MaxMemory
	}

	// Set owner reference
	ctrl.SetControllerReference(cnfDeployment, deployment, r.Scheme)

	return deployment
}

// createKubernetesService creates a Kubernetes Service for the CNF
func (r *CNFDeploymentReconciler) createKubernetesService(cnfDeployment *nephoranv1.CNFDeployment) *corev1.Service {
	labels := map[string]string{
		"app":                         cnfDeployment.Name,
		"cnf-type":                    string(cnfDeployment.Spec.CNFType),
		"cnf-function":                string(cnfDeployment.Spec.Function),
		"nephoran.nephio.org/managed": "true",
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cnfDeployment.Name,
			Namespace: cnfDeployment.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       8080,
					TargetPort: intstr.FromInt(8080),
				},
			},
		},
	}

	// Set owner reference
	ctrl.SetControllerReference(cnfDeployment, service, r.Scheme)

	return service
}

// createHorizontalPodAutoscaler creates an HPA for auto-scaling
func (r *CNFDeploymentReconciler) createHorizontalPodAutoscaler(cnfDeployment *nephoranv1.CNFDeployment) *autoscalingv2.HorizontalPodAutoscaler {
	labels := map[string]string{
		"app":                         cnfDeployment.Name,
		"cnf-type":                    string(cnfDeployment.Spec.CNFType),
		"cnf-function":                string(cnfDeployment.Spec.Function),
		"nephoran.nephio.org/managed": "true",
	}

	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cnfDeployment.Name + "-hpa",
			Namespace: cnfDeployment.Namespace,
			Labels:    labels,
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       cnfDeployment.Name,
			},
			MinReplicas: &cnfDeployment.Spec.AutoScaling.MinReplicas,
			MaxReplicas: cnfDeployment.Spec.AutoScaling.MaxReplicas,
		},
	}

	// Add metrics
	var metrics []autoscalingv2.MetricSpec

	if cnfDeployment.Spec.AutoScaling.CPUUtilization != nil {
		metrics = append(metrics, autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceCPU,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: cnfDeployment.Spec.AutoScaling.CPUUtilization,
				},
			},
		})
	}

	if cnfDeployment.Spec.AutoScaling.MemoryUtilization != nil {
		metrics = append(metrics, autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceMemory,
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: cnfDeployment.Spec.AutoScaling.MemoryUtilization,
				},
			},
		})
	}

	hpa.Spec.Metrics = metrics

	// Set owner reference
	ctrl.SetControllerReference(cnfDeployment, hpa, r.Scheme)

	return hpa
}

// getImageForCNFFunction returns the container image for a given CNF function
func (r *CNFDeploymentReconciler) getImageForCNFFunction(function nephoranv1.CNFFunction) string {
	// This is a placeholder - in a real implementation, this would map
	// CNF functions to their respective container images
	imageMap := map[nephoranv1.CNFFunction]string{
		nephoranv1.CNFFunctionAMF: "nephoran/amf:latest",
		nephoranv1.CNFFunctionSMF: "nephoran/smf:latest",
		nephoranv1.CNFFunctionUPF: "nephoran/upf:latest",
		// Add more mappings as needed
	}

	if image, exists := imageMap[function]; exists {
		return image
	}

	// Default image
	return "nephoran/cnf-default:latest"
}

// cleanupHelmDeployment cleans up a Helm deployment
func (r *CNFDeploymentReconciler) cleanupHelmDeployment(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) error {
	// Use CNF Orchestrator to cleanup Helm deployment
	if r.CNFOrchestrator != nil {
		return r.CNFOrchestrator.CleanupHelmDeployment(ctx, cnfDeployment)
	}

	// Fallback implementation
	log.FromContext(ctx).Info("CNF Orchestrator not available, using fallback Helm cleanup")
	return nil
}

// cleanupOperatorDeployment cleans up an operator deployment
func (r *CNFDeploymentReconciler) cleanupOperatorDeployment(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) error {
	// Use CNF Orchestrator to cleanup operator deployment
	if r.CNFOrchestrator != nil {
		return r.CNFOrchestrator.CleanupOperatorDeployment(ctx, cnfDeployment)
	}

	// Fallback implementation
	log.FromContext(ctx).Info("CNF Orchestrator not available, using fallback operator cleanup")
	return nil
}

// cleanupDirectDeployment cleans up a direct Kubernetes deployment
func (r *CNFDeploymentReconciler) cleanupDirectDeployment(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) error {
	logger := log.FromContext(ctx).WithValues("cnfdeployment", cnfDeployment.Name)

	// Delete HPA if it exists
	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      cnfDeployment.Name + "-hpa",
		Namespace: cnfDeployment.Namespace,
	}, hpa)
	if err == nil {
		if err := r.Delete(ctx, hpa); err != nil {
			logger.Error(err, "Failed to delete HPA")
		}
	}

	// Delete Service
	service := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      cnfDeployment.Name,
		Namespace: cnfDeployment.Namespace,
	}, service)
	if err == nil {
		if err := r.Delete(ctx, service); err != nil {
			logger.Error(err, "Failed to delete Service")
		}
	}

	// Delete Deployment
	deployment := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      cnfDeployment.Name,
		Namespace: cnfDeployment.Namespace,
	}, deployment)
	if err == nil {
		if err := r.Delete(ctx, deployment); err != nil {
			logger.Error(err, "Failed to delete Deployment")
		}
	}

	return nil
}

// cleanupGitOpsDeployment cleans up a GitOps deployment
func (r *CNFDeploymentReconciler) cleanupGitOpsDeployment(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) error {
	// Use CNF Orchestrator to cleanup GitOps deployment
	if r.CNFOrchestrator != nil {
		return r.CNFOrchestrator.CleanupGitOpsDeployment(ctx, cnfDeployment)
	}

	// Fallback implementation
	log.FromContext(ctx).Info("CNF Orchestrator not available, using fallback GitOps cleanup")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CNFDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nephoranv1.CNFDeployment{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&autoscalingv2.HorizontalPodAutoscaler{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
