package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/git"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio"
)

// Finalizer constants
const (
	NetworkIntentFinalizer = "networkintent.nephoran.com/finalizer"
)

type NetworkIntentReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	GitClient       git.ClientInterface
	LLMClient       llm.ClientInterface
	LLMProcessorURL string
	HTTPClient      *http.Client
	EventRecorder   record.EventRecorder
	PackageGen      *nephio.PackageGenerator
	UseNephioPorch  bool

	// Configuration for retry and GitOps
	MaxRetries    int
	RetryDelay    time.Duration
	GitRepoURL    string
	GitBranch     string
	GitDeployPath string
}

//+kubebuilder:rbac:groups=nephoran.com,resources=networkintents,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nephoran.com,resources=networkintents/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nephoran.com,resources=networkintents/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create

func (r *NetworkIntentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the NetworkIntent instance
	var networkIntent nephoranv1.NetworkIntent
	if err := r.Get(ctx, req.NamespacedName, &networkIntent); err != nil {
		logger.Error(err, "unable to fetch NetworkIntent")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle deletion - check if the object is being deleted
	if networkIntent.DeletionTimestamp != nil {
		return r.handleDeletion(ctx, &networkIntent)
	}

	// Add finalizer if it doesn't exist
	if !containsFinalizer(networkIntent.Finalizers, NetworkIntentFinalizer) {
		networkIntent.Finalizers = append(networkIntent.Finalizers, NetworkIntentFinalizer)
		if err := r.Update(ctx, &networkIntent); err != nil {
			logger.Error(err, "failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Set default retry configuration if not set
	if r.MaxRetries == 0 {
		r.MaxRetries = 3
	}
	if r.RetryDelay == 0 {
		r.RetryDelay = time.Second * 30
	}

	// Update observed generation
	networkIntent.Status.ObservedGeneration = networkIntent.Generation

	// Check if intent has already been processed successfully
	if isConditionTrue(networkIntent.Status.Conditions, "Processed") &&
		isConditionTrue(networkIntent.Status.Conditions, "Deployed") {
		logger.Info("NetworkIntent already processed and deployed", "intent", networkIntent.Spec.Intent)
		return ctrl.Result{}, nil
	}

	// Step 1: Process the intent using LLM client with retry logic
	if !isConditionTrue(networkIntent.Status.Conditions, "Processed") {
		// Set phase and start time if not already set
		if networkIntent.Status.Phase == "" {
			networkIntent.Status.Phase = "Processing"
			now := metav1.Now()
			networkIntent.Status.ProcessingStartTime = &now
			if err := r.Status().Update(ctx, &networkIntent); err != nil {
				logger.Error(err, "failed to update NetworkIntent status")
			}
		}

		result, err := r.processIntentWithRetry(ctx, &networkIntent)
		if err != nil {
			return result, err
		}
		if result.Requeue || result.RequeueAfter > 0 {
			return result, nil
		}
	}

	// Step 2: Deploy using GitOps if processing succeeded
	if isConditionTrue(networkIntent.Status.Conditions, "Processed") &&
		!isConditionTrue(networkIntent.Status.Conditions, "Deployed") {
		// Set phase and start time for deployment if not already set
		if networkIntent.Status.Phase != "Deploying" {
			networkIntent.Status.Phase = "Deploying"
			now := metav1.Now()
			networkIntent.Status.DeploymentStartTime = &now
			if err := r.Status().Update(ctx, &networkIntent); err != nil {
				logger.Error(err, "failed to update NetworkIntent status")
			}
		}

		result, err := r.deployViaGitOps(ctx, &networkIntent)
		if err != nil {
			return result, err
		}
		if result.Requeue || result.RequeueAfter > 0 {
			return result, nil
		}
	}

	// Final status update - mark as completed
	if networkIntent.Status.Phase != "Completed" {
		networkIntent.Status.Phase = "Completed"
		if err := r.Status().Update(ctx, &networkIntent); err != nil {
			logger.Error(err, "failed to update NetworkIntent final status")
		}
	}

	logger.Info("NetworkIntent fully processed and deployed", "intent", networkIntent.Spec.Intent)
	return ctrl.Result{}, nil
}

func (r *NetworkIntentReconciler) processIntentWithRetry(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if r.LLMClient == nil {
		logger.Error(fmt.Errorf("LLM client is nil"), "LLM client not configured")
		r.recordEvent(networkIntent, "Warning", "LLMClientNotConfigured", "LLM client is not configured")
		return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
	}

	// Get current retry count from annotations
	retryCount := getRetryCount(networkIntent, "llm-processing")

	if retryCount >= r.MaxRetries {
		// Max retries reached, mark as failed
		condition := metav1.Condition{
			Type:               "Processed",
			Status:             metav1.ConditionFalse,
			Reason:             "LLMProcessingFailedMaxRetries",
			Message:            fmt.Sprintf("Failed to process intent after %d retries", r.MaxRetries),
			LastTransitionTime: metav1.Now(),
		}
		updateCondition(&networkIntent.Status.Conditions, condition)

		if err := r.Status().Update(ctx, networkIntent); err != nil {
			logger.Error(err, "failed to update NetworkIntent status")
			return ctrl.Result{}, err
		}

		r.recordEvent(networkIntent, "Warning", "LLMProcessingFailed",
			fmt.Sprintf("Failed to process intent after %d retries", r.MaxRetries))
		return ctrl.Result{}, nil
	}

	// Attempt LLM processing
	processedResult, err := r.LLMClient.ProcessIntent(ctx, networkIntent.Spec.Intent)
	if err != nil {
		logger.Error(err, "failed to process intent with LLM", "retry", retryCount+1)

		// Increment retry count
		setRetryCount(networkIntent, "llm-processing", retryCount+1)

		// Update status with retry information
		now := metav1.Now()
		networkIntent.Status.LastRetryTime = &now
		condition := metav1.Condition{
			Type:               "Processed",
			Status:             metav1.ConditionFalse,
			Reason:             "LLMProcessingRetrying",
			Message:            fmt.Sprintf("LLM processing failed (attempt %d/%d): %v", retryCount+1, r.MaxRetries, err),
			LastTransitionTime: now,
		}
		updateCondition(&networkIntent.Status.Conditions, condition)

		if updateErr := r.Status().Update(ctx, networkIntent); updateErr != nil {
			logger.Error(updateErr, "failed to update NetworkIntent status")
		}

		r.recordEvent(networkIntent, "Warning", "LLMProcessingRetry",
			fmt.Sprintf("LLM processing failed, retry %d/%d: %v", retryCount+1, r.MaxRetries, err))

		return ctrl.Result{RequeueAfter: r.RetryDelay}, nil
	}

	// Parse the processed result and update the Parameters field
	var parameters map[string]interface{}
	if err := json.Unmarshal([]byte(processedResult), &parameters); err != nil {
		logger.Error(err, "failed to parse LLM response")

		// Increment retry count for parsing failure
		setRetryCount(networkIntent, "llm-processing", retryCount+1)

		condition := metav1.Condition{
			Type:               "Processed",
			Status:             metav1.ConditionFalse,
			Reason:             "LLMResponseParsingFailed",
			Message:            fmt.Sprintf("Failed to parse LLM response: %v", err),
			LastTransitionTime: metav1.Now(),
		}
		updateCondition(&networkIntent.Status.Conditions, condition)

		if updateErr := r.Status().Update(ctx, networkIntent); updateErr != nil {
			logger.Error(updateErr, "failed to update NetworkIntent status")
		}

		r.recordEvent(networkIntent, "Warning", "LLMResponseParsingFailed",
			fmt.Sprintf("Failed to parse LLM response: %v", err))

		return ctrl.Result{RequeueAfter: r.RetryDelay}, nil
	}

	// Update NetworkIntent with processed parameters
	parametersRaw, _ := json.Marshal(parameters)
	networkIntent.Spec.Parameters = runtime.RawExtension{Raw: parametersRaw}

	if err := r.Update(ctx, networkIntent); err != nil {
		logger.Error(err, "failed to update NetworkIntent with parameters")
		return ctrl.Result{}, err
	}

	// Clear retry count and update status to reflect successful processing
	clearRetryCount(networkIntent, "llm-processing")
	now := metav1.Now()
	networkIntent.Status.ProcessingCompletionTime = &now
	condition := metav1.Condition{
		Type:               "Processed",
		Status:             metav1.ConditionTrue,
		Reason:             "LLMProcessingSucceeded",
		Message:            "Intent successfully processed by LLM",
		LastTransitionTime: now,
	}
	updateCondition(&networkIntent.Status.Conditions, condition)

	if err := r.Status().Update(ctx, networkIntent); err != nil {
		logger.Error(err, "failed to update NetworkIntent status")
		return ctrl.Result{}, err
	}

	r.recordEvent(networkIntent, "Normal", "LLMProcessingSucceeded", "Intent successfully processed by LLM")
	logger.Info("NetworkIntent processed successfully", "intent", networkIntent.Spec.Intent)

	return ctrl.Result{}, nil
}

func (r *NetworkIntentReconciler) deployViaGitOps(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if r.GitClient == nil {
		logger.Error(fmt.Errorf("Git client is nil"), "Git client not configured")
		r.recordEvent(networkIntent, "Warning", "GitClientNotConfigured", "Git client is not configured")
		return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
	}

	// Get current retry count from annotations
	retryCount := getRetryCount(networkIntent, "git-deployment")

	if retryCount >= r.MaxRetries {
		// Max retries reached, mark as failed
		condition := metav1.Condition{
			Type:               "Deployed",
			Status:             metav1.ConditionFalse,
			Reason:             "GitDeploymentFailedMaxRetries",
			Message:            fmt.Sprintf("Failed to deploy via GitOps after %d retries", r.MaxRetries),
			LastTransitionTime: metav1.Now(),
		}
		updateCondition(&networkIntent.Status.Conditions, condition)

		if err := r.Status().Update(ctx, networkIntent); err != nil {
			logger.Error(err, "failed to update NetworkIntent status")
			return ctrl.Result{}, err
		}

		r.recordEvent(networkIntent, "Warning", "GitDeploymentFailed",
			fmt.Sprintf("Failed to deploy via GitOps after %d retries", r.MaxRetries))
		return ctrl.Result{}, nil
	}

	// Generate deployment files from processed parameters
	var deploymentFiles map[string]string
	var err error
	
	if r.UseNephioPorch && r.PackageGen != nil {
		// Use Nephio package generator for KRM package generation
		logger.Info("generating Nephio KRM package")
		deploymentFiles, err = r.PackageGen.GeneratePackage(networkIntent)
	} else {
		// Use legacy deployment file generation
		logger.Info("using legacy deployment file generation")
		deploymentFiles, err = r.generateDeploymentFiles(networkIntent)
	}
	
	if err != nil {
		logger.Error(err, "failed to generate deployment files")

		// Increment retry count
		setRetryCount(networkIntent, "git-deployment", retryCount+1)

		condition := metav1.Condition{
			Type:               "Deployed",
			Status:             metav1.ConditionFalse,
			Reason:             "DeploymentFileGenerationFailed",
			Message:            fmt.Sprintf("Failed to generate deployment files: %v", err),
			LastTransitionTime: metav1.Now(),
		}
		updateCondition(&networkIntent.Status.Conditions, condition)

		if updateErr := r.Status().Update(ctx, networkIntent); updateErr != nil {
			logger.Error(updateErr, "failed to update NetworkIntent status")
		}

		r.recordEvent(networkIntent, "Warning", "DeploymentFileGenerationFailed",
			fmt.Sprintf("Failed to generate deployment files: %v", err))

		return ctrl.Result{RequeueAfter: r.RetryDelay}, nil
	}

	// Initialize Git repository
	if err := r.GitClient.InitRepo(); err != nil {
		logger.Error(err, "failed to initialize git repository")

		// Increment retry count
		setRetryCount(networkIntent, "git-deployment", retryCount+1)

		condition := metav1.Condition{
			Type:               "Deployed",
			Status:             metav1.ConditionFalse,
			Reason:             "GitRepoInitializationFailed",
			Message:            fmt.Sprintf("Failed to initialize git repository: %v", err),
			LastTransitionTime: metav1.Now(),
		}
		updateCondition(&networkIntent.Status.Conditions, condition)

		if updateErr := r.Status().Update(ctx, networkIntent); updateErr != nil {
			logger.Error(updateErr, "failed to update NetworkIntent status")
		}

		r.recordEvent(networkIntent, "Warning", "GitRepoInitializationFailed",
			fmt.Sprintf("Failed to initialize git repository: %v", err))

		return ctrl.Result{RequeueAfter: r.RetryDelay}, nil
	}

	// Commit and push the deployment files
	commitMessage := fmt.Sprintf("Deploy NetworkIntent: %s/%s\n\nIntent: %s",
		networkIntent.Namespace, networkIntent.Name, networkIntent.Spec.Intent)

	commitHash, err := r.GitClient.CommitAndPush(deploymentFiles, commitMessage)
	if err != nil {
		logger.Error(err, "failed to commit and push deployment files")

		// Increment retry count
		setRetryCount(networkIntent, "git-deployment", retryCount+1)

		now := metav1.Now()
		networkIntent.Status.LastRetryTime = &now
		condition := metav1.Condition{
			Type:               "Deployed",
			Status:             metav1.ConditionFalse,
			Reason:             "GitCommitPushFailed",
			Message:            fmt.Sprintf("Failed to commit and push: %v", err),
			LastTransitionTime: now,
		}
		updateCondition(&networkIntent.Status.Conditions, condition)

		if updateErr := r.Status().Update(ctx, networkIntent); updateErr != nil {
			logger.Error(updateErr, "failed to update NetworkIntent status")
		}

		r.recordEvent(networkIntent, "Warning", "GitCommitPushFailed",
			fmt.Sprintf("Failed to commit and push deployment files: %v", err))

		return ctrl.Result{RequeueAfter: r.RetryDelay}, nil
	}

	// Clear retry count and update status to reflect successful deployment
	clearRetryCount(networkIntent, "git-deployment")
	now := metav1.Now()
	networkIntent.Status.DeploymentCompletionTime = &now
	networkIntent.Status.GitCommitHash = commitHash
	condition := metav1.Condition{
		Type:               "Deployed",
		Status:             metav1.ConditionTrue,
		Reason:             "GitDeploymentSucceeded",
		Message:            fmt.Sprintf("Configuration successfully deployed via GitOps (commit: %s)", commitHash[:8]),
		LastTransitionTime: now,
	}
	updateCondition(&networkIntent.Status.Conditions, condition)

	if err := r.Status().Update(ctx, networkIntent); err != nil {
		logger.Error(err, "failed to update NetworkIntent status")
		return ctrl.Result{}, err
	}

	r.recordEvent(networkIntent, "Normal", "GitDeploymentSucceeded", "Configuration successfully deployed via GitOps")
	logger.Info("NetworkIntent deployed successfully via GitOps", "intent", networkIntent.Spec.Intent)

	return ctrl.Result{}, nil
}

func (r *NetworkIntentReconciler) generateDeploymentFiles(networkIntent *nephoranv1.NetworkIntent) (map[string]string, error) {
	files := make(map[string]string)

	// Parse the processed parameters
	var parameters map[string]interface{}
	if len(networkIntent.Spec.Parameters.Raw) > 0 {
		if err := json.Unmarshal(networkIntent.Spec.Parameters.Raw, &parameters); err != nil {
			return nil, fmt.Errorf("failed to parse parameters: %w", err)
		}
	}

	// Set default deployment path if not configured
	deployPath := r.GitDeployPath
	if deployPath == "" {
		deployPath = "networkintents"
	}

	// Generate Kubernetes manifests based on parameters
	// This is a basic example - you would customize this based on your specific requirements
	manifest := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("networkintent-%s", networkIntent.Name),
			"namespace": networkIntent.Namespace,
			"labels": map[string]interface{}{
				"app.kubernetes.io/name":       "networkintent",
				"app.kubernetes.io/instance":   networkIntent.Name,
				"app.kubernetes.io/managed-by": "nephoran-intent-operator",
			},
		},
		"data": parameters,
	}

	manifestYAML, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}

	// Create file path in deployment repository
	filePath := fmt.Sprintf("%s/%s/%s-configmap.json", deployPath, networkIntent.Namespace, networkIntent.Name)
	files[filePath] = string(manifestYAML)

	return files, nil
}

// Helper functions for managing conditions and retry counts

func isConditionTrue(conditions []metav1.Condition, conditionType string) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType && condition.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}

func updateCondition(conditions *[]metav1.Condition, newCondition metav1.Condition) {
	for i, condition := range *conditions {
		if condition.Type == newCondition.Type {
			(*conditions)[i] = newCondition
			return
		}
	}
	*conditions = append(*conditions, newCondition)
}

func getRetryCount(networkIntent *nephoranv1.NetworkIntent, operation string) int {
	if networkIntent.Annotations == nil {
		return 0
	}

	key := fmt.Sprintf("nephoran.com/%s-retry-count", operation)
	if countStr, exists := networkIntent.Annotations[key]; exists {
		if count, err := fmt.Sscanf(countStr, "%d", new(int)); err == nil && count == 1 {
			var result int
			fmt.Sscanf(countStr, "%d", &result)
			return result
		}
	}
	return 0
}

func setRetryCount(networkIntent *nephoranv1.NetworkIntent, operation string, count int) {
	if networkIntent.Annotations == nil {
		networkIntent.Annotations = make(map[string]string)
	}

	key := fmt.Sprintf("nephoran.com/%s-retry-count", operation)
	networkIntent.Annotations[key] = fmt.Sprintf("%d", count)
}

func clearRetryCount(networkIntent *nephoranv1.NetworkIntent, operation string) {
	if networkIntent.Annotations == nil {
		return
	}

	key := fmt.Sprintf("nephoran.com/%s-retry-count", operation)
	delete(networkIntent.Annotations, key)
}

func (r *NetworkIntentReconciler) recordEvent(networkIntent *nephoranv1.NetworkIntent, eventType, reason, message string) {
	if r.EventRecorder != nil {
		r.EventRecorder.Event(networkIntent, eventType, reason, message)
	}
}

// handleDeletion handles resource cleanup when NetworkIntent is being deleted
func (r *NetworkIntentReconciler) handleDeletion(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Handling NetworkIntent deletion", "intent", networkIntent.Name)

	// Perform cleanup operations
	if err := r.cleanupResources(ctx, networkIntent); err != nil {
		logger.Error(err, "failed to cleanup resources")
		r.recordEvent(networkIntent, "Warning", "CleanupFailed", fmt.Sprintf("Failed to cleanup resources: %v", err))
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	// Remove finalizer to allow deletion
	networkIntent.Finalizers = removeFinalizer(networkIntent.Finalizers, NetworkIntentFinalizer)
	if err := r.Update(ctx, networkIntent); err != nil {
		logger.Error(err, "failed to remove finalizer")
		return ctrl.Result{}, err
	}

	logger.Info("NetworkIntent cleanup completed", "intent", networkIntent.Name)
	r.recordEvent(networkIntent, "Normal", "CleanupCompleted", "Successfully cleaned up resources")
	return ctrl.Result{}, nil
}

// cleanupResources performs cleanup of resources associated with the NetworkIntent
func (r *NetworkIntentReconciler) cleanupResources(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) error {
	logger := log.FromContext(ctx)

	// 1. Cleanup GitOps packages
	if r.GitClient != nil && r.GitRepoURL != "" {
		if err := r.cleanupGitOpsPackages(ctx, networkIntent); err != nil {
			logger.Error(err, "failed to cleanup GitOps packages")
			// Continue with other cleanup operations even if Git cleanup fails
		}
	}

	// 2. Cleanup any generated ConfigMaps or Secrets
	if err := r.cleanupGeneratedResources(ctx, networkIntent); err != nil {
		logger.Error(err, "failed to cleanup generated resources")
		return err
	}

	// 3. Cleanup any cached data related to this intent
	if err := r.cleanupCachedData(ctx, networkIntent); err != nil {
		logger.Error(err, "failed to cleanup cached data")
		// Continue - cache cleanup is not critical
	}

	return nil
}

// cleanupGitOpsPackages removes GitOps packages from the repository
func (r *NetworkIntentReconciler) cleanupGitOpsPackages(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) error {
	logger := log.FromContext(ctx)

	// Generate package name for cleanup
	packageName := fmt.Sprintf("%s-%s", networkIntent.Namespace, networkIntent.Name)
	packagePath := fmt.Sprintf("%s/%s", r.GitDeployPath, packageName)

	logger.Info("Cleaning up GitOps package", "package", packageName, "path", packagePath)

	// Use GitClient to remove the package directory
	if err := r.GitClient.RemoveDirectory(packagePath); err != nil {
		return fmt.Errorf("failed to remove GitOps package directory: %w", err)
	}

	// Commit the removal
	commitMessage := fmt.Sprintf("Remove NetworkIntent package: %s", packageName)
	if err := r.GitClient.CommitAndPush(commitMessage); err != nil {
		return fmt.Errorf("failed to commit package removal: %w", err)
	}

	return nil
}

// cleanupGeneratedResources removes any Kubernetes resources generated for this intent
func (r *NetworkIntentReconciler) cleanupGeneratedResources(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) error {
	logger := log.FromContext(ctx)

	// Define labels to identify resources generated by this intent
	labelSelector := map[string]string{
		"nephoran.com/created-by": "networkintent-controller",
		"nephoran.com/intent-name": networkIntent.Name,
		"nephoran.com/intent-namespace": networkIntent.Namespace,
	}

	// Clean up ConfigMaps
	if err := r.cleanupResourcesByLabel(ctx, &metav1.ListOptions{
		LabelSelector: createLabelSelector(labelSelector),
	}, "ConfigMap"); err != nil {
		logger.Error(err, "failed to cleanup ConfigMaps")
		return err
	}

	// Clean up Secrets
	if err := r.cleanupResourcesByLabel(ctx, &metav1.ListOptions{
		LabelSelector: createLabelSelector(labelSelector),
	}, "Secret"); err != nil {
		logger.Error(err, "failed to cleanup Secrets")
		return err
	}

	return nil
}

// cleanupCachedData removes any cached data related to the intent
func (r *NetworkIntentReconciler) cleanupCachedData(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) error {
	logger := log.FromContext(ctx)

	// If LLM processor is available, clear any cached processing results
	if r.LLMProcessorURL != "" && r.HTTPClient != nil {
		intentID := fmt.Sprintf("%s-%s", networkIntent.Namespace, networkIntent.Name)
		cleanupURL := fmt.Sprintf("%s/cache/clear/%s", r.LLMProcessorURL, intentID)
		
		req, err := http.NewRequestWithContext(ctx, "DELETE", cleanupURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create cache cleanup request: %w", err)
		}

		resp, err := r.HTTPClient.Do(req)
		if err != nil {
			logger.Info("Failed to clear LLM cache (non-critical)", "error", err)
			return nil // Non-critical failure
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
			logger.Info("LLM cache cleanup returned non-OK status (non-critical)", "status", resp.StatusCode)
		}
	}

	return nil
}

// Helper function to cleanup resources by label selector
func (r *NetworkIntentReconciler) cleanupResourcesByLabel(ctx context.Context, listOpts *metav1.ListOptions, resourceType string) error {
	// This is a simplified implementation
	// In a real implementation, you would use the appropriate client to list and delete resources
	logger := log.FromContext(ctx)
	logger.Info("Cleaning up resources by label", "resourceType", resourceType, "labelSelector", listOpts.LabelSelector)
	
	// TODO: Implement actual resource cleanup based on resource type
	// For now, this is a placeholder that logs the cleanup operation
	
	return nil
}

// Helper functions for finalizer management
func containsFinalizer(finalizers []string, finalizer string) bool {
	for _, f := range finalizers {
		if f == finalizer {
			return true
		}
	}
	return false
}

func removeFinalizer(finalizers []string, finalizer string) []string {
	result := make([]string, 0, len(finalizers))
	for _, f := range finalizers {
		if f != finalizer {
			result = append(result, f)
		}
	}
	return result
}

// Helper function to create label selector string
func createLabelSelector(labels map[string]string) string {
	var selectors []string
	for key, value := range labels {
		selectors = append(selectors, fmt.Sprintf("%s=%s", key, value))
	}
	return fmt.Sprintf("%v", selectors) // Simplified - real implementation would use proper formatting
}

func (r *NetworkIntentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nephoranv1.NetworkIntent{}).
		Complete(r)
}
