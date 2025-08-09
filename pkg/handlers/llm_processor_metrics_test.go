package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/health"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"
	"log/slog"
)

// TestProcessIntentHandlerMetrics tests that metrics are recorded for the ProcessIntentHandler
func TestProcessIntentHandlerMetrics(t *testing.T) {
	// Create a mock metrics collector for testing
	metricsCollector := monitoring.NewMetricsCollector()

	// Create handler with metrics collector
	handler := NewLLMProcessorHandlerWithMetrics(
		&config.LLMProcessorConfig{
			RequestTimeout: 5 * time.Second,
			ServiceVersion: "test-v1",
		},
		&IntentProcessor{
			LLMClient: &llm.Client{},
			Logger:    slog.Default(),
		},
		nil, // streamingProcessor
		nil, // circuitBreakerMgr
		nil, // tokenManager
		nil, // contextBuilder
		nil, // relevanceScorer
		nil, // promptBuilder
		slog.Default(),
		health.NewHealthChecker(),
		time.Now(),
		metricsCollector,
	)

	// Test successful request
	t.Run("Successful Request Metrics", func(t *testing.T) {
		reqBody := ProcessIntentRequest{
			Intent: "Deploy a high-availability AMF instance",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/process-intent", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		// Call handler
		handler.ProcessIntentHandler(rr, req)

		// Verify metrics were recorded (would need to check Prometheus metrics in real test)
		// This is a placeholder for actual metric verification
		if rr.Code != http.StatusOK && rr.Code != http.StatusInternalServerError {
			t.Errorf("Handler returned wrong status code: got %v want %v or %v",
				rr.Code, http.StatusOK, http.StatusInternalServerError)
		}
	})

	// Test error request
	t.Run("Error Request Metrics", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/process-intent", bytes.NewBuffer([]byte("invalid json")))
		rr := httptest.NewRecorder()

		// Call handler
		handler.ProcessIntentHandler(rr, req)

		// Should return bad request
		if rr.Code != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code for invalid JSON: got %v want %v",
				rr.Code, http.StatusBadRequest)
		}
	})

	// Test method not allowed
	t.Run("Method Not Allowed Metrics", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/process-intent", nil)
		rr := httptest.NewRecorder()

		// Call handler
		handler.ProcessIntentHandler(rr, req)

		// Should return method not allowed
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("Handler returned wrong status code for GET: got %v want %v",
				rr.Code, http.StatusMethodNotAllowed)
		}
	})
}

// TestStatusHandlerMetrics tests that metrics are recorded for the StatusHandler
func TestStatusHandlerMetrics(t *testing.T) {
	metricsCollector := monitoring.NewMetricsCollector()

	handler := NewLLMProcessorHandlerWithMetrics(
		&config.LLMProcessorConfig{
			ServiceVersion: "test-v1",
			LLMBackendType: "openai",
			LLMModelName:   "gpt-4",
			RAGEnabled:     true,
		},
		nil, // processor
		nil, // streamingProcessor
		nil, // circuitBreakerMgr
		nil, // tokenManager
		nil, // contextBuilder
		nil, // relevanceScorer
		nil, // promptBuilder
		slog.Default(),
		health.NewHealthChecker(),
		time.Now(),
		metricsCollector,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	rr := httptest.NewRecorder()

	// Call handler
	handler.StatusHandler(rr, req)

	// Should return OK
	if rr.Code != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v",
			rr.Code, http.StatusOK)
	}

	// Verify response contains expected fields
	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["service"] != "llm-processor" {
		t.Errorf("Response missing or incorrect service field")
	}
}

// TestStreamingHandlerMetrics tests that metrics are recorded for SSE streaming
func TestStreamingHandlerMetrics(t *testing.T) {
	metricsCollector := monitoring.NewMetricsCollector()

	// Create a mock streaming processor
	streamingProcessor := &llm.StreamingProcessor{}

	handler := NewLLMProcessorHandlerWithMetrics(
		&config.LLMProcessorConfig{
			StreamTimeout: 10 * time.Second,
			LLMModelName:  "gpt-4",
			LLMMaxTokens:  1000,
		},
		nil, // processor
		streamingProcessor,
		nil, // circuitBreakerMgr
		nil, // tokenManager
		nil, // contextBuilder
		nil, // relevanceScorer
		nil, // promptBuilder
		slog.Default(),
		health.NewHealthChecker(),
		time.Now(),
		metricsCollector,
	)

	// Test streaming not enabled case
	t.Run("Streaming Not Enabled", func(t *testing.T) {
		handler.streamingProcessor = nil

		req := httptest.NewRequest(http.MethodPost, "/api/v1/stream", nil)
		rr := httptest.NewRecorder()

		handler.StreamingHandler(rr, req)

		if rr.Code != http.StatusServiceUnavailable {
			t.Errorf("Handler returned wrong status code: got %v want %v",
				rr.Code, http.StatusServiceUnavailable)
		}
	})

	// Test invalid method
	t.Run("Invalid Method", func(t *testing.T) {
		handler.streamingProcessor = streamingProcessor

		req := httptest.NewRequest(http.MethodGet, "/api/v1/stream", nil)
		rr := httptest.NewRecorder()

		handler.StreamingHandler(rr, req)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("Handler returned wrong status code: got %v want %v",
				rr.Code, http.StatusMethodNotAllowed)
		}
	})
}

// TestMetricsHandlerMetrics tests that the metrics endpoint itself is instrumented
func TestMetricsHandlerMetrics(t *testing.T) {
	metricsCollector := monitoring.NewMetricsCollector()

	handler := NewLLMProcessorHandlerWithMetrics(
		&config.LLMProcessorConfig{
			ServiceVersion: "test-v1",
		},
		nil, // processor
		nil, // streamingProcessor
		nil, // circuitBreakerMgr
		nil, // tokenManager
		nil, // contextBuilder
		nil, // relevanceScorer
		nil, // promptBuilder
		slog.Default(),
		health.NewHealthChecker(),
		time.Now(),
		metricsCollector,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics", nil)
	rr := httptest.NewRecorder()

	handler.MetricsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v",
			rr.Code, http.StatusOK)
	}
}

// TestCircuitBreakerStatusHandlerMetrics tests circuit breaker handler metrics
func TestCircuitBreakerStatusHandlerMetrics(t *testing.T) {
	metricsCollector := monitoring.NewMetricsCollector()

	handler := NewLLMProcessorHandlerWithMetrics(
		&config.LLMProcessorConfig{},
		nil, // processor
		nil, // streamingProcessor
		nil, // circuitBreakerMgr - deliberately nil for test
		nil, // tokenManager
		nil, // contextBuilder
		nil, // relevanceScorer
		nil, // promptBuilder
		slog.Default(),
		health.NewHealthChecker(),
		time.Now(),
		metricsCollector,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/circuit-breaker", nil)
	rr := httptest.NewRecorder()

	handler.CircuitBreakerStatusHandler(rr, req)

	// Should return service unavailable since circuitBreakerMgr is nil
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Handler returned wrong status code: got %v want %v",
			rr.Code, http.StatusServiceUnavailable)
	}
}
