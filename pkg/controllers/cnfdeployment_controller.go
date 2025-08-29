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



	nephoranv1 "github.com/nephio-project/nephoran-intent-operator/api/v1"

	"github.com/nephio-project/nephoran-intent-operator/pkg/cnf"

	"github.com/nephio-project/nephoran-intent-operator/pkg/llm"

	"github.com/nephio-project/nephoran-intent-operator/pkg/monitoring"



	appsv1 "k8s.io/api/apps/v1"

	autoscalingv2 "k8s.io/api/autoscaling/v2"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/apimachinery/pkg/types"

	"k8s.io/client-go/tools/record"



	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/controller-runtime/pkg/controller"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"sigs.k8s.io/controller-runtime/pkg/predicate"

)



const (

	// CNFDeploymentControllerName holds cnfdeploymentcontrollername value.

	CNFDeploymentControllerName = "cnfdeployment-controller"

	// CNFReconcileInterval holds cnfreconcileinterval value.

	CNFReconcileInterval = 30 * time.Second

	// CNFStatusUpdateInterval holds cnfstatusupdateinterval value.

	CNFStatusUpdateInterval = 10 * time.Second

)



// CNFDeploymentReconciler reconciles a CNFDeployment object.

type CNFDeploymentReconciler struct {

	client.Client

	Scheme           *runtime.Scheme

	Recorder         record.EventRecorder

	CNFOrchestrator  *cnf.CNFOrchestrator

	LLMProcessor     *llm.Processor

	MetricsCollector *monitoring.MetricsCollector

	Config           *CNFControllerConfig

}



// CNFControllerConfig holds configuration for the CNF controller.

type CNFControllerConfig struct {

	ReconcileTimeout        time.Duration

	MaxConcurrentReconciles int

	EnableStatusUpdates     bool

	StatusUpdateInterval    time.Duration

	EnableMetrics           bool

	EnableEvents            bool

}



// CNFDeploymentRequest represents a CNF deployment request with context.

type CNFDeploymentRequest struct {

	CNFDeployment *nephoranv1.CNFDeployment

	Context       context.Context

	RequestID     string

	Source        string // "intent" or "direct"

	NetworkIntent *nephoranv1.NetworkIntent

}



//+kubebuilder:rbac:groups=nephoran.com,resources=cnfdeployments,verbs=get;list;watch;create;update;patch;delete

//+kubebuilder:rbac:groups=nephoran.com,resources=cnfdeployments/status,verbs=get;update;patch

//+kubebuilder:rbac:groups=nephoran.com,resources=cnfdeployments/finalizers,verbs=update

//+kubebuilder:rbac:groups=apps,resources=deployments;statefulsets;daemonsets,verbs=get;list;watch;create;update;patch;delete

//+kubebuilder:rbac:groups="",resources=services;configmaps;secrets;persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

//+kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete

//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch



// Reconcile is part of the main kubernetes reconciliation loop which aims to.

// move the current state of the cluster closer to the desired state.

func (r *CNFDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	logger := log.FromContext(ctx)

	logger.Info("Starting CNF deployment reconciliation", "cnfdeployment", req.NamespacedName)



	// Fetch the CNFDeployment instance.

	cnfDeployment := &nephoranv1.CNFDeployment{}

	if err := r.Get(ctx, req.NamespacedName, cnfDeployment); err != nil {

		if errors.IsNotFound(err) {

			logger.Info("CNF deployment not found. Ignoring since object must be deleted")

			return ctrl.Result{}, nil

		}

		logger.Error(err, "Failed to get CNF deployment")

		return ctrl.Result{}, err

	}



	// Handle deletion.

	if cnfDeployment.DeletionTimestamp != nil {

		return r.handleCNFDeploymentDeletion(ctx, cnfDeployment)

	}



	// Add finalizer if not present.

	if !controllerutil.ContainsFinalizer(cnfDeployment, cnf.CNFOrchestratorFinalizer) {

		controllerutil.AddFinalizer(cnfDeployment, cnf.CNFOrchestratorFinalizer)

		if err := r.Update(ctx, cnfDeployment); err != nil {

			logger.Error(err, "Failed to add finalizer")

			return ctrl.Result{}, err

		}

		return ctrl.Result{Requeue: true}, nil

	}



	// Validate CNF deployment.

	if err := cnfDeployment.ValidateCNFDeployment(); err != nil {

		logger.Error(err, "CNF deployment validation failed")

		r.updateCNFDeploymentStatus(ctx, cnfDeployment, "Failed", err.Error())

		return ctrl.Result{}, err

	}



	// Process CNF deployment based on current phase.

	switch cnfDeployment.Status.Phase {

	case "", "Pending":

		return r.handlePendingCNF(ctx, cnfDeployment)

	case "Deploying":

		return r.handleDeployingCNF(ctx, cnfDeployment)

	case "Running":

		return r.handleRunningCNF(ctx, cnfDeployment)

	case "Scaling":

		return r.handleScalingCNF(ctx, cnfDeployment)

	case "Upgrading":

		return r.handleUpgradingCNF(ctx, cnfDeployment)

	case "Failed":

		return r.handleFailedCNF(ctx, cnfDeployment)

	default:

		logger.Info("Unknown CNF deployment phase", "phase", cnfDeployment.Status.Phase)

		return ctrl.Result{RequeueAfter: CNFReconcileInterval}, nil

	}

}



// handlePendingCNF handles CNF deployments in pending state.

func (r *CNFDeploymentReconciler) handlePendingCNF(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) (ctrl.Result, error) {

	logger := log.FromContext(ctx)

	logger.Info("Handling pending CNF deployment", "cnf", cnfDeployment.Name)



	// Update status to deploying.

	r.updateCNFDeploymentStatus(ctx, cnfDeployment, "Deploying", "Starting CNF deployment")



	// Create deployment request.

	deployReq := &cnf.DeployRequest{

		CNFDeployment:   cnfDeployment,

		Context:         ctx,

		ProcessingPhase: "Initial",

	}



	// Check if this CNF is created from a NetworkIntent.

	if networkIntent, err := r.findAssociatedNetworkIntent(ctx, cnfDeployment); err == nil && networkIntent != nil {

		deployReq.NetworkIntent = networkIntent

		logger.Info("Found associated network intent", "intent", networkIntent.Name)

	}



	// Deploy the CNF.

	result, err := r.CNFOrchestrator.Deploy(ctx, deployReq)

	if err != nil {

		logger.Error(err, "CNF deployment failed")

		r.updateCNFDeploymentStatus(ctx, cnfDeployment, "Failed", fmt.Sprintf("Deployment failed: %v", err))

		r.Recorder.Event(cnfDeployment, "Warning", "DeploymentFailed", fmt.Sprintf("CNF deployment failed: %v", err))

		return ctrl.Result{RequeueAfter: 2 * time.Minute}, nil

	}



	// Update status with deployment result.

	cnfDeployment.Status.HelmRelease = result.ReleaseName

	cnfDeployment.Status.ServiceEndpoints = result.ServiceEndpoints

	cnfDeployment.Status.ResourceUtilization = result.ResourceStatus

	cnfDeployment.Status.DeploymentStartTime = &metav1.Time{Time: time.Now()}



	r.updateCNFDeploymentStatus(ctx, cnfDeployment, "Running", "CNF deployment completed successfully")

	r.Recorder.Event(cnfDeployment, "Normal", "DeploymentCompleted", "CNF deployment completed successfully")



	// Record metrics.

	if r.MetricsCollector != nil {

		r.MetricsCollector.RecordCNFDeployment(cnfDeployment.Spec.Function, result.Duration)

	}



	return ctrl.Result{RequeueAfter: CNFReconcileInterval}, nil

}



// handleDeployingCNF handles CNF deployments in deploying state.

func (r *CNFDeploymentReconciler) handleDeployingCNF(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) (ctrl.Result, error) {

	logger := log.FromContext(ctx)

	logger.Info("Checking CNF deployment progress", "cnf", cnfDeployment.Name)



	// Check deployment status by examining underlying resources.

	ready, err := r.checkDeploymentReadiness(ctx, cnfDeployment)

	if err != nil {

		logger.Error(err, "Failed to check deployment readiness")

		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil

	}



	if ready {

		r.updateCNFDeploymentStatus(ctx, cnfDeployment, "Running", "CNF is ready and running")

		r.Recorder.Event(cnfDeployment, "Normal", "DeploymentReady", "CNF deployment is ready")

	}



	return ctrl.Result{RequeueAfter: CNFReconcileInterval}, nil

}



// handleRunningCNF handles CNF deployments in running state.

func (r *CNFDeploymentReconciler) handleRunningCNF(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) (ctrl.Result, error) {

	logger := log.FromContext(ctx)



	// Update resource utilization metrics.

	if err := r.updateResourceMetrics(ctx, cnfDeployment); err != nil {

		logger.Error(err, "Failed to update resource metrics")

	}



	// Check if scaling is needed.

	if r.shouldScale(cnfDeployment) {

		return r.initiateScaling(ctx, cnfDeployment)

	}



	// Perform health checks.

	if err := r.performHealthChecks(ctx, cnfDeployment); err != nil {

		logger.Error(err, "Health checks failed")

		r.updateCNFDeploymentStatus(ctx, cnfDeployment, "Degraded", fmt.Sprintf("Health checks failed: %v", err))

		r.Recorder.Event(cnfDeployment, "Warning", "HealthCheckFailed", fmt.Sprintf("Health checks failed: %v", err))

	}



	return ctrl.Result{RequeueAfter: CNFReconcileInterval}, nil

}



// handleScalingCNF handles CNF deployments in scaling state.

func (r *CNFDeploymentReconciler) handleScalingCNF(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) (ctrl.Result, error) {

	logger := log.FromContext(ctx)

	logger.Info("Checking CNF scaling progress", "cnf", cnfDeployment.Name)



	// Check if scaling is complete.

	if r.isScalingComplete(ctx, cnfDeployment) {

		r.updateCNFDeploymentStatus(ctx, cnfDeployment, "Running", "CNF scaling completed")

		r.Recorder.Event(cnfDeployment, "Normal", "ScalingCompleted", "CNF scaling completed successfully")

	}



	return ctrl.Result{RequeueAfter: CNFReconcileInterval}, nil

}



// handleUpgradingCNF handles CNF deployments in upgrading state.

func (r *CNFDeploymentReconciler) handleUpgradingCNF(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) (ctrl.Result, error) {

	logger := log.FromContext(ctx)

	logger.Info("Checking CNF upgrade progress", "cnf", cnfDeployment.Name)



	// Upgrade logic would be implemented here.

	// For now, just transition back to running.

	r.updateCNFDeploymentStatus(ctx, cnfDeployment, "Running", "CNF upgrade completed")



	return ctrl.Result{RequeueAfter: CNFReconcileInterval}, nil

}



// handleFailedCNF handles CNF deployments in failed state.

func (r *CNFDeploymentReconciler) handleFailedCNF(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) (ctrl.Result, error) {

	logger := log.FromContext(ctx)

	logger.Info("Handling failed CNF deployment", "cnf", cnfDeployment.Name)



	// Implement retry logic or manual intervention workflow.

	// For now, just log and requeue for manual intervention.



	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil

}



// handleCNFDeploymentDeletion handles the deletion of a CNF deployment.

func (r *CNFDeploymentReconciler) handleCNFDeploymentDeletion(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) (ctrl.Result, error) {

	logger := log.FromContext(ctx)

	logger.Info("Handling CNF deployment deletion", "cnf", cnfDeployment.Name)



	if controllerutil.ContainsFinalizer(cnfDeployment, cnf.CNFOrchestratorFinalizer) {

		// Perform cleanup operations.

		if err := r.cleanupCNFResources(ctx, cnfDeployment); err != nil {

			logger.Error(err, "Failed to cleanup CNF resources")

			return ctrl.Result{RequeueAfter: time.Minute}, err

		}



		// Remove finalizer.

		controllerutil.RemoveFinalizer(cnfDeployment, cnf.CNFOrchestratorFinalizer)

		if err := r.Update(ctx, cnfDeployment); err != nil {

			logger.Error(err, "Failed to remove finalizer")

			return ctrl.Result{}, err

		}

	}



	return ctrl.Result{}, nil

}



// findAssociatedNetworkIntent finds the NetworkIntent that created this CNF deployment.

func (r *CNFDeploymentReconciler) findAssociatedNetworkIntent(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) (*nephoranv1.NetworkIntent, error) {

	// Check owner references.

	for _, owner := range cnfDeployment.GetOwnerReferences() {

		if owner.Kind == "NetworkIntent" && owner.APIVersion == "nephoran.com/v1" {

			intent := &nephoranv1.NetworkIntent{}

			err := r.Get(ctx, types.NamespacedName{

				Name:      owner.Name,

				Namespace: cnfDeployment.Namespace,

			}, intent)

			if err != nil {

				return nil, err

			}

			return intent, nil

		}

	}



	// Check labels.

	if intentName, exists := cnfDeployment.Labels["nephoran.com/network-intent"]; exists {

		intent := &nephoranv1.NetworkIntent{}

		err := r.Get(ctx, types.NamespacedName{

			Name:      intentName,

			Namespace: cnfDeployment.Namespace,

		}, intent)

		if err != nil {

			return nil, err

		}

		return intent, nil

	}



	return nil, fmt.Errorf("no associated network intent found")

}



// updateCNFDeploymentStatus updates the status of a CNF deployment.

func (r *CNFDeploymentReconciler) updateCNFDeploymentStatus(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment, phase, message string) error {

	cnfDeployment.Status.Phase = phase

	cnfDeployment.Status.LastUpdatedTime = &metav1.Time{Time: time.Now()}

	cnfDeployment.Status.ObservedGeneration = cnfDeployment.Generation



	// Update conditions.

	condition := metav1.Condition{

		Type:               "Ready",

		Status:             metav1.ConditionTrue,

		LastTransitionTime: metav1.Now(),

		Reason:             phase,

		Message:            message,

	}



	if phase == "Failed" {

		condition.Status = metav1.ConditionFalse

	}



	// Update or add condition.

	existingCondition := findCondition(cnfDeployment.Status.Conditions, "Ready")

	if existingCondition != nil {

		existingCondition.Status = condition.Status

		existingCondition.LastTransitionTime = condition.LastTransitionTime

		existingCondition.Reason = condition.Reason

		existingCondition.Message = condition.Message

	} else {

		cnfDeployment.Status.Conditions = append(cnfDeployment.Status.Conditions, condition)

	}



	return r.Status().Update(ctx, cnfDeployment)

}



// checkDeploymentReadiness checks if the CNF deployment is ready.

func (r *CNFDeploymentReconciler) checkDeploymentReadiness(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) (bool, error) {

	// Check deployment readiness based on deployment strategy.

	switch cnfDeployment.Spec.DeploymentStrategy {

	case nephoranv1.CNFDeploymentStrategyHelm:

		return r.checkHelmDeploymentReadiness(ctx, cnfDeployment)

	case nephoranv1.CNFDeploymentStrategyOperator:

		return r.checkOperatorDeploymentReadiness(ctx, cnfDeployment)

	case nephoranv1.CNFDeploymentStrategyDirect:

		return r.checkDirectDeploymentReadiness(ctx, cnfDeployment)

	case nephoranv1.CNFDeploymentStrategyGitOps:

		return r.checkGitOpsDeploymentReadiness(ctx, cnfDeployment)

	default:

		return false, fmt.Errorf("unsupported deployment strategy: %s", cnfDeployment.Spec.DeploymentStrategy)

	}

}



// checkHelmDeploymentReadiness checks readiness for Helm deployments.

func (r *CNFDeploymentReconciler) checkHelmDeploymentReadiness(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) (bool, error) {

	// Look for deployments with labels matching the Helm release.

	deploymentList := &appsv1.DeploymentList{}

	listOpts := []client.ListOption{

		client.InNamespace(cnfDeployment.Namespace),

		client.MatchingLabels{

			"app.kubernetes.io/instance": cnfDeployment.Status.HelmRelease,

		},

	}



	if err := r.List(ctx, deploymentList, listOpts...); err != nil {

		return false, err

	}



	if len(deploymentList.Items) == 0 {

		return false, nil

	}



	// Check if all deployments are ready.

	totalReplicas := int32(0)

	readyReplicas := int32(0)



	for _, deployment := range deploymentList.Items {

		totalReplicas += *deployment.Spec.Replicas

		readyReplicas += deployment.Status.ReadyReplicas

	}



	cnfDeployment.Status.ReadyReplicas = readyReplicas

	cnfDeployment.Status.AvailableReplicas = readyReplicas



	return readyReplicas == totalReplicas && totalReplicas > 0, nil

}



// checkOperatorDeploymentReadiness checks readiness for operator-based deployments.

func (r *CNFDeploymentReconciler) checkOperatorDeploymentReadiness(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) (bool, error) {

	// This would check the status of the custom resource managed by the operator.

	// For now, return true as a placeholder.

	return true, nil

}



// checkDirectDeploymentReadiness checks readiness for direct deployments.

func (r *CNFDeploymentReconciler) checkDirectDeploymentReadiness(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) (bool, error) {

	// Check deployments created directly by the controller.

	deploymentList := &appsv1.DeploymentList{}

	listOpts := []client.ListOption{

		client.InNamespace(cnfDeployment.Namespace),

		client.MatchingLabels{

			"nephoran.com/cnf-deployment": cnfDeployment.Name,

		},

	}



	if err := r.List(ctx, deploymentList, listOpts...); err != nil {

		return false, err

	}



	return len(deploymentList.Items) > 0 && deploymentList.Items[0].Status.ReadyReplicas > 0, nil

}



// checkGitOpsDeploymentReadiness checks readiness for GitOps deployments.

func (r *CNFDeploymentReconciler) checkGitOpsDeploymentReadiness(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) (bool, error) {

	// This would check the status via GitOps tools like ArgoCD or Flux.

	// For now, return true as a placeholder.

	return true, nil

}



// updateResourceMetrics updates resource utilization metrics.

func (r *CNFDeploymentReconciler) updateResourceMetrics(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) error {

	// Collect resource metrics from underlying pods.

	// This is a placeholder for actual metrics collection.

	if cnfDeployment.Status.ResourceUtilization == nil {

		cnfDeployment.Status.ResourceUtilization = make(map[string]string)

	}



	cnfDeployment.Status.ResourceUtilization["cpu"] = "50%"

	cnfDeployment.Status.ResourceUtilization["memory"] = "60%"

	cnfDeployment.Status.ResourceUtilization["lastUpdated"] = time.Now().Format(time.RFC3339)



	return r.Status().Update(ctx, cnfDeployment)

}



// shouldScale determines if the CNF should be scaled.

func (r *CNFDeploymentReconciler) shouldScale(cnfDeployment *nephoranv1.CNFDeployment) bool {

	if cnfDeployment.Spec.AutoScaling == nil || !cnfDeployment.Spec.AutoScaling.Enabled {

		return false

	}



	// Check if current replicas are within bounds.

	currentReplicas := cnfDeployment.Status.ReadyReplicas

	minReplicas := cnfDeployment.Spec.AutoScaling.MinReplicas

	maxReplicas := cnfDeployment.Spec.AutoScaling.MaxReplicas



	return currentReplicas < minReplicas || currentReplicas > maxReplicas

}



// initiateScaling initiates scaling for the CNF.

func (r *CNFDeploymentReconciler) initiateScaling(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) (ctrl.Result, error) {

	logger := log.FromContext(ctx)

	logger.Info("Initiating CNF scaling", "cnf", cnfDeployment.Name)



	r.updateCNFDeploymentStatus(ctx, cnfDeployment, "Scaling", "CNF scaling initiated")

	r.Recorder.Event(cnfDeployment, "Normal", "ScalingStarted", "CNF scaling started")



	// Create or update HPA.

	if err := r.createOrUpdateHPA(ctx, cnfDeployment); err != nil {

		logger.Error(err, "Failed to create or update HPA")

		return ctrl.Result{}, err

	}



	return ctrl.Result{RequeueAfter: CNFReconcileInterval}, nil

}



// isScalingComplete checks if scaling is complete.

func (r *CNFDeploymentReconciler) isScalingComplete(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) bool {

	// Check if HPA has stabilized.

	hpa := &autoscalingv2.HorizontalPodAutoscaler{}

	hpaName := fmt.Sprintf("%s-hpa", cnfDeployment.Name)

	err := r.Get(ctx, types.NamespacedName{

		Name:      hpaName,

		Namespace: cnfDeployment.Namespace,

	}, hpa)

	if err != nil {

		return false

	}



	// Check if current replicas match desired replicas.

	return hpa.Status.CurrentReplicas == hpa.Status.DesiredReplicas

}



// performHealthChecks performs health checks on the CNF.

func (r *CNFDeploymentReconciler) performHealthChecks(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) error {

	// Implement health checks based on CNF function type.

	// This is a placeholder for actual health check implementation.



	// Update health status.

	if cnfDeployment.Status.Health == nil {

		cnfDeployment.Status.Health = &nephoranv1.CNFHealthStatus{}

	}



	cnfDeployment.Status.Health.Status = "Healthy"

	cnfDeployment.Status.Health.LastCheckTime = &metav1.Time{Time: time.Now()}

	cnfDeployment.Status.Health.Details = map[string]string{

		"overall":   "healthy",

		"lastCheck": time.Now().Format(time.RFC3339),

	}



	return r.Status().Update(ctx, cnfDeployment)

}



// createOrUpdateHPA creates or updates the HorizontalPodAutoscaler for the CNF.

func (r *CNFDeploymentReconciler) createOrUpdateHPA(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) error {

	if cnfDeployment.Spec.AutoScaling == nil || !cnfDeployment.Spec.AutoScaling.Enabled {

		return nil

	}



	hpa := &autoscalingv2.HorizontalPodAutoscaler{

		ObjectMeta: metav1.ObjectMeta{

			Name:      fmt.Sprintf("%s-hpa", cnfDeployment.Name),

			Namespace: cnfDeployment.Namespace,

			Labels: map[string]string{

				"nephoran.com/cnf-deployment": cnfDeployment.Name,

				"nephoran.com/component":      "autoscaler",

			},

		},

		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{

			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{

				APIVersion: "apps/v1",

				Kind:       "Deployment",

				Name:       cnfDeployment.Name,

			},

			MinReplicas: &cnfDeployment.Spec.AutoScaling.MinReplicas,

			MaxReplicas: cnfDeployment.Spec.AutoScaling.MaxReplicas,

			Metrics:     []autoscalingv2.MetricSpec{},

		},

	}



	// Add CPU utilization metric if specified.

	if cnfDeployment.Spec.AutoScaling.CPUUtilization != nil {

		cpuMetric := autoscalingv2.MetricSpec{

			Type: autoscalingv2.ResourceMetricSourceType,

			Resource: &autoscalingv2.ResourceMetricSource{

				Name: corev1.ResourceCPU,

				Target: autoscalingv2.MetricTarget{

					Type:               autoscalingv2.UtilizationMetricType,

					AverageUtilization: cnfDeployment.Spec.AutoScaling.CPUUtilization,

				},

			},

		}

		hpa.Spec.Metrics = append(hpa.Spec.Metrics, cpuMetric)

	}



	// Add memory utilization metric if specified.

	if cnfDeployment.Spec.AutoScaling.MemoryUtilization != nil {

		memoryMetric := autoscalingv2.MetricSpec{

			Type: autoscalingv2.ResourceMetricSourceType,

			Resource: &autoscalingv2.ResourceMetricSource{

				Name: corev1.ResourceMemory,

				Target: autoscalingv2.MetricTarget{

					Type:               autoscalingv2.UtilizationMetricType,

					AverageUtilization: cnfDeployment.Spec.AutoScaling.MemoryUtilization,

				},

			},

		}

		hpa.Spec.Metrics = append(hpa.Spec.Metrics, memoryMetric)

	}



	// Set owner reference.

	if err := controllerutil.SetControllerReference(cnfDeployment, hpa, r.Scheme); err != nil {

		return err

	}



	// Create or update HPA.

	existingHPA := &autoscalingv2.HorizontalPodAutoscaler{}

	err := r.Get(ctx, types.NamespacedName{Name: hpa.Name, Namespace: hpa.Namespace}, existingHPA)

	if err != nil && errors.IsNotFound(err) {

		return r.Create(ctx, hpa)

	} else if err != nil {

		return err

	}



	// Update existing HPA.

	existingHPA.Spec = hpa.Spec

	return r.Update(ctx, existingHPA)

}



// cleanupCNFResources cleans up resources associated with a CNF deployment.

func (r *CNFDeploymentReconciler) cleanupCNFResources(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) error {

	logger := log.FromContext(ctx)

	logger.Info("Cleaning up CNF resources", "cnf", cnfDeployment.Name)



	// Delete HPA.

	hpa := &autoscalingv2.HorizontalPodAutoscaler{

		ObjectMeta: metav1.ObjectMeta{

			Name:      fmt.Sprintf("%s-hpa", cnfDeployment.Name),

			Namespace: cnfDeployment.Namespace,

		},

	}

	if err := r.Delete(ctx, hpa); err != nil && !errors.IsNotFound(err) {

		logger.Error(err, "Failed to delete HPA")

	}



	// Additional cleanup based on deployment strategy.

	switch cnfDeployment.Spec.DeploymentStrategy {

	case nephoranv1.CNFDeploymentStrategyHelm:

		// Helm cleanup would be handled by the orchestrator.

		break

	case nephoranv1.CNFDeploymentStrategyDirect:

		// Clean up directly created resources.

		if err := r.cleanupDirectResources(ctx, cnfDeployment); err != nil {

			return err

		}

	}



	r.Recorder.Event(cnfDeployment, "Normal", "ResourcesCleanedUp", "CNF resources cleaned up successfully")

	return nil

}



// cleanupDirectResources cleans up directly created resources.

func (r *CNFDeploymentReconciler) cleanupDirectResources(ctx context.Context, cnfDeployment *nephoranv1.CNFDeployment) error {

	// Delete deployments.

	deploymentList := &appsv1.DeploymentList{}

	listOpts := []client.ListOption{

		client.InNamespace(cnfDeployment.Namespace),

		client.MatchingLabels{

			"nephoran.com/cnf-deployment": cnfDeployment.Name,

		},

	}



	if err := r.List(ctx, deploymentList, listOpts...); err != nil {

		return err

	}



	for _, deployment := range deploymentList.Items {

		if err := r.Delete(ctx, &deployment); err != nil && !errors.IsNotFound(err) {

			return err

		}

	}



	// Delete services.

	serviceList := &corev1.ServiceList{}

	if err := r.List(ctx, serviceList, listOpts...); err != nil {

		return err

	}



	for _, service := range serviceList.Items {

		if err := r.Delete(ctx, &service); err != nil && !errors.IsNotFound(err) {

			return err

		}

	}



	return nil

}



// findCondition finds a condition by type in the conditions slice.

func findCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {

	for i, condition := range conditions {

		if condition.Type == conditionType {

			return &conditions[i]

		}

	}

	return nil

}



// SetupWithManager sets up the controller with the Manager.

func (r *CNFDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).

		For(&nephoranv1.CNFDeployment{}).

		Owns(&appsv1.Deployment{}).

		Owns(&corev1.Service{}).

		Owns(&autoscalingv2.HorizontalPodAutoscaler{}).

		WithOptions(controller.Options{

			MaxConcurrentReconciles: r.Config.MaxConcurrentReconciles,

		}).

		WithEventFilter(predicate.GenerationChangedPredicate{}).

		Complete(r)

}



// NewCNFDeploymentReconciler creates a new CNFDeploymentReconciler.

func NewCNFDeploymentReconciler(mgr ctrl.Manager, cnfOrchestrator *cnf.CNFOrchestrator) *CNFDeploymentReconciler {

	return &CNFDeploymentReconciler{

		Client:          mgr.GetClient(),

		Scheme:          mgr.GetScheme(),

		Recorder:        mgr.GetEventRecorderFor(CNFDeploymentControllerName),

		CNFOrchestrator: cnfOrchestrator,

		Config: &CNFControllerConfig{

			ReconcileTimeout:        10 * time.Minute,

			MaxConcurrentReconciles: 5,

			EnableStatusUpdates:     true,

			StatusUpdateInterval:    CNFStatusUpdateInterval,

			EnableMetrics:           true,

			EnableEvents:            true,

		},

	}

}

