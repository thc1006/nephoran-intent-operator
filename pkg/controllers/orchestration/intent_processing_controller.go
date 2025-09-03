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

package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/interfaces"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
)

// IntentProcessingController reconciles IntentProcessing objects.

type IntentProcessingController struct {
	client.Client

	Scheme *runtime.Scheme

	Recorder record.EventRecorder

	Logger logr.Logger

	// Services.

	LLMService *llm.Client

	RAGService *RAGService

	// Configuration.

	Config *IntentProcessingConfig

	// Event bus for coordination.

	EventBus *EventBus

	// Metrics.

	MetricsCollector *MetricsCollector
}

// IntentProcessingConfig contains configuration for the controller.

type IntentProcessingConfig struct {
	MaxConcurrentProcessing int `json:"maxConcurrentProcessing"`

	DefaultTimeout time.Duration `json:"defaultTimeout"`

	MaxRetries int `json:"maxRetries"`

	RetryBackoff time.Duration `json:"retryBackoff"`

	QualityThreshold float64 `json:"qualityThreshold"`

	ValidationEnabled bool `json:"validationEnabled"`

	// LLM Configuration.

	LLMEndpoint string `json:"llmEndpoint"`

	// Circuit Breaker Configuration.

	CircuitBreakerEnabled bool `json:"circuitBreakerEnabled"`

	FailureThreshold int `json:"failureThreshold"`

	RecoveryTimeout time.Duration `json:"recoveryTimeout"`

	// Cache Configuration.

	CacheEnabled bool `json:"cacheEnabled"`

	CacheTTL time.Duration `json:"cacheTTL"`

	// Streaming Configuration.

	StreamingEnabled bool `json:"streamingEnabled"`
}

// NewIntentProcessingController creates a new IntentProcessingController.

// RAGService represents a stub for RAG service functionality.

type RAGService struct {
	// Stub implementation.
}

// RAGRequest represents a request to the RAG service.

type RAGRequest struct {
	Query string

	MaxResults int

	MinConfidence float64

	UseHybridSearch bool

	RetrievalThreshold float64

	EnableContextBuilder bool
}

// RAGResponse represents a response from the RAG service.

type RAGResponse struct {
	Context map[string]interface{}

	Metrics *nephoranv1.RAGMetrics

	SourceDocuments []interface{}

	Metadata map[string]interface{}

	RetrievalTime int64

	Confidence float32
}

// ProcessQuery processes a query using the RAG service (stub implementation).

func (rs *RAGService) ProcessQuery(ctx context.Context, request *RAGRequest) (*RAGResponse, error) {
	// Stub implementation - return a generic response.

	return &RAGResponse{
		Context: map[string]interface{}{
			"context_summary": "Mock context for intent: " + request.Query,
		},

		Metrics: &nephoranv1.RAGMetrics{},

		SourceDocuments: []interface{}{},

		Metadata: map[string]interface{}{},

		RetrievalTime: 100,

		Confidence: 0.8,
	}, nil
}

// NewIntentProcessingController performs newintentprocessingcontroller operation.

func NewIntentProcessingController(
	client client.Client,

	scheme *runtime.Scheme,

	recorder record.EventRecorder,

	llmService *llm.Client,

	ragService *RAGService,

	eventBus *EventBus,

	config *IntentProcessingConfig,
) *IntentProcessingController {
	return &IntentProcessingController{
		Client: client,

		Scheme: scheme,

		Recorder: recorder,

		Logger: log.Log.WithName("intent-processing-controller"),

		LLMService: llmService,

		RAGService: ragService,

		EventBus: eventBus,

		Config: config,

		MetricsCollector: NewMetricsCollector(),
	}
}

// Reconcile handles IntentProcessing resources.

func (r *IntentProcessingController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Logger.WithValues("intentprocessing", req.NamespacedName)

	// Fetch the IntentProcessing instance.

	intentProcessing := &nephoranv1.IntentProcessing{}

	if err := r.Get(ctx, req.NamespacedName, intentProcessing); err != nil {

		if apierrors.IsNotFound(err) {

			log.Info("IntentProcessing resource not found, ignoring")

			return ctrl.Result{}, nil

		}

		log.Error(err, "Failed to get IntentProcessing")

		return ctrl.Result{}, err

	}

	// Handle deletion.

	if intentProcessing.DeletionTimestamp != nil {
		return r.handleDeletion(ctx, intentProcessing)
	}

	// Add finalizer if not present.

	if !controllerutil.ContainsFinalizer(intentProcessing, "intentprocessing.nephoran.com/finalizer") {

		controllerutil.AddFinalizer(intentProcessing, "intentprocessing.nephoran.com/finalizer")

		return ctrl.Result{}, r.Update(ctx, intentProcessing)

	}

	// Process the intent.

	return r.processIntent(ctx, intentProcessing)
}

// processIntent processes the natural language intent.

func (r *IntentProcessingController) processIntent(ctx context.Context, intentProcessing *nephoranv1.IntentProcessing) (ctrl.Result, error) {
	log := r.Logger.WithValues("intentprocessing", intentProcessing.Name, "namespace", intentProcessing.Namespace)

	// Check if processing is already complete.

	if intentProcessing.IsProcessingComplete() {

		log.V(1).Info("Intent processing already complete")

		return ctrl.Result{}, nil

	}

	// Check if processing failed and can retry.

	if intentProcessing.IsProcessingFailed() && !intentProcessing.CanRetry() {

		log.Info("Intent processing failed and cannot retry")

		return ctrl.Result{}, nil

	}

	// Record processing start.

	if intentProcessing.Status.ProcessingStartTime == nil {

		now := metav1.Now()

		intentProcessing.Status.ProcessingStartTime = &now

		intentProcessing.Status.Phase = nephoranv1.IntentProcessingPhaseInProgress

		if err := r.updateStatus(ctx, intentProcessing); err != nil {
			return ctrl.Result{}, err
		}

		r.MetricsCollector.RecordPhaseStart(interfaces.PhaseLLMProcessing, string(intentProcessing.UID))

	}

	// Publish processing start event.

	if err := r.EventBus.PublishPhaseEvent(ctx, interfaces.PhaseLLMProcessing, EventLLMProcessingStarted,

		string(intentProcessing.UID), false, map[string]interface{}{}); err != nil {
		log.Error(err, "Failed to publish processing start event")
	}

	// Create processing context with timeout.

	processingCtx, cancel := context.WithTimeout(ctx, intentProcessing.GetProcessingTimeout())

	defer cancel()

	// Execute LLM processing.

	result, err := r.executeLLMProcessing(processingCtx, intentProcessing)
	if err != nil {
		return r.handleProcessingError(ctx, intentProcessing, err)
	}

	// Update status with results.

	return r.handleProcessingSuccess(ctx, intentProcessing, result)
}

// executeLLMProcessing performs the actual LLM processing.

func (r *IntentProcessingController) executeLLMProcessing(ctx context.Context, intentProcessing *nephoranv1.IntentProcessing) (*LLMProcessingResult, error) {
	log := r.Logger.WithValues("intentprocessing", intentProcessing.Name)

	// Prepare LLM request.

	request := &llm.ProcessingRequest{
		Intent: intentProcessing.Spec.OriginalIntent,
	}

	// Configure LLM parameters.

	contextMap := make(map[string]interface{})

	if intentProcessing.Spec.ProcessingConfiguration != nil {

		config := intentProcessing.Spec.ProcessingConfiguration

		// Add provider info to context map.

		contextMap["provider"] = string(config.Provider)

		contextMap["intentType"] = string(intentProcessing.Spec.ParentIntentRef.Kind)

		contextMap["priority"] = string(intentProcessing.Spec.Priority)

		request.Model = config.Model

		// Temperature and MaxTokens are not available in ProcessingRequest struct
		// These parameters may be handled internally by the LLM service

		// SystemPrompt field doesn't exist in ProcessingRequest.

	}

	// Enhance with RAG if enabled.

	if intentProcessing.ShouldEnableRAG() && r.RAGService != nil {

		log.Info("Enhancing intent with RAG")

		enhancedContext, ragMetrics, err := r.enhanceWithRAG(ctx, intentProcessing.Spec.OriginalIntent, intentProcessing.Spec.ProcessingConfiguration)

		if err != nil {
			log.Error(err, "Failed to enhance with RAG, continuing without enhancement")

			// Continue without RAG enhancement rather than failing.
		} else {

			// Merge enhanced context with existing context map.

			for k, v := range enhancedContext {
				contextMap[k] = v
			}

			// Store RAG metrics in context for status update.

			contextMap["ragMetrics"] = ragMetrics

		}

	}

	// Set context map directly to ProcessingRequest.Context field.

	if len(contextMap) > 0 {
		request.Context = convertInterfaceMapToString(contextMap)
	}

	// Execute LLM processing.

	log.Info("Executing LLM processing", "model", request.Model)

	response, err := r.LLMService.ProcessIntent(ctx, request.Intent)
	if err != nil {
		return nil, fmt.Errorf("LLM processing failed: %w", err)
	}

	// Convert string response to ProcessingResponse for validation.

	processingResp := &llm.ProcessingResponse{
		Response: response,

		Confidence: 0.9, // Default confidence for now

	}

	// Validate response quality.

	qualityScore, validationErrors := r.validateResponse(processingResp)

	if qualityScore < r.Config.QualityThreshold {
		return nil, fmt.Errorf("response quality score %.2f below threshold %.2f", qualityScore, r.Config.QualityThreshold)
	}

	// Create processing result.

	result := &LLMProcessingResult{
		Response: processingResp,

		QualityScore: qualityScore,

		ValidationErrors: validationErrors,

		TokenUsage: nil, // No token usage info from string response

		RAGMetrics: extractRAGMetricsFromContext(convertStringMapToInterface(request.Context)),
	}

	// Extract structured parameters.

	processedParams, err := r.extractProcessedParameters(processingResp)

	if err != nil {
		log.Error(err, "Failed to extract processed parameters")

		// Continue with raw response.
	} else {
		result.ProcessedParameters = processedParams
	}

	// Extract telecommunications entities.

	entities, err := r.extractTelecomEntities(processingResp)

	if err != nil {
		log.Error(err, "Failed to extract telecom entities")
	} else {
		result.ExtractedEntities = entities
	}

	return result, nil
}

// enhanceWithRAG enhances the intent with RAG context.

func (r *IntentProcessingController) enhanceWithRAG(ctx context.Context, intent string, config *nephoranv1.ProcessingConfig) (map[string]interface{}, *nephoranv1.RAGMetrics, error) {
	// Prepare RAG request.

	request := &RAGRequest{
		Query: intent,
	}

	// Configure RAG parameters.

	if config != nil && config.RAGConfiguration != nil {

		ragConfig := config.RAGConfiguration

		if ragConfig.MaxDocuments != nil {
			request.MaxResults = int(*ragConfig.MaxDocuments)
		}

		if ragConfig.RetrievalThreshold != nil {
			if threshold, err := strconv.ParseFloat(*ragConfig.RetrievalThreshold, 64); err == nil {
				request.MinConfidence = threshold
			}
		}

	}

	// Execute RAG retrieval.

	response, err := r.RAGService.ProcessQuery(ctx, request)
	if err != nil {
		return nil, nil, fmt.Errorf("RAG retrieval failed: %w", err)
	}

	// Create enhanced context.

	enhancedContext := map[string]interface{}{}

	// Create RAG metrics.

	ragMetrics := &nephoranv1.RAGMetrics{
		DocumentsRetrieved: int32(len(response.SourceDocuments)),

		RetrievalDuration: int64(response.RetrievalTime),

		AverageRelevanceScore: float64Ptr(float64(response.Confidence)),

		TopRelevanceScore: float64Ptr(float64(response.Confidence)),

		QueryEnhancement: "false", // Default to false as string

	}

	return enhancedContext, ragMetrics, nil
}

// validateResponse validates the LLM response quality.

func (r *IntentProcessingController) validateResponse(response *llm.ProcessingResponse) (float64, []string) {
	var validationErrors []string

	qualityScore := 1.0

	if !r.Config.ValidationEnabled {
		return qualityScore, validationErrors
	}

	// Parse structured parameters from JSON.

	var structuredParams map[string]interface{}

	if response.ProcessedParameters != "" {
		if err := json.Unmarshal([]byte(response.ProcessedParameters), &structuredParams); err != nil {

			validationErrors = append(validationErrors, "invalid structured parameters JSON")

			qualityScore -= 0.3

		}
	} else {

		validationErrors = append(validationErrors, "response lacks structured parameters")

		qualityScore -= 0.3

	}

	// Check if response contains structured parameters (indicates network function information).

	if len(structuredParams) == 0 {

		validationErrors = append(validationErrors, "response lacks network function information")

		qualityScore -= 0.2

	}

	// Check response length.

	if len(response.Response) < 50 {

		validationErrors = append(validationErrors, "response too short")

		qualityScore -= 0.1

	}

	// Check for telecommunications keywords.

	if !r.containsTelecomKeywords(response.Response) {

		validationErrors = append(validationErrors, "response lacks telecommunications domain keywords")

		qualityScore -= 0.2

	}

	// Ensure quality score is within bounds.

	if qualityScore < 0 {
		qualityScore = 0
	}

	if qualityScore > 1 {
		qualityScore = 1
	}

	return qualityScore, validationErrors
}

// containsTelecomKeywords checks for telecommunications keywords.

func (r *IntentProcessingController) containsTelecomKeywords(text string) bool {
	lowerText := strings.ToLower(text)

	telecomKeywords := []string{
		"5g", "4g", "lte", "nr", "amf", "smf", "upf", "gnb", "ran", "core",

		"network", "slice", "function", "deployment", "scaling", "o-ran",

		"oran", "du", "cu", "ric", "smo", "kubernetes", "helm", "container",
	}

	for _, keyword := range telecomKeywords {
		if strings.Contains(lowerText, keyword) {
			return true
		}
	}

	return false
}

// extractProcessedParameters extracts structured parameters from the response.

func (r *IntentProcessingController) extractProcessedParameters(response *llm.ProcessingResponse) (*nephoranv1.ProcessedParameters, error) {
	if response.ProcessedParameters == "" {
		return nil, fmt.Errorf("no processed parameters in response")
	}

	// Parse JSON structured parameters.

	var structuredParams map[string]interface{}

	if err := json.Unmarshal([]byte(response.ProcessedParameters), &structuredParams); err != nil {
		return nil, fmt.Errorf("failed to parse structured parameters: %w", err)
	}

	params := &nephoranv1.ProcessedParameters{}

	// Extract network function.

	if nf, ok := structuredParams["network_function"].(string); ok {
		params.NetworkFunction = nf
	}

	// Extract region.

	if region, ok := structuredParams["region"].(string); ok {
		params.Region = region
	}

	// Extract scale parameters.

	if scaleParams, ok := structuredParams["scale_parameters"]; ok {
		// This would need proper conversion based on ScaleParameters type.

		// For now, just extract basic parameters.

		if scaleMap, ok := scaleParams.(map[string]interface{}); ok {

			scalingParams := &nephoranv1.ScalingParameters{}

			if replicas, ok := scaleMap["replicas"].(int); ok {

				minReplicas := int32(replicas)

				maxReplicas := int32(replicas * 3) // Default scaling range

				scalingParams.MinReplicas = &minReplicas

				scalingParams.MaxReplicas = &maxReplicas

			}

			params.Scaling = scalingParams

		}
	}

	return params, nil
}

// extractTelecomEntities extracts telecommunications entities from the response.

func (r *IntentProcessingController) extractTelecomEntities(response *llm.ProcessingResponse) (map[string]string, error) {
	entities := make(map[string]string)

	// Since ProcessingResponse doesn't have ExtractedEntities field,.

	// extract entities from ProcessedParameters JSON or Response text.

	if response.ProcessedParameters != "" {

		var params map[string]interface{}

		if err := json.Unmarshal([]byte(response.ProcessedParameters), &params); err == nil {

			// Extract known telecom entities from structured parameters.

			if nf, ok := params["network_function"].(string); ok {
				entities["network_function"] = nf
			}

			if region, ok := params["region"].(string); ok {
				entities["region"] = region
			}

			if deploymentPattern, ok := params["deployment_pattern"].(string); ok {
				entities["deployment_pattern"] = deploymentPattern
			}

		}

	}

	// Extract additional entities from response text using simple keyword detection.

	responseText := response.Response

	telecomKeywords := []string{"AMF", "SMF", "UPF", "5G", "4G", "gNB", "eNB", "PLMN", "TAC"}

	for _, keyword := range telecomKeywords {
		if strings.Contains(responseText, keyword) {
			entities["detected_"+strings.ToLower(keyword)] = keyword
		}
	}

	return entities, nil
}

// handleProcessingSuccess handles successful processing.

func (r *IntentProcessingController) handleProcessingSuccess(ctx context.Context, intentProcessing *nephoranv1.IntentProcessing, result *LLMProcessingResult) (ctrl.Result, error) {
	log := r.Logger.WithValues("intentprocessing", intentProcessing.Name)

	// Update status with results.

	now := metav1.Now()

	intentProcessing.Status.ProcessingCompletionTime = &now

	intentProcessing.Status.Phase = nephoranv1.IntentProcessingPhaseCompleted

	// Set LLM response.

	if responseBytes, err := json.Marshal(result.Response); err == nil {
		intentProcessing.Status.LLMResponse = &runtime.RawExtension{Raw: responseBytes}
	}

	// Set processed parameters.

	intentProcessing.Status.ProcessedParameters = result.ProcessedParameters

	// Set extracted entities.
	extractedEntities := make(map[string]runtime.RawExtension)
	for key, value := range result.ExtractedEntities {
		if valueBytes, err := json.Marshal(value); err == nil {
			extractedEntities[key] = runtime.RawExtension{Raw: valueBytes}
		}
	}
	intentProcessing.Status.ExtractedEntities = extractedEntities

	// Set quality score.
	qualityScoreStr := fmt.Sprintf("%.4f", result.QualityScore)
	intentProcessing.Status.QualityScore = &qualityScoreStr

	// Set validation errors.

	intentProcessing.Status.ValidationErrors = result.ValidationErrors

	// Set token usage.

	intentProcessing.Status.TokenUsage = result.TokenUsage

	// Set RAG metrics.

	intentProcessing.Status.RAGMetrics = result.RAGMetrics

	// Set telecom context - ProcessingResponse doesn't have Metadata field
	// Using empty context for now
	intentProcessing.Status.TelecomContext = make(map[string]string)

	// Calculate processing duration.

	if intentProcessing.Status.ProcessingStartTime != nil {

		duration := now.Sub(intentProcessing.Status.ProcessingStartTime.Time)

		intentProcessing.Status.ProcessingDuration = &metav1.Duration{Duration: duration}

	}

	// Update status.

	if err := r.updateStatus(ctx, intentProcessing); err != nil {
		return ctrl.Result{}, err
	}

	// Record success event.

	r.Recorder.Event(intentProcessing, "Normal", "ProcessingCompleted", "Intent processing completed successfully")

	// Publish completion event.

	if err := r.EventBus.PublishPhaseEvent(ctx, interfaces.PhaseLLMProcessing, EventLLMProcessingCompleted,

		string(intentProcessing.UID), true, map[string]interface{}{}); err != nil {
		log.Error(err, "Failed to publish completion event")
	}

	// Record metrics.

	r.MetricsCollector.RecordPhaseCompletion(interfaces.PhaseLLMProcessing, string(intentProcessing.UID), true)

	log.Info("Intent processing completed successfully", "qualityScore", result.QualityScore)

	return ctrl.Result{}, nil
}

// handleProcessingError handles processing errors with retry logic.

func (r *IntentProcessingController) handleProcessingError(ctx context.Context, intentProcessing *nephoranv1.IntentProcessing, err error) (ctrl.Result, error) {
	log := r.Logger.WithValues("intentprocessing", intentProcessing.Name)

	log.Error(err, "Intent processing failed")

	// Increment retry count.

	intentProcessing.Status.RetryCount++

	now := metav1.Now()

	intentProcessing.Status.LastRetryTime = &now

	// Check if we should retry.

	if intentProcessing.CanRetry() {

		intentProcessing.Status.Phase = nephoranv1.IntentProcessingPhaseRetrying

		// Calculate backoff duration.

		backoffDuration := r.calculateBackoff(intentProcessing.Status.RetryCount)

		if err := r.updateStatus(ctx, intentProcessing); err != nil {
			return ctrl.Result{}, err
		}

		// Record retry event.

		r.Recorder.Event(intentProcessing, "Warning", "ProcessingRetry",

			fmt.Sprintf("Retrying intent processing (attempt %d/%d): %v",

				intentProcessing.Status.RetryCount, intentProcessing.Spec.MaxRetries, err))

		// Publish retry event.

		if pubErr := r.EventBus.PublishPhaseEvent(ctx, interfaces.PhaseLLMProcessing, EventRetryRequired,

			string(intentProcessing.UID), false, map[string]interface{}{}); pubErr != nil {
			log.Error(pubErr, "Failed to publish retry event")
		}

		log.Info("Scheduling retry", "attempt", intentProcessing.Status.RetryCount, "backoff", backoffDuration)

		return ctrl.Result{RequeueAfter: backoffDuration}, nil

	}

	// Max retries exceeded - mark as permanently failed.

	intentProcessing.Status.Phase = nephoranv1.IntentProcessingPhaseFailed

	// Add failure condition.

	condition := metav1.Condition{
		Type: "ProcessingFailed",

		Status: metav1.ConditionTrue,

		ObservedGeneration: intentProcessing.Generation,

		Reason: "MaxRetriesExceeded",

		Message: fmt.Sprintf("Processing failed after %d attempts: %v", intentProcessing.Status.RetryCount, err),

		LastTransitionTime: now,
	}

	intentProcessing.Status.Conditions = append(intentProcessing.Status.Conditions, condition)

	if updateErr := r.updateStatus(ctx, intentProcessing); updateErr != nil {
		return ctrl.Result{}, updateErr
	}

	// Record failure event.

	r.Recorder.Event(intentProcessing, "Warning", "ProcessingFailed",

		fmt.Sprintf("Intent processing failed permanently after %d attempts: %v",

			intentProcessing.Status.RetryCount, err))

	// Publish failure event.

	if pubErr := r.EventBus.PublishPhaseEvent(ctx, interfaces.PhaseLLMProcessing, EventLLMProcessingFailed,

		string(intentProcessing.UID), false, map[string]interface{}{}); pubErr != nil {
		log.Error(pubErr, "Failed to publish failure event")
	}

	// Record metrics.

	r.MetricsCollector.RecordPhaseCompletion(interfaces.PhaseLLMProcessing, string(intentProcessing.UID), false)

	return ctrl.Result{}, nil
}

// calculateBackoff calculates the backoff duration for retries.

func (r *IntentProcessingController) calculateBackoff(retryCount int32) time.Duration {
	backoff := r.Config.RetryBackoff

	for i := int32(1); i < retryCount; i++ {

		backoff *= 2

		if backoff > 5*time.Minute {

			backoff = 5 * time.Minute

			break

		}

	}

	return backoff
}

// handleDeletion handles resource deletion.

func (r *IntentProcessingController) handleDeletion(ctx context.Context, intentProcessing *nephoranv1.IntentProcessing) (ctrl.Result, error) {
	log := r.Logger.WithValues("intentprocessing", intentProcessing.Name)

	log.Info("Handling IntentProcessing deletion")

	// Cleanup any resources if needed.

	// (In this case, there are no external resources to clean up).

	// Remove finalizer.

	controllerutil.RemoveFinalizer(intentProcessing, "intentprocessing.nephoran.com/finalizer")

	return ctrl.Result{}, r.Update(ctx, intentProcessing)
}

// updateStatus updates the status of the IntentProcessing resource.

func (r *IntentProcessingController) updateStatus(ctx context.Context, intentProcessing *nephoranv1.IntentProcessing) error {
	intentProcessing.Status.ObservedGeneration = intentProcessing.Generation

	return r.Status().Update(ctx, intentProcessing)
}

// SetupWithManager sets up the controller with the Manager.

func (r *IntentProcessingController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nephoranv1.IntentProcessing{}).
		Named("intentprocessing").
		Complete(r)
}

// LLMProcessingResult contains the result of LLM processing.

type LLMProcessingResult struct {
	Response *llm.ProcessingResponse

	ProcessedParameters *nephoranv1.ProcessedParameters

	ExtractedEntities map[string]string

	QualityScore float64

	ValidationErrors []string

	TokenUsage *nephoranv1.TokenUsageInfo

	RAGMetrics *nephoranv1.RAGMetrics
}

// extractRAGMetricsFromContextString extracts RAG metrics from JSON context string.

func (r *IntentProcessingController) extractRAGMetricsFromContextString(contextStr string) *nephoranv1.RAGMetrics {
	if contextStr == "" {
		return nil
	}

	var context map[string]interface{}

	if err := json.Unmarshal([]byte(contextStr), &context); err != nil {
		return nil
	}

	return extractRAGMetricsFromContext(context)
}

// extractRAGMetricsFromContext extracts RAG metrics from request context.

func extractRAGMetricsFromContext(context map[string]interface{}) *nephoranv1.RAGMetrics {
	if context == nil {
		return nil
	}

	if ragMetricsVal, ok := context["ragMetrics"]; ok {
		if ragMetrics, ok := ragMetricsVal.(*nephoranv1.RAGMetrics); ok {
			return ragMetrics
		}
	}

	return nil
}

// Default configuration values.

func DefaultIntentProcessingConfig() *IntentProcessingConfig {
	return &IntentProcessingConfig{
		MaxConcurrentProcessing: 10,

		DefaultTimeout: 120 * time.Second,

		MaxRetries: 3,

		RetryBackoff: 30 * time.Second,

		QualityThreshold: 0.7,

		ValidationEnabled: true,
	}
}

// convertStringMapToInterface converts map[string]string to map[string]interface{}
func convertStringMapToInterface(m map[string]string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		result[k] = v
	}
	return result
}

// convertInterfaceMapToString converts map[string]interface{} to map[string]string
func convertInterfaceMapToString(m map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		if str, ok := v.(string); ok {
			result[k] = str
		} else {
			result[k] = fmt.Sprintf("%v", v)
		}
	}
	return result
}

// float64Ptr returns a pointer to the given float64 value
func float64Ptr(f float64) *float64 {
	return &f
}

