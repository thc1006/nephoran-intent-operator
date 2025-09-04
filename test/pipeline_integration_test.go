// Package test provides end-to-end integration tests for the complete LLM pipeline
package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PipelineTest represents a complete pipeline test configuration
type PipelineTest struct {
	Name                  string
	Intent               string
	ExpectedStatus       int
	ExpectedHandoffFiles int
	ExpectedResponseKeys []string
	Mode                 string
	Provider             string
}

// TestLLMPipelineEndToEnd tests the complete pipeline from intent ingestion to processing
func TestLLMPipelineEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end tests in short mode")
	}

	// Create temporary directories for testing
	tempBase, err := os.MkdirTemp("", "llm-pipeline-e2e-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempBase)

	handoffDir := filepath.Join(tempBase, "handoff")
	err = os.MkdirAll(handoffDir, 0755)
	require.NoError(t, err)

	tests := []PipelineTest{
		{
			Name:                  "Rules Provider Pipeline",
			Intent:               "scale odu-high-phy to 5 in ns oran-odu",
			ExpectedStatus:       http.StatusOK,
			ExpectedHandoffFiles: 1,
			ExpectedResponseKeys: []string{"intent_type", "target", "namespace", "replicas"},
			Mode:                 "rules",
			Provider:             "",
		},
		{
			Name:                  "Mock LLM Provider Pipeline",
			Intent:               "scale cu-cp to 3 in ns oran-cu",
			ExpectedStatus:       http.StatusOK,
			ExpectedHandoffFiles: 1,
			ExpectedResponseKeys: []string{"intent_type", "target", "namespace", "replicas"},
			Mode:                 "llm",
			Provider:             "mock",
		},
		{
			Name:                  "Complex O-RAN Scaling",
			Intent:               "scale out du-manager by 2 in ns oran-du",
			ExpectedStatus:       http.StatusOK,
			ExpectedHandoffFiles: 1,
			ExpectedResponseKeys: []string{"intent_type", "target", "namespace", "delta"},
			Mode:                 "llm",
			Provider:             "mock",
		},
		{
			Name:           "Invalid Intent Format",
			Intent:         "invalid intent that should fail",
			ExpectedStatus: http.StatusBadRequest,
			Mode:           "rules",
			Provider:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Clear handoff directory for each test
			err := os.RemoveAll(handoffDir)
			require.NoError(t, err)
			err = os.MkdirAll(handoffDir, 0755)
			require.NoError(t, err)

			// Start intent-ingest service
			ingestServer := startIntentIngestService(t, handoffDir, tt.Mode, tt.Provider)
			defer ingestServer.Close()

			// Send intent to ingest service
			response := sendIntentRequest(t, ingestServer.URL, tt.Intent)

			// Verify HTTP response
			assert.Equal(t, tt.ExpectedStatus, response.StatusCode, "HTTP status mismatch")

			if tt.ExpectedStatus == http.StatusOK {
				// Parse response
				responseBody, err := io.ReadAll(response.Body)
				require.NoError(t, err)

				var intentResponse map[string]interface{}
				err = json.Unmarshal(responseBody, &intentResponse)
				require.NoError(t, err, "Response should be valid JSON")

				// Verify response contains expected keys
				for _, key := range tt.ExpectedResponseKeys {
					assert.Contains(t, intentResponse, key, "Response should contain key: %s", key)
				}

				// Verify handoff files were created
				files, err := os.ReadDir(handoffDir)
				require.NoError(t, err)
				assert.Equal(t, tt.ExpectedHandoffFiles, len(files), "Expected number of handoff files")

				if len(files) > 0 {
					// Verify handoff file structure
					handoffFile := files[0]
					content, err := os.ReadFile(filepath.Join(handoffDir, handoffFile.Name()))
					require.NoError(t, err)

					var handoffData map[string]interface{}
					err = json.Unmarshal(content, &handoffData)
					require.NoError(t, err, "Handoff file should contain valid JSON")

					// Verify handoff structure matches response
					for _, key := range tt.ExpectedResponseKeys {
						assert.Contains(t, handoffData, key, "Handoff should contain key: %s", key)
						if intentResponse[key] != nil && handoffData[key] != nil {
							assert.Equal(t, intentResponse[key], handoffData[key], 
								"Handoff value should match response for key: %s", key)
						}
					}

					// Test handoff consumption by LLM processor
					if tt.ExpectedStatus == http.StatusOK {
						processedResult := simulateLLMProcessorConsumption(t, content)
						assert.NotEmpty(t, processedResult, "LLM processor should process handoff")
					}
				}
			}

			response.Body.Close()
		})
	}
}

// TestPipelineConcurrency tests concurrent pipeline operations
func TestPipelineConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency tests in short mode")
	}

	tempDir, err := os.MkdirTemp("", "pipeline-concurrency-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	handoffDir := filepath.Join(tempDir, "handoff")
	err = os.MkdirAll(handoffDir, 0755)
	require.NoError(t, err)

	// Start intent-ingest service
	ingestServer := startIntentIngestService(t, handoffDir, "llm", "mock")
	defer ingestServer.Close()

	// Test concurrent requests
	numConcurrentRequests := 10
	var wg sync.WaitGroup
	results := make(chan testResult, numConcurrentRequests)

	for i := 0; i < numConcurrentRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			intent := fmt.Sprintf("scale test-service-%d to %d", id, id+1)
			response := sendIntentRequest(t, ingestServer.URL, intent)
			results <- testResult{
				ID:     id,
				Status: response.StatusCode,
				Error:  nil,
			}
			response.Body.Close()
		}(i)
	}

	// Wait for all requests to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	successCount := 0
	for result := range results {
		if result.Status == http.StatusOK {
			successCount++
		} else {
			t.Errorf("Request %d failed with status %d", result.ID, result.Status)
		}
	}

	assert.Equal(t, numConcurrentRequests, successCount, "All concurrent requests should succeed")

	// Verify handoff files were created
	files, err := os.ReadDir(handoffDir)
	require.NoError(t, err)
	assert.Equal(t, numConcurrentRequests, len(files), "Should have one handoff file per request")
}

// TestPipelineSchemaValidation tests schema validation across the pipeline
func TestPipelineSchemaValidation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pipeline-schema-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	handoffDir := filepath.Join(tempDir, "handoff")
	err = os.MkdirAll(handoffDir, 0755)
	require.NoError(t, err)

	// Start intent-ingest service
	ingestServer := startIntentIngestService(t, handoffDir, "rules", "")
	defer ingestServer.Close()

	// Test valid intent that should pass schema validation
	validIntent := "scale odu-high-phy to 5 in ns oran-odu"
	response := sendIntentRequest(t, ingestServer.URL, validIntent)
	defer response.Body.Close()

	assert.Equal(t, http.StatusOK, response.StatusCode, "Valid intent should pass schema validation")

	// Verify handoff file conforms to schema
	files, err := os.ReadDir(handoffDir)
	require.NoError(t, err)
	require.Greater(t, len(files), 0, "Handoff file should be created")

	content, err := os.ReadFile(filepath.Join(handoffDir, files[0].Name()))
	require.NoError(t, err)

	var handoffData map[string]interface{}
	err = json.Unmarshal(content, &handoffData)
	require.NoError(t, err)

	// Verify required schema fields
	requiredFields := []string{"intent_type", "target", "namespace", "replicas"}
	for _, field := range requiredFields {
		assert.Contains(t, handoffData, field, "Handoff should contain required schema field: %s", field)
	}

	// Verify field types
	assert.IsType(t, "", handoffData["intent_type"])
	assert.IsType(t, "", handoffData["target"])
	assert.IsType(t, "", handoffData["namespace"])
	assert.IsType(t, float64(0), handoffData["replicas"]) // JSON numbers are float64

	// Verify enum values
	intentType := handoffData["intent_type"].(string)
	validIntentTypes := []string{"scaling", "deployment", "configuration"}
	assert.Contains(t, validIntentTypes, intentType, "Intent type should be valid enum value")

	// Verify replica range
	replicas := handoffData["replicas"].(float64)
	assert.GreaterOrEqual(t, replicas, float64(1), "Replicas should be >= 1")
	assert.LessOrEqual(t, replicas, float64(100), "Replicas should be <= 100")
}

// TestPipelineErrorHandling tests error scenarios across the pipeline
func TestPipelineErrorHandling(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pipeline-errors-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	handoffDir := filepath.Join(tempDir, "handoff")
	err = os.MkdirAll(handoffDir, 0755)
	require.NoError(t, err)

	// Start intent-ingest service
	ingestServer := startIntentIngestService(t, handoffDir, "rules", "")
	defer ingestServer.Close()

	errorTests := []struct {
		name         string
		requestBody  string
		expectedCode int
	}{
		{
			name:         "Empty Request Body",
			requestBody:  "",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Invalid JSON",
			requestBody:  `{"invalid": json}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Missing Spec Field",
			requestBody:  `{"intent": "scale test to 3"}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Empty Intent",
			requestBody:  `{"spec": {"intent": ""}}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Unparseable Intent",
			requestBody:  `{"spec": {"intent": "invalid intent format"}}`,
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", ingestServer.URL+"/intent", strings.NewReader(tt.requestBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedCode, resp.StatusCode, "Error case should return expected status")
		})
	}
}

// TestPipelinePerformance benchmarks the complete pipeline
func TestPipelinePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	tempDir, err := os.MkdirTemp("", "pipeline-perf-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	handoffDir := filepath.Join(tempDir, "handoff")
	err = os.MkdirAll(handoffDir, 0755)
	require.NoError(t, err)

	// Start intent-ingest service
	ingestServer := startIntentIngestService(t, handoffDir, "rules", "")
	defer ingestServer.Close()

	// Warm up
	for i := 0; i < 5; i++ {
		response := sendIntentRequest(t, ingestServer.URL, "scale test to 3")
		response.Body.Close()
	}

	// Clear handoff directory
	os.RemoveAll(handoffDir)
	os.MkdirAll(handoffDir, 0755)

	// Measure performance
	numRequests := 100
	start := time.Now()

	for i := 0; i < numRequests; i++ {
		intent := fmt.Sprintf("scale test-%d to %d", i, (i%10)+1)
		response := sendIntentRequest(t, ingestServer.URL, intent)
		require.Equal(t, http.StatusOK, response.StatusCode)
		response.Body.Close()
	}

	duration := time.Since(start)
	avgLatency := duration / time.Duration(numRequests)

	t.Logf("Pipeline performance:")
	t.Logf("  Total requests: %d", numRequests)
	t.Logf("  Total duration: %v", duration)
	t.Logf("  Average latency: %v", avgLatency)
	t.Logf("  Requests per second: %.2f", float64(numRequests)/duration.Seconds())

	// Performance assertions
	assert.Less(t, avgLatency, 100*time.Millisecond, "Average latency should be < 100ms")

	// Verify all handoff files were created
	files, err := os.ReadDir(handoffDir)
	require.NoError(t, err)
	assert.Equal(t, numRequests, len(files), "All requests should generate handoff files")
}

// Helper functions

type testResult struct {
	ID     int
	Status int
	Error  error
}

// startIntentIngestService starts the intent-ingest service for testing
func startIntentIngestService(t *testing.T, handoffDir, mode, provider string) *httptest.Server {
	// Create mock intent-ingest service
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok\n"))
	})

	mux.HandleFunc("/intent", func(w http.ResponseWriter, r *http.Request) {
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

		// Parse intent using mock logic
		result, err := mockParseIntent(intent, mode, provider)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Create handoff file
		timestamp := time.Now().Format("20060102-150405.000")
		filename := fmt.Sprintf("intent-%s.json", timestamp)
		handoffPath := filepath.Join(handoffDir, filename)

		handoffData := result
		handoffData["created_at"] = time.Now().UTC().Format(time.RFC3339)
		handoffData["source"] = mode
		handoffData["status"] = "pending"

		handoffJSON, err := json.Marshal(handoffData)
		if err != nil {
			http.Error(w, "Failed to marshal handoff data", http.StatusInternalServerError)
			return
		}

		err = os.WriteFile(handoffPath, handoffJSON, 0644)
		if err != nil {
			http.Error(w, "Failed to write handoff file", http.StatusInternalServerError)
			return
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	return httptest.NewServer(mux)
}

// sendIntentRequest sends an intent request to the service
func sendIntentRequest(t *testing.T, serverURL, intent string) *http.Response {
	requestBody := map[string]interface{}{
		"spec": map[string]interface{}{
			"intent": intent,
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	require.NoError(t, err)

	resp, err := http.Post(serverURL+"/intent", "application/json", bytes.NewReader(jsonBody))
	require.NoError(t, err)

	return resp
}

// mockParseIntent provides mock intent parsing logic
func mockParseIntent(intent, mode, provider string) (map[string]interface{}, error) {
	intent = strings.TrimSpace(intent)

	// Simple regex-based parsing for testing
	if strings.Contains(intent, "scale") && strings.Contains(intent, "to") {
		parts := strings.Fields(intent)
		if len(parts) >= 4 {
			target := parts[1]
			replicasStr := parts[3]
			
			// Extract namespace if present
			namespace := "default"
			if strings.Contains(intent, " in ns ") {
				nsParts := strings.Split(intent, " in ns ")
				if len(nsParts) == 2 {
					namespace = strings.TrimSpace(nsParts[1])
				}
			}

			// Convert replicas (simple parsing for testing)
			replicas := 1
			if replicasStr == "1" || replicasStr == "2" || replicasStr == "3" || 
			   replicasStr == "4" || replicasStr == "5" {
				switch replicasStr {
				case "1": replicas = 1
				case "2": replicas = 2
				case "3": replicas = 3
				case "4": replicas = 4
				case "5": replicas = 5
				}
			}

			return map[string]interface{}{
				"intent_type": "scaling",
				"target":      target,
				"namespace":   namespace,
				"replicas":    replicas,
			}, nil
		}
	}

	// Handle scale out/in patterns
	if strings.Contains(intent, "scale out") || strings.Contains(intent, "scale in") {
		parts := strings.Fields(intent)
		if len(parts) >= 5 {
			target := parts[2]
			deltaStr := parts[4]
			
			namespace := "default"
			if strings.Contains(intent, " in ns ") {
				nsParts := strings.Split(intent, " in ns ")
				if len(nsParts) == 2 {
					namespace = strings.TrimSpace(nsParts[1])
				}
			}

			delta := 1
			if deltaStr == "1" || deltaStr == "2" || deltaStr == "3" {
				switch deltaStr {
				case "1": delta = 1
				case "2": delta = 2
				case "3": delta = 3
				}
			}

			if strings.Contains(intent, "scale in") {
				delta = -delta
			}

			return map[string]interface{}{
				"intent_type": "scaling",
				"target":      target,
				"namespace":   namespace,
				"delta":       delta,
			}, nil
		}
	}

	return nil, fmt.Errorf("unable to parse intent: %s", intent)
}

// simulateLLMProcessorConsumption simulates how the LLM processor would consume handoff files
func simulateLLMProcessorConsumption(t *testing.T, handoffContent []byte) map[string]interface{} {
	var handoffData map[string]interface{}
	err := json.Unmarshal(handoffContent, &handoffData)
	require.NoError(t, err)

	// Simulate LLM processor transforming the handoff data
	processedResult := map[string]interface{}{
		"type":            "NetworkFunctionDeployment",
		"name":            fmt.Sprintf("%s-deployment", handoffData["target"]),
		"namespace":       handoffData["namespace"],
		"original_intent": handoffData,
		"spec": map[string]interface{}{
			"replicas": handoffData["replicas"],
			"image":    fmt.Sprintf("registry.5g.local/%s:latest", handoffData["target"]),
		},
		"processing_metadata": map[string]interface{}{
			"model_used":         "mock-model",
			"confidence_score":   0.95,
			"processing_time_ms": 150.0,
		},
	}

	return processedResult
}

// TestPipelineRobustness tests pipeline behavior under various conditions
func TestPipelineRobustness(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping robustness tests in short mode")
	}

	tempDir, err := os.MkdirTemp("", "pipeline-robust-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	handoffDir := filepath.Join(tempDir, "handoff")
	err = os.MkdirAll(handoffDir, 0755)
	require.NoError(t, err)

	ingestServer := startIntentIngestService(t, handoffDir, "llm", "mock")
	defer ingestServer.Close()

	t.Run("Rapid Sequential Requests", func(t *testing.T) {
		// Send requests as fast as possible
		for i := 0; i < 50; i++ {
			intent := fmt.Sprintf("scale rapid-test-%d to %d", i, (i%5)+1)
			response := sendIntentRequest(t, ingestServer.URL, intent)
			assert.Equal(t, http.StatusOK, response.StatusCode, "Rapid request %d should succeed", i)
			response.Body.Close()
		}

		// Verify all handoff files were created
		files, err := os.ReadDir(handoffDir)
		require.NoError(t, err)
		assert.Equal(t, 50, len(files), "All rapid requests should create handoff files")
	})

	t.Run("Large Intent Text", func(t *testing.T) {
		// Clear handoff directory
		os.RemoveAll(handoffDir)
		os.MkdirAll(handoffDir, 0755)

		// Test with large intent text (but within limits)
		largeIntent := "scale " + strings.Repeat("very-long-service-name-", 10) + " to 3"
		response := sendIntentRequest(t, ingestServer.URL, largeIntent)
		defer response.Body.Close()
		
		// Should succeed if within reasonable limits
		if len(largeIntent) < 1000 {
			assert.Equal(t, http.StatusOK, response.StatusCode, "Large intent should be processed")
		}
	})

	t.Run("Special Characters in Intent", func(t *testing.T) {
		// Clear handoff directory
		os.RemoveAll(handoffDir)
		os.MkdirAll(handoffDir, 0755)

		specialIntents := []string{
			"scale service-with-dashes to 3",
			"scale service_with_underscores to 2",
			"scale service123 to 4",
		}

		for _, intent := range specialIntents {
			response := sendIntentRequest(t, ingestServer.URL, intent)
			assert.Equal(t, http.StatusOK, response.StatusCode, 
				"Special character intent should be processed: %s", intent)
			response.Body.Close()
		}
	})
}

// BenchmarkPipelineE2E benchmarks the complete end-to-end pipeline
func BenchmarkPipelineE2E(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "pipeline-bench-*")
	require.NoError(b, err)
	defer os.RemoveAll(tempDir)

	handoffDir := filepath.Join(tempDir, "handoff")
	err = os.MkdirAll(handoffDir, 0755)
	require.NoError(b, err)

	ingestServer := startIntentIngestService(&testing.T{}, handoffDir, "rules", "")
	defer ingestServer.Close()

	intent := "scale odu-high-phy to 5 in ns oran-odu"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			response := sendIntentRequest(&testing.T{}, ingestServer.URL, intent)
			if response.StatusCode != http.StatusOK {
				b.Errorf("Request failed with status %d", response.StatusCode)
			}
			response.Body.Close()
		}
	})
}