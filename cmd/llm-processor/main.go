package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	config     *Config
	processor  *IntentProcessor
	startTime  = time.Now()
	healthMux  sync.RWMutex
	healthy    = true
	readyMux   sync.RWMutex
	ready      = false
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
	RAGAPIURL    string
	RAGTimeout   time.Duration
	RAGEnabled   bool

	// Circuit Breaker Configuration
	CircuitBreakerEnabled    bool
	CircuitBreakerThreshold  int
	CircuitBreakerTimeout    time.Duration

	// Retry Configuration
	MaxRetries    int
	RetryDelay    time.Duration
	RetryBackoff  string

	// Rate Limiting
	RateLimitRPM   int
	RateLimitBurst int

	// Monitoring
	MetricsEnabled bool
	TracingEnabled bool
	JaegerEndpoint string
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
	Type                string                 `json:"type"`
	Name                string                 `json:"name"`
	Namespace           string                 `json:"namespace"`
	Spec                map[string]interface{} `json:"spec"`
	O1Config            string                 `json:"o1_config,omitempty"`
	A1Policy            map[string]interface{} `json:"a1_policy,omitempty"`
	OriginalIntent      string                 `json:"original_intent"`
	Timestamp           string                 `json:"timestamp"`
	ProcessingMetadata  ProcessingMetadata     `json:"processing_metadata"`
}

type ProcessingMetadata struct {
	ModelUsed        string  `json:"model_used"`
	ConfidenceScore  float64 `json:"confidence_score"`
	ProcessingTimeMS int64   `json:"processing_time_ms"`
}

type ErrorResponse struct {
	Error            string            `json:"error"`
	ErrorCode        string            `json:"error_code"`
	OriginalIntent   string            `json:"original_intent"`
	Timestamp        string            `json:"timestamp"`
	RetryAfterSeconds int              `json:"retry_after_seconds,omitempty"`
	Details          map[string]string `json:"details,omitempty"`
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
	failures  int
	lastFailure time.Time
	state     string // "closed", "open", "half-open"
	threshold int
	timeout   time.Duration
	mutex     sync.RWMutex
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
}

func NewIntentProcessor(cfg *Config) *IntentProcessor {
	return &IntentProcessor{
		config:     cfg,
		httpClient: &http.Client{Timeout: cfg.LLMTimeout},
		circuitBreaker: NewCircuitBreaker(
			cfg.CircuitBreakerThreshold,
			cfg.CircuitBreakerTimeout,
		),
		intentValidator: NewIntentValidator(),
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

	// Classify intent type
	intentType := p.classifyIntent(req.Spec.Intent)

	// Process with circuit breaker
	var response *NetworkIntentResponse
	err := p.circuitBreaker.Call(func() error {
		var processErr error
		response, processErr = p.processWithLLM(ctx, req, intentType)
		return processErr
	})

	if err != nil {
		processingErrors.WithLabelValues("llm_processing").Inc()
		return nil, err
	}

	// Add processing metadata
	response.ProcessingMetadata = ProcessingMetadata{
		ModelUsed:        p.config.LLMModelName,
		ConfidenceScore:  0.95, // This would come from actual LLM response
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

	var req NetworkIntentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		requestsTotal.WithLabelValues(r.Method, "400").Inc()
		writeErrorResponse(w, "Invalid request body", "INVALID_REQUEST", "", http.StatusBadRequest, map[string]string{
			"stage": "validation",
			"details": err.Error(),
		})
		return
	}

	response, err := processor.ProcessIntent(r.Context(), &req)
	if err != nil {
		requestsTotal.WithLabelValues(r.Method, "500").Inc()
		writeErrorResponse(w, err.Error(), "PROCESSING_FAILED", req.Spec.Intent, http.StatusInternalServerError, map[string]string{
			"stage": "llm_processing",
		})
		return
	}

	requestsTotal.WithLabelValues(r.Method, "200").Inc()
	w.Header().Set("Content-Type", "application/json")
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

	// Initialize intent processor
	processor = NewIntentProcessor(config)

	// Set up HTTP routes
	http.HandleFunc("/process", processHandler)
	http.HandleFunc("/healthz", healthzHandler)
	http.HandleFunc("/readyz", readyzHandler)

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

		log.Println("Shutting down gracefully...")

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
			log.Printf("Server forced to shutdown: %v", err)
		}
	}()

	log.Printf("Starting LLM Processor server on :%s", config.Port)
	log.Printf("Service version: %s", config.ServiceVersion)
	log.Printf("LLM Backend: %s", config.LLMBackendType)
	log.Printf("RAG API: %s (enabled: %v)", config.RAGAPIURL, config.RAGEnabled)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Println("Server stopped")
}