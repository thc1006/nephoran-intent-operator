//go:build !disable_rag
// +build !disable_rag

package llm

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ProcessingPipeline orchestrates the complete intent processing workflow
type ProcessingPipeline struct {
	preprocessor  *IntentPreprocessor
	classifier    *IntentClassifier
	enricher      *ContextEnricher
	validator     *InputValidator
	transformer   *ResponseTransformer
	postprocessor *ResponsePostprocessor
	logger        *slog.Logger
	metrics       *PipelineMetrics
	config        PipelineConfig
}

// PipelineConfig contains configuration for the processing pipeline
type PipelineConfig struct {
	EnablePreprocessing          bool
	EnableContextEnrichment      bool
	EnableResponseTransformation bool
	EnablePostprocessing         bool
	MaxProcessingTime            time.Duration
	ValidationStrictness         string // "strict", "normal", "relaxed"
	CacheEnabled                 bool
	AsyncProcessing              bool
}

// PipelineMetrics tracks processing pipeline performance
type PipelineMetrics struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	ValidationFailures int64
	PreprocessingTime  time.Duration
	ClassificationTime time.Duration
	EnrichmentTime     time.Duration
	TransformationTime time.Duration
	PostprocessingTime time.Duration
	mutex              sync.RWMutex
}

// IntentPreprocessor cleans and normalizes intent text
type IntentPreprocessor struct {
	normalizationRules map[string]string
	stopWords          []string
	logger             *slog.Logger
}

// IntentClassifier determines intent types and confidence scores
type IntentClassifier struct {
	rules   []ClassificationRule
	mlModel MLClassifier // Interface for ML-based classification
	logger  *slog.Logger
}

type ClassificationRule struct {
	Pattern    *regexp.Regexp
	IntentType string
	Confidence float64
	Priority   int
}

type ClassificationResult struct {
	IntentType      string  `json:"intent_type"`
	Confidence      float64 `json:"confidence"`
	SubCategory     string  `json:"sub_category,omitempty"`
	NetworkFunction string  `json:"network_function,omitempty"`
	OperationType   string  `json:"operation_type,omitempty"`
	Priority        string  `json:"priority,omitempty"`
}

// MLClassifier interface for machine learning-based classification
type MLClassifier interface {
	Classify(intent string) (ClassificationResult, error)
	UpdateModel(trainingData []TrainingExample) error
	GetModelMetadata() map[string]interface{}
}

type TrainingExample struct {
	Intent   string                 `json:"intent"`
	Label    string                 `json:"label"`
	Metadata map[string]interface{} `json:"metadata"`
}

// ContextEnricher adds context information to the processing
type ContextEnricher struct {
	knowledgeBase *TelecomKnowledgeBase
	contextCache  map[string]*EnrichmentContext
	mutex         sync.RWMutex
	logger        *slog.Logger
}

type EnrichmentContext struct {
	NetworkTopology   *PipelineNetworkTopology `json:"network_topology,omitempty"`
	DeploymentContext *DeploymentContext       `json:"deployment_context,omitempty"`
	PolicyContext     *PolicyContext           `json:"policy_context,omitempty"`
	HistoricalData    *HistoricalData          `json:"historical_data,omitempty"`
	Timestamp         time.Time                `json:"timestamp"`
}

type PipelineNetworkTopology struct {
	Region           string                 `json:"region"`
	AvailabilityZone string                 `json:"availability_zone"`
	NetworkSlices    []PipelineNetworkSlice `json:"network_slices"`
	Constraints      map[string]string      `json:"constraints"`
}

type PipelineNetworkSlice struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"` // eMBB, URLLC, mMTC
	Status      string  `json:"status"`
	Capacity    int     `json:"capacity"`
	Utilization float64 `json:"utilization"`
}

type DeploymentContext struct {
	Environment       string            `json:"environment"` // dev, staging, prod
	Cluster           string            `json:"cluster"`
	Namespace         string            `json:"namespace"`
	ExistingWorkloads []WorkloadInfo    `json:"existing_workloads"`
	ResourceQuotas    map[string]string `json:"resource_quotas"`
}

type WorkloadInfo struct {
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Status    string            `json:"status"`
	Resources map[string]string `json:"resources"`
	Labels    map[string]string `json:"labels"`
}

type PolicyContext struct {
	SecurityPolicies       []Policy `json:"security_policies"`
	ResourcePolicies       []Policy `json:"resource_policies"`
	NetworkPolicies        []Policy `json:"network_policies"`
	ComplianceRequirements []string `json:"compliance_requirements"`
}

type Policy struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	Rules    []PolicyRule           `json:"rules"`
	Metadata map[string]interface{} `json:"metadata"`
	Enabled  bool                   `json:"enabled"`
}

type PolicyRule struct {
	Condition string `json:"condition"`
	Action    string `json:"action"`
	Priority  int    `json:"priority"`
}

type HistoricalData struct {
	SimilarIntents     []HistoricalIntent `json:"similar_intents"`
	PerformanceMetrics map[string]float64 `json:"performance_metrics"`
	SuccessRates       map[string]float64 `json:"success_rates"`
}

type HistoricalIntent struct {
	Intent      string             `json:"intent"`
	Outcome     string             `json:"outcome"`
	Timestamp   time.Time          `json:"timestamp"`
	Performance map[string]float64 `json:"performance"`
}

// InputValidator provides comprehensive input validation
type InputValidator struct {
	rules            []ValidationRule
	customValidators map[string]func(string) error
	strictMode       bool
	logger           *slog.Logger
}

type ValidationRule struct {
	Name         string
	Pattern      *regexp.Regexp
	Required     bool
	ErrorMessage string
	Severity     string // "error", "warning", "info"
}

type PipelineValidationResult struct {
	Valid    bool                      `json:"valid"`
	Errors   []PipelineValidationError `json:"errors,omitempty"`
	Warnings []PipelineValidationError `json:"warnings,omitempty"`
	Score    float64                   `json:"score"`
}

type PipelineValidationError struct {
	Field    string `json:"field"`
	Message  string `json:"message"`
	Code     string `json:"code"`
	Severity string `json:"severity"`
}

// ResponseTransformer modifies and enhances LLM responses
type ResponseTransformer struct {
	transformers map[string]TransformationFunc
	logger       *slog.Logger
}

type TransformationFunc func(map[string]interface{}) (map[string]interface{}, error)

// ResponsePostprocessor finalizes responses with additional metadata
type ResponsePostprocessor struct {
	enrichers map[string]PostprocessingFunc
	logger    *slog.Logger
}

type PostprocessingFunc func(map[string]interface{}, *ProcessingContext) (map[string]interface{}, error)

type ProcessingContext struct {
	RequestID        string                    `json:"request_id"`
	Intent           string                    `json:"intent"`
	Classification   ClassificationResult      `json:"classification"`
	EnrichmentData   *EnrichmentContext        `json:"enrichment_data"`
	ValidationResult *PipelineValidationResult `json:"validation_result"`
	ProcessingStart  time.Time                 `json:"processing_start"`
	Metadata         map[string]interface{}    `json:"metadata"`
}

// NewProcessingPipeline creates a new processing pipeline
func NewProcessingPipeline(config PipelineConfig) *ProcessingPipeline {
	logger := slog.Default().With("component", "processing-pipeline")

	return &ProcessingPipeline{
		preprocessor:  NewIntentPreprocessor(),
		classifier:    NewIntentClassifier(),
		enricher:      NewContextEnricher(),
		validator:     NewInputValidator(config.ValidationStrictness),
		transformer:   NewResponseTransformer(),
		postprocessor: NewResponsePostprocessor(),
		logger:        logger,
		metrics:       NewPipelineMetrics(),
		config:        config,
	}
}

// NewPipelineMetrics creates new pipeline metrics
func NewPipelineMetrics() *PipelineMetrics {
	return &PipelineMetrics{}
}

// ProcessIntent processes an intent through the complete pipeline
func (pp *ProcessingPipeline) ProcessIntent(ctx context.Context, intent string, metadata map[string]interface{}) (*PipelineProcessingResult, error) {
	start := time.Now()

	// Create processing context
	processingCtx := &ProcessingContext{
		RequestID:       fmt.Sprintf("req_%d", time.Now().UnixNano()),
		Intent:          intent,
		ProcessingStart: start,
		Metadata:        metadata,
	}

	pp.logger.Info("Starting intent processing",
		slog.String("request_id", processingCtx.RequestID),
		slog.String("intent", intent),
	)

	defer func() {
		pp.updateMetrics(time.Since(start), true)
	}()

	// Step 1: Preprocessing
	if pp.config.EnablePreprocessing {
		preprocessStart := time.Now()
		preprocessedIntent, err := pp.preprocessor.Preprocess(intent)
		if err != nil {
			return nil, fmt.Errorf("preprocessing failed: %w", err)
		}
		processingCtx.Intent = preprocessedIntent
		pp.metrics.PreprocessingTime += time.Since(preprocessStart)
	}

	// Step 2: Classification
	classifyStart := time.Now()
	classification, err := pp.classifier.Classify(processingCtx.Intent)
	if err != nil {
		return nil, fmt.Errorf("classification failed: %w", err)
	}
	processingCtx.Classification = classification
	pp.metrics.ClassificationTime += time.Since(classifyStart)

	// Step 3: Validation
	validationResult := pp.validator.Validate(processingCtx.Intent)
	processingCtx.ValidationResult = &validationResult

	if !validationResult.Valid && pp.config.ValidationStrictness == "strict" {
		pp.metrics.ValidationFailures++
		return nil, fmt.Errorf("validation failed: %v", validationResult.Errors)
	}

	// Step 4: Context Enrichment
	if pp.config.EnableContextEnrichment {
		enrichStart := time.Now()
		enrichmentData, err := pp.enricher.Enrich(ctx, processingCtx)
		if err != nil {
			pp.logger.Warn("Context enrichment failed", slog.String("error", err.Error()))
		} else {
			processingCtx.EnrichmentData = enrichmentData
		}
		pp.metrics.EnrichmentTime += time.Since(enrichStart)
	}

	result := &PipelineProcessingResult{
		ProcessingContext: processingCtx,
		ProcessingTime:    time.Since(start),
		Success:           true,
	}

	pp.logger.Info("Intent processing completed",
		slog.String("request_id", processingCtx.RequestID),
		slog.String("intent_type", classification.IntentType),
		slog.Float64("confidence", classification.Confidence),
		slog.Duration("processing_time", result.ProcessingTime),
	)

	return result, nil
}

type PipelineProcessingResult struct {
	ProcessingContext *ProcessingContext `json:"processing_context"`
	ProcessingTime    time.Duration      `json:"processing_time"`
	Success           bool               `json:"success"`
	Error             string             `json:"error,omitempty"`
}

// TransformResponse applies transformations to an LLM response
func (pp *ProcessingPipeline) TransformResponse(response map[string]interface{}, context *ProcessingContext) (map[string]interface{}, error) {
	if !pp.config.EnableResponseTransformation {
		return response, nil
	}

	transformStart := time.Now()
	defer func() {
		pp.metrics.TransformationTime += time.Since(transformStart)
	}()

	return pp.transformer.Transform(response, context.Classification.IntentType)
}

// PostprocessResponse applies final postprocessing to the response
func (pp *ProcessingPipeline) PostprocessResponse(response map[string]interface{}, context *ProcessingContext) (map[string]interface{}, error) {
	if !pp.config.EnablePostprocessing {
		return response, nil
	}

	postprocessStart := time.Now()
	defer func() {
		pp.metrics.PostprocessingTime += time.Since(postprocessStart)
	}()

	return pp.postprocessor.Postprocess(response, context)
}

// updateMetrics updates pipeline metrics
func (pp *ProcessingPipeline) updateMetrics(processingTime time.Duration, success bool) {
	pp.metrics.mutex.Lock()
	defer pp.metrics.mutex.Unlock()

	pp.metrics.TotalRequests++
	if success {
		pp.metrics.SuccessfulRequests++
	} else {
		pp.metrics.FailedRequests++
	}
}

// GetMetrics returns current pipeline metrics
func (pp *ProcessingPipeline) GetMetrics() *PipelineMetrics {
	pp.metrics.mutex.RLock()
	defer pp.metrics.mutex.RUnlock()
	
	// Create a copy without the mutex
	metrics := PipelineMetrics{
		TotalRequests:      pp.metrics.TotalRequests,
		SuccessfulRequests: pp.metrics.SuccessfulRequests,
		FailedRequests:     pp.metrics.FailedRequests,
		ValidationFailures: pp.metrics.ValidationFailures,
		PreprocessingTime:  pp.metrics.PreprocessingTime,
		ClassificationTime: pp.metrics.ClassificationTime,
		EnrichmentTime:     pp.metrics.EnrichmentTime,
		TransformationTime: pp.metrics.TransformationTime,
		PostprocessingTime: pp.metrics.PostprocessingTime,
	}
	return &metrics
}

// NewIntentPreprocessor creates a new intent preprocessor
func NewIntentPreprocessor() *IntentPreprocessor {
	return &IntentPreprocessor{
		normalizationRules: map[string]string{
			"(?i)\\bupf\\b":         "User Plane Function",
			"(?i)\\bamf\\b":         "Access and Mobility Management Function",
			"(?i)\\bsmf\\b":         "Session Management Function",
			"(?i)\\bnear-rt\\s+ric": "Near Real-Time RAN Intelligent Controller",
			"(?i)\\bo-ran\\b":       "Open Radio Access Network",
		},
		stopWords: []string{"the", "a", "an", "and", "or", "but", "in", "on", "at", "to", "for", "of", "with", "by"},
		logger:    slog.Default().With("component", "preprocessor"),
	}
}

// Preprocess cleans and normalizes intent text
func (ip *IntentPreprocessor) Preprocess(intent string) (string, error) {
	if strings.TrimSpace(intent) == "" {
		return "", fmt.Errorf("empty intent")
	}

	// Apply normalization rules
	result := intent
	for pattern, replacement := range ip.normalizationRules {
		re := regexp.MustCompile(pattern)
		result = re.ReplaceAllString(result, replacement)
	}

	// Remove extra whitespace
	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")
	result = strings.TrimSpace(result)

	ip.logger.Debug("Preprocessed intent",
		slog.String("original", intent),
		slog.String("processed", result),
	)

	return result, nil
}

// NewIntentClassifier creates a new intent classifier
func NewIntentClassifier() *IntentClassifier {
	classifier := &IntentClassifier{
		logger: slog.Default().With("component", "classifier"),
	}

	// Initialize classification rules
	classifier.initializeRules()

	return classifier
}

// initializeRules sets up classification rules
func (ic *IntentClassifier) initializeRules() {
	ic.rules = []ClassificationRule{
		{
			Pattern:    regexp.MustCompile(`(?i)\b(deploy|create|install|setup|provision)\b`),
			IntentType: "NetworkFunctionDeployment",
			Confidence: 0.8,
			Priority:   1,
		},
		{
			Pattern:    regexp.MustCompile(`(?i)\b(scale|resize|increase|decrease|replicas?)\b`),
			IntentType: "NetworkFunctionScale",
			Confidence: 0.8,
			Priority:   1,
		},
		{
			Pattern:    regexp.MustCompile(`(?i)\b(update|modify|change|configure)\b`),
			IntentType: "NetworkFunctionUpdate",
			Confidence: 0.7,
			Priority:   2,
		},
		{
			Pattern:    regexp.MustCompile(`(?i)\b(delete|remove|terminate|destroy)\b`),
			IntentType: "NetworkFunctionDeletion",
			Confidence: 0.8,
			Priority:   1,
		},
	}
}

// Classify determines the intent type and confidence
func (ic *IntentClassifier) Classify(intent string) (ClassificationResult, error) {
	lowerIntent := strings.ToLower(intent)

	result := ClassificationResult{
		IntentType: "Unknown",
		Confidence: 0.0,
	}

	// Apply rule-based classification
	maxConfidence := 0.0
	for _, rule := range ic.rules {
		if rule.Pattern.MatchString(intent) {
			if rule.Confidence > maxConfidence {
				result.IntentType = rule.IntentType
				result.Confidence = rule.Confidence
				maxConfidence = rule.Confidence
			}
		}
	}

	// Detect network function
	nfPatterns := map[string]string{
		"upf|user plane function":            "UPF",
		"amf|access and mobility management": "AMF",
		"smf|session management":             "SMF",
		"pcf|policy control":                 "PCF",
		"near-rt ric|near real-time ric":     "Near-RT-RIC",
		"o-du|distributed unit":              "O-DU",
		"o-cu|central unit":                  "O-CU",
	}

	for pattern, nf := range nfPatterns {
		if matched, _ := regexp.MatchString("(?i)"+pattern, lowerIntent); matched {
			result.NetworkFunction = nf
			break
		}
	}

	// Detect operation type
	if regexp.MustCompile(`(?i)\b(high availability|ha|redundan)\b`).MatchString(lowerIntent) {
		result.OperationType = "HighAvailability"
	} else if regexp.MustCompile(`(?i)\b(edge|mec|local)\b`).MatchString(lowerIntent) {
		result.OperationType = "EdgeDeployment"
	} else if regexp.MustCompile(`(?i)\b(core|central)\b`).MatchString(lowerIntent) {
		result.OperationType = "CoreDeployment"
	}

	// Determine priority
	if regexp.MustCompile(`(?i)\b(urgent|critical|emergency)\b`).MatchString(lowerIntent) {
		result.Priority = "High"
	} else if regexp.MustCompile(`(?i)\b(low|background)\b`).MatchString(lowerIntent) {
		result.Priority = "Low"
	} else {
		result.Priority = "Medium"
	}

	ic.logger.Debug("Intent classified",
		slog.String("intent", intent),
		slog.String("type", result.IntentType),
		slog.Float64("confidence", result.Confidence),
		slog.String("network_function", result.NetworkFunction),
	)

	return result, nil
}

// NewContextEnricher creates a new context enricher
func NewContextEnricher() *ContextEnricher {
	return &ContextEnricher{
		knowledgeBase: NewTelecomKnowledgeBase(),
		contextCache:  make(map[string]*EnrichmentContext),
		logger:        slog.Default().With("component", "enricher"),
	}
}

// Enrich adds context information to the processing
func (ce *ContextEnricher) Enrich(ctx context.Context, processingCtx *ProcessingContext) (*EnrichmentContext, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s:%s", processingCtx.Classification.IntentType, processingCtx.Classification.NetworkFunction)

	ce.mutex.RLock()
	if cached, exists := ce.contextCache[cacheKey]; exists {
		// Create a copy of the cached enrichment context to avoid sharing
		cachedCopy := &EnrichmentContext{
			NetworkTopology:   cached.NetworkTopology,
			DeploymentContext: cached.DeploymentContext,
			PolicyContext:     cached.PolicyContext,
			HistoricalData:    cached.HistoricalData,
			Timestamp:         cached.Timestamp,
		}
		ce.mutex.RUnlock()
		return cachedCopy, nil
	}
	ce.mutex.RUnlock()

	// Build enrichment context
	enrichment := &EnrichmentContext{
		Timestamp: time.Now(),
	}

	// Add network topology
	enrichment.NetworkTopology = ce.buildNetworkTopology(processingCtx)

	// Add deployment context
	enrichment.DeploymentContext = ce.buildDeploymentContext(processingCtx)

	// Add policy context
	enrichment.PolicyContext = ce.buildPolicyContext(processingCtx)

	// Add historical data
	enrichment.HistoricalData = ce.buildHistoricalData(processingCtx)

	// Cache the result
	ce.mutex.Lock()
	ce.contextCache[cacheKey] = enrichment
	ce.mutex.Unlock()

	return enrichment, nil
}

// buildNetworkTopology creates network topology context
func (ce *ContextEnricher) buildNetworkTopology(ctx *ProcessingContext) *PipelineNetworkTopology {
	// This would typically query actual network topology
	return &PipelineNetworkTopology{
		Region:           "us-west-2",
		AvailabilityZone: "us-west-2a",
		NetworkSlices: []PipelineNetworkSlice{
			{ID: "slice-embb-001", Type: "eMBB", Status: "active", Capacity: 1000, Utilization: 0.65},
			{ID: "slice-urllc-001", Type: "URLLC", Status: "active", Capacity: 500, Utilization: 0.30},
		},
		Constraints: map[string]string{
			"latency":   "< 10ms",
			"bandwidth": "> 1Gbps",
		},
	}
}

// buildDeploymentContext creates deployment context
func (ce *ContextEnricher) buildDeploymentContext(ctx *ProcessingContext) *DeploymentContext {
	// This would typically query Kubernetes API
	return &DeploymentContext{
		Environment: "production",
		Cluster:     "5g-core-cluster",
		Namespace:   "5g-core",
		ExistingWorkloads: []WorkloadInfo{
			{Name: "amf-deployment", Type: "Deployment", Status: "Running", Resources: map[string]string{"cpu": "500m", "memory": "1Gi"}},
		},
		ResourceQuotas: map[string]string{
			"cpu":    "10",
			"memory": "20Gi",
		},
	}
}

// buildPolicyContext creates policy context
func (ce *ContextEnricher) buildPolicyContext(ctx *ProcessingContext) *PolicyContext {
	return &PolicyContext{
		SecurityPolicies: []Policy{
			{ID: "sec-001", Name: "Network Isolation", Type: "security", Enabled: true},
		},
		ResourcePolicies: []Policy{
			{ID: "res-001", Name: "Resource Limits", Type: "resource", Enabled: true},
		},
		ComplianceRequirements: []string{"GDPR", "SOC2", "FedRAMP"},
	}
}

// buildHistoricalData creates historical context
func (ce *ContextEnricher) buildHistoricalData(ctx *ProcessingContext) *HistoricalData {
	return &HistoricalData{
		SimilarIntents: []HistoricalIntent{
			{Intent: "Deploy UPF with 3 replicas", Outcome: "success", Timestamp: time.Now().Add(-24 * time.Hour)},
		},
		PerformanceMetrics: map[string]float64{
			"average_deployment_time": 45.5,
			"success_rate":            0.95,
		},
		SuccessRates: map[string]float64{
			"UPF_deployment": 0.98,
			"AMF_deployment": 0.92,
		},
	}
}

// NewInputValidator creates a new input validator
func NewInputValidator(strictness string) *InputValidator {
	validator := &InputValidator{
		customValidators: make(map[string]func(string) error),
		strictMode:       strictness == "strict",
		logger:           slog.Default().With("component", "validator"),
	}

	validator.initializeRules()
	return validator
}

// initializeRules sets up validation rules
func (iv *InputValidator) initializeRules() {
	iv.rules = []ValidationRule{
		{
			Name:         "MinLength",
			Pattern:      regexp.MustCompile(`.{10,}`),
			Required:     true,
			ErrorMessage: "Intent must be at least 10 characters long",
			Severity:     "error",
		},
		{
			Name:         "MaxLength",
			Pattern:      regexp.MustCompile(`^.{1,2048}$`),
			Required:     true,
			ErrorMessage: "Intent must not exceed 2048 characters",
			Severity:     "error",
		},
		{
			Name:         "NoSQLInjection",
			Pattern:      regexp.MustCompile(`(?i)(select|insert|update|delete|drop|union|script)`),
			Required:     false,
			ErrorMessage: "Intent contains potentially malicious content",
			Severity:     "error",
		},
	}
}

// Validate performs comprehensive input validation
func (iv *InputValidator) Validate(intent string) PipelineValidationResult {
	result := PipelineValidationResult{
		Valid: true,
		Score: 1.0,
	}

	for _, rule := range iv.rules {
		if rule.Required && !rule.Pattern.MatchString(intent) {
			error := PipelineValidationError{
				Field:    "intent",
				Message:  rule.ErrorMessage,
				Code:     rule.Name,
				Severity: rule.Severity,
			}

			if rule.Severity == "error" {
				result.Errors = append(result.Errors, error)
				result.Valid = false
				result.Score *= 0.5
			} else {
				result.Warnings = append(result.Warnings, error)
				result.Score *= 0.9
			}
		}
	}

	return result
}

// NewResponseTransformer creates a new response transformer
func NewResponseTransformer() *ResponseTransformer {
	transformer := &ResponseTransformer{
		transformers: make(map[string]TransformationFunc),
		logger:       slog.Default().With("component", "transformer"),
	}

	transformer.initializeTransformers()
	return transformer
}

// initializeTransformers sets up transformation functions
func (rt *ResponseTransformer) initializeTransformers() {
	rt.transformers["NetworkFunctionDeployment"] = rt.transformDeploymentResponse
	rt.transformers["NetworkFunctionScale"] = rt.transformScaleResponse
}

// Transform applies transformations to a response
func (rt *ResponseTransformer) Transform(response map[string]interface{}, intentType string) (map[string]interface{}, error) {
	if transformer, exists := rt.transformers[intentType]; exists {
		return transformer(response)
	}
	return response, nil
}

// transformDeploymentResponse transforms deployment responses
func (rt *ResponseTransformer) transformDeploymentResponse(response map[string]interface{}) (map[string]interface{}, error) {
	// Add deployment-specific metadata
	if metadata, ok := response["metadata"].(map[string]interface{}); ok {
		metadata["deployment_strategy"] = "RollingUpdate"
		metadata["min_ready_seconds"] = 10
	}

	return response, nil
}

// transformScaleResponse transforms scaling responses
func (rt *ResponseTransformer) transformScaleResponse(response map[string]interface{}) (map[string]interface{}, error) {
	// Add scaling-specific metadata
	if spec, ok := response["spec"].(map[string]interface{}); ok {
		if scaling, ok := spec["scaling"].(map[string]interface{}); ok {
			scaling["strategy"] = "gradual"
			scaling["cooldown_period"] = "5m"
		}
	}

	return response, nil
}

// NewResponsePostprocessor creates a new response postprocessor
func NewResponsePostprocessor() *ResponsePostprocessor {
	postprocessor := &ResponsePostprocessor{
		enrichers: make(map[string]PostprocessingFunc),
		logger:    slog.Default().With("component", "postprocessor"),
	}

	postprocessor.initializeEnrichers()
	return postprocessor
}

// initializeEnrichers sets up postprocessing functions
func (rp *ResponsePostprocessor) initializeEnrichers() {
	rp.enrichers["metadata"] = rp.addMetadataEnrichment
	rp.enrichers["validation"] = rp.addValidationEnrichment
	rp.enrichers["tracking"] = rp.addTrackingEnrichment
}

// Postprocess applies final processing to responses
func (rp *ResponsePostprocessor) Postprocess(response map[string]interface{}, context *ProcessingContext) (map[string]interface{}, error) {
	for _, enricher := range rp.enrichers {
		var err error
		response, err = enricher(response, context)
		if err != nil {
			return response, err
		}
	}

	return response, nil
}

// addMetadataEnrichment adds processing metadata
func (rp *ResponsePostprocessor) addMetadataEnrichment(response map[string]interface{}, context *ProcessingContext) (map[string]interface{}, error) {
	processingMetadata := map[string]interface{}{
		"request_id":       context.RequestID,
		"processing_time":  time.Since(context.ProcessingStart).Milliseconds(),
		"intent_type":      context.Classification.IntentType,
		"confidence_score": context.Classification.Confidence,
		"network_function": context.Classification.NetworkFunction,
		"processed_at":     time.Now().Format(time.RFC3339),
	}

	response["processing_metadata"] = processingMetadata
	return response, nil
}

// addValidationEnrichment adds validation results
func (rp *ResponsePostprocessor) addValidationEnrichment(response map[string]interface{}, context *ProcessingContext) (map[string]interface{}, error) {
	if context.ValidationResult != nil {
		response["validation_result"] = map[string]interface{}{
			"valid": context.ValidationResult.Valid,
			"score": context.ValidationResult.Score,
		}
	}

	return response, nil
}

// addTrackingEnrichment adds tracking information
func (rp *ResponsePostprocessor) addTrackingEnrichment(response map[string]interface{}, context *ProcessingContext) (map[string]interface{}, error) {
	response["tracking"] = map[string]interface{}{
		"trace_id":  context.RequestID,
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "v1.0.0",
	}

	return response, nil
}

// TelecomKnowledgeBase provides domain-specific knowledge
type TelecomKnowledgeBase struct {
	networkFunctions   map[string]NetworkFunctionSpec
	deploymentPatterns map[string]DeploymentPattern
	mutex              sync.RWMutex
}

type NetworkFunctionSpec struct {
	Name             string            `json:"name"`
	Type             string            `json:"type"`
	DefaultResources map[string]string `json:"default_resources"`
	RequiredPorts    []PortSpec        `json:"required_ports"`
	Dependencies     []string          `json:"dependencies"`
	Constraints      []string          `json:"constraints"`
}

type PortSpec struct {
	Name     string `json:"name"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Required bool   `json:"required"`
}

type DeploymentPattern struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Template     map[string]interface{} `json:"template"`
	Requirements []string               `json:"requirements"`
}

// NewTelecomKnowledgeBase creates a new knowledge base
func NewTelecomKnowledgeBase() *TelecomKnowledgeBase {
	kb := &TelecomKnowledgeBase{
		networkFunctions:   make(map[string]NetworkFunctionSpec),
		deploymentPatterns: make(map[string]DeploymentPattern),
	}

	kb.initializeKnowledgeBase()
	return kb
}

// initializeKnowledgeBase populates the knowledge base
func (kb *TelecomKnowledgeBase) initializeKnowledgeBase() {
	// Initialize network function specifications
	kb.networkFunctions["UPF"] = NetworkFunctionSpec{
		Name: "User Plane Function",
		Type: "5G Core",
		DefaultResources: map[string]string{
			"cpu":    "2000m",
			"memory": "4Gi",
		},
		RequiredPorts: []PortSpec{
			{Name: "n3", Port: 2152, Protocol: "UDP", Required: true},
			{Name: "n4", Port: 8805, Protocol: "UDP", Required: true},
		},
		Dependencies: []string{"AMF", "SMF"},
		Constraints:  []string{"high-performance-networking", "sr-iov"},
	}

	kb.networkFunctions["AMF"] = NetworkFunctionSpec{
		Name: "Access and Mobility Management Function",
		Type: "5G Core",
		DefaultResources: map[string]string{
			"cpu":    "500m",
			"memory": "1Gi",
		},
		RequiredPorts: []PortSpec{
			{Name: "sbi", Port: 8080, Protocol: "TCP", Required: true},
		},
		Dependencies: []string{"UDM", "PCF"},
		Constraints:  []string{"stateless"},
	}
}

// GetNetworkFunctionSpec retrieves network function specification
func (kb *TelecomKnowledgeBase) GetNetworkFunctionSpec(name string) (NetworkFunctionSpec, bool) {
	kb.mutex.RLock()
	defer kb.mutex.RUnlock()

	spec, exists := kb.networkFunctions[name]
	if !exists {
		return NetworkFunctionSpec{}, false
	}
	
	// Create a copy to avoid returning a reference to internal data
	specCopy := NetworkFunctionSpec{
		Name:             spec.Name,
		Type:             spec.Type,
		DefaultResources: make(map[string]string),
		RequiredPorts:    make([]PortSpec, len(spec.RequiredPorts)),
		Dependencies:     make([]string, len(spec.Dependencies)),
		Constraints:      make([]string, len(spec.Constraints)),
	}
	
	// Copy maps and slices
	for k, v := range spec.DefaultResources {
		specCopy.DefaultResources[k] = v
	}
	copy(specCopy.RequiredPorts, spec.RequiredPorts)
	copy(specCopy.Dependencies, spec.Dependencies)
	copy(specCopy.Constraints, spec.Constraints)
	
	return specCopy, true
}
