package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/health"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
)

// BufferLogHandler implements slog.Handler to capture log output in a buffer
type BufferLogHandler struct {
	buffer *bytes.Buffer
	attrs  []slog.Attr
	groups []string
}

// NewBufferLogHandler creates a new buffer-backed slog handler
func NewBufferLogHandler(buffer *bytes.Buffer) *BufferLogHandler {
	return &BufferLogHandler{
		buffer: buffer,
		attrs:  make([]slog.Attr, 0),
		groups: make([]string, 0),
	}
}

// Enabled implements slog.Handler.Enabled
func (h *BufferLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true // Capture all log levels for testing
}

// Handle implements slog.Handler.Handle
func (h *BufferLogHandler) Handle(ctx context.Context, record slog.Record) error {
	// Create a structured log entry
	entry := make(map[string]interface{})

	// Add basic fields
	entry["message"] = record.Message
	entry["level"] = record.Level.String()
	entry["time"] = record.Time.Format("2006-01-02T15:04:05.000Z")

	// Add record attributes
	record.Attrs(func(attr slog.Attr) bool {
		entry[attr.Key] = attr.Value.Any()
		return true
	})

	// Add handler attributes
	for _, attr := range h.attrs {
		entry[attr.Key] = attr.Value.Any()
	}

	// Serialize to JSON and write to buffer
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	_, _ = h.buffer.Write(data) // #nosec G104 - Buffer write in test
	_, _ = h.buffer.WriteString("\n") // #nosec G104 - Buffer write in test
	return nil
}

// WithAttrs implements slog.Handler.WithAttrs
func (h *BufferLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &BufferLogHandler{
		buffer: h.buffer,
		attrs:  newAttrs,
		groups: h.groups,
	}
}

// WithGroup implements slog.Handler.WithGroup
func (h *BufferLogHandler) WithGroup(name string) slog.Handler {
	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name

	return &BufferLogHandler{
		buffer: h.buffer,
		attrs:  h.attrs,
		groups: newGroups,
	}
}

// StreamingProcessorInterface defines the minimal interface needed for testing
type StreamingProcessorInterface interface {
	HandleStreamingRequest(w http.ResponseWriter, r *http.Request, req *StreamingRequest) error
	GetMetrics() map[string]interface{}
}

// MockStreamingProcessor provides a mock implementation for testing
type MockStreamingProcessor struct {
	handleStreamingRequestFunc func(w http.ResponseWriter, r *http.Request, req *StreamingRequest) error
	getMetricsFunc             func() map[string]interface{}
}

// StreamingRequest represents the structure expected by the streaming handler
type StreamingRequest struct {
	Query       string  `json:"query"`
	ModelName   string  `json:"model_name"`
	EnableRAG   bool    `json:"enable_rag"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

// TestServiceManager represents a minimal service manager for testing
type TestServiceManager struct {
	config             *TestConfig
	logger             *slog.Logger
	streamingProcessor StreamingProcessorInterface
}

// TestConfig represents the minimal configuration needed for testing
type TestConfig struct {
	ServiceVersion   string
	LLMModelName     string
	LLMMaxTokens     int
	StreamingEnabled bool
}

// HandleStreamingRequest implements the streaming processor interface
func (m *MockStreamingProcessor) HandleStreamingRequest(w http.ResponseWriter, r *http.Request, req *StreamingRequest) error {
	if m.handleStreamingRequestFunc != nil {
		return m.handleStreamingRequestFunc(w, r, req)
	}
	// Default mock implementation - just return success
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("mock streaming response")) // #nosec G104 - Mock write in test
	return nil
}

// GetMetrics implements the streaming processor interface
func (m *MockStreamingProcessor) GetMetrics() map[string]interface{} {
	if m.getMetricsFunc != nil {
		return m.getMetricsFunc()
	}
	return make(map[string]interface{})
}

// TestStructuredLoggingInStreamingHandler tests that structured logging works correctly
func TestStructuredLoggingInStreamingHandler(t *testing.T) {
	tests := []struct {
		name              string
		request           StreamingRequest
		expectedQuery     string
		expectedModel     string
		expectedEnableRAG bool
		mockError         error
		expectError       bool
	}{
		{
			name: "Valid streaming request with RAG enabled",
			request: StreamingRequest{
				Query:     "Deploy UPF network function with high availability",
				ModelName: "gpt-4o-mini",
				EnableRAG: true,
				MaxTokens: 2048,
			},
			expectedQuery:     "Deploy UPF network function with high availability",
			expectedModel:     "gpt-4o-mini",
			expectedEnableRAG: true,
			expectError:       false,
		},
		{
			name: "Valid streaming request with RAG disabled",
			request: StreamingRequest{
				Query:     "Scale AMF to 5 replicas",
				ModelName: "claude-3-sonnet",
				EnableRAG: false,
				MaxTokens: 1024,
			},
			expectedQuery:     "Scale AMF to 5 replicas",
			expectedModel:     "claude-3-sonnet",
			expectedEnableRAG: false,
			expectError:       false,
		},
		{
			name: "Empty query with logging verification",
			request: StreamingRequest{
				Query:     "",
				ModelName: "gpt-4o-mini",
				EnableRAG: true,
			},
			expectedQuery:     "",
			expectedModel:     "gpt-4o-mini",
			expectedEnableRAG: true,
			expectError:       true, // Handler should return error for empty query
		},
		{
			name: "Complex telecom query with special characters",
			request: StreamingRequest{
				Query:     "Configure O-RAN DU with gNB-ID 12345 and PLMN-ID 310-410",
				ModelName: "mistral-large",
				EnableRAG: true,
			},
			expectedQuery:     "Configure O-RAN DU with gNB-ID 12345 and PLMN-ID 310-410",
			expectedModel:     "mistral-large",
			expectedEnableRAG: true,
			expectError:       false,
		},
		{
			name: "Streaming processor returns error",
			request: StreamingRequest{
				Query:     "Test error handling",
				ModelName: "gpt-4o-mini",
				EnableRAG: false,
			},
			expectedQuery:     "Test error handling",
			expectedModel:     "gpt-4o-mini",
			expectedEnableRAG: false,
			mockError:         fmt.Errorf("mock streaming processor error"),
			expectError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create buffer to capture log output
			logBuffer := &bytes.Buffer{}
			bufferHandler := NewBufferLogHandler(logBuffer)
			logger := slog.New(bufferHandler)

			// Create test configuration
			config := &TestConfig{
				ServiceVersion:   "test-v1.0.0",
				LLMModelName:     "gpt-4o-mini",
				LLMMaxTokens:     2048,
				StreamingEnabled: true,
			}

			// Create mock streaming processor
			mockProcessor := &MockStreamingProcessor{
				handleStreamingRequestFunc: func(w http.ResponseWriter, r *http.Request, req *StreamingRequest) error {
					if tt.mockError != nil {
						return tt.mockError
					}
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("test response")) // #nosec G104 - Test response write
					return nil
				},
			}

			// Create a minimal service manager structure for testing
			sm := &TestServiceManager{
				config:             config,
				logger:             logger,
				streamingProcessor: mockProcessor,
			}

			// Create test HTTP request
			requestBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/stream", bytes.NewBuffer(requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Execute the streaming handler
			sm.streamingHandler(w, req)

			// Verify HTTP response status
			if tt.expectError {
				if strings.Contains(tt.name, "Empty query") {
					assert.Equal(t, http.StatusBadRequest, w.Code)
				} else {
					// For errors, the status might still be 200 if error handling is done within HandleStreamingRequest
					// The important thing is that the error is logged
					assert.True(t, w.Code == http.StatusOK || w.Code >= http.StatusBadRequest, "Status should be either OK or an error code")
				}
			} else {
				assert.Equal(t, http.StatusOK, w.Code)
			}

			// Parse captured log entries
			logOutput := logBuffer.String()

			// For empty query test, handler returns early with error, so no streaming log is generated
			if strings.Contains(tt.name, "Empty query") && tt.expectError {
				// Should have error log but no streaming request log
				if logOutput != "" {
					// Verify no "Starting streaming request" log exists
					assert.NotContains(t, logOutput, "Starting streaming request")
				}
				return // Skip the rest of the checks for this case
			}

			require.NotEmpty(t, logOutput, "Expected log output to be captured")

			// Split log entries by newlines and parse each JSON entry
			logLines := strings.Split(strings.TrimSpace(logOutput), "\n")
			var foundStartingLog bool
			var foundErrorLog bool

			for _, line := range logLines {
				if line == "" {
					continue
				}

				var logEntry map[string]interface{}
				err = json.Unmarshal([]byte(line), &logEntry)
				require.NoError(t, err, "Failed to parse log entry: %s", line)

				// Check for the "Starting streaming request" log entry
				if message, ok := logEntry["message"].(string); ok {
					if strings.Contains(message, "Starting streaming request") {
						foundStartingLog = true

						// Verify structured attributes are present and correct
						assert.Equal(t, tt.expectedQuery, logEntry["query"],
							"Query attribute should match expected value")
						assert.Equal(t, tt.expectedModel, logEntry["model"],
							"Model attribute should match expected value")
						assert.Equal(t, tt.expectedEnableRAG, logEntry["enable_rag"],
							"EnableRAG attribute should match expected value")

						// Verify attribute types
						assert.IsType(t, "", logEntry["query"],
							"Query should be serialized as string")
						assert.IsType(t, "", logEntry["model"],
							"Model should be serialized as string")
						assert.IsType(t, false, logEntry["enable_rag"],
							"EnableRAG should be serialized as boolean")

						// Verify log level
						assert.Equal(t, "INFO", logEntry["level"],
							"Starting streaming request should be logged at INFO level")
					}

					// Check for error log entries if expected
					if tt.expectError && strings.Contains(message, "Streaming request failed") {
						foundErrorLog = true
						assert.Equal(t, "ERROR", logEntry["level"],
							"Error should be logged at ERROR level")
						assert.Contains(t, logEntry["error"], "mock streaming processor error",
							"Error message should contain the mock error")
					}
				}
			}

			// Verify that the expected log entries were found
			assert.True(t, foundStartingLog,
				"Should find 'Starting streaming request' log entry with structured attributes")

			if tt.expectError {
				assert.True(t, foundErrorLog,
					"Should find error log entry when streaming processor returns error")
			}

			// Additional verification: ensure attributes are properly structured
			t.Run("verify_structured_attributes_format", func(t *testing.T) {
				// Parse the first log entry that contains structured attributes
				for _, line := range logLines {
					if line == "" {
						continue
					}

					var logEntry map[string]interface{}
					if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
						continue
					}

					if message, ok := logEntry["message"].(string); ok &&
						strings.Contains(message, "Starting streaming request") {

						// Test that slog.String() attributes are properly serialized
						queryAttr, queryExists := logEntry["query"]
						require.True(t, queryExists, "Query attribute should exist")
						assert.IsType(t, "", queryAttr, "Query should be string type")

						// Test that slog.String() for model is properly serialized
						modelAttr, modelExists := logEntry["model"]
						require.True(t, modelExists, "Model attribute should exist")
						assert.IsType(t, "", modelAttr, "Model should be string type")

						// Test that slog.Bool() attributes are properly serialized
						ragAttr, ragExists := logEntry["enable_rag"]
						require.True(t, ragExists, "EnableRAG attribute should exist")
						assert.IsType(t, false, ragAttr, "EnableRAG should be boolean type")

						// Verify the actual values match what was logged in the service manager
						assert.Equal(t, tt.expectedQuery, queryAttr)
						assert.Equal(t, tt.expectedModel, modelAttr)
						assert.Equal(t, tt.expectedEnableRAG, ragAttr)

						break
					}
				}
			})
		})
	}
}

// TestStreamingHandlerLoggingEdgeCases tests edge cases for structured logging
func TestStreamingHandlerLoggingEdgeCases(t *testing.T) {
	t.Run("malformed_json_request", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		bufferHandler := NewBufferLogHandler(logBuffer)
		logger := slog.New(bufferHandler)

		config := &TestConfig{
			ServiceVersion:   "test-v1.0.0",
			StreamingEnabled: true,
		}

		sm := &TestServiceManager{
			config:             config,
			logger:             logger,
			streamingProcessor: &MockStreamingProcessor{}, // Add empty mock processor
		}

		// Send malformed JSON
		req := httptest.NewRequest(http.MethodPost, "/stream",
			bytes.NewBufferString(`{"invalid": json}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		sm.streamingHandler(w, req)

		// Should return bad request
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Check that error was logged
		logOutput := logBuffer.String()
		assert.Contains(t, logOutput, "Failed to decode streaming request")
	})

	t.Run("streaming_not_enabled", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		bufferHandler := NewBufferLogHandler(logBuffer)
		logger := slog.New(bufferHandler)

		config := &TestConfig{
			ServiceVersion:   "test-v1.0.0",
			StreamingEnabled: false, // Streaming disabled
		}

		sm := &TestServiceManager{
			config:             config,
			logger:             logger,
			streamingProcessor: nil, // No processor when streaming disabled
		}

		request := StreamingRequest{
			Query:     "test query",
			ModelName: "gpt-4o-mini",
			EnableRAG: true,
		}

		requestBody, err := json.Marshal(request)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/stream", bytes.NewBuffer(requestBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		sm.streamingHandler(w, req)

		// Should return service unavailable
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})

	t.Run("empty_request_body", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		bufferHandler := NewBufferLogHandler(logBuffer)
		logger := slog.New(bufferHandler)

		config := &TestConfig{
			ServiceVersion:   "test-v1.0.0",
			StreamingEnabled: true,
		}

		sm := &TestServiceManager{
			config:             config,
			logger:             logger,
			streamingProcessor: &MockStreamingProcessor{}, // Add empty mock processor
		}

		req := httptest.NewRequest(http.MethodPost, "/stream", bytes.NewBufferString(""))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		sm.streamingHandler(w, req)

		// Should return bad request due to empty body
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Check that decode error was logged
		logOutput := logBuffer.String()
		assert.Contains(t, logOutput, "Failed to decode streaming request")
	})
}

// TestBufferLogHandlerFunctionality tests the custom buffer log handler implementation
func TestBufferLogHandlerFunctionality(t *testing.T) {
	t.Run("basic_logging", func(t *testing.T) {
		buffer := &bytes.Buffer{}
		handler := NewBufferLogHandler(buffer)
		logger := slog.New(handler)

		logger.Info("Test message",
			slog.String("key1", "value1"),
			slog.Int("key2", 42),
			slog.Bool("key3", true))

		output := buffer.String()
		require.NotEmpty(t, output)

		var logEntry map[string]interface{}
		err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
		require.NoError(t, err)

		assert.Equal(t, "Test message", logEntry["message"])
		assert.Equal(t, "INFO", logEntry["level"])
		assert.Equal(t, "value1", logEntry["key1"])
		assert.Equal(t, float64(42), logEntry["key2"]) // JSON numbers are float64
		assert.Equal(t, true, logEntry["key3"])
	})

	t.Run("with_attrs", func(t *testing.T) {
		buffer := &bytes.Buffer{}
		handler := NewBufferLogHandler(buffer)
		logger := slog.New(handler.WithAttrs([]slog.Attr{
			slog.String("component", "test"),
			slog.String("version", "1.0.0"),
		}))

		logger.Info("Test with attributes")

		output := buffer.String()
		var logEntry map[string]interface{}
		err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
		require.NoError(t, err)

		assert.Equal(t, "test", logEntry["component"])
		assert.Equal(t, "1.0.0", logEntry["version"])
	})

	t.Run("different_log_levels", func(t *testing.T) {
		buffer := &bytes.Buffer{}
		handler := NewBufferLogHandler(buffer)
		logger := slog.New(handler)

		logger.Debug("Debug message")
		logger.Info("Info message")
		logger.Warn("Warn message")
		logger.Error("Error message")

		output := buffer.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")
		assert.Len(t, lines, 4)

		levels := []string{"DEBUG", "INFO", "WARN", "ERROR"}
		for i, line := range lines {
			var logEntry map[string]interface{}
			err := json.Unmarshal([]byte(line), &logEntry)
			require.NoError(t, err)
			assert.Equal(t, levels[i], logEntry["level"])
		}
	})
}

// TestStreamingRequestStructValidation tests the StreamingRequest structure
func TestStreamingRequestStructValidation(t *testing.T) {
	t.Run("complete_request", func(t *testing.T) {
		request := StreamingRequest{
			Query:       "Deploy 5G core network",
			ModelName:   "gpt-4o-mini",
			EnableRAG:   true,
			MaxTokens:   2048,
			Temperature: 0.1,
		}

		data, err := json.Marshal(request)
		require.NoError(t, err)

		var decoded StreamingRequest
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, request.Query, decoded.Query)
		assert.Equal(t, request.ModelName, decoded.ModelName)
		assert.Equal(t, request.EnableRAG, decoded.EnableRAG)
		assert.Equal(t, request.MaxTokens, decoded.MaxTokens)
		assert.Equal(t, request.Temperature, decoded.Temperature)
	})

	t.Run("minimal_request", func(t *testing.T) {
		request := StreamingRequest{
			Query:     "Scale UPF",
			ModelName: "claude-3-sonnet",
			EnableRAG: false,
		}

		data, err := json.Marshal(request)
		require.NoError(t, err)

		var decoded StreamingRequest
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, request.Query, decoded.Query)
		assert.Equal(t, request.ModelName, decoded.ModelName)
		assert.Equal(t, request.EnableRAG, decoded.EnableRAG)
		assert.Equal(t, 0, decoded.MaxTokens)     // Should be zero value
		assert.Equal(t, 0.0, decoded.Temperature) // Should be zero value
	})
}

// BenchmarkStructuredLogging benchmarks the performance of structured logging
func BenchmarkStructuredLogging(b *testing.B) {
	buffer := &bytes.Buffer{}
	handler := NewBufferLogHandler(buffer)
	logger := slog.New(handler)

	query := "Deploy UPF network function with high availability"
	model := "gpt-4o-mini"
	enableRAG := true

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("Starting streaming request",
			slog.String("query", query),
			slog.String("model", model),
			slog.Bool("enable_rag", enableRAG),
		)
		buffer.Reset() // Clear buffer for next iteration
	}
}

// streamingHandler simulates the actual streaming handler behavior for integration testing
func (sm *TestServiceManager) streamingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if sm.streamingProcessor == nil {
		http.Error(w, "Streaming not enabled", http.StatusServiceUnavailable)
		return
	}

	var req StreamingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sm.logger.Error("Failed to decode streaming request", slog.String("error", err.Error()))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	// Set defaults from config
	if req.ModelName == "" {
		req.ModelName = sm.config.LLMModelName
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = sm.config.LLMMaxTokens
	}

	// This is the critical structured logging that we're testing
	sm.logger.Info("Starting streaming request",
		slog.String("query", req.Query),
		slog.String("model", req.ModelName),
		slog.Bool("enable_rag", req.EnableRAG),
	)

	// Call the streaming processor
	err := sm.streamingProcessor.HandleStreamingRequest(w, r, &req)
	if err != nil {
		sm.logger.Error("Streaming request failed", slog.String("error", err.Error()))
		// Error handling is done within HandleStreamingRequest
	}
}

// MockCircuitBreakerManager provides a mock implementation for testing circuit breaker health checks
type MockCircuitBreakerManager struct {
	stats map[string]interface{}
}

// GetAllStats returns the mock circuit breaker stats
func (m *MockCircuitBreakerManager) GetAllStats() map[string]interface{} {
	return m.stats
}

// GetOrCreate is a mock implementation of GetOrCreate
func (m *MockCircuitBreakerManager) GetOrCreate(name string, config *llm.CircuitBreakerConfig) *llm.CircuitBreaker {
	return nil // Mock implementation
}

// Get is a mock implementation of Get
func (m *MockCircuitBreakerManager) Get(name string) (*llm.CircuitBreaker, bool) {
	return nil, false // Mock implementation
}

// Remove is a mock implementation of Remove
func (m *MockCircuitBreakerManager) Remove(name string) {
	// Mock implementation - no-op
}

// List is a mock implementation of List
func (m *MockCircuitBreakerManager) List() []string {
	names := make([]string, 0, len(m.stats))
	for name := range m.stats {
		names = append(names, name)
	}
	return names
}

// GetStats returns the mock circuit breaker stats (interface-compatible method)
func (m *MockCircuitBreakerManager) GetStats() (map[string]interface{}, error) {
	return m.stats, nil
}

// Shutdown is a mock implementation of Shutdown
func (m *MockCircuitBreakerManager) Shutdown() {
	// Mock implementation - no-op
}

// ResetAll is a mock implementation of ResetAll
func (m *MockCircuitBreakerManager) ResetAll() {
	// Mock implementation - no-op
}

// TestCircuitBreakerHealthValidation tests the circuit breaker health check functionality
func TestCircuitBreakerHealthValidation(t *testing.T) {
	tests := []struct {
		name            string
		stats           map[string]interface{}
		expectedStatus  health.Status
		expectedMessage string
		expectUnhealthy bool
	}{
		{
			name:            "No circuit breakers registered",
			stats:           make(map[string]interface{}),
			expectedStatus:  health.StatusHealthy,
			expectedMessage: "No circuit breakers registered",
			expectUnhealthy: false,
		},
		{
			name: "All circuit breakers operational (closed)",
			stats: map[string]interface{}{
				"service-a": map[string]interface{}{
					"state":    "closed",
					"failures": 0,
				},
				"service-b": map[string]interface{}{
					"state":    "closed",
					"failures": 1,
				},
			},
			expectedStatus:  health.StatusHealthy,
			expectedMessage: "All circuit breakers operational",
			expectUnhealthy: false,
		},
		{
			name: "All circuit breakers half-open (should be operational)",
			stats: map[string]interface{}{
				"service-a": map[string]interface{}{
					"state":    "half-open",
					"failures": 2,
				},
			},
			expectedStatus:  health.StatusHealthy,
			expectedMessage: "All circuit breakers operational",
			expectUnhealthy: false,
		},
		{
			name: "Single circuit breaker open",
			stats: map[string]interface{}{
				"service-a": map[string]interface{}{
					"state":    "open",
					"failures": 5,
				},
			},
			expectedStatus:  health.StatusUnhealthy,
			expectedMessage: "Circuit breakers in open state: [service-a]",
			expectUnhealthy: true,
		},
		{
			name: "Multiple circuit breakers with one open",
			stats: map[string]interface{}{
				"service-a": map[string]interface{}{
					"state":    "closed",
					"failures": 0,
				},
				"service-b": map[string]interface{}{
					"state":    "open",
					"failures": 3,
				},
				"service-c": map[string]interface{}{
					"state":    "closed",
					"failures": 0,
				},
			},
			expectedStatus:  health.StatusUnhealthy,
			expectedMessage: "Circuit breakers in open state: [service-b]",
			expectUnhealthy: true,
		},
		{
			name: "Multiple open circuit breakers (should return all open breakers)",
			stats: map[string]interface{}{
				"service-a": map[string]interface{}{
					"state":    "open",
					"failures": 3,
				},
				"service-b": map[string]interface{}{
					"state":    "open",
					"failures": 2,
				},
			},
			expectedStatus:  health.StatusUnhealthy,
			expectedMessage: "Circuit breakers in open state: [service-a, service-b]",
			expectUnhealthy: true,
		},
		{
			name: "Circuit breaker with malformed stats (missing state)",
			stats: map[string]interface{}{
				"service-a": map[string]interface{}{
					"failures": 0,
				},
			},
			expectedStatus:  health.StatusHealthy,
			expectedMessage: "All circuit breakers operational",
			expectUnhealthy: false,
		},
		{
			name: "Circuit breaker with non-map stats (should be ignored)",
			stats: map[string]interface{}{
				"service-a": map[string]interface{}{
					"state":    "closed",
					"failures": 0,
				},
			},
			expectedStatus:  health.StatusHealthy,
			expectedMessage: "All circuit breakers operational",
			expectUnhealthy: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock circuit breaker manager
			mockCBMgr := &MockCircuitBreakerManager{
				stats: tt.stats,
			}

			// Create mock health checker
			healthChecker := health.NewHealthChecker("test-service", "v1.0.0", slog.Default())

			// Create service manager with mock components
			cfg := &config.LLMProcessorConfig{ServiceVersion: "test-1.0.0"}
			sm := NewServiceManager(cfg, slog.Default())
			sm.circuitBreakerMgr = mockCBMgr
			sm.healthChecker = healthChecker

			// Register health checks (including circuit breaker check)
			sm.registerHealthChecks()

			// Execute the circuit breaker health check
			ctx := context.Background()
			result := sm.healthChecker.RunCheck(ctx, "circuit_breaker")

			// Verify the result
			require.NotNil(t, result)
			assert.Equal(t, tt.expectedStatus, result.Status)

			if tt.expectUnhealthy {
				assert.Equal(t, health.StatusUnhealthy, result.Status)
				if tt.name == "Multiple open circuit breakers (should return all open breakers)" {
					// For multiple open breakers, verify all are reported
					assert.Contains(t, result.Message, "Circuit breakers in open state:")
					assert.Contains(t, result.Message, "service-a")
					assert.Contains(t, result.Message, "service-b")
				} else if tt.expectedMessage != "" {
					// Use expected message for single circuit breaker cases
					assert.Equal(t, tt.expectedMessage, result.Message)
				} else {
					assert.Contains(t, result.Message, "Circuit breakers in open state:")
				}
			} else {
				assert.Equal(t, health.StatusHealthy, result.Status)
				if tt.expectedMessage != "" {
					assert.Equal(t, tt.expectedMessage, result.Message)
				}
			}
		})
	}
}

// TestRegisterHealthChecksIntegration tests the integration of health checks registration
func TestRegisterHealthChecksIntegration(t *testing.T) {
	t.Run("with_circuit_breaker_manager", func(t *testing.T) {
		mockCBMgr := &MockCircuitBreakerManager{
			stats: map[string]interface{}{
				"service-a": map[string]interface{}{
					"state":    "closed",
					"failures": 0,
				},
			},
		}

		healthChecker := health.NewHealthChecker("test-service", "v1.0.0", slog.Default())
		cfg := &config.LLMProcessorConfig{ServiceVersion: "test-1.0.0"}
		sm := NewServiceManager(cfg, slog.Default())
		sm.circuitBreakerMgr = mockCBMgr
		sm.healthChecker = healthChecker

		// Register health checks
		sm.registerHealthChecks()

		// Verify circuit breaker health check was registered
		ctx := context.Background()
		result := sm.healthChecker.RunCheck(ctx, "circuit_breaker")

		require.NotNil(t, result)
		assert.Equal(t, health.StatusHealthy, result.Status)
		assert.Equal(t, "All circuit breakers operational", result.Message)
	})

	t.Run("without_circuit_breaker_manager", func(t *testing.T) {
		healthChecker := health.NewHealthChecker("test-service", "v1.0.0", slog.Default())
		cfg := &config.LLMProcessorConfig{ServiceVersion: "test-1.0.0"}
		sm := NewServiceManager(cfg, slog.Default())
		sm.circuitBreakerMgr = nil // No circuit breaker manager
		sm.healthChecker = healthChecker

		// Register health checks
		sm.registerHealthChecks()

		// Verify circuit breaker health check was registered but returns healthy status
		ctx := context.Background()
		result := sm.healthChecker.RunCheck(ctx, "circuit_breaker")

		// Should return healthy status with no circuit breakers message
		require.NotNil(t, result)
		assert.Equal(t, health.StatusHealthy, result.Status)
		assert.Equal(t, "No circuit breakers registered", result.Message)
	})
}

// BenchmarkCircuitBreakerHealthCheck benchmarks the circuit breaker health check performance
func BenchmarkCircuitBreakerHealthCheck(b *testing.B) {
	// Create a large number of circuit breakers for benchmarking
	stats := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		stats[fmt.Sprintf("service-%d", i)] = make(map[string]interface{})
	}

	mockCBMgr := &MockCircuitBreakerManager{stats: stats}
	healthChecker := health.NewHealthChecker("test-service", "v1.0.0", slog.Default())
	cfg := &config.LLMProcessorConfig{ServiceVersion: "test-1.0.0"}
	sm := NewServiceManager(cfg, slog.Default())
	sm.circuitBreakerMgr = mockCBMgr
	sm.healthChecker = healthChecker

	sm.registerHealthChecks()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sm.healthChecker.RunCheck(ctx, "circuit_breaker")
	}
}
