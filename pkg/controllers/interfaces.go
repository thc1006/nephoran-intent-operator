
package controllers



import (

	"context"



	nephoranv1 "github.com/nephio-project/nephoran-intent-operator/api/v1"



	ctrl "sigs.k8s.io/controller-runtime"

)



// PhaseProcessor defines the interface for processing different phases of NetworkIntent reconciliation.

type PhaseProcessor interface {

	// ProcessPhase processes a specific phase and returns the result and any error.

	ProcessPhase(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, processingCtx *ProcessingContext) (ctrl.Result, error)



	// GetPhaseName returns the name of the phase this processor handles.

	GetPhaseName() ProcessingPhase



	// IsPhaseComplete checks if the phase has been completed successfully.

	IsPhaseComplete(networkIntent *nephoranv1.NetworkIntent) bool

}



// LLMProcessorInterface defines the contract for LLM processing operations.

type LLMProcessorInterface interface {

	PhaseProcessor



	// ProcessLLMPhase handles the LLM processing phase specifically.

	ProcessLLMPhase(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, processingCtx *ProcessingContext) (ctrl.Result, error)



	// BuildTelecomEnhancedPrompt builds an enhanced prompt with telecom context.

	BuildTelecomEnhancedPrompt(ctx context.Context, intent string, processingCtx *ProcessingContext) (string, error)



	// ExtractTelecomContext extracts telecommunications-specific context from intent.

	ExtractTelecomContext(intent string) map[string]interface{}

}



// ResourcePlannerInterface defines the contract for resource planning operations.

type ResourcePlannerInterface interface {

	PhaseProcessor



	// PlanResources handles the resource planning phase.

	PlanResources(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, processingCtx *ProcessingContext) (ctrl.Result, error)



	// GenerateManifests handles the manifest generation phase.

	GenerateManifests(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, processingCtx *ProcessingContext) (ctrl.Result, error)



	// PlanNetworkFunction creates a planned network function from specifications.

	PlanNetworkFunction(nfName string, llmParams, telecomContext map[string]interface{}) (*PlannedNetworkFunction, error)



	// CalculateEstimatedCost calculates the estimated cost for planned resources.

	CalculateEstimatedCost(plan *ResourcePlan) (float64, error)

}



// GitOpsHandlerInterface defines the contract for GitOps operations.

type GitOpsHandlerInterface interface {

	PhaseProcessor



	// CommitToGitOps handles the GitOps commit phase.

	CommitToGitOps(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, processingCtx *ProcessingContext) (ctrl.Result, error)



	// VerifyDeployment handles the deployment verification phase.

	VerifyDeployment(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, processingCtx *ProcessingContext) (ctrl.Result, error)



	// CleanupGitOpsResources removes GitOps resources for a NetworkIntent.

	CleanupGitOpsResources(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) error

}



// ReconcilerInterface defines the main reconciler contract.

type ReconcilerInterface interface {

	// Reconcile is the main reconciliation entry point.

	Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)



	// ExecuteProcessingPipeline orchestrates the multi-phase processing.

	ExecuteProcessingPipeline(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, processingCtx *ProcessingContext) (ctrl.Result, error)



	// IsProcessingComplete checks if all processing phases are complete.

	IsProcessingComplete(networkIntent *nephoranv1.NetworkIntent) bool

}



// ControllerInterface defines the overall controller contract.

type ControllerInterface interface {

	ReconcilerInterface



	// SetupWithManager configures the controller with the manager.

	SetupWithManager(mgr ctrl.Manager) error



	// GetConfig returns the controller configuration.

	GetConfig() *Config



	// GetDependencies returns the controller dependencies.

	GetDependencies() Dependencies

}



// StatusManagerInterface defines the contract for managing NetworkIntent status.

type StatusManagerInterface interface {

	// UpdatePhase updates the processing phase.

	UpdatePhase(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, phase string) error



	// SetReadyCondition sets the Ready condition.

	SetReadyCondition(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, status, reason, message string) error



	// UpdateCondition updates a specific condition.

	UpdateCondition(networkIntent *nephoranv1.NetworkIntent, conditionType, status, reason, message string)



	// IsConditionTrue checks if a condition is true.

	IsConditionTrue(networkIntent *nephoranv1.NetworkIntent, conditionType string) bool

}



// EventManagerInterface defines the contract for event management.

type EventManagerInterface interface {

	// RecordEvent records a Kubernetes event.

	RecordEvent(networkIntent *nephoranv1.NetworkIntent, eventType, reason, message string)



	// RecordFailureEvent records a failure event.

	RecordFailureEvent(networkIntent *nephoranv1.NetworkIntent, reason, message string)



	// RecordSuccessEvent records a success event.

	RecordSuccessEvent(networkIntent *nephoranv1.NetworkIntent, reason, message string)

}



// RetryManagerInterface defines the contract for retry management.

type RetryManagerInterface interface {

	// GetRetryCount gets the current retry count for an operation.

	GetRetryCount(networkIntent *nephoranv1.NetworkIntent, operation string) int



	// SetRetryCount sets the retry count for an operation.

	SetRetryCount(networkIntent *nephoranv1.NetworkIntent, operation string, count int)



	// ClearRetryCount clears the retry count for an operation.

	ClearRetryCount(networkIntent *nephoranv1.NetworkIntent, operation string)



	// IsRetryLimitExceeded checks if retry limit has been exceeded.

	IsRetryLimitExceeded(networkIntent *nephoranv1.NetworkIntent, operation string, maxRetries int) bool



	// CalculateBackoffDelay calculates the exponential backoff delay.

	CalculateBackoffDelay(retryCount int, operation string) ctrl.Result

}



// SecurityManagerInterface defines the contract for security operations.

type SecurityManagerInterface interface {

	// SanitizeInput sanitizes user input for security.

	SanitizeInput(ctx context.Context, input string) (string, error)



	// ValidateOutput validates LLM output for security.

	ValidateOutput(ctx context.Context, output string) (string, error)



	// BuildSecurePrompt builds a secure prompt with proper boundaries.

	BuildSecurePrompt(systemPrompt, userPrompt string) string

}



// MetricsManagerInterface defines the contract for metrics collection.

type MetricsManagerInterface interface {

	// RecordProcessingMetrics records metrics for a processing phase.

	RecordProcessingMetrics(phase ProcessingPhase, duration float64, success bool)



	// RecordLLMMetrics records LLM-specific metrics.

	RecordLLMMetrics(model string, tokenCount int, duration float64, success bool)



	// RecordGitOpsMetrics records GitOps-specific metrics.

	RecordGitOpsMetrics(operation string, duration float64, success bool)



	// UpdateNetworkIntentStatus updates the status metrics.

	UpdateNetworkIntentStatus(name, namespace, intentType, status string)

}



// ValidationInterface defines the contract for validation operations.

type ValidationInterface interface {

	// ValidateNetworkIntent validates a NetworkIntent resource.

	ValidateNetworkIntent(networkIntent *nephoranv1.NetworkIntent) error



	// ValidateProcessingContext validates a ProcessingContext.

	ValidateProcessingContext(processingCtx *ProcessingContext) error



	// ValidateResourcePlan validates a ResourcePlan.

	ValidateResourcePlan(plan *ResourcePlan) error



	// ValidateConfiguration validates controller configuration.

	ValidateConfiguration(config *Config) error

}



// CleanupManagerInterface defines the contract for cleanup operations.

type CleanupManagerInterface interface {

	// PerformCleanup performs cleanup for a deleted NetworkIntent.

	PerformCleanup(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) error



	// CleanupGeneratedResources cleans up generated Kubernetes resources.

	CleanupGeneratedResources(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) error



	// CleanupCachedData cleans up cached data.

	CleanupCachedData(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) error

}



// ContextManagerInterface defines the contract for processing context management.

type ContextManagerInterface interface {

	// InitializeProcessingContext initializes a new processing context.

	InitializeProcessingContext(networkIntent *nephoranv1.NetworkIntent) *ProcessingContext



	// UpdateProcessingContext updates the processing context.

	UpdateProcessingContext(processingCtx *ProcessingContext, phase ProcessingPhase, data map[string]interface{})



	// ValidateProcessingContext validates the processing context.

	ValidateProcessingContext(processingCtx *ProcessingContext, expectedPhase ProcessingPhase) error



	// SerializeProcessingContext serializes the processing context for storage.

	SerializeProcessingContext(processingCtx *ProcessingContext) (string, error)



	// DeserializeProcessingContext deserializes the processing context from storage.

	DeserializeProcessingContext(data string) (*ProcessingContext, error)

}



// HealthCheckInterface defines the contract for health checking operations.

type HealthCheckInterface interface {

	// CheckHealth checks the overall health of the controller.

	CheckHealth(ctx context.Context) error



	// CheckDependencies checks the health of dependencies.

	CheckDependencies(ctx context.Context) error



	// CheckLLMHealth checks the health of LLM services.

	CheckLLMHealth(ctx context.Context) error



	// CheckGitHealth checks the health of Git services.

	CheckGitHealth(ctx context.Context) error

}



// Interface compliance checks (compile-time verification).

var (

	_ PhaseProcessor           = (*LLMProcessor)(nil)

	_ PhaseProcessor           = (*ResourcePlanner)(nil)

	_ PhaseProcessor           = (*GitOpsHandler)(nil)

	_ LLMProcessorInterface    = (*LLMProcessor)(nil)

	_ ResourcePlannerInterface = (*ResourcePlanner)(nil)

	_ GitOpsHandlerInterface   = (*GitOpsHandler)(nil)

	_ ReconcilerInterface      = (*Reconciler)(nil)

)

