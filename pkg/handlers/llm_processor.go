package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/thc1006/nephoran-intent-operator/internal/ingest"
	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/health"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"
)

// LLMProcessorHandler handles HTTP requests for the LLM processor service.

type LLMProcessorHandler struct {
	config *config.LLMProcessorConfig

	processor *IntentProcessor

	streamingProcessor interface {
		HandleStreamingRequest(w http.ResponseWriter, r *http.Request, req *llm.StreamingRequest) error

		GetMetrics() map[string]interface{}
	}

	circuitBreakerMgr *llm.CircuitBreakerManager

	tokenManager llm.TokenManager

	contextBuilder *llm.ContextBuilder

	relevanceScorer *llm.RelevanceScorer

	promptBuilder interface{} // Stub for RAG aware prompt builder

	logger *slog.Logger

	healthChecker *health.HealthChecker

	startTime time.Time

	metricsCollector *monitoring.MetricsCollector
}

// Request/Response structures.

type ProcessIntentRequest struct {
	Intent string `json:"intent"`

	Metadata map[string]string `json:"metadata,omitempty"`
}

// ProcessIntentResponse represents a processintentresponse.

type ProcessIntentResponse struct {
	Result string `json:"result"`

	ProcessingTime string `json:"processing_time"`

	RequestID string `json:"request_id"`

	ServiceVersion string `json:"service_version"`

	Metadata map[string]interface{} `json:"metadata,omitempty"`

	Status string `json:"status"`

	Error string `json:"error,omitempty"`
}

// HealthResponse represents a healthresponse.

type HealthResponse struct {
	Status string `json:"status"`

	Version string `json:"version"`

	Uptime string `json:"uptime"`

	Timestamp string `json:"timestamp"`
}

// ProcessIntentResult represents the result of intent processing.

type ProcessIntentResult struct {
	Result string `json:"result"`

	RequestID string `json:"request_id"`

	ProcessingTime time.Duration `json:"processing_time"`

	Metadata map[string]interface{} `json:"metadata,omitempty"`

	Status string `json:"status"`
}

// IntentProcessor handles the LLM processing logic with RAG enhancement.

type IntentProcessor struct {
	LLMClient *llm.Client

	RAGEnhancedClient interface{} // Stub for RAG enhanced processor

	CircuitBreaker *llm.CircuitBreaker

	Logger *slog.Logger
}

// NewLLMProcessorHandler creates a new handler instance.

func NewLLMProcessorHandler(

	config *config.LLMProcessorConfig,

	processor *IntentProcessor,

	streamingProcessor interface {
		HandleStreamingRequest(w http.ResponseWriter, r *http.Request, req *llm.StreamingRequest) error

		GetMetrics() map[string]interface{}
	},

	circuitBreakerMgr *llm.CircuitBreakerManager,

	tokenManager llm.TokenManager,

	contextBuilder *llm.ContextBuilder,

	relevanceScorer *llm.RelevanceScorer,

	promptBuilder interface{}, // Stub for RAG aware prompt builder

	logger *slog.Logger,

	healthChecker *health.HealthChecker,

	startTime time.Time,

) *LLMProcessorHandler {

	return NewLLMProcessorHandlerWithMetrics(

		config,

		processor,

		streamingProcessor,

		circuitBreakerMgr,

		tokenManager,

		contextBuilder,

		relevanceScorer,

		promptBuilder,

		logger,

		healthChecker,

		startTime,

		monitoring.NewMetricsCollector(),
	)

}

// NewLLMProcessorHandlerWithMetrics creates a new handler instance with a provided metrics collector.

func NewLLMProcessorHandlerWithMetrics(

	config *config.LLMProcessorConfig,

	processor *IntentProcessor,

	streamingProcessor interface {
		HandleStreamingRequest(w http.ResponseWriter, r *http.Request, req *llm.StreamingRequest) error

		GetMetrics() map[string]interface{}
	},

	circuitBreakerMgr *llm.CircuitBreakerManager,

	tokenManager llm.TokenManager,

	contextBuilder *llm.ContextBuilder,

	relevanceScorer *llm.RelevanceScorer,

	promptBuilder interface{}, // Stub for RAG aware prompt builder

	logger *slog.Logger,

	healthChecker *health.HealthChecker,

	startTime time.Time,

	metricsCollector *monitoring.MetricsCollector,

) *LLMProcessorHandler {

	return &LLMProcessorHandler{

		config: config,

		processor: processor,

		streamingProcessor: streamingProcessor,

		circuitBreakerMgr: circuitBreakerMgr,

		tokenManager: tokenManager,

		contextBuilder: contextBuilder,

		relevanceScorer: relevanceScorer,

		promptBuilder: promptBuilder,

		logger: logger,

		healthChecker: healthChecker,

		startTime: startTime,

		metricsCollector: metricsCollector,
	}

}

// ProcessIntentHandler handles intent processing requests.

func (h *LLMProcessorHandler) ProcessIntentHandler(w http.ResponseWriter, r *http.Request) {

	// Start timing for metrics.

	handlerStartTime := time.Now()

	var statusCode int = http.StatusOK // Default to OK, will be overridden on error

	// Ensure metrics are recorded when handler exits.

	defer func() {

		duration := time.Since(handlerStartTime)

		h.metricsCollector.RecordHTTPRequest(

			r.Method,

			"/api/v1/process-intent",

			strconv.Itoa(statusCode),

			duration,
		)

	}()

	if r.Method != http.MethodPost {

		statusCode = http.StatusMethodNotAllowed

		h.writeErrorResponse(w, "Method not allowed", statusCode, "")

		return

	}

	startTime := time.Now()

	reqID := fmt.Sprintf("%d", time.Now().UnixNano())

	h.logger.Info("Processing intent request", slog.String("request_id", reqID))

	var req ProcessIntentRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		h.logger.Error("Failed to decode request", slog.String("error", err.Error()))

		statusCode = http.StatusBadRequest

		h.writeErrorResponse(w, "Invalid request body", statusCode, reqID)

		return

	}

	if req.Intent == "" {

		h.logger.Error("Empty intent provided")

		statusCode = http.StatusBadRequest

		h.writeErrorResponse(w, "Intent is required", statusCode, reqID)

		return

	}

	// Process intent with context cancellation support.

	ctx, cancel := context.WithTimeout(r.Context(), h.config.RequestTimeout)

	defer cancel()

	result, err := h.processor.ProcessIntent(ctx, req.Intent, req.Metadata)

	if err != nil {

		h.logger.Error("Failed to process intent",

			slog.String("error", err.Error()),

			slog.String("intent", req.Intent),
		)

		statusCode = http.StatusInternalServerError

		response := ProcessIntentResponse{

			Status: "error",

			Error: err.Error(),

			RequestID: reqID,

			ServiceVersion: h.config.ServiceVersion,

			ProcessingTime: time.Since(startTime).String(),
		}

		h.writeJSONResponse(w, response, statusCode)

		return

	}

	response := ProcessIntentResponse{

		Result: result.Result,

		Status: result.Status,

		ProcessingTime: time.Since(startTime).String(),

		RequestID: reqID,

		ServiceVersion: h.config.ServiceVersion,

		Metadata: result.Metadata,
	}

	h.writeJSONResponse(w, response, http.StatusOK)

	h.logger.Info("Intent processed successfully",

		slog.String("request_id", reqID),

		slog.Duration("processing_time", time.Since(startTime)),
	)

}

// StatusHandler returns service status information.

func (h *LLMProcessorHandler) StatusHandler(w http.ResponseWriter, r *http.Request) {

	// Start timing for metrics.

	handlerStartTime := time.Now()

	statusCode := http.StatusOK

	// Ensure metrics are recorded when handler exits.

	defer func() {

		duration := time.Since(handlerStartTime)

		h.metricsCollector.RecordHTTPRequest(

			r.Method,

			"/api/v1/status",

			strconv.Itoa(statusCode),

			duration,
		)

	}()

	status := map[string]interface{}{

		"service": "llm-processor",

		"version": h.config.ServiceVersion,

		"uptime": time.Since(h.startTime).String(),

		"healthy": h.healthChecker.IsHealthy(),

		"ready": h.healthChecker.IsReady(),

		"backend_type": h.config.LLMBackendType,

		"model": h.config.LLMModelName,

		"rag_enabled": h.config.RAGEnabled,

		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	h.writeJSONResponse(w, status, http.StatusOK)

}

// StreamingHandler handles Server-Sent Events streaming requests.

func (h *LLMProcessorHandler) StreamingHandler(w http.ResponseWriter, r *http.Request) {

	// Start timing for HTTP request metrics.

	handlerStartTime := time.Now()

	var statusCode int = http.StatusOK

	// Start timing for SSE stream duration.

	streamStartTime := time.Now()

	streamRoute := "/api/v1/stream"

	// Ensure HTTP metrics are recorded when handler exits.

	defer func() {

		// Record HTTP request metrics.

		duration := time.Since(handlerStartTime)

		h.metricsCollector.RecordHTTPRequest(

			r.Method,

			streamRoute,

			strconv.Itoa(statusCode),

			duration,
		)

		// Record SSE stream duration only if streaming actually occurred.

		if statusCode == http.StatusOK {

			streamDuration := time.Since(streamStartTime)

			h.metricsCollector.RecordSSEStream(streamRoute, streamDuration)

		}

	}()

	if r.Method != http.MethodPost {

		statusCode = http.StatusMethodNotAllowed

		h.writeErrorResponse(w, "Method not allowed", statusCode, "")

		return

	}

	if h.streamingProcessor == nil {

		statusCode = http.StatusServiceUnavailable

		h.writeErrorResponse(w, "Streaming not enabled", statusCode, "")

		return

	}

	var req llm.StreamingRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

		h.logger.Error("Failed to decode streaming request", slog.String("error", err.Error()))

		statusCode = http.StatusBadRequest

		h.writeErrorResponse(w, "Invalid request body", statusCode, "")

		return

	}

	if req.Query == "" {

		statusCode = http.StatusBadRequest

		h.writeErrorResponse(w, "Query is required", statusCode, "")

		return

	}

	// Set defaults.

	if req.ModelName == "" {

		req.ModelName = h.config.LLMModelName

	}

	if req.MaxTokens == 0 {

		req.MaxTokens = h.config.LLMMaxTokens

	}

	h.logger.Info("Starting streaming request",

		"query", req.Query,

		"model", req.ModelName,

		"enable_rag", req.EnableRAG,
	)

	// Add timeout context for streaming.

	ctx, cancel := context.WithTimeout(r.Context(), h.config.StreamTimeout)

	defer cancel()

	// Update request with context.

	r = r.WithContext(ctx)

	err := h.streamingProcessor.HandleStreamingRequest(w, r, &req)

	if err != nil {

		h.logger.Error("Streaming request failed", slog.String("error", err.Error()))

		// Error handling is done within HandleStreamingRequest.

		// Set status code to indicate error for metrics.

		statusCode = http.StatusInternalServerError

	}

}

// MetricsHandler provides comprehensive metrics.

func (h *LLMProcessorHandler) MetricsHandler(w http.ResponseWriter, r *http.Request) {

	// Start timing for metrics.

	handlerStartTime := time.Now()

	statusCode := http.StatusOK

	// Ensure metrics are recorded when handler exits.

	defer func() {

		duration := time.Since(handlerStartTime)

		h.metricsCollector.RecordHTTPRequest(

			r.Method,

			"/api/v1/metrics",

			strconv.Itoa(statusCode),

			duration,
		)

	}()

	metrics := map[string]interface{}{

		"service": "llm-processor",

		"version": h.config.ServiceVersion,

		"uptime": time.Since(h.startTime).String(),
	}

	// Add token manager metrics.

	if h.tokenManager != nil {

		metrics["supported_models"] = h.tokenManager.GetSupportedModels()

	}

	// Add circuit breaker metrics.

	if h.circuitBreakerMgr != nil {

		metrics["circuit_breakers"] = h.circuitBreakerMgr.GetAllStats()

	}

	// Add streaming metrics.

	if h.streamingProcessor != nil {

		metrics["streaming"] = h.streamingProcessor.GetMetrics()

	}

	// Add context builder metrics.

	if h.contextBuilder != nil {

		metrics["context_builder"] = h.contextBuilder.GetMetrics()

	}

	// Add relevance scorer metrics.

	if h.relevanceScorer != nil {

		metrics["relevance_scorer"] = h.relevanceScorer.GetMetrics()

	}

	// Add prompt builder metrics.

	if h.promptBuilder != nil {

		metrics["prompt_builder"] = map[string]interface{}{"status": "stubbed"}

	}

	h.writeJSONResponse(w, metrics, http.StatusOK)

}

// CircuitBreakerStatusHandler provides circuit breaker status and control.

func (h *LLMProcessorHandler) CircuitBreakerStatusHandler(w http.ResponseWriter, r *http.Request) {

	// Start timing for metrics.

	handlerStartTime := time.Now()

	var statusCode int = http.StatusOK

	// Ensure metrics are recorded when handler exits.

	defer func() {

		duration := time.Since(handlerStartTime)

		h.metricsCollector.RecordHTTPRequest(

			r.Method,

			"/api/v1/circuit-breaker",

			strconv.Itoa(statusCode),

			duration,
		)

	}()

	if h.circuitBreakerMgr == nil {

		statusCode = http.StatusServiceUnavailable

		h.writeErrorResponse(w, "Circuit breaker manager not available", statusCode, "")

		return

	}

	// Handle POST requests for circuit breaker operations.

	if r.Method == http.MethodPost {

		var req struct {
			Action string `json:"action"`

			Name string `json:"name"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {

			statusCode = http.StatusBadRequest

			h.writeErrorResponse(w, "Invalid request body", statusCode, "")

			return

		}

		cb, exists := h.circuitBreakerMgr.Get(req.Name)

		if !exists {

			statusCode = http.StatusNotFound

			h.writeErrorResponse(w, "Circuit breaker not found", statusCode, "")

			return

		}

		switch req.Action {

		case "reset":

			cb.Reset()

			h.logger.Info("Circuit breaker reset", "name", req.Name)

		case "force_open":

			cb.ForceOpen()

			h.logger.Info("Circuit breaker forced open", "name", req.Name)

		default:

			statusCode = http.StatusBadRequest

			h.writeErrorResponse(w, "Invalid action", statusCode, "")

			return

		}

		h.writeJSONResponse(w, map[string]string{"status": "success"}, http.StatusOK)

		return

	}

	// Handle GET requests for status.

	stats := h.circuitBreakerMgr.GetAllStats()

	h.writeJSONResponse(w, stats, http.StatusOK)

}

// ProcessIntent processes an intent using the configured processor.

func (p *IntentProcessor) ProcessIntent(ctx context.Context, intent string, metadata map[string]string) (*ProcessIntentResult, error) {

	p.Logger.Debug("Processing intent with enhanced client", slog.String("intent", intent))

	// Use circuit breaker for fault tolerance.

	operation := func(ctx context.Context) (interface{}, error) {

		// RAG-enhanced processing stubbed out.

		// if p.RAGEnhancedClient != nil { ... }.

		// Fallback to base LLM client.

		return p.LLMClient.ProcessIntent(ctx, intent)

	}

	result, err := p.CircuitBreaker.Execute(ctx, operation)

	if err != nil {

		return nil, fmt.Errorf("LLM processing failed: %w", err)

	}

	// Create ProcessIntentResult from the raw LLM response.

	processedResult := &ProcessIntentResult{

		Result: result.(string),

		Status: "success",

		Metadata: map[string]interface{}{

			"original_metadata": metadata,

			"processing_method": "llm_enhanced",
		},
	}

	return processedResult, nil

}

// Helper methods.

func (h *LLMProcessorHandler) writeJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {

		h.logger.Error("Failed to encode JSON response", slog.String("error", err.Error()))

	}

}

// NLToIntentHandler handles natural language to intent conversion.

// POST /nl/intent - Accepts text/plain body and returns Intent JSON.

func (h *LLMProcessorHandler) NLToIntentHandler(w http.ResponseWriter, r *http.Request) {

	var statusCode int

	startTime := time.Now()

	defer func() {

		duration := time.Since(startTime).Seconds()

		if h.metricsCollector != nil {

			h.metricsCollector.RecordHTTPRequest(

				r.Method,

				"/nl/intent",

				strconv.Itoa(statusCode),

				time.Duration(duration*float64(time.Second)),
			)

		}

	}()

	if r.Method != http.MethodPost {

		statusCode = http.StatusMethodNotAllowed

		h.writeErrorResponse(w, "Method not allowed", statusCode, "")

		return

	}

	// Read the raw text body.

	body, err := io.ReadAll(r.Body)

	if err != nil {

		statusCode = http.StatusBadRequest

		h.writeErrorResponse(w, "Failed to read request body", statusCode, "")

		return

	}

	// Ignore body close error - defer handles cleanup.

	defer func() { _ = r.Body.Close() }()

	text := string(body)

	if text == "" {

		statusCode = http.StatusBadRequest

		h.writeErrorResponse(w, "Empty request body", statusCode, "")

		return

	}

	// Use the rule-based parser from internal/ingest.

	parser := ingest.NewRuleBasedIntentParser()

	intent, err := parser.ParseIntent(text)

	if err != nil {

		h.logger.Error("Failed to parse intent",

			slog.String("error", err.Error()),

			slog.String("text", text),
		)

		statusCode = http.StatusBadRequest

		h.writeErrorResponse(w, fmt.Sprintf("Failed to parse intent: %v", err), statusCode, "")

		return

	}

	// Validate the intent structure with JSON schema.

	if err := ingest.ValidateIntentWithSchema(intent, ""); err != nil {

		h.logger.Error("Invalid intent structure",

			slog.String("error", err.Error()),

			slog.Any("intent", intent),
		)

		statusCode = http.StatusBadRequest

		h.writeErrorResponse(w, fmt.Sprintf("Invalid intent: %v", err), statusCode, "")

		return

	}

	statusCode = http.StatusOK

	h.writeJSONResponse(w, intent, statusCode)

}

func (h *LLMProcessorHandler) writeErrorResponse(w http.ResponseWriter, message string, statusCode int, requestID string) {

	response := map[string]interface{}{

		"error": message,

		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	if requestID != "" {

		response["request_id"] = requestID

		w.Header().Set("X-Request-ID", requestID)

	}

	h.writeJSONResponse(w, response, statusCode)

}
