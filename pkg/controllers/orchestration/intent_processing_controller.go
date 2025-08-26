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
	"strings"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/interfaces"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
	"github.com/thc1006/nephoran-intent-operator/pkg/rag"
)

// IntentProcessingController reconciles IntentProcessing objects
type IntentProcessingController struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Logger   logr.Logger

	// Services
	LLMService *llm.Client
	RAGService *rag.RAGService

	// Configuration
	Config *IntentProcessingConfig

	// Event bus for coordination
	EventBus *EventBus

	// Metrics
	MetricsCollector *MetricsCollector
}

// IntentProcessingConfig contains configuration for the controller
type IntentProcessingConfig struct {
	MaxConcurrentProcessing int           `json:"maxConcurrentProcessing"`
	DefaultTimeout          time.Duration `json:"defaultTimeout"`
	MaxRetries              int           `json:"maxRetries"`
	RetryBackoff            time.Duration `json:"retryBackoff"`
	QualityThreshold        float64       `json:"qualityThreshold"`
	EnableRAG               bool          `json:"enableRAG"`
	ValidationEnabled       bool          `json:"validationEnabled"`
}

// NewIntentProcessingController creates a new IntentProcessingController
func NewIntentProcessingController(
	client client.Client,
	scheme *runtime.Scheme,
	recorder record.EventRecorder,
	llmService *llm.Client,
	ragService *rag.RAGService,
	eventBus *EventBus,
	config *IntentProcessingConfig,
) *IntentProcessingController {
	return &IntentProcessingController{
		Client:           client,
		Scheme:           scheme,
		Recorder:         recorder,
		Logger:           log.Log.WithName("intent-processing-controller"),
		LLMService:       llmService,
		RAGService:       ragService,
		EventBus:         eventBus,
		Config:           config,
		MetricsCollector: NewMetricsCollector(),
	}
}

// Reconcile handles IntentProcessing resources
func (r *IntentProcessingController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Logger.WithValues("intentprocessing", req.NamespacedName)

	// Fetch the IntentProcessing instance
	intentProcessing := &nephoranv1.IntentProcessing{}
	if err := r.Get(ctx, req.NamespacedName, intentProcessing); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("IntentProcessing resource not found, ignoring")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get IntentProcessing")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if intentProcessing.DeletionTimestamp != nil {
		return r.handleDeletion(ctx, intentProcessing)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(intentProcessing, "intentprocessing.nephoran.com/finalizer") {
		controllerutil.AddFinalizer(intentProcessing, "intentprocessing.nephoran.com/finalizer")
		return ctrl.Result{}, r.Update(ctx, intentProcessing)
	}

	// Process the intent
	return r.processIntent(ctx, intentProcessing)
}

// processIntent processes the natural language intent
func (r *IntentProcessingController) processIntent(ctx context.Context, intentProcessing *nephoranv1.IntentProcessing) (ctrl.Result, error) {
	log := r.Logger.WithValues("intentprocessing", intentProcessing.Name, "namespace", intentProcessing.Namespace)

	// Check if processing is already complete
	if intentProcessing.IsProcessingComplete() {
		log.V(1).Info("Intent processing already complete")
		return ctrl.Result{}, nil
	}

	// Check if processing failed and can retry
	if intentProcessing.IsProcessingFailed() && !intentProcessing.CanRetry() {
		log.Info("Intent processing failed and cannot retry")
		return ctrl.Result{}, nil
	}

	// Record processing start
	if intentProcessing.Status.ProcessingStartTime == nil {
		now := metav1.Now()
		intentProcessing.Status.ProcessingStartTime = &now
		intentProcessing.Status.Phase = nephoranv1.IntentProcessingPhaseInProgress
		if err := r.updateStatus(ctx, intentProcessing); err != nil {
			return ctrl.Result{}, err
		}
		r.MetricsCollector.RecordPhaseStart(interfaces.PhaseLLMProcessing, string(intentProcessing.UID))
	}

	// Publish processing start event
	if err := r.EventBus.PublishPhaseEvent(ctx, interfaces.PhaseLLMProcessing, EventLLMProcessingStarted,
		string(intentProcessing.UID), false, map[string]interface{}{
			"intent":   intentProcessing.Spec.OriginalIntent,
			"priority": intentProcessing.Spec.Priority,
		}); err != nil {
		log.Error(err, "Failed to publish processing start event")
	}

	// Create processing context with timeout
	processingCtx, cancel := context.WithTimeout(ctx, intentProcessing.GetProcessingTimeout())
	defer cancel()

	// Execute LLM processing
	result, err := r.executeLLMProcessing(processingCtx, intentProcessing)
	if err != nil {
		return r.handleProcessingError(ctx, intentProcessing, err)
	}

	// Update status with results
	return r.handleProcessingSuccess(ctx, intentProcessing, result)
}

// executeLLMProcessing performs the actual LLM processing
func (r *IntentProcessingController) executeLLMProcessing(ctx context.Context, intentProcessing *nephoranv1.IntentProcessing) (*LLMProcessingResult, error) {
	log := r.Logger.WithValues("intentprocessing", intentProcessing.Name)

	// Prepare LLM request
	request := &llm.ProcessingRequest{
		Intent:     intentProcessing.Spec.OriginalIntent,
		IntentType: string(intentProcessing.Spec.ParentIntentRef.Kind),
		Priority:   string(intentProcessing.Spec.Priority),
	}

	// Configure LLM parameters
	if intentProcessing.Spec.ProcessingConfiguration != nil {
		config := intentProcessing.Spec.ProcessingConfiguration
		request.Provider = string(config.Provider)
		request.Model = config.Model
		request.Temperature = config.Temperature
		request.MaxTokens = config.MaxTokens
		request.SystemPrompt = config.SystemPrompt
	}

	// Enhance with RAG if enabled
	if intentProcessing.ShouldEnableRAG() && r.RAGService != nil {
		log.Info("Enhancing intent with RAG")
		enhancedContext, ragMetrics, err := r.enhanceWithRAG(ctx, intentProcessing.Spec.OriginalIntent, intentProcessing.Spec.ProcessingConfiguration)
		if err != nil {
			log.Error(err, "Failed to enhance with RAG, continuing without enhancement")
			// Continue without RAG enhancement rather than failing
		} else {
			request.Context = enhancedContext
			// Store RAG metrics for status update
			request.RAGMetrics = ragMetrics
		}
	}

	// Execute LLM processing
	log.Info("Executing LLM processing", "provider", request.Provider, "model", request.Model)
	response, err := r.LLMService.ProcessIntent(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("LLM processing failed: %w", err)
	}

	// Validate response quality
	qualityScore, validationErrors := r.validateResponse(response)
	if qualityScore < r.Config.QualityThreshold {
		return nil, fmt.Errorf("response quality score %.2f below threshold %.2f", qualityScore, r.Config.QualityThreshold)
	}

	// Create processing result
	result := &LLMProcessingResult{
		Response:         response,
		QualityScore:     qualityScore,
		ValidationErrors: validationErrors,
		TokenUsage:       response.TokenUsage,
		RAGMetrics:       request.RAGMetrics,
	}

	// Extract structured parameters
	processedParams, err := r.extractProcessedParameters(response)
	if err != nil {
		log.Error(err, "Failed to extract processed parameters")
		// Continue with raw response
	} else {
		result.ProcessedParameters = processedParams
	}

	// Extract telecommunications entities
	entities, err := r.extractTelecomEntities(response)
	if err != nil {
		log.Error(err, "Failed to extract telecom entities")
	} else {
		result.ExtractedEntities = entities
	}

	return result, nil
}

// enhanceWithRAG enhances the intent with RAG context
func (r *IntentProcessingController) enhanceWithRAG(ctx context.Context, intent string, config *nephoranv1.LLMProcessingConfig) (map[string]interface{}, *nephoranv1.RAGMetrics, error) {
	// Prepare RAG request
	request := &rag.RetrievalRequest{
		Query: intent,
	}

	// Configure RAG parameters
	if config != nil && config.RAGConfiguration != nil {
		ragConfig := config.RAGConfiguration
		if ragConfig.MaxDocuments != nil {
			request.MaxDocuments = int(*ragConfig.MaxDocuments)
		}
		if ragConfig.RetrievalThreshold != nil {
			request.RetrievalThreshold = *ragConfig.RetrievalThreshold
		}
		request.KnowledgeBase = ragConfig.KnowledgeBase
		request.EmbeddingModel = ragConfig.EmbeddingModel
	}

	// Execute RAG retrieval
	response, err := r.RAGService.RetrieveContext(ctx, request)
	if err != nil {
		return nil, nil, fmt.Errorf("RAG retrieval failed: %w", err)
	}

	// Create enhanced context
	enhancedContext := map[string]interface{}{
		"original_intent":     intent,
		"retrieved_documents": response.Documents,
		"knowledge_base":      request.KnowledgeBase,
		"retrieval_metadata":  response.Metadata,
	}

	// Create RAG metrics
	ragMetrics := &nephoranv1.RAGMetrics{
		DocumentsRetrieved:    int32(len(response.Documents)),
		RetrievalDuration:     metav1.Duration{Duration: response.Duration},
		AverageRelevanceScore: response.AverageRelevanceScore,
		TopRelevanceScore:     response.TopRelevanceScore,
		KnowledgeBase:         request.KnowledgeBase,
		QueryEnhancement:      response.QueryWasEnhanced,
	}

	return enhancedContext, ragMetrics, nil
}

// validateResponse validates the LLM response quality
func (r *IntentProcessingController) validateResponse(response *llm.ProcessingResponse) (float64, []string) {
	var validationErrors []string
	qualityScore := 1.0

	if !r.Config.ValidationEnabled {
		return qualityScore, validationErrors
	}

	// Check if response contains structured parameters
	if response.StructuredParameters == nil {
		validationErrors = append(validationErrors, "response lacks structured parameters")
		qualityScore -= 0.3
	}

	// Check if response contains network function information
	if response.NetworkFunctions == nil || len(response.NetworkFunctions) == 0 {
		validationErrors = append(validationErrors, "response lacks network function information")
		qualityScore -= 0.2
	}

	// Check response length
	if len(response.ProcessedIntent) < 50 {
		validationErrors = append(validationErrors, "processed intent too short")
		qualityScore -= 0.1
	}

	// Check for telecommunications keywords
	if !r.containsTelecomKeywords(response.ProcessedIntent) {
		validationErrors = append(validationErrors, "response lacks telecommunications domain keywords")
		qualityScore -= 0.2
	}

	// Ensure quality score is within bounds
	if qualityScore < 0 {
		qualityScore = 0
	}
	if qualityScore > 1 {
		qualityScore = 1
	}

	return qualityScore, validationErrors
}

// containsTelecomKeywords checks for telecommunications keywords
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

// extractProcessedParameters extracts structured parameters from the response
func (r *IntentProcessingController) extractProcessedParameters(response *llm.ProcessingResponse) (*nephoranv1.ProcessedParameters, error) {
	if response.StructuredParameters == nil {
		return nil, fmt.Errorf("no structured parameters in response")
	}

	params := &nephoranv1.ProcessedParameters{}

	// Extract network function
	if nf, ok := response.StructuredParameters["network_function"].(string); ok {
		params.NetworkFunction = nf
	}

	// Extract deployment config
	if deployConfig, ok := response.StructuredParameters["deployment_config"]; ok {
		if configBytes, err := json.Marshal(deployConfig); err == nil {
			params.DeploymentConfig = runtime.RawExtension{Raw: configBytes}
		}
	}

	// Extract performance requirements
	if perfReq, ok := response.StructuredParameters["performance_requirements"]; ok {
		if perfBytes, err := json.Marshal(perfReq); err == nil {
			params.PerformanceRequirements = runtime.RawExtension{Raw: perfBytes}
		}
	}

	// Extract scaling policy
	if scalingPolicy, ok := response.StructuredParameters["scaling_policy"]; ok {
		if scalingBytes, err := json.Marshal(scalingPolicy); err == nil {
			params.ScalingPolicy = runtime.RawExtension{Raw: scalingBytes}
		}
	}

	// Extract security policy
	if secPolicy, ok := response.StructuredParameters["security_policy"]; ok {
		if secBytes, err := json.Marshal(secPolicy); err == nil {
			params.SecurityPolicy = runtime.RawExtension{Raw: secBytes}
		}
	}

	return params, nil
}

// extractTelecomEntities extracts telecommunications entities from the response
func (r *IntentProcessingController) extractTelecomEntities(response *llm.ProcessingResponse) (map[string]string, error) {
	entities := make(map[string]string)

	if response.ExtractedEntities == nil {
		return entities, nil
	}

	for key, value := range response.ExtractedEntities {
		if strValue, ok := value.(string); ok {
			entities[key] = strValue
		} else {
			// Convert non-string values to JSON string
			if jsonBytes, err := json.Marshal(value); err == nil {
				entities[key] = string(jsonBytes)
			}
		}
	}

	return entities, nil
}

// handleProcessingSuccess handles successful processing
func (r *IntentProcessingController) handleProcessingSuccess(ctx context.Context, intentProcessing *nephoranv1.IntentProcessing, result *LLMProcessingResult) (ctrl.Result, error) {
	log := r.Logger.WithValues("intentprocessing", intentProcessing.Name)

	// Update status with results
	now := metav1.Now()
	intentProcessing.Status.ProcessingCompletionTime = &now
	intentProcessing.Status.Phase = nephoranv1.IntentProcessingPhaseCompleted

	// Set LLM response
	if responseBytes, err := json.Marshal(result.Response); err == nil {
		intentProcessing.Status.LLMResponse = runtime.RawExtension{Raw: responseBytes}
	}

	// Set processed parameters
	intentProcessing.Status.ProcessedParameters = result.ProcessedParameters

	// Set extracted entities
	intentProcessing.Status.ExtractedEntities = result.ExtractedEntities

	// Set quality score
	intentProcessing.Status.QualityScore = &result.QualityScore

	// Set validation errors
	intentProcessing.Status.ValidationErrors = result.ValidationErrors

	// Set token usage
	intentProcessing.Status.TokenUsage = result.TokenUsage

	// Set RAG metrics
	intentProcessing.Status.RAGMetrics = result.RAGMetrics

	// Set telecom context
	if result.Response.TelecomContext != nil {
		if contextBytes, err := json.Marshal(result.Response.TelecomContext); err == nil {
			intentProcessing.Status.TelecomContext = runtime.RawExtension{Raw: contextBytes}
		}
	}

	// Calculate processing duration
	if intentProcessing.Status.ProcessingStartTime != nil {
		duration := now.Sub(intentProcessing.Status.ProcessingStartTime.Time)
		intentProcessing.Status.ProcessingDuration = &metav1.Duration{Duration: duration}
	}

	// Update status
	if err := r.updateStatus(ctx, intentProcessing); err != nil {
		return ctrl.Result{}, err
	}

	// Record success event
	r.Recorder.Event(intentProcessing, "Normal", "ProcessingCompleted", "Intent processing completed successfully")

	// Publish completion event
	if err := r.EventBus.PublishPhaseEvent(ctx, interfaces.PhaseLLMProcessing, EventLLMProcessingCompleted,
		string(intentProcessing.UID), true, map[string]interface{}{
			"quality_score":       result.QualityScore,
			"token_usage":         result.TokenUsage,
			"processing_duration": intentProcessing.Status.ProcessingDuration.Duration.String(),
		}); err != nil {
		log.Error(err, "Failed to publish completion event")
	}

	// Record metrics
	r.MetricsCollector.RecordPhaseCompletion(interfaces.PhaseLLMProcessing, string(intentProcessing.UID), true)

	log.Info("Intent processing completed successfully", "qualityScore", result.QualityScore)
	return ctrl.Result{}, nil
}

// handleProcessingError handles processing errors with retry logic
func (r *IntentProcessingController) handleProcessingError(ctx context.Context, intentProcessing *nephoranv1.IntentProcessing, err error) (ctrl.Result, error) {
	log := r.Logger.WithValues("intentprocessing", intentProcessing.Name)

	log.Error(err, "Intent processing failed")

	// Increment retry count
	intentProcessing.Status.RetryCount++
	now := metav1.Now()
	intentProcessing.Status.LastRetryTime = &now

	// Check if we should retry
	if intentProcessing.CanRetry() {
		intentProcessing.Status.Phase = nephoranv1.IntentProcessingPhaseRetrying

		// Calculate backoff duration
		backoffDuration := r.calculateBackoff(intentProcessing.Status.RetryCount)

		if err := r.updateStatus(ctx, intentProcessing); err != nil {
			return ctrl.Result{}, err
		}

		// Record retry event
		r.Recorder.Event(intentProcessing, "Warning", "ProcessingRetry",
			fmt.Sprintf("Retrying intent processing (attempt %d/%d): %v",
				intentProcessing.Status.RetryCount, *intentProcessing.Spec.MaxRetries, err))

		// Publish retry event
		if pubErr := r.EventBus.PublishPhaseEvent(ctx, interfaces.PhaseLLMProcessing, EventRetryRequired,
			string(intentProcessing.UID), false, map[string]interface{}{
				"retry_count":      intentProcessing.Status.RetryCount,
				"error":            err.Error(),
				"backoff_duration": backoffDuration.String(),
			}); pubErr != nil {
			log.Error(pubErr, "Failed to publish retry event")
		}

		log.Info("Scheduling retry", "attempt", intentProcessing.Status.RetryCount, "backoff", backoffDuration)
		return ctrl.Result{RequeueAfter: backoffDuration}, nil
	}

	// Max retries exceeded - mark as permanently failed
	intentProcessing.Status.Phase = nephoranv1.IntentProcessingPhaseFailed

	// Add failure condition
	condition := metav1.Condition{
		Type:               "ProcessingFailed",
		Status:             metav1.ConditionTrue,
		ObservedGeneration: intentProcessing.Generation,
		Reason:             "MaxRetriesExceeded",
		Message:            fmt.Sprintf("Processing failed after %d attempts: %v", intentProcessing.Status.RetryCount, err),
		LastTransitionTime: now,
	}
	intentProcessing.Status.Conditions = append(intentProcessing.Status.Conditions, condition)

	if updateErr := r.updateStatus(ctx, intentProcessing); updateErr != nil {
		return ctrl.Result{}, updateErr
	}

	// Record failure event
	r.Recorder.Event(intentProcessing, "Warning", "ProcessingFailed",
		fmt.Sprintf("Intent processing failed permanently after %d attempts: %v",
			intentProcessing.Status.RetryCount, err))

	// Publish failure event
	if pubErr := r.EventBus.PublishPhaseEvent(ctx, interfaces.PhaseLLMProcessing, EventLLMProcessingFailed,
		string(intentProcessing.UID), false, map[string]interface{}{
			"retry_count": intentProcessing.Status.RetryCount,
			"final_error": err.Error(),
		}); pubErr != nil {
		log.Error(pubErr, "Failed to publish failure event")
	}

	// Record metrics
	r.MetricsCollector.RecordPhaseCompletion(interfaces.PhaseLLMProcessing, string(intentProcessing.UID), false)

	return ctrl.Result{}, nil
}

// calculateBackoff calculates the backoff duration for retries
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

// handleDeletion handles resource deletion
func (r *IntentProcessingController) handleDeletion(ctx context.Context, intentProcessing *nephoranv1.IntentProcessing) (ctrl.Result, error) {
	log := r.Logger.WithValues("intentprocessing", intentProcessing.Name)

	log.Info("Handling IntentProcessing deletion")

	// Cleanup any resources if needed
	// (In this case, there are no external resources to clean up)

	// Remove finalizer
	controllerutil.RemoveFinalizer(intentProcessing, "intentprocessing.nephoran.com/finalizer")
	return ctrl.Result{}, r.Update(ctx, intentProcessing)
}

// updateStatus updates the status of the IntentProcessing resource
func (r *IntentProcessingController) updateStatus(ctx context.Context, intentProcessing *nephoranv1.IntentProcessing) error {
	intentProcessing.Status.ObservedGeneration = intentProcessing.Generation
	return r.Status().Update(ctx, intentProcessing)
}

// SetupWithManager sets up the controller with the Manager
func (r *IntentProcessingController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nephoranv1.IntentProcessing{}).
		Named("intentprocessing").
		WithOptions(ctrl.Options{
			MaxConcurrentReconciles: r.Config.MaxConcurrentProcessing,
		}).
		Complete(r)
}

// LLMProcessingResult contains the result of LLM processing
type LLMProcessingResult struct {
	Response            *llm.ProcessingResponse
	ProcessedParameters *nephoranv1.ProcessedParameters
	ExtractedEntities   map[string]string
	QualityScore        float64
	ValidationErrors    []string
	TokenUsage          *nephoranv1.TokenUsageInfo
	RAGMetrics          *nephoranv1.RAGMetrics
}

// Default configuration values
func DefaultIntentProcessingConfig() *IntentProcessingConfig {
	return &IntentProcessingConfig{
		MaxConcurrentProcessing: 10,
		DefaultTimeout:          120 * time.Second,
		MaxRetries:              3,
		RetryBackoff:            30 * time.Second,
		QualityThreshold:        0.7,
		EnableRAG:               true,
		ValidationEnabled:       true,
	}
}
