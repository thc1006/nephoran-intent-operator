package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/health"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"
)

// LLMProcessorHandler handles HTTP requests for the LLM processor service
type LLMProcessorHandler struct {
	config             *config.LLMProcessorConfig
	processor          *IntentProcessor
	streamingProcessor *llm.StreamingProcessor
	circuitBreakerMgr  *llm.CircuitBreakerManager
	tokenManager       *llm.TokenManager
	contextBuilder     *llm.ContextBuilder
	relevanceScorer    *llm.RelevanceScorer
	promptBuilder      *llm.RAGAwarePromptBuilder
	logger             *slog.Logger
	healthChecker      *health.HealthChecker
	startTime          time.Time
	metricsCollector   *monitoring.MetricsCollector
}

// Request/Response structures
type ProcessIntentRequest struct {
	Intent   string            `json:"intent"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type ProcessIntentResponse struct {
	Result         string                 `json:"result"`
	ProcessingTime string                 `json:"processing_time"`
	RequestID      string                 `json:"request_id"`
	ServiceVersion string                 `json:"service_version"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Status         string                 `json:"status"`
	Error          string                 `json:"error,omitempty"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Uptime    string `json:"uptime"`
	Timestamp string `json:"timestamp"`
}

// IntentProcessor handles the LLM processing logic with RAG enhancement
type IntentProcessor struct {
	LLMClient         *llm.Client
	RAGEnhancedClient interface{}
	CircuitBreaker    *llm.CircuitBreaker
	Logger            *slog.Logger
}

// NewLLMProcessorHandler creates a new handler instance
func NewLLMProcessorHandler(
	config *config.LLMProcessorConfig,
	processor *IntentProcessor,
	streamingProcessor *llm.StreamingProcessor,
	circuitBreakerMgr *llm.CircuitBreakerManager,
	tokenManager *llm.TokenManager,
	contextBuilder *llm.ContextBuilder,
	relevanceScorer *llm.RelevanceScorer,
	promptBuilder *llm.RAGAwarePromptBuilder,
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

// NewLLMProcessorHandlerWithMetrics creates a new handler instance with a provided metrics collector
func NewLLMProcessorHandlerWithMetrics(
	config *config.LLMProcessorConfig,
	processor *IntentProcessor,
	streamingProcessor *llm.StreamingProcessor,
	circuitBreakerMgr *llm.CircuitBreakerManager,
	tokenManager *llm.TokenManager,
	contextBuilder *llm.ContextBuilder,
	relevanceScorer *llm.RelevanceScorer,
	promptBuilder *llm.RAGAwarePromptBuilder,
	logger *slog.Logger,
	healthChecker *health.HealthChecker,
	startTime time.Time,
	metricsCollector *monitoring.MetricsCollector,
) *LLMProcessorHandler {
	return &LLMProcessorHandler{
		config:             config,
		processor:          processor,
		streamingProcessor: streamingProcessor,
		circuitBreakerMgr:  circuitBreakerMgr,
		tokenManager:       tokenManager,
		contextBuilder:     contextBuilder,
		relevanceScorer:    relevanceScorer,
		promptBuilder:      promptBuilder,
		logger:             logger,
		healthChecker:      healthChecker,
		startTime:          startTime,
		metricsCollector:   metricsCollector,
	}
}

// ProcessIntentHandler handles intent processing requests with optimized concurrent processing
func (h *LLMProcessorHandler) ProcessIntentHandler(w http.ResponseWriter, r *http.Request) {
	// Start timing for metrics
	handlerStartTime := time.Now()
	var statusCode int = http.StatusOK // Default to OK, will be overridden on error

	// Ensure metrics are recorded when handler exits
	defer func() {
		duration := time.Since(handlerStartTime)
		h.metricsCollector.RecordHTTPRequest(
			r.Method,
			"/api/v1/process-intent",
			strconv.Itoa(statusCode),
			duration,
		)
	}()

	// Early context cancellation check
	select {
	case <-r.Context().Done():
		statusCode = http.StatusRequestTimeout
		h.writeErrorResponse(w, "Request cancelled", statusCode, "")
		return
	default:
	}

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

	// Process intent with context cancellation support and buffered channels
	ctx, cancel := context.WithTimeout(r.Context(), h.config.RequestTimeout)
	defer cancel()

	// Use buffered channel for concurrent processing
	resultCh := make(chan struct {
		result string
		err    error
	}, 1)

	go func() {
		defer close(resultCh)

		result, err := h.processor.ProcessIntent(ctx, req.Intent)

		select {
		case resultCh <- struct {
			result string
			err    error
		}{result: result, err: err}:
		case <-ctx.Done():
			// Context cancelled, don't send result
		}
	}()

	// Wait for result or context cancellation
	select {
	case res := <-resultCh:
		if res.err != nil {
			h.logger.Error("Failed to process intent",
				slog.String("error", res.err.Error()),
				slog.String("intent", req.Intent),
			)
			statusCode = http.StatusInternalServerError
			response := ProcessIntentResponse{
				Status:         "error",
				Error:          res.err.Error(),
				RequestID:      reqID,
				ServiceVersion: h.config.ServiceVersion,
				ProcessingTime: time.Since(startTime).String(),
			}
			h.writeJSONResponse(w, response, statusCode)
			return
		}

		response := ProcessIntentResponse{
			Result:         res.result,
			Status:         "success",
			ProcessingTime: time.Since(startTime).String(),
			RequestID:      reqID,
			ServiceVersion: h.config.ServiceVersion,
		}

		h.writeJSONResponse(w, response, http.StatusOK)

		h.logger.Info("Intent processed successfully",
			slog.String("request_id", reqID),
			slog.Duration("processing_time", time.Since(startTime)),
		)
	case <-ctx.Done():
		statusCode = http.StatusRequestTimeout
		response := ProcessIntentResponse{
			Status:         "error",
			Error:          "Request timeout",
			RequestID:      reqID,
			ServiceVersion: h.config.ServiceVersion,
			ProcessingTime: time.Since(startTime).String(),
		}
		h.writeJSONResponse(w, response, statusCode)
		h.logger.Warn("Intent processing timed out", slog.String("request_id", reqID))
	}
}

// StatusHandler returns service status information
func (h *LLMProcessorHandler) StatusHandler(w http.ResponseWriter, r *http.Request) {
	// Start timing for metrics
	handlerStartTime := time.Now()
	statusCode := http.StatusOK

	// Ensure metrics are recorded when handler exits
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
		"service":      "llm-processor",
		"version":      h.config.ServiceVersion,
		"uptime":       time.Since(h.startTime).String(),
		"healthy":      h.healthChecker.IsHealthy(),
		"ready":        h.healthChecker.IsReady(),
		"backend_type": h.config.LLMBackendType,
		"model":        h.config.LLMModelName,
		"rag_enabled":  h.config.RAGEnabled,
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	}

	h.writeJSONResponse(w, status, http.StatusOK)
}

// StreamingHandler handles Server-Sent Events streaming requests
func (h *LLMProcessorHandler) StreamingHandler(w http.ResponseWriter, r *http.Request) {
	// Start timing for HTTP request metrics
	handlerStartTime := time.Now()
	var statusCode int = http.StatusOK

	// Start timing for SSE stream duration
	streamStartTime := time.Now()
	streamRoute := "/api/v1/stream"

	// Ensure HTTP metrics are recorded when handler exits
	defer func() {
		// Record HTTP request metrics
		duration := time.Since(handlerStartTime)
		h.metricsCollector.RecordHTTPRequest(
			r.Method,
			streamRoute,
			strconv.Itoa(statusCode),
			duration,
		)

		// Record SSE stream duration only if streaming actually occurred
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

	// Set defaults
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

	// Add timeout context for streaming
	ctx, cancel := context.WithTimeout(r.Context(), h.config.StreamTimeout)
	defer cancel()

	// Update request with context
	r = r.WithContext(ctx)

	err := h.streamingProcessor.HandleStreamingRequest(w, r, &req)
	if err != nil {
		h.logger.Error("Streaming request failed", slog.String("error", err.Error()))
		// Error handling is done within HandleStreamingRequest
		// Set status code to indicate error for metrics
		statusCode = http.StatusInternalServerError
	}
}

// MetricsHandler provides comprehensive metrics
func (h *LLMProcessorHandler) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	// Start timing for metrics
	handlerStartTime := time.Now()
	statusCode := http.StatusOK

	// Ensure metrics are recorded when handler exits
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
		"uptime":  time.Since(h.startTime).String(),
	}

	// Add token manager metrics
	if h.tokenManager != nil {
		metrics["supported_models"] = []string{"gpt-3.5-turbo", "gpt-4"}
	}

	// Add circuit breaker metrics
	if h.circuitBreakerMgr != nil {
		metrics["circuit_breakers"] = h.circuitBreakerMgr.GetAllStats()
	}

	// Add streaming metrics
	if h.streamingProcessor != nil {
		metrics["streaming"] = h.streamingProcessor.GetMetrics()
	}

	// Add context builder metrics
	if h.contextBuilder != nil {
		metrics["context_builder"] = h.contextBuilder.GetMetrics()
	}

	// Add relevance scorer metrics
	if h.relevanceScorer != nil {
		metrics["relevance_scorer"] = h.relevanceScorer.GetMetrics()
	}

	// Add prompt builder metrics
	if h.promptBuilder != nil {
		metrics["prompt_builder"] = h.promptBuilder.GetMetrics()
	}

	h.writeJSONResponse(w, metrics, http.StatusOK)
}

// CircuitBreakerStatusHandler provides circuit breaker status and control
func (h *LLMProcessorHandler) CircuitBreakerStatusHandler(w http.ResponseWriter, r *http.Request) {
	// Start timing for metrics
	handlerStartTime := time.Now()
	var statusCode int = http.StatusOK

	// Ensure metrics are recorded when handler exits
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

	// Handle POST requests for circuit breaker operations
	if r.Method == http.MethodPost {
		var req struct {
			Action string `json:"action"`
			Name   string `json:"name"`
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

	// Handle GET requests for status
	stats := h.circuitBreakerMgr.GetAllStats()
	h.writeJSONResponse(w, stats, http.StatusOK)
}

// ProcessIntent processes an intent using the configured processor with performance optimizations
func (p *IntentProcessor) ProcessIntent(ctx context.Context, intent string) (string, error) {
	p.Logger.Debug("Processing intent with enhanced client", slog.String("intent", intent))

	// Check for context cancellation early
	select {
	case <-ctx.Done():
		return "", fmt.Errorf("intent processing cancelled: %w", ctx.Err())
	default:
	}

	// Use circuit breaker for fault tolerance with optimized operation
	operation := func(ctx context.Context) (interface{}, error) {
		// Create a channel for concurrent processing results
		resultCh := make(chan struct {
			result string
			err    error
		}, 1)

		// Start processing in a goroutine to enable cancellation
		go func() {
			defer close(resultCh)

			// Try RAG-enhanced processing first if available
			if p.RAGEnhancedClient != nil {
				// RAG-enhanced processing would go here if implemented
				p.Logger.Info("RAG-enhanced processing not yet implemented, using base client")
			}

			// Process with base LLM client
			result, err := p.LLMClient.ProcessIntent(ctx, intent)

			select {
			case resultCh <- struct {
				result string
				err    error
			}{result: result, err: err}:
			case <-ctx.Done():
				// Context cancelled, don't send result
			}
		}()

		// Wait for result or context cancellation
		select {
		case res := <-resultCh:
			if res.err != nil {
				return "", fmt.Errorf("LLM processing failed: %w", res.err)
			}
			return res.result, nil
		case <-ctx.Done():
			return "", fmt.Errorf("LLM processing cancelled: %w", ctx.Err())
		}
	}

	result, err := p.CircuitBreaker.Execute(ctx, operation)
	if err != nil {
		return "", err
	}

	return result.(string), nil
}

// Helper methods

func (h *LLMProcessorHandler) writeJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", slog.String("error", err.Error()))
	}
}

func (h *LLMProcessorHandler) writeErrorResponse(w http.ResponseWriter, message string, statusCode int, requestID string) {
	response := map[string]interface{}{
		"error":     message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	if requestID != "" {
		response["request_id"] = requestID
		w.Header().Set("X-Request-ID", requestID)
	}

	h.writeJSONResponse(w, response, statusCode)
}
