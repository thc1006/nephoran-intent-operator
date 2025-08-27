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
			"o1_config": "<?xml version=\"1.0\"?><config><upf><mode>core</mode></upf></config>",
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

	return "", fmt.Errorf("no mock response configured for intent: %s", intent)
}

func TestLLMProcessorIntegration(t *testing.T) {
	// Setup test configuration
	config := &Config{
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

	// Create processor with mock LLM client
	processor := NewIntentProcessor(config)
	mockLLMClient := NewMockLLMClient()
	processor.llmClient = mockLLMClient

	t.Run("Test NetworkFunction Deployment Processing", func(t *testing.T) {
		intent := "Deploy UPF network function with 3 replicas"

		req := &NetworkIntentRequest{
			Spec: struct {
				Intent string `json:"intent"`
			}{
				Intent: intent,
			},
			Metadata: struct {
				Name       string `json:"name,omitempty"`
				Namespace  string `json:"namespace,omitempty"`
				UID        string `json:"uid,omitempty"`
				Generation int64  `json:"generation,omitempty"`
			}{
				Name:      "test-upf",
				Namespace: "5g-core",
				UID:       "test-uid-123",
			},
		}

		ctx := context.Background()
		response, err := processor.ProcessIntent(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, "NetworkFunctionDeployment", response.Type)
		assert.Equal(t, "upf-deployment", response.Name)
		assert.Equal(t, "5g-core", response.Namespace)
		assert.Equal(t, intent, response.OriginalIntent)

		// Verify spec contains expected fields
		spec, ok := response.Spec.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(3), spec["replicas"])
		assert.Equal(t, "registry.5g.local/upf:latest", spec["image"])

		// Verify processing metadata
		assert.Equal(t, config.LLMModelName, response.ProcessingMetadata.ModelUsed)
		assert.Greater(t, response.ProcessingMetadata.ConfidenceScore, 0.0)
		assert.Greater(t, response.ProcessingMetadata.ProcessingTimeMS, int64(0))
	})

	t.Run("Test NetworkFunction Scaling Processing", func(t *testing.T) {
		intent := "Scale AMF to 5 replicas"

		req := &NetworkIntentRequest{
			Spec: struct {
				Intent string `json:"intent"`
			}{
				Intent: intent,
			},
		}

		ctx := context.Background()
		response, err := processor.ProcessIntent(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, "NetworkFunctionScale", response.Type)
		assert.Equal(t, "amf-deployment", response.Name)
	})

	t.Run("Test Input Validation", func(t *testing.T) {
		// Test empty intent
		req := &NetworkIntentRequest{
			Spec: struct {
				Intent string `json:"intent"`
			}{
				Intent: "",
			},
		}

		ctx := context.Background()
		_, err := processor.ProcessIntent(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")

		// Test intent too long
		longIntent := string(make([]byte, 3000))
		req.Spec.Intent = longIntent
		_, err = processor.ProcessIntent(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too long")
	})

	t.Run("Test LLM Error Handling", func(t *testing.T) {
		intent := "Test error handling"
		mockLLMClient.SetError(intent, fmt.Errorf("mock LLM error"))

		req := &NetworkIntentRequest{
			Spec: struct {
				Intent string `json:"intent"`
			}{
				Intent: intent,
			},
		}

		ctx := context.Background()
		_, err := processor.ProcessIntent(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "LLM processing failed")
	})

	t.Run("Test Parameter Extraction", func(t *testing.T) {
		intent := "Deploy UPF with 5 replicas and 4GB memory"
		extractedParams := processor.promptEngine.ExtractParameters(intent)

		assert.Contains(t, extractedParams, "replicas")
		assert.Equal(t, "5", extractedParams["replicas"])
		assert.Contains(t, extractedParams, "memory")
		assert.Equal(t, "4Gi", extractedParams["memory"])
	})
}

func TestHTTPEndpoints(t *testing.T) {
	// Setup test server
	config = &Config{
		Port:                    "8080",
		ServiceVersion:          "test-v1.0.0",
		LLMBackendType:          "openai",
		LLMModelName:            "gpt-4o-mini",
		MetricsEnabled:          true,
		CircuitBreakerEnabled:   true,
		CircuitBreakerThreshold: 5,
		CircuitBreakerTimeout:   60 * time.Second,
	}

	processor = NewIntentProcessor(config)
	mockLLMClient := NewMockLLMClient()
	processor.llmClient = mockLLMClient

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
		assert.Equal(t, config.ServiceVersion, response.Version)
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
		assert.Contains(t, response.Dependencies, "llm_backend")
	})

	t.Run("Test Metrics Endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()

		http.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			// FIXME: Adding error check per errcheck linter

			_, _ = w.Write([]byte("# HELP llm_processor_requests_total Total requests\n"))
		}))

		handler, _ := http.DefaultServeMux.Handler(req)
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "llm_processor_requests_total")
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

func TestCircuitBreakerIntegration(t *testing.T) {
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

func TestTelecomPromptEngineIntegration(t *testing.T) {
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
	config := &Config{
		LLMBackendType:          "openai",
		LLMModelName:            "gpt-4o-mini",
		LLMTimeout:              30 * time.Second,
		LLMMaxTokens:            2048,
		CircuitBreakerEnabled:   true,
		CircuitBreakerThreshold: 10,
		CircuitBreakerTimeout:   60 * time.Second,
	}

	processor := NewIntentProcessor(config)
	mockLLMClient := NewMockLLMClient()
	processor.llmClient = mockLLMClient

	req := &NetworkIntentRequest{
		Spec: struct {
			Intent string `json:"intent"`
		}{
			Intent: "Deploy UPF network function with 3 replicas",
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := processor.ProcessIntent(ctx, req)
		if err != nil {
			b.Fatalf("Processing failed: %v", err)
		}
	}
}
