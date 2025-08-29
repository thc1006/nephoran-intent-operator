
package controllers



import (

	"context"

	"encoding/json"

	"fmt"

	"strings"

	"time"



	nephoranv1 "github.com/nephio-project/nephoran-intent-operator/api/v1"

	"github.com/nephio-project/nephoran-intent-operator/pkg/resilience"

	"github.com/nephio-project/nephoran-intent-operator/pkg/telecom"



	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"



	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/controller-runtime/pkg/log"

)



// LLMProcessor handles the LLM processing phase of NetworkIntent reconciliation.

type LLMProcessor struct {

	*NetworkIntentReconciler

}



// Ensure LLMProcessor implements the required interfaces.

var (

	_ LLMProcessorInterface = (*LLMProcessor)(nil)

	_ PhaseProcessor        = (*LLMProcessor)(nil)

)



// NewLLMProcessor creates a new LLM processor.

func NewLLMProcessor(r *NetworkIntentReconciler) *LLMProcessor {

	return &LLMProcessor{

		NetworkIntentReconciler: r,

	}

}



// ProcessLLMPhase implements Phase 1: LLM Processing with RAG context retrieval.

func (p *LLMProcessor) ProcessLLMPhase(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, processingCtx *ProcessingContext) (ctrl.Result, error) {

	logger := log.FromContext(ctx).WithValues("phase", "llm-processing")

	startTime := time.Now()



	processingCtx.CurrentPhase = PhaseLLMProcessing

	processingCtx.IntentType = p.extractIntentType(networkIntent.Spec.Intent)



	// Get retry count.

	retryCount := getNetworkIntentRetryCount(networkIntent, "llm-processing")

	if retryCount >= p.config.MaxRetries {

		err := fmt.Errorf("max retries (%d) exceeded for LLM processing", p.config.MaxRetries)

		// Set both Processed and Ready conditions to False.

		condition := metav1.Condition{

			Type:               "Processed",

			Status:             metav1.ConditionFalse,

			Reason:             "LLMProcessingFailedMaxRetries",

			Message:            fmt.Sprintf("Failed to process intent after %d retries", p.config.MaxRetries),

			LastTransitionTime: metav1.Now(),

		}

		updateCondition(&networkIntent.Status.Conditions, condition)



		// Set Ready condition to indicate failure.

		p.setReadyCondition(ctx, networkIntent, metav1.ConditionFalse, "LLMProcessingFailed", fmt.Sprintf("LLM processing failed after %d retries: %v", p.config.MaxRetries, err))

		return ctrl.Result{}, err

	}



	// Validate LLM client.

	llmClient := p.deps.GetLLMClient()

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



		// Set Ready condition to indicate configuration issue.

		p.setReadyCondition(ctx, networkIntent, metav1.ConditionFalse, "LLMClientNotConfigured", "LLM client is not properly configured")

		return ctrl.Result{}, err

	}



	// First, sanitize the user input to prevent injection attacks.

	sanitizedIntent, err := p.llmSanitizer.SanitizeInput(ctx, networkIntent.Spec.Intent)

	if err != nil {

		logger.Error(err, "Intent sanitization failed - potential security threat detected")

		condition := metav1.Condition{

			Type:               "Processed",

			Status:             metav1.ConditionFalse,

			Reason:             "IntentSanitizationFailed",

			Message:            fmt.Sprintf("Intent rejected due to security concerns: %v", err),

			LastTransitionTime: metav1.Now(),

		}

		updateCondition(&networkIntent.Status.Conditions, condition)

		p.setReadyCondition(ctx, networkIntent, metav1.ConditionFalse, "SecurityValidationFailed", fmt.Sprintf("Intent failed security validation: %v", err))

		p.recordFailureEvent(networkIntent, "SecurityValidationFailed", err.Error())

		return ctrl.Result{}, fmt.Errorf("intent failed security validation: %w", err)

	}



	// Build telecom-enhanced prompt with 3GPP context using sanitized input.

	enhancedPrompt, err := p.buildTelecomEnhancedPrompt(ctx, sanitizedIntent, processingCtx)

	if err != nil {

		logger.Error(err, "failed to build telecom-enhanced prompt")

		setNetworkIntentRetryCount(networkIntent, "llm-processing", retryCount+1)

		p.recordFailureEvent(networkIntent, "PromptEnhancementFailed", err.Error())

		return ctrl.Result{RequeueAfter: p.config.RetryDelay}, nil

	}



	// Build secure prompt with proper boundaries.

	systemPrompt := "You are a secure telecommunications network orchestration expert. Generate only valid JSON for network function deployments."

	securePrompt := p.llmSanitizer.BuildSecurePrompt(systemPrompt, enhancedPrompt)



	// Process with LLM using secure prompt with circuit breaker and timeout.

	logger.Info("Processing intent with LLM", "retry_count", retryCount+1, "enhanced_prompt_length", len(securePrompt))



	// Execute LLM call through circuit breaker with fallback.

	var processedResult string

	result, err := p.llmCircuitBreaker.ExecuteWithFallback(ctx,

		// Primary operation: LLM processing with timeout.

		func(ctx context.Context) (interface{}, error) {

			return p.timeoutManager.ExecuteWithTimeout(ctx, resilience.OperationTypeLLM,

				func(timeoutCtx context.Context) (interface{}, error) {

					return llmClient.ProcessIntent(timeoutCtx, securePrompt)

				})

		},

		// Fallback operation when circuit is open.

		func(ctx context.Context, circuitErr error) (interface{}, error) {

			logger.Info("LLM circuit breaker is open, using fallback response", "error", circuitErr)



			// Provide a basic fallback response for simple intents.

			fallbackResponse := p.generateFallbackResponse(sanitizedIntent)

			if fallbackResponse != "" {

				p.recordEvent(networkIntent, "Normal", "LLMFallbackUsed", "Circuit breaker open, using fallback response")

				return fallbackResponse, nil

			}



			// If no fallback possible, return circuit breaker error.

			return nil, fmt.Errorf("LLM service unavailable and no fallback possible: %w", circuitErr)

		})



	if err != nil {

		// Check if this is a circuit breaker error.

		if p.llmCircuitBreaker.IsOpen() {

			logger.Error(err, "LLM circuit breaker is open", "retry", retryCount+1)

			condition := metav1.Condition{

				Type:               "Processed",

				Status:             metav1.ConditionFalse,

				Reason:             "LLMServiceUnavailable",

				Message:            fmt.Sprintf("LLM service unavailable (circuit breaker open): %v", err),

				LastTransitionTime: metav1.Now(),

			}

			updateCondition(&networkIntent.Status.Conditions, condition)

			p.setReadyCondition(ctx, networkIntent, metav1.ConditionFalse, "LLMServiceUnavailable", fmt.Sprintf("LLM service is currently unavailable: %v", err))

			p.recordFailureEvent(networkIntent, "LLMServiceUnavailable", err.Error())



			// For circuit breaker failures, use longer backoff.

			return ctrl.Result{RequeueAfter: 2 * p.config.RetryDelay}, err

		}



		// Handle other errors (timeouts, processing failures, etc.).

		processedResult = ""

	} else {

		processedResult = result.(string)

	}



	if err != nil {

		logger.Error(err, "LLM processing failed", "retry", retryCount+1)

		setNetworkIntentRetryCount(networkIntent, "llm-processing", retryCount+1)



		condition := metav1.Condition{

			Type:               "Processed",

			Status:             metav1.ConditionFalse,

			Reason:             "LLMProcessingRetrying",

			Message:            fmt.Sprintf("LLM processing failed (attempt %d/%d): %v", retryCount+1, p.config.MaxRetries, err),

			LastTransitionTime: metav1.Now(),

		}

		updateCondition(&networkIntent.Status.Conditions, condition)



		// Set Ready condition to False while retrying.

		p.setReadyCondition(ctx, networkIntent, metav1.ConditionFalse, "LLMProcessingRetrying", fmt.Sprintf("LLM processing failed, retrying (attempt %d/%d): %v", retryCount+1, p.config.MaxRetries, err))



		p.recordFailureEvent(networkIntent, "LLMProcessingRetry", fmt.Sprintf("attempt %d/%d failed: %v", retryCount+1, p.config.MaxRetries, err))



		// Use exponential backoff with jitter for LLM operations.

		backoffDelay := calculateExponentialBackoffForOperation(retryCount, "llm-processing")

		logger.V(1).Info("Scheduling LLM processing retry with exponential backoff",

			"delay", backoffDelay,

			"attempt", retryCount+1,

			"max_retries", p.config.MaxRetries)



		return ctrl.Result{RequeueAfter: backoffDelay}, nil

	}



	// Validate LLM output for malicious content before parsing.

	validatedOutput, err := p.llmSanitizer.ValidateOutput(ctx, processedResult)

	if err != nil {

		logger.Error(err, "LLM output validation failed - potential malicious content detected")

		condition := metav1.Condition{

			Type:               "Processed",

			Status:             metav1.ConditionFalse,

			Reason:             "LLMOutputValidationFailed",

			Message:            fmt.Sprintf("LLM output rejected due to security concerns: %v", err),

			LastTransitionTime: metav1.Now(),

		}

		updateCondition(&networkIntent.Status.Conditions, condition)

		p.setReadyCondition(ctx, networkIntent, metav1.ConditionFalse, "OutputSecurityValidationFailed", fmt.Sprintf("LLM output failed security validation: %v", err))

		p.recordFailureEvent(networkIntent, "OutputSecurityValidationFailed", err.Error())

		return ctrl.Result{}, fmt.Errorf("LLM output failed security validation: %w", err)

	}



	// Parse and validate LLM response.

	var parameters map[string]interface{}

	if err := json.Unmarshal([]byte(validatedOutput), &parameters); err != nil {

		logger.Error(err, "failed to parse LLM response as JSON", "response", validatedOutput)

		setNetworkIntentRetryCount(networkIntent, "llm-processing", retryCount+1)



		condition := metav1.Condition{

			Type:               "Processed",

			Status:             metav1.ConditionFalse,

			Reason:             "LLMResponseParsingFailed",

			Message:            fmt.Sprintf("Failed to parse LLM response as JSON: %v", err),

			LastTransitionTime: metav1.Now(),

		}

		updateCondition(&networkIntent.Status.Conditions, condition)



		// Set Ready condition to False due to parsing error.

		p.setReadyCondition(ctx, networkIntent, metav1.ConditionFalse, "LLMResponseParsingFailed", fmt.Sprintf("Failed to parse LLM response as JSON: %v", err))



		// Use exponential backoff for parsing failures too.

		backoffDelay := calculateExponentialBackoffForOperation(retryCount, "llm-processing")

		return ctrl.Result{RequeueAfter: backoffDelay}, nil

	}



	// Extract structured entities from LLM response.

	processingCtx.ExtractedEntities = parameters



	// Update NetworkIntent with processed parameters.

	parametersRaw, err := json.Marshal(parameters)

	if err != nil {

		return ctrl.Result{RequeueAfter: p.config.RetryDelay}, fmt.Errorf("failed to marshal parameters: %w", err)

	}



	// Store parameters in status extensions.

	if networkIntent.Status.Extensions == nil {

		networkIntent.Status.Extensions = make(map[string]runtime.RawExtension)

	}

	networkIntent.Status.Extensions["parameters"] = runtime.RawExtension{Raw: parametersRaw}

	if err := p.safeStatusUpdate(ctx, networkIntent); err != nil {

		return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update NetworkIntent with parameters: %w", err)

	}



	// Clear retry count and update success condition.

	clearNetworkIntentRetryCount(networkIntent, "llm-processing")

	processingDuration := time.Since(startTime)

	now := metav1.Now()

	// Store processing completion time in Extensions.

	processingTime, _ := json.Marshal(now.Format(time.RFC3339))

	networkIntent.Status.Extensions["processingCompletionTime"] = runtime.RawExtension{Raw: processingTime}



	condition := metav1.Condition{

		Type:               "Processed",

		Status:             metav1.ConditionTrue,

		Reason:             "LLMProcessingSucceeded",

		Message:            fmt.Sprintf("Intent successfully processed by LLM with %d parameters in %.2fs", len(parameters), processingDuration.Seconds()),

		LastTransitionTime: now,

	}

	updateCondition(&networkIntent.Status.Conditions, condition)



	// Set Ready condition to True for LLM processing success (this phase only).

	// Note: Overall Ready status will be managed by the full pipeline.

	if err := p.safeStatusUpdate(ctx, networkIntent); err != nil {

		return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("failed to update status: %w", err)

	}



	// Record metrics.

	if metricsCollector := p.deps.GetMetricsCollector(); metricsCollector != nil {

		metricsCollector.RecordLLMRequest("gpt-4o-mini", "success", processingDuration, len(processedResult))

	}



	processingCtx.Metrics["llm_processing_duration"] = processingDuration.Seconds()

	processingCtx.Metrics["llm_parameters_count"] = float64(len(parameters))



	p.recordEvent(networkIntent, "Normal", "LLMProcessingSucceeded", fmt.Sprintf("Intent processed with %d parameters", len(parameters)))

	logger.Info("LLM processing phase completed successfully", "duration", processingDuration, "parameters_count", len(parameters))



	return ctrl.Result{}, nil

}



// buildTelecomEnhancedPrompt builds a telecom-enhanced prompt with 3GPP context and O-RAN knowledge.

func (p *LLMProcessor) buildTelecomEnhancedPrompt(ctx context.Context, intent string, processingCtx *ProcessingContext) (string, error) {

	logger := log.FromContext(ctx).WithValues("function", "buildTelecomEnhancedPrompt")

	_ = logger // Suppress unused warning



	// Get telecom knowledge base.

	kb := p.deps.GetTelecomKnowledgeBase()

	if kb == nil {

		return "", fmt.Errorf("telecom knowledge base not available")

	}



	// Extract telecom entities and context from intent.

	telecomContext := p.extractTelecomContext(intent, kb)

	processingCtx.TelecomContext = telecomContext



	// Build enhanced prompt with telecom knowledge.

	var promptBuilder strings.Builder



	promptBuilder.WriteString("You are a telecommunications network orchestration expert with deep knowledge of 5G Core (5GC), ")

	promptBuilder.WriteString("O-RAN architecture, 3GPP specifications, and network function deployment patterns.\n\n")



	// Add 3GPP and O-RAN context.

	promptBuilder.WriteString("## TELECOMMUNICATIONS CONTEXT\n")

	promptBuilder.WriteString("Based on 3GPP Release 17 and O-RAN Alliance specifications:\n\n")



	// Add relevant network function knowledge.

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



	// Add slice type information if detected.

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



	// Add deployment pattern context.

	if pattern, ok := telecomContext["detected_deployment_pattern"].(string); ok && pattern != "" {

		if dp, exists := kb.GetDeploymentPattern(pattern); exists {

			promptBuilder.WriteString(fmt.Sprintf("### Deployment Pattern: %s\n", dp.Name))

			promptBuilder.WriteString(fmt.Sprintf("- Description: %s\n", dp.Description))

			promptBuilder.WriteString(fmt.Sprintf("- Use Case: %s\n", dp.UseCase))

			promptBuilder.WriteString("- Required Network Functions:\n")

			// Note: RequiredNetworkFunctions method not available, using placeholder.

			promptBuilder.WriteString("  - Network functions from deployment pattern\n")

			promptBuilder.WriteString("\n")

		}

	}



	// Add the user intent.

	promptBuilder.WriteString("## USER INTENT\n")

	promptBuilder.WriteString(fmt.Sprintf("Process this telecommunications network intent: \"%s\"\n\n", intent))



	// Add generation instructions.

	promptBuilder.WriteString("## INSTRUCTIONS\n")

	promptBuilder.WriteString("Generate a JSON response that includes:\n")

	promptBuilder.WriteString("1. **network_functions**: Array of required network functions with configurations\n")

	promptBuilder.WriteString("2. **deployment_pattern**: Recommended deployment pattern\n")

	promptBuilder.WriteString("3. **resource_requirements**: CPU, memory, storage requirements\n")

	promptBuilder.WriteString("4. **qos_profile**: Quality of Service profile\n")

	promptBuilder.WriteString("5. **security_policies**: Required security configurations\n")

	promptBuilder.WriteString("6. **monitoring**: Monitoring and alerting specifications\n\n")



	promptBuilder.WriteString("Ensure all recommendations comply with 3GPP standards and O-RAN specifications. ")

	promptBuilder.WriteString("Use appropriate resource sizing based on the deployment pattern and expected load.\n")



	return promptBuilder.String(), nil

}



// extractTelecomContext extracts telecommunications-specific entities and context from the intent.

func (p *LLMProcessor) extractTelecomContext(intent string, kb *telecom.TelecomKnowledgeBase) map[string]interface{} {

	context := make(map[string]interface{})

	intentLower := strings.ToLower(intent)



	// Detect network functions.

	var detectedNFs []string

	// Note: GetAllNetworkFunctions method not available, using placeholder.

	type placeholder struct {

		Name string

		Type string

	}

	networkFunctions := []placeholder{{Name: "AMF", Type: "amf"}, {Name: "SMF", Type: "smf"}, {Name: "UPF", Type: "upf"}}

	for _, nf := range networkFunctions {

		nfNameLower := strings.ToLower(nf.Name)

		nfTypeLower := strings.ToLower(nf.Type)

		if strings.Contains(intentLower, nfNameLower) || strings.Contains(intentLower, nfTypeLower) {

			detectedNFs = append(detectedNFs, nf.Type)

		}

	}

	context["detected_network_functions"] = detectedNFs



	// Detect slice types.

	// Note: GetAllSliceTypes method not available, using placeholder.

	type slicePlaceholder struct {

		Description string

		UseCase     string

		SST         int

	}

	sliceTypes := []slicePlaceholder{{Description: "eMBB", UseCase: "enhanced mobile broadband", SST: 1}, {Description: "URLLC", UseCase: "ultra-reliable low latency", SST: 2}}

	for _, slice := range sliceTypes {

		sliceNameLower := strings.ToLower(slice.Description)

		if strings.Contains(intentLower, sliceNameLower) ||

			strings.Contains(intentLower, strings.ToLower(slice.UseCase)) {

			context["detected_slice_type"] = slice.Description

			context["detected_sst"] = slice.SST

			break

		}

	}



	// Detect deployment patterns.

	// Note: GetAllDeploymentPatterns method not available, using placeholder.

	type patternPlaceholder struct {

		Name    string

		UseCase string

	}

	deploymentPatterns := []patternPlaceholder{{Name: "high-availability", UseCase: "high availability deployment"}}

	for _, pattern := range deploymentPatterns {

		patternNameLower := strings.ToLower(pattern.Name)

		patternUseCaseLower := strings.ToLower(pattern.UseCase)

		if strings.Contains(intentLower, patternNameLower) ||

			strings.Contains(intentLower, patternUseCaseLower) {

			context["detected_deployment_pattern"] = pattern.Name

			break

		}

	}



	// Detect performance requirements.

	if strings.Contains(intentLower, "high performance") || strings.Contains(intentLower, "high throughput") {

		context["performance_requirement"] = "high"

	} else if strings.Contains(intentLower, "low latency") || strings.Contains(intentLower, "ultra") {

		context["performance_requirement"] = "ultra_low_latency"

	} else if strings.Contains(intentLower, "massive") || strings.Contains(intentLower, "iot") {

		context["performance_requirement"] = "massive_iot"

	}



	// Detect scale requirements.

	if strings.Contains(intentLower, "scale") || strings.Contains(intentLower, "auto") {

		context["auto_scaling"] = true

	}



	// Detect environment.

	if strings.Contains(intentLower, "production") || strings.Contains(intentLower, "prod") {

		context["environment"] = "production"

	} else if strings.Contains(intentLower, "development") || strings.Contains(intentLower, "dev") {

		context["environment"] = "development"

	} else if strings.Contains(intentLower, "staging") || strings.Contains(intentLower, "stage") {

		context["environment"] = "staging"

	}



	// Detect security requirements.

	if strings.Contains(intentLower, "secure") || strings.Contains(intentLower, "security") {

		context["security_required"] = true

	}



	return context

}



// Interface implementation methods.



// ProcessPhase implements PhaseProcessor interface.

func (p *LLMProcessor) ProcessPhase(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, processingCtx *ProcessingContext) (ctrl.Result, error) {

	return p.ProcessLLMPhase(ctx, networkIntent, processingCtx)

}



// GetPhaseName implements PhaseProcessor interface.

func (p *LLMProcessor) GetPhaseName() ProcessingPhase {

	return PhaseLLMProcessing

}



// IsPhaseComplete implements PhaseProcessor interface.

func (p *LLMProcessor) IsPhaseComplete(networkIntent *nephoranv1.NetworkIntent) bool {

	return isConditionTrue(networkIntent.Status.Conditions, "Processed")

}



// BuildTelecomEnhancedPrompt implements LLMProcessorInterface (expose existing method).

func (p *LLMProcessor) BuildTelecomEnhancedPrompt(ctx context.Context, intent string, processingCtx *ProcessingContext) (string, error) {

	return p.buildTelecomEnhancedPrompt(ctx, intent, processingCtx)

}



// ExtractTelecomContext implements LLMProcessorInterface (expose existing method).

func (p *LLMProcessor) ExtractTelecomContext(intent string) map[string]interface{} {

	kb := p.deps.GetTelecomKnowledgeBase()

	if kb == nil {

		return make(map[string]interface{})

	}

	return p.extractTelecomContext(intent, kb)

}

