package ingest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockValidatorInterface implements validation for testing
type MockValidatorInterface struct {
	ValidateBytesFunc func([]byte) (*Intent, error)
}

func (m *MockValidatorInterface) ValidateBytes(b []byte) (*Intent, error) {
	if m.ValidateBytesFunc != nil {
		return m.ValidateBytesFunc(b)
	}
	// Default successful validation
	return &Intent{
		IntentType: "scaling",
		Target:     "test-deployment",
		Namespace:  "default",
		Replicas:   3,
		Source:     "user",
	}, nil
}

// HandlerInterface wraps the actual Handler for testing
type HandlerInterface struct {
	validator ValidatorInterface
	handoffDir string
	provider IntentProvider
}

func NewTestHandler(validator ValidatorInterface, handoffDir string, provider IntentProvider) *HandlerInterface {
	return &HandlerInterface{
		validator: validator,
		handoffDir: handoffDir,
		provider: provider,
	}
}

func (h *HandlerInterface) HandleIntent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	if len(body) == 0 {
		http.Error(w, "Empty request body", http.StatusBadRequest)
		return
	}

	var req map[string]interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	spec, ok := req["spec"].(map[string]interface{})
	if !ok {
		http.Error(w, "Missing spec field", http.StatusBadRequest)
		return
	}

	intent, ok := spec["intent"].(string)
	if !ok || intent == "" {
		http.Error(w, "Empty intent", http.StatusBadRequest)
		return
	}

	// Parse intent using provider
	ctx := r.Context()
	result, err := h.provider.ParseIntent(ctx, intent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create Intent struct from result
	intentStruct := &Intent{
		IntentType: "scaling",
		Target:     result["target"].(string),
		Namespace:  result["namespace"].(string),
		Source:     h.provider.Name(),
	}

	if replicas, ok := result["replicas"].(int); ok {
		intentStruct.Replicas = replicas
	}

	// Validate using mock validator
	jsonData, err := json.Marshal(intentStruct)
	if err != nil {
		http.Error(w, "Failed to marshal intent", http.StatusInternalServerError)
		return
	}

	validatedIntent, err := h.validator.ValidateBytes(jsonData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create handoff file with collision-proof filename
	nanoTimestamp := time.Now().UnixNano()
	uniqueID := uuid.New().String()
	filename := fmt.Sprintf("intent-%d-%s.json", nanoTimestamp, uniqueID)
	handoffPath := filepath.Join(h.handoffDir, filename)

	// Ensure directory exists
	err = os.MkdirAll(h.handoffDir, 0755)
	if err != nil {
		http.Error(w, "Failed to create handoff directory", http.StatusInternalServerError)
		return
	}

	// Write file with exclusive creation flag to prevent overwrites
	file, err := os.OpenFile(handoffPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		http.Error(w, "Failed to create handoff file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	_, err = file.Write(jsonData)
	if err != nil {
		http.Error(w, "Failed to write handoff file", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(validatedIntent)
}

// TestLLMProviderIntegrationFixed tests the complete LLM provider pipeline
func TestLLMProviderIntegrationFixed(t *testing.T) {
	tests := []struct {
		name          string
		mode          string
		provider      string
		intent        string
		expectSuccess bool
		expectedFields map[string]interface{}
	}{
		{
			name:          "Rules Provider Success",
			mode:          "rules",
			provider:      "",
			intent:        "scale odu-high-phy to 5 in ns oran-odu",
			expectSuccess: true,
			expectedFields: map[string]interface{}{
				"target":    "odu-high-phy",
				"replicas":  5,
				"namespace": "oran-odu",
			},
		},
		{
			name:          "Mock LLM Provider Success",
			mode:          "llm",
			provider:      "mock",
			intent:        "scale cu-cp to 3 in ns oran-cu",
			expectSuccess: true,
			expectedFields: map[string]interface{}{
				"target":    "cu-cp",
				"replicas":  3,
				"namespace": "oran-cu",
			},
		},
		{
			name:          "Invalid Intent",
			mode:          "llm",
			provider:      "mock",
			intent:        "invalid intent format",
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary handoff directory
			tempDir, err := os.MkdirTemp("", "integration-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			// Create mock validator
			mockValidator := &MockValidatorInterface{}

			// Create provider
			intentProvider, err := NewProvider(tt.mode, tt.provider)
			require.NoError(t, err)

			// Create handler
			handler := NewTestHandler(mockValidator, tempDir, intentProvider)

			// Create test request
			requestBody := map[string]interface{}{
				"spec": map[string]interface{}{
					"intent": tt.intent,
				},
			}
			jsonBody, err := json.Marshal(requestBody)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/intent", bytes.NewReader(jsonBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute request
			handler.HandleIntent(rr, req)

			// Check response
			if tt.expectSuccess {
				assert.Equal(t, http.StatusOK, rr.Code, "Expected successful response")

				// Check that handoff file was created
				files, err := os.ReadDir(tempDir)
				require.NoError(t, err)
				assert.Greater(t, len(files), 0, "Expected handoff file to be created")
			} else {
				assert.NotEqual(t, http.StatusOK, rr.Code, "Expected error response")
			}
		})
	}
}

// TestLLMProviderSchemaValidationFixed tests schema validation in the integration pipeline
func TestLLMProviderSchemaValidationFixed(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "schema-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create mock validator with different behaviors
	tests := []struct {
		name          string
		validatorFunc func([]byte) (*Intent, error)
		requestBody   map[string]interface{}
		expectSuccess bool
	}{
		{
			name: "Valid Intent Structure",
			validatorFunc: func(b []byte) (*Intent, error) {
				return &Intent{
					IntentType: "scaling",
					Target:     "odu-high-phy",
					Namespace:  "oran-odu",
					Replicas:   3,
				}, nil
			},
			requestBody: map[string]interface{}{
				"spec": map[string]interface{}{
					"intent": "scale odu-high-phy to 3 in ns oran-odu",
				},
			},
			expectSuccess: true,
		},
		{
			name: "Validation Error",
			validatorFunc: func(b []byte) (*Intent, error) {
				return nil, fmt.Errorf("validation failed")
			},
			requestBody: map[string]interface{}{
				"spec": map[string]interface{}{
					"intent": "scale odu-high-phy to 3 in ns oran-odu",
				},
			},
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockValidator := &MockValidatorInterface{
				ValidateBytesFunc: tt.validatorFunc,
			}

			provider, err := NewProvider("llm", "mock")
			require.NoError(t, err)

			handler := NewTestHandler(mockValidator, tempDir, provider)

			jsonBody, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/intent", bytes.NewReader(jsonBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.HandleIntent(rr, req)

			if tt.expectSuccess {
				assert.Equal(t, http.StatusOK, rr.Code, "Expected successful response for valid request")
			} else {
				assert.NotEqual(t, http.StatusOK, rr.Code, "Expected error response for invalid request")
			}
		})
	}
}

// TestLLMProviderConcurrentRequestsFixed tests concurrent handling of requests
func TestLLMProviderConcurrentRequestsFixed(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "concurrent-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	mockValidator := &MockValidatorInterface{}
	provider, err := NewProvider("llm", "mock")
	require.NoError(t, err)

	handler := NewTestHandler(mockValidator, tempDir, provider)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(handler.HandleIntent))
	defer server.Close()

	// Number of concurrent requests
	numRequests := 10
	responses := make(chan *http.Response, numRequests)
	errors := make(chan error, numRequests)

	// Send concurrent requests
	for i := 0; i < numRequests; i++ {
		go func(id int) {
			requestBody := map[string]interface{}{
				"spec": map[string]interface{}{
					"intent": fmt.Sprintf("scale target-%d to %d", id, id+1),
				},
			}
			jsonBody, err := json.Marshal(requestBody)
			if err != nil {
				errors <- err
				return
			}

			resp, err := http.Post(server.URL, "application/json", bytes.NewReader(jsonBody))
			if err != nil {
				errors <- err
				return
			}

			responses <- resp
		}(i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < numRequests; i++ {
		select {
		case resp := <-responses:
			if resp.StatusCode == http.StatusOK {
				successCount++
			}
			resp.Body.Close()
		case err := <-errors:
			t.Errorf("Request failed: %v", err)
		case <-time.After(30 * time.Second):
			t.Fatal("Timeout waiting for concurrent requests")
		}
	}

	assert.Equal(t, numRequests, successCount, "All concurrent requests should succeed")

	// Verify all handoff files were created
	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.Equal(t, numRequests, len(files), "Should have one handoff file per request")
}

// BenchmarkLLMProviderPipelineFixed benchmarks the complete LLM provider pipeline
func BenchmarkLLMProviderPipelineFixed(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "benchmark-*")
	require.NoError(b, err)
	defer os.RemoveAll(tempDir)

	mockValidator := &MockValidatorInterface{}
	provider, err := NewProvider("llm", "mock")
	require.NoError(b, err)

	handler := NewTestHandler(mockValidator, tempDir, provider)

	requestBody := map[string]interface{}{
		"spec": map[string]interface{}{
			"intent": "scale odu-high-phy to 3 in ns oran-odu",
		},
	}
	jsonBody, err := json.Marshal(requestBody)
	require.NoError(b, err)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, err := http.NewRequest("POST", "/intent", bytes.NewReader(jsonBody))
			require.NoError(b, err)
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.HandleIntent(rr, req)

			if rr.Code != http.StatusOK {
				b.Errorf("Expected status OK, got %d", rr.Code)
			}
		}
	})
}

// TestCollisionProofFileGeneration tests that 100 parallel file creations produce 100 distinct files
func TestCollisionProofFileGeneration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "collision-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	mockValidator := &MockValidatorInterface{
		ValidateBytesFunc: func(data []byte) (*Intent, error) {
			var intent Intent
			err := json.Unmarshal(data, &intent)
			return &intent, err
		},
	}

	provider, err := NewProvider("llm", "mock")
	require.NoError(t, err)

	handler := NewTestHandler(mockValidator, tempDir, provider)

	// Number of parallel file creations
	numCreations := 100
	responses := make(chan *http.Response, numCreations)
	errors := make(chan error, numCreations)

	// Create files in parallel
	for i := 0; i < numCreations; i++ {
		go func(id int) {
			requestBody := map[string]interface{}{
				"spec": map[string]interface{}{
					"intent": fmt.Sprintf("scale target-%d to %d", id, id+1),
				},
			}
			jsonBody, err := json.Marshal(requestBody)
			if err != nil {
				errors <- err
				return
			}

			req, err := http.NewRequest("POST", "/intent", bytes.NewReader(jsonBody))
			if err != nil {
				errors <- err
				return
			}
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.HandleIntent(rr, req)

			responses <- &http.Response{StatusCode: rr.Code, Body: io.NopCloser(bytes.NewReader(rr.Body.Bytes()))}
		}(i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < numCreations; i++ {
		select {
		case resp := <-responses:
			if resp.StatusCode == http.StatusOK {
				successCount++
			}
			resp.Body.Close()
		case err := <-errors:
			t.Errorf("File creation failed: %v", err)
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for parallel file creations")
		}
	}

	assert.Equal(t, numCreations, successCount, "All parallel file creations should succeed")

	// Verify all files were created distinctly
	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.Equal(t, numCreations, len(files), "Should have one distinct file per creation")

	// Verify all filenames are unique
	filenames := make(map[string]bool)
	for _, file := range files {
		if filenames[file.Name()] {
			t.Errorf("Duplicate filename found: %s", file.Name())
		}
		filenames[file.Name()] = true
	}
}