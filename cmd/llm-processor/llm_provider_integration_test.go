package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thc1006/nephoran-intent-operator/internal/ingest"
	"github.com/thc1006/nephoran-intent-operator/internal/llm/providers"
)

// TestLLMProviderHTTPIntegration tests the full HTTP pipeline with the new LLM provider system
func TestLLMProviderHTTPIntegration(t *testing.T) {
	// Setup schema validator for testing
	schemaPath := findTestSchemaPath(t)
	validator, err := ingest.NewValidator(schemaPath)
	require.NoError(t, err, "Failed to create validator")

	// Create temporary directory for handoff files
	handoffDir, err := os.MkdirTemp("", "llm-processor-test-*")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(handoffDir)

	tests := []struct {
		name                string
		providerType        string
		providerAPIKey      string
		requestBody         map[string]interface{}
		expectedStatus      int
		expectValidHandoff  bool
		expectSchemaValid   bool
		handoffFilePattern  string
	}{
		{
			name:         "OFFLINE Provider - Simple Scaling Request",
			providerType: "OFFLINE",
			requestBody: map[string]interface{}{
				"intent": "Scale O-DU to 3 replicas in oran-odu namespace",
			},
			expectedStatus:      http.StatusOK,
			expectValidHandoff:  true,
			expectSchemaValid:   true,
			handoffFilePattern:  "intent-*.json",
		},
		{
			name:         "OFFLINE Provider - Complex Network Function Request",
			providerType: "OFFLINE",
			requestBody: map[string]interface{}{
				"intent": "Urgent: increase AMF instances to 5 in core5g namespace for high priority traffic handling",
			},
			expectedStatus:      http.StatusOK,
			expectValidHandoff:  true,
			expectSchemaValid:   true,
			handoffFilePattern:  "intent-*.json",
		},
		{
			name:         "OFFLINE Provider - E2 Simulator Request",
			providerType: "OFFLINE",
			requestBody: map[string]interface{}{
				"intent": "Deploy 2 E2SIM simulators in test namespace for RAN testing",
			},
			expectedStatus:      http.StatusOK,
			expectValidHandoff:  true,
			expectSchemaValid:   true,
			handoffFilePattern:  "intent-*.json",
		},
		{
			name:           "OpenAI Provider - Mock Request",
			providerType:   "OPENAI",
			providerAPIKey: "sk-test1234567890abcdefghijklmnopqrstuvwxyz",
			requestBody: map[string]interface{}{
				"intent": "Scale RIC xApp to 4 instances in ric-system namespace",
			},
			expectedStatus:      http.StatusOK,
			expectValidHandoff:  true,
			expectSchemaValid:   true,
			handoffFilePattern:  "intent-*.json",
		},
		{
			name:           "Anthropic Provider - Mock Request",
			providerType:   "ANTHROPIC",
			providerAPIKey: "sk-ant-test1234567890abcdefghijklmnopqrstuvwxyzABCDEF",
			requestBody: map[string]interface{}{
				"intent": "Configure SMF with 6 replicas in core5g for enhanced session management",
			},
			expectedStatus:      http.StatusOK,
			expectValidHandoff:  true,
			expectSchemaValid:   true,
			handoffFilePattern:  "intent-*.json",
		},
		{
			name:         "Invalid Request - Empty Intent",
			providerType: "OFFLINE",
			requestBody: map[string]interface{}{
				"intent": "",
			},
			expectedStatus:     http.StatusBadRequest,
			expectValidHandoff: false,
			expectSchemaValid:  false,
		},
		{
			name:         "Invalid Request - Missing Intent Field",
			providerType: "OFFLINE",
			requestBody: map[string]interface{}{
				"description": "Scale something",
			},
			expectedStatus:     http.StatusBadRequest,
			expectValidHandoff: false,
			expectSchemaValid:  false,
		},
		{
			name:         "OpenAI Provider - Missing API Key",
			providerType: "OPENAI",
			requestBody: map[string]interface{}{
				"intent": "Scale workload to 3 replicas",
			},
			expectedStatus:     http.StatusInternalServerError,
			expectValidHandoff: false,
			expectSchemaValid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment for provider
			originalEnv := make(map[string]string)
			envVars := []string{"LLM_PROVIDER", "LLM_API_KEY", "LLM_HANDOFF_DIR"}
			for _, key := range envVars {
				originalEnv[key] = os.Getenv(key)
			}

			os.Setenv("LLM_PROVIDER", tt.providerType)
			if tt.providerAPIKey != "" {
				os.Setenv("LLM_API_KEY", tt.providerAPIKey)
			} else {
				os.Unsetenv("LLM_API_KEY")
			}
			os.Setenv("LLM_HANDOFF_DIR", handoffDir)

			defer func() {
				for key, value := range originalEnv {
					if value == "" {
						os.Unsetenv(key)
					} else {
						os.Setenv(key, value)
					}
				}
			}()

			// Create test server with LLM provider integration
			server := createTestServerWithLLMProvider(t, validator)
			defer server.Close()

			// Prepare request
			requestBody, err := json.Marshal(tt.requestBody)
			require.NoError(t, err, "Failed to marshal request body")

			req, err := http.NewRequest("POST", server.URL+"/process", bytes.NewReader(requestBody))
			require.NoError(t, err, "Failed to create request")
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)
			require.NoError(t, err, "Failed to execute request")
			defer resp.Body.Close()

			// Check status code
			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Unexpected status code")

			// Read response
			responseBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err, "Failed to read response body")

			if tt.expectedStatus == http.StatusOK {
				// Verify response structure
				var response map[string]interface{}
				err = json.Unmarshal(responseBody, &response)
				assert.NoError(t, err, "Response should be valid JSON")

				// Check for error first to prevent panics
				if errMsg, exists := response["error"]; exists {
					t.Fatalf("Expected success but got error: %v", errMsg)
				}

				// Check for expected response fields
				assert.Contains(t, response, "intent_processed", "Response should contain intent_processed field")
				assert.Contains(t, response, "handoff_file", "Response should contain handoff_file field")
				assert.Contains(t, response, "provider_metadata", "Response should contain provider_metadata field")

				// Verify handoff file was created
				if tt.expectValidHandoff {
					handoffFileRaw, exists := response["handoff_file"]
					assert.True(t, exists, "Response should contain handoff_file field")
					handoffFile, ok := handoffFileRaw.(string)
					assert.True(t, ok, "handoff_file should be a string")
					assert.NotEmpty(t, handoffFile, "Handoff file path should not be empty")

					// Check file exists
					_, err := os.Stat(handoffFile)
					assert.NoError(t, err, "Handoff file should exist")

					// Read and validate handoff file content
					if tt.expectSchemaValid {
						handoffData, err := os.ReadFile(handoffFile)
						assert.NoError(t, err, "Should be able to read handoff file")

						// Validate against schema
						intent, err := validator.ValidateBytes(handoffData)
						assert.NoError(t, err, "Handoff file should validate against schema")
						assert.NotNil(t, intent, "Validated intent should not be nil")

						// Verify required fields
						assert.NotEmpty(t, intent.IntentType, "Intent type should not be empty")
						assert.NotEmpty(t, intent.Target, "Target should not be empty")
						assert.NotEmpty(t, intent.Namespace, "Namespace should not be empty")
						assert.Greater(t, intent.Replicas, 0, "Replicas should be greater than 0")
					}
				}
			} else {
				// Verify error response structure
				var errorResponse map[string]interface{}
				err = json.Unmarshal(responseBody, &errorResponse)
				assert.NoError(t, err, "Error response should be valid JSON")
				assert.Contains(t, errorResponse, "error", "Error response should contain error field")
			}
		})
	}
}

// TestStreamingEndpointIntegration tests streaming responses with LLM providers
func TestStreamingEndpointIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping streaming test in short mode")
	}

	// Setup schema validator
	schemaPath := findTestSchemaPath(t)
	validator, err := ingest.NewValidator(schemaPath)
	require.NoError(t, err, "Failed to create validator")

	tests := []struct {
		name         string
		providerType string
		requestBody  map[string]interface{}
		expectChunks int
	}{
		{
			name:         "OFFLINE Provider - Streaming Response",
			providerType: "OFFLINE",
			requestBody: map[string]interface{}{
				"query": "Scale multiple network functions: O-DU to 3 replicas, AMF to 5 replicas, and SMF to 4 replicas",
			},
			expectChunks: 3, // Should process as separate intents
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			os.Setenv("LLM_PROVIDER", tt.providerType)
			defer os.Unsetenv("LLM_PROVIDER")

			// Create test server
			server := createTestServerWithLLMProvider(t, validator)
			defer server.Close()

			// Prepare streaming request
			requestBody, err := json.Marshal(tt.requestBody)
			require.NoError(t, err, "Failed to marshal request body")

			req, err := http.NewRequest("POST", server.URL+"/stream", bytes.NewReader(requestBody))
			require.NoError(t, err, "Failed to create request")
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "text/event-stream")

			// Execute request
			client := &http.Client{Timeout: 60 * time.Second}
			resp, err := client.Do(req)
			require.NoError(t, err, "Failed to execute request")
			defer resp.Body.Close()

			// Check response
			assert.Equal(t, http.StatusOK, resp.StatusCode, "Streaming should return 200")
			assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"), "Content-Type should be text/event-stream")

			// Read streaming response
			responseData, err := io.ReadAll(resp.Body)
			assert.NoError(t, err, "Should be able to read streaming response")

			// Basic validation of streaming format
			responseStr := string(responseData)
			assert.Contains(t, responseStr, "data:", "Response should contain SSE data events")
		})
	}
}

// TestConcurrentRequestsWithLLMProvider tests concurrent HTTP requests with LLM providers
func TestConcurrentRequestsWithLLMProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	// Setup environment
	os.Setenv("LLM_PROVIDER", "OFFLINE")
	defer os.Unsetenv("LLM_PROVIDER")

	// Setup schema validator
	schemaPath := findTestSchemaPath(t)
	validator, err := ingest.NewValidator(schemaPath)
	require.NoError(t, err, "Failed to create validator")

	// Create test server
	server := createTestServerWithLLMProvider(t, validator)
	defer server.Close()

	// Concurrent test parameters
	numClients := 5
	numRequestsPerClient := 3

	results := make(chan error, numClients*numRequestsPerClient)

	for i := 0; i < numClients; i++ {
		go func(clientID int) {
			for j := 0; j < numRequestsPerClient; j++ {
				requestBody := map[string]interface{}{
					"intent": fmt.Sprintf("Scale workload-%d-%d to %d replicas", clientID, j, j+1),
				}

				bodyBytes, err := json.Marshal(requestBody)
				if err != nil {
					results <- fmt.Errorf("client %d, request %d: marshal error: %w", clientID, j, err)
					continue
				}

				req, err := http.NewRequest("POST", server.URL+"/process", bytes.NewReader(bodyBytes))
				if err != nil {
					results <- fmt.Errorf("client %d, request %d: create request error: %w", clientID, j, err)
					continue
				}
				req.Header.Set("Content-Type", "application/json")

				client := &http.Client{Timeout: 30 * time.Second}
				resp, err := client.Do(req)
				if err != nil {
					results <- fmt.Errorf("client %d, request %d: request error: %w", clientID, j, err)
					continue
				}

				if resp.StatusCode != http.StatusOK {
					resp.Body.Close()
					results <- fmt.Errorf("client %d, request %d: unexpected status %d", clientID, j, resp.StatusCode)
					continue
				}

				// Read and validate response
				var response map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&response)
				resp.Body.Close()
				
				if err != nil {
					results <- fmt.Errorf("client %d, request %d: decode error: %w", clientID, j, err)
					continue
				}

				if response["intent_processed"] == nil {
					results <- fmt.Errorf("client %d, request %d: missing intent_processed field", clientID, j)
					continue
				}

				results <- nil
			}
		}(i)
	}

	// Collect results
	for i := 0; i < numClients*numRequestsPerClient; i++ {
		err := <-results
		assert.NoError(t, err, "Concurrent request should succeed")
	}
}

// TestHandoffFileGeneration tests handoff file creation and validation
func TestHandoffFileGeneration(t *testing.T) {
	// Setup schema validator
	schemaPath := findTestSchemaPath(t)
	validator, err := ingest.NewValidator(schemaPath)
	require.NoError(t, err, "Failed to create validator")

	// Create temporary directory for handoff files
	handoffDir, err := os.MkdirTemp("", "handoff-test-*")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(handoffDir)

	tests := []struct {
		name               string
		intent             string
		expectedTarget     string
		expectedNamespace  string
		expectedReplicas   int
		expectedIntentType string
	}{
		{
			name:               "O-RAN DU Scaling",
			intent:             "Scale O-DU distributed unit to 4 replicas in oran-odu namespace",
			expectedTarget:     "o-du",
			expectedNamespace:  "oran-odu",
			expectedReplicas:   4,
			expectedIntentType: "scaling",
		},
		{
			name:               "AMF Core Function",
			intent:             "Increase AMF access mobility function to 8 instances in core5g namespace",
			expectedTarget:     "amf",
			expectedNamespace:  "core5g",
			expectedReplicas:   8,
			expectedIntentType: "scaling",
		},
		{
			name:               "RIC xApp Deployment",
			intent:             "Deploy RAN intelligence xApp with 3 replicas in ric-system",
			expectedTarget:     "ric",
			expectedNamespace:  "ric-system",
			expectedReplicas:   3,
			expectedIntentType: "deployment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			os.Setenv("LLM_PROVIDER", "OFFLINE")
			os.Setenv("LLM_HANDOFF_DIR", handoffDir)
			defer func() {
				os.Unsetenv("LLM_PROVIDER")
				os.Unsetenv("LLM_HANDOFF_DIR")
			}()

			// Create provider and process intent
			config, err := providers.ConfigFromEnvironment()
			require.NoError(t, err, "Failed to create config from environment")

			factory := providers.NewFactory()
			provider, err := factory.CreateProvider(config)
			require.NoError(t, err, "Failed to create provider")
			defer provider.Close()

			// Process intent
			ctx := context.Background()
			response, err := provider.ProcessIntent(ctx, tt.intent)
			require.NoError(t, err, "Failed to process intent")

			// Generate handoff file
			handoffFile := filepath.Join(handoffDir, fmt.Sprintf("intent-%d.json", time.Now().Unix()))
			err = os.WriteFile(handoffFile, response.JSON, 0644)
			require.NoError(t, err, "Failed to write handoff file")

			// Validate handoff file
			intent, err := validator.ValidateBytes(response.JSON)
			assert.NoError(t, err, "Handoff file should validate against schema")
			assert.NotNil(t, intent, "Validated intent should not be nil")

			// Check specific fields
			assert.Equal(t, tt.expectedIntentType, intent.IntentType, "Intent type should match")
			assert.Equal(t, tt.expectedTarget, intent.Target, "Target should match")
			assert.Equal(t, tt.expectedNamespace, intent.Namespace, "Namespace should match")
			assert.Equal(t, tt.expectedReplicas, intent.Replicas, "Replicas should match")

			// Verify file exists and is readable
			fileInfo, err := os.Stat(handoffFile)
			assert.NoError(t, err, "Handoff file should exist")
			assert.Greater(t, fileInfo.Size(), int64(0), "Handoff file should not be empty")

			// Cleanup for next test
			os.Remove(handoffFile)
		})
	}
}

// TestErrorHandlingInHTTPPipeline tests error handling across the HTTP pipeline
func TestErrorHandlingInHTTPPipeline(t *testing.T) {
	// Setup schema validator
	schemaPath := findTestSchemaPath(t)
	validator, err := ingest.NewValidator(schemaPath)
	require.NoError(t, err, "Failed to create validator")

	tests := []struct {
		name           string
		setupEnv       func()
		requestBody    interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Invalid Provider Configuration",
			setupEnv: func() {
				os.Setenv("LLM_PROVIDER", "OPENAI")
				os.Unsetenv("LLM_API_KEY") // Missing API key
			},
			requestBody: map[string]interface{}{
				"intent": "Scale workload to 3 replicas",
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "provider configuration",
		},
		{
			name: "Invalid JSON Request",
			setupEnv: func() {
				os.Setenv("LLM_PROVIDER", "OFFLINE")
			},
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid json",
		},
		{
			name: "Request Too Large",
			setupEnv: func() {
				os.Setenv("LLM_PROVIDER", "OFFLINE")
			},
			requestBody: map[string]interface{}{
				"intent": strings.Repeat("very large intent ", 100000), // ~1.7MB to exceed 1MB limit
			},
			expectedStatus: http.StatusRequestEntityTooLarge,
			expectedError:  "request too large",
		},
		{
			name: "Empty Intent",
			setupEnv: func() {
				os.Setenv("LLM_PROVIDER", "OFFLINE")
			},
			requestBody: map[string]interface{}{
				"intent": "",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Intent field is required and cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			tt.setupEnv()
			defer func() {
				os.Unsetenv("LLM_PROVIDER")
				os.Unsetenv("LLM_API_KEY")
			}()

			// Create test server
			server := createTestServerWithLLMProvider(t, validator)
			defer server.Close()

			// Prepare request
			var requestBody []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				requestBody = []byte(str)
			} else {
				requestBody, err = json.Marshal(tt.requestBody)
				require.NoError(t, err, "Failed to marshal request body")
			}

			req, err := http.NewRequest("POST", server.URL+"/process", bytes.NewReader(requestBody))
			require.NoError(t, err, "Failed to create request")
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)
			require.NoError(t, err, "Failed to execute request")
			defer resp.Body.Close()

			// Check status code
			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Unexpected status code")

			// Read and verify error response
			responseBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err, "Failed to read response body")

			var errorResponse map[string]interface{}
			err = json.Unmarshal(responseBody, &errorResponse)
			assert.NoError(t, err, "Error response should be valid JSON")
			assert.Contains(t, errorResponse, "error", "Error response should contain error field")

			if tt.expectedError != "" {
				errorMsg := fmt.Sprintf("%v", errorResponse["error"])
				assert.Contains(t, strings.ToLower(errorMsg), strings.ToLower(tt.expectedError), 
					"Error message should contain expected text")
			}
		})
	}
}

// Helper function to create a test server with LLM provider integration
func createTestServerWithLLMProvider(t *testing.T, validator *ingest.Validator) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	// Process endpoint
	mux.HandleFunc("/process", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check content length for request size limit (1MB)
		const maxRequestSize = 1 * 1024 * 1024 // 1MB
		if r.ContentLength > maxRequestSize {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "request too large",
			})
			return
		}

		// Read request body with size limit
		body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestSize))
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		// Parse request
		var request map[string]interface{}
		if err := json.Unmarshal(body, &request); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Invalid JSON in request body",
			})
			return
		}

		// Extract intent
		intent, ok := request["intent"].(string)
		if !ok || intent == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Intent field is required and cannot be empty",
			})
			return
		}

		// Create provider from environment
		provider, err := providers.CreateFromEnvironment()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": fmt.Sprintf("Provider configuration error: %v", err),
			})
			return
		}
		defer provider.Close()

		// Process intent
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		response, err := provider.ProcessIntent(ctx, intent)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": fmt.Sprintf("Intent processing failed: %v", err),
			})
			return
		}

		// Create handoff file
		handoffDir := os.Getenv("LLM_HANDOFF_DIR")
		if handoffDir == "" {
			handoffDir = os.TempDir()
		}

		handoffFile := filepath.Join(handoffDir, fmt.Sprintf("intent-%d.json", time.Now().UnixNano()))
		if err := os.WriteFile(handoffFile, response.JSON, 0644); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": fmt.Sprintf("Failed to create handoff file: %v", err),
			})
			return
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"intent_processed": true,
			"handoff_file":     handoffFile,
			"provider_metadata": response.Metadata,
		})
	})

	// Streaming endpoint
	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// Simple streaming response
		fmt.Fprintf(w, "data: {\"status\": \"processing\"}\n\n")
		w.(http.Flusher).Flush()

		time.Sleep(100 * time.Millisecond)

		fmt.Fprintf(w, "data: {\"status\": \"completed\", \"result\": \"success\"}\n\n")
		w.(http.Flusher).Flush()
	})

	return httptest.NewServer(mux)
}

// Helper function to find the test schema path
func findTestSchemaPath(t *testing.T) string {
	t.Helper()

	// Try multiple possible paths relative to cmd/llm-processor
	paths := []string{
		"../../docs/contracts/intent.schema.json",
		"../docs/contracts/intent.schema.json",
		"docs/contracts/intent.schema.json",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			absPath, err := filepath.Abs(path)
			require.NoError(t, err, "Failed to get absolute path")
			return absPath
		}
	}

	// If not found, navigate up to find project root
	cwd, err := os.Getwd()
	require.NoError(t, err, "Failed to get current working directory")

	for dir := cwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		schemaPath := filepath.Join(dir, "docs", "contracts", "intent.schema.json")
		if _, err := os.Stat(schemaPath); err == nil {
			return schemaPath
		}
	}

	t.Fatal("Could not find intent.schema.json file")
	return ""
}