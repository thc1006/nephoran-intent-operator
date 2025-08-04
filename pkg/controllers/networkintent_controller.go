package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/git"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// Constants for the controller
const (
	NetworkIntentFinalizer = "networkintent.nephoran.com/finalizer"

	// Default configuration values
	DefaultMaxRetries    = 3
	DefaultRetryDelay    = 30 * time.Second
	DefaultTimeout       = 5 * time.Minute
	DefaultGitDeployPath = "networkintents"

	// Validation limits
	MaxAllowedRetries    = 10
	MaxAllowedRetryDelay = time.Hour
)

// Dependencies interface defines the external dependencies for the controller
type Dependencies interface {
	GetGitClient() git.ClientInterface
	GetLLMClient() shared.ClientInterface
	GetPackageGenerator() *nephio.PackageGenerator
	GetHTTPClient() *http.Client
	GetEventRecorder() record.EventRecorder
}

// Config holds the configuration for the NetworkIntentReconciler
type Config struct {
	MaxRetries      int
	RetryDelay      time.Duration
	Timeout         time.Duration
	GitRepoURL      string
	GitBranch       string
	GitDeployPath   string
	LLMProcessorURL string
	UseNephioPorch  bool
}

// NetworkIntentReconciler orchestrates the reconciliation of NetworkIntent resources
type NetworkIntentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	deps   Dependencies
	config *Config
}

// NewNetworkIntentReconciler creates a new NetworkIntentReconciler with dependency injection
func NewNetworkIntentReconciler(client client.Client, scheme *runtime.Scheme, deps Dependencies, config *Config) (*NetworkIntentReconciler, error) {
	if client == nil {
		return nil, fmt.Errorf("client cannot be nil")
	}
	if scheme == nil {
		return nil, fmt.Errorf("scheme cannot be nil")
	}
	if deps == nil {
		return nil, fmt.Errorf("dependencies cannot be nil")
	}

	// Validate and set defaults for config
	validatedConfig, err := validateAndSetConfigDefaults(config)
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &NetworkIntentReconciler{
		Client: client,
		Scheme: scheme,
		deps:   deps,
		config: validatedConfig,
	}, nil
}

// validateAndSetConfigDefaults validates and sets default values for the configuration
func validateAndSetConfigDefaults(config *Config) (*Config, error) {
	if config == nil {
		config = &Config{}
	}

	// Set defaults
	if config.MaxRetries <= 0 {
		config.MaxRetries = DefaultMaxRetries
	}
	if config.RetryDelay <= 0 {
		config.RetryDelay = DefaultRetryDelay
	}
	if config.Timeout <= 0 {
		config.Timeout = DefaultTimeout
	}
	if config.GitDeployPath == "" {
		config.GitDeployPath = DefaultGitDeployPath
	}

	// Validate limits
	if config.MaxRetries > MaxAllowedRetries {
		return nil, fmt.Errorf("MaxRetries (%d) exceeds maximum allowed value (%d)", config.MaxRetries, MaxAllowedRetries)
	}
	if config.RetryDelay > MaxAllowedRetryDelay {
		return nil, fmt.Errorf("RetryDelay (%v) exceeds maximum allowed value (%v)", config.RetryDelay, MaxAllowedRetryDelay)
	}

	return config, nil
}

//+kubebuilder:rbac:groups=nephoran.com,resources=networkintents,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nephoran.com,resources=networkintents/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nephoran.com,resources=networkintents/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create

func (r *NetworkIntentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Add request ID for better tracking
	reqID := fmt.Sprintf("req-%d", time.Now().UnixNano())
	logger = logger.WithValues("request_id", reqID)

	// Check for context cancellation early
	select {
	case <-ctx.Done():
		logger.Info("Reconciliation cancelled due to context cancellation")
		return ctrl.Result{}, ctx.Err()
	default:
	}

	logger.V(1).Info("Starting reconciliation", "namespace", req.Namespace, "name", req.Name)

	// Fetch the NetworkIntent instance with timeout
	var networkIntent nephoranv1.NetworkIntent
	if err := r.safeGet(ctx, req.NamespacedName, &networkIntent); err != nil {
		if client.IgnoreNotFound(err) == nil {
			logger.V(1).Info("NetworkIntent not found, likely deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "failed to fetch NetworkIntent")
		return ctrl.Result{RequeueAfter: time.Minute}, fmt.Errorf("failed to fetch NetworkIntent: %w", err)
	}

	// Handle deletion with proper context checking
	if networkIntent.DeletionTimestamp != nil {
		logger.Info("NetworkIntent is being deleted, handling cleanup")
		return r.handleDeletion(ctx, &networkIntent)
	}

	// Add finalizer if it doesn't exist, with context checking
	if !containsFinalizer(networkIntent.Finalizers, NetworkIntentFinalizer) {
		logger.V(1).Info("Adding finalizer to NetworkIntent")
		networkIntent.Finalizers = append(networkIntent.Finalizers, NetworkIntentFinalizer)
		if err := r.safeUpdate(ctx, &networkIntent); err != nil {
			logger.Error(err, "failed to add finalizer")
			return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to add finalizer: %w", err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Configuration is already validated in constructor, no need to validate again

	// Update observed generation with error handling
	networkIntent.Status.ObservedGeneration = networkIntent.Generation

	// Check if already completed to avoid unnecessary work
	if isConditionTrue(networkIntent.Status.Conditions, "Processed") &&
		isConditionTrue(networkIntent.Status.Conditions, "Deployed") &&
		networkIntent.Status.Phase == "Completed" {
		logger.V(1).Info("NetworkIntent already processed and deployed, skipping")
		return ctrl.Result{}, nil
	}

	// Step 1: Process the intent using LLM client with enhanced error handling
	if !isConditionTrue(networkIntent.Status.Conditions, "Processed") {
		select {
		case <-ctx.Done():
			logger.Info("Context cancelled during processing phase")
			return ctrl.Result{}, ctx.Err()
		default:
		}

		// Set processing phase with proper error handling
		if err := r.updatePhase(ctx, &networkIntent, "Processing"); err != nil {
			logger.Error(err, "failed to update processing phase")
			return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update processing phase: %w", err)
		}

		result, err := r.processIntentWithRetry(ctx, &networkIntent)
		if err != nil {
			logger.Error(err, "intent processing failed")
			r.recordFailureEvent(&networkIntent, "ProcessingFailed", err.Error())
			return result, err
		}
		if result.Requeue || result.RequeueAfter > 0 {
			logger.V(1).Info("Intent processing requires requeue", "requeue", result.Requeue, "requeue_after", result.RequeueAfter)
			return result, nil
		}
	}

	// Step 2: Deploy using GitOps with enhanced error handling
	if isConditionTrue(networkIntent.Status.Conditions, "Processed") &&
		!isConditionTrue(networkIntent.Status.Conditions, "Deployed") {

		select {
		case <-ctx.Done():
			logger.Info("Context cancelled during deployment phase")
			return ctrl.Result{}, ctx.Err()
		default:
		}

		// Set deploying phase with proper error handling
		if err := r.updatePhase(ctx, &networkIntent, "Deploying"); err != nil {
			logger.Error(err, "failed to update deploying phase")
			return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update deploying phase: %w", err)
		}

		result, err := r.deployViaGitOps(ctx, &networkIntent)
		if err != nil {
			logger.Error(err, "deployment failed")
			r.recordFailureEvent(&networkIntent, "DeploymentFailed", err.Error())
			return result, err
		}
		if result.Requeue || result.RequeueAfter > 0 {
			logger.V(1).Info("Deployment requires requeue", "requeue", result.Requeue, "requeue_after", result.RequeueAfter)
			return result, nil
		}
	}

	// Final status update with comprehensive error handling
	if err := r.updatePhase(ctx, &networkIntent, "Completed"); err != nil {
		logger.Error(err, "failed to update completion phase")
		return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update completion phase: %w", err)
	}

	r.recordEvent(&networkIntent, "Normal", "ReconciliationCompleted", "NetworkIntent successfully processed and deployed")
	logger.Info("NetworkIntent reconciliation completed successfully",
		"intent", networkIntent.Spec.Intent,
		"phase", networkIntent.Status.Phase,
		"request_id", reqID)

	return ctrl.Result{}, nil
}

func (r *NetworkIntentReconciler) processIntentWithRetry(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("phase", "processing")

	// Check for context cancellation before starting
	select {
	case <-ctx.Done():
		return ctrl.Result{}, ctx.Err()
	default:
	}

	// Validate LLM client with comprehensive error handling
	llmClient := r.deps.GetLLMClient()
	if llmClient == nil {
		err := fmt.Errorf("LLM client is not configured")
		logger.Error(err, "LLM client validation failed")
		r.recordFailureEvent(networkIntent, "LLMClientNotConfigured", err.Error())

		// Mark as permanently failed if no client
		condition := metav1.Condition{
			Type:               "Processed",
			Status:             metav1.ConditionFalse,
			Reason:             "LLMClientNotConfigured",
			Message:            "LLM client is not configured and cannot process intent",
			LastTransitionTime: metav1.Now(),
		}
		updateCondition(&networkIntent.Status.Conditions, condition)

		if statusErr := r.safeStatusUpdate(ctx, networkIntent); statusErr != nil {
			logger.Error(statusErr, "failed to update status after LLM client validation failure")
		}

		return ctrl.Result{}, fmt.Errorf("LLM client not configured: %w", err)
	}

	// Get current retry count with error handling
	retryCount := getRetryCount(networkIntent, "llm-processing")
	logger.V(1).Info("Processing intent with LLM", "retry_count", retryCount, "max_retries", r.config.MaxRetries)

	// Check if max retries exceeded
	if retryCount >= r.config.MaxRetries {
		err := fmt.Errorf("max retries (%d) exceeded for LLM processing", r.config.MaxRetries)
		logger.Error(err, "max retries reached")

		condition := metav1.Condition{
			Type:               "Processed",
			Status:             metav1.ConditionFalse,
			Reason:             "LLMProcessingFailedMaxRetries",
			Message:            fmt.Sprintf("Failed to process intent after %d retries", r.config.MaxRetries),
			LastTransitionTime: metav1.Now(),
		}
		updateCondition(&networkIntent.Status.Conditions, condition)

		if statusErr := r.safeStatusUpdate(ctx, networkIntent); statusErr != nil {
			logger.Error(statusErr, "failed to update status after max retries exceeded")
			return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update status: %w", statusErr)
		}

		r.recordFailureEvent(networkIntent, "LLMProcessingFailed", err.Error())
		return ctrl.Result{}, err
	}

	// Validate intent before processing
	if networkIntent.Spec.Intent == "" {
		err := fmt.Errorf("intent specification is empty")
		logger.Error(err, "intent validation failed")
		r.recordFailureEvent(networkIntent, "InvalidIntent", err.Error())

		condition := metav1.Condition{
			Type:               "Processed",
			Status:             metav1.ConditionFalse,
			Reason:             "InvalidIntent",
			Message:            "Intent specification is empty",
			LastTransitionTime: metav1.Now(),
		}
		updateCondition(&networkIntent.Status.Conditions, condition)

		if statusErr := r.safeStatusUpdate(ctx, networkIntent); statusErr != nil {
			logger.Error(statusErr, "failed to update status after intent validation failure")
		}

		return ctrl.Result{}, err
	}

	// Create a timeout context for LLM processing
	processingCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)
	defer cancel()

	// Attempt LLM processing with enhanced error handling
	logger.Info("Starting LLM processing", "intent", networkIntent.Spec.Intent, "attempt", retryCount+1)
	processedResult, err := llmClient.ProcessIntent(processingCtx, networkIntent.Spec.Intent)
	if err != nil {
		logger.Error(err, "LLM processing failed", "retry", retryCount+1, "intent", networkIntent.Spec.Intent)

		// Increment retry count
		setRetryCount(networkIntent, "llm-processing", retryCount+1)

		// Update status with detailed retry information
		now := metav1.Now()
		networkIntent.Status.LastRetryTime = &now

		var reason, message string
		if ctx.Err() != nil {
			reason = "LLMProcessingContextCancelled"
			message = fmt.Sprintf("LLM processing cancelled (attempt %d/%d): %v", retryCount+1, r.config.MaxRetries, ctx.Err())
		} else {
			reason = "LLMProcessingRetrying"
			message = fmt.Sprintf("LLM processing failed (attempt %d/%d): %v", retryCount+1, r.config.MaxRetries, err)
		}

		condition := metav1.Condition{
			Type:               "Processed",
			Status:             metav1.ConditionFalse,
			Reason:             reason,
			Message:            message,
			LastTransitionTime: now,
		}
		updateCondition(&networkIntent.Status.Conditions, condition)

		if statusErr := r.safeStatusUpdate(ctx, networkIntent); statusErr != nil {
			logger.Error(statusErr, "failed to update status after LLM processing failure")
			return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update status: %w", statusErr)
		}

		r.recordFailureEvent(networkIntent, "LLMProcessingRetry",
			fmt.Sprintf("attempt %d/%d failed: %v", retryCount+1, r.config.MaxRetries, err))

		// Calculate exponential backoff delay
		backoffDelay := time.Duration(retryCount+1) * r.config.RetryDelay
		if backoffDelay > time.Minute*10 {
			backoffDelay = time.Minute * 10 // Cap at 10 minutes
		}

		logger.V(1).Info("Scheduling retry", "delay", backoffDelay, "attempt", retryCount+1)
		return ctrl.Result{RequeueAfter: backoffDelay}, nil
	}

	// Validate and parse the LLM response
	logger.V(1).Info("Parsing LLM response", "response_length", len(processedResult))

	if processedResult == "" {
		err := fmt.Errorf("LLM returned empty response")
		logger.Error(err, "LLM response validation failed")
		r.recordFailureEvent(networkIntent, "EmptyLLMResponse", err.Error())

		// Treat as retry-able error
		setRetryCount(networkIntent, "llm-processing", retryCount+1)
		condition := metav1.Condition{
			Type:               "Processed",
			Status:             metav1.ConditionFalse,
			Reason:             "EmptyLLMResponse",
			Message:            "LLM returned empty response",
			LastTransitionTime: metav1.Now(),
		}
		updateCondition(&networkIntent.Status.Conditions, condition)

		if statusErr := r.safeStatusUpdate(ctx, networkIntent); statusErr != nil {
			logger.Error(statusErr, "failed to update status after empty response")
		}

		return ctrl.Result{RequeueAfter: r.config.RetryDelay}, nil
	}

	var parameters map[string]interface{}
	if err := json.Unmarshal([]byte(processedResult), &parameters); err != nil {
		logger.Error(err, "failed to parse LLM response as JSON", "response", processedResult)

		// Increment retry count for parsing failure
		setRetryCount(networkIntent, "llm-processing", retryCount+1)

		condition := metav1.Condition{
			Type:               "Processed",
			Status:             metav1.ConditionFalse,
			Reason:             "LLMResponseParsingFailed",
			Message:            fmt.Sprintf("Failed to parse LLM response as JSON: %v", err),
			LastTransitionTime: metav1.Now(),
		}
		updateCondition(&networkIntent.Status.Conditions, condition)

		if statusErr := r.safeStatusUpdate(ctx, networkIntent); statusErr != nil {
			logger.Error(statusErr, "failed to update status after JSON parsing failure")
			return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update status: %w", statusErr)
		}

		r.recordFailureEvent(networkIntent, "LLMResponseParsingFailed",
			fmt.Sprintf("Failed to parse LLM response: %v", err))

		return ctrl.Result{RequeueAfter: r.config.RetryDelay}, nil
	}

	// Validate that we have meaningful parameters
	if len(parameters) == 0 {
		logger.Info("LLM returned empty parameters object", "response", processedResult)
	}

	// Update NetworkIntent with processed parameters
	parametersRaw, marshalErr := json.Marshal(parameters)
	if marshalErr != nil {
		logger.Error(marshalErr, "failed to marshal processed parameters")
		return ctrl.Result{RequeueAfter: r.config.RetryDelay}, fmt.Errorf("failed to marshal parameters: %w", marshalErr)
	}

	networkIntent.Spec.Parameters = runtime.RawExtension{Raw: parametersRaw}

	if err := r.safeUpdate(ctx, networkIntent); err != nil {
		logger.Error(err, "failed to update NetworkIntent with processed parameters")
		return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update NetworkIntent with parameters: %w", err)
	}

	// Clear retry count and update status to reflect successful processing
	clearRetryCount(networkIntent, "llm-processing")
	now := metav1.Now()
	networkIntent.Status.ProcessingCompletionTime = &now
	condition := metav1.Condition{
		Type:               "Processed",
		Status:             metav1.ConditionTrue,
		Reason:             "LLMProcessingSucceeded",
		Message:            "Intent successfully processed by LLM and parameters extracted",
		LastTransitionTime: now,
	}
	updateCondition(&networkIntent.Status.Conditions, condition)

	if err := r.safeStatusUpdate(ctx, networkIntent); err != nil {
		logger.Error(err, "failed to update NetworkIntent status after successful processing")
		return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update status: %w", err)
	}

	r.recordEvent(networkIntent, "Normal", "LLMProcessingSucceeded",
		fmt.Sprintf("Intent successfully processed by LLM with %d parameters", len(parameters)))
	logger.Info("NetworkIntent processed successfully",
		"intent", networkIntent.Spec.Intent,
		"parameters_count", len(parameters),
		"processing_time", time.Since(time.Now()))

	return ctrl.Result{}, nil
}

func (r *NetworkIntentReconciler) deployViaGitOps(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	gitClient := r.deps.GetGitClient()
	if gitClient == nil {
		err := fmt.Errorf("Git client is not configured")
		logger.Error(err, "Git client not configured")
		r.recordEvent(networkIntent, "Warning", "GitClientNotConfigured", "Git client is not configured")
		return ctrl.Result{RequeueAfter: time.Minute * 5}, fmt.Errorf("git client not configured: %w", err)
	}

	// Get current retry count from annotations
	retryCount := getRetryCount(networkIntent, "git-deployment")

	if retryCount >= r.config.MaxRetries {
		// Max retries reached, mark as failed
		condition := metav1.Condition{
			Type:               "Deployed",
			Status:             metav1.ConditionFalse,
			Reason:             "GitDeploymentFailedMaxRetries",
			Message:            fmt.Sprintf("Failed to deploy via GitOps after %d retries", r.config.MaxRetries),
			LastTransitionTime: metav1.Now(),
		}
		updateCondition(&networkIntent.Status.Conditions, condition)

		if err := r.Status().Update(ctx, networkIntent); err != nil {
			logger.Error(err, "failed to update NetworkIntent status")
			return ctrl.Result{}, err
		}

		r.recordEvent(networkIntent, "Warning", "GitDeploymentFailed",
			fmt.Sprintf("Failed to deploy via GitOps after %d retries", r.config.MaxRetries))
		return ctrl.Result{}, nil
	}

	// Generate deployment files from processed parameters
	var deploymentFiles map[string]string
	var err error

	packageGen := r.deps.GetPackageGenerator()
	if r.config.UseNephioPorch && packageGen != nil {
		// Use Nephio package generator for KRM package generation
		logger.Info("generating Nephio KRM package")
		deploymentFiles, err = packageGen.GeneratePackage(networkIntent)
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

		return ctrl.Result{RequeueAfter: r.config.RetryDelay}, nil
	}

	// Initialize Git repository
	if err := gitClient.InitRepo(); err != nil {
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

		return ctrl.Result{RequeueAfter: r.config.RetryDelay}, nil
	}

	// Commit and push the deployment files
	commitMessage := fmt.Sprintf("Deploy NetworkIntent: %s/%s\n\nIntent: %s",
		networkIntent.Namespace, networkIntent.Name, networkIntent.Spec.Intent)

	commitHash, err := gitClient.CommitAndPush(deploymentFiles, commitMessage)
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

		return ctrl.Result{RequeueAfter: r.config.RetryDelay}, nil
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

	// Use configured deployment path
	deployPath := r.config.GitDeployPath

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

// handleDeletion handles resource cleanup when NetworkIntent is being deleted
func (r *NetworkIntentReconciler) handleDeletion(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Handling NetworkIntent deletion", "intent", networkIntent.Name)

	// Perform cleanup operations - continue even if some cleanup fails
	cleanupErrors := []error{}
	if err := r.cleanupResources(ctx, networkIntent); err != nil {
		logger.Error(err, "failed to cleanup resources, but continuing with finalizer removal")
		r.recordEvent(networkIntent, "Warning", "CleanupFailed", fmt.Sprintf("Failed to cleanup resources: %v", err))
		cleanupErrors = append(cleanupErrors, err)
		// Continue to remove finalizer instead of returning error
	}

	// Remove finalizer to allow deletion even if cleanup had issues
	networkIntent.Finalizers = removeFinalizer(networkIntent.Finalizers, NetworkIntentFinalizer)
	if err := r.Update(ctx, networkIntent); err != nil {
		logger.Error(err, "failed to remove finalizer")
		return ctrl.Result{}, err
	}

	if len(cleanupErrors) > 0 {
		logger.Info("NetworkIntent cleanup completed with some errors", "intent", networkIntent.Name, "errors", len(cleanupErrors))
		r.recordEvent(networkIntent, "Warning", "CleanupCompletedWithErrors", "Resource cleanup completed but some operations failed")
	} else {
		logger.Info("NetworkIntent cleanup completed successfully", "intent", networkIntent.Name)
		r.recordEvent(networkIntent, "Normal", "CleanupCompleted", "Successfully cleaned up resources")
	}
	return ctrl.Result{}, nil
}

// cleanupResources performs cleanup of resources associated with the NetworkIntent
func (r *NetworkIntentReconciler) cleanupResources(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) error {
	logger := log.FromContext(ctx)
	var aggregatedErrors []error

	// 1. Cleanup GitOps packages
	gitClient := r.deps.GetGitClient()
	if gitClient != nil && r.config.GitRepoURL != "" {
		if err := r.cleanupGitOpsPackages(ctx, networkIntent, gitClient); err != nil {
			logger.Error(err, "failed to cleanup GitOps packages")
			aggregatedErrors = append(aggregatedErrors, fmt.Errorf("GitOps cleanup failed: %w", err))
			// Continue with other cleanup operations even if Git cleanup fails
		}
	} else {
		logger.V(1).Info("Skipping GitOps cleanup - no Git client or repository configured")
	}

	// 2. Cleanup any generated ConfigMaps or Secrets
	if err := r.cleanupGeneratedResources(ctx, networkIntent); err != nil {
		logger.Error(err, "failed to cleanup generated resources")
		aggregatedErrors = append(aggregatedErrors, fmt.Errorf("generated resources cleanup failed: %w", err))
		// Continue with cache cleanup even if this fails
	}

	// 3. Cleanup any cached data related to this intent
	if err := r.cleanupCachedData(ctx, networkIntent); err != nil {
		logger.Error(err, "failed to cleanup cached data")
		aggregatedErrors = append(aggregatedErrors, fmt.Errorf("cache cleanup failed: %w", err))
		// Cache cleanup is not critical - continue
	}

	// Return aggregated errors if any occurred, but allow the controller to continue
	if len(aggregatedErrors) > 0 {
		return fmt.Errorf("cleanup completed with %d errors: %v", len(aggregatedErrors), aggregatedErrors)
	}

	return nil
}

// cleanupGitOpsPackages removes GitOps packages from the repository
func (r *NetworkIntentReconciler) cleanupGitOpsPackages(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, gitClient git.ClientInterface) error {
	logger := log.FromContext(ctx)

	// Check context cancellation before starting
	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled during GitOps cleanup: %w", ctx.Err())
	default:
	}

	// Construct the package path using the configured deployment path
	packageName := fmt.Sprintf("%s-%s", networkIntent.Namespace, networkIntent.Name)
	packagePath := fmt.Sprintf("%s/%s", r.config.GitDeployPath, packageName)

	logger.Info("Starting GitOps package cleanup", "package", packageName, "path", packagePath, "git_repo", r.config.GitRepoURL)

	// Initialize the repository first to ensure it's available
	if err := gitClient.InitRepo(); err != nil {
		logger.Error(err, "failed to initialize git repository for cleanup")
		// If we can't initialize the repo, the directory likely doesn't exist anyway
		// Log the error but don't fail the cleanup
		logger.V(1).Info("Git repository initialization failed, directory may not exist - skipping cleanup gracefully")
		return nil // Return nil to indicate graceful skip
	}

	// Check context cancellation after repo initialization
	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled after git initialization: %w", ctx.Err())
	default:
	}

	// Create commit message with detailed context
	commitMessage := fmt.Sprintf("Remove NetworkIntent package: %s/%s\n\nIntent: %s\nCleanup triggered by NetworkIntent deletion",
		networkIntent.Namespace, networkIntent.Name, networkIntent.Spec.Intent)

	// Use GitClient to remove the package directory and commit in one operation
	// The RemoveDirectory method handles both directory removal and commit/push
	logger.V(1).Info("Removing directory from Git repository", "directory", packagePath)
	if err := gitClient.RemoveDirectory(packagePath, commitMessage); err != nil {
		// Check if this is a "directory doesn't exist" type error and handle gracefully
		if isDirectoryNotExistError(err) {
			logger.V(1).Info("GitOps package directory does not exist, cleanup not needed", "package", packageName, "path", packagePath)
			return nil // Gracefully skip if directory doesn't exist
		}
		logger.Error(err, "failed to remove GitOps package directory", "package", packageName, "path", packagePath)
		return fmt.Errorf("failed to remove GitOps package directory '%s': %w", packagePath, err)
	}

	logger.Info("Successfully cleaned up GitOps package", "package", packageName, "path", packagePath)
	return nil
}

// cleanupGeneratedResources removes any Kubernetes resources generated for this intent
func (r *NetworkIntentReconciler) cleanupGeneratedResources(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) error {
	logger := log.FromContext(ctx)

	// Check context cancellation before starting
	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled during resource cleanup: %w", ctx.Err())
	default:
	}

	logger.Info("Starting cleanup of generated Kubernetes resources", "intent_name", networkIntent.Name, "namespace", networkIntent.Namespace)

	// Define labels to identify resources generated by this intent
	labelSelector := map[string]string{
		"app.kubernetes.io/name":       "networkintent",
		"app.kubernetes.io/instance":   networkIntent.Name,
		"app.kubernetes.io/managed-by": "nephoran-intent-operator",
	}

	// Clean up ConfigMaps generated by this NetworkIntent
	if err := r.cleanupConfigMaps(ctx, networkIntent, labelSelector); err != nil {
		logger.Error(err, "failed to cleanup ConfigMaps")
		return fmt.Errorf("failed to cleanup ConfigMaps for NetworkIntent %s/%s: %w", networkIntent.Namespace, networkIntent.Name, err)
	}

	// Clean up Secrets generated by this NetworkIntent
	if err := r.cleanupSecrets(ctx, networkIntent, labelSelector); err != nil {
		logger.Error(err, "failed to cleanup Secrets")
		return fmt.Errorf("failed to cleanup Secrets for NetworkIntent %s/%s: %w", networkIntent.Namespace, networkIntent.Name, err)
	}

	// Clean up any additional custom resources if needed
	if err := r.cleanupCustomResources(ctx, networkIntent); err != nil {
		logger.Error(err, "failed to cleanup custom resources")
		// Log but don't fail the cleanup - custom resources might not exist
		logger.V(1).Info("Custom resource cleanup failed, continuing with other cleanup operations")
	}

	logger.Info("Successfully cleaned up generated Kubernetes resources", "intent_name", networkIntent.Name)
	return nil
}

// cleanupCachedData removes any cached data related to the intent
func (r *NetworkIntentReconciler) cleanupCachedData(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) error {
	logger := log.FromContext(ctx)

	// If LLM processor is available, clear any cached processing results
	httpClient := r.deps.GetHTTPClient()
	if r.config.LLMProcessorURL != "" && httpClient != nil {
		intentID := fmt.Sprintf("%s-%s", networkIntent.Namespace, networkIntent.Name)
		cleanupURL := fmt.Sprintf("%s/cache/clear/%s", r.config.LLMProcessorURL, intentID)

		req, err := http.NewRequestWithContext(ctx, "DELETE", cleanupURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create cache cleanup request: %w", err)
		}

		resp, err := httpClient.Do(req)
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

// cleanupConfigMaps removes ConfigMaps generated by this NetworkIntent
func (r *NetworkIntentReconciler) cleanupConfigMaps(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, labelSelector map[string]string) error {
	logger := log.FromContext(ctx)

	// Create a ConfigMap list to find resources to delete
	configMapList := &corev1.ConfigMapList{}

	// List ConfigMaps with the label selector
	listOpts := []client.ListOption{
		client.InNamespace(networkIntent.Namespace),
		client.MatchingLabels(labelSelector),
	}

	if err := r.List(ctx, configMapList, listOpts...); err != nil {
		logger.Error(err, "failed to list ConfigMaps for cleanup")
		return fmt.Errorf("failed to list ConfigMaps for cleanup: %w", err)
	}

	// Delete each ConfigMap found
	for i := range configMapList.Items {
		configMap := &configMapList.Items[i]
		logger.V(1).Info("Deleting ConfigMap", "name", configMap.Name, "namespace", configMap.Namespace)
		if err := r.Delete(ctx, configMap); err != nil {
			if client.IgnoreNotFound(err) != nil {
				logger.Error(err, "failed to delete ConfigMap", "name", configMap.Name)
				return fmt.Errorf("failed to delete ConfigMap %s: %w", configMap.Name, err)
			}
		}
	}

	logger.V(1).Info("ConfigMap cleanup completed", "count", len(configMapList.Items))
	return nil
}

// cleanupSecrets removes Secrets generated by this NetworkIntent
func (r *NetworkIntentReconciler) cleanupSecrets(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, labelSelector map[string]string) error {
	logger := log.FromContext(ctx)

	// Create a Secret list to find resources to delete
	secretList := &corev1.SecretList{}

	// List Secrets with the label selector
	listOpts := []client.ListOption{
		client.InNamespace(networkIntent.Namespace),
		client.MatchingLabels(labelSelector),
	}

	if err := r.List(ctx, secretList, listOpts...); err != nil {
		logger.Error(err, "failed to list Secrets for cleanup")
		return fmt.Errorf("failed to list Secrets for cleanup: %w", err)
	}

	// Delete each Secret found
	for i := range secretList.Items {
		secret := &secretList.Items[i]
		logger.V(1).Info("Deleting Secret", "name", secret.Name, "namespace", secret.Namespace)
		if err := r.Delete(ctx, secret); err != nil {
			if client.IgnoreNotFound(err) != nil {
				logger.Error(err, "failed to delete Secret", "name", secret.Name)
				return fmt.Errorf("failed to delete Secret %s: %w", secret.Name, err)
			}
		}
	}

	logger.V(1).Info("Secret cleanup completed", "count", len(secretList.Items))
	return nil
}

// cleanupCustomResources removes any custom resources generated by this NetworkIntent
// Currently a placeholder for future extensibility
func (r *NetworkIntentReconciler) cleanupCustomResources(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) error {
	// Placeholder for future custom resource cleanup implementation
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

// safeGet performs a get operation with proper error wrapping and context checking
func (r *NetworkIntentReconciler) safeGet(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled during get operation: %w", ctx.Err())
	default:
	}

	if err := r.Get(ctx, key, obj); err != nil {
		return fmt.Errorf("failed to get object %s/%s: %w", key.Namespace, key.Name, err)
	}
	return nil
}

// safeUpdate performs an update operation with proper error wrapping and context checking
func (r *NetworkIntentReconciler) safeUpdate(ctx context.Context, obj client.Object) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled during update operation: %w", ctx.Err())
	default:
	}

	if err := r.Update(ctx, obj); err != nil {
		return fmt.Errorf("failed to update object: %w", err)
	}
	return nil
}

// safeStatusUpdate performs a status update operation with proper error wrapping and context checking
func (r *NetworkIntentReconciler) safeStatusUpdate(ctx context.Context, obj client.Object) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled during status update operation: %w", ctx.Err())
	default:
	}

	if err := r.Status().Update(ctx, obj); err != nil {
		return fmt.Errorf("failed to update object status: %w", err)
	}
	return nil
}

// updatePhase updates the NetworkIntent phase with proper error handling
func (r *NetworkIntentReconciler) updatePhase(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, phase string) error {
	if networkIntent.Status.Phase == phase {
		return nil // No update needed
	}

	networkIntent.Status.Phase = phase
	now := metav1.Now()

	switch phase {
	case "Processing":
		if networkIntent.Status.ProcessingStartTime == nil {
			networkIntent.Status.ProcessingStartTime = &now
		}
	case "Deploying":
		if networkIntent.Status.DeploymentStartTime == nil {
			networkIntent.Status.DeploymentStartTime = &now
		}
	case "Completed":
		if networkIntent.Status.DeploymentCompletionTime == nil {
			networkIntent.Status.DeploymentCompletionTime = &now
		}
	}

	return r.safeStatusUpdate(ctx, networkIntent)
}

// recordEvent records an event for the NetworkIntent
func (r *NetworkIntentReconciler) recordEvent(networkIntent *nephoranv1.NetworkIntent, eventType, reason, message string) {
	if eventRecorder := r.deps.GetEventRecorder(); eventRecorder != nil {
		eventRecorder.Event(networkIntent, eventType, reason, message)
	}
}

// recordFailureEvent records a failure event with additional context
func (r *NetworkIntentReconciler) recordFailureEvent(networkIntent *nephoranv1.NetworkIntent, reason, message string) {
	fullMessage := fmt.Sprintf("%s: %s", reason, message)
	r.recordEvent(networkIntent, "Warning", reason, fullMessage)
}

// isDirectoryNotExistError checks if an error indicates that a directory doesn't exist
// This helps distinguish between "directory not found" and actual Git operation failures
func isDirectoryNotExistError(err error) bool {
	if err == nil {
		return false
	}

	errorStr := err.Error()
	// Check for common "directory not exist" error patterns
	return strings.Contains(errorStr, "no such file or directory") ||
		strings.Contains(errorStr, "cannot find the path") ||
		strings.Contains(errorStr, "does not exist") ||
		strings.Contains(errorStr, "not found")
}

func (r *NetworkIntentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nephoranv1.NetworkIntent{}).
		Complete(r)
}
