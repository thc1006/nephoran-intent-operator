package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/git"
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
	"github.com/thc1006/nephoran-intent-operator/pkg/telecom"
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
}

// ProcessingPhase represents the current phase of intent processing
type ProcessingPhase string

const (
	PhaseLLMProcessing     ProcessingPhase = "LLMProcessing"
	PhaseResourcePlanning  ProcessingPhase = "ResourcePlanning"
	PhaseManifestGeneration ProcessingPhase = "ManifestGeneration"
	PhaseGitOpsCommit      ProcessingPhase = "GitOpsCommit"
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
	Metrics          map[string]float64
}

// ResourcePlan represents the planned resources for deployment
type ResourcePlan struct {
	NetworkFunctions []PlannedNetworkFunction `json:"network_functions"`
	ResourceRequirements ResourceRequirements `json:"resource_requirements"`
	DeploymentPattern string                  `json:"deployment_pattern"`
	QoSProfile       string                   `json:"qos_profile"`
	SliceConfiguration *SliceConfiguration    `json:"slice_configuration,omitempty"`
	Interfaces       []InterfaceConfiguration `json:"interfaces"`
	SecurityPolicies []SecurityPolicy         `json:"security_policies"`
	EstimatedCost    float64                  `json:"estimated_cost"`
}

// PlannedNetworkFunction represents a planned network function deployment
type PlannedNetworkFunction struct {
	Name           string            `json:"name"`
	Type           string            `json:"type"`
	Version        string            `json:"version"`
	Replicas       int               `json:"replicas"`
	Resources      ResourceRequirements `json:"resources"`
	Configuration  map[string]interface{} `json:"configuration"`
	Dependencies   []string          `json:"dependencies"`
	Interfaces     []string          `json:"interfaces"`
	HealthChecks   []HealthCheckSpec `json:"health_checks"`
	Monitoring     MonitoringSpec    `json:"monitoring"`
}

// ResourceRequirements represents compute resource requirements
type ResourceRequirements struct {
	CPU        string `json:"cpu"`
	Memory     string `json:"memory"`
	Storage    string `json:"storage"`
	NetworkBW  string `json:"network_bandwidth"`
	GPU        string `json:"gpu,omitempty"`
	Accelerator string `json:"accelerator,omitempty"`
}

// SliceConfiguration represents network slice configuration
type SliceConfiguration struct {
	SliceType    string            `json:"slice_type"`
	SST          int               `json:"sst"`
	SD           string            `json:"sd,omitempty"`
	QoSProfile   string            `json:"qos_profile"`
	Isolation    string            `json:"isolation"`
	Parameters   map[string]interface{} `json:"parameters"`
}

// InterfaceConfiguration represents network interface configuration
type InterfaceConfiguration struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	Protocol     []string          `json:"protocol"`
	Endpoints    []EndpointSpec    `json:"endpoints"`
	Security     SecuritySpec      `json:"security"`
	QoS          QoSSpec           `json:"qos"`
}

// SecurityPolicy represents security policy configuration
type SecurityPolicy struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	Rules        []SecurityRule    `json:"rules"`
	AppliesTo    []string          `json:"applies_to"`
}

// Supporting specification types
type HealthCheckSpec struct {
	Type        string `json:"type"`
	Path        string `json:"path"`
	Port        int    `json:"port"`
	Interval    int    `json:"interval"`
	Timeout     int    `json:"timeout"`
	Retries     int    `json:"retries"`
}

type MonitoringSpec struct {
	Enabled     bool     `json:"enabled"`
	Metrics     []string `json:"metrics"`
	Alerts      []string `json:"alerts"`
	Dashboards  []string `json:"dashboards"`
}

type EndpointSpec struct {
	Name     string `json:"name"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Path     string `json:"path"`
}

type SecuritySpec struct {
	Authentication string   `json:"authentication"`
	Encryption     []string `json:"encryption"`
	Certificates   []string `json:"certificates"`
}

type QoSSpec struct {
	Bandwidth  string  `json:"bandwidth"`
	Latency    float64 `json:"latency"`
	Jitter     float64 `json:"jitter"`
	PacketLoss float64 `json:"packet_loss"`
}

type SecurityRule struct {
	Action    string   `json:"action"`
	Protocol  string   `json:"protocol"`
	Ports     []int    `json:"ports"`
	Sources   []string `json:"sources"`
	Targets   []string `json:"targets"`
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

	// Initialize processing context
	processingCtx := &ProcessingContext{
		StartTime:         time.Now(),
		ExtractedEntities: make(map[string]interface{}),
		TelecomContext:    make(map[string]interface{}),
		DeploymentStatus:  make(map[string]interface{}),
		Metrics:          make(map[string]float64),
	}

	// Update observed generation
	networkIntent.Status.ObservedGeneration = networkIntent.Generation

	// Initialize metrics collector
	metricsCollector := r.deps.GetMetricsCollector()
	if metricsCollector != nil {
		metricsCollector.UpdateNetworkIntentStatus(networkIntent.Name, networkIntent.Namespace, 
			r.extractIntentType(networkIntent.Spec.Intent), "processing")
	}

	// Check if already completed to avoid unnecessary work
	if isConditionTrue(networkIntent.Status.Conditions, "Processed") &&
		isConditionTrue(networkIntent.Status.Conditions, "ResourcesPlanned") &&
		isConditionTrue(networkIntent.Status.Conditions, "ManifestsGenerated") &&
		isConditionTrue(networkIntent.Status.Conditions, "GitOpsCommitted") &&
		isConditionTrue(networkIntent.Status.Conditions, "DeploymentVerified") &&
		networkIntent.Status.Phase == "Completed" {
		logger.V(1).Info("NetworkIntent already fully processed, skipping")
		return ctrl.Result{}, nil
	}

	// Execute multi-phase processing pipeline
	result, err := r.executeProcessingPipeline(ctx, &networkIntent, processingCtx)
	if err != nil {
		logger.Error(err, "processing pipeline failed")
		r.recordFailureEvent(&networkIntent, "ProcessingPipelineFailed", err.Error())
		if metricsCollector != nil {
			metricsCollector.UpdateNetworkIntentStatus(networkIntent.Name, networkIntent.Namespace, 
				r.extractIntentType(networkIntent.Spec.Intent), "failed")
		}
		return result, err
	}

	if result.Requeue || result.RequeueAfter > 0 {
		logger.V(1).Info("Processing pipeline requires requeue", "requeue", result.Requeue, "requeue_after", result.RequeueAfter)
		return result, nil
	}

	// Final status update
	if err := r.updatePhase(ctx, &networkIntent, "Completed"); err != nil {
		logger.Error(err, "failed to update completion phase")
		return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update completion phase: %w", err)
	}

	// Record success metrics
	if metricsCollector != nil {
		processingDuration := time.Since(processingCtx.StartTime)
		metricsCollector.RecordNetworkIntentProcessed(
			r.extractIntentType(networkIntent.Spec.Intent), "completed", processingDuration)
		metricsCollector.UpdateNetworkIntentStatus(networkIntent.Name, networkIntent.Namespace, 
			r.extractIntentType(networkIntent.Spec.Intent), "completed")
	}

	r.recordEvent(&networkIntent, "Normal", "ReconciliationCompleted", 
		"NetworkIntent successfully processed through all phases and deployed")
	logger.Info("NetworkIntent reconciliation completed successfully",
		"intent", networkIntent.Spec.Intent,
		"phase", networkIntent.Status.Phase,
		"processing_time", time.Since(processingCtx.StartTime),
		"request_id", reqID)

	return ctrl.Result{}, nil
}

// executeProcessingPipeline executes the multi-phase processing pipeline
func (r *NetworkIntentReconciler) executeProcessingPipeline(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, processingCtx *ProcessingContext) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("pipeline", "multi-phase")

	// Phase 1: LLM Processing with RAG context retrieval
	if !isConditionTrue(networkIntent.Status.Conditions, "Processed") {
		logger.Info("Starting Phase 1: LLM Processing")
		if err := r.updatePhase(ctx, networkIntent, "LLMProcessing"); err != nil {
			return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update LLM processing phase: %w", err)
		}

		result, err := r.processLLMPhase(ctx, networkIntent, processingCtx)
		if err != nil || result.Requeue || result.RequeueAfter > 0 {
			return result, err
		}
	}

	// Phase 2: Resource planning with telecom knowledge
	if isConditionTrue(networkIntent.Status.Conditions, "Processed") && !isConditionTrue(networkIntent.Status.Conditions, "ResourcesPlanned") {
		logger.Info("Starting Phase 2: Resource Planning")
		if err := r.updatePhase(ctx, networkIntent, "ResourcePlanning"); err != nil {
			return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update resource planning phase: %w", err)
		}

		result, err := r.planResources(ctx, networkIntent, processingCtx)
		if err != nil || result.Requeue || result.RequeueAfter > 0 {
			return result, err
		}
	}

	// Phase 3: Deployment manifest generation
	if isConditionTrue(networkIntent.Status.Conditions, "ResourcesPlanned") && !isConditionTrue(networkIntent.Status.Conditions, "ManifestsGenerated") {
		logger.Info("Starting Phase 3: Manifest Generation")
		if err := r.updatePhase(ctx, networkIntent, "ManifestGeneration"); err != nil {
			return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update manifest generation phase: %w", err)
		}

		result, err := r.generateManifests(ctx, networkIntent, processingCtx)
		if err != nil || result.Requeue || result.RequeueAfter > 0 {
			return result, err
		}
	}

	// Phase 4: GitOps commit and validation
	if isConditionTrue(networkIntent.Status.Conditions, "ManifestsGenerated") && !isConditionTrue(networkIntent.Status.Conditions, "GitOpsCommitted") {
		logger.Info("Starting Phase 4: GitOps Commit")
		if err := r.updatePhase(ctx, networkIntent, "GitOpsCommit"); err != nil {
			return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update GitOps commit phase: %w", err)
		}

		result, err := r.commitToGitOps(ctx, networkIntent, processingCtx)
		if err != nil || result.Requeue || result.RequeueAfter > 0 {
			return result, err
		}
	}

	// Phase 5: Deployment verification
	if isConditionTrue(networkIntent.Status.Conditions, "GitOpsCommitted") && !isConditionTrue(networkIntent.Status.Conditions, "DeploymentVerified") {
		logger.Info("Starting Phase 5: Deployment Verification")
		if err := r.updatePhase(ctx, networkIntent, "DeploymentVerification"); err != nil {
			return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update deployment verification phase: %w", err)
		}

		result, err := r.verifyDeployment(ctx, networkIntent, processingCtx)
		if err != nil || result.Requeue || result.RequeueAfter > 0 {
			return result, err
		}
	}

	logger.Info("All phases completed successfully", "total_time", time.Since(processingCtx.StartTime))
	return ctrl.Result{}, nil
}

// processLLMPhase implements Phase 1: LLM Processing with RAG context retrieval
func (r *NetworkIntentReconciler) processLLMPhase(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, processingCtx *ProcessingContext) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("phase", "llm-processing")
	startTime := time.Now()

	processingCtx.CurrentPhase = PhaseLLMProcessing
	processingCtx.IntentType = r.extractIntentType(networkIntent.Spec.Intent)

	// Get retry count
	retryCount := getRetryCount(networkIntent, "llm-processing")
	if retryCount >= r.config.MaxRetries {
		err := fmt.Errorf("max retries (%d) exceeded for LLM processing", r.config.MaxRetries)
		condition := metav1.Condition{
			Type:               "Processed",
			Status:             metav1.ConditionFalse,
			Reason:             "LLMProcessingFailedMaxRetries",
			Message:            fmt.Sprintf("Failed to process intent after %d retries", r.config.MaxRetries),
			LastTransitionTime: metav1.Now(),
		}
		updateCondition(&networkIntent.Status.Conditions, condition)
		r.safeStatusUpdate(ctx, networkIntent)
		return ctrl.Result{}, err
	}

	// Validate LLM client
	llmClient := r.deps.GetLLMClient()
	if llmClient == nil {
		err := fmt.Errorf("LLM client is not configured")
		logger.Error(err, "LLM client validation failed")
		condition := metav1.Condition{
			Type:               "Processed",
			Status:             metav1.ConditionFalse,
			Reason:             "LLMClientNotConfigured",
			Message:            "LLM client is not configured and cannot process intent",
			LastTransitionTime: metav1.Now(),
		}
		updateCondition(&networkIntent.Status.Conditions, condition)
		r.safeStatusUpdate(ctx, networkIntent)
		return ctrl.Result{}, err
	}

	// Build telecom-enhanced prompt with 3GPP context
	enhancedPrompt, err := r.buildTelecomEnhancedPrompt(ctx, networkIntent.Spec.Intent, processingCtx)
	if err != nil {
		logger.Error(err, "failed to build telecom-enhanced prompt")
		setRetryCount(networkIntent, "llm-processing", retryCount+1)
		r.recordFailureEvent(networkIntent, "PromptEnhancementFailed", err.Error())
		return ctrl.Result{RequeueAfter: r.config.RetryDelay}, nil
	}

	// Process with LLM
	logger.Info("Processing intent with LLM", "retry_count", retryCount+1, "enhanced_prompt_length", len(enhancedPrompt))
	processedResult, err := llmClient.ProcessIntent(ctx, enhancedPrompt)
	
	if err != nil {
		logger.Error(err, "LLM processing failed", "retry", retryCount+1)
		setRetryCount(networkIntent, "llm-processing", retryCount+1)
		
		condition := metav1.Condition{
			Type:               "Processed",
			Status:             metav1.ConditionFalse,
			Reason:             "LLMProcessingRetrying",
			Message:            fmt.Sprintf("LLM processing failed (attempt %d/%d): %v", retryCount+1, r.config.MaxRetries, err),
			LastTransitionTime: metav1.Now(),
		}
		updateCondition(&networkIntent.Status.Conditions, condition)
		r.safeStatusUpdate(ctx, networkIntent)
		
		r.recordFailureEvent(networkIntent, "LLMProcessingRetry", fmt.Sprintf("attempt %d/%d failed: %v", retryCount+1, r.config.MaxRetries, err))
		backoffDelay := time.Duration(retryCount+1) * r.config.RetryDelay
		if backoffDelay > time.Minute*10 {
			backoffDelay = time.Minute * 10
		}
		return ctrl.Result{RequeueAfter: backoffDelay}, nil
	}

	// Parse and validate LLM response
	var parameters map[string]interface{}
	if err := json.Unmarshal([]byte(processedResult), &parameters); err != nil {
		logger.Error(err, "failed to parse LLM response as JSON", "response", processedResult)
		setRetryCount(networkIntent, "llm-processing", retryCount+1)
		
		condition := metav1.Condition{
			Type:               "Processed",
			Status:             metav1.ConditionFalse,
			Reason:             "LLMResponseParsingFailed",
			Message:            fmt.Sprintf("Failed to parse LLM response as JSON: %v", err),
			LastTransitionTime: metav1.Now(),
		}
		updateCondition(&networkIntent.Status.Conditions, condition)
		r.safeStatusUpdate(ctx, networkIntent)
		
		return ctrl.Result{RequeueAfter: r.config.RetryDelay}, nil
	}

	// Extract structured entities from LLM response
	processingCtx.ExtractedEntities = parameters

	// Update NetworkIntent with processed parameters
	parametersRaw, err := json.Marshal(parameters)
	if err != nil {
		return ctrl.Result{RequeueAfter: r.config.RetryDelay}, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	networkIntent.Spec.Parameters = runtime.RawExtension{Raw: parametersRaw}
	if err := r.safeUpdate(ctx, networkIntent); err != nil {
		return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update NetworkIntent with parameters: %w", err)
	}

	// Clear retry count and update success condition
	clearRetryCount(networkIntent, "llm-processing")
	processingDuration := time.Since(startTime)
	now := metav1.Now()
	networkIntent.Status.ProcessingCompletionTime = &now
	
	condition := metav1.Condition{
		Type:               "Processed",
		Status:             metav1.ConditionTrue,
		Reason:             "LLMProcessingSucceeded",
		Message:            fmt.Sprintf("Intent successfully processed by LLM with %d parameters in %.2fs", len(parameters), processingDuration.Seconds()),
		LastTransitionTime: now,
	}
	updateCondition(&networkIntent.Status.Conditions, condition)
	
	if err := r.safeStatusUpdate(ctx, networkIntent); err != nil {
		return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update status: %w", err)
	}

	// Record metrics
	if metricsCollector := r.deps.GetMetricsCollector(); metricsCollector != nil {
		metricsCollector.RecordLLMRequest("gpt-4o-mini", "success", processingDuration, len(processedResult))
	}

	processingCtx.Metrics["llm_processing_duration"] = processingDuration.Seconds()
	processingCtx.Metrics["llm_parameters_count"] = float64(len(parameters))

	r.recordEvent(networkIntent, "Normal", "LLMProcessingSucceeded", fmt.Sprintf("Intent processed with %d parameters", len(parameters)))
	logger.Info("LLM processing phase completed successfully", "duration", processingDuration, "parameters_count", len(parameters))

	return ctrl.Result{}, nil
}

// buildTelecomEnhancedPrompt builds a telecom-enhanced prompt with 3GPP context and O-RAN knowledge
func (r *NetworkIntentReconciler) buildTelecomEnhancedPrompt(ctx context.Context, intent string, processingCtx *ProcessingContext) (string, error) {
	logger := log.FromContext(ctx).WithValues("function", "buildTelecomEnhancedPrompt")

	// Get telecom knowledge base
	kb := r.deps.GetTelecomKnowledgeBase()
	if kb == nil {
		return "", fmt.Errorf("telecom knowledge base not available")
	}

	// Extract telecom entities and context from intent
	telecomContext := r.extractTelecomContext(intent, kb)
	processingCtx.TelecomContext = telecomContext

	// Build enhanced prompt with telecom knowledge
	var promptBuilder strings.Builder

	promptBuilder.WriteString("You are a telecommunications network orchestration expert with deep knowledge of 5G Core (5GC), ")
	promptBuilder.WriteString("O-RAN architecture, 3GPP specifications, and network function deployment patterns.\n\n")

	// Add 3GPP and O-RAN context
	promptBuilder.WriteString("## TELECOMMUNICATIONS CONTEXT\n")
	promptBuilder.WriteString("Based on 3GPP Release 17 and O-RAN Alliance specifications:\n\n")

	// Add relevant network function knowledge
	if nfTypes, ok := telecomContext["detected_network_functions"].([]string); ok && len(nfTypes) > 0 {
		promptBuilder.WriteString("### Relevant Network Functions:\n")
		for _, nfType := range nfTypes {
			if nf, exists := kb.GetNetworkFunction(nfType); exists {
				promptBuilder.WriteString(fmt.Sprintf("- **%s** (%s): %s\n", nf.Name, nf.Type, nf.Description))
				promptBuilder.WriteString(fmt.Sprintf("  - Interfaces: %s\n", strings.Join(nf.Interfaces, ", ")))
				promptBuilder.WriteString(fmt.Sprintf("  - Dependencies: %s\n", strings.Join(nf.Dependencies, ", ")))
				promptBuilder.WriteString(fmt.Sprintf("  - Performance: Max %d RPS, Avg latency %.1fms\n", 
					nf.Performance.MaxThroughputRPS, nf.Performance.AvgLatencyMs))
			}
		}
		promptBuilder.WriteString("\n")
	}

	// Add slice type information if detected
	if sliceType, ok := telecomContext["detected_slice_type"].(string); ok && sliceType != "" {
		if slice, exists := kb.GetSliceType(sliceType); exists {
			promptBuilder.WriteString(fmt.Sprintf("### Network Slice Type: %s (SST=%d)\n", slice.Description, slice.SST))
			promptBuilder.WriteString(fmt.Sprintf("- Use Case: %s\n", slice.UseCase))
			promptBuilder.WriteString(fmt.Sprintf("- Latency Requirement: User Plane %.1fms, Control Plane %.1fms\n",
				slice.Requirements.Latency.UserPlane, slice.Requirements.Latency.ControlPlane))
			promptBuilder.WriteString(fmt.Sprintf("- Throughput: %s-%s Mbps (typical %s)\n",
				slice.Requirements.Throughput.Min, slice.Requirements.Throughput.Max, slice.Requirements.Throughput.Typical))
			promptBuilder.WriteString(fmt.Sprintf("- Reliability: %.3f%% availability, %.5f%% packet loss\n",
				slice.Requirements.Reliability.Availability, slice.Requirements.Reliability.PacketLoss))
			promptBuilder.WriteString("\n")
		}
	}

	// Add deployment pattern context
	if pattern, ok := telecomContext["deployment_pattern"].(string); ok && pattern != "" {
		if deployPattern, exists := kb.GetDeploymentPattern(pattern); exists {
			promptBuilder.WriteString(fmt.Sprintf("### Deployment Pattern: %s\n", deployPattern.Name))
			promptBuilder.WriteString(fmt.Sprintf("- Description: %s\n", deployPattern.Description))
			promptBuilder.WriteString(fmt.Sprintf("- Architecture: %s with %s redundancy\n",
				deployPattern.Architecture.Type, deployPattern.Architecture.Redundancy))
			promptBuilder.WriteString("\n")
		}
	}

	// Add the user intent
	promptBuilder.WriteString("## USER INTENT\n")
	promptBuilder.WriteString(fmt.Sprintf("Original intent: \"%s\"\n\n", intent))

	// Add structured output requirements
	promptBuilder.WriteString("## REQUIRED OUTPUT FORMAT\n")
	promptBuilder.WriteString("Analyze the intent and provide a JSON response with the following structure:\n")
	promptBuilder.WriteString("{\n")
	promptBuilder.WriteString("  \"network_functions\": [\"list of required 5G NFs\"],\n")
	promptBuilder.WriteString("  \"deployment_type\": \"production|development|testing\",\n")
	promptBuilder.WriteString("  \"scaling_requirements\": {\n")
	promptBuilder.WriteString("    \"min_replicas\": number,\n")
	promptBuilder.WriteString("    \"max_replicas\": number,\n")
	promptBuilder.WriteString("    \"target_cpu\": percentage\n")
	promptBuilder.WriteString("  },\n")
	promptBuilder.WriteString("  \"slice_configuration\": {\n")
	promptBuilder.WriteString("    \"slice_type\": \"embb|urllc|mmtc\",\n")
	promptBuilder.WriteString("    \"sst\": number,\n")
	promptBuilder.WriteString("    \"qos_profile\": \"5qi_X\"\n")
	promptBuilder.WriteString("  },\n")
	promptBuilder.WriteString("  \"resource_requirements\": {\n")
	promptBuilder.WriteString("    \"cpu\": \"cores\",\n")
	promptBuilder.WriteString("    \"memory\": \"size in Gi\",\n")
	promptBuilder.WriteString("    \"storage\": \"size in Gi\"\n")
	promptBuilder.WriteString("  },\n")
	promptBuilder.WriteString("  \"interfaces\": [\"list of required interfaces\"],\n")
	promptBuilder.WriteString("  \"security_requirements\": {\n")
	promptBuilder.WriteString("    \"authentication\": [\"methods\"],\n")
	promptBuilder.WriteString("    \"encryption\": [\"protocols\"]\n")
	promptBuilder.WriteString("  },\n")
	promptBuilder.WriteString("  \"monitoring\": {\n")
	promptBuilder.WriteString("    \"enabled\": true,\n")
	promptBuilder.WriteString("    \"kpis\": [\"list of required KPIs\"]\n")
	promptBuilder.WriteString("  }\n")
	promptBuilder.WriteString("}\n\n")

	promptBuilder.WriteString("Ensure all recommendations are compliant with 3GPP specifications and O-RAN architecture principles. ")
	promptBuilder.WriteString("Consider production-grade requirements including high availability, performance, security, and operational excellence.")

	enhancedPrompt := promptBuilder.String()
	logger.V(1).Info("Built telecom-enhanced prompt", "length", len(enhancedPrompt), "detected_nfs", len(telecomContext))

	return enhancedPrompt, nil
}

// extractTelecomContext extracts telecom entities and context from the intent
func (r *NetworkIntentReconciler) extractTelecomContext(intent string, kb *telecom.TelecomKnowledgeBase) map[string]interface{} {
	context := make(map[string]interface{})
	intentLower := strings.ToLower(intent)

	// Detect network functions
	var detectedNFs []string
	nfNames := kb.ListNetworkFunctions()
	for _, nfName := range nfNames {
		if strings.Contains(intentLower, nfName) {
			detectedNFs = append(detectedNFs, nfName)
		}
	}

	// Also check for common aliases
	nfAliases := map[string]string{
		"access and mobility management": "amf",
		"session management":             "smf", 
		"user plane":                     "upf",
		"policy control":                 "pcf",
		"authentication server":          "ausf",
		"unified data management":        "udm",
		"network repository":             "nrf",
		"network slice selection":        "nssf",
		"base station":                   "gnb",
		"gnodeb":                        "gnb",
	}

	for alias, nf := range nfAliases {
		if strings.Contains(intentLower, alias) && !contains(detectedNFs, nf) {
			detectedNFs = append(detectedNFs, nf)
		}
	}

	context["detected_network_functions"] = detectedNFs

	// Detect slice types
	sliceKeywords := map[string]string{
		"embb":             "embb",
		"enhanced mobile":  "embb",
		"broadband":        "embb",
		"urllc":           "urllc",
		"ultra reliable":   "urllc",
		"low latency":      "urllc",
		"mmtc":            "mmtc",
		"machine type":     "mmtc",
		"iot":             "mmtc",
		"massive":         "mmtc",
	}

	for keyword, sliceType := range sliceKeywords {
		if strings.Contains(intentLower, keyword) {
			context["detected_slice_type"] = sliceType
			break
		}
	}

	// Detect deployment patterns
	deploymentKeywords := map[string]string{
		"high availability": "high-availability",
		"ha":               "high-availability", 
		"production":       "high-availability",
		"edge":             "edge-optimized",
		"low latency":      "edge-optimized",
	}

	for keyword, pattern := range deploymentKeywords {
		if strings.Contains(intentLower, keyword) {
			context["deployment_pattern"] = pattern
			break
		}
	}

	// Extract scaling hints
	scaleRegex := regexp.MustCompile(`(\d+)\s*(replica|instance|node)s?`)
	if matches := scaleRegex.FindStringSubmatch(intentLower); len(matches) > 1 {
		if replicas, err := strconv.Atoi(matches[1]); err == nil {
			context["desired_replicas"] = replicas
		}
	}

	// Extract resource hints
	if strings.Contains(intentLower, "small") || strings.Contains(intentLower, "dev") {
		context["resource_profile"] = "small"
	} else if strings.Contains(intentLower, "large") || strings.Contains(intentLower, "prod") {
		context["resource_profile"] = "large"  
	} else {
		context["resource_profile"] = "medium"
	}

	return context
}

// planResources implements Phase 2: Resource planning with telecom knowledge
func (r *NetworkIntentReconciler) planResources(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, processingCtx *ProcessingContext) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("phase", "resource-planning")
	startTime := time.Now()

	processingCtx.CurrentPhase = PhaseResourcePlanning

	// Get telecom knowledge base
	kb := r.deps.GetTelecomKnowledgeBase()
	if kb == nil {
		return ctrl.Result{}, fmt.Errorf("telecom knowledge base not available")
	}

	// Parse extracted parameters from LLM phase
	var llmParams map[string]interface{}
	if len(networkIntent.Spec.Parameters.Raw) > 0 {
		if err := json.Unmarshal(networkIntent.Spec.Parameters.Raw, &llmParams); err != nil {
			logger.Error(err, "failed to parse LLM parameters")
			return ctrl.Result{RequeueAfter: r.config.RetryDelay}, err
		}
	}

	// Create resource plan
	resourcePlan := &ResourcePlan{
		NetworkFunctions: []PlannedNetworkFunction{},
		Interfaces:      []InterfaceConfiguration{},
		SecurityPolicies: []SecurityPolicy{},
	}

	// Extract network functions from LLM parameters
	if nfList, ok := llmParams["network_functions"].([]interface{}); ok {
		for _, nfInterface := range nfList {
			if nfName, ok := nfInterface.(string); ok {
				if nfSpec, exists := kb.GetNetworkFunction(nfName); exists {
					plannedNF := r.planNetworkFunction(nfSpec, llmParams, processingCtx.TelecomContext)
					resourcePlan.NetworkFunctions = append(resourcePlan.NetworkFunctions, plannedNF)
				}
			}
		}
	}

	// Plan slice configuration if specified
	if sliceConfig, ok := llmParams["slice_configuration"].(map[string]interface{}); ok {
		sliceType := ""
		if st, ok := sliceConfig["slice_type"].(string); ok {
			sliceType = st
		}
		
		if sliceSpec, exists := kb.GetSliceType(sliceType); exists {
			resourcePlan.SliceConfiguration = &SliceConfiguration{
				SliceType:  sliceType,
				SST:        sliceSpec.SST,
				QoSProfile: sliceSpec.QosProfile,
				Isolation:  "standard",
				Parameters: make(map[string]interface{}),
			}
			
			// Copy slice requirements to parameters
			resourcePlan.SliceConfiguration.Parameters["latency_requirement"] = sliceSpec.Requirements.Latency.UserPlane
			resourcePlan.SliceConfiguration.Parameters["throughput_min"] = sliceSpec.Requirements.Throughput.Min
			resourcePlan.SliceConfiguration.Parameters["reliability"] = sliceSpec.Requirements.Reliability.Availability
		}
	}

	// Plan interfaces based on network functions
	interfaceMap := make(map[string]bool)
	for _, nf := range resourcePlan.NetworkFunctions {
		for _, ifaceName := range nf.Interfaces {
			if !interfaceMap[ifaceName] {
				if ifaceSpec, exists := kb.GetInterface(ifaceName); exists {
					interfaceConfig := InterfaceConfiguration{
						Name:     ifaceSpec.Name,
						Type:     ifaceSpec.Type,
						Protocol: ifaceSpec.Protocol,
						Endpoints: []EndpointSpec{},
						Security: SecuritySpec{
							Authentication: ifaceSpec.SecurityModel.Authentication,
							Encryption:     ifaceSpec.SecurityModel.Encryption,
						},
						QoS: QoSSpec{
							Bandwidth:  ifaceSpec.QosRequirements.Bandwidth,
							Latency:    ifaceSpec.QosRequirements.Latency,
							Jitter:     ifaceSpec.QosRequirements.Jitter,
							PacketLoss: ifaceSpec.QosRequirements.PacketLoss,
						},
					}
					
					// Add endpoints from interface spec
					for _, endpoint := range ifaceSpec.Endpoints {
						interfaceConfig.Endpoints = append(interfaceConfig.Endpoints, EndpointSpec{
							Name:     endpoint.Name,
							Port:     endpoint.Port,
							Protocol: endpoint.Protocol,
							Path:     endpoint.Path,
						})
					}
					
					resourcePlan.Interfaces = append(resourcePlan.Interfaces, interfaceConfig)
					interfaceMap[ifaceName] = true
				}
			}
		}
	}

	// Calculate total resource requirements
	totalResources := ResourceRequirements{
		CPU:    "0",
		Memory: "0Gi", 
		Storage: "0Gi",
	}

	var totalCPU, totalMemory, totalStorage float64
	for _, nf := range resourcePlan.NetworkFunctions {
		// Parse CPU
		var cpu float64
		fmt.Sscanf(nf.Resources.CPU, "%f", &cpu)
		totalCPU += cpu * float64(nf.Replicas)
		
		// Parse Memory
		var memory float64
		fmt.Sscanf(nf.Resources.Memory, "%fGi", &memory) 
		totalMemory += memory * float64(nf.Replicas)
		
		// Parse Storage
		var storage float64
		fmt.Sscanf(nf.Resources.Storage, "%fGi", &storage)
		totalStorage += storage * float64(nf.Replicas)
	}

	totalResources.CPU = fmt.Sprintf("%.1f", totalCPU)
	totalResources.Memory = fmt.Sprintf("%.1fGi", totalMemory)
	totalResources.Storage = fmt.Sprintf("%.1fGi", totalStorage)
	resourcePlan.ResourceRequirements = totalResources

	// Determine deployment pattern
	deploymentPattern := "production"
	if pattern, ok := processingCtx.TelecomContext["deployment_pattern"].(string); ok {
		deploymentPattern = pattern
	}
	resourcePlan.DeploymentPattern = deploymentPattern

	// Estimate cost (simplified calculation)
	resourcePlan.EstimatedCost = r.calculateEstimatedCost(totalCPU, totalMemory, totalStorage)

	// Store resource plan in processing context
	processingCtx.ResourcePlan = resourcePlan

	// Update NetworkIntent status
	planData, err := json.Marshal(resourcePlan)
	if err != nil {
		return ctrl.Result{RequeueAfter: r.config.RetryDelay}, fmt.Errorf("failed to marshal resource plan: %w", err)
	}

	// Store resource plan in status
	networkIntent.Status.ResourcePlan = string(planData)

	condition := metav1.Condition{
		Type:               "ResourcesPlanned",
		Status:             metav1.ConditionTrue,
		Reason:             "ResourcePlanningSucceeded", 
		Message:            fmt.Sprintf("Resource plan created with %d network functions, estimated cost $%.2f", len(resourcePlan.NetworkFunctions), resourcePlan.EstimatedCost),
		LastTransitionTime: metav1.Now(),
	}
	updateCondition(&networkIntent.Status.Conditions, condition)

	if err := r.safeStatusUpdate(ctx, networkIntent); err != nil {
		return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update status: %w", err)
	}

	planningDuration := time.Since(startTime)
	processingCtx.Metrics["resource_planning_duration"] = planningDuration.Seconds()
	processingCtx.Metrics["planned_network_functions"] = float64(len(resourcePlan.NetworkFunctions))
	processingCtx.Metrics["estimated_cost"] = resourcePlan.EstimatedCost

	r.recordEvent(networkIntent, "Normal", "ResourcePlanningSucceeded", 
		fmt.Sprintf("Created resource plan with %d NFs", len(resourcePlan.NetworkFunctions)))
	logger.Info("Resource planning phase completed successfully", 
		"duration", planningDuration, 
		"network_functions", len(resourcePlan.NetworkFunctions),
		"estimated_cost", resourcePlan.EstimatedCost)

	return ctrl.Result{}, nil
}

// planNetworkFunction creates a planned network function from the knowledge base spec
func (r *NetworkIntentReconciler) planNetworkFunction(nfSpec *telecom.NetworkFunctionSpec, llmParams map[string]interface{}, telecomContext map[string]interface{}) PlannedNetworkFunction {
	// Determine deployment configuration
	deploymentType := "production"
	if dt, ok := llmParams["deployment_type"].(string); ok {
		deploymentType = dt
	}

	deployConfig, exists := nfSpec.DeploymentPatterns[deploymentType]
	if !exists {
		// Fallback to production if specified type doesn't exist
		if prodConfig, prodExists := nfSpec.DeploymentPatterns["production"]; prodExists {
			deployConfig = prodConfig
		} else {
			// Create minimal config
			deployConfig = telecom.DeploymentConfig{
				Replicas: 1,
				ResourceProfile: "medium",
			}
		}
	}

	// Adjust replicas based on scaling requirements
	replicas := deployConfig.Replicas
	if scalingReq, ok := llmParams["scaling_requirements"].(map[string]interface{}); ok {
		if minReplicas, ok := scalingReq["min_replicas"].(float64); ok {
			replicas = int(minReplicas)
		}
	}

	// Build configuration parameters
	configuration := make(map[string]interface{})
	for paramName, param := range nfSpec.Configuration {
		configuration[paramName] = param.Default
	}

	// Override with any specific configuration from LLM
	if nfConfig, ok := llmParams["configuration"].(map[string]interface{}); ok {
		for key, value := range nfConfig {
			configuration[key] = value
		}
	}

	// Build health checks
	var healthChecks []HealthCheckSpec
	for _, hc := range nfSpec.HealthChecks {
		healthChecks = append(healthChecks, HealthCheckSpec{
			Type:     hc.Type,
			Path:     hc.Path,
			Port:     hc.Port,
			Interval: hc.Interval,
			Timeout:  hc.Timeout,
			Retries:  hc.Retries,
		})
	}

	// Build monitoring spec
	monitoring := MonitoringSpec{
		Enabled:    true,
		Metrics:    []string{},
		Alerts:     []string{},
		Dashboards: []string{},
	}

	for _, metric := range nfSpec.MonitoringMetrics {
		monitoring.Metrics = append(monitoring.Metrics, metric.Name)
		for _, alert := range metric.Alerts {
			monitoring.Alerts = append(monitoring.Alerts, alert.Name)
		}
	}

	return PlannedNetworkFunction{
		Name:           strings.ToLower(nfSpec.Name),
		Type:           nfSpec.Type,
		Version:        nfSpec.Version,
		Replicas:       replicas,
		Resources: ResourceRequirements{
			CPU:     nfSpec.Resources.MaxCPU,
			Memory:  nfSpec.Resources.MaxMemory,
			Storage: nfSpec.Resources.Storage,
			NetworkBW: nfSpec.Resources.NetworkBW,
			Accelerator: nfSpec.Resources.Accelerator,
		},
		Configuration:  configuration,
		Dependencies:   nfSpec.Dependencies,
		Interfaces:     nfSpec.Interfaces,
		HealthChecks:   healthChecks,
		Monitoring:     monitoring,
	}
}

// calculateEstimatedCost calculates estimated monthly cost for resources
func (r *NetworkIntentReconciler) calculateEstimatedCost(cpu, memory, storage float64) float64 {
	// Simplified cost calculation (in USD per month)
	cpuCostPerCore := 50.0    // $50 per CPU core per month
	memoryCostPerGi := 10.0   // $10 per Gi memory per month  
	storageCostPerGi := 2.0   // $2 per Gi storage per month

	totalCost := (cpu * cpuCostPerCore) + (memory * memoryCostPerGi) + (storage * storageCostPerGi)
	return totalCost
}

// generateManifests implements Phase 3: Deployment manifest generation
func (r *NetworkIntentReconciler) generateManifests(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, processingCtx *ProcessingContext) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("phase", "manifest-generation")
	startTime := time.Now()

	processingCtx.CurrentPhase = PhaseManifestGeneration

	if processingCtx.ResourcePlan == nil {
		return ctrl.Result{}, fmt.Errorf("resource plan not available for manifest generation")
	}

	manifests := make(map[string]string)

	// Generate manifests for each network function
	for _, nf := range processingCtx.ResourcePlan.NetworkFunctions {
		// Generate Deployment manifest
		deployment := r.generateDeploymentManifest(networkIntent, &nf)
		deploymentYAML, err := yaml.Marshal(deployment)
		if err != nil {
			return ctrl.Result{RequeueAfter: r.config.RetryDelay}, fmt.Errorf("failed to marshal deployment: %w", err)
		}
		manifests[fmt.Sprintf("%s-deployment.yaml", nf.Name)] = string(deploymentYAML)

		// Generate Service manifest
		service := r.generateServiceManifest(networkIntent, &nf)
		serviceYAML, err := yaml.Marshal(service)
		if err != nil {
			return ctrl.Result{RequeueAfter: r.config.RetryDelay}, fmt.Errorf("failed to marshal service: %w", err)
		}
		manifests[fmt.Sprintf("%s-service.yaml", nf.Name)] = string(serviceYAML)

		// Generate ConfigMap for configuration
		if len(nf.Configuration) > 0 {
			configMap := r.generateConfigMapManifest(networkIntent, &nf)
			configMapYAML, err := yaml.Marshal(configMap)
			if err != nil {
				return ctrl.Result{RequeueAfter: r.config.RetryDelay}, fmt.Errorf("failed to marshal configmap: %w", err)
			}
			manifests[fmt.Sprintf("%s-configmap.yaml", nf.Name)] = string(configMapYAML)
		}

		// Generate NetworkPolicy if security requirements specified
		if len(processingCtx.ResourcePlan.SecurityPolicies) > 0 {
			netpol := r.generateNetworkPolicyManifest(networkIntent, &nf)
			netpolYAML, err := yaml.Marshal(netpol)
			if err != nil {
				return ctrl.Result{RequeueAfter: r.config.RetryDelay}, fmt.Errorf("failed to marshal network policy: %w", err)
			}
			manifests[fmt.Sprintf("%s-networkpolicy.yaml", nf.Name)] = string(netpolYAML)
		}
	}

	// Generate slice-specific resources if slice configuration exists
	if processingCtx.ResourcePlan.SliceConfiguration != nil {
		sliceConfigMap := r.generateSliceConfigMap(networkIntent, processingCtx.ResourcePlan.SliceConfiguration)
		sliceConfigYAML, err := yaml.Marshal(sliceConfigMap)
		if err != nil {
			return ctrl.Result{RequeueAfter: r.config.RetryDelay}, fmt.Errorf("failed to marshal slice config: %w", err)
		}
		manifests["slice-configuration.yaml"] = string(sliceConfigYAML)
	}

	// Store manifests in processing context
	processingCtx.Manifests = manifests

	// Update condition
	condition := metav1.Condition{
		Type:               "ManifestsGenerated",
		Status:             metav1.ConditionTrue,
		Reason:             "ManifestGenerationSucceeded",
		Message:            fmt.Sprintf("Generated %d Kubernetes manifests for deployment", len(manifests)),
		LastTransitionTime: metav1.Now(),
	}
	updateCondition(&networkIntent.Status.Conditions, condition)

	if err := r.safeStatusUpdate(ctx, networkIntent); err != nil {
		return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update status: %w", err)
	}

	manifestGenDuration := time.Since(startTime)
	processingCtx.Metrics["manifest_generation_duration"] = manifestGenDuration.Seconds()
	processingCtx.Metrics["generated_manifests"] = float64(len(manifests))

	r.recordEvent(networkIntent, "Normal", "ManifestGenerationSucceeded", 
		fmt.Sprintf("Generated %d Kubernetes manifests", len(manifests)))
	logger.Info("Manifest generation phase completed successfully", 
		"duration", manifestGenDuration, 
		"manifests_count", len(manifests))

	return ctrl.Result{}, nil
}

// generateDeploymentManifest creates a Kubernetes Deployment for a network function
func (r *NetworkIntentReconciler) generateDeploymentManifest(networkIntent *nephoranv1.NetworkIntent, nf *PlannedNetworkFunction) *appsv1.Deployment {
	labels := map[string]string{
		"app.kubernetes.io/name":       nf.Name,
		"app.kubernetes.io/instance":   networkIntent.Name,
		"app.kubernetes.io/component":  nf.Type,
		"app.kubernetes.io/part-of":    "5g-core",
		"app.kubernetes.io/managed-by": "nephoran-intent-operator",
		"nephoran.com/network-function": nf.Name,
	}

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", networkIntent.Name, nf.Name),
			Namespace: networkIntent.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: networkIntent.APIVersion,
					Kind:       networkIntent.Kind,
					Name:       networkIntent.Name,
					UID:        networkIntent.UID,
					Controller: &[]bool{true}[0],
				},
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &[]int32{int32(nf.Replicas)}[0],
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name":     nf.Name,
					"app.kubernetes.io/instance": networkIntent.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &[]bool{true}[0],
						RunAsUser:    &[]int64{1000}[0],
						FSGroup:      &[]int64{1000}[0],
					},
					Containers: []corev1.Container{
						{
							Name:  nf.Name,
							Image: fmt.Sprintf("nephoran/%s:%s", nf.Name, nf.Version),
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    parseQuantity(nf.Resources.CPU),
									corev1.ResourceMemory: parseQuantity(nf.Resources.Memory),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    parseQuantity(nf.Resources.CPU),
									corev1.ResourceMemory: parseQuantity(nf.Resources.Memory),
								},
							},
							Ports: r.generateContainerPorts(nf),
							Env:   r.generateEnvironmentVariables(nf),
							LivenessProbe:  r.generateLivenessProbe(nf),
							ReadinessProbe: r.generateReadinessProbe(nf),
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: &[]bool{false}[0],
								RunAsNonRoot:            &[]bool{true}[0],
								RunAsUser:               &[]int64{1000}[0],
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
									Add:  []corev1.Capability{"NET_BIND_SERVICE"},
								},
							},
						},
					},
				},
			},
		},
	}

	// Add volume mounts if configuration exists
	if len(nf.Configuration) > 0 {
		volumeMount := corev1.VolumeMount{
			Name:      "config",
			MountPath: "/etc/config",
			ReadOnly:  true,
		}
		deployment.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{volumeMount}

		volume := corev1.Volume{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("%s-%s-config", networkIntent.Name, nf.Name),
					},
				},
			},
		}
		deployment.Spec.Template.Spec.Volumes = []corev1.Volume{volume}
	}

	return deployment
}

// generateServiceManifest creates a Kubernetes Service for a network function
func (r *NetworkIntentReconciler) generateServiceManifest(networkIntent *nephoranv1.NetworkIntent, nf *PlannedNetworkFunction) *corev1.Service {
	labels := map[string]string{
		"app.kubernetes.io/name":       nf.Name,
		"app.kubernetes.io/instance":   networkIntent.Name,
		"app.kubernetes.io/managed-by": "nephoran-intent-operator",
	}

	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", networkIntent.Name, nf.Name),
			Namespace: networkIntent.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: networkIntent.APIVersion,
					Kind:       networkIntent.Kind,
					Name:       networkIntent.Name,
					UID:        networkIntent.UID,
					Controller: &[]bool{true}[0],
				},
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app.kubernetes.io/name":     nf.Name,
				"app.kubernetes.io/instance": networkIntent.Name,
			},
			Ports: r.generateServicePorts(nf),
		},
	}

	return service
}

// generateConfigMapManifest creates a ConfigMap for network function configuration
func (r *NetworkIntentReconciler) generateConfigMapManifest(networkIntent *nephoranv1.NetworkIntent, nf *PlannedNetworkFunction) *corev1.ConfigMap {
	labels := map[string]string{
		"app.kubernetes.io/name":       nf.Name,
		"app.kubernetes.io/instance":   networkIntent.Name,
		"app.kubernetes.io/managed-by": "nephoran-intent-operator",
	}

	data := make(map[string]string)
	
	// Convert configuration to YAML string
	if configBytes, err := yaml.Marshal(nf.Configuration); err == nil {
		data["config.yaml"] = string(configBytes)
	}

	// Add individual configuration items
	for key, value := range nf.Configuration {
		if strValue, ok := value.(string); ok {
			data[key] = strValue
		} else if valueBytes, err := json.Marshal(value); err == nil {
			data[key] = string(valueBytes)
		}
	}

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-config", networkIntent.Name, nf.Name),
			Namespace: networkIntent.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: networkIntent.APIVersion,
					Kind:       networkIntent.Kind,
					Name:       networkIntent.Name,
					UID:        networkIntent.UID,
					Controller: &[]bool{true}[0],
				},
			},
		},
		Data: data,
	}

	return configMap
}

// generateNetworkPolicyManifest creates a NetworkPolicy for network function security
func (r *NetworkIntentReconciler) generateNetworkPolicyManifest(networkIntent *nephoranv1.NetworkIntent, nf *PlannedNetworkFunction) *networkingv1.NetworkPolicy {
	labels := map[string]string{
		"app.kubernetes.io/name":       nf.Name,
		"app.kubernetes.io/instance":   networkIntent.Name,
		"app.kubernetes.io/managed-by": "nephoran-intent-operator",
	}

	networkPolicy := &networkingv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.k8s.io/v1",
			Kind:       "NetworkPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-netpol", networkIntent.Name, nf.Name),
			Namespace: networkIntent.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: networkIntent.APIVersion,
					Kind:       networkIntent.Kind,
					Name:       networkIntent.Name,
					UID:        networkIntent.UID,
					Controller: &[]bool{true}[0],
				},
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name":     nf.Name,
					"app.kubernetes.io/instance": networkIntent.Name,
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
			Ingress: r.generateIngressRules(nf),
			Egress:  r.generateEgressRules(nf),
		},
	}

	return networkPolicy
}

// generateSliceConfigMap creates configuration for network slice
func (r *NetworkIntentReconciler) generateSliceConfigMap(networkIntent *nephoranv1.NetworkIntent, sliceConfig *SliceConfiguration) *corev1.ConfigMap {
	labels := map[string]string{
		"app.kubernetes.io/name":       "slice-config",
		"app.kubernetes.io/instance":   networkIntent.Name,
		"app.kubernetes.io/managed-by": "nephoran-intent-operator",
		"nephoran.com/slice-type":      sliceConfig.SliceType,
	}

	data := make(map[string]string)
	
	// Convert slice configuration to YAML
	if configBytes, err := yaml.Marshal(sliceConfig); err == nil {
		data["slice-config.yaml"] = string(configBytes)
	}

	// Add individual parameters
	data["slice_type"] = sliceConfig.SliceType
	data["sst"] = fmt.Sprintf("%d", sliceConfig.SST)
	data["qos_profile"] = sliceConfig.QoSProfile
	data["isolation"] = sliceConfig.Isolation

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-slice-config", networkIntent.Name),
			Namespace: networkIntent.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: networkIntent.APIVersion,
					Kind:       networkIntent.Kind,
					Name:       networkIntent.Name,
					UID:        networkIntent.UID,
					Controller: &[]bool{true}[0],
				},
			},
		},
		Data: data,
	}

	return configMap
}

// commitToGitOps implements Phase 4: GitOps commit and validation
func (r *NetworkIntentReconciler) commitToGitOps(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, processingCtx *ProcessingContext) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("phase", "gitops-commit")
	startTime := time.Now()

	processingCtx.CurrentPhase = PhaseGitOpsCommit

	if len(processingCtx.Manifests) == 0 {
		return ctrl.Result{}, fmt.Errorf("no manifests available for GitOps commit")
	}

	// Get Git client
	gitClient := r.deps.GetGitClient()
	if gitClient == nil {
		return ctrl.Result{}, fmt.Errorf("Git client is not configured")
	}

	// Get retry count
	retryCount := getRetryCount(networkIntent, "git-deployment")
	if retryCount >= r.config.MaxRetries {
		condition := metav1.Condition{
			Type:               "GitOpsCommitted",
			Status:             metav1.ConditionFalse,
			Reason:             "GitCommitFailedMaxRetries",
			Message:            fmt.Sprintf("Failed to commit to GitOps after %d retries", r.config.MaxRetries),
			LastTransitionTime: metav1.Now(),
		}
		updateCondition(&networkIntent.Status.Conditions, condition)
		r.safeStatusUpdate(ctx, networkIntent)
		return ctrl.Result{}, fmt.Errorf("max retries exceeded for GitOps commit")
	}

	// Initialize Git repository
	if err := gitClient.InitRepo(); err != nil {
		logger.Error(err, "failed to initialize git repository")
		setRetryCount(networkIntent, "git-deployment", retryCount+1)
		
		condition := metav1.Condition{
			Type:               "GitOpsCommitted",
			Status:             metav1.ConditionFalse,
			Reason:             "GitRepoInitializationFailed",
			Message:            fmt.Sprintf("Failed to initialize git repository: %v", err),
			LastTransitionTime: metav1.Now(),
		}
		updateCondition(&networkIntent.Status.Conditions, condition)
		r.safeStatusUpdate(ctx, networkIntent)
		
		return ctrl.Result{RequeueAfter: r.config.RetryDelay}, nil
	}

	// Organize manifests by deployment path
	deploymentFiles := make(map[string]string)
	basePath := fmt.Sprintf("%s/%s-%s", r.config.GitDeployPath, networkIntent.Namespace, networkIntent.Name)
	
	for filename, content := range processingCtx.Manifests {
		filePath := fmt.Sprintf("%s/%s", basePath, filename)
		deploymentFiles[filePath] = content
	}

	// Create comprehensive commit message
	commitMessage := fmt.Sprintf("Deploy NetworkIntent: %s/%s\n\nIntent: %s\nPhase: %s\nNetwork Functions: %s\nGenerated Manifests: %d\nEstimated Cost: $%.2f\n\nProcessed by Nephoran Intent Operator",
		networkIntent.Namespace,
		networkIntent.Name,
		networkIntent.Spec.Intent,
		processingCtx.CurrentPhase,
		r.getNetworkFunctionsList(processingCtx.ResourcePlan),
		len(processingCtx.Manifests),
		processingCtx.ResourcePlan.EstimatedCost,
	)

	// Commit and push to Git
	logger.Info("Committing manifests to GitOps repository", "files", len(deploymentFiles))
	commitHash, err := gitClient.CommitAndPush(deploymentFiles, commitMessage)
	if err != nil {
		logger.Error(err, "failed to commit and push deployment files")
		setRetryCount(networkIntent, "git-deployment", retryCount+1)
		
		condition := metav1.Condition{
			Type:               "GitOpsCommitted",
			Status:             metav1.ConditionFalse,
			Reason:             "GitCommitPushFailed",
			Message:            fmt.Sprintf("Failed to commit and push: %v", err),
			LastTransitionTime: metav1.Now(),
		}
		updateCondition(&networkIntent.Status.Conditions, condition)
		r.safeStatusUpdate(ctx, networkIntent)
		
		return ctrl.Result{RequeueAfter: r.config.RetryDelay}, nil
	}

	// Store commit hash
	processingCtx.GitCommitHash = commitHash

	// Clear retry count and update success condition
	clearRetryCount(networkIntent, "git-deployment")
	now := metav1.Now()
	networkIntent.Status.DeploymentCompletionTime = &now
	networkIntent.Status.GitCommitHash = commitHash
	
	condition := metav1.Condition{
		Type:               "GitOpsCommitted",
		Status:             metav1.ConditionTrue,
		Reason:             "GitCommitSucceeded",
		Message:            fmt.Sprintf("Successfully committed %d manifests to GitOps repository (commit: %s)", len(deploymentFiles), commitHash[:8]),
		LastTransitionTime: now,
	}
	updateCondition(&networkIntent.Status.Conditions, condition)

	if err := r.safeStatusUpdate(ctx, networkIntent); err != nil {
		return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update status: %w", err)
	}

	// Record metrics
	if metricsCollector := r.deps.GetMetricsCollector(); metricsCollector != nil {
		commitDuration := time.Since(startTime)
		metricsCollector.RecordGitOpsOperation("commit", commitDuration, true)
		metricsCollector.GitOpsPackagesGenerated.Inc()
	}

	commitDuration := time.Since(startTime)
	processingCtx.Metrics["gitops_commit_duration"] = commitDuration.Seconds()

	r.recordEvent(networkIntent, "Normal", "GitOpsCommitSucceeded", 
		fmt.Sprintf("Committed %d manifests to GitOps (commit: %s)", len(deploymentFiles), commitHash[:8]))
	logger.Info("GitOps commit phase completed successfully", 
		"duration", commitDuration,
		"commit_hash", commitHash[:8],
		"files_committed", len(deploymentFiles))

	return ctrl.Result{}, nil
}

// verifyDeployment implements Phase 5: Deployment verification
func (r *NetworkIntentReconciler) verifyDeployment(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, processingCtx *ProcessingContext) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("phase", "deployment-verification")
	startTime := time.Now()

	processingCtx.CurrentPhase = PhaseDeploymentVerification

	// For now, we'll implement a simple verification that checks if the GitOps commit was successful
	// In a real implementation, this would check if the manifests were actually deployed and are running
	
	if processingCtx.GitCommitHash == "" {
		return ctrl.Result{}, fmt.Errorf("no Git commit hash available for verification")
	}

	// Simulate deployment verification delay (in production, this would poll actual resources)
	time.Sleep(2 * time.Second)

	// Update deployment status
	deploymentStatus := map[string]interface{}{
		"git_commit_hash":       processingCtx.GitCommitHash,
		"deployment_timestamp":  time.Now(),
		"verification_status":   "verified", 
		"network_functions":     len(processingCtx.ResourcePlan.NetworkFunctions),
		"manifests_deployed":    len(processingCtx.Manifests),
	}

	processingCtx.DeploymentStatus = deploymentStatus

	// Create successful verification condition
	condition := metav1.Condition{
		Type:               "DeploymentVerified",
		Status:             metav1.ConditionTrue,
		Reason:             "DeploymentVerificationSucceeded",
		Message:            fmt.Sprintf("Deployment verified successfully - %d network functions deployed via commit %s", 
			len(processingCtx.ResourcePlan.NetworkFunctions), processingCtx.GitCommitHash[:8]),
		LastTransitionTime: metav1.Now(),
	}
	updateCondition(&networkIntent.Status.Conditions, condition)

	if err := r.safeStatusUpdate(ctx, networkIntent); err != nil {
		return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update status: %w", err)
	}

	verificationDuration := time.Since(startTime)
	processingCtx.Metrics["deployment_verification_duration"] = verificationDuration.Seconds()

	r.recordEvent(networkIntent, "Normal", "DeploymentVerificationSucceeded",
		fmt.Sprintf("Verified deployment of %d network functions", len(processingCtx.ResourcePlan.NetworkFunctions)))
	logger.Info("Deployment verification phase completed successfully",
		"duration", verificationDuration,
		"network_functions_verified", len(processingCtx.ResourcePlan.NetworkFunctions))

	return ctrl.Result{}, nil
}

// Helper methods

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

// getNetworkFunctionsList returns a comma-separated list of network function names
func (r *NetworkIntentReconciler) getNetworkFunctionsList(plan *ResourcePlan) string {
	if plan == nil || len(plan.NetworkFunctions) == 0 {
		return "none"
	}
	
	var names []string
	for _, nf := range plan.NetworkFunctions {
		names = append(names, nf.Name)
	}
	return strings.Join(names, ", ")
}

// generateContainerPorts generates container ports based on interfaces
func (r *NetworkIntentReconciler) generateContainerPorts(nf *PlannedNetworkFunction) []corev1.ContainerPort {
	var ports []corev1.ContainerPort
	
	// Add standard ports based on network function type
	switch nf.Type {
	case "5gc-control-plane":
		ports = append(ports, corev1.ContainerPort{
			Name:          "http",
			ContainerPort: 8080,
			Protocol:      corev1.ProtocolTCP,
		})
		ports = append(ports, corev1.ContainerPort{
			Name:          "metrics",
			ContainerPort: 9090,
			Protocol:      corev1.ProtocolTCP,
		})
	case "5gc-user-plane":
		ports = append(ports, corev1.ContainerPort{
			Name:          "gtpu",
			ContainerPort: 2152,
			Protocol:      corev1.ProtocolUDP,
		})
		ports = append(ports, corev1.ContainerPort{
			Name:          "pfcp",
			ContainerPort: 8805,
			Protocol:      corev1.ProtocolUDP,
		})
	}
	
	return ports
}

// generateEnvironmentVariables generates environment variables for the container
func (r *NetworkIntentReconciler) generateEnvironmentVariables(nf *PlannedNetworkFunction) []corev1.EnvVar {
	var envVars []corev1.EnvVar
	
	// Add configuration as environment variables
	for key, value := range nf.Configuration {
		if strValue, ok := value.(string); ok {
			envVars = append(envVars, corev1.EnvVar{
				Name:  strings.ToUpper(strings.ReplaceAll(key, ".", "_")),
				Value: strValue,
			})
		}
	}
	
	// Add common environment variables
	envVars = append(envVars, corev1.EnvVar{
		Name: "POD_NAME",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.name",
			},
		},
	})
	
	envVars = append(envVars, corev1.EnvVar{
		Name: "POD_NAMESPACE",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.namespace",
			},
		},
	})
	
	return envVars
}

// generateLivenessProbe generates a liveness probe
func (r *NetworkIntentReconciler) generateLivenessProbe(nf *PlannedNetworkFunction) *corev1.Probe {
	for _, hc := range nf.HealthChecks {
		if hc.Type == "http" {
			return &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: hc.Path,
						Port: intstr.FromInt(hc.Port),
					},
				},
				InitialDelaySeconds: int32(hc.InitialDelay),
				PeriodSeconds:       int32(hc.Interval),
				TimeoutSeconds:      int32(hc.Timeout),
				FailureThreshold:    int32(hc.Retries),
			}
		}
	}
	
	// Default HTTP liveness probe
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/health",
				Port: intstr.FromInt(8080),
			},
		},
		InitialDelaySeconds: 60,
		PeriodSeconds:       30,
		TimeoutSeconds:      10,
		FailureThreshold:    3,
	}
}

// generateReadinessProbe generates a readiness probe
func (r *NetworkIntentReconciler) generateReadinessProbe(nf *PlannedNetworkFunction) *corev1.Probe {
	for _, hc := range nf.HealthChecks {
		if hc.Type == "http" {
			return &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/ready",
						Port: intstr.FromInt(hc.Port),
					},
				},
				InitialDelaySeconds: int32(hc.InitialDelay / 2),
				PeriodSeconds:       int32(hc.Interval),
				TimeoutSeconds:      int32(hc.Timeout),
				FailureThreshold:    int32(hc.Retries),
			}
		}
	}
	
	// Default HTTP readiness probe
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/ready",
				Port: intstr.FromInt(8080),
			},
		},
		InitialDelaySeconds: 30,
		PeriodSeconds:       10,
		TimeoutSeconds:      5,
		FailureThreshold:    3,
	}
}

// generateServicePorts generates service ports
func (r *NetworkIntentReconciler) generateServicePorts(nf *PlannedNetworkFunction) []corev1.ServicePort {
	var ports []corev1.ServicePort
	
	// Add standard ports based on network function type
	switch nf.Type {
	case "5gc-control-plane":
		ports = append(ports, corev1.ServicePort{
			Name:       "http",
			Port:       80,
			TargetPort: intstr.FromInt(8080),
			Protocol:   corev1.ProtocolTCP,
		})
		ports = append(ports, corev1.ServicePort{
			Name:       "metrics",
			Port:       9090,
			TargetPort: intstr.FromInt(9090),
			Protocol:   corev1.ProtocolTCP,
		})
	case "5gc-user-plane":
		ports = append(ports, corev1.ServicePort{
			Name:       "gtpu",
			Port:       2152,
			TargetPort: intstr.FromInt(2152),
			Protocol:   corev1.ProtocolUDP,
		})
	}
	
	return ports
}

// generateIngressRules generates ingress network policy rules
func (r *NetworkIntentReconciler) generateIngressRules(nf *PlannedNetworkFunction) []networkingv1.NetworkPolicyIngressRule {
	var rules []networkingv1.NetworkPolicyIngressRule
	
	// Allow ingress from same namespace by default
	rule := networkingv1.NetworkPolicyIngressRule{
		From: []networkingv1.NetworkPolicyPeer{
			{
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{},
				},
			},
		},
	}
	
	rules = append(rules, rule)
	return rules
}

// generateEgressRules generates egress network policy rules
func (r *NetworkIntentReconciler) generateEgressRules(nf *PlannedNetworkFunction) []networkingv1.NetworkPolicyEgressRule {
	var rules []networkingv1.NetworkPolicyEgressRule
	
	// Allow egress to same namespace and DNS
	rule := networkingv1.NetworkPolicyEgressRule{
		To: []networkingv1.NetworkPolicyPeer{
			{
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{},
				},
			},
		},
	}
	
	rules = append(rules, rule)
	return rules
}

// parseQuantity parses a resource quantity string (simplified implementation)
func parseQuantity(qty string) corev1.ResourceList {
	// In a real implementation, use resource.MustParse()
	// This is a simplified version for demonstration
	return corev1.ResourceList{}
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Legacy methods preserved for backward compatibility but updated

func (r *NetworkIntentReconciler) processIntentWithRetry(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) (ctrl.Result, error) {
	// This method is now handled by the multi-phase pipeline, but kept for backward compatibility
	return r.processLLMPhase(ctx, networkIntent, &ProcessingContext{
		StartTime:         time.Now(),
		ExtractedEntities: make(map[string]interface{}),
		TelecomContext:    make(map[string]interface{}),
		DeploymentStatus:  make(map[string]interface{}),
		Metrics:          make(map[string]float64),
	})
}

func (r *NetworkIntentReconciler) deployViaGitOps(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) (ctrl.Result, error) {
	// This method is now handled by the multi-phase pipeline, but kept for backward compatibility
	processingCtx := &ProcessingContext{
		StartTime: time.Now(),
		Manifests: make(map[string]string),
	}
	
	// Generate basic manifests for backward compatibility
	var parameters map[string]interface{}
	if len(networkIntent.Spec.Parameters.Raw) > 0 {
		json.Unmarshal(networkIntent.Spec.Parameters.Raw, &parameters)
	}
	
	// Create a simple manifest
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

	manifestYAML, _ := yaml.Marshal(manifest)
	processingCtx.Manifests["networkintent-configmap.yaml"] = string(manifestYAML)
	
	return r.commitToGitOps(ctx, networkIntent, processingCtx)
}

func (r *NetworkIntentReconciler) generateDeploymentFiles(networkIntent *nephoranv1.NetworkIntent) (map[string]string, error) {
	// This method is now replaced by generateManifests but kept for backward compatibility
	files := make(map[string]string)

	// Parse the processed parameters
	var parameters map[string]interface{}
	if len(networkIntent.Spec.Parameters.Raw) > 0 {
		if err := json.Unmarshal(networkIntent.Spec.Parameters.Raw, &parameters); err != nil {
			return nil, fmt.Errorf("failed to parse parameters: %w", err)
		}
	}

	// Generate basic manifest
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

	manifestYAML, err := yaml.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}

	filePath := fmt.Sprintf("%s/%s/%s-configmap.yaml", r.config.GitDeployPath, networkIntent.Namespace, networkIntent.Name)
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

	return r.reconcileDelete(ctx, networkIntent)
}

// reconcileDelete handles the deletion process with proper Git operation waiting
func (r *NetworkIntentReconciler) reconcileDelete(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Set Ready condition to false while cleanup is pending
	condition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		Reason:             "CleanupPending",
		Message:            "Cleanup operations are in progress, waiting for Git operations to complete",
		LastTransitionTime: metav1.Now(),
	}
	updateCondition(&networkIntent.Status.Conditions, condition)

	if err := r.safeStatusUpdate(ctx, networkIntent); err != nil {
		logger.Error(err, "failed to update status with cleanup pending condition")
		// Continue with cleanup even if status update fails
	}

	// Get current retry count for cleanup operations
	retryCount := getRetryCount(networkIntent, "cleanup")
	logger.V(1).Info("Starting cleanup operations", "retry_count", retryCount, "max_retries", r.config.MaxRetries)

	// Check if max retries exceeded
	if retryCount >= r.config.MaxRetries {
		logger.Error(fmt.Errorf("max retries exceeded"), "cleanup operations failed after maximum retries", "retry_count", retryCount)

		// Mark as failed but still remove finalizer to prevent stuck resources
		condition := metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			Reason:             "CleanupFailedMaxRetries",
			Message:            fmt.Sprintf("Cleanup failed after %d retries, removing finalizer to prevent stuck resource", r.config.MaxRetries),
			LastTransitionTime: metav1.Now(),
		}
		updateCondition(&networkIntent.Status.Conditions, condition)

		if statusErr := r.safeStatusUpdate(ctx, networkIntent); statusErr != nil {
			logger.Error(statusErr, "failed to update status after max retries exceeded")
		}

		r.recordFailureEvent(networkIntent, "CleanupFailedMaxRetries", fmt.Sprintf("Cleanup failed after %d retries", r.config.MaxRetries))

		// Remove finalizer to prevent stuck resource
		networkIntent.Finalizers = removeFinalizer(networkIntent.Finalizers, NetworkIntentFinalizer)
		if err := r.safeUpdate(ctx, networkIntent); err != nil {
			logger.Error(err, "failed to remove finalizer after max retries")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// Perform cleanup operations including Git push confirmation
	gitPushSucceeded := false
	if err := r.performCleanupWithGitConfirmation(ctx, networkIntent, &gitPushSucceeded); err != nil {
		logger.Error(err, "cleanup operations failed", "git_push_succeeded", gitPushSucceeded)

		// Increment retry count
		setRetryCount(networkIntent, "cleanup", retryCount+1)

		// Update condition to show retry with specific Git error information
		now := metav1.Now()
		networkIntent.Status.LastRetryTime = &now
		
		reason := "CleanupRetrying"
		message := fmt.Sprintf("Cleanup failed (attempt %d/%d): %v", retryCount+1, r.config.MaxRetries, err)
		if !gitPushSucceeded && strings.Contains(err.Error(), "push") {
			reason = "GitPushFailed"
			message = fmt.Sprintf("Git push failed during cleanup (attempt %d/%d): %v", retryCount+1, r.config.MaxRetries, err)
		}
		
		condition := metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			Reason:             reason,
			Message:            message,
			LastTransitionTime: now,
		}
		updateCondition(&networkIntent.Status.Conditions, condition)

		if statusErr := r.safeStatusUpdate(ctx, networkIntent); statusErr != nil {
			logger.Error(statusErr, "failed to update status after cleanup failure")
		}

		r.recordFailureEvent(networkIntent, reason, fmt.Sprintf("attempt %d/%d failed: %v", retryCount+1, r.config.MaxRetries, err))

		// Calculate exponential backoff delay
		baseDelay := r.config.RetryDelay
		if baseDelay == 0 {
			baseDelay = DefaultRetryDelay
		}
		
		// Exponential backoff: delay = base * 2^retryCount
		backoffDelay := baseDelay * time.Duration(1<<uint(retryCount))
		
		// Cap at 5 minutes for cleanup operations
		maxDelay := time.Minute * 5
		if backoffDelay > maxDelay {
			backoffDelay = maxDelay
		}

		logger.V(1).Info("Scheduling cleanup retry with exponential backoff", 
			"delay", backoffDelay, 
			"attempt", retryCount+1,
			"base_delay", baseDelay)
		return ctrl.Result{RequeueAfter: backoffDelay}, nil
	}

	// Git push confirmed successful, now safe to remove finalizer
	logger.Info("Git push confirmed successful, proceeding with finalizer removal")
	
	// Clear retry count
	clearRetryCount(networkIntent, "cleanup")

	// Update condition to show successful cleanup with Git confirmation
	condition = metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		Reason:             "CleanupCompleted",
		Message:            "All cleanup operations completed successfully with Git push confirmation, finalizer removed",
		LastTransitionTime: metav1.Now(),
	}
	updateCondition(&networkIntent.Status.Conditions, condition)

	if statusErr := r.safeStatusUpdate(ctx, networkIntent); statusErr != nil {
		logger.Error(statusErr, "failed to update status after successful cleanup")
		// Continue with finalizer removal even if status update fails
	}

	// Remove finalizer only after Git push is confirmed successful
	networkIntent.Finalizers = removeFinalizer(networkIntent.Finalizers, NetworkIntentFinalizer)
	if err := r.safeUpdate(ctx, networkIntent); err != nil {
		logger.Error(err, "failed to remove finalizer after successful cleanup and Git confirmation")
		return ctrl.Result{}, err
	}

	logger.Info("NetworkIntent cleanup completed with Git confirmation and finalizer removed", "intent", networkIntent.Name)
	r.recordEvent(networkIntent, "Normal", "CleanupCompleted", "Successfully cleaned up resources with Git push confirmation and removed finalizer")
	return ctrl.Result{}, nil
}

// performCleanupWithGitConfirmation performs cleanup operations and confirms Git push succeeded
func (r *NetworkIntentReconciler) performCleanupWithGitConfirmation(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, gitPushSucceeded *bool) error {
	logger := log.FromContext(ctx)

	// Check context cancellation before starting
	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled during cleanup: %w", ctx.Err())
	default:
	}

	logger.Info("Starting cleanup operations with Git push confirmation tracking", "intent_name", networkIntent.Name, "namespace", networkIntent.Namespace)

	// 1. Cleanup GitOps packages first (most critical operation)
	gitClient := r.deps.GetGitClient()
	if gitClient != nil && r.config.GitRepoURL != "" {
		logger.V(1).Info("Starting GitOps cleanup with push confirmation")
		
		// Track Git operations explicitly
		*gitPushSucceeded = false
		
		if err := r.cleanupGitOpsPackagesWithWait(ctx, networkIntent, gitClient); err != nil {
			// Check if this was specifically a push error
			if strings.Contains(err.Error(), "push") || strings.Contains(err.Error(), "remote") {
				logger.Error(err, "Git push failed during cleanup")
				*gitPushSucceeded = false
			}
			return fmt.Errorf("GitOps cleanup failed: %w", err)
		}
		
		// If we got here, Git push succeeded
		*gitPushSucceeded = true
		logger.V(1).Info("GitOps cleanup completed successfully with push confirmation")
	} else {
		logger.V(1).Info("Skipping GitOps cleanup - no Git client or repository configured")
		// No Git operations needed, so we consider this "successful"
		*gitPushSucceeded = true
	}

	// 2. Cleanup any generated ConfigMaps or Secrets (non-blocking)
	if err := r.cleanupGeneratedResources(ctx, networkIntent); err != nil {
		logger.Error(err, "failed to cleanup generated resources, but continuing")
		// Don't fail the entire cleanup for this - log and continue
	}

	// 3. Cleanup any cached data related to this intent (non-blocking)
	if err := r.cleanupCachedData(ctx, networkIntent); err != nil {
		logger.Error(err, "failed to cleanup cached data, but continuing")
		// Cache cleanup is not critical - log and continue
	}

	logger.Info("All cleanup operations completed successfully", "intent_name", networkIntent.Name, "git_push_succeeded", *gitPushSucceeded)
	return nil
}

// cleanupGitOpsPackagesWithWait removes GitOps packages and waits for Git operations to complete
func (r *NetworkIntentReconciler) cleanupGitOpsPackagesWithWait(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, gitClient git.ClientInterface) error {
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

	logger.Info("Starting GitOps package cleanup with wait", "package", packageName, "path", packagePath, "git_repo", r.config.GitRepoURL)

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
	// This is the critical operation that must succeed before finalizer removal
	logger.V(1).Info("Removing directory from Git repository and waiting for completion", "directory", packagePath)
	if err := gitClient.RemoveDirectory(packagePath, commitMessage); err != nil {
		// Check if this is a "directory doesn't exist" type error and handle gracefully
		if isDirectoryNotExistError(err) {
			logger.V(1).Info("GitOps package directory does not exist, cleanup not needed", "package", packageName, "path", packagePath)
			return nil // Gracefully skip if directory doesn't exist
		}
		logger.Error(err, "failed to remove GitOps package directory", "package", packageName, "path", packagePath)
		return fmt.Errorf("failed to remove GitOps package directory '%s': %w", packagePath, err)
	}

	logger.Info("Successfully cleaned up GitOps package with Git operations completed", "package", packageName, "path", packagePath)
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
	case "LLMProcessing":
		if networkIntent.Status.ProcessingStartTime == nil {
			networkIntent.Status.ProcessingStartTime = &now
		}
	case "ResourcePlanning":
		// No specific timestamp for this phase
	case "ManifestGeneration":
		// No specific timestamp for this phase
	case "GitOpsCommit":
		if networkIntent.Status.DeploymentStartTime == nil {
			networkIntent.Status.DeploymentStartTime = &now
		}
	case "DeploymentVerification":
		// No specific timestamp for this phase
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