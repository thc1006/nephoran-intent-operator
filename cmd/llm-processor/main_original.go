package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	// "github.com/prometheus/client_golang/prometheus"
	// "github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
)

var (
	config    *Config
	processor *IntentProcessor
	logger    *slog.Logger
	startTime = time.Now()
	healthMux sync.RWMutex
	healthy   = true
	readyMux  sync.RWMutex
	ready     = false
	requestID int64
)

// Configuration structure
type Config struct {
	// Service Configuration
	Port             string
	LogLevel         string
	ServiceVersion   string
	GracefulShutdown time.Duration

	// LLM Configuration
	LLMBackendType string
	LLMAPIKey      string
	LLMModelName   string
	LLMTimeout     time.Duration
	LLMMaxTokens   int

	// Mistral Configuration
	MistralAPIURL string
	MistralAPIKey string

	// OpenAI Configuration
	OpenAIAPIURL string
	OpenAIAPIKey string

	// RAG API Configuration
	RAGAPIURL  string
	RAGTimeout time.Duration
	RAGEnabled bool

	// Circuit Breaker Configuration
	CircuitBreakerEnabled   bool
	CircuitBreakerThreshold int
	CircuitBreakerTimeout   time.Duration

	// Retry Configuration
	MaxRetries   int
	RetryDelay   time.Duration
	RetryBackoff string

	// Rate Limiting
	RateLimitRPM   int
	RateLimitBurst int

	// Monitoring
	MetricsEnabled bool
	TracingEnabled bool
	JaegerEndpoint string

	// Security
	APIKeyRequired bool
	APIKey         string
	CORSEnabled    bool
	AllowedOrigins []string

	// Request Logging
	RequestLogEnabled bool
	RequestLogLevel   string
}

// Request/Response structures
type NetworkIntentRequest struct {
	Spec struct {
		Intent string `json:"intent"`
	} `json:"spec"`
	Metadata struct {
		Name       string `json:"name,omitempty"`
		Namespace  string `json:"namespace,omitempty"`
		UID        string `json:"uid,omitempty"`
		Generation int64  `json:"generation,omitempty"`
	} `json:"metadata,omitempty"`
}

type NetworkIntentResponse struct {
	Type               string                 `json:"type"`
	Name               string                 `json:"name"`
	Namespace          string                 `json:"namespace"`
	Spec               map[string]interface{} `json:"spec"`
	O1Config           string                 `json:"o1_config,omitempty"`
	A1Policy           map[string]interface{} `json:"a1_policy,omitempty"`
	OriginalIntent     string                 `json:"original_intent"`
	Timestamp          string                 `json:"timestamp"`
	ProcessingMetadata ProcessingMetadata     `json:"processing_metadata"`
}

type ProcessingMetadata struct {
	ModelUsed        string  `json:"model_used"`
	ConfidenceScore  float64 `json:"confidence_score"`
	ProcessingTimeMS int64   `json:"processing_time_ms"`
}

type ErrorResponse struct {
	Error             string            `json:"error"`
	ErrorCode         string            `json:"error_code"`
	OriginalIntent    string            `json:"original_intent"`
	Timestamp         string            `json:"timestamp"`
	RetryAfterSeconds int               `json:"retry_after_seconds,omitempty"`
	Details           map[string]string `json:"details,omitempty"`
}

type HealthResponse struct {
	Status        string `json:"status"`
	Timestamp     string `json:"timestamp"`
	Version       string `json:"version"`
	UptimeSeconds int64  `json:"uptime_seconds"`
}

type ReadinessResponse struct {
	Status       string            `json:"status"`
	Timestamp    string            `json:"timestamp"`
	Dependencies map[string]string `json:"dependencies"`
}

// Prometheus metrics
var (
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_processor_requests_total",
			Help: "Total number of requests processed",
		},
		[]string{"method", "status_code"},
	)

	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "llm_processor_request_duration_seconds",
			Help:    "Request processing duration",
			Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0},
		},
		[]string{"method"},
	)

	ragAPIRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_processor_rag_api_requests_total",
			Help: "Total requests to RAG API",
		},
		[]string{"status"},
	)

	processingErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_processor_errors_total",
			Help: "Total processing errors",
		},
		[]string{"error_type"},
	)
)

// Circuit Breaker implementation
type CircuitBreaker struct {
	failures    int
	lastFailure time.Time
	state       string // "closed", "open", "half-open"
	threshold   int
	timeout     time.Duration
	mutex       sync.RWMutex
}

func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		threshold: threshold,
		timeout:   timeout,
		state:     "closed",
	}
}

func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	// Check if circuit should transition from open to half-open
	if cb.state == "open" && time.Since(cb.lastFailure) > cb.timeout {
		cb.state = "half-open"
		cb.failures = 0
	}

	// If circuit is open, reject immediately
	if cb.state == "open" {
		return fmt.Errorf("circuit breaker is open")
	}

	// Execute function
	err := fn()
	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()

		// Open circuit if threshold exceeded
		if cb.failures >= cb.threshold {
			cb.state = "open"
		}
		return err
	}

	// Success - reset failures and close circuit
	cb.failures = 0
	cb.state = "closed"
	return nil
}

// Intent Processor with multi-model support
type IntentProcessor struct {
	config          *Config
	httpClient      *http.Client
	circuitBreaker  *CircuitBreaker
	intentValidator *IntentValidator
	llmClient       llm.ClientInterface
	promptEngine    *llm.TelecomPromptEngine
}

func NewIntentProcessor(cfg *Config) *IntentProcessor {
	// Determine LLM backend URL and configuration
	llmURL := cfg.OpenAIAPIURL
	apiKey := cfg.OpenAIAPIKey
	backendType := "openai"

	if cfg.LLMBackendType == "mistral" && cfg.MistralAPIURL != "" {
		llmURL = cfg.MistralAPIURL
		apiKey = cfg.MistralAPIKey
		backendType = "mistral"
	} else if cfg.RAGEnabled && cfg.RAGAPIURL != "" {
		llmURL = cfg.RAGAPIURL
		apiKey = ""
		backendType = "rag"
	}

	// Create LLM client with proper configuration
	llmClient := llm.NewClientWithConfig(llmURL, llm.ClientConfig{
		APIKey:      apiKey,
		ModelName:   cfg.LLMModelName,
		MaxTokens:   cfg.LLMMaxTokens,
		BackendType: backendType,
		Timeout:     cfg.LLMTimeout,
	})

	return &IntentProcessor{
		config:     cfg,
		httpClient: &http.Client{Timeout: cfg.LLMTimeout},
		circuitBreaker: NewCircuitBreaker(
			cfg.CircuitBreakerThreshold,
			cfg.CircuitBreakerTimeout,
		),
		intentValidator: NewIntentValidator(),
		llmClient:       llmClient,
		promptEngine:    llm.NewTelecomPromptEngine(),
	}
}

// Intent validation
type IntentValidator struct {
	MaxLength         int
	MinLength         int
	ForbiddenPatterns []string
}

func NewIntentValidator() *IntentValidator {
	return &IntentValidator{
		MaxLength: 2048,
		MinLength: 10,
		ForbiddenPatterns: []string{
			`(?i)delete\s+all`,
			`(?i)drop\s+database`,
			`(?i)rm\s+-rf`,
		},
	}
}

func (v *IntentValidator) Validate(intent string) error {
	if len(intent) < v.MinLength {
		return fmt.Errorf("intent too short: minimum %d characters", v.MinLength)
	}
	if len(intent) > v.MaxLength {
		return fmt.Errorf("intent too long: maximum %d characters", v.MaxLength)
	}

	for _, pattern := range v.ForbiddenPatterns {
		if matched, _ := regexp.MatchString(pattern, intent); matched {
			return fmt.Errorf("intent contains forbidden pattern")
		}
	}

	return nil
}

func (p *IntentProcessor) ProcessIntent(ctx context.Context, req *NetworkIntentRequest) (*NetworkIntentResponse, error) {
	startTime := time.Now()

	// Validate input
	if err := p.intentValidator.Validate(req.Spec.Intent); err != nil {
		processingErrors.WithLabelValues("validation").Inc()
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Classify intent type using enhanced classification
	intentType := p.classifyIntent(req.Spec.Intent)

	// Extract parameters using prompt engineering
	extractedParams := p.promptEngine.ExtractParameters(req.Spec.Intent)

	// Process with circuit breaker
	var response *NetworkIntentResponse
	err := p.circuitBreaker.Call(func() error {
		var processErr error
		response, processErr = p.processWithEnhancedLLM(ctx, req, intentType, extractedParams)
		return processErr
	})

	if err != nil {
		processingErrors.WithLabelValues("llm_processing").Inc()
		return nil, err
	}

	// Add processing metadata
	response.ProcessingMetadata = ProcessingMetadata{
		ModelUsed:        p.config.LLMModelName,
		ConfidenceScore:  p.calculateConfidenceScore(req.Spec.Intent, extractedParams),
		ProcessingTimeMS: time.Since(startTime).Milliseconds(),
	}
	response.Timestamp = time.Now().Format(time.RFC3339)
	response.OriginalIntent = req.Spec.Intent

	return response, nil
}

func (p *IntentProcessor) classifyIntent(intent string) string {
	lowerIntent := strings.ToLower(intent)

	scaleIndicators := []string{"scale", "increase", "decrease", "replicas", "instances"}
	deployIndicators := []string{"deploy", "create", "setup", "configure", "install"}

	for _, indicator := range scaleIndicators {
		if strings.Contains(lowerIntent, indicator) {
			return "NetworkFunctionScale"
		}
	}

	for _, indicator := range deployIndicators {
		if strings.Contains(lowerIntent, indicator) {
			return "NetworkFunctionDeployment"
		}
	}

	return "NetworkFunctionDeployment" // Default
}

// processWithEnhancedLLM uses the enhanced LLM client with telecom-specific prompts
func (p *IntentProcessor) processWithEnhancedLLM(ctx context.Context, req *NetworkIntentRequest, intentType string, extractedParams map[string]interface{}) (*NetworkIntentResponse, error) {
	// Use the enhanced LLM client which handles fallbacks and retries internally
	responseJSON, err := p.llmClient.ProcessIntent(ctx, req.Spec.Intent)
	if err != nil {
		return nil, fmt.Errorf("LLM processing failed: %w", err)
	}

	// Parse the JSON response
	var response NetworkIntentResponse
	if err := json.Unmarshal([]byte(responseJSON), &response); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	// Enrich response with extracted parameters if LLM didn't capture them
	p.enrichResponseWithExtractedParams(&response, extractedParams)

	// Set defaults based on intent type and network function
	p.setIntelligentDefaults(&response, intentType)

	return &response, nil
}

// calculateConfidenceScore determines confidence based on parameter extraction success
func (p *IntentProcessor) calculateConfidenceScore(intent string, extractedParams map[string]interface{}) float64 {
	baseScore := 0.7 // Base confidence

	// Increase confidence based on extracted parameters
	if len(extractedParams) > 0 {
		baseScore += 0.1
	}

	// Check for specific network function mentions
	lowerIntent := strings.ToLower(intent)
	nfKeywords := []string{"upf", "amf", "smf", "pcf", "ric", "o-du", "o-cu"}
	for _, keyword := range nfKeywords {
		if strings.Contains(lowerIntent, keyword) {
			baseScore += 0.1
			break
		}
	}

	// Check for clear action words
	actionKeywords := []string{"deploy", "scale", "create", "configure", "setup"}
	for _, keyword := range actionKeywords {
		if strings.Contains(lowerIntent, keyword) {
			baseScore += 0.05
			break
		}
	}

	// Cap at 1.0
	if baseScore > 1.0 {
		baseScore = 1.0
	}

	return baseScore
}

// enrichResponseWithExtractedParams fills in missing parameters from extraction
func (p *IntentProcessor) enrichResponseWithExtractedParams(response *NetworkIntentResponse, extractedParams map[string]interface{}) {
	if response.Spec == nil {
		response.Spec = make(map[string]interface{})
	}

	// Add extracted replicas if not present
	if _, exists := response.Spec["replicas"]; !exists {
		if replicas, ok := extractedParams["replicas"]; ok {
			if replicaStr, ok := replicas.(string); ok {
				if replicaInt, err := strconv.Atoi(replicaStr); err == nil {
					response.Spec["replicas"] = replicaInt
				}
			}
		}
	}

	// Add extracted resources if not present
	if resources, exists := response.Spec["resources"].(map[string]interface{}); exists {
		if requests, ok := resources["requests"].(map[string]interface{}); ok {
			if cpu, exists := extractedParams["cpu"]; exists && requests["cpu"] == nil {
				requests["cpu"] = cpu
			}
			if memory, exists := extractedParams["memory"]; exists && requests["memory"] == nil {
				requests["memory"] = memory
			}
		}
	}

	// Set namespace from extraction if not present
	if response.Namespace == "" {
		if namespace, ok := extractedParams["namespace"].(string); ok {
			response.Namespace = namespace
		}
	}
}

// setIntelligentDefaults applies telecom-specific defaults based on context
func (p *IntentProcessor) setIntelligentDefaults(response *NetworkIntentResponse, intentType string) {
	// Set default namespace if not specified
	if response.Namespace == "" {
		response.Namespace = "default"

		// Use intelligent namespace selection based on network function
		if nf, ok := response.Spec["network_function"].(string); ok {
			switch nf {
			case "upf", "amf", "smf", "pcf", "nssf":
				response.Namespace = "5g-core"
			case "near-rt-ric", "o-du", "o-cu":
				response.Namespace = "o-ran"
			default:
				response.Namespace = "5g-core"
			}
		}
	}

	// Set default replicas for deployment if not specified
	if intentType == "NetworkFunctionDeployment" {
		if _, exists := response.Spec["replicas"]; !exists {
			response.Spec["replicas"] = 1 // Default to 1 replica
		}
	}

	// Set intelligent resource defaults based on network function type
	p.setResourceDefaults(response)
}

// setResourceDefaults sets appropriate resource limits based on NF type
func (p *IntentProcessor) setResourceDefaults(response *NetworkIntentResponse) {
	spec := response.Spec
	if spec == nil {
		return
	}

	resources, ok := spec["resources"].(map[string]interface{})
	if !ok {
		resources = make(map[string]interface{})
		spec["resources"] = resources
	}

	requests, ok := resources["requests"].(map[string]interface{})
	if !ok {
		requests = make(map[string]interface{})
		resources["requests"] = requests
	}

	limits, ok := resources["limits"].(map[string]interface{})
	if !ok {
		limits = make(map[string]interface{})
		resources["limits"] = limits
	}

	// Set defaults based on network function type if detected
	if nf, exists := spec["network_function"].(string); exists {
		switch nf {
		case "upf":
			// UPF needs more resources for packet processing
			if requests["cpu"] == nil {
				requests["cpu"] = "2000m"
			}
			if requests["memory"] == nil {
				requests["memory"] = "4Gi"
			}
			if limits["cpu"] == nil {
				limits["cpu"] = "4000m"
			}
			if limits["memory"] == nil {
				limits["memory"] = "8Gi"
			}
		case "amf", "smf":
			// Control plane functions need moderate resources
			if requests["cpu"] == nil {
				requests["cpu"] = "500m"
			}
			if requests["memory"] == nil {
				requests["memory"] = "1Gi"
			}
			if limits["cpu"] == nil {
				limits["cpu"] = "1000m"
			}
			if limits["memory"] == nil {
				limits["memory"] = "2Gi"
			}
		case "near-rt-ric":
			// Near-RT RIC needs resources for real-time processing
			if requests["cpu"] == nil {
				requests["cpu"] = "1000m"
			}
			if requests["memory"] == nil {
				requests["memory"] = "2Gi"
			}
			if limits["cpu"] == nil {
				limits["cpu"] = "2000m"
			}
			if limits["memory"] == nil {
				limits["memory"] = "4Gi"
			}
		default:
			// Generic defaults
			if requests["cpu"] == nil {
				requests["cpu"] = "100m"
			}
			if requests["memory"] == nil {
				requests["memory"] = "256Mi"
			}
			if limits["cpu"] == nil {
				limits["cpu"] = "500m"
			}
			if limits["memory"] == nil {
				limits["memory"] = "512Mi"
			}
		}
	} else {
		// No specific NF detected, use generic defaults
		if requests["cpu"] == nil {
			requests["cpu"] = "100m"
		}
		if requests["memory"] == nil {
			requests["memory"] = "256Mi"
		}
		if limits["cpu"] == nil {
			limits["cpu"] = "500m"
		}
		if limits["memory"] == nil {
			limits["memory"] = "512Mi"
		}
	}
}

// Legacy method maintained for backward compatibility
func (p *IntentProcessor) processWithLLM(ctx context.Context, req *NetworkIntentRequest, intentType string) (*NetworkIntentResponse, error) {
	// Try primary model (Mistral) first, fallback to OpenAI
	if p.config.LLMBackendType == "mistral" && p.config.MistralAPIURL != "" {
		response, err := p.callMistralAPI(ctx, req, intentType)
		if err == nil {
			return response, nil
		}
		log.Printf("Mistral API failed, falling back to OpenAI: %v", err)
	}

	// Fallback to OpenAI or RAG API
	if p.config.RAGEnabled && p.config.RAGAPIURL != "" {
		return p.callRAGAPI(ctx, req, intentType)
	}

	return p.callOpenAIAPI(ctx, req, intentType)
}

func (p *IntentProcessor) callMistralAPI(ctx context.Context, req *NetworkIntentRequest, intentType string) (*NetworkIntentResponse, error) {
	systemPrompt := p.buildSystemPrompt(intentType)

	requestBody := map[string]interface{}{
		"model": p.config.LLMModelName,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": req.Spec.Intent},
		},
		"max_tokens":      p.config.LLMMaxTokens,
		"temperature":     0.0,
		"response_format": map[string]string{"type": "json_object"},
	}

	reqBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Mistral request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.MistralAPIURL, bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create Mistral request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.config.MistralAPIKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call Mistral API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Mistral API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var mistralResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&mistralResp); err != nil {
		return nil, fmt.Errorf("failed to decode Mistral response: %w", err)
	}

	if len(mistralResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in Mistral response")
	}

	var result NetworkIntentResponse
	if err := json.Unmarshal([]byte(mistralResp.Choices[0].Message.Content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse Mistral response JSON: %w", err)
	}

	return &result, nil
}

func (p *IntentProcessor) callRAGAPI(ctx context.Context, req *NetworkIntentRequest, intentType string) (*NetworkIntentResponse, error) {
	ragRequest := map[string]string{
		"intent": req.Spec.Intent,
	}

	reqBodyBytes, err := json.Marshal(ragRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RAG request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.RAGAPIURL, bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create RAG request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: p.config.RAGTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		ragAPIRequests.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("failed to call RAG API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ragAPIRequests.WithLabelValues("error").Inc()
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("RAG API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	ragAPIRequests.WithLabelValues("success").Inc()

	var result NetworkIntentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode RAG response: %w", err)
	}

	return &result, nil
}

func (p *IntentProcessor) callOpenAIAPI(ctx context.Context, req *NetworkIntentRequest, intentType string) (*NetworkIntentResponse, error) {
	systemPrompt := p.buildSystemPrompt(intentType)

	requestBody := map[string]interface{}{
		"model": p.config.LLMModelName,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": req.Spec.Intent},
		},
		"max_tokens":      p.config.LLMMaxTokens,
		"temperature":     0.0,
		"response_format": map[string]string{"type": "json_object"},
	}

	reqBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OpenAI request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.OpenAIAPIURL, bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.config.OpenAIAPIKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var openaiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return nil, fmt.Errorf("failed to decode OpenAI response: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in OpenAI response")
	}

	var result NetworkIntentResponse
	if err := json.Unmarshal([]byte(openaiResp.Choices[0].Message.Content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI response JSON: %w", err)
	}

	return &result, nil
}

func (p *IntentProcessor) buildSystemPrompt(intentType string) string {
	basePrompt := `You are an expert telecommunications network engineer with deep knowledge of:
- 5G Core Network Functions (AMF, SMF, UPF, NSSF, PCF)
- O-RAN architecture (O-DU, O-CU, Near-RT RIC, Non-RT RIC)
- Network slicing and Quality of Service (QoS) management
- Edge computing and Multi-access Edge Computing (MEC)
- Kubernetes and cloud-native network functions

Your task is to translate natural language network operations requests into structured JSON objects that can be used to configure and deploy network functions.

Output MUST be valid JSON matching this schema:`

	if intentType == "NetworkFunctionScale" {
		return basePrompt + `
{
  "type": "NetworkFunctionScale",
  "name": "existing-nf-name",
  "namespace": "target-namespace",
  "replicas": integer
}`
	}

	return basePrompt + `
{
  "type": "NetworkFunctionDeployment",
  "name": "descriptive-nf-name",
  "namespace": "target-namespace",
  "spec": {
    "replicas": integer,
    "image": "container-image-url",
    "resources": {
      "requests": {"cpu": "100m", "memory": "256Mi"},
      "limits": {"cpu": "500m", "memory": "512Mi"}
    },
    "ports": [{"containerPort": 8080, "protocol": "TCP"}],
    "env": [{"name": "ENV_VAR", "value": "value"}]
  },
  "o1_config": "XML configuration for FCAPS operations",
  "a1_policy": {
    "policy_type_id": "string",
    "policy_data": {}
  }
}`
}

// HTTP handlers
// chainMiddleware applies multiple middleware functions to a handler
func chainMiddleware(h http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// infoHandler provides detailed service information
func infoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, "Only GET method is allowed", "METHOD_NOT_ALLOWED", "", http.StatusMethodNotAllowed, nil)
		return
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	info := map[string]interface{}{
		"service": map[string]interface{}{
			"name":           "LLM Processor",
			"version":        config.ServiceVersion,
			"build_time":     startTime.Format(time.RFC3339),
			"uptime":         time.Since(startTime).String(),
			"uptime_seconds": int64(time.Since(startTime).Seconds()),
		},
		"configuration": map[string]interface{}{
			"llm_backend":      config.LLMBackendType,
			"llm_model":        config.LLMModelName,
			"rag_enabled":      config.RAGEnabled,
			"metrics_enabled":  config.MetricsEnabled,
			"tracing_enabled":  config.TracingEnabled,
			"cors_enabled":     config.CORSEnabled,
			"api_key_required": config.APIKeyRequired,
		},
		"runtime": map[string]interface{}{
			"go_version":   runtime.Version(),
			"go_os":        runtime.GOOS,
			"go_arch":      runtime.GOARCH,
			"cpu_count":    runtime.NumCPU(),
			"goroutines":   runtime.NumGoroutine(),
			"memory_alloc": bToMb(m.Alloc),
			"memory_total": bToMb(m.TotalAlloc),
			"memory_sys":   bToMb(m.Sys),
			"gc_runs":      m.NumGC,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(info)
}

// versionHandler provides version information
func versionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, "Only GET method is allowed", "METHOD_NOT_ALLOWED", "", http.StatusMethodNotAllowed, nil)
		return
	}

	versionInfo := map[string]interface{}{
		"version":    config.ServiceVersion,
		"build_time": startTime.Format(time.RFC3339),
		"go_version": runtime.Version(),
		"git_commit": getEnvOrDefault("GIT_COMMIT", "unknown"),
		"git_branch": getEnvOrDefault("GIT_BRANCH", "unknown"),
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(versionInfo)
}

// bToMb converts bytes to megabytes
func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func processHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	defer func() {
		requestDuration.WithLabelValues(r.Method).Observe(time.Since(startTime).Seconds())
	}()

	if r.Method != http.MethodPost {
		requestsTotal.WithLabelValues(r.Method, "405").Inc()
		writeErrorResponse(w, "Only POST method is allowed", "METHOD_NOT_ALLOWED", "", http.StatusMethodNotAllowed, nil)
		return
	}

	// Get request ID from context
	requestID := r.Context().Value("request_id")
	if requestID == nil {
		requestID = "unknown"
	}

	logger.Debug("Processing intent request",
		slog.String("request_id", requestID.(string)),
		slog.String("content_type", r.Header.Get("Content-Type")),
	)

	var req NetworkIntentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		requestsTotal.WithLabelValues(r.Method, "400").Inc()
		logger.Error("Failed to decode request body",
			slog.String("request_id", requestID.(string)),
			slog.String("error", err.Error()),
		)
		writeErrorResponse(w, "Invalid request body", "INVALID_REQUEST", "", http.StatusBadRequest, map[string]string{
			"stage":   "validation",
			"details": err.Error(),
		})
		return
	}

	logger.Info("Processing intent",
		slog.String("request_id", requestID.(string)),
		slog.String("intent", req.Spec.Intent),
		slog.String("namespace", req.Metadata.Namespace),
	)

	response, err := processor.ProcessIntent(r.Context(), &req)
	if err != nil {
		requestsTotal.WithLabelValues(r.Method, "500").Inc()
		logger.Error("Intent processing failed",
			slog.String("request_id", requestID.(string)),
			slog.String("error", err.Error()),
			slog.String("intent", req.Spec.Intent),
		)
		writeErrorResponse(w, err.Error(), "PROCESSING_FAILED", req.Spec.Intent, http.StatusInternalServerError, map[string]string{
			"stage":      "llm_processing",
			"request_id": requestID.(string),
		})
		return
	}

	requestsTotal.WithLabelValues(r.Method, "200").Inc()
	logger.Info("Intent processed successfully",
		slog.String("request_id", requestID.(string)),
		slog.String("response_type", response.Type),
		slog.String("response_name", response.Name),
		slog.Float64("confidence_score", response.ProcessingMetadata.ConfidenceScore),
		slog.Int64("processing_time_ms", response.ProcessingMetadata.ProcessingTimeMS),
	)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID.(string))
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	healthMux.RLock()
	defer healthMux.RUnlock()

	status := "ok"
	statusCode := http.StatusOK

	if !healthy {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	response := HealthResponse{
		Status:        status,
		Timestamp:     time.Now().Format(time.RFC3339),
		Version:       config.ServiceVersion,
		UptimeSeconds: int64(time.Since(startTime).Seconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

func readyzHandler(w http.ResponseWriter, r *http.Request) {
	readyMux.RLock()
	defer readyMux.RUnlock()

	status := "ready"
	statusCode := http.StatusOK
	dependencies := make(map[string]string)

	// Check RAG API if enabled
	if config.RAGEnabled {
		if err := checkRAGAPI(); err != nil {
			dependencies["rag_api"] = "unavailable"
			status = "not_ready"
			statusCode = http.StatusServiceUnavailable
		} else {
			dependencies["rag_api"] = "available"
		}
	}

	// Check LLM backend
	if err := checkLLMBackend(); err != nil {
		dependencies["llm_backend"] = "unavailable"
		status = "not_ready"
		statusCode = http.StatusServiceUnavailable
	} else {
		dependencies["llm_backend"] = "available"
	}

	if !ready {
		status = "not_ready"
		statusCode = http.StatusServiceUnavailable
	}

	response := ReadinessResponse{
		Status:       status,
		Timestamp:    time.Now().Format(time.RFC3339),
		Dependencies: dependencies,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

func writeErrorResponse(w http.ResponseWriter, message, code, originalIntent string, statusCode int, details map[string]string) {
	response := ErrorResponse{
		Error:          message,
		ErrorCode:      code,
		OriginalIntent: originalIntent,
		Timestamp:      time.Now().Format(time.RFC3339),
		Details:        details,
	}

	if statusCode == http.StatusTooManyRequests {
		response.RetryAfterSeconds = 60
		w.Header().Set("Retry-After", "60")
	}

	// Add request ID if available
	if details != nil {
		if requestID, exists := details["request_id"]; exists {
			w.Header().Set("X-Request-ID", requestID)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

func checkRAGAPI() error {
	if config.RAGAPIURL == "" {
		return fmt.Errorf("RAG API URL not configured")
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(config.RAGAPIURL + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("RAG API returned status %d", resp.StatusCode)
	}

	return nil
}

func checkLLMBackend() error {
	// For now, just check if the configuration is set
	if config.LLMBackendType == "mistral" && config.MistralAPIURL == "" {
		return fmt.Errorf("Mistral API URL not configured")
	}
	if config.LLMBackendType == "openai" && config.OpenAIAPIURL == "" {
		return fmt.Errorf("OpenAI API URL not configured")
	}
	return nil
}

func loadConfig() *Config {
	return &Config{
		// Service Configuration
		Port:             getEnvOrDefault("PORT", "8080"),
		LogLevel:         getEnvOrDefault("LOG_LEVEL", "info"),
		ServiceVersion:   getEnvOrDefault("SERVICE_VERSION", "v1.0.0"),
		GracefulShutdown: getEnvDuration("GRACEFUL_SHUTDOWN_TIMEOUT", 30*time.Second),

		// LLM Configuration
		LLMBackendType: getEnvOrDefault("LLM_BACKEND_TYPE", "openai"),
		LLMAPIKey:      os.Getenv("LLM_API_KEY"),
		LLMModelName:   getEnvOrDefault("LLM_MODEL_NAME", "gpt-4o-mini"),
		LLMTimeout:     getEnvDuration("LLM_TIMEOUT", 60*time.Second),
		LLMMaxTokens:   getEnvInt("LLM_MAX_TOKENS", 2048),

		// Mistral Configuration
		MistralAPIURL: getEnvOrDefault("MISTRAL_API_URL", ""),
		MistralAPIKey: os.Getenv("MISTRAL_API_KEY"),

		// OpenAI Configuration
		OpenAIAPIURL: getEnvOrDefault("OPENAI_API_URL", "https://api.openai.com/v1/chat/completions"),
		OpenAIAPIKey: os.Getenv("OPENAI_API_KEY"),

		// RAG API Configuration
		RAGAPIURL:  getEnvOrDefault("RAG_API_URL", "http://rag-api.default.svc.cluster.local:5001/process_intent"),
		RAGTimeout: getEnvDuration("RAG_TIMEOUT", 30*time.Second),
		RAGEnabled: getEnvBool("RAG_ENABLED", true),

		// Circuit Breaker Configuration
		CircuitBreakerEnabled:   getEnvBool("CIRCUIT_BREAKER_ENABLED", true),
		CircuitBreakerThreshold: getEnvInt("CIRCUIT_BREAKER_THRESHOLD", 5),
		CircuitBreakerTimeout:   getEnvDuration("CIRCUIT_BREAKER_TIMEOUT", 60*time.Second),

		// Retry Configuration
		MaxRetries:   getEnvInt("MAX_RETRIES", 3),
		RetryDelay:   getEnvDuration("RETRY_DELAY", 1*time.Second),
		RetryBackoff: getEnvOrDefault("RETRY_BACKOFF", "exponential"),

		// Rate Limiting
		RateLimitRPM:   getEnvInt("RATE_LIMIT_REQUESTS_PER_MINUTE", 60),
		RateLimitBurst: getEnvInt("RATE_LIMIT_BURST", 10),

		// Monitoring
		MetricsEnabled: getEnvBool("METRICS_ENABLED", true),
		TracingEnabled: getEnvBool("TRACING_ENABLED", false),
		JaegerEndpoint: getEnvOrDefault("JAEGER_ENDPOINT", ""),

		// Security
		APIKeyRequired: getEnvBool("API_KEY_REQUIRED", false),
		APIKey:         os.Getenv("API_KEY"),
		CORSEnabled:    getEnvBool("CORS_ENABLED", true),
		AllowedOrigins: getEnvStringSlice("ALLOWED_ORIGINS", []string{"*"}),

		// Request Logging
		RequestLogEnabled: getEnvBool("REQUEST_LOG_ENABLED", true),
		RequestLogLevel:   getEnvOrDefault("REQUEST_LOG_LEVEL", "info"),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

// initLogger initializes structured logging
func initLogger(logLevel string) *slog.Logger {
	level := slog.LevelInfo
	switch strings.ToLower(logLevel) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

// Middleware for request logging
func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := fmt.Sprintf("req-%d", time.Now().UnixNano())

		// Add request ID to context
		ctx := context.WithValue(r.Context(), "request_id", requestID)
		r = r.WithContext(ctx)

		// Create custom response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		logger.Info("Request started",
			slog.String("request_id", requestID),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("user_agent", r.UserAgent()),
		)

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		logger.Info("Request completed",
			slog.String("request_id", requestID),
			slog.Int("status_code", wrapped.statusCode),
			slog.Duration("duration", duration),
			slog.Int64("duration_ms", duration.Milliseconds()),
		)
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// CORS middleware
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if config.CORSEnabled {
			origin := r.Header.Get("Origin")
			allowed := false

			for _, allowedOrigin := range config.AllowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}

		next.ServeHTTP(w, r)
	}
}

// Authentication middleware
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if config.APIKeyRequired {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				apiKey = r.Header.Get("Authorization")
				if strings.HasPrefix(apiKey, "Bearer ") {
					apiKey = strings.TrimPrefix(apiKey, "Bearer ")
				}
			}

			if apiKey != config.APIKey {
				logger.Warn("Unauthorized request",
					slog.String("path", r.URL.Path),
					slog.String("remote_addr", r.RemoteAddr),
				)
				writeErrorResponse(w, "Unauthorized", "UNAUTHORIZED", "", http.StatusUnauthorized, nil)
				return
			}
		}

		next.ServeHTTP(w, r)
	}
}

// Rate limiting middleware (simple in-memory implementation)
type rateLimiter struct {
	clients map[string]*clientLimiter
	mutex   sync.RWMutex
}

type clientLimiter struct {
	lastSeen time.Time
	count    int
}

func newRateLimiter() *rateLimiter {
	rl := &rateLimiter{
		clients: make(map[string]*clientLimiter),
	}

	// Cleanup routine
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			rl.cleanup()
		}
	}()

	return rl
}

func (rl *rateLimiter) cleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	for ip, limiter := range rl.clients {
		if now.Sub(limiter.lastSeen) > time.Hour {
			delete(rl.clients, ip)
		}
	}
}

func (rl *rateLimiter) allow(ip string, limit int) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	limiter, exists := rl.clients[ip]

	if !exists {
		rl.clients[ip] = &clientLimiter{
			lastSeen: now,
			count:    1,
		}
		return true
	}

	// Reset count if more than a minute has passed
	if now.Sub(limiter.lastSeen) > time.Minute {
		limiter.count = 1
		limiter.lastSeen = now
		return true
	}

	limiter.lastSeen = now
	if limiter.count >= limit {
		return false
	}

	limiter.count++
	return true
}

var globalRateLimiter = newRateLimiter()

func rateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if config.RateLimitRPM > 0 {
			clientIP := getClientIP(r)

			if !globalRateLimiter.allow(clientIP, config.RateLimitRPM) {
				logger.Warn("Rate limit exceeded",
					slog.String("client_ip", clientIP),
					slog.String("path", r.URL.Path),
				)
				writeErrorResponse(w, "Rate limit exceeded", "RATE_LIMIT_EXCEEDED", "", http.StatusTooManyRequests, map[string]string{
					"retry_after": "60",
				})
				return
			}
		}

		next.ServeHTTP(w, r)
	}
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Use remote address
	if strings.Contains(r.RemoteAddr, ":") {
		host, _, _ := strings.Cut(r.RemoteAddr, ":")
		return host
	}

	return r.RemoteAddr
}

func init() {
	// Register Prometheus metrics
	prometheus.MustRegister(requestsTotal)
	prometheus.MustRegister(requestDuration)
	prometheus.MustRegister(ragAPIRequests)
	prometheus.MustRegister(processingErrors)
}

func main() {
	// Load configuration
	config = loadConfig()

	// Initialize structured logging
	logger = initLogger(config.LogLevel)
	slog.SetDefault(logger)

	// Initialize intent processor
	processor = NewIntentProcessor(config)

	// Set up HTTP routes with middleware
	http.HandleFunc("/process", chainMiddleware(processHandler, loggingMiddleware, corsMiddleware, authMiddleware, rateLimitMiddleware))
	http.HandleFunc("/healthz", chainMiddleware(healthzHandler, loggingMiddleware, corsMiddleware))
	http.HandleFunc("/readyz", chainMiddleware(readyzHandler, loggingMiddleware, corsMiddleware))
	http.HandleFunc("/info", chainMiddleware(infoHandler, loggingMiddleware, corsMiddleware))
	http.HandleFunc("/version", chainMiddleware(versionHandler, loggingMiddleware, corsMiddleware))

	if config.MetricsEnabled {
		http.Handle("/metrics", promhttp.Handler())
	}

	// Mark service as ready
	readyMux.Lock()
	ready = true
	readyMux.Unlock()

	// Start server
	server := &http.Server{
		Addr:    ":" + config.Port,
		Handler: nil,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		logger.Info("Received shutdown signal, starting graceful shutdown")

		// Mark as not ready and unhealthy
		readyMux.Lock()
		ready = false
		readyMux.Unlock()

		healthMux.Lock()
		healthy = false
		healthMux.Unlock()

		// Shutdown server
		ctx, cancel := context.WithTimeout(context.Background(), config.GracefulShutdown)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			logger.Error("Server forced to shutdown", slog.String("error", err.Error()))
		} else {
			logger.Info("Server shutdown completed successfully")
		}
	}()

	logger.Info("Starting LLM Processor server",
		slog.String("port", config.Port),
		slog.String("version", config.ServiceVersion),
		slog.String("llm_backend", config.LLMBackendType),
		slog.String("llm_model", config.LLMModelName),
		slog.String("rag_api", config.RAGAPIURL),
		slog.Bool("rag_enabled", config.RAGEnabled),
		slog.Bool("metrics_enabled", config.MetricsEnabled),
		slog.Bool("cors_enabled", config.CORSEnabled),
		slog.Bool("api_key_required", config.APIKeyRequired),
	)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("Failed to start server", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("Server stopped successfully")
}
