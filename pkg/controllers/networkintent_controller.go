package controllers

import (
	
	"encoding/json"
"context"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	configPkg "github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/git"
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio"
	"github.com/thc1006/nephoran-intent-operator/pkg/resilience"
	"github.com/thc1006/nephoran-intent-operator/pkg/security"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
	"github.com/thc1006/nephoran-intent-operator/pkg/telecom"
)

// Dependencies interface defines the external dependencies for the controller.

// This interface is implemented by the injection.Container.

type Dependencies interface {
	GetGitClient() git.ClientInterface

	GetLLMClient() shared.ClientInterface

	GetPackageGenerator() *nephio.PackageGenerator

	GetHTTPClient() *http.Client

	GetEventRecorder() record.EventRecorder

	GetTelecomKnowledgeBase() *telecom.TelecomKnowledgeBase

	GetMetricsCollector() monitoring.MetricsCollector
}

// GitClientInterface is an alias to the git package ClientInterface.

type GitClientInterface = git.ClientInterface

// Config holds the configuration for the NetworkIntentReconciler.

type Config struct {
	MaxRetries int

	RetryDelay time.Duration

	Timeout time.Duration

	GitRepoURL string

	GitBranch string

	GitDeployPath string

	LLMProcessorURL string

	UseNephioPorch bool

	// Configuration constants reference.

	Constants *configPkg.Constants
}

// ProcessingPhase represents the current phase of intent processing.

type ProcessingPhase string

const (

	// PhaseLLMProcessing holds phasellmprocessing value.

	PhaseLLMProcessing ProcessingPhase = "LLMProcessing"

	// PhaseResourcePlanning holds phaseresourceplanning value.

	PhaseResourcePlanning ProcessingPhase = "ResourcePlanning"

	// PhaseManifestGeneration holds phasemanifestgeneration value.

	PhaseManifestGeneration ProcessingPhase = "ManifestGeneration"

	// PhaseGitOpsCommit holds phasegitopscommit value.

	PhaseGitOpsCommit ProcessingPhase = "GitOpsCommit"

	// PhaseDeploymentVerification holds phasedeploymentverification value.

	PhaseDeploymentVerification ProcessingPhase = "DeploymentVerification"
)

// ProcessingContext holds context information for multi-phase processing.

type ProcessingContext struct {
	StartTime time.Time

	CurrentPhase ProcessingPhase

	IntentType string

	ExtractedEntities map[string]interface{}

	TelecomContext map[string]interface{}

	ResourcePlan *ResourcePlan

	Manifests map[string]string

	GitCommitHash string

	DeploymentStatus map[string]interface{}

	Metrics map[string]float64
}

// ResourcePlan represents the planned resources for deployment.

type ResourcePlan struct {
	NetworkFunctions []PlannedNetworkFunction `json:"network_functions"`

	ResourceRequirements ResourceRequirements `json:"resource_requirements"`

	DeploymentPattern string `json:"deployment_pattern"`

	QoSProfile string `json:"qos_profile"`

	SliceConfiguration *SliceConfiguration `json:"slice_configuration,omitempty"`

	Interfaces []InterfaceConfiguration `json:"interfaces"`

	SecurityPolicies []SecurityPolicy `json:"security_policies"`

	EstimatedCost float64 `json:"estimated_cost"`
}

// PlannedNetworkFunction represents a planned network function deployment.

type PlannedNetworkFunction struct {
	Name string `json:"name"`

	Type string `json:"type"`

	Version string `json:"version"`

	Replicas int `json:"replicas"`

	Resources ResourceRequirements `json:"resources"`

	Configuration json.RawMessage `json:"configuration"`

	Dependencies []string `json:"dependencies"`

	Interfaces []string `json:"interfaces"`

	HealthChecks []HealthCheckSpec `json:"health_checks"`

	Monitoring MonitoringSpec `json:"monitoring"`
}

// ResourceRequirements represents compute resource requirements.

type ResourceRequirements struct {
	CPU string `json:"cpu"`

	Memory string `json:"memory"`

	Storage string `json:"storage"`

	NetworkBW string `json:"network_bandwidth"`

	GPU string `json:"gpu,omitempty"`

	Accelerator string `json:"accelerator,omitempty"`
}

// SliceConfiguration represents network slice configuration.

type SliceConfiguration struct {
	SliceType string `json:"slice_type"`

	SST int `json:"sst"`

	SD string `json:"sd,omitempty"`

	QoSProfile string `json:"qos_profile"`

	Isolation string `json:"isolation"`

	Parameters json.RawMessage `json:"parameters"`
}

// NetworkIntentReconciler orchestrates the reconciliation of NetworkIntent resources.

type NetworkIntentReconciler struct {
	client.Client

	Scheme *runtime.Scheme

	logger logr.Logger

	deps Dependencies

	config *Config

	constants *configPkg.Constants

	llmSanitizer *security.LLMSanitizer

	circuitBreakerMgr *resilience.CircuitBreakerManager

	timeoutManager *resilience.TimeoutManager

	llmCircuitBreaker *resilience.LLMCircuitBreaker

	reconciler *Reconciler

	metrics *ControllerMetrics
}

// Note: Exponential backoff helper functions are defined in e2nodeset_controller.go to avoid duplication.

func (r *NetworkIntentReconciler) setReadyCondition(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, status metav1.ConditionStatus, reason, message string) error {
	condition := metav1.Condition{
		Type: "Ready",

		Status: status,

		Reason: reason,

		Message: message,

		LastTransitionTime: metav1.Now(),
	}

	updateCondition(&networkIntent.Status.Conditions, condition)

	return r.safeStatusUpdate(ctx, networkIntent)
}

// NewNetworkIntentReconciler creates a new NetworkIntentReconciler with proper initialization.

func NewNetworkIntentReconciler(client client.Client, scheme *runtime.Scheme, deps Dependencies, config *Config) (*NetworkIntentReconciler, error) {
	// Note: Random number generator is automatically seeded in Go 1.20+.

	if client == nil {
		return nil, fmt.Errorf("client cannot be nil")
	}

	if scheme == nil {
		return nil, fmt.Errorf("scheme cannot be nil")
	}

	if deps == nil {
		return nil, fmt.Errorf("dependencies cannot be nil")
	}

	// Load constants if not provided in config.

	constants := config.Constants

	if constants == nil {

		constants = configPkg.LoadConstants()

		config.Constants = constants

	}

	// Validate configuration using constants.

	if err := configPkg.ValidateConstants(constants); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Validate and set defaults for config.

	validatedConfig, err := validateAndSetConfigDefaults(config, constants)
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Initialize LLM sanitizer with constants configuration.

	sanitizerConfig := &security.SanitizerConfig{
		MaxInputLength: constants.MaxInputLength,

		MaxOutputLength: constants.MaxOutputLength,

		AllowedDomains: constants.AllowedDomains,

		BlockedKeywords: constants.BlockedKeywords,

		ContextBoundary: constants.ContextBoundary,

		SystemPrompt: constants.SystemPrompt,
	}

	llmSanitizer := security.NewLLMSanitizer(sanitizerConfig)

	// Initialize resilience components.

	metricsCollector := deps.GetMetricsCollector()

	// Configure circuit breaker for LLM operations using constants.

	circuitConfig := &resilience.CircuitBreakerConfig{
		FailureThreshold: constants.CircuitBreakerFailureThreshold,

		RecoveryTimeout: constants.CircuitBreakerRecoveryTimeout,

		SuccessThreshold: constants.CircuitBreakerSuccessThreshold,

		RequestTimeout: constants.CircuitBreakerRequestTimeout,

		HalfOpenMaxRequests: constants.CircuitBreakerHalfOpenMaxRequests,

		MinimumRequests: constants.CircuitBreakerMinimumRequests,

		FailureRate: constants.CircuitBreakerFailureRate,
	}

	// Configure timeouts for different operations using constants.

	timeoutConfig := &resilience.TimeoutConfig{
		LLMTimeout: constants.LLMTimeout,

		GitTimeout: constants.GitTimeout,

		KubernetesTimeout: constants.KubernetesTimeout,

		PackageGenerationTimeout: constants.PackageGenerationTimeout,

		RAGTimeout: constants.RAGTimeout,

		ReconciliationTimeout: constants.ReconciliationTimeout,

		DefaultTimeout: constants.DefaultTimeout,
	}

	// Create circuit breaker manager.

	circuitBreakerMgr := resilience.NewCircuitBreakerManager(circuitConfig, metricsCollector)

	// Create timeout manager.

	timeoutManager := resilience.NewTimeoutManager(timeoutConfig, metricsCollector)

	// Create LLM circuit breaker.

	llmCircuitBreaker := circuitBreakerMgr.GetOrCreateCircuitBreaker("llm-processor", circuitConfig)

	r := &NetworkIntentReconciler{
		Client: client,

		Scheme: scheme,

		logger: ctrl.Log.WithName("networkintent-controller"),

		deps: deps,

		config: validatedConfig,

		constants: constants,

		llmSanitizer: llmSanitizer,

		circuitBreakerMgr: circuitBreakerMgr,

		timeoutManager: timeoutManager,

		llmCircuitBreaker: llmCircuitBreaker,

		metrics: NewControllerMetrics("networkintent"),
	}

	// Initialize the modular reconciler.

	r.reconciler = NewReconciler(r)

	return r, nil
}

// GetConfig returns the reconciler configuration for testing
func (r *NetworkIntentReconciler) GetConfig() *Config {
	return r.config
}

// GetDependencies returns the reconciler dependencies for testing
func (r *NetworkIntentReconciler) GetDependencies() Dependencies {
	return r.deps
}

// validateAndSetConfigDefaults validates and sets default values for the configuration.

func validateAndSetConfigDefaults(config *Config, constants *configPkg.Constants) (*Config, error) {
	if config == nil {
		config = &Config{}
	}

	// Set defaults from constants.

	if config.MaxRetries <= 0 {
		config.MaxRetries = constants.MaxRetries
	}

	if config.RetryDelay <= 0 {
		config.RetryDelay = constants.RetryDelay
	}

	if config.Timeout <= 0 {
		config.Timeout = constants.Timeout
	}

	if config.GitDeployPath == "" {
		config.GitDeployPath = constants.GitDeployPath
	}

	// Validate limits using constants.

	if config.MaxRetries > constants.MaxAllowedRetries {
		return nil, fmt.Errorf("MaxRetries (%d) exceeds maximum allowed (%d)", config.MaxRetries, constants.MaxAllowedRetries)
	}

	if config.RetryDelay > constants.MaxAllowedRetryDelay {
		return nil, fmt.Errorf("RetryDelay (%v) exceeds maximum allowed (%v)", config.RetryDelay, constants.MaxAllowedRetryDelay)
	}

	return config, nil
}

// Reconcile is the main entry point for reconciliation.

func (r *NetworkIntentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	startTime := time.Now()

	// Delegate to the modular reconciler.

	result, err := r.reconciler.Reconcile(ctx, req)

	// Record metrics.

	processingDuration := time.Since(startTime).Seconds()

	if err != nil {

		// Determine error type for better metrics granularity.

		errorType := "unknown"

		if strings.Contains(err.Error(), "not found") {
			errorType = "not_found"
		} else if strings.Contains(err.Error(), "timeout") {
			errorType = "timeout"
		} else if strings.Contains(err.Error(), "llm") {
			errorType = "llm_processing"
		} else if strings.Contains(err.Error(), "git") {
			errorType = "git_operations"
		}

		r.metrics.RecordFailure(req.Namespace, req.Name, errorType)

	} else {
		r.metrics.RecordSuccess(req.Namespace, req.Name)
	}

	// Record processing duration.

	r.metrics.RecordProcessingDuration(req.Namespace, req.Name, "total", processingDuration)

	return result, err
}

// extractIntentType extracts the intent type from the intent text.

func (r *NetworkIntentReconciler) extractIntentType(intent string) string {
	intentLower := strings.ToLower(intent)

	// Check for slice types.

	if strings.Contains(intentLower, "embb") || strings.Contains(intentLower, "broadband") {
		return "embb"
	} else if strings.Contains(intentLower, "urllc") || strings.Contains(intentLower, "low latency") {
		return "urllc"
	} else if strings.Contains(intentLower, "mmtc") || strings.Contains(intentLower, "machine type") || strings.Contains(intentLower, "iot") {
		return "mmtc"
	}

	// Check for network functions.

	if strings.Contains(intentLower, "amf") || strings.Contains(intentLower, "access") {
		return "5gc-control"
	} else if strings.Contains(intentLower, "upf") || strings.Contains(intentLower, "user plane") {
		return "5gc-user"
	} else if strings.Contains(intentLower, "gnb") || strings.Contains(intentLower, "base station") {
		return "ran"
	}

	return "generic"
}

// Helper functions for managing retry counts in annotations.

func parseQuantity(qty string) corev1.ResourceList {
	// Simple parser for comma-separated CPU,Memory values.

	_ = strings.Split(qty, ",")

	result := make(corev1.ResourceList)

	// This is a simplified implementation - in production you'd use resource.ParseQuantity.

	return result
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}

	return false
}

// Condition and status helper functions.

// isConditionTrue and updateCondition functions are already defined in controller_utils.go

// getNetworkIntentRetryCount gets the retry count for a specific operation from NetworkIntent annotations.

func getNetworkIntentRetryCount(networkIntent *nephoranv1.NetworkIntent, operation string) int {
	if networkIntent.Annotations == nil {
		return 0
	}

	key := fmt.Sprintf("nephoran.com/retry-count-%s", operation)

	if countStr, exists := networkIntent.Annotations[key]; exists {
		if count, err := strconv.Atoi(countStr); err == nil {
			return count
		}
	}

	return 0
}

// setNetworkIntentRetryCount sets the retry count for a specific operation in NetworkIntent annotations.

func setNetworkIntentRetryCount(networkIntent *nephoranv1.NetworkIntent, operation string, count int) {
	if networkIntent.Annotations == nil {
		networkIntent.Annotations = make(map[string]string)
	}

	key := fmt.Sprintf("nephoran.com/retry-count-%s", operation)

	networkIntent.Annotations[key] = strconv.Itoa(count)
}

// clearNetworkIntentRetryCount clears the retry count for a specific operation from NetworkIntent annotations.

func clearNetworkIntentRetryCount(networkIntent *nephoranv1.NetworkIntent, operation string) {
	if networkIntent.Annotations == nil {
		return
	}

	key := fmt.Sprintf("nephoran.com/retry-count-%s", operation)

	delete(networkIntent.Annotations, key)
}

// Deletion handling functions.

func (r *NetworkIntentReconciler) handleDeletion(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) (ctrl.Result, error) {
	return r.reconcileDelete(ctx, networkIntent)
}

func (r *NetworkIntentReconciler) reconcileDelete(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("operation", "delete")

	// Perform cleanup operations.

	if err := r.performCleanup(ctx, networkIntent); err != nil {

		logger.Error(err, "Failed to perform cleanup")

		return ctrl.Result{RequeueAfter: time.Minute}, err

	}

	// Remove finalizer.

	networkIntent.Finalizers = removeFinalizer(networkIntent.Finalizers, r.constants.NetworkIntentFinalizer)

	if err := r.safeUpdate(ctx, networkIntent); err != nil {

		logger.Error(err, "Failed to remove finalizer")

		return ctrl.Result{RequeueAfter: time.Second * 10}, err

	}

	logger.Info("NetworkIntent deletion completed successfully")

	return ctrl.Result{}, nil
}

func (r *NetworkIntentReconciler) performCleanup(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) error {
	// Use GitOps handler for cleanup.

	return r.reconciler.gitopsHandler.CleanupGitOpsResources(ctx, networkIntent)
}

// cleanupGitOpsPackages removes GitOps packages for the NetworkIntent.

func (r *NetworkIntentReconciler) cleanupGitOpsPackages(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, gitClient GitClientInterface) error {
	if gitClient == nil {
		return fmt.Errorf("git client is nil")
	}

	packagePath := fmt.Sprintf("networkintents/%s", networkIntent.Name)

	commitMsg := fmt.Sprintf("Remove NetworkIntent package: %s", networkIntent.Name)

	// Remove the package directory.

	if err := gitClient.RemoveDirectory(packagePath, commitMsg); err != nil {

		// If directory doesn't exist, that's fine - it might already be cleaned up.

		if !isNotFoundError(err) {
			return fmt.Errorf("failed to remove package directory: %w", err)
		}

		r.logger.Info("Package directory not found, skipping removal", "path", packagePath)

		return nil

	}

	// Commit and push changes.

	if err := gitClient.CommitAndPushChanges(commitMsg); err != nil {
		return fmt.Errorf("failed to commit and push changes: %w", err)
	}

	r.logger.Info("Successfully cleaned up GitOps packages", "intent", networkIntent.Name, "path", packagePath)

	return nil
}

// cleanupGeneratedResources removes generated Kubernetes resources for the NetworkIntent.

func (r *NetworkIntentReconciler) cleanupGeneratedResources(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) error {
	// List all resources with the NetworkIntent label.

	labelSelector := fmt.Sprintf("nephoran.io/network-intent=%s", networkIntent.Name)

	// Remove ConfigMaps.

	if err := r.cleanupResourcesByLabel(ctx, &corev1.ConfigMapList{}, labelSelector, networkIntent.Namespace); err != nil {
		return fmt.Errorf("failed to cleanup ConfigMaps: %w", err)
	}

	// Remove Secrets.

	if err := r.cleanupResourcesByLabel(ctx, &corev1.SecretList{}, labelSelector, networkIntent.Namespace); err != nil {
		return fmt.Errorf("failed to cleanup Secrets: %w", err)
	}

	// Remove Services.

	if err := r.cleanupResourcesByLabel(ctx, &corev1.ServiceList{}, labelSelector, networkIntent.Namespace); err != nil {
		return fmt.Errorf("failed to cleanup Services: %w", err)
	}

	r.logger.Info("Successfully cleaned up generated resources", "intent", networkIntent.Name)

	return nil
}

// cleanupResourcesByLabel removes resources of a specific type that match the label selector.

func (r *NetworkIntentReconciler) cleanupResourcesByLabel(ctx context.Context, list client.ObjectList, labelSelector, namespace string) error {
	opts := []client.ListOption{
		client.InNamespace(namespace),

		client.MatchingLabels(parseLabels(labelSelector)),
	}

	if err := r.List(ctx, list, opts...); err != nil {
		return fmt.Errorf("failed to list resources: %w", err)
	}

	// Extract items using reflection.

	items := reflect.ValueOf(list).Elem().FieldByName("Items")

	if !items.IsValid() {
		return fmt.Errorf("unable to extract items from list")
	}

	for i := range items.Len() {

		item := items.Index(i).Addr().Interface().(client.Object)

		if err := r.Delete(ctx, item); err != nil && !apierrors.IsNotFound(err) {
			r.logger.Error(err, "Failed to delete resource", "resource", item.GetName())

			// Continue with other resources instead of failing completely.
		} else {
			r.logger.Info("Deleted resource", "type", reflect.TypeOf(item), "name", item.GetName())
		}

	}

	return nil
}

// Helper function to check if error is a not found error.

func isNotFoundError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "not found")
}

// Helper function to parse label selector string.

func parseLabels(labelSelector string) map[string]string {
	labels := make(map[string]string)

	if labelSelector == "" {
		return labels
	}

	pairs := strings.Split(labelSelector, ",")

	for _, pair := range pairs {

		kv := strings.SplitN(pair, "=", 2)

		if len(kv) == 2 {
			labels[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}

	}

	return labels
}

// Finalizer helper functions.

func containsFinalizer(finalizers []string, finalizer string) bool {
	for _, f := range finalizers {
		if f == finalizer {
			return true
		}
	}

	return false
}

func removeFinalizer(finalizers []string, finalizer string) []string {
	var result []string

	for _, f := range finalizers {
		if f != finalizer {
			result = append(result, f)
		}
	}

	return result
}

// Safe client operation wrappers.

func (r *NetworkIntentReconciler) safeGet(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	return r.Get(ctx, key, obj)
}

func (r *NetworkIntentReconciler) safeUpdate(ctx context.Context, obj client.Object) error {
	return r.Update(ctx, obj)
}

func (r *NetworkIntentReconciler) safeStatusUpdate(ctx context.Context, obj client.Object) error {
	return r.Status().Update(ctx, obj)
}

func (r *NetworkIntentReconciler) updatePhase(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, phase string) error {
	networkIntent.Status.Phase = nephoranv1.NetworkIntentPhase(phase)

	networkIntent.Status.LastUpdated = metav1.Now()

	return r.safeStatusUpdate(ctx, networkIntent)
}

// Event recording functions.

func (r *NetworkIntentReconciler) recordEvent(networkIntent *nephoranv1.NetworkIntent, eventType, reason, message string) {
	if recorder := r.deps.GetEventRecorder(); recorder != nil {
		recorder.Event(networkIntent, eventType, reason, message)
	}
}

func (r *NetworkIntentReconciler) recordFailureEvent(networkIntent *nephoranv1.NetworkIntent, reason, message string) {
	r.recordEvent(networkIntent, "Warning", reason, message)
}

// SetupWithManager sets up the controller with the Manager.

func (r *NetworkIntentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nephoranv1.NetworkIntent{}).
		Complete(r)
}

// generateFallbackResponse generates a basic fallback response when LLM is unavailable.

func (r *NetworkIntentReconciler) generateFallbackResponse(intent string) string {
	logger := ctrl.Log.WithName("fallback-generator")

	logger.V(1).Info("Generating fallback response for intent", "intent_length", len(intent))

	// Simple pattern matching for basic intents.

	intentLower := strings.ToLower(intent)

	// Basic AMF deployment.

	if strings.Contains(intentLower, "amf") && (strings.Contains(intentLower, "deploy") || strings.Contains(intentLower, "create")) {
		return `{

			"network_functions": ["AMF"],

			"deployment_pattern": "basic",

			"resource_requirements": {

				"cpu": "2",

				"memory": "4Gi",

				"storage": "10Gi"

			},

			"qos_profile": "standard",

			"security_policies": ["default-security"]

		}`
	}

	// Basic SMF deployment.

	if strings.Contains(intentLower, "smf") && (strings.Contains(intentLower, "deploy") || strings.Contains(intentLower, "create")) {
		return `{

			"network_functions": ["SMF"],

			"deployment_pattern": "basic",

			"resource_requirements": {

				"cpu": "2",

				"memory": "4Gi",

				"storage": "10Gi"

			},

			"qos_profile": "standard",

			"security_policies": ["default-security"]

		}`
	}

	// Basic UPF deployment.

	if strings.Contains(intentLower, "upf") && (strings.Contains(intentLower, "deploy") || strings.Contains(intentLower, "create")) {
		return `{

			"network_functions": ["UPF"],

			"deployment_pattern": "basic",

			"resource_requirements": {

				"cpu": "4",

				"memory": "8Gi",

				"storage": "20Gi"

			},

			"qos_profile": "high-throughput",

			"security_policies": ["default-security"]

		}`
	}

	// No suitable fallback found.

	return ""
}

// cleanupResources performs comprehensive cleanup of all NetworkIntent resources.

func (r *NetworkIntentReconciler) cleanupResources(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) error {
	r.logger.Info("Starting comprehensive resource cleanup", "intent", networkIntent.Name)

	// Perform GitOps cleanup.

	gitClient := r.deps.GetGitClient()

	if err := r.cleanupGitOpsPackages(ctx, networkIntent, gitClient); err != nil {

		r.logger.Error(err, "Failed to cleanup GitOps packages", "intent", networkIntent.Name)

		return err

	}

	// Perform generated resources cleanup.

	if err := r.cleanupGeneratedResources(ctx, networkIntent); err != nil {

		r.logger.Error(err, "Failed to cleanup generated resources", "intent", networkIntent.Name)

		return err

	}

	// Perform cached data cleanup.

	if err := r.cleanupCachedData(ctx, networkIntent); err != nil {
		r.logger.Error(err, "Failed to cleanup cached data", "intent", networkIntent.Name)

		// Don't return error for cache cleanup failures as they're non-critical.
	}

	r.logger.Info("Successfully completed comprehensive resource cleanup", "intent", networkIntent.Name)

	return nil
}

// cleanupCachedData performs cleanup of cached data related to the NetworkIntent.

func (r *NetworkIntentReconciler) cleanupCachedData(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) error {
	if r.config.LLMProcessorURL == "" {

		r.logger.Info("LLM processor URL not configured, skipping cache cleanup", "intent", networkIntent.Name)

		return nil

	}

	// Create cache cleanup request.

	cacheKey := fmt.Sprintf("%s-%s", networkIntent.Namespace, networkIntent.Name)

	cleanupURL := fmt.Sprintf("%s/cache/cleanup/%s", r.config.LLMProcessorURL, cacheKey)

	// Create HTTP request with timeout.

	req, err := http.NewRequestWithContext(ctx, "DELETE", cleanupURL, http.NoBody)
	if err != nil {

		r.logger.Error(err, "Failed to create cache cleanup request", "intent", networkIntent.Name)

		return nil // Non-critical, don't propagate error

	}

	// Perform the request.

	resp, err := r.deps.GetHTTPClient().Do(req)
	if err != nil {

		r.logger.Info("Cache cleanup request failed (non-critical)", "intent", networkIntent.Name, "error", err)

		return nil // Non-critical, don't propagate error

	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		r.logger.Info("Cache cleanup returned error status (non-critical)",

			"intent", networkIntent.Name, "status", resp.Status)
	} else {
		r.logger.Info("Successfully cleaned up cached data", "intent", networkIntent.Name)
	}

	return nil
}

// createLabelSelector creates a label selector string from a map of labels.

func createLabelSelector(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	// Pre-allocate slice with known capacity for better performance
	pairs := make([]string, 0, len(labels))

	for key, value := range labels {
		pairs = append(pairs, fmt.Sprintf("%s=%s", key, value))
	}

	return strings.Join(pairs, ",")
}
