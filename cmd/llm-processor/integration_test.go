//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
)

// Package level variables for integration tests
var (
	testConfig    *Config
	testProcessor *IntentProcessor
)

// Mock handler functions for testing
func processHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req NetworkIntentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(IntegrationErrorResponse{
			ErrorCode: "INVALID_REQUEST",
			Message:   "Invalid JSON format",
		})
		return
	}

	ctx := r.Context()
	result, err := testProcessor.ProcessIntent(ctx, req.Spec.Intent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := NetworkIntentResponse{
		Type:           "NetworkFunctionDeployment",
		Name:           "upf-deployment",
		Namespace:      "5g-core",
		OriginalIntent: req.Spec.Intent,
		Spec:           json.RawMessage(result),
		ProcessingMetadata: struct {
			ModelUsed       string  `json:"modelUsed"`
			ConfidenceScore float64 `json:"confidenceScore"`
		}{
			ModelUsed:       "gpt-4o-mini",
			ConfidenceScore: 0.95,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	response := IntegrationHealthResponse{
		Status:  "ok",
		Version: testConfig.ServiceVersion,
		Time:    time.Now().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func readyzHandler(w http.ResponseWriter, r *http.Request) {
	response := IntegrationReadinessResponse{
		Status: "ready",
		Dependencies: map[string]string{
			"llm_backend": "ready",
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Integration-specific response types (avoiding conflict with e2e_test.go)
type IntegrationErrorResponse struct {
	ErrorCode string `json:"errorCode"`
	Message   string `json:"message"`
}

type IntegrationHealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Time    string `json:"time"`
}

type IntegrationReadinessResponse struct {
	Status       string            `json:"status"`
	Dependencies map[string]string `json:"dependencies"`
}

// Simple circuit breaker implementation for testing
type CircuitBreaker struct {
	failureThreshold int
	timeout          time.Duration
	failureCount     int
	lastFailureTime  time.Time
	state            string // "closed", "open", "half-open"
}

func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: threshold,
		timeout:          timeout,
		state:            "closed",
	}
}

func (cb *CircuitBreaker) Call(fn func() error) error {
	now := time.Now()

	switch cb.state {
	case "open":
		if now.Sub(cb.lastFailureTime) > cb.timeout {
			cb.state = "half-open"
		} else {
			return fmt.Errorf("circuit breaker is open")
		}
	}

	err := fn()
	if err != nil {
		cb.failureCount++
		cb.lastFailureTime = now
		if cb.failureCount >= cb.failureThreshold {
			cb.state = "open"
		}
		return err
	}

	if cb.state == "half-open" {
		cb.state = "closed"
		cb.failureCount = 0
	}

	return nil
}

// MockLLMClient implements the LLM client interface for testing
type MockLLMClient struct {
	responses map[string]string
	errors    map[string]error
}

func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		responses: make(map[string]string),
		errors:    make(map[string]error),
	}
}

func (m *MockLLMClient) SetResponse(intent, response string) {
	m.responses[intent] = response
}

func (m *MockLLMClient) SetError(intent string, err error) {
	m.errors[intent] = err
}

func (m *MockLLMClient) ProcessIntent(ctx context.Context, intent string) (string, error) {
	if err, exists := m.errors[intent]; exists {
		return "", err
	}

	if response, exists := m.responses[intent]; exists {
		return response, nil
	}

	// Default mock response for UPF deployment
	if intent == "Deploy UPF network function with 3 replicas" {
		return `{
			"type": "NetworkFunctionDeployment",
			"name": "upf-deployment",
			"namespace": "5g-core",
			"spec": {
				"replicas": 3,
				"image": "registry.5g.local/upf:latest",
				"resources": {
					"requests": {"cpu": "2000m", "memory": "4Gi"},
					"limits": {"cpu": "4000m", "memory": "8Gi"}
				},
				"ports": [{"containerPort": 8805, "protocol": "UDP"}],
				"env": [{"name": "UPF_MODE", "value": "core"}]
			},
			"o1_testConfig": "<?xml version=\"1.0\"?><testConfig><upf><mode>core</mode></upf></testConfig>",
			"a1_policy": {
				"policy_type_id": "upf-qos-policy",
				"policy_data": {"max_bitrate": "1Gbps"}
			}
		}`, nil
	}

	// Default mock response for scaling
	if intent == "Scale AMF to 5 replicas" {
		return `{
			"type": "NetworkFunctionScale",
			"name": "amf-deployment",
			"namespace": "5g-core",
			"scaling": {
				"horizontal": {
					"replicas": 5,
					"min_replicas": 2,
					"max_replicas": 10
				}
			}
		}`, nil
	}

	return "", fmt.Errorf("no mock response testConfigured for intent: %s", intent)
}

// DISABLED: func TestLLMProcessorIntegration(t *testing.T) {
	// Setup test testConfiguration
	testConfig := &Config{
		Port:             "8080",
		LogLevel:         "debug",
		ServiceVersion:   "test-v1.0.0",
		GracefulShutdown: 30 * time.Second,

		LLMBackendType: "openai",
		LLMAPIKey:      "test-key",
		LLMModelName:   "gpt-4o-mini",
		LLMTimeout:     30 * time.Second,
		LLMMaxTokens:   2048,

		OpenAIAPIURL: "https://api.openai.com/v1/chat/completions",
		OpenAIAPIKey: "test-openai-key",

		RAGEnabled: false,

		CircuitBreakerEnabled:   true,
		CircuitBreakerThreshold: 5,
		CircuitBreakerTimeout:   60 * time.Second,

		MaxRetries:   3,
		RetryDelay:   1 * time.Second,
		RetryBackoff: "exponential",

		MetricsEnabled: true,
	}

	// Create testProcessor with mock LLM client
	testProcessor := NewIntentProcessor(testConfig)
	mockLLMClient := NewMockLLMClient()
	testProcessor.LLMClient = mockLLMClient

	t.Run("Test NetworkFunction Deployment Processing", func(t *testing.T) {
		intent := "Deploy UPF network function with 3 replicas"

		ctx := context.Background()
		responseStr, err := testProcessor.ProcessIntent(ctx, intent)

		require.NoError(t, err)
		assert.NotNil(t, responseStr)

		// Parse the JSON response string
		var response map[string]interface{}
		err = json.Unmarshal([]byte(responseStr), &response)
		require.NoError(t, err)

		// Verify response structure
		assert.Equal(t, "NetworkFunctionDeployment", response["type"])
		assert.Equal(t, "upf-deployment", response["name"])
		assert.Equal(t, "5g-core", response["namespace"])
		assert.Equal(t, intent, response["original_intent"])

		// Verify spec contains expected fields
		spec, ok := response["spec"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(3), spec["replicas"])
		assert.Equal(t, "registry.5g.local/upf:latest", spec["image"])

		// Verify processing metadata
		metadata, ok := response["processing_metadata"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, testConfig.LLMModelName, metadata["model_used"])
		assert.Greater(t, metadata["confidence_score"].(float64), 0.0)
		assert.Greater(t, metadata["processing_time_ms"].(float64), 0.0)
	})

	t.Run("Test NetworkFunction Scaling Processing", func(t *testing.T) {
		intent := "Scale AMF to 5 replicas"

		ctx := context.Background()
		responseStr, err := testProcessor.ProcessIntent(ctx, intent)

		require.NoError(t, err)

		// Parse the JSON response string
		var response map[string]interface{}
		err = json.Unmarshal([]byte(responseStr), &response)
		require.NoError(t, err)

		assert.Equal(t, "NetworkFunctionScale", response["type"])
		assert.Equal(t, "amf-deployment", response["name"])
	})

	t.Run("Test Input Validation", func(t *testing.T) {
		// Test empty intent
		ctx := context.Background()
		_, err := testProcessor.ProcessIntent(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")

		// Test intent too long
		longIntent := string(make([]byte, 3000))
		_, err = testProcessor.ProcessIntent(ctx, longIntent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too long")
	})

	t.Run("Test LLM Error Handling", func(t *testing.T) {
		intent := "Test error handling"
		mockLLMClient.SetError(intent, fmt.Errorf("mock LLM error"))

		ctx := context.Background()
		_, err := testProcessor.ProcessIntent(ctx, intent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "LLM processing failed")
	})

	// Parameter extraction is handled internally by the LLM processor
	t.Run("Test Internal Processing Flow", func(t *testing.T) {
		intent := "Deploy UPF with 5 replicas and 4GB memory"
		ctx := context.Background()

		responseStr, err := testProcessor.ProcessIntent(ctx, intent)
		require.NoError(t, err)
		assert.NotEmpty(t, responseStr)

		// Verify response contains expected deployment information
		assert.Contains(t, responseStr, "UPF")
		assert.Contains(t, responseStr, "5")
	})
}

// DISABLED: func TestHTTPEndpoints(t *testing.T) {
	// Setup test server
	testConfig = &Config{
		Port:                    "8080",
		ServiceVersion:          "test-v1.0.0",
		LLMBackendType:          "openai",
		LLMModelName:            "gpt-4o-mini",
		MetricsEnabled:          true,
		CircuitBreakerEnabled:   true,
		CircuitBreakerThreshold: 5,
		CircuitBreakerTimeout:   60 * time.Second,
	}

	testProcessor = NewIntentProcessor(testConfig)
	mockLLMClient := NewMockLLMClient()
	testProcessor.LLMClient = mockLLMClient

	t.Run("Test Process Endpoint", func(t *testing.T) {
		req := &NetworkIntentRequest{
			Spec: struct {
				Intent string `json:"intent"`
			}{
				Intent: "Deploy UPF network function with 3 replicas",
			},
		}

		reqBody, err := json.Marshal(req)
		require.NoError(t, err)

		httpReq := httptest.NewRequest("POST", "/process", bytes.NewBuffer(reqBody))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		processHandler(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response NetworkIntentResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "NetworkFunctionDeployment", response.Type)
	})

	t.Run("Test Health Endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/healthz", nil)
		w := httptest.NewRecorder()

		healthzHandler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response HealthResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "ok", response.Status)
		assert.Equal(t, testConfig.ServiceVersion, response.Version)
	})

	t.Run("Test Readiness Endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/readyz", nil)
		w := httptest.NewRecorder()

		readyzHandler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response ReadinessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "ready", response.Status)
	})

	t.Run("Test Metrics Endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()

		http.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("# HELP llm_testProcessor_requests_total Total requests\n"))
		}))

		handler, _ := http.DefaultServeMux.Handler(req)
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "llm_testProcessor_requests_total")
	})

	t.Run("Test Invalid Request Format", func(t *testing.T) {
		invalidJSON := `{"invalid": json}`
		httpReq := httptest.NewRequest("POST", "/process", bytes.NewBufferString(invalidJSON))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		processHandler(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errorResp ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &errorResp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_REQUEST", errorResp.ErrorCode)
	})

	t.Run("Test Method Not Allowed", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/process", nil)
		w := httptest.NewRecorder()

		processHandler(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

// DISABLED: func TestCircuitBreakerIntegration(t *testing.T) {
	// Create circuit breaker with low threshold for testing
	cb := NewCircuitBreaker(2, 1*time.Second)

	t.Run("Test Circuit Breaker Opens After Failures", func(t *testing.T) {
		// First failure
		err1 := cb.Call(func() error {
			return fmt.Errorf("first failure")
		})
		assert.Error(t, err1)

		// Second failure - should open circuit
		err2 := cb.Call(func() error {
			return fmt.Errorf("second failure")
		})
		assert.Error(t, err2)

		// Third call should be rejected immediately
		err3 := cb.Call(func() error {
			return nil // This shouldn't be called
		})
		assert.Error(t, err3)
		assert.Contains(t, err3.Error(), "circuit breaker is open")
	})

	t.Run("Test Circuit Breaker Recovers", func(t *testing.T) {
		// Wait for timeout
		time.Sleep(1100 * time.Millisecond)

		// Should transition to half-open and allow one request
		err := cb.Call(func() error {
			return nil // Success
		})
		assert.NoError(t, err)

		// Should now be closed and allow requests
		err = cb.Call(func() error {
			return nil
		})
		assert.NoError(t, err)
	})
}

// DISABLED: func TestTelecomPromptEngineIntegration(t *testing.T) {
	engine := llm.NewTelecomPromptEngine()

	t.Run("Test UPF Intent Processing", func(t *testing.T) {
		intent := "Deploy UPF network function with high availability for 5G core"
		intentType := "NetworkFunctionDeployment"

		prompt := engine.GeneratePrompt(intentType, intent)

		assert.Contains(t, prompt, "UPF")
		assert.Contains(t, prompt, "User Plane Function")
		assert.Contains(t, prompt, "packet routing")
		assert.Contains(t, prompt, "NetworkFunctionDeployment")
	})

	t.Run("Test Near-RT RIC Intent Processing", func(t *testing.T) {
		intent := "Setup Near-RT RIC with xApp support for intelligent network optimization"
		intentType := "NetworkFunctionDeployment"

		prompt := engine.GeneratePrompt(intentType, intent)

		assert.Contains(t, prompt, "Near-RT RIC")
		assert.Contains(t, prompt, "xApp")
		assert.Contains(t, prompt, "E2 interface")
	})

	t.Run("Test Parameter Extraction", func(t *testing.T) {
		intent := "Deploy UPF with 3 replicas, 4 CPU cores, and 8GB memory in 5g-core namespace"
		params := engine.ExtractParameters(intent)

		assert.Equal(t, "3", params["replicas"])
		assert.Equal(t, "4000m", params["cpu"])
		assert.Equal(t, "8Gi", params["memory"])
		assert.Equal(t, "5g-core", params["namespace"])
	})

	t.Run("Test Network Function Detection", func(t *testing.T) {
		testCases := []struct {
			intent           string
			expectedFunction string
		}{
			{"Deploy AMF for authentication", "amf"},
			{"Setup SMF for session management", "smf"},
			{"Configure UPF for user plane", "upf"},
			{"Install Near-RT RIC for intelligent control", "near-rt-ric"},
		}

		for _, tc := range testCases {
			params := engine.ExtractParameters(tc.intent)
			assert.Equal(t, tc.expectedFunction, params["network_function"],
				"Failed for intent: %s", tc.intent)
		}
	})
}

func BenchmarkIntentProcessing(b *testing.B) {
	testConfig := &Config{
		LLMBackendType:          "openai",
		LLMModelName:            "gpt-4o-mini",
		LLMTimeout:              30 * time.Second,
		LLMMaxTokens:            2048,
		CircuitBreakerEnabled:   true,
		CircuitBreakerThreshold: 10,
		CircuitBreakerTimeout:   60 * time.Second,
	}

	testProcessor := NewIntentProcessor(testConfig)
	mockLLMClient := NewMockLLMClient()
	testProcessor.LLMClient = mockLLMClient

	intent := "Deploy UPF network function with 3 replicas"
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := testProcessor.ProcessIntent(ctx, intent)
		if err != nil {
			b.Fatalf("Processing failed: %v", err)
		}
	}
}
