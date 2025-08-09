package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
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
	configPkg "github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/git"
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio"
	"github.com/thc1006/nephoran-intent-operator/pkg/resilience"
	"github.com/thc1006/nephoran-intent-operator/pkg/security"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
	"github.com/thc1006/nephoran-intent-operator/pkg/telecom"
)

// Dependencies interface defines the external dependencies for the controller
// This interface is implemented by the injection.Container
type Dependencies interface {
	GetGitClient() git.ClientInterface
	GetLLMClient() shared.ClientInterface
	GetPackageGenerator() *nephio.PackageGenerator
	GetHTTPClient() *http.Client
	GetEventRecorder() record.EventRecorder
	GetTelecomKnowledgeBase() *telecom.TelecomKnowledgeBase
	GetMetricsCollector() *monitoring.MetricsCollector
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
	
	// Configuration constants reference
	Constants *configPkg.Constants
}

// ProcessingPhase represents the current phase of intent processing
type ProcessingPhase string

const (
	PhaseLLMProcessing          ProcessingPhase = "LLMProcessing"
	PhaseResourcePlanning       ProcessingPhase = "ResourcePlanning"
	PhaseManifestGeneration     ProcessingPhase = "ManifestGeneration"
	PhaseGitOpsCommit           ProcessingPhase = "GitOpsCommit"
	PhaseDeploymentVerification ProcessingPhase = "DeploymentVerification"
)

// ProcessingContext holds context information for multi-phase processing
type ProcessingContext struct {
	StartTime         time.Time
	CurrentPhase      ProcessingPhase
	IntentType        string
	ExtractedEntities map[string]interface{}
	TelecomContext    map[string]interface{}
	ResourcePlan      *ResourcePlan
	Manifests         map[string]string
	GitCommitHash     string
	DeploymentStatus  map[string]interface{}
	Metrics           map[string]float64
}

// ResourcePlan represents the planned resources for deployment
type ResourcePlan struct {
	NetworkFunctions     []PlannedNetworkFunction `json:"network_functions"`
	ResourceRequirements ResourceRequirements     `json:"resource_requirements"`
	DeploymentPattern    string                   `json:"deployment_pattern"`
	QoSProfile           string                   `json:"qos_profile"`
	SliceConfiguration   *SliceConfiguration      `json:"slice_configuration,omitempty"`
	Interfaces           []InterfaceConfiguration `json:"interfaces"`
	SecurityPolicies     []SecurityPolicy         `json:"security_policies"`
	EstimatedCost        float64                  `json:"estimated_cost"`
}

// PlannedNetworkFunction represents a planned network function deployment
type PlannedNetworkFunction struct {
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	Version       string                 `json:"version"`
	Replicas      int                    `json:"replicas"`
	Resources     ResourceRequirements   `json:"resources"`
	Configuration map[string]interface{} `json:"configuration"`
	Dependencies  []string               `json:"dependencies"`
	Interfaces    []string               `json:"interfaces"`
	HealthChecks  []HealthCheckSpec      `json:"health_checks"`
	Monitoring    MonitoringSpec         `json:"monitoring"`
}

// ResourceRequirements represents compute resource requirements
type ResourceRequirements struct {
	CPU         string `json:"cpu"`
	Memory      string `json:"memory"`
	Storage     string `json:"storage"`
	NetworkBW   string `json:"network_bandwidth"`
	GPU         string `json:"gpu,omitempty"`
	Accelerator string `json:"accelerator,omitempty"`
}

// SliceConfiguration represents network slice configuration
type SliceConfiguration struct {
	SliceType  string                 `json:"slice_type"`
	SST        int                    `json:"sst"`
	SD         string                 `json:"sd,omitempty"`
	QoSProfile string                 `json:"qos_profile"`
	Isolation  string                 `json:"isolation"`
	Parameters map[string]interface{} `json:"parameters"`
}

// NetworkIntentReconciler orchestrates the reconciliation of NetworkIntent resources
type NetworkIntentReconciler struct {
	client.Client
	Scheme             *runtime.Scheme
	deps               Dependencies
	config             *Config
	constants          *configPkg.Constants
	llmSanitizer       *security.LLMSanitizer
	circuitBreakerMgr  *resilience.CircuitBreakerManager
	timeoutManager     *resilience.TimeoutManager
	llmCircuitBreaker  *resilience.LLMCircuitBreaker
	reconciler         *Reconciler
}

// Exponential backoff helper functions

// calculateExponentialBackoff calculates the exponential backoff delay with jitter
// retryCount: current retry attempt (0-based)
// baseDelay: base delay duration (uses configured default if zero)
// maxDelay: maximum delay duration (uses configured default if zero)
// constants: configuration constants for backoff parameters
func calculateExponentialBackoff(retryCount int, baseDelay, maxDelay time.Duration, constants *configPkg.Constants) time.Duration {
	if baseDelay <= 0 {
		baseDelay = constants.BaseBackoffDelay
	}
	if maxDelay <= 0 {
		maxDelay = constants.MaxBackoffDelay
	}

	// Calculate exponential backoff: baseDelay * (multiplier^retryCount)
	backoffDelay := float64(baseDelay) * math.Pow(constants.BackoffMultiplier, float64(retryCount))

	// Cap at maximum delay
	if backoffDelay > float64(maxDelay) {
		backoffDelay = float64(maxDelay)
	}

	// Add jitter to prevent thundering herd
	jitterRange := backoffDelay * constants.JitterFactor
	jitter := (rand.Float64() - 0.5) * 2 * jitterRange
	finalDelay := backoffDelay + jitter

	// Ensure minimum delay
	if finalDelay < float64(baseDelay) {
		finalDelay = float64(baseDelay)
	}

	return time.Duration(finalDelay)
}

// calculateExponentialBackoffForOperation calculates backoff for specific operations
func calculateExponentialBackoffForOperation(retryCount int, operation string, constants *configPkg.Constants) time.Duration {
	var baseDelay, maxDelay time.Duration

	switch operation {
	case "llm-processing":
		baseDelay = constants.LLMProcessingBaseDelay
		maxDelay = constants.LLMProcessingMaxDelay
	case "git-operations":
		baseDelay = constants.GitOperationsBaseDelay
		maxDelay = constants.GitOperationsMaxDelay
	case "resource-planning":
		baseDelay = constants.ResourcePlanningBaseDelay
		maxDelay = constants.ResourcePlanningMaxDelay
	default:
		baseDelay = constants.BaseBackoffDelay
		maxDelay = constants.MaxBackoffDelay
	}

	return calculateExponentialBackoff(retryCount, baseDelay, maxDelay, constants)
}

func (r *NetworkIntentReconciler) setReadyCondition(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, status metav1.ConditionStatus, reason, message string) error {
	condition := metav1.Condition{
		Type:               "Ready",
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}
	updateCondition(&networkIntent.Status.Conditions, condition)
	return r.safeStatusUpdate(ctx, networkIntent)
}

// NewNetworkIntentReconciler creates a new NetworkIntentReconciler with proper initialization
func NewNetworkIntentReconciler(client client.Client, scheme *runtime.Scheme, deps Dependencies, config *Config) (*NetworkIntentReconciler, error) {
	// Note: Random number generator is automatically seeded in Go 1.20+

	if client == nil {
		return nil, fmt.Errorf("client cannot be nil")
	}
	if scheme == nil {
		return nil, fmt.Errorf("scheme cannot be nil")
	}
	if deps == nil {
		return nil, fmt.Errorf("dependencies cannot be nil")
	}

	// Load constants if not provided in config
	constants := config.Constants
	if constants == nil {
		constants = configPkg.LoadConstants()
		config.Constants = constants
	}

	// Validate configuration using constants
	if err := configPkg.ValidateConstants(constants); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Validate and set defaults for config
	validatedConfig, err := validateAndSetConfigDefaults(config, constants)
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Initialize LLM sanitizer with constants configuration
	sanitizerConfig := &security.SanitizerConfig{
		MaxInputLength:  constants.MaxInputLength,
		MaxOutputLength: constants.MaxOutputLength,
		AllowedDomains:  constants.AllowedDomains,
		BlockedKeywords: constants.BlockedKeywords,
		ContextBoundary: constants.ContextBoundary,
		SystemPrompt:    constants.SystemPrompt,
	}
	llmSanitizer := security.NewLLMSanitizer(sanitizerConfig)

	// Initialize resilience components
	metricsCollector := deps.GetMetricsCollector()

	// Configure circuit breaker for LLM operations using constants
	circuitConfig := &resilience.CircuitBreakerConfig{
		FailureThreshold:    constants.CircuitBreakerFailureThreshold,
		RecoveryTimeout:     constants.CircuitBreakerRecoveryTimeout,
		SuccessThreshold:    constants.CircuitBreakerSuccessThreshold,
		RequestTimeout:      constants.CircuitBreakerRequestTimeout,
		HalfOpenMaxRequests: constants.CircuitBreakerHalfOpenMaxRequests,
		MinimumRequests:     constants.CircuitBreakerMinimumRequests,
		FailureRate:         constants.CircuitBreakerFailureRate,
	}

	// Configure timeouts for different operations using constants
	timeoutConfig := &resilience.TimeoutConfig{
		LLMTimeout:               constants.LLMTimeout,
		GitTimeout:               constants.GitTimeout,
		KubernetesTimeout:        constants.KubernetesTimeout,
		PackageGenerationTimeout: constants.PackageGenerationTimeout,
		RAGTimeout:               constants.RAGTimeout,
		ReconciliationTimeout:    constants.ReconciliationTimeout,
		DefaultTimeout:           constants.DefaultTimeout,
	}

	// Create circuit breaker manager
	circuitBreakerMgr := resilience.NewCircuitBreakerManager(circuitConfig, metricsCollector)

	// Create timeout manager
	timeoutManager := resilience.NewTimeoutManager(timeoutConfig, metricsCollector)

	// Create LLM circuit breaker
	llmCircuitBreaker := circuitBreakerMgr.GetOrCreateCircuitBreaker("llm-processor", circuitConfig)

	r := &NetworkIntentReconciler{
		Client:            client,
		Scheme:            scheme,
		deps:              deps,
		config:            validatedConfig,
		constants:         constants,
		llmSanitizer:      llmSanitizer,
		circuitBreakerMgr: circuitBreakerMgr,
		timeoutManager:    timeoutManager,
		llmCircuitBreaker: llmCircuitBreaker,
	}

	// Initialize the modular reconciler
	r.reconciler = NewReconciler(r)

	return r, nil
}

// validateAndSetConfigDefaults validates and sets default values for the configuration
func validateAndSetConfigDefaults(config *Config, constants *configPkg.Constants) (*Config, error) {
	if config == nil {
		config = &Config{}
	}

	// Set defaults from constants
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

	// Validate limits using constants
	if config.MaxRetries > constants.MaxAllowedRetries {
		return nil, fmt.Errorf("MaxRetries (%d) exceeds maximum allowed (%d)", config.MaxRetries, constants.MaxAllowedRetries)
	}
	if config.RetryDelay > constants.MaxAllowedRetryDelay {
		return nil, fmt.Errorf("RetryDelay (%v) exceeds maximum allowed (%v)", config.RetryDelay, constants.MaxAllowedRetryDelay)
	}

	return config, nil
}

// Reconcile is the main entry point for reconciliation
func (r *NetworkIntentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Delegate to the modular reconciler
	return r.reconciler.Reconcile(ctx, req)
}

// extractIntentType extracts the intent type from the intent text
func (r *NetworkIntentReconciler) extractIntentType(intent string) string {
	intentLower := strings.ToLower(intent)

	// Check for slice types
	if strings.Contains(intentLower, "embb") || strings.Contains(intentLower, "broadband") {
		return "embb"
	} else if strings.Contains(intentLower, "urllc") || strings.Contains(intentLower, "low latency") {
		return "urllc"
	} else if strings.Contains(intentLower, "mmtc") || strings.Contains(intentLower, "machine type") || strings.Contains(intentLower, "iot") {
		return "mmtc"
	}

	// Check for network functions
	if strings.Contains(intentLower, "amf") || strings.Contains(intentLower, "access") {
		return "5gc-control"
	} else if strings.Contains(intentLower, "upf") || strings.Contains(intentLower, "user plane") {
		return "5gc-user"
	} else if strings.Contains(intentLower, "gnb") || strings.Contains(intentLower, "base station") {
		return "ran"
	}

	return "generic"
}

// Helper functions for managing retry counts in annotations
func parseQuantity(qty string) corev1.ResourceList {
	// Simple parser for comma-separated CPU,Memory values
	parts := strings.Split(qty, ",")
	result := make(corev1.ResourceList)
	// This is a simplified implementation - in production you'd use resource.ParseQuantity
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

// Condition and status helper functions
func isConditionTrue(conditions []metav1.Condition, conditionType string) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType && condition.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}

func updateCondition(conditions *[]metav1.Condition, newCondition metav1.Condition) {
	if conditions == nil {
		*conditions = []metav1.Condition{newCondition}
		return
	}

	for i, condition := range *conditions {
		if condition.Type == newCondition.Type {
			(*conditions)[i] = newCondition
			return
		}
	}
	*conditions = append(*conditions, newCondition)
}

// Retry count management functions
func getRetryCount(networkIntent *nephoranv1.NetworkIntent, operation string) int {
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

func setRetryCount(networkIntent *nephoranv1.NetworkIntent, operation string, count int) {
	if networkIntent.Annotations == nil {
		networkIntent.Annotations = make(map[string]string)
	}
	key := fmt.Sprintf("nephoran.com/retry-count-%s", operation)
	networkIntent.Annotations[key] = strconv.Itoa(count)
}

func clearRetryCount(networkIntent *nephoranv1.NetworkIntent, operation string) {
	if networkIntent.Annotations == nil {
		return
	}
	key := fmt.Sprintf("nephoran.com/retry-count-%s", operation)
	delete(networkIntent.Annotations, key)
}

// Deletion handling functions
func (r *NetworkIntentReconciler) handleDeletion(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) (ctrl.Result, error) {
	return r.reconcileDelete(ctx, networkIntent)
}

func (r *NetworkIntentReconciler) reconcileDelete(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("operation", "delete")
	
	// Perform cleanup operations
	if err := r.performCleanup(ctx, networkIntent); err != nil {
		logger.Error(err, "Failed to perform cleanup")
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	// Remove finalizer
	networkIntent.Finalizers = removeFinalizer(networkIntent.Finalizers, r.constants.NetworkIntentFinalizer)
	if err := r.safeUpdate(ctx, networkIntent); err != nil {
		logger.Error(err, "Failed to remove finalizer")
		return ctrl.Result{RequeueAfter: time.Second * 10}, err
	}

	logger.Info("NetworkIntent deletion completed successfully")
	return ctrl.Result{}, nil
}

func (r *NetworkIntentReconciler) performCleanup(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) error {
	// Use GitOps handler for cleanup
	return r.reconciler.gitopsHandler.CleanupGitOpsResources(ctx, networkIntent)
}

// Finalizer helper functions
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

// Safe client operation wrappers
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
	networkIntent.Status.Phase = phase
	networkIntent.Status.LastUpdated = metav1.Now()
	return r.safeStatusUpdate(ctx, networkIntent)
}

// Event recording functions
func (r *NetworkIntentReconciler) recordEvent(networkIntent *nephoranv1.NetworkIntent, eventType, reason, message string) {
	if recorder := r.deps.GetEventRecorder(); recorder != nil {
		recorder.Event(networkIntent, eventType, reason, message)
	}
}

func (r *NetworkIntentReconciler) recordFailureEvent(networkIntent *nephoranv1.NetworkIntent, reason, message string) {
	r.recordEvent(networkIntent, "Warning", reason, message)
}

// SetupWithManager sets up the controller with the Manager
func (r *NetworkIntentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nephoranv1.NetworkIntent{}).
		Complete(r)
}

// generateFallbackResponse generates a basic fallback response when LLM is unavailable
func (r *NetworkIntentReconciler) generateFallbackResponse(intent string) string {
	logger := ctrl.Log.WithName("fallback-generator")
	logger.V(1).Info("Generating fallback response for intent", "intent_length", len(intent))

	// Simple pattern matching for basic intents
	intentLower := strings.ToLower(intent)

	// Basic AMF deployment
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

	// Basic SMF deployment
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

	// Basic UPF deployment
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

	// No suitable fallback found
	return ""
}